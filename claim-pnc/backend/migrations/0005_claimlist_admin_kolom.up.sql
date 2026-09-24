-- 0005 — T_CLAIMLIST_ADMIN: kolom untuk menggantikan dua tabel Pega (Oracle 19c)
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- Sasaran akhir: POOLDATA.T_CLAIMLIST_ADMIN menggantikan SELURUH pemakaian
--
--     DATAPEGA.PC_ASM_FW_GCNMFW_WORK   186 kolom · 123 terisi · 7.703 baris
--     DATAPEGA.PC_ASSIGN_WORKLIST       56 kolom ·  51 terisi · 259.011 baris
--
-- Keadaan sekarang: T_CLAIMLIST_ADMIN 40 kolom · 1.014 baris (13% dari klaim).
--
-- Seluruh tipe dan panjang di bawah DISALIN DARI KATALOG ORACLE
-- (`ALL_TAB_COLUMNS`) pada 2026-09-22, bukan ditebak. Angka pada komentar tiap
-- kolom adalah `NUM_DISTINCT` — banyaknya nilai berbeda yang benar-benar ada.
--
-- Dijalankan DBA. Menempuh permintaan tertulis, persetujuan Work Owner, lalu
-- pengujian dengan MENJALANKAN PEGA DAN GO BERSAMAAN (D-63). Akun aplikasi
-- tidak memiliki hak DDL.
--
-- ============================================================================
-- YANG TIDAK DITAMBAHKAN, DAN ALASANNYA
-- ============================================================================
--
-- PXREFOBJECTKEY dan PXREFOBJECTINSNAME TIDAK ditambahkan. Keduanya kunci join
-- worklist -> work object, dan karena kedua tabel kini menyatu:
--
--     PXREFOBJECTKEY     -> PZINSKEY
--     PXREFOBJECTINSNAME -> PYID
--
-- Join lamanya INNER, dan dua cacatnya ikut hilang bersama join itu: klaim
-- tanpa assignment LENYAP dari layar, klaim dengan lebih dari satu assignment
-- TAMPIL BERKALI-KALI. Menambahkan kembali kuncinya akan mengembalikan
-- keduanya.
--
-- Kolom internal engine Pega juga tidak dibawa: PXCOVERINSKEY, PXCOVEREDCOUNT,
-- PXCURRENTSTAGELABEL, PXCREATESYSTEMID, PXUPDATESYSTEMID, PXAPPLICATION,
-- PXFLOWINSKEY, PXREFOBJECTCLASS, PXREFQUEUEKEY, PYINTERESTPAGECLASS,
-- PYELAPSED*, PYSTATUSCUSTOMERSAT, dan seluruh PYORIG*/PYOWNER*/PYRESOLVED*
-- yang berupa org, division, atau workgroup.
--
-- ============================================================================
-- ENAM NAMA BERTABRAKAN — DIBERI AKHIRAN _ASSIGN
-- ============================================================================
--
-- Keenam nama ini ada di KEDUA tabel dengan arti berbeda, dan dua di antaranya
-- bahkan bertipe berbeda:
--
--     kolom                work object      worklist        keterangan
--     PXINSNAME            6.810 nilai      258.480 nilai   kunci klaim vs kunci assignment
--     PXSAVEDATETIME       TIMESTAMP(6)     DATE            TIPE BERBEDA
--     PXUPDATEDATETIME     TIMESTAMP(6)     DATE            TIPE BERBEDA
--     PXUPDATEOPERATOR     111 nilai        49 nilai
--     PXUPDATEOPNAME       111 nilai        49 nilai
--     PYLABEL              17 nilai         16.156 nilai    label klaim vs label assignment
--
-- Aturannya mengikuti yang SUDAH BERLAKU di tabel ini: PXCREATEDATETIME yang
-- ada sekarang bertipe TIMESTAMP(6), yaitu tipe work object. Jadi nama tanpa
-- akhiran berarti work object, dan varian worklist diberi akhiran _ASSIGN.
--
-- ============================================================================
-- TAHAP 1 — MENGHIDUPKAN LAYAR INBOX (13 kolom)
-- ============================================================================
--
-- Ketujuh tab layar Inbox yang belum dapat dibangun seluruhnya terhalang
-- kolom-kolom ini, ditambah satu kolom yang hari ini tampil selalu kosong.
--
-- Tahap ini berdiri sendiri: menjalankannya saja sudah membuka tab-tab itu,
-- tanpa menunggu tahap 2 dan 3.

