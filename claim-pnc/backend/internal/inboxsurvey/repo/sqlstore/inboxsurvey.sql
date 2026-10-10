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
-- `STS_SURVEY` karena itu **dibaca sebagai penyaring tab Invoice**, bukan sebagai isi kolom.
-- Kolom layar "Status ASM" ternyata bukan ini sama sekali — ia `ASMSTATUS_1` yang berisi peran
-- koasuransi, dan padanannya `T_CLAIM_PNC.LEADER_MEMBER`. Lihat PEMETAAN KOLOM di bawah.
--
-- ============================================================================
-- C. APA YANG BELUM TERBAWA
-- ============================================================================
--
-- LIMA isian masih menunggu, seluruhnya milik objek kerja `Work-SurveyClaim`. Namanya di
-- `T_SURVEYORLIST` memakai garis bawah dan TANPA akhiran `_1` — akhiran itu artefak perataan
-- Pega, dan garis bawahnya mengikuti gaya tabel ini (`SURVEYOR_NAME`, `LOCATION_SURVEY`):
--
--   kolom                menghidupkan                                  keadaan 2026-10-05
--   -------------------- --------------------------------------------- ------------------
--   ADJUSTERACCEPT       tab Outstanding, ALL, dan Invoice              ADA, masih KOSONG
--   PYSTATUSWORK         tab Close, penyaring "berkas masih terbuka"    ADA, masih KOSONG
--   REFNO                kolom Reference No, setengah kotak cari        ADA, masih KOSONG
--   ADJUSTER_PIC         kolom "PIC Loss Adjuster"                      ADA, masih KOSONG
--   RESCHEDULE_LOCATION  kolom "Location"                               ADA, masih KOSONG
--
-- Kueri di bawah **belum membaca satu pun**. Seluruhnya sudah ada, dan justru itu yang
-- berbahaya: kolom yang ADA tetapi KOSONG dibaca tanpa galat apa pun, lalu menjawab salah.
-- `ADJUSTERACCEPT IS NULL` bernilai benar untuk seluruh 17.641 baris, sehingga tab Outstanding
-- akan menampilkan seluruh antrean sebagai belum dikonfirmasi adjuster — terisi wajar, dan
-- salah. Ditahan sampai `claimpnc -periksa` melaporkan keterisiannya.
-- Lihat `docs/permintaan-kolom-t-surveyorlist.md`.
--
-- HATI-HATI: `ADJUSTER_PIC` adalah nama kolom DAN nama alias. Kueri daftar memakai
-- `s.SURVEYOR_NAME AS ADJUSTER_PIC` — alias menamai kolom LAYAR, `s.` menamai kolom BASIS
-- DATA. Keduanya akan menyatu (`s.ADJUSTER_PIC AS ADJUSTER_PIC`) begitu kolomnya terisi.
--
-- DUA PERMINTAAN YANG DICORET (2026-10-03), dan cara gugurnya layak diingat:
--
--   ADJUSTER_PIC untuk "Appointment No"
--                keliru — nomor itu `SRV-xxxxx`, dan sudah ada sebagai `s.CASEID`.
--                Kolomnya tetap diminta, tetapi untuk "PIC Loss Adjuster".
--   ASMSTATUS    tidak perlu — `T_CLAIM_PNC.LEADER_MEMBER` sudah membawanya, diukur dengan
--                nol pertentangan pada 17.633 baris. Usulan Work Owner.
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
-- Judul dari `Section/InboxSurvey_section-Section.xml`, urut dokumen. Properti dari sel data
-- grid pada berkas yang sama, juga urut dokumen, setelah membuang DUA sel `Embed-NameValuePair`
-- (parameter tautan, bukan kolom).
--
-- CARA PEMETAAN INI DIPASTIKAN, dan kenapa ia perlu dipastikan. Versi sebelumnya **bergeser
-- satu kolom** karena kedua parameter tautan ikut terhitung sebagai kolom. Yang menegakkannya
-- bukan hitungan ulang melainkan TIGA JANGKAR yang tidak bergantung urutan — judul yang
-- namanya persis sama dengan propertinya:
--
--   "Cause Of Loss" <-> .CauseOfLoss     "Location" <-> .Location     "Date of Loss" <-> .DateOfLoss
--
-- Ketiganya jatuh tepat pada posisi 7, 8, dan 11. Pergeseran satu kolom akan memindahkan
-- ketiganya sekaligus, jadi kecocokan ini tidak mungkin kebetulan.
--
-- SUMBERNYA SEKARANG KEEMPAT KUERI TAB YANG SEBENARNYA, bukan kueri rujukan. Kelima rule yang
-- hilang diterima 2026-10-03, dan **keempat browse memakai daftar alias yang IDENTIK** —
-- sehingga pemetaan di bawah berlaku untuk seluruh tab, bukan satu tab saja. Itu menutup
-- dugaan lama bahwa tiap tab mungkin punya daftar kolomnya sendiri.
--
--   judul di layar      kolom objek kerja Pega   kolom di sini             alias        status
--   ------------------- ------------------------ ------------------------- ------------ ------
--   Appointment No      a.pyid                   s.CASEID tanpa prefix     (SURVEY_ID)  setara
--   Reference No        a.REFNO_1                s.REFNO                   —            ADA, KOSONG
--   Claim No            a.CASEID_1               c.CLAIMNO                 CLAIM_NUMBER  ?
--   Policy No           a.POLICYNO               c.NOPOLIS                 POLICY_NUMBER ?
--   Insured Name        a.QQNAME                 c.QQNAME                  INSURED_NAME  ?
--   COB                 subquery BUSINESSNAME    c.BUSINESSNAME            CLASS_OF_BUSINESS ?
--   Cause Of Loss       subquery CAUSEOFLOSS     subquery yang SAMA        CAUSE_OF_LOSS setara
--   Location            a.RescheduleLocation_1   s.LOCATION_SURVEY         LOCATION      ?
--   PIC ASM             a.USERTEKNIS_1           c.PICTEKNIK               TECHNICAL_PIC ?
--   PIC Loss Adjuster   a.ADJUSTERPIC_1          s.SURVEYOR_NAME           ADJUSTER_PIC  ?
--   Date of Loss        a.DATEOFLOSS_1           c.DATEOFLOSS              DATE_OF_LOSS  ?
--   Aging               a.pxcreatedatetime       dihitung dari s.TGLINPUT  CREATED_AT    ?
--   Status ASM          a.ASMSTATUS_1            c.LEADER_MEMBER           ASM_STATUS   setara*
--
--   (asal Appointment No) —                      s.CASEID                  SURVEY_ID
--   tidak digambar      —                        s.PNCCASEID               CLAIM_ID
--   tidak digambar      —                        s.INDEX_SURVEY            SURVEY_INDEX
--   tidak digambar      —                        s.SURVEYTYPE              SURVEYOR_TYPE
--
-- ARTI KOLOM "status" DI ATAS:
--
--   setara      dibuktikan sepadan dengan kueri Pega
--   ?           padanannya MASUK AKAL tetapi BELUM diuji ke basis data — tebakan terbuka
--   BEDA        terbukti kolom yang BERLAINAN; selisihnya nyata dan belum diputuskan
--
-- CATATAN `setara*` pada "Status ASM". Kolom itu BUKAN status melainkan **peran koasuransi**:
-- `ASMSTATUS_1` hanya bernilai `LEADER`, `MEMBER`, atau kosong, dan `SetTempLostAdjuster`
-- menggambarnya lewat `@If(.UserAdmin=="", "LEADER", .UserAdmin)` — kosong tampil `LEADER`.
--
-- `T_SURVEYORLIST` tidak punya padanannya, tetapi `T_CLAIM_PNC.LEADER_MEMBER` membawanya.
-- Diukur di produksi 2026-10-03, **nol pertentangan** pada 17.633 baris:
--
--   LEADER_MEMBER   ASMSTATUS_1   baris
--   --------------- ------------- -------
--   LEADER          LEADER         15.125
--   LEADER          (kosong)        1.444
--   MEMBER          MEMBER          1.054
--   MEMBER          (kosong)           10
--
-- Tanda bintangnya untuk sepuluh baris terakhir: `LEADER_MEMBER` menyebutnya MEMBER sedangkan
-- Pega menggambarnya LEADER, karena nilai kosong jatuh ke bawaan. **0,06% dari seluruh baris**,
-- dan di sana modul ini justru lebih tepat daripada layar lama.
--
-- Usulan memakai `LEADER_MEMBER` datang dari Work Owner, dan ia menghapus satu permintaan
-- kolom yang sudah sempat diajukan.
--
-- Akibat yang harus disadari: **status perkembangan adjuster (`STS_SURVEY`) tidak lagi
-- digambar**. Itu memang perilaku Pega — `AdjusterStatus_1` dialiaskan `"CauseOfLossID"` dan
-- tidak termasuk 13 sel data grid. Kolomnya tetap dibaca saat tab Invoice dihidupkan, tetapi
-- sebagai penyaring, bukan sebagai isi kolom.
--
-- CATATAN "PIC Loss Adjuster" — dugaan yang DICABUT. Berkas ini sempat menyatakan kolom itu
-- berasal dari `SURVEYORNAME_1`, disimpulkan dari `BrowseOSLossAdjusterPIC`. Keempat kueri tab
-- membuktikan sebaliknya: ia `a.ADJUSTERPIC_1`. Apakah `s.SURVEYOR_NAME` membawa isi yang sama
-- **belum diuji**.
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
-- CATATAN 4 — URUTANNYA MENURUN, MENGIKUTI KEEMPAT KUERI TAB
-- ============================================================================
--
-- Keempat kueri tab — `BrowseOSLostAdjuster`, `BrowseConfirmLostAdjuster`,
-- `BrowseCloseLostAdjuster`, `BrowseCommunicationLostAdjuster` — seluruhnya memakai
--
--   ROW_NUMBER() OVER (ORDER BY a.pxcreatedatetime DESC)
--
-- yang TERBARU lebih dulu.
--
-- KOREKSI 2026-10-03. Sampai hari itu berkas ini menyatakan urutannya MENAIK, dengan alasan
-- "`BrowseLossAdjuster` dan `BrowseInternalSurveyor` keduanya ASC". Alasan itu runtuh begitu
-- keempat kueri tab tiba: **kedua kueri itu bukan penggerak grid ini**, dan yang sebenarnya
-- dipakai justru DESC. Layar karena itu sempat membalik urutan antrean.
--
-- Pemutus serinya `CASEID` — sesudah penyaringan langkah terakhir, tepat satu baris tersisa
-- per berkas survei, sehingga `INDEX_SURVEY` tidak lagi diperlukan sebagai pemutus.
--
-- CATATAN PENANDA BIND. Berkas ini memakai gaya Oracle `:n`, sama seperti seluruh modul lain.
-- Ia BELUM portabel ke PostgreSQL yang memakai `$n`.
--
--
-- ============================================================================
-- ANGKA BIND WAJIB MENGIKUTI URUTAN KEMUNCULANNYA — INI BUKAN KERAPIAN
-- ============================================================================
--
-- Driver yang dipakai aplikasi ini adalah `github.com/sijms/go-ora/v2`. Untuk argumen yang
-- datang dari `database/sql` tanpa nama, ia menaruh setiap argumen pada POSISI ke-x
-- (`command.go:1992-2002`, `stmt.setParam(x, *par)`) dan mengirimnya sebagai bind POSISIONAL.
-- Oracle memetakan posisi itu ke placeholder menurut URUTAN KEMUNCULANNYA di teks kueri —
-- BUKAN menurut angka yang tertulis.
--
-- Artinya `:9` tidak berarti "argumen kesembilan". Ia berarti "placeholder yang ke sekian
-- muncul". Bila keduanya berbeda, argumen masuk ke tempat yang salah.
--
-- Akibatnya dua macam, dan yang kedua jauh lebih berbahaya:
--
--   1. Tipe tidak cocok  -> galat seketika. `OFFSET :7 ROWS` yang kebagian "Resolved-Completed"
--      menghasilkan ORA-01722, dan layar menampilkan "tidak dapat dimuat".
--   2. Tipe cocok        -> TANPA GALAT, hasilnya salah. Itu yang terjadi pada `list_tasks`
--      sampai 2026-10-07: login tertukar dengan status pesan, sehingga ketiga tab komunikasi
--      SELALU kosong dan tidak seorang pun melaporkannya sebagai kerusakan.
--
-- Karena itu setiap kueri di berkas ini dinomori menurut urutan kemunculan, dan
-- `TestAngkaBindMengikutiUrutanKemunculan` menegakkannya secara mekanis. Bila sebuah kondisi
-- dipindahkan, angkanya ikut berubah — ujinya yang memberi tahu, bukan pengguna.
--
-- Modul Master Login sudah mencatat bahaya yang sama lebih dulu: "tidak semua driver memetakan
-- parameter bernomor ke posisi argumen dengan cara yang sama". Modul ini tidak mengikutinya,
-- dan membayarnya dengan tiga tab yang diam-diam kosong.

