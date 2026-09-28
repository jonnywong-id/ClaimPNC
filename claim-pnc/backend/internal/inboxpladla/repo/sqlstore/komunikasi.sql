-- Ketiga daftar KOMUNIKASI pada layar Inbox PLA DLA — `SetDataPLADLA` tipe 4, 5, dan 6.
--
-- Berkas terpisah dari inboxpladla.sql, dan pemuatnya menggabungkan keduanya: `loadQueries`
-- membaca SELURUH berkas `.sql` di paket ini dan menolak nama kueri yang sama dua kali.
-- Pemisahannya menjaga tiap berkas tetap dapat dibaca sekali duduk.
--
-- ============================================================================
-- ASALNYA SATU KUERI, DAN YANG MEMBEDAKAN KETIGANYA DIRANGKAI DI LUARNYA
-- ============================================================================
--
-- `RDB List/BrowseCommunicationReas-SQL.xml` dipakai ketiga tipe. Yang membedakannya adalah
-- dua nilai yang disiapkan `Activity/SetDataPLADLA-Act.xml` sebelum ia dijalankan:
--
--   tipe 4   TempView.DistrictID = "0"   TempView.District = "and c.COMMUNICATE_TO='…'"
--   tipe 5   TempView.DistrictID = "0"   TempView.District = "and c.sender='…'"
--   tipe 6   TempView.DistrictID = "1"   TempView.District = "and c.sender='…'"
--
-- Yang pertama DIIKAT (`{TempView.DistrictID}`); yang kedua DIRANGKAI (`{ASIS:…}`) beserta
-- login pemanggil di dalamnya. Di sini keduanya diikat.
--
-- Perbedaan sisi percakapan menjadi DUA KUERI, bukan satu kueri yang memilih sisinya lewat
-- bind. Bentuk `(:n = 'penerima' AND … OR :n = 'pengirim' AND …)` akan membuat Oracle
-- kehilangan index pada kedua kolomnya sekaligus — dan tabel percakapan tumbuh seiring
-- seluruh klaim, bukan seiring klaim satu mitra.
--
-- ============================================================================
-- DUA HAL YANG BERBEDA DARI KETIGA DAFTAR PEMBERITAHUAN — keduanya DIBAWA
-- ============================================================================
--
--  1. TIDAK ADA penyaring `GROUPPANEL`. Ketiga daftar pemberitahuan mengecualikan `002`
--     (Personal Accident) dan `005` (Travel); `BrowseCommunicationReas` tidak memuat satu
--     pun syarat itu. Akibatnya klaim PA dapat muncul di sini sementara ia tidak akan
--     pernah muncul di ketiga daftar di sebelah kiri.
--
--  2. TIDAK ADA syarat dokumen pemberitahuan terkirim. Sebuah klaim masuk daftar ini karena
--     ada PERCAKAPAN — bukan karena ada PLA maupun DLA.
--
-- Kolom "PLA No" tetap diambil dengan rantai reasuradur yang sama, dan karena butir kedua ia
-- memang BOLEH kosong di sini. Itu bukan tanda kerusakan.
--
-- ============================================================================
-- `CASEID` DI TABEL INI BERISI KUNCI KLAIM, BUKAN NAMA KANAL
-- ============================================================================
--
-- Modul `inboxkomunikasicabang` membaca tabel yang SAMA dan memakai `CASEID` sebagai nama
-- kanal (`'CABANG'`). Di sini ia berisi `T_CLAIM_PNC.CLAIMID` — kueri aslinya menuliskannya
-- sebagai `a.claimid = c.caseid`. Satu kolom, dua arti, keduanya ada di produksi.
--
-- ============================================================================
-- PENOMORAN BIND MENAIK MENURUT URUTAN KEMUNCULAN
-- ============================================================================
--
-- Oracle mengikat argumen menurut urutan KEMUNCULAN penanda di dalam teks, bukan menurut
-- angka pada `:n`. Karena kolom "PLA No" berada di klausa SELECT — sebelum WHERE — login
-- untuknya bernomor `:1`, bukan `:3`. `TestBindMarkersAppearInAscendingOrder` menjaganya.

-- name: list_komunikasi_recipient
-- Klaim yang punya percakapan DITUJUKAN kepada pemanggil (`COMMUNICATE_TO`).
--
-- Bind: :1 login (kolom No PLA) · :2 penanda pencarian · :3 pola pencarian
--       :4 status percakapan · :5 login (tujuan percakapan)
--       :6 offset · :7 jumlah baris
SELECT c.CLAIMID                         AS CLAIM_KEY,
       c.CLAIMNO                         AS CLAIM_NO,
       c.NOPOLIS                         AS POLICY_NO,
       c.QQNAME                          AS INSURED,
       c.BUSINESSNAME                    AS BUSINESS_NAME,
       c.REGISTERDATE                    AS REGISTER_DATE,
       c.DATEOFLOSS                      AS LOSS_DATE,
       c.PICTEKNIK                       AS PIC_TEKNIK,
       w.STATUSCLAIM_1                   AS STATUS_CODE,
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
  LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
         ON w.PZINSKEY = c.CLAIMID
  LEFT JOIN POOLDATA.M_STS_CLAIM s
         ON s.LSC_ID = w.STATUSCLAIM_1
 WHERE c.STATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (:2 IS NULL OR UPPER(c.CLAIMID) LIKE :3 ESCAPE '\')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.M_KOMUNIKASI_PNC k
                WHERE k.CASEID = c.CLAIMID
                  AND k.KOMUNIKASISTATUS = :4
                  AND UPPER(TRIM(k.COMMUNICATE_TO)) = UPPER(TRIM(:5)))
 ORDER BY c.REGISTERDATE DESC, c.CLAIMNO
