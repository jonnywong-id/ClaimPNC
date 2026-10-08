-- 0016 turun — mengembalikan `ASSIGNED_OPERATOR_ID` ke nilai yang benar-benar tercatat
-- sebelumnya, bukan ke nilai yang ditebak: sumbernya tabel arsip yang `0016` naik tulis.
--
-- ============================================================================
-- KAPAN BERKAS INI DIJALANKAN
-- ============================================================================
--
-- Bila penyaring E (`inboxrcl/repo/sqlstore/inboxrcl.sql`) dicabut. Tanpa penyaring itu,
-- nama orang pada kolom ini menarik klaim yang sudah lewat dokter kembali ke Inbox RCL —
-- termasuk klaim yang sudah ditolak.
--
-- Jalankan berkas ini LEBIH DULU, baru cabut kodenya. Urutan sebaliknya meninggalkan celah
-- waktu ketika klaim tertutup tergambar di antrean dokter.
--
-- ============================================================================
-- YANG TIDAK DIKEMBALIKAN
-- ============================================================================
--
-- Baris yang sesudah koreksi sudah BERPINDAH tangan lagi tidak disentuh. Nilai barunya
-- berasal dari keputusan yang benar-benar terjadi sesudah migrasi naik, dan menimpanya
-- dengan nama antrean lama akan menghapus catatan yang benar. Syarat
-- `ASSIGNED_OPERATOR_ID = USER_TEKNIS` yang menjaganya — yaitu baris yang masih persis
-- seperti yang `0016` naik tinggalkan.

DECLARE
    terkembalikan NUMBER;
    ada_arsip     NUMBER;
BEGIN
    SELECT COUNT(*) INTO ada_arsip
      FROM ALL_TABLES
     WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'CPNC_PUCL_PEMILIK_DIKOREKSI';

    IF ada_arsip = 0 THEN
        DBMS_OUTPUT.PUT_LINE('Tabel arsip tidak ada; tidak ada yang dikembalikan.');
        RETURN;
    END IF;

    UPDATE POOLDATA.TC_PNC_PUCL p
       SET ASSIGNED_OPERATOR_ID =
           (SELECT d.ASSIGNED_LAMA
              FROM POOLDATA.CPNC_PUCL_PEMILIK_DIKOREKSI d
             WHERE d.CLAIMID = p.CLAIMID)
     WHERE UPPER(TRIM(p.ASSIGNED_OPERATOR_ID)) =
           UPPER(COALESCE(NULLIF(TRIM(p.USER_TEKNIS), ''), 'JONNY'))
       AND EXISTS (SELECT 1
                     FROM POOLDATA.CPNC_PUCL_PEMILIK_DIKOREKSI d
                    WHERE d.CLAIMID = p.CLAIMID);

    terkembalikan := SQL%ROWCOUNT;
    COMMIT;

    DBMS_OUTPUT.PUT_LINE('Pemilik dikembalikan: ' || terkembalikan);
END;
/

-- Arsipnya dibuang setelah dipakai — ia sekali pakai, dan tabel sisa yang tidak dibaca
-- siapa pun hanya menyesatkan pembaca katalog berikutnya.
DECLARE
    ada NUMBER;
BEGIN
    SELECT COUNT(*) INTO ada
      FROM ALL_TABLES
     WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'CPNC_PUCL_PEMILIK_DIKOREKSI';

    IF ada > 0 THEN
        EXECUTE IMMEDIATE 'DROP TABLE POOLDATA.CPNC_PUCL_PEMILIK_DIKOREKSI';
    END IF;
END;
/
