-- Objek polis beserta coverage dan spreading-nya.
--
-- Diturunkan dari RDB List/GetListObjectTravelPA, GetListObjectFire,
-- GetListObjectMarine, GetListObjectAneka, dan kueri coverage pasangannya.
-- Seluruhnya membaca tabel milik sistem polis dan tidak pernah menulisnya.
--
-- Nilai polis dan versinya lewat parameter binding, bukan perangkaian teks seperti
-- pola ASIS warisan. Dua potongan ASIS pada kueri lama (No_Klaim dan CUSTOMER) tidak
-- dibawa: keduanya kosong pada jalur registrasi.
--
-- Nama kolom di sini adalah nama aslinya. Kueri lama mengaliaskannya ke properti Pega
-- yang tidak berhubungan (objectname AS BRANCHCODE), dan alias itu tidak dibawa (D-19).
--
-- ============================================================================
-- COVERAGE DAN SPREADING TIDAK LAGI DIBACA DARI BLOB -- Work Owner, 2026-10-06
-- ============================================================================
--
-- Sebelum ini, coverage dibaca dari kolom BLOB berisi dokumen JSON pada tabel objeknya
-- sendiri: T_PERSONLIST.COVERAGEDATA, T_PROPERTYLIST.COVERAGELIST,
-- T_CARGOLIST.COVERAGEDATA, dan T_ANEKALIST.COVERAGELIST. Spreading ada di dalam setiap
-- coverage pada dokumen yang sama.
--
-- Keempat kolom itu DIGANTI tabel relasional:
--
--   sumber objek            coverage                      spreading
--   --------------------------------------------------------------------------
--   T_PERSONLIST            T_COVERAGELIST_PERSON         T_SPREADINGLIST
--   T_PROPERTYLIST          T_COVERAGELIST_FIRE           T_SPREADINGLIST
--   T_CARGOLIST             T_COVERAGELIST_CARGO          T_SPREADINGLIST
--   T_ANEKALIST             T_COVERAGELIST_ANEKA          T_SPREADINGLIST
--
-- Tabel objeknya TETAP dibaca -- nama dan lokasi objek hanya ada di sana. Yang berhenti
-- dibaca adalah kolom BLOB-nya saja.
--
-- Keempat kueri coverage mengembalikan DELAPAN kolom dengan urutan yang sama, dan kolom
-- yang tidak dimiliki sebuah tabel diisi NULL. Bentuk yang seragam itu disengaja: ia
-- membuat satu jalur pembacaan di Go melayani keempat lini, sehingga lini yang jarang
-- dipakai tidak menempuh kode yang jarang dijalankan.
--
-- TSI dibiarkan apa adanya di sini dan diubah menjadi SEN di Go. Kolomnya bertipe
-- campuran antartabel -- SUMTSI pada T_COVERAGELIST_FIRE bertipe VARCHAR2 sementara pada
-- T_COVERAGELIST_CARGO bertipe NUMBER -- sehingga seluruhnya dibaca sebagai teks lalu
-- diurai dengan satu fungsi yang sama.

-- name: polis_objek_person
--
-- Kolom pertama adalah KUNCI penggabungan ke tabel coverage; kolom kedua adalah ID objek
-- yang dipakai klaim. Keduanya sama untuk lini ini, dan sengaja tetap ditulis dua kali
-- supaya keempat kueri objek berbentuk sama.
--
-- Kolom kedua WAJIB beralias. Tanpa alias, Oracle menolak ORDER BY INDEXOBJECT dengan
-- ORA-00960 (nama kolom ambigu di daftar SELECT) -- terjadi pada lini Person, Cargo,
-- dan Aneka, terukur 2026-10-07 pada klaim Aneka.
SELECT INDEXOBJECT, INDEXOBJECT AS ID_OBJEK, UPPER(PYFULLNAME), NULL
  FROM POOLDATA.T_PERSONLIST
 WHERE NOPOLIS = :1 AND PRODKE = :2
 ORDER BY INDEXOBJECT

-- name: polis_objek_property
--
-- Satu objek per INDEXOBJECT, seperti GROUP BY pada GetListObjectFire. Nomor objeknya
-- OBJECTNO, bukan INDEXOBJECT -- inilah satu-satunya lini yang kunci dan ID-nya berbeda.
SELECT INDEXOBJECT, MAX(OBJECTNO), MAX(OBJECTNAME), MAX(ASMADDRESS)
  FROM POOLDATA.T_PROPERTYLIST
 WHERE NOPOLIS = :1 AND PRODKE = :2
   AND (FLAGDELETE IS NULL OR FLAGDELETE = 0)
 GROUP BY INDEXOBJECT
 ORDER BY INDEXOBJECT

-- name: polis_objek_cargo
SELECT INDEXOBJECT, INDEXOBJECT AS ID_OBJEK, GOODSNAME, CONVEYANCENOTE
  FROM POOLDATA.T_CARGOLIST
 WHERE NOPOLIS = :1 AND PRODKE = :2
 ORDER BY INDEXOBJECT

