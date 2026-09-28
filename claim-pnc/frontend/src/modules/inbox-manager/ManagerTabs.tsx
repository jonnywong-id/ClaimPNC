import type { Counter, Tab } from './types'

type Props = {
  tabs: Tab[]
  counters: Counter[]

  /** Kode tab yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
}

/**
 * Bilah tab Inbox Manager.
 *
 * # Apa yang digantikan
 *
 * Tiga belas kontainer pada `Section/InboxManager_Sec-Section.xml`. Ketiganya BUKAN tab di
 * sana melainkan kontainer bersyarat:
 *
 *	<pyContainerVisibleWhen>FlagManager.AlasanKlaim==1</pyContainerVisibleWhen>
 *	…
 *	<pyContainerVisibleWhen>FlagManager.AlasanKlaim==13</pyContainerVisibleWhen>
 *
 * Yang berpindah antar kontainer adalah ringkasan pencacah di kepala layar: mengeklik sebuah
 * pencacah menetapkan `FlagManager.AlasanKlaim`, dan kontainer yang cocok muncul. Di sini
 * keduanya disatukan — pencacahnya menjadi tab, dan angkanya menempel pada tabnya.
 *
 * # Angka pada tab datang dari pencacah, bukan dari daftar
 *
 * Sepuluh pencacah dijalankan `Activity/CountDashbroardManager` sebagai kueri tersendiri, dan
 * di sini pun begitu. Menurunkannya dari panjang daftar akan salah pada tab yang berhalaman,
 * dan mustahil pada tab dashboard yang barisnya bukan pekerjaan.
 *
 * # Tab tanpa angka DIBIARKAN tanpa angka
 *
 * Tiga tab memang tidak punya pencacah di sistem lama — Produktivitas Klaim, Klaim, dan dulu
 * juga Approval Master. Menggambar "0" pada tab yang tidak punya hitungan berarti menyatakan
 * tidak ada apa-apa di sana, padahal isinya justru penuh.
 */
export function ManagerTabs({ tabs, counters, active, onSelect }: Props) {
  const byTab = new Map(counters.map((counter) => [counter.tab, counter]))

  return (
    /*
      Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Melipatnya
      menyembunyikan antrean mana saja yang menunggu — hal pertama yang ingin dilihat
      penyelia saat membuka layar.
    */
    <div className="overflow-x-auto" role="tablist" aria-label="Bagian Inbox Manager">
      <div className="flex min-w-max items-center gap-1.5 border-b border-slate-200 pb-px">
        {tabs.map((tab) => {
          const selected = tab.kode === active
          const counter = byTab.get(tab.kode)

          return (
            <button
              key={tab.kode}
              type="button"
              role="tab"
              aria-selected={selected}
              title={counter?.tidak_tersedia ? counter.tidak_tersedia : tab.keterangan}
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
              <TabBadge counter={counter} selected={selected} />
            </button>
          )
        })}
      </div>
    </div>
  )
}

/**
 * Lencana angka di sebelah judul tab.
 *
 * Tiga keadaan, dan ketiganya terlihat berbeda — karena ketiganya menuntut tindakan yang
 * berbeda:
 *
 *   tidak ada pencacah   tidak digambar apa-apa
 *   sumber tidak terbaca tanda seru, bukan angka
 *   ada angka            angka, dan nol digambar pucat supaya yang menunggu menonjol
 */
function TabBadge({
  counter,
  selected,
}: {
  counter?: Counter | undefined
  selected: boolean
}) {
  if (!counter) return null

  if (counter.tidak_tersedia) {
    return (
      <span
        aria-label="sumber antrean ini sedang tidak dapat dibaca"
        className="rounded-full bg-amber-100 px-1.5 text-xs font-semibold text-amber-800"
      >
        !
      </span>
    )
  }

  const empty = counter.jumlah === 0

  return (
    <span
      className={[
        'rounded-full px-1.5 text-xs font-semibold tabular-nums',
        empty
          ? 'bg-slate-100 text-slate-400'
          : selected
            ? 'bg-blue-100 text-blue-800'
            : 'bg-slate-200 text-slate-700',
      ].join(' ')}
    >
      {counter.jumlah}
    </span>
  )
}
