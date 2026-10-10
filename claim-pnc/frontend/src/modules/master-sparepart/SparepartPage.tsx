import { useMemo, useState } from 'react'

import { SparepartStatus, type Sparepart } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'

import {
  useCreateSparepart,
  useImportSparepartCSV,
  useSaveSparepart,
  useSparepartDocument,
  useSparepartList,
  useSparepartOptions,
  useUploadSparepartDocument,
} from './api'
import { DocumentField } from './DocumentField'
import { ImportCSVPanel } from './ImportCSVPanel'
import { SparepartForm, type SparepartFormValues } from './SparepartForm'
import { compareCodeUnits } from '@/lib/sort'

/**
 * Tiga tab, sama persis dengan layar lama — termasuk URUTANNYA.
 *
 * `Section/BrowseMasterSparepartHE-Section.xml` memuat tiga section yang ketiganya membaca
 * `BrowseSparepartHE_RD` yang sama dan hanya berbeda pada nilai APPROVAL-nya. Urutan
 * kemunculannya di dalam section itu:
 *
 *	BrowseMasterSparepartHEApprove   APPROVAL="1"
 *	BrowseMasterSparepartHEReject    APPROVAL="2"
 *	BrowseMasterSparepartHEApproval  APPROVAL="0"
 *
 * Reject berada di TENGAH, bukan di ujung — sama seperti Master Panel dan Master Bengkel.
 * Urutan itu tidak intuitif, tetapi ia yang dilihat petugas hari ini, dan `D-13` menuntut
 * tata letak yang sama supaya pengguna tidak perlu belajar ulang.
 */
const TABS = [
  {
    id: 'approve',
    label: 'Approve',
    status: SparepartStatus.disetujui,
    description: 'Sparepart yang sudah disetujui dan berlaku.',
  },
  {
    id: 'reject',
    label: 'Reject',
    status: SparepartStatus.ditolak,
    description: 'Pengajuan yang ditolak. Dapat diperbaiki lalu diajukan ulang.',
  },
  {
    id: 'menunggu',
    label: 'Waiting Approval',
    status: SparepartStatus.menunggu,
    description:
      'Pengajuan dan perubahan yang belum diputuskan. Centang barisnya untuk menyetujui atau menolak.',
  },
] as const

type TabId = (typeof TABS)[number]['id']

/**
 * Ukuran halaman diambil dari `pyPageSize` pada ketiga section tab Master Sparepart.
 *
 * Ketiganya bernilai **30** — berbeda dari Master Panel yang 15 pada satu tab dan 50 pada
 * dua tab lainnya, dan dari Master Bengkel yang 20. Tidak ada satu angka yang benar untuk
 * seluruh layar; angkanya milik layar, bukan milik komponen tabel.
 */
const PAGE_SIZE = 30

/** Kolom penanda yang nilai sahnya tidak ada di export; pilihannya dikumpulkan dari data. */
const MARK_COLUMNS = [
  'jenis_sparepart',
  'satuan',
  'status_aktif',
  'status_sparepart',
] as const

