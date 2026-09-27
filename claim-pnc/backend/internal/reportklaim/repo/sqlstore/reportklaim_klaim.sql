-- Kueri laporan kelompok Klaim.
--
-- Kelima aturan yang mengikat berkas kueri modul ini disebut di
-- reportklaim_reasuransi.sql dan berlaku sama di sini.
--
-- ============================================================================
-- Satu kueri melayani beberapa susunan kolom
-- ============================================================================
--
-- Laporan TAT punya TIGA susunan kolom (52 / 39 / 34) menurut lini bisnisnya, dan
-- laporan Close punya dua. Pemeriksaan terhadap export menunjukkan pemetaan
-- properti → kolom sumbernya **sama persis di seluruh varian** — yang berbeda hanya
-- properti mana yang ikut ke berkas.
--
-- Karena itu satu kueri mengembalikan GABUNGAN seluruh properti, dan katalog yang
-- memilih mana yang ditulis ke CSV. Alternatifnya tiga kueri yang 90% sama, dan satu
-- yang tertinggal saat diubah akan membuat satu lini bisnis memakai kolom sumber yang
-- berbeda dari lini lain — tanpa satu pun tanda.


-- name: report_tat
--
-- REPORT TAT — 52 / 39 / 34 kolom menurut lini bisnis; 71 properti seluruhnya.
--
-- Asal: `RDB List/BroswseKlaimByRegisterDate-SQL.xml`, dijalankan
-- `Activity/PNCTATReport1_Act-Act.xml`, ditambah `BroswsePLA_SQL` dan `BroswseDLA_SQL`
-- yang di sistem lama dijalankan SEKALI PER KLAIM dan di sini menjadi anak-kueri.
--
-- Bind:
--   :1  tanggal registrasi dari    DATE
--   :2  tanggal registrasi sampai  DATE
--   :3  kode lini bisnis           '' | '002' | '005' | '346' | '003'
--
-- Rentangnya atas TANGGAL REGISTRASI — nama rule aslinya menyebutkannya apa adanya,
-- "ByRegisterDate".
--
-- # DB Link yang DIPERTAHANKAN, dan kenapa hanya yang ini
--
-- `POLICYRANGE` membaca `collection.mst_det_sales@ASMD` sebagai **sub-query**. Keputusan
-- Work Owner 2026-09-24: yang berupa sub-query tetap memakai DB Link; selain itu memakai
-- koneksi langsung (`ANEKA_<PORTAL_ALIAS>_*`). Lihat keputusan-implementasi.md §49.
--
-- # Empat kolom yang SENGAJA kosong di sini
--
-- `CityID`, `CloseClaimNote`, `Conveyance`, dan `OccupationCode` — keempatnya kolom
-- "Lama proses" yang di sistem lama berisi `Param.CountBusiness`, hasil
-- `GCNMTimeDifferenceWorkCalender_Act`. Perhitungannya menuntut kalender libur, dan
-- `D-50` menetapkan logikanya ditulis ulang di Go karena ia aturan bisnis — bukan
-- pengambilan data. Kueri ini menyediakan BAHANNYA (tanggal-tanggalnya); yang menghitung
-- adalah lapisan Go.
--
-- # Dua kolom yang tidak punya sumber tabel sama sekali
--
-- `Status` berasal dari `.Notes` pada daftar adjustment objek kerja Pega — properti
-- klipboard yang TIDAK dioptimasi menjadi kolom. Pencarian ke seluruh `RDB List/` dan
-- `Database/` menemukan `NOTES` hanya pada T_PLALIST dan T_DLALIST, bukan pada
-- T_CLAIM_ADJUSTMENT. `isCompliance` tidak dipetakan sama sekali oleh activity-nya.
--
-- Keduanya dikembalikan sebagai NULL dan menjadi sel kosong — bukan diisi nilai yang
-- kebetulan mirip.
SELECT a.claimno                        AS "CaseID",
       a.nopolis                        AS "ClaimNo",
       REPLACE(a.qqname, ',', ' ')      AS "LossCoverage",
       a.businessname                   AS "AlasanDokterRejectRCL",
       REPLACE(a.sobname, ',', ' ')     AS "DollarCurrencyVal",
       a.leader_member                  AS "NamaDokterRCL",
       d.noakseptasi                    AS "IsAnalisTransfer",
       EXTRACT(YEAR FROM a.dateofloss)  AS "CauseOfLoss",
       a.dateofloss                     AS "DaftarObjek",
       a.dateofloss                     AS "DateKomite",
       a.registerdate                   AS "City",
       a.finishregisterdate             AS "Location",
       a.finishregisterdate             AS "AnalystRemaksInvestigator",
       a.dateofrequestdocument          AS "isComplianceTransfer",
       a.surveydate                     AS "UserAdmin",
       a.picteknik                      AS "NoKTP",
       a.closeclaimdate                 AS "ExGratiaNote",
       REPLACE(REPLACE(a.closeclaimnote, CHR(10), ' '), ',', ' ') AS "NIK",
       a.alasanketerlambatan            AS "Remark",
       a.transferpic_date               AS "NewNoKTP",
       a.tgldoklengkap                  AS "pyGroup",
       a.location                       AS "RiskLocation",
       a.investigator_tf_date           AS "IdxSurveyResults",
       a.analysttorclpucl_date          AS "ClaimEstimate",
       a.rclpucl_tf_toanalyst           AS "AlasanTerlambat",
       a.compliance_createdate          AS "Province",
       a.cplvalid_date                  AS "IsCFS_PNC",
       a.postaudit_tf_analystdate       AS "IsBackCFS",
       a.cplpostaudit_validdate         AS "IsTransferPIC",
       COALESCE(a.cplvalid_date, a.postaudit_tf_analystdate) AS "CompliancePosAuditByr",
       a.analysttransferdate            AS "KirimAnalystDate",
       a.trf_to_investigator            AS "KirimInvestDate",

       d.acceptance_datecomitee         AS "AnaylstRemarks",
       d.acceptance_datecomitee         AS "HideKTP",
       d.manualacceptancedatecomitee    AS "LokasiSurveyor",
       d.tglakseptasi                   AS "CountryID",
       d.tglakseptasi                   AS "ComplianceRemark",
       d.receivedatelod                 AS "ResponseNote",
       d.manual_acceptancedate          AS "IsTransferAnalisator",
       d.status                         AS "DistrictID",
       d.printlod_date                  AS "StatusWork",
       d.analyst_tfkomitedate           AS "pyCountryName",
       d.transfer_cashier_date          AS "NewEmail",
       d.transfer_cashier_date          AS "pyEmailAddress",
       d.grossvalue                     AS "FlagReject",
       CASE WHEN d.tglbayar IS NULL THEN d.tglakseptasi ELSE d.tglbayar END AS "UserBusinessPA",
       -- Kolom bantu: tanggal bayar yang belum disatukan dengan tanggal akseptasi.
       -- Ia dipakai menghitung "Lama akseptasi - tanggal bayar" dan TIDAK ikut ke berkas;
       -- yang di atasnya sudah tergabung sehingga tidak dapat dipakai berhitung.
       d.tglbayar                       AS "TanggalBayar",

       g.causeofloss                    AS "NatureOfLoss",
       g.coveragename                   AS "UserTeknisGroup",

       x.pnccaseid                      AS "RefNo",
       x.pxcreateoperator               AS "pyLabel",

       -- Keduanya teks berbentuk YYYYMMDD, bukan tanggal. Ia dikembalikan apa adanya
       -- dan disusun ulang menjadi dd/mm/yyyy di Go — sama seperti sistem lama yang
       -- memotongnya dengan @substring, bukan dengan fungsi tanggal.
       x.receiveddate_1                 AS "TanggalTerimaDokumenTeks",
       SUBSTR(x.reportdate_1, 1, 8)     AS "TanggalLaporanTeks",

       -- Tanggal lahir berasal dari daftar peserta, bukan dari klaim.
       --
       -- USIA TIDAK dihitung di sini. Kueri aslinya memakai
       -- `TRUNC(MONTHS_BETWEEN(TRUNC(dateofloss), dob) / 12)`, dan `MONTHS_BETWEEN`
       -- termasuk padanan wajib `09-DATABASE-STRATEGY.md` §4 yang harus dihitung di Go.
       -- Bahannya — tanggal lahir dan tanggal kejadian — keduanya sudah ada di baris ini.
       TO_DATE(tobj.asmdateofbirth, 'YYYYMMDD') AS "DOB",

       (SELECT COALESCE(jt.Diagnose, jt.DescDiagnose)
          FROM JSON_KLAIM jk,
               JSON_TABLE(jk.DATA_JSONBLOB, '$.ObjectList[*]'
                 COLUMNS (ObjectID VARCHAR PATH '$.ObjectID',
                          NESTED PATH '$.ObjectCoverageList[*]'
                          COLUMNS (Diagnose VARCHAR PATH '$.Diagnose',
                                   CoverageID VARCHAR PATH '$.CoverageID',
                                   DescDiagnose VARCHAR PATH '$.DescDiagnose'))) jt
         WHERE jk.IDPEGA = a.claimid
           AND jt.ObjectID = d.objectid
           AND d.objectcoverageid = jt.CoverageID) AS "Diagnose",

       -- Sub-query lewat DB Link — DIPERTAHANKAN, lihat keterangan di atas.
       (SELECT TO_CHAR(ROUND(MONTHS_BETWEEN(TRUNC(a.dateofloss),
                                            MAX(TRUNC(mds.mds_beg_date)) KEEP (DENSE_RANK LAST ORDER BY mds.mds_prod_ke)), 2),
                       '9999D99', 'NLS_NUMERIC_CHARACTERS = '',.''')
          FROM collection.mst_det_sales@ASMD.SINARMAS.CO.ID mds
         WHERE mds.mds_no_polis = a.nopolis) || ' Bulan' AS "PolicyRange",

       (SELECT v.lsc_note FROM v_sts_claim v WHERE v.lsc_id = a.statusclaim) AS "AnalystDoctorRemaks",

       (SELECT z.posisi
          FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC z
         WHERE z.caseid = a.claimno
         ORDER BY z.id DESC
         FETCH NEXT 1 ROW ONLY) AS "TelpTertanggung",

       (SELECT t.startdate FROM t_general t WHERE t.nopolis = a.nopolis AND t.prodke = a.prodke
         FETCH NEXT 1 ROW ONLY) AS "ClaimNoSRB",
       (SELECT t.enddate FROM t_general t WHERE t.nopolis = a.nopolis AND t.prodke = a.prodke
         FETCH NEXT 1 ROW ONLY) AS "NoPla",

       -- Di sistem lama keduanya kueri tersendiri yang dijalankan sekali per klaim.
       (SELECT p.tglkirim FROM POOLDATA.T_PLALIST p WHERE p.claimid = a.claimid
         ORDER BY p.tglkirim DESC FETCH NEXT 1 ROW ONLY) AS "InsuredRelationshipOthers",
       (SELECT l.tglkirim FROM POOLDATA.T_DLALIST l WHERE l.claimid = a.claimid
         ORDER BY l.tglkirim DESC FETCH NEXT 1 ROW ONLY) AS "RCV_ID",

       -- # Empat kolom yang sempat dikira nilai tetap
       --
       -- Ketiganya di bawah pernah ditulis sebagai konstanta `'RCL'`, `'0'`, dan
       -- `'Final'` karena begitulah bentuknya pada langkah PERTAMA activity-nya. Itu
       -- keliru: activity yang sama memuat langkah-langkah BERIKUTNYA yang menimpanya
       -- dengan nilai lain, masing-masing berprakondisi. Yang terbaca sebagai konstanta
       -- sebenarnya hanyalah cabang pertama dari sebuah percabangan.
       --
       -- Rantai pemetaannya dibaca dari export, dan kedua penghubungnya bernama
       -- MENYESATKAN — persis pola alias yang `03-CURRENT-ARCHITECTURE.md` §4.2 sebut:
       --
       --     Local.TipeBayar = .MARKETINGNAME  ← `d.paymenttype as "MARKETINGNAME"`
       --     Local.RCLPUCL   = .SOBNAME        ← `a.RCLPUCL      as "SOBNAME"`
       --
       -- Di sini keduanya dipetakan langsung di SQL, sehingga tidak ada lagi nama antara
       -- yang harus ditelusuri.
       CASE a.rclpucl
            WHEN '1' THEN 'RCL'
            WHEN '2' THEN 'PUCL'
            WHEN '3' THEN 'Notifikasi'
       END AS "ExGratia",
       CASE d.paymenttype
            WHEN '1' THEN 'Final'
            WHEN '2' THEN 'Interim'
            WHEN '3' THEN 'Salvage'
            WHEN '4' THEN 'Adjuster Fee'
            WHEN '5' THEN 'Adjustment'
       END AS "ReporterName",

       -- Dua kolom penanda 1/0. Sistem lama menyetelnya dari langkah berprakondisi
       -- `…KirimInvestDate == ""` dan `…IsCFS_PNC == ""` — yaitu "tanggalnya kosong".
       -- Keduanya karena itu diturunkan dari kolom tanggal yang SAMA dengan yang dipakai
       -- prakondisi itu, bukan dari kolom lain yang kebetulan mirip namanya.
       CASE WHEN a.trf_to_investigator IS NULL THEN '0' ELSE '1' END AS "IsInvest",
       CASE WHEN a.cplvalid_date IS NULL THEN '0' ELSE '1' END       AS "isCompliance"
  FROM T_CLAIM_PNC a
  JOIN t_claim_adjustment d ON d.claimid = a.claimid
  JOIN t_claim_objectcoverage g ON g.claimid = a.claimid AND d.objectcoverageid = g.objectcoverageid
  JOIN datapega.pc_asm_fw_gcnmfw_work x ON x.pzinskey = a.claimid AND x.pxobjclass = 'ASM-FW-GCNMFW-Work-PNC'
  JOIN pooldata.t_personlist tobj ON tobj.nopolis = a.nopolis AND tobj.prodke = a.prodke AND tobj.indexobject = d.objectid
 WHERE CAST(a.registerdate AS DATE) >= :1
   AND CAST(a.registerdate AS DATE) <= :2
   AND a.branchcode <> '100639'
   AND (
        :3 = ''
     OR (:3 = '002' AND a.grouppanel = '002')
     OR (:3 = '005' AND a.grouppanel = '005')
     OR (:3 = '346' AND a.grouppanel IN ('003','004','006','009')
                    AND a.businesscode NOT IN ('10145','10168','10165','10164','10053','10075','10126','10011','10077','10007'))
     OR (:3 = '003' AND a.grouppanel = '003'
                    AND a.businesscode IN ('10076','10077','10007','10011','10083','10141','10131','10126','10055','10075',
                                           '10061','10053','10143','10145','10165','10164','10168','10174','10189','10177','10190'))
       )
 ORDER BY a.claimid ASC


