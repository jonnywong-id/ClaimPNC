-- Kueri modul Daftar Detail Tipe Dokumen (MENU_ID 41).
--
-- ============================================================================
-- BACA BAGIAN INI SEBELUM MENGUBAH SATU BARIS PUN DI BAWAHNYA.
-- ============================================================================
--
-- # BERKAS INI ADALAH SATU-SATUNYA TEMPAT NAMA OBJEK BASIS DATA DISEBUT
--
-- Itu disengaja: bila skemanya kelak berubah, yang disunting hanya berkas ini — tidak ada
-- satu pun nama tabel yang tercecer di dalam kode Go.
--
-- # DIVERIFIKASI LANGSUNG KE KATALOG ORACLE 2026-09-23
--
-- Katalog dibaca langsung (`ALL_TAB_COLUMNS`, `ALL_VIEWS`) karena mode periksa menolak
-- jalur tulis dengan `ORA-00904: "TYPE_DOCUMENT": invalid identifier`. Hasilnya:
--
--   POOLDATA.LST_DET_TYPE_DOC   ADA, dan kolomnya SUDAH LENGKAP:
--       ID CHAR(5) · OLD_ID CHAR(3) · JSON_DATA CLOB · DETAIL_DOCUMENT · DOC_TYPE_ID(100)
--       USER_EDIT · STS_INSURED(100) · DOC_COL_ID(1000) · OBJ_DOC · DOC_COL_INFO · RISK
--       OBJ_DOC_DESC · TGL_EDIT            (seluruh sisanya VARCHAR2 4000)
--
--   POOLDATA.V_LST_DET_TYPE_DOC adalah SELECT KOLOM biasa — bukan JSON, bukan join.
--   `TYPE_DOCUMENT` TIDAK ADA di dalamnya; ia di-LEFT JOIN sendiri dari
--   POOLDATA.V_LST_DOC_TYPE, persis yang dilakukan modul MENU_ID 42. Inilah yang membuat
--   layarnya gagal memuat sebelum diperbaiki.
--
-- # Aturan per lini bisnis TIDAK ditangani modul ini
--
-- Koreksi Work Owner 2026-10-03: grid Lini Bisnis tidak ada di layar Pega, dan karena itu
-- tidak ada di sini. Seluruh kueri terhadap `POOLDATA.LST_DET_TYPE_DOC_BISNIS` dan
-- `POOLDATA.BUSINESS` DICABUT dari berkas ini.
--
-- Dua akibatnya, keduanya disengaja:
--
--   1. Permintaan pembuatan tabel anak kepada DBA (migrasi 0009) menjadi TIDAK PERLU.
--      Modul ini kini berjalan penuh di atas objek yang sudah ada.
--   2. Aturan per lini bisnis tetap hidup di `LST_DET_TYPE_DOC.JSON_DATA` dan tetap
--      dimiliki Pega selama masa paralel. Modul ini TIDAK MENYENTUH kolom itu —
--      tidak membaca, tidak menulis, tidak mengosongkan — sehingga menyimpan dari layar
--      ini tidak menghilangkan satu pun aturan bisnis yang sudah ada.
--
-- # JSON_DATA tidak disentuh sama sekali
--
-- `TGL_EDIT` bertipe VARCHAR2, bukan DATE, dan isinya berformat Pega:
--
--       "20231030T075651.051 GMT"
--
-- Menulis `time.Time` ke sana akan menghasilkan bentuk yang berbeda dari seluruh baris
-- yang sudah ada. Pemformatannya ada di daftardetailtipedokumen.go, satu tempat saja.
--
-- # Enam kolom ditulis meski tidak punya isian di layar
--
-- `STS_INSURED`, `DOC_COL_ID`, `DOC_COL_INFO`, `OBJ_DOC`, `OBJ_DOC_DESC`, dan `RISK`
-- tidak disunting dari layar ini — layar Pega yang berjalan hanya meminta ID Tipe Dokumen
-- dan Detail Dokumen. Keenamnya tetap ikut ditulis dengan NILAI LAMANYA, bukan dikosongkan:
-- mengosongkannya akan menghapus isi kolom yang petugas tidak pernah diberi kesempatan
-- mengubahnya.
--
-- # Satu tabel satu penulis (P-1)
--
-- Layar ini adalah SATU-SATUNYA penulis POOLDATA.LST_DET_TYPE_DOC di sistem lama —
-- `RDB List/UpdateDetTypeDoc-SQL.xml` satu-satunya rule yang memanggil
-- PEGA_LST_DET_TYPE_DOC, dan tidak ada rule lain yang menyentuh tabelnya.
--
-- POOLDATA.V_LST_DOC_TYPE HANYA DIBACA di sini; ia dimiliki modul MENU_ID 40.
--
-- # Yang membaca view induk, dan kenapa itu penting
--
-- 34 rule Pega membacanya, di antaranya SELURUH jalur validasi unggah dokumen:
-- `ValidationUploadDocument_act`, `ValidationUploadRegister`, `RequiredDocument_act`,
-- `RequiredDocPA`, `InsertDokumenPNC`, ditambah arsip dokumen dan pencarian klaim.
-- Selama masa paralel, apa pun yang ditulis modul ini WAJIB terbaca oleh view itu — bila
-- tidak, dokumen yang diminta pada klaim berhenti muncul TANPA satu pun galat.
--
-- Empat aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (`D-20`).
--   4. Tanpa pemanggilan stored procedure (`D-02`).


