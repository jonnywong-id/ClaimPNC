import { useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'
import { ReloadIcon } from '@/components/Icon'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { downloadOutstandingCSV, PAGE_SIZE, useOutstandingList } from './api'
import { DocumentStatusSummary } from './DocumentStatusSummary'
import type { DocumentStatusCode, OutstandingClaim } from './types'

/**
 * Layar Inbox Outstanding.
 *
 * Migrasi dari **`Harness/InboxRegister_Harness-Harness.xml`** beserta section yang
 * dimuatnya, `Section/InboxRegister_Section-Section.xml` — layar yang di dalamnya sendiri
 * berjudul **"Inbox Outstanding"** (`:2150`).
 *
 * | Hal | Sumbernya |
 * |---|---|
 * | Kolom dan judulnya | `Section/InboxRegister_Section-Section.xml` |
 * | Baris yang tampil | `RDB List/BrowseInboxOutstanding1-SQL.xml` |
 * | Batas data per lini | `Activity/InboxOutstanding_Act-Act.xml` |
 *
 * Bahwa section rujukan memakai properti yang PERSIS alias kueri itu — `.District` untuk
 * "Policy no", `.CountryID` untuk "Insured name", `.City` untuk "Branch name" — adalah
 * yang mengikat keduanya.
 *
 * # Ini layar PEMANTAUAN, bukan Inbox
 *
 * Isinya seluruh klaim yang masih berjalan, bukan pekerjaan pemanggil — `D-79` menyebut
 * layar semacam ini bukan Inbox meski namanya demikian. Karena itu barisnya tidak hilang
 * setelah dikerjakan, dan tidak ada tombol "Ambil".
 *
 * # Yang sengaja dibuat berbeda dari Pega
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Penyaring lini | potongan SQL dirangkai dari properti | parameter, dipilih di server |
 * | Klaim tanpa tugas terbuka | tidak muncul (INNER JOIN) | tetap muncul |
 * | Klaim dengan dua tugas | muncul dua kali | satu baris |
 * | Batas data yang tidak berlaku | diam | dinyatakan di layar |
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 *
 * # Yang BELUM ada
 *
 * Section rujukan memuat beberapa tab — ALL Case, Communication, Loss Adjuster, Temporary
 * Close — yang masing-masing punya kolom tambahan sendiri. Yang dibangun di sini adalah
 * daftar intinya; tab-tab itu menunggu keputusan lingkup.
 *
 * Tombol **Input Claim** (`CreateInputKlaim`) juga belum ada: ia membuka alur registrasi,
 * yang dimiliki modul lain.
 */
export function OutstandingPage() {
  const [search, setSearch] = useState('')
  const [offset, setOffset] = useState(0)

  // Status dokumen yang sedang dipilih. Diisi HANYA lewat panel ringkasan — tidak ada
  // dropdown terpisah, sama seperti Inbox Auto Claim dan sama seperti Pega, yang juga
  // memakai tab pada panelnya sebagai satu-satunya cara memilih.
  const [documentStatus, setDocumentStatus] = useState<DocumentStatusCode | ''>('')

  const list = useOutstandingList({ search, documentStatus, offset })

  /**
   * Mengganti status mengembalikan paginasi ke halaman pertama.
   *
   * Tanpa ini, pengguna yang sedang di halaman 5 lalu memilih status yang hanya punya 12
   * baris akan melihat grid KOSONG — dan tidak ada yang memberi tahu bahwa sebabnya
   * halaman, bukan penyaringnya.
   */
  function changeDocumentStatus(next: DocumentStatusCode | '') {
    setDocumentStatus(next)
    setOffset(0)
  }
  const portal = useSelectedPortal((state) => state.alias)
  const token = useSession((state) => state.token)

  const [downloading, setDownloading] = useState(false)
  const [downloadError, setDownloadError] = useState<string | null>(null)

  const rows = list.data?.klaim ?? []
  const total = list.data?.total ?? 0

  function changeSearch(next: string) {
    setSearch(next)
    // Halaman dikembalikan ke awal. Tanpa ini, mencari dari halaman empat akan
    // menampilkan tabel kosong yang tampak rusak.
    setOffset(0)
  }

  async function download() {
    if (token === null || portal === null) return

    setDownloading(true)
    setDownloadError(null)
    try {
      // Penyaring layar TIDAK dikirim: unduhan mencakup lini bisnis, bukan isi layar.
      await downloadOutstandingCSV(token, portal)
    } catch (failure) {
      setDownloadError(errorMessage(failure))
    } finally {
      setDownloading(false)
    }
  }

  /**
   * Kolom mengikuti `Section/InboxRegister_Section-Section.xml`, section yang dimuat
   * harness rujukan — judulnya pun sama persis, berbahasa Inggris seperti di sana (`D-13`).
   *
   * Seluruh kolom layar lama ada di sini. `Business source` dan `Aging` sempat
   * dikecualikan karena modul ini mula-mula dibangun di atas tabel yang salah; keduanya
   * kembali begitu sumbernya dibetulkan ke `POOLDATA.T_CLAIMLIST_ADMIN`, yang memuat
   * `SOBNAME` dan `AGING` sebagai kolom biasa.
   */
  const columns: Column<OutstandingClaim>[] = [
    {
      key: 'claim_no',
      title: 'Claim no',
      width: '11rem',
      value: (c) => c.nomor_klaim,
      render: (c) =>
        c.nomor_klaim ? (
          <span className="inline-flex items-center rounded-md bg-blue-50 px-2 py-0.5 font-mono text-xs font-medium text-blue-700 ring-1 ring-blue-100">
            {c.nomor_klaim}
          </span>
        ) : (
          <span
            className="text-slate-400"
            title="Nomor terbit setelah tahap Input Register lolos validasi"
          >
            belum bernomor
          </span>
        ),
    },
    {
      key: 'policy_no',
      title: 'Policy no',
      width: '11rem',
      value: (c) => c.nomor_polis,
      render: (c) => (
        <span className="truncate font-mono text-xs text-slate-700">{c.nomor_polis || '—'}</span>
      ),
    },
    {
      key: 'insured_name',
      title: 'Insured name',
      value: (c) => c.nama_tertanggung,
      render: (c) => <span className="truncate">{c.nama_tertanggung || '—'}</span>,
    },
    {
      key: 'business_name',
      title: 'Business Name',
      width: '10rem',
      value: (c) => c.nama_bisnis,
      render: (c) => <span className="truncate">{c.nama_bisnis || '—'}</span>,
    },
    {
      key: 'business_source',
      title: 'Business source',
      width: '9rem',
      value: (c) => c.sumber_bisnis,
      render: (c) => <span className="truncate">{c.sumber_bisnis || '—'}</span>,
    },
    {
      key: 'branch_name',
      title: 'Branch name',
      width: '8rem',
      value: (c) => c.nama_cabang,
      render: (c) => <span className="truncate">{c.nama_cabang || '—'}</span>,
    },
    {
      key: 'admin_name',
      title: 'Admin name',
      width: '10rem',
      value: (c) => c.admin_pnc,
      render: (c) => <span className="truncate">{c.admin_pnc || '—'}</span>,
    },
    {
      key: 'register_date',
      title: 'Register Date',
      width: '8rem',
      value: (c) => c.tanggal_pendaftaran,
      render: (c) => (
        <span className="tabular-nums">
          {c.tanggal_pendaftaran ? formatDate(c.tanggal_pendaftaran) : '—'}
        </span>
      ),
    },
    {
      key: 'date_of_loss',
      title: 'Date of loss',
      width: '8rem',
      value: (c) => c.tanggal_kejadian,
      render: (c) => (
        <span className="tabular-nums">
          {c.tanggal_kejadian ? formatDate(c.tanggal_kejadian) : '—'}
        </span>
      ),
    },
    {
      key: 'total_aging',
      title: 'Total Aging',
      width: '7rem',
      value: (c) => String(c.umur_hari),
      render: (c) => <AgeBadge days={c.umur_hari} />,
    },
    {
      key: 'aging',
      title: 'Aging',
      width: '6rem',
      value: (c) => (c.aging_hari === null ? '' : String(c.aging_hari)),
      render: (c) =>
        // null berarti belum terisi, dan itu berbeda dari nol hari. Menampilkan "0"
        // untuk keduanya akan menyembunyikan data yang belum ada.
        c.aging_hari === null ? (
          <span className="text-slate-400" title="Belum terisi di sistem lama">
            —
          </span>
        ) : (
          <span className="tabular-nums text-slate-700">{c.aging_hari} hari</span>
        ),
    },
    {
      key: 'claim_status',
      title: 'Claim status',
      width: '8rem',
      value: (c) => c.status_tampil,
      render: (c) => <StatusBadge status={c.status_tampil} />,
    },
    {
      key: 'status_asm',
      title: 'Status ASM',
      width: '10rem',
      value: (c) => c.status_klaim,
      // Ditampilkan APA ADANYA, tanpa menyatakan isinya kode atau label.
      //
      // Sempat dirender `font-mono` dengan tooltip "Kode status klaim" — sebuah klaim
      // yang tidak dapat dibuktikan. Kolom sumbernya `STATUSLOCK_1 VARCHAR2(100)`, dan
      // seratus karakter adalah lebar untuk LABEL, bukan untuk kode empat digit. Isinya
      // sendiri belum pernah terlihat: baris produksi yang diserahkan Work Owner tidak
      // menyertakan kolom itu.
      //
      // Menyatakan yang salah lebih buruk daripada tidak menyatakan apa pun — pengguna
      // yang membaca "kode" lalu melihat kalimat utuh akan menyangka layarnya rusak.
      render: (c) =>
        c.status_klaim ? (
          <span className="truncate text-slate-700">{c.status_klaim}</span>
        ) : (
          <span className="text-slate-400">—</span>
        ),
    },
    {
      key: 'asm_pic',
      title: 'ASM PIC',
      width: '10rem',
      value: (c) => c.pic_teknik,
      render: (c) => <span className="truncate">{c.pic_teknik || '—'}</span>,
    },
  ]

  return (
    <div className="mx-auto max-w-7xl px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Inbox Outstanding</h1>
        <p className="mt-1 text-sm text-slate-600">
          Seluruh klaim yang masih berjalan pada entitas yang sedang dibuka, beserta tahap
          dan pemegang tugasnya.
        </p>
      </header>

      {list.data && <OwnerNotice pemilik={list.data.pemilik} />}

      {portal === null && (
        <div className="mt-6">
          <ErrorMessage
            title="Pilih entitas lebih dulu"
            description="Layar ini membaca klaim milik satu badan hukum, sehingga entitasnya harus dipilih di bilah atas."
            tone="gangguan"
          />
        </div>
      )}

      {downloadError && (
        <div className="mt-6">
          <ErrorMessage
            title="Berkas tidak dapat diunduh"
            description={downloadError}
            tone="gangguan"
          />
        </div>
      )}

      {/*
        Panel ringkasan diletakkan DI ATAS grid, mengikuti tata letak Pega — donut dan
        tabnya berada di atas daftar klaim, bukan di sampingnya.

        Ia memakai penyaring layar yang sama kecuali status, supaya angka donut selalu
        meringkas apa yang sedang dilihat pengguna.
      */}
      <DocumentStatusSummary
        filter={{ search }}
        selected={documentStatus}
        onSelect={changeDocumentStatus}
      />

      <div className="mt-6">
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(c) => c.klaim_id}
          title="Klaim berjalan"
          // Prop tidak dikirim sama sekali saat kosong, bukan dikirim bernilai undefined:
          // tsconfig memakai exactOptionalPropertyTypes, yang membedakan keduanya.
          {...(total > 0 ? { description: `${total} klaim masih berjalan.` } : {})}
          isLoading={list.isPending && portal !== null}
          searchLabel="Cari No Klaim / No Polis / PIC"
          emptyMessage="Tidak ada klaim yang masih berjalan."
          serverSearch={{ value: search, onChange: changeSearch, matchCount: total }}
          actions={
            <>
              <Button
                tone="halus"
                onClick={() => void list.refetch()}
                disabled={list.isFetching || portal === null}
              >
                <ReloadIcon className="h-4 w-4" />
                {list.isFetching ? 'Memuat…' : 'Muat ulang'}
              </Button>
              {/*
                Tombol ini SENGAJA tidak lagi dimatikan saat daftar kosong.

                Sebelumnya `disabled` memuat `total === 0`, sehingga petugas yang tidak
                sedang memegang satu pun tugas tidak dapat menekannya sama sekali. Itu
                keliru: unduhan tidak menyaring pemilik pekerjaan, sehingga berkasnya tetap
                berisi meski inbox-nya kosong.

                Terbukti pada data ASM: seorang petugas berlini NONMBU dengan 0 pekerjaan
                di inbox tetap mengunduh 354 baris.
              */}
              <Button
                tone="kedua"
                onClick={() => void download()}
                disabled={downloading || portal === null}
                title="Unduh seluruh klaim berjalan dalam lini bisnis Anda — bukan hanya isi layar ini"
              >
                {downloading ? 'Menyiapkan…' : 'Unduh CSV'}
              </Button>
            </>
          }
          error={
            list.isError ? (
              <ErrorMessage
                title="Daftar klaim tidak dapat dimuat"
                description={errorMessage(list.error)}
                tone="gangguan"
              />
            ) : undefined
          }
        />
      </div>

      <Pagination
        offset={offset}
        shown={rows.length}
        total={total}
        onChange={setOffset}
        busy={list.isFetching}
      />
    </div>
  )
}

