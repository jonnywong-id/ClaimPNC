import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import type { ArchiveFile, BranchScope, PageInfo } from './types'

type Props = {
  files: ArchiveFile[]
  page: PageInfo | undefined
  currentPage: number
  onPageChange: (page: number) => void

  scope: BranchScope | undefined

  isLoading: boolean
  error: string

  /** Dipanggil saat satu berkas dikirim ke sistem Arsip. */
  onSend: (id: number) => void

  /** ID berkas yang sedang dikirim; tombolnya dinonaktifkan selama itu. */
  sendingID: number | null

  /** Pesan galat pengiriman terakhir, sudah siap dibaca pengguna. */
  sendError: string

  /** Pesan berhasil pengiriman terakhir. */
  sendMessage: string
}

/** Nama lini bisnis yang dikenali, untuk menerangkan cakupan kepada pengguna. */
const GROUP_PANEL_NAMES: Record<string, string> = {
  '002': 'Personal Accident',
  '003': 'Aneka',
  '004': 'Marine Cargo',
  '005': 'Travel',
  '006': 'Fire / Property',
  '009': 'Aneka',
}

/**
 * BranchQueue adalah bagian "Kirim ke Cabang".
 *
 * Isinya berkas yang belum pernah dikirim ke sistem Arsip — `CABANGSTATUS = '0'` pada
 * `Activity/GetDataArchiveCabangKlaim-Act.xml`.
 *
 * # Kenapa cakupan lini bisnis DIUMUMKAN, bukan dibiarkan senyap
 *
 * Daftar ini disaring menurut jabatan pemanggil, dan saringannya TAMPAK TERBALIK: petugas
 * berjabatan PA tidak melihat berkas Personal Accident, dan petugas Travel tidak melihat
 * berkas Travel. Itu perilaku sistem lama yang direplikasi atas keputusan Work Owner
 * 2026-09-24.
 *
 * Tanpa keterangan di layar, berkas yang hilang dari daftar akan dilaporkan berulang kali
 * sebagai kerusakan modul — dan tiap laporannya menghabiskan waktu orang untuk sampai ke
 * jawaban yang sama.
 */
export function BranchQueue({
  files,
  page,
  currentPage,
  onPageChange,
  scope,
  isLoading,
  error,
  onSend,
  sendingID,
  sendError,
  sendMessage,
}: Props) {
  const hidden = scope?.lini_disembunyikan ?? []

  const columns: Column<ArchiveFile>[] = [
    { key: 'id', title: 'ID', value: (row) => String(row.id), width: '5rem' },
    { key: 'nomor_klaim', title: 'No Klaim', value: (row) => row.nomor_klaim },
    { key: 'nomor_polis', title: 'No Polis', value: (row) => row.nomor_polis },
    { key: 'nama_tertanggung', title: 'Tertanggung', value: (row) => row.nama_tertanggung },
    { key: 'nama_box', title: 'Nama BOX', value: (row) => row.nama_box },
    { key: 'kode_filling', title: 'Kode Filling', value: (row) => row.kode_filling },
    {
      key: 'jumlah_lembar',
      title: 'Jumlah Lembar',
      value: (row) => String(row.jumlah_lembar),
      alignRight: true,
    },
    { key: 'tanggal_input', title: 'TGL INPUT', value: (row) => dateText(row.tanggal_input) },
    {
      key: 'group_panel',
      title: 'Lini',
      value: (row) => GROUP_PANEL_NAMES[row.group_panel] ?? row.group_panel ?? '—',
    },
    {
      key: 'aksi',
      title: '',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button
          type="button"
          tone="utama"
          onClick={() => onSend(row.id)}
          disabled={sendingID !== null}
        >
          {sendingID === row.id ? 'Mengirim…' : 'Kirim'}
        </Button>
      ),
    },
  ]

  return (
    <section>
      {hidden.length > 0 && (
        <p className="rounded-kontrol border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-700">
          Daftar ini disaring menurut jabatan Anda: lini{' '}
          <span className="font-medium">
            {hidden.map((code) => GROUP_PANEL_NAMES[code] ?? code).join(' dan ')}
          </span>{' '}
          tidak ditampilkan. Penyaringan ini mengikuti sistem lama apa adanya.
        </p>
      )}

      {sendMessage && (
        <p
          className="mt-3 rounded-kontrol border border-emerald-200 bg-emerald-50 px-4 py-2.5 text-sm text-emerald-900"
          role="status"
        >
          {sendMessage}
        </p>
      )}

      {sendError && (
        <div className="mt-3">
          <ErrorMessage
            title="Berkas tidak terkirim"
            description={sendError}
            tone="gangguan"
          />
        </div>
      )}

      <div className="mt-4">
        <DataTable
          columns={columns}
          rows={files}
          rowKey={(row) => String(row.id)}
          title="Kirim ke Cabang"
          description="Berkas yang belum pernah dikirim ke sistem Arsip."
          label="Daftar berkas arsip yang menunggu dikirim"
          isLoading={isLoading}
          error={error}
          emptyMessage="Tidak ada berkas yang menunggu dikirim."
          hideSearch
          // Nilai cadangan dipakai saat halaman pertama masih dimuat; lihat catatan yang
          // sama di ArchiveDocumentPage.
          pagination={{
            page: currentPage,
            size: page?.ukuran ?? 20,
            total: page?.total ?? 0,
            totalPage: page?.total_halaman ?? 1,
            onPageChange,
            isLoading,
          }}
        />
      </div>
    </section>
  )
}

function dateText(iso: string | null): string {
  return iso ? formatDate(iso) : '—'
}
