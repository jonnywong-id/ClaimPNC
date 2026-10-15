-- 0017 — POOLDATA.T_CLAIM_ADJUSTMENT: penanda persetujuan atasan (Oracle 19c)
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- Tombol SIMPAN pada Payment Klaim Akseptasi (Inbox Manager) di Pega —
-- `Activity/SaveApprovalAkseptasiPaymentLeader_Act` — selain menulis
-- T_CLAIM_AKSEPTASI_CHECKER juga menandai baris adjustment yang diputuskan di dalam BLOB case
-- (`ClaimData.ObjectList(…).ObjectCoverageList(…).AdjustmentList(…)`):
--
--     .FlagAtasan        ""  bila disetujui (status 1), "1" bila tidak (status 0 dan 2)
--                        — `@if(Param.stsapprove==1,"","1")`, baris :1639-1641
--     .StsPenerimaKlaim  KODE status keputusan apa adanya (0/1/2) — baris :1687-1689
--
-- Catatan atasan (`.RemarkManager`, :1704-1706) TIDAK butuh kolom: nilai yang sama sudah
-- ditulis activity itu ke T_CLAIM_AKSEPTASI_CHECKER.NOTEAPPROVE.
--
-- Keduanya tidak punya kolom di tabel mana pun. Aplikasi baru tidak menulis case Pega (`P-1`),
-- sehingga Work Owner memutuskan (2026-10-14) keduanya menjadi kolom baru tabel ini — tingkat
-- yang sama dengan `AdjustmentList`, dan preseden yang sama dengan migrasi 0013.
--
-- BUKAN T_CLAIMLIST_ADMIN: tabel itu satu baris per KLAIM, sedangkan penanda ini per
-- ADJUSTMENT — satu klaim dapat punya banyak nomor akseptasi yang diputuskan sendiri-sendiri.
--
-- `.TransferCashierStatus` yang ikut dikosongkan activity itu TIDAK ditambahkan: modul
-- Registrasi sudah membaca "sudah ditransfer ke kasir" dari TRANSFER_CASHIER_DATE dan IDCHASIER.
--
-- Pembacanya di Pega: `TransferToKasir_act` (jalur leader), `SetValidationReceiver`,
-- `SetTypePDFAdjustment`, `Section/AcceptationLOD_Sect`, `Section/InputAdjustment_sect`, dan
-- inbox XOL. Selama masa paralel Pega TIDAK membaca kolom ini — ia membaca BLOB-nya sendiri.
--
-- Dijalankan DBA. Menempuh permintaan tertulis, persetujuan Work Owner, lalu pengujian
-- dengan MENJALANKAN PEGA DAN GO BERSAMAAN (D-63). Akun aplikasi tidak memiliki hak DDL.
-- Seluruh kolom NULLABLE tanpa default, sehingga Pega yang tidak mengenalnya tidak
-- terpengaruh (P-4).
--
-- Sebelum menjalankan, pastikan tidak ada kolom bernama sama:
--     SELECT column_name FROM all_tab_columns
--      WHERE owner = 'POOLDATA' AND table_name = 'T_CLAIM_ADJUSTMENT'
--        AND column_name IN ('FLAGATASAN','STSPENERIMAKLAIM');
-- Hasilnya harus kosong; bila ada, hapus baris kolom itu dari pernyataan di bawah.

ALTER TABLE POOLDATA.T_CLAIM_ADJUSTMENT ADD (
    FLAGATASAN        VARCHAR2(10),     -- .FlagAtasan        "" / "1"
    STSPENERIMAKLAIM  VARCHAR2(10)      -- .StsPenerimaKlaim  kode status 0/1/2
);
