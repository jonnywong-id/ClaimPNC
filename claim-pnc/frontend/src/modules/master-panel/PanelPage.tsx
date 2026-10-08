import { useMemo, useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, PanelStatus, type Panel } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import {
  useCreatePanel,
  useImportLocationCSV,
  useImportPanelCSV,
  usePanelDocument,
  usePanelList,
  usePanelOptions,
  useSavePanel,
  useUploadPanelDocument,
} from './api'
import { DocumentField } from './DocumentField'
import { ImportCSVPanel } from './ImportCSVPanel'
import { PanelForm, type PanelFormValues } from './PanelForm'
import { compareCodeUnits } from '@/lib/sort'

/**
 * Tiga tab, sama persis dengan layar lama — termasuk URUTANNYA.
 *
 * `Section/BrowsePanelHE-Section.xml` memuat tiga section yang ketiganya membaca
 * `BrowseMasterPanel_HE_RD` yang sama dan hanya berbeda pada nilai APPROVAL-nya. Urutan
 * kemunculannya di dalam section itu, beserta caption-nya:
 *
 *	BrowsePanelHEApprove    caption "Approve"           APPROVAL="1"
 *	BrowsePanelHEReject     caption "Reject"            APPROVAL="2"
 *	BrowsePanelHEApproval   caption "Waiting Approval"  APPROVAL="0"
 *
 * Reject berada di TENGAH, bukan di ujung. Urutan itu tidak intuitif — layar master lain
 * di aplikasi ini menaruh antrean di tengah — tetapi ia yang dilihat petugas hari ini, dan
 * `D-13` menuntut tata letak yang sama supaya pengguna tidak perlu belajar ulang.
 */
/*
  Ukuran halaman BERBEDA-BEDA per tab, dan itu ditiru apa adanya.

  Ketiganya dibaca dari `pyPageSize` pada section masing-masing — atau `pyPageSizeOther`
  bila nilainya "Other":

	BrowsePanelHEApprove    pyPageSize="Other", pyPageSizeOther=15   -> 15 baris
	BrowsePanelHEReject     pyPageSize="50"                          -> 50 baris
	BrowsePanelHEApproval   pyPageSize="50"                          -> 50 baris

  Ketiganya `pyPageMode="Numeric"`, yaitu nomor halaman — bukan tombol "muat lebih
  banyak". Bentuk itu sudah dipenuhi paginator bawaan DataTable.

  Angka 15 pada tab Approve dikuatkan tangkapan layar Pega yang dikirim Work Owner:
  barisnya tepat lima belas, dari 1000106 turun sampai 1000092.

  # Kenapa tidak diseragamkan

  Perbedaan ini hampir pasti tidak disengaja — ketiga tab membaca report definition yang
  sama, dan tidak ada alasan bisnis apa pun yang membuat daftar yang disetujui layak
  dipotong lebih pendek daripada daftar yang ditolak. Ia kemungkinan besar akibat ketiga
  section dikonfigurasi sendiri-sendiri.

  Ia tetap ditiru: `D-13` menuntut tata letak yang sama, dan "menurut kami lebih rapi"
  bukan alasan yang cukup untuk menyimpang — pelajaran dari koreksi
  `keputusan-implementasi.md` §24. Penyeragamannya diangkat sebagai pertanyaan ke Work
  Owner, bukan diputuskan sendiri.
*/
const TABS = [
  {
    id: 'approve',
    label: 'Approve',
    status: PanelStatus.disetujui,
    pageSize: 15,
    description: 'Panel yang sudah disetujui dan berlaku.',
  },
  {
    id: 'reject',
    label: 'Reject',
    status: PanelStatus.ditolak,
    pageSize: 50,
    description: 'Pengajuan yang ditolak. Dapat diperbaiki lalu diajukan ulang.',
  },
  {
    id: 'menunggu',
    label: 'Waiting Approval',
    status: PanelStatus.menunggu,
    pageSize: 50,
    description:
      'Pengajuan dan perubahan yang belum diputuskan. Keputusannya diambil di layar Inbox Manager.',
  },
] as const

