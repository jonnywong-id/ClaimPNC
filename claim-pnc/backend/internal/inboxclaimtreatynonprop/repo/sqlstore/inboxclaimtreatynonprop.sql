-- Kueri modul Inbox Claim Treaty Non Prop: antrean klaim treaty non-proporsional.
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
-- TIGA TABEL, BUKAN DUA
-- ============================================================================
--
-- Layar Prop menggabungkan dua tabel. Layar ini menggabungkan TIGA, dan ketiganya
-- dibutuhkan:
--
--   DATAPEGA.PC_ASSIGN_WORKLIST / _WORKBASKET  a   baris penugasan — siapa memegang apa
--   DATAPEGA.PC_ASM_FW_GCNMFW_WORK             b   objek kerja klaim — kolom bisnisnya
--   POOLDATA.JSON_KLAIM                        c   nomor polis dan blob JSON klaim
--
-- Di layar Prop, kolom bisnis (nama tertanggung, Ceding Co, dan seterusnya) dibaca dari
-- blob JSON. Di sini kolom yang sama dibaca sebagai KOLOM TABEL pada `b`. Itu bukan pilihan
-- gaya: kueri lamanya memang begitu, dan kedua sumber itu dapat berbeda isinya.
--
-- ============================================================================
-- PEMETAAN TIGA ARAH — alias grid Pega -> asal sebenarnya -> alias di sini
-- ============================================================================
--
-- Di layar ini seluruh kolom grid bernama `CARI` ditambah nomor urut, sehingga tidak satu
-- pun namanya menyatakan isinya. Tabel ini satu-satunya tempat ketiganya dapat dibandingkan
-- berdampingan.
--
-- NOMORNYA BERBEDA DARI LAYAR PROP untuk arti yang sama. `CARI10` di sini kunci teknis,
-- sedangkan di layar Prop ia Tanggal Kejadian. Menyalin pemetaan layar Prop ke sini akan
-- menukar hampir seluruh kolom tanpa satu pun galat.
--
-- Alias Pega  Asal sebenarnya                              Alias di sini
-- ----------- -------------------------------------------- ----------------------
-- CARI10      a.PZINSKEY (kunci teknis Pega)               REFERENCE
-- CARI11      a.PXREFOBJECTINSNAME                         CLAIM_ID
-- (-)         a.PXASSIGNEDOPERATORID                       ASSIGNED_OPERATOR
-- CARI12      a.PXCREATEOPNAME                             CREATE_OPERATOR
-- CARI13      teks tetap 'Estimation' / 'Acceptation'      STATUS
-- CARI14      b.SOBNAME                                    BUSINESS_SOURCE
-- CARI15      b.BUSINESSNAME                               BUSINESS_NAME
-- CARI16      b.CEDINGCONAME                               CEDING_COMPANY
-- CARI17      b.INSUREDNAME                                INSURED_NAME
-- CARI18      c.NOPOLIS                                    POLICY_NUMBER
-- CARI19      b.MASTERID                                   MASTER_ID
-- CARI20      TRUNC(SYSDATE) - TRUNC(b.PXCREATEDATETIME)   AGING_DAYS  (lihat catatan 6)
-- CARI21      b.PXCREATEDATETIME (hanya kueri Teknik)      — hanya untuk ORDER BY
-- CARI22      c.DATA_JSON '$.DateOfLoss'                   LOSS_DATE
-- CARI23      c.DATA_JSON '$.IDMaster'                     JSON_MASTER_ID
-- CARI24      b.PXUPDATEOPERATOR                           LAST_UPDATE_OPERATOR
--
-- `ASSIGNED_OPERATOR` tidak punya pasangan alias Pega: kueri lama MENYARING menurut kolom
-- itu tetapi tidak pernah memilihnya. Ia dibawa di sini supaya penyimpanan memori dapat
-- meniru penyaring yang sama, dan tidak pernah digambar sebagai kolom.
--
-- ============================================================================
-- DUA KOLOM JSON YANG BERBEDA PADA SATU TABEL — JANGAN TERTUKAR
-- ============================================================================
--
-- `POOLDATA.JSON_KLAIM` punya DUA kolom JSON, dan kedua layar treaty membaca yang BERBEDA:
--
--   DATA_JSONBLOB   dibaca kueri layar Prop   (GetClaimTreaty_SQL:104-109)
--   DATA_JSON       dibaca kueri layar ini    (GetKlaimNonPropAdmin_SQL:43-44)
--
-- Keduanya TIDAK disamakan di sini. Apakah isinya sama tidak dapat diperiksa: DDL tabelnya
-- belum tersedia (`R-08`) dan isi kedua kolom belum pernah dilihat. Menukar salah satunya
-- ke yang lain adalah perubahan yang tidak menghasilkan galat apa pun bila salah — ia hanya
-- menampilkan tanggal dan ID master milik dokumen yang berbeda.
--
-- Yang DIUBAH hanyalah SINTAKSNYA, bukan kolomnya:
--
--   kueri lama   c.data_json.DateOfLoss                      notasi titik, khas Oracle
--   di sini      JSON_VALUE(c.DATA_JSON, '$.DateOfLoss')     SQL/JSON standar
--
-- Notasi titik menuntut kolomnya berconstraint `IS JSON` dan tidak ada padanannya di
-- PostgreSQL; `JSON_VALUE` didukung Oracle 12c+ dan PostgreSQL 17+ (`D-24`), sehingga
-- kueri ini tetap satu set untuk kedua basis data (`D-20`). Keduanya mengambil nilai skalar
-- dari dokumen dan jalur yang sama.
--
-- ============================================================================
-- KE-16 ALIAS WAJIB SAMA DI SETIAP KUERI
-- ============================================================================
--
-- Urutan DAN namanya. Dua hal bergantung padanya:
--
--   * satu pemindai Go melayani kelima kueri (scanWorkItem di
--     inboxclaimtreatynonprop.go);
--   * uji query_test.go menjaganya, dan ia gagal bila ada kueri yang aliasnya berbeda.
--
-- Kolom yang tidak berlaku bagi sebuah kueri bernilai NULL, bukan dihilangkan. Layar
-- menyembunyikannya mengikuti Tab.Columns — bukan menampilkan kolom kosong yang membuat
-- pengguna menduga datanya hilang. Yang begitu hanya JSON_MASTER_ID, dan hanya pada kueri
-- Teknik.
--
-- ============================================================================
-- LIMA KUERI UNTUK EMPAT DI SISTEM LAMA — SATU DI ANTARANYA BARU
-- ============================================================================
--
--   list_admin            GetKlaimNonPropAdmin_SQL       milik saya
--   list_admin_all        GetKlaimNonPropAdminALL_SQL    See All Claim
--   list_admin_all_tba    GetKlaimNonPropAdminTBA_SQL    See All + See TBA
--   list_admin_tba        (TIDAK ADA di sistem lama)     See TBA saja
--   list_technical        GetInboxListCNP_SQL            antrean teknik
--
-- `list_admin_tba` adalah SELISIH TERENCANA, bukan kueri yang terlewat dibaca. Di sistem
-- lama, "See TBA Claim" hanya berpengaruh bila "See All Claim" ikut dicentang: langkah 4
-- `Activity/GetDataTreatyinNonProp_Act-Act.xml` dijaga DUA prakondisi ber-AND —
-- `Inputdata.CARI10==""` (See All) DAN `Inputdata.CARI11==""` (See TBA). Mencentang TBA
-- sendirian tidak menjalankan kueri apa pun yang berbeda.
--
-- Keputusan Work Owner 2026-09-22 menjadikan keduanya penyaring yang saling bebas, dan
-- kueri ini yang melayani kombinasi yang dulu tidak terlayani. Ia dinyatakan ke pengguna
-- lewat inboxclaimtreatynonprop.PlannedDifferences.
--
-- ============================================================================
-- YANG BERUBAH DARI SISTEM LAMA, DAN KENAPA
-- ============================================================================
--
-- 1. PAGINASI DIKERJAKAN BASIS DATA.
--    Keempat kueri lama menarik SELURUH baris tanpa batas apa pun — tidak ada `OFFSET`,
--    tidak ada `FETCH`, dan `pyMaxRecords` pada rule-nya kosong. Di sini halamannya
--    dipotong dengan `OFFSET … FETCH NEXT … ROWS ONLY`, yang didukung Oracle 12c+ dan
--    PostgreSQL (`09-DATABASE-STRATEGY.md` §3.3). Ini PERUBAHAN PERILAKU yang disadari,
--    bukan pemeliharaan.
--
-- 2. JUMLAH SELURUH BARIS DIHITUNG `COUNT(*) OVER ()`.
--    Satu perjalanan, bukan dua: kueri kedua yang hanya menghitung akan membaca ulang
--    gabungan tiga tabel, dan gabungan itulah bagian yang mahal. Fungsi jendela dihitung
--    SEBELUM `OFFSET … FETCH` dipakai, sehingga angkanya jumlah seluruhnya — bukan jumlah
--    baris di halaman ini.
--
-- 3. `ORDER BY` DITAMBAHKAN PADA KETIGA KUERI ADMIN.
--    Ketiganya tidak mengurutkan hasilnya sama sekali. Itu dapat dibiarkan selama seluruh
--    baris ditarik sekaligus; begitu halamannya dipotong, urutan yang tidak ditetapkan
--    membuat satu baris muncul di dua halaman sekaligus hilang dari halaman lain.
--    Kueri Teknik SUDAH punya `ORDER BY CARI21` di sistem lama, dan urutan itu
--    dipertahankan apa adanya.
--
-- 4. GABUNGAN DITULIS SEBAGAI `JOIN`, BUKAN DAFTAR TABEL BERKOMA.
--    Kueri lama memakai gabungan gaya lama (`FROM a, b, c WHERE a.x = b.y AND …`), yang
--    membuat syarat gabungan dan syarat penyaring bercampur di satu klausa. Isinya tidak
--    berubah sedikit pun — yang berubah hanya tempat syaratnya tertulis.
--
-- 5. UMUR DIHITUNG DENGAN BENTUK YANG PORTABEL, ARTINYA TIDAK BERUBAH.
--    Kueri lama memakai `TRUNC(SYSDATE) - TRUNC(b.PXCREATEDATETIME)`. Keduanya ada di
--    daftar padanan wajib `09-DATABASE-STRATEGY.md` §4 — `SYSDATE` menjadi
--    `CURRENT_TIMESTAMP`, dan `TRUNC(tanggal)` menjadi `CAST(x AS DATE)` — sehingga
--    bentuknya di sini:
--
--      CAST(CURRENT_TIMESTAMP AS DATE) - CAST(b.PXCREATEDATETIME AS DATE)
--
--    Hasilnya SAMA PERSIS: pemangkasan ke tanggal tetap terjadi pada kedua sisi, dan
--    pengurangan dua tanggal menghasilkan bilangan hari penuh di Oracle maupun
--    PostgreSQL. Yang hilang hanya ketergantungan pada dua fungsi khas Oracle.
--
--    Pemangkasan kedua sisi itu bukan kerapian: tanpanya, selisih dihitung dari JAM,
--    sehingga pekerjaan yang dibuat kemarin sore terhitung nol hari sampai lewat 24 jam.
--
-- 6. NILAI SELALU LEWAT PARAMETER BINDING.
--    Kueri lama menyisipkan `{Inputdata.CARI10}` langsung ke teks SQL, dan
--    `Activity/GetWorkCNP_Act-Act.xml` bahkan merangkai potongan klausa `WHERE` dari
--    string. Larangan perangkaian (`08-TECHNICAL-STRATEGY.md` §4.3) TIDAK ikut
--    dikecualikan oleh keputusan mana pun: yang direplikasi adalah perilaku bisnis, bukan
--    celah injeksi.
--
-- ============================================================================
-- YANG SENGAJA TIDAK BERUBAH
-- ============================================================================
--
-- * `LEFT JOIN`, bukan `INNER JOIN`. Kueri lama memakai gabungan dalam, sehingga penugasan
--   yang klaimnya belum punya baris di JSON_KLAIM HILANG dari antrean. Di sini ia tetap
--   muncul dengan kolom kosong.
--
--   Ini satu-satunya tempat berkas ini melonggarkan gabungan, dan alasannya sempit: kueri
--   TBA justru mencari baris yang `NOPOLIS IS NULL`, yaitu klaim yang datanya belum
--   lengkap. Gabungan dalam pada tabel yang sama akan menyingkirkan sebagian dari yang
--   justru dicari. Baris yang sama sekali tidak punya pasangan di JSON_KLAIM karena itu
--   ikut terbaca sebagai TBA — dan itu memang benar: nomor polisnya memang belum ada.
--
-- * Penyaring `PXREFOBJECTINSNAME LIKE 'CLMNP-%'` dengan jangkar di depan. Awalan `CLMP`
--   milik layar Prop adalah awalan dari `CLMNP-` pada tiga huruf pertamanya, sehingga
--   penyaring longgar `'%CLMP%'` TIDAK menangkap baris non-prop, tetapi penyaring longgar
--   `'%CLMNP%'` akan menangkap baris yang nomornya kebetulan memuatnya di tengah. Jangkar
--   depan dipertahankan persis seperti aslinya. Polanya literal, bukan masukan pengguna,
--   sehingga tidak butuh ESCAPE.
--
-- * Teks status ditulis LITERAL, sama seperti kueri lama. Ia tidak dibuat bind supaya
--   bentuk kuerinya tetap dapat dibaca DBA apa adanya; kesesuaiannya dengan konstanta
--   `StatusEstimation` dan `StatusAcceptation` dijaga query_test.go.
--
-- * Nama akun antrean teknik dikirim sebagai BIND, bukan ditulis di sini. Nilainya satu
--   tempat saja — inboxclaimtreatynonprop.TechnicalWorkbasket — supaya SQL dan penyimpanan
--   memori tidak dapat berselisih tanpa ketahuan.
--
-- * `CAST(NULL AS VARCHAR2(…))` memakai tipe khas Oracle, mengikuti modul yang sudah ada
--   (riwayatklaim, inboxadmin, inboxclaimtreatyprop). Ia satu-satunya bentuk tak portabel
--   di berkas ini dan tercatat sebagai utang teknis yang diselesaikan serentak untuk
--   seluruh modul saat perpindahan ke PostgreSQL (`09-DATABASE-STRATEGY.md` §10), bukan
--   sepihak di sini.

