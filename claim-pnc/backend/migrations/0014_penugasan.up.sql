-- 0014 — Penugasan klaim: Worklist dan Workbasket dalam SATU tabel (Oracle 19c)
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- Berkas ini membuat SATU tabel baru. Ia TIDAK menyentuh satu pun objek milik sistem lama
-- — `DATAPEGA.PC_ASSIGN_WORKBASKET` dan `DATAPEGA.PC_ASSIGN_WORKLIST` dibiarkan apa adanya
-- dan tetap dimiliki Pega.
--
-- Tetap menempuh `D-63`: permintaan tertulis, persetujuan Work Owner, pelaksanaan DBA.
--
-- Dijalankan di basis data SETIAP entitas (`D-75`).
--
-- ============================================================================
-- KENAPA SATU TABEL, BUKAN DUA
-- ============================================================================
--
-- Pega punya dua tabel karena ia punya dua KELAS (`Assign-WorkList`, `Assign-WorkBasket`).
-- Kita tidak terikat itu, dan `CONTEXT.md` sudah mendefinisikan Tugas sebagai satu hal:
--
--     satu tugas selalu berada di Worklist ATAU di Workbasket, tidak pernah di keduanya
--
-- Domain Model §4 pun sudah merancangnya sebagai SATU aggregate `Penugasan`, dengan
-- `JenisPenugasan` sebagai pembeda. Tabel ini mengikuti rancangan itu, bukan membuat
-- rancangan baru.
--
-- # Alasan yang lebih menentukan: satu tabel menghapus satu kelas kegagalan
--
-- Perpindahan Compliance → Analyst adalah persis "dari workbasket ke worklist":
--
--     dua tabel : DELETE dari satu + INSERT ke lain — dua pernyataan, bukan satu
--                 transaksi. Bila yang kedua gagal, klaim HILANG DARI MANA-MANA:
--                 tidak di antrean Compliance, tidak di inbox siapa pun, tanpa galat.
--     satu tabel: UPDATE satu baris — atomik dengan sendirinya.
--
-- Itu bahaya yang sama dengan yang menghapus tabel komentar terpisah pada 0012.
--
-- ============================================================================
-- APA YANG TABEL INI BELUM GANTIKAN — DAN INI HARUS DISADARI
-- ============================================================================
--
-- Tujuh modul inbox masih MEMBACA penugasan dari `DATAPEGA.PC_ASSIGN_WORKBASKET`. Tabel
-- ini belum menggantikan mereka; ia dipakai modul Inbox Compliance lebih dulu, sebagai
-- pemakai pertama.
--
-- Konsekuensinya nyata: selama masa itu **penugasan hidup di dua tempat**. Yang menentukan
-- sebuah klaim masih di antrean Compliance adalah KEDUANYA — barisnya ada di tabel Pega,
-- DAN belum ditandai selesai di sini.
--
-- Itu keadaan sementara yang berakhir saat modul `B-6` Penugasan & Inbox dikerjakan, dan
-- ia dinyatakan di muka supaya tidak ditemukan sebagai kejanggalan.
--
-- ============================================================================
-- KENAPA TIDAK MENULIS KE TABEL PEGA
-- ============================================================================
--
-- `Activity/PNC_ReassignPNCTeknik-Act.xml` memperlihatkan caranya Pega sendiri:
--
--     TempIns.pyNote   = "ASSIGN-WORKLIST " + inskey + "!Register_Flow"
--     CALL PXTRANSFERASSIGNMENT
--
-- Pega TIDAK menulis kolom-kolomnya satu per satu — ia memanggil activity bawaan yang
-- mengurus seluruh pembukuan internalnya. `PC_ASSIGN_WORKLIST` punya **56 kolom**, 51
-- terisi, dan sebagian besar bukan data bisnis melainkan pembukuan itu.
--
-- Menirunya dengan INSERT manual berarti menebak apa yang selama ini dikerjakan satu
-- activity bawaan — dan DDL-nya pun belum kita punya (`R-08`). Bila tebakannya meleset,
-- yang rusak bukan layar kita melainkan alur Pega yang masih melayani produksi.