/**
 * Kalimat yang mengisi badan grid ketika tidak ada satu pun baris yang tergambar.
 *
 * # Satu kalimat untuk SETIAP keadaan nol baris
 *
 * Grid Pega menggambar kepala kolomnya beserta satu pesan di bawahnya saat hasilnya nol,
 * dan pesannya berbunyi "data tidak ada". Layar ini mengikutinya apa adanya (`D-13`):
 * berhasil-tetapi-kosong dan gagal-dibaca memakai kalimat yang SAMA.
 *
 * Teksnya sendiri TIDAK dapat dibaca dari export — grid lama mengambilnya dari field value
 * `GridNoResultsOnLoad`, dan tidak ada satu pun direktori `Field Value/` di sana (`R-16`).
 * Ia datang dari Work Owner yang membaca layar Pega sungguhan (2026-09-24). Tanpa nama tab,
 * persis seperti Pega: pesannya sama di ketiga tab.
 *
 * # Yang hilang karenanya, dan di mana menggantinya
 *
 * Layar TIDAK LAGI membedakan "tabelnya memang kosong" dari "tabelnya gagal dibaca".
 * Keduanya tampil identik. Pada saat tulisan ini dibuat, view `POOLDATA.SPAREPART_HE`
 * sedang rusak di Oracle (`ORA-04063`), dan layar ini menampilkannya sebagai data kosong.
 *
 * Keputusan Work Owner, diambil setelah akibatnya disampaikan tiga kali (2026-09-24). Yang
 * menggantikan pembedaan itu ada di dua tempat yang TIDAK dilihat pengguna:
 *
 *	log backend          setiap kegagalan tercatat lengkap dengan galat Oracle-nya
 *	claimpnc -periksa    menyebut objek dan galatnya, beserta kueri katalog penjawabnya
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
 * Layar Master Sparepart.
 *
 * Pengganti `Harness/SparePart_HE-Harness.xml` atas tabel POOLDATA.SPAREPART_HE
 * (MENU_ID 31).
 *
 * # Apa yang dikelola layar ini
 *
 * Daftar **suku cadang alat berat** beserta harga jual, dimensi, batas stok, dan
 * penggolongannya. Setiap sparepart menunjuk satu Kategori dan satu Tipe yang dibaca dari
 * dua tabel acuan.
 *
 * # Lima kolom, bukan dua puluh tiga
 *
 * Grid Pega hanya menampilkan ID, Nama, Harga Jual, User Update, dan Tanggal Update; sisanya
 * hanya terlihat saat sebuah baris dibuka. Susunan itu ditiru apa adanya atas keputusan Work
 * Owner (2026-09-20), termasuk tidak menambahkan kolom yang menurut kami berguna.
 *
 * Satu kolom Pega TIDAK digambar: sel yang di layar lama terikat pada `.TELP_BENGKEL` —
 * properti yang tidak ada di SPAREPART_HE, jadi selalu kosong. Ia sisa salin-tempel dari
 * grid Master Bengkel, dan menggambar kolom yang selalu kosong bukan kesetaraan melainkan
 * peniruan cacat.
 *
 * # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
 *
 * Itu bukan efek samping melainkan langkah tersendiri di sistem lama:
 * `Activity/UpdateSparepartHE_act` menetapkan `APPROVAL := "0"` tanpa syarat apa pun.
 *
 * # TIDAK ada Approve dan Reject di layar ini — dan itu mengikuti Pega
 *
 * Ketiga tabnya berperilaku SAMA: daftar, ditambah tombol Ubah per baris. Terbukti dari
 * ketiga sectionnya, yang hanya memuat dua tombol — `SIMPAN` dan `Ubah` — dan **nol**
 * `pxCheckbox` serta **nol** `pySelected`:
 *
 *	Section/BrowseMasterSparepartHEApprove-Section.xml    SIMPAN · Ubah
 *	Section/BrowseMasterSparepartHEReject-Section.xml     SIMPAN · Ubah
 *	Section/BrowseMasterSparepartHEApproval-Section.xml   SIMPAN · Ubah
 *
 * Centang, Select All, Approve, dan Reject hanya ada di `ApprovalMasterSparepartHE` — layar
 * **Inbox Manager**, bukan layar ini. Di sanalah `Activity/SetApprovalAllMaster` dipanggil.
 *
 * Layar ini sempat menggambar bilah keputusan itu dengan alasan "Inbox Manager belum
 * dibangun, tanpa ini pengajuan tertahan tanpa cara menyelesaikannya". Work Owner
 * mencabutnya (2026-10-03): tab Waiting Approval harus mengikuti perilaku Pega, dan
 * persetujuan ditempatkan di layar yang memang memilikinya.
 *
 * Akibat yang diterima: sampai Inbox Manager dibangun, **tidak ada satu pun layar yang dapat
 * menyetujui sparepart**. Endpoint `POST /api/master/sparepart/keputusan` beserta
 * `useDecideSparepart` sengaja TIDAK dihapus — keduanya sudah teruji dan menunggu layar yang
 * benar, bukan menunggu ditulis ulang.
 */
