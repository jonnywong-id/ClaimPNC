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

  /*
    TUJUH kolom, dengan label dan urutan yang SAMA PERSIS dengan grid Pega.

    Urutannya dibaca dari `Section/BrowseUserTeknis-Section.xml`, bukan dikarang:

      header  7733 Atasan · 8036 Status Aktif · 8332 Bisnis · 8761 Kelompok
              9144 Counter Klaim <1M · 9420 Counter Klaim >1M
      isi    10108 .OPERATOR_ID · 10572 .ATASAN · 10947 .STS_AKTIF · 11224 .TYPE_BUSINESS
             11569 .TEAM_GROUP · 12090 .COUNTER_QUOTA · 12390 .OLD_OPERATOR_ID
             13155 Ubah

    Tiga hal yang SENGAJA TIDAK ADA di sini, karena tidak ada pula di Pega:

      - kolom Nama tersendiri. Kolom pertama Pega berisi OPERATOR_ID, berlabel
        "Input Nama" — label yang memang tidak mencerminkan isinya, dan itu dibiarkan
        apa adanya supaya petugas melihat layar yang sama.
      - kolom TOTAL_JOB. Report Definition menyebutnya, grid-nya tidak menampilkannya.
      - gabungan "Beban / Kuota" beserta penanda "penuh". Itu karangan saya yang
        keliru: COUNTER_QUOTA bukan batas melainkan PENCACAH klaim, sehingga
        membandingkannya dengan beban tidak berarti apa-apa.
  */
  const columns: Column<Technician>[] = [
    {
      key: 'id_operator',
      // Label Pega apa adanya. Isinya ID operator, bukan nama.
      title: 'Input Nama',
      width: '14rem',
      // Nama dan surel ikut dicari meski tidak ditampilkan: petugas sering mencari lewat
      // keduanya, dan menyembunyikannya dari penyaring membuat pencarian gagal tanpa
      // sebab yang terlihat.
      value: (t) => `${t.id_operator} ${t.nama} ${t.email}`,
      render: (t) => (
        <span className="font-medium text-slate-900">{t.id_operator}</span>
      ),
    },
    {
      key: 'atasan',
      title: 'Atasan',
      width: '14rem',
      value: (t) => t.atasan,
      render: (t) =>
        t.atasan ? (
          <span className="text-slate-700">{t.atasan}</span>
        ) : (
          <span className="text-slate-300">—</span>
        ),
    },
    {
      key: 'aktif',
      title: 'Status Aktif',
      width: '7rem',
      // Ditampilkan sebagai sandi 1/0, sama dengan layar lama. Sandinya disebut di
      // keterangan tabel supaya tetap terbaca orang yang belum terbiasa.
      value: (t) => (t.aktif ? '1' : '0'),
      render: (t) => (
        <span className={t.aktif ? 'text-slate-900' : 'text-slate-400'}>
          {t.aktif ? '1' : '0'}
        </span>
      ),
    },
    {
      key: 'bisnis',
      title: 'Bisnis',
      width: '9rem',
      value: (t) => t.bisnis,
      render: (t) =>
        t.bisnis ? (
          <span className="text-slate-700">{t.bisnis}</span>
        ) : (
          <span className="text-slate-300">—</span>
        ),
    },
    {
      key: 'kelompok',
      title: 'Kelompok',
      width: '7rem',
      value: (t) => t.kelompok,
      render: (t) =>
        t.kelompok ? (
          <span className="text-slate-700">{t.kelompok}</span>
        ) : (
          <span className="text-slate-300">—</span>
        ),
    },
    {
      key: 'counter_klaim_kurang_1m',
      title: 'Counter Klaim <1M',
      width: '8rem',
      alignRight: true,
      // Diurutkan sebagai ANGKA, bukan teks: tanpa pelapisan nol, "9" akan berada di
      // bawah "1463".
      value: (t) => String(t.counter_klaim_kurang_1m).padStart(8, '0'),
      render: (t) => <span className="text-slate-700">{t.counter_klaim_kurang_1m}</span>,
    },
    {
      key: 'counter_klaim_lebih_1m',
      title: 'Counter Klaim >1M',
      width: '8rem',
      alignRight: true,
      value: (t) => String(t.counter_klaim_lebih_1m).padStart(8, '0'),
      render: (t) => <span className="text-slate-700">{t.counter_klaim_lebih_1m}</span>,
    },
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
              ? `${list.data.total} petugas pada entitas ini. Kolom Status Aktif bernilai 1 untuk yang aktif dan 0 untuk yang tidak.`
              : 'Memuat daftar PIC teknik…'
          }
          searchLabel="Cari ID operator, nama, surel, atasan, bisnis, atau kelompok"
          emptyMessage="Belum ada PIC teknik pada entitas ini."
          isLoading={list.isPending}
          error={list.isError ? <LoadErrorMessage error={list.error} /> : undefined}
          actions={
            <>
              <Button tone="kedua" onClick={() => { list.refetch() }} disabled={list.isFetching}>
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
