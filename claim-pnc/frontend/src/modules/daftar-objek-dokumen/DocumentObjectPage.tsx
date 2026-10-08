import type { DocumentObject, DocumentObjectInput } from '@/api/types'
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
  useCreateDocumentObject,
  useDocumentObject,
  useDocumentObjectList,
  useUpdateDocumentObject,
} from './api'
import { DocumentObjectForm, type DocumentObjectFields } from './DocumentObjectForm'

/**
 * Layar Daftar Objek Dokumen (MENU_ID 43).
 *
 * Pengganti `Harness/ListDocumentObject-Harness.xml`. Judul, susunan kolom, dan kedua
 * tombolnya mengikuti layar lama (`D-13`: alur dan tata letak ditiru supaya pengguna tidak
 * perlu belajar ulang):
 *
 *   - Judul "Daftar Objek Dokumen"          — `Section/BrowseDocumentObject-Section.xml:6715`
 *   - Tombol "Tambah" dan "Refresh"          — `Section/ListDocumentObject-Section.xml`
 *   - Grid dua kolom: ID dan keterangannya   — `Section/BrowseDocumentObject-Section.xml`
 *   - Aksi baris "Ubah"                      — `pyButtonLabel` pada section yang sama
 *
 * Yang SENGAJA tidak ada:
 *
 *   - Tombol hapus. Seluruh grid di layar lama ber-`pyGridDeleteActivityExists=false`, dan
 *     `D-66` melarang penghapusan fisik data bernilai bisnis. Barisnya pun dirujuk
 *     `LST_TYPE_DOC_BUSINESS.OBJECT_DOC_ID` pada data yang sudah berjalan — menghapusnya
 *     akan memutus rujukan itu.
 *   - Kolom ID lama (OLD_ID). Server mengirimkannya karena Report Definition memuatnya,
 *     tetapi grid Pega hanya memuat DUA kolom. Ketiadaannya di layar adalah keputusan,
 *     bukan kelalaian.
 */
export function DocumentObjectPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const list = useDocumentObjectList()
  const businesses = useBusinessList()
  const create = useCreateDocumentObject()
  const update = useUpdateDocumentObject()

  const form = useCrudForm(create, update, (row: DocumentObject, input: DocumentObjectInput) => ({
    id: row.id,
    input,
  }))
  const openedRow = form.openedRow
  const { openEdit } = form

  // Baris yang dibuka dimuat ULANG dari server supaya pemetaan bisnisnya ikut terbawa.
  // Daftar sengaja tidak membawanya — grid hanya menampilkan ID dan keterangannya —
  // sehingga baris yang diambil dari daftar SELALU punya `bisnis: []`. Memakainya langsung
  // akan membuat form tampak seolah seluruh bisnisnya sudah dihapus, dan menyimpannya
  // benar-benar mencabutnya.
  const detail = useDocumentObject(openedRow?.id ?? null)

  const edited = openedRow === null ? null : (detail.data?.objek_dokumen ?? openedRow)

  function save(values: DocumentObjectFields) {
    form.submit({ objek_dokumen: values.objek_dokumen, bisnis: businessNames(values.bisnis) })
  }

  // `value` dipisah dari `render` mengikuti kontrak Column: yang dicari dan diurutkan adalah
  // teks polos, yang dilihat pengguna boleh berisi markup.
  const columns: Column<DocumentObject>[] = [
    {
      key: 'id',
      title: 'ID',
      width: 'w-28',
      value: (row) => row.id,
      render: (row) => <span className="font-mono text-sm text-slate-700">{row.id}</span>,
    },
    {
      key: 'objek_dokumen',
      // Judul kolom diambil apa adanya dari sel ber-`pyCellHeader=true` di layar Pega. Ia
      // memang sama dengan judul layarnya, dan itu dibiarkan — `D-13` menetapkan teks layar
      // ditiru, bukan dirapikan.
      title: 'Daftar Objek Dokumen',
      value: (row) => row.objek_dokumen,
    },
    editColumn<DocumentObject>({
      onEdit: openEdit,
      width: 'w-24',
      ariaLabel: (row) => `Ubah objek dokumen ${row.objek_dokumen || row.id}`,
    }),
  ]

  // Judulnya diambil apa adanya dari caption layar Pega, supaya pengguna mengenalinya
  // tanpa diberi tahu.
  return (
    <MasterListLayout
      title="Daftar Objek Dokumen"
      description="Objek yang melekat pada dokumen klaim, beserta bisnis yang memakainya."
      query={list}
      portal={portal}
      crud={form}
      form={
        <>
          <BusinessListWarning show={businesses.isError} />
          <DocumentObjectForm
            edited={edited}
            onSave={save}
            {...businessFormProps(businesses, openedRow !== null && detail.isPending, form)}
          />
        </>
      }
      loadingText="Memuat daftar objek dokumen…"
      renderTable={(data) => (
        <DataTable
          columns={columns}
          rows={data.objek_dokumen}
          rowKey={(row) => row.id}
          description={`${data.total} objek dokumen terdaftar. Sumber: POOLDATA.V_LST_DOC_OBJ`}
          emptyMessage="Belum ada objek dokumen pada entitas ini."
        />
      )}
    />
  )
}
