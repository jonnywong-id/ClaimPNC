-- Kueri master Surveyor: POOLDATA.D_SURVEYORS.
--
-- # V_D_SURVEYORS membaca KOLOM — ditegaskan Work Owner 2026-09-20
--
-- Pertanyaan terbesar modul ini sudah terjawab, dan jawabannya menyederhanakan banyak hal:
--
--     POOLDATA.V_D_SURVEYORS membaca KOLOM, tidak memakai JSON_DATA lagi.
--
-- Artinya begitu aplikasi Go menulis ke kolom, hasilnya LANGSUNG terlihat oleh setiap rule
-- Pega yang membaca view itu — tanpa satu pun pernyataan DDL atas view-nya. Sama seperti
-- yang sudah terbukti pada M_SURVEYORS di modul induk.
--
-- Migrasi 0004 karena itu TIDAK mendefinisikan ulang view mana pun; ia hanya menambah lima
-- kolom dan dua indeks.
--
-- # Satu akibat yang perlu diketahui Tim Pega, dan BUKAN akibat modul ini
--
-- `Database/PEGA_D_SURVEYORS.prc` menulis HANYA ke JSON_DATA:
--
--     INSERT INTO POOLDATA.D_SURVEYORS(D_SURVEY_ID, JSON_DATA) VALUES (...)
--     UPDATE POOLDATA.D_SURVEYORS SET JSON_DATA = DataPega WHERE D_SURVEY_ID = IDPega
--
-- Bila view membaca kolom sementara procedure menulis JSON, maka baris yang ditulis LAYAR
-- PEGA akan meninggalkan kolomnya kosong — dan tampak hilang di view. Itu cacat yang sama
-- dengan yang ditemukan pada M_SURVEYORS, dan ia sudah berjalan hari ini, bukan dibawa
-- modul ini.
--
-- Bahwa 43 baris yang ada TERBACA lewat view berarti kolomnya memang terisi, sehingga
-- barisnya pasti diisi lewat jalur lain — migrasi data, penyisipan langsung, atau trigger.
-- Jalur mana persisnya belum diperiksa dan tidak perlu diperiksa untuk modul ini: sejak
-- Go menjadi penulis tunggal, yang ditulis adalah kolom yang memang dibaca.
--
-- # Tabel mana yang dibaca, dan tabel mana yang ditulis
--
-- Keduanya D_SURVEYORS — tabel dasarnya, bukan view. Membaca dari tabel dasar membuat apa
-- yang kita tulis dan apa yang kita baca kembali pasti sama, tanpa bergantung pada
-- definisi view yang dimiliki pihak lain.
--
-- Deskripsi tipe surveyor diambil dengan LEFT JOIN ke M_SURVEYORS — tabel yang dimiliki
-- `mastertipesurveyors`. Modul ini hanya MEMBACANYA, tidak pernah menulisnya.
--
-- # Kenapa tidak lagi lewat PEGA_D_SURVEYORS
--
-- `D-02` menetapkan logika stored procedure naik ke Go. `D-68` menambahkan alasan yang
-- lebih keras: kontrak galat procedure itu tidak dapat dipakai — parameter keluarannya
-- bernama `ErrMsg` tetapi pada jalur BERHASIL ia berisi kalimat "Data Sudah Disimpan
-- dengan ID : 1000123", sehingga pemanggil tidak dapat membedakan berhasil dari gagal
-- tanpa membaca teks.
--
-- `PEGA_D_SURVEYORS` bahkan lebih rapuh dari saudaranya: ia TIDAK memuat COMMIT sama
-- sekali, sehingga penyimpanannya bergantung penuh pada COMMIT yang dilakukan pemanggil
-- di `RDB List/UpdateDetailSurveyors-SQL.xml`. Bandingkan dengan `PEGA_M_SURVEYORS` yang
-- meng-COMMIT sendiri pada jalur insert — dua procedure bersaudara, dua perilaku
-- transaksi yang berbeda.
--
-- # Kolom yang DITAMBAHKAN modul ini
--
-- Lima kolom berikut TIDAK ADA di D_SURVEYORS dan ditambahkan migrasi 0004. Kedua Report
-- Definition yang ada — `BrowseVDSurveyors_RD` dan `SelectVDSurveyors_RD` — tidak memuat
-- satu pun dari kelimanya, jadi ini penambahan, bukan pemakaian kembali:
--
--     TGL_APPROVE_KOMITE   waktu keputusan komite
--     CATATAN_KOMITE       keterangan yang menyertai keputusan
--     USER_INPUT           siapa yang mengajukan
--     TGL_INPUT            kapan diajukan
--     USER_UPDATE          siapa yang terakhir mengubah
--
-- Alasannya bukan kerapian: `D-59` menetapkan tidak ada pemisahan tugas formal, sehingga
-- jejak audit menjadi SATU-SATUNYA kontrol pengimbang yang tersisa. Keputusan komite tanpa
-- pencatat dan tanpa waktu tidak dapat ditelusuri siapa pun.
--
-- BERBEDA DARI MIGRASI 0003, migrasi 0004 WAJIB dijalankan sebelum modul ini dipakai —
-- kelima kolom itu disentuh pada setiap pembacaan dan setiap penyimpanan.
--
-- # Aturan yang berlaku di seluruh berkas ini
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding;
-- tidak ada satu pun perangkaian teks SQL. Saringan opsional memakai bentuk
-- `(:n IS NULL OR ...)` sehingga satu teks kueri melayani seluruh kombinasi saringan —
-- pola yang sama dengan `masterrekening`, supaya kedua modul dapat dibaca bergantian
-- tanpa berpindah cara berpikir.

