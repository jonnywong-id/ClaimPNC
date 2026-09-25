-- Kueri laporan kelompok Akseptasi & Penyelesaian.
--
-- Kelima aturan yang mengikat berkas kueri modul ini disebut di
-- reportklaim_reasuransi.sql dan berlaku sama di sini.
--
-- ============================================================================
-- Satu pola yang berulang di kelompok ini: pembagian nilai ke para penanggung
-- ============================================================================
--
-- Tiga laporan — Akseptasi, OS Komite, dan OS Belum Komite — mengalikan nilai share ASM
-- dengan **enam belas persentase treaty** dari POOLDATA.PEGA_DASHBOARDPNC:
--
--	PRSN_BPPDAN  PRSN_FAC_OUT  PRSN_QS_RI   PRSN_FSPL     PRSN_PSRSPL  PRSN_PSPLNSRI
--	PRSN_PSRQS_RI PRSN_FESPL   PRSN_FAC_OBL PRSN_FACOBSRB PRSN_FACOBINDT
--	PRSN_ER1     PRSN_ER2      PRSN_PSS     PRSN_PRGBI    PRSN_PFRA     PRSN_XL
--
-- Keenam belasnya disalin apa adanya, termasuk urutannya, karena itulah urutan kolom di
-- berkas yang selama ini diterima penggunanya.
--
-- Yang DIKALIKAN berbeda antar laporan, dan perbedaannya bukan gaya penulisan:
--
--	Akseptasi         a.ASM_SHARE_VALUE                nilai share yang sudah diakseptasi
--	OS Komite         a.ASM_SHARE_VALUE                idem
--	OS Belum Komite   a.shareasm / 100 * c.TTLOS       estimasi, karena akseptasinya belum ada


-- name: report_kasir_sudah_bayar
--
-- REPORT DATA KASIR SUDAH BAYAR — 9 kolom.
--
-- Asal: `RDB List/GetBrowseTransferDataToKasir-SQL.xml`, dijalankan
-- `Activity/ExportTransferKeKasir-Act.xml` dengan `STSKASIR = "1"`.
--
-- Bind:
--   :1  tanggal transfer kasir dari    DATE
--   :2  tanggal transfer kasir sampai  DATE
--   :3  kode lini bisnis
--
-- Ketiga penyaringnya di sistem lama disisipkan lewat `{ASIS:}` dari halaman
-- `TempDataDetailKasir`, bukan lewat bind — sehingga membaca kuerinya saja akan
-- menyimpulkan laporan ini tidak punya penyaring sama sekali.
--
-- Pembeda panel ini dari tetangganya: `CLAIM_STATUS_PAID IS NOT NULL`.
SELECT a.claimid              AS "CaseID",
       b.clientname           AS "NamaSurveyor",
       b.pic                  AS "NIK",
       a.noakseptasi          AS "City",
       a.tglakseptasi         AS "CityID",
       a.receivername         AS "Country",
       a.asm_share            AS "NoKTP",
       a.asm_share_value      AS "NPWP",
       a.transfer_cashier_date AS "CountryID"
  FROM POOLDATA.T_CLAIM_ADJUSTMENT a
  JOIN POOLDATA.PEGA_DASHBOARDPNC b ON b.noklaim = SUBSTR(a.claimid, 20)
 WHERE a.claim_status_paid IS NOT NULL
   AND CAST(a.transfer_cashier_date AS DATE) >= :1
   AND CAST(a.transfer_cashier_date AS DATE) <= :2
   AND (
        :3 = ''
     OR (:3 = '002' AND b.group_panel = '002')
     OR (:3 = '005' AND b.group_panel = '005')
     OR (:3 = '346' AND b.group_panel IN ('003','004','006','009'))
     OR (:3 = '003' AND b.group_panel = '003')
       )
 ORDER BY a.transfer_cashier_date ASC


-- name: report_kasir_belum_bayar
--
-- REPORT DATA KASIR BELUM BAYAR — 9 kolom, susunan sama dengan panel di atas.
--
-- Pembedanya satu baris: `CLAIM_STATUS_PAID IS NULL`.
--
-- Bind: sama dengan report_kasir_sudah_bayar.
SELECT a.claimid              AS "CaseID",
       b.clientname           AS "NamaSurveyor",
       b.pic                  AS "NIK",
       a.noakseptasi          AS "City",
       a.tglakseptasi         AS "CityID",
       a.receivername         AS "Country",
       a.asm_share            AS "NoKTP",
       a.asm_share_value      AS "NPWP",
       a.transfer_cashier_date AS "CountryID"
  FROM POOLDATA.T_CLAIM_ADJUSTMENT a
  JOIN POOLDATA.PEGA_DASHBOARDPNC b ON b.noklaim = SUBSTR(a.claimid, 20)
 WHERE a.claim_status_paid IS NULL
   AND CAST(a.transfer_cashier_date AS DATE) >= :1
   AND CAST(a.transfer_cashier_date AS DATE) <= :2
   AND (
        :3 = ''
     OR (:3 = '002' AND b.group_panel = '002')
     OR (:3 = '005' AND b.group_panel = '005')
     OR (:3 = '346' AND b.group_panel IN ('003','004','006','009'))
     OR (:3 = '003' AND b.group_panel = '003')
       )
 ORDER BY a.transfer_cashier_date ASC


