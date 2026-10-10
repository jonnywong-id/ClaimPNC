-- ============================================================================
-- PERMINTAAN DBA — objek basis data untuk modul Inbox Compliance
-- ============================================================================
--
-- Tanggal      : 2026-10-07
-- Untuk        : DBA
-- Dari         : tim pengembang Claim PNC
-- Menempuh     : `D-63` — permintaan tertulis -> persetujuan Work Owner -> pelaksanaan DBA
--
-- ----------------------------------------------------------------------------
-- BACA DULU: BERKAS INI SALINAN, BUKAN SUMBERNYA
-- ----------------------------------------------------------------------------
--
-- Isinya disalin dari dua berkas migrasi di `claim-pnc/backend/migrations/`, yang tetap
-- menjadi sumber kebenaran:
--
--     0011_post_audit_compliance.up.sql
--     0012_keputusan_compliance.up.sql
--     --     --
-- Berkas ini ada semata supaya DBA menjalankan SATU hal, bukan empat. Catatan panjang yang
-- menjelaskan SEBAB setiap kolom ada di keempat berkas asli — yang di sini hanya
-- pernyataannya.
--
-- Bila kedua berkas itu berubah, berkas ini TIDAK ikut berubah sendiri. Sebelum
-- menjalankannya, pastikan ia masih sepadan.
--
-- ----------------------------------------------------------------------------
-- KENAPA INI DIBUTUHKAN
-- ----------------------------------------------------------------------------
--
-- Tanpa objek-objek ini, form Compliance Checker **tidak dapat dibuka sama sekali** —
-- aplikasi menjawab 503 dengan pesan yang menyebut nomor migrasinya. Itu bukan cacat kode;
-- itu memang keadaan yang dilaporkan apa adanya.
--
-- ----------------------------------------------------------------------------
-- DI MANA DIJALANKAN
-- ----------------------------------------------------------------------------
--
-- Di **basis data SETIAP entitas** (`D-75` — satu basis data per badan hukum). Entitas yang
-- terlewat akan gagal di entitas itu saja, dengan galat yang menyebut objeknya tidak ada.
--
-- ----------------------------------------------------------------------------
-- TENTANG GRANT — tidak ada satu pun di berkas ini, dan itu disengaja
-- ----------------------------------------------------------------------------
--
-- Menurut `claim-pnc/backend/.env` baris 60, aplikasi menyambung sebagai **POOLDATA**,
-- yakni pemilik skemanya sendiri. Pemilik skema selalu punya hak penuh atas objeknya, jadi
-- tidak ada hak yang perlu diberikan.
--
-- Menjalankan GRANT ke `APP_CLAIM_PNC` justru akan GAGAL dengan `ORA-01917` karena akun itu
-- tidak ada — dan satu pernyataan gagal dapat menghentikan skrip ini di tengah, meninggalkan
-- sebagian objek terbuat dan sebagian tidak.
--
-- Bila kelak aplikasi memakai akun terpisah, GRANT-nya ada (terkomentari) di keempat berkas
-- migrasi asli, lengkap dengan alasan tiap haknya.
--
-- ============================================================================


-- ----------------------------------------------------------------------------
-- 1 dari 2 — sequence penomoran Post Audit
-- ----------------------------------------------------------------------------
--
-- Menerbitkan nomor berbentuk `CPL.26.1`, sesuai sintaks yang ditetapkan Work Owner
-- 2026-10-06. Ejaan `COMPLIENCE` disengaja — ejaan Work Owner, sejalan dengan Pega yang
-- access group-nya pun bernama `PncComplience`.
--
-- NOCACHE supaya tidak ada nomor yang hilang saat instans dimatikan.

CREATE SEQUENCE POOLDATA.CLAIM_COMPLIENCE_SEQ
    START WITH 1
    INCREMENT BY 1
    NOCACHE
    NOCYCLE;


-- ----------------------------------------------------------------------------
-- 2 dari 2 — keputusan Compliance
-- ----------------------------------------------------------------------------
--
-- Satu baris per klaim. Form menyimpan keputusan dan komentarnya dalam satu tombol,
-- sehingga tabel ini BERPASANGAN dengan nomor 3 di bawah — menjalankan salah satu saja
-- membuat tombol Simpan gagal separuh jalan.

CREATE TABLE POOLDATA.CPNC_KEPUTUSAN_COMPLIANCE (
    NO_KLAIM              VARCHAR2(255 BYTE) NOT NULL,
    PILIHAN               VARCHAR2(1 BYTE)   NOT NULL,
    NOTE                  VARCHAR2(4000 BYTE),
    CATATAN               VARCHAR2(4000 BYTE),
    DIPUTUSKAN_OLEH       VARCHAR2(128 BYTE),
    DIPUTUSKAN_PADA       TIMESTAMP(6)       NOT NULL,
    TGL_VALID             TIMESTAMP(6),
    TGL_KIRIM_POST_AUDIT  TIMESTAMP(6),
    -- Grid komentar, satu kolom JSON. Go yang mengurainya; basis data hanya memvalidasi
    -- bentuknya. Lihat 0012_keputusan_compliance.up.sql untuk alasan lengkapnya.
    KOMENTAR_JSON         CLOB,
    CONSTRAINT CK_CPNC_KEPUTUSAN_KOMENTAR CHECK (KOMENTAR_JSON IS JSON),
    CONSTRAINT PK_CPNC_KEPUTUSAN_COMPLIANCE PRIMARY KEY (NO_KLAIM)
);


-- ============================================================================
-- PEMERIKSAAN SETELAH DIJALANKAN
-- ============================================================================
--
-- Keduanya harus muncul. Bila ada yang kurang, aplikasi akan menyebutnya sendiri saat
-- dijalankan dengan flag `-periksa`.

SELECT object_name, object_type, status
  FROM all_objects
 WHERE owner = 'POOLDATA'
   AND object_name IN ('CLAIM_COMPLIENCE_SEQ',
                       'CPNC_KEPUTUSAN_COMPLIANCE',,)
 ORDER BY object_type, object_name;


-- ============================================================================
-- SATU TABEL YANG TIDAK DIBUAT DI SINI, TETAPI WAJIB ADA
-- ============================================================================
--
-- `POOLDATA.T_CLAIM_COMPLIANCE_H` — tab Post Audit membacanya dan menulis ke sana. Ia
-- dibuat di luar repositori ini, sehingga tidak ada pernyataan CREATE untuknya di sini.
--
-- Yang dituntut aplikasi darinya, tepat enam kolom:
--
--     CASEID · NO_KLAIM · NAMA_TERTANGGUNG · NO_POLIS · REMARKS · TGL_KIRIM_POST_AUDIT
--
-- Periksa dengan:

SELECT column_name, data_type, data_length, nullable
  FROM all_tab_columns
 WHERE owner = 'POOLDATA'
   AND table_name = 'T_CLAIM_COMPLIANCE_H'
 ORDER BY column_id;
