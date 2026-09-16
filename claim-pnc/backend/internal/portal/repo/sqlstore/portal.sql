-- Kueri tabel POOLDATA.M_PORTAL_PNC.
--
-- Tabel milik sistem lama; aplikasi ini hanya MEMBACA (ADR-0004, penulis tunggal per
-- tabel). Kolom disebut namanya; SELECT * dilarang.

-- name: portal_daftar
SELECT PORTAL_ID,
       PORTAL_NAME,
       PORTAL_ALIAS
  FROM POOLDATA.M_PORTAL_PNC
 ORDER BY PORTAL_ID
