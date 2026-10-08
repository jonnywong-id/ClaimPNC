import { QueueSelect } from '@/components/inbox/QueueSelect'

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
  // "Choose" mengembalikan layar ke antrean bawaan, yaitu "Treaty-In Admin". Di Pega ia
  // tidak menjalankan pemilihan apa pun sehingga yang tampil adalah grid yang sudah dimuat
  // saat layar dibuka — dan grid itu memang grid Admin.
  return (
    <QueueSelect
      tabs={tabs}
      active={active}
      onSelect={onSelect}
      id="antrean-treaty-non-prop"
      label="Antrean klaim treaty non-proporsional"
    />
  )
}
