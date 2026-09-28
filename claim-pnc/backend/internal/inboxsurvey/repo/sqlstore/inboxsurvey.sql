-- Kueri modul My Work (MENU_ID 50) — antrean kerja Surveyor dan Loss Adjuster.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- SELURUH tabel yang dibaca berkas ini milik sistem lama. Tidak ada satu pun pernyataan yang
-- menulis, dan memang tidak boleh ada: selama masa paralel setiap tabel hanya boleh ditulis
-- SATU sistem, dan tabel-tabel ini milik Pega (`P-1`).
--
-- ============================================================================
-- YANG HARUS DIBACA DBA LEBIH DULU
-- ============================================================================
--
-- ## A. SUMBER DATANYA BERGESER DARI PEGA, DAN ITU KEPUTUSAN — BUKAN KELALAIAN
--
-- Di Pega, SELURUH kueri layar ini membaca:
--
--   DATAPEGA.PC_ASM_FW_GCNMFW_WORK  WHERE PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim'
--
-- Tetapi `POOLDATA.T_CLAIMLIST_ADMIN` — tabel yang Work Owner tetapkan menggantikan tabel
-- DATAPEGA — **tidak memuat satu pun baris `Work-SurveyClaim`**. Isinya 870 `Work-PNC` dan
-- 142 `Work-ReceiveDocument`; sudah diverifikasi langsung ke basis data dan tercatat di
-- `internal/inboxoutstanding/inboxoutstanding.go`.
--
-- Keputusan Work Owner 2026-09-28:
--
--   POOLDATA.T_SURVEYORLIST    menggerakkan baris  (satu baris per JANJI SURVEI)
--   POOLDATA.T_CLAIMLIST_ADMIN menyediakan header  (satu baris per KLAIM)
--
-- Kunci sambungnya `s.PNCCASEID = k.PZINSKEY`, dan itu BUKAN tebakan —
-- `RDB List/BroswseKlaimByNoSurvey-SQL.xml` memakainya persis begitu:
--
--   where CLAIMID = (select pnccaseid from t_surveyorlist
--                     where caseid = 'ASM-FW-GCNMFW-WORK ' || {InputData.CARI4})
--
-- ## B. AKIBAT YANG HARUS DISADARI — DUA BUTIR GRANULARITAS
--
-- 1. `ADJUSTERACCEPT_1`, `ADJUSTERSTATUS_1`, `REFNO_1`, dan `ADJUSTERPIC_1` hidup di
--    `T_CLAIMLIST_ADMIN`, yaitu PER KLAIM. Di Pega keempatnya ada pada objek SurveyClaim,
--    yaitu PER JANJI SURVEI.
--
--    Akibatnya: klaim dengan DUA janji survei yang statusnya berbeda akan menampilkan
--    status yang sama pada kedua barisnya. Itu tidak menghasilkan galat, dan tidak terlihat
--    di layar — ia hanya salah. Bila kelak terbukti mengganggu, penyelesaiannya adalah
--    meminta kolom-kolom itu ikut dipindahkan ke `T_SURVEYORLIST`, BUKAN menebaknya di sini.
--
-- 2. `T_SURVEYORLIST.STS_SURVEY` TIDAK dipakai menyaring apa pun. Nilainya tidak terbaca
--    dari export — hanya `'1'` yang muncul satu kali — sehingga memakainya berarti menebak.
--    Ia tetap DIBACA supaya domainnya terlihat dari data nyata, dan tab Close dapat
--    dikoreksi bila ternyata ia acuan yang benar.
--
-- ## C. DUA KOLOM BELUM TERKONFIRMASI — DAN `-periksa` YANG MENEMUKANNYA
--
--   ADJUSTERPIC_1   dipetakan ke kolom "Appointment No"   ** PERLU KONFIRMASI **
--   LOSSTYPE        dipetakan ke kolom "Cause Of Loss"    ** PERLU KONFIRMASI **
--
-- `ADJUSTERPIC_1` NAMANYA berbunyi "PIC", bukan nomor janji. Yang memetakannya ke
-- "Appointment No" adalah rantai berikut, dan rantai itu putus di satu tempat:
--
--   Section/InboxSurvey_section-Section.xml   "Appointment No"  <- properti `.City`
--   RDB List/BrowseLossAdjuster-SQL.xml       `a.AdjusterPIC_1 as "City"`
--
-- Yang putus: alias itu milik `BrowseLossAdjuster`, sedangkan yang benar-benar mengisi grid
-- adalah salah satu dari EMPAT Browse rule yang HILANG dari export (lihat §D). Jadi
-- pemetaannya masuk akal tetapi belum terbukti.
--
-- `LOSSTYPE` dipilih karena `BrowseLossAdjuster` tidak mengambil Cause Of Loss sama sekali,
-- dan `LOSSTYPE` adalah satu-satunya kolom pada tabel penggerak yang menyatakan jenis
-- kerugian (`Database/INSERT_SURVEYORLIST.prc`, parameter `TLOSSTYPE`).
--
-- Satu kueri katalog menutup keduanya:
--
--   SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE, NUM_DISTINCT
--     FROM ALL_TAB_COLUMNS
--    WHERE OWNER = 'POOLDATA'
--      AND ((TABLE_NAME = 'T_CLAIMLIST_ADMIN' AND COLUMN_NAME LIKE '%ADJUSTER%')
--        OR (TABLE_NAME = 'T_SURVEYORLIST'    AND COLUMN_NAME LIKE '%LOSS%'));
--
-- BILA KOLOMNYA TIDAK ADA, kueri di bawah gagal dengan ORA-00904 yang MENYEBUT NAMA
-- KOLOMNYA. Itu disengaja. Alternatifnya — menghilangkan kolomnya supaya kuerinya jalan —
-- akan menampilkan dua sel kosong tanpa seorang pun tahu kenapa.
--
-- ## D. EMPAT KUERI TAB HILANG DARI EXPORT (`R-16`)
--
-- `Activity/SetTempLostAdjuster-Act.xml` memanggil empat rule yang tidak satu pun ada:
--
--   BrowseOSLostAdjuster · BrowseConfirmLostAdjuster
--   BrowseCommunicationLostAdjuster · BrowseCloseLostAdjuster
--
-- Diperiksa lewat isi `pyRuleName`, bukan lewat nama berkas.
--
-- Yang menyelamatkan modul ini: **predikat keempatnya tetap terbaca**, karena
-- `RDB List/CountOSLostAdjuster-SQL.xml` menghitung ketujuh keranjang yang sama dalam SATU
-- kueri lewat tujuh `SUM(CASE WHEN …)`. Kueri `count_tabs` di bawah adalah penerjemahan
-- langsung darinya, keranjang demi keranjang.
--
-- Yang TIDAK terbaca hanyalah daftar SELECT dan urutan masing-masing Browse rule. Untuk itu
-- `BrowseLossAdjuster` menjadi rujukan terdekat.
--
-- ============================================================================
-- PEMETAAN KOLOM — judul di layar -> properti Pega -> kolom sebenarnya
-- ============================================================================
--
-- Judul dari `Section/InboxSurvey_section-Section.xml`; properti dari daftar Property-Set
-- pada `Activity/SetTempLostAdjuster-Act.xml`; kolom dari kedua tabel penggerak.
--
--   judul di layar      properti Pega            kolom                        alias
--   ------------------- ------------------------ ---------------------------- -------------------
--   Appointment No      .City                    k.ADJUSTERPIC_1  **?**       APPOINTMENT_NUMBER
--   Reference No        .AlasanDokterRejectRCL   k.REFNO_1                    REFERENCE_NUMBER
--   Claim No            .UserName                k.PYID                       CLAIM_NUMBER
--   Policy No           .Country                 k.POLICYNO                   POLICY_NUMBER
--   Insured Name        .AnalystDoctorRemaks     k.QQNAME                     INSURED_NAME
--   COB                 .KomiteStatus            k.BUSINESSNAME               CLASS_OF_BUSINESS
--   Cause Of Loss       .CauseOfLoss             s.LOSSTYPE  **?**            CAUSE_OF_LOSS
--   Location            .Location                s.LOCATION_SURVEY            LOCATION
--   PIC ASM             .UserTeknis              k.USERTEKNIS_1               TECHNICAL_PIC
--   PIC Loss Adjuster   .AnaylstRemarks          s.SURVEYOR_NAME              ADJUSTER_PIC
--   Date of Loss        .DateOfLoss              k.DATEOFLOSS_1               DATE_OF_LOSS
--   Aging               .CPLValidDate            k.AGING                      AGING_DAYS
--   Status ASM          .UserAdmin               k.ADJUSTERSTATUS_1           ASM_STATUS
--
--   tidak digambar      —                        s.CASEID                     SURVEY_ID
--   tidak digambar      —                        s.PNCCASEID                  CLAIM_ID
--   tidak digambar      —                        s.INDEX_SURVEY               SURVEY_INDEX
--   tidak digambar      —                        s.SURVEYTYPE                 SURVEYOR_TYPE
--   tidak digambar      —                        s.STS_SURVEY                 SURVEY_STATUS
--
-- CATATAN "SURVEYOR_TYPE". Ia dibaca dari `s.SURVEYTYPE`, BUKAN dari `k.SURVEYORTYPE_1`, dan
-- itu bukan sekadar pilihan gaya: `SURVEYTYPE` berlaku PER JANJI SURVEI sementara kolom pada
-- tabel klaim berlaku per klaim. Keduanya memang nilai yang sama, dan jalur penulisnya
-- terbaca utuh:
--
--   newWorkCover.ClaimData.SurveyData.SurveyorType   properti yang menjadi SURVEYORTYPE_1
--     -> Param.SurveyType         Activity/KomitePost_Survey-Act.xml:14950
--     -> TempSurvey.SurveyorType  Activity/SetSurveyorList-Act.xml:657
--     -> TSRVTYPE                 RDB List/CallProcedureInsertSurvey-SQL.xml
--     -> SURVEYTYPE               Database/INSERT_SURVEYORLIST.prc:32
--
-- Klaim dengan DUA janji survei berjenis berbeda — satu internal, satu loss adjuster —
-- karena itu menampilkan jenis yang benar pada masing-masing barisnya.
--
-- CATATAN "Claim No". Di Pega isinya `@substring(.CaseID,19,30)` — PZINSKEY dipotong mulai
-- karakter ke-19 untuk membuang awalan `ASM-FW-GCNMFW-WORK `. Di sini `k.PYID` dipakai
-- langsung: ia SUDAH nomor klaim yang terbaca manusia, dan memotong string adalah cara
-- sistem lama mengatasi ketiadaan kolom itu — bukan aturan bisnis yang perlu dibawa.
--
-- CATATAN "Aging". Kolom `AGING` bertipe NUMBER dan SUDAH dipakai `inboxoutstanding`. Ia
-- DIBACA, bukan dihitung — berbeda dari `inboxanalystdoctor` yang menghitung umur tugas
-- sendiri karena Report Definition-nya tidak menyediakan angkanya.
--
-- ============================================================================
-- CATATAN 1 — BATAS KEWENANGAN MEMAKAI DAFTAR NAMA, DAN ITU PUNYA HARGA
-- ============================================================================
--
-- Pega membandingkan dengan `IN {ASIS:TempOperator.CityID}` — sebuah DAFTAR, bukan satu
-- nilai. Sebabnya `MST_LOGIN_SURVEYOR.LOGINLEADER`: seorang leader melihat pekerjaan
-- anggotanya, bukan hanya miliknya.
--
-- Daftar berpanjang berubah tidak dapat dijadikan `IN (:1, :2, …)` yang jumlah bind-nya
-- tetap. Yang dipakai di sini:
--
--   INSTR(:1, '|' || UPPER(TRIM(s.SURVEYOR_NAME)) || '|') > 0
--
-- dengan `:1` berbentuk `|NAMA SATU|NAMA DUA|`. Nilainya TETAP lewat parameter binding —
-- tidak ada satu pun nama yang dirangkai ke teks SQL, sehingga larangan perangkaian
-- (`08-TECHNICAL-STRATEGY.md` §4.3) tetap utuh.
--
-- Harganya: `INSTR` tidak dapat memakai indeks pada `SURVEYOR_NAME`. Itu diterima karena
-- cakupan seorang leader berjumlah belasan, bukan ribuan, dan karena alternatifnya —
-- merangkai daftar `IN` dari nama — adalah persis celah `{ASIS:...}` yang sedang dihapus.
--
-- Pembatas `|` dipasang di KEDUA sisi tiap nama supaya "BUDI" tidak cocok dengan "BUDIONO".
-- Tanpa itu, seorang surveyor akan melihat pekerjaan surveyor lain yang namanya kebetulan
-- memuat namanya.
--
-- ============================================================================
-- CATATAN 2 — PENCOCOKAN MEMAKAI UPPER, DAN KENAPA
-- ============================================================================
--
-- `11-SECURITY.md` §3.1 mencatat kapitalisasi identitas di sistem lama TIDAK terjaga — nama
-- access group yang sama muncul dalam dua bentuk (`ViewClaimPNC`/`VIEWCLAIMPNC`). Nama
-- surveyor diketik manusia ke dua tabel berbeda (`MST_LOGIN_SURVEYOR.NAMA` dan
-- `T_SURVEYORLIST.SURVEYOR_NAME`), sehingga perbandingan persis akan membuat antrean tampak
-- KOSONG bagi sebagian pengguna — dan antrean kosong tidak pernah dilaporkan sebagai
-- kerusakan.
--
-- `TRIM` ikut dipakai karena kedua kolom diisi tanpa constraint apa pun.
--
-- ============================================================================
-- CATATAN 3 — URUTANNYA MENAIK, DAN ITU DISENGAJA
-- ============================================================================
--
-- `BrowseLossAdjuster` dan `BrowseInternalSurveyor` keduanya memakai
-- `ORDER BY a.pxCreateDateTime ASC` — yang TERTUA lebih dulu. Itu urutan antrean kerja:
-- pekerjaan yang paling lama menunggu berada di atas.
--
-- Ia BERBEDA dari inbox lain di aplikasi ini, yang menurun. Perbedaannya dibawa, bukan
-- diseragamkan (`P-5`).
--
-- Satu rule memang menyalahi: `BrowseOSLossAdjusterPIC` memakai `desc`. Rule itu melayani
-- layar PIC ASM, bukan layar ini.
--
-- Pemutus serinya `CASEID` lalu `INDEX_SURVEY` — tanpanya, dua janji survei yang diinput
-- pada detik yang sama berpindah-pindah urutan antar halaman.
--
-- ============================================================================
-- YANG BERUBAH DARI SISTEM LAMA
-- ============================================================================
--
-- 1. PAGINASI DIKERJAKAN BASIS DATA. `SetTempLostAdjuster` menyetel `.PageSize = 15` lalu
--    menomori halaman di klipboard. Di sini halamannya dipotong
--    `OFFSET … FETCH NEXT … ROWS ONLY` sebelum baris meninggalkan basis data.
--
-- 2. JUMLAH SELURUH BARIS DIHITUNG `COUNT(*) OVER ()`. Satu perjalanan, bukan dua.
--
-- 3. TAB CLOSE MEMAKAI `ADJUSTERSTATUS_1 = 'Close Case'`, bukan
--    `PYSTATUSWORK = 'Resolved-Completed'` milik objek SurveyClaim — kolom itu tidak ada di
--    tabel penggerak. Selisih terencana, dinyatakan ke pengguna (`D-54`).
--
-- 4. NILAI SELALU LEWAT PARAMETER BINDING, termasuk ketiga nilai `ADJUSTERSTATUS_1` dan
--    kedua nilai `KOMUNIKASISTATUS`. `D-15` melarang nilai bisnis tertanam di kode, dan
--    tertanam di dalam teks SQL adalah bentuk paling sulit ditemukannya.
--
-- CATATAN PENANDA BIND. Berkas ini memakai gaya Oracle `:n`, sama seperti seluruh modul lain
-- di aplikasi ini. Ia BELUM portabel ke PostgreSQL yang memakai `$n`; itu utang yang sudah
-- ada sebelum modul ini.

