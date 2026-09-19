-- Kueri tabel POOLDATA.GCNM_MST_PROGRESS — Master Status Progres 2.
--
-- PERHATIKAN NAMA TABELNYA. Yang berakhiran `_KLAIM` adalah tingkat SATU
-- (masterstatusprogres.sql). Tingkat dua memakai nama yang lebih pendek. Keduanya sekerabat,
-- namanya nyaris sama, dan tertukar sekali saja berarti layar tingkat 2 menulis ke tabel
-- induknya.
--
-- Tabel ini DITULIS aplikasi ini. Kewenangan menulisnya berpindah dari Pega ke Go saat
-- modulnya lulus gerbang 2; selama masa paralel, tepat satu sistem yang menulis
-- (ADR-0004, penulis tunggal per tabel). Selama Pega masih menjadi penulisnya, layar ini
-- harus dijalankan dalam modus baca saja di produksi.
--
-- Empat aturan yang mengikat seluruh berkas ini, sama seperti masterstatusprogres.sql:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (D-20).
--   4. Tanpa pemanggilan stored procedure (D-02).
--
-- TIDAK ADA UPDATE DAN TIDAK ADA DELETE di berkas ini, dan itu disengaja. Seluruh export
-- Pega tidak memuat satu pun pernyataan yang mengubah isi tabel ini setelah barisnya
-- tersimpan; alasan lengkapnya ada pada doc comment masterstatusprogres.Repo2.

-- name: progress_status2_list
--
-- Asal: RDB List/BrowseStatusProgress2-SQL.xml
--
--   SELECT ID_MST AS "CaseID", STS_PROGRESS1 AS "City", STS_PROGRESS2 AS "CityID",
--          ID_PROGRESS AS "District", TIPE AS "DistrictID"
--     FROM POOLDATA.GCNM_MST_PROGRESS
--    ORDER BY ID_MST ASC
--
-- Kelima alias yang menyesatkan dibuang (D-19); pengurutannya dipertahankan apa adanya
-- supaya urutan baris di layar sama dengan urutan di Pega saat uji kesetaraan.
--
-- ORDER BY ID_MST adalah pengurutan TEKS bila kolomnya bertipe teks — "10" mendahului
-- "9". Itu perilaku kueri lama dan tidak diperbaiki di sini: memperbaikinya berarti
-- mengubah urutan yang dilihat pengguna, dan itu selisih yang tidak diminta.
--
-- TIPE ikut dibaca meski tidak pernah ditulis. Ia ditampilkan supaya nilai yang
-- benar-benar tersimpan terlihat petugas; artinya tidak diketahui (R-08).
SELECT ID_MST,
       STS_PROGRESS2,
       ID_PROGRESS,
       STS_PROGRESS1,
       TIPE
  FROM POOLDATA.GCNM_MST_PROGRESS
 ORDER BY ID_MST ASC

-- name: progress_status2_get
--
-- Asal: RDB List/UpdateStatusProgress2-SQL.xml — kueri yang memuat satu baris ke modal.
-- Namanya di Pega berawalan "Update" padahal ia SELECT; nama di sini menyebutkan apa
-- yang benar-benar dilakukannya.
--
-- TRIM pada penyaringnya sama alasannya dengan progress_status_get pada tingkat 1:
-- kolom kunci di tabel-tabel ini bertipe CHAR berlebar tetap, dan CHAR memadatkan
-- nilainya dengan spasi tanpa memberi tanda apa pun. Oracle membandingkan CHAR dengan
-- CHAR secara blank-padded — sehingga kueri lama yang MERANGKAI nilainya menjadi
-- `= '12'` tetap cocok dengan "12 ". Tetapi parameter binding bertipe VARCHAR2, dan
-- perbandingan CHAR dengan VARCHAR2 memakai non-padded comparison: "12 " tidak sama
-- dengan "12", dan barisnya tidak ketemu.
--
-- Jadi menyalin `= :1` apa adanya justru MENGUBAH perilaku, bukan mempertahankannya.
-- Penyebabnya perpindahan ke parameter binding, yang wajib dan tidak dapat ditawar
-- (`08-TECHNICAL-STRATEGY.md` §4.3).
--
-- Tipe ID_MST sendiri BELUM diketahui — DDL tabel ini belum diterima (R-08). TRIM dipilih
-- karena ia benar untuk kedua kemungkinan: pada VARCHAR2 ia tidak mengubah apa pun, pada
-- CHAR ia yang membuat barisnya ketemu. Biayanya index atas ID_MST tidak terpakai; dapat
-- diterima pada tabel master berbaris sedikit, dan TIDAK boleh ditiru pada tabel besar.
SELECT ID_MST,
       STS_PROGRESS2,
       ID_PROGRESS,
       STS_PROGRESS1,
       TIPE
  FROM POOLDATA.GCNM_MST_PROGRESS
 WHERE TRIM(ID_MST) = :1

