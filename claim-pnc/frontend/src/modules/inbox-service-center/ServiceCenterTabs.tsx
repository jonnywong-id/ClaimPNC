import { TabBar } from '@/components/TabBar'

import type { Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode tab yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
}

/**
 * Bilah tab Inbox Service Center.
 *
 * # Apa yang digantikan
 *
 * Wadah `TABBED` pada `Section/BrowseServiceCenter-Section.xml`, yang di sana berupa empat
 * kontainer grid — masing-masing memanggil ulang activity `DataServiceCenter` dengan nilai
 * `stsapprove` yang berbeda. Judul tabnya dipertahankan apa adanya, termasuk yang berbahasa
 * Inggris seperti "Waiting Approval", karena `D-13` menetapkan tampilan meniru Pega supaya
 * pengguna tidak perlu belajar ulang.
 *
 * # Kenapa tanpa lencana jumlah
 *
 * Sistem lama pun tidak menampilkannya, dan menghadirkannya berarti menjalankan keempat
 * kueri sekaligus setiap kali layar dibuka — terhadap tabel yang tumbuh tanpa batas.
 */
export function ServiceCenterTabs({ tabs, active, onSelect }: Readonly<Props>) {
  // Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Empat tab
  // memang muat di kebanyakan layar, tetapi judul sepanjang "Waiting Approval" membuat
  // bilahnya melampaui lebar ponsel — dan melipatnya menyembunyikan antrean mana saja
  // yang tersedia, yaitu hal pertama yang ingin dilihat petugas saat membuka layar.
  return (
    <TabBar
      tabs={tabs}
      active={active}
      onSelect={onSelect}
      label="Antrean Inbox Service Center"
      flexItems={false}
    />
  )
}
