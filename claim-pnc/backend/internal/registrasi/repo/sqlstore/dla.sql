-- Definite Loss Advice (DLA).
--
-- DITULIS modul ini: POOLDATA.T_DLALIST (pengganti cabang DLA INSERT_PLADLA.prc) dan
-- POOLDATA.DLA (log penomoran, pengganti cabang DLA PLA_DLA.prc). Selebihnya DIBACA:
-- T_PREDLALIST, JSON_POLIS, REINSURANCETYPE, dan tiga view treaty.

-- name: dla_terbit
--
-- GetDataDLA: DLA revisi 0 satu adjustment, terurut tanggal terbit. OBJECTID berisi
-- ObjectID objek; OBJECTCOVERAGEID dan ADJUSTMENTID urutan berbasis 1 (param.cvgID,
-- param.idAdj) — sama dengan baris Pega.
SELECT d.NODLA, d.DLAREINSURER, d.REINSCODE, d.TIPEDLA, d.TGLDLA, d.NOTES, d.NILAIDLA,
       d.PERCENTDLA, d.SHARESPREADING, d.KLAIMAMOUNT, d.CURRENCY, d.CURRENCYPOLIS, d.QS_PQS,
       d.QSRI, d.NOAKSEP, d.TGLAKSEP, d.EMAILDLA, d.LOGIN, d.COUNTRY, d.ISDLA, d.ISKIRIM
  FROM POOLDATA.T_DLALIST d
 WHERE d.CLAIMID = :1 AND d.OBJECTID = :2 AND d.OBJECTCOVERAGEID = :3 AND d.ADJUSTMENTID = :4
   AND (d.REVISI = '0' OR d.REVISI IS NULL)
 ORDER BY d.TGLDLA, d.NODLA

-- name: dla_predla
--
-- GetDataPreDLA: Pre DLA adjustment itu. Kolomnya sama dengan T_DLALIST.
SELECT d.NODLA, d.DLAREINSURER, d.REINSCODE, d.TIPEDLA, d.TGLDLA, d.NOTES, d.NILAIDLA,
       d.PERCENTDLA, d.SHARESPREADING, d.KLAIMAMOUNT, d.CURRENCY, d.CURRENCYPOLIS, d.QS_PQS,
       d.QSRI, d.NOAKSEP, d.TGLAKSEP, d.EMAILDLA, d.LOGIN, d.COUNTRY, d.ISDLA, d.ISKIRIM
  FROM POOLDATA.T_PREDLALIST d
 WHERE d.CLAIMID = :1 AND d.OBJECTID = :2 AND d.OBJECTCOVERAGEID = :3 AND d.ADJUSTMENTID = :4
   AND (d.REVISI = '0' OR d.REVISI IS NULL)
 ORDER BY d.TGLDLA, d.NODLA

-- name: dla_sebelumnya
--
-- INSERT_PLADLA.prc cabang DLA: DLA terakhir kepada penerima itu pada klaim yang sama.
SELECT d.NODLA, d.TGLDLA
  FROM POOLDATA.T_DLALIST d
 WHERE d.CLAIMID = :1 AND d.REINSCODE = :2
 ORDER BY d.TGLDLA DESC
 FETCH FIRST 1 ROWS ONLY

-- name: dla_polis_dokumen
--
-- Dokumen polis terbaru — sumber CoinsList, FacOfferList, SpreadingList, dan kepala polis.
-- Diurai di Go karena bentuk FacOfferList berbeda per Group Panel.
SELECT p.POLICYDATA, p.DATA_JSONBLOB
  FROM POOLDATA.JSON_POLIS p
 WHERE p.NOPOLIS = :1
   AND (p.POLICYDATA IS NOT NULL OR p.DATA_JSONBLOB IS NOT NULL)
 ORDER BY p.TGL_INPUT DESC
 FETCH FIRST 1 ROWS ONLY

