-- Kueri laporan kelompok Operasional.
--
-- Kelima aturan yang mengikat berkas kueri modul ini disebut di
-- reportklaim_reasuransi.sql dan berlaku sama di sini.
--
-- Dua dari lima panel kelompok ini TERHALANG dan karena itu tidak punya kueri di sini:
--
--   REPORT MITRA      penyaring barisnya menempuh DB Link @ASMD (`R-03`)
--   REPORT ADJUSTER   kedua Report Definition-nya tidak ada di export (`R-16`)
--
-- REPORT COMPLIANCE juga terhalang — isinya properti klipboard Pega, bukan kolom.
-- Alasan ketiganya ada di catalog_operasional.go.


-- name: report_produksi_klaim_pa
--
-- REPORT PRODUKSI KLAIM PA — 15 kolom.
--
-- Asal: LIMA rule pencacah yang dijalankan `Activity/ReportProduksiPA_act-Act.xml`:
--
--   CountCoverageID              COVERAGEID = '10003'   Resiko A
--   CountCoverageResikoB         COVERAGEID = '10004'   Resiko B
--   CountCoverageResikoD         COVERAGEID = '10005'   Resiko D
--   CountCoverageResikoMC        COVERAGEID = '10007'   Resiko MC
--   CountCoverageResikoLainnya   COVERAGEID NOT IN ('10003','10004','10005','10007')
--
-- Bind:
--   :1  tanggal dari    DATE
--   :2  tanggal sampai  DATE
--
-- # Lima kueri menjadi satu, dan hasilnya sama
--
-- Kelimanya membaca DUA TABEL YANG SAMA, dengan penyaring periode yang sama, dan
-- mengelompokkan menurut bulan yang sama. Yang berbeda hanya kelompok coverage-nya.
-- Sistem lama menjalankan kelimanya berurutan lalu menyandingkan hasilnya kolom demi
-- kolom — lima kali pembacaan tabel adjustment untuk satu berkas.
--
-- Di sini kelimanya menjadi lima agregat bersyarat atas satu pembacaan. Barisnya tetap
-- satu per bulan dan angkanya tetap sama; yang hilang hanya empat kali pembacaan.
--
-- # Kenapa kolom "Tahun" muncul lima kali
--
-- Karena memang begitu di berkas lama: setiap kelompok membawa kolom bulannya sendiri.
-- Kelimanya bernilai sama pada satu baris. Ia tidak dirapikan menjadi satu kolom —
-- berkas ini dibaca ulang oleh berkas kerja yang sudah ada di sisi pengguna, dan
-- menghapus empat kolom akan menggeser seluruh kolom di kanannya.
--
-- # Pengelompokan menurut bulan TIDAK lagi memakai TO_CHAR
--
-- Kueri aslinya mengelompokkan dan MEMBANDINGKAN periode sebagai teks `'yyyy-mm'`.
-- Perbandingan teks atas periode kebetulan benar untuk bentuk itu, tetapi ia mengikat
-- kueri pada satu dialek dan mematikan index atas kolom tanggalnya. Penyaringnya di sini
-- membandingkan TANGGAL; pengelompokannya memakai EXTRACT, yang portabel.
SELECT EXTRACT(YEAR FROM a.createdatetime)  AS "TahunPeriode",
       EXTRACT(MONTH FROM a.createdatetime) AS "BulanPeriode",

       -- # Alias memakai nama PROPERTI CSV, bukan nama alias kueri aslinya
       --
       -- Kelima rule asal mengaliaskan kolomnya `PRODKE`, `MARKETINGNAME`, `BUSINESSNAME`,
       -- dan seterusnya — nama yang tidak ada hubungannya dengan isinya. Activity-nya
       -- lalu MENYALIN kelima belas nilai itu ke properti CSV lewat lima belas langkah
       -- Property-Set (`TempExportPA.pxResults(<CURRENT>).CityID = .ACCUMCODE`, dan
       -- seterusnya).
       --
       -- Penyalinan itu tidak menambah apa pun selain nama. Karena kueri di sini sudah
       -- bebas memilih aliasnya, ia langsung memakai nama tujuannya — dan lima belas
       -- langkah penyalinan itu hilang bersama kemungkinan salah petanya.
       --
       -- Judul kolomnya tetap menyesatkan dan ditiru apa adanya: "Resiko A" berisi
       -- CACAH klaim, dan "Jumlah Klaim" berisi JUMLAH NILAI. Menukarnya agar sesuai
       -- judul berarti mengubah isi berkas yang sudah dibaca berkas kerja penggunanya.
       COUNT(CASE WHEN a.coverageid = '10003' THEN 1 END) AS "AlasanDokterRejectRCL",
       SUM(CASE WHEN a.coverageid = '10003' THEN b.grossvalue ELSE 0 END) AS "CompliancePosAuditByr",

       COUNT(CASE WHEN a.coverageid = '10004' THEN 1 END) AS "Country",
       SUM(CASE WHEN a.coverageid = '10004' THEN b.grossvalue ELSE 0 END) AS "ComplianceRemark",

       COUNT(CASE WHEN a.coverageid = '10005' THEN 1 END) AS "AnalystDoctorRemaks",
       SUM(CASE WHEN a.coverageid = '10005' THEN b.grossvalue ELSE 0 END) AS "Conveyance",

       COUNT(CASE WHEN a.coverageid = '10007' THEN 1 END) AS "AnalystRemaksInvestigator",
       SUM(CASE WHEN a.coverageid = '10007' THEN b.grossvalue ELSE 0 END) AS "CustomerPrinciple",

       COUNT(CASE WHEN a.coverageid NOT IN ('10003','10004','10005','10007') THEN 1 END) AS "CloseClaimNote",
       SUM(CASE WHEN a.coverageid NOT IN ('10003','10004','10005','10007') THEN b.grossvalue ELSE 0 END) AS "District"
  FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE a
  JOIN POOLDATA.T_CLAIM_ADJUSTMENT b ON a.claimid = b.claimid
 WHERE CAST(a.createdatetime AS DATE) >= :1
   AND CAST(a.createdatetime AS DATE) <= :2
 GROUP BY EXTRACT(YEAR FROM a.createdatetime), EXTRACT(MONTH FROM a.createdatetime)
 ORDER BY EXTRACT(YEAR FROM a.createdatetime), EXTRACT(MONTH FROM a.createdatetime)


