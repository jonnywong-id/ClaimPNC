-- name: rekening_ambil
--
-- Master Rekening (POOLDATA.LST_ACCOUNT) untuk isian No Rekening penerima klaim —
-- `GetDataBankMaster` langkah 4 (`GetDataPenerimaKlaim`). Nomor yang sama dapat terdaftar di
-- lebih dari satu bank (kunci tabel ACCOUNT_NO + BANKID).
--
-- `GetDataPenerimaKlaim` menyaring APPROVAL = '1'. Di sini barisnya tetap dibaca beserta
-- APPROVAL-nya — supaya rekening yang sedang proses approval mendapat pesan yang benar, bukan
-- "tidak terdaftar" — dan baris yang disetujui didahulukan, lalu BANKID terkecil.
SELECT ACCOUNT_NO, ACCOUNT_NAME, BANK_NAME, BANK_BRANCH, BANK_ADDRESS, BANKID, EMAIL, TELP,
       TANGGALAPPROVEKASIR, TANGGALAPPROVEKOMITE, APPROVAL
  FROM POOLDATA.LST_ACCOUNT
 WHERE ACCOUNT_NO = :1
 ORDER BY CASE WHEN APPROVAL = '1' THEN 0 ELSE 1 END, BANKID
 FETCH FIRST 1 ROWS ONLY
