-- Claim Face Sheet: data pendamping dan revisi CFSList.
--
-- Yang DITULIS modul ini hanya POOLDATA.TC_PNC_CFS dan TC_PNC_CFS_ESTIMASI — pengganti
-- `ObjectCoverageList().CFSList()` pada JSON klaim Pega, dan hanya untuk klaim yang dibuat
-- aplikasi ini (P-1). Selebihnya DIBACA.

-- name: cfs_penyebab
--
-- Uraian penyebab kerugian — baris NATURE OF LOSS. CAUSEOFLOSSID jaminan menunjuk
-- D_COL_ID; dicocokkan pada 2.346 dari 2.556 jaminan di portal ASM (2026-09-27).
SELECT DESCRIPTION
  FROM POOLDATA.V_D_CAUSE_OF_LOSS
 WHERE D_COL_ID = :1
 FETCH FIRST 1 ROWS ONLY

-- name: cfs_operator
--
-- Nama operator — baris PIC Admin (`pyWorkPage.pxCreateOpName`). PR_OPERATORS memuat
-- seluruh operator Pega; M_LOGIN_PNC cadangan untuk login non-karyawan.
SELECT o.PYUSERNAME
  FROM DATAPEGA.PR_OPERATORS o
 WHERE UPPER(o.PYUSERIDENTIFIER) = UPPER(:1)
UNION ALL
SELECT l.LOGIN_NAME
  FROM POOLDATA.M_LOGIN_PNC l
 WHERE l.LOGIN_ID = :2

-- name: cfs_fac_offer
--
-- FacOfferList dokumen polis — baris REINS FAC OUT MEMBER SHARE. Dokumennya sama dengan
-- polis_koasuransi; dua cabang karena POLICYDATA (CLOB) dan DATA_JSONBLOB (BLOB) tidak
-- dapat digabung sebelum JSON_TABLE.
WITH terbaru AS (
    SELECT p.POLICYDATA, p.DATA_JSONBLOB
      FROM POOLDATA.JSON_POLIS p
     WHERE p.NOPOLIS = :1
       AND (p.POLICYDATA IS NOT NULL OR p.DATA_JSONBLOB IS NOT NULL)
     ORDER BY p.TGL_INPUT DESC
     FETCH FIRST 1 ROWS ONLY
)
SELECT jt.REINSURER_NAME, jt.PCT_SHARE
  FROM terbaru t,
       JSON_TABLE(t.POLICYDATA, '$.FacOfferList[*]' COLUMNS (
           REINSURER_NAME VARCHAR(200) PATH '$.ReinsurerName',
           PCT_SHARE      VARCHAR(50)  PATH '$.PctShareForAllObj')) jt
 WHERE t.POLICYDATA IS NOT NULL
UNION ALL
SELECT jt.REINSURER_NAME, jt.PCT_SHARE
  FROM terbaru t,
       JSON_TABLE(t.DATA_JSONBLOB, '$.FacOfferList[*]' COLUMNS (
           REINSURER_NAME VARCHAR(200) PATH '$.ReinsurerName',
           PCT_SHARE      VARCHAR(50)  PATH '$.PctShareForAllObj')) jt
 WHERE t.POLICYDATA IS NULL

-- name: cfs_revisi_terakhir
SELECT MAX(REVISI)
  FROM POOLDATA.TC_PNC_CFS
 WHERE CLAIMID = :1 AND OBJECTID = :2 AND OBJECTCOVERAGEID = :3

-- name: cfs_sisip
INSERT INTO POOLDATA.TC_PNC_CFS
       (CLAIMID, OBJECTID, OBJECTCOVERAGEID, REVISI, CFSDATE, FILENAME)
VALUES (:1, :2, :3, :4, :5, :6)

-- name: cfs_estimasi_sisip
--
-- CURRENCY berupa teks nama mata uang ("IDR"), bukan kode 10026 — sama dengan
-- CFSList().EstimasiList() pada JSON klaim Pega.
INSERT INTO POOLDATA.TC_PNC_CFS_ESTIMASI
       (CLAIMID, OBJECTID, OBJECTCOVERAGEID, REVISI, URUTAN, CURRENCY, ESTIMATIONDATE,
        ESTIMATIONVALUE)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8 / 100)
