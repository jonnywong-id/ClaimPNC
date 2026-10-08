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
       p.COMPLIANCE_CREATEDATE    AS COMPLIANCE_SENT_DATE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET wb
               ON wb.PXREFOBJECTKEY = A.PZINSKEY
              AND wb.PXOBJCLASS = 'Assign-WorkBasket'
       LEFT JOIN POOLDATA.T_CLAIM_PNC p
              ON p.CLAIMID = A.PZINSKEY
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND wb.PXASSIGNEDOPERATORID = :1
   AND A.PYSTATUSWORK <> 'Resolved-Completed'
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
              AND wb.PXOBJCLASS = 'Assign-WorkBasket'
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND wb.PXASSIGNEDOPERATORID = :1
   AND A.PYSTATUSWORK <> 'Resolved-Completed'

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
-- Memeriksa tabel tab Post Audit terbaca dari koneksi yang dipakai.
--
-- Terpisah dari check_table karena tabelnya pun terpisah: tab Compliance membaca tabel
-- warisan Pega, tab Post Audit membaca tabel datar yang BARU. Kegagalan keduanya berarti
-- hal yang berbeda, sehingga pesannya pun harus berbeda.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIM_COMPLIANCE_H
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
       p.COMPLIANCE_CREATEDATE    AS COMPLIANCE_SENT_DATE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET wb
               ON wb.PXREFOBJECTKEY = A.PZINSKEY
              AND wb.PXOBJCLASS = 'Assign-WorkBasket'
       LEFT JOIN POOLDATA.T_CLAIM_PNC p
              ON p.CLAIMID = A.PZINSKEY
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND wb.PXASSIGNEDOPERATORID = :1
   AND A.PYSTATUSWORK <> 'Resolved-Completed'
   AND A.PZINSKEY = :2
 FETCH FIRST 1 ROW ONLY

-- name: post_audit_next_sequence
-- Mengambil nomor urut berikutnya untuk kolom CASEID.
--
-- `NEXTVAL` adalah sintaks Oracle dan tidak portabel ke PostgreSQL, yang memakai
-- `nextval('nama_seq')`. Pengecualian dialek itu tidak terhindarkan, dan ia sengaja
-- DIISOLASI di kueri tersendiri — sama seperti generator nomor laporan pada modul
-- Pelaporan Klaim, dan sejalan dengan `D-22` yang menetapkan sakelar dialek hanya boleh
-- ada di satu tempat per kegunaan.
--
-- Angkanya diambil terpisah, lalu nomornya dirakit di Go. Merakitnya di SQL akan menyeret
-- pemformatan teks ke dalam kueri, yang `09-DATABASE-STRATEGY.md` §3.2 larang.
SELECT POOLDATA.CPNC_POST_AUDIT_SEQ.NEXTVAL
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
       TGL_KIRIM_POST_AUDIT AS SENT_TO_POST_AUDIT_AT
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
                      t.TGL_KIRIM_POST_AUDIT = :8
      WHEN NOT MATCHED THEN
           INSERT (NO_KLAIM, PILIHAN, NOTE, CATATAN,
                   DIPUTUSKAN_OLEH, DIPUTUSKAN_PADA, TGL_VALID, TGL_KIRIM_POST_AUDIT)
           VALUES (s.NO_KLAIM, :2, :3, :4, :5, :6, :7, :8)

-- name: check_table_decision
-- Membuktikan tabel keputusan Compliance ada DAN dapat dibaca akun aplikasi.
--
-- `FETCH FIRST 0 ROWS ONLY` memaksa Oracle mengurai dan memeriksa hak akses tanpa membaca
-- satu baris pun — sama polanya dengan check_table_post_audit.
SELECT 1
  FROM POOLDATA.CPNC_KEPUTUSAN_COMPLIANCE
 FETCH FIRST 0 ROWS ONLY
