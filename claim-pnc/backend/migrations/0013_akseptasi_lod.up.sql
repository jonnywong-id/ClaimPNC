-- 0013 — POOLDATA.T_CLAIM_ADJUSTMENT: isian form Persetujuan / Akseptasi (Oracle 19c)
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- Form `Section/AcceptationLOD_Sect.xml` memuat tujuh isian yang di Pega hanya hidup di
-- BLOB case (`AdjustmentList`), tanpa kolom di T_CLAIM_ADJUSTMENT. Klaim PNCN tidak punya
-- BLOB, sehingga Work Owner memutuskan (2026-09-29, keputusan-implementasi §116) ketujuhnya
-- menjadi kolom baru tabel ini.
--
-- Isian yang SUDAH punya kolom tidak ditambahkan:
--     Tanggal Cetak            .PrintDateLOD          -> PRINTLOD_DATE
--     Tanggal Terima LOD       .ReceiveDateLOD        -> RECEIVEDATELOD
--     Persetujuan Tertanggung  .AcceptationStatusLOD  -> STATUSAKSEPTASILOD
--     Penerima Klaim           .Receiver              -> RECEIVER / RECEIVERNAME
--     Nomor / tanggal akseptasi .AcceptedNo/.AcceptedDate -> NOAKSEPTASI / TGLAKSEPTASI
--
-- Panjang teks mengikuti kolom teks bebas yang sudah ada di tabel ini (NOTES,
-- ALASANTOLAK, CIRCUMCAUSEOFLOSS: VARCHAR2(4000); dibaca dari ALL_TAB_COLUMNS
-- 2026-09-29). Nilai uang NUMBER tanpa presisi, sama dengan GROSSVALUE.
--
-- Dijalankan DBA. Menempuh permintaan tertulis, persetujuan Work Owner, lalu pengujian
-- dengan MENJALANKAN PEGA DAN GO BERSAMAAN (D-63). Akun aplikasi tidak memiliki hak DDL.
-- Seluruh kolom NULLABLE tanpa default, sehingga Pega yang tidak mengenalnya tidak
-- terpengaruh (P-4).
--
-- Sebelum menjalankan, pastikan tidak ada kolom bernama sama:
--     SELECT column_name FROM all_tab_columns
--      WHERE owner = 'POOLDATA' AND table_name = 'T_CLAIM_ADJUSTMENT'
--        AND column_name IN ('TANGGALBOLEHBAYAR','RECEIVEDATEANALIST','ACCEPTANCEVALUELOD',
--                            'TIPEAKSEPTASI','KOMITEACCEPTED','REMARKACCEPTED','UPLOADNOTELOD');
-- Hasilnya harus kosong; bila ada, hapus baris kolom itu dari pernyataan di bawah.

ALTER TABLE POOLDATA.T_CLAIM_ADJUSTMENT ADD (
    TANGGALBOLEHBAYAR   DATE,             -- .TanggalBolehBayar   "Tanggal Boleh Bayar"
    RECEIVEDATEANALIST  DATE,             -- .ReceiveDateAnalist  "Tanggal Terima LOD" (PA)
    ACCEPTANCEVALUELOD  NUMBER,           -- .AcceptanceValueLOD  "Nilai LOD"
    TIPEAKSEPTASI       VARCHAR2(50),     -- .TipeAkseptasi       "Tipe Akseptasi Klaim"
    KOMITEACCEPTED      VARCHAR2(300),    -- .KomiteAccepted      "Nama Komite Akseptasi"
    REMARKACCEPTED      VARCHAR2(4000),   -- .RemarkAccepted      "Penerima Klaim" (catatan)
    UPLOADNOTELOD       VARCHAR2(4000)    -- .UploadNoteLOD       "Berita Acara"
);
