-- Kueri modul Master Login.
--
-- SATU tabel, dan aplikasi ini MENULISNYA: POOLDATA.MST_LOGIN_SURVEYOR. Tidak ada tabel
-- acuan, tidak ada sequence, dan tidak ada M_SITE_DATABASE — kuncinya diturunkan dari
-- NAMA, bukan diterbitkan.
--
-- Kewenangan menulis berpindah dari Pega ke Go saat modulnya lulus gerbang 2 (P-1). Selama
-- Pega masih penulisnya, layar ini harus dijalankan dalam modus baca saja di produksi.
--
-- Empat aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (D-20).
--   4. Tanpa pemanggilan stored procedure (D-02).
--
--
-- ============================================================================
-- TABELNYA PUNYA TUJUH KOLOM, DAN ITU SUDAH DIPASTIKAN
-- ============================================================================
--
-- Ketiga rule Pega yang menyentuh tabel ini menyebut kolom yang sama persis, dan tidak satu
-- pun menyebut kolom di luar ketujuh ini:
--
--   NAMA         GetLoginMemberSurveyor · GetLoginLeaderSurveyor · UpdateMasterLoginSurvey
--   LOGIN        ketiganya, dan ia satu-satunya penyaring WHERE pada keduanya yang menulis
--   EMAIL        ketiganya
--   TELP         ketiganya
--   ALAMAT       ketiganya
--   STSLOGIN     ketiganya
--   LOGINLEADER  ketiganya, ditambah GetLoginLeaderSurveyor yang MEMBACANYA sendirian
--
-- Tidak ada kolom APPROVAL, tidak ada pencatat pelaku, tidak ada stempel waktu, dan tidak
-- ada penanda aktif. Itu keterbatasan tabelnya, dan ia menentukan bentuk layarnya: tanpa
-- tab, tanpa persetujuan, dan tanpa cara menonaktifkan sebuah login.
--
--
-- ============================================================================
-- KENAPA ALIAS KOLOM PEGA TIDAK DIBAWA
-- ============================================================================
--
-- Rule browse-nya mengalias setiap kolom menjadi properti klipboard yang namanya tidak ada
-- hubungannya dengan isinya:
--
--   select nama as "SurveyName", login as "SurveyorID", email as "Email",
--          telp as "Ekst", alamat as "BodyLetterTo", stslogin as "ObjectName",
--          loginleader as "NamaPasien"
--
-- "NamaPasien" untuk login seorang leader, "Ekst" untuk nomor telepon, dan "BodyLetterTo"
-- untuk alamat. Itu persis bentuk utang yang 03-CURRENT-ARCHITECTURE.md §4.2 catat: nama
-- dipaksa cocok dengan property klipboard yang sudah ada. Alias itu TIDAK dibawa; kolomnya
-- disebut nama aslinya.
--
--
-- ============================================================================
-- PENYARING {ASIS:...} TIDAK DIBAWA, DAN ITU YANG PALING PENTING DI BERKAS INI
-- ============================================================================
--
-- Rule aslinya menempelkan penyaringnya sebagai TEKS:
--
--   select ... from pooldata.mst_login_surveyor {ASIS:InputLogin.IDIndex}
--
-- dan `Activity/SetLoginSurveyorValue_act` mengisinya dengan
--
--   InputLogin.IDIndex := "where login = '" + Param.operatorid + "'"
--
-- Tanpa satu pun pelolosan. Nilai yang memuat tanda kutip tunggal akan menutup literalnya
-- dan sisanya dieksekusi sebagai SQL — celah injeksi yang 03-CURRENT-ARCHITECTURE.md §4.5
-- catat, di sini pada tabel yang menentukan siapa boleh masuk sebagai surveyor.
--
-- Di berkas ini penyaringnya menjadi KUERI TERSENDIRI dengan parameter binding. Bentuk
-- tanpa penyaring dan bentuk berpenyaring ditulis terpisah, bukan satu kueri yang klausanya
-- ditempel saat penyaringnya ada.
--
--
-- ============================================================================
-- LOGIN DIBANDINGKAN DENGAN TRIM DAN UPPER
-- ============================================================================
--
-- TRIM: bila kolomnya CHAR dan bukan VARCHAR2, Oracle memadatkan pembandingnya dengan spasi
-- sehingga perbandingan langsung tetap benar — tetapi PostgreSQL tidak melakukannya, dan
-- baris yang sama akan hilang setelah pindah basis data (D-24). DDL-nya tidak ada (R-08),
-- jadi keduanya harus tetap benar.
--
-- UPPER: hanya pada pemeriksaan KEUNIKAN dan pencarian, TIDAK pada pengambilan satu baris.
-- Pembedaan itu disengaja:
--
--   * Keunikan memakai UPPER supaya "BudiSantoso" dan "budisantoso" tidak menjadi dua baris
--     yang bertabrakan pada basis data yang membandingkan tanpa membedakan huruf.
--   * Pengambilan TIDAK memakai UPPER supaya kunci yang dikirim layar menunjuk baris yang
--     persis itu — login lama yang sudah ada dibaca apa adanya, dan Pega pun tidak pernah
--     meng-uppercase-nya (lihat masterlogin.DeriveLogin).
--
-- Harganya index pada LOGIN tidak terpakai pada kueri yang memakai UPPER. Tabel ini master
-- berisi orde ratusan baris, sehingga pemindaian penuhnya tidak berarti apa-apa —
-- perhitungan yang berbeda dari tabel klaim berisi puluhan juta baris.


