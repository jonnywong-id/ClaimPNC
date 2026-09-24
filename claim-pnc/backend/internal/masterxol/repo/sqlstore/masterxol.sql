-- Kueri Master XOL: POOLDATA.MST_XOL_PNC beserta ketiga tabel anaknya.
--
-- # Empat tabel, dan hubungannya
--
--   MST_XOL_PNC       ID PK                  induk
--   MST_XOL_BUSINESS  ID → induk             grup bisnis yang dicakup, TANPA kunci utama
--   MST_XOL_LAYER     IDLAYER PK, ID → induk lapisan
--   MST_XOL_REAS      IDLAYER → lapisan      share reasuradur, TANPA kunci utama
--
-- Bentuk kolomnya dibaca dari ALL_TAB_COLUMNS portal ASM pada 2026-09-20. Dua di antara
-- keempat tabel TIDAK punya kunci utama maupun indeks unik, sehingga baris kembar mungkin
-- terjadi dan memang sudah terjadi: induk 10009 menyimpan DUA baris bisnis yang sama-sama
-- ber-IDBUSINESS NULL. Pemeriksaan ganda karena itu dikerjakan kode, bukan basis data —
-- dan penambahan indeks unik diusulkan ke DBA lewat jalur `D-63`.
--
-- # Kepemilikan tabel (P-1)
--
-- Penulis tunggalnya di sistem lama adalah layar Master XOL sendiri:
-- `RDB List/SetMasterXOL-SQL.xml` adalah satu-satunya pemanggil
-- `POOLDATA.INSERT_UPDATE_MST_XOL`, dan `Activity/DeleteFromTabelMst-Act.xml.xml`
-- satu-satunya penghapusnya. Memindahkan layarnya ke sini memindahkan kepemilikan
-- keempat tabel secara utuh; Pega berubah menjadi pembaca saja.
--
-- Tabel yang HANYA DIBACA — BUSINESS, BUSINESSGROUP, PROPORTIONALARRG, M_TREATYYEAR —
-- tetap milik sistem lain. Tidak ada satu pun pernyataan tulis terhadap keempatnya di
-- berkas ini, dan memang tidak boleh ada.
--
-- # Kenapa tidak lagi lewat INSERT_UPDATE_MST_XOL
--
-- `D-02` menetapkan logika stored procedure naik ke Go, dan `D-68` menambahkan alasan
-- yang lebih keras. Procedure ini contoh persisnya:
--
--   * Ia COMMIT sendiri di dalam cabang insert induk (`:28`) dan insert lapisan (`:85`),
--     sementara ROLLBACK-nya berada di handler yang berjalan SESUDAH commit itu — jadi
--     tidak memulihkan apa pun.
--   * Parameter keluarannya bernama `ERRMSG`, tetapi pada jalur BERHASIL ia berisi
--     kalimat `'Data Sudah Disimpan dengan ID : ' || idcount2` (`:26`). Pemanggil tidak
--     dapat membedakan berhasil dari gagal tanpa membaca teksnya — dan layar lama memang
--     memeriksanya dengan `@contains(OutputData.ResponseNote,"Error")`.
--   * `IDMST2` hanya diisi pada cabang INSERT. Pada UPDATE ia kosong, dan layar lama
--     menambalnya dengan `@If(OutputData.IDMaster=="", TempXOL.BranchID, ...)`.
--
-- # Satu jebakan portabilitas yang wajib diperhatikan
--
-- Kolom `MST_XOL_LAYER.LIMIT` bernama sama dengan kata kunci `LIMIT`. Di Oracle ia bukan
-- kata cadangan sehingga dapat ditulis polos, tetapi di PostgreSQL **ia kata cadangan**
-- dan kuerinya akan gagal. Karena `D-20` menuntut SATU set SQL yang berjalan di keduanya,
-- kolom itu selalu ditulis sebagai `"LIMIT"`. Tanda kutipnya aman di Oracle: kolomnya
-- dibuat tanpa kutip sehingga namanya tersimpan huruf besar, persis yang dirujuk di sini.
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding.