-- name: report_klaim_harian
--
-- REPORT KLAIM HARIAN — 30 kolom.
--
-- Asal: `RDB List/BroswseKlaimperday-SQL.xml`, dijalankan
-- `Activity/PNCReportHarian_act1-Act.xml`.
--
-- Bind:
--   :1  tanggal registrasi dari    DATE
--   :2  tanggal registrasi sampai  DATE
--   :3  kode lini bisnis
--
-- # Enam anak-kueri agregat menjadi satu
--
-- Kueri aslinya membaca `t_claim_adjustment` DELAPAN KALI untuk satu klaim — delapan
-- anak-kueri agregat atas tabel dan penyaring yang sama, berbeda hanya rumusnya. Di sini
-- kedelapannya dihitung dalam satu pembacaan lewat gabungan teragregasi.
--
-- Hasil tiap rumus TIDAK diubah. Yang hilang hanya tujuh kali pembacaan tabel yang di
-- laporan harian dijalankan untuk setiap baris.
SELECT b.nopolis    AS "NoKTP",
       b.claimno    AS "CaseID",
       b.picteknik  AS "UserAdmin",
       b.transferpic_date AS "TransferPICDate",
       b.registerdate     AS "AnalystTransferDate",
       b.branchname AS "AnalystRemaksInvestigator",
       b.dateofloss AS "DateOfLoss",
       EXTRACT(MONTH FROM b.registerdate) AS "Email",
       b.qqname     AS "NamaSurveyor",
       b.sobname    AS "CityID",
       b.remarkrecomendation AS "AnaylstRemarks",
       b.businessname AS "City",
       b.kronologi  AS "LokasiSurveyor",
       b.norefbroker AS "NoReffBroker",
       CASE
         WHEN b.statuswork = 'New'                THEN 'Claim On Progress'
         WHEN b.statuswork = 'Resolved-Completed' THEN 'Closed'
         WHEN b.statuswork = 'Resolved-Rejected'  THEN 'Ditolak'
         ELSE '-'
       END AS "ClaimNo",
       CASE
         WHEN b.typeofcoins = '2' THEN 'Member'
         WHEN b.typeofcoins = 'F' THEN 'Fac In'
         WHEN b.typeofcoins = '1' THEN 'Leader'
         ELSE '-'
       END AS "AgingAmount",
       (SELECT MAX(c.causeofloss) FROM t_claim_objectcoverage c WHERE c.claimid = b.claimid) AS "Country",
       (SELECT w.receiveddate_1 FROM datapega.pc_asm_fw_gcnmfw_work w WHERE w.pzinskey = b.claimid) AS "NIK",
       (SELECT MIN(o.tanggal) FROM os_akseptasi_klaim o WHERE o.caseid = b.claimid) AS "NPWP",
       (SELECT w.pxcreateopname FROM datapega.pc_asm_fw_gcnmfw_work w WHERE w.pyid = b.claimno) AS "NewEmail",
       adj.share_max          AS "AlasanTerlambat",
       adj.total_claim        AS "ComplianceRemark",
       adj.gross              AS "Conveyance",
       adj.individual_risk    AS "CloseClaimNote",
       adj.total_share_asm    AS "CommentKomiteClosecase",
       adj.loc_dan_salvage    AS "CompliancePosAuditByr",
       adj.nilai_bersih       AS "UserName",
       adj.nilai_bersih * (adj.share_max / 100) AS "UserTeknis"
  FROM POOLDATA.T_CLAIM_PNC b
  LEFT JOIN (
        SELECT claimid,
               MAX(asm_share)                                   AS share_max,
               SUM(total_claim * currencyvalue)                  AS total_claim,
               SUM(grossvalue * currencyvalue)                   AS gross,
               SUM(individual_risk_value * currencyvalue)        AS individual_risk,
               SUM((total_claim * currencyvalue) * (asm_share / 100)) AS total_share_asm,
               SUM((COALESCE(loc, 0) / 100) * total_claim * currencyvalue)
                 + SUM(COALESCE(nilai_salvage_a * currencyvalue, 0))  AS loc_dan_salvage,
               SUM(total_claim * currencyvalue)
                 - SUM(individual_risk_value * currencyvalue)
                 - SUM((COALESCE(loc, 0) / 100) * total_claim * currencyvalue)
                 + SUM(COALESCE(nilai_salvage_a * currencyvalue, 0))
                 - SUM(grossvalue * currencyvalue)                    AS nilai_bersih
          FROM pooldata.t_claim_adjustment
         GROUP BY claimid
       ) adj ON adj.claimid = b.claimid
 WHERE CAST(b.registerdate AS DATE) >= :1
   AND CAST(b.registerdate AS DATE) <= :2
   AND b.claimno IS NOT NULL
   AND b.branchname <> 'ASNET'
   AND (
        :3 = ''
     OR (:3 = '002' AND b.grouppanel = '002')
     OR (:3 = '005' AND b.grouppanel = '005')
     OR (:3 = '346' AND b.grouppanel IN ('003','004','006','009'))
     OR (:3 = '003' AND b.grouppanel = '003')
       )
 ORDER BY b.registerdate ASC


