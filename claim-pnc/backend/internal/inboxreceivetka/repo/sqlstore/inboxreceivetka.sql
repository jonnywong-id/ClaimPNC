-- Kueri modul Inbox Receive TKA.
--
-- TIGA tabel, dan aplikasi ini menulis SATU kolom pada satu di antaranya:
--
--   POOLDATA.JSON_KLAIM             sumber daftar     — hanya DIBACA (snapshot kasus Pega)
--   POOLDATA.T_CLAIM_PNC            klaim sebenarnya  — DIBACA, dan satu kolom DITULIS
--   POOLDATA.T_GENERAL              tabel polis       — hanya DIBACA
--
--
-- ============================================================================
-- SUMBER BARU (2026-10-08)
-- ============================================================================
--
-- Keputusan Work Owner 2026-10-08: tabel objek kerja Pega (`DATAPEGA.PC_ASM_FW_GCNMFW_WORK`)
-- SUDAH TIDAK DIPAKAI. Penjelasan di bawah tentang kolom `TKA_1`, `TANGGALDOKLENGKAP`,
-- `REGISTERDATE_1`, dan `PYID` adalah REKAMAN asal-usul aturan Report Definition; yang kini
-- dijalankan adalah pemetaan berikut, diukur di Oracle dev portal ASM pada 2026-10-08:
--
--   Pega (objek kerja)         Sumber baru
--   -------------------------- ----------------------------------------------------------
--   PZINSKEY                   JSON_KLAIM.IDPEGA  (= T_CLAIM_PNC.CLAIMID)
--   PYID                       REPLACE(IDPEGA, 'ASM-FW-GCNMFW-WORK ', '')
--   TKA_1  (.ClaimData.TKA)    JSON_VALUE(DATA_JSONBLOB, '$.TKA') pada snapshot TERAKHIR
--   TANGGALDOKLENGKAP          JSON_VALUE(DATA_JSONBLOB, '$.TanggalDokLengkap'), idem
--   PYSTATUSWORK               T_CLAIM_PNC.STATUSWORK
--   POLICYNO · QQNAME          T_CLAIM_PNC.NOPOLIS · QQNAME
--   DATEOFLOSS_1               T_CLAIM_PNC.DATEOFLOSS (DATE)
--   REGISTERDATE_1 (teks)      T_CLAIM_PNC.REGISTERDATE (DATE) — pemindai Go ikut berubah
--
-- # Kenapa penanda TKA diambil dari JSON_KLAIM
--
-- Penanda TKA TIDAK ADA di tabel relasional mana pun: `T_CLAIM_PNC.TKA` kosong pada seluruh
-- baris (prosedur konversi tidak mengisinya), dan `T_CLAIMLIST_ADMIN` tidak punya kolomnya.
-- Satu-satunya tempat ia tersimpan adalah snapshot kasus di `POOLDATA.JSON_KLAIM` — tabel yang
-- juga dibaca modul riwayat klaim, laporan klaim, dan akseptasi. `JSON_KLAIM` menyimpan
-- BEBERAPA versi per kasus (16.257 baris untuk 2.470 kasus); yang dipakai versi terakhir
-- menurut `TGL_INPUT`. Seri `TGL_INPUT` ada, tetapi tidak satu pun berbeda nilai TKA-nya.
--
-- # Akibat yang terukur: populasi BERTAMBAH
--
-- Kolom `TKA_1` Pega hanya terisi pada 11 kasus, padahal snapshotnya bertanda TKA pada 29
-- kasus — 18 kasus lain tidak pernah di-indeks ulang Pega setelah kolomnya di-expose. Daftar
-- menunggu karena itu naik dari 2 baris (versi lama) menjadi 15 baris. Satu baris lama hilang:
-- kasus yatim yang tidak ada di `T_CLAIM_PNC` MAUPUN `JSON_KLAIM` — penanda TKA-nya hanya
-- tercatat di tabel kerja yang kini tidak dipakai.
--
-- # Status pekerjaan NULL tidak lagi menyembunyikan baris
--
-- `T_CLAIM_PNC.STATUSWORK` dapat NULL (diisi prosedur konversi yang dapat tertinggal). Baris
-- seperti itu TETAP TAMPIL: menyembunyikannya berarti menyembunyikan pekerjaan yang belum
-- terbukti selesai. Pada data 2026-10-08 tidak ada klaim TKA yang statusnya NULL.
--
-- Empat aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (D-20).
--   4. Tanpa pemanggilan stored procedure (D-02).
--
--
-- ============================================================================
-- SUMBERNYA SAMA PERSIS DENGAN REPORT DEFINITION PEGA
-- ============================================================================
--
-- `Report Definition/InboxTKA_RD-RD.xml` berjalan atas kelas `ASM-FW-GCNMFW-Work-PNC`,
-- yaitu tabel `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`. Ketiga penyaringnya:
--
--   A  .ClaimData.TKA               =  "1"
--   B  .ClaimData.TanggalDokLengkap IS NULL
--   C  .pyStatusWork                != "Resolved-Completed"
--      logika: A AND B AND C
--      urut:   .ClaimData.RegisterDate menaik (pySortOrder = 1)
--
-- # Ketiga properti itu ADA sebagai kolom, dan itu bukan kebetulan
--
-- Report Definition menjalankan SQL terhadap kolom basis data, bukan terhadap klipboard.
-- Sebuah properti hanya dapat dipakai sebagai PENYARING bila ia di-expose sebagai kolom —
-- sehingga fakta bahwa RD ini berjalan sudah membuktikan ketiganya ada.
--
-- Diverifikasi langsung ke katalog basis data produksi pada 2026-09-24:
--
--   Properti Pega                    Kolom                 Tipe
--   -------------------------------- --------------------- -----------------
--   .ClaimData.TKA                   TKA_1                 VARCHAR2(1)
--   .ClaimData.TanggalDokLengkap     TANGGALDOKLENGKAP     TIMESTAMP(6)
--   .ClaimData.RegisterDate          REGISTERDATE_1        VARCHAR2(32)
--   .ClaimData.DateOfLoss            DATEOFLOSS_1          TIMESTAMP
--   .Policy.PolicyNo                 POLICYNO
--   .Policy.QQName                   QQNAME
--   .pyID                            PYID
--   .pzInsKey                        PZINSKEY
--
-- # Akhiran `_1` adalah penyelesai TABRAKAN NAMA, bukan penanda properti tertanam
--
-- Pega hanya menambahkannya bila dua properti bernama sama di-expose ke tabel yang sama.
-- `DateOfLoss` ada di tingkat kerja DAN di ClaimData, sehingga yang kedua menjadi
-- `DATEOFLOSS_1`. `TanggalDokLengkap` hanya ada di ClaimData, sehingga namanya polos.
--
-- Tabel ini memuat `TKA` dan `TKA_1` sekaligus. Yang membawa `.ClaimData.TKA` adalah
-- **`TKA_1`** — diverifikasi terhadap dua klaim yang benar-benar tampil di layar Pega
-- (`PNC-1546` dan `PNC-1729`): keduanya ber-`TKA_1 = '1'` sementara kolom `TKA` polosnya
-- kosong, dan cacah dengan penyaring di bawah menghasilkan **tepat 2** — sama persis dengan
-- jumlah baris pada layar Pega.
--
--
-- ============================================================================
-- DUA KOLOM YANG TIDAK DI-EXPOSE, DAN CARA MENGGANTINYA
-- ============================================================================
--
-- Report Definition MENAMPILKAN dua properti yang tidak ada kolomnya. Itu sah di Pega:
-- hanya penyaring dan pengurutan yang menuntut kolom; nilai yang sekadar ditampilkan dapat
-- dibaca Pega dari BLOB tiap baris. Kita tidak dapat membaca BLOB.
--
--   .ClaimData.ClaimNo    -> tidak ada kolomnya. Diganti `PYID`, dan itu BUKAN pendekatan:
--                            layar Pega menampilkan `PNC-1546` dan `PNC-1729` pada kolom
--                            "Nomor Klaim", yaitu nilai `PYID` keduanya.
--
--   .Policy.TheInsured    -> tidak ada kolomnya. Kolom `INSUREDNAME` pada tabel ini KOSONG
--                            pada kedua baris uji, sehingga bukan penggantinya. Diambil dari
--                            `POOLDATA.T_GENERAL.THEINSURED`, tabel polis — sumber yang sama
--                            dipakai `RDB List/GetDataOutstandingperCabangExport-SQL.xml`.
--
-- Penggantinya dijembatani `POOLDATA.T_CLAIM_PNC` karena tabel kerja Pega tidak menyimpan
-- `PRODKE`, sedangkan polis dikenali oleh `NOPOLIS` + `PRODKE`.
--
--
-- ============================================================================
-- DUA GABUNGAN, KEDUANYA LEFT
-- ============================================================================
--
--   PC_ASM_FW_GCNMFW_WORK.PZINSKEY  ->  T_CLAIM_PNC.CLAIMID     (dibaca dari
--                                       RDB List/BroswseKlaimByRegisterDate-SQL.xml)
--   T_CLAIM_PNC.NOPOLIS + PRODKE    ->  T_GENERAL.NOPOLIS + PRODKE
--
-- Keduanya LEFT, bukan INNER. Baris yang klaimnya tidak ditemukan di tabel bisnis TETAP
-- TAMPIL; yang hilang hanya nama pesertanya. Gabungan INNER akan membuang pekerjaannya
-- diam-diam, dan pekerjaan yang hilang tanpa jejak jauh lebih mahal daripada satu sel yang
-- kosong.
--
--
-- ============================================================================
-- PENYARING B DIPERLUAS, DAN ITU KONSEKUENSI JALUR TULIS
-- ============================================================================
--
-- Pega menyaring satu kolom: `TANGGALDOKLENGKAP` pada tabel kerjanya sendiri. Di sini
-- penyaringnya DUA — baris hilang bila salah satu dari kedua kolom tanggal terisi:
--
--   w.TANGGALDOKLENGKAP IS NULL   diisi Pega saat kasusnya disimpan
--   c.TGLDOKLENGKAP     IS NULL   diisi APLIKASI INI saat Submit ditekan
--
-- Sebabnya: aplikasi ini sengaja TIDAK menulis ke tabel engine Pega. Pega menyimpan nilai
-- sebenarnya di BLOB kasus lalu menyalinnya ke kolom; menulis kolomnya langsung berarti
-- nilai itu akan tertimpa tanpa satu pun tanda begitu Pega menyimpan kasus itu lagi — dan
-- kedua klaim uji berstatus `New`, yaitu masih berjalan.
--
-- Yang ditulis karena itu hanya `T_CLAIM_PNC.TGLDOKLENGKAP`, tabel bisnis. Penyaring kedua
-- inilah yang membuat barisnya tetap hilang seketika dari layar.
--
-- Harganya satu, dan ia TERLIHAT: selama masa paralel, layar TKA di Pega masih menampilkan
-- klaim itu sebagai belum lengkap sampai Pega menyinkronkan. Ketidakcocokan yang terlihat
-- jauh lebih murah daripada data yang hilang tanpa jejak.
--
--
-- ============================================================================
-- URUTAN DAFTAR
-- ============================================================================
--
-- `ORDER BY w.REGISTERDATE_1` — kolom yang SAMA dengan yang Report Definition pakai, menaik,
-- sehingga yang paling lama menunggu tampil lebih dulu.
--
-- Kolomnya `VARCHAR2(32)` berisi tanggal berformat `yyyymmdd` (terverifikasi: `20230510`,
-- `20240319`). Format itu **berlebar tetap dan berurut secara leksikografis sama dengan
-- urutan kronologisnya**, sehingga mengurutkannya sebagai teks memberi hasil yang benar
-- tanpa penguraian apa pun di dalam SQL.
--
-- `PYID` menjadi pemutus di ujung: tanpa kolom unik di akhir, dua baris bertanggal sama
-- dapat bertukar tempat antar pemuatan.
--
-- SUMBER BARU (2026-10-08): urutannya kini `c.REGISTERDATE` (DATE, menaik, NULL di akhir)
-- lalu `x.IDPEGA` sebagai pemutus unik. Tanggal di `T_CLAIM_PNC` sama dengan `REGISTERDATE_1`
-- Pega pada 1.248 kasus, berbeda pada 35, dan kosong pada 121 (seluruh kasus PNC); pada 11
-- kasus TKA yang terbaca keduanya, seluruhnya sama.
--
-- # Rangkaian FROM yang dipakai bersama
--
-- Setiap kueri pembaca berangkat dari snapshot TERAKHIR `JSON_KLAIM` (alias `x`), lalu LEFT
-- JOIN ke klaim dan polis. LEFT, bukan INNER: kasus yang klaimnya belum dikonversi ke
-- `T_CLAIM_PNC` tetap tampil dengan `CLAIM_KEY` kosong, persis perilaku versi lama.
-- Alias kolom di dalam subkueri ditulis TANPA kata `AS` — `query_test.go` membaca alias
-- `AS` sebagai daftar kolom hasil.
--
-- # Penyaring status DI DALAM subkueri, dan kenapa
--
-- `JSON_VALUE` atas BLOB adalah bagian yang mahal: diukur 2026-10-08, kueri daftar yang
-- mengurai seluruh 16.257 snapshot butuh ±4,8 detik. Karena itu snapshot milik klaim yang
-- SUDAH lengkap dokumennya atau SUDAH selesai dibuang lebih dulu lewat `NOT EXISTS` —
-- penyaringnya per kasus, sehingga nomor urut `RN` tidak berubah — dan penguraian JSON
-- hanya dikerjakan pada sisanya (±3 detik; versi lama 0,24 detik). Penyaring status
-- `:1` karena itu kini hidup di dalam subkueri; urutan bind tidak berubah.
--
-- Bentuk `NOT EXISTS (… TGLDOKLENGKAP IS NOT NULL OR STATUSWORK = :1)` setara dengan
-- `c.TGLDOKLENGKAP IS NULL AND (c.STATUSWORK IS NULL OR c.STATUSWORK <> :1)` karena
-- `CLAIMID` unik; kasus yang belum punya baris klaim lolos — itulah kasus yatim.


