-- Kueri penopang alur: nomor klaim, jejak audit, dan kotak keluar pemberitahuan.

-- name: nomor_kunci_tahun
-- Mengunci baris pencacah tahun berjalan.
--
-- FOR UPDATE menahan pemanggil kedua sampai yang pertama selesai. Tanpa itu, dua
-- pendaftaran yang tiba bersamaan membaca angka yang sama dan menerbitkan NOMOR KLAIM
-- YANG SAMA — kegagalan yang tidak dapat diperbaiki setelah surat terkirim.
--
-- Kueri ini wajib dijalankan di dalam transaksi; di luar transaksi, kuncinya lepas
-- seketika dan tidak menjaga apa pun.
SELECT TERAKHIR
  FROM CPNC_NOMOR_KLAIM
 WHERE TAHUN = :1
   FOR UPDATE

-- name: nomor_mulai_tahun
INSERT INTO CPNC_NOMOR_KLAIM (TAHUN, TERAKHIR) VALUES (:1, 1)

-- name: nomor_naikkan
UPDATE CPNC_NOMOR_KLAIM
   SET TERAKHIR = TERAKHIR + 1
 WHERE TAHUN = :1

-- name: audit_sisip
-- Jejak audit hanya bertambah (ADR-0026). Tidak ada UPDATE maupun DELETE atas tabel ini
-- di seluruh aplikasi.
INSERT INTO CPNC_JEJAK_AUDIT (ID, KLAIM_ID, NOMOR_KLAIM, PERISTIWA, PELAKU, PADA, KETERANGAN)
VALUES (:1, :2, :3, :4, :5, :6, :7)

-- name: notifikasi_sisip
-- Kotak keluar. Baris disisipkan di dalam transaksi yang sama dengan penyimpanan klaim,
-- sehingga satu klaim yang melampaui ambang menerbitkan tepat satu peristiwa.
-- Pengirimannya milik S-3, yang membaca tabel ini di luar transaksi.
INSERT INTO CPNC_NOTIFIKASI (ID, JENIS, NOMOR_KLAIM, NOMOR_POLIS, PENERIMA, NILAI_SEN, DIBUAT_PADA)
VALUES (:1, :2, :3, :4, :5, :6, :7)
