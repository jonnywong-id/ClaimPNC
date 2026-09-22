-- Kueri Inbox Laporan Klaim.
--
-- ============================================================================
-- DUA TABEL, DAN KENAPA KEDUANYA DIBACA
-- ============================================================================
--
--   DATAPEGA.PC_ASM_FW_GCNMFW_WORK   berkas laporan warisan  — HANYA DIBACA
--   POOLDATA.CPNC_LAPORAN_KLAIM      berkas laporan baru     — ditulis aplikasi ini
--
-- Work Owner menetapkan 2026-09-19: penulisan tidak lagi masuk ke tabel Pega. Selama
-- masa paralel, tepat satu sistem yang menulis sebuah tabel (ADR-0004, P-1) — dan
-- pembagian di atas memenuhinya tanpa satu baris pun dimiliki dua penulis.
--
-- Daftar yang dilihat petugas adalah GABUNGAN keduanya. Asal setiap baris dibawa apa
-- adanya di kolom ORIGIN supaya rekonsiliasi harian masa paralel dapat menjawab
-- "baris ini ditulis siapa" tanpa menebak.
--
-- ============================================================================
-- EMPAT ATURAN YANG MENGIKAT SELURUH BERKAS INI
-- ============================================================================
--
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--      Kueri lama menempelkan penyaringnya lewat {ASIS:tempQuery.NoKTP} dan tiga
--      saudaranya — 538 kemunculan pola itu di seluruh export, dan tiap satunya celah
--      injeksi (utang teknis §4.5). Tidak satu pun dibawa.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (D-20).
--   4. Tanpa pemanggilan stored procedure (D-02).
--
-- ROWNUM diganti OFFSET … FETCH NEXT … ROWS ONLY. Kueri lama membungkus hasilnya dua
-- lapis (`select b.* from (select a.*, ROWNUM rn from (…) a ) b`) lalu menempelkan
-- " WHERE rn >= x AND rn <= y" sebagai teks. Penggantinya didukung Oracle 12c+ maupun
-- PostgreSQL, dan sudah dipakai 35 rule lain di sistem lama — jadi bukan hal baru.
--
-- CATATAN PENANDA BIND. Berkas ini memakai gaya Oracle `:n`, sama seperti seluruh modul
-- lain di aplikasi ini. Ia BELUM portabel ke PostgreSQL yang memakai `$n`; itu utang
-- yang sudah ada sebelum modul ini dan berlaku untuk seluruh berkas .sql di sini.
--
-- ============================================================================
-- PEMETAAN KOLOM — alias Pega yang TIDAK dibawa (D-19)
-- ============================================================================
--
--   kolom asal                          alias Pega lama      nama di sini
--   ---------------------------------------------------------------------------
--   w.pyid                              RCVID                report_id
--   w.pnccaseid                         PNCCaseID            claim_number
--   w.statuslock_1                      StatusLock           assignment_ref
--   w.policyno                          PolicyNo             policy_number
--   w.qqname                            QQName               insured_name
--   w.businessname                      Kurir                business_name   (BUKAN kurir)
--   w.bookno_1                          Sender               reference_number
--   w.dateofloss_1                      TglKejadian          date_of_loss
--   w.pxcreatedatetime                  pxCreateDateTime     created_at
--   w.pxcreateoperator                  KodeCabang           created_by      (BUKAN kode cabang)
--   w.kodecabang_1                      StatusKomunikasi     branch_code     (BUKAN status)
--   br.branchname                       UserAdmin            branch_name     (BUKAN nama user)
--   w.dateforaging_1                    DateForAging         aging_at
--   w.keterangan_1                      SIM                  reason          (BUKAN nomor SIM)
--   w.subjectemail_1                    SubjectEmail         email_subject
--   k.message                           EmailPengirim        last_message

