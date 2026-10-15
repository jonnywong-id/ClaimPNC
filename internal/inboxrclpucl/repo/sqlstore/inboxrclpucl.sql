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
--   (dulu) DATAPEGA.PC_ASM_FW_GCNMFW_WORK — TIDAK dibaca lagi sejak 2026-10-08 (keputusan
--                                     Work Owner); lihat catatan SUMBER BARU per kueri
--   DATAPEGA.PC_ASSIGN_WORKBASKET     MILIK PEGA — lihat kueri yang memakainya
--   DATAPEGA.PC_LINK_ATTACHMENT       MILIK PEGA — kategori lampiran
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
--       :3 status kerja ditolak yang dikecualikan · :4 offset · :5 jumlah baris
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
   AND (p.STATUS_WORK IS NULL OR p.STATUS_WORK <> :3)
 ORDER BY p.TGL_CREATE_PUCL DESC, p.CLAIMID DESC
OFFSET :4 ROWS FETCH NEXT :5 ROWS ONLY

-- name: list_kelengkapan_dokumen
-- Tab "Kelengkapan Dokumen" — surat sudah dicetak, belum disetujui, bukan jalur MSIG.
-- — Report Definition/InboxPUCLCetakSurat_RD-RD.xml, pyFilterLogic "A AND B AND C AND D AND E"
-- — SQL hasil generate-nya ada utuh di RDB List/ReminderPUCL-SQL.xml
--
-- Bind: :1 status kerja yang dikecualikan · :2 nilai PUCL_APPROVE yang dikecualikan
--       :3 status kerja ditolak yang dikecualikan · :4 offset · :5 jumlah baris
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
   AND (p.STATUS_WORK IS NULL OR p.STATUS_WORK <> :3)
 ORDER BY p.TGL_CREATE_PUCL DESC, p.CLAIMID DESC
OFFSET :4 ROWS FETCH NEXT :5 ROWS ONLY

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
--       :3 penanda jalur MSIG · :4 status kerja ditolak yang dikecualikan
--       :5 offset · :6 jumlah baris
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
   AND (p.STATUS_WORK IS NULL OR p.STATUS_WORK <> :4)
 ORDER BY p.TGL_CREATE_PUCL DESC, p.CLAIMID DESC
OFFSET :5 ROWS FETCH NEXT :6 ROWS ONLY

-- name: daily_report
-- LAPORAN HARIAN RCL/PUCL — keluaran tombol ekspor tab "Cetak Surat".
-- — Activity/ExportCetakSurat_act-Act.xml menjalankan kueri di bawah lalu pxConvertResultsToCSV
-- — RDB List/GetDataPUCLRCLForDailyReport-SQL.xml
--
-- ============================================================================
-- SUMBER: POOLDATA.TC_PNC_PUCL (2026-10-08) — SEBELUMNYA SENGAJA DI TABEL PEGA
-- ============================================================================
--
-- Laporan ini sempat SENGAJA dibiarkan membaca `PC_ASM_FW_GCNMFW_WORK`, karena cabang
-- keduanya mengambil seluruh klaim Personal Accident TANPA gabungan antrean bersama. Work
-- Owner kemudian menetapkan objek kerja Pega tidak dipakai lagi dan laporan ini pindah ke
-- `TC_PNC_PUCL` ("Perubahan nama tabel untuk Inbox.xlsx", kolom E).
--
-- Kekhawatiran lamanya lebih sempit daripada kedengarannya: cabang PA TETAP menuntut
-- `TANGGALKIRIMPUCL_1` berada dalam rentang, sehingga ia hanya memuat klaim PA yang PERNAH
-- dikirim ke RCL/PUCL. Di portal ASM (diukur 2026-10-08) `TC_PNC_PUCL` menyimpan baris yang
-- antreannya sudah berpindah ke orang lain maupun yang sudah `Resolved-*` — jadi klaim
-- seperti itu tetap terbawa.
--
-- RISIKO YANG TERSISA: bila proses pengisi kelak menghapus baris yang meninggalkan antrean,
-- cabang PA kehilangan baris itu tanpa galat. Proses pengisi WAJIB menyimpan setiap klaim
-- yang pernah dikirim ke RCL/PUCL, bukan hanya yang masih menunggu.
--
-- Pemetaan: TANGGALKIRIMPUCL_1 -> TGL_KIRIM_PUCL · KOMENTARANALISATOR_1 -> KOMENTAR_ANALISATOR ·
-- TANGGALCETAKDOKUMENPUCL_1 -> TGL_CETAK_DOKUMEN_PUCL · RCL_PUCL_1 -> RCL_PUCL ·
-- STATUSCLAIM_1 -> STATUS_CLAIM · GROUPPANEL_1 -> GROUPPANEL · gabung antrean ->
-- ASSIGNED_OPERATOR_ID. Penyaring PXOBJCLASS hilang: tabelnya hanya berisi klaim (DDL §3b).
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
--   :1 akun antrean bersama · :2 awal rentang · :3 akhir rentang
-- Bind cabang Personal Accident:
--   :4 kode Group Panel PA · :5 awal rentang · :6 akhir rentang
-- Bind paginasi:
--   :7 offset · :8 jumlah baris
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
  FROM (SELECT p.CLAIMID                   AS REFERENCE,
               p.CLAIMID                   AS CASE_ID,
               p.POLICY_NO                 AS POLICY_NUMBER,
               p.QQ_NAME                   AS INSURED_NAME,
               p.TGL_KIRIM_PUCL            AS SENT_AT,
               p.KOMENTAR_ANALISATOR       AS ANALYST_NOTE,
               p.TGL_CETAK_DOKUMEN_PUCL    AS LETTER_PRINTED_AT,
               p.RCL_PUCL                  AS TRACK_CODE,
               p.STATUS_CLAIM              AS CLAIM_STATUS
          FROM POOLDATA.TC_PNC_PUCL p
         WHERE p.ASSIGNED_OPERATOR_ID = :1
           AND p.TGL_KIRIM_PUCL >= TO_DATE(:2, 'YYYY-MM-DD')
           AND p.TGL_KIRIM_PUCL < TO_DATE(:3, 'YYYY-MM-DD') + INTERVAL '1' DAY
        UNION
        SELECT p.CLAIMID                   AS REFERENCE,
               p.CLAIMID                   AS CASE_ID,
               p.POLICY_NO                 AS POLICY_NUMBER,
               p.QQ_NAME                   AS INSURED_NAME,
               p.TGL_KIRIM_PUCL            AS SENT_AT,
               p.KOMENTAR_ANALISATOR       AS ANALYST_NOTE,
               p.TGL_CETAK_DOKUMEN_PUCL    AS LETTER_PRINTED_AT,
               p.RCL_PUCL                  AS TRACK_CODE,
               p.STATUS_CLAIM              AS CLAIM_STATUS
          FROM POOLDATA.TC_PNC_PUCL p
         WHERE p.GROUPPANEL = :4
           AND p.TGL_KIRIM_PUCL >= TO_DATE(:5, 'YYYY-MM-DD')
           AND p.TGL_KIRIM_PUCL < TO_DATE(:6, 'YYYY-MM-DD') + INTERVAL '1' DAY) r
 ORDER BY r.SENT_AT DESC, r.CASE_ID DESC
