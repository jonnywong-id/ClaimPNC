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
       SUSPICIOUSCOMMENT     = :51,
       -- Isian InputRegisterDetail2_sect (tab Input Register ClaimSurvey_sect).
       EMAIL_LOD             = :52,
       REMARKRECOMENDATION   = :53,
       SUBJECTEMAIL          = :54,
       STSSALVAGE            = :55,
       -- Tanggal transfer ke Analyst (setTicketToAnalyst); DATE, jam dinding WIB.
       ANALYST_TRANSFERDATE  = :56,
       -- Catatan ke PIC Teknis layar Input Estimasi (.ClaimData.Remark).
       REMARK                = :57,
       -- No KTP dan Pengkinian Data (No. HP, Email) Input Register PA.
       PENGKINIAN_NO_KTP     = :58,
       PENGKINIAN_NO_HP      = :59,
       PENGKINIAN_EMAIL      = :60,
       -- Jenis Laporan Input Register (.ClaimData.ReportType).
       REPORTTYPE            = :61,
       -- Tanggal Keluar Rawat Inap PA (.ClaimData.TanggalSelesaiRawatInap).
       TANGGALSELESAIRAWATINAP = :62
 WHERE CLAIMID = :63

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
       POSTALCODE, CUSTOMERPRINCIPLE, SUSPICIOUSCOMMENT,
       EMAIL_LOD, REMARKRECOMENDATION, SUBJECTEMAIL, STSSALVAGE, ANALYST_TRANSFERDATE,
       REMARK, PENGKINIAN_NO_KTP, PENGKINIAN_NO_HP, PENGKINIAN_EMAIL, REPORTTYPE,
       TANGGALSELESAIRAWATINAP, CLAIMID, ADMINKLAIM, REGISTERDATE)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10,
        :11, :12, :13, :14, :15, :16, :17, :18, :19, :20,
        :21, :22, :23, :24, :25, :26, :27, :28, :29, :30,
        :31, :32, :33, :34, :35, :36, :37, :38, :39, :40,
        :41, :42, :43, :44, :45, :46, :47, :48, :49, :50,
        :51, :52, :53, :54, :55, :56, :57, :58, :59, :60,
        :61, :62, :63, :64, :65)

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
       RW, RWID, POSTALCODE, CUSTOMERPRINCIPLE, SUSPICIOUSCOMMENT,
       EMAIL_LOD, REMARKRECOMENDATION, SUBJECTEMAIL, STSSALVAGE, ANALYST_TRANSFERDATE, REMARK,
       STS_TKI,
       PENGKINIAN_NO_KTP, PENGKINIAN_NO_HP, PENGKINIAN_EMAIL, REPORTTYPE,
       TANGGALSELESAIRAWATINAP
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
       RW, RWID, POSTALCODE, CUSTOMERPRINCIPLE, SUSPICIOUSCOMMENT,
       EMAIL_LOD, REMARKRECOMENDATION, SUBJECTEMAIL, STSSALVAGE, ANALYST_TRANSFERDATE, REMARK,
       STS_TKI,
       PENGKINIAN_NO_KTP, PENGKINIAN_NO_HP, PENGKINIAN_EMAIL, REPORTTYPE,
       TANGGALSELESAIRAWATINAP
  FROM POOLDATA.T_CLAIM_PNC
 WHERE CLAIMNO = :1

-- name: objek_perbarui
UPDATE POOLDATA.T_CLAIM_OBJECTLIST
   SET OBJECTID = :1, OBJECTNAME = :2, LOKASI = :3, DIHAPUS_PADA = NULL
 WHERE CLAIMID = :4 AND URUTAN = :5

-- name: objek_sisip
INSERT INTO POOLDATA.T_CLAIM_OBJECTLIST (OBJECTID, OBJECTNAME, LOKASI, CLAIMID, URUTAN)
VALUES (:1, :2, :3, :4, :5)

-- name: objek_kunci
--
-- Objek tersimpan satu klaim, supaya saveTree hanya menulis objek yang berubah. Klaim PA
-- menyimpan seluruh peserta polis sebagai objek (ratusan baris); menulis ulang semuanya pada
-- setiap simpan membuat tombol Tambah adjustment lambat (PNCN.26.57: 897 objek).
SELECT URUTAN, OBJECTID, OBJECTNAME, LOKASI, DIHAPUS_PADA
  FROM POOLDATA.T_CLAIM_OBJECTLIST
 WHERE CLAIMID = :1

