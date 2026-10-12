-- ============================================================================
-- DICABUT 2026-09-23 — JANGAN DIJALANKAN.
-- ============================================================================
--
-- Berkas ini menyasar POOLDATA.M_CAUSE_OF_LOSS, dan itu TABEL YANG SALAH untuk layar
-- Master COL Simas Online.
--
-- Empat bukti yang menetapkannya:
--
--   1. Jalur Simpan layar Simas Online (`Activity/Online_nsertCauseOfLoss_act-Act.xml`)
--      memanggil `UpdateMCauseOfLoss_online`, BUKAN `UpdateMCauseOfLoss`.
--   2. `Database/PEGA_M_CAUSE_OF_LOSS_ONLINE` menulis POOLDATA.M_CAUSE_OF_LOSS_ONLINE
--      dan memakai urutan M_CAUSE_SEQ_ONLINE.
--   3. Dari 55 induk yang dirujuk POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL, **55 cocok**
--      dengan M_CAUSE_OF_LOSS_ONLINE dan hanya **1** cocok dengan M_CAUSE_OF_LOSS.
--   4. Keduanya sudah ada dan sudah berisi data: 55 induk dan 282 baris pemetaan.
--
-- Karena kedua tabel yang benar SUDAH ADA beserta kolomnya, tidak ada satu pun DDL yang
-- dibutuhkan modul ini. Kolom MST_COL_ID yang hendak ditambahkan berkas ini pun sudah
-- dicabut seluruhnya atas keputusan Work Owner 2026-09-23.
--
-- Berkas ini DISIMPAN, bukan dihapus, karena ia sempat diserahkan ke DBA — dan menghapus
-- berkas yang sudah beredar hanya membuat orang menjalankan salinan lamanya tanpa
-- peringatan ini.
--
-- Satu-satunya perubahan skema yang masih diminta modul ini ada di
-- `0007_col_simas_online_sts_aktif.up.sql`.
--
-- ============================================================================
-- Isi asli berkas ini dibiarkan apa adanya di bawah, sebagai rekaman.
-- ============================================================================


