import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type RecoveryPrincipalGroup, type RecoveryRow } from '@/api/types'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatMoney } from '@/lib/money'

import { PAGE_SIZE, useRecoveryList, useViewDocument } from './api'

type Props = {
  /** Tombol-tombol di kanan judul — Tambah dan Refresh, seperti di layar lama. */
  actions?: React.ReactNode
}

/**
 * Tab **Outstanding** — satu baris per principal, isinya terbuka saat diklik.
 *
 * Menggantikan grid `Data_BACTH_RECOVERY.pxResults` pada
 * `Section/OutstandingMasterRecovery-Section.xml:17510` beserta grid dalamnya.
 *
 * # Bentuknya dibaca dari layar Pega yang berjalan, bukan dari export
 *
 * Work Owner menunjukkannya pada 2026-09-29. Grid luar memuat satu baris per principal;
 * mengekliknya membuka grid kedua berisi seluruh batch principal itu, dari yang paling
 * lama, masing-masing dengan tombol **View Document**.
 *
 * Itu tidak dapat disimpulkan dari export: `pyEnableGrouping` dan `pyRDLShowDetails`
 * keduanya `false`, sehingga pengelompokannya bukan bawaan grid — ia dikerjakan rule
 * pemuat halaman, yang hilang (`R-16`).
 *
 * # Kelima kolom luar, dipetakan dari section
 *
 * | Kolom | Properti Pega | Kolom basis data |
 * |---|---|---|
 * | Nama Principal | `.NamaPrincipal` | `NAMAPRINCIPAL` |
 * | Nilai Klaim | `.TotalListClaimAmountIDR` | `NILAIKLAIM` |
 * | Nilai Pembayaran Sebelumnya | `.NilaiDeductible` | `NILAIRECOVERY` |
 * | Pembayaran | `.ClaimAmountAdjust` | `PEMBAYARAN` |
 * | Sisa | `.TPLAmount` | `SISAKLAIM` |
 *
 * Angka baris luar adalah angka batch **terakhir**, bukan jumlah — lihat
 * `masterrecovery.PrincipalGroup` di backend.
 *
 * # Yang TIDAK dapat ditiru, dan itu disebut terus terang
 *
 * Penyaring "outstanding" tidak dibuat: klausa WHERE aslinya ada di rule yang hilang.
 * Menyaring atas dasar tebakan berarti menyembunyikan baris, dan baris yang hilang
 * diam-diam tidak akan pernah dikeluhkan siapa pun.
 */
export function OutstandingTable({ actions }: Props) {
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)

  const list = useRecoveryList({ cari: search, halaman: page })

  const rows = list.data?.principal ?? []
  const total = list.data?.total ?? 0
  const totalPage = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const columns: Column<RecoveryPrincipalGroup>[] = [
    {
      key: 'nama_principal',
      title: 'Nama Principal',
      value: (row) => row.nama_principal,
      render: (row) => (
        <span className="font-medium text-slate-900">
          {row.nama_principal}
          {row.batch.length > 1 && (
            <span className="ml-2 text-xs font-normal text-slate-500">
              {row.batch.length} batch
            </span>
          )}
        </span>
      ),
    },
    money('nilai_klaim', 'Nilai Klaim', (row) => row.nilai_klaim),
    money('pembayaran_sebelumnya', 'Nilai Pembayaran Sebelumnya', (row) => row.pembayaran_sebelumnya),
    money('pembayaran', 'Pembayaran', (row) => row.pembayaran),
    {
      key: 'sisa',
      title: 'Sisa',
      width: '10rem',
      alignRight: true,
      value: (row) => String(row.sisa).padStart(20, '0'),
      render: (row) => (
        /*
          Sisa negatif diberi warna karena ia menandakan pembayaran melampaui nilai klaim —
          keadaan yang sah tercatat, tetapi yang layak diperiksa orang. Sistem lama
          menampilkannya sama saja dengan angka lain.
        */
        <span
          className={
            'font-mono text-xs font-medium tabular-nums ' +
            (row.sisa < 0 ? 'text-red-700' : 'text-slate-900')
          }
        >
          {formatMoney(row.sisa)}
        </span>
      ),
    },
  ]

  return (
    <DataTable
      columns={columns}
      rows={rows}
      rowKey={(row) => row.nama_principal}
      label="Daftar principal recovery"
      actions={actions}
      isLoading={list.isPending}
      error={list.isError ? <ListErrorMessage error={list.error} /> : undefined}
      // Kepala kolom tetap digambar saat kosong, sama seperti grid Pega — bukan diganti
      // kotak kosong.
      showHeaderWhenEmpty
      emptyMessage="Belum ada batch recovery yang tercatat pada entitas ini."
      serverSearch={{
        value: search,
        onChange: (value) => {
          setSearch(value)
          // Kembali ke halaman pertama: hasil pencarian baru hampir pasti lebih pendek,
          // dan bertahan di halaman lima akan menampilkan daftar kosong yang membingungkan.
          setPage(1)
        },
        matchCount: total,
      }}
      searchLabel="Cari nama principal"
      pagination={{
        page,
        size: PAGE_SIZE,
        total,
        totalPage,
        onPageChange: setPage,
        isLoading: list.isFetching,
      }}
      expandedRow={(row) => <BatchTable rows={row.batch} />}
    />
  )
}

