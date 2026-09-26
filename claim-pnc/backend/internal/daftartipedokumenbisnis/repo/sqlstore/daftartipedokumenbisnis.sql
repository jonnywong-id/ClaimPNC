-- Kueri modul Daftar Tipe Dokumen Bisnis.
--
-- ============================================================================
-- BACA BAGIAN INI SEBELUM MENGUBAH SATU BARIS PUN DI BAWAHNYA.
-- ============================================================================
--
-- # Berbeda dari modul sebelumnya: jalur TULIS di sini terbaca, bukan diturunkan
--
-- Modul Daftar Detail Dokumen Travel harus MENURUNKAN nama tabel tulisnya dari nama
-- view, karena kedua activity penyimpannya hilang dari export (`R-16`). Modul ini tidak:
-- procedure-nya ada, lengkap dengan daftar kolomnya.
--
--   Database/PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:25-26   INSERT beserta 10 kolomnya
--   Database/PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:39-41   UPDATE beserta 8 kolomnya
--   Database/PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:57-58   INSERT COVERAGE_DOC_BUSINESS
--   Database/PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:22      bentuk ID dan nama urutannya
--
-- Nama tabel, nama kolom, nama urutan, dan lebar padding ID karena itu DIBACA. Yang
-- masih harus dipastikan DBA hanyalah LEBAR kolomnya dan tipe MIN_DOC — lihat
-- migrations/0010_daftar_tipe_dokumen_bisnis.up.sql.
--
-- # Empat objek yang hanya DIBACA, dan tidak boleh ditulis dari sini
--
--   POOLDATA.BUSINESS             dimiliki GISFW (`D-03`)
--   POOLDATA.V_LST_DOC_TYPE       dimiliki modul Daftar Tipe Dokumen, MENU_ID 40
--   POOLDATA.V_LST_DET_TYPE_DOC   dimiliki modul Daftar Detail Tipe Dokumen, MENU_ID 41
--   POOLDATA.V_LST_DOC_OBJ        dimiliki modul Daftar Objek Dokumen, MENU_ID 43
--
-- Tidak ada satu pun INSERT, UPDATE, maupun DELETE terhadap keempatnya di berkas ini,
-- dan tidak boleh ada (`P-1`).
--
-- # Tiga kolom yang SENGAJA tidak ditulis
--
-- Keenam kueri unggah dokumen membaca tiga kolom yang procedure lama pun tidak pernah
-- mengisinya:
--
--   FLAGTYPES    `RDB List/BrowseRegisterCvg-SQL.xml`  ORDER BY FLAGTYPES, DETAIL_DOKUMEN
--   CREDENTIAL   `RDB List/BrowseRegisterCvg-SQL.xml`  kolom hasil
--   DURATION     `RDB List/BrowseRegisterCvg-SQL.xml`  kolom hasil
--
-- Ketiganya dibiarkan apa adanya. Menulisinya berarti mengarang nilai untuk kolom yang
-- tidak satu pun layar mengisinya — dan FLAGTYPES menentukan URUTAN dokumen yang dilihat
-- petugas saat meregistrasi klaim, sehingga mengisinya sembarangan akan mengubah tampilan
-- layar yang tidak sedang dimigrasikan. Dicatat sebagai pertanyaan ke DBA.
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding.
--
-- # Kenapa tidak satu pun kueri di sini memakai {ASIS:}
--
-- Kedua kueri layar lama memakainya — `Select_TYPE_DOCUMENT` menyisipkan
-- `{Asis:TempStatusAktif.BranchName}` langsung ke dalam teks SQL, dan keenam kueri unggah
-- menyisipkan daftar jaminan dengan cara yang sama. Itu perangkaian SQL dari nilai, yakni
-- celah yang `11-SECURITY.md` §5 tutup tanpa perkecualian. Penyaringnya di sini parameter
-- binding.


