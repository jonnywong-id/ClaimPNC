-- Kueri ekspor tab KPI PIC Teknik — satu per pilihan "Pilih Data KPI".
--
-- # Apa yang diekspor, dan kenapa ia BUKAN kartu skor
--
-- Sampai 2026-10-08 tombol ekspor tab ini menuliskan kartu skor — nama PIC dan nilainya.
-- Itu BUKAN yang dilakukan Pega. Pega mengekspor **data klaim mentah**, satu baris per
-- klaim, dan pilihan "Pilih Data KPI" menentukan kumpulan yang mana:
--
--	1  Export KPI Progress    GetReportprogressKlaimKPIPIC       42 kolom
--	2  Export KPI SLA         GetReportTATSLAKlaimKPIPIC         38 kolom
--	3  Export KPI Akseptasi   GetReportTATAkseptasiKlaimKPIPIC   51 kolom
--	4  Export KPI Analisis    GetReportTATAnalisisKlaimKPIPIC    51 kolom
--
-- Nilai dan labelnya dibaca dari `Activity/EksportDataKPIProgressKlaim-Act.xml`, yang
-- menyusun halaman `TempPilihan` berpasangan `FlagASO` → `NoKTP`.
--
-- # Nama kolomnya SENGAJA dibiarkan menyesatkan
--
-- `POLICYDECLARATIONNO` berisi CLAIMID. `EDMNO` berisi nomor polis. `NOPOLIS` berisi nama
-- bisnis. `pyIssue` berisi posisi progres. Seluruhnya alias warisan Pega — developer lama
-- memaksa nama kolom agar cocok dengan property clipboard yang sudah ada
-- (`03-CURRENT-ARCHITECTURE.md` §4.2).
--
-- Alias itu TETAP dipakai di sini, dan itu keputusan sadar: ia menjadi **judul kolom pada
-- berkas CSV**. Menggantinya dengan nama yang benar akan membuat berkas kami tidak lagi
-- dapat ditumpuk dengan berkas Pega pada periode paralel — padahal justru itu yang dipakai
-- membandingkan keduanya (`P-5`). Penamaan ulang `D-19` berlaku untuk nama di dalam kode,
-- bukan untuk judul kolom berkas yang dibandingkan dengan sistem lama.
--
-- # Tiga penyesuaian terhadap kueri Pega, dan hanya tiga
--
--	TO_CHAR(tanggal,'dd/mm/yyyy')  dihapus; tanggal dikembalikan sebagai TANGGAL dan
--	                               diformat di Go (`D-20`). Keluarannya sama persis.
--	TRUNC(x) >= .. AND <= ..       menjadi rentang setengah terbuka `>= :1 AND < :2 + 1
--	                               hari`. Mencakup hari yang sama, tetapi membiarkan index
--	                               terpakai — sama seperti kueri PIC Teknik lainnya.
--	{ASIS:TempDateReport.UserTeknis}  menjadi penanda /*PIC_LIST*/ yang diisi penanda bind
--	                                  sejumlah PIC. Yang disisipkan PENANDA, bukan nilai.
--
-- Selebihnya dibawa apa adanya, termasuk keanehannya: `+ INTERVAL '12' HOUR` pada
-- `os_akseptasi_klaim`, `SUBSTR(...,0,8)` yang pada Oracle berlaku seperti `1`, dan
-- penyaring `statuswork != 'Resolved-Rejected'` yang pada kueri SLA sudah tidak mungkin
-- salah karena `statuswork` di sana sudah dipatok `Resolved-Completed`.
--
-- # Penyaring yang SAMA di keempatnya
--
--	branchcode != '100639'
--	grouppanel IN ('003','004','006','009')      ← perhatikan 009; berbeda dari tab layar
--	businesscode NOT IN (sepuluh kode)
--
-- Perhatikan `009`: penyaring lini Non-MBU pada layar hanya memakai 003, 004, dan 006.
-- Kueri ekspor memakai empat. Perbedaan itu ADA di Pega dan dibawa apa adanya.

