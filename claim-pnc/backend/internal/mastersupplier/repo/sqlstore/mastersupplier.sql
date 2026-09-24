-- Kueri modul Master Supplier.
--
-- Dua tabel DITULIS aplikasi ini — M_SUPPLIER dan pooldata.proteksi_klaimmbu — dan lima
-- objek lain hanya DIBACA: M_BRANCH, CITY, COUNTRY, GENERAL.LST_BANK_GROUP, ditambah
-- POOLDATA.M_SITE_DATABASE dan sequence SUPPLIER_SEQ untuk penomoran. Seluruhnya milik
-- sistem lain (ADR-0004, penulis tunggal per tabel); tidak ada satu pun pernyataan tulis
-- terhadapnya di berkas ini.
--
-- Kewenangan menulis M_SUPPLIER berpindah dari Pega ke Go saat modulnya lulus gerbang 2.
-- Selama Pega masih penulisnya, layar ini harus dijalankan dalam modus baca saja di
-- produksi.
--
-- Empat aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (D-20).
--   4. Tanpa pemanggilan stored procedure (D-02).
--
-- TIDAK ADA DELETE di berkas ini. Sistem lama pun tidak punya satu pun terhadap kedua
-- tabel ini, dan D-66 melarang penghapusan fisik data bernilai bisnis. Supplier yang
-- tidak lagi dipakai ditandai lewat STS_AKTIF, bukan dibuang.
--
--
-- ============================================================================
-- KENAPA MEMBACA DAN MENULIS DOKUMEN JSON, PADAHAL MODUL MASTER LAIN TIDAK
-- ============================================================================
--
-- Karena tabelnya memang hanya itu. `Database/PEGA_M_SUPPLIER.prc:24,36` membuktikan
-- M_SUPPLIER hanya punya `ID` dan `JSONDATA`:
--
--   INSERT INTO M_SUPPLIER(ID,JSONDATA) VALUES(id_supplier_ins, replace(DataPega,…));
--   UPDATE M_SUPPLIER SET JSONDATA = DataPega WHERE ID = IDPega;
--
-- ditambah `OLDID` yang terbaca dari kueri bacanya. Tidak ada kembaran berkolom bernama
-- seperti POOLDATA.BENGKEL_HE, sehingga pilihan yang diambil Master Bengkel — menulis
-- kolom dan berhenti menulis JSON — tidak tersedia di sini.
--
-- Master Bengkel MENOLAK menulis dokumen JSON karena nama kuncinya tidak diketahui:
-- `@GCNM.GetPageJSONString()` hanya ada tanda tangannya di export. Alasan itu TIDAK
-- berlaku di sini, dan itulah yang membedakan keputusannya:
-- `RDB List/GetDataEditMasterSupller-SQL.xml:90-118` membaca setiap kunci satu per satu,
--
--   A.JSONDATA.NAMA AS "NAMA", A.JSONDATA.ALAMAT AS "ALAMAT", …
--
-- sehingga kedua puluh lima kuncinya terbaca lengkap. Setiap kunci yang ditulis berkas
-- ini diketahui benar, dan apa yang ditulis dapat dibaca kembali layar Pega yang sama.
--
-- ## Notasi titik Oracle diganti JSON_VALUE, dan itu bukan sekadar selera
--
-- Kueri lama memakai `A.JSONDATA.NAMA` — notasi khas Oracle yang tidak ada padanannya di
-- PostgreSQL. `JSON_VALUE(JSONDATA, '$.NAMA')` mengembalikan nilai yang sama persis dan
-- berlaku di Oracle 12c+ maupun PostgreSQL 17+, yang justru menjadi alasan D-24
-- mewajibkan versi 17 (222 pemanggilan SQL/JSON di seluruh sistem lama).
--
-- ## Yang WAJIB dipastikan DBA sebelum modul ini menulis di produksi
--
-- Lebar kolom JSONDATA, dan apakah ia CLOB atau VARCHAR2. DDL-nya tidak ada di export
-- (R-08). Bedanya nyata: seluruh dua puluh delapan nilai masuk ke SATU kolom, sehingga
-- kolom yang terlalu sempit menolak BARIS UTUH — bukan isian yang kepanjangan. Batas
-- panjang isian di paket domain memperkecil kemungkinannya, tetapi tidak menghapusnya.
--
-- `claimpnc -periksa` melaporkan keduanya — lihat checkSupplier pada cmd/claimpnc.
--
--
-- CATATAN NAMA SKEMA. `M_SUPPLIER`, `M_BRANCH`, `CITY`, dan `COUNTRY` disebut TANPA
-- skema, persis seperti seluruh kueri lama yang membacanya — termasuk procedure
-- PEGA_M_SUPPLIER sendiri, yang berada di POOLDATA dan menyebut M_SUPPLIER tanpa skema.
-- Yang terbaca karena itu mengikuti skema bawaan akun koneksi. Melengkapinya dengan
-- skema berarti menebak, dan tebakan yang salah membuat layar kosong tanpa galat yang
-- menjelaskan sebabnya.
--
-- CATATAN TRIM pada setiap penyaring kunci. Tipe kolom ID belum diketahui (R-08). Bila ia
-- CHAR berlebar tetap, nilainya dipadatkan spasi tanpa tanda apa pun; Oracle membandingkan
-- CHAR dengan CHAR secara blank-padded, sedangkan parameter binding bertipe VARCHAR2 dan
-- perbandingannya non-padded — "ABC " tidak sama dengan "ABC", dan barisnya tidak ketemu.
-- TRIM benar untuk kedua kemungkinan tipe. Biayanya index atas kolom itu tidak terpakai;
-- dapat diterima pada tabel master berbaris sedikit, dan TIDAK boleh ditiru pada tabel
-- besar.


