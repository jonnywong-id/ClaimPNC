-- Kueri modul Inbox Banding Harga Salvage (`MENU_ID 72`, pengganti harness
-- `InboxRequestSalvage`).
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- ============================================================================
-- TABEL YANG DIBACA, DAN SIAPA PEMILIKNYA
-- ============================================================================
--
--   POOLDATA.T_CLAIM_CHEKER_SALVAGE  dimiliki Pega — hanya dibaca
--   POOLDATA.DETAIL_PNC_SALVAGE      dimiliki modul Inbox Salvage — hanya dibaca di sini
--   POOLDATA.PNC_SALVAGE             dimiliki modul Inbox Salvage — hanya dibaca di sini
--
-- TIDAK ADA satu pun pernyataan yang menulis di berkas ini, dan memang tidak boleh ada.
-- Dua alasan, keduanya berlaku:
--
--   * `P-1` — selama masa paralel setiap tabel hanya boleh ditulis SATU sistem, dan kedua
--     tabel salvage sudah dimiliki modul Inbox Salvage.
--   * Jalur tulis layar ini belum diketahui sama sekali:
--     `Section/ButtonApproveRejectedRequest` tidak ada di export, sehingga kolom mana yang
--     ditulis tombol Approve/Reject tidak terbaca.
--
-- ============================================================================
-- DARI MANA KEDUA KUERI INI BERASAL
-- ============================================================================
--
--   list_request   <- RDB List/DataReqSalvage_SQL-SQL.xml
--   count_request  <- RDB List/CountRequestSalvage_Sql-SQL.xml      (lihat catatan)
--   list_history   <- RDB List/HistoryReqSalvage_SQL-SQL.xml
--   count_history  <- RDB List/CountHistoryReqSalvage_Sql-SQL.xml
--
-- Pemanggilnya `Activity/SetReqSalvage_Act-Act.xml` (daftar, param `tipe` 1/2) dan
-- `Activity/GCNMCountRequestSalvage_act-Act.xml` (pencacah).
--
-- ============================================================================
-- PEMETAAN KOLOM — judul di layar -> properti grid Pega -> kolom sebenarnya
-- ============================================================================
--
-- Alias Pega TIDAK dibawa (`D-19`). Di layar ini tidak satu pun alias menyatakan isinya, dan
-- tiga di antaranya menyatakan hal yang SALAH — menyalinnya akan menampilkan kolom yang
-- keliru tanpa satu pun galat.
--
-- Grid "Request Banding Harga" — seluruhnya dari T_CLAIM_CHEKER_SALVAGE:
--
--   Judul kolom      Properti Pega        Kolom sebenarnya   Alias di sini
--   ---------------- -------------------- ------------------ ------------------
--   Tanggal Request  .TglTerimaSalvage    TGLREQUEST         REQUEST_DATE
--   No Klaim         .ClaimID             NOKLAIM            CLAIM_NO
--   Detail Object    .ClientName      (!) IDDETAILSALVAGE    DETAIL_OBJECT
--   Nama Barang      .BranchName      (!) NAMABARANG         ITEM_NAME
--   Harga Barang     .AgentID         (!) HARGABARANG        ITEM_PRICE
--   Harga Request    .Email           (!) HARGAREQUEST       REQUEST_PRICE
--   Note Request     .BranchID        (!) ALASANREQUEST      REQUEST_NOTE
--   Aging            .AgingAmount         (dihitung, §portabilitas)  AGING_DAYS
--   Note Checker     .NoteKomite      (!) NOTEAPPROVE        CHECKER_NOTE
--   (tidak digambar) —                    IDSALVAGE          SALVAGE_ID
--   (tidak digambar) —                    NAMAKOMITE         COMMITTEE_NAME
--
-- Grid "History Cheker" — seluruhnya dari PNC_SALVAGE:
--
--   No Klaim         .ClaimID             NOKLAIM            CLAIM_NO
--   Object Name      .Email           (!) JENISSALVAGE       SALVAGE_TYPE
--   Lokasi Salvage   .AgentID         (!) LOKASISALVAGE      SALVAGE_LOCATION
--   PIC              .BranchName      (!) PIC                PIC
--
-- Tanda (!) berarti nama lamanya menyebut hal yang BERBEDA dari isinya. Empat yang paling
-- mudah menggigit: `.Email` berisi HARGA, `.AgentID` berisi harga lain pada satu grid dan
-- LOKASI pada grid lain, `.BranchName` berisi NAMA BARANG pada satu grid dan NAMA PIC pada
-- grid lain, dan kolom berjudul "Object Name" sebenarnya berisi JENIS SALVAGE.
--
-- ============================================================================
-- PENYARING — dari mana setiap potongannya
-- ============================================================================
--
-- Sistem lama merangkai sebagian penyaringnya sebagai POTONGAN SQL lewat `{Asis:...}`. Di
-- sini seluruhnya menjadi parameter binding; larangan merangkai nilai ke dalam teks SQL tidak
-- ikut dikecualikan oleh `P-5`, karena yang direplikasi adalah perilaku bisnis — bukan celah
-- injeksi (`11-SECURITY.md` §5).
--
--   NAMAKOMITE      DataReqSalvage_SQL baris 39 — `AND A.NAMAKOMITE = {TempLaporan.BranchName}`.
--                   Sudah terikat di Pega pula. Inilah yang membuat layar ini "antrean saya".
--
--   Cari No Klaim   `{Asis:TempLaporan.NoteKasir}` pada grid Request dan
--                   `{Asis:TempLaporan.CaseID}` pada grid History. Keduanya disusun
--                   `Activity/SetReqSalvage_Act-Act.xml` langkah 5 dan 11 sebagai
--                   `"and noklaim='" + Param.noklaim + "' "` — COCOK PERSIS, bukan
--                   mengandung, dan nilainya dirangkai ke dalam teks SQL. Di sini ia
--                   terikat, dan perbandingannya diseragamkan huruf besar (lihat NewQuery).
--
--   Giliran komite  `{Asis:TempLaporan.NoteKomite}`, langkah 9. Ia BUKAN penyaring
--                   kepemilikan melainkan urutan giliran: menyembunyikan baris yang komite
--                   sebelumnya belum memutuskannya. Berlaku bagi satu operator saja, dan
--                   aturannya ada di inboxbandinghargasalvage/komite.go.
--                   Hanya grid Request yang memakainya; HistoryReqSalvage_SQL tidak.
--
-- ============================================================================
-- PORTABILITAS — UMUR DIHITUNG DENGAN BENTUK YANG BERJALAN DI KEDUA BASIS DATA
-- ============================================================================
--
-- Kueri lama memakai `TRUNC (SYSDATE) - TRUNC (A.tglrequest)`. Keduanya ada di daftar
-- padanan wajib `09-DATABASE-STRATEGY.md` §4 — `SYSDATE` menjadi `CURRENT_TIMESTAMP`, dan
-- `TRUNC(tanggal)` menjadi `CAST(x AS DATE)` — sehingga bentuknya di sini:
--
--   CAST(CURRENT_TIMESTAMP AS DATE) - CAST(a.TGLREQUEST AS DATE)
--
-- Hasilnya SAMA PERSIS: pemangkasan ke tanggal tetap terjadi pada kedua sisi, dan
-- pengurangan dua tanggal menghasilkan bilangan hari penuh di Oracle maupun PostgreSQL. Yang
-- hilang hanya ketergantungan pada dua fungsi khas Oracle.
--
-- Pemangkasan kedua sisi itu bukan kerapian: tanpanya, selisih dihitung dari JAM, sehingga
-- banding yang masuk kemarin sore terhitung nol hari sampai lewat 24 jam.
--
-- Bentuk yang sama sudah dipakai modul Inbox Claim Treaty Non Prop untuk kolom Aging-nya.
--
-- ============================================================================
-- EMPAT SELISIH YANG DISENGAJA
-- ============================================================================
--
-- Seluruhnya disetujui Work Owner 2026-09-29 dan dinyatakan pula lewat
-- inboxbandinghargasalvage.PlannedDifferences, supaya uji kesetaraan gerbang 1 tidak
-- melaporkannya sebagai bug (`D-54`).
--
--   1. AGING SEBAGAI ANGKA, BUKAN TEKS
--      Kueri lama: `TRUNC (SYSDATE) - TRUNC (A.tglrequest) || ' days' AS "AgingAmount"`
--      lalu `ORDER BY "AgingAmount" DESC`. Karena hasilnya TEKS, urutannya leksikografis:
--      '9 days' didahulukan dari '30 days'. Justru banding yang paling lama menunggu yang
--      tenggelam. Di sini ia angka, dan diurutkan sebagai angka; teks '<n> days' dibentuk
--      lapisan transport.
--
--   2. KOLOM CATATAN KOMITE IKUT DIPILIH — dan namanya BUKAN yang semula disimpulkan
--
--      Grid lama menggambar kolom "Note Checker" (`.NoteKomite`) tetapi tidak pernah
--      memilih kolomnya. Kolom itu semula disimpulkan bernama `NOTEKOMITE`, mengikuti nama
--      propertinya — dan itu KELIRU. `RDB List/UpdateDataReqSalvage-SQL.xml`, yang diterima
--      kemudian, membuktikan kolom yang ditulis tombol Approve/Reject adalah
--      **`NOTEAPPROVE`**:
--
--        UPDATE POOLDATA.T_CLAIM_CHEKER_SALVAGE
--           SET STATUSAPPROVE = …, NOTEAPPROVE = …, TGLAPPROVE = sysdate
--         WHERE NAMAKOMITE = … AND IDDETAILSALVAGE = … AND TGLAPPROVE IS NULL
--
--      `NOTEKOMITE` memang ada, tetapi pada tabel LAIN — `T_CLAIM_KOMITE_LIST`
--      (`RDB List/ShowKomiteTerimaTolakNonMBU-SQL.xml`). Menyalinnya ke sini akan
--      menggagalkan SELURUH kueri dengan ORA-00904, bukan mengosongkan satu kolom.
--
--      ============================================================================
--      DAN KOLOM ITU TETAP KOSONG DI GRID INI — bukan karena kuerinya, melainkan
--      karena isi gridnya
--      ============================================================================
--
--      Grid Request hanya menampilkan baris ber-`TGLAPPROVE IS NULL`, yakni yang BELUM
--      diputus. `UpdateDataReqSalvage` menulis `NOTEAPPROVE` dan `TGLAPPROVE` SEKALIGUS,
--      sehingga baris yang catatannya terisi pasti sudah punya `TGLAPPROVE` — dan karena
--      itu sudah tidak ada lagi di grid ini.
--
--      Jadi memilih kolomnya TIDAK mengubah apa yang dilihat pengguna. Ia tetap dipilih
--      karena judul kolomnya menjanjikannya dan biayanya nol; yang keliru adalah anggapan
--      semula bahwa Pega "lupa" memilihnya. Catatan itu memang tidak pernah ada di sini.
--
--      Catatan komite juga TIDAK tergambar di panel rincian History:
--      `DetailHistReqSalvage_SQL` memilih tujuh kolom, dan `NOTEAPPROVE` bukan salah
--      satunya. Ia ditulis, tetapi tidak pernah dibaca satu layar pun.
--
--   3. PENCACAH DISAMAKAN DENGAN DAFTARNYA
--      `CountRequestSalvage_Sql` menghitung populasi yang BERBEDA dari daftarnya:
--
--        daftar    T_CLAIM_CHEKER_SALVAGE, `HARGAREQUEST IS NOT NULL`, `IN (…)` ke detail
--        pencacah  JOIN ke DETAIL_PNC_SALVAGE, `NILAI_REQUEST IS NOT NULL`
--
--      JOIN itu dapat melipatgandakan baris bila satu IDDETAILSALVAGE punya lebih dari satu
--      baris checker, dan kolom yang disyaratkannya pun ada di TABEL YANG BERBEDA. Di sini
--      count_request memakai klausa WHERE yang sama persis dengan list_request.
--
--   4. `ORDER BY TGL_REQUEST DESC` PADA PENCACAH DIBUANG
--      Ia menutup `SELECT COUNT(1)` di `CountRequestSalvage_Sql`. Tidak bermakna pada kueri
--      agregat, menyebut kolom `TGL_REQUEST` yang NOL KEMUNCULAN di seluruh export selain
--      baris itu, dan pada Oracle berpotensi menggagalkan kuerinya — yang akibatnya
--      `Pagination.TotalData` tidak pernah terisi.
--
-- ============================================================================
-- PAGINASI
-- ============================================================================
--
-- `OFFSET … FETCH NEXT` di basis data. Sistem lama menempuhnya lewat kelas
-- `ASM-FW-GISFW-Data-Pagination` (`SetReqSalvage_Act` langkah 2: `.FirstRow`, `.LastRow`,
-- `.PageSize`), dan teks SQL yang diekspor tidak memuat klausanya sama sekali — ia disisipkan
-- mekanisme paginasi Pega di luar rule-nya.
--
-- Satu hal yang HARUS ditambahkan, bukan disalin: `HistoryReqSalvage_SQL` tidak punya
-- `ORDER BY` sama sekali. Tanpa urutan yang pasti, `OFFSET … FETCH` mengembalikan halaman
-- yang isinya dapat berubah-ubah — satu baris muncul dua kali sementara baris lain terlewat.
-- Urutan ditetapkan di sini, dan pilihannya dinyatakan pada kueri yang bersangkutan.
--
-- ============================================================================
-- PENANDA PARAMETER
-- ============================================================================
--
-- Setiap penanda muncul TEPAT SEKALI dan bernomor urut: penanda berulang membuat jumlah
-- argumen tidak lagi sama dengan jumlah kemunculan, dan galatnya baru terbaca saat kueri
-- dijalankan.

