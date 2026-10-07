-- Kueri modul Inbox Admin: antrean kerja petugas admin klaim.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- SELURUH tabel yang dibaca berkas ini milik sistem lama. Tidak ada satu pun pernyataan
-- yang menulis, dan memang tidak boleh ada: selama masa paralel setiap tabel hanya boleh
-- ditulis SATU sistem, dan tabel-tabel ini milik Pega (`P-1`).
--
-- ============================================================================
-- PEMETAAN TIGA ARAH — properti grid Pega -> kolom sebenarnya -> arti
-- ============================================================================
--
-- Ini satu-satunya tempat ketiganya dapat dibandingkan berdampingan. Ia ada karena di
-- layar ini nama properti bukan sekadar menyesatkan — ia BERUBAH ARTI dari tab ke tab,
-- sebab kedelapan grid berbagi satu halaman klipboard yang sama.
--
-- Properti grid     di tab ALL / RCV / Branch Claim         Alias di sini
-- ----------------- -------------------------------------- -----------------
-- .PNCCaseID        A.PYID                                 CASE_ID
-- .TypeOfClaim  (!) A.PZINSKEY (kunci teknis Pega)         REFERENCE
-- .PolicyNo         A.POLICYNO                             POLICY_NUMBER
-- .QQName           A.QQNAME                               INSURED_NAME
-- .JenisDokumen (!) A.BUSINESSNAME                         BUSINESS_NAME
-- .RCVID        (!) A.SOBNAME (sumber bisnis)              BUSINESS_SOURCE
-- .NumberOfDoc~ (!) A.BRANCHNAME                           BRANCH_NAME
-- .UserAdmin    (!) POOLDATA.BRANCH.BRANCHNAME             CLAIM_BRANCH
-- .Keterangan   (!) A.PXCREATEOPERATOR (pembuat)           CREATOR
-- .TglKejadian      A.DATEOFLOSS_1                         LOSS_DATE
-- .ReceivedDate (!) A.REPORTDATE_1 / A.REGISTERDATE_1      REPORT_DATE
-- .pxCreateDate~    A.PXCREATEDATETIME                     INPUT_DATE
-- .DateForAging     T_CLAIM_JOB_PERSONALACCIDENT.INSERTDATE LOD_DATE
-- .TelpPengirim (!) A.NOTREGISTNOTE_1 (catatan)            NOTE
-- .PosisiProgr~ (!) CASE A.PYSTATUSWORK                    CLAIM_POSITION
-- .StatusLock   (!) V_STS_CLAIM.LSC_NOTE                   CLAIM_STATUS
-- .StatusWorkC~ (!) CASE atas STATUSLOD dan FLAGBISNIS     LOD_STATUS
-- .Kurir        (!) B.PXFLOWNAME (nama flow)               tidak dibawa
-- .StatusKomun~ (!) A.KODECABANG_1 (kode cabang)           tidak dibawa
--
-- Properti grid     di tab Request Survey                  Alias di sini
-- ----------------- -------------------------------------- -----------------
-- .PNCCaseID        A.PYID                                 CASE_ID
-- .SubjectEmail (!) T_REQ_SURVEY.CLAIMID                   REFERENCE
-- .StatusKomun~ (!) T_REQ_SURVEY.INPUTDATE (tgl request)   REQUEST_DATE
-- .Kurir        (!) A.BRANCHNAME (cabang polis)            POLICY_BRANCH
-- .Keterangan   (!) T_REQ_SURVEY.BRANCH (cabang survei)    SURVEY_BRANCH
-- .Resource     (!) A.USERTEKNIS_1 (PIC klaim)             TECHNICAL_PIC
-- .RCVID        (!) T_REQ_SURVEY.SURVEYOR                  SURVEYOR
-- .UserAdmin    (!) SUBSTR(T_REQ_SURVEY.SURVEYID, 20, 30)  SURVEY_NUMBER
--
-- Properti grid     di tab Status RCL/PUCL                 Alias di sini
-- ----------------- -------------------------------------- -----------------
-- .ClaimID          A.PYID                                 CASE_ID
-- .ClaimNo      (!) A.PZINSKEY (kunci teknis Pega)         REFERENCE
-- .NewTelpTert~ (!) A.QQNAME (nama tertanggung)            INSURED_NAME
-- .pxCreateDate~(!) A.TANGGALKIRIMPUCL_1                   INBOX_DATE
-- .NoteKomite   (!) A.KOMENTARANALISATOR_1                 ANALYST_NOTE
-- .Status           CASE A.RCL_PUCL_1                      RCL_PUCL_STATUS
-- .TanggalCetak~(!) A.TANGGALCETAKDOKUMENPUCL_1            LETTER_PRINT_DATE
-- .LOGSEEN      (!) A.LAMAKLAIM_1 (lama klaim)             CLAIM_AGE
-- .StsAcceptance(!) A.STATUSKLAIM_1 (status kadaluarsa)    EXPIRY_STATUS
--
-- Tanda (!) menandai nama yang sama sekali tidak menyatakan isinya. Perhatikan `.RCVID`,
-- `.Keterangan`, `.Kurir`, `.UserAdmin`, dan `.StatusKomunikasi`: kelimanya berarti DUA
-- hal berbeda tergantung tab mana yang sedang terbuka. Inilah utang teknis
-- `03-CURRENT-ARCHITECTURE.md` §4.2 dalam bentuk paling pekat, dan alasan nama di kode ini
-- tidak mirip nama di Pega (`D-19`).
--
-- ============================================================================
-- KE-29 ALIAS WAJIB SAMA DI SETIAP KUERI
-- ============================================================================
--
-- Urutan DAN namanya. Dua hal bergantung padanya:
--
--   * satu pemindai Go melayani ketujuh kueri (scanWorkItem di inboxadmin.go);
--   * uji query_test.go menjaganya, dan ia gagal bila ada kueri yang aliasnya berbeda.
--
-- Kolom yang tidak berlaku bagi sebuah tab bernilai NULL, bukan dihilangkan. Layar
-- menyembunyikannya mengikuti Tab.Columns — bukan menampilkan kolom kosong yang membuat
-- pengguna menduga datanya hilang.
--
-- ============================================================================
-- LIMA HAL YANG BERUBAH DARI KUERI LAMA, DAN ALASANNYA
-- ============================================================================
--
-- 1. PARAMETER BINDING, bukan perangkaian nilai.
--    Ketujuh kueri lama menyusun klausa WHERE-nya dari potongan SQL yang dirangkai di
--    activity lalu disisipkan lewat `{ASIS:TempView.Remark}`, `{ASIS:TempView.Currency}`,
--    `{ASIS:TempView.pyNote}`, `{ASIS:TempView.Location}`, `{ASIS:TempView.UserAdmin}`,
--    `{ASIS:TempContents.Keyword}`, dan `{ASIS:TempContents.NoteKasir}` — tujuh titik,
--    dan yang terakhir bahkan menyisipkan klausa paginasinya.
--    `08-TECHNICAL-STRATEGY.md` §4.3 melarangnya tanpa perkecualian, dan larangan itu
--    TIDAK ikut dikecualikan oleh keputusan "replikasi apa adanya": yang direplikasi
--    adalah perilaku bisnis, bukan celah injeksi.
--
-- 2. SETIAP kueri mengembalikan ke-29 kolom yang sama, sebagian NULL.
--    Lihat bagian di atas.
--
-- 3. TO_CHAR dibuang; tanggal dikembalikan sebagai DATE.
--    `BrowseRequestSurvey` mengembalikan tanggal sebagai teks `'dd/mm/yyyy'`, sehingga
--    pengurutannya menjadi pengurutan TEKS. Pemformatan pindah ke Go
--    (`09-DATABASE-STRATEGY.md` §3.2).
--
-- 4. ORDER BY ditambahkan pada dua kueri yang tidak punya.
--    `GetAllCaseAdmin` dan `GetRequestDokumenKomunikasi` tidak mengurutkan hasilnya sama
--    sekali. Itu dapat dibiarkan selama seluruh baris ditarik sekaligus DAN tidak
--    dipaginasi; begitu halamannya dipotong, urutan yang tidak ditetapkan membuat satu
--    baris muncul di dua halaman sekaligus hilang dari halaman lain.
--
-- 5. LIKE memakai ESCAPE.
--    Tanpa itu, satu tanda `%` yang diketik pengguna di kotak cari berubah menjadi pola
--    dan mengembalikan seluruh isi antrean.
--
-- ============================================================================
-- YANG SENGAJA TIDAK BERUBAH
-- ============================================================================
--
-- * PAGINASI TIDAK DILAKUKAN DI SINI. Tidak ada `OFFSET ... FETCH NEXT`, dan itu bukan
--   kelalaian: Work Owner memutuskan 2026-09-20 paginasi layar ini direplikasi apa adanya.
--   Sistem lama menarik seluruh baris yang cocok, lalu menghitung totalnya dari baris yang
--   sudah terlanjur ditarik. Pemotongan halaman terjadi di aplikasi — lihat
--   inboxadmin.Slice, yang juga memuat konsekuensi yang diterima secara sadar.
--
-- * PENYARING CABANG DAN KANWIL dipasang 2026-10-07 atas permintaan Work Owner, mencabut
--   penundaan 2026-09-20. Potongan `{ASIS:TempView.Currency}` (cabang) dan
--   `{ASIS:TempView.Remark}` (kanwil) kini menjadi klausa ber-parameter di akhir setiap
--   kueri yang memakainya. Cabang petugas dibaca `branch_of_login` lewat DB Link HRD yang
--   sama dengan `GetIDCabang` (`R-03` -- API penggantinya belum ada). Aturannya di
--   inboxadmin/scope.go.
--
-- * LEFT JOIN ke T_CLAIM_DATACABANG DIPERTAHANKAN meski tak satu pun kolomnya dibawa.
--   Ia dapat menggandakan baris bila satu NOKLAIM punya lebih dari satu baris di sana,
--   dan penggandaan itu terlihat pengguna sebagai jumlah baris. Membuangnya akan mengubah
--   jumlah baris tanpa satu pun pesan galat.
--
-- * Saringan `PYSTATUSWORK NOT IN ('Resolved-Completed','Resolved-Rejected')`, daftar
--   `pxtasklabel`, pengecualian cabang `ASNET`, dan `pxflowname not like
--   '%FixCorrespondence%'` dibawa apa adanya. Keempatnya MENYARING, bukan memperkaya.