-- name: report_pending_lod
--
-- REPORT DATA PENDING LOD — 11 kolom.
--
-- Asal: `RDB List/BrowseDataPendingLOD-SQL.xml`.
--
-- Bind:
--   :1  tanggal registrasi dari    DATE
--   :2  tanggal registrasi sampai  DATE
--   :3  kode lini bisnis
--
-- Yang dicari: akseptasi yang sudah bernilai tetapi LOD-nya belum tercetak —
-- `STATUSAKSEPTASILOD IS NULL` dan `NOAKSEPTASI IS NULL`, pada klaim yang belum tutup.
SELECT b.nopolis    AS "PolicyNo",
       b.claimno    AS "ClaimNo",
       b.picteknik  AS "UserTeknis",
       b.qqname     AS "InsuredName",
       b.leader_member AS "Status",
       (SELECT k.currency FROM pooldata.CURRENCY k WHERE k.id = a.currency) AS "Currency",
       a.propose_value AS "ClientID",
       a.grossvalue    AS "ClientName",
       a.individual_risk_value AS "CloseClaimNote",
       a.printlod_date AS "City",
       b.dateofloss    AS "CityID"
  FROM pooldata.t_claim_adjustment a
  JOIN pooldata.t_claim_pnc b ON a.claimid = b.claimid
  JOIN pooldata.pega_dashboardpnc c ON b.claimno = c.noklaim
  JOIN datapega.pc_asm_fw_gcnmfw_work d ON b.claimid = d.pzinskey
 WHERE a.statusakseptasi IN ('0', '1')
   AND a.statusakseptasilod IS NULL
   AND a.paymenttype IN ('1', '2')
   AND a.noakseptasi IS NULL
   AND a.grossvalue IS NOT NULL
   AND b.branchname <> 'ASNET'
   AND (d.pystatuswork NOT IN ('Resolved-Completed', 'Resolved-Rejected') OR d.ispendingclose = 'true')
   AND CAST(b.registerdate AS DATE) >= :1
   AND CAST(b.registerdate AS DATE) <= :2
   AND (
        :3 = ''
     OR (:3 = '002' AND b.grouppanel = '002')
     OR (:3 = '005' AND b.grouppanel = '005')
     OR (:3 = '346' AND b.grouppanel IN ('003','004','006','009')
                    AND b.businesscode NOT IN ('10145','10168','10165','10164','10053','10075','10126','10011','10077','10007'))
     OR (:3 = '003' AND b.grouppanel = '003'
                    AND b.businesscode IN ('10145','10168','10165','10164','10053','10075','10126','10011','10077','10007'))
       )
 ORDER BY b.registerdate DESC


-- name: report_akseptasi
--
-- REPORT AKSEPTASI — 42 kolom rinci, 26 kolom ringkas.
--
-- Asal: `RDB List/ReportAkseptasiNonMBU-SQL.xml`.
--
-- Bind:
--   :1  tanggal akseptasi dari    DATE
--   :2  tanggal akseptasi sampai  DATE
--
-- Rentangnya atas TANGGAL AKSEPTASI, bukan tanggal registrasi.
--
-- Laporan ini TIDAK menerima pilihan lini bisnis: kuerinya mematok Non-MBU
-- (`grouppanel IN ('003','004','006','009')` dikurangi sepuluh businesscode) sebagai
-- bagian dari identitasnya — nama berkasnya pun berbunyi "Non MBU".
--
-- Kotak centang di panelnya memilih SUSUNAN KOLOM, bukan baris; kuerinya satu.
--
-- # ROWNUM diganti FETCH NEXT
--
-- Ketiga anak-kueri okupasi memakai `rownum = 1` — sebagian bahkan `rownum = '1'`,
-- membandingkan angka dengan teks. Keduanya diganti `FETCH NEXT 1 ROW ONLY`, padanan
-- wajib pada `09-DATABASE-STRATEGY.md` §4 yang sudah dipakai 35 rule di sistem lama.
SELECT b.claimno       AS "CaseID",
       b.nopolis       AS "NoKTP",
       b.qqname        AS "ClaimID",
       b.picteknik     AS "Conveyance",
       b.businessname  AS "ClaimNo",
       c.occupation    AS "TreatyName",
       b.leader_member AS "UserTeknisEmail",
       CASE
         WHEN b.grouppanel IN ('003','009') THEN
           (SELECT t.occupationid FROM pooldata.t_anekalist t
             WHERE t.nopolis = b.nopolis AND t.prodke = b.prodke AND t.indexobject = a.objectid
             FETCH NEXT 1 ROW ONLY)
         WHEN b.grouppanel = '006' THEN
           (SELECT cc.OccupationCode
              FROM pooldata.t_propertylist z,
                   JSON_TABLE(z.OCCUPATIONLIST, '$'
                     COLUMNS (NESTED PATH '$.OccupationList[*]'
                              COLUMNS (OccupationCode VARCHAR PATH '$.OccupationCode'))) cc
             WHERE z.nopolis = b.nopolis AND z.indexobject = a.objectid AND z.prodke = b.prodke
             FETCH NEXT 1 ROW ONLY)
         ELSE
           (SELECT g.goodsid FROM pooldata.t_cargolist g
             WHERE g.nopolis = b.nopolis AND g.prodke = b.prodke AND g.goodsid = a.objectid
             FETCH NEXT 1 ROW ONLY)
       END AS "UserTeknis",
       CASE
         WHEN b.grouppanel IN ('003','009') THEN
           (SELECT t.occupationname FROM pooldata.t_anekalist t
             WHERE t.nopolis = b.nopolis AND t.prodke = b.prodke AND t.indexobject = a.objectid
             FETCH NEXT 1 ROW ONLY)
         WHEN b.grouppanel = '006' THEN
           (SELECT dd.OccupationName
              FROM pooldata.t_propertylist x,
                   JSON_TABLE(x.OCCUPATIONLIST, '$'
                     COLUMNS (NESTED PATH '$.OccupationList[*]'
                              COLUMNS (OccupationName VARCHAR PATH '$.OccupationName'))) dd
             WHERE x.nopolis = b.nopolis AND x.indexobject = a.objectid AND x.prodke = b.prodke
             FETCH NEXT 1 ROW ONLY)
         ELSE
           (SELECT g.goodsname FROM pooldata.t_cargolist g
             WHERE g.nopolis = b.nopolis AND g.prodke = b.prodke AND g.goodsid = a.objectid
             FETCH NEXT 1 ROW ONLY)
       END AS "UserName",
       EXTRACT(YEAR FROM c.begindate) AS "Remark",
       c.begindate     AS "RCV_ID",
       c.enddate       AS "RW",
       b.dateofloss    AS "CountryID",
       b.registerdate  AS "Country",
       a.analyst_tfkomitedate      AS "Email",
       a.acceptance_datecomitee    AS "District",
       a.tglakseptasi  AS "AlasanKlaim",
       (SELECT k.currency FROM POOLDATA.CURRENCY k WHERE k.id = a.currency) AS "NPWP",
       (SELECT SUM(e.estimationvalue) FROM POOLDATA.T_CLAIM_ESTIMASI e
         WHERE e.claimid = a.claimid AND e.estimationtype = '1' AND e.kursid = a.currency) AS "ProdKe",
       a.noakseptasi     AS "DistrictID",
       a.asm_share       AS "ASMShare",
       a.asm_share_value AS "KomiteStatus",
       a.grossvalue      AS "Currency",
       CASE WHEN a.asm_share = 100 THEN 'N' ELSE 'Y' END AS "Status",
       a.asm_share_value * c.prsn_bppdan     AS "LossCoverage",
       a.asm_share_value * c.prsn_fac_out    AS "LokasiSurveyor",
       a.asm_share_value * c.prsn_qs_ri      AS "IsTransferPIC",
       a.asm_share_value * c.prsn_fspl       AS "AnaylstRemarks",
       a.asm_share_value * c.prsn_psrspl     AS "AnalystDoctorRemaks",
       a.asm_share_value * c.prsn_psplnsri   AS "NIK",
       a.asm_share_value * c.prsn_psrqs_ri   AS "BranchID",
       a.asm_share_value * c.prsn_fespl      AS "IsAnalisTransfer",
       a.asm_share_value * c.prsn_fac_obl    AS "BranchName",
       a.asm_share_value * c.prsn_facobsrb   AS "BusinessID",
       a.asm_share_value * c.prsn_facobindt  AS "CASEDB",
       a.asm_share_value * c.prsn_er1        AS "CauseOfLoss",
       a.asm_share_value * c.prsn_er2        AS "CauseOfLossID",
       a.asm_share_value * c.prsn_pss        AS "City",
       a.asm_share_value * c.prsn_prgbi      AS "CityID",
       a.asm_share_value * c.prsn_pfra       AS "ClientID",
       a.asm_share_value * c.prsn_xl         AS "ClientName"
  FROM pooldata.t_claim_adjustment a
  JOIN pooldata.t_claim_pnc b ON a.claimid = b.claimid
  JOIN pooldata.pega_dashboardpnc c ON b.claimno = c.noklaim
  JOIN datapega.pc_asm_fw_gcnmfw_work d ON b.claimid = d.pzinskey
 WHERE a.noakseptasi IS NOT NULL
   AND a.paymenttype IN ('1', '2')
   AND b.branchname <> 'ASNET'
   AND b.grouppanel IN ('003','004','006','009')
   AND b.businesscode NOT IN ('10145','10168','10165','10164','10053','10075','10126','10011','10077','10007')
   AND (d.pystatuswork NOT IN ('Resolved-Completed', 'Resolved-Rejected') OR d.ispendingclose = 'true')
   AND CAST(a.tglakseptasi AS DATE) >= :1
   AND CAST(a.tglakseptasi AS DATE) <= :2
 ORDER BY a.tglakseptasi ASC


