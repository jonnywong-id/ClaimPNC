import { PartCategoryStatus, type PartCategory } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { ApprovalPanel } from '@/components/masterpage/ApprovalPanel'
import { reloadLoadMessage, type MessageContent } from '@/components/masterpage/loadMessage'
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

import {
  useCreatePartCategory,
  useDecidePartCategory,
  usePartCategoryList,
  useSavePartCategory,
} from './api'
import { PartCategoryForm } from './PartCategoryForm'

/**
 * Tiga tab, sama persis dengan layar lama — termasuk URUTANNYA.
 *
 * `Section/MasterKategoriSparepartHE-Section.xml` merujuk tiga section yang ketiganya
 * membaca tabel yang sama dan hanya berbeda pada nilai APPROVAL-nya. Urutan kemunculannya
 * di dalam section itu:
 *
 *	MasterKategoriSparepartHEApprove   APPROVAL="1"
 *	MasterKategoriSparepartHEReject    APPROVAL="2"
 *	MasterKategoriSparepartHEApproval  APPROVAL="0"
 *
 * Reject berada di TENGAH, bukan di ujung — sama seperti Master Sparepart, Master Panel,
 * dan Master Bengkel. Urutan itu tidak intuitif, tetapi ia yang dilihat petugas hari ini,
 * dan `D-13` menuntut tata letak yang sama supaya pengguna tidak perlu belajar ulang.
 */
const TABS = [
  {
    id: 'approve',
    label: 'Approve',
    status: PartCategoryStatus.disetujui,
    description:
      'Kategori yang sudah disetujui. Hanya yang ada di sini yang muncul sebagai pilihan ' +
      'di layar Master Sparepart dan Master Tipe Sparepart.',
  },
  {
    id: 'reject',
    label: 'Reject',
    status: PartCategoryStatus.ditolak,
    description:
      'Kategori yang ditolak. Namanya tetap terpakai — nama yang sama tidak dapat ' +
      'digunakan kategori lain sampai baris ini diubah namanya.',
  },
  {
    id: 'menunggu',
    label: 'Waiting Approval',
    status: PartCategoryStatus.menunggu,
    description: 'Kategori yang menunggu diputuskan.',
  },
] as const

/**
 * Ukuran halaman diambil dari `pyPageSize` pada ketiga section tab layar ini.
 *
 * Ketiganya bernilai **50** — berbeda dari Master Sparepart yang 30 dan Master Bengkel yang
 * 20. Tidak ada satu angka yang benar untuk seluruh layar; angkanya milik layar, bukan
 * milik komponen tabel.
 */
const PAGE_SIZE = 50

/** Mengubah galat pemuatan daftar menjadi pesan yang dapat ditindaklanjuti. */
function loadMessage(error: unknown): MessageContent {
  return reloadLoadMessage(error, { failedTitle: 'Daftar kategori sparepart tidak dapat dimuat' })
}

/**
 * Layar Master Kategori Sparepart.
 *
 * Pengganti `Harness/GCNMCatSparepart-Harness.xml` atas tabel
 * POOLDATA.GCNM_M_SPAREPART_CATEGORY (MENU_ID 33).
 *
 * # Apa yang dikelola layar ini
 *
 * **Penggolongan suku cadang alat berat** — satu baris per kategori, dan setiap kategori
 * hanya punya nama. Ia tabel acuan yang menyuapi dua layar lain: Master Sparepart memakai
 * ID-nya pada kolom KATEGORI_SPART, dan Master Tipe Sparepart menjadikannya induk setiap
 * tipe.
 *
 * # Dua kolom, dan memang hanya itu yang ada
 *
 * Grid Pega menampilkan "ID Kategori Sparepart" dan "Nama Kategori Sparepart", dan tabelnya
 * memang hanya punya tiga kolom — yang ketiga adalah status, yang sudah menjadi tab. Tidak
 * ada kolom "User Update" maupun "Tanggal Update" seperti pada Master Sparepart: tabel ini
 * tidak punya kolomnya, sehingga siapa yang mengubah sebuah kategori TIDAK tersimpan di
 * mana pun.
 *
 * # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
 *
 * Itu bukan efek samping melainkan langkah tersendiri di sistem lama:
 * `Activity/UpdateKategoriSparepart_act2` menetapkan APPROVAL := "0" tanpa syarat apa pun.
 * Akibatnya menyentuh layar LAIN — kategori yang sedang menunggu hilang dari dropdown
 * Master Sparepart — dan itu dinyatakan di muka pada form, bukan dibiarkan ditemukan.
 *
 * # Kenapa Approve dan Reject ada DI SINI, bukan di Inbox Manager
 *
 * Di Pega keduanya ada di layar lain: `Section/ApprovalMasterKategoriSparepartHE` dipakai
 * Inbox Manager. Inbox Manager belum dibangun, dan menunda keputusannya sampai layar itu ada
 * berarti setiap kategori yang ditambah tertahan di Waiting Approval tanpa satu pun cara
 * menyelesaikannya — dan selama tertahan, ia tidak dapat dipakai sparepart mana pun.
 *
 * Yang dipakai sebagai gantinya adalah BENTUK yang sama persis: centang beberapa baris,
 * lalu satu tombol untuk seluruh pilihan. Perlakuannya sama dengan Master Bengkel, Master
 * Panel, dan Master Sparepart. Keputusan Work Owner 2026-09-21.
 *
 * # Tanpa isian Catatan
 *
 * `POOLDATA.GCNM_M_SPAREPART_CATEGORY` tidak punya kolom penampung alasan penolakan.
 * Menggambar isian yang diam-diam membuang isinya lebih buruk daripada tidak
 * menggambarnya — perlakuan yang sama dengan Master Sparepart.
 */
