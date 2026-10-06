-- Kueri modul Inbox RCL (`MENU_ID 62`, `RCL_Harness`).
--
-- Nama kueri berbahasa Inggris (`D-80`); nama tabel dan kolom tetap seperti aslinya.
-- Tidak ada satu pun pernyataan yang menulis.
--
-- ============================================================================
-- SUMBERNYA POOLDATA.TC_PNC_PUCL
-- ============================================================================
--
-- Work Owner menetapkan 2026-10-05: daftar Inbox RCL dan layar kerja `RCLDokter` membaca
-- POOLDATA.TC_PNC_PUCL — satu baris per klaim, berisi `ClaimData.PUCLStatus`. Tabel yang
-- sama dibaca Inbox RCL/PUCL (`inboxrclpucl.sql`), dengan penyaring berbeda.
--
-- Sebelumnya (2026-09-27) modul ini membaca T_CLAIMLIST_ADMIN beserta tiga kolom migrasi
-- 0012. Kolom itu tidak dibaca lagi; migrasinya dibiarkan karena sudah dijalankan.
--
-- ============================================================================
-- PEMETAAN PENYARING `InboxRCLDokter_RD` (A AND B AND C AND D)
-- ============================================================================
--
--   A  pxAssignedOperatorID = assign        UPPER(TRIM(p.ASSIGNED_OPERATOR_ID)) = UPPER(:1)
--   B  pyStatusWork != Resolved-Completed   p.STATUS_WORK <> :2
--   C  TanggalAnalystSendRCL IS NOT NULL    p.TGL_KIRIM_PUCL IS NOT NULL
--   D  NamaDokterRCL = assign               TRIM(p.RCL_PUCL) IN (:3, :4)  -- '1' RCL, '3' MSIG
--
-- C: `Activity/SendToPUCL-Act.xml` langkah 8 mengisi TanggalAnalystSendRCL dan
--    PUCLStatus.TanggalKirimPUCL pada langkah yang sama.
-- D: tidak ada kolom nama dokter. `RouterRCLDokter` menugaskan ke NamaDokterRCL, sehingga
--    dokter = pemilik penugasan (A). Yang tersisa dari D: klaimnya melewati dokter — RCL dan
--    MSIG saja; PUCL ('2') tidak.
--
-- Urutan `.pxCreateDateTime DESC, .pyID DESC` -> TGL_CREATE_PUCL DESC, CLAIMID DESC.
--
-- ============================================================================
-- PEMETAAN KOLOM
-- ============================================================================
--
--   judul di layar          properti Pega                              kolom TC_PNC_PUCL
--   ----------------------- ------------------------------------------ ------------------------
--   Nomor Case              .pyID                                      CLAIMID
--   No Polis                .Policy.PolicyNo                           POLICY_NO
--   Nama Tertanggung        .Policy.QQName                             QQ_NAME
--   Tanggal Masuk Inbox     .ClaimData.TanggalAnalystSendRCL           TGL_KIRIM_PUCL
--   Deskripsi Analyst       .ClaimData.PUCLStatus.KomentarAnalisator   KOMENTAR_ANALISATOR
--
-- Layar kerja `RCLDokter` (detail):
--
--   Catatan dari Analyst            .PUCLStatus.KomentarAnalisator   KOMENTAR_ANALISATOR
--   Alasan Klaim Ditolak/RCL · MSIG .PUCLStatus.Keterangan2          KETERANGAN2
--   Alasan Dokter                   .ClaimData.AlasanDokterRejectRCL ALASAN_DOKTER_REJECT_RCL
--   (penentu mode)                  .PUCLStatus.RCL_PUCL             RCL_PUCL
--
-- ALASAN_DOKTER_REJECT_RCL ditambahkan Work Owner ke TC_PNC_PUCL pada 2026-10-05
-- (VARCHAR2(4000), nullable).
--
-- SETIAP penanda bind muncul tepat satu kali: godror mengikat menurut URUTAN KEMUNCULAN,
-- bukan nomor penanda (ORA-01008, terbukti 2026-09-27).

-- name: operator_for
-- LOGIN_ID pemanggil dari POOLDATA.M_LOGIN_PNC — pengganti `RDB List/GetOperatorID-SQL.xml`
-- (TempOperator.City dari T_ACCESS_GROUP_PNC, yang tidak dipakai lagi; Work Owner 2026-10-05).
-- MAX supaya selalu tepat satu baris; tidak ada terbaca NULL, bukan ErrNoRows.
--
-- Bind:
--   :1  login pemanggil (huruf besar, dipangkas)
--   :2  status aktif — '1'
SELECT MAX(UPPER(TRIM(l.LOGIN_ID)))
  FROM POOLDATA.M_LOGIN_PNC l
 WHERE UPPER(TRIM(l.LOGIN_ID)) = :1
   AND l.ACTIVE_STATUS = :2

