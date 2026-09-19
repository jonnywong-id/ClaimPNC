-- 0003 turun — membatalkan tabel Pelaporan Klaim (Oracle 19c)
--
-- ============================================================================
-- INI MENGHAPUS DATA. BACA SELURUHNYA SEBELUM MENJALANKAN.
-- ============================================================================
--
-- Berbeda dari migrasi turun 0002 yang hanya mengembalikan definisi view, berkas ini
-- MENGHAPUS SEBUAH TABEL BESERTA ISINYA. Setiap laporan kerugian yang sudah dicatat lewat
-- aplikasi baru akan hilang, dan tidak ada tempat lain yang menyimpannya — tabel ini
-- satu-satunya penyimpan laporan sejak modul Pelaporan Klaim menyala.
--
-- Laporan kerugian adalah pemberitahuan resmi dari tertanggung. Kehilangannya bukan
-- kehilangan data administratif: ia menghapus bukti bahwa kerugian pernah dilaporkan, dan
-- bersamanya tanggal lapor yang menentukan apakah klaim masih di dalam batas waktu polis.
--
--
-- # Yang WAJIB dilakukan sebelum menjalankan berkas ini
--
-- 1. HITUNG DULU berapa yang akan hilang. Bila jawabannya bukan nol, berhenti dan bawa
--    angkanya ke Work Owner:
--
--        SELECT COUNT(*) AS seluruhnya,
--               COUNT(NOMOR_KLAIM) AS sudah_jadi_klaim
--          FROM POOLDATA.CPNC_LAPORAN_KLAIM;
--
--    Baris yang NOMOR_KLAIM-nya terisi adalah yang paling berat: klaimnya sudah berjalan
--    dan merujuk laporan yang akan dihapus ini.
--
-- 2. SALIN ISINYA LEBIH DULU bila ada satu baris pun. Salinan itu satu-satunya jalan
--    kembali:
--
--        CREATE TABLE POOLDATA.CPNC_LAPORAN_KLAIM_ARSIP AS
--        SELECT * FROM POOLDATA.CPNC_LAPORAN_KLAIM;
--
--    (SELECT * dipakai di sini dengan sengaja, dan ini satu-satunya tempatnya: yang
--    diinginkan justru SELURUH kolom apa adanya, termasuk kolom yang ditambahkan setelah
--    berkas ini ditulis.)
--
-- 3. PASTIKAN APLIKASI SUDAH BERHENTI menulis ke tabel ini — modul Pelaporan Klaim
--    dimatikan dari menu dan rutenya, atau aplikasinya dihentikan. Menghapus tabel yang
--    sedang ditulis akan membuat permintaan pengguna gagal di tengah.
--
--
-- # Kenapa urutannya begini
--
-- Indeks lebih dulu, lalu tabel, lalu urutan. DROP TABLE sebenarnya ikut membuang indeks
-- miliknya, tetapi menyebutkannya satu per satu membuat berkas ini tetap benar bila kelak
-- dijalankan sebagian — dan membuat DBA melihat persis apa saja yang hilang.
--
-- Urutan dibuang TERAKHIR karena ia berdiri sendiri: ia tetap ada meski tabelnya sudah
-- hilang, dan meninggalkannya akan membuat migrasi naik berikutnya gagal dengan
-- ORA-00955 pada langkah 2.


DROP INDEX POOLDATA.IX_CPNC_LAPORAN_KLAIM_KLAIM;

DROP INDEX POOLDATA.IX_CPNC_LAPORAN_KLAIM_CABANG;

DROP INDEX POOLDATA.IX_CPNC_LAPORAN_KLAIM_URUT;

-- PURGE dipakai supaya tabelnya benar-benar hilang, tidak mendarat di recycle bin.
--
-- Tanpa PURGE, menjalankan migrasi naik lagi akan berhasil membuat tabel baru sementara
-- tabel lama menumpuk di recycle bin dengan nama BIN$... — dan isinya masih memuat data
-- nasabah yang dikira sudah terhapus. Bila salinan arsip langkah 2 sudah dibuat, tidak
-- ada gunanya menyimpan salinan kedua yang tak bernama.
DROP TABLE POOLDATA.CPNC_LAPORAN_KLAIM PURGE;

DROP SEQUENCE POOLDATA.CPNC_LAPORAN_KLAIM_SEQ;