OFFSET :7 ROWS FETCH NEXT :8 ROWS ONLY

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
     + COUNT(p.KETERANGAN3)
     + COUNT(p.ID_OBJECT)
     + COUNT(p.ID_COVERAGE)
     + COUNT(p.ID_ADJUSTMENT) AS PROBE
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
-- Memastikan kolom yang dipakai LAPORAN HARIAN terbaca dari tabel datar `TC_PNC_PUCL`.
--
-- Sejak 2026-10-08 laporan harian tidak lagi membaca tabel Pega. Pemeriksaannya tetap
-- terpisah dari check_rclpucl karena kolom yang dipakainya berbeda — `GROUPPANEL`,
-- `ASSIGNED_OPERATOR_ID`, `STATUS_CLAIM` tidak menyaring ketiga tab — dan bila salah satunya
-- tidak ada, yang gagal HANYA tombol unduh tab "Cetak Surat", bukan layarnya.
--
-- Satu kolom hasil, karena pemanggilnya memindai satu nilai. Keempat kolom disebut di WHERE:
-- Oracle tetap mem-parse namanya dan menjawab ORA-00904 bila salah satunya tidak ada.
SELECT COUNT(*) AS PROBE
  FROM POOLDATA.TC_PNC_PUCL p
 WHERE 1 = 0
   AND p.GROUPPANEL IS NULL
   AND p.ASSIGNED_OPERATOR_ID IS NULL
   AND p.STATUS_CLAIM IS NULL
   AND p.TGL_KIRIM_PUCL IS NULL

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
       p.QQ_NAME                        AS INSURED_PARTY,
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
       -- "Tanggal Kelengkapan Dokumen" pada tab Penerimaan Dokumen.
       --
       -- Properti section-nya `TanggalTerimaDokumenPUCL` (3 kemunculan di
       -- `SectionPenerimaanDokumenPUCL`), dan kolomnya ADA di tabel datar — sehingga isian ini
       -- TIDAK lagi bertanda "di clipboard Pega". Ditetapkan Work Owner 2026-10-01.
       p.TGL_TERIMA_DOKUMEN_PUCL        AS DOCUMENT_COMPLETE_AT,
       -- Ketiga parameter TERSEMBUNYI yang dikirim ke `PUCLPost` pada setiap tombol yang
       -- memanggilnya: Download Dokumen, Tolak Klaim, dan kedua tombol Kirim.
       --
       -- Sel-nya di `SectionPenerimaanDokumenPUCL` tidak berlabel —
       -- `.ClaimData.PUCLStatus.IDObject`, `.IDCoverage`, `.IDAdjustment` — dan tidak digambar
       -- di layar, sama seperti di Pega.
       --
       -- Ketiganya DITAMBAHKAN Work Owner ke `TC_PNC_PUCL` pada 2026-10-01, menggantikan nilai
       -- penampung `"1"` yang dipakai sehari sebelumnya. Dibaca langsung: `VARCHAR2(100)`,
       -- ketiganya terisi pada baris yang ada.
       --
       -- Ketiganya BELUM ada di `Database/CREATE_TABLE_3.SQL` — selisih yang sama dengan
       -- keempat kolom surat (§84.3), dan ditangani dengan cara yang sama lewat
       -- `letterColumnsBeyondTheSharedDDL`.
       p.ID_OBJECT                      AS ACTION_ID_OBJECT,
       p.ID_COVERAGE                    AS ACTION_ID_COVERAGE,
       p.ID_ADJUSTMENT                  AS ACTION_ID_ADJUSTMENT,
       -- "Email Tertanggung" pada tab yang sama — properti `EmailLod` (4 kemunculan).
       --
       -- ========================================================================
       -- SATU-SATUNYA isian yang diambil dari LUAR tabel datar, dan gabungannya
       -- memakai CLAIMNO — BUKAN CLAIMID
       -- ========================================================================
       -- Work Owner menetapkan "ngelink pakai CLAIMID saja". Dijalankan apa adanya, gabungan
       -- itu mengembalikan NOL baris. Dihitung langsung 2026-10-01:
       --
       --   T_CLAIM_PNC.CLAIMID berawalan 'ASM-FW-GCNMFW-WORK '   2.170 dari 2.206
       --   TC_PNC_PUCL.CLAIMID berawalan sama                         0
       --   c.CLAIMID = p.CLAIMID                                      0 cocok
       --   c.CLAIMNO = p.CLAIMID                                      1 cocok  <- ini
       --
       -- Jadi KEDUA kolom bernama `CLAIMID` menyimpan hal yang BERBEDA: yang di tabel Pega
       -- membawa kunci teknis berawalan nama kelas (utang teknis §4.1 `CLAUDE.md`), yang di
       -- tabel datar membawa nomor case polos. Yang menyimpan nomor case di `T_CLAIM_PNC`
       -- adalah `CLAIMNO`.
       --
       -- Maksud keputusannya tetap dijalankan — SATU kunci saja, yaitu penanda klaim; yang
       -- dikoreksi hanya kolom mana yang benar-benar menyimpannya.
       (SELECT c.EMAIL_LOD
          FROM POOLDATA.T_CLAIM_PNC c
         WHERE TRIM(c.CLAIMNO) = TRIM(p.CLAIMID)
         FETCH FIRST 1 ROWS ONLY)       AS INSURED_EMAIL,
       -- Grid "TANGGAL TERIMA DOKUMEN" — BARIS PERTAMA saja.
       --
       -- ========================================================================
       -- Daftarnya page list, dan page list itu TIDAK punya tabel
       -- ========================================================================
       -- Grid-nya terikat `.ClaimData.PUCLStatus.DateReceivedDocument` berkelas
       -- `ASM-FW-GCNMFW-Data-DateReceivedDocument`, dengan kolom `.DateReceived` dan
       -- `.Remarks`. Katalog Oracle dicari 2026-10-01: TIDAK ADA tabel bernama mirip itu, dan
       -- tidak ada satu pun tabel yang punya `DATERECEIVED` bersama `REMARKS`. Jadi daftarnya
       -- memang hidup di dalam blob objek kerja.
       --
       -- Yang ADA adalah DUA kolom hasil ekspos BARIS PERTAMANYA pada tabel objek kerja —
       -- pola akhiran `_1` yang sama dengan `RCL_PUCL_1`, `MSIG_1`, `LAMAKLAIM_1`:
       --
       --   RECEIVEDDATE_1  terisi 4.156 dari 7.723 · panjang maksimum 23 (satu timestamp)
       --   KETERANGAN_1    terisi    85            · panjang maksimum 32
       --
       -- Panjang maksimum itu yang memastikan keduanya BUKAN daftar berdelimiter: 23 karakter
       -- hanya cukup untuk satu timestamp.
       --
       -- Karena itu grid digambar dengan SATU baris, dan layar menyatakan bahwa baris kedua
       -- dan seterusnya tidak terbaca. Arah kegagalannya sama dengan daftar dokumen: KURANG,
       -- bukan salah.
       --
       -- SUMBER BARU (2026-10-08): tabel objek kerja Pega tidak dipakai lagi (keputusan Work
       -- Owner), sehingga kedua kolom ekspos `_1` di atas tidak dibaca lagi.
       --
       --   RECEIVEDDATE_1 -> T_CLAIM_PNC.RECEIVEDATE (DATE). Kesepadanan terukur: tanggalnya
       --                     sama pada 1.098 dari 1.121 klaim Work-PNC yang keduanya terisi,
       --                     dan 2/2 klaim PUCL yang punya objek kerja. Yang hilang: JAM-nya —
       --                     RECEIVEDATE hanya tanggal, teks Pega memuat jam GMT. Pemindai Go
       --                     tetap NullString + DisplayTimeText, sama seperti DATE_OF_LOSS.
       --   KETERANGAN_1   -> TIDAK ADA padanan; terukur 0 terisi pada 1.404 klaim Work-PNC yang
       --                     ada di T_CLAIM_PNC. Kini NULL bertipe — kolom Remarks baris
       --                     pertama selalu kosong.
       --
       -- Kunci T_CLAIM_PNC berprefix untuk klaim Pega dan polos untuk klaim PNCN, sehingga
       -- keduanya disebut; CLAIMID unik, berbeda dari CLAIMNO yang tidak unik.
       (SELECT c.RECEIVEDATE
          FROM POOLDATA.T_CLAIM_PNC c
         WHERE c.CLAIMID IN ('ASM-FW-GCNMFW-WORK ' || TRIM(p.CLAIMID), TRIM(p.CLAIMID))
         FETCH FIRST 1 ROWS ONLY)       AS RECEIVED_DATE_FIRST,
       CAST(NULL AS VARCHAR2(4000))     AS RECEIVED_NOTE_FIRST,
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
-- ============================================================================
-- DAN SEBALIKNYA: ADA BARIS YANG PEGA TIDAK PERNAH TAMPILKAN
-- ============================================================================
-- Dibandingkan langsung 2026-10-02 pada satu klaim: `PC_LINK_ATTACHMENT` memuat 4 lampiran
-- (`PUCL`, `PUCL`, `Adjustment`, `ClaimFaceSheet`), sementara `DATA_ATTACHFILE` memuat 9 —
-- enam di antaranya bernama `duplicated.JPG` berkategori `10064`.
--
-- Keenamnya TIDAK pernah tergambar di layar Pega. Kolom `PEGA_VISIBLE` di bawah menandainya,
-- dan lapisan atas yang memutuskan apa yang digambar.
--
-- Nama kategori dicari lewat LEFT JOIN, mengikuti alasan yang sama seperti `inboxpladla`:
-- satu kode yang tidak ada di master tidak boleh MENYEMBUNYIKAN dokumennya.
--
-- ============================================================================
-- KEPEMILIKAN DICOCOKKAN TERHADAP TIGA KUNCI, BUKAN HANYA KUNCI OBJEK KERJA PEGA
-- ============================================================================
--
-- Bentuk sebelumnya menuntut adanya baris `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`. Klaim `PNCN.*`
-- tidak punya baris di sana (lihat `work_object_key`), sehingga SELURUH dokumennya hilang
-- dari daftar — termasuk surat yang baru saja diterbitkan tombol "Download Dokumen", dan
-- termasuk berkas yang diunggah petugas lewat layar registrasi.
--
-- Ketiga bentuk yang benar-benar dipakai sebagai `IDPEGA` karena itu disebut semuanya:
--
--	T_CLAIM_PNC.CLAIMID                      yang ditulis modul ini dan modul registrasi
--	'ASM-FW-GCNMFW-WORK ' || CLAIMNO         bentuk berprefix milik modul registrasi
--	'ASM-FW-GCNMFW-WORK ' || nomor case      yang ditulis Pega sendiri (dulu dibaca dari
--	                                         PZINSKEY tabel objek kerja; sejak
--	                                         2026-10-08 dirangkai — lihat badan kueri)
--
-- Untuk klaim Pega ketiganya menunjuk nilai yang sama, sehingga tidak ada baris yang
-- tergambar dua kali: `IN` menguji keanggotaan, bukan menggabungkan baris.
--
-- `a.IDPEGA` sengaja TIDAK dibungkus `TRIM` — membungkusnya membuat index atas kolom itu
-- tidak terpakai, sementara nilai yang disisipkan kedua modul tidak pernah berspasi tepi.
--
-- Bind: :1 :2 :3 nomor case
SELECT a.DATAID                                     AS DOCUMENT_ID,
       a.ATTACHNAME                                 AS DOCUMENT_NAME,
       a.ATTACHMIMETYPE                             AS MIME_TYPE,
       COALESCE(t.TYPE_DOCUMENT, a.CATEGORY)        AS CATEGORY_NAME,
       COALESCE(dt.DETAIL_DOCUMENT, a.SUB_CATEGORY) AS SUBCATEGORY_NAME,
       a.INPUTDATE                                  AS UPLOADED_AT,
       a.INPUTOPERATOR                              AS UPLOADED_BY,

       -- Apakah barisnya TERGAMBAR di layar lampiran Pega.
       --
       -- `GCNMGetAllAttachments` berjalan di kelas `Link-Attachment` dan menyaring atas
       -- `pyCategory` — yang isinya NAMA kategori lampiran (`Notification`, `LOD`,
       -- `ClaimFaceSheet`). Baris yang `CATEGORY`-nya kode angka (`10064`) berasal dari
       -- mekanisme LAIN, dan penyaring itu tidak akan pernah mencocokkannya.
       --
       -- Dibandingkan terhadap daftar nama yang SAMA dengan pemilih kategori dialog unggah,
       -- bukan terhadap daftar yang ditulis di sini: kategori yang ditawarkan dropdown
       -- tidak boleh membuat barisnya hilang dari daftar.
       --
       -- Dirakit sebagai LEFT JOIN ke himpunan nama, bukan `EXISTS` per baris. Bentuk
       -- `EXISTS` diukur 2026-10-02 dan memakan **1,7 detik** untuk 9 baris: ia menyapu
       -- tabel lampiran Pega sekali untuk SETIAP baris. Himpunannya dihitung sekali.
       CASE WHEN k.CATEGORY_NAME IS NOT NULL
            THEN 1 ELSE 0 END                       AS PEGA_VISIBLE
  FROM POOLDATA.DATA_ATTACHFILE a
  LEFT JOIN POOLDATA.V_LST_DOC_TYPE t
         ON t.ID = a.CATEGORY
  LEFT JOIN POOLDATA.V_LST_DET_TYPE_DOC dt
         ON dt.ID = a.SUB_CATEGORY
  -- SUMBER BARU (2026-10-08): himpunan kategori dulu disaring lewat gabungan ke tabel objek
  -- kerja Pega (kelas Work-PNC). Kini disaring `l.PXLINKEDCLASSFROM` milik tabel lampiran
  -- itu sendiri — terukur sama pada 8.649/8.649 lampiran, dan himpunan namanya identik
  -- (selisih dua arah 0). Gabungan ke T_CLAIM_PNC SENGAJA tidak dipakai: ia ikut memuat
  -- baris RCV dan menggeser himpunannya (+ReceiverDocument, -XOLFILES).
  LEFT JOIN (SELECT DISTINCT TRIM(l.PYCATEGORY) AS CATEGORY_NAME
               FROM DATAPEGA.PC_LINK_ATTACHMENT l
              WHERE l.PXLINKEDCLASSFROM = 'ASM-FW-GCNMFW-Work-PNC'
                AND l.PYCATEGORY IS NOT NULL
                AND TRIM(l.PYCATEGORY) IS NOT NULL) k
         ON k.CATEGORY_NAME = TRIM(a.CATEGORY)
 WHERE a.ATTACHFILE IS NOT NULL
   -- SUMBER BARU (2026-10-08): kunci ketiga dulu `PZINSKEY` objek kerja Pega ber-PYID = nomor
   -- case. Kunci itu terukur sama dengan 'ASM-FW-GCNMFW-WORK ' || PYID pada 2.645 dari 2.647
   -- objek Work-PNC, sehingga kini dirangkai langsung dari nomor case — tanpa tabel. Ini
   -- penting: 1.243 objek Work-PNC TIDAK ada di T_CLAIM_PNC, dan 134 dokumen (36 klaim)
   -- hanya terjangkau lewat kunci ini. Bentuk OR, bukan UNION, karena tidak ada sumber baris.
   AND (a.IDPEGA IN (SELECT c.CLAIMID
                       FROM POOLDATA.T_CLAIM_PNC c
                      WHERE TRIM(c.CLAIMNO) = TRIM(:1)
                     UNION ALL
                     SELECT 'ASM-FW-GCNMFW-WORK ' || TRIM(c.CLAIMNO)
                       FROM POOLDATA.T_CLAIM_PNC c
                      WHERE TRIM(c.CLAIMNO) = TRIM(:2))
        OR a.IDPEGA = 'ASM-FW-GCNMFW-WORK ' || TRIM(:3))
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
-- Rantai kepemilikannya memakai ketiga kunci yang sama dengan `documents`, dan karena alasan
-- yang sama: dokumen klaim `PNCN.*` tidak dapat dibuka bila kepemilikannya hanya diakui lewat
-- tabel kerja Pega. Keduanya WAJIB sejalan — daftar yang menggambar sebuah baris sementara
-- pengambilnya menolaknya adalah tombol unduh yang selalu gagal.
--
-- Bind: :1 id dokumen · :2 :3 :4 nomor case
SELECT a.ATTACHNAME     AS DOCUMENT_NAME,
       a.ATTACHMIMETYPE AS MIME_TYPE,
       a.ATTACHFILE     AS CONTENT
  FROM POOLDATA.DATA_ATTACHFILE a
 WHERE a.DATAID = :1
   -- SUMBER BARU (2026-10-08): kunci ketiga dirangkai dari nomor case, sama persis dengan
   -- `documents` — lihat catatan di sana. Keduanya WAJIB tetap sejalan.
   AND (a.IDPEGA IN (SELECT c.CLAIMID
                       FROM POOLDATA.T_CLAIM_PNC c
                      WHERE TRIM(c.CLAIMNO) = TRIM(:2)
                     UNION ALL
                     SELECT 'ASM-FW-GCNMFW-WORK ' || TRIM(c.CLAIMNO)
                       FROM POOLDATA.T_CLAIM_PNC c
                      WHERE TRIM(c.CLAIMNO) = TRIM(:3))
        OR a.IDPEGA = 'ASM-FW-GCNMFW-WORK ' || TRIM(:4))

