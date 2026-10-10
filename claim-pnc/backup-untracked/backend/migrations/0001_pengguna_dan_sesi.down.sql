-- 0001 down — mengembalikan skema ke keadaan sebelum 0001 dijalankan.
--
-- Konsekuensi yang harus diumumkan sebelum dijalankan pada jam kerja: seluruh sesi
-- menjadi tidak sah dan semua pengguna harus masuk ulang.

DROP INDEX IX_CPNC_SESI_AKTIF_IDENT;
DROP TABLE CPNC_SESI_AKTIF;
DROP INDEX IX_CPNC_PENGGUNA_LOGIN;
DROP TABLE CPNC_PENGGUNA;