-- 0004 — Master COL Simas Online: isi pindah dari JSON ke kolom (Oracle 19c)
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- Berkas ini MENGUBAH objek milik sistem lama yang sedang melayani produksi:
--
--   * POOLDATA.M_CAUSE_OF_LOSS          — dua kolom ditambahkan dan diisi
--   * POOLDATA.M_CAUSE_OF_LOSS_BUSINESS — tabel BARU
--   * POOLDATA.V_M_CAUSE_OF_LOSS        — view-nya didefinisikan ulang
--
-- Menjalankannya menuntut permintaan perubahan skema tertulis, persetujuan Work Owner,
-- pelaksanaan oleh DBA, dan pengujian dengan MENJALANKAN PEGA DAN GO BERSAMAAN terhadap
-- skema hasil perubahan (`D-63`). Akun aplikasi tidak memiliki hak DDL.
--
-- BERKAS INI BELUM PERNAH DIJALANKAN DI LINGKUNGAN MANA PUN.
--
--
-- ## PERINGATAN — berkas ini ditulis TANPA melihat katalog
--
-- Ini perbedaan penting dari migrasi 0002, dan harus disadari sebelum menyetujui.
--
-- Migrasi 0002 mula-mula ditulis dengan menebak isi skema dari source procedure, dan
-- tebakan itu ternyata SALAH di dua tempat — kedua kolom yang hendak ditambahkannya
-- ternyata sudah ada, sehingga ALTER TABLE-nya akan gagal dengan ORA-01430 dan
-- menghentikan seluruh migrasi di baris pertama. Yang menyelamatkannya adalah pembacaan
-- katalog yang dilakukan sesudahnya.
--
-- Pembacaan katalog yang setara BELUM dilakukan untuk tabel ini. Yang diketahui hanya:
--
--   * Dari `Database/PEGA_M_CAUSE_OF_LOSS.prc:22`, tabelnya punya sekurang-kurangnya
--     dua kolom — M_COL_ID dan JSON_DATA.
--   * Dari `Report Definition/BrowseVMCauseOfLoss_RD-RD.xml`, view V_M_CAUSE_OF_LOSS
--     mengeluarkan tiga kolom: M_COL_ID, OLD_M_COL_ID, COL_DESC.
--   * Dari `Activity/SetDataCauseofflossOnline-Act.xml`, layar Simas Online membaca
--     OLD_M_COL_ID lalu MEM-PARSE-NYA SEBAGAI JSON untuk memperoleh MST_COL_ID dan
--     BISNISID. Jadi pada view yang berlaku sekarang, OLD_M_COL_ID berisi dokumen JSON —
--     bukan kode lama seperti namanya menyiratkan.
--
-- Nama kolom COL_DESC dan MST_COL_ID di bawah karena itu mengikuti nama property Pega,
-- dan kunci JSON-nya diturunkan dari nama property yang sama. KEDUANYA HARUS
-- DIVERIFIKASI DBA terhadap katalog sebelum dijalankan — lihat langkah 0.
--
--
-- ## Kenapa perubahan ini diminta
--
-- Sistem lama menyimpan seluruh baris sebagai satu dokumen JSON
-- (M_CAUSE_OF_LOSS.JSON_DATA), lalu membongkarnya kembali lewat view. Yang menulisnya
-- adalah procedure POOLDATA.PEGA_M_CAUSE_OF_LOSS.
--
-- Tiga keputusan yang sudah disetujui menutup jalan itu:
--
--   D-02  Logika stored procedure naik ke Go; aplikasi tidak memanggil procedure.
--   D-68  Kepemilikan transaksi pindah ke Go. Kontrak galat procedure lama tidak
--         dibawa: parameter keluarannya bernama ErrMsg tetapi pada jalur BERHASIL ia
--         berisi kalimat "Data Sudah Disimpan dengan ID : 1001", sehingga pemanggil
--         tidak dapat membedakan berhasil dari gagal tanpa membaca teks.
--   —     Work Owner 2026-09-21: penyimpanan tidak lagi memakai JSON; nilai disimpan
--         langsung ke kolom, di tempat yang sama seperti Pega.
--
-- Perlakuannya sama persis dengan yang sudah dijalankan migrasi 0002 untuk
-- M_STS_CLAIM — termasuk keputusan membiarkan kolom JSON lamanya tetap ada.
--
-- P-1 tetap dipatuhi — satu tabel satu penulis. Layar Master COL adalah SATU-SATUNYA
-- penulis tabel ini di sistem lama (`RDB List/UpdateMCauseOfLoss-SQL.xml`, pemanggil
-- tunggal PEGA_M_CAUSE_OF_LOSS), sehingga memindahkan layarnya memindahkan kepemilikan
-- tabelnya secara utuh. Pega berubah menjadi pembaca saja.
--
--
-- ## Yang HARUS dilakukan DBA sebelum menjalankan — LANGKAH 0
--
-- 1. SIMPAN DEFINISI VIEW YANG SEKARANG, beserta daftar kolomnya. Migrasi turun
--    membutuhkannya, dan ALL_VIEWS.TEXT tidak menyimpan daftar kolom view:
--
--        SELECT DBMS_METADATA.GET_DDL('VIEW', 'V_M_CAUSE_OF_LOSS', 'POOLDATA') FROM DUAL;
--
--    Simpan hasilnya ke berkas dan lampirkan pada permintaan perubahan skema.
--    Definisi itu juga yang MEMASTIKAN kunci JSON yang dipakai langkah 1 di bawah.
--
-- 2. BACA DAFTAR KOLOM TABEL DASARNYA. Bila COL_DESC atau MST_COL_ID ternyata SUDAH ADA
--    — seperti yang terjadi pada LSC_NOTE di migrasi 0002 — hapus ALTER TABLE yang
--    bersangkutan, jangan dijalankan:
--
--        SELECT COLUMN_NAME, DATA_TYPE, DATA_LENGTH, NULLABLE
--          FROM ALL_TAB_COLUMNS
--         WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'M_CAUSE_OF_LOSS'
--         ORDER BY COLUMN_ID;
--
--    Sekalian catat LEBAR M_COL_ID. Kode dibentuk `kode_situs || lpad(urutan, 3, '0')`,
--    sehingga penyisipan ke-1000 menghasilkan kode LIMA karakter. Bila kolomnya CHAR(4)
--    seperti LSC_ID, penyisipan itu akan ditolak ORA-12899 — dan keputusan memperlebar
--    kolom sebaiknya diambil sebelum, bukan sesudah, penyisipan pertama yang gagal.
--
-- 3. PERIKSA POSISI URUTAN. Kode yang diterbitkan aplikasi harus MELANJUTKAN deret yang
--    sudah ada, tidak menabraknya:
--
--        SELECT LAST_NUMBER FROM ALL_SEQUENCES
--         WHERE SEQUENCE_OWNER = 'POOLDATA' AND SEQUENCE_NAME = 'M_CAUSE_SEQ';
--
--        SELECT MAX(M_COL_ID) FROM POOLDATA.M_CAUSE_OF_LOSS;
--
-- 4. PERIKSA PANJANG NILAI YANG ADA. Langkah 1 akan gagal bila ada nilai yang lebih
--    panjang dari lebar kolom tujuannya:
--
--        SELECT COUNT(*) FROM POOLDATA.M_CAUSE_OF_LOSS
--         WHERE LENGTH(JSON_VALUE(JSON_DATA, '$.COL_DESC'))   > 100
--            OR LENGTH(JSON_VALUE(JSON_DATA, '$.MST_COL_ID')) > 20;
--
--    Yang diharapkan nol baris. Bila tidak nol, lebarkan kolomnya di langkah 1 dan
--    sesuaikan MaxDescriptionLength / MaxMasterCodeLength di
--    `internal/mastercolsimasonline/mastercolsimasonline.go` — kedua angka itu memang
--    penjaga sementara sampai DDL ini ada.