-- name: claim_report_source
--
-- Sumber gabungan kedua tabel, dinormalkan ke satu bentuk kolom.
--
-- Ia BUKAN kueri utuh — ia awalan WITH yang disambung salah satu badan di bawahnya.
-- Penyambungannya dilakukan Go atas teks dari berkas ini sendiri, tidak pernah atas
-- nilai dari pengguna, sehingga aturan 2 di atas tetap utuh.
--
-- Empat kolom turunan dihitung di sini supaya kesembilan tab menyaring atas dasar yang
-- sama persis, bukan atas sembilan tafsiran yang dapat menyimpang satu sama lain:
--
--   position   disalin dari CASE WHEN pada RDB List/ViewAllCase-SQL.xml
--   accepted   EXISTS noakseptasi, dari ViewTableBrowseRCVAcc
--   rejected   statuswork klaim, dari ViewTableBrowseRCVReject
--   origin     penanda tabel asal; tidak ada di sistem lama karena tabelnya satu
WITH source AS (
    SELECT w.pyid                AS report_id,
           w.pnccaseid           AS claim_number,
           w.statuslock_1        AS assignment_ref,
           w.policyno            AS policy_number,
           w.qqname              AS insured_name,
           -- Nama pelapor TIDAK dibaca dari tabel warisan: tidak satu pun dari kesembilan
           -- kueri lama menyentuh kolomnya, sehingga namanya tidak diketahui (R-08).
           -- Menebak nama kolom menghasilkan kueri yang gagal saat pertama dijalankan di
           -- produksi — jauh lebih mahal daripada satu kolom yang kosong.
           CAST(NULL AS VARCHAR(255)) AS reporter_name,
           w.businessname        AS business_name,
           w.bookno_1            AS reference_number,
           w.dateofloss_1        AS date_of_loss,
           w.pxcreatedatetime    AS created_at,
           w.pxcreateoperator    AS created_by,
           w.kodecabang_1        AS branch_code,
           w.dateforaging_1      AS aging_at,
           w.keterangan_1        AS reason,
           w.subjectemail_1      AS email_subject,
           w.pystatuswork        AS work_status,
           w.grouppanel_1        AS group_panel,
           b.businessgroupid     AS business_group,
           'pega'                AS origin,
           -- Kesepuluh isian form TIDAK dapat dibaca dari tabel warisan: tidak satu pun
           -- dari kesembilan kueri lama menyentuh kolomnya, sehingga nama kolomnya di
           -- tabel Pega tidak diketahui (R-08). Menebaknya menghasilkan kueri yang gagal
           -- saat pertama dijalankan di produksi.
           --
           -- Itu tidak menghalangi apa pun: berkas warisan memang tidak dapat disunting
           -- dari sini — penulisnya Pega selama masa paralel (ADR-0004, P-1) — sehingga
           -- form membukanya dalam modus baca saja.
           CAST(NULL AS DATE)         AS received_date,
           CAST(NULL AS VARCHAR(200)) AS reporter_email,
           CAST(NULL AS VARCHAR(64))  AS reporter_phone,
           CAST(NULL AS VARCHAR(255)) AS courier_name,
           CAST(NULL AS NUMBER)       AS estimate_value,
           CAST(NULL AS VARCHAR(500)) AS loss_location,
           CAST(NULL AS VARCHAR(4000)) AS chronology,
           CAST(NULL AS VARCHAR(4000)) AS damage_detail,
           CAST(NULL AS VARCHAR(1000)) AS not_registered_note,
           CAST(NULL AS NUMBER)       AS document_count,
           CASE
               WHEN w.pnccaseid IS NOT NULL AND w.statuslock_1 IS NOT NULL THEN 'Outstanding'
               WHEN w.statuslock_1 IS NOT NULL AND w.pnccaseid IS NULL     THEN 'Not Registered'
               ELSE 'Not Transferred'
           END                   AS position,
           CASE
               WHEN EXISTS (SELECT 1
                              FROM POOLDATA.T_CLAIM_PNC p,
                                   POOLDATA.T_CLAIM_ADJUSTMENT a
                             WHERE p.claimid = a.claimid
                               AND p.claimno = w.pnccaseid
                               AND a.noakseptasi IS NOT NULL)
               THEN '1' ELSE '0'
           END                   AS accepted,
           CASE
               WHEN EXISTS (SELECT 1
                              FROM POOLDATA.T_CLAIM_PNC p
                             WHERE p.claimno = w.pnccaseid
                               AND p.statuswork = 'Resolved-Rejected')
               THEN '1' ELSE '0'
           END                   AS rejected
      FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
      LEFT JOIN POOLDATA.BUSINESS b ON b.id = w.businesscode_1
     WHERE w.pxobjclass = 'ASM-FW-GCNMFW-Work-ReceiveDocument'

    UNION ALL

    SELECT r.no_laporan          AS report_id,
           r.no_klaim            AS claim_number,
           CAST(NULL AS VARCHAR(255)) AS assignment_ref,
           r.no_polis            AS policy_number,
           r.nama_tertanggung    AS insured_name,
           r.nama_pelapor        AS reporter_name,
           r.nama_bisnis         AS business_name,
           r.no_referensi        AS reference_number,
           r.tgl_kejadian        AS date_of_loss,
           r.dibuat_pada         AS created_at,
           r.dibuat_oleh         AS created_by,
           r.kode_cabang         AS branch_code,
           r.tgl_aging           AS aging_at,
           r.alasan              AS reason,
           r.subjek_email        AS email_subject,
           r.status_kerja        AS work_status,
           r.group_panel         AS group_panel,
           r.kode_group_bisnis   AS business_group,
           'claimpnc'            AS origin,
           r.tgl_terima_dokumen   AS received_date,
           r.email_pelapor        AS reporter_email,
           r.tlp_pelapor          AS reporter_phone,
           r.nama_kurir           AS courier_name,
           r.nilai_estimasi       AS estimate_value,
           r.lokasi_kejadian      AS loss_location,
           r.kronologis           AS chronology,
           r.rincian_kerusakan    AS damage_detail,
           r.ket_belum_registrasi AS not_registered_note,
           r.jumlah_dokumen       AS document_count,
           CASE
               WHEN r.no_klaim IS NOT NULL AND r.sts_diserahkan = '1' THEN 'Outstanding'
               WHEN r.sts_diserahkan = '1' AND r.no_klaim IS NULL     THEN 'Not Registered'
               ELSE 'Not Transferred'
           END                   AS position,
           CASE
               WHEN EXISTS (SELECT 1
                              FROM POOLDATA.T_CLAIM_PNC p,
                                   POOLDATA.T_CLAIM_ADJUSTMENT a
                             WHERE p.claimid = a.claimid
                               AND p.claimno = r.no_klaim
                               AND a.noakseptasi IS NOT NULL)
               THEN '1' ELSE '0'
           END                   AS accepted,
           CASE
               WHEN EXISTS (SELECT 1
                              FROM POOLDATA.T_CLAIM_PNC p
                             WHERE p.claimno = r.no_klaim
                               AND p.statuswork = 'Resolved-Rejected')
               THEN '1' ELSE '0'
           END                   AS rejected
      FROM POOLDATA.CPNC_LAPORAN_KLAIM r
     WHERE r.dihapus_pada IS NULL
)

