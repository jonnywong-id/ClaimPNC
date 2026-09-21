-- 0003 — Pelaporan Klaim: tabel baru milik aplikasi (Oracle 19c)
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- # Apa yang dibuat, dan apa yang TIDAK disentuh
--
-- Berkas ini hanya MENAMBAH objek baru:
--
--   * POOLDATA.CPNC_LAPORAN_KLAIM       tabel baru
--   * POOLDATA.CPNC_LAPORAN_KLAIM_SEQ   urutan nomor laporan
--   * tiga indeks pendukung
--
-- TIDAK ADA satu pun objek milik sistem lama yang diubah. Berbeda dari migrasi 0002 yang
-- mendefinisikan ulang view yang dibaca 23 rule Pega, berkas ini tidak menyentuh apa pun
-- yang sedang melayani produksi. Risikonya karena itu jauh lebih rendah — tetapi ia tetap
-- menempuh prosedur `D-63`: permintaan tertulis tim pengembang, persetujuan Work Owner,
-- pelaksanaan DBA. Akun aplikasi tidak memiliki hak DDL.
--
-- BERKAS INI BELUM PERNAH DIJALANKAN DI LINGKUNGAN MANA PUN.
--
--
-- # Kenapa tabel baru, bukan tabel yang sudah ada
--
-- Keputusan Work Owner 2026-09-18: `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` tidak dipakai lagi,
-- dan tabel baru dibuatkan.
--
-- Data laporan di sistem lama hidup di DUA tempat, dan keduanya tidak dapat dipakai:
--
--   1. Header case-nya ada di `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` — tabel milik ENGINE Pega
--      yang dibaca 116 rule dan tetap ditulis Pega untuk seluruh case type lain yang
--      belum bermigrasi. `P-1` melarang dua sistem menulis satu tabel.
--   2. Rinciannya ada di `POOLDATA.T_CLAIM_RECIVEDCLAIM`. Penulis tunggalnya memang
--      terbukti — hanya `PROCINSERTDATARECIVEDKLAIM` yang menyentuhnya, dan hanya satu
--      rule yang memanggilnya — tetapi ia HANYA MEMUAT SEBAGIAN. Penanda transfer,
--      estimasi kerugian, tipe klaim, dan jumlah dokumen tidak punya kolom di sana;
--      keempatnya hanya hidup di blob case Pega.
--
--
-- # Satu hal yang harus dijawab SEBELUM modul ini menyala di produksi
--
-- Penelusuran seluruh export TIDAK menemukan satu pun rule yang MEMBACA
-- `POOLDATA.T_CLAIM_RECIVEDCLAIM`. Sejauh yang terlihat, tabel itu write-only.
--
-- Bila ada pembaca DI LUAR export — laporan BI, perkakas cabang, atau kueri manual yang
-- dijalankan berkala — pembaca itu akan berhenti menerima baris baru begitu modul ini
-- menyala, dan berhentinya TIDAK menimbulkan galat apa pun. Ia hanya akan terlihat
-- sebagai laporan yang jumlahnya berhenti bertambah.
--
-- Pertanyaan untuk Work Owner dan DBA, bukan sesuatu yang dapat dijawab dari export:
--
--     SELECT owner, name, type FROM all_dependencies
--      WHERE referenced_owner = 'POOLDATA'
--        AND referenced_name  = 'T_CLAIM_RECIVEDCLAIM';
--
--
-- # Yang HARUS diperiksa DBA sebelum menjalankan
--
-- 1. PASTIKAN NAMANYA BELUM DIPAKAI. Ketiga objek di bawah bernama awalan CPNC_ supaya
--    tidak mungkin tertukar dengan tabel warisan berawalan T_ atau M_ (lihat migrasi
--    0001), tetapi pemeriksaannya tetap murah:
--
--        SELECT object_name, object_type FROM all_objects
--         WHERE owner = 'POOLDATA' AND object_name LIKE 'CPNC_LAPORAN_KLAIM%';
--
--    Yang diharapkan nol baris.
--
-- 2. PASTIKAN TABLESPACE-nya sesuai kebijakan. Berkas ini sengaja TIDAK menyebut
--    tablespace: menyebutkannya berarti menebak tata letak penyimpanan yang dimiliki DBA.


