-- Kueri tabel CPNC_SESI_AKTIF.
--
-- Pencabutan adalah UPDATE, bukan DELETE (ADR-0012): baris sesi tidak pernah dihapus
-- fisik supaya jejaknya tetap dapat ditelusuri.

-- name: session_insert
INSERT INTO CPNC_SESI_AKTIF (ID, SIDIK_TOKEN, IDENTITAS, DITERBITKAN_PADA, BERLAKU_SAMPAI)
VALUES (:1, :2, :3, :4, :5)

-- name: session_get_by_fingerprint
SELECT ID,
       SIDIK_TOKEN,
       IDENTITAS,
       DITERBITKAN_PADA,
       BERLAKU_SAMPAI,
       DICABUT_PADA
  FROM CPNC_SESI_AKTIF
 WHERE SIDIK_TOKEN = :1

-- name: session_revoke
UPDATE CPNC_SESI_AKTIF
   SET DICABUT_PADA = :1
 WHERE ID = :2
   AND DICABUT_PADA IS NULL

-- name: session_extend
UPDATE CPNC_SESI_AKTIF
   SET BERLAKU_SAMPAI = :1
 WHERE ID = :2
   AND DICABUT_PADA IS NULL

-- name: session_check_table
SELECT ID
  FROM CPNC_SESI_AKTIF
 WHERE 1 = 0
