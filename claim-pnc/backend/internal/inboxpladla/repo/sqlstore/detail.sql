-- Layar RINCIAN satu klaim pada Inbox PLA DLA — tombol **"Detail Claim"**.
--
-- Asalnya:
--
--   Harness/ViewDetailClaimReas-Harness.xml      grid dokumen
--   Section/ViewShowObjectAdjReas-Section.xml    grid PLA, grid DLA, riwayat komunikasi
--   Activity/SetViewAttachmentReas-Act.xml       penyusun daftar dokumen
--   RDB List/GetDokumenReas-SQL.xml              dokumen satu nomor pemberitahuan
--   RDB List/GetAttachmentFromDB_Sql-SQL.xml     ISI satu dokumen
--   RDB List/ReplyKomunikasi-SQL.xml             balasan komunikasi
--
-- ============================================================================
-- SATU ATURAN BERLAKU PADA SETIAP PERNYATAAN DI BERKAS INI
-- ============================================================================
--
-- Setiap satunya WAJIB memuat batas reasuradur pemanggil DI DALAM pernyataannya sendiri —
-- bukan mengandalkan pemeriksaan di lapisan mana pun di atasnya.
--
-- Alasannya bukan mazhab. Seluruh alamat layar ini dapat dipanggil langsung, dan kunci
-- klaim berbentuk `ASM-FW-GCNMFW-WORK PNC-xxxx` — pola yang dapat ditebak. Pernyataan yang
-- kehilangan batasnya akan menyerahkan nilai uang, dokumen, dan isi percakapan milik mitra
-- lain kepada penanya — tanpa satu pun galat.
--
-- `TestEveryDetailQueryIsScopedToTheCaller` menjaga setiap satunya.
--
-- ============================================================================
-- PEGA MENGGAMBAR SELURUH MITRA; DI SINI HANYA MILIK PEMANGGIL
-- ============================================================================
--
-- Grid PLA dan DLA di Pega dimuat dari OBJEK KERJA klaim
-- (`pyWorkPage.ClaimData.ObjectList().ObjectCoverageList().PLAList`), yang memuat seluruh
-- pemberitahuan klaim itu — termasuk milik mitra lain, beserta nilai masing-masing.
--
-- Itu kebocoran yang sejenis dengan grid XOL pada layar induk, dan ia tidak dibawa. Lihat
-- inboxpladla.PlannedDifferences.

-- name: detail_claim_header
-- Keterangan klaim di kepala layar rincian.
--
-- # Kenapa ia memeriksa KEPEMILIKAN, bukan sekadar keberadaan
--
-- Sebuah kunci klaim yang ditebak akan mengembalikan nama tertanggung dan nomor polis
-- kepada siapa pun yang menanyakannya. Syarat `EXISTS` di bawah menuntut sekurang-kurangnya
-- SATU pemberitahuan yang terkirim kepada pemanggil, ATAU satu percakapan yang
-- menyangkutnya — dan keduanya itulah yang membuat klaim ini muncul di salah satu daftar
-- layar induk.
--
-- Klaim yang tidak memenuhi keduanya dijawab "tidak ditemukan", persis seperti klaim yang
-- memang tidak ada. Membedakan keduanya memberi tahu penanya bahwa klaimnya ADA.
--
-- Bind: :1 kunci klaim · :2 login (PLA) · :3 login (DLA) · :4 login (percakapan tujuan)
--       :5 login (percakapan pengirim)
--
-- SUMBER BARU (2026-10-08): kode status dari `T_CLAIM_PNC.STATUSCLAIM`, bukan lagi
-- `STATUSCLAIM_1` tabel kerja Pega. Syarat kepemilikan tidak berubah.
SELECT c.CLAIMID                         AS CLAIM_KEY,
       c.CLAIMNO                         AS CLAIM_NO,
       c.NOPOLIS                         AS POLICY_NO,
       c.QQNAME                          AS INSURED,
       c.BUSINESSNAME                    AS BUSINESS_NAME,
       c.REGISTERDATE                    AS REGISTER_DATE,
       c.DATEOFLOSS                      AS LOSS_DATE,
       c.PICTEKNIK                       AS PIC_TEKNIK,
       c.STATUSCLAIM                     AS STATUS_CODE,
       s.LSC_NOTE                        AS STATUS_LABEL
  FROM POOLDATA.T_CLAIM_PNC c
  LEFT JOIN POOLDATA.M_STS_CLAIM s
         ON s.LSC_ID = c.STATUSCLAIM
 WHERE c.CLAIMID = :1
   AND (EXISTS (SELECT 1
                  FROM POOLDATA.T_PLALIST p
                 WHERE p.CLAIMID = c.CLAIMID
                   AND p.ISKIRIM = '1'
                   AND p.REINSCODE IN (SELECT r.REINSURERID
                                         FROM POOLDATA.T_REINSURER r
                                        WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:2))))
     OR EXISTS (SELECT 1
                  FROM POOLDATA.T_DLALIST d
                 WHERE d.CLAIMID = c.CLAIMID
                   AND d.ISKIRIM = '1'
                   AND d.REINSCODE IN (SELECT r.REINSURERID
                                         FROM POOLDATA.T_REINSURER r
                                        WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:3))))
     OR EXISTS (SELECT 1
                  FROM POOLDATA.M_KOMUNIKASI_PNC k
                 WHERE k.CASEID = c.CLAIMID
                   AND (UPPER(TRIM(k.COMMUNICATE_TO)) = UPPER(TRIM(:4))
                     OR UPPER(TRIM(k.SENDER)) = UPPER(TRIM(:5)))))

