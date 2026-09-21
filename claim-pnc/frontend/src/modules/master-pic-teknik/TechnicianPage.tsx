import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type Technician } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { AddIcon, EditIcon, ReloadIcon } from '@/components/Icon'

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
    {
      key: 'aksi',
      title: 'Aksi',
      width: '7rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (t) => (
        <Button
          tone="halus"
          onClick={() => openEdit(t)}
          aria-label={`Ubah PIC teknik ${t.nama || t.id_operator}`}
        >
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
            <li className="text-slate-700">PIC Teknik</li>
          </ol>
        </nav>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">Master PIC Teknik</h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">
          Petugas teknik yang menangani klaim, beserta grup, atasan, dan kuota
          pekerjaannya. Daftar inilah yang dipakai penugasan klaim untuk memilih petugas
          berikutnya.
        </p>

        {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani
            empat badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh
            hanya diandaikan pengguna (ADR-0030, R-20). */}
        <p className="mt-3 text-xs text-slate-500">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
        </p>
      </header>

      {formOpen && (
        <div className="mb-6">
          <TechnicianForm technician={beingEdited} onClose={closeForm} />
        </div>
      )}

      {portal === null ? (
        <ErrorMessage
          title="Portal entitas belum dipilih"
          description="Data master dimiliki masing-masing entitas. Pilih portal entitas di bilah atas halaman ini lebih dulu."
          tone="penolakan"
        />
      ) : (
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
          error={list.isError ? <LoadErrorMessage error={list.error} /> : undefined}
          actions={
            <>
              <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
                <ReloadIcon className={`h-4 w-4 ${list.isFetching ? 'animate-spin' : ''}`} />
                {list.isFetching ? 'Memuat…' : 'Refresh'}
              </Button>
              <Button tone="utama" onClick={openAdd} disabled={formOpen && !beingEdited}>
                <AddIcon className="h-4 w-4" />
                Tambah
              </Button>
            </>
          }
        />
      )}
    </div>
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
function WorkloadCell({ workload, quota }: { workload: number; quota: number }) {
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

/**
 * Gagal memuat dibedakan dari gagal menyimpan.
 *
 * Yang di sini selalu bernada gangguan: pengguna belum melakukan apa pun yang dapat salah
 * — ia baru membuka layarnya. Kecuali soal portal, yang justru dapat ia perbaiki sendiri.
 */
function LoadErrorMessage({ error }: { error: unknown }) {
  const message = loadMessage(error)
  return <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
}

function loadMessage(error: unknown): { title: string; description: string; tone: ErrorTone } {
  if (error instanceof NetworkError) {
    return {
      title: 'Tidak dapat menghubungi server',
      description: 'Daftar PIC teknik belum dapat dimuat. Periksa koneksi lalu tekan Refresh.',
      tone: 'gangguan',
    }
  }

  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description:
            'Data master dimiliki masing-masing entitas. Pilih portal entitas di bilah atas halaman ini lebih dulu.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Entitasnya sudah direncanakan, tetapi kredensial basis datanya belum diisi. Hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
      default:
        return {
          title: 'Daftar PIC teknik gagal dimuat',
          description: error.message,
          tone: 'gangguan',
        }
    }
  }

  return {
    title: 'Daftar PIC teknik gagal dimuat',
    description: 'Terjadi kesalahan pada sistem. Coba muat ulang.',
    tone: 'gangguan',
  }
}
