-- Kueri modul Inbox Compliance: antrean pemeriksaan kepatuhan.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- ============================================================================
-- TABEL MANA YANG BOLEH DITULIS, DAN MANA YANG TIDAK
-- ============================================================================
--
-- Tiga tabel warisan Pega DIBACA SAJA, dan tidak boleh ada satu pun pernyataan yang
-- menulisinya: selama masa paralel setiap tabel hanya boleh ditulis satu sistem, dan
-- ketiganya milik Pega (`P-1`).
--
--   DATAPEGA.PC_ASM_FW_GCNMFW_WORK   baca saja
--   DATAPEGA.PC_ASSIGN_WORKBASKET    baca saja
--   POOLDATA.T_CLAIM_PNC             baca saja
--
-- SATU tabel ditulis, dan hanya oleh aplikasi ini:
--
--   POOLDATA.T_CLAIM_COMPLIANCE_H    baca dan tulis
--
-- Tabel terakhir tidak dikenal Pega — ia tidak muncul di satu pun rule maupun prosedur di
-- export. Karena itu penulis tunggalnya adalah aplikasi baru, dan `P-1` terpenuhi tanpa
-- perlu perpindahan kepemilikan.
--
-- ============================================================================
-- DARI MANA BENTUK KUERI INI DIBACA
-- ============================================================================
--
-- Dari `Report Definition/InboxRegisterCompliance_RD-RD.xml`, bukan dari rule SQL — layar
-- ini memang tidak punya rule Connect-SQL sendiri. Yang dibaca dari RD itu:
--
--   Kelas dasar    ASM-FW-GCNMFW-Work-PNC
--   Join           INNER JOIN Assign-WorkBasket
--                  ON newAssignPage.pxRefObjectKey = .pzInsKey
--   Filter         newAssignPage.pxAssignedOperatorID = Param.Operator
--              AND .pyStatusWork != "Resolved-Completed"
--   Urutan         .pxCreateDateTime DESC, .pyID DESC
--   Batas          pyMaxRecords = 500
--
-- Nilai `Param.Operator` tidak tertulis di RD. Ia dibaca dari `Flow/Register_Flow.xml:3769`,
-- yakni shape assignment "Compliance" ber-`pyImplementation` WorkBasket dengan
-- `pyWorkBasket` bernilai `CompliancePNC`. Di sini ia datang sebagai bind `:1`, bukan
-- ditulis tetap di dalam teks SQL.
--
-- ============================================================================
-- PEMETAAN PROPERTI RD -> KOLOM SEBENARNYA -> ALIAS DI SINI
-- ============================================================================
--
-- Properti RD                        Kolom                          Alias
-- ---------------------------------- ------------------------------ ---------------------
-- .pyID                              A.PYID                         CASE_ID
-- .pzInsKey                          A.PZINSKEY                     REFERENCE
-- .Policy.PolicyNo                   A.POLICYNO                     POLICY_NUMBER
-- .Policy.QQName                     A.QQNAME                       INSURED_NAME
-- .Policy.Quotation.BusinessName     A.BUSINESSNAME                 BUSINESS_NAME
-- .Policy.Quotation.BranchName       A.BRANCHNAME                   BRANCH_NAME
-- .pyOrigUserID                      A.PYORIGUSERID                 ADMIN_NAME
-- .ClaimData.TanggalBuatCompliance   p.COMPLIANCE_CREATEDATE        COMPLIANCE_SENT_DATE
--
-- Dua baris terakhir menuntut penjelasan, karena keduanya BUKAN salinan harfiah.
--
-- ADMIN_NAME memakai `PYORIGUSERID`, bukan `PXCREATEOPERATOR` yang dipakai modul Inbox
-- Admin untuk kolom "Creator". Itu mengikuti RD, yang memberi label "Nama Admin" tepat pada
-- `.pyOrigUserID`. Keduanya dapat berbeda: baris yang dibuat job terjadwal atau layanan
-- REST masuk punya pembuat yang bukan orang.
--
-- COMPLIANCE_SENT_DATE dibaca dari TABEL DATAR `POOLDATA.T_CLAIM_PNC`, sedangkan RD
-- membacanya dari halaman kerja sebagai `.ClaimData.TanggalBuatCompliance`. Alasannya satu:
-- kolom `COMPLIANCE_CREATEDATE` TERBUKTI ada — ia ditulis
-- `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:513` dari kunci JSON `TanggalBuatCompliance`
-- (`:419`) — sedangkan nama kolomnya di tabel kerja Pega hanya dapat DITEBAK
-- (`TANGGALBUATCOMPLIANCE_1`, mengikuti pola `STATUSCLAIM_1` dan `GROUPPANEL_1`), dan DDL
-- tabel itu belum ada (`R-08`).
--
-- Menebak nama kolom berarti kueri yang baru gagal saat menyentuh Oracle nyata. Memakai
-- kolom yang terbukti ada berarti gagalnya tidak pernah terjadi. Bila DDL kelak menetapkan
-- nama kolom di tabel kerja, penggantinya satu baris.
--
-- Join ke `T_CLAIM_PNC` karena itu LEFT, bukan INNER: tabel datar diisi prosedur konversi
-- yang berjalan terpisah, sehingga klaim yang baru masuk antrean dapat belum punya
-- barisnya. INNER JOIN akan MENGHILANGKAN pekerjaan dari antrean hanya karena konversinya
-- tertinggal — kelas cacat yang jauh lebih berbahaya daripada kolom tanggal yang kosong.
--
-- ============================================================================
-- APA YANG BERUBAH DARI PERILAKU LAMA, DAN KENAPA
-- ============================================================================
--
-- 1. PAGINASI DI BASIS DATA, bukan menarik semuanya lalu memotong di aplikasi.
--    Standar koding menetapkan paginasi selalu server-side
--    (`08-TECHNICAL-STRATEGY.md` §5). Modul Inbox Admin memotongnya di aplikasi atas
--    keputusan Work Owner yang berlaku KHUSUS untuk layar itu; tidak ada keputusan serupa
--    untuk layar ini.
--
-- 2. BATAS 500 BARIS TIDAK DIBAWA.
--    RD lama memasang `pyMaxRecords=500`, sehingga antrean yang lebih panjang TERPOTONG
--    tanpa pemberitahuan. Sistem baru memaginasinya, sehingga seluruh barisnya terjangkau.
--    Ini penambahan kemampuan yang sudah diperkirakan `ADR-0011`, dan wajib dinyatakan di
--    muka sebagai selisih terencana pada gerbang 1 — bukan ditemukan sebagai kejutan.
--
-- 3. PARAMETER BINDING untuk nama workbasket.
--    Modul Inbox Admin menuliskan `wb.PXASSIGNEDOPERATORID = 'RCLPUCL'` langsung di dalam
--    teks SQL. Di sini nilainya datang sebagai bind, sehingga satu-satunya tempat nama
--    antrean ditulis adalah konstanta Go yang menyebut baris flow asalnya.
--
-- 4. AGING TIDAK DIHITUNG DI SINI.
--    Sistem lama memanggil `POOLDATA.GETSELISIHJAM` lewat rule SQL terpisah, satu kali per
--    baris. Di sini kueri hanya membawa tanggalnya; jamnya dihitung di Go (`D-02`, `D-50`).
--    Lihat internal/inboxcompliance/aging.go.
--
-- ============================================================================
-- DUA PENYARING YANG PERNAH ADA DI SINI DAN SUDAH DIBUANG — JANGAN DIPASANG LAGI
-- ============================================================================
--
-- Keduanya saya tambahkan sendiri, tidak satu pun berasal dari Pega, dan keduanya hanya
-- dapat MEMPERSEMPIT hasil. Dibuang 2026-10-06 setelah RD-nya dibaca utuh.
--
--   wb.PXOBJCLASS = 'Assign-WorkBasket'
--       RD tidak memilikinya. Di Pega pembatasan kelas itu datang dari join class
--       `Assign-WorkBasket` (`pyJoinClassName`), BUKAN dari predikat kolom — dan tabel
--       `PC_ASSIGN_WORKBASKET` memang hanya menyimpan assignment workbasket. Bila nilai
--       `PXOBJCLASS` yang tersimpan berbeda sedikit saja — beda kapitalisasi, atau nama
--       subkelas — predikat ini mengosongkan seluruh daftar tanpa satu pun pesan galat.
--
--   A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'   (kesamaan persis)
--       RD memasang `pyIncludeAllDescendantclasses = true`, sehingga Pega IKUT menyertakan
--       SUBKELAS. Kesamaan persis membuang subkelas apa pun. Diganti `LIKE '…%'`, yang
--       itulah terjemahan harfiah "sertakan seluruh turunan".
--
-- Yang TETAP dipertahankan karena memang ada di RD: `pyStatusWork != 'Resolved-Completed'`
-- (filter B), `pxAssignedOperatorID = Param.Operator` (filter A), dan join INNER
-- `pxRefObjectKey = .pzInsKey`.
--
-- Nilai parameternya pun terverifikasi, bukan dikira-kira:
-- `Activity/GetInboxRegisterCompliance-Act.xml` menetapkan `Param.Operator =
-- "CompliancePNC"` dan `Param.pyReportClass = "ASM-FW-GCNMFW-Work-PNC"`.

