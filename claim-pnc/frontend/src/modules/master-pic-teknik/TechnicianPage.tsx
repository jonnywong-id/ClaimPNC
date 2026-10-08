import { useState } from 'react'

import type { Technician } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { refreshLoadMessage } from '@/components/masterpage/loadMessage'
import {
  MasterDataLayout,
  MessageBox,
  RefreshAddActions,
  editColumn,
} from '@/components/masterpage/MasterPage'
import { DataTable, type Column } from '@/components/DataTable'

import { useTechnicianList } from './api'
import { TechnicianForm } from './TechnicianForm'

/**
 * Layar Master PIC Teknik.
 *
 * Menggantikan harness `UserTeknisInbox` beserta dua section-nya: `BrowseUserTeknis`
 * (grid dan form) dan `ListUserTeknis` (kerangkanya). Butir menunya `MENU_ID 13` pada
 * POOLDATA.M_MENU_APLIKASI_PNC, dengan MENU_PROGRAM `UserTeknisInbox`.
 *
 * # Apa yang dikelola di sini
 *
 * Petugas teknik yang menangani klaim — siapa mereka, di grup mana, siapa atasannya, dan
 * berapa kuota pekerjaan yang boleh dipikulnya. Isi master inilah yang dipakai penugasan
 * klaim untuk memilih petugas berikutnya.
 *
 * # Yang ditiru dari layar lama
 *
 * | Hal | Di Pega | Di sini |
 * |---|---|---|
 * | Tombol | Tambah dan Refresh saja | sama — **tidak ada Hapus** |
 * | Isi daftar | hanya `STS_AKTIF = '1'` | sama |
 * | Nama petugas | dicari ke direktori | sama |
 * | Beban kerja | kolom TOTAL_JOB, hanya dibaca | sama |
 *
 * Tidak adanya Hapus bukan kelalaian: menghapus satu petugas membuat setiap klaim yang
 * menyimpan ID-nya kehilangan rujukan penugasan. Petugas yang berhenti dinonaktifkan.
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Judul kolom | nama kolom mentah (`MCL_NAME`, `COUNTER_QUOTA`) | nama yang dibaca manusia (`D-19`) |
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Pencarian | filter per kolom | satu kotak cari yang menelusuri seluruh kolom |
 * | Pencarian nama | berjalan diam-diam | punya tombol, dan hasilnya terlihat sebelum Simpan |
 * | Email kosong | diterima | ditolak |
 * | Entitas | disimpulkan dari nama server | dipilih pengguna dan disebut di layar |
 */