-- name: xol_list
--
-- Daftar induk untuk grid, TANPA anaknya.
--
-- Kueri lama (`RDB List/GetDataMasterXOL-SQL.xml`) ikut menarik nilai minimum limit dan
-- excess seluruh lapisan lewat tiga subkueri berkorelasi, lalu mengalikannya dengan kurs.
-- Ketiganya TIDAK dibawa: grid di layar hanya menampilkan ID, Tahun, Kurs, dan Remark
-- Komite — ketiga nilai itu dihitung lalu tidak pernah ditampilkan.
--
-- TANPA ORDER BY. Pengurutannya dikerjakan di Go, dan itu perlu: ID dibentuk `max+1`
-- tanpa nol di depan, sehingga pengurutan sebagai TEKS menempatkan "10010" sebelum
-- "1009". Mengurutkannya di SQL menuntut `TO_NUMBER`, yang terikat dialek Oracle dan
-- dilarang `docs/Steering/09-DATABASE-STRATEGY.md` §4.
SELECT ID,
       NAMA,
       TAHUN,
       KURSVALUE,
       TYPEXOL,
       PIC,
       STSKOMITE,
       KOMITE,
       REMARKPIC,
       REMARKKOMITE
  FROM POOLDATA.MST_XOL_PNC

-- name: xol_get
SELECT ID,
       NAMA,
       TAHUN,
       KURSVALUE,
       TYPEXOL,
       PIC,
       STSKOMITE,
       KOMITE,
       REMARKPIC,
       REMARKKOMITE
  FROM POOLDATA.MST_XOL_PNC
 WHERE ID = :1

-- name: xol_business_list
--
-- Menggantikan `RDB List/GetDataBisnisXOL-SQL.xml`.
SELECT IDBUSINESS,
       GROUPBUSINESS
  FROM POOLDATA.MST_XOL_BUSINESS
 WHERE ID = :1

-- name: xol_layer_list
--
-- Menggantikan `RDB List/GetDataLayerXOL-SQL.xml`, ditambah CONVERT_LIMIT yang kueri lama
-- tidak ambil — layar lama menghitungnya ulang di klien, dan itu yang membuat nilai
-- tersimpan dan nilai tampil dapat berbeda.
SELECT IDLAYER,
       NAMA,
       "LIMIT",
       EXCESS,
       CONVERT_LIMIT
  FROM POOLDATA.MST_XOL_LAYER
 WHERE ID = :1

-- name: xol_reas_list
--
-- Menggantikan `RDB List/GetDataReasXOL-SQL.xml`.
SELECT IDREAS,
       NAMA,
       PERCENTSHARE
  FROM POOLDATA.MST_XOL_REAS
 WHERE IDLAYER = :1

-- name: xol_lock_master_ids
--
-- Membaca SELURUH ID induk sambil menguncinya, sebagai langkah pertama penyisipan.
--
-- Ini yang menutup cacat balapan pada `max(to_number(id))+1` di procedure lama
-- (`INSERT_UPDATE_MST_XOL.prc:16`): tanpa kunci, dua penyimpanan yang tiba bersamaan
-- sama-sama membaca nilai maksimum yang sama lalu sama-sama menyisipkan nomor itu — dan
-- karena MST_XOL_PNC punya kunci utama, yang kedua gagal dengan galat basis data mentah.
--
-- Hanya ID yang diambil: yang dibutuhkan hanyalah nomor terbesar.
--
-- Batasnya jujur: bila tabel KOSONG tidak ada baris yang dapat dikunci, sehingga dua
-- penyisipan pertama yang benar-benar bersamaan masih dapat menghasilkan nomor kembar.
SELECT ID
  FROM POOLDATA.MST_XOL_PNC
   FOR UPDATE

-- name: xol_lock_layer_ids
--
-- Sama seperti di atas, untuk IDLAYER. Nomornya berjalan GLOBAL lintas induk — produksi
-- membuktikannya: 10001 sampai 10018 tersebar di delapan induk, bukan dimulai ulang per
-- induk.
SELECT IDLAYER
  FROM POOLDATA.MST_XOL_LAYER
   FOR UPDATE

-- name: xol_insert_master
--
-- Kelima kolom yang sama dengan `INSERT_UPDATE_MST_XOL.prc:25`. Kolom komite — PIC,
-- STSKOMITE, KOMITE, REMARKPIC, REMARKKOMITE — sengaja TIDAK diisi di sini, persis
-- seperti procedure lama: yang mengisinya adalah pengajuan ke komite, sebuah langkah
-- tersendiri.
INSERT INTO POOLDATA.MST_XOL_PNC (ID, NAMA, TAHUN, KURSVALUE, TYPEXOL)
VALUES (:1, :2, :3, :4, :5)

