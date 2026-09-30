-- Persetujuan / Akseptasi LOD (SetAdjustmentAcceptation) untuk klaim yang dibuat aplikasi ini.
--
-- Kolom yang sudah ada di T_CLAIM_ADJUSTMENT ditulis akseptasi_simpan; tujuh kolom isian form
-- dari migrasi 0013 ditulis akseptasi_isian_simpan dan dibaca akseptasi_isian_ambil, terpisah,
-- agar klaim tetap dapat dimuat sebelum migrasi itu dijalankan DBA.
--
-- Kolom DATE diisi jam dinding WIB, seperti SYSDATE server pada sistem lama.

-- name: akseptasi_site
SELECT s.ID
  FROM POOLDATA.M_SITE_DATABASE s
 WHERE s.CURRENT_SITE = '1'
 FETCH FIRST 1 ROWS ONLY

-- name: akseptasi_urut
--
-- PENGECUALIAN DIALEK yang disadari, sama seperti PLA_SEQ: ACCEPTLOD_SEQ dipakai bersama
-- Pega, sehingga nomor wajib diambil dari sequence yang sama agar tidak bertabrakan.
-- Padanan PostgreSQL-nya nextval('pooldata.acceptlod_seq').
SELECT POOLDATA.ACCEPTLOD_SEQ.NEXTVAL FROM DUAL

-- name: akseptasi_nomor_sisip
--
-- PLA_DLA.prc cabang TIPE ALOD.
INSERT INTO POOLDATA.ACCEPTLOD (KEY, ID_ALOD, KODE, ID_SITE, TAHUN, COUNT)
VALUES (:1, NULL, :2, :3, :4, :5)

-- name: akseptasi_simpan
UPDATE POOLDATA.T_CLAIM_ADJUSTMENT
   SET STATUSAKSEPTASILOD = :1, NOAKSEPTASI = :2, TGLAKSEPTASI = :3, RECEIVER = :4,
       RECEIVERNAME = :5, PRINTLOD_DATE = :6, RECEIVEDATELOD = :7
 WHERE CLAIMID = :8 AND OBJECTID = :9 AND OBJECTCOVERAGEID = :10 AND ADJUSTMENTID = :11

-- name: lod_cetak_simpan
--
-- Print LOD: PRINTLOD_DATE hanya bila masih kosong (AutoPrintPDFDraftLOD langkah 1), PDFTYPE
-- jenis yang dicetak. Kedua kolom sudah ada; bukan kolom migrasi.
UPDATE POOLDATA.T_CLAIM_ADJUSTMENT
   SET PRINTLOD_DATE = COALESCE(PRINTLOD_DATE, :1), PDFTYPE = :2
 WHERE CLAIMID = :3 AND OBJECTID = :4 AND OBJECTCOVERAGEID = :5 AND ADJUSTMENTID = :6

-- name: akseptasi_isian_simpan
--
-- Kolom migrasi 0013.
UPDATE POOLDATA.T_CLAIM_ADJUSTMENT
   SET TANGGALBOLEHBAYAR = :1, RECEIVEDATEANALIST = :2, ACCEPTANCEVALUELOD = :3 / 100,
       TIPEAKSEPTASI = :4, KOMITEACCEPTED = :5, REMARKACCEPTED = :6, UPLOADNOTELOD = :7
 WHERE CLAIMID = :8 AND OBJECTID = :9 AND OBJECTCOVERAGEID = :10 AND ADJUSTMENTID = :11

-- name: akseptasi_isian_ambil
SELECT TANGGALBOLEHBAYAR, RECEIVEDATEANALIST, ROUND(ACCEPTANCEVALUELOD * 100), TIPEAKSEPTASI,
       KOMITEACCEPTED, REMARKACCEPTED, UPLOADNOTELOD
  FROM POOLDATA.T_CLAIM_ADJUSTMENT
 WHERE CLAIMID = :1 AND OBJECTID = :2 AND OBJECTCOVERAGEID = :3 AND ADJUSTMENTID = :4

