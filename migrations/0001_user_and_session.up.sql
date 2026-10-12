-- 0001 — Tabel pengguna lokal dan sesi aktif (Oracle 19c)
--
-- Dua tabel BARU. Tidak satu pun tabel yang dibaca atau ditulis Pega disentuh, sesuai
-- aturan penulis tunggal per tabel selama masa paralel (ADR-0004). Tiga tabel warisan
-- yang dipakai saat masuk — POOLDATA.M_PORTAL_PNC, POOLDATA.M_LOGIN_PNC, dan
-- POOLDATA.GCNM_CONNECT_REST — hanya DIBACA dan tidak disentuh berkas ini.
--
-- Awalan CPNC_ dipakai supaya tabel baru aplikasi ini tidak pernah tertukar dengan
-- tabel warisan berawalan T_ atau M_ di skema POOLDATA.
--
-- Kedua tabel ini hidup di basis data PORTAL UTAMA, bukan di setiap portal: ADR-0030
-- menetapkan berpindah portal tidak menuntut login ulang, dan itu hanya mungkin bila
-- sesinya tidak ikut berpindah basis data.
--
-- Seluruh kolom waktu menyimpan UTC. Konversi ke WIB hanya terjadi di aplikasi
-- (docs/Steering/08-TECHNICAL-STRATEGY.md §4.4) — tidak ada penambahan 7 jam di sini.
--
-- PERHATIAN: berkas ini BELUM dijalankan di lingkungan mana pun. Menjalankannya
-- menuntut permintaan perubahan skema tertulis, persetujuan Work Owner, dan pelaksanaan
-- oleh DBA (D-63). Akun aplikasi tidak memiliki hak DDL.

CREATE TABLE CPNC_PENGGUNA (
    IDENTITAS       VARCHAR2(64)   NOT NULL,
    JENIS           VARCHAR2(16)   NOT NULL,
    NAMA            VARCHAR2(200)  NOT NULL,
    LOGIN           VARCHAR2(200),
    EMAIL           VARCHAR2(200),
    PERUSAHAAN      VARCHAR2(64),
    CABANG          VARCHAR2(200),
    KODE_CABANG     VARCHAR2(32),
    JABATAN         VARCHAR2(200),
    OPERATOR_ID     VARCHAR2(64),
    AKTIF           CHAR(1)        DEFAULT 'Y' NOT NULL,
    DIBUAT_PADA     TIMESTAMP      NOT NULL,
    DIPERBARUI_PADA TIMESTAMP      NOT NULL,
    CONSTRAINT PK_CPNC_PENGGUNA PRIMARY KEY (IDENTITAS),
    CONSTRAINT CK_CPNC_PENGGUNA_AKTIF CHECK (AKTIF IN ('Y', 'N')),
    CONSTRAINT CK_CPNC_PENGGUNA_JENIS CHECK (JENIS IN ('KARYAWAN', 'NON_KARYAWAN'))
);

-- IDENTITAS adalah kunci alami yang berlaku untuk KEDUA jenis pengguna: NIK dari
-- HCC/HCQ untuk karyawan, LOGIN_ID dari POOLDATA.M_LOGIN_PNC untuk non-karyawan.
-- Kolomnya tidak dinamai NIK karena broker dan surveyor independen tidak punya NIK.
COMMENT ON COLUMN CPNC_PENGGUNA.IDENTITAS IS 'NIK untuk KARYAWAN, LOGIN_ID untuk NON_KARYAWAN';

-- Lima kolom berikut NULLABLE dengan sengaja. Kontrak HCC/HCQ mengisi seluruhnya;
-- POOLDATA.M_LOGIN_PNC hanya memuat login_id dan login_name sehingga sisanya kosong.
-- Memaksanya NOT NULL akan menolak seluruh pengguna non-karyawan.
COMMENT ON COLUMN CPNC_PENGGUNA.EMAIL IS 'HCQ EmpResponse.Person.pyEmail1; kosong untuk non-karyawan';
COMMENT ON COLUMN CPNC_PENGGUNA.CABANG IS 'HCQ EmpResponse.Placement.BranchName; kosong untuk non-karyawan';
COMMENT ON COLUMN CPNC_PENGGUNA.KODE_CABANG IS 'HCQ EmpResponse.Placement.BranchCode; dasar batas data per cabang (11-SECURITY 3.2)';
COMMENT ON COLUMN CPNC_PENGGUNA.JABATAN IS 'HCQ EmpResponse.Placement.PositionName; kosong untuk non-karyawan';

-- AKTIF dimiliki administrator aplikasi ini, BUKAN sistem identitas luar. Ia tidak
-- ditimpa setiap kali pengguna masuk; menimpanya akan menghidupkan kembali akun yang
-- sengaja dinonaktifkan.
COMMENT ON COLUMN CPNC_PENGGUNA.AKTIF IS 'Dikelola administrator Claim PNC; tidak pernah ditimpa oleh HCC/HCQ';

-- OPERATOR_ID sengaja dibiarkan kosong dan nullable. Cara mencocokkan identitas
-- HCC/HCQ dengan OPERATOR_ID yang dipakai seluruh data klaim BELUM ditetapkan
-- (ADR-0024, pertanyaan terbuka nomor 4). Kolomnya disediakan; pengisiannya menunggu
-- keputusan itu.
COMMENT ON COLUMN CPNC_PENGGUNA.OPERATOR_ID IS 'Menunggu keputusan pemetaan identitas HCC/HCQ ke OPERATOR_ID (ADR-0024)';

CREATE INDEX IX_CPNC_PENGGUNA_LOGIN ON CPNC_PENGGUNA (LOGIN);

CREATE TABLE CPNC_SESI_AKTIF (
    ID               VARCHAR2(32) NOT NULL,
    SIDIK_TOKEN      VARCHAR2(64) NOT NULL,
    IDENTITAS        VARCHAR2(64) NOT NULL,
    DITERBITKAN_PADA TIMESTAMP    NOT NULL,
    BERLAKU_SAMPAI   TIMESTAMP    NOT NULL,
    DICABUT_PADA     TIMESTAMP,
    CONSTRAINT PK_CPNC_SESI_AKTIF PRIMARY KEY (ID),
    CONSTRAINT UQ_CPNC_SESI_AKTIF_SIDIK UNIQUE (SIDIK_TOKEN),
    CONSTRAINT FK_CPNC_SESI_AKTIF_PENGGUNA FOREIGN KEY (IDENTITAS) REFERENCES CPNC_PENGGUNA (IDENTITAS)
);

-- Yang disimpan adalah SIDIK token, bukan tokennya. Bocornya isi tabel ini tidak
-- dengan sendirinya memberi orang lain sesi yang dapat dipakai.
COMMENT ON COLUMN CPNC_SESI_AKTIF.SIDIK_TOKEN IS 'Sidik SHA-256 token sesi; token mentah tidak pernah disimpan';

-- Pencabutan adalah penandaan, bukan penghapusan baris (ADR-0012). Sesi yang pernah
-- ada tetap dapat ditelusuri jejak audit.
COMMENT ON COLUMN CPNC_SESI_AKTIF.DICABUT_PADA IS 'Terisi saat sesi dicabut; baris tidak pernah dihapus fisik (ADR-0012)';

CREATE INDEX IX_CPNC_SESI_AKTIF_IDENT ON CPNC_SESI_AKTIF (IDENTITAS);
