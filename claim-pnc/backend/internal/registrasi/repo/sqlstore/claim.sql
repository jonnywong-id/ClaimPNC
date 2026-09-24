-- Kueri klaim beserta pohon objek–coverage–spreading di bawahnya.
--
-- ============================================================================
-- TABEL MANA, DAN KENAPA BERUBAH
-- ============================================================================
--
-- Work Owner menetapkan 2026-09-24: baris klaim ditulis LANGSUNG ke tabel bisnis yang
-- sudah dipakai Pega, dan `PEGA_CONVERT_JSONKLAIM_PNC` tidak dijalankan lagi.
--
--   CPNC_KLAIM             -> POOLDATA.T_CLAIM_PNC
--   CPNC_KLAIM_OBJEK       -> POOLDATA.T_CLAIM_OBJECTLIST
--   CPNC_KLAIM_COVERAGE    -> POOLDATA.T_CLAIM_OBJECTCOVERAGE
--   CPNC_KLAIM_SPREADING   -> POOLDATA.T_CLAIM_SPREADING   (tabel BARU, migrasi 0008)
--
-- Sembilan belas kolom yang dimodelkan modul ini tidak ada di tabel warisan dan
-- ditambahkan migrasi `0007`; lima lagi pada tabel anak oleh `0008`. Tanpa itu,
-- pemindahan ini akan membuang data diam-diam — terberat `TAHAP_KINI`, yang menyimpan
-- tahap klaim di dalam Register_Flow.
--
-- ============================================================================
-- DUA HAL YANG HARUS DIBACA SEBELUM MENYUNTING BERKAS INI
-- ============================================================================
--
-- 1. URUTAN BIND TIDAK BOLEH BERGESER. Oracle mengikat menurut URUTAN KEMUNCULAN
--    penanda, bukan menurut nomor pada `:n`. `claim.go` menyusun argumennya dalam urutan
--    tertentu, dan menukar dua kolom di sini akan menukar dua NILAI tanpa satu pun galat.
--    Cacat itu pernah terjadi di modul lain proyek ini: lencana menyebut satu angka
--    sementara tabelnya kosong, dan tidak ada yang memberi tahu.
--
-- 2. UANG PUNYA DUA SATUAN DI BERKAS INI.
--    Domain menyimpan uang sebagai SEN (`ADR-0016`). `NILAI_ESTIMASI_SEN` menerima sen
--    apa adanya — kolomnya memang ditambahkan untuk itu. Tetapi `SUMTSI` adalah kolom
--    WARISAN yang dibaca Pega dan seluruh laporan lama sebagai RUPIAH — terverifikasi
--    2026-09-24: median 45.000.000 pada mata uang terbanyak, dan 765 baris berpecahan.
--    Karena itu TSI dibagi 100 saat ditulis dan dikalikan 100 saat dibaca. Menghapus
--    pembagian itu membuat setiap nilai 100x lebih besar di mata pembaca lama.
--
-- TIDAK ADA satu pun DELETE di berkas ini. Baris anak yang tidak lagi terpakai ditandai
-- lewat DIHAPUS_PADA (ADR-0012), dan penyimpanan ulang memakai UPDATE-lalu-INSERT —
-- bukan MERGE, mengikuti keputusan 2.9 pada docs/keputusan-implementasi.md.

-- name: klaim_perbarui
UPDATE POOLDATA.T_CLAIM_PNC
   SET CLAIMNO               = :1,
       PORTAL                = :2,
       NOPOLIS               = :3,
       GROUPPANEL            = :4,
       POLIS_JENIS_BISNIS    = :5,
       POLIS_MULAI           = :6,
       POLIS_AKHIR           = :7,
       POLIS_DEKLARASI       = :8,
       POLIS_MATA_UANG       = :9,
       POLIS_PENJAMIN_KREDIT = :10,
       QQNAME                = :11,
       BRANCHCODE            = :12,
       DATEOFLOSS            = :13,
       REPORTDATE            = :14,
       RECEIVEDATE           = :15,
       LOCATION              = :16,
       KRONOLOGI             = :17,
       REPORTERNAME          = :18,
       NO_HP                 = :19,
       PELAPOR_EMAIL         = :20,
       REPORTADDRESS         = :21,
       PELAPOR_HUBUNGAN      = :22,
       PELAPOR_HUBUNGAN_LAIN = :23,
       NILAI_ESTIMASI_SEN    = :24,
       CURRENCY              = :25,
       NOMOR_SLIK            = :26,
       EXGRATIA              = :27,
       PICTEKNIK             = :28,
       RCVID                 = :29,
       RCLPUCL               = :30,
       TRANSFER_COMPLIANCE   = :31,
       MINTA_KEMBALI         = :32,
       STATUSWORK            = :33,
       STATUSCLAIM           = :34,
       FLAG_KLAIM            = :35,
       STATUS_POSISI_PROGRES = :36,
       TAHAP_KINI            = :37,
       DIUBAH_OLEH           = :38,
       DIUBAH_PADA           = :39,
       DIHAPUS_PADA          = :40,
       FLAG_NOLL             = :41
 WHERE CLAIMID = :42

