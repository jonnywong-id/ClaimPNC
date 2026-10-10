-- Kueri modul Inbox Investigator.
--
-- LIMA tabel, dan aplikasi ini TIDAK MENULIS satu pun:
--
--   DATAPEGA.PC_ASM_FW_GCNMFW_WORK   header pekerjaan  — milik engine Pega
--   DATAPEGA.PC_ASSIGN_WORKBASKET    antrean bersama   — milik engine Pega
--   POOLDATA.T_CLAIM_OBJECTLIST      objek pertanggungan
--   POOLDATA.JSON_KLAIM              dokumen klaim utuh — sumber tanggal survei & investigasi
--   POOLDATA.T_CLAIM_PNC             header klaim relasional — sumber tanggal transfer
--
-- Layar ini hanya membaca, sehingga `P-1` terpenuhi tanpa negosiasi kepemilikan: Pega tetap
-- satu-satunya yang menulis tabelnya sendiri selama masa paralel (`D-21`).
--
-- Empat aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (D-20).
--   4. Tanpa pemanggilan stored procedure (D-02).
--
--
-- ============================================================================
-- BENTUK GABUNGANNYA DIAMBIL DARI KUERI PEGA YANG SUDAH ADA
-- ============================================================================
--
-- Report Definition `InboxRegisterCompliance_RD` menyatakan penyaringnya dalam istilah
-- properti Pega:
--
--   A  .pyStatusWork                          !=  "Resolved-Completed"
--   B  newAssignPage.pxAssignedOperatorID      =  Param.Operator
--   logika: B AND A
--   gabungan: newAssignPage.pxRefObjectKey = .pzInsKey
--
-- Bentuk SQL-nya tidak perlu ditebak: `RDB List/CountKlaimPUCL-SQL.xml` dan
-- `RDB List/ReminderPUCL-SQL.xml` menuliskan gabungan yang SAMA PERSIS terhadap workbasket
-- lain (`RCLPUCL`), dan itulah yang ditiru di sini —
--
--   FROM DATAPEGA.pc_ASM_FW_GCNMFW_Work a
--   INNER JOIN DATAPEGA.pc_assign_workbasket b
--     ON ( b."PXREFOBJECTKEY" = a."PZINSKEY"
--      AND b."PXOBJCLASS" = 'Assign-WorkBasket'
--      AND a."PXOBJCLASS" = 'ASM-FW-GCNMFW-Work-PNC' )
--
-- Kedua syarat `PXOBJCLASS` bukan hiasan: satu tabel Pega memuat BANYAK kelas kasus
-- sekaligus. Tanpa keduanya, inbox investigator akan ikut memuat kasus Komite,
-- OpenProtection, ReceiveDocument, dan SurveyClaim yang kebetulan berada di antrean yang
-- sama.
--
--
-- ============================================================================
-- NAMA KOLOM FISIK, DAN DARI MANA DIBACANYA
-- ============================================================================
--
-- Pega meratakan properti tertanam menjadi kolom berakhiran `_1`. Pemetaannya dibaca dari
-- `RDB List/ReminderPUCL-SQL.xml`, SELECT terlengkap atas tabel ini di seluruh export:
--
--   Properti Pega                    Kolom fisik          Caption grid
--   -------------------------------- -------------------- ---------------------
--   .pzInsKey                        PZINSKEY             (tidak ditampilkan)
--   .pyID                            PYID                 Nomor Case
--   .Policy.PolicyNo                 POLICYNO             No Polis
--   .Policy.QQName                   QQNAME               Nama Tertanggung
--   .Policy.Quotation.BusinessName   BUSINESSNAME         Nama Bisnis
--   .Policy.Quotation.BranchName     BRANCHNAME           Nama Cabang
--   .pyOrigUserID                    PYORIGUSERID         Nama Admin
--   .pxCreateDateTime                PXCREATEDATETIME     Tanggal Pendaftaran
--   .pyStatusWork                    PYSTATUSWORK         (penyaring saja)
--
-- Perhatikan `QQNAME`: di `ReminderPUCL` ia dialiaskan menjadi `"CABANG"` sementara isinya
-- nama tertanggung. Itu persis utang yang `03-CURRENT-ARCHITECTURE.md` §4.2 catat — alias
-- dipaksa cocok dengan properti klipboard yang sudah ada. Alias itu TIDAK dibawa.
--
--
-- ============================================================================
-- DUA KOLOM YANG TIDAK ADA DI TABEL KERJA, DAN CARA MENGAMBILNYA
-- ============================================================================
--
-- `.ClaimData.ObjectList(1).ObjectName` dan `.ClaimData.SurveyResults(1).SurveyDate`
-- keduanya PAGE LIST di Pega — bukan satu nilai — sehingga tidak diratakan menjadi kolom
-- pada PC_ASM_FW_GCNMFW_WORK. Report Definition sendiri mengakuinya; `InboxRegisterCompliance_RD`
-- membawa peringatan `pxUnoptimizedForReporting` yang menyebut KEDUANYA satu per satu.
--
--   ObjectList   -> POOLDATA.T_CLAIM_OBJECTLIST (CLAIMID, OBJECTID, OBJECTNAME, ...)
--                   kolomnya dibaca dari INSERT pada Database/*.prc
--                   penghubung: CLAIMID = PZINSKEY  (RDB List/BroswseKlaimByObjectName)
--
--   SurveyResults-> POOLDATA.JSON_KLAIM.DATA_JSONBLOB, jalur
--                   `$.SurveyResults[0].SurveyDate`
--                   penghubung: IDPEGA = PZINSKEY
--
-- # Kenapa tanggal survei dibaca dari JSON, bukan dari T_SURVEYORLIST
--
-- Versi pertama modul ini membacanya dari `POOLDATA.T_SURVEYORLIST` lewat `PNCCASEID`. Itu
-- KELIRU, dan akibatnya terlihat langsung di layar: kolom "Lama Masuk Inbox" kosong
-- sementara Pega menampilkan isinya. Tabel itu memuat baris HASIL SURVEI milik kasus
-- `Work-SurveyClaim` — bukan properti klaim yang digambar grid.
--
-- Yang membuktikannya bukan pembacaan rule melainkan pengukuran terhadap basis data
-- pengembangan:
--
--   T_SURVEYORLIST ber-PNCCASEID = pekerjaan di antrean     0 dari 1
--   T_REQ_SURVEY   ber-CLAIMID   = pekerjaan di antrean     0 dari 1
--   PC_ASM_FW_GCNMFW_WORK.SURVEYDATE_1 terisi               0 dari 2.646 klaim
--   JSON_KLAIM  $.SurveyResults[0].SurveyDate terisi        1 dari 1   <- INI
--
-- Kolom `SURVEYDATE_1` memang ADA di tabel kerja, dan justru itu jebakannya: ia terbaca
-- seperti jawaban, tetapi tidak pernah diisi satu baris pun.
--
-- `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc` menutup rantainya — ia membaca dokumen yang
-- sama (`datajson_JSONKLAIM`) untuk mengisi tabel relasional, sehingga JSON_KLAIM adalah
-- SUMBER-nya, bukan salinan.
--
-- `JSON_VALUE` dipilih, bukan penguraian di Go: ia didukung Oracle 12c+ dan PostgreSQL 17+
-- (`D-24`), dan modul Inbox Klaim Treaty Non-Prop sudah memakai pola yang sama. Klausa
-- `FORMAT JSON` diperlukan karena kolomnya BLOB — kolom `DATA_JSON` yang bertipe CLOB
-- terisi hanya pada 721 dari 16.170 baris dan kosong pada pekerjaan yang sedang mengantre.
--
-- # Arti "(1)", dan kenapa urutannya DITETAPKAN di sini
--
-- `ObjectList(1)` dan `SurveyResults(1)` adalah elemen PERTAMA page list — bukan "salah
-- satu". Pada JSON, elemen pertama itu terbaca langsung sebagai `[0]`, sehingga urutannya
-- tidak perlu ditetapkan ulang. Pada tabel objek ia perlu: baris di sana tidak punya urutan
-- bawaan, sehingga subquery tanpa ORDER BY dapat mengembalikan objek yang berbeda pada dua
-- pemuatan daftar yang sama. Penentunya `T_CLAIM_OBJECTLIST ORDER BY OBJECTID`.
--
-- Satu klaim dapat punya LEBIH DARI SATU baris JSON_KLAIM — terukur: enam klaim punya
-- ganda, terbanyak lima. Karena itu ia dibaca lewat subquery berkorelasi ber-FETCH FIRST 1
-- ROW ONLY yang mengambil konversi TERBARU, bukan lewat JOIN. JOIN akan menggandakan baris
-- antrean, dan penggandaan itu baru terlihat sebagai pekerjaan yang muncul dua kali di
-- layar.
--
--
-- ============================================================================
-- URUTAN DAFTAR
-- ============================================================================
--
-- Report Definition mengurutkan dua kolom: `.pyID` lebih dulu, lalu `.pxCreateDateTime`
-- (terbaca dari pySortOrder, posisi 1 dan 2). KEDUANYA ber-`pySortType = DESC` —
-- pekerjaan terbaru di atas.
--
-- Itu ditiru apa adanya, ditambah PZINSKEY sebagai pemutus di ujung. Pemutus itu penambahan
-- yang diperlukan: tanpa kolom yang unik di akhir, dua pekerjaan ber-`pyID` sama — yang
-- mungkin terjadi karena tidak ada DDL yang membuktikan keunikannya (`R-08`) — dapat
-- bertukar tempat antar pemuatan, dan pada daftar yang dipaginasi di peramban itu membuat
-- satu baris tampak berpindah sendiri.
--
-- Arahnya BUKAN detail kerapian. Daftar dipotong pada MaxRows, sehingga arah urutan
-- menentukan baris MANA yang bertahan saat antreannya panjang.
-- ============================================================================
-- SUMBER BARU (2026-10-08): OBJEK KERJA PEGA TIDAK DIBACA LAGI
-- ============================================================================
--
-- `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` tidak dipakai lagi ("Perubahan nama tabel untuk
-- Inbox.xlsx"). Baris DIGERAKKAN ANTREAN `PC_ASSIGN_WORKBASKET`, lalu disambung LEFT JOIN:
--
--   T_CLAIM_PNC p         nomor klaim, polis, tertanggung, bisnis, cabang, status, tanggal
--   T_CLAIMLIST_ADMIN k   "Nama Admin" (`PYORIGUSERID`)
--
-- T_CLAIMLIST_ADMIN TIDAK dijadikan tabel utama karena ia hanya memuat klaim di antrean
-- Admin: diukur 2026-10-08, klaim di antrean InvestigatorPNC ada di T_CLAIM_PNC tetapi
-- **tidak satu pun** di T_CLAIMLIST_ADMIN. Menjadikannya tabel utama mengosongkan layar.
--
-- Pemetaan kolom Pega -> pengganti:
--
--   PZINSKEY          -> b.PXREFOBJECTKEY
--   PYID              -> b.PXREFOBJECTINSNAME (nomor case Pega persis), cadangan p.CLAIMNO.
--                        BUKAN CLAIMNO lebih dulu: diukur 2026-10-08, 15 klaim punya CLAIMNO
--                        yang berbeda dari nomor case-nya sendiri.
--   POLICYNO          -> p.NOPOLIS
--   PYORIGUSERID      -> k.PYORIGUSERID -> k.PXCREATEOPERATOR -> p.ADMINKLAIM
--   PXCREATEDATETIME  -> p.REGISTERDATE, cadangan saat penugasan dibuat
--   PYSTATUSWORK      -> p.STATUSWORK; NULL TIDAK dibuang — ia masih memegang tugas antrean
--
-- `ADMINKLAIM` adalah Operator ID pembuat klaim: sama dengan `PXCREATEOPERATOR` pada 504
-- dari 522 klaim yang ada di kedua tabel (diukur 2026-10-08).


-- name: investigator_inbox_list
--
-- Seluruh pekerjaan yang menunggu di workbasket Investigator.
--
-- Workbasket dikirim sebagai PARAMETER meski nilainya konstanta di dalam kode
-- (inboxinvestigator.Workbasket). Menuliskannya ke dalam teks SQL berarti merangkai nilai
-- ke dalam pernyataan — dilarang tanpa perkecualian (`08-TECHNICAL-STRATEGY.md` §4.3) —
-- dan larangan itu tidak mengenal pengecualian "nilainya toh dari kode sendiri": aturan
-- yang berlaku kadang-kadang bukan aturan.
--
-- FETCH FIRST :2 ROWS ONLY memotong pada MaxRows. Pemanggil meminta SATU baris lebih banyak
-- daripada yang akan dikirim, supaya keberadaan baris ke-(N+1) membuktikan hasilnya
-- terpotong — lihat catatan pada Repo.List. Itu yang membuat pemotongan di sini DINYATAKAN,
-- berbeda dari `pyMaxRecords = 500` sistem lama yang memotong dalam diam.
SELECT b.PXREFOBJECTKEY                                           AS REFERENCE,
       COALESCE(b.PXREFOBJECTINSNAME, p.CLAIMNO)                 AS CASE_NUMBER,
       p.NOPOLIS                                                 AS POLICY_NUMBER,
       p.QQNAME                                                  AS INSURED_NAME,
       (SELECT o.OBJECTNAME
          FROM POOLDATA.T_CLAIM_OBJECTLIST o
         WHERE o.CLAIMID = b.PXREFOBJECTKEY
         ORDER BY o.OBJECTID
         FETCH FIRST 1 ROW ONLY)                                 AS PARTICIPANT_NAME,
       p.BUSINESSNAME                                            AS BUSINESS_NAME,
       p.BRANCHNAME                                              AS BRANCH_NAME,
       COALESCE(k.PYORIGUSERID, k.PXCREATEOPERATOR, p.ADMINKLAIM) AS ADMIN_NAME,
       COALESCE(p.REGISTERDATE, b.PXCREATEDATETIME)              AS REGISTERED_AT,
       (SELECT s.SURVEYDATE
          FROM POOLDATA.T_SURVEYORLIST s
         WHERE s.PNCCASEID = b.PXREFOBJECTKEY
         ORDER BY s.INDEX_SURVEY
         FETCH FIRST 1 ROW ONLY)                                 AS SURVEY_DATE,
       -- Kode lini bisnis. TIDAK digambar sebagai kolom grid — layar lama pun tidak
       -- menggambarnya — tetapi tab Unggah Dokumen memakainya untuk menyembunyikan
       -- kategori yang tidak berlaku bagi lini itu, persis seperti di layar Registrasi.
       --
       -- Dibaca langsung dari `p` (POOLDATA.T_CLAIM_PNC) yang sudah di-LEFT JOIN di bawah,
       -- bukan dari GROUPPANEL_1 pada tabel kerja Pega — tabel itu tidak dibaca lagi.
       p.GROUPPANEL                                              AS BUSINESS_LINE
  FROM DATAPEGA.PC_ASSIGN_WORKBASKET b
  LEFT JOIN POOLDATA.T_CLAIM_PNC p
    ON p.CLAIMID = b.PXREFOBJECTKEY
  LEFT JOIN POOLDATA.T_CLAIMLIST_ADMIN k
    ON k.PZINSKEY = b.PXREFOBJECTKEY
   AND k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
 WHERE b.PXOBJCLASS = 'Assign-WorkBasket'
   AND b.PXREFOBJECTCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND (p.STATUSWORK IS NULL OR p.STATUSWORK <> 'Resolved-Completed')
   AND UPPER(TRIM(b.PXASSIGNEDOPERATORID)) = :1
 ORDER BY COALESCE(b.PXREFOBJECTINSNAME, p.CLAIMNO) DESC,
          COALESCE(p.REGISTERDATE, b.PXCREATEDATETIME) DESC,
          b.PXREFOBJECTKEY DESC
 FETCH FIRST :2 ROWS ONLY

-- name: investigator_inbox_search
--
-- Sama dengan investigator_inbox_list, ditambah penyaring kata kunci.
--
-- Kueri TERSENDIRI, bukan satu kueri yang klausanya ditempel saat kata kuncinya ada. Kueri
-- yang berubah bentuk menurut masukan adalah kueri yang tidak dapat dibaca utuh oleh siapa
-- pun, dan itu justru pola yang membuat `{ASIS:...}` warisan berbahaya.
--
-- TUJUH kolom dicari — ketujuh kolom yang benar-benar digambar sebagai teks di grid. Tanggal
-- Pendaftaran dan Tanggal Survey tidak ikut: keduanya tanggal, dan mencocokkannya sebagai
-- teks menuntut pemformatan di dalam SQL yang `D-20` larang.
--
-- Nama Peserta ikut dicari meski ia subquery. Ia salah satu kolom yang paling dihafal
-- petugas investigasi, dan menghilangkannya dari pencarian membuat pengguna menyimpulkan
-- barisnya tidak ada.
--
-- Kata kuncinya sudah dibungkus tanda persen oleh pemanggil, bukan di sini: menempelkannya
-- di dalam teks SQL berarti merangkai nilai ke dalam pernyataan.
--
-- ENAM parameter berbeda untuk nilai yang sama, bukan satu yang dipakai ulang. Oracle
-- mengizinkan pemakaian ulang, tetapi tidak semua driver memetakan parameter bernomor ke
-- posisi argumen dengan cara yang sama — dan `D-20` menuntut kueri ini berjalan sama di
-- kedua basis data.
--
-- ESCAPE '\' disebut eksplisit karena Oracle tidak punya karakter pelolos bawaan pada LIKE.
-- Tanpa itu, pengguna yang mengetik "%" akan mencocokkan seluruh antrean tanpa satu pun
-- tanda bahwa yang dicari bukan yang diketik.
SELECT b.PXREFOBJECTKEY                                           AS REFERENCE,
       COALESCE(b.PXREFOBJECTINSNAME, p.CLAIMNO)                 AS CASE_NUMBER,
       p.NOPOLIS                                                 AS POLICY_NUMBER,
       p.QQNAME                                                  AS INSURED_NAME,
       (SELECT o.OBJECTNAME
          FROM POOLDATA.T_CLAIM_OBJECTLIST o
         WHERE o.CLAIMID = b.PXREFOBJECTKEY
         ORDER BY o.OBJECTID
         FETCH FIRST 1 ROW ONLY)                                 AS PARTICIPANT_NAME,
       p.BUSINESSNAME                                            AS BUSINESS_NAME,
       p.BRANCHNAME                                              AS BRANCH_NAME,
       COALESCE(k.PYORIGUSERID, k.PXCREATEOPERATOR, p.ADMINKLAIM) AS ADMIN_NAME,
       COALESCE(p.REGISTERDATE, b.PXCREATEDATETIME)              AS REGISTERED_AT,
       (SELECT s.SURVEYDATE
          FROM POOLDATA.T_SURVEYORLIST s
         WHERE s.PNCCASEID = b.PXREFOBJECTKEY
         ORDER BY s.INDEX_SURVEY
         FETCH FIRST 1 ROW ONLY)                                 AS SURVEY_DATE,
       -- Kode lini bisnis. TIDAK digambar sebagai kolom grid — layar lama pun tidak
       -- menggambarnya — tetapi tab Unggah Dokumen memakainya untuk menyembunyikan
       -- kategori yang tidak berlaku bagi lini itu, persis seperti di layar Registrasi.
       --
       -- Dibaca langsung dari `p` (POOLDATA.T_CLAIM_PNC) yang sudah di-LEFT JOIN di bawah,
       -- bukan dari GROUPPANEL_1 pada tabel kerja Pega — tabel itu tidak dibaca lagi.
       p.GROUPPANEL                                              AS BUSINESS_LINE
  FROM DATAPEGA.PC_ASSIGN_WORKBASKET b
  LEFT JOIN POOLDATA.T_CLAIM_PNC p
    ON p.CLAIMID = b.PXREFOBJECTKEY
  LEFT JOIN POOLDATA.T_CLAIMLIST_ADMIN k
    ON k.PZINSKEY = b.PXREFOBJECTKEY
   AND k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
 WHERE b.PXOBJCLASS = 'Assign-WorkBasket'
   AND b.PXREFOBJECTCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND (p.STATUSWORK IS NULL OR p.STATUSWORK <> 'Resolved-Completed')
   AND UPPER(TRIM(b.PXASSIGNEDOPERATORID)) = :1
   AND (UPPER(TRIM(COALESCE(b.PXREFOBJECTINSNAME, p.CLAIMNO))) LIKE :2 ESCAPE '\'
     OR UPPER(TRIM(p.NOPOLIS)) LIKE :3 ESCAPE '\'
     OR UPPER(TRIM(p.QQNAME)) LIKE :4 ESCAPE '\'
     OR UPPER(TRIM(p.BUSINESSNAME)) LIKE :5 ESCAPE '\'
     OR UPPER(TRIM(p.BRANCHNAME)) LIKE :6 ESCAPE '\'
     OR UPPER(TRIM(COALESCE(k.PYORIGUSERID, k.PXCREATEOPERATOR, p.ADMINKLAIM))) LIKE :7 ESCAPE '\'
     OR UPPER(TRIM((SELECT o.OBJECTNAME
                      FROM POOLDATA.T_CLAIM_OBJECTLIST o
                     WHERE o.CLAIMID = b.PXREFOBJECTKEY
                     ORDER BY o.OBJECTID
                     FETCH FIRST 1 ROW ONLY))) LIKE :8 ESCAPE '\')
 ORDER BY COALESCE(b.PXREFOBJECTINSNAME, p.CLAIMNO) DESC,
          COALESCE(p.REGISTERDATE, b.PXCREATEDATETIME) DESC,
          b.PXREFOBJECTKEY DESC
 FETCH FIRST :9 ROWS ONLY

-- name: investigator_inbox_check_table
--
-- Membuktikan keempat tabel beserta kolom yang dibaca modul ini ada dan dapat dibaca.
--
-- Dipakai `claimpnc -periksa`. FETCH FIRST 0 ROWS ONLY: yang diperiksa adalah apakah
-- pernyataannya dapat diurai dan dijalankan, bukan isinya — menarik satu baris berarti
-- membaca data nasabah tanpa keperluan.
--
-- Pemeriksaan ini berharga justru karena gabungannya menyentuh DUA skema sekaligus,
-- DATAPEGA dan POOLDATA. Hak baca yang kurang pada salah satunya baru terlihat saat
-- pengguna membuka layar — kecuali diperiksa lebih dulu di sini.
SELECT b.PXREFOBJECTKEY                                           AS REFERENCE,
       COALESCE(b.PXREFOBJECTINSNAME, p.CLAIMNO)                 AS CASE_NUMBER,
       p.NOPOLIS                                                 AS POLICY_NUMBER,
       p.QQNAME                                                  AS INSURED_NAME,
       (SELECT o.OBJECTNAME
          FROM POOLDATA.T_CLAIM_OBJECTLIST o
         WHERE o.CLAIMID = b.PXREFOBJECTKEY
         ORDER BY o.OBJECTID
         FETCH FIRST 1 ROW ONLY)                                 AS PARTICIPANT_NAME,
       p.BUSINESSNAME                                            AS BUSINESS_NAME,
       p.BRANCHNAME                                              AS BRANCH_NAME,
       COALESCE(k.PYORIGUSERID, k.PXCREATEOPERATOR, p.ADMINKLAIM) AS ADMIN_NAME,
       COALESCE(p.REGISTERDATE, b.PXCREATEDATETIME)              AS REGISTERED_AT,
       (SELECT s.SURVEYDATE
          FROM POOLDATA.T_SURVEYORLIST s
         WHERE s.PNCCASEID = b.PXREFOBJECTKEY
         ORDER BY s.INDEX_SURVEY
         FETCH FIRST 1 ROW ONLY)                                 AS SURVEY_DATE,
       -- Kode lini bisnis. TIDAK digambar sebagai kolom grid — layar lama pun tidak
       -- menggambarnya — tetapi tab Unggah Dokumen memakainya untuk menyembunyikan
       -- kategori yang tidak berlaku bagi lini itu, persis seperti di layar Registrasi.
       --
       -- Dibaca langsung dari `p` (POOLDATA.T_CLAIM_PNC) yang sudah di-LEFT JOIN di bawah,
       -- bukan dari GROUPPANEL_1 pada tabel kerja Pega — tabel itu tidak dibaca lagi.
       p.GROUPPANEL                                              AS BUSINESS_LINE
  FROM DATAPEGA.PC_ASSIGN_WORKBASKET b
  LEFT JOIN POOLDATA.T_CLAIM_PNC p
    ON p.CLAIMID = b.PXREFOBJECTKEY
  LEFT JOIN POOLDATA.T_CLAIMLIST_ADMIN k
    ON k.PZINSKEY = b.PXREFOBJECTKEY
   AND k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
 WHERE b.PXOBJCLASS = 'Assign-WorkBasket'
   AND b.PXREFOBJECTCLASS = 'ASM-FW-GCNMFW-Work-PNC'
 FETCH FIRST 0 ROWS ONLY

-- name: investigator_inbox_count_waiting
--
-- Cacah pekerjaan yang menunggu di workbasket Investigator.
--
-- Dipakai `claimpnc -periksa`. Angkanya menjawab pertanyaan yang tidak dapat dijawab layar
-- ketika hasilnya terpotong: BERAPA SEBENARNYA yang menunggu.
SELECT COUNT(*)
  FROM DATAPEGA.PC_ASSIGN_WORKBASKET b
  LEFT JOIN POOLDATA.T_CLAIM_PNC p
    ON p.CLAIMID = b.PXREFOBJECTKEY
  LEFT JOIN POOLDATA.T_CLAIMLIST_ADMIN k
    ON k.PZINSKEY = b.PXREFOBJECTKEY
   AND k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
 WHERE b.PXOBJCLASS = 'Assign-WorkBasket'
   AND b.PXREFOBJECTCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND (p.STATUSWORK IS NULL OR p.STATUSWORK <> 'Resolved-Completed')
   AND UPPER(TRIM(b.PXASSIGNEDOPERATORID)) = :1

-- name: investigator_inbox_count_without_survey
--
-- Cacah pekerjaan menunggu yang TIDAK punya satu pun baris survei.
--
-- # Kenapa ini layak diperiksa
--
-- Tanggal survei mengisi kolom KESEMBILAN grid, yang captionnya berbunyi "Lama Masuk Inbox"
-- meski isinya tanggal. Pekerjaan tanpa baris survei karena itu tampil dengan sel kosong di
-- kolom itu.
--
-- Bila cacahnya besar, kolom itu kosong bagi sebagian besar antrean, dan pilihan titik
-- awalnya perlu ditinjau ulang. Itu pertanyaan yang hanya dapat dijawab data produksi,
-- bukan export.
SELECT COUNT(*)
  FROM DATAPEGA.PC_ASSIGN_WORKBASKET b
  LEFT JOIN POOLDATA.T_CLAIM_PNC p
    ON p.CLAIMID = b.PXREFOBJECTKEY
  LEFT JOIN POOLDATA.T_CLAIMLIST_ADMIN k
    ON k.PZINSKEY = b.PXREFOBJECTKEY
   AND k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
 WHERE b.PXOBJCLASS = 'Assign-WorkBasket'
   AND b.PXREFOBJECTCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND (p.STATUSWORK IS NULL OR p.STATUSWORK <> 'Resolved-Completed')
   AND UPPER(TRIM(b.PXASSIGNEDOPERATORID)) = :1
   AND NOT EXISTS (SELECT 1
                     FROM POOLDATA.T_SURVEYORLIST s
                    WHERE s.PNCCASEID = b.PXREFOBJECTKEY
                      AND s.SURVEYDATE IS NOT NULL)

-- name: investigator_export
--
-- Baris berkas **Export Data Investigation**.
--
-- # Sumber barisnya BUKAN antrean, dan itu mengejutkan
--
-- Tombol ekspor berada di layar inbox, tetapi daftar yang diekspornya tidak datang dari
-- antrean workbasket. `Activity/ExportDataInvestigator-Act.xml` langkah 5 menjalankan
-- `RDB List/ExportDatainvestigator-SQL.xml`:
--
--   select CLAIMID, INVESTIGATOR_TF_DATE from POOLDATA.T_CLAIM_PNC
--    where trunc(INVESTIGATOR_TF_DATE) between {Dari} and {Sampai}
--
-- yaitu SELURUH klaim yang pernah dipindahkan ke investigator dalam rentang tanggal —
-- termasuk yang sudah selesai dan tidak lagi ada di antrean. Itulah gunanya kedua isian
-- "Dari" dan "Sampai" di kepala layar, dan itulah sebabnya keduanya BUKAN penyaring grid.
--
-- Langkah 2 activity itu memang menjalankan Report Definition berparameter
-- `Operator = "InvestigatorPNC"`, tetapi hasilnya tidak pernah dipakai menyusun berkas —
-- yang dipakai `TempDataExport`, halaman milik RDB-List di atas. Sisa langkah yang tidak
-- terpakai itu tidak dibawa.
--
-- # Ketiga belas kolomnya
--
-- Langkah 6 membuka tiap klaim dengan `Obj-Open-By-Handle` lalu menyalin dua belas properti
-- `ClaimData.SurveyResults(1).SurveyList(1).*`. Judul berkasnya ditulis sebagai satu teks
-- tetap pada `Activity/ExportDataInvestigator-Act.xml:2332`, dan itu disalin apa adanya ke
-- http/export.go.
--
-- Di sini kedua belas properti itu dibaca dari `POOLDATA.JSON_KLAIM.DATA_JSONBLOB` — jalur
-- yang sama, satu kueri, tanpa membuka ribuan objek satu per satu. Keberadaannya terukur:
-- `SurveyList` muncul pada 725 dokumen, `AlamatRSKlinik` pada 33, `NoRekapMedis` pada 19.
--
-- # Penyaring "Pilih Investigation"
--
-- Langkah 6.3 bersyarat
-- `SurveyList(1).IsInvestigated == TempInvestigateChose.IsInvestigated`, sehingga pilihan
-- dropdown WAJIB dan nilainya `'1'` atau `'0'` (terukur: 7 dokumen `'1'`, 2 dokumen `'0'`).
--
-- # Satu selisih yang BELUM dapat dipastikan, dan dinyatakan di sini
--
-- Syarat itu melekat pada langkah PENYALINAN, bukan pada pengambilan barisnya. Baris
-- berkasnya sudah terbentuk di langkah 5, sehingga ada kemungkinan Pega tetap menerbitkan
-- baris yang TIDAK cocok — dengan dua belas kolom investigasinya kosong, hanya tanggalnya
-- yang terisi.
--
-- Di sini baris yang tidak cocok **dibuang seluruhnya**. Itu yang dijanjikan dropdown
-- "Pilih Investigation" kepada pengguna, dan berkas yang sebagian besar barisnya kosong
-- tidak dapat dipakai untuk apa pun.
--
-- Yang tidak dapat dibuktikan dari export adalah arti angka `pyStepsPreCondParamsWhenFalse`
-- (`4`) pada langkah itu — apakah ia melewati langkah atau menghentikan putaran. Menebaknya
-- berarti mengarang. Bila Work Owner mendapati berkas lama memuat baris berkolom kosong,
-- perubahannya satu baris: penyaring `IsInvestigated` dipindahkan dari WHERE ke CASE pada
-- kedua belas kolomnya.
--
-- # Rentang tanggal memakai setengah terbuka, bukan TRUNC
--
-- Kueri lama memakai `trunc(kolom) BETWEEN …`. `TRUNC` tidak portabel, dan membungkus
-- kolom dengan fungsi membuang kemungkinan memakai index. Di sini batas atasnya dikirim
-- pemanggil sebagai hari berikutnya, sehingga perbandingannya `>= awal AND < sesudahAkhir`
-- — sama hasilnya, portabel, dan kolomnya tetap telanjang.
--
-- # Kenapa JSON_KLAIM dibaca lewat ROW_NUMBER, bukan JOIN biasa
--
-- Satu klaim dapat punya lebih dari satu baris JSON_KLAIM. JOIN biasa karena itu
-- menggandakan baris ekspor — satu klaim muncul dua sampai lima kali dengan isi yang
-- hampir sama, dan pembaca berkas tidak punya cara mengetahui mana yang berlaku. Subquery
-- ber-ROW_NUMBER memilih konversi TERBARU dan hanya itu. `ROW_NUMBER() OVER` sudah dipakai
-- 14 rule di sistem lama dan didukung kedua basis data sasaran.
SELECT c.INVESTIGATOR_TF_DATE                                    AS INVESTIGATED_AT,
       JSON_VALUE(j.DATA_JSONBLOB FORMAT JSON, '$.SurveyResults[0].SurveyList[0].AlamatRSKlinik')             AS HOSPITAL_ADDRESS,
       JSON_VALUE(j.DATA_JSONBLOB FORMAT JSON, '$.SurveyResults[0].SurveyList[0].CheckBoxAsuransiLain')       AS PAID_BY_OTHER_INSURER,
       JSON_VALUE(j.DATA_JSONBLOB FORMAT JSON, '$.SurveyResults[0].SurveyList[0].CheckBoxPasien')             AS PAID_BY_PATIENT,
       JSON_VALUE(j.DATA_JSONBLOB FORMAT JSON, '$.SurveyResults[0].SurveyList[0].CheckBoxPerusahaan')         AS PAID_BY_COMPANY,
       JSON_VALUE(j.DATA_JSONBLOB FORMAT JSON, '$.SurveyResults[0].SurveyList[0].CheckBoxTidakadapembayaran') AS NO_PAYMENT,
       JSON_VALUE(j.DATA_JSONBLOB FORMAT JSON, '$.SurveyResults[0].SurveyList[0].IsInvestigated')             AS INVESTIGATED,
       JSON_VALUE(j.DATA_JSONBLOB FORMAT JSON, '$.SurveyResults[0].SurveyList[0].KonfirmasiModelKwitansi')    AS RECEIPT_CONFIRMATION,
       JSON_VALUE(j.DATA_JSONBLOB FORMAT JSON, '$.SurveyResults[0].SurveyList[0].NoRekapMedis')               AS MEDICAL_RECORD_NUMBER,
       JSON_VALUE(j.DATA_JSONBLOB FORMAT JSON, '$.SurveyResults[0].SurveyList[0].NoTelpDiHubungi')            AS PHONE_CALLED,
       JSON_VALUE(j.DATA_JSONBLOB FORMAT JSON, '$.SurveyResults[0].SurveyList[0].PasienTerdaftar')            AS PATIENT_REGISTERED,
       JSON_VALUE(j.DATA_JSONBLOB FORMAT JSON, '$.SurveyResults[0].SurveyList[0].Remaks')                     AS REMARKS,
       JSON_VALUE(j.DATA_JSONBLOB FORMAT JSON, '$.SurveyResults[0].SurveyList[0].SelectRS')                   AS HOSPITAL_KIND
  FROM POOLDATA.T_CLAIM_PNC c
 INNER JOIN (SELECT k.IDPEGA,
                    k.DATA_JSONBLOB,
                    ROW_NUMBER() OVER (PARTITION BY k.IDPEGA
                                           ORDER BY k.TGL_INPUT DESC) AS ROW_RANK
               FROM POOLDATA.JSON_KLAIM k) j
    ON j.IDPEGA = c.CLAIMID
   AND j.ROW_RANK = 1
 WHERE c.INVESTIGATOR_TF_DATE >= :1
   AND c.INVESTIGATOR_TF_DATE < :2
   AND JSON_VALUE(j.DATA_JSONBLOB FORMAT JSON,
                  '$.SurveyResults[0].SurveyList[0].IsInvestigated') = :3
 ORDER BY c.INVESTIGATOR_TF_DATE DESC, c.CLAIMID DESC
 FETCH FIRST :4 ROWS ONLY

-- ============================================================================
-- FORMULIR KERJA INVESTIGATOR
-- ============================================================================
--
-- Ketiga kueri di bawah melayani Flow Action `InputInvestigator` — formulir yang dibuka
-- saat Nomor Case ditekan. Berbeda dari seluruh kueri di atasnya, DUA di antaranya
-- MENULIS.
--
-- Tabelnya `POOLDATA.TC_PNC_INVESTIGASI`, milik aplikasi ini sepenuhnya. DDL beserta
-- pemetaan ketiga puluh kolomnya ke properti Pega ada di
-- `docs/ddl/tc_pnc_investigasi.sql`.
--
-- `P-1` terpenuhi tanpa negosiasi: tabel ini tidak pernah ditulis Pega. Yang perlu
-- diperhatikan justru `T_CLAIM_PNC` pada kueri ketiga — tabel itu sudah ditulis aplikasi
-- ini lewat modul Registrasi, sehingga kepemilikannya pun sudah ada.

-- name: investigasi_ambil
--
-- Hasil investigasi yang sudah tersimpan untuk satu pekerjaan.
--
-- `DIHAPUS_PADA IS NULL` WAJIB ada: `D-66` menetapkan soft delete menyeluruh, dan satu
-- kueri yang lupa menyaringnya akan menampilkan isian yang seharusnya sudah hilang.
--
-- Diurutkan lalu dipotong satu baris. Hari ini satu klaim selalu punya paling banyak satu
-- baris investigasi, tetapi kuncinya mengizinkan lebih — dan kueri tanpa urutan yang
-- ditetapkan akan mengembalikan baris yang berbeda antar dua pemuatan formulir yang sama.
SELECT KLAIM_ID, URUTAN_SURVEI, URUTAN,
       TANGGAL_INVESTIGASI, IS_INVESTIGATED, SELECT_RS,
       RS_KLINIK_DISURVEI, RS_KLINIK_DISURVEI_LAIN, ALAMAT_RS_KLINIK,
       NO_REKAP_MEDIS, NAMA_PASIEN, TANGGAL_LAHIR,
       FLAG_DOB, KETERANGAN_DOB, PASIEN_TERDAFTAR, PT_REG,
       TANGGAL_PERAWATAN, TANGGAL_SELESAI_PERAWATAN,
       TOTAL_TAGIHAN, TAGIHAN_LUNAS,
       BAYAR_PASIEN, BAYAR_PERUSAHAAN, BAYAR_ASURANSI_LAIN, BAYAR_TIDAK_ADA,
       ASURANSI_LAIN, KONFIRMASI_KWITANSI,
       NAMA_PIC_RS, NAMA_PENELEPON, NAMA_KARYAWAN,
       KODE_AREA_TELP, NO_TELP_DIHUBUNGI, EKSTENSI,
       REMAKS
  FROM POOLDATA.TC_PNC_INVESTIGASI
 WHERE KLAIM_ID = :1
   AND DIHAPUS_PADA IS NULL
 ORDER BY URUTAN_SURVEI, URUTAN
 FETCH FIRST 1 ROW ONLY

-- name: investigasi_perbarui
--
-- Memperbarui formulir yang sudah ada.
--
-- # DUA pernyataan, bukan MERGE — dan itu bukan selera
--
-- `MERGE ... USING (SELECT ... FROM DUAL)` adalah bentuk Oracle; PostgreSQL tidak punya
-- `DUAL`, dan `D-20` menuntut satu set SQL yang berjalan sama di kedua basis data. Pola
-- "perbarui dulu, sisipkan bila tidak ada yang tersentuh" yang dipakai di sini sudah
-- dipakai modul Registrasi (`objek_perbarui` + `objek_sisip`), sehingga ia bukan hal baru
-- bagi pembaca berikutnya.
--
-- Pemanggil menjalankan keduanya DI DALAM SATU TRANSAKSI; lihat Repo.Save.
--
-- `DIHAPUS_PADA = NULL` ikut disetel: baris yang pernah ditandai terhapus lalu diisi ulang
-- lewat formulir kembali hidup, bukan tertinggal tak terlihat selamanya (`D-66`).
UPDATE POOLDATA.TC_PNC_INVESTIGASI
   SET TANGGAL_INVESTIGASI       = :1,
       IS_INVESTIGATED           = :2,
       SELECT_RS                 = :3,
       RS_KLINIK_DISURVEI        = :4,
       RS_KLINIK_DISURVEI_LAIN   = :5,
       ALAMAT_RS_KLINIK          = :6,
       NO_REKAP_MEDIS            = :7,
       NAMA_PASIEN               = :8,
       TANGGAL_LAHIR             = :9,
       FLAG_DOB                  = :10,
       KETERANGAN_DOB            = :11,
       PASIEN_TERDAFTAR          = :12,
       PT_REG                    = :13,
       TANGGAL_PERAWATAN         = :14,
       TANGGAL_SELESAI_PERAWATAN = :15,
       TOTAL_TAGIHAN             = :16,
       TAGIHAN_LUNAS             = :17,
       BAYAR_PASIEN              = :18,
       BAYAR_PERUSAHAAN          = :19,
       BAYAR_ASURANSI_LAIN       = :20,
       BAYAR_TIDAK_ADA           = :21,
       ASURANSI_LAIN             = :22,
       KONFIRMASI_KWITANSI       = :23,
       NAMA_PIC_RS               = :24,
       NAMA_PENELEPON            = :25,
       NAMA_KARYAWAN             = :26,
       KODE_AREA_TELP            = :27,
       NO_TELP_DIHUBUNGI         = :28,
       EKSTENSI                  = :29,
       REMAKS                    = :30,
       DIUBAH_OLEH               = :31,
       DIUBAH_PADA               = :32,
       DIHAPUS_PADA              = NULL
 WHERE KLAIM_ID = :33 AND URUTAN_SURVEI = :34 AND URUTAN = :35

-- name: investigasi_sisip
--
-- Menyisipkan formulir yang belum pernah ada.
--
-- Dijalankan HANYA bila investigasi_perbarui tidak menyentuh satu baris pun. `DIBUAT_*`
-- diisi di sini dan tidak pernah di-UPDATE, sehingga "siapa yang pertama kali
-- menginvestigasi" tidak tertimpa koreksi berikutnya.
INSERT INTO POOLDATA.TC_PNC_INVESTIGASI (
       TANGGAL_INVESTIGASI, IS_INVESTIGATED, SELECT_RS,
       RS_KLINIK_DISURVEI, RS_KLINIK_DISURVEI_LAIN, ALAMAT_RS_KLINIK,
       NO_REKAP_MEDIS, NAMA_PASIEN, TANGGAL_LAHIR,
       FLAG_DOB, KETERANGAN_DOB, PASIEN_TERDAFTAR, PT_REG,
       TANGGAL_PERAWATAN, TANGGAL_SELESAI_PERAWATAN,
       TOTAL_TAGIHAN, TAGIHAN_LUNAS,
       BAYAR_PASIEN, BAYAR_PERUSAHAAN, BAYAR_ASURANSI_LAIN, BAYAR_TIDAK_ADA,
       ASURANSI_LAIN, KONFIRMASI_KWITANSI,
       NAMA_PIC_RS, NAMA_PENELEPON, NAMA_KARYAWAN,
       KODE_AREA_TELP, NO_TELP_DIHUBUNGI, EKSTENSI,
       REMAKS, DIBUAT_OLEH, DIBUAT_PADA,
       KLAIM_ID, URUTAN_SURVEI, URUTAN)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10,
        :11, :12, :13, :14, :15, :16, :17, :18, :19, :20,
        :21, :22, :23, :24, :25, :26, :27, :28, :29, :30,
        :31, :32, :33, :34, :35)

-- name: investigasi_pindahkan_klaim
--
-- Memindahkan klaim dari Investigator ke Analyst.
--
-- Keempat kolomnya dibaca dari `Activity/SetStatusInvestigator_Act-Act.xml`:
--
--   ClaimData.StatusClaim         := '1151'        -> STATUSCLAIM
--   ClaimData.InvestTfDate        := waktu kini    -> INVESTIGATOR_TF_DATE
--   ClaimData.AnalystTransferDate := waktu kini    -> ANALYST_TRANSFERDATE
--   SurveyResults(1).SurveyStatus := 5             -> SURVEYSTATUS  (lihat catatan)
--
-- SATU waktu dipakai untuk kedua kolom tanggal, bukan dua pemanggilan `CURRENT_TIMESTAMP`.
-- Dua pemanggilan dapat berbeda sedetik, dan selisih itu akan terbaca sebagai klaim yang
-- berpindah ke Analyst sebelum investigasinya selesai.
--
-- `PNCStatus` TIDAK ditulis di sini: kolomnya tidak ada pada `T_CLAIM_PNC`. Ia dicatat
-- sebagai pertanyaan terbuka di `docs/ddl/tc_pnc_investigasi.sql` §3 — bukan dikarang
-- menjadi kolom lain yang kebetulan namanya mirip.
UPDATE POOLDATA.T_CLAIM_PNC
   SET STATUSCLAIM          = :1,
       INVESTIGATOR_TF_DATE = :2,
       ANALYST_TRANSFERDATE = :2
 WHERE CLAIMID = :3

-- name: investigasi_check_table
--
-- Membuktikan tabel hasil investigasi ada dan dapat dibaca.
--
-- Dipakai `claimpnc -periksa`. FETCH FIRST 0 ROWS ONLY: yang diperiksa adalah apakah
-- pernyataannya dapat diurai dan dijalankan, bukan isinya — menarik satu baris berarti
-- membaca data medis tanpa keperluan (`FR-R2`).
SELECT KLAIM_ID, URUTAN_SURVEI, URUTAN, TANGGAL_INVESTIGASI, IS_INVESTIGATED
  FROM POOLDATA.TC_PNC_INVESTIGASI
 FETCH FIRST 0 ROWS ONLY
