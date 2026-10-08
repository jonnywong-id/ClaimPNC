import { useState } from 'react'

import type { SurveyorType } from '@/api/types'
import { DataTable, type Column } from '@/components/DataTable'
import { useSelectedPortal } from '@/app/portal'
import { refreshLoadMessage } from '@/components/masterpage/loadMessage'
import {
  MasterDataLayout,
  MessageBox,
  RefreshAddActions,
  editColumn,
} from '@/components/masterpage/MasterPage'

import { useSurveyorTypeList } from './api'
import { SurveyorTypeForm } from './SurveyorTypeForm'

/**
 * Layar Master Tipe Surveyors.
 *
 * Menggantikan harness `SurveyorsInbox` beserta dua section-nya: `BrowseSuveryors` (grid)
 * dan `GridSurveyors` (kerangka dan tombolnya). Butir menunya `MENU_ID 14` pada
 * POOLDATA.M_MENU_APLIKASI_PNC.
 *
 * # Apa yang dikelola di sini
 *
 * GOLONGAN petugas survei — Internal Surveyor, Loss Adjuster, Expert, Survey Agent —
 * bukan daftar orangnya. Daftar orang ada satu tingkat di bawah, di butir menu "Master
 * Surveyors" (`MENU_ID 15`, harness `DetailSurveyorsInbox`), dan belum dibangun.
 *
 * # Yang ditiru dari layar lama
 *
 * Fungsinya sama persis: daftar bergrid dengan kolom kode dan deskripsi, tombol
 * **Tambah**, tombol **Refresh**, dan aksi **Ubah** per baris. **Tidak ada Hapus** —
 * layar Pega pun tidak punya, dan menghapus satu tipe akan membuat puluhan baris
 * D_SURVEYORS kehilangan golongannya.
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Judul kolom | nama kolom mentah (`M_SURVEY_ID`, `DESCRIPTION`) | nama yang dibaca manusia (`D-19`) |
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Pencarian | filter per kolom | satu kotak cari yang menelusuri seluruh kolom |
 * | Isian kosong | diterima | ditolak |
 * | Nama ganda | diterima | ditolak |
 * | Entitas | disimpulkan dari nama server | dipilih pengguna dan disebut di layar |
 */
export function SurveyorTypePage() {
  const portal = useSelectedPortal((state) => state.alias)
  const list = useSurveyorTypeList()

  // null = form tertutup; { kode: '' } = sedang menambah; berisi = sedang mengubah.
  const [beingEdited, setBeingEdited] = useState<SurveyorType | null>(null)
  const [formOpen, setFormOpen] = useState(false)

  function openAdd() {
    setBeingEdited(null)
    setFormOpen(true)
  }

  function openEdit(surveyorType: SurveyorType) {
    setBeingEdited(surveyorType)
    setFormOpen(true)
  }

  function closeForm() {
    setFormOpen(false)
    setBeingEdited(null)
  }

  const rows = list.data?.tipe_surveyor ?? []

  // Kolom kode lama hanya ditampilkan bila ADA yang mengisinya.
  //
  // Pada portal ASM seluruh empat barisnya kosong, dan kolom yang selamanya berisi tanda
  // hubung hanya menambah lebar tabel tanpa memberi tahu apa pun. Ia tetap disiapkan
  // karena basis data entitas lain belum diperiksa — bila ternyata terisi di sana,
  // kolomnya muncul dengan sendirinya tanpa perubahan kode.
  const anyLegacyCode = rows.some((t) => t.kode_lama.trim() !== '')

  const columns: Column<SurveyorType>[] = [
    {
      key: 'kode',
      title: 'Kode',
      width: '8rem',
      value: (t) => t.kode,
      render: (t) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs font-medium text-slate-700 ring-1 ring-slate-200">
          {t.kode}
        </span>
      ),
    },
    {
      key: 'deskripsi',
      title: 'Tipe Surveyor',
      value: (t) => t.deskripsi,
      render: (t) => <span className="font-medium text-slate-900">{t.deskripsi}</span>,
    },
    ...(anyLegacyCode
      ? [
          {
            key: 'kode_lama',
            title: 'Kode lama',
            width: '9rem',
            value: (t: SurveyorType) => t.kode_lama,
            render: (t: SurveyorType) =>
              t.kode_lama ? (
                <span className="inline-flex items-center rounded-md bg-blue-50 px-2 py-0.5 font-mono text-xs font-medium text-blue-700 ring-1 ring-blue-100">
                  {t.kode_lama}
                </span>
              ) : (
                <span className="text-slate-400" title="Tipe ini tidak punya penomoran lama">
                  —
                </span>
              ),
          },
        ]
      : []),
    editColumn<SurveyorType>({
      onEdit: openEdit,
      width: '7rem',
      tone: 'halus',
      ariaLabel: (t) => `Ubah tipe surveyor ${t.deskripsi}`,
      withIcon: true,
    }),
  ]

  return (
    <MasterDataLayout
      crumb="Tipe Surveyors"
      title="Master Tipe Surveyors"
      description={
        <>
          Golongan petugas survei — Internal Surveyor, Loss Adjuster, Expert, dan
          seterusnya. Setiap surveyor digolongkan ke salah satu tipe di sini, dan nama
          yang diubah langsung terbaca layar pemilihan surveyor.
        </>
      }
      portalLabel={list.data?.portal ?? portal}
      form={formOpen && <SurveyorTypeForm surveyorType={beingEdited} onClose={closeForm} />}
      portal={portal}
    >
      <DataTable
        columns={columns}
        rows={rows}
        rowKey={(t) => t.kode}
        title="Daftar Tipe Surveyor"
        description={
          list.data
            ? `${list.data.total} tipe surveyor terdaftar pada entitas ini.`
            : 'Memuat daftar tipe surveyor…'
        }
        searchLabel="Cari kode atau nama tipe surveyor"
        emptyMessage="Belum ada tipe surveyor pada entitas ini."
        isLoading={list.isPending}
        // Gagal memuat dibedakan dari gagal menyimpan. Yang di sini selalu bernada
        // gangguan: pengguna belum melakukan apa pun yang dapat salah — ia baru membuka
        // layarnya. Kecuali soal portal, yang justru dapat ia perbaiki sendiri.
        error={
          list.isError ? (
            <MessageBox
              message={refreshLoadMessage(list.error, {
                subject: 'Daftar tipe surveyor',
                failedTitle: 'Daftar tipe surveyor gagal dimuat',
              })}
            />
          ) : undefined
        }
        actions={
          <RefreshAddActions
            query={list}
            onAdd={openAdd}
            addDisabled={formOpen && !beingEdited}
          />
        }
      />
    </MasterDataLayout>
  )
}
