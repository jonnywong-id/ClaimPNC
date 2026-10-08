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
-- SUMBER BARU (2026-10-08) — objek kerja `b` DILEPAS
-- ============================================================================
--
-- Keputusan Work Owner 2026-10-08: DATAPEGA.PC_ASM_FW_GCNMFW_WORK sudah tidak dipakai.
-- Kueri lama menggabungkan TIGA tabel; kini DUA:
--
--   DATAPEGA.PC_ASSIGN_WORKLIST / _WORKBASKET  a   baris penugasan — tetap penggerak
--   POOLDATA.JSON_KLAIM                        c   nomor polis dan dokumen JSON klaim
--
-- Kolom bisnis yang dulu dibaca dari `b` kini dipetik dari `c.DATA_JSONBLOB` — dokumen
-- ClaimData klaim treaty, dengan jalur yang sama persis dengan yang dipakai rule Pega layar
-- saudaranya (`RDB List/GetClaimTreaty_SQL-SQL.xml`: `$.QuotationData.BusinessName`,
-- `$.QuotationData.SobName`, `$.QuotationData.CedingCoName`, `$.InsuredName`, `$.IDMaster`).
-- Kolom `b.MASTERID` dkk. adalah properti ClaimData yang SAMA, diekspos ke tabel objek kerja.
--
-- Kenapa bukan T_CLAIM_PNC / T_CLAIMLIST_ADMIN: terukur di Oracle dev (2026-10-08),
-- KEDUANYA memuat 0 dari 25 objek kerja ClaimTreatyNonProp yang ditunjuk penugasan.
-- JSON_KLAIM memuat 14 dari 25 (IDPEGA unik — LEFT JOIN tidak menggandakan baris).
--
-- Kenapa DATA_JSONBLOB, bukan DATA_JSON milik layar ini: terukur, DATA_JSON KOSONG pada
-- 14/14 baris treaty, DATA_JSONBLOB terisi 14/14. Akibatnya kolom LOSS_DATE dan
-- JSON_MASTER_ID (tetap DATA_JSON, tidak diubah) selalu kosong di dev — temuan lama,
-- bukan akibat perubahan ini.
--
-- Akibat yang terlihat pengguna (terukur di dev):
--   * kolom bisnis b.* pada kelas non-prop terisi 0/25 — kini terisi bila JSON_KLAIM ada
--     (BusinessName 12, SobName/CedingCoName/IDMaster 13, InsuredName 8 dari 25);
--   * WORK_CREATED_AT (b.PXCREATEDATETIME, kolom "Status" grid Teknik dan bahan Aging) TIDAK
--     punya padanan: PXCREATEDATETIME penugasan hanya sama pada 14/25, JSON_KLAIM.TGL_INPUT
--     0/14. Dikirim NULL bertipe — kolom "Status" kosong dan Aging 0;
--   * LAST_UPDATE_OPERATOR (b.PXUPDATEOPERATOR) TIDAK punya padanan (operator pengubah
--     penugasan sama 0/25). Dikirim NULL;
--   * urutan baris kini `a.PXCREATEDATETIME` (waktu penugasan), bukan waktu objek kerja.
-- Jumlah baris setiap tab TIDAK berubah: penugasan tetap penggerak, gabungan tetap LEFT.
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
-- CARI13      teks tetap 'Estimation' / 'Acceptation'      (tidak diambil, lihat bawah)
-- CARI14      b.SOBNAME                                    BUSINESS_SOURCE
-- CARI15      b.BUSINESSNAME                               BUSINESS_NAME
-- CARI16      b.CEDINGCONAME                               CEDING_COMPANY
-- CARI17      b.INSUREDNAME                                INSURED_NAME
-- CARI18      c.NOPOLIS                                    POLICY_NUMBER
-- CARI19      b.MASTERID                                   MASTER_ID
-- CARI20      TRUNC(SYSDATE) - TRUNC(b.PXCREATEDATETIME)   (dihitung di Go, lihat catatan 5)
-- CARI21      b.PXCREATEDATETIME                           WORK_CREATED_AT
-- CARI22      c.DATA_JSON '$.DateOfLoss'                   LOSS_DATE
-- CARI23      c.DATA_JSON '$.IDMaster'                     JSON_MASTER_ID
-- CARI24      b.PXUPDATEOPERATOR                           LAST_UPDATE_OPERATOR
--
-- Kolom "asal sebenarnya" ber-`b.` di atas adalah asal di SISTEM LAMA. Sejak 2026-10-08
-- (lihat SUMBER BARU): CARI14–17 dan CARI19 dari c.DATA_JSONBLOB; CARI21 dan CARI24 NULL.
--
-- `ASSIGNED_OPERATOR` tidak punya pasangan alias Pega: kueri lama MENYARING menurut kolom
-- itu tetapi tidak pernah memilihnya. Ia dibawa di sini supaya penyimpanan memori dapat
-- meniru penyaring yang sama, dan tidak pernah digambar sebagai kolom.
--
-- `CARI13` sebaliknya: kueri lama MEMILIHNYA tetapi tidak satu pun sel di
-- `Section/InboxClaimNonProp_Harness-Section.xml` terikat padanya, sehingga kedua teks tetap
-- itu tidak pernah sampai ke layar. Ia karena itu tidak diambil di sini. Yang digambar di
-- bawah judul kolom "Status" adalah `CARI21` — waktu objek kerja dibuat, bukan status.
--
-- `WORK_CREATED_AT` diambil KELIMA kueri, meski hanya grid Teknik yang menggambarnya di
-- bawah judul "Status". Ke-15 alias wajib sama di setiap kueri (lihat bawah), dan kolomnya
-- memang ada di tabel yang sama pada kelimanya — mengambilnya tidak menambah satu gabungan
-- pun. Ia sekaligus BAHAN perhitungan Aging di Go, lihat catatan 5.
-- Sejak 2026-10-08 nilainya NULL bertipe di kelima kueri (lihat SUMBER BARU); aliasnya
-- tetap ada supaya pemindai dan kontrak kolom tidak berubah.
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
-- KOREKSI 2026-10-08: sejak objek kerja dilepas, layar ini membaca KEDUANYA — DATA_JSON
-- tetap untuk CARI22/CARI23 (tidak diubah), DATA_JSONBLOB untuk kolom yang dulu milik `b`.
-- Isinya kini terukur di dev: DATA_JSON kosong pada 14/14 baris treaty non-prop,
-- DATA_JSONBLOB terisi 14/14. Apakah CARI22/CARI23 sebaiknya ikut pindah ke DATA_JSONBLOB
-- adalah keputusan Work Owner, bukan bagian dari migrasi ini.
--
-- Kedua pembacaan lama TIDAK disamakan di sini. Menukar salah satunya
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
-- KE-15 ALIAS WAJIB SAMA DI SETIAP KUERI
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
--    dipertahankan apa adanya — SAMPAI 2026-10-08: CARI21 (waktu objek kerja) tidak lagi
--    terbaca, sehingga kelima kueri kini mengurutkan menurut `a.PXCREATEDATETIME` (waktu
--    penugasan) dengan arah yang sama seperti sebelumnya.
--
-- 4. GABUNGAN DITULIS SEBAGAI `JOIN`, BUKAN DAFTAR TABEL BERKOMA.
--    Kueri lama memakai gabungan gaya lama (`FROM a, b, c WHERE a.x = b.y AND …`), yang
--    membuat syarat gabungan dan syarat penyaring bercampur di satu klausa. Isinya tidak
--    berubah sedikit pun — yang berubah hanya tempat syaratnya tertulis.
--
-- 5. UMUR TIDAK DIHITUNG DI SINI. Ia dihitung di Go, dan itu KOREKSI atas cacat nyata.
--    Kueri lama memakai `TRUNC(SYSDATE) - TRUNC(b.PXCREATEDATETIME)`. Padanan portabel yang
--    dianjurkan `09-DATABASE-STRATEGY.md` §4 adalah `CAST(x AS DATE)`, dan berkas ini pernah
--    memakainya:
--
--      CAST(CURRENT_TIMESTAMP AS DATE) - CAST(b.PXCREATEDATETIME AS DATE)
--
--    Catatan di sini dulu menyatakan hasilnya "SAMA PERSIS". Itu KELIRU. DATE pada Oracle
--    membawa jam, sehingga cast tidak memangkas apa pun dan selisihnya berupa PECAHAN hari.
--    Terukur langsung dari basis data pengembangan pada 2026-09-30:
--
--      CAST(CURRENT_TIMESTAMP AS DATE) - CAST(b.PXCREATEDATETIME AS DATE)  ->  979.8191898…
--      TRUNC(SYSDATE)                  - TRUNC(b.PXCREATEDATETIME)         ->  979
--
--    Akibatnya bukan angka yang kurang rapi melainkan SELURUH ANTREAN GAGAL DIMUAT: godror
--    menyerahkan angka berpecahan itu sebagai teks, dan pemindainya mengharapkan bilangan
--    bulat. Tab Admin lolos hanya karena pemanggil yang belum punya penugasan tidak memindai
--    satu baris pun — tab Teknik membaca antrean bersama yang selalu berisi, sehingga di
--    sanalah galatnya muncul.
--
--    Tidak ada bentuk SQL yang memangkas jam di Oracle DAN di PostgreSQL sekaligus: `TRUNC`
--    khas Oracle, `DATE_TRUNC` khas PostgreSQL. Perhitungannya karena itu pindah ke
--    `inboxclaimtreatynonprop.AgingDaysSince`, terhadap tanggal WIB (`F-5`), memakai
--    `WORK_CREATED_AT` yang memang sudah diambil. Modul `inboxosclaimpercabang` menempuh
--    jalan yang sama atas sebab yang sama.
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
-- * `WORK_CREATED_AT` dikembalikan sebagai NILAI WAKTU, bukan teks berformat. Bentuk yang
--   dibaca pengguna (`20240201T095612.955 GMT`) disusun di Go oleh
--   `inboxclaimtreatynonprop.FormatPegaDateTime`. Memformatnya di SQL menuntut `TO_CHAR`,
--   yang ada di daftar terlarang `08-TECHNICAL-STRATEGY.md` §4.3 — dan pemformatan tampilan
--   memang milik Go, bukan milik kueri. (Sejak 2026-10-08 nilainya NULL bertipe
--   `CAST(NULL AS TIMESTAMP)`; pemindai `sql.NullTime` menghasilkan teks kosong dan umur 0.)
--
-- * Nama akun antrean teknik dikirim sebagai BIND, bukan ditulis di sini. Nilainya satu
--   tempat saja — inboxclaimtreatynonprop.TechnicalWorkbasket — supaya SQL dan penyimpanan
--   memori tidak dapat berselisih tanpa ketahuan.
--
-- * `CAST(NULL AS VARCHAR2(…))` memakai tipe khas Oracle, mengikuti modul yang sudah ada
--   (riwayatklaim, inboxadmin, inboxclaimtreatyprop). Ia satu-satunya bentuk tak portabel
--   di berkas ini dan tercatat sebagai utang teknis yang diselesaikan serentak untuk
--   seluruh modul saat perpindahan ke PostgreSQL (`09-DATABASE-STRATEGY.md` §10), bukan
--   sepihak di sini. Sejak 2026-10-08 ada bentuk tak portabel KEDUA: `JSON_VALUE` atas
--   kolom BLOB (`DATA_JSONBLOB`) — di PostgreSQL kolom itu harus `jsonb`, bukan `bytea`.
--   Bentuk yang sama sudah dipakai rule Pega layar Prop; ia ikut daftar utang yang sama.

