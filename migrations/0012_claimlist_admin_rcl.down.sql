-- 0012 down — membatalkan penambahan tiga kolom Inbox RCL pada T_CLAIMLIST_ADMIN
--
-- ============================================================================
-- MENJALANKAN BERKAS INI MENGHAPUS DATA, BUKAN HANYA KOLOM.
-- ============================================================================
--
-- `DROP COLUMN` membuang isinya. Down hanya sah bila ketiga kolom BELUM PERNAH
-- DIISI — 0012 baru dijalankan lalu langsung dibatalkan. Bila proses pengisi
-- sudah menulisnya, jangan jalankan berkas ini: hentikan pemakaiannya lebih dulu,
-- lalu hapus pada rilis berikutnya (`P-4`, `D-66`).
--
-- Sebelum menjalankan, hentikan modul Inbox RCL: kuerinya membaca ketiga kolom
-- ini dan akan gagal dengan ORA-00904 begitu kolomnya hilang.

ALTER TABLE POOLDATA.T_CLAIMLIST_ADMIN DROP (
    TANGGALANALYSTSENDRCL_1,
    NAMADOKTERRCL_1,
    KOMENTARANALISATOR_1
);
