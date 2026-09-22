-- Kueri tabel CPNC_KLAIM dan pohon objek–coverage–spreading di bawahnya.
--
-- TIDAK ADA satu pun DELETE di berkas ini. Baris anak yang tidak lagi terpakai ditandai
-- lewat DIHAPUS_PADA (ADR-0012), dan penyimpanan ulang memakai UPDATE-lalu-INSERT —
-- bukan MERGE, mengikuti keputusan 2.9 pada docs/keputusan-implementasi.md.

-- name: klaim_perbarui
UPDATE CPNC_KLAIM
   SET NOMOR                  = :1,
       PORTAL                 = :2,
       POLIS_NOMOR            = :3,
       POLIS_LINI             = :4,
       POLIS_JENIS_BISNIS     = :5,
       POLIS_MULAI            = :6,
       POLIS_AKHIR            = :7,
       POLIS_DEKLARASI        = :8,
       POLIS_MATA_UANG        = :9,
       POLIS_PENJAMIN_KREDIT  = :10,
       POLIS_TERTANGGUNG      = :11,
       POLIS_KODE_CABANG      = :12,
       TANGGAL_KEJADIAN       = :13,
       TANGGAL_LAPOR          = :14,
       TANGGAL_TERIMA_DOKUMEN = :15,
       LOKASI                 = :16,
       KRONOLOGI              = :17,
       PELAPOR_NAMA           = :18,
       PELAPOR_TELEPON        = :19,
       PELAPOR_EMAIL          = :20,
       PELAPOR_ALAMAT         = :21,
       PELAPOR_HUBUNGAN       = :22,
       PELAPOR_HUBUNGAN_LAIN  = :23,
       NILAI_ESTIMASI_SEN     = :24,
       MATA_UANG              = :25,
       NOMOR_SLIK             = :26,
       EX_GRATIA              = :27,
       USER_TEKNIS            = :28,
       RCV_ID                 = :29,
       STATUS_PUCL            = :30,
       TRANSFER_COMPLIANCE    = :31,
       MINTA_KEMBALI          = :32,
       STATUS_PROSES          = :33,
       STATUS_KLAIM           = :34,
       FLAG_KLAIM             = :35,
       STATUS_POSISI_PROGRES  = :36,
       TAHAP_KINI             = :37,
       DIUBAH_OLEH            = :38,
       DIUBAH_PADA            = :39,
       DIHAPUS_PADA           = :40
 WHERE ID = :41

-- name: klaim_sisip
INSERT INTO CPNC_KLAIM (
       NOMOR, PORTAL,
       POLIS_NOMOR, POLIS_LINI, POLIS_JENIS_BISNIS, POLIS_MULAI, POLIS_AKHIR,
       POLIS_DEKLARASI, POLIS_MATA_UANG, POLIS_PENJAMIN_KREDIT, POLIS_TERTANGGUNG,
       POLIS_KODE_CABANG,
       TANGGAL_KEJADIAN, TANGGAL_LAPOR, TANGGAL_TERIMA_DOKUMEN,
       LOKASI, KRONOLOGI,
       PELAPOR_NAMA, PELAPOR_TELEPON, PELAPOR_EMAIL, PELAPOR_ALAMAT,
       PELAPOR_HUBUNGAN, PELAPOR_HUBUNGAN_LAIN,
       NILAI_ESTIMASI_SEN, MATA_UANG, NOMOR_SLIK, EX_GRATIA, USER_TEKNIS, RCV_ID,
       STATUS_PUCL, TRANSFER_COMPLIANCE, MINTA_KEMBALI,
       STATUS_PROSES, STATUS_KLAIM, FLAG_KLAIM, STATUS_POSISI_PROGRES,
       TAHAP_KINI,
       DIUBAH_OLEH, DIUBAH_PADA, DIHAPUS_PADA,
       ID, DIBUAT_OLEH, DIBUAT_PADA)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10,
        :11, :12, :13, :14, :15, :16, :17, :18, :19, :20,
        :21, :22, :23, :24, :25, :26, :27, :28, :29, :30,
        :31, :32, :33, :34, :35, :36, :37, :38, :39, :40,
        :41, :42, :43)

