-- Kueri modul Outstanding Claim: rincian satu klaim treaty proporsional.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- SELURUH tabel yang dibaca berkas ini milik sistem lama. Tidak ada satu pun pernyataan yang
-- menulis, dan memang tidak boleh ada: Flow Action `OutstandingClaim` di Pega MENYIMPAN
-- kembali objek kerjanya, tetapi selama masa paralel tabel itu dimiliki Pega (`P-1`).
--
-- ============================================================================
-- SATU KUERI UNTUK 97 ISIAN DAN 10 GRID
-- ============================================================================
--
-- Layar ini mengikat hampir seluruh isiannya ke `.ClaimData.*`, satu halaman klipboard yang
-- di basis data tersimpan sebagai SATU dokumen JSON — `POOLDATA.JSON_KLAIM.DATA_JSONBLOB`.
-- Dokumen itu diambil UTUH di sini lalu diurai di Go (document.go), bukan dipetik isian demi
-- isian dengan `JSON_VALUE` dan senarai demi senarai dengan `JSON_TABLE`.
--
-- Tiga alasan, dan ketiganya menyentuh hal yang berbeda:
--
--   SATU PERJALANAN     97 isian dan 10 grid berarti sepuluh `JSON_TABLE` bila dipetik di
--                       sini, masing-masing membaca ulang dokumen yang sama.
--
--   KEGAGALAN TERBACA   jalur yang salah pada `JSON_TABLE` mengembalikan KOSONG tanpa satu
--                       pun galat. Bentuk dokumen ini belum pernah diperiksa — DDL-nya tidak
--                       tersedia (`R-08`) — sehingga jalur yang salah adalah kemungkinan
--                       nyata, bukan kemungkinan teoretis. Diurai di Go, "jalur tidak ada"
--                       dapat dibedakan dari "jalur ada tetapi kosong", dan perbedaan itu
--                       yang membedakan cacat dari data yang memang belum diisi.
--
--   PORTABILITAS        sepuluh `JSON_TABLE` harus berperilaku sama di Oracle 19c dan
--                       PostgreSQL 17 (`D-20`, `D-24`). Satu kolom CLOB yang dibaca apa
--                       adanya tidak menuntut apa pun dari keduanya.
--
-- ============================================================================
-- DUA TABEL, DAN PEMBAGIAN TUGASNYA
-- ============================================================================
--
--   (lama) DATAPEGA.PC_ASM_FW_GCNMFW_WORK  w   KEADAAN objek kerja — nomor, status, pengubah
--   POOLDATA.JSON_KLAIM             b   ISI klaimnya — satu dokumen JSON
--
-- Pembagian itu bukan pilihan gaya: status alur kerja dan petugas pengubah adalah milik
-- objek kerja Pega dan tidak ikut tersimpan di dalam dokumen `.ClaimData`.
--
-- SUMBER BARU (2026-10-08): objek kerja Pega tidak dipakai lagi. Keberadaan klaim dibuktikan
-- dari `JSON_KLAIM.IDPEGA` + `PC_ASSIGN_WORKLIST`/`PC_ASSIGN_WORKBASKET.PXREFOBJECTKEY`; status
-- alur kerja dan petugas pengubah karenanya KOSONG (tidak ada padanan terbukti). Rincian di
-- `find_claim`. Kata "objek kerja" di catatan berikut merujuk perilaku lama.
--
-- Gabungannya `LEFT JOIN`, dengan alasan yang sama seperti di Inbox Claim Treaty Prop: klaim
-- yang belum punya baris di JSON_KLAIM tetap DAPAT DIBUKA, dengan seluruh isian kosong dan
-- nomor klaimnya tetap terbaca. Menggantinya dengan INNER akan membuat klaim seperti itu
-- dijawab "tidak ditemukan" — padahal ia ada, hanya isinya yang belum tersalin.
--
-- Kunci gabungannya `w.PZINSKEY = b.IDPEGA`, kolom yang sama yang dipakai Inbox Claim Treaty
-- Prop (di sana lewat `a.PXREFOBJECTKEY`, yang isinya `PZINSKEY` objek kerja yang ditunjuk).
--
-- ============================================================================
-- KENAPA ADA PENYARING `LIKE '%CLMP%'` PADA KUERI RINCIAN
-- ============================================================================
--
-- Karena `PC_ASM_FW_GCNMFW_WORK` memuat objek kerja SELURUH jenis klaim, bukan hanya treaty
-- proporsional. Tanpa penyaring itu, alamat layar ini dapat diisi nomor klaim PNC biasa dan
-- layar akan menggambarnya dengan susunan treaty — 97 isian yang hampir seluruhnya kosong,
-- tanpa satu pun tanda bahwa yang dibuka adalah jenis klaim yang berbeda.
--
-- Polanya literal, bukan masukan pengguna, sehingga tidak butuh ESCAPE.