-- name: list_admin
-- Tab Admin tanpa checkbox apa pun — antrean milik pemanggil.
-- — RDB List/GetKlaimNonPropAdmin_SQL-SQL.xml
--
-- Bind: :1 login pemanggil · :2 offset · :3 jumlah baris
SELECT a.PZINSKEY                                    AS REFERENCE,
       a.PXREFOBJECTINSNAME                          AS CLAIM_ID,
       a.PXASSIGNEDOPERATORID                        AS ASSIGNED_OPERATOR,
       b.MASTERID                                    AS MASTER_ID,
       JSON_VALUE(c.DATA_JSON, '$.IDMaster')         AS JSON_MASTER_ID,
       c.NOPOLIS                                     AS POLICY_NUMBER,
       JSON_VALUE(c.DATA_JSON, '$.DateOfLoss')       AS LOSS_DATE,
       b.BUSINESSNAME                                AS BUSINESS_NAME,
       b.SOBNAME                                     AS BUSINESS_SOURCE,
       b.CEDINGCONAME                                AS CEDING_COMPANY,
       b.INSUREDNAME                                 AS INSURED_NAME,
       'Estimation'                                  AS STATUS,
       (CAST(CURRENT_TIMESTAMP AS DATE)
        - CAST(b.PXCREATEDATETIME AS DATE))          AS AGING_DAYS,
       a.PXCREATEOPNAME                              AS CREATE_OPERATOR,
       b.PXUPDATEOPERATOR                            AS LAST_UPDATE_OPERATOR,
       COUNT(*) OVER ()                              AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK b
              ON a.PXREFOBJECTKEY = b.PZINSKEY
       LEFT JOIN POOLDATA.JSON_KLAIM c
              ON a.PXREFOBJECTKEY = c.IDPEGA
 WHERE a.PXREFOBJECTINSNAME LIKE 'CLMNP-%'
   AND a.PXASSIGNEDOPERATORID = :1
 ORDER BY b.PXCREATEDATETIME DESC, a.PXREFOBJECTINSNAME
OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY

-- name: list_admin_all
-- Tab Admin dengan "See All Claim" tercentang — seluruh petugas.
-- — RDB List/GetKlaimNonPropAdminALL_SQL-SQL.xml
--
-- Bedanya dengan list_admin HANYA hilangnya penyaring operator. Keduanya sengaja tidak
-- disatukan menjadi satu kueri ber-`:1 IS NULL`: penyaring yang dapat dimatikan membuat
-- rencana eksekusinya berubah-ubah, dan pada tabel penugasan yang besar perbedaannya nyata.
--
-- Bind: :1 offset · :2 jumlah baris
SELECT a.PZINSKEY                                    AS REFERENCE,
       a.PXREFOBJECTINSNAME                          AS CLAIM_ID,
       a.PXASSIGNEDOPERATORID                        AS ASSIGNED_OPERATOR,
       b.MASTERID                                    AS MASTER_ID,
       JSON_VALUE(c.DATA_JSON, '$.IDMaster')         AS JSON_MASTER_ID,
       c.NOPOLIS                                     AS POLICY_NUMBER,
       JSON_VALUE(c.DATA_JSON, '$.DateOfLoss')       AS LOSS_DATE,
       b.BUSINESSNAME                                AS BUSINESS_NAME,
       b.SOBNAME                                     AS BUSINESS_SOURCE,
       b.CEDINGCONAME                                AS CEDING_COMPANY,
       b.INSUREDNAME                                 AS INSURED_NAME,
       'Estimation'                                  AS STATUS,
       (CAST(CURRENT_TIMESTAMP AS DATE)
        - CAST(b.PXCREATEDATETIME AS DATE))          AS AGING_DAYS,
       a.PXCREATEOPNAME                              AS CREATE_OPERATOR,
       b.PXUPDATEOPERATOR                            AS LAST_UPDATE_OPERATOR,
       COUNT(*) OVER ()                              AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK b
              ON a.PXREFOBJECTKEY = b.PZINSKEY
       LEFT JOIN POOLDATA.JSON_KLAIM c
              ON a.PXREFOBJECTKEY = c.IDPEGA
 WHERE a.PXREFOBJECTINSNAME LIKE 'CLMNP-%'
 ORDER BY b.PXCREATEDATETIME DESC, a.PXREFOBJECTINSNAME
