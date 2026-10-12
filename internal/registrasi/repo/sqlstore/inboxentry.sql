-- Baris daftar kerja My Inbox — POOLDATA.T_CLAIMLIST_ADMIN (lihat registrasi/inbox_entry.go).
--
-- Hanya baris klaim PNCN yang ditulis di sini; kuncinya nomor PNCN, sehingga baris Pega
-- (berkunci ASM-FW-GCNMFW-WORK PNC-...) tidak pernah tersentuh.
--
-- UPDATE lalu INSERT, bukan MERGE — keputusan 2.9: MERGE bukan sintaks yang sama antara
-- Oracle dan PostgreSQL.
--
-- BUSINESSGROUPID diturunkan dari POOLDATA.BUSINESS lewat kode bisnis. Hitungan terhadap
-- 1.016 baris yang ada: seluruhnya sama dengan BUSINESS.BUSINESSGROUPID untuk
-- BUSINESSCODE_1-nya, jadi itulah aturan pengisinya. Export My Inbox memakai kolom ini
-- sebagai penentu cakupan lini Bonding.
--
-- Kolom yang tidak disebut di sini (OS_CATEGORY, STATUSPROGRESS1, STATUSPROGRESS2, AGING,
-- DATEFORAGING_1, PXCREATEOPNAME, dan kolom tahap 1-2 migrasi 0005) sengaja tidak diisi —
-- aturan pengisinya tidak diketahui. Lihat inbox_entry.go.
--
-- Penanda bernomor sesuai URUTAN KEMUNCULAN: go-ora mengikat menurut urutan itu.

-- name: daftar_kerja_perbarui
UPDATE POOLDATA.T_CLAIMLIST_ADMIN
   SET PYID                 = :1,
       PXOBJCLASS           = :2,
       PXFLOWNAME           = :3,
       PYSTATUSWORK         = :4,
       PXASSIGNEDOPERATORID = :5,
       PXTASKLABEL          = :6,
       PNCCASEID            = :7,
       BUSINESSCODE_1       = :8,
       BUSINESSGROUPID      = (SELECT b.BUSINESSGROUPID
                                 FROM POOLDATA.BUSINESS b
                                WHERE b.ID = :9),
       KODECABANG_1         = :10,
       BRANCHNAME           = :11,
       PXCREATEOPERATOR     = :12,
       PXCREATEDATETIME     = :13,
       REGISTERDATE_1       = :14,
       USERTEKNIS_1         = :15,
       GROUPPANEL_1         = :16,
       SOBNAME              = :17,
       BUSINESSNAME         = :18,
       QQNAME               = :19,
       POLICYNO             = :20,
       DATEOFLOSS_1         = :21,
       REPORTDATE_1         = :22,
       STARTDATE            = :23,
       ENDDATE              = :24,
       PRODKE               = :25,
       STATUSCLAIM_1        = :26,
       STS_AKTIF            = :27
 WHERE PZINSKEY = :28

-- name: daftar_kerja_sisip
-- STATUSLOCK_1 diisi '1' langsung saat baris dibuat -- keputusan Work Owner 2026-10-07.
-- Inbox Admin menyaring kolom ini, dan tidak ada proses lain yang mengisinya sesudahnya.
INSERT INTO POOLDATA.T_CLAIMLIST_ADMIN (
       PZINSKEY, PYID, PXOBJCLASS, PXFLOWNAME, PYSTATUSWORK,
       PXASSIGNEDOPERATORID, PXTASKLABEL, PNCCASEID, BUSINESSCODE_1, BUSINESSGROUPID,
       KODECABANG_1, BRANCHNAME, PXCREATEOPERATOR, PXCREATEDATETIME, REGISTERDATE_1,
       USERTEKNIS_1, GROUPPANEL_1, SOBNAME, BUSINESSNAME, QQNAME,
       POLICYNO, DATEOFLOSS_1, REPORTDATE_1, STARTDATE, ENDDATE,
       PRODKE, STATUSCLAIM_1, STS_AKTIF, STATUSLOCK_1)
VALUES (:1, :2, :3, :4, :5,
       :6, :7, :8, :9,
       (SELECT b.BUSINESSGROUPID FROM POOLDATA.BUSINESS b WHERE b.ID = :10),
       :11, :12, :13, :14, :15,
       :16, :17, :18, :19, :20,
       :21, :22, :23, :24, :25,
       :26, :27, :28, '1')
