-- Membatalkan 0012.
--
-- BACA DULU. DROP COLUMN menghapus wilayah kejadian dan Prinsip Mengenal Nasabah setiap
-- klaim yang sudah diisi. Isian itu tidak ada di tempat lain — Pega menyimpannya hanya di
-- dokumen case, yang tidak ditulis aplikasi ini. Melepas kolom juga menulis ulang setiap
-- baris; jalankan di luar jam kerja.
--
-- Idempoten: kolom yang sudah tidak ada dilewati.

DECLARE
    TYPE t_daftar IS TABLE OF VARCHAR2(30);
    daftar t_daftar := t_daftar(
        'COUNTRY', 'COUNTRYID', 'PROVINCE', 'PROVINCEID', 'CITY', 'CITYID',
        'DISTRICT', 'DISTRICTID', 'RW', 'RWID', 'POSTALCODE',
        'CUSTOMERPRINCIPLE', 'SUSPICIOUSCOMMENT'
    );
    ada NUMBER;
BEGIN
    FOR i IN 1 .. daftar.COUNT LOOP
        SELECT COUNT(*) INTO ada
          FROM all_tab_cols
         WHERE owner = 'POOLDATA'
           AND table_name = 'T_CLAIM_PNC'
           AND column_name = daftar(i);

        IF ada = 1 THEN
            EXECUTE IMMEDIATE 'ALTER TABLE POOLDATA.T_CLAIM_PNC DROP COLUMN ' || daftar(i);
        END IF;
    END LOOP;
END;
/
