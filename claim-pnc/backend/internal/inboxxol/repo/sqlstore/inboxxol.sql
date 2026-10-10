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
-- # Koreksi: min(LIMIT) TERNYATA ditampilkan
--
-- Catatan sebelumnya di sini menyatakan `min(LIMIT)` dan `min(EXCESS)` dari
-- `MST_XOL_LAYER` tidak dipakai grid mana pun. Itu KELIRU: grid di layar rincian
-- menampilkan keduanya sebagai "Min Limit" dan "Min Limit IDR", persis seperti alias
-- `AIDiterima` dan `ClaimAmount` pada `RDB List/GetDataMasterXOL-SQL.xml:11`.
--
-- `MIN_LIMIT` karena itu dibawa. `min(EXCESS)` tetap tidak dibawa — kueri lama memang
-- mengambilnya (alias `KategoriKronologi`), tetapi tidak satu pun kolom layar
-- menampilkannya.
--
-- "Min Limit IDR" TIDAK dihitung di sini. Kueri lama merangkainya sebagai
-- `min(limit)*A.KURSVALUE`; perkaliannya dipindahkan ke Go, tempat kurs sudah ada —
-- satu nilai turunan yang dihitung dua kali di dua lapisan akan menyimpang diam-diam.
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
         FETCH NEXT 1 ROW ONLY) AS PIC_EMAIL,
       (SELECT MIN(l.LIMIT)
          FROM POOLDATA.MST_XOL_LAYER l
         WHERE l.ID = m.ID) AS MIN_LIMIT
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
         FETCH NEXT 1 ROW ONLY) AS PIC_EMAIL,
       (SELECT MIN(l.LIMIT)
          FROM POOLDATA.MST_XOL_LAYER l
         WHERE l.ID = m.ID) AS MIN_LIMIT
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

-- name: summary_business
-- Grid "Summary Data XOL" pada layar rincian — satu baris per group business yang
-- menanggung klaim pada tanggal kejadian dan penyebab kerugian itu.
--
-- Sumber: `RDB List/GetBusinessnameXOLForSummerry-SQL.xml`, dipanggil
-- `Activity/SummaryXOLBeforeGeneratedKlaim-Act.xml` untuk mengisi page `BusinessXOLL`.
--
-- # Dua sumber disatukan UNION, bukan dijumlahkan
--
-- Klaim milik sendiri datang dari `T_CLAIM_XOL`, klaim treaty inward dari
-- `T_CLAIM_INWARD_XOL`. Keduanya tidak punya group business yang sebanding — yang kedua
-- diberi nama tetap "Treaty Inward" oleh kueri lama, dan itu dipertahankan.
--
-- `DATEOFLOSS` kedua tabel bertipe BERBEDA: pada `T_CLAIM_XOL` ia tanggal, pada
-- `T_CLAIM_INWARD_XOL` ia teks `dd/mm/yyyy`. Perlakuannya karena itu juga berbeda —
-- sama seperti pada `breakdown_treaty_inward`.
--
-- `ROWNUM = 1` pada anak kueri nama diganti `FETCH NEXT 1 ROW ONLY` (`DB-3`); keduanya
-- memilih satu baris sembarang, dan kueri lama memang tidak menetapkan baris yang mana.
--
-- ORDER BY ditambahkan; kueri lama tidak punya urutan sama sekali. Grid yang barisnya
-- berpindah tanpa sebab terbaca sebagai kerusakan.
--
-- :1 tanggal kejadian `dd/mm/yyyy`   :2 penyebab kerugian
SELECT b.BUSINESS_GROUP_ID, b.BUSINESS_GROUP_NAME
  FROM (SELECT x.BUSINESSGROUPID AS BUSINESS_GROUP_ID,
               (SELECT g.BUSINESSGROUPNAME
                  FROM POOLDATA.BUSINESS g
                 WHERE g.BUSINESSGROUPID = x.BUSINESSGROUPID
                 FETCH NEXT 1 ROW ONLY) AS BUSINESS_GROUP_NAME
          FROM POOLDATA.T_CLAIM_XOL x
         WHERE TO_CHAR(x.DATEOFLOSS, 'dd/mm/yyyy') = :1
           AND x.COL_DESC = :2
         GROUP BY x.BUSINESSGROUPID
        UNION
        SELECT a.BUSINESSID AS BUSINESS_GROUP_ID,
               'Treaty Inward' AS BUSINESS_GROUP_NAME
          FROM POOLDATA.T_CLAIM_INWARD_XOL a
         WHERE TO_CHAR(TO_DATE(a.DATEOFLOSS, 'dd/mm/yyyy'), 'dd/mm/yyyy') = :1
           AND a.CAUSEOFLOSS = :2
         GROUP BY a.BUSINESSID) b
 ORDER BY b.BUSINESS_GROUP_NAME, b.BUSINESS_GROUP_ID