-- name: list_tasks
-- Satu halaman satu tab.
--
-- Bind:
--   :1  cakupan nama surveyor, berbentuk `|NAMA SATU|NAMA DUA|`  (lihat CATATAN 1)
--   :2  tab yang dibuka — nilai inboxsurvey.Tab
--   :3  login pemanggil, dipakai ketiga tab komunikasi
--   :4  ADJUSTERACCEPT_1 yang berarti sudah dikonfirmasi  -> "1"
--   :5  ADJUSTERSTATUS_1 tab Invoice                      -> "Invoice Fee"
--   :6  ADJUSTERSTATUS_1 tab Close                        -> "Close Case"
--   :7  KOMUNIKASISTATUS terbuka                          -> "0"
--   :8  KOMUNIKASISTATUS sudah dijawab                    -> "1"
--   :9  kata kunci pencarian, atau NULL bila kotak carinya kosong
--   :10 offset
--   :11 jumlah baris
SELECT s.CASEID                AS SURVEY_ID,
       s.PNCCASEID             AS CLAIM_ID,
       s.INDEX_SURVEY          AS SURVEY_INDEX,
       k.ADJUSTERPIC_1         AS APPOINTMENT_NUMBER,
       k.REFNO_1               AS REFERENCE_NUMBER,
       k.PYID                  AS CLAIM_NUMBER,
       k.POLICYNO              AS POLICY_NUMBER,
       k.QQNAME                AS INSURED_NAME,
       k.BUSINESSNAME          AS CLASS_OF_BUSINESS,
       s.LOSSTYPE              AS CAUSE_OF_LOSS,
       s.LOCATION_SURVEY       AS LOCATION,
       k.USERTEKNIS_1          AS TECHNICAL_PIC,
       s.SURVEYOR_NAME         AS ADJUSTER_PIC,
       k.DATEOFLOSS_1          AS DATE_OF_LOSS,
       k.AGING                 AS AGING_DAYS,
       k.ADJUSTERSTATUS_1      AS ASM_STATUS,
       s.SURVEYTYPE            AS SURVEYOR_TYPE,
       s.STS_SURVEY            AS SURVEY_STATUS,
       COUNT(*) OVER ()        AS TOTAL_ROWS
  FROM POOLDATA.T_SURVEYORLIST s
       INNER JOIN POOLDATA.T_CLAIMLIST_ADMIN k
               ON k.PZINSKEY = s.PNCCASEID
 WHERE k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND INSTR(:1, '|' || UPPER(TRIM(s.SURVEYOR_NAME)) || '|') > 0
   AND ((:2 = 'outstanding'
         AND k.ADJUSTERACCEPT_1 IS NULL)
     OR (:2 = 'all'
         AND k.ADJUSTERACCEPT_1 = :4)
     OR (:2 = 'invoice'
         AND k.ADJUSTERACCEPT_1 = :4
         AND k.ADJUSTERSTATUS_1 = :5)
     OR (:2 = 'close'
         AND k.ADJUSTERSTATUS_1 = :6)
     OR (:2 = 'belum-dijawab'
         AND EXISTS (SELECT 1
                       FROM POOLDATA.M_KOMUNIKASI_PNC kom
                      WHERE kom.CASEID = s.CASEID
                        AND kom.KOMUNIKASISTATUS = :7
                        AND UPPER(TRIM(kom.SENDER)) <> UPPER(TRIM(:3))))
     OR (:2 = 'belum-dibalas-asm'
         AND EXISTS (SELECT 1
                       FROM POOLDATA.M_KOMUNIKASI_PNC kom
                      WHERE kom.CASEID = s.CASEID
                        AND kom.KOMUNIKASISTATUS = :7
                        AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:3))))
     OR (:2 = 'sudah-dibalas-asm'
         AND EXISTS (SELECT 1
                       FROM POOLDATA.M_KOMUNIKASI_PNC kom
                      WHERE kom.CASEID = s.CASEID
                        AND kom.KOMUNIKASISTATUS = :8
                        AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:3)))))
   AND (:9 IS NULL
        OR UPPER(k.PYID) LIKE '%' || UPPER(:9) || '%'
        OR UPPER(k.REFNO_1) LIKE '%' || UPPER(:9) || '%')
 ORDER BY s.TGLINPUT, s.CASEID, s.INDEX_SURVEY