-- name: report_os_komite
--
-- REPORT OS KOMITE — 43 kolom rinci, 27 kolom ringkas.
--
-- Asal: `RDB List/GetOSKomiteNonMBU-SQL.xml`.
--
-- Bind:
--   :1  tanggal registrasi dari    DATE
--   :2  tanggal registrasi sampai  DATE
--
-- Yang dicari: adjustment yang SUDAH bernilai tetapi BELUM bernomor akseptasi —
-- `STATUSAKSEPTASI IN ('0','1')` dan `NOAKSEPTASI IS NULL`. Kolom "TKI" menyatakan
-- keduanya: sudah komite bila statusnya '1', menunggu komite bila '0'.
SELECT b.claimno       AS "CaseID",
       b.nopolis       AS "NoKTP",
       b.qqname        AS "ClaimID",
       b.picteknik     AS "Conveyance",
       b.businessname  AS "ClaimNo",
       b.leader_member AS "UserTeknisEmail",
       c.occupationid   AS "UserTeknis",
       c.occupation     AS "UserName",
       b.dateofloss    AS "CountryID",
       b.registerdate  AS "Country",
       a.analyst_tfkomitedate   AS "Email",
       a.acceptance_datecomitee AS "District",
       a.tglakseptasi  AS "AlasanKlaim",
       (SELECT k.currency FROM pooldata.CURRENCY k WHERE k.id = a.currency) AS "NPWP",
       (SELECT SUM(e.estimationvalue) FROM pooldata.T_CLAIM_ESTIMASI e
         WHERE e.claimid = a.claimid AND e.estimationtype = '1' AND e.kursid = a.currency) AS "ProdKe",
       CASE
         WHEN c.leader_member = 'LEADER'
         THEN (SELECT SUM(e.estimationvalue) FROM pooldata.T_CLAIM_ESTIMASI e
                WHERE e.claimid = a.claimid AND e.estimationtype = '1' AND e.kursid = a.currency) - a.grossvalue
         ELSE (SELECT SUM(e.estimationvalue) FROM pooldata.T_CLAIM_ESTIMASI e
                WHERE e.claimid = a.claimid AND e.estimationtype = '1' AND e.kursid = a.currency) - a.asm_share_value
       END AS "FlagASO",
       a.noakseptasi     AS "DistrictID",
       a.asm_share       AS "ASMShare",
       a.grossvalue      AS "Currency",
       a.asm_share_value AS "KomiteStatus",
       a.grossvalue - a.asm_share_value AS "TKA",
       -- Tiga kolom periode polis yang TERLEWAT saat kueri ini dipindahkan, dan
       -- ditemukan uji TestSetiapKolomBerkasPunyaSumber — bukan oleh pembacaan ulang.
       --
       -- Sumbernya punya ketiganya: tahun mulai, tanggal mulai, dan tanggal berakhir
       -- periode polis. Tanpa keduanya kolom UW YEAR, START DATE, dan END DATE kosong
       -- tanpa satu pun tanda. Tahunnya dihitung di Go dari tanggal yang sama.
       c.begindate       AS "RCV_ID",
       c.enddate         AS "RW",
       CASE WHEN a.asm_share = 100 THEN 'N' ELSE 'Y' END AS "Status",
       a.asm_share_value * c.prsn_bppdan     AS "LossCoverage",
       a.asm_share_value * c.prsn_fac_out    AS "LokasiSurveyor",
       a.asm_share_value * c.prsn_qs_ri      AS "IsTransferPIC",
       a.asm_share_value * c.prsn_fspl       AS "AnaylstRemarks",
       a.asm_share_value * c.prsn_psrspl     AS "AnalystDoctorRemaks",
       a.asm_share_value * c.prsn_psplnsri   AS "NIK",
       a.asm_share_value * c.prsn_psrqs_ri   AS "BranchID",
       a.asm_share_value * c.prsn_fespl      AS "IsAnalisTransfer",
       a.asm_share_value * c.prsn_fac_obl    AS "BranchName",
       a.asm_share_value * c.prsn_facobsrb   AS "BusinessID",
       a.asm_share_value * c.prsn_facobindt  AS "CASEDB",
       a.asm_share_value * c.prsn_er1        AS "CauseOfLoss",
       a.asm_share_value * c.prsn_er2        AS "CauseOfLossID",
       a.asm_share_value * c.prsn_pss        AS "City",
       a.asm_share_value * c.prsn_prgbi      AS "CityID",
       a.asm_share_value * c.prsn_pfra       AS "ClientID",
       a.asm_share_value * c.prsn_xl         AS "ClientName",
       CASE
         WHEN a.noakseptasi IS NULL AND a.statusakseptasi = '1' THEN 'SUDAH KOMITE'
         WHEN a.noakseptasi IS NULL AND a.statusakseptasi = '0' THEN 'MENUNGGU KOMITE'
       END AS "TKI"
  FROM pooldata.t_claim_adjustment a
  JOIN pooldata.t_claim_pnc b ON a.claimid = b.claimid
  JOIN pooldata.pega_dashboardpnc c ON b.claimno = c.noklaim
  JOIN datapega.pc_asm_fw_gcnmfw_work d ON b.claimid = d.pzinskey
 WHERE a.statusakseptasi IN ('0','1')
   AND (a.statusakseptasilod IS NULL OR a.statusakseptasilod <> '0')
   AND a.paymenttype IN ('1','2')
   AND a.noakseptasi IS NULL
   AND b.branchname <> 'ASNET'
   AND b.grouppanel IN ('003','004','006','009')
   AND b.businesscode NOT IN ('10145','10168','10165','10164','10053','10075','10126','10011','10077','10007')
   AND (d.pystatuswork NOT IN ('Resolved-Completed', 'Resolved-Rejected') OR d.ispendingclose = 'true')
   AND CAST(b.registerdate AS DATE) >= :1
   AND CAST(b.registerdate AS DATE) <= :2
 ORDER BY b.registerdate DESC