-- name: list_admin
-- Tab Admin tanpa checkbox apa pun — antrean milik pemanggil.
-- — RDB List/GetKlaimNonPropAdmin_SQL-SQL.xml
--
-- Bind: :1 login pemanggil · :2 offset · :3 jumlah baris
SELECT a.PZINSKEY                                    AS REFERENCE,
       a.PXREFOBJECTINSNAME                          AS CLAIM_ID,
       a.PXASSIGNEDOPERATORID                        AS ASSIGNED_OPERATOR,
       JSON_VALUE(c.DATA_JSONBLOB, '$.IDMaster')     AS MASTER_ID,
       JSON_VALUE(c.DATA_JSON, '$.IDMaster')         AS JSON_MASTER_ID,
       c.NOPOLIS                                     AS POLICY_NUMBER,
       JSON_VALUE(c.DATA_JSON, '$.DateOfLoss')       AS LOSS_DATE,
       JSON_VALUE(c.DATA_JSONBLOB, '$.QuotationData.BusinessName')
                                                     AS BUSINESS_NAME,
       JSON_VALUE(c.DATA_JSONBLOB, '$.QuotationData.SobName')
                                                     AS BUSINESS_SOURCE,
       JSON_VALUE(c.DATA_JSONBLOB, '$.QuotationData.CedingCoName')
                                                     AS CEDING_COMPANY,
       JSON_VALUE(c.DATA_JSONBLOB, '$.InsuredName')  AS INSURED_NAME,
       CAST(NULL AS TIMESTAMP)                       AS WORK_CREATED_AT,
       a.PXCREATEOPNAME                              AS CREATE_OPERATOR,
       CAST(NULL AS VARCHAR2(100))                   AS LAST_UPDATE_OPERATOR,
       COUNT(*) OVER ()                              AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       LEFT JOIN POOLDATA.JSON_KLAIM c
              ON a.PXREFOBJECTKEY = c.IDPEGA
 WHERE a.PXREFOBJECTINSNAME LIKE 'CLMNP-%'
   AND a.PXASSIGNEDOPERATORID = :1
 ORDER BY a.PXCREATEDATETIME DESC, a.PXREFOBJECTINSNAME
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
       JSON_VALUE(c.DATA_JSONBLOB, '$.IDMaster')     AS MASTER_ID,
       JSON_VALUE(c.DATA_JSON, '$.IDMaster')         AS JSON_MASTER_ID,
       c.NOPOLIS                                     AS POLICY_NUMBER,
       JSON_VALUE(c.DATA_JSON, '$.DateOfLoss')       AS LOSS_DATE,
       JSON_VALUE(c.DATA_JSONBLOB, '$.QuotationData.BusinessName')
                                                     AS BUSINESS_NAME,
       JSON_VALUE(c.DATA_JSONBLOB, '$.QuotationData.SobName')
                                                     AS BUSINESS_SOURCE,
       JSON_VALUE(c.DATA_JSONBLOB, '$.QuotationData.CedingCoName')
                                                     AS CEDING_COMPANY,
       JSON_VALUE(c.DATA_JSONBLOB, '$.InsuredName')  AS INSURED_NAME,
       CAST(NULL AS TIMESTAMP)                       AS WORK_CREATED_AT,
       a.PXCREATEOPNAME                              AS CREATE_OPERATOR,
       CAST(NULL AS VARCHAR2(100))                   AS LAST_UPDATE_OPERATOR,
       COUNT(*) OVER ()                              AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       LEFT JOIN POOLDATA.JSON_KLAIM c
              ON a.PXREFOBJECTKEY = c.IDPEGA
 WHERE a.PXREFOBJECTINSNAME LIKE 'CLMNP-%'
 ORDER BY a.PXCREATEDATETIME DESC, a.PXREFOBJECTINSNAME
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
       JSON_VALUE(c.DATA_JSONBLOB, '$.IDMaster')     AS MASTER_ID,
       JSON_VALUE(c.DATA_JSON, '$.IDMaster')         AS JSON_MASTER_ID,
       c.NOPOLIS                                     AS POLICY_NUMBER,
       JSON_VALUE(c.DATA_JSON, '$.DateOfLoss')       AS LOSS_DATE,
       JSON_VALUE(c.DATA_JSONBLOB, '$.QuotationData.BusinessName')
                                                     AS BUSINESS_NAME,
       JSON_VALUE(c.DATA_JSONBLOB, '$.QuotationData.SobName')
                                                     AS BUSINESS_SOURCE,
       JSON_VALUE(c.DATA_JSONBLOB, '$.QuotationData.CedingCoName')
                                                     AS CEDING_COMPANY,
       JSON_VALUE(c.DATA_JSONBLOB, '$.InsuredName')  AS INSURED_NAME,
       CAST(NULL AS TIMESTAMP)                       AS WORK_CREATED_AT,
       a.PXCREATEOPNAME                              AS CREATE_OPERATOR,
       CAST(NULL AS VARCHAR2(100))                   AS LAST_UPDATE_OPERATOR,
       COUNT(*) OVER ()                              AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       LEFT JOIN POOLDATA.JSON_KLAIM c
              ON a.PXREFOBJECTKEY = c.IDPEGA
 WHERE a.PXREFOBJECTINSNAME LIKE 'CLMNP-%'
   AND a.PXASSIGNEDOPERATORID = :1
   AND c.NOPOLIS IS NULL
 ORDER BY a.PXCREATEDATETIME DESC, a.PXREFOBJECTINSNAME
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
       JSON_VALUE(c.DATA_JSONBLOB, '$.IDMaster')     AS MASTER_ID,
       JSON_VALUE(c.DATA_JSON, '$.IDMaster')         AS JSON_MASTER_ID,
       c.NOPOLIS                                     AS POLICY_NUMBER,
       JSON_VALUE(c.DATA_JSON, '$.DateOfLoss')       AS LOSS_DATE,
       JSON_VALUE(c.DATA_JSONBLOB, '$.QuotationData.BusinessName')
                                                     AS BUSINESS_NAME,
       JSON_VALUE(c.DATA_JSONBLOB, '$.QuotationData.SobName')
                                                     AS BUSINESS_SOURCE,
       JSON_VALUE(c.DATA_JSONBLOB, '$.QuotationData.CedingCoName')
                                                     AS CEDING_COMPANY,
       JSON_VALUE(c.DATA_JSONBLOB, '$.InsuredName')  AS INSURED_NAME,
       CAST(NULL AS TIMESTAMP)                       AS WORK_CREATED_AT,
       a.PXCREATEOPNAME                              AS CREATE_OPERATOR,
       CAST(NULL AS VARCHAR2(100))                   AS LAST_UPDATE_OPERATOR,
       COUNT(*) OVER ()                              AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       LEFT JOIN POOLDATA.JSON_KLAIM c
              ON a.PXREFOBJECTKEY = c.IDPEGA
 WHERE a.PXREFOBJECTINSNAME LIKE 'CLMNP-%'
   AND c.NOPOLIS IS NULL
 ORDER BY a.PXCREATEDATETIME DESC, a.PXREFOBJECTINSNAME