-- name: list_compliance
-- Tab Compliance — Report Definition/InboxRegisterCompliance_RD-RD.xml
--
-- Bind: :1 nama workbasket · :2 offset · :3 jumlah baris
SELECT A.PYID                     AS CASE_ID,
       A.PZINSKEY                 AS REFERENCE,
       A.POLICYNO                 AS POLICY_NUMBER,
       A.QQNAME                   AS INSURED_NAME,
       A.BUSINESSNAME             AS BUSINESS_NAME,
       A.BRANCHNAME               AS BRANCH_NAME,
       A.PYORIGUSERID             AS ADMIN_NAME,
       -- Lini bisnis. Ia tidak digambar di satu kolom pun — ia menentukan tombol mana yang
       -- muncul pada form Compliance Checker (`IsTravel` = "005", `IsPA` = "002"). Lihat
       -- ActionsFor di checker.go.
       --
       -- Ia ikut di kueri DAFTAR pula meski daftar tidak punya tombol, karena kedua kueri
       -- ini dilayani SATU pemindai. Menambahkannya hanya di salah satu akan memecah
       -- pemindai itu.
       --
       -- Join-nya LEFT, sehingga nilainya boleh NULL. Ditangani di Go, dan hasilnya sama
       -- dengan Pega: perbandingan terhadap nilai kosong bernilai salah.
       p.GROUPPANEL               AS GROUP_PANEL,
       -- PIC Teknik klaim. Tidak digambar; ia TUJUAN perpindahan setelah Compliance
       -- memutuskan. Nama kolomnya terbukti dari dashboardclaim/pindahpic.sql, yang
       -- memindahkan PIC dengan `UPDATE … SET USERTEKNIS_1`.
       A.USERTEKNIS_1             AS TECHNICIAN_ID,
       p.COMPLIANCE_CREATEDATE    AS COMPLIANCE_SENT_DATE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET wb
               ON wb.PXREFOBJECTKEY = A.PZINSKEY
       LEFT JOIN POOLDATA.T_CLAIM_PNC p
              ON p.CLAIMID = A.PZINSKEY
 WHERE A.PXOBJCLASS LIKE 'ASM-FW-GCNMFW-Work-PNC%'
   AND wb.PXASSIGNEDOPERATORID = :1
   AND A.PYSTATUSWORK <> 'Resolved-Completed'
   -- Klaim yang tahap Compliance-nya SUDAH SELESAI tidak lagi ditampilkan.
   --
   -- Inilah yang membuat klaim benar-benar HILANG dari antrean setelah diputuskan —
   -- padanan Ticket `SendtoAnalysator` yang di Pega memindahkan assignment-nya.
   --
   -- Penyaringnya di SINI, bukan dengan menghapus baris Pega, karena tabel assignment
   -- Pega tidak kita tulis sama sekali (lihat 0014_penugasan.up.sql). Selama masa
   -- paralel, "masih di antrean" berarti DUA hal sekaligus: barisnya ada di tabel Pega,
   -- DAN belum ditandai selesai di tabel kita.
   AND NOT EXISTS (
           SELECT 1
             FROM POOLDATA.CPNC_PENUGASAN g
            WHERE g.NO_KLAIM = A.PZINSKEY
              AND g.TAHAP    = 'Compliance'
              AND g.STATUS   = 'SELESAI')
 ORDER BY A.PXCREATEDATETIME DESC, A.PYID DESC
 OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY

-- name: count_compliance
-- Jumlah seluruh baris tab Compliance, untuk penomoran halaman.
--
-- Predikatnya WAJIB sama persis dengan list_compliance. Bila keduanya berbeda, layar akan
-- menggambar halaman yang tidak pernah berisi apa pun — dan selisihnya tidak terlihat
-- sampai seseorang membuka halaman terakhir.
--
-- `T_CLAIM_PNC` sengaja TIDAK ikut di-join di sini: ia LEFT JOIN yang tidak menyaring apa
-- pun, sehingga menyertakannya hanya menambah beban tanpa mengubah hasil hitungan.
--
-- Bind: :1 nama workbasket
SELECT COUNT(*)
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET wb
               ON wb.PXREFOBJECTKEY = A.PZINSKEY
 WHERE A.PXOBJCLASS LIKE 'ASM-FW-GCNMFW-Work-PNC%'
   AND wb.PXASSIGNEDOPERATORID = :1
   AND A.PYSTATUSWORK <> 'Resolved-Completed'
   -- Klaim yang tahap Compliance-nya SUDAH SELESAI tidak lagi ditampilkan.
   --
   -- Inilah yang membuat klaim benar-benar HILANG dari antrean setelah diputuskan —
   -- padanan Ticket `SendtoAnalysator` yang di Pega memindahkan assignment-nya.
   --
   -- Penyaringnya di SINI, bukan dengan menghapus baris Pega, karena tabel assignment
   -- Pega tidak kita tulis sama sekali (lihat 0014_penugasan.up.sql). Selama masa
   -- paralel, "masih di antrean" berarti DUA hal sekaligus: barisnya ada di tabel Pega,
   -- DAN belum ditandai selesai di tabel kita.
   AND NOT EXISTS (
           SELECT 1
             FROM POOLDATA.CPNC_PENUGASAN g
            WHERE g.NO_KLAIM = A.PZINSKEY
              AND g.TAHAP    = 'Compliance'
              AND g.STATUS   = 'SELESAI')

