import type { SimasOnlineCauseOfLoss, SimasOnlineCauseOfLossInput } from '@/api/types'
import { DataTable, type Column } from '@/components/DataTable'
import { useSelectedPortal } from '@/app/portal'
import {
  BusinessListWarning,
  businessFormProps,
  MasterListLayout,
  businessNames,
  editColumn,
} from '@/components/masterpage/MasterPage'
import { useCrudForm } from '@/components/masterpage/useCrudForm'

import {
  useBusinessList,
  useCauseOfLoss,
  useCauseOfLossList,
  useCreateCauseOfLoss,
  useUpdateCauseOfLoss,
} from './api'
import { CauseOfLossForm, type CauseOfLossFields } from './CauseOfLossForm'

/**
 * Layar Master COL Simas Online.
 *
 * Pengganti `Harness/CauseOfLossInboxSimasOnline-Harness.xml` atas tabel
 * POOLDATA.M_CAUSE_OF_LOSS beserta pemetaan bisnisnya. Judul, susunan kolom, dan kedua
 * tombolnya mengikuti layar lama (`D-13`: alur dan tata letak ditiru supaya pengguna
 * tidak perlu belajar ulang):
 *
 *   - Judul "Master Cause Of Loss Simas Online" — `Section/Online_GridCauseOfLoss-Section.xml`
 *   - Tombol "Tambah" dan "Refresh"           — section yang sama
 *   - Grid dua kolom: ID dan Description      — section yang sama
 *
 * Yang SENGAJA tidak ada: tombol hapus. Sistem lama tidak punya satu pun pernyataan
 * DELETE terhadap tabel ini — `PEGA_M_CAUSE_OF_LOSS.prc` hanya INSERT dan UPDATE — dan
 * `D-66` melarang penghapusan fisik data bernilai bisnis. Menambahkannya berarti
 * mengarang perilaku yang tidak pernah ada, sekaligus berisiko: baris ini dirujuk
 * `D_CAUSE_OF_LOSS.M_COL_ID` pada data yang sudah berjalan.
 */
export function CauseOfLossPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const list = useCauseOfLossList()
  const businesses = useBusinessList()
  const create = useCreateCauseOfLoss()
  const update = useUpdateCauseOfLoss()

  const form = useCrudForm(
    create,
    update,
    (row: SimasOnlineCauseOfLoss, input: SimasOnlineCauseOfLossInput) => ({ id: row.id, input }),
  )
  const openedRow = form.openedRow
  const { openEdit } = form

  // Baris yang dibuka dimuat ULANG dari server supaya pemetaan bisnisnya ikut terbawa.
  // Daftar sengaja tidak membawanya — grid hanya menampilkan ID dan nama — sehingga baris
  // yang diambil dari daftar SELALU punya `bisnis: []`. Memakainya langsung akan membuat
  // form tampak seolah seluruh bisnisnya sudah dihapus, dan menyimpannya benar-benar
  // menghapusnya.
  const detail = useCauseOfLoss(openedRow?.id ?? null)

  const edited = openedRow === null ? null : (detail.data?.cause_of_loss ?? openedRow)

  function save(values: CauseOfLossFields) {
    form.submit({ nama: values.nama, bisnis: businessNames(values.bisnis) })
  }

  // `value` dipisah dari `render` mengikuti kontrak Column: yang dicari dan diurutkan
  // adalah teks polos, yang dilihat pengguna boleh berisi markup.
  const columns: Column<SimasOnlineCauseOfLoss>[] = [
    { key: 'id', title: 'ID', width: 'w-28', value: (row) => row.id },
    {
      key: 'nama',
      title: 'Description',
      // Grid Pega hanya menampilkan ID dan Description, dan sejak MST_COL_ID dicabut
      // (Work Owner 2026-09-23) tidak ada lagi yang perlu ditampilkan di sampingnya.
      value: (row) => row.nama,
    },
    editColumn<SimasOnlineCauseOfLoss>({
      onEdit: openEdit,
      width: 'w-24',
      ariaLabel: (row) => `Ubah ${row.nama}`,
    }),
  ]

  // Judulnya diambil apa adanya dari caption layar Pega, supaya pengguna mengenalinya
  // tanpa diberi tahu.
  return (
    <MasterListLayout
      title="Master Cause Of Loss Simas Online"
      description="Penyebab kerugian beserta padanannya di Simas Online dan bisnis yang memakainya."
      query={list}
      portal={portal}
      crud={form}
      form={
        <>
          <BusinessListWarning show={businesses.isError} />
          <CauseOfLossForm
            edited={edited}
            onSave={save}
            {...businessFormProps(businesses, openedRow !== null && detail.isPending, form)}
          />
        </>
      }
      loadingText="Memuat daftar cause of loss…"
      renderTable={(data) => (
        <DataTable
          columns={columns}
          rows={data.cause_of_loss}
          rowKey={(row) => row.id}
          description="Sumber: POOLDATA.M_CAUSE_OF_LOSS"
          emptyMessage="Belum ada cause of loss pada entitas ini."
        />
      )}
    />
  )
}
