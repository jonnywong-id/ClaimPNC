import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import {
  CauseOfLossDetailErrorCode,
  ErrorCode,
  type CauseOfLossDetail,
  type CauseOfLossDetailInput,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import {
  useCauseOfLossDetail,
  useCauseOfLossDetailList,
  useCauseOfLossOptions,
  useCreateCauseOfLossDetail,
  useSaveCauseOfLossDetail,
} from './api'
import { DetailForm, type DetailFormValues } from './DetailForm'

/**
 * Ukuran halaman diambil dari `pyPageSize` pada grid utama
 * `Section/BrowseDetailCauseOfLoss-Section.xml:8070`.
 *
 * **50**, dan itu jauh lebih besar daripada layar master lain di aplikasi ini — 15 pada
 * Master Rekening dan kerabatnya, 20 pada Supplier, Bengkel, serta Panel. Angkanya milik
 * layar, bukan milik komponen tabel, dan dipertahankan apa adanya (`D-13`).
 *
 * Mode paginasinya `Numeric` (`:8068`), sama dengan `DataTable` — sehingga di layar ini
 * tidak ada selisih mode seperti pada layar yang memakai `Next Previous`.
 */
const PAGE_SIZE = 50

type MessageContent = { title: string; description: string; tone: ErrorTone }

/** Mengubah galat pemuatan daftar menjadi pesan yang dapat ditindaklanjuti. */
function loadMessage(error: unknown): MessageContent {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan, lalu muat ulang halaman ini.',
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
            'Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian ' +
            'atas halaman ini lebih dulu.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk ' +
            'melengkapi kredensial basis datanya.',
          tone: 'gangguan',
        }
      default:
        return {
          title: 'Daftar detail penyebab kerugian tidak dapat dimuat',
          description: error.message,
          tone: 'gangguan',
        }
    }
  }
  return {
    title: 'Terjadi kesalahan pada sistem',
    description: 'Coba muat ulang halaman ini. Bila berulang, hubungi administrator Claim PNC.',
    tone: 'gangguan',
  }
}

/** Mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti. */
function saveMessage(error: unknown): MessageContent {
  if (error instanceof APIError) {
    if (error.kode === CauseOfLossDetailErrorCode.idTaken) {
      return {
        title: 'Penomoran ID bentrok',
        description:
          'Isian Anda sudah benar — yang bermasalah adalah nomor yang diterbitkan sistem. ' +
          'Coba simpan sekali lagi; bila tetap ditolak, hubungi administrator Claim PNC.',
        tone: 'gangguan',
      }
    }
    if (error.kode === ErrorCode.notFound) {
      return {
        title: 'Baris ini sudah tidak ada',
        description:
          'Mungkin sudah diubah petugas lain. Tutup form ini dan muat ulang daftarnya.',
        tone: 'penolakan',
      }
    }
    // Portal yang belum dipilih ditangani DI SINI, bukan dengan mengunci tombol Tambah.
    // Pengguna tetap dapat membuka form; yang ditolak adalah penyimpanannya, dengan pesan
    // yang menyebut apa yang harus dilakukan.
    if (
      error.kode === ErrorCode.portalNotStated ||
      error.kode === ErrorCode.portalUnknown
    ) {
      return {
        title: 'Portal entitas belum dipilih',
        description:
          'Isian Anda belum tersimpan. Pilih portal entitas di bagian atas halaman ini, ' +
          'lalu tekan Simpan lagi — isian yang sudah diketik tidak hilang.',
        tone: 'penolakan',
      }
    }
    if (error.kode === ErrorCode.portalNotReady) {
      return {
        title: 'Basis data entitas ini belum tersedia',
        description:
          'Isian Anda belum tersimpan, dan mengulang tidak akan menolong. Hubungi ' +
          'administrator Claim PNC untuk melengkapi kredensial basis datanya.',
        tone: 'gangguan',
      }
    }
    return { title: 'Gagal menyimpan', description: error.message, tone: 'gangguan' }
  }
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Perubahan belum tersimpan. Periksa koneksi jaringan, lalu coba lagi.',
      tone: 'gangguan',
    }
  }
  return {
    title: 'Terjadi kesalahan pada sistem',
    description: 'Perubahan belum tersimpan. Coba lagi beberapa saat kemudian.',
    tone: 'gangguan',
  }
}

