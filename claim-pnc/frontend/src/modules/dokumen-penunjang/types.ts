/**
 * Bentuk dokumen penunjang sebagaimana dikirim server.
 *
 * Nama field mengikuti kontrak API — berbahasa Indonesia, salah satu dari lima pengecualian
 * `D-80`.
 */
export interface DokumenPenunjang {
  id: string

  /**
   * Nama berkas TERSIMPAN, bukan nama yang diketik pengguna.
   *
   * Pega membersihkan nama dengan `[^a-zA-Z0-9]` — termasuk titik ekstensinya — sehingga
   * `Foto Kerugian.pdf` tersimpan sebagai `FotoKerugianpdf`. Perilaku itu ditiru (`P-5`),
   * dan layar menampilkan apa yang benar-benar tersimpan alih-alih menyamarkannya.
   */
  nama_berkas: string

  jenis_dokumen: string

  /** Kosong bila metadata belum memuat alamatnya; bukan berarti unggahnya gagal. */
  url: string

  tanggal_unggah: string
  berlaku_sampai: string

  /** Dihitung DI SERVER, bukan dari jam peramban yang dapat salah berjam-jam. */
  kedaluwarsa: boolean
}

export interface DokumenPenunjangListResponse {
  data: DokumenPenunjang[]
}

export interface DokumenPenunjangItemResponse {
  data: DokumenPenunjang
}
