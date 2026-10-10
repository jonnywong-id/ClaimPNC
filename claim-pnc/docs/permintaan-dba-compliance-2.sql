-- ============================================================================
-- PERMINTAAN DBA — lanjutan untuk modul Inbox Compliance
-- ============================================================================
--
-- Tanggal      : 2026-10-08
-- Untuk        : DBA
-- Dari         : tim pengembang Claim PNC
-- Menempuh     : `D-63` — permintaan tertulis -> persetujuan Work Owner -> pelaksanaan DBA
--
-- ----------------------------------------------------------------------------
-- HUBUNGANNYA DENGAN BERKAS SEBELUMNYA
-- ----------------------------------------------------------------------------
--
-- `permintaan-dba-compliance.sql` (2026-10-07) meminta DUA objek:
--
--     POOLDATA.CLAIM_COMPLIENCE_SEQ
--     POOLDATA.CPNC_KEPUTUSAN_COMPLIANCE
--
-- Berkas INI meminta satu objek lagi yang terlewat dari berkas itu:
--
--     POOLDATA.CPNC_PENUGASAN
--
-- Keduanya dibutuhkan bersama. Bila berkas pertama belum dijalankan, jalankan ia lebih
-- dulu — BAGIAN 0 di bawah membantu memastikannya tanpa menebak.
--
-- ----------------------------------------------------------------------------
-- KENAPA YANG INI PALING MENDESAK
-- ----------------------------------------------------------------------------
--
-- Tanpa `CPNC_PENUGASAN`, yang gagal BUKAN hanya perpindahan klaim — **seluruh antrean
-- Compliance tampil KOSONG**. Kueri daftar menyaring klaim yang sudah berpindah lewat
-- `NOT EXISTS` ke tabel ini, dan penyaring yang tabelnya hilang menjatuhkan seluruh baris.
--
-- Gejalanya karena itu menyesatkan: layar terbuka normal, tanpa galat, tanpa satu baris
-- pun. Petugas akan melaporkannya sebagai "tidak ada pekerjaan", bukan sebagai kerusakan.
--
-- ----------------------------------------------------------------------------
-- BERKAS INI SALINAN, BUKAN SUMBERNYA
-- ----------------------------------------------------------------------------
--
-- Disalin dari `claim-pnc/backend/migrations/0014_penugasan.up.sql`, yang tetap menjadi
-- sumber kebenaran. Catatan panjang yang menjelaskan SEBAB setiap kolom ada di sana; yang
-- di sini hanya pernyataannya beserta alasan singkatnya.
--
-- ----------------------------------------------------------------------------
-- DI MANA DIJALANKAN
-- ----------------------------------------------------------------------------
--
-- Di basis data **SETIAP entitas** (`D-75` — satu basis data per badan hukum). Entitas
-- yang terlewat gagal di entitas itu saja.
--
-- ----------------------------------------------------------------------------
-- TENTANG GRANT — tidak ada, dan itu disengaja
-- ----------------------------------------------------------------------------
--
-- Alasannya sama persis dengan berkas pertama: aplikasi menyambung sebagai pemilik
-- skemanya sendiri, sehingga tidak ada hak yang perlu diberikan. Menjalankan GRANT ke akun
-- yang tidak ada justru menggagalkan skrip di tengah dan meninggalkan sebagian objek
-- terbuat. Penjelasan lengkapnya di `0012_keputusan_compliance.up.sql`.
--
-- Itu berlaku juga untuk `POOLDATA.DATA_ATTACHFILE` dan `POOLDATA.C_COUNTER_ATTACHMENT`
-- yang modul ini TULIS — keduanya sudah ada dan sudah milik skema yang sama.
--
-- ============================================================================


-- ============================================================================
-- BAGIAN 0 — PERIKSA DULU (hanya SELECT, tidak mengubah apa pun)
-- ============================================================================
--
-- Jalankan keempat kueri ini LEBIH DULU di setiap entitas, lalu kirimkan hasilnya. Dari
-- situ terlihat apa yang sudah ada dan apa yang belum, tanpa ada yang perlu ditebak —
-- dan `CREATE` atas objek yang sudah ada akan gagal dengan `ORA-00955`.

