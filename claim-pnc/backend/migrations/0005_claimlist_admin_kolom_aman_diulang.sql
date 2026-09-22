-- 0005 (varian AMAN DIULANG) — menambah kolom T_CLAIMLIST_ADMIN
--
-- ============================================================================
-- KENAPA ADA DUA BERKAS
-- ============================================================================
--
--   0005_claimlist_admin_kolom.up.sql        ALTER polos — mudah di-review,
--                                            GAGAL bila kolomnya sudah ada
--   0005_claimlist_admin_kolom_aman_diulang  berkas ini — boleh dijalankan
--                                            berkali-kali
--
-- Lingkup sekarang: **ASM saja**. Entitas lain belum dimigrasikan.
--
-- Berkas ini opsional. Gunakan bila tahap 1 dan tahap 2 dijalankan terpisah,
-- atau bila satu jalannya gagal di tengah: `ALTER TABLE ... ADD` polos berhenti
-- dengan ORA-01430 pada kolom pertama yang sudah ada, dan menyisakan tabel
-- setengah jadi. Berkas ini memeriksa lebih dulu, melewati yang sudah ada, dan
-- mencetak apa yang dikerjakannya.
--
-- Bila cukup sekali jalan dan ingin DDL yang paling mudah di-review, pakai
-- 0005_claimlist_admin_kolom.up.sql saja.
--
-- Tipe dan panjang seluruhnya disalin dari katalog Oracle 2026-09-22 —
-- `ALL_TAB_COLUMNS` pada DATAPEGA.PC_ASM_FW_GCNMFW_WORK dan PC_ASSIGN_WORKLIST.
--
-- Jalankan sebagai pemilik POOLDATA, atau dengan hak ALTER ANY TABLE.
-- Dijalankan DBA (`D-63`); akun aplikasi tidak memiliki hak DDL.

SET SERVEROUTPUT ON SIZE UNLIMITED

