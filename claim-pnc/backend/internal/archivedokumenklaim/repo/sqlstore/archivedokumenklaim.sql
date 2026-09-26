-- Kueri modul Archive Dokumen Klaim: POOLDATA.T_CLAIM_ARCHIVE_FILE dan kerabatnya.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- ============================================================================
-- PEMETAAN TIGA ARAH — properti klipboard Pega -> kolom sebenarnya -> arti
-- ============================================================================
--
-- Layar lama memakai properti yang sudah ada alih-alih membuat properti baru, sehingga
-- namanya tidak lagi menyatakan isinya. Ini satu-satunya tempat ketiganya dapat
-- dibandingkan berdampingan. Sumbernya `RDB List/SearchDataArchiveFillingCase-SQL.xml`
-- (alias kolomnya) dan captions harness `PNCArchiveDokumen` (arti bagi pengguna).
--
-- Properti klipboard        Kolom sebenarnya   Arti bagi pengguna    Alias di sini
-- ------------------------- ------------------ --------------------- --------------------
-- .IDMasterTONP         (!) ID_ARCHIVE         kunci baris           ARCHIVE_ID
-- .CaseID                   NOKLAIM            No Klaim              CLAIM_NUMBER
-- .PolicyNo                 NOPOLIS            No Polis              POLICY_NUMBER
-- .NIK                  (!) TERTANGGUNG        Nama Tertanggung      INSURED_NAME
-- .DateOfLoss               DOL                Tgl Kejadian          LOSS_DATE
-- .UserTeknis               PICTEKNIK          PIC Teknis            TECHNICAL_PIC
-- .TanggalCetakDLA      (!) TGLTERIMADOK       Tgl Terima Dokumen    RECEIVED_DATE
-- .TanggalAnalystSendRCL(!) TGLINPUT           TGL INPUT             INPUT_DATE
-- .AgingAmount          (!) JUMLAHLEMBAR       Jumlah Lembar         SHEET_COUNT
-- .RWID                 (!) TIPEDOK            Tipe Dokumen          DOCUMENT_TYPE_CODE
-- .TelpTertanggung      (!) JENISDOK           Jenis Dokumen         DOCUMENT_KIND_CODE
-- .CABANG               (!) NAMABOX            Nama BOX              BOX_NAME
-- .KodeCabang           (!) KODEFILLING        Kode Filling          FILLING_CODE
-- .UserName                 USERINPUT          User Input            INPUT_USER
-- .TanggalAI            (!) TGLKIRIMDOK        Tanggal Kirim Dok     SENT_DATE
--
-- Tanda (!) menandai nama yang sama sekali tidak menyatakan isinya. Sebelas dari lima
-- belas. Inilah utang teknis `03-CURRENT-ARCHITECTURE.md` §4.2, dan alasan nama di kode
-- ini tidak mirip nama di Pega (`D-19`).
--
-- Enam kolom lagi tidak pernah muncul di grid lama tetapi menentukan perilaku, sehingga
-- ikut dibaca di sini: GROUPPANEL (saringan kirim ke cabang), CABANGSTATUS (sudah dikirim
-- atau belum), KODECABANG, KODESERVICE, NOTESERVICE, dan HITARCHIVE.
--
-- ============================================================================
-- KEDUA PULUH SATU ALIAS WAJIB SAMA DI SETIAP KUERI DAFTAR
-- ============================================================================
--
-- Urutan DAN namanya. Dua hal bergantung padanya:
--
--   * satu pemindai Go melayani kelima kueri daftar (scanFile di archivedokumenklaim.go);
--   * kueri dibungkus subquery untuk paginasi dan penghitungan (paged/counted di
--     query.go), dan subquery tanpa nama kolom tidak dapat dirujuk dari luar.
--
-- Uji query_test.go menjaga keseragaman ini; ia gagal bila ada kueri daftar yang aliasnya
-- berbeda.
--
-- ============================================================================
-- LIMA HAL YANG BERUBAH DARI KUERI LAMA, DAN ALASANNYA
-- ============================================================================
--
-- 1. PARAMETER BINDING, bukan perangkaian nilai.
--
--    `Activity/SearchDataArchiveFilling-Act.xml` menyusun seluruh klausa WHERE-nya
--    sebagai TEKS lalu menempelkannya lewat `{ASIS:TempClaimAttach.NoteKasir}`. Kata
--    kunci yang diketik pengguna masuk langsung ke dalam teks SQL, dan satu petik
--    tunggal sudah cukup untuk mengubah arti kuerinya. Celah itu tidak dibawa
--    (`08-TECHNICAL-STRATEGY.md` §4.3); larangan perangkaian TIDAK ikut dikecualikan
--    oleh keputusan "replikasi apa adanya" atas aturan bisnis.
--
-- 2. NAMA tipe dan jenis dokumen diambil lewat JOIN, bukan kueri per baris.
--
--    Sistem lama menjalankan DUA kueri tambahan untuk SETIAP baris hasil (langkah 12–16),
--    yaitu 2N+1 perjalanan ke basis data untuk satu halaman. Di sini keduanya menjadi dua
--    LEFT JOIN. Hasilnya sama persis — termasuk nama yang kosong bila kodenya tidak ada
--    di master, karena LEFT JOIN meniru perilaku kueri yang tidak mengembalikan baris.
--
-- 3. PAGINASI dan URUTAN yang ditetapkan.
--
--    Kueri lama tidak menyetel MaxRecords dan tidak punya ORDER BY sama sekali, sehingga
--    pencarian rentang tanggal yang lebar menarik seluruh isi tabel dalam urutan yang
--    diserahkan kepada basis data. Tanpa urutan yang ditetapkan, paginasi membuat satu
--    baris muncul di dua halaman sekaligus hilang dari halaman lain.
--
--    Urutannya ARCHIVE_ID menurun — berkas terbaru lebih dulu, dan ID_ARCHIVE naik
--    monoton sehingga ia juga urutan pengarsipan.
--
-- 4. Kata kunci dibandingkan setelah DIBESARKAN hurufnya di KEDUA sisi.
--
--    Kueri lama menulis `UPPER(NOKLAIM)='<kata kunci>'` — membesarkan huruf kolomnya
--    tetapi tidak nilainya, sehingga nomor klaim yang diketik berhuruf kecil tidak pernah
--    cocok. Dua kolom lainnya (`NAMABOX`, `TERTANGGUNG`) tidak dibesarkan sama sekali.
--    Ketiganya disamakan di sini, dan perbedaan hasilnya disebut terang di
--    docs/keputusan-implementasi.md.
--
-- 5. Pencarian klaim dibatasi jumlah barisnya.
--
--    `SearchArchiveInsert` tidak punya batas. Ketiga bandingannya memang sama-persis,
--    bukan sebagian, sehingga hasilnya wajar sedikit — batas 100 baris di sini adalah
--    pengaman terhadap data yang tidak wajar, bukan paginasi.
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding.

