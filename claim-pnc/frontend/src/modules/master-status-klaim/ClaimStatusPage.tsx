import { useState } from 'react'

import type { ClaimStatus } from '@/api/types'
import { DataTable, type Column } from '@/components/DataTable'
import { useSelectedPortal } from '@/app/portal'
import { refreshLoadMessage } from '@/components/masterpage/loadMessage'
import {
  MasterDataLayout,
  MessageBox,
  RefreshAddActions,
  editColumn,
} from '@/components/masterpage/MasterPage'

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
 * tombol **Refresh**, dan aksi **Ubah** per baris. **Tidak ada Hapus** — layar Pega
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
  const portal = useSelectedPortal((state) => state.alias)
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
      title: 'Kode',
      width: '8rem',
      value: (s) => s.kode,
      render: (s) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs font-medium text-slate-700 ring-1 ring-slate-200">
          {s.kode}
        </span>
      ),
    },
    {
      key: 'label',
      title: 'Status',
      value: (s) => s.label,
      render: (s) => <span className="font-medium text-slate-900">{s.label}</span>,
    },
    {
      key: 'kode_lama',
      title: 'Kode lama',
      width: '9rem',
      value: (s) => s.kode_lama,
      render: (s) =>
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
    editColumn<ClaimStatus>({
      onEdit: openEdit,
      width: '7rem',
      tone: 'halus',
      ariaLabel: (s) => `Ubah status ${s.label}`,
      withIcon: true,
    }),
  ]

  return (
    <MasterDataLayout
      crumb="Status Klaim"
      title="Master Status Klaim"
      description={
        <>
          Daftar keadaan bisnis sebuah klaim — Register, Claim Committee, Paid, dan
          seterusnya. Nama status yang diubah di sini langsung terbaca layar search
          klaim dan laporan.
        </>
      }
      portalLabel={list.data?.portal ?? portal}
      form={
        formTerbuka && (
          <ClaimStatusForm status={sedangDisunting} tutup={closeForm} />
        )
      }
      portal={portal}
    >
      <DataTable
        columns={columns}
        rows={list.data?.status_klaim ?? []}
        rowKey={(s) => s.kode}
        title="Daftar Status Klaim"
        description={
          list.data
            ? `${list.data.total} status terdaftar pada entitas ini.`
            : 'Memuat daftar status…'
        }
        searchLabel="Cari kode atau nama status"
        emptyMessage="Belum ada status klaim yang terdaftar."
        isLoading={list.isPending}
        // Gagal memuat dibedakan dari gagal menyimpan. Yang di sini selalu bernada gangguan:
        // pengguna belum melakukan apa pun yang dapat salah — ia baru membuka layarnya.
        error={
          list.isError ? (
            <MessageBox
              message={refreshLoadMessage(list.error, {
                subject: 'Daftar status',
                failedTitle: 'Daftar status gagal dimuat',
              })}
            />
          ) : undefined
        }
        actions={
          <RefreshAddActions
            query={list}
            onAdd={openAdd}
            addDisabled={formTerbuka && !sedangDisunting}
          />
        }
      />
    </MasterDataLayout>
  )
}

