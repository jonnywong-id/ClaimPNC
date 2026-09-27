-- Kueri modul Inbox Komunikasi Cabang (`MENU_ID 70`, pengganti
-- `Harness/InboxKomunikasiCabang`).
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- SELURUH tabel di berkas ini milik sistem lama.
--
-- Sampai 2026-09-24 berkas ini TIDAK memuat satu pun penulisan. Keputusan Work Owner pada
-- tanggal itu mengubahnya: tiga pernyataan yang menulis ditambahkan di bagian paling bawah,
-- dan bersamanya KEPEMILIKAN `POOLDATA.M_KOMUNIKASI_PNC` serta
-- `POOLDATA.M_KOMUNIKASI_CABANG` berpindah ke sistem baru.
--
-- `P-1` tidak dilanggar olehnya — ia justru mekanismenya: tepat satu sistem menulis sebuah
-- tabel, dan kepemilikan berpindah saat modulnya pindah. Yang WAJIB mengikuti adalah sisi
-- Pega: layar `InboxKomunikasiCabang` di sana tidak boleh lagi menulisi kedua tabel itu.
--
-- ============================================================================
-- SATU TABEL, DUA TAB, SATU PENYARING YANG MEMBEDAKAN
-- ============================================================================
--
-- Kedua kueri daftar membaca `POOLDATA.M_KOMUNIKASI_PNC` dengan penyaring yang sama persis
-- kecuali satu:
--
--   list_not_answered   REPLYMESSAGE IS NULL       ORDER BY CREATEDDATE ASC
--   list_answered       REPLYMESSAGE IS NOT NULL   ORDER BY CREATEDATEREPLY DESC
--
-- Keduanya ditambah `CASEID = :1`, batas cabang, `SENDER IS NOT NULL`, dan
-- `MESSAGE IS NOT NULL`.
--
-- Urutannya BERLAWANAN, dan itu bukan kekeliruan penyalinan: yang belum dijawab dibaca dari
-- yang paling lama menunggu, yang sudah dijawab dari yang paling baru dibalas. Keduanya apa
-- adanya dari `GetInboxKomunikasiCabang-SQL.xml` dan
-- `GetInboxKomunikasiCabangAnswered-SQL.xml`.
--
-- ============================================================================
-- PEMETAAN KOLOM — alias Pega -> kolom sebenarnya -> alias di sini
-- ============================================================================
--
-- Kueri lama mengalias hampir setiap kolom dengan nama yang menyebut hal lain. Tidak satu
-- pun dibawa (`D-19`):
--
--   alias Pega           kolom              isi sebenarnya               alias di sini
--   -------------------- ------------------ ---------------------------- ----------------
--   CloseClaimDate       CREATEDDATE        tanggal pesan                CREATED_AT
--   Email                MESSAGE            isi pesan                    MESSAGE
--   UserTeknis           SENDER             Operator ID pengirim         SENDER_OPERATOR
--   pzInsKey             CASEID             kanal percakapan             (tidak dipilih)
--   ClaimNo              KOMUNIKASIID       nomor percakapan             CONVERSATION_ID
--   CloseClaimNote       REPLYMESSAGE       isi balasan                  REPLY_MESSAGE
--   StatusClaim          KOMUNIKASISTATUS   status percakapan            STATUS
--   AnalystTransferDate  CREATEDATEREPLY    tanggal balasan              REPLIED_AT
--   UserAdmin            REPLYFROMNAME      nama penjawab                REPLIER_NAME
--   UserName             COMMUNICATE_FROM   kode asal                    ORIGIN_CODE
--   UserTeknisEmail      COMMUNICATE_TO     kode tujuan                  RECIPIENT_CODE
--
-- Tiga di antaranya patut disebut khusus:
--
--   * `ClaimNo` BUKAN nomor klaim. Tabel ini tidak memuat nomor klaim sama sekali.
--   * `Email` BUKAN alamat surel. Ia isi pesannya.
--   * `pzInsKey` BUKAN kunci objek kerja Pega. Ia penanda kanal, bernilai 'CABANG'.
--
-- ============================================================================
-- SATU ALIAS UNTUK DUA KOLOM — DAN ITU YANG MENENTUKAN APA YANG TAMPIL
-- ============================================================================
--
-- `GetInboxKomunikasiCabang-SQL.xml` memakai alias `UserName` DUA KALI dalam satu SELECT:
--
--   sendername       AS "UserName"    <- kolom ketiga
--   COMMUNICATE_FROM AS "UserName"    <- kolom terakhir, enam baris kemudian
--
-- Yang menang adalah yang TERAKHIR, dan itu terbukti dari pemakaiannya sendiri:
-- `PNCGetInboxKomunikasiCabang_Act` langkah 5 memeriksa `.UserName == "1"` lalu menimpanya
-- dengan `"PUSAT"` — perbandingan yang hanya masuk akal untuk kode asal, bukan nama orang.
--
-- `SENDERNAME` karena itu TIDAK DIPILIH sama sekali di sini. Memilihnya berarti membawa
-- kolom yang di layar lama pun tidak pernah tampil, lalu harus menjelaskan mengapa ia
-- diabaikan. Ia dinyatakan lewat PlannedDifferences.
--
-- ============================================================================
-- KODE ASAL TIDAK DITERJEMAHKAN DI SINI
-- ============================================================================
--
-- `COMMUNICATE_FROM` dan `COMMUNICATE_TO` dikembalikan MENTAH; penerjemahannya menjadi
-- "PUSAT"/"CABANG" dikerjakan `inboxkomunikasicabang.OriginOf` dan `RecipientOf` di lapisan
-- domain.
--
-- Alasannya: penyimpanan SQL dan penyimpanan memori wajib menghasilkan teks yang sama
-- persis, dan dua penerjemah di dua tempat dapat menyimpang tanpa ketahuan. Sistem lama pun
-- menerjemahkannya di luar SQL — di `PNCGetInboxKomunikasiCabang_Act` langkah 5–7, bukan di
-- kuerinya.
--
-- ============================================================================
-- BATAS CABANG
-- ============================================================================
--
-- Sistem lama merangkai penyaringnya sebagai TEKS lalu menyisipkannya lewat `{ASIS:}`:
--
--   " (COMMUNICATE_TO = '"+Local.IDCABANG+"' OR COMMUNICATE_FROM = '"+Local.IDCABANG+"') "
--
-- Di sini ia menjadi predikat tetap dengan parameter binding. Perangkaian teks tidak
-- direplikasi: yang direplikasi adalah perilaku bisnis, bukan celah injeksi
-- (`08-TECHNICAL-STRATEGY.md` §4.3).
--
-- Perhatikan ia `OR`, bukan `AND`. Sebuah percakapan terlihat oleh cabang yang MENGIRIM
-- maupun yang MENERIMA; menukarnya dengan `AND` mengosongkan seluruh layar.
--
-- Kode yang dipakai DUA KALI di dalam satu predikat sengaja diikat DUA penanda bind
-- (`:2` dan `:3`) yang keduanya diisi nilai yang sama, bukan satu penanda yang dirujuk dua
-- kali. Alasannya ada di CATATAN PENANDA BIND di bawah.
--
-- ============================================================================
-- YANG BERUBAH DARI SISTEM LAMA, DAN KENAPA
-- ============================================================================
--
-- 1. PAGINASI DIKERJAKAN BASIS DATA.
--    Grid lama menarik seluruh barisnya lebih dulu lalu menomori halamannya di klipboard.
--    Di sini halamannya dipotong dengan `OFFSET … FETCH NEXT … ROWS ONLY` sebelum baris
--    meninggalkan basis data — didukung Oracle 12c+ dan PostgreSQL
--    (`09-DATABASE-STRATEGY.md` §3.3). Ukuran halamannya tetap 20, sama dengan
--    `<pyPageSize>20</pyPageSize>` pada section.
--
-- 2. JUMLAH SELURUH BARIS DIHITUNG `COUNT(*) OVER ()`.
--    Satu perjalanan, bukan dua. Fungsi jendela dihitung SEBELUM `OFFSET … FETCH` dipakai,
--    sehingga angkanya jumlah seluruhnya — bukan jumlah baris di halaman ini.
--
-- 3. PEMUTUS SERI DITAMBAHKAN PADA `ORDER BY`.
--    Kueri lama mengurutkan hanya menurut tanggal. Dua baris bertanggal sama karena itu
--    dapat berpindah urutan di antara dua permintaan — dan pada daftar yang dipaginasi,
--    urutan yang tidak tetap membuat sebuah baris terlewat di halaman satu lalu muncul dua
--    kali di halaman dua. `KOMUNIKASIID` dipakai sebagai pemutus seri. Ia TIDAK mengubah
--    baris mana yang tampil, hanya memastikannya tetap.
--
-- 4. GABUNGAN DITULIS SEBAGAI `JOIN`, BUKAN DAFTAR TABEL BERKOMA.
--    `GetDocumentKomunikasi1-SQL.xml` menggabungkan empat tabel lewat koma pada `FROM`.
--    `09-DATABASE-STRATEGY.md` §4 menuntut `JOIN` eksplisit demi portabilitas, dan bentuk
--    itu juga membuat arah sambungannya terbaca.
--
-- ============================================================================
-- YANG SENGAJA TIDAK BERUBAH — termasuk yang tampak seperti cacat
-- ============================================================================
--
-- * PENCACAH MEMERIKSA DUA KOLOM, GRID HANYA SATU.
--   `GetCountKomunikasiCabangAnswered` menyaring `REPLYFROM IS NOT NULL AND REPLYMESSAGE IS
--   NOT NULL`; kedua kueri grid hanya memeriksa `REPLYMESSAGE`. Akibatnya percakapan yang
--   dibalas TANPA penjawab tercatat MUNCUL di tabel tetapi TIDAK terhitung di pencacah mana
--   pun — bukan di "Answered" karena `REPLYFROM` kosong, bukan pula di "Not Answered" karena
--   `REPLYMESSAGE` terisi.
--
--   Keduanya dibawa apa adanya (`P-5`). Menyeragamkannya akan mengubah angka yang dilihat
--   pengguna hari ini, dan itu selisih yang belum diputuskan siapa pun.
--
-- * PENCACAH TIDAK MENYARING PENGIRIM DAN PESAN.
--   Kedua kueri pencacah tidak memuat `SENDER IS NOT NULL` maupun `MESSAGE IS NOT NULL`,
--   padahal kedua kueri grid memilikinya. Baris yang pengirimnya kosong karena itu ikut
--   terhitung tetapi tidak tampil. Dibawa apa adanya, sebab yang sama.
--
-- * `CASEID` DIBANDINGKAN PERSIS, BUKAN DENGAN `LIKE`.
--   Nilai penutupnya `CABANG SELESAI` BERAWALAN `CABANG`, sehingga penyaring berbasis awalan
--   akan meloloskan percakapan yang justru sudah ditutup. Perbandingan persis inilah yang
--   membuat baris hilang dari layar begitu percakapannya selesai.
--
-- CATATAN PENANDA BIND. Berkas ini memakai gaya Oracle `:n`, sama seperti seluruh modul lain
-- di aplikasi ini. Ia BELUM portabel ke PostgreSQL yang memakai `$n`; itu utang yang sudah
-- ada sebelum modul ini dan berlaku untuk seluruh berkas .sql di sini.
--
-- PENOMORANNYA MENAIK MENURUT URUTAN KEMUNCULAN, dan itu BUKAN kerapian. Oracle mengikat
-- argumen menurut URUTAN KEMUNCULAN penanda di dalam teks kueri, BUKAN menurut angka pada
-- `:n` — `docs/catatan-pengembangan.md` §19.14 mencatat cacat nyata yang lahir dari
-- melupakannya: penyaring cabang menerima NULL sementara pencacah menerima kode cabang,
-- tanpa satu pun galat karena setiap bind tetap terisi sesuatu.
--
-- Itu pula sebabnya kode cabang yang dipakai dua kali diikat DUA penanda: satu penanda yang
-- dirujuk dua kali akan membuat urutan kemunculan tidak lagi sama dengan urutan argumennya.