-- name: supplier_list
--
-- TIDAK ADA ASAL DI EXPORT, dan itu harus dinyatakan terang-terangan.
--
-- Grid `Section/InboxMasterSupplier-Section.xml` membaca page list
-- `ListMasterSupllier.pxResults`, dan **tidak ada satu pun rule di seluruh export yang
-- mengisinya** — bukan report definition, bukan RDB List, bukan activity (R-16). Yang
-- terbaca hanyalah KOLOM yang ditampilkannya: ID, NAMA, ALAMAT, TELP, JENIS SUPPLIER,
-- STATUS REKANAN, STATUS AKTIF, dan POSISI.
--
-- Kueri ini karena itu disusun dari kolom-kolom itu ditambah seluruh kunci yang dibutuhkan
-- form penyuntingan, sehingga membuka sebuah baris tidak menuntut perjalanan kedua.
--
-- Kolom POSISI TIDAK ada di sini. Ia milik `pooldata.proteksi_klaimmbu`, dan tidak ada
-- satu pun rule di export yang membacanya — lihat catatan pada approval_insert. Menebak
-- cara menggabungkannya berarti menampilkan angka yang tidak diketahui artinya di kolom
-- yang dibaca petugas sebagai posisi persetujuan.
--
-- Kedua puluh sembilan kolomnya disebut pada urutan yang SAMA dengan supplier_get dan
-- supplier_find_by_name — satu fungsi scanRow membaca ketiganya berdasarkan POSISI, dan
-- satu kolom yang bergeser akan menaruh nomor rekening ke kolom alamat tanpa satu pun
-- galat. `TestReaderQueriesShareColumnOrder` yang menjaganya.
--
-- ORDER BY atas nama. Tanpa kueri lamanya tidak ada urutan yang dapat ditiru; yang dapat
-- dilakukan adalah memilih urutan yang tetap, alih-alih urutan apa pun yang kebetulan
-- dikembalikan basis data.
SELECT ID,
       OLDID,
       JSON_VALUE(JSONDATA, '$.NAMA'),
       JSON_VALUE(JSONDATA, '$.ALAMAT'),
       JSON_VALUE(JSONDATA, '$.KOTA'),
       JSON_VALUE(JSONDATA, '$.NAMA_CABANG'),
       JSON_VALUE(JSONDATA, '$.KODE_POS'),
       JSON_VALUE(JSONDATA, '$.NEGARA'),
       JSON_VALUE(JSONDATA, '$.TELEPON'),
       JSON_VALUE(JSONDATA, '$.FAX'),
       JSON_VALUE(JSONDATA, '$.EMAIL'),
       JSON_VALUE(JSONDATA, '$.NPWP'),
       JSON_VALUE(JSONDATA, '$.CONTACT_PERSON'),
       JSON_VALUE(JSONDATA, '$.STS_REKANAN'),
       JSON_VALUE(JSONDATA, '$.JENIS_STATUS'),
       JSON_VALUE(JSONDATA, '$.SUPPLIER_HE'),
       JSON_VALUE(JSONDATA, '$.TOP'),
       JSON_VALUE(JSONDATA, '$.TOD'),
       JSON_VALUE(JSONDATA, '$.KETERANGAN'),
       JSON_VALUE(JSONDATA, '$.BANK'),
       JSON_VALUE(JSONDATA, '$.ACCOUNT_NO'),
       JSON_VALUE(JSONDATA, '$.ACCOUNT_NAME'),
       JSON_VALUE(JSONDATA, '$.BANK_BRANCH'),
       JSON_VALUE(JSONDATA, '$.JENIS_SUPPLIER'),
       JSON_VALUE(JSONDATA, '$.STS_AKTIF_PROMLIST'),
       JSON_VALUE(JSONDATA, '$.STS_AKTIF'),
       JSON_VALUE(JSONDATA, '$.STS_AUTOPAYMENT'),
       JSON_VALUE(JSONDATA, '$.USERKLAIMID'),
       JSON_VALUE(JSONDATA, '$.TGL_INSERT')
  FROM M_SUPPLIER
 ORDER BY JSON_VALUE(JSONDATA, '$.NAMA')

