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
-- Fac Offer polis — baris REINS FAC OUT MEMBER SHARE.
--
-- Sumbernya POOLDATA.T_FACOFFER menurut POLICYNO dan PRODKE snapshot klaim (Work Owner,
-- 2026-10-01). Satu baris per reasuradur, tetapi JSONDATA setiap baris memuat SELURUH
-- FacOfferList polis — karena itu hanya entri yang ReinsurerID-nya sama dengan
-- REINSURER_ID baris itu yang diambil; tanpa saringan itu reasuradur terhitung berkali-kali.
SELECT jt.REINSURER_NAME, jt.PCT_SHARE
  FROM POOLDATA.T_FACOFFER f,
       JSON_TABLE(f.JSONDATA, '$.FacOfferList[*]' COLUMNS (
           REINSURER_ID   VARCHAR(50)  PATH '$.ReinsurerID',
           REINSURER_NAME VARCHAR(200) PATH '$.ReinsurerName',
           PCT_SHARE      VARCHAR(50)  PATH '$.PctShareForAllObj')) jt
 WHERE f.POLICYNO = :1
   AND f.PRODKE = :2
   AND TRIM(jt.REINSURER_ID) = TRIM(f.REINSURER_ID)
 ORDER BY f.REINSURER_ID

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
