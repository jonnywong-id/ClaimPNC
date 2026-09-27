import type { Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode tab yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
}

/**
 * Bilah tab Report KPI PNC.
 *
 * # Apa yang digantikan
 *
 * Ketiga layout group pada `Section/ReportKPI_Section-Section.xml`, yang judulnya terbaca
 * apa adanya sebagai `pyTitle`: **KPI PIC Teknik**, **KPI Adjuster**, dan **KPI Admin**.
 * Tidak satu pun dikarang (`D-13`).
 *
 * # Kenapa tab yang belum dibangun tetap digambar
 *
 * Karena menyembunyikannya membuat layar ini terlihat seperti layar satu-tab, dan pengguna
 * yang terbiasa dengan tiga tab di Pega akan melaporkannya sebagai fitur yang hilang.
 * Keputusan Work Owner 2026-09-18 untuk butir menu berlaku sama di sini: kemajuan migrasi
 * terbaca langsung dari layar.
 *
 * Alasan terhalangnya ikut digambar sebagai `title` DAN dapat dibuka di bawah bilah, bukan
 * hanya sebagai lencana "belum tersedia" — yang membacanya adalah penguji yang sedang
 * memutuskan apakah ini cacat atau bukan, dan lencana saja tidak menjawab itu.
 */
export function ReportKPITabs({ tabs, active, onSelect }: Props) {
  return (
    /*
      Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Melipatnya
      menyembunyikan tab mana saja yang ada — dan pada layar yang dua dari tiga tabnya
      belum dibangun, daftar tab adalah satu-satunya hal yang menjelaskan cakupannya.
    */
    <div className="overflow-x-auto" role="tablist" aria-label="Kelompok laporan KPI">
      <div className="flex min-w-max items-center gap-1.5 border-b border-slate-200 pb-px">
        {tabs.map((tab) => {
          const selected = tab.kode === active
          return (
            <button
              key={tab.kode}
              type="button"
              role="tab"
              aria-selected={selected}
              // Tab terhalang tidak dapat diklik. Membiarkannya dapat dipilih lalu
              // menampilkan layar kosong akan terbaca sebagai kerusakan, bukan sebagai
              // pekerjaan yang belum selesai.
              disabled={tab.terhalang}
              title={tab.terhalang ? tab.alasan_terhalang : undefined}
              onClick={() => onSelect(tab.kode)}
              className={[
                'flex items-center gap-2 rounded-t-kontrol border-b-2 px-3.5 py-2.5',
                'text-sm font-medium whitespace-nowrap',
                'transition-[color,border-color,background-color] duration-150 ease-halus',
                'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
                tab.terhalang
                  ? 'cursor-not-allowed border-transparent text-slate-400'
                  : selected
                    ? 'border-blue-600 text-blue-700'
                    : 'border-transparent text-slate-600 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900',
              ].join(' ')}
            >
              {tab.judul}

              {/*
                Penanda dibaca pembaca layar pula, bukan hanya terlihat. Tab yang belum
                dibangun adalah keadaan yang harus diketahui SEBELUM diklik.
              */}
              {tab.terhalang && (
                <span className="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-normal text-amber-900">
                  belum tersedia
                </span>
              )}
            </button>
          )
        })}
      </div>
    </div>
  )
}
