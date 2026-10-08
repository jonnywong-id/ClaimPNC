-- Pencarian kode diagnosa modal "Transfer Claim ke Komite" (PA) — RDB List/GetKodeDiagnosaKlaimPa.
--
-- Master diagnosa (ICD) tinggal di database ASMD dan dibaca lewat DB Link, sama seperti kueri
-- Pega-nya. Pega menyisipkan pola LIKE apa adanya ({ASIS:TempKodeDiagnosa.Remark}); di sini
-- polanya dikirim sebagai parameter, dan karakter wildcard dari pengguna di-escape.
--
-- Pega tidak membatasi jumlah baris; di sini 100 baris pertama menurut kode (batas API, D-10).

-- name: diagnosa_cari
SELECT DIAGNOSIS_KODE, DIAGNOSIS_DESC
  FROM sm.m_diagnosis@asmd.sinarmas.co.id
 WHERE UPPER(DIAGNOSIS_KODE) = UPPER(:1)
    OR UPPER(DIAGNOSIS_DESC) LIKE :2 ESCAPE '\'
 ORDER BY DIAGNOSIS_KODE
 FETCH FIRST 100 ROWS ONLY
