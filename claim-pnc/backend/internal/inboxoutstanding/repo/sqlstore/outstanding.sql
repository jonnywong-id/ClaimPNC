-- Kueri layar Inbox Outstanding (`Harness/InboxRegister_Harness-Harness.xml`).
--
-- ============================================================================
-- SUMBER DATA: POOLDATA.T_CLAIMLIST_ADMIN
-- ============================================================================
--
-- Tabel itulah yang menggantikan `datapega.pc_asm_fw_gcnmfw_work` — ditetapkan Work Owner
-- 2026-09-21. Ia tabel DATAR, satu baris per klaim, dan memuat seluruh yang dibutuhkan
-- layar ini TANPA SATU PUN JOIN.
--
-- Kueri lama `RDB List/BrowseInboxOutstanding1-SQL.xml` menempuh empat tabel:
--
--     FROM datapega.pc_asm_fw_gcnmfw_work a,
--          datapega.pc_assign_worklist b,
--          pooldata.business c,
--          pooldata.businessgroup d
--
-- Keempatnya tidak diperlukan lagi. Kolom yang dulu harus ditarik lewat join — pemegang
-- tugas, tahap, kelompok bisnis — kini kolom biasa di tabel yang sama.
--
-- Satu akibat yang perlu dicatat: **persoalan INNER JOIN versus LEFT JOIN GUGUR
-- seluruhnya.** Tanpa join, tidak ada lagi klaim yang hilang karena tidak punya
-- assignment, dan tidak ada lagi klaim yang muncul dua kali karena punya dua assignment.
--
-- ============================================================================
-- PEMETAAN KOLOM — alias lama TIDAK dibawa
-- ============================================================================
--
-- Judul kolom diambil dari `Section/InboxRegister_Section-Section.xml`, section yang
-- dimuat harness rujukan dan yang di dalamnya sendiri berjudul "Inbox Outstanding"
-- (`:2150`).
--
--   kolom layar      properti section   alias kueri lama   kolom di T_CLAIMLIST_ADMIN
--   ---------------  -----------------  -----------------  --------------------------
--   Claim no         .ClaimNo           ClaimNo            PYID
--   Policy no        .District          District (!)       POLICYNO
--   Insured name     .CountryID         CountryID (!)      QQNAME
--   Business Name    .Country           Country (!)        BUSINESSNAME
--   Business source  .CityID            CityID (!)         SOBNAME
--   Branch name      .City              City (!)           BRANCHNAME
--   Admin name       .ReporterName      ReporterName       PXCREATEOPERATOR
--   Register Date    .pxCreateDateTime  StatusWork (!)     REGISTERDATE_1
--   Date of loss     .DateOfLoss        RW (!)             DATEOFLOSS_1
--   Aging            .DateForAging      —                  AGING
--   Claim status     .StatusClaim       StatusClaim (!)    PYSTATUSWORK
--   Status ASM       .LSC_ID            —                  STATUSLOCK_1  (belum pasti)
--   ASM PIC          —                  UserTeknis         USERTEKNIS_1
--   Total Aging      —                  —                  dihitung dari REGISTERDATE_1
--
-- Tanda (!) menandai alias yang artinya BERLAWANAN dengan isinya: "District" berisi nomor
-- polis, "City" berisi nama cabang, "StatusWork" berisi TANGGAL, "RW" berisi tanggal
-- kejadian, dan "CityID" berisi sumber bisnis.
--
-- DUA KOLOM YANG SEBELUMNYA DIKECUALIKAN KINI ADA: `SOBNAME` untuk "Business source" dan
-- `AGING` untuk "Aging". Keduanya hilang hanya karena tabel yang dipakai sebelumnya salah.
--
-- `AGING` dibaca APA ADANYA, tidak dihitung ulang. Tabel ini juga punya `DATEFORAGING_1`,
-- yang tampaknya menjadi acuannya — tetapi artinya belum dipastikan, sehingga menghitung
-- ulang berarti menebak dari tanggal mana.
--
-- ============================================================================
-- PENYARING — disalin dari kueri lama, baris per baris
-- ============================================================================
--
-- `RDB List/BrowseInboxOutstanding1-SQL.xml:121-128`:
--
--     AND a.pxobjclass = 'ASM-FW-GCNMFW-Work-PNC'
--     AND pystatuswork NOT IN ('Resolved-Completed', 'Resolved-Rejected')
--     AND b.pxflowname not in ('FixCorrespondence', 'Register_Flow_1')
--     AND b.PXTASKLABEL not in ('FixCorrespondence')
--     AND a.branchname != 'ASNET'
--
-- Kelimanya dipertahankan. Kolomnya semua ada di tabel ini, sehingga tidak satu pun
-- penyaring hilang oleh hilangnya join.
--
-- SATU PENYARING SENGAJA TIDAK DITAMBAHKAN: `STS_AKTIF`. Kolomnya ada, dan tabel datar
-- biasanya memakainya untuk menandai baris aktif — tetapi kueri lama tidak menyebutnya,
-- dan menambahkannya berarti mengubah perilaku tanpa dasar. Dicatat sebagai pertanyaan
-- terbuka.
--
-- ============================================================================
-- PENANDA /*SCOPE*/
-- ============================================================================
--
-- Diganti di Go dengan daftar PENANDA parameter, bukan nilainya — lihat expandScope. Ini
-- menggantikan `{ASIS:TempView.pyNote}` sistem lama, yang merangkai potongan WHERE dari
-- nilai properti klipboard.
--
-- Nomor parameter TIDAK harus urut dengan posisinya di dalam teks, dan di sini itu
-- dimanfaatkan: paginasi memakai :9 dan :10 meski tertulis paling bawah, supaya penanda
-- scope dapat mulai dari :11 tanpa bergeser saat jumlah lini berubah.
--
--     outstanding_list   :1..:4 pencarian · :5,:6 tahap · :7,:8 cabang · :9 offset · :10 limit
--                        -> scope mulai :11
--     outstanding_count  :1..:4 pencarian · :5,:6 tahap · :7,:8 cabang
--                        -> scope mulai :9
--
-- # Tiap kemunculan punya nomornya sendiri, dan itu WAJIB
--
-- Tahap dan cabang masing-masing muncul DUA KALI — sekali pada `IS NULL`, sekali pada
-- perbandingannya. Keduanya sempat memakai nomor yang sama (`:5 IS NULL OR … = :5`),
-- karena `:5` tampak sebagai satu variabel bernama "5".
--
-- Oracle menolaknya: **ORA-01008 not all variables bound**. Driver mengikat argumen
-- menurut URUTAN KEMUNCULAN penanda di dalam teks, bukan menurut nomornya — delapan
-- kemunculan menuntut delapan argumen, berapa pun nomor uniknya.
--
-- Tidak satu pun kueri lain di repo ini mengulang penanda dalam satu pernyataan, sehingga
-- pola itu tidak pernah teruji sampai kueri ini menyentuh Oracle. Nilainya tetap dikirim
-- dua kali dari `filterArgs`; yang berubah hanya penomorannya.
--
-- Kedua angka itu konstanta di sisi Go dan dijaga sebuah uji.

