-- 0012 — CPNC_PENGGUNA: kolom KODE_CABANG_DETAIL (Oracle 19c)
--
-- Dijalankan DBA. Menempuh permintaan tertulis, persetujuan Work Owner, lalu pengujian
-- dengan MENJALANKAN PEGA DAN GO BERSAMAAN (`D-63`). Akun aplikasi tidak memiliki hak DDL.
--
-- ============================================================================
-- KENAPA SATU KOLOM LAGI, PADAHAL KODE_CABANG SUDAH ADA
-- ============================================================================
--
-- Karena HCQ mengirim EMPAT nilai cabang pada satu respons, dan ketiganya berada di ruang
-- kode yang BERBEDA. Contoh nyata yang diberikan Work Owner 2026-09-16
-- (`internal/auth/provider/hcq_test.go`):
--
--     "BranchCode"       : "001"            3 digit
--     "DetailBranchCode" : "001"            3 digit
--     "NewBranchCode"    : "100081"         6 digit
--     "BranchName"       : "KANTOR PUSAT"
--
-- `KODE_CABANG` yang sudah ada menyimpan `BranchCode`. Kolom ini menyimpan
-- `DetailBranchCode`, yang Work Owner tetapkan (2026-09-28) SAMA DENGAN `LDC_ID` pada
-- `GENERAL.LST_DET_CABANG@asmd.sinarmas.co.id` — dan tabel itulah yang menerjemahkannya
-- ke kode cabang yang dipakai data klaim, lewat kolom `LDC_ID_PEGA`.
--
-- Pada contoh di atas keduanya kebetulan bernilai sama. Keduanya tetap disimpan terpisah
-- karena tidak ada yang menjamin itu berlaku untuk setiap pegawai, dan menyamakannya
-- berarti menebak — pada nilai yang menentukan data cabang MANA yang terlihat seseorang.
--
-- ============================================================================
-- PANJANG DAN TIPE
-- ============================================================================
--
-- `VARCHAR2(32)`, sama dengan `KODE_CABANG` di sebelahnya. Diukur dari katalog:
-- `GENERAL.LST_DET_CABANG.LDC_ID` berisi 804 nilai, seluruhnya unik, dan yang terpanjang
-- jauh di bawah 32 karakter. Panjang yang sama dengan tetangganya dipilih supaya tidak ada
-- dua batas berbeda untuk dua kolom yang menyimpan hal sejenis.
--
-- NULLABLE dengan sengaja, dan itu bukan kelonggaran: `POOLDATA.M_LOGIN_PNC` — sumber
-- identitas pengguna NON-KARYAWAN — hanya memuat `LOGIN_ID`, `LOGIN_NAME`,
-- `HASH_PASSWORD`, `ACTIVE_STATUS`, dan `LINE_BUSINESS`. Tidak ada satu pun kolom cabang di
-- sana, sehingga pengguna non-karyawan memang tidak punya nilai ini.
--
-- ============================================================================
-- AMAN DIULANG
-- ============================================================================
--
-- Menambah kolom yang sudah ada menghasilkan ORA-01430. Blok di bawah menangkapnya
-- sehingga berkas ini dapat dijalankan ulang tanpa merusak apa pun — DBA tidak perlu
-- memeriksa lebih dulu apakah ia sudah pernah dijalankan.

DECLARE
    sudah_ada EXCEPTION;
    PRAGMA EXCEPTION_INIT(sudah_ada, -1430);
BEGIN
    EXECUTE IMMEDIATE
        'ALTER TABLE CPNC_PENGGUNA ADD (KODE_CABANG_DETAIL VARCHAR2(32))';
EXCEPTION
    WHEN sudah_ada THEN NULL;
END;
/

COMMENT ON COLUMN CPNC_PENGGUNA.KODE_CABANG_DETAIL IS
    'HCQ Placement.DetailBranchCode = LDC_ID pada GENERAL.LST_DET_CABANG@asmd';
