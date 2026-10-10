-- ============================================================================
-- Mengisi POOLDATA.T_CLAIMLIST_ADMIN dari dua tabel Pega
-- ============================================================================
--
--     sumber   DATAPEGA.PC_ASM_FW_GCNMFW_WORK    7.703 baris   tabel kerja
--              DATAPEGA.PC_ASSIGN_WORKLIST     259.011 baris   antrean tugas
--     tujuan   POOLDATA.T_CLAIMLIST_ADMIN        1.014 baris   (13%)
--
-- ############################################################################
-- #  JANGAN JALANKAN LANGKAH 3 SEBELUM LANGKAH 1 DAN 2 DIBACA HASILNYA.      #
-- #  SKRIP INI MENULIS DATA PRODUKSI.                                        #
-- ############################################################################
--
-- Saya TIDAK menuliskan daftar kolomnya dari ingatan. Daftar kolom tabel kerja Pega tidak
-- pernah ada di repo ini — yang tercatat hanya "186 kolom, 123 terisi". Menebak nama kolom
-- pada pernyataan yang menulis 7.703 baris adalah cara paling mudah merusak data diam-diam.
--
-- Karena itu LANGKAH 1 **menghasilkan** daftar kolomnya dari katalog Oracle, dan LANGKAH 3
-- memakai daftar itu. Tidak ada satu nama kolom pun yang saya karang.

-- ============================================================================
-- LANGKAH 1 — periksa kedua tabel, dan apa yang benar-benar cocok
-- ============================================================================
--
-- 1a. Kolom yang ADA DI KEDUANYA dengan nama sama. Inilah yang akan disalin apa adanya.

SELECT t.column_name,
       t.data_type || '(' || NVL(TO_CHAR(t.data_length), '-') || ')' AS tipe_tujuan,
       s.data_type || '(' || NVL(TO_CHAR(s.data_length), '-') || ')' AS tipe_sumber,
       CASE WHEN t.data_type <> s.data_type THEN '<-- TIPE BERBEDA' END AS catatan
  FROM all_tab_columns t
       JOIN all_tab_columns s
         ON s.owner = 'DATAPEGA'
        AND s.table_name = 'PC_ASM_FW_GCNMFW_WORK'
        AND s.column_name = t.column_name
 WHERE t.owner = 'POOLDATA'
   AND t.table_name = 'T_CLAIMLIST_ADMIN'
 ORDER BY t.column_name;

-- 1b. Kolom tujuan yang TIDAK punya pasangan senama di tabel kerja.
--     Sebagian datang dari worklist, sebagian dihitung, sebagian memang tidak diisi.

SELECT t.column_name
  FROM all_tab_columns t
 WHERE t.owner = 'POOLDATA'
   AND t.table_name = 'T_CLAIMLIST_ADMIN'
   AND NOT EXISTS (SELECT 1
                     FROM all_tab_columns s
                    WHERE s.owner = 'DATAPEGA'
                      AND s.table_name = 'PC_ASM_FW_GCNMFW_WORK'
                      AND s.column_name = t.column_name)
 ORDER BY t.column_name;

-- 1c. Berapa baris yang akan ditambahkan, dan berapa yang sudah ada.

