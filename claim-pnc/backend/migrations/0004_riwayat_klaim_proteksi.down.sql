-- 0004 turun — membatalkan jejak pemakaian proteksi data (Oracle 19c)
--
-- ============================================================================
-- INI MENGHAPUS JEJAK AUDIT. BACA SELURUHNYA SEBELUM MENJALANKAN.
-- ============================================================================
--
-- Berkas ini menghapus sebuah tabel beserta isinya. Yang hilang BUKAN data operasional —
-- tidak ada satu klaim pun yang bergantung padanya, dan modul View History Claim tetap
-- dapat dibangun ulang tanpa isinya.
--
-- Yang hilang adalah JAWABAN atas pertanyaan "siapa pernah mencari data siapa".
--
-- Bobotnya perlu dinyatakan terang. `D-59` menetapkan satuan izin adalah menu dan TIDAK
-- ADA pemisahan tugas formal, sehingga jejak audit menjadi satu-satunya kontrol
-- pengimbang yang tersisa. Menghapus tabel ini berarti menghapus kontrol itu untuk
-- seluruh periode yang tercatat di dalamnya — dan tidak ada tempat lain yang
-- menyalinnya: `POOLDATA.LOG_DATA_PROTEKSI_KLAIM` milik sistem lama TIDAK menerima
-- satu baris pun dari aplikasi ini.
--
--
-- # Satu akibat kedua yang mudah terlewat
--
-- Sisa jatah pencarian DIHITUNG dari isi tabel ini, bukan disimpan. Menghapusnya
-- membuat seluruh pemakaian yang pernah tercatat menjadi tidak pernah terjadi, dan
-- setiap pengguna kembali memperoleh jatah penuh menurut master proteksi.
--
-- Bila itu yang diinginkan — mengembalikan jatah semua orang — ada cara yang jauh lebih
-- sempit daripada menghapus tabelnya, dan ia tidak menghapus jejak apa pun:
--
--     -- kembalikan jatah TANPA menghapus jejak: tambah jatahnya di master,
--     -- lewat layar Master Proteksi Data milik sistem lama.
--
-- Menghapus tabel ini untuk mengembalikan jatah adalah menghapus catatan untuk mengubah
-- angka. Jangan.
--
--
-- # Yang WAJIB dilakukan sebelum menjalankan berkas ini
--
-- 1. HITUNG DULU berapa yang akan hilang. Bila jawabannya bukan nol, berhenti dan bawa
--    angkanya ke Work Owner dan Compliance:
--
--        SELECT COUNT(*)                               AS seluruhnya,
--               COUNT(NILAI_PENCARIAN)                 AS berisi_nilai_pencarian,
--               MIN(DIPAKAI_PADA)                      AS terlama,
--               MAX(DIPAKAI_PADA)                      AS terbaru
--          FROM POOLDATA.CPNC_PEMAKAIAN_PROTEKSI;
--
-- 2. SALIN ISINYA LEBIH DULU bila ada satu baris pun. Salinan itu satu-satunya jalan
--    kembali:
--
--        CREATE TABLE POOLDATA.CPNC_PEMAKAIAN_PROTEKSI_ARSIP AS
--        SELECT * FROM POOLDATA.CPNC_PEMAKAIAN_PROTEKSI;
--
--    (SELECT * dipakai di sini dengan sengaja, dan ini satu-satunya tempatnya: yang
--    diinginkan justru SELURUH kolom apa adanya, termasuk kolom yang ditambahkan setelah
--    berkas ini ditulis.)
--
--    Salinannya memuat data nasabah pada kolom NILAI_PENCARIAN. Ia harus diperlakukan
--    setara dengan tabel aslinya, bukan sebagai tabel sementara yang boleh dibiarkan
--    terbuka.
--
-- 3. PASTIKAN APLIKASI SUDAH BERHENTI menulis ke tabel ini — modul View History Claim
--    dimatikan dari menu dan rutenya, atau aplikasinya dihentikan. Menjalankan berkas ini
--    saat aplikasi masih hidup membuat setiap pembukaan layar gagal dengan galat 500.
--
--
-- # Yang TIDAK disentuh berkas ini
--
-- POOLDATA.MST_PROTEKSI_DATA_PNC dan POOLDATA.LOG_DATA_PROTEKSI_KLAIM — keduanya milik
-- sistem lama, tidak pernah ditulis aplikasi ini, dan tidak ikut dihapus.


-- ---------------------------------------------------------------------------
-- Langkah 1 — indeks.
--
-- Dihapus lebih dulu, meski DROP TABLE akan membawanya serta. Menyebutkannya membuat
-- berkas ini terbaca sebagai kebalikan langsung dari berkas naiknya, dan membuat
-- kegagalan di tengah jalan lebih mudah ditelusuri.
-- ---------------------------------------------------------------------------

DROP INDEX POOLDATA.IX_CPNC_PEMAKAIAN_TELUSUR;
DROP INDEX POOLDATA.IX_CPNC_PEMAKAIAN_JATAH;


-- ---------------------------------------------------------------------------
-- Langkah 2 — urutan.
-- ---------------------------------------------------------------------------

DROP SEQUENCE POOLDATA.CPNC_PEMAKAIAN_PROTEKSI_SEQ;


-- ---------------------------------------------------------------------------
-- Langkah 3 — tabel.
--
-- TANPA PURGE, sehingga tabelnya masuk recycle bin dan masih dapat dipulihkan dengan
-- FLASHBACK selama recycle bin belum dibersihkan:
--
--     FLASHBACK TABLE POOLDATA.CPNC_PEMAKAIAN_PROTEKSI TO BEFORE DROP;
--
-- Ia jaring pengaman terakhir, bukan pengganti langkah 2 di atas.
-- ---------------------------------------------------------------------------

DROP TABLE POOLDATA.CPNC_PEMAKAIAN_PROTEKSI;
