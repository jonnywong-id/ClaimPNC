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
-- ============================================================================
-- SUMBER BERPINDAH KE POOLDATA.T_SURVEYORLIST (2026-10-12)
-- ============================================================================
--
-- Keempat rule Pega membaca `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`, tempat kasus survei hidup
-- sebagai BARIS ANAK klaim (`PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim'`), ditaut
-- `CASEID_1 -> PZINSKEY` induknya.
--
-- Skema `DATAPEGA` tidak dipakai lagi — keputusan Work Owner. Penggantinya ditetapkan
-- `Sample Form/Perubahan nama tabel untuk Inbox.xlsx`: keempat kueri ini pindah ke
-- **`POOLDATA.T_SURVEYORLIST`**, tabel bisnis yang memang menyimpan berkas survei.
--
-- # Kenapa pernah gagal, dan kenapa tidak terdeteksi
--
-- Pada 2026-10-08 berkas ini sempat diarahkan ke `T_CLAIMLIST_ADMIN` begitu saja. Tabel itu
-- TIDAK memuat baris kasus survei (nol baris ber-`Work-SurveyClaim`) dan tidak punya kolom
-- `CASEID_1`, sehingga kedua kueri gagal:
--
--     ORA-00904: "S"."CASEID_1": invalid identifier
--
-- Satu kueri gagal membuat SELURUH ringkasan menjawab 500, sehingga KEEMPAT kartu
-- menampilkan 0 — termasuk Outstanding yang sebenarnya sehat. Dan `-periksa` tetap HIJAU,
-- karena tidak ada satu pun probe yang menyentuh kueri survei. Probe `survey_check` di
-- bawah ada supaya kegagalan itu tidak dapat terulang diam-diam.
--
-- # Tautan ke klaim — diukur, bukan diduga
--
--     T_SURVEYORLIST.PNCCASEID  = T_CLAIMLIST_ADMIN.PZINSKEY     cocok 113 dari 113
--     T_SURVEYORLIST.CASEID     = kunci kasus SURVEI-nya sendiri  cocok 0 — BUKAN penaut
--
-- Keduanya berbentuk kunci penuh Pega dan mudah tertukar. Yang menaut ke klaim adalah
-- `PNCCASEID`. Keduanya sudah huruf besar tanpa spasi di ujung (280 dari 280 dan 2.890 dari
-- 2.890), sehingga `UPPER`/`TRIM` TIDAK dipasang — memasangnya hanya mematikan index.
--
-- # Satu berkas survei punya banyak LANGKAH
--
--     280 baris · 189 CASEID unik · (CASEID, INDEX_SURVEY) unik 280 · maksimum 13 langkah
--
-- Jadi butiran barisnya adalah LANGKAH, bukan berkas. Layar menampilkan satu baris per
-- BERKAS, dan yang diambil **langkah terakhir** — ditetapkan `Sample Form` ("langkah
-- terakhir per berkas"). Tanpa itu satu berkas bersurvei ulang muncul berkali-kali.
--
-- `INDEX_SURVEY` bertipe `VARCHAR2(5)` berisi angka, sehingga diurutkan dengan `LPAD(...,5,'0')`
-- agar '10' tidak mendahului '9'. `LPAD` dipilih, bukan `TO_NUMBER`, karena `TO_NUMBER` akan
-- MELEDAK bila kelak ada satu isi non-angka, dan karena ia portabel (`D-20`) sementara
-- `DEFAULT NULL ON CONVERSION ERROR` khas Oracle.
--
-- ============================================================================
-- PEMETAAN KOLOM — DIBACA DARI KATALOG, BUKAN DARI DOKUMEN
-- ============================================================================
--
--     Pega (PC_ASM_FW_GCNMFW_WORK)   T_SURVEYORLIST          terisi di DEV 2026-10-12
--     SURVEYORTYPE_1                 SURVEYTYPE              279/280
--     SURVEYORNAME_1                 SURVEYOR_NAME           266/280
--     ADJUSTERPIC_1                  ADJUSTER_PIC            151/280
--     ADJUSTERSTATUS_1               STS_SURVEY              279/280
--     REFNO_1                        REFNO                   130/280
--     RESCHEDULELOCATION_1           RESCHEDULE_LOCATION     279/280
--     RESCHEDULEDATE_1               RESCHEDULE_DATE           1/280   <- lihat catatan
--     PYSTATUSWORK                   PYSTATUSWORK            278/280
--
-- `docs/permintaan-kolom-t-surveyorlist.md` menyebut kelimanya masih KOSONG; itu benar pada
-- 2026-10-05 dan sudah tidak berlaku. Yang dipakai di sini angka katalog hari ini.
--
-- # TANGGAL SURVEI — satu keputusan yang belum final
--
-- Padanan harfiah `RESCHEDULEDATE_1` adalah `RESCHEDULE_DATE`, tetapi kolom itu terisi
-- **1 dari 280**; memakainya membuat kolom "Tanggal Survei" kosong hampir selalu. Yang
-- dipakai di sini **`SURVEYDATE`** (128 dari 280) — kolom yang namanya memang tanggal survei,
-- dan sejalan dengan catatan `Sample Form`: *"Tanggal Survei loss adjuster kini terisi (dulu
-- selalu kosong)"*, yang mustahil terjadi lewat kolom berisi satu baris.
--
-- **BELUM DIPUTUSKAN — pertanyaan terbuka** (pemilik: Work Owner). Keduanya berbeda arti:
-- `RESCHEDULE_DATE` tanggal janji ulang, `SURVEYDATE` tanggal survei dikerjakan. Bila Work
-- Owner memilih yang pertama, yang diganti hanya DUA baris `AS TANGGAL_SURVEI` di bawah.
--
-- ============================================================================
-- KETIGA KETIDAKSEJAJARAN PEGA YANG TETAP DIPERTAHANKAN (`P-5`)
-- ============================================================================
--
--   a. PENYARING LINI BISNIS dulu berada di tempat berbeda — pada Loss Adjuster di dalam
--      EXISTS menyaring KLAIM INDUK, pada Internal Surveyor di luar menyaring BARIS SURVEI.
--      Perbedaan itu KINI TIDAK DAPAT DIPERTAHANKAN: kolom lini bisnis hanya ada pada baris
--      klaim, sehingga keduanya menyaring klaim. Ini SATU-SATUNYA ketidaksejajaran yang
--      hilang, dan hilangnya karena sumbernya berubah — bukan karena diseragamkan.
--
--   b. STATUS KLAIM INDUK hanya diperiksa di satu sisi. `BrowseInternalSurveyor` menuntut
--      induknya belum selesai; `BrowseLossAdjuster` tidak. DIPERTAHANKAN.
--
--   c. STATUS SURVEI hanya diperiksa di satu sisi pada kueri hitung.
--      `Get_CountInternalSurveyor` membuang survei yang sudah selesai; padanan loss
--      adjuster-nya tidak. DIPERTAHANKAN.
--
-- ============================================================================
-- KEDUA KUERI HITUNG MENGHITUNG SATUAN BERBEDA — KEPUTUSAN WORK OWNER 2026-09-26
-- ============================================================================
--
--     Get_CountLostAdjusterClaim   menghitung JUMLAH KLAIM yang punya survei adjuster
--     Get_CountInternalSurveyor    menghitung JUMLAH SURVEI
--
-- Padahal kedua kartu menelusur ke daftar yang sama-sama berisi BERKAS SURVEI.
-- Penyeragaman sempat dikerjakan lalu DICABUT: keduanya mengikuti Pega apa adanya (`P-5`),
-- sehingga gerbang 1 dapat membandingkannya tanpa pengecualian.
--
-- Akibat yang dinyatakan di layar: **angka kartu tidak akan sama dengan jumlah baris
-- telusurnya.** Dijaga `TestCountAndListMayDifferForSurveyTiles`.
--
-- Satu penggandaan LAMA ikut hilang: kedua kueri hitung Pega menggabung ke
-- `PC_ASSIGN_WORKLIST` sementara `SUM` berjalan atas hasil gabungan, sehingga klaim dengan
-- lebih dari satu penugasan terhitung berkali-kali. Gabungan itu sudah tidak ada.
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
-- ============================================================================
-- NOMOR SURVEI, BUKAN KUNCI INTERNAL PEGA
-- ============================================================================
--
-- `T_SURVEYORLIST.CASEID` berisi kunci teknis Pega lengkap dengan nama kelas internal
-- (`ASM-FW-GCNMFW-WORK …`) — persis utang teknis §4.1, dan `D-22` menetapkannya tidak
-- dibawa ke layar. Yang ditampilkan karena itu EKORNYA setelah spasi, yaitu nomor survei
-- yang terbaca pengguna. Kunci utuhnya tetap dikembalikan sebagai `ID_SURVEY`, dipakai
-- sebagai kunci baris di antarmuka, bukan untuk dibaca orang.

