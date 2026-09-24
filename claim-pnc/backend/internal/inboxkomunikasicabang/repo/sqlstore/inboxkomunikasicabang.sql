-- Kueri modul Inbox Komunikasi Cabang (`MENU_ID 70`, pengganti
-- `Harness/InboxKomunikasiCabang`).
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- SELURUH tabel yang dibaca berkas ini milik sistem lama. Tidak ada satu pun pernyataan yang
-- menulis, dan memang tidak boleh ada: selama masa paralel setiap tabel hanya boleh ditulis
-- SATU sistem, dan tabel-tabel ini milik Pega (`P-1`).
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

-- name: detail_thread
-- Utas satu percakapan — layar "Detail Komunikasi".
--
-- ============================================================================
-- KUERI ASLINYA HILANG DARI EXPORT
-- ============================================================================
--
-- `Section/DETAILKOMUNIKASICABANG_ACT-Act.xml` menjalankan RDB List bernama
-- `GetInboxKomunikasiCabang_detail`, dan berkas itu TIDAK ADA di direktori `RDB List/`
-- maupun di mana pun dalam export (kelas `R-16`).
--
-- Bentuk di bawah karena itu diturunkan dari yang MEMANG terbaca, bukan dikarang:
--
--   * `Data Transform/DetailKomunikasi_dt-DT.xml` mengirim DUA parameter dan hanya dua —
--     `Param.KOMID` (nomor percakapan) dan `Param.KODECABANG` (kode cabang). Itulah
--     seluruh masukan yang layar detail terima.
--   * `Section/BalasKomunikasiCabang-Section.xml` menggambar TIGA kolom: Tanggal,
--     Pengirim, Pesan.
--
-- Penyaringnya karena itu nomor percakapan ditambah batas cabang yang sama dengan grid —
-- keduanya parameter yang benar-benar dikirim. Tidak ada penyaring lain yang ditambahkan,
-- dan khususnya `CASEID` TIDAK disaring: percakapan yang sudah ditutup tetap dapat dibaca
-- utasnya bila nomornya diketahui, dan menutupnya adalah aturan yang tidak dapat dibaca
-- dari mana pun.
--
-- Bind: :1 nomor percakapan · :2 kode cabang (tujuan) · :3 kode cabang (asal)
SELECT k.CREATEDDATE         AS CREATED_AT,
       k.COMMUNICATE_FROM    AS ORIGIN_CODE,
       k.SENDER              AS SENDER_OPERATOR,
       k.MESSAGE             AS MESSAGE,
       k.REPLYMESSAGE        AS REPLY_MESSAGE,
       k.REPLYFROMNAME       AS REPLIER_NAME,
       k.CREATEDATEREPLY     AS REPLIED_AT
  FROM POOLDATA.M_KOMUNIKASI_PNC k
 WHERE k.KOMUNIKASIID = :1
   AND (k.COMMUNICATE_TO = :2 OR k.COMMUNICATE_FROM = :3)
 ORDER BY k.CREATEDDATE ASC

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
