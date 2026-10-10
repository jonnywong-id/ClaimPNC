-- ============================================================================
-- Mengisi BARIS KASUS SURVEI ke POOLDATA.T_CLAIMLIST_ADMIN
-- ============================================================================
--
--     sumber   DATAPEGA.PC_ASM_FW_GCNMFW_WORK   367 baris ber-PXOBJCLASS
--                                               'ASM-FW-GCNMFW-Work-SurveyClaim'
--     tujuan   POOLDATA.T_CLAIMLIST_ADMIN       0 baris kasus survei (2026-10-08)
--
-- PRASYARAT: migrasi `0015_claimlist_admin_kasus_survei.up.sql` sudah dijalankan.
-- Tanpa itu LANGKAH 3 gagal ORA-00904 pada CASEID_1.
--
-- Jalankan di TOAD dengan F5 (Execute as Script), BUKAN Ctrl+Enter — Ctrl+Enter mengirim
-- titik koma beserta pernyataannya dan Oracle menolaknya dengan ORA-00933.
--
-- KENAPA INI DIBUTUHKAN. Tile Loss Adjuster dan Internal Surveyor menghitung KASUS SURVEI,
-- yang di Pega adalah baris ANAK pada tabel kerja, ditaut `CASEID_1 -> PZINSKEY` klaimnya
-- (`RDB List/BrowseInternalSurveyor-SQL.xml`). Backfill 2026-10-08 hanya membawa baris
-- ber-PXOBJCLASS 'ASM-FW-GCNMFW-Work-PNC', sehingga kasus surveinya tidak ikut sama sekali.

-- ----------------------------------------------------------------------------
-- LANGKAH 1 — pastikan ketiga kolom baru sudah ada
-- ----------------------------------------------------------------------------
-- Harus mengembalikan TIGA baris. Kalau kurang, migrasi 0015 belum jalan — berhenti di sini.

SELECT COLUMN_NAME, DATA_TYPE, DATA_LENGTH
  FROM ALL_TAB_COLUMNS
 WHERE OWNER = 'POOLDATA'
   AND TABLE_NAME = 'T_CLAIMLIST_ADMIN'
   AND COLUMN_NAME IN ('CASEID_1', 'RESCHEDULEDATE_1', 'RESCHEDULELOCATION_1')
 ORDER BY COLUMN_NAME;

-- ----------------------------------------------------------------------------
-- LANGKAH 2 — periksa lebar kolom SEBELUM menulis
-- ----------------------------------------------------------------------------
-- Hasil KOSONG = aman. Ada baris = kolom tujuan lebih sempit daripada isi sumbernya, dan
-- MERGE-nya akan gagal ORA-12899 di tengah jalan.
--
-- Langkah ini ada karena backfill sebelumnya gagal persis begitu, EMPAT KALI, masing-masing
-- menampilkan satu kolom saja. Memeriksa seluruhnya sekaligus menggantikan empat putaran
-- coba-jalankan dengan satu kueri.

SELECT t.COLUMN_NAME,
       t.DATA_LENGTH AS lebar_tujuan,
       s.DATA_LENGTH AS lebar_sumber
  FROM ALL_TAB_COLUMNS t
       JOIN ALL_TAB_COLUMNS s
         ON s.COLUMN_NAME = t.COLUMN_NAME
        AND s.OWNER = 'DATAPEGA'
        AND s.TABLE_NAME = 'PC_ASM_FW_GCNMFW_WORK'
 WHERE t.OWNER = 'POOLDATA'
   AND t.TABLE_NAME = 'T_CLAIMLIST_ADMIN'
   AND t.DATA_TYPE = 'VARCHAR2'
   AND s.DATA_TYPE = 'VARCHAR2'
   AND t.DATA_LENGTH < s.DATA_LENGTH
   AND t.COLUMN_NAME IN ('PZINSKEY','PYID','PXOBJCLASS','PYSTATUSWORK','POLICYNO','QQNAME',
                         'REFNO_1','SURVEYORNAME_1','SURVEYORTYPE_1','USERTEKNIS_1',
                         'ADJUSTERPIC_1','ADJUSTERSTATUS_1','CASEID_1','RESCHEDULELOCATION_1',
                         'BRANCHNAME','BUSINESSCODE_1','GROUPPANEL_1','PXFLOWNAME',
                         'PXTASKLABEL','PXCREATEOPNAME','PXASSIGNEDOPERATORID')
 ORDER BY 1;

-- ----------------------------------------------------------------------------
-- LANGKAH 3 — isi barisnya
-- ----------------------------------------------------------------------------
-- MERGE, bukan INSERT: aman diulang, dan ia memperbaiki baris yang sudah terlanjur masuk
-- alih-alih menggandakannya.
--
-- Hanya kolom yang BENAR-BENAR DIBACA modul ini yang dibawa. Membawa seluruh 186 kolom
-- tabel sumber akan menyalin banyak kolom yang tidak ada padanannya di tabel datar, dan
-- setiap satunya satu kemungkinan gagal tanpa menambah satu pun angka di layar.
--
-- PENUGASAN SENGAJA TIDAK DIGABUNG. Berbeda dari backfill klaim, di sini TIDAK ada LEFT JOIN
-- ke PC_ASSIGN_WORKLIST: baris kasus survei disaring oleh PYSTATUSWORK-nya sendiri, bukan oleh
-- PXFLOWNAME/PXTASKLABEL. Menyalin kolom penugasan ke sini hanya akan menambah NULL yang
-- tidak dibaca siapa pun.

