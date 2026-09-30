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