ALTER TABLE POOLDATA.T_CLAIMLIST_ADMIN ADD (
    DOKUMENLENGKAP_1               VARCHAR2(32 CHAR),    --   2 · tab Complete / Not complete documents
    ISPENDINGCLOSE                 VARCHAR2(5 CHAR),     --   2 · tab Temporary Close
    SURVEYORTYPE_1                 VARCHAR2(32 CHAR),    --   4 · tab Internal Surveyor / Loss Adjuster
    ADJUSTERPIC_1                  VARCHAR2(150 CHAR),   --  93 · tab Loss Adjuster
    ADJUSTERSTATUS_1               VARCHAR2(150 CHAR),   --  19 · tab Loss Adjuster
    STATUSKOMUNIKASI_1             VARCHAR2(32 CHAR),    --   2 · tab Not Answered / Replied
    SURVEYORNAME_1                 VARCHAR2(150 CHAR),   -- 101
    SURVEYORNAMEMARINE_1           VARCHAR2(150 CHAR),   --   4 · surveyor lini Marine
    TANGGALDOKLENGKAP              TIMESTAMP(6),         -- 181
    STATUSCLAIM_1                  VARCHAR2(100 CHAR),   --  24 · SUMBER kolom layar "Status ASM"
    PXDEADLINETIME                 DATE,                 --  53 · tab Deadline To Temporary Close
    PXGOALTIME                     DATE,                 --  53
    PYASSIGNMENTSTATUS             VARCHAR2(32 CHAR)     -- 344
);

-- Kenapa STATUSCLAIM_1 masuk tahap 1 meski bukan soal tab.
--
-- Kolom layar "Status ASM" hari ini dipetakan ke STATUSLOCK_1, dan kolom itu
-- KOSONG DI SELURUH 1.014 BARIS. Sistem lama tidak menyimpan labelnya: ia
-- menyimpan KODE di STATUSCLAIM_1, lalu mencarinya ke v_sts_claim.LSC_ID.
--
-- Tanpa kolom ini, kolom "Status ASM" tidak punya sumber sama sekali.

-- ============================================================================
-- TAHAP 2 — DATA BISNIS YANG DIPAKAI LAPORAN DAN ALUR LAIN (56 kolom)
-- ============================================================================

