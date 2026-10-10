-- 0015 — Memulihkan klaim RCL/PUCL yang ditandai "surat dicetak" padahal suratnya tidak terbit
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- Ia MENULIS ke `POOLDATA.TC_PNC_PUCL` — tabel yang dibagi dengan Pega selama masa paralel.
-- Baris yang disentuh DIBATASI pada klaim `PNCN.%`, yaitu baris yang disisipkan aplikasi ini
-- (lihat `registrasi/repo/sqlstore/rclpucl.sql`). Baris Pega — berkunci `PNC-xxxx` — tidak
-- pernah tersentuh, dan pembatasan itu ditegakkan klausa `WHERE`, bukan oleh kehati-hatian
-- pelaksananya.
--
-- ============================================================================
-- APA YANG TERJADI PADA KLAIM-KLAIM INI
-- ============================================================================
--
-- Tombol "Download Dokumen" menempuh dua langkah berurutan:
--
--	1. menandai surat sudah dicetak   -> TGL_CETAK_DOKUMEN_PUCL, STATUS_CASE, STATUS_CLAIM
--	2. menerbitkan PDF suratnya       -> POOLDATA.DATA_ATTACHFILE
--
-- Langkah 2 membaca kunci klaim lewat kueri `work_object_key`, dan kueri itu sampai
-- 2026-10-05 mencarinya di `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` — tabel yang HANYA memuat klaim
-- yang lahir di Pega. Klaim `PNCN.*` tidak punya baris di sana, sehingga langkah 2 selalu
-- gagal, sementara langkah 1 sudah terlanjur terjadi.
--
-- Kegagalannya SENGAJA tidak membatalkan tindakan (lihat `Service.PerformAction`), sehingga
-- petugas hanya membaca *"Berkas suratnya TIDAK berhasil diterbitkan kali ini"* dan klaimnya
-- berpindah ke tab "Kelengkapan Dokumen" tanpa pernah punya surat.
--
-- Kuerinya sudah diperbaiki pada 2026-10-05; berkas ini membereskan baris yang terlanjur
-- tertinggal.
--
-- ============================================================================
-- KENAPA DUA KOLOM, BUKAN SATU
-- ============================================================================
--
-- Mengosongkan `TGL_CETAK_DOKUMEN_PUCL` SAJA membuat klaimnya HILANG DARI KEDUA TAB:
--
--	tab "Cetak Surat"          TGL_CETAK_DOKUMEN_PUCL IS NULL  AND  STATUS_CASE = '0'
--	tab "Kelengkapan Dokumen"  TGL_CETAK_DOKUMEN_PUCL IS NOT NULL
--
-- Penandaan tadi menaikkan `STATUS_CASE` menjadi `'1'`. Baris ber-tanggal-kosong dan
-- ber-`STATUS_CASE = '1'` tidak memenuhi syarat tab mana pun — klaimnya tidak lagi tergambar
-- di layar mana pun, dan tidak ada satu pun galat yang menyebutkannya. Karena itu `STATUS_CASE`
-- WAJIB ikut dikembalikan ke `'0'`.
--
-- `STATUS_CLAIM` dikembalikan ke `NULL` karena `'1157'` berarti "Document Waiting RCL/PUCL" —
-- pernyataan bahwa suratnya sudah dikirim dan dokumennya sedang ditunggu. Itu tidak pernah
-- terjadi pada baris-baris ini, dan nilainya ikut terbaca laporan harian. Untuk klaim `PNCN.*`
-- nilai sebelumnya memang `NULL`: modul registrasi tidak pernah menulis kolom itu (ia menulis
-- `STATUS_KLAIM`, kolom yang BERBEDA), dan satu-satunya penulis `STATUS_CLAIM` pada jalur ini
-- adalah penandaan tadi.
--
-- `PUCL_APPROVE` TIDAK disentuh — penandaan tadi pun tidak menyentuhnya.
--
-- ============================================================================
-- BARIS MANA YANG DISENTUH
-- ============================================================================
--
--	CLAIMID LIKE 'PNCN.%'          baris milik aplikasi ini (`P-1`)
--	TGL_CETAK_DOKUMEN_PUCL ada     sudah ditandai tercetak
--	PUCL_APPROVE <> '1'            MASIH pekerjaan PUCL — belum dikirim ke Analyst
--	MSIG IS NULL                   jalur non-MSIG; hanya jalur ini punya tombol unduh
--	tidak ada lampiran suratnya    inilah yang membedakan "gagal" dari "berhasil"
--
-- Syarat ketiga penting: klaim yang sudah ditekan "Kirim Ke Analyst" pun mengisi kolom
-- tanggal yang sama, tetapi klaim itu SUDAH BERPINDAH tahap. Mengembalikannya ke tab "Cetak
-- Surat" akan menariknya mundur dari pekerjaan yang sudah selesai.
--
-- Syarat kelima membuat berkas ini IDEMPOTEN dan AMAN dijalankan ulang: begitu suratnya
-- benar-benar terbit, barisnya tidak lagi memenuhi syarat.
--
-- ============================================================================
-- JALANKAN INI LEBIH DULU — LIHAT APA YANG AKAN DISENTUH
-- ============================================================================
--
-- Pernyataan di bawah tidak menulis apa pun. Jalankan, baca hasilnya, dan bawa ke Work Owner
-- sebelum melanjutkan (`D-63`). Nol baris berarti tidak ada yang perlu dikerjakan.
--
--	SELECT p.CLAIMID, p.TGL_CETAK_DOKUMEN_PUCL, p.STATUS_CASE, p.STATUS_CLAIM,
--	       p.RCL_PUCL, p.PUCL_APPROVE
--	  FROM POOLDATA.TC_PNC_PUCL p
--	 WHERE TRIM(p.CLAIMID) LIKE 'PNCN.%'
--	   AND p.TGL_CETAK_DOKUMEN_PUCL IS NOT NULL
--	   AND (p.PUCL_APPROVE IS NULL OR TRIM(p.PUCL_APPROVE) <> '1')
--	   AND p.MSIG IS NULL
--	   AND NOT EXISTS (SELECT 1
--	                     FROM POOLDATA.DATA_ATTACHFILE a
--	                    WHERE a.ATTACHFILE IS NOT NULL
--	                      AND TRIM(a.CATEGORY) = 'Notification'
--	                      AND TRIM(a.ATTACHNAME) IN ('PUCL.pdf', 'RCL.pdf', 'Notification.pdf')
--	                      AND a.IDPEGA IN (SELECT c.CLAIMID
--	                                         FROM POOLDATA.T_CLAIM_PNC c
--	                                        WHERE TRIM(c.CLAIMNO) = TRIM(p.CLAIMID)
--	                                       UNION ALL
--	                                       SELECT 'ASM-FW-GCNMFW-WORK ' || TRIM(c.CLAIMNO)
--	                                         FROM POOLDATA.T_CLAIM_PNC c
--	                                        WHERE TRIM(c.CLAIMNO) = TRIM(p.CLAIMID)))
--	 ORDER BY p.TGL_CETAK_DOKUMEN_PUCL;
--
-- ============================================================================
-- SESUDAHNYA
-- ============================================================================
--
-- Klaimnya kembali ke tab "Cetak Surat". Petugas menekan "Download Dokumen" sekali lagi, dan
-- kali ini suratnya terbit — kueri yang menggagalkannya sudah diperbaiki.


