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


-- name: report_mitra
--
-- Produktivitas mitra — satu baris per penugasan, beserta empat kolom rekapitulasi
-- per petugas.
--
-- Asal: `RDB List/ExportDetailMitraReport-SQL.xml`, dijalankan
-- `Activity/PNCMitraReport_Act-Act.xml`.
--
-- Bind:
--   :1  tanggal dari    DATE
--   :2  tanggal sampai  DATE
--
-- ============================================================================
-- Gabungan DB Link DIBUANG dari sini — dan itu MENGUBAH baris, bukan kolom
-- ============================================================================
--
-- Kueri aslinya menggabung `general.lst_mitra@asmd` sebagai INNER JOIN:
--
--     FROM pooldata.PNC_CHRONOLOGYTAT a, general.lst_mitra@asmd... b
--    WHERE a.userassign = b.login_aplikasi
--
-- Tidak satu pun kolom diambil dari `b`. Gabungan itu murni PENYARING: ia membatasi
-- laporan pada petugas yang terdaftar sebagai mitra.
--
-- Karena `b` berada di basis data lain, penyaringnya dipindahkan ke koneksi kedua
-- (`report_mitra_logins` pada reportklaim_aneka.sql) lalu diterapkan di Go. Keputusan
-- Work Owner 2026-09-24: yang berupa sub-query tetap memakai DB Link, selain itu memakai
-- koneksi langsung — dan gabungan ini bukan sub-query.
--
-- **Kueri ini karena itu TIDAK boleh dijalankan tanpa penyaring mitranya.** Tanpa itu
-- seluruh petugas masuk ke laporan produktivitas mitra: berkasnya tetap terbit, angkanya
-- tetap masuk akal, dan isinya bukan yang diminta. Penjagaannya ada di Repo.mitraLogins,
-- yang MENOLAK menjalankan laporan bila daftar mitranya tidak dapat dibaca.
--
-- ============================================================================
-- Tiga hal yang ditiru APA ADANYA dari Pega
-- ============================================================================
--
--  1. **`'YUNIARTAULIASI'` sebagai kolom "Atasan".** Nama orang yang ditulis sebagai
--     literal di dalam SQL — salah satu dari 24 Operator ID hardcode yang `D-15`
--     haruskan menjadi master data. Tidak ada sumber penggantinya di export, dan
--     Work Owner menetapkan yang sudah sesuai Pega dibiarkan apa adanya.
--
--  2. **Keempat kolom rekapitulasi TIDAK disaring periode.** Sub-kueri pencacahnya
--     hanya menyaring `userassign`, sehingga "Total Produktivitas" menghitung SELURUH
--     riwayat petugas itu — bukan hanya periode yang dipilih. Baris rinciannya disaring
--     periode; rekapitulasinya tidak. Itu perilaku aslinya.
--
--  3. **Pembagi tidak dijaga NULLIF.** Ia memang tidak dapat nol: baris `a` sendiri
--     sudah membuat pencacah penyebutnya minimal satu.
--
-- ============================================================================
-- Yang berbeda dari kueri asli, dan sebabnya
-- ============================================================================
--
--  * `TRUNC(a.insertdate) >= :1 AND TRUNC(a.insertdate) <= :2` menjadi rentang setengah
--    terbuka. `TRUNC` pada kolom mematikan index dan memaksa pemindaian tabel penuh;
--    rentang ini mencakup hari yang sama persis.
--
--  * `ORDER BY` ditambahkan. Kueri asli tidak punya, sehingga urutan barisnya ditentukan
--    Oracle dan dapat berbeda antar-jalan. Urutan tetap membuat dua berkas ekspor dapat
--    dibandingkan baris per baris — yang justru dibutuhkan saat menguji kesetaraan.
--
--  * `TO_CHAR` pada `timein`/`timeout` **DIPERTAHANKAN**, berbeda dari kueri lain di
--    modul ini. Alasannya: kedua kolom itu menampilkan JAM, sedangkan pemformat bersama
--    `text()` hanya mengeluarkan tanggal. Yang dilarang `D-20` adalah TO_CHAR untuk
--    memformat tampilan tanggal; di sini jamnya bagian dari isi laporan — laporan SLA
--    yang kehilangan jam masuk dan jam keluar tidak dapat dipakai.
SELECT a.userassign AS "QQNAME",
       a.position AS "NOPOLIS",
       'YUNIARTAULIASI' AS "THEINSURED",
       a.idpega AS "IDPEGA",
       a.jenis_klaim AS "BUSINESSCODE",
       TO_CHAR(a.timein, 'dd/mm/rrrr hh24:mi:ss') AS "SOBLEADER0",
       TO_CHAR(a.timeout, 'dd/mm/rrrr hh24:mi:ss') AS "SOBLEADER1",
       FLOOR(a.aging / 28800) AS "SOBLEADER2",
       CASE
           WHEN FLOOR(a.aging / 28800) <= 1 THEN 'Y'
           ELSE 'N'
       END AS "AUTOCANCELPRINTSTATUS",
       ROUND(
           (
               (SELECT COUNT(*)
                  FROM pooldata.PNC_CHRONOLOGYTAT
                 WHERE userassign = a.userassign
                   AND FLOOR(aging / 28800) <= 1)
               /
               (SELECT COUNT(*)
                  FROM pooldata.PNC_CHRONOLOGYTAT
                 WHERE userassign = a.userassign)
           ) * 100
       ) AS "BUSINESSNAME",
       (SELECT COUNT(*)
          FROM pooldata.PNC_CHRONOLOGYTAT
         WHERE userassign = a.userassign) AS "BRANCHCODE",
       (SELECT COUNT(*)
          FROM pooldata.PNC_CHRONOLOGYTAT
         WHERE userassign = a.userassign
           AND FLOOR(aging / 28800) <= 1) AS "BRANCHNAME",
       (SELECT COUNT(*)
          FROM pooldata.PNC_CHRONOLOGYTAT
         WHERE userassign = a.userassign
           AND FLOOR(aging / 28800) > 1) AS "ACCUMCODE"
  FROM pooldata.PNC_CHRONOLOGYTAT a
 WHERE a.insertdate >= :1
   AND a.insertdate < :2 + INTERVAL '1' DAY
 ORDER BY a.userassign, a.timein


