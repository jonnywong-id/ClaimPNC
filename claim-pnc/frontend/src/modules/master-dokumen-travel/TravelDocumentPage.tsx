import type { TravelDocument } from '@/api/types'
import { DataTable, type Column } from '@/components/DataTable'
import { useSelectedPortal } from '@/app/portal'
import { retryLoadMessage } from '@/components/masterpage/loadMessage'
import { editColumn, MasterListLayout } from '@/components/masterpage/MasterPage'
import { useCrudForm } from '@/components/masterpage/useCrudForm'

import {
  useCreateTravelDocument,
  useTravelDocumentList,
  useUpdateTravelDocument,
} from './api'
import { TravelDocumentForm } from './TravelDocumentForm'

/**
 * Layar Master Dokumen Travel.
 *
 * Pengganti `Harness/BrowseMasterDocumentTravel_Harness-Harness.xml` atas tabel
 * POOLDATA.M_DOCTRAVEL. Judul, susunan kolom, dan tombolnya mengikuti layar lama
 * (`D-13`: alur dan tata letak ditiru supaya pengguna tidak perlu belajar ulang):
 *
 *   - Judul "Master Dokumen Travel"  — `Section/BrowseMasterDocumentTravel-Section.xml`
 *   - Tombol "Tambah" dan "Refresh"  — section yang sama
 *   - Grid dua kolom: ID dan Judul Dokumen — `BrowseMstDocTravel_RD-RD.xml`
 *   - Aksi "Ubah" per baris          — memanggil `SetMstDocTravelValue_act(docid)`
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Pencarian | tidak ada | satu kotak cari yang menelusuri kedua kolom |
 * | Entitas yang dilihat | tidak pernah disebut | disebut terang-terangan (`R-20`) |
 *
 * # Yang sengaja TIDAK berbeda
 *
 * Tidak ada validasi. Judul kosong diterima dan judul ganda diterima, persis seperti
 * layar lama — keputusan Work Owner 2026-09-21. Ini berbeda dari Master Status Klaim,
 * yang justru diputuskan diperketat pada 2026-09-17; perbedaan keduanya disengaja dan
 * dicatat di docs/keputusan-implementasi.md.
 *
 * Tidak ada tombol hapus. Layar Pega pun tidak punya, `DOCTRAVEL_CVG.prc` hanya mengenal
 * INSERT dan UPDATE, dan DOCID dirujuk baris V_LST_DOC_TRAVEL beserta dokumen klaim yang
 * sudah terunggah (`ADR-0012`).
 */
export function TravelDocumentPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const list = useTravelDocumentList()
  const create = useCreateTravelDocument()
  const update = useUpdateTravelDocument()

  const form = useCrudForm(create, update, (row: TravelDocument, input) => ({ id: row.id, input }))
  const edited = form.openedRow
  const { openEdit, closeForm, isSaving, saveError } = form

  // `value` dipisah dari `render` mengikuti kontrak Column: yang dicari dan diurutkan
  // adalah teks polos, yang dilihat pengguna boleh berisi markup.
  const columns: Column<TravelDocument>[] = [
    {
      key: 'id',
      title: 'ID',
      width: 'w-32',
      value: (row) => row.id,
      render: (row) => <span className="font-mono text-sm text-slate-700">{row.id}</span>,
    },
    {
      // "Judul Dokumen Travel" — header kolom kedua pada grid Pega
      // (`Section/BrowseMasterDocumentTravel-Section.xml`, Embed-Display-Table-Cell
      // indeks 2). Ia sengaja BERBEDA dari label isian di form, yang berbunyi "Judul
      // Dokumen" saja; kedua teks itu memang tidak sama di layar lama, dan keduanya
      // ditiru apa adanya (`D-13`, keputusan Work Owner 2026-09-21).
      key: 'judul',
      title: 'Judul Dokumen Travel',
      value: (row) => row.judul,
      render: (row) =>
        row.judul ? (
          <span className="font-medium text-slate-900">{row.judul}</span>
        ) : (
          // Judul kosong memang dapat tersimpan — layar lama pun menerimanya. Ia
          // ditandai supaya barisnya tidak tampak seperti baris rusak atau sel yang
          // gagal dimuat.
          <span className="text-slate-400" title="Baris ini tersimpan tanpa judul">
            (tanpa judul)
          </span>
        ),
    },
    editColumn<TravelDocument>({
      onEdit: openEdit,
      width: 'w-24',
      ariaLabel: (row) => `Ubah dokumen ${row.judul || row.id}`,
    }),
  ]

  return (
    <MasterListLayout
      title="Master Dokumen Travel"
      description="Daftar jenis dokumen yang dapat diminta pada klaim lini Travel."
      query={list}
      portal={portal}
      crud={form}
      form={
        <TravelDocumentForm
          edited={edited}
          isSaving={isSaving}
          error={saveError}
          onSave={form.submit}
          onCancel={closeForm}
        />
      }
      loadingText="Memuat daftar dokumen travel…"
      toMessage={(error) => retryLoadMessage(error, { owner: 'Daftar dokumen Travel' })}
      portalOwner="Daftar dokumen Travel"
      renderTable={(data) => (
        <DataTable
          columns={columns}
          rows={data.dokumen_travel}
          rowKey={(row) => row.id}
          // Tanpa kotak cari, dan dipaginasi 50 baris per halaman — keduanya meniru
          // grid Pega apa adanya (`pyPageSize=50`, dan tidak ada satu pun
          // `pySortFilterProperty` yang terisi). Keputusan Work Owner 2026-09-21.
          //
          // Berbeda dari Master Status Klaim, yang justru ditambahi kotak cari pada
          // 2026-09-17. Perbedaan itu disengaja dan dicatat di
          // docs/keputusan-implementasi.md.
          searchable={false}
          pageSize={50}
          description={`${data.total} dokumen terdaftar. Sumber: POOLDATA.M_DOCTRAVEL`}
          emptyMessage="Belum ada dokumen travel pada entitas ini."
        />
      )}
    />
  )
}
