-- Kueri modul Inbox Manager Receive / PUCL.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- SELURUH tabel yang dibaca berkas ini milik sistem lama. Tidak ada satu pun pernyataan yang
-- menulis, dan memang tidak boleh ada: selama masa paralel setiap tabel hanya boleh ditulis
-- SATU sistem, dan tabel-tabel ini milik Pega (`P-1`).
--
-- ============================================================================
-- SATU TABEL OBJEK KERJA, DUA KELAS YANG BERBEDA SAMA SEKALI
-- ============================================================================
--
-- DATAPEGA.PC_ASM_FW_GCNMFW_WORK menampung DUA jenis objek kerja sekaligus, dan yang
-- membedakannya hanya kolom PXOBJCLASS:
--
--   'ASM-FW-GCNMFW-Work-ReceiveDocument'  berkas penerimaan dokumen   -> tab Receive
--   'ASM-FW-GCNMFW-Work-PNC'              klaim PNC                   -> tab RCL/PUCL
--
-- Melupakan penyaring itu mencampur berkas penerimaan dokumen dengan klaim. Keduanya punya
-- PYID, POLICYNO, dan QQNAME, sehingga hasilnya TIDAK menghasilkan galat apa pun — ia hanya
-- menampilkan baris yang tidak seharusnya ada, dengan kolom yang kebetulan terisi.
--
-- Tabel PENUGASANNYA pun berbeda, dan itu bukan pilihan gaya:
--
--   tab Receive    DATAPEGA.PC_ASSIGN_WORKLIST     penugasan per orang
--   tab RCL/PUCL   DATAPEGA.PC_ASSIGN_WORKBASKET   antrean bersama
--
-- Keduanya diambil dari Report Definition dan kueri aslinya, bukan disamakan.
--
-- ============================================================================
-- PEMETAAN KOLOM — properti Pega -> kolom sebenarnya -> alias di sini
-- ============================================================================
--
-- TAB RECEIVE — dari `Report Definition/ManagementRecieveView-RD.xml`
--
--   properti Pega                       kolom sebenarnya              alias di sini
--   ----------------------------------- ----------------------------- ---------------------
--   .pzInsKey                           w.PZINSKEY                    REFERENCE
--   .pyID                               w.PYID                        CASE_ID
--   .ReceiveDocument.PolicyNo           w.POLICYNO                    POLICY_NUMBER
--   .ReceiveDocument.PNCCaseID          w.PNCCASEID                   CLAIM_NUMBER
--   .ReceiveDocument.QQName             w.QQNAME                      INSURED_NAME
--   .ReceiveDocument.DateOfLoss         w.DATEOFLOSS_1                LOSS_DATE
--   .ReceiveDocument.TypeOfClaim        (TIDAK ADA — lihat catatan 1)  GROUP_PANEL
--   .ReceiveDocument.Sender             r.NAMAPELAPOR                 SENDER_NAME
--   .ReceiveDocument.ReceivedDate       r.TANGGALTERIMADOKUMEN        DOCUMENT_RECEIVED_DATE
--   .ReceiveDocument.NumberOfDocument   (TIDAK ADA — lihat catatan 2)  DOCUMENT_SHEET_COUNT
--
-- TAB RCL/PUCL — dari `Report Definition/InboxRCLPUCL_RD-RD.xml`, dengan nama kolom yang
-- dipastikan lewat `RDB List/ReminderPUCL-SQL.xml`
--
--   properti Pega                                 kolom sebenarnya             alias di sini
--   --------------------------------------------- ---------------------------- -------------
--   .pyID                                         w.PYID                       CASE_ID
--   .Policy.PolicyNo                              w.POLICYNO                   POLICY_NUMBER
--   .Policy.QQName                                w.QQNAME                     INSURED_NAME
--   .pxCreateDateTime                             w.PXCREATEDATETIME           INBOX_ENTRY_AT
--   .ClaimData.PUCLStatus.KomentarAnalisator      w.KOMENTARANALISATOR_1       ANALYST_NOTE
--   .ClaimData.PUCLStatus.RCL_PUCL                w.RCL_PUCL_1                 TRACK
--   .ClaimData.PUCLStatus.StatusKlaim             w.STATUSKLAIM_1              TRACK_STATUS
--   .ClaimData.PUCLStatus.TanggalCetakDokumenPUCL w.TANGGALCETAKDOKUMENPUCL_1  LETTER_PRINTED_AT
--   .ClaimData.PUCLStatus.LamaKlaim               w.LAMAKLAIM_1                CLAIM_AGE
--   .ClaimData.PUCLStatus.StatusCase              w.STATUSCASE_1               EXPIRY_STATUS
--
-- Alias Pega yang TIDAK dibawa (`D-19`), karena tidak satu pun menyatakan isinya:
--
--   KOMENTARANALISATOR_1  dialiaskan "NoteKomite" di GetReminderPUCL, "LOGSEARCH" di ReminderPUCL
--   LAMAKLAIM_1           dialiaskan "LOGSEEN" di GetReminderPUCL, "MODUL" di ReminderPUCL
--   STATUSKLAIM_1         dialiaskan "StsAcceptance" di GetReminderPUCL, "STS_EMAIL" di ReminderPUCL
--   QQNAME                dialiaskan "NewTelpTertanggung" di GetReminderPUCL, "CABANG" di ReminderPUCL
--   STATUSCASE_1          dialiaskan "ClaimData.PUCLStatus.Stat25L" — nama terpotong batas alias Oracle
--
-- Baris terakhir patut diperhatikan: satu kolom yang sama dialiaskan "CABANG" di satu rule
-- dan "NewTelpTertanggung" di rule lain, padahal isinya nama tertanggung. Itu utang teknis
-- §4.2 apa adanya, dan tidak satu pun dibawa.
--
-- ============================================================================
-- CATATAN 1 — `.ReceiveDocument.TypeOfClaim` TIDAK PUNYA KOLOM
-- ============================================================================
--
-- `ManagementRecieveView-RD.xml` menandainya sendiri:
--
--   <pzPropertyType>unexposed</pzPropertyType>
--
-- ditambah dua peringatan Pega — "Not optimized for reporting" dan "Not optimized for
-- filtering" — yang keduanya menyebut properti itu satu-satunya. Artinya ia hidup di dalam
-- blob Pega dan tidak dapat disaring SQL.
--
-- Penelusuran seluruh export menguatkannya: `TYPEOFCLAIM_1`, `SENDER_1`, dan
-- `NUMBEROFDOCUMENT_1` NOL KEMUNCULAN, sementara 15 kueri lain yang membaca kelas yang sama
-- memakai kolom `STATUSLOCK_1`, `KODECABANG_1`, `DATEFORAGING_1`, `BUSINESSCODE_1`,
-- `DATEOFLOSS_1`, `BOOKNO_1`, dan `GROUPPANEL_1`.
--
-- Penggantinya `GROUPPANEL_1`: `002` adalah Personal Accident (`CONTEXT.md`), dan kolomnya
-- memang dibaca kueri lain pada kelas yang sama
-- (`RDB List/GetDataRCVallKlaimPATravel-SQL.xml:10`). Keputusan Work Owner 2026-09-22,
-- dinyatakan ke pengguna lewat PlannedDifferences.
--
-- Perhatikan akibatnya pada baris tanpa Group Panel: `GROUPPANEL_1 <> '002'` TIDAK menangkap
-- NULL di Oracle maupun PostgreSQL, sehingga baris itu tidak muncul di tab mana pun. Itu
-- perilaku yang sama dengan layar lama, tempat berkas tanpa `TypeOfClaim` tidak cocok dengan
-- grid mana pun.
--
-- ============================================================================
-- CATATAN 2 — "Jumlah Lembar Dokumen" MEMANG KOSONG
-- ============================================================================
--
-- Tidak ada kolom untuknya, dan POOLDATA.T_CLAIM_RECIVEDCLAIM tidak menyimpannya:
-- procedure yang mengisinya (`Database/PROCINSERTDATARECIVEDKLAIM.prc`) menerima 26
-- parameter dan tidak satu pun berisi jumlah lembar.
--
-- `CAST(NULL …)` dipakai, bukan kolomnya dihilangkan, karena ke-16 alias WAJIB sama di setiap
-- kueri — lihat bagian berikutnya. Kolomnya tetap DIGAMBAR di layar supaya isian yang belum
-- terbawa terlihat, bukan tersamar sebagai layar yang sudah setara.
--
-- ============================================================================
-- CATATAN 3 — TABEL CERMIN YANG TIDAK PERNAH DIBACA
-- ============================================================================
--
-- POOLDATA.T_CLAIM_RECIVEDCLAIM memasok dua kolom tab Receive: NAMAPELAPOR ("Nama Pengirim")
-- dan TANGGALTERIMADOKUMEN ("Tanggal Terima Dokumen").
--
-- Bahwa NAMAPELAPOR memang "nama pengirim" terbaca dari label layar input:
-- `Section/ViewInputReceiveDocument_sec-Section.xml` memberi `.ReceiveDocument.Sender` judul
-- "Nama Pengirim / Pelapor Dokumen". Ia BUKAN NAMAKURIRASM, yang berjudul "Nama Kurir ASM".
--
-- Yang harus disadari: tabel itu TIDAK PERNAH DIBACA sistem lama. Penelusuran seluruh export
-- menemukan satu-satunya penyentuhnya adalah procedure yang MENULISINYA. Kelengkapan isinya
-- karena itu belum terverifikasi, dan gabungannya sengaja `LEFT JOIN` — baris tanpa pasangan
-- tetap muncul dengan kedua kolom kosong, bukan hilang dari daftar.
--
-- Kuncinya `CLAIMID = w.PZINSKEY`, dan itu bukan tebakan: procedure-nya menerima
-- `TCLAIMID := {TampunganPages.pzInsKey}` (`Rcv_ProcInsertRecivedDocument-SQL.xml`).
--
-- ============================================================================
-- KE-17 ALIAS WAJIB SAMA DI SETIAP KUERI
-- ============================================================================
--
-- Urutan DAN namanya. Dua hal bergantung padanya:
--
--   * satu pemindai Go melayani ketiga kueri (scanWorkItem di
--     inboxmanagerreceivepucl.go);
--   * uji query_test.go menjaganya, dan ia gagal bila ada kueri yang aliasnya berbeda.
--
-- Kolom yang tidak berlaku bagi sebuah kueri bernilai NULL, bukan dihilangkan. Layar
-- menyembunyikannya mengikuti Tab.Columns — bukan menampilkan kolom kosong yang membuat
-- pengguna menduga datanya hilang.
--
-- ============================================================================
-- YANG BERUBAH DARI SISTEM LAMA, DAN KENAPA
-- ============================================================================
--
-- 1. PAGINASI DIKERJAKAN BASIS DATA.
--    Ketiga grid di sistem lama memotong hasilnya di `pyMaxRecords` 500 SETELAH seluruh
--    barisnya ditarik, lalu menomori halamannya di klipboard. Di sini halamannya dipotong
--    dengan `OFFSET … FETCH NEXT … ROWS ONLY` sebelum baris meninggalkan basis data —
--    didukung Oracle 12c+ dan PostgreSQL (`09-DATABASE-STRATEGY.md` §3.3). Ini PERUBAHAN
--    PERILAKU yang disadari, bukan pemeliharaan.
--
-- 2. JUMLAH SELURUH BARIS DIHITUNG `COUNT(*) OVER ()`.
--    Satu perjalanan, bukan dua: kueri kedua yang hanya menghitung akan membaca ulang
--    gabungan yang sama, dan gabungan itulah bagian yang mahal. Fungsi jendela dihitung
--    SEBELUM `OFFSET … FETCH` dipakai, sehingga angkanya jumlah seluruhnya — bukan jumlah
--    baris di halaman ini.
--
-- 3. `ORDER BY` DITAMBAHKAN PADA KETIGA KUERI.
--    Tidak satu pun dari ketiga Report Definition menetapkan urutan (`pySortOrder` 99999
--    pada seluruh isiannya berarti "tidak diurutkan"). Itu dapat dibiarkan selama seluruh
--    baris ditarik sekaligus; begitu halamannya dipotong, urutan yang tidak ditetapkan
--    membuat satu baris muncul di dua halaman sekaligus hilang dari halaman lain.
--
--    Yang dipilih `PXCREATEDATETIME DESC` — yang terbaru masuk lebih dulu — dengan `PYID`
--    sebagai pemutus supaya dua baris berwaktu sama tetap berurutan tetap.
--
-- 4. GABUNGAN DITULIS SEBAGAI `JOIN`, BUKAN DAFTAR TABEL BERKOMA.
--    `ReminderPUCL-SQL.xml` sudah memakai `INNER JOIN`; `CountKlaimPUCL-SQL.xml` pun. Yang
--    diubah hanyalah tempat syarat penyaring ditulis, bukan isinya.
--
-- 5. NILAI SELALU LEWAT PARAMETER BINDING.
--    Kueri lama menyisipkan `{TempRCLPUCLReport.AlasanKlaim}` dan saudaranya langsung ke
--    teks SQL (`GetDataPUCLRCLForDailyReport-SQL.xml`). Larangan perangkaian
--    (`08-TECHNICAL-STRATEGY.md` §4.3) tidak dikecualikan oleh keputusan mana pun: yang
--    direplikasi adalah perilaku bisnis, bukan celah injeksi.
--
-- ============================================================================
-- YANG SENGAJA TIDAK BERUBAH
-- ============================================================================
--
-- * `INNER JOIN` ke tabel penugasan, bukan `EXISTS`. Report Definition-nya memakai gabungan
--   dalam, sehingga objek kerja yang punya DUA penugasan terbuka muncul DUA KALI. Itu
--   perilaku sistem lama apa adanya (`P-5`), dan menggantinya dengan `EXISTS` akan mengubah
--   jumlah baris yang terlihat pengguna tanpa satu pun keputusan yang mendasarinya.
--
-- * Penerjemahan `RCL_PUCL_1` ditulis sebagai `CASE` tanpa `ELSE`, persis seperti
--   `GetReminderPUCL-SQL.xml:7-9`. Nilai di luar `1` dan `2` karena itu menghasilkan NULL,
--   bukan teks lain — dan sel kosong di layar adalah jawaban yang benar untuk jalur yang
--   tidak dikenali.
--
-- * Ketiga penyaring tambahan `ReminderPUCL-SQL.xml` TIDAK dibawa:
--   `TANGGALCETAKDOKUMENPUCL_1 IS NOT NULL`, `PUCLAPPROVE_1 <> '1'`, dan `MSIG_1 IS NULL`.
--   Ketiganya milik JOB PENGINGAT — klaim yang suratnya sudah dicetak tetapi belum
--   disetujui — bukan milik antrean inbox. Membawanya akan menyembunyikan klaim yang
--   suratnya belum dicetak, padahal justru itu yang menunggu tindakan.
--
-- * `CAST(NULL AS VARCHAR2(…))` memakai tipe khas Oracle, mengikuti modul yang sudah ada
--   (riwayatklaim, inboxadmin, inboxclaimtreatyprop, inboxclaimtreatynonprop). Ia
--   satu-satunya bentuk tak portabel di berkas ini dan tercatat sebagai utang teknis yang
--   diselesaikan serentak untuk seluruh modul saat perpindahan ke PostgreSQL
--   (`09-DATABASE-STRATEGY.md` §10), bukan sepihak di sini.
--
-- CATATAN PENANDA BIND. Berkas ini memakai gaya Oracle `:n`, sama seperti seluruh modul lain
-- di aplikasi ini. Ia BELUM portabel ke PostgreSQL yang memakai `$n`; itu utang yang sudah
-- ada sebelum modul ini dan berlaku untuk seluruh berkas .sql di sini.