-- name: ekspor_pic_sla
-- Meniru `GetReportTATSLAKlaimKPIPIC` — klaim yang sudah selesai pada rentang registrasi.
SELECT
    a.CLAIMID                                         AS "POLICYDECLARATIONNO",
    a.CLAIMNO                                         AS "IDPEGA",
    a.NOPOLIS                                         AS "EDMNO",
    REPLACE(a.QQNAME, ',', ' ')                       AS "QQNAME",
    a.BUSINESSNAME                                    AS "NOPOLIS",
    a.LEADER_MEMBER                                   AS "BUSINESSTYPE",
    a.DATEOFLOSS                                      AS "ENDDATE",
    a.DATEOFLOSS                                      AS "EDMDATE",
    (SELECT x.RECEIVEDDATE_1
       FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK x
      WHERE x.PZINSKEY = a.CLAIMID)                   AS "PRODKE",
    a.FINISHREGISTERDATE                              AS "ACCUMCODE",
    a.CLOSECLAIMDATE                                  AS "TABLEOBJECTNAME",
    a.REGISTERDATE                                    AS "REGISTERID",
    a.TRANSFERPIC_DATE                                AS "WARRANTYNO",
    (SELECT MIN(TANGGAL + INTERVAL '12' HOUR)
       FROM OS_AKSEPTASI_KLAIM
      WHERE CASEID = a.CLAIMID)                       AS "pyCountry",
    a.DATEOFREQUESTDOCUMENT                           AS "THEINSURED",
    a.SURVEYDATE                                      AS "BUSINESSCODE",
    a.PICTEKNIK                                       AS "OLDPOLICYNO",
    REPLACE(a.CLOSECLAIMNOTE, CHR(10), ' ')           AS "GROUPPANEL",
    a.INVESTIGATOR_TF_DATE                            AS "SOBLEADER1",
    a.ANALYSTTORCLPUCL_DATE                           AS "SOBLEADER2",
    a.RCLPUCL_TF_TOANALYST                            AS "SOBLEADER0",
    a.RCLPUCL                                         AS "SOBNAME",
    a.COMPLIANCE_CREATEDATE                           AS "SOURCEOFBUSINESS",
    a.CPLVALID_DATE                                   AS "ISMAINFOLLOWING",
    a.POSTAUDIT_TF_ANALYSTDATE                        AS "STATUSBUSINESS",
    a.CPLPOSTAUDIT_VALIDDATE                          AS "EDMTYPE",
    a.ALASANKETERLAMBATAN                             AS "MARKETINGCODE",
    a.BUSINESSCODE                                    AS "BUSINESSNAME",
    (SELECT b.LSC_NOTE
       FROM V_STS_CLAIM b
      WHERE a.STATUSCLAIM = b.LSC_ID)                 AS "CLIENTID",
    a.SOBNAME                                         AS "ISFOLLOWINGCHILD",
    a.CLAIMNO                                         AS "pyNote",
    (SELECT x.PXCREATEOPERATOR
       FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK x
      WHERE x.PZINSKEY = a.CLAIMID)                   AS "pyLabel",
    a.TGLDOKLENGKAP                                   AS "pyGroup",
    a.TRF_TO_INVESTIGATOR                             AS "pzInsKey",
    (SELECT SUBSTR(x.REPORTDATE_1, 1, 8)
       FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK x
      WHERE x.PZINSKEY = a.CLAIMID)                   AS "IndexSpreading",
    (SELECT z.POSISI
       FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC z
      WHERE z.CASEID = a.CLAIMNO
        AND z.ID = (SELECT MAX(w.ID)
                      FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC w
                     WHERE w.CASEID = z.CASEID))      AS "pyIssue",
    (SELECT g.STARTDATE
       FROM T_GENERAL g
      WHERE g.NOPOLIS = a.NOPOLIS AND g.PRODKE = a.PRODKE)  AS "PlatNo",
    (SELECT g.ENDDATE
       FROM T_GENERAL g
      WHERE g.NOPOLIS = a.NOPOLIS AND g.PRODKE = a.PRODKE)  AS "Position"
FROM POOLDATA.T_CLAIM_PNC a
WHERE a.REGISTERDATE >= :1
  AND a.REGISTERDATE < :2 + INTERVAL '1' DAY
  AND a.STATUSWORK = 'Resolved-Completed'
  AND a.CLOSECLAIMDATE IS NOT NULL
  AND a.STATUSWORK <> 'Resolved-Rejected'
  AND a.BRANCHCODE <> '100639'
  AND a.GROUPPANEL IN ('003', '004', '006', '009')
  AND a.BUSINESSCODE NOT IN
      ('10145', '10168', '10165', '10164', '10053',
       '10075', '10126', '10011', '10077', '10007')
  AND a.PICTEKNIK IN /*PIC_LIST*/
