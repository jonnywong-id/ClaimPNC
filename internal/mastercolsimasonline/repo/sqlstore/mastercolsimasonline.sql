-- Kueri master COL Simas Online.
--
-- # Tabel mana yang dipakai, dan bagaimana itu dibuktikan
--
--   POOLDATA.M_CAUSE_OF_LOSS_ONLINE          induk — satu penyebab kerugian
--   POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL   anak  — pemetaan ke lini bisnis
--   POOLDATA.BUSINESS                        master bisnis milik GISFW, HANYA DIBACA
--
-- Versi sebelumnya berkas ini membaca POOLDATA.M_CAUSE_OF_LOSS — master COL biasa. Itu
-- KELIRU, dan keliru dengan cara yang tidak menimbulkan galat: kedua tabel punya nama
-- kolom yang sama persis, sehingga kueri tetap berjalan dan hanya menampilkan baris milik
-- master yang lain.
--
-- Empat bukti yang menetapkan tabel ONLINE sebagai yang benar:
--
--   1. `Activity/Online_nsertCauseOfLoss_act-Act.xml` — jalur Simpan layar Simas Online —
--      memanggil `UpdateMCauseOfLoss_online`, BUKAN `UpdateMCauseOfLoss`. Yang terakhir
--      adalah pemanggil `PEGA_M_CAUSE_OF_LOSS` dan milik layar Master COL biasa.
--   2. `Database/PEGA_M_CAUSE_OF_LOSS_ONLINE` menulis
--      `POOLDATA.M_CAUSE_OF_LOSS_ONLINE(M_COL_ID, NAME_M_COL_ID, JSON_DATA)` dan
--      membentuk kodenya dari `M_SITE_DATABASE.ID || lpad(M_CAUSE_SEQ_ONLINE.nextval,3,'0')`.
--   3. Kecocokan kunci pada basis data ini: dari 55 induk yang dirujuk
--      M_CAUSE_OF_LOSS_ONLINE_DETAIL.M_COL_ID, **55 cocok** dengan
--      M_CAUSE_OF_LOSS_ONLINE dan **1 cocok** dengan M_CAUSE_OF_LOSS.
--   4. M_COL_ID pada tabel ONLINE berada di rentang 1465..1519 dan lebarnya VARCHAR2(4) —
--      persis bentuk yang dihasilkan procedure di butir 2.
--
-- # Yang TIDAK dipakai lagi
--
-- MST_COL_ID — "ID Master Kerugian" — dicabut seluruhnya atas keputusan Work Owner
-- 2026-09-23: kolomnya sudah tidak dipakai. Ia tidak pernah ada sebagai kolom pada tabel
-- mana pun; versi sebelumnya membacanya dengan mem-parse OLD_M_COL_ID sebagai JSON, dan
-- pada seluruh 31 baris ber-OLD_M_COL_ID isinya bukan JSON sehingga hasilnya selalu
-- kosong. Tidak ada data yang hilang karena mencabutnya.
--
-- View POOLDATA.V_M_CAUSE_OF_LOSS juga tidak dipakai: definisinya
-- `SELECT M_COL_ID, OLD_M_COL_ID, COL_DESC FROM M_CAUSE_OF_LOSS` — ia memandang master
-- yang SALAH untuk layar ini.
--
-- # Kenapa tidak lewat procedure
--
-- `D-02` menetapkan logika procedure naik ke Go. `D-68` menambahkan alasan yang lebih
-- keras, dan PEGA_M_CAUSE_OF_LOSS_ONLINE memperagakannya: parameter keluarannya bernama
-- `ErrMsg` tetapi pada jalur BERHASIL ia berisi kalimat "Data Sudah Disimpan dengan ID :
-- 1520", sehingga pemanggil tidak dapat membedakan berhasil dari gagal tanpa membaca
-- teksnya. Procedure itu pun berstatus INVALID di basis data ini.
--
-- Empat aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (`D-20`).
--   4. Tanpa pemanggilan stored procedure (`D-02`).


-- name: cause_of_loss_list
--
-- Sumber grid layar (`Section/Online_GridCauseOfLoss-Section.xml`): ID dan Description.
--
-- COL_DESC yang dibaca, bukan NAME_M_COL_ID. Keduanya terisi dan pada seluruh 55 baris
-- isinya SAMA PERSIS, tetapi COL_DESC-lah yang dibaca grid Pega. Penyimpanan menulis
-- keduanya supaya kesamaan itu tetap berlaku untuk baris baru.
--
-- Pemetaan bisnis sengaja tidak ikut di sini: grid hanya menampilkan dua kolom itu, dan
-- menariknya untuk seluruh baris berarti satu kueri yang hasilnya tidak pernah dilihat.
SELECT M_COL_ID,
       COL_DESC
  FROM POOLDATA.M_CAUSE_OF_LOSS_ONLINE
 ORDER BY M_COL_ID

