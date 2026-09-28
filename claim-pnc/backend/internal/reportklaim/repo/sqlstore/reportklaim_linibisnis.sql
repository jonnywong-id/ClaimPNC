-- Kueri laporan kelompok Lini Bisnis & Mitra Kerja Sama, ditambah pilihan dropdown
-- "Bisnis".
--
-- Kelima aturan yang mengikat berkas kueri modul ini disebut di
-- reportklaim_reasuransi.sql dan berlaku sama di sini.


-- name: report_business_options
--
-- Isi autocomplete "Bisnis" pada panel REPORT KLAIM PER BISNIS.
--
-- Asal: `Report Definition/BrowseBusiness_RD-RD.xml`, kelas `ASM-FW-GISFW-Int-BUSINESS`.
-- Harness memakai `.Note` sebagai yang dibaca pengguna dan menyimpan `.ID` ke
-- `TempLaporan.CountryID` — dan `.ID` itulah yang dibandingkan dengan `businesscode`.
--
-- Diurutkan menurut nama, bukan menurut kode. Report Definition aslinya tidak menyatakan
-- urutan sama sekali, dan daftar tanpa urutan pada autocomplete berisi ratusan baris
-- adalah daftar yang harus dibaca seluruhnya.
SELECT id   AS code,
       note AS name
  FROM business
 WHERE note IS NOT NULL
 ORDER BY note


-- name: report_klaim_he
--
-- REPORT KLAIM HE — 27 kolom. HE = Heavy Equipment.
--
-- Asal: `RDB List/BrowseKlaimHE-SQL.xml`, dijalankan
-- `Activity/PNCReportKlaimHE_act-Act.xml` dengan `idreportKlaim = "1"`.
--
-- Bind:
--   :1  tanggal registrasi dari    DATE
--   :2  tanggal registrasi sampai  DATE
--
-- `BUSINESSCODE='10043'` adalah penyaring lini HE itu sendiri; ia tetap literal karena
-- ia bagian dari identitas laporan ini, bukan pilihan pengguna.
SELECT a.claimno    AS "ClaimNo",
       a.nopolis    AS "PolicyNo",
       a.picteknik  AS "PICRekanan",
       a.registerdate AS "RegisterDate",
       a.dateofloss AS "DateOfLoss",
       a.qqname     AS "UserName",
       b.causeofloss AS "CauseOfLoss",
       a.sobname    AS "BusinessName",
       (SELECT w.location_1 FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w WHERE w.pzinskey = a.claimid) AS "Location",
       (SELECT SUM(t.price)          FROM POOLDATA.T_CLAIM_TREATMENT t WHERE t.claimid = a.claimid AND t.treatmenttype = '1') AS "AlasanKlaim",
       (SELECT AVG(t.diskon)         FROM POOLDATA.T_CLAIM_TREATMENT t WHERE t.claimid = a.claimid AND t.treatmenttype = '1') AS "AlasanTerlambat",
       (SELECT SUM(t.treatmentvalue) FROM POOLDATA.T_CLAIM_TREATMENT t WHERE t.claimid = a.claimid AND t.treatmenttype = '1') AS "City",
       (SELECT SUM(t.price)          FROM POOLDATA.T_CLAIM_TREATMENT t WHERE t.claimid = a.claimid AND t.treatmenttype = '2') AS "District",
       (SELECT AVG(t.diskon)         FROM POOLDATA.T_CLAIM_TREATMENT t WHERE t.claimid = a.claimid AND t.treatmenttype = '2') AS "BranchName",
       (SELECT SUM(t.treatmentvalue) FROM POOLDATA.T_CLAIM_TREATMENT t WHERE t.claimid = a.claimid AND t.treatmenttype = '2') AS "ConsultantName",
       (SELECT p.nama_bengkel    FROM POOLDATA.T_CLAIM_PENAWARAN p WHERE p.claimid = a.claimid FETCH NEXT 1 ROW ONLY) AS "InsuredName",
       (SELECT p.wilayah_bengkel FROM POOLDATA.T_CLAIM_PENAWARAN p WHERE p.claimid = a.claimid FETCH NEXT 1 ROW ONLY) AS "NamaDokumen",
       (SELECT n.suppliername    FROM POOLDATA.T_CLAIM_PANEL n     WHERE n.claimid = a.claimid FETCH NEXT 1 ROW ONLY) AS "NamaSurveyor",
       (SELECT z.occupationname  FROM POOLDATA.T_ANEKALIST z
         WHERE z.nopolis = a.nopolis AND z.prodke = a.prodke AND z.indexobject = b.objectid
         FETCH NEXT 1 ROW ONLY) AS "Occupation",
       (SELECT c.modelname      FROM POOLDATA.T_CLAIM_OBJECTLIST c WHERE c.claimid = a.claimid AND c.objectid = b.objectid) AS "UserTeknis",
       (SELECT c.brandname      FROM POOLDATA.T_CLAIM_OBJECTLIST c WHERE c.claimid = a.claimid AND c.objectid = b.objectid) AS "UserTeknisEmail",
       (SELECT c.manufactureyear FROM POOLDATA.T_CLAIM_OBJECTLIST c WHERE c.claimid = a.claimid AND c.objectid = b.objectid) AS "TreatyName",
       (SELECT c.enginenumber   FROM POOLDATA.T_CLAIM_OBJECTLIST c WHERE c.claimid = a.claimid AND c.objectid = b.objectid) AS "TreatyYear",
       (SELECT c.chasissnumber  FROM POOLDATA.T_CLAIM_OBJECTLIST c WHERE c.claimid = a.claimid AND c.objectid = b.objectid) AS "ClientName",
       (SELECT SUM(e.convertvalue) FROM POOLDATA.T_CLAIM_ESTIMASI e   WHERE e.claimid = a.claimid) AS "Remark",
       (SELECT SUM(x.total_claim)  FROM POOLDATA.T_CLAIM_ADJUSTMENT x WHERE x.claimid = a.claimid) AS "EmailTertanggung",
       (SELECT SUM(x.grossvalue)   FROM POOLDATA.T_CLAIM_ADJUSTMENT x WHERE x.claimid = a.claimid) AS "Province"
  FROM T_CLAIM_PNC a
  JOIN T_CLAIM_OBJECTCOVERAGE b ON a.claimid = b.claimid
 WHERE a.businesscode = '10043'
   AND CAST(a.registerdate AS DATE) >= :1
   AND CAST(a.registerdate AS DATE) <= :2
 ORDER BY a.registerdate ASC