ORDER BY a.CLAIMID ASC

-- name: ekspor_pic_akseptasi
-- Meniru `GetReportTATAkseptasiKlaimKPIPIC` — klaim yang sudah diakseptasi pada rentang
-- TANGGAL AKSEPTASI, bukan tanggal registrasi.
--
-- Kolomnya sama dengan `ekspor_pic_analisis`; yang berbeda hanya penyaringnya. Keduanya
-- ditulis utuh, tidak dibagi, karena berkas .sql tidak mengenal penyertaan — dan dua kueri
-- yang kolomnya kebetulan sama lebih mudah dibaca daripada satu kueri bercabang.
SELECT
    a.CLAIMID                                         AS "POLICYDECLARATIONNO",
    a.CLAIMNO                                         AS "IDPEGA",
    a.NOPOLIS                                         AS "EDMNO",
    REPLACE(a.QQNAME, ',', ' ')                       AS "QQNAME",
    a.BUSINESSNAME                                    AS "NOPOLIS",
    a.LEADER_MEMBER                                   AS "BUSINESSTYPE",
    a.DATEOFLOSS                                      AS "ENDDATE",
    a.DATEOFLOSS                                      AS "EDMDATE",
    (SELECT x.RECEIVEDDATE_1
       FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK x
      WHERE x.PZINSKEY = a.CLAIMID)                   AS "PRODKE",
    a.FINISHREGISTERDATE                              AS "ACCUMCODE",
    a.CLOSECLAIMDATE                                  AS "TABLEOBJECTNAME",
    a.REGISTERDATE                                    AS "REGISTERID",
    a.TRANSFERPIC_DATE                                AS "WARRANTYNO",
    (SELECT MIN(TANGGAL + INTERVAL '12' HOUR)
       FROM OS_AKSEPTASI_KLAIM
      WHERE CASEID = a.CLAIMID)                       AS "pyCountry",
    a.DATEOFREQUESTDOCUMENT                           AS "THEINSURED",
    a.SURVEYDATE                                      AS "BUSINESSCODE",
    a.PICTEKNIK                                       AS "OLDPOLICYNO",
    REPLACE(a.CLOSECLAIMNOTE, CHR(10), ' ')           AS "GROUPPANEL",
    a.INVESTIGATOR_TF_DATE                            AS "SOBLEADER1",
    a.ANALYSTTORCLPUCL_DATE                           AS "SOBLEADER2",
    a.RCLPUCL_TF_TOANALYST                            AS "SOBLEADER0",
    a.RCLPUCL                                         AS "SOBNAME",
    a.COMPLIANCE_CREATEDATE                           AS "SOURCEOFBUSINESS",
    a.CPLVALID_DATE                                   AS "ISMAINFOLLOWING",
    a.POSTAUDIT_TF_ANALYSTDATE                        AS "STATUSBUSINESS",
    a.CPLPOSTAUDIT_VALIDDATE                          AS "EDMTYPE",
    a.ALASANKETERLAMBATAN                             AS "MARKETINGCODE",
    a.BUSINESSCODE                                    AS "BUSINESSNAME",
    (SELECT b.LSC_NOTE
       FROM V_STS_CLAIM b
      WHERE a.STATUSCLAIM = b.LSC_ID)                 AS "CLIENTID",
    d.ACCEPTANCE_DATECOMITEE                          AS "SYARIAHSTATUS",
    d.TGLBAYAR                                        AS "BRANCHCODE",
    d.STATUS                                          AS "BRANCHNAME",
    d.PAYMENTTYPE                                     AS "MARKETINGNAME",
    d.NOAKSEPTASI                                     AS "STARTDATE",
    d.PRINTLOD_DATE                                   AS "FLAGEDMBATAL",
    d.TGLAKSEPTASI                                    AS "FOLLOWEDPOLICY",
    d.MANUAL_ACCEPTANCEDATE                           AS "POOLINGSTATUS",
    d.MANUALACCEPTANCEDATECOMITEE                     AS "AUTOCANCELPRINTSTATUS",
    TO_NUMBER(d.GROSSVALUE)                           AS "ASMAnalisator",
    a.SOBNAME                                         AS "ISFOLLOWINGCHILD",
    a.CLAIMNO                                         AS "pyNote",
    (SELECT x.PXCREATEOPERATOR
       FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK x
      WHERE x.PZINSKEY = a.CLAIMID)                   AS "pyLabel",
    a.TGLDOKLENGKAP                                   AS "pyGroup",
    d.ANALYST_TFKOMITEDATE                            AS "pyCountryName",
    a.TRF_TO_INVESTIGATOR                             AS "pzInsKey",
    d.RECEIVEDATELOD                                  AS "pyID",
    (SELECT SUBSTR(x.REPORTDATE_1, 1, 8)
       FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK x
      WHERE x.PZINSKEY = a.CLAIMID)                   AS "IndexSpreading",
    d.TRANSFER_CASHIER_DATE                           AS "pyEmailAddress",
    (SELECT z.POSISI
       FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC z
      WHERE z.CASEID = a.CLAIMNO
        AND z.ID = (SELECT MAX(w.ID)
                      FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC w
                     WHERE w.CASEID = z.CASEID))      AS "pyIssue",
    (SELECT g.STARTDATE
       FROM T_GENERAL g
      WHERE g.NOPOLIS = a.NOPOLIS AND g.PRODKE = a.PRODKE)  AS "PlatNo",
    (SELECT g.ENDDATE
       FROM T_GENERAL g
      WHERE g.NOPOLIS = a.NOPOLIS AND g.PRODKE = a.PRODKE)  AS "Position"
