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
--    KOREKSI (2026-10-06): alasan yang pernah tertulis di sini SALAH. Ia berbunyi "keduanya
--    juga tidak ditampilkan layar ini" dengan rujukan `Section/DashboardClaim_Section2`.
--    Section itu bukan grid Outstanding.
--
--    Grid Outstanding yang sebenarnya adalah `Section/InboxOutstandingClaim_Section`, dan ia
--    MENGIKAT keduanya sebagai dua kolom:
--
--        GET_POSISI_PROGRESS_PNC(pyID, 'POSISI')   AS "ClaimNo"        -> Posisi Klaim
--        GET_POSISI_PROGRESS_PNC(pyID, 'sts_prg2') AS "CloseClaimNote" -> Progress Klaim
--
--    (`RDB List/GcnmBrowseCase_SQL-SQL.xml`; caption "Posisi Klaim" dan "Progress Klaim" ada
--    di section itu, dan `.ClaimNo`/`.CloseClaimNote` terikat di selnya.)
--
--    Jadi yang dihilangkan BUKAN dua pemanggilan yang hasilnya dibuang, melainkan **dua
--    kolom yang dibaca pengguna**. Keduanya masih kurang.
--
--    `D-02` tetap melarang memanggil procedure-nya, sehingga penggantinya adalah kueri biasa
--    atas `POOLDATA.GCNM_PROGRESS_CLAIM` — modul `inboxprogressclaim` sudah menggantikan
--    fungsi yang sama untuk ragam `'POSISI'` dan `'sts_prg2'` lewat kueri `positions`, dan
--    pola itu yang dipakai bila kolom ini dilengkapi.
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
--
-- ============================================================================
-- PERBEDAAN KEDUA YANG DIWARISI: "BELUM LUNAS" MEMBUANG STATUS KOSONG
-- ============================================================================
--
-- Penyaring Status Pembayaran ditulis `A.STATUSCLAIM_1 <> '1163'` PERSIS seperti Pega
-- (`Activity/GCNMGetManagerCase_Act-Act.xml:7485`). Di Oracle maupun PostgreSQL perbandingan
-- itu menghasilkan NULL bila `STATUSCLAIM_1` kosong, sehingga klaim yang belum punya status
-- klaim sama sekali TIDAK tampil pada pilihan "Belum Lunas" — padahal ia jelas belum lunas.
--
-- Diganti `(… <> '1163' OR … IS NULL)` akan memperbaikinya, dan justru karena itu TIDAK
-- dilakukan di sini: ia menambah baris pada layar manajerial tanpa keputusan yang
-- mendasarinya, dan selisihnya akan muncul pada gerbang 1 sebagai cacat yang tidak dapat
-- dipetakan ke satu pun dari 13 butir `P-5`.
--
-- ============================================================================
-- ALIAS SUB-KUERI MEMAKAI T, BUKAN B
-- ============================================================================
--
-- Pega menulis sub-kueri EXISTS dengan alias `B`. Di sini `B` sudah dipakai
-- `PC_ASSIGN_WORKLIST` pada kueri luar, sehingga alias yang sama akan menaungi tabel luar
-- dan membuat `A.CLAIMID` merujuk sesuatu yang lain. Aliasnya diganti `T`; hasilnya identik.

