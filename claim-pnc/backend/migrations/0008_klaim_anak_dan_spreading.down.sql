-- 0008 — pembatalan
--
-- BACA DULU. `DROP` di sini menghapus data yang tidak ada di tempat lain:
--   - urutan objek dan coverage, yang menjadi dasar penandaan baris terbuang
--   - lokasi kejadian sebagai teks — LOCATIONID hanya berisi kode, bukan penggantinya
--   - SELURUH isi T_CLAIM_SPREADING, yaitu pembagian risiko setiap klaim
--
-- Berbeda dari `up`, melepas kolom menulis ulang setiap baris. Jalankan di luar jam kerja.
--
-- Ia ada karena `09-DATABASE-STRATEGY.md` §9 mewajibkan setiap migrasi punya `down` yang
-- benar-benar berfungsi — bukan karena melepasnya pernah menjadi rencana.

DECLARE
    TYPE t_kolom IS RECORD (tabel VARCHAR2(30), nama VARCHAR2(30));
    TYPE t_daftar IS TABLE OF t_kolom;

    daftar t_daftar := t_daftar(
        t_kolom('T_CLAIM_OBJECTLIST',     'URUTAN'),
        t_kolom('T_CLAIM_OBJECTLIST',     'LOKASI'),
        t_kolom('T_CLAIM_OBJECTLIST',     'DIHAPUS_PADA'),
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'URUTAN_OBJEK'),
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'URUTAN'),
        t_kolom('T_CLAIM_OBJECTCOVERAGE', 'DIHAPUS_PADA')
    );

    ada NUMBER;
BEGIN
    SELECT COUNT(*) INTO ada
      FROM all_tables WHERE owner = 'POOLDATA' AND table_name = 'T_CLAIM_SPREADING';
    IF ada = 1 THEN
        EXECUTE IMMEDIATE 'DROP TABLE POOLDATA.T_CLAIM_SPREADING';
    END IF;

    FOR i IN 1 .. daftar.COUNT LOOP
        SELECT COUNT(*) INTO ada
          FROM all_tab_cols
         WHERE owner = 'POOLDATA'
           AND table_name = daftar(i).tabel
           AND column_name = daftar(i).nama;

        IF ada = 1 THEN
            EXECUTE IMMEDIATE
                'ALTER TABLE POOLDATA.' || daftar(i).tabel ||
                ' DROP COLUMN ' || daftar(i).nama;
        END IF;
    END LOOP;
END;
/
