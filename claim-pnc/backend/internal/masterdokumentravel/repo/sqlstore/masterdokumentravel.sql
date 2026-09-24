-- Kueri master Dokumen Travel: POOLDATA.M_DOCTRAVEL.
--
-- # Kenapa tidak lagi lewat DOCTRAVEL_CVG
--
-- `D-02` menetapkan logika stored procedure naik ke Go dan aplikasi tidak memanggil
-- procedure. `D-68` menambahkan alasan yang lebih keras, dan procedure ini adalah
-- contoh bukunya:
--
--   1. KONTRAK GALATNYA TIDAK DAPAT DIPAKAI. Parameter keluarannya bernama `ErrMsg`,
--      tetapi pada jalur BERHASIL ia berisi kalimat "Data Sudah Disimpan dengan ID :
--      100001" (`DOCTRAVEL_CVG.prc:24`). Pemanggil tidak dapat membedakan berhasil dari
--      gagal tanpa membaca teksnya.
--
--   2. PARAMETER KELUARANNYA TERTUKAR DI SISI PEGA. `UpdateMstDocTravel-SQL.xml`
--      memetakan `{OutputData.DOCID out}` ke parameter `format` dan
--      `{OutputData.NAMADOKUMEN out}` ke `ErrMsg`. Padahal `format` TIDAK PERNAH DIISI
--      pada jalur berhasil — ia hanya di-set `null` di handler galat terluar
--      (`:51`). Akibatnya `TempMstDocTravel.pyNote := OutputData.DOCID`
--      (`CNMInsertMstDocTravel_act` langkah terakhir) SELALU kosong, dan pesan
--      "Data Sudah Disimpan dengan ID ..." mendarat di kolom judul dokumen.
--
--   3. IA COMMIT SENDIRI. `:25` dan `:39` melakukan COMMIT di dalam cabangnya
--      masing-masing, sementara ROLLBACK terluar (`:53`) berjalan SESUDAH commit itu
--      sehingga tidak memulihkan apa pun.
--
-- Ketiganya tidak dibawa. Yang dibawa hanyalah ATURANNYA: bentuk DOCID, pilihan
-- INSERT versus UPDATE, dan kolom mana yang disentuh.
--
-- # Satu tabel satu penulis (P-1)
--
-- Layar Master Dokumen Travel adalah SATU-SATUNYA penulis M_DOCTRAVEL di sistem lama —
-- `UpdateMstDocTravel-SQL.xml` adalah satu-satunya rule yang memanggil DOCTRAVEL_CVG,
-- dan tidak ada rule lain yang menulis tabel itu. Memindahkan layarnya karena itu
-- memindahkan kepemilikan tabelnya secara utuh; Pega berubah menjadi pembaca saja.
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding.

-- name: travel_document_list
--
-- Urutannya mengikuti `BrowseMstDocTravel_RD-RD.xml`: DOCID menaik.
--
-- Perlu disadari: DOCID adalah TEKS, sehingga urutannya leksikografis. Selama nomor
-- urutnya masih lima digit berpadding nol, urutan teks dan urutan penerbitan sama
-- persis — padding itulah yang membuatnya sama. Ia berhenti sama begitu nomor urut
-- melewati 99999 (lihat masterdokumentravel.FormatID).
SELECT DOCID,
       NAMADOKUMEN
  FROM POOLDATA.M_DOCTRAVEL
 ORDER BY DOCID

-- name: travel_document_get
SELECT DOCID,
       NAMADOKUMEN
  FROM POOLDATA.M_DOCTRAVEL
 WHERE DOCID = :1

-- name: travel_document_site
--
-- Kode situs, bagian pertama dari setiap DOCID. Meniru `DOCTRAVEL_CVG.prc:12` persis,
-- termasuk pembandingnya yang berupa TEKS '1' dan bukan angka.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: travel_document_next_sequence
--
-- Urutan yang SAMA dengan yang dipakai procedure lama (`DOCTRAVEL_CVG.prc:20`), supaya
-- DOCID yang diterbitkan aplikasi ini melanjutkan deret yang sudah ada dan tidak pernah
-- bertabrakan dengan DOCID yang pernah diterbitkan Pega.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada `ADR-0005`.
SELECT POOLDATA.DOCTRAVEL_SEQ.NEXTVAL
  FROM DUAL

-- name: travel_document_insert
--
-- Kedua kolom diisi. Tabel ini memang hanya punya dua kolom — tidak ada penanda aktif,
-- tidak ada jejak siapa yang menambahkan, dan tidak ada waktu pembuatan. Itu keadaan
-- tabel warisan apa adanya; menambah kolom adalah perubahan skema tersendiri yang
-- menempuh `D-63`, bukan sesuatu yang layak diselipkan modul ini.
INSERT INTO POOLDATA.M_DOCTRAVEL (DOCID, NAMADOKUMEN)
VALUES (:1, :2)

-- name: travel_document_update
--
-- Hanya NAMADOKUMEN yang diubah. DOCID tidak pernah berubah — mengubahnya akan memutus
-- baris V_LST_DOC_TRAVEL dan dokumen klaim yang sudah merujuknya. Procedure lama pun
-- hanya memakainya sebagai penyaring WHERE (`DOCTRAVEL_CVG.prc:37`).
UPDATE POOLDATA.M_DOCTRAVEL
   SET NAMADOKUMEN = :1
 WHERE DOCID = :2

-- name: travel_document_check_table
--
-- Memastikan tabelnya ada dan kedua kolomnya dapat dibaca akun aplikasi, tanpa
-- mengambil satu baris pun. Dipakai mode periksa untuk membedakan dua sebab kegagalan
-- yang tampak mirip tetapi perbaikannya berbeda jauh: tabel tidak ada di basis data
-- entitas itu, versus akun aplikasi tidak punya hak baca atasnya.
SELECT DOCID,
       NAMADOKUMEN
  FROM POOLDATA.M_DOCTRAVEL
 WHERE 1 = 0