-- name: list_tasks
-- Satu halaman satu tab, satu baris per BERKAS SURVEI.
--
-- HANYA tab yang dapat dihitung yang sampai ke sini. Keempat tab yang membutuhkan
-- `ADJUSTERACCEPT` atau `PYSTATUSWORK` dicegat lebih dulu di Go (`Tab.Available`).
--
-- Bind:
--   :scope  cakupan nama surveyor, berbentuk `|NAMA SATU|NAMA DUA|`  (lihat CATATAN 1)
--   :tab  tab yang dibuka — nilai inboxsurvey.Tab
--   :login  login pemanggil, dipakai ketiga tab komunikasi
--   :msg_open  KOMUNIKASISTATUS terbuka        -> "0"
--   :msg_answered  KOMUNIKASISTATUS sudah dijawab  -> "1"
--   :search  kata kunci pencarian, atau NULL bila kotak carinya kosong
--   :skip  offset
--   :take  jumlah baris
SELECT s.CASEID            AS SURVEY_ID,
       s.PNCCASEID         AS CLAIM_ID,
       s.INDEX_SURVEY      AS SURVEY_INDEX,
       CAST(NULL AS VARCHAR2(101))
                           AS REFERENCE_NUMBER,
       c.CLAIMNO           AS CLAIM_NUMBER,
       c.NOPOLIS           AS POLICY_NUMBER,
       c.QQNAME            AS INSURED_NAME,
       c.BUSINESSNAME      AS CLASS_OF_BUSINESS,
       (SELECT cov.CAUSEOFLOSS
          FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE cov
         WHERE cov.CLAIMID = s.PNCCASEID
         FETCH NEXT 1 ROW ONLY)
                           AS CAUSE_OF_LOSS,
       s.LOCATION_SURVEY   AS LOCATION,
       c.PICTEKNIK         AS TECHNICAL_PIC,
       s.SURVEYOR_NAME     AS ADJUSTER_PIC,
       c.DATEOFLOSS        AS DATE_OF_LOSS,
       s.TGLINPUT          AS CREATED_AT,
       c.LEADER_MEMBER     AS ASM_STATUS,
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
   AND INSTR(:scope, '|' || UPPER(TRIM(s.SURVEYOR_NAME)) || '|') > 0
   AND ((:tab = 'belum-dijawab'
         AND EXISTS (SELECT 1
                       FROM POOLDATA.M_KOMUNIKASI_PNC kom
                      WHERE kom.CASEID = s.CASEID
                        AND UPPER(TRIM(kom.SENDER)) <> UPPER(TRIM(:login))
                        AND kom.KOMUNIKASISTATUS = :msg_open))
     OR (:tab = 'belum-dibalas-asm'
         AND EXISTS (SELECT 1
                       FROM POOLDATA.M_KOMUNIKASI_PNC kom
                      WHERE kom.CASEID = s.CASEID
                        AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:login))
                        AND kom.KOMUNIKASISTATUS = :msg_open))
     OR (:tab = 'sudah-dibalas-asm'
         AND EXISTS (SELECT 1
                       FROM POOLDATA.M_KOMUNIKASI_PNC kom
                      WHERE kom.CASEID = s.CASEID
                        AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:login))
                        AND kom.KOMUNIKASISTATUS = :msg_answered)))
   AND (:search IS NULL
        OR UPPER(c.CLAIMNO) LIKE '%' || UPPER(:search) || '%')
 ORDER BY s.TGLINPUT DESC NULLS LAST, s.CASEID DESC
