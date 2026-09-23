-- Kueri Inbox Komite — menggantikan TIGA rule SQL sistem lama.
--
-- ============================================================================
-- SELURUH PERNYATAAN DI BERKAS INI HANYA MEMBACA.
-- ============================================================================
--
-- Tidak ada satu pun INSERT, UPDATE, atau DELETE. Keempat tabel di bawah masih ditulis
-- Pega, dan `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem. Yang ditulis
-- aplikasi ini adalah tabel keputusannya SENDIRI — lihat decision.sql.
--
--
-- # Rule yang digantikan
--
--   RDB List/GetKomitePAOutstanding-SQL.xml       → kotak Outstanding
--   RDB List/GetKomitePAditerima-SQL.xml          → kotak Diterima
--   RDB List/ShowKomiteTerimaTolakNonMBU-SQL.xml  → kotak Ditolak
--
-- Ketiganya membaca tabel yang sama dengan penyaring berbeda, dan ketiganya merangkai
-- penyaringnya ke dalam teks SQL lewat `{ASIS:TempKomiteInput.CauseOfLoss}` dan
-- `{Asis:LaporanDataAIKlaim.NoteAITerima}`. Perangkaian itu bukan sekadar tidak rapi: ia
-- celah injeksi (`K-29`, 538 kemunculan pola `{ASIS:...}` di seluruh export).
--
-- Di sini seluruh nilai lewat parameter binding, tanpa perkecualian.
--
--
-- # Tabel yang dibaca
--
--   DATAPEGA.PC_ASM_FW_GCNMFW_WORK      header case; disaring PXOBJCLASS = Work-Komite
--   DATAPEGA.PC_ASSIGN_WORKLIST         penugasan — inilah yang menentukan MILIK SIAPA
--   POOLDATA.T_CLAIM_KOMITE_LIST        nilai, tipe komite, jenjang, keputusan Pega
--   POOLDATA.T_CLAIM_DATA_RESULTS_AI    penilaian AI (OUTER — lihat catatan di bawah)
--   POOLDATA.T_CLAIM_PNC                PIC Teknik klaim
--   POOLDATA.PEGA_DASHBOARDPNC          persentase OR untuk Nilai OR ASM
--   POOLDATA.CPNC_KOMITE_KEPUTUSAN      keputusan MILIK APLIKASI INI (migrasi 0004)
--
--
-- # OUTER JOIN ke penilaian AI WAJIB dipertahankan
--
-- Kueri lama memakai sintaks Oracle lama `A.pyID = AI.KOMITE(+)`. Di sini ia ditulis
-- sebagai LEFT JOIN — `09-DATABASE-STRATEGY.md` §4 menetapkan `(+)` diganti LEFT JOIN
-- demi portabilitas — tetapi SIFATNYA tidak berubah.
--
-- Mengubahnya menjadi INNER akan membuat kasus yang belum dinilai AI HILANG dari inbox
-- tanpa satu pun tanda. Itu kelas cacat paling mahal yang bisa ada di sebuah daftar
-- pekerjaan, dan ia dijaga uji `TestKasusTanpaPenilaianAITetapMuncul`.
--
--
-- # Dua join DIAGREGASI, dan itu penyimpangan yang disengaja
--
-- `T_CLAIM_DATA_RESULTS_AI` dan `PEGA_DASHBOARDPNC` dibaca lewat subkueri ber-GROUP BY,
-- bukan dijoin langsung. Sebabnya berbeda untuk masing-masing, dan keduanya nyata:
--
--   * Penilaian AI. Tidak ada yang menjamin satu baris per KOMITE. Join langsung akan
--     MENGGANDAKAN baris inbox bila ada dua penilaian — dan kueri jumlah di bawah tidak
--     menyentuh tabel itu sama sekali, sehingga jumlah halaman akan berbeda dari isinya.
--     Paginasi yang jumlahnya berbeda dari isinya adalah pekerjaan yang hilang.
--
--   * Persentase OR. Rule lama memakai SUBKUERI SKALAR, yang di Oracle GAGAL bila
--     mengembalikan lebih dari satu baris. Menggantinya dengan join langsung akan
--     menukar galat itu menjadi penggandaan baris yang senyap — jauh lebih buruk.
--     MAX mempertahankan "satu nilai per klaim" tanpa dapat menggandakan apa pun.
--
--
-- # PRSN_PSPLNSOR DIJUMLAHKAN DUA KALI — ditiru, dan diduga cacat
--
-- Rumus persentase OR disalin apa adanya dari `ShowKomiteTerimaTolakNonMBU`:
--
--     PRSN_OR + PRSN_ORS + PRSN_PSRQS_OR + PRSN_FSPLNSOR + PRSN_PSPLNSOR + PRSN_PSPLNSOR
--
-- Suku terakhir muncul DUA KALI, dan kolom `PRSN_FSPLNSOR` yang namanya mirip hanya
-- sekali. Pola itu sangat menyerupai salin-tempel yang lupa diganti — dan bila benar, ia
-- MELEBIHKAN "Nilai OR ASM" yang dibaca komite saat menyetujui uang.
--
-- Ia TETAP DITIRU. `P-5` menetapkan perilaku dipertahankan lebih dulu, dan dugaan ini
-- TIDAK ada di antara 13 perbaikan eksplisit yang `D-49` setujui. Memperbaikinya diam-diam
-- akan memunculkan selisih pada uji kesetaraan yang tidak dapat dipetakan ke butir mana
-- pun — dan `D-54` menuntut setiap selisih seperti itu disetujui Work Owner tertulis.
--
-- Ia dicatat sebagai pertanyaan terbuka di `docs/keputusan-implementasi.md`, bukan
-- diperbaiki di sini.
--
--
-- # Aturan kotak hidup di DUA tempat, dan itu disengaja
--
-- Definisi kanoniknya ada di Go — `komite.CommitteeCase.InBox`. Klausa di bawah
-- MENIRUNYA. Menyaring seluruhnya di Go akan menuntut seluruh antrean komite dibaca ke
-- memori sebelum satu halaman ditampilkan, dan kasus komite tumbuh bersama jumlah klaim.
--
-- Bila salah satu diubah, yang lain WAJIB ikut. Adapter memori memakai definisi Go
-- langsung, sehingga setiap uji yang berjalan tanpa basis data menguji definisi itu —
-- bukan salinan ini.
--
--
-- # Pemotongan prefix kelas Pega
--
-- Awalan `ASM-FW-GCNMFW-WORK ` bocor ke dalam data bisnis sebagai bagian kunci
-- (utang teknis §4.1):
--
--     CLAIMID = 'ASM-FW-GCNMFW-WORK ' || {no_klaim}
--
-- Ia dibuang di sini supaya tidak pernah sampai ke domain maupun ke layar (`D-22`).
--
-- # Kenapa REPLACE, bukan SUBSTR + pencari posisi
--
-- Rule lama memakai `SUBSTR(PNCCASEID, <posisi spasi pertama> + 1)`. Fungsi pencari
-- posisi Oracle TIDAK portabel, dan padanan standarnya yang `09-DATABASE-STRATEGY.md` §4
-- sebut tidak tersedia di Oracle 19c — sehingga kedua bentuk itu melanggar salah satu
-- dari dua basis data yang `D-20` tuntut dilayani satu set SQL yang sama.
--
-- `REPLACE` ada di keduanya, dan ia langsung MEMBALIK penggabungan yang terdokumentasi di
-- atas alih-alih menebaknya dari letak spasi. `TRIM` menutup sisa spasi bila awalannya
-- tersimpan dengan pemisah yang sedikit berbeda.
--
--
-- # Gaya SQL
--
-- COALESCE bukan NVL · CASE WHEN bukan DECODE · LEFT JOIN bukan `(+)` ·
-- OFFSET/FETCH bukan ROWNUM · tanpa TO_CHAR untuk tampilan · kolom selalu disebut
-- namanya. Seluruhnya mengikuti `09-DATABASE-STRATEGY.md` §4, sehingga kueri ini berjalan
-- apa adanya di Oracle 19c maupun PostgreSQL 17+.


-- name: inbox_list
--
-- Satu halaman inbox. Penyaring memakai pola `:n IS NULL OR ...`, sehingga satu kueri
-- melayani seluruh gabungan penyaring tanpa satu pun potongan teks yang dirangkai.
--
-- Urutannya: yang paling lama menunggu di ATAS, persis `ORDER BY "AgingKomite" DESC` pada
-- `GetKomitePAOutstanding`. CASE_ID menjadi pemecah seri supaya urutannya PASTI — dua
-- kasus bertanggal sama tidak boleh berpindah tempat antar permintaan, karena halaman
-- kedua akan melewatkan baris yang berpindah ke halaman pertama.
SELECT c.CASE_ID,
       c.CLAIM_NUMBER,
       c.POLICY_NUMBER,
       c.INSURED_NAME,
       c.BUSINESS_NAME,
       c.SOURCE_OF_BUSINESS,
       c.BRANCH_NAME,
       c.GROUP_PANEL,
       c.CLAIM_PIC,
       c.ASSIGNED_OPERATOR,
       c.COMMITTEE_DATE,
       c.CREATED_AT,
       c.WORK_STATUS,
       c.TYPE_KOMITE,
       c.PAYMENT_TYPE,
       c.CLAIM_VALUE,
       c.ASM_SHARE_VALUE,
       c.OR_VALUE,
       c.COMMITTEE_NOTE,
       c.LEGACY_APPROVE,
       c.LEGACY_TIER,
       c.AI_RESULT,
       c.AI_NOTE_ACCEPTED,
       c.AI_NOTE_REJECTED,
       c.AI_ASSESSED_AT,
       c.AI_PRESENT
  FROM (
        SELECT a.PYID                                              AS CASE_ID,
               TRIM(REPLACE(a.PNCCASEID, 'ASM-FW-GCNMFW-WORK ', ''))    AS CLAIM_NUMBER,
               a.POLICYNO                                          AS POLICY_NUMBER,
               a.QQNAME                                            AS INSURED_NAME,
               a.BUSINESSNAME                                      AS BUSINESS_NAME,
               a.SOBNAME                                           AS SOURCE_OF_BUSINESS,
               a.BRANCHNAME                                        AS BRANCH_NAME,
               a.GROUPPANEL_1                                      AS GROUP_PANEL,
               pnc.PICTEKNIK                                       AS CLAIM_PIC,
               COALESCE(w.PXASSIGNEDOPERATORID, a.PYRESOLVEDUSERID) AS ASSIGNED_OPERATOR,
               COALESCE(k.TANGGALKOMITE, a.PXCREATEDATETIME)       AS COMMITTEE_DATE,
               a.PXCREATEDATETIME                                  AS CREATED_AT,
               a.PYSTATUSWORK                                      AS WORK_STATUS,
               k.TYPEKOMITE                                        AS TYPE_KOMITE,
               k.PAYMENTTYPE                                       AS PAYMENT_TYPE,
               k.NILAIKLAIM                                        AS CLAIM_VALUE,
               k.NILAIKLAIM * k.SHAREASM / 100                     AS ASM_SHARE_VALUE,
               k.NILAIKLAIM * dash.OR_PERCENT / 100               AS OR_VALUE,
               k.NOTEKOMITE                                        AS COMMITTEE_NOTE,
               k.STATUSAPPROVE                                     AS LEGACY_APPROVE,
               k.KOMITEKE                                          AS LEGACY_TIER,
               ai.RESULTAI                                         AS AI_RESULT,
               ai.NOTETERIMA                                       AS AI_NOTE_ACCEPTED,
               ai.NOTETOLAK                                        AS AI_NOTE_REJECTED,
               ai.TGLAI                                            AS AI_ASSESSED_AT,
               CASE WHEN ai.KOMITE IS NULL THEN 0 ELSE 1 END       AS AI_PRESENT,
               d.KEPUTUSAN                                         AS MY_DECISION
          FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
          LEFT JOIN DATAPEGA.PC_ASSIGN_WORKLIST w
                 ON w.PXREFOBJECTKEY = a.PZINSKEY
                AND w.PXOBJCLASS = 'Assign-Worklist'
                AND UPPER(TRIM(w.PXASSIGNEDOPERATORID)) = :1
          LEFT JOIN (
                    SELECT KOMITE_ID,
                           MAX(TYPEKOMITE)    AS TYPEKOMITE,
                           MAX(PAYMENTTYPE)   AS PAYMENTTYPE,
                           MAX(NILAIKLAIM)    AS NILAIKLAIM,
                           MAX(SHAREASM)      AS SHAREASM,
                           MAX(STATUSAPPROVE) AS STATUSAPPROVE,
                           MAX(NOTEKOMITE)    AS NOTEKOMITE,
                           MAX(KOMITEKE)      AS KOMITEKE,
                           MAX(TANGGALKOMITE) AS TANGGALKOMITE
                      FROM POOLDATA.T_CLAIM_KOMITE_LIST
                     GROUP BY KOMITE_ID
                    ) k ON k.KOMITE_ID = a.PYID
          LEFT JOIN (
            SELECT KOMITE,
                   MAX(RESULTAI)   AS RESULTAI,
                   MAX(NOTETERIMA) AS NOTETERIMA,
                   MAX(NOTETOLAK)  AS NOTETOLAK,
                   MAX(TGLAI)      AS TGLAI
              FROM POOLDATA.T_CLAIM_DATA_RESULTS_AI
             GROUP BY KOMITE
            ) ai ON ai.KOMITE = a.PYID
          LEFT JOIN POOLDATA.T_CLAIM_PNC pnc ON pnc.CLAIMID = a.PNCCASEID
          LEFT JOIN (
                    SELECT NOKLAIM,
                           MAX(PRSN_OR + PRSN_ORS + PRSN_PSRQS_OR
                                 + PRSN_FSPLNSOR + PRSN_PSPLNSOR + PRSN_PSPLNSOR) AS OR_PERCENT
                      FROM POOLDATA.PEGA_DASHBOARDPNC
                      GROUP BY NOKLAIM
                    ) dash ON dash.NOKLAIM = TRIM(REPLACE(a.PNCCASEID, 'ASM-FW-GCNMFW-WORK ', ''))
          LEFT JOIN POOLDATA.CPNC_KOMITE_KEPUTUSAN d
                 ON d.CASE_ID = a.PYID
                AND d.ACTOR_LOGIN = :2
         WHERE a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-Komite'
       ) c
 WHERE (c.ASSIGNED_OPERATOR IS NOT NULL
        AND UPPER(TRIM(c.ASSIGNED_OPERATOR)) = :3)
   AND (CASE
          WHEN :4 = 'outstanding'
               AND c.MY_DECISION IS NULL
               AND c.WORK_STATUS <> 'Resolved-Completed' THEN 1
          WHEN :5 = 'diterima'
               AND (c.MY_DECISION = 'setuju'
                    OR (c.MY_DECISION IS NULL AND c.LEGACY_APPROVE = '1')) THEN 1
          WHEN :6 = 'ditolak'
               AND (c.MY_DECISION IN ('tolak', 'kembalikan')
                    OR (c.MY_DECISION IS NULL
                        AND c.LEGACY_APPROVE IS NOT NULL
                        AND c.LEGACY_APPROVE <> '1')) THEN 1
          ELSE 0
        END) = 1
   AND (:7 IS NULL
        OR UPPER(c.CASE_ID)      LIKE '%' || UPPER(:8) || '%' ESCAPE '\'
        OR UPPER(c.CLAIM_NUMBER) LIKE '%' || UPPER(:9) || '%' ESCAPE '\')
   AND (:10 IS NULL OR c.CREATED_AT >= :11)
   AND (:12 IS NULL OR c.CREATED_AT < :13)
 ORDER BY c.COMMITTEE_DATE ASC, c.CASE_ID ASC
OFFSET :14 ROWS FETCH NEXT :15 ROWS ONLY


-- name: inbox_count
--
-- Banyaknya baris yang cocok SEBELUM dipotong paginasi.
--
-- Penyaringnya WAJIB sama persis dengan inbox_list. Bila keduanya berbeda, layar akan
-- menampilkan jumlah halaman yang tidak pernah ada isinya — dan pengguna akan melaporkan
-- pekerjaan yang hilang.
SELECT COUNT(1)
  FROM (
        SELECT a.PYID                                              AS CASE_ID,
               TRIM(REPLACE(a.PNCCASEID, 'ASM-FW-GCNMFW-WORK ', ''))    AS CLAIM_NUMBER,
               COALESCE(w.PXASSIGNEDOPERATORID, a.PYRESOLVEDUSERID) AS ASSIGNED_OPERATOR,
               a.PXCREATEDATETIME                                  AS CREATED_AT,
               a.PYSTATUSWORK                                      AS WORK_STATUS,
               k.STATUSAPPROVE                                     AS LEGACY_APPROVE,
               d.KEPUTUSAN                                         AS MY_DECISION
          FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
          LEFT JOIN DATAPEGA.PC_ASSIGN_WORKLIST w
                 ON w.PXREFOBJECTKEY = a.PZINSKEY
                AND w.PXOBJCLASS = 'Assign-Worklist'
                AND UPPER(TRIM(w.PXASSIGNEDOPERATORID)) = :1
          LEFT JOIN (
                    SELECT KOMITE_ID, MAX(STATUSAPPROVE) AS STATUSAPPROVE
                      FROM POOLDATA.T_CLAIM_KOMITE_LIST
                     GROUP BY KOMITE_ID
                    ) k ON k.KOMITE_ID = a.PYID
          LEFT JOIN POOLDATA.CPNC_KOMITE_KEPUTUSAN d
                 ON d.CASE_ID = a.PYID
                AND d.ACTOR_LOGIN = :2
         WHERE a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-Komite'
       ) c
 WHERE (c.ASSIGNED_OPERATOR IS NOT NULL
        AND UPPER(TRIM(c.ASSIGNED_OPERATOR)) = :3)
   AND (CASE
          WHEN :4 = 'outstanding'
               AND c.MY_DECISION IS NULL
               AND c.WORK_STATUS <> 'Resolved-Completed' THEN 1
          WHEN :5 = 'diterima'
               AND (c.MY_DECISION = 'setuju'
                    OR (c.MY_DECISION IS NULL AND c.LEGACY_APPROVE = '1')) THEN 1
          WHEN :6 = 'ditolak'
               AND (c.MY_DECISION IN ('tolak', 'kembalikan')
                    OR (c.MY_DECISION IS NULL
                        AND c.LEGACY_APPROVE IS NOT NULL
                        AND c.LEGACY_APPROVE <> '1')) THEN 1
          ELSE 0
        END) = 1
   AND (:7 IS NULL
        OR UPPER(c.CASE_ID)      LIKE '%' || UPPER(:8) || '%' ESCAPE '\'
        OR UPPER(c.CLAIM_NUMBER) LIKE '%' || UPPER(:9) || '%' ESCAPE '\')
   AND (:10 IS NULL OR c.CREATED_AT >= :11)
   AND (:12 IS NULL OR c.CREATED_AT < :13)


-- name: inbox_summary
--
-- Jumlah baris KETIGA kotak dalam SATU perjalanan ke basis data.
--
-- Bentuk `SUM(CASE WHEN ...)` diambil dari `RDB List/BrowseClaimRCV_Aksep-SQL.xml`, yang
-- menghitung seluruh lencana sekaligus dengan cara yang sama. Sifatnya dipertahankan
-- karena itulah yang membuat lencana tidak dapat berselisih dengan isi tabel di bawahnya.
--
-- Penyaring kotak TIDAK diterapkan di sini — pencarian dan rentang tanggal diterapkan.
-- Dengan begitu lencana menjawab pertanyaan yang benar: "berapa yang cocok dengan
-- pencarian saya di kotak lain", bukan "berapa isi kotak lain seluruhnya" — yang akan
-- membuat pengguna berpindah tab lalu menemukan tabel kosong.
SELECT SUM(CASE
             WHEN c.MY_DECISION IS NULL
                  AND c.WORK_STATUS <> 'Resolved-Completed' THEN 1
             ELSE 0
           END) AS OUTSTANDING_COUNT,
       SUM(CASE
             WHEN c.MY_DECISION = 'setuju'
                  OR (c.MY_DECISION IS NULL AND c.LEGACY_APPROVE = '1') THEN 1
             ELSE 0
           END) AS ACCEPTED_COUNT,
       SUM(CASE
             WHEN c.MY_DECISION IN ('tolak', 'kembalikan')
                  OR (c.MY_DECISION IS NULL
                      AND c.LEGACY_APPROVE IS NOT NULL
                      AND c.LEGACY_APPROVE <> '1') THEN 1
             ELSE 0
           END) AS REJECTED_COUNT
  FROM (
        SELECT a.PYID                                              AS CASE_ID,
               TRIM(REPLACE(a.PNCCASEID, 'ASM-FW-GCNMFW-WORK ', ''))    AS CLAIM_NUMBER,
               COALESCE(w.PXASSIGNEDOPERATORID, a.PYRESOLVEDUSERID) AS ASSIGNED_OPERATOR,
               a.PXCREATEDATETIME                                  AS CREATED_AT,
               a.PYSTATUSWORK                                      AS WORK_STATUS,
               k.STATUSAPPROVE                                     AS LEGACY_APPROVE,
               d.KEPUTUSAN                                         AS MY_DECISION
          FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
          LEFT JOIN DATAPEGA.PC_ASSIGN_WORKLIST w
                 ON w.PXREFOBJECTKEY = a.PZINSKEY
                AND w.PXOBJCLASS = 'Assign-Worklist'
                AND UPPER(TRIM(w.PXASSIGNEDOPERATORID)) = :1
          LEFT JOIN (
                    SELECT KOMITE_ID, MAX(STATUSAPPROVE) AS STATUSAPPROVE
                      FROM POOLDATA.T_CLAIM_KOMITE_LIST
                     GROUP BY KOMITE_ID
                    ) k ON k.KOMITE_ID = a.PYID
          LEFT JOIN POOLDATA.CPNC_KOMITE_KEPUTUSAN d
                 ON d.CASE_ID = a.PYID
                AND d.ACTOR_LOGIN = :2
         WHERE a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-Komite'
       ) c
 WHERE (c.ASSIGNED_OPERATOR IS NOT NULL
        AND UPPER(TRIM(c.ASSIGNED_OPERATOR)) = :3)
   AND (:4 IS NULL
        OR UPPER(c.CASE_ID)      LIKE '%' || UPPER(:5) || '%' ESCAPE '\'
        OR UPPER(c.CLAIM_NUMBER) LIKE '%' || UPPER(:6) || '%' ESCAPE '\')
   AND (:7 IS NULL OR c.CREATED_AT >= :8)
   AND (:9 IS NULL OR c.CREATED_AT < :10)


-- name: inbox_get
--
-- Satu kasus, TANPA menyaring pemilik.
--
-- Pemeriksaan kepemilikan sengaja tidak ada di sini — ia dikerjakan lapisan usecase lewat
-- BelongsTo, supaya "tidak ada" dan "bukan milik Anda" dapat dibedakan di log meski
-- disamakan di peramban.
--
-- Karena itu pula kolom MY_DECISION tidak dibaca di sini: keputusan diambil terpisah
-- lewat decision.sql untuk SELURUH kasus pada halaman sekaligus.
SELECT a.PYID                                              AS CASE_ID,
       TRIM(REPLACE(a.PNCCASEID, 'ASM-FW-GCNMFW-WORK ', ''))    AS CLAIM_NUMBER,
       a.POLICYNO                                          AS POLICY_NUMBER,
       a.QQNAME                                            AS INSURED_NAME,
       a.BUSINESSNAME                                      AS BUSINESS_NAME,
       a.SOBNAME                                           AS SOURCE_OF_BUSINESS,
       a.BRANCHNAME                                        AS BRANCH_NAME,
       a.GROUPPANEL_1                                      AS GROUP_PANEL,
       pnc.PICTEKNIK                                       AS CLAIM_PIC,
       COALESCE(w.PXASSIGNEDOPERATORID, a.PYRESOLVEDUSERID) AS ASSIGNED_OPERATOR,
       COALESCE(k.TANGGALKOMITE, a.PXCREATEDATETIME)       AS COMMITTEE_DATE,
       a.PXCREATEDATETIME                                  AS CREATED_AT,
       a.PYSTATUSWORK                                      AS WORK_STATUS,
       k.TYPEKOMITE                                        AS TYPE_KOMITE,
       k.PAYMENTTYPE                                       AS PAYMENT_TYPE,
       k.NILAIKLAIM                                        AS CLAIM_VALUE,
       k.NILAIKLAIM * k.SHAREASM / 100                     AS ASM_SHARE_VALUE,
       k.NILAIKLAIM * dash.OR_PERCENT / 100               AS OR_VALUE,
       k.NOTEKOMITE                                        AS COMMITTEE_NOTE,
       k.STATUSAPPROVE                                     AS LEGACY_APPROVE,
       k.KOMITEKE                                          AS LEGACY_TIER,
       ai.RESULTAI                                         AS AI_RESULT,
       ai.NOTETERIMA                                       AS AI_NOTE_ACCEPTED,
       ai.NOTETOLAK                                        AS AI_NOTE_REJECTED,
       ai.TGLAI                                            AS AI_ASSESSED_AT,
       CASE WHEN ai.KOMITE IS NULL THEN 0 ELSE 1 END       AS AI_PRESENT
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
  LEFT JOIN DATAPEGA.PC_ASSIGN_WORKLIST w
         ON w.PXREFOBJECTKEY = a.PZINSKEY
        AND w.PXOBJCLASS = 'Assign-Worklist'
  LEFT JOIN (
            SELECT KOMITE_ID,
                   MAX(TYPEKOMITE)    AS TYPEKOMITE,
                   MAX(PAYMENTTYPE)   AS PAYMENTTYPE,
                   MAX(NILAIKLAIM)    AS NILAIKLAIM,
                   MAX(SHAREASM)      AS SHAREASM,
                   MAX(STATUSAPPROVE) AS STATUSAPPROVE,
                   MAX(NOTEKOMITE)    AS NOTEKOMITE,
                   MAX(KOMITEKE)      AS KOMITEKE,
                   MAX(TANGGALKOMITE) AS TANGGALKOMITE
              FROM POOLDATA.T_CLAIM_KOMITE_LIST
             GROUP BY KOMITE_ID
            ) k ON k.KOMITE_ID = a.PYID
  LEFT JOIN (
            SELECT KOMITE,
                   MAX(RESULTAI)   AS RESULTAI,
                   MAX(NOTETERIMA) AS NOTETERIMA,
                   MAX(NOTETOLAK)  AS NOTETOLAK,
                   MAX(TGLAI)      AS TGLAI
              FROM POOLDATA.T_CLAIM_DATA_RESULTS_AI
             GROUP BY KOMITE
            ) ai ON ai.KOMITE = a.PYID
  LEFT JOIN POOLDATA.T_CLAIM_PNC pnc ON pnc.CLAIMID = a.PNCCASEID
  LEFT JOIN (
            SELECT NOKLAIM,
                   MAX(PRSN_OR + PRSN_ORS + PRSN_PSRQS_OR
                         + PRSN_FSPLNSOR + PRSN_PSPLNSOR + PRSN_PSPLNSOR) AS OR_PERCENT
              FROM POOLDATA.PEGA_DASHBOARDPNC
              GROUP BY NOKLAIM
            ) dash ON dash.NOKLAIM = TRIM(REPLACE(a.PNCCASEID, 'ASM-FW-GCNMFW-WORK ', ''))
 WHERE a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-Komite'
   AND a.PYID = :1


-- name: inbox_check_table
--
-- Memastikan seluruh tabel dan kolomnya dapat dibaca akun aplikasi, tanpa mengambil satu
-- baris pun. Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip:
-- tabelnya tidak ada versus tidak punya hak baca.
SELECT a.PYID,
       a.PNCCASEID,
       a.POLICYNO,
       a.QQNAME,
       a.BUSINESSNAME,
       a.SOBNAME,
       a.BRANCHNAME,
       a.GROUPPANEL_1,
       a.PXCREATEDATETIME,
       a.PYSTATUSWORK,
       a.PYRESOLVEDUSERID,
       a.PZINSKEY
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
 WHERE 1 = 0