-- name: tka_inbox_list
--
-- Seluruh klaim TKA yang tanggal kelengkapan dokumennya belum diisi.
--
-- FETCH FIRST :2 ROWS ONLY memotong pada MaxRows. Pemanggil meminta SATU baris lebih banyak
-- daripada yang akan dikirim, supaya keberadaan baris ke-(N+1) membuktikan hasilnya
-- terpotong. Itu yang membuat pemotongan di sini DINYATAKAN, berbeda dari
-- `pyMaxRecords = 500` sistem lama yang memotong dalam diam.
SELECT x.IDPEGA                                    AS REFERENCE,
       c.CLAIMID                                   AS CLAIM_KEY,
       REPLACE(x.IDPEGA, 'ASM-FW-GCNMFW-WORK ', '') AS CLAIM_NUMBER,
       c.NOPOLIS                                   AS POLICY_NUMBER,
       c.QQNAME                                    AS INSURED_NAME,
       g.THEINSURED                                AS PARTICIPANT_NAME,
       c.DATEOFLOSS                                AS DATE_OF_LOSS,
       c.REGISTERDATE                              AS REGISTERED_ON
  FROM (SELECT j.IDPEGA,
               j.DATA_JSONBLOB,
               ROW_NUMBER() OVER (PARTITION BY j.IDPEGA
                                  ORDER BY j.TGL_INPUT DESC NULLS LAST,
                                           j.TGL_KONVERSI DESC NULLS LAST) RN
          FROM POOLDATA.JSON_KLAIM j
         WHERE NOT EXISTS (SELECT 1
                             FROM POOLDATA.T_CLAIM_PNC c0
                            WHERE c0.CLAIMID = j.IDPEGA
                              AND (c0.TGLDOKLENGKAP IS NOT NULL OR c0.STATUSWORK = :1))) x
  LEFT JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = x.IDPEGA
  LEFT JOIN POOLDATA.T_GENERAL g
    ON g.NOPOLIS = c.NOPOLIS
   AND g.PRODKE = c.PRODKE
 WHERE x.RN = 1
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TKA') = '1'
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TanggalDokLengkap') IS NULL
   AND c.TGLDOKLENGKAP IS NULL
 ORDER BY c.REGISTERDATE, x.IDPEGA
 FETCH FIRST :2 ROWS ONLY

