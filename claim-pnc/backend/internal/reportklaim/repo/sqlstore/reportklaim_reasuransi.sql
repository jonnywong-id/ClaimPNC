-- Kueri laporan kelompok PLA / DLA.
--
-- ============================================================================
-- Lima aturan yang mengikat SELURUH berkas kueri modul ini
-- ============================================================================
--
--   1. Kolom disebut namanya; SELECT * dilarang.
--
--   2. Nilai selalu lewat parameter binding. Kueri aslinya menyisipkan penyaring lewat
--      `{ASIS:TempLaporan.UserTeknis}` — celah injeksi yang TIDAK dibawa. Penggantinya
--      ada di bawah: potongan penyaringnya ditulis di dalam kueri sebagai cabang yang
--      dipilih SATU bind, bukan dirangkai dari teks.
--
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR untuk tampilan (`D-20`).
--      Kolom tanggal dikembalikan sebagai DATE dan diformat di Go — satu tempat, zona
--      WIB. Lihat fungsi text pada reportklaim.go.
--
--   4. Tanpa pemanggilan stored procedure (`D-02`).
--
--   5. Setiap kolom dialiaskan ke NAMA PROPERTI pada `CSVProperties` milik langkah
--      `pxConvertResultsToCSV`. Itulah yang membuat satu pemindai melayani 26 kueri,
--      dan yang membuat satu kolom dapat ditelusuri dari judul di berkas sampai ke
--      kolom tabelnya tanpa tabel penerjemah.
--
-- ============================================================================
-- Kenapa daftar kode bisnis ditulis sebagai literal, dan itu BUKAN pelanggaran
-- ============================================================================
--
-- Daftar `businesscode` di bawah adalah ATURAN BISNIS yang tertulis di dalam rule Pega,
-- bukan nilai yang datang dari pengguna. Ia konstanta, sama seperti kode Group Panel.
-- Yang dilarang `08-TECHNICAL-STRATEGY.md` §4.3 adalah merangkai NILAI PENGGUNA ke dalam
-- teks SQL — dan tidak satu pun nilai pengguna menyentuh kueri ini di luar bind.
--
-- Daftar itu BERBEDA-BEDA antar laporan, dan perbedaannya nyata: laporan Close memakai
-- enam kode yang dikecualikan, laporan TAT memakai sepuluh. Menyatukannya menjadi satu
-- daftar bersama akan mengubah isi beberapa laporan sekaligus tanpa satu pun tanda.


-- name: report_pla
--
-- REPORT DATA PLA — 12 kolom.
--
-- Asal: `RDB List/ExportDataPLA-SQL.xml`, dijalankan `Activity/PNCReportDataPLA_act-Act.xml`.
--
-- Bind:
--   :1  tanggal PLA dari       DATE
--   :2  tanggal PLA sampai     DATE
--   :3  kode lini bisnis       '' | '002' | '005' | '346' | '003'
--
-- Rentangnya atas TANGGAL PLA, bukan tanggal registrasi maupun tanggal kejadian.
SELECT b.claimno          AS "CaseID",
       b.nopolis          AS "NoKTP",
       b.businessname     AS "ClaimID",
       b.qqname           AS "ClaimNo",
       a.nopla            AS "Conveyance",
       a.tglpla           AS "Country",
       EXTRACT(MONTH FROM a.tglpla) AS "District",
       a.plareinsurer     AS "CountryID",
       b.coinsname        AS "Email",
       b.dateofloss       AS "RefNo",
       b.registerdate     AS "Remark",
       b.picteknik        AS "RW"
  FROM t_plalist a
  JOIN t_claim_pnc b ON a.claimid = b.claimid
 WHERE CAST(a.tglpla AS DATE) >= :1
   AND CAST(a.tglpla AS DATE) <= :2
   AND (
        :3 = ''
     OR (:3 = '002' AND b.grouppanel = '002')
     OR (:3 = '005' AND b.grouppanel = '005')
     OR (:3 = '346' AND b.grouppanel IN ('003','004','006','009')
                    AND b.businesscode NOT IN ('10008','10010','10015','10023','10168','10145'))
     OR (:3 = '003' AND b.grouppanel = '003'
                    AND b.businesscode IN ('10076','10077','10007','10011','10083','10141','10131','10126','10055','10075'))
       )
 ORDER BY a.tglpla ASC


