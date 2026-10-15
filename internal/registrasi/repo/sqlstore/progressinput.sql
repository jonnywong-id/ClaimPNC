-- Modal Input Progress Claim — lihat registrasi/progressinput.go.

-- name: progres_input_posisi
--
-- ViewStatusProgress: posisi progres klaim, urut PROGRESSDATE.
SELECT p.ID, p.CASEID, p.POSISI, p.STATUSPOSISI, p.PROGRESSDATE
  FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC p
 WHERE p.CLAIMNO = :1
 ORDER BY p.PROGRESSDATE ASC, p.ID ASC

-- name: progres_input_status1
--
-- Status Progress 1. Aktivitas pemuatnya (`BrowseLastStatusProgress_act`) belum ada di export;
-- sementara dari master GCNM_MST_PROGRESS_KLAIM yang berlaku untuk semua (STATUS 'ALL').
SELECT m.ID_PROGRESS, m.STS_PROGRESS1
  FROM POOLDATA.GCNM_MST_PROGRESS_KLAIM m
 WHERE m.STATUS = 'ALL'
 ORDER BY m.STS_PROGRESS1 ASC

-- name: progres_input_status
--
-- Dropdown Status. Daftarnya hanya ada di `BrowseLastStatusProgress_act` yang belum ada di
-- export; sementara dari nilai STATUS yang pernah tersimpan.
SELECT DISTINCT TRIM(g.STATUS)
  FROM POOLDATA.GCNM_PROGRESS_CLAIM g
 WHERE TRIM(g.STATUS) IS NOT NULL
 ORDER BY 1

-- name: progres_input_tipe_dokumen
--
-- Tipe Dokumen (Status Progress 1 = 003). Data Transform `SetDocType` tidak ada di Pega;
-- pilihannya mengikuti kolom TIPE yang disaring BrowseStsProgressDoc.
SELECT DISTINCT TRIM(m.TIPE)
  FROM POOLDATA.GCNM_MST_PROGRESS m
 WHERE m.ID_PROGRESS = '003'
   AND TRIM(m.TIPE) IS NOT NULL
 ORDER BY 1

-- name: progres_input_perlu
--
-- BrowseStatusProgress2_sql tanpa penyaring tipe.
SELECT a.ID_MST, a.STS_PROGRESS2
  FROM POOLDATA.GCNM_MST_PROGRESS a
 WHERE a.ID_PROGRESS = :1
 ORDER BY LENGTH(a.ID_MST) ASC, a.ID_MST ASC

-- name: progres_input_perlu_tipe
--
-- BrowseStatusProgress2_sql dengan `and tipe='…'` (BrowseStsProgressDoc).
SELECT a.ID_MST, a.STS_PROGRESS2
  FROM POOLDATA.GCNM_MST_PROGRESS a
 WHERE a.ID_PROGRESS = :1
   AND a.TIPE = :2
 ORDER BY LENGTH(a.ID_MST) ASC, a.ID_MST ASC

-- name: progres_input_nama_perlu
--
-- ViewStatusProgress111.
SELECT a.STS_PROGRESS2
  FROM POOLDATA.GCNM_MST_PROGRESS a
 WHERE a.ID_MST = :1

-- name: progres_input_hari_ini
--
-- ValidationProgressClaim_act1 langkah 11 atas grid ViewStatusProgress11: STATUS PROGRESS 2
-- progres hari ini yang KETERANGAN-nya tidak mengandung "AUTO". JSON kosong dilewati karena
-- GET_POSISI_PROGRESS2 gagal atasnya.
SELECT POOLDATA.GET_POSISI_PROGRESS2(a.PNCCASEID, a.ID_UPDATE, a.JSONSTATUS_PROGRESS2)
  FROM POOLDATA.GCNM_PROGRESS_CLAIM a
 WHERE a.PNCCASEID = :1
   AND a.TGL_INPUT >= :2
   AND a.TGL_INPUT < :3
   AND UPPER(COALESCE(a.KETERANGAN, ' ')) NOT LIKE '%AUTO%'
   AND a.JSONSTATUS_PROGRESS2 IS NOT NULL

-- name: progres_input_sisip
--
-- PROGRESS_CLAIM_PNC.prc cabang ELSE (`:90-101`): POSISIID posisi yang dipilih, JSON dan
-- DOCTYPE dari pemanggil, TGL_INPUT bawaan kolom (sysdate).
INSERT INTO POOLDATA.GCNM_PROGRESS_CLAIM
       (ID_UPDATE, PNCCASEID, KETERANGAN, STATUS_PROGRESS1, STATUS_PROGRESS2, STATUS,
        NEXT_FOLLOWUP, USER_INPUT, POSISIID, JSONSTATUS_PROGRESS2, DOCTYPE)
SELECT COALESCE(MAX(g.ID_UPDATE), 0) + 1, :1, :2, :3, :4, :5, :6, :7, :8, :9, :10
  FROM POOLDATA.GCNM_PROGRESS_CLAIM g
 WHERE g.PNCCASEID = :11

-- name: progres_input_email_broker
--
-- Email Progress Klaim yang tersimpan untuk polis ini (MST_EMAIL_BROKER_KLAIM).
SELECT b.EMAILBROKER
  FROM POOLDATA.MST_EMAIL_BROKER_KLAIM b
 WHERE b.NOPOLIS = :1 AND b.GROUPPANEL = :2 AND b.BUSINESSCODE = :3 AND b.BRANCHCODE = :4
   AND b.EMAILBROKER IS NOT NULL
 FETCH FIRST 1 ROWS ONLY

-- name: progres_input_broker_hitung
--
-- INSERTDATABROKEREMAILMASTER: baris master polis yang sama.
SELECT COUNT(1)
  FROM POOLDATA.MST_EMAIL_BROKER_KLAIM a
 WHERE a.NOPOLIS = :1 AND a.GROUPPANEL = :2 AND a.BUSINESSCODE = :3 AND a.BRANCHCODE = :4

-- name: progres_input_broker_agen
--
-- INSERTDATABROKEREMAILMASTER: TOPLEADER dan ID sumber bisnis polis.
SELECT g.TOPLEADER, g.ID
  FROM POOLDATA.AGENT g
 WHERE g.ID IN (SELECT t.SOURCEOFBUSINESS FROM POOLDATA.T_GENERAL t WHERE t.NOPOLIS = :1 FETCH FIRST 1 ROWS ONLY)

-- name: progres_input_broker_sisip
INSERT INTO POOLDATA.MST_EMAIL_BROKER_KLAIM
       (NOPOLIS, BRANCHCODE, BUSINESSCODE, GROUPPANEL, EMAILBROKER, KLAIMID, BROKERTYPE, SUMBIS)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8)

-- name: progres_input_broker_ubah
UPDATE POOLDATA.MST_EMAIL_BROKER_KLAIM a
   SET a.EMAILBROKER = :1
 WHERE a.NOPOLIS = :2 AND a.GROUPPANEL = :3 AND a.BUSINESSCODE = :4 AND a.BRANCHCODE = :5
   AND a.BROKERTYPE = :6 AND a.SUMBIS = :7
