-- 0013 — Baris klaim Master Recovery pindah dari kolom JSON ke tabel
--
-- ============================================================================
-- KENAPA BERKAS INI ADA
-- ============================================================================
--
-- Keputusan Work Owner 2026-09-29: **tidak lagi lewat JSON, langsung ke basis data.**
--
-- Sistem lama menyimpan daftar polis sebuah batch recovery sebagai satu dokumen JSON di
-- kolom `POOLDATA.MST_RECOVERY_ASM_PENJAMINAN.JSON_POLIS`. Dokumen itu dibentuk
-- `@GCNM.GetPageJSONString()` — serialisasi mentah halaman klipboard Pega, lengkap dengan
-- properti internalnya.
--
-- Itu bentuk penyimpanan yang `R-10` catat sebagai utang: data yang sama hidup dua kali,
-- satu sebagai kolom dan satu sebagai dokumen, tanpa satu pun mekanisme yang memeriksa
-- keduanya masih sejalan. Berkas ini memindahkan yang kedua menjadi tabel sungguhan.
--
-- ============================================================================
-- KENAPA AMAN DIJALANKAN KAPAN SAJA
-- ============================================================================
--
-- 1. Tabelnya BARU. Tidak ada satu baris pun yang ditulis ulang, tidak ada kolom yang
--    berubah tipe, dan tidak ada yang dihapus.
--
-- 2. Kolom `JSON_POLIS` yang lama TIDAK disentuh — tidak diubah, tidak dikosongkan,
--    tidak dihapus. Baris lama tetap membawa dokumennya sebagai jejak, dan baris yang
--    ditulis Pega selama masa paralel tetap dapat mengisinya.
--
-- 3. Tidak ada rule Pega yang MEMBACA tabel recovery sama sekali — diperiksa ke seluruh
--    export. Satu-satunya yang menyentuhnya adalah penerbit nomor batch
--    (`GetMasterRecoveryClaimSPK`, isinya `select nvl(max(BATCH),0)+1`). Karena itu
--    berhenti mengisi `JSON_POLIS` tidak memutus apa pun di sisi Pega.
--
-- 4. Idempoten: dijalankan berkali-kali tidak menghasilkan galat. Ia dijalankan sekali
--    per basis data portal (`ADR-0030`).
--
-- ============================================================================
-- BENTUKNYA
-- ============================================================================
--
-- BATCH       menunjuk baris induk di MST_RECOVERY_ASM_PENJAMINAN. Tipe dan presisinya
--             disamakan dengan induknya — NUMBER tanpa presisi, sesuai katalog.
-- URUTAN      urutan baris DI DALAM berkas yang diunggah. Ia ada karena daftar polis
--             adalah DERET: petugas membandingkan hasil unggahan dengan berkas aslinya
--             baris per baris, dan tanpa urutan perbandingan itu tidak punya dasar.
-- NOPOLIS     VARCHAR2(200), sama dengan kolom NOPOLIS pada induknya.
-- NILAIKLAIM  NUMBER, NULLABLE — berkas contoh Pega hanya memuat kolom nomor polis,
--             sehingga nilai klaim boleh belum terisi saat diunggah.
--
-- Kunci utama (BATCH, URUTAN): satu batch tidak dapat memuat dua baris pada urutan yang
-- sama, dan penyisipan ulang yang tidak disengaja ditolak alih-alih menggandakan.
--
-- FOREIGN KEY ke induknya sengaja DIPASANG: baris klaim tanpa batch tidak berarti apa
-- pun, dan membiarkannya yatim hanya memindahkan kekacauan ke pembaca berikutnya.

DECLARE
    sudah_ada NUMBER;
BEGIN
    SELECT COUNT(*) INTO sudah_ada
      FROM ALL_TABLES
     WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'CPNC_RECOVERY_BARIS_KLAIM';

    IF sudah_ada = 0 THEN
        EXECUTE IMMEDIATE '
            CREATE TABLE POOLDATA.CPNC_RECOVERY_BARIS_KLAIM (
                BATCH       NUMBER          NOT NULL,
                URUTAN      NUMBER(10)      NOT NULL,
                NOPOLIS     VARCHAR2(200)   NOT NULL,
                NILAIKLAIM  NUMBER,
                CONSTRAINT CPNC_RECOVERY_BARIS_KLAIM_PK
                    PRIMARY KEY (BATCH, URUTAN),
                CONSTRAINT CPNC_RECOVERY_BARIS_KLAIM_FK
                    FOREIGN KEY (BATCH)
                    REFERENCES POOLDATA.MST_RECOVERY_ASM_PENJAMINAN (BATCH)
            )';
    END IF;
END;
/

-- Indeks pencarian menurut nomor polis.
--
-- Dibuat TERPISAH dari tabelnya supaya migrasi ini tetap idempoten: tabel yang sudah ada
-- dari jalannya berkas ini sebelumnya tetap memperoleh indeksnya.
--
-- Gunanya nyata, bukan berjaga-jaga: pertanyaan "batch mana saja yang mencakup polis
-- ini" adalah satu-satunya cara menelusuri sebuah polis kembali ke pemulihan dananya,
-- dan tanpa indeks ia memindai seluruh tabel.
DECLARE
    sudah_ada NUMBER;
BEGIN
    SELECT COUNT(*) INTO sudah_ada
      FROM ALL_INDEXES
     WHERE OWNER = 'POOLDATA' AND INDEX_NAME = 'IX_CPNC_RECOVERY_BARIS_POLIS';

    IF sudah_ada = 0 THEN
        EXECUTE IMMEDIATE '
            CREATE INDEX POOLDATA.IX_CPNC_RECOVERY_BARIS_POLIS
                ON POOLDATA.CPNC_RECOVERY_BARIS_KLAIM (NOPOLIS)';
    END IF;
END;
/
