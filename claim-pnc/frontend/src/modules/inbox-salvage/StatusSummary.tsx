import {
  StatusSummaryTable,
  SummaryLinkButton,
  SummaryLoading,
} from '@/components/inbox/StatusSummaryTable'

import type { StatusCount } from './types'

type Props = {
  rows: StatusCount[]
  /** Kode daftar yang sedang terbuka, supaya barisnya dapat ditandai. */
  active: string
  onSelect: (code: string) => void
  isLoading?: boolean
}

/**
 * Tabel ringkas **"Status Salvage / Jumlah"** di atas grid.
 *
 * # Apa yang digantikan
 *
 * Tabel dua kolom pada `Section/InboxSalvage-Section.xml` — `pyCaption Status Salvage` dan
 * `pyCaption Jumlah` — yang diisi `Activity/GCNMCountSalvage_act-Act.xml`.
 *
 * # Ia NAVIGASI, bukan hiasan
 *
 * Activity pencacah menulis TIGA isian per baris, dan dua di antaranya adalah kode daftar
 * yang dituju. Artinya mengklik satu baris MEMBUKA daftarnya. Menggambarnya sebagai angka
 * mati akan menghilangkan satu-satunya cara pengguna berpindah daftar di layar lama.
 *
 * # Angka di sini TIDAK selalu sama dengan jumlah baris daftarnya
 *
 * Dan itu bukan kerusakan. Tiga baris menghitung populasi yang BERBEDA dari daftar yang
 * dibukanya — "Outstanding", "Checker", dan "Histori Salvage" — karena begitulah kueri
 * pencacahnya di Pega. Keputusan Work Owner 2026-09-25: direplikasi, karena angkanya
 * berjalan dan dibaca orang setiap hari.
 *
 * Keterangannya ada di `selisih_terencana`, yang digambar halaman ini di bawah tabel.
 */
export function StatusSummary({ rows, active, onSelect, isLoading }: Readonly<Props>) {
  if (isLoading) {
    return <SummaryLoading>Menghitung ringkasan status salvage…</SummaryLoading>
  }

  if (rows.length === 0) return null

  return (
    <StatusSummaryTable label="Ringkasan jumlah pengajuan per status salvage">
      {rows.map((row) => {
        // Baris yang TIDAK menuju daftar mana pun digambar sebagai teks biasa, bukan
        // tombol yang tidak melakukan apa-apa. Satu baris memang begitu — "Tidak
        // Terjual", yang di Pega pun tidak punya tab.
        const reachable = Boolean(row.daftar)
        const selected = reachable && row.daftar === active

        return (
          <tr
            key={row.status_salvage}
            className={selected ? 'bg-blue-50' : undefined}
          >
            <th scope="row" className="px-4 py-2 text-left font-normal">
              {reachable ? (
                <SummaryLinkButton selected={selected} onClick={() => onSelect(row.daftar ?? '')}>
                  {row.status_salvage}
                </SummaryLinkButton>
              ) : (
                <span
                  className="px-1 text-slate-600"
                  title={
                    'Baris ini tidak punya daftar sendiri — di Pega pun tidak ada ' +
                    'tab yang menerimanya.'
                  }
                >
                  {row.status_salvage}
                </span>
              )}
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
