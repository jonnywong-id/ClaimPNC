-- 0012 — Wilayah kejadian dan Prinsip Mengenal Nasabah pada POOLDATA.T_CLAIM_PNC
--
-- # Kenapa kolom baru
--
-- Layar Input Register Pega (`Section/ViewInputRegisterDetail-Section.xml`) memuat isian
-- wilayah kejadian bertingkat — Negara, Provinsi, Kota, Kabupaten, Kelurahan, Kode Pos —
-- serta Prinsip Mengenal Nasabah (NORMAL / SUSPICIOUS) beserta komentarnya.
--
-- Pega menyimpannya hanya di dokumen case (`ClaimData.*` di JSON_KLAIM). Terverifikasi
-- 2026-09-26: tidak satu pun procedure di `Database/` menuliskannya ke tabel, dan
-- T_CLAIM_PNC — yang dirapikan Work Owner hari ini menjadi 82 kolom — tidak memuat
-- kolomnya. Work Owner menetapkan tempatnya: kolom baru di T_CLAIM_PNC.
--
-- Nama kolom mengikuti nama properti Pega, sehingga penelusurannya ke rule lama langsung.
--
--   COUNTRY / COUNTRYID      ClaimData.Country / CountryID      sumber POOLDATA.COUNTRY
--   PROVINCE / PROVINCEID    ClaimData.Province / ProvinceID    sumber POOLDATA.PROVINCE
--   CITY / CITYID            ClaimData.City / CityID            sumber POOLDATA.CITYINPUT
--   DISTRICT / DISTRICTID    ClaimData.District / DistrictID    sumber POOLDATA.DISTRICTINPUT
--   RW / RWID                ClaimData.RW / RWID  (Kelurahan)   sumber POOLDATA.M_RW
--   POSTALCODE               ClaimData.PostalCode               diisi dari M_RW, dapat diubah
--   CUSTOMERPRINCIPLE        ClaimData.CustomerPrinciple        '1' NORMAL, '2' SUSPICIOUS
--   SUSPICIOUSCOMMENT        ClaimData.SuspiciousComment        tampil bila '2'
--
-- CUSTOMERPRINCIPLE bukan hiasan: `Activity/SetEmailKomite-Act.xml` memeriksa
-- `CustomerPrinciple == "2"` saat menentukan komite.
--
-- # Kenapa aman
--
-- Seluruh kolom NULLABLE tanpa DEFAULT — perubahan metadata saja, tidak ada baris yang
-- ditulis ulang, dan pembaca lama (termasuk rule Pega) tidak melihat perbedaan apa pun.
--
-- # Dijalankan sekali per portal (D-75), idempoten

-- Daftarnya berupa teks "NAMA TIPE", bukan RECORD: Oracle di lingkungan ini tidak
-- menyediakan konstruktor untuk tipe RECORD (PLS-00222, 2026-09-26).
DECLARE
    TYPE t_daftar IS TABLE OF VARCHAR2(80);

    daftar t_daftar := t_daftar(
        'COUNTRY VARCHAR2(200)',
        'COUNTRYID VARCHAR2(20)',
        'PROVINCE VARCHAR2(200)',
        'PROVINCEID VARCHAR2(20)',
        'CITY VARCHAR2(200)',
        'CITYID VARCHAR2(20)',
        'DISTRICT VARCHAR2(200)',
        'DISTRICTID VARCHAR2(20)',
        'RW VARCHAR2(200)',
        'RWID VARCHAR2(20)',
        'POSTALCODE VARCHAR2(10)',
        'CUSTOMERPRINCIPLE VARCHAR2(1)',
        'SUSPICIOUSCOMMENT VARCHAR2(4000)'
    );

    nama VARCHAR2(30);
    ada  NUMBER;
BEGIN
    FOR i IN 1 .. daftar.COUNT LOOP
        nama := SUBSTR(daftar(i), 1, INSTR(daftar(i), ' ') - 1);

        SELECT COUNT(*) INTO ada
          FROM all_tab_cols
         WHERE owner = 'POOLDATA'
           AND table_name = 'T_CLAIM_PNC'
           AND column_name = nama;

        IF ada = 0 THEN
            EXECUTE IMMEDIATE 'ALTER TABLE POOLDATA.T_CLAIM_PNC ADD (' || daftar(i) || ')';
        END IF;
    END LOOP;
END;
/
