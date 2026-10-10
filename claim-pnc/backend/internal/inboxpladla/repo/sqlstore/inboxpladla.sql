-- ============================================================================
-- SUMBER BARU (2026-10-08) — tabel kerja Pega tidak dibaca lagi
-- ============================================================================
--
-- Keputusan Work Owner: `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` sudah tidak dipakai. Seluruh daftar
-- di berkas ini (dan di komunikasi.sql, detail.sql) dulu menggabungkannya hanya untuk dua
-- kolom, dan keduanya kini dibaca dari baris klaim yang SUDAH menjadi sumber utama kueri:
--
--     STATUSCLAIM_1    ->  T_CLAIM_PNC.STATUSCLAIM
--     ISPENDINGCLOSE   ->  T_CLAIM_PNC.ISPENDINGCLOSE
--
-- `T_CLAIMLIST_ADMIN` tidak dipakai: kedua kolom itu kosong 100% di sana, dan ia hanya
-- memuat klaim di antrean Admin — bukan klaim yang sudah dikirimi PLA/DLA.
--
-- Akibat yang terukur (Oracle dev ASM, populasi klaim non-PA/non-Travel yang punya PLA/DLA
-- terkirim, 262 klaim):
--
--   * Kode status berbeda dari tabel kerja pada 50 klaim (umumnya tabel kerja `1149`/`1147`
--     sementara `T_CLAIM_PNC` `1138`). Kolom "Status" dan tabel ringkas "Status / Jumlah"
--     karena itu dapat berbeda dari layar Pega untuk klaim-klaim itu.
--   * `ISPENDINGCLOSE` berbeda pada 21 klaim; 20 di antaranya `false` lawan NULL yang
--     artinya sama di sini. Yang bermakna: 6 klaim (seluruh tabel) yang di tabel kerja
--     `true` tetapi NULL di `T_CLAIM_PNC` — klaim itu pindah dari daftar DLA (kode `1139`)
--     ke daftar Close bila PLA-nya terkirim.

-- name: reinsurer_codes
-- Kode reasuradur milik satu login, terurut MENURUN.
--
-- Bind: :1 login pemanggil
--
-- Urutan menurun BUKAN kerapian: dua dari tiga daftar memakai kode TERTINGGI saja
-- (`ORDER BY reinsurerid DESC FETCH NEXT 1 ROW ONLY` di kueri lama), sehingga yang
-- pertama di senarai inilah yang mereka pakai.
--
-- Hasil KOSONG berarti loginnya bukan reasuradur — dan itu dijawab dengan pesan
-- tersendiri, bukan dengan daftar kosong. Lihat inboxpladla.ErrCallerNotAReinsurer.
SELECT REINSURERID
  FROM POOLDATA.T_REINSURER
 WHERE UPPER(TRIM(LOGIN)) = UPPER(TRIM(:1))
 ORDER BY REINSURERID DESC

