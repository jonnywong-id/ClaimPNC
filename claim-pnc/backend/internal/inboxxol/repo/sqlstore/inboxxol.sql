-- Kueri modul Inbox XOL (`MENU_ID 53`, harness `Inbox_XOL_Harness`).
--
-- ============================================================================
-- SELURUHNYA MEMBACA — TIDAK ADA SATU PUN PERNYATAAN YANG MENULIS
-- ============================================================================
--
-- Keputusan Work Owner 2026-09-20. Empat tabel yang ditulis sistem lama tetap dimiliki
-- Pega sepenuhnya selama masa paralel, sejalan dengan `P-1`:
--
--     POOLDATA.XOL_TABLE_ALL_KLAIM   ditulis "INSERT DOL DAN COL"
--     POOLDATA.T_PLA_XOL             ditulis penerbitan dan persetujuan PLA
--     POOLDATA.T_DLA_XOL             ditulis penerbitan dan persetujuan DLA
--     POOLDATA.MST_XOL_PNC           ditulis pengajuan master ke komite
--
-- Berkas ini karena itu tidak boleh memuat INSERT, UPDATE, DELETE, maupun MERGE. Larangan
-- itu DIUJI di query_test.go, bukan sekadar dituliskan di sini.
--
-- ============================================================================
-- ALIAS YANG MENYESATKAN — PEMETAAN LENGKAPNYA
-- ============================================================================
--
-- Kueri sistem lama menamai kolomnya mengikuti properti klipboard Pega yang sudah ada,
-- bukan mengikuti isinya. Akibatnya nama kolom hasil TIDAK berarti seperti bunyinya:
--
--   GetDataXOL_Calulation
--     DOL          as "ASMFull"        → Tanggal Kejadian
--     CAUSEOFLOSS  as "AcceptedNo"     → Penyebab Kerugian
--     SUM(OSVALUE) as "Currency"       → Nilai Outstanding
--     SUM(AKSEPVALUE) as "CurrencyID"  → Nilai Akseptasi
--
--   GetDataXOLPerBusiness
--     count(distinct claimno) as "IsDLA"   → Jumlah Klaim
--     SUM(os_value)    as "DLAShare"       → Nilai Outstanding
--     SUM(aksep_value) as "KlaimAmount"    → Nilai Akseptasi
--     businessgroup note as "Country"      → Nama Group Business
--     businessgroupid  as "IsKirim"        → Kode Group Business
--
--   BrowseAllDataXOL_PLA
--     NO_PLADLAXOL as "CaseID"/"ref_no"    → Nomor PLA/DLA
--     NAMAREAS     as "CoverInsKey"        → Nama Reasuradur
--     NAMALAYER    as "CABANG"             → Nama Layer
--     TAHUN        as "ClaimFrom"          → Tahun XOL
--     KURS         as "PNCSearch"          → Kurs
--     PERCENT      as "ERROR"              → Share Percent
--     EMAIL        as "NOTE"               → Alamat Surel
--     REMARKREAS   as "HASIL5"             → Catatan Reasuradur
--     REMARKAPPROVE as "HASIL2"            → Catatan Persetujuan
--     REMARKPIC    as "AlasanQuotationStock" → Catatan PIC
--     LIMIT_XOL    as "Status"             → Batas Layer
--     STATUSAPPROVE as "SISI"              → Status Persetujuan
--     IDLAYER      as "CARI6"              → Kode Layer
--     IDMASTER     as "pyBPNotes"          → Kode Master XOL
--     USERINPUT    as "LastNoteBy"         → Penerbit
--
--   GetDataMasterXOLForKomiteApprove
--     ID        as "City"        → Kode Master XOL
--     Nama      as "CityID"      → Nama Master XOL
--     TAHUN     as "Country"     → Tahun XOL
--     KURSVALUE as "CountryID"   → Kurs
--     PIC       as "Type"        → Operator Pengaju
--
-- Di berkas ini setiap kolom disebut dengan nama aslinya, lalu dialiaskan ke nama yang
-- menyatakan isinya.
--
-- ============================================================================
-- TIGA HAL YANG SENGAJA BERBEDA DARI SISTEM LAMA
-- ============================================================================
--
--  1. TIDAK ADA PERANGKAIAN SQL. Seluruh kueri XOL lama merangkai klausa FROM dan WHERE
--     dari properti klipboard lalu menyisipkannya mentah dengan `{ASIS:…}` — termasuk
--     NAMA TABEL, yang dipilih dengan merangkai "T_PLA_XOL" atau "T_DLA_XOL" ke dalam
--     teks (`Activity/BrowseDataXOLPLADLAGenerated-Act.xml`). Di sini tabelnya dipilih
--     dengan MEMILIH KUERI, dan setiap nilai menempuh parameter binding.
--
--  2. TIDAK ADA PEMANGGILAN FUNCTION BASIS DATA. `D-02` melarangnya. Dua function yang
--     dipakai jalur ini ditulis ulang di tempatnya:
--
--       GET_GROUPBUSINESS_XOL  → kueri master_business_list + perakitan di Go
--       GETCURRENCYSTANDARD    → subkueri kurs pada breakdown_treaty_inward
--
--  3. KURS MEMAKAI TANGGAL KEJADIAN. `Database/GETCURRENCYSTANDARD.fnc:3` menerima
--     parameter tanggal lalu TIDAK memakainya — yang dipakai `TRUNC(sysdate)` pada baris
--     14. `D-49` butir 4 memutuskan cacat itu diperbaiki, dan perbaikannya diterapkan di
--     sini. Ia akan memunculkan selisih pada uji kesetaraan untuk seluruh data historis
--     valuta asing, dan selisih itu SUDAH disetujui lebih dulu (`D-48`, `ADR-0015`).
--
-- ============================================================================
-- PENANDA /*:ids*/ — DAFTAR PANJANG YANG TETAP TERIKAT PARAMETER
-- ============================================================================
--
-- Tiga kueri menyaring dengan `IN (...)` berisi kode group business, yang jumlahnya
-- berubah menurut perjanjian XOL. Penanda `/*:ids*/` digantikan DERETAN PLACEHOLDER
-- (`:3, :4, :5`) sebelum kueri dikirim — bukan digantikan nilainya.
--
-- Perbedaannya menentukan: yang dirangkai adalah tanda tanya, bukan isi. Tidak ada satu
-- pun karakter dari pengguna yang menyentuh teks SQL. Penggantiannya dikerjakan
-- expandIDs di query.go dan diuji di query_test.go.