/** * OwnerNotice menyatakan pekerjaan SIAPA yang sedang ditampilkan. * * Ini bukan hiasan. Daftar kosong pada layar bernama "My Inbox" punya dua sebab yang * tampak persis sama — memang tidak ada pekerjaan, atau penyaringnya salah orang — dan * tanpa keterangan ini pengguna tidak punya cara membedakannya. */function OwnerNotice({ pemilik }: { pemilik: string }) {  return (    <p className="mt-4 text-sm text-slate-600">      Menampilkan pekerjaan milik{' '}      <span className="font-medium text-slate-900">{pemilik}</span>.    </p>  )}

/**
 * Umur klaim, dengan penegasan pada yang sudah lama.
 *
 * Pembedaannya TIDAK hanya warna: angka dan satuannya tetap terbaca, dan ambangnya
 * dijelaskan lewat title. Pembedaan yang hanya mengandalkan warna tidak terbaca pengguna
 * dengan gangguan penglihatan warna.
 */
function AgeBadge({ days }: { days: number }) {
  const lama = days >= 30

  return (
    <span
      className={[
        'inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium tabular-nums',
        lama ? 'bg-amber-50 text-amber-800 ring-1 ring-amber-200' : 'text-slate-700',
      ].join(' ')}
      title={lama ? 'Berjalan 30 hari atau lebih' : undefined}
    >
      {days} hari
    </span>
  )
}

