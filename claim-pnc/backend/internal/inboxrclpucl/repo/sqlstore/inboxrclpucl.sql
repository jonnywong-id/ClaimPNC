-- Kueri modul Inbox RCL/PUCL (`MENU_ID 61`, pengganti `Harness/RCLPUCL_Harness`).
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
-- SATU ANTREAN, TIGA PARTISI
-- ============================================================================
--
-- Ketiga kueri daftar membaca tabel yang SAMA dan antrean bersama yang SAMA. Yang
-- membedakan hanyalah tiga penyaring, dan ketiganya menyangkut perjalanan surat PUCL:
--
--   list_cetak_surat            TANGGALCETAKDOKUMENPUCL_1 IS NULL     surat belum dicetak
--                               STATUSCASE_1 = :status
--   list_kelengkapan_dokumen    TANGGALCETAKDOKUMENPUCL_1 IS NOT NULL surat sudah dicetak
--                               PUCLAPPROVE_1 <> :disetujui
--                               MSIG_1 IS NULL
--   list_klaim_msig             sama seperti di atas, tetapi MSIG_1 = :msig
--
-- Ketiganya ditambah PYSTATUSWORK <> :selesai dan gabungan ke antrean bersama `RCLPUCL`.
--
-- Penyaringnya diambil dari `pyFilters` ketiga Report Definition, dan dipastikan ulang
-- terhadap SQL hasil generate Pega sendiri di `RDB List/ReminderPUCL-SQL.xml` — yang memuat
-- penyaring list_kelengkapan_dokumen kata demi kata, termasuk literal 'RCLPUCL'.
--
-- ============================================================================
-- PEMETAAN KOLOM — properti Pega -> kolom sebenarnya -> alias di sini
-- ============================================================================
--
-- Diambil dari SEL GRID ketiga section (bukan dari pyListFields Report Definition, yang
-- mengambil 17–25 isian sementara yang digambar hanya sembilan), dan nama kolomnya
-- dipastikan lewat `RDB List/ReminderPUCL-SQL.xml` dan `RDB List/GetReminderPUCL-SQL.xml`.
--
--   judul kolom          properti Pega                                  kolom          alias
--   -------------------- --------------------------------------------- -------------- -----------------
--   (tidak digambar)     .pzInsKey                                      PZINSKEY       REFERENCE
--   Nomor Case           .pyID                                          PYID           CASE_ID
--   No Polis             .Policy.PolicyNo                               POLICYNO       POLICY_NUMBER
--   Nama Tertanggung     .Policy.QQName                                 QQNAME         INSURED_NAME
--   Tanggal Masuk Inbox  .ClaimData.PUCLStatus.TanggalKirimPUCL         TANGGALKIRIMPUCL_1        INBOX_ENTRY_AT
--   Deskripsi Analyst    .ClaimData.PUCLStatus.KomentarAnalisator       KOMENTARANALISATOR_1      ANALYST_NOTE
--   Status RCL/PUCL      .ClaimData.PUCLStatus.RCL_PUCL                 RCL_PUCL_1                TRACK_CODE
--   Tanggal Cetak Surat  .ClaimData.PUCLStatus.TanggalCetakDokumenPUCL  TANGGALCETAKDOKUMENPUCL_1 LETTER_PRINTED_AT
--   Lama Klaim           .ClaimData.PUCLStatus.LamaKlaim                LAMAKLAIM_1               CLAIM_AGE
--   Status Kadaluarsa    .ClaimData.PUCLStatus.StatusKlaim              STATUSKLAIM_1             EXPIRY_STATUS
--
-- DUA BARIS TERAKHIR ADALAH TEMPAT PALING MUDAH SALAH DI SELURUH BERKAS INI.
--
-- Layar Inbox Manager Receive / PUCL memakai DUA judul yang sama untuk kolom yang BERBEDA:
--
--   judul                layar ini        Inbox Manager Receive / PUCL
--   -------------------- ---------------- ----------------------------
--   Status RCL/PUCL      RCL_PUCL_1       STATUSKLAIM_1
--   Status Kadaluarsa    STATUSKLAIM_1    STATUSCASE_1
--
-- Keduanya diverifikasi dari sel grid section masing-masing. Menyalin pemetaan satu layar ke
-- layar lain akan menampilkan kolom yang salah TANPA satu pun galat.
--
-- Perhatikan pula `STATUSCASE_1`: ia MENYARING list_cetak_surat tetapi TIDAK digambar satu
-- sel pun di layar ini. Itu perilaku sistem lama apa adanya.
--
-- Alias Pega yang TIDAK dibawa (`D-19`), karena tidak satu pun menyatakan isinya:
--
--   KOMENTARANALISATOR_1  dialiaskan "LOGSEARCH" di ReminderPUCL, "NoteKomite" di
--                         GetReminderPUCL, "CloseClaimNote" di GetDataPUCLRCLForDailyReport
--   LAMAKLAIM_1           dialiaskan "MODUL" di ReminderPUCL, "LOGSEEN" di GetReminderPUCL
--   STATUSKLAIM_1         dialiaskan "STS_EMAIL" di ReminderPUCL, "StsAcceptance" di GetReminderPUCL
--   QQNAME                dialiaskan "CABANG" di ReminderPUCL, "NewTelpTertanggung" di
--                         GetReminderPUCL, "NamaSurveyor" di GetDataPUCLRCLForDailyReport
--   POLICYNO              dialiaskan "ProdKe" di GetDataPUCLRCLForDailyReport
--   TANGGALKIRIMPUCL_1    dialiaskan "City" di GetDataPUCLRCLForDailyReport
--   STATUSCASE_1          dialiaskan "ClaimData.PUCLStatus.Stat25L" — terpotong batas alias Oracle
--
-- Satu kolom yang sama dialiaskan "CABANG" di satu rule dan "NamaSurveyor" di rule lain,
-- padahal isinya nama tertanggung. Itu utang teknis §4.2 apa adanya.
--
-- ============================================================================
-- KODE JALUR TIDAK DITERJEMAHKAN DI SINI
-- ============================================================================
--
-- `RCL_PUCL_1` dikembalikan MENTAH sebagai TRACK_CODE, dan penerjemahannya menjadi "RCL"
-- atau "PUCL" dikerjakan `inboxrclpucl.TrackOf` di lapisan domain.
--
-- Ini BERBEDA dari modul Inbox Manager Receive / PUCL, yang menuliskan `CASE` penerjemah di
-- dalam SQL-nya. Yang dipakai di sini adalah pola yang sama dengan `ClaimTypeOf` pada modul
-- itu, dan alasannya sama: penyimpanan SQL dan penyimpanan memori wajib menghasilkan teks
-- yang sama persis, dan dua penerjemah di dua tempat dapat menyimpang tanpa ketahuan.
--
-- Hasilnya identik dengan `CASE` tanpa `ELSE` di sistem lama
-- (`GetReminderPUCL-SQL.xml:7-9`): kode di luar '1' dan '2' menghasilkan teks kosong.
--
-- ============================================================================
-- YANG BERUBAH DARI SISTEM LAMA, DAN KENAPA
-- ============================================================================
--
-- 1. PAGINASI DIKERJAKAN BASIS DATA.
--    Ketiga grid memotong hasilnya di `pyMaxRecords` 500 SETELAH seluruh barisnya ditarik,
--    lalu menomori halamannya di klipboard. Di sini halamannya dipotong dengan
--    `OFFSET … FETCH NEXT … ROWS ONLY` sebelum baris meninggalkan basis data — didukung
--    Oracle 12c+ dan PostgreSQL (`09-DATABASE-STRATEGY.md` §3.3). Ukuran halamannya tetap
--    50, sama dengan `<pyPageSize>50</pyPageSize>` ketiga section.
--
-- 2. JUMLAH SELURUH BARIS DIHITUNG `COUNT(*) OVER ()`.
--    Satu perjalanan, bukan dua. Fungsi jendela dihitung SEBELUM `OFFSET … FETCH` dipakai,
--    sehingga angkanya jumlah seluruhnya — bukan jumlah baris di halaman ini.
--
-- 3. NILAI SELALU LEWAT PARAMETER BINDING.
--    `GetDataPUCLRCLForDailyReport-SQL.xml` menyisipkan `{TempRCLPUCLReport.AlasanKlaim}`
--    dan `{TempRCLPUCLReport.NoteKasir}` — kedua isian tanggal yang DIKETIK PENGGUNA —
--    langsung ke teks SQL-nya. Larangan perangkaian (`08-TECHNICAL-STRATEGY.md` §4.3) tidak
--    dikecualikan oleh keputusan mana pun: yang direplikasi adalah perilaku bisnis, bukan
--    celah injeksi.
--
-- 4. `TRUNC` PADA KOLOM TANGGAL DIGANTI RENTANG SETENGAH TERBUKA.
--    Kueri lama menulis `trunc(TANGGALKIRIMPUCL_1) >= … AND trunc(…) <= …`. `TRUNC` pada
--    kolom dilarang (`09-DATABASE-STRATEGY.md` §4) dan mematikan index. Penggantinya
--    `>= awal AND < akhir + 1 hari`, yang MEMILIH BARIS YANG SAMA PERSIS dan tetap dapat
--    memakai index. Ia pula portabel ke PostgreSQL, sementara `TRUNC(date)` tidak.
--
-- 5. GABUNGAN DITULIS SEBAGAI `JOIN`, BUKAN DAFTAR TABEL BERKOMA.
--    `ReminderPUCL-SQL.xml` sudah memakai `INNER JOIN`. Yang diubah hanyalah tempat syarat
--    penyaring ditulis, bukan isinya.
--
-- ============================================================================
-- YANG SENGAJA TIDAK BERUBAH — termasuk yang tampak seperti cacat
-- ============================================================================
--
-- * URUTAN mengikuti Report Definition apa adanya: `PXCREATEDATETIME DESC, PYID DESC`,
--   terbaca dari `ReminderPUCL-SQL.xml` sebagai `ORDER BY 5 DESC, 7 DESC` (kolom ke-5 dan
--   ke-7 pada daftar pilihnya).
--
--   Yang harus disadari: `PXCREATEDATETIME` TIDAK digambar di layar ini. Yang digambar
--   sebagai "Tanggal Masuk Inbox" adalah `TANGGALKIRIMPUCL_1`, dan keduanya dapat terpaut
--   berbulan-bulan — sebuah klaim lahir jauh sebelum ia masuk antrean RCL/PUCL. Akibatnya
--   tabel dapat TERBACA tidak urut oleh penggunanya.
--
--   Ia tetap tidak diubah. Mengganti kunci urut mengubah baris mana yang ada di halaman
--   pertama, dan itu selisih yang tidak diputuskan siapa pun (`P-5`). Ia dinyatakan ke
--   pengguna lewat PlannedDifferences, bukan diperbaiki sepihak.
--
-- * `INNER JOIN` ke tabel antrean bersama, bukan `EXISTS`. Report Definition-nya memakai
--   gabungan dalam, sehingga objek kerja yang punya DUA penugasan terbuka di antrean yang
--   sama muncul DUA KALI. Itu perilaku sistem lama apa adanya, dan menggantinya dengan
--   `EXISTS` akan mengubah jumlah baris yang terlihat pengguna tanpa satu pun keputusan
--   yang mendasarinya.
--
-- * `PUCLAPPROVE_1 <> :disetujui` TIDAK MENANGKAP NULL, dan itu dibiarkan.
--   `NULL <> '1'` menghasilkan UNKNOWN — bukan TRUE — di Oracle maupun PostgreSQL, sehingga
--   klaim yang penanda persetujuannya belum pernah diisi TIDAK muncul di tab Kelengkapan
--   Dokumen maupun Klaim MSIG. Kolomnya hanya punya DUA nilai berbeda di produksi
--   (`docs/kolom-t-claimlist-admin.md` §B.3), sehingga jumlah baris yang terdampak bisa
--   besar.
--
--   Memperbaikinya menjadi `(… IS NULL OR … <> :disetujui)` akan MENAMBAH baris yang di
--   Pega tidak pernah terlihat. Itu perubahan perilaku pada layar yang sedang diuji
--   kesetaraannya, dan bukan wewenang berkas ini. Ia dicatat sebagai pertanyaan terbuka.
--
-- * Pembanding `PUCLAPPROVE_1` diikat sebagai TEKS, mengikuti
--   `RDB List/CountKlaimPUCL-SQL.xml` yang menulis `<> '1'`. `ReminderPUCL-SQL.xml` menulis
--   `<> 1` tanpa kutip pada kolom yang sama — dua rule Pega yang tidak sepakat tentang tipe
--   kolomnya sendiri. Yang dipilih bentuk bertanda kutip karena DDL-nya tidak tersedia
--   (`R-08`) dan seluruh kolom ber-akhiran `_1` lain di tabel ini dibaca sebagai teks.
--
-- CATATAN PENANDA BIND. Berkas ini memakai gaya Oracle `:n`, sama seperti seluruh modul lain
-- di aplikasi ini. Ia BELUM portabel ke PostgreSQL yang memakai `$n`; itu utang yang sudah
-- ada sebelum modul ini dan berlaku untuk seluruh berkas .sql di sini.