-- name: master_list
-- Seluruh perjanjian XOL, satu baris per perjanjian.
--
-- Menggabungkan dua kueri lama yang membaca tabel yang sama dengan alias berbeda:
-- `GetDataMasterXOL` (kelas Data-ClaimData) dan `GetDataMasterXOLForKomiteApprove`.
--
-- # Satu rule yang memang tidak ada di export
--
-- `Activity/GetShowDataMasterXOL-Act.xml` dan `GetClaimXOL` memanggil
-- `ASM-FW-GCNMFW-Data-Adjustment GCNM GetDataMasterXOL` — kelas **Data-Adjustment**.
-- Yang ada di export hanyalah versi kelas **Data-ClaimData**; versi Data-Adjustment
-- termasuk ±242 rule yang hilang (`R-16`).
--
-- Isinya tetap terbaca dari tiga sisi yang saling menguatkan, jadi ia disusun ulang
-- dari bukti — bukan ditebak:
--
--   Section/InboxClaimXOL-Section.xml  kolom grid: Tahun, Kurs, Group Business
--   Activity/GetClaimXOL-Act.xml       .CurrencyName dipakai sebagai TAHUN penyaring,
--                                      .AcceptedNo sebagai pembagi kurs,
--                                      .CurrencyID sebagai daftar kode group business
--   GetDataMasterXOLForKomiteApprove   kolom yang sama, dari tabel yang sama
--
-- Kolom min(LIMIT) dan min(EXCESS) dari MST_XOL_LAYER yang dibawa kueri lama TIDAK ikut:
-- tidak satu pun dari keenam grid layar ini menampilkannya.
SELECT m.ID                AS MASTER_ID,
       m.NAMA              AS MASTER_NAME,
       m.TAHUN             AS YEAR_XOL,
       m.KURSVALUE         AS EXCHANGE_RATE,
       m.TYPEXOL           AS MASTER_TYPE,
       m.STSKOMITE         AS COMMITTEE_STATUS,
       m.REMARKKOMITE      AS COMMITTEE_NOTE,
       m.PIC               AS PIC_OPERATOR,
       m.REMARKPIC         AS PIC_NOTE,
       (SELECT u.EMAIL
          FROM POOLDATA.MST_USER_TEKNIK u
         WHERE UPPER(TRIM(u.OPERATOR_ID)) = UPPER(TRIM(m.PIC))
         FETCH NEXT 1 ROW ONLY) AS PIC_EMAIL
  FROM POOLDATA.MST_XOL_PNC m
 ORDER BY m.ID