FROM POOLDATA.T_CLAIM_PNC a
JOIN POOLDATA.T_CLAIM_ADJUSTMENT d ON d.CLAIMID = a.CLAIMID
WHERE d.TGLAKSEPTASI >= :1
  AND d.TGLAKSEPTASI < :2 + INTERVAL '1' DAY
  AND d.NOAKSEPTASI IS NOT NULL
  AND a.STATUSWORK <> 'Resolved-Rejected'
  AND a.BRANCHCODE <> '100639'
  AND a.GROUPPANEL IN ('003', '004', '006', '009')
  AND a.BUSINESSCODE NOT IN
      ('10145', '10168', '10165', '10164', '10053',
       '10075', '10126', '10011', '10077', '10007')
  AND a.PICTEKNIK IN /*PIC_LIST*/
ORDER BY a.CLAIMID ASC

-- name: ekspor_pic_analisis
-- Meniru `GetReportTATAnalisisKlaimKPIPIC` — klaim yang sudah ditransfer ke komite oleh
-- analis, pada rentang TANGGAL REGISTRASI.
--
-- Hanya adjustment PERTAMA tiap klaim yang dihitung (`MIN(ADJUSTMENTID)`), sama seperti
-- `analysis_spans` pada kartu skor. Klaim dengan beberapa adjustment tetap satu baris.
SELECT
    a.CLAIMID                                         AS "POLICYDECLARATIONNO",
    a.CLAIMNO                                         AS "IDPEGA",
    a.NOPOLIS                                         AS "EDMNO",
    REPLACE(a.QQNAME, ',', ' ')                       AS "QQNAME",
    a.BUSINESSNAME                                    AS "NOPOLIS",
    a.LEADER_MEMBER                                   AS "BUSINESSTYPE",
    a.DATEOFLOSS                                      AS "ENDDATE",
    a.DATEOFLOSS                                      AS "EDMDATE",
    (SELECT x.RECEIVEDDATE_1
       FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK x
      WHERE x.PZINSKEY = a.CLAIMID)                   AS "PRODKE",
    a.FINISHREGISTERDATE                              AS "ACCUMCODE",
    a.CLOSECLAIMDATE                                  AS "TABLEOBJECTNAME",
    a.REGISTERDATE                                    AS "REGISTERID",
    a.TRANSFERPIC_DATE                                AS "WARRANTYNO",
    (SELECT MIN(TANGGAL + INTERVAL '12' HOUR)
       FROM OS_AKSEPTASI_KLAIM
      WHERE CASEID = a.CLAIMID)                       AS "pyCountry",
    a.DATEOFREQUESTDOCUMENT                           AS "THEINSURED",
    a.SURVEYDATE                                      AS "BUSINESSCODE",
    a.PICTEKNIK                                       AS "OLDPOLICYNO",
    REPLACE(a.CLOSECLAIMNOTE, CHR(10), ' ')           AS "GROUPPANEL",
    a.INVESTIGATOR_TF_DATE                            AS "SOBLEADER1",
    a.ANALYSTTORCLPUCL_DATE                           AS "SOBLEADER2",
    a.RCLPUCL_TF_TOANALYST                            AS "SOBLEADER0",
    a.RCLPUCL                                         AS "SOBNAME",
    a.COMPLIANCE_CREATEDATE                           AS "SOURCEOFBUSINESS",
    a.CPLVALID_DATE                                   AS "ISMAINFOLLOWING",
    a.POSTAUDIT_TF_ANALYSTDATE                        AS "STATUSBUSINESS",
    a.CPLPOSTAUDIT_VALIDDATE                          AS "EDMTYPE",
    a.ALASANKETERLAMBATAN                             AS "MARKETINGCODE",
    a.BUSINESSCODE                                    AS "BUSINESSNAME",
    (SELECT b.LSC_NOTE
       FROM V_STS_CLAIM b
      WHERE a.STATUSCLAIM = b.LSC_ID)                 AS "CLIENTID",
    d.ACCEPTANCE_DATECOMITEE                          AS "SYARIAHSTATUS",
    d.TGLBAYAR                                        AS "BRANCHCODE",
    d.STATUS                                          AS "BRANCHNAME",
    d.PAYMENTTYPE                                     AS "MARKETINGNAME",
    d.NOAKSEPTASI                                     AS "STARTDATE",
    d.PRINTLOD_DATE                                   AS "FLAGEDMBATAL",
    d.TGLAKSEPTASI                                    AS "FOLLOWEDPOLICY",
    d.MANUAL_ACCEPTANCEDATE                           AS "POOLINGSTATUS",
    d.MANUALACCEPTANCEDATECOMITEE                     AS "AUTOCANCELPRINTSTATUS",
    TO_NUMBER(d.GROSSVALUE)                           AS "ASMAnalisator",
    a.SOBNAME                                         AS "ISFOLLOWINGCHILD",
    a.CLAIMNO                                         AS "pyNote",
    (SELECT x.PXCREATEOPERATOR
       FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK x
      WHERE x.PZINSKEY = a.CLAIMID)                   AS "pyLabel",
    a.TGLDOKLENGKAP                                   AS "pyGroup",
    d.ANALYST_TFKOMITEDATE                            AS "pyCountryName",
    a.TRF_TO_INVESTIGATOR                             AS "pzInsKey",
    d.RECEIVEDATELOD                                  AS "pyID",
    (SELECT SUBSTR(x.REPORTDATE_1, 1, 8)
       FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK x
      WHERE x.PZINSKEY = a.CLAIMID)                   AS "IndexSpreading",
    d.TRANSFER_CASHIER_DATE                           AS "pyEmailAddress",
    (SELECT z.POSISI
       FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC z
      WHERE z.CASEID = a.CLAIMNO
        AND z.ID = (SELECT MAX(w.ID)
                      FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC w
                     WHERE w.CASEID = z.CASEID))      AS "pyIssue",
    (SELECT g.STARTDATE
       FROM T_GENERAL g
      WHERE g.NOPOLIS = a.NOPOLIS AND g.PRODKE = a.PRODKE)  AS "PlatNo",
    (SELECT g.ENDDATE
       FROM T_GENERAL g
      WHERE g.NOPOLIS = a.NOPOLIS AND g.PRODKE = a.PRODKE)  AS "Position"
