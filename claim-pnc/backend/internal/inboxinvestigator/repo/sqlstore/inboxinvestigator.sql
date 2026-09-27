-- Kueri modul Inbox Investigator.
--
-- EMPAT tabel, dan aplikasi ini TIDAK MENULIS satu pun:
--
--   DATAPEGA.PC_ASM_FW_GCNMFW_WORK   header pekerjaan  — milik engine Pega
--   DATAPEGA.PC_ASSIGN_WORKBASKET    antrean bersama   — milik engine Pega
--   POOLDATA.T_CLAIM_OBJECTLIST      objek pertanggungan
--   POOLDATA.T_SURVEYORLIST          hasil survei
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
-- pada PC_ASM_FW_GCNMFW_WORK. Report Definition membacanya dari BLOB; di sini keduanya
-- diambil dari tabel bisnisnya sendiri lewat subquery berkorelasi.
--
--   ObjectList   -> POOLDATA.T_CLAIM_OBJECTLIST (CLAIMID, OBJECTID, OBJECTNAME, ...)
--                   kolomnya dibaca dari INSERT pada Database/*.prc
--                   penghubung: CLAIMID = PZINSKEY  (RDB List/BroswseKlaimByObjectName)
--
--   SurveyResults-> POOLDATA.T_SURVEYORLIST (TGLINPUT, CASEID, PNCCASEID, SURVEYTYPE,
--                   LOSSTYPE, SURVEYOR_NAME, SURVEYDATE, LOCATION_SURVEY, OBJECT_NAME,
--                   LOCATION_OBJECT, IDOBJECT, INDEX_SURVEY, ...)
--                   kolomnya dibaca dari Database/INSERT_SURVEYORLIST.prc
--                   penghubung: PNCCASEID = PZINSKEY  (RDB List/BroswseKlaimByNoSurvey)
--
-- # Arti "(1)", dan kenapa urutannya DITETAPKAN di sini
--
-- `ObjectList(1)` dan `SurveyResults(1)` adalah elemen PERTAMA page list — bukan "salah
-- satu". Baris pada kedua tabel itu tidak punya urutan bawaan, sehingga subquery tanpa
-- ORDER BY dapat mengembalikan objek yang berbeda pada dua pemuatan daftar yang sama.
--
-- Yang dipakai sebagai penentu urutan adalah kolom indeks masing-masing tabel:
--
--   T_CLAIM_OBJECTLIST  ORDER BY OBJECTID
--   T_SURVEYORLIST      ORDER BY INDEX_SURVEY   <- kolom yang MEMANG menyimpan indeks page
--                                                  list; INSERT_SURVEYORLIST menerimanya
--                                                  sebagai parameter TSRVINDEX
--
-- Ini PENAMBAHAN terhadap sistem lama, dan ia perlu: tanpa urutan yang ditetapkan, kolom
-- Nama Peserta dapat berubah isinya antar dua penyegaran tanpa ada yang berubah di data.
--
--
-- ============================================================================
-- URUTAN DAFTAR
-- ============================================================================
--
-- Report Definition mengurutkan dua kolom: `.pyID` lebih dulu, lalu `.pxCreateDateTime`
-- (terbaca dari pySortOrder, posisi 1 dan 2). Arah tidak dinyatakan, sehingga menaik.
--
-- Itu ditiru apa adanya, ditambah PZINSKEY sebagai pemutus di ujung. Pemutus itu penambahan
-- yang diperlukan: tanpa kolom yang unik di akhir, dua pekerjaan ber-`pyID` sama — yang
-- mungkin terjadi karena tidak ada DDL yang membuktikan keunikannya (`R-08`) — dapat
-- bertukar tempat antar pemuatan, dan pada daftar yang dipaginasi di peramban itu membuat
-- satu baris tampak berpindah sendiri.


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
SELECT a.PZINSKEY                                                AS REFERENCE,
       a.PYID                                                    AS CASE_NUMBER,
       a.POLICYNO                                                AS POLICY_NUMBER,
       a.QQNAME                                                  AS INSURED_NAME,
       (SELECT o.OBJECTNAME
          FROM POOLDATA.T_CLAIM_OBJECTLIST o
         WHERE o.CLAIMID = a.PZINSKEY
         ORDER BY o.OBJECTID
         FETCH FIRST 1 ROW ONLY)                                 AS PARTICIPANT_NAME,
       a.BUSINESSNAME                                            AS BUSINESS_NAME,
       a.BRANCHNAME                                              AS BRANCH_NAME,
       a.PYORIGUSERID                                            AS ADMIN_NAME,
       a.PXCREATEDATETIME                                        AS REGISTERED_AT,
       (SELECT s.SURVEYDATE
          FROM POOLDATA.T_SURVEYORLIST s
         WHERE s.PNCCASEID = a.PZINSKEY
         ORDER BY s.INDEX_SURVEY
         FETCH FIRST 1 ROW ONLY)                                 AS SURVEY_DATE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
 INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET b
    ON b.PXREFOBJECTKEY = a.PZINSKEY
   AND b.PXOBJCLASS = 'Assign-WorkBasket'
   AND a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
 WHERE a.PYSTATUSWORK <> 'Resolved-Completed'
   AND UPPER(TRIM(b.PXASSIGNEDOPERATORID)) = :1
 ORDER BY a.PYID, a.PXCREATEDATETIME, a.PZINSKEY
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
SELECT a.PZINSKEY                                                AS REFERENCE,
       a.PYID                                                    AS CASE_NUMBER,
       a.POLICYNO                                                AS POLICY_NUMBER,
       a.QQNAME                                                  AS INSURED_NAME,
       (SELECT o.OBJECTNAME
          FROM POOLDATA.T_CLAIM_OBJECTLIST o
         WHERE o.CLAIMID = a.PZINSKEY
         ORDER BY o.OBJECTID
         FETCH FIRST 1 ROW ONLY)                                 AS PARTICIPANT_NAME,
       a.BUSINESSNAME                                            AS BUSINESS_NAME,
       a.BRANCHNAME                                              AS BRANCH_NAME,
       a.PYORIGUSERID                                            AS ADMIN_NAME,
       a.PXCREATEDATETIME                                        AS REGISTERED_AT,
       (SELECT s.SURVEYDATE
          FROM POOLDATA.T_SURVEYORLIST s
         WHERE s.PNCCASEID = a.PZINSKEY
         ORDER BY s.INDEX_SURVEY
         FETCH FIRST 1 ROW ONLY)                                 AS SURVEY_DATE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
 INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET b
    ON b.PXREFOBJECTKEY = a.PZINSKEY
   AND b.PXOBJCLASS = 'Assign-WorkBasket'
   AND a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
 WHERE a.PYSTATUSWORK <> 'Resolved-Completed'
   AND UPPER(TRIM(b.PXASSIGNEDOPERATORID)) = :1
   AND (UPPER(TRIM(a.PYID)) LIKE :2 ESCAPE '\'
     OR UPPER(TRIM(a.POLICYNO)) LIKE :3 ESCAPE '\'
     OR UPPER(TRIM(a.QQNAME)) LIKE :4 ESCAPE '\'
     OR UPPER(TRIM(a.BUSINESSNAME)) LIKE :5 ESCAPE '\'
     OR UPPER(TRIM(a.BRANCHNAME)) LIKE :6 ESCAPE '\'
     OR UPPER(TRIM(a.PYORIGUSERID)) LIKE :7 ESCAPE '\'
     OR UPPER(TRIM((SELECT o.OBJECTNAME
                      FROM POOLDATA.T_CLAIM_OBJECTLIST o
                     WHERE o.CLAIMID = a.PZINSKEY
                     ORDER BY o.OBJECTID
                     FETCH FIRST 1 ROW ONLY))) LIKE :8 ESCAPE '\')
 ORDER BY a.PYID, a.PXCREATEDATETIME, a.PZINSKEY
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
SELECT a.PZINSKEY                                                AS REFERENCE,
       a.PYID                                                    AS CASE_NUMBER,
       a.POLICYNO                                                AS POLICY_NUMBER,
       a.QQNAME                                                  AS INSURED_NAME,
       (SELECT o.OBJECTNAME
          FROM POOLDATA.T_CLAIM_OBJECTLIST o
         WHERE o.CLAIMID = a.PZINSKEY
         ORDER BY o.OBJECTID
         FETCH FIRST 1 ROW ONLY)                                 AS PARTICIPANT_NAME,
       a.BUSINESSNAME                                            AS BUSINESS_NAME,
       a.BRANCHNAME                                              AS BRANCH_NAME,
       a.PYORIGUSERID                                            AS ADMIN_NAME,
       a.PXCREATEDATETIME                                        AS REGISTERED_AT,
       (SELECT s.SURVEYDATE
          FROM POOLDATA.T_SURVEYORLIST s
         WHERE s.PNCCASEID = a.PZINSKEY
         ORDER BY s.INDEX_SURVEY
         FETCH FIRST 1 ROW ONLY)                                 AS SURVEY_DATE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
 INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET b
    ON b.PXREFOBJECTKEY = a.PZINSKEY
   AND b.PXOBJCLASS = 'Assign-WorkBasket'
   AND a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
 FETCH FIRST 0 ROWS ONLY

-- name: investigator_inbox_count_waiting
--
-- Cacah pekerjaan yang menunggu di workbasket Investigator.
--
-- Dipakai `claimpnc -periksa`. Angkanya menjawab pertanyaan yang tidak dapat dijawab layar
-- ketika hasilnya terpotong: BERAPA SEBENARNYA yang menunggu.
SELECT COUNT(*)
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
 INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET b
    ON b.PXREFOBJECTKEY = a.PZINSKEY
   AND b.PXOBJCLASS = 'Assign-WorkBasket'
   AND a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
 WHERE a.PYSTATUSWORK <> 'Resolved-Completed'
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
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
 INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET b
    ON b.PXREFOBJECTKEY = a.PZINSKEY
   AND b.PXOBJCLASS = 'Assign-WorkBasket'
   AND a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
 WHERE a.PYSTATUSWORK <> 'Resolved-Completed'
   AND UPPER(TRIM(b.PXASSIGNEDOPERATORID)) = :1
   AND NOT EXISTS (SELECT 1
                     FROM POOLDATA.T_SURVEYORLIST s
                    WHERE s.PNCCASEID = a.PZINSKEY
                      AND s.SURVEYDATE IS NOT NULL)