OFFSET :1 ROWS FETCH NEXT :2 ROWS ONLY

-- name: list_technical
-- Tab Teknik — RDB List/GetInboxListCNP_SQL-SQL.xml
--
-- Ia membaca TABEL YANG BERBEDA: PC_ASSIGN_WORKBASKET, bukan PC_ASSIGN_WORKLIST. Itulah
-- yang membedakan antrean bersama dari penugasan perorangan (`D-26`), dan itu pula yang
-- membuat kueri ini tidak dapat disatukan dengan keempat kueri di atas.
--
-- Dua hal yang HANYA berlaku di sini:
--   * JSON_MASTER_ID tidak diambil sama sekali — kueri lamanya tidak memuat CARI23, dan
--     grid Teknik memang memakai `b.MASTERID` sebagai kolom "ID Master" (sejak 2026-10-08
--     dipetik dari `$.IDMaster` pada DATA_JSONBLOB — properti yang sama);
--   * hanya grid inilah yang MENGGAMBAR `WORK_CREATED_AT`, di bawah judul "Status" (sejak
--     2026-10-08 NULL — lihat SUMBER BARU).
--
-- Urutannya dulu `ORDER BY CARI21`, yaitu `b.PXCREATEDATETIME` MENAIK. Sejak 2026-10-08
-- waktu objek kerja tidak lagi terbaca; arahnya dipertahankan (MENAIK) atas waktu penugasan
-- `a.PXCREATEDATETIME`. Antrean bersama memang wajar didahulukan yang paling lama menunggu.
--
-- Bind: :1 nama akun antrean teknik · :2 offset · :3 jumlah baris
SELECT a.PZINSKEY                                    AS REFERENCE,
       a.PXREFOBJECTINSNAME                          AS CLAIM_ID,
       a.PXASSIGNEDOPERATORID                        AS ASSIGNED_OPERATOR,
       JSON_VALUE(c.DATA_JSONBLOB, '$.IDMaster')     AS MASTER_ID,
       CAST(NULL AS VARCHAR2(100))                   AS JSON_MASTER_ID,
       c.NOPOLIS                                     AS POLICY_NUMBER,
       JSON_VALUE(c.DATA_JSON, '$.DateOfLoss')       AS LOSS_DATE,
       JSON_VALUE(c.DATA_JSONBLOB, '$.QuotationData.BusinessName')
                                                     AS BUSINESS_NAME,
       JSON_VALUE(c.DATA_JSONBLOB, '$.QuotationData.SobName')
                                                     AS BUSINESS_SOURCE,
       JSON_VALUE(c.DATA_JSONBLOB, '$.QuotationData.CedingCoName')
                                                     AS CEDING_COMPANY,
       JSON_VALUE(c.DATA_JSONBLOB, '$.InsuredName')  AS INSURED_NAME,
       CAST(NULL AS TIMESTAMP)                       AS WORK_CREATED_AT,
       a.PXCREATEOPNAME                              AS CREATE_OPERATOR,
       CAST(NULL AS VARCHAR2(100))                   AS LAST_UPDATE_OPERATOR,
       COUNT(*) OVER ()                              AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKBASKET a
       LEFT JOIN POOLDATA.JSON_KLAIM c
              ON a.PXREFOBJECTKEY = c.IDPEGA
 WHERE a.PXREFOBJECTINSNAME LIKE 'CLMNP-%'
   AND a.PXASSIGNEDOPERATORID = :1
 ORDER BY a.PXCREATEDATETIME, a.PXREFOBJECTINSNAME
OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY

-- name: check_admin
-- Dipakai perintah `-periksa`: memastikan tabel penugasan perorangan DAN tabel yang
-- digabungkan kepadanya (sejak 2026-10-08 hanya POOLDATA.JSON_KLAIM) terbaca dari koneksi
-- yang dipakai.
--
-- Ia tidak menyentuh satu baris pun — yang diperiksa adalah hak baca dan keberadaan
-- tabelnya, bukan isinya. Keduanya diperiksa sekaligus karena kegagalan yang paling
-- mungkin terjadi bukan "tabel tidak ada" melainkan "hak baca hanya diberikan pada
-- sebagiannya".
SELECT COUNT(*) AS PROBE
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
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
