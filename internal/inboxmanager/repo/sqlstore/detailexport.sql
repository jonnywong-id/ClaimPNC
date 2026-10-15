-- Kueri "Export Data Detail Klaim" — tombol "Export to Excel" pada tab Outstanding.
--
-- Asal: `Activity/ExportDataDetailKlaim` -> `RDB List/GetExportDataDetailKlaim-SQL.xml`,
-- dipanggil tombol `pyButtonLabel Export To Excel` di `Section/PNCDashboardOS`.
--
-- Activity itu punya DUA cabang kueri, dipilih `TempVisibility.Email=="8"`. Nilai "8" adalah
-- tab Temporary Close milik layar LAIN (`Section/InboxRegister_Section`), sehingga dari
-- Inbox Manager yang berjalan selalu cabang utama — `GetExportDataDetailKlaim` dengan
-- saringan tanggal `c.pxcreatedatetime`. Cabang `…TempClose` tidak dibawa.
--
-- ============================================================================
-- SUMBER: T_CLAIMLIST_ADMIN + T_CLAIM_PNC, BUKAN DATAPEGA (2026-09-28, 2026-10-14)
-- ============================================================================
--
-- Sejak 15/06/2026 kueri Pega membaca `datapega.pc_asm_fw_gcnmfw_work c JOIN
-- datapega.pc_assign_worklist b`. Keduanya diganti tabel datar yang sama dengan dashboard,
-- atas permintaan Work Owner 2026-10-14 ("t_claim_pnc dan t_claimlist_admin"). Seluruh kolom
-- `c.` yang dipakai ADA di tabel datar, dan sejak backfill 2026-10-08 kolom surveyor/adjuster
-- TERISI — diperiksa 2026-10-14 di DEV_PEGA83: 2.947 baris, SURVEYORTYPE_1 674,
-- ADJUSTERPIC_1 198, SURVEYORNAME_1 96, SURVEYORNAMEMARINE_1 10.
--
-- Join worklist hilang bersama dua cacatnya (klaim tanpa penugasan lenyap, klaim
-- berpenugasan banyak berulang) — sama dengan dashboard; lihat PlannedDifferences.
--
-- Enam anak-kueri `pooldata.t_claim_pnc D where d.claimid = c.pzinskey` menjadi SATU LEFT
-- JOIN `d`. Anak-kueri skalar aslinya GAGAL bila `claimid` tidak unik, sehingga join tidak
-- dapat menghasilkan baris yang di Pega tampil berbeda.
--
-- ============================================================================
-- TIGA NILAI YANG PEGA BACA DARI OBJEK KERJA, BUKAN DARI KUERI
-- ============================================================================
--
-- Untuk setiap baris, activity-nya menjalankan `Obj-Open-By-Handle pyWorkPage` dan membaca
-- `ClaimData.SubjectEmail`, `ClaimData.PolicyLeaderNo`, dan daftar okupasi polis — isi BLOB
-- Pega yang tidak punya kolom. `T_CLAIM_PNC` punya `SUBJECTEMAIL` dan `POLISLEADER`, tetapi
-- KOSONG pada seluruh klaim terbuka (diperiksa 2026-10-14), sehingga kedua kolom berkas itu
-- kosong. Okupasi didekati lewat `T_PROPERTYLIST` — lihat kolom "AnaylstRemarks".
--
-- ============================================================================
-- YANG BERUBAH BENTUK, MENGIKUTI MODUL REPORT KLAIM
-- ============================================================================
--
--  * `TO_CHAR` untuk tampilan dihapus: tanggal dikembalikan apa adanya dan diformat di Go,
--    termasuk bulan, tahun, dan TAT.
--  * `Get_InterpolasiPNC` dan `GET_POSISI_PROGRESS2` tidak dipanggil (`D-02`): yang
--    dikembalikan BAHANNYA, dan aturannya berjalan di Go.
--  * Kelima kolom "… OR" dikalikan di Go dengan aritmetika desimal, persis
--    `@toDecimal(a) * @toDecimal(b)` di activity.
--  * `SYSDATE` menjadi `CURRENT_DATE`, `NVL` menjadi `COALESCE`, dan `DECODE` menjadi
--    `CASE` (`09-DATABASE-STRATEGY.md` §4, dikunci TestTidakAdaPolaSQLTerlarang).
--  * Saringan tanggal menjadi selang setengah terbuka atas kolomnya, bukan `trunc(kolom)`.
--  * Belasan anak-kueri progres terakhir menjadi dua LEFT JOIN (`p` dan `pk`).