ALTER TABLE POOLDATA.T_CLAIMLIST_ADMIN ADD (
    -- identitas dan penomoran
    PXINSNAME                      VARCHAR2(128 CHAR),   -- 6.810 · kunci work object
    CASEID                         VARCHAR2(32 CHAR),    --   263
    CASEID_1                       VARCHAR2(32 CHAR),    --   264
    REFNO_1                        VARCHAR2(101 CHAR),   --   192
    BOOKNO_1                       VARCHAR2(32 CHAR),    --   109
    IDADJUSTCLAIM_1                VARCHAR2(32 CHAR),    --    15

    -- lini bisnis dan sumber
    SOURCEOFBUSINESS               VARCHAR2(32 CHAR),    --   118
    BUSINESSTYPE                   VARCHAR2(32 CHAR),    --    40
    TYPEPROTECTION                 VARCHAR2(32 CHAR),    --     9
    TYPEOFCLAIM_1                  VARCHAR2(32 CHAR),    --     2
    TKA_1                          VARCHAR2(1 CHAR),     --     2
    EXGRATIA_1                     VARCHAR2(32 CHAR),    --     2
    MSIG_1                         VARCHAR2(32 CHAR),    --     1

    -- status
    PNCSTATUS_1                    VARCHAR2(100 CHAR),   --     5
    STATUSCASE_1                   VARCHAR2(150 CHAR),   --     3
    STATUSKLAIM_1                  VARCHAR2(100 CHAR),   --     3
    ASMSTATUS_1                    VARCHAR2(32 CHAR),    --     2

    -- tanggal proses
    RECEIVEDDATE_1                 VARCHAR2(32 CHAR),    -- 2.827
    DATEOFCOMITEE_1                VARCHAR2(32 CHAR),    --   991
    DATEOFSENTDOCUMENT_1           TIMESTAMP(6),         --   970
    CLOSECLAIMDATE_1               TIMESTAMP(6),         --   514
    APPOINTMENTDATE_1              TIMESTAMP(6),         --   254
    RESCHEDULEDATE_1               TIMESTAMP(6),         --   320
    SURVEYDATE_1                   TIMESTAMP(6),         --    29
    LAMAKLAIM_1                    TIMESTAMP(6),         --    86
    INPUTDATE                      VARCHAR2(32 CHAR),    --   108

    -- survei dan penutupan
    RESCHEDULELOCATION_1           VARCHAR2(32 CHAR),    --   108
    CLOSECLAIMNOTE_1               VARCHAR2(1500 CHAR),  --   126
    NUMBEROFDOCUMENT_1             NUMBER(18,0),         --     8
    KETERANGAN_1                   VARCHAR2(32 CHAR),    --    49
    ADJUSTERACCEPT_1               VARCHAR2(32 CHAR),    --     2

    -- alur PUCL / RCL
    RCL_PUCL_1                     VARCHAR2(32 CHAR),    --     3
    PUCLAPPROVE_1                  VARCHAR2(32 CHAR),    --     2
    TANGGALKIRIMPUCL_1             TIMESTAMP(6),         --    88
    TANGGALCETAKDOKUMENPUCL_1      TIMESTAMP(6),         --    74
    KOMENTARPUCL_1                 VARCHAR2(1000 CHAR),  --    30
    KOMENTARANALISATOR_1           VARCHAR2(1500 CHAR),  --    45

    -- jejak siapa-kapan, dari work object
    PXSAVEDATETIME                 TIMESTAMP(6),         -- 6.811
    PXUPDATEDATETIME               TIMESTAMP(6),         -- 6.808
    PXUPDATEOPERATOR               VARCHAR2(128 CHAR),   --   111
    PXUPDATEOPNAME                 VARCHAR2(128 CHAR),   --   111
    PYLABEL                        VARCHAR2(64 CHAR),    --    17
    PYORIGUSERID                   VARCHAR2(128 CHAR),   --    90
    PYRESOLVEDUSERID               VARCHAR2(128 CHAR),   --    56
    PYRESOLVEDTIMESTAMP            TIMESTAMP(6),         -- 3.142
    PYREOPENTIMESTAMP              TIMESTAMP(6),         --    47
    PYREOPENCOUNT                  NUMBER(18,0),         --     7

    -- jejak siapa-kapan, dari worklist — AKHIRAN _ASSIGN, lihat catatan di atas
    PXINSNAME_ASSIGN               VARCHAR2(128 CHAR),   -- 258.480
    PXSAVEDATETIME_ASSIGN          DATE,                 -- 106.544 · DATE, bukan TIMESTAMP
    PXUPDATEDATETIME_ASSIGN        DATE,                 --     193 · DATE, bukan TIMESTAMP
    PXUPDATEOPERATOR_ASSIGN        VARCHAR2(128 CHAR),   --      49
    PXUPDATEOPNAME_ASSIGN          VARCHAR2(128 CHAR),   --      49
    PYLABEL_ASSIGN                 VARCHAR2(64 CHAR),    --  16.156
    PXASSIGNEDUSERNAME             VARCHAR2(128 CHAR),   --     981
    PXTASKNAME                     VARCHAR2(32 CHAR),    --      73
    PYERRORMESSAGE                 VARCHAR2(128 CHAR)    --     100
);