-- ---------------------------------------------------------------------------
-- LANGKAH 1 — tabel arsip
-- ---------------------------------------------------------------------------
--
-- Nilai lama DISALIN sebelum ditimpa, dan itu bukan kelebihan kehati-hatian:
--
--   * `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang, dan berkas ini menulis
--     kolom bernilai bisnis pada tabel yang dibagi dengan sistem yang sedang melayani
--     produksi.
--   * `P-4` menuntut setiap migrasi punya `down` yang BENAR-BENAR berfungsi. Tanpa salinan
--     ini, `down` hanya dapat menebak — dan tebakan pada kolom penyaring tab adalah cara
--     membuat klaim hilang dari layar.
--
-- Tabelnya kecil dan sekali pakai. Ia boleh dibuang setelah pemulihannya dinyatakan selesai;
-- `down` membuangnya sendiri.
--
-- Idempoten: dijalankan berkali-kali tidak menghasilkan galat.
DECLARE
    sudah_ada NUMBER;
BEGIN
    SELECT COUNT(*) INTO sudah_ada
      FROM ALL_TABLES
     WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'CPNC_PUCL_CETAK_DIPULIHKAN';

    IF sudah_ada = 0 THEN
        EXECUTE IMMEDIATE '
            CREATE TABLE POOLDATA.CPNC_PUCL_CETAK_DIPULIHKAN (
                CLAIMID          VARCHAR2(50 CHAR)  NOT NULL,
                TGL_CETAK_LAMA   TIMESTAMP(6),
                STATUS_CASE_LAMA VARCHAR2(10 CHAR),
                STATUS_CLAIM_LAMA VARCHAR2(10 CHAR),
                DIPULIHKAN_PADA  TIMESTAMP(6)       DEFAULT CURRENT_TIMESTAMP NOT NULL,
                CONSTRAINT CPNC_PUCL_CETAK_DIPULIHKAN_PK PRIMARY KEY (CLAIMID)
            )';
    END IF;
END;
/

-- ---------------------------------------------------------------------------
-- LANGKAH 2 — salin nilai lama, lalu kembalikan ketiga kolomnya
-- ---------------------------------------------------------------------------
--
-- Keduanya dalam SATU transaksi: arsip tanpa pemulihan hanyalah baris yang membingungkan,
-- dan pemulihan tanpa arsip tidak dapat dibatalkan.
--
-- `WHERE NOT EXISTS` pada penyisipan arsip menjaga baris yang sudah pernah diarsipkan tidak
-- ditimpa nilai yang sudah dipulihkan — itu akan menghapus justru yang hendak disimpan.
DECLARE
    terpulihkan NUMBER;
BEGIN
    INSERT INTO POOLDATA.CPNC_PUCL_CETAK_DIPULIHKAN
           (CLAIMID, TGL_CETAK_LAMA, STATUS_CASE_LAMA, STATUS_CLAIM_LAMA)
    SELECT p.CLAIMID, p.TGL_CETAK_DOKUMEN_PUCL, p.STATUS_CASE, p.STATUS_CLAIM
      FROM POOLDATA.TC_PNC_PUCL p
     WHERE TRIM(p.CLAIMID) LIKE 'PNCN.%'
       AND p.TGL_CETAK_DOKUMEN_PUCL IS NOT NULL
       AND (p.PUCL_APPROVE IS NULL OR TRIM(p.PUCL_APPROVE) <> '1')
       AND p.MSIG IS NULL
       AND NOT EXISTS (SELECT 1
                         FROM POOLDATA.CPNC_PUCL_CETAK_DIPULIHKAN d
                        WHERE d.CLAIMID = p.CLAIMID)
       AND NOT EXISTS (SELECT 1
                         FROM POOLDATA.DATA_ATTACHFILE a
                        WHERE a.ATTACHFILE IS NOT NULL
                          AND TRIM(a.CATEGORY) = 'Notification'
                          AND TRIM(a.ATTACHNAME) IN ('PUCL.pdf', 'RCL.pdf', 'Notification.pdf')
                          AND a.IDPEGA IN (SELECT c.CLAIMID
                                             FROM POOLDATA.T_CLAIM_PNC c
                                            WHERE TRIM(c.CLAIMNO) = TRIM(p.CLAIMID)
                                           UNION ALL
                                           SELECT 'ASM-FW-GCNMFW-WORK ' || TRIM(c.CLAIMNO)
                                             FROM POOLDATA.T_CLAIM_PNC c
                                            WHERE TRIM(c.CLAIMNO) = TRIM(p.CLAIMID)));

    -- Kolom pada `SET` sengaja TIDAK diberi awalan alias. Alias pada target `UPDATE` boleh di
    -- Oracle tetapi ditolak PostgreSQL, dan modul lain di repo ini sudah punya uji yang
    -- menjaganya (`inboxpladla`, TestTheReplyStatementIsPortable). Aliasnya tetap ada supaya
    -- subkueri terkorelasi dapat menunjuk baris yang sedang diperbarui.
    UPDATE POOLDATA.TC_PNC_PUCL p
       SET TGL_CETAK_DOKUMEN_PUCL = NULL,
           STATUS_CASE            = '0',
           STATUS_CLAIM           = CASE WHEN TRIM(p.STATUS_CLAIM) = '1157'
                                         THEN NULL ELSE p.STATUS_CLAIM END
     WHERE TRIM(p.CLAIMID) LIKE 'PNCN.%'
       AND p.TGL_CETAK_DOKUMEN_PUCL IS NOT NULL
       AND (p.PUCL_APPROVE IS NULL OR TRIM(p.PUCL_APPROVE) <> '1')
       AND p.MSIG IS NULL
       AND NOT EXISTS (SELECT 1
                         FROM POOLDATA.DATA_ATTACHFILE a
                        WHERE a.ATTACHFILE IS NOT NULL
                          AND TRIM(a.CATEGORY) = 'Notification'
                          AND TRIM(a.ATTACHNAME) IN ('PUCL.pdf', 'RCL.pdf', 'Notification.pdf')
                          AND a.IDPEGA IN (SELECT c.CLAIMID
                                             FROM POOLDATA.T_CLAIM_PNC c
                                            WHERE TRIM(c.CLAIMNO) = TRIM(p.CLAIMID)
                                           UNION ALL
                                           SELECT 'ASM-FW-GCNMFW-WORK ' || TRIM(c.CLAIMNO)
                                             FROM POOLDATA.T_CLAIM_PNC c
                                            WHERE TRIM(c.CLAIMNO) = TRIM(p.CLAIMID)));

    terpulihkan := SQL%ROWCOUNT;
    COMMIT;

    DBMS_OUTPUT.PUT_LINE('Klaim RCL/PUCL dipulihkan ke tab "Cetak Surat": ' || terpulihkan);
END;
/
