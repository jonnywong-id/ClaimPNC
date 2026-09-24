-- Kueri Inbox Auto Claim — {{TABEL}} dan POOLDATA.M_AUTO_CLAIM_PNC.
--
-- ============================================================================
-- DISELARASKAN KE KUERI PEGA YANG ASLI — 2026-09-19
-- ============================================================================
--
-- Versi pertama berkas ini adalah REKONSTRUKSI, karena keenam kueri penggerak layar ini
-- tidak ada di export. Kueri aslinya diterima 2026-09-19 dan seluruh kueri di bawah sudah
-- diselaraskan kepadanya. Sumbernya sekarang disebut per kueri sebagai `berkas:tag`.
--
-- Rekonstruksi itu ternyata BENAR pada bagian yang paling berisiko — kedelapan pemetaan
-- kolom grid dan ketiga ambang hitungan — dan MELESET pada tujuh hal yang kini diperbaiki.
-- Riwayat lengkapnya di docs/keputusan-implementasi.md §18.
--
-- ============================================================================
-- Empat aturan yang mengikat seluruh berkas ini
-- ============================================================================
--
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding. Kueri asli menyisipkan penyaringnya lewat
--      {ASIS:...} — celah injeksi yang TIDAK dibawa; penggantinya kueri terpisah.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR untuk tampilan — SQL harus berjalan
--      sama di Oracle 19c dan PostgreSQL 17+ (D-20). Kueri asli memakai ROWNUM dan
--      TO_CHAR; keduanya diganti OFFSET/FETCH dan pemformatan di Go.
--   4. Tanpa pemanggilan stored procedure (D-02). Kueri asli menyisipkan lewat
--      POOLDATA.INSERT_AUTOCLAIM; di sini penyisipannya ditulis langsung.
--
-- ============================================================================
-- DDL yang sudah diterima (Database/CREATE_TABLE_1.sql) dan akibatnya
-- ============================================================================
--
--   BATCH        NUMBER         -> nomor batch memang angka
--   INISIALID    VARCHAR2(20)   -> BUKAN CHAR; TRIM() tidak diperlukan dan justru
--                                  mematikan index TMP_BATCH_AUTO_CLAIM_IDX
--   TGLKEJADIAN  VARCHAR2(50)   -> tanggal memang TEKS
--   TGLLAPOR     VARCHAR2(50)
--   TGLPROSES    TIMESTAMP(6)   -> BAGIAN DARI PRIMARY KEY; wajib diisi saat menyisipkan
--   NILAIKLAIM   NUMBER
--   CURRENCY     VARCHAR2(15)   -> berisi ID, bukan kode; di-lookup ke POOLDATA.CURRENCY
--   OBJECTNAME, FLAGTIDAKBAYAR  -> dua kolom yang tidak diketahui saat rekonstruksi
--
--   PRIMARY KEY (NOPOLIS, TGLPROSES, TGLKEJADIAN)
--   INDEX (BATCH, {{KOLOM}}, NOPOLIS, PRODKE, TGLKEJADIAN, TMP_MESSAGE, IDPEGA)