-- ============================================================================
-- TAB POST AUDIT — SUMBERNYA BUKAN TABEL PEGA
-- ============================================================================
--
-- `Report Definition/InboxCompliance_RD-RD.xml` membacanya dari kelas
-- `ASM-FW-GCNMFW-Work-Compliance` dengan sub-report ke Work-PNC lewat `pxCoverInsKey`.
-- Keputusan Work Owner 2026-09-24 mengarahkannya ke **tabel datar**
-- `POOLDATA.T_CLAIM_COMPLIANCE_H`, dan DDL-nya diterima pada tanggal yang sama:
--
--   CASEID                VARCHAR2(100)
--   NO_KLAIM              VARCHAR2(1000)
--   NAMA_TERTANGGUNG      VARCHAR2(4000)
--   NO_POLIS              VARCHAR2(4000)
--   REMARKS               VARCHAR2(4000)
--   TGL_KIRIM_POST_AUDIT  DATE
--
-- Pemetaannya:
--
-- Kolom layar Pega             Properti RD          Kolom                  Alias
-- ---------------------------- -------------------- ---------------------- --------------------
-- Nomor Case                   .pyID                CASEID                 CASE_ID
-- No Klaim                     .pxCoverInsKey       NO_KLAIM               CLAIM_NUMBER
-- Nama Tertanggung             subr.Nama_Tertanggung NAMA_TERTANGGUNG      INSURED_NAME
-- No Polis                     subr.No_Polis        NO_POLIS               POLICY_NUMBER
-- Catatan                      .ComplianceRemarks   REMARKS                COMPLIANCE_REMARKS
-- Tanggal Kirim Audit Compl.   .TanggalKirimPostAudit TGL_KIRIM_POST_AUDIT POST_AUDIT_SENT_DATE
-- OutStanding                  .pyNote              -- dihitung di Go --
--
-- **KOREKSI atas versi pertama berkas ini.** Versi pertama memetakan `NO_KLAIM` ke kolom
-- "Case ID" dan menyembunyikan `CASEID` sebagai kunci teknis — kebalikan dari yang benar.
--
-- Dasarnya waktu itu preseden `POOLDATA.T_CLAIM_PNC`, yang memisahkan `CLAIMID` (kunci
-- teknis) dari `CLAIMNO` (nomor yang dibaca orang). Penalarannya masuk akal dan tetap
-- salah, karena tabel ini tidak mengikuti preseden itu.
--
-- Yang membetulkannya adalah layar Pega yang berjalan, bukan penalaran ulang:
--
--   Nomor Case : CPL-19
--   No Klaim   : ASM-FW-GCNMFW-WORK PNC-2114
--
-- Jadi `CASEID` berisi nomor kasus Work-Compliance (`CPL-nn`) dan `NO_KLAIM` justru berisi
-- kunci teknis Pega. Namanya menyesatkan — kolom bernama "No Klaim" tidak memuat nomor
-- klaim — dan `D-13` menetapkan tampilannya ditiru, bukan diperbaiki diam-diam.
--
-- KEDUANYA DITAMPILKAN. Tidak ada kolom yang disembunyikan di tab ini, berbeda dari tab
-- Compliance yang menyembunyikan `PZINSKEY`.
--
-- ============================================================================
-- DUA HAL YANG TIDAK DAPAT DIREPLIKASI DARI RD LAMA, DAN AKIBATNYA
-- ============================================================================
--
-- 1. **Penyaring `.pyStatusWork = "New"` TIDAK ADA di sini**, karena tabelnya tidak punya
--    kolom status sama sekali.
--
--    Pembacaan yang dipakai: tabel ini memang sudah berisi apa yang hendak ditampilkan —
--    barisnya lahir ketika Post Audit dikirim, dan akhiran `_H` beserta kolom
--    `TGL_KIRIM_POST_AUDIT` menguatkannya. Menambahkan penyaring buatan di atasnya berarti
--    mengarang aturan yang tidak ada sumbernya.
--
--    Akibat yang harus disadari: bila tabel ini ternyata menyimpan SELURUH riwayat dan
--    bukan hanya yang belum ditindaklanjuti, tab ini akan menampilkan lebih banyak baris
--    daripada Pega. Itu pertanyaan terbuka untuk Work Owner, bukan sesuatu yang dapat
--    dijawab dari DDL.
--
-- 2. **Kunci urut kedua `.pxCreateDateTime DESC` tidak dapat dipakai**, karena tabelnya
--    tidak punya kolom waktu buat. Kunci PERTAMA `.pyID DESC` tetap dipakai, dan itu yang
--    menentukan urutan yang terlihat.
--
-- ============================================================================
-- URUTANNYA TEKS, BUKAN ANGKA — DAN ITU TERBACA DARI LAYAR PEGA
-- ============================================================================
--
-- Layar Pega yang berjalan menampilkan barisnya dalam urutan:
--
--   CPL-3, CPL-2, CPL-19, CPL-17, CPL-16, CPL-15, CPL-1
--
-- Itu BUKAN urutan angka — `CPL-19` akan berada di atas `CPL-3` bila angkanya yang
-- diurutkan. Itu urutan TEKS menurun, dan cocok persis: `'CPL-3' > 'CPL-2' > 'CPL-19'`
-- karena perbandingan berhenti di karakter kelima.
--
-- Jadi `ORDER BY CASEID DESC` apa adanya sudah benar, dan justru MENGURUTKAN ANGKANYA yang
-- akan menyimpang dari Pega. Bentuk yang tampak "lebih rapi" di sini adalah bentuk yang
-- salah.
--
-- Tabelnya tidak punya primary key maupun constraint unik, sehingga baris kembar mungkin
-- ada. Urutannya karena itu diberi dua pemutus supaya paginasi tetap: tanpa itu, satu baris
-- dapat muncul di dua halaman sekaligus sementara baris lain hilang.

-- name: list_post_audit
-- Tab Post Audit — POOLDATA.T_CLAIM_COMPLIANCE_H
--
-- Bind: :1 offset · :2 jumlah baris
SELECT h.CASEID                AS CASE_ID,
       h.NO_KLAIM              AS CLAIM_NUMBER,
       h.NAMA_TERTANGGUNG      AS INSURED_NAME,
       h.NO_POLIS              AS POLICY_NUMBER,
       h.REMARKS               AS COMPLIANCE_REMARKS,
       h.TGL_KIRIM_POST_AUDIT  AS POST_AUDIT_SENT_DATE
  FROM POOLDATA.T_CLAIM_COMPLIANCE_H h
 ORDER BY h.CASEID DESC NULLS LAST,
          h.TGL_KIRIM_POST_AUDIT DESC NULLS LAST,
          h.NO_KLAIM DESC NULLS LAST
 OFFSET :1 ROWS FETCH NEXT :2 ROWS ONLY

-- name: count_post_audit
-- Jumlah seluruh baris tab Post Audit, untuk penomoran halaman.
--
-- Tanpa bind: tab ini tidak punya penyaring apa pun, dan itu bukan kelalaian — lihat
-- catatan "DUA HAL YANG TIDAK DAPAT DIREPLIKASI" di atas.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIM_COMPLIANCE_H

-- name: check_table_post_audit
-- Memeriksa tabel tab Post Audit ADA, terbaca, DAN punya keenam kolom yang dipakai.
--
-- Terpisah dari check_table karena tabelnya pun terpisah: tab Compliance membaca tabel
-- warisan Pega, tab Post Audit membaca tabel datar yang BARU. Kegagalan keduanya berarti
-- hal yang berbeda, sehingga pesannya pun harus berbeda.
--
-- # Kenapa kolomnya DISEBUT satu per satu, bukan COUNT(*)
--
-- Versi sebelumnya berbunyi `SELECT COUNT(*) … WHERE 1 = 0`, dan itu hanya membuktikan
-- tabelnya terjangkau. Ia akan melaporkan HIJAU atas tabel yang ada tetapi kolomnya
-- bernama lain — lalu layar gagal saat dipakai, dengan `ORA-00904` yang baru terlihat di
-- log.
--
-- Itu bukan kekhawatiran teoretis. Tabel ini dibuat DBA di luar repositori ini, bukan oleh
-- migrasi di sini, sehingga nama kolomnya memang dapat berbeda dari yang diandaikan kueri
-- — dan satu-satunya cara mengetahuinya lebih awal adalah menyebut keenamnya di sini.
--
-- `WHERE 1 = 0` tetap dipakai supaya tidak ada baris yang benar-benar dibaca: yang diuji
-- keberadaan kolom, bukan isinya. Oracle tetap memvalidasi seluruh nama kolom saat
-- mem-parse pernyataannya.
SELECT h.CASEID,
       h.NO_KLAIM,
       h.NAMA_TERTANGGUNG,
       h.NO_POLIS,
       h.REMARKS,
       h.TGL_KIRIM_POST_AUDIT
  FROM POOLDATA.T_CLAIM_COMPLIANCE_H h
 WHERE 1 = 0

-- name: check_post_audit_sequence
-- Memeriksa sequence penomoran Post Audit ADA dan dapat dipakai akun aplikasi.
--
-- # Kenapa ini diperiksa terpisah, dan kenapa ia tidak boleh digabung ke check_table
--
-- Karena ia memeriksa jalur TULIS, sedangkan check_table memeriksa jalur BACA, dan
-- kegagalan keduanya berakibat berbeda: tanpa hak baca seluruh layar kosong, sedangkan
-- tanpa sequence layar tetap utuh dan hanya tombol Kirim yang gagal. Menggabungkannya
-- membuat modul yang sebenarnya 90% berfungsi dilaporkan mati total.
--
-- # Kenapa ALL_SEQUENCES, bukan memanggil NEXTVAL
--
-- Karena `-periksa` berjanji tidak menulis apa pun, dan `NEXTVAL` MENGHABISKAN satu nomor
-- setiap kali dipanggil — sekalipun transaksinya di-rollback, sequence Oracle tidak ikut
-- mundur. Memeriksa dengan NEXTVAL berarti setiap kali aplikasi start, satu nomor Post
-- Audit hilang. Lubang penomoran itu persis yang NOCACHE pada migrasi 0011 hindari.
--
-- `ALL_SEQUENCES` hanya memuat sequence yang DAPAT DIAKSES akun saat ini, sehingga satu
-- kueri ini membuktikan dua hal sekaligus: objeknya ada, dan haknya diberikan. Keduanya
-- gagal dengan pesan Oracle yang sama menyesatkan ("sequence does not exist"), sehingga
-- membedakannya di sini tidak berguna — yang berguna adalah tahu sebelum tombol diklik.
--
-- Yang TIDAK dibuktikan kueri ini: hak INSERT pada tabelnya. Satu-satunya cara
-- membuktikannya adalah benar-benar menyisipkan baris, dan itu melanggar janji `-periksa`.
--
-- Bind: :1 pemilik sequence · :2 nama sequence
SELECT COUNT(*)
  FROM ALL_SEQUENCES
 WHERE SEQUENCE_OWNER = :1
   AND SEQUENCE_NAME = :2

