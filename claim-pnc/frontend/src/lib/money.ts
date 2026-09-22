/**
 * Pemformatan dan pembacaan nilai uang.
 *
 * # Kenapa ia di satu tempat, bukan di modul yang memakainya
 *
 * `docs/Steering/08-TECHNICAL-STRATEGY.md` §3 aturan 4 menetapkan seluruh pemformatan
 * angka dan mata uang melewati satu berkas bersama. Alasannya konkret dan berasal dari
 * sistem lama: di sana pemformatan dikerjakan `TO_CHAR` yang tersebar di 411 tempat, dan
 * akibatnya tanggal serta angka tampil dengan bentuk yang berbeda-beda antarlayar tanpa
 * ada yang berwenang menyeragamkannya.
 *
 * Master Recovery adalah modul pertama yang menangani nilai uang, sehingga berkas ini
 * lahir bersamanya. Modul berikutnya memakainya kembali, bukan menulis versinya sendiri.
 *
 * # Satuannya RUPIAH UTUH, bukan sen
 *
 * Backend mengirim dan menerima bilangan bulat rupiah, dan MENOLAK pecahan. Alasannya
 * beserta buktinya ada di `masterrecovery.Amount` — ringkasnya: kolomnya NUMBER tanpa
 * skala sehingga tidak membatasi apa pun, seluruh baris yang ada bilangan bulat, dan
 * float dilarang untuk nilai uang tanpa perkecualian.
 */

/**
 * formatMoney menampilkan nilai uang dengan pemisah ribuan Indonesia.
 *
 * Lambang "Rp" TIDAK disertakan: label kolom sudah menyebutkannya, dan mengulanginya di
 * setiap sel membuat tabel bernilai banyak menjadi sulit dibaca. Layar yang membutuhkannya
 * menuliskannya sendiri di label.
 *
 * Nilai negatif ditampilkan apa adanya dengan tanda minus. Itu disengaja: sisa klaim
 * memang dapat negatif ketika pembayaran melampaui nilai klaim, dan menyembunyikannya
 * akan menyembunyikan kelebihan bayar dari orang yang membaca layar.
 */
export function formatMoney(value: number): string {
  if (!Number.isFinite(value)) return '0'
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 0 }).format(value)
}

/**
 * parseMoney membaca nilai uang dari isian yang diketik orang.
 *
 * Ia lapang terhadap BENTUK dan ketat terhadap ISI, sama seperti pembacanya di server:
 * pemisah ribuan berupa titik maupun koma dibuang — keduanya muncul di layar lama karena
 * formatnya tidak pernah diseragamkan — sedangkan yang bukan bilangan bulat ditolak.
 *
 * Mengembalikan `null` bila isiannya tidak dapat dibaca sebagai rupiah utuh. Pemanggil
 * yang memutuskan apa artinya: pada isian wajib ia pelanggaran, pada isian opsional ia
 * nol.
 */
export function parseMoney(text: string): number | null {
  const clean = text.trim().replaceAll('.', '').replaceAll(',', '')
  if (clean === '') return null
  // Regex, bukan Number(): Number("") bernilai 0, Number("1e3") bernilai 1000, dan
  // Number(" 12 ") bernilai 12 — ketiganya menerima bentuk yang tidak pernah diketik
  // orang sebagai nilai uang, dan server akan menolaknya.
  if (!/^-?\d+$/.test(clean)) return null

  const value = Number(clean)
  return Number.isSafeInteger(value) ? value : null
}

/**
 * remainder menghitung Sisa Klaim, meniru `masterrecovery.Remainder` di server PERSIS.
 *
 * # Kenapa aturannya ada di DUA tempat
 *
 * Duplikasi yang DISENGAJA, bukan kelalaian. Yang di sini memperlihatkan angkanya seketika
 * saat petugas mengetik — layar lama pun demikian, lewat aksi `HitungSisaKlaimRecovery`
 * pada setiap perubahan isian. Yang di server adalah yang MENGIKAT, dan nilainya tidak
 * pernah diterima dari badan permintaan.
 *
 * Menghapus yang di sini berarti setiap ketikan menunggu perjalanan jaringan; menghapus
 * yang di server berarti mempercayai angka yang dikirim klien untuk nilai uang.
 *
 * # Cabang kedua yang tampak keliru, dan kenapa ia dipertahankan
 *
 * Ketika ada pembayaran sebelumnya, pembayaran batch berjalan TIDAK mengurangi sisa.
 * Dibaca sekilas itu tampak cacat — dan ketiga baris produksi membuktikan itu memang
 * perilakunya. `P-5` menetapkan perilaku dipertahankan lebih dulu, dan aturan ini tidak
 * ada di daftar 13 perbaikan eksplisit `D-49`.
 *
 * Bila rumus ini berubah, `masterrecovery.Remainder` di backend WAJIB ikut berubah.
 */
export function remainder(claimAmount: number, previousPayment: number, payment: number): number {
  if (previousPayment === 0) return claimAmount - payment
  return claimAmount - previousPayment
}
