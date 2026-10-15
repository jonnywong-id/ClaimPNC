import { useEffect, useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import type { AutoClaimBatch } from '@/api/types'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { DownloadIcon } from '@/components/Icon'

import { useAutoClaimBatchList, useExportAutoClaim } from './api'
import { BatchDetail } from './BatchDetail'
import { loadMessage } from './messages'

type Props = {
  source: string
  company: string
  companyName: string
}

/**
 * Daftar batch SATU perusahaan — panel kanan pada CompanyBrowser.
 *
 * # Muat satu layar (permintaan Work Owner 2026-09-29)
 *
 * Grid 9 kolom layar lama (`Section/Inbox_AS_KREDIT_Sect-Section.xml`) ditambah tiga
 * tombol tidak muat di samping daftar perusahaan, dan kolom User Upload terpotong. Tiga
 * penyesuaian, tanpa membuang satu angka pun:
 *
 *   - KODE dan Nama Perusahaan DIBUANG dari grid: keduanya sudah tertulis di kepala panel,
 *     dan setiap baris grid ini milik perusahaan yang sama. Peringatan "tidak terdaftar di
 *     Master Auto Claim" pindah menjadi satu pita di atas grid.
 *   - Di Upload dan Diproses DIGABUNG menjadi "diproses / diunggah" — keduanya dibaca
 *     berpasangan, dan selisihnya disebut sebagai "+n menunggu".
 *   - Tombol dipadatkan: Detail, lalu dua tombol unduh berlabel pendek. Nama
 *     aksesibilitasnya tetap lengkap ("Export berhasil batch …").
 *
 * Jarak sel memakai mode rapat `DataTable` (`dense`). Di layar sempit grid berubah menjadi
 * kartu — perilaku bawaan `DataTable`.
 *
 * Paginasinya tetap di SERVER.
 */
export function CompanyBatches({ source, company, companyName }: Props) {
  const [page, setPage] = useState(1)
  const [opened, setOpened] = useState<string | null>(null)

  const list = useAutoClaimBatchList(source, company, page)
  const exportFile = useExportAutoClaim()

  const label = companyName === '' ? company : companyName

  const compact = '!px-2.5 !py-1 text-xs'

  const columns: Column<AutoClaimBatch>[] = [
    { key: 'batch', title: 'Batch', width: '4.5rem', value: (row) => row.batch },
    {
      // Bagian KUNCI PENGELOMPOKAN: satu nomor batch yang diunggah pada dua tanggal tampil
      // sebagai dua baris, dan tanpa kolom ini keduanya tampak kembar.
      key: 'tanggal',
      title: 'Tgl Proses',
      width: '6.5rem',
      value: (row) => row.tanggal_proses,
      render: (row) => <span className="tabular-nums">{row.tanggal_proses}</span>,
    },
    {
      key: 'proses',
      title: 'Diproses',
      width: '7rem',
      alignRight: true,
      value: (row) => `${row.jumlah_proses}/${row.jumlah_upload}`,
      render: (row) => (
        <span className="inline-flex flex-col items-end tabular-nums">
          <span>
            {row.jumlah_proses}
            <span className="text-slate-400"> / {row.jumlah_upload}</span>
          </span>
          {row.jumlah_belum_proses > 0 && (
            <span className="text-xs font-normal text-amber-700">
              +{row.jumlah_belum_proses} menunggu
            </span>
          )}
        </span>
      ),
    },
    {
      key: 'berhasil',
      title: 'Berhasil',
      width: '5rem',
      alignRight: true,
      value: (row) => String(row.jumlah_berhasil),
      render: (row) => (
        <span className="tabular-nums font-medium text-emerald-700">{row.jumlah_berhasil}</span>
      ),
    },
    {
      key: 'gagal',
      title: 'Gagal',
      width: '4.5rem',
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
      value: (row) => row.user_upload,
      render: (row) => (
        <span className="block max-w-[9rem] truncate" title={row.user_upload}>
          {row.user_upload || '—'}
        </span>
      ),
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: '17rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <span className="inline-flex flex-wrap justify-end gap-1">
          <Button
            tone="kedua"
            className={compact}
            onClick={() => {
              exportFile.reset()
              setOpened(row.batch)
            }}
            aria-label={`Detail batch ${row.batch} ${row.kode_perusahaan}`}
          >
            Detail
          </Button>
          {/* Proses Klaim kini PER BATCH (permintaan Work Owner 2026-09-29) — layar lama
              menaruhnya sekali per tab dan memproses semua batch sekaligus
              (`CreateCasePNCAgent_*`). Generate DLA memang per baris di layar lama
              (`GenerateDLA_Askredit`). Mesin keduanya belum dibangun, jadi tombolnya
              nonaktif; alasannya ada di `title`, bukan catatan di layar. */}
          <Button
            tone="utama"
            className={compact}
            disabled
            aria-label={`Proses klaim batch ${row.batch} ${row.kode_perusahaan}`}
            title="Proses Klaim batch ini — belum tersedia di aplikasi ini"
          >
            Proses Klaim
          </Button>
          <Button
            tone="kedua"
            className={compact}
            disabled
            aria-label={`Generate DLA batch ${row.batch} ${row.kode_perusahaan}`}
            title="Generate DLA batch ini — belum tersedia di aplikasi ini"
          >
            Generate DLA
          </Button>
          <Button
            tone="kedua"
            className={compact}
            disabled={row.jumlah_berhasil === 0 || exportFile.isPending}
            aria-label={`Export berhasil batch ${row.batch} ${row.kode_perusahaan}`}
            title="Export baris berhasil"
            onClick={() =>
              exportFile.mutate({ source, company, batch: row.batch, hasil: 'berhasil' })
            }
          >
            <DownloadIcon className="h-3.5 w-3.5" />
            Berhasil
          </Button>
          <Button
            tone="kedua"
            className={compact}
            disabled={row.jumlah_gagal === 0 || exportFile.isPending}
            aria-label={`Export gagal batch ${row.batch} ${row.kode_perusahaan}`}
            title="Export baris gagal"
            onClick={() => exportFile.mutate({ source, company, batch: row.batch, hasil: 'gagal' })}
          >
            <DownloadIcon className="h-3.5 w-3.5" />
            Gagal
          </Button>
        </span>
      ),
    },
  ]

  if (list.isError) {
    const message = loadMessage(list.error)
    return (
      <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
    )
  }

  return (
    <div className="flex flex-col gap-3">
      {companyName === '' && (
        // Baris perusahaan seperti ini TIDAK AKAN PERNAH berhasil diproses — pencarian
        // penerima klaim membaca master yang sama — jadi petugas perlu melihatnya.
        <p
          role="note"
          className="rounded-kartu border border-amber-200 bg-amber-50 px-4 py-2.5 text-sm text-amber-900"
        >
          Tidak terdaftar di Master Auto Claim — kode {company} perlu didaftarkan. Batch perusahaan
          ini tidak akan berhasil diproses sebelum itu.
        </p>
      )}

      {exportFile.isError && (
        <ErrorMessage
          title="Berkas gagal diunduh"
          description={
            exportFile.error instanceof APIError
              ? exportFile.error.message
              : 'Periksa koneksi jaringan Anda, lalu coba lagi.'
          }
          tone={exportFile.error instanceof NetworkError ? 'gangguan' : 'penolakan'}
        />
      )}

      <DataTable
        columns={columns}
        rows={list.data?.batch ?? []}
        rowKey={(row) => `${row.kode_perusahaan}-${row.batch}-${row.tanggal_proses}`}
        isLoading={list.isPending}
        // Nama tabelnya menyebut perusahaannya: layar ini memuat daftar perusahaan DAN
        // grid batch, dan pembaca layar perlu membedakan keduanya.
        label={`Batch ${label}`}
        hideSearch
        dense
        emptyMessage="Perusahaan ini belum punya batch klaim."
        pagination={{
          page: list.data?.paginasi.halaman ?? page,
          size: list.data?.paginasi.ukuran ?? 15,
          total: list.data?.paginasi.total ?? 0,
          totalPage: list.data?.paginasi.total_halaman ?? 0,
          onPageChange: setPage,
          isLoading: list.isFetching,
        }}
      />

      {opened !== null && (
        <DetailDialog batch={opened} onClose={() => setOpened(null)}>
          <BatchDetail
            company={company}
            companyName={companyName}
            source={source}
            batch={opened}
            onClose={() => setOpened(null)}
          />
        </DetailDialog>
      )}
    </div>
  )
}

/**
 * Pop-up rincian batch (permintaan Work Owner 2026-09-29).
 *
 * Sebelumnya rincian dibuka DI BAWAH grid, sehingga petugas harus menggulir untuk
 * melihatnya dan grid terdorong keluar layar. Pop-up menjaga grid tetap di tempat dan
 * menutup kembali dengan Escape, tombol "Tutup rincian", atau klik di luar kotaknya —
 * pola yang sama dengan pop-up Upload Data Klaim.
 */
function DetailDialog({
  batch,
  onClose,
  children,
}: {
  batch: string
  onClose: () => void
  children: React.ReactNode
}) {
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={`Rincian batch ${batch}`}
      className="fixed inset-0 z-50 flex items-start justify-center bg-slate-900/40 px-3 py-6 sm:items-center sm:px-6"
      onClick={(event) => {
        // Hanya klik pada latar yang menutup; klik di dalam kotak tidak.
        if (event.target === event.currentTarget) onClose()
      }}
    >
      <div className="max-h-full w-full max-w-5xl overflow-y-auto rounded-kartu bg-white p-4 shadow-terbang sm:p-6">
        {children}
      </div>
    </div>
  )
}
