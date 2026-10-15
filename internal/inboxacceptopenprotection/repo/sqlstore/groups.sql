-- Kueri kewenangan modul Inbox Accept Open Protection.
--
-- ============================================================================
-- TABEL INI MILIK BASIS DATA UTAMA, BUKAN BASIS DATA PORTAL
-- ============================================================================
--
-- `D-78` menetapkan satu identitas berlaku di keempat portal, sehingga keanggotaan group
-- tinggal bersama identitas — bukan di dalam basis data tiap entitas. Modul `menu` pun
-- membaca tabel yang sama dari koneksi utama.
--
-- Repo ini karena itu TIDAK dipasang lewat pemilih portal. Memasangnya per portal akan
-- membuat kewenangan seseorang berubah-ubah mengikuti entitas yang sedang dibuka — dan
-- perubahannya tidak menghasilkan galat, hanya layar yang kadang terbuka kadang tidak.
--
-- ============================================================================
-- ISI GROUP_ID
-- ============================================================================
--
-- Nama access group Pega TANPA awalan `GCNMFW:` — Work Owner, 2026-09-25. Jadi
-- `GCNMFW:PncCollection` tersimpan sebagai `PncCollection`.
--
-- Perbandingannya dilakukan DI GO, bukan di sini: `D-58` mencatat kapitalisasi Pega tidak
-- konsisten, dan normalisasinya perlu membuang awalan sekaligus menyeragamkan huruf. Kueri
-- ini mengembalikan nilai apa adanya supaya yang tersimpan tetap terbaca saat ditelusuri.


-- name: groups_of_login
-- Access group yang diikuti sebuah login.
--
-- Dicocokkan dengan `UPPER(TRIM(...))` pada kedua sisi, sama dengan modul `menu`: kolomnya
-- diisi manusia, dan spasi di ujung tidak boleh menghapus kewenangan seseorang.
SELECT g.GROUP_ID
  FROM POOLDATA.M_LOGIN_GROUP_PNC g
 WHERE UPPER(TRIM(g.LOGIN_ID)) = UPPER(TRIM(:1))
 ORDER BY g.GROUP_ID
