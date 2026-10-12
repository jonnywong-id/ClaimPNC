/**
 * Sakelar fitur dokumen penunjang.
 *
 * # Riwayatnya
 *
 * Dimatikan atas instruksi Work Owner 2026-09-27: *"ternyata modul gcs belum disiapkan,
 * tolong munculkan saja dulu tombol unggah file penunjang tanpa fungsinya"*. Saat itu
 * layanan penyimpanannya belum ada, sehingga tidak ada yang dapat dituju — baik untuk
 * mengunggah maupun untuk membaca daftar.
 *
 * **Dinyalakan kembali 2026-10-03** atas instruksi Work Owner: *"jika saat ini sudah ada
 * tolong buat supaya tombol bisa upload dokumen ke gcs"*.
 *
 * # Apa yang perlu HIDUP di luar sakelar ini
 *
 * Menyalakannya tidak membuat unggahan berhasil dengan sendirinya. Yang harus ada:
 *
 *	layanan penyimpanan    alamat bawaannya dari `Connect REST/UploadDokumenPNC`
 *	layanan konversi AVIF  alamat bawaannya dari `Connect REST/KonversiAvif`
 *	`GENERAL.GET_TOKEN_STORAGE`  dipanggil saat `PENYIMPANAN_DOKUMEN_KODE_AKSES` kosong
 *
 * Ketiganya punya nilai bawaan atau jalur cadangan, sehingga tidak ada variabel lingkungan
 * yang WAJIB diisi. Yang tidak dapat dijamin dari sini adalah apakah ketiganya benar-benar
 * dapat dihubungi dari server tempat aplikasi berjalan — itu hanya terbukti saat berkas
 * pertama diunggah.
 *
 * Bila layanannya menolak, kegagalannya muncul sebagai `ErrUnggahGagal` dengan pesan yang
 * menyebut layanannya — bukan sebagai tombol yang diam.
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
 * Mematikannya kembali cukup mengubah baris ini menjadi `false`.
 */
export const FITUR_DOKUMEN_PENUNJANG_AKTIF = true