-- name: search_keyword
--
-- Mode "Keyword" — `Activity/SearchDataArchiveFilling-Act.xml` langkah 2:
--
--   WHERE UPPER(NOKLAIM)='<kata kunci>' or NAMABOX='<kata kunci>'
--                                       or TERTANGGUNG='<kata kunci>'
--
-- Ketiganya OR, dan ketiganya cocok PERSIS — bukan sebagian. Itu dipertahankan: mengubah
-- `=` menjadi `LIKE '%…%'` akan membuat pencarian mengembalikan baris yang dulu tidak
-- pernah muncul, dan pada tabel arsip yang tumbuh terus itu perubahan yang tidak dapat
-- ditarik kembali diam-diam.
SELECT a.ID_ARCHIVE                     AS ARCHIVE_ID,
       a.NOKLAIM                        AS CLAIM_NUMBER,
       a.NOPOLIS                        AS POLICY_NUMBER,
       a.TERTANGGUNG                    AS INSURED_NAME,
       a.DOL                            AS LOSS_DATE,
       a.PICTEKNIK                      AS TECHNICAL_PIC,
       a.TGLTERIMADOK                   AS RECEIVED_DATE,
       a.TGLINPUT                       AS INPUT_DATE,
       a.JUMLAHLEMBAR                   AS SHEET_COUNT,
       a.TIPEDOK                        AS DOCUMENT_TYPE_CODE,
       t.STS_PROSES                     AS DOCUMENT_TYPE_NAME,
       a.JENISDOK                       AS DOCUMENT_KIND_CODE,
       d.DETAIL_DOCUMENT                AS DOCUMENT_KIND_NAME,
       a.NAMABOX                        AS BOX_NAME,
       a.KODEFILLING                    AS FILLING_CODE,
       a.USERINPUT                      AS INPUT_USER,
       a.TGLKIRIMDOK                    AS SENT_DATE,
       a.GROUPPANEL                     AS GROUP_PANEL,
       a.CABANGSTATUS                   AS BRANCH_STATUS,
       a.KODESERVICE                    AS SERVICE_CODE,
       a.NOTESERVICE                    AS SERVICE_NOTE
  FROM POOLDATA.T_CLAIM_ARCHIVE_FILE a
  LEFT JOIN POOLDATA.V_LST_DOC_TYPE t
         ON t.ID = a.TIPEDOK
  LEFT JOIN POOLDATA.V_LST_DET_TYPE_DOC d
         ON d.ID = a.JENISDOK
        AND d.DOC_TYPE_ID = a.TIPEDOK
