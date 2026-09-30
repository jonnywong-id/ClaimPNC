-- Preliminary Loss Advice (PLA) koasuransi.
--
-- DITULIS modul ini, hanya untuk klaim yang dibuat aplikasi ini (keputusan Work Owner
-- 2026-09-27): POOLDATA.T_PLALIST (pengganti cabang PLA INSERT_PLADLA.prc) dan
-- POOLDATA.PLA (log penomoran, pengganti cabang PLA PLA_DLA.prc). Selebihnya DIBACA.

-- name: pla_koasuransi
--
-- CoinsList dokumen polis beserta CoinsID dan FlagDelete. Dokumennya sama dengan
-- polis_koasuransi; dua cabang karena POLICYDATA (CLOB) dan DATA_JSONBLOB (BLOB).
WITH terbaru AS (
    SELECT p.POLICYDATA, p.DATA_JSONBLOB
      FROM POOLDATA.JSON_POLIS p
     WHERE p.NOPOLIS = :1
       AND (p.POLICYDATA IS NOT NULL OR p.DATA_JSONBLOB IS NOT NULL)
     ORDER BY p.TGL_INPUT DESC
     FETCH FIRST 1 ROWS ONLY
)
SELECT jt.COINS_ID, jt.COINS_NAME, jt.LEADER, jt.PERCENT_SHARE, jt.FLAG_DELETE
  FROM terbaru t,
       JSON_TABLE(t.POLICYDATA, '$.CoinsList[*]' COLUMNS (
           COINS_ID      VARCHAR(50)  PATH '$.CoinsID',
           COINS_NAME    VARCHAR(200) PATH '$.CoinsName',
           LEADER        VARCHAR(10)  PATH '$.Leader',
           PERCENT_SHARE VARCHAR(50)  PATH '$.PercentShare',
           FLAG_DELETE   VARCHAR(10)  PATH '$.FlagDelete')) jt
 WHERE t.POLICYDATA IS NOT NULL
UNION ALL
SELECT jt.COINS_ID, jt.COINS_NAME, jt.LEADER, jt.PERCENT_SHARE, jt.FLAG_DELETE
  FROM terbaru t,
       JSON_TABLE(t.DATA_JSONBLOB, '$.CoinsList[*]' COLUMNS (
           COINS_ID      VARCHAR(50)  PATH '$.CoinsID',
           COINS_NAME    VARCHAR(200) PATH '$.CoinsName',
           LEADER        VARCHAR(10)  PATH '$.Leader',
           PERCENT_SHARE VARCHAR(50)  PATH '$.PercentShare',
           FLAG_DELETE   VARCHAR(10)  PATH '$.FlagDelete')) jt
 WHERE t.POLICYDATA IS NULL

-- name: pla_penerima
--
-- INSERT_PLADLA.prc: cocokkan kode DAN nama lebih dulu, lalu kode saja.
SELECT r.LOGIN, r.COUNTRY, r.EMAIL
  FROM POOLDATA.T_REINSURER r
 WHERE r.REINSURERID = :1
 ORDER BY CASE WHEN r.REINSURERNAME = :2 THEN 0 ELSE 1 END
 FETCH FIRST 1 ROWS ONLY

-- name: pla_sebelumnya
SELECT d.NOPLA, d.TGLPLA
  FROM POOLDATA.T_PLALIST d
 WHERE d.CLAIMID = :1 AND d.REINSCODE = :2
 ORDER BY d.TGLPLA DESC
 FETCH FIRST 1 ROWS ONLY

-- name: pla_terbit
SELECT d.NOPLA, d.PLAREINSURER, d.REINSCODE, d.TGLPLA, d.NOTES, d.CURRENCYPOLIS, d.JSON_PLA, d.EMAILPLA
  FROM POOLDATA.T_PLALIST d
 WHERE d.CLAIMID = :1 AND d.OBJECTID = :2 AND d.OBJECTCOVERAGEID = :3 AND d.REVISI = :4
   AND d.TIPEPLA = :5
 ORDER BY d.NOPLA

-- name: pla_site
SELECT s.ID
  FROM POOLDATA.M_SITE_DATABASE s
 WHERE s.CURRENT_SITE = '1'
 FETCH FIRST 1 ROWS ONLY

-- name: pla_urut
--
-- PENGECUALIAN DIALEK yang disadari: PLA_SEQ dipakai bersama Pega yang masih menerbitkan
-- PLA untuk klaim PNC-xxxx, sehingga nomor WAJIB diambil dari sequence yang sama agar tidak
-- bertabrakan. Padanan PostgreSQL-nya nextval('pooldata.pla_seq').
SELECT POOLDATA.PLA_SEQ.NEXTVAL FROM DUAL

-- name: pla_nomor_sisip
INSERT INTO POOLDATA.PLA (KEY, ID_PLA, KODE, ID_SITE, TAHUN, COUNT)
VALUES (:1, NULL, :2, :3, :4, :5)

-- name: pla_sisip
--
-- Kolom angka diisi 0 seperti baris Pega; nilainya disimpan di JSON_PLA.EstimasiList.
INSERT INTO POOLDATA.T_PLALIST
       (CLAIMID, OBJECTID, OBJECTCOVERAGEID, NOPLA, NILAIPLA, PLAREINSURER, REVISI, TIPEPLA,
        TGLPLA, NOTES, REINSCODE, CURRENCYPOLIS, PERCENTPLA, ESTIMASI, ESTIMASISHARE,
        EMAILPLA, LOGIN, COUNTRY, JSON_PLA)
VALUES (:1, :2, :3, :4, '0', :5, :6, :7, :8, :9, :10, :11, '0', '0', '0', :12, :13, :14, :15)

-- name: pla_ttd
SELECT m.NAME, m.JSONDATA
  FROM POOLDATA.MTTD m
 WHERE m.ID = :1

-- name: pla_catatan
--
-- INSERT_PLADLA.prc cabang PLA, baris sudah ada dan catatan terisi: NOTES diganti, ISPLA = 1.
UPDATE POOLDATA.T_PLALIST
   SET NOTES = :1, ISPLA = '1'
 WHERE CLAIMID = :2 AND NOPLA = :3 AND REVISI = :4

-- name: lod_email_tertanggung
--
-- Isian Email LOD (`SetDataEmailTertanggung` ← `ClaimData.Email`), yang diisi
-- `GetDataPengkinianDataTertanggung` dari `GetDataPengkinianData_SQLF`: OLDEMAIL baris
-- pengkinian data klaim itu. Pega mengambil baris pertama tanpa urutan.
SELECT u.OLDEMAIL
  FROM POOLDATA.UPDATE_PENGKINIANDATA u
 WHERE u.PYID = :1
 FETCH FIRST 1 ROWS ONLY

-- name: lod_email_pic
--
-- `ClaimData.UserTeknisEmail` — `BrowseEmailUserTeknis` (InputRegister_act langkah 67-68).
SELECT t.EMAIL
  FROM POOLDATA.MST_USER_TEKNIK t
 WHERE t.OPERATOR_ID = :1
 FETCH FIRST 1 ROWS ONLY
