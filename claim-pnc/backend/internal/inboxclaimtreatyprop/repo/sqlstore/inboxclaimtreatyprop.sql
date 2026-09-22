-- Kueri modul Inbox Claim Treaty Prop: antrean klaim treaty proporsional.
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
-- PEMETAAN TIGA ARAH — alias grid Pega -> asal sebenarnya -> alias di sini
-- ============================================================================
--
-- Di layar ini seluruh kolom grid bernama `CARI` ditambah nomor urut, sehingga tidak satu
-- pun namanya menyatakan isinya. Tabel ini satu-satunya tempat ketiganya dapat dibandingkan
-- berdampingan.
--
-- Alias Pega  Asal sebenarnya                                        Alias di sini
-- ----------- ------------------------------------------------------ ------------------
-- CARI1       a.PXREFOBJECTKEY                                        WORK_KEY
-- CARI2       a.PXREFOBJECTINSNAME                                    CLAIM_ID
-- CARI3       a.PXASSIGNEDOPERATORID                                  ASSIGNED_OPERATOR
-- CARI4       a.PZINSKEY (kunci teknis Pega)                          REFERENCE
-- CARI5       b.NOPOLIS                                               POLICY_NUMBER
-- CARI6       JSON_VALUE(b.DATA_JSONBLOB,'$.QuotationData.BusinessName') BUSINESS_NAME
-- CARI7       JSON_VALUE(b.DATA_JSONBLOB,'$.QuotationData.SobName')   BUSINESS_SOURCE
-- CARI8       JSON_VALUE(b.DATA_JSONBLOB,'$.QuotationData.CedingCoName') CEDING_COMPANY
-- CARI9       JSON_VALUE(b.DATA_JSONBLOB,'$.InsuredName')             INSURED_NAME
-- CARI10 (!)  JSON_VALUE(b.DATA_JSONBLOB,'$.DateOfLoss')              LOSS_DATE
-- CARI13 (!)  JSON_VALUE(b.DATA_JSONBLOB,'$.DateOfLoss')              LOSS_DATE
-- CARI14      JSON_VALUE(b.DATA_JSONBLOB,'$.IsSubjectivity')          SUBJECTIVITY
-- CARI15      JSON_VALUE(b.DATA_JSONBLOB,'$.IDMaster')                MASTER_ID
--
-- Tanda (!) menandai CACAT YANG DIPERBAIKI, bukan sekadar nama yang menyesatkan.
-- Tanggal Kejadian dialiaskan `CARI10` oleh kedua kueri worklist tetapi `CARI13` oleh kueri
-- antrean teknik, sedangkan KETIGA grid di
-- `Section/InboxClaimTreaty_Section-Section.xml` terikat ke `.CARI10`. Akibatnya kolom
-- "Date Of Loss" pada grid antrean teknik SELALU KOSONG di sistem lama.
--
-- Di sini ketiga kueri mengaliaskannya ke satu nama yang sama, sehingga kolomnya terisi di
-- seluruh tab. Perbaikan ini disetujui Work Owner 2026-09-21 sebagai selisih terencana
-- `P-5`, dan dinyatakan ke pengguna lewat inboxclaimtreatyprop.PlannedDifferences.
--
-- ============================================================================
-- KE-13 ALIAS WAJIB SAMA DI SETIAP KUERI
-- ============================================================================
--
-- Urutan DAN namanya. Dua hal bergantung padanya:
--
--   * satu pemindai Go melayani ketiga kueri (scanWorkItem di inboxclaimtreatyprop.go);
--   * uji query_test.go menjaganya, dan ia gagal bila ada kueri yang aliasnya berbeda.
--
-- Kolom yang tidak berlaku bagi sebuah kueri bernilai NULL, bukan dihilangkan. Layar
-- menyembunyikannya mengikuti Tab.Columns — bukan menampilkan kolom kosong yang membuat
-- pengguna menduga datanya hilang. Yang begitu hanya SUBJECTIVITY, dan hanya pada kedua
-- kueri worklist.
--
-- ============================================================================
-- YANG BERUBAH DARI SISTEM LAMA, DAN KENAPA
-- ============================================================================
--
-- 1. PAGINASI DIKERJAKAN BASIS DATA.
--    Ketiga kueri lama menarik SELURUH baris tanpa batas apa pun — tidak ada `OFFSET`,
--    tidak ada `FETCH`, dan `pyMaxRecords` pada rule-nya kosong. Di sini halamannya
--    dipotong dengan `OFFSET … FETCH NEXT … ROWS ONLY`, yang didukung Oracle 12c+ dan
--    PostgreSQL (`09-DATABASE-STRATEGY.md` §3.3). Ini PERUBAHAN PERILAKU yang disadari,
--    bukan pemeliharaan.
--
-- 2. JUMLAH SELURUH BARIS DIHITUNG `COUNT(*) OVER ()`.
--    Satu perjalanan, bukan dua: kueri kedua yang hanya menghitung akan membaca ulang
--    seluruh gabungan JSON_KLAIM, dan gabungan itulah bagian yang mahal. Fungsi jendela
--    dihitung SEBELUM `OFFSET … FETCH` dipakai, sehingga angkanya jumlah seluruhnya —
--    bukan jumlah baris di halaman ini.
--
-- 3. `ORDER BY` DITAMBAHKAN PADA KUERI ANTREAN TEKNIK.
--    `GetClaimTreatyTeknik_SQL` dan `GetClaimTreatyAllAdmin_SQL` tidak mengurutkan
--    hasilnya sama sekali. Itu dapat dibiarkan selama seluruh baris ditarik sekaligus;
--    begitu halamannya dipotong, urutan yang tidak ditetapkan membuat satu baris muncul di
--    dua halaman sekaligus hilang dari halaman lain.
--
-- 4. NILAI SELALU LEWAT PARAMETER BINDING.
--    Kueri lama menyisipkan `{OperatorID.pyUserIdentifier}` langsung ke teks SQL. Larangan
--    perangkaian (`08-TECHNICAL-STRATEGY.md` §4.3) TIDAK ikut dikecualikan oleh keputusan
--    mana pun: yang direplikasi adalah perilaku bisnis, bukan celah injeksi.
--
-- ============================================================================
-- YANG SENGAJA TIDAK BERUBAH
-- ============================================================================
--
-- * `LEFT JOIN`, bukan `INNER JOIN`. Penugasan yang klaimnya belum punya baris di
--   JSON_KLAIM tetap muncul dengan kolom kosong — persis seperti sistem lama. Menggantinya
--   dengan INNER akan MENGHILANGKAN pekerjaan dari antrean tanpa satu pun pesan galat.
--
-- * Penyaring `PXREFOBJECTKEY LIKE '%CLMP%'`. Ia yang memisahkan objek kerja klaim treaty
--   dari objek kerja lain di tabel penugasan yang sama. Polanya literal, bukan masukan
--   pengguna, sehingga tidak butuh ESCAPE.
--
-- * Nama akun antrean teknik dikirim sebagai BIND, bukan ditulis di sini. Nilainya satu
--   tempat saja — inboxclaimtreatyprop.TechnicalWorkbasket — supaya SQL dan penyimpanan
--   memori tidak dapat berselisih tanpa ketahuan.
--
-- * `CAST(NULL AS VARCHAR2(…))` memakai tipe khas Oracle, mengikuti modul yang sudah ada
--   (riwayatklaim, inboxadmin). Ia satu-satunya bentuk tak portabel di berkas ini dan
--   tercatat sebagai utang teknis yang diselesaikan serentak untuk seluruh modul saat
--   perpindahan ke PostgreSQL (`09-DATABASE-STRATEGY.md` §10), bukan sepihak di sini.