-- ---------------------------------------------------------------------------
-- Langkah 1 — tambahkan kolom, lalu pindahkan isi JSON ke dalamnya.
--
-- JANGAN JALANKAN ALTER TABLE bila langkah 0 butir 2 menunjukkan kolomnya sudah ada.
--
-- Backward-compatible: selama langkah 4 belum jalan, view masih membaca JSON_DATA dan
-- Pega tidak terganggu sedetik pun (`P-4`).
-- ---------------------------------------------------------------------------

ALTER TABLE POOLDATA.M_CAUSE_OF_LOSS ADD (COL_DESC   VARCHAR2(100));
ALTER TABLE POOLDATA.M_CAUSE_OF_LOSS ADD (MST_COL_ID VARCHAR2(20));

UPDATE POOLDATA.M_CAUSE_OF_LOSS
   SET COL_DESC   = JSON_VALUE(JSON_DATA, '$.COL_DESC'),
       MST_COL_ID = JSON_VALUE(JSON_DATA, '$.MST_COL_ID')
 WHERE JSON_DATA IS NOT NULL;

COMMIT;


-- ---------------------------------------------------------------------------
-- Langkah 2 — tabel pemetaan COL ke Bisnis.
--
-- Di sistem lama pemetaan ini hidup DI DALAM dokumen JSON sebagai senarai BISNISID,
-- dan dibongkar view V_M_CAUSE_OF_LOSS_BUSINESS. Bentuk yang setara sudah ada di tingkat
-- detail — `RDB List/GetLBUID_SQL-SQL.xml` membaca V_D_CAUSE_OF_LOSS_BUSINESS dengan
-- kolom D_COL_ID dan BISNISID — dan nama kolom di bawah mengikuti bentuk itu.
--
-- STS_AKTIF ada karena `D-66`: tidak ada penghapusan fisik pada data bernilai bisnis.
-- Pemetaan yang dibuang pengguna DITANDAI, bukan dihapus. Penanda dipilih bukan karena
-- selera — ia bentuk yang memang sudah dipakai domain ini: V_D_CAUSE_OF_LOSS punya kolom
-- STS_AKTIF.
--
-- DEFAULT '1' supaya baris yang disisipkan tanpa menyebut kolom itu tetap aktif; kode Go
-- tetap menyebutnya eksplisit.
-- ---------------------------------------------------------------------------