-- name: auto_claim_batch_list
--
-- Daftar batch — delapan kolom grid.
--
-- Asal: InboxAutoClaim/BrowseClaimSPKAutoClaim-SQL.xml
--
-- Pemetaan kolomnya diikuti apa adanya:
--
--   INISIALID      AS "CaseID"                  -> KODE
--   BATCH          AS "CauseOfLoss"             -> Batch
--   USERINPUT      AS "AnaylstRemarks"          -> User Upload
--   NAMA_PENERIMA  AS "AlasanTerlambat"         -> Nama Perusahaan
--   COUNT(NOPOLIS) AS "ChronologicalOfIncodent" -> Di Upload
--   ... TMP_MESSAGE IS NOT NULL      AS "City"   -> Diproses
--   ... TMP_MESSAGE = 'Sukses Klaim' AS "CityID" -> Berhasil
--   ... TMP_MESSAGE != ... AND IS NOT NULL AS "ClaimID" -> Gagal
--
-- TIGA PENYELARASAN terhadap versi rekonstruksi:
--
--   1. GROUP BY menyertakan TANGGAL TGLPROSES. Satu batch yang diproses pada dua tanggal
--      berbeda muncul sebagai DUA baris — itu perilaku aslinya, dan rekonstruksi yang
--      selalu meringkasnya menjadi satu baris MENYEMBUNYIKAN pemrosesan bertahap.
--      Kueri asli menulisnya `to_date(to_char(a.TGLPROSES,'dd/mm/yyyy'),'dd/mm/yyyy')`.
--
--      KOREKSI 2026-09-20: padanannya SEMULA ditulis `CAST(A.TGLPROSES AS DATE)` dan
--      diberi keterangan "portabel dan berarti sama". **Itu salah di Oracle.** Tipe DATE
--      Oracle MEMBAWA JAM, sehingga CAST dari TIMESTAMP tidak memotong apa pun — satu
--      hari kalender terpecah menjadi satu kelompok per detik yang berbeda. Di
--      PostgreSQL cast yang sama memang memotong ke hari, jadi kedua basis data
--      berperilaku BERBEDA pada kueri yang seharusnya satu.
--
--      Gejalanya: baris grid yang kembar persis — seluruh kolom sama, termasuk tanggal,
--      karena Go memformatnya hanya sampai hari. Terukur 38 baris di tab Asuransi Kredit
--      dan 2 di ANEKA, dan dilaporkan Work Owner sebagai "detail masih salah".
--
--      Penggantinya EXTRACT tahun/bulan/hari — ANSI, berarti sama di kedua basis data,
--      dan tidak bergantung pada apakah tipe DATE setempat membawa jam. Wakil tanggal
--      yang ditampilkan diambil `MIN(A.TGLPROSES)`: seluruh anggota kelompok berada pada
--      hari yang sama, jadi nilai mana pun memberi tampilan yang sama.
--   2. ORDER BY BATCH DESC — batch terbaru di atas. Rekonstruksi mengurutkan menaik.
--      DITAMBAH tiga kolom pemecah seri (2026-09-20), dan itu WAJIB di sini meski tidak
--      ada di kueri asli: BATCH berulang antar perusahaan dan antar tanggal, sehingga
--      `ORDER BY BATCH DESC` saja tidak menentukan satu susunan tunggal. `OFFSET … FETCH`
--      memotong menurut urutan, jadi susunan yang boleh berbeda antar eksekusi membuat
--      satu baris muncul di dua halaman sementara baris lain tidak pernah muncul.
--      Pega tidak menghadapinya karena gridnya memuat seluruh page list sekaligus.
--      Keempat kolomnya bersama-sama adalah kunci GROUP BY, sehingga urutannya total.
--   3. Hitungannya memakai subkueri berkorelasi seperti aslinya, bukan SUM(CASE WHEN).
--      Hasilnya sama; bentuknya dipertahankan supaya perbandingan baris per baris pada
--      gerbang 1 tidak perlu menerjemahkan dua bentuk yang berbeda.
--
-- SATU PERBEDAAN YANG SENGAJA DIPERTAHANKAN: join-nya LEFT, aslinya INNER
-- (`FROM a, b WHERE a.INISIALID = b.INISIALID`). Dengan INNER, batch yang kode
-- perusahaannya tidak ada di master HILANG dari layar tanpa satu pun tanda — padahal
-- baris seperti itu justru yang tidak akan pernah berhasil diproses, karena pencarian
-- penerima klaim membaca master yang sama. Ini perbaikan terencana, bukan cacat, dan
-- wajib disebut saat uji kesetaraan.
--
-- # NAMA PERUSAHAAN DIAGREGASI — master berbaris ganda untuk kode yang sama
--
-- `POOLDATA.M_AUTO_CLAIM_PNC` memuat LEBIH DARI SATU baris untuk satu `INISIALID`.
-- Selama NAMA_PENERIMA ikut di GROUP BY, join ini MENGGANDAKAN baris grid: satu batch
-- tampil dua atau tiga kali, seluruh kolomnya sama persis sehingga tidak ada apa pun di
-- layar yang menjelaskannya.
--
-- Akibatnya terukur pada basis data hari ini: **38 baris kembar di tab Asuransi Kredit
-- dan 2 di ANEKA** (`-periksa`, 2026-09-20). Ia juga merusak paginasi — kueri penghitung
-- TIDAK menjoin master, jadi "Total Data" menyebut angka yang lebih kecil daripada baris
-- yang benar-benar tampil, dan halaman terakhir menjadi pendek.
--
-- `MAX(...)` memilih satu nama secara deterministik dan mengeluarkannya dari GROUP BY,
-- sehingga jumlah baris grid kembali sama dengan jumlah kelompok
-- `(perusahaan, batch, pengunggah, tanggal)` — persis yang dihitung kueri penghitung.
--
-- Kueri Pega tidak menghadapinya karena ia memakai INNER JOIN gaya lama
-- (`FROM a, b WHERE …`) yang menggandakan dengan cara yang sama — perbedaannya, gridnya
-- memuat seluruh page list sehingga barisnya kembar di layar tanpa merusak paginasi.
-- Pilihan `MAX` karena itu **memperbaiki**, bukan menyalin.
--
-- # :1 DAN :2 SENGAJA MEMBAWA NILAI YANG SAMA
--
-- Keduanya diisi 'Sukses Klaim'. Menulisnya `:1` dua kali terasa lebih rapi dan
-- MEMBUAT KUERI INI GAGAL: driver mengikat nilai menurut URUTAN KEMUNCULAN, bukan
-- menurut nomornya, sehingga bind terakhir tidak pernah terisi dan Oracle menolak
-- seluruh pernyataan dengan `ORA-01008: not all variables bound`.
--
-- Cacat itu benar-benar terjadi dan baru terlihat saat layar dibuka terhadap Oracle
-- sungguhan — tidak satu pun uji menangkapnya, karena penyimpanan memori tidak
-- menguraikan SQL. Penjaganya sekarang TestQueryBindsAreNumberedInOrder.
SELECT A.{{KOLOM}},
       MAX(B.NAMA_PENERIMA) AS NAMA_PENERIMA,
       A.BATCH,
       MIN(A.TGLPROSES) AS TANGGAL_PROSES,
       (SELECT COUNT(C.NOPOLIS)
          FROM {{TABEL}} C
         WHERE C.BATCH = A.BATCH
           AND C.{{KOLOM}} = A.{{KOLOM}})                AS JUMLAH_UPLOAD,
       (SELECT COUNT(C.NOPOLIS)
          FROM {{TABEL}} C
         WHERE C.BATCH = A.BATCH
           AND C.{{KOLOM}} = A.{{KOLOM}}
           AND C.TMP_MESSAGE IS NOT NULL)                AS JUMLAH_PROSES,
       (SELECT COUNT(C.NOPOLIS)
          FROM {{TABEL}} C
         WHERE C.BATCH = A.BATCH
           AND C.{{KOLOM}} = A.{{KOLOM}}
           AND C.TMP_MESSAGE = :1)                       AS JUMLAH_BERHASIL,
       (SELECT COUNT(C.NOPOLIS)
          FROM {{TABEL}} C
         WHERE C.BATCH = A.BATCH
           AND C.{{KOLOM}} = A.{{KOLOM}}
           AND C.TMP_MESSAGE <> :2
           AND C.TMP_MESSAGE IS NOT NULL)                AS JUMLAH_GAGAL,
       A.USERINPUT
  FROM {{TABEL}} A
  LEFT JOIN POOLDATA.M_AUTO_CLAIM_PNC B
    ON B.INISIALID = A.{{KOLOM}}
 GROUP BY A.{{KOLOM}}, A.BATCH, A.USERINPUT,
          EXTRACT(YEAR FROM A.TGLPROSES), EXTRACT(MONTH FROM A.TGLPROSES), EXTRACT(DAY FROM A.TGLPROSES)
 ORDER BY A.BATCH DESC, A.{{KOLOM}}, MIN(A.TGLPROSES) DESC, A.USERINPUT
OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY

-- name: auto_claim_batch_list_by_company
--
-- Sama, disaring pada satu perusahaan.
--
-- Kueri TERPISAH, bukan satu kueri berpenyaring opsional. Aslinya menyisipkan penyaring
-- lewat `{ASIS:TemporaryInboxKasirAutoClaim.CaseID}` yang dirangkai dari NAMA perusahaan
-- di Activity/INBOX_AS_KREDIT_ACT_AUTOCLAIM-Act.xml — celah injeksi yang tidak dibawa.
--
-- Penyaringnya pada KODE, bukan nama: nama perusahaan tidak dijamin unik, dan
-- INISIALID adalah kolom pertama kedua index yang ada.
SELECT A.{{KOLOM}},
       MAX(B.NAMA_PENERIMA) AS NAMA_PENERIMA,
       A.BATCH,
       MIN(A.TGLPROSES) AS TANGGAL_PROSES,
       (SELECT COUNT(C.NOPOLIS)
          FROM {{TABEL}} C
         WHERE C.BATCH = A.BATCH
           AND C.{{KOLOM}} = A.{{KOLOM}})                AS JUMLAH_UPLOAD,
       (SELECT COUNT(C.NOPOLIS)
          FROM {{TABEL}} C
         WHERE C.BATCH = A.BATCH
           AND C.{{KOLOM}} = A.{{KOLOM}}
           AND C.TMP_MESSAGE IS NOT NULL)                AS JUMLAH_PROSES,
       (SELECT COUNT(C.NOPOLIS)
          FROM {{TABEL}} C
         WHERE C.BATCH = A.BATCH
           AND C.{{KOLOM}} = A.{{KOLOM}}
           AND C.TMP_MESSAGE = :1)                       AS JUMLAH_BERHASIL,
       (SELECT COUNT(C.NOPOLIS)
          FROM {{TABEL}} C
         WHERE C.BATCH = A.BATCH
           AND C.{{KOLOM}} = A.{{KOLOM}}
           AND C.TMP_MESSAGE <> :2
           AND C.TMP_MESSAGE IS NOT NULL)                AS JUMLAH_GAGAL,
       A.USERINPUT
  FROM {{TABEL}} A
  LEFT JOIN POOLDATA.M_AUTO_CLAIM_PNC B
    ON B.INISIALID = A.{{KOLOM}}
 WHERE A.{{KOLOM}} = :3
 GROUP BY A.{{KOLOM}}, A.BATCH, A.USERINPUT,
          EXTRACT(YEAR FROM A.TGLPROSES), EXTRACT(MONTH FROM A.TGLPROSES), EXTRACT(DAY FROM A.TGLPROSES)
 ORDER BY A.BATCH DESC, A.{{KOLOM}}, MIN(A.TGLPROSES) DESC, A.USERINPUT
OFFSET :4 ROWS FETCH NEXT :5 ROWS ONLY

