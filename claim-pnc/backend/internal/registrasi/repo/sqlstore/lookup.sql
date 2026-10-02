-- Kueri sumber acuan: kurs dan polis.
--
-- Keduanya membaca tabel WARISAN dan tidak pernah menulisnya.

-- name: kurs_pada_tanggal
--
-- Kurs yang berlaku untuk sebuah mata uang pada sebuah tanggal.
--
-- # Kenapa "<=" dan bukan "="
--
-- POOLDATA.M_CURRENCYSTANDARD tidak memuat satu baris untuk setiap hari — 6.995 baris
-- untuk 40 mata uang berarti rata-rata 175 tanggal per mata uang, bukan 365. Kurs yang
-- berlaku pada tanggal kejadian karena itu adalah kurs TERBARU yang tidak melewati
-- tanggal itu, persis seperti papan kurs yang tidak berubah di hari libur.
--
-- # ID adalah KODE ANGKA, bukan simbol ISO
--
-- Terverifikasi 2026-09-24: kolom ID berisi 10001 (USD), 10025 (EUR), 10026 (IDR) — bukan
-- "USD"/"EUR"/"IDR". Kode itulah yang dibawa polis pada $.Currency, sehingga pemanggil
-- meneruskannya apa adanya dan TIDAK perlu menerjemahkan lebih dulu. Menerjemahkannya
-- justru akan membuat kuerinya tidak pernah menemukan baris.
--
-- Artinya juga sebaliknya: mencoba kueri ini dengan "USD" akan menjawab "tidak ada",
-- dan jawaban itu TIDAK berarti kursnya belum dimuat.
--
-- IDR bernilai 1 dan ikut tersimpan seperti mata uang lain, jadi klaim rupiah menempuh
-- jalur yang sama persis — tidak ada cabang khusus di mana pun.
--
-- # Kenapa nilainya dikembalikan sebagai TEKS
--
-- CURRENCYVALUE bertipe VARCHAR2 dan memakai KOMA sebagai pemisah desimal ("21509,68").
-- Mengubahnya menjadi angka di dalam SQL menuntut TO_NUMBER dengan format mask, dan
-- hasilnya bergantung pada NLS server — nilai yang sama dapat terbaca berbeda di dua
-- lingkungan. Penguraiannya dilakukan Go, di satu tempat, dan dapat diuji.
--
-- Tidak ditemukan berarti TIDAK ADA baris, bukan nol. `D-48` menetapkan klaim ditolak
-- bila kurs tanggal kejadian tidak tersedia — tidak ada nilai bawaan, dan tidak ada
-- RETURN 1 seperti GETCURRENCYSTANDARD yang lama.
SELECT CURRENCYVALUE
  FROM POOLDATA.M_CURRENCYSTANDARD
 WHERE TRIM(ID) = :1
   AND CURRENCYDATE <= :2
 ORDER BY CURRENCYDATE DESC
 FETCH FIRST 1 ROWS ONLY

