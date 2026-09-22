import { useMemo, useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, PanelStatus, type Panel } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import { useCreatePanel, useDecidePanel, usePanelList, useSavePanel } from './api'
import { PanelForm, type PanelFormValues } from './PanelForm'

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
      'Pengajuan dan perubahan yang belum diputuskan. Centang barisnya untuk menyetujui atau menolak.',
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
 * # Kenapa Approve dan Reject ada DI SINI, bukan di Inbox Manager
 *
 * Di Pega keduanya ada di layar lain: `Section/ApprovalMasterPanelHE` dipakai Inbox
 * Manager, dan keputusannya dijalankan `Activity/SetApprovalAllMaster` yang melayani
 * bengkel, panel, dan sparepart sekaligus.
 *
 * Inbox Manager belum dibangun. Menunda keputusannya sampai layar itu ada berarti setiap
 * panel yang ditambah tertahan di Waiting Approval tanpa satu pun cara menyelesaikannya —
 * dan alur ini tidak dapat dicoba sama sekali. Yang dipakai sebagai gantinya adalah BENTUK
 * yang sama persis: centang beberapa baris, satu catatan, lalu satu tombol untuk seluruh
 * pilihan. Memindahkannya ke Inbox Manager kelak hanya soal letak, bukan soal perilaku.
 */
