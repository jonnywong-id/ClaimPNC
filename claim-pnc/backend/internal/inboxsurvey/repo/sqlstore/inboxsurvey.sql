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
-- A. DUA TABEL, DAN PERAN MASING-MASING
-- ============================================================================
--
--   POOLDATA.T_SURVEYORLIST   menggerakkan baris   (JEJAK PERKEMBANGAN survei)
--   POOLDATA.T_CLAIM_PNC      header klaim         (satu baris per KLAIM)
--
-- disambung `c.CLAIMID = s.PNCCASEID` — persis seperti
-- `RDB List/BroswseKlaimByNoSurvey-SQL.xml` menyambungkannya.
--
-- Di Pega, SELURUH kueri layar ini membaca `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` dengan
-- `PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim'`, ditambah `DATAPEGA.PC_ASSIGN_WORKLIST`.
-- Kedua tabel DATAPEGA itu **dicabut dari pemakaian** (Work Owner 2026-09-28).
--
-- ## Dua tabel yang sempat dipakai dan sudah dilepas
--
-- `POOLDATA.T_CLAIMLIST_ADMIN` (dilepas 2026-09-29) — tabel datar itu hanya memuat klaim yang
-- tugasnya berada di antrean Admin, salah satu labelnya Choose Surveyor. Survei yang SEDANG
-- BERJALAN berarti klaimnya sudah MELEWATI tahap itu, sehingga `INNER JOIN` ke sana membuang
-- justru baris yang dicari layar ini. Cacat seperti itu tidak menghasilkan galat: layarnya
-- terisi sebagian, dan tampak wajar.
--
-- `POOLDATA.T_CLAIM_SURVEY_DATAPEGA` (dilepas 2026-09-30, keputusan Work Owner) — tabel cermin
-- objek kerja Pega. Ia sempat disambung `LEFT JOIN` untuk membawa `STATUSWORK`, tetapi tabelnya
-- **belum pernah terisi satu baris pun**: penjaganya di `Database/INSERT_SURVEYORLIST.prc:65-66`
-- membandingkan satu kolom `pzinskey` dengan DUA parameter berbeda (`= TCASEID` dan
-- `= TPNCCASEID`), sehingga tidak pernah terpenuhi.
--
-- ============================================================================
-- B. `T_SURVEYORLIST` ADALAH JEJAK PERKEMBANGAN, BUKAN DAFTAR PENUGASAN
-- ============================================================================
--
-- Ini pemahaman yang paling menentukan bentuk kueri di bawah, dan ia **salah dibaca dua kali**
-- sebelum diukur.
--
-- `INDEX_SURVEY` adalah nomor urut LANGKAH, dan nilainya bertambah tiap perubahan status:
--
--   Activity/SetSurveyorList-Act.xml
--     TempSurvey.IdxSurveyResults := @if(Param.IndexSurvey=="", local.index+1, Param.IndexSurvey)
--     childPageSurveyClaim.SurveyData.SurveyList(<LAST>).IdxSurveyResults := local.index+1
--
--   RDB List/GetDataProgressSurvey-SQL.xml     -- dibaca kembali sebagai RIWAYAT
--     order by to_number(index_survey) asc
--
-- Diukur di produksi 2026-09-29:
--
--   2.448 berkas survei  ->  17.641 baris     rata-rata 7,21 langkah per berkas
--   69,1% berkas punya lebih dari satu baris; terburuk SATU berkas = 176 baris
--
-- Layar Pega menampilkan **satu baris per berkas survei**. Tanpa penyaringan, layar ini akan
-- menampilkan tujuh baris untuk setiap satu yang benar, dengan Claim No berulang.
--
-- Karena itu kueri daftar mengambil **langkah TERAKHIR** tiap berkas — lihat CATATAN 3.
--
-- ## `STS_SURVEY` adalah `ADJUSTERSTATUS_1`
--
-- Terbukti dari sebaran nilainya di produksi: ketiga nilai yang dipakai Pega sebagai penyaring
-- ada di sana dengan jumlah yang nyata — `Final Report` 1.466, `Invoice Fee` 1.069,
-- `Close Case` 316 — berdampingan dengan seluruh tahapan hidup survei.
--
-- Pernyataan sebelumnya bahwa kolom ini hanya berisi `"On Progress"` **dicabut**: itu hanya
-- satu dari 22 nilai, dan penulis lainnya berada di luar export (`R-01`, `R-16`).
--
-- Kolom "Status ASM" karena itu diisi `s.STS_SURVEY` pada langkah terakhir.
--
-- ============================================================================
-- C. APA YANG BELUM TERBAWA
-- ============================================================================
--
-- Empat isian masih menunggu, seluruhnya milik objek kerja `Work-SurveyClaim`:
--
--   ADJUSTERACCEPT_1  -> tab Outstanding, ALL, dan Invoice
--   ADJUSTERPIC_1     -> kolom "Appointment No"
--   REFNO_1           -> kolom "Reference No" dan setengah kotak cari
--   PYSTATUSWORK      -> tab Close, dan penyaring "berkas survei masih terbuka"
--
-- Keempatnya diminta ditambahkan ke `POOLDATA.T_SURVEYORLIST` — lihat
-- `docs/permintaan-kolom-t-surveyorlist.md`. Kueri di bawah **belum membacanya**: menuliskan
-- kolom yang belum ada menghasilkan ORA-00904 yang menjatuhkan SELURUH layar, bukan sel kosong.
--
-- ## Kenapa `STS_SURVEY` tidak dapat menggantikan `PYSTATUSWORK`
--
-- Diuji langsung, dan gagal. Dari 1.070 berkas yang sudah `Resolved-*` di Pega, hanya 308
-- (28,8%) berakhir di `Close Case`/`Reject Case`:
--
--   Resolved-Completed -> Invoice Fee 497 · Close Case 301 · Final Report 95 · On Progress 45
--   Resolved-Rejected  -> On Progress 42 · (kosong) 22 · Reject Case **0 dari 70**
--
-- `Invoice Fee` adalah langkah terakhir pekerjaan adjuster — ia menagih, lalu berkasnya
-- ditutup petugas ASM. Adjuster tidak pernah mencatat "Close Case" sendiri.
--
-- Memakai `STS_SURVEY = 'Close Case'` sebagai pengganti tab Close akan menampilkan 307 dari
-- 1.000 berkas — kehilangan 69%. Itu bukan selisih terencana melainkan tab yang rusak, dan
-- penggantinya **dicabut**.
--
-- ============================================================================
-- PEMETAAN KOLOM — judul di layar -> kolom sebenarnya
-- ============================================================================
--
-- Judul dari `Section/InboxSurvey_section-Section.xml`; properti dari daftar Property-Set
-- pada `Activity/SetTempLostAdjuster-Act.xml`.
--
--   judul di layar      properti Pega            kolom sekarang            alias
--   ------------------- ------------------------ ------------------------- -------------------
--   Appointment No      .City                    — menunggu ADJUSTERPIC_1  —
--   Reference No        .AlasanDokterRejectRCL   — menunggu REFNO_1        —
--   Claim No            .UserName                c.CLAIMNO                 CLAIM_NUMBER
--   Policy No           .Country                 c.NOPOLIS                 POLICY_NUMBER
--   Insured Name        .AnalystDoctorRemaks     c.QQNAME                  INSURED_NAME
--   COB                 .KomiteStatus            c.BUSINESSNAME            CLASS_OF_BUSINESS
--   Cause Of Loss       .CauseOfLoss             s.LOSSTYPE  **?**         CAUSE_OF_LOSS
--   Location            .Location                s.LOCATION_SURVEY         LOCATION
--   PIC ASM             .UserTeknis              c.PICTEKNIK               TECHNICAL_PIC
--   PIC Loss Adjuster   .AnaylstRemarks          s.SURVEYOR_NAME  = *      ADJUSTER_PIC
--   Date of Loss        .DateOfLoss              c.DATEOFLOSS              DATE_OF_LOSS
--   Aging               .CPLValidDate            dihitung dari s.TGLINPUT  CREATED_AT
--   Status ASM          .UserAdmin               s.STS_SURVEY              ASM_STATUS
--
--   tidak digambar      —                        s.CASEID                  SURVEY_ID
--   tidak digambar      —                        s.PNCCASEID               CLAIM_ID
--   tidak digambar      —                        s.INDEX_SURVEY            SURVEY_INDEX
--   tidak digambar      —                        s.SURVEYTYPE              SURVEYOR_TYPE
--
-- CATATAN `= *` pada "PIC Loss Adjuster". `s.SURVEYOR_NAME` adalah padanan `SURVEYORNAME_1`
-- milik objek kerja — **kolom yang SAMA, orang yang sama**. Karena itu `SURVEYORNAME_1` TIDAK
-- perlu diminta.
--
-- Kolom itu dialiaskan TIGA nama berbeda di tiga rule, dan tidak satu pun mencerminkan isinya:
--
--   BrowseOSLossAdjusterPIC  SURVEYORNAME_1 AS "AnaylstRemarks"   <- yang dipakai section
--   BrowseLossAdjuster       SURVEYORNAME_1 AS "CountryID"
--   BrowseInternalSurveyor   SURVEYORNAME_1 AS "CountryID" DAN as "ComplianceRemark"
--
-- Yang mengikat kolom layar adalah `.AnaylstRemarks`, terbaca dari
-- `Section/InboxSurvey_section-Section.xml`. Jadi "PIC Loss Adjuster" = `SURVEYORNAME_1`.
--
-- ## Penyaringan langkah terakhir yang membuat keduanya BENAR-BENAR sama
--
-- `T_SURVEYORLIST` adalah jejak perkembangan, sehingga `SURVEYOR_NAME` dapat berbeda antar
-- langkah bila surveyornya diganti di tengah jalan. Objek kerja hanya punya SATU
-- `SURVEYORNAME_1` — yang berlaku sekarang.
--
-- Mengambil langkah TERAKHIR membuat keduanya sepadan. Dan karena penyaring cakupan dipasang
-- SESUDAH penyaringan itu (lihat CATATAN 3), surveyor yang sudah diganti tidak lagi melihat
-- berkas itu — persis perilaku Pega, yang menyaring atas keadaan objek kerja hari ini.
--
-- CATATAN "Aging". Ia DIHITUNG di Go dari `s.TGLINPUT`, bukan dibaca — `AGING` adalah kolom
-- tabel datar yang sudah tidak dipakai. Perhitungannya TIDAK dilakukan di SQL: "hari" yang
-- dimaksud pengguna adalah hari WIB sementara kolomnya UTC, dan menaruh konversi zona waktu di
-- dalam SQL adalah cara paling cepat menyebarkannya ke tempat yang lupa melakukannya — persis
-- cacat `Set7Hours` sistem lama.
--
-- CATATAN "Cause Of Loss". `s.LOSSTYPE` masih dugaan: kueri Pega yang mengisinya hilang dari
-- export (`R-16`), dan `LOSSTYPE` adalah satu-satunya kolom pada tabel penggerak yang
-- menyatakan jenis kerugian (`INSERT_SURVEYORLIST.prc`, parameter `TLOSSTYPE`).
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
-- tetap. Yang dipakai:
--
--   INSTR(:1, '|' || UPPER(TRIM(s.SURVEYOR_NAME)) || '|') > 0
--
-- dengan `:1` berbentuk `|NAMA SATU|NAMA DUA|`. Nilainya TETAP lewat parameter binding —
-- tidak ada satu pun nama yang dirangkai ke teks SQL.
--
-- Harganya: `INSTR` tidak dapat memakai indeks pada `SURVEYOR_NAME`. Diterima karena cakupan
-- seorang leader berjumlah belasan, dan karena alternatifnya adalah persis celah
-- `{ASIS:...}` yang sedang dihapus.
--
-- Pembatas `|` dipasang di KEDUA sisi tiap nama supaya "BUDI" tidak cocok dengan "BUDIONO".
--
-- ============================================================================
-- CATATAN 2 — PENCOCOKAN MEMAKAI UPPER, DAN KENAPA
-- ============================================================================
--
-- `11-SECURITY.md` §3.1 mencatat kapitalisasi identitas di sistem lama TIDAK terjaga. Nama
-- surveyor diketik manusia ke dua tabel berbeda (`MST_LOGIN_SURVEYOR.NAMA` dan
-- `T_SURVEYORLIST.SURVEYOR_NAME`), sehingga perbandingan persis akan membuat antrean tampak
-- KOSONG bagi sebagian pengguna — dan antrean kosong tidak pernah dilaporkan sebagai
-- kerusakan.
--
-- ============================================================================
-- CATATAN 3 — CARA MEMILIH LANGKAH TERAKHIR, DAN KENAPA BUKAN `TO_NUMBER`
-- ============================================================================
--
--   ROW_NUMBER() OVER (PARTITION BY s.CASEID
--                      ORDER BY LPAD(TRIM(s.INDEX_SURVEY), 10, '0') DESC NULLS LAST,
--                               s.TGLINPUT DESC NULLS LAST)
--
-- Tiga keputusan di dalam satu klausa itu:
--
--   `LPAD`, bukan `TO_NUMBER`.  `INDEX_SURVEY` bertipe TEKS, dan `to_number` khas Oracle —
--      PostgreSQL menuntut format mask, sehingga memakainya melanggar `D-20`. `LPAD` ada di
--      keduanya, dan mengurutkan `'9'` sebelum `'10'` dengan benar. Ia juga TIDAK dapat gagal
--      pada nilai yang bukan angka, sedangkan `TO_NUMBER` menjatuhkan seluruh layar dengan
--      ORA-01722 pada satu baris warisan yang cacat.
--
--   `NULLS LAST` disebut TEGAS.  Bawaan Oracle untuk `DESC` adalah `NULLS FIRST` — tanpa itu,
--      baris ber-`INDEX_SURVEY` kosong akan terpilih sebagai "langkah terakhir".
--
--   Penyaring cakupan TIDAK ditaruh di dalam partisi.  Kalau nama surveyor berganti di tengah
--      jalan, menyaring lebih dulu akan memilih langkah terakhir MILIK SURVEYOR ITU, bukan
--      langkah terakhir berkasnya. Tabelnya 17.641 baris, sehingga memindai seluruhnya murah.
--
-- ============================================================================
-- CATATAN 4 — URUTANNYA MENAIK, DAN ITU DISENGAJA
-- ============================================================================
--
-- `BrowseLossAdjuster` dan `BrowseInternalSurveyor` keduanya `ORDER BY … ASC` — yang TERTUA
-- lebih dulu. Itu urutan antrean kerja. Ia BERBEDA dari inbox lain di aplikasi ini yang
-- menurun, dan perbedaannya dibawa (`P-5`).
--
-- Pemutus serinya `CASEID` — sesudah penyaringan langkah terakhir, tepat satu baris tersisa
-- per berkas survei, sehingga `INDEX_SURVEY` tidak lagi diperlukan sebagai pemutus.
--
-- CATATAN PENANDA BIND. Berkas ini memakai gaya Oracle `:n`, sama seperti seluruh modul lain.
-- Ia BELUM portabel ke PostgreSQL yang memakai `$n`.

