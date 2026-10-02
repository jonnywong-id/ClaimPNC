-- Tabel baru: daftar "Tanggal terima Dokumen" pada layar kerja RCL/PUCL.
--
-- Diminta 2026-10-02 oleh tim pengembang, menempuh `D-63`:
--   permintaan tertulis (berkas ini) -> persetujuan Work Owner -> pelaksanaan DBA
--   -> verifikasi dengan menjalankan Pega dan Go bersamaan.
--
-- ============================================================================
-- KENAPA TABEL BARU, BUKAN KOLOM YANG SUDAH ADA
-- ============================================================================
--
-- Grid "Tanggal terima Dokumen" di layar lama adalah DAFTAR — page list
-- `.ClaimData.PUCLStatus.DateReceivedDocument` — dan Pega menyimpannya di dalam
-- objek kerja. Hanya BARIS PERTAMA yang diekspos sebagai kolom:
--
--     DATAPEGA.PC_ASM_FW_GCNMFW_WORK.RECEIVEDDATE_1
--     DATAPEGA.PC_ASM_FW_GCNMFW_WORK.KETERANGAN_1
--
-- Keduanya milik Pega (`P-1`), dan keduanya hanya menampung SATU baris.
--
-- `POOLDATA.TC_PNC_PUCL.TGL_TERIMA_DOKUMEN_PUCL` BUKAN tempatnya: kolom itu
-- memegang isian "Tanggal Kelengkapan Dokumen"
-- (`.ClaimData.PUCLStatus.TanggalTerimaDokumenPUCL`), satu nilai tunggal yang
-- digambar di bawah grid — bukan daftarnya.
--
-- ============================================================================
-- APA YANG INI MEMUNGKINKAN
-- ============================================================================
--
-- Tombol "✚ Tambah" dan "Hapus". Keduanya di Pega murni operasi sisi klien —
-- terverifikasi: `pyAction addRow` tanpa satu pun `pyActivity`, dan `deleteRow`
-- tidak ada sama sekali di section. Yang MENYIMPANNYA adalah tombol "Save",
-- lewat `Obj-Save` pada objek kerja.
--
-- Tanpa tabel ini, baris yang ditambah petugas hilang begitu layar dimuat ulang —
-- dan kehilangan data yang terlihat seperti berhasil lebih buruk daripada tombol
-- yang jelas belum ada.
--
-- ============================================================================

CREATE TABLE POOLDATA.TC_PNC_PUCL_TERIMA_DOKUMEN (
  CLAIMID      VARCHAR2(50)   NOT NULL,
  URUTAN       NUMBER(3)      NOT NULL,
  TGL_TERIMA   TIMESTAMP(6),
  KETERANGAN   VARCHAR2(500),

  -- Jejak minimum. `TC_PNC_PUCL` sendiri tidak punya kolom pelaku, sehingga
  -- modul ini mencatat pelakunya di log. Pada tabel BARU, mencatatnya di baris
  -- lebih murah daripada menelusuri log — dan `D-59` menjadikan jejak audit
  -- satu-satunya kontrol pengimbang.
  DIUBAH_OLEH  VARCHAR2(64),
  DIUBAH_PADA  TIMESTAMP(6)   DEFAULT SYSTIMESTAMP,

  CONSTRAINT PK_TC_PNC_PUCL_TERIMA_DOKUMEN PRIMARY KEY (CLAIMID, URUTAN)
);

-- Pengambilan selalu per klaim, berurutan. Kunci utamanya sudah melayani itu,
-- sehingga TIDAK ada index tambahan — index yang tidak dipakai tetap dibayar
-- setiap kali baris ditulis.

COMMENT ON TABLE  POOLDATA.TC_PNC_PUCL_TERIMA_DOKUMEN IS
  'Daftar tanggal terima dokumen RCL/PUCL. Pengganti page list DateReceivedDocument.';
COMMENT ON COLUMN POOLDATA.TC_PNC_PUCL_TERIMA_DOKUMEN.URUTAN IS
  'Urutan baris dalam daftar, mulai 1. Setara indeks page list di Pega.';

-- ============================================================================
-- ROLLBACK
-- ============================================================================
--
-- `P-4` mewajibkan setiap perubahan skema punya jalan mundur yang benar-benar
-- berfungsi. Tabel ini BARU dan tidak dibaca sistem mana pun selain modul Inbox
-- RCL/PUCL, sehingga membuangnya tidak menyentuh Pega:
--
--   DROP TABLE POOLDATA.TC_PNC_PUCL_TERIMA_DOKUMEN;
--
-- Modul ini menggambar grid-nya KOSONG bila tabelnya tidak ada — bukan gagal.