-- name: report_regist_simas_online
--
-- REPORT DATA REGIST SIMAS ON LINE — 12 kolom.
--
-- Asal: `RDB List/GetDataRegistBySimasOnline-SQL.xml`, dijalankan
-- `Activity/PNCReportKlaimHE_act-Act.xml` dengan `idreportKlaim = "2"`.
--
-- Tanpa bind: kuerinya memang tidak menerima satu pun parameter.
--
-- # Satu alias berspasi yang TIDAK diperbaiki
--
-- Kueri aslinya menulis `b.QQNAME AS " QQNAME"` — dengan SPASI DI DEPAN. Nama properti
-- yang dicari `CSVProperties` adalah `QQNAME` tanpa spasi, sehingga kolom "Nama
-- Tertanggung" pada berkas lama **selalu kosong**.
--
-- Spasinya sempat dibuang supaya kolomnya terisi. Itu DICABUT: Work Owner menetapkan
-- 2026-09-25 bahwa cacat yang berasal dari Pega dibiarkan seperti Pega, dan yang
-- diperbaiki hanya cacat pemindahan.
--
-- Yang berubah karena itu bukan isinya, melainkan bahwa ketiadaannya kini TERCATAT di
-- kolomTanpaSumber dan terkunci uji.
SELECT a.pxcreatedatetime AS "EDMDATE",
       a.pyid             AS "EDMNO",
       a.pnccaseid        AS "IDPEGA",
       a.policyno         AS "NOPOLIS",
       b.qqname           AS " QQNAME",
       a.dateofloss_1     AS "STARTDATE",
       a.businessname     AS "SOBNAME",
       b.closeclaimdate_1 AS "ENDDATE",
       b.closeclaimnote_1 AS "BUSINESSCODE",
       (SELECT z.branchname FROM branch z WHERE z.id = a.kodecabang_1) AS "MARKETINGNAME",
       (SELECT v.lsc_note
          FROM pooldata.v_sts_claim v
         WHERE v.lsc_id = (SELECT p.statusclaim FROM pooldata.t_claim_pnc p WHERE p.claimno = b.pyid)) AS "CLIENTID",
       b.userteknis_1     AS "WARRANTYNO"
  FROM datapega.pc_asm_fw_gcnmfw_work a
  JOIN datapega.pc_asm_fw_gcnmfw_work b ON a.pnccaseid = b.pyid
 WHERE a.kurir = 'Auto Service'
   AND a.pxobjclass = 'ASM-FW-GCNMFW-Work-ReceiveDocument'
   AND b.pxobjclass = 'ASM-FW-GCNMFW-Work-PNC'
 ORDER BY a.pxcreatedatetime ASC


