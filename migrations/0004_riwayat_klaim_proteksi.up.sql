-- 0004 — View History Claim: jejak pemakaian proteksi data (Oracle 19c)
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- # Apa yang dibuat, dan apa yang TIDAK disentuh
--
-- Berkas ini hanya MENAMBAH objek baru:
--
--   * POOLDATA.CPNC_PEMAKAIAN_PROTEKSI       tabel baru
--   * POOLDATA.CPNC_PEMAKAIAN_PROTEKSI_SEQ   urutan nomor barisnya
--   * dua indeks pendukung
--
-- TIDAK ADA satu pun objek milik sistem lama yang diubah. Khususnya:
-- POOLDATA.MST_PROTEKSI_DATA_PNC dan POOLDATA.LOG_DATA_PROTEKSI_KLAIM TIDAK DISENTUH
-- SAMA SEKALI — keduanya tetap dibaca dan ditulis Pega seperti sebelumnya.
--
-- BERKAS INI BELUM PERNAH DIJALANKAN DI LINGKUNGAN MANA PUN.
--
-- Ia tetap menempuh prosedur `D-63`: permintaan tertulis tim pengembang, persetujuan Work
-- Owner, pelaksanaan DBA. Akun aplikasi tidak memiliki hak DDL.
--
--
-- # Kenapa tabel baru, bukan tabel yang sudah ada
--
-- Keputusan Work Owner 2026-09-20.
--
-- Sistem lama mengurangi jatah pencarian dengan mengubah tabelnya sendiri
-- (`Activity/InsertLogProteksiDataKlaimMasking-Act.xml`):
--
--     update POOLDATA.MST_PROTEKSI_DATA_PNC set LOGSEARCH = <sisa-1>
--      where LOGIN = <operator> and MODUL = 'PNCSearchKlaim'
--
-- Menirunya berarti aplikasi ini dan Pega sama-sama menulis satu tabel selama masa
-- paralel — tepat yang dilarang `P-1` (`ADR-0004`). Akibatnya BUKAN galat, dan itulah
-- yang membuatnya berbahaya: kedua sistem menulis sisa menurut hitungannya
-- masing-masing, yang menulis belakangan menang, dan jatah seorang pengguna bertambah
-- atau berkurang tanpa satu pun jejak yang menjelaskannya.
--
-- Di sini jatah TIDAK disimpan melainkan DIHITUNG: jatah menurut master dikurangi jumlah
-- baris di tabel ini. Master tetap dibaca apa adanya, dan Pega tetap satu-satunya yang
-- menulisnya.
--
-- Polanya sama persis dengan migrasi 0003: baca tabel lama, tulis tabel sendiri.
--
--
-- # Satu akibat yang harus disadari SEBELUM modul ini menyala di produksi
--
-- Jatah menjadi DUA HITUNGAN yang berjalan berdampingan:
--
--   * Pega mengurangi LOGSEARCH setiap kali layar lamanya dibuka;
--   * aplikasi ini menghitung pemakaiannya sendiri dari tabel ini.
--
-- Selama kedua layar sama-sama hidup, seorang pengguna karena itu memperoleh jatah yang
-- LEBIH BANYAK daripada yang tertulis di master — sebanyak pemakaian di salah satu sistem
-- tidak terlihat oleh sistem yang lain.
--
-- Itu diterima secara sadar: pilihan lainnya adalah dua sistem menulis satu tabel, yang
-- merusak lebih dalam dan lebih sulit dilacak. Ia berakhir dengan sendirinya saat layar
-- Pega dimatikan (Tahap 7).
--
--
-- # Yang HARUS diperiksa DBA sebelum menjalankan
--
-- 1. PASTIKAN NAMANYA BELUM DIPAKAI:
--
--        SELECT object_name, object_type FROM all_objects
--         WHERE owner = 'POOLDATA' AND object_name LIKE 'CPNC_PEMAKAIAN_PROTEKSI%';
--
--    Yang diharapkan nol baris.
--
-- 2. PASTIKAN MASTER PROTEKSI MEMANG BERISI BARIS UNTUK MODUL INI. Tanpa satu baris pun,
--    layar View History Claim menolak SETIAP pengguna — dan penolakan itu benar menurut
--    aturan, tetapi akan dilaporkan sebagai modul yang rusak:
--
--        SELECT COUNT(*) FROM POOLDATA.MST_PROTEKSI_DATA_PNC
--         WHERE UPPER(TRIM(MODUL)) = 'PNCSEARCHKLAIM';
--
--    Bila nol, pendaftaran penggunanya harus dilakukan lebih dulu lewat layar Master
--    Proteksi Data milik sistem lama.
--
-- 3. PASTIKAN TABLESPACE-nya sesuai kebijakan. Berkas ini sengaja TIDAK menyebut
--    tablespace: menyebutkannya berarti menebak tata letak penyimpanan yang dimiliki DBA.


