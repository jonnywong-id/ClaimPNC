-- Tombol Tutup Klaim — Activity/CloseClaim-act.xml dan SetStatusCloseSementara-act.xml.

-- name: klaim_tutup_simpan
--
-- Isian dialog Prevent Close Claim yang berkolom, ISPENDINGCLOSE ('true' seperti Pega), dan
-- CLOSECLAIMDATE (CloseClaim langkah 15; jam dinding WIB — Pega menambah 7 jam pada GMT).
-- CLOSECLAIMDATE yang tidak dikirim (tutup sementara) tidak menghapus nilai lama.
UPDATE POOLDATA.T_CLAIM_PNC
   SET CLOSECLAIMNOTE = :1,
       USULAN         = :2,
       EFFORT_CLOSE   = :3,
       KENDALA_CLOSE  = :4,
       ISPENDINGCLOSE = :5,
       CLOSECLAIMDATE = COALESCE(:6, CLOSECLAIMDATE)
 WHERE CLAIMID = :7

-- name: klaim_tutup_log
--
-- RDB List/Insert_to_log_SQL-SQL.xml:
--   INSERT INTO POOLDATA.CloseRejectClaim_Log VALUES ({City}, {CityID}, sysdate, {District},
--   {AlasanKlaim}, {Notes})
-- Kolom disebut namanya (urutan tabel terverifikasi 2026-10-04).
INSERT INTO POOLDATA.CLOSEREJECTCLAIM_LOG (CASEID, NO_KLAIM, TGL_UPDATE, USER_INPUT, ACTION, NOTE)
VALUES (:1, :2, :3, :4, :5, :6)

-- name: klaim_tutup_dashboard
--
-- RDB List/UpdateStsKlaimClose_sql-SQL.xml.
UPDATE POOLDATA.PEGA_DASHBOARDPNC SET STSKLAIM = '3' WHERE NOKLAIM = :1

-- name: klaim_tutup_beban_pic
--
-- RDB List/UpdateTotalJob_sql-SQL.xml (TOTAL_JOB bertipe VARCHAR2; Oracle mengubahnya ke angka).
UPDATE POOLDATA.MST_USER_TEKNIS SET TOTAL_JOB = TOTAL_JOB - 1 WHERE OPERATOR_ID = :1

-- name: klaim_tutup_sementara
--
-- Penanda tutup sementara klaim — dibaca layar klaim dan tombol Tutup Klaim.
SELECT ISPENDINGCLOSE FROM POOLDATA.T_CLAIM_PNC WHERE CLAIMID = :1
