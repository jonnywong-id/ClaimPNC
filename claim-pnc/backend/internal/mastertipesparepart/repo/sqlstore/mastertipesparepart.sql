-- Kueri modul Master Tipe Sparepart.
--
-- DUA tabel. Aplikasi ini MENULIS yang pertama dan hanya MEMBACA yang kedua:
--
--   POOLDATA.GCNM_M_SPAREPART_TYPE      ditulis modul ini
--   POOLDATA.GCNM_M_SPAREPART_CATEGORY  dibaca saja; penulisnya masterkategorisparepart
--
-- Pembagian itu memenuhi P-1 — satu tabel, satu penulis. Tidak ada satu pun pernyataan di
-- berkas ini yang menulis tabel kedua.
--
-- Kewenangan menulis berpindah dari Pega ke Go saat modulnya lulus gerbang 2 (P-1). Selama
-- Pega masih penulisnya, layar ini harus dijalankan dalam modus baca saja di produksi.
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
-- TABELNYA PUNYA EMPAT KOLOM, DAN ITU SUDAH DIPASTIKAN
-- ============================================================================
--
-- Kesembilan rule Pega yang menyentuh tabel ini tidak satu pun menyebut kolom di luar
-- keempat ini:
--
--   PART_SECTION_ID     BrowseTipeSparepart · BrowseSparepartTipeClaimHE2
--                       BrowseSparepartTypeClaimHE_sql · BrowseMasterSparepartTypeClaimHE_sql
--                       InsertMasterSparepartType_sql · UpdateMasterSparepartType_sql2
--                       CountMasterTipeSparepartManager
--   PART_SECTION_NAME   keenam rule di atas, ditambah ValidationSparepartType
--   PART_CATEGORY_ID    keenam rule browse, insert, dan update
--   APPROVAL            BrowseTipeSparepart · BrowseSparepartTipeClaimHE2
--                       BrowseMasterSparepartTypeClaimHE_sql · InsertMasterSparepartType_sql
--                       UpdateMasterSparepartType_sql2 · CountMasterTipeSparepartManager
--
-- Tidak ada kolom pencatat pelaku, tidak ada stempel waktu, dan tidak ada kolom alasan
-- penolakan. Itu keterbatasan tabelnya, dan ia menentukan bentuk layarnya.
--
--
-- ============================================================================
-- KENAPA ALIAS KOLOM PEGA TIDAK DIBAWA
-- ============================================================================
--
-- Seluruh rule browse mengalias kolomnya menjadi nama yang tidak ada hubungannya dengan
-- isinya, dan di modul ini bentuknya yang paling menyesatkan dari seluruh rumpun sparepart:
--
--   select a.PART_SECTION_ID   as "CityID",
--          a.PART_SECTION_NAME as "City",
--          a.PART_CATEGORY_ID  as "District",
--          b.PART_CATEGORY_NAME as "DistrictID"
--
-- Perhatikan dua yang terakhir: ID kategori dialiaskan "District" dan NAMANYA dialiaskan
-- "DistrictID" — berlawanan dengan pola "…ID" yang dipakai dua alias sebelumnya. Membaca
-- rule lama berarti mengingat bahwa "DistrictID" justru bukan sebuah ID. Alias itu TIDAK
-- dibawa; kolomnya disebut nama aslinya.
--
--
-- ============================================================================
-- KEDUA KUNCI BERTIPE ANGKA, DAN ITU BUKAN TEBAKAN
-- ============================================================================
--
-- `RDB List/InsertMasterSparepartType_sql-SQL.xml` menerbitkan PART_SECTION_ID dengan
-- `nvl(max(PART_SECTION_ID),0)+1`. Bila kolomnya VARCHAR2, `max(...)` akan mengembalikan
-- maksimum LEKSIKOGRAFIS — "9" lebih besar dari "10" — sehingga ID ke-11 akan bertabrakan
-- dengan yang sudah ada dan sistem lama akan rusak sejak baris kesepuluh. Ia tidak rusak,
-- jadi kolomnya angka. Alasan yang sama berlaku pada PART_CATEGORY_ID di tabel kategori.
--
-- Itu sebabnya `type_list` MENGURUTKAN berdasarkan kolomnya langsung dan bukan berdasarkan
-- teksnya — berbeda dari Master Sparepart, yang ID-nya memang teks.
--
--
-- ============================================================================
-- INNER JOIN SISTEM LAMA DIGANTI LEFT JOIN, DAN ITU DISENGAJA
-- ============================================================================
--
-- `BrowseMasterSparepartTypeClaimHE_sql` menggabungkan kedua tabel dengan gaya koma:
--
--   from POOLDATA.gcnm_m_sparepart_type a, POOLDATA.GCNM_M_SPAREPART_CATEGORY b
--   where A.PART_CATEGORY_ID = B.PART_CATEGORY_ID
--
-- Itu INNER JOIN. Akibatnya di sistem lama: **tipe yang menunjuk kategori yang tidak ada
-- akan HILANG dari daftar** — tidak muncul di tab mana pun, tidak dapat disunting, dan
-- tidak dapat diperbaiki dari layar. Barisnya tetap ada di basis data, dan tetap terbaca
-- oleh dropdown Tipe pada layar Master Sparepart yang tidak melakukan JOIN sama sekali.
--
-- Modul ini memakai LEFT JOIN, sehingga barisnya TETAP TERLIHAT dengan kolom Kategori
-- kosong. Ini SELISIH PERILAKU yang disengaja terhadap Pega, dan satu-satunya pada jalur
-- baca modul ini. Alasannya: baris yang tidak dapat dilihat tidak dapat diperbaiki, dan
-- modul ini menuntut kategori diisi saat menyimpan — tanpa LEFT JOIN, baris rusak itu
-- terkunci selamanya.
--
-- Selisihnya akan muncul pada uji kesetaraan gerbang 1 sebagai "baris berlebih" pada
-- entitas yang datanya memang sudah rusak. `claimpnc -periksa` melaporkan jumlahnya lebih
-- dulu lewat type_count_orphan_category, supaya selisihnya dapat dijelaskan sebelum
-- pengujian dijalankan, bukan sesudah.
--
--
-- ============================================================================
-- APPROVAL DIBANDINGKAN DENGAN TRIM
-- ============================================================================
--
-- Rule lama membandingkannya langsung (`where APPROVAL = '0'`). Bila kolomnya CHAR dan
-- bukan VARCHAR2, Oracle memadatkan pembandingnya dengan spasi sehingga perbandingan itu
-- tetap benar — tetapi PostgreSQL tidak melakukannya, dan baris yang sama akan hilang
-- setelah pindah basis data (D-24).
--
-- TRIM dipasang di kedua sisi supaya kedua basis data menjawab sama. Ia tidak mengubah
-- hasil di Oracle, dan ia menyelamatkan hasil di PostgreSQL.
--
-- Harganya index pada APPROVAL tidak terpakai. Tabel ini master penggolongan yang isinya
-- berorde puluhan sampai ratusan baris, sehingga pemindaian penuhnya tidak berarti apa-apa
-- — perhitungan yang berbeda dari tabel klaim berisi puluhan juta baris.


