-- =====================================================================
-- DICABUT 2026-09-23 — JANGAN DIJALANKAN
-- =====================================================================
--
-- Work Owner menetapkan berkas laporan TIDAK ditulis ke tabel milik aplikasi ini,
-- melainkan ke POOLDATA.T_CLAIM_RECIVEDCLAIM — tabel bisnis yang sudah dipakai Pega
-- lewat Database/PROCINSERTDATARECIVEDKLAIM.prc.
--
-- Berkas ini dipertahankan sebagai REKAMAN rancangan yang pernah diambil, bukan
-- sebagai langkah pemasangan. Menjalankannya akan membuat tabel yang tidak dibaca
-- maupun ditulis satu baris kode pun.
--
-- Alasan lengkapnya di docs/keputusan-implementasi.md.
-- =====================================================================

-- 0003 — Tabel berkas laporan klaim milik aplikasi (Oracle 19c)
--
-- Satu tabel BARU dan satu sequence BARU. Tidak satu pun tabel yang dibaca atau ditulis
-- Pega disentuh berkas ini, sesuai aturan penulis tunggal per tabel selama masa paralel
-- (ADR-0004, P-1).
--
-- ============================================================================
-- KENAPA TABEL INI ADA
-- ============================================================================
--
-- Layar Inbox Laporan Klaim di sistem lama menyimpan kepala berkasnya di
-- DATAPEGA.PC_ASM_FW_GCNMFW_WORK — tabel milik ENGINE Pega, yang dibaca 116 rule.
-- Work Owner menetapkan 2026-09-19 bahwa aplikasi ini TIDAK LAGI menulis ke sana.
--
-- Tabel ini menggantikan perannya untuk berkas yang diterbitkan aplikasi ini. Berkas
-- lama tetap tinggal dan tetap dibaca di tempatnya; daftar yang dilihat petugas adalah
-- gabungan keduanya. Pembagiannya karena itu bersih: satu baris punya tepat satu
-- penulis, dan tidak ada baris yang dimiliki dua sistem.
--
-- Awalan CPNC_ dipakai supaya tabel baru aplikasi ini tidak pernah tertukar dengan tabel
-- warisan berawalan T_ atau M_ di skema POOLDATA.
--
-- ============================================================================
-- DI BASIS DATA MANA IA DIBUAT
-- ============================================================================
--
-- Di SETIAP portal entitas, bukan hanya di portal utama — berbeda dari migrasi 0001.
-- Berkas laporan adalah data bisnis milik satu badan hukum, dan ADR-0030 menetapkan
-- pemisahannya ada di tingkat koneksi. Menaruhnya di satu basis data bersama berarti
-- pemisahan bergantung pada tidak adanya satu pun kueri yang lupa menyaring, dan itu
-- kelas kesalahan yang tidak dapat diuji habis (R-20).
--
-- Akibat yang harus disiapkan: berkas ini dijalankan EMPAT KALI, dan gagal di salah
-- satunya membuat portal tersebut tertinggal versi.
--
-- ============================================================================
-- WAKTU
-- ============================================================================
--
-- Seluruh kolom waktu menyimpan UTC. Konversi ke WIB hanya terjadi di aplikasi
-- (docs/Steering/08-TECHNICAL-STRATEGY.md §4.4) — tidak ada penambahan 7 jam di sini.
--
-- PERHATIAN: berkas ini BELUM dijalankan di lingkungan mana pun. Menjalankannya menuntut
-- permintaan perubahan skema tertulis, persetujuan Work Owner, dan pelaksanaan oleh DBA
-- (D-63). Akun aplikasi tidak memiliki hak DDL.

CREATE TABLE CPNC_LAPORAN_KLAIM (
    NO_LAPORAN        VARCHAR2(32)   NOT NULL,
    NO_KLAIM          VARCHAR2(32),
    NO_POLIS          VARCHAR2(64),
    NAMA_TERTANGGUNG  VARCHAR2(255),
    NAMA_PELAPOR      VARCHAR2(255),
    NAMA_BISNIS       VARCHAR2(255),
    NO_REFERENSI      VARCHAR2(64),
    GROUP_PANEL       VARCHAR2(8),
    KODE_GROUP_BISNIS VARCHAR2(16),
    TGL_KEJADIAN      DATE,
    TGL_AGING         TIMESTAMP      NOT NULL,
    ALASAN            VARCHAR2(1000),
    SUBJEK_EMAIL      VARCHAR2(1000),
    KODE_CABANG       VARCHAR2(32)   NOT NULL,
    STS_DISERAHKAN    CHAR(1)        DEFAULT '0' NOT NULL,
    STATUS_KERJA      VARCHAR2(64),
    DIBUAT_OLEH       VARCHAR2(64)   NOT NULL,
    DIBUAT_PADA       TIMESTAMP      NOT NULL,
    DIUBAH_OLEH       VARCHAR2(64),
    DIUBAH_PADA       TIMESTAMP,
    DIHAPUS_PADA      TIMESTAMP,
    DIHAPUS_OLEH      VARCHAR2(64),
    CONSTRAINT PK_CPNC_LAPORAN_KLAIM PRIMARY KEY (NO_LAPORAN),
    CONSTRAINT CK_CPNC_LAPORAN_DISERAHKAN CHECK (STS_DISERAHKAN IN ('0', '1'))
);

COMMENT ON TABLE CPNC_LAPORAN_KLAIM IS 'Berkas laporan klaim masuk yang diterbitkan aplikasi Claim PNC; pengganti peran DATAPEGA.PC_ASM_FW_GCNMFW_WORK untuk berkas baru';