-- name: detail_advices_pla
-- Grid **"PLA"** pada layar rincian — pemberitahuan PLA milik pemanggil yang SUDAH terkirim.
--
-- `ISKIRIM = '1'` dituntut, dan itu selisih terhadap grid Pega yang memuat seluruh baris
-- apa pun keadaannya. Dokumen yang belum dikirim belum menjadi milik penerimanya.
--
-- Kolom `ACCEPTANCE_NO` diisi NULL, bukan dihilangkan: satu pemindai Go melayani kedua
-- kueri, dan kolom yang hilang pada salah satunya akan membuatnya gagal memindai.
--
-- Bind: :1 kunci klaim · :2 login
SELECT p.NOPLA                           AS ADVICE_NO,
       p.TIPEPLA                         AS ADVICE_TYPE,
       p.NILAIPLA                        AS AMOUNT,
       NULL                              AS ACCEPTANCE_NO,
       p.TGLPLA                          AS ADVICE_DATE,
       p.TGLKIRIM                        AS SENT_DATE
  FROM POOLDATA.T_PLALIST p
 WHERE p.CLAIMID = :1
   AND p.ISKIRIM = '1'
   AND p.REINSCODE IN (SELECT r.REINSURERID
                         FROM POOLDATA.T_REINSURER r
                        WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:2)))
 ORDER BY p.TGLPLA, p.NOPLA

-- name: detail_advices_dla
-- Grid **"DLA"** pada layar rincian.
--
-- Berbeda dari PLA pada satu kolom: `NOAKSEP` hanya ada di sini. Ia tetap berada di posisi
-- yang SAMA pada senarai kolom supaya satu pemindai melayani kedua kueri.
--
-- Bind: :1 kunci klaim · :2 login
SELECT d.NODLA                           AS ADVICE_NO,
       d.TIPEDLA                         AS ADVICE_TYPE,
       d.NILAIDLA                        AS AMOUNT,
       d.NOAKSEP                         AS ACCEPTANCE_NO,
       d.TGLDLA                          AS ADVICE_DATE,
       d.TGLKIRIM                        AS SENT_DATE
  FROM POOLDATA.T_DLALIST d
 WHERE d.CLAIMID = :1
   AND d.ISKIRIM = '1'
   AND d.REINSCODE IN (SELECT r.REINSURERID
                         FROM POOLDATA.T_REINSURER r
                        WHERE UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:2)))
 ORDER BY d.TGLDLA, d.NODLA

