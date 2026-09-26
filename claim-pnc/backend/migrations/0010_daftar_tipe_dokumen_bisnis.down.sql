-- 0010 turun — Daftar Tipe Dokumen Bisnis
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- ## Yang DAPAT dibatalkan, dan yang TIDAK
--
-- Migrasi naiknya tidak mengubah bentuk apa pun — ia hanya memberi hak akses, dan
-- menanyakan hal-hal yang jawabannya tidak mengubah basis data. Karena itu satu-satunya
-- yang dapat dibatalkan adalah hak aksesnya.
--
-- Yang TIDAK dapat dibatalkan, dan tidak boleh dicoba:
--
--   * BARIS YANG SUDAH DITULIS APLIKASI GO. Ia data bisnis yang sah — aturan kelengkapan
--     dokumen yang mungkin sudah dipakai klaim yang sedang berjalan. Mencabut hak akses
--     TIDAK menghapusnya, dan memang tidak boleh menghapusnya (`D-66`).
--
--   * NOMOR URUT YANG SUDAH TERPAKAI. LST_TYPE_DOC_BUSINESS_SEQ sudah bergerak maju.
--     Mengembalikannya akan menerbitkan ID yang bertabrakan dengan baris yang sudah ada.
--
-- Setelah berkas ini dijalankan, Pega kembali menjadi satu-satunya penulis kedua tabel —
-- dan itu memang keadaan yang benar bila modulnya ditarik.
--
--
-- ## Kapan berkas ini dijalankan
--
-- Hanya bila modul Daftar Tipe Dokumen Bisnis ditarik dari produksi. Ia BUKAN bagian dari
-- pemutakhiran biasa.
--
-- Sebelum menjalankannya, PASTIKAN aplikasi Go sudah berhenti melayani layar ini di
-- entitas yang bersangkutan. Mencabut hak akses selagi layarnya masih hidup akan membuat
-- setiap penyimpanan gagal dengan ORA-00942 — galat yang di layar terbaca sebagai
-- kerusakan aplikasi, bukan sebagai pencabutan yang disengaja.


-- ---------------------------------------------------------------------------
-- Langkah 1 — cabut hak tulis, biarkan hak baca.
--
-- Hak baca dibiarkan dengan sengaja: selama masa paralel, layar Go mungkin masih dipakai
-- untuk MELIHAT sementara penulisannya dikembalikan ke Pega. Mencabut keduanya sekaligus
-- menutup kemungkinan itu tanpa alasan.
--
-- Bila hak baca pun hendak dicabut, jalankan juga bagian yang dikomentari di bawahnya —
-- dan sadari bahwa layarnya akan gagal memuat, bukan tampil kosong.
--
-- Ganti <AKUN_APLIKASI> dengan nama akun yang sebenarnya, dan jalankan di SETIAP basis
-- data entitas tempat migrasi naiknya pernah dijalankan.
-- ---------------------------------------------------------------------------

-- REVOKE INSERT, UPDATE ON POOLDATA.LST_TYPE_DOC_BUSINESS TO <AKUN_APLIKASI>;
-- REVOKE INSERT         ON POOLDATA.COVERAGE_DOC_BUSINESS TO <AKUN_APLIKASI>;


-- ---------------------------------------------------------------------------
-- Langkah 2 — hak baca, HANYA bila modulnya benar-benar dibuang.
--
-- Keempat master rujukan sengaja TIDAK disebut di sini. Ketiganya dibaca modul LAIN yang
-- masih berjalan — BUSINESS oleh Master COL Simas Online, V_LST_DOC_TYPE oleh Daftar Tipe
-- Dokumen, V_LST_DOC_OBJ oleh Daftar Objek Dokumen — dan mencabut haknya akan mematikan
-- layar yang tidak ada hubungannya dengan modul ini.
-- ---------------------------------------------------------------------------

-- REVOKE SELECT ON POOLDATA.LST_TYPE_DOC_BUSINESS     TO <AKUN_APLIKASI>;
-- REVOKE SELECT ON POOLDATA.COVERAGE_DOC_BUSINESS     TO <AKUN_APLIKASI>;
-- REVOKE SELECT ON POOLDATA.LST_TYPE_DOC_BUSINESS_SEQ TO <AKUN_APLIKASI>;


-- ---------------------------------------------------------------------------
-- Langkah 3 — index, HANYA bila Bagian 3 migrasi naiknya benar-benar dijalankan.
--
-- Jangan menjalankannya bila index-nya sudah ada SEBELUM migrasi ini — membuangnya akan
-- memperlambat kueri Pega yang memakainya, dan itu kerusakan yang tidak disebabkan modul
-- ini.
-- ---------------------------------------------------------------------------

-- DROP INDEX POOLDATA.IX_LTDB_BUSINESSID;
-- DROP INDEX POOLDATA.IX_CDB_ID;