-- name: list_not_answered
-- Tab "Belum Dijawab" — percakapan yang pesannya belum dibalas.
-- — RDB List/GetInboxKomunikasiCabang-SQL.xml
--
-- REPLY_MESSAGE, REPLIER_NAME, dan REPLIED_AT pada kueri ini SELALU kosong: penyaringnya
-- `IS NULL`. Ketiganya tetap dipilih, bukan diganti NULL tetap, supaya kedua kueri daftar
-- punya bentuk yang sama persis dan satu pemindai Go dapat melayani keduanya.
--
-- Bind: :1 kanal percakapan · :2 kode cabang (tujuan) · :3 kode cabang (asal)
--       :4 offset · :5 jumlah baris
SELECT k.KOMUNIKASIID        AS CONVERSATION_ID,
       k.CREATEDDATE         AS CREATED_AT,
       k.COMMUNICATE_FROM    AS ORIGIN_CODE,
       k.SENDER              AS SENDER_OPERATOR,
       k.MESSAGE             AS MESSAGE,
       k.REPLYMESSAGE        AS REPLY_MESSAGE,
       k.REPLYFROMNAME       AS REPLIER_NAME,
       k.COMMUNICATE_TO      AS RECIPIENT_CODE,
       k.KOMUNIKASISTATUS    AS STATUS,
       k.CREATEDATEREPLY     AS REPLIED_AT,
       COUNT(*) OVER ()      AS TOTAL_ROWS
  FROM POOLDATA.M_KOMUNIKASI_PNC k
 WHERE k.CASEID = :1
   AND (k.COMMUNICATE_TO = :2 OR k.COMMUNICATE_FROM = :3)
   AND k.SENDER IS NOT NULL
   AND k.MESSAGE IS NOT NULL
   AND k.REPLYMESSAGE IS NULL
 ORDER BY k.CREATEDDATE ASC, k.KOMUNIKASIID ASC
