import type { Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode tab yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
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
 * # Kenapa tanpa lencana jumlah
 *
 * Karena jumlahnya sudah digambar tabel ringkas tepat di atas bilah ini, dan tabel itu pula
 * yang menjadi navigasi di layar lama. Mengulangnya di dua tempat berarti dua angka yang dapat
 * berselisih saat salah satu belum dimuat ulang.
 */
export function BandingHargaTabs({ tabs, active, onSelect }: Readonly<Props>) {
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
          return (
            <button
              key={tab.kode}
              type="button"
              role="tab"
              aria-selected={selected}
              title={tab.keterangan}
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
            </button>
          )
        })}
      </div>
    </div>
  )
}