-- name: xol_update_master
--
-- # Kenapa TYPEXOL ikut diubah, sementara procedure lama tidak
--
-- `INSERT_UPDATE_MST_XOL.prc:39` hanya menyetel NAMA, TAHUN, dan KURSVALUE pada cabang
-- update — TYPEXOL tertinggal, meski layarnya menampilkan isian itu dan mengirim
-- nilainya. Akibatnya: mengubah Type XOL pada induk yang sudah ada TIDAK PERNAH tersimpan
-- di sistem lama.
--
-- Itu cacat, bukan aturan: parameter `TTypes` diterima procedure lalu dibuang pada cabang
-- update, sementara cabang insert memakainya. Di sini ia ikut disimpan, dan selisihnya
-- dicatat terbuka — layar akan mulai menyimpan perubahan yang dulu hilang diam-diam.
UPDATE POOLDATA.MST_XOL_PNC
   SET NAMA = :1,
       TAHUN = :2,
       KURSVALUE = :3,
       TYPEXOL = :4
 WHERE ID = :5

-- name: xol_business_count
--
-- Menghitung baris bisnis yang sama sebelum menyisipkan, meniru
-- `INSERT_UPDATE_MST_XOL.prc:52` yang memeriksa `count(1)` lebih dulu. Pemeriksaan ini
-- ada di kode, bukan di basis data, karena MST_XOL_BUSINESS tidak punya indeks unik.
SELECT COUNT(*)
  FROM POOLDATA.MST_XOL_BUSINESS
 WHERE ID = :1
   AND IDBUSINESS = :2

-- name: xol_insert_business
INSERT INTO POOLDATA.MST_XOL_BUSINESS (ID, GROUPBUSINESS, IDBUSINESS)
VALUES (:1, :2, :3)

-- name: xol_insert_layer
INSERT INTO POOLDATA.MST_XOL_LAYER (ID, IDLAYER, NAMA, "LIMIT", EXCESS, CONVERT_LIMIT)
VALUES (:1, :2, :3, :4, :5, :6)

-- name: xol_update_layer
--
-- Kolom ID sengaja TIDAK ikut diubah: sebuah lapisan tidak pernah berpindah induk, dan
-- membiarkannya berpindah akan memutus baris reas yang menunjuk IDLAYER itu.
UPDATE POOLDATA.MST_XOL_LAYER
   SET NAMA = :1,
       "LIMIT" = :2,
       EXCESS = :3,
       CONVERT_LIMIT = :4
 WHERE IDLAYER = :5

-- name: xol_reas_count
--
-- Meniru `INSERT_UPDATE_MST_XOL.prc:110`, yang memutuskan update atau insert berdasarkan
-- `count(1)` atas pasangan IDLAYER + IDREAS.
SELECT COUNT(*)
  FROM POOLDATA.MST_XOL_REAS
 WHERE IDLAYER = :1
   AND IDREAS = :2

-- name: xol_insert_reas
INSERT INTO POOLDATA.MST_XOL_REAS (IDLAYER, NAMA, IDREAS, PERCENTSHARE)
VALUES (:1, :2, :3, :4)

-- name: xol_update_reas
--
-- # Kenapa NAMA ikut diubah, sementara procedure lama tidak
--
-- `INSERT_UPDATE_MST_XOL.prc:115` hanya menyetel PERCENTSHARE pada cabang update, dan
-- membiarkan NAMA apa adanya. Akibatnya nama reasuradur yang dibetulkan pengguna tidak
-- pernah tersimpan selama ID-nya tidak berubah.
--
-- Sama seperti TYPEXOL di atas: ini cacat, bukan aturan, dan selisihnya dicatat terbuka.
UPDATE POOLDATA.MST_XOL_REAS
   SET NAMA = :1,
       PERCENTSHARE = :2
 WHERE IDLAYER = :3
   AND IDREAS = :4

-- name: xol_delete_master
DELETE FROM POOLDATA.MST_XOL_PNC
 WHERE ID = :1