-- Nilainya SUDAH dibesarkan hurufnya di Go, dan ketiga penanda diisi nilai yang sama.
-- Penanda yang berbeda untuk nilai yang sama tampak berlebihan, tetapi itulah yang
-- ditempuh seluruh modul lain: satu penanda yang dipakai berulang berperilaku berbeda
-- antar driver, dan perbedaannya baru terlihat saat dijalankan.
 WHERE UPPER(a.NOKLAIM) = :1
    OR UPPER(a.NAMABOX) = :2
    OR UPPER(a.TERTANGGUNG) = :3

-- name: search_input_date
--
-- Mode "Tgl Input" — `Activity/SearchDataArchiveFilling-Act.xml` langkah 10:
--
--   WHERE trunc(TGLINPUT) >= to_date(<awal>) and trunc(TGLINPUT) <= to_date(<akhir>)
--
-- Rentangnya INKLUSIF di kedua ujung, dan `trunc` membuang jamnya — berkas yang diinput
-- pukul 16:00 pada tanggal akhir tetap ikut. Keduanya dipertahankan.
--
-- Perbedaan bentuk: `trunc(TGLINPUT)` diganti perbandingan rentang terhadap kolom apa
-- adanya, dengan batas atas digeser satu hari dan dibuat eksklusif. Hasilnya identik,
-- tetapi index pada TGLINPUT tetap dapat dipakai — `trunc` pada kolom membuat setiap
-- pencarian memindai seluruh tabel (`09-DATABASE-STRATEGY.md` §3.2).
SELECT a.ID_ARCHIVE                     AS ARCHIVE_ID,
       a.NOKLAIM                        AS CLAIM_NUMBER,
       a.NOPOLIS                        AS POLICY_NUMBER,
       a.TERTANGGUNG                    AS INSURED_NAME,
       a.DOL                            AS LOSS_DATE,
       a.PICTEKNIK                      AS TECHNICAL_PIC,
       a.TGLTERIMADOK                   AS RECEIVED_DATE,
       a.TGLINPUT                       AS INPUT_DATE,
       a.JUMLAHLEMBAR                   AS SHEET_COUNT,
       a.TIPEDOK                        AS DOCUMENT_TYPE_CODE,
       t.STS_PROSES                     AS DOCUMENT_TYPE_NAME,
       a.JENISDOK                       AS DOCUMENT_KIND_CODE,
       d.DETAIL_DOCUMENT                AS DOCUMENT_KIND_NAME,
       a.NAMABOX                        AS BOX_NAME,
       a.KODEFILLING                    AS FILLING_CODE,
       a.USERINPUT                      AS INPUT_USER,
       a.TGLKIRIMDOK                    AS SENT_DATE,
       a.GROUPPANEL                     AS GROUP_PANEL,
       a.CABANGSTATUS                   AS BRANCH_STATUS,
       a.KODESERVICE                    AS SERVICE_CODE,
       a.NOTESERVICE                    AS SERVICE_NOTE
  FROM POOLDATA.T_CLAIM_ARCHIVE_FILE a
  LEFT JOIN POOLDATA.V_LST_DOC_TYPE t
         ON t.ID = a.TIPEDOK
  LEFT JOIN POOLDATA.V_LST_DET_TYPE_DOC d
         ON d.ID = a.JENISDOK
        AND d.DOC_TYPE_ID = a.TIPEDOK
 WHERE a.TGLINPUT >= :1
   AND a.TGLINPUT < :2

