-- Kueri Master Pasal Kerugian.
--
--   POOLDATA.V_M_DATA_PASAL   IDDATA (kunci) · IDPASAL (No Pasal) · JSONPASAL (CLOB)
--   POOLDATA.BUSINESS         ID · NOTE — master lini bisnis, HANYA DIBACA
--
-- Tabel pertama DITULIS aplikasi ini. Kewenangan menulisnya berpindah dari Pega ke Go saat
-- modulnya lulus gerbang 2; selama masa paralel, tepat satu sistem yang menulis
-- (ADR-0004, penulis tunggal per tabel). Selama Pega masih menjadi penulisnya, layar ini
-- harus dijalankan dalam modus baca saja di produksi.
--
-- Tabel kedua milik ruleset GISFW dan TIDAK PERNAH ditulis di sini.
--
-- Lima aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (D-20).
--   4. Tanpa pemanggilan stored procedure (D-02). `PEGA_D_PASAL_MASTER` logikanya naik
--      ke Go; objeknya boleh ditinggalkan (D-68).
--   5. ADA DELETE di sini, dan itu pengecualian yang disadari terhadap D-66. Alasan
--      lengkapnya ada pada doc comment masterpasal.Repo. Ringkasnya: layar lama punya
--      tombolnya, tabelnya tidak punya kolom penanda terhapus, dan Work Owner memilih
--      "jalankan as is" pada 2026-09-19.
--
-- # SATU HAL YANG TIDAK DIBAWA DARI PEGA, DAN KENAPA
--
-- Pega membaca daftar lini bisnis sebuah pasal dari view `POOLDATA.View_DATA_PASAL`
-- (`RDB List/BrowseCOLByPasalDataBisnis_Sql-SQL.xml`). View itu TIDAK ADA di export dan
-- DDL-nya belum diterima (`R-08`), sehingga isinya tidak dapat dibaca siapa pun di tim
-- ini — dan ia hampir pasti memakai `JSON_TABLE` khas Oracle yang tidak dapat dipindahkan
-- apa adanya.
--
-- Yang dipakai sebagai gantinya: `JSONPASAL` dibaca UTUH lalu dibongkar di Go. Hasilnya
-- sama — daftar lini bisnis yang sama, dari dokumen yang sama — tetapi tanpa bergantung
-- pada objek basis data yang tidak dapat dibaca maupun dipindahkan. Kueri lamanya juga
-- merangkai penyaringnya dari `{ASIS:TempSearchBisnis.DESCRIPTION}`, dan pola itu memang
-- dilarang (§4.3 `08-TECHNICAL-STRATEGY.md`).

-- name: clause_list
--
-- Asal: RDB List/GetDataCOLByPasalBisnis_Sql-SQL.xml
--
--   SELECT IDDATA as "OLD_M_COL_ID", IDPASAL as "M_COL_ID",
--          json_value (JSONPASAL, '$.DESCRIPTION')  as "DESCRIPTION",
--          json_value (JSONPASAL, '$.OLD_D_COL_ID') as "OLD_D_COL_ID",
--          json_value (JSONPASAL, '$.pyCountry')    as "pyCountry",
--          json_value (JSONPASAL, '$.LOSS_CODE')    as "LOSS_CODE"
--     FROM POOLDATA.V_M_DATA_PASAL
--
-- DUA hal yang berbeda dari kueri lama, keduanya disengaja:
--
-- 1. KEEMPAT `json_value` DIGANTI SATU KOLOM `JSONPASAL` yang dibongkar di Go. Ia lebih
--    portabel — `json_value` baru tersedia di PostgreSQL 17 (D-24), dan ketersediaannya
--    tidak perlu dipertaruhkan untuk sesuatu yang dapat dikerjakan di aplikasi — dan ia
--    membaca dokumen yang SAMA sekali jalan, sehingga daftar lini bisnis ikut terbaca
--    tanpa kueri kedua.
--
-- 2. `ORDER BY` DITAMBAHKAN. Kueri lama tidak punya satu pun, sehingga urutan barisnya
--    adalah apa pun yang dikembalikan basis data — dan dapat berbeda antar pemanggilan.
--    Ini SELISIH TERENCANA: yang berubah hanyalah urutan, bukan barisnya.
--
--    Diurutkan menurut IDPASAL — No Pasal — karena itulah yang dibaca manusia di kolom
--    pertama. Ia pengurutan TEKS pada kolom bertipe teks, sehingga "10" mendahului "9";
--    itu diterima apa adanya, sama seperti pada modul master lain, karena memperbaikinya
--    menuntut menebak bahwa isinya selalu angka — dan No Pasal justru nomor yang diketik
--    bebas.
--
--    IDDATA menjadi kunci urutan KEDUA. No Pasal tidak dijamin unik (lihat Clause.Number),
--    sehingga tanpa kunci kedua dua baris bernomor sama dapat bertukar tempat di antara
--    dua pemuatan.
SELECT IDDATA,
       IDPASAL,
       JSONPASAL
  FROM POOLDATA.V_M_DATA_PASAL
 ORDER BY IDPASAL ASC, IDDATA ASC

