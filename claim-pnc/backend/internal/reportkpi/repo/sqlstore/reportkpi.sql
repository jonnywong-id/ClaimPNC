-- Kueri modul Report KPI PNC (`MENU_ID 84`, pengganti `Harness/ReportKPIHarness`),
-- tab **KPI Adjuster**.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- SELURUH pernyataan di sini MEMBACA. Tidak ada satu pun yang menulis, dan memang tidak
-- boleh ada: `POOLDATA.DETAIL_KPI_ADJUSTER` diisi Pega lewat
-- `Database/INSERT_KPIADJUSTER.prc`, dan selama masa paralel tepat satu sistem yang boleh
-- menulis sebuah tabel (`P-1`).
--
-- ============================================================================
-- SATU TABEL, DAN ITULAH SELURUHNYA
-- ============================================================================
--
-- Ketiga kueri Pega yang menjadi rujukan berkas ini — `GetSummaryKPIAdjuster`,
-- `GetSummaryKPIAdjusterALL`, `GetSummaryKPIAdjusterKuartal` — sama-sama membaca
-- `pooldata.DETAIL_KPI_ADJUSTER` dan tidak satu pun menggabungkannya ke tabel lain.
--
-- Itu menjelaskan kenapa tab ini dipilih lebih dulu: ia tidak menyentuh
-- `DATAMINING.GET_WORKING_HOURS@ASMD` (DB link, `R-03`) maupun `POOLDATA.M_KPI_PNC`
-- (isinya tidak ada di export, `R-16`) — keduanya menghalangi dua tab lainnya.
--
-- ============================================================================
-- PEMETAAN KOLOM — kolom sebenarnya -> alias Pega -> alias di sini
-- ============================================================================
--
-- Nama kolomnya dipastikan dari `Database/INSERT_KPIADJUSTER.prc`, satu-satunya artefak
-- yang MENULIS tabel ini dan karena itu menyebut ketiga belas kolomnya lengkap. Alias
-- Pega diambil dari `RDB List/GetSummaryKPIAdjuster-SQL.xml`.
--
-- Perhatikan betapa jauh alias Pega dari isinya — utang teknis §4.2 yang `D-19` hapus:
--
--   kolom              alias Pega        alias di sini        judul layar
--   ------------------ ----------------- -------------------- ----------------------
--   ADJUSTER           UserTeknisGroup   ADJUSTER             ADJUSTER
--   CASEID             (tidak diambil)   CASE_ID              NO CASE
--   TIPE               StatusWork        REPORT_TYPE          TIPE
--   TANGGAL            (tidak diambil)   SCORED_ON            TANGGAL
--   SURVEYLAP          ProdKe            SURVEY               PENJADWALAN SURVEY
--   IMMEDIATEADVICE    CityID            IMMEDIATE_ADVICE     IMMEDIATE ADVICE
--   PRELIMINARYADVICE  ResponseNote      PRELIMINARY_ADVICE   PRELIMINARY ADVICE
--   INTERIM            ReporterName      INTERIM_REPORT       INTERIM REPORT
--   PROGRESS           CountryID         PROGRESS             UPDATE PROGRESS
--   KOMUNIKASI         RWID              COMMUNICATION        TANGGAPAN KOMUNIKASI
--   PROPOSE            AlasanKlaim       PROPOSE              PROPOSE ADJUSTMENT
--   FINALREPORT        ProvinceID        FINAL_REPORT         FINAL REPORT
--   NILAI              Keyword           TOTAL_SCORE          NILAI
--
-- `UserTeknisGroup` untuk nama adjuster dan `AlasanKlaim` untuk nilai propose adalah dua
-- yang paling menyesatkan: keduanya menyebut hal yang sama sekali berbeda dari isinya.
--
-- ============================================================================
-- KENAPA to_number DIBUNGKUS, DAN KENAPA BENTUKNYA DIBIARKAN APA ADANYA
-- ============================================================================
--
-- Kesembilan komponennya disimpan sebagai TEKS — `INSERT_KPIADJUSTER.prc` mendeklarasikan
-- seluruhnya `in varchar2`. Kueri Pega karena itu membungkus setiap satunya dengan
-- `to_number(...)` sebelum merata-ratakan, dan itu ditiru di sini KATA DEMI KATA.
--
-- Oracle 12.2+ menyediakan `TO_NUMBER(x DEFAULT NULL ON CONVERSION ERROR)`, yang akan
-- membuat satu baris rusak diabaikan alih-alih menggagalkan laporan. Ia sengaja TIDAK
-- dipakai, dan alasannya bukan kelalaian:
--
--   1. Ia mengubah ANGKA, bukan hanya penanganan galat. `AVG` mengabaikan NULL, sehingga
--      baris rusak akan hilang dari pembagi dan rata-ratanya BERGESER tanpa seorang pun
--      tahu. `P-5` menuntut hasil yang sama dengan Pega kecuali untuk perbaikan yang
--      diputuskan eksplisit, dan ini bukan salah satu dari 13 butir `D-49`.
--   2. Kegagalan `ORA-01722` itu KERAS dan terlihat. Angka yang bergeser diam-diam jauh
--      lebih berbahaya pada laporan penilaian kinerja daripada laporan yang menolak
--      tampil.
--   3. Ia khas Oracle 12.2+, sedangkan `D-20` menetapkan satu set SQL yang juga berjalan
--      di PostgreSQL 17. Menambahnya berarti menambah pengecualian portabilitas keempat
--      tanpa keputusan tertulis.
--
-- ============================================================================
-- PENYARING
-- ============================================================================
--
-- Ketiganya diambil dari `Activity/PNCReportKPIAdjuster_act-Act.xml`, yang merangkainya
-- sebagai potongan teks SQL lalu menyisipkannya lewat `{ASIS:...}`:
--
--   TempAdjComp.ASMFull       -> where tipe = <pilihan>
--   TempAdjComp.UploadLOD     -> "and adjuster='" + TempAdjComp.NameOfBank + "'"
--   TempAdjComp.CaseIDKomite  -> "and trunc(TANGGAL)>=to_date('" + awal + "','dd/mm/yyyy')
--                                  and trunc(TANGGAL)<=to_date('" + akhir + "','dd/mm/yyyy')"
--
-- Ketiganya dirangkai dari nilai yang datang dari layar. Di sini seluruhnya lewat
-- PARAMETER BINDING tanpa perkecualian — itu yang menutup celah `{ASIS:...}` warisan
-- (`08-TECHNICAL-STRATEGY.md` §4.3, utang teknis §4.5).
--
-- DUA hal pada penyaring tanggal berbeda bentuknya dari sistem lama, dan keduanya
-- menghasilkan baris yang SAMA:
--
--   `TRUNC(TANGGAL) <= to_date(akhir)` ditulis sebagai `TANGGAL < akhir + 1 hari`.
--   `TRUNC` pada KOLOM mematikan index-nya, dan `D-20` mendaftarkannya sebagai bentuk
--   khas Oracle yang diganti. Bentuk setengah terbuka ini pula yang sudah dipakai modul
--   Inbox RCL/PUCL, sehingga kedua modul menyaring tanggal dengan cara yang sama.
--
--   Tanggal dikirim sebagai teks `YYYY-MM-DD`, bukan `dd/mm/yyyy`. Bentuk ISO tidak dapat
--   dibaca terbalik sebagai bulan-tanggal, dan kekeliruan seperti itu menghasilkan rentang
--   yang SAH tetapi salah — tanpa satu pun galat.

