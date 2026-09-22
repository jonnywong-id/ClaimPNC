-- 0002 — Klaim, pohon datanya, tugas, dan penopang alur Register (Oracle 19c)
--
-- Delapan tabel BARU. Tidak satu pun tabel yang dibaca atau ditulis Pega disentuh,
-- sesuai aturan penulis tunggal per tabel selama masa paralel (ADR-0004, P-1).
--
-- Awalan CPNC_ melanjutkan penamaan migrasi 0001, supaya tabel milik aplikasi ini tidak
-- pernah tertukar dengan tabel warisan berawalan T_ atau M_ di skema POOLDATA.
--
-- Seluruh kolom waktu menyimpan UTC. Konversi ke WIB hanya terjadi di aplikasi
-- (docs/Steering/08-TECHNICAL-STRATEGY.md §4.4) — tidak ada penambahan 7 jam di sini.
--
-- UANG disimpan sebagai BILANGAN BULAT SEN, dan PERSENTASE sebagai bilangan bulat
-- 1/10000 persen. Keduanya eksak. ADR-0016 menuntut presisi penuh saat menyimpan dan
-- pembulatan hanya saat menampilkan; tipe pecahan tidak dapat memenuhi itu.
--
-- PERHATIAN: berkas ini BELUM dijalankan di lingkungan mana pun. Menjalankannya menuntut
-- permintaan perubahan skema tertulis, persetujuan Work Owner, dan pelaksanaan oleh DBA
-- (D-63). Akun aplikasi tidak memiliki hak DDL.

-- ── Klaim ────────────────────────────────────────────────────────────────────────

CREATE TABLE CPNC_KLAIM (
    ID                      VARCHAR2(32)   NOT NULL,
    NOMOR                   VARCHAR2(32),
    PORTAL                  VARCHAR2(32),

    POLIS_NOMOR             VARCHAR2(64)   NOT NULL,
    POLIS_LINI              VARCHAR2(8)    NOT NULL,
    POLIS_JENIS_BISNIS      VARCHAR2(64),
    POLIS_MULAI             TIMESTAMP,
    POLIS_AKHIR             TIMESTAMP,
    POLIS_DEKLARASI         CHAR(1)        DEFAULT 'N' NOT NULL,
    POLIS_MATA_UANG         VARCHAR2(8),
    POLIS_PENJAMIN_KREDIT   CHAR(1)        DEFAULT 'N' NOT NULL,
    POLIS_TERTANGGUNG       VARCHAR2(200),
    POLIS_KODE_CABANG       VARCHAR2(32),

    TANGGAL_KEJADIAN        TIMESTAMP,
    TANGGAL_LAPOR           TIMESTAMP,
    TANGGAL_TERIMA_DOKUMEN  TIMESTAMP,

    LOKASI                  VARCHAR2(500),
    KRONOLOGI               CLOB,

    PELAPOR_NAMA            VARCHAR2(200),
    PELAPOR_TELEPON         VARCHAR2(50),
    PELAPOR_EMAIL           VARCHAR2(200),
    PELAPOR_ALAMAT          VARCHAR2(500),
    PELAPOR_HUBUNGAN        NUMBER(3),
    PELAPOR_HUBUNGAN_LAIN   VARCHAR2(200),

    NILAI_ESTIMASI_SEN      NUMBER(20)     DEFAULT 0 NOT NULL,
    MATA_UANG               VARCHAR2(8),
    NOMOR_SLIK              VARCHAR2(64),
    EX_GRATIA               CHAR(1)        DEFAULT 'N' NOT NULL,
    USER_TEKNIS             VARCHAR2(64),
    RCV_ID                  VARCHAR2(64),

    STATUS_PUCL             NUMBER(3)      DEFAULT 0 NOT NULL,
    TRANSFER_COMPLIANCE     CHAR(1)        DEFAULT 'N' NOT NULL,
    MINTA_KEMBALI           CHAR(1)        DEFAULT 'N' NOT NULL,

    STATUS_PROSES           VARCHAR2(16)   NOT NULL,
    STATUS_KLAIM            VARCHAR2(8),
    FLAG_KLAIM              VARCHAR2(4),
    STATUS_POSISI_PROGRES   VARCHAR2(32),

    TAHAP_KINI              VARCHAR2(64),

    DIBUAT_OLEH             VARCHAR2(64)   NOT NULL,
    DIBUAT_PADA             TIMESTAMP      NOT NULL,
    DIUBAH_OLEH             VARCHAR2(64)   NOT NULL,
    DIUBAH_PADA             TIMESTAMP      NOT NULL,
    DIHAPUS_PADA            TIMESTAMP,

    CONSTRAINT PK_CPNC_KLAIM PRIMARY KEY (ID),
    CONSTRAINT UQ_CPNC_KLAIM_NOMOR UNIQUE (NOMOR)
);

