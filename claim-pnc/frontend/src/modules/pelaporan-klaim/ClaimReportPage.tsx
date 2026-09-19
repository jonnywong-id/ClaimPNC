import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ReportStage, type ClaimReport } from '@/api/types'
import { ReloadIcon, AddIcon, EditIcon } from '@/components/Icon'
import { ErrorMessage } from '@/components/ErrorMessage'
import { DataTable, type Column } from '@/components/DataTable'
import { Button } from '@/components/Button'

import { useLinkClaim, useReportList, useTransferReport } from './api'
import { ClaimReportForm } from './ClaimReportForm'
import { StageTabs } from './StageTabs'

/**
 * Layar Pelaporan Klaim.
 *
 * Menggantikan harness `InboxRCVApp_Harness` — yang di menu portal Pega berjudul
 * **"Inbox Laporan Klaim"** — beserta section `ViewStatusReceiveDocument` dan flow action
 * `InputReceiveDocument`.
 *
 * # Yang ditiru dari layar lama
 *
 * | Hal | Sumbernya |
 * |---|---|
 * | Tab per tahap beserta lencana jumlah | `Section/ViewStatusReceiveDocument-Section.xml` |
 * | Kolom daftar | `Report Definition/BrowseCaseReceivedDocList_RD-RD.xml` |
 * | Isian form | `Section/ViewInputReceiveDocument_sec-Section.xml` |
 * | Tidak ada Hapus | procedure lama hanya mengenal INSERT dan UPDATE |
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Judul tab | campur dua bahasa | seragam bahasa Indonesia |
 * | Perpindahan tahap | efek samping penyimpanan layar | tombol tersendiri, dan tidak dapat terjadi dua kali |
 * | Kunci layar | `StatusLock`, dapat dilewati jabatan tertentu | terkunci saat REGISTRASI, tanpa pengecualian |
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Pencarian | tiga rule SQL terpisah ber-`{ASIS:}` | satu endpoint berparameter |
 */
