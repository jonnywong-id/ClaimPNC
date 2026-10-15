-- ============================================================================
-- DICABUT 2026-10-07 — JANGAN DIJALANKAN
-- ============================================================================
--
-- Tabel ini TIDAK JADI DIPAKAI. Work Owner memutuskan Transfer mengikuti Pega apa
-- adanya: PIC Teknik dipindahkan LANGSUNG, tanpa antrean permintaan.
--
-- Dua kenyataan yang mendasarinya:
--
--   1. Antreannya tidak punya pelaksana. Tidak ada satu pun job Pega yang membacanya;
--      kelima job terjadwal (D-57) seluruhnya lebih tua daripada tabel ini. Permintaan
--      akan menumpuk berstatus menunggu, dan pengguna menunggu sesuatu yang tidak
--      akan datang.
--
--   2. P-1 tidak berlaku di jalur ini. Assign per baris di Pega TIDAK memanggil
--      pxTransferAssignment — PNC_ReassignPNCTeknik hanya mengubah ClaimData.UserTeknis.
--      Antrean tugas PC_ASSIGN_WORKLIST tidak disentuh sama sekali.
--
-- Rinciannya di docs/keputusan-implementasi.md §204.
--
-- Yang dibutuhkan sebagai GANTINYA, dan hanya ini:
--
--     GRANT UPDATE (USERTEKNIS_1) ON DATAPEGA.PC_ASM_FW_GCNMFW_WORK TO <akun aplikasi>;
--
-- Berkas ini TIDAK dihapus karena ia rekaman: ia memuat alasan rancangan antrean beserta
-- bentuk tabelnya, dan itu yang dibutuhkan bila kelak jejak audit dituntut kembali.
-- Yang dikembalikan nanti bukan antrean melainkan tabel LOG yang ditulis SESUDAH
-- pemindahan berhasil — bentuknya mirip, artinya berbeda.
--
-- ============================================================================

-- 0014 naik — tabel permintaan TRANSFER penugasan klaim.
--
-- ============================================================================
-- KENAPA PERMINTAAN, BUKAN PEMINDAHAN
-- ============================================================================
--
-- Layar lama menggambar tombol "Transfer" pada setiap baris Inbox Outstanding dan Inbox
-- Tampungan PIC, ditambah "Transfer All Case By UserID" yang memindahkan seluruh pekerjaan
-- satu operator sekaligus. Keduanya MENULIS `DATAPEGA.PC_ASSIGN_WORKLIST`.
--
-- `P-1` menetapkan satu tabel hanya ditulis satu sistem, dan selama masa paralel tabel itu
-- milik Pega. Dua sistem yang sama-sama memindahkan penugasan akan menghasilkan tugas yang
-- hilang atau terpegang dua orang — dan konflik seperti itu nyaris mustahil dilacak.
--
-- Karena itu aplikasi ini mencatat PERMINTAANNYA, dan Pega yang menjalankan. Pola yang sama
-- sudah dipakai migrasi `0006` untuk ReOpen dan Copy Klaim.
--
-- ============================================================================
-- KENAPA TABEL SENDIRI, BUKAN JENIS BARU DI CPNC_PERMINTAAN_KLAIM
-- ============================================================================
--
-- Menambahkan `JENIS = 'transfer'` ke tabel `0006` sempat dipertimbangkan dan TIDAK diambil.
-- Alasannya bukan kerapian melainkan bentuk datanya: permintaan transfer membawa dua hal
-- yang tidak dimiliki ReOpen maupun Copy — operator ASAL dan operator TUJUAN — sementara
-- tabel `0006` hanya punya `ALASAN` sebagai teks bebas.
--
-- Menampungnya di `ALASAN` berarti menyimpan data terstruktur sebagai kalimat, dan
-- membacanya kembali berarti menguraikan kalimat. Yang pertama kali salah ketik tidak
-- menghasilkan galat apa pun — hanya permintaan yang dijalankan ke orang yang keliru.
--
-- ============================================================================
-- BELUM DIJALANKAN DBA
-- ============================================================================
--
-- Sama seperti `0006`, migrasi ini menempuh `D-63`: permintaan tertulis tim pengembang,
-- persetujuan Work Owner, pelaksanaan DBA. Sampai itu terjadi, tombol Transfer menjawab
-- galat yang menyebutkan sebabnya — bukan diam-diam gagal.