-- name: polis_ambil
--
-- Snapshot polis (`D-04`).
--
-- # Sumbernya TABEL polis, dokumen hanya cadangan (Work Owner, 2026-10-01)
--
-- Data polis tidak lagi dibaca dari JSON_POLIS.DATA_JSONBLOB. Urutannya per kolom:
--
--   1. tabel polis menurut NOPOLIS + PRODKE — T_GENERAL (kepala polis), T_OFFERFACIN
--      (CedingCoName, OfferFacIn.PercentShare), T_GENERAL_DELIVERYADDRESSLIST (alamat);
--   2. bila kolom tabelnya kosong atau barisnya tidak ada: dokumen JSON_POLIS.POLICYDATA.
--
-- Cadangan ke POLICYDATA tetap perlu. Terverifikasi 2026-10-01 pada polis klaim: T_GENERAL
-- ada untuk 32 dari 37 PRODKE, kolom CURRENCY dan TYPEOFCOINS kosong pada sebagian besar
-- baris lama, dan T_GENERAL_DELIVERYADDRESSLIST kosong untuk 37 dari 37.
--
-- # PRODKE — versi terbaru, sama dengan Pega
--
-- PRODKE adalah PRODKE TERBESAR polis itu di T_GENERAL — persis RDB List/
-- BroswsePolisByPolicyNo-SQL.xml:23 (`PRODKE = (SELECT MAX(TO_NUMBER(PRODKE)) FROM T_GENERAL
-- ...)`), kueri yang mengisi daftar polis layar View Polis (SearchPolicy) dan yang dipakai
-- Input Receive Document (`claim_report_policy_find`). Dengan begitu RCVN dan PNCN membaca
-- versi polis yang sama.
--
-- Bila polis tidak punya baris T_GENERAL, PRODKE terbesar JSON_POLIS yang ber-POLICYDATA
-- dipakai. Bukan lagi "baris JSON_POLIS terbaru menurut TGL_INPUT": urutan input tidak sama
-- dengan urutan versi, dan dokumen dapat dikonversi ulang setelah versi berikutnya terbit.
--
-- Dokumen POLICYDATA cadangan diambil dari baris JSON_POLIS PRODKE yang SAMA (Pega
-- BrowsePolis-SQL.xml:86 `a.prodke = {TempPolis.ProdKe}`), terbaru menurut TGL_INPUT.
--
-- Bind: :1 dan :2 sama-sama nomor polis — pengikatan posisi Oracle menuntut satu nilai per
-- kemunculan.
--
-- # Periode
--
-- Dua kolom terakhir adalah T_GENERAL.STARTDATE/ENDDATE (DATE jam dinding WIB) — CADANGAN
-- teks Pega StartDateTime/EndDateTime dokumen. Periode SATU-SATUNYA yang mendahulukan
-- dokumen: T_GENERAL tidak mengikuti endorsemen (lihat PolicyRepo.Get).
WITH versi AS (
    SELECT v.NOPOLIS, v.PRODKE
      FROM (SELECT g.NOPOLIS, TRIM(g.PRODKE) AS PRODKE, 1 AS SUMBER
              FROM POOLDATA.T_GENERAL g
             WHERE g.NOPOLIS = :1
            UNION ALL
            SELECT p.NOPOLIS, CAST(p.PRODKE AS VARCHAR(30)), 2
              FROM POOLDATA.JSON_POLIS p
             WHERE p.NOPOLIS = :2
               AND p.POLICYDATA IS NOT NULL) v
     ORDER BY v.SUMBER, CAST(v.PRODKE AS DECIMAL(20)) DESC
     FETCH FIRST 1 ROWS ONLY
),
dok AS (
    SELECT v.NOPOLIS, v.PRODKE,
           (SELECT p.POLICYDATA
              FROM POOLDATA.JSON_POLIS p
             WHERE p.NOPOLIS = v.NOPOLIS
               AND CAST(p.PRODKE AS VARCHAR(30)) = v.PRODKE
               AND p.POLICYDATA IS NOT NULL
             ORDER BY p.TGL_INPUT DESC
             FETCH FIRST 1 ROWS ONLY) AS POLICYDATA
      FROM versi v
),
gen AS (
    SELECT g.*,
           ROW_NUMBER() OVER (ORDER BY CASE WHEN g.TGL_INPUT IS NULL THEN 1 ELSE 0 END,
                                       g.TGL_INPUT DESC) AS RN
      FROM POOLDATA.T_GENERAL g
     WHERE g.NOPOLIS = (SELECT NOPOLIS FROM dok) AND g.PRODKE = (SELECT PRODKE FROM dok)
),
fac AS (
    SELECT o.CEDINGNAME, CAST(o.PERCENTSHARE AS VARCHAR(50)) AS PERCENTSHARE,
           ROW_NUMBER() OVER (ORDER BY CASE WHEN o.TGL_INSERT IS NULL THEN 1 ELSE 0 END,
                                       o.TGL_INSERT DESC) AS RN
      FROM POOLDATA.T_OFFERFACIN o
     WHERE o.POLICYNO = (SELECT NOPOLIS FROM dok)
       AND o.PRODKE = (SELECT PRODKE FROM dok)
),
alamat AS (
    SELECT a.ADDRESS,
           ROW_NUMBER() OVER (ORDER BY a.ADDRESSTYPE) AS RN
      FROM POOLDATA.T_GENERAL_DELIVERYADDRESSLIST a
     WHERE a.NOPOLIS = (SELECT NOPOLIS FROM dok)
       AND a.PRODKE = (SELECT PRODKE FROM dok)
)
SELECT COALESCE(g.NOPOLIS, JSON_VALUE(d.POLICYDATA, '$.PolicyNo'), d.NOPOLIS),
       COALESCE(g.GROUPPANEL, JSON_VALUE(d.POLICYDATA, '$.Quotation.GroupPanel')),
       COALESCE(g.BUSINESSTYPE, JSON_VALUE(d.POLICYDATA, '$.Quotation.BusinessType')),
       COALESCE(g.BUSINESSNAME, JSON_VALUE(d.POLICYDATA, '$.Quotation.BusinessName')),
       JSON_VALUE(d.POLICYDATA, '$.StartDateTime'),
       JSON_VALUE(d.POLICYDATA, '$.EndDateTime'),
       COALESCE(g.TYPEOFPOLICY, JSON_VALUE(d.POLICYDATA, '$.TypeOfPolicy')),
       COALESCE(g.CURRENCY, JSON_VALUE(d.POLICYDATA, '$.Currency')),
       COALESCE(g.THEINSURED, JSON_VALUE(d.POLICYDATA, '$.TheInsured' RETURNING VARCHAR2(4000))),
       COALESCE(g.QQNAME, JSON_VALUE(d.POLICYDATA, '$.QQName' RETURNING VARCHAR2(4000))),
       COALESCE(g.BRANCHCODE, JSON_VALUE(d.POLICYDATA, '$.Quotation.BranchCode')),
       COALESCE(CAST(g.FINISHEDSPREADING AS VARCHAR(10)), JSON_VALUE(d.POLICYDATA, '$.SpreadingStatus')),
       -- Sembilan jalur berikut diisi Pega ke T_CLAIM_PNC saat klaim dibuat
       -- (PEGA_CONVERT_JSONKLAIM_PNC.prc baris 317-373).
       COALESCE(g.BUSINESSCODE, JSON_VALUE(d.POLICYDATA, '$.Quotation.BusinessCode')),
       COALESCE(g.BRANCHNAME, JSON_VALUE(d.POLICYDATA, '$.Quotation.BranchName')),
       COALESCE(g.SOURCEOFBUSINESS, JSON_VALUE(d.POLICYDATA, '$.Quotation.SourceOfBusiness')),
       COALESCE(g.SOBNAME, JSON_VALUE(d.POLICYDATA, '$.Quotation.SobName')),
       d.PRODKE,
       COALESCE(g.LEADERPOLICYCOAS, JSON_VALUE(d.POLICYDATA, '$.PolicyLeader')),
       COALESCE(g.TYPEOFCOINS, JSON_VALUE(d.POLICYDATA, '$.TypeOfCoins')),
       COALESCE(f.CEDINGNAME, JSON_VALUE(d.POLICYDATA, '$.CedingCoName')),
       COALESCE(f.PERCENTSHARE, JSON_VALUE(d.POLICYDATA, '$.OfferFacIn.PercentShare')),
       -- Alamat penerima klaim bawaan: InputRegister_act mengisi ReceiverClaim.Address
       -- dari .Policy.DeliveryAddressList(1).ASMAddress.
       COALESCE(a.ADDRESS, JSON_VALUE(d.POLICYDATA, '$.DeliveryAddressList[0].ASMAddress')),
       g.STARTDATE,
       g.ENDDATE
  FROM dok d
  LEFT JOIN gen g ON g.RN = 1
  LEFT JOIN fac f ON f.RN = 1
  LEFT JOIN alamat a ON a.RN = 1

