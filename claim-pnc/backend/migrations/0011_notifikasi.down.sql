-- Membatalkan 0011.
--
-- Menghapus tabel ini MENGHILANGKAN peristiwa yang belum sempat dikirim `S-3`, dan
-- peristiwa Notice of Large Losses tidak dapat diterbitkan ulang: klaimnya sudah tersimpan
-- dan penandanya sudah dinaikkan. Jalankan hanya setelah memastikan tidak ada baris
-- ber-DIKIRIM_PADA kosong.
--
-- Idempoten: dijalankan pada skema yang tabelnya sudah tidak ada tidak menghasilkan galat.

DECLARE
  jumlah NUMBER;
BEGIN
  SELECT COUNT(*) INTO jumlah
    FROM ALL_TABLES
   WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'CPNC_NOTIFIKASI';

  IF jumlah = 1 THEN
    EXECUTE IMMEDIATE 'DROP TABLE POOLDATA.CPNC_NOTIFIKASI';
  END IF;
END;
