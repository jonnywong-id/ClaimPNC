-- Daftar PIC Teknik yang dapat menerima pemindahan klaim.
--
-- Menggantikan `Report Definition/BrowseVMstUserTeknis_RD-RD.xml`.
--
-- TERVERIFIKASI: `Section/PNCTransferManagement_sec-Section.xml:5855` memang memakai RD ini
-- sebagai sumber grid-nya. Section itu sempat tidak ada di export dan kaitannya hanya
-- diduga; Work Owner mengambilkannya pada 2026-10-06 dan dugaannya terbukti.
--
-- Asal `param.type_business` SUDAH diketahui: `PNCTransferManagement_sec:3494` mengisinya
-- dari `TempIns.ClaimData.CityID`, dan `GCNMTransferAssignmentManager_act:967` mengisi
-- properti itu dari `OperatorID.pyPosition`.
--
-- Tetapi nilai itu TIDAK dapat dipakai di sistem baru. Pada Pega, `pyPosition` rupanya diisi
-- "NONMBU", "TRAVEL", "BONDING", atau "PA"; HCC/HCQ mengembalikan jabatan sebenarnya
-- (`Placement.PositionName`), dan keduanya bukan hal yang sama. Mencocokkan "Staf" dengan
-- `TYPE_BUSINESS` mengembalikan nol baris — daftar kosong yang tidak menjelaskan dirinya.
--
-- Karena itu lini bisnis DIPILIH PENGGUNA, dan keterbatasannya dinyatakan di layar — cara
-- yang sama yang sudah ditempuh `inboxprogressclaim` untuk persoalan yang persis sama.
-- Sampai pemetaan pengguna ke lini bisnis menjadi master data (`F-4`), itu jawaban yang
-- paling jujur.
--
-- ============================================================================
-- DUA KOREKSI SETELAH DICOBA TERHADAP ORACLE (2026-10-06)
-- ============================================================================
--
-- 1. TOTAL_JOB DIBUANG — kolomnya tidak ada di sini
--
--    Kueri ini sempat membaca `A.TOTAL_JOB` dan GAGAL di Oracle. Kolom itu memang disebut
--    `AddTJobCQuota_SQL` dan `UpdateTotalJobChild_sql`, tetapi keduanya MENULIS; satu-satunya
--    kueri yang MEMBACANYA — `SetTotalJobMstUserTeknis-SQL.xml` — membacanya dari tabel
--    REMOTE `new_general.m_user_job@opjava`, bukan dari master ini.
--
--    Modul `internal/masterpicteknik` yang sudah berjalan atas tabel yang sama juga tidak
--    membacanya. Daftar kolom yang terbukti dapat dibaca adalah daftar miliknya.
--
-- 2. PENYARING TYPE_BUSINESS DIBUANG — nilainya tidak ada di sistem baru
--
--    RD aslinya menyaring `TYPE_BUSINESS = Param.type_business`, dan parameter itu diisi
--    `OperatorID.pyPosition` — properti Pega yang rupanya memuat "NONMBU", "TRAVEL",
--    "BONDING", atau "PA". HCC/HCQ mengembalikan jabatan sebenarnya
--    (`Placement.PositionName`), dan keduanya bukan hal yang sama.
--
--    Tiga bentuk sudah dicoba dan ketiganya keliru: memblokir daftar dengan peringatan,
--    memakai jabatan sesi (daftar kosong tanpa penjelasan), lalu meminta pengguna memilih
--    lini bisnis — langkah yang di layar lama TIDAK ADA.
--
--    Work Owner memutuskan (2026-10-06) layar ini mengikuti Pega apa adanya. Yang tersisa
--    karena itu hanya `STS_AKTIF = 1`.
--
-- # SELISIH YANG DITERIMA, DAN HARUS DIKETAHUI
--
-- Daftar ini LEBIH LUAS daripada daftar Pega: di sana petugas lini lain tersaring keluar,
-- di sini tidak. Memindahkan klaim ke petugas lini lain karena itu MUNGKIN, dan tidak ada
-- yang mencegahnya.
--
-- Penyempitannya menunggu pemetaan pengguna ke lini bisnis menjadi master data (`F-4`).
-- Sampai itu ada, menyaring dengan nilai yang salah menghasilkan daftar kosong — dan daftar
-- kosong lebih buruk daripada daftar yang terlalu luas, karena ia tidak menjelaskan dirinya.
--
-- Penyaring yang tersisa dari RD:
--
--	STS_AKTIF = 1
--
-- Kolomnya dari `RDB List/GetMasterPICTeknis-SQL.xml`, yang membaca
-- `POOLDATA.MST_USER_TEKNIK`.
--
-- COUNTER_QUOTA ikut dibaca meski tidak digambar RD-nya: ia pencacah beban yang dipakai
-- `BrowsePICRandomTeam-SQL` untuk memilih petugas paling senggang (`R-04`). Menampilkannya
-- membuat pemilihan manual mempertimbangkan hal yang sama dengan pemilihan otomatis.
--
-- Pola pencarian yang sama ditulis sebagai dua penanda terpisah (`:3` dan `:4`), bukan satu
-- penanda yang diulang. Itu aturan modul ini: satu penanda satu kemunculan, supaya urutan
-- argumen terbaca dari SQL-nya saja.
--
-- ============================================================================
-- DUA HAL DI SINI ADALAH TAMBAHAN, BUKAN REPLIKASI
-- ============================================================================
--
-- 1. KOTAK CARI (`:2`…`:4`) TIDAK ADA DI RD
--
--    `BrowseVMstUserTeknis_RD` hanya punya dua penyaring, dan keduanya sudah disalin di
--    atas. Pencarian nama/User ID ditambahkan karena daftar ini dipaginasi, sementara grid
--    lama memuat seluruh barisnya sekaligus — pada daftar berhalaman, "petugasnya tidak ada
--    di halaman ini" tidak dapat dibedakan dari "petugasnya tidak ada".
--
-- 2. URUTANNYA TIDAK ADA DI RD
--
--    RD itu tidak menetapkan urutan sama sekali, sehingga Pega mengembalikan baris dalam
--    urutan yang tidak dijamin. Itu AMAN selama seluruh baris tampil sekaligus, dan menjadi
--    cacat begitu dipaginasi: tanpa urutan yang tetap, satu baris dapat tampil di dua
--    halaman sementara baris lain tidak pernah tampil.
--
--    `COUNTER_QUOTA ASC` dipilih — bukan sekadar urutan apa pun — karena itulah urutan yang
--    dipakai `BrowsePICRandomTeam-SQL` saat sistem lama memilih petugas secara otomatis
--    (`R-04`). Dengan begitu pemilihan manual melihat petugas paling senggang lebih dulu,
--    sama seperti pemilihan otomatis. `MCL_NAME` hanya pemutus seri.
--
--    Keduanya menambah kemampuan, bukan mengubah hasil: himpunan baris yang memenuhi syarat
--    tetap sama persis dengan RD-nya.