-- name: master_pending_committee
-- Perjanjian XOL yang masih menunggu persetujuan komite — grid "DATA MASTER XOL".
--
-- Penyaring `STSKOMITE='0'` dan urutan menurut TAHUN dibawa apa adanya dari
-- `RDB List/GetDataMasterXOLForKomiteApprove-SQL.xml`.
SELECT m.ID                AS MASTER_ID,
       m.NAMA              AS MASTER_NAME,
       m.TAHUN             AS YEAR_XOL,
       m.KURSVALUE         AS EXCHANGE_RATE,
       m.TYPEXOL           AS MASTER_TYPE,
       m.STSKOMITE         AS COMMITTEE_STATUS,
       m.REMARKKOMITE      AS COMMITTEE_NOTE,
       m.PIC               AS PIC_OPERATOR,
       m.REMARKPIC         AS PIC_NOTE,
       (SELECT u.EMAIL
          FROM POOLDATA.MST_USER_TEKNIK u
         WHERE UPPER(TRIM(u.OPERATOR_ID)) = UPPER(TRIM(m.PIC))
         FETCH NEXT 1 ROW ONLY) AS PIC_EMAIL
  FROM POOLDATA.MST_XOL_PNC m
 WHERE TRIM(m.STSKOMITE) = '0'
 ORDER BY m.TAHUN


-- name: master_business_list
-- Group business yang ditanggung setiap perjanjian XOL.
--
-- Ini pengganti `Database/GET_GROUPBUSINESS_XOL.fnc`, yang di sistem lama merakit daftar
-- itu menjadi SATU TEKS — dan untuk tipe 'id' sudah mengutip tiap nilai supaya dapat
-- disisipkan mentah ke dalam `IN (...)`. Merakitnya di Go menghapus keperluan itu.
--
-- # Kenapa join ke MST_XOL_PNC tidak diperlukan
--
-- Cursor pada function menyaring `b.id = m_id AND a.tahun = tahun AND a.id = b.id`.
-- Karena `a.id = b.id` dan tahun yang dikirim selalu tahun master itu sendiri, syarat
-- tahunnya tidak menyaring apa pun. Yang tersisa hanyalah pengelompokan menurut
-- `MST_XOL_BUSINESS.ID` — dan itulah yang dikerjakan di sini, sekali untuk seluruh
-- perjanjian, bukan satu kueri per perjanjian.
--
-- Nama yang NULL dibiarkan NULL. Penggantiannya menjadi "TREATY INWARD" terjadi di Go
-- (inboxxol.BusinessGroup.DisplayName), supaya teks yang dilihat pengguna tidak disusun
-- basis data.
SELECT b.ID          AS MASTER_ID,
       b.IDBUSINESS  AS BUSINESS_GROUP_ID,
       (SELECT g.NOTE
          FROM POOLDATA.BUSINESSGROUP g
         WHERE g.ID = b.IDBUSINESS
         FETCH NEXT 1 ROW ONLY) AS BUSINESS_GROUP_NAME
  FROM POOLDATA.MST_XOL_BUSINESS b
 ORDER BY b.ID, b.IDBUSINESS