CREATE TABLE POOLDATA.CPNC_PERMINTAAN_TRANSFER (
    -- Pengenal acak 128 bit dalam heksadesimal, dibangkitkan aplikasi.
    --
    -- Acak, bukan berurut: pengenal permintaan tidak boleh membocorkan berapa banyak
    -- permintaan yang sudah tercatat.
    ID                  VARCHAR2(32)   NOT NULL,

    -- 'baris' atau 'massal'.
    --
    --   baris    satu klaim, dari tombol Transfer pada barisnya
    --   massal   seluruh pekerjaan satu operator, dari "Transfer All Case By UserID"
    --
    -- Keduanya dicatat di tabel yang SAMA karena akibatnya sama — penugasan berpindah — dan
    -- yang membedakan hanya cakupannya. Memisahkannya menjadi dua tabel akan membuat
    -- penelusuran "siapa memindahkan apa" harus membaca dua tempat.
    LINGKUP             VARCHAR2(20)   NOT NULL,

    -- `PZINSKEY` klaim yang dipindahkan. KOSONG pada permintaan massal.
    --
    -- TIDAK ada foreign key ke tabel warisan, dan ketiadaannya disengaja: constraint dari
    -- tabel milik kita ke tabel milik Pega akan membuat pengarsipan di sisi Pega GAGAL
    -- karena baris kita.
    CASE_ID             VARCHAR2(64),

    -- Nomor klaim, disimpan SEBAGAI SALINAN supaya jejak tetap terbaca utuh bila klaim
    -- warisannya kelak diarsipkan. KOSONG pada permintaan massal.
    NOMOR_KLAIM         VARCHAR2(50),

    -- Operator ASAL — "User ID Lama" pada layar lama. Hanya terisi pada permintaan massal.
    OPERATOR_ASAL       VARCHAR2(64),

    -- Operator TUJUAN — "User ID Baru". WAJIB pada kedua lingkup.
    --
    -- Permintaan tanpa tujuan tidak punya arti, dan menolaknya di sini membuat penolakannya
    -- terjadi sebelum baris tercatat — bukan saat pelaksana menemukannya.
    OPERATOR_TUJUAN     VARCHAR2(64)   NOT NULL,

    -- "Type User" pada layar lama — peran yang dituju. BOLEH kosong.
    --
    -- Isian itu dropdown di layar lama dan daftar pilihannya TIDAK ADA di export (`R-16`),
    -- sehingga nilainya diteruskan apa adanya tanpa divalidasi. Dicatat terbuka: yang
    -- memeriksa keabsahannya adalah pelaksana, bukan aplikasi ini.
    TIPE_PENGGUNA       VARCHAR2(50),

    -- Alasan yang diketik pengguna. BOLEH kosong — layar lama tidak memintanya.
    ALASAN              VARCHAR2(1500),

    -- 'menunggu', 'dijalankan', atau 'dibatalkan'.
    --
    -- Aplikasi hanya pernah menulis 'menunggu'. Dua nilai lainnya ditulis pelaksana.
    STATUS              VARCHAR2(20)   DEFAULT 'menunggu' NOT NULL,

    -- Siapa yang meminta, dan kapan.
    --
    -- `D-59` menghapus pemisahan tugas, sehingga jejak semacam ini adalah satu-satunya
    -- kontrol pengimbang yang tersisa. NAMA disimpan sebagai salinan di samping login:
    -- jejak yang namanya diambil lewat join akan berubah ketika orangnya berganti nama, dan
    -- jejak yang dapat berubah bukan jejak.
    PEMOHON             VARCHAR2(64)   NOT NULL,
    PEMOHON_NAMA        VARCHAR2(128),
    DIBUAT_PADA         TIMESTAMP      DEFAULT SYSTIMESTAMP NOT NULL,

    CONSTRAINT PK_CPNC_PERMINTAAN_TRANSFER PRIMARY KEY (ID),

    -- Lingkup menentukan field mana yang wajib, dan aturannya ditegakkan BASIS DATA — bukan
    -- hanya kode. Aturan yang hanya ada di kode dapat dilanggar oleh kode berikutnya.
    CONSTRAINT CK_CPNC_TRANSFER_LINGKUP CHECK (LINGKUP IN ('baris', 'massal')),
    CONSTRAINT CK_CPNC_TRANSFER_ISI CHECK (
        (LINGKUP = 'baris'  AND CASE_ID IS NOT NULL)
     OR (LINGKUP = 'massal' AND OPERATOR_ASAL IS NOT NULL)
    ),
    CONSTRAINT CK_CPNC_TRANSFER_STATUS CHECK (STATUS IN ('menunggu', 'dijalankan', 'dibatalkan'))
);

-- Menemukan permintaan yang masih menunggu atas satu klaim.
--
-- Dipakai layar untuk menandai baris yang sudah diajukan — tanpa penanda itu, pengguna yang
-- sudah menekan Transfer tidak melihat perubahan apa pun (klaimnya memang belum berpindah)
-- lalu menekannya lagi.
CREATE INDEX IX_CPNC_TRANSFER_CASE ON POOLDATA.CPNC_PERMINTAAN_TRANSFER (CASE_ID, STATUS);

-- Menemukan permintaan massal yang masih menunggu atas satu operator asal.
CREATE INDEX IX_CPNC_TRANSFER_ASAL ON POOLDATA.CPNC_PERMINTAAN_TRANSFER (OPERATOR_ASAL, STATUS);

-- Antrean pelaksana: yang menunggu, terlama lebih dulu.
CREATE INDEX IX_CPNC_TRANSFER_ANTREAN ON POOLDATA.CPNC_PERMINTAAN_TRANSFER (STATUS, DIBUAT_PADA);