-- 0.1 Ketiga objek modul ini: mana yang sudah ada?
--
-- Memakai sub-kueri COUNT, bukan `EXISTS`: Oracle tidak mengizinkan `EXISTS` di dalam
-- daftar SELECT — hanya di klausa WHERE.
SELECT 'CLAIM_COMPLIENCE_SEQ' AS OBJEK,
       CASE WHEN (SELECT COUNT(*) FROM ALL_SEQUENCES
                   WHERE SEQUENCE_OWNER = 'POOLDATA'
                     AND SEQUENCE_NAME  = 'CLAIM_COMPLIENCE_SEQ') > 0
            THEN 'ADA' ELSE 'BELUM ADA' END AS STATUS
  FROM DUAL
UNION ALL
SELECT 'CPNC_KEPUTUSAN_COMPLIANCE',
       CASE WHEN (SELECT COUNT(*) FROM ALL_TABLES
                   WHERE OWNER      = 'POOLDATA'
                     AND TABLE_NAME = 'CPNC_KEPUTUSAN_COMPLIANCE') > 0
            THEN 'ADA' ELSE 'BELUM ADA' END
  FROM DUAL
UNION ALL
SELECT 'CPNC_PENUGASAN',
       CASE WHEN (SELECT COUNT(*) FROM ALL_TABLES
                   WHERE OWNER      = 'POOLDATA'
                     AND TABLE_NAME = 'CPNC_PENUGASAN') > 0
            THEN 'ADA' ELSE 'BELUM ADA' END
  FROM DUAL;


-- 0.2 Kedua tabel lampiran yang modul ini TULIS — keduanya seharusnya sudah ada.
--
-- Bila salah satunya tidak ada, unggah dokumen dan penerbitan Surat Penolakan gagal.
-- Keduanya tabel LAMA; kami tidak meminta pembuatannya, hanya memastikan keberadaannya.
SELECT TABLE_NAME, 'ADA' AS STATUS
  FROM ALL_TABLES
 WHERE OWNER = 'POOLDATA'
   AND TABLE_NAME IN ('DATA_ATTACHFILE', 'C_COUNTER_ATTACHMENT');


-- 0.3 Baris folder penyimpanan dokumen — WAJIB terbaca dari basis data SETIAP entitas.
--
-- Nilainya dipakai sebagai `DocAPI.App` saat mengunggah, dan ia BUKAN teks "KLAIMPNC"
-- melainkan nama foldernya. Bila barisnya tidak ada, aplikasi menolak unggahan dengan
-- pesan yang menyebut tabel ini — sengaja, karena layanan penyimpanan di seberang tidak
-- memeriksa apa pun dan berkas nasabah akan mendarat di folder yang salah tanpa galat.
--
-- Tabelnya ada di skema GENERAL, dicapai lewat DB link pada sebagian entitas. Bila kueri
-- ini gagal karena link-nya, itu sendiri temuan yang perlu dilaporkan.
SELECT APLIKASI, NAMA_FOLDER
  FROM GENERAL.T_FOLDER_STORAGE
 WHERE APLIKASI = 'KLAIMPNC';


-- 0.4 Sanity check isi antrean — berapa klaim yang sedang menunggu di Compliance.
--
-- Dijalankan SEBELUM dan SESUDAH BAGIAN 1. Angkanya harus SAMA.
--
-- Bila sesudahnya menjadi nol, berarti ada yang keliru pada pembuatan tabel — dan lebih
-- baik ketahuan di sini daripada dilaporkan petugas sebagai "antrean saya hilang".
--
-- Kueri ini menyalin penyaring kueri daftar aplikasi APA ADANYA, dikurangi klausa
-- `NOT EXISTS` ke CPNC_PENUGASAN — yang memang belum dapat dijalankan sebelum tabelnya
-- ada. Karena itu angkanya sepadan dengan yang dilihat petugas di layar.
SELECT COUNT(*) AS JUMLAH_ANTREAN_COMPLIANCE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET wb
               ON wb.PXREFOBJECTKEY = a.PZINSKEY
 WHERE a.PXOBJCLASS LIKE 'ASM-FW-GCNMFW-Work-PNC%'
   AND wb.PXASSIGNEDOPERATORID = 'CompliancePNC'
   AND a.PYSTATUSWORK <> 'Resolved-Completed';


