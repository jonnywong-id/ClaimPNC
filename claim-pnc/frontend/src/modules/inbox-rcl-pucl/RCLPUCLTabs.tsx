import { TabBar } from '@/components/TabBar'

import type { Tab } from './types'

type Props = {
  tabs: Tab[]
  /** Kode tab yang sedang terbuka. */
  active: string
  onSelect: (code: string) => void
}

/**
 * Bilah tab Inbox RCL/PUCL.
 *
 * # Apa yang digantikan
 *
 * `Section/InputPUCL-RCL_Section-Section.xml`, yang menyertakan tiga SUB_SECTION berurutan:
 * `InboxCetakSuratPUCLRCL_Section`, `InboxKelengkapanDocPUCLRCL_Section`, dan
 * `InboxAJSMSIG_Section`. Ketiga judulnya ada di sana apa adanya sebagai
 * `pyCaption Cetak Surat`, `pyCaption Kelengkapan Dokumen`, dan `pyCaption Klaim MSIG` —
 * tidak satu pun dikarang (`D-13`).
 *
 * # Kenapa keterangan tab di sini lebih penting daripada di layar lain
 *
 * Karena ketiga tab punya KOLOM YANG IDENTIK. Di layar lain, pengguna yang tersesat ke tab
 * yang salah segera menyadarinya dari kolom yang berbeda; di sini tidak ada petunjuk
 * semacam itu sama sekali. Yang membedakan ketiganya hanyalah perjalanan surat PUCL — belum
 * dicetak, sudah dicetak, dan jalur MSIG — dan itu hanya terbaca dari keterangannya.
 *
 * Karena itu keterangan tab ikut digambar sebagai `title`, bukan hanya di bawah bilah.
 */
export function RCLPUCLTabs({ tabs, active, onSelect }: Readonly<Props>) {
  // Digulir menyamping pada layar sempit, bukan dilipat menjadi dropdown. Melipatnya
  // menyembunyikan antrean mana saja yang tersedia — dan pada layar yang ketiga tabnya
  // tampak sama, daftar tab adalah satu-satunya hal yang menjelaskan strukturnya.
  //
  // Tidak ada tab terhalang di layar ini hari ini — tab "Klaim MSIG" yang kemungkinan
  // kosong TIDAK ditandai begitu, karena kosongnya adalah jawaban, bukan ketidakmampuan
  // menjawab. Penandanya tetap digambar supaya penambahan tab terhalang kelak cukup
  // dilakukan di backend.
  return (
    <TabBar
      tabs={tabs.map((tab) => ({
        kode: tab.kode,
        nama: tab.nama,
        keterangan: tab.terhalang ? tab.alasan_terhalang : tab.keterangan,
        badge: tab.terhalang ? 'belum tersedia' : undefined,
      }))}
      active={active}
      onSelect={onSelect}
      label="Antrean klaim RCL/PUCL"
    />
  )
}
