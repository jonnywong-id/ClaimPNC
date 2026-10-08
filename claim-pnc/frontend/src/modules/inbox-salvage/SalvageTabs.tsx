import { TabBar } from '@/components/TabBar'

import type { Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode daftar yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
}

/**
 * Bilah daftar Inbox Salvage.
 *
 * # Apa yang digantikan
 *
 * Ketiga belas grid pada `Section/InboxSalvageASM-Section.xml`. Di Pega ketiganya TIDAK
 * berupa bilah tab melainkan kumpulan grid yang digambar berurutan ke bawah dalam satu
 * halaman yang sangat panjang, dan yang memilih grid mana yang terisi adalah pasangan
 * `Param.tipe`/`Param.tipe2` yang dikirim tabel ringkas di atasnya.
 *
 * Bilah tab dipilih di sini karena tiga belas grid bertumpuk pada satu halaman menuntut
 * pengguna menggulir untuk mengetahui daftar apa saja yang ada — dan judulnya mirip-mirip
 * ("Checker", "Salvage Diterima", "Balai Lelang", "Request Balai Lelang"), sehingga
 * menggulir bukan cara yang baik untuk menemukannya. Alur kerjanya tidak berubah: yang
 * dilihat pengguna tetap satu daftar pada satu waktu, dipilih dari tabel ringkas yang sama.
 *
 * # Kenapa keterangan daftar di sini lebih penting daripada di layar lain
 *
 * Karena EMPAT pasang daftar punya kolom yang identik, dan satu pasang bahkan punya ISI
 * yang identik — "Checker" dan "Salvage Diterima" menyaring status transfer yang sama.
 * Pengguna yang tersesat ke daftar yang salah tidak punya satu pun petunjuk visual.
 *
 * Karena itu keterangan ikut digambar sebagai `title`, bukan hanya di bawah bilah.
 */
export function SalvageTabs({ tabs, active, onSelect }: Readonly<Props>) {
  // Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Melipatnya
  // menyembunyikan daftar mana saja yang tersedia — dan pada layar yang punya tiga belas
  // daftar berjudul mirip, bilah ini adalah satu-satunya hal yang menjelaskan strukturnya.
  return (
    <TabBar
      tabs={tabs}
      active={active}
      onSelect={onSelect}
      label="Daftar pengajuan salvage"
    />
  )
}