-- name: check_table
-- Memeriksa tabel inti modul ini terbaca dari koneksi yang dipakai.
--
-- Ia dipanggil perintah `-periksa` saat aplikasi start, dan sengaja tidak menyentuh baris
-- mana pun: yang diperiksa adalah HAK BACA dan keberadaan tabelnya, bukan isinya.
--
-- Ketiga tabel diperiksa sekaligus lewat satu join, karena ketiganya memang dibutuhkan
-- list_compliance — memeriksa satu saja akan meloloskan keadaan yang tetap membuat layar
-- gagal.
SELECT COUNT(*)
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET wb
               ON wb.PXREFOBJECTKEY = A.PZINSKEY
       LEFT JOIN POOLDATA.T_CLAIM_PNC p
              ON p.CLAIMID = A.PZINSKEY
 WHERE 1 = 0

-- ============================================================================
-- PENGIRIMAN KE POST AUDIT — SATU-SATUNYA JALUR TULIS MODUL INI
-- ============================================================================
--
-- Yang ditulis HANYA `POOLDATA.T_CLAIM_COMPLIANCE_H`, tabel baru yang tidak dikenal Pega.
-- Ketiga tabel warisan tetap dibaca saja, sehingga `P-1` tidak dilanggar: untuk tabel ini
-- penulis tunggalnya adalah aplikasi baru.
--
-- Alurnya meniru Pega (keputusan Work Owner 2026-09-24): petugas meneruskan klaim yang
-- SEDANG menunggu di antrean Compliance. Karena itu kuerinya tiga, bukan satu — klaimnya
-- dicari lebih dulu di antrean, dan pengiriman atas klaim yang tidak ada di sana ditolak.

-- name: find_compliance_claim
-- Mencari satu klaim yang sedang menunggu di antrean Compliance.
--
-- Predikatnya WAJIB sama dengan list_compliance. Bila berbeda, akan ada klaim yang tampil
-- di layar tetapi ditolak saat dikirim — atau sebaliknya, klaim yang tidak tampil tetapi
-- dapat dikirim lewat permintaan langsung.
--
-- Aliasnya sama dengan list_compliance supaya satu pemindai melayani keduanya.
--
-- Bind: :1 nama workbasket · :2 kunci klaim (PZINSKEY)
SELECT A.PYID                     AS CASE_ID,
       A.PZINSKEY                 AS REFERENCE,
       A.POLICYNO                 AS POLICY_NUMBER,
       A.QQNAME                   AS INSURED_NAME,
       A.BUSINESSNAME             AS BUSINESS_NAME,
       A.BRANCHNAME               AS BRANCH_NAME,
       A.PYORIGUSERID             AS ADMIN_NAME,
       -- Lini bisnis. Ia tidak digambar di satu kolom pun — ia menentukan tombol mana yang
       -- muncul pada form Compliance Checker (`IsTravel` = "005", `IsPA` = "002"). Lihat
       -- ActionsFor di checker.go.
       --
       -- Ia ikut di kueri DAFTAR pula meski daftar tidak punya tombol, karena kedua kueri
       -- ini dilayani SATU pemindai. Menambahkannya hanya di salah satu akan memecah
       -- pemindai itu.
       --
       -- Join-nya LEFT, sehingga nilainya boleh NULL. Ditangani di Go, dan hasilnya sama
       -- dengan Pega: perbandingan terhadap nilai kosong bernilai salah.
       p.GROUPPANEL               AS GROUP_PANEL,
       -- PIC Teknik klaim. Tidak digambar; ia TUJUAN perpindahan setelah Compliance
       -- memutuskan. Nama kolomnya terbukti dari dashboardclaim/pindahpic.sql, yang
       -- memindahkan PIC dengan `UPDATE … SET USERTEKNIS_1`.
       A.USERTEKNIS_1             AS TECHNICIAN_ID,
       p.COMPLIANCE_CREATEDATE    AS COMPLIANCE_SENT_DATE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET wb
               ON wb.PXREFOBJECTKEY = A.PZINSKEY
       LEFT JOIN POOLDATA.T_CLAIM_PNC p
              ON p.CLAIMID = A.PZINSKEY
 WHERE A.PXOBJCLASS LIKE 'ASM-FW-GCNMFW-Work-PNC%'
   AND wb.PXASSIGNEDOPERATORID = :1
   AND A.PYSTATUSWORK <> 'Resolved-Completed'
   -- Klaim yang tahap Compliance-nya SUDAH SELESAI tidak lagi ditampilkan.
   --
   -- Inilah yang membuat klaim benar-benar HILANG dari antrean setelah diputuskan —
   -- padanan Ticket `SendtoAnalysator` yang di Pega memindahkan assignment-nya.
   --
   -- Penyaringnya di SINI, bukan dengan menghapus baris Pega, karena tabel assignment
   -- Pega tidak kita tulis sama sekali (lihat 0014_penugasan.up.sql). Selama masa
   -- paralel, "masih di antrean" berarti DUA hal sekaligus: barisnya ada di tabel Pega,
   -- DAN belum ditandai selesai di tabel kita.
   AND NOT EXISTS (
           SELECT 1
             FROM POOLDATA.CPNC_PENUGASAN g
            WHERE g.NO_KLAIM = A.PZINSKEY
              AND g.TAHAP    = 'Compliance'
              AND g.STATUS   = 'SELESAI')
   AND A.PZINSKEY = :2
 FETCH FIRST 1 ROW ONLY

-- name: post_audit_next_sequence
-- Menerbitkan nomor Post Audit berikutnya, LENGKAP — bukan hanya angkanya.
--
-- Sintaksnya ditetapkan Work Owner 2026-10-06, dan dipakai APA ADANYA:
--
--     'CPL' || '.' || TO_CHAR(SYSDATE,'RR') || '.'
--            || TO_CHAR(POOLDATA.CLAIM_COMPLIENCE_SEQ.NEXTVAL)
--
-- # Kenapa dirakit DI SINI, padahal §3.2 melarang pemformatan di SQL
--
-- Karena larangan itu mengenai pemformatan untuk DITAMPILKAN — tanggal dan angka yang
-- dikembalikan sebagai teks lalu diurutkan sebagai teks. Generator nomor adalah
-- pengecualian yang `09-DATABASE-STRATEGY.md` §3.1 sebut sendiri, dengan contoh yang
-- bentuknya persis sama untuk nomor klaim `PNCN.YY.xxxx` (`D-71`).
--
-- Merakitnya di Go justru lebih buruk di sini: tahunnya akan datang dari jam aplikasi
-- sementara nomor urutnya dari basis data, sehingga keduanya dapat tidak sepakat tentang
-- tahun pada satu nomor yang sama di sekitar pergantian tahun.
--
-- # Sakelar dialek
--
-- Kueri ini satu-satunya tempat di modul ini yang memuat sintaks khas Oracle, dan ia
-- memuat DUA sekaligus: `SEQ.NEXTVAL` (PostgreSQL: `nextval('…')`) dan
-- `TO_CHAR(SYSDATE,'RR')` (PostgreSQL: `to_char(current_date,'YY')`). Keduanya diisolasi
-- di satu kueri, sejalan dengan `D-22`.
--
-- # Ejaan COMPLIENCE
--
-- Sengaja apa adanya — ejaan Work Owner, dan sejalan dengan sistem lama yang access
-- group-nya pun bernama `PncComplience`.
SELECT 'CPL' || '.' || TO_CHAR(SYSDATE, 'RR') || '.'
       || TO_CHAR(POOLDATA.CLAIM_COMPLIENCE_SEQ.NEXTVAL) AS CASE_ID
  FROM DUAL

-- name: insert_post_audit
-- Menulis satu baris Post Audit.
--
-- Keenam kolomnya diisi seluruhnya — tidak ada yang dibiarkan mengambil nilai bawaan,
-- karena tabelnya memang tidak punya satu pun.
--
-- Bind: :1 CASEID · :2 NO_KLAIM · :3 NAMA_TERTANGGUNG · :4 NO_POLIS · :5 REMARKS
--       :6 TGL_KIRIM_POST_AUDIT
INSERT INTO POOLDATA.T_CLAIM_COMPLIANCE_H
       (CASEID, NO_KLAIM, NAMA_TERTANGGUNG, NO_POLIS, REMARKS, TGL_KIRIM_POST_AUDIT)
VALUES (:1, :2, :3, :4, :5, :6)

