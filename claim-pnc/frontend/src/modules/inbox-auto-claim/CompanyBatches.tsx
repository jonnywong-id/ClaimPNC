import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import type { AutoClaimBatch } from '@/api/types'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'

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
 * Susunan kolomnya sama dengan grid 9 kolom layar lama
 * (`Section/Inbox_AS_KREDIT_Sect-Section.xml`) — yang berubah hanya tempatnya: panel kanan
 * di samping daftar perusahaan (keputusan Work Owner 2026-09-28, menggantikan baris melar
 * 2026-09-27). KODE dan Nama Perusahaan tetap ada walau perusahaannya sudah tertulis di
 * kepala panel: contoh yang disetujui 2026-09-27 memuatnya, dan baris yang disalin
 * petugas tetap lengkap.
 *
 * Paginasinya tetap di SERVER. Satu perusahaan dapat punya puluhan batch (DIRECT MO di tab
 * Asuransi Kredit memuat 27 pada saat ditulis).
 */
export function CompanyBatches({ source, company, companyName }: Props) {
  const [page, setPage] = useState(1)
  const [opened, setOpened] = useState<string | null>(null)

  const list = useAutoClaimBatchList(source, company, page)
  const exportFile = useExportAutoClaim()

  const label = companyName === '' ? company : companyName

  const columns: Column<AutoClaimBatch>[] = [
    { key: 'kode', title: 'KODE', width: '6rem', value: (row) => row.kode_perusahaan },
    {
      key: 'perusahaan',
      title: 'Nama Perusahaan',
      value: (row) => `${row.nama_perusahaan} ${row.kode_perusahaan}`,
      render: (row) =>
        row.nama_perusahaan === '' ? (
          // Baris seperti ini TIDAK AKAN PERNAH berhasil diproses — pencarian penerima
          // klaim membaca master yang sama — jadi petugas perlu melihatnya, bukan menebak
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
      // Bagian KUNCI PENGELOMPOKAN: satu nomor batch yang diunggah pada dua tanggal tampil
      // sebagai dua baris, dan tanpa kolom ini keduanya tampak kembar.
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
    { key: 'user', title: 'User Upload', width: '9rem', value: (row) => row.user_upload },
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
            onClick={() => {
              exportFile.reset()
              setOpened(row.batch)
            }}
            aria-label={`Detail batch ${row.batch} ${row.kode_perusahaan}`}
          >
            Detail
          </Button>
          <Button
            tone="kedua"
            disabled={row.jumlah_berhasil === 0 || exportFile.isPending}
            aria-label={`Export berhasil batch ${row.batch} ${row.kode_perusahaan}`}
            onClick={() =>
              exportFile.mutate({ source, company, batch: row.batch, hasil: 'berhasil' })
            }
          >
            Export Berhasil
          </Button>
          <Button
            tone="kedua"
            disabled={row.jumlah_gagal === 0 || exportFile.isPending}
            aria-label={`Export gagal batch ${row.batch} ${row.kode_perusahaan}`}
            onClick={() => exportFile.mutate({ source, company, batch: row.batch, hasil: 'gagal' })}
          >
            Export Gagal
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
    <div className="flex flex-col gap-4">
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
        // Nama tabelnya menyebut perusahaannya: layar ini memuat tabel perusahaan DAN
        // tabel batch di dalamnya, dan pembaca layar perlu membedakan keduanya.
        label={`Batch ${label}`}
        hideSearch
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
        <BatchDetail
          company={company}
          companyName={companyName}
          source={source}
          batch={opened}
          onClose={() => setOpened(null)}
        />
      )}
    </div>
  )
}