-- name: claim_list
-- Grid "No Klaim · Os Value · Accept Value · Currency" pada layar rincian.
--
-- Sumber: `RDB List/BrowserT_claim_xolDesc-SQL.xml`, dipanggil
-- `Activity/ShowDataKlaimXOLKlaimBeforeGenerated-Act.xml` — activity di balik tombol
-- "Pilih" DAN "Show All Data" — untuk mengisi page `TempAllData`.
--
-- # Empat kolom dibawa, enam ditinggalkan
--
-- Kueri lama mengembalikan sepuluh kolom; grid menampilkan empat. Yang ditinggalkan dan
-- alasannya:
--
--	rownum "City"		nomor baris; digambar layar, bukan dibaca dari basis data
--	salvage_value		tidak ada kolomnya di grid, dan ia menyeret T_SALVAGE_MBU
--	LBU_ID			tidak ada kolomnya, dan ia menyeret DB LINK `@asmd`
--				(`D-25`, `R-03`) — satu-satunya DB Link di seluruh modul ini
--	BusinessID/Name		tidak ada kolomnya di grid
--	IDCURRENCY		kode mata uang; yang ditampilkan NAMANYA
--
-- Membawanya hanya untuk dibuang berarti modul ini bergantung pada DB Link yang sudah
-- diputuskan diganti API, demi kolom yang tidak pernah dilihat siapa pun.
--
-- # Dua sumber, dua perlakuan nilai
--
-- Klaim sendiri (`T_CLAIM_XOL`) sudah bernilai rupiah dan tinggal dikalikan share.
-- Treaty inward (`T_CLAIM_INWARD_XOL`) menyimpan nilainya sebagai TEKS bergaya Indonesia
-- dan bermata uang sendiri, sehingga dikonversi lewat `M_CURRENCYSTANDARD` — pola yang
-- sama persis dengan `breakdown_treaty_inward`, termasuk alasannya: `GETCURRENCYSTANDARD`
-- mengembalikan `1` saat kurs tidak ditemukan (`D-48`), dan nilai yang salah tetapi tampak
-- benar lebih berbahaya daripada nilai yang hilang.
--
-- Baris treaty inward memakai COMPANYNAME sebagai "No Klaim" — begitulah kueri lama
-- (`A.COMPANYNAME as CLAIMNO`), karena klaim inward tidak punya nomor klaim ASM.
--
-- :1 tanggal kejadian `dd/mm/yyyy`   :2 penyebab kerugian   :3 kode mata uang dasar
SELECT x.CLAIM_NO, x.CURRENCY_NAME, x.SOURCE,
       x.OUTSTANDING_VALUE, x.ACCEPTED_VALUE, x.RATE_MISSING
  FROM (SELECT k.CLAIMNO AS CLAIM_NO,
               (SELECT c.CURRENCY
                  FROM POOLDATA.CURRENCY c
                 WHERE c.ID = k.CURRENCY
                 FETCH NEXT 1 ROW ONLY) AS CURRENCY_NAME,
               'bisnis' AS SOURCE,
               SUM(k.OS_VALUE    * k.CLAIM_OR * k.CLAIM_SHARE_ASM) AS OUTSTANDING_VALUE,
               SUM(k.AKSEP_VALUE * k.CLAIM_OR * k.CLAIM_SHARE_ASM) AS ACCEPTED_VALUE,
               0 AS RATE_MISSING
          FROM POOLDATA.T_CLAIM_XOL k
         WHERE TO_CHAR(k.DATEOFLOSS, 'dd/mm/yyyy') = :1
           AND k.COL_DESC = :2
         GROUP BY k.CLAIMNO, k.CURRENCY
        UNION ALL
        SELECT t.COMPANY_NAME AS CLAIM_NO,
               t.CURRENCY_NAME,
               'treaty' AS SOURCE,
               SUM(t.CLAIM_AMOUNT * t.CLAIM_RATE / t.BASE_RATE) AS OUTSTANDING_VALUE,
               SUM(t.PAID_SHARE   * t.CLAIM_RATE / t.BASE_RATE) AS ACCEPTED_VALUE,
               MAX(CASE WHEN t.CLAIM_RATE IS NULL
                          OR t.BASE_RATE IS NULL
                          OR t.BASE_RATE = 0
                        THEN 1 ELSE 0 END) AS RATE_MISSING
          FROM (SELECT i.COMPANYNAME AS COMPANY_NAME,
                       (SELECT c.CURRENCY
                          FROM POOLDATA.CURRENCY c
                         WHERE c.ID = i.CURRENCYID
                         FETCH NEXT 1 ROW ONLY) AS CURRENCY_NAME,
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
                   AND i.CAUSEOFLOSS = :2) t
         GROUP BY t.COMPANY_NAME, t.CURRENCY_NAME) x
 ORDER BY x.SOURCE, x.CLAIM_NO