-- name: tka_inbox_search
--
-- Sama dengan tka_inbox_list, ditambah penyaring kata kunci.
--
-- Kueri TERSENDIRI, bukan satu kueri yang klausanya ditempel saat kata kuncinya ada. Kueri
-- yang berubah bentuk menurut masukan adalah kueri yang tidak dapat dibaca utuh oleh siapa
-- pun, dan itu justru pola yang membuat `{ASIS:...}` warisan berbahaya.
--
-- EMPAT kolom dicari — keempat kolom teks yang mengidentifikasi pekerjaan. Date Of Loss dan
-- Aging tidak ikut: keduanya tanggal, dan mencocokkannya sebagai teks menuntut pemformatan
-- di dalam SQL yang `D-20` larang.
--
-- Kata kuncinya sudah dibungkus tanda persen oleh pemanggil, bukan di sini: menempelkannya
-- di dalam teks SQL berarti merangkai nilai ke dalam pernyataan.
--
-- EMPAT parameter berbeda untuk nilai yang sama, bukan satu yang dipakai ulang. Oracle
-- mengizinkan pemakaian ulang, tetapi tidak semua driver memetakan parameter bernomor ke
-- posisi argumen dengan cara yang sama — dan `D-20` menuntut kueri ini berjalan sama di
-- kedua basis data.
--
-- ESCAPE '\' disebut eksplisit karena Oracle tidak punya karakter pelolos bawaan pada LIKE.
SELECT x.IDPEGA                                    AS REFERENCE,
       c.CLAIMID                                   AS CLAIM_KEY,
       REPLACE(x.IDPEGA, 'ASM-FW-GCNMFW-WORK ', '') AS CLAIM_NUMBER,
       c.NOPOLIS                                   AS POLICY_NUMBER,
       c.QQNAME                                    AS INSURED_NAME,
       g.THEINSURED                                AS PARTICIPANT_NAME,
       c.DATEOFLOSS                                AS DATE_OF_LOSS,
       c.REGISTERDATE                              AS REGISTERED_ON
  FROM (SELECT j.IDPEGA,
               j.DATA_JSONBLOB,
               ROW_NUMBER() OVER (PARTITION BY j.IDPEGA
                                  ORDER BY j.TGL_INPUT DESC NULLS LAST,
                                           j.TGL_KONVERSI DESC NULLS LAST) RN
          FROM POOLDATA.JSON_KLAIM j
         WHERE NOT EXISTS (SELECT 1
                             FROM POOLDATA.T_CLAIM_PNC c0
                            WHERE c0.CLAIMID = j.IDPEGA
                              AND (c0.TGLDOKLENGKAP IS NOT NULL OR c0.STATUSWORK = :1))) x
  LEFT JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = x.IDPEGA
  LEFT JOIN POOLDATA.T_GENERAL g
    ON g.NOPOLIS = c.NOPOLIS
   AND g.PRODKE = c.PRODKE
 WHERE x.RN = 1
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TKA') = '1'
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TanggalDokLengkap') IS NULL
   AND c.TGLDOKLENGKAP IS NULL
   AND (UPPER(TRIM(REPLACE(x.IDPEGA, 'ASM-FW-GCNMFW-WORK ', ''))) LIKE :2 ESCAPE '\'
     OR UPPER(TRIM(c.NOPOLIS)) LIKE :3 ESCAPE '\'
     OR UPPER(TRIM(c.QQNAME)) LIKE :4 ESCAPE '\'
     OR UPPER(TRIM(g.THEINSURED)) LIKE :5 ESCAPE '\')
 ORDER BY c.REGISTERDATE, x.IDPEGA
 FETCH FIRST :6 ROWS ONLY

-- name: tka_inbox_find_one
--
-- Membaca satu baris yang akan diisi, beserta kunci klaimnya.
--
-- Dijalankan DI DALAM transaksi pengisian, sebelum UPDATE. Surel pemberitahuan memuat lima
-- nilai milik baris ini, dan barisnya lenyap dari daftar begitu terisi — membacanya sesudah
-- penulisan karena itu mustahil.
--
-- # TANPA `FOR UPDATE`, dan itu keputusan yang disengaja
--
-- Kueri ini menyentuh `T_CLAIM_PNC`, tabel yang juga ditulis prosedur konversi Pega.
-- Menguncinya berarti menahan baris yang sedang dilayani aplikasi lama, dan kunci yang
-- ditahan permintaan kita dapat menghentikan konversi yang berjalan di atas kasus yang sama.
-- (Sebelum 2026-10-08 kueri ini membaca tabel objek kerja Pega; alasannya tetap sama.)
--
-- Nomor kasus dicocokkan dengan `IDPEGA` tanpa awalan kelasnya — padanan `PYID` Pega.
--
-- Penguncian tidak dibutuhkan di sini: pengaman pengisian ganda ada pada UPDATE-nya sendiri,
-- yang menyertakan `TGLDOKLENGKAP IS NULL` dan memeriksa jumlah baris terpengaruhnya. Dua
-- permintaan bersamaan hanya membuat satu di antaranya menyentuh satu baris; yang kedua
-- menyentuh nol dan ditolak.
SELECT x.IDPEGA                                    AS REFERENCE,
       c.CLAIMID                                   AS CLAIM_KEY,
       REPLACE(x.IDPEGA, 'ASM-FW-GCNMFW-WORK ', '') AS CLAIM_NUMBER,
       c.NOPOLIS                                   AS POLICY_NUMBER,
       c.QQNAME                                    AS INSURED_NAME,
       g.THEINSURED                                AS PARTICIPANT_NAME,
       c.DATEOFLOSS                                AS DATE_OF_LOSS,
       c.REGISTERDATE                              AS REGISTERED_ON
  FROM (SELECT j.IDPEGA,
               j.DATA_JSONBLOB,
               ROW_NUMBER() OVER (PARTITION BY j.IDPEGA
                                  ORDER BY j.TGL_INPUT DESC NULLS LAST,
                                           j.TGL_KONVERSI DESC NULLS LAST) RN
          FROM POOLDATA.JSON_KLAIM j
         WHERE NOT EXISTS (SELECT 1
                             FROM POOLDATA.T_CLAIM_PNC c0
                            WHERE c0.CLAIMID = j.IDPEGA
                              AND (c0.TGLDOKLENGKAP IS NOT NULL OR c0.STATUSWORK = :1))) x
  LEFT JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = x.IDPEGA
  LEFT JOIN POOLDATA.T_GENERAL g
    ON g.NOPOLIS = c.NOPOLIS
   AND g.PRODKE = c.PRODKE
 WHERE x.RN = 1
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TKA') = '1'
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TanggalDokLengkap') IS NULL
   AND c.TGLDOKLENGKAP IS NULL
   AND TRIM(REPLACE(x.IDPEGA, 'ASM-FW-GCNMFW-WORK ', '')) = :2

-- name: tka_claim_set_document_date
--
-- Menuliskan tanggal kelengkapan dokumen ke klaim yang sebenarnya.
--
-- Kolomnya `TGLDOKLENGKAP` pada `POOLDATA.T_CLAIM_PNC`; pemetaannya dari properti Pega
-- terbaca di `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc`, yang mengambil kunci JSON
-- `TanggalDokLengkap` ke dalam kolom itu.
--
-- Penyaringnya memakai `CLAIMID`, bukan `CLAIMNO`: `CLAIMID` sama dengan `JSON_KLAIM.IDPEGA`
-- (dulu `PZINSKEY`) dan itulah kunci yang barusan dibaca kueri daftar. Memakai nomor klaim
-- berarti mencocokkan teks yang keunikannya tidak dibuktikan DDL mana pun (`R-08`).
--
-- SUMBER BARU (2026-10-08): pernyataan ini TIDAK berubah. WHERE-nya diverifikasi di Oracle
-- dev dengan SELECT setara (`CLAIMID = :2 AND TGLDOKLENGKAP IS NULL`), tanpa menulis.
--
-- # `TGLDOKLENGKAP IS NULL` pada WHERE adalah pengaman pengisian ganda
--
-- Ia bukan pengulangan penyaring daftar. Bersama pemeriksaan jumlah baris terpengaruh, ia
-- yang membuat dua permintaan bersamaan tidak dapat sama-sama berhasil — dan karena itu
-- surel ganda tidak dapat terjadi, tanpa perlu kunci idempotensi terpisah.
--
-- # Snapshot kasus Pega TIDAK ikut ditulis
--
-- Pega menyimpan nilai sebenarnya di BLOB kasus (snapshotnya di `JSON_KLAIM`). Menulis
-- salinannya dari luar akan tertimpa tanpa satu pun tanda begitu Pega menyimpan kasusnya
-- lagi. Lihat kepala berkas ini.
--
-- Penulisan ini tetap menuntut serah-terima kepemilikan tulis (`P-1`, `D-63`): permintaan
-- tertulis, persetujuan Work Owner, pelaksanaan DBA.
UPDATE POOLDATA.T_CLAIM_PNC
   SET TGLDOKLENGKAP = :1
 WHERE CLAIMID = :2
   AND TGLDOKLENGKAP IS NULL

-- name: tka_inbox_check_table
--
-- Membuktikan ketiga tabel beserta kolom yang DIBACA modul ini ada dan dapat dibaca.
--
-- Dipakai `claimpnc -periksa`. FETCH FIRST 0 ROWS ONLY: yang diperiksa adalah apakah
-- pernyataannya dapat diurai dan dijalankan, bukan isinya — menarik satu baris berarti
-- membaca data nasabah tanpa keperluan.
--
-- SUMBER BARU (2026-10-08): ketiga tabelnya kini berada di skema POOLDATA (`JSON_KLAIM`,
-- `T_CLAIM_PNC`, `T_GENERAL`); pemeriksaan ini sekaligus membuktikan `JSON_VALUE` atas
-- `DATA_JSONBLOB` dapat diurai.
SELECT x.IDPEGA                                    AS REFERENCE,
       c.CLAIMID                                   AS CLAIM_KEY,
       REPLACE(x.IDPEGA, 'ASM-FW-GCNMFW-WORK ', '') AS CLAIM_NUMBER,
       c.NOPOLIS                                   AS POLICY_NUMBER,
       c.QQNAME                                    AS INSURED_NAME,
       g.THEINSURED                                AS PARTICIPANT_NAME,
       c.DATEOFLOSS                                AS DATE_OF_LOSS,
       c.REGISTERDATE                              AS REGISTERED_ON
  FROM (SELECT j.IDPEGA,
               j.DATA_JSONBLOB,
               ROW_NUMBER() OVER (PARTITION BY j.IDPEGA
                                  ORDER BY j.TGL_INPUT DESC NULLS LAST,
                                           j.TGL_KONVERSI DESC NULLS LAST) RN
          FROM POOLDATA.JSON_KLAIM j) x
  LEFT JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = x.IDPEGA
  LEFT JOIN POOLDATA.T_GENERAL g
    ON g.NOPOLIS = c.NOPOLIS
   AND g.PRODKE = c.PRODKE
 WHERE x.RN = 1
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TKA') = '1'
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TanggalDokLengkap') IS NULL
 FETCH FIRST 0 ROWS ONLY

-- name: tka_inbox_check_claim_column
--
-- Membuktikan kolom sasaran tulis ada dan dapat dibaca.
--
-- Ia tidak menulis apa pun — yang diperiksa hanya keberadaan kolomnya. Hak TULIS-nya tidak
-- dapat diperiksa tanpa benar-benar menulis, dan itu tidak dilakukan terhadap basis data
-- yang melayani produksi.
SELECT c.TGLDOKLENGKAP AS CLAIM_DOCUMENT_DATE
  FROM POOLDATA.T_CLAIM_PNC c
 FETCH FIRST 0 ROWS ONLY

-- name: tka_inbox_count_waiting
--
-- Cacah pekerjaan yang menunggu, tanpa dipotong.
--
-- Dipakai `claimpnc -periksa`. Angkanya menjawab pertanyaan yang tidak dapat dijawab layar
-- ketika hasilnya terpotong: BERAPA SEBENARNYA yang menunggu. Penyaringnya sama persis
-- dengan tka_inbox_list.
--
-- SUMBER BARU (2026-10-08): angka ini TIDAK LAGI sama dengan jumlah baris layar Pega —
-- Pega menyaring kolom `TKA_1` yang hanya terisi pada sebagian kasus. Lihat kepala berkas.
SELECT COUNT(*)
  FROM (SELECT j.IDPEGA,
               j.DATA_JSONBLOB,
               ROW_NUMBER() OVER (PARTITION BY j.IDPEGA
                                  ORDER BY j.TGL_INPUT DESC NULLS LAST,
                                           j.TGL_KONVERSI DESC NULLS LAST) RN
          FROM POOLDATA.JSON_KLAIM j
         WHERE NOT EXISTS (SELECT 1
                             FROM POOLDATA.T_CLAIM_PNC c0
                            WHERE c0.CLAIMID = j.IDPEGA
                              AND (c0.TGLDOKLENGKAP IS NOT NULL OR c0.STATUSWORK = :1))) x
  LEFT JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = x.IDPEGA
 WHERE x.RN = 1
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TKA') = '1'
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TanggalDokLengkap') IS NULL
   AND c.TGLDOKLENGKAP IS NULL

-- name: tka_inbox_count_pega_only
--
-- Cacah pekerjaan yang MASIH tampil di layar Pega tetapi SUDAH dikerjakan lewat aplikasi
-- ini.
--
-- # Angka inilah harga dari tidak menulis ke data kasus Pega
--
-- Aplikasi ini mengisi `T_CLAIM_PNC.TGLDOKLENGKAP`; Pega memegang `TanggalDokLengkap` di
-- BLOB kasusnya (snapshotnya `JSON_KLAIM`). Selama Pega belum menyinkronkan keduanya,
-- barisnya hilang dari layar kita dan tetap ada di layar Pega.
--
-- Itu perbedaan yang DISENGAJA dan sudah dinyatakan di kepala berkas ini — tetapi ia harus
-- dapat diukur, bukan sekadar diketahui. Bila angkanya menumpuk, petugas Pega akan mengisi
-- ulang tanggal yang sebenarnya sudah diisi.
--
-- SUMBER BARU (2026-10-08): sisi Pega kini dibaca dari snapshot `JSON_KLAIM` terakhir,
-- bukan dari kolom tabel objek kerja.
SELECT COUNT(*)
  FROM (SELECT j.IDPEGA,
               j.DATA_JSONBLOB,
               ROW_NUMBER() OVER (PARTITION BY j.IDPEGA
                                  ORDER BY j.TGL_INPUT DESC NULLS LAST,
                                           j.TGL_KONVERSI DESC NULLS LAST) RN
          FROM POOLDATA.JSON_KLAIM j
         WHERE EXISTS (SELECT 1
                         FROM POOLDATA.T_CLAIM_PNC c0
                        WHERE c0.CLAIMID = j.IDPEGA
                          AND c0.TGLDOKLENGKAP IS NOT NULL
                          AND (c0.STATUSWORK IS NULL OR c0.STATUSWORK <> :1))) x
  INNER JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = x.IDPEGA
 WHERE x.RN = 1
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TKA') = '1'
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TanggalDokLengkap') IS NULL
   AND c.TGLDOKLENGKAP IS NOT NULL

-- name: tka_inbox_count_orphan_claim
--
-- Cacah pekerjaan TKA yang klaimnya tidak ada di `POOLDATA.T_CLAIM_PNC`.
--
-- Baris seperti ini **TETAP TAMPIL** di daftar — dan itu bukan kelalaian melainkan akibat
-- langsung dari gabungan LEFT: bila tidak ada pasangannya, `c.TGLDOKLENGKAP` bernilai NULL,
-- sehingga penyaring `IS NULL` justru terpenuhi. Perilakunya karena itu sama dengan Pega,
-- yang juga menampilkannya.
--
-- Yang berbeda adalah nasibnya saat Submit: tidak ada baris klaim yang dapat diperbarui,
-- sehingga pengisiannya ditolak dengan `ErrClaimMissing`. Layar mengetahuinya lebih dulu
-- lewat kolom `CLAIM_KEY` yang kosong, dan mematikan isian pada baris itu alih-alih
-- membiarkan pengguna menekan tombol yang sudah pasti gagal.
--
-- Bila angkanya besar, yang perlu ditinjau adalah kelengkapan `T_CLAIM_PNC`, bukan layarnya.
--
-- SUMBER BARU (2026-10-08): kasusnya kini dikenali dari snapshot `JSON_KLAIM`. Tanpa baris
-- klaim, status pekerjaannya tidak diketahui, sehingga penyaring status terpenuhi lewat
-- cabang NULL — persis sebagaimana baris itu tampil di tka_inbox_list.
SELECT COUNT(*)
  FROM (SELECT j.IDPEGA,
               j.DATA_JSONBLOB,
               ROW_NUMBER() OVER (PARTITION BY j.IDPEGA
                                  ORDER BY j.TGL_INPUT DESC NULLS LAST,
                                           j.TGL_KONVERSI DESC NULLS LAST) RN
          FROM POOLDATA.JSON_KLAIM j
         WHERE NOT EXISTS (SELECT 1
                             FROM POOLDATA.T_CLAIM_PNC c0
                            WHERE c0.CLAIMID = j.IDPEGA
                              AND (c0.TGLDOKLENGKAP IS NOT NULL OR c0.STATUSWORK = :1))) x
  LEFT JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = x.IDPEGA
 WHERE x.RN = 1
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TKA') = '1'
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TanggalDokLengkap') IS NULL
   AND c.CLAIMID IS NULL

-- name: tka_inbox_count_missing_participant
--
-- Cacah pekerjaan menunggu yang nama pesertanya tidak ditemukan di tabel polis.
--
-- `.Policy.TheInsured` tidak di-expose sebagai kolom pada tabel kerja Pega, sehingga
-- nilainya diambil dari `POOLDATA.T_GENERAL`. Bila angkanya besar, penggantinya keliru dan
-- kolom "Nama Peserta" akan kosong bagi sebagian besar daftar — keadaan yang hanya dapat
-- diketahui dari data nyata.
SELECT COUNT(*)
  FROM (SELECT j.IDPEGA,
               j.DATA_JSONBLOB,
               ROW_NUMBER() OVER (PARTITION BY j.IDPEGA
                                  ORDER BY j.TGL_INPUT DESC NULLS LAST,
                                           j.TGL_KONVERSI DESC NULLS LAST) RN
          FROM POOLDATA.JSON_KLAIM j
         WHERE NOT EXISTS (SELECT 1
                             FROM POOLDATA.T_CLAIM_PNC c0
                            WHERE c0.CLAIMID = j.IDPEGA
                              AND (c0.TGLDOKLENGKAP IS NOT NULL OR c0.STATUSWORK = :1))) x
  LEFT JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = x.IDPEGA
  LEFT JOIN POOLDATA.T_GENERAL g
    ON g.NOPOLIS = c.NOPOLIS
   AND g.PRODKE = c.PRODKE
 WHERE x.RN = 1
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TKA') = '1'
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TanggalDokLengkap') IS NULL
   AND c.TGLDOKLENGKAP IS NULL
   AND (g.THEINSURED IS NULL OR TRIM(g.THEINSURED) IS NULL OR TRIM(g.THEINSURED) = '')

-- name: tka_inbox_sample_registered_on
--
-- Dua puluh nilai tanggal registrasi yang berbeda, untuk dilihat manusia.
--
-- SUMBER BARU (2026-10-08): sumbernya kini `T_CLAIM_PNC.REGISTERDATE` bertipe DATE, bukan
-- `REGISTERDATE_1` teks `yyyymmdd` milik tabel objek kerja Pega. Pemindai Go membaca DATE
-- lalu memformatnya `yyyymmdd`, sehingga pemeriksaan `claimpnc -periksa` tetap menerima
-- bentuk yang sama; yang kini diperiksanya adalah apakah tanggalnya terbaca, bukan formatnya.
--
-- Ia tidak mengembalikan data nasabah: tanggal registrasi bukan identitas.
SELECT DISTINCT c.REGISTERDATE AS REGISTERED_ON
  FROM (SELECT j.IDPEGA,
               j.DATA_JSONBLOB,
               ROW_NUMBER() OVER (PARTITION BY j.IDPEGA
                                  ORDER BY j.TGL_INPUT DESC NULLS LAST,
                                           j.TGL_KONVERSI DESC NULLS LAST) RN
          FROM POOLDATA.JSON_KLAIM j) x
  INNER JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = x.IDPEGA
 WHERE x.RN = 1
   AND JSON_VALUE(x.DATA_JSONBLOB, '$.TKA') = '1'
   AND c.REGISTERDATE IS NOT NULL
 FETCH FIRST 20 ROWS ONLY
