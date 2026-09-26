-- Kueri modul Master Reas.
--
-- SATU tabel, dan aplikasi ini TIDAK MENULISNYA: POOLDATA.T_REINSURER.
--
-- Berkas ini hanya memuat SELECT. Itu bukan kelalaian melainkan temuan: satu-satunya penulis
-- tabel ini di sistem lama adalah alur PLA/DLA lewat Database/UPDATEREAS.prc, yang dipanggil
-- Activity/UpdateDetailPLA2 dan Activity/UpdateDetailDLA2 — bukan layar master. Lihat banner
-- paket masterreas.
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
-- TABELNYA PUNYA TUJUH KOLOM; ENAM DIBACA, SATU TIDAK
-- ============================================================================
--
-- Ketujuhnya terbaca dari INSERT pada Database/UPDATEREAS.prc:
--
--   INSERT INTO POOLDATA.T_REINSURER
--     (REINSURERID, REINSURERNAME, LOGIN, EMAIl, COUNTRY, COUNTRYID, TYPE)
--
-- COUNTRYID DITULIS di sana — diterjemahkan dari COUNTRY lewat
-- `SELECT ID FROM COUNTRY WHERE COUNTRY = tCOUNTRY` — lalu TIDAK DIBACA satu pun rule di
-- seluruh export. Ia karena itu tidak ikut di-SELECT: mengambil kolom yang tidak ada
-- pembacanya hanya menambah lalu lintas, dan menampilkannya berarti mengarang kegunaan.
--
-- Keenam yang dibaca terbukti dari empat rule yang saling menguatkan:
--
--   REINSURERID    BrowseEmailReas · GetListDataLoginReas · GetDataPreDLA · INSERT_PLADLA
--   REINSURERNAME  BrowseEmailReas · GetListDataLoginReas · INSERT_PLADLA
--   LOGIN          keempatnya, dan lima kueri inbox menyaring klaim dengannya
--   EMAIL          keempatnya
--   COUNTRY        BrowseEmailReas · GetDataPreDLA · BrowseAllDataXOL_PLA · INSERT_PLADLA
--   TYPE           BrowseEmailReas (penyaring) · GetDataPreDLA (substr(NODLA,0,1) = TYPE)
--
--
-- ============================================================================
-- KENAPA ALIAS KOLOM PEGA TIDAK DIBAWA
-- ============================================================================
--
-- Rule browse-nya mengalias setiap kolom menjadi properti klipboard yang namanya tidak ada
-- hubungannya dengan isinya:
--
--   select email          as "City",
--          reinsurername  as "District",
--          login          as "DistrictID",
--          reinsurerid    as "CityID",
--          country        as "Country"
--     from pooldata.t_reinsurer
--
-- Alamat surel menjadi "City", dan nama perusahaan reasuransi menjadi "District". Itu persis
-- bentuk utang yang 03-CURRENT-ARCHITECTURE.md §4.2 catat: nama dipaksa cocok dengan
-- property klipboard yang sudah ada. Alias itu TIDAK dibawa; kolomnya disebut nama aslinya.
--
--
-- ============================================================================
-- PENYARING DIRANGKAI DARI TEKS DI SISTEM LAMA; DI SINI TIDAK
-- ============================================================================
--
-- Activity/SetDataPLADLA dan Activity/XOLByPerCauseOfLossForKomite merangkai penyaringnya:
--
--   "... where login='" + Local.loginreas + "'"
--
-- Tanpa satu pun pelolosan. Nilai yang memuat tanda kutip tunggal akan menutup literalnya
-- dan sisanya dieksekusi sebagai SQL — celah injeksi yang 03-CURRENT-ARCHITECTURE.md §4.5
-- catat. Di berkas ini setiap nilai adalah parameter, dan bentuk tanpa penyaring ditulis
-- sebagai kueri TERSENDIRI alih-alih satu kueri yang klausanya ditempel.
--
--
-- ============================================================================
-- PEMBANDINGAN MEMAKAI TRIM, DAN PENCARIAN MEMAKAI UPPER
-- ============================================================================
--
-- TRIM: bila kolomnya CHAR dan bukan VARCHAR2, Oracle memadatkan pembandingnya dengan spasi
-- sehingga perbandingan langsung tetap benar — tetapi PostgreSQL tidak melakukannya, dan
-- baris yang sama akan hilang setelah pindah basis data (D-24). DDL-nya tidak ada (R-08),
-- jadi keduanya harus tetap benar.
--
-- UPPER hanya pada PENCARIAN, supaya orang yang mengetik huruf kecil tetap menemukan nama
-- perusahaan yang tersimpan dengan huruf besar. Ia tidak dipakai di tempat lain di berkas
-- ini karena tidak ada pengambilan satu baris menurut kunci.


