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


-- 0004 turun — mengembalikan Master COL Simas Online ke penyimpanan JSON.
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- BERKAS INI BELUM PERNAH DIJALANKAN DI LINGKUNGAN MANA PUN.
--
--
-- ## Apa yang benar-benar dapat dikembalikan, dan apa yang tidak
--
-- Migrasi ini TIDAK sepenuhnya reversibel, dan itu harus disadari sebelum memulai —
-- bukan ditemukan di tengah jalan.
--
--   DAPAT dikembalikan  definisi kedua view, dan struktur tabel.
--   TIDAK dapat         setiap perubahan yang dibuat lewat aplikasi Go SESUDAH langkah 5
--                       migrasi naik dijalankan.
--
-- Sebabnya: sejak Go menjadi penulis, ia menulis ke KOLOM dan tidak pernah menyentuh
-- JSON_DATA. Kolom JSON_DATA karena itu berhenti pada nilai terakhir yang ditulis Pega.
-- Mengembalikan view supaya membaca JSON_DATA berarti mengembalikan pula keadaan data ke
-- saat itu — setiap penambahan dan penyuntingan yang dibuat di antaranya HILANG DARI
-- PANDANGAN PEGA.
--
-- Langkah 1 di bawah menyusun ulang JSON_DATA dari kolom supaya kehilangan itu tidak
-- terjadi. Ia WAJIB dijalankan lebih dulu, dan DBA wajib memeriksa hasilnya sebelum
-- lanjut.
--
--
-- ## Langkah 0 — sebelum apa pun
--
-- 1. Ambil kembali definisi view yang disimpan saat migrasi naik (langkah 0 butir 1 pada
--    berkas .up.sql). Bila berkas itu tidak ada, BERHENTI: definisi di bawah adalah
--    rekonstruksi dari cara Pega memakainya, bukan salinan yang disimpan, dan
--    menjalankannya berisiko memasang view yang berbeda dari aslinya.
--
-- 2. Hitung berapa baris yang lahir setelah cutover, supaya besarnya persoalan diketahui
--    sebelum diputuskan:
--
--        SELECT COUNT(*) FROM POOLDATA.M_CAUSE_OF_LOSS WHERE JSON_DATA IS NULL;


-- ---------------------------------------------------------------------------
-- Langkah 1 — susun ulang JSON_DATA dari kolom.
--
-- Bentuk dokumennya mengikuti nama property Pega, sama seperti yang dibaca migrasi naik.
-- Senarai BISNISID disusun dari tabel pemetaan, dan HANYA baris yang masih aktif yang
-- ikut — baris bertanda '0' memang sudah dibuang pengguna.
--
-- JSON_OBJECT dan JSON_ARRAYAGG menuntut Oracle 12.2+. Berkas ini dijalankan DBA
-- langsung di Oracle dan tidak pernah ikut ke PostgreSQL, sehingga aturan SQL portabel
-- (`D-20`) tidak berlaku di sini.
-- ---------------------------------------------------------------------------

UPDATE POOLDATA.M_CAUSE_OF_LOSS m
   SET m.JSON_DATA = (
       SELECT JSON_OBJECT(
                  'M_COL_ID'   VALUE m.M_COL_ID,
                  'COL_DESC'   VALUE m.COL_DESC,
                  'MST_COL_ID' VALUE m.MST_COL_ID,
                  'BISNISID'   VALUE (
                      -- Kedua kunci disusun ulang: 'Note' yang dibaca layar, dan 'ID'
                      -- yang ikut terisi saat dipilih dari daftar. 'ID' boleh NULL pada
                      -- baris yang namanya dulu diketik bebas — dan harus tetap begitu,
                      -- karena itulah yang membuat baris tersebut kembali seperti semula.
                      --
                      -- ORDER BY URUTAN mengembalikan susunan barisnya.
                      SELECT JSON_ARRAYAGG(
                                 JSON_OBJECT('ID'   VALUE b.BISNISID,
                                             'Note' VALUE b.NAMA_BISNIS)
                                 ORDER BY b.URUTAN
                             )
                        FROM POOLDATA.M_CAUSE_OF_LOSS_BUSINESS b
                       WHERE b.M_COL_ID = m.M_COL_ID
                         AND b.STS_AKTIF = '1'
                  )
              )
         FROM DUAL
       );

COMMIT;


-- ---------------------------------------------------------------------------
-- Langkah 2 — VERIFIKASI. Jangan lanjut bila jawabannya tidak seperti yang disebutkan.
-- ---------------------------------------------------------------------------