-- name: list_pla
-- Klaim yang PLA-nya sudah dikirimkan kepada pemanggil, tetapi DLA-nya belum.
--
-- Bind: :1 login (kolom No PLA) · :2 penanda pencarian · :3 pola pencarian
--       :4 login (syarat PLA terkirim) · :5 login (syarat DLA belum terkirim)
--       :6 offset · :7 jumlah baris
--
-- Penomorannya MENAIK MENURUT URUTAN KEMUNCULAN — lihat catatan di kaki berkas ini.
SELECT c.CLAIMID                         AS CLAIM_KEY,
       c.CLAIMNO                         AS CLAIM_NO,
       c.NOPOLIS                         AS POLICY_NO,
       c.QQNAME                          AS INSURED,
       c.BUSINESSNAME                    AS BUSINESS_NAME,
       c.REGISTERDATE                    AS REGISTER_DATE,
       c.DATEOFLOSS                      AS LOSS_DATE,
       c.PICTEKNIK                       AS PIC_TEKNIK,
       c.STATUSCLAIM                     AS STATUS_CODE,
       s.LSC_NOTE                        AS STATUS_LABEL,
       (SELECT p.NOPLA
          FROM POOLDATA.T_PLALIST p
         WHERE p.CLAIMID = c.CLAIMID
           AND p.REINSCODE = (SELECT r.REINSURERID
                                FROM POOLDATA.T_REINSURER r
                               WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:1))
                               ORDER BY r.REINSURERID DESC
                               FETCH NEXT 1 ROWS ONLY)
         ORDER BY p.REVISI DESC
         FETCH NEXT 1 ROWS ONLY)         AS ADVICE_NO,
       c.CLOSECLAIMNOTE                  AS CLOSE_NOTE,
       COUNT(*) OVER ()                  AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
  LEFT JOIN POOLDATA.M_STS_CLAIM s
         ON s.LSC_ID = c.STATUSCLAIM
 WHERE c.GROUPPANEL NOT IN ('002', '005')
   AND c.STATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (:2 IS NULL OR UPPER(c.CLAIMID) LIKE :3 ESCAPE '\')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_PLALIST a
                WHERE a.CLAIMID = c.CLAIMID
                  AND a.ISKIRIM = '1'
                  AND a.TGLKIRIM IS NOT NULL
                  AND a.EMAILPLA IS NOT NULL
                  AND a.REINSCODE = (SELECT r.REINSURERID
                                       FROM POOLDATA.T_REINSURER r
                                      WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:4))
                                      ORDER BY r.REINSURERID DESC
                                      FETCH NEXT 1 ROWS ONLY))
   AND NOT EXISTS (SELECT 1
                     FROM POOLDATA.T_DLALIST d
                    WHERE d.CLAIMID = c.CLAIMID
                      AND d.ISKIRIM = '1'
                      AND d.TGLKIRIM IS NOT NULL
                      AND d.EMAILDLA IS NOT NULL
                      AND d.REINSCODE = (SELECT r.REINSURERID
                                           FROM POOLDATA.T_REINSURER r
                                          WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:5))
                                          ORDER BY r.REINSURERID DESC
                                          FETCH NEXT 1 ROWS ONLY))
 ORDER BY c.CLAIMNO
OFFSET :6 ROWS FETCH NEXT :7 ROWS ONLY

-- name: count_pla
-- Tabel ringkas "Status / Jumlah" untuk daftar PLA.
--
-- Bind: :1 penanda pencarian · :2 pola pencarian · :3 login · :4 login
--
-- Penyaringnya WAJIB sama persis dengan list_pla. Angka yang tidak cocok dengan tabel di
-- bawahnya adalah hal pertama yang dilaporkan pengguna sebagai kerusakan.
SELECT c.STATUSCLAIM                     AS STATUS_CODE,
       MAX(s.LSC_NOTE)                   AS STATUS_LABEL,
       COUNT(*)                          AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
  LEFT JOIN POOLDATA.M_STS_CLAIM s
         ON s.LSC_ID = c.STATUSCLAIM
 WHERE c.GROUPPANEL NOT IN ('002', '005')
   AND c.STATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (:1 IS NULL OR UPPER(c.CLAIMID) LIKE :2 ESCAPE '\')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_PLALIST a
                WHERE a.CLAIMID = c.CLAIMID
                  AND a.ISKIRIM = '1'
                  AND a.TGLKIRIM IS NOT NULL
                  AND a.EMAILPLA IS NOT NULL
                  AND a.REINSCODE = (SELECT r.REINSURERID
                                       FROM POOLDATA.T_REINSURER r
                                      WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:3))
                                      ORDER BY r.REINSURERID DESC
                                      FETCH NEXT 1 ROWS ONLY))
   AND NOT EXISTS (SELECT 1
                     FROM POOLDATA.T_DLALIST d
                    WHERE d.CLAIMID = c.CLAIMID
                      AND d.ISKIRIM = '1'
                      AND d.TGLKIRIM IS NOT NULL
                      AND d.EMAILDLA IS NOT NULL
                      AND d.REINSCODE = (SELECT r.REINSURERID
                                           FROM POOLDATA.T_REINSURER r
                                          WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:4))
                                          ORDER BY r.REINSURERID DESC
                                          FETCH NEXT 1 ROWS ONLY))
 GROUP BY c.STATUSCLAIM
 ORDER BY c.STATUSCLAIM