-- name: type_list
--
-- Daftar satu tab, beserta nama kategori induknya.
--
-- Padanan `RDB List/BrowseMasterSparepartTypeClaimHE_sql-SQL.xml` — satu-satunya rule
-- browse yang membawa nama kategori — dengan penyaring status yang berparameter seperti di
-- sana, dan tanpa penyaring `PART_SECTION_ID` karena yang diminta di sini adalah daftar,
-- bukan satu baris.
--
-- LEFT JOIN, bukan inner join. Lihat banner di kepala berkas ini.
SELECT T.PART_SECTION_ID,
       T.PART_SECTION_NAME,
       T.PART_CATEGORY_ID,
       C.PART_CATEGORY_NAME,
       T.APPROVAL
  FROM POOLDATA.GCNM_M_SPAREPART_TYPE T
  LEFT JOIN POOLDATA.GCNM_M_SPAREPART_CATEGORY C
         ON C.PART_CATEGORY_ID = T.PART_CATEGORY_ID
 WHERE TRIM(T.APPROVAL) = :1
 ORDER BY T.PART_SECTION_ID

-- name: type_list_search
--
-- Sama dengan type_list, ditambah penyaring kata kunci.
--
-- Kueri TERSENDIRI, bukan satu kueri yang klausanya ditempel saat kata kuncinya ada.
-- Merangkai teks SQL adalah persis pola `{ASIS:...}` yang 03-CURRENT-ARCHITECTURE.md §4.5
-- catat sebagai celah injeksi, dan pemisahan ini membuat kedua bentuknya dapat dibaca utuh
-- di berkas ini.
--
-- Kata kuncinya dicocokkan ke NAMA TIPE dan NAMA KATEGORI sekaligus — berbeda dari Master
-- Kategori Sparepart yang hanya punya satu kolom untuk dicari. Alasannya ada pada
-- mastertipesparepart.Filter: kategori adalah cara pengguna mengelompokkan tipe di
-- kepalanya.
--
-- Parameter yang sama dipakai DUA KALI (`:2` pada kedua sisi OR). Bentuk itu sengaja
-- dipilih alih-alih mengirim nilai yang sama dua kali sebagai `:2` dan `:3` — driver Oracle
-- dan PostgreSQL keduanya menerima penyebutan ulang satu parameter posisional, dan satu
-- nilai untuk satu maksud lebih sulit dibuat tidak sinkron.
--
-- UPPER di kedua sisi, bukan LOWER: rule validasi lamanya memakai `upper(...)`, dan memakai
-- pasangan yang sama membuat pencarian dan pemeriksaan keunikan tidak pernah berbeda soal
-- huruf besar-kecil.
--
-- Kata kuncinya sudah dibungkus tanda persen oleh pemanggil, bukan di sini: menempelkannya
-- di dalam teks SQL berarti merangkai nilai ke dalam pernyataan.
SELECT T.PART_SECTION_ID,
       T.PART_SECTION_NAME,
       T.PART_CATEGORY_ID,
       C.PART_CATEGORY_NAME,
       T.APPROVAL
  FROM POOLDATA.GCNM_M_SPAREPART_TYPE T
  LEFT JOIN POOLDATA.GCNM_M_SPAREPART_CATEGORY C
         ON C.PART_CATEGORY_ID = T.PART_CATEGORY_ID
 WHERE TRIM(T.APPROVAL) = :1
   AND (UPPER(T.PART_SECTION_NAME) LIKE :2
        OR UPPER(C.PART_CATEGORY_NAME) LIKE :2)
 ORDER BY T.PART_SECTION_ID