-- name: list_request
-- Satu halaman banding harga yang BELUM diputus, milik satu komite.
--
--   :1  nama komite pemanggil, huruf besar
--   :2  kata kunci, NULL bila tidak mencari — penjaga penyaring pencarian
--   :3  nomor klaim yang dicari, huruf besar
--   :4  nama komite yang gilirannya ditunggu, NULL bila tidak ada — penjaga NOT EXISTS
--   :5  nama komite yang gilirannya ditunggu, huruf besar
--   :6  offset
--   :7  jumlah baris
--
-- `ORDER BY AGING_DAYS DESC` — yang paling lama menunggu di atas, sama maksudnya dengan
-- kueri lama. Bedanya ia mengurutkan ANGKA, bukan teks (lihat selisih 1 di kepala berkas).
--
-- `IDDETAILSALVAGE` sebagai pemecah seri supaya urutan baris tidak berubah antar halaman
-- ketika beberapa banding punya umur yang sama persis.
SELECT a.NOKLAIM                                  AS CLAIM_NO,
       a.TGLREQUEST                               AS REQUEST_DATE,
       a.IDDETAILSALVAGE                          AS DETAIL_OBJECT,
       a.NAMABARANG                               AS ITEM_NAME,
       a.HARGABARANG                              AS ITEM_PRICE,
       a.HARGAREQUEST                             AS REQUEST_PRICE,
       a.ALASANREQUEST                            AS REQUEST_NOTE,
       (CAST(CURRENT_TIMESTAMP AS DATE)
        - CAST(a.TGLREQUEST AS DATE))             AS AGING_DAYS,
       a.NOTEAPPROVE                              AS CHECKER_NOTE,
       a.IDSALVAGE                                AS SALVAGE_ID,
       a.NAMAKOMITE                               AS COMMITTEE_NAME
  FROM POOLDATA.T_CLAIM_CHEKER_SALVAGE a
 WHERE a.IDDETAILSALVAGE IN (SELECT d.IDDETAILSALVAGE
                               FROM POOLDATA.DETAIL_PNC_SALVAGE d)
   AND a.HARGAREQUEST IS NOT NULL
   AND a.TGLAPPROVE IS NULL
   AND UPPER(TRIM(a.NAMAKOMITE)) = :1
   AND (:2 IS NULL OR UPPER(TRIM(a.NOKLAIM)) = :3)
   AND (:4 IS NULL
        OR NOT EXISTS (SELECT 1
                         FROM POOLDATA.T_CLAIM_CHEKER_SALVAGE c
                        WHERE c.IDDETAILSALVAGE = a.IDDETAILSALVAGE
                          AND UPPER(TRIM(c.NAMAKOMITE)) = :5
                          AND c.STATUSAPPROVE IS NULL))
 ORDER BY AGING_DAYS DESC, a.IDDETAILSALVAGE