-- name: objek_tandai_sisa
UPDATE POOLDATA.T_CLAIM_OBJECTLIST
   SET DIHAPUS_PADA = :1
 WHERE CLAIMID = :2 AND URUTAN > :3 AND DIHAPUS_PADA IS NULL

-- name: objek_daftar
--
-- URUTAN boleh KOSONG, dan pada data warisan ia hampir selalu kosong: 2.611 dari 2.726
-- baris aktif (diukur 2026-10-07). Kolom itu ditambahkan proyek ini; baris yang ditulis
-- Pega tidak pernah mengisinya. Karena itu penjodohan ke coverage memakai OBJECTID — kolom
-- `NOT NULL` di kedua tabel — dan URUTAN hanya dipakai untuk mengurutkan.
--
-- OBJECTID menjadi pemecah seri supaya urutan baris warisan tetap sama setiap kali dibaca;
-- tanpa itu Oracle bebas mengembalikannya dalam urutan apa pun.
--
-- Pekerjaan dan Tanggal Lahir peserta (grid objek PA, ShowObjectAdj) dibaca dari T_PERSONLIST polis
-- pada NOPOLIS + PRODKE klaim, dicocokkan INDEXOBJECT = OBJECTID seperti GetListObjectPATravel.
-- Objek bukan peserta tidak punya baris di sana dan kedua kolomnya NULL.
--
-- Kolom grid objek per lini (Section/InputRegisterDetail-sect.xml) dibaca dengan cara yang
-- sama, baca saja, dari tabel objek polis:
--   Travel -- KTP/Paspor (.ObjectIDCard) dan Status (.ObjectParticipantStatus) dari
--             T_PERSONLIST.ASMIDCARD / ASMPARTICIPANTSTATUS;
--   HE     -- Model, Merk, Nama Tipe, Nomor Chasis dari T_ANEKALIST, mengikuti alias
--             RDB List/GetListObjectAneka: VEHICLEHEOBJECTNAMEHE, VEHICLEHEBRANDNAME,
--             VEHICLEHETYPENAME, VEHICLEHECHASSISNUMBER.
--
-- Kedua tabel polis digabung SEKALI per klaim (dikelompokkan per INDEXOBJECT), bukan lewat
-- delapan subkueri berkorelasi per objek: klaim PA menyimpan seluruh peserta sebagai objek, dan
-- PNCN.26.57 (897 objek) butuh 10,6 detik dengan cara lama, 0,35 detik dengan cara ini — hasil
-- identik baris demi baris (diukur di TEST 2026-10-09). CLAIMID diikat tiga kali (:1, :2, :3).
SELECT o.URUTAN, o.OBJECTID, o.OBJECTNAME, o.LOKASI,
       pl.JOB, pl.BIRTH, pl.IDCARD, pl.STATUS,
       al.MODEL, al.BRAND, al.KIND, al.CHASSIS
  FROM POOLDATA.T_CLAIM_OBJECTLIST o
  LEFT JOIN (SELECT CAST(p.INDEXOBJECT AS VARCHAR(20)) AS KEY,
                    MAX(p.ASMJOBNAME) AS JOB, MAX(p.ASMDATEOFBIRTH) AS BIRTH,
                    MAX(p.ASMIDCARD) AS IDCARD, MAX(p.ASMPARTICIPANTSTATUS) AS STATUS
               FROM POOLDATA.T_PERSONLIST p
               JOIN POOLDATA.T_CLAIM_PNC c
                 ON p.NOPOLIS = c.NOPOLIS AND p.PRODKE = c.PRODKE
              WHERE c.CLAIMID = :1
              GROUP BY CAST(p.INDEXOBJECT AS VARCHAR(20))) pl
         ON pl.KEY = TRIM(o.OBJECTID)
  LEFT JOIN (SELECT CAST(p.INDEXOBJECT AS VARCHAR(20)) AS KEY,
                    MAX(p.VEHICLEHEOBJECTNAMEHE) AS MODEL, MAX(p.VEHICLEHEBRANDNAME) AS BRAND,
                    MAX(p.VEHICLEHETYPENAME) AS KIND, MAX(p.VEHICLEHECHASSISNUMBER) AS CHASSIS
               FROM POOLDATA.T_ANEKALIST p
               JOIN POOLDATA.T_CLAIM_PNC c
                 ON p.NOPOLIS = c.NOPOLIS AND p.PRODKE = c.PRODKE
              WHERE c.CLAIMID = :2
              GROUP BY CAST(p.INDEXOBJECT AS VARCHAR(20))) al
         ON al.KEY = TRIM(o.OBJECTID)
 WHERE o.CLAIMID = :3 AND o.DIHAPUS_PADA IS NULL
 ORDER BY o.URUTAN, o.OBJECTID

