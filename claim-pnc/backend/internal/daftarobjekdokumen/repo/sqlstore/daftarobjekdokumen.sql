-- Kueri modul Daftar Objek Dokumen (MENU_ID 43).
--
-- # BERKAS INI ADALAH SATU-SATUNYA TEMPAT NAMA OBJEK BASIS DATA DISEBUT
--
-- Itu disengaja, dan alasannya ada di bawah: sebagian nama di sini adalah DUGAAN. Bila
-- jawaban DBA berbeda, yang berubah hanya berkas ini — tidak ada satu pun nama tabel yang
-- tercecer di dalam kode Go.
--
-- # Mana yang TERBACA dari export, dan mana yang DUGAAN
--
--   POOLDATA.V_LST_DOC_OBJ        TERBACA  `Report Definition/BrowseVLstDocObj_RD-RD.xml`
--                                          menyebut kelas ASM-FW-GCNMFW-Int-V_LST_DOC_OBJ
--                                          dengan kolom ID, KET_DOC_OBJ, dan OLD_ID
--   POOLDATA.M_SITE_DATABASE      TERBACA  `Database/PEGA_LST_DOC_TYPE.prc:11`
--   POOLDATA.BUSINESS             TERBACA  `RDB List/GetLBUID_SQL-SQL.xml`
--
--   POOLDATA.LST_DOC_OBJ          DUGAAN   diturunkan dari nama view-nya, mengikuti
--                                          pasangan V_LST_DOC_TYPE -> LST_DOC_TYPE yang
--                                          terbukti di `Database/PEGA_LST_DOC_TYPE.prc`
--   POOLDATA.SET_LST_DOC_OBJ      DUGAAN   diturunkan dari pola SET_LST_DOC_TYPE pada
--                                          procedure yang sama
--   POOLDATA.LST_DOC_OBJ_BUSINESS DUGAAN   TABEL BARU — belum ada di basis data mana pun;
--                                          bentuknya dirancang di migrasi 0008
--
-- Ketiga dugaan itu ADA sebabnya, bukan karangan: jalur simpan layar lama menunjuk
-- `CNMInsertLstDocObj_act` dan `SetsLstDocObjValue_act`, dan **keduanya hilang dari
-- export** (`R-16`). Tidak ada pula `PEGA_LST_DOC_OBJ.prc` di `Database/`. APA yang
-- disimpan terbaca lengkap dari form dan Report Definition-nya; yang tidak terbaca hanya
-- KE MANA.
--
-- Migrasi `0008_daftar_objek_dokumen.up.sql` ditulis sebagai DAFTAR PERTANYAAN untuk DBA,
-- bukan sekadar DDL. Bagian 0-nya menanyakan tepat ketiga nama di atas.
--
-- # Membaca lewat view, menulis ke tabel dasar
--
-- Sama seperti modul Daftar Tipe Dokumen. Yang dibaca adalah objek yang NAMANYA PASTI
-- (view-nya terbukti dari Report Definition); yang ditulis adalah tabel dasarnya, yang
-- namanya dugaan. Selama migrasi 0008 belum dijalankan DBA, keduanya dapat tidak sepakat —
-- dan mode `-periksa` melaporkannya, bukan membiarkannya terlihat sebagai data hilang.
--
-- # Kenapa tidak lewat procedure
--
-- `D-02` menetapkan logika procedure naik ke Go. `D-68` menambahkan alasan yang lebih
-- keras, dan rumpun ini memperagakannya: `PEGA_LST_DOC_TYPE.prc` memakai parameter keluaran
-- bernama `ErrMsg` yang pada jalur BERHASIL berisi kalimat "Data Sudah Disimpan dengan ID :
-- 1011", sehingga pemanggil tidak dapat membedakan berhasil dari gagal tanpa membaca teks.
--
-- Empat aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (`D-20`).
--   4. Tanpa pemanggilan stored procedure (`D-02`).


-- name: document_object_list
--
-- Sumber grid layar (`Section/BrowseDocumentObject-Section.xml`): kolom "ID" dan kolom
-- "Daftar Objek Dokumen".
--
-- OLD_ID ikut dibaca meski grid tidak menampilkannya: lapisan data mengikuti Report
-- Definition, yang memuatnya. Lihat catatan pada DocumentObject.OldID.
--
-- Pemetaan bisnis sengaja tidak ikut di sini: grid hanya menampilkan dua kolom itu, dan
-- menariknya untuk seluruh baris berarti satu kueri yang hasilnya tidak pernah dilihat.
SELECT ID,
       KET_DOC_OBJ,
       OLD_ID
  FROM POOLDATA.V_LST_DOC_OBJ
 ORDER BY ID