-- name: return_to_analyst
-- Menandai klaim SUDAH SELESAI dikerjakan PUCL — tombol "Kirim Ke Analyst" dan
-- "Kirim ke PIC Teknik".
--
-- # Satu-satunya pernyataan yang MENULIS di berkas ini
--
-- Dan ia menulis `TC_PNC_PUCL`, tabel milik APLIKASI INI — bukan tabel `DATAPEGA`. Tidak ada
-- satu pun tabel engine Pega yang disentuh, sehingga `P-1` tidak dilanggar.
--
-- # Kenapa hanya SATU kolom
--
-- Karena hanya kolom itu yang menentukan apa yang dilihat petugas. `PUCLAPPROVE_1` adalah
-- penyaring tab 2 dan tab 3 (`PUCL_APPROVE <> :disetujui`); mengubahnya menjadi `'1'`
-- mengeluarkan klaim dari antrean PUCL. Kolom lain yang disentuh `PUCLPost` — tanggal cetak,
-- komentar analisator — TIDAK ikut ditulis karena tidak satu pun menentukan perpindahan ini,
-- dan menulis kolom yang tidak perlu adalah perubahan data yang tidak dapat dibenarkan.
--
-- # Kenapa klaimnya tidak hilang entah ke mana
--
-- Klaim RCL/PUCL sudah memegang baris `PC_ASSIGN_WORKLIST` Register_Flow-nya, dan pemegang
-- baris itu adalah PIC Teknik klaim tersebut. Menandainya selesai mengembalikan klaim kepada
-- orang yang memang sudah memegangnya.
--
-- # Urutan penandanya WAJIB menaik, dan tiap penanda muncul SEKALI
--
-- Bentuk pertama pernyataan ini menulis `SET PUCL_APPROVE = :2 WHERE … :1`, dengan `:2`
-- muncul DUA KALI. Penggerak Oracle yang dipakai mengikat argumen menurut URUTAN KEMUNCULAN
-- penanda, bukan menurut angkanya — sehingga nomor klaim masuk ke kolom penanda, penyaringnya
-- membandingkan `CLAIMID` dengan `'1'`, dan pernyataannya mengenai NOL baris.
--
-- Kegagalannya SENYAP: layar tetap menjawab "Klaim diteruskan ke Analyst", sementara klaimnya
-- tidak bergerak sedikit pun. Ditemukan 2026-10-01 oleh Work Owner, bukan oleh uji.
--
-- Perangkap ini sudah tercatat di berkas yang sama — lihat catatan pada `daily_report`, yang
-- MENGIKAT rentang tanggalnya dua kali alih-alih mengulang penandanya.
--
-- # Kenapa TIDAK ada lagi penyaring "hanya bila nilainya berbeda"
--
-- Bentuk pertama menyaring `PUCL_APPROVE <> :2` supaya penulisan ulang tidak terjadi. Itu
-- membuat NOL baris berarti DUA hal — klaim tidak ada, atau sudah ditandai — sehingga jumlah
-- baris tidak dapat dipakai menilai keberhasilan. Justru itulah yang menyembunyikan cacat di
-- atas.
--
-- Tanpa penyaring itu, menulis nilai yang sama dua kali tetap tidak berakibat apa-apa pada
-- DATA, dan NOL baris kini berarti SATU hal saja: klaimnya tidak ada. Itu dapat dilaporkan.
--
-- # TIGA kolom, bukan satu
--
-- Bentuk pertama hanya menulis `PUCL_APPROVE`, karena hanya itu yang dipakai kueri inbox
-- sebagai penyaring. Itu cara mencari yang salah arah: kolom yang tidak menyaring apa pun
-- menjadi tidak terlihat, padahal ia yang menjawab "klaim ini sekarang di mana".
--
-- Pembacaan ulang `Activity/PUCLPost-Act.xml` menemukan ketiganya:
--
--	langkah  4   .ClaimData.PUCLStatus.TanggalCetakDokumenPUCL := @CurrentDateTime()
--	langkah 15   .ClaimData.PUCLStatus.PUCLApprove             := 1
--	langkah 15   .ClaimData.StatusClaim                        := "1151"
--	langkah 51   .ClaimData.PUCLStatus.TanggalCetakDokumenPUCL := @CurrentDateTime()
--
-- `1151` berarti **"Analyst"** menurut master `POOLDATA.V_STS_CLAIM` — jadi tombolnya
-- MENYATAKAN klaim berpindah tangan, bukan sekadar mengosongkan antrean.
--
-- Langkah 4 dan 51 keduanya menyetel tanggal cetak, dan keduanya berjalan untuk tombol Kirim
-- — bukan hanya untuk "Download Dokumen". Ditiru apa adanya (`P-5`), meski akibatnya kolom
-- "Tanggal Cetak Surat" ikut tersetel saat klaim dikirim.
--
-- `CURRENT_TIMESTAMP`, bukan `SYSDATE` — portabel ke PostgreSQL (`09-DATABASE-STRATEGY` §4).
--
-- Bind: :1 nilai "sudah kembali ke Analyst" · :2 Status Klaim "Analyst" · :3 nomor klaim
UPDATE POOLDATA.TC_PNC_PUCL
   SET PUCL_APPROVE           = :1,
       STATUS_CLAIM           = :2,
       TGL_CETAK_DOKUMEN_PUCL = CURRENT_TIMESTAMP
 WHERE TRIM(CLAIMID) = TRIM(:3)