-- name: report_adjuster
--
-- Survei yang sudah SELESAI, gabungan surveyor internal dan eksternal.
--
-- Asal: `Activity/PNCAdjusterReport_Act-Act.xml`.
--
-- Bind:
--   :1  tanggal dari    DATE
--   :2  tanggal sampai  DATE
--
-- ============================================================================
-- Kueri ini DIREKONSTRUKSI, bukan disalin
-- ============================================================================
--
-- Kedua Report Definition yang dipanggil activity-nya — `InboxSurveyClose_rd` dan
-- `InboxInternalSurveyClose_rd` — TIDAK ADA di export (`R-16`). Yang tersedia hanya
-- pemetaan kolomnya di activity itu, dan itu sudah lengkap: kesepuluh kolom laporan
-- diketahui berikut properti sumbernya.
--
-- Yang harus direkonstruksi adalah PENYARINGNYA. Bahan rekonstruksinya
-- `RDB List/BrowseInternalSurveyor-SQL.xml` — inbox survei yang BELUM selesai, dibaca
-- terbalik.
--
-- Empat hal yang diambil dari sana apa adanya:
--
--   tabelnya              DATAPEGA.PC_ASM_FW_GCNMFW_WORK (kini diganti — lihat SUMBER BARU)
--   pembeda kelasnya      PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim'
--   pembeda internal      SURVEYORTYPE_1 = '1'
--   keterkaitan klaimnya  CASEID_1 = pzinskey klaim induknya
--
-- ============================================================================
-- TIGA hal yang masih SIMPULAN, dan harus dikonfirmasi
-- ============================================================================
--
--  1. **Arti "Close".** `BrowseInternalSurveyor` menyaring
--     `PYSTATUSWORK NOT IN ('Resolved-Completed','Resolved-Rejected')` untuk inbox yang
--     masih berjalan. Laporan ini memakai kebalikannya. Dasarnya kuat, tetapi ia tetap
--     kebalikan yang disimpulkan — bukan yang terbaca dari rule-nya sendiri.
--
--  2. **Kolom yang disaring periode.** Dipilih `SURVEYDATE_1`, karena kolom tanggal pada
--     laporannya sendiri berjudul "Tgl Pengajuan Survey". Bila Report Definition-nya
--     ternyata menyaring `pxCreateDateTime`, hasilnya akan berbeda pada survei yang
--     dibuat dan dijadwalkan di bulan yang berlainan.
--
--  3. **Rumus "Nilai Reserve Klaim ASM".** Activity-nya membaca `.ClaimData.ClaimEstimate`
--     — properti klipboard yang tidak punya kolom sendiri. Yang dipakai di sini
--     `SUM(convertvalue)` dari `T_CLAIM_ESTIMASI`, mengikuti rumus yang sudah dipakai
--     modul ini untuk kolom berjudul "Reserve" (lihat report_klaim_harian).
--
-- Ketiganya TIDAK dapat diselesaikan tanpa kedua Report Definition itu, dan ketiganya
-- mengubah ISI laporan — bukan bentuknya.
--
-- ============================================================================
-- Kedua jenis surveyor DIGABUNG, dan itu memang perilakunya
-- ============================================================================
--
-- Activity-nya memanggil KEDUA Report Definition berurutan lalu menyalin hasilnya ke
-- daftar yang SAMA (`TempExportAdjuster.pxResults(<APPEND>)`). Jadi berkasnya memuat
-- survei internal dan eksternal sekaligus, tanpa kolom yang membedakan keduanya.
-- Karena itu `SURVEYORTYPE_1` tidak disaring di sini.
--
-- ============================================================================
-- SUMBER BARU (2026-10-08) — objek kerja SurveyClaim diganti T_SURVEYORLIST
-- ============================================================================
--
-- `PC_ASM_FW_GCNMFW_WORK` tidak dipakai lagi. Satu berkas survei kini = LANGKAH TERAKHIR
-- per `CASEID` di `T_SURVEYORLIST` (tabel itu jejak per langkah), klaimnya `T_CLAIM_PNC`
-- lewat `PNCCASEID`. Pemetaan dan kecocokannya, diukur atas 187 berkas yang ada di kedua
-- sumber (Oracle dev):
--
--   PYSTATUSWORK      -> t.PYSTATUSWORK       187/187 sama
--   CASEID_1          -> t.PNCCASEID          156 sama; 31 sisanya CASEID_1 KOSONG di
--                                             objek kerja (T_SURVEYORLIST justru terisi)
--   PYID klaim        -> REPLACE(t.PNCCASEID, prefix Pega, '')  (kesepadanan PYID Work-PNC)
--   SURVEYORNAME_1    -> t.SURVEYOR_NAME      173/175 terisi sama
--   ADJUSTERSTATUS_1  -> t.STS_SURVEY         155/187 sama
--   USERTEKNIS_1      -> c.PICTEKNIK          114/150 sama (ADJUSTER_PIC 0 sama)
--   POLICYNO, QQNAME  -> c.NOPOLIS, c.QQNAME  170/187 sama
--   KETERANGAN_1      -> t.KETERANGAN         BELUM TERBUKTI — di objek kerja 0/187 terisi
--   SURVEYDATE_1      -> t.SURVEYDATE         BELUM TERBUKTI — hanya 1 dari 29 sama harinya;
--                                             dipakai karena tidak ada kolom lain yang lebih
--                                             dekat, dan tanpa tanggal laporan ini kosong
--
-- Akibat pada populasi: objek kerja memuat 367 survei, T_SURVEYORLIST hanya 188 berkas.
-- Yang selesai: 134 lama vs 48 baru; yang selesai DAN bertanggal survei: 7 lama vs 13 baru.
SELECT REPLACE(t.PNCCASEID, 'ASM-FW-GCNMFW-WORK ', '') AS "IDSurvey",
       c.NOPOLIS AS "IDObject",
       REPLACE(c.QQNAME, ',', ' ') AS "InsuredPIC",
       c.BUSINESSNAME AS "CouseOfLos",
       (SELECT SUM(e.CONVERTVALUE)
          FROM POOLDATA.T_CLAIM_ESTIMASI e
         WHERE e.CLAIMID = t.PNCCASEID) AS "Salvage",
       t.SURVEYDATE AS "BodyLetterOP",
       t.SURVEYOR_NAME AS "SurveyorName",
       t.KETERANGAN AS "KeteranganLain",
       c.PICTEKNIK AS "AdjusterPIC",
       t.STS_SURVEY AS "AdjusterStatus"
  FROM (SELECT l.CASEID, l.PNCCASEID, l.PYSTATUSWORK, l.SURVEYDATE, l.SURVEYOR_NAME,
               l.KETERANGAN, l.STS_SURVEY,
               ROW_NUMBER() OVER (PARTITION BY l.CASEID
                                  ORDER BY LPAD(TRIM(l.INDEX_SURVEY), 10, '0') DESC NULLS LAST,
                                           l.TGLINPUT DESC NULLS LAST) AS rn
          FROM POOLDATA.T_SURVEYORLIST l) t
  LEFT JOIN POOLDATA.T_CLAIM_PNC c ON c.CLAIMID = t.PNCCASEID
 WHERE t.rn = 1
   AND t.PYSTATUSWORK IN ('Resolved-Completed', 'Resolved-Rejected')
   AND t.SURVEYDATE >= :1
   AND t.SURVEYDATE < :2 + INTERVAL '1' DAY
 ORDER BY t.SURVEYDATE, REPLACE(t.CASEID, 'ASM-FW-GCNMFW-WORK ', '')