-- name: auto_claim_batch_count
--
-- Jumlah baris daftar batch, untuk "Total Data :" dan tombol Last.
--
-- TIDAK ada padanannya di Pega: BrowseClaimSPK_COUNT_AutoClaim dan BrowseAutoClaim_COUNT
-- ternyata kueri yang IDENTIK, dan keduanya menghitung baris SATU batch — itu jumlah
-- untuk layar DETAIL, bukan untuk daftarnya.
--
-- Kueri ini karena itu TAMBAHAN. Pengelompokannya dibuat sama persis dengan kueri
-- daftarnya, termasuk tanggal TGLPROSES, supaya keduanya tidak dapat berselisih.
SELECT COUNT(*)
  FROM (SELECT A.{{KOLOM}}, A.BATCH, A.USERINPUT
          FROM {{TABEL}} A
         GROUP BY A.{{KOLOM}}, A.BATCH, A.USERINPUT,
                  EXTRACT(YEAR FROM A.TGLPROSES), EXTRACT(MONTH FROM A.TGLPROSES), EXTRACT(DAY FROM A.TGLPROSES)) RINGKAS

-- name: auto_claim_batch_count_by_company
SELECT COUNT(*)
  FROM (SELECT A.{{KOLOM}}, A.BATCH, A.USERINPUT
          FROM {{TABEL}} A
         WHERE A.{{KOLOM}} = :1
         GROUP BY A.{{KOLOM}}, A.BATCH, A.USERINPUT,
                  EXTRACT(YEAR FROM A.TGLPROSES), EXTRACT(MONTH FROM A.TGLPROSES), EXTRACT(DAY FROM A.TGLPROSES)) RINGKAS

-- name: auto_claim_company_list
--
-- Pilihan penyaring Nama Perusahaan.
--
-- Asal: InboxAutoClaim/BrowseCompanyClaimCredit-SQL.xml
--
--   select D.NAMA_PENERIMA as "City" from POOLDATA.M_AUTO_CLAIM_PNC d
--
-- DISELARASKAN: sumbernya MASTER, bukan tabel batch seperti pada rekonstruksi. Perusahaan
-- yang belum pernah mengirim apa pun karena itu ikut tampil — memilihnya menghasilkan
-- daftar kosong, dan itu memang perilaku aslinya.
--
-- Kodenya ikut diambil walau kueri asli hanya mengambil nama: penyaringnya bekerja pada
-- kode (lihat auto_claim_batch_list_by_company), dan nama tidak dijamin unik.
SELECT A.INISIALID,
       A.NAMA_PENERIMA
  FROM POOLDATA.M_AUTO_CLAIM_PNC A
 ORDER BY A.NAMA_PENERIMA, A.INISIALID

-- name: auto_claim_line_list
--
-- Rincian satu batch — layar DETAIL, 15 baris per halaman.
--
-- Asal: InboxAutoClaim/BrowseClaimSPK_detail_AutoClaim-SQL.xml
--
--   nopolis                    AS "City"
--   TO_CHAR(TGLPROSES,'DD/MM/YYYY') AS "CityID"
--   IDPEGA                     AS "ClaimNo"
--   noakseptasi                AS "CaseID"
--   nilaiklaim                 AS "District"
--   col_id                     AS "DistrictID"
--   TMP_MESSAGE                AS "Country"
--   (select CURRENCY from POOLDATA.CURRENCY where id = a.CURRENCY) AS "CountryID"
--
-- DUA PENYELARASAN:
--
--   1. CURRENCY DI-LOOKUP. Kolomnya menyimpan ID, bukan kode — rekonstruksi
--      menampilkannya apa adanya, sehingga layar memperlihatkan angka, bukan "IDR".
--   2. Kolom yang ditampilkan adalah TGLPROSES, bukan TGLKEJADIAN + TGLLAPOR.
--      Keduanya tetap diambil di sini karena dipakai kunci baris dan pengurutan, tetapi
--      yang ditampilkan layar mengikuti aslinya.
--
-- TO_CHAR tidak dipakai: pemformatan tanggal dilakukan di Go (D-20). TGLPROSES karena itu
-- dibaca sebagai TIMESTAMP.
--
-- Pengurutannya memakai kunci primer (NOPOLIS, TGLPROSES, TGLKEJADIAN), sehingga OFFSET
-- tidak dapat melewatkan atau menggandakan baris. Kueri asli tidak punya ORDER BY sama
-- sekali — urutannya karena itu tidak ditentukan, dan berpindah halaman di Pega dapat
-- menampilkan baris yang sama dua kali. Itu cacat yang TIDAK direplikasi.
SELECT A.{{KOLOM}},
       A.BATCH,
       A.NOPOLIS,
       A.PRODKE,
       A.IDPEGA,
       A.NOAKSEPTASI,
       A.CURRENCY,
       (SELECT M.CURRENCY FROM POOLDATA.CURRENCY M WHERE M.ID = A.CURRENCY) AS KODE_MATA_UANG,
       A.NILAIKLAIM,
       A.COL_ID,
       A.TGLKEJADIAN,
       A.TGLLAPOR,
       A.TGLPROSES,
       A.NOTE,
       A.KEYWORD,
       A.OBJECTNAME,
       A.FLAGTIDAKBAYAR,
       A.TMP_MESSAGE,
       A.USERINPUT
  FROM {{TABEL}} A
 WHERE A.{{KOLOM}} = :1
   AND A.BATCH = :2
 ORDER BY A.NOPOLIS, A.TGLPROSES, A.TGLKEJADIAN
OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY

-- name: auto_claim_line_list_succeeded
SELECT A.{{KOLOM}},
       A.BATCH,
       A.NOPOLIS,
       A.PRODKE,
       A.IDPEGA,
       A.NOAKSEPTASI,
       A.CURRENCY,
       (SELECT M.CURRENCY FROM POOLDATA.CURRENCY M WHERE M.ID = A.CURRENCY) AS KODE_MATA_UANG,
       A.NILAIKLAIM,
       A.COL_ID,
       A.TGLKEJADIAN,
       A.TGLLAPOR,
       A.TGLPROSES,
       A.NOTE,
       A.KEYWORD,
       A.OBJECTNAME,
       A.FLAGTIDAKBAYAR,
       A.TMP_MESSAGE,
       A.USERINPUT
  FROM {{TABEL}} A
 WHERE A.{{KOLOM}} = :1
   AND A.BATCH = :2
   AND A.TMP_MESSAGE = :3
 ORDER BY A.NOPOLIS, A.TGLPROSES, A.TGLKEJADIAN
OFFSET :4 ROWS FETCH NEXT :5 ROWS ONLY

-- name: auto_claim_line_list_failed
--
-- Perhatikan IS NOT NULL: baris yang BELUM diproses bukan baris gagal. Kueri ekspor
-- sistem lama menegaskannya —
-- "AND TMP_MESSAGE!='Sukses Klaim' and TMP_MESSAGE is not null".
SELECT A.{{KOLOM}},
       A.BATCH,
       A.NOPOLIS,
       A.PRODKE,
       A.IDPEGA,
       A.NOAKSEPTASI,
       A.CURRENCY,
       (SELECT M.CURRENCY FROM POOLDATA.CURRENCY M WHERE M.ID = A.CURRENCY) AS KODE_MATA_UANG,
       A.NILAIKLAIM,
       A.COL_ID,
       A.TGLKEJADIAN,
       A.TGLLAPOR,
       A.TGLPROSES,
       A.NOTE,
       A.KEYWORD,
       A.OBJECTNAME,
       A.FLAGTIDAKBAYAR,
       A.TMP_MESSAGE,
       A.USERINPUT
  FROM {{TABEL}} A
 WHERE A.{{KOLOM}} = :1
   AND A.BATCH = :2
   AND A.TMP_MESSAGE IS NOT NULL
   AND A.TMP_MESSAGE <> :3
 ORDER BY A.NOPOLIS, A.TGLPROSES, A.TGLKEJADIAN
OFFSET :4 ROWS FETCH NEXT :5 ROWS ONLY

-- name: auto_claim_line_count
--
-- Asal: InboxAutoClaim/BrowseClaimSPK_COUNT_AutoClaim-SQL.xml, yang isinya sama persis
-- dengan BrowseAutoClaim_COUNT — dua rule Pega untuk satu kueri yang identik.
SELECT COUNT(1)
  FROM {{TABEL}} A
 WHERE A.{{KOLOM}} = :1
   AND A.BATCH = :2

-- name: auto_claim_line_count_succeeded
SELECT COUNT(1)
  FROM {{TABEL}} A
 WHERE A.{{KOLOM}} = :1
   AND A.BATCH = :2
   AND A.TMP_MESSAGE = :3

-- name: auto_claim_line_count_failed
SELECT COUNT(1)
  FROM {{TABEL}} A
 WHERE A.{{KOLOM}} = :1
   AND A.BATCH = :2
   AND A.TMP_MESSAGE IS NOT NULL
   AND A.TMP_MESSAGE <> :3

-- name: auto_claim_export
--
-- Isi kedua berkas CSV.
--
-- Asal: InboxAutoClaim/BrowseReportClaimSPK_AutoClaim-SQL.xml
--
-- TIGA PENYELARASAN yang seluruhnya memperbaiki tebakan rekonstruksi:
--
--   1. "No Objek"    = COL_ID, bukan PRODKE.
--   2. "Currency"    = lookup POOLDATA.CURRENCY, bukan nilai kolom apa adanya.
--   3. "No Klaim"    = IDPEGA yang DIPANGKAS bila memuat 'PNC':
--                      `case when idpega like '%PNC%' then substr(IDPEGA,20,30) else idpega end`
--
-- Butir 3 baru masuk akal setelah membaca Activity/InsertKlaimToTable_Other-Act.xml:
-- saat sebuah baris GAGAL, Pega mengisi IDPEGA, NOAKSEPTASI, DAN TMP_MESSAGE ketiganya
-- dengan TEKS GALAT. Jadi IDPEGA tidak selalu berisi nomor klaim — dan CASE WHEN itulah
-- yang membedakan nomor klaim sungguhan (ber-'PNC', berprefix kelas Pega yang dibuang)
-- dari pesan galat yang dibiarkan utuh.
--
-- SATU KOLOM YANG BELUM DAPAT DIISI: "No Ref Bank" aslinya subkueri ke
-- `gl.t_claim_asuransi_credit@asmd.sinarmas.co.id` lewat DB LINK. D-25 mengganti seluruh
-- DB Link dengan API, dan API-nya belum ada (R-03) — kolomnya karena itu dikirim KOSONG,
-- bukan diisi tebakan. Lihat docs/keputusan-implementasi.md §18.
SELECT A.{{KOLOM}},
       A.NOPOLIS,
       A.IDPEGA,
       A.NOAKSEPTASI,
       A.NILAIKLAIM,
       A.COL_ID,
       (SELECT M.CURRENCY FROM POOLDATA.CURRENCY M WHERE M.ID = A.CURRENCY) AS KODE_MATA_UANG,
       A.TMP_MESSAGE
  FROM {{TABEL}} A
 WHERE A.{{KOLOM}} = :1
   AND A.BATCH = :2
 ORDER BY A.NOPOLIS, A.TGLPROSES, A.TGLKEJADIAN