-- name: report_reject_klaim
--
-- REPORT DATA REJECT KLAIM — 33 kolom.
--
-- Asal: `RDB List/ExportDataRejectKlaim-SQL.xml`.
--
-- Bind:
--   :1  tanggal tutup klaim dari    DATE
--   :2  tanggal tutup klaim sampai  DATE
--   :3  kode lini bisnis
--
-- Kolom "Bulan Close" di kueri asli adalah `SUBSTR(d.closeclaimdate, 4, 3)` — memotong
-- tiga huruf dari tanggal yang diubah menjadi teks memakai format bawaan basis data.
-- Hasilnya bergantung pada pengaturan NLS server, dan itu tidak dapat ditiru secara
-- portabel. Tanggalnya dikembalikan apa adanya; singkatan bulannya disusun di Go.
SELECT d.claimno       AS "CaseID",
       d.nopolis       AS "NoKTP",
       d.qqname        AS "ClaimID",
       d.businessname  AS "ClaimNo",
       -- Keduanya berisi nilai YANG SAMA di sistem lama, dengan judul berbeda:
       -- "Alasan Reject" dibaca dari alias kueri, "Reject Note" dari properti objek
       -- kerja `workpage.ClaimData.CloseClaimNote` — yang sumbernya kolom ini juga.
       d.closeclaimnote AS "CityID",
       d.closeclaimnote AS "CloseClaimNote",
       d.picteknik     AS "Conveyance",
       d.businesscode  AS "City",
       d.registerdate  AS "Country",
       d.dateofloss    AS "CountryID",
       d.closeclaimdate AS "Email",
       d.closeclaimdate AS "BulanCloseTanggal",
       d.reportername  AS "ReceiverClaim",
       (SELECT SUM(x.estimationvalue * x.kursvalue)
          FROM T_CLAIM_ESTIMASI x
         WHERE x.claimid = d.claimid) AS "Resources",
       a.share_asm     AS "ClaimAmount",
       (a.prsn_or + a.prsn_ors + a.prsn_psrqs_or + a.prsn_fsplnsor + a.prsn_psplnsor + a.prsn_psplnsor) AS "OwnRisk",
       a.own_retension AS "ExGratiaNote",
       a.coins         AS "FlagReject",
       a.psrspl        AS "InsuredRelationship",
       a.qs_ri         AS "PolicyObjectLocationRWNote",
       a.er1           AS "IsBackCFS",
       -- Kolom "ER2", dan aliasnya SENGAJA tidak dipakai berkas.
       --
       -- Berkas CSV meminta properti `IsCFS`; alias di sini `IsCFS_PNC` — berbeda satu
       -- akhiran, sehingga kolom "ER2" SELALU KOSONG di berkas lama.
       --
       -- Sempat diselaraskan menjadi `IsCFS` supaya terisi, lalu DICABUT: Work Owner
       -- menetapkan 2026-09-25 bahwa cacat yang berasal dari Pega dibiarkan seperti Pega.
       -- Nilainya tetap diambil — persis seperti sumbernya — dan tetap tidak terpakai.
       -- Yang berubah hanyalah bahwa ketiadaannya kini TERCATAT, di kolomTanpaSumber.
       --
       -- Catatan yang mudah terlewat: laporan Close Klaim Non-MBU punya cacat yang SAMA
       -- dengan arah TERBALIK — di sana aliasnya `IsCFS` dan propertinya `IsCFS_PNC`.
       a.er2           AS "IsCFS_PNC",
       a.surplus1      AS "isComplianceTransfer",
       a.surplus2      AS "IsTransferAnalisator",
       a.psrqs_ri      AS "IsTransferPIC",
       a.psrqs_or      AS "InsuredRelationshipOthers",
       a.ors           AS "KomiteStatus",
       a.facultative   AS "District",
       a.facobl        AS "LokasiSurveyor",
       a.bppdan        AS "LossCoverage",
       a.xl            AS "NamaDokumen",
       a.pss           AS "NamaSurveyor",
       a.prgbi         AS "NIK",
       a.fespl         AS "AnalystRemaksInvestigator",
       a.pfra          AS "ProdKe",
       a.prsn_fac_out  AS "AlasanDokterRejectRCL"
  FROM pooldata.t_claim_pnc d
  JOIN POOLDATA.PEGA_DASHBOARDPNC a ON d.claimno = a.noklaim
 WHERE d.statuswork = 'Resolved-Rejected'
   AND CAST(d.closeclaimdate AS DATE) >= :1
   AND CAST(d.closeclaimdate AS DATE) <= :2
   AND (
        :3 = ''
     OR (:3 = '002' AND d.grouppanel = '002')
     OR (:3 = '005' AND d.grouppanel = '005')
     OR (:3 = '346' AND d.grouppanel IN ('003','004','006','009')
                    AND d.businesscode NOT IN ('10008','10010','10015','10023','10168','10145'))
     OR (:3 = '003' AND d.grouppanel = '003'
                    AND d.businesscode IN ('10076','10077','10007','10011','10083','10141','10131','10126','10055','10075'))
       )
 ORDER BY d.closeclaimdate ASC