-- name: report_os_belum_komite
--
-- REPORT OS BELUM KOMITE — 36 kolom rinci, 21 kolom ringkas.
--
-- Asal: `RDB List/GetOSBlmKomiteNonMBU-SQL.xml`.
--
-- Bind:
--   :1  tanggal registrasi dari    DATE
--   :2  tanggal registrasi sampai  DATE
--
-- Pembedanya dari OS Komite satu baris, dan artinya besar: `NOT EXISTS` atas
-- `t_claim_adjustment` — klaim ini BELUM punya baris adjustment sama sekali, sehingga
-- belum pernah sampai ke komite.
--
-- Akibatnya nilai yang dikalikan pun berbeda: karena tidak ada akseptasi, yang dipakai
-- adalah ESTIMASI — `shareasm / 100 * TTLOS`, bukan `ASM_SHARE_VALUE`.
SELECT a.claimno       AS "CaseID",
       a.nopolis       AS "NoKTP",
       a.qqname        AS "ClaimID",
       a.picteknik     AS "Conveyance",
       a.businessname  AS "ClaimNo",
       a.leader_member AS "UserTeknisEmail",
       CASE
         WHEN a.grouppanel IN ('003','009') THEN
           (SELECT t.occupationid FROM pooldata.t_anekalist t
             WHERE t.nopolis = a.nopolis AND t.prodke = a.prodke AND t.indexobject = e.objectid
             FETCH NEXT 1 ROW ONLY)
         WHEN a.grouppanel = '006' THEN
           (SELECT cc.OccupationCode
              FROM pooldata.t_propertylist z,
                   JSON_TABLE(z.OCCUPATIONLIST, '$'
                     COLUMNS (NESTED PATH '$.OccupationList[*]'
                              COLUMNS (OccupationCode VARCHAR PATH '$.OccupationCode'))) cc
             WHERE z.nopolis = a.nopolis AND z.indexobject = e.objectid AND z.prodke = a.prodke
             FETCH NEXT 1 ROW ONLY)
         ELSE
           (SELECT g.goodsid FROM pooldata.t_cargolist g
             WHERE g.nopolis = a.nopolis AND g.prodke = a.prodke AND g.goodsid = e.objectid
             FETCH NEXT 1 ROW ONLY)
       END AS "UserTeknis",
       CASE
         WHEN a.grouppanel IN ('003','009') THEN
           (SELECT t.occupationname FROM pooldata.t_anekalist t
             WHERE t.nopolis = a.nopolis AND t.prodke = a.prodke AND t.indexobject = e.objectid
             FETCH NEXT 1 ROW ONLY)
         WHEN a.grouppanel = '006' THEN
           (SELECT dd.OccupationName
              FROM pooldata.t_propertylist x,
                   JSON_TABLE(x.OCCUPATIONLIST, '$'
                     COLUMNS (NESTED PATH '$.OccupationList[*]'
                              COLUMNS (OccupationName VARCHAR PATH '$.OccupationName'))) dd
             WHERE x.nopolis = a.nopolis AND x.indexobject = e.objectid AND x.prodke = a.prodke
             FETCH NEXT 1 ROW ONLY)
         ELSE e.objectname
       END AS "UserName",
       EXTRACT(YEAR FROM b.begindate) AS "Remark",
       b.begindate    AS "RCV_ID",
       b.enddate      AS "RW",
       a.dateofloss   AS "CountryID",
       a.registerdate AS "Country",
       c.currency     AS "NPWP",
       c.ttlos        AS "ProdKe",
       a.shareasm     AS "ASMShare",
       CASE WHEN a.shareasm = 100 THEN 'N' ELSE 'Y' END AS "Status",
       a.shareasm / 100 * c.ttlos AS "KomiteStatus",
       a.shareasm / 100 * c.ttlos * b.prsn_bppdan    AS "LossCoverage",
       a.shareasm / 100 * c.ttlos * b.prsn_fac_out   AS "LokasiSurveyor",
       a.shareasm / 100 * c.ttlos * b.prsn_qs_ri     AS "IsTransferPIC",
       a.shareasm / 100 * c.ttlos * b.prsn_fspl      AS "AnaylstRemarks",
       a.shareasm / 100 * c.ttlos * b.prsn_psrspl    AS "AnalystDoctorRemaks",
       a.shareasm / 100 * c.ttlos * b.prsn_psplnsri  AS "NIK",
       a.shareasm / 100 * c.ttlos * b.prsn_psrqs_ri  AS "BranchID",
       a.shareasm / 100 * c.ttlos * b.prsn_fespl     AS "IsAnalisTransfer",
       a.shareasm / 100 * c.ttlos * b.prsn_fac_obl   AS "BranchName",
       a.shareasm / 100 * c.ttlos * b.prsn_facobsrb  AS "BusinessID",
       a.shareasm / 100 * c.ttlos * b.prsn_facobindt AS "CASEDB",
       a.shareasm / 100 * c.ttlos * b.prsn_er1       AS "CauseOfLoss",
       a.shareasm / 100 * c.ttlos * b.prsn_er2       AS "CauseOfLossID",
       a.shareasm / 100 * c.ttlos * b.prsn_pss       AS "City",
       a.shareasm / 100 * c.ttlos * b.prsn_prgbi     AS "CityID",
       a.shareasm / 100 * c.ttlos * b.prsn_pfra      AS "ClientID",
       a.shareasm / 100 * c.ttlos * b.prsn_xl        AS "ClientName"
  FROM pooldata.t_claim_pnc a
  JOIN pooldata.pega_dashboardpnc b ON a.claimno = b.noklaim
  JOIN pooldata.t_claim_objectlist e ON a.claimid = e.claimid
  JOIN datapega.pc_asm_fw_gcnmfw_work f ON a.claimid = f.pzinskey
  JOIN (SELECT d.claimid,
               SUM(d.estimationvalue) AS ttlos,
               MIN(k.currency)        AS currency
          FROM pooldata.T_CLAIM_ESTIMASI d
          LEFT JOIN pooldata.CURRENCY k ON k.id = d.kursid
         GROUP BY d.claimid) c ON a.claimid = c.claimid
 WHERE a.branchname <> 'ASNET'
   AND a.grouppanel IN ('003','004','006','009')
   AND a.businesscode NOT IN ('10145','10168','10165','10164','10053','10075','10126','10011','10077','10007')
   AND (f.pystatuswork NOT IN ('Resolved-Completed', 'Resolved-Rejected') OR f.ispendingclose = 'true')
   AND NOT EXISTS (SELECT 1 FROM pooldata.t_claim_adjustment adj WHERE adj.claimid = a.claimid)
   AND CAST(a.registerdate AS DATE) >= :1
   AND CAST(a.registerdate AS DATE) <= :2
 ORDER BY a.registerdate DESC


