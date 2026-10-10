-- 0016 — Mengembalikan `ASSIGNED_OPERATOR_ID` klaim PNCN dari nama antrean menjadi nama orang
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- Ia MENULIS ke `POOLDATA.TC_PNC_PUCL` — tabel yang dibagi dengan Pega selama masa paralel.
-- Baris yang disentuh DIBATASI pada klaim `PNCN.%`, yaitu baris yang disisipkan aplikasi ini.
-- Baris Pega — berkunci `PNC-xxxx` — tidak pernah tersentuh, dan pembatasan itu ditegakkan
-- klausa `WHERE`, bukan oleh kehati-hatian pelaksananya.
--
-- ============================================================================
-- APA YANG TERJADI PADA BARIS-BARIS INI
-- ============================================================================
--
-- Sampai 2026-10-06, kolom `ASSIGNED_OPERATOR_ID` memikul DUA arti sekaligus:
--
--	"siapa pemilik klaim ini"          -> nama orang
--	"klaim ini masih di tahap dokter"  -> ditimpa literal `RCLPUCL` begitu dokter memutuskan
--
-- Arti kedua menimpa arti pertama. Di Pega hal itu tidak terjadi karena penyaringnya membaca
-- `pxAssignedOperatorID` sebuah PENUGASAN, dan penugasan lenyap saat tahapnya selesai;
-- `TC_PNC_PUCL` menyimpan satu baris per klaim yang tidak pernah lenyap.
--
-- Work Owner menetapkan 2026-10-07 kolom itu **selalu berisi user teknis**. Arti kedua pindah
-- ke `CPNC_TUGAS` — penyaring E pada `inboxrcl/repo/sqlstore/inboxrcl.sql`. Berkas ini
-- membereskan baris yang terlanjur memegang nama antrean.
--
-- ============================================================================
-- BARIS MANA YANG DISENTUH
-- ============================================================================
--
-- Klaim `PNCN.%` yang `ASSIGNED_OPERATOR_ID`-nya berisi nama antrean. Bacaan katalog
-- 2026-10-07 menemukan TIGA baris, dan ketiganya disetujui Work Owner untuk diperbarui.
--
-- Nilai penggantinya `USER_TEKNIS` baris itu sendiri — bukan nama yang ditetapkan dari luar.
-- Satu baris tidak punya `USER_TEKNIS`; untuk baris seperti itu dipakai `JONNY`, akun yang
-- Work Owner sebut pada 2026-10-07. Mengosongkannya bukan pilihan: kolom kosong tidak cocok
-- dengan penyaring A mana pun, dan klaimnya hilang dari setiap layar tanpa satu pun galat.
--
-- ============================================================================
-- KENAPA INI TIDAK MENARIK KLAIM TERTUTUP KEMBALI KE INBOX RCL
-- ============================================================================
--
-- Penyaring Inbox RCL hanya mengecualikan `Resolved-Completed`, sehingga baris
-- `Resolved-Rejected` LOLOS penyaring status — dan sebelum penyaring E ada, memberi nama
-- orang pada baris seperti itu akan menarik klaim yang sudah ditolak kembali ke antrean
-- dokter.
--
-- Penyaring E yang menutupnya: klaim yang punya tugas di aplikasi ini hanya tampil bila
-- tugas tahap `rcl-dokter`-nya MASIH TERBUKA. Ketiga baris di atas tidak memenuhinya — dua
-- sudah tertutup seluruh tugasnya, satu sedang berada di tahap `rcl-pucl`.
--
-- JANGAN JALANKAN BERKAS INI SEBELUM PENYARING E TERPASANG. Urutannya mengikat: pasang
-- kodenya lebih dulu, baru jalankan migrasi ini.

DECLARE
    terkoreksi NUMBER;
    ada_arsip  NUMBER;
BEGIN
    SELECT COUNT(*) INTO ada_arsip
      FROM ALL_TABLES
     WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'CPNC_PUCL_PEMILIK_DIKOREKSI';

    IF ada_arsip = 0 THEN
        EXECUTE IMMEDIATE '
            CREATE TABLE POOLDATA.CPNC_PUCL_PEMILIK_DIKOREKSI (
                CLAIMID       VARCHAR2(64 CHAR) PRIMARY KEY,
                ASSIGNED_LAMA VARCHAR2(128 CHAR),
                DIKOREKSI_PADA TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )';
    END IF;

    -- Arsip nilai lama — sumber satu-satunya berkas turunnya. Baris yang sudah pernah
    -- diarsipkan tidak diarsipkan ulang, sehingga menjalankan berkas ini dua kali aman.
    INSERT INTO POOLDATA.CPNC_PUCL_PEMILIK_DIKOREKSI (CLAIMID, ASSIGNED_LAMA)
    SELECT p.CLAIMID, p.ASSIGNED_OPERATOR_ID
      FROM POOLDATA.TC_PNC_PUCL p
     WHERE TRIM(p.CLAIMID) LIKE 'PNCN.%'
       AND UPPER(TRIM(p.ASSIGNED_OPERATOR_ID)) = 'RCLPUCL'
       AND NOT EXISTS (SELECT 1
                         FROM POOLDATA.CPNC_PUCL_PEMILIK_DIKOREKSI d
                        WHERE d.CLAIMID = p.CLAIMID);

    -- Kolom pada `SET` sengaja TIDAK diberi awalan alias. Alias pada target `UPDATE` boleh di
    -- Oracle tetapi ditolak PostgreSQL; aliasnya tetap ada supaya `COALESCE` di bawah dapat
    -- menunjuk baris yang sedang diperbarui.
    UPDATE POOLDATA.TC_PNC_PUCL p
       SET ASSIGNED_OPERATOR_ID = COALESCE(NULLIF(TRIM(p.USER_TEKNIS), ''), 'JONNY')
     WHERE TRIM(p.CLAIMID) LIKE 'PNCN.%'
       AND UPPER(TRIM(p.ASSIGNED_OPERATOR_ID)) = 'RCLPUCL';

    terkoreksi := SQL%ROWCOUNT;
    COMMIT;

    DBMS_OUTPUT.PUT_LINE('Pemilik baris RCL/PUCL dikoreksi: ' || terkoreksi);
END;
/
