-- Kueri modul Master Sparepart.
--
-- SATU tabel DITULIS aplikasi ini — POOLDATA.SPAREPART_HE — ditambah empat objek yang
-- hanya DIBACA: POOLDATA.GCNM_M_SPAREPART_CATEGORY, POOLDATA.GCNM_M_SPAREPART_TYPE,
-- POOLDATA.M_SITE_DATABASE, dan sequence SPAREPART_HE_SEQ. Seluruhnya milik sistem lama
-- (ADR-0004, penulis tunggal per tabel).
--
-- Kewenangan menulis SPAREPART_HE berpindah dari Pega ke Go saat modulnya lulus gerbang 2.
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
--
-- ============================================================================
-- KENAPA MENULIS KE SPAREPART_HE, PADAHAL PEGA MENULIS KE M_SPAREPART_HE_BU
-- ============================================================================
--
-- Sistem lama memakai DUA tabel untuk satu master, dan itu pola dua-penyimpanan yang
-- `03-CURRENT-ARCHITECTURE.md` §3.2 catat sebagai R-10:
--
--   tulis  Activity/UpdateSparepartHE_act   ->  InputSparepart.NAMA_SPART := @GCNM.GetPageJSONString()
--          RDB List/UpdateSparepartHE-SQL   ->  POOLDATA.PEGA_M_SPAREPART_HE(Datapega, IDPega, out)
--          Database/PEGA_M_SPAREPART_HE.prc:24 -> INSERT INTO POOLDATA.M_SPAREPART_HE_BU(ID, JSONDATA)
--
--   baca   Report Definition/BrowseSparepartHE_RD  ->  POOLDATA.SPAREPART_HE, 23 kolom
--          RDB List/ValidationMasterSparepartNo    ->  pooldata.sparepart_he
--          RDB List/ValidationMasterSparepartName  ->  pooldata.sparepart_he
--          RDB List/ValidationMasterSparepartCode  ->  pooldata.sparepart_he
--          RDB List/CountMasterSparepartManager    ->  POOLDATA.SPAREPART_HE
--          RDB List/GetIDDokumenSparepart          ->  POOLDATA.SPAREPART_HE
--          RDB List/GetDataSparepart               ->  pooldata.sparepart_he
--
-- PERHATIKAN NAMA TABEL TULISNYA: `M_SPAREPART_HE_BU`, dengan akhiran `_BU` — bukan
-- `M_SPAREPART_HE` seperti yang disebut `Activity/SetApprovalAllMaster` sebagai jenis
-- master. Keduanya nama yang berbeda untuk hal yang berdekatan, dan tidak ada satu pun
-- keterangan di export tentang apa arti akhiran itu.
--
-- Menulisnya lewat procedure dilarang D-02. Yang tersisa adalah dua pilihan, dan hanya
-- satu yang dapat dikerjakan tanpa menebak:
--
--   (a) menulis dokumen JSON ke M_SPAREPART_HE_BU.JSONDATA dengan SQL biasa.
--       TIDAK DAPAT DIKERJAKAN: nama kunci JSON-nya diterbitkan fungsi
--       `@GCNM.GetPageJSONString()`, dan `Function/GetPageJSONString-Function.xml` HANYA
--       memuat tanda tangannya — badan fungsinya tidak ikut di export (R-16). Setiap
--       kunci yang ditulis akan menjadi tebakan.
--
--   (b) menulis kolom bernama ke SPAREPART_HE.
--       DAPAT DIKERJAKAN: kedua puluh tiga kolomnya terbaca lengkap dari
--       `BrowseSparepartHE_RD-RD.xml`, ditambah DOKUMENID dari
--       `RDB List/GetIDDokumenSparepart-SQL.xml` atas tabel yang sama.
--
-- (b) yang dipakai. Perlakuannya sama dengan Master Bengkel, Master Panel, dan Master
-- Status Klaim, yang menghadapi keluarga procedure PEGA_M_* yang sama persis dan
-- memutuskan hal yang sama (D-02, D-68): Go menjadi penulis tunggal dan berhenti menulis
-- JSONDATA.
--
-- Satu hal lagi yang TIDAK dibawa dari procedure itu: kontrak galat berbasis string.
-- `PEGA_M_SPAREPART_HE.prc:23` menuliskan `ErrMsg := 'Data Sudah Disimpan dengan ID : '`
-- pada jalur SUKSES, lewat parameter yang namanya berarti pesan galat. Satu kolom keluaran
-- memikul dua arti — persis pola yang `D-68` tolak pada ADD_NEWMASTERVIRTUALACCOUNT.
--
--
-- ============================================================================
-- TIPE KEDUA KOLOM TANGGAL — ASUMSI YANG DISADARI
-- ============================================================================
--
-- `PROD_DATE` dan `TGL_UPDATE_HARGA` tipenya belum diketahui (R-08), dan export tidak
-- memberi satu pun petunjuk: tidak ada TO_CHAR, TRUNC, maupun TO_DATE yang dikenakan pada
-- keduanya di seluruh export.
--
-- Yang dipakai di berkas ini, beserta alasannya:
--
--   PROD_DATE         TEKS. Pega merendernya `pxTextInput` TANPA pyDateTimeFormat —
--                     bukan kontrol tanggal. Nilainya diketik dan disalin apa adanya
--                     (`Activity/SetDataSparepart-Act.xml`).
--
--   TGL_UPDATE_HARGA  WAKTU. Pega mengisinya `@DateTime.CurrentDateTime()`, dan properti
--                     DateTime Pega dipetakan ke kolom DATE.
--
-- Bila salah satunya ternyata bertipe lain, penyimpanan akan gagal dengan galat konversi
-- pada percobaan pertama — gagal keras dan terlihat, bukan diam-diam menulis nilai yang
-- salah. Pemeriksaannya sudah terpasang: `claimpnc -periksa` membaca kedua kolom lewat
-- sparepart_check_table. Lihat checkSparepart pada cmd/claimpnc.
--
--
-- CATATAN TRIM pada setiap penyaring kunci. Tipe kolom belum diketahui (R-08). Bila ID
-- bertipe CHAR berlebar tetap, nilainya dipadatkan spasi tanpa tanda apa pun; Oracle
-- membandingkan CHAR dengan CHAR secara blank-padded, sehingga kueri lama yang MERANGKAI
-- nilainya tetap cocok. Parameter binding bertipe VARCHAR2, dan perbandingan CHAR dengan
-- VARCHAR2 memakai non-padded comparison: "ABC " tidak sama dengan "ABC", dan barisnya
-- tidak ketemu. Menyalin `= :1` apa adanya karena itu justru MENGUBAH perilaku. TRIM benar
-- untuk kedua kemungkinan tipe. Biayanya index atas kolom itu tidak terpakai; dapat
-- diterima pada tabel master berbaris sedikit, dan TIDAK boleh ditiru pada tabel besar.