OFFSET :10 ROWS FETCH NEXT :11 ROWS ONLY

-- name: count_tabs
-- Jumlah baris KETUJUH tab sekaligus, dalam satu perjalanan.
--
-- Ia penerjemahan langsung `RDB List/CountOSLostAdjuster-SQL.xml`, yang menghitung seluruh
-- keranjang dengan tujuh `SUM(CASE WHEN …)` di atas satu pemindaian.
--
-- # Kenapa satu kueri, bukan tujuh
--
-- Karena bilah tab digambar SEKALIGUS. Tujuh perjalanan akan membaca gabungan yang sama
-- tujuh kali, dan gabungan itulah bagian yang mahal — bukan penjumlahannya.
--
-- Bind:
--   :1  cakupan nama surveyor
--   :2  login pemanggil
--   :3  ADJUSTERACCEPT_1 yang berarti sudah dikonfirmasi  -> "1"
--   :4  ADJUSTERSTATUS_1 tab Invoice                      -> "Invoice Fee"
--   :5  ADJUSTERSTATUS_1 tab Close                        -> "Close Case"
--   :6  KOMUNIKASISTATUS terbuka                          -> "0"
--   :7  KOMUNIKASISTATUS sudah dijawab                    -> "1"
SELECT SUM(CASE WHEN k.ADJUSTERACCEPT_1 IS NULL THEN 1 ELSE 0 END)
          AS COUNT_OUTSTANDING,
       SUM(CASE WHEN k.ADJUSTERACCEPT_1 = :3
                 AND k.ADJUSTERSTATUS_1 = :4 THEN 1 ELSE 0 END)
          AS COUNT_INVOICE,
       SUM(CASE WHEN k.ADJUSTERSTATUS_1 = :5 THEN 1 ELSE 0 END)
          AS COUNT_CLOSE,
       SUM(CASE WHEN k.ADJUSTERACCEPT_1 = :3 THEN 1 ELSE 0 END)
          AS COUNT_ALL,
       SUM(CASE WHEN EXISTS (SELECT 1
                               FROM POOLDATA.M_KOMUNIKASI_PNC kom
                              WHERE kom.CASEID = s.CASEID
                                AND kom.KOMUNIKASISTATUS = :6
                                AND UPPER(TRIM(kom.SENDER)) <> UPPER(TRIM(:2)))
                THEN 1 ELSE 0 END)
          AS COUNT_NOT_ANSWERED,
       SUM(CASE WHEN EXISTS (SELECT 1
                               FROM POOLDATA.M_KOMUNIKASI_PNC kom
                              WHERE kom.CASEID = s.CASEID
                                AND kom.KOMUNIKASISTATUS = :6
                                AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:2)))
                THEN 1 ELSE 0 END)
          AS COUNT_NOT_REPLIED,
       SUM(CASE WHEN EXISTS (SELECT 1
                               FROM POOLDATA.M_KOMUNIKASI_PNC kom
                              WHERE kom.CASEID = s.CASEID
                                AND kom.KOMUNIKASISTATUS = :7
                                AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:2)))
                THEN 1 ELSE 0 END)
          AS COUNT_REPLIED
  FROM POOLDATA.T_SURVEYORLIST s
       INNER JOIN POOLDATA.T_CLAIMLIST_ADMIN k
               ON k.PZINSKEY = s.PNCCASEID
 WHERE k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND INSTR(:1, '|' || UPPER(TRIM(s.SURVEYOR_NAME)) || '|') > 0

