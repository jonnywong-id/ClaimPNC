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
--   CPNC_KLAIM_SPREADING   -> POOLDATA.T_CLAIM_SPREADING   (DDL Work Owner, CREATE_TABLE_2.sql)
--
-- # T_CLAIM_PNC diubah Work Owner, 2026-09-26 15:31
--
-- Dari 97 menjadi 82 kolom. Dari kolom tambahan migrasi 0007, tujuh DIPERTAHANKAN
-- (POLIS_MATA_UANG, NOMOR_SLIK, PELAPOR_HUBUNGAN, PELAPOR_HUBUNGAN_LAIN, PORTAL,
-- POLIS_JENIS_BISNIS, FLAG_NOLL) dan sisanya DIBUANG — termasuk tahap klaim, periode
-- polis, estimasi, email pelapor, penanda proses, dan kolom jejak ubah/hapus.
--
-- Kueri di bawah menulis dan membaca 82 kolom itu saja. Medan yang kolomnya dibuang
-- dipulihkan dari sumber lain atau diturunkan — lihat restoreDropped di claim.go.
-- Kepala klaim karena itu juga TIDAK lagi punya penanda hapus; tabel anaknya masih.
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
--    Domain menyimpan uang sebagai SEN (`ADR-0016`). `SUMTSI` adalah kolom
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
       POLIS_MATA_UANG       = :6,
       QQNAME                = :7,
       BRANCHCODE            = :8,
       DATEOFLOSS            = :9,
       REPORTDATE            = :10,
       RECEIVEDATE           = :11,
       LOCATION              = :12,
       KRONOLOGI             = :13,
       REPORTERNAME          = :14,
       NO_HP                 = :15,
       REPORTADDRESS         = :16,
       PELAPOR_HUBUNGAN      = :17,
       PELAPOR_HUBUNGAN_LAIN = :18,
       CURRENCY              = :19,
       NOMOR_SLIK            = :20,
       EXGRATIA              = :21,
       PICTEKNIK             = :22,
       RCVID                 = :23,
       RCLPUCL               = :24,
       STATUSWORK            = :25,
       STATUSCLAIM           = :26,
       FLAG_NOLL             = :27,
       SOBNAME               = :28,
       SOBNAMEID             = :29,
       BRANCHNAME            = :30,
       BUSINESSCODE          = :31,
       BUSINESSNAME          = :32,
       PRODKE                = :33,
       TYPEOFCOINS           = :34,
       COINSNAME             = :35,
       LEADER_MEMBER         = :36,
       SHAREASM              = :37,
       POLISLEADER           = :38,
       COUNTRY               = :39,
       COUNTRYID             = :40,
       PROVINCE              = :41,
       PROVINCEID            = :42,
       CITY                  = :43,
       CITYID                = :44,
       DISTRICT              = :45,
       DISTRICTID            = :46,
       RW                    = :47,
       RWID                  = :48,
       POSTALCODE            = :49,
       CUSTOMERPRINCIPLE     = :50,
       SUSPICIOUSCOMMENT     = :51
 WHERE CLAIMID = :52

-- name: klaim_sisip
--
-- CLAIMID memuat pengenal internal klaim, CLAIMNO memuat nomornya. Pembagian itu
-- mengikuti tabel warisan apa adanya: di sana CLAIMID berisi
-- `ASM-FW-GCNMFW-WORK PNC-996` dan CLAIMNO berisi `PNC-996`.
--
-- Bedanya, klaim terbitan aplikasi ini TIDAK memakai prefix Pega (`D-22`): CLAIMID
-- berisi pengenal internalnya, dan CLAIMNO berisi `PNCN.YY.xxxx`.
INSERT INTO POOLDATA.T_CLAIM_PNC (
       CLAIMNO, PORTAL, NOPOLIS, GROUPPANEL, POLIS_JENIS_BISNIS, POLIS_MATA_UANG,
       QQNAME, BRANCHCODE, DATEOFLOSS, REPORTDATE, RECEIVEDATE, LOCATION,
       KRONOLOGI, REPORTERNAME, NO_HP, REPORTADDRESS, PELAPOR_HUBUNGAN, PELAPOR_HUBUNGAN_LAIN,
       CURRENCY, NOMOR_SLIK, EXGRATIA, PICTEKNIK, RCVID, RCLPUCL,
       STATUSWORK, STATUSCLAIM, FLAG_NOLL, SOBNAME, SOBNAMEID, BRANCHNAME,
       BUSINESSCODE, BUSINESSNAME, PRODKE, TYPEOFCOINS, COINSNAME, LEADER_MEMBER,
       SHAREASM, POLISLEADER, COUNTRY, COUNTRYID, PROVINCE, PROVINCEID,
       CITY, CITYID, DISTRICT, DISTRICTID, RW, RWID,
       POSTALCODE, CUSTOMERPRINCIPLE, SUSPICIOUSCOMMENT, CLAIMID, ADMINKLAIM, REGISTERDATE)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10,
        :11, :12, :13, :14, :15, :16, :17, :18, :19, :20,
        :21, :22, :23, :24, :25, :26, :27, :28, :29, :30,
        :31, :32, :33, :34, :35, :36, :37, :38, :39, :40,
        :41, :42, :43, :44, :45, :46, :47, :48, :49, :50,
        :51, :52, :53, :54)