OFFSET :6 ROWS FETCH NEXT :7 ROWS ONLY

-- name: count_request
-- Jumlah seluruh banding yang cocok, tanpa paginasi.
--
-- Klausa WHERE-nya WAJIB sama persis dengan list_request. Keduanya dijaga query_test.go,
-- karena penyaring yang berbeda menghasilkan "halaman 1 dari 7" yang halaman ketujuhnya
-- kosong — dan di layar ini ia juga menghasilkan angka ringkas yang tidak pernah cocok
-- dengan gridnya, persis cacat yang selisih 3 di kepala berkas perbaiki.
--
--   :1 … :5  sama dengan list_request
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIM_CHEKER_SALVAGE a
 WHERE a.IDDETAILSALVAGE IN (SELECT d.IDDETAILSALVAGE
                               FROM POOLDATA.DETAIL_PNC_SALVAGE d)
   AND a.HARGAREQUEST IS NOT NULL
   AND a.TGLAPPROVE IS NULL
   AND UPPER(TRIM(a.NAMAKOMITE)) = :1
   AND (:2 IS NULL OR UPPER(TRIM(a.NOKLAIM)) = :3)
   AND (:4 IS NULL
        OR NOT EXISTS (SELECT 1
                         FROM POOLDATA.T_CLAIM_CHEKER_SALVAGE c
                        WHERE c.IDDETAILSALVAGE = a.IDDETAILSALVAGE
                          AND UPPER(TRIM(c.NAMAKOMITE)) = :5
                          AND c.STATUSAPPROVE IS NULL))

