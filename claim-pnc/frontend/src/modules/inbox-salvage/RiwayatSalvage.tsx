import { DataTable, type Column } from '@/components/DataTable'

import type { HistoryRow } from './types'

/**
 * Kolom grid **"Detail History Salvage"** — lima, dan kelimanya apa adanya.
 *
 * Judulnya disalin dari `Section/DetailPengajuanSalvage-Section.xml`, yang hanya memuat
 * lima caption: Tanggal Input · Nomor Klaim · PIC · Nilai Minimum · Posisi Salvage.
 *
 * Satu di antaranya berganti judul antarlayar: kolom yang di panel rincian berjudul
 * "Estimasi" di sini berjudul **"Nilai Minimum"**, padahal kolom basis datanya sama.
 * Itu bukan salah salin — kedua layar lama memang menamainya berbeda.
 */
const HISTORY_COLUMNS: Column<HistoryRow>[] = [
  {
    key: 'tanggal_input',
    title: 'Tanggal Input',
    width: '10rem',
    value: (row) => row.tanggal_input,
  },
  { key: 'no_klaim', title: 'Nomor Klaim', width: '10rem', value: (row) => row.no_klaim },
  { key: 'pic', title: 'PIC', width: '12rem', value: (row) => row.pic },
  {
    key: 'nilai_minimum',
    title: 'Nilai Minimum',
    width: '10rem',
    alignRight: true,
    value: (row) => row.nilai_minimum,
  },
  {
    key: 'posisi_salvage',
    title: 'Posisi Salvage',
    width: '14rem',
    value: (row) => row.posisi_salvage,
  },
]

/**
 * Grid **"Detail History Salvage"**.
 *
 * # Dipakai DUA layar, karena di Pega pun begitu
 *
 * `Section/DetailPengajuanSalvage-Section.xml` disisipkan oleh `DataDetail_Salvage`,
 * `Data_Salvage`, maupun `TambahData_Salvage` — satu grid yang sama digambar di panel
 * rincian dan di form Tambah. Menyalinnya menjadi dua berarti menunggu keduanya bergeser.
 *
 * # Ia digambar SELALU, termasuk ketika kosong
 *
 * Kekosongannya adalah keterangan, bukan ketiadaan: ia menyatakan klaim ini belum pernah
 * diajukan salvage sama sekali. Menyembunyikan gridnya membuat keadaan itu tidak dapat
 * dibedakan dari grid yang gagal dimuat.
 */
export function RiwayatSalvage({ rows }: { rows: HistoryRow[] }) {
  return (
    <section aria-label="Detail History Salvage">
      <h3 className="mb-3 text-sm font-semibold text-slate-800">Detail History Salvage</h3>

      <DataTable
        columns={HISTORY_COLUMNS}
        rows={rows}
        rowKey={(row) => row.id_salvage}
        emptyMessage="Klaim ini belum pernah diajukan salvage."
        hideSearch
      />
    </section>
  )
}