-- Empat kolom status, empat konsep yang berbeda (ADR-0018). Namanya sengaja dibuat
-- tidak dapat tertukar; di sistem lama StatusClaim dan ClaimStatus berbeda hanya pada
-- urutan kata.
COMMENT ON COLUMN CPNC_KLAIM.STATUS_PROSES IS 'StatusWork lama: posisi dalam alur kerja';
COMMENT ON COLUMN CPNC_KLAIM.STATUS_KLAIM IS 'StatusClaim lama: 33 kode 1134-1166 berlabel di master';
COMMENT ON COLUMN CPNC_KLAIM.FLAG_KLAIM IS 'ClaimStatus lama: penanda biner; maknanya masih perlu dikonfirmasi';
COMMENT ON COLUMN CPNC_KLAIM.STATUS_POSISI_PROGRES IS 'StatusPosisi lama: On Progress / Done';

-- NOMOR nullable dengan sengaja: klaim yang baru dibuka belum bernomor. Nomor terbit di
-- ujung tahap Input Register, dan tidak dapat ditarik kembali setelah terbit (ADR-0009).
COMMENT ON COLUMN CPNC_KLAIM.NOMOR IS 'PNCN.YY.xxxx (D-71); kosong sampai tahap Input Register lolos validasi';

-- Penghapusan adalah penandaan, bukan penghapusan baris (ADR-0012, D-66).
COMMENT ON COLUMN CPNC_KLAIM.DIHAPUS_PADA IS 'Terisi saat klaim ditandai terhapus; baris tidak pernah dihapus fisik';

-- Index kunci duplikasi. Pemeriksaan klaim ganda berjalan pada setiap penyimpanan
-- registrasi, dan tanpa index ia memindai seluruh tabel klaim (TKT-B02-003).
CREATE INDEX IX_CPNC_KLAIM_DUPLIKAT ON CPNC_KLAIM (POLIS_NOMOR, DIHAPUS_PADA);
CREATE INDEX IX_CPNC_KLAIM_TAHAP ON CPNC_KLAIM (TAHAP_KINI, DIHAPUS_PADA);

-- ── Pohon objek — coverage — spreading ───────────────────────────────────────────
--
-- Ketiga tabel di bawah membawa DIHAPUS_PADA, dan itu keputusan yang perlu dijelaskan.
--
-- Saat petugas menyimpan ulang klaim yang objeknya berkurang, cara yang paling mudah
-- adalah menghapus seluruh baris anak lalu menyisipkannya kembali — persis pola
-- `PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503` di sistem lama. Pola itu DILARANG dua kali:
-- ADR-0012 melarang penghapusan fisik atas data bernilai bisnis, dan ADR-0013 yang akan
-- menetapkan penggantinya masih berstatus Proposed.
--
-- Yang dilakukan di sini karena itu adalah upsert berdasarkan (KLAIM_ID, URUTAN…) lalu
-- MENANDAI baris yang tidak lagi terpakai. Nol DELETE, dan tidak satu pun keputusan
-- ADR-0013 didahului.

CREATE TABLE CPNC_KLAIM_OBJEK (
    KLAIM_ID   VARCHAR2(32)  NOT NULL,
    URUTAN     NUMBER(5)     NOT NULL,
    OBJEK_ID   VARCHAR2(64)  NOT NULL,
    NAMA       VARCHAR2(500),
    LOKASI     VARCHAR2(500),
    DIHAPUS_PADA TIMESTAMP,
    CONSTRAINT PK_CPNC_KLAIM_OBJEK PRIMARY KEY (KLAIM_ID, URUTAN),
    CONSTRAINT FK_CPNC_KLAIM_OBJEK FOREIGN KEY (KLAIM_ID) REFERENCES CPNC_KLAIM (ID)
);

CREATE INDEX IX_CPNC_KLAIM_OBJEK_ID ON CPNC_KLAIM_OBJEK (OBJEK_ID);