-- name: klaim_ambil
SELECT ID, NOMOR, PORTAL,
       POLIS_NOMOR, POLIS_LINI, POLIS_JENIS_BISNIS, POLIS_MULAI, POLIS_AKHIR,
       POLIS_DEKLARASI, POLIS_MATA_UANG, POLIS_PENJAMIN_KREDIT, POLIS_TERTANGGUNG,
       POLIS_KODE_CABANG,
       TANGGAL_KEJADIAN, TANGGAL_LAPOR, TANGGAL_TERIMA_DOKUMEN,
       LOKASI, KRONOLOGI,
       PELAPOR_NAMA, PELAPOR_TELEPON, PELAPOR_EMAIL, PELAPOR_ALAMAT,
       PELAPOR_HUBUNGAN, PELAPOR_HUBUNGAN_LAIN,
       NILAI_ESTIMASI_SEN, MATA_UANG, NOMOR_SLIK, EX_GRATIA, USER_TEKNIS, RCV_ID,
       STATUS_PUCL, TRANSFER_COMPLIANCE, MINTA_KEMBALI,
       STATUS_PROSES, STATUS_KLAIM, FLAG_KLAIM, STATUS_POSISI_PROGRES,
       TAHAP_KINI,
       DIBUAT_OLEH, DIBUAT_PADA, DIUBAH_OLEH, DIUBAH_PADA, DIHAPUS_PADA
  FROM CPNC_KLAIM
 WHERE ID = :1

-- name: klaim_ambil_per_nomor
SELECT ID, NOMOR, PORTAL,
       POLIS_NOMOR, POLIS_LINI, POLIS_JENIS_BISNIS, POLIS_MULAI, POLIS_AKHIR,
       POLIS_DEKLARASI, POLIS_MATA_UANG, POLIS_PENJAMIN_KREDIT, POLIS_TERTANGGUNG,
       POLIS_KODE_CABANG,
       TANGGAL_KEJADIAN, TANGGAL_LAPOR, TANGGAL_TERIMA_DOKUMEN,
       LOKASI, KRONOLOGI,
       PELAPOR_NAMA, PELAPOR_TELEPON, PELAPOR_EMAIL, PELAPOR_ALAMAT,
       PELAPOR_HUBUNGAN, PELAPOR_HUBUNGAN_LAIN,
       NILAI_ESTIMASI_SEN, MATA_UANG, NOMOR_SLIK, EX_GRATIA, USER_TEKNIS, RCV_ID,
       STATUS_PUCL, TRANSFER_COMPLIANCE, MINTA_KEMBALI,
       STATUS_PROSES, STATUS_KLAIM, FLAG_KLAIM, STATUS_POSISI_PROGRES,
       TAHAP_KINI,
       DIBUAT_OLEH, DIBUAT_PADA, DIUBAH_OLEH, DIUBAH_PADA, DIHAPUS_PADA
  FROM CPNC_KLAIM
 WHERE NOMOR = :1

-- name: objek_perbarui
UPDATE CPNC_KLAIM_OBJEK
   SET OBJEK_ID = :1, NAMA = :2, LOKASI = :3, DIHAPUS_PADA = NULL
 WHERE KLAIM_ID = :4 AND URUTAN = :5

-- name: objek_sisip
INSERT INTO CPNC_KLAIM_OBJEK (OBJEK_ID, NAMA, LOKASI, KLAIM_ID, URUTAN)
VALUES (:1, :2, :3, :4, :5)

-- name: objek_tandai_sisa
UPDATE CPNC_KLAIM_OBJEK
   SET DIHAPUS_PADA = :1
 WHERE KLAIM_ID = :2 AND URUTAN > :3 AND DIHAPUS_PADA IS NULL

-- name: objek_daftar
SELECT URUTAN, OBJEK_ID, NAMA, LOKASI
  FROM CPNC_KLAIM_OBJEK
 WHERE KLAIM_ID = :1 AND DIHAPUS_PADA IS NULL
 ORDER BY URUTAN

-- name: coverage_perbarui
UPDATE CPNC_KLAIM_COVERAGE
   SET COVERAGE_ID = :1, PENYEBAB_KERUGIAN = :2, TSI_SEN = :3, DIHAPUS_PADA = NULL
 WHERE KLAIM_ID = :4 AND URUTAN_OBJEK = :5 AND URUTAN = :6

-- name: coverage_sisip
INSERT INTO CPNC_KLAIM_COVERAGE (COVERAGE_ID, PENYEBAB_KERUGIAN, TSI_SEN, KLAIM_ID, URUTAN_OBJEK, URUTAN)
VALUES (:1, :2, :3, :4, :5, :6)