-- name: summary
-- Grid "Summary KPI Adjuster" — satu baris per adjuster, kesembilan nilai dirata-ratakan.
-- — RDB List/GetSummaryKPIAdjuster-SQL.xml (satu tipe)
-- — RDB List/GetSummaryKPIAdjusterALL-SQL.xml (tipe ALL, UNION ALL dua kelompok)
--
-- SATU kueri melayani ketiga tipe, dan itu perbedaan bentuk yang disengaja terhadap Pega
-- yang memakai dua rule terpisah. Yang membedakan keduanya di sana hanyalah `UNION ALL`
-- dan kolom tipe yang ditulis sebagai literal; di sini keduanya jatuh dari `GROUP BY`
-- terhadap kolom `TIPE` yang memang ada di tabelnya.
--
-- Akibatnya satu hal yang perlu disadari: pada tipe ALL, Pega selalu menghasilkan kedua
-- kelompok meski salah satunya kosong, sedangkan di sini kelompok yang tidak punya baris
-- tidak muncul. Yang tidak muncul itu adalah kelompok TANPA DATA — bukan kelompok
-- bernilai nol.
--
-- Bind: :1 tipe (NULL = seluruh tipe) · :2 adjuster (NULL = seluruh adjuster)
--       :3 periode dari · :4 periode sampai
SELECT k.ADJUSTER                                 AS ADJUSTER,
       k.TIPE                                     AS REPORT_TYPE,
       ROUND(AVG(TO_NUMBER(k.SURVEYLAP)), 2)         AS SURVEY,
       ROUND(AVG(TO_NUMBER(k.IMMEDIATEADVICE)), 2)   AS IMMEDIATE_ADVICE,
       ROUND(AVG(TO_NUMBER(k.PRELIMINARYADVICE)), 2) AS PRELIMINARY_ADVICE,
       ROUND(AVG(TO_NUMBER(k.INTERIM)), 2)           AS INTERIM_REPORT,
       ROUND(AVG(TO_NUMBER(k.PROGRESS)), 2)          AS PROGRESS,
       ROUND(AVG(TO_NUMBER(k.KOMUNIKASI)), 2)        AS COMMUNICATION,
       ROUND(AVG(TO_NUMBER(k.PROPOSE)), 2)           AS PROPOSE,
       ROUND(AVG(TO_NUMBER(k.FINALREPORT)), 2)       AS FINAL_REPORT,
       ROUND(AVG(TO_NUMBER(k.NILAI)), 2)             AS TOTAL_SCORE
  FROM POOLDATA.DETAIL_KPI_ADJUSTER k
 WHERE (:1 IS NULL OR k.TIPE = :1)
   AND (:2 IS NULL OR k.ADJUSTER = :2)
   AND k.TANGGAL >= TO_DATE(:3, 'YYYY-MM-DD')
   AND k.TANGGAL < TO_DATE(:4, 'YYYY-MM-DD') + INTERVAL '1' DAY
 GROUP BY k.ADJUSTER, k.TIPE
 ORDER BY k.ADJUSTER, k.TIPE