-- name: auto_claim_export_succeeded
SELECT A.{{KOLOM}},
       A.NOPOLIS,
       A.IDPEGA,
       A.NOAKSEPTASI,
       A.NILAIKLAIM,
       A.COL_ID,
       (SELECT M.CURRENCY FROM POOLDATA.CURRENCY M WHERE M.ID = A.CURRENCY) AS KODE_MATA_UANG,
       A.TMP_MESSAGE
  FROM {{TABEL}} A
 WHERE A.{{KOLOM}} = :1
   AND A.BATCH = :2
   AND A.TMP_MESSAGE = :3
 ORDER BY A.NOPOLIS, A.TGLPROSES, A.TGLKEJADIAN

-- name: auto_claim_export_failed
SELECT A.{{KOLOM}},
       A.NOPOLIS,
       A.IDPEGA,
       A.NOAKSEPTASI,
       A.NILAIKLAIM,
       A.COL_ID,
       (SELECT M.CURRENCY FROM POOLDATA.CURRENCY M WHERE M.ID = A.CURRENCY) AS KODE_MATA_UANG,
       A.TMP_MESSAGE
  FROM {{TABEL}} A
 WHERE A.{{KOLOM}} = :1
   AND A.BATCH = :2
   AND A.TMP_MESSAGE IS NOT NULL
   AND A.TMP_MESSAGE <> :3
 ORDER BY A.NOPOLIS, A.TGLPROSES, A.TGLKEJADIAN

-- name: auto_claim_batch_exists
--
-- Memastikan pasangan (perusahaan, batch) ada tanpa membaca isinya.
--
-- Dipakai sebelum menerbitkan berkas unduhan: berkas kosong karena batch-nya salah ketik
-- tidak dapat dibedakan dari berkas kosong karena batch-nya memang belum diproses.
SELECT 1
  FROM {{TABEL}} A
 WHERE A.{{KOLOM}} = :1
   AND A.BATCH = :2
 FETCH NEXT 1 ROWS ONLY

-- name: auto_claim_receiver
--
-- Menurunkan KODE PERUSAHAAN dari nomor polis — langkah pertama rantai unggah.
--
-- Asal: RDB List/GetReceiverClaimAsuransiKredit-SQL.xml, dipanggil
-- Activity/InsertKlaimToTable_Other-Act.xml pada langkah
-- "cari penerima klaim berdasarkan SourceOfBusiness".
--
-- Kode perusahaan TIDAK datang dari berkas unggahan: ia diturunkan dari
-- `t_general.sourceofbusiness` milik polis, lalu dicocokkan ke master. Kedua penyaring
-- `claim_allowed = 1` dan `APPROVAL = '1'` dipertahankan — perusahaan yang belum disetujui
-- menghasilkan "Sumber Bisnis Tidak ditemukan", persis seperti aslinya.
--
-- Subkueri prodke-nya memakai MAX, mengikuti aslinya: yang dipakai adalah baris polis
-- termutakhir.
SELECT A.INISIALID,
       A.NAMA_PENERIMA
  FROM POOLDATA.M_AUTO_CLAIM_PNC A
 WHERE A.INISIALID = (SELECT B.SOURCEOFBUSINESS
                        FROM T_GENERAL B
                       WHERE B.NOPOLIS = :1
                         AND B.PRODKE = (SELECT MAX(C.PRODKE)
                                           FROM T_GENERAL C
                                          WHERE C.NOPOLIS = B.NOPOLIS)
                       FETCH NEXT 1 ROWS ONLY)
   AND A.CLAIM_ALLOWED = 1
   AND A.APPROVAL = '1'

-- name: auto_claim_policy
--
-- Mencari polis dan PRODKE termutakhirnya — langkah kedua rantai unggah.
--
-- Asal: RDB List/BrowsePolisAso-SQL.xml, dipanggil pada langkah "get prodke".
--
-- DATA_JSONBLOB milik kueri asli TIDAK diambil: isinya snapshot polis, dan membacanya
-- menuntut model data polis yang menjadi lingkup B-1. Akibatnya CURRENCY tidak dapat
-- diisi saat unggah — lihat catatan pada auto_claim_line_insert.
SELECT A.PRODKE
  FROM JSON_POLIS A, T_GENERAL B
 WHERE A.NOPOLIS = :1
   AND A.NOPOLIS = B.NOPOLIS
   AND A.PRODKE = B.PRODKE
 ORDER BY CAST(B.PRODKE AS INT) DESC
 FETCH NEXT 1 ROWS ONLY

-- name: auto_claim_batch_number_used
--
-- Nomor batch yang sudah dipakai, untuk menurunkan nomor berikutnya.
--
-- Pega memakai `GetMaxBatchAutoClaim` lalu `local.batch = max + 1`
-- (Activity/InsertKlaimToTable_Other-Act.xml). Rule itu BELUM ADA di export, sehingga
-- lingkupnya — global atau per perusahaan — belum dapat dipastikan. Yang dipakai di sini
-- lingkup PER PERUSAHAAN, mengikuti kunci pengelompokan (INISIALID, BATCH) yang dipakai
-- GroupingAutoClaim2.
--
-- Nomornya dihitung di Go, bukan dengan MAX di sini, supaya penurunannya berada di satu
-- tempat bersama pemeriksaan tabrakan (lihat inboxautoclaim.NextBatchNumber).
SELECT A.BATCH
  FROM {{TABEL}} A
 WHERE A.{{KOLOM}} = :1
 GROUP BY A.BATCH

-- name: auto_claim_line_insert
--
-- Menyisipkan satu baris hasil unggahan.
--
-- Pega menyisipkannya lewat POOLDATA.INSERT_AUTOCLAIM (RDB List/InsertUpdateAutoClaim-SQL.xml).
-- D-02 melarang memanggil stored procedure, jadi penyisipannya ditulis langsung dengan
-- pemetaan parameter yang sama:
--
--   tBATCH   <- BATCH        tINISIAL <- INISIALID     tPOLIS <- NOPOLIS
--   tPRODKE  <- PRODKE       tUSER    <- USERINPUT     tIDPEGA <- IDPEGA
--   tDOL     <- TGLKEJADIAN  tTGLLAPOR<- TGLLAPOR      tCURRENCY <- CURRENCY
--   tCOL     <- COL_ID       tNILAI   <- NILAIKLAIM    tNOTE  <- NOTE
--   tKEYWORD <- KEYWORD      tMSG     <- TMP_MESSAGE   tAKSEP <- NOAKSEPTASI
--
-- TGLPROSES WAJIB DIISI — ia bagian PRIMARY KEY (NOPOLIS, TGLPROSES, TGLKEJADIAN).
-- Rekonstruksi tidak mengisinya, dan penyisipannya akan ditolak ORA-01400. Nilainya
-- CURRENT_TIMESTAMP, bukan SYSDATE (D-20).
--
-- TIGA KOLOM YANG DIISI TEKS GALAT saat baris tidak lolos validasi: IDPEGA, NOAKSEPTASI,
-- dan TMP_MESSAGE ketiganya menerima pesan yang sama. Itu perilaku
-- Activity/InsertKlaimToTable_Other-Act.xml, bukan karangan — dan itulah sebabnya kueri
-- ekspor perlu memangkas IDPEGA hanya bila ia memuat 'PNC'.
--
-- Baris yang LOLOS validasi meninggalkan ketiganya NULL, dan itu yang membuatnya terambil
-- pemrosesan: GroupingAutoClaim2 mencari `idpega is null AND (progress is null OR
-- progress='0')`, GroupingAutoClaim mencari `noakseptasi IS NULL`.
--
-- CURRENCY dikirim NULL sampai B-1 tersedia: Pega mengambilnya dari snapshot polis
-- (`TempPNC2.Policy.Currency`), dan menebaknya dari berkas unggahan akan mengisi kolom
-- ID dengan nilai yang tidak pernah cocok dengan POOLDATA.CURRENCY.
INSERT INTO {{TABEL}}
       (BATCH, {{KOLOM}}, NOPOLIS, PRODKE, TGLPROSES, USERINPUT,
        IDPEGA, TGLKEJADIAN, TGLLAPOR, CURRENCY, COL_ID, NILAIKLAIM,
        NOTE, KEYWORD, TMP_MESSAGE, NOAKSEPTASI, OBJECTNAME, FLAGTIDAKBAYAR)
VALUES (:1, :2, :3, :4, CURRENT_TIMESTAMP, :5,
        :6, :7, :8, :9, :10, :11,
        :12, :13, :14, :15, :16, :17)

-- name: auto_claim_company_summary
--
-- Menghitung JUMLAH BATCH per perusahaan untuk panel ringkasan.
--
-- # Tidak ada padanannya di export Pega
--
-- Layar Pega memakai komponen ringkasan bawaan platform yang meringkas hasil report
-- definition-nya sendiri, bukan kueri SQL tersendiri. Kueri ini karena itu **kemampuan
-- baru**, bukan pemindahan — dan tidak ada yang dapat dibandingkan dengannya pada
-- gerbang 1.
--
-- Yang menjaganya tetap setara adalah SATU HAL: satuan hitungnya dibuat persis sama
-- dengan satu baris grid, yaitu satu kelompok
-- `(INISIALID, BATCH, USERINPUT, tanggal TGLPROSES)`. Pengelompokan itu disalin apa
-- adanya dari `auto_claim_batch_count`, sehingga:
--
--   jumlah pada baris "All"            = total pada paginasi grid tanpa penyaring
--   jumlah pada satu baris perusahaan  = total pada paginasi grid setelah disaring
--
-- Bila keduanya pernah berbeda, yang salah adalah kueri ini — bukan gridnya.
--
-- # Hanya perusahaan yang punya batch DI TAB INI
--
-- Ini dua kali berubah arah, dan sebab keduanya patut dicatat supaya tidak berputar lagi.
--
-- Versi pertama menghitung dari tabel batch saja. Work Owner melaporkan "hanya muncul 2
-- perusahaan, seharusnya lebih" — maka diganti FULL OUTER JOIN ke master, sehingga
-- seluruh ~19 perusahaan terdaftar ikut tampil dengan jumlah 0.
--
-- Begitu KETIGA tab ada, akibatnya terlihat: ketiga tab menampilkan daftar perusahaan
-- yang SAMA PERSIS, karena masternya memang satu untuk ketiganya. Yang berbeda hanya
-- angkanya, dan itu terbaca sebagai "penyaringnya belum berfungsi" — dilaporkan Work
-- Owner 2026-09-20.
--
-- Laporan pertama pun sebenarnya bukan tentang master: yang dicari Work Owner data
-- Asuransi Kredit, yang saat itu belum punya tab sendiri. Tab ANEKA memang hanya punya
-- 2 perusahaan berbatch.
--
-- Arah yang benar karena itu sama dengan Pega, dan sama dengan kueri contoh yang dikirim
-- Work Owner: hitung dari TABEL BATCH TAB INI, lalu ambil namanya dari master.
--
-- # LEFT JOIN, bukan INNER seperti Pega
--
-- Satu selisih yang disengaja. Kueri Pega memakai inner join
-- (`WHERE C.AGENID = D.INISIALID`), sehingga batch yang kodenya TIDAK ADA di master
-- hilang dari ringkasan — padahal ia tetap tampil di grid. Angka panel dan isi grid jadi
-- tidak cocok, dan justru baris seperti itulah yang tidak akan pernah berhasil diproses.
--
-- Dengan LEFT JOIN ia tetap terhitung, dengan nama kosong, dan layar menandainya.
--
-- # Akibat yang diterima sadar
--
-- Perusahaan terdaftar yang BELUM pernah mengirim batch tidak muncul, sehingga tidak
-- dapat dipilih sebagai penyaring. Memilihnya pun akan menghasilkan grid kosong, jadi
-- yang hilang hanyalah cara menyatakan "rekanan ini belum mengirim apa pun" — dan itu
-- pertanyaan master data, bukan pertanyaan inbox.
--
-- # Urutan
--
-- Menurun berdasarkan jumlah, lalu kode. Irisan terbesar tampil lebih dulu di grafik dan
-- di tabel, mengikuti contoh tampilan yang disetujui Work Owner 2026-09-19.
-- # MAX(NAMA_PENERIMA), bukan M.NAMA_PENERIMA apa adanya
--
-- POOLDATA.M_AUTO_CLAIM_PNC memuat LEBIH DARI SATU baris untuk kode perusahaan yang sama.
-- Tanpa agregasi, join ini menggandakan baris ringkasan: satu perusahaan tampil dua atau
-- tiga kali dengan angka yang sama persis. Lihat catatan pada auto_claim_batch_list.
SELECT R.INISIALID,
       MAX(M.NAMA_PENERIMA) AS NAMA_PENERIMA,
       R.JUMLAH_BATCH
  FROM (SELECT G.{{KOLOM}} AS INISIALID, COUNT(*) AS JUMLAH_BATCH
          FROM (SELECT A.{{KOLOM}},
                       A.BATCH,
                       A.USERINPUT,
                       MIN(A.TGLPROSES) AS TANGGAL_PROSES
                  FROM {{TABEL}} A
                 GROUP BY A.{{KOLOM}}, A.BATCH, A.USERINPUT,
                          EXTRACT(YEAR FROM A.TGLPROSES), EXTRACT(MONTH FROM A.TGLPROSES), EXTRACT(DAY FROM A.TGLPROSES)) G
         GROUP BY G.{{KOLOM}}) R
  LEFT JOIN POOLDATA.M_AUTO_CLAIM_PNC M
    ON M.INISIALID = R.INISIALID
 GROUP BY R.INISIALID, R.JUMLAH_BATCH
 ORDER BY R.JUMLAH_BATCH DESC, R.INISIALID

-- name: auto_claim_check_table
--
-- Memastikan tabelnya ada dan dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
SELECT A.{{KOLOM}}
  FROM {{TABEL}} A
 WHERE 1 = 0

-- name: auto_claim_check_master
SELECT A.INISIALID
  FROM POOLDATA.M_AUTO_CLAIM_PNC A
 WHERE 1 = 0

-- name: auto_claim_check_currency
--
-- POOLDATA.CURRENCY dipakai me-lookup kode mata uang pada layar DETAIL dan pada kedua
-- berkas ekspor. Tanpa hak baca padanya, kolom mata uang kosong di ketiganya — dan
-- kosongnya tidak menghasilkan galat apa pun, jadi ia harus diperiksa di sini.
SELECT A.ID
  FROM POOLDATA.CURRENCY A
 WHERE 1 = 0
