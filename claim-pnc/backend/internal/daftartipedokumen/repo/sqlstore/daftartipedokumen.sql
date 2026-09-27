-- Kueri Daftar Tipe Dokumen: POOLDATA.LST_DOC_TYPE.
--
-- # Tabel mana yang dibaca, dan tabel mana yang ditulis
--
-- MEMBACA lewat view POOLDATA.V_LST_DOC_TYPE — sama seperti Pega. MENULIS ke tabel
-- dasarnya, POOLDATA.LST_DOC_TYPE.
--
-- Keduanya semula membaca tabel dasar, karena migrasi `0005_daftar_tipe_dokumen`
-- direncanakan memindahkan isi JSON_DATA menjadi kolom bernama. Migrasi itu belum
-- dijalankan, dan tabel dasarnya sampai sekarang hanya punya ID, OLD_ID, dan JSON_DATA —
-- sehingga setiap pembacaan gagal dengan `ORA-00904: "TGL_EDIT": invalid identifier` dan
-- layarnya tidak dapat dibuka sama sekali.
--
-- Keputusan Work Owner 2026-09-22: ikuti Pega dan baca langsung dari basis data, bukan
-- lewat JSON. View inilah yang Pega baca, dan ia sudah memaparkan TYPE_DOCUMENT,
-- STS_PROSES, USER_EDIT, dan TGL_EDIT sebagai kolom.
--
-- # Yang BELUM dapat dikerjakan, dan kenapa
--
-- MENYIMPAN masih menuntut migrasi 0005. Kolom TYPE_DOCUMENT, STS_PROSES, USER_EDIT, dan
-- TGL_EDIT tidak ada pada tabel dasarnya, dan view berisi ekspresi JSON tidak dapat
-- ditulisi. Hanya ada dua jalan: menulis JSON_DATA — yang justru dilarang keputusan di
-- atas — atau menjalankan migrasinya. Sampai itu terjadi, layarnya DAPAT DIBUKA dan
-- menampilkan isi, tetapi tombol Tambah dan Simpan akan menolak.
--
-- # Kenapa tidak lagi lewat PEGA_LST_DOC_TYPE
--
-- `D-02` menetapkan logika stored procedure naik ke Go dan aplikasi tidak memanggil
-- procedure. `D-68` menambahkan alasan yang lebih keras, dan procedure ini memperlihatkan
-- keduanya sekaligus:
--
--   1. KONTRAK GALATNYA TIDAK DAPAT DIPAKAI. Parameter keluarannya bernama `ErrMsg`,
--      tetapi pada jalur BERHASIL ia berisi kalimat "Data Sudah Disimpan dengan ID :
--      10001" (`PEGA_LST_DOC_TYPE.prc:22` dan `:35`). Pemanggil tidak dapat membedakan
--      berhasil dari gagal tanpa membaca teksnya.
--
--   2. KEGAGALANNYA TIDAK PERNAH SAMPAI KE PEMANGGIL SEBAGAI KEGAGALAN. Setiap blok
--      EXCEPTION mengisi ErrMsg lalu `RETURN` begitu saja (`:15-18`, `:25-29`, `:38-42`),
--      sehingga pemanggil menerima jawaban yang sama persis bentuknya seperti jalur
--      berhasil.
--
-- Keduanya tidak dibawa. Yang dibawa hanyalah ATURANNYA: bentuk ID, pilihan INSERT versus
-- UPDATE, dan kolom mana yang disentuh.
--
-- # Penyimpanan tidak lagi memakai JSON
--
-- Keputusan Work Owner 2026-09-21. Sistem lama menyimpan seluruh baris sebagai satu
-- dokumen JSON (`LST_DOC_TYPE.JSON_DATA`) lalu membongkarnya kembali lewat view. Migrasi
-- 0005 memindahkan isinya ke kolom dan mendefinisikan ulang view-nya, persis seperti yang
-- sudah dijalankan migrasi 0002 untuk M_STS_CLAIM dan migrasi 0004 untuk M_CAUSE_OF_LOSS.
--
-- # Satu tabel satu penulis (P-1)
--
-- Layar Daftar Tipe Dokumen adalah SATU-SATUNYA penulis LST_DOC_TYPE di sistem lama —
-- `RDB List/UpdateLstDocType-SQL.xml` adalah satu-satunya rule yang memanggil
-- PEGA_LST_DOC_TYPE, dan tidak ada rule lain yang menulis tabel itu. Memindahkan layarnya
-- karena itu memindahkan kepemilikan tabelnya secara utuh; Pega berubah menjadi pembaca
-- saja lewat V_LST_DOC_TYPE.
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding.