-- name: business_list
--
-- Grid tingkat pertama: lini bisnis yang SUDAH punya aturan dokumen.
--
-- Menggantikan `RDB List/BrowseLSTDetailDocument_sql-SQL.xml`, yang bentuknya:
--
--   select * from (select b.id as BUSINESSID, b.note as DOCUMENT_TYPE_ID
--                    from LST_TYPE_DOC_BUSINESS a, BUSINESS b where a.businessid = b.id
--                  union
--                  select b.id, b.note from ... same ...)z
--
-- UNION-nya adalah UNION dari dua kueri yang SAMA PERSIS — kedua cabangnya memilih
-- `b.id, b.note` dari join yang identik. Jadi ia tidak menggabungkan apa pun; ia bekerja
-- sebagai DISTINCT. Yang ditulis di bawah menyatakan maksud itu terang-terangan.
--
-- Alias `DOCUMENT_TYPE_ID` untuk `b.note` tidak dibawa: isinya nama bisnis, bukan kode
-- tipe dokumen, dan alias semacam itu persis yang `D-19` perintahkan dinamai ulang.
--
-- Urutannya menurut NAMA, bukan ID. Kueri lama tidak mengurutkannya sama sekali, sehingga
-- urutannya ditentukan rencana eksekusi dan dapat berubah sendiri di antara dua kali
-- pemuatan. Grid yang berpindah urutan tanpa sebab terbaca sebagai datanya berubah.
SELECT DISTINCT b.ID,
       b.NOTE
  FROM POOLDATA.LST_TYPE_DOC_BUSINESS a
  JOIN POOLDATA.BUSINESS b ON b.ID = a.BUSINESSID
 ORDER BY b.NOTE

-- name: rule_list_by_business
--
-- Grid tingkat kedua: aturan dokumen milik satu lini bisnis.
--
-- Menggantikan `RDB List/Select_TYPE_DOCUMENT-SQL.xml`. Seluruh alias menyesatkan di sana
-- dibuang (`D-19`) — sebagai catatan bagi yang membandingkan keduanya, pemetaannya:
--
--   a.id              IDPEGA          -> ID
--   b.id              BUSINESSCODE    -> BUSINESSID
--   b.note            BUSINESSNAME    -> BUSINESS_NAME
--   c.id              ACCUMCODE       -> DOCUMENT_TYPE_ID
--   c.type_document   BRANCHNAME      -> DOCUMENT_TYPE_NAME
--   a.object_doc_id   BUSINESSTYPE    -> OBJECT_DOC_ID
--   e.id              CLIENTID        -> DOC_TYPE_DT_ID
--   e.detail_document EDMNO           -> (tidak dipakai; lihat di bawah)
--   a.sts_wajib       EDMTYPE         -> STS_WAJIB
--   a.min_doc         FOLLOWEDPOLICY  -> MIN_DOC
--   a.user_edit       MARKETINGCODE   -> USER_EDIT
--
-- # Dua penyimpangan dari kueri lama, keduanya disengaja
--
-- 1. DETAIL_DOKUMEN dibaca dari BARIS INI, bukan `e.detail_document` dari masternya.
--    Kueri lama membaca nama dari master rinciannya, sementara SELURUH enam kueri unggah
--    dokumen membaca `b.DETAIL_DOKUMEN` dari baris ini. Memakai nama master di layar
--    master akan menampilkan nama yang BERBEDA dari yang kelak dilihat petugas saat
--    mengunggah — dan perbedaan itu tidak akan pernah disadari siapa pun.
--
-- 2. INNER JOIN, MENIRU kueri lama yang merangkai keempat tabel dengan koma lalu
--    menyamakannya di WHERE.
--
--    Versi pertama modul ini memakai LEFT JOIN supaya baris yang masternya sudah hilang
--    tetap terlihat. Itu PENYIMPANGAN yang saya buat sendiri, dan Work Owner menetapkan
--    2026-09-23 layar ini mengikuti Pega apa adanya — maka ia dicabut.
--
--    Akibatnya perlu diketahui, bukan disembunyikan: baris yang DOCUMENT_TYPE_ID,
--    OBJECT_DOC_ID, atau DOC_TYPE_DT_ID-nya tidak lagi ada di masternya TIDAK MUNCUL di
--    layar ini — sementara keenam kueri unggah tetap memakainya, karena mereka membaca
--    `lst_type_doc_business` langsung tanpa join. Aturan yang tak terlihat tetap berlaku
--    pada klaim, dan petugas tidak dapat memperbaikinya dari sini.
--
--    Itu keadaan sistem lama apa adanya (`P-5`). Ia dicatat sebagai pertanyaan ke Work
--    Owner di migrasi 0010 Bagian 1, bukan diperbaiki sepihak dari modul ini.
--
--    OBJECT_DOC_ID boleh NULL (procedure lama mengosongkannya dengan sengaja), sehingga
--    join ke V_LST_DOC_OBJ tetap LEFT — bila ia INNER, setiap baris tanpa objek dokumen
--    ikut hilang, dan itu BUKAN perilaku Pega: kueri lama menyamakan `a.object_doc_id`
--    hanya lewat kolom hasil, tidak lewat WHERE.
SELECT a.ID,
       a.BUSINESSID,
       b.NOTE,
       a.DOCUMENT_TYPE_ID,
       c.TYPE_DOCUMENT,
       a.OBJECT_DOC_ID,
       o.KET_DOC_OBJ,
       a.DOC_TYPE_DT_ID,
       a.DETAIL_DOKUMEN,
       a.STS_WAJIB,
       a.MIN_DOC
  FROM POOLDATA.LST_TYPE_DOC_BUSINESS a
  JOIN POOLDATA.BUSINESS b            ON b.ID = a.BUSINESSID
  JOIN POOLDATA.V_LST_DOC_TYPE c      ON c.ID = a.DOCUMENT_TYPE_ID
  JOIN POOLDATA.V_LST_DET_TYPE_DOC e  ON e.ID = a.DOC_TYPE_DT_ID
  LEFT JOIN POOLDATA.V_LST_DOC_OBJ o  ON o.ID = a.OBJECT_DOC_ID
 WHERE a.BUSINESSID = :1
 ORDER BY a.ID