-- name: klaim_ambil
SELECT CLAIMID, CLAIMNO, PORTAL,
       NOPOLIS, GROUPPANEL, POLIS_JENIS_BISNIS, POLIS_MATA_UANG, QQNAME,
       BRANCHCODE,
       DATEOFLOSS, REPORTDATE, RECEIVEDATE,
       LOCATION, KRONOLOGI,
       REPORTERNAME, NO_HP, REPORTADDRESS,
       PELAPOR_HUBUNGAN, PELAPOR_HUBUNGAN_LAIN,
       CURRENCY, NOMOR_SLIK, EXGRATIA, PICTEKNIK, RCVID,
       RCLPUCL,
       STATUSWORK, STATUSCLAIM,
       ADMINKLAIM, REGISTERDATE,
       FLAG_NOLL,
       SOBNAME, SOBNAMEID, BRANCHNAME, BUSINESSCODE, BUSINESSNAME, PRODKE, TYPEOFCOINS,
       COINSNAME, LEADER_MEMBER, ROUND(SHAREASM * 10000), POLISLEADER,
       COUNTRY, COUNTRYID, PROVINCE, PROVINCEID, CITY, CITYID, DISTRICT, DISTRICTID,
       RW, RWID, POSTALCODE, CUSTOMERPRINCIPLE, SUSPICIOUSCOMMENT
  FROM POOLDATA.T_CLAIM_PNC
 WHERE CLAIMID = :1

-- name: klaim_ambil_per_nomor
SELECT CLAIMID, CLAIMNO, PORTAL,
       NOPOLIS, GROUPPANEL, POLIS_JENIS_BISNIS, POLIS_MATA_UANG, QQNAME,
       BRANCHCODE,
       DATEOFLOSS, REPORTDATE, RECEIVEDATE,
       LOCATION, KRONOLOGI,
       REPORTERNAME, NO_HP, REPORTADDRESS,
       PELAPOR_HUBUNGAN, PELAPOR_HUBUNGAN_LAIN,
       CURRENCY, NOMOR_SLIK, EXGRATIA, PICTEKNIK, RCVID,
       RCLPUCL,
       STATUSWORK, STATUSCLAIM,
       ADMINKLAIM, REGISTERDATE,
       FLAG_NOLL,
       SOBNAME, SOBNAMEID, BRANCHNAME, BUSINESSCODE, BUSINESSNAME, PRODKE, TYPEOFCOINS,
       COINSNAME, LEADER_MEMBER, ROUND(SHAREASM * 10000), POLISLEADER,
       COUNTRY, COUNTRYID, PROVINCE, PROVINCEID, CITY, CITYID, DISTRICT, DISTRICTID,
       RW, RWID, POSTALCODE, CUSTOMERPRINCIPLE, SUSPICIOUSCOMMENT
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
       OBJECTCOVERAGEID = :5, COVERAGENAME = :6, DIHAPUS_PADA = NULL
 WHERE CLAIMID = :7 AND URUTAN_OBJEK = :8 AND URUTAN = :9

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
        CLAIMID, URUTAN_OBJEK, URUTAN, COVERAGENAME)
VALUES (:1, :2, :3 / 100, :4, :5, :6, :7, :8, :9, :10)

-- name: coverage_tandai_sisa
UPDATE POOLDATA.T_CLAIM_OBJECTCOVERAGE
   SET DIHAPUS_PADA = :1
 WHERE CLAIMID = :2 AND URUTAN_OBJEK = :3 AND URUTAN > :4 AND DIHAPUS_PADA IS NULL

-- name: coverage_daftar
--
-- Dikalikan 100 supaya domain menerima SEN, satuan yang dipakainya.
SELECT URUTAN_OBJEK, URUTAN, COVERAGEID, CAUSEOFLOSSID, SUMTSI * 100, COVERAGENAME
  FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE
 WHERE CLAIMID = :1 AND DIHAPUS_PADA IS NULL
 ORDER BY URUTAN_OBJEK, URUTAN

