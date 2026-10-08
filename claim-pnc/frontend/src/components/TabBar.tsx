/**
 * Satu butir pada bilah daftar.
 *
 * Bentuknya sengaja MINIM — kode, nama, keterangan — supaya modul apa pun dapat
 * memetakan tipe daftarnya sendiri ke sini tanpa mengubah tipe itu. Modul yang berbeda
 * membawa isian yang berbeda-beda pada daftarnya, dan memaksa mereka memakai satu tipe
 * bersama akan menarik seluruh isian itu ke dalam komponen tampilan.
 */
export type TabItem = {
  kode: string
  nama: string
  /** Kalimat penjelas; digambar sebagai `title` pada butirnya. Boleh kosong. */
  keterangan?: string | undefined
  /**
   * Penanda kecil di samping nama, mis. "belum tersedia" untuk tab yang terhalang.
   *
   * Penanda dibaca pembaca layar pula, bukan hanya terlihat: tab yang terhalang adalah
   * keadaan yang harus diketahui SEBELUM diklik, bukan sesudahnya.
   */
  badge?: string | undefined
}

type Props = {
  tabs: TabItem[]
  /** Kode daftar yang sedang terbuka. */
  active: string
  onSelect: (kode: string) => void

  /**
   * Nama bilahnya bagi pembaca layar.
   *
   * Diperlukan begitu satu halaman memuat lebih dari satu bilah, dan berguna bahkan bila
   * hanya ada satu: tanpa nama, ia hanya dibacakan sebagai "daftar tab".
   */
  label: string

  /**
   * Isi butir disusun flex (nama + penanda sejajar). Bawaannya `true`; beberapa layar
   * inbox menggambar butirnya tanpa susunan flex, dan bentuk itu dipertahankan.
   */
  flexItems?: boolean | undefined
}

/**
 * TabBar adalah bilah pemilih daftar yang dipakai layar-layar berisi beberapa antrean.
 *
 * # Kenapa ia komponen bersama, bukan milik satu modul
 *
 * Karena dua layar memakainya dengan bentuk yang sama persis — Inbox PLA, DLA, Pre DLA
 * dan Inbox PLA DLA — dan keduanya dibangun bersamaan. Menyalinnya ke masing-masing
 * berarti dua tafsir tentang bagaimana sebuah tab terlihat saat terpilih, dan pada
 * layar yang bersebelahan di menu perbedaan itu langsung terlihat.
 *
 * Bilah tab layar-layar inbox lain (`InboxTabs`, `SalvageTabs`, `RCLPUCLTabs`, dan
 * saudaranya) kini juga digambar oleh komponen ini — masing-masing tinggal pembungkus
 * tipis yang menetapkan label, susunan butir, dan penandanya.
 *
 * # Digulir menyamping, bukan dilipat
 *
 * Pada layar sempit bilahnya digulir. Melipatnya menjadi dropdown menyembunyikan daftar
 * mana saja yang tersedia — dan pada layar yang tabnya menentukan populasi baris, bukan
 * sekadar penyaring tampilan, itu menyembunyikan struktur layarnya sendiri.
 */
export function TabBar({ tabs, active, onSelect, label, flexItems = true }: Readonly<Props>) {
  return (
    <div className="overflow-x-auto" role="tablist" aria-label={label}>
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
                flexItems
                  ? 'flex items-center gap-2 rounded-t-kontrol border-b-2 px-3.5 py-2.5'
                  : 'rounded-t-kontrol border-b-2 px-3.5 py-2.5',
                'text-sm font-medium whitespace-nowrap',
                'transition-[color,border-color,background-color] duration-150 ease-halus',
                'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
                selected
                  ? 'border-blue-600 text-blue-700'
                  : 'border-transparent text-slate-600 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900',
              ].join(' ')}
            >
              {tab.nama}
              {tab.badge ? (
                <span className="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-normal text-amber-900">
                  {tab.badge}
                </span>
              ) : null}
            </button>
          )
        })}
      </div>
    </div>
  )
}
