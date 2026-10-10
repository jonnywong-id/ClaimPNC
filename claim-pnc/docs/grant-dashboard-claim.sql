-- ============================================================================
-- GRANT untuk modul Dashboard Claim
-- ============================================================================
--
-- ############################################################################
-- #  JANGAN DIJALANKAN PADA ENTITAS YANG APLIKASINYA MENYAMBUNG SEBAGAI       #
-- #  PEMILIK SKEMA. PADA ASM, TIDAK ADA SATU PUN GRANT YANG DIBUTUHKAN.       #
-- ############################################################################
--
-- KOREKSI 2026-10-08. Berkas ini semula memuat empat GRANT dan menyebut dirinya "siap
-- dijalankan". Itu keliru, dan menghabiskan waktu Work Owner pada tiga galat beruntun
-- (ORA-00987, ORA-00933, ORA-01917) sebelum sebabnya ketahuan:
--
--     .env baris 60 -> POOLDATA_ASM_PENGGUNA=POOLDATA
--
-- Aplikasi menyambung **sebagai pengguna `POOLDATA`**, yaitu PEMILIK skema itu sendiri. Di
-- Oracle, pemilik skema sudah memiliki seluruh hak atas objeknya sendiri secara implisit —
-- tidak ada yang perlu diberikan, dan `GRANT … TO POOLDATA` justru ditolak:
--
--     ORA-01749: you may not GRANT/REVOKE privileges to/from yourself
--
-- Jadi bila tombol Transfer gagal pada ASM, sebabnya BUKAN hak akses. Periksa nama kolom,
-- nama tabel, atau isi datanya. Satu kegagalan serupa sudah pernah terjadi: `pencacah_naik`
-- menulis `TOTAL_JOB`, kolom yang ternyata hanya ada pada view `V_MST_USER_TEKNIS`.
--
-- ============================================================================
-- KAPAN BERKAS INI BARU BERGUNA
-- ============================================================================
--
-- Hanya bila sebuah entitas disetel memakai akun NON-pemilik — yaitu nilai
-- `POOLDATA_<ENTITAS>_PENGGUNA` BUKAN `POOLDATA`.
--
-- Per 2026-10-08 hanya ASM yang terisi, dan nilainya `POOLDATA`. Kelima entitas lain
-- (`ASI`, `SMAS`, `SMI`, `SPK`, `SPKS`) masih kosong — belum disetel sama sekali.
--
-- Periksa dulu, di entitas yang bersangkutan:
--
--     SELECT USER FROM DUAL;        -- dijalankan OLEH akun aplikasi
--
-- Bila hasilnya `POOLDATA`, berhenti di sini: tidak ada yang perlu dijalankan.

-- ----------------------------------------------------------------------------
-- LANGKAH 1 — ganti satu baris ini
-- ----------------------------------------------------------------------------
--
-- Isi dengan nama akun NON-pemilik milik entitas yang sedang disambung. Nilainya ada di
-- `.env` pada `POOLDATA_<ENTITAS>_PENGGUNA`.

DEFINE AKUN = GANTI_DENGAN_NAMA_AKUN

-- ----------------------------------------------------------------------------
-- LANGKAH 2 — blok keempat baris, lalu tekan F5 (Execute as Script)
-- ----------------------------------------------------------------------------
--
-- JANGAN Ctrl+Enter. TOAD mengirim satu pernyataan BESERTA titik komanya, dan Oracle menolak
-- `;` dengan ORA-00933 — titik koma itu pemisah milik alat, bukan sintaks SQL.
--
-- Ketiga hak tulis dipakai SATU transaksi: tombol Transfer. Bila salah satu kurang,
-- pemindahannya batal di tengah transaksi, bukan sebagian berhasil.

GRANT UPDATE (USERTEKNIS_1)  ON POOLDATA.T_CLAIMLIST_ADMIN  TO &AKUN;
GRANT UPDATE (COUNTER_QUOTA) ON POOLDATA.MST_USER_TEKNIK    TO &AKUN;
GRANT UPDATE (PIC)           ON POOLDATA.PEGA_DASHBOARDPNC  TO &AKUN;
GRANT SELECT                 ON POOLDATA.PEGA_DASHBOARDPNC  TO &AKUN;

-- Baris keempat paling mudah terlewat. Oracle TIDAK mendukung SELECT per kolom. Pernyataan
-- tulisnya berbunyi `UPDATE … SET PIC = :1 WHERE NOKLAIM = :2`, dan Oracle menuntut SELECT atas
-- kolom di klausa WHERE. Tanpa baris itu UPDATE-nya ditolak meski hak UPDATE-nya sudah ada.
-- `PEGA_DASHBOARDPNC` satu-satunya tabel yang modul ini tulis tanpa pernah membacanya.

-- ----------------------------------------------------------------------------
-- LANGKAH 3 — ulangi di setiap entitas berakun non-pemilik
-- ----------------------------------------------------------------------------
--
-- Dengan nama akun milik entitas itu sendiri (`ADR-0030`). Menjalankannya hanya di satu entitas
-- membuat Transfer bekerja di satu portal dan gagal di portal lain, dengan galat yang sama
-- persis — dan itu sulit dikenali sebagai masalah izin.

-- ============================================================================
-- PEMERIKSAAN — harus mengembalikan EMPAT baris
-- ============================================================================

SELECT table_name, column_name, privilege
  FROM all_col_privs
 WHERE grantee = UPPER('&AKUN')
   AND table_schema = 'POOLDATA'
   AND (table_name, column_name) IN (('T_CLAIMLIST_ADMIN', 'USERTEKNIS_1'),
                                     ('MST_USER_TEKNIK',   'COUNTER_QUOTA'),
                                     ('PEGA_DASHBOARDPNC', 'PIC'))
UNION ALL
SELECT table_name, '(seluruh kolom)', privilege
  FROM all_tab_privs
 WHERE grantee = UPPER('&AKUN')
   AND table_schema = 'POOLDATA'
   AND table_name = 'PEGA_DASHBOARDPNC'
   AND privilege = 'SELECT'
 ORDER BY 1, 2;

-- ============================================================================
-- ISI TABEL — SUDAH TERTANGANI (2026-10-08)
-- ============================================================================
--
-- Berkas ini semula memperingatkan bahwa `POOLDATA.T_CLAIMLIST_ADMIN` baru terisi 13% dengan
-- tiga kolom kosong di seluruh baris. Backfill sudah dijalankan — `docs/isi-t-claimlist-admin.sql`.
--
-- Keadaan DEV_PEGA83 setelahnya:
--
--     baris               1.059  ->  2.648
--     SURVEYORTYPE_1          0  ->    674     tile Loss Adjuster & Internal Surveyor
--     STATUSCLAIM_1          25  ->  1.998     kolom Claim status
--
-- Angka "1.014 dari 7.703" berasal dari lingkungan LAIN. Memakai angka satu lingkungan untuk
-- menilai lingkungan lain itulah yang membuat peringatan ini bertahan lebih lama daripada
-- seharusnya.
--
-- YANG MASIH TERBUKA, dan ia bukan pekerjaan kode: backfill itu mengisi SEKALI. Pega masih
-- membuat klaim baru (`P-3`), aplikasi Go menulis `USERTEKNIS_1`, dan catatan migrasi menyebut
-- ada proses pengisi lain. Tiga penulis atas satu tabel melanggar `P-1`, dan akibatnya bukan
-- galat melainkan data yang saling menimpa.

UNDEFINE AKUN