CREATE TABLE CPNC_KLAIM_COVERAGE (
    KLAIM_ID           VARCHAR2(32)  NOT NULL,
    URUTAN_OBJEK       NUMBER(5)     NOT NULL,
    URUTAN             NUMBER(5)     NOT NULL,
    COVERAGE_ID        VARCHAR2(64),
    PENYEBAB_KERUGIAN  VARCHAR2(32),
    TSI_SEN            NUMBER(20)    DEFAULT 0 NOT NULL,
    DIHAPUS_PADA       TIMESTAMP,
    CONSTRAINT PK_CPNC_KLAIM_COVERAGE PRIMARY KEY (KLAIM_ID, URUTAN_OBJEK, URUTAN),
    CONSTRAINT FK_CPNC_KLAIM_COVERAGE FOREIGN KEY (KLAIM_ID, URUTAN_OBJEK)
        REFERENCES CPNC_KLAIM_OBJEK (KLAIM_ID, URUTAN)
);

-- Penyebab kerugian 12002 adalah bagian kunci duplikasi khusus lini Personal Accident.
COMMENT ON COLUMN CPNC_KLAIM_COVERAGE.PENYEBAB_KERUGIAN IS 'Kode 12002 ikut menjadi kunci duplikasi pada lini PA';

CREATE TABLE CPNC_KLAIM_SPREADING (
    KLAIM_ID         VARCHAR2(32)  NOT NULL,
    URUTAN_OBJEK     NUMBER(5)     NOT NULL,
    URUTAN_COVERAGE  NUMBER(5)     NOT NULL,
    URUTAN           NUMBER(5)     NOT NULL,
    JENIS_TREATY     VARCHAR2(32),
    NAMA             VARCHAR2(200),
    SHARE_E4         NUMBER(12)    DEFAULT 0 NOT NULL,
    DIHAPUS          CHAR(1)       DEFAULT 'N' NOT NULL,
    OBJEK_FAC_OFFER  VARCHAR2(200),
    DIHAPUS_PADA     TIMESTAMP,
    CONSTRAINT PK_CPNC_KLAIM_SPREADING PRIMARY KEY (KLAIM_ID, URUTAN_OBJEK, URUTAN_COVERAGE, URUTAN),
    CONSTRAINT FK_CPNC_KLAIM_SPREADING FOREIGN KEY (KLAIM_ID, URUTAN_OBJEK, URUTAN_COVERAGE)
        REFERENCES CPNC_KLAIM_COVERAGE (KLAIM_ID, URUTAN_OBJEK, URUTAN)
);

-- 100% disimpan sebagai 1000000. Empat desimal adalah presisi yang ADR-0016 pakai untuk
-- memvalidasi total: ROUND(SUM(share),4) BETWEEN 99.9999 AND 100.0001.
COMMENT ON COLUMN CPNC_KLAIM_SPREADING.SHARE_E4 IS 'Persentase dikali 10000; 100% = 1000000';

-- DIHAPUS adalah penandaan baris spreading di layar, bukan penghapusan fisik (ADR-0012).
COMMENT ON COLUMN CPNC_KLAIM_SPREADING.DIHAPUS IS 'Baris ditandai terhapus; tidak ikut dihitung pada total share';

-- ── Tugas ────────────────────────────────────────────────────────────────────────

CREATE TABLE CPNC_TUGAS (
    ID             VARCHAR2(32)  NOT NULL,
    KLAIM_ID       VARCHAR2(32)  NOT NULL,
    NOMOR_KLAIM    VARCHAR2(32),
    TAHAP          VARCHAR2(64)  NOT NULL,
    ANTREAN        VARCHAR2(16)  NOT NULL,
    WORKBASKET     VARCHAR2(64),
    PEMILIK        VARCHAR2(64),
    DIBUAT_PADA    TIMESTAMP     NOT NULL,
    DIAMBIL_PADA   TIMESTAMP,
    SELESAI_PADA   TIMESTAMP,
    ALASAN_SELESAI VARCHAR2(64),
    CONSTRAINT PK_CPNC_TUGAS PRIMARY KEY (ID),
    CONSTRAINT FK_CPNC_TUGAS_KLAIM FOREIGN KEY (KLAIM_ID) REFERENCES CPNC_KLAIM (ID),
    CONSTRAINT CK_CPNC_TUGAS_ANTREAN CHECK (ANTREAN IN ('WORKLIST', 'WORKBASKET'))
);

-- Satu tugas selalu berada di Worklist ATAU di Workbasket, tidak pernah di keduanya
-- (D-26). Tugas Workbasket lahir tanpa pemilik; ia baru bertuan setelah diambil.
COMMENT ON COLUMN CPNC_TUGAS.PEMILIK IS 'Terisi sejak lahir untuk WORKLIST; kosong untuk WORKBASKET sampai diambil';

