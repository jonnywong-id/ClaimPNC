-- Kueri modul Master Kategori Sparepart.
--
-- SATU tabel, dan aplikasi ini MENULISNYA: POOLDATA.GCNM_M_SPAREPART_CATEGORY. Tidak ada
-- tabel acuan, tidak ada sequence, dan tidak ada M_SITE_DATABASE — berbeda dari Master
-- Sparepart yang menyentuh lima objek sekaligus.
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
-- TABELNYA HANYA PUNYA TIGA KOLOM, DAN ITU SUDAH DIPASTIKAN
-- ============================================================================
--
-- Kesembilan rule Pega yang menyentuh tabel ini tidak satu pun menyebut kolom di luar
-- ketiga ini:
--
--   PART_CATEGORY_ID    BrowseSparepartCategoryClaimHE · BrowseMasterSparepartCategoryClaimHE
--                       BrowseSparepartCategoryClaimHE_sql · InsertMasterSparepartCategory_sql
--                       UpdateMasterSparepartCategory_sql2 · CountMasterKatSparepartManager
--                       BrowseSparepartTypeClaimHE_sql (join) · BrowseMasterSparepartTypeClaimHE_sql (join)
--   PART_CATEGORY_NAME  keenam rule di atas, ditambah ValidationSparepartCat
--   APPROVAL            BrowseSparepartCategoryClaimHE · BrowseMasterSparepartCategoryClaimHE
--                       InsertMasterSparepartCategory_sql · UpdateMasterSparepartCategory_sql2
--                       CountMasterKatSparepartManager
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
-- isinya:
--
--   select PART_CATEGORY_ID as "CityID", PART_CATEGORY_NAME as "City" ...
--
-- "City" untuk nama kategori suku cadang. Itu persis bentuk utang yang
-- 03-CURRENT-ARCHITECTURE.md §4.2 catat: nama kolom dipaksa cocok dengan property
-- klipboard Pega yang sudah ada. Alias itu TIDAK dibawa; kolomnya disebut nama aslinya.
--
--
-- ============================================================================
-- PART_CATEGORY_ID BERTIPE ANGKA, DAN ITU BUKAN TEBAKAN
-- ============================================================================
--
-- `RDB List/InsertMasterSparepartCategory_sql-SQL.xml` menerbitkannya dengan
-- `nvl(max(PART_CATEGORY_ID),0)+1`. Bila kolomnya VARCHAR2, `max(...)` akan mengembalikan
-- maksimum LEKSIKOGRAFIS — "9" lebih besar dari "10" — sehingga ID ke-11 akan bertabrakan
-- dengan yang sudah ada dan sistem lama akan rusak sejak baris kesepuluh. Ia tidak rusak,
-- jadi kolomnya angka.
--
-- Itu sebabnya `category_list` MENGURUTKAN berdasarkan kolomnya langsung dan bukan
-- berdasarkan teksnya — berbeda dari Master Sparepart, yang ID-nya memang teks.
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


-- name: category_list
--
-- Daftar satu tab. Padanan `RDB List/BrowseSparepartCategoryClaimHE-SQL.xml` beserta
-- kembarannya yang berparameter, `BrowseMasterSparepartCategoryClaimHE-SQL.xml`.
--
-- Keduanya membaca tabel yang sama dan hanya berbeda pada dari mana nilai penyaringnya
-- datang: yang pertama menanamnya sebagai '0', yang kedua menerimanya dari
-- `TempStatus.City`. Yang berparameter yang ditiru — ia yang melayani ketiga tab.
SELECT PART_CATEGORY_ID,
       PART_CATEGORY_NAME,
       APPROVAL
  FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY
 WHERE TRIM(APPROVAL) = :1
 ORDER BY PART_CATEGORY_ID

-- name: category_list_search
--
-- Sama dengan category_list, ditambah penyaring kata kunci pada nama.
--
-- Kueri TERSENDIRI, bukan satu kueri yang klausanya ditempel saat kata kuncinya ada.
-- Merangkai teks SQL adalah persis pola `{ASIS:...}` yang 03-CURRENT-ARCHITECTURE.md §4.5
-- catat sebagai celah injeksi, dan pemisahan ini membuat kedua bentuknya dapat dibaca utuh
-- di berkas ini.
--
-- UPPER di kedua sisi, bukan LOWER: rule validasi lamanya memakai `upper(...)`, dan memakai
-- pasangan yang sama membuat pencarian dan pemeriksaan keunikan tidak pernah berbeda soal
-- huruf besar-kecil.
--
-- Kata kuncinya sudah dibungkus tanda persen oleh pemanggil, bukan di sini: menempelkannya
-- di dalam teks SQL berarti merangkai nilai ke dalam pernyataan.
SELECT PART_CATEGORY_ID,
       PART_CATEGORY_NAME,
       APPROVAL
  FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY
 WHERE TRIM(APPROVAL) = :1
   AND UPPER(PART_CATEGORY_NAME) LIKE :2
 ORDER BY PART_CATEGORY_ID