type TabId = (typeof TABS)[number]['id']

/** Kolom yang nilai sahnya tidak ada di export; sarannya dikumpulkan dari data. */
const SUGGESTED_COLUMNS = [
  'status_repair',
  'status_edit_quantity',
  'status_premium_repair',
  'status_pecah',
  'status_sticker',
  'status_sisi',
  'status_rusak_parah',
  'status_aktif',
  'exclusion_c',
] as const

type MessageContent = { title: string; description: string; tone: ErrorTone }

function loadMessage(error: unknown): MessageContent {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan Anda, lalu muat ulang.',
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
            'Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu.',
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
          title: 'Daftar panel tidak dapat dimuat',
          description: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
    }
  }
  return {
    title: 'Daftar panel tidak dapat dimuat',
    description: 'Coba beberapa saat lagi.',
    tone: 'gangguan',
  }
}

/**
 * Layar Master Panel.
 *
 * Pengganti `Harness/MasterPanel_HE-Harness.xml` atas tabel POOLDATA.PANEL_HE beserta
 * tabel anaknya POOLDATA.LOKASI_PANEL_HE (MENU_ID 30).
 *
 * # Apa yang dikelola layar ini
 *
 * Daftar **panel bodi kendaraan berat** beserta perlakuan klaim yang berlaku atasnya —
 * boleh diperbaiki atau tidak, boleh diubah kuantitasnya atau tidak, kena premium repair
 * atau tidak. Setiap panel punya daftar **lokasi** beserta **sisi**-nya, tersimpan di
 * tabel terpisah.
 *
 * Ia layar master pertama yang mengelola BARIS ANAK; seluruh layar master sebelumnya rata.
 *
 * # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
 *
 * Itu bukan efek samping melainkan langkah tersendiri di sistem lama:
 * `Activity/CNMUpdatePanelHE_act` menetapkan `APPROVAL := "0"` tanpa syarat apa pun.
 *
 * # Layar ini TIDAK punya persetujuan, dan itu mengikuti Pega
 *
 * Ketiga tab Master Panel di Pega hanya punya tombol **Simpan**, **Ubah**, dan **Upload
 * Document** (`pyButtonLabel` pada ketiga section), serta **nol `pySelected`** — tidak ada
 * centang, tidak ada pilihan borongan.
 *
 * Approve, Reject, Select All, dan Deselect All ada di `Section/ApprovalMasterPanelHE`,
 * yang dimuat `Harness/UserInbox_Harness`, `Section/InboxManager_Sec`, dan
 * `InboxManager_Section2` — **layar Inbox Manager, bukan layar ini**. Keputusannya
 * dijalankan `Activity/SetApprovalAllMaster` yang melayani bengkel, panel, dan sparepart
 * sekaligus lewat `Param.TIPE2`.
 *
 * Keduanya sempat digambar di sini dengan alasan "Inbox Manager belum dibangun, bentuknya
 * sama". Work Owner mencabutnya (2026-10-03): bila Pega tidak punya, layar ini pun tidak.
 * Tab Waiting Approval tetap ada — ia memang salah satu dari tiga tab Pega — tetapi ia
 * daftar baca saja.
 *
 * Endpoint `POST /api/master/panel/keputusan` tetap ada di backend untuk Inbox Manager;
 * layar ini tidak pernah memanggilnya.
 */