-- name: coverage_perbarui
--
-- SUMTSI dalam RUPIAH; domain mengirim SEN. Lihat catatan satuan uang di kepala berkas.
--
-- OBJECTID IKUT DIKIRIM, dan itu bukan pilihan gaya: kolomnya `NOT NULL` di tabel
-- warisan. Domain mengenali objek lewat URUTAN, sedangkan tabel ini mengenalinya lewat
-- id bisnisnya — keduanya harus ada supaya baris ini dapat dibaca dari kedua arah.
UPDATE POOLDATA.T_CLAIM_OBJECTCOVERAGE
   SET COVERAGEID = :1, CAUSEOFLOSSID = :2, SUMTSI = :3 / 100, OBJECTID = :4,
       OBJECTCOVERAGEID = :5, COVERAGENAME = :6, DIHAPUS_PADA = NULL,
       -- Penanda Transfer ke Analyst (setTicketToAnalyst step 5): hanya diisi, tidak pernah
       -- dikosongkan. Nilai yang sama dikirim tiga kali supaya tiap placeholder tunggal.
       ISANALISTRANSFER = CASE WHEN :7 = 1 THEN 1 ELSE ISANALISTRANSFER END,
       ISKOMITETRANSFER = CASE WHEN :8 = 1 THEN 1 ELSE ISKOMITETRANSFER END,
       USERBUSINESSPA   = CASE WHEN :9 = 1 THEN 1 ELSE USERBUSINESSPA END
 WHERE CLAIMID = :10 AND URUTAN_OBJEK = :11 AND URUTAN = :12

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

-- name: coverage_kunci
--
-- Seluruh baris coverage satu klaim -- termasuk yang sudah bertanda DIHAPUS_PADA -- untuk
-- menentukan coverage mana yang sudah dibuang petugas sebelum pohon disimpan ulang.
-- Lihat ClaimStore.dropRemoved.
SELECT URUTAN_OBJEK, URUTAN, OBJECTID, OBJECTCOVERAGEID
  FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE
 WHERE CLAIMID = :1

-- name: coverage_hapus_sisa
--
-- Coverage yang dibuang petugas DIHAPUS, bukan lagi ditandai DIHAPUS_PADA -- permintaan
-- Work Owner 2026-10-07: saat data coverage dihapus, datanya di T_CLAIM_OBJECTCOVERAGE
-- juga ikut dihapus. Menyupersede penandaan ADR-0012 untuk tabel ini.
DELETE FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE
 WHERE CLAIMID = :1 AND URUTAN_OBJEK = :2 AND URUTAN > :3

-- name: coverage_hapus_objek_sisa
--
-- Coverage milik objek yang dibuang petugas ikut dihapus. Objeknya sendiri tetap
-- ditandai DIHAPUS_PADA (objek_tandai_sisa) -- permintaan itu tidak menyangkut objek.
DELETE FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE
 WHERE CLAIMID = :1 AND URUTAN_OBJEK > :2

