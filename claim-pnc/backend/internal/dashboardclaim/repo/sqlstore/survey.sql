-- Kueri tile LOSS ADJUSTER dan INTERNAL SURVEYOR.
--
-- Menggantikan empat rule sistem lama, dipanggil `SetDashboardClaim` saat `param.tipe`
-- bernilai 2 dan 3:
--
--     RDB List/Get_CountLostAdjusterClaim-SQL.xml   angka kartu Loss Adjuster
--     RDB List/BrowseLossAdjuster-SQL.xml           telusur Loss Adjuster
--     RDB List/Get_CountInternalSurveyor-SQL.xml    angka kartu Internal Surveyor
--     RDB List/BrowseInternalSurveyor-SQL.xml       telusur Internal Surveyor
--
-- Keduanya membaca tabel yang sama, dibedakan `SURVEYORTYPE_1`: '2' loss adjuster eksternal,
-- '1' surveyor internal ASM.
--
-- ============================================================================
-- KEEMPAT KUERI LAMA TIDAK SEJAJAR SATU SAMA LAIN
-- ============================================================================
--
-- Ini bukan penyederhanaan bacaan — keempatnya benar-benar berbeda syarat, dan selisihnya
-- terbaca langsung dari sumbernya:
--
--   a. PENYARING LINI BISNIS BERADA DI TEMPAT YANG BERBEDA.
--      Pada Loss Adjuster ia di dalam EXISTS, menyaring KLAIM INDUK.
--      Pada Internal Surveyor ia di luar, menyaring BARIS SURVEI itu sendiri.
--      Keduanya dipertahankan apa adanya (`P-5`).
--
--   b. STATUS KLAIM INDUK HANYA DIPERIKSA DI SATU SISI.
--      `BrowseInternalSurveyor` menuntut induknya belum selesai; `BrowseLossAdjuster`
--      tidak memeriksanya sama sekali pada kueri daftar, meski kueri hitungnya memeriksa.
--
--   c. STATUS SURVEI HANYA DIPERIKSA DI SATU SISI PADA KUERI HITUNG.
--      `Get_CountInternalSurveyor` membuang survei yang sudah selesai; padanan loss
--      adjuster-nya tidak.
--
-- ============================================================================
-- KEDUA KUERI HITUNG MENGIKUTI PEGA APA ADANYA — KEPUTUSAN WORK OWNER 2026-09-26
-- ============================================================================
--
-- Kedua kueri hitung lama TIDAK MENGHITUNG SATUAN YANG SAMA:
--
--     Get_CountLostAdjusterClaim   SUM(CASE WHEN (…) > 0 THEN 1 ELSE 0 END)
--     Get_CountInternalSurveyor    SUM(CASE WHEN (…) > 0 THEN (… COUNT … ) ELSE 0 END)
--
-- Yang pertama menghitung JUMLAH KLAIM yang punya survei adjuster. Yang kedua menghitung
-- JUMLAH SURVEI-nya. Padahal kedua kartu menelusur ke daftar yang sama-sama berisi BARIS
-- SURVEI.
--
-- Ditambah: kedua kueri hitung menggabung ke `PC_ASSIGN_WORKLIST` sementara `SUM` berjalan
-- atas baris hasil gabungan itu, sehingga klaim yang memegang lebih dari satu penugasan
-- TERHITUNG BERKALI-KALI.
--
-- Penyeragaman sempat dikerjakan lalu DICABUT: Work Owner memutuskan ketiganya mengikuti
-- Pega apa adanya (`P-5`), sehingga tidak ada satu pun selisih terencana pada kedua angka
-- ini dan gerbang 1 dapat membandingkannya baris per baris tanpa pengecualian.
--
-- Akibat yang harus disadari, dan yang karena itu dinyatakan di layar: **angka pada kartu
-- tidak akan sama dengan jumlah baris telusurnya.** Ia bukan cacat modul ini melainkan
-- perilaku yang diwarisi, dan menyatakannya di layar adalah satu-satunya hal yang
-- ditambahkan — bukan angkanya yang diubah.
--
-- Pemeriksaan bahwa kueri hitung dan kueri daftar memang BOLEH berbeda dijaga
-- `TestCountAndListMayDifferForSurveyTiles`. Ia menjaga keputusan ini tetap disengaja: bila
-- seseorang kelak menyeragamkannya, uji itu yang akan menanyakan alasannya.
--
-- ============================================================================
-- ALIAS MENYESATKAN YANG DIBERI NAMA YANG BENAR (`D-19`)
-- ============================================================================
--
--     a.PYID                 AS "CaseID"       -> NO_SURVEY
--     a.PZINSKEY             AS "ClaimNo"      -> ID_SURVEY
--     a.POLICYNO             AS "CityID"       -> NO_POLIS
--     a.QQNAME               AS "Country"      -> NAMA_TERTANGGUNG
--     a.REFNO_1              AS "Province"     -> NO_REFERENSI
--     a.SURVEYORNAME_1       AS "CountryID"    -> NAMA_SURVEYOR
--     a.USERTEKNIS_1         AS "District"     -> PIC_TEKNIK
--     a.RescheduleLocation_1 AS "DistrictID"   -> LOKASI_SURVEI
--     a.AdjusterStatus_1     AS "ProvinceID"   -> STATUS_SURVEI
--     a.PYSTATUSWORK         AS "NamaSurveyor" -> STATUS_PROSES
--     a.AdjusterPIC_1        AS "City"         -> PIC_ADJUSTER
--     (subquery pyid induk)  AS "NamaDokterRCL"-> NO_KLAIM
--
-- Tiga di antaranya layak diperhatikan: `PYSTATUSWORK` dialiaskan menjadi `"NamaSurveyor"`,
-- `POLICYNO` menjadi `"CityID"`, dan nomor klaim induk menjadi `"NamaDokterRCL"`. Tidak satu
-- pun mencerminkan isinya.
--
-- ============================================================================
-- NOMOR KLAIM INDUK, BUKAN KUNCI INTERNAL PEGA
-- ============================================================================
--
-- `BrowseInternalSurveyor` mengembalikan `A.CASEID_1` apa adanya — kunci teknis Pega yang
-- memuat nama kelas internal (`ASM-FW-GCNMFW-WORK …`). Itu persis utang teknis §4.1: kunci
-- teknis Pega bocor ke data bisnis, dan `D-22` menetapkannya tidak dibawa ke layar.
--
-- Kedua kueri di sini karena itu memakai subquery yang sama — cara yang sudah dipakai
-- `BrowseLossAdjuster` — sehingga keduanya menampilkan NOMOR KLAIM yang terbaca pengguna.

