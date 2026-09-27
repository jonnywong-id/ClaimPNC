-- Kueri tile OUTSTANDING.
--
-- Menggantikan `RDB List/GcnmBrowseCase_SQL-SQL.xml`, dipanggil
-- `Activity/GCNMGetManagerCase_Act-Act.xml` dari `SetDashboardClaim` saat `param.tipe == 0`.
--
-- ============================================================================
-- APA YANG BERUBAH DARI KUERI LAMA, DAN KENAPA
-- ============================================================================
--
-- 1. ENAM PENANDA `{ASIS:…}` DIGANTI PARAMETER BINDING
--
--    Kueri lama menyisipkan enam potongan klausa WHERE langsung ke dalam teks SQL —
--    `{ASIS:TempFilter.CaseID}`, `.City`, `.CityID`, `.District`, `.DistrictID`,
--    `.Country` — beberapa di antaranya dirangkai dari isian yang diketik pengguna.
--    Itu celah SQL injection sekaligus penghalang portabilitas (utang teknis §4.5).
--
-- 2. `ROWNUM` DIGANTI `OFFSET … FETCH NEXT`
--
--    Kueri lama membungkus hasil dua lapis hanya untuk memberi nomor baris. Pola
--    `OFFSET … FETCH` didukung Oracle 12c+ dan PostgreSQL, sehingga satu teks SQL
--    berjalan di keduanya (`D-20`).
--
-- 3. DUA PEMANGGILAN `GET_POSISI_PROGRESS_PNC` TIDAK DIBAWA
--
--    Kueri lama memanggil fungsi basis data itu dua kali per baris, untuk kolom `"ClaimNo"`
--    dan `"CloseClaimNote"`. `D-02` menetapkan aplikasi tidak memanggil stored procedure.
--
--    Keduanya juga TIDAK ditampilkan layar ini: `Section/DashboardClaim_Section2-Section.xml`
--    menggambar enam kolom — No Klaim, No Polis, Nama Tertanggung, Nama Bisnis, Sumber
--    Bisnis, Nama Cabang. Jadi yang dihilangkan bukan data yang dibaca pengguna, melainkan
--    dua pemanggilan fungsi per baris yang hasilnya dibuang.
--
--    Modul `inboxprogressclaim` sudah menggantikan fungsi yang sama dengan kueri biasa atas
--    `POOLDATA.GCNM_PROGRESS_CLAIM` bila kelak kolomnya memang diperlukan di sini.
--
-- 4. ALIAS MENYESATKAN DIBERI NAMA YANG BENAR
--
--    Kueri lama memaksa nama kolom agar cocok dengan properti klipboard Pega yang sudah
--    ada, sehingga namanya tidak lagi mencerminkan isinya (`D-19`):
--
--        a.pyid         AS "City"        -> NO_KLAIM
--        a.pzinskey     AS "CaseID"      -> ID_KLAIM
--        policyno       AS "Currency"    -> NO_POLIS
--        qqname         AS "CityID"      -> NAMA_TERTANGGUNG
--        businessname   AS "District"    -> NAMA_BISNIS
--        sobname        AS "DistrictID"  -> SUMBER_BISNIS
--        branchname     AS "Country"     -> NAMA_CABANG
--        dateofloss_1   AS "CountryID"   -> TANGGAL_KEJADIAN
--        userteknis_1   AS "CauseOfLoss" -> PIC_TEKNIK
--        pxCreateOpName AS "ClaimID"     -> ADMIN_PNC
--
--    Perhatikan dua baris terakhir: alias `"ClaimID"` dipakai untuk NAMA OPERATOR,
--    sementara `"CaseID"` dipakai untuk kunci klaim. Dan `"CauseOfLoss"` — Penyebab
--    Kerugian — sebenarnya berisi PIC Teknik.
--
-- 5. TANGGAL KEJADIAN DIKEMBALIKAN SEBAGAI TANGGAL, BUKAN TEKS
--
--    Kueri lama memformatnya `to_char(TRUNC(dateofloss_1),'dd-mm-yyyy')`, sehingga
--    pengurutan tanggal menjadi pengurutan TEKS — `01/12/2024` dianggap lebih kecil dari
--    `02/01/2020`. Pemformatan pindah ke Go (`09-DATABASE-STRATEGY` §3.2).
--
--    `TRUNC` sendiri diganti `CAST(… AS DATE)`: keduanya membuang bagian jam, tetapi yang
--    kedua berjalan di Oracle maupun PostgreSQL (`D-20`, `09-DATABASE-STRATEGY` §4).
--
-- ============================================================================
-- SATU PERBEDAAN YANG DIWARISI DAN BELUM DIPUTUSKAN
-- ============================================================================
--
-- Syarat cabang di bawah ditulis `A.BRANCHNAME <> 'ASNET'` PERSIS seperti Pega. Di Oracle
-- perbandingan itu menghasilkan NULL bila `BRANCHNAME` kosong, sehingga baris tanpa cabang
-- IKUT TERBUANG.
--
-- Modul `inboxcloseclaim` — yang melayani tile CLOSE CLAIM di layar yang sama — menuliskannya
-- `(A.BRANCHNAME <> 'ASNET' OR A.BRANCHNAME IS NULL)`, sehingga baris tanpa cabang IKUT
-- TERHITUNG di sana.
--
-- Akibatnya kedua tile pada layar ini menghitung populasi yang sedikit berbeda bila ada
-- klaim ber-`BRANCHNAME` NULL. Yang dipilih di sini adalah perilaku Pega (`P-5`); pelebaran
-- di modul sebelah tidak dicatat alasannya di sana.
--
-- Mana yang benar adalah pertanyaan untuk Work Owner, dan jawabannya menyentuh KEDUA modul —
-- bukan sesuatu yang diselaraskan sepihak dari sini. Dicatat di
-- docs/keputusan-implementasi.md.