DECLARE
    -- Daftar kolom yang akan ditambahkan.
    --
    -- TAHAP dipakai supaya satu berkas dapat menjalankan tahap 1 saja, atau
    -- tahap 1 dan 2 sekaligus. Ubah c_tahap_maksimum di bawah.
    TYPE t_kolom IS RECORD (
        tahap  PLS_INTEGER,
        nama   VARCHAR2(30),
        tipe   VARCHAR2(40)
    );
    TYPE t_daftar IS TABLE OF t_kolom;

    -- Tahap 1 = menghidupkan layar Inbox · Tahap 2 = data bisnis lain.
    -- Tahap 3 (data pribadi) TIDAK ada di sini; ia menuntut keputusan
    -- tersendiri. Lihat 0005_claimlist_admin_kolom.up.sql.
    c_tahap_maksimum CONSTANT PLS_INTEGER := 2;

    l_kolom t_daftar := t_daftar(
        -- ---- TAHAP 1 — menghidupkan tujuh tab layar Inbox -----------------
        t_kolom(1, 'DOKUMENLENGKAP_1',          'VARCHAR2(32 CHAR)'),
        t_kolom(1, 'ISPENDINGCLOSE',            'VARCHAR2(5 CHAR)'),
        t_kolom(1, 'SURVEYORTYPE_1',            'VARCHAR2(32 CHAR)'),
        t_kolom(1, 'ADJUSTERPIC_1',             'VARCHAR2(150 CHAR)'),
        t_kolom(1, 'ADJUSTERSTATUS_1',          'VARCHAR2(150 CHAR)'),
        t_kolom(1, 'STATUSKOMUNIKASI_1',        'VARCHAR2(32 CHAR)'),
        t_kolom(1, 'SURVEYORNAME_1',            'VARCHAR2(150 CHAR)'),
        t_kolom(1, 'SURVEYORNAMEMARINE_1',      'VARCHAR2(150 CHAR)'),
        t_kolom(1, 'TANGGALDOKLENGKAP',         'TIMESTAMP(6)'),
        t_kolom(1, 'STATUSCLAIM_1',             'VARCHAR2(100 CHAR)'),
        t_kolom(1, 'PXDEADLINETIME',            'DATE'),
        t_kolom(1, 'PXGOALTIME',                'DATE'),
        t_kolom(1, 'PYASSIGNMENTSTATUS',        'VARCHAR2(32 CHAR)'),

        -- ---- TAHAP 2 — identitas dan penomoran ---------------------------
        t_kolom(2, 'PXINSNAME',                 'VARCHAR2(128 CHAR)'),
        t_kolom(2, 'CASEID',                    'VARCHAR2(32 CHAR)'),
        t_kolom(2, 'CASEID_1',                  'VARCHAR2(32 CHAR)'),
        t_kolom(2, 'REFNO_1',                   'VARCHAR2(101 CHAR)'),
        t_kolom(2, 'BOOKNO_1',                  'VARCHAR2(32 CHAR)'),
        t_kolom(2, 'IDADJUSTCLAIM_1',           'VARCHAR2(32 CHAR)'),

        -- ---- TAHAP 2 — lini bisnis dan sumber ----------------------------
        t_kolom(2, 'SOURCEOFBUSINESS',          'VARCHAR2(32 CHAR)'),
        t_kolom(2, 'BUSINESSTYPE',              'VARCHAR2(32 CHAR)'),
        t_kolom(2, 'TYPEPROTECTION',            'VARCHAR2(32 CHAR)'),
        t_kolom(2, 'TYPEOFCLAIM_1',             'VARCHAR2(32 CHAR)'),
        t_kolom(2, 'TKA_1',                     'VARCHAR2(1 CHAR)'),
        t_kolom(2, 'EXGRATIA_1',                'VARCHAR2(32 CHAR)'),
        t_kolom(2, 'MSIG_1',                    'VARCHAR2(32 CHAR)'),

        -- ---- TAHAP 2 — status --------------------------------------------
        t_kolom(2, 'PNCSTATUS_1',               'VARCHAR2(100 CHAR)'),
        t_kolom(2, 'STATUSCASE_1',              'VARCHAR2(150 CHAR)'),
        t_kolom(2, 'STATUSKLAIM_1',             'VARCHAR2(100 CHAR)'),
        t_kolom(2, 'ASMSTATUS_1',               'VARCHAR2(32 CHAR)'),

        -- ---- TAHAP 2 — tanggal proses ------------------------------------
        t_kolom(2, 'RECEIVEDDATE_1',            'VARCHAR2(32 CHAR)'),
        t_kolom(2, 'DATEOFCOMITEE_1',           'VARCHAR2(32 CHAR)'),
        t_kolom(2, 'DATEOFSENTDOCUMENT_1',      'TIMESTAMP(6)'),
        t_kolom(2, 'CLOSECLAIMDATE_1',          'TIMESTAMP(6)'),
        t_kolom(2, 'APPOINTMENTDATE_1',         'TIMESTAMP(6)'),
        t_kolom(2, 'RESCHEDULEDATE_1',          'TIMESTAMP(6)'),
        t_kolom(2, 'SURVEYDATE_1',              'TIMESTAMP(6)'),
        t_kolom(2, 'LAMAKLAIM_1',               'TIMESTAMP(6)'),
        t_kolom(2, 'INPUTDATE',                 'VARCHAR2(32 CHAR)'),

        -- ---- TAHAP 2 — survei dan penutupan ------------------------------
        t_kolom(2, 'RESCHEDULELOCATION_1',      'VARCHAR2(32 CHAR)'),
        t_kolom(2, 'CLOSECLAIMNOTE_1',          'VARCHAR2(1500 CHAR)'),
        t_kolom(2, 'NUMBEROFDOCUMENT_1',        'NUMBER(18,0)'),
        t_kolom(2, 'KETERANGAN_1',              'VARCHAR2(32 CHAR)'),
        t_kolom(2, 'ADJUSTERACCEPT_1',          'VARCHAR2(32 CHAR)'),

        -- ---- TAHAP 2 — alur PUCL / RCL -----------------------------------
        t_kolom(2, 'RCL_PUCL_1',                'VARCHAR2(32 CHAR)'),
        t_kolom(2, 'PUCLAPPROVE_1',             'VARCHAR2(32 CHAR)'),
        t_kolom(2, 'TANGGALKIRIMPUCL_1',        'TIMESTAMP(6)'),
        t_kolom(2, 'TANGGALCETAKDOKUMENPUCL_1', 'TIMESTAMP(6)'),
        t_kolom(2, 'KOMENTARPUCL_1',            'VARCHAR2(1000 CHAR)'),
        t_kolom(2, 'KOMENTARANALISATOR_1',      'VARCHAR2(1500 CHAR)'),

        -- ---- TAHAP 2 — jejak dari WORK OBJECT ----------------------------
        t_kolom(2, 'PXSAVEDATETIME',            'TIMESTAMP(6)'),
        t_kolom(2, 'PXUPDATEDATETIME',          'TIMESTAMP(6)'),
        t_kolom(2, 'PXUPDATEOPERATOR',          'VARCHAR2(128 CHAR)'),
        t_kolom(2, 'PXUPDATEOPNAME',            'VARCHAR2(128 CHAR)'),
        t_kolom(2, 'PYLABEL',                   'VARCHAR2(64 CHAR)'),
        t_kolom(2, 'PYORIGUSERID',              'VARCHAR2(128 CHAR)'),
        t_kolom(2, 'PYRESOLVEDUSERID',          'VARCHAR2(128 CHAR)'),
        t_kolom(2, 'PYRESOLVEDTIMESTAMP',       'TIMESTAMP(6)'),
        t_kolom(2, 'PYREOPENTIMESTAMP',         'TIMESTAMP(6)'),
        t_kolom(2, 'PYREOPENCOUNT',             'NUMBER(18,0)'),

        -- ---- TAHAP 2 — jejak dari WORKLIST, berakhiran _ASSIGN -----------
        -- Keenam nama pertama bertabrakan dengan kolom work object di atas,
        -- dan DUA di antaranya bertipe berbeda: DATE, bukan TIMESTAMP(6).
        -- Nama tanpa akhiran = work object, mengikuti PXCREATEDATETIME yang
        -- sudah ada di tabel ini dan bertipe TIMESTAMP(6).
        t_kolom(2, 'PXINSNAME_ASSIGN',          'VARCHAR2(128 CHAR)'),
        t_kolom(2, 'PXSAVEDATETIME_ASSIGN',     'DATE'),
        t_kolom(2, 'PXUPDATEDATETIME_ASSIGN',   'DATE'),
        t_kolom(2, 'PXUPDATEOPERATOR_ASSIGN',   'VARCHAR2(128 CHAR)'),
        t_kolom(2, 'PXUPDATEOPNAME_ASSIGN',     'VARCHAR2(128 CHAR)'),
        t_kolom(2, 'PYLABEL_ASSIGN',            'VARCHAR2(64 CHAR)'),
        t_kolom(2, 'PXASSIGNEDUSERNAME',        'VARCHAR2(128 CHAR)'),
        t_kolom(2, 'PXTASKNAME',                'VARCHAR2(32 CHAR)'),
        t_kolom(2, 'PYERRORMESSAGE',            'VARCHAR2(128 CHAR)')
    );

    l_ada        PLS_INTEGER;
    l_ditambah   PLS_INTEGER := 0;
    l_dilewati   PLS_INTEGER := 0;
    l_tabel_ada  PLS_INTEGER;