-- name: type_get
--
-- Satu baris menurut kuncinya. Padanan
-- `RDB List/BrowseSparepartTypeClaimHE_sql-SQL.xml`, yang dipakai
-- `Activity/SetMasterTipeSparepart_act` untuk memuat baris ke form.
--
-- Rule itu memang TIDAK menyaring APPROVAL, dan itu ditiru: form harus dapat membuka baris
-- dari tab mana pun. Kembarannya yang berparameter status,
-- `BrowseMasterSparepartTypeClaimHE_sql`, dipakai jalur lain.
--
-- TRIM pada kuncinya dengan alasan yang sama seperti pada APPROVAL, ditambah satu lagi:
-- nilai ini datang dari jalur URL dan dari kolom SPAREPART_HE.TIPE_SPART yang bertipe teks,
-- sehingga spasi ujung benar-benar mungkin sampai ke sini.
SELECT T.PART_SECTION_ID,
       T.PART_SECTION_NAME,
       T.PART_CATEGORY_ID,
       C.PART_CATEGORY_NAME,
       T.APPROVAL
  FROM POOLDATA.GCNM_M_SPAREPART_TYPE T
  LEFT JOIN POOLDATA.GCNM_M_SPAREPART_CATEGORY C
         ON C.PART_CATEGORY_ID = T.PART_CATEGORY_ID
 WHERE TRIM(CAST(T.PART_SECTION_ID AS VARCHAR(64))) = :1