-- name: resolve_surveyor
-- Jembatan identitas: login pemanggil menjadi identitas surveyor.
--
-- Ia penerjemahan langsung `RDB List/GetLoginLeaderSurveyor-SQL.xml`:
--
--   select loginleader from pooldata.mst_login_surveyor
--    where login = {OperatorID.pyUserIdentifier}
--
-- ditambah `NAMA`, yang di Pega diambil `GetLoginMemberSurveyor` dari tabel yang sama.
--
-- # Lapisannya DUA, sesuai keputusan Work Owner 2026-09-28
--
--   M_LOGIN_PNC.LOGIN_ID        master pengguna aplikasi — siapa yang boleh masuk
--   MST_LOGIN_SURVEYOR.LOGIN    data surveyor            — siapa dia di data survei
--
-- Keduanya dicocokkan pada nilai login yang sama. Kueri ini membaca lapisan KEDUA saja;
-- lapisan pertama sudah dilewati saat pengguna berhasil masuk, dan mengulang pemeriksaannya
-- di sini hanya menambah satu gabungan tanpa menambah satu pun jaminan.
--
-- Bind:
--   :1  login pemanggil
SELECT m.LOGIN        AS SURVEYOR_LOGIN,
       m.NAMA         AS SURVEYOR_NAME,
       m.LOGINLEADER  AS LEADER_LOGIN,
       m.STSLOGIN     AS LOGIN_STATUS
  FROM POOLDATA.MST_LOGIN_SURVEYOR m
 WHERE UPPER(TRIM(m.LOGIN)) = UPPER(TRIM(:1))