-- name: rule_get
--
-- Satu baris aturan. Bentuk kolom DAN bentuk join-nya sama persis dengan
-- rule_list_by_business — bila keduanya berbeda, sebuah baris dapat terbuka lewat
-- penyuntingan tetapi tidak pernah tampil di grid, atau sebaliknya.
SELECT a.ID,
       a.BUSINESSID,
       b.NOTE,
       a.DOCUMENT_TYPE_ID,
       c.TYPE_DOCUMENT,
       a.OBJECT_DOC_ID,
       o.KET_DOC_OBJ,
       a.DOC_TYPE_DT_ID,
       a.DETAIL_DOKUMEN,
       a.STS_WAJIB,
       a.MIN_DOC
  FROM POOLDATA.LST_TYPE_DOC_BUSINESS a
  JOIN POOLDATA.BUSINESS b            ON b.ID = a.BUSINESSID
  JOIN POOLDATA.V_LST_DOC_TYPE c      ON c.ID = a.DOCUMENT_TYPE_ID
  JOIN POOLDATA.V_LST_DET_TYPE_DOC e  ON e.ID = a.DOC_TYPE_DT_ID
  LEFT JOIN POOLDATA.V_LST_DOC_OBJ o  ON o.ID = a.OBJECT_DOC_ID
 WHERE a.ID = :1

-- name: rule_business_of
--
-- Lini bisnis pemilik sebuah aturan, dan sekaligus pemastian bahwa barisnya ada.
--
-- Kueri tersendiri, bukan memakai kembali rule_get, dan itu disengaja: rule_get
-- mengembalikan sebelas kolom, sehingga memakainya hanya untuk mengambil satu kolom
-- menuntut sebelas penampung berposisi yang harus ikut berubah setiap kali daftar kolomnya
-- berubah. Satu kolom yang tertukar di sana tidak menimbulkan galat kompilasi — ia hanya
-- menulis BUSINESSID yang salah ke baris jaminan.
SELECT BUSINESSID
  FROM POOLDATA.LST_TYPE_DOC_BUSINESS
 WHERE ID = :1

-- name: coverage_list
--
-- Jaminan milik satu aturan dokumen.
--
-- Disaring ID saja, bukan ID beserta BUSINESSID seperti keenam kueri unggah
-- (`where a.id=b.id and a.businessid=b.businessid`). Keduanya memberi hasil yang sama —
-- ID baris aturan sudah menentukan bisnisnya — dan penyaring tunggal membuat pemanggil
-- tidak perlu membawa serta bisnis yang sudah tersirat.
SELECT COVERAGEID
  FROM POOLDATA.COVERAGE_DOC_BUSINESS
 WHERE ID = :1
   AND NOKLAIM IS NULL
 ORDER BY COVERAGEID