-- name: find_compliance_decision
-- Mengambil keputusan Compliance yang sudah tersimpan atas satu klaim.
--
-- Tabelnya MILIK aplikasi ini, bukan tabel warisan — lihat catatan pada SaveDecision di
-- inboxcompliance.go. Tidak ada baris berarti klaimnya belum pernah diputuskan, dan itu
-- keadaan normal, bukan galat.
--
-- Bind: :1 kunci klaim (PZINSKEY)
SELECT PILIHAN            AS CHOICE,
       NOTE               AS NOTE,
       CATATAN            AS REMARKS,
       DIPUTUSKAN_OLEH    AS DECIDED_BY,
       DIPUTUSKAN_PADA    AS DECIDED_AT,
       TGL_VALID          AS VALIDATED_AT,
       TGL_KIRIM_POST_AUDIT AS SENT_TO_POST_AUDIT_AT,
       -- Grid komentar, satu kolom JSON. Go yang mengurainya — bukan `JSON_TABLE` —
       -- supaya kueri ini tetap SQL biasa dan portabel apa adanya ke PostgreSQL (`D-20`).
       KOMENTAR_JSON      AS COMMENTS_JSON
  FROM POOLDATA.CPNC_KEPUTUSAN_COMPLIANCE
 WHERE NO_KLAIM = :1

-- name: upsert_compliance_decision
-- Menyimpan keputusan Compliance, menimpa keputusan sebelumnya atas klaim yang sama.
--
-- MERGE, bukan INSERT: satu klaim hanya punya SATU keputusan yang berlaku, karena
-- `.ClaimData.PilihanCompliance` adalah satu properti pada klaimnya — bukan daftar.
-- Petugas yang membuka form kedua kalinya dan mengubah pilihannya mengubah keputusan itu,
-- tidak menambah keputusan kedua.
--
-- Riwayat perubahannya TIDAK disimpan di sini. Di Pega ia ada di `InsertHistoryClaimPNC`,
-- tabel yang berbeda dan dimiliki Pega — dan sampai `S-5` ada, satu-satunya jejak
-- perubahan keputusan di aplikasi ini adalah baris log (`D-59`).
--
-- `MERGE` didukung Oracle 9i+ dan PostgreSQL 15+, sehingga ia tidak menambah pengecualian
-- dialek baru (`D-20`, `D-24` menargetkan PostgreSQL 17+).
--
-- Bind: :1 NO_KLAIM · :2 PILIHAN · :3 NOTE · :4 CATATAN · :5 DIPUTUSKAN_OLEH
--       :6 DIPUTUSKAN_PADA · :7 TGL_VALID · :8 TGL_KIRIM_POST_AUDIT
MERGE INTO POOLDATA.CPNC_KEPUTUSAN_COMPLIANCE t
     USING (SELECT :1 AS NO_KLAIM FROM DUAL) s
        ON (t.NO_KLAIM = s.NO_KLAIM)
      WHEN MATCHED THEN
           UPDATE SET t.PILIHAN              = :2,
                      t.NOTE                 = :3,
                      t.CATATAN              = :4,
                      t.DIPUTUSKAN_OLEH      = :5,
                      t.DIPUTUSKAN_PADA      = :6,
                      t.TGL_VALID            = :7,
                      t.TGL_KIRIM_POST_AUDIT = :8,
                      t.KOMENTAR_JSON        = :9
      WHEN NOT MATCHED THEN
           INSERT (NO_KLAIM, PILIHAN, NOTE, CATATAN,
                   DIPUTUSKAN_OLEH, DIPUTUSKAN_PADA, TGL_VALID, TGL_KIRIM_POST_AUDIT,
                   KOMENTAR_JSON)
           VALUES (s.NO_KLAIM, :2, :3, :4, :5, :6, :7, :8, :9)

-- name: check_table_decision
-- Membuktikan tabel keputusan Compliance ada DAN dapat dibaca akun aplikasi.
--
-- `FETCH FIRST 0 ROWS ONLY` memaksa Oracle mengurai dan memeriksa hak akses tanpa membaca
-- satu baris pun — sama polanya dengan check_table_post_audit.
SELECT 1
  FROM POOLDATA.CPNC_KEPUTUSAN_COMPLIANCE
 FETCH FIRST 0 ROWS ONLY

-- name: apply_decision_to_claim
-- Menulis akibat keputusan Compliance ke KLAIMNYA — padanan tiga langkah pertama
-- `Activity/SetComplianceResult`.
--
-- Bind: :1 STATUSCLAIM · :2 CPLVALID_DATE atau NULL · :3 POSTAUDIT_TF_ANALYSTDATE atau
--       NULL · :4 kunci klaim (CLAIMID)
--
-- ============================================================================
-- KENAPA HANYA TIGA KOLOM, PADAHAL PEGA MENULIS SEMBILAN HAL
-- ============================================================================
--
-- Karena hanya tiga di antaranya yang PUNYA KOLOM. Diperiksa langsung ke
-- `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc` — procedure yang meratakan klaim dari JSON ke
-- tabel, dan karenanya daftar kolom yang sahih:
--
--     CPLVALID_DATE              4 kemunculan   ✅
--     POSTAUDIT_TF_ANALYSTDATE   4 kemunculan   ✅
--     STATUSCLAIM                4 kemunculan   ✅
--     isComplianceTransfer       0              ❌ tidak punya kolom
--     UserBusinessPA             0              ❌ tidak punya kolom
--     ComplianceStatus           0              ❌ tidak punya kolom
--
-- Ketiga yang terakhir hidup HANYA di klipboard Pega, sama seperti `ComplianceRemark`.
-- Menuliskannya menuntut Tim Pega mengeksposnya ke kolom lebih dulu.
--
-- Satu jebakan yang nyaris menyesatkan: pencarian teks menemukan ketiganya di
-- `RDB List/ExportDataDetailKlaim-SQL.xml` — tetapi sebagai ALIAS, bukan kolom, dan
-- aliasnya menyesatkan sejauh `a.surplus1 AS "isComplianceTransfer"`. Keberadaan sebuah
-- nama di teks SQL tidak membuktikan kolomnya ada (`03-CURRENT-ARCHITECTURE.md` §4.2).
--
-- ============================================================================
-- KENAPA COALESCE, BUKAN TIGA KUERI TERPISAH
-- ============================================================================
--
-- Di Pega, langkah 2 (`CPLValidDate`) hanya berjalan pada pilihan Bayar/Valid, dan langkah
-- 15 (`PostAudtiTfAnalyst`) hanya pada Bayar/PostAudit. Langkah yang tidak berjalan
-- MENINGGALKAN nilai lamanya — ia tidak mengosongkannya.
--
-- `COALESCE(:2, CPLVALID_DATE)` meniru itu persis: bind NULL berarti "jangan sentuh".
-- Mengirim NULL apa adanya justru akan MENGHAPUS tanggal yang sudah ada — perbedaan yang
-- tidak terlihat sampai seseorang mengubah keputusan dari Valid menjadi Lain-Lain dan
-- tanggal validnya lenyap.
--
-- STATUSCLAIM tidak memakai COALESCE karena langkah 1 di Pega TANPA syarat: ia di-set pada
-- setiap keputusan, termasuk Fraud/Tolak.
UPDATE POOLDATA.T_CLAIM_PNC
   SET STATUSCLAIM              = :1,
       CPLVALID_DATE            = COALESCE(:2, CPLVALID_DATE),
       POSTAUDIT_TF_ANALYSTDATE = COALESCE(:3, POSTAUDIT_TF_ANALYSTDATE)
 WHERE CLAIMID = :4

