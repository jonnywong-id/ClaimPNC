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
-- YANG HARUS DIBACA DBA LEBIH DULU — DUA KOLOM BELUM TERKONFIRMASI
-- ============================================================================
--
-- `Report Definition/InboxAnalystDoctor_RD-RD.xml` menandai DUA propertinya sendiri:
--
--   <pzPropertyType>unexposed</pzPropertyType>
--
-- yaitu `.ClaimData.isComplianceTransfer` dan `.ClaimData.AnalystDoctorRemaks`. Properti tak
-- terekspos hidup di dalam blob Pega, BUKAN sebagai kolom SQL. Pega tetap dapat
-- menyaringnya karena ia memuat blob lalu menyaring di memori; Go tidak dapat.
--
-- Work Owner menjawab 2026-09-23: **nilainya langsung di-set 2**. Jadi ia memang nilai
-- tersimpan, dan yang dibutuhkan hanyalah nama kolomnya.
--
-- Nama yang dipakai di bawah mengikuti konvensi `_1` yang berlaku pada SELURUH properti
-- `ClaimData` lain di tabel yang sama, dan konvensi itu bukan tebakan — ia terbaca dari
-- kolom yang sudah terbukti ada:
--
--   .ClaimData.PUCLStatus.StatusKlaim   -> STATUSKLAIM_1
--   .ClaimData.PUCLStatus.RCL_PUCL      -> RCL_PUCL_1
--   .ClaimData.PUCLStatus.LamaKlaim     -> LAMAKLAIM_1
--   .ClaimData.DateOfLoss               -> DATEOFLOSS_1
--   .ClaimData.UserTeknis               -> USERTEKNIS_1
--
-- sehingga:
--
--   .ClaimData.isComplianceTransfer     -> ISCOMPLIANCETRANSFER_1   ** PERLU KONFIRMASI **
--   .ClaimData.AnalystDoctorRemaks      -> ANALYSTDOCTORREMAKS_1    ** PERLU KONFIRMASI **
--
-- KEDUANYA TIDAK ADA di `docs/kolom-t-claimlist-admin.md`, yang disusun dari katalog Oracle
-- langsung (`ALL_TAB_COLUMNS`) dan mencatat 186 kolom dengan 123 di antaranya terisi.
-- Dokumen itu hanya memuat kolom YANG DIBUTUHKAN, bukan seluruhnya, sehingga ketiadaannya di
-- sana belum membuktikan ketiadaannya di tabel.
--
-- Satu kueri katalog menutup pertanyaan ini:
--
--   SELECT COLUMN_NAME, DATA_TYPE, NUM_DISTINCT
--     FROM ALL_TAB_COLUMNS
--    WHERE OWNER = 'DATAPEGA'
--      AND TABLE_NAME = 'PC_ASM_FW_GCNMFW_WORK'
--      AND (COLUMN_NAME LIKE '%COMPLIANCE%' OR COLUMN_NAME LIKE '%ANALYSTDOCTOR%');
--
-- BILA KOLOMNYA TIDAK ADA, kueri di bawah gagal dengan ORA-00904 yang MENYEBUT NAMA
-- KOLOMNYA. Itu disengaja. Alternatifnya — menghilangkan penyaringnya supaya kuerinya jalan —
-- akan menampilkan SELURUH tugas worklist pemanggil sebagai tugas medis, tanpa satu pun
-- pesan galat. Kegagalan yang menyebut sebabnya jauh lebih murah daripada layar yang
-- terlihat benar.
--
-- `-periksa` menembak kueri `check_columns` di bawah supaya keadaan ini diketahui SEBELUM
-- ada pengguna yang membukanya.
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
--   Tanggal Pendaftaran        .pxCreateDateTime                 w.PXCREATE…    REGISTERED_AT
--   Nama Admin                 .pyOrigUserID                     w.PYORIGUSERID ADMIN_NAME
--   Komentar dari PIC Teknis   .ClaimData.AnalystDoctorRemaks    (unexposed)    TECHNICAL_PIC_NOTE
--   Lama Waktu Klaim           — dihitung, lihat catatan 2       —              —
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
--   'ASM-FW-GCNMFW-Work-PNC'              klaim PNC             <- yang ini
--   'ASM-FW-GCNMFW-Work-ReceiveDocument'  berkas penerimaan dokumen
--
-- Report Definition tidak menyaringnya karena ia TIDAK PERLU: di Pega, sebuah Report
-- Definition terikat kelasnya sendiri (`pyClassName = ASM-FW-GCNMFW-Work-PNC`) dan engine
-- yang menambahkan penyaring kelasnya. Menulis SQL langsung berarti penyaring itu harus
-- ditulis tangan.
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
-- 1. PAGINASI DIKERJAKAN BASIS DATA.
--    Report Definition memakai `pyPageSize = 50` dan `pyMaxRecords = 500`, yang berarti
--    seluruh baris ditarik, dipotong di 500, lalu dinomori di klipboard. Di sini halamannya
--    dipotong `OFFSET … FETCH NEXT … ROWS ONLY` sebelum baris meninggalkan basis data —
--    didukung Oracle 12c+ dan PostgreSQL (`09-DATABASE-STRATEGY.md` §3.3). Ini PERUBAHAN
--    PERILAKU yang disadari: antrean di atas 500 baris kini terlihat utuh.
--
-- 2. JUMLAH SELURUH BARIS DIHITUNG `COUNT(*) OVER ()`.
--    Satu perjalanan, bukan dua: kueri kedua yang hanya menghitung akan membaca ulang
--    gabungan yang sama, dan gabungan itulah bagian yang mahal. Fungsi jendela dihitung
--    SEBELUM `OFFSET … FETCH` dipakai, sehingga angkanya jumlah seluruhnya — bukan jumlah
--    baris di halaman ini.
--
-- 3. KOTAK CARI DITAMBAHKAN.
--    Layar lama tidak punya penyaring apa pun. Begitu antreannya dipaginasi, satu klaim
--    menjadi sulit ditemukan — dan pencarian di peramban hanya menyentuh halaman yang
--    sedang terbuka, sehingga hasilnya bohong. Ia dinyatakan ke pengguna sebagai selisih
--    terencana (`D-54`).
--
-- 4. NILAI SELALU LEWAT PARAMETER BINDING.
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
--   ditolak tetap muncul di antrean ini. Itu isi `pyFilterOperation != ` pada filter C apa
--   adanya, dan ia BERBEDA dari `inboxoutstanding` yang mengecualikan keduanya. Perbedaannya
--   dibawa, bukan diseragamkan.
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
--   :1  penanda antrean — `ClaimData.isComplianceTransfer`, bernilai "2"
--   :2  Operator ID pemanggil
--   :3  status kerja yang DIKECUALIKAN — "Resolved-Completed"
--   :4  kata kunci pencarian, atau NULL bila kotak carinya kosong
--   :5  offset
--   :6  jumlah baris
SELECT w.PZINSKEY                    AS REFERENCE,
       w.PYID                        AS CASE_ID,
       w.POLICYNO                    AS POLICY_NUMBER,
       w.QQNAME                      AS INSURED_NAME,
       w.BRANCHNAME                  AS BRANCH_NAME,
       w.PYORIGUSERID                AS ADMIN_NAME,
       w.USERTEKNIS_1                AS TECHNICAL_PIC,
       w.ANALYSTDOCTORREMAKS_1       AS TECHNICAL_PIC_NOTE,
       w.PXCREATEDATETIME            AS REGISTERED_AT,
       w.PYSTATUSWORK                AS PROCESS_STATUS,
       a.PXASSIGNEDOPERATORID        AS ASSIGNED_OPERATOR,
       COUNT(*) OVER ()              AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST a
               ON a.PXREFOBJECTKEY = w.PZINSKEY
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND w.ISCOMPLIANCETRANSFER_1 = :1
   AND UPPER(a.PXASSIGNEDOPERATORID) = UPPER(:2)
   AND w.PYSTATUSWORK <> :3
   AND (:4 IS NULL
        OR UPPER(w.PYID) LIKE '%' || UPPER(:4) || '%'
        OR UPPER(w.POLICYNO) LIKE '%' || UPPER(:4) || '%')
 ORDER BY w.PXCREATEDATETIME DESC, w.PZINSKEY DESC
OFFSET :5 ROWS FETCH NEXT :6 ROWS ONLY

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
-- Dipakai perintah `-periksa`: memastikan KEDUA kolom yang belum terkonfirmasi memang ada.
--
-- Ia terpisah dari check_tables dengan sengaja. Keduanya gagal karena sebab yang sangat
-- berbeda — yang satu hak baca, yang satu nama kolom yang belum dipastikan DBA — dan galat
-- yang menyebut sebab yang salah akan mengirim orang yang memperbaikinya ke arah yang keliru.
--
-- `WHERE 1 = 0` membuat Oracle tetap MEM-PARSE kedua kolom tanpa membaca satu baris pun.
-- Parsing itulah yang menghasilkan ORA-00904 bila namanya salah, dan itu yang dicari di sini.
SELECT COUNT(w.ISCOMPLIANCETRANSFER_1) AS PROBE_TRANSFER,
       COUNT(w.ANALYSTDOCTORREMAKS_1)  AS PROBE_NOTE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
 WHERE 1 = 0