-- name: list_tasks
-- Satu halaman antrean RCL Dokter milik seorang operator.
--
-- Bind:
--   :1  LOGIN_ID — penyaring A
--   :2  status kerja yang DIKECUALIKAN — "Resolved-Completed"
--   :3  '1' (RCL)  — penyaring D
--   :4  '3' (MSIG) — penyaring D
--   :5  kata kunci, atau NULL bila kotak carinya kosong
--   :6  pola LIKE untuk CLAIMID   ('%KATA%', huruf besar, karakter khusus di-escape)
--   :7  pola LIKE untuk POLICY_NO (sama dengan :6)
--   :8  offset
--   :9  jumlah baris
SELECT p.CLAIMID                     AS CASE_ID,
       p.POLICY_NO                   AS POLICY_NUMBER,
       p.QQ_NAME                     AS INSURED_NAME,
       p.TGL_KIRIM_PUCL              AS SENT_TO_RCL_AT,
       p.KOMENTAR_ANALISATOR         AS ANALYST_NOTE,
       p.RCL_PUCL                    AS RCL_MODE,
       p.TGL_CREATE_PUCL             AS REGISTERED_AT,
       p.STATUS_WORK                 AS PROCESS_STATUS,
       p.ASSIGNED_OPERATOR_ID        AS ASSIGNED_OPERATOR,
       COUNT(*) OVER ()              AS TOTAL_ROWS
  FROM POOLDATA.TC_PNC_PUCL p
 WHERE UPPER(TRIM(p.ASSIGNED_OPERATOR_ID)) = UPPER(:1)
   AND p.STATUS_WORK <> :2
   AND p.TGL_KIRIM_PUCL IS NOT NULL
   AND TRIM(p.RCL_PUCL) IN (:3, :4)
   AND (:5 IS NULL
        OR UPPER(p.CLAIMID) LIKE :6 ESCAPE '\'
        OR UPPER(p.POLICY_NO) LIKE :7 ESCAPE '\')
 ORDER BY p.TGL_CREATE_PUCL DESC, p.CLAIMID DESC
OFFSET :8 ROWS FETCH NEXT :9 ROWS ONLY

-- name: claim_detail
-- Isi layar kerja `RCLDokter` untuk satu klaim — HANYA bila klaim itu ada di antrean
-- pemanggil (penyaring A–D sama dengan list_tasks). Klaim milik orang lain tidak terbaca.
--
-- Bind:
--   :1  nomor klaim (CLAIMID)
--   :2  LOGIN_ID — penyaring A
--   :3  status kerja yang DIKECUALIKAN — "Resolved-Completed"
--   :4  '1' (RCL)
--   :5  '3' (MSIG)
SELECT p.CLAIMID                     AS CASE_ID,
       p.POLICY_NO                   AS POLICY_NUMBER,
       p.QQ_NAME                     AS INSURED_NAME,
       p.RCL_PUCL                    AS RCL_MODE,
       p.KOMENTAR_ANALISATOR         AS ANALYST_NOTE,
       p.KETERANGAN2                 AS REASON,
       p.ALASAN_DOKTER_REJECT_RCL    AS DOCTOR_REASON,
       p.STATUS_CLAIM                AS STATUS_CLAIM,
       p.STATUS_WORK                 AS PROCESS_STATUS,
       p.ASSIGNED_OPERATOR_ID        AS ASSIGNED_OPERATOR,
       p.TGL_KIRIM_PUCL              AS SENT_TO_RCL_AT
  FROM POOLDATA.TC_PNC_PUCL p
 WHERE UPPER(TRIM(p.CLAIMID)) = UPPER(:1)
   AND UPPER(TRIM(p.ASSIGNED_OPERATOR_ID)) = UPPER(:2)
   AND p.STATUS_WORK <> :3
   AND p.TGL_KIRIM_PUCL IS NOT NULL
   AND TRIM(p.RCL_PUCL) IN (:4, :5)

-- name: check_tables
-- Dipakai `-periksa`: memastikan KEDUA tabel terbaca. `WHERE 1 = 0` — hak baca, bukan isi.
SELECT COUNT(*) AS PROBE
  FROM POOLDATA.TC_PNC_PUCL p
       CROSS JOIN POOLDATA.M_LOGIN_PNC l
 WHERE 1 = 0

-- name: check_columns
-- Dipakai `-periksa`: memastikan kolom yang ditambahkan 2026-10-05 memang ada.
SELECT COUNT(p.ALASAN_DOKTER_REJECT_RCL) AS PROBE_DOCTOR_REASON
  FROM POOLDATA.TC_PNC_PUCL p
 WHERE 1 = 0
