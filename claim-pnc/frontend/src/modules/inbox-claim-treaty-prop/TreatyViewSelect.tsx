import type { Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode antrean yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
}

/**
 * Pemilih antrean pada layar Inbox Claim Treaty Prop.
 *
 * # Kenapa DROPDOWN, bukan bilah tab
 *
 * Karena itulah bentuknya di Pega produksi: satu `<select>` di atas grid, bukan deretan tab.
 * `D-13` menetapkan tampilan meniru Pega supaya pengguna tidak perlu belajar ulang, dan
 * bentuk kontrolnya termasuk di dalamnya. Versi sebelumnya di sini memakai bilah tab —
 * diganti atas permintaan Work Owner 2026-09-29 setelah layar produksinya dibandingkan
 * berdampingan.
 *
 * # Dari mana daftar pilihannya
 *
 * Bukan dari judul kontainer grid, melainkan dari halaman klipboard `Operator.pxResults`
 * yang disusun `Data Transform/FilterWorkBasket_Act-DT.xml`:
 *
 *	CARI2 (label)             CARI1 (nilai)        kapan muncul
 *	"Prop Treaty-in Admin"    ""                   selalu
 *	"Prop Treaty-in Teknik"   "TreatyinPNCTeknik"  hanya bila pemanggil ANGGOTA antrean itu
 *	"Choose"                  ""                   entri kosong bawaan dropdown
 *
 * Teks pilihan itu BERBEDA dari judul grid di bawahnya, dan keduanya tampil bersamaan:
 * dropdown bertuliskan "Prop Treaty-in Admin", grid di bawahnya berjudul "Work List
 * Treatyin Propotional".
 *
 * Memilih menjalankan `postValue` lalu `refresh` dengan activity `GetDataTreatyin_Act`. Di
 * sini padanannya adalah permintaan ulang daftar ke server — bukan pemuatan ulang halaman.
 *
 * # Yang berbeda dari Pega, dan disadari
 *
 * Pilihan "Prop Treaty-in Teknik" di Pega hanya muncul bagi petugas yang terdaftar sebagai
 * anggota antrean `TreatyinPNCTeknik`. Keanggotaan itu tidak ada di basis data maupun di
 * export (`TKT-F3-004`), sehingga di sini pilihannya selalu terlihat. Menyembunyikannya
 * berdasarkan tebakan akan membuat petugas yang seharusnya melihatnya justru tidak
 * melihatnya sama sekali.
 *
 * "Komite Treaty ASM" di Pega bukan pilihan pada dropdown ini melainkan pada DROPDOWN KEDUA
 * di mode komite. Ia dibawa ke sini sebagai pilihan yang terhalang, dengan alasan yang sama.
 *
 * # Antrean terhalang tetap dapat dipilih
 *
 * Keputusan Work Owner 2026-09-21. Ia digambar dengan penanda pada teks pilihannya, dan
 * memilihnya menampilkan alasan beserta pemiliknya — bukan tabel kosong yang terbaca sebagai
 * "tidak ada pekerjaan". Menonaktifkan pilihannya akan membuat pengguna tidak pernah tahu
 * kenapa.
 */
export function TreatyViewSelect({ tabs, active, onSelect }: Readonly<Props>) {
  return (
    <div>
      {/*
        Labelnya disembunyikan dari mata, bukan dihilangkan. Pega tidak menggambar label
        apa pun di atas dropdown-nya, dan `D-13` menetapkan tampilan mengikuti Pega —
        tetapi dropdown tanpa nama sama sekali hanya dibacakan sebagai "combo box" oleh
        pembaca layar, sehingga pengguna tidak punya cara tahu apa yang sedang dipilihnya.
      */}
      <label htmlFor="antrean-treaty-prop" className="sr-only">
        Antrean klaim treaty proporsional
      </label>
      <select
        id="antrean-treaty-prop"
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

          Memilihnya menampilkan antrean yang SAMA dengan "Prop Treaty-in Admin", dan itu
          perilaku Pega — bukan cacat: keduanya bernilai `CARI1` kosong, sehingga
          pemeriksaan yang memilih antrean teknik gagal pada keduanya. Karena itu kotaknya
          kembali menunjuk "Prop Treaty-in Admin" sesudah dipilih, alih-alih menggantung
          pada pilihan yang tidak menyaring apa pun.
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