-- name: list_tasks
-- Satu halaman satu tab, satu baris per BERKAS SURVEI.
--
-- HANYA tab yang dapat dihitung yang sampai ke sini. Keempat tab yang membutuhkan
-- `ADJUSTERACCEPT_1` atau `PYSTATUSWORK` dicegat lebih dulu di Go (`Tab.Available`).
--
-- Bind:
--   :1  cakupan nama surveyor, berbentuk `|NAMA SATU|NAMA DUA|`  (lihat CATATAN 1)
--   :2  tab yang dibuka — nilai inboxsurvey.Tab
--   :3  login pemanggil, dipakai ketiga tab komunikasi
--   :4  KOMUNIKASISTATUS terbuka        -> "0"
--   :5  KOMUNIKASISTATUS sudah dijawab  -> "1"
--   :6  kata kunci pencarian, atau NULL bila kotak carinya kosong
--   :7  offset
--   :8  jumlah baris
SELECT s.CASEID            AS SURVEY_ID,
       s.PNCCASEID         AS CLAIM_ID,
       s.INDEX_SURVEY      AS SURVEY_INDEX,
       c.CLAIMNO           AS CLAIM_NUMBER,
       c.NOPOLIS           AS POLICY_NUMBER,
       c.QQNAME            AS INSURED_NAME,
       c.BUSINESSNAME      AS CLASS_OF_BUSINESS,
       s.LOSSTYPE          AS CAUSE_OF_LOSS,
       s.LOCATION_SURVEY   AS LOCATION,
       c.PICTEKNIK         AS TECHNICAL_PIC,
       s.SURVEYOR_NAME     AS ADJUSTER_PIC,
       c.DATEOFLOSS        AS DATE_OF_LOSS,
       s.TGLINPUT          AS CREATED_AT,
       s.STS_SURVEY        AS ASM_STATUS,
       s.SURVEYTYPE        AS SURVEYOR_TYPE,
       COUNT(*) OVER ()    AS TOTAL_ROWS
  FROM (SELECT t.*,
               ROW_NUMBER() OVER (PARTITION BY t.CASEID
                                  ORDER BY LPAD(TRIM(t.INDEX_SURVEY), 10, '0') DESC NULLS LAST,
                                           t.TGLINPUT DESC NULLS LAST) AS STEP_RANK
          FROM POOLDATA.T_SURVEYORLIST t) s
       INNER JOIN POOLDATA.T_CLAIM_PNC c
               ON c.CLAIMID = s.PNCCASEID
 WHERE s.STEP_RANK = 1
   AND INSTR(:1, '|' || UPPER(TRIM(s.SURVEYOR_NAME)) || '|') > 0
   AND ((:2 = 'belum-dijawab'
         AND EXISTS (SELECT 1
                       FROM POOLDATA.M_KOMUNIKASI_PNC kom
                      WHERE kom.CASEID = s.CASEID
                        AND kom.KOMUNIKASISTATUS = :4
                        AND UPPER(TRIM(kom.SENDER)) <> UPPER(TRIM(:3))))
     OR (:2 = 'belum-dibalas-asm'
         AND EXISTS (SELECT 1
                       FROM POOLDATA.M_KOMUNIKASI_PNC kom
                      WHERE kom.CASEID = s.CASEID
                        AND kom.KOMUNIKASISTATUS = :4
                        AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:3))))
     OR (:2 = 'sudah-dibalas-asm'
         AND EXISTS (SELECT 1
                       FROM POOLDATA.M_KOMUNIKASI_PNC kom
                      WHERE kom.CASEID = s.CASEID
                        AND kom.KOMUNIKASISTATUS = :5
                        AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:3)))))
   AND (:6 IS NULL
        OR UPPER(c.CLAIMNO) LIKE '%' || UPPER(:6) || '%')
 ORDER BY s.TGLINPUT, s.CASEID
