-- Kueri Inbox Komite.
--
-- ============================================================================
-- SELURUH PERNYATAAN DI BERKAS INI HANYA MEMBACA.
-- ============================================================================
--
-- Tidak ada satu pun INSERT, UPDATE, atau DELETE. Ketiga tabel di bawah masih ditulis
-- Pega, dan `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem.
--
--
-- ============================================================================
-- SUMBER: InboxRegisterKomite_RD DAN SetDataKomitePNC_Act
-- ============================================================================
--
-- Ditetapkan Work Owner 2026-09-28. Keduanyalah yang benar-benar memunculkan data komite
-- di sistem lama, dan keduanya HANYA MEMBACA:
--
--   Report Definition/InboxRegisterKomite_RD-RD.xml
--       Dipakai `Harness/InboxKomite_Harness` dan `Section/InboxKomite_section` sebagai
--       sumber grid. Kelasnya `ASM-FW-GCNMFW-Work-Komite`, INNER JOIN ke `Assign-Worklist`
--       berprefix `newAssignPage`, dan penyaringnya `(A) AND B AND F1`:
--
--           A   newAssignPage.pxAssignedOperatorID = Param.assign
--           B   .pyStatusWork != "Resolved-Completed"
--           F1  pxYearNumber(.pxCreateDateTime) >= "2024"
--
--   Activity/SetDataKomitePNC_Act-Act.xml
--       Pemuat portal (kelas `Data-Portal`, 18 langkah). Ia bercabang pada
--       `FlagKomiteKlaims.pyCaseID`: 1 → Outstanding lewat `GetKomitePAOutstanding`,
--       2 → Diterima lewat `GetKomitePAditerima`, 3 → CariData. Blok `Terima` dan `Tolak`
--       di dalamnya HANYA `PROPERTY-SET` — tidak satu pun menulis.
--
-- Kesimpulan kotak Diterima dan Ditolak karena itu diturunkan dari
-- `T_CLAIM_KOMITE_LIST.STATUSAPPROVE`, persis seperti `GetKomitePAditerima`:
--
--     case when b.STATUSAPPROVE = '1' then 'DITERIMA' else 'DITOLAK' end
--
--
-- ============================================================================
-- POOLDATA.CPNC_KOMITE_KEPUTUSAN TIDAK DIPAKAI DI BERKAS INI
-- ============================================================================
--
-- Tabel itu tidak pernah ada di Pega — ia rancangan aplikasi ini sendiri untuk MENULIS
-- keputusan, dan `0004_komite_keputusan.up.sql` belum pernah dijalankan di lingkungan mana
-- pun. Menggabungkannya ke kueri daftar membuat ketiadaannya mematikan SELURUH layar
-- dengan `ORA-00942`, padahal yang hilang hanya kolom pelengkap.
--
-- Work Owner menetapkan tabel itu tidak dipakai untuk memunculkan data. Jalur tulisnya
-- tetap ada di kode tetapi tidak aktif — lihat decision.sql dan tersedia.go.
--
-- ============================================================================
-- YANG WAJIB DIKETAHUI BILA JALUR TULIS ITU KELAK DIHIDUPKAN
-- ============================================================================
--
-- Penyaring kotak di bawah TIDAK mengenal keputusan milik aplikasi ini. Selama tidak satu
-- pun keputusan dapat tercatat, ia setara dengan `komite.CommitteeCase.InBox` — karena di
-- sana `decided` selalu salah.
--
-- Begitu keputusan dapat tercatat, keduanya BERSELISIH: kasus yang sudah diputuskan
-- seseorang akan tetap muncul di kotak Outstanding miliknya. Yang menghidupkan jalur tulis
-- WAJIB mengembalikan penyaring itu ke sini. Uji `TestKotakSQLSelarasDenganDefinisiDomain`
-- menyebut kewajiban ini secara eksplisit.
--
--
-- ============================================================================
-- DUA KOLOM YANG DIMINTA RD TETAPI BUKAN KOLOM BASIS DATA
-- ============================================================================
--
-- Diverifikasi ke `ALL_TAB_COLUMNS` pada basis data ASM, bukan disimpulkan:
--
--   .CoverID              TIDAK ADA di DATAPEGA.PC_ASM_FW_GCNMFW_WORK (186 kolom)
--   .Komite.DateOfComitee TIDAK ADA di tabel yang sama
--
-- Keduanya hidup di blob Pega, dan Pega dapat membacanya; SQL tidak. Penggantinya diambil
-- dari kedua rule yang Work Owner sebut, bukan dikarang:
--
--   .CoverID              `SUBSTR(PNCCASEID, INSTR(PNCCASEID,' ')+1)` pada
--                         `GetKomitePAOutstanding` — memotong prefix kelas Pega
--   .Komite.DateOfComitee `TRUNC(A.PXCREATEDATETIME)` pada `GetKomitePAOutstanding`,
--                         dan `b.tanggalkomite` pada `GetKomitePAditerima`
--
-- Karena itu Tgl Komite di sini `COALESCE(TANGGALKOMITE, CAST(PXCREATEDATETIME AS DATE))`:
-- baris yang sudah punya tanggal komite memakainya, sisanya jatuh ke tanggal pembuatan
-- case — persis pembagian yang kedua rule itu lakukan.
--
--
-- ============================================================================
-- KENAPA PENYARING OPERATOR TIDAK DI-UPPER DAN TIDAK DI-TRIM
-- ============================================================================
--
-- Dua sebab, dan keduanya menunjuk arah yang sama.
--
-- Pertama, KESETARAAN. RD membandingkan `newAssignPage.pxAssignedOperatorID = Param.assign`
-- apa adanya. Menambahkan UPPER dan TRIM akan membuat layar ini menampilkan baris yang
-- TIDAK pernah dilihat pengguna di Pega — selisih yang tidak dapat dipetakan ke satu pun
-- dari 13 perbaikan `P-5` (`D-49`).
--
-- Kedua, KINERJA. `DATAPEGA.PC_ASSIGN_WORKLIST` punya indeks
-- `BULKPROCESSFROMLIST(PXASSIGNEDOPERATORID, PXREFOBJECTKEY, …)` — persis urutan yang kueri
-- ini butuhkan. Membungkus kolomnya dengan UPPER/TRIM membuat indeks itu tidak terpakai,
-- dan sisi lain join-nya `POOLDATA.T_CLAIM_KOMITE_LIST` yang berisi **39 juta baris**.
--
-- Perapian nilainya dikerjakan di Go lewat `komite.OperatorKey` sebelum dikirim ke sini.
--
--
-- ============================================================================
-- KENAPA T_CLAIM_KOMITE_LIST DIBACA LEWAT SUBKUERI SKALAR
-- ============================================================================
--
-- Tabel itu berisi **39.067.250 baris**. Membacanya lewat subkueri ber-GROUP BY atas
-- SELURUH tabel — bentuk yang dipakai versi sebelumnya — memaksa agregasi penuh pada setiap
-- permintaan inbox, berapa pun kecilnya halaman yang diminta.
--
-- Subkueri skalar berkorelasi pada `KOMITE_ID` memakai indeks `T_CLAIM_KOMITE_LIST_KMT`,
-- sehingga biayanya sebanding dengan jumlah baris yang benar-benar ditampilkan. `MAX`
-- mempertahankan "satu nilai per case" tanpa dapat menggandakan baris inbox.
--
--
-- ============================================================================
-- TABEL YANG DIBACA
-- ============================================================================
--
--   DATAPEGA.PC_ASM_FW_GCNMFW_WORK   header case; disaring PXOBJCLASS = Work-Komite
--   DATAPEGA.PC_ASSIGN_WORKLIST      penugasan — inilah yang menentukan MILIK SIAPA
--   POOLDATA.T_CLAIM_KOMITE_LIST     keputusan dan tanggal komite MENURUT PEGA
--
-- Tidak lebih. Penilaian AI, nilai uang, tipe komite, dan PIC teknik TIDAK dibaca: tidak
-- satu pun dari ketiganya ada di RD, dan yang menampilkannya di sistem lama adalah
-- `ShowKomiteTerimaTolakNonMBU` — jalur Non-MBU yang bukan sumber layar ini.
--
--
-- ============================================================================
-- GAYA SQL
-- ============================================================================
--
-- COALESCE bukan NVL · CASE WHEN bukan DECODE · LEFT/INNER JOIN bukan `(+)` ·
-- OFFSET/FETCH bukan ROWNUM · tanpa TO_CHAR untuk tampilan · `CAST(x AS DATE)` bukan
-- `TRUNC` · kolom selalu disebut namanya. Seluruhnya mengikuti
-- `09-DATABASE-STRATEGY.md` §4, sehingga kueri ini berjalan apa adanya di Oracle 19c
-- maupun PostgreSQL 17+.


-- name: inbox_list
--
-- Satu halaman inbox.
--
-- Urutannya: yang paling lama menunggu di ATAS. `GetKomitePAOutstanding` menuliskannya
-- sebagai `ORDER BY "AgingKomite" DESC`, dan karena Aging dihitung dari `PXCREATEDATETIME`,
-- urutan itu sama dengan `CREATED_AT ASC`. Bentuk ini dipakai karena ia memakai kolom
-- tabel apa adanya alih-alih ekspresi turunan.
--
-- CASE_ID menjadi pemecah seri supaya urutannya PASTI: dua case bertanggal sama tidak boleh
-- berpindah tempat antar permintaan, karena halaman kedua akan melewatkan baris yang
-- berpindah ke halaman pertama.
SELECT c.CASE_ID,
       c.CLAIM_NUMBER,
       c.POLICY_NUMBER,
       c.INSURED_NAME,
       c.BUSINESS_NAME,
       c.SOURCE_OF_BUSINESS,
       c.BRANCH_NAME,
       c.ASSIGNED_OPERATOR,
       c.COMMITTEE_DATE,
       c.CREATED_AT,
       c.WORK_STATUS,
       c.LEGACY_APPROVE
  FROM (
        SELECT a.PYID                                             AS CASE_ID,
               TRIM(REPLACE(a.PNCCASEID, 'ASM-FW-GCNMFW-WORK ', '')) AS CLAIM_NUMBER,
               a.POLICYNO                                         AS POLICY_NUMBER,
               a.QQNAME                                           AS INSURED_NAME,
               a.BUSINESSNAME                                     AS BUSINESS_NAME,
               a.SOBNAME                                          AS SOURCE_OF_BUSINESS,
               a.BRANCHNAME                                       AS BRANCH_NAME,
               w.PXASSIGNEDOPERATORID                             AS ASSIGNED_OPERATOR,
               COALESCE((SELECT MAX(k.TANGGALKOMITE)
                           FROM POOLDATA.T_CLAIM_KOMITE_LIST k
                          WHERE k.KOMITE_ID = a.PYID),
                        CAST(a.PXCREATEDATETIME AS DATE))         AS COMMITTEE_DATE,
               a.PXCREATEDATETIME                                 AS CREATED_AT,
               a.PYSTATUSWORK                                     AS WORK_STATUS,
               (SELECT MAX(k.STATUSAPPROVE)
                  FROM POOLDATA.T_CLAIM_KOMITE_LIST k
                 WHERE k.KOMITE_ID = a.PYID)                      AS LEGACY_APPROVE
          FROM DATAPEGA.PC_ASSIGN_WORKLIST w
          JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
                 ON a.PZINSKEY = w.PXREFOBJECTKEY
         WHERE w.PXOBJCLASS = 'Assign-Worklist'
           AND w.PXASSIGNEDOPERATORID = :1
           AND a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-Komite'
           AND a.PXCREATEDATETIME >= :2
       ) c
 WHERE (CASE
          WHEN :3 = 'outstanding'
               AND c.WORK_STATUS <> 'Resolved-Completed' THEN 1
          WHEN :4 = 'diterima'
               AND c.LEGACY_APPROVE = '1' THEN 1
          WHEN :5 = 'ditolak'
               AND c.LEGACY_APPROVE IS NOT NULL
               AND c.LEGACY_APPROVE <> '1' THEN 1
          ELSE 0
        END) = 1
   AND (:6 IS NULL
        OR UPPER(c.CASE_ID)      LIKE '%' || UPPER(:7) || '%' ESCAPE '\'
        OR UPPER(c.CLAIM_NUMBER) LIKE '%' || UPPER(:8) || '%' ESCAPE '\')
   AND (:9 IS NULL OR c.CREATED_AT >= :10)
   AND (:11 IS NULL OR c.CREATED_AT < :12)
 ORDER BY c.CREATED_AT ASC, c.CASE_ID ASC
OFFSET :13 ROWS FETCH NEXT :14 ROWS ONLY


-- name: inbox_count
--
-- Banyaknya baris yang cocok SEBELUM dipotong paginasi.
--
-- Penyaringnya WAJIB sama persis dengan inbox_list. Bila keduanya berbeda, layar akan
-- menampilkan jumlah halaman yang tidak pernah ada isinya — dan pengguna akan melaporkan
-- pekerjaan yang hilang. `TestPenyaringDaftarDanPenghitungSama` menegakkannya.
--
-- COMMITTEE_DATE tidak ikut dibaca di sini: ia tidak dipakai satu pun penyaring, dan
-- membacanya berarti satu subkueri berkorelasi tambahan untuk setiap baris yang dihitung.
SELECT COUNT(1)
  FROM (
        SELECT a.PYID                                             AS CASE_ID,
               TRIM(REPLACE(a.PNCCASEID, 'ASM-FW-GCNMFW-WORK ', '')) AS CLAIM_NUMBER,
               a.PXCREATEDATETIME                                 AS CREATED_AT,
               a.PYSTATUSWORK                                     AS WORK_STATUS,
               (SELECT MAX(k.STATUSAPPROVE)
                  FROM POOLDATA.T_CLAIM_KOMITE_LIST k
                 WHERE k.KOMITE_ID = a.PYID)                      AS LEGACY_APPROVE
          FROM DATAPEGA.PC_ASSIGN_WORKLIST w
          JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
                 ON a.PZINSKEY = w.PXREFOBJECTKEY
         WHERE w.PXOBJCLASS = 'Assign-Worklist'
           AND w.PXASSIGNEDOPERATORID = :1
           AND a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-Komite'
           AND a.PXCREATEDATETIME >= :2
       ) c
 WHERE (CASE
          WHEN :3 = 'outstanding'
               AND c.WORK_STATUS <> 'Resolved-Completed' THEN 1
          WHEN :4 = 'diterima'
               AND c.LEGACY_APPROVE = '1' THEN 1
          WHEN :5 = 'ditolak'
               AND c.LEGACY_APPROVE IS NOT NULL
               AND c.LEGACY_APPROVE <> '1' THEN 1
          ELSE 0
        END) = 1
   AND (:6 IS NULL
        OR UPPER(c.CASE_ID)      LIKE '%' || UPPER(:7) || '%' ESCAPE '\'
        OR UPPER(c.CLAIM_NUMBER) LIKE '%' || UPPER(:8) || '%' ESCAPE '\')
   AND (:9 IS NULL OR c.CREATED_AT >= :10)
   AND (:11 IS NULL OR c.CREATED_AT < :12)


-- name: inbox_summary
--
-- Jumlah baris KETIGA kotak dalam SATU perjalanan ke basis data.
--
-- Bentuk `SUM(CASE WHEN ...)` diambil dari `RDB List/BrowseClaimRCV_Aksep-SQL.xml`, yang
-- menghitung seluruh lencana sekaligus dengan cara yang sama. Sifatnya dipertahankan karena
-- itulah yang membuat lencana tidak dapat berselisih dengan isi tabel di bawahnya.
--
-- Penyaring kotak TIDAK diterapkan di sini — pencarian dan rentang tanggal diterapkan.
-- Dengan begitu lencana menjawab pertanyaan yang benar: "berapa yang cocok dengan pencarian
-- saya di kotak lain", bukan "berapa isi kotak lain seluruhnya" — yang akan membuat pengguna
-- berpindah tab lalu menemukan tabel kosong.
SELECT SUM(CASE
             WHEN c.WORK_STATUS <> 'Resolved-Completed' THEN 1
             ELSE 0
           END) AS OUTSTANDING_COUNT,
       SUM(CASE
             WHEN c.LEGACY_APPROVE = '1' THEN 1
             ELSE 0
           END) AS ACCEPTED_COUNT,
       SUM(CASE
             WHEN c.LEGACY_APPROVE IS NOT NULL
                  AND c.LEGACY_APPROVE <> '1' THEN 1
             ELSE 0
           END) AS REJECTED_COUNT
  FROM (
        SELECT a.PYID                                             AS CASE_ID,
               TRIM(REPLACE(a.PNCCASEID, 'ASM-FW-GCNMFW-WORK ', '')) AS CLAIM_NUMBER,
               a.PXCREATEDATETIME                                 AS CREATED_AT,
               a.PYSTATUSWORK                                     AS WORK_STATUS,
               (SELECT MAX(k.STATUSAPPROVE)
                  FROM POOLDATA.T_CLAIM_KOMITE_LIST k
                 WHERE k.KOMITE_ID = a.PYID)                      AS LEGACY_APPROVE
          FROM DATAPEGA.PC_ASSIGN_WORKLIST w
          JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
                 ON a.PZINSKEY = w.PXREFOBJECTKEY
         WHERE w.PXOBJCLASS = 'Assign-Worklist'
           AND w.PXASSIGNEDOPERATORID = :1
           AND a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-Komite'
           AND a.PXCREATEDATETIME >= :2
       ) c
 WHERE (:3 IS NULL
        OR UPPER(c.CASE_ID)      LIKE '%' || UPPER(:4) || '%' ESCAPE '\'
        OR UPPER(c.CLAIM_NUMBER) LIKE '%' || UPPER(:5) || '%' ESCAPE '\')
   AND (:6 IS NULL OR c.CREATED_AT >= :7)
   AND (:8 IS NULL OR c.CREATED_AT < :9)


-- name: inbox_get
--
-- Satu kasus, TANPA menyaring pemilik.
--
-- Pemeriksaan kepemilikan sengaja tidak ada di sini — ia dikerjakan lapisan usecase lewat
-- BelongsTo, supaya "tidak ada" dan "bukan milik Anda" dapat dibedakan di log meski
-- disamakan di peramban.
--
-- Penyaring tahun juga TIDAK diterapkan. Penyaring itu membatasi DAFTAR; menerapkannya di
-- sini akan membuat sebuah case yang nomornya sudah dipegang seseorang menjawab "tidak
-- ditemukan" hanya karena umurnya — dan itu pesan yang menyesatkan.
--
-- Penugasannya dibaca LEFT, bukan INNER: case yang penugasannya sudah selesai tetap harus
-- dapat dibuka, dan `PYRESOLVEDUSERID` menjadi penggantinya persis seperti pada riwayat.
SELECT a.PYID                                             AS CASE_ID,
       TRIM(REPLACE(a.PNCCASEID, 'ASM-FW-GCNMFW-WORK ', '')) AS CLAIM_NUMBER,
       a.POLICYNO                                         AS POLICY_NUMBER,
       a.QQNAME                                           AS INSURED_NAME,
       a.BUSINESSNAME                                     AS BUSINESS_NAME,
       a.SOBNAME                                          AS SOURCE_OF_BUSINESS,
       a.BRANCHNAME                                       AS BRANCH_NAME,
       COALESCE(w.PXASSIGNEDOPERATORID, a.PYRESOLVEDUSERID) AS ASSIGNED_OPERATOR,
       COALESCE((SELECT MAX(k.TANGGALKOMITE)
                   FROM POOLDATA.T_CLAIM_KOMITE_LIST k
                  WHERE k.KOMITE_ID = a.PYID),
                CAST(a.PXCREATEDATETIME AS DATE))         AS COMMITTEE_DATE,
       a.PXCREATEDATETIME                                 AS CREATED_AT,
       a.PYSTATUSWORK                                     AS WORK_STATUS,
       (SELECT MAX(k.STATUSAPPROVE)
          FROM POOLDATA.T_CLAIM_KOMITE_LIST k
         WHERE k.KOMITE_ID = a.PYID)                      AS LEGACY_APPROVE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
  LEFT JOIN DATAPEGA.PC_ASSIGN_WORKLIST w
         ON w.PXREFOBJECTKEY = a.PZINSKEY
        AND w.PXOBJCLASS = 'Assign-Worklist'
 WHERE a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-Komite'
   AND a.PYID = :1


-- name: inbox_check_table
--
-- Memastikan ketiga tabel beserta kolomnya dapat dibaca akun aplikasi, tanpa mengambil satu
-- baris pun. Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip:
-- tabelnya tidak ada versus tidak punya hak baca.
--
-- Ketiganya disebut di sini, bukan hanya tabel kerjanya. Versi sebelumnya hanya menyentuh
-- `PC_ASM_FW_GCNMFW_WORK`, sehingga tabel lain yang tidak terbaca lolos dari pemeriksaan dan
-- baru ketahuan ketika pengguna membuka layarnya.
SELECT a.PYID,
       a.PNCCASEID,
       a.POLICYNO,
       a.QQNAME,
       a.BUSINESSNAME,
       a.SOBNAME,
       a.BRANCHNAME,
       a.PXCREATEDATETIME,
       a.PYSTATUSWORK,
       a.PYRESOLVEDUSERID,
       a.PZINSKEY,
       w.PXASSIGNEDOPERATORID,
       k.STATUSAPPROVE,
       k.TANGGALKOMITE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
  LEFT JOIN DATAPEGA.PC_ASSIGN_WORKLIST w
         ON w.PXREFOBJECTKEY = a.PZINSKEY
  LEFT JOIN POOLDATA.T_CLAIM_KOMITE_LIST k
         ON k.KOMITE_ID = a.PYID
 WHERE 1 = 0
