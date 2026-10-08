import { useState, useMemo } from 'react'

import { PanelStatus, type Panel } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { ApprovalPanel } from '@/components/masterpage/ApprovalPanel'
import { retryLoadMessage, type MessageContent } from '@/components/masterpage/loadMessage'
import { useApprovalTabs } from '@/components/masterpage/useApprovalTabs'
import { useCrudForm } from '@/components/masterpage/useCrudForm'
import {
  AddButton,
  ListHeader,
  RefreshButton,
  editColumn,
  renderListState,
} from '@/components/masterpage/MasterPage'
import { DataTable, type Column } from '@/components/DataTable'
import { selectColumn } from '@/components/ApprovalControls'

import { useCreatePanel, useDecidePanel, usePanelList, useSavePanel } from './api'
import { PanelForm } from './PanelForm'
import { collectKnownValues } from '@/components/masterpage/knownValues'

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

function loadMessage(error: unknown): MessageContent {
  return retryLoadMessage(error, { failedTitle: 'Daftar panel tidak dapat dimuat' })
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

  const { tab, setTab, active, chosen, setChosen, toggle } = useApprovalTabs(TABS, 'approve')
  const [note, setNote] = useState('')

  const list = usePanelList(active.status)
  const create = useCreatePanel()
  const save = useSavePanel()
  const form = useCrudForm(create, save, (row: Panel, input) => ({ id: row.id_panel, input }))
  const editing = form.openedRow
  const isFormOpen = form.isOpen
  const { closeForm, openCreate: openAdd, openEdit } = form
  const decide = useDecidePanel()

  const rows = list.data?.panel ?? []

  /*
    Saran nilai untuk kolom yang daftar pilihannya tidak ada di export (R-16).

    Dikumpulkan dari baris yang SEDANG TERMUAT, bukan dari daftar yang dikarang. Ia jawaban
    terbaik yang tersedia atas pertanyaan "nilai apa yang sah di kolom ini" — lihat
    PanelForm bagian "Penanda perlakuan klaim".
  */
  const knownValues = useMemo(() => collectKnownValues(rows, SUGGESTED_COLUMNS), [rows])

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
    ...selectColumn<Panel>({
      enabled: tab === 'menunggu',
      chosen,
      idOf: (row) => row.id_panel,
      nameOf: (row) => row.nama_panel,
      disabled: decide.isPending,
      onToggle: toggle,
    }),
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
    editColumn<Panel>({
      onEdit: openEdit,
      width: '7rem',
      tone: 'halus',
      disabled: save.isPending,
    }),
  ]

  // Isi bagian daftar menurut keadaan portal dan kueri.
  function renderList() {
    return renderListState({
      portal,
      query: list,
      loadingText: 'Memuat daftar panel…',
      toMessage: loadMessage,
      render: () => (
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(row) => row.id_panel}
          description="Sumber: POOLDATA.PANEL_HE dan POOLDATA.LOKASI_PANEL_HE"
          searchLabel="Cari panel"
          emptyMessage={`Belum ada panel pada tab ${active.label}.`}
          pageSize={active.pageSize}
        />
      ),
    })
  }

  return (
    <main className="mx-auto max-w-7xl px-4 py-8">
      {/* Judulnya dibaca dari `pyCaption Master Panel HE` pada Section/ListPanelHE —
          "HE" ikut, karena itulah yang tertulis di layar lama (D-13). */}
      <ListHeader
        title="Master Panel HE"
        description="Panel bodi kendaraan berat beserta perlakuan klaimnya, dan daftar lokasi pada setiap panel."
      >
        <AddButton onClick={openAdd} disabled={isFormOpen} />
        <RefreshButton query={list} />
      </ListHeader>

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
      <ApprovalPanel
        tabLabel="Tab Master Panel"
        tabs={TABS}
        active={active}
        onSwitch={setTab}
        onBeforeSwitch={() => {
          closeForm()
          setNote('')
        }}
        portal={list.data?.portal ?? portal}
        decide={decide}
        feedbackNoun="panel"
        barNoun="panel"
        chosenCount={chosen.size}
        onApprove={() => runDecision(PanelStatus.disetujui)}
        onReject={() => runDecision(PanelStatus.ditolak)}
        onClear={() => setChosen(new Set())}
        note={{ value: note, onChange: setNote }}
      />

      {isFormOpen && (
        <section className="mt-5">
          <PanelForm
            editing={editing}
            knownValues={knownValues}
            isSaving={form.isSaving}
            error={form.saveError}
            onSave={form.submit}
            onCancel={closeForm}
          />
        </section>
      )}

      <section className="mt-6">
        {renderList()}
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