BEGIN
    -- Tabelnya sendiri harus ada lebih dulu. Pada portal yang belum punya
    -- T_CLAIMLIST_ADMIN, berkas ini berhenti dengan pesan yang jelas alih-alih
    -- menghasilkan 70 galat berturut-turut.
    SELECT COUNT(*) INTO l_tabel_ada
      FROM all_tables
     WHERE owner = 'POOLDATA' AND table_name = 'T_CLAIMLIST_ADMIN';

    IF l_tabel_ada = 0 THEN
        DBMS_OUTPUT.PUT_LINE('BERHENTI: POOLDATA.T_CLAIMLIST_ADMIN belum ada di database ini.');
        DBMS_OUTPUT.PUT_LINE('          Buat tabelnya lebih dulu sebelum menjalankan 0005.');
        RETURN;
    END IF;

    FOR i IN 1 .. l_kolom.COUNT LOOP
        CONTINUE WHEN l_kolom(i).tahap > c_tahap_maksimum;

        SELECT COUNT(*) INTO l_ada
          FROM all_tab_columns
         WHERE owner       = 'POOLDATA'
           AND table_name  = 'T_CLAIMLIST_ADMIN'
           AND column_name = l_kolom(i).nama;

        IF l_ada > 0 THEN
            l_dilewati := l_dilewati + 1;
            DBMS_OUTPUT.PUT_LINE('lewat   ' || l_kolom(i).nama || ' — sudah ada');
        ELSE
            EXECUTE IMMEDIATE
                'ALTER TABLE POOLDATA.T_CLAIMLIST_ADMIN ADD ('
                || l_kolom(i).nama || ' ' || l_kolom(i).tipe || ')';
            l_ditambah := l_ditambah + 1;
            DBMS_OUTPUT.PUT_LINE('tambah  ' || l_kolom(i).nama || ' ' || l_kolom(i).tipe);
        END IF;
    END LOOP;

    DBMS_OUTPUT.PUT_LINE('---');
    DBMS_OUTPUT.PUT_LINE('ditambah ' || l_ditambah || ' kolom · dilewati ' || l_dilewati);
