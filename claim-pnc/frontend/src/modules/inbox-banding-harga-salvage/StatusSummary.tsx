import {
  StatusSummaryTable,
  SummaryLinkButton,
  SummaryLoading,
} from '@/components/inbox/StatusSummaryTable'

import type { SummaryRow } from './types'

type Props = {
  rows: SummaryRow[]
  /** Kode tab yang sedang terbuka, supaya barisnya dapat ditandai. */
  active: string
  onSelect: (code: string) => void
  isLoading?: boolean
}

/**
 * Tabel ringkas **"Status Salvage / Jumlah"** di atas grid.
 *
 * # Apa yang digantikan
 *
 * Tabel dua kolom pada `Section/InboxReqSalvageASM-Section.xml` — `pyCaption Status Salvage`
 * dan `pyCaption Jumlah` — yang diisi `Activity/GCNMCountRequestSalvage_act-Act.xml` langkah
 * 9 sampai 12. Label kedua barisnya disalin harfiah dari `Local.LOOP` di sana.
 *
 * Harness lama menggambar isi yang sama DUA KALI: sebagai tabel dan sebagai diagram
 * (`pxChart`). Diagramnya tidak dibawa — dua baris angka tidak memerlukan grafik, dan
 * menggambarkannya hanya menambah satu hal yang harus dijaga tetap sepadan dengan tabelnya.
 *
 * # Ia NAVIGASI, bukan hiasan
 *
 * Kedua barisnya menuju tab yang bersangkutan, sehingga mengkliknya MEMBUKA daftarnya.
 * Menggambarnya sebagai angka mati akan menghilangkan satu cara berpindah yang sudah ada.
 *
 * # Angka di sini SAMA dengan jumlah baris daftarnya
 *
 * Dan itu berbeda dari layar lama. Di sana pencacah dan daftarnya menghitung populasi yang
 * berbeda — kolom yang disyaratkan bahkan berada di tabel yang berlainan — sehingga angkanya
 * tidak pernah cocok. Perbaikan itu disetujui Work Owner 2026-09-29 dan dinyatakan lewat
 * `selisih_terencana`, yang digambar halaman ini di bawah tabel.
 */
export function StatusSummary({ rows, active, onSelect, isLoading }: Readonly<Props>) {
  if (isLoading) {
    return <SummaryLoading>Menghitung ringkasan banding harga…</SummaryLoading>
  }

  if (rows.length === 0) return null

  return (
    <StatusSummaryTable label="Ringkasan jumlah banding harga per status salvage">
      {rows.map((row) => {
        const selected = row.tab === active

        return (
          <tr key={row.tab} className={selected ? 'bg-blue-50' : undefined}>
            <th scope="row" className="px-4 py-2 text-left font-normal">
              <SummaryLinkButton selected={selected} onClick={() => onSelect(row.tab)}>
                {row.status_salvage}
              </SummaryLinkButton>
            </th>

            <td className="px-4 py-2 text-right tabular-nums text-slate-900">
              {row.jumlah.toLocaleString('id-ID')}
            </td>
          </tr>
        )
      })}
    </StatusSummaryTable>
  )
}