-- name: document_type_list
--
-- Urutannya mengikuti `Report Definition/BrowseLstDocType_RD-RD.xml`, yang menyetel
-- `pySortOrder=1` dan `pySortType=ASC` pada `.ID` — bukan pada TYPE_DOCUMENT dan bukan
-- pada TGL_EDIT, dua kolom yang letaknya berdekatan di berkas itu dan mudah tertukar.
--
-- Perlu disadari: ID adalah TEKS, sehingga urutannya leksikografis. Selama nomor urutnya
-- masih empat digit berpadding nol, urutan teks dan urutan penerbitan sama persis —
-- padding itulah yang membuatnya sama. Ia berhenti sama begitu nomor urut melewati 9999
-- (lihat daftartipedokumen.FormatID).
SELECT ID,
       TYPE_DOCUMENT,
       STS_PROSES
  FROM POOLDATA.V_LST_DOC_TYPE
 ORDER BY ID

-- name: document_type_get
SELECT ID,
       TYPE_DOCUMENT,
       STS_PROSES
  FROM POOLDATA.V_LST_DOC_TYPE
 WHERE ID = :1

-- name: document_type_site
--
-- Kode situs, bagian pertama dari setiap ID. Meniru `PEGA_LST_DOC_TYPE.prc:12` persis,
-- termasuk pembandingnya yang berupa TEKS '1' dan bukan angka.
--
-- Kueri inilah yang membuktikan modul ini per entitas: kode situs melekat pada basis data
-- tempat ia dijalankan, sehingga setiap entitas menerbitkan awalan ID-nya sendiri.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: document_type_next_sequence
--
-- Urutan yang SAMA dengan yang dipakai procedure lama (`PEGA_LST_DOC_TYPE.prc:21`),
-- supaya ID yang diterbitkan aplikasi ini melanjutkan deret yang sudah ada dan tidak
-- pernah bertabrakan dengan ID yang pernah diterbitkan Pega.
--
-- Perhatikan: nama urutannya `SET_LST_DOC_TYPE`, bukan `LST_DOC_TYPE_SEQ` seperti pola
-- penamaan modul tetangga. Ia dibaca apa adanya dari procedure, bukan ditebak dari pola.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada `ADR-0005`.
SELECT POOLDATA.SET_LST_DOC_TYPE.NEXTVAL
  FROM DUAL

-- name: document_type_insert
--
-- OLD_ID sengaja tidak diisi. Ia penomoran warisan yang hanya melekat pada baris lama;
-- procedure penyimpan pun tidak pernah menyentuhnya pada jalur INSERT, dan layar Pega
-- tidak menampilkannya sama sekali.
--
-- USER_EDIT dan TGL_EDIT diisi, dan itu meniru sistem lama:
-- `Activity/CNMInsertListDocumentType_act-Act.xml` menetapkan
-- `TempDcol.USER_EDIT := OperatorID.pyUserIdentifier` dan
-- `TempDcol.TGL_EDIT := @getCurrentTimeStamp()` sebelum menyimpan.
--
-- Waktunya datang sebagai parameter, BUKAN dari CURRENT_TIMESTAMP di dalam SQL. Dua
-- sebab: `F-5` menetapkan konversi dan pembacaan waktu hanya lewat satu seam, dan nilai
-- yang datang dari luar membuat penyimpanan dapat diuji secara deterministik.
INSERT INTO POOLDATA.LST_DOC_TYPE (ID, TYPE_DOCUMENT, STS_PROSES, USER_EDIT, TGL_EDIT)
VALUES (:1, :2, :3, :4, :5)

-- name: document_type_update
--
-- Hanya isian yang memang dapat diubah pengguna, ditambah kedua jejak simpan. ID tidak
-- pernah berubah — mengubahnya akan memutus dua master turunan
-- (`V_LST_DET_TYPE_DOC`, `LST_TYPE_DOC_BUSINESS`) beserta dokumen klaim yang sudah
-- merujuknya. Procedure lama pun hanya memakainya sebagai penyaring `WHERE`
-- (`PEGA_LST_DOC_TYPE.prc:34`).
--
-- OLD_ID juga tidak disentuh: ia jejak sejarah, bukan field yang dikelola.
UPDATE POOLDATA.LST_DOC_TYPE
   SET TYPE_DOCUMENT = :1,
       STS_PROSES    = :2,
       USER_EDIT     = :3,
       TGL_EDIT      = :4
 WHERE ID = :5

-- name: document_type_check_table
--
-- Memastikan tabelnya ada dan seluruh kolom yang dipakai modul ini dapat dibaca akun
-- aplikasi, tanpa mengambil satu baris pun. Dipakai mode periksa untuk membedakan tiga
-- sebab kegagalan yang tampak mirip tetapi perbaikannya berbeda jauh: migrasi 0005 belum
-- dijalankan DBA, tabelnya tidak ada di basis data entitas itu, atau akun aplikasi tidak
-- punya hak baca atasnya.
SELECT ID,
       TYPE_DOCUMENT,
       STS_PROSES,
       USER_EDIT,
       TGL_EDIT
  FROM POOLDATA.V_LST_DOC_TYPE
 WHERE 1 = 0