-- name: reas_list
--
-- Seluruh baris entitas ini.
--
-- # CAKUPANNYA ADALAH REKONSTRUKSI
--
-- Section grid layar lama (`BrowseListMemberReas`) TIDAK ADA di antara 2.634 berkas export
-- (R-16), sehingga tidak dapat dipastikan apakah gridnya menampilkan seluruh baris atau
-- disaring lebih dulu.
--
-- Kedua kueri yang benar-benar ada atas tabel ini berpenyaring, dan penyaringnya selalu satu
-- reasuransi tertentu — keduanya milik alur PLA/DLA, bukan milik layar master:
--
--   BrowseEmailReas       where reinsurerid = {TempReasPLA.ReinsCode} and (type = ... )
--   GetListDataLoginReas  where reinsurername = {TempReasPLA.PLAReinsurer}
--
-- Layar berjudul "Data Member" bertombol Refresh tidak punya keduanya untuk diisi. Yang
-- dipakai di sini adalah bentuk kueri itu TANPA klausa WHERE-nya.
--
-- # URUTANNYA ADALAH PENAMBAHAN, DAN ITU PERLU
--
-- Tidak satu pun kueri lama atas tabel ini memakai ORDER BY, sehingga urutan apa pun adalah
-- penambahan. Tanpa urutan, basis data bebas mengembalikan baris dalam urutan yang berbeda
-- setiap kali — dan pada daftar yang dipaginasi di layar, itu membuat baris yang sama muncul
-- di dua halaman sekaligus sementara yang lain tidak muncul sama sekali.
--
-- REINSURERNAME lebih dulu karena itulah yang dicari orang. TYPE menyusul supaya beberapa
-- baris milik satu perusahaan berurutan dan terbaca sebagai satu kelompok. REINSURERID
-- terakhir sebagai pemutus, supaya urutannya tetap sama ketika dua perusahaan bernama sama.
SELECT REINSURERID,
       REINSURERNAME,
       LOGIN,
       EMAIL,
       COUNTRY,
       TYPE
  FROM POOLDATA.T_REINSURER
 ORDER BY REINSURERNAME, TYPE, REINSURERID

-- name: reas_list_search
--
-- Sama dengan reas_list, ditambah penyaring kata kunci.
--
-- Kueri TERSENDIRI, bukan satu kueri yang klausanya ditempel saat kata kuncinya ada — lihat
-- banner berkas ini.
--
-- EMPAT kolom dicari sekaligus: kode reas, nama reas, login, dan email. Keempatnya yang
-- dihafal orang. COUNTRY tidak ikut — ia penggolongan, bukan cara sebuah perusahaan
-- disebut — dan TYPE juga tidak, karena nilainya satu karakter sehingga akan mencocokkan
-- hampir setiap baris.
--
-- Kata kuncinya sudah dibungkus tanda persen oleh pemanggil, bukan di sini: menempelkannya
-- di dalam teks SQL berarti merangkai nilai ke dalam pernyataan.
--
-- EMPAT parameter berbeda untuk nilai yang sama, bukan satu yang dipakai ulang. Oracle
-- mengizinkan pemakaian ulang, tetapi tidak semua driver memetakan parameter bernomor ke
-- posisi argumen dengan cara yang sama — dan D-20 menuntut kueri ini berjalan sama di kedua
-- basis data. Mengirim nilai yang sama empat kali tidak berbiaya apa-apa; menebak perilaku
-- driver berbiaya sebuah daftar yang diam-diam kosong.
--
-- ESCAPE '\' disebut eksplisit karena Oracle tidak punya karakter pelolos bawaan pada LIKE.
-- Tanpa itu, pengguna yang mengetik "%" akan mencocokkan seluruh baris tanpa satu pun tanda
-- bahwa yang dicari bukan yang diketik.
SELECT REINSURERID,
       REINSURERNAME,
       LOGIN,
       EMAIL,
       COUNTRY,
       TYPE
  FROM POOLDATA.T_REINSURER
 WHERE UPPER(TRIM(REINSURERID)) LIKE :1 ESCAPE '\'
    OR UPPER(TRIM(REINSURERNAME)) LIKE :2 ESCAPE '\'
    OR UPPER(TRIM(LOGIN)) LIKE :3 ESCAPE '\'
    OR UPPER(TRIM(EMAIL)) LIKE :4 ESCAPE '\'
 ORDER BY REINSURERNAME, TYPE, REINSURERID

-- name: reas_check_table
--
-- Membuktikan tabel beserta keenam kolom yang dibaca modul ini ada dan dapat dibaca.
--
-- Dipakai `claimpnc -periksa`. FETCH FIRST 0 ROWS ONLY: yang diperiksa adalah apakah
-- pernyataannya dapat diurai dan dijalankan, bukan isinya — menarik satu baris berarti
-- membaca data nasabah tanpa keperluan.
SELECT REINSURERID,
       REINSURERNAME,
       LOGIN,
       EMAIL,
       COUNTRY,
       TYPE
  FROM POOLDATA.T_REINSURER
 FETCH FIRST 0 ROWS ONLY

-- name: reas_count_all
--
-- Cacah seluruh baris.
SELECT COUNT(*)
  FROM POOLDATA.T_REINSURER

