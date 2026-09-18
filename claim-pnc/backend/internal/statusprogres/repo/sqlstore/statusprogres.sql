-- Kueri tabel POOLDATA.GCNM_MST_PROGRESS_KLAIM — Master Status Progres 1.
--
-- Tabel ini DITULIS aplikasi ini. Kewenangan menulisnya berpindah dari Pega ke Go saat
-- modulnya lulus gerbang 2; selama masa paralel, tepat satu sistem yang menulis
-- (ADR-0004, penulis tunggal per tabel). Selama Pega masih menjadi penulisnya, layar
-- ini harus dijalankan dalam modus baca saja di produksi.
--
-- Empat aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (D-20).
--   4. Tanpa pemanggilan stored procedure (D-02).
--
-- Asal setiap pernyataan dari export Pega disebut pada masing-masing kueri.

-- name: statusprogres_daftar
--
-- Asal: RDB List/BrowseStatusProgress-SQL.xml
--
--   SELECT ID_PROGRESS AS "CaseID", STS_PROGRESS1 AS "City", STATUS AS "CityID"
--     FROM POOLDATA.GCNM_MST_PROGRESS_KLAIM
--    ORDER BY ID_PROGRESS ASC
--
-- Ketiga alias yang menyesatkan dibuang (D-19); pengurutannya dipertahankan apa adanya
-- supaya urutan baris di layar sama dengan urutan di Pega saat uji kesetaraan.
SELECT ID_PROGRESS,
       STS_PROGRESS1,
       STATUS
  FROM POOLDATA.GCNM_MST_PROGRESS_KLAIM
 ORDER BY ID_PROGRESS ASC

-- name: statusprogres_ambil
--
-- Asal: RDB List/UpdateStatusProgress1-SQL.xml — kueri yang memuat satu baris ke modal
-- sunting. Namanya di Pega berawalan "Update" padahal ia SELECT; nama di sini
-- menyebutkan apa yang benar-benar dilakukannya.
--
-- # Kenapa TRIM, padahal kueri lama menulis `= {param}` apa adanya
--
-- Work Owner menetapkan 2026-09-17 bahwa ID_PROGRESS bertipe **CHAR berlebar tetap**.
-- Kolom CHAR memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, sehingga
-- "01" tersimpan sebagai "01 ".
--
-- Oracle membandingkan dua nilai CHAR dengan **blank-padded comparison** — spasi di
-- ujung diabaikan. Literal teks di dalam SQL bertipe CHAR, sehingga kueri lama yang
-- MERANGKAI nilainya menjadi `= '01'` memang cocok dengan "01 ". Tetapi **parameter
-- binding bertipe VARCHAR2**, dan perbandingan CHAR dengan VARCHAR2 memakai
-- **non-padded comparison**: "01 " tidak sama dengan "01", dan barisnya tidak ketemu.
--
-- Jadi menyalin `= :1` apa adanya justru MENGUBAH perilaku — bukan mempertahankannya.
-- Penyebabnya perpindahan dari perangkaian string ke parameter binding, yang wajib
-- (`08-TECHNICAL-STRATEGY.md` §4.3) dan tidak dapat ditawar.
--
-- TRIM dipilih, bukan CAST ke CHAR(n), karena lebar kolomnya belum diketahui dan TRIM
-- benar untuk lebar berapa pun. Ia juga portabel: Oracle maupun PostgreSQL sama-sama
-- mendukungnya, sehingga D-20 tidak dilanggar.
--
-- Biayanya: index atas ID_PROGRESS tidak terpakai. Dapat diterima di sini — tabel ini
-- master berbaris sedikit. Ia TIDAK boleh ditiru begitu saja pada tabel bervolume besar.
SELECT ID_PROGRESS,
       STS_PROGRESS1,
       STATUS
  FROM POOLDATA.GCNM_MST_PROGRESS_KLAIM
 WHERE TRIM(ID_PROGRESS) = :1

