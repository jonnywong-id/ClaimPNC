-- 0006 — Permintaan ReOpen dan Copy Klaim: tabel baru milik aplikasi (Oracle 19c)
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- # Apa yang dibuat, dan apa yang TIDAK disentuh
--
-- Berkas ini hanya MENAMBAH objek baru:
--
--   * POOLDATA.CPNC_PERMINTAAN_KLAIM   permintaan ReOpen dan Copy Klaim
--   * satu indeks pendukung
--   * satu indeks unik berbasis fungsi
--
-- TIDAK ADA satu pun objek milik sistem lama yang diubah. Tidak ada ALTER, tidak ada
-- penggantian view, tidak ada perubahan hak pada tabel yang sedang melayani produksi.
--
-- Ia tetap menempuh prosedur `D-63`: permintaan tertulis tim pengembang, persetujuan Work
-- Owner, pelaksanaan DBA, lalu pengujian dengan MENJALANKAN PEGA DAN GO BERSAMAAN. Akun
-- aplikasi tidak memiliki hak DDL.
--
-- BERKAS INI BELUM PERNAH DIJALANKAN DI LINGKUNGAN MANA PUN.
--
--
-- # Kenapa tabel PERMINTAAN, dan bukan menulis langsung ke klaimnya
--
-- Yang benar-benar harus berubah saat sebuah klaim dibuka kembali adalah
-- `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` — `PYSTATUSWORK`, `STATUSCLAIM_1`, `PYREOPENCOUNT`,
-- `PYREOPENTIMESTAMP`. Tabel itu hari ini DITULIS PEGA, dan `P-1`/`ADR-0004` menetapkan
-- satu tabel hanya boleh ditulis satu sistem.
--
-- Dua penulis dengan aturan yang berbeda tidak menghasilkan galat apa pun — hanya data
-- yang berubah sendiri. Work Owner memutuskan 2026-09-23 aplikasi ini mencatat
-- PERMINTAANNYA; eksekusinya tetap di Pega.
--
--
-- # Akibat yang HARUS diketahui sebelum modul ini menyala di produksi
--
-- Klaim TIDAK berubah saat tombolnya ditekan. Barisnya tetap tampil di layar Inbox Close
-- Claim sampai permintaannya dijalankan. Tiga hal mengikuti dari itu:
--
--   1. Layar WAJIB menandai baris yang permintaannya sudah terkirim. Tanpa itu, pengguna
--      yang tidak melihat perubahan akan menekan tombolnya lagi.
--   2. SIAPA yang menjalankan permintaan ini, dan SEBERAPA SERING, belum ditetapkan. Sampai
--      itu ada, barisnya menumpuk dalam keadaan 'menunggu' dan tidak ada yang terjadi.
--   3. Tidak ada pemberitahuan otomatis kepada pemohon saat permintaannya dijalankan.
--
--
-- # Yang HARUS dikerjakan DBA bersama tabel ini
--
--  1. PASTIKAN NAMANYA BELUM DIPAKAI:
--
--       SELECT object_name, object_type FROM all_objects
--        WHERE owner = 'POOLDATA' AND object_name LIKE 'CPNC_PERMINTAAN%';
--
--     Berkas ini TIDAK memakai `CREATE OR REPLACE` dan tidak menimpa apa pun; bila namanya
--     sudah ada, pernyataannya gagal — dan itu memang yang diinginkan.
--
--  2. BERI HAK YANG TEPAT, DAN HANYA ITU.
--
--       GRANT SELECT, INSERT ON POOLDATA.CPNC_PERMINTAAN_KLAIM TO <akun aplikasi>;
--
--     JANGAN memberikan UPDATE maupun DELETE kepada akun aplikasi. Ia hanya mencatat
--     permintaan dan membacanya kembali; yang memindahkan STATUS ke 'dijalankan' adalah
--     pihak yang benar-benar mengeksekusi, dengan akunnya sendiri.
--
--     Aturan yang hanya ada di dalam kode dapat dilanggar oleh kode berikutnya; aturan yang
--     ada di hak akses tidak. Ini lebih penting di sini daripada di tabel mana pun: `D-59`
--     menghapus pemisahan tugas, sehingga jejak inilah satu-satunya kontrol pengimbang yang
--     tersisa.
--
--  3. BERI HAK UPDATE kepada akun PELAKSANA — dan hanya pada kolom STATUS bila dapat
--     dibatasi:
--
--       GRANT UPDATE (STATUS) ON POOLDATA.CPNC_PERMINTAAN_KLAIM TO <akun pelaksana>;
--
--  4. RETENSI mengikuti retensi data klaim yang berlaku sekarang (`D-62`) — satu kebijakan
--     untuk keduanya. ANGKANYA belum diserahkan, dan sampai itu tiba tidak ada penghapusan
--     berkala yang boleh dijadwalkan.
--
--
-- # Empat portal, empat kali
--
-- `D-75` menetapkan satu database per entitas. Migrasi ini karena itu dijalankan EMPAT
-- KALI — sekali per portal — dan gagal di salah satunya membuat portal itu tertinggal
-- versi. Layar Inbox Close Claim pada portal yang tertinggal tetap MENAMPILKAN klaim
-- (daftarnya tidak menyentuh tabel ini), tetapi penanda permintaan tidak pernah muncul dan
-- setiap penekanan tombol dijawab galat.
--
-- Perilaku itu disengaja dan diuji: daftar tidak boleh mati hanya karena tabel permintaan
-- belum ada. Lihat `usecase.ListResult.PendingLookupError`.