-- ---------------------------------------------------------------------------
-- Langkah 1 — tabel jejak pemakaian.
--
-- # Satu tabel untuk dua hal, dan kenapa tidak dipisah
--
-- Ia memuat DUA jenis baris yang dibedakan kolom MEMAKAI_JATAH:
--
--   MEMAKAI_JATAH = 1   layar dibuka   -> mengurangi jatah
--   MEMAKAI_JATAH = 0   pencarian      -> hanya mencatat
--
-- Pembedaan itu meniru sistem lama, yang hanya mengurangi jatah SEKALI saat layar dibuka
-- (prakondisi langkahnya `TempSearch.SearchType==""`) dan tidak menguranginya lagi pada
-- pencarian berikutnya.
--
-- Keduanya tinggal di satu tabel supaya urutan kejadian pada satu kunjungan terbaca utuh
-- dalam satu urutan waktu. Memisahkannya menjadi dua tabel membuat pertanyaan "apa yang
-- dilakukan pengguna itu sore tadi" harus disusun ulang dari dua tempat.
--
-- # Kenapa nilai pencarian ikut disimpan
--
-- Karena itulah gunanya jejak ini. `D-59` menetapkan satuan izin adalah menu dan TIDAK
-- ADA pemisahan tugas, sehingga jejak audit menjadi satu-satunya kontrol pengimbang yang
-- tersisa. Pertanyaan yang harus dapat dijawabnya adalah "siapa mencari data siapa", dan
-- itu tidak terjawab bila yang tercatat hanya "seseorang membuka layar".
--
-- Isinya adalah nomor polis, nama tertanggung, atau tanggal — data nasabah. Ia tinggal di
-- basis data dan TIDAK PERNAH ikut ke log aplikasi maupun ke dokumen yang di-commit
-- (`D-69`). Akses ke tabel ini karena itu setara dengan akses ke data klaimnya sendiri.
--
-- # Tipe kolom mengikuti 09-DATABASE-STRATEGY.md §5
--
--   * Teks pendek -> VARCHAR2(n) dengan panjang eksplisit
--   * Boolean     -> NUMBER(1), dipetakan di adapter
--   * Waktu       -> TIMESTAMP WITH TIME ZONE, disimpan UTC (DB-8)
-- ---------------------------------------------------------------------------

CREATE TABLE POOLDATA.CPNC_PEMAKAIAN_PROTEKSI (
    ID                NUMBER(19)                 NOT NULL,

    -- LOGIN adalah nama pengguna yang DIKETIK saat masuk, bukan NIK. Itulah yang
    -- dicocokkan ke kolom LOGIN pada MST_PROTEKSI_DATA_PNC, dan itu pula yang dipakai
    -- OperatorID.pyUserIdentifier di sistem lama.
    LOGIN             VARCHAR2(100)              NOT NULL,

    -- MODUL berisi nama HARNESS sistem lama, bukan nama menu — 'PNCSearchKlaim'.
    -- Nilainya harus sama persis dengan kolom MODUL di master proteksi, karena keduanya
    -- dipasangkan saat menghitung sisa jatah.
    MODUL             VARCHAR2(100)              NOT NULL,

    -- Kosong pada baris pembukaan layar: pada saat itu tipe pencarian memang belum
    -- dipilih. NULL, bukan teks kosong — keduanya berbeda artinya di sini.
    TIPE_PENCARIAN    VARCHAR2(10),
    NILAI_PENCARIAN   VARCHAR2(400),

    MEMAKAI_JATAH     NUMBER(1)     DEFAULT 0    NOT NULL,
    DIPAKAI_PADA      TIMESTAMP WITH TIME ZONE   NOT NULL,

    CONSTRAINT CPNC_PEMAKAIAN_PROTEKSI_PK PRIMARY KEY (ID),

    -- Sandi kolom dipagari di basis data, bukan hanya dipercayakan pada kode. Nilai di
    -- luar daftar tidak akan membuat apa pun gagal secara terlihat — ia hanya membuat
    -- barisnya tidak terhitung sebagai pemakaian jatah, dan pengguna memperoleh jatah
    -- tanpa batas tanpa ada yang menyadarinya.
    CONSTRAINT CPNC_PEMAKAIAN_PROTEKSI_CK_JTH CHECK (MEMAKAI_JATAH IN (0, 1))
);


