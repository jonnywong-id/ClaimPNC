import { SalvageDonut } from './SalvageDonut'
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
 * Dan itu bukan kerusakan. Dua baris menghitung populasi yang BERBEDA dari daftar yang
 * dibukanya — "Checker" dan "Histori Salvage" — karena begitulah kueri pencacahnya di Pega.
 * Keputusan Work Owner 2026-09-25: direplikasi, karena angkanya berjalan dan dibaca orang
 * setiap hari.
 *
 * Baris "Outstanding" DULU termasuk, dan sejak 2026-10-08 tidak lagi: angkanya ternyata
 * tidak sepadan dengan daftar Pega mana pun, dan Work Owner memutuskan ia mengikuti
 * daftarnya. Lihat `inboxsalvage.OutstandingSalvageStatuses` di backend.
 *
 * Keterangannya TIDAK lagi digambar di layar: panel selisih terencana dihapus atas
 * keputusan Work Owner 2026-10-06. Daftarnya tetap hidup di `inboxsalvage.PlannedDifferences`
 * sebagai pemetaan ke butir `P-5` untuk uji kesetaraan gerbang 1 (`D-54`).
 *
 * # Grafik di kiri, tabel di kanan
 *
 * Susunan itu disalin dari layar lama, dan keduanya memang satu kesatuan: grafiknya
 * menjawab "sebarannya bagaimana", tabelnya menjawab "berapa tepatnya, dan bawa saya ke
 * sana". Pada layar sempit keduanya bertumpuk — grafik lebih dulu, karena ia yang terbaca
 * sekilas.
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
    <div className="flex flex-wrap items-start justify-center gap-8 rounded-kartu border border-slate-200 bg-white p-4">
      <SalvageDonut rows={rows} />

      <div className="min-w-0 grow overflow-x-auto sm:max-w-md">
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

                        // `aria-expanded` menyatakan baris ini MEMBUKA sesuatu, dan
                        // apakah yang dibukanya sedang terbuka.
                        //
                        // Ia yang menyampaikan kepada pembaca layar hal yang bagi pembaca
                        // awas disampaikan tanda panah di sebelah kiri: daftarnya belum
                        // tergambar sampai baris ini ditekan.
                        aria-expanded={selected}
                        className={[
                          'flex items-center gap-1.5 rounded-kontrol px-1 text-left',
                          'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
                          selected
                            ? 'font-semibold text-blue-700'
                            : 'text-blue-700 hover:underline',
                        ].join(' ')}
                      >
                        {/*
                          Tanda panah, dan ia BERPUTAR saat barisnya terbuka.

                          Digambar sebagai SVG, bukan sebagai karakter `▶`. Sebabnya bukan
                          selera: glif di dalam aliran teks ikut terbawa `textContent`,
                          ikut tersalin saat orang menyalin nama statusnya, dan ikut
                          muncul di setiap perbandingan teks. SVG tidak meninggalkan
                          simpul teks sama sekali.

                          `aria-hidden` karena artinya sudah dibawa `aria-expanded` di
                          atas; membacakannya dua kali membuat setiap baris terdengar
                          berulang.
                        */}
                        <svg
                          aria-hidden="true"
                          viewBox="0 0 8 10"
                          className={[
                            'h-2.5 w-2 shrink-0 transition-transform',
                            selected ? 'rotate-90 text-blue-700' : 'text-slate-400',
                          ].join(' ')}
                        >
                          <path d="M0 0l8 5-8 5z" fill="currentColor" />
                        </svg>
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
    </div>
  )
}
