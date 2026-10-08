import { PartTypeStatus, type PartType } from '@/api/types'
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
  useCreatePartType,
  useDecidePartType,
  usePartTypeList,
  usePartTypeOptions,
  useSavePartType,
} from './api'
import { PartTypeForm } from './PartTypeForm'

/**
 * Tiga tab, sama persis dengan layar lama — termasuk URUTANNYA.
 *
 * `Section/MasterTipeSparepartHE-Section.xml` merujuk tiga section yang ketiganya membaca
 * tabel yang sama dan hanya berbeda pada nilai APPROVAL-nya:
 *
 *	MasterTipeSparepartHEApprove   APPROVAL="1"
 *	MasterTipeSparepartHEReject    APPROVAL="2"
 *	MasterTipeSparepartHEApproval  APPROVAL="0"
 *
 * Reject berada di TENGAH, bukan di ujung — sama seperti Master Kategori Sparepart, Master
 * Sparepart, Master Panel, dan Master Bengkel. Urutan itu tidak intuitif, tetapi ia yang
 * dilihat petugas hari ini, dan `D-13` menuntut tata letak yang sama supaya pengguna tidak
 * perlu belajar ulang.
 */
const TABS = [
  {
    id: 'approve',
    label: 'Approve',
    status: PartTypeStatus.disetujui,
    description:
      'Tipe yang sudah disetujui. Hanya yang ada di sini yang muncul sebagai pilihan Tipe ' +
      'di layar Master Sparepart.',
  },
  {
    id: 'reject',
    label: 'Reject',
    status: PartTypeStatus.ditolak,
    description:
      'Tipe yang ditolak. Namanya tetap terpakai — nama yang sama tidak dapat digunakan ' +
      'tipe lain, di kategori mana pun, sampai baris ini diubah namanya.',
  },
  {
    id: 'menunggu',
    label: 'Waiting Approval',
    status: PartTypeStatus.menunggu,
    description: 'Tipe yang menunggu diputuskan.',
  },
] as const

/**
 * Ukuran halaman diambil dari `pyPageSize` pada ketiga section tab layar ini.
 *
 * Ketiganya bernilai **50** — sama dengan Master Kategori Sparepart, berbeda dari Master
 * Sparepart yang 30 dan Master Bengkel yang 20. Tidak ada satu angka yang benar untuk
 * seluruh layar; angkanya milik layar, bukan milik komponen tabel.
 */
const PAGE_SIZE = 50

/** Mengubah galat pemuatan daftar menjadi pesan yang dapat ditindaklanjuti. */
function loadMessage(error: unknown): MessageContent {
  return reloadLoadMessage(error, { failedTitle: 'Daftar tipe sparepart tidak dapat dimuat' })
}

