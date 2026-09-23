-- Kueri Inbox Laporan Klaim.
--
-- ============================================================================
-- TIGA TABEL, DAN PERAN MASING-MASING
-- ============================================================================
--
--   POOLDATA.T_CLAIMLIST_ADMIN       kumpulan BARIS daftar    — hanya dibaca
--   DATAPEGA.PC_ASM_FW_GCNMFW_WORK   empat kolom penentu tab  — hanya dibaca
--   POOLDATA.CPNC_LAPORAN_KLAIM      berkas terbitan sendiri  — dibaca DAN ditulis
--
-- Work Owner menetapkan 2026-09-23 daftar ditarik dari T_CLAIMLIST_ADMIN. Tabel itu
-- menentukan BARIS MANA yang tampil; empat kolom yang menentukan TAB tidak pernah terisi
-- di sana dan diambil dari tabel kerja Pega — alasannya di catatan claim_report_source.
--
-- Penulisan tidak berubah dan tetap hanya ke CPNC_LAPORAN_KLAIM (Work Owner, 2026-09-19).
-- Selama masa paralel, tepat satu sistem yang menulis sebuah tabel (ADR-0004, P-1); dua
-- tabel pertama hanya dibaca, sehingga pembagian itu tetap utuh.
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
-- Sumber daftar, dinormalkan ke bentuk kolom modul ini.
--
-- Ia BUKAN kueri utuh — ia awalan WITH yang disambung salah satu badan di bawahnya.
-- Penyambungannya dilakukan Go atas teks dari berkas ini sendiri, tidak pernah atas
-- nilai dari pengguna, sehingga aturan 2 di atas tetap utuh.
--
-- # Tabel mana yang menentukan BARIS, dan kenapa
--
-- Barisnya berasal dari DATAPEGA.PC_ASM_FW_GCNMFW_WORK — tabel yang sama dengan yang
-- dibaca layar lama.
--
-- POOLDATA.T_CLAIMLIST_ADMIN sempat dijadikan sumber baris (Work Owner, 2026-09-23), lalu
-- dikembalikan setelah dibandingkan langsung dengan angka layar lama pada 2026-09-23:
--
--   layar Pega, saringan Bisnis = NONMBU      ALL 671 · Outstanding 340 · Not Registered 123 · Not Transferred 42
--   tabel kerja Pega, saringan yang sama      ALL 671 · Outstanding 340 · Not Registered 123 · Not Transferred 42
--   T_CLAIMLIST_ADMIN                         142 baris Receive Document, dan `kodecabang_1` NULL pada SELURUHNYA
--
-- Tabel admin adalah daftar pekerjaan OUTSTANDING (`OS_CATEGORY` = 'OS PELAPORAN KLAIM'),
-- bukan daftar laporan yang utuh: 142 baris berbanding 2.800, tanpa kode cabang, dan tanpa
-- kolom penentu tab. Dipakai sebagai sumber baris, layar kehilangan 95% isinya dan seluruh
-- batas cabang.
--
-- Ia TETAP dibaca, tetapi untuk empat hal yang memang hanya ada di sana.
--
-- # Empat kolom yang datang dari tabel admin
--
--   sts_aktif        '0' berarti klaim sudah tidak aktif dan TIDAK ditampilkan lagi
--   aging            kolom "Aging" pada grid, dibedakan dari "Total Aging"
--   kurir            nama kurir pada form
--   notregistnote_1  keterangan belum registrasi
--
-- Baris yang tidak ada di tabel admin ikut tampil: `sts_aktif` NULL berarti penandanya
-- tidak ditetapkan, bukan tidak aktif. Menyembunyikannya akan menghapus 2.658 dari 2.800
-- baris sekaligus.
--
-- # Empat kolom turunan
--
--   position   disalin dari CASE WHEN pada RDB List/ViewAllCase-SQL.xml
--   accepted   EXISTS noakseptasi, dari ViewTableBrowseRCVAcc
--   rejected   statuswork klaim, dari ViewTableBrowseRCVReject
--   origin     diturunkan dari AWALAN NOMOR, bukan dari tabel asal
--
-- `origin` diturunkan dari nomor: berawalan `RCVN.` berarti terbitan aplikasi ini (lihat
-- ReportNumberPrefix di number.go), sisanya warisan Pega. Awalan itu sengaja dipilih pada
-- `D-71` justru supaya asal sebuah berkas terbaca dari nomornya tanpa tabel pemetaan.
--
-- # Posisi punya EMPAT keadaan, bukan tiga
--
-- Pemetaan sebelumnya menaruh seluruh sisanya di `ELSE` sebagai "Not Transferred", dan itu
-- SALAH — terbukti dari perbandingan langsung 2026-09-23 pada saringan NONMBU:
--
--   pnccaseid      statuslock_1    jumlah   tab
--   NULL           NULL             42      Not Transferred
--   NULL           terisi          123      Not Registered
--   terisi         terisi          340      Outstanding
--   terisi         NULL            166      TIDAK masuk tab mana pun
--
-- Keempat angka pertama sama persis dengan layar lama. Yang keempat — 166 baris ber-nomor
-- klaim tetapi belum terkunci — di layar lama tidak muncul di ketiga tab itu, dan hanya
-- ikut terhitung pada "All data" (42 + 123 + 340 + 166 = 671).
--
-- Karena itu posisinya NULL, bukan dipaksakan ke salah satu tab. `ELSE 'Not Transferred'`
-- yang lama membuat tab itu menyebut 208 di tempat layar lama menyebut 42.
--
-- # Penyaring yang DIPERTAHANKAN
--
-- `pxobjclass` disaring persis seperti kueri lama: tabel ini memuat seluruh case Pega,
-- bukan hanya Receive Document.
WITH source AS (
    SELECT w.pyid                AS report_id,
           w.pnccaseid           AS claim_number,
           w.statuslock_1        AS assignment_ref,
           w.policyno            AS policy_number,
           w.qqname              AS insured_name,
           -- Nama pembuat berkas, sama seperti `Sender := OperatorID.pyUserName` pada
           -- Activity/CreateNewCaseRCV-Act.xml. Hanya ada di tabel admin.
           t.pxcreateopname      AS reporter_name,
           w.businessname        AS business_name,
           w.bookno_1            AS reference_number,
           w.dateofloss_1        AS date_of_loss,
           w.pxcreatedatetime    AS created_at,
           w.pxcreateoperator    AS created_by,
           w.kodecabang_1        AS branch_code,
           w.dateforaging_1      AS aging_at,
           -- Keduanya tidak dipakai dulu (Work Owner, 2026-09-23).
           CAST(NULL AS VARCHAR(1000)) AS reason,
           CAST(NULL AS VARCHAR(1000)) AS email_subject,
           w.pystatuswork        AS work_status,
           w.grouppanel_1        AS group_panel,
           b.businessgroupid     AS business_group,
           CASE
               WHEN w.pyid LIKE 'RCVN.%' THEN 'claimpnc'
               ELSE 'pega'
           END                   AS origin,
           CAST(NULL AS DATE)         AS received_date,
           CAST(NULL AS VARCHAR(200)) AS reporter_email,
           CAST(NULL AS VARCHAR(64))  AS reporter_phone,
           t.kurir                    AS courier_name,
           CAST(NULL AS NUMBER)       AS estimate_value,
           CAST(NULL AS VARCHAR(500)) AS loss_location,
           CAST(NULL AS VARCHAR(4000)) AS chronology,
           CAST(NULL AS VARCHAR(4000)) AS damage_detail,
           t.notregistnote_1          AS not_registered_note,
           t.aging                    AS aging_value,
           CAST(NULL AS NUMBER)       AS document_count,
           CASE
               WHEN w.pnccaseid IS NOT NULL AND w.statuslock_1 IS NOT NULL THEN 'Outstanding'
               WHEN w.pnccaseid IS NULL     AND w.statuslock_1 IS NOT NULL THEN 'Not Registered'
               WHEN w.pnccaseid IS NULL     AND w.statuslock_1 IS NULL     THEN 'Not Transferred'
               ELSE NULL
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
      LEFT JOIN POOLDATA.BUSINESS b
             ON b.id = w.businesscode_1
      LEFT JOIN POOLDATA.T_CLAIMLIST_ADMIN t
             ON t.pyid = w.pyid
            AND t.pxobjclass = w.pxobjclass
     WHERE w.pxobjclass = 'ASM-FW-GCNMFW-Work-ReceiveDocument'
       -- '0' berarti klaim sudah tidak aktif dan tidak ditampilkan lagi (Work Owner,
       -- 2026-09-23). Yang dikecualikan HANYA yang bernilai '0' secara tegas: baris yang
       -- tidak ada di tabel admin ber-NULL, dan menyembunyikannya akan menghapus hampir
       -- seluruh daftar.
       AND (t.sts_aktif IS NULL OR TRIM(t.sts_aktif) <> '0')
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
       s.aging_value,
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
--    :7..:27  sama persis dengan badan daftar
--    :28,:29  identitas pemanggil     penjaga + pembanding pembuat berkas
--    :30,:31  status percakapan       penjaga + pembanding (EXISTS)
--    :32,:33  pengirim yang DICARI    penjaga + pembanding (EXISTS)
--    :34,:35  pengirim yang DIHINDARI penjaga + pembanding (EXISTS)
--    :1..:6 keenam bind yang sama, untuk mengambil pesan terakhirnya
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
       s.aging_value,
       (SELECT k.message
          FROM POOLDATA.M_KOMUNIKASI_PNC k
         WHERE k.caseid = s.report_id
           AND (:1 IS NULL OR k.komunikasistatus = :2)
           AND (:3 IS NULL OR k.sender = :4)
           AND (:5 IS NULL OR k.sender <> :6)
         ORDER BY k.createddate DESC
         FETCH NEXT 1 ROWS ONLY) AS last_message
  FROM source s
 WHERE (:7 IS NULL OR s.branch_code = :8)
   AND (:9 IS NULL OR s.branch_code IN (SELECT br.id FROM POOLDATA.BRANCH br WHERE br.basterritory = :10))
   AND (:11 IS NULL OR s.report_id = :12)
   AND (:13 IS NULL OR s.group_panel IN (:14, :15, :16, :17))
   AND (:18 IS NULL OR s.business_group IN (:19, :20, :21, :22))
   AND (:23 IS NULL OR s.business_group NOT IN (:24, :25, :26, :27))
   AND (:28 IS NULL OR s.created_by = :29)
   AND EXISTS (SELECT 1
                 FROM POOLDATA.M_KOMUNIKASI_PNC k
                WHERE k.caseid = s.report_id
                  AND (:30 IS NULL OR k.komunikasistatus = :31)
                  AND (:32 IS NULL OR k.sender = :33)
                  AND (:34 IS NULL OR k.sender <> :35))
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
--    :10..:30  penyaring yang sama dengan daftar, TANPA penyaring kategori
--    :1..:9 identitas pemanggil, tiga kali tiga, untuk ketiga pencacah komunikasi.
--             Ia NULL bila pemanggil tidak dikenali, dan ketiga pencacahnya menjadi nol —
--             bukan menghitung percakapan milik semua orang.
SELECT COUNT(1) AS total,
       SUM(CASE WHEN s.position = 'Not Transferred' THEN 1 ELSE 0 END) AS not_transferred,
       SUM(CASE WHEN s.position = 'Not Registered'  THEN 1 ELSE 0 END) AS unregistered,
       SUM(CASE WHEN s.position = 'Outstanding'     THEN 1 ELSE 0 END) AS outstanding,
       SUM(CASE WHEN s.accepted = '1'               THEN 1 ELSE 0 END) AS accepted,
       SUM(CASE
               WHEN :1 IS NOT NULL
                AND s.created_by = :2
                AND EXISTS (SELECT 1
                              FROM POOLDATA.M_KOMUNIKASI_PNC k
                             WHERE k.caseid = s.report_id
                               AND k.komunikasistatus = '0'
                               AND k.sender <> :3)
               THEN 1 ELSE 0
           END) AS message_unanswered,
       SUM(CASE
               WHEN :4 IS NOT NULL
                AND s.created_by = :5
                AND EXISTS (SELECT 1
                              FROM POOLDATA.M_KOMUNIKASI_PNC k
                             WHERE k.caseid = s.report_id
                               AND k.komunikasistatus = '0'
                               AND k.sender = :6)
               THEN 1 ELSE 0
           END) AS message_waiting,
       SUM(CASE
               WHEN :7 IS NOT NULL
                AND s.created_by = :8
                AND EXISTS (SELECT 1
                              FROM POOLDATA.M_KOMUNIKASI_PNC k
                             WHERE k.caseid = s.report_id
                               AND k.komunikasistatus = '1'
                               AND k.sender = :9)
               THEN 1 ELSE 0
           END) AS message_replied
  FROM source s
 WHERE (:10 IS NULL OR s.branch_code = :11)
   AND (:12 IS NULL OR s.branch_code IN (SELECT br.id FROM POOLDATA.BRANCH br WHERE br.basterritory = :13))
   AND (:14 IS NULL OR s.report_id = :15)
   AND (:16 IS NULL OR s.group_panel IN (:17, :18, :19, :20))
   AND (:21 IS NULL OR s.business_group IN (:22, :23, :24, :25))
   AND (:26 IS NULL OR s.business_group NOT IN (:27, :28, :29, :30))
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
       s.aging_value,
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
--
-- Sumber daftar. Namanya tetap "legacy" karena isinya memang berkas warisan; yang berubah
-- hanyalah tabelnya — dari DATAPEGA.PC_ASM_FW_GCNMFW_WORK menjadi tabel ini (Work Owner,
-- 2026-09-23).
SELECT t.pyid
  FROM POOLDATA.T_CLAIMLIST_ADMIN t
 WHERE 1 = 0

-- name: claim_report_get_own_body
--
-- Satu berkas TERBITAN APLIKASI INI, dibaca langsung dari POOLDATA.CPNC_LAPORAN_KLAIM.
--
-- # Kenapa jalur tersendiri, bukan lewat CTE source
--
-- Sejak daftar ditarik dari T_CLAIMLIST_ADMIN, berkas yang baru dibuat **belum ada di
-- sana** sampai proses pengisinya berjalan. Tanpa jalur ini, menekan "Buat Baru" akan
-- menerbitkan berkas lalu membuka form yang menjawab "laporan tidak ditemukan" — tombol
-- yang tampak rusak, persis kelas kegagalan yang sudah dua kali menimpa layar ini.
--
-- # Kenapa dipilih menurut AWALAN NOMOR
--
-- Nomor berawalan `RCVN.` hanya diterbitkan aplikasi ini (`D-71`, ReportNumberPrefix).
-- Pemilihannya karena itu pasti, tidak menuntut pembacaan dua tabel, dan tidak dapat
-- salah sasaran. Lihat Repo.Get.
--
-- # Kenapa isian formnya lengkap
--
-- Berkas terbitan aplikasi ini adalah SATU-SATUNYA yang dapat disunting (ADR-0004, P-1),
-- dan seluruh kesepuluh isiannya hidup di tabel ini — bukan di T_CLAIMLIST_ADMIN, yang
-- hanya membawa dua di antaranya.
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
       (SELECT br.branchname FROM POOLDATA.BRANCH br WHERE br.id = r.kode_cabang) AS branch_name,
       r.tgl_aging           AS aging_at,
       r.alasan              AS reason,
       r.subjek_email        AS email_subject,
       CASE
           WHEN r.no_klaim IS NOT NULL AND r.sts_diserahkan = '1' THEN 'Outstanding'
           WHEN r.sts_diserahkan = '1' AND r.no_klaim IS NULL     THEN 'Not Registered'
           ELSE 'Not Transferred'
       END                   AS position,
       'claimpnc'            AS origin,
       CAST(NULL AS NUMBER)  AS aging_value,
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
       CAST(NULL AS VARCHAR(4000)) AS last_message
  FROM POOLDATA.CPNC_LAPORAN_KLAIM r
 WHERE r.no_laporan = :1
   AND r.dihapus_pada IS NULL