-- name: list_worklist
-- Tab "Work List Treatyin Propotional" tanpa "See All Claim"
-- — RDB List/GetClaimTreaty_SQL-SQL.xml
--
-- Bind: :1 login pemanggil · :2 offset · :3 jumlah baris
SELECT a.PXREFOBJECTKEY                                          AS WORK_KEY,
       a.PZINSKEY                                                AS REFERENCE,
       a.PXREFOBJECTINSNAME                                      AS CLAIM_ID,
       a.PXASSIGNEDOPERATORID                                    AS ASSIGNED_OPERATOR,
       JSON_VALUE(b.DATA_JSONBLOB, '$.IDMaster')                 AS MASTER_ID,
       b.NOPOLIS                                                 AS POLICY_NUMBER,
       JSON_VALUE(b.DATA_JSONBLOB, '$.DateOfLoss')               AS LOSS_DATE,
       JSON_VALUE(b.DATA_JSONBLOB, '$.QuotationData.BusinessName') AS BUSINESS_NAME,
       JSON_VALUE(b.DATA_JSONBLOB, '$.QuotationData.SobName')    AS BUSINESS_SOURCE,
       JSON_VALUE(b.DATA_JSONBLOB, '$.QuotationData.CedingCoName') AS CEDING_COMPANY,
       JSON_VALUE(b.DATA_JSONBLOB, '$.InsuredName')              AS INSURED_NAME,
       CAST(NULL AS VARCHAR2(100))                               AS SUBJECTIVITY,
       COUNT(*) OVER ()                                          AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       LEFT JOIN POOLDATA.JSON_KLAIM b
              ON a.PXREFOBJECTKEY = b.IDPEGA
 WHERE a.PXREFOBJECTKEY LIKE '%CLMP%'
   AND a.PXASSIGNEDOPERATORID = :1
 ORDER BY a.PXCREATEDATETIME DESC, a.PXREFOBJECTINSNAME
OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY

-- name: list_worklist_all
-- Tab "Work List Treatyin Propotional" dengan "See All Claim" tercentang
-- — RDB List/GetClaimTreatyAllAdmin_SQL-SQL.xml
--
-- Bedanya dengan list_worklist HANYA hilangnya penyaring operator. Keduanya sengaja tidak
-- disatukan menjadi satu kueri ber-`:1 IS NULL`: penyaring yang dapat dimatikan membuat
-- rencana eksekusinya berubah-ubah, dan pada tabel penugasan yang besar perbedaannya nyata.
--
-- Bind: :1 offset · :2 jumlah baris
SELECT a.PXREFOBJECTKEY                                          AS WORK_KEY,
       a.PZINSKEY                                                AS REFERENCE,
       a.PXREFOBJECTINSNAME                                      AS CLAIM_ID,
       a.PXASSIGNEDOPERATORID                                    AS ASSIGNED_OPERATOR,
       JSON_VALUE(b.DATA_JSONBLOB, '$.IDMaster')                 AS MASTER_ID,
       b.NOPOLIS                                                 AS POLICY_NUMBER,
       JSON_VALUE(b.DATA_JSONBLOB, '$.DateOfLoss')               AS LOSS_DATE,
       JSON_VALUE(b.DATA_JSONBLOB, '$.QuotationData.BusinessName') AS BUSINESS_NAME,
       JSON_VALUE(b.DATA_JSONBLOB, '$.QuotationData.SobName')    AS BUSINESS_SOURCE,
       JSON_VALUE(b.DATA_JSONBLOB, '$.QuotationData.CedingCoName') AS CEDING_COMPANY,
       JSON_VALUE(b.DATA_JSONBLOB, '$.InsuredName')              AS INSURED_NAME,
       CAST(NULL AS VARCHAR2(100))                               AS SUBJECTIVITY,
       COUNT(*) OVER ()                                          AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       LEFT JOIN POOLDATA.JSON_KLAIM b
              ON a.PXREFOBJECTKEY = b.IDPEGA
 WHERE a.PXREFOBJECTKEY LIKE '%CLMP%'
 ORDER BY a.PXCREATEDATETIME DESC, a.PXREFOBJECTINSNAME