OFFSET :4 ROWS FETCH NEXT :5 ROWS ONLY

-- name: list_answered
-- Tab "Sudah Dijawab" — percakapan yang sudah dibalas.
-- — RDB List/GetInboxKomunikasiCabangAnswered-SQL.xml
--
-- Urutannya MENURUN menurut tanggal balasan, berlawanan dengan kueri di atas. Itu perilaku
-- sistem lama apa adanya.
--
-- Bind: :1 kanal percakapan · :2 kode cabang (tujuan) · :3 kode cabang (asal)
--       :4 offset · :5 jumlah baris
SELECT k.KOMUNIKASIID        AS CONVERSATION_ID,
       k.CREATEDDATE         AS CREATED_AT,
       k.COMMUNICATE_FROM    AS ORIGIN_CODE,
       k.SENDER              AS SENDER_OPERATOR,
       k.MESSAGE             AS MESSAGE,
       k.REPLYMESSAGE        AS REPLY_MESSAGE,
       k.REPLYFROMNAME       AS REPLIER_NAME,
       k.COMMUNICATE_TO      AS RECIPIENT_CODE,
       k.KOMUNIKASISTATUS    AS STATUS,
       k.CREATEDATEREPLY     AS REPLIED_AT,
       COUNT(*) OVER ()      AS TOTAL_ROWS
  FROM POOLDATA.M_KOMUNIKASI_PNC k
 WHERE k.CASEID = :1
   AND (k.COMMUNICATE_TO = :2 OR k.COMMUNICATE_FROM = :3)
   AND k.SENDER IS NOT NULL
   AND k.MESSAGE IS NOT NULL
   AND k.REPLYMESSAGE IS NOT NULL
 ORDER BY k.CREATEDATEREPLY DESC, k.KOMUNIKASIID DESC
OFFSET :4 ROWS FETCH NEXT :5 ROWS ONLY

-- name: count_answered
-- Pencacah "Answered" pada diagram di atas grid.
-- — RDB List/GetCountKomunikasiCabangAnswered-SQL.xml
--
-- DUA kolom balasan diperiksa, berbeda dari kueri grid yang hanya memeriksa satu. Lihat
-- catatan di kepala berkas.
--
-- Bind: :1 kanal percakapan · :2 kode cabang (tujuan) · :3 kode cabang (asal)
SELECT COUNT(*) AS TOTAL_ROWS
  FROM POOLDATA.M_KOMUNIKASI_PNC k
 WHERE k.CASEID = :1
   AND (k.COMMUNICATE_TO = :2 OR k.COMMUNICATE_FROM = :3)
   AND k.REPLYFROM IS NOT NULL
   AND k.REPLYMESSAGE IS NOT NULL

-- name: count_not_answered
-- Pencacah "Not Answered" pada diagram di atas grid.
-- — RDB List/GetCountKomunikasiCabangNotAnswered-SQL.xml
--
-- Bind: :1 kanal percakapan · :2 kode cabang (tujuan) · :3 kode cabang (asal)
SELECT COUNT(*) AS TOTAL_ROWS
  FROM POOLDATA.M_KOMUNIKASI_PNC k
 WHERE k.CASEID = :1
   AND (k.COMMUNICATE_TO = :2 OR k.COMMUNICATE_FROM = :3)
   AND k.REPLYFROM IS NULL
   AND k.REPLYMESSAGE IS NULL