-- name: report_klaim_asuransi_kredit
--
-- REPORT KLAIM ASURANSI KREDIT — 3 kolom.
--
-- Asal: `RDB List/GetDataKlaimAsuransiKredit-SQL.xml` dan
-- `RDB List/GetClientNameAutoKlaim-SQL.xml`, dijalankan
-- `Activity/PNCReportKlaimKredit_act-Act.xml`.
--
-- Bind:
--   :1  tanggal proses dari    DATE
--   :2  tanggal proses sampai  DATE
--
-- Kolom "Sumber Bisnis" di sistem lama diambil lewat kueri KEDUA yang dijalankan sekali
-- per baris hasil kueri pertama — satu pembacaan basis data per agen. Di sini ia menjadi
-- anak-kueri pada kueri yang sama: hasilnya identik, dan jumlah pembacaannya satu.
SELECT a.userinput          AS "EDMNO",
       SUM(a.nilaiklaim)    AS "CLIENTID",
       (SELECT m.nama_penerima
          FROM pooldata.m_auto_claim_pnc m
         WHERE m.inisialid = a.agenid) AS "PRODKE"
  FROM pooldata.tmp_batch_claim_kredit a
 WHERE a.tmp_message = 'Sukses Klaim'
   AND CAST(a.tglproses AS DATE) >= :1
   AND CAST(a.tglproses AS DATE) <= :2
 GROUP BY a.agenid, a.userinput
 ORDER BY a.userinput ASC


-- name: report_klaim_per_bisnis
--
-- REPORT KLAIM PER BISNIS — 14 kolom.
--
-- Asal: `RDB List/BrowseDataClaimBusiness-SQL.xml`.
--
-- Bind:
--   :1  kode bisnis terpilih   dari autocomplete "Bisnis"
--
-- # Dua anak-kueri JSON ditulis ulang, bukan disalin
--
-- Kueri aslinya membaca JSON dengan notasi titik khas Oracle
-- (`d.DATA_JSONBLOB.CIFData.Customer_P.ASMClientID`). Notasi itu tidak ada di
-- PostgreSQL. Penggantinya `JSON_VALUE` dengan jalur SQL/JSON standar — sintaks yang
-- SAMA di Oracle 12c+ dan PostgreSQL 17+, dan justru itulah sebab `D-24` menetapkan
-- PostgreSQL 17 sebagai syarat mengikat.
SELECT a.noklaim   AS "ClaimNo",
       a.nopolis   AS "PolicyNo",
       a.begindate AS "City",
       a.enddate   AS "CityID",
       a.tgl_proses AS "Country",
       a.dateofloss AS "CountryID",
       a.pic       AS "UserTeknis",
       a.tsi       AS "TKI",
       a.ttlos     AS "CaseID",
       a.ttlaksep  AS "RefNo",
       a.sobname   AS "SIM",
       CASE
         WHEN a.stsklaim IN ('2', '3') THEN 'REJECT'
         WHEN a.stsklaim = '1'         THEN 'CLOSE'
         WHEN a.stsklaim = '0' OR a.stsklaim IS NULL THEN 'OS'
       END AS "StatusClaim",
       (SELECT c.telfaxnumber
          FROM t_mclienttelfax c
          JOIN JSON_POLIS d
            ON c.clientid IN (
                 JSON_VALUE(d.data_jsonblob, '$.CIFData.Customer_P.ASMClientID'),
                 JSON_VALUE(d.data_jsonblob, '$.CIFData.Customer_C.ASMClientID'))
         WHERE d.nopolis = a.nopolis
           AND c.telfaxtype = '6'
           AND REPLACE(d.prodke, ' ', '') = REPLACE(a.prod_ke, ' ', '')) AS "NewEmail",
       (SELECT c.telfaxnumber
          FROM t_mclienttelfax c
          JOIN JSON_POLIS d
            ON c.clientid IN (
                 JSON_VALUE(d.data_jsonblob, '$.CIFData.Customer_P.ASMClientID'),
                 JSON_VALUE(d.data_jsonblob, '$.CIFData.Customer_C.ASMClientID'))
         WHERE d.nopolis = a.nopolis
           AND c.telfaxtype = '4'
           AND REPLACE(d.prodke, ' ', '') = REPLACE(a.prod_ke, ' ', '')) AS "NewTelpTertanggung"
  FROM pooldata.pega_dashboardpnc a
 WHERE EXISTS (SELECT 1
                 FROM pooldata.t_claim_pnc p
                WHERE p.businesscode = :1
                  AND p.claimno = a.noklaim)
 ORDER BY a.tgl_proses ASC