-- name: pending_all
--
-- Daftar kirim ke cabang, tanpa saringan lini bisnis — jabatan pemanggil bukan NONMBU,
-- PA, maupun TRAVEL. `Activity/GetDataArchiveCabangKlaim-Act.xml` langkah 1.
SELECT a.ID_ARCHIVE                     AS ARCHIVE_ID,
       a.NOKLAIM                        AS CLAIM_NUMBER,
       a.NOPOLIS                        AS POLICY_NUMBER,
       a.TERTANGGUNG                    AS INSURED_NAME,
       a.DOL                            AS LOSS_DATE,
       a.PICTEKNIK                      AS TECHNICAL_PIC,
       a.TGLTERIMADOK                   AS RECEIVED_DATE,
       a.TGLINPUT                       AS INPUT_DATE,
       a.JUMLAHLEMBAR                   AS SHEET_COUNT,
       a.TIPEDOK                        AS DOCUMENT_TYPE_CODE,
       t.STS_PROSES                     AS DOCUMENT_TYPE_NAME,
       a.JENISDOK                       AS DOCUMENT_KIND_CODE,
       d.DETAIL_DOCUMENT                AS DOCUMENT_KIND_NAME,
       a.NAMABOX                        AS BOX_NAME,
       a.KODEFILLING                    AS FILLING_CODE,
       a.USERINPUT                      AS INPUT_USER,
       a.TGLKIRIMDOK                    AS SENT_DATE,
       a.GROUPPANEL                     AS GROUP_PANEL,
       a.CABANGSTATUS                   AS BRANCH_STATUS,
       a.KODESERVICE                    AS SERVICE_CODE,
       a.NOTESERVICE                    AS SERVICE_NOTE
  FROM POOLDATA.T_CLAIM_ARCHIVE_FILE a
  LEFT JOIN POOLDATA.V_LST_DOC_TYPE t
         ON t.ID = a.TIPEDOK
  LEFT JOIN POOLDATA.V_LST_DET_TYPE_DOC d
         ON d.ID = a.JENISDOK
        AND d.DOC_TYPE_ID = a.TIPEDOK
 WHERE a.CABANGSTATUS = '0'

-- name: pending_exclude_one
--
-- Satu lini bisnis disembunyikan — jabatan PA atau TRAVEL.
--
-- Bahwa petugas PA justru tidak melihat berkas PA TAMPAK TERBALIK, dan memang begitulah
-- sistem lama berperilaku. Ia direplikasi atas keputusan Work Owner 2026-09-24; lihat
-- archivedokumenklaim.BranchScope.
SELECT a.ID_ARCHIVE                     AS ARCHIVE_ID,
       a.NOKLAIM                        AS CLAIM_NUMBER,
       a.NOPOLIS                        AS POLICY_NUMBER,
       a.TERTANGGUNG                    AS INSURED_NAME,
       a.DOL                            AS LOSS_DATE,
       a.PICTEKNIK                      AS TECHNICAL_PIC,
       a.TGLTERIMADOK                   AS RECEIVED_DATE,
       a.TGLINPUT                       AS INPUT_DATE,
       a.JUMLAHLEMBAR                   AS SHEET_COUNT,
       a.TIPEDOK                        AS DOCUMENT_TYPE_CODE,
       t.STS_PROSES                     AS DOCUMENT_TYPE_NAME,
       a.JENISDOK                       AS DOCUMENT_KIND_CODE,
       d.DETAIL_DOCUMENT                AS DOCUMENT_KIND_NAME,
       a.NAMABOX                        AS BOX_NAME,
       a.KODEFILLING                    AS FILLING_CODE,
       a.USERINPUT                      AS INPUT_USER,
       a.TGLKIRIMDOK                    AS SENT_DATE,
       a.GROUPPANEL                     AS GROUP_PANEL,
       a.CABANGSTATUS                   AS BRANCH_STATUS,
       a.KODESERVICE                    AS SERVICE_CODE,
       a.NOTESERVICE                    AS SERVICE_NOTE
  FROM POOLDATA.T_CLAIM_ARCHIVE_FILE a
  LEFT JOIN POOLDATA.V_LST_DOC_TYPE t
         ON t.ID = a.TIPEDOK
  LEFT JOIN POOLDATA.V_LST_DET_TYPE_DOC d
         ON d.ID = a.JENISDOK
        AND d.DOC_TYPE_ID = a.TIPEDOK
 WHERE a.CABANGSTATUS = '0'
   AND (a.GROUPPANEL IS NULL OR a.GROUPPANEL <> :1)