-- ---------------------------------------------------------------------------
-- Langkah 1 — tabel laporan klaim.
--
-- # Tipe kolom mengikuti 09-DATABASE-STRATEGY.md §5
--
--   * Teks pendek  -> VARCHAR2(n) dengan panjang eksplisit
--   * Teks panjang -> VARCHAR2(4000), bukan CLOB: batas domainnya 4.000 karakter, dan
--                     VARCHAR2 dapat diindeks serta dibandingkan tanpa perlakuan khusus
--   * Boolean      -> NUMBER(1), dipetakan di adapter. Tabel ini baru, sehingga tidak ada
--                     sandi warisan "Ya"/"Tidak" yang harus dihormati seperti LST_ACCOUNT
--   * Waktu        -> TIMESTAMP WITH TIME ZONE, disimpan UTC (DB-8)
--   * Tanggal murni-> DATE. Tanggal kejadian dan tanggal terima dokumen menjawab
--                     "hari apa", bukan "detik ke berapa"
--
-- # Satu cacat sistem lama yang sengaja TIDAK diwarisi
--
-- Kolom TANGGALTERIMADOKUMEN pada T_CLAIM_RECIVEDCLAIM bertipe VARCHAR2 — tanggal
-- disimpan sebagai TEKS (`Database/PROCINSERTDATARECIVEDKLAIM.prc:4`). Akibatnya
-- pengurutan tanggal menjadi pengurutan teks dan penyaringan rentang tidak dapat memakai
-- indeks; persis cacat yang 09-DATABASE-STRATEGY §3.2 perintahkan dihapus. Di sini ia
-- DATE.
--
-- # Kenapa hampir seluruh kolom nullable
--
-- Bukan kelonggaran, melainkan sifat pekerjaannya. Laporan kerugian datang lewat telepon
-- dan surel dengan kelengkapan yang berbeda-beda, dan petugas harus dapat mencatatnya
-- SEKARANG lalu melengkapinya kemudian. Layar Pega pun tidak mewajibkan satu field pun.
-- Kelengkapan yang sesungguhnya ditegakkan saat REGISTRASI (B-2), tempat invarian I-2
-- sampai I-10 berlaku.
--
-- Yang NOT NULL hanya lima: nomor, nama pelapor, penanda transfer, dan dua waktu jejak.
-- ---------------------------------------------------------------------------

