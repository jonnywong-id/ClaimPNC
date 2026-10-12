-- 0014 — POOLDATA.T_CLAIM_ADJUSTMENT: penanda hapus baris adjustment (Oracle 19c)
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- Grid Adjustment Pega (`ShowAdjustment` / `InputAdjustment`) punya tombol Hapus pada baris yang
-- belum diakseptasi. Work Owner memutuskan (2026-10-03) form adjustment mengikuti Pega persis:
-- tanpa Simpan/Batal, setiap perubahan isian langsung disimpan, dan baris dihapus lewat Hapus.
--
-- Aplikasi ini tidak menghapus baris secara fisik (ADR-0012, D-66): penghapusan dinyatakan lewat
-- kolom DIHAPUS_PADA, seperti T_CLAIM_OBJECTLIST dan T_CLAIM_OBJECTCOVERAGE. T_CLAIM_ADJUSTMENT
-- belum punya kolom itu (dibaca dari ALL_TAB_COLUMNS 2026-10-03), sehingga Hapus pada baris yang
-- sudah tersimpan belum dapat dijalankan sampai berkas ini dijalankan.
--
-- Dijalankan DBA. Menempuh permintaan tertulis, persetujuan Work Owner, lalu pengujian dengan
-- MENJALANKAN PEGA DAN GO BERSAMAAN (D-63). Akun aplikasi tidak memiliki hak DDL. Kolom NULLABLE
-- tanpa default, sehingga Pega yang tidak mengenalnya tidak terpengaruh (P-4).
--
-- Sebelum menjalankan, pastikan kolomnya belum ada:
--     SELECT column_name FROM all_tab_columns
--      WHERE owner = 'POOLDATA' AND table_name = 'T_CLAIM_ADJUSTMENT' AND column_name = 'DIHAPUS_PADA';
-- Hasilnya harus kosong.

ALTER TABLE POOLDATA.T_CLAIM_ADJUSTMENT ADD (
    DIHAPUS_PADA  TIMESTAMP(6)   -- waktu baris adjustment dihapus lewat tombol Hapus; NULL = aktif
);