-- ============================================================================
-- TAHAP 3 — DATA PRIBADI NASABAH: JANGAN JALANKAN TANPA KEPUTUSAN (8 kolom)
-- ============================================================================
--
-- Kedelapan kolom ini memuat data pribadi. Menambahkannya memperluas permukaan
-- data pribadi ke tabel baru — dan karena `D-64` menetapkan staging memakai
-- salinan produksi APA ADANYA tanpa penyamaran, seluruhnya ikut tersalin ke
-- staging.
--
-- Blok ini sengaja DIKOMENTARI. Ia dijalankan hanya setelah Work Owner
-- menyatakan kolom mana yang benar-benar dibutuhkan layar atau laporan.
--
-- ALTER TABLE POOLDATA.T_CLAIMLIST_ADMIN ADD (
--     NOKTP_1                        VARCHAR2(32 CHAR),    -- 325
--     EMAIL_1                        VARCHAR2(256 CHAR),   -- 212
--     SENDER_1                       VARCHAR2(100 CHAR),   -- 181
--     SUBJECTEMAIL_1                 VARCHAR2(32 CHAR),    --  12
--     ASMCLIENTID_1                  VARCHAR2(100 CHAR),   -- 266
--     ASMCLIENTID_2                  VARCHAR2(100 CHAR),   -- 186
--     LOCATION_1                     VARCHAR2(1500 CHAR),  -- 546
--     TANGGALSELESAIRAWATINAP_1      TIMESTAMP(6)          --  14 · data medis
-- );

-- ============================================================================
-- INDEX
-- ============================================================================
--
-- Hanya untuk kolom yang benar-benar dipakai MENYARING, bukan yang sekadar
-- ditampilkan. Index atas kolom bernilai dua — DOKUMENLENGKAP_1,
-- ISPENDINGCLOSE — sengaja TIDAK dibuat: selektivitasnya terlalu rendah untuk
-- membayar biaya tulisnya.

CREATE INDEX IX_CLAIMLIST_SURVEYORTYPE ON POOLDATA.T_CLAIMLIST_ADMIN (SURVEYORTYPE_1);
CREATE INDEX IX_CLAIMLIST_STATUSCLAIM  ON POOLDATA.T_CLAIMLIST_ADMIN (STATUSCLAIM_1);
CREATE INDEX IX_CLAIMLIST_DEADLINE     ON POOLDATA.T_CLAIMLIST_ADMIN (PXDEADLINETIME);

-- ============================================================================
-- SESUDAH DIJALANKAN
-- ============================================================================
--
-- 1. Statistik tabel ini menyesatkan: ALL_TABLES mencatat NUM_ROWS = 1
--    sementara isinya 1.014. Kumpulkan ulang:
--
--        BEGIN DBMS_STATS.GATHER_TABLE_STATS('POOLDATA','T_CLAIMLIST_ADMIN'); END;
--
-- 2. Kolom yang ditambahkan TIDAK terisi dengan sendirinya. Proses pengisi
--    tabel — yang menurut Work Owner baru berjalan di lingkungan testing —
--    harus ikut diperluas. Tanpa itu, hasilnya sama dengan keadaan
--    STATUSLOCK_1 dan REQUESTSURVEY_1 hari ini: kolomnya ada, isinya tidak
--    pernah ditulis.
--
-- 3. Selama proses pengisi dan aplikasi Go sama-sama menulis tabel ini, `P-1`
--    dilanggar — satu tabel wajib punya tepat satu penulis. Siapa penulisnya
--    harus ditetapkan sebelum modul mana pun mulai menulis.