-- name: report_close_klaim
--
-- REPORT DATA CLOSE KLAIM, susunan ringkas — 11 kolom.
--
-- Asal: `RDB List/ExportDataCloseKlaim-SQL.xml`.
--
-- Bind:
--   :1  tanggal tutup klaim dari    DATE
--   :2  tanggal tutup klaim sampai  DATE
--   :3  kode lini bisnis
--
-- Berlaku untuk seluruh lini SELAIN Non-MBU; Non-MBU memakai kueri tersendiri yang
-- menghasilkan 81 kolom — lihat report_close_klaim_nonmbu.
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


-- name: report_temporary_close_klaim
--
-- REPORT DATA TEMPORARY CLOSE KLAIM, susunan ringkas — 11 kolom.
--
-- Activity yang SAMA dengan panel Close Klaim di atas, dibedakan oleh parameter `temp`.
-- Yang berbeda hanya keadaan klaimnya: penutupan SEMENTARA ditandai `ISPENDINGCLOSE`
-- pada objek kerja, bukan `statuswork = 'Resolved-Completed'`.
--
-- Bind: sama dengan report_close_klaim.
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
  JOIN datapega.pc_asm_fw_gcnmfw_work w ON w.pzinskey = a.claimid
 WHERE w.ispendingclose = 'true'
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