-- name: polis_objek_aneka
SELECT INDEXOBJECT, INDEXOBJECT AS ID_OBJEK, OBJECTNAME, ASMADDRESS
  FROM POOLDATA.T_ANEKALIST
 WHERE NOPOLIS = :1 AND PRODKE = :2
 ORDER BY INDEXOBJECT

-- name: jenis_treaty_nama
--
-- Nama treaty per ID -- master yang sama dengan dropdown Nama Treaty Pega.
SELECT CAST(ID AS VARCHAR(20)), NOTE
  FROM POOLDATA.REINSURANCETYPE

-- name: polis_coverage_person
--
-- T_COVERAGELIST_PERSON tidak punya kolom COVERAGE; kode coverage-nya COVERAGEID.
-- Ia juga tidak punya SUMTSI maupun TSISUBLIMIT, sehingga TSI coverage selalu jatuh ke
-- kolom TSI (lihat coverageTSI di internal/registrasi/policyitems.go).
SELECT INDEXOBJECT, INDEXCOVERAGE, COVERAGEID, COVERAGENOTE,
       TSI, NULL, NULL, FLAGDELETE
  FROM POOLDATA.T_COVERAGELIST_PERSON
 WHERE NOPOLIS = :1 AND PRODKE = :2
 ORDER BY INDEXOBJECT, INDEXCOVERAGE

-- name: polis_coverage_property
--
-- Lini Fire -- satu-satunya yang punya TSISUBLIMIT, dan TSISUBLIMIT itulah yang menang
-- atas TSI bila terisi.
SELECT INDEXOBJECT, INDEXCOVERAGE, COVERAGE, COVERAGENOTE,
       TSI, SUMTSI, TSISUBLIMIT, FLAGDELETE
  FROM POOLDATA.T_COVERAGELIST_FIRE
 WHERE NOPOLIS = :1 AND PRODKE = :2
 ORDER BY INDEXOBJECT, INDEXCOVERAGE

-- name: polis_coverage_cargo
SELECT INDEXOBJECT, INDEXCOVERAGE, COVERAGE, COVERAGENOTE,
       TSI, SUMTSI, NULL, FLAGDELETE
  FROM POOLDATA.T_COVERAGELIST_CARGO
 WHERE NOPOLIS = :1 AND PRODKE = :2
 ORDER BY INDEXOBJECT, INDEXCOVERAGE

-- name: polis_coverage_aneka
--
-- T_COVERAGELIST_ANEKA TIDAK punya kolom INDEXOBJECT maupun SUMTSI (terukur dari
-- ALL_TAB_COLUMNS 2026-10-07; kueri lama gagal ORA-00904). Penghubungnya ke objek adalah
-- INDEXTANEKALIST, sehingga INDEXOBJECT diambil dari T_ANEKALIST lewat kolom itu -- kunci
-- objek tetap INDEXOBJECT supaya cocok dengan T_SPREADINGLIST.INDEXOBJECT.
SELECT a.INDEXOBJECT, c.INDEXCOVERAGE, c.COVERAGE, c.COVERAGENOTE,
       c.TSI, NULL, NULL, c.FLAGDELETE
  FROM POOLDATA.T_COVERAGELIST_ANEKA c
       INNER JOIN POOLDATA.T_ANEKALIST a
               ON a.NOPOLIS = c.NOPOLIS
              AND a.PRODKE = c.PRODKE
              AND a.INDEXTANEKALIST = c.INDEXTANEKALIST
 WHERE c.NOPOLIS = :1 AND c.PRODKE = :2
 ORDER BY a.INDEXOBJECT, c.INDEXCOVERAGE

-- name: polis_spreading
--
-- Seluruh spreading satu polis dalam SATU kueri, bukan satu kueri per coverage. Polis
-- Fire berobjek banyak dapat punya puluhan coverage, dan satu kueri per coverage membuat
-- pembukaan satu layar registrasi menjadi puluhan perjalanan ke basis data.
--
-- Baris digabungkan ke coverage-nya lewat INDEXOBJECT + INDEXCOVERAGE. Kolom COVERAGE
-- ikut dibaca sebagai jalur cadangan: bila sebuah coverage tidak mendapat satu baris pun
-- lewat INDEXCOVERAGE, pencocokan diulang lewat kode coverage-nya. Tanpa cadangan itu,
-- satu polis yang INDEXCOVERAGE-nya tidak terisi akan kehilangan SELURUH spreading-nya,
-- dan gerbang validasi menolaknya dengan pesan "total spreading bukan 100%" yang tidak
-- menyebut sebab sesungguhnya.
--
-- INDEXSPREADING menentukan urutan di dalam satu coverage, sama seperti urutan baris
-- SpreadingList pada dokumen lama.
SELECT INDEXOBJECT, INDEXCOVERAGE, COVERAGE,
       TREATYTYPE, TREATYNAME, SHAREPERCENTAGE, FLAGDELETE
  FROM POOLDATA.T_SPREADINGLIST
 WHERE NOPOLIS = :1 AND PRODKE = :2
 ORDER BY INDEXOBJECT, INDEXCOVERAGE, INDEXSPREADING
