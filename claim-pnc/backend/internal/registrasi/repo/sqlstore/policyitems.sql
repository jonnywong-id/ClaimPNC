-- Objek polis beserta dokumen coverage-nya, satu kueri per tabel sumber.
--
-- Diturunkan dari RDB List/GetListObjectTravelPA, GetListObjectFire,
-- GetListObjectMarine, GetListObjectAneka, dan kueri coverage pasangannya.
-- Keempatnya membaca tabel milik sistem polis dan tidak pernah menulisnya.
--
-- Nilai polis dan versinya lewat parameter binding, bukan perangkaian teks seperti
-- pola ASIS warisan. Dua potongan ASIS pada kueri lama (No_Klaim dan CUSTOMER) tidak
-- dibawa: keduanya kosong pada jalur registrasi.
--
-- Nama kolom di sini adalah nama aslinya. Kueri lama mengaliaskannya ke properti Pega
-- yang tidak berhubungan (objectname AS BRANCHCODE), dan alias itu tidak dibawa (D-19).
--
-- Dokumen JSON disimpan sebagai BLOB. TO_CLOB mengubahnya menjadi teks, sama seperti
-- kueri lama.

-- name: polis_objek_person
SELECT INDEXOBJECT, UPPER(PYFULLNAME), NULL, TO_CLOB(COVERAGEDATA)
  FROM POOLDATA.T_PERSONLIST
 WHERE NOPOLIS = :1 AND PRODKE = :2
 ORDER BY INDEXOBJECT

-- name: polis_objek_property
--
-- Satu objek per INDEXOBJECT, seperti GROUP BY pada GetListObjectFire. Nomor objeknya
-- OBJECTNO, bukan INDEXOBJECT.
SELECT INDEXOBJECT, MAX(OBJECTNO), MAX(OBJECTNAME), MAX(ASMADDRESS)
  FROM POOLDATA.T_PROPERTYLIST
 WHERE NOPOLIS = :1 AND PRODKE = :2
   AND (FLAGDELETE IS NULL OR FLAGDELETE = 0)
 GROUP BY INDEXOBJECT
 ORDER BY INDEXOBJECT

-- name: jenis_treaty_nama
--
-- Nama treaty per ID — master yang sama dengan dropdown Nama Treaty Pega.
SELECT CAST(ID AS VARCHAR(20)), NOTE
  FROM POOLDATA.REINSURANCETYPE

-- name: polis_coverage_property
SELECT TO_CLOB(COVERAGELIST)
  FROM POOLDATA.T_PROPERTYLIST
 WHERE NOPOLIS = :1 AND PRODKE = :2 AND INDEXOBJECT = :3

-- name: polis_objek_cargo
SELECT INDEXOBJECT, GOODSNAME, CONVEYANCENOTE, TO_CLOB(COVERAGEDATA)
  FROM POOLDATA.T_CARGOLIST
 WHERE NOPOLIS = :1 AND PRODKE = :2
 ORDER BY INDEXOBJECT

-- name: polis_objek_aneka
SELECT INDEXOBJECT, OBJECTNAME, ASMADDRESS, TO_CLOB(COVERAGELIST)
  FROM POOLDATA.T_ANEKALIST
 WHERE NOPOLIS = :1 AND PRODKE = :2
 ORDER BY INDEXOBJECT
