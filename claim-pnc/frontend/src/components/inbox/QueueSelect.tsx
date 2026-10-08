type Props = {
  tabs: { kode: string; nama: string; terhalang: boolean }[]
  /** Kode antrean yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
  /** `id` elemen `<select>`. */
  id: string
  /** Nama dropdown bagi pembaca layar. */
  label: string
}

/**
 * Pemilih antrean berbentuk DROPDOWN — bentuknya di Pega produksi untuk layar Claim
 * Treaty Prop dan Non Prop: satu `<select>` di atas grid, bukan deretan tab (`D-13`).
 *
 * Labelnya disembunyikan dari mata, bukan dihilangkan. Pega tidak menggambar label apa
 * pun di atas dropdown-nya, tetapi dropdown tanpa nama sama sekali hanya dibacakan sebagai
 * "combo box" oleh pembaca layar.
 *
 * "Choose" adalah entri kosong bawaan dropdown Pega (`pyHasNoSelection=true`,
 * `pyNoSelectionText="Choose"`), bukan sebuah antrean.
 *
 * Penanda "belum tersedia" ikut masuk ke TEKS pilihannya, bukan digambar sebagai lencana
 * di sampingnya. Isi `<option>` hanya boleh berupa teks, dan keadaan itu harus diketahui
 * SEBELUM dipilih — bukan sesudahnya.
 */
export function QueueSelect({ tabs, active, onSelect, id, label }: Readonly<Props>) {
  return (
    <div>
      <label htmlFor={id} className="sr-only">
        {label}
      </label>
      <select
        id={id}
        value={active}
        onChange={(event) => onSelect(event.target.value)}
        className={[
          'w-full max-w-xs rounded-kontrol border border-slate-300 bg-white px-3 py-2',
          'text-sm text-slate-900 shadow-lembut',
          'transition-[border-color,box-shadow] duration-150 ease-halus',
          'focus:border-blue-500 focus:outline-none focus-visible:ring-4',
          'focus-visible:ring-blue-500/20',
        ].join(' ')}
      >
        <option value="">Choose</option>

        {tabs.map((tab) => (
          <option key={tab.kode} value={tab.kode}>
            {tab.terhalang ? `${tab.nama} — belum tersedia` : tab.nama}
          </option>
        ))}
      </select>
    </div>
  )
}
