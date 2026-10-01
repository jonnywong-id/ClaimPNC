-- Kueri modul Inbox RCL/PUCL (`MENU_ID 61`, pengganti `Harness/RCLPUCL_Harness`).
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- ============================================================================
-- DUA SUMBER, DAN BATASNYA TEGAS
-- ============================================================================
--
-- Sejak 2026-10-01 berkas ini membaca DUA kelompok tabel, bukan satu:
--
--   POOLDATA.TC_PNC_PUCL              MILIK APLIKASI INI — tabel datar RCL/PUCL
--                                     ketiga tab + layar kerja
--                                     DDL: Database/CREATE_TABLE_3.SQL
--                                          claim-pnc/docs/ddl/tc_pnc_pucl.sql
--
--   DATAPEGA.PC_ASM_FW_GCNMFW_WORK    MILIK PEGA — hanya `daily_report`, lihat alasannya
--   DATAPEGA.PC_ASSIGN_WORKBASKET     pada kueri itu
--   POOLDATA.T_CLAIM_PNC              MILIK PEGA — kunci teknis klaim
--   POOLDATA.T_CLAIM_OBJECTLIST       MILIK PEGA — isian turunan layar kerja
--   POOLDATA.T_CLAIM_ADJUSTMENT       MILIK PEGA — isian turunan layar kerja
--
-- Tidak ada satu pun pernyataan yang menulis, dan memang tidak boleh ada: selama masa
-- paralel setiap tabel hanya boleh ditulis SATU sistem (`P-1`). `TC_PNC_PUCL` pun hanya
-- DIBACA di sini — yang menulisinya adalah proses pengisi, bukan modul ini.
--
-- ============================================================================
-- APA YANG BERUBAH SAAT TABEL DATAR DIPAKAI, DAN APA YANG TIDAK
-- ============================================================================
--
-- TIDAK BERUBAH: penyaring bisnis ketiga tab, urutan baris, ukuran halaman, kolom yang
-- digambar, dan judulnya. Ketiganya tetap turunan langsung dari Report Definition-nya.
--
-- BERUBAH, dan ketiganya harus disadari:
--
--   1. PENYARING ANTREAN BERSAMA HILANG SELURUHNYA — bukan hanya gabungannya.
--
--      Di Pega, `PXASSIGNEDOPERATORID = 'RCLPUCL'` adalah cara MENEMUKAN klaim RCL/PUCL di
--      antara seluruh objek kerja. `TC_PNC_PUCL` adalah tabel KHUSUS RCL/PUCL: setiap
--      barisnya sudah klaim RCL/PUCL menurut proses pengisinya, sehingga penyaring itu
--      tidak lagi menyeleksi apa pun.
--
--      Yang memastikannya bukan penalaran itu melainkan datanya. Dibaca 2026-10-01:
--      kolomnya berisi **nama orang** (`OPERATOR_ID` bernilai sama persis), bukan nama
--      antrean. Menyaringnya dengan literal `'RCLPUCL'` karena itu mengosongkan KETIGA tab
--      — dan itulah yang terjadi sampai hari ini.
--
--      Akibat lain yang ikut dari tabelnya: klaim yang punya DUA penugasan terbuka di
--      antrean yang sama TIDAK LAGI muncul dua kali — `PRIMARY KEY (CLAIMID)` tidak
--      mengizinkannya. Itu selisih terhadap Pega; lihat `docs/ddl/tc_pnc_pucl.sql` §4.
--
--      RISIKO YANG DITERIMA: bila proses pengisi kelak memasukkan baris di luar antrean
--      RCL/PUCL, tidak ada lagi yang menyaringnya di sini. Pemisahan itu berpindah menjadi
--      syarat pengisi, sama seperti pemisahan kelas objek kerja pada butir 2.
--
--   2. PENYARING KELAS OBJEK KERJA HILANG. `TC_PNC_PUCL` tidak punya `PXOBJCLASS`.
--      Pemisahan Work-PNC dari Work-ReceiveDocument karena itu menjadi tanggung jawab
--      PROSES PENGISI, bukan kueri ini. Ia tidak dapat ditegakkan dari sini.
--
--   3. `REFERENCE` DAN `CASE_ID` KINI BERNILAI SAMA. Tabel lama punya `PZINSKEY` (kunci
--      teknis) dan `PYID` (nomor case) sebagai dua kolom; tabel datar hanya menyimpan yang
--      kedua, dengan nama `CLAIMID`. Keduanya tetap dikembalikan sebagai dua alias supaya
--      kontrak ke layar tidak berubah — lihat catatan pada `list_cetak_surat`.
--
-- ============================================================================
-- SATU ANTREAN, TIGA PARTISI
-- ============================================================================
--
-- Ketiga kueri daftar membaca tabel yang SAMA. Yang membedakan hanyalah tiga penyaring, dan
-- ketiganya menyangkut perjalanan surat PUCL:
--
--   list_cetak_surat            TGL_CETAK_DOKUMEN_PUCL IS NULL     surat belum dicetak
--                               STATUS_CASE = :status
--   list_kelengkapan_dokumen    TGL_CETAK_DOKUMEN_PUCL IS NOT NULL surat sudah dicetak
--                               PUCL_APPROVE <> :disetujui
--                               MSIG IS NULL
--   list_klaim_msig             sama seperti di atas, tetapi MSIG = :msig
--
-- Ketiganya ditambah `STATUS_WORK <> :selesai`. Penyaring antrean TIDAK ada lagi — lihat
-- butir 1 di atas.
--
-- Penyaringnya diambil dari `pyFilters` ketiga Report Definition, dan dipastikan ulang
-- terhadap SQL hasil generate Pega sendiri di `RDB List/ReminderPUCL-SQL.xml` — yang memuat
-- penyaring list_kelengkapan_dokumen kata demi kata.
--
-- ============================================================================
-- PEMETAAN KOLOM — properti Pega -> kolom Pega -> kolom tabel datar -> alias
-- ============================================================================
--
-- Diambil dari SEL GRID ketiga section (bukan dari pyListFields Report Definition, yang
-- mengambil 17–25 isian sementara yang digambar hanya sembilan).
--
--   judul kolom          kolom Pega                  kolom tabel datar       alias
--   -------------------- --------------------------- ----------------------- -----------------
--   (tidak digambar)     PZINSKEY                    — TIDAK ADA             REFERENCE *
--   Nomor Case           PYID                        CLAIMID                 CASE_ID
--   No Polis             POLICYNO                    POLICY_NO               POLICY_NUMBER
--   Nama Tertanggung     QQNAME                      QQ_NAME                 INSURED_NAME
--   Tanggal Masuk Inbox  TANGGALKIRIMPUCL_1          TGL_KIRIM_PUCL          INBOX_ENTRY_AT
--   Deskripsi Analyst    KOMENTARANALISATOR_1        KOMENTAR_ANALISATOR     ANALYST_NOTE
--   Status RCL/PUCL      RCL_PUCL_1                  RCL_PUCL                TRACK_CODE
--   Tanggal Cetak Surat  TANGGALCETAKDOKUMENPUCL_1   TGL_CETAK_DOKUMEN_PUCL  LETTER_PRINTED_AT
--   Lama Klaim           LAMAKLAIM_1                 LAMA_KLAIM              CLAIM_AGE
--   Status Kadaluarsa    STATUSKLAIM_1               STATUS_KLAIM            EXPIRY_STATUS
--   (pengurut)           PXCREATEDATETIME            TGL_CREATE_PUCL         CREATED_AT
--
--   * REFERENCE kini diisi `CLAIMID` pula. Lihat catatan pada list_cetak_surat.
--
-- DUA BARIS YANG PALING MUDAH SALAH DI SELURUH BERKAS INI.
--
-- Layar Inbox Manager Receive / PUCL memakai DUA judul yang sama untuk kolom yang BERBEDA:
--
--   judul                layar ini        Inbox Manager Receive / PUCL
--   -------------------- ---------------- ----------------------------
--   Status RCL/PUCL      RCL_PUCL         STATUSKLAIM_1
--   Status Kadaluarsa    STATUS_KLAIM     STATUSCASE_1
--
-- Keduanya diverifikasi dari sel grid section masing-masing. Menyalin pemetaan satu layar ke
-- layar lain akan menampilkan kolom yang salah TANPA satu pun galat.
--
-- Perhatikan pula `STATUS_CASE`: ia MENYARING list_cetak_surat tetapi TIDAK digambar satu
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
-- `RCL_PUCL` dikembalikan MENTAH sebagai TRACK_CODE, dan penerjemahannya menjadi "RCL" atau
-- "PUCL" dikerjakan `inboxrclpucl.TrackOf` di lapisan domain.
--
-- Ini BERBEDA dari modul Inbox Manager Receive / PUCL, yang menuliskan `CASE` penerjemah di
-- dalam SQL-nya. Yang dipakai di sini adalah pola yang sama dengan `ClaimTypeOf` pada modul
-- itu, dan alasannya sama: penyimpanan SQL dan penyimpanan memori wajib menghasilkan teks
-- yang sama persis, dan dua penerjemah di dua tempat dapat menyimpang tanpa ketahuan.
--
-- Ketiga kode yang dipakai layar lama punya teksnya sendiri — '1' RCL, '2' PUCL,
-- '3' Notification — dan kode di luar ketiganya menghasilkan teks kosong. Sumber teksnya
-- BUKAN `CASE` pada `GetReminderPUCL-SQL.xml`: kueri itu memasok PENGINGAT, bukan grid.
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
-- 5. KETIGA TAB TIDAK LAGI MENGGABUNG DUA TABEL. Lihat butir 1 di kepala berkas ini.
--
-- ============================================================================
-- YANG SENGAJA TIDAK BERUBAH — termasuk yang tampak seperti cacat
-- ============================================================================
--
-- * URUTAN mengikuti Report Definition apa adanya: `TGL_CREATE_PUCL DESC, CLAIMID DESC`,
--   terbaca dari `ReminderPUCL-SQL.xml` sebagai `ORDER BY 5 DESC, 7 DESC` (kolom ke-5 dan
--   ke-7 pada daftar pilihnya, yaitu `PXCREATEDATETIME` lalu `PYID`).
--
--   Yang harus disadari: `TGL_CREATE_PUCL` TIDAK digambar di layar lama. Yang digambar
--   sebagai "Tanggal Masuk Inbox" adalah `TGL_KIRIM_PUCL`, dan keduanya dapat terpaut
--   berbulan-bulan — sebuah klaim lahir jauh sebelum ia masuk antrean RCL/PUCL. Akibatnya
--   tabel dapat TERBACA tidak urut oleh penggunanya, dan karena itu kolomnya DITAMPILKAN
--   sebagai kolom terakhir (keputusan Work Owner 2026-09-30).
--
-- * `PUCL_APPROVE <> :disetujui` TIDAK MENANGKAP NULL, dan itu dibiarkan.
--   `NULL <> '1'` menghasilkan UNKNOWN — bukan TRUE — di Oracle maupun PostgreSQL, sehingga
--   klaim yang penanda persetujuannya belum pernah diisi TIDAK muncul di tab Kelengkapan
--   Dokumen maupun Klaim MSIG. Kolom asalnya hanya punya DUA nilai berbeda di produksi
--   (`docs/kolom-t-claimlist-admin.md` §B.3), sehingga jumlah baris yang terdampak bisa
--   besar.
--
--   Memperbaikinya menjadi `(… IS NULL OR … <> :disetujui)` akan MENAMBAH baris yang di
--   Pega tidak pernah terlihat. Itu perubahan perilaku pada layar yang sedang diuji
--   kesetaraannya, dan bukan wewenang berkas ini. Ia dicatat sebagai pertanyaan terbuka.
--
-- * Pembanding `PUCL_APPROVE` diikat sebagai TEKS, mengikuti
--   `RDB List/CountKlaimPUCL-SQL.xml` yang menulis `<> '1'`. `ReminderPUCL-SQL.xml` menulis
--   `<> 1` tanpa kutip pada kolom yang sama — dua rule Pega yang tidak sepakat tentang tipe
--   kolomnya sendiri. Yang dipilih bentuk bertanda kutip, dan `CREATE_TABLE_3.SQL` kini
--   menutup keraguannya: kolomnya `VARCHAR2(5 CHAR)`.
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
-- KENAPA `CLAIMID` DIPILIH DUA KALI, SEBAGAI REFERENCE DAN SEBAGAI CASE_ID
--
-- Tabel Pega punya dua kolom untuk dua peran: `PZINSKEY` sebagai kunci teknis yang dipakai
-- tautan baris, dan `PYID` sebagai nomor case yang digambar. `TC_PNC_PUCL` hanya menyimpan
-- yang kedua.
--
-- Keduanya tetap dikembalikan supaya kontrak ke layar tidak berubah: `REFERENCE` yang
-- dipakai tombol rincian, `CASE_ID` yang digambar sebagai kolom "Nomor Case". Yang berubah
-- hanyalah ISINYA — alamat layar kerja kini memuat `PNC-1865`, bukan
-- `ASM-FW-GCNMFW-WORK PNC-1865`. Tautan lama berbentuk panjang itu TIDAK akan ditemukan.
--
-- Bind: :1 status kerja yang dikecualikan · :2 nilai STATUS_CASE yang diterima
--       :3 offset · :4 jumlah baris
SELECT p.CLAIMID                        AS REFERENCE,
       p.CLAIMID                        AS CASE_ID,
       p.POLICY_NO                      AS POLICY_NUMBER,
       p.QQ_NAME                        AS INSURED_NAME,
       p.TGL_KIRIM_PUCL                 AS INBOX_ENTRY_AT,
       p.KOMENTAR_ANALISATOR            AS ANALYST_NOTE,
       p.RCL_PUCL                       AS TRACK_CODE,
       p.TGL_CETAK_DOKUMEN_PUCL         AS LETTER_PRINTED_AT,
       p.LAMA_KLAIM                     AS CLAIM_AGE,
       p.STATUS_KLAIM                   AS EXPIRY_STATUS,
       p.TGL_CREATE_PUCL                AS CREATED_AT,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM POOLDATA.TC_PNC_PUCL p
 WHERE p.STATUS_WORK <> :1
   AND p.TGL_CETAK_DOKUMEN_PUCL IS NULL
   AND p.STATUS_CASE = :2
 ORDER BY p.TGL_CREATE_PUCL DESC, p.CLAIMID DESC
OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY

-- name: list_kelengkapan_dokumen
-- Tab "Kelengkapan Dokumen" — surat sudah dicetak, belum disetujui, bukan jalur MSIG.
-- — Report Definition/InboxPUCLCetakSurat_RD-RD.xml, pyFilterLogic "A AND B AND C AND D AND E"
-- — SQL hasil generate-nya ada utuh di RDB List/ReminderPUCL-SQL.xml
--
-- Bind: :1 status kerja yang dikecualikan · :2 nilai PUCL_APPROVE yang dikecualikan
--       :3 offset · :4 jumlah baris
SELECT p.CLAIMID                        AS REFERENCE,
       p.CLAIMID                        AS CASE_ID,
       p.POLICY_NO                      AS POLICY_NUMBER,
       p.QQ_NAME                        AS INSURED_NAME,
       p.TGL_KIRIM_PUCL                 AS INBOX_ENTRY_AT,
       p.KOMENTAR_ANALISATOR            AS ANALYST_NOTE,
       p.RCL_PUCL                       AS TRACK_CODE,
       p.TGL_CETAK_DOKUMEN_PUCL         AS LETTER_PRINTED_AT,
       p.LAMA_KLAIM                     AS CLAIM_AGE,
       p.STATUS_KLAIM                   AS EXPIRY_STATUS,
       p.TGL_CREATE_PUCL                AS CREATED_AT,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM POOLDATA.TC_PNC_PUCL p
 WHERE p.STATUS_WORK <> :1
   AND p.TGL_CETAK_DOKUMEN_PUCL IS NOT NULL
   AND p.PUCL_APPROVE <> :2
   AND p.MSIG IS NULL
 ORDER BY p.TGL_CREATE_PUCL DESC, p.CLAIMID DESC
OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY

-- name: list_klaim_msig
-- Tab "Klaim MSIG" — sama seperti list_kelengkapan_dokumen, tetapi jalur MSIG.
-- — Report Definition/InboxMISG_RD-RD.xml, pyFilterLogic "A AND B AND C AND D AND E"
--
-- SATU-SATUNYA perbedaan terhadap kueri di atasnya adalah baris MSIG: `IS NULL` menjadi
-- `= :4`. Keduanya sengaja TIDAK disatukan menjadi satu kueri berparameter: penyaring yang
-- artinya berbalik menurut nilai bind adalah tempat paling mudah menukar isi dua tab, dan
-- tidak ada apa pun di layar yang menandakannya bila itu terjadi.
--
-- KUERI INI NYARIS SELALU MENGEMBALIKAN NOL BARIS, tetapi TIDAK selalu. Hitungan langsung
-- pada 2026-09-30 atas kolom asalnya di portal ASM: `GROUP BY MSIG_1` mengembalikan 'MSIG'
-- SATU baris dan kosong 7.721 baris. Kolomnya terisi, hanya sangat jarang.
--
-- Kuerinya dibangun apa adanya — keputusan Work Owner 2026-09-23 — dan jarangnya isi tab
-- ini dinyatakan ke pengguna lewat Tab.Notice, bukan disamarkan.
--
-- Bind: :1 status kerja yang dikecualikan · :2 nilai PUCL_APPROVE yang dikecualikan
--       :3 penanda jalur MSIG · :4 offset · :5 jumlah baris
SELECT p.CLAIMID                        AS REFERENCE,
       p.CLAIMID                        AS CASE_ID,
       p.POLICY_NO                      AS POLICY_NUMBER,
       p.QQ_NAME                        AS INSURED_NAME,
       p.TGL_KIRIM_PUCL                 AS INBOX_ENTRY_AT,
       p.KOMENTAR_ANALISATOR            AS ANALYST_NOTE,
       p.RCL_PUCL                       AS TRACK_CODE,
       p.TGL_CETAK_DOKUMEN_PUCL         AS LETTER_PRINTED_AT,
       p.LAMA_KLAIM                     AS CLAIM_AGE,
       p.STATUS_KLAIM                   AS EXPIRY_STATUS,
       p.TGL_CREATE_PUCL                AS CREATED_AT,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM POOLDATA.TC_PNC_PUCL p
 WHERE p.STATUS_WORK <> :1
   AND p.TGL_CETAK_DOKUMEN_PUCL IS NOT NULL
   AND p.PUCL_APPROVE <> :2
   AND p.MSIG = :3
 ORDER BY p.TGL_CREATE_PUCL DESC, p.CLAIMID DESC
