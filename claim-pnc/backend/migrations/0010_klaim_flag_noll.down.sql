-- Membatalkan 0010.
--
-- Menghapus kolom ini MENGHILANGKAN jejak bahwa sebuah klaim sudah pernah menerbitkan
-- Notice of Large Losses. Setelah dibatalkan, pemberitahuan berikutnya atas klaim yang
-- sama akan terbaca sebagai yang pertama — bukan revisi.
--
-- Idempoten: dijalankan pada skema yang kolomnya sudah tidak ada tidak menghasilkan galat.

DECLARE
  jumlah NUMBER;
BEGIN
  SELECT COUNT(*) INTO jumlah
    FROM ALL_TAB_COLUMNS
   WHERE OWNER = 'POOLDATA'
     AND TABLE_NAME = 'T_CLAIM_PNC'
     AND COLUMN_NAME = 'FLAG_NOLL';

  IF jumlah = 1 THEN
    EXECUTE IMMEDIATE 'ALTER TABLE POOLDATA.T_CLAIM_PNC DROP COLUMN FLAG_NOLL';
  END IF;
END;