-- name: coverage_exists
--
-- Apakah jaminan ini sudah terpasang pada aturan itu.
--
-- Meniru `PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:53` apa adanya, termasuk keputusannya untuk
-- DIAM ketika jaminannya sudah ada — bukan menolaknya sebagai galat.
SELECT COUNT(*)
  FROM POOLDATA.COVERAGE_DOC_BUSINESS
 WHERE ID = :1
   AND COVERAGEID = :2
   AND NOKLAIM IS NULL

-- name: coverage_insert
--
-- NOKLAIM sengaja TIDAK diisi, dan itu keputusan yang perlu dibaca sebelum diubah.
--
-- Kolom itu ada, dan procedure lama mengisinya — tetapi hanya pada cabang yang menuntut
-- `claimno is not null` (`PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:62-71`), dan nilainya datang
-- dari `InputData.DISC2`. Layar master ini tidak pernah mengisinya: cabang yang dipakainya
-- adalah `tCvg IS NOT NULL and claimno is null` (`:51`).
--
-- Baris ber-NOKLAIM karena itu milik JALUR KLAIM, bukan milik layar ini — ia jaminan yang
-- dipasang pada satu klaim tertentu, bukan aturan yang berlaku umum. Menulisnya dari sini
-- akan mencampur keduanya di satu tabel tanpa cara membedakannya kembali, dan itulah sebab
-- coverage_list di atas menyaring `NOKLAIM IS NULL`.
INSERT INTO POOLDATA.COVERAGE_DOC_BUSINESS (ID, BUSINESSID, COVERAGEID)
VALUES (:1, :2, :3)

-- name: rule_next_sequence
--
-- Nomor urut ID.
--
-- Namanya LST_TYPE_DOC_BUSINESS_SEQ, dibaca dari
-- `Database/PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:22` — bukan diturunkan dari nama tabel
-- seperti terpaksa dilakukan modul Daftar Detail Dokumen Travel.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada `ADR-0005`.
--
-- Pemformatan nomornya menjadi ID dikerjakan di Go, bukan dengan LPAD di sini: LPAD dan
-- TO_CHAR termasuk yang dilarang `09-DATABASE-STRATEGY.md` §4 karena keduanya mengikat
-- kueri pada dialek Oracle.
SELECT POOLDATA.LST_TYPE_DOC_BUSINESS_SEQ.NEXTVAL
  FROM DUAL

-- name: rule_site
--
-- Kode situs, awalan ID. Ia yang membuat tiap entitas menerbitkan awalan ID-nya sendiri,
-- dan karena itulah modul ini portal-aware — persis mekanisme `PEGA_LST_DOC_TYPE.prc:12`
-- pada modul Daftar Tipe Dokumen.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: rule_insert
--
-- Sepuluh kolom, sama persis dengan `PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:25-26`.
--
-- STS_WAJIB diisi TEKS '1' atau '0', bukan angka. Buktinya keenam kueri pembacanya, yang
-- seluruhnya membandingkannya dengan literal berkutip: `WHEN sts_wajib = '0'`.
INSERT INTO POOLDATA.LST_TYPE_DOC_BUSINESS
       (ID, BUSINESSID, DOCUMENT_TYPE_ID, OBJECT_DOC_ID, DOC_TYPE_DT_ID,
        DETAIL_DOKUMEN, STS_WAJIB, MIN_DOC, EDIT_DATE, USER_EDIT)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10)

-- name: rule_update
--
-- Delapan kolom, sama persis dengan `PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:39-41`.
--
-- BUSINESSID TIDAK ada di sini, dan itu bukan kelalaian: procedure lama pun tidak
-- mengubahnya. Sebuah aturan dokumen tidak dapat dipindahkan ke lini bisnis lain — ia
-- disalin, dan itulah guna tombol Copy di layar lama.
--
-- ID hanya dipakai sebagai penyaring. Mengubahnya akan memutus baris jaminan yang
-- merujuknya, karena COVERAGE_DOC_BUSINESS.ID adalah rujukan itu.
UPDATE POOLDATA.LST_TYPE_DOC_BUSINESS
   SET DOCUMENT_TYPE_ID = :1,
       OBJECT_DOC_ID = :2,
       DOC_TYPE_DT_ID = :3,
       DETAIL_DOKUMEN = :4,
       STS_WAJIB = :5,
       MIN_DOC = :6,
       EDIT_DATE = :7,
       USER_EDIT = :8
 WHERE ID = :9