-- name: list_cetak_surat
-- Tab "Cetak Surat" — klaim RCL/PUCL yang suratnya BELUM dicetak.
-- — Report Definition/InboxPUCL_RD-RD.xml, pyFilterLogic "A AND B AND C AND D"
--
-- LETTER_PRINTED_AT pada kueri ini SELALU kosong: penyaringnya `IS NULL`. Kolomnya tetap
-- dipilih, bukan diganti NULL tetap, supaya ketiga kueri punya bentuk yang sama persis dan
-- satu pemindai Go dapat melayani ketiganya.
--
-- Bind: :1 kelas objek kerja · :2 akun antrean bersama · :3 status kerja yang dikecualikan
--       :4 nilai STATUSCASE_1 yang diterima · :5 offset · :6 jumlah baris
SELECT w.PZINSKEY                       AS REFERENCE,
       w.PYID                           AS CASE_ID,
       w.POLICYNO                       AS POLICY_NUMBER,
       w.QQNAME                         AS INSURED_NAME,
       w.TANGGALKIRIMPUCL_1             AS INBOX_ENTRY_AT,
       w.KOMENTARANALISATOR_1           AS ANALYST_NOTE,
       w.RCL_PUCL_1                     AS TRACK_CODE,
       w.TANGGALCETAKDOKUMENPUCL_1      AS LETTER_PRINTED_AT,
       w.LAMAKLAIM_1                    AS CLAIM_AGE,
       w.STATUSKLAIM_1                  AS EXPIRY_STATUS,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET b
               ON b.PXREFOBJECTKEY = w.PZINSKEY
              AND b.PXOBJCLASS = 'Assign-WorkBasket'
 WHERE w.PXOBJCLASS = :1
   AND b.PXASSIGNEDOPERATORID = :2
   AND w.PYSTATUSWORK <> :3
   AND w.TANGGALCETAKDOKUMENPUCL_1 IS NULL
   AND w.STATUSCASE_1 = :4
 ORDER BY w.PXCREATEDATETIME DESC, w.PYID DESC