OFFSET :skip ROWS FETCH NEXT :take ROWS ONLY

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
-- Bind — ANGKANYA MENGIKUTI URUTAN KEMUNCULAN, bukan urutan yang enak dibaca. Lihat banner
-- berkas ini; menukar urutan kondisi di bawah berarti menukar arti angkanya.
--   :login  login pemanggil
--   :msg_open  KOMUNIKASISTATUS terbuka        -> "0"
--   :msg_answered  KOMUNIKASISTATUS sudah dijawab  -> "1"
--   :scope  cakupan nama surveyor
SELECT SUM(CASE WHEN EXISTS (SELECT 1
                               FROM POOLDATA.M_KOMUNIKASI_PNC kom
                              WHERE kom.CASEID = s.CASEID
                                AND UPPER(TRIM(kom.SENDER)) <> UPPER(TRIM(:login))
                                AND kom.KOMUNIKASISTATUS = :msg_open)
                THEN 1 ELSE 0 END)
          AS COUNT_NOT_ANSWERED,
       SUM(CASE WHEN EXISTS (SELECT 1
                               FROM POOLDATA.M_KOMUNIKASI_PNC kom
                              WHERE kom.CASEID = s.CASEID
                                AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:login))
                                AND kom.KOMUNIKASISTATUS = :msg_open)
                THEN 1 ELSE 0 END)
          AS COUNT_NOT_REPLIED,
       SUM(CASE WHEN EXISTS (SELECT 1
                               FROM POOLDATA.M_KOMUNIKASI_PNC kom
                              WHERE kom.CASEID = s.CASEID
                                AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:login))
                                AND kom.KOMUNIKASISTATUS = :msg_answered)
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
   AND INSTR(:scope, '|' || UPPER(TRIM(s.SURVEYOR_NAME)) || '|') > 0

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
--   :login  login pemanggil
SELECT m.LOGIN        AS SURVEYOR_LOGIN,
       m.NAMA         AS SURVEYOR_NAME,
       m.LOGINLEADER  AS LEADER_LOGIN,
       m.STSLOGIN     AS LOGIN_STATUS
  FROM POOLDATA.MST_LOGIN_SURVEYOR m
 WHERE UPPER(TRIM(m.LOGIN)) = UPPER(TRIM(:login))

-- name: resolve_members
-- Nama seluruh surveyor yang berada di bawah seorang leader.
--
-- Dicocokkan pada `LOGINLEADER`, bukan pada `NAMA`: kolom itu menyimpan LOGIN atasan — terbaca
-- dari `GetLoginLeaderSurveyor` yang membandingkannya dengan `OperatorID.pyUserIdentifier`.
--
-- Bind:
--   :leader_login  login leader
SELECT m.NAMA AS SURVEYOR_NAME
  FROM POOLDATA.MST_LOGIN_SURVEYOR m
 WHERE UPPER(TRIM(m.LOGINLEADER)) = UPPER(TRIM(:leader_login))