-- name: reject_claim
-- MENUTUP klaim sebagai ditolak — tombol "Tolak Klaim".
--
-- # Ketiga kolomnya, dan dari langkah mana
--
--	langkah  4   .ClaimData.PUCLStatus.TanggalCetakDokumenPUCL := @CurrentDateTime()
--	langkah 12   .ClaimData.PUCLStatus.PUCLApprove             := "0"
--	langkah 42   ASMForceCaseClose(WorkStatus = "Resolved-Rejected")
--	langkah 43   idem - dipanggil DUA KALI, berketerangan "(2x supaya sts jadi reject)"
--
-- Langkah 4 tanpa prekondisi, sehingga ia berjalan untuk SETIAP tombol yang memanggil
-- `PUCLPost` — termasuk tombol ini. Ditiru apa adanya (`P-5`), sama seperti pada
-- `return_to_analyst`.
--
-- Pemanggilan ganda langkah 42-43 TIDAK ditiru. Keduanya menulis nilai yang sama persis, dan
-- satu `UPDATE` menghasilkan keadaan akhir yang identik; yang di Pega menuntut dua panggilan
-- adalah mesin alur kerjanya, bukan datanya.
--
-- # `STATUS_CLAIM` sengaja TIDAK disentuh
--
-- `PUCLPost` menulisnya pada jalur ini hanya di langkah 39 — `"1143"`, Close Claim for this
-- object — dan langkah itu berprekondisi `local.isCFS=="1"`, yang baru benar bila klaimnya
-- sudah punya tanggal OS Akseptasi (langkah 9). Menuliskannya tanpa syarat akan mengubah
-- status klaim yang di Pega tidak berubah.
--
-- # Kenapa `STATUS_WORK` yang ditulis, bukan penanda lain
--
-- Karena itulah yang `ASMForceCaseClose` tulis, dan karena ketiga kueri daftar di atas kini
-- mengecualikannya — lihat `inboxrclpucl.WorkStatusRejected`. Tanpa kolom ini klaim yang
-- baru ditolak tetap duduk di tab "Kelengkapan Dokumen", karena `PUCL_APPROVE = '0'` justru
-- MENAHANNYA di sana.
--
-- Bind: :1 nilai PUCL_APPROVE sesudah ditolak · :2 status kerja ditolak · :3 nomor klaim
UPDATE POOLDATA.TC_PNC_PUCL
   SET PUCL_APPROVE           = :1,
       STATUS_WORK            = :2,
       TGL_CETAK_DOKUMEN_PUCL = CURRENT_TIMESTAMP
 WHERE TRIM(CLAIMID) = TRIM(:3)

