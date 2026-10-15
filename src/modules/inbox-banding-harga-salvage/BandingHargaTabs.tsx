import type { SummaryRow, Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode tab yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void

  /**
   * Jumlah baris tiap antrean, untuk lencana di sebelah nama tabnya.
   *
   * Lencananya digambar HANYA bila antreannya berisi. Angka yang belum tiba dan angka nol
   * sama-sama tidak dilencanai — lihat catatan di dalam `map` di bawah.
   */
  counts?: SummaryRow[]
}

/**
 * Bilah tab Inbox Banding Harga Salvage.
 *
 * # Apa yang digantikan
 *
 * DUA kontainer bersyarat pada `Section/InboxReqSalvageASM-Section.xml`, yang di sana
 * dinyalakan bergantian oleh sepasang properti:
 *
 *   tempQuery.FlagReject==1 && tempQuery.FlagASO==1   grid Request Banding Harga
 *   tempQuery.FlagReject=1  && tempQuery.FlagASO=2    grid History Cheker
 *
 * Keduanya memanggil ulang activity `SetReqSalvage_Act` dengan parameter `tipe` yang berbeda.
 * Judul tabnya diambil dari kolom "Status Salvage" tabel ringkas, dan salah ketik "Cheker"
 * dipertahankan karena itulah yang dibaca pengguna hari ini (`D-13`).
 *
 * # Kenapa jumlahnya ADA DI SINI, bukan di tabel tersendiri
 *
 * Semula angka itu digambar tabel ringkas "Status Salvage / Jumlah" tepat di atas bilah ini,
 * dengan kedua barisnya dapat diklik untuk berpindah antrean. Akibatnya DUA kendali berlabel
 * SAMA PERSIS berdiri berurutan, untuk satu sakelar yang sama.
 *
 * Tabel itu dibuang (Work Owner, 2026-10-05), dan pemeriksaan ke Pega membenarkannya dua kali:
 *
 *   - `Section/InboxReqSalvageASM` memuat NOL "Jumlah" dan nol "Status Salvage" — tabel itu
 *     milik layar Inbox Salvage (`MENU_ID 71`), bukan layar ini.
 *   - `Activity/GCNMCountRequestSalvage_act` yang memasok angkanya tidak dipanggil harness
 *     maupun section mana pun.
 *
 * Angkanya tetap berguna — "berapa yang menunggu saya" adalah hal pertama yang dicari komite —
 * sehingga ia pindah menjadi lencana di sebelah nama tabnya. Satu kendali, dua pekerjaan.
 */
export function BandingHargaTabs({ tabs, active, onSelect, counts = [] }: Props) {
  return (
    /*
      Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Dua tab memang
      muat di hampir setiap layar, tetapi melipatnya menyembunyikan antrean mana saja yang
      tersedia — hal pertama yang ingin dilihat komite saat membuka layar.
    */
    <div className="overflow-x-auto" role="tablist" aria-label="Antrean banding harga salvage">
      <div className="flex min-w-max items-center gap-1.5 border-b border-slate-200 pb-px">
        {tabs.map((tab) => {
          const selected = tab.kode === active
          /*
            Lencana hanya digambar bila antreannya BERISI.

            Dua keadaan sengaja tidak dilencanai, dan keduanya karena alasan yang sama —
            lencana itu menyampaikan sesuatu yang tidak perlu disampaikan:

              belum tiba   angkanya belum dimuat; "0" akan membuat antrean yang berisi
                           tampak kosong — kelas cacat `GETSELISIHJAM` (`D-49` butir 10)
              nol          tabelnya di bawah sudah menyatakannya kosong, dan lencana "0"
                           hanya menambah benda untuk dibaca (Work Owner, 2026-10-05)

            Keduanya menghasilkan `undefined`, sehingga tabnya tergambar apa adanya.
          */
          const terhitung = counts.find((baris) => baris.tab === tab.kode)?.jumlah
          const jumlah = terhitung !== undefined && terhitung > 0 ? terhitung : undefined
          return (
            <button
              key={tab.kode}
              type="button"
              role="tab"
              aria-selected={selected}
              /*
                Tanpa `title`: isinya kalimat yang SAMA dengan keterangan tab, dan keterangan
                itu sudah dibuang dari layar (Work Owner, 2026-10-05). Membiarkannya berarti
                kalimat yang baru saja dihapus muncul kembali begitu tabnya disentuh tetikus.

                Tidak ada yang hilang bagi pembaca layar: nama tabnya sendiri — "Request
                Banding Harga" dan "History Cheker" — sudah menyatakan isinya, dan keduanya
                caption Pega apa adanya.
              */
              onClick={() => onSelect(tab.kode)}
              className={[
                'rounded-t-kontrol border-b-2 px-3.5 py-2.5',
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
                Spasi ini BUKAN hiasan. Tanpanya, nama aksesibilitas tabnya menjadi
                "History Cheker4" — jarak visual datang dari `ml-2`, yang tidak menyisipkan
                teks apa pun. Pembaca layar lalu menyebut angkanya menempel pada kata
                terakhir.
              */}
              {jumlah !== undefined && ' '}
              {jumlah !== undefined && (
                /*
                  Lencana ini TIDAK diberi `aria-hidden`: jumlahnya bagian dari nama
                  aksesibilitas tabnya, sehingga pembaca layar menyebut "Request Banding
                  Harga 3" — persis yang dibaca orang yang melihatnya.
                */
                <span
                  className={[
                    'ml-2 rounded-full px-2 py-0.5 text-xs font-semibold tabular-nums',
                    selected ? 'bg-blue-100 text-blue-800' : 'bg-slate-100 text-slate-600',
                  ].join(' ')}
                >
                  {jumlah}
                </span>
              )}
            </button>
          )
        })}
      </div>
    </div>
  )
}