-- ---------------------------------------------------------------------------
-- Langkah 2 — urutan nomor baris.
--
-- NOCACHE dipakai supaya nomor tidak melompat jauh saat instans direstart. Pada dua
-- instans di belakang load balancer (`D-27`), cache yang besar membuat deret nomor
-- terbelah — dan pada tabel jejak, deret yang terbelah membuat urutan kejadian sulit
-- dibaca saat ditelusuri.
-- ---------------------------------------------------------------------------

CREATE SEQUENCE POOLDATA.CPNC_PEMAKAIAN_PROTEKSI_SEQ
    START WITH 1
    INCREMENT BY 1
    NOCACHE
    NOCYCLE;


-- ---------------------------------------------------------------------------
-- Langkah 3 — indeks pendukung.
--
-- Keduanya menjawab kueri yang benar-benar ada, bukan kueri yang dibayangkan:
--
--   IX_..._JATAH   protection_count_usage — dijalankan DUA KALI pada setiap pencarian
--                  dan setiap pembukaan layar. Tanpa indeks ini, setiap pencarian
--                  menuntut pemindaian seluruh tabel jejak, dan tabel jejak hanya
--                  bertambah besar seiring waktu.
--
--   IX_..._TELUSUR penelusuran audit: "apa yang dilakukan pengguna ini, kapan".
--                  Ia TIDAK dipakai aplikasi — ia dipakai manusia yang memeriksa, dan
--                  itulah satu-satunya alasan tabel ini ada.
-- ---------------------------------------------------------------------------

CREATE INDEX POOLDATA.IX_CPNC_PEMAKAIAN_JATAH
    ON POOLDATA.CPNC_PEMAKAIAN_PROTEKSI (LOGIN, MODUL, MEMAKAI_JATAH);

CREATE INDEX POOLDATA.IX_CPNC_PEMAKAIAN_TELUSUR
    ON POOLDATA.CPNC_PEMAKAIAN_PROTEKSI (DIPAKAI_PADA DESC, LOGIN);


-- ---------------------------------------------------------------------------
-- Langkah 4 — hak akses untuk akun aplikasi.
--
-- Diberikan sesempit mungkin: SELECT dan INSERT saja.
--
-- TANPA UPDATE dan TANPA DELETE, dan ketiadaannya disengaja. Tabel ini JEJAK AUDIT, dan
-- `D-28` menetapkan jejak audit bersifat append-only — tidak boleh diubah maupun dihapus
-- oleh jalur aplikasi mana pun. Menegakkannya lewat hak akses, bukan hanya lewat kode,
-- adalah yang `09-DATABASE-STRATEGY.md` §8 tuntut: aturan yang hanya ada di kode dapat
-- dilanggar oleh kode berikutnya; aturan yang ada di hak akses tidak.
--
-- Sistem lama melanggarnya pada tabel lognya sendiri — `UPDATE` pada
-- `pooldata.claim_service_log` dan `DELETE` pada `POOLDATA.JSON_KLAIM_LOG`. Keduanya
-- tidak dibawa (`D-66`).
--
-- Ganti <AKUN_APLIKASI> dengan nama akun yang sebenarnya.
-- ---------------------------------------------------------------------------

-- GRANT SELECT, INSERT ON POOLDATA.CPNC_PEMAKAIAN_PROTEKSI TO <AKUN_APLIKASI>;
-- GRANT SELECT ON POOLDATA.CPNC_PEMAKAIAN_PROTEKSI_SEQ TO <AKUN_APLIKASI>;

-- Akun aplikasi juga membutuhkan hak BACA pada master proteksi milik sistem lama.
-- Ia HANYA SELECT — aplikasi ini tidak pernah menulisnya (lihat bagian "Kenapa tabel
-- baru" di atas).
--
-- GRANT SELECT ON POOLDATA.MST_PROTEKSI_DATA_PNC TO <AKUN_APLIKASI>;