OFFSET :4 ROWS FETCH NEXT :5 ROWS ONLY

-- name: daily_report
-- LAPORAN HARIAN RCL/PUCL — keluaran tombol ekspor tab "Cetak Surat".
-- — Activity/ExportCetakSurat_act-Act.xml menjalankan kueri di bawah lalu pxConvertResultsToCSV
-- — RDB List/GetDataPUCLRCLForDailyReport-SQL.xml
--
-- ============================================================================
-- SATU-SATUNYA KUERI DI BERKAS INI YANG MASIH MEMBACA TABEL PEGA — DAN ITU DISENGAJA
-- ============================================================================
--
-- Ketiga tab dan layar kerja sudah pindah ke `TC_PNC_PUCL`. Laporan ini TIDAK, dan sebabnya
-- bukan pekerjaan yang tertunda melainkan ISI YANG BERBEDA.
--
-- Cabang keduanya mengambil SELURUH klaim ber-GROUPPANEL '002' (Personal Accident) pada
-- rentang tanggal itu, **tanpa gabungan antrean bersama sama sekali** — termasuk klaim yang
-- tidak pernah masuk antrean RCL/PUCL. `TC_PNC_PUCL` berisi antrean RCL/PUCL; klaim PA di
-- luar antrean itu TIDAK ADA di sana.
--
-- Memindahkannya sekarang akan membuat berkas unduhan kehilangan baris TANPA satu pun
-- galat, dan tidak ada apa pun di layar yang menandakannya. Ia baru dapat pindah bila proses
-- pengisi dinyatakan memuat klaim PA di luar antrean pula — dan itu belum diputuskan.
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
--   * Ia ber-UNION dengan cabang kedua yang dijelaskan di atas.
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
-- Dipakai perintah `-periksa`: memastikan tabel datar RCL/PUCL terbaca dari koneksi yang
-- dipakai.
--
-- Ia tidak menyentuh satu baris pun — yang diperiksa adalah hak baca dan keberadaan
-- tabelnya, bukan isinya.
--
-- `COUNT(*)` dipakai, bukan sebuah kolom, supaya hasilnya SELALU tepat satu baris meski
-- penyaringnya tidak meloloskan apa pun — pemanggil karena itu tidak perlu membedakan
-- "tidak ada baris" dari "gagal dibaca".
--
-- TABEL INI BARU DAN AKAN KOSONG SAMPAI PROSES PENGISI BERJALAN. Pemeriksaan ini lolos pada
-- tabel kosong, dan memang harus begitu: kosong adalah jawaban, tidak-ada adalah kerusakan.
SELECT COUNT(*) AS PROBE
  FROM POOLDATA.TC_PNC_PUCL p
 WHERE 1 = 0

-- name: check_columns
-- Memastikan kolom PUCL yang MENYARING layar ini dan yang MENGISI layar kerja terbaca.
--
-- Ia terpisah dari check_rclpucl dengan sengaja. Tabelnya sama, tetapi yang diuji berbeda:
-- di atas keberadaan TABEL, di sini keberadaan KOLOM. Kolom yang tidak ada menghasilkan
-- galat yang menyebut namanya, dan itulah yang membedakan "modul ini belum dapat dipakai di
-- sini" dari "antreannya memang kosong".
--
-- Kelima kolom penyaring diperiksa justru karena nama kolom tabel ini BERBEDA dari nama
-- kolom Pega yang digantikannya — akhiran `_1` dibuang dan kata dipisah garis bawah. Satu
-- nama yang tertinggal pada bentuk lama akan gagal dengan ORA-00904 pada permintaan pertama
-- di produksi, bukan saat build.
--
-- KEEMPAT KOLOM ISIAN SURAT IKUT DIPERIKSA, dan alasannya berbeda dari kelima di atas:
-- `PERIHAL`, `KETERANGAN1`, `KETERANGAN2`, `KETERANGAN3` **tidak ada di
-- `Database/CREATE_TABLE_3.SQL`**. Tabel yang berjalan memilikinya, berkas DDL-nya tidak —
-- sehingga portal yang dibuat dari berkas itu akan kehilangan keempatnya, dan `detail`
-- gagal pada klaim pertama yang dibuka. Di sinilah keadaan itu terbaca lebih dulu.
SELECT COUNT(p.TGL_CETAK_DOKUMEN_PUCL)
     + COUNT(p.STATUS_CASE)
     + COUNT(p.PUCL_APPROVE)
     + COUNT(p.MSIG)
     + COUNT(p.TGL_KIRIM_PUCL)
     + COUNT(p.PERIHAL)
     + COUNT(p.KETERANGAN1)
     + COUNT(p.KETERANGAN2)
     + COUNT(p.KETERANGAN3) AS PROBE
  FROM POOLDATA.TC_PNC_PUCL p
 WHERE 1 = 0

