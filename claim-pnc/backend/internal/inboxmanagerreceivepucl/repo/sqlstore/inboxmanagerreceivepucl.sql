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
-- SUMBER BARU (2026-10-08): tabel objek kerja Pega TIDAK DIPAKAI LAGI. Tab RCL/PUCL membaca
-- `POOLDATA.TC_PNC_PUCL`; tab Receive digerakkan `PC_ASSIGN_WORKLIST` (disaring
-- `PXREFOBJECTCLASS`) dan kolom berkasnya diambil dari `T_CLAIMLIST_ADMIN`,
-- `T_CLAIM_RECIVEDCLAIM`, dan `T_CLAIM_PNC` — rinciannya di kepala list_receive dan
-- detail_receive_document. Kolom `w.…` pada tabel pemetaan di bawah adalah REKAMAN asal
-- properti Pega, bukan kolom yang kini dibaca.
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

-- name: list_receive
-- Tab Receive — berkas penerimaan dokumen, Personal Accident maupun di luarnya.
-- — Report Definition/ManagementRecieveView-RD.xml, dijalankan DUA KALI di layar lama:
--   sekali dengan Position1="PA", sekali dengan Position1="NONMBU"
--
-- ============================================================================
-- KENAPA SATU KUERI, DAN KENAPA HIMPUNAN BARISNYA TETAP SAMA
-- ============================================================================
--
-- Kedua grid Pega tampil bersamaan di dalam satu tab (`pyVisible` ALWAYS pada keduanya),
-- sehingga yang dilihat pengguna adalah gabungan keduanya. Modul ini menggambarnya sebagai
-- satu tabel, dan penyaringnya karena itu harus menjadi GABUNGAN TEPAT dari kedua penyaring
-- lama — bukan "tanpa penyaring".
--
-- Penyaring lama:   GROUPPANEL_1 = '002'   (grid PA)
--                   GROUPPANEL_1 <> '002'  (grid NONMBU)
--
-- Gabungannya persis `GROUPPANEL_1 IS NOT NULL`, dan itu berlaku di KEDUA dialek:
--
--   nilai        Oracle lama        Oracle baru   PostgreSQL lama      PostgreSQL baru
--   ---------    ----------------   -----------   ------------------   ---------------
--   '002'        masuk grid PA      masuk         masuk grid PA        masuk
--   nilai lain   masuk grid NONMBU  masuk         masuk grid NONMBU    masuk
--   ''           '' IS NULL →       tidak masuk   '' <> '002' TRUE →   masuk
--                tidak masuk di                   masuk grid NONMBU
--                kedua grid
--   NULL         UNKNOWN di         tidak masuk   UNKNOWN di kedua     tidak masuk
--                kedua grid                       grid
--
-- Menghapus penyaringnya sama sekali akan MENAMBAH baris yang tidak pernah terlihat di Pega
-- — berkas tanpa Group Panel — dan penambahan itu tidak diputuskan siapa pun.
--
-- Perhatikan kolom GROUPPANEL_1 tetap dipilih: ia yang menjadi kolom "Jenis Klaim" di layar,
-- yang kini memikul pembedaan yang dulu dipikul "tabel yang mana". Penerjemahannya
-- dikerjakan Go, bukan SQL — lihat scanWorkItem.
--
-- Bind: :1 offset · :2 jumlah baris
--
-- ============================================================================
-- SUMBER BARU (2026-10-08)
-- ============================================================================
--
-- Tabel objek kerja Pega tidak dipakai lagi (keputusan Work Owner 2026-10-08). Kueri ini
-- kini DIGERAKKAN tabel penugasan (`PC_ASSIGN_WORKLIST`, disaring `PXREFOBJECTCLASS`), dan
-- kolom berkasnya diambil berlapis — yang pertama terisi menang:
--
--   kolom Pega         1. T_CLAIMLIST_ADMIN k   2. T_CLAIM_RECIVEDCLAIM r   3. T_CLAIM_PNC c
--   ------------------ ------------------------ --------------------------- ----------------
--   PYID               a.PXREFOBJECTINSNAME (2.149/2.149 sama), lalu k.PYID
--   POLICYNO           k.POLICYNO               r.NOPOLIS                   c.NOPOLIS
--   PNCCASEID          k.PNCCASEID (kosong)     r.NOKLAIM tanpa awalan       cr (RCVID)
--   QQNAME             r.NAMATERTANGGUNG        c.QQNAME                    k.QQNAME
--   DATEOFLOSS_1       k.DATEOFLOSS_1           r.DOL                       c.DATEOFLOSS
--   GROUPPANEL_1       k.GROUPPANEL_1           r.GROUPPANEL                c.GROUPPANEL
--   PXCREATEDATETIME   k.PXCREATEDATETIME       r.TANGGALINPUTDOKUMEN (sama sampai detik)
--
-- `T_CLAIMLIST_ADMIN` hanya memuat 143 berkas penerimaan (yang tugasnya di antrean Admin),
-- sehingga sebagian besar baris dipasok tabel cermin `T_CLAIM_RECIVEDCLAIM` (2.880 baris).
-- `r.NOKLAIM` berisi kunci klaim BERPREFIX (= awalan kelas + `PNCCASEID` pada seluruh 1.477
-- baris yang terisi keduanya); klaim yang tertaut lewat `T_CLAIM_PNC.RCVID` (`cr`) dipakai
-- bila `NOKLAIM` kosong, HANYA bila nomor RCV itu tertaut ke TEPAT SATU klaim — supaya
-- gabungan tidak menggandakan baris dan kolomnya tidak tercampur dari dua klaim.
-- `QQNAME` mendahulukan tabel cermin karena `k.QQNAME` berbeda dari nama tertanggung Pega
-- pada 101 dari 143 berkas penerimaan.
--
-- Akibat yang terukur (Oracle dev 2026-10-08): versi lama 2.149 baris, versi baru 2.026.
-- 2.007 baris lama tetap tampil; 142 hilang karena Group Panel-nya tidak tercatat di tabel
-- mana pun yang masih dipakai, dan 19 baris baru muncul karena Group Panel-nya kini terbaca
-- padahal kolom Pega-nya kosong. Kecocokan kolom pada baris lama: Group Panel 2.003 (4
-- berbeda), polis 1.966 (5 berbeda), nama tertanggung 1.383 (25 berbeda, sisanya kosong),
-- tanggal kejadian 1.855, nomor klaim PNC 1.412 (3 berbeda).
--
-- Penyaring gabungan kedua grid tetap `IS NOT NULL` — kini atas nilai berlapis di atas.
SELECT a.PXREFOBJECTKEY                                      AS REFERENCE,
       COALESCE(a.PXREFOBJECTINSNAME, k.PYID)                AS CASE_ID,
       COALESCE(k.POLICYNO, r.NOPOLIS, c.NOPOLIS)            AS POLICY_NUMBER,
       COALESCE(k.PNCCASEID,
                REPLACE(COALESCE(r.NOKLAIM, cr.CLAIMID),
                        'ASM-FW-GCNMFW-WORK ', ''))          AS CLAIM_NUMBER,
       COALESCE(r.NAMATERTANGGUNG, c.QQNAME, k.QQNAME)       AS INSURED_NAME,
       COALESCE(k.DATEOFLOSS_1,
                CAST(r.DOL AS TIMESTAMP),
                CAST(c.DATEOFLOSS AS TIMESTAMP))             AS LOSS_DATE,
       COALESCE(k.GROUPPANEL_1, r.GROUPPANEL, c.GROUPPANEL) AS GROUP_PANEL,
       r.NAMAPELAPOR                    AS SENDER_NAME,
       r.TANGGALTERIMADOKUMEN           AS DOCUMENT_RECEIVED_DATE,
       CAST(NULL AS VARCHAR2(50))       AS DOCUMENT_SHEET_COUNT,
       COALESCE(k.PXCREATEDATETIME, r.TANGGALINPUTDOKUMEN)  AS INBOX_ENTRY_AT,
       CAST(NULL AS VARCHAR2(4000))     AS ANALYST_NOTE,
       CAST(NULL AS VARCHAR2(10))       AS TRACK,
       CAST(NULL AS VARCHAR2(100))      AS TRACK_STATUS,
       CAST(NULL AS VARCHAR2(100))      AS LETTER_PRINTED_AT,
       CAST(NULL AS VARCHAR2(100))      AS CLAIM_AGE,
       CAST(NULL AS VARCHAR2(100))      AS EXPIRY_STATUS,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       LEFT JOIN POOLDATA.T_CLAIMLIST_ADMIN k
              ON k.PZINSKEY = a.PXREFOBJECTKEY
             AND k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-ReceiveDocument'
       LEFT JOIN POOLDATA.T_CLAIM_RECIVEDCLAIM r
              ON r.CLAIMID = a.PXREFOBJECTKEY
       LEFT JOIN (SELECT c1.RCVID, MIN(c1.CLAIMID) CLAIMID
                    FROM POOLDATA.T_CLAIM_PNC c1
                   WHERE c1.RCVID IS NOT NULL
                   GROUP BY c1.RCVID
                  HAVING COUNT(*) = 1) cr
              ON cr.RCVID = a.PXREFOBJECTINSNAME
       LEFT JOIN POOLDATA.T_CLAIM_PNC c
              ON c.CLAIMID = COALESCE(r.NOKLAIM, cr.CLAIMID)
 WHERE a.PXREFOBJECTCLASS = 'ASM-FW-GCNMFW-Work-ReceiveDocument'
   AND COALESCE(k.GROUPPANEL_1, r.GROUPPANEL, c.GROUPPANEL) IS NOT NULL
 ORDER BY COALESCE(k.PXCREATEDATETIME, r.TANGGALINPUTDOKUMEN) DESC, a.PXREFOBJECTINSNAME
