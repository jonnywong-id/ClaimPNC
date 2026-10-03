-- Kueri modul Acceptation Claim: akseptasi satu klaim treaty NON-proporsional.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- ============================================================================
-- SATU KUERI UNTUK ~50 ISIAN DAN 13 GRID
-- ============================================================================
--
-- Layar ini mengikat hampir seluruh isiannya ke `.ClaimData.*`, satu halaman klipboard yang di
-- basis data tersimpan sebagai SATU dokumen JSON. Dokumen itu diambil UTUH di sini lalu diurai
-- di Go (document.go), bukan dipetik isian demi isian dengan `JSON_VALUE` dan senarai demi
-- senarai dengan `JSON_TABLE`.
--
-- Tiga alasan, dan ketiganya menyentuh hal yang berbeda:
--
--   SATU PERJALANAN     ~50 isian dan 13 grid berarti tiga belas `JSON_TABLE` bila dipetik di
--                       sini, masing-masing membaca ulang dokumen yang sama.
--
--   KEGAGALAN TERBACA   jalur yang salah pada `JSON_TABLE` mengembalikan KOSONG tanpa satu pun
--                       galat. Bentuk dokumen ini belum pernah diperiksa — DDL-nya tidak
--                       tersedia (`R-08`) — sehingga jalur yang salah adalah kemungkinan
--                       nyata. Diurai di Go, "jalur tidak ada" dapat dibedakan dari "jalur ada
--                       tetapi kosong", dan perbedaan itu yang membedakan cacat dari data yang
--                       memang belum diisi. Penghitungnya ada di Stats.
--
--   PORTABILITAS        tiga belas `JSON_TABLE` harus berperilaku sama di Oracle 19c dan
--                       PostgreSQL 17 (`D-20`, `D-24`). Satu kolom CLOB yang dibaca apa adanya
--                       tidak menuntut apa pun dari keduanya.
--
-- ============================================================================
-- KOLOMNYA `DATA_JSONBLOB` — DIUKUR, BUKAN DISIMPULKAN
-- ============================================================================
--
-- `POOLDATA.JSON_KLAIM` punya DUA kolom JSON, dan keempat kueri inbox non-prop membaca
-- `DATA_JSON` (`c.data_json.DateOfLoss`, `c.data_json.IDMaster`). Berkas ini semula mengikuti
-- mereka. Itu KELIRU, dan yang membuktikannya adalah pengukuran langsung ke basis data
-- pengembangan pada 2026-09-30:
--
--   objek kerja `PYID LIKE 'CLMNP-%'`            22 baris
--   yang punya pasangan di POOLDATA.JSON_KLAIM   14 baris
--   LENGTH(DATA_JSON) pada kelimabelasnya        0
--   LENGTH(DATA_JSONBLOB)                        5.327 – 9.878
--
-- Dokumen klaim treaty non-proporsional tersimpan di `DATA_JSONBLOB`. Ke-38 kunci tingkat
-- atasnya cocok satu-untuk-satu dengan jalur di section.go — `IDMaster`, `TreatyName`,
-- `DateOfLoss`, `InterestList`, `ListClaimAmount`, `CNPSpreadLoss`, `SpreadingRisk`,
-- `SpreadingClaim`, `SpreadingBreakQS`, `SuggestList`, dan seterusnya — sehingga kolomnya yang
-- keliru, bukan jalurnya.
--
-- Layar Prop (`outstandingclaim`) membaca kolom yang SAMA, dan itu bukan kebetulan: keduanya
-- membaca dokumen objek kerja, bukan salinan yang dipetik kueri inbox.
--
-- # Akibatnya bagi kueri INBOX non-prop, yang tidak disentuh berkas ini
--
-- `JSON_VALUE(c.DATA_JSON, '$.DateOfLoss')` pada kueri inbox mengembalikan NULL untuk seluruh
-- baris — dan itulah sebab kolom "Date of Loss" dan "ID Master" kosong di layar antrean, di
-- sistem baru MAUPUN di Pega. Kuerinya di Pega membaca kolom yang sama. Membetulkannya adalah
-- perubahan perilaku yang menuntut keputusan `P-5` tersendiri, bukan perbaikan yang diambil
-- diam-diam dari berkas ini.
--
-- ============================================================================
-- DUA TABEL, DAN PEMBAGIAN TUGASNYA
-- ============================================================================
--
--   DATAPEGA.PC_ASM_FW_GCNMFW_WORK  w   KEADAAN objek kerja — nomor, status, pengubah
--   POOLDATA.JSON_KLAIM             b   ISI klaimnya — satu dokumen JSON
--
-- Pembagian itu bukan pilihan gaya: status alur kerja dan petugas pengubah adalah milik objek
-- kerja Pega dan tidak ikut tersimpan di dalam dokumen `.ClaimData`.
--
-- Gabungannya `LEFT JOIN`, dengan alasan yang sama seperti di Inbox Claim Treaty Non Prop:
-- klaim yang belum punya baris di JSON_KLAIM tetap DAPAT DIBUKA, dengan seluruh isian kosong
-- dan nomor klaimnya tetap terbaca. Menggantinya dengan INNER akan membuat klaim seperti itu
-- dijawab "tidak ditemukan" — padahal ia ada, hanya isinya yang belum tersalin.
--
-- Kunci gabungannya `w.PZINSKEY = b.IDPEGA`, kolom yang sama yang dipakai Inbox Claim Treaty
-- Non Prop (di sana lewat `a.PXREFOBJECTKEY`, yang isinya `PZINSKEY` objek kerja yang
-- ditunjuk).
--
-- ============================================================================
-- KENAPA ADA PENYARING `LIKE 'CLMNP-%'` PADA KUERI RINCIAN
-- ============================================================================
--
-- Karena `PC_ASM_FW_GCNMFW_WORK` memuat objek kerja SELURUH jenis klaim, bukan hanya treaty
-- non-proporsional. Tanpa penyaring itu, alamat layar ini dapat diisi nomor klaim PNC biasa
-- dan layar akan menggambarnya dengan susunan akseptasi treaty — ~50 isian yang hampir
-- seluruhnya kosong, tanpa satu pun tanda bahwa yang dibuka adalah jenis klaim yang berbeda.
--
-- Penyaringnya dipasang pada `PYID` — nomor klaim — dengan JANGKAR DEPAN, sama dengan keempat
-- kueri inbox non-prop. Itu berbeda dari layar Prop, yang menyaring `PZINSKEY LIKE '%CLMP%'`
-- tanpa jangkar. Perbedaan itu penting justru di modul INI: `CLMP` adalah awalan dari `CLMNP`
-- pada tiga huruf pertamanya, sehingga penyaring longgar akan mencampur kedua layar treaty.
--
-- Polanya literal, bukan masukan pengguna, sehingga tidak butuh ESCAPE.
--
-- ============================================================================
-- TIDAK ADA SATU PUN PERNYATAAN YANG MENULIS — DAN ITU DISENGAJA
-- ============================================================================
--
-- Work Owner memutuskan layar ini dibangun PENUH termasuk Submit (2026-09-30). Yang TIDAK
-- berubah oleh keputusan itu adalah `P-1`: selama masa paralel, setiap tabel hanya boleh
-- ditulis SATU sistem, dan kedua tabel di atas masih ditulis Pega.
--
-- Karena itu jalur Submit ADA dan tervalidasi penuh di domain, tetapi pengisi seam ini menolak
-- menulis dengan `ErrWriteNotOwned` — penolakan yang menyebut sebab dan jalan keluarnya,
-- bukan penyimpanan diam-diam ke tabel milik sistem lain. Perpindahan kepemilikannya menempuh
-- `D-63`: permintaan tertulis, persetujuan Work Owner, pelaksanaan DBA, lalu diuji dengan
-- menjalankan Pega dan Go bersamaan.
--
-- Menambahkan `UPDATE` di berkas ini sebelum itu terjadi bukan mempercepat apa pun: ia membuat
-- dua sistem menulisi nilai akseptasi yang sama, dan akibatnya bukan galat melainkan data yang
-- saling menimpa tanpa jejak.