-- ============================================================================
-- SPREADING — insert bila belum ada, tanpa proses hapus
-- ============================================================================
--
-- Bentuk tabel POOLDATA.T_CLAIM_SPREADING DITETAPKAN Work Owner (CREATE_TABLE_2.sql,
-- 2026-09-26), dan cara menulisinya ditetapkan bersamanya: dari aplikasi hanya INSERT
-- bila barisnya belum ada, dan TIDAK ADA proses DELETE.
--
-- Kuncinya (CLAIMID, OBJECTID, OBJECTCOVERAGEID, TREATYTYPE) — satu baris per jenis
-- treaty pada satu coverage. Dengan aturan tulis di atas, kunci itu konsisten: tidak ada
-- baris yang ditandai terhapus lalu digantikan baris berjenis sama.
--
-- # Kenapa DUA pernyataan, bukan satu
--
-- Insert-bila-belum-ada dalam satu pernyataan menuntut MERGE atau FROM DUAL. Keduanya
-- ditolak: MERGE bukan sintaks yang sama di PostgreSQL (keputusan 2.9), dan FROM DUAL
-- dilarang Database Strategy §4. Memeriksa lebih dulu lalu menyisipkan berjalan apa
-- adanya di kedua basis data, dan membuat aturannya terbaca di kode.
--
-- # Satuan share
--
-- Domain menyimpan share sebagai BILANGAN BULAT x 10.000 (Percent, 100% = 1.000.000)
-- supaya toleransi 99,9999–100,0001 (D-51) tidak pernah bergantung pada pembulatan
-- pecahan biner. Kolomnya NUMBER(9,6) — desimal eksak, cukup untuk empat desimal itu.
-- Pembagian dilakukan di Go dan hasilnya dibulatkan kembali di SQL saat dibaca, sehingga
-- nilai yang keluar sama persis dengan yang masuk.

-- name: spreading_ada
SELECT 1
  FROM POOLDATA.T_CLAIM_SPREADING
 WHERE CLAIMID = :1 AND OBJECTID = :2 AND OBJECTCOVERAGEID = :3 AND TREATYTYPE = :4

-- name: spreading_sisip
INSERT INTO POOLDATA.T_CLAIM_SPREADING
       (CLAIMID, OBJECTID, OBJECTCOVERAGEID, TREATYTYPE, TREATYNAME, SHAREPERCENTAGE, URUTAN)
VALUES (:1, :2, :3, :4, :5, :6, :7)

-- name: spreading_daftar
--
-- SHAREPERCENTAGE dikembalikan ke satuan domain: dikalikan 10.000 lalu dibulatkan, supaya
-- hasilnya bilangan bulat yang sama dengan yang disimpan pemanggil.
--
-- Penjodohan ke pohon memakai OBJECTID dan OBJECTCOVERAGEID — kolom yang memang ada di
-- tabel ini — bukan urutan objek/coverage. Pemanggil menerjemahkannya lewat daftar objek
-- yang sudah dibacanya lebih dulu.
--
-- Nama treaty yang kosong dilengkapi dari master REINSURANCETYPE: klaim yang dibuka sebelum
-- nama treaty ikut diisi menyimpan TREATYNAME kosong, dan baris spreading tidak pernah
-- di-UPDATE (lihat spreading_sisip).
SELECT s.OBJECTID, s.OBJECTCOVERAGEID, s.URUTAN, s.TREATYTYPE,
       COALESCE(TRIM(s.TREATYNAME), r.NOTE),
       ROUND(s.SHAREPERCENTAGE * 10000)
  FROM POOLDATA.T_CLAIM_SPREADING s
  LEFT JOIN POOLDATA.REINSURANCETYPE r
         ON CAST(r.ID AS VARCHAR(20)) = TRIM(s.TREATYTYPE)
 WHERE s.CLAIMID = :1
 ORDER BY s.OBJECTID, s.OBJECTCOVERAGEID, s.URUTAN

-- name: klaim_cari_ganda
-- Pemeriksaan klaim ganda.
--
-- Empat hal yang membuat kueri ini benar:
--   1. Baris objek yang ditandai terhapus tidak ikut (ADR-0012). Kepala klaim tidak lagi
--      punya penanda hapus sejak T_CLAIM_PNC diubah Work Owner (2026-09-26).
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

-- name: penerima_perbarui
--
-- Penerima klaim, POOLDATA.T_CLAIM_RECEIVER (CLAIMID, IDRECEIVER NOT NULL, NAME, NAMEOFBANK,
-- NOACCOUNT, ADDRESS). Tabel warisan tanpa penanda hapus: baris hanya diperbarui atau
-- disisipkan (D-66). Ditulis hanya untuk klaim yang dibuat aplikasi ini.
UPDATE POOLDATA.T_CLAIM_RECEIVER
   SET NAME = :1, ADDRESS = :2, NAMEOFBANK = :3, NOACCOUNT = :4
 WHERE CLAIMID = :5 AND IDRECEIVER = :6

-- name: penerima_sisip
INSERT INTO POOLDATA.T_CLAIM_RECEIVER (NAME, ADDRESS, NAMEOFBANK, NOACCOUNT, CLAIMID, IDRECEIVER)
VALUES (:1, :2, :3, :4, :5, :6)

-- name: penerima_daftar
SELECT IDRECEIVER, NAME, ADDRESS, NAMEOFBANK, NOACCOUNT
  FROM POOLDATA.T_CLAIM_RECEIVER
 WHERE CLAIMID = :1
 ORDER BY TO_NUMBER(IDRECEIVER)