-- name: category_get
--
-- Satu baris menurut kuncinya. Padanan
-- `RDB List/BrowseSparepartCategoryClaimHE_sql-SQL.xml`, yang dipakai
-- `Activity/SetMasterKategoriSparepart_act` untuk memuat baris ke form.
--
-- TRIM pada kuncinya dengan alasan yang sama seperti pada APPROVAL, ditambah satu lagi:
-- nilai ini datang dari jalur URL dan dari kolom SPAREPART_HE.KATEGORI_SPART yang bertipe
-- teks, sehingga spasi ujung benar-benar mungkin sampai ke sini.
SELECT PART_CATEGORY_ID,
       PART_CATEGORY_NAME,
       APPROVAL
  FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY
 WHERE TRIM(CAST(PART_CATEGORY_ID AS VARCHAR(64))) = :1

-- name: category_find_by_name
--
-- Padanan `RDB List/ValidationSparepartCat-SQL.xml`:
--
--   select PART_CATEGORY_NAME from POOLDATA.GCNM_M_SPAREPART_CATEGORY
--    where upper(PART_CATEGORY_NAME) = {TempValidateKategoriSparepart.CaseID}
--
-- PERHATIKAN APA YANG TIDAK ADA DI SANA: penyaring APPROVAL. Nama kategori yang pernah
-- DITOLAK tetap memblokir pemakaian nama itu. Ditiru apa adanya atas keputusan Work Owner
-- 2026-09-21 (P-5).
--
-- Yang DITAMBAHKAN hanyalah TRIM dan kolom ID pada hasilnya. TRIM supaya nama berspasi
-- ujung tidak lolos sebagai nama yang berbeda; ID supaya pemanggil dapat mengecualikan
-- baris yang sedang disunting — tanpa itu, menyimpan baris tanpa mengubah namanya akan
-- ditolak oleh dirinya sendiri.
--
-- FETCH FIRST 1 ROW ONLY, bukan ROWNUM: yang dibutuhkan hanya keberadaannya, dan bentuk ini
-- berjalan di kedua basis data (D-20). Ia sudah dipakai 35 rule pada sistem lama, jadi bukan
-- hal baru bagi tim.
SELECT PART_CATEGORY_ID,
       PART_CATEGORY_NAME,
       APPROVAL
  FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY
 WHERE UPPER(TRIM(PART_CATEGORY_NAME)) = :1
 ORDER BY PART_CATEGORY_ID
 FETCH FIRST 1 ROW ONLY

-- name: category_lock_table
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
-- Sistem lama tidak menjaganya sama sekali — `nvl(max(PART_CATEGORY_ID),0)+1` berada di
-- dalam satu INSERT tanpa penguncian apa pun. Work Owner memutuskan (2026-09-21) balapan
-- itu ditutup dengan penguncian, bukan dengan meminta sequence baru: bentuk ID-nya tetap
-- sama persis dengan Pega (P-5), dan modulnya tidak tertahan menunggu D-63.
--
-- # Harganya, dan kenapa ia terjangkau di SINI
--
-- EXCLUSIVE menahan penulisan lain atas tabel ini sampai transaksinya selesai. Itu harga
-- yang mahal pada tabel transaksi, dan hampir gratis di sini: kategori suku cadang
-- ditambahkan beberapa kali setahun, transaksinya memuat tiga pernyataan pendek, dan
-- PEMBACAAN tidak terhalang di kedua basis data.
--
-- Pemilihan modus itu disengaja. Oracle EXCLUSIVE dan PostgreSQL EXCLUSIVE keduanya
-- mengizinkan SELECT berjalan terus, sehingga dropdown Kategori pada layar Master Sparepart
-- tidak pernah tertahan oleh penambahan yang sedang berjalan.
--
-- # Yang TIDAK ditutupnya
--
-- Penguncian ini hanya mengikat penulis yang melewati basis data yang sama — termasuk
-- kedua instans aplikasi di belakang load balancer (D-27), dan termasuk Pega selama masa
-- paralel, karena kunci tabel ditegakkan basis data dan bukan aplikasi. Yang TIDAK
-- ditutupnya adalah baris kembar yang sudah terlanjur ada sebelum modul ini hidup.
-- Penutupnya constraint unik, dan itu menunggu DDL (R-08) serta prosedur perubahan skema
-- (D-63).
--
-- LOCK TABLE adalah satu-satunya pernyataan non-DML di seluruh modul ini. Bentuknya sama
-- persis di Oracle 19c dan PostgreSQL 17+, sehingga ia tidak melanggar D-20.
LOCK TABLE POOLDATA.GCNM_M_SPAREPART_CATEGORY IN EXCLUSIVE MODE

