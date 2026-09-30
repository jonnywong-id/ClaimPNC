-- ============================================================================
-- TC_PNC_KOMITE — kepala kasus komite adjustment klaim PNCN
-- ============================================================================
--
-- STATUS: RANCANGAN (2026-09-29). Belum dijalankan. Menempuh D-63 (permintaan tertulis
-- -> persetujuan Work Owner -> DBA -> uji Pega+Go bersamaan). Portal ASM lebih dulu.
--
-- KENAPA TABEL BARU
--   Pega menyimpan kasus komite sebagai case Work-Komite (DATAPEGA.PC_ASM_FW_GCNMFW_WORK):
--   KomiteLoop, KomiteCount, Type, TransferType, AcceptStatus, pyStatusWork, dan adjustment
--   mana yang diputus. Aplikasi ini tidak membentuk case Pega, jadi data itu BELUM PUNYA
--   TABEL — syarat TC_PNC_* (Work Owner, 2026-09-26).
--   Anggota komite per jenjang TETAP di POOLDATA.T_CLAIM_KOMITE_LIST (sudah ada, lengkap):
--   satu baris per anggota, KOMITE_ID sama dengan kolom KOMITE_ID di sini.
--
-- SUMBER
--   SetListComiteeClaimPerObjAdj step 56-58 (KomiteLoop, KomiteCount, Type,
--   IsKomiteTransfer), SetChildKomitePerAdjustment_act (TransferType "2", Adjustment),
--   SetEmailKomite (tempAdj.pyMemo = TYPE_BUSINESS, tempAdj.AcceptedNo = TYPE_KOMITE,
--   tempAdj.ConvertAdjustmentValue), KomitePost_Adjustment (AcceptStatus, KomiteCount,
--   case ditutup Resolved-Completed).
--
-- KUNCI
--   KOMITE_ID "KMTN.26.1" (sejak 2026-09-29; satu baris lama berbentuk KMTN-00001) —
--   VARCHAR2(10) sama dengan T_CLAIM_KOMITE_LIST.KOMITE_ID, sehingga paling banyak
--   KMTN.YY.99 per tahun sampai kedua kolom dilebarkan; awalan KMTN membedakannya dari KMT- Pega. PRIMARY KEY mencegah dua transfer serentak
--   menerbitkan nomor yang sama (yang kalah menerima galat, bukan nomor ganda).
--   Alamat adjustment memakai kunci T_CLAIM_ADJUSTMENT (CLAIMID, OBJECTID,
--   OBJECTCOVERAGEID, ADJUSTMENTID) tanpa FOREIGN KEY — tabel lama tidak punya PK.
--
-- TIPE. Uang NUMBER tanpa skala (D-51, I-12), dalam rupiah/valuta seperti kolom lama.
--   Waktu TIMESTAMP UTC (DB-8). Soft delete lewat DIHAPUS_* (D-66).
-- ============================================================================