-- name: claim_summary
-- Akumulasi klaim satu perjanjian, per Tanggal Kejadian dan Penyebab Kerugian.
--
-- Sumber: `RDB List/GetDataXOL_Calulation-SQL.xml`. Nilainya masih RUPIAH — pembagian
-- dengan kurs perjanjian terjadi di usecase, tempat yang sama dengan sistem lama.
--
-- Penyaring `CAUSEOFLOSS IN (SELECT DESCRIPTION FROM V_D_CAUSE_OF_LOSS)` dibawa apa
-- adanya. Ia membuang baris yang penyebab kerugiannya tidak lagi ada di master — perilaku
-- yang berakibat pada ANGKA, bukan hanya pada tampilan, sehingga menghapusnya akan
-- mengubah total yang dilaporkan.
--
-- ORDER BY ditambahkan; kueri lama tidak punya urutan sama sekali. Hasil tanpa urutan
-- yang ditetapkan berpindah-pindah antar-pemanggilan, dan grid yang barisnya berpindah
-- tanpa sebab terbaca sebagai kerusakan.
SELECT k.DOL              AS LOSS_DATE,
       k.CAUSEOFLOSS      AS CAUSE_OF_LOSS,
       SUM(k.OSVALUE)     AS OUTSTANDING_VALUE,
       SUM(k.AKSEPVALUE)  AS ACCEPTED_VALUE
  FROM POOLDATA.XOL_TABLE_ALL_KLAIM k
 WHERE TO_CHAR(TO_DATE(k.DOL, 'dd/mm/yyyy'), 'yyyy') = :1
   AND k.CAUSEOFLOSS IN (SELECT v.DESCRIPTION FROM POOLDATA.V_D_CAUSE_OF_LOSS v)
   AND k.GROUPBUSINESS IN (/*:ids*/)
 GROUP BY k.DOL, k.CAUSEOFLOSS
 ORDER BY k.DOL, k.CAUSEOFLOSS


-- name: breakdown_business
-- Rincian klaim milik sendiri, per group business, untuk satu Tanggal Kejadian dan satu
-- Penyebab Kerugian.
--
-- Sumber: `RDB List/GetDataXOLPerBusiness-SQL.xml`. Nilainya masih rupiah.
--
-- Kueri lama mengelompokkan menurut `TO_CHAR(dateofloss,…), col_desc, businessgroupid`;
-- kedua yang pertama sudah dipastikan oleh WHERE, sehingga pengelompokan menurut
-- businessgroupid saja menghasilkan baris yang sama persis.
SELECT (SELECT g.NOTE
          FROM POOLDATA.BUSINESSGROUP g
         WHERE g.ID = c.BUSINESSGROUPID
         FETCH NEXT 1 ROW ONLY)   AS BUSINESS_GROUP_NAME,
       c.BUSINESSGROUPID          AS BUSINESS_GROUP_ID,
       COUNT(DISTINCT c.CLAIMNO)  AS CLAIM_COUNT,
       SUM(c.OS_VALUE)            AS OUTSTANDING_VALUE,
       SUM(c.AKSEP_VALUE)         AS ACCEPTED_VALUE
  FROM POOLDATA.T_CLAIM_XOL c
 WHERE TO_CHAR(c.DATEOFLOSS, 'dd/mm/yyyy') = :1
   AND c.COL_DESC = :2
   AND c.BUSINESSGROUPID IN (/*:ids*/)
 GROUP BY c.BUSINESSGROUPID
 ORDER BY c.BUSINESSGROUPID


