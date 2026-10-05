-- 0015 turun — mengembalikan penandaan "surat dicetak" yang dibatalkan `0015` naik.
--
-- ============================================================================
-- KAPAN BERKAS INI DIJALANKAN
-- ============================================================================
--
-- Bila ternyata klaim yang dipulihkan SEHARUSNYA tetap berada di tab "Kelengkapan Dokumen" —
-- misalnya karena suratnya ternyata memang pernah terbit lewat jalur lain, dan syarat
-- "tidak ada lampiran suratnya" pada `0015` naik ternyata terlalu sempit.
--
-- Ia mengembalikan ketiga kolom ke nilai yang BENAR-BENAR tercatat sebelumnya, bukan ke
-- nilai yang ditebak: sumbernya tabel arsip yang `0015` naik tulis.
--
-- ============================================================================
-- YANG TIDAK DIKEMBALIKAN
-- ============================================================================
--
-- Klaim yang sesudah pemulihan sudah DITEKAN ULANG tombol "Download Dokumen"-nya tidak
-- disentuh. Baris seperti itu sudah punya tanggal cetak yang BARU beserta suratnya, dan
-- menimpanya dengan tanggal lama akan mengganti catatan yang benar dengan catatan yang
-- sudah tidak berlaku. Syarat `TGL_CETAK_DOKUMEN_PUCL IS NULL` yang menjaganya.
--
-- Akibatnya berkas ini tidak selalu mengembalikan SELURUH baris yang diarsipkan — dan itu
-- yang dikehendaki.

DECLARE
    terkembalikan NUMBER;
    ada_arsip     NUMBER;
BEGIN
    SELECT COUNT(*) INTO ada_arsip
      FROM ALL_TABLES
     WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'CPNC_PUCL_CETAK_DIPULIHKAN';

    IF ada_arsip = 0 THEN
        DBMS_OUTPUT.PUT_LINE('Tabel arsip tidak ada; tidak ada yang dikembalikan.');
        RETURN;
    END IF;

    -- `SET (kolom, …) = (SELECT …)` adalah sintaks Oracle; berkas migrasi ini memang khusus
    -- Oracle 19c (seluruh blok PL/SQL di sini pun demikian). Kolomnya tidak diberi awalan
    -- alias, dengan alasan yang sama seperti pada berkas naiknya.
    UPDATE POOLDATA.TC_PNC_PUCL p
       SET (TGL_CETAK_DOKUMEN_PUCL, STATUS_CASE, STATUS_CLAIM) =
           (SELECT d.TGL_CETAK_LAMA, d.STATUS_CASE_LAMA, d.STATUS_CLAIM_LAMA
              FROM POOLDATA.CPNC_PUCL_CETAK_DIPULIHKAN d
             WHERE d.CLAIMID = p.CLAIMID)
     WHERE p.TGL_CETAK_DOKUMEN_PUCL IS NULL
       AND EXISTS (SELECT 1
                     FROM POOLDATA.CPNC_PUCL_CETAK_DIPULIHKAN d
                    WHERE d.CLAIMID = p.CLAIMID);

    terkembalikan := SQL%ROWCOUNT;
    COMMIT;

    DBMS_OUTPUT.PUT_LINE('Penandaan dikembalikan: ' || terkembalikan);
END;
/

-- Arsipnya dibuang setelah dipakai — ia sekali pakai, dan tabel sisa yang tidak dibaca
-- siapa pun hanya menyesatkan pembaca katalog berikutnya.
--
-- Dijalankan TERPISAH dari blok di atas supaya kegagalan pengembalian tidak ikut membuang
-- satu-satunya salinan nilai lamanya.
DECLARE
    sudah_ada NUMBER;
BEGIN
    SELECT COUNT(*) INTO sudah_ada
      FROM ALL_TABLES
     WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'CPNC_PUCL_CETAK_DIPULIHKAN';

    IF sudah_ada > 0 THEN
        EXECUTE IMMEDIATE 'DROP TABLE POOLDATA.CPNC_PUCL_CETAK_DIPULIHKAN';
    END IF;
END;
/