-- ============================================================================
-- BAGIAN 1 — POOLDATA.CPNC_PENUGASAN
-- ============================================================================
--
-- Satu tabel untuk Worklist DAN Workbasket sekaligus.
--
-- Pega punya dua tabel karena ia punya dua kelas. Kami tidak terikat itu, dan satu tabel
-- menghapus satu kelas kegagalan: perpindahan antar antrean menjadi UPDATE satu baris —
-- atomik dengan sendirinya — alih-alih DELETE di satu tabel ditambah INSERT di tabel lain.
-- Bila yang kedua gagal, klaimnya hilang dari mana-mana tanpa galat.
--
-- Tabel ini TIDAK menyentuh satu pun objek Pega. `DATAPEGA.PC_ASSIGN_WORKLIST` dan
-- `PC_ASSIGN_WORKBASKET` dibiarkan apa adanya dan tetap dimiliki Pega.

CREATE TABLE POOLDATA.CPNC_PENUGASAN (
    -- Kunci baris, diterbitkan aplikasi.
    PENUGASAN_ID   VARCHAR2(64 BYTE)  NOT NULL,

    -- Klaimnya — `PZINSKEY`, sama dengan kolom bernama sama pada
    -- CPNC_KEPUTUSAN_COMPLIANCE.
    NO_KLAIM       VARCHAR2(255 BYTE) NOT NULL,

    -- Tahap alur yang menugaskannya: 'Compliance', 'Send To Analis', dan seterusnya.
    -- Nilainya nama shape pada `Flow/Register_Flow.xml`, supaya baris di sini dapat
    -- ditelusuri balik ke flow aslinya.
    TAHAP          VARCHAR2(100 BYTE) NOT NULL,

    -- 'WORKLIST' atau 'WORKBASKET' — pembeda yang `D-26` tetapkan.
    JENIS          VARCHAR2(10 BYTE)  NOT NULL,

    -- Terisi HANYA pada WORKLIST — login orang yang memegangnya.
    DITUGASKAN_KE  VARCHAR2(128 BYTE),

    -- Terisi HANYA pada WORKBASKET — nama antreannya, mis. 'CompliancePNC'.
    WORKBASKET     VARCHAR2(100 BYTE),

    -- Penguncian antar-pengguna. Belum dipakai hari ini; kolomnya disiapkan supaya
    -- penguncian kelak tidak menuntut DDL baru — dan DDL baru menempuh `D-63` lagi.
    DIAMBIL_OLEH   VARCHAR2(128 BYTE),
    DIAMBIL_PADA   TIMESTAMP(6),

    -- 'MENUNGGU' atau 'SELESAI'. Penugasan yang selesai TIDAK dihapus (`D-66`) — ia
    -- ditandai, sehingga riwayat perpindahan satu klaim tetap terbaca seluruhnya.
    STATUS         VARCHAR2(20 BYTE)  NOT NULL,

    DIBUAT_PADA    TIMESTAMP(6)       NOT NULL,
    DIUBAH_PADA    TIMESTAMP(6),

    CONSTRAINT PK_CPNC_PENUGASAN PRIMARY KEY (PENUGASAN_ID),

    CONSTRAINT CK_CPNC_PENUGASAN_JENIS  CHECK (JENIS IN ('WORKLIST', 'WORKBASKET')),
    CONSTRAINT CK_CPNC_PENUGASAN_STATUS CHECK (STATUS IN ('MENUNGGU', 'SELESAI')),

    -- "Tidak pernah di keduanya" ditegakkan BASIS DATA, bukan hanya kode.
    --
    -- Tanpa ini, satu cacat kode dapat menyimpan baris yang sekaligus milik seseorang DAN
    -- milik sebuah antrean — dan baris seperti itu muncul di dua inbox sekaligus, lalu
    -- dikerjakan dua kali.
    CONSTRAINT CK_CPNC_PENUGASAN_ISI CHECK (
        (JENIS = 'WORKLIST'   AND DITUGASKAN_KE IS NOT NULL AND WORKBASKET    IS NULL) OR
        (JENIS = 'WORKBASKET' AND WORKBASKET    IS NOT NULL AND DITUGASKAN_KE IS NULL))
);

