-- 0007 — Sembilan belas kolom tambahan pada POOLDATA.T_CLAIM_PNC (Oracle 19c)
--
-- ============================================================================
-- KENAPA BERKAS INI ADA
-- ============================================================================
--
-- Work Owner menetapkan 2026-09-24 bahwa baris klaim ditulis LANGSUNG ke
-- POOLDATA.T_CLAIM_PNC, dan PEGA_CONVERT_JSONKLAIM_PNC tidak dijalankan lagi.
--
-- Modul Registrasi Klaim (`B-2`, internal/registrasi) memodelkan 43 kolom. Setelah
-- dipetakan satu per satu ke 75 kolom yang sudah ada, DUA PULUH ENAM memetakan bersih
-- dan SEMBILAN BELAS tidak punya rumah. Berkas ini menambahkan yang sembilan belas.
--
-- Yang paling menentukan adalah TAHAP_KINI: ia menyimpan tahap klaim di dalam
-- Register_Flow. Tanpanya klaim terbit lalu tidak diketahui sedang di tahap mana, dan
-- layar registrasi tidak dapat bekerja sama sekali.
--
-- Dua kolom membayar utang yang sempat diterima pada RCV: DIUBAH_OLEH/PADA
-- mengembalikan jejak siapa mengubah (`D-28`), dan DIHAPUS_PADA mengembalikan soft
-- delete (`ADR-0012`) yang pada T_CLAIM_RECIVEDCLAIM terpaksa hilang.
--
-- ============================================================================
-- KENAPA AMAN DIJALANKAN PADA TABEL YANG MELAYANI PRODUKSI
-- ============================================================================
--
-- Seluruh kolom NULLABLE dan tanpa DEFAULT. Pada Oracle 11g ke atas, penambahan kolom
-- semacam itu adalah operasi METADATA SAJA — tidak satu baris pun ditulis ulang.
--
-- Pega yang membaca tabel ini tidak terpengaruh: kuerinya menyebut kolom satu per satu,
-- dan kolom yang tidak ia sebut tidak mengubah apa pun. Itu tepat yang `P-4` tuntut.
--
-- Risiko yang tersisa satu: `ALTER` menuntut kunci DDL sesaat, sehingga ia dapat gagal
-- dengan ORA-00054 bila ada transaksi Pega yang sedang menahan tabelnya. Kegagalan itu
-- BERSIH — tidak ada perubahan separuh jadi — dan perintahnya tinggal diulang.
--
-- ============================================================================
-- KENAPA DITULIS SEBAGAI BLOK PL/SQL, BUKAN ALTER BIASA
-- ============================================================================
--
-- Oracle tidak mengenal `ADD COLUMN IF NOT EXISTS`. Dijalankan dua kali, `ALTER` biasa
-- gagal dengan ORA-01430 dan menghentikan sisa berkas.
--
-- Berkas ini dijalankan EMPAT KALI — satu per basis data portal (`ADR-0030`) — dan
-- pengulangan pada portal yang sudah terpasang adalah kejadian yang harus diperkirakan,
-- bukan kecelakaan. Karena itu tiap kolom diperiksa lebih dulu.
--
-- ============================================================================
-- DI BASIS DATA MANA
-- ============================================================================
--
-- Di SETIAP portal entitas. Gagal di salah satunya membuat portal itu tertinggal versi,
-- dan modul registrasi pada portal tersebut akan menolak menyimpan.

DECLARE
    -- Daftar kolom beserta tipenya, dalam satu tempat supaya penambahan berikutnya
    -- cukup menyunting satu baris.
    TYPE t_kolom IS RECORD (nama VARCHAR2(30), tipe VARCHAR2(40));
    TYPE t_daftar IS TABLE OF t_kolom;

    daftar t_daftar := t_daftar(
        t_kolom('TAHAP_KINI',            'VARCHAR2(50)'),
        t_kolom('NILAI_ESTIMASI_SEN',    'NUMBER(18)'),
        t_kolom('POLIS_MULAI',           'DATE'),
        t_kolom('POLIS_AKHIR',           'DATE'),
        t_kolom('POLIS_DEKLARASI',       'VARCHAR2(5)'),
        t_kolom('POLIS_PENJAMIN_KREDIT', 'VARCHAR2(5)'),
        t_kolom('PELAPOR_EMAIL',         'VARCHAR2(200)'),
        t_kolom('TRANSFER_COMPLIANCE',   'VARCHAR2(5)'),
        t_kolom('MINTA_KEMBALI',         'VARCHAR2(5)'),
        t_kolom('FLAG_KLAIM',            'VARCHAR2(5)'),
        t_kolom('STATUS_POSISI_PROGRES', 'VARCHAR2(50)'),
        t_kolom('DIUBAH_OLEH',           'VARCHAR2(64)'),
        t_kolom('DIUBAH_PADA',           'TIMESTAMP'),
        t_kolom('DIHAPUS_OLEH',          'VARCHAR2(64)'),
        t_kolom('DIHAPUS_PADA',          'TIMESTAMP'),
        -- Empat berikut ditemukan saat memetakan kolom per kolom, SESUDAH kelima belas
        -- di atas dijalankan. Dua terlewat; dua lagi sempat dipetakan ke kolom warisan
        -- (REPORTTYPE, LAINYA) yang artinya belum dipastikan — menebak arti kolom
        -- adalah utang teknis 4.2 yang proyek ini ada untuk menghapusnya.
        t_kolom('POLIS_MATA_UANG',       'VARCHAR2(10)'),
        t_kolom('NOMOR_SLIK',            'VARCHAR2(50)'),
        t_kolom('PELAPOR_HUBUNGAN',      'VARCHAR2(10)'),
        t_kolom('PELAPOR_HUBUNGAN_LAIN', 'VARCHAR2(500)')
    );

    ada NUMBER;
BEGIN
    FOR i IN 1 .. daftar.COUNT LOOP
        SELECT COUNT(*) INTO ada
          FROM all_tab_cols
         WHERE owner = 'POOLDATA'
           AND table_name = 'T_CLAIM_PNC'
           AND column_name = daftar(i).nama;

        IF ada = 0 THEN
            EXECUTE IMMEDIATE
                'ALTER TABLE POOLDATA.T_CLAIM_PNC ADD (' ||
                daftar(i).nama || ' ' || daftar(i).tipe || ')';
        END IF;
    END LOOP;
END;
/

COMMENT ON COLUMN POOLDATA.T_CLAIM_PNC.TAHAP_KINI IS 'Tahap klaim pada Register_Flow; diisi modul Registrasi Klaim (B-2)';
COMMENT ON COLUMN POOLDATA.T_CLAIM_PNC.NILAI_ESTIMASI_SEN IS 'Nilai estimasi dalam SEN, bukan rupiah (ADR-0016)';
COMMENT ON COLUMN POOLDATA.T_CLAIM_PNC.DIHAPUS_PADA IS 'Penanda soft delete (ADR-0012); NULL berarti baris masih berlaku';