OFFSET :7 ROWS FETCH NEXT :8 ROWS ONLY

-- name: count_tabs
-- Jumlah berkas survei pada ketiga tab komunikasi, dalam satu perjalanan.
--
-- Keempat tab lain TIDAK dihitung di sini — kolom penggeraknya belum ada. Mengembalikan nol
-- untuk keempatnya akan menyatakan "tab ini kosong", padahal yang benar adalah "tab ini belum
-- dapat dihitung". Perbedaannya disampaikan Go, bukan disamarkan menjadi angka nol.
--
-- Penyaring langkah terakhir SAMA PERSIS dengan list_tasks. Kalau berbeda, bilah tab akan
-- menyebut angka yang tidak sesuai isi tabnya — dan itu meruntuhkan kepercayaan pada seluruh
-- layar.
--
-- Bind:
--   :1  cakupan nama surveyor
--   :2  login pemanggil
--   :3  KOMUNIKASISTATUS terbuka        -> "0"
--   :4  KOMUNIKASISTATUS sudah dijawab  -> "1"
SELECT SUM(CASE WHEN EXISTS (SELECT 1
                               FROM POOLDATA.M_KOMUNIKASI_PNC kom
                              WHERE kom.CASEID = s.CASEID
                                AND kom.KOMUNIKASISTATUS = :3
                                AND UPPER(TRIM(kom.SENDER)) <> UPPER(TRIM(:2)))
                THEN 1 ELSE 0 END)
          AS COUNT_NOT_ANSWERED,
       SUM(CASE WHEN EXISTS (SELECT 1
                               FROM POOLDATA.M_KOMUNIKASI_PNC kom
                              WHERE kom.CASEID = s.CASEID
                                AND kom.KOMUNIKASISTATUS = :3
                                AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:2)))
                THEN 1 ELSE 0 END)
          AS COUNT_NOT_REPLIED,
       SUM(CASE WHEN EXISTS (SELECT 1
                               FROM POOLDATA.M_KOMUNIKASI_PNC kom
                              WHERE kom.CASEID = s.CASEID
                                AND kom.KOMUNIKASISTATUS = :4
                                AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:2)))
                THEN 1 ELSE 0 END)
          AS COUNT_REPLIED
  FROM (SELECT t.*,
               ROW_NUMBER() OVER (PARTITION BY t.CASEID
                                  ORDER BY LPAD(TRIM(t.INDEX_SURVEY), 10, '0') DESC NULLS LAST,
                                           t.TGLINPUT DESC NULLS LAST) AS STEP_RANK
          FROM POOLDATA.T_SURVEYORLIST t) s
       INNER JOIN POOLDATA.T_CLAIM_PNC c
               ON c.CLAIMID = s.PNCCASEID
 WHERE s.STEP_RANK = 1
   AND INSTR(:1, '|' || UPPER(TRIM(s.SURVEYOR_NAME)) || '|') > 0