-- name: diagnose_empty
-- Menjelaskan MENGAPA ketiga tab kosong, dengan angka — bukan dengan dugaan.
--
-- # Kenapa ini ada
--
-- Ketiga tab yang kosong terbaca persis sama dengan antrean yang memang sepi, dan kedua
-- keadaan itu menuntut tindakan yang berbeda: yang satu menunggu pekerjaan, yang satu
-- menunggu perbaikan proses pengisi. Tanpa angka ini, membedakannya menuntut seseorang
-- membuka SQL*Plus dan menyusun penyaringnya satu per satu.
--
-- Setiap kolom menghitung baris yang LOLOS satu penyaring, berdiri sendiri. Penyaring yang
-- menghasilkan NOL sementara TOTAL tidak nol adalah penyaring yang mengosongkan layar.
--
-- Ia tidak menyaring apa pun dan tidak menyentuh satu baris pun di luar penghitungan.
SELECT COUNT(*)                                                          AS TOTAL_ROWS,
       COUNT(CASE WHEN p.STATUS_WORK <> :1 THEN 1 END)                   AS BUKAN_SELESAI,
       COUNT(CASE WHEN p.TGL_CETAK_DOKUMEN_PUCL IS NULL THEN 1 END)      AS BELUM_BERSURAT,
       COUNT(CASE WHEN p.STATUS_CASE = :2 THEN 1 END)                    AS STATUS_CASE_COCOK,
       COUNT(CASE WHEN p.TGL_CETAK_DOKUMEN_PUCL IS NOT NULL THEN 1 END)  AS SUDAH_BERSURAT,
       COUNT(CASE WHEN p.PUCL_APPROVE <> :3 THEN 1 END)                  AS MASIH_DI_PUCL,
       COUNT(CASE WHEN p.MSIG IS NULL THEN 1 END)                        AS BUKAN_MSIG,
       COUNT(CASE WHEN p.MSIG = :4 THEN 1 END)                           AS JALUR_MSIG
  FROM POOLDATA.TC_PNC_PUCL p

-- name: check_laporan
-- Memastikan kedua tabel PEGA yang masih dipakai LAPORAN HARIAN terbaca.
--
-- Terpisah dari check_rclpucl karena yang diperiksa memang milik sistem lain. Sejak ketiga
-- tab pindah ke tabel datar, kedua tabel ini hanya dipakai `daily_report` — dan bila
-- keduanya tidak terbaca, yang gagal HANYA tombol unduh tab "Cetak Surat", bukan layarnya.
-- Galat yang menyebut tabel yang salah menyesatkan orang yang memperbaikinya.
SELECT COUNT(*) AS PROBE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKBASKET b
               ON b.PXREFOBJECTKEY = w.PZINSKEY
 WHERE 1 = 0