-- name: akseptasi_dla_lain
--
-- ValidationDLA_Act: setiap adjustment klaim yang Persetujuan Tertanggung-nya 1 dan bukan
-- Ex-Gratia, DLA revisi 0 miliknya harus sudah dicetak (ISDLA) dan dikirim (ISKIRIM).
-- Adjustment yang sedang diakseptasi belum bernomor sehingga tidak punya DLA; ia tetap
-- dikecualikan agar aturan ini tidak bergantung pada urutan penulisan.
SELECT d.NODLA, d.ISDLA, d.ISKIRIM
  FROM POOLDATA.T_DLALIST d
  JOIN POOLDATA.T_CLAIM_ADJUSTMENT a
    ON a.CLAIMID = d.CLAIMID AND a.OBJECTID = d.OBJECTID
   AND a.OBJECTCOVERAGEID = d.OBJECTCOVERAGEID AND a.ADJUSTMENTID = d.ADJUSTMENTID
 WHERE d.CLAIMID = :1
   AND a.STATUSAKSEPTASILOD = '1'
   AND COALESCE(a.EXGRATIA, '0') <> '1'
   AND COALESCE(d.REVISI, '0') = '0'
   AND NOT (a.OBJECTID = :2 AND a.OBJECTCOVERAGEID = :3 AND a.ADJUSTMENTID = :4)
 ORDER BY d.TGLDLA

-- name: riwayat_sisip
--
-- PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc.
INSERT INTO POOLDATA.LIST_HISTORY_CLAIM_PNC (CASEID, CREATEDATETIME, STATUSNOTE, USERUPDATE)
VALUES (:1, :2, :3, :4)

-- name: operator_lama
--
-- GetOperatorIDs: USER_INPUT progres memakai identitas lama pengguna bila ada.
SELECT t.OLD_OPERATOR_ID
  FROM POOLDATA.T_ACCESS_GROUP_PNC t
 WHERE t.OPERATOR_ID = :1 AND t.STS_AKTIF = '1'
 FETCH FIRST 1 ROWS ONLY

-- name: progres_posisi_terbuka
--
-- Posisi terbuka terakhir klaim itu menurut nama posisinya: AKSEPTASI (dibuka
-- KomitePost_Adjustment langkah 19–21, ditutup SetAdjustmentAcceptation langkah 69–70 /
-- 95–96) atau KOMITE (ditutup KomitePost_Adjustment langkah 22–23 / 61–62).
SELECT ID
  FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC
 WHERE CLAIMNO = :1 AND POSISI = :2 AND STATUSPOSISI = 'On Progress'
 ORDER BY ID DESC
 FETCH FIRST 1 ROWS ONLY

-- name: progres_posisi_berikut
--
-- PROGRESS_CLAIM_PNC.prc kategori INSERT: nvl(max(ID),0)+1 per CLAIMNO.
SELECT COALESCE(MAX(p.ID), 0) + 1
  FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC p
 WHERE p.CLAIMNO = :1

-- name: progres_posisi_sisip
INSERT INTO POOLDATA.GCNM_PROGRESS_POSISI_PNC (ID, CLAIMNO, CASEID, STATUSPOSISI, PROGRESSDATE, POSISI)
VALUES (:1, :2, :3, :4, :5, :6)

-- name: progres_sisip_buka
--
-- PROGRESS_CLAIM_PNC.prc kategori INSERT, langkah kedua: STATUS dari pstsprog (tidak diisi
-- pemanggil ini), NEXT_FOLLOWUP sysdate+7 bila tglfollow kosong, POSISIID posisi baru.
INSERT INTO POOLDATA.GCNM_PROGRESS_CLAIM
       (ID_UPDATE, PNCCASEID, KETERANGAN, STATUS_PROGRESS1, STATUS_PROGRESS2, STATUS,
        NEXT_FOLLOWUP, USER_INPUT, POSISIID, JSONSTATUS_PROGRESS2)