OFFSET :6 ROWS FETCH NEXT :7 ROWS ONLY

-- name: count_komunikasi_recipient
-- Tabel ringkas "Status / Jumlah" untuk daftar komunikasi masuk.
--
-- Penyaringnya WAJIB sama persis dengan list_komunikasi_recipient, kecuali kolom "PLA No"
-- yang tidak diambil — sehingga rantai reasuradur untuk kolom itu tidak ada di sini.
--
-- Bind: :1 penanda pencarian · :2 pola pencarian · :3 status percakapan · :4 login
SELECT w.STATUSCLAIM_1                   AS STATUS_CODE,
       MAX(s.LSC_NOTE)                   AS STATUS_LABEL,
       COUNT(*)                          AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
  LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
         ON w.PZINSKEY = c.CLAIMID
  LEFT JOIN POOLDATA.M_STS_CLAIM s
         ON s.LSC_ID = w.STATUSCLAIM_1
 WHERE c.STATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (:1 IS NULL OR UPPER(c.CLAIMID) LIKE :2 ESCAPE '\')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.M_KOMUNIKASI_PNC k
                WHERE k.CASEID = c.CLAIMID
                  AND k.KOMUNIKASISTATUS = :3
                  AND UPPER(TRIM(k.COMMUNICATE_TO)) = UPPER(TRIM(:4)))
 GROUP BY w.STATUSCLAIM_1
 ORDER BY w.STATUSCLAIM_1

-- name: list_komunikasi_sender
-- Klaim yang punya percakapan DIKIRIM pemanggil (`SENDER`).
--
-- Dipakai DUA tab sekaligus, yang dibedakan NILAI bind status — `0` belum dijawab, `1`
-- sudah. Keduanya memakai kueri yang sama karena penyaringnya memang sama kecuali nilai
-- itu; menyalinnya menjadi dua kueri akan membuat perubahan berikutnya berlaku di satu
-- tempat saja.
--
-- Bind: :1 login (kolom No PLA) · :2 penanda pencarian · :3 pola pencarian
--       :4 status percakapan · :5 login (pengirim percakapan)
--       :6 offset · :7 jumlah baris
SELECT c.CLAIMID                         AS CLAIM_KEY,
       c.CLAIMNO                         AS CLAIM_NO,
       c.NOPOLIS                         AS POLICY_NO,
       c.QQNAME                          AS INSURED,
       c.BUSINESSNAME                    AS BUSINESS_NAME,
       c.REGISTERDATE                    AS REGISTER_DATE,
       c.DATEOFLOSS                      AS LOSS_DATE,
       c.PICTEKNIK                       AS PIC_TEKNIK,
       w.STATUSCLAIM_1                   AS STATUS_CODE,
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
  LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
         ON w.PZINSKEY = c.CLAIMID
  LEFT JOIN POOLDATA.M_STS_CLAIM s
         ON s.LSC_ID = w.STATUSCLAIM_1
 WHERE c.STATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (:2 IS NULL OR UPPER(c.CLAIMID) LIKE :3 ESCAPE '\')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.M_KOMUNIKASI_PNC k
                WHERE k.CASEID = c.CLAIMID
                  AND k.KOMUNIKASISTATUS = :4
                  AND UPPER(TRIM(k.SENDER)) = UPPER(TRIM(:5)))
 ORDER BY c.REGISTERDATE DESC, c.CLAIMNO
OFFSET :6 ROWS FETCH NEXT :7 ROWS ONLY

-- name: count_komunikasi_sender
-- Tabel ringkas "Status / Jumlah" untuk kedua daftar komunikasi terkirim.
--
-- Bind: :1 penanda pencarian · :2 pola pencarian · :3 status percakapan · :4 login
SELECT w.STATUSCLAIM_1                   AS STATUS_CODE,
       MAX(s.LSC_NOTE)                   AS STATUS_LABEL,
       COUNT(*)                          AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
  LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
         ON w.PZINSKEY = c.CLAIMID
  LEFT JOIN POOLDATA.M_STS_CLAIM s
         ON s.LSC_ID = w.STATUSCLAIM_1
 WHERE c.STATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (:1 IS NULL OR UPPER(c.CLAIMID) LIKE :2 ESCAPE '\')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.M_KOMUNIKASI_PNC k
                WHERE k.CASEID = c.CLAIMID
                  AND k.KOMUNIKASISTATUS = :3
                  AND UPPER(TRIM(k.SENDER)) = UPPER(TRIM(:4)))
 GROUP BY w.STATUSCLAIM_1
 ORDER BY w.STATUSCLAIM_1