END;
/

-- ============================================================================
-- INDEX — juga aman diulang
-- ============================================================================
--
-- Hanya untuk kolom yang dipakai MENYARING, bukan yang sekadar ditampilkan.
-- Kolom bernilai dua — DOKUMENLENGKAP_1, ISPENDINGCLOSE — sengaja TIDAK
-- di-index: selektivitasnya terlalu rendah untuk membayar biaya tulisnya.

DECLARE
    TYPE t_index IS RECORD (nama VARCHAR2(30), kolom VARCHAR2(30));
    TYPE t_daftar IS TABLE OF t_index;

    l_index t_daftar := t_daftar(
        t_index('IX_CLAIMLIST_SURVEYORTYPE', 'SURVEYORTYPE_1'),
        t_index('IX_CLAIMLIST_STATUSCLAIM',  'STATUSCLAIM_1'),
        t_index('IX_CLAIMLIST_DEADLINE',     'PXDEADLINETIME')
    );
    l_ada PLS_INTEGER;
BEGIN
    FOR i IN 1 .. l_index.COUNT LOOP
        SELECT COUNT(*) INTO l_ada
          FROM all_indexes
         WHERE owner = 'POOLDATA' AND index_name = l_index(i).nama;

        IF l_ada > 0 THEN
            DBMS_OUTPUT.PUT_LINE('lewat   index ' || l_index(i).nama || ' — sudah ada');
        ELSE
            EXECUTE IMMEDIATE
                'CREATE INDEX POOLDATA.' || l_index(i).nama
                || ' ON POOLDATA.T_CLAIMLIST_ADMIN (' || l_index(i).kolom || ')';
            DBMS_OUTPUT.PUT_LINE('buat    index ' || l_index(i).nama);
        END IF;
    END LOOP;
END;
/

-- ============================================================================
-- STATISTIK
-- ============================================================================
--
-- ALL_TABLES mencatat NUM_ROWS = 1 untuk tabel ini sementara isinya 1.014 —
-- statistik dikumpulkan saat tabelnya masih berisi satu baris. Optimizer akan
-- memilih rencana yang salah selama itu dibiarkan.

BEGIN
    DBMS_STATS.GATHER_TABLE_STATS('POOLDATA', 'T_CLAIMLIST_ADMIN');
END;
/

-- ============================================================================
-- VERIFIKASI — jalankan setelahnya
-- ============================================================================
--
-- Harapan: 53 kolom setelah tahap 1, 109 setelah tahap 2.
-- (40 kolom awal + 13 + 56)

SELECT COUNT(*) AS JUMLAH_KOLOM
  FROM all_tab_columns
 WHERE owner = 'POOLDATA' AND table_name = 'T_CLAIMLIST_ADMIN';

-- Kolom mana yang masih kosong setelah pengisi tabel dijalankan. Yang muncul
-- di sini adalah kolom yang ada tetapi tidak pernah ditulis — keadaan yang
-- sekarang dialami STATUSLOCK_1 dan REQUESTSURVEY_1.

SELECT column_name
  FROM all_tab_columns
 WHERE owner = 'POOLDATA'
   AND table_name = 'T_CLAIMLIST_ADMIN'
   AND NVL(num_distinct, 0) = 0
 ORDER BY column_name;
