import type { Tab } from './types'

type Props = {
  tabs: Tab[]

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
 * pencacah itu menjadi tab.
 *
 * # Angkanya TIDAK digambar pada tabnya
 *
 * Pega memang menghitungnya — `Activity/CountDashbroardManager` menjalankan sepuluh kueri —
 * tetapi angkanya ditampilkan sebagai tabel tersendiri berkolom `Status` dan `Jumlah`
 * (`Section/InboxManager_Sec`, grid `TempCountDashboard.pxResults`), BUKAN sebagai lencana
 * pada tombolnya.
 *
 * Lencana itu sempat digambar di sini dan dicabut atas permintaan Work Owner (2026-10-08).
 * Keterangan "sumber tidak terbaca" yang sempat menempel sebagai tooltip ikut dicabut pada
 * hari yang sama: ia ditujukan ke tim teknis, bukan ke pengguna, dan sebabnya keadaan
 * pengembangan — tabelnya memang belum ada. Layar ini karena itu TIDAK lagi membaca pencacah
 * sama sekali.
 *
 * # Hanya EMPAT tab digambar di sini, bukan tiga belas
 *
 * Kesembilan antrean persetujuan adalah ANAK "Approval Master", bukan tab sejajar dengannya —
 * `Activity/CountDashbroardManager` menulis empat pencacah pertama ke
 * `TempCountDashboard.pxResults(<APPEND>)` dan sembilan sisanya ke
 * `.pxResults(4).pxResults(<APPEND>)`, dan gridnya ber-`pyRepeatDirection` TreeGrid.
 * Menggambar ketiga belasnya berjajar membuat sembilan antrean tampak setara dengan induknya
 * sendiri.
 *
 * Saat sebuah antrean terbuka, yang tersorot di bilah ini tetap INDUKNYA — supaya posisi
 * pengguna di dalam jenjang tidak hilang begitu ia masuk ke salah satu antrean.
 */
export function ManagerTabs({ tabs, active, onSelect }: Props) {

  /*
    Sebuah tab dianggap anak hanya bila induknya BENAR-BENAR ada di daftar yang diterima.

    Ini bukan kehati-hatian berlebihan: daftar tab disaring lini bisnis di server, dan tab yang
    induknya tidak ikut terkirim akan hilang sama sekali dari layar bila jenjangnya dipercaya
    begitu saja — tidak di bilah atas, dan tidak pula di dalam induk yang tidak ada.
  */
  const present = new Set(tabs.map((tab) => tab.kode))
  const isChild = (tab: Tab) => Boolean(tab.induk) && present.has(tab.induk ?? '')

  const top = tabs.filter((tab) => !isChild(tab))

  const activeTab = tabs.find((tab) => tab.kode === active)
  const activeTop = activeTab && isChild(activeTab) ? (activeTab.induk ?? active) : active

  return (
    /*
      Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Melipatnya
      menyembunyikan antrean mana saja yang menunggu — hal pertama yang ingin dilihat
      penyelia saat membuka layar.
    */
    <div className="overflow-x-auto" role="tablist" aria-label="Bagian Inbox Manager">
      <div className="flex min-w-max items-center gap-1.5 border-b border-slate-200 pb-px">
        {top.map((tab) => {
          const selected = tab.kode === activeTop

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
            </button>
          )
        })}
      </div>
    </div>
  )
}

type SubProps = {
  /** Tab induknya — dipakai sebagai judul kelompok dan sebagai jalan kembali ke ringkasan. */
  parent: Tab
  tabs: Tab[]
  active: string
  onSelect: (code: string) => void
}

/**
 * Bilah antrean DI DALAM "Approval Master".
 *
 * # Kenapa ada bilah kedua, padahal kartunya sudah ada
 *
 * Kartu ringkasan hanya jalan MASUK. Setelah sebuah antrean terbuka, tanpa bilah ini satu-
 * satunya cara berpindah ke antrean lain adalah kembali dulu ke ringkasan — sembilan antrean
 * berarti dua klik untuk tiap perpindahan, pada layar yang justru dipakai untuk menyapu
 * seluruh antrean berurutan.
 *
 * # Bentuknya pil, bukan tab bergaris
 *
 * Supaya ia terbaca sebagai tingkat kedua, bukan sebagai bilah tab kedua yang sederajat.
 * Perbedaan bentuk inilah yang menyatakan jenjangnya; menyamakan gayanya dengan bilah atas
 * akan mengulang persis kekeliruan yang diperbaiki di sini.
 */
export function QueueSubTabs({ parent, tabs, active, onSelect }: SubProps) {

  return (
    <div className="rounded-kontrol border border-slate-200 bg-slate-50 p-3">
      <p className="mb-2 text-xs font-medium tracking-wide text-slate-500 uppercase">
        Antrean di dalam {parent.nama}
      </p>

      {/*
        TIDAK ada pil "Ringkasan" — Pega tidak punya, dan bilahnya memuat tepat delapan
        section yang disertakan `InboxManager_Section2`. Jalan kembali ke ringkasan adalah
        tab "Approval Master" di bilah atas, yang memang tetap tersorot selama salah satu
        antreannya terbuka.
      */}
      <div className="overflow-x-auto" role="tablist" aria-label={`Antrean ${parent.nama}`}>
        <div className="flex min-w-max items-center gap-1.5">
          {tabs.map((tab) => {
            return (
              <SubTab
                key={tab.kode}
                label={tab.nama}
                title={tab.keterangan}
                selected={tab.kode === active}
                onClick={() => onSelect(tab.kode)}
              />
            )
          })}
        </div>
      </div>
    </div>
  )
}

function SubTab({
  label,
  title,
  selected,
  onClick,
}: {
  label: string
  title: string
  selected: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={selected}
      title={title}
      onClick={onClick}
      className={[
        'flex items-center gap-2 rounded-full border px-3 py-1.5',
        'text-sm font-medium whitespace-nowrap',
        'transition-[color,border-color,background-color] duration-150 ease-halus',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
        selected
          ? 'border-blue-600 bg-blue-600 text-white'
          : 'border-slate-300 bg-white text-slate-700 hover:border-blue-300 hover:bg-blue-50 hover:text-blue-800',
      ].join(' ')}
    >
      {label}
    </button>
  )
}