-- name: pending_exclude_two
--
-- Dua lini bisnis disembunyikan — jabatan NONMBU, yang tidak melihat PA maupun TRAVEL.
--
-- `GROUPPANEL IS NULL` sengaja diloloskan. `not in ('002','005')` pada Oracle TIDAK
-- meloloskan baris ber-NULL, sehingga berkas yang lini bisnisnya kosong hilang dari
-- daftar tanpa satu pun tanda — dan berkas yang hilang dari daftar tidak akan pernah
-- dikirim ke sistem Arsip. Perbedaan ini disebut terang di
-- docs/keputusan-implementasi.md.
SELECT a.ID_ARCHIVE                     AS ARCHIVE_ID,
       a.NOKLAIM                        AS CLAIM_NUMBER,
       a.NOPOLIS                        AS POLICY_NUMBER,
       a.TERTANGGUNG                    AS INSURED_NAME,
       a.DOL                            AS LOSS_DATE,
       a.PICTEKNIK                      AS TECHNICAL_PIC,
       a.TGLTERIMADOK                   AS RECEIVED_DATE,
       a.TGLINPUT                       AS INPUT_DATE,
       a.JUMLAHLEMBAR                   AS SHEET_COUNT,
       a.TIPEDOK                        AS DOCUMENT_TYPE_CODE,
       t.STS_PROSES                     AS DOCUMENT_TYPE_NAME,
       a.JENISDOK                       AS DOCUMENT_KIND_CODE,
       d.DETAIL_DOCUMENT                AS DOCUMENT_KIND_NAME,
       a.NAMABOX                        AS BOX_NAME,
       a.KODEFILLING                    AS FILLING_CODE,
       a.USERINPUT                      AS INPUT_USER,
       a.TGLKIRIMDOK                    AS SENT_DATE,
       a.GROUPPANEL                     AS GROUP_PANEL,
       a.CABANGSTATUS                   AS BRANCH_STATUS,
       a.KODESERVICE                    AS SERVICE_CODE,
       a.NOTESERVICE                    AS SERVICE_NOTE
  FROM POOLDATA.T_CLAIM_ARCHIVE_FILE a
  LEFT JOIN POOLDATA.V_LST_DOC_TYPE t
         ON t.ID = a.TIPEDOK
  LEFT JOIN POOLDATA.V_LST_DET_TYPE_DOC d
         ON d.ID = a.JENISDOK
        AND d.DOC_TYPE_ID = a.TIPEDOK
 WHERE a.CABANGSTATUS = '0'
   AND (a.GROUPPANEL IS NULL OR a.GROUPPANEL NOT IN (:1, :2))

-- name: find_by_id
--
-- Satu baris arsip. Bentuk kolomnya sama persis dengan kueri daftar supaya keduanya dapat
-- dibaca satu pemindai — bila keduanya berbeda, satu perubahan kolom harus diingat di dua
-- tempat.
SELECT a.ID_ARCHIVE                     AS ARCHIVE_ID,
       a.NOKLAIM                        AS CLAIM_NUMBER,
       a.NOPOLIS                        AS POLICY_NUMBER,
       a.TERTANGGUNG                    AS INSURED_NAME,
       a.DOL                            AS LOSS_DATE,
       a.PICTEKNIK                      AS TECHNICAL_PIC,
       a.TGLTERIMADOK                   AS RECEIVED_DATE,
       a.TGLINPUT                       AS INPUT_DATE,
       a.JUMLAHLEMBAR                   AS SHEET_COUNT,
       a.TIPEDOK                        AS DOCUMENT_TYPE_CODE,
       t.STS_PROSES                     AS DOCUMENT_TYPE_NAME,
       a.JENISDOK                       AS DOCUMENT_KIND_CODE,
       d.DETAIL_DOCUMENT                AS DOCUMENT_KIND_NAME,
       a.NAMABOX                        AS BOX_NAME,
       a.KODEFILLING                    AS FILLING_CODE,
       a.USERINPUT                      AS INPUT_USER,
       a.TGLKIRIMDOK                    AS SENT_DATE,
       a.GROUPPANEL                     AS GROUP_PANEL,
       a.CABANGSTATUS                   AS BRANCH_STATUS,
       a.KODESERVICE                    AS SERVICE_CODE,
       a.NOTESERVICE                    AS SERVICE_NOTE
  FROM POOLDATA.T_CLAIM_ARCHIVE_FILE a
  LEFT JOIN POOLDATA.V_LST_DOC_TYPE t
         ON t.ID = a.TIPEDOK
  LEFT JOIN POOLDATA.V_LST_DET_TYPE_DOC d
         ON d.ID = a.JENISDOK
        AND d.DOC_TYPE_ID = a.TIPEDOK
 WHERE a.ID_ARCHIVE = :1