-- name: insert_history_claim
-- Satu baris riwayat keputusan Compliance — padanan `Call InsertHistoryClaimPNC`,
-- langkah 16-18 `SetComplianceResult`.
--
-- Bind: :1 CASEID (PZINSKEY) · :2 CREATEDATETIME · :3 STATUSNOTE · :4 USERUPDATE
--
-- # Kenapa INSERT langsung, bukan memanggil procedure-nya
--
-- `D-02`: aplikasi tidak memanggil stored procedure; logikanya naik ke Go. Isi
-- `Database/PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc` memang hanya satu INSERT:
--
--     INSERT INTO LIST_HISTORY_CLAIM_PNC (CASEID, CREATEDATETIME, STATUSNOTE, USERUPDATE)
--     VALUES (CaseID, CURRENT_TIMESTAMP, StatusNote, UserUpdate);
--
-- # Satu perbedaan yang disengaja: waktunya dari Go, bukan CURRENT_TIMESTAMP
--
-- Procedure-nya memakai jam BASIS DATA. Di sini waktunya dikirim sebagai bind dari seam
-- Clock (`F-5`), dan nilainya SAMA PERSIS dengan `DIPUTUSKAN_PADA` pada baris keputusan.
--
-- Itu bukan kelalaian meniru melainkan perbaikan kecil yang disadari: dengan jam basis
-- data, baris riwayat dan baris keputusan dapat berselisih beberapa milidetik, dan selisih
-- itu membuat keduanya sulit dipasangkan saat menelusuri. Selisihnya tidak terlihat
-- pengguna, sehingga ia tidak menimbulkan selisih pada gerbang 1.
--
-- # Skema
--
-- Procedure-nya menyebut tabelnya TANPA skema, sehingga ia resolve ke skema pemilik
-- procedure — POOLDATA. Itu KESIMPULAN, bukan bacaan langsung; `check_table_history` di
-- bawah yang membuktikannya saat `-periksa`, jauh sebelum petugas menekan Simpan.
INSERT INTO POOLDATA.LIST_HISTORY_CLAIM_PNC
       (CASEID, CREATEDATETIME, STATUSNOTE, USERUPDATE)
VALUES (:1, :2, :3, :4)

-- name: check_table_history
-- Memeriksa tabel riwayat ADA, terbaca, DAN punya keempat kolom yang ditulis.
--
-- Kolomnya disebut satu per satu dengan alasan yang sama seperti check_table_post_audit:
-- `COUNT(*)` hanya membuktikan tabelnya terjangkau, dan akan melaporkan hijau atas tabel
-- yang kolomnya bernama lain.
--
-- Di sini pemeriksaan itu lebih penting lagi, karena SKEMA tabelnya pun kesimpulan —
-- procedure aslinya menyebutnya tanpa skema.
SELECT CASEID, CREATEDATETIME, STATUSNOTE, USERUPDATE
  FROM POOLDATA.LIST_HISTORY_CLAIM_PNC
 WHERE 1 = 0

-- name: insert_penugasan
-- Satu baris penugasan. Dipakai dua kali per perpindahan: menutup tahap lama, membuka
-- tahap baru.
--
-- Bind: :1 PENUGASAN_ID · :2 NO_KLAIM · :3 TAHAP · :4 JENIS · :5 DITUGASKAN_KE ·
--       :6 WORKBASKET · :7 STATUS · :8 DIBUAT_PADA
--
-- Keduanya dibungkus SATU TRANSAKSI di Go — lihat MoveAssignment. Itu bukan kehati-hatian
-- berlebihan: separuh perpindahan berarti klaim hilang dari antrean tanpa tiba di mana
-- pun, dan tidak ada galat yang akan memberitahukannya.
INSERT INTO POOLDATA.CPNC_PENUGASAN
       (PENUGASAN_ID, NO_KLAIM, TAHAP, JENIS, DITUGASKAN_KE, WORKBASKET,
        STATUS, DIBUAT_PADA)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8)

-- name: check_table_penugasan
-- Memeriksa tabel penugasan ADA, terbaca, DAN punya setiap kolom yang ditulis.
--
-- Kolomnya disebut satu per satu, sama alasannya dengan check_table_post_audit. Di sini
-- taruhannya lebih besar lagi: kueri DAFTAR ikut menyentuh tabel ini lewat `NOT EXISTS`,
-- sehingga tabel yang hilang tidak hanya menggagalkan perpindahan — ia mengosongkan
-- seluruh antrean Compliance.
SELECT PENUGASAN_ID, NO_KLAIM, TAHAP, JENIS, DITUGASKAN_KE, WORKBASKET,
       STATUS, DIBUAT_PADA, DIUBAH_PADA
  FROM POOLDATA.CPNC_PENUGASAN
 WHERE 1 = 0

-- name: find_survey_results
-- Hasil investigasi satu klaim — blok "Hasil Investigasi" form Compliance Checker.
--
-- Bind: :1 PNCCASEID (yakni PZINSKEY klaimnya)
--
-- Keempat kolom adalah yang benar-benar digambar grid `isPA_PNC` pada
-- `Section/ViewHasilSurvey-Section.xml` (sel 154–157), tidak lebih:
--
--   SURVEYDATE       "Tanggal Investigasi"
--   OBJECT_NAME      "Nama Peserta"      <- pada PA, objek pertanggungannya ORANG
--   LOCATION_OBJECT  "Lokasi Objek"
--   STS_SURVEY       "Status"
--
-- Nama kolomnya dibaca dari `Database/INSERT_SURVEYORLIST.prc:31-32` — satu-satunya tempat
-- seluruh kolom T_SURVEYORLIST disebut berurutan, sehingga tidak perlu ditebak dari alias
-- kueri lain yang terbukti menyesatkan (utang teknis 4.2).
--
-- ## Kenapa `PNCCASEID`, bukan `CASEID`
--
-- Tabel ini punya KEDUANYA. `CASEID` adalah kunci case survei, `PNCCASEID` kunci klaim
-- PNC-nya — dan yang kita punya di form ini adalah `PZINSKEY` klaim. Modul
-- `inboxinvestigator` sudah memakai pasangan yang sama
-- (`inboxinvestigator.sql`, `s.PNCCASEID = a.PZINSKEY`).
--
-- ## Urutan
--
-- `INDEX_SURVEY` — kolom yang MEMANG menyimpan urutan page list di Pega, dan yang
-- `INSERT_SURVEYORLIST` terima sebagai `TSRVINDEX`. Mengurutkan menurut tanggal akan
-- mengubah urutan yang dilihat pengguna ketika dua survei bertanggal sama.
--
-- Tanpa batas baris: satu klaim punya segelintir survei, dan memotongnya diam-diam akan
-- menyembunyikan hasil investigasi dari petugas yang justru sedang memutuskan klaimnya.
SELECT s.SURVEYDATE      AS SURVEYED_AT,
       s.OBJECT_NAME     AS OBJECT_NAME,
       s.LOCATION_OBJECT AS OBJECT_LOCATION,
       s.STS_SURVEY      AS SURVEY_STATUS
  FROM POOLDATA.T_SURVEYORLIST s
 WHERE s.PNCCASEID = :1
 ORDER BY s.INDEX_SURVEY

-- name: find_claim_documents
-- Dokumen satu klaim — grid S8/S9 `Section/CompliancePNC-Section.xml`.
--
-- Bind: :1 IDPEGA (yakni PZINSKEY klaimnya)
--
-- ## Dari mana kunci dan kolomnya dibaca
--
--   `RDB List/CountUpload-SQL.xml`           IDPEGA = pzInsKey, CATEGORY, SUB_CATEGORY,
--                                            dan penyaring IMAGEID IS NOT NULL
--   `RDB List/GetAttachmentFromDB_Sql-SQL`    ATTACHNAME, ATTACHMIMETYPE, INPUTDATE, IMAGEID
--   `RDB List/GetDocumentData-SQL.xml`        DATAID
--
-- ## `IMAGEID IS NOT NULL` disalin apa adanya, dan itu bermakna
--
-- `IMAGEID` adalah kunci berkas di penyimpanan luar, diisi SETELAH `UploadDokumenPNC`
-- berhasil. Baris tanpa IMAGEID adalah unggahan yang belum selesai — berkasnya tidak
-- dapat dibuka. Menampilkannya hanya memberi pengguna baris yang tombolnya tidak bekerja.
--
-- ## Kolom BLOB tidak diambil
--
-- `ATTACHFILE` sengaja TIDAK dibaca. Isi berkas tidak pernah digambar di grid, dan
-- membacanya berarti menarik seluruh BLOB setiap kali form dibuka. Berkasnya diambil
-- lewat tautan dari `NewLinkDokumenPNC`, bukan dari kolom ini.
--
-- Tanpa batas baris: satu klaim punya segelintir lampiran, dan memotongnya diam-diam
-- akan menyembunyikan dokumen dari petugas yang sedang memutuskan klaimnya.
SELECT d.DATAID         AS DOCUMENT_ID,
       d.ATTACHNAME     AS DOCUMENT_NAME,
       d.ATTACHMIMETYPE AS MIME_TYPE,
       d.CATEGORY       AS CATEGORY,
       d.SUB_CATEGORY   AS SUB_CATEGORY,
       d.IMAGEID        AS STORAGE_ID,
       d.INPUTDATE      AS UPLOADED_AT
  FROM POOLDATA.DATA_ATTACHFILE d
 WHERE d.IDPEGA = :1
   AND d.IMAGEID IS NOT NULL
 ORDER BY d.INPUTDATE, d.DATAID