OFFSET :1 ROWS FETCH NEXT :2 ROWS ONLY

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
--
-- SUMBER (2026-10-08): `POOLDATA.TC_PNC_PUCL` — tabel datar antrean RCL/PUCL, pengganti
-- gabungan `PC_ASM_FW_GCNMFW_WORK` + `PC_ASSIGN_WORKBASKET` ("Perubahan nama tabel untuk
-- Inbox.xlsx", kolom E). Nama kolomnya ditetapkan Work Owner; pemetaannya ada di
-- `docs/ddl/tc_pnc_pucl.sql` §1. Satu baris per klaim, sehingga gabung ke antrean hilang:
-- nama antreannya dibawa `ASSIGNED_OPERATOR_ID`.
--
-- REFERENCE tetap kunci teknis berprefix (`PZINSKEY`), karena tombol rincian memakainya.
-- `TC_PNC_PUCL.CLAIMID` hanya nomor case, jadi kunci berprefixnya DIBACA dari `T_CLAIM_PNC`
-- (`c.CLAIMNO = p.CLAIMID`) — bukan dirangkai sendiri, yang gagal diam-diam untuk klaim
-- ber-nomor `PNCN.YY.xxxx` (§3a DDL). Bila pasangannya belum ada, nomor case yang dipakai.
-- Ia SUBKUERI, bukan gabung: `T_CLAIM_PNC` tidak punya kunci utama, dan diukur 2026-10-08
-- satu `CLAIMNO` muncul lebih dari sekali — gabung biasa akan menggandakan baris antrean.
--
-- Dua kolom yang dulu selalu kosong di tab ini kini terisi dari tabel yang sama:
-- LOSS_DATE (`DATE_OF_LOSS`) dan GROUP_PANEL (`GROUPPANEL`).
--
-- `STATUS_WORK` boleh NULL di tabel datar; baris seperti itu TIDAK dibuang — sama dengan
-- perlakuan modul inboxrclpucl atas tabel yang sama.
SELECT COALESCE((SELECT MIN(c.CLAIMID)
                   FROM POOLDATA.T_CLAIM_PNC c
                  WHERE c.CLAIMNO = p.CLAIMID),
                p.CLAIMID)              AS REFERENCE,
       p.CLAIMID                        AS CASE_ID,
       p.POLICY_NO                      AS POLICY_NUMBER,
       CAST(NULL AS VARCHAR2(100))      AS CLAIM_NUMBER,
       p.QQ_NAME                        AS INSURED_NAME,
       p.DATE_OF_LOSS                   AS LOSS_DATE,
       p.GROUPPANEL                     AS GROUP_PANEL,
       CAST(NULL AS VARCHAR2(255))      AS SENDER_NAME,
       CAST(NULL AS VARCHAR2(100))      AS DOCUMENT_RECEIVED_DATE,
       CAST(NULL AS VARCHAR2(50))       AS DOCUMENT_SHEET_COUNT,
       p.TGL_CREATE_PUCL                AS INBOX_ENTRY_AT,
       p.KOMENTAR_ANALISATOR            AS ANALYST_NOTE,
       CASE
           WHEN p.RCL_PUCL = '1' THEN 'RCL'
           WHEN p.RCL_PUCL = '2' THEN 'PUCL'
       END                              AS TRACK,
       p.STATUS_KLAIM                   AS TRACK_STATUS,
       p.TGL_CETAK_DOKUMEN_PUCL         AS LETTER_PRINTED_AT,
       p.LAMA_KLAIM                     AS CLAIM_AGE,
       p.STATUS_CASE                    AS EXPIRY_STATUS,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM POOLDATA.TC_PNC_PUCL p
 WHERE (p.STATUS_WORK IS NULL OR p.STATUS_WORK <> :1)
   AND p.ASSIGNED_OPERATOR_ID = :2
 ORDER BY p.TGL_CREATE_PUCL DESC, p.CLAIMID
OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY

-- name: detail_receive_document
-- LAYAR KERJA penerimaan dokumen — flow action `InputReceiveDocument`, untuk SATU berkas.
--
-- Ia yang di Pega terbuka ketika nomor case pada grid Receive diklik: `runActivity`
-- `SetAssignmentInboxReceive_act` dengan `kunci = .pzInsKey`, lalu Open Assignment.
--
-- ============================================================================
-- PEMETAAN KOLOM — judul layar -> properti Pega -> kolom sebenarnya
-- ============================================================================
--
-- Judulnya dari `Section/InputReceiveDocument_sect.xml`; propertinya dari `pyValue` sel yang
-- sama; kolomnya dari pernyataan `update` pada
-- `Database/PROCINSERTDATARECIVEDKLAIM.prc` untuk tabel cermin, dan dari kueri Pega yang
-- membaca kelas yang sama untuk objek kerja.
--
--   judul layar                       properti Pega                          kolom
--   --------------------------------- -------------------------------------- ---------------
--   Tanggal Input Dokumen             .pxCreateDateTime                      w.PXCREATEDATETIME
--   Tanggal Terima Dokumen            .ReceiveDocument.ReceivedDate          r.TANGGALTERIMADOKUMEN
--   Nama Pengirim / Pelapor Dokumen   .ReceiveDocument.Sender                r.NAMAPELAPOR
--   Email Pengirim                    .ReceiveDocument.EmailPengirim         r.EMAILPENGIRIM
--   No. HP Pengirim                   .ReceiveDocument.TelpPengirim          r.TLPPENGIRIM
--   Nama Kurir ASM                    .ReceiveDocument.Kurir                 r.NAMAKURIRASM
--   Nama Tertanggung                  .ReceiveDocument.QQName                w.QQNAME
--   Nomor Polis                       .ReceiveDocument.PolicyNo              w.POLICYNO
--   Tanggal Kejadian                  .ReceiveDocument.TglKejadian           w.DATEOFLOSS_1
--   No. Referensi/Placing Slip        .Policy.BookNo                         r.NOREFERENSI
--   Email Tertanggung                 .ReceiveDocument.EmailLOD              r.EMAILTERTANGGUNG
--   Lokasi Kejadian                   .ReceiveDocument.LokasiKejadian        r.LOKASIKEJADIAN
--   SIM Pengendara                    .ReceiveDocument.SIM                   r.SIMPENGENDARA
--   Kronologis Kejadian               .ReceiveDocument.KronologisKejadian    r.KRONOLOGIKEJADIAN
--   Rincian Kerusakan                 .ReceiveDocument.RincianKerusakan      r.RINCIANKERUSAKAN
--   Alasan Belum Transfer             .ReceiveDocument.Keterangan            r.ALASANBLMTRANSFER
--   Subjek Email                      .ReceiveDocument.SubjectEmail          r.SUBJECTEMAIL
--   Keterangan Belum Registrasi       .ReceiveDocument.NotRegistNote         w.NOTREGISTNOTE_1
--
-- Enam belas isian layar TIDAK ada di sini karena tidak punya kolom mana pun — Polis Leader,
-- Nama Bisnis, Estimasi Kerugian, No Ref Broker, Source Of Reports, Total Jumlah Dokumen,
-- dan sebelas isian blok Data Pelapor. Seluruhnya bertanda Blocked di
-- inboxmanagerreceivepucl.DocumentFieldGroups dan tetap DIGAMBAR beserta alasannya, bukan
-- dihilangkan.
--
-- ============================================================================
-- DUA HAL YANG MEMBUAT KUERI INI GAGAL TANPA SATU PUN GALAT BILA DILUPAKAN
-- ============================================================================
--
-- 1. PENYARING PXOBJCLASS. Tabel objek kerja menampung DUA kelas, dan keduanya punya PYID,
--    POLICYNO, dan QQNAME. Tanpa penyaring ini, kunci milik sebuah klaim akan membuka layar
--    kerja penerimaan dokumen yang isinya klaim — tanpa satu pun tanda bahwa itu keliru.
--
-- 2. `LEFT JOIN` ke tabel cermin. Tabel itu TIDAK PERNAH DIBACA sistem lama, sehingga
--    kelengkapan isinya belum terverifikasi. Gabungan dalam akan membuat berkas yang tidak
--    punya pasangan di sana dinyatakan TIDAK ADA — padahal ia ada, dan justru itulah berkas
--    yang paling perlu dilihat.
--
-- TIDAK ada gabungan ke tabel penugasan di sini, dan itu disengaja. Grid menyaringnya karena
-- grid memang daftar PEKERJAAN yang menunggu; layar kerja membuka SATU berkas yang kuncinya
-- sudah di tangan. Menambahkan gabungan itu akan membuat berkas yang penugasannya baru saja
-- selesai gagal dibuka di tengah pengerjaan, dengan pesan "tidak ditemukan" yang menyesatkan.
--
-- Bind: :1 kunci berkas (PZINSKEY)
--
-- ============================================================================
-- SUMBER BARU (2026-10-08)
-- ============================================================================
--
-- Objek kerja Pega tidak dipakai lagi. Berkasnya kini dibaca dari tabel cermin
-- `T_CLAIM_RECIVEDCLAIM` (r) — yang memuat SELURUH 143 berkas penerimaan di
-- `T_CLAIMLIST_ADMIN` dan 2.802 dari 2.816 objek kerja penerimaan lama — dengan
-- `T_CLAIMLIST_ADMIN` (k) didahulukan bila ada, dan klaim tertautnya di `T_CLAIM_PNC` (c).
-- Pemetaan kolomnya sama dengan list_receive; tambahan khusus layar kerja:
--
--   PYSTATUSWORK      HANYA k.PYSTATUSWORK. `r.PYSTATUSWORK` adalah status saat berkas
--                     DITERIMA dan tidak diperbarui: ia berbeda dari status Pega pada
--                     ±360 dari 2.149 berkas yang menunggu. Status yang salah lebih buruk
--                     daripada kosong, sehingga di luar 143 berkas antrean Admin ia KOSONG.
--   NOTREGISTNOTE_1   k.NOTREGISTNOTE_1, lalu r.KETERANGANBLMREGIST (2.812/2.816 sama).
--
-- PENYARING KELAS. Tabel cermin tidak punya PXOBJCLASS dan juga memuat beberapa kunci
-- KLAIM (awalan `PNC-`/`CN-`) serta berkas sistem baru (`RCVN…`). Yang diterima hanya
-- kunci yang tercatat sebagai kelas 'ASM-FW-GCNMFW-Work-ReceiveDocument' di
-- `T_CLAIMLIST_ADMIN`, atau berawalan kunci berkas penerimaan Pega — sehingga kunci milik
-- sebuah klaim tetap tidak membuka layar ini.
--
-- Akibat: 14 objek kerja penerimaan lama yang tidak punya baris di tabel cermin kini
-- "tidak ditemukan".
SELECT r.CLAIMID                        AS REFERENCE,
       COALESCE(k.PYID,
                REPLACE(r.CLAIMID, 'ASM-FW-GCNMFW-WORK ', ''))        AS CASE_ID,
       COALESCE(k.PNCCASEID,
                REPLACE(COALESCE(r.NOKLAIM, cr.CLAIMID),
                        'ASM-FW-GCNMFW-WORK ', ''))                   AS CLAIM_NUMBER,
       COALESCE(k.GROUPPANEL_1, r.GROUPPANEL, c.GROUPPANEL)          AS GROUP_PANEL,
       k.PYSTATUSWORK                   AS WORK_STATUS,
       COALESCE(k.PXCREATEDATETIME, r.TANGGALINPUTDOKUMEN)           AS CREATED_AT,
       r.TANGGALTERIMADOKUMEN           AS RECEIVED_AT,
       r.NAMAPELAPOR                    AS SENDER_NAME,
       r.EMAILPENGIRIM                  AS SENDER_EMAIL,
       r.TLPPENGIRIM                    AS SENDER_PHONE,
       r.NAMAKURIRASM                   AS COURIER_NAME,
       COALESCE(r.NAMATERTANGGUNG, c.QQNAME, k.QQNAME)               AS INSURED_NAME,
       COALESCE(k.POLICYNO, r.NOPOLIS, c.NOPOLIS)                    AS POLICY_NUMBER,
       COALESCE(k.DATEOFLOSS_1,
                CAST(r.DOL AS TIMESTAMP),
                CAST(c.DATEOFLOSS AS TIMESTAMP))                     AS LOSS_DATE,
       r.NOREFERENSI                    AS REFERENCE_NUMBER,
       r.EMAILTERTANGGUNG               AS INSURED_EMAIL,
       r.LOKASIKEJADIAN                 AS LOSS_LOCATION,
       r.SIMPENGENDARA                  AS DRIVER_LICENCE,
       r.KRONOLOGIKEJADIAN              AS CHRONOLOGY,
       r.RINCIANKERUSAKAN               AS DAMAGE_DETAIL,
       r.ALASANBLMTRANSFER              AS TRANSFER_REASON,
       r.SUBJECTEMAIL                   AS EMAIL_SUBJECT,
       COALESCE(k.NOTREGISTNOTE_1, r.KETERANGANBLMREGIST)           AS NOT_REGISTERED_NOTE
  FROM POOLDATA.T_CLAIM_RECIVEDCLAIM r
       LEFT JOIN POOLDATA.T_CLAIMLIST_ADMIN k
              ON k.PZINSKEY = r.CLAIMID
             AND k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-ReceiveDocument'
       LEFT JOIN (SELECT c1.RCVID, MIN(c1.CLAIMID) CLAIMID
                    FROM POOLDATA.T_CLAIM_PNC c1
                   WHERE c1.RCVID IS NOT NULL
                   GROUP BY c1.RCVID
                  HAVING COUNT(*) = 1) cr
              ON cr.RCVID = REPLACE(r.CLAIMID, 'ASM-FW-GCNMFW-WORK ', '')
       LEFT JOIN POOLDATA.T_CLAIM_PNC c
              ON c.CLAIMID = COALESCE(r.NOKLAIM, cr.CLAIMID)
 WHERE r.CLAIMID = :1
   AND (k.PZINSKEY IS NOT NULL OR r.CLAIMID LIKE 'ASM-FW-GCNMFW-WORK RCV-%')

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
--
-- SUMBER BARU (2026-10-08): tabel objek kerja Pega diganti `T_CLAIMLIST_ADMIN` dan
-- `T_CLAIM_PNC`; keempat tabel yang dibaca tab Receive diperiksa sekaligus.
SELECT COUNT(*) AS PROBE
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       LEFT JOIN POOLDATA.T_CLAIMLIST_ADMIN k
              ON k.PZINSKEY = a.PXREFOBJECTKEY
             AND k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-ReceiveDocument'
       LEFT JOIN POOLDATA.T_CLAIM_RECIVEDCLAIM r
              ON r.CLAIMID = a.PXREFOBJECTKEY
       LEFT JOIN POOLDATA.T_CLAIM_PNC c
              ON c.CLAIMID = r.NOKLAIM
 WHERE 1 = 0

-- name: check_rclpucl
-- Memastikan tabel antrean bersama terbaca. Ia terpisah dari check_receive karena tabelnya
-- memang berbeda, dan galat yang menyebut tabel yang salah menyesatkan orang yang
-- memperbaikinya.
SELECT COUNT(*) AS PROBE
  FROM POOLDATA.TC_PNC_PUCL p
       LEFT JOIN POOLDATA.T_CLAIM_PNC c
              ON c.CLAIMNO = p.CLAIMID
 WHERE 1 = 0