-- name: type_find_by_name
--
-- Padanan `RDB List/ValidationSparepartType-SQL.xml`:
--
--   select PART_SECTION_NAME from POOLDATA.gcnm_m_sparepart_type
--    where upper(PART_SECTION_NAME) = {TempValidateTypeSparepart.CaseID}
--
-- PERHATIKAN DUA HAL YANG TIDAK ADA DI SANA: penyaring APPROVAL, dan penyaring
-- PART_CATEGORY_ID. Akibatnya nama tipe harus unik di SELURUH tabel, bukan di dalam satu
-- kategori — "KACA DEPAN" tidak dapat ada sekaligus di BODY dan KABIN — dan nama yang
-- pernah DITOLAK tetap memblokir selamanya.
--
-- Keduanya ditiru apa adanya atas keputusan Work Owner 2026-09-21 (P-5). Lihat
-- mastertipesparepart.ErrNameTaken.
--
-- Yang DITAMBAHKAN hanyalah TRIM dan kolom selain nama pada hasilnya. TRIM supaya nama
-- berspasi ujung tidak lolos sebagai nama yang berbeda; kolom lainnya supaya pemanggil
-- dapat mengecualikan baris yang sedang disunting — tanpa itu, menyimpan baris tanpa
-- mengubah namanya akan ditolak oleh dirinya sendiri.
--
-- TANPA JOIN ke tabel kategori: yang dibutuhkan pemanggil hanyalah kuncinya, dan JOIN di
-- sini akan menambah pekerjaan pada jalur yang dilewati SETIAP penyimpanan.
--
-- FETCH FIRST 1 ROW ONLY, bukan ROWNUM: yang dibutuhkan hanya keberadaannya, dan bentuk ini
-- berjalan di kedua basis data (D-20). Ia sudah dipakai 35 rule pada sistem lama, jadi bukan
-- hal baru bagi tim.
SELECT PART_SECTION_ID,
       PART_SECTION_NAME,
       PART_CATEGORY_ID,
       APPROVAL
  FROM POOLDATA.GCNM_M_SPAREPART_TYPE
 WHERE UPPER(TRIM(PART_SECTION_NAME)) = :1
 ORDER BY PART_SECTION_ID
 FETCH FIRST 1 ROW ONLY

-- name: type_lock_table
--
-- Mengunci tabel untuk seluruh sisa transaksi penambahan.
--
-- # Kenapa mengunci TABEL, bukan baris
--
-- Karena yang perlu dijaga adalah nilai yang BELUM ADA. ID diterbitkan dengan
-- `MAX(...)+1`, dan dua penyimpanan bersamaan dapat membaca nilai maksimum yang sama lalu
-- menyisipkan ID kembar. `FOR UPDATE` tidak dapat menolongnya: ia mengunci baris yang sudah
-- ada, sedangkan yang bertabrakan adalah baris yang sedang dibuat keduanya.
--
-- Sistem lama tidak menjaganya sama sekali — `nvl(max(PART_SECTION_ID),0)+1` berada di
-- dalam satu INSERT tanpa penguncian apa pun. Penutupannya mengikuti keputusan yang sama
-- pada Master Kategori Sparepart (Work Owner, 2026-09-21): bentuk ID-nya tetap sama persis
-- dengan Pega (P-5), dan modulnya tidak tertahan menunggu D-63.
--
-- # Yang dikunci hanya SATU tabel
--
-- Hanya tabel tipe. Tabel kategori TIDAK dikunci meski ikut dibaca di dalam transaksi yang
-- sama: modul ini tidak pernah menulisnya, dan menguncinya akan menahan penambahan kategori
-- di layar lain tanpa satu pun alasan.
--
-- Konsekuensinya disadari: kategori yang dipilih dapat DITOLAK oleh petugas lain pada
-- detik antara pemeriksaannya dan penyisipannya. Hasilnya sebuah tipe yang menunjuk
-- kategori tidak-disetujui — keadaan yang sama persis dengan yang terjadi bila kategori
-- ditolak semenit kemudian, dan yang memang tidak dijaga apa pun di sistem lama maupun di
-- sini. Menguncinya tidak akan menutup keadaan itu, hanya mempersempit jendelanya dari
-- selamanya menjadi selamanya-dikurangi-sedetik.
--
-- # Harganya, dan kenapa ia terjangkau di SINI
--
-- EXCLUSIVE menahan penulisan lain atas tabel ini sampai transaksinya selesai. Itu harga
-- yang mahal pada tabel transaksi, dan hampir gratis di sini: tipe suku cadang ditambahkan
-- beberapa kali setahun, transaksinya memuat empat pernyataan pendek, dan PEMBACAAN tidak
-- terhalang di kedua basis data.
--
-- Pemilihan modus itu disengaja. Oracle EXCLUSIVE dan PostgreSQL EXCLUSIVE keduanya
-- mengizinkan SELECT berjalan terus, sehingga dropdown Tipe pada layar Master Sparepart
-- tidak pernah tertahan oleh penambahan yang sedang berjalan.
--
-- LOCK TABLE adalah satu-satunya pernyataan non-DML di seluruh modul ini. Bentuknya sama
-- persis di Oracle 19c dan PostgreSQL 17+, sehingga ia tidak melanggar D-20.
LOCK TABLE POOLDATA.GCNM_M_SPAREPART_TYPE IN EXCLUSIVE MODE

