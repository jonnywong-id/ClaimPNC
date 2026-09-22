-- Kueri master COL Simas Online: POOLDATA.M_CAUSE_OF_LOSS dan tabel pemetaan bisnisnya.
--
-- # Tabel mana yang dibaca, dan tabel mana yang ditulis
--
-- Keduanya tabel DASAR — M_CAUSE_OF_LOSS dan M_CAUSE_OF_LOSS_BUSINESS — bukan view
-- POOLDATA.V_M_CAUSE_OF_LOSS. View itu tetap ada dan tetap dibaca rule Pega selama masa
-- paralel; aplikasi ini tidak menyentuhnya. Membaca dari tabel dasar membuat apa yang
-- kita tulis dan apa yang kita baca kembali pasti sama, tanpa bergantung pada definisi
-- view yang dimiliki pihak lain. Pola ini sama dengan modul Master Status Klaim.
--
-- POOLDATA.BUSINESS hanya DIBACA. Ia milik GISFW (`D-03`), dan tidak ada satu pun
-- pernyataan tulis terhadapnya di berkas ini.
--
-- # Kenapa tidak lagi lewat PEGA_M_CAUSE_OF_LOSS
--
-- `D-02` menetapkan logika stored procedure naik ke Go dan aplikasi tidak memanggil
-- procedure. `D-68` menambahkan alasan yang lebih keras: kontrak galat procedure itu
-- tidak dapat dipakai — parameter keluarannya bernama `ErrMsg`, tetapi pada jalur
-- BERHASIL ia berisi kalimat "Data Sudah Disimpan dengan ID : 1001"
-- (`Database/PEGA_M_CAUSE_OF_LOSS.prc:23`), sehingga pemanggil tidak dapat membedakan
-- berhasil dari gagal tanpa membaca teks. Procedure itu juga COMMIT sendiri di dalam
-- cabang INSERT sementara ROLLBACK-nya berada di handler terluar yang berjalan SESUDAH
-- commit — sehingga tidak memulihkan apa pun.
--
-- # Yang berubah dari penyimpanan lama: JSON tidak dipakai lagi
--
-- Keputusan Work Owner 2026-09-21: nilai disimpan LANGSUNG KE KOLOM, bukan ke dokumen
-- JSON. Sistem lama menyimpan seluruh baris sebagai satu CLOB `JSON_DATA` lalu
-- membongkarnya kembali lewat view — pola yang sama dengan M_STS_CLAIM sebelum migrasi
-- 0002, dan diperlakukan sama di sini. Kolom dan tabel yang dibutuhkan dibuat migrasi
-- 0003.
--
-- Empat aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (`D-20`).
--   4. Tanpa pemanggilan stored procedure (`D-02`).


-- name: cause_of_loss_list
--
-- Asal: `Report Definition/BrowseVMCauseOfLoss_RD-RD.xml` — sumber grid layar Simas
-- Online, dengan kolom .M_COL_ID, .OLD_M_COL_ID, dan .COL_DESC.
--
-- OLD_M_COL_ID TIDAK ikut dibaca. Di sistem lama kolom itu menampung dokumen JSON yang
-- dibongkar layar untuk memperoleh MST_COL_ID dan BISNISID
-- (`Activity/SetDataCauseofflossOnline-Act.xml`). Sejak penyimpanannya pindah ke kolom,
-- keduanya dibaca dari kolom dan tabel pemetaannya sendiri — yang jauh lebih penting,
-- keduanya menjadi dapat disaring dan diurutkan basis data.
--
-- Pemetaan bisnis sengaja tidak ikut di sini: grid layar hanya menampilkan ID dan
-- Description (`Section/Online_GridCauseOfLoss-Section.xml`), dan menariknya untuk
-- seluruh baris berarti satu kueri yang hasilnya tidak pernah dilihat siapa pun.
SELECT M_COL_ID,
       COL_DESC,
       MST_COL_ID
  FROM POOLDATA.M_CAUSE_OF_LOSS
 ORDER BY M_COL_ID

-- name: cause_of_loss_get
--
-- Satu baris untuk dimuat ke form sunting.
SELECT M_COL_ID,
       COL_DESC,
       MST_COL_ID
  FROM POOLDATA.M_CAUSE_OF_LOSS
 WHERE M_COL_ID = :1

