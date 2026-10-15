-- 0013 turun — membuang tabel baris klaim Master Recovery.
--
-- ============================================================================
-- YANG HILANG BILA INI DIJALANKAN
-- ============================================================================
--
-- **Seluruh daftar polis pada setiap batch yang dicatat aplikasi baru.**
--
-- Ia tidak dapat dipulihkan dari mana pun: sejak `0013` naik, aplikasi berhenti mengisi
-- kolom `JSON_POLIS`, sehingga tidak ada salinan kedua. Batch-nya sendiri tetap utuh di
-- MST_RECOVERY_ASM_PENJAMINAN — yang hilang adalah rincian polis di bawahnya.
--
-- Karena itu berkas ini bukan langkah pembatalan yang wajar. Ia ada supaya `P-4` terpenuhi
-- — setiap migrasi punya `down` yang benar-benar berfungsi — bukan supaya dijalankan.
--
-- Sebelum menjalankannya, salin isinya lebih dulu:
--
--     CREATE TABLE POOLDATA.CPNC_RECOVERY_BARIS_KLAIM_ARSIP
--         AS SELECT * FROM POOLDATA.CPNC_RECOVERY_BARIS_KLAIM;
--
-- Indeksnya ikut terbuang bersama tabelnya; ia tidak perlu dibuang terpisah.

DECLARE
    sudah_ada NUMBER;
BEGIN
    SELECT COUNT(*) INTO sudah_ada
      FROM ALL_TABLES
     WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'CPNC_RECOVERY_BARIS_KLAIM';

    IF sudah_ada > 0 THEN
        EXECUTE IMMEDIATE 'DROP TABLE POOLDATA.CPNC_RECOVERY_BARIS_KLAIM';
    END IF;
END;
/