-- name: type_next_id
--
-- Menerbitkan ID berikutnya, meniru `InsertMasterSparepartType_sql` apa adanya kecuali
-- NVL yang diganti COALESCE (D-20).
--
-- WAJIB dijalankan setelah type_lock_table, di dalam transaksi yang sama. Di luar itu
-- nilainya dapat basi sebelum dipakai.
--
-- Tanpa FROM DUAL: agregat atas tabelnya sendiri sudah menyediakan baris hasil, sehingga
-- bentuk khas Oracle itu tidak diperlukan sama sekali di modul ini.
SELECT COALESCE(MAX(PART_SECTION_ID), 0) + 1
  FROM POOLDATA.GCNM_M_SPAREPART_TYPE

-- name: type_insert
--
-- Keempat kolomnya pada urutan yang sama dengan Pega.
--
-- ID diterbitkan lebih dulu oleh type_next_id dan dikirim sebagai parameter, bukan ditanam
-- sebagai sub-kueri seperti pada rule lama. Alasannya bukan selera: baris yang tersimpan
-- harus dikembalikan ke layar beserta ID-nya, dan sub-kueri di dalam VALUES tidak memberi
-- tahu pemanggil nilai apa yang terpakai.
INSERT INTO POOLDATA.GCNM_M_SPAREPART_TYPE
       (PART_SECTION_ID, PART_SECTION_NAME, PART_CATEGORY_ID, APPROVAL)
VALUES (:1, :2, :3, :4)

-- name: type_update
--
-- Padanan `RDB List/UpdateMasterSparepartType_sql2-SQL.xml`:
--
--   UPDATE POOLDATA.gcnm_m_sparepart_type
--      SET PART_SECTION_NAME = {InputKategori.CITY_ID},
--          PART_CATEGORY_ID  = {InputKategori.DISC_JASA},
--          APPROVAL          = {InputKategori.NO_ACCOUNT}
--    WHERE PART_SECTION_ID   = {InputKategori.ACCOUNT_ID}
--
-- PERHATIKAN NAMA PROPERTINYA, DAN DARI MANA ASALNYA. Nama tipe dikirim lewat `CITY_ID`,
-- kategori lewat `DISC_JASA`, status lewat `NO_ACCOUNT`, dan kuncinya lewat `ACCOUNT_ID` —
-- empat property yang namanya tidak ada hubungannya dengan apa yang dibawanya. Keempatnya
-- bahkan milik kelas LAIN: `ASM-FW-GCNMFW-Int-BENGKEL_HE`, kelas Master Bengkel. Layar tipe
-- sparepart meminjam property bengkel karena bentuknya kebetulan cocok.
--
-- Membaca rule lama berarti menelusuri keempatnya sampai ke pemanggilnya untuk tahu isian
-- mana yang mana; di sini keempatnya disebut apa adanya.
--
-- ID TIDAK ikut ditulis — ia penyaring WHERE, dan berada di posisi parameter terakhir.
UPDATE POOLDATA.GCNM_M_SPAREPART_TYPE
   SET PART_SECTION_NAME = :1,
       PART_CATEGORY_ID  = :2,
       APPROVAL          = :3
 WHERE TRIM(CAST(PART_SECTION_ID AS VARCHAR(64))) = :4

