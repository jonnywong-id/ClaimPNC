-- Kueri Master Pasal AI.
--
--   POOLDATA.MST_PASAL_AI   WP_ID (kunci) · WP_PASAL · WP_AYAT · WP_KEJADIAN
--
-- Tabel ini HANYA DIBACA modul ini. Tidak ada satu pun jalur tulis di layar lamanya, dan
-- tidak ada rule di export yang menulisinya — siapa yang mengisinya belum diketahui, dan itu
-- dicatat sebagai pertanyaan terbuka, bukan ditebak.
--
-- Lima aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (`D-20`).
--   4. Tanpa pemanggilan stored procedure (`D-02`).
--   5. TANPA satu pun DELETE maupun UPDATE — modul ini baca-saja.
--
-- # DUA HAL YANG SENGAJA TIDAK DISALIN DARI KUERI LAMA
--
-- ## 1. `{ASIS:TempQuery.AlasanKlaim}` — klausa WHERE yang dirangkai sebagai teks
--
-- Kueri lama menerima seluruh klausa `WHERE` sebagai teks yang disusun activity:
--
--   TempQuery.AlasanKlaim := "WHERE (WP_PASAL LIKE '%"+TempSearch.Country+"%' OR ...)"
--
-- lalu menempelkannya apa adanya. Itu pola `{ASIS:…}` yang `08-TECHNICAL-STRATEGY.md` §4.3
-- larang mutlak, dan celah SQL injection yang utang teknis 4.5 sebut: kata kunci yang memuat
-- satu tanda kutip tunggal sudah cukup mengubah bentuk kuerinya.
--
-- Penggantinya DUA kueri terpisah — satu dengan penyaring, satu tanpa — dengan kata kuncinya
-- sebagai parameter terikat. Bukan satu kueri yang klausanya ditempel, karena menempel
-- klausa adalah persis hal yang sedang dihindari.
--
-- ## 2. `ROWNUM` — jendela halaman khas Oracle
--
-- Kueri lama memaginasi dengan tiga tingkat subquery dan `ROWNUM`:
--
--   SELECT B.* FROM (SELECT A.*, ROWNUM RN FROM (SELECT … ORDER BY WP_ID) A) B
--    WHERE RN >= {Pagination.FirstRow} AND RN <= {Pagination.LastRow}
--
-- `ROWNUM` tidak ada di PostgreSQL (`D-20`), dan `FirstRow`/`LastRow` pun ditempel sebagai
-- teks. Penggantinya `OFFSET … FETCH NEXT … ROWS ONLY`, yang berlaku di Oracle 12c+ maupun
-- PostgreSQL, dengan keduanya sebagai parameter terikat.
--
-- Hasilnya SAMA: `FirstRow = offset + 1` dan `LastRow = offset + PageSize`.

-- name: clause_count
--
-- Cacah seluruh baris. Padanan `CountDataPasalAI` tanpa penyaring.
--
-- Kueri lama mengaliaskan hasilnya `AS "BranchID"` — nama properti klipboard warisan yang
-- tidak ada hubungannya dengan cabang mana pun. Alias itu tidak dibawa; yang dibaca di sini
-- adalah kolom pertama apa adanya.
SELECT COUNT(*)
  FROM POOLDATA.MST_PASAL_AI

-- name: clause_count_search
--
-- Cacah baris yang cocok dengan kata kunci.
--
-- Ketiga kolom dicocokkan dan digabung `OR`, persis ekspresi aslinya
-- (`Activity/GetListPasalAI_act.xml:548`).
--
-- `UPPER` dipasang di KEDUA sisi supaya pencarian tidak bergantung pada besar-kecil huruf
-- yang kebetulan tersimpan — kueri lama membandingkan apa adanya, dan Oracle peka huruf pada
-- `LIKE`. Ini SELISIH TERENCANA yang memperluas hasil, bukan mempersempitnya.
--
-- `ESCAPE '\'` disebut eksplisit karena Oracle TIDAK punya karakter pelolos bawaan pada
-- `LIKE`; PostgreSQL memakai backslash sebagai bawaan, dan menyebutkannya membuat keduanya
-- berperilaku sama (`D-20`).
--
-- Nilai yang sama dikirim tiga kali, bukan sekali: parameter terikat tidak dapat dipakai
-- ulang secara portabel — Oracle menomori `:1`, PostgreSQL `$1`, dan driver keduanya
-- memperlakukan pengulangan berbeda.
SELECT COUNT(*)
  FROM POOLDATA.MST_PASAL_AI
 WHERE UPPER(WP_PASAL) LIKE :1 ESCAPE '\'
    OR UPPER(WP_AYAT) LIKE :2 ESCAPE '\'
    OR UPPER(WP_KEJADIAN) LIKE :3 ESCAPE '\'

-- name: clause_list
--
-- Satu halaman tanpa penyaring. Padanan `GetListDataPasalAI` dengan `AlasanKlaim` kosong.
--
-- `ORDER BY WP_ID` disalin apa adanya dari kueri lama — ia BUKAN tambahan. Tanpa urutan yang
-- pasti, dua halaman berturut-turut dapat memuat baris yang sama sementara baris lain tidak
-- pernah muncul sama sekali.
SELECT WP_ID,
       WP_PASAL,
       WP_AYAT,
       WP_KEJADIAN
  FROM POOLDATA.MST_PASAL_AI
 ORDER BY WP_ID
OFFSET :1 ROWS FETCH NEXT :2 ROWS ONLY

-- name: clause_list_search
--
-- Satu halaman dengan penyaring kata kunci.
--
-- Penyaringnya sama persis dengan clause_count_search; keduanya WAJIB tetap sama, karena
-- cacah yang dihitung atas penyaring yang berbeda akan membuat paginator melaporkan halaman
-- yang tidak ada. Dijaga query_test.go.
SELECT WP_ID,
       WP_PASAL,
       WP_AYAT,
       WP_KEJADIAN
  FROM POOLDATA.MST_PASAL_AI
 WHERE UPPER(WP_PASAL) LIKE :1 ESCAPE '\'
    OR UPPER(WP_AYAT) LIKE :2 ESCAPE '\'
    OR UPPER(WP_KEJADIAN) LIKE :3 ESCAPE '\'
 ORDER BY WP_ID
OFFSET :4 ROWS FETCH NEXT :5 ROWS ONLY

-- name: clause_check_table
--
-- Memastikan tabel beserta keempat kolomnya ada dan dapat dibaca akun aplikasi, tanpa
-- mengambil satu baris pun.
SELECT WP_ID,
       WP_PASAL,
       WP_AYAT,
       WP_KEJADIAN
  FROM POOLDATA.MST_PASAL_AI
 WHERE 1 = 0
