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

-- name: loss_adjuster_count
-- Menghitung baris survei loss adjuster yang cocok.
--
-- Menghitung **JUMLAH KLAIM** yang punya sekurang-kurangnya satu survei loss adjuster.
--
-- Bentuknya mengikuti `RDB List/Get_CountLostAdjusterClaim-SQL.xml` apa adanya — keputusan
-- Work Owner 2026-09-26. Tiga akibatnya dinyatakan di sini supaya tidak dibaca sebagai cacat:
--
--   1. Satuannya KLAIM, sedangkan loss_adjuster_list mengembalikan baris SURVEI. Angka kartu
--      karena itu TIDAK akan sama dengan jumlah baris telusurnya, dan memang begitu di Pega.
--   2. Penyaring lini bisnis dan kotak cari berlaku atas KLAIM induk, bukan atas baris survei
--      — berbeda dari kueri daftarnya, dan itu pun apa adanya.
--   3. Gabung ke PC_ASSIGN_WORKLIST menggandakan baris klaim yang memegang lebih dari satu
--      penugasan, dan SUM berjalan di atas baris hasil gabungan itu. Klaim seperti itu
--      terhitung berkali-kali.
--
-- Status klaim induk TIDAK disaring di WHERE luar, melainkan ikut sebagai syarat di dalam
-- CASE — persis seperti aslinya. Akibatnya klaim yang sudah selesai tetap menyumbang baris,
-- hanya menyumbang nilai 0.
--
-- COALESCE ditambahkan karena SUM atas nol baris menghasilkan NULL, dan NULL tidak dapat
-- dipindai ke int di Go. Ia tidak mengubah angka yang dibaca pengguna — layar lama pun
-- menampilkan kosong sebagai nol.
--
-- ORDER BY pada kueri aslinya TIDAK dibawa: pada SELECT beragregat tanpa GROUP BY ia tidak
-- mengurutkan apa pun, dan Oracle menolaknya dengan ORA-00979.
SELECT COALESCE(SUM(CASE
                    WHEN (SELECT COUNT(*)
                            FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK s
                           WHERE s.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim'
                             AND s.SURVEYORTYPE_1 = '2'
                             AND s.CASEID_1 = A.PZINSKEY
                             AND A.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')) > 0
                    THEN 1
                    ELSE 0
                 END), 0)
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST B
               ON A.PZINSKEY = B.PXREFOBJECTKEY
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.BRANCHNAME <> 'ASNET'
   AND B.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1')
   AND B.PXTASKLABEL NOT IN ('FixCorrespondence')
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
-- Membaca satu halaman survei loss adjuster.
SELECT A.PZINSKEY             AS ID_SURVEY,
       A.PYID                 AS NO_SURVEY,
       (SELECT p.PYID
          FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK p
         WHERE p.PZINSKEY = A.CASEID_1
           AND p.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC') AS NO_KLAIM,
       A.POLICYNO             AS NO_POLIS,
       A.QQNAME               AS NAMA_TERTANGGUNG,
       A.REFNO_1              AS NO_REFERENSI,
       A.SURVEYORNAME_1       AS NAMA_SURVEYOR,
       A.USERTEKNIS_1         AS PIC_TEKNIK,
       A.RESCHEDULELOCATION_1 AS LOKASI_SURVEI,
       A.ADJUSTERSTATUS_1     AS STATUS_SURVEI,
       A.PYSTATUSWORK         AS STATUS_PROSES,
       A.ADJUSTERPIC_1        AS PIC_ADJUSTER,
       CAST(NULL AS DATE)     AS TANGGAL_SURVEI,
       A.PXCREATEDATETIME     AS TANGGAL_TUGAS
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim'
   AND A.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND A.SURVEYORTYPE_1 = '2'
   AND EXISTS (SELECT 1
                 FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK z
                      INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST b
                              ON z.PZINSKEY = b.PXREFOBJECTKEY
                      INNER JOIN POOLDATA.BUSINESS c
                              ON z.BUSINESSCODE_1 = c.ID
                      INNER JOIN POOLDATA.BUSINESSGROUP d
                              ON c.BUSINESSGROUPID = d.ID
                WHERE z.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
                  AND z.BRANCHNAME <> 'ASNET'
                  AND b.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1')
                  AND b.PXTASKLABEL NOT IN ('FixCorrespondence')
                  AND A.CASEID_1 = z.PZINSKEY
                  AND (:1 = 'ALL'
                       OR (:2 = 'NONMBU'
                           AND z.GROUPPANEL_1 IN ('003', '004', '006')
                           AND c.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023'))
                       OR (:3 = 'BONDING'
                           AND c.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023'))
                       OR (:4 = 'PA' AND z.GROUPPANEL_1 = '002')
                       OR (:5 = 'TRAVEL' AND z.GROUPPANEL_1 = '005')))
   AND (:6 IS NULL
        OR UPPER(A.POLICYNO) LIKE :7 ESCAPE '\'
        OR UPPER(A.PYID) LIKE :8 ESCAPE '\')
 ORDER BY A.PXCREATEDATETIME ASC, A.PZINSKEY
OFFSET :9 ROWS FETCH NEXT :10 ROWS ONLY

-- name: internal_surveyor_count
-- Menghitung **JUMLAH SURVEI** internal, dijumlahkan per klaim yang memilikinya.
--
-- Bentuknya mengikuti `RDB List/Get_CountInternalSurveyor-SQL.xml` apa adanya — keputusan
-- Work Owner 2026-09-26.
--
-- Perhatikan ia TIDAK sejajar dengan padanan loss adjuster-nya, dan ketidaksejajaran itu ada
-- di sistem lama:
--
--   * yang ini menjumlahkan BANYAKNYA survei; yang itu menghitung 1 per klaim;
--   * yang ini menyaring status survei di dalam subkueri; yang itu tidak menyaringnya.
--
-- Subkueri yang sama ditulis DUA KALI — sekali pada syarat WHEN, sekali sebagai nilai THEN —
-- persis seperti aslinya. Menggantinya dengan satu subkueri akan mengubah bentuk rencana
-- eksekusi, dan angka yang dihasilkan memang sama; ia dibiarkan supaya kesetaraannya dengan
-- kueri lama dapat dibaca berdampingan tanpa menafsirkan.
--
-- Penyaring lini bisnis dan kotak cari berlaku atas KLAIM induk, sedangkan kueri daftarnya
-- menyaring baris survei. Akibatnya angka kartu dan jumlah baris telusur dapat berbeda.
SELECT COALESCE(SUM(CASE
                    WHEN (SELECT COUNT(*)
                            FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK s
                           WHERE s.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim'
                             AND s.SURVEYORTYPE_1 = '1'
                             AND s.PYSTATUSWORK <> 'Resolved-Completed'
                             AND s.PYSTATUSWORK <> 'Resolved-Rejected'
                             AND s.CASEID_1 = A.PZINSKEY
                             AND A.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')) > 0
                    THEN (SELECT COUNT(*)
                            FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK s
                           WHERE s.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim'
                             AND s.SURVEYORTYPE_1 = '1'
                             AND s.PYSTATUSWORK <> 'Resolved-Completed'
                             AND s.PYSTATUSWORK <> 'Resolved-Rejected'
                             AND s.CASEID_1 = A.PZINSKEY
                             AND A.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected'))
                    ELSE 0
                 END), 0)
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST B
               ON A.PZINSKEY = B.PXREFOBJECTKEY
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.BRANCHNAME <> 'ASNET'
   AND B.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1')
   AND B.PXTASKLABEL NOT IN ('FixCorrespondence')
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
-- Membaca satu halaman survei internal.
--
-- Gabung ke `PC_ASSIGN_WORKLIST` pada kueri lama TIDAK dibawa. Di sana ia dipakai hanya
-- untuk mengambil `B.PXFLOWNAME` sebagai kolom `"RefNo"` — kolom yang tidak digambar
-- `Section/DashboardClaim_Section1-Section.xml`. Gabungnya sendiri menggandakan baris survei
-- untuk klaim yang memegang lebih dari satu penugasan, sehingga membawanya berarti menampilkan
-- survei yang sama berkali-kali demi kolom yang tidak dibaca siapa pun.
SELECT A.PZINSKEY             AS ID_SURVEY,
       A.PYID                 AS NO_SURVEY,
       (SELECT p.PYID
          FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK p
         WHERE p.PZINSKEY = A.CASEID_1
           AND p.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC') AS NO_KLAIM,
       A.POLICYNO             AS NO_POLIS,
       A.QQNAME               AS NAMA_TERTANGGUNG,
       A.REFNO_1              AS NO_REFERENSI,
       A.SURVEYORNAME_1       AS NAMA_SURVEYOR,
       A.USERTEKNIS_1         AS PIC_TEKNIK,
       A.RESCHEDULELOCATION_1 AS LOKASI_SURVEI,
       A.ADJUSTERSTATUS_1     AS STATUS_SURVEI,
       A.PYSTATUSWORK         AS STATUS_PROSES,
       A.SURVEYORNAME_1       AS PIC_ADJUSTER,
       A.RESCHEDULEDATE_1     AS TANGGAL_SURVEI,
       A.PXCREATEDATETIME     AS TANGGAL_TUGAS
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim'
   AND A.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND A.SURVEYORTYPE_1 = '1'
   AND EXISTS (SELECT 1
                 FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK z
                WHERE z.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
                  AND z.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
                  AND A.CASEID_1 = z.PZINSKEY)
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
 ORDER BY A.PXCREATEDATETIME ASC, A.PZINSKEY
OFFSET :9 ROWS FETCH NEXT :10 ROWS ONLY