-- name: supplier_list_search
--
-- Sama dengan supplier_list, ditambah penyaring kata kunci atas nama, kota, dan contact
-- person — ketiga kolom yang paling mungkin diketik petugas yang mencari satu supplier.
--
-- KUERI TERSENDIRI, bukan satu kueri yang klausanya ditempel. Sistem lama menempuh cara
-- yang kedua di banyak tempat lewat pola `{ASIS:...}` — nilai dirangkai langsung ke teks
-- SQL — dan `08-TECHNICAL-STRATEGY.md` §4.3 melarangnya tanpa perkecualian.
--
-- UPPER dipasang pada NILAINYA juga, bukan hanya pada kata kuncinya: nama supplier
-- tersimpan dengan besar-kecil huruf apa adanya, dan pencarian yang hanya meng-uppercase
-- kata kunci tidak akan pernah menemukan baris yang namanya huruf kecil.
--
-- `ESCAPE '\'` disebut eksplisit karena Oracle TIDAK punya karakter pelolos bawaan pada
-- LIKE. Tanpa itu, tanda persen yang diketik pengguna tetap berlaku sebagai wildcard meski
-- sudah diloloskan di Go — dan hasil pencariannya tidak dapat dijelaskan kepada yang
-- mengetiknya. PostgreSQL memakai backslash sebagai bawaan; menyebutkannya eksplisit
-- membuat kedua basis data berperilaku sama (D-20).
SELECT ID,
       OLDID,
       JSON_VALUE(JSONDATA, '$.NAMA'),
       JSON_VALUE(JSONDATA, '$.ALAMAT'),
       JSON_VALUE(JSONDATA, '$.KOTA'),
       JSON_VALUE(JSONDATA, '$.NAMA_CABANG'),
       JSON_VALUE(JSONDATA, '$.KODE_POS'),
       JSON_VALUE(JSONDATA, '$.NEGARA'),
       JSON_VALUE(JSONDATA, '$.TELEPON'),
       JSON_VALUE(JSONDATA, '$.FAX'),
       JSON_VALUE(JSONDATA, '$.EMAIL'),
       JSON_VALUE(JSONDATA, '$.NPWP'),
       JSON_VALUE(JSONDATA, '$.CONTACT_PERSON'),
       JSON_VALUE(JSONDATA, '$.STS_REKANAN'),
       JSON_VALUE(JSONDATA, '$.JENIS_STATUS'),
       JSON_VALUE(JSONDATA, '$.SUPPLIER_HE'),
       JSON_VALUE(JSONDATA, '$.TOP'),
       JSON_VALUE(JSONDATA, '$.TOD'),
       JSON_VALUE(JSONDATA, '$.KETERANGAN'),
       JSON_VALUE(JSONDATA, '$.BANK'),
       JSON_VALUE(JSONDATA, '$.ACCOUNT_NO'),
       JSON_VALUE(JSONDATA, '$.ACCOUNT_NAME'),
       JSON_VALUE(JSONDATA, '$.BANK_BRANCH'),
       JSON_VALUE(JSONDATA, '$.JENIS_SUPPLIER'),
       JSON_VALUE(JSONDATA, '$.STS_AKTIF_PROMLIST'),
       JSON_VALUE(JSONDATA, '$.STS_AKTIF'),
       JSON_VALUE(JSONDATA, '$.STS_AUTOPAYMENT'),
       JSON_VALUE(JSONDATA, '$.USERKLAIMID'),
       JSON_VALUE(JSONDATA, '$.TGL_INSERT')
  FROM M_SUPPLIER
 WHERE UPPER(JSON_VALUE(JSONDATA, '$.NAMA')) LIKE :1 ESCAPE '\'
    OR UPPER(JSON_VALUE(JSONDATA, '$.KOTA')) LIKE :1 ESCAPE '\'
    OR UPPER(JSON_VALUE(JSONDATA, '$.CONTACT_PERSON')) LIKE :1 ESCAPE '\'
 ORDER BY JSON_VALUE(JSONDATA, '$.NAMA')