/**
 * Mengambil pesan galat per isian dari jawaban 422.
 *
 * Memakai `APIError.violations()` — helper bersama yang menyatukan kedua bentuk detail
 * galat yang hidup berdampingan di aplikasi ini (`detail[].field` dan `detail[].kolom`).
 * Menguraikannya sendiri di sini berarti layar ini harus tahu bentuk mana yang dipakai
 * modulnya, dan akan ikut berubah saat TKT-F1-004 menyeragamkannya.
 */
function fieldErrorOf(error: unknown): Record<string, string> {
  return error instanceof APIError ? error.violations() : {}
}

/**
 * Layar Detail Penyebab Kerugian.
 *
 * Pengganti `Harness/DetailCauseOfLoss-Harness.xml` atas POOLDATA.D_CAUSE_OF_LOSS
 * (MENU_ID 38).
 *
 * # Apa yang dikelola layar ini
 *
 * **Rincian di bawah Master Penyebab Kerugian** — butir-butir sebab kerugian yang
 * benar-benar dipilih petugas saat klaim diregistrasi. Master menyebut sebabnya secara umum
 * (kebakaran, kecelakaan, kehilangan); layar inilah yang memecahnya.
 *
 * # Induknya BELUM dibangun, dan layar ini hanya membacanya
 *
 * Master Penyebab Kerugian adalah MENU_ID 20 (`CauseOfLossInbox`), dan modulnya belum ada.
 * Isian "ID Master Kerugian" karena itu hanya MEMBACA `POOLDATA.V_M_CAUSE_OF_LOSS` sebagai
 * daftar pilihan — tidak ada satu pun jalur di layar ini yang menulis ke sana.
 *
 * Akibatnya yang harus disadari: **induk baru tidak dapat dibuat dari sini.** Bila sebuah
 * sebab kerugian belum ada di master, ia harus ditambahkan lewat Pega sampai modul induknya
 * dibangun.
 *
 * # TANPA tombol Hapus, dan itu mengikuti layar lama
 *
 * Tombol yang ada di Pega hanya `Simpan`, `Ubah`, `Cari Data`, dan `Clear Pencarian`.
 * `pyDeleteSQL` pada rule simpannya kosong, dan tidak ada satu pun activity penghapus di
 * export. `D-66` pun melarang penghapusan fisik data bernilai bisnis.
 *
 * Baris yang tidak lagi dipakai dinyatakan lewat **Status Aktif** — itulah gunanya kolom
 * itu, dan itulah satu-satunya cara menonaktifkan sebuah detail.
 *
 * # Daftarnya TIDAK menyaring yang tidak aktif
 *
 * Report Definition yang mengisi grid Pega — `BrowseVDCauseOfLoss_RD` — **tidak ada di
 * export** (`R-16`), sehingga tidak ada yang dapat memastikan apakah ia menyaring sesuatu.
 * Yang dipilih di sini adalah TIDAK menyaring, mengikuti
 * `RDB List/QueryGetAllDataCauseOfLoss-SQL.xml` yang membaca view yang sama tanpa satu pun
 * penyaring.
 *
 * Alasannya: baris tidak aktif yang disembunyikan akan tampak hilang bagi petugas, dan
 * tidak ada tombol mana pun di layar ini untuk memunculkannya kembali.
 *
 * # Isinya belum dapat dipakai terhadap Oracle
 *
 * Sampai hak akses `POOLDATA.D_CAUSE_OF_LOSS` diberikan DBA, layar ini berjalan atas
 * penyimpanan memori yang isinya **dikarang** — dan isi itu lenyap saat aplikasi berhenti.
 * Keadaan itu dinyatakan di layar alih-alih ditutupi.
 */