-- name: klaim_sisip
--
-- CLAIMID memuat pengenal internal klaim, CLAIMNO memuat nomornya. Pembagian itu
-- mengikuti tabel warisan apa adanya: di sana CLAIMID berisi
-- `ASM-FW-GCNMFW-WORK PNC-996` dan CLAIMNO berisi `PNC-996`.
--
-- Bedanya, klaim terbitan aplikasi ini TIDAK memakai prefix Pega (`D-22`): CLAIMID
-- berisi pengenal internalnya, dan CLAIMNO berisi `PNCN.YY.xxxx`.
INSERT INTO POOLDATA.T_CLAIM_PNC (
       CLAIMNO, PORTAL,
       NOPOLIS, GROUPPANEL, POLIS_JENIS_BISNIS, POLIS_MULAI, POLIS_AKHIR,
       POLIS_DEKLARASI, POLIS_MATA_UANG, POLIS_PENJAMIN_KREDIT, QQNAME,
       BRANCHCODE,
       DATEOFLOSS, REPORTDATE, RECEIVEDATE,
       LOCATION, KRONOLOGI,
       REPORTERNAME, NO_HP, PELAPOR_EMAIL, REPORTADDRESS,
       PELAPOR_HUBUNGAN, PELAPOR_HUBUNGAN_LAIN,
       NILAI_ESTIMASI_SEN, CURRENCY, NOMOR_SLIK, EXGRATIA, PICTEKNIK, RCVID,
       RCLPUCL, TRANSFER_COMPLIANCE, MINTA_KEMBALI,
       STATUSWORK, STATUSCLAIM, FLAG_KLAIM, STATUS_POSISI_PROGRES,
       TAHAP_KINI,
       DIUBAH_OLEH, DIUBAH_PADA, DIHAPUS_PADA, FLAG_NOLL,
       CLAIMID, ADMINKLAIM, REGISTERDATE)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10,
        :11, :12, :13, :14, :15, :16, :17, :18, :19, :20,
        :21, :22, :23, :24, :25, :26, :27, :28, :29, :30,
        :31, :32, :33, :34, :35, :36, :37, :38, :39, :40,
        :41, :42, :43, :44)

-- name: klaim_ambil
SELECT CLAIMID, CLAIMNO, PORTAL,
       NOPOLIS, GROUPPANEL, POLIS_JENIS_BISNIS, POLIS_MULAI, POLIS_AKHIR,
       POLIS_DEKLARASI, POLIS_MATA_UANG, POLIS_PENJAMIN_KREDIT, QQNAME,
       BRANCHCODE,
       DATEOFLOSS, REPORTDATE, RECEIVEDATE,
       LOCATION, KRONOLOGI,
       REPORTERNAME, NO_HP, PELAPOR_EMAIL, REPORTADDRESS,
       PELAPOR_HUBUNGAN, PELAPOR_HUBUNGAN_LAIN,
       NILAI_ESTIMASI_SEN, CURRENCY, NOMOR_SLIK, EXGRATIA, PICTEKNIK, RCVID,
       RCLPUCL, TRANSFER_COMPLIANCE, MINTA_KEMBALI,
       STATUSWORK, STATUSCLAIM, FLAG_KLAIM, STATUS_POSISI_PROGRES,
       TAHAP_KINI,
       ADMINKLAIM, REGISTERDATE, DIUBAH_OLEH, DIUBAH_PADA, DIHAPUS_PADA,
       FLAG_NOLL
  FROM POOLDATA.T_CLAIM_PNC
 WHERE CLAIMID = :1