-- name: clause_get
--
-- Membaca satu baris untuk dimuat ke form.
--
-- Padanan `Activity/PNCGetListPasalDataCOL_Act-Act.xml` langkah 7, yang menyaring
-- `OLD_M_COL_ID = '<IDDATA>'` lewat rangkaian teks:
--
--   TempSearchBisnis.DESCRIPTION := "OLD_M_COL_ID ='" + Param.idstatusp + "'"
--
-- Rangkaian itu tidak dibawa; penyaringnya menjadi parameter terikat.
--
-- TRIM pada penyaringnya sama alasannya dengan modul master lain: kolom kunci di
-- tabel-tabel warisan ini bertipe teks yang lebarnya belum diketahui (R-08), dan bila ia
-- CHAR berlebar tetap, nilainya dipadatkan dengan spasi tanpa memberi tanda apa pun.
-- Oracle membandingkan CHAR dengan CHAR secara blank-padded — sehingga rangkaian teks
-- `= '12'` pada kueri lama tetap cocok dengan "12 ". Tetapi parameter binding bertipe
-- VARCHAR2, dan perbandingan CHAR dengan VARCHAR2 memakai non-padded comparison: "12 "
-- tidak sama dengan "12", dan barisnya tidak ketemu.
--
-- Jadi menyalin `= :1` apa adanya justru MENGUBAH perilaku, bukan mempertahankannya.
-- Biayanya index atas IDDATA tidak terpakai; dapat diterima pada tabel master berbaris
-- sedikit, dan TIDAK boleh ditiru pada tabel besar.
SELECT IDDATA,
       IDPASAL,
       JSONPASAL
  FROM POOLDATA.V_M_DATA_PASAL
 WHERE TRIM(IDDATA) = :1

-- name: clause_list_id_locked
--
-- Mengunci seluruh baris, lalu mengembalikan IDDATA-nya untuk menurunkan nomor berikutnya.
--
-- KENAPA FOR UPDATE. Nomor berikutnya diturunkan dari isi tabel — di procedure lama lewat
-- `select max(TO_NUMBER(IDDATA)) INTO JUM_PASAL` (`PEGA_D_PASAL_MASTER.prc:11`) yang
-- dijalankan sebagai pernyataan lepas, lalu hasilnya dipakai INSERT tiga baris kemudian.
-- Di antara keduanya tidak ada apa pun yang menghalangi penambahan lain masuk lebih dulu,
-- sehingga dua petugas yang menambah bersamaan dapat menerima nomor yang sama.
--
-- FOR UPDATE membuat penambahan kedua menunggu sampai yang pertama selesai, lalu membaca
-- ulang termasuk baris yang baru masuk. Didukung Oracle maupun PostgreSQL (D-20).
--
-- Hanya IDDATA yang diambil: JSONPASAL adalah CLOB, dan membaca seluruhnya hanya untuk
-- menghitung nomor berarti menarik seluruh isi tabel ke memori pada setiap penambahan.
SELECT IDDATA
  FROM POOLDATA.V_M_DATA_PASAL
 FOR UPDATE

-- name: clause_insert
--
-- Asal: Database/PEGA_D_PASAL_MASTER.prc:14
--
--   INSERT INTO POOLDATA.V_M_DATA_PASAL(IDDATA,IDPASAL,JSONPASAL)
--   VALUES(to_char(JUM_PASAL), IDPasal_p, Datapega);
--
-- Ketiga kolom itu saja — tabelnya memang tidak punya kolom lain. Tidak ada kolom
-- pencatat siapa dan kapan, sehingga jejak audit perubahan master (D-28, modul S-5) belum
-- dapat disandarkan padanya. Dicatat sebagai keterbatasan, bukan ditambal dengan kolom
-- yang dikarang: menambah kolom menuntut persetujuan Work Owner dan pelaksanaan DBA
-- (D-63).
--
-- KETERBATASAN ITU MENGGIGIT PALING KERAS PADA DELETE. Baris yang dihapus tidak
-- meninggalkan jejak apa pun — tidak ada yang tahu siapa menghapusnya dan kapan.
--
-- JSONPASAL bertipe CLOB dan diikat sebagai teks biasa. Nilai yang lebih panjang dari
-- 4000 karakter dapat ditolak driver dengan ORA-01461 bila ia mengikatnya sebagai
-- VARCHAR2. Belum dapat dibuktikan di lingkungan ini karena tidak ada Oracle untuk
-- diuji; dicatat sebagai hal yang WAJIB dicoba pada basis data sungguhan sebelum modul
-- dinyatakan lulus, dengan satu pasal berisi teks panjang.
INSERT INTO POOLDATA.V_M_DATA_PASAL (IDDATA, IDPASAL, JSONPASAL)
VALUES (:1, :2, :3)

-- name: clause_update
--
-- Asal: Database/PEGA_D_PASAL_MASTER.prc:25
--
--   UPDATE POOLDATA.V_M_DATA_PASAL
--      SET IDPASAL = IDPasal_p, JSONPASAL = Datapega
--    WHERE IDDATA = IDdatap;
--
-- IDDATA tidak pernah ikut di-SET: ia kunci baris, dan procedure lama pun hanya
-- memakainya sebagai penyaring WHERE. TRIM pada penyaring, alasannya sama seperti
-- clause_get.
UPDATE POOLDATA.V_M_DATA_PASAL
   SET IDPASAL   = :1,
       JSONPASAL = :2
 WHERE TRIM(IDDATA) = :3