-- name: detail_header
-- Keberadaan dan arah sebuah percakapan — dipakai layar "Detail Komunikasi".
--
-- KENAPA IA ADA, PADAHAL UTASNYA DIBACA KUERI LAIN
--
-- Karena sejak utas dibaca dari `M_KOMUNIKASI_CABANG` (lihat detail_thread), utas yang KOSONG
-- tidak lagi berarti "percakapan tidak ada". Percakapan yang dibuat lewat jalur lain — layar
-- klaim, atau data warisan sebelum tabel riwayat dipakai — punya baris header tanpa satu pun
-- baris riwayat, dan layar detailnya memang kosong.
--
-- Tanpa pemeriksaan terpisah ini, keduanya akan dijawab "tidak ditemukan": percakapan yang
-- benar-benar tidak ada, DAN percakapan nyata yang riwayatnya belum pernah ditulis. Yang
-- kedua akan dilaporkan sebagai kerusakan.
--
-- Ia juga satu-satunya sumber ARAH percakapan: `M_KOMUNIKASI_CABANG` tidak punya kolom asal
-- maupun tujuan sama sekali.
--
-- Bind: :1 nomor percakapan · :2 kode cabang (tujuan) · :3 kode cabang (asal)
SELECT k.COMMUNICATE_FROM AS ORIGIN_CODE,
       k.COMMUNICATE_TO   AS RECIPIENT_CODE
  FROM POOLDATA.M_KOMUNIKASI_PNC k
 WHERE k.KOMUNIKASIID = :1
   AND (k.COMMUNICATE_TO = :2 OR k.COMMUNICATE_FROM = :3)

-- name: detail_thread
-- Utas satu percakapan — layar "Detail Komunikasi".
--
-- ============================================================================
-- DIBACA DARI TABEL RIWAYAT, BUKAN DARI TABEL PERCAKAPAN
-- ============================================================================
--
-- Ini KOREKSI atas versi sebelumnya, yang membaca `M_KOMUNIKASI_PNC`. Keterangan Work Owner
-- pada 2026-09-24: layar detail membaca `M_KOMUNIKASI_CABANG` — tabel yang sama yang diisi
-- "Kirim Pesan" dan "Balas".
--
-- Itu menjelaskan seluruh rancangannya, dan sekaligus menutup pertanyaan terbuka yang sempat
-- dicatat ("apakah M_KOMUNIKASI_CABANG dibaca siapa pun"):
--
--   M_KOMUNIKASI_PNC      SATU baris per percakapan — kepala. Isinya pesan TERAKHIR dan
--                         balasan TERAKHIR, dan itulah yang digambar kedua grid.
--   M_KOMUNIKASI_CABANG   SATU baris per UCAPAN — utasnya. Setiap pesan dan setiap balasan
--                         menambah satu baris, dan itulah yang digambar layar detail.
--
-- Versi sebelumnya karena itu menggambar utas yang SELALU berisi tepat satu baris, betapapun
-- panjang percakapannya.
--
-- ============================================================================
-- KUERI ASLINYA MASIH HILANG DARI EXPORT
-- ============================================================================
--
-- `Section/DETAILKOMUNIKASICABANG_ACT-Act.xml` — sebuah Activity yang tersimpan di direktori
-- `Section/` — menjalankan tiga langkah:
--
--   Page-New   TEMPHISTORYCABANGDETAIL          (kelas Code-Pega-List)
--   RDB-List   GetInboxKomunikasiCabang_detail  -> TEMPHISTORYCABANGDETAIL.pxResults
--   Property-Set
--
-- `GetInboxKomunikasiCabang_detail-SQL.xml` TIDAK ADA di `RDB List/` (`R-16`).
--
-- Yang terbaca dari `Section/BalasKomunikasiCabang-Section.xml` adalah ketiga kolomnya
-- beserta properti yang digambarnya:
--
--   Tanggal   -> .CloseClaimDate  (pxDateTime)
--   Pengirim  -> .UserName
--   Pesan     -> .Email
--
-- Ketiganya alias yang SAMA dengan yang dipakai `GetInboxKomunikasiCabang` terhadap
-- `M_KOMUNIKASI_PNC` — `createddate AS "CloseClaimDate"`, `sender...AS "UserName"`,
-- `message AS "Email"` — sehingga pemetaan kolomnya dapat disimpulkan dengan yakin.
--
-- ============================================================================
-- SATU NAMA KOLOM YANG DITEBAK
-- ============================================================================
--
-- `ReplyKomunikasiCabang` mengisi EMPAT kolom saja: SENDER, MESSAGE, KOMUNIKASIID,
-- KODECABANG. Tidak ada kolom tanggal di antaranya, sehingga tanggalnya diisi basis data —
-- persis seperti `CREATEDDATE` pada `M_KOMUNIKASI_PNC`, yang juga tidak pernah diisi INSERT.
--
-- NAMANYA DITEBAK `CREATEDDATE`, dengan dua dasar: konvensi tabel saudaranya di skema yang
-- sama, dan alias `.CloseClaimDate` yang di kueri sebelah memang berasal dari `createddate`.
-- DDL-nya belum ada (`R-08`), sehingga tebakan itu tidak dapat diperiksa dari export.
--
-- Bila tebakannya salah, kegagalannya KERAS dan langsung terlihat — Oracle menolak dengan
-- "invalid identifier", dan layar detail menampilkan galat. Itu mode kegagalan yang jauh
-- lebih baik daripada diam: tidak ada baris yang salah tampil, dan perbaikannya satu kata di
-- berkas ini. `-periksa` menjalankannya terhadap basis data sungguhan justru untuk itu.
--
-- ============================================================================
-- BATAS CABANG LEWAT GABUNGAN, KARENA TABELNYA TIDAK PUNYA
-- ============================================================================
--
-- `M_KOMUNIKASI_CABANG` tidak memuat COMMUNICATE_TO maupun COMMUNICATE_FROM. Batas cabang
-- karena itu ditegakkan lewat gabungan ke tabel kepala.
--
-- Ia TETAP ada meski detail_header sudah memeriksanya lebih dulu. Gabungannya tidak menambah
-- biaya yang berarti, dan pernyataan yang aman berdiri sendiri tidak bergantung pada urutan
-- pemanggilan yang dapat berubah kemudian.
--
-- ============================================================================
-- URUTAN TANPA PEMUTUS SERI
-- ============================================================================
--
-- `ORDER BY CREATEDDATE ASC` saja — utas dibaca dari yang paling awal. Tidak ada pemutus seri
-- karena tidak ada kolom lain yang dapat dipakai: keempat kolom yang diketahui tidak ada yang
-- unik per baris.
--
-- Akibatnya dua ucapan bertanggal SAMA dapat berpindah urutan di antara dua pembukaan. Itu
-- diterima: layar detail tidak dipaginasi, sehingga seluruh barisnya tampil sekaligus dan
-- tidak ada baris yang dapat terlewat karenanya.
--
-- Bila DDL-nya kelak tiba dan tabelnya ternyata punya kunci, kunci itu layak ditambahkan
-- sebagai pemutus seri.
--
-- Bind: :1 nomor percakapan · :2 nomor percakapan (gabungan)
--       :3 kode cabang (tujuan) · :4 kode cabang (asal)
SELECT c.CREATEDDATE AS CREATED_AT,
       c.SENDER      AS SENDER_OPERATOR,
       c.MESSAGE     AS MESSAGE
  FROM POOLDATA.M_KOMUNIKASI_CABANG c
  JOIN POOLDATA.M_KOMUNIKASI_PNC k ON k.KOMUNIKASIID = c.KOMUNIKASIID
 WHERE c.KOMUNIKASIID = :1
   AND k.KOMUNIKASIID = :2
   AND (k.COMMUNICATE_TO = :3 OR k.COMMUNICATE_FROM = :4)
 ORDER BY c.CREATEDDATE ASC