-- # Kenapa NAMA_BISNIS ada, dan kenapa ia yang menjadi kunci
--
-- Isian Bisnis di Pega ber-`pyAllowFreeFormInput=true`
-- (`Section/Online_BrowseCauseOfLoss-Section.xml:6089`): petugas boleh mengetik nama
-- yang TIDAK ada di master, dan baris itu tersimpan tanpa ID. Work Owner menetapkan
-- 2026-09-21 perilaku itu dipertahankan.
--
-- Akibatnya BISNISID **boleh NULL**, sehingga ia tidak dapat menjadi bagian kunci. Yang
-- selalu ada adalah namanya — dan nama itu pula yang benar-benar terikat di layar Pega
-- (sel gridnya `pyValue = .Note`), sedangkan `.ID` hanya kolom tersembunyi yang ikut
-- terisi saat sebuah pilihan diambil dari daftar.
--
-- URUTAN menyimpan susunan baris sebagaimana pengguna menyusunnya di grid. Tanpa kolom
-- ini, satu-satunya pengurutan yang tersedia adalah BISNISID — yang akan menempatkan
-- seluruh baris tanpa ID di satu ujung, dan layar menampilkan urutan yang berbeda dari
-- yang baru saja disimpan.
-- # Kenapa kuncinya (M_COL_ID, URUTAN)
--
-- Ketiga kandidat dipertimbangkan, dan dua gugur karena bukti dari export:
--
--   BISNISID     boleh NULL — nama yang diketik bebas tidak punya ID.
--   NAMA_BISNIS  TIDAK unik — grid Pega tidak punya satu pun penanda keunikan, sehingga
--                satu bisnis memang boleh dipilih dua kali. Work Owner menetapkan
--                2026-09-21 perilaku itu dipertahankan.
--
-- Yang tersisa adalah POSISI baris di dalam grid, dan itu memang identitas yang benar:
-- `TempCauseOfLoss.BISNISID` di Pega adalah PageList, yang barisnya pun dikenali lewat
-- nomor urutnya.
--
-- URUTAN sekaligus menyimpan susunan sebagaimana pengguna menyusunnya. Tanpa kolom ini,
-- satu-satunya pengurutan yang tersedia adalah BISNISID — yang akan menempatkan seluruh
-- baris tanpa ID di satu ujung, dan layar menampilkan urutan yang berbeda dari yang baru
-- saja disimpan.
CREATE TABLE POOLDATA.M_CAUSE_OF_LOSS_BUSINESS (
    M_COL_ID    VARCHAR2(10)  NOT NULL,
    URUTAN      NUMBER(4)     NOT NULL,
    BISNISID    VARCHAR2(10)  NULL,
    NAMA_BISNIS VARCHAR2(100) NOT NULL,
    STS_AKTIF   CHAR(1)       DEFAULT '1' NOT NULL,
    CONSTRAINT M_CAUSE_OF_LOSS_BUSINESS_PK PRIMARY KEY (M_COL_ID, URUTAN)
);

-- Indeks pembaca. Kueri yang dipakai layar menyaring M_COL_ID dan STS_AKTIF bersamaan,
-- lalu mengurutkan menurut URUTAN. Kunci utama saja tidak cukup karena STS_AKTIF bukan
-- bagian darinya.
CREATE INDEX POOLDATA.IX_M_COL_BUSINESS_ACTIVE
    ON POOLDATA.M_CAUSE_OF_LOSS_BUSINESS (M_COL_ID, STS_AKTIF, URUTAN);

-- Kunci asing ke tabel induknya. TIDAK ada ON DELETE CASCADE, dan itu disengaja: `D-66`
-- melarang penghapusan fisik, sehingga cascade adalah jalur yang seharusnya tidak pernah
-- terpakai — dan menyediakannya berarti menyediakan jalan yang dilarang.
ALTER TABLE POOLDATA.M_CAUSE_OF_LOSS_BUSINESS
  ADD CONSTRAINT M_CAUSE_OF_LOSS_BUSINESS_FK
      FOREIGN KEY (M_COL_ID) REFERENCES POOLDATA.M_CAUSE_OF_LOSS (M_COL_ID);

