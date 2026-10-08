import { useState } from 'react'

import type { DominantFactor } from '@/api/types'
import { DataTable, type Column } from '@/components/DataTable'
import { useSelectedPortal } from '@/app/portal'
import { refreshLoadMessage } from '@/components/masterpage/loadMessage'
import {
  MasterDataLayout,
  MessageBox,
  RefreshAddActions,
  editColumn,
} from '@/components/masterpage/MasterPage'

import { useDominantFactorList } from './api'
import { DominantFactorForm } from './DominantFactorForm'

/**
 * Layar Master Dominan Factor.
 *
 * Menggantikan harness `DetailDominanFactor` beserta section `DetailDominanFactor_Sec`.
 *
 * # Yang ditiru dari layar lama
 *
 * Fungsinya sama persis: daftar bergrid dengan kolom **ID** dan **Keterangan**, tombol
 * **Tambah**, tombol **Refresh**, dan aksi **Ubah** per baris. **Tidak ada Hapus** —
 * layar Pega pun tidak punya, `Database/PEGA_M_DOMINAN_FACTOR.prc` tidak punya cabangnya,
 * dan `ADR-0012` melarang master dihapus permanen karena klaim lama merujuknya.
 *
 * Judul form pun diambil apa adanya dari `pyTitle` layar lama: "Menambah Data" dan
 * "Memperbaharui Data" (`D-13`).
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Urutan baris | tanpa ORDER BY — ditentukan basis data | numerik menurut ID, tetap |
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Pencarian | tidak ada | satu kotak cari yang menelusuri ID dan keterangan |
 * | Keterangan kosong | diterima | **tetap diterima** (Work Owner 2026-09-20) |
 * | Keterangan ganda | diterima | **tetap diterima** (idem), dengan peringatan di form |
 *
 * Dua baris terakhir membedakan modul ini dari Master Status Klaim dan Master Tipe
 * Surveyors, yang justru menolak keduanya. Perbedaannya disengaja dan dicatat supaya
 * tidak "diseragamkan" tanpa keputusan baru.
 */
export function DominantFactorPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const list = useDominantFactorList()

  // null = sedang menambah; berisi = sedang mengubah baris itu.
  const [sedangDisunting, setSedangDisunting] = useState<DominantFactor | null>(null)
  const [formTerbuka, setFormTerbuka] = useState(false)

  function openAdd() {
    setSedangDisunting(null)
    setFormTerbuka(true)
  }

  function openEdit(factor: DominantFactor) {
    setSedangDisunting(factor)
    setFormTerbuka(true)
  }

  function closeForm() {
    setFormTerbuka(false)
    setSedangDisunting(null)
  }

  const columns: Column<DominantFactor>[] = [
    {
      key: 'id',
      title: 'ID',
      width: '7rem',
      value: (f) => f.id,
      render: (f) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs font-medium text-slate-700 ring-1 ring-slate-200">
          {f.id}
        </span>
      ),
    },
    {
      key: 'nama',
      title: 'Keterangan',
      value: (f) => f.nama,
      render: (f) =>
        // Keterangan kosong DITERIMA modul ini, sehingga ia pasti akan muncul di daftar.
        // Ditandai, bukan dibiarkan sebagai sel kosong yang terlihat seperti tabel rusak.
        f.nama ? (
          <span className="font-medium text-slate-900">{f.nama}</span>
        ) : (
          <span className="italic text-slate-400" title="Baris ini tidak punya keterangan">
            (tanpa keterangan)
          </span>
        ),
    },
    editColumn<DominantFactor>({
      onEdit: openEdit,
      width: '7rem',
      tone: 'halus',
      ariaLabel: (f) => `Ubah faktor dominan ${f.nama || f.id}`,
      withIcon: true,
    }),
  ]

  return (
    <MasterDataLayout
      crumb="Dominan Factor"
      title="Master Dominan Factor"
      description={
        'Daftar acuan faktor dominan penyebab kerugian sebuah klaim. Keterangan yang ' +
        'diubah di sini ikut terbaca laporan Outstanding per Cabang, yang merangkai ' +
        'seluruh faktor satu klaim menjadi satu baris.'
      }
      portalLabel={list.data?.portal ?? portal}
      form={
        formTerbuka && (
          <DominantFactorForm factor={sedangDisunting} tutup={closeForm} />
        )
      }
      portal={portal}
    >
      <DataTable
        columns={columns}
        rows={list.data?.dominan_factor ?? []}
        rowKey={(f) => f.id}
        title="Daftar Faktor Dominan"
        description={
          list.data
            ? `${list.data.total} faktor dominan terdaftar pada entitas ini.`
            : 'Memuat daftar faktor dominan…'
        }
        searchLabel="Cari ID atau keterangan"
        emptyMessage="Belum ada faktor dominan yang terdaftar."
        isLoading={list.isPending}
        // Gagal memuat dibedakan dari gagal menyimpan. Yang di sini selalu bernada
        // gangguan: pengguna belum melakukan apa pun yang dapat salah — ia baru membuka
        // layarnya.
        error={
          list.isError ? (
            <MessageBox
              message={refreshLoadMessage(list.error, {
                subject: 'Daftar faktor dominan',
                failedTitle: 'Daftar faktor dominan gagal dimuat',
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