-- name: list_dla
-- Klaim yang DLA-nya sudah dikirimkan kepada pemanggil.
--
-- Bind: :1 login (kolom No PLA) · :2 penanda pencarian · :3 pola pencarian
--       :4 login (syarat DLA terkirim) · :5 offset · :6 jumlah baris
--
-- Kode status DIGANTI `1139` ketika klaimnya menunggu penutupan — satu-satunya daftar yang
-- melakukannya. Label statusnya ikut dicari untuk kode pengganti itu, bukan untuk kode
-- aslinya; tanpa itu, baris yang digambar `1139` akan membawa keterangan status yang lain.
--
-- Kueri lama Pega menggabungkan tabel kerja secara INNER (`b.claimid=z.pzinskey` di klausa
-- FROM) karena `ISPENDINGCLOSE` dibaca dari sana. SUMBER BARU (2026-10-08): kolom itu kini
-- dibaca dari `T_CLAIM_PNC.ISPENDINGCLOSE`, sehingga gabungan INNER itu HILANG — klaim yang
-- tidak punya baris kerja Pega (mis. klaim PNCN) kini ikut. Diukur di Oracle dev: populasi
-- PLA/DLA yang tidak punya baris kerja hanya 3 klaim, dan daftar DLA reasuradur teramai
-- tetap 15 baris lama = 15 baris baru. Lihat catatan di kepala berkas.
--
-- Kueri lama memuat DUA sub-kueri EXISTS yang ISINYA SAMA PERSIS, digabung dengan `OR`.
-- Duplikasi itu tidak mengubah hasil dan tidak dibawa.
SELECT c.CLAIMID                         AS CLAIM_KEY,
       c.CLAIMNO                         AS CLAIM_NO,
       c.NOPOLIS                         AS POLICY_NO,
       c.QQNAME                          AS INSURED,
       c.BUSINESSNAME                    AS BUSINESS_NAME,
       c.REGISTERDATE                    AS REGISTER_DATE,
       c.DATEOFLOSS                      AS LOSS_DATE,
       c.PICTEKNIK                       AS PIC_TEKNIK,
       CASE WHEN c.ISPENDINGCLOSE = 'true' THEN '1139'
            ELSE c.STATUSCLAIM END       AS STATUS_CODE,
       (SELECT s.LSC_NOTE
          FROM POOLDATA.M_STS_CLAIM s
         WHERE s.LSC_ID = CASE WHEN c.ISPENDINGCLOSE = 'true' THEN '1139'
                               ELSE c.STATUSCLAIM END) AS STATUS_LABEL,
       (SELECT p.NOPLA
          FROM POOLDATA.T_PLALIST p
         WHERE p.CLAIMID = c.CLAIMID
           AND p.REINSCODE = (SELECT r.REINSURERID
                                FROM POOLDATA.T_REINSURER r
                               WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:1))
                               ORDER BY r.REINSURERID DESC
                               FETCH NEXT 1 ROWS ONLY)
         ORDER BY p.REVISI DESC
         FETCH NEXT 1 ROWS ONLY)         AS ADVICE_NO,
       c.CLOSECLAIMNOTE                  AS CLOSE_NOTE,
       COUNT(*) OVER ()                  AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE c.GROUPPANEL NOT IN ('002', '005')
   AND (c.STATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
        OR (c.STATUSWORK = 'Resolved-Completed' AND c.ISPENDINGCLOSE = 'true'))
   AND (:2 IS NULL OR UPPER(c.CLAIMID) LIKE :3 ESCAPE '\')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_DLALIST a
                WHERE a.CLAIMID = c.CLAIMID
                  AND a.ISKIRIM = '1'
                  AND a.TGLKIRIM IS NOT NULL
                  AND a.EMAILDLA IS NOT NULL
                  AND a.REINSCODE = (SELECT r.REINSURERID
                                       FROM POOLDATA.T_REINSURER r
                                      WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:4))
                                      ORDER BY r.REINSURERID DESC
                                      FETCH NEXT 1 ROWS ONLY))
 ORDER BY c.CLAIMNO
OFFSET :5 ROWS FETCH NEXT :6 ROWS ONLY