-- name: supplier_get
--
-- Satu baris, untuk dimuat ke form penyuntingan. Urutan kolomnya sama dengan
-- supplier_list.
--
-- Asal: RDB List/GetDataEditMasterSupller-SQL.xml, yang menyaring `A.ID = {ParamSP.ID}`.
-- TRIM ditambahkan; lihat catatan di kepala berkas.
SELECT ID,
       OLDID,
       JSON_VALUE(JSONDATA, '$.NAMA'),
       JSON_VALUE(JSONDATA, '$.ALAMAT'),
       JSON_VALUE(JSONDATA, '$.KOTA'),
       JSON_VALUE(JSONDATA, '$.NAMA_CABANG'),
       JSON_VALUE(JSONDATA, '$.KODE_POS'),
       JSON_VALUE(JSONDATA, '$.NEGARA'),
       JSON_VALUE(JSONDATA, '$.TELEPON'),
       JSON_VALUE(JSONDATA, '$.FAX'),
       JSON_VALUE(JSONDATA, '$.EMAIL'),
       JSON_VALUE(JSONDATA, '$.NPWP'),
       JSON_VALUE(JSONDATA, '$.CONTACT_PERSON'),
       JSON_VALUE(JSONDATA, '$.STS_REKANAN'),
       JSON_VALUE(JSONDATA, '$.JENIS_STATUS'),
       JSON_VALUE(JSONDATA, '$.SUPPLIER_HE'),
       JSON_VALUE(JSONDATA, '$.TOP'),
       JSON_VALUE(JSONDATA, '$.TOD'),
       JSON_VALUE(JSONDATA, '$.KETERANGAN'),
       JSON_VALUE(JSONDATA, '$.BANK'),
       JSON_VALUE(JSONDATA, '$.ACCOUNT_NO'),
       JSON_VALUE(JSONDATA, '$.ACCOUNT_NAME'),
       JSON_VALUE(JSONDATA, '$.BANK_BRANCH'),
       JSON_VALUE(JSONDATA, '$.JENIS_SUPPLIER'),
       JSON_VALUE(JSONDATA, '$.STS_AKTIF_PROMLIST'),
       JSON_VALUE(JSONDATA, '$.STS_AKTIF'),
       JSON_VALUE(JSONDATA, '$.STS_AUTOPAYMENT'),
       JSON_VALUE(JSONDATA, '$.USERKLAIMID'),
       JSON_VALUE(JSONDATA, '$.TGL_INSERT')
  FROM M_SUPPLIER
 WHERE TRIM(ID) = :1

-- name: supplier_find_by_name
--
-- TIDAK ADA ASAL DI EXPORT. Berbeda dari Master Bengkel yang punya
-- `RDB List/ValidationMasterBengkel-SQL.xml`, tidak ada satu pun rule yang memeriksa nama
-- supplier ganda.
--
-- Ia ditambahkan karena layar sendiri memperlakukan nama sebagai kunci alami: sekali
-- tersimpan, isiannya menjadi read-only
-- (`pyReadOnlyCondition: MasterSupplier.ID != ''`) dan tidak dapat diperbaiki lagi. Dua
-- supplier bernama sama karena itu akan hidup selamanya tanpa satu pun cara membedakannya
-- dari layar.
--
-- Perlakuan `UPPER(TRIM(...))` ditiru dari kueri padanannya di Master Bengkel — itu yang
-- membuat "Supplier Jaya" dan "SUPPLIER JAYA " dikenali sebagai nama yang sama.
--
-- Seluruh baris dibaca dengan urutan yang sama seperti supplier_list, supaya pemanggil
-- dapat menyebutkan ID DAN nama supplier yang sudah memakai nama itu di dalam pesan
-- galatnya — "sudah dipakai" tanpa menyebut yang mana memaksa pengguna mencarinya sendiri.
SELECT ID,
       OLDID,
       JSON_VALUE(JSONDATA, '$.NAMA'),
       JSON_VALUE(JSONDATA, '$.ALAMAT'),
       JSON_VALUE(JSONDATA, '$.KOTA'),
       JSON_VALUE(JSONDATA, '$.NAMA_CABANG'),
       JSON_VALUE(JSONDATA, '$.KODE_POS'),
       JSON_VALUE(JSONDATA, '$.NEGARA'),
       JSON_VALUE(JSONDATA, '$.TELEPON'),
       JSON_VALUE(JSONDATA, '$.FAX'),
       JSON_VALUE(JSONDATA, '$.EMAIL'),
       JSON_VALUE(JSONDATA, '$.NPWP'),
       JSON_VALUE(JSONDATA, '$.CONTACT_PERSON'),
       JSON_VALUE(JSONDATA, '$.STS_REKANAN'),
       JSON_VALUE(JSONDATA, '$.JENIS_STATUS'),
       JSON_VALUE(JSONDATA, '$.SUPPLIER_HE'),
       JSON_VALUE(JSONDATA, '$.TOP'),
       JSON_VALUE(JSONDATA, '$.TOD'),
       JSON_VALUE(JSONDATA, '$.KETERANGAN'),
       JSON_VALUE(JSONDATA, '$.BANK'),
       JSON_VALUE(JSONDATA, '$.ACCOUNT_NO'),
       JSON_VALUE(JSONDATA, '$.ACCOUNT_NAME'),
       JSON_VALUE(JSONDATA, '$.BANK_BRANCH'),
       JSON_VALUE(JSONDATA, '$.JENIS_SUPPLIER'),
       JSON_VALUE(JSONDATA, '$.STS_AKTIF_PROMLIST'),
       JSON_VALUE(JSONDATA, '$.STS_AKTIF'),
       JSON_VALUE(JSONDATA, '$.STS_AUTOPAYMENT'),
       JSON_VALUE(JSONDATA, '$.USERKLAIMID'),
       JSON_VALUE(JSONDATA, '$.TGL_INSERT')
  FROM M_SUPPLIER
 WHERE UPPER(TRIM(JSON_VALUE(JSONDATA, '$.NAMA'))) = :1
 FETCH NEXT 1 ROW ONLY