-- name: parameter_ambil
--
-- Satu parameter bisnis dari POOLDATA.M_PARAMETER.
--
-- # Kenapa tabel ini, bukan tabel baru
--
-- M_PARAMETER sudah ada, berbentuk (ID, JSONDATA), dan KOSONG — nol baris pada
-- 2026-09-24. Ia master parameter umum yang belum dipakai domain mana pun, sehingga
-- memakainya tidak menabrak siapa pun dan tidak menambah tabel.
--
-- ID diberi awalan `PNC.` supaya parameter modul ini tidak dapat tertukar dengan
-- parameter domain lain bila tabel ini kelak ikut dipakai mereka.
--
-- # Kenapa BUKAN berkas konfigurasi
--
-- `D-15` menolaknya secara tegas. Menaruh ambang di berkas konfigurasi berarti setiap
-- perubahan kebijakan membutuhkan deployment, dan itu persis masalah yang membuatnya
-- di-hardcode sejak awal. Ambang dan penerima notifikasi adalah MASTER DATA milik
-- pengguna bisnis, bukan konfigurasi milik tim infrastruktur.
SELECT JSONDATA
  FROM POOLDATA.M_PARAMETER
 WHERE ID = :1

-- name: pic_teknik_paling_ringan
--
-- Petugas teknis dengan beban paling sedikit untuk sebuah lini bisnis.
--
-- Disalin dari `RDB List/BrowsePICRandomTeam-SQL.xml` — `ORDER BY counter_quota ASC`,
-- yakni yang paling sedikit bebannya mendapat tugas berikutnya. Itulah algoritma yang
-- `R-04` sebut hilang bersama PNCAdminRouter dan PNCTeknikRouter, dan yang terbaca
-- kembali dari kueri ini.
--
-- # Satu hal yang TIDAK dibawa
--
-- Kueri lama memuat `and operator_ID != 'ELLENSUPRIYATI'` — satu dari 24 Operator ID yang
-- tertanam di dalam kode. `D-15` menetapkan seluruhnya menjadi master data, dan `P-5`
-- butir 1 menjadikannya perbaikan yang direncanakan. Mekanisme yang benar sudah ada di
-- tabel ini: `STS_AKTIF`. Mengecualikan seseorang dilakukan dengan menonaktifkannya di
-- master, bukan dengan menulis namanya di dalam kueri.
--
-- Selisih yang mungkin timbul pada uji kesetaraan karena itu SUDAH DIPERKIRAKAN, dan
-- terpetakan ke butir `P-5` nomor 1.
SELECT OPERATOR_ID
  FROM POOLDATA.MST_USER_TEKNIK
 WHERE STS_AKTIF = '1'
   AND TRIM(TYPE_BUSINESS) = :1
 ORDER BY COUNTER_QUOTA ASC, OPERATOR_ID ASC
 FETCH FIRST 1 ROWS ONLY