-- name: search_claim_any
--
-- Calon klaim untuk tipe input "No Klaim" dan "Nama Tertanggung" —
-- `RDB List/SearchArchiveInsert-SQL.xml`.
--
-- Ketiga bandingannya OR, persis kueri lama: apa pun tipe input yang dipilih, nilai yang
-- diketik dicocokkan ke nomor klaim, nomor polis, DAN nama tertanggung sekaligus.
-- Dropdown "Tipe Input Archive" karena itu praktis tidak mempersempit apa pun — dan itu
-- memang perilaku sistem lama, bukan kelalaian pembacaan.
--
-- SATU PERUBAHAN, dan ia dituntut `D-22`. Kueri lama mencocokkan
-- `CLAIMID = 'ASM-FW-GCNMFW-WORK ' || <nilai>`, merangkai nama kelas internal Pega ke
-- dalam kunci pencarian. Klaim terbitan sistem baru berformat `PNCN.YY.xxxx` dan tidak
-- pernah menulis awalan itu lagi (`D-71`), sehingga kueri lama tidak akan pernah
-- menemukannya. Yang dicocokkan di sini adalah CLAIMNO — setara untuk baris warisan,
-- dan ikut menemukan baris terbitan sistem baru. Sama dengan yang ditempuh modul View
-- History Claim.
SELECT c.CLAIMNO                                        AS CLAIM_NUMBER,
       c.NOPOLIS                                        AS POLICY_NUMBER,
       c.QQNAME                                         AS INSURED_NAME,
       c.DATEOFLOSS                                     AS LOSS_DATE,
       c.BUSINESSNAME                                   AS BUSINESS_NAME,
       c.BRANCHNAME                                     AS BRANCH_NAME,
       c.STATUSWORK                                     AS WORK_STATUS,
       (SELECT s.LSC_NOTE FROM POOLDATA.V_STS_CLAIM s
         WHERE s.LSC_ID = c.STATUSCLAIM)                AS CLAIM_POSITION,
       c.CLOSECLAIMDATE                                 AS CLOSE_DATE,
       c.CLOSECLAIMNOTE                                 AS CLOSE_NOTE,
       c.PICTEKNIK                                      AS TECHNICAL_PIC,
       c.GROUPPANEL                                     AS GROUP_PANEL
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE UPPER(c.CLAIMNO) = :1
    OR UPPER(c.NOPOLIS) = :2
    OR UPPER(c.QQNAME) = :3
 ORDER BY c.CLAIMNO
 FETCH FIRST 100 ROWS ONLY

-- name: search_claim_by_policy
--
-- Calon klaim untuk tipe input "No Polis" —
-- `RDB List/SearchArchiveInsertPolis-SQL.xml`.
--
-- Hanya nomor polis yang dicocokkan; inilah satu-satunya tipe input yang benar-benar
-- mempersempit pencarian. Satu nomor polis wajar membuahkan beberapa klaim.
SELECT c.CLAIMNO                                        AS CLAIM_NUMBER,
       c.NOPOLIS                                        AS POLICY_NUMBER,
       c.QQNAME                                         AS INSURED_NAME,
       c.DATEOFLOSS                                     AS LOSS_DATE,
       c.BUSINESSNAME                                   AS BUSINESS_NAME,
       c.BRANCHNAME                                     AS BRANCH_NAME,
       c.STATUSWORK                                     AS WORK_STATUS,
       (SELECT s.LSC_NOTE FROM POOLDATA.V_STS_CLAIM s
         WHERE s.LSC_ID = c.STATUSCLAIM)                AS CLAIM_POSITION,
       c.CLOSECLAIMDATE                                 AS CLOSE_DATE,
       c.CLOSECLAIMNOTE                                 AS CLOSE_NOTE,
       c.PICTEKNIK                                      AS TECHNICAL_PIC,
       c.GROUPPANEL                                     AS GROUP_PANEL
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE UPPER(c.NOPOLIS) = :1
 ORDER BY c.CLAIMNO
 FETCH FIRST 100 ROWS ONLY

-- name: next_archive_id
--
-- Penomoran ID_ARCHIVE, direplikasi APA ADANYA dari
-- `Database/INSERTDATASFILLINGARCHIVE.prc` atas keputusan Work Owner 2026-09-24.
--
-- Prosedur lama bercabang dua: bila tabelnya kosong ia memakai 1, selain itu
-- `max(ID_ARCHIVE)+1`. `NVL(MAX(...),0)+1` menghasilkan angka yang sama persis pada
-- kedua cabang, sehingga percabangannya tidak perlu ikut ditiru.
--
-- CACAT YANG IKUT TERREPLIKASI, dan ia disebut terang supaya tidak tertimbun: dua
-- penyimpanan yang berjalan bersamaan membaca angka yang SAMA, dan yang kedua menimpa
-- baris yang pertama. Tidak ada galat, tidak ada jejak — hanya satu berkas yang hilang.
-- Usul memperbaikinya dengan sequence ditolak demi kesetaraan `P-5` murni; pertanyaannya
-- tercatat di docs/permintaan-artefak-pega.md.
SELECT NVL(MAX(ID_ARCHIVE), 0) + 1
  FROM POOLDATA.T_CLAIM_ARCHIVE_FILE

