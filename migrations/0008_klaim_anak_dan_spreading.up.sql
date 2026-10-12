-- 0008 — Kolom tambahan pada dua tabel anak klaim
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
-- ulang.
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
-- T_CLAIM_SPREADING — SENGAJA TIDAK DIBUAT DI SINI
-- ============================================================================
--
-- Versi pertama berkas ini membuat T_CLAIM_SPREADING dengan bentuk rancangan sendiri
-- (URUTAN_OBJEK, URUTAN_COVERAGE, SHARE_E4, ...). Di ASM tabel itu ternyata SUDAH ADA
-- dengan bentuk lain, sehingga pagar IF ada = 0 melewatinya tanpa suara — dan modul
-- menulis kolom yang tidak ada (ORA-00904), terbukti 2026-09-26.
--
-- Bentuk tabelnya DITETAPKAN Work Owner di Database/CREATE_TABLE_2.sql, dan dijalankan
-- lewat DBA sesuai D-63. Berkas ini tidak lagi menyentuhnya, supaya:
--
--   1. tidak ada dua versi DDL untuk satu tabel yang dapat berbeda (D-72);
--   2. tiga portal yang belum dimigrasi tidak menerima bentuk yang keliru;
--   3. pembatalan 0008 tidak menghapus tabel yang bukan miliknya.
--
-- Keberadaannya diperiksa mode -periksa, bukan diandaikan.

COMMENT ON COLUMN POOLDATA.T_CLAIM_OBJECTLIST.LOKASI IS 'Lokasi kejadian sebagai TEKS; LOCATIONID yang sudah ada berisi kode';
