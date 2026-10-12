/**
 * Pembaca dokumen klaim — isi `POOLDATA.JSON_KLAIM.DATA_JSON`.
 *
 * Bentuk dokumen ini BELUM PERNAH DIPERIKSA (`R-08`), sehingga pembacaannya lewat jalur, bukan
 * lewat tipe yang dikarang. Yang membedakan pembaca ini dari `obj?.a?.b` biasa adalah ia
 * membedakan **jalur yang tidak ada** dari **jalur yang ada tetapi kosong** — dua keadaan yang
 * tampak sama di layar tetapi berarti sangat berbeda saat menelusuri kenapa sel kosong.
 */

/** Hasil pembacaan satu jalur. */
export type Terbaca =
  | { ada: false }
  | { ada: true; nilai: string }

/**
 * Membaca satu jalur bertitik, mis. `ClaimData.DateOfLoss`.
 *
 * Nilai non-teks diubah menjadi teks apa adanya — angka dan boolean memang tersimpan begitu di
 * dokumennya, dan mengubahnya menjadi kosong akan menyembunyikan isian yang sebenarnya terisi.
 *
 * `null` diperlakukan sebagai ADA tetapi kosong, bukan tidak ada: jalurnya memang ada di
 * dokumen, hanya nilainya yang belum diisi.
 */
export function ambil(dokumen: Record<string, unknown>, jalur: string): Terbaca {
  let simpul: unknown = dokumen

  for (const bagian of jalur.split('.')) {
    if (simpul === null || typeof simpul !== 'object' || Array.isArray(simpul)) {
      return { ada: false }
    }
    if (!(bagian in (simpul as Record<string, unknown>))) {
      return { ada: false }
    }
    simpul = (simpul as Record<string, unknown>)[bagian]
  }

  if (simpul === null || simpul === undefined) return { ada: true, nilai: '' }
  if (typeof simpul === 'object') return { ada: true, nilai: '' }

  return { ada: true, nilai: String(simpul) }
}

/**
 * Membaca daftar baris pada satu jalur, mis. `ClaimData.ObjectList`.
 *
 * Dokumen yang menyimpan satu baris sebagai objek — bukan larik berisi satu — ikut ditangani:
 * Pega menulis keduanya, dan layar yang hanya menerima larik akan menampilkan grid kosong
 * untuk klaim berobjek tunggal.
 */
export function ambilDaftar(
  dokumen: Record<string, unknown>,
  jalur: string,
): Record<string, unknown>[] {
  let simpul: unknown = dokumen

  for (const bagian of jalur.split('.')) {
    if (simpul === null || typeof simpul !== 'object') return []
    simpul = (simpul as Record<string, unknown>)[bagian]
  }

  if (Array.isArray(simpul)) {
    return simpul.filter(
      (baris): baris is Record<string, unknown> =>
        baris !== null && typeof baris === 'object' && !Array.isArray(baris),
    )
  }
  if (simpul !== null && typeof simpul === 'object') {
    return [simpul as Record<string, unknown>]
  }
  return []
}