-- name: sparepart_list
--
-- Asal: Report Definition/BrowseSparepartHE_RD-RD.xml, dipakai ketiga tab
-- `Section/BrowseMasterSparepartHE-Section.xml` yang hanya berbeda pada nilai APPROVAL-nya.
--
-- Kedua puluh empat kolomnya disebut pada urutan yang SAMA dengan sparepart_get dan ketiga
-- kueri pencarian kunci — satu fungsi scanRow membaca kelimanya berdasarkan POSISI, dan
-- satu kolom yang bergeser akan menaruh harga jual ke kolom berat tanpa satu pun galat.
-- `TestReaderQueriesShareColumnOrder` yang menjaganya.
--
-- DOKUMENID ikut dibaca meski `BrowseSparepartHE_RD` tidak menyebutnya: ia dibaca terpisah
-- oleh `RDB List/GetIDDokumenSparepart-SQL.xml` atas tabel yang sama. Membacanya sekaligus
-- menghindarkan satu perjalanan basis data per baris, dan membuat penyimpanan dapat
-- menulis balik nilainya apa adanya alih-alih mengosongkannya.
--
-- ORDER BY DITAMBAHKAN. `BrowseSparepartHE_RD` tidak menyebut satu pun kolom pengurut —
-- `pySortOrder` bernilai 99999 pada SELURUH kolomnya, yang berarti "tidak diurutkan" —
-- sehingga Pega menerima baris dalam urutan apa pun yang dikembalikan Oracle. Urutan yang
-- tidak ditentukan tidak dapat dipakai layar: daftar yang berubah urutan antar pemuatan
-- membuat baris melompat di bawah kursor pengguna.
--
-- `ID` yang dipilih, menaik, karena ia kunci yang diterbitkan sequence — sehingga urutannya
-- sama dengan urutan penerbitan, yaitu urutan yang paling mendekati apa yang dikembalikan
-- Oracle hari ini untuk tabel master yang jarang dihapus.
--
-- `pyMaxRecords=500` pada report definition lama TIDAK direplikasi. Ia memotong daftar di
-- 500 baris tanpa satu pun tanda di layar; penggantinya adalah penyaring kata kunci pada
-- sparepart_list_search.
SELECT ID,
       NAMA_SPART,
       NO_SPART,
       KODE_SPART,
       HARGA_JUAL,
       KATEGORI_SPART,
       TIPE_SPART,
       BERAT,
       PANJANG,
       LEBAR,
       TINGGI,
       MIN_STOCK,
       MAX_STOCK,
       QTY_PESAN,
       PROD_DATE,
       SUBSTITUSI_SPART,
       JENIS_SPART,
       SATUAN,
       STS_AKTIF,
       STS_PART,
       USER_UPDATE,
       TGL_UPDATE_HARGA,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.SPAREPART_HE
 WHERE TRIM(APPROVAL) = :1
 ORDER BY ID