-- name: breakdown_treaty_inward
-- Rincian klaim treaty inward untuk satu Tanggal Kejadian dan satu Penyebab Kerugian.
--
-- Sumber: `RDB List/GetDataTrytyInwardFromUploadData-SQL.xml`. Satu baris saja —
-- seluruh treaty inward digabung, tanpa pemisahan per group business, karena tabelnya
-- memang tidak punya kolom itu.
--
-- # Nilainya SUDAH dikonversi di sini, dan itu berbeda dari kueri lain
--
-- Setiap baris treaty inward punya mata uangnya sendiri (`CURRENCYID`) yang tidak terbawa
-- ke hasil. Konversinya karena itu wajib terjadi sebelum penjumlahan, bukan sesudahnya.
-- Akibatnya baris ini TIDAK ikut dibagi kurs perjanjian di usecase — membaginya lagi akan
-- mengecilkan nilainya sebesar kurs untuk kedua kalinya.
--
-- # Kurs ditulis ulang, dan memakai tanggal kejadian
--
-- Pengganti `POOLDATA.GETCURRENCYSTANDARD`, yang isinya terbaca di
-- `Database/GETCURRENCYSTANDARD.fnc`: nilai kurs terakhir sebelum sebuah tanggal, dengan
-- CurrencyValue bertanda desimal koma yang harus diubah menjadi titik.
--
-- Dua hal berbeda dari function aslinya, keduanya disengaja:
--
--   a. Tanggalnya TANGGAL KEJADIAN, bukan hari eksekusi. Function menerima parameter
--      tanggal lalu mengabaikannya (`:3` versus `:14`); `D-49` butir 4 memutuskan itu
--      diperbaiki.
--   b. Kurs yang tidak ditemukan menghasilkan NULL, bukan 1. Function mengembalikan `1`
--      pada NO_DATA_FOUND (`:22`), sehingga valuta asing diperlakukan satu banding satu
--      terhadap rupiah tanpa satu pun tanda. `D-49` butir 5 memutuskan itu diperbaiki.
--
-- Butir (b) diterapkan LEBIH SEMPIT daripada bunyi `D-48`. `D-48` menolak TRANSAKSI yang
-- kursnya tidak ada; layar ini tidak bertransaksi, ia meringkas. Menolak seluruh layar
-- karena satu baris historis kehilangan kurs akan menutup data yang lain tanpa sebab.
-- Yang dikerjakan: barisnya ditandai lewat RATE_MISSING, nilainya tidak dikarang, dan
-- layar menyatakan kursnya tidak tersedia. Penyempitan ini dicatat di
-- `keputusan-implementasi.md`, bukan diputuskan diam-diam.
--
-- :1 tanggal kejadian `dd/mm/yyyy`   :2 penyebab kerugian   :3 kode mata uang dasar
SELECT 'Treaty Inward'                                   AS BUSINESS_GROUP_NAME,
       COUNT(DISTINCT x.COMPANY_NAME)                    AS CLAIM_COUNT,
       SUM(x.CLAIM_AMOUNT * x.CLAIM_RATE / x.BASE_RATE)  AS OUTSTANDING_VALUE,
       SUM(x.PAID_SHARE   * x.CLAIM_RATE / x.BASE_RATE)  AS ACCEPTED_VALUE,
       MAX(CASE WHEN x.CLAIM_RATE IS NULL
                  OR x.BASE_RATE IS NULL
                  OR x.BASE_RATE = 0
                THEN 1 ELSE 0 END)                       AS RATE_MISSING
  FROM (SELECT i.COMPANYNAME AS COMPANY_NAME,
               CAST(REPLACE(REPLACE(i.CLAIMAMOUNT, '.', ''), ',', '.') AS NUMERIC)
                   AS CLAIM_AMOUNT,
               CAST(REPLACE(REPLACE(i.PAIDCLAIMAMOUNTSHARE, '.', ''), ',', '.') AS NUMERIC)
                   AS PAID_SHARE,
               (SELECT CAST(REPLACE(r.CURRENCYVALUE, ',', '.') AS NUMERIC)
                  FROM POOLDATA.M_CURRENCYSTANDARD r
                 WHERE r.ID = i.CURRENCYID
                   AND CAST(r.CURRENCYDATE AS DATE)
                       <= CAST(TO_DATE(i.DATEOFLOSS, 'dd/mm/yyyy') AS DATE)
                 ORDER BY r.CURRENCYDATE DESC
                 FETCH NEXT 1 ROW ONLY) AS CLAIM_RATE,
               (SELECT CAST(REPLACE(r.CURRENCYVALUE, ',', '.') AS NUMERIC)
                  FROM POOLDATA.M_CURRENCYSTANDARD r
                 WHERE r.ID = :3
                   AND CAST(r.CURRENCYDATE AS DATE)
                       <= CAST(TO_DATE(i.DATEOFLOSS, 'dd/mm/yyyy') AS DATE)
                 ORDER BY r.CURRENCYDATE DESC
                 FETCH NEXT 1 ROW ONLY) AS BASE_RATE
          FROM POOLDATA.T_CLAIM_INWARD_XOL i
         WHERE TO_CHAR(TO_DATE(i.DATEOFLOSS, 'dd/mm/yyyy'), 'dd/mm/yyyy') = :1
           AND i.CAUSEOFLOSS = :2) x