export function TechnicianPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const list = useTechnicianList()

  const [beingEdited, setBeingEdited] = useState<Technician | null>(null)
  const [formOpen, setFormOpen] = useState(false)

  function openAdd() {
    setBeingEdited(null)
    setFormOpen(true)
  }

  function openEdit(technician: Technician) {
    setBeingEdited(technician)
    setFormOpen(true)
  }

  function closeForm() {
    setFormOpen(false)
    setBeingEdited(null)
  }

  const rows = list.data?.pic_teknik ?? []

  // Kolom "Kuota Sistem Lain" hanya ditampilkan bila ADA yang mengisinya.
  //
  // Nilainya diisi DB link ke sistem luar yang tidak dibawa ke sini (`ADR-0008`), sehingga
  // pada banyak entitas ia nol seluruhnya — dan kolom yang selamanya berisi nol hanya
  // menambah lebar tabel tanpa memberi tahu apa pun. Ia muncul dengan sendirinya bila
  // ternyata terisi, tanpa perubahan kode.
  const anyExternalQuota = rows.some((t) => t.kuota_luar > 0)

  const columns: Column<Technician>[] = [
    {
      key: 'id_operator',
      title: 'ID Operator',
      width: '11rem',
      value: (t) => t.id_operator,
      render: (t) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs font-medium text-slate-700 ring-1 ring-slate-200">
          {t.id_operator}
        </span>
      ),
    },
    {
      key: 'nama',
      title: 'Nama',
      // Nama DAN surel dicari bersamaan: petugas sering dicari lewat surelnya, dan
      // menyembunyikan surel dari penyaring membuat pencarian itu gagal tanpa sebab yang
      // terlihat.
      value: (t) => `${t.nama} ${t.email}`,
      render: (t) => (
        <div className="min-w-0">
          <p className="truncate font-medium text-slate-900">{t.nama}</p>
          <p className="truncate text-xs text-slate-500">{t.email}</p>
        </div>
      ),
    },
    {
      key: 'grup',
      title: 'Grup',
      width: '12rem',
      value: (t) => `${t.grup} ${t.lini_bisnis}`,
      render: (t) => (
        <div className="min-w-0">
          <p className="truncate text-slate-800">{t.grup || <span className="text-slate-400">—</span>}</p>
          {t.lini_bisnis !== '' && (
            <p className="truncate text-xs text-slate-500">{t.lini_bisnis}</p>
          )}
        </div>
      ),
    },
    {
      key: 'atasan',
      title: 'Atasan',
      width: '10rem',
      value: (t) => t.atasan,
      render: (t) =>
        t.atasan ? (
          <span className="font-mono text-xs text-slate-600">{t.atasan}</span>
        ) : (
          <span className="text-slate-400" title="Petugas ini tidak punya atasan di master">
            —
          </span>
        ),
    },
    {
      key: 'beban',
      title: 'Beban / Kuota',
      width: '9rem',
      // Diurutkan menurut BEBAN, bukan teks gabungannya: yang ingin diketahui pengguna
      // adalah siapa yang paling penuh.
      value: (t) => String(t.beban_kerja).padStart(6, '0'),
      render: (t) => <WorkloadCell workload={t.beban_kerja} quota={t.kuota} />,
    },
    ...(anyExternalQuota
      ? [
          {
            key: 'kuota_luar',
            title: 'Kuota sistem lain',
            width: '9rem',
            value: (t: Technician) => String(t.kuota_luar).padStart(6, '0'),
            render: (t: Technician) =>
              t.kuota_luar > 0 ? (
                <span className="text-slate-700">{t.kuota_luar}</span>
              ) : (
                <span className="text-slate-400">—</span>
              ),
          },
        ]
      : []),
    editColumn<Technician>({
      onEdit: openEdit,
      width: '7rem',
      tone: 'halus',
      ariaLabel: (t) => `Ubah PIC teknik ${t.nama || t.id_operator}`,
      withIcon: true,
    }),
  ]

  return (
    <MasterDataLayout
      crumb="PIC Teknik"
      title="Master PIC Teknik"
      description={
        <>
          Petugas teknik yang menangani klaim, beserta grup, atasan, dan kuota
          pekerjaannya. Daftar inilah yang dipakai penugasan klaim untuk memilih petugas
          berikutnya.
        </>
      }
      portalLabel={list.data?.portal ?? portal}
      form={
        formOpen && (
          <TechnicianForm technician={beingEdited} onClose={closeForm} />
        )
      }
      portal={portal}
    >
      <DataTable
        columns={columns}
        rows={rows}
        rowKey={(t) => t.id_operator}
        title="Daftar PIC Teknik"
        description={
          list.data
            ? `${list.data.total} petugas aktif pada entitas ini. Petugas nonaktif tidak ditampilkan, sama seperti di sistem lama.`
            : 'Memuat daftar PIC teknik…'
        }
        searchLabel="Cari ID operator, nama, surel, atau grup"
        emptyMessage="Belum ada PIC teknik aktif pada entitas ini."
        isLoading={list.isPending}
        // Gagal memuat dibedakan dari gagal menyimpan. Yang di sini selalu bernada
        // gangguan: pengguna belum melakukan apa pun yang dapat salah — ia baru membuka
        // layarnya. Kecuali soal portal, yang justru dapat ia perbaiki sendiri.
        error={
          list.isError ? (
            <MessageBox
              message={refreshLoadMessage(list.error, {
                subject: 'Daftar PIC teknik',
                failedTitle: 'Daftar PIC teknik gagal dimuat',
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

/**
 * Beban dan kuota ditampilkan berdampingan karena keduanya hanya bermakna bersama:
 * "9 pekerjaan" tidak memberi tahu apa pun tanpa mengetahui batasnya.
 *
 * Keadaan penuh ditandai DUA cara — warna dan teks "penuh" — bukan warna saja. Sekitar
 * satu dari dua belas laki-laki mengalami buta warna merah-hijau, dan bagi mereka angka
 * berwarna kuning tidak berbeda dari angka biasa.
 */
function WorkloadCell({ workload, quota }: Readonly<{ workload: number; quota: number }>) {
  const full = quota > 0 && workload >= quota

  return (
    <span className="inline-flex items-baseline gap-1 whitespace-nowrap">
      <span className={full ? 'font-semibold text-amber-700' : 'font-medium text-slate-900'}>
        {workload}
      </span>
      <span className="text-slate-400">/</span>
      <span className="text-slate-600">{quota}</span>
      {full && (
        <span className="ml-1 rounded bg-amber-50 px-1.5 py-0.5 text-[0.65rem] font-medium text-amber-700 ring-1 ring-amber-100">
          penuh
        </span>
      )}
    </span>
  )
}

