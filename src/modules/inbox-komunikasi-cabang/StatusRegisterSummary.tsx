import { FolderIcon } from '@/components/Icon'

import { KomunikasiDonut, type DonutSlice } from './KomunikasiDonut'
import type { Summary, Tab } from './types'

/**
 * Warna kedua irisan, DITETAPKAN menurut status — bukan menurut urutan baris.
 *
 * Petugas yang membuka layar ini tiap hari mengenali irisannya dari warnanya sebelum
 * membaca legendanya, sehingga warna sebuah status tidak boleh berpindah ketika salah
 * satunya kebetulan bernilai nol dan irisannya tidak digambar.
 */
const COLOR_ANSWERED = '#3b9fd8'
const COLOR_NOT_ANSWERED = '#e8a33d'

type Props = {
  tabs: Tab[]
  /** Kedua pencacah. `undefined` berarti angkanya belum tiba. */
  summary: Summary | undefined
  /** Kode tab yang sedang terbuka, supaya barisnya dapat ditandai. */
  active: string
  onSelect: (code: string) => void
  isLoading?: boolean
}

type Row = {
  /** Kode tab yang dibuka baris ini. */
  tab: string
  label: string
  value: number
  color: string
}

/**
 * Kepala layar Inbox Komunikasi Cabang: **grafik di kiri, tabel "Status Register" di kanan**.
 *
 * # Apa yang digantikan
 *
 * Satu baris layout pada `Section/InboxKomunikasi-Section.xml` yang memuat dua hal
 * berdampingan — grafik donat `PietempCountKomunikasiCabang`, dan tabel dua kolom
 * `pyCaption Status Register` / `pyCaption Jumlah` yang dibaca dari
 * `tempCountKomunikasiCabang`. Keduanya diisi `PNCCountKomunikasiCabang_Act` dalam satu
 * jalan: langkah 10 mencacah yang sudah dijawab, langkah 13 yang belum.
 *
 * Bagian ini HILANG pada versi pertama modul, dan angkanya dipindahkan menjadi lencana di
 * bilah tab. Ia dikembalikan atas permintaan Work Owner 2026-10-09: layar yang dibuka tiap
 * pagi untuk memeriksa "berapa yang menunggu dijawab" kehilangan satu-satunya hal yang
 * menjawabnya tanpa membaca.
 *
 * # Ia NAVIGASI, bukan hiasan
 *
 * Sama seperti di Pega, tempat setiap baris ringkasnya membuka daftarnya. Menggambarnya
 * sebagai angka mati akan menambah tabel yang tidak melakukan apa pun di atas tabel yang
 * melakukan segalanya.
 *
 * # Urutan barisnya mengikuti Pega, BUKAN urutan tab
 *
 * "Answered" lebih dulu, "Not Answered" sesudahnya — urutan langkah pencacahnya di
 * `PNCCountKomunikasiCabang_Act`, dan urutan yang terbaca di layar lama. Bilah tab di
 * bawahnya berurutan sebaliknya karena ia antrean pekerjaan, dan antrean dibuka dari yang
 * menunggu.
 *
 * # Labelnya datang dari NAMA TAB, bukan ditulis di sini
 *
 * Sejak keputusan Work Owner 2026-10-09 nama tab berbunyi "Answered" dan "Not Answered" —
 * kata yang sama persis dengan label pencacah Pega. Membacanya dari `Tab.nama` alih-alih
 * menuliskannya ulang menjaga tabel ini dan bilah tab di bawahnya tidak dapat berselisih
 * nama untuk daftar yang sama.
 *
 * Judul kolomnya pun harfiah dari layar lama: "Status Register" dan "Jumlah".
 *
 * # Angka di sini TIDAK selalu sama dengan jumlah baris tabel di bawahnya
 *
 * Dan itu bukan kerusakan: pencacahnya memeriksa DUA kolom balasan sementara grid hanya
 * memeriksa satu, dan pencacah tidak menyaring pesan maupun pengirim kosong sementara grid
 * menyaringnya. Keduanya begitu di layar lama, dan perbedaannya sudah dinyatakan lewat
 * `selisih_terencana`.
 */
