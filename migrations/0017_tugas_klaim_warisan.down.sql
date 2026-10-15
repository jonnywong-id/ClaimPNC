-- 0017 (down) — Mencabut tugas klaim warisan yang dibuat migrasi naiknya
--
-- ============================================================================
-- APA YANG DIHAPUS, DAN APA YANG TIDAK
-- ============================================================================
--
-- Hanya tugas yang MASIH TERBUKA milik klaim bernomor `PNC-xxxx`. Tiga pembatasan menjaganya:
--
--	NOMOR_KLAIM LIKE 'PNC-%'   klaim terbitan aplikasi ini (`PNCN.`) tidak tersentuh
--	SELESAI_PADA IS NULL       tugas yang sudah dikerjakan TIDAK dihapus — menghapusnya
--	                           membuang jejak pekerjaan yang benar-benar terjadi
--	DIAMBIL_PADA IS NULL       tugas yang sudah DIAMBIL seseorang juga tidak dihapus
--
-- Dua pembatasan terakhir penting: begitu migrasi naik dijalankan, petugas dapat langsung
-- mengerjakan klaim-klaim itu. Turun yang menghapus membabi buta akan membuang pekerjaan
-- mereka tanpa jejak, dan tidak ada yang dapat memulihkannya.
--
-- Akibatnya turun ini TIDAK memulihkan keadaan persis seperti sebelum naik, bila sudah ada
-- yang dikerjakan. Itu disengaja: kehilangan pekerjaan lebih buruk daripada sisa baris.

DELETE FROM CPNC_TUGAS
 WHERE NOMOR_KLAIM LIKE 'PNC-%'
   AND SELESAI_PADA IS NULL
   AND DIAMBIL_PADA IS NULL;

COMMIT;