-- ============================================================================
-- LIMA KUERI KPI, DAN KENAPA LIMA
-- ============================================================================
--
-- Panel KPI layar lama menjalankan rule yang BERBEDA menurut kombinasi isiannya, dan setiap
-- rule menghasilkan baris yang artinya berbeda pula. Ini bukan lima tampilan dari satu
-- himpunan angka:
--
--	kpi_by_adjuster         GetSummaryKPIAdjuster            satu baris per ADJUSTER
--	kpi_by_adjuster_all     GetSummaryKPIAdjusterALL         DUA baris per adjuster
--	kpi_by_year             GetSummaryKPIAdjusterKuartal     satu baris per TAHUN
--	kpi_by_quarter_year     GetSummaryKPIAdjusterALLKuartal  EMPAT baris per tahun
--	kpi_detail              GetDetailKPIAdjusterKuartal      satu baris per BERKAS
--
-- Pemilihnya `inboxsurvey.KPIFilter.Shape()`, yang meniru percabangan
-- `Activity/GetReportKPIAdjuster-Act.xml`.
--
--
-- # Ketiga `{ASIS:...}` warisan dan penggantinya
--
-- Ketiga rule aslinya menempelkan penyaringnya sebagai TEKS SQL — pola yang
-- `03-CURRENT-ARCHITECTURE.md` §4.5 catat sebagai celah injeksi:
--
--	TempAdjComp.UploadLOD  := "and adjuster='" + TempAdjComp.NameOfBank + "'"
--	Filter.Province        := "and to_char(tanggal,'yyyy') = '" + TempAdjComp.IsDLA + "'"
--	Filter.ProdKe          := "and to_char(tanggal,'mm') in ('07','08','09')"
--
-- Ketiganya menjadi parameter di sini. Yang pertama diganti penyaring CAKUPAN berbasis
-- `INSTR` — bukan satu nama, melainkan daftar nama yang boleh dilihat pemanggil, karena
-- tabel ini memuat penilaian SELURUH adjuster dan tanpa penyaring itu tab ini berubah
-- menjadi papan peringkat yang tidak pernah diminta siapa pun.
--
--
-- # Satu keanehan yang DIPERTAHANKAN
--
-- Ketiga jalur berkuartal mematok `tipe='FINAL'` DI DALAM rule-nya masing-masing — Status
-- Survey tidak berpengaruh di sana, bahkan ketika dipilih OUTSTANDING. `P-5` menetapkan
-- perilakunya yang dibawa, bukan yang masuk akal.
--
--
-- # `to_number` dibawa apa adanya
--
-- Adanya fungsi itu di sistem lama menyiratkan kolomnya bertipe TEKS. Menghilangkannya akan
-- gagal ORA-01722 pada baris pertama yang bukan angka — dan kegagalan itu justru keterangan
-- yang berguna.
--
--
-- # Bentuk kolom KELIMA kueri DISERAGAMKAN
--
-- Keempat kolom pertama selalu `GROUP_KEY`, `STATUS_KEY`, `QUARTER_KEY`, `MONTH_KEY`,
-- `CASE_KEY`, diikuti kesembilan angka — kolom yang tidak berlaku diisi NULL. Dengan begitu
-- kelimanya dibaca `scanKPI` yang SATU, dan satu kolom yang bergeser tidak dapat memindahkan
-- angka ke kolom tetangganya tanpa ada yang menyadarinya.

-- name: kpi_by_adjuster
-- Ringkasan per ADJUSTER untuk satu nilai `tipe`.
--
-- Penerjemahan `GetSummaryKPIAdjuster-SQL.xml`, yang menerima `tipe` dari layar
-- (`where tipe = {TempAdjComp.ASMFull}`) — satu-satunya dari kelimanya yang begitu.
--
-- Bind: :scope cakupan nama adjuster · :kpi_type nilai kolom `tipe`
SELECT d.ADJUSTER                                    AS GROUP_KEY,
       CAST(NULL AS VARCHAR2(20))                    AS STATUS_KEY,
       CAST(NULL AS VARCHAR2(4))                     AS QUARTER_KEY,
       CAST(NULL AS VARCHAR2(4))                     AS MONTH_KEY,
       CAST(NULL AS VARCHAR2(64))                    AS CASE_KEY,
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
 WHERE INSTR(:scope, '|' || UPPER(TRIM(d.ADJUSTER)) || '|') > 0
   AND UPPER(TRIM(d.TIPE)) = UPPER(TRIM(:kpi_type))
 GROUP BY d.ADJUSTER
 ORDER BY d.ADJUSTER

-- name: kpi_by_adjuster_all
-- Ringkasan per adjuster untuk KEDUA kategori sekaligus — Status Survey "ALL".
--
-- Penerjemahan `GetSummaryKPIAdjusterALL-SQL.xml` apa adanya: DUA blok `UNION ALL`, masing-
-- masing memakai `tipe` tetapnya sendiri dan membawa label kategorinya sebagai kolom.
--
-- # Kenapa BUKAN satu kueri tanpa penyaring `tipe`
--
-- Karena hasilnya berbeda, bukan sekadar disusun berbeda. Satu kueri tanpa penyaring
-- menghasilkan SATU baris per adjuster yang merata-ratakan kedua kategori menjadi satu
-- angka; Pega menghasilkan DUA baris dengan angka masing-masing. Angka yang pertama tidak
-- pernah ada di layar lama, dan ia terlihat sangat masuk akal — itu yang membuatnya
-- berbahaya.
--
-- `UNION ALL`, bukan `UNION`: dua baris yang kebetulan sama angkanya tetap dua baris, karena
-- keduanya menyatakan kategori yang berbeda.
--
-- Bind: :scope cakupan nama adjuster
SELECT GROUP_KEY, STATUS_KEY, QUARTER_KEY, MONTH_KEY, CASE_KEY,
       SURVEY_SCHEDULING, IMMEDIATE_ADVICE, PRELIMINARY_ADVICE, INTERIM_REPORT,
       PROGRESS_UPDATE, COMMUNICATION_RESPONSE, PROPOSE_ADJUSTMENT, FINAL_REPORT, VALUE_SCORE
  FROM (SELECT d.ADJUSTER                                    AS GROUP_KEY,
               'OUTSTANDING'                                 AS STATUS_KEY,
               CAST(NULL AS VARCHAR2(4))                     AS QUARTER_KEY,
               CAST(NULL AS VARCHAR2(4))                     AS MONTH_KEY,
               CAST(NULL AS VARCHAR2(64))                    AS CASE_KEY,
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
         WHERE INSTR(:scope, '|' || UPPER(TRIM(d.ADJUSTER)) || '|') > 0
           AND UPPER(TRIM(d.TIPE)) = 'OUTSTANDING'
         GROUP BY d.ADJUSTER
        UNION ALL
        SELECT d.ADJUSTER,
               'FINAL',
               CAST(NULL AS VARCHAR2(4)),
               CAST(NULL AS VARCHAR2(4)),
               CAST(NULL AS VARCHAR2(64)),
               ROUND(AVG(TO_NUMBER(d.SURVEYLAP)), 2),
               ROUND(AVG(TO_NUMBER(d.IMMEDIATEADVICE)), 2),
               ROUND(AVG(TO_NUMBER(d.PRELIMINARYADVICE)), 2),
               ROUND(AVG(TO_NUMBER(d.INTERIM)), 2),
               ROUND(AVG(TO_NUMBER(d.PROGRESS)), 2),
               ROUND(AVG(TO_NUMBER(d.KOMUNIKASI)), 2),
               ROUND(AVG(TO_NUMBER(d.PROPOSE)), 2),
               ROUND(AVG(TO_NUMBER(d.FINALREPORT)), 2),
               ROUND(AVG(TO_NUMBER(d.NILAI)), 2)
          FROM POOLDATA.DETAIL_KPI_ADJUSTER d
         WHERE INSTR(:scope, '|' || UPPER(TRIM(d.ADJUSTER)) || '|') > 0
           AND UPPER(TRIM(d.TIPE)) = 'FINAL'
         GROUP BY d.ADJUSTER) gabungan
 ORDER BY GROUP_KEY, STATUS_KEY

