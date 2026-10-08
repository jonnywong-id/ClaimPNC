import { TabBar } from '@/components/TabBar'

import type { Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode tab yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
}

/**
 * Bilah tab Inbox Manager Admin.
 *
 * # Apa yang digantikan
 *
 * Tiga kontainer grid pada `Section/InboxManagerAdmin_Section-Section.xml`, masing-masing
 * berjudul `<pyTitle>` "Manajemen Admin - Non MBU", "- PA", dan "- Travel". Ketiganya
 * BUKAN tab di sana melainkan kontainer terpisah yang tampil menurut jabatan pengguna:
 *
 *	OperatorID.pyPosition = 'NONMBU' || OperatorID.pyOrgUnit = 'Development'
 *	OperatorID.pyPosition = 'PA'     || OperatorID.pyOrgUnit = 'Development'
 *	OperatorID.pyPosition = 'TRAVEL' || OperatorID.pyOrgUnit = 'Development'
 *
 * Akibatnya petugas biasa melihat tepat satu tabel, sedangkan petugas berjabatan Development
 * melihat ketiganya bertumpuk ke bawah — masing-masing dengan penomoran halamannya sendiri.
 * Di sini ketiganya menjadi tab yang dapat dipilih. Isi, kolom, dan urutan kolomnya sama
 * persis; yang berubah hanya cara ketiganya dipisahkan di layar.
 *
 * # Judulnya dipertahankan apa adanya
 *
 * Termasuk tanda hubung berspasi dan ejaan "Non MBU" tanpa tanda hubung — keduanya ditulis
 * persis seperti di section (`D-13`).
 *
 * # Yang tidak digambar di sini: tab yang bukan hak pengguna
 *
 * Daftar yang diterima komponen ini SUDAH disaring server. Menyembunyikan di layar bukan
 * kendali (`11-SECURITY.md` §3.1) — yang menjadi kendali adalah penolakan di server, dan
 * komponen ini hanya mengikuti hasilnya.
 */
export function ManagerAdminTabs({ tabs, active, onSelect }: Readonly<Props>) {
  // Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Melipatnya
  // menyembunyikan antrean mana saja yang tersedia — hal pertama yang ingin dilihat
  // penyelia saat membuka layar.
  return (
    <TabBar
      tabs={tabs}
      active={active}
      onSelect={onSelect}
      label="Antrean manajemen admin"
    />
  )
}