-- name: dla_jenis_reas
--
-- GetTypeReinsuranceSpreading: `select type, note from reinsurancetype where id = …`.
SELECT TO_CHAR(r.ID), TO_CHAR(r.TYPE)
  FROM POOLDATA.REINSURANCETYPE r

-- name: dla_treaty_grup
--
-- GetTreatyGroupID. Pega tidak mengurutkan; baris pertama yang dipakai.
SELECT b.TREATYGROUPID
  FROM POOLDATA.TREATYBUSINESS b
 WHERE b.BIZCODE = :1 AND b.TREATYYEAR = :2 AND b.REINSTYPEID = :3
 FETCH FIRST 1 ROWS ONLY

-- name: dla_treaty_reas
--
-- SelectTreatyReinsurer — daftar penerima DLA treaty.
SELECT REPLACE(r.PCTSHARE, ',', '.'), r.NAME, r.REINSURERID, r.REINSTYPENAME
  FROM POOLDATA.TREATYREINSURER r
 WHERE r.TREATYYEAR = :1 AND r.TREATYGROUPID = :2 AND r.REINSTYPEID = :3

-- name: dla_treaty_limit
--
-- SelectProportionalArrg: limit RP (treatydescid 10003).
SELECT a.RP
  FROM POOLDATA.PROPORTIONALARRG a
 WHERE a.TREATYDESCID = '10003' AND a.REINSTYPEID = :1 AND a.TREATYYEAR = :2 AND a.TREATYGROUPID = :3
 FETCH FIRST 1 ROWS ONLY

-- name: dla_treaty_qs
--
-- searchQSReins2_SQL: persen QS. Pega mengambil baris TERAKHIR hasil tanpa urutan.
SELECT REPLACE(a.PCT, ',', '.')
  FROM POOLDATA.PROPORTIONALARRG a
 WHERE a.TREATYYEAR = :1 AND a.PARENTREINSTYPEID = :2 AND a.TREATYGROUPID = :3
   AND a.PCT IS NOT NULL AND a.TREATYDESCID = '10001'

-- name: dla_urut
--
-- PENGECUALIAN DIALEK yang disadari, sama seperti pla_urut: DLA_SEQ dipakai bersama Pega.
-- Padanan PostgreSQL-nya nextval('pooldata.dla_seq').
SELECT POOLDATA.DLA_SEQ.NEXTVAL FROM DUAL

-- name: dla_nomor_sisip
INSERT INTO POOLDATA.DLA (KEY, ID_DLA, KODE, ID_SITE, TAHUN, COUNT)
VALUES (:1, NULL, :2, :3, :4, :5)

-- name: dla_sisip
--
-- INSERT_PLADLA.prc cabang DLA, baris belum ada. TGLDLA = jam dinding WIB (SYSDATE Pega).
INSERT INTO POOLDATA.T_DLALIST
       (CLAIMID, OBJECTID, OBJECTCOVERAGEID, ADJUSTMENTID, NODLA, DLAREINSURER, REVISI, NOAKSEP,
        TIPEDLA, TGLDLA, NILAIDLA, NOTES, REINSCODE, CURRENCY, CURRENCYPOLIS, SHARESPREADING,
        PERCENTDLA, KLAIMAMOUNT, QS_PQS, QSRI, EMAILDLA, LOGIN, COUNTRY, TGLAKSEP, LOCATIONID)
VALUES (:1, :2, :3, :4, :5, :6, '0', :7, :8, :9, :10, :11, :12, :13, :14, :15, :16, :17, :18, :19,
        :20, :21, :22, :23, NULL)

-- name: dla_cetak
--
-- INSERT_PLADLA.prc cabang DLA, baris sudah ada dan catatan terisi (PRINT pertama,
-- DownloadDLA langkah 36): NOTES diganti, ISDLA = 1.
UPDATE POOLDATA.T_DLALIST
   SET NOTES = :1, ISDLA = '1'
 WHERE CLAIMID = :2 AND NODLA = :3 AND (REVISI = '0' OR REVISI IS NULL)