-- name: advice_list_pla
-- Pemberitahuan PLA yang sudah diterbitkan untuk satu tahun dan satu penyebab kerugian.
--
-- Sumber: `RDB List/BrowseAllDataXOL_PLA-SQL.xml`, yang di sistem lama membaca T_PLA_XOL
-- atau T_DLA_XOL tergantung teks yang dirangkai pemanggilnya. Di sini keduanya menjadi
-- DUA kueri, sehingga nama tabel tidak pernah berasal dari luar.
--
-- Tiga perbedaan dari kueri lama, seluruhnya menyangkut TEMPAT pekerjaan dikerjakan,
-- bukan hasilnya:
--
--   a. `PERCENT || ' %'` tidak dirangkai di SQL — tanda persen ditambahkan saat
--      ditampilkan. Nilai yang bersatuan tidak dapat dijumlahkan maupun diurutkan.
--   b. `CASE REVISI WHEN '0' THEN … ELSE … || ' / ' || REVISI END` tidak dirakit di SQL;
--      nomor dan revisinya dibawa terpisah (`08-TECHNICAL-STRATEGY.md` §4.3).
--   c. `CASE WHEN Email IS NULL THEN … ELSE Email END` menjadi COALESCE, mengikuti
--      padanan portabel pada `09-DATABASE-STRATEGY.md` §4.
--
-- Satu hal yang TIDAK dibawa sama sekali: `Activity/BrowseDataXOLPLADLAGenerated-Act.xml`
-- menimpa kolom surel dengan satu alamat tetap apabila operator yang membuka layar
-- bernama tertentu. Itu hardcode identitas di jalur produksi (`D-15`), dan `D-67`
-- menetapkan tidak satu pun akun pribadi dibawa ke sistem baru.
--
-- Urutan `REVISI, IDLAYER` dibawa apa adanya dari kueri lama.
SELECT a.NO_PLADLAXOL   AS ADVICE_NUMBER,
       a.REVISI         AS REVISION,
       a.IDREAS         AS REINSURER_ID,
       a.NAMAREAS       AS REINSURER_NAME,
       a.IDLAYER        AS LAYER_ID,
       a.NAMALAYER      AS LAYER_NAME,
       a.TAHUN          AS YEAR_XOL,
       a.CAUSEOFLOSS    AS CAUSE_OF_LOSS,
       a.KURS           AS EXCHANGE_RATE,
       a.PERCENT        AS SHARE_PERCENT,
       a.LIMIT_XOL      AS LAYER_LIMIT,
       a.STATUSAPPROVE  AS APPROVAL_STATUS,
       a.IDMASTER       AS MASTER_ID,
       a.USERINPUT      AS INPUT_BY,
       a.REMARKREAS     AS REINSURER_NOTE,
       a.REMARKAPPROVE  AS APPROVAL_NOTE,
       a.REMARKPIC      AS PIC_NOTE,
       COALESCE(a.EMAIL,
                (SELECT r.EMAIL
                   FROM POOLDATA.T_REINSURER r
                  WHERE r.REINSURERID = a.IDREAS
                  FETCH NEXT 1 ROW ONLY)) AS REINSURER_EMAIL,
       (SELECT r.COUNTRY
          FROM POOLDATA.T_REINSURER r
         WHERE r.REINSURERID = a.IDREAS
         FETCH NEXT 1 ROW ONLY)           AS REINSURER_COUNTRY,
       (SELECT u.EMAIL
          FROM POOLDATA.MST_USER_TEKNIK u
         WHERE UPPER(TRIM(u.OPERATOR_ID)) = UPPER(TRIM(a.USERINPUT))
         FETCH NEXT 1 ROW ONLY)           AS INPUT_BY_EMAIL,
       TO_CHAR(a.TGLINSERT, 'dd/mm/yyyy') AS ISSUED_ON
  FROM POOLDATA.T_PLA_XOL a
 WHERE a.TAHUN = :1
   AND a.CAUSEOFLOSS = :2
 ORDER BY a.REVISI, a.IDLAYER