-- name: progress_status2_list_id_locked
--
-- Mengunci seluruh baris yang ada, lalu mengembalikan ID-nya untuk menurunkan nomor
-- berikutnya.
--
-- KENAPA FOR UPDATE. Nomor berikutnya diturunkan dari isi tabel — di Pega lewat
-- `SELECT NVL(MAX(B.ID_MST),0)+1` (RDB List/BrowseIDStatusProgress-SQL.xml) yang
-- dijalankan sebagai kueri lepas, lalu hasilnya dipakai INSERT beberapa langkah kemudian
-- (Activity/InsertMstStatusProgress2_act-Act.xml). Di antara keduanya tidak ada apa pun
-- yang menghalangi penambahan lain masuk lebih dulu, sehingga dua petugas yang menambah
-- bersamaan dapat menerima nomor yang sama.
--
-- FOR UPDATE membuat penambahan kedua menunggu sampai yang pertama selesai, lalu membaca
-- ulang termasuk baris yang baru masuk. Didukung Oracle maupun PostgreSQL (D-20).
--
-- Ini bukan penggantian aturan bisnis, hanya penutupan lubang balapan pada cara nomor
-- yang sama diturunkan — bentuk nomornya tetap sama (masterstatusprogres.FormatNomor2).
SELECT ID_MST
  FROM POOLDATA.GCNM_MST_PROGRESS
 FOR UPDATE

-- name: progress_status2_insert
--
-- Asal: RDB List/InsertStatusProgress2-SQL.xml
--
--   INSERT INTO POOLDATA.GCNM_MST_PROGRESS (ID_MST,STS_PROGRESS1,STS_PROGRESS2,ID_PROGRESS)
--   VALUES ({TempInputStatus2.CaseID},{TempInputStatus2.City},
--           {TempInputStatus2.District},{TempInputStatus2.CityID})
--
-- Urutan nilainya di kueri lama tidak sejalan dengan urutan aliasnya pada SELECT:
-- di sini `City` adalah nama INDUK dan `District` adalah nama baris ini sendiri,
-- sedangkan pada SELECT `City` dialiaskan ke STS_PROGRESS1 dan `District` ke ID_PROGRESS.
-- Keduanya memang memakai page klipboard yang berbeda, sehingga tidak saling merusak —
-- tetapi itulah sebabnya alias lama tidak dipakai sebagai petunjuk arti di mana pun pada
-- modul ini. Yang dipetakan adalah KOLOMNYA, bukan aliasnya.
--
-- Empat kolom itu saja. TIPE sengaja TIDAK disebut: sistem lama pun tidak menyebutnya,
-- sehingga basis data mengisinya dengan default kolomnya sendiri. Menyebutkannya dengan
-- nilai tebakan berarti mengarang, dan menyebutkannya dengan NULL berarti memutuskan
-- sesuatu yang belum diputuskan siapa pun.
--
-- Tabel ini juga tidak punya kolom pencatat siapa dan kapan. Jejak audit perubahan master
-- (D-28, modul S-5) karena itu belum dapat disandarkan padanya — dicatat sebagai
-- keterbatasan, bukan ditambal dengan kolom yang dikarang, karena menambah kolom menuntut
-- persetujuan Work Owner dan pelaksanaan DBA (D-63).
INSERT INTO POOLDATA.GCNM_MST_PROGRESS (ID_MST, STS_PROGRESS1, STS_PROGRESS2, ID_PROGRESS)
VALUES (:1, :2, :3, :4)

-- name: progress_status2_check_table
--
-- Memastikan tabel ada dan dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
-- Dipakai mode periksa, mengikuti pola progress_status_check_table.
SELECT ID_MST
  FROM POOLDATA.GCNM_MST_PROGRESS
 WHERE 1 = 0