-- name: surveyor_list
--
-- Sepuluh argumen saringan, masing-masing dikirim DUA KALI — sekali untuk pemeriksaan
-- IS NULL, sekali untuk perbandingannya — lalu dua argumen paginasi di akhir.
--
-- LEFT JOIN, bukan INNER JOIN, dan itu disengaja: surveyor yang tipenya sudah tidak ada
-- di M_SURVEYORS tetap harus terbaca. INNER JOIN akan membuatnya HILANG dari layar tanpa
-- satu pun pesan — cacat yang paling sulit disadari, karena yang salah adalah barisnya
-- tidak muncul.
--
-- Urutannya menurut NAME, meniru ketiga kueri Pega yang membaca daftar surveyor
-- (`BrowseSurveyorTypeLossAdjuster-SQL.xml` dan dua saudaranya, seluruhnya
-- `order by NAME`). D_SURVEY_ID disertakan sebagai pemutus seri supaya urutan halaman
-- kedua tidak berubah-ubah ketika ada dua surveyor bernama sama.
SELECT d.D_SURVEY_ID,
       d.OLD_D_SURVEY_ID,
       d.M_SURVEY_ID,
       m.DESCRIPTION,
       d.NAME,
       d.ADDRESS,
       d.KDPOS,
       d.STATE,
       d.TELEPHONE,
       d.FAKSIMILE,
       d.EMAIL,
       d.OTHER_CONTACT,
       d.BRANCH,
       d.BRANCHNAME,
       d.LOGIN_APLIKASI,
       d.DOCID,
       d.APPROVAL,
       d.KOMITE,
       d.TRFKOMITE,
       CAST(NULL AS TIMESTAMP)     AS TGL_APPROVE_KOMITE,
       CAST(NULL AS VARCHAR(4000)) AS CATATAN_KOMITE,
       CAST(NULL AS VARCHAR(100))  AS USER_INPUT,
       CAST(NULL AS TIMESTAMP)     AS TGL_INPUT,
       CAST(NULL AS VARCHAR(100))  AS USER_UPDATE
  FROM POOLDATA.D_SURVEYORS d
  LEFT JOIN POOLDATA.M_SURVEYORS m ON m.M_SURVEY_ID = d.M_SURVEY_ID
 WHERE (:1 IS NULL OR d.APPROVAL = :2)
   AND (:3 IS NULL OR UPPER(d.NAME) LIKE '%' || UPPER(:4) || '%')
   AND (:5 IS NULL OR UPPER(d.LOGIN_APLIKASI) LIKE '%' || UPPER(:6) || '%')
   AND (:7 IS NULL OR d.M_SURVEY_ID = :8)
   AND (:9 IS NULL OR (UPPER(TRIM(d.KOMITE)) = UPPER(:10) AND d.APPROVAL = '0'))
 ORDER BY d.NAME, d.D_SURVEY_ID
OFFSET :11 ROWS FETCH NEXT :12 ROWS ONLY