-- name: export_per_business
-- Isi "Export to Excel" untuk satu group business biasa — bukan MBU, bukan treaty inward.
--
-- Sumber: `RDB List/ExportDetailXOLPerBiz-SQL.xml`, dipanggil
-- `Activity/GenerateDetailClaimBusinessXOL-Act.xml` lalu diubah menjadi berkas oleh
-- `pxConvertResultsToCSV`. Jadi "Excel" pada label tombolnya sebenarnya CSV — sejalan
-- dengan `D-11`, dan dengan pola unduhan yang sudah dipakai panel PLA/DLA.
--
-- # Alias Pega DIPERTAHANKAN
--
-- Nama kolomnya tetap `"AlasanKlaim"`, `"Keyword"`, `"District"` dan seterusnya — nama
-- properti Pega yang sebagian tidak ada hubungannya dengan isinya. Itu disengaja: urutan
-- dan judul kolom CSV ditentukan parameter `CSVProperties` dan `CSVPropHeaders` pada
-- activity, dan keduanya menyebut properti dengan nama itu. Menamai ulang di sini berarti
-- memutus satu-satunya rujukan yang membuat pemetaan 42 kolom dapat diperiksa.
--
-- Nama yang dibaca PENGGUNA tetap benar: ia datang dari `CSVPropHeaders` (INSURED,
-- RISK LOCATION, ASM POLICY, …), bukan dari alias ini.
--
-- # Dua penyesuaian portabilitas
--
--	ROWNUM = 1	→ FETCH NEXT 1 ROW ONLY	(`DB-3`)
--	{TempExport.*}	→ :1 :2 :3		parameter binding, bukan perangkaian teks
--
-- `TO_CHAR(dateofloss,'dd/mm/yyyy')` DIBIARKAN: ia sah di Oracle maupun PostgreSQL, dan
-- format tanggalnya adalah bagian dari bentuk berkas yang diterima pengguna.
--
-- :1 tanggal kejadian `dd/mm/yyyy`   :2 penyebab kerugian   :3 kode group business
SELECT a.picteknik AS "UserTeknis",
       a.claimno AS "ClaimNo",
       qqname AS "AlasanKlaim",
       b.risk_loc_klaim AS "Location",
       a.nopolis AS "PolicyNo",
       TO_CHAR (a.dateofloss, 'dd/mm/yyyy') AS "AnalystDoctorRemaks",
       a.coinsname AS "AllBusinessFlag",
       (SELECT currency
          FROM pooldata.currency
         WHERE id = z.currency)
          AS "Currency",
       CASE WHEN A.STATUSCLAIM = '1143' THEN z.ACCEPTED ELSE z.RESERVE END
          AS "Keyword",
       z.ACCEPTED AS "CauseOfLossID",
       CASE
          WHEN A.STATUSCLAIM = '1143' THEN z.ASM_SHARE
          ELSE z.ASM_SHARE_OS
       END
          AS "ClaimAmount",
       z.ASM_SHARE_PRSN_OS AS "ASMShare",
       CASE WHEN A.STATUSCLAIM = '1143' THEN z.OR_AKSEP ELSE z.OR_OS END
          AS "BranchID",
       CASE WHEN A.STATUSCLAIM = '1143' THEN z.BPPDAN ELSE z.BPPDAN_OS END
          AS "BranchName",
       CASE WHEN A.STATUSCLAIM = '1143' THEN z.FACOUT ELSE z.FACOUT_OS END
          AS "BusinessID",
       CASE WHEN A.STATUSCLAIM = '1143' THEN z.FSPL ELSE z.FSPL_OS END
          AS "BusinessName",
       CASE WHEN A.STATUSCLAIM = '1143' THEN z.PSPL ELSE z.PSPL_OS END
          AS "CASEDB",
       CASE WHEN A.STATUSCLAIM = '1143' THEN z.AKSEP_QS ELSE z.OS_QS END
          AS "CaseID",
       z.ASM_SHARE AS "SIM",
       z.OR_AKSEP AS "City",
       z.BPPDAN AS "CityID",
       z.FACOUT AS "ClaimEstimate",
       z.FSPL AS "ClaimID",
       z.PSPL AS "ClaimNoSRB",
       z.AKSEP_QS AS "ClientID",
       z.SALVAGE AS "District",
       a.LEADER_MEMBER AS "EmailTertanggung",
       a.SOBNAME AS "FlagASO",
       b.OCCUPATION AS "IsPLA",
       CASE
          WHEN A.STATUSCLAIM = '1143' THEN 0
          ELSE z.RESERVE - z.ACCEPTED
       END
          AS "ClientName",
       CASE
          WHEN A.STATUSCLAIM = '1143' THEN 0
          ELSE (z.RESERVE - z.ACCEPTED) * z.CLAIM_OR
       END
          AS "CloseClaimNote",
       CASE
          WHEN A.STATUSCLAIM = '1143' THEN 0
          ELSE (z.RESERVE - z.ACCEPTED) * z.BPPDAN_PRSN
       END
          AS "ConsultantID",
       CASE
          WHEN A.STATUSCLAIM = '1143' THEN 0
          ELSE (z.RESERVE - z.ACCEPTED) * z.FACOUT_PRSN
       END
          AS "ConsultantName",
       CASE
          WHEN A.STATUSCLAIM = '1143' THEN 0
          ELSE (z.RESERVE - z.ACCEPTED) * z.FSPL_PRSN
       END
          AS "Conveyance",
       CASE
          WHEN A.STATUSCLAIM = '1143' THEN 0
          ELSE (z.RESERVE - z.ACCEPTED) * z.PSPL_PRSN
       END
          AS "ContractNo",
       CASE
          WHEN A.STATUSCLAIM = '1143' THEN 0
          ELSE (z.RESERVE - z.ACCEPTED) * z.QSRI_PRSN
       END
          AS "Country",
       z.ADJUSTER AS "Email",
       (SELECT SURVEYOR_NAME
          FROM pooldata.t_surveyorlist
         WHERE pnccaseid = a.claimid
         FETCH NEXT 1 ROW ONLY)
          AS "IDMaster",
       CASE
          when (select ISPENDINGCLOSE from DATAPEGA.PC_ASM_FW_GCNMFW_WORK where pyid=A.CLAIMNO)='true' then
          (REPLACE (
                (SELECT G.KETERANGAN
                   FROM POOLDATA.GCNM_PROGRESS_CLAIM G
                  WHERE     G.PNCCASEID = a.claimno
                        AND ID_UPDATE = (SELECT MAX (ID_UPDATE)
                                           FROM POOLDATA.GCNM_PROGRESS_CLAIM
                                          WHERE PNCCASEID = a.claimno)),
                '\\n',
                ''))
          WHEN A.STATUSCLAIM = '1143'
          THEN
             'Closed'
          ELSE
             REPLACE (
                (SELECT G.KETERANGAN
                   FROM POOLDATA.GCNM_PROGRESS_CLAIM G
                  WHERE     G.PNCCASEID = a.claimno
                        AND ID_UPDATE = (SELECT MAX (ID_UPDATE)
                                           FROM POOLDATA.GCNM_PROGRESS_CLAIM
                                          WHERE PNCCASEID = a.claimno)
                        AND STATUS_PROGRESS2 NOT IN ('2', '24', '60', '59')),
                '\\n',
                '')
       END
          AS "Remark",
       CASE
        when (select X.PYSTATUSWORK from DATAPEGA.PC_ASM_FW_GCNMFW_WORK x where pyid=A.CLAIMNO) in ('New','Open') then
          'Outstanding'
          WHEN A.STATUSCLAIM = '1143'
          THEN
             'Closed'
          ELSE
             (SELECT I.STS_PROGRESS1
                FROM POOLDATA.GCNM_PROGRESS_CLAIM G,
                     POOLDATA.GCNM_MST_PROGRESS_KLAIM I
               WHERE     G.STATUS_PROGRESS1 = I.ID_PROGRESS
                     AND G.PNCCASEID = a.claimno
                     AND ID_UPDATE =
                            (SELECT MAX (ID_UPDATE)
                               FROM POOLDATA.GCNM_PROGRESS_CLAIM
                              WHERE     PNCCASEID = a.claimno
                                    AND STATUS_PROGRESS2 NOT IN
                                           ('2', '24', '60', '59')))
       END
          AS "ReporterName"
  FROM pooldata.t_claim_pnc a,
       pooldata.pega_dashboardpnc b,
       (  SELECT claimno,
                 currency,
                 MAX (claim_or) AS CLAIM_OR,
                 MAX (bppdan) AS BPPDAN_PRSN,
                 MAX (facout) AS FACOUT_PRSN,
                 MAX (fspl) AS FSPL_PRSN,
                 MAX (pspl) AS PSPL_PRSN,
                 MAX (qsri) AS QSRI_PRSN,
                 MAX (claim_share_asm) * 100 AS ASM_SHARE_PRSN_OS,
                 SUM (
                    CASE
                       WHEN paymenttype = '3' OR paymenttype = '4' THEN 0
                       ELSE os_value
                    END)
                    AS RESERVE,
                 SUM (
                    CASE
                       WHEN paymenttype = '3' OR paymenttype = '4' THEN 0
                       ELSE aksep_value
                    END)
                    AS ACCEPTED,
                   MAX (claim_share_asm)
                 * SUM (
                      CASE
                         WHEN paymenttype = '3' OR paymenttype = '4' THEN 0
                         ELSE os_value
                      END)
                    AS ASM_SHARE_OS,
                   MAX (claim_share_asm)
                 * SUM (
                      CASE
                         WHEN paymenttype = '3' OR paymenttype = '4' THEN 0
                         ELSE aksep_value
                      END)
                    AS ASM_SHARE,
                   MAX (claim_or)
                 * SUM (
                      CASE
                         WHEN paymenttype != '3' AND paymenttype != '4' THEN os_value
                         ELSE 0
                      END)
                    AS OR_OS,
                   MAX (bppdan)
                 * SUM (
                      CASE
                         WHEN paymenttype = '3' OR paymenttype = '4' THEN 0
                         ELSE os_value
                      END)
                    AS BPPDAN_OS,
                   MAX (facout)
                 * MAX (CLAIM_SHARE_ASM)
                 * SUM (
                      CASE
                         WHEN paymenttype = '3' OR paymenttype = '4' THEN 0
                         ELSE os_value
                      END)
                    AS FACOUT_OS,
                   MAX (fspl)
                 * MAX (CLAIM_SHARE_ASM)
                 * SUM (
                      CASE
                         WHEN paymenttype = '3' OR paymenttype = '4' THEN 0
                         ELSE os_value
                      END)
                    AS FSPL_OS,
                   MAX (pspl)
                 * MAX (CLAIM_SHARE_ASM)
                 * SUM (
                      CASE
                         WHEN paymenttype = '3' OR paymenttype = '4' THEN 0
                         ELSE os_value
                      END)
                    AS PSPL_OS,
                   MAX (qsri)
                 * MAX (CLAIM_SHARE_ASM)
                 * SUM (
                      CASE
                         WHEN paymenttype = '3' OR paymenttype = '4' THEN 0
                         ELSE os_value
                      END)
                    AS OS_QS,
                   MAX (claim_or)
                 * MAX (CLAIM_SHARE_ASM)
                 * SUM (
                      CASE
                         WHEN paymenttype = '3' OR paymenttype = '4' THEN 0
                         ELSE aksep_value
                      END)
                    AS OR_AKSEP,
                   MAX (bppdan)
                 * SUM (
                      CASE
                         WHEN paymenttype = '3' OR paymenttype = '4' THEN 0
                         ELSE aksep_value
                      END)
                    AS BPPDAN,
                   MAX (facout)
                 * MAX (CLAIM_SHARE_ASM)
                 * SUM (
                      CASE
                         WHEN paymenttype = '3' OR paymenttype = '4' THEN 0
                         ELSE aksep_value
                      END)
                    AS FACOUT,
                   MAX (fspl)
                 * MAX (CLAIM_SHARE_ASM)
                 * SUM (
                      CASE
                         WHEN paymenttype = '3' OR paymenttype = '4' THEN 0
                         ELSE aksep_value
                      END)
                    AS FSPL,
                   MAX (pspl)
                 * MAX (CLAIM_SHARE_ASM)
                 * SUM (
                      CASE
                         WHEN paymenttype = '3' OR paymenttype = '4' THEN 0
                         ELSE aksep_value
                      END)
                    AS PSPL,
                   MAX (qsri)
                 * MAX (CLAIM_SHARE_ASM)
                 * SUM (
                      CASE
                         WHEN paymenttype = '3' OR paymenttype = '4' THEN 0
                         ELSE aksep_value
                      END)
                    AS AKSEP_QS,
                   MAX (claim_or)/100
                 * SUM (
                      CASE
                         WHEN paymenttype = '3' THEN aksep_value * -1
                         ELSE 0
                      END)
                    AS SALVAGE,
                   MAX (claim_or)
                 * MAX (CLAIM_SHARE_ASM)
                 * SUM (
                      CASE WHEN paymenttype = '4' THEN aksep_value ELSE 0 END)
                    AS ADJUSTER
            FROM pooldata.t_claim_xol
        GROUP BY claimno, currency) z
 WHERE     a.claimno = b.noklaim
       AND a.claimno = z.claimno
       AND EXISTS (select 1 from pooldata.t_claim_xol where claimno=a.claimno and businessgroupid=:3 and to_char(dateofloss,'dd/mm/yyyy')=:1 and col_desc=:2)