-- name: coverage_tandai_sisa
UPDATE CPNC_KLAIM_COVERAGE
   SET DIHAPUS_PADA = :1
 WHERE KLAIM_ID = :2 AND URUTAN_OBJEK = :3 AND URUTAN > :4 AND DIHAPUS_PADA IS NULL

-- name: coverage_daftar
SELECT URUTAN_OBJEK, URUTAN, COVERAGE_ID, PENYEBAB_KERUGIAN, TSI_SEN
  FROM CPNC_KLAIM_COVERAGE
 WHERE KLAIM_ID = :1 AND DIHAPUS_PADA IS NULL
 ORDER BY URUTAN_OBJEK, URUTAN

-- name: spreading_perbarui
UPDATE CPNC_KLAIM_SPREADING
   SET JENIS_TREATY = :1, NAMA = :2, SHARE_E4 = :3, DIHAPUS = :4,
       OBJEK_FAC_OFFER = :5, DIHAPUS_PADA = NULL
 WHERE KLAIM_ID = :6 AND URUTAN_OBJEK = :7 AND URUTAN_COVERAGE = :8 AND URUTAN = :9

-- name: spreading_sisip
INSERT INTO CPNC_KLAIM_SPREADING (JENIS_TREATY, NAMA, SHARE_E4, DIHAPUS, OBJEK_FAC_OFFER,
                                  KLAIM_ID, URUTAN_OBJEK, URUTAN_COVERAGE, URUTAN)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9)

-- name: spreading_tandai_sisa
UPDATE CPNC_KLAIM_SPREADING
   SET DIHAPUS_PADA = :1
 WHERE KLAIM_ID = :2 AND URUTAN_OBJEK = :3 AND URUTAN_COVERAGE = :4
   AND URUTAN > :5 AND DIHAPUS_PADA IS NULL

-- name: spreading_daftar
SELECT URUTAN_OBJEK, URUTAN_COVERAGE, URUTAN, JENIS_TREATY, NAMA, SHARE_E4, DIHAPUS, OBJEK_FAC_OFFER
  FROM CPNC_KLAIM_SPREADING
 WHERE KLAIM_ID = :1 AND DIHAPUS_PADA IS NULL
 ORDER BY URUTAN_OBJEK, URUTAN_COVERAGE, URUTAN

-- name: klaim_cari_ganda
-- Pemeriksaan klaim ganda.
--
-- Empat hal yang membuat kueri ini benar:
--   1. Klaim yang ditandai terhapus tidak ikut (ADR-0012), dan begitu pula baris objek
--      yang ditandai terhapus.
--   2. Klaim yang belum bernomor tidak ikut: ia belum benar-benar terdaftar.
--   3. Klaim yang sedang disimpan dikecualikan lewat :2, supaya penyimpanan ulang tidak
--      menganggap dirinya sendiri duplikat.
--   4. Lokasi dibandingkan setelah spasi dibuang dan huruf disamakan — meniru
--      `replace(upper(location_1),' ','')` pada InputRegister_act langkah 32.2.
--
-- Lokasi dan penyebab kerugian ikut diperiksa HANYA bila kuncinya menyebutkannya.
-- Sakelarnya adalah parameter angka (:4 dan :6), bukan potongan SQL yang dirangkai —
-- teks kueri ini sama persis pada setiap pemanggilan, sehingga basis data dapat memakai
-- ulang rencana eksekusinya dan tidak ada nilai yang pernah menyentuh teks SQL.
SELECT DISTINCT k.NOMOR, o.OBJEK_ID
  FROM CPNC_KLAIM k
  JOIN CPNC_KLAIM_OBJEK o ON o.KLAIM_ID = k.ID AND o.DIHAPUS_PADA IS NULL
 WHERE k.POLIS_NOMOR = :1
   AND k.ID <> :2
   AND k.NOMOR IS NOT NULL
   AND k.DIHAPUS_PADA IS NULL
   AND o.OBJEK_ID = :3
   AND (:4 = 0 OR REPLACE(UPPER(k.LOKASI), ' ', '') = REPLACE(UPPER(:5), ' ', ''))
   AND (:6 = 0 OR EXISTS (
         SELECT 1
           FROM CPNC_KLAIM_COVERAGE c
          WHERE c.KLAIM_ID = k.ID
            AND c.URUTAN_OBJEK = o.URUTAN
            AND c.DIHAPUS_PADA IS NULL
            AND c.PENYEBAB_KERUGIAN = :7))
 ORDER BY k.NOMOR, o.OBJEK_ID