-- name: cause_of_loss_get
--
-- Satu baris untuk dimuat ke form sunting.
SELECT M_COL_ID,
       COL_DESC
  FROM POOLDATA.M_CAUSE_OF_LOSS_ONLINE
 WHERE M_COL_ID = :1

-- name: cause_of_loss_business_list
--
-- Pemetaan bisnis satu penyebab kerugian, beserta nama bisnisnya.
--
-- Asal bentuknya `RDB List/GetLBUID_SQL-SQL.xml` — `select ID, NOTE` — dan tabel anaknya
-- memakai nama kolom yang sama persis: ID adalah BUSINESS.ID, NOTE adalah namanya.
--
-- Dibaca LANGSUNG dari tabel, bukan dengan mem-parse JSON_DATA induknya. Keputusan Work
-- Owner 2026-09-22, dan pada data hari ini tabel inilah yang memang berisi: 282 baris
-- untuk 55 induk, dan tidak ada satu pun induk yang tanpa baris detail.
--
-- POOLDATA.BUSINESS tidak di-join. NOTE pada tabel detail sudah menyimpan namanya, dan
-- pemeriksaan membuktikan ia cocok dengan BUSINESS.NOTE pada SELURUH baris yang ID-nya
-- ketemu. Menjoin akan membuang satu baris yang ID-nya memang tidak ada di master —
-- yaitu nama yang diketik bebas, yang justru wajib tetap terbaca
-- (`pyAllowFreeFormInput=true`).
--
-- Diurutkan menurut NOTE karena tabel ini TIDAK punya kolom urutan. Akibatnya susunan
-- baris yang disimpan pengguna tidak dapat dipertahankan; lihat catatan pada
-- cause_of_loss_business_insert.
SELECT ID,
       NOTE
  FROM POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL
 WHERE M_COL_ID = :1
 ORDER BY NOTE

-- name: cause_of_loss_site
--
-- Kode situs, bagian pertama dari setiap M_COL_ID. Meniru
-- `Database/PEGA_M_CAUSE_OF_LOSS_ONLINE` persis, termasuk pembandingnya yang berupa teks
-- '1' dan bukan angka.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: cause_of_loss_next_sequence
--
-- Urutan MILIK JALUR ONLINE — M_CAUSE_SEQ_ONLINE, bukan M_CAUSE_SEQ.
--
-- Keduanya ada, keduanya bernama mirip, dan keduanya memasok master yang BERBEDA.
-- Tertukar berarti kode yang diterbitkan layar ini bertabrakan dengan deret milik master
-- COL biasa. Urutan ini melanjutkan deret yang sudah ada sehingga kode baru tidak pernah
-- bertabrakan dengan kode yang pernah diterbitkan Pega.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada `ADR-0005`.
SELECT POOLDATA.M_CAUSE_SEQ_ONLINE.NEXTVAL
  FROM DUAL

-- name: cause_of_loss_insert
--
-- NAME_M_COL_ID dan COL_DESC diisi nilai yang SAMA.
--
-- Bukan duplikasi asal: pada seluruh 55 baris yang ada, keduanya memang sama persis.
-- Procedure lama hanya mengisi NAME_M_COL_ID, sedangkan grid membaca COL_DESC — sehingga
-- mengisi salah satu saja akan menghasilkan baris yang tersimpan tetapi tidak muncul di
-- layar, atau muncul tanpa nama.
--
-- JSON_DATA sengaja TIDAK diisi. Sejak Work Owner menetapkan penyimpanan langsung ke
-- kolom, kolom itu menjadi peninggalan: ia dibiarkan berisi nilai terakhir yang ditulis
-- Pega pada baris lama, dan kosong pada baris baru.
INSERT INTO POOLDATA.M_CAUSE_OF_LOSS_ONLINE (M_COL_ID, NAME_M_COL_ID, COL_DESC)
VALUES (:1, :2, :3)

-- name: cause_of_loss_update
--
-- M_COL_ID hanya menyaring, tidak pernah ikut di-SET — sama seperti cabang UPDATE pada
-- procedure lama. Ia dirujuk M_CAUSE_OF_LOSS_ONLINE_DETAIL.M_COL_ID pada data yang sudah
-- berjalan.
UPDATE POOLDATA.M_CAUSE_OF_LOSS_ONLINE
   SET NAME_M_COL_ID = :1,
       COL_DESC      = :2
 WHERE M_COL_ID = :3