-- name: mirror_daftar_kerja_tolak
-- Baris daftar kerja **My Inbox** klaim yang DITUTUP sebagai ditolak.
--
-- Ia sepupu `mirror_daftar_kerja`, dan dipisah karena menulis kolom yang BERBEDA:
--
--	mirror_daftar_kerja        pemilik BARU  + nama tahap  -> klaim berpindah tangan
--	mirror_daftar_kerja_tolak  pemilik KOSONG + status kerja -> klaim tidak di tangan siapa pun
--
-- `PXASSIGNEDOPERATORID` dikosongkan karena kasusnya tutup: tidak ada lagi orang yang
-- memegangnya, dan membiarkannya terisi menampilkan klaim tertutup di My Inbox petugas
-- RCL/PUCL tanpa satu pun tindakan yang dapat dilakukan padanya.
--
-- `PYSTATUSWORK` ikut ditulis supaya layar lain yang membaca tabel INI melihat keadaan yang
-- sama dengan `TC_PNC_PUCL`. Kolomnya sudah memuat `Resolved-Rejected` pada 7 baris
-- (`docs/kolom-t-claimlist-admin.md` §B.3), jadi nilainya bukan bentuk baru bagi tabel ini.
--
-- Tabel kerja Pega TIDAK disentuh (`P-1`). Klaim yang lahir di Pega karena itu tetap
-- tergambar di inbox Pega sampai Pega sendiri menutupnya — konsekuensi masa paralel yang
-- sama dengan seluruh tindakan lain di modul ini.
--
-- NOL BARIS BUKAN GALAT: tidak setiap klaim punya baris di tabel ini.
--
-- Bind: :1 status kerja ditolak · :2 label tahap · :3 nomor klaim
UPDATE POOLDATA.T_CLAIMLIST_ADMIN
   SET PXASSIGNEDOPERATORID = NULL,
       PYSTATUSWORK         = :1,
       PXTASKLABEL          = :2
 WHERE PYID = :3

-- name: save_receipt
-- Menyimpan kedua isian Penerimaan Dokumen yang dapat diketik — tombol "Save".
--
-- # Kenapa hanya dua kolom
--
-- Karena hanya dua sel section yang `pyReadOnly false` DAN hidup di tabel ini. Yang ketiga,
-- Email Tertanggung, ada di `POOLDATA.T_CLAIM_PNC.EMAIL_LOD` — tabel lain yang modul ini tidak
-- tulis. Lihat `inboxrclpucl.ReceiptInput`.
--
-- # Kenapa TIDAK ikut menyentuh PUCL_APPROVE
--
-- Karena "Save" di layar lama memang tidak memanggil `PUCLPost` sama sekali — ia menempuh
-- `SaveInputRegisterDetail2`, yang berakhir pada `Obj-Save`. Menyimpan BUKAN memindahkan;
-- klaimnya tetap menjadi pekerjaan PUCL sesudahnya.
--
-- Bind: :1 catatan untuk Analyst · :2 tanggal kelengkapan dokumen · :3 nomor klaim
UPDATE POOLDATA.TC_PNC_PUCL
   SET KOMENTAR_PUCL          = :1,
       TGL_TERIMA_DOKUMEN_PUCL = :2
 WHERE TRIM(CLAIMID) = TRIM(:3)

-- name: mark_letter_printed
-- Menandai surat RCL/PUCL sudah diterbitkan — tombol "Download Dokumen".
--
-- # Ketiga kolomnya, dan dari mana nomor langkahnya
--
--	langkah  4   .ClaimData.PUCLStatus.TanggalCetakDokumenPUCL := @CurrentDateTime()
--	langkah 17   .ClaimData.PUCLStatus.StatusCase              := param.statusCase  -- "1"
--	langkah 17   .ClaimData.StatusClaim                        := "1157"
--	langkah 51   .ClaimData.PUCLStatus.TanggalCetakDokumenPUCL := @CurrentDateTime()
--
-- `1157` berarti "Document Waiting RCL/PUCL" menurut master — klaimnya menunggu kelengkapan
-- dokumen, tepat menggambarkan tahap sesudah suratnya dicetak.
--
-- # Kolom tanggal inilah yang MEMINDAHKAN klaim
--
-- Penyaring tab "Cetak Surat" adalah `TGL_CETAK_DOKUMEN_PUCL IS NULL`. Mengisinya memindahkan
-- klaim ke tab "Kelengkapan Dokumen" — dan itulah akibat yang dirasakan petugas, bukan PDF-nya.
--
-- `PUCL_APPROVE` TIDAK disentuh: langkah 15 dan 16 berprekondisi `param.Status` 1 atau 0,
-- sementara "Download Dokumen" mengirimnya KOSONG. Klaimnya tetap menjadi pekerjaan PUCL.
--
-- Bind: :1 penanda sudah dicetak · :2 Status Klaim "Document Waiting" · :3 nomor klaim
UPDATE POOLDATA.TC_PNC_PUCL
   SET TGL_CETAK_DOKUMEN_PUCL = CURRENT_TIMESTAMP,
       STATUS_CASE            = :1,
       STATUS_CLAIM           = :2
 WHERE TRIM(CLAIMID) = TRIM(:3)