SELECT COALESCE(MAX(g.ID_UPDATE), 0) + 1, :1, :2, :3, :4, NULL, :5, :6, :7, :8
  FROM POOLDATA.GCNM_PROGRESS_CLAIM g
 WHERE g.PNCCASEID = :9

-- name: progres_posisi_selesai
--
-- PROGRESS_CLAIM_PNC.prc kategori UPDATE, langkah pertama.
UPDATE POOLDATA.GCNM_PROGRESS_POSISI_PNC
   SET STATUSPOSISI = :1, PROGRESSDATEDONE = :2
 WHERE CLAIMNO = :3 AND ID = :4

-- name: progres_sisip
--
-- PROGRESS_CLAIM_PNC.prc kategori UPDATE, langkah kedua: NEXT_FOLLOWUP dan STATUS kosong,
-- TGL_INPUT bawaan kolom (sysdate). ID_UPDATE lanjutan per nomor klaim, seperti prosedurnya.
INSERT INTO POOLDATA.GCNM_PROGRESS_CLAIM
       (ID_UPDATE, PNCCASEID, KETERANGAN, STATUS_PROGRESS1, STATUS_PROGRESS2, STATUS,
        NEXT_FOLLOWUP, USER_INPUT, POSISIID, JSONSTATUS_PROGRESS2)
SELECT COALESCE(MAX(g.ID_UPDATE), 0) + 1, :1, :2, :3, :4, NULL, NULL, :5, :6, :7
  FROM POOLDATA.GCNM_PROGRESS_CLAIM g
 WHERE g.PNCCASEID = :8

-- name: premi_polis_caseid
--
-- Policy.CaseID untuk parameter caseId layanan premi (GetStatusPremi langkah 3), dari
-- dokumen polis yang sama dengan polis_ambil.
SELECT COALESCE(JSON_VALUE(p.POLICYDATA, '$.CaseID'), JSON_VALUE(p.DATA_JSONBLOB, '$.CaseID'))
  FROM POOLDATA.JSON_POLIS p
 WHERE p.NOPOLIS = :1
   AND (p.POLICYDATA IS NOT NULL OR p.DATA_JSONBLOB IS NOT NULL)
 ORDER BY p.TGL_INPUT DESC
 FETCH FIRST 1 ROWS ONLY

-- name: premi_open_protection
--
-- InboxOpenProtection_RD dengan TypePro 2 (premi) atas T_CLAIM_OPENPROTECTION: polis sama,
-- nomor klaim pada ID_CLAIM atau CLAIM_NO, disetujui, dan aktif. Penyaring IsUsedPNC tidak
-- dapat dibawa karena tabel ini tidak punya kolomnya.
SELECT COUNT(1)
  FROM POOLDATA.T_CLAIM_OPENPROTECTION o
 WHERE o.POLICY_NO = :1
   AND (o.ID_CLAIM = :2 OR o.CLAIM_NO = :3)
   AND TRIM(o.PROTECTION_TYPE_ID) = '2'
   AND TRIM(o.APPROVAL_STATUS) = '1'
   AND (o.STATUS_ACTIVE IS NULL OR TRIM(o.STATUS_ACTIVE) = '1')

-- name: premi_agen_travel
--
-- BrowseClientNameTravel_SQL: CLIENTNAME agen leader sumber bisnis polis.
SELECT a.CLIENTNAME
  FROM POOLDATA.AGENT a
 WHERE a.ID = (SELECT l.LEADER
                 FROM POOLDATA.AGENT l
                WHERE l.ID = (SELECT g.SOURCEOFBUSINESS
                                FROM POOLDATA.T_GENERAL g
                               WHERE g.NOPOLIS = :1
                               FETCH FIRST 1 ROWS ONLY)
                FETCH FIRST 1 ROWS ONLY)
 FETCH FIRST 1 ROWS ONLY
