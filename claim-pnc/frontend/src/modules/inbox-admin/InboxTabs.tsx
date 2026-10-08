import { TabBar } from '@/components/TabBar'

import type { Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode tab yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
}

/**
 * Bilah tab Inbox Admin.
 *
 * # Apa yang digantikan
 *
 * Bilah tab pada `Section/PNCInboxAdmin-Section.xml`, yang di sana berupa delapan tombol
 * yang menyetel properti `TempView.CityID` lalu menjalankan ulang seluruh activity. Judul
 * tabnya dipertahankan apa adanya — termasuk yang berbahasa Inggris seperti "Unregistered
 * RCV Online" — karena `D-13` menetapkan tampilan meniru Pega supaya pengguna tidak perlu
 * belajar ulang.
 *
 * # Kenapa tanpa lencana jumlah
 *
 * Berbeda dengan layar Pelaporan Klaim, di sini jumlah per tab TIDAK ditampilkan. Sistem
 * lama pun tidak menampilkannya, dan menghadirkannya berarti menjalankan kedelapan kueri
 * sekaligus setiap kali layar dibuka — sementara paginasi layar ini menarik seluruh baris
 * tiap kueri (keputusan Work Owner 2026-09-20). Lencana yang harganya delapan kali beban
 * layar bukan lencana yang sepadan.
 *
 * # Tab yang tidak dibangun
 *
 * Ketiga tab Komunikasi tidak digambar sama sekali, dan ketiadaannya dijelaskan di bawah
 * tabel — bukan digambar sebagai tab mati yang mengundang klik.
 */
export function InboxTabs({ tabs, active, onSelect }: Readonly<Props>) {
  // Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Delapan tab
  // tidak muat berjajar di ponsel, tetapi melipatnya menyembunyikan antrean mana saja
  // yang tersedia — dan itulah hal pertama yang ingin dilihat petugas saat membuka layar.
  return (
    <TabBar
      tabs={tabs}
      active={active}
      onSelect={onSelect}
      label="Antrean Inbox Admin"
      flexItems={false}
    />
  )
}
