/**
 * Tipe kontrak API modul Master Pasal AI.
 *
 * # Kenapa di sini, bukan di `src/api/types.ts`
 *
 * `api/types.ts` memuat kontrak SELURUH aplikasi, dan ia berkas bersama yang sedang disunting
 * beberapa modul sekaligus. Modul ini belum terpasang di menu — sumber datanya masih ditunggu
 * dari Tim Pega — sehingga menaruh tipenya di sana berarti menambah muatan ke berkas bersama
 * demi layar yang belum dapat dibuka siapa pun.
 *
 * Ia dipindahkan ke `api/types.ts` bersama pemasangan rutenya, dalam satu langkah. Modul
 * `riwayat-klaim` memakai pola yang sama (`modules/riwayat-klaim/types.ts`).
 */

/**
 * Satu baris wording polis.
 *
 * Nama field-nya mengikuti **judul kolom di layar**, bukan nama kolom basis data maupun nama
 * properti klipboard Pega. Ketiganya berbeda di modul ini:
 *
 *	layar       kolom basis data   properti Pega (warisan, menyesatkan)
 *	No Pasal    WP_PASAL           .City
 *	Ayat        WP_AYAT            .CityID
 *	Kejadian    WP_KEJADIAN        .District
 */
export type ClauseAI = {
  /**
   * Kolom `WP_ID` — kunci baris.
   *
   * TIDAK digambar sebagai kolom: layar lamanya hanya punya tiga. Ia dipakai sebagai kunci
   * baris tabel, dan kueri lamanya pun memilihnya serta mengurutkan dengannya.
   */
  id: string
  /** Kolom `WP_PASAL` — di grid berlabel "No Pasal". */
  no_pasal: string
  /** Kolom `WP_AYAT` — di grid berlabel "Ayat". */
  ayat: string
  /**
   * Kolom `WP_KEJADIAN` — di grid berlabel "Kejadian".
   *
   * Teks panjang: di layar Pega ia satu-satunya yang digambar `pxTextArea`, dan kolomnya
   * paling lebar (283 px berbanding 48 dan 47).
   */
  kejadian: string
}

/**
 * Keterangan paginasi yang menyertai setiap daftar.
 *
 * Ia ada karena paginasi modul ini dikerjakan di **server** — berbeda dari seluruh layar
 * master lain di aplikasi ini, yang memaginasi di peramban.
 *
 * Itu bukan pilihan kami: layar lamanya pun sudah begitu. Grid Pega-nya ber-`pyPageMode =
 * None`, dan jendelanya dihitung activity lewat `FirstRow`/`LastRow`.
 */
export type ClauseAIPagination = {
  /** Nomor halaman yang dikembalikan, mulai dari 1. */
  halaman: number
  /** Banyaknya baris per halaman — 25, ditetapkan server. */
  ukuran_halaman: number
  /** Banyaknya halaman yang tersedia, minimal 1. */
  jumlah_halaman: number
  /** Cacah **seluruh** baris yang cocok, bukan cacah baris pada halaman ini. */
  jumlah_baris: number
}

/** Jawaban `GET /api/master/pasal-ai`. */
export type ClauseAIListResponse = {
  pasal_ai: ClauseAI[]
  paginasi: ClauseAIPagination
  portal: string
}

/**
 * Modul ini TIDAK punya kode galat sendiri.
 *
 * Ia baca-saja: tidak ada isian yang dapat cacat, tidak ada baris yang dapat bentrok, dan
 * tidak ada penyimpanan yang dapat gagal separuh. Yang mungkin terjadi hanyalah galat portal
 * dan kegagalan teknis — keduanya dipetakan penulis galat bersama.
 */