-- name: category_next_id
--
-- Menerbitkan ID berikutnya, meniru `InsertMasterSparepartCategory_sql` apa adanya kecuali
-- NVL yang diganti COALESCE (D-20).
--
-- WAJIB dijalankan setelah category_lock_table, di dalam transaksi yang sama. Di luar itu
-- nilainya dapat basi sebelum dipakai.
--
-- Tanpa FROM DUAL: agregat atas tabelnya sendiri sudah menyediakan baris hasil, sehingga
-- bentuk khas Oracle itu tidak diperlukan sama sekali di modul ini — berbeda dari Master
-- Sparepart, yang NEXTVAL-nya menuntutnya.
SELECT COALESCE(MAX(PART_CATEGORY_ID), 0) + 1
  FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY

-- name: category_insert
--
-- Ketiga kolomnya pada urutan yang sama dengan Pega.
--
-- ID diterbitkan lebih dulu oleh category_next_id dan dikirim sebagai parameter, bukan
-- ditanam sebagai sub-kueri seperti pada rule lama. Alasannya bukan selera: baris yang
-- tersimpan harus dikembalikan ke layar beserta ID-nya, dan sub-kueri di dalam VALUES tidak
-- memberi tahu pemanggil nilai apa yang terpakai.
INSERT INTO POOLDATA.GCNM_M_SPAREPART_CATEGORY
       (PART_CATEGORY_ID, PART_CATEGORY_NAME, APPROVAL)
VALUES (:1, :2, :3)

-- name: category_update
--
-- Padanan `RDB List/UpdateMasterSparepartCategory_sql2-SQL.xml`:
--
--   update POOLDATA.gcnm_m_sparepart_category
--      set PART_CATEGORY_NAME = {InputKategori.CITY_ID},
--          APPROVAL           = {InputKategori.LOGIN_APLIKASI}
--    where PART_CATEGORY_ID   = {InputKategori.ACCOUNT_ID}
--
-- PERHATIKAN NAMA PROPERTINYA. Nama kategori dikirim lewat `CITY_ID`, status persetujuan
-- lewat `LOGIN_APLIKASI`, dan kuncinya lewat `ACCOUNT_ID` — tiga property yang namanya
-- tidak ada hubungannya dengan apa yang dibawanya. Membaca rule lama berarti menelusuri
-- ketiganya sampai ke pemanggilnya untuk tahu isian mana yang mana; di sini ketiganya
-- disebut apa adanya.
--
-- ID TIDAK ikut ditulis — ia penyaring WHERE, dan berada di posisi parameter terakhir.
UPDATE POOLDATA.GCNM_M_SPAREPART_CATEGORY
   SET PART_CATEGORY_NAME = :1,
       APPROVAL           = :2
 WHERE TRIM(CAST(PART_CATEGORY_ID AS VARCHAR(64))) = :3

-- name: category_set_status
--
-- Menetapkan APPROVAL satu baris. Dipanggil sekali per baris di dalam SATU transaksi.
--
-- # BENTUKNYA DIREKONSTRUKSI, BUKAN DIBACA
--
-- `Activity/UpdateKategoriSparepart_act` menerima Param.ID dan Param.Approval lalu
-- memanggil `UpdateSparepartCategoryClaimHE_sql` — dan rule itu TIDAK ADA di antara 2.634
-- berkas export (R-16). Kategori juga tidak ikut `Activity/SetApprovalAllMaster`, yang
-- hanya melayani M_BENGKEL_HE, M_PANEL_HE, dan M_SPAREPART_HE.
--
-- Yang dapat dipastikan hanyalah: activity itu menyiapkan kunci dan status, lalu memanggil
-- sebuah rule UPDATE atas tabel ini. Bentuk di bawah mengikuti category_update yang
-- memang terbaca, dikurangi kolom nama yang tidak disentuh sebuah keputusan.
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
UPDATE POOLDATA.GCNM_M_SPAREPART_CATEGORY
   SET APPROVAL = :1
 WHERE TRIM(CAST(PART_CATEGORY_ID AS VARCHAR(64))) = :2