-- name: cause_of_loss_business_list
--
-- Pemetaan bisnis satu penyebab kerugian, beserta nama bisnisnya.
--
-- Asal bentuknya: `RDB List/GetLBUID_SQL-SQL.xml`
--
--   select ID, NOTE as "Note" from V_D_CAUSE_OF_LOSS_BUSINESS a, BUSINESS b
--    where a.BISNISID = b.ID and D_COL_ID = {InputCOL.D_COL_ID}
--
-- # Kenapa TIDAK di-join ke POOLDATA.BUSINESS sama sekali
--
-- Karena namanya sudah tersimpan di NAMA_BISNIS, dan nama itulah yang benar-benar
-- terikat di layar Pega (`pyValue = .Note`). Isian Bisnis ber-`pyAllowFreeFormInput=true`
-- (Work Owner 2026-09-21), sehingga sebagian baris memang TIDAK punya BISNISID — dan
-- join apa pun terhadap baris itu tidak akan menemukan apa-apa.
--
-- Kueri lama yang men-join implisit lewat WHERE justru MEMBUANG baris seperti itu: ia
-- hilang dari layar tanpa satu pun tanda, dan pengguna menyimpan ulang tanpa sadar telah
-- kehilangan satu pemetaan. Membaca nama dari kolomnya sendiri menutup kelas cacat itu
-- seluruhnya, sekaligus menghemat satu join.
--
-- URUTAN mempertahankan susunan yang disusun pengguna di grid. Mengurutkannya menurut
-- BISNISID akan menempatkan baris tanpa ID di satu ujung, dan layar menampilkan urutan
-- yang berbeda dari yang baru saja disimpan.
--
-- STS_AKTIF menyaring baris yang sudah tidak berlaku — lihat catatan pada
-- cause_of_loss_business_deactivate_all.
SELECT BISNISID,
       NAMA_BISNIS,
       URUTAN
  FROM POOLDATA.M_CAUSE_OF_LOSS_BUSINESS
 WHERE M_COL_ID = :1
   AND STS_AKTIF = '1'
 ORDER BY URUTAN

-- name: cause_of_loss_site
--
-- Kode situs, bagian pertama dari setiap M_COL_ID. Meniru
-- `Database/PEGA_M_CAUSE_OF_LOSS.prc:12` persis, termasuk pembandingnya yang berupa
-- teks '1' dan bukan angka.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: cause_of_loss_next_sequence
--
-- Urutan yang SAMA dengan yang dipakai procedure lama
-- (`Database/PEGA_M_CAUSE_OF_LOSS.prc:19`), supaya kode yang diterbitkan aplikasi ini
-- melanjutkan deret yang sudah ada dan tidak pernah bertabrakan dengan kode yang pernah
-- diterbitkan Pega.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada `ADR-0005`, satu-satunya tempat
-- lain yang dibenarkan memuat percabangan dialek.
SELECT POOLDATA.M_CAUSE_SEQ.NEXTVAL
  FROM DUAL

-- name: cause_of_loss_insert
--
-- JSON_DATA sengaja TIDAK diisi. Sejak Work Owner menetapkan penyimpanan pindah ke
-- kolom, kolom itu menjadi peninggalan: ia dibiarkan berisi nilai terakhir yang ditulis
-- Pega pada baris lama, dan kosong pada baris baru. Alasan lengkapnya ada di migrasi
-- 0004 langkah 7, mengikuti perlakuan JSONDATA pada migrasi 0002.
INSERT INTO POOLDATA.M_CAUSE_OF_LOSS (M_COL_ID, COL_DESC, MST_COL_ID)
VALUES (:1, :2, :3)

-- name: cause_of_loss_update
--
-- M_COL_ID hanya menyaring, tidak pernah ikut di-SET — sama seperti
-- `Database/PEGA_M_CAUSE_OF_LOSS.prc:34`. Ia dirujuk D_CAUSE_OF_LOSS.M_COL_ID pada data
-- yang sudah berjalan.
UPDATE POOLDATA.M_CAUSE_OF_LOSS
   SET COL_DESC   = :1,
       MST_COL_ID = :2
 WHERE M_COL_ID = :3

-- name: cause_of_loss_business_deactivate_all
--
-- Menandai SELURUH pemetaan bisnis satu penyebab kerugian sebagai tidak berlaku,
-- sebelum pilihan yang baru ditandai berlaku kembali.
--
-- # Kenapa ditandai, bukan dihapus
--
-- `D-66` menetapkan tidak ada penghapusan fisik pada data bernilai bisnis, dan secara
-- khusus mencabut pola **hapus-lalu-sisip-ulang** yang dipakai
-- `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503`. Menyimpan pilihan grid dengan
-- DELETE lalu INSERT adalah pola yang sama persis, hanya dalam ukuran kecil — dan
-- penggantinya memang belum diputuskan secara umum (`ADR-0013`).
--
-- Penanda dipilih, bukan versioning, karena bentuknya sudah ada di sistem lama:
-- `V_D_CAUSE_OF_LOSS` memuat kolom `STS_AKTIF`, sehingga penandaan aktif/tidak-aktif
-- adalah cara yang memang sudah dipakai domain ini untuk data COL.
--
-- Konsekuensi yang mengikat: SETIAP kueri pembaca wajib menyaring `STS_AKTIF = '1'`.
-- Satu kueri yang lupa akan menampilkan pemetaan yang seharusnya sudah hilang.
UPDATE POOLDATA.M_CAUSE_OF_LOSS_BUSINESS
   SET STS_AKTIF = '0'
 WHERE M_COL_ID = :1

