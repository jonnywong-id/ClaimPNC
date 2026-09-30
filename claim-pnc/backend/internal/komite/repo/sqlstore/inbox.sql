-- Kueri Inbox Komite.
--
-- ============================================================================
-- SELURUH PERNYATAAN DI BERKAS INI HANYA MEMBACA.
-- ============================================================================
--
-- Tidak ada satu pun INSERT, UPDATE, atau DELETE. `T_CLAIM_KOMITE_LIST` masih ditulis Pega
-- (kasus KMT-) sekaligus modul registrasi (kasus KMTN), dan `P-1` menetapkan satu tabel
-- hanya boleh ditulis satu sistem per baris. Keputusan KMTN ditulis modul registrasi.
--
--
-- ============================================================================
-- SUMBER: POOLDATA.T_CLAIM_KOMITE_LIST (Work Owner, 2026-09-29)
-- ============================================================================
--
-- Sebelumnya inbox dibaca dari case Pega (`InboxRegisterKomite_RD`: Work-Komite INNER JOIN
-- Assign-Worklist). Work Owner menetapkan sumbernya `T_CLAIM_KOMITE_LIST` — satu baris per
-- ANGGOTA komite — supaya kasus KMTN yang diterbitkan aplikasi ini ikut muncul.
--
-- Satu baris inbox = satu keanggotaan seseorang pada satu case:
--
--   pemilik         NAMAKOMITE (dibandingkan UPPER/TRIM: 72 baris sejak 2024 berhuruf
--                   kecil atau berspasi)
--   jenjang         KOMITEKE — anggota yang jenjang di bawahnya BELUM seluruhnya setuju
--                   tidak ditampilkan: di Pega tugasnya belum diberikan kepadanya
--   keputusan       STATUSAPPROVE: '1' diterima, nilai lain yang terisi ditolak, kosong
--                   atau '0' belum diputus (Pega menulis NULL; aplikasi ini menulis '0')
--   status          STATUSCASE per anggota; anggota yang sudah memutus dianggap
--                   Resolved-Completed, sehingga kotak Outstanding = belum memutus
--   tanggal         DATEOFCOMMITE_CREATE (dibuat), TANGGALKOMITE (diputus)
--   header klaim    T_CLAIM_PNC lewat CLAIMNO = NO_KLAIM (No Polis, Tertanggung, Bisnis,
--                   SOB, Cabang) — kolom itu tidak ada di T_CLAIM_KOMITE_LIST
--
-- Baris yang KOMITE_ID-nya kosong (7.166 baris sampah) dibuang. Anggota yang tercatat dua
-- kali pada jenjang yang sama (9 kasus sejak 2024) digabung lewat GROUP BY.
--
--
-- ============================================================================
-- AKIBAT YANG DITERIMA WORK OWNER (diukur 2026-09-29 pada basis data ASM)
-- ============================================================================
--
--   * Pada 189 case Pega yang masih berjalan sejak 2024, hanya 83 yang anggota tertundanya
--     di tabel ini sama dengan pemegang worklist Pega; 100 tidak punya baris tertunda.
--   * 65 case yang di Pega sudah Resolved-Completed masih punya anggota berstatus New di
--     tabel ini, sehingga tampil di kotak Outstanding.
--   * Tabel ini tidak berindeks pada NAMAKOMITE; penyaring pemilik menyisir tabel.
--
--
-- ============================================================================
-- PENYARING PEMILIK DAPAT DIMATIKAN — DAN ITU HANYA UNTUK PENGEMBANGAN
-- ============================================================================
--
-- Bentuknya `(:1 IS NULL OR UPPER(TRIM(k.NAMAKOMITE)) = :2)`. Ketika `:1` NULL, daftarnya
-- menjadi SELURUH antrean komite — satu baris per anggota. Penjagaannya tidak berubah:
-- `InboxFilter.AllOperators` harus diminta, `KOMITE_TANPA_PENYARING_OPERATOR` menolak
-- berjalan di luar `APP_ENV=development`, dan layar menyatakannya.
--
--
-- ============================================================================
-- GAYA SQL
-- ============================================================================
--
-- COALESCE bukan NVL · CASE WHEN bukan DECODE · OFFSET/FETCH bukan ROWNUM · tanpa TO_CHAR
-- untuk tampilan · `CAST(x AS DATE)` bukan `TRUNC` · `CAST(x AS INTEGER)` untuk jenjang
-- (seluruh KOMITEKE berisi angka, diperiksa 2026-09-29) · kolom selalu disebut namanya.