-- name: resolve_surveyor
-- Jembatan identitas: login pemanggil menjadi identitas surveyor.
--
-- Penerjemahan langsung `RDB List/GetLoginLeaderSurveyor-SQL.xml`:
--
--   select loginleader from pooldata.mst_login_surveyor
--    where login = {OperatorID.pyUserIdentifier}
--
-- ditambah `NAMA`, yang di Pega diambil `GetLoginMemberSurveyor` dari tabel yang sama.
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
-- Dicocokkan pada `LOGINLEADER`, bukan pada `NAMA`: kolom itu menyimpan LOGIN atasan — terbaca
-- dari `GetLoginLeaderSurveyor` yang membandingkannya dengan `OperatorID.pyUserIdentifier`.
--
-- Bind:
--   :1  login leader
SELECT m.NAMA AS SURVEYOR_NAME
  FROM POOLDATA.MST_LOGIN_SURVEYOR m
 WHERE UPPER(TRIM(m.LOGINLEADER)) = UPPER(TRIM(:1))

-- name: kpi_by_adjuster
-- Ringkasan KPI dikelompokkan per adjuster.
--
-- Penerjemahan `GetSummaryKPIAdjuster-SQL.xml` dan `GetSummaryKPIAdjusterALL-SQL.xml`, yang
-- berbeda hanya pada penyaring `tipe`.
--
-- `to_number` dibawa apa adanya: adanya fungsi itu di sistem lama menyiratkan kolomnya
-- bertipe TEKS. Menghilangkannya akan gagal ORA-01722 pada baris pertama yang bukan angka —
-- dan kegagalan itu justru keterangan yang berguna.
--
-- Bind:
--   :1  cakupan nama surveyor
--   :2  nilai kolom `tipe`, atau NULL untuk seluruh kategori
--   :3  tahun, atau NULL untuk seluruh tahun
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
-- Penerjemahan `GetSummaryKPIAdjusterKuartal-SQL.xml`. Namanya menyebut kuartal,
-- pengelompokannya `to_char(tanggal,'yyyy')` — per TAHUN. Perilakunya yang dibawa, bukan
-- namanya (`P-5`).
--
-- Bind: sama dengan kpi_by_adjuster.
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
-- Dipakai perintah `-periksa`: memastikan ketiga tabel terbaca dari koneksi yang dipakai.
--
-- `POOLDATA.T_SURVEYORLIST` yang paling patut diperhatikan: ia tabel yang BELUM pernah dibaca
-- modul mana pun di aplikasi ini, sehingga hak bacanya belum pernah terbukti.
SELECT COUNT(*) AS PROBE
  FROM POOLDATA.T_SURVEYORLIST s
       INNER JOIN POOLDATA.T_CLAIM_PNC c
               ON c.CLAIMID = s.PNCCASEID
       INNER JOIN POOLDATA.MST_LOGIN_SURVEYOR m
               ON UPPER(TRIM(m.NAMA)) = UPPER(TRIM(s.SURVEYOR_NAME))
 WHERE 1 = 0