-- name: outstanding_count
-- Menghitung SELURUH klaim berjalan yang cocok, bukan baris pada halaman ini.
--
-- Syarat WHERE-nya wajib sama persis dengan outstanding_list. Bila keduanya menyimpang,
-- pengguna membaca satu angka pada kartu lalu menemukan jumlah baris yang lain saat
-- menelusurinya — dan tidak ada galat yang muncul. `query_test.go` menjaganya baris per baris.
--
-- COUNT(DISTINCT A.PZINSKEY), bukan COUNT(*): gabung ke PC_ASSIGN_WORKLIST menggandakan
-- baris untuk klaim yang punya lebih dari satu penugasan. Kueri lama memakai SELECT DISTINCT
-- dengan alasan yang sama.
SELECT COUNT(DISTINCT A.PZINSKEY)
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST B
               ON A.PZINSKEY = B.PXREFOBJECTKEY
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.PYSTATUSWORK <> 'Resolved-Completed'
   AND A.PYSTATUSWORK <> 'Resolved-Rejected'
   AND A.BRANCHNAME <> 'ASNET'
   AND B.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1')
   AND B.PXTASKLABEL NOT IN ('FixCorrespondence')
   AND (:1 IS NULL
        OR UPPER(A.POLICYNO) LIKE :2 ESCAPE '\'
        OR UPPER(A.PYID) LIKE :3 ESCAPE '\')
   AND (:4 = 'ALL'
        OR (:5 = 'NONMBU'
            AND A.GROUPPANEL_1 IN ('003', '004', '006')
            AND c.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023'))
        OR (:6 = 'BONDING'
            AND c.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023'))
        OR (:7 = 'PA' AND A.GROUPPANEL_1 = '002')
        OR (:8 = 'TRAVEL' AND A.GROUPPANEL_1 = '005'))

-- name: outstanding_list
-- Membaca satu halaman klaim berjalan.
--
-- DISTINCT dipertahankan dengan alasan yang sama seperti pada kueri hitung: satu klaim dapat
-- memegang lebih dari satu penugasan di PC_ASSIGN_WORKLIST, dan tanpa DISTINCT ia tampil
-- berkali-kali di grid.
--
-- Urutannya `PXCREATEDATETIME ASC` mengikuti kueri lama, ditambah PZINSKEY sebagai pemutus
-- seri. Tanpa pemutus seri, dua baris berwaktu sama dapat bertukar urutan antar halaman
-- sehingga satu baris tampil dua kali dan satu lagi tidak pernah tampil.
SELECT DISTINCT
       A.PZINSKEY          AS ID_KLAIM,
       A.PYID              AS NO_KLAIM,
       A.POLICYNO          AS NO_POLIS,
       A.QQNAME            AS NAMA_TERTANGGUNG,
       A.BUSINESSNAME      AS NAMA_BISNIS,
       A.SOBNAME           AS SUMBER_BISNIS,
       A.BRANCHNAME        AS NAMA_CABANG,
       A.USERTEKNIS_1      AS PIC_TEKNIK,
       A.PXCREATEOPNAME    AS ADMIN_PNC,
       A.STATUSCLAIM_1     AS KODE_STATUS_KLAIM,
       A.PYSTATUSWORK      AS STATUS_PROSES,
       CAST(A.DATEOFLOSS_1 AS DATE) AS TANGGAL_KEJADIAN,
       A.PXCREATEDATETIME  AS TANGGAL_REGISTER
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST B
               ON A.PZINSKEY = B.PXREFOBJECTKEY
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.PYSTATUSWORK <> 'Resolved-Completed'
   AND A.PYSTATUSWORK <> 'Resolved-Rejected'
   AND A.BRANCHNAME <> 'ASNET'
   AND B.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1')
   AND B.PXTASKLABEL NOT IN ('FixCorrespondence')
   AND (:1 IS NULL
        OR UPPER(A.POLICYNO) LIKE :2 ESCAPE '\'
        OR UPPER(A.PYID) LIKE :3 ESCAPE '\')
   AND (:4 = 'ALL'
        OR (:5 = 'NONMBU'
            AND A.GROUPPANEL_1 IN ('003', '004', '006')
            AND c.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023'))
        OR (:6 = 'BONDING'
            AND c.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023'))
        OR (:7 = 'PA' AND A.GROUPPANEL_1 = '002')
        OR (:8 = 'TRAVEL' AND A.GROUPPANEL_1 = '005'))
 ORDER BY A.PXCREATEDATETIME ASC, A.PZINSKEY
OFFSET :9 ROWS FETCH NEXT :10 ROWS ONLY