-- name: xol_delete_business_of_master
DELETE FROM POOLDATA.MST_XOL_BUSINESS
 WHERE ID = :1

-- name: xol_delete_reas_of_master
--
-- Dijalankan SEBELUM lapisannya dihapus: begitu barisnya hilang, tidak ada lagi cara
-- mengetahui IDLAYER mana yang milik induk ini. Urutan inilah yang membuat kaskade benar.
DELETE FROM POOLDATA.MST_XOL_REAS
 WHERE IDLAYER IN (SELECT IDLAYER
                     FROM POOLDATA.MST_XOL_LAYER
                    WHERE ID = :1)

-- name: xol_delete_layer_of_master
DELETE FROM POOLDATA.MST_XOL_LAYER
 WHERE ID = :1

-- name: xol_delete_business
DELETE FROM POOLDATA.MST_XOL_BUSINESS
 WHERE ID = :1
   AND IDBUSINESS = :2

-- Tidak ada kueri untuk menghapus baris bisnis yang IDBUSINESS-nya NULL, dan itu
-- disengaja. `= NULL` tidak pernah bernilai benar, sehingga kueri di atas tidak akan
-- pernah menyentuhnya — dan layar Pega pun demikian, karena kueri hapusnya memakai
-- pencocokan yang sama. Dua baris seperti itu benar-benar ada di produksi (induk 10009)
-- dan memang sudah tidak dapat dihapus dari layar lama hari ini.
--
-- Batasannya dipertahankan (`P-5`); pembersihannya menempuh jalur DBA (`D-63`). Sebuah
-- kueri yang dapat menghapusnya sempat ditulis lalu DIBUANG: ia tidak dipanggil dari mana
-- pun, dan kueri hapus yang menganggur adalah kode mati yang berbahaya.

-- name: xol_delete_layer
DELETE FROM POOLDATA.MST_XOL_LAYER
 WHERE ID = :1
   AND IDLAYER = :2

-- name: xol_delete_reas_of_layer
DELETE FROM POOLDATA.MST_XOL_REAS
 WHERE IDLAYER = :1

-- name: xol_delete_reas
DELETE FROM POOLDATA.MST_XOL_REAS
 WHERE IDLAYER = :1
   AND IDREAS = :2

-- name: xol_layer_owner
--
-- Memastikan sebuah lapisan benar-benar milik induk yang disebut sebelum ia — atau
-- reas-nya — dihapus. Tanpa ini, permintaan hapus dapat menyebut induk A dan lapisan
-- milik induk B, dan penghapusannya tetap berjalan.
SELECT IDLAYER
  FROM POOLDATA.MST_XOL_LAYER
 WHERE ID = :1
   AND IDLAYER = :2

-- name: xol_submit_committee
--
-- Menggantikan pernyataan yang di sistem lama DIRANGKAI SEBAGAI TEKS lalu dijalankan:
-- `Activity/UpdateStatusMasterKomitexol-Act.xml` menyusun
-- `"update POOLDATA.MST_XOL_PNC set PIC='" + Local.userlogin + "' ..."` dari nilai yang
-- berasal dari sesi dan dari isian pengguna. Catatan PIC yang memuat satu tanda kutip
-- tunggal sudah cukup untuk mematahkannya.
--
-- Di sini ketiganya parameter terikat, sehingga isinya tidak pernah menjadi bagian dari
-- teks SQL (`docs/Steering/11-SECURITY.md` §4.1).
UPDATE POOLDATA.MST_XOL_PNC
   SET PIC = :1,
       STSKOMITE = :2,
       REMARKPIC = :3
 WHERE ID = :4

-- name: xol_year_list
--
-- Pilihan Tahun. Sumbernya POOLDATA.M_TREATYYEAR — lihat masterxol.Repo.ListYear untuk
-- alasannya dan untuk batas keyakinannya.
--
-- TANPA ORDER BY; pengurutannya di Go, supaya tidak bergantung pada aturan pengurutan
-- teks yang berbeda antara Oracle dan PostgreSQL.
SELECT DISTINCT OLD_THN_TREATY
  FROM POOLDATA.M_TREATYYEAR
 WHERE OLD_THN_TREATY IS NOT NULL

