-- Kueri peta menu dan kewenangannya.
--
-- # Tabel yang dibaca
--
--	POOLDATA.M_APLIKASI           daftar aplikasi; yang dipakai APP_DESC = 'CLAIM PNC'
--	POOLDATA.M_MENU_APLIKASI_PNC  butir menu, berjenjang lewat MENU_ID_LEADER
--	POOLDATA.M_LOGIN_GROUP_PNC    keanggotaan login pada group
--	POOLDATA.M_OTORISASI_PNC      izin per butir menu, untuk login MAUPUN untuk group
--
-- Keempatnya dibaca SAJA. Tidak ada satu pun pernyataan yang menulis: pengelolaan menu
-- dan otorisasi belum punya layar, dan membuatnya adalah pekerjaan tersendiri.
--
-- # Kenapa disaring dengan APP_DESC, bukan APP_ID
--
-- Mengikuti kueri yang ditetapkan Work Owner. Alasannya masuk akal: APP_ID adalah nomor
-- urut yang dapat berbeda antar lingkungan, sedangkan nama aplikasinya tidak.
--
-- Perbandingannya dibungkus UPPER(TRIM(...)) di kedua sisi. APP_DESC bertipe VARCHAR2
-- tanpa penyeragaman apa pun, dan satu spasi di ujung akan membuat seluruh menu hilang
-- tanpa satu pun pesan galat.
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding.

-- name: menu_list
--
-- Bentuknya mengikuti kueri yang ditetapkan Work Owner apa adanya, termasuk join
-- bergaya koma dan ORDER BY MENU_SEQUENCE. Keduanya portabel di Oracle maupun
-- PostgreSQL.
--
-- MENU_ID_LEADER dibaca apa adanya; NULL berarti butir ini kelompok tingkat atas.
SELECT menu.MENU_ID,
       menu.MENU_DESC,
       menu.MENU_PROGRAM,
       menu.MENU_ID_LEADER,
       menu.MENU_SEQUENCE
  FROM POOLDATA.M_APLIKASI app,
       POOLDATA.M_MENU_APLIKASI_PNC menu
 WHERE app.APP_ID = menu.APP_ID
   AND UPPER(TRIM(app.APP_DESC)) = UPPER(TRIM(:1))
 ORDER BY menu.MENU_SEQUENCE

-- name: menu_app_exists
--
-- Dipakai membedakan "aplikasinya tidak terdaftar" dari "aplikasinya ada tetapi belum
-- punya butir menu". Tanpa pembedaan itu, salah ketik APP_DESC terbaca sebagai menu
-- kosong — dan menu kosong tidak menyebut sebabnya.
SELECT app.APP_ID
  FROM POOLDATA.M_APLIKASI app
 WHERE UPPER(TRIM(app.APP_DESC)) = UPPER(TRIM(:1))

-- name: menu_groups_of_login
--
-- Langkah pertama aturan yang ditetapkan Work Owner: dari login yang diketik, cari
-- GROUP_ID apa saja yang diikutinya.
SELECT g.GROUP_ID
  FROM POOLDATA.M_LOGIN_GROUP_PNC g
 WHERE UPPER(TRIM(g.LOGIN_ID)) = UPPER(TRIM(:1))
 ORDER BY g.GROUP_ID

-- name: menu_authorized_ids
--
-- Langkah kedua dan ketiga sekaligus: izin untuk group-group itu DAN untuk loginnya
-- sendiri. Kolom LOGIN_ID_GROUP menampung keduanya, sehingga satu pernyataan cukup.
--
-- Penanda /*SUBJECTS*/ diganti daftar parameter `:2, :3, …` oleh pemanggil — lihat
-- expandSubjects. Yang disisipkan hanyalah PENANDA PARAMETER, tidak pernah nilainya;
-- tidak ada satu pun nilai yang dirangkai ke dalam teks SQL.
SELECT DISTINCT o.MENU_ID
  FROM POOLDATA.M_APLIKASI app,
       POOLDATA.M_OTORISASI_PNC o
 WHERE app.APP_ID = o.APP_ID
   AND UPPER(TRIM(app.APP_DESC)) = UPPER(TRIM(:1))
   AND UPPER(TRIM(o.LOGIN_ID_GROUP)) IN (/*SUBJECTS*/)

-- name: menu_check_table
--
-- Memastikan keempat tabel ada dan dapat dibaca akun aplikasi, tanpa mengambil satu
-- baris pun — sehingga aman dijalankan terhadap produksi.
SELECT app.APP_ID
  FROM POOLDATA.M_APLIKASI app,
       POOLDATA.M_MENU_APLIKASI_PNC menu,
       POOLDATA.M_OTORISASI_PNC o,
       POOLDATA.M_LOGIN_GROUP_PNC g
 WHERE app.APP_ID = menu.APP_ID
   AND o.APP_ID = menu.APP_ID
   AND o.MENU_ID = menu.MENU_ID
   AND g.LOGIN_ID = o.LOGIN_ID_GROUP
   AND 1 = 0