CREATE TABLE POOLDATA.CPNC_LAPORAN_KLAIM (
    NOMOR                    VARCHAR2(20)              NOT NULL,

    -- Pelapor
    NAMA_PELAPOR             VARCHAR2(100)             NOT NULL,
    EMAIL_PENGIRIM           VARCHAR2(100),
    TELEPON_PENGIRIM         VARCHAR2(30),
    NAMA_KURIR               VARCHAR2(100),
    SUBJEK_EMAIL             VARCHAR2(100),

    -- Polis dan tertanggung SEBAGAIMANA DISEBUT PELAPOR.
    -- Ini bukan snapshot polis: snapshot yang sah baru terbentuk saat registrasi (D-04).
    -- Pelapor sering menyebut nomor polis yang keliru, dan laporannya tetap harus dapat
    -- dicatat.
    NOMOR_POLIS              VARCHAR2(50),
    NAMA_TERTANGGUNG         VARCHAR2(100),
    EMAIL_TERTANGGUNG        VARCHAR2(100),
    KODE_BISNIS              VARCHAR2(20),
    GROUP_PANEL              VARCHAR2(20),
    NOMOR_REFERENSI          VARCHAR2(50),

    -- Kerugian yang dilaporkan
    TANGGAL_KEJADIAN         DATE,
    LOKASI_KEJADIAN          VARCHAR2(4000),
    KRONOLOGI                VARCHAR2(4000),
    RINCIAN_KERUSAKAN        VARCHAR2(4000),
    SIM_PENGENDARA           VARCHAR2(50),

    -- NILAI_ESTIMASI adalah VARCHAR2, dan itu PENYIMPANGAN SADAR dari
    -- 09-DATABASE-STRATEGY.md §5 yang menetapkan nilai uang bertipe NUMBER(18,2).
    --
    -- Alasannya bukan kemudahan. Aplikasi membawa nilai ini sebagai TEKS DESIMAL, karena
    -- pustaka standar Go tidak punya tipe desimal dan `float64` akan membulatkan diam-diam
    -- — hal yang I-12 larang. Menuliskan teks itu ke kolom NUMBER menyerahkan konversinya
    -- kepada Oracle, yang memakai NLS_NUMERIC_CHARACTERS: pada sesi yang pemisah
    -- desimalnya koma, "1234.56" akan DITOLAK. Kegagalan itu bergantung lingkungan, dan
    -- di mesin tempat berkas ini ditulis TIDAK ADA basis data untuk membuktikannya.
    --
    -- Memilih tipe yang kegagalannya tidak dapat saya deteksi adalah pilihan yang salah.
    -- Teks menyimpan angka yang diketik pengguna PERSIS seperti adanya, tanpa konversi
    -- dan tanpa kehilangan satu digit pun. Bentuknya dipagari CHECK di bawah dan
    -- diperiksa lagi di domain (pelaporanklaim.NilaiUangMasukAkal).
    --
    -- BATAS YANG HARUS DISADARI: kolom ini TIDAK dapat dijumlahkan atau dibandingkan
    -- sebagai angka di SQL. Itu dapat diterima karena nilai ini TIDAK dipakai perhitungan
    -- apa pun — ambang komite dan Notice of Large Losses dihitung dari nilai pada KLAIM
    -- (B-5), bukan dari perkiraan pelapor.
    --
    -- PERTANYAAN TERBUKA untuk Work Owner: apakah pustaka desimal boleh ditambahkan
    -- sebagai dependensi? Bila ya, kolom ini menjadi NUMBER(18,2) lewat migrasi
    -- tersendiri, dan yang berubah hanya adapter — domain sudah membawanya sebagai teks
    -- desimal yang eksak.
    NILAI_ESTIMASI           VARCHAR2(30),
    TIPE_KLAIM               VARCHAR2(20),

    JUMLAH_DOKUMEN           NUMBER(5)      DEFAULT 0,
    TANGGAL_TERIMA_DOKUMEN   DATE,

    -- Daur hidup.
    --
    -- Tahap TIDAK disimpan sebagai kolom; ia dihitung dari ketiga kolom di bawah, persis
    -- seperti sistem lama menurunkannya dari kombinasi PNCCASEID dan STATUSLOCK_1
    -- (`RDB List/BrowseClaimRCV_Aksep-SQL.xml`). Menyimpannya akan membuat dua sumber
    -- kebenaran yang dapat berselisih tanpa ada yang menyadarinya.
    NOMOR_KLAIM              VARCHAR2(30),
    DITRANSFER               NUMBER(1)      DEFAULT 0   NOT NULL,
    TANGGAL_TRANSFER         TIMESTAMP WITH TIME ZONE,
    TANGGAL_REGISTRASI       TIMESTAMP WITH TIME ZONE,
    ALASAN_BELUM_TRANSFER    VARCHAR2(4000),
    CATATAN_BELUM_REGISTRASI VARCHAR2(4000),

    -- HASIL_KLAIM diisi MODUL KLAIM (B-5 dan B-10), bukan modul ini. Ia kosong untuk
    -- seluruh laporan sampai kedua modul itu ada.
    HASIL_KLAIM              VARCHAR2(20),

    -- Jejak
    KODE_CABANG              VARCHAR2(20),
    DIINPUT_OLEH             VARCHAR2(100),
    DIINPUT_PADA             TIMESTAMP WITH TIME ZONE   NOT NULL,
    DIUBAH_PADA              TIMESTAMP WITH TIME ZONE   NOT NULL,

    -- Nama constraint dipakai kode Go untuk menerjemahkan galat bentrok menjadi pesan
    -- yang dapat dibaca pengguna (konstanta NamaKunciUtama di
    -- internal/pelaporanklaim/repo/sqlstore/laporan.go). Mengganti namanya di sini tanpa
    -- mengganti konstanta itu akan membuat bentrok nomor muncul sebagai galat 500.
    CONSTRAINT CPNC_LAPORAN_KLAIM_PK PRIMARY KEY (NOMOR),

    -- Sandi kolom dipagari di basis data, bukan hanya dipercayakan pada kode. Nilai di
    -- luar daftar tidak akan membuat apa pun gagal secara terlihat — ia hanya membuat
    -- laporannya jatuh ke tahap yang salah, dan kegagalan diam seperti itu baru ketahuan
    -- di produksi. Pelajaran ini tercatat pada Master Rekening
    -- (`keputusan-implementasi.md` §10.9).
    CONSTRAINT CPNC_LAPORAN_KLAIM_CK_TRF  CHECK (DITRANSFER IN (0, 1)),
    CONSTRAINT CPNC_LAPORAN_KLAIM_CK_HSL  CHECK (HASIL_KLAIM IS NULL
                                                 OR HASIL_KLAIM IN ('DIAKSEPTASI', 'DITOLAK')),
    CONSTRAINT CPNC_LAPORAN_KLAIM_CK_JML  CHECK (JUMLAH_DOKUMEN >= 0),

    -- Estimasi wajib berbentuk angka desimal tanpa tanda dan tanpa pemisah ribuan.
    -- Ekspresinya harus SAMA ARTINYA dengan pelaporanklaim.NilaiUangMasukAkal di Go;
    -- bila keduanya berbeda, aplikasi akan menerima nilai yang kemudian ditolak basis
    -- data sebagai galat 500.
    CONSTRAINT CPNC_LAPORAN_KLAIM_CK_EST  CHECK (NILAI_ESTIMASI IS NULL
                                                 OR REGEXP_LIKE(NILAI_ESTIMASI,
                                                                '^[0-9]{1,16}(\.[0-9]{1,2})?$'))
);


