-- Kueri modul Daftar Objek Dokumen (MENU_ID 43).
--
-- # SELURUH NAMA DI BERKAS INI DIBACA DARI KATALOG, BUKAN DITEBAK
--
-- Versi pertama berkas ini menebak tiga nama karena jalur simpan layar lama hilang dari
-- export (`R-16`). Pembacaan langsung ke katalog Oracle pada 2026-10-03 mengoreksi tiga
-- hal, dan dua di antaranya membuat penyimpanan GAGAL TOTAL:
--
--   tebakan                        kenyataan
--   ------------------------------ --------------------------------------------------
--   SET_LST_DOC_OBJ (urutan)       LST_DOC_OBJ_SEQ. SET_LST_DOC_OBJ memang ada, tetapi
--                                  ia PROCEDURE, bukan urutan
--   nomor urut 4 digit             5 digit — ID berbentuk CHAR(6), '1' + lpad(seq,5,'0')
--   LST_DOC_OBJ_BUSINESS dipakai   tabel itu ada tetapi KOSONG dan hanya punya dua kolom
--                                  (ID_DOC_OBJ, BISNISID); pemetaan yang sesungguhnya
--                                  ada di dalam JSON_DATA
--
-- # Di mana isi master ini SEBENARNYA tinggal
--
-- Di `JSON_DATA`, bukan di kolom. Terbukti dari data: pada seluruh 12 baris yang ada,
-- `KET_DOC_OBJ` **kosong** sementara `JSON_DATA` terisi. Bentuknya:
--
--   {"ID":"100766","LIST_LBU_ID":[{"ID":"10027"},{"ID":"10045"}],"KET_DOC_OBJ":"STOCK"}
--
-- Dan `POOLDATA.V_LST_DOC_OBJ_BISNIS` membongkar jalur `$.LIST_LBU_ID[*].ID` yang sama —
-- 44 baris pemetaan, dan seluruh 44 ID-nya ketemu di POOLDATA.BUSINESS.
--
-- Itu sebabnya layar menampilkan 12 baris TANPA NAMA sebelum perbaikan ini: view
-- POOLDATA.V_LST_DOC_OBJ hanya membaca kolom, dan kolomnya memang belum pernah diisi.
--
-- # Yang ditulis modul ini: JSON_DATA **dan** kolomnya
--
-- `JSON_DATA` ditulis karena di situlah isi yang sesungguhnya, dan karena
-- V_LST_DOC_OBJ_BISNIS — yang dibaca rule lain selama masa paralel — membacanya.
--
-- `KET_DOC_OBJ` ikut ditulis supaya V_LST_DOC_OBJ berhenti mengembalikan NULL. Itu
-- memperbaiki keadaan hari ini, bukan merusaknya: kolom itu kosong untuk SEMUA pembaca
-- sekarang.
--
-- Ini mengoreksi keputusan §23.1 butir 2, yang menetapkan "tulis ke kolom, bukan JSON".
-- Keputusan itu diambil atas premis bahwa kolomlah yang hidup. Premisnya salah.
--
-- # Yang TIDAK disentuh
--
--   POOLDATA.LST_DOC_OBJ_BUSINESS   tabel kosong yang tidak dibaca siapa pun. Menulisnya
--                                   berarti membuat sumber kebenaran KEDUA yang tidak ada
--                                   pembacanya
--   POOLDATA.BUSINESS               milik GISFW, HANYA DIBACA (`D-03`)
--
-- # Kenapa dokumen JSON-nya disusun di Go, bukan di SQL
--
-- Supaya tidak ada satu pun fungsi JSON pada jalur TULIS. `JSON_OBJECT` dan `JSON_ARRAY`
-- ada di Oracle dan PostgreSQL 17+ dengan sintaks yang berbeda-beda cukup jauh; menyusunnya
-- di Go membuat kueri tulis menjadi parameter biasa. Pada jalur BACA, `JSON_VALUE` dipakai
-- dan ia memang portabel (`D-24`).
--
-- Empat aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, LPAD, dan TO_CHAR (`D-20`).
--   4. Tanpa pemanggilan stored procedure (`D-02`).


-- name: document_object_list
--
-- Sumber grid layar: kolom "ID" dan kolom "Daftar Objek Dokumen".
--
-- COALESCE, bukan salah satu saja. Hari ini seluruh nama ada di JSON dan kolomnya kosong;
-- baris yang ditulis modul ini mengisi keduanya. Membaca kolomnya saja menampilkan 12
-- baris kosong; membaca JSON-nya saja mengabaikan kolom yang sudah benar. Keduanya dibaca,
-- dan kolom didahulukan karena ia yang akan selalu benar untuk baris baru.
--
-- OLD_ID ikut dibaca meski grid tidak menampilkannya: lapisan data mengikuti Report
-- Definition, yang memuatnya.
SELECT ID,
       COALESCE(KET_DOC_OBJ, JSON_VALUE(JSON_DATA, '$.KET_DOC_OBJ')),
       OLD_ID
  FROM POOLDATA.LST_DOC_OBJ
 ORDER BY ID