-- name: coverage_daftar
--
-- Dikalikan 100 supaya domain menerima SEN, satuan yang dipakainya.
--
-- OBJECTID dan OBJECTCOVERAGEID IKUT DIBACA, dan keduanya yang menjodohkan baris ini ke
-- pohon klaim. URUTAN_OBJEK dan URUTAN kosong pada 2.598 dari 2.635 baris aktif (diukur
-- 2026-10-07) — persis sebanyak baris objek yang URUTAN-nya juga kosong. Kedua kolom itu
-- ditambahkan proyek ini dan tidak pernah diisi Pega.
--
-- Keduanya `NOT NULL` di tabel ini, sehingga penjodohannya berlaku untuk baris warisan
-- maupun baris baru.
SELECT OBJECTID, OBJECTCOVERAGEID, URUTAN_OBJEK, URUTAN, COVERAGEID, CAUSEOFLOSSID,
       SUMTSI * 100, COVERAGENAME, ISANALISTRANSFER,
       CURICUMOFLOSS, EXTENTOFLOSS, LEGALLIABILITY, REMARKS, REMARKINVESTIGATION,
       DIAGNOSE, CODEDIAGNOSE, DESCDIAGNOSE, TEMPRECEIVER, INITIALNAME, TANGGALCOMITEE
  FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE
 WHERE CLAIMID = :1 AND DIHAPUS_PADA IS NULL
 ORDER BY URUTAN_OBJEK, URUTAN, OBJECTID, OBJECTCOVERAGEID

-- name: coverage_catatan_komite
--
-- Isian modal "Transfer Claim ke Komite" (Section/ClaimComitee_OC) satu jaminan — lihat
-- registrasi.CommitteeNote. INITIALNAME dan TANGGALCOMITEE tidak ditulis: activity yang
-- mengisinya (PNCSaveButton2) tidak ada di export.
UPDATE POOLDATA.T_CLAIM_OBJECTCOVERAGE
   SET CURICUMOFLOSS = :1, EXTENTOFLOSS = :2, LEGALLIABILITY = :3, REMARKS = :4,
       REMARKINVESTIGATION = :5, DIAGNOSE = :6, CODEDIAGNOSE = :7, DESCDIAGNOSE = :8,
       TEMPRECEIVER = :9
 WHERE CLAIMID = :10 AND URUTAN_OBJEK = :11 AND URUTAN = :12 AND DIHAPUS_PADA IS NULL

-- ============================================================================
-- SPREADING — sisip, perbarui, dan hapus mengikuti isi layar
-- ============================================================================
--
-- Bentuk tabel POOLDATA.T_CLAIM_SPREADING DITETAPKAN Work Owner (CREATE_TABLE_2.sql,
-- 2026-09-26). Aturan tulis awalnya hanya INSERT tanpa DELETE; Work Owner
-- MENGUBAHNYA 2026-10-07: saat data spreading dihapus, datanya di T_CLAIM_SPREADING
-- juga ikut dihapus. Tanpa penghapusan, spreading yang dibuang petugas muncul kembali
-- saat klaim dibuka ulang dan ikut terhitung pada aturan total 100% (D-51).
--
-- Kuncinya (CLAIMID, OBJECTID, OBJECTCOVERAGEID, TREATYTYPE) — satu baris per jenis
-- treaty pada satu coverage. Baris yang tetap ada DIPERBARUI, bukan dihapus lalu
-- disisipkan ulang: tabelnya punya 18 kolom sementara aplikasi hanya menulis tujuh, dan
-- penggantian utuh akan mengosongkan sebelas sisanya.
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

-- name: spreading_jenis
SELECT TREATYTYPE
  FROM POOLDATA.T_CLAIM_SPREADING
 WHERE CLAIMID = :1 AND OBJECTID = :2 AND OBJECTCOVERAGEID = :3

-- name: spreading_perbarui
UPDATE POOLDATA.T_CLAIM_SPREADING
   SET TREATYNAME = :1, SHAREPERCENTAGE = :2, URUTAN = :3
 WHERE CLAIMID = :4 AND OBJECTID = :5 AND OBJECTCOVERAGEID = :6 AND TREATYTYPE = :7

-- name: spreading_hapus
DELETE FROM POOLDATA.T_CLAIM_SPREADING
 WHERE CLAIMID = :1 AND OBJECTID = :2 AND OBJECTCOVERAGEID = :3 AND TREATYTYPE = :4

-- name: spreading_hapus_coverage
DELETE FROM POOLDATA.T_CLAIM_SPREADING
 WHERE CLAIMID = :1 AND OBJECTID = :2 AND OBJECTCOVERAGEID = :3

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