export function ClaimReportPage() {
  const [stage, setStage] = useState('')
  const [search, setSearch] = useState('')
  const [offset, setOffset] = useState(0)

  const list = useReportList({ stage, search, offset })
  const transfer = useTransferReport()
  const linkClaim = useLinkClaim()

  // null = form tertutup; { nomor: '' } bukan keadaan yang ada — menambah ditandai
  // formOpen tanpa baris terpilih.
  const [editing, setEditing] = useState<ClaimReport | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [linking, setLinking] = useState<ClaimReport | null>(null)

  const data = list.data
  const rows = data?.laporan ?? []

  function changeStage(next: string) {
    setStage(next)
    // Halaman dikembalikan ke awal. Tanpa ini, berpindah dari tab berisi 200 baris di
    // halaman empat ke tab berisi 3 baris akan menampilkan tabel kosong yang tampak rusak.
    setOffset(0)
  }

  function changeSearch(next: string) {
    setSearch(next)
    setOffset(0)
  }

  function openCreate() {
    setEditing(null)
    setLinking(null)
    setFormOpen(true)
  }

  function openEdit(report: ClaimReport) {
    setEditing(report)
    setLinking(null)
    setFormOpen(true)
  }

  function closeForm() {
    setFormOpen(false)
    setEditing(null)
  }

  const columns: Column<ClaimReport>[] = [
    {
      key: 'nomor',
      title: 'Nomor',
      width: '10rem',
      value: (r) => r.nomor,
      render: (r) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs font-medium text-slate-700 ring-1 ring-slate-200">
          {r.nomor}
        </span>
      ),
    },
    {
      key: 'tahap',
      title: 'Tahap',
      width: '11rem',
      value: (r) => r.tahap_label,
      render: (r) => <StageBadge stage={r.tahap} label={r.tahap_label} />,
    },
    {
      key: 'pelapor',
      title: 'Pelapor',
      value: (r) => r.nama_pelapor,
      render: (r) => (
        <div className="min-w-0">
          <p className="truncate font-medium text-slate-900">{r.nama_pelapor}</p>
          {r.email_pengirim && (
            <p className="truncate text-xs text-slate-500">{r.email_pengirim}</p>
          )}
        </div>
      ),
    },
    {
      key: 'tertanggung',
      title: 'Polis & tertanggung',
      value: (r) => `${r.nomor_polis} ${r.nama_tertanggung}`,
      render: (r) => (
        <div className="min-w-0">
          <p className="truncate text-slate-900">{r.nama_tertanggung || '—'}</p>
          <p className="truncate font-mono text-xs text-slate-500">{r.nomor_polis || '—'}</p>
        </div>
      ),
    },
    {
      key: 'tanggal_kejadian',
      title: 'Tgl kejadian',
      width: '9rem',
      value: (r) => r.tanggal_kejadian,
      render: (r) => <span className="tabular-nums">{readableDate(r.tanggal_kejadian)}</span>,
    },
    {
      key: 'nomor_klaim',
      title: 'No klaim',
      width: '10rem',
      value: (r) => r.nomor_klaim,
      render: (r) =>
        r.nomor_klaim ? (
          <span className="inline-flex items-center rounded-md bg-blue-50 px-2 py-0.5 font-mono text-xs font-medium text-blue-700 ring-1 ring-blue-100">
            {r.nomor_klaim}
          </span>
        ) : (
          <span className="text-slate-400" title="Laporan ini belum diregistrasi menjadi klaim">
            —
          </span>
        ),
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: '16rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (r) => (
        <div className="flex flex-wrap justify-end gap-1.5">
          <Button
            tone="halus"
            onClick={() => openEdit(r)}
            disabled={!r.dapat_diubah}
            title={
              r.dapat_diubah
                ? undefined
                : 'Laporan yang sudah menjadi klaim tidak dapat diubah lagi.'
            }
            aria-label={`Ubah laporan ${r.nomor}`}
          >
            <EditIcon className="h-3.5 w-3.5" />
            Ubah
          </Button>

          {r.dapat_ditransfer && (
            <Button
              tone="kedua"
              onClick={() => transfer.mutate(r.nomor)}
              disabled={transfer.isPending}
              aria-label={`Transfer laporan ${r.nomor} ke ASM pusat`}
            >
              Transfer
            </Button>
          )}

          {!r.nomor_klaim && r.ditransfer && (
            <Button
              tone="kedua"
              onClick={() => {
                setLinking(r)
                setFormOpen(false)
              }}
              aria-label={`Tautkan laporan ${r.nomor} ke nomor klaim`}
            >
              Registrasi
            </Button>
          )}
        </div>
      ),
    },
  ]

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <header className="mb-6">
        <nav aria-label="Jejak lokasi" className="mb-2 text-xs font-medium text-slate-500">
          <ol className="flex items-center gap-1.5">
            <li>Proses Klaim</li>
            <li aria-hidden="true" className="text-slate-300">
              /
            </li>
            <li className="text-slate-700">Pelaporan Klaim</li>
          </ol>
        </nav>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">Pelaporan Klaim</h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">
          Laporan kerugian yang masuk sebelum klaim diregistrasi — siapa yang melapor, polis
          dan tertanggung yang dirujuk, apa yang terjadi, dan berapa perkiraan kerugiannya.
          Tanggal terima dokumen yang dicatat di sini menjadi salah satu batas pada validasi
          registrasi.
        </p>
      </header>

      <div className="mb-5">
        <StageTabs
          summary={data?.ringkasan ?? []}
          active={stage}
          onSelect={changeStage}
          total={(data?.ringkasan ?? []).reduce((sum, s) => sum + s.jumlah, 0)}
          loading={list.isFetching}
        />
      </div>

      {(transfer.isError || linkClaim.isError) && (
        <div className="mb-5">
          <ActionErrorMessage error={transfer.error ?? linkClaim.error} />
        </div>
      )}

      {formOpen && (
        <div className="mb-6">
          <ClaimReportForm report={editing} onClose={closeForm} />
        </div>
      )}

      {linking && (
        <div className="mb-6">
          <RegistrationPanel
            report={linking}
            working={linkClaim.isPending}
            onClose={() => setLinking(null)}
            onSave={(claimNumber) =>
              linkClaim.mutate(
                { number: linking.nomor, claimNumber },
                { onSuccess: () => setLinking(null) },
              )
            }
          />
        </div>
      )}

      <DataTable
        columns={columns}
        rows={rows}
        rowKey={(r) => r.nomor}
        title="Daftar Laporan Klaim"
        description={data ? `${data.jumlah} laporan pada tahap ini.` : 'Memuat daftar laporan…'}
        searchLabel="Cari nomor laporan, nomor klaim, polis, tertanggung, atau pelapor"
        emptyMessage="Belum ada laporan pada tahap ini."
        isLoading={list.isPending}
        error={list.isError ? <LoadErrorMessage error={list.error} /> : undefined}
        serverSearch={{ value: search, onChange: changeSearch, matchCount: data?.jumlah }}
        actions={
          <>
            <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
              <ReloadIcon className={`h-4 w-4 ${list.isFetching ? 'animate-spin' : ''}`} />
              {list.isFetching ? 'Memuat…' : 'Muat ulang'}
            </Button>
            <Button tone="utama" onClick={openCreate} disabled={formOpen && !editing}>
              <AddIcon className="h-4 w-4" />
              Catat laporan
            </Button>
          </>
        }
      />

      {data && data.jumlah > rows.length && (
        <Pagination
          offset={data.lewati}
          limit={data.batas}
          total={data.jumlah}
          visible={rows.length}
          onMove={setOffset}
          loading={list.isFetching}
        />
      )}
    </div>
  )
}

