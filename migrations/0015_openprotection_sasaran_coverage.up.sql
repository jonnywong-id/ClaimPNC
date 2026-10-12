-- 0015 — Sasaran perubahan Cause of Loss pada POOLDATA.T_CLAIM_OPENPROTECTION
--
-- ============================================================================
-- SUDAH DIJALANKAN Work Owner 2026-10-05 — dengan pernyataan yang ditulis sendiri:
--
--     ALTER TABLE T_CLAIM_OPENPROTECTION ADD(
--         OBJECT_ID          VARCHAR2(30 BYTE),
--         OBJECT_COVERAGE_ID VARCHAR2(30 BYTE)
--     )
--
-- Tipenya COCOK dengan kolom asalnya, jadi blok di bawah tidak mengubah apa pun lagi
-- (ia melewati kolom yang sudah ada). Terverifikasi dari DDL proyek yang ditulis terhadap
-- skema nyata — `docs/ddl/tc_pnc_komite.sql:39-40` dan tujuh tabel lain di
-- `tc_pnc_object_tree.sql` seluruhnya mendeklarasikan OBJECTID dan OBJECTCOVERAGEID
-- sebagai VARCHAR2(30).
--
-- Berkas ini tetap disimpan dan tetap berlaku untuk BASIS DATA PORTAL LAIN yang belum
-- menerima kolomnya (`ADR-0030` — satu database per entitas).
-- ============================================================================
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- Dijalankan DBA. Menempuh `D-63`: permintaan tertulis tim pengembang, persetujuan Work
-- Owner, pelaksanaan DBA, lalu diuji dengan menjalankan Pega dan Go bersamaan. Akun
-- aplikasi tidak memiliki hak DDL.
--
-- Penjelasan lengkapnya untuk Work Owner ada di
-- `docs/permintaan-kolom-t-claim-openprotection.md`. Berkas ini hanya pelaksanaannya.
--
--
-- ============================================================================
-- KENAPA BERKAS INI ADA
-- ============================================================================
--
-- Akseptasi Open Protection tipe '8' — **Perubahan Cause of Loss** — hari ini menyimpan
-- keputusannya tetapi TIDAK mengubah klaim. Seluruh modul
-- `inbox-accept-open-protection` hanya punya dua pernyataan tulis:
--
--     acceptance_decide        UPDATE POOLDATA.T_CLAIM_OPENPROTECTION  (keputusan)
--     claim_apply_loss_date    UPDATE POOLDATA.T_CLAIM_PNC             (DOL, tipe '7')
--
-- Tipe '7' bisa karena sasarannya KEPALA klaim: satu kolom, satu baris, kunci `CLAIMID`.
-- Tipe '8' tidak bisa karena sasarannya `POOLDATA.T_CLAIM_OBJECTCOVERAGE` — tabel ANAK,
-- yang satu klaimnya dapat memuat banyak objek dan tiap objek banyak coverage.
--
-- `T_CLAIM_OPENPROTECTION` tidak menyimpan coverage MANA yang diubah. Keenam belas
-- kolomnya:
--
--     OPEN_PROTECTION_ID · CREATE_DATE · CREATED_BY · RESOLVED_BY · RESOLVED_DATETIME
--     POLICY_NO · CLAIM_NO · ID_CLAIM · PROTECTION_TYPE_ID · APPROVAL_STATUS · NOTES
--     OLD_DATA · NEW_DATA · OBJECT_NAME · BRANCH_NAME · STATUS_ACTIVE
--
-- `OLD_DATA`/`NEW_DATA` MEMANG berisi Cause of Loss sebelum dan sesudah, jadi nilai
-- barunya sudah diketahui. Yang tidak diketahui adalah ke BARIS MANA ia ditulis.
--
-- `OBJECT_NAME` tidak dapat dipakai sebagai gantinya: ia NAMA (VARCHAR2(500)), bukan
-- kunci. Nama objek tidak dijamin unik dalam satu klaim, dan satu objek dapat punya
-- beberapa coverage — sehingga pencocokan lewat nama dapat menulis Penyebab Kerugian ke
-- coverage YANG SALAH, tanpa galat dan tanpa gejala. Itu kelas cacat yang sama dengan
-- `R-19`, dan itulah alasan tipe '8' sengaja dibiarkan tidak diterapkan sampai kolom ini
-- ada (`docs/catatan-pengembangan.md` §38.14).
--
--
-- ============================================================================
-- KENAPA DUA KOLOM INI, DAN BUKAN YANG LAIN
-- ============================================================================
--
-- Kunci alami satu baris coverage adalah TIGA kolom:
--
--     CLAIMID + OBJECTID + OBJECTCOVERAGEID
--
-- `CLAIMID` sudah ada di tabel proteksi sebagai `ID_CLAIM`. Yang kurang dua sisanya.
--
-- `OBJECTCOVERAGEID` berisi URUTAN coverage DI DALAM objeknya — terverifikasi 2026-09-24
-- pada data nyata: nilainya "1", dan pasangan (klaim, nilai itu) BERULANG lintas objek.
-- Jadi ia bukan pengenal global, dan wajib dipasangkan dengan `OBJECTID`. Dua kolom,
-- bukan satu.
--
-- Dua kandidat lain dipertimbangkan dan ditolak:
--
--   COVERAGEID
--     Itu KODE JENIS JAMINAN, bukan pengenal baris. Satu objek dapat punya dua coverage
--     berkode sama pada klaim yang berbeda, dan ia tidak menunjuk baris.
--
--   URUTAN_OBJEK + URUTAN
--     Pasangan inilah yang dipakai modul `registrasi` (`coverage_perbarui`,
--     `coverage_tandai_sisa`), sehingga tampak paling konsisten. Ia TIDAK dipakai di sini
--     karena kedua kolom itu DITAMBAHKAN migrasi `0008` — artinya hanya terisi pada baris
--     yang DITULIS aplikasi ini.
--
--     DIUKUR 2026-10-05 (kueri §B, dijalankan Work Owner):
--
--         baris coverage seluruhnya          2.630
--         URUTAN_OBJEK terisi                   32   1,2%
--         URUTAN terisi                         32   1,2%
--         OBJECTID terisi                    2.630   100%
--         OBJECTCOVERAGEID terisi            2.630   100%
--
--     Memakai pasangan URUTAN karena itu akan membuat perubahan COL gagal menemukan
--     barisnya pada 2.598 dari 2.630 baris — tanpa galat, hanya UPDATE yang tidak
--     menyentuh apa pun.
--
-- Pasangan `OBJECTID` + `OBJECTCOVERAGEID` dipilih karena ia kunci yang SAMA dengan yang
-- Work Owner tetapkan sendiri untuk `POOLDATA.T_CLAIM_SPREADING` (CREATE_TABLE_2.sql,
-- 2026-09-26): `(CLAIMID, OBJECTID, OBJECTCOVERAGEID, TREATYTYPE)`. Memakai kunci yang
-- sama membuat kedua tabel baru menunjuk baris coverage dengan cara yang sama.
--
--
-- ============================================================================
-- A. YANG HARUS DIPERIKSA DBA SEBELUM MENJALANKAN
-- ============================================================================
--
-- 1. PASTIKAN NAMANYA BELUM DIPAKAI — diharapkan NOL baris:
--
--        SELECT column_name FROM all_tab_cols
--         WHERE owner = 'POOLDATA' AND table_name = 'T_CLAIM_OPENPROTECTION'
--           AND column_name IN ('OBJECT_ID', 'OBJECT_COVERAGE_ID');
--
-- 2. PASTIKAN KOLOM ASALNYA ADA — diharapkan DUA baris. Berkas ini MENYALIN tipenya dari
--    sini, jadi tanpa keduanya ia berhenti dengan galat, bukan menebak:
--
--        SELECT column_name, data_type, char_length, data_precision, data_scale, char_used
--          FROM all_tab_cols
--         WHERE owner = 'POOLDATA' AND table_name = 'T_CLAIM_OBJECTCOVERAGE'
--           AND column_name IN ('OBJECTID', 'OBJECTCOVERAGEID');
--
-- 3. PASTIKAN TABLESPACE sesuai kebijakan. Berkas ini sengaja TIDAK menyebutnya:
--    menyebutkannya berarti menebak tata letak penyimpanan yang dimiliki DBA.
--
--
-- ============================================================================
-- B. KUERI YANG MENENTUKAN APAKAH BERKAS INI BENAR — SUDAH DIJAWAB
-- ============================================================================
--
-- Dijalankan Work Owner 2026-10-05 pada DEV_PEGA83G; hasilnya `2630 · 32 · 32 · 2630 ·
-- 2630`, dan ia MEMBENARKAN pilihan kunci di atas. Angkanya tercatat di bagian
-- "URUTAN_OBJEK + URUTAN".
--
-- Kueri ini tetap dicantumkan supaya dapat diulang pada basis data portal lain — isinya
-- tidak harus sama (`ADR-0030`), dan kesimpulannya hanya berlaku untuk portal yang diukur:
--
--     SELECT COUNT(*)                                         AS BARIS,
--            COUNT(URUTAN_OBJEK)                              AS URUTAN_OBJEK_TERISI,
--            COUNT(URUTAN)                                    AS URUTAN_TERISI,
--            COUNT(OBJECTID)                                  AS OBJECTID_TERISI,
--            COUNT(OBJECTCOVERAGEID)                          AS OBJCOVID_TERISI
--       FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE;
--
-- Yang diharapkan: `OBJECTID_TERISI` dan `OBJCOVID_TERISI` sama dengan `BARIS` (keduanya
-- `NOT NULL` di tabel warisan), sedangkan kedua kolom URUTAN jauh lebih kecil.
--
--
-- ============================================================================
-- KENAPA AMAN
-- ============================================================================
--
-- Kedua kolom NULLABLE tanpa DEFAULT — metadata saja, tidak ada satu pun baris yang
-- ditulis ulang. Tidak ada objek milik sistem lama yang diubah: `T_CLAIM_OBJECTCOVERAGE`
-- hanya DIBACA berkas ini, untuk menyalin tipe kolomnya.
--
-- Idempoten, karena berkas ini dijalankan sekali per basis data portal (`ADR-0030`).
--
-- TIDAK ADA INDEX yang ditambahkan, dan itu disengaja. Penerapan perubahan menemukan
-- barisnya lewat `OPEN_PROTECTION_ID` (primary key); kedua kolom baru dibaca SESUDAH baris
-- itu ketemu, tidak pernah dipakai mencari. Index yang tidak menjawab kueri nyata hanya
-- memperlambat tulis.


