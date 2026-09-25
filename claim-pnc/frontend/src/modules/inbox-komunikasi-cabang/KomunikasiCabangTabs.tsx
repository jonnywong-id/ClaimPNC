import type { Summary, Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode tab yang sedang terbuka. */
  active: string
  /** Kedua pencacah; lencananya digambar dari sini. */
  summary: Summary | undefined
  onSelect: (code: string) => void
}

/**
 * Bilah tab Inbox Komunikasi Cabang.
 *
 * # Apa yang digantikan, dan apa yang TIDAK ada di layar lama
 *
 * Layar lama TIDAK punya tab sama sekali. `Section/InboxKomunikasi-Section.xml` menggambar
 * kedua grid BERTUMPUK pada satu halaman — yang sudah dijawab di atas, yang belum dijawab di
 * bawah — tanpa kontainer tab apa pun.
 *
 * Pemisahannya menjadi tab dinyatakan lewat `selisih_terencana`, bukan disamarkan. Yang
 * TIDAK berubah adalah isinya, penyaringnya, dan urutannya.
 *
 * # Kenapa lencana angkanya ada di sini, bukan di tempat lain
 *
 * Karena layar lama menggambar kedua angka itu BERSAMAAN di atas kedua grid, sebagai
 * diagram lingkaran berlabel "Answered" dan "Not Answered". Begitu kedua daftar menjadi tab,
 * satu-satunya cara mempertahankan sifat itu — melihat berapa yang menunggu di tab sebelah
 * TANPA berpindah ke sana — adalah menaruh angkanya di bilah tab.
 *
 * Tanpa itu, pemisahan menjadi tab akan MENGHILANGKAN keterangan yang selama ini terlihat
 * sekilas. Itu akan menjadikannya selisih yang merugikan, bukan sekadar selisih bentuk.
 */
export function KomunikasiCabangTabs({ tabs, active, summary, onSelect }: Props) {
  return (
    /*
      Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Melipatnya
      menyembunyikan lencana angkanya, dan angka itulah satu-satunya hal yang menyatakan ada
      pekerjaan di tab sebelah.
    */
    <div className="overflow-x-auto" role="tablist" aria-label="Kotak komunikasi cabang">
      <div className="flex min-w-max items-center gap-1.5 border-b border-slate-200 pb-px">
        {tabs.map((tab) => {
          const selected = tab.kode === active
          const count = countOf(tab, summary)

          return (
            <button
              key={tab.kode}
              type="button"
              role="tab"
              aria-selected={selected}
              title={tab.keterangan}
              onClick={() => onSelect(tab.kode)}
              className={[
                'flex items-center gap-2 rounded-t-kontrol border-b-2 px-3.5 py-2.5',
                'text-sm font-medium whitespace-nowrap',
                'transition-[color,border-color,background-color] duration-150 ease-halus',
                'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
                selected
                  ? 'border-blue-600 text-blue-700'
                  : 'border-transparent text-slate-600 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900',
              ].join(' ')}
            >
              {tab.nama}

              {/*
                Lencana hanya digambar bila angkanya SUDAH diketahui.

                Menggambar "0" selama pencacahnya masih dimuat akan menyatakan tidak ada
                pekerjaan pada tab yang mungkin penuh — dan pada layar yang dibuka untuk
                memeriksa apakah ada yang perlu dijawab, itu kekeliruan yang mahal.
              */}
              {count !== undefined && (
                <span
                  className={[
                    'rounded-full px-2 py-0.5 text-xs font-normal tabular-nums',
                    selected ? 'bg-blue-100 text-blue-900' : 'bg-slate-100 text-slate-700',
                  ].join(' ')}
                >
                  {count}
                </span>
              )}
            </button>
          )
        })}
      </div>
    </div>
  )
}

/**
 * countOf memilih pencacah yang sesuai sebuah tab.
 *
 * Ia memakai KODE tab, bukan urutannya di senarai: urutan dapat berubah di server tanpa
 * mengubah artinya, dan lencana yang tertukar akan menyatakan kebalikan dari yang benar.
 *
 * # Kenapa angkanya dapat tidak sama dengan jumlah baris tabel
 *
 * Karena pencacah memeriksa DUA kolom balasan sementara tabel hanya memeriksa satu, dan
 * pencacah TIDAK menyaring pesan maupun pengirim kosong sementara tabel menyaringnya.
 * Keduanya begitu di layar lama, dan perbedaannya dinyatakan lewat `selisih_terencana`.
 */
function countOf(tab: Tab, summary: Summary | undefined): number | undefined {
  if (summary === undefined) return undefined

  // Kode tab BUKAN kontrak yang dibaca di sini; yang dibaca adalah ada-tidaknya kolom
  // balasan pada tab tersebut. Dengan begitu penambahan tab kelak tidak menuntut suntingan
  // di berkas ini.
  const showsReply = tab.kolom.some((column) => column.kunci === 'jawaban_terakhir')
  return showsReply ? summary.sudah_dijawab : summary.belum_dijawab
}
