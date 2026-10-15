import type { Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode tab yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
}

/**
 * Bilah tab Inbox RCL/PUCL.
 *
 * # Apa yang digantikan
 *
 * `Section/InputPUCL-RCL_Section-Section.xml`, yang menyertakan tiga SUB_SECTION berurutan:
 * `InboxCetakSuratPUCLRCL_Section`, `InboxKelengkapanDocPUCLRCL_Section`, dan
 * `InboxAJSMSIG_Section`. Ketiga judulnya ada di sana apa adanya sebagai
 * `pyCaption Cetak Surat`, `pyCaption Kelengkapan Dokumen`, dan `pyCaption Klaim MSIG` —
 * tidak satu pun dikarang (`D-13`).
 *
 * # Kenapa keterangan tab di sini lebih penting daripada di layar lain
 *
 * Karena ketiga tab punya KOLOM YANG IDENTIK. Di layar lain, pengguna yang tersesat ke tab
 * yang salah segera menyadarinya dari kolom yang berbeda; di sini tidak ada petunjuk
 * semacam itu sama sekali. Yang membedakan ketiganya hanyalah perjalanan surat PUCL — belum
 * dicetak, sudah dicetak, dan jalur MSIG — dan itu hanya terbaca dari keterangannya.
 *
 * Karena itu keterangan tab ikut digambar sebagai `title`, bukan hanya di bawah bilah.
 */
export function RCLPUCLTabs({ tabs, active, onSelect }: Props) {
  return (
    /*
      Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Melipatnya
      menyembunyikan antrean mana saja yang tersedia — dan pada layar yang ketiga tabnya
      tampak sama, daftar tab adalah satu-satunya hal yang menjelaskan strukturnya.
    */
    <div
      className="overflow-x-auto"
      role="tablist"
      aria-label="Antrean klaim RCL/PUCL"
    >
      <div className="flex min-w-max items-center gap-1.5 border-b border-slate-200 pb-px">
        {tabs.map((tab) => {
          const selected = tab.kode === active
          return (
            <button
              key={tab.kode}
              type="button"
              role="tab"
              aria-selected={selected}
              title={tab.terhalang ? tab.alasan_terhalang : tab.keterangan}
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
                Penanda dibaca pembaca layar pula, bukan hanya terlihat. Tab yang terhalang
                adalah keadaan yang harus diketahui SEBELUM diklik, bukan sesudahnya.

                Tidak ada tab terhalang di layar ini hari ini — tab "Klaim MSIG" yang
                kemungkinan kosong TIDAK ditandai begitu, karena kosongnya adalah jawaban,
                bukan ketidakmampuan menjawab. Penandanya tetap digambar supaya penambahan
                tab terhalang kelak cukup dilakukan di backend.
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