CREATE TABLE POOLDATA.TC_PNC_KOMITE (
    KOMITE_ID                   VARCHAR2(10)    NOT NULL,   -- pyID case komite; = T_CLAIM_KOMITE_LIST.KOMITE_ID
    CLAIMID                     VARCHAR2(100)   NOT NULL,   -- klaim (T_CLAIM_PNC.CLAIMID)
    NO_KLAIM                    VARCHAR2(100)   NOT NULL,   -- nomor klaim PNCN.YY.xxxx
    OBJECTID                    VARCHAR2(30)    NOT NULL,   -- T_CLAIM_ADJUSTMENT.OBJECTID
    OBJECTCOVERAGEID            VARCHAR2(30)    NOT NULL,   -- T_CLAIM_ADJUSTMENT.OBJECTCOVERAGEID (urutan coverage)
    ADJUSTMENTID                VARCHAR2(10)    NOT NULL,   -- T_CLAIM_ADJUSTMENT.ADJUSTMENTID (urutan adjustment)
    PAYMENTTYPE                 VARCHAR2(5),                -- Type: 1 Final, 2 Interim, 3 Salvage, 4 Adjuster Fee, 5 Adjustment, 6 Tolak
    TRANSFERTYPE                VARCHAR2(5),                -- TransferType: "2" transfer adjustment
    TYPE_BUSINESS               VARCHAR2(20),               -- lini komite EMAILKOMITE (tempAdj.pyMemo): NONMBU, NONMBUAB, PA, TRAVEL, BONDING
    TYPE_KOMITE                 VARCHAR2(5),                -- pita Non-MBU (tempAdj.AcceptedNo); kosong di lini lain
    CURRENCY                    VARCHAR2(20),               -- mata uang adjustment
    CURRENCYVALUE               NUMBER,                     -- kurs tanggal kejadian (D-48)
    NILAIADJUSTMENT             NUMBER,                     -- AdjustmentValue: bagian ASM, dalam mata uang adjustment
    CONVERTADJUSTMENTVALUE      NUMBER,                     -- nilai pembanding ambang komite, IDR
    KOMITELOOP                  NUMBER(3),                  -- KomiteLoop: jumlah jenjang
    KOMITECOUNT                 NUMBER(3),                  -- KomiteCount: jenjang yang ditunggu / terakhir
    ACCEPTSTATUS                VARCHAR2(5),                -- hasil: kosong berjalan, 1 disetujui, 2 ditolak
    STATUSWORK                  VARCHAR2(30),               -- pyStatusWork: New / Resolved-Completed
    PENGAJU                     VARCHAR2(64),               -- operator yang mentransfer (dikecualikan dari penyetuju)
    DIBUAT_PADA                 TIMESTAMP       NOT NULL,   -- waktu transfer (= T_CLAIM_ADJUSTMENT.ANALYST_TFKOMITEDATE)
    DIBUAT_OLEH                 VARCHAR2(64),
    DIUBAH_PADA                 TIMESTAMP,
    DIUBAH_OLEH                 VARCHAR2(64),
    DIPUTUS_PADA                TIMESTAMP,                  -- putusan akhir (= T_CLAIM_ADJUSTMENT.ACCEPTANCE_DATECOMITEE)
    DIHAPUS_OLEH                VARCHAR2(64),
    DIHAPUS_PADA                TIMESTAMP,                  -- soft delete (D-66)
    CONSTRAINT PK_TC_PNC_KOMITE PRIMARY KEY (KOMITE_ID)
);

CREATE INDEX POOLDATA.IX_TC_PNC_KOMITE_ADJ
    ON POOLDATA.TC_PNC_KOMITE (CLAIMID, OBJECTID, OBJECTCOVERAGEID, ADJUSTMENTID);

COMMENT ON TABLE  POOLDATA.TC_PNC_KOMITE IS 'Kepala kasus komite adjustment klaim PNCN (pengganti case Work-Komite Pega); anggota per jenjang di T_CLAIM_KOMITE_LIST';
COMMENT ON COLUMN POOLDATA.TC_PNC_KOMITE.CONVERTADJUSTMENTVALUE IS 'Nilai pembanding ambang komite dalam IDR (D-47, D-48)';
COMMENT ON COLUMN POOLDATA.TC_PNC_KOMITE.TYPE_KOMITE IS 'Pita nilai Non-MBU (D-52, D-70); kosong di lini lain';

-- Hak akun aplikasi mengikuti tabel TC_PNC_* lain (SELECT, INSERT, UPDATE; tanpa DELETE —
-- penghapusan lewat DIHAPUS_*). Nama akun aplikasi tidak ditulis di sini.


-- ============================================================================
-- ROLLBACK (P-4) — dijalankan DBA HANYA bila perubahan ini harus ditarik
-- ============================================================================
--   DROP INDEX POOLDATA.IX_TC_PNC_KOMITE_ADJ;
--   DROP TABLE POOLDATA.TC_PNC_KOMITE;
--
-- Setelah aplikasi mulai menulis, DROP menghapus kepala kasus komite: baris anggota di
-- T_CLAIM_KOMITE_LIST tetap ada, tetapi alamat adjustment dan hasil kasusnya hilang.