/**
 * Layar Master Tipe Sparepart.
 *
 * Pengganti `Harness/GCNMMasterSparepartType-Harness.xml` atas tabel
 * POOLDATA.GCNM_M_SPAREPART_TYPE (MENU_ID 34).
 *
 * # Apa yang dikelola layar ini
 *
 * **Penggolongan tingkat kedua suku cadang alat berat** — setiap tipe berinduk pada sebuah
 * kategori. Ia tabel acuan yang menyuapi satu layar lain: Master Sparepart memakai ID-nya
 * pada kolom TIPE_SPART.
 *
 * # Empat kolom, dan salah satunya milik tabel lain
 *
 * Grid Pega menampilkan "ID Tipe Sparepart", "Nama Tipe Sparepart", "ID Kategori
 * Sparepart", dan "Kategori Sparepart". Yang terakhir bukan kolom tabel ini — ia
 * PART_CATEGORY_NAME milik tabel kategori, dibaca lewat JOIN.
 *
 * Tidak ada kolom "User Update" maupun "Tanggal Update" seperti pada Master Sparepart:
 * tabel ini tidak punya kolomnya, sehingga siapa yang mengubah sebuah tipe TIDAK tersimpan
 * di mana pun.
 *
 * # Baris tanpa kategori TETAP terlihat di sini, dan itu berbeda dari Pega
 *
 * `BrowseMasterSparepartTypeClaimHE_sql` memakai inner join, sehingga tipe yang menunjuk
 * kategori yang sudah tidak ada HILANG dari layar lama — tidak dapat dilihat, tidak dapat
 * diperbaiki. Layar ini memakai LEFT JOIN dan menampilkannya dengan tanda "—" beserta
 * keterangan. Selisih perilaku ini disengaja; `claimpnc -periksa` melaporkan jumlahnya
 * lebih dulu supaya ia dapat dijelaskan sebelum uji kesetaraan dijalankan.
 *
 * # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
 *
 * Itu bukan efek samping melainkan langkah tersendiri di sistem lama:
 * `Activity/UpdateTypeSparepart_act2` menetapkan APPROVAL := "0" tanpa syarat apa pun.
 * Akibatnya menyentuh layar LAIN — tipe yang sedang menunggu hilang dari dropdown Master
 * Sparepart — dan itu dinyatakan di muka pada form, bukan dibiarkan ditemukan.
 *
 * # Kenapa Approve dan Reject ada DI SINI, bukan di Inbox Manager
 *
 * Di Pega keduanya ada di layar lain: `Section/ApprovalMasterTipeSparepartHE` dipakai Inbox
 * Manager. Inbox Manager belum dibangun, dan menunda keputusannya sampai layar itu ada
 * berarti setiap tipe yang ditambah tertahan di Waiting Approval tanpa satu pun cara
 * menyelesaikannya — dan selama tertahan, ia tidak dapat dipakai sparepart mana pun.
 *
 * Yang dipakai sebagai gantinya adalah BENTUK yang sama persis: centang beberapa baris,
 * lalu satu tombol untuk seluruh pilihan. Perlakuannya sama dengan Master Kategori
 * Sparepart, Master Bengkel, Master Panel, dan Master Sparepart. Keputusan Work Owner
 * 2026-09-21.
 *
 * # Tanpa isian Catatan
 *
 * `POOLDATA.GCNM_M_SPAREPART_TYPE` tidak punya kolom penampung alasan penolakan.
 * Menggambar isian yang diam-diam membuang isinya lebih buruk daripada tidak
 * menggambarnya.
 */