-- name: login_list
--
-- Seluruh baris. Padanan `RDB List/GetLoginMemberSurveyor-SQL.xml` TANPA penyaring, yaitu
-- bentuk kueri itu ketika `InputLogin.IDIndex` kosong.
--
-- # CAKUPANNYA ADALAH REKONSTRUKSI
--
-- Rule yang mengisi page list grid layar lama (`LoginMemberSurvey.pxResults`) TIDAK ADA di
-- antara 2.634 berkas export (R-16); yang ada hanya pemuat SATU baris untuk tombol Ubah.
-- Karena itu tidak dapat dipastikan apakah grid Pega menampilkan seluruh baris atau hanya
-- baris satu tim. Lihat doc comment masterlogin.Filter untuk kedua bacaannya.
--
-- Yang dipakai di sini adalah bentuk kueri yang BENAR-BENAR ADA, tanpa penyaring.
--
-- ORDER BY NAMA — sistem lama tidak punya urutan sama sekali (`pySortType = NONE`,
-- `pyDisplayInitialSort = false` pada gridnya), sehingga urutan apa pun adalah penambahan.
-- Nama dipilih karena itulah yang dicari orang di daftar ini; LOGIN menyusul sebagai
-- pemutus supaya urutannya tetap tetap ketika ada dua nama yang sama.
SELECT NAMA,
       LOGIN,
       EMAIL,
       TELP,
       ALAMAT,
       STSLOGIN,
       LOGINLEADER
  FROM POOLDATA.MST_LOGIN_SURVEYOR
 ORDER BY NAMA, LOGIN

-- name: login_list_search
--
-- Sama dengan login_list, ditambah penyaring kata kunci.
--
-- Kueri TERSENDIRI, bukan satu kueri yang klausanya ditempel saat kata kuncinya ada —
-- lihat banner berkas ini.
--
-- TIGA kolom dicari sekaligus: Nama, Login, dan Email. Ketiganya yang dihafal orang. Telp
-- dan Alamat tidak ikut; keduanya tidak pernah menjadi cara seseorang disebut.
--
-- Kata kuncinya sudah dibungkus tanda persen oleh pemanggil, bukan di sini: menempelkannya
-- di dalam teks SQL berarti merangkai nilai ke dalam pernyataan.
--
-- TIGA parameter berbeda untuk nilai yang sama, bukan satu yang dipakai ulang. Oracle
-- mengizinkan pemakaian ulang, tetapi tidak semua driver memetakan parameter bernomor ke
-- posisi argumen dengan cara yang sama — dan D-20 menuntut kueri ini berjalan sama di kedua
-- basis data. Mengirim nilai yang sama tiga kali tidak berbiaya apa-apa; menebak perilaku
-- driver berbiaya sebuah daftar yang diam-diam kosong.
--
-- `ESCAPE '\'` disebut eksplisit karena Oracle tidak punya karakter pelolos bawaan pada
-- LIKE; lihat likePattern di berkas .go.
SELECT NAMA,
       LOGIN,
       EMAIL,
       TELP,
       ALAMAT,
       STSLOGIN,
       LOGINLEADER
  FROM POOLDATA.MST_LOGIN_SURVEYOR
 WHERE UPPER(NAMA) LIKE :1 ESCAPE '\'
    OR UPPER(LOGIN) LIKE :2 ESCAPE '\'
    OR UPPER(EMAIL) LIKE :3 ESCAPE '\'
 ORDER BY NAMA, LOGIN