-- name: list_all
-- Tab ALL (`TempView.CityID = '3'`) — RDB List/BrowseClaimALL-SQL.xml
--
-- Bind (urut kemunculan): kata kunci (boleh NULL) x3 · lini bisnis ('ALL' bila tanpa
-- saringan) x5. Setiap kemunculan bernomor sendiri -- lihat bindArgs di inboxadmin.go.
SELECT A.PYID                                            AS CASE_ID,
       A.PZINSKEY                                        AS REFERENCE,
       A.POLICYNO                                        AS POLICY_NUMBER,
       A.QQNAME                                          AS INSURED_NAME,
       A.BUSINESSNAME                                    AS BUSINESS_NAME,
       A.SOBNAME                                         AS BUSINESS_SOURCE,
       A.BRANCHNAME                                      AS BRANCH_NAME,
       (SELECT br.BRANCHNAME FROM POOLDATA.BRANCH br
         WHERE br.ID = A.KODECABANG_1)                   AS CLAIM_BRANCH,
       A.PXCREATEOPERATOR                                AS CREATOR,
       A.DATEOFLOSS_1                                    AS LOSS_DATE,
       A.REPORTDATE_1                                    AS REPORT_DATE,
       A.PXCREATEDATETIME                                AS INPUT_DATE,
       CAST(NULL AS DATE)                                AS LOD_DATE,
       CAST(NULL AS VARCHAR2(400))                       AS NOTE,
       CASE A.PYSTATUSWORK
            WHEN 'New'                THEN 'On Progress'
            WHEN 'Resolved-Rejected'  THEN 'Reject'
            WHEN 'Resolved-Completed' THEN 'Close'
       END                                               AS CLAIM_POSITION,
       (SELECT s.LSC_NOTE FROM POOLDATA.V_STS_CLAIM s
         WHERE s.LSC_ID = A.STATUSCLAIM_1)               AS CLAIM_STATUS,
       CAST(NULL AS VARCHAR2(40))                        AS LOD_STATUS,
       CAST(NULL AS DATE)                                AS REQUEST_DATE,
       CAST(NULL AS VARCHAR2(200))                       AS POLICY_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS SURVEY_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(200))                       AS SURVEYOR,
       CAST(NULL AS VARCHAR2(100))                       AS SURVEY_NUMBER,
       CAST(NULL AS DATE)                                AS INBOX_DATE,
       CAST(NULL AS VARCHAR2(2000))                      AS ANALYST_NOTE,
       CAST(NULL AS VARCHAR2(40))                        AS RCL_PUCL_STATUS,
       CAST(NULL AS DATE)                                AS LETTER_PRINT_DATE,
       CAST(NULL AS VARCHAR2(100))                       AS CLAIM_AGE,
       CAST(NULL AS VARCHAR2(100))                       AS EXPIRY_STATUS
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
       LEFT JOIN POOLDATA.T_CLAIM_DATACABANG E
              ON A.PZINSKEY = E.NOKLAIM
       INNER JOIN POOLDATA.BUSINESS c
              ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
              ON c.BUSINESSGROUPID = d.ID
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND A.PXFLOWNAME NOT LIKE '%FixCorrespondence%'
   AND A.PXTASKLABEL IN ('Input Register', 'Input Estimasi', 'Estimation')
   AND (A.BRANCHNAME <> 'ASNET' OR A.BRANCHNAME IS NULL)
   AND (:1 IS NULL
        OR A.PYID LIKE '%' || :2 || '%' ESCAPE '\'
        OR A.POLICYNO LIKE '%' || :3 || '%' ESCAPE '\')
   AND (:4 = 'ALL'
        OR (:5 = 'NONMBU'
            AND A.GROUPPANEL_1 IN ('003', '004', '006', '009')
            AND c.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023'))
        OR (:6 = 'BONDING'
            AND c.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023'))
        OR (:7 = 'PA' AND A.GROUPPANEL_1 = '002')
        OR (:8 = 'TRAVEL' AND A.GROUPPANEL_1 = '005'))
   -- Batas cabang petugas (langkah 8). NULL = tidak dibatasi; lihat inboxadmin.ResolveScope.
   AND (:9 IS NULL OR A.KODECABANG_1 = :10)
   -- Kanwil pilihan manajer (langkah 2). NULL = tidak dibatasi.
   AND (:11 IS NULL
        OR A.KODECABANG_1 IN (SELECT br.ID FROM POOLDATA.BRANCH br
                   WHERE br.BASTERRITORY = :12))
 ORDER BY A.PXCREATEDATETIME DESC

-- name: list_unregistered
-- Tab Unregistered RCV (`CityID = '7'`) dan Unregistered RCV Online (`'8'`)
-- — RDB List/BrowseClaimNotRegistAll-SQL.xml
--
-- Satu kueri melayani DUA tab, persis seperti di Pega: yang membedakan keduanya hanyalah
-- saringan kurir, yang di sistem lama disisipkan sebagai `{ASIS:TempView.Location}` dari
-- langkah 17 dan 18 activity.
--
-- Bind (urut kemunculan): mode kurir ('NORMAL' | 'ONLINE') x2 · kata kunci x3 · lini bisnis x5
SELECT A.PYID                                            AS CASE_ID,
       A.PZINSKEY                                        AS REFERENCE,
       A.POLICYNO                                        AS POLICY_NUMBER,
       A.QQNAME                                          AS INSURED_NAME,
       A.BUSINESSNAME                                    AS BUSINESS_NAME,
       A.SOBNAME                                         AS BUSINESS_SOURCE,
       A.BRANCHNAME                                      AS BRANCH_NAME,
       (SELECT br.BRANCHNAME FROM POOLDATA.BRANCH br
         WHERE br.ID = A.KODECABANG_1)                   AS CLAIM_BRANCH,
       A.PXCREATEOPERATOR                                AS CREATOR,
       A.DATEOFLOSS_1                                    AS LOSS_DATE,
       A.REGISTERDATE_1                                  AS REPORT_DATE,
       A.PXCREATEDATETIME                                AS INPUT_DATE,
       CAST(NULL AS DATE)                                AS LOD_DATE,
       A.NOTREGISTNOTE_1                                 AS NOTE,
       CAST(NULL AS VARCHAR2(40))                        AS CLAIM_POSITION,
       CAST(NULL AS VARCHAR2(400))                       AS CLAIM_STATUS,
       CAST(NULL AS VARCHAR2(40))                        AS LOD_STATUS,
       CAST(NULL AS DATE)                                AS REQUEST_DATE,
       CAST(NULL AS VARCHAR2(200))                       AS POLICY_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS SURVEY_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(200))                       AS SURVEYOR,
       CAST(NULL AS VARCHAR2(100))                       AS SURVEY_NUMBER,
       CAST(NULL AS DATE)                                AS INBOX_DATE,
       CAST(NULL AS VARCHAR2(2000))                      AS ANALYST_NOTE,
       CAST(NULL AS VARCHAR2(40))                        AS RCL_PUCL_STATUS,
       CAST(NULL AS DATE)                                AS LETTER_PRINT_DATE,
       CAST(NULL AS VARCHAR2(100))                       AS CLAIM_AGE,
       CAST(NULL AS VARCHAR2(100))                       AS EXPIRY_STATUS
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
       INNER JOIN POOLDATA.BUSINESS c
              ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
              ON c.BUSINESSGROUPID = d.ID
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-ReceiveDocument'
   AND A.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND A.PNCCASEID IS NULL
   AND A.STATUSLOCK_1 = '1'
   AND (A.BRANCHNAME <> 'ASNET' OR A.BRANCHNAME IS NULL)
   AND ((:1 = 'NORMAL' AND (A.KURIR IS NULL OR A.KURIR <> 'Auto Service'))
        OR (:2 = 'ONLINE' AND A.KURIR = 'Auto Service'))
   AND (:3 IS NULL
        OR A.PYID LIKE '%' || :4 || '%' ESCAPE '\'
        OR A.POLICYNO LIKE '%' || :5 || '%' ESCAPE '\')
   AND (:6 = 'ALL'
        OR (:7 = 'NONMBU'
            AND A.GROUPPANEL_1 IN ('003', '004', '006', '009')
            AND c.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023'))
        OR (:8 = 'BONDING'
            AND c.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023'))
        OR (:9 = 'PA' AND A.GROUPPANEL_1 = '002')
        OR (:10 = 'TRAVEL' AND A.GROUPPANEL_1 = '005'))
   -- Batas cabang petugas (langkah 8). NULL = tidak dibatasi; lihat inboxadmin.ResolveScope.
   AND (:11 IS NULL OR A.KODECABANG_1 = :12)
   -- Kanwil pilihan manajer (langkah 2). NULL = tidak dibatasi.
   AND (:13 IS NULL
        OR A.KODECABANG_1 IN (SELECT br.ID FROM POOLDATA.BRANCH br
                   WHERE br.BASTERRITORY = :14))
 ORDER BY A.PXCREATEDATETIME DESC

-- name: list_request_survey
-- Tab Request Survey (`CityID = '9'`) — RDB List/BrowseRequestSurvey-SQL.xml
--
-- Saringan `NOT EXISTS` atas T_CLAIM_ADJUSTMENT dibawa apa adanya: permintaan survei yang
-- sudah punya adjustment bertipe selain salvage tidak lagi menunggu dikerjakan.
--
-- Bind (urut kemunculan): login pemanggil · kata kunci (boleh NULL) x3
SELECT A.PYID                                            AS CASE_ID,
       s.CLAIMID                                         AS REFERENCE,
       A.POLICYNO                                        AS POLICY_NUMBER,
       CAST(NULL AS VARCHAR2(400))                       AS INSURED_NAME,
       CAST(NULL AS VARCHAR2(200))                       AS BUSINESS_NAME,
       CAST(NULL AS VARCHAR2(200))                       AS BUSINESS_SOURCE,
       CAST(NULL AS VARCHAR2(200))                       AS BRANCH_NAME,
       CAST(NULL AS VARCHAR2(200))                       AS CLAIM_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS CREATOR,
       CAST(NULL AS DATE)                                AS LOSS_DATE,
       CAST(NULL AS DATE)                                AS REPORT_DATE,
       CAST(NULL AS DATE)                                AS INPUT_DATE,
       CAST(NULL AS DATE)                                AS LOD_DATE,
       CAST(NULL AS VARCHAR2(400))                       AS NOTE,
       CAST(NULL AS VARCHAR2(40))                        AS CLAIM_POSITION,
       CAST(NULL AS VARCHAR2(400))                       AS CLAIM_STATUS,
       CAST(NULL AS VARCHAR2(40))                        AS LOD_STATUS,
       s.INPUTDATE                                       AS REQUEST_DATE,
       A.BRANCHNAME                                      AS POLICY_BRANCH,
       s.BRANCH                                          AS SURVEY_BRANCH,
       A.USERTEKNIS_1                                    AS TECHNICAL_PIC,
       s.SURVEYOR                                        AS SURVEYOR,
       SUBSTR(s.SURVEYID, 20, 30)                        AS SURVEY_NUMBER,
       CAST(NULL AS DATE)                                AS INBOX_DATE,
       CAST(NULL AS VARCHAR2(2000))                      AS ANALYST_NOTE,
       CAST(NULL AS VARCHAR2(40))                        AS RCL_PUCL_STATUS,
       CAST(NULL AS DATE)                                AS LETTER_PRINT_DATE,
       CAST(NULL AS VARCHAR2(100))                       AS CLAIM_AGE,
       CAST(NULL AS VARCHAR2(100))                       AS EXPIRY_STATUS
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
       INNER JOIN POOLDATA.T_REQ_SURVEY s
              ON A.PZINSKEY = s.CLAIMID
 WHERE A.REQUESTSURVEY_1 = '1'
   AND A.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND A.PXASSIGNEDOPERATORID = :1
   AND NOT EXISTS (SELECT 1
                     FROM POOLDATA.T_CLAIM_ADJUSTMENT adj
                    WHERE adj.CLAIMID = A.PZINSKEY
                      AND adj.PAYMENTTYPE <> '3')
   AND (:2 IS NULL
        OR A.PYID LIKE '%' || :3 || '%' ESCAPE '\'
        OR A.POLICYNO LIKE '%' || :4 || '%' ESCAPE '\')
   -- Batas cabang petugas (langkah 8). NULL = tidak dibatasi; lihat inboxadmin.ResolveScope.
   AND (:5 IS NULL OR A.KODECABANG_1 = :6)
 ORDER BY s.INPUTDATE DESC

-- name: list_request_document
-- Tab Request Dokumen (`CityID = '10'`) — RDB List/GetRequestDokumenKomunikasi-SQL.xml
--
-- Antrean permintaan dokumen yang DIKIRIM pemanggil dan belum dijawab penerimanya
-- (`KOMUNIKASISTATUS = '0'`).
--
-- Kueri lama tidak mengurutkan hasilnya sama sekali; urutan di sini ditambahkan — lihat
-- catatan nomor 4 di kepala berkas.
--
-- Bind: :1 login pemanggil
SELECT p.CLAIMNO                                         AS CASE_ID,
       A.PZINSKEY                                        AS REFERENCE,
       p.NOPOLIS                                         AS POLICY_NUMBER,
       A.QQNAME                                          AS INSURED_NAME,
       p.BUSINESSNAME                                    AS BUSINESS_NAME,
       A.SOBNAME                                         AS BUSINESS_SOURCE,
       CAST(NULL AS VARCHAR2(200))                       AS BRANCH_NAME,
       CAST(NULL AS VARCHAR2(200))                       AS CLAIM_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS CREATOR,
       p.DATEOFLOSS                                      AS LOSS_DATE,
       p.REPORTDATE                                      AS REPORT_DATE,
       A.PXCREATEDATETIME                                AS INPUT_DATE,
       CAST(NULL AS DATE)                                AS LOD_DATE,
       CAST(NULL AS VARCHAR2(400))                       AS NOTE,
       CAST(NULL AS VARCHAR2(40))                        AS CLAIM_POSITION,
       CAST(NULL AS VARCHAR2(400))                       AS CLAIM_STATUS,
       CAST(NULL AS VARCHAR2(40))                        AS LOD_STATUS,
       CAST(NULL AS DATE)                                AS REQUEST_DATE,
       CAST(NULL AS VARCHAR2(200))                       AS POLICY_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS SURVEY_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(200))                       AS SURVEYOR,
       CAST(NULL AS VARCHAR2(100))                       AS SURVEY_NUMBER,
       CAST(NULL AS DATE)                                AS INBOX_DATE,
       CAST(NULL AS VARCHAR2(2000))                      AS ANALYST_NOTE,
       CAST(NULL AS VARCHAR2(40))                        AS RCL_PUCL_STATUS,
       CAST(NULL AS DATE)                                AS LETTER_PRINT_DATE,
       CAST(NULL AS VARCHAR2(100))                       AS CLAIM_AGE,
       CAST(NULL AS VARCHAR2(100))                       AS EXPIRY_STATUS
  FROM POOLDATA.T_CLAIM_PNC p
       INNER JOIN POOLDATA.T_CLAIMLIST_ADMIN A
              ON p.CLAIMNO = A.PYID
 WHERE UPPER(A.PXCREATEOPERATOR) = UPPER(:1)
   AND EXISTS (SELECT 1
                 FROM POOLDATA.M_KOMUNIKASI_PNC k
                WHERE k.COMMUNICATE_TO = A.PYID
                  AND k.KOMUNIKASISTATUS = '0')
 ORDER BY A.PXCREATEDATETIME DESC

-- name: list_all_case_admin
-- Tab All Case Admin (`CityID = '11'`) — RDB List/GetAllCaseAdmin-SQL.xml
--
-- Ini tab yang terbuka pertama kali. Ia menyaring `PXCREATEOPNAME` ke pemanggil, sehingga
-- layar terbuka pada klaim MILIK petugas itu sendiri.
--
-- Kueri lama tidak mengurutkan hasilnya sama sekali; urutan di sini ditambahkan.
--
-- Bind (urut kemunculan): login pemanggil · kata kunci (boleh NULL) x3
SELECT p.CLAIMNO                                         AS CASE_ID,
       A.PZINSKEY                                        AS REFERENCE,
       p.NOPOLIS                                         AS POLICY_NUMBER,
       A.QQNAME                                          AS INSURED_NAME,
       p.BUSINESSNAME                                    AS BUSINESS_NAME,
       A.SOBNAME                                         AS BUSINESS_SOURCE,
       CAST(NULL AS VARCHAR2(200))                       AS BRANCH_NAME,
       CAST(NULL AS VARCHAR2(200))                       AS CLAIM_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS CREATOR,
       p.DATEOFLOSS                                      AS LOSS_DATE,
       p.REPORTDATE                                      AS REPORT_DATE,
       A.PXCREATEDATETIME                                AS INPUT_DATE,
       CAST(NULL AS DATE)                                AS LOD_DATE,
       CAST(NULL AS VARCHAR2(400))                       AS NOTE,
       CAST(NULL AS VARCHAR2(40))                        AS CLAIM_POSITION,
       CAST(NULL AS VARCHAR2(400))                       AS CLAIM_STATUS,
       CAST(NULL AS VARCHAR2(40))                        AS LOD_STATUS,
       CAST(NULL AS DATE)                                AS REQUEST_DATE,
       CAST(NULL AS VARCHAR2(200))                       AS POLICY_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS SURVEY_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(200))                       AS SURVEYOR,
       CAST(NULL AS VARCHAR2(100))                       AS SURVEY_NUMBER,
       CAST(NULL AS DATE)                                AS INBOX_DATE,
       CAST(NULL AS VARCHAR2(2000))                      AS ANALYST_NOTE,
       CAST(NULL AS VARCHAR2(40))                        AS RCL_PUCL_STATUS,
       CAST(NULL AS DATE)                                AS LETTER_PRINT_DATE,
       CAST(NULL AS VARCHAR2(100))                       AS CLAIM_AGE,
       CAST(NULL AS VARCHAR2(100))                       AS EXPIRY_STATUS
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
       INNER JOIN POOLDATA.T_CLAIM_PNC p
              ON A.PZINSKEY = p.CLAIMID
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND UPPER(A.PXCREATEOPERATOR) = UPPER(:1)
   AND (:2 IS NULL
        OR A.PYID LIKE '%' || :3 || '%' ESCAPE '\'
        OR p.NOPOLIS LIKE '%' || :4 || '%' ESCAPE '\')
   -- Batas cabang petugas (langkah 8). NULL = tidak dibatasi; lihat inboxadmin.ResolveScope.
   AND (:5 IS NULL OR A.KODECABANG_1 = :6)
 ORDER BY A.PXCREATEDATETIME DESC

-- name: list_branch_claim
-- Tab Branch Claim (`CityID = '12'`) — RDB List/GetKlaimCabang-SQL.xml
--
-- Saringan `STATUSCLAIM_1 = '1150'` dibawa apa adanya: di sistem lama ia disisipkan lewat
-- `{ASIS:TempView.Location}` dari langkah 19 activity, dan `1150` adalah kode status
-- "LOD Report" pada V_STS_CLAIM (`R-06` tertutup, 33 kode `1134`–`1166`).
--
-- Kolom LOD_STATUS memuat cacat yang DIREPLIKASI: arti nilai `STATUSLOD` berbalik menurut
-- `FLAGBISNIS`. Pada lini OTO dan BFI nilai `1` berarti "Belum Upload", sedangkan pada lini
-- lain nilai `1` berarti "Sudah Upload" dan `0` berarti "Belum Upload". Satu nilai yang
-- sama karena itu berarti dua hal yang berlawanan, dan tidak ada nilai ketiga yang
-- membedakannya. Ia dipertahankan karena `P-5`, dan dicatat di sini supaya tidak
-- "diperbaiki" tanpa keputusan.
--
-- Bind (urut kemunculan): kata kunci (boleh NULL) x3 · lini bisnis x5
SELECT A.PYID                                            AS CASE_ID,
       A.PZINSKEY                                        AS REFERENCE,
       A.POLICYNO                                        AS POLICY_NUMBER,
       A.QQNAME                                          AS INSURED_NAME,
       A.BUSINESSNAME                                    AS BUSINESS_NAME,
       A.SOBNAME                                         AS BUSINESS_SOURCE,
       A.BRANCHNAME                                      AS BRANCH_NAME,
       (SELECT br.BRANCHNAME FROM POOLDATA.BRANCH br
         WHERE br.ID = A.KODECABANG_1)                   AS CLAIM_BRANCH,
       A.PXCREATEOPERATOR                                AS CREATOR,
       A.DATEOFLOSS_1                                    AS LOSS_DATE,
       A.REPORTDATE_1                                    AS REPORT_DATE,
       A.PXCREATEDATETIME                                AS INPUT_DATE,
       j.INSERTDATE                                      AS LOD_DATE,
       CAST(NULL AS VARCHAR2(400))                       AS NOTE,
       CASE A.PYSTATUSWORK
            WHEN 'New'                THEN 'On Progress'
            WHEN 'Resolved-Rejected'  THEN 'Reject'
            WHEN 'Resolved-Completed' THEN 'Close'
       END                                               AS CLAIM_POSITION,
       CAST(NULL AS VARCHAR2(400))                       AS CLAIM_STATUS,
       CASE
            WHEN j.FLAGBISNIS IN ('OTO', 'BFI') THEN
                 CASE j.STATUSLOD
                      WHEN '1' THEN 'Belum Upload'
                      WHEN '2' THEN 'Sudah Upload'
                 END
            ELSE
                 CASE j.STATUSLOD
                      WHEN '0' THEN 'Belum Upload'
                      WHEN '1' THEN 'Sudah Upload'
                 END
       END                                               AS LOD_STATUS,
       CAST(NULL AS DATE)                                AS REQUEST_DATE,
       CAST(NULL AS VARCHAR2(200))                       AS POLICY_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS SURVEY_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(200))                       AS SURVEYOR,
       CAST(NULL AS VARCHAR2(100))                       AS SURVEY_NUMBER,
       CAST(NULL AS DATE)                                AS INBOX_DATE,
       CAST(NULL AS VARCHAR2(2000))                      AS ANALYST_NOTE,
       CAST(NULL AS VARCHAR2(40))                        AS RCL_PUCL_STATUS,
       CAST(NULL AS DATE)                                AS LETTER_PRINT_DATE,
       CAST(NULL AS VARCHAR2(100))                       AS CLAIM_AGE,
       CAST(NULL AS VARCHAR2(100))                       AS EXPIRY_STATUS
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
       INNER JOIN POOLDATA.BUSINESS c
              ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
              ON c.BUSINESSGROUPID = d.ID
       INNER JOIN POOLDATA.T_CLAIM_JOB_PERSONALACCIDENT j
              ON j.CLAIMID = A.PZINSKEY
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.PXFLOWNAME NOT LIKE '%FixCorrespondence%'
   AND A.PXTASKLABEL IN ('Input Register', 'Input Estimasi', 'Estimation')
   AND (A.BRANCHNAME <> 'ASNET' OR A.BRANCHNAME IS NULL)
   AND A.STATUSCLAIM_1 = '1150'
   AND (:1 IS NULL
        OR A.PYID LIKE '%' || :2 || '%' ESCAPE '\'
        OR A.POLICYNO LIKE '%' || :3 || '%' ESCAPE '\')
   AND (:4 = 'ALL'
        OR (:5 = 'NONMBU'
            AND A.GROUPPANEL_1 IN ('003', '004', '006', '009')
            AND c.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023'))
        OR (:6 = 'BONDING'
            AND c.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023'))
        OR (:7 = 'PA' AND A.GROUPPANEL_1 = '002')
        OR (:8 = 'TRAVEL' AND A.GROUPPANEL_1 = '005'))
   -- Batas cabang petugas (langkah 8). NULL = tidak dibatasi; lihat inboxadmin.ResolveScope.
   AND (:9 IS NULL OR A.KODECABANG_1 = :10)
   -- Kanwil pilihan manajer (langkah 2). NULL = tidak dibatasi.
   AND (:11 IS NULL
        OR A.KODECABANG_1 IN (SELECT br.ID FROM POOLDATA.BRANCH br
                   WHERE br.BASTERRITORY = :12))
 ORDER BY A.PYSTATUSWORK ASC, A.PXCREATEDATETIME DESC

-- name: list_rcl_pucl
-- Tab Status RCL/PUCL (`CityID = '13'`) — RDB List/GetReminderPUCL-SQL.xml
--
-- Kueri ini tidak menerima satu pun bind: ia tidak punya kotak cari, tidak punya penyaring
-- lini bisnis, dan tidak menyaring menurut pemanggil. Antreannya milik workbasket
-- `RCLPUCL`, bukan milik satu orang.
--
-- Perhatikan saringannya berbeda dari tab lain: ia hanya mengecualikan
-- `Resolved-Completed`, sehingga klaim ber-`Resolved-Rejected` TETAP muncul. Itu masuk
-- akal untuk antrean penolakan, dan dibawa apa adanya.
SELECT A.PYID                                            AS CASE_ID,
       A.PZINSKEY                                        AS REFERENCE,
       A.POLICYNO                                        AS POLICY_NUMBER,
       A.QQNAME                                          AS INSURED_NAME,
       CAST(NULL AS VARCHAR2(200))                       AS BUSINESS_NAME,
       CAST(NULL AS VARCHAR2(200))                       AS BUSINESS_SOURCE,
       CAST(NULL AS VARCHAR2(200))                       AS BRANCH_NAME,
       CAST(NULL AS VARCHAR2(200))                       AS CLAIM_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS CREATOR,
       CAST(NULL AS DATE)                                AS LOSS_DATE,
       CAST(NULL AS DATE)                                AS REPORT_DATE,
       CAST(NULL AS DATE)                                AS INPUT_DATE,
       CAST(NULL AS DATE)                                AS LOD_DATE,
       CAST(NULL AS VARCHAR2(400))                       AS NOTE,
       CAST(NULL AS VARCHAR2(40))                        AS CLAIM_POSITION,
       CAST(NULL AS VARCHAR2(400))                       AS CLAIM_STATUS,
       CAST(NULL AS VARCHAR2(40))                        AS LOD_STATUS,
       CAST(NULL AS DATE)                                AS REQUEST_DATE,
       CAST(NULL AS VARCHAR2(200))                       AS POLICY_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS SURVEY_BRANCH,
       CAST(NULL AS VARCHAR2(200))                       AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(200))                       AS SURVEYOR,
       CAST(NULL AS VARCHAR2(100))                       AS SURVEY_NUMBER,
       A.TANGGALKIRIMPUCL_1                              AS INBOX_DATE,
       A.KOMENTARANALISATOR_1                            AS ANALYST_NOTE,
       CASE A.RCL_PUCL_1
            WHEN '1' THEN 'RCL'
            WHEN '2' THEN 'PUCL'
       END                                               AS RCL_PUCL_STATUS,
       A.TANGGALCETAKDOKUMENPUCL_1                       AS LETTER_PRINT_DATE,
       A.LAMAKLAIM_1                                     AS CLAIM_AGE,
       A.STATUSKLAIM_1                                   AS EXPIRY_STATUS
  FROM POOLDATA.T_CLAIMLIST_ADMIN L
       -- Tujuh kolom PUCL (tanggal kirim/cetak, status, lama klaim, persetujuan, MSIG)
       -- TIDAK ada di T_CLAIMLIST_ADMIN, sehingga masih dibaca dari tabel kerja Pega.
       -- Penugasannya -- siapa dan di workbasket mana -- dibaca dari T_CLAIMLIST_ADMIN.
       INNER JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
              ON A.PZINSKEY = L.PZINSKEY
 WHERE L.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND L.PYSTATUSWORK <> 'Resolved-Completed'
   AND L.PXASSIGNEDOPERATORID = 'RCLPUCL'
   AND A.TANGGALCETAKDOKUMENPUCL_1 IS NOT NULL
   AND A.PUCLAPPROVE_1 <> '1'
   AND A.MSIG_1 IS NULL
 ORDER BY A.TANGGALKIRIMPUCL_1 DESC

-- ============================================================================
-- SUMBER ANTREAN: POOLDATA.T_CLAIMLIST_ADMIN -- keputusan Work Owner 2026-10-07
-- ============================================================================
--
-- Ketujuh kueri tab membaca POOLDATA.T_CLAIMLIST_ADMIN, bukan lagi
-- DATAPEGA.PC_ASM_FW_GCNMFW_WORK + PC_ASSIGN_WORKLIST. Tabel itu memuat kepala klaim
-- beserta penugasannya dalam satu baris (PXASSIGNEDOPERATORID, PXTASKLABEL, PXFLOWNAME),
-- untuk klaim Pega MAUPUN klaim PNCN -- registrasi Go menulisnya lewat inboxentry.sql.
-- Karena itu kueri PNCN terpisah (list_all_pncn, list_all_case_admin_pncn) dibuang:
-- mempertahankannya akan menampilkan klaim PNCN dua kali.
--
-- Akibat yang diterima secara sadar, terukur 2026-10-07: klaim yang belum ada di tabel
-- ini tidak tampil. Saat itu Outstanding turun dari 983 ke sekitar 520, Unregistered RCV
-- dari 22 ke 1, LOD dan PUCL menjadi 0, dan baris Pega terbaru di tabel ini dibuat
-- 2026-10-03.
--
-- "Pembuat" pada All Case Admin dan Request Dokumen dicocokkan lewat PXCREATEOPERATOR,
-- bukan PXCREATEOPNAME seperti SQL Pega: kolom kedua kosong pada baris PNCN, sehingga
-- klaim PNCN tidak akan pernah ditemukan pemiliknya (keputusan Work Owner 2026-10-07).
--
-- Satu pengecualian: tab PUCL masih membaca tujuh kolom PUCL dari tabel kerja Pega,
-- karena kolom itu tidak ada di T_CLAIMLIST_ADMIN.

-- ============================================================================
-- EKSPOR AUTO CLAIM -- tombol "Export Hasil Auto Claim" dan "Export Klaim Gagal"
-- ============================================================================
--
-- Keduanya membaca POOLDATA.TMP_BATCH_CLAIM_KREDIT, tabel hasil batch pembuatan klaim
-- kredit otomatis. Alias Pega (`NILAIKLAIM AS "BRANCHCODE"`, `ACCEPTNO AS "EDMNO"`, ...)
-- TIDAK dibawa; nama kolom dibaca apa adanya dan dipetakan ke judul CSV di Go.

-- name: auto_claim_results
-- Tombol "Export Hasil Auto Claim" -- RDB List/GetHasilAutoClaim-SQL.xml
--
-- `trunc(tglproses) = trunc(sysdate)` diganti rentang setengah-terbuka yang dihitung di
-- Go (inboxadmin.ProcessingDay): TRUNC dan SYSDATE tidak portabel, dan TRUNC pada kolom
-- mematikan indeksnya.
--
-- Bind (urut kemunculan): awal hari · akhir hari (eksklusif) · login pemanggil
SELECT AGENID, NOPOLIS, IDPEGA, ACCEPTNO, NILAIKLAIM, NOASURANSI, TMP_MESSAGE
  FROM POOLDATA.TMP_BATCH_CLAIM_KREDIT
 WHERE TGLPROSES >= :1
   AND TGLPROSES < :2
   AND TMP_MESSAGE = 'Sukses Klaim'
   AND IDPEGA IS NOT NULL
   AND USERINPUT = :3
 ORDER BY TGLPROSES, NOPOLIS

-- name: auto_claim_failures
-- Tombol "Export Klaim Gagal" -- RDB List/GetHasilAutoClaimGagal-SQL.xml
--
-- Kueri lama TIDAK menyaring tanggal maupun pengguna: ia mengekspor SELURUH baris gagal
-- yang pernah ada. Itu dipertahankan (`P-5`).
--
-- Nama mata uang diambil activity lama satu per baris lewat GetIDCurrencyByID
-- (`select currency from pooldata.currency where id = ...`). Di sini ia LEFT JOIN: hasilnya
-- sama, tanpa satu perjalanan ke basis data per baris. Bila kodenya tidak ada di master,
-- kodenya sendiri yang ditampilkan alih-alih sel kosong.
SELECT b.NOPOLIS, b.NOASURANSI, b.NILAIKLAIM, b.TYPEKLAIM,
       COALESCE(cur.CURRENCY, b.CURRENCY) AS CURRENCY_NAME,
       b.AGENID, b.TMP_MESSAGE
  FROM POOLDATA.TMP_BATCH_CLAIM_KREDIT b
       LEFT JOIN POOLDATA.CURRENCY cur
              ON cur.ID = b.CURRENCY
 WHERE b.TMP_MESSAGE <> 'Sukses Klaim'
   AND b.TMP_MESSAGE IS NOT NULL
 ORDER BY b.NOPOLIS

-- ============================================================================
-- BATAS DATA -- cabang petugas, access group, dan daftar kanwil
-- ============================================================================
--
-- Lihat inboxadmin/scope.go untuk aturannya.

-- name: branch_of_login
-- Cabang petugas -- RDB List/GetIDCabang-SQL.xml, bentuk yang sama dengan modul Inbox
-- Laporan Klaim (`inboxlaporanklaim/repo/sqlstore/branch.sql`).
--
-- Ia MASIH memakai dua DB Link sistem lama (`R-03`): API pengganti HRD belum ada, dan
-- objeknya sama dengan yang dibaca Pega. Ia berada di balik seam inboxadmin.ScopeRepo,
-- sehingga saat API-nya tiba yang berubah hanya kueri ini.
--
-- Kueri lama mengambil `pxResults(1)` tanpa urutan; di sini urutannya ditegaskan supaya
-- batas data seorang petugas tidak berpindah-pindah di antara dua permintaan.
--
-- Bind: login petugas
SELECT a.ID
  FROM POOLDATA.BRANCH a
  JOIN LST_USER_ASURANSI@asmd.sinarmas.co.id b ON b.CAB_ID = a.OLDID
  JOIN HRDASM.V_HRD_MST@asmd.sinarmas.co.id c ON c.NIK = b.NIK
 WHERE c.LOGIN_APLIKASI = :1
 ORDER BY a.ID
 FETCH NEXT 1 ROWS ONLY

-- name: groups_of_login
-- Access group petugas -- POOLDATA.M_LOGIN_GROUP_PNC, sumber peran yang sama dengan modul
-- menu dan Inbox Accept Open Protection.
--
-- Bind: login petugas
SELECT g.GROUP_ID
  FROM POOLDATA.M_LOGIN_GROUP_PNC g
 WHERE UPPER(TRIM(g.LOGIN_ID)) = UPPER(TRIM(:1))
 ORDER BY g.GROUP_ID

-- name: region_list
-- Isi dropdown "Pilih Kanwil" -- nilai BASTERRITORY yang benar-benar dipakai cabang,
-- sama dengan modul Inbox Laporan Klaim.
SELECT DISTINCT br.BASTERRITORY
  FROM POOLDATA.BRANCH br
 WHERE br.BASTERRITORY IS NOT NULL
 ORDER BY br.BASTERRITORY

-- name: check_table
-- Memeriksa tabel inti modul ini terbaca dari koneksi yang dipakai.
--
-- Ia dipanggil perintah `-periksa` saat aplikasi start, dan sengaja tidak menyentuh baris
-- mana pun: yang diperiksa adalah HAK BACA dan keberadaan tabelnya, bukan isinya.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIMLIST_ADMIN
 WHERE 1 = 0