-- name: pic_teknik_count
SELECT COUNT(*)
  FROM POOLDATA.MST_USER_TEKNIK A
 WHERE A.STS_AKTIF = '1'
   AND (:1 IS NULL OR UPPER(A.OPERATOR_ID) LIKE :2 ESCAPE '\'
        OR UPPER(A.MCL_NAME) LIKE :3 ESCAPE '\')

-- name: pic_teknik_list
SELECT A.OPERATOR_ID,
       A.MCL_NAME,
       A.EMAIL,
       A.TEAM_GROUP,
       A.COUNTER_QUOTA
  FROM POOLDATA.MST_USER_TEKNIK A
 WHERE A.STS_AKTIF = '1'
   AND (:1 IS NULL OR UPPER(A.OPERATOR_ID) LIKE :2 ESCAPE '\'
        OR UPPER(A.MCL_NAME) LIKE :3 ESCAPE '\')
 ORDER BY A.COUNTER_QUOTA ASC, A.MCL_NAME ASC
OFFSET :4 ROWS FETCH NEXT :5 ROWS ONLY

-- name: pic_teknik_check
-- Membuktikan master PIC Teknik ada DAN kolom yang dibaca `pic_teknik_list` benar-benar ada.
--
-- Kolomnya disebut satu per satu, bukan `SELECT 1` atau `COUNT(*)`. Alasannya konkret:
-- kueri ini sempat membaca `A.TOTAL_JOB`, kolom yang tidak ada pada tabel ini, dan probe
-- `SELECT 1` akan **lulus** sementara daftar PIC gagal dimuat. Yang lulus harus yang sama
-- dengan yang dipakai.
--
-- `WHERE 1 = 0` membuatnya tidak membaca satu baris pun; yang diperiksa parse-nya.
SELECT A.OPERATOR_ID,
       A.MCL_NAME,
       A.EMAIL,
       A.TEAM_GROUP,
       A.COUNTER_QUOTA,
       A.STS_AKTIF
  FROM POOLDATA.MST_USER_TEKNIK A
 WHERE 1 = 0
