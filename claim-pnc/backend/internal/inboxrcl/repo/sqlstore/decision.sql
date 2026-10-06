-- Kueri keputusan dokter RCL — pengganti `Activity/SendToPUCL-Act.xml` (lihat
-- `inboxrcl/decision.go` untuk langkah yang dibawa dan yang tidak).
--
-- Berkas ini SATU-SATUNYA tempat modul Inbox RCL menulis. Seluruh pernyataannya dijalankan
-- dalam SATU transaksi (`D-68`): tugas lama yang tertutup tanpa tugas baru terbuka membuat
-- klaim hilang dari setiap inbox, dan status yang berubah tanpa perpindahan tahap membuat
-- klaim tampil di antrean yang salah.
--
-- Tabel yang ditulis — tidak satu pun tabel engine Pega (`P-1`):
--
--   POOLDATA.TC_PNC_PUCL             status RCL/PUCL klaim (keputusan Work Owner 2026-10-05)
--   POOLDATA.T_CLAIMLIST_ADMIN       baris daftar kerja klaim PNCN saja (kunci = nomor PNCN)
--   CPNC_TUGAS                       perpindahan tahap — padanan Ticket rule
--   POOLDATA.LIST_HISTORY_CLAIM_PNC  riwayat klaim
--
-- SETIAP penanda bind muncul tepat satu kali dan berurutan: godror mengikat menurut URUTAN
-- KEMUNCULAN (ORA-01008).

-- name: decision_lock
-- Mengunci baris TC_PNC_PUCL klaim — HANYA bila klaim masih di antrean pemanggil (penyaring
-- yang sama dengan `claim_detail`). Dua tab yang menekan tombol bersamaan: yang kedua
-- menunggu kunci, lalu tidak menemukan barisnya lagi karena ASSIGNED_OPERATOR_ID sudah
-- berpindah, dan tidak menulis apa pun.
--
-- Kolom yang dibaca adalah yang dibaca `SendToPUCL` sebelum menulis (langkah 2–7), ditambah
-- nilai yang harus dipertahankan bila langkahnya tidak berlaku.
--
-- Bind: :1 nomor klaim · :2 operator · :3 status kerja selesai · :4 mode RCL · :5 mode MSIG
SELECT p.CLAIMID                  AS CASE_ID,
       TRIM(p.RCL_PUCL)           AS RCL_MODE,
       p.STATUS_CASE              AS STATUS_CASE,
       p.PUCL_APPROVE             AS PUCL_APPROVE,
       p.STATUS_KLAIM             AS STATUS_KLAIM,
       p.TGL_SELESAI_RI           AS DISCHARGED_AT,
       p.DATE_OF_LOSS             AS DATE_OF_LOSS,
       p.TGL_CETAK_DOKUMEN_PUCL   AS LETTER_PRINTED_AT,
       p.ALASAN_DOKTER_REJECT_RCL AS DOCTOR_REASON,
       p.USER_TEKNIS              AS USER_TEKNIS
  FROM POOLDATA.TC_PNC_PUCL p
 WHERE UPPER(TRIM(p.CLAIMID)) = UPPER(:1)
   AND UPPER(TRIM(p.ASSIGNED_OPERATOR_ID)) = UPPER(:2)
   AND p.STATUS_WORK <> :3
   AND p.TGL_KIRIM_PUCL IS NOT NULL
   AND TRIM(p.RCL_PUCL) IN (:4, :5)
   FOR UPDATE

-- name: decision_technical_pic
-- PIC Teknik klaim — pemilik tugas Send To Analis saat Dokter Tidak Setuju/Back.
--
-- `PNCTeknikRouter` tidak ada di export (`R-04`). Sumbernya `T_CLAIM_PNC.PICTEKNIK`, sama
-- dengan Inbox RCL/PUCL (Work Owner 2026-10-02: `USERTEKNIS_1` diambil dari `PICTEKNIK`).
-- MAX atas himpunan kosong mengembalikan satu baris NULL, bukan nol baris.
--
-- Bind: :1 nomor klaim
SELECT MAX(TRIM(c.PICTEKNIK)) AS TECHNICAL_PIC
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE TRIM(c.CLAIMNO) = TRIM(:1)

-- name: decision_claim_key
-- KLAIM_ID tugas klaim ini — disalin dari tugasnya sendiri, tugas terbuka lebih dulu, supaya
-- tugas baru tertaut ke klaim dengan kunci yang SAMA dengan yang ditulis modul alur.
--
-- Bind: :1 nomor klaim
SELECT t.KLAIM_ID AS CLAIM_KEY
  FROM CPNC_TUGAS t
 WHERE t.NOMOR_KLAIM = :1
 ORDER BY CASE WHEN t.SELESAI_PADA IS NULL THEN 0 ELSE 1 END, t.DIBUAT_PADA DESC
 FETCH FIRST 1 ROWS ONLY

