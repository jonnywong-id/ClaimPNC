-- Panel detail Payment Klaim Akseptasi — tombol DETAILS dan SIMPAN.
--
-- Asal: `Section/Akseptasi_PaymentLeader1`, `Activity/GetDataAkseptasiKlaimKasir`, dan
-- `Activity/SaveApprovalAkseptasiPaymentLeader_Act`. Seluruhnya membaca tabel POOLDATA,
-- bukan case Pega — keputusan Work Owner 2026-10-14; pemetaan isiannya ada di
-- inboxmanager/payment.go.
--
-- # Adjustment dicari lewat CLAIMID + NOAKSEPTASI, bukan OBJID/CVGID/ADJID
--
-- Tabel checker punya kolom penunjuk adjustment, tetapi kolomnya hampir selalu kosong:
-- diperiksa 2026-10-14, OBJID/ADJID terisi pada 1 dari 8 baris yang menunggu. Pega sendiri
-- mencari adjustment-nya dengan `.AcceptedNo == Param.noaksept` di dalam case — padanannya di
-- tabel adalah nomor akseptasi pada klaim yang sama.

-- name: payment_detail
-- Satu nomor akseptasi yang SEDANG menunggu.
--
-- Tabel checker dapat memuat nomor akseptasi yang sama lebih dari sekali (diperiksa
-- 2026-10-14: 8 baris menunggu, 6 nomor berbeda). Baris yang tampil adalah yang terbaru
-- diinput — SIMPAN di Pega pun memutuskan seluruh baris bernomor itu sekaligus.
--
-- Bind: :1 nomor akseptasi
SELECT ch.CLAIMID                                    AS CLAIM_ID,
       TRIM(ch.NOAKSEPTASI)                          AS ACCEPTED_NO,
       ch.PIC                                        AS PIC,
       ch.PICNOTES                                   AS REMARK_PIC,
       (SELECT x.CLAIMNO
          FROM POOLDATA.T_CLAIM_PNC x
         WHERE x.CLAIMID = ch.CLAIMID)               AS CLAIM_NO,
       (SELECT c.QQNAME
          FROM POOLDATA.T_CLAIMLIST_ADMIN c
         WHERE c.PZINSKEY = ch.CLAIMID
         FETCH FIRST 1 ROW ONLY)                     AS INSURED_NAME,
       (SELECT u.EMAIL
          FROM POOLDATA.MST_USER_TEKNIK u
         WHERE UPPER(TRIM(u.OPERATOR_ID)) = UPPER(TRIM(ch.PIC))
         FETCH FIRST 1 ROW ONLY)                     AS PIC_EMAIL,
       adj.CLAIMID                                   AS ADJ_CLAIM_ID,
       adj.RECEIVERNAME                              AS RECEIVER,
       adj.PAYMENTTYPE                               AS PAYMENT_TYPE,
       adj.GROSSVALUE                                AS GROSS_VALUE,
       adj.ASM_SHARE_VALUE                           AS CLAIM_VALUE,
       cur.CURRENCY                                  AS CURRENCY,
       adj.CURRENCY                                  AS CURRENCY_ID,
       adj.ASM_SHARE                                 AS SHARE_ASM
  FROM POOLDATA.T_CLAIM_AKSEPTASI_CHECKER ch
  LEFT JOIN POOLDATA.T_CLAIM_ADJUSTMENT adj
    ON adj.CLAIMID = ch.CLAIMID
   AND TRIM(adj.NOAKSEPTASI) = TRIM(ch.NOAKSEPTASI)
  LEFT JOIN POOLDATA.CURRENCY cur
    ON cur.ID = adj.CURRENCY
 WHERE TRIM(ch.NOAKSEPTASI) = :1
   AND TRIM(ch.STSAPP) = '0'
 ORDER BY ch.TGLINPUT DESC, adj.OBJECTID, adj.OBJECTCOVERAGEID, adj.ADJUSTMENTID
 FETCH FIRST 1 ROW ONLY

-- name: payment_history
-- Grid "History Payment Akseptasi Manager".
--
-- Kueri aslinya (`HistoryManagerPaymetApproval_SQL`) TIDAK ada di export. Penggantinya baris
-- checker untuk klaim yang sama yang SUDAH diputuskan — keputusan Work Owner 2026-10-14,
-- BELUM DIVERIFIKASI terhadap layar Pega.
--
-- Bind: :1 CLAIMID
SELECT (SELECT x.CLAIMNO
          FROM POOLDATA.T_CLAIM_PNC x
         WHERE x.CLAIMID = h.CLAIMID) AS CLAIM_NO,
       TRIM(h.NOAKSEPTASI)            AS ACCEPTED_NO,
       h.TGLAPPROVE                   AS DECIDED_AT,
       h.NOTEAPPROVE                  AS NOTE_MANAGER,
       h.PICNOTES                     AS NOTE_PIC,
       TRIM(h.STSAPP)                 AS STATUS
  FROM POOLDATA.T_CLAIM_AKSEPTASI_CHECKER h
 WHERE h.CLAIMID = :1
   AND TRIM(h.STSAPP) <> '0'
 ORDER BY h.TGLAPPROVE DESC NULLS LAST, h.NOAKSEPTASI