-- name: check_columns
-- Dipakai perintah `-periksa`: memastikan setiap kolom yang dibaca kueri daftar memang ada.
--
-- Terpisah dari check_tables dengan sengaja — keduanya gagal karena sebab yang berbeda, dan
-- galat yang menyebut sebab yang salah mengirim orang yang memperbaikinya ke arah keliru.
--
-- `WHERE 1 = 0` membuat Oracle tetap MEM-PARSE seluruh kolom tanpa membaca satu baris pun.
-- Parsing itulah yang menghasilkan ORA-00904 bila namanya salah.
--
-- Kolom `T_CLAIM_PNC` ikut diperiksa meski tabelnya sudah dipakai modul lain: penamaannya
-- BERBEDA JAUH dari tabel datar (`CLAIMNO` bukan `PYID`, `NOPOLIS` bukan `POLICYNO`,
-- `PICTEKNIK` bukan `USERTEKNIS_1`), dan salah satu saja menjatuhkan seluruh layar.
SELECT COUNT(c.CLAIMID)           AS PROBE_CLAIM_ID,
       COUNT(c.CLAIMNO)           AS PROBE_CLAIM_NO,
       COUNT(c.NOPOLIS)           AS PROBE_POLICY_NO,
       COUNT(c.QQNAME)            AS PROBE_INSURED,
       COUNT(c.BUSINESSNAME)      AS PROBE_COB,
       COUNT(c.PICTEKNIK)         AS PROBE_TECHNICAL_PIC,
       COUNT(c.DATEOFLOSS)        AS PROBE_DATE_OF_LOSS,
       COUNT(s.LOSSTYPE)          AS PROBE_CAUSE_OF_LOSS,
       COUNT(s.LOCATION_SURVEY)   AS PROBE_LOCATION,
       COUNT(s.SURVEYTYPE)        AS PROBE_SURVEYOR_TYPE,
       COUNT(s.STS_SURVEY)        AS PROBE_ASM_STATUS,
       COUNT(s.INDEX_SURVEY)      AS PROBE_SURVEY_INDEX,
       COUNT(s.TGLINPUT)          AS PROBE_CREATED_AT
  FROM POOLDATA.T_SURVEYORLIST s
       INNER JOIN POOLDATA.T_CLAIM_PNC c
               ON c.CLAIMID = s.PNCCASEID
 WHERE 1 = 0