-- name: xol_business_group_list
--
-- Pilihan grup bisnis menurut Type XOL.
--
-- Menggantikan `RDB List/GetDataBisnisXol_Sql-SQL.xml`, yang menerima potongan klausa
-- WHERE lewat `{Asis:TempXOL.AgentID}` — perangkaian teks SQL. Di sini ketiga polanya
-- menjadi PARAMETER TERIKAT, sehingga bentuk kuerinya tetap dan isinya tidak pernah
-- menjadi bagian dari teksnya.
--
-- Selalu tiga pola, bahkan ketika aturannya hanya menyebut dua: yang terakhir diulang,
-- dan `A OR B OR B` bernilai sama dengan `A OR B`. Itu yang membuat kuerinya dapat tetap
-- berupa satu teks tetap. Lihat masterxol.BusinessGroupPattern.
SELECT b.BUSINESSGROUPID,
       b.BUSINESSGROUPNAME
  FROM POOLDATA.BUSINESS b
 WHERE b.BUSINESSGROUPID IN (
           SELECT g.ID
             FROM POOLDATA.BUSINESSGROUP g
            WHERE g.TOPID IN (
                      SELECT DISTINCT p.TREATYGROUPID
                        FROM POOLDATA.PROPORTIONALARRG p
                       WHERE p.TREATYGROUPNAME LIKE :1
                          OR p.TREATYGROUPNAME LIKE :2
                          OR p.TREATYGROUPNAME LIKE :3))
 GROUP BY b.BUSINESSGROUPID, b.BUSINESSGROUPNAME

-- name: xol_orphan_count
--
-- Menghitung baris anak yang induknya sudah tidak ada.
--
-- Ia BUKAN bagian dari jalur layar — hanya mode periksa yang memanggilnya. Alasannya
-- konkret: `Activity/DeleteFromTabelMst-Act.xml.xml` menghapus satu tabel per pemanggilan,
-- sehingga induk yang dibuang dari layar lama meninggalkan anaknya. Induk 10003 sudah
-- terhapus di produksi tetapi lapisan dan baris bisnisnya masih ada, dan baris seperti itu
-- tidak dapat dicapai layar mana pun.
--
-- Aplikasi ini berkaskade sehingga tidak menambah yatim baru; yang sudah ada tetap perlu
-- dilaporkan supaya pembersihannya dapat diajukan ke DBA (`D-63`).
-- Bentuknya UNION ALL atas tiga agregat, bukan tiga subkueri pada satu baris. Yang
-- terakhir menuntut sebuah tabel untuk digantungi — `FROM DUAL` di Oracle, `ROWNUM = 1`
-- untuk membatasinya — dan keduanya dilarang disiplin SQL portabel. Agregat selalu
-- mengembalikan tepat satu baris tanpa perlu digantungi apa pun.
SELECT 'layer' AS JENIS,
       COUNT(*) AS JUMLAH
  FROM POOLDATA.MST_XOL_LAYER l
 WHERE NOT EXISTS (SELECT 1 FROM POOLDATA.MST_XOL_PNC p WHERE p.ID = l.ID)
UNION ALL
SELECT 'bisnis',
       COUNT(*)
  FROM POOLDATA.MST_XOL_BUSINESS b
 WHERE NOT EXISTS (SELECT 1 FROM POOLDATA.MST_XOL_PNC p WHERE p.ID = b.ID)
UNION ALL
SELECT 'reas',
       COUNT(*)
  FROM POOLDATA.MST_XOL_REAS r
 WHERE NOT EXISTS (SELECT 1 FROM POOLDATA.MST_XOL_LAYER l WHERE l.IDLAYER = r.IDLAYER)

-- name: xol_check_table
--
-- Memastikan keempat tabel ada dan kolomnya dapat dibaca akun aplikasi, tanpa mengambil
-- satu baris pun. Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak
-- mirip tetapi perbaikannya berbeda jauh: tabelnya tidak ada di portal itu, versus akun
-- aplikasi tidak punya hak baca atas tabel warisan.
SELECT p.ID,
       b.IDBUSINESS,
       l.IDLAYER,
       l."LIMIT",
       r.IDREAS
  FROM POOLDATA.MST_XOL_PNC p,
       POOLDATA.MST_XOL_BUSINESS b,
       POOLDATA.MST_XOL_LAYER l,
       POOLDATA.MST_XOL_REAS r
 WHERE 1 = 0
