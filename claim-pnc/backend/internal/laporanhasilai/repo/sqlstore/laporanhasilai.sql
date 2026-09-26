-- Kueri Laporan Hasil AI (MENU_ID 82).
--
--   POOLDATA.T_CLAIM_DATA_RESULTS_AI   penilaian AI — satu baris per objek pertanggungan
--   POOLDATA.T_CLAIM_KOMITE_LIST       keputusan komite — satu baris per jenjang
--
-- Keduanya HANYA DIBACA. Layar ini tidak punya satu pun jalur tulis: kedua tombolnya —
-- "Cari Data" dan "Export To Excel" — memanggil activity yang sama,
-- `SearchDataLaporanAI(flagss=2)`, dan activity itu tidak memuat satu pun langkah tulis.
--
-- Asalnya: `RDB List/CountAIDiterima_SQL-SQL.xml`, ruleset GCNMFW 01-03-10,
-- `pyMemo = "work in progress"`.
--
-- Lima aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (`D-20`).
--   4. Tanpa pemanggilan stored procedure (`D-02`).
--   5. TANPA satu pun INSERT, UPDATE, maupun DELETE — modul ini baca-saja.
--
--
-- ============================================================================
-- EMPAT HAL YANG SENGAJA TIDAK DISALIN DARI KUERI LAMA
-- ============================================================================
--
-- ## 1. `{Asis:TempDatalaporanAI.NoteAITerima}` — klausa WHERE yang dirangkai sebagai teks
--
-- Kueri lama menerima seluruh syarat tanggalnya sebagai teks yang disusun activity dari
-- DUA ISIAN YANG DIKETIK PENGGUNA (`Activity/SearchDataLaporanAI-Act.xml`):
--
--     TempDatalaporanAI.NoteAITerima :=
--       " AND trunc(TANGGALKOMITE) >= to_date('" + Local.awal + "','dd/mm/yyyy')
--         and trunc(TANGGALKOMITE) <= to_date('" + Local.akhir + "','dd/mm/yyyy')"
--
-- lalu menempelkannya apa adanya. Itu pola `{ASIS:…}` yang `08-TECHNICAL-STRATEGY.md` §4.3
-- larang mutlak, dan celah SQL injection yang utang teknis 4.5 sebut. Larangan itu tidak
-- dikecualikan oleh keputusan mana pun: yang direplikasi adalah perilaku bisnis, bukan
-- celah injeksi.
--
-- Penggantinya dua parameter terikat. Karena penyaringnya kini WAJIB ada (lihat doc
-- `Filter.Validate`), tidak dibutuhkan varian kueri "tanpa penyaring".
--
--
-- ## 2. `TRUNC` pada kolom tanggal — diganti rentang setengah terbuka
--
-- `trunc(TANGGALKOMITE) >= awal AND trunc(TANGGALKOMITE) <= akhir` menjadi
-- `TANGGALKOMITE >= awal AND TANGGALKOMITE < akhir + 1 hari`.
--
-- Ia MEMILIH BARIS YANG SAMA PERSIS, tetap dapat memakai index, dan portabel ke
-- PostgreSQL sementara `TRUNC(date)` tidak (`09-DATABASE-STRATEGY.md` §4). Satu harinya
-- ditambahkan pemanggil, lewat `Filter.ToExclusive`. Pola yang sama sudah dipakai
-- `inboxrclpucl` dan `inboxoutstanding`.
--
--
-- ## 3. `TO_CHAR(SYSDATE-7,'dd/mm/yyyy') AS "TanggalKomite"` — kolom mati
--
-- Kueri lama memilih kolom bernama "TanggalKomite" yang isinya BUKAN tanggal komite
-- melainkan tanggal hari ini dikurangi tujuh hari — nilai yang sama untuk setiap baris,
-- dan berubah setiap hari tanpa ada yang mengubah datanya.
--
-- Ia tidak pernah tergambar: kolom layar "Tanggal Komite" membaca `.TanggalComitee`, yang
-- berasal dari `B.TANGGALKOMITE`. Tidak dibawa. Ia pula memakai `SYSDATE` dan `TO_CHAR`
-- yang keduanya dilarang.
--
--
-- ## 4. `B.KOMITEKE IN (SELECT MAX(B.KOMITEKE) FROM … C WHERE …)` — ditulis sebagai EXISTS
--
-- Bentuk aslinya menyesatkan. `B` di dalam subkueri itu BUKAN alias baru melainkan baris
-- terluar, sehingga `MAX(B.KOMITEKE)` mengagregasi sebuah nilai tetap:
--
--     AND B.KOMITEKE IN (
--           SELECT MAX(B.KOMITEKE) FROM POOLDATA.T_CLAIM_KOMITE_LIST C
--            WHERE C.STATUSCASE = 'Resolved-Completed'
--              AND B.KOMITE_ID = C.KOMITE_ID
--              AND B.TANGGALKOMITE IS NOT NULL)
--
-- Hasilnya `B.KOMITEKE = B.KOMITEKE` bila ada C yang cocok, dan `IN (NULL)` — yang selalu
-- salah — bila tidak ada. Dua syarat yang tersisa karena itu:
--
--     ada baris komite lain ber-KOMITE_ID sama yang STATUSCASE-nya Resolved-Completed
--     DAN baris ini sendiri ber-TANGGALKOMITE tidak NULL
--
-- Ia TIDAK menyaring jenjang terakhir, meski bentuknya menyerupai itu. Ditulis di sini
-- sebagai `EXISTS` + `IS NOT NULL` — himpunan baris yang sama persis, dengan maksud yang
-- akhirnya terbaca.
--
--
-- ============================================================================
-- YANG SENGAJA TIDAK DIPILIH — kolom yang membuat lima kolom layar kosong
-- ============================================================================
--
-- `OBJECTNAME`, `NOTETERIMA`, `NOTETOLAK`, `COVERAGE_AI_FINAL`, dan `KATEGORI_KRONOLOGI`
-- TIDAK dipilih di sini, sehingga kelima kolom layar yang membacanya tergambar kosong —
-- persis seperti Pega hari ini.
--
-- Itu keputusan Work Owner 2026-09-26: replikasi apa adanya. Lihat doc paket
-- `laporanhasilai` bagian "KUERINYA TERTINGGAL DARI LAYARNYA".
--
-- Kelimanya ADA di `POOLDATA.T_CLAIM_DATA_RESULTS_AI`; `RDB List/GetKomitePAditerima-SQL.xml`
-- membaca kelimanya dari tabel yang sama. Membalik keputusan ini berarti menambahkannya
-- pada report_list dan report_check_table, lalu mengisinya di `scanRow` — tanpa artefak
-- baru dari siapa pun.
--
-- Dua kolom lain juga tidak dipilih, dengan alasan berbeda: `B.STATUSCASE` dan
-- `B.NAMAKOMITE` dipilih kueri lama tetapi tidak dibaca properti mana pun di layar.
--
--
-- ============================================================================
-- PENANDA PARAMETER
-- ============================================================================
--
--     :1   batas bawah TANGGALKOMITE, INKLUSIF
--     :2   batas atas TANGGALKOMITE, EKSKLUSIF (tanggal akhir + 1 hari)
--     :3   offset halaman   — hanya pada report_list
--     :4   ukuran halaman   — hanya pada report_list


