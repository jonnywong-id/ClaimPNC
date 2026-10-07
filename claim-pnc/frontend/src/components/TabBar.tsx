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
  keterangan?: string
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
 * `SalvageTabs` pada modul Inbox Salvage mendahuluinya dan TIDAK diubah: modul itu sudah
 * selesai, dan menyatukan keduanya sekarang berarti menyentuh layar yang tidak sedang
 * dikerjakan.
 *
 * # Digulir menyamping, bukan dilipat
 *
 * Pada layar sempit bilahnya digulir. Melipatnya menjadi dropdown menyembunyikan daftar
 * mana saja yang tersedia — dan pada layar yang tabnya menentukan populasi baris, bukan
 * sekadar penyaring tampilan, itu menyembunyikan struktur layarnya sendiri.
 */
export function TabBar({ tabs, active, onSelect, label }: Readonly<Props>) {
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