-- name: export_mbu
-- Isi "Export to Excel" untuk group business MBU.
--
-- Sumber: `RDB List/ExportDetailXOLMBU-SQL.xml`. Dipilih saat `Param.grpbzid == '10004'`
-- pada `GenerateDetailClaimBusinessXOL` — dan kodenya memang dipatok di dalam kueri
-- (`BUSINESSGROUPID='10004'`), sehingga group business TIDAK menjadi parameter di sini.
--
-- Alias Pega dipertahankan; alasannya sama dengan export_per_business.
--
-- :1 tanggal kejadian `dd/mm/yyyy`   :2 penyebab kerugian
select claimno as "ClaimNo", 
    no_polis as "PolicyNo",
    (select currency from pooldata.currency where id=a.currency) as "Currency",
    TO_CHAR(DATEOFLOSS,'dd/mm/yyyy') as "BranchID",
    case when max(sts_aksep)='1' THEN 'CLOSE'
    ELSE 'OS'END AS "StatusClaim",
    max(claim_share_asm)*sum (os_value) as "ClaimEstimate",
    sum (os_value) as "ClaimAmount",
    max(claim_share_asm)*sum (aksep_value) as "BusinessID",
    sum (aksep_value) as "ASMShare",
    max(qq) as "AlasanKlaim",
    max(coverage) as "CoverageNote",
    max(detail_object_desc) as "Message",
    (select sum(salvageos) from pooldata.t_salvage_mbu b where b.claimno=a.claimno and b.currency=a.currency group by currency) as "District",
    (select sum(salvageaksep) from pooldata.t_salvage_mbu b where b.claimno=a.claimno and b.currency=a.currency group by currency) as "DistrictID"