-- name: detail_attachments
-- Lampiran satu percakapan.
-- — RDB List/GetDocumentKomunikasi1-SQL.xml
--
-- Kueri lama menggabungkan EMPAT tabel lewat koma pada `FROM`; di sini keempatnya menjadi
-- `JOIN` eksplisit. Yang berubah hanyalah bentuk gabungannya, bukan barisnya:
--
--   from POOLDATA.D_KOMUNIKASI_PNC A, POOLDATA.M_KOMUNIKASI_PNC B,
--        pooldata.v_lst_det_type_doc c, pooldata.v_lst_doc_type d
--   WHERE A.KOMUNIKASI_ID = B.KOMUNIKASIID
--     AND B.KOMUNIKASIID = {getkomunikasi.BRANCHNAME}
--     and A.CATEGORYID = D.ID
--     and A.TYPEID = C.ID
--
-- ============================================================================
-- GABUNGAN KE MASTER TETAP `INNER`, DAN ITU MENYEMBUNYIKAN BARIS
-- ============================================================================
--
-- Lampiran yang `CATEGORYID` atau `TYPEID`-nya tidak ada di master TIDAK MUNCUL sama
-- sekali. Dengan `LEFT JOIN` ia akan muncul dengan nama jenis kosong.
--
-- Bentuk `INNER` dipertahankan (`P-5`): mengganti ke `LEFT JOIN` akan MENAMBAH baris yang
-- di Pega tidak pernah terlihat, dan itu perubahan perilaku pada layar yang sedang diuji
-- kesetaraannya. Ia dicatat sebagai pertanyaan terbuka, bukan diperbaiki sepihak.
--
-- Gabungan ke `M_KOMUNIKASI_PNC` dipertahankan pula — kueri lama memakainya, dan di sini ia
-- punya guna tambahan: ia yang membawa kolom batas cabang, sehingga lampiran tidak dapat
-- dibaca lewat nomor percakapan milik cabang lain.
--
-- Bind: :1 nomor percakapan · :2 kode cabang (tujuan) · :3 kode cabang (asal)
SELECT d.DOCUMENTID          AS DOCUMENT_ID,
       t.TYPE_DOCUMENT       AS TYPE_NAME,
       c.DETAIL_DOCUMENT     AS DETAIL_NAME,
       d.NOTE                AS NOTE,
       d.UPLOADDATE          AS UPLOADED_AT
  FROM POOLDATA.D_KOMUNIKASI_PNC d
       INNER JOIN POOLDATA.M_KOMUNIKASI_PNC k
               ON k.KOMUNIKASIID = d.KOMUNIKASI_ID
       INNER JOIN POOLDATA.V_LST_DOC_TYPE t
               ON t.ID = d.CATEGORYID
       INNER JOIN POOLDATA.V_LST_DET_TYPE_DOC c
               ON c.ID = d.TYPEID
 WHERE d.KOMUNIKASI_ID = :1
   AND (k.COMMUNICATE_TO = :2 OR k.COMMUNICATE_FROM = :3)
 ORDER BY d.UPLOADDATE ASC, d.DOCUMENTID ASC

-- name: check_table
-- Memastikan tabel percakapan dapat dibaca akun aplikasi.
--
-- Ia dijalankan dengan kanal karangan yang pasti tidak ada, sehingga tidak mengembalikan
-- satu baris pun — yang diuji adalah KETERBACAAN tabelnya, bukan isinya.
SELECT COUNT(*) AS TOTAL_ROWS
  FROM POOLDATA.M_KOMUNIKASI_PNC k
 WHERE k.CASEID = :1

-- name: check_attachment_table
-- NILAI SENTINELNYA WAJIB BERUPA ANGKA — `KOMUNIKASI_ID` bertipe NUMBER, terbukti 2026-09-25.
-- Memastikan ketiga tabel lampiran dapat dibaca akun aplikasi.
--
-- Terpisah dari check_table karena kegagalannya berbeda artinya: yang ini menunjuk tabel
-- lampiran atau salah satu master jenis dokumen, bukan tabel percakapan. Layar tetap dapat
-- dipakai tanpa lampiran, sehingga kedua kegagalan itu tidak boleh terbaca sama.
SELECT COUNT(*) AS TOTAL_ROWS
  FROM POOLDATA.D_KOMUNIKASI_PNC d
       INNER JOIN POOLDATA.V_LST_DOC_TYPE t
               ON t.ID = d.CATEGORYID
       INNER JOIN POOLDATA.V_LST_DET_TYPE_DOC c
               ON c.ID = d.TYPEID
 WHERE d.KOMUNIKASI_ID = :1