-- name: claim_report_list_body
--
-- Satu halaman berkas laporan untuk tab NON-komunikasi.
--
-- Ketiga tab komunikasi punya badan sendiri (claim_report_message_body) karena bentuk
-- penyaringnya berbeda: ia menyaring keberadaan percakapan, bukan keadaan berkas.
-- Menyatukan keduanya menghasilkan satu kueri dengan belasan bind yang separuhnya
-- selalu NULL — lebih pendek ditulis, jauh lebih sulit dibaca dan dibuktikan benar.
--
-- URUTAN BIND (lihat listArguments di inboxlaporanklaim.go):
--    :1,:2    kode cabang           penjaga + pembanding
--    :3,:4    kode kanwil           penjaga + pembanding
--    :5,:6    kata kunci            penjaga + pembanding
--    :7       penjaga Group Panel
--    :8..:11  empat kode Group Panel yang diterima
--    :12      penjaga kelompok bisnis yang HARUS termasuk
--    :13..:16 empat kode kelompok bisnis
--    :17      penjaga kelompok bisnis yang DIKECUALIKAN
--    :18..:21 empat kode kelompok bisnis yang dikecualikan
--    :22      '1' bila berkas selesai/ditolak dikecualikan, '0' bila tidak
--    :23,:24  posisi berkas         penjaga + pembanding
--    :25      '1' bila hanya yang sudah berakseptasi
--    :26      '1' bila hanya yang ditolak
--    :27,:28  giliran               penjaga + banyaknya baris
SELECT s.report_id,
       s.claim_number,
       s.assignment_ref,
       s.policy_number,
       s.insured_name,
       s.reporter_name,
       s.business_name,
       s.reference_number,
       s.date_of_loss,
       s.created_at,
       s.created_by,
       s.branch_code,
       (SELECT br.branchname FROM POOLDATA.BRANCH br WHERE br.id = s.branch_code) AS branch_name,
       s.aging_at,
       s.reason,
       s.email_subject,
       s.position,
       s.origin,
       CAST(NULL AS VARCHAR(4000)) AS last_message
  FROM source s
 WHERE (:1 IS NULL OR s.branch_code = :2)
   AND (:3 IS NULL OR s.branch_code IN (SELECT br.id FROM POOLDATA.BRANCH br WHERE br.basterritory = :4))
   AND (:5 IS NULL OR s.report_id = :6)
   AND (:7 IS NULL OR s.group_panel IN (:8, :9, :10, :11))
   AND (:12 IS NULL OR s.business_group IN (:13, :14, :15, :16))
   AND (:17 IS NULL OR s.business_group NOT IN (:18, :19, :20, :21))
   AND (:22 = '0' OR s.work_status IS NULL OR s.work_status NOT IN ('Resolved-Completed', 'Resolved-Rejected'))
   AND (:23 IS NULL OR s.position = :24)
   AND (:25 IS NULL OR s.accepted = '1')
   AND (:26 IS NULL OR s.rejected = '1')
 ORDER BY s.aging_at DESC, s.report_id DESC