-- name: resolve_members
-- Nama seluruh surveyor yang berada di bawah seorang leader.
--
-- Dipakai HANYA bila pemanggil ternyata seorang leader. Itulah yang membuat kueri antrean
-- membandingkan dengan DAFTAR, bukan satu nama — sama seperti `IN {ASIS:TempOperator.CityID}`
-- di sistem lama.
--
-- # Kenapa dicocokkan pada LOGINLEADER, bukan pada NAMA
--
-- Karena `LOGINLEADER` menyimpan LOGIN leadernya, bukan namanya — terbaca dari
-- `GetLoginLeaderSurveyor` yang membandingkannya dengan `OperatorID.pyUserIdentifier`.
--
-- Bind:
--   :1  login leader
SELECT m.NAMA AS SURVEYOR_NAME
  FROM POOLDATA.MST_LOGIN_SURVEYOR m
 WHERE UPPER(TRIM(m.LOGINLEADER)) = UPPER(TRIM(:1))

-- name: kpi_by_adjuster
-- Ringkasan KPI dikelompokkan per adjuster.
--
-- Penerjemahan `RDB List/GetSummaryKPIAdjuster-SQL.xml` dan
-- `GetSummaryKPIAdjusterALL-SQL.xml`, yang berbeda hanya pada penyaring `tipe`: yang pertama
-- menerimanya dari pemanggil, yang kedua mematok `'FINAL'`. Keduanya disatukan di sini
-- karena badan kuerinya identik — yang berbeda hanya nilai yang dikirim ke `:2`.
--
-- # `to_number` dibawa, dan itu memberi tahu sesuatu
--
-- Sistem lama menulis `round(avg(to_number(surveylap)),2)`. Adanya `to_number` menyiratkan
-- kolomnya bertipe TEKS di basis data. Ia dibawa apa adanya: menghilangkannya akan gagal
-- dengan ORA-01722 pada baris pertama yang tidak berisi angka, dan kegagalan itu justru
-- keterangan yang berguna.
--
-- Bind:
--   :1  cakupan nama surveyor
--   :2  nilai kolom `tipe`, atau NULL untuk seluruh kategori
--   :3  tahun `to_char(tanggal,'yyyy')`, atau NULL untuk seluruh tahun
SELECT d.ADJUSTER                                    AS GROUP_KEY,
       ROUND(AVG(TO_NUMBER(d.SURVEYLAP)), 2)         AS SURVEY_SCHEDULING,
       ROUND(AVG(TO_NUMBER(d.IMMEDIATEADVICE)), 2)   AS IMMEDIATE_ADVICE,
       ROUND(AVG(TO_NUMBER(d.PRELIMINARYADVICE)), 2) AS PRELIMINARY_ADVICE,
       ROUND(AVG(TO_NUMBER(d.INTERIM)), 2)           AS INTERIM_REPORT,
       ROUND(AVG(TO_NUMBER(d.PROGRESS)), 2)          AS PROGRESS_UPDATE,
       ROUND(AVG(TO_NUMBER(d.KOMUNIKASI)), 2)        AS COMMUNICATION_RESPONSE,
       ROUND(AVG(TO_NUMBER(d.PROPOSE)), 2)           AS PROPOSE_ADJUSTMENT,
       ROUND(AVG(TO_NUMBER(d.FINALREPORT)), 2)       AS FINAL_REPORT,
       ROUND(AVG(TO_NUMBER(d.NILAI)), 2)             AS VALUE_SCORE
  FROM POOLDATA.DETAIL_KPI_ADJUSTER d
 WHERE INSTR(:1, '|' || UPPER(TRIM(d.ADJUSTER)) || '|') > 0
   AND (:2 IS NULL OR UPPER(TRIM(d.TIPE)) = UPPER(TRIM(:2)))
   AND (:3 IS NULL OR TO_CHAR(d.TANGGAL, 'yyyy') = :3)
 GROUP BY d.ADJUSTER
 ORDER BY d.ADJUSTER

