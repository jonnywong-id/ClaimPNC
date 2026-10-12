/**
 * Pemformatan dan pelabelan khas layar Master XOL.
 *
 * Dikumpulkan di satu berkas supaya angka dan label yang sama tidak diformat dengan dua
 * cara berbeda di grid dan di form — persis kekacauan yang `docs/Steering/07-TECHNICAL-STRATEGY.md`
 * §3 cegah dengan mewajibkan seluruh pemformatan lewat satu tempat.
 */

/**
 * Memformat bilangan bulat dengan pemisah ribuan gaya Indonesia.
 *
 * Seluruh angka di modul ini bilangan BULAT — diperiksa ke produksi pada 2026-09-20,
 * keempat kolom angkanya tidak punya satu pun nilai pecahan. Karena itu tidak ada digit
 * desimal yang perlu ditampilkan, dan menambahkannya justru akan menyiratkan ketelitian
 * yang tidak ada.
 */
export function formatMoney(value: number): string {
  if (!Number.isFinite(value)) return '0'
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 0 }).format(value)
}

/**
 * Label status komite.
 *
 * Ketiga nilainya dibaca dari produksi: `'0'` pada lima induk, `'1'` pada dua, dan kosong
 * pada satu. Artinya diturunkan dari kedua pemakainya —
 * `GetDataMasterXOLForKomiteApprove` menyaring `STSKOMITE='0'` untuk menyusun antrean
 * persetujuan, dan `UpdateStatusMasterKomitexol` menyetelnya ke `'0'` saat PIC mengajukan.
 *
 * Nilai di luar ketiganya ditampilkan APA ADANYA, bukan dipaksa menjadi "tidak dikenal":
 * kolomnya VARCHAR2(100) tanpa constraint, dan menyembunyikan nilai yang tidak terduga
 * membuat data yang aneh tampak normal.
 */
export function committeeLabel(status: string): string {
  switch (status) {
    case '':
      return 'Belum diajukan'
    case '0':
      return 'Menunggu komite'
    case '1':
      return 'Disetujui'
    default:
      return status
  }
}

/** Kelas warna lencana status komite. */
export function committeeTone(status: string): string {
  switch (status) {
    case '0':
      return 'bg-amber-50 text-amber-800 ring-amber-200'
    case '1':
      return 'bg-emerald-50 text-emerald-800 ring-emerald-200'
    case '':
      return 'bg-slate-100 text-slate-600 ring-slate-200'
    default:
      return 'bg-slate-100 text-slate-600 ring-slate-200'
  }
}
