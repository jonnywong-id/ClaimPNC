-- 0009 — CPNC_TUGAS dan CPNC_JEJAK_AUDIT (Oracle 19c)
--
-- ============================================================================
-- DUA DARI EMPAT, BUKAN EMPAT
-- ============================================================================
--
-- Migrasi `0002` merancang tujuh tabel milik aplikasi. Setelah klaim, objek, coverage,
-- dan spreading pindah ke tabel bisnis (`0007`, `0008`), tersisa empat — dan Work Owner
-- menetapkan 2026-09-24 bahwa hanya DUA yang benar-benar dibuat:
--
--   CPNC_NOMOR_KLAIM   DILEPAS. Nomor diturunkan dari MAX(CLAIMNO)+1 atas T_CLAIM_PNC,
--                      pola yang sudah terbukti pada RCVN.YY.xxxx. Tidak ada tabel.
--   CPNC_NOTIFIKASI    DITUNDA. Ia kotak keluar surel, bukan syarat registrasi berjalan
--                      — dan SMTP pun belum dikonfigurasi.
--
-- Keduanya yang dibuat di sini TIDAK punya padanan yang sah di skema warisan:
--
--   CPNC_TUGAS         Yang ada hanya DATAPEGA.PC_ASSIGN_WORKLIST dan
--                      PC_ASSIGN_WORKBASKET — tabel ENGINE Pega, dan `D-21` justru
--                      menyebut keduanya sebagai tabel yang harus digantikan. Menulis ke
--                      sana berarti memalsukan baris engine.
--                      POOLDATA.M_ASSIGNMENT bukan penugasan klaim: isinya ID + JSONDATA,
--                      713 baris, tabel konfigurasi.
--
--   CPNC_JEJAK_AUDIT   Pega TIDAK punya jejak audit klaim sama sekali (`S-5` memang baru
--                      100% tanpa baseline). Yang paling mirip — CLAIM_SERVICE_LOG dan
--                      JSON_KLAIM_LOG — justru anti-pola yang Steering §8 tolak: sistem
--                      lama meng-UPDATE yang pertama dan men-DELETE yang kedua. Log yang
--                      dapat diubah dan dihapus bukan log.
--
-- ============================================================================
-- SATU CONSTRAINT YANG DILEPAS DARI RANCANGAN `0002`
-- ============================================================================
--
-- `0002` memasang FOREIGN KEY dari CPNC_TUGAS.KLAIM_ID ke CPNC_KLAIM(ID). Tabel itu
-- tidak jadi dibuat, dan penggantinya — POOLDATA.T_CLAIM_PNC — TIDAK DAPAT menjadi
-- acuan foreign key: ketiga indeksnya NONUNIQUE, dan Oracle menuntut kolom acuan punya
-- kunci utama atau constraint unik.
--
-- Menambahkan constraint unik ke T_CLAIM_PNC BUKAN pilihan yang ringan: tabel itu dibagi
-- dengan Pega, dan constraint baru dapat gagal atas 2.166 baris warisan yang sudah ada
-- sekaligus menolak tulisan Pega yang selama ini diterima.
--
-- Akibatnya diterima secara sadar: keterkaitan tugas ke klaim dijaga APLIKASI, bukan
-- basis data. Itu kelas jaminan yang lebih lemah, dan dicatat di sini supaya kelak tidak
-- terbaca sebagai kelalaian.

DECLARE
    ada NUMBER;
BEGIN
    SELECT COUNT(*) INTO ada
      FROM all_tables WHERE owner = 'POOLDATA' AND table_name = 'CPNC_TUGAS';

    IF ada = 0 THEN
        EXECUTE IMMEDIATE '
            CREATE TABLE POOLDATA.CPNC_TUGAS (
                ID             VARCHAR2(32)  NOT NULL,
                KLAIM_ID       VARCHAR2(100) NOT NULL,
                NOMOR_KLAIM    VARCHAR2(32),
                TAHAP          VARCHAR2(64)  NOT NULL,
                ANTREAN        VARCHAR2(16)  NOT NULL,
                WORKBASKET     VARCHAR2(64),
                PEMILIK        VARCHAR2(64),
                DIBUAT_PADA    TIMESTAMP     NOT NULL,
                DIAMBIL_PADA   TIMESTAMP,
                SELESAI_PADA   TIMESTAMP,
                ALASAN_SELESAI VARCHAR2(64),
                CONSTRAINT PK_CPNC_TUGAS PRIMARY KEY (ID),
                CONSTRAINT CK_CPNC_TUGAS_ANTREAN CHECK (ANTREAN IN (''WORKLIST'', ''WORKBASKET''))
            )';

        EXECUTE IMMEDIATE 'CREATE INDEX POOLDATA.IX_CPNC_TUGAS_PEMILIK ON POOLDATA.CPNC_TUGAS (PEMILIK, SELESAI_PADA)';
        EXECUTE IMMEDIATE 'CREATE INDEX POOLDATA.IX_CPNC_TUGAS_ANTREAN ON POOLDATA.CPNC_TUGAS (WORKBASKET, SELESAI_PADA)';
        EXECUTE IMMEDIATE 'CREATE INDEX POOLDATA.IX_CPNC_TUGAS_KLAIM   ON POOLDATA.CPNC_TUGAS (KLAIM_ID, SELESAI_PADA)';
    END IF;

    SELECT COUNT(*) INTO ada
      FROM all_tables WHERE owner = 'POOLDATA' AND table_name = 'CPNC_JEJAK_AUDIT';

    IF ada = 0 THEN
        EXECUTE IMMEDIATE '
            CREATE TABLE POOLDATA.CPNC_JEJAK_AUDIT (
                ID          VARCHAR2(32)   NOT NULL,
                KLAIM_ID    VARCHAR2(100),
                NOMOR_KLAIM VARCHAR2(32),
                PERISTIWA   VARCHAR2(64)   NOT NULL,
                PELAKU      VARCHAR2(64)   NOT NULL,
                PADA        TIMESTAMP      NOT NULL,
                KETERANGAN  VARCHAR2(2000),
                CONSTRAINT PK_CPNC_JEJAK_AUDIT PRIMARY KEY (ID)
            )';

        EXECUTE IMMEDIATE 'CREATE INDEX POOLDATA.IX_CPNC_JEJAK_AUDIT_KLAIM ON POOLDATA.CPNC_JEJAK_AUDIT (KLAIM_ID, PADA)';
    END IF;
END;
/

-- KLAIM_ID dilebarkan dari VARCHAR2(32) rancangan `0002` menjadi VARCHAR2(100),
-- menyamai POOLDATA.T_CLAIM_PNC.CLAIMID. Kunci klaim warisan berbentuk
-- `ASM-FW-GCNMFW-WORK PNC-996` — 27 karakter — dan membiarkannya 32 akan memotong
-- kunci yang lebih panjang tanpa peringatan.
COMMENT ON COLUMN POOLDATA.CPNC_TUGAS.KLAIM_ID IS 'CLAIMID pada T_CLAIM_PNC; keterkaitannya dijaga aplikasi, bukan foreign key';
COMMENT ON TABLE POOLDATA.CPNC_JEJAK_AUDIT IS 'Jejak audit klaim, hanya bertambah (D-28, ADR-0026); TIDAK ada UPDATE maupun DELETE';
