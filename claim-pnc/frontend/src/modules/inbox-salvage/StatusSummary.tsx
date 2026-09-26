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
export function StatusSummary({ rows, active, onSelect, isLoading }: Props) {
  if (isLoading) {
    return (
      <div
        className="rounded-kartu border border-slate-200 bg-white p-4 text-sm text-slate-500"
        role="status"
      >
        Menghitung ringkasan status salvage…
      </div>
    )
  }

  if (rows.length === 0) return null

  return (
    <div className="overflow-x-auto rounded-kartu border border-slate-200 bg-white">
      <table
        className="w-full min-w-max text-sm"
        aria-label="Ringkasan jumlah pengajuan per status salvage"
      >
        <caption className="sr-only">
          Ringkasan jumlah pengajuan per status salvage. Pilih satu baris untuk membuka
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
                    <button
                      type="button"
                      onClick={() => onSelect(row.daftar ?? '')}
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
        </tbody>
      </table>
    </div>
  )
}