CREATE TABLE POOLDATA.CPNC_PENUGASAN (
    -- Kunci baris, diterbitkan aplikasi.
    PENUGASAN_ID   VARCHAR2(64 BYTE)  NOT NULL,

    -- Klaimnya — `PZINSKEY`, sama dengan kolom bernama sama pada
    -- `CPNC_KEPUTUSAN_COMPLIANCE`.
    NO_KLAIM       VARCHAR2(255 BYTE) NOT NULL,

    -- Tahap alur yang menugaskannya: "Compliance", "Send To Analis", dan seterusnya.
    --
    -- Nilainya nama shape pada `Flow/Register_Flow.xml`, bukan karangan — supaya baris di
    -- sini dapat ditelusuri balik ke flow aslinya.
    TAHAP          VARCHAR2(100 BYTE) NOT NULL,

    -- 'WORKLIST' atau 'WORKBASKET'. Pembeda yang `D-26` tetapkan.
    JENIS          VARCHAR2(10 BYTE)  NOT NULL,

    -- Terisi HANYA pada WORKLIST — login orang yang memegangnya.
    DITUGASKAN_KE  VARCHAR2(128 BYTE),

    -- Terisi HANYA pada WORKBASKET — nama antreannya, mis. "CompliancePNC".
    WORKBASKET     VARCHAR2(100 BYTE),

    -- Penguncian antar-pengguna: siapa yang sedang mengerjakannya.
    --
    -- Dibawa dari rancangan `Penugasan` di Domain Model §4. Belum dipakai modul mana pun
    -- hari ini; kolomnya disiapkan supaya penguncian kelak tidak menuntut DDL baru.
    DIAMBIL_OLEH   VARCHAR2(128 BYTE),
    DIAMBIL_PADA   TIMESTAMP(6),

    -- 'MENUNGGU' atau 'SELESAI'.
    --
    -- Penugasan yang selesai TIDAK dihapus — `D-66`. Ia ditandai, sehingga riwayat
    -- perpindahan satu klaim tetap terbaca seluruhnya.
    STATUS         VARCHAR2(20 BYTE)  NOT NULL,

    DIBUAT_PADA    TIMESTAMP(6)       NOT NULL,
    DIUBAH_PADA    TIMESTAMP(6),

    CONSTRAINT PK_CPNC_PENUGASAN PRIMARY KEY (PENUGASAN_ID),

    CONSTRAINT CK_CPNC_PENUGASAN_JENIS  CHECK (JENIS IN ('WORKLIST', 'WORKBASKET')),
    CONSTRAINT CK_CPNC_PENUGASAN_STATUS CHECK (STATUS IN ('MENUNGGU', 'SELESAI')),

    -- "Tidak pernah di keduanya" ditegakkan BASIS DATA, bukan hanya kode.
    --
    -- Tanpa ini, satu cacat kode dapat menyimpan baris yang sekaligus milik seseorang DAN
    -- milik sebuah antrean — dan baris seperti itu akan muncul di dua inbox sekaligus,
    -- lalu dikerjakan dua kali.
    CONSTRAINT CK_CPNC_PENUGASAN_ISI CHECK (
        (JENIS = 'WORKLIST'   AND DITUGASKAN_KE IS NOT NULL AND WORKBASKET    IS NULL) OR
        (JENIS = 'WORKBASKET' AND WORKBASKET    IS NOT NULL AND DITUGASKAN_KE IS NULL))
);

-- Antrean bersama: "apa yang menunggu di workbasket ini".
CREATE INDEX POOLDATA.CPNC_PENUGASAN_IDX01
    ON POOLDATA.CPNC_PENUGASAN (WORKBASKET, STATUS, DIBUAT_PADA);

-- Inbox pribadi: "apa yang menunggu saya".
CREATE INDEX POOLDATA.CPNC_PENUGASAN_IDX02
    ON POOLDATA.CPNC_PENUGASAN (DITUGASKAN_KE, STATUS, DIBUAT_PADA);

-- Penyaring daftar Compliance: "klaim ini sudah selesai di tahap itu atau belum".
--
-- Dipakai `NOT EXISTS` pada kueri daftar, dan tanpa index ini penyaring itu memaksa
-- pemindaian penuh pada setiap halaman inbox.
CREATE INDEX POOLDATA.CPNC_PENUGASAN_IDX03
    ON POOLDATA.CPNC_PENUGASAN (NO_KLAIM, TAHAP, STATUS);

-- GRANT tidak dicantumkan: aplikasi menyambung sebagai POOLDATA, pemilik skemanya sendiri.
-- Penjelasan lengkapnya ada di 0012_keputusan_compliance.up.sql.