-- NO_LAPORAN berbentuk RCVN.YY.xxxx, mengikuti pola nomor klaim PNCN.YY.xxxx (D-71).
-- Awalan RCVN membuat asal sebuah berkas terbaca langsung dari nomornya tanpa tabel
-- pemetaan — sifat yang berharga selama masa paralel yang panjang.
COMMENT ON COLUMN CPNC_LAPORAN_KLAIM.NO_LAPORAN IS 'RCVN.YY.xxxx; berkas warisan Pega memakai bentuk pyID yang lain';

-- STS_DISERAHKAN menggantikan statuslock_1 pada tabel Pega, yang di sana menyimpan
-- pzInsKey penugasan dan dibaca kesembilan kueri HANYA sebagai "ada" atau "tidak ada".
-- Yang disimpan di sini karena itu maknanya, bukan kunci teknis milik engine lain.
COMMENT ON COLUMN CPNC_LAPORAN_KLAIM.STS_DISERAHKAN IS '1 bila berkas sudah diserahkan ke petugas klaim; menggantikan statuslock_1';

-- NAMA_PELAPOR diisi nama PETUGAS saat berkas dibuat, mengikuti CreateNewCaseRCV yang
-- mengisi Sender dengan OperatorID.pyUserName. Nama pelapor yang sebenarnya menimpanya
-- di layar rinci (B-14), yang belum dibangun.
--
-- Pasangannya di sistem lama — TelpPengirim dari OperatorID.pyTelephone — SENGAJA tidak
-- dibuatkan kolom: kontrak HCC/HCQ tidak memuat nomor telepon sama sekali, sehingga
-- kolomnya akan selamanya kosong dan tampak seperti data yang belum diisi.
COMMENT ON COLUMN CPNC_LAPORAN_KLAIM.NAMA_PELAPOR IS 'Nama petugas pembuat saat berkas lahir; ditimpa nama pelapor sebenarnya di layar rinci';

-- Empat kolom jejak dan dua kolom penghapusan ada sejak baris pertama. ADR-0012
-- melarang penghapusan fisik data bernilai bisnis, dan menambah kolomnya belakangan
-- menuntut perubahan skema di tengah masa paralel — yang menempuh tiga pihak (D-63).
COMMENT ON COLUMN CPNC_LAPORAN_KLAIM.DIHAPUS_PADA IS 'Penanda soft delete (ADR-0012); NULL berarti baris masih berlaku';

-- Index pengurutan utama. KESEMBILAN kueri layar mengurutkan ORDER BY aging DESC, dan
-- itu pula yang dipakai memotong halaman — tanpa index ini, setiap pindah halaman
-- memaksa basis data mengurutkan seluruh tabel lebih dulu.
CREATE INDEX IX_CPNC_LAPORAN_AGING ON CPNC_LAPORAN_KLAIM (TGL_AGING DESC, NO_LAPORAN DESC);

-- Index batas data. Penyaring cabang berlaku di SETIAP permintaan daftar; ia bukan
-- kenyamanan melainkan batas data (lihat inboxlaporanklaim.Caller.BranchCode).
CREATE INDEX IX_CPNC_LAPORAN_CABANG ON CPNC_LAPORAN_KLAIM (KODE_CABANG);

-- Index ketiga tab komunikasi, yang seluruhnya menyaring berkas milik pemanggil sendiri.
CREATE INDEX IX_CPNC_LAPORAN_PEMBUAT ON CPNC_LAPORAN_KLAIM (DIBUAT_OLEH);

-- Sequence TERSENDIRI, bukan POOLDATA.CLAIM_NO_NONPEGA_SEQ yang D-71 peruntukkan bagi
-- nomor klaim. Dua deret untuk dua hal yang berbeda, supaya nomor klaim dan nomor
-- laporan tidak saling memakan urutan — dan supaya lubang pada salah satunya tidak
-- terbaca sebagai lubang pada yang lain.
--
-- NOCACHE dipilih dengan sengaja. Cache membuat nomor melompat saat basis data
-- di-restart, dan pada deret yang nomornya dibaca manusia, lompatan itu akan terus
-- ditanyakan. Biayanya satu perjalanan per penerbitan, dan berkas laporan tidak
-- diterbitkan dalam laju yang membuat itu terasa.
CREATE SEQUENCE CPNC_LAPORAN_KLAIM_SEQ START WITH 1 INCREMENT BY 1 NOCACHE NOCYCLE;

-- ============================================================================
-- YANG SENGAJA TIDAK ADA DI BERKAS INI
-- ============================================================================
--
-- 1. FOREIGN KEY ke POOLDATA.BRANCH. Tabel itu milik sistem lama dan dibaca banyak rule;
--    menambahkan constraint padanya berarti menyentuh objek milik penulis lain.
--    Keabsahan kode cabang dijaga aplikasi saat berkas dibuat.
--
-- 2. FOREIGN KEY ke T_CLAIM_PNC lewat NO_KLAIM. Alasannya sama, ditambah satu: berkas
--    laporan lahir SEBELUM klaimnya ada, sehingga kolomnya memang kosong pada sebagian
--    besar umur barisnya.
--
-- 3. Baris apa pun. Tabel ini lahir kosong; berkas lama tidak dipindahkan ke sini dan
--    tetap dibaca di tempatnya (P-3: berkas yang sudah berjalan diselesaikan di sistem
--    tempat ia dimulai).