-- name: supplier_lock_by_name
--
-- Mengunci baris yang namanya sama, dipakai sebelum menyisipkan.
--
-- FOR UPDATE di sini MEMPERSEMPIT lubang balapan antara pemeriksaan dan penyisipan, tetapi
-- TIDAK MENUTUPNYA — dan itu harus dinyatakan terang-terangan. Mengunci baris yang belum
-- ada tidak mungkin: bila dua penambahan sama-sama menemukan nol baris, keduanya lolos.
--
-- Penutup yang sebenarnya adalah constraint unik atas namanya. Di sini ia bahkan lebih
-- jauh daripada di modul master lain: nama supplier tersimpan DI DALAM dokumen JSON,
-- sehingga constraint unik atasnya menuntut index berbasis fungsi lebih dulu — dan
-- keduanya menempuh DDL (R-08) beserta prosedur perubahan skema (D-63).
SELECT ID
  FROM M_SUPPLIER
 WHERE UPPER(TRIM(JSON_VALUE(JSONDATA, '$.NAMA'))) = :1
 FOR UPDATE

-- name: supplier_read_document
--
-- Membaca dokumen JSON apa adanya, untuk digabung sebelum disimpan kembali.
--
-- Ia yang membuat kunci yang TIDAK dikenal modul ini ikut bertahan. `PEGA_M_SUPPLIER.prc:36`
-- mengganti seluruh dokumen (`SET JSONDATA = DataPega`), dan yang dikirim Pega hanyalah
-- kunci yang ada di halaman klipboardnya — sehingga kunci yang tidak dibaca
-- `GetDataEditMasterSupller` LENYAP pada setiap penyimpanan.
--
-- Perlakuannya sama dengan DOKUMENID pada Master Bengkel: jalur yang menghapus datanya
-- sendiri tidak ikut dibawa.
--
-- FOR UPDATE dipasang karena hasilnya dipakai menyusun nilai yang langsung ditulis balik.
-- Tanpa itu, dua penyimpanan atas baris yang sama akan sama-sama membaca dokumen lama dan
-- yang terakhir menimpa perubahan yang pertama — tanpa satu pun tanda.
SELECT JSONDATA
  FROM M_SUPPLIER
 WHERE TRIM(ID) = :1
 FOR UPDATE

-- name: supplier_insert
--
-- Dua kolom saja, karena tabelnya memang hanya itu.
--
-- OLDID TIDAK disebut. Tidak ada satu pun rule di export yang mengisinya — `PEGA_M_SUPPLIER.prc:24`
-- pun hanya menulis ID dan JSONDATA — sehingga menuliskan nilai apa pun ke sana berarti
-- menebak apa gunanya.
--
-- ID TIDAK diterbitkan basis data. Ia dibentuk di Go dari kode situs dan sequence, persis
-- seperti `Database/PEGA_M_SUPPLIER.prc:21` — lihat supplier_site dan
-- supplier_next_sequence.
--
-- Penggantian teks `'UnknownID'` pada procedure lama tidak ditiru: ID yang sebenarnya
-- sudah ditulis ke dalam dokumen sebelum sampai ke sini. Lihat usecase.Service.Create.
INSERT INTO M_SUPPLIER
       (ID, JSONDATA)
VALUES (:1, :2)

-- name: supplier_update
--
-- Padanan `Database/PEGA_M_SUPPLIER.prc:36` apa adanya. ID tidak disebut di SET karena ia
-- kunci baris, bukan isian — memindahkan sebuah baris ke ID lain berarti menambah baris
-- baru, bukan mengubah yang ada.
--
-- Dokumen yang ditulis sudah merupakan hasil penggabungan; lihat supplier_read_document.
UPDATE M_SUPPLIER
   SET JSONDATA = :1
 WHERE TRIM(ID) = :2

