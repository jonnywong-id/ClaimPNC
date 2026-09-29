/**
 * Sakelar fitur dokumen penunjang.
 *
 * # Kenapa dimatikan
 *
 * Work Owner 2026-09-27: *"ternyata modul gcs belum disiapkan, tolong munculkan saja dulu
 * tombol unggah file penunjang tanpa fungsinya"*. Layanan penyimpanannya belum ada, jadi
 * tidak ada yang dapat dituju — baik untuk mengunggah maupun untuk membaca daftar.
 *
 * # Kenapa satu sakelar, bukan menghapus kodenya
 *
 * Seluruh jalurnya sudah dibangun dan diuji: domain, kedua adapter, transport, dan
 * perakitan. Menghapusnya berarti membangunnya lagi dari nol saat layanannya siap, dan yang
 * hilang bukan hanya waktu melainkan seluruh alasan di dalam komentarnya — kenapa PDF tidak
 * berganti tipe, kenapa konversi gagal membatalkan unggahan, kenapa `"-"` dipakai saat
 * nomor klaim kosong.
 *
 * # Kenapa ia berkas TERSENDIRI, bukan konstanta di dalam komponennya
 *
 * Supaya kedua keadaan tetap dapat diuji. Konstanta di dalam komponen tidak dapat diganti
 * dari uji, sehingga mematikan fiturnya akan ikut mematikan dua belas uji yang menjaga
 * perilakunya — dan menghidupkannya kembali kelak menjadi langkah yang tidak teruji sama
 * sekali.
 *
 * Menghidupkannya cukup mengubah baris ini menjadi `true`.
 */
export const FITUR_DOKUMEN_PENUNJANG_AKTIF = false