-- name: detail
-- Grid "Detail KPI Adjuster" — satu baris per kasus survei yang sudah dinilai.
--
-- Ia TIDAK punya rujukan kueri langsung di export, dan itu perlu dinyatakan
-- terang-terangan: di Pega grid ini digambar dari halaman clipboard yang baru saja
-- DIHITUNG `PNCReportKPIAdjuster_act`, bukan dibaca dari tabel. Yang dibaca di sini adalah
-- hasil perhitungan itu setelah disimpan — sumber yang sama, tanpa langkah menulisnya.
--
-- Penyaring dan kolomnya karena itu mengikuti kueri Summary persis; yang berbeda hanya
-- ketiadaan agregasi. Kesamaan itu dijaga query_test.go, dan bukan demi kerapian: satu
-- penyaring yang tertinggal di sini membuat Summary dan Detail menjawab pertanyaan yang
-- berbeda dengan tampilan yang sama.
--
-- `TANGGAL` diambil sebagai TANGGAL, bukan `TO_CHAR`. Pemformatannya dikerjakan Go
-- (`08-TECHNICAL-STRATEGY.md` §4.3): tanggal yang dikembalikan sebagai teks membuat
-- pengurutan menjadi pengurutan teks, dan itu persis cacat yang `D-20` hapus dengan
-- membuang 411 pemakaian `TO_CHAR` dari SQL.
--
-- Bind: :1 tipe (NULL = seluruh tipe) · :2 adjuster (NULL = seluruh adjuster)
--       :3 periode dari · :4 periode sampai · :5 offset · :6 jumlah baris
SELECT k.ADJUSTER                    AS ADJUSTER,
       k.CASEID                      AS CASE_ID,
       k.TIPE                        AS REPORT_TYPE,
       k.TANGGAL                     AS SCORED_ON,
       TO_NUMBER(k.SURVEYLAP)         AS SURVEY,
       TO_NUMBER(k.IMMEDIATEADVICE)   AS IMMEDIATE_ADVICE,
       TO_NUMBER(k.PRELIMINARYADVICE) AS PRELIMINARY_ADVICE,
       TO_NUMBER(k.INTERIM)           AS INTERIM_REPORT,
       TO_NUMBER(k.PROGRESS)          AS PROGRESS,
       TO_NUMBER(k.KOMUNIKASI)        AS COMMUNICATION,
       TO_NUMBER(k.PROPOSE)           AS PROPOSE,
       TO_NUMBER(k.FINALREPORT)       AS FINAL_REPORT,
       TO_NUMBER(k.NILAI)             AS TOTAL_SCORE,
       COUNT(*) OVER ()              AS TOTAL_ROWS
  FROM POOLDATA.DETAIL_KPI_ADJUSTER k
 WHERE (:1 IS NULL OR k.TIPE = :1)
   AND (:2 IS NULL OR k.ADJUSTER = :2)
   AND k.TANGGAL >= TO_DATE(:3, 'YYYY-MM-DD')
   AND k.TANGGAL < TO_DATE(:4, 'YYYY-MM-DD') + INTERVAL '1' DAY
 ORDER BY k.ADJUSTER, k.TANGGAL DESC, k.CASEID
