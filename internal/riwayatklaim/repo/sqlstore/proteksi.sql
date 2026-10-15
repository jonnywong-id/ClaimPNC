-- Kueri gerbang proteksi data untuk layar View History Claim.
--
-- ============================================================================
-- DUA TABEL, DAN KENAPA YANG SATU HANYA DIBACA
-- ============================================================================
--
--   POOLDATA.MST_PROTEKSI_DATA_PNC   milik sistem lama  — HANYA DIBACA
--   POOLDATA.CPNC_PEMAKAIAN_PROTEKSI milik aplikasi ini — ditulis (migrasi 0004)
--
-- Sistem lama mengurangi jatah dengan mengubah tabelnya sendiri:
--
--     update POOLDATA.MST_PROTEKSI_DATA_PNC set LOGSEARCH = <sisa-1>
--      where LOGIN = <operator> and MODUL = 'PNCSearchKlaim'
--
-- Menirunya berarti aplikasi ini dan Pega sama-sama menulis satu tabel selama masa
-- paralel — tepat yang dilarang `P-1`. Akibatnya bukan galat melainkan jatah yang saling
-- menimpa: Pega menulis sisa menurut hitungannya, aplikasi ini menurut hitungannya, dan
-- yang menulis belakangan menang tanpa meninggalkan jejak.
--
-- Work Owner memutuskan 2026-09-20 penulisannya diarahkan ke tabel baru milik aplikasi,
-- mengikuti pola yang sama dengan modul Pelaporan Klaim. Sisa jatah karena itu DIHITUNG,
-- bukan disimpan: jatah menurut master dikurangi pemakaian yang tercatat di sini.
--
-- ============================================================================
-- ALIAS YANG MENYESATKAN, SEKALI LAGI
-- ============================================================================
--
-- Kueri lama membaca master proteksi dengan alias yang tidak menyatakan isinya:
--
--     LOGSEEN    as "City"        LOGSEARCH as "CityID"     STS_NOTELP as "Country"
--     STS_EMAIL  as "CountryID"   STS_KTP   as "Province"   SUBMODUL   as "ProvinceID"
--
-- Arti keenamnya hanya terbaca dari komentar langkah 3
-- `Activity/InsertLogProteksiDataKlaimMasking-Act.xml`: "1: log seen(city), 2: log search
-- (CityID), 3 STS_telp(Country), 4 sts_email(CountryID), 5: stsktp (Province)". Tanpa
-- komentar itu, tidak ada cara mengetahui bahwa "City" berarti jatah lihat data.
--
-- Di sini kolomnya disebut dengan nama aslinya.

-- name: protection_find
-- Baris proteksi milik satu pengguna untuk satu modul.
--
-- Kueri lama TIDAK menyaring STS_AKTF di jalur ini — berbeda dari
-- `CekmaskingDataPerLoginUserKlaim` yang menyaring `STS_AKTF='AKTIF'` untuk keperluan
-- lain. Perbedaan itu dipertahankan apa adanya: menambahkan saringan yang tidak ada di
-- sistem lama akan menolak pengguna yang hari ini dapat masuk, dan penolakan itu akan
-- terbaca sebagai kerusakan modul, bukan sebagai pengetatan yang disengaja.
--
-- FETCH NEXT 1 ROW ONLY menjaga satu baris hasil meski master memuat baris kembar untuk
-- pasangan LOGIN + MODUL yang sama. Tabelnya tidak punya kunci unik yang mencegahnya, dan
-- kueri lama pun mengambil satu baris pertama (`pxResults(1)`).
SELECT p.LOGSEARCH,
       p.LOGSEEN,
       p.STS_NOTELP,
       p.STS_EMAIL,
       p.STS_KTP,
       p.SUBMODUL
  FROM POOLDATA.MST_PROTEKSI_DATA_PNC p
 WHERE UPPER(TRIM(p.LOGIN)) = UPPER(TRIM(:1))
   AND UPPER(TRIM(p.MODUL)) = UPPER(TRIM(:2))
 FETCH NEXT 1 ROW ONLY

-- name: protection_count_usage
-- Jumlah pemakaian yang MENGURANGI jatah.
--
-- Baris yang hanya mencatat pencarian (MEMAKAI_JATAH = 0) tidak ikut terhitung. Keduanya
-- tinggal di tabel yang sama supaya jejak audit tetap utuh dalam satu urutan waktu —
-- memisahkannya menjadi dua tabel membuat "apa yang terjadi pada kunjungan itu" harus
-- disusun ulang dari dua tempat.
SELECT COUNT(*)
  FROM POOLDATA.CPNC_PEMAKAIAN_PROTEKSI u
 WHERE UPPER(TRIM(u.LOGIN)) = UPPER(TRIM(:1))
   AND UPPER(TRIM(u.MODUL)) = UPPER(TRIM(:2))
   AND u.MEMAKAI_JATAH = 1

-- name: protection_record_usage
-- Mencatat satu pemakaian.
--
-- ID diambil dari urutan di dalam pernyataan yang sama, bukan diminta lebih dulu lalu
-- dipakai beberapa langkah kemudian: jarak antara mengambil nomor dan memakainya adalah
-- jarak yang membuat dua pencatatan bersamaan menerima nomor yang sama.
INSERT INTO POOLDATA.CPNC_PEMAKAIAN_PROTEKSI
       (ID, LOGIN, MODUL, TIPE_PENCARIAN, NILAI_PENCARIAN, MEMAKAI_JATAH, DIPAKAI_PADA)
VALUES (POOLDATA.CPNC_PEMAKAIAN_PROTEKSI_SEQ.NEXTVAL, :1, :2, :3, :4, :5, :6)

-- name: protection_check_table
-- Memastikan tabel pemakaian sudah ada, untuk mode -periksa.
--
-- Ia TIDAK membaca satu baris pun data: yang ditanyakan hanya keberadaan objeknya, supaya
-- perintah periksa dapat dijalankan terhadap basis data produksi tanpa menyentuh isinya.
SELECT COUNT(*)
  FROM ALL_TABLES t
 WHERE t.OWNER = 'POOLDATA'
   AND t.TABLE_NAME = 'CPNC_PEMAKAIAN_PROTEKSI'