-- name: loss_adjuster_count
-- Angka kartu LOSS ADJUSTER — `RDB List/Get_CountLostAdjusterClaim-SQL.xml`.
--
-- Menghitung JUMLAH KLAIM yang punya sekurang-kurangnya satu berkas survei adjuster, BUKAN
-- jumlah berkasnya (lihat catatan kepala). Status berkas survei TIDAK diperiksa di sini —
-- ketidaksejajaran (c) yang dipertahankan.
SELECT COUNT(DISTINCT A.PZINSKEY)
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.BRANCHNAME <> 'ASNET'
   AND A.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1')
   AND A.PXTASKLABEL NOT IN ('FixCorrespondence')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_SURVEYORLIST s
                WHERE s.PNCCASEID = A.PZINSKEY
                  AND s.SURVEYTYPE = '2')
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
-- Telusur LOSS ADJUSTER — `RDB List/BrowseLossAdjuster-SQL.xml`.
--
-- Satu baris per BERKAS survei, diambil dari LANGKAH TERAKHIRNYA. Status klaim induk TIDAK
-- diperiksa — ketidaksejajaran (b) yang dipertahankan.
WITH berkas AS (
  SELECT s.CASEID,
         s.PNCCASEID,
         s.REFNO,
         s.SURVEYOR_NAME,
         s.RESCHEDULE_LOCATION,
         s.STS_SURVEY,
         s.PYSTATUSWORK,
         s.ADJUSTER_PIC,
         s.SURVEYDATE,
         s.TGLINPUT,
         ROW_NUMBER() OVER (PARTITION BY s.CASEID
                                ORDER BY LPAD(s.INDEX_SURVEY, 5, '0') DESC) AS langkah_terakhir
    FROM POOLDATA.T_SURVEYORLIST s
   WHERE s.SURVEYTYPE = '2'
)
SELECT b.CASEID                                             AS ID_SURVEY,
       b.CASEID                                             AS NO_SURVEY,
       A.PYID                                               AS NO_KLAIM,
       A.POLICYNO                                           AS NO_POLIS,
       A.QQNAME                                             AS NAMA_TERTANGGUNG,
       b.REFNO                                              AS NO_REFERENSI,
       b.SURVEYOR_NAME                                      AS NAMA_SURVEYOR,
       A.USERTEKNIS_1                                       AS PIC_TEKNIK,
       b.RESCHEDULE_LOCATION                                AS LOKASI_SURVEI,
       b.STS_SURVEY                                         AS STATUS_SURVEI,
       b.PYSTATUSWORK                                       AS STATUS_PROSES,
       b.ADJUSTER_PIC                                       AS PIC_ADJUSTER,
       b.SURVEYDATE                                         AS TANGGAL_SURVEI,
       b.TGLINPUT                                           AS TANGGAL_TUGAS
  FROM berkas b
       INNER JOIN POOLDATA.T_CLAIMLIST_ADMIN A
               ON A.PZINSKEY = b.PNCCASEID
              AND A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE b.langkah_terakhir = 1
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
 ORDER BY b.TGLINPUT ASC, b.CASEID
