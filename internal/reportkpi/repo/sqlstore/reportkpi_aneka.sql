-- Kueri yang dijalankan di KONEKSI KEDUA portal (`ANEKA_<PORTAL_ALIAS>_*`), bukan di
-- basis data portalnya sendiri.
--
-- # Kenapa berkas ini terpisah
--
-- Objek di sini TIDAK ADA di basis data portal. Di Pega ia dicapai lewat DB Link
-- `@ASMD.SINARMAS.CO.ID`, dan `D-25`/`R-03` menggantinya dengan koneksi kedua selama API
-- penggantinya belum ada.
--
-- Pemisahan berkas inilah yang membuat kesalahan sambungan tidak dapat terjadi diam-diam:
-- kueri di berkas ini TIDAK BOLEH dijalankan di `r.db`, dan kueri di berkas lain tidak
-- boleh dijalankan di `r.aneka`. Menaruhnya di satu berkas berarti membedakan keduanya
-- hanya dari ingatan — dan itu persis kesalahan yang berkas ini lahir untuk menutupnya.
--
-- # Cacat yang melahirkannya (2026-10-08)
--
-- Kueri `holidays` semula berada di `reportkpi_picteknik.sql` dan dijalankan di koneksi
-- POOLDATA sebagai `FROM GENERAL.HRD_LBR` — tanpa akhiran DB Link dan tanpa koneksi
-- kedua. Objek itu tidak ada di sana, sehingga Oracle menolaknya dan SELURUH tab KPI PIC
-- Teknik gagal dengan "Terjadi kesalahan pada sistem".
--
-- Modul Report Klaim sudah melakukannya dengan benar sejak awal
-- (`reportklaim_aneka.sql`); yang keliru hanya modul ini, dan ketidakkonsistenan itulah
-- yang tidak terlihat selama keduanya ditulis di tempat yang berbeda.
--
-- Satu penyesuaian terhadap kueri Pega, dan hanya satu: **akhiran DB Link dihapus**,
-- karena koneksinya sendiri sudah menunjuk basis data itu. Isi predikatnya tidak diubah.

-- name: holidays
-- Hari libur pada satu rentang, DI LUAR akhir pekan. Meniru `CheckHoliday_SQL`.
--
-- Akhir pekan dikecualikan di sini, bukan di Go, dan itu disengaja: begitulah kueri lama
-- melakukannya, sehingga libur yang jatuh pada Sabtu atau Minggu tidak terpotong dua kali.
--
-- Nama harinya dibandingkan dalam dua bahasa karena `TO_CHAR(...,'DAY')` mengikuti setelan
-- bahasa sesi basis data — dan setelan itu tidak dijamin sama antar lingkungan.
SELECT TANGGAL
FROM GENERAL.HRD_LBR
WHERE TANGGAL >= :1
  AND TANGGAL < :2 + INTERVAL '1' DAY
  AND TRIM(TO_CHAR(TANGGAL, 'DAY')) NOT IN ('SABTU', 'MINGGU', 'SATURDAY', 'SUNDAY')
ORDER BY TANGGAL

-- name: probe_hari_libur
-- Probe `-periksa`: membuktikan kalender libur TERBACA di koneksi kedua.
--
-- `WHERE 1 = 0` membuatnya tidak membaca satu baris pun; yang diperiksa adalah
-- keberadaan objek, hak SELECT, dan nama kolomnya.
SELECT TANGGAL
FROM GENERAL.HRD_LBR
WHERE 1 = 0