export function SparepartPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const [tab, setTab] = useState<TabId>('approve')
  const [isAdding, setAdding] = useState(false)
  const [editing, setEditing] = useState<Sparepart | null>(null)

  const active = TABS.find((t) => t.id === tab) ?? TABS[0]
  const list = useSparepartList(active.status)
  const create = useCreateSparepart()
  const save = useSaveSparepart()
  const upload = useUploadSparepartDocument()

  /*
    Berkas dan catatannya dipegang HALAMAN, bukan panelnya, karena unggahannya terjadi
    SESUDAH penyimpanan berhasil — dan yang tahu penyimpanan berhasil adalah halaman.

    Keduanya TIDAK dihapus saat form dibuka: berkas dipilih lebih dulu, form dibuka
    kemudian, dan menghapusnya di sana membuang berkas tepat sebelum Simpan ditekan tanpa
    satu pun tanda. Pega pun menyimpannya di halaman klipboard yang tidak ikut terhapus.
    Yang menghapusnya hanya closeForm, SESUDAH penyimpanan beserta unggahannya selesai.
  */
  const [documentFile, setDocumentFile] = useState<File | null>(null)
  const [documentNote, setDocumentNote] = useState('')

  /*
    Panel unggah yang sedang terbuka — SATU saja pada satu waktu.

    Keduanya menempati tempat yang sama di bawah kepala halaman, dan membuka dua sekaligus
    hanya memanjangkan layar tanpa menambah apa pun. Bentuknya sama dengan Master Panel, yang
    menampung tiga panel pada satu keadaan.
  */
  const [panelUnggah, setPanelUnggah] = useState<'dokumen' | 'csv' | null>(null)

  function bukaPanel(nama: 'dokumen' | 'csv') {
    setPanelUnggah((sekarang) => (sekarang === nama ? null : nama))
  }

  const importCSV = useImportSparepartCSV()
  const options = useSparepartOptions()
  const documentOf = useSparepartDocument(editing?.id_sparepart ?? null)

  /** Dibaca dari `unggah_tersedia`; menentukan hidup-matinya tombol di kepala halaman. */
  const uploadAvailable = options.data?.unggah_tersedia ?? false

  const isFormOpen = isAdding || editing !== null
  const rows = list.data?.sparepart ?? []

  /*
    Pilihan nilai untuk keempat penanda yang daftar pilihannya tidak ada di export (R-16).

    Dikumpulkan dari baris yang SEDANG TERMUAT, bukan dari daftar yang dikarang. Ia jawaban
    terbaik yang tersedia atas pertanyaan "nilai apa yang sah di kolom ini" — lihat
    ChoiceField.
  */
  const knownValues = useMemo(() => {
    const collected: Record<string, string[]> = {}
    for (const column of MARK_COLUMNS) {
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
    upload.reset()
    setDocumentFile(null)
    setDocumentNote('')
    setAdding(false)
    setEditing(null)
  }

  function openAdd() {
    create.reset()
    save.reset()
    setEditing(null)
    setAdding(true)
  }

  function openEdit(row: Sparepart) {
    create.reset()
    save.reset()
    setAdding(false)
    setEditing(row)
  }

  /**
   * Sparepart disimpan LEBIH DULU, dokumennya menyusul.
   *
   * Urutan itu mengikuti Pega, dan pada jalur Tambah ia satu-satunya yang mungkin: ID
   * sparepart diterbitkan server dan baru lahir setelah tersimpan, sementara dokumen
   * menuntut ID untuk ditautkan.
   *
   * Form ditutup HANYA setelah keduanya selesai. Menutupnya lebih dulu akan membuang isian
   * pengguna saat penyimpanan gagal — dan pada form berisi dua puluh isian, itu kehilangan
   * yang tidak dapat dimaafkan.
   *
   * Bila unggahan gagal SESUDAH sparepart tersimpan, form dibiarkan TERBUKA dengan pesan
   * yang menyebutkan barisnya sudah tersimpan. Menutupnya akan menyembunyikan kegagalan itu,
   * dan pengguna baru menyadarinya saat mencari dokumen yang tidak pernah ada.
   */
  function submit(values: SparepartFormValues) {
    const attach = (id: string) => {
      if (documentFile === null) {
        closeForm()
        return
      }
      upload.mutate({ id, file: documentFile, note: documentNote }, { onSuccess: closeForm })
    }

    if (editing) {
      save.mutate(
        { id: editing.id_sparepart, input: values },
        { onSuccess: () => attach(editing.id_sparepart) },
      )
      return
    }
    create.mutate(values, { onSuccess: (created) => attach(created.sparepart.id_sparepart) })
  }

  /*
    Susunan kolom mengikuti grid Pega APA ADANYA — kelimanya berdampingan pada urutan yang
    sama.

    Dua kolomnya di layar lama TIDAK punya judul sama sekali: selnya bertuliskan
    `.HARGA_JUAL` dan `.TGL_UPDATE_HARGA`, yakni nama propertinya sendiri yang bocor ke
    layar. Judulnya diambil dari caption grid Pega yang sebenarnya — "Harga (Rp)" dan
    "Tanggal Update" —
    karena nama properti yang bocor bukan tata letak yang layak ditiru, melainkan cacat.
  */
  const columns: Column<Sparepart>[] = [
    {
      key: 'id',
      title: 'ID Sparepart',
      width: '9rem',
      value: (row) => row.id_sparepart,
    },
    {
      key: 'nama',
      title: 'Nama Sparepart',
      width: '16rem',
      value: (row) => row.nama_sparepart,
    },
    {
      key: 'harga',
      title: 'Harga (Rp)',
      width: '10rem',
      alignRight: true,
      // Yang dicari dan diurutkan adalah teks aslinya, sedangkan yang dilihat pengguna
      // adalah bentuk berpemisah ribuan. Mengurutkan bentuk terformat akan menaruh
      // "1.000.000" sebelum "900.000".
      value: (row) => row.harga_jual,
      render: (row) => (
        <span className="tabular-nums text-sm text-slate-900">{rupiah(row.harga_jual)}</span>
      ),
    },
    {
      key: 'user',
      title: 'User Update',
      width: '10rem',
      value: (row) => row.user_update,
      render: (row) => (
        <span className="text-sm text-slate-900">{row.user_update || '—'}</span>
      ),
    },
    {
      key: 'tanggal',
      title: 'Tanggal Update',
      width: '12rem',
      value: (row) => row.tanggal_update_harga ?? '',
      render: (row) => (
        <span className="text-sm text-slate-900">{tanggalWIB(row.tanggal_update_harga)}</span>
      ),
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: '7rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button tone="halus" onClick={() => openEdit(row)} disabled={save.isPending}>
          Ubah
        </Button>
      ),
    },
  ]

  return (
    <main className="mx-auto max-w-7xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/* Judulnya dibaca dari `pyCaption Master Sparepart HE` pada
              Harness/SparePart_HE — "HE" ikut, karena itulah yang tertulis di layar lama
              (D-13). */}
          <h1 className="text-xl font-semibold text-slate-900">Master Sparepart HE</h1>
          <p className="text-sm text-slate-600">
            Suku cadang alat berat beserta harga jual, dimensi, dan batas stoknya.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button tone="utama" onClick={openAdd} disabled={isFormOpen}>
            Tambah
          </Button>
          {/*
            Letaknya di kepala halaman mengikuti Pega apa adanya
            (`Section/BrowseMasterSparepartHE`, offset 36628 — sejajar dengan tab).

            Ia membuka panel inline, bukan modal, karena aplikasi ini memang tidak punya
            modal: form tambah dan ubah pun digambar sebagai panel inline di bawah kepala
            halaman.
          */}
          <Button
            tone="kedua"
            onClick={() => { bukaPanel('dokumen') }}
            disabled={!uploadAvailable}
          >
            Upload Document
          </Button>
          {/*
            Tombol KEDUA layar Pega. Ia TIDAK bergantung pada `unggah_tersedia`: yang diperiksa
            penanda itu adalah layanan penyimpanan dokumen, sedangkan unggah CSV menulis
            langsung ke tabel sparepart dan tidak menyentuh layanan itu sama sekali.
          */}
          <Button tone="kedua" onClick={() => { bukaPanel('csv') }}>
            Upload Data Master Sparepart
          </Button>
          <Button tone="kedua" onClick={() => { list.refetch() }} disabled={list.isFetching}>
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
        </div>
      </header>

      {panelUnggah === 'dokumen' && (
        <section className="mt-4 rounded-kontrol border border-slate-200 bg-white p-4">
          <h2 className="text-sm font-semibold text-slate-900">Upload Document</h2>
          <p className="mb-3 mt-0.5 text-xs text-slate-600">
            Dokumen pendukung untuk <span className="font-medium">satu</span> sparepart.
          </p>
          <DocumentField
            current={documentOf.data?.data ?? null}
            available={uploadAvailable}
            file={documentFile}
            note={documentNote}
            onPick={setDocumentFile}
            onNote={setDocumentNote}
            isSaving={create.isPending || save.isPending || upload.isPending}
            error={upload.error}
            onClose={() => { setPanelUnggah(null) }}
          />
        </section>
      )}

      {panelUnggah === 'csv' && (
        <section className="mt-4 rounded-kontrol border border-slate-200 bg-white p-4">
          <h2 className="text-sm font-semibold text-slate-900">Upload Data Master Sparepart</h2>
          <p className="mb-3 mt-0.5 text-xs text-slate-600">
            Berkas CSV berisi <span className="font-medium">banyak</span> sparepart sekaligus.
            Barisnya dicocokkan menurut <span className="font-medium">nomor sparepart</span>, dan
            seluruhnya masuk antrean persetujuan.
          </p>
          <ImportCSVPanel
            isSending={importCSV.isPending}
            report={importCSV.data?.data ?? null}
            error={importCSV.error}
            onSend={(berkas) => { importCSV.mutate(berkas) }}
            onClose={() => { setPanelUnggah(null) }}
          />
        </section>
      )}

      {/*
        Kedua tombol unggah layar Pega kini lengkap, dan aturannya diturunkan dari activity
        yang ADA di export — bukan dari Flow Action pemanggilnya.

        `Flow Action/PNCUploadMasterSparepartCSV-FA` isinya hanya pemilih berkas. Yang
        menentukan perilaku ada di `Activity/PNCUploadMasterSparepart_Act`, yang terbaca utuh:

          header CSV = nama kolom tabel (pxUploadCSVResults memetakannya langsung,
            lalu @GCNM.GetPageJSONString() menserialkan halamannya apa adanya)
          kunci upsert NO_SPART  —  GCNM GetSparepartFromNoSparepart, nol penyaring tambahan
          tidak ketemu -> ID "UnknownID", diterbitkan PEGA_M_SPAREPART_HE.prc
          NAMA_SPART DIHURUFBESARKAN  —  hanya di jalur ini, tidak di jalur form
          APPROVAL := "0" tanpa syarat
          USER_UPDATE dari pengunggah, TGL_UPDATE_HARGA dari jam sistem

        Satu jebakan yang sama dengan Master Panel: yang dijalankan adalah `PropertiesValue`
        di bawah `Embed-MethodParams`; `pyExpression` di bawah `PegaGadget-ExpressionBuilder`
        hanyalah draf editor yang tertinggal.
      */}
      <nav
        aria-label="Tab Master Sparepart"
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
          <SparepartForm
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

        Tidak ada lagi kotak galat yang menggantikan tabelnya: keterangan apa pun tinggal di
        dalam grid lewat emptyMessage. Lihat emptyMessageFor untuk alasan gagal dan kosong
        tetap dibedakan kalimatnya.
      */}
      <section className="mt-6">
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(row) => row.id_sparepart}
          description="Sumber: POOLDATA.SPAREPART_HE"
          searchLabel="Cari sparepart"
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
 * rupiah menampilkan nilai harga dengan pemisah ribuan.
 *
 * Nilai yang BUKAN angka ditampilkan apa adanya, tidak dikosongkan: baris lama dapat memuat
 * apa saja di kolom itu — sistem lama tidak pernah memeriksanya — dan menyembunyikannya
 * berarti petugas tidak punya cara mengetahui bahwa isinya perlu diperbaiki.
 */
function rupiah(value: string): string {
  const clean = value.trim()
  if (clean === '') return '—'

  const number = Number(clean.replace(',', '.'))
  if (!Number.isFinite(number)) return clean

  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(number)
}

/**
 * tanggalWIB menampilkan stempel UTC dari server dalam waktu Jakarta.
 *
 * Konversi terjadi DI SINI, di satu tempat, lewat Intl — tidak ada satu pun penambahan 7 jam
 * manual (`F-5`, `08-TECHNICAL-STRATEGY.md` §4.4).
 */
function tanggalWIB(value: string | undefined): string {
  if (!value) return '—'

  const at = new Date(value)
  if (Number.isNaN(at.getTime())) return value

  return new Intl.DateTimeFormat('id-ID', {
    dateStyle: 'medium',
    timeStyle: 'short',
    timeZone: 'Asia/Jakarta',
  }).format(at)
}