-- name: report_komunikasi_klaim
--
-- REPORT DATA KOMUNIKASI KLAIM — 7 kolom.
--
-- Asal: `RDB List/ReportDataInboxKomunikasi_Klaim-SQL.xml`.
--
-- Tanpa bind: kuerinya mengambil SELURUH percakapan tanpa satu pun penyaring — termasuk
-- tanpa batas periode. Itu ditiru apa adanya; membubuhkan batas yang tidak ada di sistem
-- lama akan membuat berkasnya berbeda isi dari yang selama ini diterima.
--
-- # Empat kolom kueri asli tidak dipakai, dan satu alias kembar dibuang
--
-- Kueri aslinya memilih 11 kolom sementara `CSVProperties` hanya menyebut 7. Tiga
-- sisanya tidak pernah sampai ke berkas.
--
-- Yang keempat berbeda sebabnya: `sendername` dan `COMMUNICATE_FROM` sama-sama
-- dialiaskan `"UserName"`. Alias kembar berarti salah satunya menimpa yang lain, dan
-- yang menang bergantung pada urutan pembacaan driver. Yang dipakai di sini adalah
-- `sendername` — kolom "Pengirim" pada berkas lama — dan yang kembar dibuang.
SELECT caseid           AS "pzInsKey",
       createddate      AS "CloseClaimDate",
       message          AS "Email",
       sendername       AS "UserName",
       replymessage     AS "CloseClaimNote",
       createdatereply  AS "AnalystTransferDate",
       replyfromname    AS "UserAdmin"
  FROM pooldata.m_komunikasi_pnc
 ORDER BY createddate DESC