-- name: statusprogres_daftar_id_terkunci
--
-- Mengunci seluruh baris yang ada, lalu mengembalikan ID-nya untuk menurunkan nomor
-- berikutnya.
--
-- KENAPA FOR UPDATE. Nomor berikutnya diturunkan dari isi tabel — di Pega lewat
-- `SELECT NVL(MAX(A.ID_PROGRESS),0)+1` (RDB List/BrowseIDStatusProgress-SQL.xml) yang
-- dijalankan sebagai kueri lepas, lalu hasilnya dipakai INSERT beberapa langkah
-- kemudian (Activity/InsertMstStatusProgress1_act-Act.xml). Di antara keduanya tidak
-- ada apa pun yang menghalangi penambahan lain masuk lebih dulu, sehingga dua petugas
-- yang menambah bersamaan dapat menerima nomor yang sama.
--
-- FOR UPDATE membuat penambahan kedua menunggu sampai yang pertama selesai, lalu
-- membaca ulang termasuk baris yang baru masuk. Ia didukung Oracle maupun PostgreSQL,
-- sehingga tidak melanggar D-20. Biayanya dapat diterima: tabel ini master berbaris
-- sedikit dan nyaris tidak pernah ditulis.
--
-- Ini bukan penggantian aturan bisnis, hanya penutupan lubang balapan pada cara nomor
-- yang sama diturunkan — bentuk nomornya tetap sama (statusprogres.FormatNomor).
SELECT ID_PROGRESS
  FROM POOLDATA.GCNM_MST_PROGRESS_KLAIM
 FOR UPDATE

-- name: statusprogres_sisip
--
-- Asal: RDB List/InsertStatusProgress1-SQL.xml
--
--   INSERT INTO POOLDATA.GCNM_MST_PROGRESS_KLAIM (ID_PROGRESS, STS_PROGRESS1, STATUS)
--   VALUES ({TempInputStatus.CaseID},{TempInputStatus.City},{TempInputStatus.CityID})
--
-- ID_PROGRESS bertipe CHAR berlebar tetap, sehingga nilai yang lebih pendek dipadatkan
-- spasi oleh basis data — itu wajar dan sudah ditangani saat pembacaan. Yang TIDAK
-- ditangani di sini adalah nilai yang lebih PANJANG dari lebar kolom: basis data akan
-- menolaknya, dan penolakan itu memang yang diinginkan. Lebar kolomnya sendiri belum
-- diketahui; ia ikut diminta bersama DDL.
--
-- Ketiga kolom itu saja; tabel ini tidak punya kolom pencatat siapa dan kapan.
-- Jejak audit perubahan master (D-28, modul S-5) karena itu belum dapat disandarkan
-- pada tabel ini sendiri — dicatat sebagai keterbatasan, bukan ditambal dengan kolom
-- yang dikarang, karena menambah kolom menuntut persetujuan Work Owner dan DBA (D-63).
INSERT INTO POOLDATA.GCNM_MST_PROGRESS_KLAIM (ID_PROGRESS, STS_PROGRESS1, STATUS)
VALUES (:1, :2, :3)

-- name: statusprogres_perbarui
--
-- Asal: RDB List/UpdateStatusProgress1_sql-SQL.xml
--
--   UPDATE POOLDATA.GCNM_MST_PROGRESS_KLAIM
--      SET STS_PROGRESS1 = {TempInputStatus.City}, STATUS = {TempInputStatus.CityID}
--    WHERE ID_PROGRESS = {TempInputStatus.CaseID}
--
-- ID_PROGRESS hanya menyaring, tidak pernah ikut di-SET: ia dirujuk
-- GCNM_PROGRESS_CLAIM.STATUS_PROGRESS1 dan GCNM_MST_PROGRESS.ID_PROGRESS.
--
-- TRIM pada penyaringnya sama alasannya dengan statusprogres_ambil: kolomnya CHAR
-- berlebar tetap, dan parameter binding tidak memakai blank-padded comparison. Tanpa
-- TRIM, UPDATE ini mengenai NOL baris — dan nol baris diartikan repo sebagai
-- "tidak ditemukan", sehingga penyuntingan gagal untuk SETIAP baris tanpa satu pun
-- galat basis data yang menjelaskan sebabnya.
UPDATE POOLDATA.GCNM_MST_PROGRESS_KLAIM
   SET STS_PROGRESS1 = :1,
       STATUS        = :2
 WHERE TRIM(ID_PROGRESS) = :3

-- name: statusprogres_periksa_tabel
--
-- Memastikan tabel ada dan dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
-- Dipakai mode periksa, mengikuti pola pengguna_periksa_tabel pada modul auth.
SELECT ID_PROGRESS
  FROM POOLDATA.GCNM_MST_PROGRESS_KLAIM
 WHERE 1 = 0