-- name: login_get
--
-- Satu baris menurut kuncinya. Padanan `GetLoginMemberSurveyor` sebagaimana dipakai
-- `Activity/SetLoginSurveyorValue_act` untuk memuat baris ke form — dengan penyaring yang
-- di sana dirangkai dari teks, dan di sini menjadi parameter.
--
-- TANPA UPPER pada pembandingnya; lihat banner berkas ini.
SELECT NAMA,
       LOGIN,
       EMAIL,
       TELP,
       ALAMAT,
       STSLOGIN,
       LOGINLEADER
  FROM POOLDATA.MST_LOGIN_SURVEYOR
 WHERE TRIM(LOGIN) = :1

-- name: login_find_leader
--
-- Isi LOGINLEADER milik satu login. Padanan `RDB List/GetLoginLeaderSurveyor-SQL.xml`:
--
--   select loginleader as "NamaPasien" from pooldata.mst_login_surveyor
--    where login = {OperatorID.pyUserIdentifier}
--
-- Yang dicari adalah baris milik PENGGUNA YANG SEDANG MENYIMPAN, dan yang diambil adalah
-- kolom LOGINLEADER-nya — bukan login pengguna itu sendiri. Lihat usecase.Service.Create.
--
-- FETCH FIRST 1 ROW ONLY, bukan ROWNUM: yang dibutuhkan hanya satu nilai, dan bentuk ini
-- berjalan di kedua basis data (D-20). Ia DITAMBAHKAN terhadap rule aslinya, yang tidak
-- membatasi apa pun — tanpa constraint unik pada LOGIN (R-08), dua baris berlogin sama akan
-- membuat langkah `Property-Set` Pega menyalin `pxResults(1)` begitu saja. Membatasi di
-- sini membuat hasilnya sama, dan tidak menarik baris yang pasti dibuang.
SELECT LOGINLEADER
  FROM POOLDATA.MST_LOGIN_SURVEYOR
 WHERE TRIM(LOGIN) = :1
 ORDER BY LOGIN
 FETCH FIRST 1 ROW ONLY

-- name: login_find_by_key
--
-- Keberadaan sebuah LOGIN, tanpa memandang huruf besar-kecilnya.
--
-- Dipakai pemeriksaan keunikan pada jalur penambahan. Ia BUKAN padanan rule Pega mana pun:
-- `Activity/SetLoginSurveyor_act` memeriksa login ganda terhadap **tabel operator Pega**
-- (`Data-Admin-Operator-ID` lewat report definition `GCNMGetListOfOperators`), yang tidak
-- ada di sistem baru. Lihat masterlogin.ErrLoginTaken untuk alasan penggantinya.
--
-- UPPER di kedua sisi; lihat banner berkas ini.
SELECT NAMA,
       LOGIN,
       EMAIL,
       TELP,
       ALAMAT,
       STSLOGIN,
       LOGINLEADER
  FROM POOLDATA.MST_LOGIN_SURVEYOR
 WHERE UPPER(TRIM(LOGIN)) = :1
 ORDER BY LOGIN
 FETCH FIRST 1 ROW ONLY

-- name: login_lock_table
--
-- Mengunci tabel untuk seluruh sisa transaksi penambahan.
--
-- # Kenapa mengunci TABEL, bukan baris
--
-- Karena yang perlu dijaga adalah baris yang BELUM ADA. Dua penambahan bersamaan dengan
-- Nama yang sama menghasilkan LOGIN yang sama, dan keduanya akan lolos pemeriksaan keunikan
-- sebelum salah satunya menyisipkan. `FOR UPDATE` tidak dapat menolongnya: ia mengunci
-- baris yang sudah ada, sedangkan yang bertabrakan adalah baris yang sedang dibuat keduanya.
--
-- Sistem lama tidak menjaganya sama sekali — pemeriksaannya bahkan berjalan di LAYAR, saat
-- Nama diketik, jauh sebelum Simpan ditekan.
--
-- # Harganya, dan kenapa ia terjangkau di SINI
--
-- EXCLUSIVE menahan penulisan lain atas tabel ini sampai transaksinya selesai. Itu harga
-- yang mahal pada tabel transaksi, dan hampir gratis di sini: login surveyor ditambahkan
-- beberapa kali sebulan, transaksinya memuat tiga pernyataan pendek, dan PEMBACAAN tidak
-- terhalang di kedua basis data.
--
-- Pemilihan modus itu disengaja. Oracle EXCLUSIVE dan PostgreSQL EXCLUSIVE keduanya
-- mengizinkan SELECT berjalan terus.
--
-- # Yang TIDAK ditutupnya
--
-- Penguncian ini mengikat setiap penulis yang melewati basis data yang sama — termasuk
-- kedua instans aplikasi di belakang load balancer (D-27), dan termasuk Pega selama masa
-- paralel, karena kunci tabel ditegakkan basis data dan bukan aplikasi. Yang TIDAK
-- ditutupnya adalah baris kembar yang sudah terlanjur ada sebelum modul ini hidup.
-- Penutupnya constraint unik, dan itu menunggu DDL (R-08) serta prosedur perubahan skema
-- (D-63).
--
-- LOCK TABLE adalah satu-satunya pernyataan non-DML di seluruh modul ini. Bentuknya sama
-- persis di Oracle 19c dan PostgreSQL 17+, sehingga ia tidak melanggar D-20.
LOCK TABLE POOLDATA.MST_LOGIN_SURVEYOR IN EXCLUSIVE MODE