-- ============================================================================
-- SUMBER BARU (2026-10-08): T_CLAIMLIST_ADMIN + T_SURVEYORLIST
-- ============================================================================
--
-- `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` dan `DATAPEGA.PC_ASSIGN_WORKLIST` tidak dipakai lagi
-- ("Perubahan nama tabel untuk Inbox.xlsx", kolom E):
--
--   objek kerja Work-PNC          -> POOLDATA.T_CLAIMLIST_ADMIN  (satu baris per klaim,
--                                    sudah memuat PXFLOWNAME/PXTASKLABEL penugasannya)
--   objek kerja Work-SurveyClaim  -> POOLDATA.T_SURVEYORLIST
--
-- Pemetaan kolom survei (nama di T_SURVEYORLIST TANPA akhiran `_1`, dibaca dari katalog
-- 2026-10-08 — BUKAN nama yang tercatat di docs/permintaan-kolom-t-surveyorlist.md):
--
--   PZINSKEY             -> CASEID                (kunci berprefix, `ASM-FW-GCNMFW-WORK SRV-…`)
--   PYID                 -> CASEID tanpa prefix
--   CASEID_1             -> PNCCASEID             (kunci klaim induk)
--   SURVEYORTYPE_1       -> SURVEYTYPE            (domain sama: 1 internal, 2 loss adjuster, 4)
--   SURVEYORNAME_1       -> SURVEYOR_NAME
--   ADJUSTERPIC_1        -> ADJUSTER_PIC
--   ADJUSTERSTATUS_1     -> STS_SURVEY
--   REFNO_1              -> REFNO
--   RESCHEDULELOCATION_1 -> RESCHEDULE_LOCATION
--   RESCHEDULEDATE_1     -> RESCHEDULE_DATE       (kini juga mengisi Tanggal Survei loss adjuster,
--                                                  yang di kueri lama selalu kosong)
--   PYSTATUSWORK         -> PYSTATUSWORK
--   PXCREATEDATETIME     -> MIN(TGLINPUT) per berkas — saat berkas survei pertama tercatat
--   POLICYNO, QQNAME, USERTEKNIS_1 -> dari klaim induk di T_CLAIMLIST_ADMIN
--
-- DUA PERBEDAAN BENTUK YANG MENGUBAH ANGKA, dan tidak dapat dihindari:
--
--   1. T_SURVEYORLIST adalah JEJAK PERKEMBANGAN — satu baris per perubahan status, bukan per
--      berkas. Setiap kueri di bawah membaca LANGKAH TERAKHIR tiap berkas (`STEP_RANK = 1`),
--      sama seperti modul inboxsurvey. Tanpa itu satu berkas tampil berkali-kali.
--   2. T_CLAIMLIST_ADMIN satu baris per klaim. Penggandaan baris akibat gabung ke
--      PC_ASSIGN_WORKLIST — yang menurut keputusan Work Owner 2026-09-26 dipertahankan apa
--      adanya — HILANG dengan sendirinya: tabel penggantinya tidak punya baris ganda.
--
-- `PYSTATUSWORK` berkas survei dibandingkan dengan bentuk yang menerima NULL
-- (`IS NULL OR NOT IN …`). Kolomnya baru diisi belakangan; tanpa itu `NOT IN` atas NULL
-- membuang SELURUH baris yang belum terisi tanpa galat apa pun.