export function PartTypePage() {
  const portal = useSelectedPortal((state) => state.alias)

  const { tab, setTab, active, chosen, setChosen, toggle } = useApprovalTabs(TABS, 'approve')

  const list = usePartTypeList(active.status)
  const options = usePartTypeOptions()
  const create = useCreatePartType()
  const save = useSavePartType()
  const form = useCrudForm(create, save, (row: PartType, input) => ({
    id: row.id_tipe_sparepart,
    input,
  }))
  const editing = form.openedRow
  const isFormOpen = form.isOpen
  const { closeForm } = form
  const decide = useDecidePartType()

  const rows = list.data?.tipe_sparepart ?? []

  function openAdd() {
    form.openCreate()
    // Daftar pilihan disegarkan setiap kali form dibuka. Ia bercache panjang karena jarang
    // berubah, dan justru karena itu ia dapat basi tepat pada saat ia dipakai — kategori
    // yang ditolak sejak halaman dibuka akan tetap tampak dapat dipilih.
    options.refetch()
  }

  function openEdit(row: PartType) {
    form.openEdit(row)
    options.refetch()
  }

  function runDecision(status: string) {
    decide.mutate(
      { id_tipe_sparepart: [...chosen], status },
      { onSuccess: () => setChosen(new Set()) },
    )
  }

  /*
    Empat kolom, mengikuti grid Pega apa adanya, ditambah kolom aksi dan — pada tab Waiting
    Approval — kolom centang.

    Tidak ada kolom status: ia sudah menjadi tab, dan menggambarnya lagi di setiap baris
    hanya mengulang hal yang sama di seluruh halaman.
  */
  const columns: Column<PartType>[] = [
    ...selectColumn<PartType>({
      enabled: tab === 'menunggu',
      chosen,
      idOf: (row) => row.id_tipe_sparepart,
      nameOf: (row) => row.nama_tipe_sparepart,
      disabled: decide.isPending,
      onToggle: toggle,
    }),
    {
      key: 'id',
      title: 'ID Tipe Sparepart',
      width: '10rem',
      value: (row) => row.id_tipe_sparepart,
    },
    {
      key: 'nama',
      title: 'Nama Tipe Sparepart',
      value: (row) => row.nama_tipe_sparepart,
    },
    {
      key: 'id_kategori',
      title: 'ID Kategori Sparepart',
      width: '11rem',
      value: (row) => row.id_kategori_sparepart,
    },
    {
      key: 'kategori',
      title: 'Kategori Sparepart',
      // Nilai pencariannya tetap nama kategori apa adanya — termasuk saat kosong — supaya
      // pencarian bawaan DataTable tidak ikut menjaring tanda "—" yang hanya tampilan.
      value: (row) => row.nama_kategori_sparepart,
      render: (row) =>
        row.nama_kategori_sparepart === '' ? (
          /*
            Baris yatim: kategorinya tidak ada di master kategori.

            Di Pega baris ini tidak akan pernah terlihat — inner join-nya membuangnya. Di
            sini ia ditampilkan beserta alasannya, karena hanya dengan begitu petugas dapat
            memperbaikinya lewat tombol Ubah.
          */
          <span
            className="text-amber-700"
            title="Kategori induknya tidak ada di Master Kategori Sparepart. Buka Ubah lalu pilih kategori yang sah."
          >
            — kategori tidak ditemukan
          </span>
        ) : (
          <span>{row.nama_kategori_sparepart}</span>
        ),
    },
    editColumn<PartType>({
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
      loadingText: 'Memuat daftar tipe sparepart…',
      toMessage: loadMessage,
      render: () => (
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(row) => row.id_tipe_sparepart}
          description="Sumber: POOLDATA.GCNM_M_SPAREPART_TYPE"
          searchLabel="Cari tipe atau kategori sparepart"
          pageSize={PAGE_SIZE}
          emptyMessage={`Belum ada tipe sparepart pada tab ${active.label}.`}
        />
      ),
    })
  }

  return (
    <main className="mx-auto max-w-6xl px-4 py-8">
      {/* Judulnya dibaca dari `pyCaption Master Tipe Sparepart` pada
          Section/MasterTipeSparepartHE (D-13). */}
      <ListHeader
        title="Master Tipe Sparepart"
        description="Penggolongan tingkat kedua suku cadang alat berat. Setiap tipe berinduk pada satu kategori, dan menjadi pilihan Tipe di layar Master Sparepart."
      >
        <AddButton onClick={openAdd} disabled={isFormOpen} />
        <RefreshButton query={list} />
      </ListHeader>

      <ApprovalPanel
        tabLabel="Tab Master Tipe Sparepart"
        tabs={TABS}
        active={active}
        onSwitch={setTab}
        onBeforeSwitch={closeForm}
        portal={list.data?.portal ?? portal}
        decide={decide}
        feedbackNoun="tipe sparepart"
        barNoun="tipe"
        chosenCount={chosen.size}
        onApprove={() => runDecision(PartTypeStatus.disetujui)}
        onReject={() => runDecision(PartTypeStatus.ditolak)}
        onClear={() => setChosen(new Set())}
      />

      {isFormOpen && (
        <section className="mt-5">
          <PartTypeForm
            editing={editing}
            category={options.data?.kategori ?? []}
            isLoadingCategory={options.isPending || options.isFetching}
            isCategoryTruncated={options.data?.terpotong ?? false}
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

