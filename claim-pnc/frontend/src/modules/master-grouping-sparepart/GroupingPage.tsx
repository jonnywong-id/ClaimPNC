import { GroupingStatus, type Grouping } from '@/api/types'
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

import { useCreateGrouping, useDecideGrouping, useGroupingList, useSaveGrouping } from './api'
import { GroupingForm } from './GroupingForm'

/**
 * Tiga tab, sama persis dengan layar lama — termasuk URUTANNYA.
 *
 * `Section/PNCMasterGroupingSparepartHE-Section.xml` memuat tiga section yang ketiganya
 * membaca `GetDataMasterGrouping` yang sama dan hanya berbeda pada nilai APPROVAL-nya. Urutan
 * kemunculannya di dalam section itu:
 *
 *	MasterGroupingSparepartHEApprove   APPROVAL="1"
 *	MasterGroupingSparepartHEReject    APPROVAL="2"
 *	MasterGroupingSparepartHEApproval  APPROVAL="0"
 *
 * Reject berada di TENGAH, bukan di ujung — sama seperti Master Sparepart, Master Panel, dan
 * Master Bengkel. Urutan itu tidak intuitif, tetapi ia yang dilihat petugas hari ini, dan
 * `D-13` menuntut tata letak yang sama supaya pengguna tidak perlu belajar ulang.
 */
const TABS = [
  {
    id: 'approve',
    label: 'Approve',
    status: GroupingStatus.disetujui,
    description: 'Grouping yang sudah disetujui dan berlaku.',
  },
  {
    id: 'reject',
    label: 'Reject',
    status: GroupingStatus.ditolak,
    description: 'Pengajuan yang ditolak. Dapat diperbaiki lalu diajukan ulang.',
  },
  {
    id: 'menunggu',
    label: 'Waiting Approval',
    status: GroupingStatus.menunggu,
    description:
      'Pengajuan dan perubahan yang belum diputuskan. Centang barisnya untuk menyetujui atau menolak.',
  },
] as const

/**
 * Ukuran halaman diambil dari `pyPageSizeOther` pada ketiga section tab.
 *
 * Ketiganya bernilai **15** — berbeda dari Master Sparepart yang 30, Master Bengkel yang 20,
 * dan Master Panel yang 15 pada satu tab dan 50 pada dua tab lainnya. Tidak ada satu angka
 * yang benar untuk seluruh layar; angkanya milik layar, bukan milik komponen tabel.
 */
const PAGE_SIZE = 15

function loadMessage(error: unknown): MessageContent {
  return retryLoadMessage(error, { failedTitle: 'Daftar grouping tidak dapat dimuat' })
}

/**
 * Layar Master Grouping Sparepart.
 *
 * Pengganti `Harness/GroupingSparePart_HE-Harness.xml` atas
 * POOLDATA.SPAREPART_HE_VIN_KEY beserta pendampingnya SPAREPART_HE_VIN_GROUP (MENU_ID 32).
 *
 * # Apa yang dikelola layar ini
 *
 * Penautan sebuah **suku cadang alat berat** ke sebuah **panel bodi** pada sebuah
 * **kendaraan**. Baris yang menunjuk kendaraan yang sama dikumpulkan di bawah satu **Nomor
 * Grup** — itulah arti kata "grouping" pada namanya, bukan penggolongan suku cadang.
 *
 * # Enam kolom, bukan sebelas
 *
 * Grid Pega hanya menampilkan ID, No Sparepart, Nama Sparepart, Nama Panel, Sisi Panel, dan
 * No Rangka; kelima isian sisanya hanya terlihat saat sebuah baris dibuka. Susunan itu ditiru
 * apa adanya, termasuk tidak menambahkan kolom yang menurut kami berguna.
 *
 * Satu kolom DITAMBAHKAN: **Nomor Grup**. Ia tidak digambar di layar lama sama sekali, dan
 * tanpa itu pengguna tidak punya satu pun cara melihat baris mana yang tergabung dengan baris
 * mana — padahal itulah seluruh gunanya layar ini. Selisih yang dicatat, bukan disembunyikan.
 *
 * # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
 *
 * Di sistem lama status yang disimpan datang dari pemanggil (`Param.Approval`), dan layar
 * penyuntingannya selalu mengirim "0".
 *
 * # Kenapa Approve dan Reject ada DI SINI, bukan di Inbox Manager
 *
 * Di Pega keduanya ada di layar lain: `Section/ApprovalPNCMasterGroupingSparepartHE` dipakai
 * Inbox Manager. Inbox Manager belum dibangun, dan menunda keputusannya sampai layar itu ada
 * berarti setiap grouping yang ditambah tertahan di Waiting Approval tanpa satu pun cara
 * menyelesaikannya. Yang dipakai sebagai gantinya adalah BENTUK yang sama persis: centang
 * beberapa baris, lalu satu tombol untuk seluruh pilihan — sama dengan ketiga master alat
 * berat lain.
 *
 * # Tanpa isian Catatan pada keputusan
 *
 * Kedua tabel modul ini tidak punya kolom penampung alasan penolakan, dan layar persetujuan
 * Pega pun tidak punya isian catatan. Menggambar isian yang diam-diam membuang isinya lebih
 * buruk daripada tidak menggambarnya.
 */