OFFSET :5 ROWS FETCH NEXT :6 ROWS ONLY

-- name: list_kelengkapan_dokumen
-- Tab "Kelengkapan Dokumen" — surat sudah dicetak, belum disetujui, bukan jalur MSIG.
-- — Report Definition/InboxPUCLCetakSurat_RD-RD.xml, pyFilterLogic "A AND B AND C AND D AND E"
-- — SQL hasil generate-nya ada utuh di RDB List/ReminderPUCL-SQL.xml
--
-- Bind: :1 kelas objek kerja · :2 akun antrean bersama · :3 status kerja yang dikecualikan
--       :4 nilai PUCLAPPROVE_1 yang dikecualikan · :5 offset · :6 jumlah baris
SELECT w.PZINSKEY                       AS REFERENCE,
       w.PYID                           AS CASE_ID,
       w.POLICYNO                       AS POLICY_NUMBER,
       w.QQNAME                         AS INSURED_NAME,
       w.TANGGALKIRIMPUCL_1             AS INBOX_ENTRY_AT,
       w.KOMENTARANALISATOR_1           AS ANALYST_NOTE,
       w.RCL_PUCL_1                     AS TRACK_CODE,
       w.TANGGALCETAKDOKUMENPUCL_1      AS LETTER_PRINTED_AT,
       w.LAMAKLAIM_1                    AS CLAIM_AGE,
       w.STATUSKLAIM_1                  AS EXPIRY_STATUS,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET b
               ON b.PXREFOBJECTKEY = w.PZINSKEY
              AND b.PXOBJCLASS = 'Assign-WorkBasket'
 WHERE w.PXOBJCLASS = :1
   AND b.PXASSIGNEDOPERATORID = :2
   AND w.PYSTATUSWORK <> :3
   AND w.TANGGALCETAKDOKUMENPUCL_1 IS NOT NULL
   AND w.PUCLAPPROVE_1 <> :4
   AND w.MSIG_1 IS NULL
 ORDER BY w.PXCREATEDATETIME DESC, w.PYID DESC