-- Kunci asing ke POOLDATA.BUSINESS SENGAJA TIDAK DIBUAT, dan kali ini alasannya bahkan
-- lebih keras daripada kepemilikan tabel.
--
-- Nama bisnis boleh diketik bebas, sehingga BISNISID memang BOLEH NULL dan memang boleh
-- menunjuk bisnis yang tidak ada. Kunci asing akan menolak tepat baris yang Work Owner
-- putuskan harus diterima.
--
-- Di luar itu, tabel BUSINESS milik GISFW (`D-03`): memasang constraint terhadap tabel
-- milik tim lain berarti perubahan mereka dapat menggagalkan penyimpanan di sini tanpa
-- mereka tahu.


-- ---------------------------------------------------------------------------
-- Langkah 3 — pindahkan senarai bisnis dari JSON ke tabel.
--
-- JSON_TABLE membongkar senarai `$.BISNISID[*]` menjadi baris. Kunci `$.ID` di dalamnya
-- mengikuti nama property kelas ASM-FW-GISFW-Int-BUSINESS
-- (`Report Definition/BrowseBusiness_RD-RD.xml` memuat `.ID` dan `.Note`).
--
-- Kedua kolomnya diambil: `$.ID` dan `$.Note`. Autocomplete Pega memetakan keduanya —
-- `.Note` yang terlihat dan `.ID` yang tersembunyi tetapi ikut terisi saat sebuah pilihan
-- diambil dari daftar (`pySetValueOnSelect=true`, section baris 6150-6180).
--
-- Baris yang namanya DIKETIK BEBAS tidak punya `$.ID`, dan baris itu TETAP DISALIN —
-- dengan BISNISID NULL. Menyaringnya keluar berarti membuang pemetaan yang memang sah
-- menurut layar lama.
--
-- URUTAN diambil dari posisi elemen di dalam senarai JSON, supaya susunan yang dulu
-- disusun petugas tidak hilang.
--
-- VERIFIKASI DULU bentuk JSON-nya pada satu baris sebelum menjalankan ini:
--
--     SELECT JSON_DATA FROM POOLDATA.M_CAUSE_OF_LOSS WHERE ROWNUM = 1;
--
-- Bila senarainya tersimpan sebagai senarai teks polos (`["ANEKA","FIRE"]`) dan bukan
-- senarai objek, ganti kedua PATH menjadi `NAMA_BISNIS PATH '$'` dan buang kolom
-- BISNISID-nya.
-- ---------------------------------------------------------------------------

INSERT INTO POOLDATA.M_CAUSE_OF_LOSS_BUSINESS (M_COL_ID, BISNISID, NAMA_BISNIS, URUTAN, STS_AKTIF)
SELECT m.M_COL_ID, b.BISNISID, b.NAMA_BISNIS, b.URUTAN, '1'
  FROM POOLDATA.M_CAUSE_OF_LOSS m,
       JSON_TABLE(m.JSON_DATA, '$.BISNISID[*]'
                  COLUMNS (URUTAN      FOR ORDINALITY,
                           BISNISID    VARCHAR2(10)  PATH '$.ID',
                           NAMA_BISNIS VARCHAR2(100) PATH '$.Note')) b
 WHERE m.JSON_DATA IS NOT NULL
   AND b.NAMA_BISNIS IS NOT NULL;

COMMIT;


-- ---------------------------------------------------------------------------
-- Langkah 4 — VERIFIKASI. Jangan lanjut bila salah satu jawabannya tidak seperti yang
-- disebutkan. Setelah langkah 5 berjalan, kesalahan di sini tidak dapat lagi dideteksi
-- dengan membandingkan ke view.
-- ---------------------------------------------------------------------------