-- name: next_attachment_runno
-- Nomor urut lampiran berikutnya.
--
-- Diisolasi di kueri tersendiri karena `NEXTVAL` adalah sintaks Oracle — pola yang sama
-- dipakai modul `daftardetaildokumentravel` dan `daftardetailtipedokumen`. Padanan
-- PostgreSQL-nya `nextval('pooldata.attachfile_seq')`, dan HANYA baris ini yang berubah.
--
-- Sumbernya `Database/SET_ATTACHMENT_64BIT.prc:19`.
SELECT POOLDATA.ATTACHFILE_SEQ.NEXTVAL AS RUNNO FROM DUAL

-- name: insert_attachment_counter
-- Baris pendamping pada `C_COUNTER_ATTACHMENT`.
--
-- Bind: :1 KEY (UUID) · :2 YEAR (dua digit) · :3 RUNNO
--
-- ## Kenapa tabel ini ikut diisi
--
-- Karena `SET_ATTACHMENT_64BIT.prc:21-27` mengisinya, lalu MEMBACA KEMBALI darinya untuk
-- membentuk `DATAID`:
--
--     INSERT INTO C_COUNTER_ATTACHMENT (KEY, YEAR, RUNNO) VALUES (new_uuid, yy, nextval)
--     SELECT year || lpad(runno,10,'0') INTO tDATAID FROM C_COUNTER_ATTACHMENT WHERE key = pkey
--
-- Melewatinya akan menghasilkan `DATAID` yang bentuknya benar tetapi tidak punya baris
-- pendamping — berbeda dari setiap lampiran yang pernah dibuat Pega, dan perbedaan itu
-- baru terlihat ketika ada yang menelusuri asal sebuah lampiran.
--
-- `new_uuid` sendiri TIDAK ADA di export; ia jelas pembangkit UUID, dan Go
-- membangkitkannya sendiri. Itu satu-satunya bagian yang tidak disalin harfiah.
INSERT INTO POOLDATA.C_COUNTER_ATTACHMENT (KEY, YEAR, RUNNO) VALUES (:1, :2, :3)

-- name: insert_attachment
-- Satu baris lampiran pada `POOLDATA.DATA_ATTACHFILE`.
--
-- Bind: :1 DATAID · :2 INPUTOPERATOR · :3 ATTACHNAME · :4 ATTACHNOTE ·
--       :5 ATTACHMIMETYPE · :6 IMAGEID · :7 CATEGORY · :8 SUB_CATEGORY · :9 IDPEGA
--
-- Kolom dan urutannya mengikuti `SET_ATTACHMENT_64BIT.prc:30-31` apa adanya, kecuali dua
-- hal:
--
--   `INPUTDATE`   diisi CURRENT_TIMESTAMP, bukan SYSDATE — `D-20` portabilitas.
--   `ATTACHFILE`  TIDAK diisi. Procedure pun tidak mengisinya pada jalur ini; isi
--                 berkasnya ada di penyimpanan luar, dan kolom BLOB itu milik jalur
--                 lampiran lama yang menyimpan berkas di basis data.
--
-- Procedure-nya sendiri TIDAK dipanggil (`D-02`); yang disalin adalah perilakunya.
INSERT INTO POOLDATA.DATA_ATTACHFILE
       (DATAID, INPUTDATE, INPUTOPERATOR, ATTACHNAME, ATTACHNOTE,
        ATTACHMIMETYPE, IMAGEID, CATEGORY, SUB_CATEGORY, IDPEGA)
VALUES (:1, CURRENT_TIMESTAMP, :2, :3, :4, :5, :6, :7, :8, :9)

-- name: delete_attachment
-- Menghapus satu baris lampiran.
--
-- Bind: :1 DATAID · :2 IDPEGA
--
-- ## Hapus FISIK, dan itu keputusan sadar
--
-- `D-66` menetapkan soft delete menyeluruh. Di tabel INI ia tidak diberlakukan, atas
-- persetujuan Work Owner (2026-10-07), karena tiga hal:
--
--   1. `DATA_ATTACHFILE` tidak punya kolom penanda hapus, sehingga soft delete menuntut
--      perubahan skema lewat `D-63`.
--   2. Kueri Pega tidak akan menyaring kolom baru itu, sehingga dokumen yang "terhapus"
--      di aplikasi baru TETAP muncul di layar Pega — dua sistem tidak sepakat soal
--      dokumen klaim yang sama.
--   3. Berkasnya sendiri dihapus dari penyimpanan luar. Baris yang ditahan akan menunjuk
--      berkas yang sudah tidak ada, dan tombol Lihat-nya gagal.
--
-- Jejaknya tidak hilang: satu baris riwayat ditulis SEBELUM penghapusan — lihat
-- DeleteDocument pada usecase. Itu yang menutup kerugian utama hapus fisik tanpa
-- menyentuh skema.
--
-- ## Kenapa IDPEGA ikut disyaratkan
--
-- Supaya satu id dokumen yang keliru — atau dikarang — tidak dapat menghapus lampiran
-- milik klaim lain. Procedure Pega menghapus hanya dengan `DATAID`; penyaring kedua ini
-- TAMBAHAN kita, dan ia pengetatan yang disengaja.
DELETE FROM POOLDATA.DATA_ATTACHFILE
 WHERE DATAID = :1
   AND IDPEGA = :2

-- name: find_reject_prefill
-- Isian pra-isi form Surat Penolakan.
--
-- Bind: :1 PZINSKEY klaimnya
--
-- ## Dari mana ketiganya berasal
--
-- `Activity/AutoFillFormReject_Pre-Act.xml` langkah 1 mengisi empat isian:
--
--     RejectCompliance.TanggalKejadian          <- ClaimData.DateOfLoss
--     RejectCompliance.TempatKejadian           <- ClaimData.Location
--     RejectCompliance.NamaPasien               <- ClaimData.ObjectList(1).ObjectName
--     RejectCompliance.TanggalSelesaiRawatInap  <- ClaimData.TanggalSelesaiRawatInap
--
-- Hanya TIGA yang dapat dibaca. Nama kolomnya diambil dari daftar INSERT pada
-- `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:509-515` — sumber sahih, bukan dari alias
-- kueri lain yang di tabel ini terbukti menyesatkan (`BUSINESSNAME AS "Location"`,
-- `ANALYST_TFKOMITEDATE as "DateOfLoss"`).
--
-- ## Yang KEEMPAT tidak ada, dan itu bukan kelalaian
--
-- `TanggalSelesaiRawatInap` **tidak punya kolom** di `T_CLAIM_PNC`. Ia properti klipboard
-- yang hanya muncul sekali di seluruh export — sebagai parameter `TREGISTDATE` pada
-- `RDB List/Rcv_ProcInsertRecivedDocument-SQL.xml`, yakni jalur Receive Document, bukan
-- jalur klaim.
--
-- Akibatnya isian "Tanggal Keluar Rawat Inap" terbuka KOSONG dan diketik petugas. Itu
-- tidak menghalangi: selnya memang dapat disunting (`pxDateTime`, bukan read-only), dan
-- di Pega pun ia kosong ketika klaimnya bukan berasal dari Receive Document.
--
-- ## Nama pasien
--
-- Objek pertama klaim, diurutkan `OBJECTID` — pola yang sama dipakai
-- `inboxinvestigator.sql` untuk kolom "Nama Peserta". Pada lini PA objek pertanggungannya
-- adalah orang, sehingga nama objek memang nama pasiennya.
SELECT a.DATEOFLOSS AS DATE_OF_LOSS,
       a.LOCATION   AS LOSS_LOCATION,
       (SELECT o.OBJECTNAME
          FROM POOLDATA.T_CLAIM_OBJECTLIST o
         WHERE o.CLAIMID = a.CLAIMID
         ORDER BY o.OBJECTID
         FETCH FIRST 1 ROW ONLY) AS PATIENT_NAME
  FROM POOLDATA.T_CLAIM_PNC a
 WHERE a.CLAIMID = :1

