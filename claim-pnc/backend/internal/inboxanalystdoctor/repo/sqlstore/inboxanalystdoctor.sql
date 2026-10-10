-- Kueri modul Inbox Analyst Doctor.
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
-- DUA KOLOM TEBAKAN SUDAH DIUJI KE ORACLE — KEDUANYA TIDAK ADA (2026-10-09)
-- ============================================================================
--
-- Versi sebelumnya berkas ini menyaring `w.ISCOMPLIANCETRANSFER_1 = 2` dan mengambil
-- `w.ANALYSTDOCTORREMAKS_1`. Kedua nama itu adalah TEBAKAN yang mengikuti konvensi `_1`,
-- karena `Report Definition/InboxAnalystDoctor_RD-RD.xml` menandai kedua propertinya
-- sendiri `<pzPropertyType>unexposed</pzPropertyType>` — properti tak terekspos hidup di
-- dalam blob Pega, bukan sebagai kolom SQL.
--
-- Tebakan itu kini sudah diuji langsung ke katalog Oracle:
--
--   SELECT COLUMN_NAME, DATA_TYPE, NUM_DISTINCT FROM ALL_TAB_COLUMNS
--    WHERE TABLE_NAME = 'PC_ASM_FW_GCNMFW_WORK'
--      AND (COLUMN_NAME LIKE '%COMPLIANCE%' OR COLUMN_NAME LIKE '%ANALYST%'
--           OR COLUMN_NAME LIKE '%REMAK%');
--
-- Hasilnya SATU baris: `ANALYSTTRANSFERDATE_1`. Tidak ada `ISCOMPLIANCETRANSFER_1`, dan
-- tidak ada `ANALYSTDOCTORREMAKS_1`. Itulah sebab layar ini menjawab "Antrean tidak dapat
-- dimuat — Terjadi kesalahan pada sistem": kueri daftarnya gagal ORA-00904 pada SETIAP
-- permintaan, bukan sesekali.
--
-- `docs/keputusan-implementasi.md` §147.3 sudah mencatat temuan yang sama dari arah lain,
-- sewaktu membaca `PUCLPost`: "Kolom ISCOMPLIANCETRANSFER_1 yang dipakai kueri
-- inboxanalystdoctor tidak ada di basis data ini." Catatan itu benar; yang kurang hanyalah
-- tindakannya.
--
-- ============================================================================
-- PENGGANTI PENYARING UTAMA — PENUGASANNYA SENDIRI, BUKAN PENANDA DI DALAM BLOB
-- ============================================================================
--
-- Yang menempatkan sebuah klaim di antrean ini BUKAN penanda di dalam blob, melainkan
-- penugasannya. `Flow/Register_Flow.xml` `Assignment13` berbunyi:
--
--   <pyMOName>Analyst Doctor</pyMOName>
--   <pyImplementation>WorkList</pyImplementation>
--   <pyRouteTo>Operator</pyRouteTo>
--
-- dan Pega menyimpan `pyMOName` sebuah assignment sebagai `PXTASKLABEL` pada baris
-- worklist-nya. Kesepadanan itu bukan dugaan — ia terbaca langsung dari data, dengan SETIAP
-- label yang muncul untuk `Register_Flow` sama persis dengan `pyMOName` salah satu
-- assignment di flow itu:
--
--   Input Register · Input Estimasi · Estimation · Choose Surveyor · View Polis ·
--   Send To Analis · RCLDokter
--
-- Karena itu penyaringnya menjadi `a.PXTASKLABEL = :1` dengan nilai "Analyst Doctor".
--
-- Label itu AMAN dipakai sebagai penanda antrean: dari keenam flow di export, hanya
-- `Register_Flow` yang memuat assignment bernama "Analyst Doctor" — kelima flow lain nol
-- kemunculan — sehingga tidak ada flow lain yang dapat menghasilkan label yang sama.
--
-- Pola yang sama sudah dipakai modul lain dan bukan hal baru di aplikasi ini:
-- `inboxadmin/repo/sqlstore/inboxadmin.sql:191` menyaring
-- `B.PXTASKLABEL IN (Input Register, Input Estimasi, Estimation)`.
--
-- SELISIH YANG DITIMBULKANNYA, dan ia dinyatakan di layar sebagai selisih terencana
-- (`D-54`), bukan disamarkan:
--
--   Pega menyaring  "klaim yang PERNAH ditandai transfer ke Analyst Doctor".
--   Kueri ini       "klaim yang SEKARANG berada di tahap Analyst Doctor".
--
-- Keduanya berimpit selama klaimnya memang masih menunggu penilaian medis. Keduanya BERBEDA
-- untuk klaim yang penandanya masih bernilai 2 tetapi penugasannya sudah berpindah — klaim
-- itu muncul di Pega dan tidak muncul di sini. Untuk sebuah Inbox, yang kedua justru bacaan
-- yang benar menurut `D-79`: barisnya adalah pekerjaan yang MENUNGGU dikerjakan, dan
-- barisnya hilang begitu tugasnya berpindah.
--
-- ============================================================================
-- KOLOM "Komentar dari PIC Teknis" TIDAK LAGI DIAMBIL DARI SQL
-- ============================================================================
--
-- `.ClaimData.AnalystDoctorRemaks` tidak punya kolom. Dua calon penggantinya sudah diperiksa
-- dan KEDUANYA DITOLAK:
--
--   KOMENTARANALISATOR_1   properti `.ClaimData.KomentarAnalisator`, dipakai jalur PUCL
--                          (`Activity/PUCLPost-Act.xml`), bukan penilaian medis.
--
--   alias di rule SQL      `a.QQNAME AS "AnalystDoctorRemaks"` dan
--                          `a.clientname AS "AnalystDoctorRemaks"` muncul di belasan rule,
--                          menunjuk kolom yang berbeda-beda. Itu alias yang MENYESATKAN —
--                          utang teknis §4.2 — bukan bukti tempat penyimpanan.
--
-- Kolomnya karena itu TETAP DIGAMBAR di layar tetapi isinya kosong, dan layar menyatakan
-- alasannya. Itu pilihan yang sama dengan sebelumnya; yang berubah hanyalah ia tidak lagi
-- menjatuhkan SELURUH halaman hanya untuk mendapatkannya.
--
-- ============================================================================
-- PEMETAAN KOLOM — properti Pega -> kolom sebenarnya -> alias di sini
-- ============================================================================
--
-- Dari `Report Definition/InboxAnalystDoctor_RD-RD.xml` (isian) dan
-- `Harness/inboxAnalystDoctor_Harness-Harness.xml` (judul yang dilihat pengguna).
--
--   judul di layar             properti Pega                     kolom          alias
--   -------------------------- --------------------------------- -------------- ------------------
--   Nomor Case                 .pyID                             w.PYID         CASE_ID
--   No Polis                   .Policy.PolicyNo                  w.POLICYNO     POLICY_NUMBER
--   Nama Tertanggung           .Policy.QQName                    w.QQNAME       INSURED_NAME
--   Nama Cabang                .Policy.Quotation.BranchName      w.BRANCHNAME   BRANCH_NAME
--   Tanggal Pendaftaran        .pxCreateDateTime                 w.PXCREATE..   REGISTERED_AT
--   Nama Admin                 .pyOrigUserID                     w.PYORIGUSERID ADMIN_NAME
--   Komentar dari PIC Teknis   .ClaimData.AnalystDoctorRemaks    TIDAK ADA      (kosong)
--   Lama Waktu Klaim           dihitung, lihat catatan 2         --             --
--
--   tidak digambar             .ClaimData.UserTeknis             w.USERTEKNIS_1 TECHNICAL_PIC
--   tidak digambar             .pyStatusWork                     w.PYSTATUSWORK PROCESS_STATUS
--   tidak digambar             .pzInsKey                         w.PZINSKEY     REFERENCE
--
-- `.ClaimData.UserTeknis` DIAMBIL Report Definition tetapi TIDAK punya judul kolom di
-- harness. Ia dibaca dan dikirim ke layar, tidak digambar sebagai kolom (`D-13`).
--
-- ============================================================================
-- CATATAN 1 — PXOBJCLASS WAJIB DISARING, DAN ITU TIDAK ADA DI REPORT DEFINITION
-- ============================================================================
--
-- DATAPEGA.PC_ASM_FW_GCNMFW_WORK menampung DUA jenis objek kerja sekaligus:
--
--   ASM-FW-GCNMFW-Work-PNC               klaim PNC             <- yang ini
--   ASM-FW-GCNMFW-Work-ReceiveDocument   berkas penerimaan dokumen
--
-- Report Definition tidak menyaringnya karena ia TIDAK PERLU: di Pega, sebuah Report
-- Definition terikat kelasnya sendiri dan engine yang menambahkan penyaring kelasnya.
-- Menulis SQL langsung berarti penyaring itu harus ditulis tangan.
--
-- Melupakannya tidak menghasilkan galat apa pun — ia hanya mencampur berkas penerimaan
-- dokumen ke dalam antrean medis, dengan kolom yang kebetulan terisi karena keduanya
-- sama-sama punya PYID, POLICYNO, dan QQNAME. Pelajaran ini sudah dibayar sekali di
-- `inboxmanagerreceivepucl`.
--
-- ============================================================================
-- CATATAN 2 — "Lama Waktu Klaim" DIHITUNG, BUKAN DIBACA
-- ============================================================================
--
-- Report Definition layar ini tidak mengambil satu pun properti durasi. Kolom `LAMAKLAIM_1`
-- memang ada di tabel dengan 86 nilai berbeda, tetapi TIDAK dibaca layar ini, dan artinya —
-- dihitung sampai kapan, diperbarui kapan — tidak terbaca dari export mana pun.
--
-- Ia karena itu dihitung di Go dari REGISTERED_AT sampai hari ini, mengikuti preseden
-- `inboxcloseclaim.DurationDays` yang sudah disetujui Work Owner 2026-09-23. Perhitungannya
-- TIDAK dilakukan di SQL: "hari" yang dimaksud pengguna adalah hari WIB sementara kolomnya
-- UTC, dan menaruh konversi zona waktu di dalam SQL adalah cara paling cepat menyebarkannya
-- ke tempat-tempat yang lupa melakukannya — persis cacat `Set7Hours` sistem lama.
--
-- ============================================================================
-- CATATAN 3 — PERBANDINGAN OPERATOR MEMAKAI UPPER, DAN ITU PUNYA HARGA
-- ============================================================================
--
-- `UPPER(a.PXASSIGNEDOPERATORID) = UPPER(:2)` tidak dapat memakai indeks biasa pada kolom
-- itu; Oracle membutuhkan function-based index untuk itu.
--
-- Ia tetap dipilih, dan alasannya bukan kenyamanan. `11-SECURITY.md` §3.1 mencatat bahwa
-- nama access group muncul dalam DUA kapitalisasi di export — `ViewClaimPNC`/`VIEWCLAIMPNC`
-- dan `PncReceive`/`PNCRECEIVE` — sehingga keseragaman huruf pada identitas di sistem lama
-- memang tidak terjaga. Perbandingan persis akan membuat antrean tampak KOSONG bagi pengguna
-- yang login-nya tersimpan berbeda huruf, dan antrean kosong tidak pernah dilaporkan
-- siapa pun sebagai kerusakan.
--
-- Bila pengukuran nyata menunjukkan ia menjadi hambatan, penyelesaiannya adalah
-- function-based index dari DBA — bukan melonggarkan perbandingannya.
--
-- ============================================================================
-- YANG BERUBAH DARI SISTEM LAMA, DAN KENAPA
-- ============================================================================
--
-- 1. ANTREAN DIKENALI DARI PENUGASANNYA, bukan dari penanda di dalam blob. Lihat bagian
--    PENGGANTI PENYARING UTAMA di atas, termasuk selisih yang ditimbulkannya.
--
-- 2. PAGINASI DIKERJAKAN BASIS DATA.
--    Report Definition memakai `pyPageSize = 50` dan `pyMaxRecords = 500`, yang berarti
--    seluruh baris ditarik, dipotong di 500, lalu dinomori di klipboard. Di sini halamannya
--    dipotong OFFSET .. FETCH NEXT .. ROWS ONLY sebelum baris meninggalkan basis data —
--    didukung Oracle 12c+ dan PostgreSQL (`09-DATABASE-STRATEGY.md` §3.3). Ini PERUBAHAN
--    PERILAKU yang disadari: antrean di atas 500 baris kini terlihat utuh.
--
-- 3. JUMLAH SELURUH BARIS DIHITUNG `COUNT(*) OVER ()`.
--    Satu perjalanan, bukan dua: kueri kedua yang hanya menghitung akan membaca ulang
--    gabungan yang sama, dan gabungan itulah bagian yang mahal. Fungsi jendela dihitung
--    SEBELUM pemotongan halaman dipakai, sehingga angkanya jumlah seluruhnya — bukan jumlah
--    baris di halaman ini.
--
-- 4. KOTAK CARI DITAMBAHKAN.
--    Layar lama tidak punya penyaring apa pun. Begitu antreannya dipaginasi, satu klaim
--    menjadi sulit ditemukan — dan pencarian di peramban hanya menyentuh halaman yang
--    sedang terbuka, sehingga hasilnya bohong. Ia dinyatakan ke pengguna sebagai selisih
--    terencana (`D-54`).
--
-- 5. NILAI SELALU LEWAT PARAMETER BINDING.
--    Tidak ada satu pun nilai yang dirangkai ke teks SQL. Larangan perangkaian
--    (`08-TECHNICAL-STRATEGY.md` §4.3) tidak dikecualikan oleh keputusan mana pun: yang
--    direplikasi adalah perilaku bisnis, bukan celah injeksi.
--
-- ============================================================================
-- YANG SENGAJA TIDAK BERUBAH
-- ============================================================================
--
-- * `INNER JOIN` ke tabel penugasan, bukan `EXISTS`. Report Definition memakai
--   `pyJoinType = INNER`, sehingga klaim yang punya DUA penugasan terbuka pada operator yang
--   sama muncul DUA KALI. Itu perilaku sistem lama apa adanya (`P-5`), dan menggantinya
--   dengan `EXISTS` akan mengubah jumlah baris yang terlihat pengguna tanpa satu pun
--   keputusan yang mendasarinya.
--
-- * HANYA `Resolved-Completed` yang dikecualikan. `Resolved-Rejected` TIDAK — klaim yang
--   ditolak tetap muncul di antrean ini. Itu isi filter C pada Report Definition apa adanya,
--   dan ia BERBEDA dari `inboxoutstanding` yang mengecualikan keduanya. Perbedaannya dibawa,
--   bukan diseragamkan.
--
-- * Urutan `PXCREATEDATETIME DESC, PZINSKEY DESC` mengikuti kedua `pySortType = DESC` pada
--   Report Definition, termasuk pemutus serinya.
--
-- CATATAN PENANDA BIND. Berkas ini memakai gaya Oracle `:n`, sama seperti seluruh modul lain
-- di aplikasi ini. Ia BELUM portabel ke PostgreSQL yang memakai `$n`; itu utang yang sudah
-- ada sebelum modul ini dan berlaku untuk seluruh berkas .sql di sini.