-- name: document_object_get
--
-- Satu baris untuk dimuat ke form sunting.
SELECT ID,
       KET_DOC_OBJ,
       OLD_ID
  FROM POOLDATA.V_LST_DOC_OBJ
 WHERE ID = :1

-- name: document_object_business_list
--
-- Pemetaan bisnis satu objek dokumen, beserta nama bisnisnya.
--
-- Asal bentuknya `RDB List/GetLBUID_SQL-SQL.xml` — `select ID, NOTE` dari tabel pemetaan
-- yang di-join ke `BUSINESS (ID, NOTE)`.
--
-- POOLDATA.BUSINESS TIDAK di-join di sini. NOTE pada tabel pemetaan sudah menyimpan
-- namanya, dan menjoin akan MEMBUANG tepat baris yang namanya diketik bebas — baris yang
-- BISNISID-nya memang tidak ada di master, dan yang justru wajib tetap terbaca.
--
-- Hanya baris aktif yang dibaca. Baris yang dicabut pengguna ditandai STS_AKTIF = '0',
-- tidak dihapus (`D-66`).
--
-- Diurutkan menurut URUTAN, bukan menurut nama: susunan baris di grid adalah susunan yang
-- disimpan pengguna, dan mengurutkannya ulang akan membuat layar menampilkan urutan yang
-- berbeda dari yang baru saja ia simpan.
SELECT BISNISID,
       NOTE
  FROM POOLDATA.LST_DOC_OBJ_BUSINESS
 WHERE ID_DOC_OBJ = :1
   AND STS_AKTIF  = '1'
 ORDER BY URUTAN

-- name: document_object_site
--
-- Kode situs, bagian pertama dari setiap ID. Meniru `Database/PEGA_LST_DOC_TYPE.prc:11`
-- persis, termasuk pembandingnya yang berupa teks '1' dan bukan angka.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: document_object_next_sequence
--
-- Urutan milik tabel ini. Namanya DUGAAN, diturunkan dari `SET_LST_DOC_TYPE` yang dipakai
-- tabel bersaudaranya pada `Database/PEGA_LST_DOC_TYPE.prc:21`.
--
-- Bila urutan ini sudah ada di basis data, memakainya membuat ID baru MELANJUTKAN deret
-- yang pernah diterbitkan Pega sehingga tidak pernah bertabrakan. Bila belum ada, migrasi
-- 0008 membuatnya — dan nilai awalnya WAJIB diambil dari ID tertinggi yang sudah ada,
-- bukan dari 1.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada `ADR-0005`, satu-satunya tempat lain
-- yang dibenarkan memuat percabangan dialek.
SELECT POOLDATA.SET_LST_DOC_OBJ.NEXTVAL
  FROM DUAL

-- name: document_object_insert
--
-- OLD_ID sengaja tidak diisi: penomoran lama melekat pada baris warisan dan tidak pernah
-- diberikan pada baris baru.
--
-- Bila tabel dasarnya ternyata masih menyimpan dokumen JSON seperti LST_DOC_TYPE sebelum
-- migrasi 0005, kolom itu pun sengaja tidak diisi — keputusan Work Owner 2026-09-23 untuk
-- modul ini: penyimpanan langsung ke kolom, bukan ke JSON. Akibatnya dicatat di migrasi
-- 0008: baris yang ditulis aplikasi ini TIDAK punya dokumen JSON.
INSERT INTO POOLDATA.LST_DOC_OBJ (ID, KET_DOC_OBJ)
VALUES (:1, :2)

-- name: document_object_update
--
-- Hanya KET_DOC_OBJ yang diubah. ID tidak pernah berubah — mengubahnya akan memutus setiap
-- baris LST_TYPE_DOC_BUSINESS yang merujuknya lewat OBJECT_DOC_ID — dan OLD_ID adalah
-- jejak sejarah.
UPDATE POOLDATA.LST_DOC_OBJ
   SET KET_DOC_OBJ = :1
 WHERE ID = :2

-- name: document_object_business_deactivate
--
-- Langkah PERTAMA dari penggantian pemetaan bisnis: seluruh baris milik satu objek dokumen
-- ditandai tidak aktif.
--
-- Grid di form mengirim susunan akhir yang dikehendaki petugas, tanpa satu pun penanda
-- baris mana yang baru, mana yang berubah, dan mana yang dibuang. Penggantian menyeluruh
-- karena itu satu-satunya tafsiran yang tidak menebak — perlakuan yang sama dengan sub-grid
-- pada modul Daftar Detail Dokumen Travel.
--
-- Yang dikerjakan adalah PENANDAAN, bukan DELETE. `D-66` menetapkan tidak ada penghapusan
-- fisik pada data bernilai bisnis, dan pemetaan ini menyatakan kewenangan sebuah bisnis atas
-- sebuah objek dokumen — jejak bahwa ia pernah berlaku punya arti.
UPDATE POOLDATA.LST_DOC_OBJ_BUSINESS
   SET STS_AKTIF = '0'
 WHERE ID_DOC_OBJ = :1

