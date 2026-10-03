-- Kueri yang dijalankan pada KONEKSI KEDUA portal, bukan pada basis data portalnya.
--
-- ============================================================================
-- Kenapa berkas ini terpisah
-- ============================================================================
--
-- Karena kueri di sini berjalan pada koneksi yang BERBEDA — `ANEKA_<PORTAL_ALIAS>_*`,
-- pengganti DB Link `@ASMD` yang `R-03` belum sediakan API-nya (keputusan Work Owner
-- 2026-09-24, keputusan-implementasi.md §49).
--
-- Menaruhnya berdampingan dengan kueri portal akan membuat siapa pun yang membacanya
-- mengira ia dapat dijalankan pada koneksi yang sama. Ia tidak — tabelnya tidak ada di
-- sana, dan galatnya baru muncul saat laporan dijalankan.
--
-- ============================================================================
-- Aturan yang berlaku sama seperti berkas kueri lain
-- ============================================================================
--
-- Kolom disebut namanya, nilai selalu lewat parameter binding, dan tanpa NVL, SYSDATE,
-- DECODE, ROWNUM, maupun TO_CHAR untuk tampilan.


-- name: report_holiday_calendar
--
-- Daftar hari libur perusahaan pada satu rentang tanggal.
--
-- Asal: `RDB List/CheckHoliday_SQL-SQL.xml`, yang dijalankan
-- `Activity/GCNMTimeDifferenceWorkCalender_Act-Act.xml` untuk menghitung selisih hari
-- kerja — dasar keempat kolom "Lama proses" pada laporan TAT.
--
-- Bind:
--   :1  tanggal dari    DATE
--   :2  tanggal sampai  DATE
--
-- # Tiga perbedaan dari kueri asli, seluruhnya disengaja
--
--  1. **Tanpa akhiran DB Link.** `general.hrd_lbr@ASMD.SINARMAS.CO.ID` menjadi
--     `general.hrd_lbr`, karena koneksinya sendiri yang sudah menunjuk basis data itu.
--
--  2. **Mengembalikan TANGGALNYA, bukan JUMLAHNYA.** Kueri asli menghitung
--     `COUNT(1)` di basis data. Di sini daftar tanggalnya yang dikembalikan, dan yang
--     menghitung adalah `reportklaim.WorkingDaysBetween` — sesuai `D-50` yang menetapkan
--     perhitungan hari kerja ditulis ulang di Go karena ia aturan bisnis.
--
--     Akibat baiknya: satu pembacaan melayani SELURUH baris laporan. Kueri asli
--     dijalankan sekali per klaim per pasangan tanggal — empat kali per baris.
--
--  3. **Tanpa penyaring hari Sabtu/Minggu.** Kueri asli membuangnya dengan
--     `TRIM(TO_CHAR(tanggal,'DAY')) NOT IN ('SABTU','MINGGU','SATURDAY','SUNDAY')` supaya
--     hari libur yang jatuh pada akhir pekan tidak dikurangkan dua kali. Penyaring itu
--     bergantung pada bahasa NLS server — ia menyebut nama hari dalam dua bahasa justru
--     karena pengaturannya tidak pasti.
--
--     Di sini pencegahan penghitungan gandanya ada di perhitungannya sendiri: sebuah hari
--     dihitung bila ia bukan akhir pekan DAN bukan hari libur, sehingga tidak ada yang
--     dapat terkurang dua kali. Lihat reportklaim.WorkingDaysBetween.
SELECT tanggal
  FROM general.hrd_lbr
 WHERE tanggal >= :1
   AND tanggal <= :2
 ORDER BY tanggal


-- name: report_mitra_logins
--
-- Daftar login petugas yang terdaftar sebagai MITRA.
--
-- Asal: gabungan `general.lst_mitra@asmd.sinarmas.co.id` pada
-- `RDB List/ExportDetailMitraReport-SQL.xml`.
--
-- Tanpa bind.
--
-- # Ia PENYARING, bukan pelengkap kolom
--
-- Di kueri asli, tabel ini digabung INNER JOIN tanpa satu pun kolomnya diambil:
-- `a.userassign = b.login_aplikasi`. Gabungan itu membatasi laporan pada petugas mitra.
--
-- Karena itu daftar ini TIDAK boleh diperlakukan seperti kolom bersumber DB Link di
-- modul lain, yang cukup dikosongkan bila koneksinya tidak ada. Bila daftar ini gagal
-- dibaca, laporannya DITOLAK — lihat Repo.mitraLogins. Menjalankannya tanpa penyaring
-- akan memasukkan seluruh petugas ke laporan produktivitas mitra, dan berkasnya tetap
-- terlihat wajar.
--
-- # Tanpa akhiran DB Link
--
-- `general.lst_mitra@asmd.sinarmas.co.id` menjadi `general.lst_mitra`, karena koneksi
-- inilah yang sudah menunjuk basis data itu.
SELECT login_aplikasi
  FROM general.lst_mitra
 WHERE login_aplikasi IS NOT NULL