export function PartCategoryPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const { tab, setTab, active, chosen, setChosen, toggle } = useApprovalTabs(TABS, 'approve')

  const list = usePartCategoryList(active.status)
  const create = useCreatePartCategory()
  const save = useSavePartCategory()
  const form = useCrudForm(create, save, (row: PartCategory, input) => ({
    id: row.id_kategori_sparepart,
    input,
  }))
  const editing = form.openedRow
  const isFormOpen = form.isOpen
  const { closeForm, openCreate: openAdd, openEdit } = form
  const decide = useDecidePartCategory()

  const rows = list.data?.kategori_sparepart ?? []

  function runDecision(status: string) {
    decide.mutate(
      { id_kategori_sparepart: [...chosen], status },
      { onSuccess: () => setChosen(new Set()) },
    )
  }

  /*
    Dua kolom, mengikuti grid Pega apa adanya, ditambah kolom aksi dan — pada tab Waiting
    Approval — kolom centang.

    Tidak ada kolom status: ia sudah menjadi tab, dan menggambarnya lagi di setiap baris
    hanya mengulang hal yang sama di seluruh halaman.
  */
  const columns: Column<PartCategory>[] = [
    ...selectColumn<PartCategory>({
      enabled: tab === 'menunggu',
      chosen,
      idOf: (row) => row.id_kategori_sparepart,
      nameOf: (row) => row.nama_kategori_sparepart,
      disabled: decide.isPending,
      onToggle: toggle,
    }),
    {
      key: 'id',
      title: 'ID Kategori Sparepart',
      width: '12rem',
      value: (row) => row.id_kategori_sparepart,
    },
    {
      key: 'nama',
      title: 'Nama Kategori Sparepart',
      value: (row) => row.nama_kategori_sparepart,
    },
    editColumn<PartCategory>({
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
      loadingText: 'Memuat daftar kategori sparepart…',
      toMessage: loadMessage,
      render: () => (
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(row) => row.id_kategori_sparepart}
          description="Sumber: POOLDATA.GCNM_M_SPAREPART_CATEGORY"
          searchLabel="Cari kategori sparepart"
          pageSize={PAGE_SIZE}
          emptyMessage={`Belum ada kategori sparepart pada tab ${active.label}.`}
        />
      ),
    })
  }

  return (
    <main className="mx-auto max-w-5xl px-4 py-8">
      {/* Judulnya dibaca dari `pyCaption Master Kategori Sparepart` pada
          Section/MasterKategoriSparepartHE (D-13). */}
      <ListHeader
        title="Master Kategori Sparepart"
        description="Penggolongan suku cadang alat berat yang menjadi pilihan Kategori di layar Master Sparepart dan Master Tipe Sparepart."
      >
        <AddButton onClick={openAdd} disabled={isFormOpen} />
        <RefreshButton query={list} />
      </ListHeader>

      <ApprovalPanel
        tabLabel="Tab Master Kategori Sparepart"
        tabs={TABS}
        active={active}
        onSwitch={setTab}
        onBeforeSwitch={closeForm}
        portal={list.data?.portal ?? portal}
        decide={decide}
        feedbackNoun="kategori sparepart"
        barNoun="kategori"
        chosenCount={chosen.size}
        onApprove={() => runDecision(PartCategoryStatus.disetujui)}
        onReject={() => runDecision(PartCategoryStatus.ditolak)}
        onClear={() => setChosen(new Set())}
      />

      {isFormOpen && (
        <section className="mt-5">
          <PartCategoryForm
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