OFFSET :1 ROWS FETCH NEXT :2 ROWS ONLY

-- name: list_admin_tba
-- Tab Admin dengan "See TBA Claim" saja — antrean milik pemanggil yang polisnya belum
-- terbit.
--
-- KUERI INI TIDAK ADA DI SISTEM LAMA. Ia melayani kombinasi checkbox yang dulu tidak
-- terlayani sama sekali; lihat catatan "LIMA KUERI" di kepala berkas.
--
-- Bind: :1 login pemanggil · :2 offset · :3 jumlah baris
SELECT a.PZINSKEY                                    AS REFERENCE,
       a.PXREFOBJECTINSNAME                          AS CLAIM_ID,
       a.PXASSIGNEDOPERATORID                        AS ASSIGNED_OPERATOR,
       b.MASTERID                                    AS MASTER_ID,
       JSON_VALUE(c.DATA_JSON, '$.IDMaster')         AS JSON_MASTER_ID,
       c.NOPOLIS                                     AS POLICY_NUMBER,
       JSON_VALUE(c.DATA_JSON, '$.DateOfLoss')       AS LOSS_DATE,
       b.BUSINESSNAME                                AS BUSINESS_NAME,
       b.SOBNAME                                     AS BUSINESS_SOURCE,
       b.CEDINGCONAME                                AS CEDING_COMPANY,
       b.INSUREDNAME                                 AS INSURED_NAME,
       'Estimation'                                  AS STATUS,
       (CAST(CURRENT_TIMESTAMP AS DATE)
        - CAST(b.PXCREATEDATETIME AS DATE))          AS AGING_DAYS,
       a.PXCREATEOPNAME                              AS CREATE_OPERATOR,
       b.PXUPDATEOPERATOR                            AS LAST_UPDATE_OPERATOR,
       COUNT(*) OVER ()                              AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK b
              ON a.PXREFOBJECTKEY = b.PZINSKEY
       LEFT JOIN POOLDATA.JSON_KLAIM c
              ON a.PXREFOBJECTKEY = c.IDPEGA
 WHERE a.PXREFOBJECTINSNAME LIKE 'CLMNP-%'
   AND a.PXASSIGNEDOPERATORID = :1
   AND c.NOPOLIS IS NULL
 ORDER BY b.PXCREATEDATETIME DESC, a.PXREFOBJECTINSNAME
OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY

-- name: list_admin_all_tba
-- Tab Admin dengan KEDUA checkbox tercentang.
-- — RDB List/GetKlaimNonPropAdminTBA_SQL-SQL.xml
--
-- Inilah satu-satunya keadaan yang menjalankan kueri TBA di sistem lama.
--
-- Bind: :1 offset · :2 jumlah baris
SELECT a.PZINSKEY                                    AS REFERENCE,
       a.PXREFOBJECTINSNAME                          AS CLAIM_ID,
       a.PXASSIGNEDOPERATORID                        AS ASSIGNED_OPERATOR,
       b.MASTERID                                    AS MASTER_ID,
       JSON_VALUE(c.DATA_JSON, '$.IDMaster')         AS JSON_MASTER_ID,
       c.NOPOLIS                                     AS POLICY_NUMBER,
       JSON_VALUE(c.DATA_JSON, '$.DateOfLoss')       AS LOSS_DATE,
       b.BUSINESSNAME                                AS BUSINESS_NAME,
       b.SOBNAME                                     AS BUSINESS_SOURCE,
       b.CEDINGCONAME                                AS CEDING_COMPANY,
       b.INSUREDNAME                                 AS INSURED_NAME,
       'Estimation'                                  AS STATUS,
       (CAST(CURRENT_TIMESTAMP AS DATE)
        - CAST(b.PXCREATEDATETIME AS DATE))          AS AGING_DAYS,
       a.PXCREATEOPNAME                              AS CREATE_OPERATOR,
       b.PXUPDATEOPERATOR                            AS LAST_UPDATE_OPERATOR,
       COUNT(*) OVER ()                              AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK b
              ON a.PXREFOBJECTKEY = b.PZINSKEY
       LEFT JOIN POOLDATA.JSON_KLAIM c
              ON a.PXREFOBJECTKEY = c.IDPEGA
 WHERE a.PXREFOBJECTINSNAME LIKE 'CLMNP-%'
   AND c.NOPOLIS IS NULL
 ORDER BY b.PXCREATEDATETIME DESC, a.PXREFOBJECTINSNAME