-- name: loss_adjuster_count
-- Menghitung **JUMLAH KLAIM** yang punya sekurang-kurangnya satu berkas loss adjuster.
--
-- Satuannya KLAIM, sedangkan loss_adjuster_list mengembalikan berkas SURVEI — angka kartu
-- karena itu TIDAK akan sama dengan jumlah baris telusurnya, sama seperti di Pega.
--
-- Status klaim induk ikut sebagai syarat di dalam CASE, persis seperti aslinya: klaim yang
-- sudah selesai tetap menyumbang baris, hanya menyumbang nilai 0.
SELECT COALESCE(SUM(CASE
                    WHEN A.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
                     AND EXISTS (SELECT 1
                                   FROM POOLDATA.T_SURVEYORLIST s
                                  WHERE s.SURVEYTYPE = '2'
                                    AND s.PNCCASEID = A.PZINSKEY)
                    THEN 1
                    ELSE 0
                 END), 0)
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.BRANCHNAME <> 'ASNET'
   AND A.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1')
   AND A.PXTASKLABEL NOT IN ('FixCorrespondence')
   AND (:1 = 'ALL'
        OR (:2 = 'NONMBU'
            AND A.GROUPPANEL_1 IN ('003', '004', '006')
            AND c.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023'))
        OR (:3 = 'BONDING'
            AND c.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023'))
        OR (:4 = 'PA' AND A.GROUPPANEL_1 = '002')
        OR (:5 = 'TRAVEL' AND A.GROUPPANEL_1 = '005'))
   AND (:6 IS NULL
        OR UPPER(A.POLICYNO) LIKE :7 ESCAPE '\'
        OR UPPER(A.PYID) LIKE :8 ESCAPE '\')

-- name: loss_adjuster_list
-- Membaca satu halaman berkas survei loss adjuster — langkah terakhir tiap berkas.
SELECT s.CASEID                                         AS ID_SURVEY,
       REPLACE(s.CASEID, 'ASM-FW-GCNMFW-WORK ', '')     AS NO_SURVEY,
       z.PYID                                           AS NO_KLAIM,
       z.POLICYNO                                       AS NO_POLIS,
       z.QQNAME                                         AS NAMA_TERTANGGUNG,
       s.REFNO                                          AS NO_REFERENSI,
       s.SURVEYOR_NAME                                  AS NAMA_SURVEYOR,
       COALESCE(z.USERTEKNIS_1, p.PICTEKNIK)            AS PIC_TEKNIK,
       s.RESCHEDULE_LOCATION                            AS LOKASI_SURVEI,
       s.STS_SURVEY                                     AS STATUS_SURVEI,
       s.PYSTATUSWORK                                   AS STATUS_PROSES,
       s.ADJUSTER_PIC                                   AS PIC_ADJUSTER,
       s.RESCHEDULE_DATE                                AS TANGGAL_SURVEI,
       s.CREATED_AT                                     AS TANGGAL_TUGAS
  FROM (SELECT t.*,
               ROW_NUMBER() OVER (PARTITION BY t.CASEID
                                  ORDER BY LPAD(TRIM(t.INDEX_SURVEY), 10, '0') DESC NULLS LAST,
                                           t.TGLINPUT DESC NULLS LAST) AS STEP_RANK,
               MIN(t.TGLINPUT) OVER (PARTITION BY t.CASEID)            AS CREATED_AT
          FROM POOLDATA.T_SURVEYORLIST t) s
       INNER JOIN POOLDATA.T_CLAIMLIST_ADMIN z
               ON z.PZINSKEY = s.PNCCASEID
              AND z.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
       INNER JOIN POOLDATA.BUSINESS c
               ON z.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
       LEFT JOIN POOLDATA.T_CLAIM_PNC p
              ON p.CLAIMID = s.PNCCASEID
 WHERE s.STEP_RANK = 1
   AND s.SURVEYTYPE = '2'
   AND (s.PYSTATUSWORK IS NULL
        OR s.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected'))
   AND z.BRANCHNAME <> 'ASNET'
   AND z.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1')
   AND z.PXTASKLABEL NOT IN ('FixCorrespondence')
   AND (:1 = 'ALL'
        OR (:2 = 'NONMBU'
            AND z.GROUPPANEL_1 IN ('003', '004', '006')
            AND c.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023'))
        OR (:3 = 'BONDING'
            AND c.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023'))
        OR (:4 = 'PA' AND z.GROUPPANEL_1 = '002')
        OR (:5 = 'TRAVEL' AND z.GROUPPANEL_1 = '005'))
   AND (:6 IS NULL
        OR UPPER(z.POLICYNO) LIKE :7 ESCAPE '\'
        OR UPPER(s.CASEID) LIKE :8 ESCAPE '\')
 ORDER BY s.CREATED_AT ASC, s.CASEID
OFFSET :9 ROWS FETCH NEXT :10 ROWS ONLY

-- name: internal_surveyor_count
-- Menghitung **JUMLAH BERKAS SURVEI** internal yang masih terbuka, dijumlahkan per klaim
-- yang memilikinya — satuan yang BERBEDA dari padanan loss adjuster-nya, sama seperti di
-- Pega (keputusan Work Owner 2026-09-26).
--
-- Yang dihitung adalah BERKAS (langkah terakhir), bukan baris jejak: menghitung baris
-- T_SURVEYORLIST akan melipatgandakan angka dengan banyaknya langkah tiap berkas.
SELECT COALESCE(SUM(CASE
                    WHEN A.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
                    THEN (SELECT COUNT(*)
                            FROM (SELECT t.PNCCASEID, t.SURVEYTYPE, t.PYSTATUSWORK,
                                         ROW_NUMBER() OVER (PARTITION BY t.CASEID
                                                            ORDER BY LPAD(TRIM(t.INDEX_SURVEY), 10, '0') DESC NULLS LAST,
                                                                     t.TGLINPUT DESC NULLS LAST) AS STEP_RANK
                                    FROM POOLDATA.T_SURVEYORLIST t) s
                           WHERE s.STEP_RANK = 1
                             AND s.SURVEYTYPE = '1'
                             AND (s.PYSTATUSWORK IS NULL
                                  OR s.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected'))
                             AND s.PNCCASEID = A.PZINSKEY)
                    ELSE 0
                 END), 0)
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.BRANCHNAME <> 'ASNET'
   AND A.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1')
   AND A.PXTASKLABEL NOT IN ('FixCorrespondence')
   AND (:1 = 'ALL'
        OR (:2 = 'NONMBU'
            AND A.GROUPPANEL_1 IN ('003', '004', '006')
            AND c.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023'))
        OR (:3 = 'BONDING'
            AND c.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023'))
        OR (:4 = 'PA' AND A.GROUPPANEL_1 = '002')
        OR (:5 = 'TRAVEL' AND A.GROUPPANEL_1 = '005'))
   AND (:6 IS NULL
        OR UPPER(A.POLICYNO) LIKE :7 ESCAPE '\'
        OR UPPER(A.PYID) LIKE :8 ESCAPE '\')

-- name: internal_surveyor_list
-- Membaca satu halaman berkas survei internal — langkah terakhir tiap berkas.
--
-- Klaim induknya wajib belum selesai, dan penyaring lini bisnis berlaku atas klaim induk
-- itu (di Pega ia berlaku atas salinan kolom yang sama pada objek kerja survei).
SELECT s.CASEID                                         AS ID_SURVEY,
       REPLACE(s.CASEID, 'ASM-FW-GCNMFW-WORK ', '')     AS NO_SURVEY,
       z.PYID                                           AS NO_KLAIM,
       z.POLICYNO                                       AS NO_POLIS,
       z.QQNAME                                         AS NAMA_TERTANGGUNG,
       s.REFNO                                          AS NO_REFERENSI,
       s.SURVEYOR_NAME                                  AS NAMA_SURVEYOR,
       COALESCE(z.USERTEKNIS_1, p.PICTEKNIK)            AS PIC_TEKNIK,
       s.RESCHEDULE_LOCATION                            AS LOKASI_SURVEI,
       s.STS_SURVEY                                     AS STATUS_SURVEI,
       s.PYSTATUSWORK                                   AS STATUS_PROSES,
       s.SURVEYOR_NAME                                  AS PIC_ADJUSTER,
       s.RESCHEDULE_DATE                                AS TANGGAL_SURVEI,
       s.CREATED_AT                                     AS TANGGAL_TUGAS
  FROM (SELECT t.*,
               ROW_NUMBER() OVER (PARTITION BY t.CASEID
                                  ORDER BY LPAD(TRIM(t.INDEX_SURVEY), 10, '0') DESC NULLS LAST,
                                           t.TGLINPUT DESC NULLS LAST) AS STEP_RANK,
               MIN(t.TGLINPUT) OVER (PARTITION BY t.CASEID)            AS CREATED_AT
          FROM POOLDATA.T_SURVEYORLIST t) s
       INNER JOIN POOLDATA.T_CLAIMLIST_ADMIN z
               ON z.PZINSKEY = s.PNCCASEID
              AND z.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
       INNER JOIN POOLDATA.BUSINESS c
               ON z.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
       LEFT JOIN POOLDATA.T_CLAIM_PNC p
              ON p.CLAIMID = s.PNCCASEID
 WHERE s.STEP_RANK = 1
   AND s.SURVEYTYPE = '1'
   AND (s.PYSTATUSWORK IS NULL
        OR s.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected'))
   AND z.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (:1 = 'ALL'
        OR (:2 = 'NONMBU'
            AND z.GROUPPANEL_1 IN ('003', '004', '006')
            AND c.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023'))
        OR (:3 = 'BONDING'
            AND c.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023'))
        OR (:4 = 'PA' AND z.GROUPPANEL_1 = '002')
        OR (:5 = 'TRAVEL' AND z.GROUPPANEL_1 = '005'))
   AND (:6 IS NULL
        OR UPPER(z.POLICYNO) LIKE :7 ESCAPE '\'
        OR UPPER(s.CASEID) LIKE :8 ESCAPE '\')
 ORDER BY s.CREATED_AT ASC, s.CASEID
OFFSET :9 ROWS FETCH NEXT :10 ROWS ONLY
