import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type AutoClaimBatch, type AutoClaimUploadResponse } from '@/api/types'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { useSelectedPortal } from '@/app/portal'

import { useAutoClaimBatchList, useAutoClaimTabList, useExportAutoClaim } from './api'
import { BatchDetail } from './BatchDetail'
import { CompanySummary } from './CompanySummary'
import { SourceTab } from './SourceTab'
import { UploadForm } from './UploadForm'

/** Batch yang sedang dibuka rinciannya. */
type OpenedBatch = { company: string; companyName: string; batch: string }

type MessageContent = { title: string; description: string; tone: ErrorTone }

function loadMessage(error: unknown): MessageContent {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan Anda, lalu muat ulang.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description:
            'Data klaim dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Entitasnya sudah direncanakan, tetapi kredensial basis datanya belum diisi. Hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
      default:
        return {
          title: 'Daftar batch tidak dapat dimuat',
          description: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
    }
  }
  return {
    title: 'Daftar batch tidak dapat dimuat',
    description: 'Coba beberapa saat lagi.',
    tone: 'gangguan',
  }
}

/**
 * Layar Inbox Auto Claim.
 *
 * Pengganti `Harness/InboxAutoClaim-Harness.xml` atas POOLDATA.TMP_BATCH_AUTO_CLAIM.
 * Judul, susunan kolom, penyaring, dan tombolnya mengikuti layar lama (`D-13`: alur dan
 * tata letak ditiru supaya pengguna tidak perlu belajar ulang):
 *
 *   - Judul "INBOX AUTO CLAIM"      — `Section/Inbox_AS_KREDIT_Sect-Section.xml`
 *   - Penyaring "Nama Perusahaan"   — section yang sama
 *   - Grid 9 kolom                  — section yang sama, urutan kolomnya dipertahankan
 *   - Paginasi First/Prev/Next/Last — `Section/ButtonPagingInbox-Section.xml`
 *   - Tujuh tombol                  — section yang sama
 *
 * # TIGA DARI TUJUH TOMBOL BELUM DAPAT DIKERJAKAN, DAN ITU DINYATAKAN DI LAYAR
 *
 * Yang sudah jalan: Upload Data Klaim, DETAIL, EXPORT BERHASIL, EXPORT GAGAL.
 * Yang belum: Proses Klaim, Generate DLA, Cek Premi — ketiganya memanggil mesin yang
 * modulnya belum dibangun.
 *
 * Tombol yang mesinnya belum dibangun TETAP DITAMPILKAN dalam keadaan nonaktif beserta
 * alasannya, mengikuti perlakuan yang sama pada butir menu yang belum punya layar
 * (keputusan Work Owner 2026-09-18). Dengan begitu kemajuan migrasi terbaca langsung dari
 * layar, dan petugas tidak melaporkan tombol yang "hilang".
 *
 * Menyembunyikannya akan membuat layar ini tampak SELESAI padahal separuh alurnya belum
 * ada — dan itu kesalahpahaman yang paling mahal di antara semua pilihan.
 */