OFFSET :5 ROWS FETCH NEXT :6 ROWS ONLY

-- name: adjusters
-- Isi dropdown "Pilih Adjuster".
--
-- Diambil dari kolom yang SAMA dengan yang disaring, bukan dari master surveyor — lihat
-- catatan pada reportkpi.Repo.Adjusters. Rule Pega pengisinya (`BrowseAdjsuterExternal`)
-- tidak ada di export (`R-16`).
--
-- Baris tanpa nama adjuster dibuang: ia tidak dapat dipilih, dan menampilkannya sebagai
-- pilihan kosong hanya menambah satu baris yang tidak berarti di puncak dropdown.
--
-- Bind: :1 tipe (NULL = seluruh tipe) · :2 periode dari · :3 periode sampai
SELECT DISTINCT k.ADJUSTER AS ADJUSTER
  FROM POOLDATA.DETAIL_KPI_ADJUSTER k
 WHERE (:1 IS NULL OR k.TIPE = :1)
   AND k.ADJUSTER IS NOT NULL
   AND k.TANGGAL >= TO_DATE(:2, 'YYYY-MM-DD')
   AND k.TANGGAL < TO_DATE(:3, 'YYYY-MM-DD') + INTERVAL '1' DAY
 ORDER BY k.ADJUSTER

-- name: check_source
-- Dipakai `-periksa`: menjawab "tabelnya ada dan terbaca?" tanpa menarik satu baris pun.
--
-- Ia TIDAK memeriksa isi. Tabel yang ada tetapi kosong adalah keadaan yang sah — Pega
-- baru mengisinya ketika seseorang membuka tab KPI Adjuster di sana — dan membedakannya
-- dari tabel yang tidak ada adalah justru yang membuat laporan periksa berguna.
SELECT COUNT(*) AS TOTAL_ROWS
  FROM POOLDATA.DETAIL_KPI_ADJUSTER

-- name: check_distinct_types
-- Dipakai `-periksa`: menyebut nilai `TIPE` yang BENAR-BENAR ada di basis data.
--
-- Kenapa ini layak diperiksa: kedua tipe yang dikenal modul ini — OUTSTANDING dan FINAL —
-- dibaca dari literal di dalam `GetSummaryKPIAdjusterALL-SQL.xml`, bukan dari master mana
-- pun. Bila produksi memuat nilai ketiga, dropdown tidak akan pernah menampilkannya dan
-- barisnya tidak akan pernah terlihat — tanpa satu pun galat.
SELECT DISTINCT k.TIPE AS REPORT_TYPE
  FROM POOLDATA.DETAIL_KPI_ADJUSTER k
 WHERE k.TIPE IS NOT NULL
 ORDER BY k.TIPE