-- name: count_dla
-- Tabel ringkas "Status / Jumlah" untuk daftar DLA.
--
-- Bind: :1 penanda pencarian · :2 pola pencarian · :3 login
--
-- Ia mengelompokkan kode yang SUDAH diganti, bukan kode aslinya. Mengelompokkan kode asli
-- akan menghasilkan angka yang tidak dapat dicocokkan dengan kolom Status di bawahnya.
SELECT CASE WHEN c.ISPENDINGCLOSE = 'true' THEN '1139'
            ELSE c.STATUSCLAIM END       AS STATUS_CODE,
       MAX(s.LSC_NOTE)                   AS STATUS_LABEL,
       COUNT(*)                          AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
  LEFT JOIN POOLDATA.M_STS_CLAIM s
         ON s.LSC_ID = CASE WHEN c.ISPENDINGCLOSE = 'true' THEN '1139'
                            ELSE c.STATUSCLAIM END
 WHERE c.GROUPPANEL NOT IN ('002', '005')
   AND (c.STATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
        OR (c.STATUSWORK = 'Resolved-Completed' AND c.ISPENDINGCLOSE = 'true'))
   AND (:1 IS NULL OR UPPER(c.CLAIMID) LIKE :2 ESCAPE '\')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_DLALIST a
                WHERE a.CLAIMID = c.CLAIMID
                  AND a.ISKIRIM = '1'
                  AND a.TGLKIRIM IS NOT NULL
                  AND a.EMAILDLA IS NOT NULL
                  AND a.REINSCODE = (SELECT r.REINSURERID
                                       FROM POOLDATA.T_REINSURER r
                                      WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:3))
                                      ORDER BY r.REINSURERID DESC
                                      FETCH NEXT 1 ROWS ONLY))
 GROUP BY CASE WHEN c.ISPENDINGCLOSE = 'true' THEN '1139'
               ELSE c.STATUSCLAIM END
 ORDER BY 1

-- name: list_close
-- Klaim yang sudah selesai dan tidak lagi menunggu penutupan.
--
-- Bind: :1 login (kolom No PLA) · :2 penanda pencarian · :3 pola pencarian
--       :4 login (syarat PLA terkirim) · :5 offset · :6 jumlah baris
--
-- DUA hal yang mudah dikira salah ketik, dan keduanya memang begitu di Pega:
--
--   1. Disaring T_PLALIST, BUKAN T_DLALIST — meski ini daftar klaim yang sudah selesai.
--   2. Memakai `IN` atas SELURUH kode reasuradur milik login, sementara kolom No PLA di
--      atasnya tetap memakai kode TERTINGGI saja. Satu kueri, dua aturan berbeda.
--
-- `Resolved-Rejected` TIDAK termasuk. Klaim yang ditolak tidak muncul di daftar mana pun
-- pada layar ini, sehingga klaim yang sudah dikirimi PLA lalu ditolak menghilang dari
-- pandangan reasuradur tanpa satu pun pemberitahuan.
SELECT c.CLAIMID                         AS CLAIM_KEY,
       c.CLAIMNO                         AS CLAIM_NO,
       c.NOPOLIS                         AS POLICY_NO,
       c.QQNAME                          AS INSURED,
       c.BUSINESSNAME                    AS BUSINESS_NAME,
       c.REGISTERDATE                    AS REGISTER_DATE,
       c.DATEOFLOSS                      AS LOSS_DATE,
       c.PICTEKNIK                       AS PIC_TEKNIK,
       c.STATUSCLAIM                     AS STATUS_CODE,
       s.LSC_NOTE                        AS STATUS_LABEL,
       (SELECT p.NOPLA
          FROM POOLDATA.T_PLALIST p
         WHERE p.CLAIMID = c.CLAIMID
           AND p.REINSCODE = (SELECT r.REINSURERID
                                FROM POOLDATA.T_REINSURER r
                               WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:1))
                               ORDER BY r.REINSURERID DESC
                               FETCH NEXT 1 ROWS ONLY)
         ORDER BY p.REVISI DESC
         FETCH NEXT 1 ROWS ONLY)         AS ADVICE_NO,
       c.CLOSECLAIMNOTE                  AS CLOSE_NOTE,
       COUNT(*) OVER ()                  AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
  LEFT JOIN POOLDATA.M_STS_CLAIM s
         ON s.LSC_ID = c.STATUSCLAIM
 WHERE c.GROUPPANEL NOT IN ('002', '005')
   AND c.STATUSWORK = 'Resolved-Completed'
   AND (c.ISPENDINGCLOSE <> 'true' OR c.ISPENDINGCLOSE IS NULL)
   AND (:2 IS NULL OR UPPER(c.CLAIMID) LIKE :3 ESCAPE '\')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_PLALIST a
                WHERE a.CLAIMID = c.CLAIMID
                  AND a.ISKIRIM = '1'
                  AND a.TGLKIRIM IS NOT NULL
                  AND a.EMAILPLA IS NOT NULL
                  AND a.REINSCODE IN (SELECT r.REINSURERID
                                        FROM POOLDATA.T_REINSURER r
                                       WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:4))))
 ORDER BY c.CLAIMNO