-- name: login_insert
--
-- Ketujuh kolomnya pada urutan yang sama dengan Pega. Padanan bagian simpan
-- `RDB List/GetLoginLeaderSurveyor-SQL.xml`:
--
--   insert into pooldata.mst_login_surveyor (nama,login,email,telp,alamat,stslogin,loginleader)
--   values ({TempLoginSurvey.SurveyName},{TempLoginSurvey.SurveyorID},{TempLoginSurvey.Email},
--           {TempLoginSurvey.Ekst},{TempLoginSurvey.BodyLetterTo},{TempLoginSurvey.ObjectName},
--           {TempLoginSurvey.NamaPasien})
--
-- Perhatikan bahwa rule INSERT-nya bernama `GetLoginLeaderSurveyor` — nama yang menyebut
-- pembacaan, pada rule yang menulis. Kedua pernyataan memang hidup di satu rule Connect-SQL
-- (`pyBrowseSQL` dan `pySaveSQL`), dan yang dinamai hanyalah yang pertama.
INSERT INTO POOLDATA.MST_LOGIN_SURVEYOR
       (NAMA, LOGIN, EMAIL, TELP, ALAMAT, STSLOGIN, LOGINLEADER)
VALUES (:1, :2, :3, :4, :5, :6, :7)

-- name: login_update
--
-- Padanan `RDB List/UpdateMasterLoginSurvey-SQL.xml` apa adanya:
--
--   update pooldata.mst_login_surveyor
--      set nama={...}, login={...}, email={...}, telp={...},
--          alamat={...}, stslogin={...}, loginleader={...}
--    where login={TempLoginSurvey.SurveyorID}
--
-- KETUJUH kolom ditulis, termasuk LOGIN yang juga menjadi penyaingnya. Ditiru apa adanya.
--
-- Bahwa LOGIN ikut di-SET adalah hal yang patut diperhatikan: bila nilainya berbeda dari
-- penyaringnya, baris itu berpindah kunci dan penyimpanan berikutnya tidak akan
-- menemukannya. Sistem lama membiarkannya mungkin karena isian Nama — asal LOGIN —
-- dikunci saat menyunting. Modul ini menutupnya dengan lebih tegas: nilai yang dikirim
-- selalu berasal dari baris yang tersimpan, bukan dari permintaan (lihat
-- usecase.Service.Save), sehingga parameter pertama dan parameter terakhir selalu sama.
UPDATE POOLDATA.MST_LOGIN_SURVEYOR
   SET NAMA        = :1,
       LOGIN       = :2,
       EMAIL       = :3,
       TELP        = :4,
       ALAMAT      = :5,
       STSLOGIN    = :6,
       LOGINLEADER = :7
 WHERE TRIM(LOGIN) = :8

-- name: login_count_all
--
-- Pencacah seluruh baris. Dipakai `claimpnc -periksa`, bukan oleh layar: layar sudah
-- menerima barisnya dan dapat menghitungnya sendiri tanpa perjalanan kedua.
SELECT COUNT(LOGIN)
  FROM POOLDATA.MST_LOGIN_SURVEYOR

-- name: login_check_table
--
-- Membuktikan tabel beserta ketujuh kolomnya benar-benar ada dan dapat dibaca, tanpa
-- menarik satu baris pun.
--
-- Dipakai `claimpnc -periksa`. Bila kolomnya berbeda dari asumsi berkas ini, kegagalannya
-- muncul di sini — saat pemeriksaan dijalankan dengan sengaja — bukan saat petugas menekan
-- Simpan.
SELECT NAMA,
       LOGIN,
       EMAIL,
       TELP,
       ALAMAT,
       STSLOGIN,
       LOGINLEADER
  FROM POOLDATA.MST_LOGIN_SURVEYOR
 WHERE 1 = 0