export function GroupingPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const { tab, setTab, active, chosen, setChosen, toggle } = useApprovalTabs(TABS, 'approve')

  const list = useGroupingList(active.status)
  const create = useCreateGrouping()
  const save = useSaveGrouping()
  const form = useCrudForm(create, save, (row: Grouping, input) => ({ id: row.id_grouping, input }))
  const editing = form.openedRow
  const isFormOpen = form.isOpen
  const { closeForm, openCreate: openAdd, openEdit } = form
  const decide = useDecideGrouping()

  const rows = list.data?.grouping ?? []

  function runDecision(status: string) {
    decide.mutate({ id_grouping: [...chosen], status }, { onSuccess: () => setChosen(new Set()) })
  }

  /*
    Susunan kolom mengikuti grid Pega APA ADANYA — keenamnya berdampingan pada urutan yang
    sama: ID, No Sparepart, Nama Sparepart, Nama Panel, Sisi Panel, No Rangka.

    Judulnya pun diambil dari grid, bukan dari form: grid menulis "No Sparepart" dan "Sisi
    Panel" sementara form menulis "Nomor Sparepart" dan "Sisi". Keduanya dipertahankan di
    tempatnya masing-masing (D-13).
  */
  const columns: Column<Grouping>[] = [
    ...selectColumn<Grouping>({
      enabled: tab === 'menunggu',
      chosen,
      idOf: (row) => row.id_grouping,
      nameOf: (row) => `grouping ${row.nomor_sparepart} pada ${row.nama_panel}`,
      disabled: decide.isPending,
      onToggle: toggle,
    }),
    {
      key: 'id',
      title: 'ID',
      width: '6rem',
      value: (row) => row.id_grouping,
    },
    {
      key: 'nomor',
      title: 'No Sparepart',
      width: '10rem',
      value: (row) => row.nomor_sparepart,
    },
    {
      key: 'nama',
      title: 'Nama Sparepart',
      width: '14rem',
      value: (row) => row.nama_sparepart,
      render: (row) => (
        <span className="text-sm text-slate-900">{row.nama_sparepart || '—'}</span>
      ),
    },
    {
      key: 'panel',
      title: 'Nama Panel',
      width: '12rem',
      value: (row) => row.nama_panel,
    },
    {
      key: 'sisi',
      title: 'Sisi Panel',
      width: '7rem',
      // Yang dicari dan diurutkan adalah sebutannya, bukan sandinya: pengguna mencari "KIRI",
      // bukan "1".
      value: (row) => row.sisi_label,
      render: (row) => <span className="text-sm text-slate-900">{row.sisi_label || '—'}</span>,
    },
    {
      key: 'rangka',
      title: 'No Rangka',
      width: '13rem',
      value: (row) => row.no_rangka,
      render: (row) => (
        <span className="font-mono text-xs text-slate-900">{row.no_rangka || '—'}</span>
      ),
    },
    {
      // DITAMBAHKAN terhadap grid Pega; lihat catatan pada GroupingPage.
      key: 'grup',
      title: 'Nomor Grup',
      width: '8rem',
      value: (row) => row.nomor_grup,
      render: (row) => (
        <span className="tabular-nums text-sm text-slate-900">{row.nomor_grup || '—'}</span>
      ),
    },
    editColumn<Grouping>({
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
      loadingText: 'Memuat daftar grouping…',
      toMessage: loadMessage,
      render: () => (
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(row) => row.id_grouping}
          description="Sumber: POOLDATA.SPAREPART_HE_VIN_KEY + SPAREPART_HE_VIN_GROUP"
          searchLabel="Cari grouping"
          pageSize={PAGE_SIZE}
          emptyMessage={`Belum ada grouping pada tab ${active.label}.`}
        />
      ),
    })
  }

  return (
    <main className="mx-auto max-w-7xl px-4 py-8">
      {/* Judulnya dibaca dari Section/MasterGroupingSparepartHE — "HE" ikut, karena itulah
          yang tertulis di layar lama (D-13).

          Kedua tombolnya diambil dari `pyLabel` pada Section/MasterGroupingSparepartHE:
          "Tambah" dan "Refresh". */}
      <ListHeader
        title="Master Grouping Sparepart HE"
        description="Penautan suku cadang ke panel bodi pada sebuah kendaraan, dikelompokkan menurut nomor rangka."
      >
        <AddButton onClick={openAdd} disabled={isFormOpen} />
        <RefreshButton query={list} />
      </ListHeader>

      <ApprovalPanel
        tabLabel="Tab Master Grouping Sparepart"
        tabs={TABS}
        active={active}
        onSwitch={setTab}
        onBeforeSwitch={closeForm}
        portal={list.data?.portal ?? portal}
        decide={decide}
        feedbackNoun="grouping"
        barNoun="grouping"
        chosenCount={chosen.size}
        onApprove={() => runDecision(GroupingStatus.disetujui)}
        onReject={() => runDecision(GroupingStatus.ditolak)}
        onClear={() => setChosen(new Set())}
      />

      {isFormOpen && (
        <section className="mt-5">
          <GroupingForm
            editing={editing}
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