-- name: detail_documents
-- Dokumen satu nomor pemberitahuan — tombol **"Dokumen"** pada baris PLA atau DLA.
--
-- Aslinya, kata demi kata:
--
--   select dokumenid as "DATAID"
--     from pooldata.t_doc_reas
--    where claimid={TempDokReas.CaseID}
--      and no_pladla={TempFilePLADLA.NO_DLA}
--      and tipe_pladla={TempFilePLADLA.DLAType}
--
-- ============================================================================
-- SATU PENYARING DITAMBAHKAN: `LOGIN`
-- ============================================================================
--
-- `T_DOC_REAS` punya kolom `LOGIN` — `InsertDokumenPLADLA` mengisinya saat dokumennya
-- dikirim — tetapi `GetDokumenReas` TIDAK memakainya. Penambahannya diputuskan Work Owner
-- pada 2026-09-28.
--
-- Konsekuensinya disadari dan dinyatakan: bila kolom `LOGIN` pada data lama tidak terisi
-- konsisten, dokumen yang di Pega terlihat akan HILANG di sini. Itu arah kegagalan yang
-- dipilih — kurang, bukan lebih — pada layar yang dibaca pihak luar.
--
-- ============================================================================
-- NAMA JENIS DOKUMEN DICARI LEWAT `LEFT JOIN`, BUKAN `INNER`
-- ============================================================================
--
-- Modul `inboxkomunikasicabang` memakai `INNER JOIN` ke kedua master yang sama, dan di sana
-- itu benar: ia meniru kueri Pega yang memang `INNER`.
--
-- Di sini tidak ada kueri Pega yang menjadi acuan — `SetViewAttachmentReas` membaca nama
-- jenisnya dari objek kerja, bukan dari master. `LEFT JOIN` dipilih karena arah kegagalannya
-- berbeda: dengan `INNER`, satu kode jenis yang tidak ada di master akan MENYEMBUNYIKAN
-- dokumen yang memang hak pemanggil. Kode yang digambar apa adanya lebih baik daripada
-- dokumen yang hilang.
--
-- Bind: :1 kunci klaim · :2 nomor pemberitahuan · :3 jenis pemberitahuan
--       :4 login (pemilik dokumen) · :5 login (batas pemberitahuan)
SELECT a.DATAID                          AS DOCUMENT_ID,
       COALESCE(t.TYPE_DOCUMENT, a.CATEGORY)       AS CATEGORY_NAME,
       COALESCE(dt.DETAIL_DOCUMENT, a.SUB_CATEGORY) AS SUBCATEGORY_NAME,
       a.ATTACHNAME                      AS DOCUMENT_NAME,
       a.ATTACHMIMETYPE                  AS MIME_TYPE
  FROM POOLDATA.T_DOC_REAS r
  JOIN POOLDATA.DATA_ATTACHFILE a
    ON a.DATAID = r.DOKUMENID
  LEFT JOIN POOLDATA.V_LST_DOC_TYPE t
         ON t.ID = a.CATEGORY
  LEFT JOIN POOLDATA.V_LST_DET_TYPE_DOC dt
         ON dt.ID = a.SUB_CATEGORY
 WHERE r.CLAIMID = :1
   AND r.NO_PLADLA = :2
   AND r.TIPE_PLADLA = :3
   AND UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:4))
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_PLALIST p
                WHERE p.CLAIMID = r.CLAIMID
                  AND p.NOPLA = r.NO_PLADLA
                  AND p.ISKIRIM = '1'
                  AND p.REINSCODE IN (SELECT x.REINSURERID
                                        FROM POOLDATA.T_REINSURER x
                                       WHERE UPPER(TRIM(x.LOGIN)) = UPPER(TRIM(:5)))
                UNION ALL
               SELECT 1
                 FROM POOLDATA.T_DLALIST d
                WHERE d.CLAIMID = r.CLAIMID
                  AND d.NODLA = r.NO_PLADLA
                  AND d.ISKIRIM = '1'
                  AND d.REINSCODE IN (SELECT x.REINSURERID
                                        FROM POOLDATA.T_REINSURER x
                                       WHERE UPPER(TRIM(x.LOGIN)) = UPPER(TRIM(:5))))
 ORDER BY a.ATTACHNAME, a.DATAID