OFFSET :27 ROWS FETCH NEXT :28 ROWS ONLY

-- name: claim_report_count_body
--
-- Banyaknya baris yang cocok dengan penyaring yang SAMA, sebelum dipotong halaman.
-- Bind :1..:26 identik dengan badan daftar; giliran tidak dipakai di sini.
SELECT COUNT(1)
  FROM source s
 WHERE (:1 IS NULL OR s.branch_code = :2)
   AND (:3 IS NULL OR s.branch_code IN (SELECT br.id FROM POOLDATA.BRANCH br WHERE br.basterritory = :4))
   AND (:5 IS NULL OR s.report_id = :6)
   AND (:7 IS NULL OR s.group_panel IN (:8, :9, :10, :11))
   AND (:12 IS NULL OR s.business_group IN (:13, :14, :15, :16))
   AND (:17 IS NULL OR s.business_group NOT IN (:18, :19, :20, :21))
   AND (:22 = '0' OR s.work_status IS NULL OR s.work_status NOT IN ('Resolved-Completed', 'Resolved-Rejected'))
   AND (:23 IS NULL OR s.position = :24)
   AND (:25 IS NULL OR s.accepted = '1')
   AND (:26 IS NULL OR s.rejected = '1')

-- name: claim_report_message_body
--
-- Satu halaman untuk ketiga tab komunikasi.
--
-- Asal: RDB List/ViewRejectKomunikasiUser-SQL.xml. Tiga hal dibawa apa adanya —
-- berkasnya harus dibuat pemanggil sendiri, percakapannya harus ada, dan yang
-- ditampilkan adalah pesan TERAKHIR. Yang tidak dibawa: penyaringnya ditempelkan sebagai
-- teks lewat {ASIS:tempQuery.CaseID}, dan `fetch next 1 rows only` di dalam SELECT
-- dipertahankan karena ia memang sudah portabel.
--
-- URUTAN BIND:
--    :1..:21  sama persis dengan badan daftar
--    :22,:23  identitas pemanggil     penjaga + pembanding pembuat berkas
--    :24,:25  status percakapan       penjaga + pembanding (EXISTS)
--    :26,:27  pengirim yang DICARI    penjaga + pembanding (EXISTS)
--    :28,:29  pengirim yang DIHINDARI penjaga + pembanding (EXISTS)
--    :30..:35 keenam bind yang sama, untuk mengambil pesan terakhirnya
--    :36,:37  giliran
SELECT s.report_id,
       s.claim_number,
       s.assignment_ref,
       s.policy_number,
       s.insured_name,
       s.reporter_name,
       s.business_name,
       s.reference_number,
       s.date_of_loss,
       s.created_at,
       s.created_by,
       s.branch_code,
       (SELECT br.branchname FROM POOLDATA.BRANCH br WHERE br.id = s.branch_code) AS branch_name,
       s.aging_at,
       s.reason,
       s.email_subject,
       s.position,
       s.origin,
       (SELECT k.message
          FROM POOLDATA.M_KOMUNIKASI_PNC k
         WHERE k.caseid = s.report_id
           AND (:30 IS NULL OR k.komunikasistatus = :31)
           AND (:32 IS NULL OR k.sender = :33)
           AND (:34 IS NULL OR k.sender <> :35)
         ORDER BY k.createddate DESC
         FETCH NEXT 1 ROWS ONLY) AS last_message
  FROM source s
 WHERE (:1 IS NULL OR s.branch_code = :2)
   AND (:3 IS NULL OR s.branch_code IN (SELECT br.id FROM POOLDATA.BRANCH br WHERE br.basterritory = :4))
   AND (:5 IS NULL OR s.report_id = :6)
   AND (:7 IS NULL OR s.group_panel IN (:8, :9, :10, :11))
   AND (:12 IS NULL OR s.business_group IN (:13, :14, :15, :16))
   AND (:17 IS NULL OR s.business_group NOT IN (:18, :19, :20, :21))
   AND (:22 IS NULL OR s.created_by = :23)
   AND EXISTS (SELECT 1
                 FROM POOLDATA.M_KOMUNIKASI_PNC k
                WHERE k.caseid = s.report_id
                  AND (:24 IS NULL OR k.komunikasistatus = :25)
                  AND (:26 IS NULL OR k.sender = :27)
                  AND (:28 IS NULL OR k.sender <> :29))
 ORDER BY s.aging_at DESC, s.report_id DESC