-- name: outstanding_list
SELECT k.PZINSKEY,
       k.PYID,
       k.POLICYNO,
       k.QQNAME,
       k.BUSINESSNAME,
       k.SOBNAME,
       k.BRANCHNAME,
       k.GROUPPANEL_1,
       k.BUSINESSGROUPID,
       k.REGISTERDATE_1,
       k.PXCREATEDATETIME,
       k.DATEOFLOSS_1,
       k.REPORTDATE_1,
       k.AGING,
       k.PYSTATUSWORK,
       k.STATUSLOCK_1,
       k.STATUSPROGRESS1,
       k.USERTEKNIS_1,
       k.PXCREATEOPERATOR,
       k.PXTASKLABEL,
       k.PXASSIGNEDOPERATORID
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
 WHERE k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND k.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND k.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1')
   AND k.PXTASKLABEL NOT IN ('FixCorrespondence')
   AND (k.BRANCHNAME <> 'ASNET' OR k.BRANCHNAME IS NULL)
   AND (:1 IS NULL
        OR UPPER(k.PYID) LIKE :2 ESCAPE '\'
        OR UPPER(k.POLICYNO) LIKE :3 ESCAPE '\'
        OR UPPER(k.USERTEKNIS_1) LIKE :4 ESCAPE '\')
   AND (:5 IS NULL OR UPPER(TRIM(k.PXTASKLABEL)) = :6)
   AND (:7 IS NULL OR UPPER(TRIM(k.BRANCHNAME)) = :8)
   /*SCOPE*/
 ORDER BY k.PXCREATEDATETIME DESC, k.PZINSKEY
OFFSET :9 ROWS FETCH NEXT :10 ROWS ONLY

-- name: outstanding_count
-- Menghitung SELURUH baris yang cocok, bukan baris pada halaman ini.
--
-- Syarat WHERE-nya wajib sama persis dengan outstanding_list. Bila keduanya menyimpang,
-- pengguna melihat "247 baris cocok" lalu menemukan jumlah yang berbeda saat menelusuri
-- halamannya — dan tidak ada galat yang muncul.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
 WHERE k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND k.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND k.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1')
   AND k.PXTASKLABEL NOT IN ('FixCorrespondence')
   AND (k.BRANCHNAME <> 'ASNET' OR k.BRANCHNAME IS NULL)
   AND (:1 IS NULL
        OR UPPER(k.PYID) LIKE :2 ESCAPE '\'
        OR UPPER(k.POLICYNO) LIKE :3 ESCAPE '\'
        OR UPPER(k.USERTEKNIS_1) LIKE :4 ESCAPE '\')
   AND (:5 IS NULL OR UPPER(TRIM(k.PXTASKLABEL)) = :6)
   AND (:7 IS NULL OR UPPER(TRIM(k.BRANCHNAME)) = :8)
   /*SCOPE*/
