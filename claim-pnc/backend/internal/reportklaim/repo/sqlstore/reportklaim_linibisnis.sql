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
--
-- SUMBER BARU (2026-10-08): kolom "Location" dulu anak-kueri ke objek kerja Pega
-- (`PC_ASM_FW_GCNMFW_WORK.location_1`), kini `T_CLAIM_PNC.LOCATION` (`a`) langsung.
-- Terukur di Oracle dev untuk 49 klaim HE: objek kerja berisi lokasi pada 20, T_CLAIM_PNC
-- pada 29, dan 19 nilainya sama — kolom ini jadi lebih sering terisi, bukan berkurang.
SELECT a.claimno    AS "ClaimNo",
       a.nopolis    AS "PolicyNo",
       a.picteknik  AS "PICRekanan",
       a.registerdate AS "RegisterDate",
       a.dateofloss AS "DateOfLoss",
       a.qqname     AS "UserName",
       b.causeofloss AS "CauseOfLoss",
       a.sobname    AS "BusinessName",
       a.location   AS "Location",
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
--
-- # Objek kerja Pega tidak dibaca lagi (2026-10-08)
--
-- ("Perubahan nama tabel untuk Inbox.xlsx", baris 17.)
--
--   a  berkas penerimaan dokumen  -> `T_CLAIMLIST_ADMIN` baris ber-PXOBJCLASS ReceiveDocument;
--                                    ia membawa KURIR, PNCCASEID, KODECABANG_1, dan kolom polis
--   b  klaim pasangannya          -> `T_CLAIM_PNC`, disambung `b.RCVID = a.PYID`.
--
-- Sambungan lamanya `a.PNCCASEID = b.PYID` TIDAK dapat dipakai: `PNCCASEID` pada baris berkas
-- penerimaan di T_CLAIMLIST_ADMIN kosong di SELURUH 143 baris (diukur 2026-10-08). Arah
-- sebaliknya terisi — `T_CLAIM_PNC.RCVID` menyimpan nomor berkas penerimaan asal klaim, dan
-- sama dengan `PNCCASEID` objek kerja klaim pada 501 dari 503 klaim yang punya keduanya.
--
-- Kolom IDPEGA (nomor klaim) karena itu dibaca dari `b`: ekor `CLAIMID` tanpa prefix
-- `ASM-FW-GCNMFW-WORK `, yaitu nomor case Pega persis — bukan `CLAIMNO`, yang pada 15 klaim
-- berbeda dari nomor case-nya sendiri.
--
--   b.CLOSECLAIMDATE_1 -> CLOSECLAIMDATE · b.CLOSECLAIMNOTE_1 -> CLOSECLAIMNOTE ·
--   b.USERTEKNIS_1 -> PICTEKNIK
--
-- Kode status klaim kini dibaca dari baris `b` itu sendiri. Kueri lama mencarinya lagi ke
-- `T_CLAIM_PNC` lewat `CLAIMNO = PYID` dalam subkueri skalar — yang GAGAL (ORA-01427) bila
-- satu CLAIMNO muncul dua kali, dan diukur ada satu yang begitu.
--
-- AKIBAT YANG HARUS DISADARI: `T_CLAIMLIST_ADMIN` hanya memuat berkas penerimaan yang MASIH
-- di antrean Admin — artinya yang BELUM diregistrasi menjadi klaim. Laporan ini justru
-- menampilkan berkas yang SUDAH menjadi klaim. Diukur 2026-10-08: dari 112 berkas Auto
-- Service di tabel itu, hanya 1 yang punya klaim. Laporan ini karena itu nyaris kosong selama
-- T_CLAIMLIST_ADMIN tidak menyimpan berkas yang sudah keluar antrean.
SELECT a.pxcreatedatetime AS "EDMDATE",
       a.pyid             AS "EDMNO",
       COALESCE(a.pnccaseid, REPLACE(b.claimid, 'ASM-FW-GCNMFW-WORK ', '')) AS "IDPEGA",
       a.policyno         AS "NOPOLIS",
       b.qqname           AS " QQNAME",
       a.dateofloss_1     AS "STARTDATE",
       a.businessname     AS "SOBNAME",
       b.closeclaimdate   AS "ENDDATE",
       b.closeclaimnote   AS "BUSINESSCODE",
       (SELECT z.branchname FROM branch z WHERE z.id = a.kodecabang_1) AS "MARKETINGNAME",
       (SELECT v.lsc_note
          FROM pooldata.v_sts_claim v
         WHERE v.lsc_id = b.statusclaim) AS "CLIENTID",
       b.picteknik        AS "WARRANTYNO"
  FROM pooldata.t_claimlist_admin a
  JOIN pooldata.t_claim_pnc b ON b.rcvid = a.pyid AND b.claimno IS NOT NULL
 WHERE a.kurir = 'Auto Service'
   AND a.pxobjclass = 'ASM-FW-GCNMFW-Work-ReceiveDocument'
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
-- # Dua anak-kueri ditulis ulang, bukan disalin
--
-- Kueri aslinya membaca ClientID dari dokumen polis
-- (`d.DATA_JSONBLOB.CIFData.Customer_P/Customer_C.ASMClientID`). Sejak 2026-10-01 data
-- polis tidak dibaca dari DATA_JSONBLOB: ClientID diambil dari T_GENERAL.CLIENTID pada
-- NOPOLIS + PRODKE. Terverifikasi sama dengan Customer_P (atau Customer_C bila Customer_P
-- kosong) pada 10 dari 10 contoh. Customer_C yang BERBEDA dari Customer_P tidak lagi ikut
-- dicocokkan. FETCH FIRST 1 mencegah galat bila T_GENERAL punya lebih dari satu baris.
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
          JOIN T_GENERAL d
            ON c.clientid = d.CLIENTID
         WHERE d.nopolis = a.nopolis
           AND c.telfaxtype = '6'
           AND REPLACE(d.prodke, ' ', '') = REPLACE(a.prod_ke, ' ', '')
         FETCH FIRST 1 ROWS ONLY) AS "NewEmail",
       (SELECT c.telfaxnumber
          FROM t_mclienttelfax c
          JOIN T_GENERAL d
            ON c.clientid = d.CLIENTID
         WHERE d.nopolis = a.nopolis
           AND c.telfaxtype = '4'
           AND REPLACE(d.prodke, ' ', '') = REPLACE(a.prod_ke, ' ', '')
         FETCH FIRST 1 ROWS ONLY) AS "NewTelpTertanggung"
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
