import { useMemo, useState } from 'react'

import { WorkshopStatus, type Workshop } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'

import { useCreateWorkshop, useSaveWorkshop, useWorkshopList } from './api'
import { DocumentPanel } from './DocumentPanel'
import { WorkshopForm, type WorkshopFormValues } from './WorkshopForm'
import { compareCodeUnits } from '@/lib/sort'

/**
 * Tiga tab, sama persis dengan layar lama — termasuk URUTANNYA.
 *
 * Caption-nya dibaca dari `Section/BrowseMasterHE-Section.xml` dan urutannya dari layar
 * Pega yang berjalan: **Approve · Reject · Waiting Approval**. Ketiganya membaca
 * `BrowseBengkelHE_RD` yang sama, berbeda hanya pada nilai APPROVAL-nya — yang di Pega
 * dikirim sebagai parameter report definition (`pyReportDefParams`, `APPROVAL="1"` pada
 * tab Approve).
 *
 * Urutannya bukan detail kosmetik: tab pertama adalah yang terbuka saat layar dibuka, dan
 * "Reject" di tengah berarti petugas yang terbiasa menekan tab kedua akan menekan tab yang
 * berbeda bila urutannya diubah.
 */
const TABS = [
  {
    id: 'approve',
    label: 'Approve',
    status: WorkshopStatus.disetujui,
    description: 'Bengkel yang sudah disetujui dan berlaku sebagai rekanan.',
  },
  {
    id: 'reject',
    label: 'Reject',
    status: WorkshopStatus.ditolak,
    description: 'Pengajuan yang ditolak. Dapat diperbaiki lalu diajukan ulang.',
  },
  {
    id: 'menunggu',
    label: 'Waiting Approval',
    status: WorkshopStatus.menunggu,
    description:
      'Pengajuan dan perubahan yang belum diputuskan. Centang barisnya untuk menyetujui atau menolak.',
  },
] as const

type TabId = (typeof TABS)[number]['id']

/**
 * Banyaknya baris per halaman.
 *
 * Dibaca dari `pyPageSizeOther` pada `Section/BrowseMasterHEApprove-Section.xml`, yang
 * bernilai `20` karena `pyPageSize` di section yang sama bernilai `"Other"`. Ia angka
 * milik LAYAR INI — layar master lain memakai 15 — sehingga ia disebut di sini, bukan
 * dijadikan bawaan komponen tabel.
 */
const PAGE_SIZE = 20

/** Kolom yang nilai sahnya tidak ada di export; sarannya dikumpulkan dari data. */
const SUGGESTED_COLUMNS = [
  'status_bengkel',
  'jenis_pph',
  'status_disupply_asm',
  'status_eklaim',
  'status_auto_aksep',
  'status_payment',
  'status_autopayment',
  'status_tekno',
  'status_order',
] as const

/**
 * Keterangan di dalam grid saat tidak ada satu baris pun tergambar.
 *
 * # Gagal dan kosong sengaja tampil sama
 *
 * Layar ini TIDAK membedakan "tabelnya memang kosong" dari "tabelnya gagal dibaca".
 * Keduanya tampil identik. Pada saat tulisan ini dibuat, view `POOLDATA.BENGKEL_HE`
 * sedang rusak di Oracle (`ORA-04063`, karena `ACCOUNT_ID` pada tabel dasarnya tidak
 * lagi dikenali), dan layar ini menampilkannya sebagai data kosong.
 *
 * Keputusan Work Owner (2026-09-28), mengikuti keputusan yang sama untuk Master
 * Sparepart pada 2026-09-24. Yang menggantikan pembedaan itu ada di dua tempat yang
 * TIDAK dilihat pengguna:
 *
 *	log backend          setiap kegagalan tercatat lengkap dengan galat Oracle-nya
 *	claimpnc -periksa    menyebut objek dan galatnya, beserta kueri katalog penjawabnya
 *
 * Akibat yang diterima secara sadar: pengguna tidak punya cara membedakan tab yang
 * memang belum berisi dari basis data yang sedang rusak, sehingga kerusakan berikutnya
 * hanya terbaca dari log — bukan dari layar.
 *
 * Satu pengecualian yang dipertahankan: sebelum portal dipilih, kuerinya belum pernah
 * dijalankan sama sekali — tidak ada "hasil nol" untuk dilaporkan, dan yang dibutuhkan
 * pengguna adalah petunjuk tindakan, bukan keterangan data.
 */