-- name: surveyor_count
--
-- Menghitung seluruh baris yang cocok SEBELUM dipotong paginasi. Saringannya wajib sama
-- persis dengan surveyor_list — bila keduanya berbeda, jumlah halaman yang ditampilkan
-- tidak akan cocok dengan isinya.
--
-- Tidak ada LEFT JOIN di sini: menghitung tidak membutuhkan deskripsi tipe, dan join yang
-- tidak dipakai hanya menambah kerja basis data pada setiap pembacaan halaman.
SELECT COUNT(*)
  FROM POOLDATA.D_SURVEYORS d
 WHERE (:1 IS NULL OR d.APPROVAL = :2)
   AND (:3 IS NULL OR UPPER(d.NAME) LIKE '%' || UPPER(:4) || '%')
   AND (:5 IS NULL OR UPPER(d.LOGIN_APLIKASI) LIKE '%' || UPPER(:6) || '%')
   AND (:7 IS NULL OR d.M_SURVEY_ID = :8)
   AND (:9 IS NULL OR (UPPER(TRIM(d.KOMITE)) = UPPER(:10) AND d.APPROVAL = '0'))

-- name: surveyor_get
SELECT d.D_SURVEY_ID,
       d.OLD_D_SURVEY_ID,
       d.M_SURVEY_ID,
       m.DESCRIPTION,
       d.NAME,
       d.ADDRESS,
       d.KDPOS,
       d.STATE,
       d.TELEPHONE,
       d.FAKSIMILE,
       d.EMAIL,
       d.OTHER_CONTACT,
       d.BRANCH,
       d.BRANCHNAME,
       d.LOGIN_APLIKASI,
       d.DOCID,
       d.APPROVAL,
       d.KOMITE,
       d.TRFKOMITE,
       CAST(NULL AS TIMESTAMP)     AS TGL_APPROVE_KOMITE,
       CAST(NULL AS VARCHAR(4000)) AS CATATAN_KOMITE,
       CAST(NULL AS VARCHAR(100))  AS USER_INPUT,
       CAST(NULL AS TIMESTAMP)     AS TGL_INPUT,
       CAST(NULL AS VARCHAR(100))  AS USER_UPDATE
  FROM POOLDATA.D_SURVEYORS d
  LEFT JOIN POOLDATA.M_SURVEYORS m ON m.M_SURVEY_ID = d.M_SURVEY_ID
 WHERE d.D_SURVEY_ID = :1

-- name: surveyor_by_name_key
--
-- Ekspresi keunikannya HARUS sama persis dengan mastersurveyors.NameKey di kode Go:
-- huruf diseragamkan DAN seluruh spasi dibuang. Ia meniru
-- `Activity/ValidasiMasterSurveyor-Act.xml`:
--
--     @toUpperCase(@replaceAll(.NAME," ","")) == @toUpperCase(@replaceAll(TempDetailSurveyors.NAME," ",""))
--
-- REPLACE ditulis dengan TIGA argumen, bukan dua. Bentuk dua argumen sah di Oracle dan
-- lebih ringkas, tetapi tidak ada di PostgreSQL — dan `D-20` menetapkan satu set SQL yang
-- berjalan di keduanya. Indeks unik pada migrasi 0004 memakai ekspresi YANG SAMA,
-- sehingga pemeriksaan di sini dan penegakan di sana tidak dapat berbeda pendapat.
SELECT d.D_SURVEY_ID,
       d.OLD_D_SURVEY_ID,
       d.M_SURVEY_ID,
       m.DESCRIPTION,
       d.NAME,
       d.ADDRESS,
       d.KDPOS,
       d.STATE,
       d.TELEPHONE,
       d.FAKSIMILE,
       d.EMAIL,
       d.OTHER_CONTACT,
       d.BRANCH,
       d.BRANCHNAME,
       d.LOGIN_APLIKASI,
       d.DOCID,
       d.APPROVAL,
       d.KOMITE,
       d.TRFKOMITE,
       CAST(NULL AS TIMESTAMP)     AS TGL_APPROVE_KOMITE,
       CAST(NULL AS VARCHAR(4000)) AS CATATAN_KOMITE,
       CAST(NULL AS VARCHAR(100))  AS USER_INPUT,
       CAST(NULL AS TIMESTAMP)     AS TGL_INPUT,
       CAST(NULL AS VARCHAR(100))  AS USER_UPDATE
  FROM POOLDATA.D_SURVEYORS d
  LEFT JOIN POOLDATA.M_SURVEYORS m ON m.M_SURVEY_ID = d.M_SURVEY_ID
 WHERE UPPER(REPLACE(d.NAME, ' ', '')) = :1

