-- name: rekening_ambil
--
-- Master Rekening (POOLDATA.LST_ACCOUNT) untuk isian No Rekening penerima klaim, pengganti
-- activity GetDataBankMaster yang tidak ada di export. Nomor yang sama dapat terdaftar di
-- lebih dari satu bank (kunci tabel ACCOUNT_NO + BANKID); yang BANKID-nya terkecil dipakai.
-- Status approval tidak menyaring: rekening yang masih menunggu approval komite juga
-- terpakai sebagai penerima di data Pega (119 dari 362 penerima bermaster).
SELECT ACCOUNT_NO, ACCOUNT_NAME, BANK_NAME, BANK_BRANCH, BANK_ADDRESS, BANKID, EMAIL, TELP,
       TANGGALAPPROVEKASIR, TANGGALAPPROVEKOMITE
  FROM POOLDATA.LST_ACCOUNT
 WHERE ACCOUNT_NO = :1
 ORDER BY BANKID
 FETCH FIRST 1 ROWS ONLY