-- ---------------------------------------------------------------------------
-- Penambahan kolom — tipenya DISALIN dari kolom asalnya, tidak ditulis tangan.
--
-- Tim pengembang tidak memiliki akses katalog, sehingga menuliskan `VARCHAR2(20)` di sini
-- berarti menebak. Tebakan yang meleset tidak menghasilkan galat: Oracle mengonversi tipe
-- secara diam-diam saat membandingkan, dan yang terlihat hanyalah kueri yang melambat atau
-- pencocokan yang gagal pada nilai berspasi.
--
-- Karena itu blok ini MEMBACA tipe `T_CLAIM_OBJECTCOVERAGE.OBJECTID` dan
-- `.OBJECTCOVERAGEID`, lalu memakai tipe yang sama persis. Bila kolom asalnya tidak
-- ditemukan, ia BERHENTI dengan galat — bukan melanjutkan dengan nilai bawaan.
-- ---------------------------------------------------------------------------

DECLARE
    TYPE t_kolom  IS RECORD (baru VARCHAR2(30), sumber VARCHAR2(30));
    TYPE t_daftar IS TABLE OF t_kolom;

    daftar t_daftar := t_daftar(
        -- OBJECT_ID          <- T_CLAIM_OBJECTCOVERAGE.OBJECTID
        --                       id bisnis objek pertanggungan
        t_kolom('OBJECT_ID',          'OBJECTID'),

        -- OBJECT_COVERAGE_ID <- T_CLAIM_OBJECTCOVERAGE.OBJECTCOVERAGEID
        --                       urutan coverage DI DALAM objek itu; bukan pengenal global
        t_kolom('OBJECT_COVERAGE_ID', 'OBJECTCOVERAGEID')
    );

    ada        NUMBER;
    v_tipe     VARCHAR2(60);
    v_datatype all_tab_cols.data_type%TYPE;
    v_len      all_tab_cols.char_length%TYPE;
    v_prec     all_tab_cols.data_precision%TYPE;
    v_scale    all_tab_cols.data_scale%TYPE;
    v_charused all_tab_cols.char_used%TYPE;