OFFSET :5 ROWS FETCH NEXT :6 ROWS ONLY

-- name: list_klaim_msig
-- Tab "Klaim MSIG" — sama seperti list_kelengkapan_dokumen, tetapi jalur MSIG.
-- — Report Definition/InboxMISG_RD-RD.xml, pyFilterLogic "A AND B AND C AND D AND E"
--
-- SATU-SATUNYA perbedaan terhadap kueri di atasnya adalah baris MSIG_1: `IS NULL` menjadi
-- `= :5`. Keduanya sengaja TIDAK disatukan menjadi satu kueri berparameter: penyaring yang
-- artinya berbalik menurut nilai bind adalah tempat paling mudah menukar isi dua tab, dan
-- tidak ada apa pun di layar yang menandakannya bila itu terjadi.
--
-- KUERI INI KEMUNGKINAN SELALU MENGEMBALIKAN NOL BARIS. `MSIG_1` tidak muncul di inventaris
-- kolom terisi yang dibaca langsung dari katalog Oracle pada 2026-09-22
-- (`docs/kolom-t-claimlist-admin.md` §B.3), sementara `PUCLAPPROVE_1`, `STATUSCASE_1`,
-- `RCL_PUCL_1`, dan `TANGGALKIRIMPUCL_1` semuanya ada di sana. Artinya kolomnya ada tetapi
-- tampaknya belum pernah diisi.
--
-- Kuerinya tetap dibangun apa adanya — keputusan Work Owner 2026-09-23 — dan kosongnya
-- dinyatakan ke pengguna lewat Tab.Notice, bukan disamarkan. Menunggu pemastian DBA.
--
-- Bind: :1 kelas objek kerja · :2 akun antrean bersama · :3 status kerja yang dikecualikan
--       :4 nilai PUCLAPPROVE_1 yang dikecualikan · :5 penanda jalur MSIG · :6 offset
--       :7 jumlah baris
SELECT w.PZINSKEY                       AS REFERENCE,
       w.PYID                           AS CASE_ID,
       w.POLICYNO                       AS POLICY_NUMBER,
       w.QQNAME                         AS INSURED_NAME,
       w.TANGGALKIRIMPUCL_1             AS INBOX_ENTRY_AT,
       w.KOMENTARANALISATOR_1           AS ANALYST_NOTE,
       w.RCL_PUCL_1                     AS TRACK_CODE,
       w.TANGGALCETAKDOKUMENPUCL_1      AS LETTER_PRINTED_AT,
       w.LAMAKLAIM_1                    AS CLAIM_AGE,
       w.STATUSKLAIM_1                  AS EXPIRY_STATUS,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET b
               ON b.PXREFOBJECTKEY = w.PZINSKEY
              AND b.PXOBJCLASS = 'Assign-WorkBasket'
 WHERE w.PXOBJCLASS = :1
   AND b.PXASSIGNEDOPERATORID = :2
   AND w.PYSTATUSWORK <> :3
   AND w.TANGGALCETAKDOKUMENPUCL_1 IS NOT NULL
   AND w.PUCLAPPROVE_1 <> :4
   AND w.MSIG_1 = :5
 ORDER BY w.PXCREATEDATETIME DESC, w.PYID DESC