-- 4a. Berapa baris yang COL_DESC-nya masih kosong padahal JSON-nya terisi?  DIHARAPKAN: 0
--
--     SELECT COUNT(*) FROM POOLDATA.M_CAUSE_OF_LOSS
--      WHERE JSON_DATA IS NOT NULL AND COL_DESC IS NULL;

-- 4b. Adakah baris yang kolomnya berbeda dari view?  DIHARAPKAN: 0
--
--     SELECT COUNT(*)
--       FROM POOLDATA.M_CAUSE_OF_LOSS m
--       JOIN POOLDATA.V_M_CAUSE_OF_LOSS v ON v.M_COL_ID = m.M_COL_ID
--      WHERE NVL(TRIM(m.COL_DESC), '~') <> NVL(TRIM(v.COL_DESC), '~');
--
--     NVL dipakai di sini dengan sengaja meski COALESCE yang portabel: berkas ini
--     dijalankan DBA langsung di Oracle dan tidak pernah ikut ke PostgreSQL.

-- 4c. Apakah jumlah pemetaan bisnis sama dengan yang dibaca view lama?  DIHARAPKAN: sama
--
--     SELECT COUNT(*) FROM POOLDATA.M_CAUSE_OF_LOSS_BUSINESS WHERE STS_AKTIF = '1';
--     SELECT COUNT(*) FROM POOLDATA.V_M_CAUSE_OF_LOSS_BUSINESS;

-- 4d. Berapa pemetaan yang TIDAK punya ID bisnis?  Angka ini TIDAK harus nol.
--
--     SELECT COUNT(*) FROM POOLDATA.M_CAUSE_OF_LOSS_BUSINESS WHERE BISNISID IS NULL;
--
--     Ia menghitung baris yang namanya dulu DIKETIK BEBAS oleh petugas — keadaan sah
--     yang memang diizinkan layar lama (`pyAllowFreeFormInput=true`), bukan data rusak.
--     Laporkan angkanya ke Work Owner sebagai gambaran seberapa sering isian bebas
--     dipakai; itu bahan untuk memutuskan apakah kelak isiannya diperketat.

-- 4e. Berapa cause of loss yang memetakan satu bisnis LEBIH DARI SEKALI?
--
--     SELECT COUNT(*) FROM (
--       SELECT M_COL_ID, UPPER(TRIM(NAMA_BISNIS))
--         FROM POOLDATA.M_CAUSE_OF_LOSS_BUSINESS
--        GROUP BY M_COL_ID, UPPER(TRIM(NAMA_BISNIS))
--       HAVING COUNT(*) > 1);
--
--     Angka ini TIDAK harus nol dan TIDAK menghentikan migrasi: grid Pega memang tidak
--     pernah memeriksa keunikan, sehingga baris kembar adalah data sah. Laporkan
--     angkanya ke Work Owner sebagai gambaran — ia bahan untuk memutuskan apakah kelak
--     keunikan diberlakukan.


-- ---------------------------------------------------------------------------
-- Langkah 5 — definisikan ulang view supaya membaca kolom, bukan JSON.
--
-- Inilah pernyataan yang menyentuh Pega. Sesudah ini, apa yang ditulis aplikasi Go
-- langsung terlihat oleh rule yang membaca view ini.
--
-- Tiga hal yang WAJIB dipertahankan:
--
--   * URUTAN KOLOM: M_COL_ID, OLD_M_COL_ID, COL_DESC — sesuai
--     `Report Definition/BrowseVMCauseOfLoss_RD-RD.xml`. Menukarnya akan mengubah hasil
--     setiap pembacaan posisional.
--   * DAFTAR NAMA KOLOM ditulis eksplisit; tanpa itu kolom ketiga akan bernama
--     "JSON_VALUE(JSON_DATA,'$.COL_DESC')" dan rule Pega kehilangan kolom COL_DESC.
--   * CREATE OR REPLACE, bukan DROP lalu CREATE. Yang pertama mempertahankan seluruh
--     grant; yang kedua menghapusnya dan membuat Pega kehilangan hak baca.
--
-- ## Akibat yang HARUS disetujui sebelum langkah ini dijalankan
--
-- OLD_M_COL_ID tetap menyajikan JSON_DATA apa adanya, dan sejak aplikasi Go menulis,
-- JSON_DATA menjadi USANG. Artinya **layar Simas Online di Pega berhenti menampilkan
-- perubahan yang dibuat lewat aplikasi baru**: ia satu-satunya pembaca yang mem-parse
-- kolom itu (`Activity/SetDataCauseofflossOnline-Act.xml`).
--
-- Itu memang yang dikehendaki — layar itulah yang digantikan modul ini, dan `P-1`
-- menuntut tepat satu penulis. Tetapi ia berarti langkah 5 adalah TITIK CUTOVER layar
-- Simas Online, bukan sekadar perubahan teknis. Jangan jalankan sebelum modulnya lulus
-- gerbang 2.
--
-- Pembaca LAIN tidak terganggu: yang mereka baca adalah COL_DESC, dan COL_DESC justru
-- menjadi mutakhir oleh langkah ini.
-- ---------------------------------------------------------------------------