function money(
  key: string,
  title: string,
  ambil: (row: RecoveryPrincipalGroup) => number,
): Column<RecoveryPrincipalGroup> {
  return {
    key,
    title,
    width: '11rem',
    alignRight: true,
    // Yang diurutkan adalah angka berlapis nol, bukan teks terformat: "1.000.000" dan
    // "900.000" diurutkan sebagai teks akan menaruh yang kecil di atas.
    value: (row) => String(ambil(row)).padStart(20, '0'),
    render: (row) => (
      <span className="font-mono text-xs tabular-nums text-slate-800">
        {formatMoney(ambil(row))}
      </span>
    ),
  }
}

/**
 * Grid DALAM — riwayat batch satu principal.
 *
 * Kesepuluh kolomnya mengikuti layar lama apa adanya, termasuk urutannya: Tanggal Input,
 * NO HPLL, Tahun, Nilai Klaim, Nilai Pembayaran Sebelumnya, Nilai Pembayaran, Sisa Klaim,
 * Keterangan, Posisi, dan tombol View Document.
 *
 * Ia ditulis sebagai tabel biasa, bukan `DataTable` kedua: yang bersarang tidak butuh
 * pencarian, pengurutan, maupun paginasi sendiri — seluruh riwayat satu principal sudah
 * di tangan — dan menyarangkan dua bilah pencarian pada satu layar justru membingungkan.
 */