-- name: find_claim
-- Rincian akseptasi satu klaim treaty non-proporsional menurut nomor klaimnya.
--
-- Kolomnya `DATA_JSONBLOB` — lihat catatan "KOLOMNYA DATA_JSONBLOB" di kepala berkas.
--
-- Bind: :1 nomor klaim (mis. `CLMNP-232`)
SELECT w.PYID                                        AS CLAIM_ID,
       w.PZINSKEY                                    AS REFERENCE,
       w.PYSTATUSWORK                                AS STATUS_WORK,
       w.PXUPDATEOPERATOR                            AS LAST_UPDATE_OPERATOR,
       b.DATA_JSONBLOB                               AS CLAIM_DOCUMENT
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       LEFT JOIN POOLDATA.JSON_KLAIM b
              ON w.PZINSKEY = b.IDPEGA
 WHERE w.PYID = :1
   AND w.PYID LIKE 'CLMNP-%'

-- name: check_tables
-- Dipakai perintah `-periksa`: memastikan kedua tabel DAN gabungannya terbaca dari koneksi
-- yang dipakai.
--
-- Ia tidak menyentuh satu baris pun — yang diperiksa adalah hak baca dan keberadaan tabelnya,
-- bukan isinya. Gabungannya ikut diperiksa karena kegagalan yang paling mungkin terjadi bukan
-- "tabel tidak ada" melainkan "hak baca hanya diberikan pada salah satunya".
SELECT COUNT(*) AS PROBE
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
       LEFT JOIN POOLDATA.JSON_KLAIM b
              ON w.PZINSKEY = b.IDPEGA
 WHERE 1 = 0