OFFSET :5 ROWS FETCH NEXT :6 ROWS ONLY

-- name: count_close
-- Tabel ringkas "Status / Jumlah" untuk daftar Close.
--
-- Bind: :1 penanda pencarian · :2 pola pencarian · :3 login
SELECT c.STATUSCLAIM                     AS STATUS_CODE,
       MAX(s.LSC_NOTE)                   AS STATUS_LABEL,
       COUNT(*)                          AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
  LEFT JOIN POOLDATA.M_STS_CLAIM s
         ON s.LSC_ID = c.STATUSCLAIM
 WHERE c.GROUPPANEL NOT IN ('002', '005')
   AND c.STATUSWORK = 'Resolved-Completed'
   AND (c.ISPENDINGCLOSE <> 'true' OR c.ISPENDINGCLOSE IS NULL)
   AND (:1 IS NULL OR UPPER(c.CLAIMID) LIKE :2 ESCAPE '\')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_PLALIST a
                WHERE a.CLAIMID = c.CLAIMID
                  AND a.ISKIRIM = '1'
                  AND a.TGLKIRIM IS NOT NULL
                  AND a.EMAILPLA IS NOT NULL
                  AND a.REINSCODE IN (SELECT r.REINSURERID
                                        FROM POOLDATA.T_REINSURER r
                                       WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:3))))
 GROUP BY c.STATUSCLAIM
 ORDER BY c.STATUSCLAIM

-- name: xol_summary
-- Grid "DATA PLA DLA XOL KLAIM" — ringkasan pemberitahuan XOL yang SUDAH terkirim.
--
-- Bind: :1 login (bagian DLA) · :2 login (bagian PLA)
--
-- # Loginnya DITURUNKAN DARI PEMANGGIL, dan di sinilah ia berbeda dari Pega
--
-- Kueri lama merangkai `Local.loginreas`, yang `Activity/SetDataPLADLA-Act.xml` tetapkan
-- SATU KALI ke nama satu reasuradur tertentu dan tidak pernah ditimpa. Akibatnya setiap
-- reasuradur yang membuka layar itu melihat ringkasan XOL milik mitra lain.
--
-- Itu kebocoran data antar pihak ketiga, bukan keanehan yang layak ditiru. `D-15` melarang
-- nilai bisnis ditulis tetap, dan di sini pelanggarannya bukan sekadar soal kerapian.
-- Selisihnya dinyatakan di PlannedDifferences.
SELECT YEAR_OF                           AS YEAR_OF,
       CAUSE_OF_LOSS                     AS CAUSE_OF_LOSS,
       ADVICE_KIND                       AS ADVICE_KIND,
       LAST_INSERT                       AS LAST_INSERT
  FROM (SELECT d.TAHUN            AS YEAR_OF,
               d.CAUSEOFLOSS      AS CAUSE_OF_LOSS,
               'DLA'              AS ADVICE_KIND,
               MAX(d.TGLINSERT)   AS LAST_INSERT
          FROM POOLDATA.T_DLA_XOL d
         WHERE d.SENDDATE IS NOT NULL
           AND d.IDREAS IN (SELECT r.REINSURERID
                              FROM POOLDATA.T_REINSURER r
                             WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:1)))
         GROUP BY d.TAHUN, d.CAUSEOFLOSS
         UNION
        SELECT p.TAHUN            AS YEAR_OF,
               p.CAUSEOFLOSS      AS CAUSE_OF_LOSS,
               'PLA'              AS ADVICE_KIND,
               MAX(p.TGLINSERT)   AS LAST_INSERT
          FROM POOLDATA.T_PLA_XOL p
         WHERE p.SENDDATE IS NOT NULL
           AND p.IDREAS IN (SELECT r.REINSURERID
                              FROM POOLDATA.T_REINSURER r
                             WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:2)))
         GROUP BY p.TAHUN, p.CAUSEOFLOSS)
 ORDER BY ADVICE_KIND, YEAR_OF DESC