-- ---------------------------------------------------------------------------
-- CPNC_PERMINTAAN_KLAIM — satu baris per permintaan.
--
-- Bukan satu baris per klaim. Sebuah klaim dapat diminta dibuka kembali, lalu ditutup lagi,
-- lalu diminta lagi — dan setiap permintaan meninggalkan barisnya sendiri. Yang dibatasi
-- hanyalah permintaan yang SEDANG MENUNGGU; lihat indeks unik di bawah.
-- ---------------------------------------------------------------------------

CREATE TABLE POOLDATA.CPNC_PERMINTAAN_KLAIM (
    -- Pengenal acak 128 bit dalam heksadesimal, dibangkitkan aplikasi.
    --
    -- Acak, bukan berurut: pengenal permintaan tidak boleh membocorkan berapa banyak
    -- permintaan yang sudah tercatat.
    ID                  VARCHAR2(32)   NOT NULL,

    -- 'reopen' atau 'salin'.
    --
    -- Nilainya berbahasa Indonesia karena ia sama dengan nilai pada kontrak API.
    JENIS               VARCHAR2(20)   NOT NULL,

    -- `PZINSKEY` klaim yang dimaksud — yang di sistem lama dikirim tombolnya sebagai
    -- parameter `casePNC`.
    --
    -- TIDAK ada foreign key ke tabel warisan, dan ketiadaannya disengaja: constraint dari
    -- tabel milik kita ke tabel milik Pega akan membuat pengarsipan di sisi Pega GAGAL
    -- karena baris kita. Kita tidak boleh menghalangi sistem yang sedang melayani produksi.
    --
    -- Lebarnya 64: kunci warisan berbentuk `ASM-FW-GCNMFW-WORK PNC-9001`, yaitu nama kelas
    -- internal Pega ditambah spasi ditambah nomor klaim.
    CASE_ID             VARCHAR2(64)   NOT NULL,

    -- Nomor klaim, disimpan SEBAGAI SALINAN.
    --
    -- Disalin, bukan dirujuk, supaya jejak ini tetap terbaca utuh bila klaim warisannya
    -- kelak diarsipkan.
    NOMOR_KLAIM         VARCHAR2(50),

    -- Alasan yang diketik pengguna. BOLEH kosong.
    --
    -- Work Owner memilih efek reopen tanpa mewajibkan alasan (2026-09-23). Kolomnya tetap
    -- ada supaya kewajiban itu dapat dinyalakan kelak tanpa perubahan skema — menambah
    -- kolom wajib pada tabel berisi data menuntut nilai bawaan yang mengarang.
    ALASAN              VARCHAR2(1500),

    -- 'menunggu', 'dijalankan', atau 'dibatalkan'.
    --
    -- Aplikasi hanya pernah menulis 'menunggu'. Dua nilai lainnya ditulis pelaksana.
    STATUS              VARCHAR2(20)   DEFAULT 'menunggu' NOT NULL,

    -- Niat yang tercatat: apa yang akan berubah saat permintaan ini dijalankan.
    --
    -- # Kenapa disimpan, padahal aturannya konstan di dalam kode
    --
    -- Supaya permintaan yang dijalankan bulan depan dijalankan menurut aturan yang berlaku
    -- SAAT IA DIAJUKAN. Bila Work Owner kelak mengubah aturannya, baris lama tetap
    -- menyimpan niat aslinya — dan yang menjalankannya tidak perlu menebak aturan mana yang
    -- berlaku. Menyimpan aturan hanya di dalam kode membuat jejaknya hilang begitu kodenya
    -- berubah.
    --
    -- Untuk 'reopen': PYSTATUSWORK -> 'New' dan STATUSCLAIM_1 -> '1164' (Reopen Claim),
    -- ditambah PYREOPENCOUNT + 1 dan PYREOPENTIMESTAMP diisi saat dijalankan.
    EFEK_STATUS_KERJA   VARCHAR2(32),
    EFEK_STATUS_KLAIM   VARCHAR2(10),

    -- Untuk 'salin': lingkup yang disepakati — 'polis_objek_coverage'.
    --
    -- Yaitu snapshot polis, objek pertanggungan, dan coverage; nilai estimasi, usulan,
    -- akseptasi, dan pembayaran TIDAK disalin (Work Owner, 2026-09-23).
    LINGKUP_SALIN       VARCHAR2(40),

    -- Login yang DIKETIK pengguna, dinormalkan huruf besar.
    --
    -- Inilah kunci yang Work Owner tetapkan untuk mencocokkan identitas sesi dengan
    -- `OPERATOR_ID` sistem lama.
    ACTOR_LOGIN         VARCHAR2(64)   NOT NULL,

    -- Nama pemohon, disimpan BERSAMA permintaannya.
    --
    -- Tidak dirujuk ke tabel pengguna: jejak yang namanya diambil lewat join akan BERUBAH
    -- ketika orangnya berganti nama atau catatannya dihapus — dan jejak yang dapat berubah
    -- bukan jejak.
    ACTOR_NAMA          VARCHAR2(100),

    -- Waktu permintaan, dalam UTC.
    --
    -- `DB-8` menetapkan seluruh waktu disimpan UTC, dan konversi ke WIB terjadi di SATU
    -- tempat saja (`F-5`). Ini meninggalkan pola lama yang menambahkan tujuh jam secara
    -- manual di 118 titik pada 36 activity (`R-12`).
    PADA                TIMESTAMP(6) WITH TIME ZONE NOT NULL,

    CONSTRAINT CPNC_PERMINTAAN_KLAIM_PK PRIMARY KEY (ID),

    -- Jenis yang tidak dikenali tidak boleh pernah tersimpan.
    --
    -- Nilainya sudah divalidasi di Go. Diulang di sini karena satu baris ber-JENIS salah
    -- ketik tidak akan pernah terbaca pelaksana maupun layar — permintaannya ada, tetapi
    -- tidak pernah dijalankan dan tidak pernah tampil sebagai tertunda.
    CONSTRAINT CPNC_PERMINTAAN_KLAIM_CK_JENIS
        CHECK (JENIS IN ('reopen', 'salin')),

    CONSTRAINT CPNC_PERMINTAAN_KLAIM_CK_STATUS
        CHECK (STATUS IN ('menunggu', 'dijalankan', 'dibatalkan'))
);