-- name: document_object_business_activate
--
-- Langkah KEDUA: baris pada posisi tersebut dihidupkan kembali dengan isi yang baru.
--
-- Kuncinya (ID_DOC_OBJ, URUTAN) — POSISI baris di grid, bukan namanya. Ini berbeda dari
-- modul Master COL Simas Online, dan perbedaannya disengaja:
--
--	di sana  tabelnya SUDAH ADA tanpa kolom urutan, sehingga barisnya terpaksa dikenali
--	         dari namanya — dan akibatnya urutan yang disimpan pengguna hilang, serta
--	         nama kembar menyatu menjadi satu baris
--	di sini  tabelnya BELUM ADA sama sekali, sehingga kolom urutan dapat dirancang sejak
--	         awal — urutan pengguna terjaga, dan nama kembar tetap dua baris
--
-- Nama kembar memang sah: grid Pega tidak punya satu pun penanda keunikan.
UPDATE POOLDATA.LST_DOC_OBJ_BUSINESS
   SET BISNISID  = :1,
       NOTE      = :2,
       STS_AKTIF = '1'
 WHERE ID_DOC_OBJ = :3
   AND URUTAN     = :4

-- name: document_object_business_insert
--
-- Dijalankan hanya bila document_object_business_activate tidak mengenai satu baris pun.
-- Ini upsert yang portabel: MERGE didukung keduanya tetapi sintaksnya berbeda cukup jauh,
-- sedangkan `INSERT ... ON CONFLICT` hanya ada di PostgreSQL.
--
-- BISNISID boleh NULL — nama yang diketik bebas memang tidak punya ID.
INSERT INTO POOLDATA.LST_DOC_OBJ_BUSINESS (ID_DOC_OBJ, URUTAN, BISNISID, NOTE, STS_AKTIF)
VALUES (:1, :2, :3, :4, '1')

-- name: business_list
--
-- Daftar bisnis untuk saran isian di layar.
--
-- Asal kolomnya: `RDB List/GetLBUID_SQL-SQL.xml` yang membaca `BUSINESS` dengan `ID` dan
-- `NOTE`. Tabel ini milik GISFW dan HANYA DIBACA (`D-03`).
--
-- Diurutkan menurut NOTE, bukan ID: pengguna mencari bisnis dengan namanya, dan urutan kode
-- tidak berarti apa-apa baginya. Pengurutan di basis data, bukan di Go, supaya daftar
-- panjang tidak perlu dimuat seluruhnya ke memori lebih dulu.
SELECT ID,
       NOTE
  FROM POOLDATA.BUSINESS
 ORDER BY NOTE

-- name: document_object_check_table
--
-- Memastikan view induk dapat dibaca akun aplikasi, tanpa mengambil satu baris pun. Dipakai
-- mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip: objeknya tidak ada
-- versus tidak punya hak baca.
SELECT ID,
       KET_DOC_OBJ,
       OLD_ID
  FROM POOLDATA.V_LST_DOC_OBJ
 WHERE 1 = 0

-- name: document_object_write_check_table
--
-- Memastikan TABEL DASAR yang ditulis modul ini benar-benar ada dan dapat dibaca.
--
-- Terpisah dari pemeriksaan view dengan sengaja: nama tabel dasarnya DUGAAN, dan inilah
-- pemeriksaan yang membuktikan dugaan itu benar atau salah — sebelum pengguna pertama
-- menekan Simpan, bukan sesudahnya.
SELECT ID,
       KET_DOC_OBJ,
       OLD_ID
  FROM POOLDATA.LST_DOC_OBJ
 WHERE 1 = 0

-- name: document_object_business_check_table
--
-- Memastikan tabel pemetaan bisnis ada, dengan bentuk yang BENAR-BENAR dipakai
-- document_object_business_list. Tabel ini dibuat migrasi 0008; sebelum migrasi itu
-- dijalankan, pemeriksaan ini memang gagal — dan itulah yang harus dilaporkan.
SELECT ID_DOC_OBJ,
       URUTAN,
       BISNISID,
       NOTE,
       STS_AKTIF
  FROM POOLDATA.LST_DOC_OBJ_BUSINESS
 WHERE 1 = 0

-- name: business_check_table
--
-- Memastikan master bisnis milik GISFW dapat dibaca akun aplikasi.
SELECT ID,
       NOTE
  FROM POOLDATA.BUSINESS
 WHERE 1 = 0
