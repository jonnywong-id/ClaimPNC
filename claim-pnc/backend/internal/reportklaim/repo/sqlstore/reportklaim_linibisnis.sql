-- Kueri laporan kelompok Lini Bisnis & Mitra Kerja Sama, ditambah pilihan dropdown
-- "Bisnis".
--
-- Kelima aturan yang mengikat berkas kueri modul ini disebut di
-- reportklaim_reasuransi.sql dan berlaku sama di sini.


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

-- ============================================================================
-- Kueri yang DICABUT 2026-10-13
-- ============================================================================
--
-- Panel pemiliknya tidak ada di Pega produksi, dan Work Owner menetapkan migrasi
-- mengikuti produksi. Yang dicabut dari berkas ini:
----   report_business_options
--   report_klaim_asuransi_kredit
--   report_klaim_per_bisnis
--   report_klaim_traveloka
--   report_klaim_pegipegi