-- name: report_dla
--
-- REPORT DATA DLA — 16 kolom.
--
-- Asal: `RDB List/ExportDataDLA-SQL.xml`, dijalankan `Activity/PNCReportDataDLA_act-Act.xml`.
--
-- Bind:
--   :1  tanggal DLA dari       DATE
--   :2  tanggal DLA sampai     DATE
--   :3  kode lini bisnis
--
-- Daftar kode Bonding di sini DUA KODE LEBIH PANJANG daripada laporan PLA di atas —
-- '10053' dan '10168' ikut. Perbedaannya ada di rule aslinya, bukan salah salin.
SELECT b.claimno      AS "CaseID",
       b.nopolis      AS "NoKTP",
       b.businessname AS "ClaimID",
       b.qqname       AS "ClaimNo",
       a.nodla        AS "Conveyance",
       a.tgldla       AS "Country",
       a.dlareinsurer AS "CountryID",
       a.noaksep      AS "District",
       a.nilaidla     AS "Province",
       b.coinsname    AS "Email",
       b.dateofloss   AS "RefNo",
       b.registerdate AS "Remark",
       b.picteknik    AS "RW",
       a.revisi       AS "NIK",
       EXTRACT(MONTH FROM a.tgldla) AS "Location",
       (SELECT c.tglakseptasi
          FROM t_claim_adjustment c
         WHERE c.claimid = a.claimid
           AND c.objectid = a.objectid
           AND c.adjustmentid = a.adjustmentid) AS "DistrictID"
  FROM t_dlalist a
  JOIN t_claim_pnc b ON a.claimid = b.claimid
 WHERE CAST(a.tgldla AS DATE) >= :1
   AND CAST(a.tgldla AS DATE) <= :2
   AND (
        :3 = ''
     OR (:3 = '002' AND b.grouppanel = '002')
     OR (:3 = '005' AND b.grouppanel = '005')
     OR (:3 = '346' AND b.grouppanel IN ('003','004','006','009')
                    AND b.businesscode NOT IN ('10008','10010','10015','10023','10168','10145'))
     OR (:3 = '003' AND b.grouppanel = '003'
                    AND b.businesscode IN ('10053','10168','10076','10077','10007','10011','10083','10141','10131','10126','10055','10075'))
       )
 ORDER BY a.tgldla ASC