BEGIN
    FOR i IN 1 .. daftar.COUNT LOOP

        SELECT COUNT(*) INTO ada
          FROM all_tab_cols
         WHERE owner       = 'POOLDATA'
           AND table_name  = 'T_CLAIM_OPENPROTECTION'
           AND column_name = daftar(i).baru;

        CONTINUE WHEN ada > 0;

        BEGIN
            SELECT data_type, char_length, data_precision, data_scale, char_used
              INTO v_datatype, v_len, v_prec, v_scale, v_charused
              FROM all_tab_cols
             WHERE owner       = 'POOLDATA'
               AND table_name  = 'T_CLAIM_OBJECTCOVERAGE'
               AND column_name = daftar(i).sumber;
        EXCEPTION
            WHEN NO_DATA_FOUND THEN
                raise_application_error(-20015,
                    'Kolom asal POOLDATA.T_CLAIM_OBJECTCOVERAGE.' || daftar(i).sumber ||
                    ' tidak ditemukan. Berkas ini berhenti alih-alih menebak tipe.');
        END;

        IF v_datatype IN ('VARCHAR2', 'NVARCHAR2', 'CHAR') THEN
            v_tipe := v_datatype || '(' || v_len ||
                      CASE WHEN v_charused = 'C' THEN ' CHAR' ELSE ' BYTE' END || ')';

        ELSIF v_datatype = 'NUMBER' THEN
            v_tipe := CASE
                        WHEN v_prec IS NULL                  THEN 'NUMBER'
                        WHEN v_scale IS NULL OR v_scale = 0  THEN 'NUMBER(' || v_prec || ')'
                        ELSE 'NUMBER(' || v_prec || ',' || v_scale || ')'
                      END;
        ELSE
            raise_application_error(-20016,
                'Tipe kolom asal ' || daftar(i).sumber || ' adalah ' || v_datatype ||
                ', di luar yang ditangani berkas ini. Hentikan dan tinjau ulang.');
        END IF;

        EXECUTE IMMEDIATE
            'ALTER TABLE POOLDATA.T_CLAIM_OPENPROTECTION ADD (' ||
            daftar(i).baru || ' ' || v_tipe || ')';

    END LOOP;
END;
/


-- ---------------------------------------------------------------------------
-- Hak akses untuk akun aplikasi.
--
-- TIDAK ADA grant baru yang dibutuhkan: kedua kolom menumpang tabel yang akun aplikasi
-- sudah boleh SELECT dan UPDATE.
--
-- Yang BELUM tentu ada adalah hak tulis ke tabel sasarannya. Penerapan perubahan Cause of
-- Loss menuntut UPDATE pada POOLDATA.T_CLAIM_OBJECTCOVERAGE, dan sampai hari ini modul
-- Open Protection tidak pernah menulisinya:
--
-- GRANT SELECT, UPDATE ON POOLDATA.T_CLAIM_OBJECTCOVERAGE TO <AKUN_APLIKASI>;
--
-- Itu bukan bagian dari ALTER di atas, dan sengaja dibiarkan sebagai komentar: pemberian
-- hak tulis atas tabel klaim adalah keputusan tersendiri (`P-1` — satu tabel satu penulis),
-- dan harus disetujui bersama, bukan ikut terbawa.
-- ---------------------------------------------------------------------------