export function DetailPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const list = useCauseOfLossDetailList()
  const options = useCauseOfLossOptions()

  // null berarti form tertutup; '' berarti form terbuka untuk baris BARU.
  const [editingID, setEditingID] = useState<string | null>(null)
  const loaded = useCauseOfLossDetail(editingID === '' ? null : editingID)

  const create = useCreateCauseOfLossDetail()
  const save = useSaveCauseOfLossDetail()

  const busy = create.isPending || save.isPending
  const saveError = create.error ?? save.error
  const fieldError = fieldErrorOf(saveError)

  function closeForm() {
    setEditingID(null)
    create.reset()
    save.reset()
  }

  function submit(values: DetailFormValues) {
    const input: CauseOfLossDetailInput = {
      id_lama: values.id_lama,
      id_master: values.id_master,
      deskripsi_kerugian: values.deskripsi_kerugian,
      kode_kehilangan: values.kode_kehilangan,
      status_aktif: values.status_aktif,
      bisnis: values.bisnis,
    }

    if (editingID === '' || editingID === null) {
      create.mutate(input, { onSuccess: closeForm })
      return
    }
    save.mutate({ id: editingID, input }, { onSuccess: closeForm })
  }

  const columns: Column<CauseOfLossDetail>[] = [
    {
      key: 'id',
      title: 'ID',
      value: (row) => row.id,
      width: '9rem',
    },
    {
      key: 'nama_master',
      // Judulnya "ID Master Kerugian" — dibaca dari sel header grid
      // (`Section/BrowseDetailCauseOfLoss-Section.xml:8414`), BUKAN dari label form.
      //
      // # Judulnya menyebut ID, tetapi ISINYA sebutan master
      //
      // Sel datanya terikat `.pyNote` (`:9291`), dan `.pyNote` diisi sub-kueri pada
      // `RDB List/BrowseCOLByBisnis_Sql-SQL.xml:64`:
      //
      //   (select col_desc from v_m_cause_of_loss where m_col_id = a.m_col_id) as "pyNote"
      //
      // Jadi yang tampil adalah COL_DESC induknya, bukan kode M_COL_ID. Judul dan isi yang
      // tidak sejalan seperti ini lazim di layar lama, dan judulnya tetap disalin apa
      // adanya (`D-13`).
      title: 'ID Master Kerugian',
      value: (row) => row.nama_master,
      render: (row) =>
        row.nama_master === '' ? (
          <span className="text-slate-400">
            {row.id_master === '' ? '—' : `${row.id_master} (induk tidak ada)`}
          </span>
        ) : (
          row.nama_master
        ),
    },
    {
      key: 'deskripsi_kerugian',
      title: 'Deskripsi Kerugian',
      value: (row) => row.deskripsi_kerugian,
    },
    {
      key: 'status_aktif',
      title: 'Status Aktif',
      value: (row) => row.label_status_aktif,
      width: '9rem',
      render: (row) => (
        <span
          className={[
            'rounded-full px-2 py-0.5 text-xs ring-1',
            row.status_aktif === '1'
              ? 'bg-emerald-50 text-emerald-800 ring-emerald-200'
              : row.status_aktif === ''
                ? 'bg-slate-50 text-slate-600 ring-slate-200'
                : 'bg-amber-50 text-amber-800 ring-amber-200',
          ].join(' ')}
        >
          {row.label_status_aktif}
        </span>
      ),
    },
    {
      key: 'kode_kehilangan',
      title: 'Kode Kehilangan',
      value: (row) => row.kode_kehilangan,
      width: '10rem',
    },
    {
      key: 'aksi',
      // "Aksi", bukan disalin dari Pega. Ini SATU-SATUNYA judul kolom yang sengaja tidak
      // menyalin literal Pega: di sana sel judulnya berbunyi "Button" — sisa pengembang,
      // bukan istilah bisnis — dan kolom tanpa judul yang bermakna tampak TIDAK ADA bagi
      // pengguna yang membaca kepala tabel.
      //
      // Ditetapkan Work Owner 2026-10-03 dan berlaku seluruh modul; mengosongkannya sudah
      // dicoba pada Master Login 2026-10-04 dan ditolak pada hari yang sama.
      title: 'Aksi',
      value: () => '',
      noSort: true,
      alignRight: true,
      width: '6rem',
      render: (row) => (
        <Button tone="halus" onClick={() => setEditingID(row.id)}>
          Ubah
        </Button>
      ),
    },
  ]

  const loadError = list.error === null ? null : loadMessage(list.error)
  const formMessage = saveError == null ? null : saveMessage(saveError)

  /**
   * Panel form, diangkat menjadi variabel supaya urutannya di layar terbaca dalam satu
   * baris pada JSX di bawah alih-alih tersembunyi di balik puluhan baris markup.
   */
  const formPanel =
    editingID === null ? null : (
      <section className="rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut">
        <h2 className="text-base font-semibold text-slate-900">
          {editingID === '' ? 'Tambah Detail Penyebab Kerugian' : 'Memperbaharui Data'}
        </h2>

        {formMessage !== null && (
          <div className="mt-3">
            <ErrorMessage
              title={formMessage.title}
              description={formMessage.description}
              tone={formMessage.tone}
            />
          </div>
        )}

        <div className="mt-4">
          {editingID !== '' && loaded.isLoading ? (
            <p className="text-sm text-slate-500">Memuat baris…</p>
          ) : editingID !== '' && loaded.error !== null ? (
            <ErrorMessage
              title="Baris ini tidak dapat dimuat"
              description="Tutup form ini dan muat ulang daftarnya."
              tone="gangguan"
            />
          ) : (
            <DetailForm
              // key memaksa form lahir ulang saat berpindah baris. Tanpa itu, nilai
              // awalnya tidak ikut berganti — `useState` hanya membaca nilai awal sekali.
              key={editingID === '' ? 'baru' : editingID}
              existing={editingID === '' ? null : (loaded.data?.detail ?? null)}
              options={options.data?.status_aktif ?? []}
              fieldError={fieldError}
              busy={busy}
              onSubmit={submit}
              onCancel={closeForm}
            />
          )}
        </div>
      </section>
    )

  return (
    // PEMBUNGKUS HALAMAN — `mx-auto max-w-6xl px-4 py-8 sm:px-6`, sama dengan layar master
    // lain. Tanpa ini isinya menempel ke sidebar; dilaporkan Work Owner 2026-10-05.
    //
    // `<div>`, BUKAN `<main>` seperti kebanyakan layar master: `src/app/PageShell.tsx:144`
    // sudah merender `<main>` mengelilingi setiap halaman, dan landmark `main` bersarang
    // tidak sah menurut HTML. Bentuk `<div>` berkelas sama sudah dipakai Master Status
    // Klaim, Home, Dashboard Klaim, dan Ambang Komite.
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      {/*
        KEPALA HALAMAN DI LUAR KANVAS TABEL.

        Judul, keterangan, dan kedua tombol sempat diserahkan ke prop `title`,
        `description`, dan `actions` milik `DataTable` — sehingga semuanya tergambar DI
        DALAM kartu yang sama dengan tabelnya. Dilaporkan Work Owner 2026-10-05.

        Ia menyimpang sendirian: dari 90 berkas pemakai `DataTable`, hanya DUA lagi yang
        memberinya `title`, dan keduanya tabel TERSEMAT di dalam lembar kasus Inbox Komite
        (`ClaimSheet`, `CommitteeBottom`) — bukan halaman. Seluruh halaman lain menggambar
        kepalanya sendiri di atas kartu, persis seperti di sini.
      */}
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/* Judulnya dibaca dari MENU_DESC pada m_menu_aplikasi_pnc.csv:33 (D-13). */}
          <h1 className="text-xl font-semibold text-slate-900">Detail Penyebab Kerugian</h1>
          {/*
            SATU kalimat saja, dan panjangnya disengaja.

            Keterangan ini sempat dua kalimat (±180 aksara). Pada `<header>` ber-`flex-wrap`,
            lebar max-content sebuah paragraf panjang mendorong kelompok tombol TURUN ke
            baris berikutnya — sehingga Tambah dan Refresh tidak lagi sejajar judul di
            kanan. Dilaporkan Work Owner 2026-10-05.

            Panjangnya kini sebanding dengan layar master lain (Master Login Surveyor
            ±85 aksara), sehingga tata letaknya berperilaku sama tanpa menambah kelas
            penahan apa pun.

            Kalimat keduanya sempat dipindah ke `<footer>`; kaki halaman itu DICABUT atas
            permintaan Work Owner 2026-10-05.
          */}
          <p className="text-sm text-slate-600">
            Rincian di bawah Master Penyebab Kerugian — butir yang dipilih petugas saat
            klaim diregistrasi.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button
            tone="utama"
            // Digerbangi HANYA oleh form yang sedang terbuka, sama seperti layar master
            // lain. Portal yang belum dipilih sengaja TIDAK ikut menggerbanginya — server
            // yang menolaknya (TKT-F6-002), dan penolakannya diterjemahkan saveMessage.
            disabled={editingID !== null}
            onClick={() => {
              create.reset()
              save.reset()
              setEditingID('')
            }}
          >
            Tambah
          </Button>
          {/*
            Refresh. Layar Pega tidak punya tombol ini — keempat tombolnya hanya Simpan,
            Ubah, Cari Data, dan Clear Pencarian — tetapi di sana grid memuat ulang sendiri
            setiap kali form disimpan. Di aplikasi ini daftarnya di-cache TanStack Query,
            sehingga tanpa tombol ini tidak ada cara menarik perubahan petugas LAIN tanpa
            memuat ulang seluruh halaman.
          */}
          <Button
            tone="kedua"
            onClick={() => void list.refetch()}
            disabled={list.isFetching}
          >
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
        </div>
      </header>

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani empat
          badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh hanya
          diandaikan pengguna (ADR-0030, R-20).

          Pada layar ini ia berarti: sebab kerugian yang dapat dipilih menentukan bagaimana
          klaim dinilai pada badan hukum itu. */}
      <p className="mt-3 text-xs text-slate-500">
        Daftar ini memuat seluruh detail penyebab kerugian pada entitas yang sedang dibuka.
        <span className="ml-1">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">
            {list.data?.portal ?? portal ?? '—'}
          </span>
        </span>
      </p>

      {/*
        PANEL FORM DI ATAS TABEL.

        Sebelumnya di bawah, sehingga menekan Tambah atau Ubah membuka form di luar layar
        dan pengguna harus menggulir — dilaporkan Work Owner 2026-10-05.

        Posisi ini JUGA yang benar terhadap layar lama: pada
        `Section/BrowseDetailCauseOfLoss-Section.xml`, panel "Memperbaharui Data" ada di
        baris ~1165 sedangkan gridnya baru di ~7923.
      */}
      {formPanel !== null && <section className="mt-5">{formPanel}</section>}

      <section className="mt-6">
        {portal === null ? (
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        ) : list.isPending ? (
          <p className="text-sm text-slate-500">Memuat daftar detail penyebab kerugian…</p>
        ) : loadError !== null ? (
          <ErrorMessage
            title={loadError.title}
            description={loadError.description}
            tone={loadError.tone}
          />
        ) : (
          <DataTable
            columns={columns}
            rows={list.data?.detail ?? []}
            rowKey={(row) => row.id}
            description="Sumber: POOLDATA.V_D_CAUSE_OF_LOSS"
            searchLabel="Cari detail penyebab kerugian"
            pageSize={PAGE_SIZE}
            emptyMessage="Belum ada detail penyebab kerugian pada entitas ini."
          />
        )}
      </section>

    </div>
  )
}
