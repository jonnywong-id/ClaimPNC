import { useCauseOfLossOptions, useClaimCoverages } from './api'
import type { CauseOfLossOption, CoverageRow } from './types'

/** Baris coverage yang sedang dipilih, sebagai satu keadaan. */
export type SelectedCoverage = {
  objek: string
  coverage: string
}

/** Segala yang dibutuhkan panel "Detail Perubahan Cause Of Loss" untuk menggambar dirinya. */
export type CauseOfLossPanel = {
  rows: CoverageRow[]

  /** Baris yang sedang dipilih, atau `null` bila belum memilih — atau pilihannya hilang. */
  selectedRow: CoverageRow | null

  options: CauseOfLossOption[]

  memuatCoverage: boolean
  coverageGagal: boolean
  pilihanGagal: boolean

  /**
   * Pilihan menunjuk baris yang SUDAH TIDAK ADA pada klaim.
   *
   * Terjadi saat permintaan lama dibuka kembali setelah coverage-nya dibuang dari klaim.
   * Dibiarkan diam, panel akan tampak "sudah dipilih" tanpa baris yang tersorot — lalu
   * server menolaknya saat disimpan, dengan sebab yang tidak terlihat di layar.
   */
  pilihanHilang: boolean
}

/**
 * Menyiapkan isi panel "Detail Perubahan Cause Of Loss".
 *
 * # Kenapa ini hook tersendiri, bukan bagian form
 *
 * Form sudah memikul pencarian klaim, master tipe, validasi, dan dua panel. Menambahkan
 * percabangan panel ini ke dalamnya membuat satu fungsi memutuskan terlalu banyak hal
 * sekaligus — terbaca sebagai kerumitan yang diperingatkan lint, dan lebih penting lagi
 * sebagai berkas yang harus dibaca utuh untuk mengubah satu panel.
 *
 * Di sini seluruh turunan panel hidup berdampingan dengan alasannya, dan form hanya
 * menggambar.
 *
 * # Keduanya hanya dimuat saat panelnya muncul
 *
 * `aktif` menahan kedua kueri selama tipe proteksinya bukan perubahan Cause of Loss.
 * Memuatnya pada setiap tipe berarti dua perjalanan jaringan pada form yang tidak akan
 * pernah menampilkan panelnya.
 */
export function useCauseOfLossPanel(
  claimNumber: string,
  aktif: boolean,
  selected: SelectedCoverage,
): CauseOfLossPanel {
  const coverages = useClaimCoverages(claimNumber, aktif)
  const causeOptions = useCauseOfLossOptions(claimNumber, aktif)

  const rows = coverages.data?.coverage ?? []

  // Dicari lewat PASANGAN kunci, bukan lewat nama.
  //
  // Nama berulang pada klaim nyata — `JackHugh / Resiko A` muncul tiga kali pada `PNC-1452` —
  // sehingga pencarian berbasis nama akan menyorot baris yang salah dua dari tiga kali.
  const selectedRow =
    rows.find(
      (row) => row.id_objek === selected.objek && row.id_coverage === selected.coverage,
    ) ?? null

  return {
    rows,
    selectedRow,
    options: causeOptions.data?.penyebab_kerugian ?? [],

    memuatCoverage: aktif && claimNumber.trim() !== '' && coverages.isPending,
    coverageGagal: coverages.isError,
    pilihanGagal: causeOptions.isError,

    pilihanHilang: aktif && selected.objek !== '' && selectedRow === null && coverages.isSuccess,
  }
}
