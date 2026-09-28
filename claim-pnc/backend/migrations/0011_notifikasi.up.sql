-- 0011 — kotak keluar pemberitahuan
--
-- # Kenapa tabel ini AKHIRNYA dibuat
--
-- Ia sempat ditunda dengan alasan pengirimannya milik `S-3`. Alasan itu benar untuk
-- PENGIRIMAN, tetapi salah untuk PENCATATAN, dan menundanya meninggalkan cacat yang
-- ditemukan 2026-09-24: modul registrasi merakit pengisi SQL untuk seam Notifier,
-- sedangkan tabelnya tidak ada. Akibatnya klaim yang melampaui Rp 1.000.000.000 akan
-- menabrak ORA-00942 DI DALAM transaksi, dan seluruh pendaftarannya dibatalkan.
--
-- Menggantinya dengan pencatat di memori tidak menutup cacat itu, hanya memindahkannya:
-- peristiwanya hilang saat aplikasi berhenti, dan janji `TKT-B02-004` — klaim yang
-- melampaui ambang menerbitkan TEPAT SATU peristiwa — menjadi janji yang tidak dijaga
-- apa pun.
--
-- # Ia KOTAK KELUAR, bukan pengirim
--
-- Baris disisipkan di dalam transaksi yang sama dengan penyimpanan klaim, sehingga satu
-- klaim menerbitkan tepat satu peristiwa: tidak nol karena surel gagal, tidak dua karena
-- permintaan diulang. Pengirimannya dikerjakan `S-3`, yang membaca tabel ini DI LUAR
-- transaksi dan mengisi DIKIRIM_PADA.
--
-- # Dijalankan EMPAT KALI — sekali per portal (D-75)
--
-- Blok ini idempoten: dijalankan dua kali tidak menghasilkan galat.

DECLARE
  jumlah NUMBER;
BEGIN
  SELECT COUNT(*) INTO jumlah
    FROM ALL_TABLES
   WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'CPNC_NOTIFIKASI';

  IF jumlah = 0 THEN
    EXECUTE IMMEDIATE '
      CREATE TABLE POOLDATA.CPNC_NOTIFIKASI (
        ID           VARCHAR2(32)  NOT NULL,
        JENIS        VARCHAR2(64)  NOT NULL,
        NOMOR_KLAIM  VARCHAR2(32),
        NOMOR_POLIS  VARCHAR2(64),
        PENERIMA     VARCHAR2(2000),
        NILAI_SEN    NUMBER(20)    DEFAULT 0 NOT NULL,
        REVISI       VARCHAR2(1),
        DIBUAT_PADA  TIMESTAMP     NOT NULL,
        DIKIRIM_PADA TIMESTAMP,
        CONSTRAINT PK_CPNC_NOTIFIKASI PRIMARY KEY (ID)
      )';
  END IF;
END;