-- name: check_new_columns
-- Dipakai perintah `-periksa`: melaporkan kolom mana dari keempatnya yang SUDAH ada.
--
-- # Kenapa lewat katalog, bukan `SELECT COUNT(kolom) … WHERE 1=0`
--
-- Probe berbasis parsing bersifat SEMUA-ATAU-TIDAK: satu kolom yang belum ada menghasilkan
-- ORA-00904, dan ketiga kolom lain yang sudah ada ikut terbaca sebagai belum ada. Itu persis
-- keadaan 2026-09-30 — dua dari empat kolom tiba lebih dulu, dan probe lama tidak dapat
-- menunjukkannya.
--
-- Katalog melaporkan per kolom, dan tidak dapat gagal karena kolomnya tidak ada.
--
-- CATATAN NAMA. Kolom yang ditambahkan 2026-09-30 TANPA akhiran `_1` — `ADJUSTERACCEPT`, bukan
-- `ADJUSTERACCEPT_1`. Akhiran itu artefak perataan Pega, dan menghilangkannya memang lebih
-- bersih; yang penting nama di sini mengikuti nama SEBENARNYA di basis data.
SELECT COUNT(CASE WHEN COLUMN_NAME = 'ADJUSTERACCEPT' THEN 1 END) AS HAS_ACCEPT,
       COUNT(CASE WHEN COLUMN_NAME = 'ADJUSTERPIC'    THEN 1 END) AS HAS_APPOINTMENT,
       COUNT(CASE WHEN COLUMN_NAME = 'REFNO'          THEN 1 END) AS HAS_REFERENCE,
       COUNT(CASE WHEN COLUMN_NAME = 'PYSTATUSWORK'   THEN 1 END) AS HAS_WORK_STATUS
  FROM ALL_TAB_COLUMNS
 WHERE OWNER = 'POOLDATA'
   AND TABLE_NAME = 'T_SURVEYORLIST'