SELECT (SELECT COUNT(*) FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK
         WHERE PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC')              AS klaim_di_pega,
       (SELECT COUNT(*) FROM POOLDATA.T_CLAIMLIST_ADMIN)           AS baris_sekarang,
       (SELECT COUNT(*) FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
         WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
           AND NOT EXISTS (SELECT 1 FROM POOLDATA.T_CLAIMLIST_ADMIN t
                            WHERE t.PZINSKEY = w.PZINSKEY))        AS akan_ditambah
  FROM DUAL;

-- ============================================================================
-- LANGKAH 2 — periksa dua hal yang menentukan BENTUK pernyataannya
-- ============================================================================
--
-- 2a. Satu klaim bisa punya BANYAK baris worklist. Tabel tujuan satu baris per klaim, jadi
--     harus dipilih SATU. Periksa dulu seberapa sering itu terjadi.

SELECT COUNT(*) AS klaim_punya_assignment,
       SUM(CASE WHEN jml > 1 THEN 1 ELSE 0 END) AS klaim_punya_lebih_dari_satu,
       MAX(jml) AS terbanyak
  FROM (SELECT PXREFOBJECTKEY, COUNT(*) AS jml
          FROM DATAPEGA.PC_ASSIGN_WORKLIST
         GROUP BY PXREFOBJECTKEY);

-- 2b. Berapa klaim yang TIDAK punya baris worklist sama sekali.
--
--     Angka ini penting: kueri lama memakai INNER JOIN, sehingga klaim tanpa assignment
--     LENYAP dari layar. Skrip ini memakai LEFT JOIN — klaim tanpa assignment TETAP masuk,
--     dengan ketiga kolom assignment bernilai NULL. Itu perbaikan yang disengaja, dan
--     angkanya adalah selisih yang akan muncul dibanding layar Pega.

SELECT COUNT(*) AS klaim_tanpa_assignment
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND NOT EXISTS (SELECT 1 FROM DATAPEGA.PC_ASSIGN_WORKLIST a
                    WHERE a.PXREFOBJECTKEY = w.PZINSKEY);

-- ============================================================================
-- LANGKAH 3 — MERGE
-- ============================================================================
--
-- Daftar kolom di bawah adalah YANG DITULIS APLIKASI SENDIRI saat klaim baru didaftarkan
-- (`internal/registrasi/repo/sqlstore/inboxentry.sql`), ditambah empat kolom yang dibutuhkan
-- layar Dashboard Claim dan hari ini kosong di seluruh baris.
--
-- KOREKSI 2026-10-08: `PRODKE`, `STARTDATE`, dan `ENDDATE` DIBUANG — Work Owner memastikan kolom itu tidak ada di
-- tabel kerja Pega. Ia milik tabel polis (dipakai `inboxacceptopenprotection` sebagai
-- `g.PRODKE`), dan modul Dashboard Claim tidak memakainya sama sekali.
--
-- COCOKKAN DULU dengan hasil LANGKAH 1a. Kolom yang tidak muncul di sana TIDAK ADA di tabel
-- kerja Pega — buang barisnya dari MERGE ini, jangan biarkan ia gagal dengan ORA-00904.
--
-- ----------------------------------------------------------------------------
-- Tiga keputusan yang melekat pada pernyataan ini
-- ----------------------------------------------------------------------------
--
-- 1. MERGE, bukan INSERT. Aman diulang, dan ia juga MEMPERBAIKI 1.014 baris yang sudah ada —
--    yang justru paling bermasalah, karena SURVEYORTYPE_1 dan STATUSCLAIM_1-nya kosong.
--
-- 2. `USERTEKNIS_1` TIDAK ditimpa bila tujuan sudah berisi. Kolom itu satu-satunya yang
--    DITULIS aplikasi Go (tombol Transfer). Menimpanya dari Pega akan mengembalikan PIC
--    Teknik yang baru saja dipindahkan petugas — tanpa galat, dan tanpa ada yang menyadarinya.
--    Baris lain ditimpa karena Pega masih pemiliknya (`P-1`).
--
-- 3. Baris worklist dipilih dengan ROW_NUMBER() yang urutannya DITETAPKAN. Tanpa urutan yang
--    pasti, dua kali menjalankan skrip ini dapat memilih baris yang berbeda untuk klaim yang
--    sama — dan selisihnya tidak akan terlihat sebagai kesalahan.

MERGE INTO POOLDATA.T_CLAIMLIST_ADMIN t
USING (
  SELECT w.PZINSKEY,
         w.PYID,
         w.PXOBJCLASS,
         w.PYSTATUSWORK,
         w.PNCCASEID,
         w.BUSINESSCODE_1,
         w.KODECABANG_1,
         w.BRANCHNAME,
         w.PXCREATEOPERATOR,
         w.PXCREATEOPNAME,
         w.PXCREATEDATETIME,
         -- VARCHAR2 'YYYYMMDD' di sumber, DATE di tujuan. Sebagian baris KOSONG.
         TO_DATE(TRIM(w.REGISTERDATE_1) DEFAULT NULL ON CONVERSION ERROR,
                 'YYYYMMDD')                  AS REGISTERDATE_1,
         w.USERTEKNIS_1,
         w.GROUPPANEL_1,
         w.SOBNAME,
         w.BUSINESSNAME,
         w.QQNAME,
         w.POLICYNO,
         w.DATEOFLOSS_1,
         -- VARCHAR2 '20200105T170000.000 GMT' di sumber, DATE di tujuan.
         -- Lima belas aksara pertama saja; sisanya milidetik dan penanda zona.
         TO_DATE(SUBSTR(TRIM(w.REPORTDATE_1), 1, 15) DEFAULT NULL ON CONVERSION ERROR,
                 'YYYYMMDD"T"HH24MISS')        AS REPORTDATE_1,
         w.STATUSCLAIM_1,
         -- empat kolom yang hari ini kosong di SELURUH 1.014 baris tujuan
         w.SURVEYORTYPE_1,
         w.ADJUSTERPIC_1,
         w.ADJUSTERSTATUS_1,
         w.SURVEYORNAME_1,
         -- kelompok bisnis diturunkan, bukan disalin
         (SELECT b.BUSINESSGROUPID FROM POOLDATA.BUSINESS b
           WHERE b.ID = w.BUSINESSCODE_1)      AS BUSINESSGROUPID,
         -- ketiganya dari worklist, lewat baris terpilih
         a.PXASSIGNEDOPERATORID,
         a.PXFLOWNAME,
         a.PXTASKLABEL
    FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
         LEFT JOIN (SELECT PXREFOBJECTKEY,
                           PXASSIGNEDOPERATORID,
                           PXFLOWNAME,
                           PXTASKLABEL,
                           ROW_NUMBER() OVER (PARTITION BY PXREFOBJECTKEY
                                                  ORDER BY PXCREATEDATETIME DESC,
                                                           PXINSNAME DESC) AS rn
                      FROM DATAPEGA.PC_ASSIGN_WORKLIST) a
                ON a.PXREFOBJECTKEY = w.PZINSKEY
               AND a.rn = 1
   WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
) s
ON (t.PZINSKEY = s.PZINSKEY)
WHEN MATCHED THEN UPDATE SET
       t.PYID                 = s.PYID,
       t.PYSTATUSWORK         = s.PYSTATUSWORK,
       t.PNCCASEID            = s.PNCCASEID,
       t.BUSINESSCODE_1       = s.BUSINESSCODE_1,
       t.BUSINESSGROUPID      = s.BUSINESSGROUPID,
       t.KODECABANG_1         = s.KODECABANG_1,
       t.BRANCHNAME           = s.BRANCHNAME,
       t.PXCREATEOPERATOR     = s.PXCREATEOPERATOR,
       t.PXCREATEOPNAME       = s.PXCREATEOPNAME,
       t.PXCREATEDATETIME     = s.PXCREATEDATETIME,
       t.REGISTERDATE_1       = s.REGISTERDATE_1,
       -- TIDAK ditimpa bila sudah berisi — lihat keputusan 2 di atas
       t.USERTEKNIS_1         = NVL(t.USERTEKNIS_1, s.USERTEKNIS_1),
       t.GROUPPANEL_1         = s.GROUPPANEL_1,
       t.SOBNAME              = s.SOBNAME,
       t.BUSINESSNAME         = s.BUSINESSNAME,
       t.QQNAME               = s.QQNAME,
       t.POLICYNO             = s.POLICYNO,
       t.DATEOFLOSS_1         = s.DATEOFLOSS_1,
       t.REPORTDATE_1         = s.REPORTDATE_1,
       t.STATUSCLAIM_1        = s.STATUSCLAIM_1,
       t.SURVEYORTYPE_1       = s.SURVEYORTYPE_1,
       t.ADJUSTERPIC_1        = s.ADJUSTERPIC_1,
       t.ADJUSTERSTATUS_1     = s.ADJUSTERSTATUS_1,
       t.SURVEYORNAME_1       = s.SURVEYORNAME_1,
       t.PXASSIGNEDOPERATORID = s.PXASSIGNEDOPERATORID,
       t.PXFLOWNAME           = s.PXFLOWNAME,
       t.PXTASKLABEL          = s.PXTASKLABEL
WHEN NOT MATCHED THEN INSERT (
       t.PZINSKEY, t.PYID, t.PXOBJCLASS, t.PYSTATUSWORK, t.PNCCASEID,
       t.BUSINESSCODE_1, t.BUSINESSGROUPID, t.KODECABANG_1, t.BRANCHNAME,
       t.PXCREATEOPERATOR, t.PXCREATEOPNAME, t.PXCREATEDATETIME, t.REGISTERDATE_1,
       t.USERTEKNIS_1, t.GROUPPANEL_1, t.SOBNAME, t.BUSINESSNAME, t.QQNAME,
       t.POLICYNO, t.DATEOFLOSS_1, t.REPORTDATE_1,
       t.STATUSCLAIM_1, t.SURVEYORTYPE_1, t.ADJUSTERPIC_1,
       t.ADJUSTERSTATUS_1, t.SURVEYORNAME_1,
       t.PXASSIGNEDOPERATORID, t.PXFLOWNAME, t.PXTASKLABEL, t.STS_AKTIF)
VALUES (
       s.PZINSKEY, s.PYID, s.PXOBJCLASS, s.PYSTATUSWORK, s.PNCCASEID,
       s.BUSINESSCODE_1, s.BUSINESSGROUPID, s.KODECABANG_1, s.BRANCHNAME,
       s.PXCREATEOPERATOR, s.PXCREATEOPNAME, s.PXCREATEDATETIME, s.REGISTERDATE_1,
       s.USERTEKNIS_1, s.GROUPPANEL_1, s.SOBNAME, s.BUSINESSNAME, s.QQNAME,
       s.POLICYNO, s.DATEOFLOSS_1, s.REPORTDATE_1,
       s.STATUSCLAIM_1, s.SURVEYORTYPE_1, s.ADJUSTERPIC_1,
       s.ADJUSTERSTATUS_1, s.SURVEYORNAME_1,
       s.PXASSIGNEDOPERATORID, s.PXFLOWNAME, s.PXTASKLABEL, '1');

-- JANGAN COMMIT DULU. Periksa hasilnya lebih dulu — lihat LANGKAH 4.
-- Bila ada yang janggal: ROLLBACK;

-- ============================================================================
-- LANGKAH 4 — periksa sebelum COMMIT
-- ============================================================================

SELECT COUNT(*)                                            AS total_baris,
       COUNT(USERTEKNIS_1)                                 AS ada_pic_teknik,
       COUNT(SURVEYORTYPE_1)                               AS ada_tipe_surveyor,
       COUNT(STATUSCLAIM_1)                                AS ada_status_klaim,
       SUM(CASE WHEN SURVEYORTYPE_1 = '1' THEN 1 ELSE 0 END) AS surveyor_internal,
       SUM(CASE WHEN SURVEYORTYPE_1 = '2' THEN 1 ELSE 0 END) AS loss_adjuster
  FROM POOLDATA.T_CLAIMLIST_ADMIN;

-- Yang diharapkan: total_baris mendekati `klaim_di_pega` dari LANGKAH 1c, dan ketiga kolom
-- yang tadinya 0 kini berisi. Bila `ada_tipe_surveyor` masih 0, kolomnya memang kosong juga
-- di tabel SUMBER — dan kedua tile survei akan tetap nol. Itu temuan, bukan kegagalan skrip.

-- COMMIT;

-- ============================================================================
-- SESUDAH COMMIT — dua hal
-- ============================================================================
--
-- 1. Statistik tabel ini menyesatkan (ALL_TABLES mencatat NUM_ROWS = 1 sementara isinya 1.014).
--    Kumpulkan ulang, atau rencana eksekusi kueri layar akan dipilih dari angka yang salah:
--
--        BEGIN DBMS_STATS.GATHER_TABLE_STATS('POOLDATA', 'T_CLAIMLIST_ADMIN'); END;
--
-- 2. SIAPA PENULIS TABEL INI SETELAH HARI INI — belum ditetapkan, dan ini bukan detail.
--
--    Pega masih membuat klaim baru (`P-3`), aplikasi Go menulis `USERTEKNIS_1`, dan menurut
--    catatan migrasi `0005` ada pula "proses pengisi" yang berjalan di lingkungan testing.
--    Tiga penulis atas satu tabel melanggar `P-1`, dan akibatnya bukan galat melainkan data
--    yang saling menimpa.
--
--    Skrip ini mengisi SEKALI. Ia tidak menjawab bagaimana tabelnya tetap terisi besok.