/** Status yang dilihat pengguna; teksnya mengikuti layar Pega apa adanya (`D-13`). */
function StatusBadge({ status }: { status: string }) {
  const tone =
    status === 'Reject'
      ? 'bg-red-50 text-red-700 ring-red-100'
      : status === 'Close'
        ? 'bg-slate-100 text-slate-700 ring-slate-200'
        : 'bg-emerald-50 text-emerald-700 ring-emerald-100'

  return (
    <span
      className={`inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium ring-1 ${tone}`}
    >
      {status}
    </span>
  )
}

/**
 * Paginasi "muat halaman berikutnya", bukan nomor halaman.
 *
 * `10-API-STRATEGY.md` §4 menghindari nomor halaman pada data besar. Di sini total memang
 * tersedia, sehingga keterangannya dapat menyebut angka — tetapi navigasinya tetap maju
 * mundur satu halaman, bukan melompat ke halaman sekian.
 */
function Pagination({
  offset,
  shown,
  total,
  onChange,
  busy,
}: {
  offset: number
  shown: number
  total: number
  onChange: (next: number) => void
  busy: boolean
}) {
  if (total === 0) return null

  const first = offset + 1
  const last = offset + shown
  const hasPrevious = offset > 0
  const hasNext = last < total

  return (
    <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
      <p className="text-sm text-slate-600 tabular-nums">
        Menampilkan {first}–{last} dari {total}
      </p>
      <div className="flex gap-2">
        <Button
          tone="halus"
          onClick={() => onChange(Math.max(0, offset - PAGE_SIZE))}
          disabled={!hasPrevious || busy}
        >
          Sebelumnya
        </Button>
        <Button
          tone="halus"
          onClick={() => onChange(offset + PAGE_SIZE)}
          disabled={!hasNext || busy}
        >
          Berikutnya
        </Button>
      </div>
    </div>
  )
}

function errorMessage(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}