MERGE INTO POOLDATA.T_CLAIMLIST_ADMIN t
USING (
  SELECT w.PZINSKEY,
         w.PYID,
         w.PXOBJCLASS,
         w.PYSTATUSWORK,
         w.CASEID_1,
         w.POLICYNO,
         w.QQNAME,
         w.REFNO_1,
         w.SURVEYORTYPE_1,
         w.SURVEYORNAME_1,
         w.USERTEKNIS_1,
         w.ADJUSTERPIC_1,
         w.ADJUSTERSTATUS_1,
         w.RESCHEDULEDATE_1,
         w.RESCHEDULELOCATION_1,
         w.PXCREATEOPERATOR,
         w.PXCREATEOPNAME,
         w.PXCREATEDATETIME
    FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
   WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim'
) s
ON (t.PZINSKEY = s.PZINSKEY)
WHEN MATCHED THEN UPDATE SET
       t.PYID                 = s.PYID,
       t.PYSTATUSWORK         = s.PYSTATUSWORK,
       t.CASEID_1             = s.CASEID_1,
       t.POLICYNO             = s.POLICYNO,
       t.QQNAME               = s.QQNAME,
       t.REFNO_1              = s.REFNO_1,
       t.SURVEYORTYPE_1       = s.SURVEYORTYPE_1,
       t.SURVEYORNAME_1       = s.SURVEYORNAME_1,
       t.USERTEKNIS_1         = s.USERTEKNIS_1,
       t.ADJUSTERPIC_1        = s.ADJUSTERPIC_1,
       t.ADJUSTERSTATUS_1     = s.ADJUSTERSTATUS_1,
       t.RESCHEDULEDATE_1     = s.RESCHEDULEDATE_1,
       t.RESCHEDULELOCATION_1 = s.RESCHEDULELOCATION_1,
       t.PXCREATEOPERATOR     = s.PXCREATEOPERATOR,
       t.PXCREATEOPNAME       = s.PXCREATEOPNAME,
       t.PXCREATEDATETIME     = s.PXCREATEDATETIME
WHEN NOT MATCHED THEN INSERT (
       t.PZINSKEY, t.PYID, t.PXOBJCLASS, t.PYSTATUSWORK, t.CASEID_1,
       t.POLICYNO, t.QQNAME, t.REFNO_1,
       t.SURVEYORTYPE_1, t.SURVEYORNAME_1, t.USERTEKNIS_1,
       t.ADJUSTERPIC_1, t.ADJUSTERSTATUS_1,
       t.RESCHEDULEDATE_1, t.RESCHEDULELOCATION_1,
       t.PXCREATEOPERATOR, t.PXCREATEOPNAME, t.PXCREATEDATETIME,
       t.STS_AKTIF)
VALUES (
       s.PZINSKEY, s.PYID, s.PXOBJCLASS, s.PYSTATUSWORK, s.CASEID_1,
       s.POLICYNO, s.QQNAME, s.REFNO_1,
       s.SURVEYORTYPE_1, s.SURVEYORNAME_1, s.USERTEKNIS_1,
       s.ADJUSTERPIC_1, s.ADJUSTERSTATUS_1,
       s.RESCHEDULEDATE_1, s.RESCHEDULELOCATION_1,
       s.PXCREATEOPERATOR, s.PXCREATEOPNAME, s.PXCREATEDATETIME,
       '1');

COMMIT;

-- ----------------------------------------------------------------------------
-- LANGKAH 4 — buktikan hasilnya
-- ----------------------------------------------------------------------------
-- Yang diharapkan: KASUS_SURVEI 367, dan TANPA_INDUK 0.
--
-- TANPA_INDUK > 0 berarti ada kasus survei yang klaim induknya tidak ikut terbawa backfill
-- sebelumnya. Baris itu TIDAK akan muncul di layar — kedua kueri menuntut induknya ada —
-- dan itu selisih yang harus dijelaskan, bukan diabaikan.

SELECT COUNT(*)                                                           AS kasus_survei,
       COUNT(CASE WHEN SURVEYORTYPE_1 = '2' THEN 1 END)                   AS loss_adjuster,
       COUNT(CASE WHEN SURVEYORTYPE_1 = '1' THEN 1 END)                   AS internal_surveyor,
       COUNT(CASE WHEN CASEID_1 IS NULL THEN 1 END)                       AS tanpa_penaut
  FROM POOLDATA.T_CLAIMLIST_ADMIN
 WHERE PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim';

SELECT COUNT(*) AS tanpa_induk
  FROM POOLDATA.T_CLAIMLIST_ADMIN s
 WHERE s.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim'
   AND NOT EXISTS (SELECT 1
                     FROM POOLDATA.T_CLAIMLIST_ADMIN p
                    WHERE p.PZINSKEY = s.CASEID_1
                      AND p.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC');

-- Statistik dikumpulkan ulang: tanpa ini Oracle masih memilih rencana kueri untuk tabel
-- berukuran lama, dan sub-kueri kasus survei dijalankan untuk setiap baris klaim.
BEGIN
  DBMS_STATS.GATHER_TABLE_STATS('POOLDATA', 'T_CLAIMLIST_ADMIN');
END;
/