-- ---------------------------------------------------------------------------
-- Langkah 2 — urutan nomor laporan.
--
-- Nomornya berbentuk LPK.YY.xxxx, mengikuti bentuk nomor klaim yang D-71 tetapkan
-- (PNCN.YY.xxxx). Bentuk lama TIDAK dapat ditiru: ia ditentukan pyWorkIDPrefix pada rule
-- kelas Pega, yang tidak ada di export, dan pencarian seluruh export tidak menemukan satu
-- pun contoh nilainya.
--
-- Urutan ini TIDAK direset tiap tahun, sama seperti urutan nomor klaim. Akibatnya nomor
-- urut menembus pergantian tahun dan segmen tahun menjadi PENANDA, bukan penghitung per
-- tahun: LPK.26.8125 diikuti LPK.27.8126. Dipertahankan sengaja supaya keduanya tidak
-- berbeda aturan tanpa alasan.
--
-- NOCACHE dipakai supaya nomor tidak melompat jauh saat instans direstart. Pada dua
-- instans di belakang load balancer (D-27), cache yang besar membuat deret nomor
-- terbelah — laporan yang dicatat berurutan tampak bernomor acak bagi petugas.
-- ---------------------------------------------------------------------------

CREATE SEQUENCE POOLDATA.CPNC_LAPORAN_KLAIM_SEQ
    START WITH 1
    INCREMENT BY 1
    NOCACHE
    NOCYCLE;


-- ---------------------------------------------------------------------------
-- Langkah 3 — indeks pendukung.
--
-- Ketiganya menjawab kueri yang benar-benar ada, bukan kueri yang dibayangkan:
--
--   IX_..._URUT     ORDER BY DIINPUT_PADA DESC, NOMOR DESC pada report_list. Ini
--                   urutan baku setiap kali layar dibuka, dan tanpanya setiap pembukaan
--                   layar menuntut pengurutan seluruh tabel.
--   IX_..._CABANG   penyaring cabang pada report_list dan report_summary.
--   IX_..._KLAIM    penautan balik dari klaim ke laporannya (LinkClaim), dan
--                   penyaring tahap yang membedakan sudah/belum registrasi.
--
-- Indeks untuk pencarian teks SENGAJA tidak dibuat. Pencariannya memakai
-- `UPPER(kolom) LIKE '%...%'` dengan wildcard di depan, dan indeks B-tree biasa TIDAK
-- dapat dipakai untuk pola seperti itu — membuatnya hanya menambah biaya tulis tanpa
-- mempercepat satu kueri pun. Bila pencarian kelak menjadi lambat pada data nyata,
-- jawabannya adalah pencarian teks penuh (17-FUTURE-ENHANCEMENT §2.4), bukan indeks ini.
-- ---------------------------------------------------------------------------

CREATE INDEX POOLDATA.IX_CPNC_LAPORAN_KLAIM_URUT
    ON POOLDATA.CPNC_LAPORAN_KLAIM (DIINPUT_PADA DESC, NOMOR DESC);

CREATE INDEX POOLDATA.IX_CPNC_LAPORAN_KLAIM_CABANG
    ON POOLDATA.CPNC_LAPORAN_KLAIM (KODE_CABANG);

CREATE INDEX POOLDATA.IX_CPNC_LAPORAN_KLAIM_KLAIM
    ON POOLDATA.CPNC_LAPORAN_KLAIM (NOMOR_KLAIM);


-- ---------------------------------------------------------------------------
-- Langkah 4 — hak akses untuk akun aplikasi.
--
-- Diberikan sesempit mungkin: SELECT, INSERT, dan UPDATE. TANPA DELETE — tidak ada satu
-- pun jalur di aplikasi yang menghapus laporan, dan hak yang tidak diberikan tidak dapat
-- disalahgunakan kode yang ditulis kemudian.
--
-- Ganti <AKUN_APLIKASI> dengan nama akun yang sebenarnya.
-- ---------------------------------------------------------------------------

-- GRANT SELECT, INSERT, UPDATE ON POOLDATA.CPNC_LAPORAN_KLAIM TO <AKUN_APLIKASI>;
-- GRANT SELECT ON POOLDATA.CPNC_LAPORAN_KLAIM_SEQ TO <AKUN_APLIKASI>;
