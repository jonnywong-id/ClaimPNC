-- 0009 — pembatalan
--
-- BACA DULU. Menghapus CPNC_JEJAK_AUDIT membuang JEJAK AUDIT KLAIM, dan `D-59`
-- menjadikannya satu-satunya kontrol pengimbang setelah pemisahan tugas ditiadakan.
-- Tidak ada salinannya di tempat lain; sistem lama tidak pernah punya jejak ini.
--
-- Menghapus CPNC_TUGAS membuang riwayat siapa mengerjakan tahap apa.
--
-- Ia ada karena `09-DATABASE-STRATEGY.md` §9 mewajibkan setiap migrasi punya `down` yang
-- benar-benar berfungsi — bukan karena melepasnya pernah menjadi rencana.

DECLARE
    ada NUMBER;
BEGIN
    FOR t IN (SELECT 'CPNC_TUGAS' AS nama FROM dual
              UNION ALL SELECT 'CPNC_JEJAK_AUDIT' FROM dual) LOOP
        SELECT COUNT(*) INTO ada
          FROM all_tables WHERE owner = 'POOLDATA' AND table_name = t.nama;
        IF ada = 1 THEN
            EXECUTE IMMEDIATE 'DROP TABLE POOLDATA.' || t.nama || ' PURGE';
        END IF;
    END LOOP;
END;
/