-- name: advice_list_dla
-- Pemberitahuan DLA yang sudah diterbitkan. Kembar dengan advice_list_pla; yang berbeda
-- HANYA tabelnya.
--
-- Kembarannya disengaja dan tidak dirangkai menjadi satu kueri berparameter tabel:
-- nama tabel tidak dapat menempuh parameter binding, dan merangkainya berarti membuka
-- kembali persis celah yang berkas ini tutup. Kesesuaian kedua kueri DIUJI di
-- query_test.go, sehingga salah satu yang berubah sendirian akan tertangkap.
SELECT a.NO_PLADLAXOL   AS ADVICE_NUMBER,
       a.REVISI         AS REVISION,
       a.IDREAS         AS REINSURER_ID,
       a.NAMAREAS       AS REINSURER_NAME,
       a.IDLAYER        AS LAYER_ID,
       a.NAMALAYER      AS LAYER_NAME,
       a.TAHUN          AS YEAR_XOL,
       a.CAUSEOFLOSS    AS CAUSE_OF_LOSS,
       a.KURS           AS EXCHANGE_RATE,
       a.PERCENT        AS SHARE_PERCENT,
       a.LIMIT_XOL      AS LAYER_LIMIT,
       a.STATUSAPPROVE  AS APPROVAL_STATUS,
       a.IDMASTER       AS MASTER_ID,
       a.USERINPUT      AS INPUT_BY,
       a.REMARKREAS     AS REINSURER_NOTE,
       a.REMARKAPPROVE  AS APPROVAL_NOTE,
       a.REMARKPIC      AS PIC_NOTE,
       COALESCE(a.EMAIL,
                (SELECT r.EMAIL
                   FROM POOLDATA.T_REINSURER r
                  WHERE r.REINSURERID = a.IDREAS
                  FETCH NEXT 1 ROW ONLY)) AS REINSURER_EMAIL,
       (SELECT r.COUNTRY
          FROM POOLDATA.T_REINSURER r
         WHERE r.REINSURERID = a.IDREAS
         FETCH NEXT 1 ROW ONLY)           AS REINSURER_COUNTRY,
       (SELECT u.EMAIL
          FROM POOLDATA.MST_USER_TEKNIK u
         WHERE UPPER(TRIM(u.OPERATOR_ID)) = UPPER(TRIM(a.USERINPUT))
         FETCH NEXT 1 ROW ONLY)           AS INPUT_BY_EMAIL,
       TO_CHAR(a.TGLINSERT, 'dd/mm/yyyy') AS ISSUED_ON
  FROM POOLDATA.T_DLA_XOL a
 WHERE a.TAHUN = :1
   AND a.CAUSEOFLOSS = :2
 ORDER BY a.REVISI, a.IDLAYER


