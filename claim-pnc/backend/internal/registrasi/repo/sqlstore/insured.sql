-- name: polis_cif
--
-- Data tertanggung dari CIF polis (Policy.CIFData dokumen POLICYDATA) untuk blok Alamat dan
-- Telephone dan Email pada tab Register (Section InputAddress_PNC_Klaim). Dokumen dengan PRODKE
-- klaim didahulukan, lalu yang terbaru. DATA_JSONBLOB tidak dibaca (Work Owner, 2026-10-01).
--
-- Bind: :1 nomor polis, :2 PRODKE klaim
SELECT JSON_QUERY(p.POLICYDATA, '$.CIFData' RETURNING CLOB)
  FROM POOLDATA.JSON_POLIS p
 WHERE p.NOPOLIS = :1
   AND p.POLICYDATA IS NOT NULL
 ORDER BY CASE WHEN TRIM(p.PRODKE) = :2 THEN 0 ELSE 1 END, p.TGL_INPUT DESC
 FETCH FIRST 1 ROWS ONLY