-- name: kpi_by_year
-- Ringkasan KPI dikelompokkan per TAHUN.
--
-- Penerjemahan `RDB List/GetSummaryKPIAdjusterKuartal-SQL.xml`, yang mengelompokkan
-- `group by to_char(tanggal,'yyyy')`.
--
-- # Namanya menyebut kuartal, pengelompokannya per tahun
--
-- Itu isi rule-nya apa adanya. Nama rule menyesatkan sejak di Pega; perilakunya yang dibawa,
-- bukan namanya (`P-5`). Bila yang dikehendaki memang per kuartal, itu perubahan perilaku
-- yang menempuh persetujuan — bukan perbaikan diam-diam di sini.
--
-- Bind:
--   :1  cakupan nama surveyor
--   :2  nilai kolom `tipe`, atau NULL untuk seluruh kategori
--   :3  tahun, atau NULL untuk seluruh tahun
SELECT TO_CHAR(d.TANGGAL, 'yyyy')                    AS GROUP_KEY,
       ROUND(AVG(TO_NUMBER(d.SURVEYLAP)), 2)         AS SURVEY_SCHEDULING,
       ROUND(AVG(TO_NUMBER(d.IMMEDIATEADVICE)), 2)   AS IMMEDIATE_ADVICE,
       ROUND(AVG(TO_NUMBER(d.PRELIMINARYADVICE)), 2) AS PRELIMINARY_ADVICE,
       ROUND(AVG(TO_NUMBER(d.INTERIM)), 2)           AS INTERIM_REPORT,
       ROUND(AVG(TO_NUMBER(d.PROGRESS)), 2)          AS PROGRESS_UPDATE,
       ROUND(AVG(TO_NUMBER(d.KOMUNIKASI)), 2)        AS COMMUNICATION_RESPONSE,
       ROUND(AVG(TO_NUMBER(d.PROPOSE)), 2)           AS PROPOSE_ADJUSTMENT,
       ROUND(AVG(TO_NUMBER(d.FINALREPORT)), 2)       AS FINAL_REPORT,
       ROUND(AVG(TO_NUMBER(d.NILAI)), 2)             AS VALUE_SCORE
  FROM POOLDATA.DETAIL_KPI_ADJUSTER d
 WHERE INSTR(:1, '|' || UPPER(TRIM(d.ADJUSTER)) || '|') > 0
   AND (:2 IS NULL OR UPPER(TRIM(d.TIPE)) = UPPER(TRIM(:2)))
   AND (:3 IS NULL OR TO_CHAR(d.TANGGAL, 'yyyy') = :3)
 GROUP BY TO_CHAR(d.TANGGAL, 'yyyy')
 ORDER BY TO_CHAR(d.TANGGAL, 'yyyy') DESC