-- name: report_ai_klaim
--
-- REPORT DATA AI KLAIM — 14 kolom.
--
-- Asal: `RDB List/GetBrowseDataAIPA-SQL.xml`, dijalankan
-- `Activity/ExportHasilDataAIKlaim-Act.xml`.
--
-- Bind:
--   :1  kode lini bisnis
--
-- Kueri aslinya tidak menerima satu pun parameter, termasuk tidak menerima rentang
-- tanggal. Penyaring lini bisnis ditambahkan karena panelnya memang menyediakan
-- dropdown itu; kosong berarti seluruhnya, sama seperti perilaku lama.
--
-- # JSON_TABLE dipertahankan, NVL dan INSTR tidak
--
-- Pembacaan JSON bersarangnya disalin apa adanya: `JSON_TABLE` bersintaks sama di
-- Oracle 12c+ dan PostgreSQL 17+, dan justru itulah sebab `D-24` menetapkan PostgreSQL 17
-- sebagai syarat mengikat.
--
-- Yang diganti: `NVL` menjadi `COALESCE`, dan `INSTR` menjadi `POSITION` — keduanya ada
-- di daftar padanan wajib `09-DATABASE-STRATEGY.md` §4.
SELECT a.idpega   AS "CaseID",
       a.nopolis  AS "PolicyNo",
       (SELECT r.qqname       FROM T_CLAIM_PNC r WHERE r.claimid = a.idpega) AS "City",
       (SELECT r.businessname FROM T_CLAIM_PNC r WHERE r.claimid = a.idpega) AS "COB",
       TO_DATE(SUBSTR(jt.DateOfLoss, 1, 8), 'YYYYMMDD') AS "CityID",
       (SELECT y.startdate FROM T_GENERAL y WHERE y.nopolis = a.nopolis FETCH NEXT 1 ROW ONLY) AS "Country",
       (SELECT y.enddate   FROM T_GENERAL y WHERE y.nopolis = a.nopolis FETCH NEXT 1 ROW ONLY) AS "CountryID",
       jt.ResultAI            AS "AlasanKlaim",
       jt.COVERAGE_AI_FINAL   AS "NoteKasir",
       jt.CAUSE_OF_LOSS_AI    AS "FlagASO",
       jt.CoverageKronologi   AS "CABANG",
       jt.pyNote              AS "District",
       -- Kolom "Tipe Note AI" (`NamaSurveyor`) sengaja TIDAK ada di sini: kueri asli
       -- tidak menghasilkan alias bernama itu, sehingga kolomnya kosong di berkas lama.
       -- Mengarang sumbernya berarti mengisi kolom yang selama ini kosong dengan nilai
       -- yang tidak pernah dimaksudkan siapa pun.
       CASE WHEN jt.pyLabel = '1' THEN 'Note Terima' ELSE 'Note Tolak' END AS "DistrictID"
  FROM pooldata.json_klaim a,
       JSON_TABLE(a.DATA_JSONBLOB, '$'
         COLUMNS (Location VARCHAR PATH '$.Location',
                  DateOfLoss VARCHAR PATH '$.DateOfLoss',
                  ReportDescription VARCHAR PATH '$.ReportDescription',
                  NESTED PATH '$.ObjectList[*]'
                  COLUMNS (ObjectName VARCHAR PATH '$.ObjectName',
                           NESTED PATH '$.ObjectCoverageList[*]'
                           COLUMNS (NESTED PATH '$.AdjustmentList[*]'
                                    COLUMNS (CAUSE_OF_LOSS_AI VARCHAR PATH '$.CAUSE_OF_LOSS',
                                             ResultAI VARCHAR PATH '$.ResultAI',
                                             COVERAGE_AI_FINAL VARCHAR PATH '$.COVERAGE_AI_FINAL',
                                             CoverageKronologi VARCHAR PATH '$.CoverageKronologi',
                                             NESTED PATH '$.NotesAI[*]'
                                             COLUMNS (pyNote VARCHAR PATH '$.pyNote',
                                                      pyLabel VARCHAR PATH '$.pyLabel')))))) jt
 WHERE a.idpega IN (
         SELECT 'ASM-FW-GCNMFW-WORK ' ||
                CASE WHEN s.serviceid LIKE '%/%'
                     THEN COALESCE(SUBSTR(s.serviceid, 1, POSITION('/' IN s.serviceid) - 1), s.serviceid)
                     ELSE s.serviceid
                END
           FROM POOLDATA.CLAIM_SERVICE_LOG s
          WHERE s.categoryservice = 'RESULT AI PA')
   AND (
        :1 = ''
     OR EXISTS (SELECT 1 FROM T_CLAIM_PNC p
                 WHERE p.claimid = a.idpega
                   AND (   (:1 = '002' AND p.grouppanel = '002')
                        OR (:1 = '005' AND p.grouppanel = '005')
                        OR (:1 = '346' AND p.grouppanel IN ('003','004','006','009'))
                        OR (:1 = '003' AND p.grouppanel = '003')))
       )
 ORDER BY a.idpega DESC