-- ============================================================================
-- PERNYATAAN YANG MENULIS — ditambahkan 2026-09-24
-- ============================================================================
--
-- Sampai tanggal itu berkas ini TIDAK memuat satu pun penulisan, dan kepala berkas ini
-- menyatakannya. Keputusan Work Owner mengubahnya: Balas dan Selesai Komunikasi dibangun.
--
-- Yang berubah bukan sekadar dua pernyataan melainkan PEMILIK TABEL. `P-1` menetapkan tepat
-- satu sistem menulis sebuah tabel selama masa paralel, dan kepemilikan berpindah saat
-- modulnya pindah. Sejak keputusan ini, POOLDATA.M_KOMUNIKASI_PNC dan
-- POOLDATA.M_KOMUNIKASI_CABANG ditulis SISTEM BARU — dan layar Pega yang sama wajib berhenti
-- menulisinya.
--
-- SELURUH penulisan di bawah membawa batas cabang di klausa WHERE-nya. Sistem lama TIDAK
-- memeriksanya — tombolnya hanya ada pada baris yang sudah tersaring — tetapi tombol bukan
-- penjagaan: nomor percakapan berurutan dan mudah ditebak, dan permintaan dapat disusun
-- tanpa layar sama sekali.

-- name: reply_update
-- Menyimpan balasan atas sebuah percakapan.
-- — RDB List/ReplyKomunikasi-SQL.xml (pySaveSQL), dijalankan PNCReplyMessageCabang langkah 3
--
-- Aslinya, kata demi kata:
--
--   update POOLDATA.m_komunikasi_pnc
--      set replymessage    = {tempReply.ADDRESS},
--          replyfrom       = {OperatorID.pyUserIdentifier},
--          CREATEDATEREPLY = {temp.AnalystTransferDate DateTime},
--          replyfromname   = {OperatorID.pyUserName},
--          komunikasistatus= '1'
--    where komunikasiid = {tempReply.M_SURVEY_ID}
--
-- EMPAT hal yang BERBEDA dari aslinya, dan keempatnya disengaja:
--
--  1. BATAS CABANG ditambahkan ke WHERE. Tanpanya, nomor percakapan milik cabang lain dapat
--     dibalas dari layar cabang mana pun.
--
--  2. `komunikasistatus` tetap '1' tetapi diikat sebagai PARAMETER, bukan literal. Nilainya
--     hidup sebagai konstanta domain (`StatusAnswered`) supaya penyimpanan memori memakai
--     nilai yang sama persis.
--
--  3. Waktu balasan datang dari APLIKASI, bukan dari basis data. `SYSDATE` dilarang
--     (`09-DATABASE-STRATEGY.md` §4), dan waktu yang lahir di dua tempat tidak dapat diuji
--     secara deterministik.
--
--  4. `CASEID = :7` — percakapan yang SUDAH DITUTUP tidak dapat dibalas.
--
--     Aslinya tidak punya syarat ini, dan di Pega ia memang tidak dibutuhkan: layar balasan
--     hanya dapat dicapai lewat tombol Detail pada grid, dan grid sudah menyaring
--     `CASEID = 'CABANG'`. Di sistem baru alamatnya dapat dipanggil langsung, sehingga
--     penjagaannya harus ada di pernyataannya sendiri.
--
--     Tanpanya, membalas percakapan yang sudah ditutup akan "berhasil" — menyetel
--     `komunikasistatus = '1'` pada baris yang tetap tidak muncul di kedua tab, lalu
--     dijawab layar dengan kalimat "percakapan ini berpindah ke tab Sudah Dijawab" yang
--     tidak terjadi. Ia satu-satunya penyimpangan di berkas ini yang MENAMBAH penolakan
--     bukan demi keamanan melainkan demi kejujuran jawaban.
--
-- Bind: :1 isi balasan · :2 penjawab (login) · :3 waktu balasan · :4 nama penjawab
--       :5 status · :6 nomor percakapan · :7 kanal berjalan
--       :8 kode cabang (tujuan) · :9 kode cabang (asal)
--
-- CATATAN PORTABILITAS. Tabelnya sengaja TIDAK diberi alias. Oracle mengizinkan
-- `UPDATE tabel alias SET alias.kolom = …`, PostgreSQL TIDAK — ia menolak awalan alias pada
-- klausa SET. Bentuk tanpa alias sah di keduanya, dan `D-20` menuntut satu set SQL yang
-- berjalan di keduanya.
UPDATE POOLDATA.M_KOMUNIKASI_PNC
   SET REPLYMESSAGE     = :1,
       REPLYFROM        = :2,
       CREATEDATEREPLY  = :3,
       REPLYFROMNAME    = :4,
       KOMUNIKASISTATUS = :5
 WHERE KOMUNIKASIID = :6
   AND CASEID = :7
   AND (COMMUNICATE_TO = :8 OR COMMUNICATE_FROM = :9)

-- name: reply_history_insert
-- Mencatat balasan cabang ke tabel riwayatnya.
-- — RDB List/ReplyKomunikasiCabang-SQL.xml, dijalankan PNCReplyMessageCabang langkah 4
--
-- Aslinya:
--
--   insert into pooldata.m_komunikasi_cabang(sender,message,komunikasiid,kodecabang)
--   values ({OperatorID.pyUserIdentifier},{tempReply.ADDRESS},
--           {tempReply.M_SURVEY_ID},{tempReply.NAME})
--
-- PERANGKAP PENAMAAN: kolom `kodecabang` TIDAK menerima kode cabang. `tempReply.NAME` diisi
-- `Param.kodecabang`, yang dikirim section sebagai `TempView2.City` — dan itu berasal dari
-- `DetailKomunikasi_dt` yang mengisinya dari `.pzInsKey`, yakni `CASEID`. Jadi yang tersimpan
-- di sana adalah PENANDA KANAL (`CABANG`), bukan kode cabang.
--
-- Ia direplikasi apa adanya (`P-5`): mengisinya dengan kode cabang yang sebenarnya akan
-- membuat baris baru berbeda isi dari seluruh baris yang sudah ada di tabel itu.
--
-- Bind: :1 pengirim (login penjawab) · :2 isi balasan · :3 nomor percakapan · :4 penanda kanal
INSERT INTO POOLDATA.M_KOMUNIKASI_CABANG (SENDER, MESSAGE, KOMUNIKASIID, KODECABANG)
VALUES (:1, :2, :3, :4)