-- name: type_set_status
--
-- Menetapkan APPROVAL satu baris. Dipanggil sekali per baris di dalam SATU transaksi.
--
-- # BENTUKNYA DIREKONSTRUKSI, BUKAN DIBACA
--
-- `Activity/UpdateTipeSparepart_act` menyiapkan kunci dan status lalu memanggil sebuah rule
-- UPDATE — dan rule itu TIDAK ADA di antara 2.634 berkas export (R-16). Tipe juga tidak
-- ikut `Activity/SetApprovalAllMaster`, yang hanya melayani M_BENGKEL_HE, M_PANEL_HE, dan
-- M_SPAREPART_HE.
--
-- Yang dapat dipastikan hanyalah: activity itu menyiapkan kunci dan status, lalu memanggil
-- sebuah rule UPDATE atas tabel ini — dan langkah-langkahnya (`set ID dan approval`,
-- `update ke tabel`) terbaca dari pyStepsDescription-nya. Bentuk di bawah mengikuti
-- type_update yang memang terbaca, dikurangi kedua kolom yang tidak disentuh sebuah
-- keputusan.
--
-- Dinyatakan di sini supaya ia dapat diuji ulang begitu rule aslinya tiba — bukan tersamar
-- sebagai fakta.
--
-- # Kenapa satu baris per pernyataan, bukan IN (...)
--
-- Karena jumlah parameter IN berubah-ubah, dan satu-satunya cara menuliskannya sebagai satu
-- pernyataan adalah merangkai teks SQL — persis yang dilarang aturan (2) di kepala berkas
-- ini. Keputusan borongan tetap ATOMIK karena seluruh pernyataannya berada di dalam satu
-- transaksi; jumlah barisnya dibatasi transport (lihat maxDecisionRows).
UPDATE POOLDATA.GCNM_M_SPAREPART_TYPE
   SET APPROVAL = :1
 WHERE TRIM(CAST(PART_SECTION_ID AS VARCHAR(64))) = :2

-- name: type_category_list
--
-- Daftar kategori yang menjadi pilihan dropdown `---PILIH KATEGORI---`.
--
-- Padanan `RDB List/BrowseMasterSparepartCategoryClaimHE-SQL.xml`:
--
--   select PART_CATEGORY_ID as "CityID", PART_CATEGORY_NAME as "City"
--     from POOLDATA.gcnm_m_sparepart_category where APPROVAL = {TempStatus.City}
--
-- Penyaring APPROVAL-nya berparameter dan diisi '1' oleh pemanggil — nilai itu
-- REKONSTRUKSI, bukan pembacaan; rantai buktinya ada pada
-- mastertipesparepart.LookupRepo.ListCategories.
--
-- TABEL INI TIDAK PERNAH DITULIS modul ini. Penulisnya masterkategorisparepart (P-1).
--
-- ORDER BY DITAMBAHKAN; kueri lama tidak menyebut satu pun. Nama yang dipilih, bukan ID,
-- karena inilah daftar yang dipindai mata pengguna pada dropdown — sama dengan
-- `sparepart_category_list` pada modul Master Sparepart.
SELECT PART_CATEGORY_ID,
       PART_CATEGORY_NAME
  FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY
 WHERE TRIM(APPROVAL) = :1
 ORDER BY PART_CATEGORY_NAME

-- name: type_count_by_status
--
-- Pencacah satu status. Padanan `RDB List/CountMasterTipeSparepartManager-SQL.xml`, yang
-- mencacah `APPROVAL = '0'` untuk lencana antrean pada Inbox Manager.
--
-- Dipakai `claimpnc -periksa`, bukan oleh layar: layar sudah menerima barisnya dan dapat
-- menghitungnya sendiri tanpa perjalanan kedua.
SELECT COUNT(PART_SECTION_ID)
  FROM POOLDATA.GCNM_M_SPAREPART_TYPE
 WHERE TRIM(APPROVAL) = :1

-- name: type_count_all
--
-- Pencacah seluruh baris, tanpa memandang status. Dipakai `claimpnc -periksa` untuk
-- membedakan "tabelnya kosong" dari "tabelnya berisi tetapi tidak satu pun berstatus yang
-- dicari" — dua keadaan yang sangat berbeda artinya dan mudah tertukar.
SELECT COUNT(PART_SECTION_ID)
  FROM POOLDATA.GCNM_M_SPAREPART_TYPE

-- name: type_check_table
--
-- Membuktikan tabel beserta keempat kolomnya benar-benar ada dan dapat dibaca, tanpa
-- menarik satu baris pun.
--
-- Dipakai `claimpnc -periksa`. Bila kolomnya berbeda dari asumsi berkas ini, kegagalannya
-- muncul di sini — saat pemeriksaan dijalankan dengan sengaja — bukan saat petugas menekan
-- Simpan.
SELECT PART_SECTION_ID,
       PART_SECTION_NAME,
       PART_CATEGORY_ID,
       APPROVAL
  FROM POOLDATA.GCNM_M_SPAREPART_TYPE
 WHERE 1 = 0