-- name: list_history
-- Satu halaman pengajuan salvage yang banding harganya SUDAH diputus komite ini.
--
--   :1  nama komite pemanggil, huruf besar
--   :2  kata kunci, NULL bila tidak mencari — penjaga penyaring pencarian
--   :3  nomor klaim yang dicari, huruf besar
--   :4  offset
--   :5  jumlah baris
--
-- # Barisnya PENGAJUAN SALVAGE, bukan klaim
--
-- Kueri lama memilih dari `PNC_SALVAGE` tanpa `DISTINCT`, sehingga satu klaim yang punya
-- beberapa pengajuan salvage muncul beberapa kali. Perilakunya ditiru apa adanya (`P-5`):
-- menambahkan `DISTINCT` akan mengubah jumlah baris yang dilihat pengguna, dan itu selisih
-- yang akan dilaporkan uji kesetaraan gerbang 1.
--
-- # ORDER BY ditambahkan, dan itu bukan pilihan gaya
--
-- Kueri lama tidak punya `ORDER BY` sama sekali. Tanpa urutan yang pasti, `OFFSET … FETCH`
-- boleh mengembalikan baris dalam urutan mana pun — termasuk urutan yang BERBEDA antar
-- halaman, sehingga satu baris muncul dua kali sementara baris lain tidak pernah terlihat.
--
-- `NOKLAIM` naik, lalu `IDSALVAGE` sebagai pemecah seri. Nomor klaim memuat tahun pada
-- segmen keduanya (`PNCN.YY.xxxx`, `D-71`), sehingga urutan ini terbaca sebagai urutan
-- penerbitan bagi klaim sistem baru.
SELECT s.NOKLAIM                                  AS CLAIM_NO,
       s.JENISSALVAGE                             AS SALVAGE_TYPE,
       s.LOKASISALVAGE                            AS SALVAGE_LOCATION,
       s.PIC                                      AS PIC
  FROM POOLDATA.PNC_SALVAGE s
 WHERE s.NOKLAIM IN (SELECT c.NOKLAIM
                       FROM POOLDATA.T_CLAIM_CHEKER_SALVAGE c
                      WHERE UPPER(TRIM(c.NAMAKOMITE)) = :1
                        AND c.STATUSAPPROVE IS NOT NULL
                        AND c.TGLAPPROVE IS NOT NULL)
   AND (:2 IS NULL OR UPPER(TRIM(s.NOKLAIM)) = :3)
 ORDER BY s.NOKLAIM, s.IDSALVAGE