-- name: sparepart_list_search
--
-- Sama dengan sparepart_list, ditambah penyaring kata kunci.
--
-- KUERI TERSENDIRI, bukan satu kueri yang klausanya ditempel. Sistem lama menempuh cara
-- yang kedua di banyak tempat lewat pola `{ASIS:...}` — nilai dirangkai langsung ke teks
-- SQL — dan `08-TECHNICAL-STRATEGY.md` §4.3 melarangnya tanpa perkecualian.
--
-- TIGA kolom dicari sekaligus: nama, nomor, DAN kode. Ketiganya kunci alami di modul ini,
-- dan petugas gudang mencari suku cadang lewat nomornya jauh lebih sering daripada lewat
-- namanya. Master Panel hanya mencari pada namanya karena hanya itu kunci alaminya.
--
-- UPPER dipasang pada KOLOMNYA juga, bukan hanya pada kata kuncinya: ketiganya tersimpan
-- dengan besar-kecil huruf apa adanya, dan pencarian yang hanya meng-uppercase kata kunci
-- tidak akan pernah menemukan baris yang isinya huruf kecil.
--
-- `ESCAPE '\'` disebut eksplisit karena Oracle TIDAK punya karakter pelolos bawaan pada
-- LIKE. PostgreSQL memakai backslash sebagai bawaan; menyebutkannya eksplisit membuat
-- kedua basis data berperilaku sama (D-20).
--
-- Parameter kata kuncinya disebut TIGA KALI (:2, :3, :4) dengan nilai yang sama, bukan
-- sekali. Oracle memperlakukan setiap penanda posisi sebagai bind terpisah, sehingga
-- menyebut `:2` tiga kali akan menuntut driver mengirim satu nilai untuk tiga posisi —
-- perilaku yang berbeda antar driver. Tiga penanda dengan nilai yang sama berperilaku sama
-- di keduanya.
SELECT ID,
       NAMA_SPART,
       NO_SPART,
       KODE_SPART,
       HARGA_JUAL,
       KATEGORI_SPART,
       TIPE_SPART,
       BERAT,
       PANJANG,
       LEBAR,
       TINGGI,
       MIN_STOCK,
       MAX_STOCK,
       QTY_PESAN,
       PROD_DATE,
       SUBSTITUSI_SPART,
       JENIS_SPART,
       SATUAN,
       STS_AKTIF,
       STS_PART,
       USER_UPDATE,
       TGL_UPDATE_HARGA,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.SPAREPART_HE
 WHERE TRIM(APPROVAL) = :1
   AND (UPPER(NAMA_SPART) LIKE :2 ESCAPE '\'
        OR UPPER(NO_SPART) LIKE :3 ESCAPE '\'
        OR UPPER(KODE_SPART) LIKE :4 ESCAPE '\')
 ORDER BY ID

-- name: sparepart_get
--
-- Kolomnya sama dan pada urutan yang sama dengan sparepart_list.
SELECT ID,
       NAMA_SPART,
       NO_SPART,
       KODE_SPART,
       HARGA_JUAL,
       KATEGORI_SPART,
       TIPE_SPART,
       BERAT,
       PANJANG,
       LEBAR,
       TINGGI,
       MIN_STOCK,
       MAX_STOCK,
       QTY_PESAN,
       PROD_DATE,
       SUBSTITUSI_SPART,
       JENIS_SPART,
       SATUAN,
       STS_AKTIF,
       STS_PART,
       USER_UPDATE,
       TGL_UPDATE_HARGA,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.SPAREPART_HE
 WHERE TRIM(ID) = :1

-- name: sparepart_find_by_name
--
-- Padanan `RDB List/ValidationMasterSparepartName-SQL.xml`:
--
--   select NAMA_SPART from pooldata.sparepart_he
--    where upper(NAMA_SPART) = {TempInputPanelHE.CaseID}
--
-- PERHATIKAN NAMA HALAMAN PEMBANDINGNYA: `TempInputPanelHE` — halaman milik Master PANEL,
-- dipakai di rule Master SPAREPART. Ketiga rule validasi sparepart memakainya, dan ketiga
-- propertinya pun tidak ada hubungannya dengan isinya: `.CaseID` untuk nama, `.City` untuk
-- nomor, `.Country` untuk kode. Itu persis bentuk utang yang
-- `03-CURRENT-ARCHITECTURE.md` §4.2 catat, dan tidak satu pun dibawa.
--
-- Yang berubah selain itu hanyalah cara nilainya sampai — parameter binding menggantikan
-- perangkaian teks — dan kolom yang dibaca, supaya pemanggil dapat menyebut sparepart mana
-- yang memakai nama itu alih-alih hanya mengatakan "sudah dipakai".
--
-- TRIM DITAMBAHKAN pada kolomnya. Kueri lama hanya meng-UPPER tanpa memangkas, sehingga
-- baris yang tersimpan dengan spasi di ujung lolos sebagai nama yang berbeda. Itu selisih
-- yang direncanakan: ia menolak baris yang di sistem lama diterima, dan baris lama tetap
-- dibaca apa adanya.
--
-- Kolomnya sama dan pada urutan yang sama dengan sparepart_list.
SELECT ID,
       NAMA_SPART,
       NO_SPART,
       KODE_SPART,
       HARGA_JUAL,
       KATEGORI_SPART,
       TIPE_SPART,
       BERAT,
       PANJANG,
       LEBAR,
       TINGGI,
       MIN_STOCK,
       MAX_STOCK,
       QTY_PESAN,
       PROD_DATE,
       SUBSTITUSI_SPART,
       JENIS_SPART,
       SATUAN,
       STS_AKTIF,
       STS_PART,
       USER_UPDATE,
       TGL_UPDATE_HARGA,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.SPAREPART_HE
 WHERE UPPER(TRIM(NAMA_SPART)) = :1

-- name: sparepart_find_by_number
--
-- Padanan `RDB List/ValidationMasterSparepartNo-SQL.xml`. Kolomnya sama dan pada urutan
-- yang sama dengan sparepart_list.
SELECT ID,
       NAMA_SPART,
       NO_SPART,
       KODE_SPART,
       HARGA_JUAL,
       KATEGORI_SPART,
       TIPE_SPART,
       BERAT,
       PANJANG,
       LEBAR,
       TINGGI,
       MIN_STOCK,
       MAX_STOCK,
       QTY_PESAN,
       PROD_DATE,
       SUBSTITUSI_SPART,
       JENIS_SPART,
       SATUAN,
       STS_AKTIF,
       STS_PART,
       USER_UPDATE,
       TGL_UPDATE_HARGA,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.SPAREPART_HE
 WHERE UPPER(TRIM(NO_SPART)) = :1

-- name: sparepart_find_by_code
--
-- Padanan `RDB List/ValidationMasterSparepartCode-SQL.xml`. Kolomnya sama dan pada urutan
-- yang sama dengan sparepart_list.
SELECT ID,
       NAMA_SPART,
       NO_SPART,
       KODE_SPART,
       HARGA_JUAL,
       KATEGORI_SPART,
       TIPE_SPART,
       BERAT,
       PANJANG,
       LEBAR,
       TINGGI,
       MIN_STOCK,
       MAX_STOCK,
       QTY_PESAN,
       PROD_DATE,
       SUBSTITUSI_SPART,
       JENIS_SPART,
       SATUAN,
       STS_AKTIF,
       STS_PART,
       USER_UPDATE,
       TGL_UPDATE_HARGA,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.SPAREPART_HE
 WHERE UPPER(TRIM(KODE_SPART)) = :1

-- name: sparepart_lock_by_keys
--
-- Dipakai di dalam transaksi penambahan. FOR UPDATE mengunci baris yang SUDAH ADA sehingga
-- dua penambahan atas kunci yang sama tidak dapat berjalan berdampingan.
--
-- KETIGA kunci alami diperiksa dalam SATU kueri, bukan tiga kueri berturut-turut. Tiga
-- kueri berarti tiga perjalanan basis data di dalam satu transaksi yang menahan kunci
-- baris, dan tabel ini dibaca Pega yang sedang melayani produksi (D-21).
--
-- Kolom yang dibaca menyebut kunci MANA yang bentrok, sehingga pemanggil dapat melaporkan
-- isian yang tepat alih-alih mengatakan "salah satu sudah dipakai".
--
-- SEBERAPA JAUH LUBANGNYA TERTUTUP: tidak sepenuhnya. FOR UPDATE tidak dapat mengunci
-- baris yang belum ada, sehingga dua penambahan yang sama-sama menemukan nol baris tetap
-- lolos berdampingan. Yang benar-benar menutupnya adalah constraint unik pada ketiga
-- kolom, dan itu menunggu DDL (R-08) serta prosedur perubahan skema (D-63).
SELECT ID,
       NAMA_SPART,
       NO_SPART,
       KODE_SPART
  FROM POOLDATA.SPAREPART_HE
 WHERE UPPER(TRIM(NAMA_SPART)) = :1
    OR UPPER(TRIM(NO_SPART)) = :2
    OR UPPER(TRIM(KODE_SPART)) = :3
 FOR UPDATE

-- name: sparepart_insert
--
-- Kedua puluh empat kolomnya pada urutan yang sama dengan insertArguments. Urutan itu
-- WAJIB sama; `TestInsertArgumentsMatchColumnOrder` yang menjaganya.
INSERT INTO POOLDATA.SPAREPART_HE
       (ID, NAMA_SPART, NO_SPART, KODE_SPART, HARGA_JUAL,
        KATEGORI_SPART, TIPE_SPART, BERAT, PANJANG, LEBAR,
        TINGGI, MIN_STOCK, MAX_STOCK, QTY_PESAN, PROD_DATE,
        SUBSTITUSI_SPART, JENIS_SPART, SATUAN, STS_AKTIF, STS_PART,
        USER_UPDATE, TGL_UPDATE_HARGA, DOKUMENID, APPROVAL)
VALUES (:1, :2, :3, :4, :5,
        :6, :7, :8, :9, :10,
        :11, :12, :13, :14, :15,
        :16, :17, :18, :19, :20,
        :21, :22, :23, :24)

-- name: sparepart_update
--
-- ID TIDAK disebut sebagai kolom yang ditulis — ia penyaring WHERE, dan berada di posisi
-- terakhir pada updateArguments. Kunci baris tidak pernah berpindah.
UPDATE POOLDATA.SPAREPART_HE
   SET NAMA_SPART = :1,
       NO_SPART = :2,
       KODE_SPART = :3,
       HARGA_JUAL = :4,
       KATEGORI_SPART = :5,
       TIPE_SPART = :6,
       BERAT = :7,
       PANJANG = :8,
       LEBAR = :9,
       TINGGI = :10,
       MIN_STOCK = :11,
       MAX_STOCK = :12,
       QTY_PESAN = :13,
       PROD_DATE = :14,
       SUBSTITUSI_SPART = :15,
       JENIS_SPART = :16,
       SATUAN = :17,
       STS_AKTIF = :18,
       STS_PART = :19,
       USER_UPDATE = :20,
       TGL_UPDATE_HARGA = :21,
       DOKUMENID = :22,
       APPROVAL = :23
 WHERE TRIM(ID) = :24

-- name: sparepart_set_status
--
-- Satu pernyataan per baris, bukan satu pernyataan dengan daftar kunci yang panjangnya
-- berubah-ubah. Daftar `IN (:1, :2, …)` yang panjangnya mengikuti jumlah baris menghasilkan
-- teks SQL yang berbeda setiap kali dipanggil — setiap bentuk menempati satu slot pada
-- shared pool Oracle, dan tabel master yang diputuskan borongan akan menghasilkan puluhan
-- bentuk berbeda dari satu operasi yang sama.
--
-- `AND TRIM(APPROVAL) <> :1` membuat baris yang sudah berstatus itu TIDAK terhitung sebagai
-- berubah — itulah yang membuat jumlah yang dilaporkan ke layar bermakna.
--
-- HANYA APPROVAL yang disentuh, tanpa alasan penolakan dan tanpa USER_UPDATE. Itu meniru
-- `Activity/SetApprovalAllMaster` apa adanya: ia menetapkan `APPROVAL := Param.approval`
-- dan tidak satu pun kolom lain. USER_UPDATE karena itu tetap berisi siapa yang MENGAJUKAN,
-- bukan siapa yang memutuskan — dan tidak ada tempat di tabel ini untuk mencatat yang
-- kedua.
UPDATE POOLDATA.SPAREPART_HE
   SET APPROVAL = :1
 WHERE TRIM(ID) = :2
   AND TRIM(APPROVAL) <> :1

-- name: sparepart_category_list
--
-- Padanan `RDB List/BrowseMasterSparepartCategoryClaimHE-SQL.xml` yang dijalankan
-- `Activity/BrowseTipeKategoriPart-Act.xml` dengan `TempStatus.City := "1"`:
--
--   select PART_CATEGORY_ID as "CityID", PART_CATEGORY_NAME as "City"
--     from POOLDATA.gcnm_m_sparepart_category where APPROVAL = {TempStatus.City}
--
-- Alias `"CityID"` dan `"City"` TIDAK dibawa — keduanya nama yang tidak ada hubungannya
-- dengan isinya. Lihat catatan pada mastersparepart.Category.
--
-- ORDER BY DITAMBAHKAN; kueri lama tidak menyebut satu pun. Nama yang dipilih, bukan ID,
-- karena inilah daftar yang dipindai mata pengguna pada dropdown.
SELECT PART_CATEGORY_ID,
       PART_CATEGORY_NAME
  FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY
 WHERE TRIM(APPROVAL) = :1
 ORDER BY PART_CATEGORY_NAME

-- name: sparepart_type_list
--
-- Padanan `RDB List/BrowseTipeSparepart-SQL.xml`:
--
--   select PART_SECTION_ID as "CityID", PART_SECTION_NAME as "City",
--          PART_CATEGORY_ID as "District"
--     from POOLDATA.gcnm_m_sparepart_type where APPROVAL = '1'
--    order by PART_SECTION_ID desc
--
-- PART_CATEGORY_ID ikut dibaca karena Tipe bercabang dari Kategori; layar memakainya untuk
-- mempersempit daftar Tipe begitu Kategori dipilih.
--
-- TANPA JOIN ke tabel kategori. `BrowseSparepartTypeClaimHE_sql` melakukannya untuk
-- mengambil nama kategorinya sekaligus, tetapi layar ini sudah memuat daftar kategori
-- lengkap dalam jawaban yang sama — menariknya lagi lewat JOIN berarti mengirim nama yang
-- sama berulang kali, sekali untuk setiap tipe.
--
-- `order by PART_SECTION_ID desc` pada kueri lama DIGANTI urutan menurut nama. Urutan
-- menurun berdasarkan ID pada sebuah daftar pilihan tidak menolong siapa pun mencari, dan
-- ia bukan tata letak yang dilihat pengguna — dropdown Pega menampilkan namanya.
SELECT PART_SECTION_ID,
       PART_SECTION_NAME,
       PART_CATEGORY_ID
  FROM POOLDATA.GCNM_M_SPAREPART_TYPE
 WHERE TRIM(APPROVAL) = :1
 ORDER BY PART_SECTION_NAME

-- name: sparepart_count_by_status
--
-- Padanan `RDB List/CountMasterSparepartManager-SQL.xml`:
--
--   SELECT count(ID) FROM POOLDATA.SPAREPART_HE WHERE APPROVAL = 0
--
-- PERHATIKAN PEMBANDINGNYA pada kueri lama: angka `0`, bukan teks `'0'` — sementara ketiga
-- section tab membandingkannya sebagai teks. Bila kolomnya bertipe teks, perbandingan
-- dengan angka memaksa konversi implisit yang perilakunya berbeda antar basis data, dan
-- pada baris yang isinya bukan angka ia melempar ORA-01722.
--
-- Di sini ia selalu teks, sama dengan seluruh kueri lain di berkas ini.
SELECT COUNT(ID)
  FROM POOLDATA.SPAREPART_HE
 WHERE TRIM(APPROVAL) = :1

-- name: sparepart_count_all
SELECT COUNT(ID)
  FROM POOLDATA.SPAREPART_HE

-- name: sparepart_check_table
--
-- Tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi. Yang
-- dibuktikannya hanyalah tabelnya ada dan kedua puluh empat kolomnya dapat dibaca akun
-- aplikasi — TERMASUK kedua kolom tanggal, justru kolom yang asumsi tipenya perlu
-- dibuktikan sebelum jalur tulis dipakai.
SELECT ID,
       NAMA_SPART,
       NO_SPART,
       KODE_SPART,
       HARGA_JUAL,
       KATEGORI_SPART,
       TIPE_SPART,
       BERAT,
       PANJANG,
       LEBAR,
       TINGGI,
       MIN_STOCK,
       MAX_STOCK,
       QTY_PESAN,
       PROD_DATE,
       SUBSTITUSI_SPART,
       JENIS_SPART,
       SATUAN,
       STS_AKTIF,
       STS_PART,
       USER_UPDATE,
       TGL_UPDATE_HARGA,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.SPAREPART_HE
 WHERE 1 = 0

-- name: sparepart_check_category_table
SELECT PART_CATEGORY_ID,
       PART_CATEGORY_NAME,
       APPROVAL
  FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY
 WHERE 1 = 0

-- name: sparepart_check_type_table
SELECT PART_SECTION_ID,
       PART_SECTION_NAME,
       PART_CATEGORY_ID,
       APPROVAL
  FROM POOLDATA.GCNM_M_SPAREPART_TYPE
 WHERE 1 = 0

-- name: sparepart_check_json_mirror
--
-- Tabel JSON milik Pega, dengan akhiran `_BU` yang artinya tidak diketahui. Perbandingan
-- jumlah barisnya dengan SPAREPART_HE adalah cara termurah mengetahui apakah keduanya satu
-- sumber; lihat banner berkas ini.
SELECT ID
  FROM POOLDATA.M_SPAREPART_HE_BU
 WHERE 1 = 0

-- name: sparepart_count_json_mirror
SELECT COUNT(ID)
  FROM POOLDATA.M_SPAREPART_HE_BU

-- name: sparepart_count_orphan_category
--
-- Baris yang KATEGORI_SPART-nya tidak ada di tabel kategori.
--
-- Ia tidak dapat lahir lewat modul ini — dropdown hanya menawarkan kategori yang ada —
-- tetapi dapat sudah ada di data warisan, dan jumlahnya menentukan apakah dropdown yang
-- menyaring `APPROVAL = '1'` akan menampilkan baris lama sebagai "kategori tidak dikenal".
--
-- Baris yang kolomnya kosong TIDAK terhitung: kategori memang boleh tidak diisi.
SELECT COUNT(S.ID)
  FROM POOLDATA.SPAREPART_HE S
 WHERE S.KATEGORI_SPART IS NOT NULL
   AND TRIM(S.KATEGORI_SPART) <> ''
   AND NOT EXISTS (
       SELECT 1
         FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY C
        WHERE TRIM(C.PART_CATEGORY_ID) = TRIM(S.KATEGORI_SPART))

-- name: sparepart_count_orphan_type
--
-- Baris yang TIPE_SPART-nya tidak ada di tabel tipe. Alasannya sama.
SELECT COUNT(S.ID)
  FROM POOLDATA.SPAREPART_HE S
 WHERE S.TIPE_SPART IS NOT NULL
   AND TRIM(S.TIPE_SPART) <> ''
   AND NOT EXISTS (
       SELECT 1
         FROM POOLDATA.GCNM_M_SPAREPART_TYPE T
        WHERE TRIM(T.PART_SECTION_ID) = TRIM(S.TIPE_SPART))

-- name: sparepart_site
--
-- Kode situs, dibaca persis seperti `Database/PEGA_M_SPAREPART_HE.prc:12`:
--
--   SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
--
-- Pembandingnya TEKS '1', bukan angka 1, dan itu ditiru apa adanya: bila kolomnya bertipe
-- teks, perbandingan dengan angka akan memaksa konversi implisit yang perilakunya berbeda
-- antar basis data.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: sparepart_next_sequence
--
-- Sequence yang SAMA dengan yang dipakai procedure lama, supaya ID yang diterbitkan
-- aplikasi ini melanjutkan deret yang sudah ada dan tidak pernah bertabrakan dengan ID yang
-- pernah diterbitkan Pega.
--
-- `Database/PEGA_M_SPAREPART_HE.prc:21` menyebutnya tanpa skema (`SPAREPART_HE_SEQ`),
-- sehingga yang terpakai bergantung pada skema bawaan akun koneksi — dan itu berbeda antar
-- lingkungan. Di sini ia dilengkapi POOLDATA, skema yang sama dengan tabel yang disisipi
-- procedure itu.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada ADR-0005, satu-satunya tempat lain
-- yang dibenarkan memuat percabangan dialek.
SELECT POOLDATA.SPAREPART_HE_SEQ.NEXTVAL
  FROM DUAL