OFFSET :6 ROWS FETCH NEXT :7 ROWS ONLY

-- name: daily_report
-- LAPORAN HARIAN RCL/PUCL — keluaran tombol ekspor tab "Cetak Surat".
-- — Activity/ExportCetakSurat_act-Act.xml menjalankan kueri di bawah lalu pxConvertResultsToCSV
-- — RDB List/GetDataPUCLRCLForDailyReport-SQL.xml
--
-- ============================================================================
-- ISI LAPORAN INI TIDAK SAMA DENGAN ISI TABEL DI ATASNYA
-- ============================================================================
--
-- Itu bukan cacat dan bukan salah baca; itu memang kueri yang berbeda. Empat perbedaannya:
--
--   * Ia disaring RENTANG TANGGAL atas TANGGALKIRIMPUCL_1. Grid tidak menyaring tanggal.
--   * Ia TIDAK menyaring TANGGALCETAKDOKUMENPUCL_1 maupun STATUSCASE_1, sehingga memuat
--     klaim yang suratnya SUDAH dicetak — yang di layar ada di tab lain.
--   * Ia TIDAK menyaring PYSTATUSWORK, sehingga memuat klaim yang sudah selesai.
--   * Ia ber-UNION dengan cabang kedua yang mengambil seluruh klaim ber-GROUPPANEL_1 '002'
--     (Personal Accident) pada rentang yang sama, TANPA gabungan antrean bersama sama
--     sekali — sehingga memuat klaim PA yang tidak pernah masuk antrean RCL/PUCL.
--
-- Keputusan Work Owner 2026-09-23: replikasi apa adanya (`P-5`). Ia dinyatakan ke pengguna
-- lewat PlannedDifferences, bukan disamarkan.
--
-- `UNION` dipertahankan, BUKAN `UNION ALL`. Ia membuang baris kembar, dan klaim PA yang
-- berada di antrean RCL/PUCL memenuhi KEDUA cabang sekaligus — dengan `UNION ALL` ia akan
-- muncul dua kali. Itu perbedaan yang terlihat langsung di berkas.
--
-- `ORDER BY` DITAMBAHKAN. Kueri lama tidak punya sama sekali — dapat dibiarkan selama
-- seluruh baris ditarik sekaligus, tetapi membuat satu baris muncul di dua halaman begitu
-- hasilnya dipotong per halaman. Kunci urutnya TANGGALKIRIMPUCL_1, yaitu kolom yang
-- MENYARING laporan ini dan yang digambar sebagai kolom keempatnya — bukan PXCREATEDATETIME,
-- yang tidak dipilih kueri lama dan karena itu tidak tersedia setelah UNION.
--
-- Bind cabang antrean bersama:
--   :1 kelas objek kerja · :2 akun antrean bersama · :3 awal rentang · :4 akhir rentang
-- Bind cabang Personal Accident:
--   :5 kelas objek kerja · :6 kode Group Panel PA · :7 awal rentang · :8 akhir rentang
-- Bind paginasi:
--   :9 offset · :10 jumlah baris
--
-- Kedua rentang diikat TERPISAH meski nilainya sama. Menulis `:3` dua kali akan bergantung
-- pada cara driver menafsirkan penanda berulang, dan itu perbedaan yang tidak terlihat saat
-- membaca kueri.
SELECT r.REFERENCE,
       r.CASE_ID,
       r.POLICY_NUMBER,
       r.INSURED_NAME,
       r.SENT_AT,
       r.ANALYST_NOTE,
       r.LETTER_PRINTED_AT,
       r.TRACK_CODE,
       r.CLAIM_STATUS,
       COUNT(*) OVER () AS TOTAL_ROWS
  FROM (SELECT w.PZINSKEY                  AS REFERENCE,
               w.PYID                      AS CASE_ID,
               w.POLICYNO                  AS POLICY_NUMBER,
               w.QQNAME                    AS INSURED_NAME,
               w.TANGGALKIRIMPUCL_1        AS SENT_AT,
               w.KOMENTARANALISATOR_1      AS ANALYST_NOTE,
               w.TANGGALCETAKDOKUMENPUCL_1 AS LETTER_PRINTED_AT,
               w.RCL_PUCL_1                AS TRACK_CODE,
               w.STATUSCLAIM_1             AS CLAIM_STATUS
          FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
               INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET b
                       ON b.PXREFOBJECTKEY = w.PZINSKEY
                      AND b.PXOBJCLASS = 'Assign-WorkBasket'
         WHERE w.PXOBJCLASS = :1
           AND b.PXASSIGNEDOPERATORID = :2
           AND w.TANGGALKIRIMPUCL_1 >= TO_DATE(:3, 'YYYY-MM-DD')
           AND w.TANGGALKIRIMPUCL_1 < TO_DATE(:4, 'YYYY-MM-DD') + INTERVAL '1' DAY
        UNION
        SELECT w.PZINSKEY                  AS REFERENCE,
               w.PYID                      AS CASE_ID,
               w.POLICYNO                  AS POLICY_NUMBER,
               w.QQNAME                    AS INSURED_NAME,
               w.TANGGALKIRIMPUCL_1        AS SENT_AT,
               w.KOMENTARANALISATOR_1      AS ANALYST_NOTE,
               w.TANGGALCETAKDOKUMENPUCL_1 AS LETTER_PRINTED_AT,
               w.RCL_PUCL_1                AS TRACK_CODE,
               w.STATUSCLAIM_1             AS CLAIM_STATUS
          FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
         WHERE w.PXOBJCLASS = :5
           AND w.GROUPPANEL_1 = :6
           AND w.TANGGALKIRIMPUCL_1 >= TO_DATE(:7, 'YYYY-MM-DD')
           AND w.TANGGALKIRIMPUCL_1 < TO_DATE(:8, 'YYYY-MM-DD') + INTERVAL '1' DAY) r
 ORDER BY r.SENT_AT DESC, r.CASE_ID DESC
