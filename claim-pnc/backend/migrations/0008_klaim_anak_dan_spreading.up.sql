-- 0008 — Kolom tambahan pada dua tabel anak klaim, dan tabel T_CLAIM_SPREADING
--
-- ============================================================================
-- KENAPA BERKAS INI ADA
-- ============================================================================
--
-- Lanjutan `0007`. Setelah kepala klaim punya rumah di POOLDATA.T_CLAIM_PNC, pohon di
-- bawahnya menyusul: objek, coverage, dan spreading.
--
-- Work Owner menetapkan 2026-09-24 bahwa spreading memakai tabel BARU bernama
-- T_CLAIM_SPREADING — bukan CPNC_KLAIM_SPREADING, dan bukan pula T_SPREADINGLIST yang
-- sudah ada di skema ini untuk keperluan lain.
--
-- ============================================================================
-- TIGA KOLOM YANG HILANG DARI TABEL ANAK, DAN KENAPA MEREKA PENTING
-- ============================================================================
--
-- URUTAN / URUTAN_OBJEK
--   Modul menyimpan objek dan coverage sebagai DERET, dan deret itulah yang dipakai
--   `objek_tandai_sisa` dan `coverage_tandai_sisa` untuk menandai baris yang petugas
--   buang. Tanpa urutan, keduanya tidak punya dasar — dan keduanya satu-satunya cara
--   modul ini "menghapus" tanpa DELETE (`ADR-0012`).
--
-- LOKASI
--   Kolom LOCATIONID yang sudah ada TIDAK dapat dipakai: isinya KODE, bukan teks —
--   terverifikasi 2026-09-24, seluruh 1.425 baris terisi berupa angka. Lokasi kejadian
--   pada domain adalah kalimat yang diketik petugas. Memaksakannya ke kolom kode adalah
--   utang teknis 4.2 yang proyek ini ada untuk menghapusnya.
--
-- DIHAPUS_PADA
--   Soft delete (`ADR-0012`). Tabel warisan tidak punya penandanya sama sekali.
--
-- ============================================================================
-- KENAPA AMAN
-- ============================================================================
--
-- Seluruh kolom NULLABLE tanpa DEFAULT — metadata saja, tidak ada baris yang ditulis
-- ulang. T_CLAIM_SPREADING adalah tabel BARU, sehingga tidak menyentuh apa pun yang
-- sedang dibaca Pega.
--
-- Idempoten, karena berkas ini dijalankan sekali per basis data portal (`ADR-0030`).

DECLARE
    TYPE t_kolom IS RECORD (tabel VARCHAR2(30), nama VARCHAR2(30), tipe VARCHAR2(40));
    TYPE t_daftar IS TABLE OF t_kolom;

    daftar t_daftar := t_daftar(
        t_kolom('T_CLAIM_OBJECTLIST',     'URUTAN',        'NUMBER(10)'),
        t_kolom('T_CLAIM_OBJECTLIST',     'LOKASI',        'VARCHAR2(1000)'),
        t_kolom('T_CLAIM_OBJECTLIST',     'DIHAPUS_PADA',  'TIMESTAMP'),
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'URUTAN_OBJEK',  'NUMBER(10)'),
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'URUTAN',        'NUMBER(10)'),
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'DIHAPUS_PADA',  'TIMESTAMP')
    );

    ada NUMBER;
BEGIN
    FOR i IN 1 .. daftar.COUNT LOOP
        SELECT COUNT(*) INTO ada
          FROM all_tab_cols
         WHERE owner = 'POOLDATA'
           AND table_name = daftar(i).tabel
           AND column_name = daftar(i).nama;

        IF ada = 0 THEN
            EXECUTE IMMEDIATE
                'ALTER TABLE POOLDATA.' || daftar(i).tabel || ' ADD (' ||
                daftar(i).nama || ' ' || daftar(i).tipe || ')';
        END IF;
    END LOOP;
END;
/

-- ============================================================================
-- T_CLAIM_SPREADING
-- ============================================================================
--
-- Pembagian risiko satu coverage ke para penanggung. Total share WAJIB 100% dengan
-- toleransi empat desimal, `99,9999`–`100,0001` (`D-51`).
--
-- SHARE_E4 menyimpan share dikalikan 10.000 sebagai BILANGAN BULAT. Alasannya sama
-- dengan `ADR-0016` menyimpan uang sebagai sen: pembulatan pecahan biner membuat aturan
-- 100% gagal secara acak dan tidak dapat direproduksi — cacat yang `I-1` ada untuk
-- mencegahnya. 100% tersimpan sebagai 1000000.
--
-- Kuncinya mengikuti tabel anak yang lain: CLAIMID milik klaimnya, lalu posisi di dalam
-- pohon objek–coverage–spreading.
DECLARE
    ada NUMBER;
BEGIN
    SELECT COUNT(*) INTO ada
      FROM all_tables WHERE owner = 'POOLDATA' AND table_name = 'T_CLAIM_SPREADING';

    IF ada = 0 THEN
        EXECUTE IMMEDIATE '
            CREATE TABLE POOLDATA.T_CLAIM_SPREADING (
                CLAIMID          VARCHAR2(100)  NOT NULL,
                URUTAN_OBJEK     NUMBER(10)     NOT NULL,
                URUTAN_COVERAGE  NUMBER(10)     NOT NULL,
                URUTAN           NUMBER(10)     NOT NULL,
                JENIS_TREATY     VARCHAR2(30),
                NAMA             VARCHAR2(200),
                SHARE_E4         NUMBER(12),
                DIHAPUS          VARCHAR2(5),
                OBJEK_FAC_OFFER  VARCHAR2(500),
                DIHAPUS_PADA     TIMESTAMP,
                CONSTRAINT PK_T_CLAIM_SPREADING
                    PRIMARY KEY (CLAIMID, URUTAN_OBJEK, URUTAN_COVERAGE, URUTAN)
            )';

        EXECUTE IMMEDIATE
            'CREATE INDEX POOLDATA.IX_T_CLAIM_SPREADING_KLAIM
                 ON POOLDATA.T_CLAIM_SPREADING (CLAIMID)';
    END IF;
END;
/

COMMENT ON TABLE POOLDATA.T_CLAIM_SPREADING IS 'Pembagian risiko per coverage; ditulis modul Registrasi Klaim (B-2)';
COMMENT ON COLUMN POOLDATA.T_CLAIM_SPREADING.SHARE_E4 IS 'Share x 10000 sebagai bilangan bulat; 100% = 1000000 (D-51)';
COMMENT ON COLUMN POOLDATA.T_CLAIM_SPREADING.DIHAPUS_PADA IS 'Penanda soft delete (ADR-0012); NULL berarti baris masih berlaku';
COMMENT ON COLUMN POOLDATA.T_CLAIM_OBJECTLIST.LOKASI IS 'Lokasi kejadian sebagai TEKS; LOCATIONID yang sudah ada berisi kode';