/**
 * Lencana tahap: warna DAN teks, tidak pernah warna saja.
 *
 * Warna tidak terbaca oleh sekitar satu dari dua belas laki-laki yang mengalami buta warna
 * merah-hijau. Teksnya selalu ada, dan warnanya hanya mempercepat pembacaan bagi yang
 * dapat melihatnya.
 */
function StageBadge({ stage, label }: { stage: string; label: string }) {
  const styles: Record<string, string> = {
    [ReportStage.notTransferred]: 'bg-amber-50 text-amber-800 ring-amber-200',
    [ReportStage.notRegistered]: 'bg-sky-50 text-sky-800 ring-sky-200',
    [ReportStage.registered]: 'bg-blue-50 text-blue-800 ring-blue-200',
    [ReportStage.accepted]: 'bg-emerald-50 text-emerald-800 ring-emerald-200',
    [ReportStage.rejected]: 'bg-slate-100 text-slate-700 ring-slate-300',
  }

  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ${
        styles[stage] ?? 'bg-slate-100 text-slate-700 ring-slate-300'
      }`}
    >
      {label}
    </span>
  )
}

/** Panel kecil untuk menautkan laporan ke nomor klaim. */
function RegistrationPanel({
  report,
  working,
  onClose,
  onSave,
}: {
  report: ClaimReport
  working: boolean
  onClose: () => void
  onSave: (claimNumber: string) => void
}) {
  const [claimNumber, setClaimNumber] = useState('')

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault()
        if (claimNumber.trim() !== '') onSave(claimNumber.trim())
      }}
      className="overflow-hidden rounded-kartu border border-slate-200 border-l-4 border-l-blue-500 bg-white shadow-angkat"
      aria-label={`Tautkan laporan ${report.nomor} ke klaim`}
    >
      <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
        <h3 className="text-base font-semibold text-slate-900">
          Tautkan {report.nomor} ke klaim
        </h3>
        <p className="mt-1 text-sm text-slate-600">
          Isi nomor klaim yang lahir dari laporan ini. Setelah tertaut, laporannya tidak
          dapat diubah lagi — yang berlaku sejak saat itu adalah data klaimnya.
        </p>
      </div>

      <div className="flex flex-wrap items-end gap-3 p-5">
        <div className="min-w-[14rem] flex-1">
          <label htmlFor="nomor_klaim" className="block text-sm font-medium text-slate-700">
            Nomor klaim
          </label>
          <input
            id="nomor_klaim"
            value={claimNumber}
            onChange={(e) => setClaimNumber(e.target.value)}
            placeholder="PNCN.26.0148"
            autoComplete="off"
            disabled={working}
            className="mt-1.5 w-full rounded-kontrol border border-slate-300 bg-white px-3 py-2.5 text-sm text-slate-900 transition-[border-color,box-shadow] duration-150 ease-halus placeholder:text-slate-400 hover:border-slate-400 focus:border-blue-500 focus:outline-none focus:ring-4 focus:ring-blue-500/15 disabled:bg-slate-50"
          />
        </div>
        <Button type="submit" tone="utama" disabled={working || claimNumber.trim() === ''}>
          {working ? 'Menyimpan…' : 'Tautkan'}
        </Button>
        <Button tone="halus" onClick={onClose} disabled={working}>
          Batal
        </Button>
      </div>
    </form>
  )
}

/**
 * Paginasi "muat halaman berikutnya", bukan nomor halaman.
 *
 * `10-API-STRATEGY.md` §4 menetapkan total baris tidak dikembalikan pada data besar karena
 * `COUNT(*)` atas puluhan juta baris mahal. Di sini jumlahnya memang dikembalikan —
 * tabelnya masih kecil — tetapi bentuk kendalinya sudah disiapkan untuk saat jumlah itu
 * kelak dilepas.
 */
function Pagination({
  offset,
  limit,
  total,
  visible,
  onMove,
  loading,
}: {
  offset: number
  limit: number
  total: number
  visible: number
  onMove: (offset: number) => void
  loading: boolean
}) {
  const first = visible === 0 ? 0 : offset + 1
  const last = offset + visible

  return (
    <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
      <p className="text-sm text-slate-600" role="status">
        Menampilkan {first}–{last} dari {total} laporan.
      </p>
      <div className="flex gap-2">
        <Button
          tone="kedua"
          onClick={() => onMove(Math.max(0, offset - limit))}
          disabled={offset === 0 || loading}
        >
          Sebelumnya
        </Button>
        <Button
          tone="kedua"
          onClick={() => onMove(offset + limit)}
          disabled={last >= total || loading}
        >
          Berikutnya
        </Button>
      </div>
    </div>
  )
}

/**
 * Tanggal `YYYY-MM-DD` ditampilkan apa adanya, bukan diubah menjadi teks lokal.
 *
 * Mengubahnya di sini berarti pemformatan tanggal hidup di dua tempat — di sini dan di
 * `shared/lib` yang `08-TECHNICAL-STRATEGY.md` §3 tetapkan sebagai satu-satunya tempatnya.
 * Berkas itu belum ada; sampai ia ada, bentuk yang datang dari server dipakai apa adanya
 * daripada menambah tafsir ketiga.
 */
function readableDate(date: string): string {
  return date === '' ? '—' : date
}

/** Gagal memuat selalu bernada gangguan: pengguna baru membuka layarnya. */
function LoadErrorMessage({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Daftar laporan belum dapat dimuat. Periksa koneksi lalu tekan Muat ulang."
        tone="gangguan"
      />
    )
  }
  return (
    <ErrorMessage
      title="Daftar laporan gagal dimuat"
      description={
        error instanceof APIError ? error.message : 'Terjadi kesalahan pada sistem. Coba muat ulang.'
      }
      tone="gangguan"
    />
  )
}

/**
 * Gagal pada aksi tahap dibedakan dari gagal memuat.
 *
 * Konflik — sudah ditransfer, sudah diregistrasi — adalah penolakan yang dapat
 * ditindaklanjuti: baris itu sudah dikerjakan orang lain, dan yang perlu dilakukan adalah
 * memuat ulang. Gangguan sistem tidak dapat ditolong dengan mencoba berkali-kali.
 */
function ActionErrorMessage({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Aksi belum tersimpan. Periksa koneksi lalu coba lagi."
        tone="gangguan"
      />
    )
  }
  if (error instanceof APIError && error.status === 409) {
    return (
      <ErrorMessage
        title="Laporan sudah berubah"
        description={`${error.message} Muat ulang daftarnya untuk melihat keadaan terbaru.`}
        tone="penolakan"
      />
    )
  }
  return (
    <ErrorMessage
      title="Aksi gagal"
      description={
        error instanceof APIError ? error.message : 'Terjadi kesalahan pada sistem. Coba lagi.'
      }
      tone="gangguan"
    />
  )
}