OFFSET :4 ROWS FETCH NEXT :5 ROWS ONLY

-- name: count_history
-- Jumlah seluruh baris History yang cocok, tanpa paginasi.
--
-- Klausa WHERE-nya sama persis dengan list_history, dengan alasan yang sama seperti
-- count_request.
--
--   :1 … :3  sama dengan list_history
SELECT COUNT(*)
  FROM POOLDATA.PNC_SALVAGE s
 WHERE s.NOKLAIM IN (SELECT c.NOKLAIM
                       FROM POOLDATA.T_CLAIM_CHEKER_SALVAGE c
                      WHERE UPPER(TRIM(c.NAMAKOMITE)) = :1
                        AND c.STATUSAPPROVE IS NOT NULL
                        AND c.TGLAPPROVE IS NOT NULL)
   AND (:2 IS NULL OR UPPER(TRIM(s.NOKLAIM)) = :3)

-- name: check_table
-- Memastikan tabel inti modul ini terbaca, BESERTA setiap kolom yang dipakainya.
--
-- Ia dipanggil perintah `-periksa` saat aplikasi start, dan sengaja tidak menyentuh baris
-- mana pun: yang diperiksa adalah HAK BACA dan keberadaan kolomnya, bukan isinya.
--
-- # Kenapa kolomnya disebut satu per satu, bukan `SELECT COUNT(*)` saja
--
-- Karena DDL `T_CLAIM_CHEKER_SALVAGE` belum pernah dibaca — seluruh nama kolom di berkas ini
-- disimpulkan dari teks kueri Pega. Satu di antaranya PERNAH keliru: kolom catatan komite
-- sempat ditulis `NOTEKOMITE`, disimpulkan dari nama properti gridnya, sampai
-- `UpdateDataReqSalvage` membuktikan namanya `NOTEAPPROVE`. `COUNT(*)` akan lolos meski
-- kolomnya tidak ada; menyebutkannya membuat kekeliruan itu terbaca saat start, lengkap
-- dengan nama kolom yang salah.
--
-- `WHERE 1 = 0` menjaga agar tidak satu baris pun dibaca; agregatnya tetap mengembalikan satu
-- baris berisi nol.
SELECT COUNT(a.NOKLAIM)
     + COUNT(a.IDSALVAGE)
     + COUNT(a.IDDETAILSALVAGE)
     + COUNT(a.NAMAKOMITE)
     + COUNT(a.NAMABARANG)
     + COUNT(a.HARGABARANG)
     + COUNT(a.HARGAREQUEST)
     + COUNT(a.ALASANREQUEST)
     + COUNT(a.TGLREQUEST)
     + COUNT(a.TGLAPPROVE)
     + COUNT(a.STATUSAPPROVE)
     + COUNT(a.NOTEAPPROVE)
  FROM POOLDATA.T_CLAIM_CHEKER_SALVAGE a
 WHERE 1 = 0

-- name: check_write_targets
-- Memastikan KETIGA tabel yang ditulis tombol Approve/Reject punya kolom yang ditulisnya.
--
-- Dipanggil perintah `-periksa`. Ia membaca KATALOG, bukan tabelnya — menguji hak tulis
-- dengan benar-benar menulis akan meninggalkan baris percobaan di tabel produksi.
--
-- # Kenapa ini perlu, padahal check_table sudah ada
--
-- Karena check_table hanya menyentuh tabel yang DIBACA. Dua tabel lain baru tersentuh sejak
-- tombolnya dibangun, dan nama kolom keduanya pun disimpulkan dari teks kueri Pega:
--
--   POOLDATA.SALAVAGEDOCUMENT      IDBALAILELANG, NOKLAIM, TIPEDOCSALVAGE
--   POOLDATA.DETAIL_PNC_SALVAGE    HARGAITEM, IDSALVAGE, IDDETAILSALVAGE
--
-- Kolom yang ternyata bernama lain akan menggagalkan seluruh transaksi keputusan — dan itu
-- baru ketahuan pada saat yang paling buruk, yakni ketika seorang komite menekan Simpan atas
-- putusan yang sudah ia pertimbangkan.
--
-- Hasilnya jumlah kolom yang DITEMUKAN. Pemanggil membandingkannya dengan yang diharapkan,
-- sehingga pesannya dapat menyebut berapa yang kurang.
SELECT COUNT(*)
  FROM ALL_TAB_COLUMNS t
 WHERE t.OWNER = 'POOLDATA'
   AND ( (t.TABLE_NAME = 'SALAVAGEDOCUMENT'
          AND t.COLUMN_NAME IN ('IDBALAILELANG', 'NOKLAIM', 'TIPEDOCSALVAGE'))
      OR (t.TABLE_NAME = 'DETAIL_PNC_SALVAGE'
          AND t.COLUMN_NAME IN ('HARGAITEM', 'IDSALVAGE', 'IDDETAILSALVAGE')) )

