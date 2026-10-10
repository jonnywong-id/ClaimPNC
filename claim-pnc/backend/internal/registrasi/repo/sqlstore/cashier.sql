-- Transfer Kasir.
--
-- DIBACA: GENERAL.LST_BANK_GROUP (lewat DB link ASMD, satu-satunya tempat tabel itu ada).
-- DITULIS: POOLDATA.TRF_KASIR_LOG dan dua kolom T_CLAIM_ADJUSTMENT.

-- name: kasir_kode_bank
--
-- TransferToKasir_act :13537 — `select LBG_ID as "CityID" FROM GENERAL.LST_BANK_GROUP
-- WHERE BANK_GROUP = {nama bank}`; baris yang dipakai adalah yang sama dengan IDBank penerima.
SELECT b.LBG_ID
  FROM GENERAL.LST_BANK_GROUP@asmd.sinarmas.co.id b
 WHERE b.BANK_GROUP = :1

-- name: kasir_log
--
-- InsertLogKasir_sql. TGL_TRF diisi nilai bawaan tabelnya, seperti Pega.
INSERT INTO POOLDATA.TRF_KASIR_LOG (NOAKSEPTASI, NOKLAIM, PIC, STATUS, ALASAN)
VALUES (:1, :2, :3, :4, :5)

-- name: kasir_tandai
--
-- Transfer berhasil: TransferCashierDate (bila masih kosong) dan CaseIDCashier.
UPDATE POOLDATA.T_CLAIM_ADJUSTMENT
   SET TRANSFER_CASHIER_DATE = COALESCE(TRANSFER_CASHIER_DATE, :1), IDCHASIER = :2
 WHERE CLAIMID = :3 AND OBJECTID = :4 AND OBJECTCOVERAGEID = :5 AND ADJUSTMENTID = :6

-- name: kasir_rekening_terdaftar
--
-- `GetDataBankMaster` langkah 5 (`GetFileOnPc_link_attachmentGCNM` atas SQL yang DIRANGKAI dari
-- No Rekening — di sini parameter binding): rekening aktif di master rekening Kasir.
SELECT COUNT(*)
  FROM COLLECTION.LST_ACCOUNT@asmd.sinarmas.co.id a
 WHERE a.ACCOUNT_NO = :1 AND a.LBG_ID = :2 AND a.STS_AKTIF = '1'

-- name: kasir_log_layanan
--
-- Satu baris POOLDATA.CLAIM_SERVICE_LOG jenis "Log Kasir" — padanan `InsertUpdateLogService`
-- + `UpdateLogServiceClaim`, tetapi SATU INSERT berisi JSONIN dan JSONOUT sekaligus (Steering
-- §8 menolak UPDATE pada tabel log ini). INSERTDATE memakai default kolomnya (sysdate).
--
-- ID mengikuti `QueryLogServiceClaim`: MAX(ID)+1. Tabel ini juga ditulis Pega dan tidak punya
-- sequence; kunci utamanya (SERVICEID, ID), sehingga tabrakan hanya mungkin bila dua penulis
-- menyisipkan log klaim yang SAMA pada saat yang sama.
INSERT INTO POOLDATA.CLAIM_SERVICE_LOG (SERVICEID, JSONIN, JSONOUT, CATEGORYSERVICE, ID, SERVICEREF)
SELECT :1, :2, :3, :4, COALESCE(MAX(l.ID), 0) + 1, :5
  FROM POOLDATA.CLAIM_SERVICE_LOG l

-- name: kasir_riwayat
--
-- Grid "Histori Transfer Kasir" (Section/InputAdjustment_sect.xml, TempDataLogKasir). Rule
-- pengisinya tidak ada di export; kolomnya diturunkan dari InsertLogKasir_sql, yang menulis
-- tabel yang sama. Baca saja.
SELECT PIC, TGL_TRF, STATUS, ALASAN
  FROM POOLDATA.TRF_KASIR_LOG
 WHERE NOAKSEPTASI = :1
 ORDER BY TGL_TRF