-- name: check_filled_columns
-- Dipakai perintah `-periksa`: menghitung berapa baris kolom yang SUDAH ADA benar-benar terisi.
--
-- # Kenapa TERPISAH dari check_new_columns
--
-- Karena keduanya diperbaiki langkah yang berbeda: kolom ditambahkan DBA lewat `ALTER`,
-- sedangkan isinya ditulis procedure. Menyatukannya membuat "kolomnya sudah ada tetapi masih
-- kosong" terbaca sebagai siap — dan menghidupkan tab atas dasar itu menghasilkan tab kosong
-- yang terbaca sebagai "tidak ada pekerjaan".
--
-- Yang paling menentukan `FILLED_ACCEPT`: tab Outstanding menyaring `ADJUSTERACCEPT IS NULL`,
-- sehingga kolom yang ADA tetapi SELURUHNYA kosong akan menampilkan **seluruh antrean** sebagai
-- "belum dikonfirmasi adjuster". Itu cacat diam — layarnya terisi wajar dan isinya salah.
--
-- Ia HANYA memuat kolom yang sudah ada per 2026-09-30. `ADJUSTERPIC` dan `PYSTATUSWORK` belum
-- ditambahkan, dan menuliskannya di sini akan membuat kueri ini gagal seluruhnya — termasuk
-- untuk dua kolom yang justru ingin diukur.
SELECT COUNT(*)                 AS TOTAL_ROWS,
       COUNT(s.ADJUSTERACCEPT)  AS FILLED_ACCEPT,
       COUNT(s.REFNO)           AS FILLED_REFERENCE
  FROM POOLDATA.T_SURVEYORLIST s

-- name: check_kpi
-- Dipakai perintah `-periksa`: memastikan tabel KPI beserta kolom angkanya ada.
--
-- Terpisah karena tab KPI dapat hidup atau mati SENDIRI: `POOLDATA.DETAIL_KPI_ADJUSTER` diisi
-- `Database/INSERT_KPIADJUSTER.prc`, dan ketiadaannya tidak menghalangi tab INBOX.
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