-- name: detail_list
--
-- Grid layar: ID, Tipe Dokumen, dan Detail Dokumen — tiga kolom, sama dengan
-- `Section/BrowseListDetailTypeDocument-Section.xml`.
--
-- Keenam kolom lain ikut dibaca meski tidak ditampilkan: nilainya dibutuhkan saat
-- menyimpan, supaya kolom yang tidak disunting dapat ditulis kembali apa adanya.
--
-- Diurutkan menurut ID, mengikuti `BrowseVLstDetTypeDoc_RD-RD.xml`.
--
-- Batas `pyMaxRecords=500` milik Report Definition itu tidak ditiru. Ia bukan aturan
-- bisnis melainkan pemotongan senyap (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2), dan
-- menirunya berarti menyembunyikan baris yang benar-benar ada dari petugas yang sedang
-- menyuntingnya.
SELECT a.ID,
       a.DOC_TYPE_ID,
       t.TYPE_DOCUMENT,
       a.DETAIL_DOCUMENT,
       a.STS_INSURED,
       a.DOC_COL_ID,
       a.DOC_COL_INFO,
       a.OBJ_DOC,
       a.OBJ_DOC_DESC,
       a.RISK
  FROM POOLDATA.V_LST_DET_TYPE_DOC a
  LEFT JOIN POOLDATA.V_LST_DOC_TYPE t ON t.ID = a.DOC_TYPE_ID
 ORDER BY a.ID

-- name: detail_get
--
-- Satu baris untuk dimuat ke form sunting. Bentuk kolomnya sama persis dengan
-- detail_list supaya keduanya dapat dibaca satu fungsi pemindai — bila keduanya berbeda,
-- satu perubahan kolom harus diingat di dua tempat.
SELECT a.ID,
       a.DOC_TYPE_ID,
       t.TYPE_DOCUMENT,
       a.DETAIL_DOCUMENT,
       a.STS_INSURED,
       a.DOC_COL_ID,
       a.DOC_COL_INFO,
       a.OBJ_DOC,
       a.OBJ_DOC_DESC,
       a.RISK
  FROM POOLDATA.V_LST_DET_TYPE_DOC a
  LEFT JOIN POOLDATA.V_LST_DOC_TYPE t ON t.ID = a.DOC_TYPE_ID
 WHERE a.ID = :1

-- name: detail_site
--
-- Kode situs, bagian pertama dari setiap ID. Meniru
-- `Database/PEGA_LST_DET_TYPE_DOC.prc:11` persis, termasuk pembandingnya yang berupa teks
-- '1' dan bukan angka.
--
-- Inilah yang membuat tiap entitas menerbitkan awalan ID-nya sendiri, dan karena itulah
-- modul ini portal-aware.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: detail_next_sequence
--
-- Nomor urut ID. Namanya DIBACA dari `Database/PEGA_LST_DET_TYPE_DOC.prc:19`, bukan
-- diturunkan dari nama tabel.
--
-- Memakai urutan yang SAMA dengan procedure lama membuat ID yang diterbitkan aplikasi ini
-- melanjutkan deret yang sudah ada dan tidak pernah bertabrakan dengan ID yang pernah
-- diterbitkan Pega.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada `ADR-0005`, satu-satunya tempat
-- lain yang dibenarkan memuat percabangan dialek.
--
-- Pemformatan nomornya menjadi ID dikerjakan di Go, bukan dengan LPAD di sini: LPAD dan
-- TO_CHAR termasuk yang dilarang `09-DATABASE-STRATEGY.md` §4 karena keduanya mengikat
-- kueri pada dialek Oracle.
SELECT POOLDATA.LST_DET_TYPE_DOC_SEQ.NEXTVAL
  FROM DUAL