COMMENT ON TABLE POOLDATA.CPNC_PERMINTAAN_KLAIM IS
    'Permintaan ReOpen dan Copy Klaim dari layar Inbox Close Claim. Akun aplikasi hanya INSERT dan SELECT';

-- ---------------------------------------------------------------------------
-- SATU PERMINTAAN MENUNGGU PER (JENIS, KLAIM)
-- ---------------------------------------------------------------------------
--
-- Indeks unik BERBASIS FUNGSI, bukan constraint unik biasa. Alasannya: yang harus unik
-- bukan (JENIS, CASE_ID) — sebuah klaim boleh dibuka kembali hari ini dan diminta lagi
-- tahun depan — melainkan (JENIS, CASE_ID) DI ANTARA YANG MASIH MENUNGGU.
--
-- Oracle tidak mengindeks baris yang seluruh kunci indeksnya NULL. Kedua ekspresi di bawah
-- bernilai NULL begitu STATUS bukan 'menunggu', sehingga baris yang sudah dijalankan atau
-- dibatalkan keluar dari indeks dan tidak lagi menghalangi permintaan baru.
--
-- # Apa yang dijaganya, dan kenapa itu bukan kerapian
--
-- Lapisan usecase sudah memeriksanya lebih dulu, tetapi pemeriksaan itu dan penyimpanannya
-- BUKAN satu operasi atomik: dua permintaan yang tiba bersamaan — dua tab yang terbuka,
-- keduanya ditekan — dapat lolos keduanya.
--
-- Pada 'salin', dua baris yang lolos berarti DUA KLAIM BARU dari satu tombol yang ditekan
-- dua kali.
--
-- Namanya dipakai kode untuk menerjemahkan bentrok menjadi pesan yang dapat dibaca, lewat
-- konstanta `sqlstore.UniqueKeyName`. Menggantinya di sini tanpa mengganti konstanta itu
-- akan membuat permintaan ganda muncul sebagai galat 500.
CREATE UNIQUE INDEX POOLDATA.CPNC_PERMINTAAN_KLAIM_UK_PENDING
    ON POOLDATA.CPNC_PERMINTAAN_KLAIM (
        CASE WHEN STATUS = 'menunggu' THEN JENIS   END,
        CASE WHEN STATUS = 'menunggu' THEN CASE_ID END
    );

-- Layar membaca permintaan tertunda SELURUH baris satu halaman sekaligus, dikunci CASE_ID
-- dan disaring STATUS.
CREATE INDEX POOLDATA.IX_CPNC_PERMINTAAN_KLAIM_CASE
    ON POOLDATA.CPNC_PERMINTAAN_KLAIM (CASE_ID, STATUS);

-- Menelusuri seluruh permintaan seseorang — pertanyaan yang pada layar ini benar-benar akan
-- diajukan, karena `D-59` menghapus pemisahan tugas dan jejak inilah kontrol pengimbangnya.
CREATE INDEX POOLDATA.IX_CPNC_PERMINTAAN_KLAIM_ACTOR
    ON POOLDATA.CPNC_PERMINTAAN_KLAIM (ACTOR_LOGIN, PADA);
