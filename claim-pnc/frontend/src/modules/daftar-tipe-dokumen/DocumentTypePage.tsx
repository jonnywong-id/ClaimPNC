import type { DocumentType } from '@/api/types'
import { DataTable, type Column } from '@/components/DataTable'
import { useSelectedPortal } from '@/app/portal'
import { retryLoadMessage } from '@/components/masterpage/loadMessage'
import { editColumn, MasterListLayout } from '@/components/masterpage/MasterPage'
import { useCrudForm } from '@/components/masterpage/useCrudForm'

import {
  useCreateDocumentType,
  useDocumentTypeList,
  useUpdateDocumentType,
} from './api'
import { DocumentTypeForm } from './DocumentTypeForm'

/**
 * Layar Daftar Tipe Dokumen.
 *
 * Pengganti `Harness/ListDocumentTypeInbox-Harness.xml` atas tabel POOLDATA.LST_DOC_TYPE.
 * Judul, susunan kolom, dan tombolnya mengikuti layar lama (`D-13`: alur dan tata letak
 * ditiru supaya pengguna tidak perlu belajar ulang):
 *
 *   - Tombol "Tambah" dan "Refresh"   — `Section/ListDocumentType-Section.xml`
 *   - Grid tiga kolom: ID, Tipe Dokumen, Status Proses
 *                                     — `Section/BrowseListDocumentType-Section.xml`
 *   - Aksi "Update Data" per baris    — memanggil `CNMSetListDocumentType_act`
 *   - 50 baris per halaman            — `BrowseLstDocType_RD-RD.xml`, `pyPageSize=50`
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Entitas yang dilihat | tidak pernah disebut | disebut terang-terangan (`R-20`) |
 * | Arti "Status Proses" | tidak dijelaskan | diberi keterangan di form |
 *
 * # Yang sengaja TIDAK berbeda
 *
 * Tidak ada validasi. Nama kosong diterima dan nama ganda diterima, persis seperti layar
 * lama — keputusan Work Owner 2026-09-21. Ini berbeda dari Master Status Klaim, yang justru
 * diputuskan diperketat pada 2026-09-17; perbedaan keduanya disengaja dan dicatat di
 * docs/keputusan-implementasi.md.
 *
 * Tidak ada kotak cari. Layar lama tidak punya penyaring sama sekali — keputusan Work Owner
 * 2026-09-21, sama seperti Master Dokumen Travel dan berbeda dari Master Status Klaim.
 *
 * Tidak ada tombol hapus. Layar Pega pun tidak punya, `PEGA_LST_DOC_TYPE.prc` hanya mengenal
 * INSERT dan UPDATE, dan ID-nya dirujuk dua master turunan beserta dokumen klaim yang sudah
 * terunggah (`ADR-0012`, `D-66`).
 */
export function DocumentTypePage() {
  const portal = useSelectedPortal((state) => state.alias)

  const list = useDocumentTypeList()
  const create = useCreateDocumentType()
  const update = useUpdateDocumentType()

  const form = useCrudForm(create, update, (row: DocumentType, input) => ({ id: row.id, input }))
  const edited = form.openedRow
  const { openEdit, closeForm, isSaving, saveError } = form

  // `value` dipisah dari `render` mengikuti kontrak Column: yang diurutkan adalah teks
  // polos, yang dilihat pengguna boleh berisi markup.
  const columns: Column<DocumentType>[] = [
    {
      key: 'id',
      title: 'ID',
      width: 'w-28',
      value: (row) => row.id,
      render: (row) => <span className="font-mono text-sm text-slate-700">{row.id}</span>,
    },
    {
      // "Tipe Dokumen" — header kolom pada grid Pega
      // (`Section/BrowseListDocumentType-Section.xml`, pyCaption "Tipe Dokumen"). Ia
      // sengaja BERBEDA dari label isian di form, yang berbunyi "Jenis Dokumen"; kedua teks
      // itu memang tidak sama di layar lama, dan keduanya ditiru apa adanya (`D-13`).
      key: 'tipe_dokumen',
      title: 'Tipe Dokumen',
      value: (row) => row.tipe_dokumen,
      render: (row) =>
        row.tipe_dokumen ? (
          <span className="font-medium text-slate-900">{row.tipe_dokumen}</span>
        ) : (
          // Nama kosong memang dapat tersimpan — layar lama pun menerimanya. Ia ditandai
          // supaya barisnya tidak tampak seperti baris rusak atau sel yang gagal dimuat.
          <span className="text-slate-400" title="Baris ini tersimpan tanpa nama">
            (tanpa nama)
          </span>
        ),
    },
    {
      key: 'status_proses',
      title: 'Status Proses',
      width: 'w-56',
      value: (row) => row.status_proses,
      // Dirender sebagai TEKS BIASA, bukan lencana berwarna. Ia catatan bebas, bukan
      // status — merendernya sebagai lencana akan menyiratkan domain tertutup yang tidak
      // ada, dan membuat nilai yang tidak dikenali tampak seperti data rusak.
      render: (row) =>
        row.status_proses ? (
          <span className="text-slate-700">{row.status_proses}</span>
        ) : (
          <span className="text-slate-400">—</span>
        ),
    },
    editColumn<DocumentType>({
      onEdit: openEdit,
      width: 'w-32',
      ariaLabel: (row) => `Ubah tipe dokumen ${row.tipe_dokumen || row.id}`,
      label: 'Update Data',
    }),
  ]

  return (
    <MasterListLayout
      title="Daftar Tipe Dokumen"
      description="Kategori dokumen yang dapat dilampirkan pada klaim."
      query={list}
      portal={portal}
      crud={form}
      form={
        <DocumentTypeForm
          edited={edited}
          isSaving={isSaving}
          error={saveError}
          onSave={form.submit}
          onCancel={closeForm}
        />
      }
      loadingText="Memuat daftar tipe dokumen…"
      toMessage={(error) => retryLoadMessage(error, { owner: 'Daftar tipe dokumen' })}
      portalOwner="Daftar tipe dokumen"
      renderTable={(data) => (
        <DataTable
          columns={columns}
          rows={data.tipe_dokumen}
          rowKey={(row) => row.id}
          // Tanpa kotak cari, dan dipaginasi 50 baris per halaman — keduanya meniru grid
          // Pega apa adanya (`pyPageSize=50`, dan tidak ada satu pun penyaring di
          // sectionnya). Keputusan Work Owner 2026-09-21.
          searchable={false}
          pageSize={50}
          description={`${data.total} tipe dokumen terdaftar. Sumber: POOLDATA.LST_DOC_TYPE`}
          emptyMessage="Belum ada tipe dokumen pada entitas ini."
        />
      )}
    />
  )
}
