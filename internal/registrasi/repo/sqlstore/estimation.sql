-- Item objek dan estimasinya, tahap Input Estimasi.
--
-- TC_PNC_OBJECTITEM dirancang Work Owner (Database/CREATE_TABLE_2.sql) dan punya penanda
-- hapus. T_CLAIM_ESTIMASI adalah tabel warisan tanpa penanda hapus dan tanpa kunci primer;
-- barisnya karena itu hanya diperbarui atau disisipkan, tidak pernah dihapus (D-66).
--
-- Kuncinya mengikuti procedure konversi Pega: OBJECTID objek, OBJECTCOVERAGEID urutan
-- coverage di dalam objek, OBJECTITEMID urutan item, ESTIMASIID urutan estimasi.
--
-- Satuan: nilai uang dalam rupiah di kolom, sen di domain; kurs dikali 10.000 di domain.
--
-- PRINTFACECLAIM 1 berarti estimasi sudah dibuatkan Claim Face Sheet (dikunci); 0 atau NULL
-- berarti belum, sama dengan nilai "0" pada Pega. CFSDATE tanggal CFS-nya.

-- name: item_perbarui
UPDATE POOLDATA.TC_PNC_OBJECTITEM
   SET OBJECTITEMNAME = :1, DESKRIPSIOBJECT = :2, SUMESTIMATION = :3 / 100,
       DIUBAH_OLEH = :4, DIUBAH_PADA = :5, PROPERTYITEMGROUP = :6,
       DIHAPUS_OLEH = NULL, DIHAPUS_PADA = NULL
 WHERE CLAIMID = :7 AND OBJECTID = :8 AND OBJECTCOVERAGEID = :9 AND OBJECTITEMID = :10

-- name: item_sisip
INSERT INTO POOLDATA.TC_PNC_OBJECTITEM
       (OBJECTITEMNAME, DESKRIPSIOBJECT, SUMESTIMATION, DIUBAH_OLEH, DIUBAH_PADA,
        CLAIMID, OBJECTID, OBJECTCOVERAGEID, OBJECTITEMID, DIBUAT_OLEH, DIBUAT_PADA,
        PROPERTYITEMGROUP)
VALUES (:1, :2, :3 / 100, :4, :5, :6, :7, :8, :9, :10, :11, :12)

-- name: item_tandai_sisa
UPDATE POOLDATA.TC_PNC_OBJECTITEM
   SET DIHAPUS_OLEH = :1, DIHAPUS_PADA = :2
 WHERE CLAIMID = :3 AND OBJECTID = :4 AND OBJECTCOVERAGEID = :5
   AND TO_NUMBER(OBJECTITEMID) > :6 AND DIHAPUS_PADA IS NULL

-- name: item_daftar
SELECT OBJECTID, OBJECTCOVERAGEID, OBJECTITEMID, OBJECTITEMNAME, DESKRIPSIOBJECT,
       PROPERTYITEMGROUP
  FROM POOLDATA.TC_PNC_OBJECTITEM
 WHERE CLAIMID = :1 AND DIHAPUS_PADA IS NULL
 ORDER BY OBJECTID, TO_NUMBER(OBJECTCOVERAGEID), TO_NUMBER(OBJECTITEMID)

-- name: estimasi_perbarui
UPDATE POOLDATA.T_CLAIM_ESTIMASI
   SET ESTIMATIONTYPE = :1, KURSID = :2, ESTIMATIONVALUE = :3 / 100, KURSVALUE = :4 / 10000,
       CONVERTVALUE = :5 / 100, ESTIMATIONDATE = :6, PRINTFACECLAIM = :7, CFSDATE = :8
 WHERE CLAIMID = :9 AND OBJECTID = :10 AND OBJECTCOVERAGEID = :11 AND OBJECTITEMID = :12
   AND ESTIMASIID = :13

