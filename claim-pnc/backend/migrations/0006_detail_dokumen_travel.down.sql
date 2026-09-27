-- 0006 turun — mencabut hak akses Daftar Detail Dokumen Travel.
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- BERKAS INI BELUM PERNAH DIJALANKAN DI LINGKUNGAN MANA PUN.
--
-- Sama seperti migrasi naiknya, ia dijalankan di BASIS DATA SETIAP ENTITAS yang pernah
-- dinaikkan — entitas yang terlewat akan tertinggal pada hak akses yang berbeda dari
-- yang lain.
--
--
-- ## Yang dikembalikan, dan yang TIDAK
--
--   DIKEMBALIKAN  hak akses akun aplikasi atas objek modul ini.
--   TIDAK         data. Setiap aturan dokumen dan setiap pembatasan plan yang sudah
--                 terlanjur disimpan lewat layar Go TETAP ADA, dan memang harus tetap
--                 ada.
--
-- Sebabnya bukan kelalaian: baris itu dibaca `Activity/TravelDocument_act-Act.xml` saat
-- klaim Travel diregistrasi. Menghapusnya berarti mengubah kelengkapan dokumen yang
-- dituntut pada klaim yang sedang berjalan — akibat yang jauh lebih besar daripada
-- sekadar membatalkan sebuah migrasi, dan tidak terlihat sebagai galat di layar mana pun.
--
-- Bila baris tertentu memang harus dibuang, itu permintaan perubahan data tersendiri
-- yang menempuh persetujuan Work Owner (`D-63`) dan menyebutkan barisnya satu per satu —
-- bukan akibat sampingan dari menurunkan migrasi.
--
--
-- ## Yang HARUS diperiksa sebelum menjalankan
--
-- 1. PASTIKAN APLIKASI SUDAH TIDAK MEMAKAI LAYARNYA. Mencabut hak selagi aplikasi hidup
--    akan membuat layar Daftar Detail Dokumen Travel menjawab galat hak akses kepada
--    petugas yang sedang menyuntingnya, dan isian yang belum tersimpan hilang.
--
-- 2. CATAT JUMLAH BARIS SEKARANG, supaya dapat dipastikan tidak ada yang ikut hilang:
--
--        SELECT COUNT(*) FROM POOLDATA.V_LST_DOC_TRAVEL;
--        SELECT COUNT(*) FROM POOLDATA.V_LST_DOC_TRAVEL_COVERAGE;


-- ---------------------------------------------------------------------------
-- Langkah 1 — cabut hak tulis lebih dulu, baru hak baca.
--
-- Urutannya disengaja: mencabut hak baca lebih dulu membuat aplikasi gagal saat MEMBACA
-- padahal ia masih dapat MENULIS, dan pada jeda itu sebuah penyimpanan masih mungkin
-- berjalan atas data yang tidak dapat ditampilkan kembali.
--
-- Ganti <AKUN_APLIKASI> dengan nama akun yang sebenarnya, dan jalankan di SETIAP basis
-- data entitas yang pernah dinaikkan.
--
-- Nama tabel dasar di bawah mengikuti jawaban DBA atas Bagian 1 butir 1b pada berkas
-- .up.sql. Bila namanya ternyata berbeda, sesuaikan di sini juga.
-- ---------------------------------------------------------------------------

-- REVOKE INSERT, UPDATE ON POOLDATA.LST_DOC_TRAVEL FROM <AKUN_APLIKASI>;
-- REVOKE INSERT, UPDATE, DELETE ON POOLDATA.LST_DOC_TRAVEL_COVERAGE FROM <AKUN_APLIKASI>;
-- REVOKE SELECT ON POOLDATA.LST_DOC_TRAVEL_SEQ FROM <AKUN_APLIKASI>;

-- REVOKE SELECT ON POOLDATA.LST_DOC_TRAVEL FROM <AKUN_APLIKASI>;
-- REVOKE SELECT ON POOLDATA.LST_DOC_TRAVEL_COVERAGE FROM <AKUN_APLIKASI>;
-- REVOKE SELECT ON POOLDATA.V_LST_DOC_TRAVEL FROM <AKUN_APLIKASI>;
-- REVOKE SELECT ON POOLDATA.V_LST_DOC_TRAVEL_COVERAGE FROM <AKUN_APLIKASI>;
-- REVOKE SELECT ON POOLDATA.M_PLANTRAVEL FROM <AKUN_APLIKASI>;


-- ---------------------------------------------------------------------------
-- Langkah 2 — yang SENGAJA TIDAK dicabut
-- ---------------------------------------------------------------------------
--
-- POOLDATA.M_DOCTRAVEL. Haknya diberikan migrasi 0003 untuk modul Master Dokumen Travel,
-- dan modul itu MASIH MEMAKAINYA. Mencabutnya di sini akan mematikan layar yang tidak
-- ada hubungannya dengan migrasi ini — kelas kerusakan yang paling mudah terjadi ketika
-- dua modul berbagi satu tabel.
--
-- Pencabutannya, bila kelak diperlukan, ada di berkas turun migrasi 0003.


-- ---------------------------------------------------------------------------
-- Langkah 3 — verifikasi, dijalankan dengan AKUN APLIKASI
--
-- DIHARAPKAN GAGAL dengan ORA-00942. Berhasil berarti masih ada hak yang tertinggal —
-- kemungkinan besar diberikan lewat peran, bukan langsung ke akunnya.
-- ---------------------------------------------------------------------------

--     SELECT ID FROM POOLDATA.V_LST_DOC_TRAVEL WHERE 1 = 0;
