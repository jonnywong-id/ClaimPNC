-- 0010 — penanda Notice of Large Losses pernah terbit
--
-- # Kenapa kolom baru, bukan kolom yang sudah ada
--
-- Sistem lama menyimpan `ClaimData.FlagNOLL` di dalam BLOB work object Pega. Terverifikasi
-- 2026-09-24: TIDAK ADA satu pun kolom bernama `%NOLL%` di schema POOLDATA maupun DATAPEGA,
-- sehingga tidak ada kolom lama yang dapat dipakai ulang.
--
-- Menumpangkannya pada FLAG_KLAIM akan salah: `ADR-0018` menetapkan keempat konsep status
-- memang berbeda, dan FLAG_KLAIM adalah `ClaimStatus` — bukan penanda pemberitahuan.
--
-- # Apa yang ia tentukan
--
-- SUBJEK pemberitahuan berikutnya, bukan apakah ia dikirim:
--   kosong → "NOTICE OF LARGE LOSSES"
--   '1'    → "NOTICE OF LARGE LOSSES (REVISE)"
-- (`Activity/SendEmailLargeLoss_act.xml` langkah 7, 8, dan 11)
--
-- Tanpa kolom ini setiap pemberitahuan akan selamanya terbaca sebagai yang pertama, dan
-- penerima tidak punya cara tahu bahwa angkanya menggantikan angka sebelumnya.
--
-- # Dijalankan EMPAT KALI — sekali per portal (D-75)
--
-- Blok ini idempoten: dijalankan dua kali tidak menghasilkan galat.

DECLARE
  jumlah NUMBER;
BEGIN
  SELECT COUNT(*) INTO jumlah
    FROM ALL_TAB_COLUMNS
   WHERE OWNER = 'POOLDATA'
     AND TABLE_NAME = 'T_CLAIM_PNC'
     AND COLUMN_NAME = 'FLAG_NOLL';

  IF jumlah = 0 THEN
    EXECUTE IMMEDIATE 'ALTER TABLE POOLDATA.T_CLAIM_PNC ADD (FLAG_NOLL VARCHAR2(1))';
  END IF;
END;