export function StatusRegisterSummary({ tabs, summary, active, onSelect, isLoading }: Props) {
  if (isLoading === true) {
    return (
      <div
        className="mt-4 rounded-kartu border border-slate-200 bg-white p-4 text-sm text-slate-500"
        role="status"
      >
        Menghitung ringkasan status register…
      </div>
    )
  }

  // Angkanya tidak tiba dan pemuatannya pun sudah selesai — artinya permintaannya GAGAL.
  // Kerangka kosong yang tetap digambar di situ akan terbaca sebagai "nol percakapan",
  // dan nol adalah jawaban yang menenangkan atas pertanyaan yang justru belum terjawab.
  // Kegagalannya sendiri sudah dinyatakan oleh tabel di bawahnya.
  if (summary === undefined) return null

  const rows = rowsOf(tabs, summary)
  if (rows.length === 0) return null

  const slices: DonutSlice[] = rows.map((row) => ({
    label: row.label,
    value: row.value,
    color: row.color,
  }))

  return (
    <div className="mt-4 flex flex-wrap items-start justify-center gap-8 rounded-kartu border border-slate-200 bg-white p-4">
      <KomunikasiDonut slices={slices} />

      <div className="min-w-0 grow overflow-x-auto sm:max-w-md">
        <table
          className="w-full min-w-max text-sm"
          aria-label="Ringkasan jumlah percakapan per status register"
        >
          <caption className="sr-only">
            Ringkasan jumlah percakapan per status register. Pilih satu baris untuk membuka
            daftarnya.
          </caption>

          <thead className="border-b border-slate-200 text-left text-xs font-semibold tracking-wide text-slate-600 uppercase">
            <tr>
              <th scope="col" className="px-4 py-2.5">
                Status Register
              </th>
              <th scope="col" className="px-4 py-2.5">
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
                      // `aria-expanded` menyatakan baris ini MEMBUKA sesuatu, dan apakah
                      // yang dibukanya sedang terbuka. Ia yang menyampaikan kepada pembaca
                      // layar hal yang bagi pembaca awas disampaikan ikon map di sebelah
                      // kiri — karena itu ikonnya sendiri tetap `aria-hidden`.
                      aria-expanded={selected}
                      className={[
                        'flex items-center gap-1.5 rounded-kontrol px-1 text-left',
                        'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
                        selected
                          ? 'font-semibold text-blue-700'
                          : 'text-blue-700 hover:underline',
                      ].join(' ')}
                    >
                      <FolderIcon
                        className={[
                          'h-3.5 w-3.5 shrink-0',
                          selected ? 'text-blue-700' : 'text-amber-500',
                        ].join(' ')}
                      />
                      {row.label}
                    </button>
                  </th>

                  <td className="px-4 py-2 tabular-nums text-slate-900">
                    {row.value.toLocaleString('id-ID')}
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

/**
 * rowsOf menyusun kedua baris ringkas dari bentuk tab dan pencacahnya.
 *
 * Tab dikenali dari ADA-TIDAKNYA kolom balasan padanya, bukan dari kodenya — alasan yang
 * sama dipakai lencana di bilah tab: kode tab dapat berubah di server tanpa mengubah
 * artinya, dan baris yang tertukar akan menyatakan kebalikan dari yang benar.
 *
 * Tab yang tidak ditemukan DILEWATI, bukan digambar tanpa tujuan. Baris ringkas yang tidak
 * membuka apa pun adalah tombol yang diam saat ditekan.
 */
function rowsOf(tabs: Tab[], summary: Summary): Row[] {
  const answered = tabs.find((tab) => showsReply(tab))
  const notAnswered = tabs.find((tab) => !showsReply(tab))

  const rows: Row[] = []

  if (answered) {
    rows.push({
      tab: answered.kode,
      label: answered.nama,
      value: summary.sudah_dijawab,
      color: COLOR_ANSWERED,
    })
  }

  if (notAnswered) {
    rows.push({
      tab: notAnswered.kode,
      label: notAnswered.nama,
      value: summary.belum_dijawab,
      color: COLOR_NOT_ANSWERED,
    })
  }

  return rows
}

/** showsReply menyatakan sebuah tab menggambar kolom balasan — penanda tab "Sudah Dijawab". */
function showsReply(tab: Tab): boolean {
  return tab.kolom.some((column) => column.kunci === 'jawaban_terakhir')
}