OFFSET :36 ROWS FETCH NEXT :37 ROWS ONLY

-- name: claim_report_message_count_body
--
-- Banyaknya baris pada tab komunikasi. Bind :1..:29 identik dengan badan di atas.
SELECT COUNT(1)
  FROM source s
 WHERE (:1 IS NULL OR s.branch_code = :2)
   AND (:3 IS NULL OR s.branch_code IN (SELECT br.id FROM POOLDATA.BRANCH br WHERE br.basterritory = :4))
   AND (:5 IS NULL OR s.report_id = :6)
   AND (:7 IS NULL OR s.group_panel IN (:8, :9, :10, :11))
   AND (:12 IS NULL OR s.business_group IN (:13, :14, :15, :16))
   AND (:17 IS NULL OR s.business_group NOT IN (:18, :19, :20, :21))
   AND (:22 IS NULL OR s.created_by = :23)
   AND EXISTS (SELECT 1
                 FROM POOLDATA.M_KOMUNIKASI_PNC k
                WHERE k.caseid = s.report_id
                  AND (:24 IS NULL OR k.komunikasistatus = :25)
                  AND (:26 IS NULL OR k.sender = :27)
                  AND (:28 IS NULL OR k.sender <> :29))

-- name: claim_report_summary_body
--
-- Kedelapan pencacah di atas daftar, dalam SATU kueri.
--
-- Asal: RDB List/BrowseClaimRCV_Aksep-SQL.xml, yang juga mengembalikan kedelapannya
-- sekaligus. Bentuk itu dipertahankan dengan sengaja — menghitungnya lewat delapan kueri
-- berarti kedelapan angka berasal dari delapan saat yang berbeda, dan jumlahnya tidak
-- lagi cocok dengan totalnya.
--
-- Berkas yang sudah selesai atau ditolak DIKECUALIKAN, persis seperti kueri lama. Itulah
-- sebabnya tab "Data rejected" tidak punya lencana: kueri pencacahnya memang tidak
-- pernah menghitungnya.
--
-- URUTAN BIND:
--    :1..:21  penyaring yang sama dengan daftar, TANPA penyaring kategori
--    :22..:30 identitas pemanggil, tiga kali tiga, untuk ketiga pencacah komunikasi.
--             Ia NULL bila pemanggil tidak dikenali, dan ketiga pencacahnya menjadi nol —
--             bukan menghitung percakapan milik semua orang.
SELECT COUNT(1) AS total,
       SUM(CASE WHEN s.position = 'Not Transferred' THEN 1 ELSE 0 END) AS not_transferred,
       SUM(CASE WHEN s.position = 'Not Registered'  THEN 1 ELSE 0 END) AS unregistered,
       SUM(CASE WHEN s.position = 'Outstanding'     THEN 1 ELSE 0 END) AS outstanding,
       SUM(CASE WHEN s.accepted = '1'               THEN 1 ELSE 0 END) AS accepted,
       SUM(CASE
               WHEN :22 IS NOT NULL
                AND s.created_by = :23
                AND EXISTS (SELECT 1
                              FROM POOLDATA.M_KOMUNIKASI_PNC k
                             WHERE k.caseid = s.report_id
                               AND k.komunikasistatus = '0'
                               AND k.sender <> :24)
               THEN 1 ELSE 0
           END) AS message_unanswered,
       SUM(CASE
               WHEN :25 IS NOT NULL
                AND s.created_by = :26
                AND EXISTS (SELECT 1
                              FROM POOLDATA.M_KOMUNIKASI_PNC k
                             WHERE k.caseid = s.report_id
                               AND k.komunikasistatus = '0'
                               AND k.sender = :27)
               THEN 1 ELSE 0
           END) AS message_waiting,
       SUM(CASE
               WHEN :28 IS NOT NULL
                AND s.created_by = :29
                AND EXISTS (SELECT 1
                              FROM POOLDATA.M_KOMUNIKASI_PNC k
                             WHERE k.caseid = s.report_id
                               AND k.komunikasistatus = '1'
                               AND k.sender = :30)
               THEN 1 ELSE 0
           END) AS message_replied
  FROM source s
 WHERE (:1 IS NULL OR s.branch_code = :2)
   AND (:3 IS NULL OR s.branch_code IN (SELECT br.id FROM POOLDATA.BRANCH br WHERE br.basterritory = :4))
   AND (:5 IS NULL OR s.report_id = :6)
   AND (:7 IS NULL OR s.group_panel IN (:8, :9, :10, :11))
   AND (:12 IS NULL OR s.business_group IN (:13, :14, :15, :16))
   AND (:17 IS NULL OR s.business_group NOT IN (:18, :19, :20, :21))
   AND (s.work_status IS NULL OR s.work_status NOT IN ('Resolved-Completed', 'Resolved-Rejected'))