-- name: list_receive_pa
-- Tab Receive PA — berkas penerimaan dokumen Personal Accident.
-- — Report Definition/ManagementRecieveView-RD.xml, dijalankan dengan Position1="PA"
--
-- Bind: :1 kode Group Panel PA · :2 offset · :3 jumlah baris
SELECT w.PZINSKEY                       AS REFERENCE,
       w.PYID                           AS CASE_ID,
       w.POLICYNO                       AS POLICY_NUMBER,
       w.PNCCASEID                      AS CLAIM_NUMBER,
       w.QQNAME                         AS INSURED_NAME,
       w.DATEOFLOSS_1                   AS LOSS_DATE,
       w.GROUPPANEL_1                   AS GROUP_PANEL,
       r.NAMAPELAPOR                    AS SENDER_NAME,
       r.TANGGALTERIMADOKUMEN           AS DOCUMENT_RECEIVED_DATE,
       CAST(NULL AS VARCHAR2(50))       AS DOCUMENT_SHEET_COUNT,
       w.PXCREATEDATETIME               AS INBOX_ENTRY_AT,
       CAST(NULL AS VARCHAR2(4000))     AS ANALYST_NOTE,
       CAST(NULL AS VARCHAR2(10))       AS TRACK,
       CAST(NULL AS VARCHAR2(100))      AS TRACK_STATUS,
       CAST(NULL AS VARCHAR2(100))      AS LETTER_PRINTED_AT,
       CAST(NULL AS VARCHAR2(100))      AS CLAIM_AGE,
       CAST(NULL AS VARCHAR2(100))      AS EXPIRY_STATUS,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST a
               ON a.PXREFOBJECTKEY = w.PZINSKEY
       LEFT JOIN POOLDATA.T_CLAIM_RECIVEDCLAIM r
              ON r.CLAIMID = w.PZINSKEY
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-ReceiveDocument'
   AND w.GROUPPANEL_1 = :1
 ORDER BY w.PXCREATEDATETIME DESC, w.PYID
OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY

-- name: list_receive_non_mbu
-- Tab Receive NONMBU — berkas penerimaan dokumen di luar Personal Accident.
-- — Report Definition/ManagementRecieveView-RD.xml, dijalankan dengan Position1="NONMBU"
--
-- Bedanya dengan list_receive_pa HANYA arah pembanding Group Panel. Keduanya sengaja tidak
-- disatukan menjadi satu kueri berparameter arah: penyaring yang artinya berbalik menurut
-- nilai bind adalah tempat paling mudah menampilkan lini bisnis yang salah, dan tidak ada
-- apa pun di layar yang menandakannya bila itu terjadi.
--
-- Bind: :1 kode Group Panel PA (yang DIKECUALIKAN) · :2 offset · :3 jumlah baris
SELECT w.PZINSKEY                       AS REFERENCE,
       w.PYID                           AS CASE_ID,
       w.POLICYNO                       AS POLICY_NUMBER,
       w.PNCCASEID                      AS CLAIM_NUMBER,
       w.QQNAME                         AS INSURED_NAME,
       w.DATEOFLOSS_1                   AS LOSS_DATE,
       w.GROUPPANEL_1                   AS GROUP_PANEL,
       r.NAMAPELAPOR                    AS SENDER_NAME,
       r.TANGGALTERIMADOKUMEN           AS DOCUMENT_RECEIVED_DATE,
       CAST(NULL AS VARCHAR2(50))       AS DOCUMENT_SHEET_COUNT,
       w.PXCREATEDATETIME               AS INBOX_ENTRY_AT,
       CAST(NULL AS VARCHAR2(4000))     AS ANALYST_NOTE,
       CAST(NULL AS VARCHAR2(10))       AS TRACK,
       CAST(NULL AS VARCHAR2(100))      AS TRACK_STATUS,
       CAST(NULL AS VARCHAR2(100))      AS LETTER_PRINTED_AT,
       CAST(NULL AS VARCHAR2(100))      AS CLAIM_AGE,
       CAST(NULL AS VARCHAR2(100))      AS EXPIRY_STATUS,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST a
               ON a.PXREFOBJECTKEY = w.PZINSKEY
       LEFT JOIN POOLDATA.T_CLAIM_RECIVEDCLAIM r
              ON r.CLAIMID = w.PZINSKEY
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-ReceiveDocument'
   AND w.GROUPPANEL_1 <> :1
 ORDER BY w.PXCREATEDATETIME DESC, w.PYID
OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY

-- name: list_rclpucl
-- Tab RCL/PUCL — klaim yang ditolak atau diproses ulang, menunggu keputusan.
-- — Report Definition/InboxRCLPUCL_RD-RD.xml untuk kolom dan penyaring status
-- — RDB List/CountKlaimPUCL-SQL.xml untuk gabungan antrean bersama
-- — RDB List/ReminderPUCL-SQL.xml untuk nama kolom sebenarnya
--
-- Tiga kolom tab Receive bernilai NULL di sini, dan ketiganya memang tidak berlaku: berkas
-- penerimaan dokumen punya nomor klaim PNC, nama pengirim, dan tanggal terima dokumen —
-- klaim tidak.
--
-- Bind: :1 status kerja yang dikecualikan · :2 akun antrean bersama · :3 offset
--       :4 jumlah baris
SELECT w.PZINSKEY                       AS REFERENCE,
       w.PYID                           AS CASE_ID,
       w.POLICYNO                       AS POLICY_NUMBER,
       CAST(NULL AS VARCHAR2(100))      AS CLAIM_NUMBER,
       w.QQNAME                         AS INSURED_NAME,
       CAST(NULL AS VARCHAR2(100))      AS LOSS_DATE,
       CAST(NULL AS VARCHAR2(10))       AS GROUP_PANEL,
       CAST(NULL AS VARCHAR2(255))      AS SENDER_NAME,
       CAST(NULL AS VARCHAR2(100))      AS DOCUMENT_RECEIVED_DATE,
       CAST(NULL AS VARCHAR2(50))       AS DOCUMENT_SHEET_COUNT,
       w.PXCREATEDATETIME               AS INBOX_ENTRY_AT,
       w.KOMENTARANALISATOR_1           AS ANALYST_NOTE,
       CASE
           WHEN w.RCL_PUCL_1 = '1' THEN 'RCL'
           WHEN w.RCL_PUCL_1 = '2' THEN 'PUCL'
       END                              AS TRACK,
       w.STATUSKLAIM_1                  AS TRACK_STATUS,
       w.TANGGALCETAKDOKUMENPUCL_1      AS LETTER_PRINTED_AT,
       w.LAMAKLAIM_1                    AS CLAIM_AGE,
       w.STATUSCASE_1                   AS EXPIRY_STATUS,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET b
               ON b.PXREFOBJECTKEY = w.PZINSKEY
              AND b.PXOBJCLASS = 'Assign-WorkBasket'
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND w.PYSTATUSWORK <> :1
   AND b.PXASSIGNEDOPERATORID = :2
 ORDER BY w.PXCREATEDATETIME DESC, w.PYID
OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY

-- name: check_receive
-- Dipakai perintah `-periksa`: memastikan ketiga tabel tab Receive terbaca dari koneksi yang
-- dipakai.
--
-- Ia tidak menyentuh satu baris pun — yang diperiksa adalah hak baca dan keberadaan
-- tabelnya, bukan isinya. Ketiganya diperiksa sekaligus karena kegagalan yang paling mungkin
-- terjadi bukan "tabel tidak ada" melainkan "hak baca hanya diberikan pada sebagiannya".
--
-- `COUNT(*)` dipakai, bukan sebuah kolom, supaya hasilnya SELALU tepat satu baris meski
-- penyaringnya tidak meloloskan apa pun — pemanggil karena itu tidak perlu membedakan
-- "tidak ada baris" dari "gagal dibaca".
SELECT COUNT(*) AS PROBE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST a
               ON a.PXREFOBJECTKEY = w.PZINSKEY
       LEFT JOIN POOLDATA.T_CLAIM_RECIVEDCLAIM r
              ON r.CLAIMID = w.PZINSKEY
 WHERE 1 = 0

-- name: check_rclpucl
-- Memastikan tabel antrean bersama terbaca. Ia terpisah dari check_receive karena tabelnya
-- memang berbeda, dan galat yang menyebut tabel yang salah menyesatkan orang yang
-- memperbaikinya.
SELECT COUNT(*) AS PROBE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET b
               ON b.PXREFOBJECTKEY = w.PZINSKEY
 WHERE 1 = 0