CREATE OR REPLACE VIEW POOLDATA.V_M_CAUSE_OF_LOSS (M_COL_ID, OLD_M_COL_ID, COL_DESC) AS
SELECT M_COL_ID,
       JSON_DATA,
       COL_DESC
  FROM POOLDATA.M_CAUSE_OF_LOSS;

-- View pemetaan bisnis ikut didefinisikan ulang supaya membaca tabel, bukan JSON.
-- Nama dan urutan kolomnya mengikuti V_D_CAUSE_OF_LOSS_BUSINESS yang sudah ada.
CREATE OR REPLACE VIEW POOLDATA.V_M_CAUSE_OF_LOSS_BUSINESS (M_COL_ID, BISNISID) AS
SELECT M_COL_ID,
       BISNISID
  FROM POOLDATA.M_CAUSE_OF_LOSS_BUSINESS
 WHERE STS_AKTIF = '1';


-- ---------------------------------------------------------------------------
-- Langkah 6 — hak akses untuk akun aplikasi.
--
-- Diberikan sesempit mungkin: SELECT, INSERT, dan UPDATE. TANPA DELETE — tidak ada satu
-- pun jalur di aplikasi yang menghapus baris master (`D-66`), dan hak yang tidak
-- diberikan tidak dapat disalahgunakan kode yang ditulis kemudian.
--
-- POOLDATA.BUSINESS hanya SELECT: ia milik GISFW dan tidak pernah ditulis modul ini.
-- Hak baca atasnya belum tentu ada — sampai modul ini, aplikasi tidak pernah
-- menyentuhnya.
--
-- Ganti <AKUN_APLIKASI> dengan nama akun yang sebenarnya.
-- ---------------------------------------------------------------------------

-- GRANT SELECT, INSERT, UPDATE ON POOLDATA.M_CAUSE_OF_LOSS          TO <AKUN_APLIKASI>;
-- GRANT SELECT, INSERT, UPDATE ON POOLDATA.M_CAUSE_OF_LOSS_BUSINESS TO <AKUN_APLIKASI>;
-- GRANT SELECT                  ON POOLDATA.BUSINESS                TO <AKUN_APLIKASI>;
-- GRANT SELECT                  ON POOLDATA.M_SITE_DATABASE         TO <AKUN_APLIKASI>;
-- GRANT SELECT                  ON POOLDATA.M_CAUSE_SEQ             TO <AKUN_APLIKASI>;


-- ---------------------------------------------------------------------------
-- Langkah 7 — JSON_DATA setelah migrasi ini.
--
-- Kolomnya SENGAJA TIDAK DIHAPUS dan tidak dikosongkan. Ia dibiarkan berisi nilai
-- terakhir yang ditulis Pega, sebagai bahan pembanding bila ada yang meragukan hasil
-- perpindahan ini, dan sebagai satu-satunya bahan untuk migrasi turun.
--
-- Kapan JSON_DATA boleh dibuang adalah keputusan tersendiri, dan sebaiknya diambil
-- setelah masa pengamatan berjalan — bukan di berkas ini. Perlakuannya sama dengan
-- JSONDATA pada migrasi 0002.
