-- Lampiran klaim — POOLDATA.DATA_ATTACHFILE, tombol Unggah Dokumen pada tab checklist.

-- name: lampiran_sisip
--
-- Pengganti `SaveAttachmentToDB_Sql` → `SET_ATTACHMENT_64BIT`, ditulis langsung (`D-02`).
--
-- # DATAID dari POOLDATA.ATTACHFILE_SEQ, sama dengan Pega
--
-- Procedure lama menyusun DATAID = tahun dua digit || LPAD(ATTACHFILE_SEQ.NEXTVAL, 10, '0')
-- — terverifikasi pada baris 2026-09-28 (`260001895401`). Tabel ini MASIH ditulis Pega setiap
-- hari, sehingga nomornya WAJIB datang dari sequence yang sama: MAX+1 akan membuat
-- penyisipan Pega berikutnya menabrak kunci utama DATAID. Ini pengecualian dialek kedua
-- setelah generator nomor klaim (`ADR-0005`); padanan PostgreSQL-nya `nextval(...)`.
--
-- Baris C_COUNTER_ATTACHMENT yang disisipkan procedure hanya perantara pembentuk DATAID
-- dan tidak dibaca di mana pun, sehingga tidak dibawa.
--
-- :1 tahun dua digit (WIB), :2 IDPEGA, :3 INPUTOPERATOR, :4 ATTACHNAME, :5 ATTACHNOTE,
-- :6 ATTACHMIMETYPE (ekstensi), :7 IMAGEID, :8 CATEGORY, :9 SUB_CATEGORY, :10 INPUTDATE.
INSERT INTO POOLDATA.DATA_ATTACHFILE
       (DATAID, IDPEGA, INPUTOPERATOR, ATTACHNAME, ATTACHNOTE, ATTACHMIMETYPE, IMAGEID,
        CATEGORY, SUB_CATEGORY, INPUTDATE)
VALUES (:1 || LPAD(TO_CHAR(POOLDATA.ATTACHFILE_SEQ.NEXTVAL), 10, '0'), :2, :3, :4, :5, :6, :7,
        :8, :9, :10)