-- name: report_compliance
--
-- Klaim Personal Accident beserta nilai propose dan nilai dibayarnya.
--
-- Asal: `Activity/PNCComplianceReport_Act-Act.xml`.
--
-- Bind:
--   :1  tanggal dari    DATE
--   :2  tanggal sampai  DATE
--
-- ============================================================================
-- DUA kolom sengaja TIDAK diambil, dan penyaring statusnya TIDAK ada
-- ============================================================================
--
-- Activity aslinya membaca tiga hal dari klipboard Pega, bukan dari tabel:
--
--     .ClaimData.ComplianceList(<LAST>).Compliance      -> "Komentar Compliance"
--     .ClaimData.ComplianceList(<LAST>).ComplianceDate  -> "Tanggal Compliance"
--     .ClaimData.PilihanCompliance                      -> penyaring Status Compliance
--
-- Ketiganya properti yang BELUM dioptimasi. Dipastikan tiga kali: keenam rule Property-nya
-- tidak punya `pyColumnInclusion`; kueri katalog atas `PC_ASM_FW_GCNMFW_WORK` nol kolom
-- ber-`%COMPLIANCE%`; dan kelas `ASM-FW-GCNMFW-Work-Compliance` ternyata
-- `belongs to a class group` — jadi ia menumpang tabel yang sama, tidak punya tabel sendiri.
--
-- Kedua kolom komentar karena itu dibiarkan KOSONG di berkas, bukan dihapus: bentuk
-- berkasnya tetap 12 kolom seperti Pega, sehingga dapat dibandingkan berdampingan.
--
-- **Penyaring status tidak ada.** Berkasnya memuat SELURUH klaim PA pada periode itu, bukan
-- hanya yang berstatus tertentu. Itu perbedaan yang mengubah jumlah baris, dan karena itu
-- dinyatakan di LABEL TOMBOLNYA — bukan disembunyikan di dokumen.
--
-- ============================================================================
-- Yang ditiru apa adanya
-- ============================================================================
--
-- `GROUP_PANEL = '002'`. Activity-nya memasang `Param.GroupPanel = "002"` sebagai nilai
-- tetap: laporan ini memang laporan Personal Accident saja, dan dropdown lini bisnis tidak
-- berpengaruh padanya.
--
-- ============================================================================
-- Dua kolom nilai — rumusnya SIMPULAN
-- ============================================================================
--
-- Activity membaca `.EstimationValue` dan `.AdjustmentValue` dari halaman adjustment yang
-- sedang diputar. Padanan tabelnya dipilih mengikuti rumus yang sudah dipakai modul ini di
-- laporan lain: estimasi dari `T_CLAIM_ESTIMASI`, nilai dibayar dari `T_CLAIM_ADJUSTMENT`.
-- Keduanya perlu dicocokkan ke Pega sebelum dipercaya.
SELECT a.CLAIMNO AS "CaseID",
       a.NOPOLIS AS "ClaimNo",
       (SELECT g.STARTDATE
          FROM POOLDATA.T_GENERAL g
         WHERE g.NOPOLIS = a.NOPOLIS
           AND g.PRODKE = a.PRODKE) AS "AnaylstRemarks",
       (SELECT g.ENDDATE
          FROM POOLDATA.T_GENERAL g
         WHERE g.NOPOLIS = a.NOPOLIS
           AND g.PRODKE = a.PRODKE) AS "CountryID",
       a.DATEOFLOSS AS "AnalystDoctorRemaks",
       a.QQNAME AS "CityID",
       a.LOCATION AS "Location",
       (SELECT MIN(o.OBJECTNAME)
          FROM POOLDATA.T_CLAIM_OBJECTLIST o
         WHERE o.CLAIMID = a.CLAIMID) AS "ReporterName",
       (SELECT SUM(e.ESTIMATIONVALUE)
          FROM POOLDATA.T_CLAIM_ESTIMASI e
         WHERE e.CLAIMID = a.CLAIMID) AS "ClaimEstimate",
       (SELECT SUM(x.GROSSVALUE)
          FROM POOLDATA.T_CLAIM_ADJUSTMENT x
         WHERE x.CLAIMID = a.CLAIMID) AS "Country"
  FROM POOLDATA.T_CLAIM_PNC a
-- Kolomnya `GROUPPANEL` tanpa garis bawah. `GROUP_PANEL` memang ada, tetapi di
-- `PEGA_DASHBOARDPNC` — bukan di tabel ini. Keduanya dipakai berdampingan di modul ini,
-- dan nama yang nyaris sama inilah yang membuatnya tertukar.
 WHERE a.GROUPPANEL = '002'
   AND a.REGISTERDATE >= :1
   AND a.REGISTERDATE < :2 + INTERVAL '1' DAY
 ORDER BY a.REGISTERDATE, a.CLAIMNO