-- name: document_object_get
--
-- Satu baris untuk dimuat ke form sunting, beserta dokumen JSON-nya.
--
-- JSON_DATA ikut diambil karena pemetaan bisnisnya ada di dalamnya, dan karena penyimpanan
-- harus menyusun ulang dokumen itu secara utuh — bagian yang tidak dikenal modul ini pun
-- harus ikut terbawa, bukan hilang.
SELECT ID,
       COALESCE(KET_DOC_OBJ, JSON_VALUE(JSON_DATA, '$.KET_DOC_OBJ')),
       OLD_ID,
       JSON_DATA
  FROM POOLDATA.LST_DOC_OBJ
 WHERE ID = :1

-- name: document_object_site
--
-- Kode situs, bagian pertama dari setiap ID. Meniru POOLDATA.PEGA_LST_DOC_OBJ — procedure
-- yang BENAR-BENAR ADA di basis data meski hilang dari export — termasuk pembandingnya
-- yang berupa teks '1' dan bukan angka.
--
-- Nilainya "1" pada basis data ini, sehingga ID menjadi '1' + lima digit = CHAR(6).
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: document_object_next_sequence
--
-- Urutan yang SAMA dengan yang dipakai procedure lama, sehingga ID yang diterbitkan
-- aplikasi ini melanjutkan deret yang sudah ada dan tidak pernah bertabrakan.
--
-- Namanya LST_DOC_OBJ_SEQ. Jangan tertukar dengan POOLDATA.SET_LST_DOC_OBJ, yang namanya
-- mirip tetapi merupakan PROCEDURE penyusun senarai bisnis — bukan urutan. Tertukar
-- sekali, dan itu salah satu sebab penyimpanan gagal pada versi pertama modul ini.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada `ADR-0005`.
SELECT POOLDATA.LST_DOC_OBJ_SEQ.NEXTVAL
  FROM DUAL

-- name: document_object_insert
--
-- OLD_ID sengaja tidak diisi: penomoran lama melekat pada baris warisan — ia diisi
-- POOLDATA.PROCESS_LST_DOC_OBJ saat memindahkan data dari GENERAL.LST_DOC_OBJ@ASMD — dan
-- tidak pernah diberikan pada baris baru.
INSERT INTO POOLDATA.LST_DOC_OBJ (ID, KET_DOC_OBJ, JSON_DATA)
VALUES (:1, :2, :3)

-- name: document_object_update
--
-- ID hanya menyaring, tidak pernah ikut di-SET — sama seperti cabang UPDATE pada procedure
-- lama. Ia dirujuk POOLDATA.LST_TYPE_DOC_BUSINESS.OBJECT_DOC_ID pada data yang sudah
-- berjalan, dan mengubahnya akan memutus setiap baris yang bernaung di bawahnya.
--
-- OLD_ID juga tidak disentuh: ia jejak sejarah.
UPDATE POOLDATA.LST_DOC_OBJ
   SET KET_DOC_OBJ = :1,
       JSON_DATA   = :2
 WHERE ID = :3

-- name: business_list
--
-- Daftar bisnis untuk saran isian di layar, dan sumber pencocokan nama ke ID.
--
-- Asal kolomnya: `RDB List/GetLBUID_SQL-SQL.xml` yang membaca `BUSINESS` dengan `ID` dan
-- `NOTE`. Tabel ini milik GISFW dan HANYA DIBACA (`D-03`).
--
-- Diurutkan menurut NOTE, bukan ID: pengguna mencari bisnis dengan namanya, dan urutan kode
-- tidak berarti apa-apa baginya.
SELECT ID,
       NOTE
  FROM POOLDATA.BUSINESS
 ORDER BY NOTE

-- name: document_object_check_table
--
-- Memastikan tabel induk dapat dibaca akun aplikasi, dengan kolom yang BENAR-BENAR dipakai
-- — termasuk JSON_DATA, tempat isinya tinggal. Tanpa mengambil satu baris pun, sehingga
-- aman dijalankan terhadap produksi.
SELECT ID,
       KET_DOC_OBJ,
       OLD_ID,
       JSON_DATA
  FROM POOLDATA.LST_DOC_OBJ
 WHERE 1 = 0

-- name: business_check_table
--
-- Memastikan master bisnis milik GISFW dapat dibaca akun aplikasi.
SELECT ID,
       NOTE
  FROM POOLDATA.BUSINESS
 WHERE 1 = 0