-- name: surveyor_by_login
--
-- Pemeriksaan bentrok nama login DI DALAM master ini sendiri.
--
-- Ia BUKAN pemeriksaan yang sama dengan yang dilakukan sistem lama. Sistem lama
-- membandingkannya terhadap Operator ID Pega — langkah 10 sampai 14
-- `CNMInsertDetailSurveyors_act`. Pemeriksaan terhadap identitas sistem ada di pengisi
-- seam AccountRegistrar, yang mengetahui identitas apa saja yang sudah ada; yang di sini
-- hanya menjaga agar dua surveyor tidak memakai satu nama login.
--
-- Baris ber-LOGIN_APLIKASI kosong tidak pernah ikut: surveyor eksternal boleh tidak punya
-- login, dan dua-duanya kosong bukan bentrok.
SELECT d.D_SURVEY_ID,
       d.OLD_D_SURVEY_ID,
       d.M_SURVEY_ID,
       m.DESCRIPTION,
       d.NAME,
       d.ADDRESS,
       d.KDPOS,
       d.STATE,
       d.TELEPHONE,
       d.FAKSIMILE,
       d.EMAIL,
       d.OTHER_CONTACT,
       d.BRANCH,
       d.BRANCHNAME,
       d.LOGIN_APLIKASI,
       d.DOCID,
       d.APPROVAL,
       d.KOMITE,
       d.TRFKOMITE,
       CAST(NULL AS TIMESTAMP)     AS TGL_APPROVE_KOMITE,
       CAST(NULL AS VARCHAR(4000)) AS CATATAN_KOMITE,
       CAST(NULL AS VARCHAR(100))  AS USER_INPUT,
       CAST(NULL AS TIMESTAMP)     AS TGL_INPUT,
       CAST(NULL AS VARCHAR(100))  AS USER_UPDATE
  FROM POOLDATA.D_SURVEYORS d
  LEFT JOIN POOLDATA.M_SURVEYORS m ON m.M_SURVEY_ID = d.M_SURVEY_ID
 WHERE d.LOGIN_APLIKASI IS NOT NULL
   AND UPPER(TRIM(d.LOGIN_APLIKASI)) = :1

-- name: surveyor_site
--
-- Kode situs, bagian pertama dari setiap D_SURVEY_ID. Meniru
-- `Database/PEGA_D_SURVEYORS.prc:11` persis, termasuk pembandingnya yang berupa teks '1'
-- dan bukan angka.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: surveyor_next_sequence
--
-- Urutan yang SAMA dengan yang dipakai procedure lama, supaya kode yang diterbitkan
-- aplikasi ini melanjutkan deret yang sudah ada dan tidak pernah bertabrakan dengan kode
-- yang pernah diterbitkan Pega.
--
-- PERHATIKAN: urutannya D_SURVEYORS_SEQ, BUKAN M_SURVEYORS_SEQ. Keduanya ada, keduanya
-- bernama mirip, dan keduanya dipakai membentuk kode dengan jumlah digit yang BERBEDA —
-- enam di sini, tiga di master tipe. Tertukar berarti kode yang terbit salah panjang dan
-- bertabrakan dengan deret milik master lain.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri.
SELECT POOLDATA.D_SURVEYORS_SEQ.NEXTVAL
  FROM DUAL

-- name: surveyor_insert
--
-- JSON_DATA sengaja TIDAK diisi. Baris baru meninggalkannya NULL, dan itu keputusan yang
-- sama dengan modul induk: yang ditulis adalah kolom yang memang dibaca — dan sejak
-- 2026-09-20 sudah ditegaskan bahwa V_D_SURVEYORS memang membaca kolom.
--
-- OLD_D_SURVEY_ID tidak diisi: ia jejak penomoran sistem sebelumnya, bukan field yang
-- dikelola. TGL_INPUT memakai CURRENT_TIMESTAMP, bukan SYSDATE — `D-20` melarang SYSDATE
-- karena tidak portabel, dan `F-5` menetapkan waktu disimpan sebagai UTC.
-- USER_INPUT dan TGL_INPUT juga tidak diisi, dan bukan karena dilupakan: keduanya BELUM
-- ADA di POOLDATA.D_SURVEYORS — migrasi `0004_master_surveyor` yang membuatnya, dan
-- migrasi itu belum dijalankan. Pega pun tidak punya keduanya.
INSERT INTO POOLDATA.D_SURVEYORS (
       D_SURVEY_ID, M_SURVEY_ID, NAME, ADDRESS, KDPOS, STATE,
       TELEPHONE, FAKSIMILE, EMAIL, OTHER_CONTACT,
       BRANCH, BRANCHNAME, LOGIN_APLIKASI, DOCID,
       APPROVAL, KOMITE, TRFKOMITE)