-- name: supplier_code_distinct
--
-- Sandi yang BENAR-BENAR dipakai baris yang ada, untuk kelima dropdown bersandi.
--
-- Ia ada karena daftar pilihan kelimanya tidak diketahui: isiannya `pxDropdown` bersumber
-- `associated` di Pega, artinya daftarnya ada di rule **Field Value** — dan tidak satu pun
-- rule Field Value ikut di export (R-16). Lihat mastersupplier.CodeOption untuk ketiga
-- jalan yang dipertimbangkan dan alasan yang ini dipilih.
--
-- SATU kueri untuk lima daftar, bukan lima. Kelimanya dibaca dari tabel yang sama, dan
-- memecahnya berarti lima kali memindai tabel yang sama untuk mengisi satu form.
--
-- Penyaring `IS NOT NULL` DAN `<> ''` keduanya dipasang, dan itu bukan pengulangan: Oracle
-- memperlakukan teks kosong sebagai NULL sehingga penyaring pertama yang menangkapnya,
-- sedangkan PostgreSQL membedakan keduanya sehingga penyaring kedua yang menangkapnya.
-- Satu saja akan meloloskan pilihan kosong di salah satu basis data (D-20).
SELECT DISTINCT 'STS_REKANAN', TRIM(JSON_VALUE(JSONDATA, '$.STS_REKANAN'))
  FROM M_SUPPLIER
 WHERE TRIM(JSON_VALUE(JSONDATA, '$.STS_REKANAN')) IS NOT NULL
   AND TRIM(JSON_VALUE(JSONDATA, '$.STS_REKANAN')) <> ''
UNION ALL
SELECT DISTINCT 'JENIS_STATUS', TRIM(JSON_VALUE(JSONDATA, '$.JENIS_STATUS'))
  FROM M_SUPPLIER
 WHERE TRIM(JSON_VALUE(JSONDATA, '$.JENIS_STATUS')) IS NOT NULL
   AND TRIM(JSON_VALUE(JSONDATA, '$.JENIS_STATUS')) <> ''
UNION ALL
SELECT DISTINCT 'JENIS_SUPPLIER', TRIM(JSON_VALUE(JSONDATA, '$.JENIS_SUPPLIER'))
  FROM M_SUPPLIER
 WHERE TRIM(JSON_VALUE(JSONDATA, '$.JENIS_SUPPLIER')) IS NOT NULL
   AND TRIM(JSON_VALUE(JSONDATA, '$.JENIS_SUPPLIER')) <> ''
UNION ALL
SELECT DISTINCT 'STS_AKTIF_PROMLIST', TRIM(JSON_VALUE(JSONDATA, '$.STS_AKTIF_PROMLIST'))
  FROM M_SUPPLIER
 WHERE TRIM(JSON_VALUE(JSONDATA, '$.STS_AKTIF_PROMLIST')) IS NOT NULL
   AND TRIM(JSON_VALUE(JSONDATA, '$.STS_AKTIF_PROMLIST')) <> ''
UNION ALL
SELECT DISTINCT 'STS_AUTOPAYMENT', TRIM(JSON_VALUE(JSONDATA, '$.STS_AUTOPAYMENT'))
  FROM M_SUPPLIER
 WHERE TRIM(JSON_VALUE(JSONDATA, '$.STS_AUTOPAYMENT')) IS NOT NULL
   AND TRIM(JSON_VALUE(JSONDATA, '$.STS_AUTOPAYMENT')) <> ''

-- name: supplier_count_all
--
-- Jumlah seluruh baris. Dipakai mode periksa, bukan layar — pencacah "Total Data :" pada
-- `Section/DataCountMasterSupllier-Section.xml` menghitung baris yang sudah terkirim,
-- bukan baris di basis data.
SELECT COUNT(ID)
  FROM M_SUPPLIER

-- name: supplier_check_table
--
-- Memastikan tabel ada dan dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
SELECT ID
  FROM M_SUPPLIER
 WHERE 1 = 0

-- name: supplier_check_document
--
-- Memastikan kolom JSONDATA benar-benar berisi JSON yang dapat dibaca JSON_VALUE.
--
-- Ia TIDAK dipakai jalur mana pun selain mode periksa, dan itu memang tujuannya: bila
-- kolomnya ternyata bukan JSON yang sah — atau kunci NAMA tidak ada di dalamnya — seluruh
-- layar akan menampilkan baris berisi kolom kosong TANPA satu pun galat, karena JSON_VALUE
-- menjawab NULL alih-alih gagal.
SELECT COUNT(ID)
  FROM M_SUPPLIER
 WHERE JSON_VALUE(JSONDATA, '$.NAMA') IS NOT NULL