-- name: cause_of_loss_business_update
--
-- Melengkapi ID pada baris pemetaan yang SUDAH ada, dikenali dari namanya.
--
-- # Kenapa kuncinya NOTE, bukan ID
--
-- ID boleh kosong: nama yang diketik bebas tidak punya ID sama sekali, dan pada data hari
-- ini memang ada satu baris seperti itu. NOTE-lah yang selalu terisi, dan NOTE pula yang
-- benar-benar terikat di layar Pega — sel gridnya `pyValue = .Note`, sedangkan `.ID` hanya
-- kolom tersembunyi yang ikut terisi saat sebuah pilihan diambil dari daftar.
--
-- Pemeriksaan pada data hari ini: pasangan (M_COL_ID, ID) tidak pernah muncul dua kali,
-- dan NOTE cocok dengan BUSINESS.NOTE pada seluruh baris yang ID-nya ketemu.
UPDATE POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL
   SET ID = :1
 WHERE M_COL_ID = :2
   AND NOTE     = :3

-- name: cause_of_loss_business_insert
--
-- Dijalankan hanya bila cause_of_loss_business_update tidak mengenai satu baris pun.
-- Ini upsert yang portabel: MERGE didukung keduanya tetapi sintaksnya berbeda cukup jauh,
-- sedangkan `INSERT ... ON CONFLICT` hanya ada di PostgreSQL.
--
-- # PERINGATAN — pemetaan hanya dapat DITAMBAH, belum dapat DICABUT
--
-- Tabel ini tidak punya kolom penanda aktif. Mencabut pilihan di layar karena itu tidak
-- dapat disimpan: `D-66` melarang penghapusan fisik data bernilai bisnis, dan penanda yang
-- akan menggantikannya belum ada.
--
-- Yang dibutuhkan untuk menutupnya adalah SATU kolom dari DBA:
--
--   ALTER TABLE POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL ADD (STS_AKTIF VARCHAR2(1) DEFAULT '1');
--
-- Begitu kolom itu ada, kueri pembacanya menyaring `STS_AKTIF = '1'` dan pencabutan
-- menjadi UPDATE penanda. Sampai saat itu perilakunya ADITIF, dan itu keadaan yang
-- diketahui — bukan cacat yang belum ketahuan.
--
-- TGL_INPUT dan USER_INPUT tidak diisi. Keduanya nullable, dan modul ini belum punya seam
-- jam maupun identitas pelaku; mengisinya dari jam basis data akan melanggar `F-5` dan
-- membawa kembali `R-12`.
INSERT INTO POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL (M_COL_ID, ID, NOTE)
VALUES (:1, :2, :3)

-- name: business_list
--
-- Daftar bisnis untuk pilihan di layar.
--
-- Asal kolomnya: `RDB List/GetLBUID_SQL-SQL.xml` yang membaca `BUSINESS` dengan `ID` dan
-- `NOTE`. Tabel ini milik GISFW dan hanya dibaca (`D-03`).
--
-- Diurutkan menurut NOTE, bukan ID: pengguna mencari bisnis dengan namanya, dan urutan
-- kode tidak berarti apa-apa baginya. Pengurutan di basis data, bukan di Go, supaya daftar
-- panjang tidak perlu dimuat seluruhnya ke memori lebih dulu.
SELECT ID,
       NOTE
  FROM POOLDATA.BUSINESS
 ORDER BY NOTE

-- name: cause_of_loss_check_table
--
-- Memastikan induk dapat dibaca akun aplikasi, tanpa mengambil satu baris pun. Dipakai
-- mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip: tabelnya tidak ada
-- versus tidak punya hak baca.
SELECT M_COL_ID,
       COL_DESC
  FROM POOLDATA.M_CAUSE_OF_LOSS_ONLINE
 WHERE 1 = 0

-- name: cause_of_loss_business_check_table
--
-- Memastikan tabel pemetaan bisnis dapat dibaca, dengan bentuk yang BENAR-BENAR dipakai
-- cause_of_loss_business_list.
SELECT ID,
       NOTE
  FROM POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL
 WHERE 1 = 0

-- name: business_check_table
--
-- Memastikan master bisnis milik GISFW dapat dibaca akun aplikasi.
SELECT ID,
       NOTE
  FROM POOLDATA.BUSINESS
 WHERE 1 = 0