-- name: pic_teknik_naikkan_beban
--
-- Menaikkan pencacah beban petugas yang baru saja menerima tugas.
--
-- Disalin dari `RDB List/AddTJobCounterPIC_SQL-SQL.xml`. Tanpa langkah ini, petugas yang
-- sama akan terus terpilih karena bebannya tidak pernah bertambah.
UPDATE POOLDATA.MST_USER_TEKNIK
   SET COUNTER_QUOTA = COUNTER_QUOTA + 1
 WHERE OPERATOR_ID = :1

-- name: polis_koasuransi
--
-- Baris koasuransi polis, untuk DeriveCoinsurance.
--
-- Sumbernya POOLDATA.T_COINSLIST menurut NOPOLIS dan PRODKE snapshot klaim (Work Owner,
-- 2026-10-01: data polis tidak dibaca dari JSON_POLIS.DATA_JSONBLOB bila ada tabelnya).
-- Kolomnya padanan CoinsList dokumen polis: COINSID, COINSNAME, LEADER ('true'/'false'),
-- PERCENT_SHARE, FLAGDELETE. PRODKE yang tidak punya baris berarti polis tanpa koasuransi.
SELECT c.LEADER, c.COINSNAME, c.PERCENT_SHARE
  FROM POOLDATA.T_COINSLIST c
 WHERE c.NOPOLIS = :1
   AND c.PRODKE = :2
 ORDER BY c.COINSID

-- ============================================================================
-- WILAYAH KEJADIAN — daftar pilihan bertingkat layar Input Register
-- ============================================================================
--
-- Sumber tiap tingkat dan alasannya ada di catatan AreaDirectory (seam.go). Tiga dari
-- lima report definition Pega hilang dari export; tabel sumbernya dipastikan dengan
-- menelusuri satu contoh nyata dari layar Pega sampai ke kode posnya.

-- name: penyebab_kerugian_bisnis
--
-- Pilihan Penyebab Kerugian satu kode bisnis — pengganti BrowseCouseOfLoss_Business yang
-- hilang dari export. Satu D_COL_ID dapat muncul lebih dari sekali untuk bisnis yang sama
-- bila view-nya menggandakan baris; DISTINCT menjaga daftar tetap satu baris per pilihan.
SELECT DISTINCT CAST(D_COL_ID AS VARCHAR(20)), DESCRIPTION
  FROM POOLDATA.V_D_CAUSE_OF_LOSS_BUSINESS
 WHERE CAST(BISNISID AS VARCHAR(20)) = :1
   AND STS_AKTIF = '1'
   AND DESCRIPTION IS NOT NULL
 ORDER BY DESCRIPTION

-- name: wilayah_negara
SELECT ID, COUNTRY, CAST(NULL AS VARCHAR(10))
  FROM POOLDATA.COUNTRY
 WHERE COUNTRY IS NOT NULL
 ORDER BY COUNTRY

-- name: wilayah_provinsi
--
-- Disaring menurut NAMA negara. NATIONID tidak dapat dipakai: skema kodenya berbeda dari
-- COUNTRY.ID, sehingga menyaring dengan kode negara mengosongkan daftar Indonesia.
SELECT ID, NOTE, CAST(NULL AS VARCHAR(10))
  FROM POOLDATA.PROVINCE
 WHERE UPPER(NATIONNAME) = UPPER(:1)
 ORDER BY NOTE

-- name: wilayah_kota
SELECT ID, NOTE, CAST(NULL AS VARCHAR(10))
  FROM POOLDATA.CITYINPUT
 WHERE PROVINCEID = :1
 ORDER BY NOTE

-- name: wilayah_kabupaten
SELECT ID, DISTRICTNAME, CAST(NULL AS VARCHAR(10))
  FROM POOLDATA.DISTRICTINPUT
 WHERE CITYID = :1
 ORDER BY DISTRICTNAME

-- name: wilayah_kelurahan
--
-- M_RW menyimpan kelurahan sebagai dokumen JSON: ID, DistrictID, Note, ZipCode. ZipCode
-- itulah yang Pega salin ke ClaimData.PostalCode saat kelurahan dipilih.
SELECT ID, JSON_VALUE(JSONDATA, '$.Note'), JSON_VALUE(JSONDATA, '$.ZipCode')
  FROM POOLDATA.M_RW
 WHERE JSON_VALUE(JSONDATA, '$.DistrictID') = :1
 ORDER BY 2