-- name: kpi_by_year
-- Ringkasan per TAHUN untuk SATU kuartal.
--
-- Penerjemahan `GetSummaryKPIAdjusterKuartal-SQL.xml`. Namanya menyebut kuartal,
-- pengelompokannya `to_char(tanggal,'yyyy')` — per TAHUN; kuartalnya ada di PENYARING, bukan
-- di pengelompokan. Perilakunya yang dibawa, bukan namanya (`P-5`).
--
-- `tipe='FINAL'` dipatok di dalam rule aslinya, sehingga Status Survey tidak berpengaruh di
-- jalur ini. Dipertahankan apa adanya.
--
-- Kuartal diterjemahkan menjadi `TO_CHAR(TANGGAL,'Q')` — satu predikat, bukan daftar bulan.
-- Keduanya memilih baris yang sama persis, dan `'Q'` berlaku sama di Oracle dan PostgreSQL
-- (`D-20`), sementara daftar bulan harus dirangkai dari luar.
--
-- Bind: :scope · :year tahun atau NULL · :quarter "1".."4" atau NULL
SELECT TO_CHAR(d.TANGGAL, 'yyyy')                    AS GROUP_KEY,
       CAST(NULL AS VARCHAR2(20))                    AS STATUS_KEY,
       CAST(NULL AS VARCHAR2(4))                     AS QUARTER_KEY,
       CAST(NULL AS VARCHAR2(4))                     AS MONTH_KEY,
       CAST(NULL AS VARCHAR2(64))                    AS CASE_KEY,
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
 WHERE INSTR(:scope, '|' || UPPER(TRIM(d.ADJUSTER)) || '|') > 0
   AND UPPER(TRIM(d.TIPE)) = 'FINAL'
   AND (:year IS NULL OR TO_CHAR(d.TANGGAL, 'yyyy') = :year)
   AND (:quarter IS NULL OR TO_CHAR(d.TANGGAL, 'Q') = :quarter)
 GROUP BY TO_CHAR(d.TANGGAL, 'yyyy')
 ORDER BY TO_CHAR(d.TANGGAL, 'yyyy') DESC

-- name: kpi_by_quarter_year
-- Ringkasan KEEMPAT kuartal sekaligus, per tahun — pilihan Kuartal "ALL".
--
-- Penerjemahan `GetSummaryKPIAdjusterALLKuartal-SQL.xml` **apa adanya**: empat blok `UNION`,
-- masing-masing menyaring tiga bulan dan membawa label kuartalnya sebagai konstanta.
--
-- # Kenapa empat blok, padahal satu `GROUP BY` cukup
--
-- Versi sebelumnya berkas ini memakai satu `GROUP BY ... , TO_CHAR(TANGGAL,'Q')`, yang
-- menghasilkan baris yang sama persis dengan biaya satu kali baca alih-alih empat. Bentuk itu
-- **dikembalikan** atas keputusan Work Owner 2026-10-08: ikuti Pega.
--
-- Harganya nyata dan diterima: tabelnya dibaca empat kali. Yang dibeli juga nyata — bila kelak
-- salah satu blok Pega ternyata berbeda dari ketiga saudaranya (daftar bulan yang tidak
-- simetris, penyaring tambahan di satu blok), perbedaan itu akan terbawa dengan sendirinya,
-- bukan hilang ke dalam penyederhanaan yang kelihatannya setara.
--
-- `UNION`, bukan `UNION ALL` — sama dengan rule aslinya. Keempat blok tidak pernah
-- menghasilkan baris kembar karena labelnya berbeda, sehingga pilihan itu tidak mengubah
-- hasilnya; ia ditiru supaya tidak ada satu pun keputusan yang berbeda tanpa alasan.
--
-- Bind: :scope · :year tahun atau NULL
SELECT TO_CHAR(d.TANGGAL, 'yyyy')                    AS GROUP_KEY,
       CAST(NULL AS VARCHAR2(20))                    AS STATUS_KEY,
       '1'                                           AS QUARTER_KEY,
       CAST(NULL AS VARCHAR2(4))                     AS MONTH_KEY,
       CAST(NULL AS VARCHAR2(64))                    AS CASE_KEY,
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
 WHERE INSTR(:scope, '|' || UPPER(TRIM(d.ADJUSTER)) || '|') > 0
   AND UPPER(TRIM(d.TIPE)) = 'FINAL'
   AND (:year IS NULL OR TO_CHAR(d.TANGGAL, 'yyyy') = :year)
   AND TO_CHAR(d.TANGGAL, 'mm') IN ('01', '02', '03')
 GROUP BY TO_CHAR(d.TANGGAL, 'yyyy')
UNION
SELECT TO_CHAR(d.TANGGAL, 'yyyy'),
       CAST(NULL AS VARCHAR2(20)),
       '2',
       CAST(NULL AS VARCHAR2(4)),
       CAST(NULL AS VARCHAR2(64)),
       ROUND(AVG(TO_NUMBER(d.SURVEYLAP)), 2),
       ROUND(AVG(TO_NUMBER(d.IMMEDIATEADVICE)), 2),
       ROUND(AVG(TO_NUMBER(d.PRELIMINARYADVICE)), 2),
       ROUND(AVG(TO_NUMBER(d.INTERIM)), 2),
       ROUND(AVG(TO_NUMBER(d.PROGRESS)), 2),
       ROUND(AVG(TO_NUMBER(d.KOMUNIKASI)), 2),
       ROUND(AVG(TO_NUMBER(d.PROPOSE)), 2),
       ROUND(AVG(TO_NUMBER(d.FINALREPORT)), 2),
       ROUND(AVG(TO_NUMBER(d.NILAI)), 2)
  FROM POOLDATA.DETAIL_KPI_ADJUSTER d
 WHERE INSTR(:scope, '|' || UPPER(TRIM(d.ADJUSTER)) || '|') > 0
   AND UPPER(TRIM(d.TIPE)) = 'FINAL'
   AND (:year IS NULL OR TO_CHAR(d.TANGGAL, 'yyyy') = :year)
   AND TO_CHAR(d.TANGGAL, 'mm') IN ('04', '05', '06')
 GROUP BY TO_CHAR(d.TANGGAL, 'yyyy')