-- name: list_decisions
-- Keputusan banding harga satu klaim — panel rincian pada grid History Cheker.
--
--   :1  nomor klaim, huruf besar
--   :2  nama komite pemanggil, huruf besar
--
-- Menggantikan `RDB List/DetailHistReqSalvage_SQL-SQL.xml`, dengan DUA perbedaan yang
-- disengaja dan satu yang tidak:
--
--   1. `STATUSAPPROVE` dikirim sebagai KODE, bukan diterjemahkan di dalam `CASE WHEN`.
--      Pemetaannya pindah ke inboxbandinghargasalvage.DecisionLabel. Alasannya sama dengan
--      modul Inbox Salvage: pemetaan yang hidup di dalam SQL tidak dapat diuji tanpa basis
--      data, sementara pemetaan yang salah menampilkan keputusan yang keliru tanpa galat.
--
--   2. Nomor klaim dan nama komite DIIKAT, bukan dirangkai. Kueri lama menyisipkan keduanya
--      lewat `{TempLaporan.…}`; di sini keduanya parameter.
--
-- Yang TIDAK berbeda: penyaring `NAMAKOMITE`. Ia ada di kueri aslinya, dan tanpa itu seorang
-- komite dapat membaca keputusan komite lain hanya dengan menebak nomor klaim.
--
-- `ORDER BY TGLAPPROVE DESC` diikuti `IDDETAILSALVAGE` sebagai pemecah seri. Kueri lama hanya
-- punya yang pertama; tanpa pemecah seri, dua keputusan yang jatuh pada cap waktu yang sama
-- dapat bertukar urutan antar pemuatan — pada panel yang tidak berhalaman itu tidak merusak,
-- tetapi urutan yang berubah-ubah tanpa sebab membingungkan pembacanya.
SELECT a.TGLAPPROVE                               AS APPROVED_AT,
       a.IDDETAILSALVAGE                          AS DETAIL_OBJECT,
       a.NAMABARANG                               AS ITEM_NAME,
       a.HARGABARANG                              AS ITEM_PRICE,
       a.HARGAREQUEST                             AS REQUEST_PRICE,
       a.STATUSAPPROVE                            AS DECISION_STATUS,
       a.NAMAKOMITE                               AS COMMITTEE_NAME
  FROM POOLDATA.T_CLAIM_CHEKER_SALVAGE a
 WHERE UPPER(TRIM(a.NOKLAIM)) = :1
   AND UPPER(TRIM(a.NAMAKOMITE)) = :2
   AND a.STATUSAPPROVE IS NOT NULL
 ORDER BY a.TGLAPPROVE DESC, a.IDDETAILSALVAGE

-- ============================================================================
-- KEPUTUSAN BANDING — keempat pernyataan tulis
-- ============================================================================
--
-- Seluruhnya berjalan dalam SATU transaksi yang dimiliki Go (`D-68`). Di Pega tiap
-- pernyataan berdiri sendiri dan menyimpan seketika, sehingga kegagalan di tengah
-- meninggalkan putusan tanpa harga — atau harga tanpa putusan. Di sini kegagalan
-- mengembalikan keadaan seperti semula.
--
-- Itu SELISIH YANG DISENGAJA, dan ia mengubah keadaan akhir saat terjadi kegagalan — sama
-- halnya dengan `B-4`/`B-9` pada `D-68`. Ia dinyatakan lewat PlannedDifferences.
--
-- Kepemilikan tabel: `T_CLAIM_CHEKER_SALVAGE` dimiliki modul ini; `DETAIL_PNC_SALVAGE` dan
-- `SALAVAGEDOCUMENT` dimiliki modul Inbox Salvage, dan modul ini hanya menyentuh SATU kolom
-- pada masing-masing — keputusan Work Owner 2026-09-30.

-- name: decide_record
-- Mencatat putusan komite atas satu barang.
--
--   :1  STATUSAPPROVE — '1' setuju, '0' tidak setuju
--   :2  NOTEAPPROVE   — catatan komite; boleh kosong
--   :3  nama komite, huruf besar
--   :4  IDDETAILSALVAGE
--
-- `TGLAPPROVE = CURRENT_TIMESTAMP`, bukan `sysdate` — padanan wajib
-- `09-DATABASE-STRATEGY.md` §4. Artinya sama.
--
-- Penyaring `TGLAPPROVE IS NULL` dibawa apa adanya, dan ia yang membuat pernyataan ini
-- AMAN DIULANG: penekanan tombol kedua tidak mengubah apa pun, dan jumlah baris yang
-- terpengaruh nol — yang oleh pemanggil dijawab sebagai "sudah diputus", bukan "berhasil".
UPDATE POOLDATA.T_CLAIM_CHEKER_SALVAGE
   SET STATUSAPPROVE = :1,
       NOTEAPPROVE   = :2,
       TGLAPPROVE    = CURRENT_TIMESTAMP
 WHERE UPPER(TRIM(NAMAKOMITE)) = :3
   AND IDDETAILSALVAGE = :4
   AND TGLAPPROVE IS NULL