-- name: report_close_klaim_nonmbu
--
-- REPORT DATA CLOSE KLAIM dan TEMPORARY CLOSE KLAIM, lini Non-MBU — 81 kolom.
--
-- Asal: `RDB List/ExportDataCloseKlaimNONMBU-SQL.xml`, dijalankan
-- `Activity/PNCReportDataClose_act-Act.xml`.
--
-- Bind:
--   :1  tanggal tutup klaim dari    DATE
--   :2  tanggal tutup klaim sampai  DATE
--   :3  penutupan sementara         'true' | 'false'
--
-- # Kenapa SATU kueri melayani dua panel
--
-- Karena di sistem lama pun satu rule melayani keduanya. Pembedanya disisipkan sebagai
-- potongan teks SQL dari langkah 9 dan 10 activity-nya:
--
--     TempDataPending.NoteKasir = "AND Z.ISPENDINGCLOSE = 'true'"                       ← sementara
--     TempDataPending.NoteKasir = "AND (Z.ISPENDINGCLOSE = 'false' or … is null)"       ← tetap
--
-- Penyisipan teks itulah yang dilarang Coding Standards §4.3. Di sini pembedanya menjadi
-- PARAMETER, bukan potongan kueri — dan karena itu satu kueri, bukan dua salinan yang
-- 190 baris identik dan hanya berbeda satu baris. Dua salinan akan menyimpang begitu
-- salah satunya disunting.
--
-- Lini bisnisnya TIDAK diikat: kueri ini hanya dipakai lini Non-MBU, dan penyaring
-- lininya sudah tertanam sebagai `group_panel IN ('003','004','006')` ditambah
-- pengecualian `groupbisnisid`. Menerima parameter lini yang tidak berpengaruh akan
-- membuat pemanggilnya mengira ia dapat menyaring lini lain lewat kueri ini.
--
-- # Empat hal yang berubah bentuk, dan sebabnya
--
--  1. **Anak-kueri `c` dihapus.** Sumber aslinya membuka `PC_ASM_FW_GCNMFW_WORK c` di
--     dalam DELAPAN anak-kueri berpenyaring `c.pyid = a.noklaim` — penyaring yang SAMA
--     PERSIS dengan join `z` di klausa FROM. Keduanya karena itu menunjuk baris yang
--     sama: bila lebih dari satu baris cocok, anak-kueri skalar aslinya melempar galat;
--     bila tidak ada yang cocok, join `z` sudah membuang barisnya lebih dulu. `z` dipakai
--     langsung.
--
--     Begitu pula `'ASM-FW-GCNMFW-WORK ' || a.noklaim` yang dirangkai di lima tempat: ia
--     persis isi `z.pzinskey` (`03-CURRENT-ARCHITECTURE.md` §4.1), dan memakai kolomnya
--     menghapus lima perangkaian teks sekaligus.
--
--  2. **`TO_CHAR` untuk tampilan dihapus.** Tanggal dikembalikan sebagai DATE dan
--     diformat di Go; bulan, tahun, dan selisih hari dihitung di Go dari tanggal yang
--     sama.
--
--  3. **`Get_InterpolasiPNC` tidak dipanggil.** `D-02` melarangnya. Yang dikembalikan
--     adalah BAHANNYA — jumlah `TOTAL_CLAIM`, fee yang sudah bertipe Adjuster Fee, dan
--     cacah barisnya — dan tangga fee dibaca sekali per laporan lalu diinterpolasi di Go.
--     Lihat reportklaim.FeeScale.
--
--  4. **`GET_POSISI_PROGRESS2` tidak dipanggil.** Yang dikembalikan adalah CLOB JSON
--     mentahnya; penguraian dan pencarian namanya terjadi di Go. Lihat
--     reportklaim.ProgressNames.
--
-- # Satu penyederhanaan yang HASILNYA sama
--
-- Kolom "Selisih Estimasi Klaim dengan Adjustment" (`Remark`) di sumbernya adalah CASE
-- bercabang dua atas `SURVEYORTYPE_1`, dan **kedua cabangnya menghitung hal yang sama**:
-- jumlah propose dikurangi jumlah gross. Ia ditulis sebagai satu ekspresi.
--
-- # Dua kolom yang penalarannya TIDAK dapat diverifikasi dari export
--
-- `ResponseNote` dan `StatusKomunikasi` — "Jumlah FU Terlambat" dan "Jumlah Tidak FU
-- Terlambat". Sumber aslinya menulis:
--
--     FROM POOLDATA.GCNM_PROGRESS_CLAIM A WHERE PNCCASEID = A.NOKLAIM
--
-- Alias `A` di sana menaungi tabel progres, sehingga `A.NOKLAIM` tidak lagi menunjuk
-- `pega_dashboardpnc a` di kueri luar. Dan `GCNM_PROGRESS_CLAIM` tidak punya kolom
-- `NOKLAIM` — daftar kolomnya terbaca dari `INSERT INTO POOLDATA.GCNM_PROGRESS_CLAIM (…)`
-- di `Database/`. Sembilan rujukan lain di export menulis `PNCCASEID = <luar>.NOKLAIM`
-- dengan alias luar yang berbeda, sehingga maksudnya tidak diragukan.
--
-- Yang ditulis di sini adalah MAKSUDNYA — dihitung per klaim, dengan alias yang dibuat
-- tidak mungkin menaungi. Ini di luar 13 butir `D-49` dan menunggu persetujuan Work Owner
-- (`D-54`).
--
-- # CURRENT_DATE menggantikan SYSDATE
--
-- Padanan wajib `09-DATABASE-STRATEGY.md` §4. Keduanya sama-sama tanggal server, sehingga
-- tidak ada perbedaan hasil.
SELECT a.sobname                                               AS "UserBusinessPA",
       CASE WHEN z.surveyortype_1 IN ('2','3','4')
            THEN z.adjusterpic_1 ELSE '' END                   AS "StatusAnalystRemarks",
       CASE WHEN z.surveyortype_1 = '1'
            THEN z.surveyorname_1
            WHEN z.surveyortype_1 = '2' AND z.businessname = 'MARINE HULL'
            THEN z.surveyornamemarine_1
            ELSE '' END                                        AS "SuspiciousComment",
       a.noklaim                                               AS "CaseID",
       a.nopolis                                               AS "TKI",
       a.clientname                                            AS "AnalystDoctorRemaks",
       a.nama_mo                                               AS "AnalystRemaksInvestigator",
       a.occupation                                            AS "AnaylstRemarks",
       a.noaksep                                               AS "City",
       a.col_desc                                              AS "CauseOfLoss",
       a.coveragename                                          AS "CityID",
       CASE WHEN e.exgratia_1 = '1' THEN 'YES' ELSE 'NO' END   AS "ClaimEstimate",
       a.dateofloss                                            AS "StatusWork",
       a.tgl_proses                                            AS "UserTeknisEmail",
       a.thnregis                                              AS "CommentKomiteClosecase",
       a.tgl_aksep                                             AS "DollarCurrencyVal",
       a.tgl_reject                                            AS "UserTeknis",
       a.begindate                                             AS "UserTeknisGroup",
       a.enddate                                               AS "RemarkRecommendation",
       a.tsi                                                   AS "ProvinceID",
       a.ttlaksep                                              AS "CountryID",
       a.ttlos                                                 AS "Currency",
       CASE a.reinsurer WHEN '1' THEN 'LEADER'
                        WHEN '2' THEN 'MEMBER'
                        WHEN 'F' THEN 'FAC-IN' END             AS "DaftarObjek",
       a.risk_loc                                              AS "Location",
       a.risk_loc_klaim                                        AS "pyID",
       a.pic                                                   AS "DistrictID",
       CASE WHEN a.ttlaksep < 100000000 THEN 'YES' ELSE 'NO' END AS "DokumenLengkap",
       a.own_retension                                         AS "ExGratiaNote",
       a.coins                                                 AS "FlagReject",
       a.psrspl                                                AS "InsuredRelationship",
       a.qs_ri                                                 AS "Investigasi",
       a.er1                                                   AS "IsBackCFS",
       -- Alias Pega `IsCFS`, properti berkas `IsCFS_PNC` — kolom "ER2" selalu kosong.
       -- Arahnya TERBALIK dari laporan Reject Klaim, dan keduanya dibiarkan apa adanya
       -- (keputusan Work Owner 2026-09-25). Lihat kolomTanpaSumber.
       a.er2                                                   AS "IsCFS",
       a.surplus1                                              AS "isComplianceTransfer",
       a.surplus2                                              AS "IsTransferAnalisator",
       a.psrqs_ri                                              AS "IsTransferPIC",
       a.psrqs_or                                              AS "InsuredRelationshipOthers",
       a.ors                                                   AS "KomiteStatus",
       a.facultative                                           AS "District",
       a.facobl                                                AS "LokasiSurveyor",
       a.bppdan                                                AS "LossCoverage",
       a.xl                                                    AS "NamaDokumen",
       a.pss                                                   AS "NamaSurveyor",
       a.prgbi                                                 AS "NIK",
       a.fespl                                                 AS "NoKTP",
       a.pfra                                                  AS "ProdKe",
       CASE WHEN a.stsklaim = '3' THEN a.ttlos ELSE 0 END      AS "Province",
       (SELECT SUM(s.propose_value * s.currencyvalue)
          FROM pooldata.t_claim_adjustment s
         WHERE s.claimid = z.pzinskey
           AND s.noakseptasi IS NOT NULL)                      AS "Password",
       CASE WHEN z.surveyortype_1 IN ('2','3','4')
            THEN (SELECT SUM(CASE WHEN s.paymenttype <> '4'
                                  THEN s.grossvalue * s.currencyvalue ELSE 0 END)
                    FROM pooldata.t_claim_adjustment s
                   WHERE s.claimid = z.pzinskey
                     AND s.noakseptasi IS NOT NULL)
            ELSE 0 END                                         AS "ReportAddress",
       CASE WHEN z.surveyortype_1 IN ('2','3','4') THEN 0
            ELSE (SELECT SUM(CASE WHEN s.paymenttype <> '4'
                                  THEN s.grossvalue * s.currencyvalue ELSE 0 END)
                    FROM pooldata.t_claim_adjustment s
                   WHERE s.claimid = z.pzinskey
                     AND s.noakseptasi IS NOT NULL)
            END                                                AS "RCV_ID",
       (SELECT SUM(s.propose_value * s.currencyvalue)
          FROM pooldata.t_claim_adjustment s
         WHERE s.claimid = z.pzinskey
           AND s.noakseptasi IS NOT NULL)
       - (SELECT SUM(CASE WHEN s.paymenttype <> '4'
                          THEN s.grossvalue * s.currencyvalue ELSE 0 END)
            FROM pooldata.t_claim_adjustment s
           WHERE s.claimid = z.pzinskey
             AND s.noakseptasi IS NOT NULL)                    AS "Remark",
       (SELECT SUM(s.total_claim)
          FROM pooldata.t_claim_adjustment s
         WHERE s.claimid = z.pzinskey
           AND s.noakseptasi = a.noaksep)                      AS "FeeDasarTotalClaim",
       (SELECT SUM(CASE WHEN s.paymenttype = '4'
                        THEN s.grossvalue * s.currencyvalue ELSE 0 END)
          FROM pooldata.t_claim_adjustment s
         WHERE s.claimid = z.pzinskey
           AND s.noakseptasi = a.noaksep)                      AS "FeeLangsung",
       (SELECT COUNT(*)
          FROM pooldata.t_claim_adjustment s
         WHERE s.claimid = z.pzinskey
           AND s.noakseptasi = a.noaksep)                      AS "FeeJumlahBaris",
       (SELECT COUNT(CASE WHEN s.paymenttype = '4' THEN NULL ELSE 1 END)
          FROM pooldata.t_claim_adjustment s
         WHERE s.claimid = z.pzinskey
           AND s.noakseptasi = a.noaksep)                      AS "FeeJumlahBarisInterpolasi",
       (SELECT s.asm_share
          FROM pooldata.t_claim_adjustment s
         WHERE s.claimid = z.pzinskey
         FETCH FIRST 1 ROW ONLY)
       * (a.prsn_or + a.prsn_ors + a.prsn_psrqs_or
          + a.prsn_fsplnsor + a.prsn_psplnsor + a.prsn_psplnsor) / 100
                                                               AS "ReportDescription",
       (SELECT SUM(s.nilai_salvage_a * s.currencyvalue)
          FROM pooldata.t_claim_adjustment s
         WHERE s.claimid = z.pzinskey
           AND s.noakseptasi IS NOT NULL)                      AS "RWID",
       p.sts_progress1                                         AS "AgingAmount",
       p.jsonstatus_progress2                                  AS "ProgresJSON",
       p.tgl_input                                             AS "TglUpdateProgres",
       p.keterangan                                            AS "pyNote",
       p.next_followup                                         AS "StatusReceiver",
       (SELECT COUNT(*)
          FROM pooldata.gcnm_progress_claim fu
         WHERE fu.pnccaseid = a.noklaim
           AND fu.status_progress2 NOT IN ('2','24','60','59')
           AND (CAST(fu.next_followup AS DATE)
                  < (SELECT CAST(nx.tgl_input AS DATE)
                       FROM pooldata.gcnm_progress_claim nx
                      WHERE nx.pnccaseid = fu.pnccaseid
                        AND nx.id_update = fu.id_update + 1)
             OR (CAST(fu.next_followup AS DATE) < CURRENT_DATE
                 AND (SELECT COUNT(ct.id_update)
                        FROM pooldata.gcnm_progress_claim ct
                       WHERE ct.pnccaseid = fu.pnccaseid) = fu.id_update)))
                                                               AS "ResponseNote",
       (SELECT COUNT(*)
          FROM pooldata.gcnm_progress_claim fu
         WHERE fu.pnccaseid = a.noklaim
           AND fu.status_progress2 NOT IN ('2','24','60','59')
           AND (CAST(fu.next_followup AS DATE)
                  >= (SELECT CAST(nx.tgl_input AS DATE)
                        FROM pooldata.gcnm_progress_claim nx
                       WHERE nx.pnccaseid = fu.pnccaseid
                         AND nx.id_update = fu.id_update + 1)
             OR (CAST(fu.next_followup AS DATE) >= CURRENT_DATE
                 AND (SELECT COUNT(ct.id_update)
                        FROM pooldata.gcnm_progress_claim ct
                       WHERE ct.pnccaseid = fu.pnccaseid) = fu.id_update)))
                                                               AS "StatusKomunikasi",
       b.coinsname                                             AS "OwnRisk",
       b.businessname                                          AS "KodeCabang",
       z.registerdate_1                                        AS "TanggalRegistrasiTeks",
       b.finishregisterdate                                    AS "NewTelpTertanggung",
       z.closeclaimdate_1                                      AS "NewEmail",
       z.closeclaimnote                                        AS "AlasanDokterRejectRCL",
       -- Kolom "Dominan Factor".
       --
       -- Sistem lama TIDAK mengambilnya lewat kueri ini: activity-nya menelusuri page list
       -- `TempWorkPage.ClaimData.DominanFactorList` milik objek kerja, lalu merangkai
       -- `.DominanName` setiap barisnya dipisah SATU SPASI
       -- (`TempDominan.City = TempDominan.City + " " + .DominanName`).
       --
       -- Isinya sendiri hidup di basis data — `T_CLAIM_DOMINANFACTOR` yang dinamai lewat
       -- `M_DOMINAN_FACTOR`, persis seperti yang dibaca `GetDataDominanFactorListOS`.
       --
       -- # Kenapa hanya KUNCINYA yang diambil di sini
       --
       -- Karena perangkaiannya tidak dapat ditulis secara portabel: Oracle punya
       -- `LISTAGG`, PostgreSQL punya `STRING_AGG`, dan tidak ada satu pun yang berjalan
       -- di keduanya. `09-DATABASE-STRATEGY.md` §4 melarang yang pertama justru karena
       -- itu — sementara yang kedua tidak ada di Oracle, tempat aplikasi ini berjalan
       -- hari ini.
       --
       -- Daftarnya karena itu dibaca SEKALI per laporan lewat report_dominant_factors,
       -- lalu dirangkai di Go. Lihat reportklaim.DominantFactors.
       z.pzinskey                                              AS "KunciKlaimDominan"
  FROM pooldata.pega_dashboardpnc a
  JOIN pooldata.t_claim_pnc b              ON b.claimno = a.noklaim
  JOIN datapega.pc_asm_fw_gcnmfw_work z    ON z.pyid = a.noklaim
  JOIN datapega.pc_asm_fw_gcnmfw_work e    ON e.pxinsname = a.noklaim
  LEFT JOIN pooldata.gcnm_progress_claim p
         ON p.pnccaseid = a.noklaim
        AND p.id_update = (SELECT MAX(m.id_update)
                             FROM pooldata.gcnm_progress_claim m
                            WHERE m.pnccaseid = a.noklaim
                              AND m.status_progress2 NOT IN ('2','24','60','59'))
 WHERE a.group_panel IN ('003','004','006')
   AND a.groupbisnisid NOT IN ('09','11','16','25')
   AND a.stsklaim IN ('1','3')
   AND z.pystatuswork = 'Resolved-Completed'
   AND ((:3 = 'true'  AND z.ispendingclose = 'true')
     OR (:3 = 'false' AND (z.ispendingclose = 'false' OR z.ispendingclose IS NULL)))
   AND CAST(z.closeclaimdate_1 AS DATE) >= :1
   AND CAST(z.closeclaimdate_1 AS DATE) <= :2
 ORDER BY z.closeclaimdate_1 ASC