-- name: reas_count_duplicate_key
--
-- Cacah kunci alami yang dipakai lebih dari satu baris.
--
-- Kunci alaminya TIGA kolom — REINSURERID + REINSURERNAME + TYPE — dibaca dari
-- Database/UPDATEREAS.prc, yang memeriksa keberadaan baris dengan ketiganya sekaligus.
--
-- Kenapa ini layak diperiksa: tidak ada DDL-nya (R-08), sehingga tidak diketahui apakah ada
-- constraint unik. Bila ada baris kembar, `UPDATEREAS` akan memperbarui SELURUHNYA sekaligus
-- — UPDATE-nya tidak membatasi jumlah baris — sementara pembacaan PLA/DLA memakai
-- `fetch next 1 row only` dan hanya melihat salah satunya.
SELECT COUNT(*)
  FROM (SELECT REINSURERID, REINSURERNAME, TYPE
          FROM POOLDATA.T_REINSURER
         GROUP BY REINSURERID, REINSURERNAME, TYPE
        HAVING COUNT(*) > 1) DUPLIKAT

-- name: reas_count_shared_login
--
-- Cacah LOGIN yang dipakai lebih dari satu REINSURERID.
--
-- # Ini pemeriksaan yang paling berharga di berkas ini
--
-- LOGIN menentukan klaim mana yang dilihat seorang mitra reasuransi — lima kueri inbox
-- menyaringnya dengan `where login = {OperatorID.pyUserIdentifier}`. Satu login yang
-- menunjuk beberapa kode reasuransi berarti seseorang berpotensi melihat klaim milik mitra
-- lain.
--
-- Sistem lama TAHU keadaan ini mungkin terjadi, dan menyelesaikannya dengan memilih
-- sembarang satu: RDB List/GetPNCList_PLA1-SQL.xml membaca
--
--   (select reinsurerid from pooldata.t_reinsurer
--     where login = {OperatorID.pyUserIdentifier}
--     order by reinsurerid desc fetch next 1 row only)
--
-- `order by ... desc fetch next 1 row only` adalah pengakuan bahwa hasilnya dapat lebih dari
-- satu. Yang dicacah di sini adalah berapa kali keadaan itu benar-benar terjadi di data.
--
-- Baris ber-LOGIN kosong DIKECUALIKAN: ia tidak menunjuk siapa pun, dan mengelompokkannya
-- akan melaporkan satu temuan palsu yang besar. Ia dicacah tersendiri; lihat
-- reas_count_empty_login.
SELECT COUNT(*)
  FROM (SELECT UPPER(TRIM(LOGIN)) AS LOGIN_KEY
          FROM POOLDATA.T_REINSURER
         WHERE LOGIN IS NOT NULL
           AND TRIM(LOGIN) <> ''
         GROUP BY UPPER(TRIM(LOGIN))
        HAVING COUNT(DISTINCT TRIM(REINSURERID)) > 1) BERBAGI

-- name: reas_count_empty_login
--
-- Cacah baris yang LOGIN-nya kosong atau NULL.
--
-- Baris seperti ini tidak dapat dipakai masuk oleh mitra mana pun, sehingga mitra itu tidak
-- akan pernah melihat klaimnya sendiri — dan tidak ada apa pun di layar lama yang
-- menunjukkannya.
SELECT COUNT(*)
  FROM POOLDATA.T_REINSURER
 WHERE LOGIN IS NULL
    OR TRIM(LOGIN) = ''

-- name: reas_count_missing_email
--
-- Cacah baris yang EMAIL-nya kosong atau NULL.
--
-- Surel pada baris ini adalah tujuan pemberitahuan PLA, Pre-DLA, dan DLA. Baris tanpa surel
-- gagal diam-diam: dokumennya terbit, tercatat terkirim, dan tidak pernah sampai ke siapa
-- pun.
SELECT COUNT(*)
  FROM POOLDATA.T_REINSURER
 WHERE EMAIL IS NULL
    OR TRIM(EMAIL) = ''

-- name: reas_count_without_fallback
--
-- Cacah perusahaan reasuransi yang TIDAK punya satu pun baris ber-TYPE '1'.
--
-- Baris ber-TYPE '1' adalah baris CADANGAN: BrowseEmailReas memakainya ketika tidak ada
-- baris yang TYPE-nya cocok dengan jenis dokumen yang sedang dikirim —
--
--   where reinsurerid = {TempReasPLA.ReinsCode} and (type = {TempReasPLA.NoPLA} or type = '1')
--
-- Perusahaan tanpa baris cadangan karena itu hanya terlayani untuk jenis dokumen yang
-- kebetulan sudah punya barisnya sendiri. Untuk jenis lain, kuerinya tidak mengembalikan apa
-- pun — dan surel tujuannya kosong.
--
-- Keadaan ini DAPAT tercipta oleh sistem lama sendiri: UPDATEREAS mengubah baris '1' menjadi
-- tipe yang diminta alih-alih menyisipkan baris baru, sehingga baris cadangannya habis
-- terpakai.
SELECT COUNT(*)
  FROM (SELECT TRIM(REINSURERID) AS REINS_KEY
          FROM POOLDATA.T_REINSURER
         GROUP BY TRIM(REINSURERID)
        HAVING SUM(CASE WHEN TRIM(TYPE) = '1' THEN 1 ELSE 0 END) = 0) TANPA_CADANGAN
