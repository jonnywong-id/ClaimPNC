-- Pemindahan PIC Teknik sebuah klaim — tombol **Assign** pada layar Transfer.
--
-- ============================================================================
-- KENAPA MENULIS LANGSUNG, BUKAN MENCATAT PERMINTAAN
-- ============================================================================
--
-- Bentuk pertama modul ini MENCATAT PERMINTAAN dan menyerahkan pelaksanaannya ke Pega,
-- karena `P-1` menetapkan satu tabel satu penulis selama masa paralel.
--
-- Work Owner memutuskan (2026-10-06) jalur ini mengikuti Pega apa adanya, dan keputusan itu
-- berdiri di atas satu kenyataan: **antrean permintaan tidak punya pelaksana.** Tidak ada
-- satu pun job Pega yang membacanya — kelima job terjadwal (`D-57`) seluruhnya lebih tua
-- daripada tabelnya. Antrean tanpa pelaksana menumpuk dalam keadaan `menunggu`, dan pengguna
-- membaca "permintaan tercatat" untuk sesuatu yang tidak akan pernah dijalankan. Itu lebih
-- buruk daripada menyatakan fiturnya belum siap.
--
-- ============================================================================
-- APA YANG DIUBAH PEGA, DAN APA YANG TIDAK
-- ============================================================================
--
-- `Activity/PNC_ReassignPNCTeknik-Act.xml:4526` menyalin `PNCWorkPage.ClaimData.UserTeknis`
-- ke `pyWorkPage.ClaimData.UserTeknis`, lalu `Obj-Save` + `Commit`. Properti itu memetakan ke
-- `USERTEKNIS_1` pada tabel kerja.
--
-- Yang ia TIDAK lakukan: memanggil `pxTransferAssignment`. Pemanggilan itu ada di
-- `GCNMTransferDataKlaim_act:11527` — jalur **Transfer All Case By UserID**, bukan jalur ini.
--
-- Artinya Assign per baris TIDAK memindahkan penugasan alur kerja. Yang berpindah adalah
-- **siapa PIC Teknik klaim itu**, bukan antrean tugasnya.
--
-- ============================================================================
-- SKEMA DATAPEGA TIDAK DIPAKAI LAGI (2026-10-08)
-- ============================================================================
--
-- Seluruh kueri modul ini — baca maupun tulis — berpindah dari
--
--     DATAPEGA.PC_ASM_FW_GCNMFW_WORK   tabel kerja Pega
--     DATAPEGA.PC_ASSIGN_WORKLIST      antrean tugas Pega
--
-- ke satu tabel datar `POOLDATA.T_CLAIMLIST_ADMIN`, atas keputusan Work Owner. Kedua tabel
-- itu menyatu di sana, sehingga **delapan INNER JOIN hilang** dari modul ini.
--
-- Ketiga tabel yang ditulis jalur ini seluruhnya milik skema `POOLDATA`:
--
--     POOLDATA.T_CLAIMLIST_ADMIN   kolom USERTEKNIS_1
--     POOLDATA.MST_USER_TEKNIK     kolom COUNTER_QUOTA
--     POOLDATA.PEGA_DASHBOARDPNC   kolom PIC
--
-- TIDAK ADA GRANT YANG DIBUTUHKAN pada entitas yang aplikasinya menyambung sebagai pemilik
-- skema — dan pada ASM memang demikian (`.env` -> `POOLDATA_ASM_PENGGUNA=POOLDATA`). Pemilik
-- skema sudah memegang seluruh hak atas objeknya sendiri; `GRANT … TO POOLDATA` justru ditolak
-- `ORA-01749`. Jadi bila Transfer gagal di sini, periksa nama kolom atau isi datanya — BUKAN
-- hak akses. Rinciannya, beserta kapan GRANT baru benar-benar perlu, di
-- `docs/grant-dashboard-claim.sql`.
--
-- ISI TABEL. Backfill dari kedua tabel Pega dijalankan 2026-10-08 (`docs/isi-t-claimlist-admin.sql`).
-- Di DEV_PEGA83 tabel ini kini memuat **2.648 klaim**, dengan `SURVEYORTYPE_1` terisi pada 674
-- baris dan `STATUSCLAIM_1` pada 1.998. Angka "baru terisi 13%" yang beredar di dokumen lain
-- berasal dari lingkungan berbeda dan tidak berlaku di sini.
--
-- Yang masih terbuka dan BUKAN soal kode: backfill itu mengisi SEKALI. Siapa yang menjaga tabel
-- ini tetap mutakhir untuk klaim baru belum ditetapkan (`P-1` — satu tabel satu penulis).
--
-- ============================================================================
-- YANG DIREPLIKASI APA ADANYA
-- ============================================================================
--
-- `counter_quota = counter_quota + 1` pada baris yang nilainya NULL menghasilkan NULL, bukan
-- 1. Itu perilaku `RDB List/AddTJobCQuota_SQL-SQL.xml` dan dibawa apa adanya (`P-5`).
-- Membungkusnya `COALESCE` akan membuat pencacah melompat dari kosong menjadi 1 pada
-- pemindahan pertama — perubahan angka yang tidak diminta siapa pun.
--
-- Nama tabelnya ditulis `POOLDATA.MST_USER_TEKNIK`, sedangkan kueri Pega menulis
-- `mst_user_teknis`. Work Owner menegaskan (2026-10-06) keduanya **sinonim**; yang dipakai di
-- sini nama yang sama dengan kueri pembacanya, supaya satu modul tidak memakai dua nama untuk
-- satu objek.

