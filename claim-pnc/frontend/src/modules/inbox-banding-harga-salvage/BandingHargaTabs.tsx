import { TabBar } from '@/components/TabBar'

import type { Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode tab yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
}

/**
 * Bilah tab Inbox Banding Harga Salvage.
 *
 * # Apa yang digantikan
 *
 * DUA kontainer bersyarat pada `Section/InboxReqSalvageASM-Section.xml`, yang di sana
 * dinyalakan bergantian oleh sepasang properti:
 *
 *   tempQuery.FlagReject==1 && tempQuery.FlagASO==1   grid Request Banding Harga
 *   tempQuery.FlagReject=1  && tempQuery.FlagASO=2    grid History Cheker
 *
 * Keduanya memanggil ulang activity `SetReqSalvage_Act` dengan parameter `tipe` yang berbeda.
 * Judul tabnya diambil dari kolom "Status Salvage" tabel ringkas, dan salah ketik "Cheker"
 * dipertahankan karena itulah yang dibaca pengguna hari ini (`D-13`).
 *
 * # Kenapa tanpa lencana jumlah
 *
 * Karena jumlahnya sudah digambar tabel ringkas tepat di atas bilah ini, dan tabel itu pula
 * yang menjadi navigasi di layar lama. Mengulangnya di dua tempat berarti dua angka yang dapat
 * berselisih saat salah satu belum dimuat ulang.
 */
export function BandingHargaTabs({ tabs, active, onSelect }: Readonly<Props>) {
  // Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Dua tab memang
  // muat di hampir setiap layar, tetapi melipatnya menyembunyikan antrean mana saja yang
  // tersedia — hal pertama yang ingin dilihat komite saat membuka layar.
  return (
    <TabBar
      tabs={tabs}
      active={active}
      onSelect={onSelect}
      label="Antrean banding harga salvage"
      flexItems={false}
    />
  )
}