OFFSET :1 ROWS FETCH NEXT :2 ROWS ONLY

-- name: list_technical
-- Tab Teknik — RDB List/GetInboxListCNP_SQL-SQL.xml
--
-- Ia membaca TABEL YANG BERBEDA: PC_ASSIGN_WORKBASKET, bukan PC_ASSIGN_WORKLIST. Itulah
-- yang membedakan antrean bersama dari penugasan perorangan (`D-26`), dan itu pula yang
-- membuat kueri ini tidak dapat disatukan dengan keempat kueri di atas.
--
-- Dua hal yang HANYA berlaku di sini:
--   * statusnya 'Acceptation', bukan 'Estimation';
--   * JSON_MASTER_ID tidak diambil sama sekali — kueri lamanya tidak memuat CARI23.
--
-- Urutannya `ORDER BY CARI21`, yaitu `b.PXCREATEDATETIME` MENAIK, dan itu dipertahankan
-- apa adanya meski berlawanan arah dengan urutan tab Admin. Antrean bersama memang wajar
-- didahulukan yang paling lama menunggu.
--
-- Bind: :1 nama akun antrean teknik · :2 offset · :3 jumlah baris
SELECT a.PZINSKEY                                    AS REFERENCE,
       a.PXREFOBJECTINSNAME                          AS CLAIM_ID,
       a.PXASSIGNEDOPERATORID                        AS ASSIGNED_OPERATOR,
       b.MASTERID                                    AS MASTER_ID,
       CAST(NULL AS VARCHAR2(100))                   AS JSON_MASTER_ID,
       c.NOPOLIS                                     AS POLICY_NUMBER,
       JSON_VALUE(c.DATA_JSON, '$.DateOfLoss')       AS LOSS_DATE,
       b.BUSINESSNAME                                AS BUSINESS_NAME,
       b.SOBNAME                                     AS BUSINESS_SOURCE,
       b.CEDINGCONAME                                AS CEDING_COMPANY,
       b.INSUREDNAME                                 AS INSURED_NAME,
       'Acceptation'                                 AS STATUS,
       (CAST(CURRENT_TIMESTAMP AS DATE)
        - CAST(b.PXCREATEDATETIME AS DATE))          AS AGING_DAYS,
       a.PXCREATEOPNAME                              AS CREATE_OPERATOR,
       b.PXUPDATEOPERATOR                            AS LAST_UPDATE_OPERATOR,
       COUNT(*) OVER ()                              AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKBASKET a
       LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK b
              ON a.PXREFOBJECTKEY = b.PZINSKEY
       LEFT JOIN POOLDATA.JSON_KLAIM c
              ON a.PXREFOBJECTKEY = c.IDPEGA
 WHERE a.PXREFOBJECTINSNAME LIKE 'CLMNP-%'
   AND a.PXASSIGNEDOPERATORID = :1
 ORDER BY b.PXCREATEDATETIME, a.PXREFOBJECTINSNAME
OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY

-- name: check_admin
-- Dipakai perintah `-periksa`: memastikan tabel penugasan perorangan DAN kedua tabel yang
-- digabungkan kepadanya terbaca dari koneksi yang dipakai.
--
-- Ia tidak menyentuh satu baris pun — yang diperiksa adalah hak baca dan keberadaan
-- tabelnya, bukan isinya. Ketiganya diperiksa sekaligus karena kegagalan yang paling
-- mungkin terjadi bukan "tabel tidak ada" melainkan "hak baca hanya diberikan pada
-- sebagiannya".
SELECT COUNT(*) AS PROBE
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK b
              ON a.PXREFOBJECTKEY = b.PZINSKEY
       LEFT JOIN POOLDATA.JSON_KLAIM c
              ON a.PXREFOBJECTKEY = c.IDPEGA
 WHERE 1 = 0

-- name: check_technical
-- Dipakai perintah `-periksa`: memastikan tabel antrean bersama terbaca.
--
-- Ia terpisah dari check_admin karena tabelnya memang berbeda, dan hak baca atas yang satu
-- tidak menyatakan apa pun tentang yang lain. Tab Teknik bergantung HANYA pada tabel ini.
SELECT COUNT(*) AS PROBE
  FROM DATAPEGA.PC_ASSIGN_WORKBASKET
 WHERE 1 = 0