-- name: klaim_ambil_per_nomor
SELECT CLAIMID, CLAIMNO, PORTAL,
       NOPOLIS, GROUPPANEL, POLIS_JENIS_BISNIS, POLIS_MULAI, POLIS_AKHIR,
       POLIS_DEKLARASI, POLIS_MATA_UANG, POLIS_PENJAMIN_KREDIT, QQNAME,
       BRANCHCODE,
       DATEOFLOSS, REPORTDATE, RECEIVEDATE,
       LOCATION, KRONOLOGI,
       REPORTERNAME, NO_HP, PELAPOR_EMAIL, REPORTADDRESS,
       PELAPOR_HUBUNGAN, PELAPOR_HUBUNGAN_LAIN,
       NILAI_ESTIMASI_SEN, CURRENCY, NOMOR_SLIK, EXGRATIA, PICTEKNIK, RCVID,
       RCLPUCL, TRANSFER_COMPLIANCE, MINTA_KEMBALI,
       STATUSWORK, STATUSCLAIM, FLAG_KLAIM, STATUS_POSISI_PROGRES,
       TAHAP_KINI,
       ADMINKLAIM, REGISTERDATE, DIUBAH_OLEH, DIUBAH_PADA, DIHAPUS_PADA,
       FLAG_NOLL
  FROM POOLDATA.T_CLAIM_PNC
 WHERE CLAIMNO = :1

-- name: objek_perbarui
UPDATE POOLDATA.T_CLAIM_OBJECTLIST
   SET OBJECTID = :1, OBJECTNAME = :2, LOKASI = :3, DIHAPUS_PADA = NULL
 WHERE CLAIMID = :4 AND URUTAN = :5

-- name: objek_sisip
INSERT INTO POOLDATA.T_CLAIM_OBJECTLIST (OBJECTID, OBJECTNAME, LOKASI, CLAIMID, URUTAN)
VALUES (:1, :2, :3, :4, :5)

-- name: objek_tandai_sisa
UPDATE POOLDATA.T_CLAIM_OBJECTLIST
   SET DIHAPUS_PADA = :1
 WHERE CLAIMID = :2 AND URUTAN > :3 AND DIHAPUS_PADA IS NULL

-- name: objek_daftar
SELECT URUTAN, OBJECTID, OBJECTNAME, LOKASI
  FROM POOLDATA.T_CLAIM_OBJECTLIST
 WHERE CLAIMID = :1 AND DIHAPUS_PADA IS NULL
 ORDER BY URUTAN

-- name: coverage_perbarui
--
-- SUMTSI dalam RUPIAH; domain mengirim SEN. Lihat catatan satuan uang di kepala berkas.
--
-- OBJECTID IKUT DIKIRIM, dan itu bukan pilihan gaya: kolomnya `NOT NULL` di tabel
-- warisan. Domain mengenali objek lewat URUTAN, sedangkan tabel ini mengenalinya lewat
-- id bisnisnya — keduanya harus ada supaya baris ini dapat dibaca dari kedua arah.
UPDATE POOLDATA.T_CLAIM_OBJECTCOVERAGE
   SET COVERAGEID = :1, CAUSEOFLOSSID = :2, SUMTSI = :3 / 100, OBJECTID = :4,
       OBJECTCOVERAGEID = :5, DIHAPUS_PADA = NULL
 WHERE CLAIMID = :6 AND URUTAN_OBJEK = :7 AND URUTAN = :8

-- name: coverage_sisip
--
-- Empat kolom tabel warisan bersifat NOT NULL: CLAIMID, OBJECTID, OBJECTCOVERAGEID, dan
-- CREATEDATETIME. Ketiadaan salah satunya ditolak basis data, bukan diam-diam dibiarkan.
--
-- OBJECTCOVERAGEID diisi URUTAN coverage di dalam objeknya — itulah bentuk yang dipakai
-- data nyata (terverifikasi 2026-09-24: nilainya "1", dan pasangan (klaim, nilai itu)
-- berulang lintas objek, jadi ia bukan pengenal global).
--
-- CREATEDATETIME diisi waktu dari pemanggil, BUKAN SYSDATE: `D-20` melarang SYSDATE demi
-- portabilitas, dan `F-5` menetapkan waktu datang dari satu tempat.
INSERT INTO POOLDATA.T_CLAIM_OBJECTCOVERAGE
       (COVERAGEID, CAUSEOFLOSSID, SUMTSI, OBJECTID, OBJECTCOVERAGEID, CREATEDATETIME,
        CLAIMID, URUTAN_OBJEK, URUTAN)
