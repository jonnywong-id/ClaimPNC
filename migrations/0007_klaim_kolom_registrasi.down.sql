-- 0007 — pembatalan: melepas kelima belas kolom tambahan dari POOLDATA.T_CLAIM_PNC
--
-- ============================================================================
-- BACA INI SEBELUM MENJALANKANNYA
-- ============================================================================
--
-- `DROP COLUMN` MENGHAPUS ISINYA, dan isinya adalah data klaim yang tidak ada di kolom
-- lain mana pun — tahap alur, nilai estimasi, jejak perubahan, penanda penghapusan.
-- Tidak ada cara memulihkannya selain dari cadangan.
--
-- Berbeda dari `up`, berkas ini TIDAK murah dan TIDAK metadata saja: Oracle menulis
-- ulang setiap baris untuk melepas kolom. Pada tabel yang melayani produksi, jalankan
-- hanya di luar jam kerja.
--
-- Ia ada karena `09-DATABASE-STRATEGY.md` §9 mewajibkan setiap migrasi punya `down` yang
-- benar-benar berfungsi — bukan karena melepasnya pernah menjadi rencana.
--
-- Dijalankan setelah modul Registrasi Klaim dipasang, ia akan membuat modul itu menolak
-- menyimpan. Hentikan aplikasinya lebih dulu.

DECLARE
    TYPE t_daftar IS TABLE OF VARCHAR2(30);

    daftar t_daftar := t_daftar(
        'TAHAP_KINI', 'NILAI_ESTIMASI_SEN', 'POLIS_MULAI', 'POLIS_AKHIR',
        'POLIS_DEKLARASI', 'POLIS_PENJAMIN_KREDIT', 'PELAPOR_EMAIL',
        'TRANSFER_COMPLIANCE', 'MINTA_KEMBALI', 'FLAG_KLAIM',
        'STATUS_POSISI_PROGRES', 'DIUBAH_OLEH', 'DIUBAH_PADA',
        'DIHAPUS_OLEH', 'DIHAPUS_PADA',
        'POLIS_MATA_UANG', 'NOMOR_SLIK', 'PELAPOR_HUBUNGAN', 'PELAPOR_HUBUNGAN_LAIN'
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
            EXECUTE IMMEDIATE
                'ALTER TABLE POOLDATA.T_CLAIM_PNC DROP COLUMN ' || daftar(i);
        END IF;
    END LOOP;
END;
/