-- name: detail_document_content
-- ISI satu dokumen.
--
-- Aslinya `GetAttachmentFromDB_Sql` membungkusnya `pooldata.base64encode(attachfile)`.
-- Pembungkusan itu TIDAK dibawa, dan sebabnya dua: ia memanggil procedure basis data yang
-- `D-02` larang, dan ia membesarkan muatan sepertiga tanpa satu pun manfaat — isinya
-- diserahkan ke klien sebagai berkas, bukan sebagai teks di dalam JSON.
--
-- ============================================================================
-- RANTAI KEPEMILIKANNYA DIPERIKSA ULANG SELURUHNYA
-- ============================================================================
--
-- `DATAID` adalah ANGKA, dan angka dapat ditebak. Pernyataan ini karena itu tidak menerima
-- id dokumen apa adanya: ia menuntut dokumennya terdaftar di `T_DOC_REAS` atas klaim yang
-- diminta, atas login pemanggil, dan atas nomor pemberitahuan yang memang terkirim
-- kepadanya.
--
-- Bind: :1 id dokumen · :2 kunci klaim · :3 login (pemilik dokumen)
--       :4 login (batas pemberitahuan)
SELECT a.ATTACHNAME                      AS DOCUMENT_NAME,
       a.ATTACHMIMETYPE                  AS MIME_TYPE,
       a.ATTACHFILE                      AS CONTENT
  FROM POOLDATA.DATA_ATTACHFILE a
 WHERE a.DATAID = :1
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_DOC_REAS r
                WHERE r.DOKUMENID = a.DATAID
                  AND r.CLAIMID = :2
                  AND UPPER(TRIM(r.LOGIN)) = UPPER(TRIM(:3))
                  AND (EXISTS (SELECT 1
                                 FROM POOLDATA.T_PLALIST p
                                WHERE p.CLAIMID = r.CLAIMID
                                  AND p.NOPLA = r.NO_PLADLA
                                  AND p.ISKIRIM = '1'
                                  AND p.REINSCODE IN
                                      (SELECT x.REINSURERID
                                         FROM POOLDATA.T_REINSURER x
                                        WHERE UPPER(TRIM(x.LOGIN)) = UPPER(TRIM(:4))))
                    OR EXISTS (SELECT 1
                                 FROM POOLDATA.T_DLALIST d
                                WHERE d.CLAIMID = r.CLAIMID
                                  AND d.NODLA = r.NO_PLADLA
                                  AND d.ISKIRIM = '1'
                                  AND d.REINSCODE IN
                                      (SELECT x.REINSURERID
                                         FROM POOLDATA.T_REINSURER x
                                        WHERE UPPER(TRIM(x.LOGIN)) = UPPER(TRIM(:4))))))

-- name: detail_conversations
-- Riwayat komunikasi satu klaim yang menyangkut pemanggil.
--
-- Grid `tempHistoryKomunikasi` pada `ViewShowObjectAdjReas`. Aliasnya di Pega menyesatkan
-- seluruhnya — `.Email` berisi ISI PESAN dan `.CloseClaimNote` berisi balasannya.
--
-- Kedua sisi percakapan diterima (`COMMUNICATE_TO` atau `SENDER`), sama seperti ketiga
-- daftar komunikasi di layar induk: pemanggil berhak membaca percakapan yang ditujukan
-- kepadanya MAUPUN yang ia kirim sendiri.
--
-- Bind: :1 kunci klaim · :2 login (tujuan) · :3 login (pengirim)
SELECT k.KOMUNIKASIID                    AS CONVERSATION_ID,
       k.CREATEDDATE                     AS CREATED_AT,
       k.SENDERNAME                      AS SENDER_NAME,
       k.MESSAGE                         AS MESSAGE,
       k.REPLYMESSAGE                    AS REPLY_MESSAGE,
       k.REPLYFROMNAME                   AS REPLIER_NAME,
       k.CREATEDATEREPLY                 AS REPLIED_AT,
       k.KOMUNIKASISTATUS                AS STATUS
  FROM POOLDATA.M_KOMUNIKASI_PNC k
 WHERE k.CASEID = :1
   AND (UPPER(TRIM(k.COMMUNICATE_TO)) = UPPER(TRIM(:2))
     OR UPPER(TRIM(k.SENDER)) = UPPER(TRIM(:3)))
 ORDER BY k.CREATEDDATE ASC, k.KOMUNIKASIID ASC