-- name: insert_archive
--
-- Penyisipan berkas arsip, menggantikan cabang `insert` prosedur lama.
--
-- TIGA KOLOM YANG DITULIS DI SINI TETAPI TIDAK ADA DI `.prc` EXPORT:
--
--   GROUPPANEL   Ia MENENTUKAN siapa melihat barisnya di daftar kirim ke cabang. Berkas
--                yang lini bisnisnya kosong tidak dapat disaring, dan `.prc` di export
--                memang tidak menulisnya — tetapi pemanggilnya,
--                `RDB List/InsertToClaimArchive-SQL.xml`, MENGIRIMKANNYA sebagai
--                parameter ke-17. Prosedur di export karena itu revisi yang lebih tua
--                daripada pemanggilnya, dan yang diikuti di sini adalah pemanggilnya.
--
--   KODECABANG   Alasan yang sama: parameter ke-16 pada pemanggil, tidak ada di `.prc`.
--                Tidak ada satu pun rule di export yang MEMBACA kolom ini kembali.
--
--   CABANGSTATUS Prosedur lama tidak menulisnya sama sekali, sehingga barisnya bernilai
--                apa pun yang menjadi bawaan kolom. Daftar kirim ke cabang menyaring
--                `CABANGSTATUS = '0'` — bila bawaannya bukan itu, berkas yang baru
--                diarsipkan TIDAK PERNAH muncul di sana. DDL tabelnya tidak ada di export
--                (`R-08`), sehingga bawaannya tidak dapat dipastikan. Menulisnya
--                eksplisit membuat perilakunya sama pada bawaan mana pun.
--
-- TGLINPUT memakai waktu basis data, bukan nilai yang dikirim pemanggil — persis
-- prosedur lama, yang menerima `tTGLINPUT` lalu membuangnya dan memakai `sysdate`. Ini
-- perilaku yang MEMANG BENAR, sejenis dengan tanggal PLA/DLA pada `D-49` butir 7: tanggal
-- input adalah saat berkas tercatat, bukan tanggal yang dipilih seseorang.
INSERT INTO POOLDATA.T_CLAIM_ARCHIVE_FILE
       (ID_ARCHIVE, NOKLAIM, NOPOLIS, TERTANGGUNG, DOL, PICTEKNIK,
        TGLTERIMADOK, TGLINPUT, JUMLAHLEMBAR, TIPEDOK, JENISDOK,
        NAMABOX, KODEFILLING, USERINPUT, KODECABANG, GROUPPANEL, CABANGSTATUS)
VALUES (:1, :2, :3, :4, :5, :6,
        :7, CURRENT_TIMESTAMP, :8, :9, :10,
        :11, :12, :13, :14, :15, '0')

-- name: update_archive
--
-- Pengubahan berkas arsip, menggantikan cabang `update` prosedur lama.
--
-- YANG SENGAJA TIDAK IKUT BERUBAH, persis prosedur lama: USERINPUT, TGLINPUT, KODECABANG,
-- CABANGSTATUS, dan ketiga kolom jawaban layanan Arsip. USERINPUT menyatakan siapa yang
-- MENGARSIPKAN, bukan siapa yang terakhir menyunting; dan menimpa CABANGSTATUS di sini
-- akan mengirim ulang berkas yang sudah sampai ke sistem Arsip hanya karena jumlah
-- lembarnya dibetulkan.
--
-- GROUPPANEL ikut berubah karena ia datang dari klaim yang dipilih, dan mengubah berkas
-- berarti memilih klaimnya lagi.
UPDATE POOLDATA.T_CLAIM_ARCHIVE_FILE
   SET NOKLAIM = :1,
       NOPOLIS = :2,
       TERTANGGUNG = :3,
       DOL = :4,
       PICTEKNIK = :5,
       TGLTERIMADOK = :6,
       JUMLAHLEMBAR = :7,
       TIPEDOK = :8,
       JENISDOK = :9,
       NAMABOX = :10,
       KODEFILLING = :11,
       GROUPPANEL = :12
 WHERE ID_ARCHIVE = :13

-- name: mark_sent
--
-- Penyimpanan jawaban layanan Arsip, menggabungkan DUA pernyataan sistem lama:
--
--   RDB List/UpdateDataArchiveKlaimSetelahService-SQL.xml  KODESERVICE, NOTESERVICE, HITARCHIVE
--   RDB List/SearchDataArchiveFillingCase-SQL.xml pySaveSQL CABANGSTATUS = '1'
--
-- Keduanya disatukan menjadi satu pernyataan supaya tidak ada keadaan di antara: sistem
-- lama dapat berhenti tepat setelah yang pertama, meninggalkan berkas yang jawabannya
-- sudah tersimpan tetapi statusnya masih "belum dikirim" — dan berkas itu akan dikirim
-- lagi.
--
-- TGLKIRIMDOK ikut diisi. Sistem lama TIDAK pernah mengisinya di jalur mana pun yang
-- terbaca di export, padahal grid menampilkannya sebagai "Tanggal Kirim Dok" — kolom yang
-- ditampilkan tetapi selalu kosong hanya memberi tahu pengguna bahwa datanya hilang.
UPDATE POOLDATA.T_CLAIM_ARCHIVE_FILE
   SET KODESERVICE = :1,
       NOTESERVICE = :2,
       HITARCHIVE = :3,
       TGLKIRIMDOK = :4,
       CABANGSTATUS = '1'
 WHERE ID_ARCHIVE = :5