function emptyMessageFor(hasPortal: boolean): string {
  if (!hasPortal) {
    return 'Pilih portal entitas di bagian atas halaman untuk menampilkan daftarnya.'
  }
  return 'Data tidak ada'
}

/**
 * Layar Master Bengkel.
 *
 * Pengganti `Harness/BengkelHE-Harness.xml` atas tabel POOLDATA.BENGKEL_HE (MENU_ID 28).
 *
 * # Apa yang dikelola layar ini
 *
 * Daftar **bengkel rekanan** beserta syarat kerja samanya: ke mana pembayarannya dikirim,
 * berapa diskon jasa dan sparepart yang berlaku, pajak apa yang dipotong, berapa SLA-nya,
 * dan kanal mana saja yang boleh dipakai bengkel itu. Sebagian kolomnya menentukan hasil
 * hitungan uang pada klaim yang memakai bengkel tersebut — ia master komersial, bukan
 * sekadar daftar alamat.
 *
 * # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
 *
 * Itu bukan efek samping melainkan langkah tersendiri di sistem lama:
 * `Activity/UpdateBengkelHE_act` step 7 menetapkan `APPROVAL := "0"` tanpa syarat apa
 * pun. Bengkel yang sudah disetujui lalu disunting kembali menunggu — dan itu benar untuk
 * master yang menentukan diskon, pajak, dan rekening tujuan pembayaran.
 *
 * # Approve dan Reject TIDAK ada di layar ini — sama seperti Pega
 *
 * Tab Waiting Approval hanya MENDAFTAR dan MENGUBAH. Tidak ada cara menyetujui dari sini,
 * dan itu bukan kekurangan: `Section/BrowseMasterHEApproval-Section.xml` memang tidak
 * punya tombol keputusan sama sekali — nol rujukan ke `SetApprovalAllMaster`, nol Select
 * All, dan keenam kemunculan kata "Approve"/"Reject" di dalamnya hanyalah nama halaman
 * internal Pega (`pgRepPgSubSectionBrowseMasterHEApproveBB`).
 *
 * Keputusannya ada di `Section/ApprovalMasterBengkelHE`, yang dipakai `InboxManager_Sec`
 * — satu layar "Approval Master" berisi enam master sekaligus, dijalankan
 * `Activity/SetApprovalAllMaster`.
 *
 * # Bilah keputusan yang sempat ada di sini, dan kenapa dicabut
 *
 * Versi pertama modul ini memasang centang dan tombol keputusan di tab Waiting Approval,
 * karena saat itu Inbox Manager belum dibangun dan setiap bengkel baru akan tertahan tanpa
 * satu pun cara menyelesaikannya.
 *
 * Alasan itu habis masa berlakunya. Modul `inbox-manager` (MENU_ID 58) sudah ada beserta
 * tab Master Bengkel-nya, dan kueri `decide_bengkel` di sana memang meng-UPDATE
 * `POOLDATA.BENGKEL_HE`. Membiarkan keduanya berarti satu keputusan punya dua pintu, dan
 * yang satu tidak berpadanan di layar lama.
 *
 * Dicabut atas permintaan Work Owner (2026-10-03).
 */