VALUES (:1, :2, :3, :4, :5, :6,
        :7, :8, :9, :10,
        :11, :12, :13, :14,
        :15, :16, :17)

-- name: surveyor_update
--
-- Empat kolom sengaja TIDAK ikut diubah:
--
--     D_SURVEY_ID      kunci, tidak pernah berubah
--     OLD_D_SURVEY_ID  jejak sejarah
--     KOMITE           ditetapkan SEKALI saat pengajuan pertama. Prasyarat aslinya
--                      `TempDetailSurveyors.KOMITE==""` pada langkah 6 — menetapkannya
--                      ulang pada setiap penyuntingan akan memindahkan kewenangan
--                      memutuskan ke orang lain di tengah jalan
--     USER_INPUT       pengaju pertama
--
-- # Tiga kolom jejak yang TIDAK ditulis, dan kenapa
--
-- TGL_APPROVE_KOMITE, CATATAN_KOMITE, dan USER_UPDATE **tidak ada di POOLDATA.D_SURVEYORS**
-- — ketiganya baru dibuat migrasi `0004_master_surveyor`, dan migrasi itu belum dijalankan.
-- Ketiganya juga tidak ada di Pega: layar lamanya tidak pernah mencatat siapa memutuskan,
-- kapan, dan dengan catatan apa.
--
-- Keputusan Work Owner 2026-09-22: ikuti Pega dan tulis LANGSUNG ke kolom yang ada, bukan
-- ke JSON. Ketiganya karena itu tidak ditulis sama sekali. Keputusan komitenya sendiri
-- TETAP tersimpan — ia ada di kolom APPROVAL yang memang sudah ada; yang hilang hanyalah
-- jejak siapa dan kapan.
--
-- Konsekuensinya diterima secara sadar dan perlu diketahui: `D-59` menjadikan jejak audit
-- satu-satunya kontrol pengimbang karena tidak ada pemisahan tugas. Sampai migrasi 0004
-- dijalankan, persetujuan surveyor tidak meninggalkan jejak pelaku.
UPDATE POOLDATA.D_SURVEYORS
   SET M_SURVEY_ID        = :1,
       NAME               = :2,
       ADDRESS            = :3,
       KDPOS              = :4,
       STATE              = :5,
       TELEPHONE          = :6,
       FAKSIMILE          = :7,
       EMAIL              = :8,
       OTHER_CONTACT      = :9,
       BRANCH             = :10,
       BRANCHNAME         = :11,
       LOGIN_APLIKASI     = :12,
       DOCID              = :13,
       APPROVAL           = :14,
       TRFKOMITE          = :15
 WHERE D_SURVEY_ID        = :16

-- name: surveyor_check_table
--
-- Memastikan seluruh kolom yang dipakai modul ini ada dan dapat dibaca akun aplikasi,
-- tanpa mengambil satu baris pun. Aman dijalankan terhadap produksi, dan dipakai untuk
-- membedakan dua sebab kegagalan yang tampak mirip: kolomnya belum ada karena migrasi
-- 0004 belum dijalankan, versus tidak punya hak baca.
SELECT d.D_SURVEY_ID,
       d.OLD_D_SURVEY_ID,
       d.M_SURVEY_ID,
       d.NAME,
       d.ADDRESS,
       d.KDPOS,
       d.STATE,
       d.TELEPHONE,
       d.FAKSIMILE,
       d.EMAIL,
       d.OTHER_CONTACT,
       d.BRANCH,
       d.BRANCHNAME,
       d.LOGIN_APLIKASI,
       d.DOCID,
       d.APPROVAL,
       d.KOMITE,
       d.TRFKOMITE,
       CAST(NULL AS TIMESTAMP)     AS TGL_APPROVE_KOMITE,
       CAST(NULL AS VARCHAR(4000)) AS CATATAN_KOMITE,
       CAST(NULL AS VARCHAR(100))  AS USER_INPUT,
       CAST(NULL AS TIMESTAMP)     AS TGL_INPUT,
       CAST(NULL AS VARCHAR(100))  AS USER_UPDATE
  FROM POOLDATA.D_SURVEYORS d
 WHERE 1 = 0