-- Index inbox. Kedua kelompok isi inbox — tugas milik saya, dan tugas antrean yang belum
-- bertuan — dibaca pada setiap pemuatan layar.
CREATE INDEX IX_CPNC_TUGAS_PEMILIK ON CPNC_TUGAS (PEMILIK, SELESAI_PADA);
CREATE INDEX IX_CPNC_TUGAS_ANTREAN ON CPNC_TUGAS (WORKBASKET, SELESAI_PADA);
CREATE INDEX IX_CPNC_TUGAS_KLAIM ON CPNC_TUGAS (KLAIM_ID, SELESAI_PADA);

-- ── Nomor klaim ──────────────────────────────────────────────────────────────────

CREATE TABLE CPNC_NOMOR_KLAIM (
    TAHUN    NUMBER(4)  NOT NULL,
    TERAKHIR NUMBER(10) DEFAULT 0 NOT NULL,
    CONSTRAINT PK_CPNC_NOMOR_KLAIM PRIMARY KEY (TAHUN)
);

-- Pencacah per TAHUN, bukan satu sequence global. Format PNCN.YY.xxxx (D-71) membawa
-- tahun di dalam nomornya; tanpa reset tahunan segmen itu tidak membedakan apa pun.
--
-- CATATAN: apakah urutan memang direset tiap tahun, dan apakah lebar segmen terakhir
-- dibuat tetap, masih pertanyaan terbuka TKT-F2-006. Bentuk tabel ini mengikuti tafsir
-- yang paling langsung dari formatnya; mengubahnya kelak menuntut migrasi tersendiri.
COMMENT ON TABLE CPNC_NOMOR_KLAIM IS 'Pencacah nomor klaim per tahun; dikunci SELECT FOR UPDATE saat menerbitkan';

-- ── Jejak audit ──────────────────────────────────────────────────────────────────

CREATE TABLE CPNC_JEJAK_AUDIT (
    ID          VARCHAR2(32)  NOT NULL,
    KLAIM_ID    VARCHAR2(32),
    NOMOR_KLAIM VARCHAR2(32),
    PERISTIWA   VARCHAR2(64)  NOT NULL,
    PELAKU      VARCHAR2(64)  NOT NULL,
    PADA        TIMESTAMP     NOT NULL,
    KETERANGAN  VARCHAR2(2000),
    CONSTRAINT PK_CPNC_JEJAK_AUDIT PRIMARY KEY (ID)
);

-- Jejak audit hanya bertambah; ia tidak pernah diubah maupun dihapus (ADR-0026).
-- Pencabutan hak UPDATE dan DELETE atas tabel ini adalah bagian dari permintaan skema.
COMMENT ON TABLE CPNC_JEJAK_AUDIT IS 'Append-only (ADR-0026); akun aplikasi tidak boleh punya hak UPDATE/DELETE';

CREATE INDEX IX_CPNC_JEJAK_AUDIT_KLAIM ON CPNC_JEJAK_AUDIT (KLAIM_ID, PADA);

-- ── Pemberitahuan (outbox) ───────────────────────────────────────────────────────

CREATE TABLE CPNC_NOTIFIKASI (
    ID           VARCHAR2(32)   NOT NULL,
    JENIS        VARCHAR2(64)   NOT NULL,
    NOMOR_KLAIM  VARCHAR2(32),
    NOMOR_POLIS  VARCHAR2(64),
    PENERIMA     VARCHAR2(2000),
    NILAI_SEN    NUMBER(20)     DEFAULT 0 NOT NULL,
    DIBUAT_PADA  TIMESTAMP      NOT NULL,
    DIKIRIM_PADA TIMESTAMP,
    CONSTRAINT PK_CPNC_NOTIFIKASI PRIMARY KEY (ID)
);

-- Tabel ini adalah KOTAK KELUAR, bukan pengirim. Baris disisipkan di dalam transaksi
-- yang sama dengan penyimpanan klaim, sehingga klaim yang melampaui ambang menerbitkan
-- TEPAT SATU peristiwa Notice of Large Losses (TKT-B02-004). Pengirimannya milik S-3,
-- yang membaca tabel ini di luar transaksi.
COMMENT ON TABLE CPNC_NOTIFIKASI IS 'Kotak keluar peristiwa; pengiriman dilakukan modul S-3';
