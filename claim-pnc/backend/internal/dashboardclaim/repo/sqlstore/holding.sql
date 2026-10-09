-- Kueri tab **Inbox Tampungan PIC**.
--
-- Menggantikan `RDB List/BrowseCaseNotAssigned-SQL.xml` dan
-- `RDB List/CountCaseNotAssigned-SQL.xml` — tab kedua pada
-- `Harness/DashboardClaim_Harness-Harness.xml`, bersebelahan dengan tab Dashboard Claim.
--
-- ============================================================================
-- APA ISI "TAMPUNGAN", DAN KENAPA TIGA SYARAT SEKALIGUS
-- ============================================================================
--
-- Ia klaim yang SUDAH terdaftar tetapi BELUM punya PIC Teknik. Ketiga syaratnya bekerja
-- bersama, dan tidak satu pun cukup sendirian:
--
--   b.PXASSIGNEDOPERATORID = 'ServicePNC'   tugasnya memang sedang di penampungan
--   A.USERTEKNIS_1 IS NULL                  belum ada PIC Teknik yang memegangnya
--   A.POLICYNO IS NOT NULL                  polisnya sudah terisi — bukan draf
--
-- Operator `ServicePNC` di sini BUKAN orang. Ia akun penampung tempat tugas diparkir sampai
-- seseorang mengambilnya; di sistem baru ia tetap dibandingkan apa adanya (`P-5`), meski
-- nilainya layak menjadi master data kelak (`D-15`).
--
-- ============================================================================
-- TAB INI TIDAK MENGIKUTI PENYARING LINI BISNIS
-- ============================================================================
--
-- Kueri lamanya tidak punya penanda lini bisnis sama sekali, dan layar lamanya memang tidak
-- menggambar dropdown Bisnis pada tab ini — dropdown itu hanya ada di tab Dashboard Claim.
--
-- Satu-satunya penyaringnya adalah kotak cari No Klaim (`{ASIS:TempCari.AlasanKlaim}`), yang
-- di sini menjadi parameter binding.
--
-- ============================================================================
-- YANG BERUBAH DARI KUERI LAMA
-- ============================================================================
--
--   `{ASIS:…}` tiga tempat          -> parameter binding
--   ROWNUM dua lapis                -> OFFSET … FETCH NEXT (`D-20`)
--   TO_CHAR(…, 'dd/mm/yyyy')        -> tanggal dikembalikan apa adanya, diformat di Go
--   gabung koma lama                -> INNER JOIN eksplisit
--
-- Alias menyesatkan yang diberi nama benar (`D-19`):
--
--     A.POLICYNO        AS "City"       -> NO_POLIS
--     A.QQNAME          AS "CityID"     -> NAMA_TERTANGGUNG
--     A.BUSINESSNAME    AS "Country"    -> NAMA_BISNIS
--     A.SOBNAME         AS "CountryID"  -> SUMBER_BISNIS
--     A.BRANCHNAME      AS "District"   -> NAMA_CABANG
--     A.PXCREATEOPNAME  AS "DistrictID" -> ADMIN_PNC
--     PXCREATEDATETIME  AS "Province"   -> TANGGAL_PENDAFTARAN
--
-- Ketujuhnya memakai nama geografis untuk data yang sama sekali bukan geografis.
--
-- ============================================================================
-- SUMBER BARU (2026-10-08)
-- ============================================================================
--
-- Objek kerja Pega (`DATAPEGA` objek kerja, alias A lama) tidak dipakai lagi (keputusan Work
-- Owner). Kepala klaim kini dibaca dari POOLDATA.T_CLAIMLIST_ADMIN — nama kolomnya sama persis
-- (PYID, POLICYNO, QQNAME, BUSINESSNAME, SOBNAME, BRANCHNAME, PXCREATEOPNAME,
-- PXCREATEDATETIME, USERTEKNIS_1, PXOBJCLASS), sehingga alias dan bind tidak berubah.
-- Gabungannya ke worklist kini `A.PZINSKEY = B.PXREFOBJECTKEY` (tabel baru tidak punya
-- PXINSNAME).
--
-- Diukur di Oracle dev: dari 103 tugas Work-PNC milik `ServicePNC`, 101 punya baris di
-- T_CLAIMLIST_ADMIN (T_CLAIM_PNC hanya 20 — karena itu BUKAN sumbernya); kesembilan kolom di
-- atas sama dengan objek kerja pada 101/101 baris; populasi tampungan lama 57 dan baru 57,
-- selisih himpunan 0.

-- name: holding_count
-- Menghitung SELURUH klaim di penampungan yang cocok.
--
-- Syarat WHERE-nya wajib sama persis dengan holding_list; `query_test.go` menjaganya.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST B
               ON A.PZINSKEY = B.PXREFOBJECTKEY
 WHERE B.PXASSIGNEDOPERATORID = 'ServicePNC'
   AND A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.USERTEKNIS_1 IS NULL
   AND A.POLICYNO IS NOT NULL
   AND (:1 IS NULL OR UPPER(A.PYID) LIKE :2 ESCAPE '\')

-- name: holding_list
-- Membaca satu halaman klaim di penampungan.
--
-- Urutannya `PXCREATEDATETIME ASC` ditambah PZINSKEY sebagai pemutus seri. Kueri lama tidak
-- menyebut ORDER BY sama sekali, sehingga urutan barisnya diserahkan pada basis data — dan
-- urutan yang tidak ditetapkan membuat satu baris dapat tampil di dua halaman sekaligus
-- sementara baris lain tidak pernah tampil.
--
-- Menetapkan urutan karena itu BUKAN perubahan perilaku melainkan syarat agar paginasinya
-- benar sama sekali.
SELECT A.PZINSKEY         AS ID_KLAIM,
       A.PYID             AS NO_KLAIM,
       A.POLICYNO         AS NO_POLIS,
       A.QQNAME           AS NAMA_TERTANGGUNG,
       A.BUSINESSNAME     AS NAMA_BISNIS,
       A.SOBNAME          AS SUMBER_BISNIS,
       A.BRANCHNAME       AS NAMA_CABANG,
       A.PXCREATEOPNAME   AS ADMIN_PNC,
       A.PXCREATEDATETIME AS TANGGAL_PENDAFTARAN
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST B
               ON A.PZINSKEY = B.PXREFOBJECTKEY
 WHERE B.PXASSIGNEDOPERATORID = 'ServicePNC'
   AND A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.USERTEKNIS_1 IS NULL
   AND A.POLICYNO IS NOT NULL
   AND (:1 IS NULL OR UPPER(A.PYID) LIKE :2 ESCAPE '\')
 ORDER BY A.PXCREATEDATETIME ASC, A.PZINSKEY
OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY
