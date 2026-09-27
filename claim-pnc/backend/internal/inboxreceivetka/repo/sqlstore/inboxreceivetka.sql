-- Kueri modul Inbox Receive TKA.
--
-- TIGA tabel, dan aplikasi ini menulis SATU kolom pada satu di antaranya:
--
--   DATAPEGA.PC_ASM_FW_GCNMFW_WORK  sumber daftar     — hanya DIBACA, milik engine Pega
--   POOLDATA.T_CLAIM_PNC            klaim sebenarnya  — DIBACA, dan satu kolom DITULIS
--   POOLDATA.T_GENERAL              tabel polis       — hanya DIBACA
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


-- name: tka_inbox_list
--
-- Seluruh klaim TKA yang tanggal kelengkapan dokumennya belum diisi.
--
-- FETCH FIRST :2 ROWS ONLY memotong pada MaxRows. Pemanggil meminta SATU baris lebih banyak
-- daripada yang akan dikirim, supaya keberadaan baris ke-(N+1) membuktikan hasilnya
-- terpotong. Itu yang membuat pemotongan di sini DINYATAKAN, berbeda dari
-- `pyMaxRecords = 500` sistem lama yang memotong dalam diam.
SELECT w.PZINSKEY       AS REFERENCE,
       c.CLAIMID        AS CLAIM_KEY,
       w.PYID           AS CLAIM_NUMBER,
       w.POLICYNO       AS POLICY_NUMBER,
       w.QQNAME         AS INSURED_NAME,
       g.THEINSURED     AS PARTICIPANT_NAME,
       w.DATEOFLOSS_1   AS DATE_OF_LOSS,
       w.REGISTERDATE_1 AS REGISTERED_ON
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
  LEFT JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = w.PZINSKEY
  LEFT JOIN POOLDATA.T_GENERAL g
    ON g.NOPOLIS = c.NOPOLIS
   AND g.PRODKE = c.PRODKE
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND w.TKA_1 = '1'
   AND w.TANGGALDOKLENGKAP IS NULL
   AND c.TGLDOKLENGKAP IS NULL
   AND w.PYSTATUSWORK <> :1
 ORDER BY w.REGISTERDATE_1, w.PYID
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
SELECT w.PZINSKEY       AS REFERENCE,
       c.CLAIMID        AS CLAIM_KEY,
       w.PYID           AS CLAIM_NUMBER,
       w.POLICYNO       AS POLICY_NUMBER,
       w.QQNAME         AS INSURED_NAME,
       g.THEINSURED     AS PARTICIPANT_NAME,
       w.DATEOFLOSS_1   AS DATE_OF_LOSS,
       w.REGISTERDATE_1 AS REGISTERED_ON
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
  LEFT JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = w.PZINSKEY
  LEFT JOIN POOLDATA.T_GENERAL g
    ON g.NOPOLIS = c.NOPOLIS
   AND g.PRODKE = c.PRODKE
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND w.TKA_1 = '1'
   AND w.TANGGALDOKLENGKAP IS NULL
   AND c.TGLDOKLENGKAP IS NULL
   AND w.PYSTATUSWORK <> :1
   AND (UPPER(TRIM(w.PYID)) LIKE :2 ESCAPE '\'
     OR UPPER(TRIM(w.POLICYNO)) LIKE :3 ESCAPE '\'
     OR UPPER(TRIM(w.QQNAME)) LIKE :4 ESCAPE '\'
     OR UPPER(TRIM(g.THEINSURED)) LIKE :5 ESCAPE '\')
 ORDER BY w.REGISTERDATE_1, w.PYID
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
-- Kueri ini menyentuh tabel engine Pega. Menguncinya berarti menahan baris yang sedang
-- dilayani aplikasi lama, dan kunci yang ditahan permintaan kita dapat menghentikan alur
-- kerja Pega yang berjalan di atas kasus yang sama.
--
-- Penguncian tidak dibutuhkan di sini: pengaman pengisian ganda ada pada UPDATE-nya sendiri,
-- yang menyertakan `TGLDOKLENGKAP IS NULL` dan memeriksa jumlah baris terpengaruhnya. Dua
-- permintaan bersamaan hanya membuat satu di antaranya menyentuh satu baris; yang kedua
-- menyentuh nol dan ditolak.
SELECT w.PZINSKEY       AS REFERENCE,
       c.CLAIMID        AS CLAIM_KEY,
       w.PYID           AS CLAIM_NUMBER,
       w.POLICYNO       AS POLICY_NUMBER,
       w.QQNAME         AS INSURED_NAME,
       g.THEINSURED     AS PARTICIPANT_NAME,
       w.DATEOFLOSS_1   AS DATE_OF_LOSS,
       w.REGISTERDATE_1 AS REGISTERED_ON
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
  LEFT JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = w.PZINSKEY
  LEFT JOIN POOLDATA.T_GENERAL g
    ON g.NOPOLIS = c.NOPOLIS
   AND g.PRODKE = c.PRODKE
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND w.TKA_1 = '1'
   AND w.TANGGALDOKLENGKAP IS NULL
   AND c.TGLDOKLENGKAP IS NULL
   AND w.PYSTATUSWORK <> :1
   AND TRIM(w.PYID) = :2

-- name: tka_claim_set_document_date
--
-- Menuliskan tanggal kelengkapan dokumen ke klaim yang sebenarnya.
--
-- Kolomnya `TGLDOKLENGKAP` pada `POOLDATA.T_CLAIM_PNC`; pemetaannya dari properti Pega
-- terbaca di `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc`, yang mengambil kunci JSON
-- `TanggalDokLengkap` ke dalam kolom itu.
--
-- Penyaringnya memakai `CLAIMID`, bukan `CLAIMNO`: `CLAIMID` sama dengan `PZINSKEY` dan
-- itulah kunci yang barusan dibaca dari tabel kerja Pega. Memakai nomor klaim berarti
-- mencocokkan teks yang keunikannya tidak dibuktikan DDL mana pun (`R-08`).
--
-- # `TGLDOKLENGKAP IS NULL` pada WHERE adalah pengaman pengisian ganda
--
-- Ia bukan pengulangan penyaring daftar. Bersama pemeriksaan jumlah baris terpengaruh, ia
-- yang membuat dua permintaan bersamaan tidak dapat sama-sama berhasil — dan karena itu
-- surel ganda tidak dapat terjadi, tanpa perlu kunci idempotensi terpisah.
--
-- # Tabel engine Pega TIDAK ikut ditulis
--
-- Pega menyimpan nilai sebenarnya di BLOB kasus dan menyalinnya ke kolom
-- `PC_ASM_FW_GCNMFW_WORK.TANGGALDOKLENGKAP`. Menulis kolom itu langsung akan tertimpa tanpa
-- satu pun tanda begitu Pega menyimpan kasusnya lagi. Lihat kepala berkas ini.
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
-- Pemeriksaan ini berharga justru karena gabungannya menyentuh DUA skema sekaligus,
-- DATAPEGA dan POOLDATA. Hak baca yang kurang pada salah satunya baru terlihat saat
-- pengguna membuka layar — kecuali diperiksa lebih dulu di sini.
SELECT w.PZINSKEY       AS REFERENCE,
       c.CLAIMID        AS CLAIM_KEY,
       w.PYID           AS CLAIM_NUMBER,
       w.POLICYNO       AS POLICY_NUMBER,
       w.QQNAME         AS INSURED_NAME,
       g.THEINSURED     AS PARTICIPANT_NAME,
       w.DATEOFLOSS_1   AS DATE_OF_LOSS,
       w.REGISTERDATE_1 AS REGISTERED_ON
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
  LEFT JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = w.PZINSKEY
  LEFT JOIN POOLDATA.T_GENERAL g
    ON g.NOPOLIS = c.NOPOLIS
   AND g.PRODKE = c.PRODKE
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
-- ketika hasilnya terpotong: BERAPA SEBENARNYA yang menunggu. Ia juga angka yang dapat
-- dibandingkan langsung dengan jumlah baris pada layar Pega — keduanya kini memakai
-- penyaring yang sama.
SELECT COUNT(*)
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
  LEFT JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = w.PZINSKEY
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND w.TKA_1 = '1'
   AND w.TANGGALDOKLENGKAP IS NULL
   AND c.TGLDOKLENGKAP IS NULL
   AND w.PYSTATUSWORK <> :1

-- name: tka_inbox_count_pega_only
--
-- Cacah pekerjaan yang MASIH tampil di layar Pega tetapi SUDAH dikerjakan lewat aplikasi
-- ini.
--
-- # Angka inilah harga dari tidak menulis ke tabel engine Pega
--
-- Aplikasi ini mengisi `T_CLAIM_PNC.TGLDOKLENGKAP`; Pega menyaring
-- `PC_ASM_FW_GCNMFW_WORK.TANGGALDOKLENGKAP`. Selama Pega belum menyinkronkan keduanya,
-- barisnya hilang dari layar kita dan tetap ada di layar Pega.
--
-- Itu perbedaan yang DISENGAJA dan sudah dinyatakan di kepala berkas ini — tetapi ia harus
-- dapat diukur, bukan sekadar diketahui. Bila angkanya menumpuk, petugas Pega akan mengisi
-- ulang tanggal yang sebenarnya sudah diisi.
SELECT COUNT(*)
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
  INNER JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = w.PZINSKEY
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND w.TKA_1 = '1'
   AND w.TANGGALDOKLENGKAP IS NULL
   AND c.TGLDOKLENGKAP IS NOT NULL
   AND w.PYSTATUSWORK <> :1

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
SELECT COUNT(*)
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND w.TKA_1 = '1'
   AND w.TANGGALDOKLENGKAP IS NULL
   AND w.PYSTATUSWORK <> :1
   AND NOT EXISTS (SELECT 1
                     FROM POOLDATA.T_CLAIM_PNC c
                    WHERE c.CLAIMID = w.PZINSKEY)

-- name: tka_inbox_count_missing_participant
--
-- Cacah pekerjaan menunggu yang nama pesertanya tidak ditemukan di tabel polis.
--
-- `.Policy.TheInsured` tidak di-expose sebagai kolom pada tabel kerja Pega, sehingga
-- nilainya diambil dari `POOLDATA.T_GENERAL`. Bila angkanya besar, penggantinya keliru dan
-- kolom "Nama Peserta" akan kosong bagi sebagian besar daftar — keadaan yang hanya dapat
-- diketahui dari data nyata.
SELECT COUNT(*)
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
  LEFT JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = w.PZINSKEY
  LEFT JOIN POOLDATA.T_GENERAL g
    ON g.NOPOLIS = c.NOPOLIS
   AND g.PRODKE = c.PRODKE
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND w.TKA_1 = '1'
   AND w.TANGGALDOKLENGKAP IS NULL
   AND c.TGLDOKLENGKAP IS NULL
   AND w.PYSTATUSWORK <> :1
   AND (g.THEINSURED IS NULL OR TRIM(g.THEINSURED) IS NULL OR TRIM(g.THEINSURED) = '')

-- name: tka_inbox_sample_registered_on
--
-- Dua puluh nilai `REGISTERDATE_1` yang berbeda, untuk dilihat manusia.
--
-- Kolomnya `VARCHAR2(32)` dan isinya terverifikasi berformat `yyyymmdd` pada dua baris uji
-- (`20230510`, `20240319`). Kueri ini memastikan bentuk itu berlaku pada seluruh daftar,
-- bukan hanya pada dua baris — karena kolom teks dapat memuat apa saja, dan penguraiannya
-- di Go bergantung pada bentuk itu.
--
-- Ia tidak mengembalikan data nasabah: tanggal registrasi bukan identitas.
SELECT DISTINCT w.REGISTERDATE_1 AS REGISTERED_ON
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND w.TKA_1 = '1'
   AND w.REGISTERDATE_1 IS NOT NULL
 FETCH FIRST 20 ROWS ONLY
