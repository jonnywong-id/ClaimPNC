-- Lini bisnis seorang pengguna, pengganti `OperatorID.pyPosition` Pega.
--
-- ============================================================================
-- KOLOM INI BELUM ADA
-- ============================================================================
--
-- `LINEBUSINESS` ditambahkan migrasi 0004, yang menempuh `D-63` — permintaan tertulis,
-- persetujuan Work Owner, pelaksanaan DBA — dan BELUM dijalankan di lingkungan mana pun.
--
-- Sampai itu terjadi, kueri ini gagal dengan ORA-00904 (identifier tidak sah). Kegagalan
-- itu DITANGANI, bukan dibiarkan mematikan layar: usecase memperlakukannya sama dengan
-- "lini tidak diketahui", yang jatuh ke tanpa batas — persis perilaku Pega saat potongan
-- WHERE-nya tidak terbentuk.
--
-- ============================================================================
-- KENAPA M_LOGIN_PNC, BUKAN MST_USER_TEKNIK
-- ============================================================================
--
-- `POOLDATA.MST_USER_TEKNIK` sudah memuat kolom `TYPE_BUSINESS` dengan domain nilai yang
-- sama, dan sempat diusulkan sebagai sumbernya. Usul itu DITOLAK Work Owner
-- (2026-09-19) dengan alasan yang menentukan: tabel itu hanya memuat **PIC Teknik**,
-- bukan seluruh pengguna aplikasi.
--
-- `M_LOGIN_PNC` dipilih karena ia akan dipakai untuk karyawan juga — mengatur group akses
-- dan akses menu — bukan hanya non-karyawan seperti yang berlaku hari ini.
--
-- ============================================================================
-- PENCOCOKAN LOGIN_ID
-- ============================================================================
--
-- Memakai UPPER(TRIM(...)) di KEDUA sisi. Kolomnya VARCHAR2 tanpa penyeragaman, dan
-- perbandingan yang hanya satu sisinya diseragamkan tidak pernah cocok — gagalnya diam,
-- dan akibatnya pengguna kehilangan batas datanya tanpa satu pun pesan.

-- name: line_business_for
SELECT LINEBUSINESS
  FROM POOLDATA.M_LOGIN_PNC
 WHERE UPPER(TRIM(LOGIN_ID)) = UPPER(TRIM(:1))
   AND ACTIVE_STATUS = '1'

-- name: line_business_check_table
-- Memeriksa keberadaan kolom tanpa membaca satu baris pun.
--
-- Dipakai mode `-periksa` untuk menjawab "apakah migrasi 0004 sudah dijalankan?" — sebuah
-- pertanyaan yang hari ini hanya dapat dijawab dengan mencoba. WHERE 1 = 0 membuat Oracle
-- tetap memvalidasi nama kolomnya, sehingga ORA-00904 tetap muncul bila belum ada.
SELECT LINEBUSINESS
  FROM POOLDATA.M_LOGIN_PNC
 WHERE 1 = 0