-- name: report_list
--
-- Satu halaman grid rincian.
--
-- # Urutannya diperluas, dan itu TUNTUTAN paginasi
--
-- Kueri lama mengurutkan `z."CaseIDKomite", z."Komiteno"` — yakni KOMITE_ID lalu KOMITEKE.
-- Keduanya TIDAK unik: satu jenjang komite dapat memuat beberapa penilaian AI, satu per
-- objek pertanggungan.
--
-- Di Pega itu tidak berakibat apa pun; grid-nya memuat seluruh hasil sekaligus. Di sini
-- barisnya dipaginasi lewat OFFSET, dan urutan yang tidak pasti membuat dua halaman
-- berturut-turut dapat memuat baris yang sama sementara baris lain tidak pernah muncul.
--
-- OBJECTID dan COVERAGEID karena itu ditambahkan sebagai pemutus seri. Keduanya pula yang
-- menyusun kunci baris di layar — lihat doc `Row.ID`.
SELECT B.KOMITE_ID,
       B.KOMITEKE,
       B.NO_KLAIM,
       B.STATUSAPPROVE,
       B.TANGGALKOMITE,
       A.OBJECTID,
       A.COVERAGEID,
       A.RESULTAI,
       A.TGLAI
  FROM POOLDATA.T_CLAIM_DATA_RESULTS_AI A
  JOIN POOLDATA.T_CLAIM_KOMITE_LIST B
    ON A.CLAIMID = B.NO_KLAIM
   AND A.KOMITE = B.KOMITE_ID
 WHERE B.TANGGALKOMITE IS NOT NULL
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_CLAIM_KOMITE_LIST C
                WHERE C.KOMITE_ID = B.KOMITE_ID
                  AND C.STATUSCASE = 'Resolved-Completed')
   AND B.TANGGALKOMITE >= :1
   AND B.TANGGALKOMITE < :2
 ORDER BY B.KOMITE_ID, B.KOMITEKE, A.OBJECTID, A.COVERAGEID
OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY

-- name: report_count
--
-- Cacah SELURUH baris yang cocok — penggerak paginator.
--
-- Penyaringnya WAJIB sama persis dengan report_list dan report_summary. Cacah yang
-- dihitung atas penyaring yang berbeda akan membuat paginator melaporkan halaman yang
-- tidak ada. Dijaga query_test.go.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIM_DATA_RESULTS_AI A
  JOIN POOLDATA.T_CLAIM_KOMITE_LIST B
    ON A.CLAIMID = B.NO_KLAIM
   AND A.KOMITE = B.KOMITE_ID
 WHERE B.TANGGALKOMITE IS NOT NULL
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_CLAIM_KOMITE_LIST C
                WHERE C.KOMITE_ID = B.KOMITE_ID
                  AND C.STATUSCASE = 'Resolved-Completed')
   AND B.TANGGALKOMITE >= :1
   AND B.TANGGALKOMITE < :2

-- name: report_summary
--
-- Isi grid ringkasan, dihitung atas SELURUH baris yang cocok — bukan atas halaman yang
-- sedang terbuka.
--
-- # Pembandingnya PERSIS, tanpa TRIM, dan itu disengaja
--
-- Precondition langkah pencacah pada activity lama membandingkan persis:
-- `.ResultAI=="DITERIMA"` dan `.KOMITESTATUS=="1"`. Menambahkan `TRIM` di sini akan
-- MEMPERLUAS hasilnya melampaui yang dihitung Pega, dan selisih semacam itu wajib
-- disetujui Work Owner tertulis (`D-54`) — bukan diambil diam-diam demi kerapian.
--
-- Nilai pembanding STATUSAPPROVE ditulis sebagai TEKS. Tipe kolomnya belum diketahui
-- (`R-08`): bila ia NUMBER, kedua basis data mengonversi literalnya; bila ia VARCHAR2 atau
-- CHAR, perbandingannya langsung. Menulisnya sebagai ANGKA justru berbahaya — pada kolom
-- teks, Oracle akan mengonversi KOLOMNYA dan gagal pada baris yang tidak numerik.
--
-- # Kolom kelima: cacah seluruh baris
--
-- Dari situlah "Menunggu" diturunkan — baris yang bukan diterima dan bukan ditolak.
-- Menghitungnya sebagai sisa, bukan sebagai kondisi tersendiri, membuatnya mustahil
-- meleset: berapa pun nilai tak terduga yang ada di kolomnya, ia tetap terhitung.
SELECT SUM(CASE WHEN A.RESULTAI = 'DITERIMA' THEN 1 ELSE 0 END)     AS AI_ACCEPTED,
       SUM(CASE WHEN A.RESULTAI = 'DITOLAK' THEN 1 ELSE 0 END)      AS AI_REJECTED,
       SUM(CASE WHEN B.STATUSAPPROVE = '1' THEN 1 ELSE 0 END)       AS COMMITTEE_ACCEPTED,
       SUM(CASE WHEN B.STATUSAPPROVE = '2' THEN 1 ELSE 0 END)       AS COMMITTEE_REJECTED,
       COUNT(*)                                                     AS ROW_COUNT
  FROM POOLDATA.T_CLAIM_DATA_RESULTS_AI A
  JOIN POOLDATA.T_CLAIM_KOMITE_LIST B
    ON A.CLAIMID = B.NO_KLAIM
   AND A.KOMITE = B.KOMITE_ID
 WHERE B.TANGGALKOMITE IS NOT NULL
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_CLAIM_KOMITE_LIST C
                WHERE C.KOMITE_ID = B.KOMITE_ID
                  AND C.STATUSCASE = 'Resolved-Completed')
   AND B.TANGGALKOMITE >= :1
   AND B.TANGGALKOMITE < :2

-- name: report_check_table
--
-- Memastikan kedua tabel beserta seluruh kolom yang dibaca ada dan dapat diakses akun
-- aplikasi, tanpa mengambil satu baris pun.
--
-- Urutan kolomnya WAJIB sama dengan report_list; itu yang membuat satu fungsi pembaca
-- baris cukup untuk keduanya. Dijaga query_test.go.
SELECT B.KOMITE_ID,
       B.KOMITEKE,
       B.NO_KLAIM,
       B.STATUSAPPROVE,
       B.TANGGALKOMITE,
       A.OBJECTID,
       A.COVERAGEID,
       A.RESULTAI,
       A.TGLAI
  FROM POOLDATA.T_CLAIM_DATA_RESULTS_AI A
  JOIN POOLDATA.T_CLAIM_KOMITE_LIST B
    ON A.CLAIMID = B.NO_KLAIM
   AND A.KOMITE = B.KOMITE_ID
 WHERE 1 = 0