-- name: category_count_by_status
--
-- Pencacah satu status. Padanan `RDB List/CountMasterKatSparepartManager-SQL.xml`, yang
-- mencacah `APPROVAL = '0'` untuk lencana antrean pada Inbox Manager.
--
-- Dipakai `claimpnc -periksa`, bukan oleh layar: layar sudah menerima barisnya dan dapat
-- menghitungnya sendiri tanpa perjalanan kedua.
SELECT COUNT(PART_CATEGORY_ID)
  FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY
 WHERE TRIM(APPROVAL) = :1

-- name: category_count_all
--
-- Pencacah seluruh baris, tanpa memandang status. Dipakai `claimpnc -periksa` untuk
-- membedakan "tabelnya kosong" dari "tabelnya berisi tetapi tidak satu pun berstatus yang
-- dicari" — dua keadaan yang sangat berbeda artinya dan mudah tertukar.
SELECT COUNT(PART_CATEGORY_ID)
  FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY

-- name: category_check_table
--
-- Membuktikan tabel beserta ketiga kolomnya benar-benar ada dan dapat dibaca, tanpa menarik
-- satu baris pun.
--
-- Dipakai `claimpnc -periksa`. Bila kolomnya berbeda dari asumsi berkas ini, kegagalannya
-- muncul di sini — saat pemeriksaan dijalankan dengan sengaja — bukan saat petugas menekan
-- Simpan.
SELECT PART_CATEGORY_ID,
       PART_CATEGORY_NAME,
       APPROVAL
  FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY
 WHERE 1 = 0

-- name: category_count_unknown_status
--
-- Baris yang APPROVAL-nya di luar '0', '1', dan '2'.
--
-- Baris seperti itu TIDAK MUNCUL di satu pun dari ketiga tab — ia ada di basis data tetapi
-- tidak dapat dilihat maupun diputuskan siapa pun dari layar. Sistem lama punya cacat yang
-- sama persis, dan di sana pun tidak ada yang melaporkannya.
--
-- Dilaporkan `claimpnc -periksa` supaya keadaannya diketahui sebelum petugas melaporkan
-- "kategori saya hilang".
SELECT COUNT(PART_CATEGORY_ID)
  FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY
 WHERE TRIM(APPROVAL) NOT IN ('0', '1', '2')
    OR APPROVAL IS NULL

-- name: category_count_duplicate_name
--
-- Banyaknya NAMA yang dipakai lebih dari satu baris.
--
-- Modul ini menolak nama ganda, tetapi tidak ada constraint unik yang menjaganya di basis
-- data (R-08) dan sistem lama pun tidak punya. Baris kembar yang sudah terlanjur ada akan
-- membuat pemeriksaan keunikan menolak penyimpanan yang sebenarnya sah — pengguna melihat
-- "nama sudah dipakai" atas baris yang sedang ia sunting sendiri.
--
-- Dilaporkan `claimpnc -periksa` supaya keadaan itu diketahui lebih dulu, bukan ditemukan
-- oleh petugas yang tidak dapat menyimpan pekerjaannya.
SELECT COUNT(*)
  FROM (SELECT UPPER(TRIM(PART_CATEGORY_NAME)) AS NAMA
          FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY
         GROUP BY UPPER(TRIM(PART_CATEGORY_NAME))
        HAVING COUNT(*) > 1) KEMBAR

-- name: category_count_orphan_sparepart
--
-- Baris SPAREPART_HE yang KATEGORI_SPART-nya tidak ada di tabel ini.
--
-- Ia pemeriksaan dari arah sebaliknya: modul Master Sparepart sudah memeriksa hal yang sama
-- lewat `sparepart_count_orphan_category`, dan diulang di sini karena modul inilah yang
-- kelak MENOLAK sebuah kategori — dan penolakan tidak memutuskan tautan yang sudah ada.
--
-- Kategori yang ditolak tetap ditunjuk sparepart yang menautkannya sewaktu ia masih
-- disetujui. Sistem lama tidak memeriksanya, dan modul ini pun tidak menolak penolakan itu;
-- yang dikerjakannya hanyalah membuat keadaannya terlihat.
SELECT COUNT(S.ID)
  FROM POOLDATA.SPAREPART_HE S
 WHERE S.KATEGORI_SPART IS NOT NULL
   AND TRIM(S.KATEGORI_SPART) <> ''
   AND NOT EXISTS (
       SELECT 1
         FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY C
        WHERE TRIM(CAST(C.PART_CATEGORY_ID AS VARCHAR(64))) = TRIM(S.KATEGORI_SPART))
