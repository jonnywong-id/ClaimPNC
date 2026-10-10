-- 0015 down — membatalkan sasaran perubahan Cause of Loss pada T_CLAIM_OPENPROTECTION
--
-- MENJALANKAN BERKAS INI MENGHAPUS DATA, BUKAN HANYA KOLOM: permintaan perubahan Cause of
-- Loss yang sudah dibuat akan kehilangan SASARANNYA — baris permintaannya tetap ada, tetapi
-- coverage mana yang hendak diubah hilang dan tidak dapat dipulihkan dari kolom lain.
-- `OBJECT_NAME` tidak menggantikannya: ia nama, bukan kunci.
--
-- Hanya sah bila kedua kolom belum pernah diisi. Bila sudah, hentikan pemakaiannya lebih
-- dulu dan hapus pada rilis berikutnya (`P-4`, penghapusan dua tahap).
--
-- Memastikannya sebelum menjalankan — diharapkan NOL pada keduanya:
--
--     SELECT COUNT(OBJECT_ID)          AS OBJECT_ID_TERISI,
--            COUNT(OBJECT_COVERAGE_ID) AS OBJECT_COVERAGE_ID_TERISI
--       FROM POOLDATA.T_CLAIM_OPENPROTECTION;

ALTER TABLE POOLDATA.T_CLAIM_OPENPROTECTION DROP (OBJECT_ID, OBJECT_COVERAGE_ID);