-- name: pic_sekarang
-- Membaca PIC yang berlaku SEBELUM dipindahkan.
--
-- Dibaca di dalam transaksi yang sama dengan pemindahannya, dan dengan `FOR UPDATE`: tanpa
-- kunci itu, dua pemindahan yang berjalan bersamaan sama-sama membaca PIC lama yang sama,
-- lalu keduanya menurunkan pencacah orang yang sama dua kali.
SELECT A.USERTEKNIS_1
  FROM POOLDATA.T_CLAIMLIST_ADMIN A
 WHERE A.PZINSKEY = :1
   FOR UPDATE

-- name: pindah_pic
-- Memindahkan PIC Teknik klaim. Satu kolom, satu baris.
UPDATE POOLDATA.T_CLAIMLIST_ADMIN
   SET USERTEKNIS_1 = :1
 WHERE PZINSKEY = :2

-- name: pencacah_naik
-- Menaikkan beban PIC baru — `RDB List/AddTJobCQuota_SQL-SQL.xml`.
--
-- ============================================================================
-- TOTAL_JOB DIBUANG — kolomnya tidak ada pada TABEL ini (2026-10-07)
-- ============================================================================
--
-- Kueri Pega-nya berbunyi:
--
--     update mst_user_teknis set total_job = total_job+1, counter_quota = counter_quota+1
--      where operator_id = {TempRepType.SearchName}
--
-- Dua kolom. Di sini hanya SATU yang dibawa, dan sebabnya bukan pilihan gaya:
--
--     POOLDATA.MST_USER_TEKNIK      TABEL. COUNTER_QUOTA ada; TOTAL_JOB TIDAK ADA.
--     POOLDATA.V_MST_USER_TEKNIS    VIEW.  TOTAL_JOB ada di sini, dan hanya DIBACA.
--
-- Ejaannya berbeda dan itu bukan salah ketik — view berakhiran **TEKNIS**, tabel berakhiran
-- **TEKNIK**. Pembagian ini sudah dipetakan modul `internal/masterpicteknik`, yang berjalan
-- di atas kedua objek yang sama: ia membaca daftar dari VIEW dan menulis hanya ke TABEL.
--
-- Menulis `TOTAL_JOB` ke tabel menjawab ORA-00904, dan galat itu BUKAN kekurangan hak akses
-- sehingga ia jatuh sebagai `500` — bukan `503` yang menyebut apa yang kurang. Itulah yang
-- terjadi pada penekanan Assign pertama di Oracle.
--
-- # YANG HILANG KARENANYA, DAN PERTANYAANNYA BELUM TERJAWAB
--
-- `TOTAL_JOB` tidak lagi bergerak. Ia **tidak dipakai** memilih petugas otomatis —
-- `BrowsePICRandomTeam-SQL` mengurutkan `COUNTER_QUOTA`, bukan `TOTAL_JOB` (`R-04`) —
-- sehingga pemilihan otomatis tetap benar. Yang terpengaruh hanya angka yang ditampilkan
-- daftar PIC, dan itu dibaca dari view.
--
-- BELUM DIPUTUSKAN — pertanyaan terbuka (pemilik: DBA, lalu Work Owner):
-- dari mana `V_MST_USER_TEKNIS.TOTAL_JOB` berasal. Dua kemungkinan berakibat berbeda:
--
--   1. ia DIHITUNG di dalam view  → tidak ada yang perlu dipelihara, dan kedua kueri Pega
--                                   di atas memang sudah tidak berfungsi hari ini;
--   2. ia kolom pada objek lain   → ada pemeliharaan yang belum kita tiru.
--
-- Satu kueri menjawabnya, dan ia tidak menyentuh data:
--
--     SELECT text FROM all_views WHERE view_name = 'V_MST_USER_TEKNIS';
--
-- Sampai terjawab, pencacah yang DAPAT dibuktikan ada-lah yang dibawa.
UPDATE POOLDATA.MST_USER_TEKNIK
   SET COUNTER_QUOTA = COUNTER_QUOTA + 1
 WHERE OPERATOR_ID = :1