-- name: report_komite
--
-- REPORT DATA KOMITE, susunan ringkas — 11 kolom.
--
-- Asal: `RDB List/ExportDataCloseKlaim-SQL.xml` — kueri yang SAMA dengan panel Close
-- Klaim. `Activity/PNCReportDataKomites_act-Act.xml` memakainya untuk seluruh lini
-- SELAIN Non-MBU; Non-MBU memakai `ExportDataKomitesKlaimNONMBU` yang menghasilkan
-- 80 kolom.
--
-- Bind:
--   :1  tanggal tutup klaim dari    DATE
--   :2  tanggal tutup klaim sampai  DATE
--   :3  kode lini bisnis
--
-- # Kenapa `statusapprove` TIDAK ikut di sini
--
-- Karena kueri aslinya memang tidak menerimanya. Tombol "Export Data Approve" dan
-- "Export Data Rejected" mengirim `statusapprove` "1" atau "2", tetapi nilai itu hanya
-- dipakai pada jalur Non-MBU. Pada jalur ini kedua tombol menghasilkan berkas yang SAMA.
--
-- Itu keanehan sistem lama yang ditiru apa adanya — memberinya penyaring baru berarti
-- mengubah isi berkas yang selama ini diterima, dan itu bukan salah satu dari 13 butir
-- `D-49`.
SELECT a.claimno      AS "CaseID",
       a.nopolis      AS "NoKTP",
       a.qqname       AS "ClaimID",
       a.businessname AS "ClaimNo",
       a.picteknik    AS "Conveyance",
       a.businesscode AS "City",
       a.registerdate AS "Country",
       a.dateofloss   AS "CountryID",
       b.tglakseptasi AS "District",
       b.noakseptasi  AS "DistrictID",
       a.closeclaimdate AS "Email",
       a.closeclaimdate AS "BulanCloseTanggal"
  FROM pooldata.t_claim_pnc a
  JOIN pooldata.t_claim_adjustment b ON a.claimid = b.claimid
 WHERE a.statuswork = 'Resolved-Completed'
   AND CAST(a.closeclaimdate AS DATE) >= :1
   AND CAST(a.closeclaimdate AS DATE) <= :2
   AND (
        :3 = ''
     OR (:3 = '002' AND a.grouppanel = '002')
     OR (:3 = '005' AND a.grouppanel = '005')
     OR (:3 = '346' AND a.grouppanel IN ('003','004','006','009')
                    AND a.businesscode NOT IN ('10008','10010','10015','10023','10168','10145'))
     OR (:3 = '003' AND a.grouppanel = '003'
                    AND a.businesscode IN ('10076','10077','10007','10011','10083','10141','10131','10126','10055','10075'))
       )
 ORDER BY a.closeclaimdate ASC