-- name: check_tables
-- Dipakai perintah `-periksa`: memastikan KEEMPAT tabel terbaca dari koneksi yang dipakai.
--
-- Ia tidak menyentuh satu baris pun — yang diperiksa hak baca dan keberadaan tabelnya, bukan
-- isinya. Keempatnya diperiksa sekaligus karena kegagalan yang paling mungkin terjadi bukan
-- "tabel tidak ada" melainkan "hak baca hanya diberikan pada sebagian".
--
-- `POOLDATA.T_SURVEYORLIST` yang paling patut diperhatikan: ia tabel yang BELUM pernah
-- dibaca modul mana pun di aplikasi ini.
SELECT COUNT(*) AS PROBE
  FROM POOLDATA.T_SURVEYORLIST s
       INNER JOIN POOLDATA.T_CLAIMLIST_ADMIN k
               ON k.PZINSKEY = s.PNCCASEID
       INNER JOIN POOLDATA.MST_LOGIN_SURVEYOR m
               ON UPPER(TRIM(m.NAMA)) = UPPER(TRIM(s.SURVEYOR_NAME))
 WHERE 1 = 0

-- name: check_columns
-- Dipakai perintah `-periksa`: memastikan kolom yang BELUM terkonfirmasi memang ada.
--
-- Ia terpisah dari check_tables dengan sengaja. Keduanya gagal karena sebab yang sangat
-- berbeda — yang satu hak baca, yang satu nama kolom yang belum dipastikan DBA — dan galat
-- yang menyebut sebab yang salah akan mengirim orang yang memperbaikinya ke arah keliru.
--
-- `WHERE 1 = 0` membuat Oracle tetap MEM-PARSE seluruh kolom tanpa membaca satu baris pun.
-- Parsing itulah yang menghasilkan ORA-00904 bila namanya salah.
--
-- Yang diperiksa di sini BUKAN hanya kedua kolom yang diragukan. Kolom tab ikut masuk —
-- `ADJUSTERACCEPT_1` dan `ADJUSTERSTATUS_1` — karena keduanya menentukan ISI setiap tab, dan
-- tab yang kolomnya hilang akan gagal seluruhnya, bukan menampilkan satu sel kosong.
SELECT COUNT(k.ADJUSTERPIC_1)     AS PROBE_APPOINTMENT,
       COUNT(k.ADJUSTERACCEPT_1)  AS PROBE_ACCEPT,
       COUNT(k.ADJUSTERSTATUS_1)  AS PROBE_STATUS,
       COUNT(k.REFNO_1)           AS PROBE_REFERENCE,
       COUNT(k.AGING)             AS PROBE_AGING,
       COUNT(k.PXOBJCLASS)        AS PROBE_OBJECT_CLASS,
       COUNT(s.SURVEYTYPE)        AS PROBE_SURVEYOR_TYPE,
       COUNT(s.LOSSTYPE)          AS PROBE_CAUSE_OF_LOSS,
       COUNT(s.LOCATION_SURVEY)   AS PROBE_LOCATION,
       COUNT(s.STS_SURVEY)        AS PROBE_SURVEY_STATUS,
       COUNT(s.INDEX_SURVEY)      AS PROBE_SURVEY_INDEX
  FROM POOLDATA.T_SURVEYORLIST s
       INNER JOIN POOLDATA.T_CLAIMLIST_ADMIN k
               ON k.PZINSKEY = s.PNCCASEID
 WHERE 1 = 0