-- name: detail_insert
--
-- Sebelas kolom. JSON_DATA sengaja TIDAK diisi — penyimpanan langsung ke kolom, bukan ke
-- dokumen JSON (keputusan Work Owner 2026-09-23).
--
-- OLD_ID tidak diisi: penomoran lama melekat pada baris warisan dan tidak pernah
-- diberikan pada baris baru.
INSERT INTO POOLDATA.LST_DET_TYPE_DOC
       (ID, DOC_TYPE_ID, DETAIL_DOCUMENT, STS_INSURED, DOC_COL_ID, DOC_COL_INFO,
        OBJ_DOC, OBJ_DOC_DESC, RISK, TGL_EDIT, USER_EDIT)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11)

-- name: detail_update
--
-- ID tidak pernah berubah — ia dirujuk `LST_TYPE_DOC_BUSINESS.DOC_TYPE_DT_ID` milik modul
-- MENU_ID 42. Ia hanya dipakai sebagai penyaring WHERE.
--
-- OLD_ID tidak ikut diubah: ia jejak sejarah.
--
-- JSON_DATA tidak ikut diubah, dan itu yang menjaga aturan per lini bisnis milik Pega
-- tetap utuh setelah baris ini disunting dari sini.
UPDATE POOLDATA.LST_DET_TYPE_DOC
   SET DOC_TYPE_ID = :1,
       DETAIL_DOCUMENT = :2,
       STS_INSURED = :3,
       DOC_COL_ID = :4,
       DOC_COL_INFO = :5,
       OBJ_DOC = :6,
       OBJ_DOC_DESC = :7,
       RISK = :8,
       TGL_EDIT = :9,
       USER_EDIT = :10
 WHERE ID = :11

-- name: document_type_choice_list
--
-- Pilihan isian ID Tipe Dokumen.
--
-- Menggantikan dropdown pada `Section/BrowseListDetailTypeDocument-Section.xml`, yang
-- membaca kelas `ASM-FW-GCNMFW-Int-V_LST_DOC_TYPE` dengan `.ID` sebagai nilai dan
-- `.TYPE_DOCUMENT` sebagai tampilannya.
--
-- HANYA SELECT. Tabel ini dimiliki modul Daftar Tipe Dokumen, MENU_ID 40 (`P-1`).
SELECT ID,
       TYPE_DOCUMENT
  FROM POOLDATA.V_LST_DOC_TYPE
 ORDER BY TYPE_DOCUMENT

-- name: detail_check_table
--
-- Memastikan view induk ada dan kolomnya dapat dibaca akun aplikasi, tanpa mengambil satu
-- baris pun. Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip
-- tetapi perbaikannya berbeda jauh: objeknya tidak ada di basis data entitas itu, versus
-- akun aplikasi tidak punya hak baca atasnya.
--
-- Yang TIDAK diperiksa: urutan penerbit ID. Memeriksanya berarti MENGHABISKAN satu nomor
-- — efek samping yang tidak pantas dimiliki mode periksa.
SELECT a.ID,
       a.DOC_TYPE_ID,
       t.TYPE_DOCUMENT,
       a.DETAIL_DOCUMENT,
       a.STS_INSURED,
       a.DOC_COL_ID,
       a.DOC_COL_INFO,
       a.OBJ_DOC,
       a.OBJ_DOC_DESC,
       a.RISK
  FROM POOLDATA.V_LST_DET_TYPE_DOC a
  LEFT JOIN POOLDATA.V_LST_DOC_TYPE t ON t.ID = a.DOC_TYPE_ID
 WHERE 1 = 0

-- name: detail_write_check_table
--
-- Memastikan TABEL DASAR induk benar-benar punya kolom yang ditulis modul ini.
--
-- Terpisah dari pemeriksaan view dengan sengaja: bila tabelnya ternyata masih hanya
-- (ID, JSON_DATA) seperti yang dilakukan procedure lama, kueri ini gagal dengan
-- ORA-00904 — dan kegagalan itu terbaca di mode periksa, sebelum pengguna pertama
-- menekan Simpan.
SELECT ID,
       DOC_TYPE_ID,
       DETAIL_DOCUMENT,
       STS_INSURED,
       DOC_COL_ID,
       DOC_COL_INFO,
       OBJ_DOC,
       OBJ_DOC_DESC,
       RISK,
       TGL_EDIT,
       USER_EDIT
  FROM POOLDATA.LST_DET_TYPE_DOC
 WHERE 1 = 0

-- name: document_type_check_table
SELECT ID,
       TYPE_DOCUMENT
  FROM POOLDATA.V_LST_DOC_TYPE
 WHERE 1 = 0