-- 2a. Masih adakah baris yang JSON_DATA-nya kosong?  DIHARAPKAN: 0
--
--     SELECT COUNT(*) FROM POOLDATA.M_CAUSE_OF_LOSS WHERE JSON_DATA IS NULL;

-- 2b. Adakah baris yang JSON-nya tidak sesuai kolomnya?  DIHARAPKAN: 0
--
--     SELECT COUNT(*) FROM POOLDATA.M_CAUSE_OF_LOSS
--      WHERE NVL(JSON_VALUE(JSON_DATA, '$.COL_DESC'), '~') <> NVL(COL_DESC, '~');

-- 2c. Periksa SATU dokumen dengan mata, bukan hanya dengan hitungan — bentuk senarainya
--     yang paling mungkin berbeda dari aslinya:
--
--     SELECT JSON_DATA FROM POOLDATA.M_CAUSE_OF_LOSS
--      WHERE M_COL_ID IN (SELECT M_COL_ID FROM POOLDATA.M_CAUSE_OF_LOSS_BUSINESS
--                          WHERE STS_AKTIF = '1' AND ROWNUM = 1);


-- ---------------------------------------------------------------------------
-- Langkah 3 — kembalikan kedua view supaya membaca JSON lagi.
--
-- PAKAI DEFINISI YANG DISIMPAN bila ada. Yang di bawah adalah rekonstruksi.
-- ---------------------------------------------------------------------------

CREATE OR REPLACE VIEW POOLDATA.V_M_CAUSE_OF_LOSS (M_COL_ID, OLD_M_COL_ID, COL_DESC) AS
SELECT M_COL_ID,
       JSON_DATA,
       JSON_VALUE(JSON_DATA, '$.COL_DESC')
  FROM POOLDATA.M_CAUSE_OF_LOSS;

CREATE OR REPLACE VIEW POOLDATA.V_M_CAUSE_OF_LOSS_BUSINESS (M_COL_ID, BISNISID) AS
SELECT m.M_COL_ID,
       b.BISNISID
  FROM POOLDATA.M_CAUSE_OF_LOSS m,
       JSON_TABLE(m.JSON_DATA, '$.BISNISID[*]'
                  COLUMNS (BISNISID VARCHAR2(10) PATH '$.ID')) b
 WHERE m.JSON_DATA IS NOT NULL;

-- CATATAN: view di atas hanya mengeluarkan BISNISID, sehingga baris yang namanya diketik
-- bebas (tanpa ID) tampil sebagai NULL. Itu memang keadaan view aslinya — dan itulah
-- sebabnya kolom NAMA_BISNIS pada tabel TIDAK boleh dibuang sebelum benar-benar yakin
-- tidak akan kembali ke atas: JSON_DATA hasil langkah 1 adalah satu-satunya tempat nama
-- itu masih tersimpan setelah tabelnya di-DROP.


-- ---------------------------------------------------------------------------
-- Langkah 4 — buang tabel pemetaan dan kolom yang ditambahkan.
--
-- SENGAJA DIKOMENTARI. Setelah langkah 3, kedua view sudah membaca JSON lagi dan Pega
-- sudah berfungsi penuh — tabel dan kolom di bawah tidak mengganggu apa pun, sedangkan
-- membuangnya menghapus satu-satunya bahan untuk menjalankan migrasi naik lagi tanpa
-- mengulang seluruh pemindahan.
--
-- Buang hanya setelah masa pengamatan berjalan dan diputuskan tidak akan kembali.
-- ---------------------------------------------------------------------------

-- DROP TABLE POOLDATA.M_CAUSE_OF_LOSS_BUSINESS;
-- ALTER TABLE POOLDATA.M_CAUSE_OF_LOSS DROP (COL_DESC);
-- ALTER TABLE POOLDATA.M_CAUSE_OF_LOSS DROP (MST_COL_ID);


-- ---------------------------------------------------------------------------
-- Langkah 5 — cabut hak tulis akun aplikasi.
--
-- Setelah turun, Pega kembali menjadi penulis tunggal (`P-1`). Hak tulis yang dibiarkan
-- menempel akan membuat dua sistem sama-sama BISA menulis — keadaan yang justru dicegah
-- aturan penulis tunggal.
--
-- Ganti <AKUN_APLIKASI> dengan nama akun yang sebenarnya.
-- ---------------------------------------------------------------------------

-- REVOKE INSERT, UPDATE ON POOLDATA.M_CAUSE_OF_LOSS TO <AKUN_APLIKASI>;