UNION
SELECT TO_CHAR(d.TANGGAL, 'yyyy'),
       CAST(NULL AS VARCHAR2(20)),
       '3',
       CAST(NULL AS VARCHAR2(4)),
       CAST(NULL AS VARCHAR2(64)),
       ROUND(AVG(TO_NUMBER(d.SURVEYLAP)), 2),
       ROUND(AVG(TO_NUMBER(d.IMMEDIATEADVICE)), 2),
       ROUND(AVG(TO_NUMBER(d.PRELIMINARYADVICE)), 2),
       ROUND(AVG(TO_NUMBER(d.INTERIM)), 2),
       ROUND(AVG(TO_NUMBER(d.PROGRESS)), 2),
       ROUND(AVG(TO_NUMBER(d.KOMUNIKASI)), 2),
       ROUND(AVG(TO_NUMBER(d.PROPOSE)), 2),
       ROUND(AVG(TO_NUMBER(d.FINALREPORT)), 2),
       ROUND(AVG(TO_NUMBER(d.NILAI)), 2)
  FROM POOLDATA.DETAIL_KPI_ADJUSTER d
 WHERE INSTR(:scope, '|' || UPPER(TRIM(d.ADJUSTER)) || '|') > 0
   AND UPPER(TRIM(d.TIPE)) = 'FINAL'
   AND (:year IS NULL OR TO_CHAR(d.TANGGAL, 'yyyy') = :year)
   AND TO_CHAR(d.TANGGAL, 'mm') IN ('07', '08', '09')
 GROUP BY TO_CHAR(d.TANGGAL, 'yyyy')
UNION
SELECT TO_CHAR(d.TANGGAL, 'yyyy'),
       CAST(NULL AS VARCHAR2(20)),
       '4',
       CAST(NULL AS VARCHAR2(4)),
       CAST(NULL AS VARCHAR2(64)),
       ROUND(AVG(TO_NUMBER(d.SURVEYLAP)), 2),
       ROUND(AVG(TO_NUMBER(d.IMMEDIATEADVICE)), 2),
       ROUND(AVG(TO_NUMBER(d.PRELIMINARYADVICE)), 2),
       ROUND(AVG(TO_NUMBER(d.INTERIM)), 2),
       ROUND(AVG(TO_NUMBER(d.PROGRESS)), 2),
       ROUND(AVG(TO_NUMBER(d.KOMUNIKASI)), 2),
       ROUND(AVG(TO_NUMBER(d.PROPOSE)), 2),
       ROUND(AVG(TO_NUMBER(d.FINALREPORT)), 2),
       ROUND(AVG(TO_NUMBER(d.NILAI)), 2)
  FROM POOLDATA.DETAIL_KPI_ADJUSTER d
 WHERE INSTR(:scope, '|' || UPPER(TRIM(d.ADJUSTER)) || '|') > 0
   AND UPPER(TRIM(d.TIPE)) = 'FINAL'
   AND (:year IS NULL OR TO_CHAR(d.TANGGAL, 'yyyy') = :year)
   AND TO_CHAR(d.TANGGAL, 'mm') IN ('10', '11', '12')
 GROUP BY TO_CHAR(d.TANGGAL, 'yyyy')
 ORDER BY 1 DESC, 3

-- name: kpi_years
-- Isi dropdown **Tahun Kuartal**.
--
-- # Kenapa ia ada, dan kenapa isinya rekonstruksi
--
-- Layar lama mengisi kendali itu dari `TahunKPI.pxResults` — sebuah page list yang TIDAK
-- PERNAH diisi di mana pun dalam export (`R-16`), sama seperti `TempKuartal.pxResults`. Yang
-- terbaca hanyalah BENTUKNYA: `pySourceName = TahunKPI.pxResults` dengan `--Pilih--` di
-- puncaknya, yaitu sebuah dropdown.
--
-- Isinya karena itu direkonstruksi dari data: tahun yang BENAR-BENAR ada pada baris milik
-- cakupan pemanggil. Itu daftar yang tidak pernah menawarkan tahun yang hasilnya pasti kosong,
-- dan tidak pernah menyembunyikan tahun yang datanya ada.
--
-- Bila rule pengisi aslinya kelak tiba dan ternyata berbeda — misalnya rentang tetap, atau
-- tahun berjalan ditambah lima ke belakang — kueri inilah yang diganti, bukan layarnya.
--
-- Disaring CAKUPAN, sama seperti seluruh kueri KPI lain: tabel ini memuat penilaian seluruh
-- adjuster, dan daftar tahun pun tidak boleh membocorkan keberadaan data orang lain.
--
-- Bind: :scope
SELECT DISTINCT TO_CHAR(d.TANGGAL, 'yyyy') AS YEAR_KEY
  FROM POOLDATA.DETAIL_KPI_ADJUSTER d
 WHERE INSTR(:scope, '|' || UPPER(TRIM(d.ADJUSTER)) || '|') > 0
   AND d.TANGGAL IS NOT NULL
 ORDER BY YEAR_KEY DESC