-- name: claim_report_get_body
--
-- Satu berkas menurut nomor registernya, dari tabel mana pun asalnya.
--
-- Ia memilih SEPULUH KOLOM LEBIH BANYAK daripada badan daftar, dan itu disengaja: hanya
-- form Input Receive Document yang membutuhkan isian berkas, dan dua di antaranya —
-- kronologis dan rincian kerusakan — berlebar 4.000 karakter. Menariknya pada setiap
-- halaman daftar berarti memindahkan ratusan kilobita yang tidak pernah digambar, pada
-- tabel berpuluh juta baris (D-10).
SELECT s.report_id,
       s.claim_number,
       s.assignment_ref,
       s.policy_number,
       s.insured_name,
       s.reporter_name,
       s.business_name,
       s.reference_number,
       s.date_of_loss,
       s.created_at,
       s.created_by,
       s.branch_code,
       (SELECT br.branchname FROM POOLDATA.BRANCH br WHERE br.id = s.branch_code) AS branch_name,
       s.aging_at,
       s.reason,
       s.email_subject,
       s.position,
       s.origin,
       s.received_date,
       s.reporter_email,
       s.reporter_phone,
       s.courier_name,
       s.estimate_value,
       s.loss_location,
       s.chronology,
       s.damage_detail,
       s.not_registered_note,
       s.document_count,
       CAST(NULL AS VARCHAR(4000)) AS last_message
  FROM source s
 WHERE s.report_id = :1

-- name: claim_report_region_list
--
-- Isi dropdown "Pilih Kanwil".
--
-- Asal: tempQuery.Remark pada SetListRCV_Act, yang menyaring cabang lewat kolom
-- BASTERRITORY. Daftar kanwilnya sendiri tidak punya tabel tersendiri di export —
-- ia disimpulkan dari nilai berbeda yang benar-benar dipakai cabang, dan itulah yang
-- dilakukan DISTINCT di sini.
--
-- Cabang tanpa kanwil dibuang: baris kosong pada dropdown tidak dapat dipilih dan hanya
-- menambah barang di layar.
SELECT DISTINCT br.basterritory
  FROM POOLDATA.BRANCH br
 WHERE br.basterritory IS NOT NULL
 ORDER BY br.basterritory

-- name: claim_report_next_sequence
--
-- Nomor urut berikutnya untuk berkas yang diterbitkan aplikasi ini.
--
-- Sequence-nya milik aplikasi ini sendiri dan dibuat migrasi 0003; ia TIDAK memakai
-- POOLDATA.CLAIM_NO_NONPEGA_SEQ yang `D-71` peruntukkan bagi nomor klaim. Dua deret
-- nomor untuk dua hal yang berbeda, supaya nomor klaim dan nomor laporan tidak saling
-- memakan urutan.
SELECT POOLDATA.CPNC_LAPORAN_KLAIM_SEQ.NEXTVAL FROM DUAL