-- name: approval_advice_queue
-- Antrean persetujuan pemberitahuan pada tab Komite.
--
-- Sumber: `RDB List/GetDataXOLForKomiteApprove-SQL.xml` apa adanya, termasuk UNION-nya.
-- Satu baris mewakili SEKUMPULAN pemberitahuan — satu tahun × satu penyebab kerugian ×
-- satu tipe — bukan satu pemberitahuan.
--
-- # Satu cacat yang direplikasi
--
-- `MAX(TO_CHAR(TGLINSERT,'dd/mm/yyyy'))` mengambil teks terbesar, bukan tanggal terbaru.
-- Pada format `dd/mm/yyyy` keduanya berbeda: `31/01/2024` lebih besar daripada
-- `01/12/2024` sebagai teks. `P-5` menetapkan perilaku dipertahankan lebih dulu, dan
-- cacat ini tidak termasuk 13 butir perbaikan eksplisit `D-49` — karena itu dibawa apa
-- adanya dan dicatat, bukan diperbaiki sepihak.
--
-- Alias `q` pada subkueri wajib: Oracle mengizinkan subkueri FROM tanpa alias,
-- PostgreSQL tidak (`D-20`, satu set SQL untuk keduanya).
SELECT q.YEAR_XOL,
       q.CAUSE_OF_LOSS,
       q.ADVICE_TYPE,
       q.LAST_INSERTED
  FROM (SELECT d.TAHUN       AS YEAR_XOL,
               d.CAUSEOFLOSS AS CAUSE_OF_LOSS,
               'DLA'         AS ADVICE_TYPE,
               MAX(TO_CHAR(d.TGLINSERT, 'dd/mm/yyyy')) AS LAST_INSERTED
          FROM POOLDATA.T_DLA_XOL d
         WHERE TRIM(d.STATUSAPPROVE) = '0'
         GROUP BY d.TAHUN, d.CAUSEOFLOSS
         UNION
        SELECT p.TAHUN,
               p.CAUSEOFLOSS,
               'PLA',
               MAX(TO_CHAR(p.TGLINSERT, 'dd/mm/yyyy'))
          FROM POOLDATA.T_PLA_XOL p
         WHERE TRIM(p.STATUSAPPROVE) = '0'
         GROUP BY p.TAHUN, p.CAUSEOFLOSS) q
 ORDER BY q.ADVICE_TYPE, q.YEAR_XOL DESC


-- name: cause_of_loss_list
-- Daftar Penyebab Kerugian untuk dropdown pada modal "Tambah Data DOL dan COL".
--
-- Sumber: report definition `SelectVDCauseOfLoss_RD` pada kelas
-- `ASM-FW-GCNMFW-Int-V_D_CAUSE_OF_LOSS`, yang menampilkan `.DESCRIPTION` dan menyimpan
-- `.D_COL_ID`.
--
-- Penyaring `.D_COL_ID = Param.id` pada report definition TIDAK dibawa: ia dipakai saat
-- report yang sama mencari satu baris tertentu, sedangkan dropdown membutuhkan seluruh
-- daftar. Tidak ada penyaring status aktif di report definition itu, dan tidak
-- ditambahkan di sini — menambahkan saringan yang tidak ada di sistem lama akan
-- menghilangkan pilihan yang hari ini masih dapat dipilih.
--
-- Yang tersimpan di kolom CAUSEOFLOSS pada tabel klaim XOL adalah DESKRIPSI-nya, bukan
-- kodenya. Itu terbaca dari claim_summary, yang menyaring
-- `CAUSEOFLOSS IN (SELECT DESCRIPTION …)`.
SELECT v.D_COL_ID    AS CAUSE_OF_LOSS_ID,
       v.DESCRIPTION AS CAUSE_OF_LOSS_DESC
  FROM POOLDATA.V_D_CAUSE_OF_LOSS v
 ORDER BY v.DESCRIPTION