from pooldata.t_claim_xol a where 
BUSINESSGROUPID='10004' and col_desc=:2 and to_char(dateofloss,'dd/mm/yyyy')=:1
group by claimno,no_polis,currency,TO_CHAR(DATEOFLOSS,'dd/mm/yyyy')


-- name: export_treaty_inward
-- Isi "Export to Excel" untuk baris treaty inward.
--
-- Sumber: `RDB List/ExportDetailTreatyInward-SQL.xml`. Dipilih saat
-- `Param.grpbzid == 'treaty'` — nilai khusus, bukan kode group business.
--
-- Nilainya dibawa APA ADANYA, tanpa konversi kurs: kolom `CURRENCY` ikut diekspor,
-- sehingga pembaca berkas dapat melihat satuan tiap barisnya sendiri. Ini berbeda dari
-- grid di layar, yang menyatukan seluruh baris ke satu mata uang dan karena itu harus
-- mengonversi.
--
-- :1 tanggal kejadian `dd/mm/yyyy`   :2 penyebab kerugian
select A.COMPANYNAME as "pyCompany",A.RESERVEDCLAIM as "IsReservedClaim",A.OWNRISKVALUE as "OwnRiskValue", A.QRTREATY as "QsTreaty",
A.DEDUCTIBLEVALUE as "DeductibleValue", A.CLAIMAMOUNT as "ClaimAmount",A.SHAREASM as "ASMShare",A.CLAIMAMOUNTSHARE as "ClaimAmountShareASM",
A.PAIDCLAIMAMOUNTSHARE as "PaidClaimAmountShare",A.BALANCECLAIMASMSHARE as "BalanceClaimASMShare",A.CURRENCY as "Currency",A.CAUSEOFLOSS as "CauseOfLoss",A.DATEOFLOSS as "DateOfLoss" from POOLDATA.T_CLAIM_INWARD_XOL a where A.DATEOFLOSS=:1 and A.CAUSEOFLOSS=:2