-- name: claim_report_insert
--
-- Menyimpan berkas laporan baru.
--
-- Kolom yang tidak disebut di sini memang belum punya isi: tombol "Buat Baru" di sistem
-- lama membuat berkas KOSONG dan menyerahkan pengisiannya ke layar berikutnya
-- (`B-14`). Lihat Activity/CreateNewCaseRCV-Act.xml, yang hanya mengisi lima nilai.
--
-- DIHAPUS_PADA sengaja ada meski tidak pernah diisi modul ini: `ADR-0012` melarang
-- penghapusan fisik data bernilai bisnis, dan kolomnya harus sudah ada sejak baris
-- pertama supaya penghapusan kelak tidak menuntut perubahan skema di tengah masa
-- paralel (D-63).
INSERT INTO POOLDATA.CPNC_LAPORAN_KLAIM
    (NO_LAPORAN, NAMA_PELAPOR, KODE_CABANG, DIBUAT_OLEH, DIBUAT_PADA, TGL_AGING, STS_DISERAHKAN)
VALUES (:1, :2, :3, :4, :5, :6, '0')

-- name: claim_report_update
--
-- Menyimpan isian form Input Receive Document ke atas berkas yang sudah ada.
--
-- Asal: flow action `InputReceiveDocument` pada assignment tunggal
-- `Flow/InputReceiveDocument.xml`, yang di sistem lama menyimpan lewat
-- `Database/PROCINSERTDATARECIVEDKLAIM.prc`. Procedure itu TIDAK dipanggil (`D-02`);
-- yang dipakai adalah pernyataan langsung, dan tiga cacatnya tidak ikut dibawa:
-- `COMMIT` di dalam procedure, `ROLLBACK` yang terjadi sesudahnya, dan kontrak galat
-- berbasis teks `ErrMsg` (`D-68`).
--
-- # Yang TIDAK pernah ikut di-SET, dan kenapa
--
--   NO_LAPORAN       kunci baris; ia menyaring, tidak pernah berubah
--   NO_KLAIM         terbit saat registrasi (`B-2`), bukan dari form ini
--   KODE_CABANG      batas data; memindahkan berkas antarcabang bukan tindakan form ini
--   STS_DISERAHKAN   perpindahan tahap adalah tindakan tersendiri
--   DIBUAT_OLEH/PADA jejak pembuatan tidak pernah ditulis ulang
--   DIHAPUS_PADA     penghapusan dinyatakan lewat penanda (`ADR-0012`), bukan di sini
--
-- # Penyaring `DIHAPUS_PADA IS NULL`
--
-- Bukan kerapian: tanpa itu, berkas yang sudah ditandai terhapus tetap dapat disunting
-- lewat alamat yang masih dipegang peramban seseorang.
UPDATE POOLDATA.CPNC_LAPORAN_KLAIM
   SET TGL_TERIMA_DOKUMEN   = :1,
       TGL_KEJADIAN         = :2,
       NAMA_PELAPOR         = :3,
       EMAIL_PELAPOR        = :4,
       TLP_PELAPOR          = :5,
       NAMA_KURIR           = :6,
       NO_POLIS             = :7,
       NAMA_TERTANGGUNG     = :8,
       NAMA_BISNIS          = :9,
       NO_REFERENSI         = :10,
       NILAI_ESTIMASI       = :11,
       LOKASI_KEJADIAN      = :12,
       SUBJEK_EMAIL         = :13,
       KRONOLOGIS           = :14,
       RINCIAN_KERUSAKAN    = :15,
       ALASAN               = :16,
       KET_BELUM_REGISTRASI = :17,
       JUMLAH_DOKUMEN       = :18,
       DIUBAH_OLEH          = :19,
       DIUBAH_PADA          = :20
 WHERE NO_LAPORAN = :21
   AND DIHAPUS_PADA IS NULL

-- name: claim_report_check_table
--
-- Memastikan kedua tabel ada dan dapat dibaca akun aplikasi, tanpa mengambil satu baris
-- pun. Dipakai mode periksa, mengikuti pola modul-modul sebelumnya.
SELECT r.no_laporan
  FROM POOLDATA.CPNC_LAPORAN_KLAIM r
 WHERE 1 = 0

-- name: claim_report_check_legacy_table
SELECT w.pyid
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
 WHERE 1 = 0