FROM POOLDATA.T_CLAIM_PNC a
JOIN POOLDATA.T_CLAIM_ADJUSTMENT d ON d.CLAIMID = a.CLAIMID
WHERE a.REGISTERDATE >= :1
  AND a.REGISTERDATE < :2 + INTERVAL '1' DAY
  AND d.ADJUSTMENTID = (SELECT MIN(r.ADJUSTMENTID)
                          FROM POOLDATA.T_CLAIM_ADJUSTMENT r
                         WHERE r.CLAIMID = d.CLAIMID)
  AND d.ANALYST_TFKOMITEDATE IS NOT NULL
  AND a.BRANCHCODE <> '100639'
  AND a.GROUPPANEL IN ('003', '004', '006', '009')
  AND a.BUSINESSCODE NOT IN
      ('10145', '10168', '10165', '10164', '10053',
       '10075', '10126', '10011', '10077', '10007')
  AND a.PICTEKNIK IN /*PIC_LIST*/
ORDER BY a.CLAIMID ASC

-- name: ekspor_pic_progress
-- Meniru `GetReportprogressKlaimKPIPIC` — satu baris per PEMBARUAN PROGRES, bukan per
-- klaim. Inilah pilihan bawaan dropdown "Pilih Data KPI".
--
-- # Kenapa sub-kueri berkorelasinya DIPERTAHANKAN, meski berulang tiga puluh kali
--
-- Tabel dasarnya `GCNM_PROGRESS_CLAIM` dan `PEGA_DASHBOARDPNC`; kolom klaimnya diambil
-- satu per satu dari `T_CLAIM_PNC` lewat sub-kueri skalar yang isinya sama persis.
-- Menggantinya dengan satu LEFT JOIN akan jauh lebih pendek dan lebih cepat — tetapi
-- MENGUBAH perilaku bila pasangannya tidak tunggal: sub-kueri skalar gagal dengan
-- `ORA-01427`, sedangkan join MENGGANDAKAN baris. Yang pertama terlihat, yang kedua diam.
--
-- Karena ekspor ini dibandingkan baris per baris dengan berkas Pega (`P-5`), bentuknya
-- dipertahankan apa adanya.
--
-- # Penyaring lini
--
-- Kueri Pega menutupnya dengan `{ASIS:TempLaporan.UserTeknis}` — potongan SQL yang disusun
-- `EksportDataAllKPIPICKlaim` (langkah "set default NONMBU", "jika PA", "jika TRAVEL",
-- "jika BONDING"). Tab ini selalu NONMBU (lihat ReportKPIPICTeknik.tsx), sehingga yang
-- berlaku adalah penyaring Non-MBU — sama persis dengan `progress_counts` pada kartu skor.
SELECT
    (SELECT a.CLAIMID FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "POLICYDECLARATIONNO",
    b.NOKLAIM                                                                            AS "IDPEGA",
    b.NOPOLIS                                                                            AS "EDMNO",
    (SELECT REPLACE(a.QQNAME, ',', ' ') FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "QQNAME",
    (SELECT a.BUSINESSNAME FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "NOPOLIS",
    (SELECT a.LEADER_MEMBER FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "BUSINESSTYPE",
    b.DATEOFLOSS                                                                         AS "ENDDATE",
    (SELECT x.RECEIVEDDATE_1 FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK x
      WHERE x.PZINSKEY = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM)                             AS "PRODKE",
    (SELECT a.FINISHREGISTERDATE FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "ACCUMCODE",
    (SELECT a.CLOSECLAIMDATE FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "TABLEOBJECTNAME",
    (SELECT a.REGISTERDATE FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "REGISTERID",
    (SELECT a.TRANSFERPIC_DATE FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "WARRANTYNO",
    (SELECT MIN(TANGGAL + INTERVAL '12' HOUR) FROM OS_AKSEPTASI_KLAIM
      WHERE CASEID = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM)                                 AS "pyCountry",
    (SELECT a.DATEOFREQUESTDOCUMENT FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "THEINSURED",
    (SELECT a.SURVEYDATE FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "BUSINESSCODE",
    b.PIC                                                                                AS "OLDPOLICYNO",
    (SELECT REPLACE(a.CLOSECLAIMNOTE, CHR(10), ' ') FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "GROUPPANEL",
    (SELECT a.INVESTIGATOR_TF_DATE FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "SOBLEADER1",
    (SELECT a.ANALYSTTORCLPUCL_DATE FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "SOBLEADER2",
    (SELECT a.RCLPUCL_TF_TOANALYST FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "SOBLEADER0",
    (SELECT a.RCLPUCL FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "SOBNAME",
    (SELECT a.COMPLIANCE_CREATEDATE FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "SOURCEOFBUSINESS",
    (SELECT a.CPLVALID_DATE FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "ISMAINFOLLOWING",
    (SELECT a.POSTAUDIT_TF_ANALYSTDATE FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "STATUSBUSINESS",
    (SELECT a.CPLPOSTAUDIT_VALIDDATE FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "EDMTYPE",
    (SELECT a.ALASANKETERLAMBATAN FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "MARKETINGCODE",
    (SELECT a.BUSINESSCODE FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "BUSINESSNAME",
    (SELECT s.LSC_NOTE FROM V_STS_CLAIM s
      WHERE s.LSC_ID = (SELECT a.STATUSCLAIM FROM POOLDATA.T_CLAIM_PNC a
                         WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM
                           AND a.NOPOLIS = b.NOPOLIS))                                   AS "CLIENTID",
    (SELECT a.SOBNAME FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "ISFOLLOWINGCHILD",
    b.NOKLAIM                                                                            AS "pyNote",
    (SELECT x.PXCREATEOPERATOR FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK x
      WHERE x.PZINSKEY = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM)                             AS "pyLabel",
    (SELECT a.TGLDOKLENGKAP FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "pyGroup",
    (SELECT a.TRF_TO_INVESTIGATOR FROM POOLDATA.T_CLAIM_PNC a
      WHERE a.CLAIMNO = 'ASM-FW-GCNMFW-WORK ' || b.NOKLAIM AND a.NOPOLIS = b.NOPOLIS)   AS "pzInsKey",
    (SELECT SUBSTR(x.REPORTDATE_1, 1, 8) FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK x
      WHERE x.PYID = b.NOKLAIM)                                                          AS "IndexSpreading",
    (SELECT z.POSISI FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC z
      WHERE z.CASEID = b.NOKLAIM
        AND z.ID = (SELECT MAX(w.ID) FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC w
                     WHERE w.CASEID = z.CASEID))                                         AS "pyIssue",
    (SELECT g.STARTDATE FROM T_GENERAL g
      WHERE g.NOPOLIS = b.NOPOLIS AND g.PRODKE = b.PROD_KE)                              AS "PlatNo",
    (SELECT g.ENDDATE FROM T_GENERAL g
      WHERE g.NOPOLIS = b.NOPOLIS AND g.PRODKE = b.PROD_KE)                              AS "Position",
    c.TGL_INPUT                                                                          AS "DISC",
    c.NEXT_FOLLOWUP                                                                      AS "DISC2",
    CASE
        WHEN (SELECT t.TGL_INPUT
                FROM POOLDATA.GCNM_PROGRESS_CLAIM t
               WHERE t.PNCCASEID = c.PNCCASEID AND t.ID_UPDATE > c.ID_UPDATE
               ORDER BY t.ID_UPDATE ASC FETCH NEXT 1 ROW ONLY) IS NULL THEN 'Tercapai'
        ELSE
            CASE
                WHEN c.NEXT_FOLLOWUP + INTERVAL '1' DAY >=
                     (SELECT t.TGL_INPUT
                        FROM POOLDATA.GCNM_PROGRESS_CLAIM t
                       WHERE t.PNCCASEID = c.PNCCASEID AND t.ID_UPDATE > c.ID_UPDATE
                       ORDER BY t.ID_UPDATE ASC FETCH NEXT 1 ROW ONLY) THEN 'Tercapai'
                ELSE 'Tidak Tercapai'
            END
    END                                                                                  AS "ASMCityId",
    c.KETERANGAN                                                                         AS "CARI6",
    (SELECT x.STS_PROGRESS1 FROM POOLDATA.GCNM_MST_PROGRESS x
      WHERE x.ID_PROGRESS = c.STATUS_PROGRESS1 FETCH NEXT 1 ROW ONLY)                    AS "AidaAlamat",
    (SELECT j.SerialNo
       FROM POOLDATA.GCNM_PROGRESS_CLAIM g,
            JSON_TABLE(g.JSONSTATUS_PROGRESS2, '$'
                       COLUMNS (NESTED PATH '$.ObjectList[*]'
                                COLUMNS (SerialNo VARCHAR2(100) PATH '$.SerialNo'))) j
      WHERE g.PNCCASEID = c.PNCCASEID
        AND g.ID_UPDATE = c.ID_UPDATE
        AND g.STATUS_PROGRESS2 NOT IN ('2', '24', '60', '59')
      FETCH NEXT 1 ROW ONLY)                                                             AS "AidaNama"
FROM POOLDATA.GCNM_PROGRESS_CLAIM c
JOIN POOLDATA.PEGA_DASHBOARDPNC b ON c.PNCCASEID = b.NOKLAIM
WHERE b.STSKLAIM NOT IN ('1', '2', '3')
  AND UPPER(c.KETERANGAN) NOT LIKE '%AUTO%'
  AND UPPER(c.KETERANGAN) NOT LIKE '%SYSTEM%'
  AND b.PIC = c.USER_INPUT
  AND b.PIC <> 'ASNET'
  AND c.TGL_INPUT >= :1
  AND c.TGL_INPUT < :2 + INTERVAL '1' DAY
  AND b.GROUP_PANEL IN ('003', '004', '006')
  AND b.GROUPBISNISID NOT IN ('09', '11', '16', '25')
  AND b.PIC IN /*PIC_LIST*/
