import { DataTable, type Column } from '@/components/DataTable'
import { useSelectedPortal } from '@/app/portal'
import type { ProgressStatus } from '@/api/types'
import { MasterListLayout, editColumn } from '@/components/masterpage/MasterPage'
import { useCrudForm } from '@/components/masterpage/useCrudForm'

import {
  useClaimPositionList,
  useCreateProgressStatus,
  useProgressStatusList,
  useUpdateProgressStatus,
} from './api'
import { ProgressStatusForm, type ProgressStatusFields } from './ProgressStatusForm'

/**
 * Layar Master Status Progres 1.
 *
 * Pengganti `Harness/StatusProgress-Harness.xml` atas tabel
 * POOLDATA.GCNM_MST_PROGRESS_KLAIM. Judul, susunan kolom, dan kedua tombolnya mengikuti
 * layar lama (`D-13`: alur dan tata letak ditiru supaya pengguna tidak perlu belajar
 * ulang):
 *
 *   - Judul "Master Status Progres 1" — `Section/MasterStatusProgress-Section.xml`
 *   - Tombol "Tambah" dan "Refresh"  — section yang sama
 *   - Grid tiga kolom: ID, Status Progres, Posisi — `BrowseStatusProgress-Section.xml`
 *
 * Yang SENGAJA tidak ada: tombol hapus. Sistem lama tidak punya satu pun pernyataan
 * DELETE terhadap tabel ini — sudah diperiksa ke seluruh export — dan tabelnya pun tidak
 * punya kolom penanda terhapus yang dapat dipakai `D-66`. Menambahkannya berarti
 * mengarang perilaku yang tidak pernah ada, sekaligus berisiko: baris ini dirujuk
 * `GCNM_PROGRESS_CLAIM.STATUS_PROGRESS1` pada data klaim yang sudah berjalan.
 */
export function ProgressStatusPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const list = useProgressStatusList()
  const positions = useClaimPositionList()
  const create = useCreateProgressStatus()
  const update = useUpdateProgressStatus()

  const form = useCrudForm(create, update, (row: ProgressStatus, input: ProgressStatusFields) => ({
    id: row.id,
    input,
  }))
  const edited = form.openedRow
  const { openEdit, closeForm } = form

  // `nilai` dipisah dari `tampil` mengikuti kontrak Column: yang dicari dan diurutkan
  // adalah teks polos, yang dilihat pengguna boleh berisi markup. Menyatukannya akan
  // membuat pencarian ikut menelusuri kelas CSS.
  const columns: Column<ProgressStatus>[] = [
    { key: 'id', title: 'ID', width: 'w-20', value: (row) => row.id },
    { key: 'nama', title: 'Status Progres', value: (row) => row.nama },
    {
      key: 'posisi',
      title: 'Posisi',
      width: 'w-40',
      // Hanya SATU nilai yang digambar, bukan label plus kode di sebelahnya.
      //
      // Versi sebelumnya menyandingkan keduanya karena saya mengira yang tersimpan
      // adalah kode angka ("002") dan labelnya terpisah. Koreksi 2026-09-20 membuktikan
      // sebaliknya: dropdown-nya di Pega mengikat nilai simpanan dan label ke properti
      // yang sama, sehingga keduanya identik — menyandingkannya hanya menulis
      // "REGISTER REGISTER".
      value: (row) => row.nama_posisi,
    },
    editColumn<ProgressStatus>({
      onEdit: openEdit,
      width: 'w-24',
      ariaLabel: (row) => `Ubah ${row.nama}`,
    }),
  ]

  // Tidak ada tautan "kembali ke beranda" di sini: menu utama di kerangka sudah
  // menyediakannya, dan dua jalan ke tempat yang sama pada satu layar membuat pengguna
  // menebak mana yang dimaksud.
  return (
    <MasterListLayout
      title="Master Status Progres 1"
      description="Daftar status progres yang dapat dicatat petugas pada setiap posisi klaim."
      query={list}
      portal={portal}
      crud={form}
      form={
        <ProgressStatusForm
          edited={edited}
          positions={positions.data?.posisi ?? []}
          isSaving={form.isSaving}
          error={form.saveError}
          onSave={form.submit}
          onCancel={closeForm}
        />
      }
      loadingText="Memuat daftar status progres…"
      renderTable={(data) => (
        <DataTable
          columns={columns}
          rows={data.status_progres}
          rowKey={(row) => row.id}
          description="Sumber: POOLDATA.GCNM_MST_PROGRESS_KLAIM"
          emptyMessage="Belum ada status progres pada entitas ini."
          // 15 baris per halaman, sama seperti layar lama. Angkanya BUKAN dikarang:
          // `Section/BrowseStatusProgress-Section.xml` menyisipkan `pyGridPaginator`
          // dengan `pyPageSize = Other` dan `pyPageSizeOther = 15`.
          //
          // Ditambahkan 2026-09-20 setelah Work Owner menemukan grid ini menggambar
          // seluruh baris sekaligus, padahal layar lamanya berhalaman.
          pageSize={15}
        />
      )}
    />
  )
}