-- name: detail_export
-- Bind: :1..:5 bendera lini bisnis (lihat lineFlags; cabang NONMBU di sini IKUT menuntut
--       USERTEKNIS_1 terisi, sesuai `TempBisnis.BUSINESSTYPE` activity-nya)
--       :6 bendera "tanpa saringan tanggal" · :7 dari (inklusif) · :8 sampai (eksklusif)
SELECT c.PYSTATUSWORK                                          AS "pyStatusMessage",
       c.SOBNAME                                               AS "UserBusinessPA",
       CASE WHEN c.SURVEYORTYPE_1 IN ('2', '3', '4')
            THEN c.ADJUSTERPIC_1 ELSE '' END                   AS "StatusAnalystRemarks",
       CASE WHEN c.SURVEYORTYPE_1 = '1'
            THEN c.SURVEYORNAME_1
            WHEN c.SURVEYORTYPE_1 = '2' AND c.BUSINESSNAME = 'MARINE HULL'
            THEN c.SURVEYORNAMEMARINE_1
            ELSE '' END                                        AS "SuspiciousComment",
       c.PYID                                                  AS "CaseID",
       a.LGB_NOTE                                              AS "AlasanDokterRejectRCL",
       c.POLICYNO                                              AS "TKI",
       d.BRANCHNAME                                            AS "Usulan",
       (SELECT br.KANWIL
          FROM POOLDATA.BRANCH br
         WHERE br.ID = d.BRANCHCODE)                           AS "ResponseMsg",
       a.CLIENTNAME                                            AS "AnalystDoctorRemaks",
       a.NAMA_MO                                               AS "AnalystRemaksInvestigator",
       -- Okupasi. Activity-nya memilih, berurutan: okupasi `IsUsedFlag` pada
       -- `Policy.OccupationList` objek kerja (BLOB — tidak terbaca di sini), lalu kolom
       -- `a.occupation`, lalu okupasi `IsUsedFlag` pada `Policy.PropertyList` (yang tersimpan
       -- sebagai `T_PROPERTYLIST`), lalu okupasi PERTAMA `T_PROPERTYLIST` milik objek pertama
       -- klaim. Tiga yang terakhir dibawa dengan urutan yang sama.
       COALESCE(
         a.OCCUPATION,
         (SELECT oc.NAMA
            FROM POOLDATA.T_PROPERTYLIST pl,
                 JSON_TABLE(pl.OCCUPATIONLIST, '$.OccupationList[*]'
                   COLUMNS (DIPAKAI VARCHAR2(10)  PATH '$.IsUsedFlag',
                            NAMA    VARCHAR2(400) PATH '$.OccupationName')) oc
           WHERE pl.NOPOLIS = c.POLICYNO
             AND pl.PRODKE = c.PRODKE
             AND oc.DIPAKAI = 'true'
           FETCH FIRST 1 ROW ONLY),
         (SELECT oc.NAMA
            FROM POOLDATA.T_PROPERTYLIST pl,
                 JSON_TABLE(pl.OCCUPATIONLIST, '$.OccupationList[*]'
                   COLUMNS (NAMA VARCHAR2(400) PATH '$.OccupationName')) oc
           WHERE pl.NOPOLIS = c.POLICYNO
             AND pl.PRODKE = c.PRODKE
             AND pl.INDEXOBJECT = (SELECT ob.OBJECTID
                                     FROM POOLDATA.T_CLAIM_OBJECTLIST ob
                                    WHERE ob.CLAIMID = c.PZINSKEY
                                    ORDER BY ob.OBJECTINDEX, ob.OBJECTID
                                    FETCH FIRST 1 ROW ONLY)
           FETCH FIRST 1 ROW ONLY))                            AS "AnaylstRemarks",
       a.NOAKSEP                                               AS "City",
       a.COL_DESC                                              AS "CauseOfLoss",
       a.COVERAGENAME                                          AS "CityID",
       -- `DECODE(a.sts_exgratia, '0', 'NO', NULL, 'YES')` — nilai lain menghasilkan kosong.
       CASE WHEN a.STS_EXGRATIA = '0' THEN 'NO'
            WHEN a.STS_EXGRATIA IS NULL THEN 'YES' END         AS "ClaimEstimate",
       a.DATEOFLOSS                                            AS "TglDateOfLoss",
       a.TGL_PROSES                                            AS "TglProses",
       a.THNREGIS                                              AS "CommentKomiteClosecase",
       a.BEGINDATE                                             AS "TglBegin",
       a.TGL_AKSEP                                             AS "TglAksep",
       a.TGL_REJECT                                            AS "TglReject",
       a.ENDDATE                                               AS "TglEnd",
       a.TSI                                                   AS "ProvinceID",
       a.TTLAKSEP                                              AS "CountryID",
       a.TTLOS                                                 AS "Currency",
       -- `DECODE(NVL(a.reinsurer, d.leader_member), …)` — nilai lain menghasilkan kosong.
       CASE COALESCE(a.REINSURER, d.LEADER_MEMBER)
            WHEN '1' THEN 'LEADER' WHEN 'LEADER' THEN 'LEADER'
            WHEN '2' THEN 'MEMBER' WHEN 'MEMBER' THEN 'MEMBER'
            WHEN 'F' THEN 'FAC-IN' WHEN 'FAC IN' THEN 'FAC-IN'
       END                                                     AS "DaftarObjek",
       a.RISK_LOC                                              AS "Location",
       a.RISK_LOC_KLAIM                                        AS "pyID",
       a.PIC                                                   AS "DistrictID",
       CASE WHEN a.TTLAKSEP < 100000000
            THEN 'YES' ELSE 'NO' END                           AS "DokumenLengkap",
       a.OWN_RETENSION                                         AS "ExGratiaNote",
       a.COINS                                                 AS "FlagReject",
       a.PSRSPL                                                AS "InsuredRelationship",
       a.QS_RI                                                 AS "PolicyObjectLocationRWNote",
       a.ER1                                                   AS "IsBackCFS",
       -- Kolom "ER2" TIDAK dipilih: aliasnya di Pega `IsCFS` sementara berkasnya meminta
       -- `IsCFS_PNC`, sehingga kolom itu SELALU KOSONG di berkas lama. Dibiarkan kosong —
       -- ketetapan Work Owner 2026-09-25, cacat Pega dibiarkan seperti Pega; preseden yang
       -- sama dengan modul Report Klaim.
       a.SURPLUS1                                              AS "isComplianceTransfer",
       a.SURPLUS2                                              AS "IsTransferAnalisator",
       a.PSRQS_RI                                              AS "IsTransferPIC",
       a.PSRQS_OR                                              AS "InsuredRelationshipOthers",
       a.ORS                                                   AS "KomiteStatus",
       a.FACULTATIVE                                           AS "District",
       a.FACOBL                                                AS "LokasiSurveyor",
       a.BPPDAN                                                AS "LossCoverage",
       a.XL                                                    AS "NamaDokumen",
       a.PSS                                                   AS "NamaSurveyor",
       a.PRGBI                                                 AS "NIK",
       a.FESPL                                                 AS "NoKTP",
       d.REPORTERNAME                                          AS "Resources",
       a.PFRA                                                  AS "ProdKe",
       CASE WHEN a.STSKLAIM = '3' THEN a.TTLOS ELSE 0 END      AS "Province",
       (SELECT SUM(z.PROPOSE_VALUE * z.CURRENCYVALUE)
          FROM POOLDATA.T_CLAIM_ADJUSTMENT z
         WHERE z.CLAIMID = 'ASM-FW-GCNMFW-WORK ' || a.NOKLAIM
           AND z.STATUSAKSEPTASI = '1'
           AND (z.STATUSAKSEPTASILOD = '1' OR z.STATUSAKSEPTASILOD IS NULL))
                                                               AS "Password",
       CASE WHEN c.SURVEYORTYPE_1 IN ('2', '3', '4')
            THEN (SELECT SUM(CASE WHEN z.PAYMENTTYPE != '4'
                                  THEN z.GROSSVALUE * z.CURRENCYVALUE ELSE 0 END)
                    FROM POOLDATA.T_CLAIM_ADJUSTMENT z
                   WHERE z.CLAIMID = c.PZINSKEY
                     AND z.STATUSAKSEPTASI = '1'
                     AND (z.STATUSAKSEPTASILOD = '1' OR z.STATUSAKSEPTASILOD IS NULL))
            ELSE 0 END                                         AS "ReportAddress",
       CASE WHEN c.SURVEYORTYPE_1 IN ('2', '3', '4') THEN 0
            ELSE (SELECT SUM(CASE WHEN z.PAYMENTTYPE != '4'
                                  THEN z.GROSSVALUE * z.CURRENCYVALUE ELSE 0 END)
                    FROM POOLDATA.T_CLAIM_ADJUSTMENT z
                   WHERE z.CLAIMID = c.PZINSKEY
                     AND z.STATUSAKSEPTASI = '1'
                     AND (z.STATUSAKSEPTASILOD = '1' OR z.STATUSAKSEPTASILOD IS NULL))
            END                                                AS "RCV_ID",
       -- "Selisih … Adjuster" (Remark) dan "Selisih … Adjustment" (StatusEkonomis): propose
       -- dikurangi gross, dibelah menurut tipe surveyor seperti dua kolom di atas.
       CASE WHEN c.SURVEYORTYPE_1 IN ('2', '3', '4')
            THEN (SELECT SUM(z.PROPOSE_VALUE * z.CURRENCYVALUE)
                    FROM POOLDATA.T_CLAIM_ADJUSTMENT z
                   WHERE z.CLAIMID = c.PZINSKEY
                     AND z.STATUSAKSEPTASI = '1'
                     AND (z.STATUSAKSEPTASILOD = '1' OR z.STATUSAKSEPTASILOD IS NULL))
               - (SELECT SUM(CASE WHEN z.PAYMENTTYPE != '4'
                                  THEN z.GROSSVALUE * z.CURRENCYVALUE ELSE 0 END)
                    FROM POOLDATA.T_CLAIM_ADJUSTMENT z
                   WHERE z.CLAIMID = c.PZINSKEY
                     AND z.STATUSAKSEPTASI = '1'
                     AND (z.STATUSAKSEPTASILOD = '1' OR z.STATUSAKSEPTASILOD IS NULL))
            ELSE 0 END                                         AS "Remark",
       CASE WHEN c.SURVEYORTYPE_1 IN ('2', '3', '4') THEN 0
            ELSE (SELECT SUM(z.PROPOSE_VALUE * z.CURRENCYVALUE)
                    FROM POOLDATA.T_CLAIM_ADJUSTMENT z
                   WHERE z.CLAIMID = c.PZINSKEY
                     AND z.STATUSAKSEPTASI = '1'
                     AND (z.STATUSAKSEPTASILOD = '1' OR z.STATUSAKSEPTASILOD IS NULL))
               - (SELECT SUM(CASE WHEN z.PAYMENTTYPE != '4'
                                  THEN z.GROSSVALUE * z.CURRENCYVALUE ELSE 0 END)
                    FROM POOLDATA.T_CLAIM_ADJUSTMENT z
                   WHERE z.CLAIMID = c.PZINSKEY
                     AND z.STATUSAKSEPTASI = '1'
                     AND (z.STATUSAKSEPTASILOD = '1' OR z.STATUSAKSEPTASILOD IS NULL))
            END                                                AS "StatusEkonomis",
       -- Bahan "Adjuster Fee". Sumbernya menjumlahkan, atas setiap baris settlement,
       -- `grossvalue*currencyvalue` bila bertipe '4' dan `Get_InterpolasiPNC(SUM(TOTAL_CLAIM))`
       -- bila bukan. Interpolasinya berjalan di Go (`D-02`).
       (SELECT SUM(z.TOTAL_CLAIM)
          FROM POOLDATA.T_CLAIM_ADJUSTMENT z
         WHERE z.CLAIMID = 'ASM-FW-GCNMFW-WORK ' || a.NOKLAIM
           AND z.STATUSAKSEPTASI = '1'
           AND (z.STATUSAKSEPTASILOD = '1' OR z.STATUSAKSEPTASILOD IS NULL))
                                                               AS "FeeDasarTotalClaim",
       (SELECT SUM(CASE WHEN z.PAYMENTTYPE = '4'
                        THEN z.GROSSVALUE * z.CURRENCYVALUE ELSE 0 END)
          FROM POOLDATA.T_CLAIM_ADJUSTMENT z
         WHERE z.CLAIMID = 'ASM-FW-GCNMFW-WORK ' || a.NOKLAIM
           AND z.STATUSAKSEPTASI = '1'
           AND (z.STATUSAKSEPTASILOD = '1' OR z.STATUSAKSEPTASILOD IS NULL))
                                                               AS "FeeLangsung",
       (SELECT COUNT(*)
          FROM POOLDATA.T_CLAIM_ADJUSTMENT z
         WHERE z.CLAIMID = 'ASM-FW-GCNMFW-WORK ' || a.NOKLAIM
           AND z.STATUSAKSEPTASI = '1'
           AND (z.STATUSAKSEPTASILOD = '1' OR z.STATUSAKSEPTASILOD IS NULL))
                                                               AS "FeeJumlahBaris",
       (SELECT COUNT(CASE WHEN z.PAYMENTTYPE = '4' THEN NULL ELSE 1 END)
          FROM POOLDATA.T_CLAIM_ADJUSTMENT z
         WHERE z.CLAIMID = 'ASM-FW-GCNMFW-WORK ' || a.NOKLAIM
           AND z.STATUSAKSEPTASI = '1'
           AND (z.STATUSAKSEPTASILOD = '1' OR z.STATUSAKSEPTASILOD IS NULL))
                                                               AS "FeeJumlahBarisInterpolasi",
       -- Faktor bagian OR — SATU nilai yang di Pega ditulis lima kali dengan lima alias.
       -- Empat kolom berkas mengalikannya dengan nilai lain di activity; yang kelima
       -- ("Selisih … Adjuster OR") menuliskannya tanpa dikalikan. Keduanya di Go.
       (SELECT z.ASM_SHARE
          FROM POOLDATA.T_CLAIM_ADJUSTMENT z
         WHERE z.CLAIMID = 'ASM-FW-GCNMFW-WORK ' || a.NOKLAIM
           AND z.STATUSAKSEPTASI = '1'
           AND (z.STATUSAKSEPTASILOD = '1' OR z.STATUSAKSEPTASILOD IS NULL)
         FETCH NEXT 1 ROW ONLY)
       * (a.PRSN_OR + a.PRSN_ORS + a.PRSN_PSRQS_OR
          + a.PRSN_FSPLNSOR + a.PRSN_PSPLNSOR + a.PRSN_PSPLNSOR) / 100
                                                               AS "FaktorOR",
       (SELECT SUM(z.NILAI_SALVAGE_A * z.CURRENCYVALUE)
          FROM POOLDATA.T_CLAIM_ADJUSTMENT z
         WHERE z.CLAIMID = 'ASM-FW-GCNMFW-WORK ' || a.NOKLAIM
           AND z.STATUSAKSEPTASI = '1'
           AND (z.STATUSAKSEPTASILOD = '1' OR z.STATUSAKSEPTASILOD IS NULL))
                                                               AS "RWID",
       mprog.STS_PROGRESS1                                     AS "AgingAmount",
       p.JSONSTATUS_PROGRESS2                                  AS "ProgresJSON",
       p.DOCTYPE                                               AS "ReporterTelp",
       p.TGL_INPUT                                             AS "TglUpdateProgres",
       p.KETERANGAN                                            AS "pyNote",
       p.NEXT_FOLLOWUP                                         AS "TglNextFollowUp",
       -- "Jumlah FU Terlambat" dan "Jumlah Tidak FU Terlambat". Kueri Pega memberinya alias
       -- HURUF BESAR (`RESPONSENOTE`, `STATUSKOMUNIKASI`) sementara berkasnya membaca
       -- `ResponseNote` dan `StatusKomunikasi`; apakah pemetaan RDB-List Pega peka huruf tidak
       -- terbaca dari export. Yang dibawa adalah MAKSUDNYA — dihitung — sama dengan modul
       -- Report Klaim.
       --
       -- `HAVING COUNT(*) > 0` menjaga klaim tanpa progres tetap KOSONG, bukan nol — sumbernya
       -- `SUM` atas himpunan kosong, yang menghasilkan NULL.
       (SELECT COUNT(CASE WHEN CAST(fu.NEXT_FOLLOWUP AS DATE)
                                 < (SELECT CAST(nx.TGL_INPUT AS DATE)
                                      FROM POOLDATA.GCNM_PROGRESS_CLAIM nx
                                     WHERE nx.PNCCASEID = fu.PNCCASEID
                                       AND nx.ID_UPDATE = fu.ID_UPDATE + 1)
                               OR (CAST(fu.NEXT_FOLLOWUP AS DATE) < CURRENT_DATE
                                   AND (SELECT COUNT(ct.ID_UPDATE)
                                          FROM POOLDATA.GCNM_PROGRESS_CLAIM ct
                                         WHERE ct.PNCCASEID = fu.PNCCASEID) = fu.ID_UPDATE)
                          THEN 1 END)
          FROM POOLDATA.GCNM_PROGRESS_CLAIM fu
         WHERE fu.PNCCASEID = a.NOKLAIM
           AND fu.STATUS_PROGRESS2 NOT IN ('2', '24', '60', '59')
        HAVING COUNT(*) > 0)                                   AS "ResponseNote",
       (SELECT COUNT(CASE WHEN CAST(fu.NEXT_FOLLOWUP AS DATE)
                                 >= (SELECT CAST(nx.TGL_INPUT AS DATE)
                                       FROM POOLDATA.GCNM_PROGRESS_CLAIM nx
                                      WHERE nx.PNCCASEID = fu.PNCCASEID
                                        AND nx.ID_UPDATE = fu.ID_UPDATE + 1)
                               OR (CAST(fu.NEXT_FOLLOWUP AS DATE) >= CURRENT_DATE
                                   AND (SELECT COUNT(ct.ID_UPDATE)
                                          FROM POOLDATA.GCNM_PROGRESS_CLAIM ct
                                         WHERE ct.PNCCASEID = fu.PNCCASEID) = fu.ID_UPDATE)
                          THEN 1 END)
          FROM POOLDATA.GCNM_PROGRESS_CLAIM fu
         WHERE fu.PNCCASEID = a.NOKLAIM
           AND fu.STATUS_PROGRESS2 NOT IN ('2', '24', '60', '59')
        HAVING COUNT(*) > 0)                                   AS "StatusKomunikasi",
       (SELECT cp.COINSNAME
          FROM POOLDATA.T_CLAIM_PNC cp
         WHERE cp.CLAIMID = 'ASM-FW-GCNMFW-WORK ' || a.NOKLAIM) AS "OwnRisk",
       SUBSTR(k.OBJECTNAME, 1, 100)                            AS "TypeOfAddress",
       pk.TGL_INPUT                                            AS "TglKirimKomite",
       pk.NEXT_FOLLOWUP                                        AS "TglJatuhTempoKomite",
       a.NOPOLIS || '/' || a.NOKLAIM                           AS "viewnotelp",
       (SELECT bk.PINWILKLAIM
          FROM POOLDATA.BRANCHKLAIMPNC bk
         WHERE bk.ID = d.BRANCHCODE)                           AS "HideKTP",
       (SELECT bk.KANWILKLAIM
          FROM POOLDATA.BRANCHKLAIMPNC bk
         WHERE bk.ID = d.BRANCHCODE)                           AS "HidePelapor",
       (SELECT z.ALASANTOLAK
          FROM POOLDATA.T_CLAIM_ADJUSTMENT z
         WHERE z.CLAIMID = 'ASM-FW-GCNMFW-WORK ' || a.NOKLAIM
           AND z.OBJECTID = k.OBJECTID
           AND z.ALASANTOLAK IS NOT NULL
         FETCH NEXT 1 ROW ONLY)                                AS "TypeOldNoHp"
  FROM POOLDATA.T_CLAIMLIST_ADMIN c
  JOIN POOLDATA.BUSINESS f
    ON c.BUSINESSCODE_1 = f.ID
  JOIN POOLDATA.BUSINESSGROUP e
    ON f.BUSINESSGROUPID = e.ID
  LEFT JOIN POOLDATA.T_CLAIM_OBJECTLIST k
    ON k.CLAIMID = c.PZINSKEY
  LEFT JOIN POOLDATA.PEGA_DASHBOARDPNC a
    ON a.NOKLAIM = c.PYID
  LEFT JOIN POOLDATA.T_CLAIM_PNC d
    ON d.CLAIMID = c.PZINSKEY
  -- Progres TERAKHIR, mengabaikan empat tahapan yang Pega abaikan.
  LEFT JOIN POOLDATA.GCNM_PROGRESS_CLAIM p
    ON p.PNCCASEID = a.NOKLAIM
   AND p.ID_UPDATE = (SELECT MAX(m.ID_UPDATE)
                        FROM POOLDATA.GCNM_PROGRESS_CLAIM m
                       WHERE m.PNCCASEID = a.NOKLAIM
                         AND m.STATUS_PROGRESS2 NOT IN ('2', '24', '60', '59'))
  LEFT JOIN POOLDATA.GCNM_MST_PROGRESS_KLAIM mprog
    ON mprog.ID_PROGRESS = p.STATUS_PROGRESS1
  -- Progres terakhir bertahap '017'/'018' — "Tgl Kirim" dan "Tgl Jatuh Tempo".
  LEFT JOIN POOLDATA.GCNM_PROGRESS_CLAIM pk
    ON pk.PNCCASEID = a.NOKLAIM
   AND pk.ID_UPDATE = (SELECT MAX(m.ID_UPDATE)
                         FROM POOLDATA.GCNM_PROGRESS_CLAIM m
                        WHERE m.PNCCASEID = a.NOKLAIM
                          AND m.STATUS_PROGRESS1 IN ('017', '018'))
 WHERE c.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND c.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (c.PXFLOWNAME IS NULL OR c.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1'))
   AND (c.PXTASKLABEL IS NULL OR c.PXTASKLABEL NOT IN ('FixCorrespondence'))
   AND (c.ISPENDINGCLOSE != 'true' OR c.ISPENDINGCLOSE IS NULL)
   AND ( :1 = 1
      OR ( :2 = 1
           AND c.GROUPPANEL_1 IN ('003', '004', '006')
           AND f.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023')
           AND c.BRANCHNAME <> 'ASNET'
           AND c.USERTEKNIS_1 IS NOT NULL )
      OR ( :3 = 1
           AND f.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023') )
      OR ( :4 = 1 AND c.GROUPPANEL_1 = '002' )
      OR ( :5 = 1 AND c.GROUPPANEL_1 = '005' ) )
   AND ( :6 = 1 OR (c.PXCREATEDATETIME >= :7 AND c.PXCREATEDATETIME < :8) )
 ORDER BY c.PYID, k.OBJECTID

-- name: detail_fee_scale
-- Tangga fee adjuster — isi yang dibaca `POOLDATA.Get_InterpolasiPNC` (`D-02`). Dibaca SEKALI
-- per berkas. Kueri yang sama dengan `report_fee_scale` modul Report Klaim.
SELECT f.INDEX_FEE   AS "IndexFee",
       f.LOSS_AMOUNT AS "LossAmount",
       f.FEE         AS "Fee"
  FROM POOLDATA.GCNM_FEE_SCALE f
 ORDER BY f.INDEX_FEE

-- name: detail_progress_names
-- Nama tahapan progres — isi yang dibaca `POOLDATA.GET_POSISI_PROGRESS2` (`D-02`). Dibaca
-- SEKALI per berkas. Kueri yang sama dengan `report_progress_names` modul Report Klaim.
SELECT m.ID_MST        AS "IDMst",
       m.STS_PROGRESS2 AS "Nama"
  FROM POOLDATA.GCNM_MST_PROGRESS m