-- name: work_object_key
-- Kunci klaim yang disimpan `DATA_ATTACHFILE.IDPEGA` dan `LIST_HISTORY_CLAIM_PNC.CASEID`.
--
-- Dibutuhkan unggahan dan penerbitan surat: kueri `documents` menggabungkannya kembali lewat
-- kolom yang sama. Baris yang `IDPEGA`-nya tidak cocok tidak akan pernah muncul di daftar
-- dokumen klaimnya.
--
-- # TABEL PEGA TIDAK LAGI MENJADI PENGGERAKNYA — koreksi yang sama seperti `technical_pic`
--
-- Bentuk sebelumnya membacanya dari `DATAPEGA.PC_ASM_FW_GCNMFW_WORK.PZINSKEY`. Tabel itu
-- hanya memuat klaim yang LAHIR DI PEGA; klaim yang dibuka aplikasi ini — bernomor `PNCN.*`,
-- dan sejak modul registrasi punya tombol "Kirim ke RCL/PUCL" klaim seperti itu MEMANG masuk
-- ke antrean layar ini — tidak punya baris di sana sama sekali.
--
-- Akibatnya `AddDocument` menjawab `ErrClaimNotFound` untuk setiap klaim PNCN, dan karena
-- penerbitan surat sengaja TIDAK membatalkan tindakannya (lihat Service.PerformAction),
-- kegagalannya muncul ke petugas sebagai *"Berkas suratnya TIDAK berhasil diterbitkan kali
-- ini"* — klaimnya berpindah tab, suratnya tidak pernah terbit, dan sebabnya tidak terbaca
-- di layar mana pun. `RecordHistory` gagal diam-diam dengan sebab yang sama.
--
-- `T_CLAIM_PNC` memuat KEDUANYA, dan kolomnya sepadan satu lawan satu:
--
--	klaim       CLAIMNO       CLAIMID                       PZINSKEY Pega
--	Pega        PNC-2067      ASM-FW-GCNMFW-WORK PNC-2067   ASM-FW-GCNMFW-WORK PNC-2067
--	aplikasi    PNCN.26.28    PNCN.26.28                    (tidak ada)
--
-- `CLAIMID` karena itu menggantikan `PZINSKEY` apa adanya — nilainya SAMA PERSIS untuk klaim
-- Pega — dan sekaligus menjadi nilai yang benar untuk klaim PNCN. Ia pula nilai yang dipakai
-- modul `registrasi` sebagai salah satu dari tiga kunci lampirannya (`lampiran_daftar`),
-- sehingga surat yang terbit di sini terbaca pula di tab dokumen layar registrasi.
--
-- Ia MEMBACA, dan itu sah: `P-1` membatasi yang MENULIS.
--
-- Bind: :1 nomor klaim
SELECT c.CLAIMID AS WORK_KEY
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE TRIM(c.CLAIMNO) = TRIM(:1)
 FETCH FIRST 1 ROWS ONLY

-- name: next_attachment_number
-- Nomor urut berikutnya untuk `DATAID` sebuah lampiran.
--
-- `SET_ATTACHMENT_64BIT.prc` menyusun `DATAID` begini:
--
--	select_sequence('ATTACHFILE_SEQ');
--	count_ATTACH := ATTACHFILE_SEQ.nextval;
--	INSERT INTO C_COUNTER_ATTACHMENT (KEY, YEAR, RUNNO) VALUES (new_uuid, to_char(sysdate,'yy'), count_ATTACH);
--	SELECT year || lpad(runno,10,'0') INTO tDATAID FROM C_COUNTER_ATTACHMENT WHERE key = pkey;
--
-- Yang ditiru NOMOR dan BENTUKNYA, bukan cara mengambilnya kembali: `DATAID` disusun di Go
-- dari kedua nilai di bawah, sehingga tidak perlu membaca ulang baris yang baru disisipkan.
--
-- `select_sequence` TIDAK dipanggil — ia tidak ada di antara 62 berkas `Database/` yang
-- diterima (§150). Namanya menyiratkan ia menyiapkan sequence-nya; bila ternyata ia mengatur
-- ulang penomoran tiap tahun, `DATAID` kami tetap unik karena memuat tahunnya.
SELECT TO_CHAR(CURRENT_DATE, 'RR') AS YEAR_TWO,
       ATTACHFILE_SEQ.NEXTVAL      AS RUN_NO
  FROM DUAL

-- name: insert_attachment_counter
-- Baris pencacah lampiran — ditiru dari `SET_ATTACHMENT_64BIT.prc`.
--
-- Ia TIDAK dibaca modul ini sama sekali; `DATAID` sudah disusun di Go. Ia tetap ditulis
-- karena procedure aslinya menulisnya, dan tabel pencacah yang berlubang akan menyesatkan
-- siapa pun yang kelak menelusuri penomoran lampiran.
--
-- Bind: :1 kunci baris · :2 dua digit tahun · :3 nomor urut
INSERT INTO C_COUNTER_ATTACHMENT (KEY, YEAR, RUNNO) VALUES (:1, :2, :3)

-- name: insert_attachment
-- Satu baris lampiran klaim.
--
-- # Kenapa ISINYA ditulis ke kolom BLOB
--
-- `DATA_ATTACHFILE` menampung DUA mekanisme, dan Pega membaca keduanya. `GetAttachmentFromDB_Sql`
-- mengambil `ATTACHFILE` maupun `IMAGEID` dari baris yang sama, dengan
-- `CASE WHEN attachfile IS NULL THEN NULL ELSE pooldata.base64encode(attachfile) END` —
-- percabangan itu ADA justru karena kedua keadaan memang terjadi.
--
-- Unggahan Pega hari ini menempuh mekanisme kedua: berkasnya naik ke layanan penyimpanan luar
-- dan hanya `IMAGEID` yang tersimpan. Jalur itu TIDAK dapat kami tempuh — nama host
-- layanannya tidak dapat diterjemahkan dari peladen ini, dan `general.T_FOLDER_STORAGE`
-- serta `general.T_STORAGE_IMAGE` tidak terlihat oleh akun basis data kami (§154).
--
-- Yang ditempuh karena itu mekanisme PERTAMA, yang seluruhnya berada di tabel yang dapat kami
-- akses — dan yang masih dibaca Pega.
--
-- `IMAGEID` sengaja tidak diisi — ia milik mekanisme kedua. `CATEGORY` DIISI sejak
-- 2026-10-02, berupa NAMA kategori lampiran yang dipilih petugas; kosong hanya bila ia tidak
-- memilih apa pun.
--
-- `SUB_CATEGORY` tidak diisi: dialog `SetUploadDocPUCL` di Pega pun hanya punya SATU pemilih.
--
-- Bind: :1 dataid · :2 pelaku · :3 nama berkas · :4 keterangan · :5 jenis isi · :6 isi ·
--       :7 kunci objek kerja
INSERT INTO POOLDATA.DATA_ATTACHFILE
       (DATAID, INPUTDATE, INPUTOPERATOR, ATTACHNAME, ATTACHNOTE,
        ATTACHMIMETYPE, ATTACHFILE, IDPEGA, CATEGORY)
VALUES (:1, CURRENT_TIMESTAMP, :2, :3, :4, :5, :6, :7, :8)

-- name: document_categories
-- Pilihan kolom "Category" pada dialog unggah.
--
-- # Kenapa dari tabel lampiran Pega, bukan dari master jenis dokumen
--
-- Karena di Pega pilihan ini adalah KATEGORI LAMPIRAN, bukan jenis dokumen. Nilainya berupa
-- nama (`AcceptanceNote`, `ClaimFaceSheet`, `LOD`), dan yang mendefinisikannya adalah rule
-- `Rule-Obj-AttachmentCategory` — tipe rule yang TIDAK ADA di export sama sekali (`R-16`).
--
-- Karena rule-nya tidak ada, daftarnya diturunkan dari kategori yang BENAR-BENAR DIPAKAI
-- lampiran klaim PNC. Batasnya satu, dan nyata: kategori yang sudah didefinisikan tetapi
-- belum pernah dipakai TIDAK muncul.
--
-- `V_LST_DOC_TYPE` — yang ditunjuk `BrowseLstDocType_RD` — diperiksa dan TIDAK dapat dipakai:
-- isinya 7 baris, 6 di antaranya tanpa nama sama sekali.
--
-- Pengurutannya `UPPER(...)` supaya `AttachAIFILE` dan `ATTACHTEMPS` bersebelahan, persis
-- seperti di layar lama. Mengurutkan apa adanya menaruh seluruh nama berhuruf besar lebih
-- dulu, dan daftarnya tidak lagi terbaca sebagai daftar yang sama.
--
-- MEMBACA tabel engine Pega, tidak menulisnya — `P-1` melarang menulis, bukan membaca.
--
-- SUMBER BARU (2026-10-08): dulu disaring lewat gabungan ke tabel objek kerja Pega (kelas
-- Work-PNC). Kini `a.PXLINKEDCLASSFROM` milik tabel lampiran itu sendiri — terukur sama
-- pada 8.649/8.649 lampiran, 30 nama kategori, selisih himpunan dua arah 0. Sama persis
-- dengan himpunan PEGA_VISIBLE pada `documents`.
SELECT TRIM(a.PYCATEGORY) AS CATEGORY_NAME
  FROM DATAPEGA.PC_LINK_ATTACHMENT a
 WHERE a.PXLINKEDCLASSFROM = 'ASM-FW-GCNMFW-Work-PNC'
   AND a.PYCATEGORY IS NOT NULL
   AND TRIM(a.PYCATEGORY) IS NOT NULL
 GROUP BY TRIM(a.PYCATEGORY)
 ORDER BY UPPER(TRIM(a.PYCATEGORY))