-- name: detail_reply
-- **SATU-SATUNYA pernyataan yang MENULIS di modul ini**, dan pelakunya PIHAK LUAR.
--
-- Aslinya, kata demi kata:
--
--   update POOLDATA.m_komunikasi_pnc
--      set replymessage={tempReply.ADDRESS},
--          replyfrom={OperatorID.pyUserIdentifier},
--          CREATEDATEREPLY={temp.AnalystTransferDate DateTime},
--          replyfromname={OperatorID.pyUserName},
--          komunikasistatus='1'
--    where komunikasiid={tempReply.M_SURVEY_ID}
--
-- ============================================================================
-- TIGA SYARAT DITAMBAHKAN PADA KLAUSA `WHERE`
-- ============================================================================
--
--  1. `CASEID = :6` — percakapannya wajib milik klaim yang alamatnya dibuka. Tanpa itu,
--     sebuah nomor percakapan milik klaim lain dapat dibalas lewat alamat klaim yang
--     memang berhak dibuka pemanggil.
--
--  2. `COMMUNICATE_TO = :7 OR SENDER = :8` — percakapannya wajib menyangkut pemanggil.
--     Di Pega tombol balas hanya dapat dicapai dari layar yang sudah menyaringnya; di sini
--     alamatnya dapat dipanggil langsung.
--
--  3. `REPLYMESSAGE IS NULL` — percakapan yang SUDAH dijawab tidak dapat dibalas lagi.
--
--     Ia yang paling penting. `REPLYMESSAGE` adalah kolom TUNGGAL, bukan tabel anak:
--     balasan kedua MENIMPA yang pertama, dan yang pertama tidak dapat dipulihkan dari
--     mana pun. Pemeriksaan terpisah sebelum menulis tidak cukup — dua permintaan yang
--     datang bersamaan akan sama-sama lolos.
--
--     Jumlah baris terpengaruh itulah jawabannya: `0` berarti salah satu dari ketiga
--     syarat gagal, dan pemanggil membedakan sebabnya dengan satu kueri baca SESUDAHNYA —
--     ketika tidak ada lagi yang dapat rusak karenanya.
--
-- Waktu balasannya DIIKAT (`:3`), bukan `sysdate`. `09-DATABASE-STRATEGY.md` §4
-- menuntutnya, dan pada tulisan pihak luar alasannya nyata: waktu yang lahir di basis data
-- tidak dapat diuji, dan balasan pihak luar adalah hal yang paling mungkin dipersoalkan.
--
-- CATATAN PORTABILITAS. Tabelnya sengaja TIDAK diberi alias. Oracle mengizinkan
-- `UPDATE tabel alias SET alias.kolom = …`, PostgreSQL TIDAK.
--
-- Bind: :1 isi balasan · :2 login pembalas · :3 waktu balasan · :4 nama pembalas
--       :5 status · :6 kunci klaim · :7 login (tujuan) · :8 login (pengirim)
--       :9 nomor percakapan
UPDATE POOLDATA.M_KOMUNIKASI_PNC
   SET REPLYMESSAGE     = :1,
       REPLYFROM        = :2,
       CREATEDATEREPLY  = :3,
       REPLYFROMNAME    = :4,
       KOMUNIKASISTATUS = :5
 WHERE CASEID = :6
   AND (UPPER(TRIM(COMMUNICATE_TO)) = UPPER(TRIM(:7))
     OR UPPER(TRIM(SENDER)) = UPPER(TRIM(:8)))
   AND KOMUNIKASIID = :9
   AND REPLYMESSAGE IS NULL

-- name: detail_conversation_exists
-- Memastikan sebuah percakapan ADA dan menyangkut pemanggil.
--
-- Ia dijalankan HANYA ketika detail_reply tidak mengenai satu baris pun, dan gunanya satu:
-- membedakan "percakapannya bukan milik Anda" dari "percakapannya sudah dijawab".
--
-- Kedua sebab itu menuntut kalimat yang berbeda bagi pengguna — yang pertama berarti ia
-- salah alamat, yang kedua berarti pekerjaannya sudah selesai — dan tanpa kueri ini
-- keduanya akan dijawab sama.
--
-- Bind: :1 kunci klaim · :2 nomor percakapan · :3 login (tujuan) · :4 login (pengirim)
SELECT COUNT(*)                          AS TOTAL_ROWS
  FROM POOLDATA.M_KOMUNIKASI_PNC k
 WHERE k.CASEID = :1
   AND k.KOMUNIKASIID = :2
   AND (UPPER(TRIM(k.COMMUNICATE_TO)) = UPPER(TRIM(:3))
     OR UPPER(TRIM(k.SENDER)) = UPPER(TRIM(:4)))