-- name: list_tasks
-- Satu halaman antrean Analyst Doctor milik seorang operator.
--
-- Bind:
--   :1  label tahap penugasan — "Analyst Doctor", dari Register_Flow Assignment13
--   :2  Operator ID pemanggil
--   :3  status kerja yang DIKECUALIKAN — "Resolved-Completed"
--   :4  kata kunci pencarian, atau NULL bila kotak carinya kosong
--   :5  pola LIKE untuk Nomor Case — sudah ber-wildcard dan ber-escape, dibentuk di Go
--   :6  pola LIKE untuk No Polis — nilainya sama dengan :5
--   :7  offset
--   :8  jumlah baris
--
-- KENAPA :4, :5, DAN :6 MEMBAWA NILAI YANG SAMA DI BAWAH TIGA PENANDA BERBEDA
--
-- Karena satu penanda TIDAK BOLEH muncul dua kali. `database/sql` mengirim argumen menurut
-- posisi, dan driver go-ora menghitung SETIAP kemunculan `:n` sebagai satu variabel yang
-- harus diikat. Menulis `:4` tiga kali berarti kueri menuntut delapan ikatan sementara
-- pemanggil hanya mengirim enam, dan Oracle menjawab:
--
--   ORA-01008: not all variables bound
--
-- Itu BUKAN dugaan — ia terjadi pada berkas ini, dan tersembunyi di belakang ORA-00904
-- sampai penyebab yang pertama diperbaiki. Pola satu-penanda-satu-kemunculan adalah pola
-- yang sudah berlaku di modul lain; lihat `inboxcloseclaim.sql:171-176`.
SELECT w.PZINSKEY                    AS REFERENCE,
       w.PYID                        AS CASE_ID,
       w.POLICYNO                    AS POLICY_NUMBER,
       w.QQNAME                      AS INSURED_NAME,
       w.BRANCHNAME                  AS BRANCH_NAME,
       w.PYORIGUSERID                AS ADMIN_NAME,
       w.USERTEKNIS_1                AS TECHNICAL_PIC,
       w.PXCREATEDATETIME            AS REGISTERED_AT,
       w.PYSTATUSWORK                AS PROCESS_STATUS,
       a.PXASSIGNEDOPERATORID        AS ASSIGNED_OPERATOR,
       COUNT(*) OVER ()              AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST a
               ON a.PXREFOBJECTKEY = w.PZINSKEY
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND a.PXTASKLABEL = :1
   AND UPPER(a.PXASSIGNEDOPERATORID) = UPPER(:2)
   AND w.PYSTATUSWORK <> :3
   AND (:4 IS NULL
        OR UPPER(w.PYID) LIKE :5 ESCAPE '\'
        OR UPPER(w.POLICYNO) LIKE :6 ESCAPE '\')
 ORDER BY w.PXCREATEDATETIME DESC, w.PZINSKEY DESC
OFFSET :7 ROWS FETCH NEXT :8 ROWS ONLY

-- name: check_tables
-- Dipakai perintah `-periksa`: memastikan KEDUA tabel terbaca dari koneksi yang dipakai.
--
-- Ia tidak menyentuh satu baris pun — yang diperiksa adalah hak baca dan keberadaan
-- tabelnya, bukan isinya. Keduanya diperiksa sekaligus karena kegagalan yang paling mungkin
-- terjadi bukan "tabel tidak ada" melainkan "hak baca hanya diberikan pada salah satunya".
--
-- `COUNT(*)` dipakai, bukan sebuah kolom, supaya hasilnya SELALU tepat satu baris meski
-- penyaringnya tidak meloloskan apa pun — pemanggil karena itu tidak perlu membedakan "tidak
-- ada baris" dari "gagal dibaca".
SELECT COUNT(*) AS PROBE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST a
               ON a.PXREFOBJECTKEY = w.PZINSKEY
 WHERE 1 = 0

-- name: check_columns
-- Dipakai perintah `-periksa`: memastikan SETIAP kolom yang dibaca kueri daftar memang ada.
--
-- Ia terpisah dari check_tables dengan sengaja. Keduanya gagal karena sebab yang sangat
-- berbeda — yang satu hak baca, yang satu nama kolom — dan galat yang menyebut sebab yang
-- salah akan mengirim orang yang memperbaikinya ke arah yang keliru.
--
-- `WHERE 1 = 0` membuat Oracle tetap MEM-PARSE seluruh kolomnya tanpa membaca satu baris
-- pun. Parsing itulah yang menghasilkan ORA-00904 bila ada nama yang salah, dan itu yang
-- dicari di sini.
--
-- Yang diperiksa kini SELURUH kolom yang benar-benar dipakai, bukan hanya dua kolom tebakan
-- seperti versi sebelumnya. Alasannya langsung: versi sebelumnya memeriksa dua kolom yang
-- ternyata tidak ada lalu BERHENTI di situ, sehingga kolom lain tidak pernah sempat
-- terperiksa. Pemeriksaan yang menyerah pada temuan pertama menyembunyikan temuan kedua.
--
-- `PXTASKLABEL` ikut diperiksa karena sejak 2026-10-09 ia penyaring utama layar ini.
SELECT COUNT(w.PYID)                 AS PROBE_CASE_ID,
       COUNT(w.POLICYNO)             AS PROBE_POLICY,
       COUNT(w.QQNAME)               AS PROBE_INSURED,
       COUNT(w.BRANCHNAME)           AS PROBE_BRANCH,
       COUNT(w.PYORIGUSERID)         AS PROBE_ADMIN,
       COUNT(w.USERTEKNIS_1)         AS PROBE_TECHNICAL_PIC,
       COUNT(w.PXCREATEDATETIME)     AS PROBE_REGISTERED_AT,
       COUNT(w.PYSTATUSWORK)         AS PROBE_STATUS,
       COUNT(a.PXTASKLABEL)          AS PROBE_TASK_LABEL,
       COUNT(a.PXASSIGNEDOPERATORID) AS PROBE_OPERATOR
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST a
               ON a.PXREFOBJECTKEY = w.PZINSKEY
 WHERE 1 = 0
