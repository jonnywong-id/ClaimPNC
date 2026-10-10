-- Preliminary Loss Advice (PLA) koasuransi.
--
-- DITULIS modul ini, hanya untuk klaim yang dibuat aplikasi ini (keputusan Work Owner
-- 2026-09-27): POOLDATA.T_PLALIST (pengganti cabang PLA INSERT_PLADLA.prc) dan
-- POOLDATA.PLA (log penomoran, pengganti cabang PLA PLA_DLA.prc). Selebihnya DIBACA.

-- name: pla_koasuransi
--
-- Koasuransi polis beserta CoinsID dan penanda hapus.
--
-- Sumbernya POOLDATA.T_COINSLIST menurut NOPOLIS dan PRODKE snapshot klaim (Work Owner,
-- 2026-10-01: data polis tidak dibaca dari JSON_POLIS.DATA_JSONBLOB bila ada tabelnya).
-- Kolomnya padanan CoinsList dokumen polis: COINSID, COINSNAME, LEADER ('true'/'false'),
-- PERCENT_SHARE, FLAGDELETE. PRODKE yang tidak punya baris berarti polis tanpa koasuransi.
SELECT c.COINSID, c.COINSNAME, c.LEADER, c.PERCENT_SHARE, c.FLAGDELETE
  FROM POOLDATA.T_COINSLIST c
 WHERE c.NOPOLIS = :1
   AND c.PRODKE = :2
 ORDER BY c.COINSID

-- name: pla_penerima
--
-- INSERT_PLADLA.prc: cocokkan kode DAN nama lebih dulu, lalu kode saja.
SELECT r.LOGIN, r.COUNTRY, r.EMAIL
  FROM POOLDATA.T_REINSURER r
 WHERE r.REINSURERID = :1
 ORDER BY CASE WHEN r.REINSURERNAME = :2 THEN 0 ELSE 1 END
 FETCH FIRST 1 ROWS ONLY

-- name: pla_sebelumnya
SELECT d.NOPLA, d.TGLPLA
  FROM POOLDATA.T_PLALIST d
 WHERE d.CLAIMID = :1 AND d.REINSCODE = :2
 ORDER BY d.TGLPLA DESC
 FETCH FIRST 1 ROWS ONLY

-- name: pla_terbit
--
-- PLA yang diterbitkan layar Print PLA: koasuransi (COINS), fakultatif keluar (FACOUT), dan
-- BPPDAN / EQ POOL (GeneratePLAList).
SELECT d.NOPLA, d.PLAREINSURER, d.REINSCODE, d.TGLPLA, d.NOTES, d.CURRENCYPOLIS, d.JSON_PLA, d.EMAILPLA,
       d.TIPEPLA, COALESCE(d.ISKIRIM, '0')
  FROM POOLDATA.T_PLALIST d
 WHERE d.CLAIMID = :1 AND d.OBJECTID = :2 AND d.OBJECTCOVERAGEID = :3 AND d.REVISI = :4
   AND d.TIPEPLA IN (:5, :6, :7, :8)
 ORDER BY d.NOPLA

-- name: pla_fac_offer
--
-- Fac Offer polis untuk PLA FAC OUT: JSONDATA (FacOfferList, diurai di Go seperti DLA) dan
-- kolom datar sebagai cadangan bila JSONDATA kosong — 33 dari 409 baris di TEST.
SELECT f.REINSURER_ID, f.REINSURER_NAME, f.PCT_SHAREREAS, f.JSONDATA
  FROM POOLDATA.T_FACOFFER f
 WHERE f.POLICYNO = :1
   AND f.PRODKE = :2
 ORDER BY f.REINSURER_ID

-- name: pla_spreading_tsi
--
-- TSISPREADED spreading polis satu objek dan coverage — cadangan PLA FAC OUT untuk Fac Offer
-- tanpa JSONDATA. Objek dicocokkan lewat INDEXOBJECT; untuk Fire (Group Panel 006) ID objek
-- klaim adalah OBJECTNO, sehingga INDEXOBJECT-nya dicari lebih dulu di T_PROPERTYLIST.
SELECT s.TREATYTYPE, s.TSISPREADED
  FROM POOLDATA.T_SPREADINGLIST s
 WHERE s.NOPOLIS = :1
   AND s.PRODKE = :2
   AND s.COVERAGE = :3
   AND (s.FLAGDELETE IS NULL OR s.FLAGDELETE <> '1')
   AND ((:4 <> '006' AND s.INDEXOBJECT = :5)
        OR (:6 = '006' AND s.INDEXOBJECT IN (SELECT p.INDEXOBJECT
                                               FROM POOLDATA.T_PROPERTYLIST p
                                              WHERE p.NOPOLIS = :7
                                                AND p.PRODKE = :8
                                                AND p.OBJECTNO = :9)))