-- name: currency_id_by_name
-- Mencari ID mata uang dari namanya, untuk unggahan MBU Salvage.
--
-- Sumber: `RDB List/BrowseCurrency-SQL.xml` —
--   select ID from Currency where currency = {TempDefaultCurrency.Currency}
--
-- Nama tabelnya di rule lama ditulis tanpa skema, sehingga ia mengikuti skema milik
-- pengguna koneksi. Di sini skemanya disebut tegas, sama seperti kueri lain modul ini.
--
-- Perbandingannya DINAIKKAN menjadi tanpa membedakan huruf besar-kecil. Nilainya datang
-- dari berkas yang diketik pengguna, dan "usd" yang ditolak sementara "USD" diterima
-- adalah kegagalan yang tidak dapat dijelaskan kepada siapa pun.
SELECT ID AS CURRENCY_ID
  FROM POOLDATA.CURRENCY
 WHERE UPPER(TRIM(CURRENCY)) = UPPER(TRIM(:1))
 FETCH NEXT 1 ROW ONLY


-- name: insert_salvage_mbu
-- Menyisipkan satu baris hasil unggahan "Upload MBU Salvage".
--
-- Sumber: `RDB List/InsertDataSalvageMBU-SQL.xml`, disalin apa adanya kecuali dua hal.
--
-- Pertama, nilainya diikat sebagai parameter, bukan dirangkai ke dalam teks SQL seperti
-- pola `{TempInsertSalvage.ClaimNo}` warisan — isi berkas berasal dari pengguna, dan
-- merangkainya adalah celah injeksi (§4.5 utang teknis).
--
-- Kedua, `SYSDATE` menjadi `CURRENT_TIMESTAMP` demi portabilitas (`DB-3`).
INSERT INTO POOLDATA.T_SALVAGE_MBU
       (CLAIMNO, DATEOFLOSS, CURRENCY, SALVAGEAKSEP, SALVAGEOS,
        CAUSEOFLOSS, BUSINESSID, INSERTDATE)