-- Antrean bersama: "apa yang menunggu di workbasket ini".
CREATE INDEX POOLDATA.CPNC_PENUGASAN_IDX01
    ON POOLDATA.CPNC_PENUGASAN (WORKBASKET, STATUS, DIBUAT_PADA);

-- Inbox pribadi: "apa yang menunggu saya".
CREATE INDEX POOLDATA.CPNC_PENUGASAN_IDX02
    ON POOLDATA.CPNC_PENUGASAN (DITUGASKAN_KE, STATUS, DIBUAT_PADA);

-- Penyaring daftar Compliance: "klaim ini sudah selesai di tahap itu atau belum".
--
-- Dipakai `NOT EXISTS` pada kueri daftar. Tanpa index ini, penyaring itu memaksa
-- pemindaian penuh pada SETIAP halaman inbox.
CREATE INDEX POOLDATA.CPNC_PENUGASAN_IDX03
    ON POOLDATA.CPNC_PENUGASAN (NO_KLAIM, TAHAP, STATUS);

COMMIT;


-- ============================================================================
-- BAGIAN 2 — PERIKSA SESUDAHNYA
-- ============================================================================

-- 2.1 Tabel dan ketiga index-nya terbentuk.
SELECT 'TABEL' AS JENIS, TABLE_NAME AS NAMA
  FROM ALL_TABLES
 WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'CPNC_PENUGASAN'
UNION ALL
SELECT 'INDEX', INDEX_NAME
  FROM ALL_INDEXES
 WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'CPNC_PENUGASAN'
 ORDER BY 1, 2;

-- 2.2 Kesembilan kolom yang aplikasi tulis benar-benar ada.
--
-- Hasilnya harus SEMBILAN baris. Kurang satu pun berarti aplikasi gagal saat menyimpan,
-- bukan saat membaca — dan itu ketahuan jauh lebih terlambat.
SELECT COLUMN_NAME, DATA_TYPE, NULLABLE
  FROM ALL_TAB_COLUMNS
 WHERE OWNER = 'POOLDATA'
   AND TABLE_NAME = 'CPNC_PENUGASAN'
   AND COLUMN_NAME IN ('PENUGASAN_ID','NO_KLAIM','TAHAP','JENIS','DITUGASKAN_KE',
                       'WORKBASKET','STATUS','DIBUAT_PADA','DIUBAH_PADA')
 ORDER BY COLUMN_ID;

-- 2.3 Ulangi kueri 0.4. Angkanya HARUS sama dengan sebelum BAGIAN 1 dijalankan.


-- ============================================================================
-- YANG KAMI MINTA DILAPORKAN BALIK
-- ============================================================================
--
--   1. Hasil 0.1 per entitas — objek mana yang sudah ada, mana yang baru dibuat
--   2. Hasil 0.3 per entitas — baris KLAIMPNC ada atau tidak, beserta NAMA_FOLDER-nya
--   3. Angka 0.4 sebelum dan sesudah
--   4. Entitas mana saja yang sudah dijalankan
--
-- Bila 0.3 tidak mengembalikan baris di sebuah entitas, tolong sebutkan — itu bukan
-- kesalahan skrip ini, melainkan prasyarat yang memang belum ada, dan penanganannya ada
-- di pihak pemilik layanan penyimpanan dokumen.
