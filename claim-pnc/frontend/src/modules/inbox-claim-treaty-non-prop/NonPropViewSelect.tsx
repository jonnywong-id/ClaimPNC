import type { Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode antrean yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
}

/**
 * Pemilih antrean pada layar Inbox Claim Treaty Non Prop.
 *
 * # Kenapa DROPDOWN, bukan bilah tab
 *
 * Karena itulah bentuknya di Pega: satu `<select>` di antara judul layar dan grid, bukan
 * deretan tab. `D-13` menetapkan tampilan meniru Pega supaya pengguna tidak perlu belajar
 * ulang, dan bentuk kontrolnya termasuk di dalamnya. Versi sebelumnya di sini memakai bilah
 * tab — diganti setelah layar Pega dibandingkan berdampingan, sama seperti yang sudah
 * dilakukan pada layar saudaranya, Treaty Prop.
 *
 * # Dari mana daftar pilihannya
 *
 * Dari `Activity/GetWorkCNP_Act-Act.xml`, yang mengisi halaman klipboard
 * `ListWorkBasket.pxResults`:
 *
 *	CARI1 (label sekaligus nilai)   kapan muncul
 *	"Treaty-In Admin"               selalu
 *	"Treaty-In Teknik"              selalu
 *	"Choose"                        entri kosong bawaan dropdown
 *
 * Perhatikan label dan nilainya SATU properti yang sama di layar ini — prakondisi
 * langkah-langkah activity itu membandingkan `ParamInboxCNP.CARI22 == "Treaty-In Teknik"`,
 * yaitu teks yang dibaca pengguna itu sendiri. Itu berbeda dari layar Treaty Prop, yang
 * memisahkan label (`CARI2`) dari nilainya (`CARI1`).
 *
 * Teks pilihan itu BERBEDA dari judul grid di bawahnya, dan keduanya tampil bersamaan:
 * dropdown bertuliskan "Treaty-In Teknik", grid di bawahnya berjudul "Work Treatyin Non
 * Propotional Teknik".
 *
 * # Antrean komite: digambar, tetapi bukan pilihan di Pega
 *
 * Di Pega, grid komite TIDAK dipilih lewat dropdown ini. Ia muncul sendiri bila
 * `GetWorkCNP_Act` menyetel `InputData.CARI13 = "tampil"`, yaitu bila pemanggil terdaftar
 * sebagai anggota komite. Pemeriksaan peran itu belum ada (`TKT-F3-004`), sehingga di sini
 * ia dibawa sebagai pilihan yang terhalang — perlakuan yang sama dengan "Komite Treaty ASM"
 * pada layar Treaty Prop.
 *
 * # Antrean terhalang tetap dapat dipilih
 *
 * Keputusan Work Owner 2026-09-21. Ia digambar dengan penanda pada teks pilihannya, dan
 * memilihnya menampilkan alasan beserta pemiliknya — bukan tabel kosong yang terbaca sebagai
 * "tidak ada pekerjaan". Menonaktifkan pilihannya akan membuat pengguna tidak pernah tahu
 * kenapa.
 */
export function NonPropViewSelect({ tabs, active, onSelect }: Readonly<Props>) {
  return (
    <div>
      {/*
        Labelnya disembunyikan dari mata, bukan dihilangkan. Pega tidak menggambar label
        apa pun di atas dropdown-nya, dan `D-13` menetapkan tampilan mengikuti Pega —
        tetapi dropdown tanpa nama sama sekali hanya dibacakan sebagai "combo box" oleh
        pembaca layar, sehingga pengguna tidak punya cara tahu apa yang sedang dipilihnya.
      */}
      <label htmlFor="antrean-treaty-non-prop" className="sr-only">
        Antrean klaim treaty non-proporsional
      </label>
      <select
        id="antrean-treaty-non-prop"
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
        {/*
          "Choose" adalah entri kosong bawaan dropdown Pega (`pyHasNoSelection=true`,
          `pyNoSelectionText="Choose"`), bukan sebuah antrean.

          Memilihnya mengembalikan layar ke antrean bawaan, yaitu "Treaty-In Admin". Di Pega
          ia tidak menjalankan pemilihan apa pun sehingga yang tampil adalah grid yang sudah
          dimuat saat layar dibuka — dan grid itu memang grid Admin.
        */}
        <option value="">Choose</option>

        {tabs.map((tab) => (
          <option key={tab.kode} value={tab.kode}>
            {/*
              Penanda ikut masuk ke TEKS pilihannya, bukan digambar sebagai lencana di
              sampingnya. Isi `<option>` hanya boleh berupa teks, dan keadaan "belum
              tersedia" harus diketahui SEBELUM dipilih — bukan sesudahnya.
            */}
            {tab.terhalang ? `${tab.nama} — belum tersedia` : tab.nama}
          </option>
        ))}
      </select>
    </div>
  )
}
