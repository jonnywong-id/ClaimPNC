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
-- Snapshot polis diambil dari dokumen JSON milik GISFW (`D-04`).
--
-- # Kenapa dari JSON_POLIS, bukan dari tabel polis
--
-- Data polis dimiliki tim lain (`D-03`), dan satu-satunya bentuk yang tersedia di basis
-- data ini adalah dokumennya. Seluruh field yang dibutuhkan terbukti ada: GroupPanel,
-- BusinessCode, BranchCode muncul di 200 dari 200 dokumen yang diperiksa 2026-09-24.
--
-- # Kenapa baris terbaru yang diambil
--
-- Satu nomor polis dapat muncul lebih dari sekali — endorsement, perpanjangan, atau
-- konversi ulang. Yang dipakai registrasi adalah keadaan polis TERAKHIR yang tercatat.
--
-- JSON_VALUE dipakai, bukan penguraian di Go: ia portabel ke PostgreSQL 17+ (`D-24`),
-- dan membaca sepuluh field tanpa mengangkut CLOB berukuran puluhan kilobyte ke aplikasi.
-- # DUA kolom memuat dokumennya, dan TIDAK ADA yang lengkap
--
-- Terverifikasi 2026-09-24 atas 211.590 baris:
--
--   POLICYDATA (CLOB)     terisi pada 168.298
--   DATA_JSONBLOB (BLOB)  terisi pada 197.687
--   keduanya kosong       pada  13.732
--
-- Membaca salah satu saja menolak polis yang sebenarnya ada. Versi pertama kueri ini
-- hanya membaca POLICYDATA, dan akibatnya tombol Register Klaim menjawab galat umum
-- untuk 29.389 polis yang dokumennya hanya ada di BLOB.
--
-- # Kenapa POLICYDATA didahulukan
--
-- Pada 168.127 baris keduanya terisi, dan pada 199 dari 200 contoh keduanya menyebut
-- nomor polis yang sama. Mendahulukan POLICYDATA membuat polis yang SUDAH terbaca
-- sebelumnya tetap terbaca sama persis; perubahan ini hanya MENAMBAH yang tadinya
-- gagal. Satu contoh yang berbeda dicatat sebagai pertanyaan terbuka — mana yang benar
-- saat keduanya tidak sepakat belum diketahui.
--
SELECT
       COALESCE(JSON_VALUE(p.POLICYDATA, '$.PolicyNo'),
                JSON_VALUE(p.DATA_JSONBLOB, '$.PolicyNo')),
       COALESCE(JSON_VALUE(p.POLICYDATA, '$.Quotation.GroupPanel'),
                JSON_VALUE(p.DATA_JSONBLOB, '$.Quotation.GroupPanel')),
       COALESCE(JSON_VALUE(p.POLICYDATA, '$.Quotation.BusinessType'),
                JSON_VALUE(p.DATA_JSONBLOB, '$.Quotation.BusinessType')),
       COALESCE(JSON_VALUE(p.POLICYDATA, '$.Quotation.BusinessName'),
                JSON_VALUE(p.DATA_JSONBLOB, '$.Quotation.BusinessName')),
       COALESCE(JSON_VALUE(p.POLICYDATA, '$.StartDateTime'),
                JSON_VALUE(p.DATA_JSONBLOB, '$.StartDateTime')),
       COALESCE(JSON_VALUE(p.POLICYDATA, '$.EndDateTime'),
                JSON_VALUE(p.DATA_JSONBLOB, '$.EndDateTime')),
       COALESCE(JSON_VALUE(p.POLICYDATA, '$.TypeOfPolicy'),
                JSON_VALUE(p.DATA_JSONBLOB, '$.TypeOfPolicy')),
       COALESCE(JSON_VALUE(p.POLICYDATA, '$.Currency'),
                JSON_VALUE(p.DATA_JSONBLOB, '$.Currency')),
       COALESCE(JSON_VALUE(p.POLICYDATA, '$.TheInsured'),
                JSON_VALUE(p.DATA_JSONBLOB, '$.TheInsured')),
       COALESCE(JSON_VALUE(p.POLICYDATA, '$.QQName'),
                JSON_VALUE(p.DATA_JSONBLOB, '$.QQName')),
       COALESCE(JSON_VALUE(p.POLICYDATA, '$.Quotation.BranchCode'),
                JSON_VALUE(p.DATA_JSONBLOB, '$.Quotation.BranchCode')),
       COALESCE(JSON_VALUE(p.POLICYDATA, '$.SpreadingStatus'),
                JSON_VALUE(p.DATA_JSONBLOB, '$.SpreadingStatus'))
  FROM POOLDATA.JSON_POLIS p
 WHERE p.NOPOLIS = :1
   AND (p.POLICYDATA IS NOT NULL OR p.DATA_JSONBLOB IS NOT NULL)
 ORDER BY p.TGL_INPUT DESC
 FETCH FIRST 1 ROWS ONLY

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
