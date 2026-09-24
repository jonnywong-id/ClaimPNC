-- Kueri master ambang komite: POOLDATA.EMAILKOMITE.
--
-- # POOLDATA yang mana
--
-- Setiap server punya POOLDATA-nya sendiri, dan aplikasi lama membaca POOLDATA milik
-- server tempat ia berjalan (Work Owner, 2026-09-19). Isi EMAILKOMITE karena itu BERBEDA
-- antar server: baris ber-TYPE_BUSINESS `SIMASNET` hanya ada di POOLDATA server Simasnet,
-- dan `Database/emailkomite.csv` yang kita pegang berasal dari POOLDATA server ASM.
--
-- Konsekuensinya untuk sistem baru: kueri ini WAJIB dijalankan pada koneksi portal yang
-- sedang AKTIF, bukan pada koneksi utama. Selama itu belum terpasang (`TKT-F6-002`),
-- portal mana pun yang dipilih akan membaca tangga ambang milik portal utama — angka yang
-- salah tanpa satu pun tanda. Catatannya ada di cmd/claimpnc/main.go.
--
-- # Tabel ini DIBACA SAJA
--
-- Work Owner menetapkan 2026-09-17 bahwa aplikasi ini tidak menulis ke tabel ini selama
-- masa paralel. `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem, dan tabel
-- ini masih ditulis Pega serta dibaca 17 kueri di sana — di antaranya
-- `EmailKomiteBerjenjang_sql`, `EmailKomiteBerjenjangPA_sql`,
-- `EmailKomiteBerjenjangTravel_sql`, dan `EmailKomiteSalvage_sql`.
--
-- Memindahkan kepemilikannya menuntut prosedur `D-63`: permintaan perubahan skema
-- tertulis, persetujuan Work Owner, pelaksanaan DBA, lalu pengujian dengan MENJALANKAN
-- PEGA DAN GO BERSAMAAN. Itu belum ditempuh, sehingga tidak ada satu pun pernyataan
-- INSERT, UPDATE, atau DELETE di berkas ini.
--
-- # Kenapa seluruh baris dibaca, tanpa WHERE
--
-- Penyaringan — aktif, jenis, lini, pita, dan akumulasi batas bawah — dikerjakan di Go
-- sebagai fungsi murni. Yang diperoleh: aturan penjenjangan hidup di SATU tempat yang
-- dapat diuji tanpa basis data, dan perilakunya dijamin sama persis antara Oracle dan
-- penyimpanan di memori.
--
-- Menaruh penyaringan di klausa WHERE akan memecah aturan itu menjadi dua salinan yang
-- dapat berbeda pendapat — persis pola yang membuat sistem lama menyebarkan satu aturan
-- bisnis ke activity, SQL, dan stored procedure sekaligus.
--
-- Biayanya nol: isinya 30 baris.
--
-- # Kolom yang SENGAJA tidak dibaca
--
-- EMAIL dan CC tidak diambil. Modul ini menghitung SIAPA yang menyetujui, bukan ke mana
-- pemberitahuan dikirim — yang terakhir adalah `S-3`. Dan `D-67` menetapkan alamat
-- pribadi pada master lama, sekurang-kurangnya enam akun Gmail di jalur produksi, TIDAK
-- dibawa ke sistem baru sama sekali. Tidak membacanya sejak awal membuat alamat itu
-- tidak pernah sampai ke peramban.
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding.

-- name: ambang_komite_daftar
--
-- Urutan di sini hanya membuat keluaran enak dibaca saat kueri dijalankan langsung oleh
-- DBA. Aturan urutan yang MENGIKAT ada di Go (komite.Tentukan), karena `ORDER BY DEGREE`
-- saja tidak menentukan urutan saat DEGREE seri — dan seri itu benar-benar ada pada
-- master: Non-MBU pita 1 memiliki dua baris ber-DEGREE 1.
SELECT ID,
       NAME,
       OPERATOR_ID,
       TYPE_BUSINESS,
       TYPE_KOMITE,
       LIMIT_BOTTOM,
       LIMIT_TOP,
       DEGREE,
       STS_AKTIF,
       STS_ADJ,
       STS_REG,
       STS_REJECT,
       STS_ABS
  FROM POOLDATA.EMAILKOMITE
 ORDER BY TYPE_BUSINESS, TYPE_KOMITE, LIMIT_BOTTOM, DEGREE, ID

-- name: ambang_komite_periksa_tabel
--
-- Memastikan tabel dan seluruh kolomnya dapat dibaca akun aplikasi, tanpa mengambil satu
-- baris pun. Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip:
-- tabelnya tidak ada versus tidak punya hak baca.
SELECT ID,
       NAME,
       OPERATOR_ID,
       TYPE_BUSINESS,
       TYPE_KOMITE,
       LIMIT_BOTTOM,
       LIMIT_TOP,
       DEGREE,
       STS_AKTIF,
       STS_ADJ,
       STS_REG,
       STS_REJECT,
       STS_ABS
  FROM POOLDATA.EMAILKOMITE
 WHERE 1 = 0
