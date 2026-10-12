-- Kueri master Dominan Factor: POOLDATA.M_DOMINAN_FACTOR.
--
-- # Tabel mana yang dibaca, dan tabel mana yang ditulis
--
-- Keduanya M_DOMINAN_FACTOR — tabel dasarnya. Tidak ada view perantara untuk master ini,
-- berbeda dari M_STS_CLAIM yang dibaca Pega lewat V_STS_CLAIM. Ketiga rule Pega yang
-- memakainya menembak tabelnya langsung:
--
--   RDB List/GetDataDominanFactor-SQL.xml            SELECT ID, NAME  (layar master)
--   RDB List/GetDataDominanFactorListOS-SQL.xml      JOIN dari T_CLAIM_DOMINANFACTOR
--   RDB List/GetDataOutstandingperCabangExport-SQL.xml  idem, untuk laporan
--
-- Artinya begitu aplikasi Go menulis ke tabel ini, hasilnya LANGSUNG terlihat ketiganya
-- tanpa satu pun pernyataan DDL. Modul ini karena itu TIDAK menuntut berkas migrasi.
--
-- # Kepemilikan tabel (P-1)
--
-- Penulis tunggalnya di sistem lama adalah `RDB List/InsertDominanfactor-SQL.xml`, satu-
-- satunya pemanggil `POOLDATA.PEGA_M_DOMINAN_FACTOR` di seluruh export — diperiksa dengan
-- menghitung langsung, bukan diandaikan. Memindahkan layarnya ke sini memindahkan
-- kepemilikan tabelnya secara utuh; Pega berubah menjadi pembaca saja.
--
-- # Kenapa tidak lagi lewat PEGA_M_DOMINAN_FACTOR
--
-- `D-02` menetapkan logika stored procedure naik ke Go dan aplikasi tidak memanggil
-- procedure. `D-68` menambahkan alasan yang lebih keras, dan procedure ini contoh
-- persisnya: parameter keluarannya bernama `ErrMsg`, tetapi pada jalur BERHASIL ia berisi
-- kalimat `'Data Sudah Disimpan dengan ID : ' || ID_SITE` (`:14`). Pemanggil tidak dapat
-- membedakan berhasil dari gagal tanpa membaca teksnya.
--
-- Ditambah cacat transaksi yang sama seperti yang `D-68` catat: procedure itu COMMIT
-- sendiri di dalam cabang INSERT (`:15`) maupun UPDATE (`:24`), sementara ROLLBACK-nya
-- berada di handler yang berjalan SESUDAH commit — sehingga tidak memulihkan apa pun.
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding.

-- name: dominant_factor_list
--
-- TANPA ORDER BY, persis seperti kueri Pega. Pengurutannya dikerjakan di Go, dan itu
-- perlu: ID dibentuk `max+1` tanpa nol di depan, sehingga pengurutan sebagai TEKS
-- menempatkan `10` sebelum `9`. Mengurutkannya di SQL menuntut `TO_NUMBER`, yang terikat
-- dialek Oracle dan dilarang `docs/Steering/09-DATABASE-STRATEGY.md` §4.
SELECT ID,
       NAME
  FROM POOLDATA.M_DOMINAN_FACTOR

-- name: dominant_factor_get
SELECT ID,
       NAME
  FROM POOLDATA.M_DOMINAN_FACTOR
 WHERE ID = :1

-- name: dominant_factor_lock_ids
--
-- Membaca SELURUH ID sambil menguncinya, sebagai langkah pertama penyisipan.
--
-- Ini yang menutup cacat balapan pada `max+1` di procedure lama
-- (`Database/PEGA_M_DOMINAN_FACTOR.prc:11`): tanpa kunci, dua penyimpanan yang tiba
-- bersamaan sama-sama membaca nilai maksimum yang sama lalu sama-sama menyisipkan nomor
-- itu. FOR UPDATE membuat penyimpanan kedua menunggu sampai yang pertama selesai.
--
-- Hanya ID yang diambil, bukan seluruh baris: yang dibutuhkan hanyalah nomor terbesar,
-- dan mengambil NAME sekaligus akan menarik isi yang tidak dipakai ke dalam kunci.
--
-- Batasnya jujur: bila tabel KOSONG tidak ada baris yang dapat dikunci, sehingga dua
-- penyisipan pertama yang benar-benar bersamaan masih dapat menghasilkan nomor kembar.
-- Keadaan itu hanya mungkin sekali seumur hidup tabel, dan penjagaan sisanya ada di
-- pemeriksaan ID di kode Go.
SELECT ID
  FROM POOLDATA.M_DOMINAN_FACTOR
   FOR UPDATE

-- name: dominant_factor_insert
INSERT INTO POOLDATA.M_DOMINAN_FACTOR (ID, NAME)
VALUES (:1, :2)

-- name: dominant_factor_update
--
-- Hanya NAME yang diubah. ID tidak pernah berubah — mengubahnya akan memutus setiap
-- baris T_CLAIM_DOMINANFACTOR yang menyimpan nomor itu, dan laporan Outstanding per
-- Cabang akan kehilangan faktornya tanpa pesan apa pun.
UPDATE POOLDATA.M_DOMINAN_FACTOR
   SET NAME = :1
 WHERE ID = :2

-- name: dominant_factor_check_table
--
-- Memastikan tabel ada dan kedua kolomnya dapat dibaca akun aplikasi, tanpa mengambil
-- satu baris pun. Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak
-- mirip tetapi perbaikannya berbeda jauh: tabelnya tidak ada di portal itu, versus akun
-- aplikasi tidak punya hak baca atas tabel warisan.
SELECT ID,
       NAME
  FROM POOLDATA.M_DOMINAN_FACTOR
 WHERE 1 = 0