export function WorkshopPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const [tab, setTab] = useState<TabId>('approve')
  const [isAdding, setAdding] = useState(false)
  const [editing, setEditing] = useState<Workshop | null>(null)
  const [isDocumentOpen, setDocumentOpen] = useState(false)

  const active = TABS.find((t) => t.id === tab) ?? TABS[0]
  const list = useWorkshopList(active.status)
  const create = useCreateWorkshop()
  const save = useSaveWorkshop()

  const isFormOpen = isAdding || editing !== null
  const rows = list.data?.bengkel ?? []

  /*
    Apakah jalur unggah siap dipakai, dibaca dari jawaban daftar.

    Layar memerlukannya SEBELUM pengguna memilih berkas: menonaktifkan tombolnya beserta
    sebabnya jauh lebih terbaca daripada membiarkan pengguna memilih berkas, menunggu
    unggahan, lalu menerima penolakan.

    Bawaannya `false` selama daftar belum termuat — tombol yang sempat hidup lalu mati
    sendiri lebih membingungkan daripada tombol yang baru hidup setelah jawabannya tiba.
  */
  const uploadAvailable = list.data?.unggah_tersedia ?? false

  /*
    Saran nilai untuk kolom yang daftar pilihannya tidak ada di export (R-16).

    Dikumpulkan dari baris yang SEDANG TERMUAT, bukan dari daftar yang dikarang. Ia
    jawaban terbaik yang tersedia atas pertanyaan "nilai apa yang sah di kolom ini" —
    lihat WorkshopForm bagian "Penanda sistem".
  */
  const knownValues = useMemo(() => {
    const collected: Record<string, string[]> = {}
    for (const column of SUGGESTED_COLUMNS) {
      const unique = new Set<string>()
      for (const row of rows) {
        const value = row[column]
        if (value !== '') unique.add(value)
      }
      collected[column] = [...unique].sort(compareCodeUnits)
    }
    return collected
  }, [rows])

  function closeForm() {
    create.reset()
    save.reset()
    setAdding(false)
    setEditing(null)
    // Panel dokumen melekat pada baris yang sedang dibuka; menutup formnya tanpa menutup
    // panelnya menyisakan panel yang menunjuk bengkel yang sudah tidak terpilih.
    setDocumentOpen(false)
  }

  function openAdd() {
    create.reset()
    save.reset()
    setEditing(null)
    setAdding(true)
    // Baris baru belum punya ID, sehingga belum ada yang dapat dilampiri.
    setDocumentOpen(false)
  }

  function openEdit(row: Workshop) {
    create.reset()
    save.reset()
    setAdding(false)
    setEditing(row)
    // Berpindah baris menutup panelnya: dokumen yang tampil harus selalu milik bengkel yang
    // sedang dibuka, bukan sisa dari baris sebelumnya.
    setDocumentOpen(false)
  }

  function submit(values: WorkshopFormValues) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal — dan pada form tiga puluh tiga
    // isian, itu kehilangan yang tidak dapat dimaafkan.
    if (editing) {
      save.mutate({ id: editing.id_bengkel, input: values }, { onSuccess: closeForm })
      return
    }
    create.mutate(values, { onSuccess: closeForm })
  }


  /*
    Susunan kolom mengikuti grid Pega APA ADANYA.

    `Section/BrowseMasterHEApprove-Section.xml` menyetel `pyColumnCount = 7`, dan keenam
    kolom datanya muncul pada urutan ini di dalam definisi gridnya:

      ID_BENGKEL       ID Bengkel        <- kolom pertama, pySortType=DESC pySortOrder=1
      NAMA_BENGKEL     Nama Bengkel
      ALM_BENGKEL      Alamat Bengkel
      TELP_BENGKEL     Telp Bengkel
      NOHP_BENGKEL     No HP Bengkel
      LOGIN_APLIKASI   Login Aplikasi
      (aksi)           Ubah

    Ketiga puluh empat kolom selebihnya ADA di tabel tetapi TIDAK di grid — keseluruhannya
    hanya muncul di form. Menampilkan lebih banyak "supaya informatif" berarti membuat
    layar yang berbeda dari yang dipakai petugas hari ini (`D-13`).

    TIDAK ADA kolom tambahan. Kotak centang untuk keputusan borongan sempat ada pada tab
    Waiting Approval, lalu DICABUT pada 2026-10-03 — lihat doc comment WorkshopPage.
  */
  const columns: Column<Workshop>[] = [
    {
      key: 'id_bengkel',
      title: 'ID Bengkel',
      width: '10rem',
      value: (row) => row.id_bengkel,
    },
    {
      key: 'nama_bengkel',
      title: 'Nama Bengkel',
      value: (row) => row.nama_bengkel,
    },
    {
      key: 'alamat_bengkel',
      title: 'Alamat Bengkel',
      value: (row) => row.alamat_bengkel,
    },
    {
      key: 'telp_bengkel',
      title: 'Telp Bengkel',
      width: '10rem',
      value: (row) => row.telp_bengkel,
    },
    {
      key: 'nohp_bengkel',
      title: 'No HP Bengkel',
      width: '10rem',
      value: (row) => row.nohp_bengkel,
    },
    {
      key: 'login_aplikasi',
      title: 'Login Aplikasi',
      width: '11rem',
      value: (row) => row.login_aplikasi,
      render: (row) => <LoginCell row={row} />,
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: '6rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      // Hanya "Ubah".
      //
      // Tombol "Dokumen" per baris DICABUT pada 2026-10-03: di Pega unggah dokumen adalah
      // tombol tingkat layar, dan Work Owner meminta layar ini mengikutinya. Menyisakan
      // keduanya berarti layar kita punya tombol yang tidak ada di layar lama.
      render: (row) => (
        <Button tone="kedua" onClick={() => openEdit(row)} disabled={save.isPending}>
          Ubah
        </Button>
      ),
    },
  ]

  return (
    <main className="mx-auto max-w-7xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/*
            "Master Bengkel HE", bukan "Master Bengkel".

            Itu judul yang tertulis di layar Pega — caption
            `Section/MasterBengkelHE-Section.xml`. Butir menunya memang bernama "Master
            Bengkel" (`M_MENU_APLIKASI_PNC.MENU_DESC`), dan keduanya memang berbeda di
            sistem lama; `D-13` menuntut teks LAYAR yang diikuti.
          */}
          <h1 className="text-xl font-semibold text-slate-900">Master Bengkel HE</h1>
          <p className="text-sm text-slate-600">
            Bengkel rekanan beserta syarat kerja samanya — rekening pembayaran, diskon, pajak,
            dan SLA.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button tone="utama" onClick={openAdd} disabled={isFormOpen}>
            Tambah
          </Button>
          {/*
            Tombol PERTAMA layar Pega, captionnya dari field value `pyButtonLabel Upload
            Document` pada `Section/BrowseMasterHE-Section.xml`.

            Ia bergantung pada DUA hal, dan keduanya punya sebab berbeda:

              `uploadAvailable`  layanan penyimpanan dokumen terpasang di lingkungan ini
              `editing`          bengkel mana yang dilampiri

            Yang kedua bukan selera. Di Pega ia tombol tingkat layar, tetapi yang disimpannya
            adalah `BENGKEL_HE.DOKUMENID` — kolom milik SATU baris. Pada layar daftar, tombol
            tingkat layar tidak menyatakan baris mana yang dilampiri, sehingga ia disandarkan
            pada baris yang sedang dibuka. Master Sparepart menempuh cara yang sama.

            Tombol KEDUA layar lama — "Upload Data Master Bengkel", unggah CSV — TETAP
            DITARIK (Work Owner, 2026-10-07): ia bukan bagian migrasi yang dikerjakan di
            sini. Ketiadaannya disengaja, bukan rule yang belum ditemukan.
          */}
          <Button
            tone="kedua"
            onClick={() => { setDocumentOpen(true) }}
            disabled={!uploadAvailable || editing === null || isDocumentOpen}
          >
            Upload Document
          </Button>
          <Button tone="kedua" onClick={() => { list.refetch() }} disabled={list.isFetching}>
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
        </div>
      </header>

      {/*
        Panel digambar DI BAWAH kepala halaman dan DI ATAS tab, sama seperti Master Sparepart.

        Letaknya tidak meniru Pega, dan itu disengaja: di Pega ia modal (`Flow Action`
        `UploadDocument` dibuka sebagai overlay). Aplikasi ini tidak memakai modal — form
        tambah dan ubah pun digambar sebagai panel inline — sehingga satu pola dipakai untuk
        semuanya.
      */}
      {isDocumentOpen && editing !== null && (
        <DocumentPanel
          workshop={editing}
          available={uploadAvailable}
          onClose={() => { setDocumentOpen(false) }}
        />
      )}

      <nav
        aria-label="Tab Master Bengkel"
        className="mt-4 flex flex-wrap gap-1 border-b border-slate-200"
      >
        {TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            aria-current={tab === t.id ? 'page' : undefined}
            onClick={() => {
              closeForm()
              setTab(t.id)
            }}
            className={[
              'rounded-t px-3 py-2 text-sm font-medium',
              'transition-colors duration-150 ease-halus',
              'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
              tab === t.id
                ? 'border-b-2 border-blue-600 text-blue-700'
                : 'text-slate-500 hover:text-slate-800',
            ].join(' ')}
          >
            {t.label}
          </button>
        ))}
      </nav>

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani empat
          badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh hanya
          diandaikan pengguna (ADR-0030, R-20). */}
      <p className="mt-3 text-xs text-slate-500">
        {active.description}{' '}
        <span className="ml-1">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
        </span>
      </p>

      {isFormOpen && (
        <section className="mt-5">
          <WorkshopForm
            editing={editing}
            knownValues={knownValues}
            isSaving={create.isPending || save.isPending}
            error={editing ? save.error : create.error}
            onSave={submit}
            onCancel={closeForm}
          />
        </section>
      )}

      {/*
        Tabelnya digambar dalam SETIAP keadaan — termuat, kosong, gagal, bahkan sebelum
        portal dipilih. Kolomnya karena itu selalu terlihat, persis seperti grid Pega yang
        menggambar kepala kolomnya beserta pyGridNoResultsMessage di bawahnya.

        Tidak ada lagi kotak galat yang menggantikan tabelnya: keterangan apa pun tinggal
        di dalam grid lewat emptyMessage. Lihat emptyMessageFor untuk alasan gagal dan
        kosong sengaja tampil sama.
      */}
      <section className="mt-6">
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(row) => row.id_bengkel}
          description="Sumber: POOLDATA.BENGKEL_HE"
          searchLabel="Cari bengkel"
          pageSize={PAGE_SIZE}
          isLoading={portal !== null && list.isPending}
          showHeaderWhenEmpty
          emptyMessage={emptyMessageFor(portal !== null)}
        />
      </section>
    </main>
  )
}


