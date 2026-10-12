-- 0005 turun — mengembalikan Daftar Tipe Dokumen ke penyimpanan JSON.
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- BERKAS INI BELUM PERNAH DIJALANKAN DI LINGKUNGAN MANA PUN.
--
-- Sama seperti migrasi naiknya, ia dijalankan di BASIS DATA SETIAP ENTITAS yang pernah
-- dinaikkan — entitas yang terlewat akan tertinggal pada skema yang berbeda dari yang lain.
--
--
-- ## Apa yang benar-benar dapat dikembalikan, dan apa yang tidak
--
-- Migrasi ini TIDAK sepenuhnya reversibel, dan itu harus disadari sebelum memulai — bukan
-- ditemukan di tengah jalan.
--
--   DAPAT dikembalikan  definisi view, dan struktur tabel.
--   TIDAK dapat         setiap perubahan yang dibuat lewat aplikasi Go SESUDAH langkah 3
--                       migrasi naik dijalankan.
--
-- Sebabnya: sejak Go menjadi penulis, ia menulis ke KOLOM dan tidak pernah menyentuh
-- JSON_DATA. Kolom JSON_DATA karena itu berhenti pada nilai terakhir yang ditulis Pega.
-- Mengembalikan view supaya membaca JSON_DATA berarti mengembalikan pula keadaan data ke
-- saat itu — setiap penambahan dan penyuntingan yang dibuat di antaranya HILANG DARI
-- PANDANGAN PEGA.
--
-- Langkah 1 di bawah menyusun ulang JSON_DATA dari kolom supaya kehilangan itu tidak
-- terjadi. Ia WAJIB dijalankan lebih dulu, dan DBA wajib memeriksa hasilnya sebelum lanjut.
--
--
-- ## Langkah 0 — sebelum apa pun
--
-- 1. Ambil kembali definisi view yang disimpan saat migrasi naik (langkah 0 butir 1 pada
--    berkas .up.sql). Bila berkas itu tidak ada, BERHENTI: definisi di bawah adalah
--    rekonstruksi dari cara Pega memakainya, bukan salinan yang disimpan, dan
--    menjalankannya berisiko memasang view yang berbeda dari aslinya.
--
--    Ia juga satu-satunya yang menjawab apakah OLD_ID kolom sungguhan atau nilai dari
--    JSON. Langkah 3 di bawah menganggapnya KOLOM; sesuaikan bila ternyata bukan.
--
-- 2. Hitung berapa baris yang lahir setelah cutover, supaya besarnya persoalan diketahui
--    sebelum diputuskan:
--
--        SELECT COUNT(*) FROM POOLDATA.LST_DOC_TYPE WHERE JSON_DATA IS NULL;


-- ---------------------------------------------------------------------------
-- Langkah 1 — susun ulang JSON_DATA dari kolom.
--
-- Bentuk dokumennya mengikuti nama property Pega, sama seperti yang dibaca migrasi naik.
--
-- TGL_EDIT dikembalikan ke bentuk cap waktu Pega ('20260921T030000.000 GMT'), bukan ke
-- format tanggal Oracle: itulah bentuk yang ditulis `@getCurrentTimeStamp()` dan yang
-- diharapkan rule yang membacanya. Baris yang TGL_EDIT-nya NULL menghasilkan kunci bernilai
-- null, dan itu memang keadaan yang benar untuk baris yang tidak punya jejak simpan.
--
-- OLD_ID ikut disusun ulang supaya dokumennya utuh seperti semula. Ia dibaca dari kolom
-- bila kolomnya ada; bila langkah 0 butir 1 menunjukkan ia memang berasal dari JSON, hapus
-- barisnya — nilainya sudah ada di dalam JSON_DATA yang sedang ditulis ulang ini.
--
-- JSON_OBJECT menuntut Oracle 12.2+. Berkas ini dijalankan DBA langsung di Oracle dan tidak
-- pernah ikut ke PostgreSQL, sehingga aturan SQL portabel (`D-20`) tidak berlaku di sini.
-- ---------------------------------------------------------------------------

UPDATE POOLDATA.LST_DOC_TYPE
   SET JSON_DATA = JSON_OBJECT(
                       'ID'            VALUE ID,
                       'OLD_ID'        VALUE OLD_ID,
                       'TYPE_DOCUMENT' VALUE TYPE_DOCUMENT,
                       'STS_PROSES'    VALUE STS_PROSES,
                       'USER_EDIT'     VALUE USER_EDIT,
                       'TGL_EDIT'      VALUE TO_CHAR(TGL_EDIT, 'YYYYMMDD"T"HH24MISS".000 GMT"')
                   );