-- name: business_choice_list
--
-- Pilihan lini bisnis pada layar Tambah: SELURUH bisnis, bukan hanya yang sudah punya
-- aturan.
--
-- Bedanya dengan business_list di atas menentukan apakah layar Tambah berguna: bisnis
-- yang belum punya satu baris pun justru yang paling sering dituju, dan memakai kueri yang
-- sama akan membuatnya tidak pernah muncul sebagai pilihan.
--
-- HANYA SELECT. Tabel ini dimiliki GISFW (`D-03`).
SELECT ID,
       NOTE
  FROM POOLDATA.BUSINESS
 ORDER BY NOTE

-- name: document_type_choice_list
--
-- Pilihan isian Tipe Dokumen — tahap klaim.
--
-- Menggantikan autocomplete `BrowseLstDocType_RD`. HANYA SELECT: tabel ini dimiliki modul
-- Daftar Tipe Dokumen, MENU_ID 40 (`P-1`).
--
-- Batas `pyMaxRecords=500` milik Report Definition itu tidak ditiru. Ia bukan aturan
-- bisnis melainkan pemotongan senyap (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2), dan
-- menirunya berarti menyembunyikan tahap yang benar-benar ada dari petugas yang sedang
-- memilihnya.
SELECT ID,
       TYPE_DOCUMENT
  FROM POOLDATA.V_LST_DOC_TYPE
 ORDER BY TYPE_DOCUMENT

-- name: detail_type_doc_choice_list
--
-- Pilihan isian Detail Dokumen.
--
-- Menggantikan autocomplete `BrowseVLstDetTypeDoc_RD`. HANYA SELECT: tabel ini dimiliki
-- modul Daftar Detail Tipe Dokumen, MENU_ID 41 (`P-1`).
--
-- DOC_TYPE_ID ikut dibaca, dan itu BUKAN kolom hiasan: Report Definition lamanya menyaring
-- `.DOC_TYPE_ID = Param.idDocument`, yakni tahap dokumen yang sudah dipilih pada baris
-- yang sama. Tanpa kolom ini, layar tidak dapat menyempitkan daftarnya dan akan menawarkan
-- rincian milik tahap lain sebagai pilihan yang sah.
--
-- Penyaringnya TIDAK dipasang di sini. Daftar dikirim utuh sekali, lalu disaring layar —
-- alasannya sama dengan daftar jaminan pada modul Daftar Detail Dokumen Travel, dan
-- tertulis lengkap di komentar Reference.ParentID pada lapisan domain.
SELECT ID,
       DETAIL_DOCUMENT,
       DOC_TYPE_ID
  FROM POOLDATA.V_LST_DET_TYPE_DOC
 ORDER BY DETAIL_DOCUMENT

-- name: object_doc_choice_list
--
-- Pilihan isian Object Dokumen.
--
-- Menggantikan autocomplete `BrowseVLstDocObj_RD`, yang mengambil ID, OLD_ID, dan
-- KET_DOC_OBJ. OLD_ID tidak dibawa: ia jejak sejarah yang tidak pernah ditampilkan
-- maupun disimpan layar ini.
SELECT ID,
       KET_DOC_OBJ
  FROM POOLDATA.V_LST_DOC_OBJ
 ORDER BY KET_DOC_OBJ

-- name: rule_check_table
--
-- Memastikan tabel modul ini ada dan kolomnya dapat dibaca akun aplikasi, tanpa mengambil
-- satu baris pun. Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak
-- mirip tetapi perbaikannya berbeda jauh: objeknya tidak ada di basis data entitas itu,
-- versus akun aplikasi tidak punya hak baca atasnya.
--
-- Yang TIDAK diperiksa: urutan penerbit ID. Memeriksanya berarti MENGHABISKAN satu nomor
-- — efek samping yang tidak pantas dimiliki mode periksa.
SELECT ID,
       BUSINESSID,
       DOCUMENT_TYPE_ID,
       OBJECT_DOC_ID,
       DOC_TYPE_DT_ID,
       DETAIL_DOKUMEN,
       STS_WAJIB,
       MIN_DOC,
       EDIT_DATE,
       USER_EDIT
  FROM POOLDATA.LST_TYPE_DOC_BUSINESS
 WHERE 1 = 0

-- name: coverage_check_table
SELECT ID,
       BUSINESSID,
       COVERAGEID,
       NOKLAIM
  FROM POOLDATA.COVERAGE_DOC_BUSINESS
 WHERE 1 = 0