-- name: supplier_site
--
-- Kode situs, bagian pertama dari setiap ID supplier. Meniru
-- `Database/PEGA_M_SUPPLIER.prc:12` persis, termasuk pembandingnya yang berupa TEKS '1'
-- dan bukan angka.
--
-- Tabel yang sama dibaca Master Bengkel dan Master Status Klaim lewat kueri
-- masing-masing. Ketiganya sengaja tidak dipakai bersama: modul tidak saling mengimpor,
-- dan kueri bersama akan membuat perubahan di satu modul menyeret modul lain.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: supplier_next_sequence
--
-- Urutan yang SAMA dengan yang dipakai procedure lama, supaya ID yang diterbitkan aplikasi
-- ini melanjutkan deret yang sudah ada dan tidak pernah bertabrakan dengan ID yang pernah
-- diterbitkan Pega.
--
-- `Database/PEGA_M_SUPPLIER.prc:21` menyebutnya tanpa skema (`supplier_seq`), sehingga yang
-- terpakai bergantung pada skema bawaan akun koneksi. Ia dibiarkan tanpa skema di sini
-- juga, mengikuti catatan nama skema di kepala berkas.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada ADR-0005, satu-satunya tempat lain
-- yang dibenarkan memuat percabangan dialek.
SELECT SUPPLIER_SEQ.NEXTVAL
  FROM DUAL

-- name: supplier_branch_list
--
-- Asal: Report Definition/BrowseBranchForGKM_RD-RD.xml atas kelas ASM-FW-GISFW-Int-BRANCH,
-- yang menyuapi dropdown "Cabang".
--
-- KELASNYA MEMETAKAN KE TABEL JSON. Itu terbaca dari kueri lain yang membaca nama cabang
-- dari ID-nya, misalnya `RDB List/GetBranchName-SQL.xml`:
--
--   select b.JSONDATA.Name as LCA_NOTE from M_BRANCH b where b.id = {…}
--
-- dan dipakai dengan bentuk yang sama di tujuh rule `SearchKlaimBy*_RDB`. Notasi titiknya
-- diganti JSON_VALUE demi portabilitas; nilainya sama persis.
--
-- Perhatikan besar-kecil huruf kuncinya: `$.Name`, bukan `$.NAMA`. Dokumen M_BRANCH
-- memakai penamaan yang berbeda dari M_SUPPLIER, dan jalur JSON bersifat case-sensitive —
-- salah satu huruf saja membuat dropdown Cabang kosong tanpa galat.
--
-- Baris tanpa nama dibuang: ia tidak dapat dipilih maupun ditampilkan, dan kehadirannya
-- hanya menambah baris kosong di puncak daftar.
SELECT ID,
       JSON_VALUE(JSONDATA, '$.Name')
  FROM M_BRANCH
 WHERE JSON_VALUE(JSONDATA, '$.Name') IS NOT NULL
 ORDER BY JSON_VALUE(JSONDATA, '$.Name')

-- name: supplier_city_search
--
-- Asal: Report Definition/BrowseCity_RD-RD.xml atas kelas ASM-FW-GISFW-Int-CITY, yang
-- menyuapi dropdown "Kota".
--
-- PERHATIKAN NAMA KOLOMNYA: nama kota tersimpan di kolom bernama NOTE, bukan NAME maupun
-- CITYNAME. Itu terbaca dari `RDB List/BrowseRW_SQL-SQL.xml`:
-- `(select distinct a.note from city a where a.id = b.cityid)`.
--
-- Pencariannya juga menerima ID persis, supaya baris lama yang menyimpan kode alih-alih
-- nama tetap dapat ditemukan.
--
-- FETCH NEXT membatasi hasil; report definition lamanya tidak membatasinya sama sekali
-- (`pyMaxRecords=0`) dan memuat seluruh tabel ke klipboard.
SELECT ID,
       NOTE
  FROM CITY
 WHERE UPPER(NOTE) LIKE :1 ESCAPE '\'
    OR UPPER(TRIM(ID)) = :2
 ORDER BY NOTE
 FETCH NEXT 50 ROWS ONLY