COMMIT;


-- ---------------------------------------------------------------------------
-- Langkah 2 — VERIFIKASI. Jangan lanjut bila jawabannya tidak seperti yang disebutkan.
-- ---------------------------------------------------------------------------

-- 2a. Masih adakah baris yang JSON_DATA-nya kosong?  DIHARAPKAN: 0
--
--     SELECT COUNT(*) FROM POOLDATA.LST_DOC_TYPE WHERE JSON_DATA IS NULL;

-- 2b. Adakah baris yang JSON-nya tidak sesuai kolomnya?  DIHARAPKAN: 0
--
--     SELECT COUNT(*) FROM POOLDATA.LST_DOC_TYPE
--      WHERE NVL(JSON_VALUE(JSON_DATA, '$.TYPE_DOCUMENT'), '~') <> NVL(TYPE_DOCUMENT, '~');

-- 2c. Periksa SATU dokumen dengan mata, bukan hanya dengan hitungan — bentuk cap waktunya
--     yang paling mungkin berbeda dari aslinya:
--
--     SELECT JSON_DATA FROM POOLDATA.LST_DOC_TYPE WHERE TGL_EDIT IS NOT NULL AND ROWNUM = 1;


-- ---------------------------------------------------------------------------
-- Langkah 3 — kembalikan view supaya membaca JSON lagi.
--
-- PAKAI DEFINISI YANG DISIMPAN bila ada. Yang di bawah adalah rekonstruksi.
--
-- TGL_EDIT dikembalikan sebagai TEKS, bukan DATE, karena itulah yang keluar dari JSON. Bila
-- definisi aslinya membungkusnya TO_DATE, ikuti yang asli — beda tipe kolom view berarti
-- beda perilaku bagi rule yang membacanya.
-- ---------------------------------------------------------------------------

CREATE OR REPLACE VIEW POOLDATA.V_LST_DOC_TYPE
    (ID, OLD_ID, TYPE_DOCUMENT, STS_PROSES, USER_EDIT, TGL_EDIT) AS
SELECT ID,
       JSON_VALUE(JSON_DATA, '$.OLD_ID'),
       JSON_VALUE(JSON_DATA, '$.TYPE_DOCUMENT'),
       JSON_VALUE(JSON_DATA, '$.STS_PROSES'),
       JSON_VALUE(JSON_DATA, '$.USER_EDIT'),
       JSON_VALUE(JSON_DATA, '$.TGL_EDIT')
  FROM POOLDATA.LST_DOC_TYPE;


-- ---------------------------------------------------------------------------
-- Langkah 4 — buang kolom yang ditambahkan.
--
-- SENGAJA DIKOMENTARI. Setelah langkah 3, view sudah membaca JSON lagi dan Pega sudah
-- berfungsi penuh — kolom di bawah tidak mengganggu apa pun, sedangkan membuangnya
-- menghapus satu-satunya bahan untuk menjalankan migrasi naik lagi tanpa mengulang seluruh
-- pemindahan.
--
-- Buang hanya setelah masa pengamatan berjalan dan diputuskan tidak akan kembali.
-- ---------------------------------------------------------------------------

-- ALTER TABLE POOLDATA.LST_DOC_TYPE DROP (TYPE_DOCUMENT);
-- ALTER TABLE POOLDATA.LST_DOC_TYPE DROP (STS_PROSES);
-- ALTER TABLE POOLDATA.LST_DOC_TYPE DROP (USER_EDIT);
-- ALTER TABLE POOLDATA.LST_DOC_TYPE DROP (TGL_EDIT);


-- ---------------------------------------------------------------------------
-- Langkah 5 — cabut hak tulis akun aplikasi.
--
-- Setelah turun, Pega kembali menjadi penulis tunggal (`P-1`). Hak tulis yang dibiarkan
-- menempel akan membuat dua sistem sama-sama BISA menulis — keadaan yang justru dicegah
-- aturan penulis tunggal.
--
-- Ganti <AKUN_APLIKASI> dengan nama akun yang sebenarnya.
-- ---------------------------------------------------------------------------

-- REVOKE INSERT, UPDATE ON POOLDATA.LST_DOC_TYPE FROM <AKUN_APLIKASI>;