/**
 * LoginCell menggambar kolom Login Aplikasi.
 *
 * Isinya nilai kolom apa adanya — persis seperti grid Pega, yang juga mengosongkannya
 * bila belum ada.
 *
 * # Satu keterangan yang DITAMBAHKAN, dan hanya pada satu keadaan
 *
 * Bengkel berstatus **rekanan** yang login aplikasinya kosong ditandai. Di Pega keadaan
 * itu terlihat sebagai sel kosong yang tidak berbeda dari sel kosong milik bengkel
 * non-rekanan — padahal artinya jauh berbeda: yang satu memang tidak diberi login, yang
 * lain **seharusnya punya tetapi tidak**, dan bengkelnya tidak akan pernah dapat masuk.
 *
 * Kolom "Rekanan" tersendiri sempat ada di layar ini lalu dicabut, karena grid Pega tidak
 * punya kolom itu. Keterangan ini yang menggantikannya: ia menyampaikan hal yang sama
 * pada baris yang benar-benar memerlukannya, tanpa menambah kolom.
 */
function LoginCell({ row }: { row: Workshop }) {
  if (row.login_aplikasi !== '') {
    return <span>{row.login_aplikasi}</span>
  }
  if (!row.rekanan) {
    return <span className="text-slate-400">—</span>
  }
  return (
    <span className="inline-flex flex-col">
      <span className="text-slate-400">—</span>
      <span className="text-xs text-amber-700">rekanan tanpa login</span>
    </span>
  )
}