-- name: pencacah_check
-- Membuktikan pencacah beban dapat ditulis akun aplikasi.
--
-- Ia menyebut `COUNTER_QUOTA` — kolom yang sama dengan `pencacah_naik`. Probe yang hanya
-- membuktikan tabelnya ada akan LULUS terhadap kolom yang tidak ada, dan itu persis
-- kegagalan yang membuat probe ini dibuat.
--
-- `OPERATOR_ID` ikut disebut sejak 2026-10-08. Versi sebelumnya hanya menyebut kolom yang
-- di-SET, sehingga kolom KUNCI-nya tidak terbukti ada — probe lulus, kueri nyatanya gagal
-- `ORA-00904`. Oracle memvalidasi SELURUH rujukan kolom saat mem-parse, termasuk yang berada
-- di belakang `1 = 0`, jadi menyebutnya di sini tidak membuat probe berhenti inert.
UPDATE POOLDATA.MST_USER_TEKNIK
   SET COUNTER_QUOTA = COUNTER_QUOTA
 WHERE 1 = 0
   AND OPERATOR_ID IS NOT NULL


-- name: dashboard_pic
-- Memperbarui PIC pada tabel ringkasan dashboard — `RDB List/UpdateTotalJob_sql-SQL.xml`.
--
-- Namanya menyesatkan: kueri bernama "UpdateTotalJob" itu tidak menyentuh TOTAL_JOB sama
-- sekali. Yang diperbaruinya PIC pada `PEGA_DASHBOARDPNC`.
UPDATE POOLDATA.PEGA_DASHBOARDPNC
   SET PIC = :1
 WHERE NOKLAIM = :2