VALUES (:1, :2, :3, :4, :5, :6, :7, CURRENT_TIMESTAMP)


-- name: insert_dol_col
-- Menyisipkan satu baris hasil modal "INSERT DOL DAN COL".
--
-- Sumber: `RDB List/DeleteDataInXOLSummarybasedondol-SQL.xml`, tab Save —
--   Insert into POOLDATA.XOL_TABLE_ALL_KLAIM
--          (GROUPBUSINESS,DOL,CURRENCY,LBU_ID,OSVALUE,AKSEPVALUE,CAUSEOFLOSS,SALVAGEVALUE)
--   values ({TempGetDOLCOL.District},{TempGetDOLCOL.NoKTP},…)
--
-- Nama rule-nya menyebut "Delete" karena tab Browse rule yang sama memuat sebuah DELETE.
-- Tab itu TIDAK dijalankan di jalur ini: `Activity/InsertDateAndCauseLossXOL-Act.xml`
-- hanya memanggil RDB-Save, yang menjalankan tab Save. Yang disalin ke sini karena itu
-- hanya sisipannya.
--
-- # Akibat yang harus disadari
--
-- Karena hanya menyisipkan, menyimpan kombinasi Tanggal Kejadian × Penyebab Kerugian yang
-- SAMA dua kali menghasilkan dua baris, dan grid menjumlahkan keduanya. Itu perilaku
-- sistem lama, dibawa apa adanya (`P-5`). Mengubahnya menjadi hapus-lalu-sisip adalah
-- keputusan Work Owner, bukan keputusan teknis — ia mengubah data yang sudah ada.
--
-- Dua hal yang BERUBAH dari rule lama, keduanya wajib:
--
--   * Nilainya diikat sebagai parameter, bukan dirangkai ke dalam teks SQL seperti pola
--     `{TempGetDOLCOL.District}` warisan. Penyebab Kerugian datang dari isian pengguna,
--     dan merangkainya adalah celah injeksi (§4.5 utang teknis).
--   * Nama kolomnya ditulis tegas pada daftar INSERT, sehingga kolom baru di tabel tidak
--     diam-diam mengubah arti posisi nilai.
INSERT INTO POOLDATA.XOL_TABLE_ALL_KLAIM
       (GROUPBUSINESS, DOL, CURRENCY, LBU_ID,
        OSVALUE, AKSEPVALUE, CAUSEOFLOSS, SALVAGEVALUE)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8)