OFFSET :9 ROWS FETCH NEXT :10 ROWS ONLY

-- name: internal_surveyor_count
-- Angka kartu INTERNAL SURVEYOR — `RDB List/Get_CountInternalSurveyor-SQL.xml`.
--
-- Menghitung JUMLAH BERKAS survei, bukan jumlah klaim — berbeda dari padanan loss
-- adjuster-nya, dan itu disengaja.
--
-- Berkas yang sudah SELESAI dibuang — ketidaksejajaran (c). `PYSTATUSWORK` kosong dianggap
-- MASIH TERBUKA, mengikuti `Sample Form` no. 9: berkas berstatus kosong ikut tampil sebagai
-- terbuka. Tanpa penegasan itu, `NOT IN` atas NULL menghasilkan NULL dan barisnya terbuang
-- diam-diam.
SELECT COUNT(*)
  FROM (SELECT s.CASEID,
               s.PNCCASEID,
               s.PYSTATUSWORK,
               ROW_NUMBER() OVER (PARTITION BY s.CASEID
                                      ORDER BY LPAD(s.INDEX_SURVEY, 5, '0') DESC) AS langkah_terakhir
          FROM POOLDATA.T_SURVEYORLIST s
         WHERE s.SURVEYTYPE = '1') b
       INNER JOIN POOLDATA.T_CLAIMLIST_ADMIN A
               ON A.PZINSKEY = b.PNCCASEID
              AND A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE b.langkah_terakhir = 1
   AND (b.PYSTATUSWORK IS NULL
        OR b.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected'))
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
-- Telusur INTERNAL SURVEYOR — `RDB List/BrowseInternalSurveyor-SQL.xml`.
--
-- Berbeda dari telusur loss adjuster, kueri ini MENUNTUT KLAIM INDUKNYA BELUM SELESAI —
-- ketidaksejajaran (b) yang dipertahankan.
WITH berkas AS (
  SELECT s.CASEID,
         s.PNCCASEID,
         s.REFNO,
         s.SURVEYOR_NAME,
         s.RESCHEDULE_LOCATION,
         s.STS_SURVEY,
         s.PYSTATUSWORK,
         s.ADJUSTER_PIC,
         s.SURVEYDATE,
         s.TGLINPUT,
         ROW_NUMBER() OVER (PARTITION BY s.CASEID
                                ORDER BY LPAD(s.INDEX_SURVEY, 5, '0') DESC) AS langkah_terakhir
    FROM POOLDATA.T_SURVEYORLIST s
   WHERE s.SURVEYTYPE = '1'
)
SELECT b.CASEID                                             AS ID_SURVEY,
       b.CASEID                                             AS NO_SURVEY,
       A.PYID                                               AS NO_KLAIM,
       A.POLICYNO                                           AS NO_POLIS,
       A.QQNAME                                             AS NAMA_TERTANGGUNG,
       b.REFNO                                              AS NO_REFERENSI,
       b.SURVEYOR_NAME                                      AS NAMA_SURVEYOR,
       A.USERTEKNIS_1                                       AS PIC_TEKNIK,
       b.RESCHEDULE_LOCATION                                AS LOKASI_SURVEI,
       b.STS_SURVEY                                         AS STATUS_SURVEI,
       b.PYSTATUSWORK                                       AS STATUS_PROSES,
       b.ADJUSTER_PIC                                       AS PIC_ADJUSTER,
       b.SURVEYDATE                                         AS TANGGAL_SURVEI,
       b.TGLINPUT                                           AS TANGGAL_TUGAS
  FROM berkas b
       INNER JOIN POOLDATA.T_CLAIMLIST_ADMIN A
               ON A.PZINSKEY = b.PNCCASEID
              AND A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE b.langkah_terakhir = 1
   AND A.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
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
 ORDER BY b.TGLINPUT ASC, b.CASEID
OFFSET :9 ROWS FETCH NEXT :10 ROWS ONLY

-- name: survey_check
-- Membuktikan kedua kueri survei dapat di-parse — termasuk kolom berkas surveinya.
--
-- Probe ini lahir dari kegagalan nyata 2026-10-08: `ORA-00904 "S"."CASEID_1"` membuat
-- `/ringkasan` menjawab 500 dan KEEMPAT kartu menampilkan 0, sementara `-periksa` tetap
-- hijau seluruhnya karena tidak ada probe yang menyentuh kueri survei.
--
-- Kolomnya disebut satu per satu, bukan `SELECT 1`: probe yang hanya membuktikan tabelnya
-- ada akan LULUS terhadap kolom yang tidak ada, dan itu persis kegagalan di atas.
--
-- Gabungannya ikut disertakan supaya `PNCCASEID` dan `PZINSKEY` terbukti SEPADAN, bukan
-- hanya ada — tautan antar dua tabel yang tidak pernah diuji adalah tautan yang belum
-- terbukti. `WHERE 1 = 0` membuatnya tidak membaca satu baris pun.
SELECT s.CASEID, s.PNCCASEID, s.INDEX_SURVEY, s.SURVEYTYPE,
       s.SURVEYOR_NAME, s.ADJUSTER_PIC, s.STS_SURVEY, s.REFNO,
       s.RESCHEDULE_LOCATION, s.RESCHEDULE_DATE, s.PYSTATUSWORK,
       s.SURVEYDATE, s.TGLINPUT,
       A.PZINSKEY, A.PYID, A.POLICYNO, A.QQNAME, A.USERTEKNIS_1,
       A.PYSTATUSWORK, A.BRANCHNAME, A.PXFLOWNAME, A.PXTASKLABEL,
       A.BUSINESSCODE_1, A.GROUPPANEL_1, A.PXOBJCLASS
  FROM POOLDATA.T_SURVEYORLIST s
       INNER JOIN POOLDATA.T_CLAIMLIST_ADMIN A
               ON A.PZINSKEY = s.PNCCASEID
 WHERE 1 = 0