-- name: insert_history
-- Menulis SATU baris riwayat klaim — pengganti `InsertHistoryClaimPNC`.
--
-- # Dari mana pernyataan ini berasal
--
-- Dari procedure-nya, BUKAN dari rule yang memanggilnya. `Database/
-- PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc` memuat satu pernyataan, dan inilah dia:
--
--	INSERT INTO LIST_HISTORY_CLAIM_PNC (CASEID, CREATEDATETIME, STATUSNOTE, USERUPDATE)
--	VALUES (CaseID, CURRENT_TIMESTAMP, StatusNote, UserUpdate);
--
-- Logikanya naik ke Go sesuai `D-02`; procedure-nya tidak dipanggil.
--
-- # Nama parameter di rule Pega tidak boleh dipercaya
--
-- `RDB List/InsertHistoryClaimPNC-SQL.xml` memanggilnya dengan nama properti clipboard
-- `POLICY_NO`, `BUSINESS_CODE`, `BRANCH_CODE` — dan pemetaannya POSISIONAL, bukan menurut
-- nama. Posisi 1 adalah CaseID, bukan nomor polis. Membaca namanya akan menulis nomor polis
-- ke kolom `CASEID`, dan barisnya tersimpan rapi di tempat yang salah tanpa satu pun galat.
--
-- # `CASEID` berisi `PZINSKEY`, bukan nomor klaim
--
-- `PUCLPost` langkah 35 mengirim `caseID = pyWorkPage.pzInsKey`, yang berbentuk
-- `ASM-FW-GCNMFW-WORK PNC-xxxx`. Pemanggil membacanya lebih dulu lewat `work_object_key`.
--
-- # Kenapa `P-1` tidak dilanggar
--
-- `LIST_HISTORY_CLAIM_PNC` tabel bisnis `POOLDATA`, bukan tabel engine Pega. Ia pun tidak
-- dibaca satu rule pun di seluruh export — hanya ditulis. Baris yang kami tambahkan
-- berdampingan dengan baris Pega, tidak menimpanya.
--
-- `CURRENT_TIMESTAMP`, bukan `SYSDATE` — portabel ke PostgreSQL, dan procedure-nya pun
-- memakai `CURRENT_TIMESTAMP`.
--
-- Bind: :1 PZINSKEY objek kerja · :2 teks riwayat · :3 pelaku
INSERT INTO POOLDATA.LIST_HISTORY_CLAIM_PNC (CASEID, CREATEDATETIME, STATUSNOTE, USERUPDATE)
VALUES (:1, CURRENT_TIMESTAMP, :2, :3)

-- name: technical_pic
-- PIC Teknik klaim beserta kunci objek kerjanya.
--
-- Keduanya dibaca SEKALIGUS karena keduanya dibutuhkan perpindahan tahap, dan membacanya
-- dua kali membuka celah: nilainya dapat berubah di antara kedua pembacaan.
--
-- # Sumbernya `PICTEKNIK`, BUKAN `USERTEKNIS_1`
--
-- Bentuk pertama pernyataan ini membaca `DATAPEGA.PC_ASM_FW_GCNMFW_WORK.USERTEKNIS_1`.
-- Work Owner mengoreksinya 2026-10-02: **`USERTEKNIS_1` diambil dari `PICTEKNIK`**, dan
-- export membuktikannya — kolomnya dialiaskan tepat begitu:
--
--	POOLDATA.T_CLAIM_PNC.PICTEKNIK  AS "UserTeknis"
--	  `RDB List/GcnmSalvageData_OS_SQL-SQL.xml`, `GcnmSalvageData_ekonomisdanTba-SQL.xml`
--
-- Jadi `USERTEKNIS_1` adalah SALINAN yang Pega ekspos dari properti clipboard, sementara
-- `PICTEKNIK` adalah tempat nilainya benar-benar tinggal. Membaca salinan berarti bergantung
-- pada Pega sempat menuliskannya — dan klaim yang dibuka aplikasi ini tidak melewati Pega.
--
-- Ini contoh lain dari alias menyesatkan yang `D-19` tetapkan untuk tidak dibawa: nama kolom
-- dan nama alias di sistem lama memang tidak saling menjelaskan.
--
-- # TABEL PEGA TIDAK LAGI MENJADI PENGGERAKNYA — DAN ITU MEMPERBAIKI CACAT
--
-- Bentuk sebelumnya menggerakkan kueri ini dari `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`:
--
--	FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
--	LEFT JOIN POOLDATA.T_CLAIM_PNC c ON TRIM(c.CLAIMNO) = TRIM(w.PYID)
--	WHERE TRIM(w.PYID) = TRIM(:1)
--
-- Tabel itu hanya memuat klaim yang LAHIR DI PEGA. Klaim yang dibuka aplikasi ini —
-- bernomor `PNCN.*` — tidak punya baris di sana sama sekali (diperiksa 2026-10-04:
-- `PNC-2067` ada, `PNCN.26.28` tidak). Akibatnya tombol "Kirim Ke Analyst" dan "Kirim ke
-- PIC Teknik" MENOLAK setiap klaim PNCN dengan "klaim tidak ditemukan", padahal klaimnya
-- ada dan sedang dibuka di layar yang sama.
--
-- `T_CLAIM_PNC` memuat keduanya, dan kolomnya sepadan satu lawan satu:
--
--	klaim       CLAIMNO       CLAIMID                       PZINSKEY Pega
--	Pega        PNC-2067      ASM-FW-GCNMFW-WORK PNC-2067   ASM-FW-GCNMFW-WORK PNC-2067
--	aplikasi    PNCN.26.28    PNCN.26.28                    (tidak ada)
--
-- `CLAIMID` karena itu menggantikan `PZINSKEY` apa adanya — nilainya SAMA PERSIS untuk
-- klaim Pega, diperiksa langsung — dan sekaligus menjadi nilai yang benar untuk klaim
-- PNCN. Ia pula yang sudah dipakai `CPNC_TUGAS.KLAIM_ID` pada kedua jenis klaim.
--
-- Akibat yang disadari: klaim yang TIDAK punya baris `T_CLAIM_PNC` kini ditolak sebagai
-- "klaim tidak ditemukan", bukan "PIC Teknik tidak diketahui". Itu perubahan sebab
-- penolakan, bukan perubahan apakah ia ditolak — bentuk lama pun menolaknya, hanya dengan
-- kalimat yang berbeda.
--
-- Ia MEMBACA, dan itu sah: `P-1` membatasi yang MENULIS.
--
-- Bind: :1 nomor klaim
SELECT c.CLAIMID    AS WORK_KEY,
       c.PICTEKNIK  AS TECHNICAL_PIC
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE TRIM(c.CLAIMNO) = TRIM(:1)

