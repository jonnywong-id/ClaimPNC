-- 0003 turun — mencabut hak akses Master Dokumen Travel (Oracle 19c)
--
-- Migrasi naiknya tidak mengubah satu pun objek basis data, sehingga tidak ada struktur
-- yang perlu dikembalikan. Yang dapat dicabut hanyalah hak yang diberikannya.
--
--
-- ## Baca ini sebelum menjalankan
--
-- Mencabut hak TIDAK menghapus data yang telanjur ditulis modul ini. Baris baru di
-- POOLDATA.M_DOCTRAVEL tetap ada, dan memang HARUS tetap ada: DOCID-nya sudah dapat
-- dirujuk baris V_LST_DOC_TRAVEL dan dokumen klaim yang terunggah sesudahnya. Menghapus
-- baris master akan membuat rujukan itu kehilangan artinya (`ADR-0012`).
--
-- Setelah dicabut, layar Master Dokumen Travel akan menjawab galat hak akses. Sistem
-- lama TIDAK ikut terpengaruh: ia menulis lewat procedure DOCTRAVEL_CVG dengan akun
-- Pega, bukan dengan akun aplikasi ini.
--
-- Dijalankan di SETIAP basis data entitas tempat migrasi naiknya pernah dijalankan.
--
-- Ganti <AKUN_APLIKASI> dengan nama akun yang sebenarnya.


-- ---------------------------------------------------------------------------
-- Langkah 1 — cabut hak tulis, biarkan hak baca.
--
-- Dua langkah terpisah dengan sengaja. Mencabut hak TULIS saja sudah cukup untuk
-- menghentikan modul ini menyentuh data, dan itu yang biasanya dimaksud saat sebuah
-- modul ditarik. Hak BACA sering dibutuhkan modul lain yang menyusul — layar unggah
-- dokumen klaim Travel akan membaca tabel yang sama.
-- ---------------------------------------------------------------------------

-- REVOKE INSERT, UPDATE ON POOLDATA.M_DOCTRAVEL FROM <AKUN_APLIKASI>;


-- ---------------------------------------------------------------------------
-- Langkah 2 — cabut sisanya, HANYA bila akun aplikasi memang tidak lagi memakainya.
--
-- PERIKSA LEBIH DULU. Ketiga objek di bawah dipakai bersama modul lain:
--
--   * M_SITE_DATABASE  juga dibaca Master Status Klaim untuk membentuk LSC_ID
--                      (migrasi 0002). Mencabutnya akan merusak modul itu.
--   * DOCTRAVEL_SEQ    hanya dipakai modul ini.
--   * M_DOCTRAVEL      akan dibaca modul Daftar Detail Dokumen Travel bila kelak
--                      dibangun.
-- ---------------------------------------------------------------------------

-- REVOKE SELECT ON POOLDATA.DOCTRAVEL_SEQ FROM <AKUN_APLIKASI>;
-- REVOKE SELECT ON POOLDATA.M_DOCTRAVEL FROM <AKUN_APLIKASI>;
--
-- JANGAN dijalankan tanpa memeriksa modul lain lebih dulu:
-- REVOKE SELECT ON POOLDATA.M_SITE_DATABASE FROM <AKUN_APLIKASI>;