-- name: pla_site
SELECT s.ID
  FROM POOLDATA.M_SITE_DATABASE s
 WHERE s.CURRENT_SITE = '1'
 FETCH FIRST 1 ROWS ONLY

-- name: pla_urut
--
-- PENGECUALIAN DIALEK yang disadari: PLA_SEQ dipakai bersama Pega yang masih menerbitkan
-- PLA untuk klaim PNC-xxxx, sehingga nomor WAJIB diambil dari sequence yang sama agar tidak
-- bertabrakan. Padanan PostgreSQL-nya nextval('pooldata.pla_seq').
SELECT POOLDATA.PLA_SEQ.NEXTVAL FROM DUAL

-- name: pla_nomor_sisip
INSERT INTO POOLDATA.PLA (KEY, ID_PLA, KODE, ID_SITE, TAHUN, COUNT)
VALUES (:1, NULL, :2, :3, :4, :5)

-- name: pla_sisip
--
-- Kolom angka diisi 0 seperti baris Pega; nilainya disimpan di JSON_PLA.EstimasiList.
INSERT INTO POOLDATA.T_PLALIST
       (CLAIMID, OBJECTID, OBJECTCOVERAGEID, NOPLA, NILAIPLA, PLAREINSURER, REVISI, TIPEPLA,
        TGLPLA, NOTES, REINSCODE, CURRENCYPOLIS, PERCENTPLA, ESTIMASI, ESTIMASISHARE,
        EMAILPLA, LOGIN, COUNTRY, JSON_PLA)
VALUES (:1, :2, :3, :4, '0', :5, :6, :7, :8, :9, :10, :11, '0', '0', '0', :12, :13, :14, :15)

-- name: pla_ttd
SELECT m.NAME, m.JSONDATA
  FROM POOLDATA.MTTD m
 WHERE m.ID = :1

-- name: pla_catatan
--
-- INSERT_PLADLA.prc cabang PLA, baris sudah ada dan catatan terisi: NOTES diganti, ISPLA = 1.
UPDATE POOLDATA.T_PLALIST
   SET NOTES = :1, ISPLA = '1'
 WHERE CLAIMID = :2 AND NOPLA = :3 AND REVISI = :4

-- name: pla_email
--
-- Isian Email layar PrintPLA_dtl (`.pyEmailAddress`, pxTextArea Editable).
UPDATE POOLDATA.T_PLALIST
   SET EMAILPLA = :1
 WHERE CLAIMID = :2 AND NOPLA = :3 AND REVISI = :4

-- name: lod_email_tertanggung
--
-- Isian Email LOD (`SetDataEmailTertanggung` ← `ClaimData.Email`), yang diisi
-- `GetDataPengkinianDataTertanggung` dari `GetDataPengkinianData_SQLF`: OLDEMAIL baris
-- pengkinian data klaim itu. Pega mengambil baris pertama tanpa urutan.
SELECT u.OLDEMAIL
  FROM POOLDATA.UPDATE_PENGKINIANDATA u
 WHERE u.PYID = :1
 FETCH FIRST 1 ROWS ONLY

-- name: lod_email_pic
--
-- `ClaimData.UserTeknisEmail` — `BrowseEmailUserTeknis` (InputRegister_act langkah 67-68).
SELECT t.EMAIL
  FROM POOLDATA.MST_USER_TEKNIK t
 WHERE t.OPERATOR_ID = :1
 FETCH FIRST 1 ROWS ONLY

-- name: pa_ttd
--
-- BrowseSignature-SQL: tanda tangan menurut SIGNATURE_ID. JSONDATA adalah dokumen JSON yang
-- di Pega dimuat apa adanya ke SignatureFiles (adoptJSONObject); gambar di kunci TTD
-- (tersimpan dengan spasi di belakang) dan nama di SIGNATURE_NAME.
SELECT JSONDATA
  FROM POOLDATA.M_SIGNATURE1
 WHERE SIGNATURE_ID = :1
