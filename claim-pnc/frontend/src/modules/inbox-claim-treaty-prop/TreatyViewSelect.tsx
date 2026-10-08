import { QueueSelect } from '@/components/inbox/QueueSelect'

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
  // "Choose" menampilkan antrean yang SAMA dengan "Prop Treaty-in Admin", dan itu perilaku
  // Pega — bukan cacat: keduanya bernilai `CARI1` kosong, sehingga pemeriksaan yang memilih
  // antrean teknik gagal pada keduanya. Karena itu kotaknya kembali menunjuk "Prop Treaty-in
  // Admin" sesudah dipilih, alih-alih menggantung pada pilihan yang tidak menyaring apa pun.
  return (
    <QueueSelect
      tabs={tabs}
      active={active}
      onSelect={onSelect}
      id="antrean-treaty-prop"
      label="Antrean klaim treaty proporsional"
    />
  )
}