export function AutoClaimInboxPage() {
  const portal = useSelectedPortal((state) => state.alias)

  // Tab yang sedang terbuka. Bawaannya datang dari server supaya layar tidak memuat
  // daftar tab-nya sendiri.
  const [source, setSource] = useState('')

  const [company, setCompany] = useState('')
  const [page, setPage] = useState(1)
  const [uploadOpen, setUploadOpen] = useState(false)
  const [opened, setOpened] = useState<OpenedBatch | null>(null)
  const [uploadNote, setUploadNote] = useState<AutoClaimUploadResponse | null>(null)

  const tabList = useAutoClaimTabList()
  const activeSource = source === '' ? (tabList.data?.bawaan ?? '') : source
  const activeTable = tabList.data?.tab.find((t) => t.kode === activeSource)?.tabel ?? ''

  const list = useAutoClaimBatchList(activeSource, company, page)
  const exportFile = useExportAutoClaim()

  // Berpindah tab mengosongkan penyaring perusahaan dan menutup rincian yang terbuka.
  //
  // Bukan kerapian: ketiga tab membaca TABEL yang berbeda, sehingga kode perusahaan yang
  // dipilih pada satu tab belum tentu ada di tab lain — dan rincian batch yang terbuka
  // pasti tidak ada. Membawanya ikut berpindah akan menampilkan tabel kosong atau galat
  // "batch tidak ditemukan" yang sebabnya tidak terbaca di layar.
  function changeSource(value: string) {
    setSource(value)
    setCompany('')
    setPage(1)
    setOpened(null)
    exportFile.reset()
  }

  function changeCompany(value: string) {
    setCompany(value)
    // Halaman dikembalikan ke 1 setiap kali penyaring berubah. Tanpa itu, menyaring
    // perusahaan yang hanya punya satu halaman sementara pengguna berada di halaman 3
    // akan menampilkan tabel kosong yang tampak seperti "tidak ada data".
    setPage(1)
    setOpened(null)
  }

  function openDetail(row: AutoClaimBatch) {
    exportFile.reset()
    setOpened({ company: row.kode_perusahaan, companyName: row.nama_perusahaan, batch: row.batch })
  }

  const columns: Column<AutoClaimBatch>[] = [
    {
      key: 'kode',
      title: 'KODE',
      width: '6rem',
      value: (row) => row.kode_perusahaan,
    },
    {
      key: 'perusahaan',
      title: 'Nama Perusahaan',
      value: (row) => `${row.nama_perusahaan} ${row.kode_perusahaan}`,
      render: (row) =>
        row.nama_perusahaan === '' ? (
          // Kode yang tidak ada di Master Auto Claim ditandai terang-terangan. Baris
          // seperti ini TIDAK AKAN PERNAH berhasil diproses — pencarian penerima klaim
          // membaca master yang sama — jadi petugas perlu melihatnya, bukan menebak
          // kenapa kolomnya kosong.
          <span className="flex flex-col">
            <span className="text-slate-500">Tidak terdaftar di Master Auto Claim</span>
            <span className="text-xs text-amber-700">
              Kode {row.kode_perusahaan} perlu didaftarkan
            </span>
          </span>
        ) : (
          row.nama_perusahaan
        ),
    },
    { key: 'batch', title: 'Batch', width: '5.5rem', value: (row) => row.batch },
    {
      // Kolom ini bagian KUNCI PENGELOMPOKAN, bukan tambahan tampilan. Grid Pega
      // mengelompokkan menurut (batch, inisialid, userinput, nama_penerima, tanggal
      // tglproses), sehingga satu nomor batch yang diunggah pada dua tanggal berbeda
      // tampil sebagai DUA baris. Tanpa kolomnya, kedua baris itu tampak kembar dan
      // petugas tidak punya cara membedakannya.
      key: 'tanggal',
      title: 'Tgl Proses',
      width: '7rem',
      value: (row) => row.tanggal_proses,
      render: (row) => <span className="tabular-nums">{row.tanggal_proses}</span>,
    },
    {
      key: 'upload',
      title: 'Di Upload',
      width: '6rem',
      alignRight: true,
      value: (row) => String(row.jumlah_upload),
      render: (row) => <span className="tabular-nums">{row.jumlah_upload}</span>,
    },
    {
      key: 'proses',
      title: 'Diproses',
      width: '7rem',
      alignRight: true,
      value: (row) => String(row.jumlah_proses),
      render: (row) => (
        <span className="tabular-nums">
          {row.jumlah_proses}
          {/* Sisa yang belum diproses disebut di sini, bukan sebagai kolom kesembilan:
              ia selisih dua kolom yang sudah ada, dan kolom baru akan memperlebar grid
              tanpa menambah informasi. */}
          {row.jumlah_belum_proses > 0 && (
            <span className="ml-1.5 text-xs font-normal text-amber-700">
              +{row.jumlah_belum_proses} menunggu
            </span>
          )}
        </span>
      ),
    },
    {
      key: 'berhasil',
      title: 'Berhasil',
      width: '6rem',
      alignRight: true,
      value: (row) => String(row.jumlah_berhasil),
      render: (row) => (
        <span className="tabular-nums font-medium text-emerald-700">{row.jumlah_berhasil}</span>
      ),
    },
    {
      key: 'gagal',
      title: 'Gagal',
      width: '5.5rem',
      alignRight: true,
      value: (row) => String(row.jumlah_gagal),
      render: (row) => (
        <span
          className={
            row.jumlah_gagal > 0
              ? 'tabular-nums font-medium text-red-700'
              : 'tabular-nums text-slate-400'
          }
        >
          {row.jumlah_gagal}
        </span>
      ),
    },
    {
      key: 'user',
      title: 'User Upload',
      width: '9rem',
      value: (row) => row.user_upload,
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: '17rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <span className="inline-flex flex-wrap justify-end gap-1.5">
          <Button
            tone="kedua"
            onClick={() => openDetail(row)}
            aria-label={`Detail batch ${row.batch} ${row.kode_perusahaan}`}
          >
            Detail
          </Button>
          <Button
            tone="kedua"
            disabled={row.jumlah_berhasil === 0 || exportFile.isPending}
            aria-label={`Export berhasil batch ${row.batch} ${row.kode_perusahaan}`}
            onClick={() =>
              exportFile.mutate({
                company: row.kode_perusahaan,
                source: activeSource,
                batch: row.batch,
                hasil: 'berhasil',
              })
            }
          >
            Export Berhasil
          </Button>
          <Button
            tone="kedua"
            disabled={row.jumlah_gagal === 0 || exportFile.isPending}
            aria-label={`Export gagal batch ${row.batch} ${row.kode_perusahaan}`}
            onClick={() =>
              exportFile.mutate({
                source: activeSource,
                company: row.kode_perusahaan,
                batch: row.batch,
                hasil: 'gagal',
              })
            }
          >
            Export Gagal
          </Button>
        </span>
      ),
    },
  ]

  return (
    <main className="mx-auto max-w-7xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Inbox Auto Claim</h1>
          <p className="text-sm text-slate-600">
            Batch klaim borongan dari perusahaan rekanan beserta hasil pemrosesannya.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
          <Button tone="utama" onClick={() => setUploadOpen(true)} disabled={uploadOpen}>
            Upload Data Klaim
          </Button>
        </div>
      </header>

      <p className="mt-3 text-xs text-slate-500">
        Portal entitas:{' '}
        <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
      </p>

      {/* Tab duduk di atas segalanya, seperti pada harness lama: ia memilih TABEL yang
          dibaca seluruh layar, bukan menyaring isi tabel yang sama. */}
      <div className="mt-4">
        <SourceTab tab={tabList.data?.tab ?? []} selected={activeSource} onSelect={changeSource} />
      </div>

      {/* Panel ringkasan duduk DI ATAS grid dan sekaligus menjadi SATU-SATUNYA penyaring
          perusahaan — dropdown yang dulu ada di bilah judul grid sudah dibuang (keputusan
          Work Owner 2026-09-20), mengikuti layar Pega yang juga tidak punya dropdown.
          Baris "All" pada tabel ringkasan inilah cara membatalkan penyaringnya. */}
      <CompanySummary source={activeSource} selected={company} onSelect={changeCompany} />

      <PendingActions />

      {uploadOpen && (
        <section className="mt-5">
          <UploadForm
            source={activeSource}
            onClose={() => setUploadOpen(false)}
            onUploaded={(result) => {
              setUploadNote(result)
              setUploadOpen(false)
            }}
          />
        </section>
      )}

      {uploadNote && (
        // Panel hasil unggahan membedakan TIGA keadaan, dan pembedaannya bukan kerapian
        // tampilan: yang "bertanda" tersimpan dan terlihat di grid, yang "ditolak" tidak
        // ada di mana pun. Pengguna yang mengira keduanya sama akan mencari baris yang
        // tidak pernah tersimpan.
        <div
          role="status"
          className={
            uploadNote.ditolak.length > 0
              ? 'mt-5 rounded-kartu border border-amber-200 bg-amber-50/80 p-4 text-sm shadow-lembut'
              : 'mt-5 rounded-kartu border border-emerald-200 bg-emerald-50/80 p-4 text-sm shadow-lembut'
          }
        >
          <p className="font-medium text-slate-900">
            {uploadNote.jumlah_baris === 0
              ? 'Tidak ada baris yang tersimpan'
              : `${uploadNote.jumlah_baris} baris tersimpan`}
          </p>

          {uploadNote.batch.length > 0 && (
            <ul className="mt-1 space-y-0.5 text-slate-800">
              {uploadNote.batch.map((b) => (
                <li key={`${b.kode_perusahaan}-${b.batch}`}>
                  {b.nama_perusahaan === '' ? b.kode_perusahaan : b.nama_perusahaan} — batch{' '}
                  <span className="font-medium">{b.batch}</span>, {b.jumlah_baris} baris
                  {b.jumlah_bertanda > 0 && (
                    <span className="text-amber-800">
                      {' '}
                      ({b.jumlah_lolos} menunggu proses, {b.jumlah_bertanda} bertanda gagal)
                    </span>
                  )}
                </li>
              ))}
            </ul>
          )}

          {uploadNote.ditolak.length > 0 && (
            <div className="mt-3 border-t border-amber-200 pt-3">
              <p className="font-medium text-amber-900">
                {uploadNote.ditolak.length} baris TIDAK tersimpan
              </p>
              <p className="mt-0.5 text-xs text-amber-800">
                Perusahaan rekanannya tidak dapat ditemukan dari nomor polis, sehingga barisnya
                tidak punya tempat di grid mana pun. Perbaiki nomor polisnya lalu unggah ulang baris
                ini saja.
              </p>
              <ul className="mt-2 space-y-0.5 text-xs text-amber-900">
                {uploadNote.ditolak.map((row) => (
                  <li key={`${row.baris}-${row.nomor_polis}`}>
                    Baris <span className="font-medium tabular-nums">{row.baris}</span> —{' '}
                    {row.nomor_polis === '' ? '(nomor polis kosong)' : row.nomor_polis} —{' '}
                    {row.pesan}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {uploadNote.batch.length > 0 && (
            <p className="mt-2 text-xs text-slate-700">
              Baris yang menunggu proses belum menjadi klaim. Pemrosesan menunggu tombol Proses
              Klaim, yang belum tersedia di aplikasi ini.
            </p>
          )}

          <div className="mt-3">
            <Button tone="halus" onClick={() => setUploadNote(null)}>
              Tutup
            </Button>
          </div>
        </div>
      )}

      {exportFile.isError && (
        <div className="mt-5">
          <ErrorMessage
            title="Berkas gagal diunduh"
            description={
              exportFile.error instanceof APIError
                ? exportFile.error.message
                : 'Periksa koneksi jaringan Anda, lalu coba lagi.'
            }
            tone={exportFile.error instanceof NetworkError ? 'gangguan' : 'penolakan'}
          />
        </div>
      )}

      <section className="mt-6">
        {portal === null ? (
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Data klaim dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        ) : list.isError ? (
          (() => {
            const message = loadMessage(list.error)
            return (
              <ErrorMessage
                title={message.title}
                description={message.description}
                tone={message.tone}
              />
            )
          })()
        ) : (
          <DataTable
            columns={columns}
            rows={list.data?.batch ?? []}
            // Tanggal proses ikut menjadi kunci baris karena ia bagian kunci
            // pengelompokan: tanpa itu, dua baris yang nomor batch-nya sama akan berbagi
            // kunci React dan salah satunya tidak terbarui saat data berubah.
            rowKey={(row) => `${row.kode_perusahaan}-${row.batch}-${row.tanggal_proses}`}
            isLoading={list.isPending}
            title="Daftar Batch"
            // Diberi nama karena layar ini memuat DUA tabel. Tanpa nama, pembaca layar
            // membacakan keduanya hanya sebagai "tabel".
            label="Daftar batch"
            // Nama tabelnya datang bersama tabnya, tidak diketik di sini: ketiga tab
            // membaca tabel yang berbeda, dan teks tetap akan salah pada dua dari tiga.
            description={`Sumber: ${activeTable === '' ? '…' : activeTable}`}
            // Pencarian di peramban disembunyikan: ia hanya menyaring halaman yang sedang
            // tampil, dan pengguna mengira ia mencari ke seluruh data.
            hideSearch
            emptyMessage={
              company === ''
                ? 'Belum ada batch klaim pada entitas ini.'
                : 'Perusahaan ini belum punya batch klaim.'
            }
            pagination={{
              page: list.data?.paginasi.halaman ?? page,
              size: list.data?.paginasi.ukuran ?? 15,
              total: list.data?.paginasi.total ?? 0,
              totalPage: list.data?.paginasi.total_halaman ?? 0,
              onPageChange: setPage,
              isLoading: list.isFetching,
            }}
          />
        )}
      </section>

      {opened && (
        <BatchDetail
          company={opened.company}
          companyName={opened.companyName}
          source={activeSource}
          batch={opened.batch}
          onClose={() => setOpened(null)}
        />
      )}
    </main>
  )
}

/**
 * Tiga tombol layar lama yang mesinnya belum dibangun.
 *
 * Mereka ditampilkan NONAKTIF beserta alasannya, bukan disembunyikan — perlakuan yang
 * sama dengan butir menu yang belum punya layar. Menyembunyikannya membuat layar ini
 * tampak selesai padahal separuh alurnya belum ada.
 *
 * Setiap tombol menyebut modul yang menahannya, supaya "kapan tersedia" dapat dijawab
 * tanpa membuka dokumen apa pun.
 */
function PendingActions() {
  const pending = [
    {
      label: 'Proses Klaim',
      reason:
        'Membuat case klaim dari setiap baris batch. Menunggu modul Input Register, Objek & Coverage, Estimasi, dan Akseptasi.',
    },
    {
      label: 'Generate DLA',
      reason: 'Menerbitkan DLA untuk klaim batch ini. Menunggu modul PLA/Pre-DLA/DLA.',
    },
    {
      label: 'Cek Premi',
      reason:
        'Membaca total premi dan total klaim lewat layanan REST di luar aplikasi. Menunggu modul Integrasi Sistem Luar.',
    },
  ]

  return (
    <section
      aria-labelledby="aksi-belum-tersedia"
      className="mt-4 rounded-kartu border border-slate-200 bg-slate-50/70 p-4"
    >
      <h2 id="aksi-belum-tersedia" className="text-sm font-medium text-slate-800">
        Belum tersedia di aplikasi ini
      </h2>
      <p className="mt-1 text-xs text-slate-600">
        Ketiga tindakan berikut ada di layar lama dan masih dikerjakan di Pega. Gunakan aplikasi
        lama untuk menjalankannya.
      </p>
      <ul className="mt-3 flex flex-col gap-2 sm:flex-row sm:flex-wrap">
        {pending.map((item) => (
          <li key={item.label} className="flex items-start gap-2">
            <Button tone="kedua" disabled aria-describedby={`alasan-${item.label}`}>
              {item.label}
            </Button>
            <span id={`alasan-${item.label}`} className="max-w-xs text-xs text-slate-600">
              {item.reason}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}