-- name: pindah_pic_massal
-- Memindahkan SELURUH klaim berjalan milik satu petugas — "Transfer All Case By UserID".
--
-- Syarat klaim berjalannya disalin dari `outstanding.sql`: tanpa itu, klaim yang sudah
-- selesai ikut berpindah, dan pemindahan klaim tutup tidak tampak sebagai galat — hanya
-- sebagai angka beban yang tidak masuk akal.
--
-- `pxCreateDateTime` TIDAK ikut disaring: yang dipindahkan adalah seluruh pekerjaan petugas
-- itu, bukan sebagian.
--
-- ============================================================================
-- SELISIH YANG DITERIMA — JALUR INI TIDAK SAMA DENGAN JALUR PER BARIS
-- ============================================================================
--
-- `pindah_pic` di atas aman terhadap `P-1` karena `PNC_ReassignPNCTeknik` pun tidak menyentuh
-- `PC_ASSIGN_WORKLIST`. **Jalur ini berbeda.**
--
-- `Activity/GCNMTransferDataKlaim_act-Act.xml:11527` MEMANGGIL `pxTransferAssignment`, yang
-- memindahkan penugasan Pega di `DATAPEGA.PC_ASSIGN_WORKLIST`. Kueri ini tidak.
--
--     Pega         USERTEKNIS_1 berpindah  +  PC_ASSIGN_WORKLIST berpindah
--     di sini      USERTEKNIS_1 berpindah  +  PC_ASSIGN_WORKLIST TIDAK disentuh
--
-- Akibatnya selama masa paralel: daftar di aplikasi ini menunjukkan petugas yang BARU,
-- sedangkan **inbox Pega masih menunjukkan yang LAMA**.
--
-- Selisih ini DITERIMA, bukan ditambal. Menulis `PC_ASSIGN_WORKLIST` melanggar `P-1` — tabel
-- itu ditulis Pega selama masa paralel — dan di jalur inilah keberatan itu benar-benar sah.
--
-- Bila inbox Pega harus ikut berpindah, itu **perubahan `P-1`** yang menempuh Work Owner,
-- bukan penambahan satu kueri. Dicatat juga di `keputusan-implementasi.md` §204.
UPDATE POOLDATA.T_CLAIMLIST_ADMIN A
   SET A.USERTEKNIS_1 = :1
 WHERE A.USERTEKNIS_1 = :2
   AND A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.PYSTATUSWORK <> 'Resolved-Completed'
   AND A.PYSTATUSWORK <> 'Resolved-Rejected'

-- name: pindah_pic_check
-- Membuktikan hak UPDATE pada kolom PIC Teknik sudah diberikan.
--
-- `WHERE 1 = 0` membuatnya TIDAK mengubah satu baris pun, sementara Oracle tetap memeriksa
-- hak akses saat mem-parse pernyataannya: tanpa GRANT, ia tetap menjawab ORA-01031.
--
-- Itu sebabnya probe ini berupa UPDATE, bukan SELECT. Hak SELECT pada tabel ini sudah
-- dimiliki — probe SELECT akan lulus sementara tombol Transfer tetap menjawab 503, dan
-- probe yang lulus saat fiturnya mati lebih buruk daripada tidak ada probe.
--
-- `PZINSKEY` ikut disebut dengan alasan yang sama seperti pada `pencacah_check`: kolom kunci
-- yang tidak pernah disebut probe adalah kolom yang tidak pernah dibuktikan ada.
UPDATE POOLDATA.T_CLAIMLIST_ADMIN A
   SET A.USERTEKNIS_1 = A.USERTEKNIS_1
 WHERE 1 = 0
   AND A.PZINSKEY IS NOT NULL

-- name: dashboard_pic_check
-- Probe hak tulis `POOLDATA.PEGA_DASHBOARDPNC(PIC)` — lihat CheckDashboardWritable.
--
-- Inert: `WHERE 1 = 0` dan kolomnya diisi dengan dirinya sendiri, sehingga tidak ada baris
-- yang berubah. Oracle tetap memeriksa hak akses saat mem-parse-nya.
--
-- `NOKLAIM` WAJIB ikut disebut, dan di sinilah ia paling penting dari ketiga probe.
-- `PEGA_DASHBOARDPNC` satu-satunya tabel yang modul ini TULIS TANPA PERNAH BACA — tidak ada
-- satu pun kueri lain di modul ini yang membuktikan kolom kuncinya ada. Tanpa baris ini,
-- `-periksa` menjawab hijau sementara `dashboard_pic` gagal `ORA-00904` saat tombol Transfer
-- ditekan, dan kegagalannya muncul pertama kali di tangan penguji.
UPDATE POOLDATA.PEGA_DASHBOARDPNC
   SET PIC = PIC
 WHERE 1 = 0
   AND NOKLAIM IS NOT NULL