-- name: login_count_duplicate_key
--
-- Banyaknya LOGIN yang dipakai lebih dari satu baris, tanpa memandang huruf besar-kecilnya.
--
-- Modul ini menolak login ganda, tetapi tidak ada constraint unik yang menjaganya di basis
-- data (R-08) dan sistem lama pun tidak punya — pemeriksaannya bahkan menembak tabel yang
-- berbeda sama sekali.
--
-- Baris kembar yang sudah terlanjur ada berakibat LANGSUNG dan diam-diam: setiap
-- penyimpanan menyaring `where login = ...`, sehingga satu penyimpanan mengubah KEDUA baris
-- sekaligus, dan pemuatan ke form menampilkan salah satunya tanpa menyebutkan ada yang lain.
--
-- Dilaporkan `claimpnc -periksa` supaya keadaan itu diketahui lebih dulu, bukan ditemukan
-- oleh petugas yang perubahannya mengenai orang yang salah.
SELECT COUNT(*)
  FROM (SELECT UPPER(TRIM(LOGIN)) AS KUNCI
          FROM POOLDATA.MST_LOGIN_SURVEYOR
         GROUP BY UPPER(TRIM(LOGIN))
        HAVING COUNT(*) > 1) KEMBAR

-- name: login_count_empty_key
--
-- Baris yang LOGIN-nya kosong atau NULL.
--
-- Baris seperti itu TIDAK DAPAT dibuka, disunting, maupun ditunjuk siapa pun dari layar —
-- kuncinya kosong, dan setiap penyaring `where login = ''` akan mengenai semuanya sekaligus.
-- Sistem lama punya cacat yang sama persis: `CNMInsertMstLoginSurveyor_act` memang menolak
-- LOGIN kosong, tetapi pemeriksaan itu berjalan SETELAH baris lama sempat tersimpan lewat
-- jalur lain mana pun.
--
-- Dilaporkan `claimpnc -periksa` supaya keadaannya diketahui sebelum petugas melaporkan
-- "login saya hilang".
SELECT COUNT(*)
  FROM POOLDATA.MST_LOGIN_SURVEYOR
 WHERE LOGIN IS NULL
    OR TRIM(LOGIN) = ''

-- name: login_count_orphan_leader
--
-- Baris yang LOGINLEADER-nya menunjuk login yang tidak ada di tabel ini.
--
-- Ia pemeriksaan keutuhan tautan tim. LOGINLEADER diisi dari leader milik pengguna yang
-- menyimpan (lihat usecase.Service.Create), dan tidak ada apa pun yang menjaga bahwa nilai
-- itu masih menunjuk baris yang ada — tidak ada foreign key (R-08), dan tabelnya tidak
-- punya cara menyatakan sebuah login sudah tidak berlaku.
--
-- Baris yatim seperti itu tidak menggagalkan apa pun hari ini; ia baru berarti ketika
-- cakupan daftar diputuskan disaring per tim (lihat masterlogin.Filter). Dilaporkan supaya
-- keadaannya diketahui SEBELUM keputusan itu diambil, bukan sesudahnya.
--
-- LOGINLEADER kosong TIDAK dihitung yatim: ia keadaan yang sah dan memang mungkin terjadi
-- pada penambahan oleh pengguna yang bukan surveyor.
SELECT COUNT(*)
  FROM POOLDATA.MST_LOGIN_SURVEYOR A
 WHERE A.LOGINLEADER IS NOT NULL
   AND TRIM(A.LOGINLEADER) <> ''
   AND NOT EXISTS (
       SELECT 1
         FROM POOLDATA.MST_LOGIN_SURVEYOR B
        WHERE TRIM(B.LOGIN) = TRIM(A.LOGINLEADER))

-- name: login_count_missing_contact
--
-- Baris yang EMAIL atau TELP-nya kosong.
--
-- Keduanya WAJIB di layar Pega (`pyRequired = true`), tetapi kewajiban itu hanya ditegakkan
-- di antarmuka — `CNMInsertMstLoginSurveyor_act` sendiri hanya menolak LOGIN yang kosong.
-- Karena itu baris tanpa surel benar-benar mungkin ada pada data lama.
--
-- Akibatnya diam-diam: surel pada baris ini adalah alamat yang dipakai memberi tahu
-- surveyor tentang penugasannya, dan baris tanpa surel gagal tanpa satu pun galat.
--
-- Dilaporkan `claimpnc -periksa` supaya jumlahnya diketahui sebelum modul ini mulai menolak
-- penyimpanan yang dulu diterima — lihat masterlogin.Input.Check.
SELECT COUNT(*)
  FROM POOLDATA.MST_LOGIN_SURVEYOR
 WHERE EMAIL IS NULL
    OR TRIM(EMAIL) = ''
    OR TELP IS NULL
    OR TRIM(TELP) = ''