-- name: kpi_detail
-- Laporan DATA DETAIL — satu baris per BERKAS, angkanya MENTAH.
--
-- Penerjemahan `GetDetailKPIAdjusterKuartal-SQL.xml`. Tidak ada satu pun `AVG` di sini, dan
-- tidak ada `GROUP BY`: setiap baris tabel menjadi satu baris laporan.
--
-- Itu yang membuatnya laporan BERBEDA, bukan tampilan lain dari ringkasan. Angka 85 pada
-- ringkasan adalah rata-rata; angka 85 di sini adalah nilai satu berkas.
--
-- Rule aslinya mengalias kolomnya menjadi properti klipboard yang tidak ada hubungannya
-- dengan isinya — `to_char(tanggal,'mm') as "Password"`, `adjuster as "UserTeknisGroup"`.
-- Alias itu TIDAK dibawa; kolomnya disebut nama aslinya (`03-CURRENT-ARCHITECTURE.md` §4.2).
--
-- Bind: :scope · :year · :quarter
SELECT d.ADJUSTER                   AS GROUP_KEY,
       CAST(NULL AS VARCHAR2(20))   AS STATUS_KEY,
       TO_CHAR(d.TANGGAL, 'Q')      AS QUARTER_KEY,
       TO_CHAR(d.TANGGAL, 'mm')     AS MONTH_KEY,
       d.CASEID                     AS CASE_KEY,
       TO_NUMBER(d.SURVEYLAP)       AS SURVEY_SCHEDULING,
       TO_NUMBER(d.IMMEDIATEADVICE) AS IMMEDIATE_ADVICE,
       TO_NUMBER(d.PRELIMINARYADVICE) AS PRELIMINARY_ADVICE,
       TO_NUMBER(d.INTERIM)         AS INTERIM_REPORT,
       TO_NUMBER(d.PROGRESS)        AS PROGRESS_UPDATE,
       TO_NUMBER(d.KOMUNIKASI)      AS COMMUNICATION_RESPONSE,
       TO_NUMBER(d.PROPOSE)         AS PROPOSE_ADJUSTMENT,
       TO_NUMBER(d.FINALREPORT)     AS FINAL_REPORT,
       TO_NUMBER(d.NILAI)           AS VALUE_SCORE
  FROM POOLDATA.DETAIL_KPI_ADJUSTER d
 WHERE INSTR(:scope, '|' || UPPER(TRIM(d.ADJUSTER)) || '|') > 0
   AND UPPER(TRIM(d.TIPE)) = 'FINAL'
   AND (:year IS NULL OR TO_CHAR(d.TANGGAL, 'yyyy') = :year)
   AND (:quarter IS NULL OR TO_CHAR(d.TANGGAL, 'Q') = :quarter)
 ORDER BY d.TANGGAL DESC NULLS LAST, d.CASEID

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
SELECT COUNT(CASE WHEN COLUMN_NAME = 'ADJUSTERACCEPT'     THEN 1 END) AS HAS_ACCEPT,
       COUNT(CASE WHEN COLUMN_NAME = 'REFNO'              THEN 1 END) AS HAS_REFERENCE,
       COUNT(CASE WHEN COLUMN_NAME = 'PYSTATUSWORK'       THEN 1 END) AS HAS_WORK_STATUS,
       COUNT(CASE WHEN COLUMN_NAME = 'ADJUSTER_PIC'        THEN 1 END) AS HAS_ADJUSTER_PIC,
       COUNT(CASE WHEN COLUMN_NAME = 'RESCHEDULE_LOCATION' THEN 1 END) AS HAS_SURVEY_LOCATION
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
-- Ia memuat KELIMA kolom yang ditunggu, dan seluruhnya sudah ada di basis data per 2026-10-03.
-- Menuliskan kolom yang BELUM ada akan membuat kueri ini gagal seluruhnya — termasuk untuk
-- kolom lain yang justru ingin diukur. Yang melaporkan ketiadaan sebuah kolom adalah
-- `check_new_columns` lewat katalog, karena ia tidak dapat gagal karena sebab itu.
SELECT COUNT(*)                     AS TOTAL_ROWS,
       COUNT(s.ADJUSTERACCEPT)      AS FILLED_ACCEPT,
       COUNT(s.REFNO)               AS FILLED_REFERENCE,
       COUNT(s.PYSTATUSWORK)        AS FILLED_WORK_STATUS,
       COUNT(s.ADJUSTER_PIC)         AS FILLED_ADJUSTER_PIC,
       COUNT(s.RESCHEDULE_LOCATION)  AS FILLED_SURVEY_LOCATION
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

-- name: list_tasks_full
-- Kembaran `list_tasks` untuk portal yang KELIMA kolomnya sudah ada DAN terisi.
--
-- # Kenapa dua kueri, bukan satu yang bercabang
--
-- Kolom yang belum ada tidak dapat disebut sama sekali. `ADJUSTERACCEPT` pada portal yang belum
-- di-ALTER menghasilkan ORA-00904 SAAT PARSE — sebelum satu baris pun dibaca, dan tanpa
-- memedulikan `CASE WHEN` apa pun yang membungkusnya. Tidak ada bentuk percabangan di dalam SQL
-- yang dapat menghindarinya.
--
-- Pemilihnya `inboxsurvey.Readiness.Complete()`, dibaca dari katalog kolom portal yang
-- bersangkutan. `D-75` menetapkan satu basis data per entitas, sehingga portal yang sudah
-- di-ALTER dan yang belum HIDUP BERDAMPINGAN — per 2026-10-07 persis begitu: `pega_dev83`
-- sudah, produksi belum.
--
-- # Yang BERBEDA dari list_tasks
--
--   1. Kolom "Reference No" terisi dari `s.REFNO`, bukan tempat kosong
--   2. Kolom "Location" dari `s.RESCHEDULE_LOCATION`, bukan `s.LOCATION_SURVEY`
--   3. Kolom "PIC Loss Adjuster" dari `s.ADJUSTER_PIC`, bukan `s.SURVEYOR_NAME`
--   4. EMPAT cabang tab tambahan: Outstanding, ALL, Invoice, Close
--   5. Ketiga tab komunikasi ikut menyaring berkas yang sudah tutup
--   6. Kotak cari mencari pada Claim No DAN Reference No — seperti layar lama
--
-- Daftar alias dan urutannya SAMA PERSIS dengan `list_tasks`, sehingga keduanya dibaca
-- `scanTask` yang satu. Satu alias yang bergeser di salah satunya akan memindahkan nomor polis
-- ke kolom nama tertanggung, tanpa galat apa pun.
--
-- Bind — ANGKANYA MENGIKUTI URUTAN KEMUNCULAN, bukan urutan yang enak dibaca. Lihat banner
-- berkas ini; memindahkan satu cabang tab berarti menggeser arti seluruh angka sesudahnya.
--   :scope  cakupan nama surveyor, berbentuk |NAMA SATU|NAMA DUA|
--   :tab  tab yang dibuka — nilai inboxsurvey.Tab
--   :work_done  status kerja SELESAI   ·  :work_rejected  status kerja DITOLAK
--   :adjuster_confirmed  nilai ADJUSTERACCEPT yang berarti sudah dikonfirmasi
--   :invoice_fee  nilai STS_SURVEY untuk tab Invoice
--   :login  login pemanggil
--   :msg_open  status pesan TERBUKA   ·  :msg_answered  status pesan SUDAH DIBALAS
--   :search kata cari, boleh NULL
--   :skip offset                 ·  :take jumlah baris
SELECT s.CASEID               AS SURVEY_ID,
       s.PNCCASEID            AS CLAIM_ID,
       s.INDEX_SURVEY         AS SURVEY_INDEX,
       s.REFNO                AS REFERENCE_NUMBER,
       c.CLAIMNO              AS CLAIM_NUMBER,
       c.NOPOLIS              AS POLICY_NUMBER,
       c.QQNAME               AS INSURED_NAME,
       c.BUSINESSNAME         AS CLASS_OF_BUSINESS,
       (SELECT cov.CAUSEOFLOSS
          FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE cov
         WHERE cov.CLAIMID = s.PNCCASEID
         FETCH NEXT 1 ROW ONLY)
                              AS CAUSE_OF_LOSS,
       s.RESCHEDULE_LOCATION  AS LOCATION,
       c.PICTEKNIK            AS TECHNICAL_PIC,
       s.ADJUSTER_PIC         AS ADJUSTER_PIC,
       c.DATEOFLOSS           AS DATE_OF_LOSS,
       s.TGLINPUT             AS CREATED_AT,
       c.LEADER_MEMBER        AS ASM_STATUS,
       s.SURVEYTYPE           AS SURVEYOR_TYPE,
       COUNT(*) OVER ()       AS TOTAL_ROWS
  FROM (SELECT t.*,
               ROW_NUMBER() OVER (PARTITION BY t.CASEID
                                  ORDER BY LPAD(TRIM(t.INDEX_SURVEY), 10, '0') DESC NULLS LAST,
                                           t.TGLINPUT DESC NULLS LAST) AS STEP_RANK
          FROM POOLDATA.T_SURVEYORLIST t) s
       INNER JOIN POOLDATA.T_CLAIM_PNC c
               ON c.CLAIMID = s.PNCCASEID
 WHERE s.STEP_RANK = 1
   AND INSTR(:scope, '|' || UPPER(TRIM(s.SURVEYOR_NAME)) || '|') > 0
   AND ((:tab = 'outstanding'
         AND s.PYSTATUSWORK NOT IN (:work_done, :work_rejected)
         AND s.ADJUSTERACCEPT IS NULL)
     OR (:tab = 'all'
         AND s.PYSTATUSWORK NOT IN (:work_done, :work_rejected)
         AND s.ADJUSTERACCEPT = :adjuster_confirmed)
     OR (:tab = 'invoice'
         AND s.PYSTATUSWORK NOT IN (:work_done, :work_rejected)
         AND s.ADJUSTERACCEPT = :adjuster_confirmed
         AND s.STS_SURVEY = :invoice_fee)
     OR (:tab = 'close'
         AND s.PYSTATUSWORK = :work_done)
     OR (:tab = 'belum-dijawab'
         AND s.PYSTATUSWORK NOT IN (:work_done, :work_rejected)
         AND EXISTS (SELECT 1
                       FROM POOLDATA.M_KOMUNIKASI_PNC kom
                      WHERE kom.CASEID = s.CASEID
                        AND UPPER(TRIM(kom.SENDER)) <> UPPER(TRIM(:login))
                        AND kom.KOMUNIKASISTATUS = :msg_open))
     OR (:tab = 'belum-dibalas-asm'
         AND s.PYSTATUSWORK NOT IN (:work_done, :work_rejected)
         AND EXISTS (SELECT 1
                       FROM POOLDATA.M_KOMUNIKASI_PNC kom
                      WHERE kom.CASEID = s.CASEID
                        AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:login))
                        AND kom.KOMUNIKASISTATUS = :msg_open))
     OR (:tab = 'sudah-dibalas-asm'
         AND s.PYSTATUSWORK NOT IN (:work_done, :work_rejected)
         AND EXISTS (SELECT 1
                       FROM POOLDATA.M_KOMUNIKASI_PNC kom
                      WHERE kom.CASEID = s.CASEID
                        AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:login))
                        AND kom.KOMUNIKASISTATUS = :msg_answered)))
   AND (:search IS NULL
        OR UPPER(c.CLAIMNO) LIKE '%' || UPPER(:search) || '%'
        OR UPPER(s.REFNO)   LIKE '%' || UPPER(:search) || '%')
 ORDER BY s.TGLINPUT DESC NULLS LAST, s.CASEID DESC
