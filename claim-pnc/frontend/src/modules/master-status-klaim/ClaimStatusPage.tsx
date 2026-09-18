import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import type { ClaimStatus } from '@/api/types'
import { ReloadIcon, AddIcon, EditIcon } from '@/components/Icon'
import { ErrorMessage } from '@/components/ErrorMessage'
import { DataTable, type Column } from '@/components/DataTable'
import { Button } from '@/components/Button'

import { useClaimStatusList } from './api'
import { ClaimStatusForm } from './ClaimStatusForm'

/**
 * Layar Master Status Klaim.
 *
 * Menggantikan harness `StatusClaimInbox` beserta dua section-nya:
 * `BrowseStatusClaim` (grid) dan `ListStatusClaim` (kerangka dan tombolnya).
 *
 * # Yang ditiru dari layar lama
 *
 * Fungsinya sama persis: daftar bergrid dengan kolom kode dan status, tombol **Tambah**,
 * tombol **Muat ulang**, dan aksi **Ubah** per baris. **Tidak ada Hapus** — layar Pega
 * pun tidak punya, dan `ADR-0012` melarang master dihapus permanen karena klaim lama
 * merujuknya.
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Pencarian | filter per kolom | satu kotak cari yang menelusuri seluruh kolom |
 * | Kolom kode lama | tidak ditampilkan | ditampilkan, karena ia menjelaskan data historis |
 * | Isian kosong | diterima | ditolak (keputusan Work Owner 2026-09-17) |
 * | Nama ganda | diterima | ditolak (idem) |
 */
export function ClaimStatusPage() {
  const list = useClaimStatusList()

  // null = form tertutup; { kode: '' } = sedang menambah; berisi = sedang mengubah.
  const [sedangDisunting, setSedangDisunting] = useState<ClaimStatus | null>(null)
  const [formTerbuka, setFormTerbuka] = useState(false)

  function openAdd() {
    setSedangDisunting(null)
    setFormTerbuka(true)
  }

  function openEdit(status: ClaimStatus) {
    setSedangDisunting(status)
    setFormTerbuka(true)
  }

  function closeForm() {
    setFormTerbuka(false)
    setSedangDisunting(null)
  }

  const columns: Column<ClaimStatus>[] = [
    {
      key: 'kode',
      judul: 'Kode',
      lebar: '8rem',
      nilai: (s) => s.kode,
      tampil: (s) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs font-medium text-slate-700 ring-1 ring-slate-200">
          {s.kode}
        </span>
      ),
    },
    {
      key: 'label',
      judul: 'Status',
      nilai: (s) => s.label,
      tampil: (s) => <span className="font-medium text-slate-900">{s.label}</span>,
    },
    {
      key: 'kode_lama',
      judul: 'Kode lama',
      lebar: '9rem',
      nilai: (s) => s.kode_lama,
      tampil: (s) =>
        s.kode_lama ? (
          <span className="inline-flex items-center rounded-md bg-blue-50 px-2 py-0.5 font-mono text-xs font-medium text-blue-700 ring-1 ring-blue-100">
            {s.kode_lama}
          </span>
        ) : (
          <span className="text-slate-400" title="Kode ini tidak pernah punya penomoran lama">
            —
          </span>
        ),
    },
    {
      key: 'aksi',
      judul: 'Aksi',
      lebar: '7rem',
      tanpaUrut: true,
      keKanan: true,
      nilai: () => '',
      tampil: (s) => (
        <Button tone="halus" onClick={() => openEdit(s)} aria-label={`Ubah status ${s.label}`}>
          <EditIcon className="h-3.5 w-3.5" />
          Ubah
        </Button>
      ),
    },
  ]

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <header className="mb-6">
        <nav aria-label="Jejak lokasi" className="mb-2 text-xs font-medium text-slate-500">
          <ol className="flex items-center gap-1.5">
            <li>Master Data</li>
            <li aria-hidden="true" className="text-slate-300">
              /
            </li>
            <li className="text-slate-700">Status Klaim</li>
          </ol>
        </nav>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
          Master Status Klaim
        </h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">
          Daftar state bisnis sebuah klaim — Register, Claim Committee, Paid, dan
          seterusnya. Nama status yang diubah di sini langsung terbaca layar search
          klaim dan laporan.
        </p>
      </header>

      {formTerbuka && (
        <div className="mb-6">
          <ClaimStatusForm status={sedangDisunting} tutup={closeForm} />
        </div>
      )}

      <DataTable
        columns={columns}
        rows={list.data?.status_klaim ?? []}
        rowKey={(s) => s.kode}
        judul="Daftar Status Klaim"
        keterangan={
          list.data ? `${list.data.total} status terdaftar.` : 'Memuat daftar status…'
        }
        searchLabel="Cari kode atau nama status"
        emptyMessage="Belum ada status klaim yang terdaftar."
        isLoading={list.isPending}
        error={list.isError ? <LoadErrorMessage error={list.error} /> : undefined}
        aksi={
          <>
            <Button
              tone="kedua"
              onClick={() => void list.refetch()}
              disabled={list.isFetching}
            >
              <ReloadIcon className={`h-4 w-4 ${list.isFetching ? 'animate-spin' : ''}`} />
              {list.isFetching ? 'Memuat…' : 'Muat ulang'}
            </Button>
            <Button tone="utama" onClick={openAdd} disabled={formTerbuka && !sedangDisunting}>
              <AddIcon className="h-4 w-4" />
              Tambah
            </Button>
          </>
        }
      />
    </div>
  )
}

/**
 * Gagal memuat dibedakan dari gagal menyimpan.
 *
 * Yang di sini selalu bernada gangguan: pengguna belum melakukan apa pun yang dapat
 * salah — ia baru membuka layarnya.
 */
function LoadErrorMessage({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        judul="Tidak dapat menghubungi server"
        keterangan="Daftar status belum dapat dimuat. Periksa koneksi lalu tekan Muat ulang."
        tone="gangguan"
      />
    )
  }
  return (
    <ErrorMessage
      judul="Daftar status gagal dimuat"
      keterangan={
        error instanceof APIError ? error.message : 'Terjadi kesalahan pada sistem. Coba muat ulang.'
      }
      tone="gangguan"
    />
  )
}
