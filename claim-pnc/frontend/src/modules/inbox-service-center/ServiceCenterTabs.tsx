import type { Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode tab yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
}

/**
 * Bilah tab Inbox Service Center.
 *
 * # Apa yang digantikan
 *
 * Wadah `TABBED` pada `Section/BrowseServiceCenter-Section.xml`, yang di sana berupa empat
 * kontainer grid — masing-masing memanggil ulang activity `DataServiceCenter` dengan nilai
 * `stsapprove` yang berbeda. Judul tabnya dipertahankan apa adanya, termasuk yang berbahasa
 * Inggris seperti "Waiting Approval", karena `D-13` menetapkan tampilan meniru Pega supaya
 * pengguna tidak perlu belajar ulang.
 *
 * # Kenapa tanpa lencana jumlah
 *
 * Sistem lama pun tidak menampilkannya, dan menghadirkannya berarti menjalankan keempat
 * kueri sekaligus setiap kali layar dibuka — terhadap tabel yang tumbuh tanpa batas.
 */
export function ServiceCenterTabs({ tabs, active, onSelect }: Readonly<Props>) {
  return (
    /*
      Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Empat tab
      memang muat di kebanyakan layar, tetapi judul sepanjang "Waiting Approval" membuat
      bilahnya melampaui lebar ponsel — dan melipatnya menyembunyikan antrean mana saja
      yang tersedia, yaitu hal pertama yang ingin dilihat petugas saat membuka layar.
    */
    <div className="overflow-x-auto" role="tablist" aria-label="Antrean Inbox Service Center">
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