OFFSET :1 ROWS FETCH NEXT :2 ROWS ONLY

-- name: list_workbasket
-- Tab "Work Teknik Treatyin" — RDB List/GetClaimTreatyTeknik_SQL-SQL.xml
--
-- Ia membaca TABEL YANG BERBEDA: PC_ASSIGN_WORKBASKET, bukan PC_ASSIGN_WORKLIST. Itulah
-- yang membedakan antrean bersama dari penugasan perorangan (`D-26`), dan itu pula yang
-- membuat kueri ini tidak dapat disatukan dengan kedua kueri di atas.
--
-- Hanya kueri INI yang membawa Subjectivity.
--
-- Bind: :1 nama akun antrean teknik · :2 offset · :3 jumlah baris
SELECT a.PXREFOBJECTKEY                                          AS WORK_KEY,
       a.PZINSKEY                                                AS REFERENCE,
       a.PXREFOBJECTINSNAME                                      AS CLAIM_ID,
       a.PXASSIGNEDOPERATORID                                    AS ASSIGNED_OPERATOR,
       JSON_VALUE(b.DATA_JSONBLOB, '$.IDMaster')                 AS MASTER_ID,
       b.NOPOLIS                                                 AS POLICY_NUMBER,
       JSON_VALUE(b.DATA_JSONBLOB, '$.DateOfLoss')               AS LOSS_DATE,
       JSON_VALUE(b.DATA_JSONBLOB, '$.QuotationData.BusinessName') AS BUSINESS_NAME,
       JSON_VALUE(b.DATA_JSONBLOB, '$.QuotationData.SobName')    AS BUSINESS_SOURCE,
       JSON_VALUE(b.DATA_JSONBLOB, '$.QuotationData.CedingCoName') AS CEDING_COMPANY,
       JSON_VALUE(b.DATA_JSONBLOB, '$.InsuredName')              AS INSURED_NAME,
       JSON_VALUE(b.DATA_JSONBLOB, '$.IsSubjectivity')           AS SUBJECTIVITY,
       COUNT(*) OVER ()                                          AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKBASKET a
       LEFT JOIN POOLDATA.JSON_KLAIM b
              ON a.PXREFOBJECTKEY = b.IDPEGA
 WHERE a.PXREFOBJECTKEY LIKE '%CLMP%'
   AND a.PXASSIGNEDOPERATORID = :1
 ORDER BY a.PXCREATEDATETIME DESC, a.PXREFOBJECTINSNAME
OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY

-- name: check_worklist
-- Dipakai perintah `-periksa`: memastikan tabel penugasan perorangan DAN gabungannya ke
-- JSON_KLAIM terbaca dari koneksi yang dipakai.
--
-- Ia tidak menyentuh satu baris pun — yang diperiksa adalah hak baca dan keberadaan
-- tabelnya, bukan isinya. Gabungannya ikut diperiksa karena kegagalan yang paling mungkin
-- terjadi bukan "tabel tidak ada" melainkan "hak baca hanya diberikan pada salah satunya".
SELECT COUNT(*) AS PROBE
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       LEFT JOIN POOLDATA.JSON_KLAIM b
              ON a.PXREFOBJECTKEY = b.IDPEGA
 WHERE 1 = 0

-- name: check_workbasket
-- Dipakai perintah `-periksa`: memastikan tabel antrean bersama terbaca.
--
-- Ia terpisah dari check_worklist karena tabelnya memang berbeda, dan hak baca atas yang
-- satu tidak menyatakan apa pun tentang yang lain. Tab "Work Teknik Treatyin" bergantung
-- HANYA pada tabel ini.
SELECT COUNT(*) AS PROBE
  FROM DATAPEGA.PC_ASSIGN_WORKBASKET
 WHERE 1 = 0
