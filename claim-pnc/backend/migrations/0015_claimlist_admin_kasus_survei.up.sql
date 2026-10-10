-- ============================================================================
-- 0015 — kolom penaut kasus survei pada POOLDATA.T_CLAIMLIST_ADMIN
-- ============================================================================
--
-- SEBABNYA. Kedua tile survei Dashboard Claim — Loss Adjuster dan Internal Surveyor —
-- menghitung KASUS SURVEI, yang di Pega berupa BARIS ANAK pada tabel kerja:
--
--     Work-PNC           baris klaim
--     Work-SurveyClaim   baris kasus survei, ditaut CASEID_1 -> PZINSKEY klaimnya
--
-- Terbukti dari `RDB List/BrowseInternalSurveyor-SQL.xml`, yang membaca
-- `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` dan menaut lewat `a.CASEID_1`.
--
-- Setelah modul ini berhenti membaca skema DATAPEGA, ketiga kolom di bawah tidak ada
-- padanannya di tabel datar, sehingga kedua kueri gagal:
--
--     ORA-00904: "S"."CASEID_1": invalid identifier
--
-- Satu kueri gagal membuat SELURUH ringkasan menjawab 500, sehingga keempat kartu
-- menampilkan 0 — termasuk Outstanding yang sebenarnya sehat (1.658). Itu sebabnya
-- gejalanya terlihat seperti "semua data hilang", padahal yang rusak dua kueri.
--
-- TIPE DAN PANJANGNYA DISALIN DARI SUMBER, bukan ditebak. Dibaca dari ALL_TAB_COLUMNS
-- pada DATAPEGA.PC_ASM_FW_GCNMFW_WORK (2026-10-08):
--
--     CASEID_1              VARCHAR2(128)    terpanjang terpakai  28
--     RESCHEDULEDATE_1      TIMESTAMP(6)     --
--     RESCHEDULELOCATION_1  VARCHAR2(128)    terpanjang terpakai  32
--
-- Panjang aslinya dipertahankan meski pemakaian nyatanya jauh lebih pendek: menyempitkan
-- di sini akan mengulang ORA-12899 yang sudah terjadi pada backfill 2026-10-08, dan
-- VARCHAR2 hanya memakai ruang sepanjang isinya.
--
-- MIGRASI INI AMAN TERHADAP P-4. Ia hanya MENAMBAH kolom nullable: Pega tidak membacanya,
-- versi aplikasi yang lama tidak menyebutnya, dan keduanya tetap berjalan atas skema ini.
--
-- SETELAH migrasi ini, barisnya masih harus diisi — lihat
-- `docs/isi-kasus-survei-t-claimlist-admin.sql` (367 baris di DEV_PEGA83).
-- Tanpa itu kedua tile berhenti galat tetapi menjawab 0.

ALTER TABLE POOLDATA.T_CLAIMLIST_ADMIN ADD (
  CASEID_1             VARCHAR2(128),
  RESCHEDULEDATE_1     TIMESTAMP(6),
  RESCHEDULELOCATION_1 VARCHAR2(128)
);

-- Penaut kasus survei ke klaim induknya. Kedua tile menjalankan sub-kueri
-- `WHERE s.CASEID_1 = A.PZINSKEY` untuk SETIAP baris klaim, sehingga tanpa index ini
-- biayanya tumbuh seiring jumlah klaim — dan tabel ini menuju puluhan juta baris (`D-10`).
--
-- Parsial lewat trik NULL: hanya baris kasus survei yang punya CASEID_1, sehingga baris
-- klaim biasa tidak ikut masuk index. Oracle memang tidak mengindeks baris ber-kunci NULL.
CREATE INDEX POOLDATA.IX_CLAIMLIST_ADMIN_CASEID ON POOLDATA.T_CLAIMLIST_ADMIN (CASEID_1);
