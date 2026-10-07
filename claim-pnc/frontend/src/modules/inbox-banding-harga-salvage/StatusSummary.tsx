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
    return (
      <output
        className="block rounded-kartu border border-slate-200 bg-white p-4 text-sm text-slate-500"
      >
        Menghitung ringkasan banding harga…
      </output>
    )
  }

  if (rows.length === 0) return null

  return (
    <div className="overflow-x-auto rounded-kartu border border-slate-200 bg-white">
      <table
        className="w-full min-w-max text-sm"
        aria-label="Ringkasan jumlah banding harga per status salvage"
      >
        <caption className="sr-only">
          Ringkasan jumlah banding harga per status salvage. Pilih satu baris untuk membuka
          daftarnya.
        </caption>

        <thead className="bg-slate-50 text-left text-xs font-semibold tracking-wide text-slate-600 uppercase">
          <tr>
            <th scope="col" className="px-4 py-2.5">
              Status Salvage
            </th>
            <th scope="col" className="px-4 py-2.5 text-right">
              Jumlah
            </th>
          </tr>
        </thead>

        <tbody className="divide-y divide-slate-100">
          {rows.map((row) => {
            const selected = row.tab === active

            return (
              <tr key={row.tab} className={selected ? 'bg-blue-50' : undefined}>
                <th scope="row" className="px-4 py-2 text-left font-normal">
                  <button
                    type="button"
                    onClick={() => onSelect(row.tab)}
                    aria-current={selected ? 'true' : undefined}
                    className={[
                      'rounded-kontrol px-1 text-left',
                      'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
                      selected
                        ? 'font-semibold text-blue-700'
                        : 'text-blue-700 hover:underline',
                    ].join(' ')}
                  >
                    {row.status_salvage}
                  </button>
                </th>

                <td className="px-4 py-2 text-right tabular-nums text-slate-900">
                  {row.jumlah.toLocaleString('id-ID')}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