-- name: detail
-- LAYAR KERJA RCL/PUCL untuk SATU klaim — yang terbuka saat nomor klaim diklik.
-- — Section/SendtoRCLPUCL-Section.xml, kontainer dua bagian
-- — Section/SectionLampiranSuratPUCL-Section.xml      bagian "Lampiran Surat"
-- — Section/SectionPenerimaanDokumenPUCL-Section.xml  bagian "Penerimaan Dokumen"
-- — Activity/SetDataLampiranSuratRCLPUCL_Act-Act.xml  asal ketiga isian TURUNAN
--
-- ============================================================================
-- KUNCINYA NOMOR CASE, DAN ANAK KLAIMNYA DICAPAI LEWAT T_CLAIM_PNC
-- ============================================================================
--
-- INI BAGIAN YANG PALING MUDAH SALAH DI SELURUH BERKAS INI, dan kesalahannya TIDAK
-- menghasilkan galat — hanya layar kerja yang tiga isiannya kosong.
--
-- Kolom bernama `CLAIMID` di tiga tabel TIDAK berisi hal yang sama:
--
--   POOLDATA.TC_PNC_PUCL.CLAIMID         nomor case      "PNC-1865"
--   POOLDATA.T_CLAIM_PNC.CLAIMID         kunci teknis    "ASM-FW-GCNMFW-WORK PNC-1865"
--   POOLDATA.T_CLAIM_OBJECTLIST.CLAIMID  kunci teknis    "ASM-FW-GCNMFW-WORK PNC-1865"
--   POOLDATA.T_CLAIM_ADJUSTMENT.CLAIMID  kunci teknis    "ASM-FW-GCNMFW-WORK PNC-1865"
--
-- Menggabungkan yang pertama langsung dengan ketiga sisanya SELALU mengembalikan nol baris.
-- Join-nya tetap sah; hanya tidak pernah cocok.
--
-- Jembatannya `POOLDATA.T_CLAIM_PNC`, yang memuat KEDUA bentuk: `CLAIMNO` sama dengan
-- nomor case, `CLAIMID` sama dengan kunci teknis. Pasangan itu dipakai Pega sendiri
-- (`p.CLAIMNO = A.PYID` pada `inboxadmin`) dan sudah dipakai modul lain di aplikasi ini.
--
-- KENAPA LEWAT TABEL, BUKAN MERANGKAI PREFIX-NYA
--
-- Menulis `'ASM-FW-GCNMFW-WORK ' || p.CLAIMID` akan bekerja untuk klaim WARISAN dan GAGAL
-- DIAM-DIAM untuk klaim baru: `D-22` dan `D-71` menghapus prefix itu bagi klaim ber-nomor
-- `PNCN.YY.xxxx`. Membaca pasangannya dari `T_CLAIM_PNC` benar untuk kedua bentuk, karena
-- yang dibaca adalah nilainya — bukan tebakan tentang bentuknya.
--
-- ============================================================================
-- TIGA ISIAN DITURUNKAN, BUKAN DIBACA DARI KOLOMNYA SENDIRI
-- ============================================================================
--
-- Activity penyusun lampiran mengisinya dari anak-anak klaim, dan ketiganya hanya diisi
-- bila masih kosong (precondition `.<isian>==""`):
--
--   .UP            <- pyWorkPage.ClaimData.ObjectList(1).ObjectName
--   .NamaPeserta   <- pyWorkPage.ClaimData.ObjectList(1).ObjectName
--   .JumlahTagihan <- pyWorkPage.ClaimData.ObjectList(1).ObjectCoverageList(1)
--                                          .AdjustmentList(1).ProposeValue
--
-- Indeksnya SELALU `(1)`. Klaim dengan banyak objek hanya membawa yang PERTAMA ke suratnya,
-- dan itu perilaku sistem lama apa adanya (`P-5`) — bukan penyederhanaan di sini.
--
-- ============================================================================
-- "UP" BERISI NAMA OBJEK, DAN ITU MEMANG BENAR — JANGAN "DIPERBAIKI"
-- ============================================================================
--
-- PERHATIKAN KEDUA EKSPRESI DI ATAS SAMA PERSIS. Kolom "UP" (Uang Pertanggungan) di Pega
-- karena itu berisi NAMA OBJEK, bukan angka.
--
-- Ia terbaca seperti salin-tempel yang keliru, dan pada 2026-09-24 ia memang sempat
-- "diperbaiki" di berkas ini menjadi `SumTSI` pada coverage pertama — lengkap dengan
-- subkueri kedua, alias, uji, dan pernyataan selisih terencana.
--
-- **Work Owner meralatnya pada hari yang sama: UP memang `ObjectName`.** Seluruh perbaikan
-- itu dicabut (`keputusan-implementasi.md` §47.1), dan `P-5` berlaku apa adanya — perilaku
-- direplikasi KECUALI perbaikannya diputuskan eksplisit, dan untuk yang ini TIDAK.
--
-- Karena itu kueri di bawah memilih SATU subkueri nama objek, dan nilainya mengisi DUA isian
-- sekaligus. `POOLDATA.T_CLAIM_OBJECTCOVERAGE` TIDAK disentuh sama sekali, dan satu uji
-- kueri menjaganya tetap begitu — berpasangan dengan satu uji di penyimpanan memori yang
-- menuntut kedua isian SAMA. Dua lapis penahan, karena isian ini sudah dua kali terbaca
-- sebagai cacat.
--
-- ============================================================================
-- APA ARTI "PERTAMA" DI SINI
-- ============================================================================
--
-- Di Pega, `(1)` adalah entri pertama pada page list KLIPBOARD, dan urutannya ditentukan
-- cara halaman itu dimuat — sesuatu yang TIDAK terbaca dari export mana pun.
--
-- Di sini "pertama" ditetapkan tegas: `OBJECTID` terkecil untuk objek, lalu
-- `OBJECTCOVERAGEID` dan `ADJUSTMENTID` terkecil untuk adjustment-nya. Itu SELISIH TERENCANA
-- — urutan yang tidak ditetapkan membuat isian surat berubah-ubah antar pemanggilan pada
-- klaim yang punya lebih dari satu objek.
--
-- PERHATIKAN NAMANYA `OBJECTCOVERAGEID`, BUKAN `COVERAGEID`. Nama pendeknya ditebak saat
-- modul ini ditulis, dan tebakan itu membuat SETIAP pembukaan layar kerja gagal dengan
-- `ORA-00904: "J"."COVERAGEID": invalid identifier` — bukan sebagian, melainkan seluruhnya.
-- Nama yang benar dibaca dari `ALL_TAB_COLUMNS` pada 2026-09-30.
--
-- ============================================================================
-- PEMETAAN KOLOM
-- ============================================================================
--
--   isian layar            properti Pega                              kolom tabel datar
--   ---------------------- ----------------------------------------- ---------------------
--   (kunci)                .pyID                                     p.CLAIMID
--   judul layar            .pyID                                     p.CLAIMID
--   RCL/PUCL               .ClaimData.PUCLStatus.RCL_PUCL            p.RCL_PUCL
--   Deskripsi Analyst      .ClaimData.PUCLStatus.KomentarAnalisator  p.KOMENTAR_ANALISATOR
--   No Polis               .Policy.PolicyNo                          p.POLICY_NO
--   Tanggal Kejadian       .ClaimData.DateOfLoss                     p.DATE_OF_LOSS
--   Komentar PUCL          .ClaimData.PUCLStatus.KomentarPUCL        p.KOMENTAR_PUCL
--   Perihal                .ClaimData.PUCLStatus.Perihal             p.PERIHAL
--   Keterangan Pembuka     .ClaimData.PUCLStatus.Keterangan1         p.KETERANGAN1
--   Keterangan Isi         .ClaimData.PUCLStatus.Keterangan2         p.KETERANGAN2
--   Keterangan Penutup     .ClaimData.PUCLStatus.Keterangan3         p.KETERANGAN3
--   Nama Peserta           .ClaimData.PUCLStatus.NamaPeserta         TURUNAN (objek pertama)
--   UP                     .ClaimData.PUCLStatus.UP                  TURUNAN (objek pertama — SAMA)
--   Jumlah Tagihan         .ClaimData.PUCLStatus.JumlahTagihan       TURUNAN (adjustment pertama)
--
-- PENYARING KELAS OBJEK KERJA HILANG, dan itu akibat tabelnya — `TC_PNC_PUCL` tidak punya
-- `PXOBJCLASS`. Perlindungan yang dulu diberikannya (kunci milik kelas lain mengembalikan
-- baris berkolom PUCL kosong) kini berpindah ke proses pengisi, yang hanya boleh memuat
-- baris `ASM-FW-GCNMFW-Work-PNC`.
--
-- ============================================================================
-- EMPAT ISIAN CLIPBOARD AKHIRNYA PUNYA KOLOM — dan itu mengubah angka sembilan
-- ============================================================================
--
-- `PERIHAL`, `KETERANGAN1`, `KETERANGAN2`, dan `KETERANGAN3` ADA di `TC_PNC_PUCL` yang
-- berjalan, dan terisi. Dibaca langsung dari katalog dan datanya pada 2026-10-01, dan isinya
-- cocok kata demi kata dengan layar Pega:
--
--   PERIHAL      "Kelengkapan Data Dokumen Klaim Polis Asuransi Kecelakaan Pribadi"
--   KETERANGAN1  "Sehubungan dengan telah diterimanya dokumen klaim polis Asuransi, ..."
--   KETERANGAN2  "Kwitansi Asli dari Biaya Konsultasi Dokter(bukan Nota / Invoice / ..."
--   KETERANGAN3  "Bila dokumen yang diminta tidak dilengkapi atau kelengkapan dokumen ..."
--
-- PERHATIKAN KEEMPATNYA TIDAK ADA DI `Database/CREATE_TABLE_3.SQL`. Berkas itu mendefinisikan
-- 26 kolom; tabel yang berjalan punya 30. DDL di repo tertinggal dari tabelnya, dan selisih
-- itu BUKAN urusan kerapian: portal lain yang dibuat dari berkas itu akan kehilangan keempat
-- kolom ini, dan kueri di bawah gagal `ORA-00904` pada permintaan pertama. `check_columns`
-- memeriksanya supaya keadaan itu terbaca saat `-periksa`, bukan saat pengguna membuka layar.
--
-- ============================================================================
-- LIMA ISIAN YANG TETAP TIDAK PUNYA KOLOM — dan ini BUKAN kelalaian
-- ============================================================================
--
-- NIK ("No Kontrak") · BusinessUnitSeksi · TanggalTerimaDokumenPUCL ("Tanggal Kelengkapan
-- Dokumen") · EmailLOD ("Email Tertanggung"), ditambah daftar berulang
-- "Tanggal terima Dokumen / Tanggal / Keterangan" pada bagian kedua.
--
-- SEBABNYA BUKAN KOLOM YANG BELUM DITEMUKAN. Work Owner menjelaskan 2026-09-24 bahwa
-- kesembilannya — sebelum keempat di atas muncul — diambil dari **clipboard** Pega
-- (`.ClaimData.PUCLStatus.*`), dan properti clipboard yang tidak dioptimasi memang TIDAK
-- punya kolom sendiri. Yang mengubah keadaan ini hanyalah mengeksposnya sebagai kolom, dan
-- itulah yang sudah terjadi pada keempat isian di atas (`keputusan-implementasi.md` §47.2,
-- §79.3).
--
-- Dua di antaranya patut disebut khusus:
--
--   * `ID_PERIHAL` dan `PERIHAL_NAME` pada pemilih "Perihal" BUKAN master Perihal. Kedua
--     properti itu dipakai ulang untuk hal yang sama sekali berbeda di
--     `RDB List/CheckHoliday_SQL-SQL.xml`, tempat keduanya menampung TANGGAL. Itu utang
--     teknis §4.2 apa adanya.
--   * `TANGGALTERIMADOKUMEN` ADA di export, tetapi pada
--     `POOLDATA.T_CLAIM_RECIVEDCLAIM` — tabel BERKAS PENERIMAAN DOKUMEN, bukan kolom PUCL
--     pada objek kerja klaim. Keduanya bernama mirip dan mudah tertukar.
--
-- Kelimanya tetap DIGAMBAR di layar, di tempatnya, bertanda "di clipboard Pega" —
-- bukan dihilangkan dan bukan digambar sebagai sel kosong. Sel kosong berarti PETUGAS belum
-- mengisinya; "di clipboard Pega" berarti nilainya ADA tetapi tidak terbaca dari tabel.
-- Keduanya menuntut tindakan yang berbeda dari orang yang berbeda. Mengarang isinya
-- melanggar larangan paling dasar proyek ini.
--
-- Bind: :1 nomor case (CLAIMID)
SELECT p.CLAIMID                        AS REFERENCE,
       p.CLAIMID                        AS CLAIM_NUMBER,
       p.RCL_PUCL                       AS TRACK_CODE,
       p.KOMENTAR_ANALISATOR            AS ANALYST_NOTE,
       p.POLICY_NO                      AS POLICY_NUMBER,
       p.DATE_OF_LOSS                   AS LOSS_DATE,
       p.KOMENTAR_PUCL                  AS PUCL_NOTE,
       p.PERIHAL                        AS SUBJECT,
       p.KETERANGAN1                    AS OPENING_NOTE,
       p.KETERANGAN2                    AS BODY_NOTE,
       p.KETERANGAN3                    AS CLOSING_NOTE,
       -- Kedua kolom berikut TIDAK digambar sebagai isian. Keduanya menentukan TOMBOL mana
       -- yang muncul: `MSIG` memilih antara "Download Dokumen" dan "Tutup Klaim",
       -- `GROUPPANEL` memilih antara "Kirim Ke Analyst" (PA) dan "Kirim ke PIC Teknik"
       -- (Travel). Syaratnya dibaca dari `pyUserData/pyCondition` tiap sel `pxButton` pada
       -- kedua section, dan ditegakkan di domain — lihat ClaimDetail.Buttons.
       p.MSIG                           AS MSIG_FLAG,
       p.GROUPPANEL                     AS GROUP_PANEL,
       (SELECT o.OBJECTNAME
          FROM POOLDATA.T_CLAIM_OBJECTLIST o
               INNER JOIN POOLDATA.T_CLAIM_PNC c
                       ON c.CLAIMID = o.CLAIMID
         WHERE c.CLAIMNO = p.CLAIMID
         ORDER BY o.OBJECTID
         FETCH FIRST 1 ROWS ONLY)       AS FIRST_OBJECT_NAME,
       (SELECT j.PROPOSE_VALUE
          FROM POOLDATA.T_CLAIM_ADJUSTMENT j
               INNER JOIN POOLDATA.T_CLAIM_PNC c
                       ON c.CLAIMID = j.CLAIMID
         WHERE c.CLAIMNO = p.CLAIMID
           AND j.OBJECTID = (SELECT o2.OBJECTID
                               FROM POOLDATA.T_CLAIM_OBJECTLIST o2
                                    INNER JOIN POOLDATA.T_CLAIM_PNC c2
                                            ON c2.CLAIMID = o2.CLAIMID
                              WHERE c2.CLAIMNO = p.CLAIMID
                              ORDER BY o2.OBJECTID
                              FETCH FIRST 1 ROWS ONLY)
         ORDER BY j.OBJECTCOVERAGEID, j.ADJUSTMENTID
         FETCH FIRST 1 ROWS ONLY)       AS FIRST_PROPOSE_VALUE
  FROM POOLDATA.TC_PNC_PUCL p
 WHERE p.CLAIMID = :1

-- name: check_detail
-- Memastikan KETIGA tabel yang dipakai isian TURUNAN terbaca, beserta jembatan kuncinya.
--
-- Terpisah dari check_rclpucl karena tabelnya memang berbeda, dan galat yang menyebut tabel
-- yang salah menyesatkan orang yang memperbaikinya. Tanpa ketiganya, layar kerja tetap
-- terbuka tetapi "Nama Peserta", "UP", dan "Jumlah Tagihan" diam-diam kosong.
--
-- `T_CLAIM_PNC` ikut diperiksa karena ia JEMBATAN antara nomor case di tabel datar dan kunci
-- teknis di kedua tabel anak. Tanpa tabel itu, kedua isian turunan tidak dapat dicapai sama
-- sekali — dan kegagalannya terbaca persis seperti klaim yang memang tidak punya objek.
SELECT COUNT(*) AS PROBE
  FROM POOLDATA.T_CLAIM_PNC c
       INNER JOIN POOLDATA.T_CLAIM_OBJECTLIST o
               ON o.CLAIMID = c.CLAIMID
       INNER JOIN POOLDATA.T_CLAIM_ADJUSTMENT j
               ON j.CLAIMID = o.CLAIMID
              AND j.OBJECTID = o.OBJECTID
 WHERE 1 = 0

-- name: documents
-- Daftar dokumen satu klaim — tombol **"Lihat Dokumen"** (`localAction ViewAttachmentPUCL`).
--
-- ============================================================================
-- JALUR DATANYA DIPILIH DARI BUKTI, BUKAN DARI NAMA SECTION
-- ============================================================================
-- `Section/ViewAttachmentPUCLDetail-Section.xml` memakai Report Definition
-- `GCNMGetAllAttachments` pada kelas `Link-Attachment`, yaitu tabel lampiran Pega
-- (`DATAPEGA.PC_LINK_ATTACHMENT`). Meniru itu apa adanya akan membawa kita ke
-- `PC_DATA_WORKATTACH.PZPVSTREAM` — blob serialisasi Pega, BUKAN berkas mentah.
--
-- Katalog dan data dibaca langsung 2026-10-01, dan memberi jalan yang lebih baik:
--
--   POOLDATA.DATA_ATTACHFILE  9.937 baris · ATTACHFILE BLOB berisi BERKAS MENTAH
--   IDPEGA cocok PZINSKEY objek kerja   3.915 dari 3.974 yang terisi
--   IDPEGA cocok PYID (nomor case)         32   <- bukan ini
--   IDPEGA cocok kunci PC_LINK_ATTACHMENT    0   <- bukan ini
--
-- Jadi `IDPEGA` menyimpan **kunci objek kerja**, dan berkasnya dapat dibaca apa adanya —
-- tanpa membongkar blob Pega dan tanpa memanggil `pooldata.base64encode` yang `D-02` larang.
--
-- ============================================================================
-- YANG TIDAK TERLIHAT LEWAT JALUR INI, DAN ITU DINYATAKAN
-- ============================================================================
-- Untuk seluruh klaim PNC, `PC_LINK_ATTACHMENT` memuat 12.051 lampiran sementara
-- `DATA_ATTACHFILE` hanya 9.937 baris seluruhnya. Dokumen yang HANYA ada di tabel Pega —
-- lampiran lama yang diunggah lewat jalur bawaan Pega — tidak muncul di sini.
--
-- Layar menyatakan kemungkinan itu alih-alih menampilkan daftar kosong seolah klaimnya
-- memang tidak berdokumen. Arah kegagalannya dipilih sadar: KURANG, bukan salah.
--
-- Nama kategori dicari lewat LEFT JOIN, mengikuti alasan yang sama seperti `inboxpladla`:
-- satu kode yang tidak ada di master tidak boleh MENYEMBUNYIKAN dokumennya.
--
-- Bind: :1 nomor case
SELECT a.DATAID                                     AS DOCUMENT_ID,
       a.ATTACHNAME                                 AS DOCUMENT_NAME,
       a.ATTACHMIMETYPE                             AS MIME_TYPE,
       COALESCE(t.TYPE_DOCUMENT, a.CATEGORY)        AS CATEGORY_NAME,
       COALESCE(dt.DETAIL_DOCUMENT, a.SUB_CATEGORY) AS SUBCATEGORY_NAME,
       a.INPUTDATE                                  AS UPLOADED_AT,
       a.INPUTOPERATOR                              AS UPLOADED_BY
  FROM POOLDATA.DATA_ATTACHFILE a
  LEFT JOIN POOLDATA.V_LST_DOC_TYPE t
         ON t.ID = a.CATEGORY
  LEFT JOIN POOLDATA.V_LST_DET_TYPE_DOC dt
         ON dt.ID = a.SUB_CATEGORY
 WHERE a.ATTACHFILE IS NOT NULL
   AND EXISTS (SELECT 1
                 FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
                WHERE w.PZINSKEY = a.IDPEGA
                  AND TRIM(w.PYID) = TRIM(:1))
 ORDER BY a.INPUTDATE DESC NULLS LAST, a.DATAID DESC

-- name: document_content
-- ISI satu dokumen.
--
-- `DATAID` dapat ditebak, sehingga pernyataan ini TIDAK menerimanya apa adanya: dokumennya
-- wajib terbukti milik klaim yang diminta. Rantai kepemilikan diperiksa di dalam kueri, bukan
-- di lapisan Go — pemeriksaan yang berada di luar kueri dapat terlewat oleh pemanggil baru.
--
-- Isinya dibaca sebagai bita apa adanya dari kolom BLOB. `GetAttachmentFromDB_Sql` lama
-- membungkusnya `pooldata.base64encode(attachfile)`; pembungkusan itu tidak dibawa karena
-- memanggil procedure basis data (`D-02`) dan membesarkan muatan sepertiga tanpa manfaat.
--
-- Bind: :1 id dokumen · :2 nomor case
SELECT a.ATTACHNAME     AS DOCUMENT_NAME,
       a.ATTACHMIMETYPE AS MIME_TYPE,
       a.ATTACHFILE     AS CONTENT
  FROM POOLDATA.DATA_ATTACHFILE a
 WHERE a.DATAID = :1
   AND EXISTS (SELECT 1
                 FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
                WHERE w.PZINSKEY = a.IDPEGA
                  AND TRIM(w.PYID) = TRIM(:2))