-- name: inbox_list
--
-- Satu halaman inbox. Yang paling lama menunggu di ATAS; CASE_ID dan jenjang menjadi
-- pemecah seri supaya urutannya pasti antar halaman.
SELECT c.CASE_ID,
       c.CLAIM_NUMBER,
       (SELECT MAX(p.NOPOLIS) FROM POOLDATA.T_CLAIM_PNC p WHERE p.CLAIMNO = c.CLAIM_NUMBER)      AS POLICY_NUMBER,
       (SELECT MAX(p.QQNAME) FROM POOLDATA.T_CLAIM_PNC p WHERE p.CLAIMNO = c.CLAIM_NUMBER)       AS INSURED_NAME,
       (SELECT MAX(p.BUSINESSNAME) FROM POOLDATA.T_CLAIM_PNC p WHERE p.CLAIMNO = c.CLAIM_NUMBER) AS BUSINESS_NAME,
       (SELECT MAX(p.SOBNAME) FROM POOLDATA.T_CLAIM_PNC p WHERE p.CLAIMNO = c.CLAIM_NUMBER)      AS SOURCE_OF_BUSINESS,
       (SELECT MAX(p.BRANCHNAME) FROM POOLDATA.T_CLAIM_PNC p WHERE p.CLAIMNO = c.CLAIM_NUMBER)   AS BRANCH_NAME,
       c.ASSIGNED_OPERATOR,
       c.COMMITTEE_DATE,
       c.CREATED_AT,
       c.WORK_STATUS,
       c.LEGACY_APPROVE
  FROM (
        SELECT k.KOMITE_ID                                                     AS CASE_ID,
               MAX(TRIM(REPLACE(k.NO_KLAIM, 'ASM-FW-GCNMFW-WORK ', '')))       AS CLAIM_NUMBER,
               MAX(TRIM(k.NAMAKOMITE))                                         AS ASSIGNED_OPERATOR,
               CAST(k.KOMITEKE AS INTEGER)                                     AS LEVEL_NO,
               COALESCE(MAX(k.TANGGALKOMITE), CAST(MIN(k.DATEOFCOMMITE_CREATE) AS DATE)) AS COMMITTEE_DATE,
               MIN(k.DATEOFCOMMITE_CREATE)                                     AS CREATED_AT,
               CASE
                 WHEN MAX(NULLIF(TRIM(k.STATUSAPPROVE), '0')) IS NOT NULL THEN 'Resolved-Completed'
                 ELSE MAX(k.STATUSCASE)
               END                                                             AS WORK_STATUS,
               MAX(NULLIF(TRIM(k.STATUSAPPROVE), '0'))                         AS LEGACY_APPROVE
          FROM POOLDATA.T_CLAIM_KOMITE_LIST k
         WHERE k.KOMITE_ID IS NOT NULL
           AND (:1 IS NULL OR UPPER(TRIM(k.NAMAKOMITE)) = :2)
           AND k.DATEOFCOMMITE_CREATE >= :3
           AND NOT (COALESCE(TRIM(k.STATUSAPPROVE), '0') = '0'
                    AND EXISTS (SELECT 1
                                  FROM POOLDATA.T_CLAIM_KOMITE_LIST l
                                 WHERE l.KOMITE_ID = k.KOMITE_ID
                                   AND CAST(l.KOMITEKE AS INTEGER) < CAST(k.KOMITEKE AS INTEGER)
                                   AND COALESCE(TRIM(l.STATUSAPPROVE), '0') <> '1'))
         GROUP BY k.KOMITE_ID, UPPER(TRIM(k.NAMAKOMITE)), CAST(k.KOMITEKE AS INTEGER)
       ) c
 WHERE (CASE
          WHEN :4 = 'outstanding'
               AND c.WORK_STATUS <> 'Resolved-Completed' THEN 1
          WHEN :5 = 'diterima'
               AND c.LEGACY_APPROVE = '1' THEN 1
          WHEN :6 = 'ditolak'
               AND c.LEGACY_APPROVE IS NOT NULL
               AND c.LEGACY_APPROVE <> '1' THEN 1
          ELSE 0
        END) = 1
   AND (:7 IS NULL
        OR UPPER(c.CASE_ID)      LIKE '%' || UPPER(:8) || '%' ESCAPE '\'
        OR UPPER(c.CLAIM_NUMBER) LIKE '%' || UPPER(:9) || '%' ESCAPE '\')
   AND (:10 IS NULL OR c.CREATED_AT >= :11)
   AND (:12 IS NULL OR c.CREATED_AT < :13)
 ORDER BY c.CREATED_AT ASC, c.CASE_ID ASC, c.LEVEL_NO ASC
OFFSET :14 ROWS FETCH NEXT :15 ROWS ONLY


-- name: inbox_count
--
-- Banyaknya baris yang cocok SEBELUM dipotong paginasi. Penyaringnya WAJIB sama persis
-- dengan inbox_list (`TestPenyaringDaftarDanPenghitungSama`); header klaim tidak dibaca.
SELECT COUNT(1)
  FROM (
        SELECT k.KOMITE_ID                                                     AS CASE_ID,
               MAX(TRIM(REPLACE(k.NO_KLAIM, 'ASM-FW-GCNMFW-WORK ', '')))       AS CLAIM_NUMBER,
               MIN(k.DATEOFCOMMITE_CREATE)                                     AS CREATED_AT,
               CASE
                 WHEN MAX(NULLIF(TRIM(k.STATUSAPPROVE), '0')) IS NOT NULL THEN 'Resolved-Completed'
                 ELSE MAX(k.STATUSCASE)
               END                                                             AS WORK_STATUS,
               MAX(NULLIF(TRIM(k.STATUSAPPROVE), '0'))                         AS LEGACY_APPROVE
          FROM POOLDATA.T_CLAIM_KOMITE_LIST k
         WHERE k.KOMITE_ID IS NOT NULL
           AND (:1 IS NULL OR UPPER(TRIM(k.NAMAKOMITE)) = :2)
           AND k.DATEOFCOMMITE_CREATE >= :3
           AND NOT (COALESCE(TRIM(k.STATUSAPPROVE), '0') = '0'
                    AND EXISTS (SELECT 1
                                  FROM POOLDATA.T_CLAIM_KOMITE_LIST l
                                 WHERE l.KOMITE_ID = k.KOMITE_ID
                                   AND CAST(l.KOMITEKE AS INTEGER) < CAST(k.KOMITEKE AS INTEGER)
                                   AND COALESCE(TRIM(l.STATUSAPPROVE), '0') <> '1'))
         GROUP BY k.KOMITE_ID, UPPER(TRIM(k.NAMAKOMITE)), CAST(k.KOMITEKE AS INTEGER)
       ) c
 WHERE (CASE
          WHEN :4 = 'outstanding'
               AND c.WORK_STATUS <> 'Resolved-Completed' THEN 1
          WHEN :5 = 'diterima'
               AND c.LEGACY_APPROVE = '1' THEN 1
          WHEN :6 = 'ditolak'
               AND c.LEGACY_APPROVE IS NOT NULL
               AND c.LEGACY_APPROVE <> '1' THEN 1
          ELSE 0
        END) = 1
   AND (:7 IS NULL
        OR UPPER(c.CASE_ID)      LIKE '%' || UPPER(:8) || '%' ESCAPE '\'
        OR UPPER(c.CLAIM_NUMBER) LIKE '%' || UPPER(:9) || '%' ESCAPE '\')
   AND (:10 IS NULL OR c.CREATED_AT >= :11)
   AND (:12 IS NULL OR c.CREATED_AT < :13)


-- name: inbox_summary
--
-- Jumlah baris KETIGA kotak dalam SATU perjalanan ke basis data. Penyaring kotak tidak
-- diterapkan; pencarian dan rentang tanggal diterapkan.
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
        SELECT k.KOMITE_ID                                                     AS CASE_ID,
               MAX(TRIM(REPLACE(k.NO_KLAIM, 'ASM-FW-GCNMFW-WORK ', '')))       AS CLAIM_NUMBER,
               MIN(k.DATEOFCOMMITE_CREATE)                                     AS CREATED_AT,
               CASE
                 WHEN MAX(NULLIF(TRIM(k.STATUSAPPROVE), '0')) IS NOT NULL THEN 'Resolved-Completed'
                 ELSE MAX(k.STATUSCASE)
               END                                                             AS WORK_STATUS,
               MAX(NULLIF(TRIM(k.STATUSAPPROVE), '0'))                         AS LEGACY_APPROVE
          FROM POOLDATA.T_CLAIM_KOMITE_LIST k
         WHERE k.KOMITE_ID IS NOT NULL
           AND (:1 IS NULL OR UPPER(TRIM(k.NAMAKOMITE)) = :2)
           AND k.DATEOFCOMMITE_CREATE >= :3
           AND NOT (COALESCE(TRIM(k.STATUSAPPROVE), '0') = '0'
                    AND EXISTS (SELECT 1
                                  FROM POOLDATA.T_CLAIM_KOMITE_LIST l
                                 WHERE l.KOMITE_ID = k.KOMITE_ID
                                   AND CAST(l.KOMITEKE AS INTEGER) < CAST(k.KOMITEKE AS INTEGER)
                                   AND COALESCE(TRIM(l.STATUSAPPROVE), '0') <> '1'))
         GROUP BY k.KOMITE_ID, UPPER(TRIM(k.NAMAKOMITE)), CAST(k.KOMITEKE AS INTEGER)
       ) c
 WHERE (:4 IS NULL
        OR UPPER(c.CASE_ID)      LIKE '%' || UPPER(:5) || '%' ESCAPE '\'
        OR UPPER(c.CLAIM_NUMBER) LIKE '%' || UPPER(:6) || '%' ESCAPE '\')
   AND (:7 IS NULL OR c.CREATED_AT >= :8)
   AND (:9 IS NULL OR c.CREATED_AT < :10)


-- name: inbox_get
--
-- Satu kasus menurut KOMITE_ID (:2), dengan baris anggota yang dipilih berurutan.
--
-- Penandanya dinomori menurut URUTAN KEMUNCULAN — operator (:1) muncul lebih dulu di SELECT
-- — karena go-ora mengisi penanda secara posisional. Menomorinya sebaliknya menukar
-- keduanya diam-diam: setiap case menjawab "tidak ditemukan".
--
--
--   1. baris milik pemanggil (:1) — supaya anggota yang membuka case dari kotak Diterima
--      atau Ditolak miliknya melihat keanggotaannya sendiri, bukan orang lain;
--   2. anggota yang belum memutus pada jenjang terendah — yang sedang ditunggu;
--   3. jenjang terendah.
--
-- Pemeriksaan kepemilikan tetap di usecase lewat BelongsTo. Penyaring tahun TIDAK
-- diterapkan: ia membatasi DAFTAR, bukan kasus yang nomornya sudah dipegang seseorang.
SELECT c.CASE_ID,
       c.CLAIM_NUMBER,
       (SELECT MAX(p.NOPOLIS) FROM POOLDATA.T_CLAIM_PNC p WHERE p.CLAIMNO = c.CLAIM_NUMBER)      AS POLICY_NUMBER,
       (SELECT MAX(p.QQNAME) FROM POOLDATA.T_CLAIM_PNC p WHERE p.CLAIMNO = c.CLAIM_NUMBER)       AS INSURED_NAME,
       (SELECT MAX(p.BUSINESSNAME) FROM POOLDATA.T_CLAIM_PNC p WHERE p.CLAIMNO = c.CLAIM_NUMBER) AS BUSINESS_NAME,
       (SELECT MAX(p.SOBNAME) FROM POOLDATA.T_CLAIM_PNC p WHERE p.CLAIMNO = c.CLAIM_NUMBER)      AS SOURCE_OF_BUSINESS,
       (SELECT MAX(p.BRANCHNAME) FROM POOLDATA.T_CLAIM_PNC p WHERE p.CLAIMNO = c.CLAIM_NUMBER)   AS BRANCH_NAME,
       c.ASSIGNED_OPERATOR,
       c.COMMITTEE_DATE,
       c.CREATED_AT,
       c.WORK_STATUS,
       c.LEGACY_APPROVE
  FROM (
        SELECT k.KOMITE_ID                                                   AS CASE_ID,
               TRIM(REPLACE(k.NO_KLAIM, 'ASM-FW-GCNMFW-WORK ', ''))          AS CLAIM_NUMBER,
               TRIM(k.NAMAKOMITE)                                            AS ASSIGNED_OPERATOR,
               COALESCE(k.TANGGALKOMITE, CAST(k.DATEOFCOMMITE_CREATE AS DATE)) AS COMMITTEE_DATE,
               k.DATEOFCOMMITE_CREATE                                        AS CREATED_AT,
               CASE
                 WHEN NULLIF(TRIM(k.STATUSAPPROVE), '0') IS NOT NULL THEN 'Resolved-Completed'
                 ELSE k.STATUSCASE
               END                                                           AS WORK_STATUS,
               NULLIF(TRIM(k.STATUSAPPROVE), '0')                            AS LEGACY_APPROVE,
               CASE WHEN UPPER(TRIM(k.NAMAKOMITE)) = :1 THEN 0 ELSE 1 END    AS MINE,
               CASE WHEN NULLIF(TRIM(k.STATUSAPPROVE), '0') IS NULL THEN 0 ELSE 1 END AS DECIDED,
               CAST(k.KOMITEKE AS INTEGER)                                   AS LEVEL_NO
          FROM POOLDATA.T_CLAIM_KOMITE_LIST k
         WHERE k.KOMITE_ID = :2
       ) c
 ORDER BY c.MINE ASC, c.DECIDED ASC, c.LEVEL_NO ASC
 FETCH FIRST 1 ROWS ONLY


-- name: inbox_check_table
--
-- Memastikan kedua tabel beserta kolomnya dapat dibaca akun aplikasi, tanpa mengambil satu
-- baris pun. Dipakai mode periksa untuk membedakan tabel yang tidak ada dari hak baca.
SELECT k.KOMITE_ID,
       k.NO_KLAIM,
       k.NAMAKOMITE,
       k.KOMITEKE,
       k.STATUSAPPROVE,
       k.STATUSCASE,
       k.DATEOFCOMMITE_CREATE,
       k.TANGGALKOMITE,
       p.CLAIMNO,
       p.NOPOLIS,
       p.QQNAME,
       p.BUSINESSNAME,
       p.SOBNAME,
       p.BRANCHNAME
  FROM POOLDATA.T_CLAIM_KOMITE_LIST k
  LEFT JOIN POOLDATA.T_CLAIM_PNC p
         ON p.CLAIMNO = k.NO_KLAIM
 WHERE 1 = 0