-- name: finish_update
-- Menutup sebuah percakapan — tombol "Selesai Komunikasi".
-- — RDB List/ENDMessageCABANG_PNC-SQL.xml, dijalankan EndKomunikasiCabang langkah 2
--
-- Aslinya:
--
--   UPDATE POOLDATA.M_KOMUNIKASI_PNC
--      SET CASEID = {TempEndKomunikasi.CaseID}
--    WHERE KOMUNIKASIID = {TempEndKomunikasi.ClaimID}
--
-- Nilai barunya `CABANG SELESAI`, diisi activity-nya sendiri pada langkah 1.
--
-- DUA hal ditambahkan ke WHERE, dan keduanya menutup cacat yang nyata:
--
--  1. BATAS CABANG — sama seperti reply_update.
--
--  2. `CASEID = :2` — percakapan yang SUDAH ditutup tidak ditutup ulang. Tanpa syarat ini,
--     menekan tombolnya dua kali tetap "berhasil" dan menimpa nilai yang sama; dengan syarat
--     ini, penekanan kedua tidak mengubah satu baris pun dan pemanggil dapat mengatakannya.
--     Itu yang membedakan "sudah selesai" dari "tidak ditemukan".
--
-- Bind: :1 kanal penutup · :2 kanal berjalan · :3 nomor percakapan
--       :4 kode cabang (tujuan) · :5 kode cabang (asal)
-- Tanpa alias, dengan alasan portabilitas yang sama seperti reply_update.
UPDATE POOLDATA.M_KOMUNIKASI_PNC
   SET CASEID = :1
 WHERE CASEID = :2
   AND KOMUNIKASIID = :3
   AND (COMMUNICATE_TO = :4 OR COMMUNICATE_FROM = :5)

-- name: branch_options
-- Daftar cabang yang dapat dipilih sebagai tujuan pesan baru.
-- — pemilih cabang pada Section/InboxKomunikasi-Section.xml,
--   pySourceName = TempResultSurveyor.pxResults
--
-- KUERINYA TIDAK ADA DI EXPORT. Yang terbaca hanyalah:
--
--   * KELAS halamannya, dari Harness/InboxKomunikasiCabang-Harness.xml:
--     `TempResultSurveyor.pxResults` berkelas `ASM-FW-GCNMFW-Int-V_D_SURVEYORS`,
--     yakni view POOLDATA.V_D_SURVEYORS.
--   * KETIGA isian yang dipakai autocomplete-nya, dari section:
--     .BRANCHNAME (ditampilkan) · .BRANCH (kode) · .EMAIL (alamat surel).
--
-- Penyaring dan urutannya karena itu DITEBAK, dan tebakannya dinyatakan di sini supaya
-- dapat diperiksa: seluruh baris yang punya kode DAN nama, tanpa duplikat, urut menurut
-- nama. Bila daftar yang tampil berbeda dari Pega, inilah tempat pertama yang diperiksa.
--
-- DISTINCT dipakai karena V_D_SURVEYORS adalah view SURVEYOR, bukan view cabang: satu
-- cabang hampir pasti muncul berkali-kali, sekali per surveyor. Tanpa DISTINCT, pemilihnya
-- menampilkan nama cabang yang sama berulang-ulang.
--
-- Bind: (tidak ada)
SELECT DISTINCT s.BRANCH     AS BRANCH_CODE,
                s.BRANCHNAME AS BRANCH_NAME,
                s.EMAIL      AS BRANCH_EMAIL
  FROM POOLDATA.V_D_SURVEYORS s
 WHERE s.BRANCH IS NOT NULL
   AND s.BRANCHNAME IS NOT NULL
 ORDER BY s.BRANCHNAME ASC

-- name: message_insert
-- Membuat percakapan BARU.
-- — RDB List/InsertMessageCABANG_PNC-SQL.xml (pySaveSQL),
--   dijalankan PNCSendMessageKomunikasiCabang
--
-- Aslinya, kata demi kata:
--
--   INSERT INTO POOLDATA.M_KOMUNIKASI_PNC(
--       CASEID, SENDER, MESSAGE, SENDERNAME, KOMUNIKASISTATUS,
--       COMMUNICATE_TO, COMMUNICATE_FROM)
--   VALUES ({TempEmail.IDSurvey},      {TempEmail.BodyLetterAttn},
--           {TempEmail.BodyLetterEmail},{TempEmail.BodyLetterOP},
--           {TempEmail.BranchToTransfer},
--           {TempEmail.InsuredPIC},    {TempEmail.IDSurveyCase})
--
-- TUJUH NAMA PROPERTI, TUJUH-TUJUHNYA MENYESATKAN. Pemetaannya, dibaca dari langkah
-- Property-Set activity-nya:
--
--   TempEmail.IDSurvey         = "CABANG"                     -> CASEID
--   TempEmail.BodyLetterAttn   = OperatorID.pyUserIdentifier  -> SENDER      (login)
--   TempEmail.BodyLetterEmail  = TempInputKomunikasi.City     -> MESSAGE     (isi pesan!)
--   TempEmail.BodyLetterOP     = OperatorID.pyUserName        -> SENDERNAME  (nama)
--   TempEmail.BranchToTransfer = "0"                          -> KOMUNIKASISTATUS
--   TempEmail.InsuredPIC       = "1" | ...DistrictID          -> COMMUNICATE_TO
--   TempEmail.IDSurveyCase     = KodeCabang | "1"             -> COMMUNICATE_FROM
--
-- Tidak satu pun dibawa (`D-19`). Yang paling berbahaya bila dipercaya: `BodyLetterEmail`
-- BUKAN alamat surel melainkan ISI PESAN, dan `BranchToTransfer` BUKAN cabang melainkan
-- STATUS.
--
-- `KOMUNIKASISTATUS = '0'` di sini MEMBUKTIKAN arti kode itu: pesan yang baru dibuat belum
-- dijawab. Ia diikat sebagai parameter dari konstanta domain StatusNotAnswered, bukan
-- literal, supaya penyimpanan memori memakai nilai yang sama persis.
--
-- KOLOM CREATEDDATE TIDAK DISEBUT. Tanggalnya karena itu diisi basis data — lewat default
-- kolom atau trigger yang tidak terbaca dari export (`R-08`). Itu direplikasi apa adanya:
-- mengisinya dari aplikasi akan membuat baris baru berbeda perlakuan dari seluruh baris yang
-- sudah ada, dan kolom itu DASAR PENGURUTAN tab "Belum Dijawab".
--
-- Bind: :1 kanal · :2 pengirim (login) · :3 isi pesan · :4 nama pengirim
--       :5 status · :6 tujuan · :7 asal
INSERT INTO POOLDATA.M_KOMUNIKASI_PNC
       (CASEID, SENDER, MESSAGE, SENDERNAME, KOMUNIKASISTATUS,
        COMMUNICATE_TO, COMMUNICATE_FROM)