-- name: estimasi_sisip
INSERT INTO POOLDATA.T_CLAIM_ESTIMASI
       (ESTIMATIONTYPE, KURSID, ESTIMATIONVALUE, KURSVALUE, CONVERTVALUE, ESTIMATIONDATE,
        CLAIMID, OBJECTID, OBJECTCOVERAGEID, OBJECTITEMID, ESTIMASIID, DIBUAT_OLEH, DIBUAT_PADA,
        PRINTFACECLAIM, CFSDATE)
VALUES (:1, :2, :3 / 100, :4 / 10000, :5 / 100, :6, :7, :8, :9, :10, :11, :12, :13, :14, :15)

-- name: estimasi_daftar
SELECT OBJECTID, OBJECTCOVERAGEID, OBJECTITEMID, ESTIMASIID, ESTIMATIONTYPE, KURSID,
       ROUND(ESTIMATIONVALUE * 100), ROUND(KURSVALUE * 10000), ROUND(CONVERTVALUE * 100),
       ESTIMATIONDATE, PRINTFACECLAIM, CFSDATE
  FROM POOLDATA.T_CLAIM_ESTIMASI
 WHERE CLAIMID = :1
 ORDER BY OBJECTID, TO_NUMBER(OBJECTCOVERAGEID), TO_NUMBER(OBJECTITEMID), TO_NUMBER(ESTIMASIID)

-- name: mata_uang_daftar
--
-- Pilihan Mata Uang pada baris estimasi, dari master POOLDATA.CURRENCY.
SELECT ID, CURRENCY
  FROM POOLDATA.CURRENCY
 ORDER BY CURRENCY

-- name: status_nama
--
-- Nama Status Klaim dari master V_STS_CLAIM.
SELECT LSC_NOTE FROM POOLDATA.V_STS_CLAIM WHERE LSC_ID = :1

-- name: item_pilihan_properti
--
-- Pilihan Objek pada item estimasi lini Fire: item properti objek polis ini, dari tabel
-- relasional POOLDATA.T_PROPERTYITEMLIST. Kolom JSON T_PROPERTYLIST.PROPERTYITEMLIST tidak
-- dibaca lagi (Work Owner, 2026-10-09).
--
-- ID objek klaim Fire adalah OBJECTNO, sedangkan T_PROPERTYITEMLIST dikunci INDEXOBJECT,
-- sehingga objeknya dicari lewat T_PROPERTYLIST. Objek tanpa baris di tabel ini berarti
-- polis tidak punya daftar item (terukur 2026-10-09: 318 dari 20.305 objek Fire terisi).
SELECT i.ITEMTYPE, i.PROPERTYITEMGROUP, i.TSIOBJECTITEM
  FROM POOLDATA.T_PROPERTYITEMLIST i
  JOIN POOLDATA.T_PROPERTYLIST p
    ON p.NOPOLIS = i.NOPOLIS AND p.PRODKE = i.PRODKE AND p.INDEXOBJECT = i.INDEXOBJECT
 WHERE i.NOPOLIS = :1 AND i.PRODKE = :2 AND p.OBJECTNO = :3
   AND (p.FLAGDELETE IS NULL OR p.FLAGDELETE <> '1')
   AND (i.FLAGDELETE IS NULL OR i.FLAGDELETE <> '1')
 ORDER BY i.PROPERTYITEMCODE, i.ITEMTYPE

-- name: item_pilihan_travel
--
-- Pilihan Objek pada item estimasi lini Travel: manfaat plan dari master
-- POOLDATA.COVERAGETRAVEL — padanan `SearchCoverageTravel_RD` (Param.plan = kode coverage
-- Travel). PLANID adalah PLANTRAVEL.ID (terukur 2026-10-09: 4.057 dari 4.057 baris cocok),
-- dan ID-nya menjadi ObjectItemID item. LIMIT (CoverageDetailLimit) tidak dibaca: isinya teks
-- berformat campuran seperti "1500,000", dan TSIperCoverage tidak disimpan aplikasi ini.
SELECT c.ID, c.INDCOVERAGENAME
  FROM POOLDATA.COVERAGETRAVEL c
 WHERE c.PLANID = :1
 ORDER BY c.ID