OFFSET :9 ROWS FETCH NEXT :10 ROWS ONLY

-- name: check_rclpucl
-- Dipakai perintah `-periksa`: memastikan kedua tabel yang disentuh modul ini terbaca dari
-- koneksi yang dipakai.
--
-- Ia tidak menyentuh satu baris pun — yang diperiksa adalah hak baca dan keberadaan
-- tabelnya, bukan isinya. Keduanya diperiksa sekaligus karena kegagalan yang paling mungkin
-- terjadi bukan "tabel tidak ada" melainkan "hak baca hanya diberikan pada salah satunya".
--
-- `COUNT(*)` dipakai, bukan sebuah kolom, supaya hasilnya SELALU tepat satu baris meski
-- penyaringnya tidak meloloskan apa pun — pemanggil karena itu tidak perlu membedakan
-- "tidak ada baris" dari "gagal dibaca".
SELECT COUNT(*) AS PROBE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET b
               ON b.PXREFOBJECTKEY = w.PZINSKEY
 WHERE 1 = 0

-- name: check_columns
-- Memastikan kelima kolom PUCL yang MENYARING layar ini benar-benar terbaca.
--
-- Ia terpisah dari check_rclpucl dengan sengaja. Tabelnya sama, tetapi yang diuji berbeda:
-- di atas keberadaan TABEL, di sini keberadaan KOLOM. Kolom yang tidak ada menghasilkan
-- galat yang menyebut namanya, dan itulah yang membedakan "modul ini belum dapat dipakai di
-- sini" dari "antreannya memang kosong".
--
-- `MSIG_1` ikut diperiksa justru karena ia yang paling diragukan — lihat catatan pada
-- list_klaim_msig. Bila ia TIDAK ADA sebagai kolom, pemeriksaan ini gagal dan sebabnya
-- terbaca; bila ia ADA tetapi kosong, pemeriksaan ini lolos dan kosongnya adalah jawaban.
SELECT COUNT(w.TANGGALCETAKDOKUMENPUCL_1)
     + COUNT(w.STATUSCASE_1)
     + COUNT(w.PUCLAPPROVE_1)
     + COUNT(w.MSIG_1)
     + COUNT(w.TANGGALKIRIMPUCL_1) AS PROBE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
 WHERE 1 = 0