export function PanelPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const [tab, setTab] = useState<TabId>('approve')
  const [isAdding, setAdding] = useState(false)
  const [editing, setEditing] = useState<Panel | null>(null)
  const [chosen, setChosen] = useState<Set<string>>(new Set())
  const [note, setNote] = useState('')

  const active = TABS.find((t) => t.id === tab) ?? TABS[0]
  const list = usePanelList(active.status)
  const create = useCreatePanel()
  const save = useSavePanel()
  const decide = useDecidePanel()

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
      collected[column] = [...unique].sort()
    }
    return collected
  }, [rows])

  function closeForm() {
    create.reset()
    save.reset()
    setAdding(false)
    setEditing(null)
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

  function toggle(id: string) {
    setChosen((current) => {
      const next = new Set(current)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function submit(values: PanelFormValues) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal — dan pada form yang memuat daftar
    // lokasi yang baru disusun, itu kehilangan yang tidak dapat dimaafkan.
    if (editing) {
      save.mutate({ id: editing.id_panel, input: values }, { onSuccess: closeForm })
      return
    }
    create.mutate(values, { onSuccess: closeForm })
  }

  function runDecision(status: string) {
    decide.mutate(
      { id_panel: [...chosen], status, catatan: note },
      {
        onSuccess: () => {
          setChosen(new Set())
          setNote('')
        },
      },
    )
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

  const columns: Column<Panel>[] = [
    ...(tab === 'menunggu'
      ? [
          {
            key: 'pilih',
            title: 'Pilih',
            width: '4.5rem',
            noSort: true,
            value: (row: Panel) => (chosen.has(row.id_panel) ? 'dipilih' : ''),
            render: (row: Panel) => (
              <label className="inline-flex items-center gap-2">
                <input
                  type="checkbox"
                  className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500/50"
                  checked={chosen.has(row.id_panel)}
                  disabled={decide.isPending}
                  onChange={() => toggle(row.id_panel)}
                />
                <span className="sr-only">Pilih {row.nama_panel}</span>
              </label>
            ),
          } satisfies Column<Panel>,
        ]
      : []),
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
          <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
        </div>
      </header>

      {/*
        Ketiga tombol unggah layar Pega — "Upload Document", "Upload Data Master Panel",
        dan "Upload Data Lokasi Panel" (`pyButtonLabel` pada Section/BrowsePanelHE) —
        SENGAJA TIDAK DIGAMBAR di sini.

        Ketiganya memanggil local action `UploadDocument`, `PNCUploadMasterPanelCSV`, dan
        `PNCUploadLokasiPanelCSV`; tidak satu pun ada di export (`R-16`), sehingga susunan
        kolom CSV-nya, validasinya, dan — yang paling menentukan — apakah baris hasil
        unggah masuk antrean persetujuan, seluruhnya tidak diketahui.

        Sempat digambar dalam keadaan mati supaya ketiadaannya terbaca dari layar. Work
        Owner memilih menghapusnya sama sekali (2026-09-20); lihat
        docs/keputusan-implementasi.md §25.
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
              // Centang dan catatan dibuang saat berpindah tab: baris yang dipilih milik
              // tab sebelumnya, dan menyimpannya berarti keputusan dapat mengenai baris
              // yang tidak sedang dilihat siapa pun.
              setChosen(new Set())
              setNote('')
              decide.reset()
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

      {decide.isError && (
        <div className="mt-4">
          <ErrorMessage
            title="Keputusan belum tersimpan"
            description={
              decide.error instanceof APIError
                ? decide.error.message
                : 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.'
            }
            tone="gangguan"
          />
        </div>
      )}

      {decide.isSuccess && decide.data && (
        <p className="mt-4 rounded-kontrol border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-900">
          {decide.data.jumlah_berubah} panel dipindahkan ke{' '}
          <span className="font-medium">{decide.data.status_label}</span>.
        </p>
      )}

      {tab === 'menunggu' && (
        <DecisionBar
          count={chosen.size}
          note={note}
          onNoteChange={setNote}
          isBusy={decide.isPending}
          onApprove={() => runDecision(PanelStatus.disetujui)}
          onReject={() => runDecision(PanelStatus.ditolak)}
          onClear={() => setChosen(new Set())}
        />
      )}

      {isFormOpen && (
        <section className="mt-5">
          <PanelForm
            editing={editing}
            knownValues={knownValues}
            isSaving={create.isPending || save.isPending}
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

/**
 * DecisionBar adalah tombol Approve dan Reject untuk seluruh baris yang dicentang,
 * beserta satu isian Catatan.
 *
 * Ia padanan `Section/ApprovalMasterPanelHE-Section.xml` yang menyediakan Select All,
 * Deselect All, Approve, dan Reject, ditambah isian `TempStsClaim.pyNote` yang sejajar
 * dengan kolom ALASAN_TOLAK.
 *
 * Tombolnya mati selama belum ada yang dicentang — bukan disembunyikan. Tombol yang hilang
 * membuat pengguna mencari fiturnya; tombol yang mati menunjukkan apa yang harus dilakukan
 * lebih dulu.
 */
function DecisionBar({
  count,
  note,
  onNoteChange,
  isBusy,
  onApprove,
  onReject,
  onClear,
}: {
  count: number
  note: string
  onNoteChange: (value: string) => void
  isBusy: boolean
  onApprove: () => void
  onReject: () => void
  onClear: () => void
}) {
  return (
    <div className="mt-4 space-y-3 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <div className="flex flex-wrap items-center gap-2">
        <span className="min-w-0 flex-1 text-sm text-slate-700">
          {count === 0 ? (
            'Centang panel yang akan diputuskan.'
          ) : (
            <>
              <span className="font-medium">{count} panel</span> dipilih.
            </>
          )}
        </span>
        {count > 0 && (
          <Button tone="halus" onClick={onClear} disabled={isBusy}>
            Bersihkan
          </Button>
        )}
        {/*
          Namanya "Approve terpilih", bukan "Approve" saja.

          Bukan sekadar demi kejelasan kalimat: tab di atasnya juga bernama "Approve" dan
          "Reject" — caption Pega yang memang harus ditiru (D-13) — sehingga tombol bernama
          sama membuat dua kontrol yang sama sekali berbeda tidak dapat dibedakan dari
          namanya. Pembaca layar mengumumkan keduanya dengan kata yang sama persis.
        */}
        <Button tone="utama" onClick={onApprove} disabled={isBusy || count === 0}>
          {isBusy ? 'Menyimpan…' : 'Approve terpilih'}
        </Button>
        <Button tone="kedua" onClick={onReject} disabled={isBusy || count === 0}>
          Reject terpilih
        </Button>
      </div>

      {/*
        Catatan hanya tersimpan pada keputusan TOLAK; kolomnya memang bernama ALASAN_TOLAK.
        Itu dinyatakan di layar, bukan dibiarkan menjadi kejutan saat petugas mengetiknya
        lalu menekan Approve.
      */}
      <div>
        <label htmlFor="catatan-keputusan" className="block text-sm font-medium text-slate-700">
          Catatan
        </label>
        <input
          id="catatan-keputusan"
          type="text"
          value={note}
          maxLength={250}
          disabled={isBusy || count === 0}
          onChange={(event) => onNoteChange(event.target.value)}
          className={
            'mt-1 w-full rounded-kontrol border border-slate-300 bg-white px-3 py-2 text-slate-900 ' +
            'shadow-lembut transition-[border-color,box-shadow] duration-150 ease-halus ' +
            'focus:border-blue-500 focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/20 ' +
            'disabled:bg-slate-100 disabled:text-slate-500'
          }
        />
        <p className="mt-1 text-xs text-slate-500">
          Tersimpan sebagai alasan penolakan. Pada keputusan Approve, catatan ini tidak ikut
          tersimpan.
        </p>
      </div>
    </div>
  )
}