-- name: close_open_tasks
-- Menutup SELURUH tugas klaim yang masih terbuka — langkah "Finish Assignment".
--
-- # Kenapa seluruhnya, bukan satu
--
-- Karena yang hendak dipastikan adalah keadaan SESUDAHNYA: klaim ini tidak lagi menjadi
-- pekerjaan siapa pun di tahap lama. Menutup "tugas terbuka pertama" meninggalkan sisanya
-- bila ternyata ada lebih dari satu, dan sisa itu membuat klaim tetap tergambar di inbox
-- lama tanpa satu pun galat.
--
-- Tugas TIDAK dihapus — `SELESAI_PADA` yang diisi. Riwayat siapa mengerjakan apa adalah
-- bagian jejak audit klaim (`D-66`, `D-59`).
--
-- Nol baris BUKAN galat: klaim yang dimulai di Pega belum pernah punya tugas di tabel ini.
--
-- Bind: :1 waktu selesai · :2 alasan · :3 nomor klaim
UPDATE CPNC_TUGAS
   SET SELESAI_PADA   = :1,
       ALASAN_SELESAI = :2
 WHERE NOMOR_KLAIM    = :3
   AND SELESAI_PADA IS NULL

-- name: tahap_tugas_terbuka
-- Tahap tugas yang MASIH terbuka pada sebuah klaim, bila ada.
--
-- Dipakai menjaga "Kirim Ke Analyst" dari berjalan dua kali: klaim yang sudah di
-- `kirim-analis` tidak punya apa pun untuk dipindahkan, dan menutup-lalu-membuka tahap
-- yang sama hanya menambah baris riwayat tanpa memindahkan klaimnya.
--
-- Nol baris BUKAN galat — klaim yang masih dikerjakan Pega belum punya tugas di sini,
-- dan untuk klaim seperti itu tindakannya justru sah.
SELECT TAHAP FROM CPNC_TUGAS
 WHERE NOMOR_KLAIM = :1
   AND SELESAI_PADA IS NULL
 FETCH NEXT 1 ROWS ONLY

-- name: mirror_daftar_kerja
-- Baris daftar kerja **My Inbox** klaim ini — `POOLDATA.T_CLAIMLIST_ADMIN`.
--
-- # Kenapa pernyataan ini ada
--
-- Layar "My Inbox" (`MENU_ID 51`) TIDAK membaca `CPNC_TUGAS`. Ia menyaring
-- `PXASSIGNEDOPERATORID` pada tabel ini. Memindahkan tugas tanpa memperbarui baris ini
-- membuat klaim berpindah di satu tempat dan tidak berpindah di tempat lain: inbox modul
-- registrasi menampilkannya pada pemilik baru, sementara My Inbox masih menampilkannya
-- pada pemilik LAMA — dan tidak ada galat yang menandainya.
--
-- Terukur pada `PNCN.26.28` (2026-10-04): barisnya masih `ServicePNC` / `RCLDokter`
-- setelah tugasnya berpindah, sehingga klaim tidak pernah sampai ke My Inbox analis.
--
-- Modul registrasi menulis baris yang sama lewat `inboxentry.sql`; keduanya mengisi kedua
-- kolom ini dari sumber yang sama — pemilik tugas yang sedang berjalan dan nama tahapnya.
--
-- NOL BARIS BUKAN GALAT. Klaim yang lahir di Pega belum tentu punya baris di sini
-- (`PNC-2067` tidak punya), dan tabel ini memang hanya memuat klaim yang pernah lewat
-- proses pengisinya.
--
-- Bind: :1 pemilik tugas baru · :2 nama tahap · :3 nomor klaim
UPDATE POOLDATA.T_CLAIMLIST_ADMIN
   SET PXASSIGNEDOPERATORID = :1,
       PXTASKLABEL          = :2
 WHERE PYID = :3

-- name: open_task
-- Membuka satu tugas baru pada sebuah tahap — akibat lompatan ticket.
--
-- `KLAIM_ID` diisi kunci objek kerja Pega (`PZINSKEY`), sama seperti kolom `IDPEGA` pada
-- tabel lampiran. Ia yang menautkan tugas ke klaimnya pada data yang sudah ada.
--
-- `DIAMBIL_PADA` diisi bersamaan dengan `PEMILIK`: tugas Worklist sudah bertuan sejak lahir,
-- sehingga tidak ada yang perlu "mengambilnya" (`D-26`).
--
-- Bind: :1 id · :2 klaim id · :3 nomor klaim · :4 tahap · :5 antrean · :6 pemilik
--       :7 dibuat pada · :8 diambil pada
INSERT INTO CPNC_TUGAS
       (ID, KLAIM_ID, NOMOR_KLAIM, TAHAP, ANTREAN, WORKBASKET, PEMILIK,
        DIBUAT_PADA, DIAMBIL_PADA, SELESAI_PADA, ALASAN_SELESAI)
VALUES (:1, :2, :3, :4, :5, NULL, :6, :7, :8, NULL, NULL)

-- name: letters_missing
-- Berapa klaim yang DITANDAI suratnya tercetak padahal suratnya tidak pernah terbit.
--
-- # Kenapa keadaan ini perlu dihitung sama sekali
--
-- Tombol "Download Dokumen" menempuh dua langkah: menandai, lalu menerbitkan PDF. Kegagalan
-- langkah kedua SENGAJA tidak membatalkan langkah pertama (lihat `Service.PerformAction`),
-- sehingga klaimnya tetap berpindah tab dan petugas hanya membaca satu kalimat yang lewat.
-- Sesudah itu tidak ada apa pun di layar mana pun yang membedakan klaim bersurat dari klaim
-- yang suratnya tidak ada — keduanya duduk berdampingan di tab "Kelengkapan Dokumen".
--
-- Satu-satunya jejaknya adalah baris `Warn` di log peladen, dan log dibaca ketika seseorang
-- sudah curiga. Hitungan ini yang membuat kecurigaan itu tidak perlu lebih dulu ada.
--
-- Sebab yang sudah diketahui — `work_object_key` yang menggerakkan dirinya dari tabel kerja
-- Pega, sehingga setiap klaim `PNCN.*` gagal — sudah diperbaiki 2026-10-05. Hitungan ini
-- TETAP dipasang: sebab lain tetap mungkin (basis data menolak, perender gagal), dan
-- perilakunya yang membiarkan kegagalan lewat tidak berubah.
--
-- # Yang dihitung dan yang TIDAK
--
-- Hanya klaim yang MASIH pekerjaan PUCL (`PUCL_APPROVE <> '1'`) dan non-MSIG. Klaim yang
-- sudah dikirim ke Analyst memang mengisi kolom tanggal yang sama, tetapi ia sudah berpindah
-- tahap — melaporkannya di sini hanya akan menghasilkan angka yang tidak dapat ditindak.
--
-- Lampiran suratnya dicari lewat KEDUA bentuk kunci yang dipakai aplikasi ini, dengan alasan
-- yang sama seperti kueri `documents`.
SELECT COUNT(*) AS TANPA_SURAT
  FROM POOLDATA.TC_PNC_PUCL p
 WHERE p.TGL_CETAK_DOKUMEN_PUCL IS NOT NULL
   AND (p.PUCL_APPROVE IS NULL OR TRIM(p.PUCL_APPROVE) <> '1')
   AND p.MSIG IS NULL
   AND NOT EXISTS (SELECT 1
                     FROM POOLDATA.DATA_ATTACHFILE a
                    WHERE a.ATTACHFILE IS NOT NULL
                      AND TRIM(a.CATEGORY) = 'Notification'
                      AND TRIM(a.ATTACHNAME) IN ('PUCL.pdf', 'RCL.pdf', 'Notification.pdf')
                      AND a.IDPEGA IN (SELECT c.CLAIMID
                                         FROM POOLDATA.T_CLAIM_PNC c
                                        WHERE TRIM(c.CLAIMNO) = TRIM(p.CLAIMID)
                                       UNION ALL
                                       SELECT 'ASM-FW-GCNMFW-WORK ' || TRIM(c.CLAIMNO)
                                         FROM POOLDATA.T_CLAIM_PNC c
                                        WHERE TRIM(c.CLAIMNO) = TRIM(p.CLAIMID)))