-- name: report_komite_nonmbu
--
-- REPORT DATA KOMITE, lini Non-MBU — susunan rinci 80 kolom.
--
-- Asal: `RDB List/ExportDataKomitesKlaimNONMBU-SQL.xml`, dijalankan
-- `Activity/PNCReportDataKomites_act-Act.xml`.
--
-- Bind:
--   :1  tanggal tutup klaim dari    DATE
--   :2  tanggal tutup klaim sampai  DATE
--   :3  status approve komite       '1' Approved | '2' Rejected
--
-- Berbeda dari jalur lini lain, di sini `statusapprove` BENAR-BENAR menyaring — kedua
-- tombol menghasilkan berkas yang berbeda. Lihat catatan pada report_komite.
--
-- Lini bisnisnya tidak diikat; penyaringnya tertanam sebagai `grouppanel IN (…)` ditambah
-- pengecualian `businesscode`, persis seperti sumbernya.
--
-- # Yang berubah bentuk
--
--  1. **Dua pemanggilan `get_working_hours@ASMD` dihapus.** Keduanya bukan sub-query
--     melainkan pemanggilan fungsi per baris, dan `D-50` menetapkan logikanya ditulis
--     ulang di Go. Yang dikembalikan di sini adalah PASANGAN TANGGALNYA; jumlah hari
--     kerjanya dihitung reportklaim.WorkingDaysBetween atas kalender libur yang dibaca
--     sekali per laporan lewat koneksi kedua.
--
--     Kesetaraannya TIDAK dapat dibuktikan dari export: `GET_WORKING_HOURS` hidup di
--     basis data ASMD dan source-nya tidak ada di `Database/`. Yang diketahui hanyalah
--     satuannya — hasilnya dibagi 28800, yaitu delapan jam. Perhitungan di Go menghitung
--     HARI PENUH, sehingga hari kerja sebagian dapat berbeda satu. Ini di luar 13 butir
--     `D-49` dan menunggu persetujuan Work Owner (`D-54`).
--
--  2. **`GET_POSISI_PROGRESS2` tidak dipanggil** — CLOB JSON-nya dikembalikan mentah dan
--     diuraikan di Go. Sama dengan report_close_klaim_nonmbu.
--
--  3. **`rownum = 1` menjadi `FETCH FIRST 1 ROW ONLY`**, padanan wajib
--     `09-DATABASE-STRATEGY.md` §4. Keduanya sama-sama mengambil baris sembarang: tidak
--     ada `ORDER BY` pada anak-kueri itu, di sumbernya maupun di sini.
--
--  4. **Tiga alias yang salah ketik TIDAK diperbaiki**, dan ketiga kolomnya tetap kosong:
--
--         alias SQL           properti berkas    judul kolom
--         "month CopyFrom"    CopyFrom           "BULAN CLOSE"
--         "year CreateFrom"   CreateFrom         "TAHUN CLOSE"
--         "KomiteAccepted"    Note Komite        "KomiteAccepted"
--
--     Ketiganya sempat diselaraskan supaya terisi. Penyelarasan itu DICABUT: Work Owner
--     menetapkan 2026-09-25 bahwa cacat yang berasal dari Pega dibiarkan seperti Pega —
--     yang diperbaiki hanya cacat pemindahan.
--
--     Yang berubah dari sistem lama karena itu bukan isinya, melainkan bahwa ketiadaannya
--     kini TERCATAT dan terkunci uji (lihat kolomTanpaSumber). Sebelumnya tidak ada apa pun
--     yang menyatakan ketiga kolom itu memang tidak akan pernah terisi.
--
--  5. **Enam alias yang TIDAK dipakai berkas dihapus** — `LDR_NOTE`, `LDR_ID`,
--     `LOSS_ADJUSTER_FEE` (dua kali, nama yang sama), `ReceiverClaim`, `StatusReceiver`.
--     Tidak ada satu pun properti CSV yang menunjuknya; menghitungnya berarti membayar
--     dua anak-kueri berkorelasi per baris untuk nilai yang dibuang.
--
-- # Dua kolom yang memang tidak punya sumber
--
-- `Location` (judul "OR ASM") dan `NoteKasir` (judul "CLOSE CLAIM NOTE") tidak punya satu
-- pun ekspresi di kueri aslinya. Keduanya SELALU kosong di berkas lama, dan dibiarkan
-- kosong di sini — menebak `PRSN_OR` sebagai sumber "OR ASM" adalah karangan, bukan port.
SELECT b.claimno                                               AS "CaseID",
       b.nopolis                                               AS "NoKTP",
       b.qqname                                                AS "ClaimID",
       b.picteknik                                             AS "ConsultantName",
       (SELECT bg.businessgroupname
          FROM pooldata.business bg
         WHERE bg.id = b.businesscode)                         AS "ClaimNo",
       b.businessname                                          AS "Resources",
       b.sobname                                               AS "IsPLA",
       c.nama_mo                                               AS "IsSendPremi",
       b.leader_member                                         AS "UserTeknisEmail",
       b.norefbroker                                           AS "NoReffBroker",
       b.sts_banding                                           AS "Banding",
       b.sts_kepuasan                                          AS "SurveyKepuasan",
       b.effort_close                                          AS "EffortClose",
       b.kendala_close                                         AS "KendalaClose",
       b.sts_paperless                                         AS "Paperless",
       b.usulan                                                AS "Usulan",
       (SELECT oc.causeofloss
          FROM pooldata.t_claim_objectcoverage oc
         WHERE oc.claimid = b.claimid
           AND oc.objectid = a.objectid
           AND oc.objectcoverageid = a.objectcoverageid)       AS "InsuredName",
       (SELECT oc.coveragename
          FROM pooldata.t_claim_objectcoverage oc
         WHERE oc.claimid = b.claimid
           AND oc.objectid = a.objectid
           AND oc.objectcoverageid = a.objectcoverageid)       AS "TelpTertanggung",
       CASE WHEN b.exgratia = '1' THEN 'YES' ELSE 'NO' END     AS "AnalystRemaksInvestigator",
       CASE
            WHEN b.grouppanel IN ('003','009')
            THEN (SELECT an.occupationid
                    FROM pooldata.t_anekalist an
                   WHERE an.nopolis = b.nopolis
                     AND an.prodke = b.prodke
                     AND an.indexobject = a.objectid
                   FETCH FIRST 1 ROW ONLY)
            WHEN b.grouppanel = '006'
            THEN (SELECT cc.occupationcode
                    FROM pooldata.t_propertylist pl,
                         JSON_TABLE (pl.occupationlist, '$'
                             COLUMNS (NESTED PATH '$.OccupationList[*]'
                                 COLUMNS (occupationcode VARCHAR(60) PATH '$.OccupationCode'))) cc
                   WHERE pl.nopolis = b.nopolis
                     AND pl.indexobject = a.objectid
                     AND pl.prodke = b.prodke
                   FETCH FIRST 1 ROW ONLY)
            ELSE (SELECT cg.goodsid
                    FROM pooldata.t_cargolist cg
                   WHERE cg.nopolis = b.nopolis
                     AND cg.prodke = b.prodke
                     AND cg.goodsid = a.objectid
                   FETCH FIRST 1 ROW ONLY)
       END                                                     AS "UserTeknis",
       CASE
            WHEN b.grouppanel IN ('003','009')
            THEN (SELECT an.occupationname
                    FROM pooldata.t_anekalist an
                   WHERE an.nopolis = b.nopolis
                     AND an.prodke = b.prodke
                     AND an.indexobject = a.objectid
                   FETCH FIRST 1 ROW ONLY)
            WHEN b.grouppanel = '006'
            THEN (SELECT dd.occupationname
                    FROM pooldata.t_propertylist pl,
                         JSON_TABLE (pl.occupationlist, '$'
                             COLUMNS (NESTED PATH '$.OccupationList[*]'
                                 COLUMNS (occupationname VARCHAR(200) PATH '$.OccupationName'))) dd
                   WHERE pl.nopolis = b.nopolis
                     AND pl.indexobject = a.objectid
                     AND pl.prodke = b.prodke
                   FETCH FIRST 1 ROW ONLY)
            ELSE (SELECT cg.goodsname
                    FROM pooldata.t_cargolist cg
                   WHERE cg.nopolis = b.nopolis
                     AND cg.prodke = b.prodke
                     AND cg.goodsid = a.objectid
                   FETCH FIRST 1 ROW ONLY)
       END                                                     AS "UserName",
       c.begindate                                             AS "RCV_ID",
       c.enddate                                               AS "RW",
       b.dateofloss                                            AS "CountryID",
       b.registerdate                                          AS "Country",
       a.analyst_tfkomitedate                                  AS "Email",
       a.acceptance_datecomitee                                AS "District",
       a.tglakseptasi                                          AS "AlasanKlaim",
       d.closeclaimdate_1                                      AS "Conveyance",
       b.analyst_transferdate                                  AS "EmailTertanggung",
       b.closeclaimdate                                        AS "TanggalCloseUntukTAT",
       a.receivedatelod                                        AS "TanggalTerimaLOD",
       a.transfer_cashier_date                                 AS "TanggalTransferKasir",
       CASE WHEN d.surveyortype_1 IN ('2','3','4')
            THEN d.adjusterpic_1 ELSE '' END                   AS "FlagNOLL",
       CASE WHEN d.surveyortype_1 = '1' AND b.businessname = 'MARINE HULL'
            THEN d.surveyorname_1
            WHEN d.surveyortype_1 = '2' AND b.businessname = 'MARINE HULL'
            THEN d.surveyornamemarine_1
            ELSE '' END                                        AS "FlagReject",
       (SELECT cu.currency
          FROM pooldata.currency cu
         WHERE cu.id = a.currency)                             AS "NPWP",
       CASE
            WHEN a.paymenttype IN ('1','2','5')
            THEN (SELECT SUM(es.estimationvalue)
                    FROM pooldata.t_claim_estimasi es
                   WHERE es.claimid = a.claimid
                     AND es.estimationtype = '1'
                     AND es.kursid = a.currency
                     AND es.objectid = a.objectid
                     AND es.objectcoverageid = a.objectcoverageid
                     AND es.cfsdate <= CAST(a.analyst_tfkomitedate AS DATE))
            WHEN a.paymenttype = '4'
            THEN (SELECT SUM(es.estimationvalue)
                    FROM pooldata.t_claim_estimasi es
                   WHERE es.claimid = a.claimid
                     AND es.estimationtype = '2'
                     AND es.kursid = a.currency
                     AND es.objectid = a.objectid
                     AND es.objectcoverageid = a.objectcoverageid
                     AND es.cfsdate <= CAST(a.analyst_tfkomitedate AS DATE))
            ELSE 0
       END                                                     AS "ProdKe",
       a.noakseptasi                                           AS "DistrictID",
       a.asm_share                                             AS "ASMShare",
       a.asm_share_value                                       AS "KomiteStatus",
       a.propose_value                                         AS "OldNoHp",
       CASE a.paymenttype WHEN '1' THEN 'Final'
                          WHEN '2' THEN 'Interim'
                          WHEN '3' THEN 'Salvage'
                          WHEN '4' THEN 'Adjuster Fee'
                          WHEN '5' THEN 'Adjustment'
                          WHEN '6' THEN 'Tolak Klaim' END      AS "IBNR",
       a.grossvalue                                            AS "Currency",
       CASE WHEN a.asm_share = 100 THEN 'N' ELSE 'Y' END       AS "Status",
       a.asm_share_value * c.prsn_bppdan                       AS "LossCoverage",
       a.asm_share_value * c.prsn_fac_out                      AS "LokasiSurveyor",
       a.asm_share_value * c.prsn_qs_ri                        AS "IsTransferPIC",
       a.asm_share_value * c.prsn_fspl                         AS "AnaylstRemarks",
       a.asm_share_value * c.prsn_psrspl                       AS "AnalystDoctorRemaks",
       a.asm_share_value * c.prsn_psplnsri                     AS "NIK",
       a.asm_share_value * c.prsn_psrqs_ri                     AS "BranchID",
       a.asm_share_value * c.prsn_fespl                        AS "IsAnalisTransfer",
       a.asm_share_value * c.prsn_fac_obl                      AS "BranchName",
       a.asm_share_value * c.prsn_facobsrb                     AS "BusinessID",
       a.asm_share_value * c.prsn_facobindt                    AS "CASEDB",
       a.asm_share_value * c.prsn_er1                          AS "CauseOfLoss",
       a.asm_share_value * c.prsn_er2                          AS "CauseOfLossID",
       a.asm_share_value * c.prsn_pss                          AS "City",
       a.asm_share_value * c.prsn_prgbi                        AS "CityID",
       a.asm_share_value * c.prsn_pfra                         AS "ClientID",
       a.asm_share_value * c.prsn_xl                           AS "ClientName",
       CASE
            WHEN a.paymenttype IN ('1','2','5')
            THEN (SELECT SUM(es.estimationvalue)
                    FROM pooldata.t_claim_estimasi es
                   WHERE es.claimid = a.claimid
                     AND es.objectid = a.objectid
                     AND es.objectcoverageid = a.objectcoverageid
                     AND es.estimationtype = '1'
                     AND es.kursid = a.currency
                     AND es.cfsdate <= CAST(a.analyst_tfkomitedate AS DATE)) - a.nilaiakseptasi
            WHEN a.paymenttype = '4'
            THEN (SELECT SUM(es.estimationvalue)
                    FROM pooldata.t_claim_estimasi es
                   WHERE es.claimid = a.claimid
                     AND es.objectid = a.objectid
                     AND es.objectcoverageid = a.objectcoverageid
                     AND es.estimationtype = '2'
                     AND es.kursid = a.currency
                     AND es.cfsdate <= CAST(a.analyst_tfkomitedate AS DATE)) - a.nilaiakseptasi
            ELSE 0
       END                                                     AS "IDMaster",
       c.tsi                                                   AS "IdxSurveyResults",
       b.leader_member                                         AS "DaftarObjek",
       p.sts_progress1                                         AS "AgingAmount",
       p.jsonstatus_progress2                                  AS "ProgresJSON",
       p.keterangan                                            AS "pyNote",
       b.coinsname                                             AS "OwnRisk",
       (SELECT mp.note_st
          FROM pooldata.mst_penolakan_klaim_2 mp
         WHERE mp.id_st = a.idpenolakanst
           AND mp.id_nd = a.idpenolakandua)                    AS "KodeCabang",
       (SELECT mp.note_nd
          FROM pooldata.mst_penolakan_klaim_2 mp
         WHERE mp.id_st = a.idpenolakanst
           AND mp.id_nd = a.idpenolakandua)                    AS "KodeKBRU",
       CASE k.statusapprove WHEN '1' THEN 'Approved'
                            WHEN '2' THEN 'Rejected'
                            ELSE 'Waiting' END                 AS "StatusApprove",
       k.komite_id                                             AS "ClaimNoExt",
       k.namakomite                                            AS "NamaPrincipal",
       -- Properti berkasnya bernama `Note Komite` — dengan SPASI, yang bukan nama
       -- properti Pega yang sah — sementara aliasnya `KomiteAccepted`. Keduanya tidak
       -- pernah bertemu, dan kolom berjudul "KomiteAccepted" selalu kosong.
       --
       -- Dibiarkan apa adanya (keputusan Work Owner 2026-09-25).
       k.notekomite                                            AS "KomiteAccepted"
  FROM pooldata.t_claim_adjustment a
  JOIN pooldata.t_claim_pnc b            ON a.claimid = b.claimid
  JOIN pooldata.pega_dashboardpnc c      ON b.claimno = c.noklaim
  JOIN datapega.pc_asm_fw_gcnmfw_work d  ON b.claimid = d.pzinskey
  JOIN pooldata.t_claim_komite_list k    ON k.no_klaim = d.pyid
                                        AND k.komite_id = a.caseidkomite
  LEFT JOIN pooldata.gcnm_progress_claim p
         ON p.pnccaseid = c.noklaim
        AND p.id_update = (SELECT MAX(m.id_update)
                             FROM pooldata.gcnm_progress_claim m
                            WHERE m.pnccaseid = c.noklaim
                              AND m.status_progress2 NOT IN ('2','24','60','59'))
 WHERE b.branchname <> 'ASNET'
   AND b.grouppanel IN ('003','004','006','009')
   AND b.businesscode NOT IN ('10145','10168','10165','10164','10053',
                              '10075','10126','10011','10077','10007')
   AND k.komite_id IN (SELECT f.komite_id
                         FROM pooldata.t_claim_komite_list f
                        WHERE f.statusapprove = :3
                          AND f.no_klaim = d.pyid)
   AND CAST(d.closeclaimdate_1 AS DATE) >= :1
   AND CAST(d.closeclaimdate_1 AS DATE) <= :2
 ORDER BY d.closeclaimdate_1 ASC