-- name: decide_cascade
-- Menutup pula baris komite BERIKUTNYA dengan status yang sama.
--
--   :1  STATUSAPPROVE
--   :2  nama komite yang ikut ditutup, huruf besar
--   :3  IDDETAILSALVAGE
--
-- Hanya berjalan ketika jenjang pertama MENOLAK — tidak ada gunanya menanyakan persetujuan
-- atas harga yang sudah ditolak. Prakondisinya ada di inboxbandinghargasalvage.PlanDecision.
--
-- Di Pega pernyataan ini DIRANGKAI sebagai teks, lengkap dengan nama komitenya:
--
--   "UPDATE POOLDATA.T_CLAIM_CHEKER_SALVAGE SET STATUSAPPROVE = '" + … +
--   "' WHERE NAMAKOMITE='DANIELLISWANDI' AND IDDETAILSALVAGE='" + … + "' …"
--
-- Nama komitenya di sini DIIKAT, bukan ditanam di dalam teks. Nilainya tetap sama —
-- aturannya ditiru (`P-5`) — tetapi ia datang dari satu tempat yang dapat dicabut sekaligus.
--
-- `NOTEAPPROVE` sengaja TIDAK ikut ditulis: pernyataan aslinya pun tidak menulisnya. Baris
-- yang ditutup begini karena itu tidak punya catatan, dan memang tidak pernah punya.
UPDATE POOLDATA.T_CLAIM_CHEKER_SALVAGE
   SET STATUSAPPROVE = :1,
       TGLAPPROVE    = CURRENT_TIMESTAMP
 WHERE UPPER(TRIM(NAMAKOMITE)) = :2
   AND IDDETAILSALVAGE = :3
   AND TGLAPPROVE IS NULL

-- name: decide_mark_document
-- Menandai dokumen banding sebagai ditolak checker.
--
--   :1  IDDETAILSALVAGE
--
-- ============================================================================
-- PENYARINGNYA TAMPAK KELIRU, DAN ITU DITIRU APA ADANYA
-- ============================================================================
--
-- Kueri aslinya menyaring `WHERE NOKLAIM = {TempInsert.ClaimNo}` — dan `TempInsert.ClaimNo`
-- diisi parameter `iddetailsalvage`, bukan nomor klaim
-- (`Activity/ApprovalCheckerSalvage-Act.xml` langkah 2). Membandingkan kolom NOKLAIM dengan
-- sebuah ID detail salvage hampir pasti tidak mencocokkan satu baris pun.
--
-- Keputusan Work Owner 2026-09-30: **ditiru apa adanya**, dicatat sebagai selisih yang
-- diketahui. Yang dibawa adalah perilaku bisnisnya — termasuk ketika perilaku itu berarti
-- "tidak melakukan apa-apa".
--
-- Akibatnya perlu dibaca apa adanya: penandaan dokumen kemungkinan besar TIDAK PERNAH
-- berjalan, di Pega maupun di sini. Bila kelak dipastikan keliru, yang berubah cukup satu
-- nama kolom di baris WHERE ini.
UPDATE POOLDATA.SALAVAGEDOCUMENT
   SET IDBALAILELANG = 'Reject Checker'
 WHERE NOKLAIM = :1
   AND TIPEDOCSALVAGE = 'Request Banding Harga Salvage'
   AND IDBALAILELANG IS NULL

-- name: decide_apply_price
-- Menerapkan harga tandingan balai lelang ke barang salvage-nya.
--
--   :1  HARGAREQUEST yang disetujui
--   :2  IDSALVAGE
--   :3  IDDETAILSALVAGE
--
-- Satu-satunya pernyataan modul ini yang menyentuh `DETAIL_PNC_SALVAGE`, dan ia menyentuh
-- SATU kolom saja. Tabel itu dimiliki modul Inbox Salvage; batas ini keputusan Work Owner
-- 2026-09-30 dan tidak boleh diperluas tanpa keputusan baru.
--
-- Hanya berjalan ketika jenjang TERAKHIR menyetujui — lihat
-- inboxbandinghargasalvage.PlanDecision, termasuk akibatnya bagi komite lain.
UPDATE POOLDATA.DETAIL_PNC_SALVAGE
   SET HARGAITEM = :1
 WHERE IDSALVAGE = :2
   AND IDDETAILSALVAGE = :3