-- name: report_pengiriman_pla
--
-- REPORT DATA PENGIRIMAN PLA — 12 kolom.
--
-- Asal: `RDB List/ExportDataPengirimanPLA-SQL.xml`.
--
-- Bind:
--   :1  tanggal PLA dari       DATE
--   :2  tanggal PLA sampai     DATE
--
-- TANPA penyaring lini bisnis, dan itu koreksi yang disengaja — kueri aslinya memuat
-- `{ASIS:TempLaporan.UserTeknis}` sementara activity-nya tidak pernah mengisi properti
-- itu, sehingga yang tersisip adalah sisa dari laporan yang dijalankan sebelumnya.
-- Alasan lengkapnya di catalog_reasuransi.go.
--
-- (Riwayat) Keempat anak-kueri ke `datapega.pc_asm_fw_gcnmfw_work` diganti satu JOIN: keduanya
-- membaca baris YANG SAMA, dan empat anak-kueri berarti empat kali pembacaan tabel yang
-- dibaca 116 rule Pega. Hasilnya identik karena `pzinskey` unik.
--
-- SUMBER BARU (2026-10-08): JOIN `w` ke objek kerja Pega diganti `T_CLAIM_PNC` (`c`),
-- dengan cadangan `T_CLAIMLIST_ADMIN` (`k`, baris Work-PNC). Pemetaan:
--   POLICYNO -> c.NOPOLIS · QQNAME -> c.QQNAME · DATEOFLOSS_1 -> c.DATEOFLOSS (DATE) ·
--   USERTEKNIS_1 -> c.PICTEKNIK · PYID -> ekor x.CLAIMID tanpa prefix Pega.
-- Diukur atas 1.533 baris PLA (Oracle dev): T_CLAIM_PNC memuat klaimnya pada 1.416 baris
-- dan nilainya sama dengan objek kerja pada 1.400 (PICTEKNIK 1.396). Sisa 117 baris
-- (4 klaim) belum ada di T_CLAIM_PNC tetapi ada di T_CLAIMLIST_ADMIN — itu gunanya `k`.
-- PYID dari CLAIMID cocok 1.532/1.532. Jumlah baris tidak berubah (keduanya LEFT JOIN).
SELECT v.nopla        AS "City",
       v.nilaipla     AS "Amount",
       COALESCE(c.nopolis, k.policyno) AS "AlasanKlaim",
       COALESCE(c.qqname, k.qqname)    AS "Keyword",
       COALESCE(c.dateofloss, CAST(k.dateofloss_1 AS DATE)) AS "Other",
       x.plareinsurer AS "CityID",
       REPLACE(x.claimid, 'ASM-FW-GCNMFW-WORK ', '') AS "CaseID",
       x.tglpla       AS "District",
       x.tglkirim     AS "DistrictID",
       x.tglterimapla AS "Country",
       CASE
         WHEN x.iskirim IS NULL THEN 'Belum Dikirim'
         WHEN x.iskirim = '1'   THEN 'Sudah Dikirim'
       END            AS "CountryID",
       COALESCE(c.picteknik, k.userteknis_1) AS "UserTeknis"
  FROM POOLDATA.t_plalist x
  JOIN POOLDATA.VIEW_PLALIST v ON x.nopla = v.nopla
  LEFT JOIN POOLDATA.t_claim_pnc c ON c.claimid = x.claimid
  LEFT JOIN POOLDATA.t_claimlist_admin k
         ON k.pzinskey = x.claimid
        AND k.pxobjclass = 'ASM-FW-GCNMFW-Work-PNC'
 WHERE CAST(x.tglpla AS DATE) >= :1
   AND CAST(x.tglpla AS DATE) <= :2
 ORDER BY x.tglpla ASC


-- name: report_pengiriman_dla
--
-- REPORT DATA PENGIRIMAN DLA — 9 kolom.
--
-- Asal: `RDB List/ExportDataPengirimanDLA-SQL.xml`.
--
-- Bind:
--   :1  tanggal DLA dari       DATE
--   :2  tanggal DLA sampai     DATE
--
-- Kueri aslinya memang tidak memuat penyaring lini bisnis sama sekali — berbeda dari
-- ketiga tetangganya, dan berbeda pula sebabnya dari laporan Pengiriman PLA di atas.
SELECT b.claimno      AS "CaseID",
       a.nodla        AS "City",
       a.dlareinsurer AS "CityID",
       a.nilaidla     AS "Amount",
       a.tgldla       AS "District",
       a.tglkirim     AS "DistrictID",
       a.tglterimadla AS "Country",
       b.picteknik    AS "Keyword",
       CASE
         WHEN a.iskirim IS NULL THEN 'Belum Dikirim'
         WHEN a.iskirim = '1'   THEN 'Sudah Dikirim'
       END            AS "CountryID"
  FROM t_dlalist a
  JOIN t_claim_pnc b ON a.claimid = b.claimid
 WHERE CAST(a.tgldla AS DATE) >= :1
   AND CAST(a.tgldla AS DATE) <= :2
 ORDER BY a.tgldla ASC