function BatchTable({ rows }: { rows: RecoveryRow[] }) {
  const [modal, setModal] = useState<RecoveryRow | null>(null)

  return (
    <div className="overflow-x-auto rounded-kontrol border border-slate-200 bg-white">
      <table className="w-full min-w-[56rem] border-collapse text-left text-xs">
        <caption className="sr-only">Riwayat batch principal ini</caption>
        <thead>
          <tr className="border-b border-slate-200 bg-slate-50">
            {[
              'Tanggal Input',
              'NO HPLL',
              'Tahun',
              'Nilai Klaim',
              'Nilai Pembayaran Sebelumnya',
              'Nilai Pembayaran',
              'Sisa Klaim',
              'Keterangan',
              'Posisi',
              '',
            ].map((judul, index) => (
              <th
                key={judul || `aksi-${index}`}
                scope="col"
                className={
                  'px-3 py-2 font-semibold text-slate-600 ' +
                  (index >= 3 && index <= 6 ? 'text-right' : '')
                }
              >
                {judul}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.batch} className="border-b border-slate-100 last:border-0">
              <td className="px-3 py-2 whitespace-nowrap text-slate-800">
                {formatInputDate(row.tanggal_input)}
              </td>
              <td className="px-3 py-2 text-slate-800">{row.no_hpll || '—'}</td>
              <td className="px-3 py-2 text-slate-800">{row.tahun || '—'}</td>
              <td className="px-3 py-2 text-right font-mono tabular-nums text-slate-800">
                {formatMoney(row.nilai_klaim)}
              </td>
              <td className="px-3 py-2 text-right font-mono tabular-nums text-slate-800">
                {formatMoney(row.pembayaran_sebelumnya)}
              </td>
              <td className="px-3 py-2 text-right font-mono tabular-nums text-slate-800">
                {formatMoney(row.pembayaran)}
              </td>
              <td
                className={
                  'px-3 py-2 text-right font-mono font-medium tabular-nums ' +
                  (row.sisa < 0 ? 'text-red-700' : 'text-slate-900')
                }
              >
                {formatMoney(row.sisa)}
              </td>
              <td className="px-3 py-2 text-slate-800">{row.keterangan || '—'}</td>
              <td className="px-3 py-2 text-slate-800">{row.posisi_kasus || '—'}</td>
              <td className="px-3 py-2 text-right">
                {/*
                  Tombol SELALU tampil, sama seperti layar lama — termasuk pada batch tanpa
                  lampiran. Yang membedakan adalah isi modalnya: "Data Tidak Ada".

                  Ini berubah dari versi sebelumnya, yang menyembunyikan tombol pada batch
                  tanpa lampiran. Menyembunyikannya tampak lebih ramah, tetapi membuat
                  petugas tidak punya cara memastikan sebuah batch memang belum berlampiran
                  — dan layar lama menjawab pertanyaan itu lewat modalnya.
                */}
                <Button
                  tone="halus"
                  onClick={(event) => {
                    // Baris luar adalah tombol pembuka/penutup; tanpa ini, menekan
                    // View Document ikut menutup barisnya.
                    event.stopPropagation()
                    setModal(row)
                  }}
                >
                  View Document
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {modal !== null && (
        <SupportingDocumentModal batch={modal} onClose={() => setModal(null)} />
      )}
    </div>
  )
}

/**
 * Modal **View Dokument Pendukung**.
 *
 * Meniru modal yang dibuka tombol View Document pada layar lama: judulnya sama, tabelnya
 * berkolom **Input Nama** dan **Tanggal**, dan saat kosong ia menulis **Data Tidak Ada**.
 *
 * # Kenapa ia daftar, bukan langsung membuka berkas
 *
 * Karena itulah yang dilakukan layar lama — terlihat pada tangkapan layar Work Owner
 * (2026-09-29): tombolnya membuka modal berisi daftar, bukan mengunduh sesuatu.
 *
 * Rule yang membangunnya TIDAK ADA di export: teks "View Dokument" nol kemunculan di
 * seluruh berkas (`R-16`). Pemetaan kedua kolomnya disandarkan pada kolom yang memang ada
 * di `POOLDATA.DATA_ATTACHFILE` — `INPUTOPERATOR` dan `INPUTDATE`.
 *
 * # Satu batas yang harus disebut
 *
 * Tautan batch ke lampiran di sistem lama hanyalah SATU kolom, `DOKUMENID`. Karena itu
 * daftar ini berisi paling banyak satu baris hari ini. Bentuknya tetap dibuat daftar
 * supaya sama dengan layar lama, dan supaya menambah lampiran kedua kelak tidak menuntut
 * layar ini dirombak.
 */
function SupportingDocumentModal({
  batch,
  onClose,
}: {
  batch: RecoveryRow
  onClose: () => void
}) {
  const view = useViewDocument()
  const dokumen = batch.dokumen

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-label="View Dokument Pendukung"
      // Mengeklik latar menutup modal, sama seperti menekan tombol silang. Klik di
      // dalamnya dihentikan supaya tidak ikut menutup.
      onClick={onClose}
    >
      <div
        className="w-full max-w-md overflow-hidden rounded-kartu bg-white shadow-lg"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4">
          <h3 className="text-base font-semibold text-slate-900">View Dokument Pendukung</h3>
          <button
            type="button"
            onClick={onClose}
            aria-label="Tutup"
            className="rounded p-1 text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50"
          >
            ✕
          </button>
        </div>

        <div className="px-5 py-4">
          <table className="w-full border-collapse text-left text-sm">
            <thead>
              <tr className="border-b border-slate-300">
                <th scope="col" className="py-2 font-medium text-slate-700">
                  Input Nama
                </th>
                <th scope="col" className="py-2 font-medium text-slate-700">
                  Tanggal
                </th>
                <th scope="col" className="py-2" />
              </tr>
            </thead>
            <tbody>
              {dokumen === null ? (
                <tr className="border-b border-slate-200">
                  {/* Kalimatnya sama persis dengan layar lama. */}
                  <td colSpan={3} className="py-3 text-slate-500">
                    Data Tidak Ada
                  </td>
                </tr>
              ) : (
                <tr className="border-b border-slate-200">
                  <td className="py-3 text-slate-800">{dokumen.input_nama || '—'}</td>
                  <td className="py-3 text-slate-800">{formatInputDate(dokumen.tanggal)}</td>
                  <td className="py-3 text-right">
                    <Button
                      tone="halus"
                      onClick={() => view.mutate(dokumen.id)}
                      disabled={view.isPending}
                    >
                      {view.isPending ? 'Membuka…' : 'Buka berkas'}
                    </Button>
                  </td>
                </tr>
              )}
            </tbody>
          </table>

          {dokumen !== null && dokumen.nama_berkas !== '' && (
            <p className="mt-3 text-xs text-slate-500">{dokumen.nama_berkas}</p>
          )}

          {view.isError && (
            <div className="mt-4">
              <DocumentErrorMessage error={view.error} />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

/**
 * Tanggal input ditampilkan sebagai `dd/mm/yyyy hh:mm`, sama dengan layar lama.
 *
 * Kosong menjadi tanda pisah, bukan "Invalid Date": baris warisan boleh tidak punya
 * INSERTDATE, dan itu bukan kesalahan yang perlu diteriakkan di setiap barisnya.
 *
 * Jamnya selalu WIB, bukan zona waktu komputer pengguna — `Asia/Jakarta` disebut namanya,
 * sama seperti layar lain, supaya pengguna di zona lain melihat jam yang sama.
 */
function formatInputDate(raw: string): string {
  if (raw === '') return '—'
  const waktu = new Date(raw)
  if (Number.isNaN(waktu.getTime())) return '—'

  const bagian = Object.fromEntries(
    INPUT_DATE_FORMAT.formatToParts(waktu).map((part) => [part.type, part.value]),
  )
  return `${bagian.day}/${bagian.month}/${bagian.year} ${bagian.hour}:${bagian.minute}`
}

const INPUT_DATE_FORMAT = new Intl.DateTimeFormat('en-GB', {
  timeZone: 'Asia/Jakarta',
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
})

function DocumentErrorMessage({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Bukti bayar belum dapat dibuka. Periksa koneksi lalu coba lagi."
        tone="gangguan"
      />
    )
  }

  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.documentNotFound:
        return (
          <ErrorMessage
            title="Bukti bayar tidak ditemukan"
            description="Penandanya menunjuk lampiran yang tidak ada. Unggah ulang bukti bayarnya."
            tone="penolakan"
          />
        )
      case ErrorCode.documentElsewhere:
        return (
          <ErrorMessage
            title="Berkasnya tidak tersimpan di basis data"
            description="Bukti bayar ini berada di penyimpanan dokumen yang belum terhubung ke aplikasi baru."
            tone="gangguan"
          />
        )
      default:
        return (
          <ErrorMessage
            title="Bukti bayar gagal dibuka"
            description={error.message}
            tone="gangguan"
          />
        )
    }
  }

  return (
    <ErrorMessage
      title="Bukti bayar gagal dibuka"
      description="Terjadi kesalahan pada sistem. Coba beberapa saat lagi."
      tone="gangguan"
    />
  )
}

/**
 * Gagal memuat daftar dibedakan dari daftar yang memang kosong.
 *
 * Tanpa pembedaan itu, gangguan jaringan terbaca sebagai "belum ada data" — dan petugas
 * dapat menyimpulkan batch yang sudah dicatatnya hilang.
 */
function ListErrorMessage({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Daftar belum dapat dimuat. Periksa koneksi lalu tekan Refresh."
        tone="gangguan"
      />
    )
  }

  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return (
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Daftar recovery dimiliki masing-masing entitas. Pilih portal entitas lebih dulu."
            tone="penolakan"
          />
        )
      case ErrorCode.portalNotReady:
        return (
          <ErrorMessage
            title="Basis data entitas ini belum tersedia"
            description="Entitasnya sudah direncanakan, tetapi kredensial basis datanya belum diisi. Hubungi administrator Claim PNC."
            tone="gangguan"
          />
        )
      default:
        return (
          <ErrorMessage title="Daftar gagal dimuat" description={error.message} tone="gangguan" />
        )
    }
  }

  return (
    <ErrorMessage
      title="Daftar gagal dimuat"
      description="Terjadi kesalahan pada sistem. Coba tekan Refresh."
      tone="gangguan"
    />
  )
}