VALUES (:1, :2, :3, :4, :5, :6, :7)

-- name: message_max_id
-- Nomor percakapan yang baru saja terbit.
-- — RDB List/GetIDMaxKom_cabang-SQL.xml, dijalankan tepat sesudah message_insert
--
-- Aslinya:
--
--   select max(komunikasiid) AS "City" FROM pooldata.m_komunikasi_pnc
--
-- (Alias `"City"` untuk sebuah nomor percakapan. Tidak dibawa — `D-19`.)
--
-- KENAPA `max()` SAMA SEKALI. Karena message_insert TIDAK menyebut KOMUNIKASIID, sehingga
-- nomornya terbit di basis data dan aplikasi tidak mengetahuinya. `max()` adalah cara
-- sistem lama menemukannya kembali.
--
-- ITU POLA YANG TIDAK AMAN, dan di sinilah transaksi menjadi wajib: tanpa transaksi, dua
-- pengiriman yang berjalan bersamaan membaca nomor yang sama, dan riwayat yang satu tertaut
-- ke percakapan yang lain. Sistem lama menjalankannya sebagai dua langkah activity tanpa
-- transaksi apa pun.
--
-- Penggantinya yang benar adalah `INSERT ... RETURNING`, dan itu TIDAK dipakai: ia menuntut
-- mengetahui nama sequence atau trigger yang menerbitkan nomornya, dan DDL-nya belum ada
-- (`R-08`). Bila DDL itu tiba, pernyataan ini layak dicabut.
--
-- Bind: (tidak ada)
SELECT MAX(KOMUNIKASIID) AS LATEST_ID
  FROM POOLDATA.M_KOMUNIKASI_PNC

-- name: message_history_insert
-- Mencatat pesan baru ke tabel riwayat cabang.
-- — RDB List/ReplyKomunikasiCabang-SQL.xml, dijalankan PNCSendMessageKomunikasiCabang
--
-- Pernyataan yang SAMA dengan reply_history_insert, tetapi kolom `kodecabang`-nya menerima
-- NILAI YANG BERBEDA — dan ini temuan yang mengoreksi catatan sebelumnya:
--
--   dari "Balas"       tempReply.NAME = Param.kodecabang      -> CASEID ("CABANG")
--   dari "Kirim Pesan" tempReply.NAME = TempInputKomunikasi.DistrictID -> KODE CABANG
--
-- Jadi satu kolom memuat DUA konvensi, tergantung tombol mana yang menulisnya. Keduanya
-- direplikasi apa adanya (`P-5`): menyeragamkannya akan membuat baris baru berbeda isi dari
-- baris yang sudah ada, dan tidak ada satu pun layar yang membaca kolom ini sehingga tidak
-- ada yang dapat membuktikan mana yang "benar".
--
-- Pada tujuan PUSAT, kode cabangnya kosong — sama seperti yang dikirim layar lama ketika
-- pemilih cabangnya tidak diisi.
--
-- Bind: :1 pengirim (login) · :2 isi pesan · :3 nomor percakapan · :4 kode cabang tujuan
INSERT INTO POOLDATA.M_KOMUNIKASI_CABANG (SENDER, MESSAGE, KOMUNIKASIID, KODECABANG)
VALUES (:1, :2, :3, :4)

-- name: check_history_table
-- Memastikan tabel RIWAYAT dapat dibaca akun aplikasi, beserta kolom tanggalnya.
--
-- Ia TERPISAH dari check_table karena kegagalannya berbeda artinya: tanpa tabel percakapan
-- layar tidak dapat dipakai sama sekali; tanpa tabel riwayat daftar tetap tampil dan
-- percakapan tetap dapat dibalas — yang lumpuh hanya layar detail.
--
-- KOLOM TANGGALNYA IKUT DIPILIH DENGAN SENGAJA. Namanya DITEBAK (lihat detail_thread), dan
-- pemeriksaan yang hanya menghitung baris tidak akan menguji tebakan itu sama sekali.
-- Memilihnya membuat `-periksa` menjawab pertanyaan yang sebenarnya: apakah kolom bernama
-- CREATEDDATE memang ada di tabel ini.
--
-- NILAI SENTINELNYA WAJIB BERUPA ANGKA. `KOMUNIKASIID` pada tabel ini bertipe NUMBER —
-- terbukti pada 2026-09-25, ketika sentinel bertipe teks menghasilkan `ORA-01722: invalid
-- number` alih-alih jawaban. Pemeriksaan yang gagal karena nilai sentinelnya sendiri tidak
-- memeriksa apa pun, dan lebih buruk lagi: galatnya terbaca seolah tabelnya bermasalah.
--
-- Bind: :1 nomor percakapan (angka; dipakai sentinel yang tidak pernah ada)
SELECT COUNT(c.CREATEDDATE) AS TOTAL_ROWS
  FROM POOLDATA.M_KOMUNIKASI_CABANG c
 WHERE c.KOMUNIKASIID = :1