-- name: find_document_checklist
-- Daftar periksa kelengkapan dokumen — tab "Dokumen" (lini Travel).
--
-- Bind: :1 PZINSKEY klaimnya, dipakai DUA kali
--
-- ## Ditiru dari dua rule, bukan dikarang
--
-- Bentuk barisnya dari `RDB List/BrowseRegister_upload-SQL.xml`, dan pencacahnya dari
-- `RDB List/CountUpload-SQL.xml`. Keduanya memakai master dan kunci yang sama dengan
-- grid pada layar Compliance, sehingga penggabungannya di sini bukan rancangan baru.
--
-- ## Penyaring TAHAP dicabut 2026-10-10, atas pengukuran — bukan atas dugaan
--
-- Versi pertama menyaring `V_LST_DOC_TYPE.TYPE_DOCUMENT = 'REGISTER'`, meniru
-- `BrowseRegister_upload-SQL.xml`. Pengukuran langsung ke basis data membatalkannya:
--
--     seluruh LST_TYPE_DOC_BUSINESS  1848 baris bertahap KOSONG, 2 baris bertahap "Dokumen"
--     lini Travel klaim contoh       6 baris, SELURUHNYA bertahap NULL
--
-- Dengan penyaring itu, tab Dokumen akan KOSONG untuk setiap klaim Travel. Tanpanya ia
-- mengembalikan keenam kategorinya. Jadi penyaring tahap bukan hanya tidak terbukti —
-- ia terbukti SALAH pada data yang ada.
--
-- Bila kelak produksi mengisi `TYPE_DOCUMENT`, penyaringnya mungkin perlu kembali.
-- Yang membatalkannya adalah pengukuran di atas, dan pengukuran itu dapat diulang kapan
-- saja lewat `claimpnc -periksa` yang mencetak cacahan per tahap.
--
-- ## Dua penyaring yang ditiru apa adanya
--
--     DETAIL_DOKUMEN <> '-'   membuang baris penanda; ada di rule aslinya
--     ORDER BY DETAIL_DOKUMEN ASC   urutan aslinya, bukan menurut id
--
-- ## Kolom "Wajib Unggah" punya EMPAT keadaan, bukan dua
--
-- CASE di bawah menyalin `BrowseRegister_upload` persis, termasuk akibat yang mudah
-- terlewat: ketika `STS_WAJIB` NULL, tidak ada satu pun arm yang cocok dan hasilnya
-- NULL — tergambar KOSONG. Itu memang yang terlihat pada layar Pega yang diperlihatkan
-- Work Owner: kolom ini kosong di seluruh baris, sementara "Minimal Unggah" terisi.
--
-- Satu penyimpangan yang tidak terhindarkan: rule aslinya membandingkan `OBJECT_DOC_ID`
-- dengan `{ASIS:TempParam.BRANCH_NAME}` — sebuah DAFTAR cabang yang dirangkai ke dalam
-- teks SQL. Dari mana daftar itu diisi tidak terbaca di export, dan merangkai daftar ke
-- dalam SQL dilarang (`{ASIS:}` adalah celah injeksi yang justru kita hapus).
--
-- Di sini pembandingnya **cabang klaim itu sendiri**. Akibatnya terbatas dan dapat
-- dinyatakan: baris yang `OBJECT_DOC_ID`-nya menunjuk cabang LAIN akan terbaca "Tidak"
-- di sini, sedangkan Pega dapat menyebutnya "Ya" bila cabang itu termasuk daftar
-- petugasnya. Arm ketiga menangkap keduanya, sehingga tidak ada baris yang kehilangan
-- nilai — yang berbeda hanya "Ya" versus "Tidak" pada baris lintas cabang.
--
-- ## Pencacah memakai IMAGEID IS NOT NULL
--
-- Sama dengan `CountUpload` dan dengan kueri daftar dokumen kami. Baris tanpa kunci
-- penyimpanan adalah unggahan yang tidak selesai — berkasnya tidak pernah dapat dibuka,
-- sehingga menghitungnya membuat kategori tampak lengkap padahal kosong.
SELECT b.DOC_TYPE_DT_ID AS CATEGORY_ID,
       b.DETAIL_DOKUMEN AS CATEGORY_NAME,
       CASE
           WHEN b.STS_WAJIB = '0' THEN 'Tidak'
           WHEN b.STS_WAJIB = '1' AND b.OBJECT_DOC_ID = c.BRANCHCODE THEN 'Ya'
           WHEN b.STS_WAJIB = '1' THEN 'Tidak'
       END              AS MANDATORY_LABEL,
       b.MIN_DOC        AS MIN_UPLOAD,
       (SELECT COUNT(*)
          FROM POOLDATA.DATA_ATTACHFILE d
         WHERE d.IDPEGA = :1
           AND d.IMAGEID IS NOT NULL
           AND d.CATEGORY = b.DOC_TYPE_DT_ID) AS UPLOADED_COUNT
  FROM POOLDATA.T_CLAIM_PNC c
       INNER JOIN POOLDATA.LST_TYPE_DOC_BUSINESS b
               ON b.BUSINESSID = c.BUSINESSCODE
 WHERE c.CLAIMID = :2
   AND b.DETAIL_DOKUMEN <> '-'
 ORDER BY b.DETAIL_DOKUMEN ASC

-- name: check_table_document_checklist
-- Memeriksa ketiga objek tab Dokumen ADA dan terbaca.
--
-- Ketiganya disebut dalam satu kueri karena ketiganya dibutuhkan bersama: master jenis
-- dokumen tanpa view tahapnya tidak dapat disaring, dan tanpa tabel lampiran pencacahnya
-- gagal. Satu pemeriksaan, satu jawaban.
SELECT b.DOC_TYPE_DT_ID, b.DETAIL_DOKUMEN, b.STS_WAJIB, b.MIN_DOC,
       b.OBJECT_DOC_ID, b.BUSINESSID, b.DOCUMENT_TYPE_ID,
       a.ID, a.TYPE_DOCUMENT,
       d.CATEGORY
  FROM POOLDATA.LST_TYPE_DOC_BUSINESS b,
       POOLDATA.V_LST_DOC_TYPE a,
       POOLDATA.DATA_ATTACHFILE d
 WHERE 1 = 0

-- name: update_document_category
-- Memindahkan satu lampiran ke kategori lain — tombol "Ubah Kategori Dok".
--
-- Bind: :1 kategori baru, :2 DATAID dokumennya, :3 PZINSKEY klaimnya
--
-- ## Kenapa satu UPDATE, sedangkan Pega menempuh sembilan belas langkah
--
-- `Activity/SetCategoryAttachment-Act.xml` panjang bukan karena aturannya rumit,
-- melainkan karena model lampiran Pega TERPECAH di tiga tempat: `Link-Attachment`,
-- halaman `WorkAttach`, dan tabel `TEMP_DATA_ATTACHMENT`. Kategori yang sama harus
-- ditulis ke ketiganya, masing-masing dengan Obj-Save dan Commit sendiri — ditambah
-- pembacaan ulang metadata berkas lewat `ASMGetMetaDataGCNM`.
--
-- Di sistem baru lampirannya SATU baris di `DATA_ATTACHFILE` (`D-16`, `D-21`), dan
-- kategorinya satu kolom. Tidak ada yang perlu disinkronkan, sehingga tidak ada yang
-- dapat gagal separuh jalan — kebalikan dari jalur Pega yang Commit-nya empat kali.
--
-- ## Dua penyaring, dan keduanya wajib
--
-- `DATAID` saja TIDAK cukup. Tanpa `IDPEGA`, siapa pun yang mengetahui sebuah DATAID
-- dapat memindahkan dokumen milik klaim lain — dan layanan penyimpanan di seberang tidak
-- memeriksa kepemilikan apa pun. Pola yang sama dipakai `delete_attachment`.
UPDATE POOLDATA.DATA_ATTACHFILE
   SET CATEGORY = :1
 WHERE DATAID = :2
   AND IDPEGA = :3

-- name: count_document_stages
-- Mencacah kategori dokumen per TAHAP — menjawab asumsi DocumentChecklistStage.
--
-- Tanpa bind: ia diagnostik, dijalankan perintah `check`, bukan jalur layar.
--
-- Angkanya dibandingkan dengan jumlah baris grid "DOKUMEN TRAVEL" pada layar Pega. Tahap
-- yang cocok adalah tahap yang sebenarnya dipakai tab Dokumen.
--
-- Penyaring `DETAIL_DOKUMEN <> '-'` ditiru dari `BrowseRegister_upload-SQL.xml`, supaya
-- cacahannya sepadan dengan apa yang benar-benar digambar grid.
SELECT a.TYPE_DOCUMENT AS STAGE,
       COUNT(*)        AS JUMLAH
  FROM POOLDATA.LST_TYPE_DOC_BUSINESS b
       INNER JOIN POOLDATA.V_LST_DOC_TYPE a
               ON a.ID = b.DOCUMENT_TYPE_ID
 WHERE b.DETAIL_DOKUMEN <> '-'
 GROUP BY a.TYPE_DOCUMENT
 ORDER BY a.TYPE_DOCUMENT