-- name: clause_delete
--
-- Asal: RDB List/DeleteDataPasalDataMaster-SQL.xml
--
--   BEGIN
--     Delete from POOLDATA.V_M_DATA_PASAL where IDDATA = {InputData.OLD_M_COL_ID} ;
--     COMMIT;
--   END;
--
-- Blok PL/SQL-nya tidak dibawa, dan `COMMIT` di dalamnya pun tidak: kepemilikan transaksi
-- berpindah sepenuhnya ke Go (D-68). Yang tersisa adalah pernyataannya sendiri.
--
-- INI PENGHAPUSAN FISIK, dan ia menyupersede D-66 untuk tabel ini atas keputusan Work
-- Owner 2026-09-19. Barisnya tidak dapat dipulihkan dan tidak meninggalkan jejak.
DELETE FROM POOLDATA.V_M_DATA_PASAL
 WHERE TRIM(IDDATA) = :1

-- name: clause_check_table
--
-- Memastikan tabel ada dan dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
SELECT IDDATA
  FROM POOLDATA.V_M_DATA_PASAL
 WHERE 1 = 0

-- name: business_search
--
-- Daftar pilihan lini bisnis pada isian "Bisnis".
--
-- Asal: autocomplete `.Note` pada Section/BrowsePasalDeatailMaster-Section.xml, yang
-- bersumber Report Definition/BrowseBusiness_RD-RD.xml atas kelas
-- `ASM-FW-GISFW-Int-BUSINESS` — yaitu POOLDATA.BUSINESS. Kolom yang ditampilkan `.Note`,
-- dan yang ikut disetel saat dipilih `.ID`.
--
-- Report Definition itu tidak menyaring apa pun dan memakai `pyMaxRecords=200`: ia
-- menarik dua ratus baris pertama lalu menyaringnya di peramban. Di sini penyaringnya
-- pindah ke basis data, sehingga yang dikirim hanyalah yang benar-benar cocok.
--
-- UPPER dipasang di KEDUA sisi supaya pencarian tidak bergantung pada besar-kecil huruf
-- yang kebetulan tersimpan. `ESCAPE '\'` disebut eksplisit karena Oracle TIDAK punya
-- karakter pelolos bawaan pada LIKE; PostgreSQL memakai backslash sebagai bawaan, dan
-- menyebutkannya membuat keduanya berperilaku sama (D-20).
--
-- `CAST(ID AS VARCHAR(64))` pada cabang kedua: bila ID bertipe angka, perbandingan
-- langsung memaksa basis data mengubah kata kunci menjadi angka — dan kata kunci berupa
-- nama akan menghasilkan ORA-01722, menggagalkan SELURUH pencarian alih-alih hanya
-- bagian itu. Tipe ID sendiri belum diketahui (R-08).
--
-- FETCH NEXT membatasi hasil; angkanya sama dengan masterpasal.MaxLookupRows, dan
-- query_test.go menjaga keduanya tidak berpisah diam-diam.
SELECT ID,
       NOTE
  FROM POOLDATA.BUSINESS
 WHERE UPPER(NOTE) LIKE :1 ESCAPE '\'
    OR CAST(ID AS VARCHAR(64)) = :2
 ORDER BY NOTE ASC
 FETCH NEXT 50 ROWS ONLY

-- name: business_get
--
-- Membaca nama sebuah lini bisnis dari kodenya.
--
-- Inilah padanan sub-kueri pada RDB List/BrowseCOLByPasalDataBisnis_Sql-SQL.xml:
--
--   (select NOTE from BUSINESS c where c.ID = A.D_COL_ID) as "LOSS_CODE"
--
-- Perhatikan alias `LOSS_CODE` di sana TIDAK ada hubungannya dengan `$.LOSS_CODE` di
-- dalam JSONPASAL, yang berisi sebutan kategori. Satu nama, dua isi yang berbeda — dan
-- itulah sebabnya yang dipetakan di modul ini selalu kolomnya, bukan aliasnya (D-19).
--
-- Ia dipanggil sekali per lini bisnis yang menempel pada satu pasal, bukan sekali untuk
-- seluruh daftar. Jumlahnya kecil — ia daftar lini bisnis satu pasal, bukan satu tabel —
-- dan bentuk ini tidak menuntut merangkai klausa IN yang panjangnya berubah-ubah, yang
-- pada parameter binding berarti satu teks kueri berbeda untuk setiap jumlah baris.
SELECT ID,
       NOTE
  FROM POOLDATA.BUSINESS
 WHERE TRIM(CAST(ID AS VARCHAR(64))) = :1

-- name: business_check_table
--
-- Memastikan master lini bisnis ada dan dapat dibaca akun aplikasi.
SELECT ID
  FROM POOLDATA.BUSINESS
 WHERE 1 = 0