-- name: cause_of_loss_business_activate
--
-- Mengisi ulang baris pemetaan pada satu POSISI grid, sekaligus menghidupkannya kembali.
--
-- Dijalankan lebih dulu, dan hanya bila ia mengenai NOL baris barulah penyisipan
-- dilakukan. Ini upsert yang portabel: MERGE didukung keduanya tetapi sintaksnya berbeda
-- cukup jauh, sedangkan `INSERT ... ON CONFLICT` hanya ada di PostgreSQL.
--
-- # Kenapa kuncinya URUTAN, bukan NAMA_BISNIS dan bukan BISNISID
--
-- Ketiganya sempat dipertimbangkan, dan dua gugur karena bukti:
--
--   BISNISID   boleh NULL — nama yang diketik bebas tidak punya ID sama sekali.
--   NAMA_BISNIS  TIDAK unik. Pemeriksaan ulang 2026-09-21 membuktikan grid Pega tidak
--                punya satu pun penanda keunikan, sehingga satu bisnis memang boleh
--                dipilih dua kali — dan Work Owner menetapkan perilaku itu dipertahankan.
--
-- Yang tersisa adalah posisinya di dalam grid, dan itu memang identitas yang benar:
-- halaman `TempCauseOfLoss.BISNISID` di Pega adalah PageList, yang barisnya pun dikenali
-- lewat nomor urutnya.
UPDATE POOLDATA.M_CAUSE_OF_LOSS_BUSINESS
   SET STS_AKTIF   = '1',
       BISNISID    = :1,
       NAMA_BISNIS = :2
 WHERE M_COL_ID = :3
   AND URUTAN   = :4

-- name: cause_of_loss_business_insert
--
-- Dijalankan hanya bila cause_of_loss_business_activate tidak mengenai satu baris pun —
-- yaitu ketika grid bertambah panjang daripada yang pernah tersimpan.
--
-- BISNISID boleh NULL — lihat catatan pada cause_of_loss_business_activate.
INSERT INTO POOLDATA.M_CAUSE_OF_LOSS_BUSINESS (M_COL_ID, BISNISID, NAMA_BISNIS, URUTAN, STS_AKTIF)
VALUES (:1, :2, :3, :4, '1')

-- name: business_list
--
-- Daftar bisnis untuk pilihan di layar.
--
-- Asal kolomnya: `RDB List/GetLBUID_SQL-SQL.xml` yang membaca `BUSINESS` dengan
-- `ID` dan `NOTE`. Tabel ini milik GISFW dan hanya dibaca (`D-03`).
--
-- Diurutkan menurut NOTE, bukan ID: pengguna mencari bisnis dengan namanya, dan urutan
-- kode tidak berarti apa-apa baginya. Pengurutan di basis data, bukan di Go, supaya
-- daftar panjang tidak perlu dimuat seluruhnya ke memori lebih dulu.
SELECT ID,
       NOTE
  FROM POOLDATA.BUSINESS
 ORDER BY NOTE

-- name: cause_of_loss_check_table
--
-- Memastikan kolom baru migrasi 0004 sudah ada dan dapat dibaca akun aplikasi, tanpa
-- mengambil satu baris pun. Dipakai mode periksa untuk membedakan dua sebab kegagalan
-- yang tampak mirip: migrasi belum dijalankan versus tidak punya hak baca.
SELECT M_COL_ID,
       COL_DESC,
       MST_COL_ID
  FROM POOLDATA.M_CAUSE_OF_LOSS
 WHERE 1 = 0

-- name: cause_of_loss_business_check_table
--
-- Memastikan tabel pemetaan yang dibuat migrasi 0004 ada dan dapat dibaca.
SELECT M_COL_ID,
       BISNISID,
       NAMA_BISNIS,
       URUTAN,
       STS_AKTIF
  FROM POOLDATA.M_CAUSE_OF_LOSS_BUSINESS
 WHERE 1 = 0

-- name: business_check_table
--
-- Memastikan master bisnis milik GISFW dapat dibaca akun aplikasi. Hak bacanya belum
-- tentu ada — sampai modul ini, aplikasi tidak pernah menyentuh tabel itu.
SELECT ID,
       NOTE
  FROM POOLDATA.BUSINESS
 WHERE 1 = 0