-- name: supplier_country_list
--
-- Asal: Report Definition/BrowseCountry_RD-RD.xml atas kelas ASM-FW-GISFW-Int-COUNTRY,
-- yang menyuapi isian "Negara".
--
-- Tabelnya terbaca dari `Database/UPDATEREAS.prc:41,55`:
--
--   SELECT ID INTO NEGARA_ID FROM COUNTRY WHERE COUNTRY = tCOUNTRY;
--
-- TABEL dan KOLOM-nya sama-sama bernama COUNTRY. Alias `c` dipasang supaya kolomnya
-- disebut tanpa ambiguitas — tanpa itu, `ORDER BY COUNTRY` dapat terbaca sebagai nama
-- tabel di sebagian basis data.
--
-- Report definition lamanya membatasi diri di 500 baris; batas itu tidak dipasang karena
-- daftar negara memang berjumlah di bawah itu, dan memotongnya berarti sebagian negara
-- tidak pernah dapat dipilih.
SELECT c.ID,
       c.COUNTRY
  FROM COUNTRY c
 WHERE c.COUNTRY IS NOT NULL
 ORDER BY c.COUNTRY

-- name: supplier_bank_list
--
-- Asal: Report Definition/BrowseBankGroup-RD.xml atas GENERAL.LST_BANK_GROUP.
--
-- Di Pega dropdown ini disuapi page list `DataBank.pxResults` yang diisi
-- `Activity/GetDataMasterBank-Act.xml` — activity yang isinya ternyata pemuat Master
-- Rekening lengkap dengan penyaring persetujuan dan komite, bukan pemuat daftar bank. Yang
-- dibutuhkan isian ini hanyalah nama banknya, sehingga yang dipakai adalah sumber bank
-- yang sebenarnya, sama dengan tiga modul master lain.
--
-- Tabel yang sama dibaca Master Rekening, Master Auto Claim, dan Master Bengkel lewat
-- kueri masing-masing. Keempatnya sengaja tidak dipakai bersama; yang dipakai bersama
-- adalah tabelnya, bukan kodenya.
SELECT LBG_ID,
       BANK_GROUP
  FROM GENERAL.LST_BANK_GROUP
 ORDER BY BANK_GROUP

-- name: approval_insert
--
-- Asal: RDB List/InsertProteksiKlaimMBU_SQL-SQL.xml, ditiru kolom per kolom.
--
-- DUA PERUBAHAN terhadap kueri lama, keduanya tidak mengubah baris yang dihasilkan:
--
--  1. `SYSDATE` menjadi parameter. Bukan demi portabilitas saja (D-20 melarang SYSDATE):
--     waktunya datang dari seam Clock, sehingga jejak waktu yang sama dipakai dokumen
--     supplier dan baris permintaannya — dan keduanya dapat diuji deterministik.
--  2. `'18'` tetap literal. Ia yang membedakan permintaan supplier dari jenis permintaan
--     lain di tabel yang sama, dan ia bukan masukan pengguna.
--
-- PROTEKSI_ID TIDAK dapat ditiru. Di Pega ia `ChildPageProtection.pyID`, yaitu ID work
-- object case `ASM-FW-GKM-Work-Protection`. Penggantinya dibentuk
-- mastersupplier.ComposeApprovalID; lihat doc comment-nya untuk alasan bentuknya dan untuk
-- keputusan yang masih terbuka.
--
-- ## Sisi pemutusnya TIDAK ADA di export
--
-- Hanya penyisipan ini yang ada. Tidak ada satu pun rule di seluruh export yang MEMBACA
-- `proteksi_klaimmbu`, menyetujuinya, menolaknya, atau memajukan POSISI-nya — pencarian
-- atas nama tabel itu dan atas `PROTEKSI_TIPE` menemukan tepat satu berkas, yaitu kueri
-- yang ditiru di sini (R-16). Layar yang memutuskannya adalah lingkup tersendiri.
INSERT INTO POOLDATA.PROTEKSI_KLAIMMBU
       (PROTEKSI_ID, TGL_INPUT, PROTEKSI_TIPE, APPROVAL, USER_REQ,
        CATATAN, ALASAN_REQ, POSISI, JSONDATA, NO_KLAIM)
VALUES (:1, :2, '18', :3, :4,
        :5, :6, :7, :8, :9)

-- name: approval_check_table
--
-- Memastikan tabel antrean persetujuan ada dan dapat DITULIS akun aplikasi — diperiksa
-- lewat pembacaan, tanpa mengambil satu baris pun.
--
-- Ia penting justru karena tabelnya milik proses lain: supplier yang tersimpan tanpa baris
-- permintaannya akan tertahan selamanya tanpa satu pun tanda di layar.
SELECT PROTEKSI_ID
  FROM POOLDATA.PROTEKSI_KLAIMMBU
 WHERE 1 = 0

-- name: approval_count_pending
--
-- Jumlah permintaan supplier yang masih di posisi awal. Dipakai mode periksa, bukan layar:
-- ia menjawab "berapa banyak yang menunggu" sebelum layarnya dibuka.
SELECT COUNT(PROTEKSI_ID)
  FROM POOLDATA.PROTEKSI_KLAIMMBU
 WHERE TRIM(PROTEKSI_TIPE) = '18'
   AND TRIM(POSISI) = :1