-- name: decision_update_pucl
-- Menulis keadaan TC_PNC_PUCL sesudah keputusan — seluruh kolom yang disentuh `SendToPUCL`
-- ditambah pemilik penugasan.
--
-- ASSIGNED_OPERATOR_ID berpindah ke pemilik tahap berikutnya: akun antrean `RCLPUCL` (Setuju,
-- Submit) atau PIC Teknik (Tidak Setuju, Back). Itulah yang mengeluarkan klaim dari antrean
-- Dokter — penyaring A.
--
-- Bind: :1 status claim · :2 status klaim · :3 status case · :4 PUCL approve
--       :5 tanggal cetak · :6 tanggal kirim PUCL · :7 tanggal analyst send RCL
--       :8 lama klaim · :9 alasan dokter · :10 pemilik penugasan · :11 nomor klaim
UPDATE POOLDATA.TC_PNC_PUCL
   SET STATUS_CLAIM             = :1,
       STATUS_KLAIM             = :2,
       STATUS_CASE              = :3,
       PUCL_APPROVE             = :4,
       TGL_CETAK_DOKUMEN_PUCL   = :5,
       TGL_KIRIM_PUCL           = :6,
       TGL_ANALYST_SEND_RCL     = :7,
       LAMA_KLAIM               = :8,
       ALASAN_DOKTER_REJECT_RCL = :9,
       ASSIGNED_OPERATOR_ID     = :10
 WHERE TRIM(CLAIMID) = TRIM(:11)

-- name: decision_update_worklist
-- Baris daftar kerja klaim (My Inbox) mengikuti tahap baru.
--
-- Hanya baris PNCN yang kenanya: kuncinya nomor PNCN apa adanya, sedangkan baris Pega
-- berkunci `ASM-FW-GCNMFW-WORK PNC-…` — tidak pernah tersentuh. Nol baris BUKAN galat.
--
-- PXASSIGNEDOPERATORID kosong untuk tugas antrean bersama yang belum diambil, sama dengan
-- aturan modul Registrasi yang mengisi tabel ini.
--
-- Bind: :1 status claim · :2 tanggal analyst send RCL · :3 pemilik · :4 nama tahap
--       :5 nomor klaim
UPDATE POOLDATA.T_CLAIMLIST_ADMIN
   SET STATUSCLAIM_1           = :1,
       TANGGALANALYSTSENDRCL_1 = :2,
       PXASSIGNEDOPERATORID    = :3,
       PXTASKLABEL             = :4
 WHERE PZINSKEY = :5

-- name: decision_close_tasks
-- Menutup seluruh tugas klaim yang masih terbuka — tugas tahap RCL Dokter. Tidak dihapus;
-- SELESAI_PADA diisi (`D-66`). Nol baris BUKAN galat: klaim yang dimulai di Pega belum pernah
-- punya tugas di tabel ini.
--
-- Bind: :1 waktu selesai · :2 alasan (nama Ticket) · :3 nomor klaim
UPDATE CPNC_TUGAS
   SET SELESAI_PADA   = :1,
       ALASAN_SELESAI = :2
 WHERE NOMOR_KLAIM    = :3
   AND SELESAI_PADA IS NULL

-- name: decision_open_task
-- Membuka tugas tahap berikutnya — akibat Ticket rule SendtoPUCL atau SendtoAnalysator.
--
-- Antrean bersama: WORKBASKET terisi, PEMILIK dan DIAMBIL_PADA kosong. Worklist: PEMILIK
-- terisi dan DIAMBIL_PADA sama dengan DIBUAT_PADA — bertuan sejak lahir (`D-26`).
--
-- Bind: :1 id · :2 klaim id · :3 nomor klaim · :4 tahap · :5 antrean · :6 workbasket
--       :7 pemilik · :8 dibuat pada · :9 diambil pada
INSERT INTO CPNC_TUGAS
       (ID, KLAIM_ID, NOMOR_KLAIM, TAHAP, ANTREAN, WORKBASKET, PEMILIK,
        DIBUAT_PADA, DIAMBIL_PADA, SELESAI_PADA, ALASAN_SELESAI)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, NULL, NULL)

-- name: decision_insert_history
-- Satu baris riwayat — pengganti `InsertHistoryClaimPNC` (`Database/
-- PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc`: CREATEDATETIME = CURRENT_TIMESTAMP).
--
-- Di Oracle CURRENT_TIMESTAMP dihitung per pernyataan, sehingga kedua baris riwayat satu
-- keputusan tersimpan berurutan.
--
-- Bind: :1 CASEID · :2 catatan · :3 pelaku
INSERT INTO POOLDATA.LIST_HISTORY_CLAIM_PNC (CASEID, CREATEDATETIME, STATUSNOTE, USERUPDATE)
VALUES (:1, CURRENT_TIMESTAMP, :2, :3)