-- name: outstanding_count
-- Menghitung SELURUH klaim berjalan yang cocok, bukan baris pada halaman ini.
--
-- Syarat WHERE-nya wajib sama persis dengan outstanding_list. Bila keduanya menyimpang,
-- pengguna membaca satu angka pada kartu lalu menemukan jumlah baris yang lain saat
-- menelusurinya — dan tidak ada galat yang muncul. `query_test.go` menjaganya baris per baris.
--
-- COUNT(DISTINCT A.PZINSKEY) DIPERTAHANKAN meski penggandaannya sudah tidak mungkin.
--
-- Alasan aslinya hilang pada 2026-10-08: gabung ke `PC_ASSIGN_WORKLIST` dulu menggandakan
-- baris untuk klaim yang punya lebih dari satu penugasan, dan join itu kini tidak ada lagi
-- karena kedua tabel menyatu di `T_CLAIMLIST_ADMIN` — satu klaim satu baris.
--
-- Tetap DISTINCT karena ia tidak lagi menyembunyikan apa pun: pada satu baris per klaim,
-- COUNT(DISTINCT PZINSKEY) dan COUNT(*) memberi angka yang sama. Membuangnya menghemat
-- tidak banyak dan menghilangkan jaring pengaman bila kelak ada join baru.
SELECT COUNT(DISTINCT A.PZINSKEY)
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.PYSTATUSWORK <> 'Resolved-Completed'
   AND A.PYSTATUSWORK <> 'Resolved-Rejected'
   AND A.BRANCHNAME <> 'ASNET'
   AND A.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1')
   AND A.PXTASKLABEL NOT IN ('FixCorrespondence')
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
   AND (:9 IS NULL OR UPPER(A.POLICYNO) LIKE :10 ESCAPE '\')
   AND (:11 IS NULL OR UPPER(A.PYID) LIKE :12 ESCAPE '\')
   AND (:13 IS NULL OR UPPER(A.USERTEKNIS_1) LIKE :14 ESCAPE '\')
   AND (:15 IS NULL
        OR (:16 = 'SUDAH'
            AND EXISTS (SELECT T.CLAIMID
                          FROM POOLDATA.T_CLAIM_ADJUSTMENT T
                         WHERE T.TRANSFER_CASHIER_DATE IS NOT NULL
                           AND T.CLAIMID = A.PZINSKEY))
        OR (:17 = 'BELUM'
            AND NOT EXISTS (SELECT T.CLAIMID
                              FROM POOLDATA.T_CLAIM_ADJUSTMENT T
                             WHERE T.TRANSFER_CASHIER_DATE IS NOT NULL
                               AND T.CLAIMID = A.PZINSKEY)))
   AND (:18 IS NULL
        OR (:19 = 'LUNAS' AND A.STATUSCLAIM_1 = '1163')
        OR (:20 = 'BELUM' AND A.STATUSCLAIM_1 <> '1163'))

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
       (SELECT s.LSC_NOTE
          FROM POOLDATA.V_STS_CLAIM s
         WHERE s.LSC_ID = A.STATUSCLAIM_1) AS LABEL_STATUS_KLAIM,
       A.PYSTATUSWORK      AS STATUS_PROSES,
       CAST(A.DATEOFLOSS_1 AS DATE) AS TANGGAL_KEJADIAN,
       A.REPORTDATE_1    AS TANGGAL_LAPOR,
       A.PXCREATEDATETIME  AS TANGGAL_PENDAFTARAN
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.PYSTATUSWORK <> 'Resolved-Completed'
   AND A.PYSTATUSWORK <> 'Resolved-Rejected'
   AND A.BRANCHNAME <> 'ASNET'
   AND A.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1')
   AND A.PXTASKLABEL NOT IN ('FixCorrespondence')
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
   AND (:9 IS NULL OR UPPER(A.POLICYNO) LIKE :10 ESCAPE '\')
   AND (:11 IS NULL OR UPPER(A.PYID) LIKE :12 ESCAPE '\')
   AND (:13 IS NULL OR UPPER(A.USERTEKNIS_1) LIKE :14 ESCAPE '\')
   AND (:15 IS NULL
        OR (:16 = 'SUDAH'
            AND EXISTS (SELECT T.CLAIMID
                          FROM POOLDATA.T_CLAIM_ADJUSTMENT T
                         WHERE T.TRANSFER_CASHIER_DATE IS NOT NULL
                           AND T.CLAIMID = A.PZINSKEY))
        OR (:17 = 'BELUM'
            AND NOT EXISTS (SELECT T.CLAIMID
                              FROM POOLDATA.T_CLAIM_ADJUSTMENT T
                             WHERE T.TRANSFER_CASHIER_DATE IS NOT NULL
                               AND T.CLAIMID = A.PZINSKEY)))
   AND (:18 IS NULL
        OR (:19 = 'LUNAS' AND A.STATUSCLAIM_1 = '1163')
        OR (:20 = 'BELUM' AND A.STATUSCLAIM_1 <> '1163'))
 ORDER BY A.PXCREATEDATETIME ASC, A.PZINSKEY
OFFSET :21 ROWS FETCH NEXT :22 ROWS ONLY

-- name: pindah_pic_saring
-- Memindahkan SELURUH klaim yang cocok dengan penyaring layar — "Select All" lintas halaman.
--
-- # Kenapa ini ada, dan bukan perulangan per klaim
--
-- "Select All" berarti seluruh hasil penyaring, bukan 25 baris pada halaman yang terbuka.
-- Pada data hari ini itu 1.639 klaim. Mengirimkannya sebagai 1.639 permintaan terpisah
-- membebani server dan membuat kegagalan di tengah meninggalkan sebagian berpindah dan
-- sebagian tidak — keadaan yang tidak dapat dibedakan dari pemindahan yang berhasil.
--
-- Satu pernyataan, satu transaksi, satu jawaban berisi jumlah barisnya.
--
-- # WHERE-nya SALINAN PERSIS dari outstanding_count
--
-- Ia wajib memindahkan tepat klaim yang terlihat pengguna. Bila syaratnya menyimpang, tombol
-- memindahkan himpunan yang BERBEDA dari yang tercentang di layar — dan tidak ada galat yang
-- muncul; yang terjadi hanya klaim yang pindah tanpa ada yang memintanya.
--
-- `TestSelectAllMovesExactlyWhatTheListShows` menjaganya baris per baris.
--
-- Penandanya bergeser satu: `:1` dipakai operator tujuan, sehingga penyaringnya mulai di
-- `:2`. Pergeseran itu satu-satunya perbedaan yang DIIZINKAN terhadap outstanding_count, dan
-- penjaganya memperhitungkannya secara eksplisit — bukan dengan melonggarkan perbandingannya.
UPDATE POOLDATA.T_CLAIMLIST_ADMIN
   SET USERTEKNIS_1 = :1
 WHERE PZINSKEY IN (
       SELECT DISTINCT A.PZINSKEY
         FROM POOLDATA.T_CLAIMLIST_ADMIN A
              INNER JOIN POOLDATA.BUSINESS c
                      ON A.BUSINESSCODE_1 = c.ID
              INNER JOIN POOLDATA.BUSINESSGROUP d
                      ON c.BUSINESSGROUPID = d.ID
        WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
          AND A.PYSTATUSWORK <> 'Resolved-Completed'
          AND A.PYSTATUSWORK <> 'Resolved-Rejected'
          AND A.BRANCHNAME <> 'ASNET'
          AND A.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1')
          AND A.PXTASKLABEL NOT IN ('FixCorrespondence')
          AND (:2 IS NULL
               OR UPPER(A.POLICYNO) LIKE :3 ESCAPE '\'
               OR UPPER(A.PYID) LIKE :4 ESCAPE '\')
          AND (:5 = 'ALL'
               OR (:6 = 'NONMBU'
                   AND A.GROUPPANEL_1 IN ('003', '004', '006')
                   AND c.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023'))
               OR (:7 = 'BONDING'
                   AND c.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023'))
               OR (:8 = 'PA' AND A.GROUPPANEL_1 = '002')
               OR (:9 = 'TRAVEL' AND A.GROUPPANEL_1 = '005'))
          AND (:10 IS NULL OR UPPER(A.POLICYNO) LIKE :11 ESCAPE '\')
          AND (:12 IS NULL OR UPPER(A.PYID) LIKE :13 ESCAPE '\')
          AND (:14 IS NULL OR UPPER(A.USERTEKNIS_1) LIKE :15 ESCAPE '\')
          AND (:16 IS NULL
               OR (:17 = 'SUDAH'
                   AND EXISTS (SELECT T.CLAIMID
                                 FROM POOLDATA.T_CLAIM_ADJUSTMENT T
                                WHERE T.TRANSFER_CASHIER_DATE IS NOT NULL
                                  AND T.CLAIMID = A.PZINSKEY))
               OR (:18 = 'BELUM'
                   AND NOT EXISTS (SELECT T.CLAIMID
                                     FROM POOLDATA.T_CLAIM_ADJUSTMENT T
                                    WHERE T.TRANSFER_CASHIER_DATE IS NOT NULL
                                      AND T.CLAIMID = A.PZINSKEY)))
          AND (:19 IS NULL
               OR (:20 = 'LUNAS' AND A.STATUSCLAIM_1 = '1163')
               OR (:21 = 'BELUM' AND A.STATUSCLAIM_1 <> '1163')))
