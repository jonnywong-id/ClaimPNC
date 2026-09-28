-- Kueri penopang alur: nomor klaim, jejak audit, dan kotak keluar pemberitahuan.

-- name: nomor_terakhir_tahun
-- Nomor urut tertinggi yang sudah terbit pada tahun yang diminta.
--
-- # Kenapa BUKAN tabel pencacah ber-FOR UPDATE
--
-- Rancangan `0002` memakai CPNC_NOMOR_KLAIM dengan `FOR UPDATE`, dan itu memang
-- penjagaan yang lebih kuat. Work Owner menetapkan 2026-09-24 bahwa tabel itu TIDAK
-- dibuat: nomor diturunkan dari isi tabel klaim, pola yang sudah berjalan untuk
-- `RCVN.YY.xxxx` pada modul Inbox Laporan Klaim.
--
-- # Kenapa aman dibaca sebagai teks
--
-- Nomor berbentuk `PNCN.YY.xxxx` dengan lebar tetap dan dipadatkan nol, sehingga urutan
-- teks sama dengan urutan angka DI DALAM SATU TAHUN. Penyaring membatasi pada tahun yang
-- diminta, jadi pergantian tahun tidak membuat deretnya melompat.
--
-- # Yang TIDAK dijamin kueri ini
--
-- Dua pendaftaran bersamaan dapat membaca angka yang sama. Yang menjaganya bukan kueri
-- ini melainkan penyisipan yang gagal lalu diulang dengan nomor berikutnya — lihat
-- NumberIssuer.Issue. Penjagaan itu lebih lemah daripada `FOR UPDATE`, dan itu harga yang
-- dibayar untuk berjalan tanpa tabel tambahan.
SELECT COALESCE(MAX(TO_NUMBER(SUBSTR(CLAIMNO, 9))), 0)
  FROM POOLDATA.T_CLAIM_PNC
 WHERE CLAIMNO LIKE :1

-- name: audit_sisip
-- Jejak audit hanya bertambah (ADR-0026). Tidak ada UPDATE maupun DELETE atas tabel ini
-- di seluruh aplikasi.
INSERT INTO CPNC_JEJAK_AUDIT (ID, KLAIM_ID, NOMOR_KLAIM, PERISTIWA, PELAKU, PADA, KETERANGAN)
VALUES (:1, :2, :3, :4, :5, :6, :7)

-- name: notifikasi_sisip
-- Kotak keluar. Baris disisipkan di dalam transaksi yang sama dengan penyimpanan klaim,
-- sehingga satu klaim yang melampaui ambang menerbitkan tepat satu peristiwa.
-- Pengirimannya milik S-3, yang membaca tabel ini di luar transaksi.
INSERT INTO POOLDATA.CPNC_NOTIFIKASI
       (ID, JENIS, NOMOR_KLAIM, NOMOR_POLIS, PENERIMA, NILAI_SEN, REVISI, DIBUAT_PADA)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8)