OFFSET :skip ROWS FETCH NEXT :take ROWS ONLY

-- name: count_tabs_full
-- Kembaran `count_tabs` untuk portal yang kelima kolomnya sudah siap.
--
-- Mengembalikan KETUJUH angka dalam SATU kueri, bukan tujuh perjalanan — sama seperti
-- `CountOSLostAdjuster` di Pega menghitung ketujuh keranjang sekaligus.
--
-- URUTAN KOLOMNYA WAJIB sama dengan urutan tab pada `inboxsurvey.Tabs()`:
--
--   Outstanding · Invoice · Close · ALL · belum-dijawab · belum-dibalas · sudah-dibalas
--
-- Repo membacanya berurutan terhadap tab yang tersedia. Satu kolom yang bergeser akan menukar
-- jumlah tab Invoice dengan tab Close — dua angka yang sama-sama masuk akal, sehingga
-- tertukarnya tidak akan disadari siapa pun.
--
-- Bind — ANGKANYA MENGIKUTI URUTAN KEMUNCULAN, bukan urutan yang enak dibaca. Lihat banner
-- berkas ini; memindahkan satu SUM berarti menggeser arti seluruh angka sesudahnya.
--   :work_done kerja selesai · :work_rejected kerja ditolak · :adjuster_confirmed adjuster dikonfirmasi · :invoice_fee status Invoice Fee
--   :login login · :msg_open pesan terbuka · :msg_answered pesan dibalas · :scope cakupan nama surveyor
SELECT SUM(CASE WHEN s.PYSTATUSWORK NOT IN (:work_done, :work_rejected)
                 AND s.ADJUSTERACCEPT IS NULL
                THEN 1 ELSE 0 END)
          AS COUNT_OUTSTANDING,
       SUM(CASE WHEN s.PYSTATUSWORK NOT IN (:work_done, :work_rejected)
                 AND s.ADJUSTERACCEPT = :adjuster_confirmed
                 AND s.STS_SURVEY = :invoice_fee
                THEN 1 ELSE 0 END)
          AS COUNT_INVOICE,
       SUM(CASE WHEN s.PYSTATUSWORK = :work_done
                THEN 1 ELSE 0 END)
          AS COUNT_CLOSE,
       SUM(CASE WHEN s.PYSTATUSWORK NOT IN (:work_done, :work_rejected)
                 AND s.ADJUSTERACCEPT = :adjuster_confirmed
                THEN 1 ELSE 0 END)
          AS COUNT_ALL,
       SUM(CASE WHEN s.PYSTATUSWORK NOT IN (:work_done, :work_rejected)
                 AND EXISTS (SELECT 1
                               FROM POOLDATA.M_KOMUNIKASI_PNC kom
                              WHERE kom.CASEID = s.CASEID
                                AND UPPER(TRIM(kom.SENDER)) <> UPPER(TRIM(:login))
                                AND kom.KOMUNIKASISTATUS = :msg_open)
                THEN 1 ELSE 0 END)
          AS COUNT_NOT_ANSWERED,
       SUM(CASE WHEN s.PYSTATUSWORK NOT IN (:work_done, :work_rejected)
                 AND EXISTS (SELECT 1
                               FROM POOLDATA.M_KOMUNIKASI_PNC kom
                              WHERE kom.CASEID = s.CASEID
                                AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:login))
                                AND kom.KOMUNIKASISTATUS = :msg_open)
                THEN 1 ELSE 0 END)
          AS COUNT_NOT_REPLIED,
       SUM(CASE WHEN s.PYSTATUSWORK NOT IN (:work_done, :work_rejected)
                 AND EXISTS (SELECT 1
                               FROM POOLDATA.M_KOMUNIKASI_PNC kom
                              WHERE kom.CASEID = s.CASEID
                                AND UPPER(TRIM(kom.SENDER)) = UPPER(TRIM(:login))
                                AND kom.KOMUNIKASISTATUS = :msg_answered)
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
   AND INSTR(:scope, '|' || UPPER(TRIM(s.SURVEYOR_NAME)) || '|') > 0