-- name: payment_documents
-- Lampiran tombol "Dokumen" — `GetAttachmentFromDB2_Sql` dan `GetOldAttachmentFromDB2_Sql`:
-- `pooldata.data_attachfile where dataid = <nomor akseptasi>`.
--
-- Lampiran baru menyimpan IMAGEID (layanan penyimpanan); lampiran lama menyimpan berkasnya di
-- ATTACHFILE. Isi berkasnya TIDAK dibaca di sini — hanya ada-tidaknya — supaya daftar tidak
-- menarik isi seluruh berkas. `base64encode` kueri lama tidak dibawa (`D-02`).
--
-- Urutannya TETAP dan menjadi kunci lampiran lama (lihat payment_document_content).
--
-- Bind: :1 nomor akseptasi
SELECT a.ATTACHNAME                                     AS DOCUMENT_NAME,
       a.ATTACHMIMETYPE                                 AS MIME_TYPE,
       a.ATTACHNOTE                                     AS NOTE,
       a.INPUTDATE                                      AS INPUT_DATE,
       a.IMAGEID                                        AS IMAGE_ID,
       CASE WHEN a.ATTACHFILE IS NOT NULL THEN 1 ELSE 0 END AS HAS_FILE
  FROM POOLDATA.DATA_ATTACHFILE a
 WHERE TRIM(a.DATAID) = :1
 ORDER BY a.INPUTDATE, a.ATTACHNAME, a.IMAGEID

-- name: payment_document_content
-- Isi satu lampiran lama milik nomor akseptasi itu. Urutannya SAMA PERSIS dengan
-- payment_documents — kunci lampiran adalah nomor urutnya di daftar itu, sehingga hanya
-- baris yang diminta yang isinya ditarik. Kepemilikan dipastikan oleh `DATAID = nomor
-- akseptasi`: kunci dari nomor lain tidak menunjuk apa pun.
--
-- Bind: :1 nomor akseptasi · :2 jumlah baris yang dilompati (nomor urut - 1)
SELECT a.ATTACHNAME     AS DOCUMENT_NAME,
       a.ATTACHMIMETYPE AS MIME_TYPE,
       a.IMAGEID        AS IMAGE_ID,
       a.ATTACHFILE     AS CONTENT
  FROM POOLDATA.DATA_ATTACHFILE a
 WHERE TRIM(a.DATAID) = :1
 ORDER BY a.INPUTDATE, a.ATTACHNAME, a.IMAGEID
OFFSET :2 ROWS FETCH NEXT 1 ROW ONLY

-- name: payment_claim_ids
-- Klaim pemilik nomor akseptasi yang sedang menunggu — dibaca SIMPAN di dalam transaksinya,
-- sebelum baris checker berubah, untuk menandai adjustment klaim itu.
--
-- Bind: :1 nomor akseptasi
SELECT DISTINCT ch.CLAIMID AS CLAIM_ID
  FROM POOLDATA.T_CLAIM_AKSEPTASI_CHECKER ch
 WHERE TRIM(ch.NOAKSEPTASI) = :1
   AND TRIM(ch.STSAPP) = '0'

-- name: payment_marks_probe
-- Apakah kolom penanda migrasi 0017 sudah dibuat DBA. Tidak membaca satu baris pun.
--
-- Dijalankan DI LUAR transaksi SIMPAN: di PostgreSQL pernyataan yang gagal membatalkan seluruh
-- transaksinya, sehingga mencoba UPDATE lalu menelan ORA-00904 hanya aman di Oracle.
SELECT adj.FLAGATASAN, adj.STSPENERIMAKLAIM
  FROM POOLDATA.T_CLAIM_ADJUSTMENT adj
 WHERE 1 = 0

-- name: payment_mark_adjustment
-- Penanda persetujuan atasan pada adjustment — padanan baris `.FlagAtasan` dan
-- `.StsPenerimaKlaim` yang ditulis SIMPAN ke dalam case (`SaveApprovalAkseptasiPaymentLeader_Act`
-- :1639-1689). Kolomnya dibuat migrasi 0017; sebelum DBA menjalankannya pernyataan ini tidak
-- dijalankan (lihat payment_marks_probe), dan SIMPAN tetap berjalan tanpa penanda.
--
-- Pega hanya menandai adjustment PERTAMA yang nomornya cocok (langkahnya melompat keluar
-- setelah cocok). Di sini seluruh adjustment bernomor sama pada klaim itu — nomor akseptasi
-- satu adjustment, sehingga keduanya sama selama nomornya tidak ganda.
--
-- Bind: :1 FLAGATASAN · :2 STSPENERIMAKLAIM · :3 CLAIMID · :4 nomor akseptasi
UPDATE POOLDATA.T_CLAIM_ADJUSTMENT
   SET FLAGATASAN = :1,
       STSPENERIMAKLAIM = :2
 WHERE CLAIMID = :3
   AND TRIM(NOAKSEPTASI) = :4
