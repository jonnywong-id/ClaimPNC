-- 0003 — Pembatalan tabel berkas laporan klaim (Oracle 19c)
--
-- Membuang tabel dan sequence yang dibuat 0003. Karena keduanya BARU dan tidak satu pun
-- rule Pega merujuknya, pembatalan ini tidak menyentuh sistem lama sama sekali.
--
-- YANG HILANG BILA INI DIJALANKAN: seluruh berkas laporan yang diterbitkan aplikasi ini.
-- Berkas warisan Pega tidak tersentuh — ia tidak pernah tinggal di sini.
--
-- Karena itu berkas ini hanya benar dijalankan ketika 0003 baru saja dipasang dan belum
-- ada satu berkas pun yang dibuat petugas. Setelah pemakaian nyata dimulai, membatalkan
-- migrasi ini berarti membuang data bisnis, dan itu menuntut keputusan tersendiri —
-- bukan sekadar menjalankan berkas down.
--
-- Index ikut terbuang bersama tabelnya; ia tidak perlu disebut sendiri.

DROP SEQUENCE CPNC_LAPORAN_KLAIM_SEQ;

DROP TABLE CPNC_LAPORAN_KLAIM;
