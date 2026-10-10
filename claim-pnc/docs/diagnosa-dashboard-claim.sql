-- ============================================================================
-- Diagnosa tile Dashboard Claim bernilai 0 dan /ringkasan menjawab 500
-- ============================================================================
--
-- Jalankan di TOAD dengan F5 (Execute as Script), BUKAN Ctrl+Enter.
-- Seluruhnya SELECT — tidak ada yang menulis.
--
-- Kelima bagian menjawab pertanyaan yang berbeda. Jalankan semuanya; yang
-- menarik adalah bagian mana yang angkanya jatuh ke nol.

-- ----------------------------------------------------------------------------
-- BAGIAN 1 — kueri hitung Outstanding, PERSIS seperti aplikasi, penyaring kosong
-- ----------------------------------------------------------------------------
-- Kalau bagian ini melempar ORA-xxxxx, itulah sebab galat 500-nya.
-- Kalau ia menjawab 0, sebabnya ada di salah satu syarat — lihat bagian 2.

SELECT COUNT(DISTINCT A.PZINSKEY) AS OUTSTANDING
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE A.PXOBJCLASS   = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.PYSTATUSWORK <> 'Resolved-Completed'
   AND A.PYSTATUSWORK <> 'Resolved-Rejected'
   AND A.BRANCHNAME   <> 'ASNET'
   AND A.PXFLOWNAME   NOT IN ('FixCorrespondence', 'Register_Flow_1')
   AND A.PXTASKLABEL  NOT IN ('FixCorrespondence');

-- ----------------------------------------------------------------------------
-- BAGIAN 2 — syarat mana yang membuang barisnya
-- ----------------------------------------------------------------------------
-- Dibaca dari atas ke bawah. Angka yang JATUH DRASTIS menunjuk syarat
-- penyebabnya.
--
-- Perhatikan tiga baris terakhir. Di Oracle, `NULL <> 'ASNET'` dan
-- `NULL NOT IN (...)` keduanya menghasilkan NULL — BUKAN benar — sehingga
-- baris berkolom kosong IKUT TERBUANG tanpa galat apa pun.

SELECT COUNT(*)                                                   AS semua_baris,
       COUNT(CASE WHEN A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
                  THEN 1 END)                                     AS kelas_cocok,
       COUNT(CASE WHEN A.PYSTATUSWORK NOT IN ('Resolved-Completed',
                                              'Resolved-Rejected')
                  THEN 1 END)                                     AS masih_berjalan,
       COUNT(A.BUSINESSCODE_1)                                    AS ada_businesscode,
       COUNT(A.BRANCHNAME)                                        AS ada_branchname,
       COUNT(A.PXFLOWNAME)                                        AS ada_pxflowname,
       COUNT(A.PXTASKLABEL)                                       AS ada_pxtasklabel,
       COUNT(A.PXASSIGNEDOPERATORID)                              AS ada_operator
  FROM POOLDATA.T_CLAIMLIST_ADMIN A;

-- ----------------------------------------------------------------------------
-- BAGIAN 3 — apakah INNER JOIN ke master BUSINESS yang membuangnya
-- ----------------------------------------------------------------------------
-- Kueri memakai INNER JOIN. Klaim ber-BUSINESSCODE_1 yang tidak ada padanannya
-- di POOLDATA.BUSINESS akan hilang tanpa galat.

SELECT COUNT(*) AS klaim_tanpa_padanan_business
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND NOT EXISTS (SELECT 1 FROM POOLDATA.BUSINESS c
                    WHERE c.ID = A.BUSINESSCODE_1);

-- ----------------------------------------------------------------------------
-- BAGIAN 4 — kedua tile survei
-- ----------------------------------------------------------------------------
-- 674 baris punya SURVEYORTYPE_1 setelah backfill. Kalau angka di bawah nol,
-- yang membuangnya syarat lain, bukan ketiadaan data.

SELECT A.SURVEYORTYPE_1,
       COUNT(*)                        AS jumlah,
       COUNT(A.PXFLOWNAME)             AS ada_pxflowname,
       COUNT(A.BRANCHNAME)             AS ada_branchname
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
 WHERE A.SURVEYORTYPE_1 IS NOT NULL
 GROUP BY A.SURVEYORTYPE_1
 ORDER BY 1;

-- ----------------------------------------------------------------------------
-- BAGIAN 5 — tab Inbox Tampungan PIC
-- ----------------------------------------------------------------------------
-- Tab ini menyaring PXASSIGNEDOPERATORID = 'ServicePNC'. Bila kolom itu kosong
-- di seluruh baris, tabnya pasti kosong — dan itu bukan cacat kode.

SELECT COUNT(*)                                                       AS semua,
       COUNT(A.PXASSIGNEDOPERATORID)                                  AS ada_operator,
       COUNT(CASE WHEN A.PXASSIGNEDOPERATORID = 'ServicePNC' THEN 1 END) AS servicepnc,
       COUNT(CASE WHEN A.USERTEKNIS_1 IS NULL THEN 1 END)              AS tanpa_pic
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC';