-- name: list_documents
-- Dokumen banding satu barang — isi dialog "Lihat File".
--
--   :1  IDDETAILSALVAGE
--   :2  IDSALVAGE
--   :3  nama komite pemanggil, huruf besar
--
-- Menggantikan dua kueri `Activity/LihatDokRequestSalvage-Act.xml`, dengan empat perbedaan
-- yang disengaja.
--
-- # 1. Satu kueri, bukan N+1
--
-- Activity lama menjalankan satu kueri daftar, lalu SATU KUERI LAGI PER BARIS untuk mengambil
-- nama berkasnya. Di sini keduanya disatukan dengan JOIN. Hasilnya sama; yang berbeda hanya
-- jumlah perjalanan ke basis data.
--
-- # 2. Nilai diikat, bukan dirangkai
--
-- Kueri pertama di sana tidak punya rule sendiri: teksnya dirangkai menjadi properti
-- klipboard (`TempClaimAttach.AlasanKlaim`) lalu dijalankan mentah lewat rule ber-`pyBrowseSQL`
-- `{ASIS:…}`. Apa pun yang ada di kedua parameter itu masuk langsung ke dalam teks SQL.
--
-- # 3. Penyaring KEPEMILIKAN ditambahkan — tidak ada di sistem lama
--
-- Kueri lama hanya menyaring IDDETAILSALVAGE dan IDSALVAGE, keduanya dikirim pemanggil.
-- Tanpa penyaring ketiga, siapa pun yang tahu sepasang id dapat membaca daftar dokumen
-- banding komite lain. Klausa EXISTS di bawah menuntut banding itu memang ditangani komite
-- pemanggil, sama seperti kedua grid layar ini (`11-SECURITY.md` §3.2 — batas data ditegakkan
-- di lapisan kueri, bukan disaring setelah data terambil).
--
-- # 4. Kolom NOKLAIM dibandingkan dengan IDDETAILSALVAGE
--
-- Ini BUKAN salah ketik: kueri lama melakukan hal yang sama. Karena ia kueri BACA yang
-- hasilnya langsung terlihat pengguna, perbandingan itu pastilah mencocokkan sesuatu —
-- sehingga kolom `SALAVAGEDOCUMENT.NOKLAIM` hampir pasti berisi id detail salvage meski
-- namanya menyatakan nomor klaim. Lihat PlannedDifferences.
--
-- `IDBALAILELANG IS NULL` ikut ditiru: dokumen yang sudah ditandai ditolak tidak lagi muncul.
SELECT d.IDDOC                           AS DOCUMENT_ID,
       a.ATTACHNAME                      AS DOCUMENT_NAME,
       d.TGLINS                          AS UPLOADED_AT
  FROM POOLDATA.SALAVAGEDOCUMENT d
  JOIN POOLDATA.DATA_ATTACHFILE a
    ON a.DATAID = d.IDDOC
 WHERE d.NOKLAIM = :1
   AND d.IDSALVAGE = :2
   AND d.TIPEDOCSALVAGE = 'Request Banding Harga Salvage'
   AND d.IDBALAILELANG IS NULL
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_CLAIM_CHEKER_SALVAGE c
                WHERE c.IDDETAILSALVAGE = :1
                  AND c.IDSALVAGE = :2
                  AND UPPER(TRIM(c.NAMAKOMITE)) = :3)
 ORDER BY d.TGLINS DESC, d.IDDOC DESC

-- name: document_content
-- Isi satu dokumen banding.
--
--   :1  IDDOC / DATAID
--   :2  IDDETAILSALVAGE
--   :3  IDSALVAGE
--   :4  nama komite pemanggil, huruf besar
--
-- Menggantikan kueri kedua `LihatDokRequestSalvage` — rule `GetAttachmentReqSalvage`, yang
-- TIDAK ada di export. Isinya tidak ditebak: kelasnya `ASM-FW-GCNMFW-Int-DATA_ATTACHFILE`,
-- kuncinya `pyID` yang diisi `IDDOC`, dan ketiga kolom yang dibaca hasilnya terbaca dari
-- langkah Property-Set sesudahnya. Kueri yang sama sudah berjalan di modul `inboxpladla`.
--
-- # Kenapa penyaringnya jauh lebih ketat daripada sekadar DATAID
--
-- Karena yang diserahkan adalah ISI BERKAS. Dengan `WHERE DATAID = :1` saja, siapa pun yang
-- masuk dapat mengunduh lampiran mana pun di seluruh basis data hanya dengan menebak
-- angkanya — termasuk lampiran klaim yang tidak ada hubungannya dengan salvage. Klausa
-- EXISTS menuntut dokumen itu memang dokumen banding, dari banding yang ditangani komite
-- pemanggil.
--
-- Berkas yang telanjur terunduh tidak dapat ditarik kembali; penyaring inilah satu-satunya
-- yang mencegahnya (`11-SECURITY.md` §3.2).
SELECT a.ATTACHNAME                      AS DOCUMENT_NAME,
       a.ATTACHMIMETYPE                  AS MIME_TYPE,
       a.ATTACHFILE                      AS CONTENT
  FROM POOLDATA.DATA_ATTACHFILE a
 WHERE a.DATAID = :1
   AND EXISTS (SELECT 1
                 FROM POOLDATA.SALAVAGEDOCUMENT d
                WHERE d.IDDOC = a.DATAID
                  AND d.NOKLAIM = :2
                  AND d.IDSALVAGE = :3
                  AND d.TIPEDOCSALVAGE = 'Request Banding Harga Salvage'
                  AND EXISTS (SELECT 1
                                FROM POOLDATA.T_CLAIM_CHEKER_SALVAGE c
                               WHERE c.IDDETAILSALVAGE = d.NOKLAIM
                                 AND c.IDSALVAGE = d.IDSALVAGE
                                 AND UPPER(TRIM(c.NAMAKOMITE)) = :4))