-- name: report_klaim_traveloka
--
-- REPORT KLAIM TRAVELOKA — 6 kolom.
--
-- Asal: `RDB List/BrowseDataClaimTraveloka-SQL.xml`, dijalankan
-- `Activity/ExportDataClaimTravel-Act.xml` dengan `tipe = "TRVLK"`.
--
-- Bind:
--   :1  tanggal registrasi dari    DATE
--   :2  tanggal registrasi sampai  DATE
--
-- Penyaring mitranya `a.nopolis LIKE '%T%'`, disisipkan activity lewat
-- `TempLaporan.Message`. Ia ditulis di sini sebagai bagian kueri karena ia memang
-- konstanta — dan karena laporan PegiPegi di bawah memakai penyaring yang BERBEDA atas
-- kueri yang sama.
SELECT a.nopolis        AS "PolicyNo",
       a.registerdate   AS "BranchID",
       a.claimno        AS "ClaimNo",
       b.objectname     AS "BranchName",
       c.noakseptasi    AS "CaseID",
       c.nilaiakseptasi AS "CauseOfLoss"
  FROM t_claim_pnc a
  JOIN t_claim_objectlist b ON a.claimid = b.claimid
  JOIN t_claim_adjustment c ON b.claimid = c.claimid AND b.objectid = c.objectid
 WHERE a.grouppanel = '005'
   AND a.nopolis LIKE '%T%'
   AND CAST(a.registerdate AS DATE) >= :1
   AND CAST(a.registerdate AS DATE) <= :2
 ORDER BY a.registerdate ASC


-- name: report_klaim_pegipegi
--
-- REPORT KLAIM PEGIPEGI — 6 kolom, susunan sama dengan Traveloka.
--
-- Asal: kueri yang SAMA, dijalankan dengan `tipe = "PEGI"`, yang menukar penyaring
-- nomor polis menjadi `a.nopolis LIKE '122N%'`.
--
-- Bind:
--   :1  tanggal registrasi dari    DATE
--   :2  tanggal registrasi sampai  DATE
SELECT a.nopolis        AS "PolicyNo",
       a.registerdate   AS "BranchID",
       a.claimno        AS "ClaimNo",
       b.objectname     AS "BranchName",
       c.noakseptasi    AS "CaseID",
       c.nilaiakseptasi AS "CauseOfLoss"
  FROM t_claim_pnc a
  JOIN t_claim_objectlist b ON a.claimid = b.claimid
  JOIN t_claim_adjustment c ON b.claimid = c.claimid AND b.objectid = c.objectid
 WHERE a.grouppanel = '005'
   AND a.nopolis LIKE '122N%'
   AND CAST(a.registerdate AS DATE) >= :1
   AND CAST(a.registerdate AS DATE) <= :2
 ORDER BY a.registerdate ASC