-- name: type_count_unknown_status
--
-- Baris yang APPROVAL-nya di luar '0', '1', dan '2'.
--
-- Baris seperti itu TIDAK MUNCUL di satu pun dari ketiga tab — ia ada di basis data tetapi
-- tidak dapat dilihat maupun diputuskan siapa pun dari layar. Sistem lama punya cacat yang
-- sama persis, dan di sana pun tidak ada yang melaporkannya.
--
-- Dilaporkan `claimpnc -periksa` supaya keadaannya diketahui sebelum petugas melaporkan
-- "tipe saya hilang".
SELECT COUNT(PART_SECTION_ID)
  FROM POOLDATA.GCNM_M_SPAREPART_TYPE
 WHERE TRIM(APPROVAL) NOT IN ('0', '1', '2')
    OR APPROVAL IS NULL

-- name: type_count_duplicate_name
--
-- Banyaknya NAMA yang dipakai lebih dari satu baris.
--
-- Modul ini menolak nama ganda, tetapi tidak ada constraint unik yang menjaganya di basis
-- data (R-08) dan sistem lama pun tidak punya. Baris kembar yang sudah terlanjur ada akan
-- membuat pemeriksaan keunikan menolak penyimpanan yang sebenarnya sah — pengguna melihat
-- "nama sudah dipakai" atas baris yang sedang ia sunting sendiri.
--
-- Pencacahnya TIDAK mengelompokkan menurut kategori, meniru cakupan
-- `ValidationSparepartType` apa adanya: dua tipe bernama sama di kategori BERBEDA pun
-- terhitung kembar di sini, karena memang itulah yang akan ditolak saat disimpan.
--
-- Dilaporkan `claimpnc -periksa` supaya keadaan itu diketahui lebih dulu, bukan ditemukan
-- oleh petugas yang tidak dapat menyimpan pekerjaannya.
SELECT COUNT(*)
  FROM (SELECT UPPER(TRIM(PART_SECTION_NAME)) AS NAMA
          FROM POOLDATA.GCNM_M_SPAREPART_TYPE
         GROUP BY UPPER(TRIM(PART_SECTION_NAME))
        HAVING COUNT(*) > 1) KEMBAR

-- name: type_count_orphan_category
--
-- Baris tipe yang PART_CATEGORY_ID-nya tidak ada di tabel kategori.
--
-- Inilah baris yang di sistem lama HILANG dari layar karena inner join-nya; lihat banner di
-- kepala berkas ini. Di modul ini ia tetap terlihat dengan kolom Kategori kosong, dan
-- pencacah ini yang membuat jumlahnya diketahui sebelum selisihnya muncul pada uji
-- kesetaraan.
--
-- Baris berkategori KOSONG ikut terhitung: ia sama-sama tidak punya induk, dan sama-sama
-- tidak akan pernah muncul di sistem lama.
SELECT COUNT(T.PART_SECTION_ID)
  FROM POOLDATA.GCNM_M_SPAREPART_TYPE T
 WHERE NOT EXISTS (
       SELECT 1
         FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY C
        WHERE TRIM(CAST(C.PART_CATEGORY_ID AS VARCHAR(64)))
            = TRIM(CAST(T.PART_CATEGORY_ID AS VARCHAR(64))))

-- name: type_count_orphan_sparepart
--
-- Baris SPAREPART_HE yang TIPE_SPART-nya tidak ada di tabel ini.
--
-- Ia pemeriksaan dari arah sebaliknya: modul Master Sparepart sudah memeriksa hal yang sama
-- dari sisinya, dan diulang di sini karena modul INILAH yang kelak MENOLAK sebuah tipe —
-- dan penolakan tidak memutuskan tautan yang sudah ada.
--
-- Tipe yang ditolak tetap ditunjuk sparepart yang menautkannya sewaktu ia masih disetujui.
-- Sistem lama tidak memeriksanya, dan modul ini pun tidak menolak penolakan itu; yang
-- dikerjakannya hanyalah membuat keadaannya terlihat.
SELECT COUNT(S.ID)
  FROM POOLDATA.SPAREPART_HE S
 WHERE S.TIPE_SPART IS NOT NULL
   AND TRIM(S.TIPE_SPART) <> ''
   AND NOT EXISTS (
       SELECT 1
         FROM POOLDATA.GCNM_M_SPAREPART_TYPE T
        WHERE TRIM(CAST(T.PART_SECTION_ID AS VARCHAR(64))) = TRIM(S.TIPE_SPART))