-- name: store_receipt
--
-- Penyimpanan jawaban layanan Arsip TANPA menandai barisnya terkirim — jalur "Save To
-- Archive", yang di sistem lama mengirim berkasnya langsung setelah menyisipkan
-- (`Activity/SaveAttachArchiveToDatabase-Act.xml` langkah 5).
--
-- Ia SENGAJA tidak menyentuh CABANGSTATUS. `SaveAttachArchiveToDatabase` memanggil
-- `SendDataArchiveDOcumentByService`, yang hanya menjalankan
-- `UpdateDataArchiveKlaimSetelahService` — ketiga kolom jawaban, bukan statusnya. Yang
-- menandai `'1'` hanya jalur Dokument Cabang.
--
-- Akibatnya berkas yang sama dikirim DUA KALI, dan itu direplikasi atas keputusan Work
-- Owner 2026-09-25. Bedanya dengan `mark_sent` karena itu bukan kelalaian; menyamakan
-- keduanya akan menghapus pengiriman kedua.
UPDATE POOLDATA.T_CLAIM_ARCHIVE_FILE
   SET KODESERVICE = :1,
       NOTESERVICE = :2,
       HITARCHIVE = :3,
       TGLKIRIMDOK = :4
 WHERE ID_ARCHIVE = :5

-- name: document_types
--
-- Pilihan Tipe Dokumen. Yang ditampilkan adalah STS_PROSES, bukan TYPE_DOCUMENT —
-- itulah kolom yang dibaca `Activity/SearchDataArchiveFilling-Act.xml` saat mengisi nama
-- tipe dokumen pada grid, sehingga dropdown dan grid menyebut hal yang sama.
SELECT ID,
       STS_PROSES
  FROM POOLDATA.V_LST_DOC_TYPE
 ORDER BY STS_PROSES

-- name: document_kinds
--
-- Pilihan Jenis Dokumen beserta tipe pemiliknya.
--
-- INNER JOIN, bukan LEFT: kueri lama pun menggabungkannya dengan INNER JOIN, sehingga
-- jenis dokumen yang tipe pemiliknya sudah tidak ada tidak pernah dapat dipilih.
SELECT d.ID,
       d.DETAIL_DOCUMENT,
       d.DOC_TYPE_ID
  FROM POOLDATA.V_LST_DET_TYPE_DOC d
 INNER JOIN POOLDATA.V_LST_DOC_TYPE t
         ON t.ID = d.DOC_TYPE_ID
 ORDER BY d.DOC_TYPE_ID, d.DETAIL_DOCUMENT

-- name: filling_codes
--
-- Isi pemilih "Pilih Kode".
--
-- Ia REKONSTRUKSI, bukan pembacaan: activity `SetKodeandSearchArchiveDoc` yang mengisi
-- pemilih ini TIDAK ADA di export, dan tidak ada satu pun tabel master kode arsip di
-- seluruh 2.634 berkas (`R-16`). Alasan memilih bentuk ini ada di
-- archivedokumenklaim.FillingCodeOption.
--
-- Kedua penanda diisi pola yang sama: teks pencarian yang sudah dibesarkan hurufnya dan
-- dibungkus tanda persen DI GO, bukan dirangkai ke dalam teks SQL. Pencarian yang kosong
-- dikirim sebagai `%`, sehingga tidak ada cabang kueri kedua yang harus dijaga sejalan.
SELECT KODEFILLING,
       NAMABOX,
       COUNT(*) AS USAGE_COUNT
  FROM POOLDATA.T_CLAIM_ARCHIVE_FILE
 WHERE KODEFILLING IS NOT NULL
   AND (UPPER(KODEFILLING) LIKE :1 ESCAPE '\'
        OR UPPER(NAMABOX) LIKE :2 ESCAPE '\')
 GROUP BY KODEFILLING, NAMABOX
 ORDER BY KODEFILLING, NAMABOX
 FETCH FIRST 200 ROWS ONLY