VALUES (:1, :2, :3 / 100, :4, :5, :6, :7, :8, :9)

-- name: coverage_tandai_sisa
UPDATE POOLDATA.T_CLAIM_OBJECTCOVERAGE
   SET DIHAPUS_PADA = :1
 WHERE CLAIMID = :2 AND URUTAN_OBJEK = :3 AND URUTAN > :4 AND DIHAPUS_PADA IS NULL

-- name: coverage_daftar
--
-- Dikalikan 100 supaya domain menerima SEN, satuan yang dipakainya.
SELECT URUTAN_OBJEK, URUTAN, COVERAGEID, CAUSEOFLOSSID, SUMTSI * 100
  FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE
 WHERE CLAIMID = :1 AND DIHAPUS_PADA IS NULL
 ORDER BY URUTAN_OBJEK, URUTAN

-- name: spreading_perbarui
UPDATE POOLDATA.T_CLAIM_SPREADING
   SET JENIS_TREATY = :1, NAMA = :2, SHARE_E4 = :3, DIHAPUS = :4,
       OBJEK_FAC_OFFER = :5, DIHAPUS_PADA = NULL
 WHERE CLAIMID = :6 AND URUTAN_OBJEK = :7 AND URUTAN_COVERAGE = :8 AND URUTAN = :9

-- name: spreading_sisip
INSERT INTO POOLDATA.T_CLAIM_SPREADING
       (JENIS_TREATY, NAMA, SHARE_E4, DIHAPUS, OBJEK_FAC_OFFER,
        CLAIMID, URUTAN_OBJEK, URUTAN_COVERAGE, URUTAN)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9)

-- name: spreading_tandai_sisa
UPDATE POOLDATA.T_CLAIM_SPREADING
   SET DIHAPUS_PADA = :1
 WHERE CLAIMID = :2 AND URUTAN_OBJEK = :3 AND URUTAN_COVERAGE = :4
   AND URUTAN > :5 AND DIHAPUS_PADA IS NULL

-- name: spreading_daftar
SELECT URUTAN_OBJEK, URUTAN_COVERAGE, URUTAN, JENIS_TREATY, NAMA, SHARE_E4, DIHAPUS, OBJEK_FAC_OFFER
  FROM POOLDATA.T_CLAIM_SPREADING
 WHERE CLAIMID = :1 AND DIHAPUS_PADA IS NULL
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
--
-- CATATAN PENTING atas tabel warisan: T_CLAIM_PNC memuat baris terbitan Pega DAN
-- terbitan aplikasi ini. Pemeriksaan ganda karena itu melihat KEDUANYA — dan memang
-- harus: klaim ganda tetap ganda meski yang pertama dibuat sistem lama.
SELECT DISTINCT k.CLAIMNO, o.OBJECTID
  FROM POOLDATA.T_CLAIM_PNC k
  JOIN POOLDATA.T_CLAIM_OBJECTLIST o
    ON o.CLAIMID = k.CLAIMID AND o.DIHAPUS_PADA IS NULL
 WHERE k.NOPOLIS = :1
   AND k.CLAIMID <> :2
   AND k.CLAIMNO IS NOT NULL
   AND k.DIHAPUS_PADA IS NULL
   AND o.OBJECTID = :3
   AND (:4 = 0 OR REPLACE(UPPER(k.LOCATION), ' ', '') = REPLACE(UPPER(:5), ' ', ''))
   AND (:6 = 0 OR EXISTS (
         SELECT 1
           FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE c
          WHERE c.CLAIMID = k.CLAIMID
            AND c.URUTAN_OBJEK = o.URUTAN
            AND c.DIHAPUS_PADA IS NULL
            AND c.CAUSEOFLOSSID = :7))
 ORDER BY k.CLAIMNO, o.OBJECTID