-- name: check_kpi
-- Dipakai perintah `-periksa`: memastikan tabel KPI beserta kesembilan kolom angkanya ada.
--
-- Terpisah dari kedua pemeriksaan di atas karena tab KPI dapat hidup atau mati SENDIRI:
-- `POOLDATA.DETAIL_KPI_ADJUSTER` diisi `Database/INSERT_KPIADJUSTER.prc`, dan ketiadaannya
-- tidak menghalangi tab INBOX sama sekali.
SELECT COUNT(d.ADJUSTER)          AS PROBE_ADJUSTER,
       COUNT(d.TIPE)              AS PROBE_TYPE,
       COUNT(d.TANGGAL)           AS PROBE_DATE,
       COUNT(d.SURVEYLAP)         AS PROBE_SCHEDULING,
       COUNT(d.IMMEDIATEADVICE)   AS PROBE_IMMEDIATE,
       COUNT(d.PRELIMINARYADVICE) AS PROBE_PRELIMINARY,
       COUNT(d.INTERIM)           AS PROBE_INTERIM,
       COUNT(d.PROGRESS)          AS PROBE_PROGRESS,
       COUNT(d.KOMUNIKASI)        AS PROBE_COMMUNICATION,
       COUNT(d.PROPOSE)           AS PROBE_PROPOSE,
       COUNT(d.FINALREPORT)       AS PROBE_FINAL,
       COUNT(d.NILAI)             AS PROBE_VALUE
  FROM POOLDATA.DETAIL_KPI_ADJUSTER d
 WHERE 1 = 0