-- name: report_fee_scale
--
-- Tangga fee adjuster — 17 pita. Sumber `POOLDATA.Get_InterpolasiPNC`, yang `D-02`
-- haruskan ditulis ulang di Go. Dibaca SEKALI per laporan, lihat reportklaim.FeeScale.
SELECT index_fee  AS "IndexFee",
       loss_amount AS "LossAmount",
       fee         AS "Fee"
  FROM pooldata.gcnm_fee_scale
 ORDER BY index_fee


-- name: report_progress_names
--
-- Nama tahapan progres. Sumber `POOLDATA.GET_POSISI_PROGRESS2`, yang `D-02` haruskan
-- ditulis ulang di Go. Dibaca SEKALI per laporan, lihat reportklaim.ProgressNames.
SELECT id_mst        AS "IDMst",
       sts_progress2 AS "Nama"
  FROM pooldata.gcnm_mst_progress


-- name: report_dominant_factors
--
-- Faktor dominan per klaim, untuk kolom "Dominan Factor" pada laporan Close Klaim
-- dan Temporary Close Klaim susunan rinci.
--
-- Bind:
--   :1  tanggal tutup klaim dari    DATE
--   :2  tanggal tutup klaim sampai  DATE
--
-- Rentangnya SAMA dengan laporan yang memakainya. Tanpa batas itu, kueri ini membaca
-- seluruh riwayat faktor dominan untuk melayani satu bulan laporan.
--
-- Urutannya ditetapkan DI SINI, bukan di Go: `idx_dominanfactor` adalah satu-satunya
-- sumber urutan, dan membawanya ke Go hanya untuk mengurutkan ulang berarti
-- menduakannya.
SELECT tdf.claimid AS "ClaimID",
       mdf.name    AS "Nama"
  FROM pooldata.t_claim_dominanfactor tdf
  JOIN pooldata.m_dominan_factor mdf ON mdf.id = tdf.id_dominanfactor
  JOIN datapega.pc_asm_fw_gcnmfw_work w ON w.pzinskey = tdf.claimid
 WHERE CAST(w.closeclaimdate_1 AS DATE) >= :1
   AND CAST(w.closeclaimdate_1 AS DATE) <= :2
 ORDER BY tdf.claimid, tdf.idx_dominanfactor