-- name: find_claim
-- Rincian satu klaim treaty menurut nomor klaimnya.
--
-- SUMBER BARU (2026-10-08). Objek kerja Pega (`PC_ASM_FW_GCNMFW_WORK`) tidak dipakai lagi
-- (keputusan Work Owner). Klaim treaty (`CLMP-*`) TIDAK ada di `T_CLAIM_PNC` maupun
-- `T_CLAIMLIST_ADMIN`, sehingga keberadaannya kini dibuktikan dari tempat yang memuat kuncinya:
-- `JSON_KLAIM.IDPEGA` (isi klaim) serta `PC_ASSIGN_WORKLIST`/`PC_ASSIGN_WORKBASKET`
-- (`PXREFOBJECTKEY`, tugas yang masih terbuka). Diukur di Oracle dev: ketiganya mencakup 68 dari
-- 69 objek kerja ber-`CLMP` (JSON_KLAIM saja hanya 17); satu yang tidak tercakup berstatus
-- Resolved-Rejected dan kini dijawab "tidak ditemukan". Kuncinya `'ASM-FW-GCNMFW-WORK ' +
-- nomor` — terukur sama dengan `PZINSKEY` pada 91/91 objek treaty.
--
-- Akibat: STATUS_WORK dan LAST_UPDATE_OPERATOR tidak punya padanan terbukti di luar objek
-- kerja (`PXUPDATEOPERATOR` tabel penugasan sama dengan milik objek kerja pada 0/91 baris),
-- sehingga keduanya kini NULL bertipe — layar menampilkannya kosong.
--
-- Bind: :1 :2 :3 nomor klaim yang SAMA (mis. `CLMP-70`) — go-ora mengikat menurut urutan.
SELECT REPLACE(k.CLAIM_KEY, 'ASM-FW-GCNMFW-WORK ', '')               AS CLAIM_ID,
       k.CLAIM_KEY                                               AS REFERENCE,
       CAST(NULL AS VARCHAR2(32))                                AS STATUS_WORK,
       CAST(NULL AS VARCHAR2(128))                               AS LAST_UPDATE_OPERATOR,
       b.DATA_JSONBLOB                                           AS CLAIM_DOCUMENT
  FROM (SELECT j.IDPEGA CLAIM_KEY FROM POOLDATA.JSON_KLAIM j
         WHERE j.IDPEGA = CONCAT('ASM-FW-GCNMFW-WORK ', :1)
        UNION
        SELECT l.PXREFOBJECTKEY FROM DATAPEGA.PC_ASSIGN_WORKLIST l
         WHERE l.PXREFOBJECTKEY = CONCAT('ASM-FW-GCNMFW-WORK ', :2)
        UNION
        SELECT q.PXREFOBJECTKEY FROM DATAPEGA.PC_ASSIGN_WORKBASKET q
         WHERE q.PXREFOBJECTKEY = CONCAT('ASM-FW-GCNMFW-WORK ', :3)) k
       LEFT JOIN POOLDATA.JSON_KLAIM b
              ON k.CLAIM_KEY = b.IDPEGA
 WHERE k.CLAIM_KEY LIKE '%CLMP%'

-- name: check_tables
-- Dipakai perintah `-periksa`: memastikan ketiga tabel DAN gabungannya terbaca dari koneksi
-- yang dipakai.
--
-- Ia tidak menyentuh satu baris pun — yang diperiksa adalah hak baca dan keberadaan
-- tabelnya, bukan isinya. Gabungannya ikut diperiksa karena kegagalan yang paling mungkin
-- terjadi bukan "tabel tidak ada" melainkan "hak baca hanya diberikan pada salah satunya".
--
-- SUMBER BARU (2026-10-08): mengikuti `find_claim` — objek kerja Pega diganti tabel penugasan.
SELECT COUNT(*) AS PROBE
  FROM DATAPEGA.PC_ASSIGN_WORKLIST l
       LEFT JOIN DATAPEGA.PC_ASSIGN_WORKBASKET q
              ON l.PXREFOBJECTKEY = q.PXREFOBJECTKEY
       LEFT JOIN POOLDATA.JSON_KLAIM b
              ON l.PXREFOBJECTKEY = b.IDPEGA
 WHERE 1 = 0