export function PanelPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const [tab, setTab] = useState<TabId>('approve')
  const [isAdding, setAdding] = useState(false)
  const [editing, setEditing] = useState<Panel | null>(null)

  // Berkas DITAHAN di sini, bukan dikirim saat dipilih. Pega menahannya juga —
  // `SaveFilePenunjang` hanya menaruhnya di halaman sementara, dan `CNMUpdatePanelHE_act`
  // yang menyimpannya saat panel disimpan. Lihat DocumentField.
  const [file, setFile] = useState<File | null>(null)
  const [note, setNote] = useState('')
  // Panel unggah yang sedang terbuka. Satu saja pada satu waktu — ketiganya menempati
  // tempat yang sama di bawah kepala halaman, dan membuka dua sekaligus hanya memanjangkan
  // layar tanpa menambah apa pun.
  const [panelUnggah, setPanelUnggah] = useState<'dokumen' | 'csv' | 'csv-lokasi' | null>(null)

  function bukaPanel(nama: 'dokumen' | 'csv' | 'csv-lokasi') {
    setPanelUnggah((sekarang) => (sekarang === nama ? null : nama))
  }

  const upload = useUploadPanelDocument()
  const importCSV = useImportPanelCSV()
  const importLocationCSV = useImportLocationCSV()
  const options = usePanelOptions()

  // Dokumen panel yang sedang disunting. Null pada mode Tambah: panel yang belum ada tidak
  // mungkin punya dokumen, dan menembak endpoint-nya hanya untuk menerima 404 akan mengisi
  // konsol dengan galat yang bukan galat.
  const currentDocument = usePanelDocument(editing?.id_panel ?? null)


  const active = TABS.find((t) => t.id === tab) ?? TABS[0]
  const list = usePanelList(active.status)
  const create = useCreatePanel()
  const save = useSavePanel()

  const isFormOpen = isAdding || editing !== null
  const rows = list.data?.panel ?? []

  /*
    Saran nilai untuk kolom yang daftar pilihannya tidak ada di export (R-16).

    Dikumpulkan dari baris yang SEDANG TERMUAT, bukan dari daftar yang dikarang. Ia jawaban
    terbaik yang tersedia atas pertanyaan "nilai apa yang sah di kolom ini" — lihat
    PanelForm bagian "Penanda perlakuan klaim".
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
    upload.reset()
    setAdding(false)
    setEditing(null)
    setFile(null)
    setNote('')
  }

  function openAdd() {
    create.reset()
    save.reset()
    setEditing(null)
    setAdding(true)
  }

  function openEdit(row: Panel) {
    create.reset()
    save.reset()
    setAdding(false)
    setEditing(row)
  }

  /**
   * Panel disimpan LEBIH DULU, dokumennya menyusul.
   *
   * Urutan itu mengikuti Pega, dan pada jalur Tambah ia satu-satunya yang mungkin: ID panel
   * baru lahir setelah tersimpan, sementara dokumen menuntut ID untuk ditautkan.
   *
   * Form ditutup HANYA setelah keduanya selesai. Menutupnya lebih dulu akan membuang isian
   * pengguna saat penyimpanan gagal — dan pada form yang memuat daftar lokasi yang baru
   * disusun, itu kehilangan yang tidak dapat dimaafkan.
   */
  function submit(values: PanelFormValues) {
    const attach = (id: string) => {
      if (file === null) {
        closeForm()
        return
      }
      upload.mutate(
        { id, file, note },
        {
          onSuccess: closeForm,
          // Panelnya DIBUKA KEMBALI saat unggahan gagal.
          //
          // Pesan galatnya digambar di dalam panel dokumen, dan pengguna sudah menutupnya
          // lewat Submit jauh sebelum Simpan ditekan. Tanpa membukanya kembali, kegagalan
          // yang paling berbahaya — berkas terkirim tetapi catatannya gagal — tidak
          // terlihat sama sekali.
          onError: () => { setPanelUnggah('dokumen') },
        },
      )
    }

    if (editing) {
      save.mutate(
        { id: editing.id_panel, input: values },
        { onSuccess: () => attach(editing.id_panel) },
      )
      return
    }
    create.mutate(values, { onSuccess: (created) => attach(created.panel.id_panel) })
  }

  /*
    Susunan kolom mengikuti grid Pega APA ADANYA — kesebelas kolomnya berdampingan, pada
    urutan dan dengan kata yang sama.

    Judul kolomnya dibaca dari `pyLabelFieldValue` pada
    `Section/BrowsePanelHEApproval-Section.xml`, termasuk huruf besarnya: sembilan
    bertuliskan kapital dan satu — "Exclusion C" — tidak. Ketidakseragaman itu ikut ditiru,
    karena itulah yang dibaca petugas hari ini (`D-13`).

    Nilainya ditampilkan APA ADANYA, sebagai sandi. Menggantinya dengan "Ya"/"Tidak"
    berarti menebak domain sembilan kolom yang daftar nilainya tidak ada di export
    (`R-16`) — dan menebak di tempat yang paling terlihat.

    Kolom lokasi TIDAK ada di grid Pega, dan karena itu tidak ada di sini. Daftar lokasi
    dikelola di dalam form.
  */
  const statusColumn = (
    key: string,
    title: string,
    read: (row: Panel) => string,
  ): Column<Panel> => ({
    key,
    title,
    width: '7.5rem',
    value: read,
    render: (row) => <span className="text-sm text-slate-900">{read(row) || '—'}</span>,
  })

  /*
    TIDAK ADA kolom "Pilih" di sini, dan itu mengikuti Pega.

    Ketiga tab layar Master Panel — BrowsePanelHEApprove, …Reject, dan …Approval —
    seluruhnya punya NOL `pySelected`: tidak ada centang, tidak ada pilihan borongan.
    Tombolnya pun hanya Simpan, Ubah, dan Upload Document.

    Yang punya centang beserta Approve/Reject/Select All/Deselect All adalah
    `Section/ApprovalMasterPanelHE`, dan ia dimuat `Harness/UserInbox_Harness` serta
    `Section/InboxManager_Sec` — LAYAR LAIN, modul lain. Lihat catatan pada TABS.
  */
  const columns: Column<Panel>[] = [
    {
      key: 'id',
      title: 'ID',
      width: '7rem',
      value: (row) => row.id_panel,
    },
    {
      key: 'nama',
      title: 'NAMA PANEL',
      width: '12rem',
      value: (row) => row.nama_panel,
    },
    statusColumn('repair', 'STATUS REPAIR', (row) => row.status_repair),
    statusColumn('edit_qty', 'STATUS EDIT QUANTITY', (row) => row.status_edit_quantity),
    statusColumn('premium', 'STATUS PREMIUM REPAIR', (row) => row.status_premium_repair),
    statusColumn('pecah', 'STATUS PECAH', (row) => row.status_pecah),
    statusColumn('sticker', 'STATUS STICKER', (row) => row.status_sticker),
    statusColumn('sisi', 'STATUS SISI', (row) => row.status_sisi),
    statusColumn('rusak_parah', 'STATUS RUSAK PARAH', (row) => row.status_rusak_parah),
    statusColumn('aktif', 'STATUS AKTIF', (row) => row.status_aktif),
    statusColumn('exclusion_c', 'Exclusion C', (row) => row.exclusion_c),
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
          {/* Judulnya dibaca dari `pyCaption Master Panel HE` pada Section/ListPanelHE —
              "HE" ikut, karena itulah yang tertulis di layar lama (D-13). */}
          <h1 className="text-xl font-semibold text-slate-900">Master Panel HE</h1>
          <p className="text-sm text-slate-600">
            Panel bodi kendaraan berat beserta perlakuan klaimnya, dan daftar lokasi pada
            setiap panel.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button tone="utama" onClick={openAdd} disabled={isFormOpen}>
            Tambah
          </Button>
          {/*
            Tombolnya TIDAK pernah dimatikan, dan itu mengikuti Pega: sel tombolnya di
            `Section/BrowsePanelHEApprove-Section.xml` berbunyi `pyDisabledNew = false`,
            tanpa satu pun `pyVisible`/`pyCondition`.

            Ia membuka panel inline, bukan modal, karena aplikasi ini memang tidak punya
            modal: form tambah dan ubah pun digambar sebagai panel inline. Menambahkan satu
            mekanisme dialog hanya untuk satu tombol akan menjadikannya satu-satunya di
            seluruh aplikasi.
          */}
          <Button tone="kedua" onClick={() => { bukaPanel('dokumen') }}>
            Upload Document
          </Button>
          <Button tone="kedua" onClick={() => { bukaPanel('csv') }}>
            Upload Data Master Panel
          </Button>
          <Button tone="kedua" onClick={() => { bukaPanel('csv-lokasi') }}>
            Upload Data Lokasi Panel
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
            Dokumen pendukung untuk <span className="font-medium">satu</span> panel. Berkasnya
            disimpan di layanan penyimpanan internal (GCS); yang tersimpan di basis data
            hanya catatannya.
          </p>
          <DocumentField
            current={currentDocument.data?.data ?? null}
            available={options.data?.unggah_tersedia ?? false}
            file={file}
            note={note}
            onPick={setFile}
            onNote={setNote}
            isSaving={create.isPending || save.isPending || upload.isPending}
            error={upload.error}
            onClose={() => { setPanelUnggah(null) }}
          />
        </section>
      )}

      {panelUnggah === 'csv' && (
        <section className="mt-4 rounded-kontrol border border-slate-200 bg-white p-4">
          <h2 className="text-sm font-semibold text-slate-900">Upload Data Master Panel</h2>
          <p className="mb-3 mt-0.5 text-xs text-slate-600">
            Berkas CSV berisi <span className="font-medium">banyak</span> panel sekaligus.
            Barisnya dicocokkan menurut <span className="font-medium">nama panel</span>, dan
            seluruhnya masuk antrean persetujuan.
          </p>
          <ImportCSVPanel
            isSending={importCSV.isPending}
            report={importCSV.data?.data ?? null}
            error={importCSV.error}
            onSend={(berkas) => { importCSV.mutate(berkas) }}
            onClose={() => { setPanelUnggah(null) }}
            rowLabel="Nama Panel"
          />
        </section>
      )}

      {panelUnggah === 'csv-lokasi' && (
        <section className="mt-4 rounded-kontrol border border-slate-200 bg-white p-4">
          <h2 className="text-sm font-semibold text-slate-900">Upload Data Lokasi Panel</h2>
          <p className="mb-3 mt-0.5 text-xs text-slate-600">
            Berkas CSV berisi daftar lokasi. Barisnya{' '}
            <span className="font-medium">ditambahkan</span> ke panel yang disebut — lokasi
            lain panel itu tidak dihapus, dan yang sudah ada tidak digandakan.
          </p>
          <ImportCSVPanel
            isSending={importLocationCSV.isPending}
            report={importLocationCSV.data?.data ?? null}
            error={importLocationCSV.error}
            onSend={(berkas) => { importLocationCSV.mutate(berkas) }}
            onClose={() => { setPanelUnggah(null) }}
            rowLabel="Panel"
          />
        </section>
      )}

      {/*
        Ketiga tombol unggah layar Pega kini lengkap, dan aturannya diturunkan dari activity
        yang ADA di export — bukan dari Flow Action pemanggilnya.

        Kedua Flow Action CSV (`PNCUploadMasterPanelCSV`, `PNCUploadLokasiPanelCSV`) memang
        hilang, tetapi isinya hanya pemilih berkas. Yang menentukan perilaku ada di
        `Activity/PNCUploadMasterPanel_Act` dan `Activity/PNCUploadLokasiSisiPanel_Act`,
        keduanya terbaca utuh:

          header CSV = nama kolom tabel (pxUploadCSVResults memetakannya langsung)
          nilainya KATA, bukan sandi  —  "TIDAK"→0, "GANTI"→1, "JASA"→2, lainnya→3
          NAME dihurufbesarkan, STS_AKTIF dipaksa "1" dan tidak dibaca dari berkas
          APPROVAL := "0" tanpa syarat pada KEDUA jalur
          lokasi memakai LOKASI(<APPEND>) — menambah, bukan mengganti

        Satu jebakan yang nyaris menyesatkan: berkas activity master memuat DUA versi rule
        (01-01-91 dan 01-01-89) beserta dua ekspresi yang bertentangan. Yang dijalankan
        adalah `PropertiesValue` di bawah `Embed-MethodParams`; `pyExpression` di bawah
        `PegaGadget-ExpressionBuilder` hanyalah draf editor yang tertinggal.
      */}
      <nav
        aria-label="Tab Master Panel"
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
          <PanelForm
            editing={editing}
            knownValues={knownValues}
            isSaving={create.isPending || save.isPending || upload.isPending}
            error={editing ? save.error : create.error}
            onSave={submit}
            onCancel={closeForm}
          />
        </section>
      )}

      <section className="mt-6">
        {portal === null ? (
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        ) : list.isPending ? (
          <p className="text-sm text-slate-500">Memuat daftar panel…</p>
        ) : list.isError ? (
          (() => {
            const message = loadMessage(list.error)
            return (
              <ErrorMessage
                title={message.title}
                description={message.description}
                tone={message.tone}
              />
            )
          })()
        ) : (
          <DataTable
            columns={columns}
            rows={rows}
            rowKey={(row) => row.id_panel}
            description="Sumber: POOLDATA.PANEL_HE dan POOLDATA.LOKASI_PANEL_HE"
            searchLabel="Cari panel"
            emptyMessage={`Belum ada panel pada tab ${active.label}.`}
            pageSize={active.pageSize}
            /*
              Kepala kolom tetap digambar meski tidak ada satu pun baris.

              Bukan pilihan tampilan: `pyHideGridHeaderWhenNoRows` bernilai **false** pada
              ketiga section tab (BrowsePanelHEApprove, …Reject, …Approval), dan Pega
              menaruh `pyGridNoResultsMessage` di bawah kepala kolomnya — bukan
              menggantinya dengan kotak kosong.

              Paling terasa di tab Waiting Approval, yang sering kosong: tanpa kepala
              kolom, tab itu tidak memberi tahu apa pun tentang bentuk datanya.
            */
            showHeaderWhenEmpty
          />
        )}
      </section>
    </main>
  )
}

/*
  Ringkasan lokasi per baris TIDAK ada di sini, dan itu disengaja.

  Grid Pega tidak punya kolom lokasi sama sekali — kesebelas kolomnya seluruhnya milik
  tabel induk. Daftar lokasi hanya muncul saat sebuah panel dibuka, dan di sanalah ia
  dikelola (lihat LocationEditor).
*/

/*
  TIDAK ADA bilah keputusan di layar ini, dan itu mengikuti Pega.

  Ketiga tab Master Panel hanya punya tombol Simpan, Ubah, dan Upload Document
  (`pyButtonLabel` pada ketiga section), serta NOL `pySelected` — tidak ada centang dan
  tidak ada Approve/Reject.

  Approve, Reject, Select All, dan Deselect All ada di `Section/ApprovalMasterPanelHE`,
  yang dimuat `Harness/UserInbox_Harness`, `Section/InboxManager_Sec`, dan
  `InboxManager_Section2` — layar Inbox Manager, bukan layar ini.

  Sempat digambar di sini dengan alasan "Inbox Manager belum dibangun, bentuknya sama".
  Work Owner mencabutnya (2026-10-03): bila Pega tidak punya, layar ini pun tidak.
  Keputusannya di docs/keputusan-implementasi.md.
*/
