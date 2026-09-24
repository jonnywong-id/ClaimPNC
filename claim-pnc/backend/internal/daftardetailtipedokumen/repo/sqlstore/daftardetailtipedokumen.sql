-- Kueri modul Daftar Detail Tipe Dokumen (MENU_ID 41).
--
-- ============================================================================
-- BACA BAGIAN INI SEBELUM MENGUBAH SATU BARIS PUN DI BAWAHNYA.
-- ============================================================================
--
-- # BERKAS INI ADALAH SATU-SATUNYA TEMPAT NAMA OBJEK BASIS DATA DISEBUT
--
-- Itu disengaja: bila skemanya kelak berubah, yang disunting hanya berkas ini — tidak ada
-- satu pun nama tabel yang tercecer di dalam kode Go.
--
-- Seluruh nama dan kolom di bawah **sudah diverifikasi langsung ke katalog Oracle**
-- (2026-09-23), kecuali SATU yang memang belum ada dan sedang diminta ke DBA:
-- `POOLDATA.LST_DET_TYPE_DOC_BISNIS`.
--
-- # DIVERIFIKASI LANGSUNG KE ORACLE 2026-09-23 — tidak ada lagi dugaan kolom
--
-- Katalog dibaca langsung (`ALL_TAB_COLUMNS`, `ALL_VIEWS`) karena mode periksa menolak
-- jalur tulis dengan `ORA-00904: "TYPE_DOCUMENT": invalid identifier`. Hasilnya:
--
--   POOLDATA.LST_DET_TYPE_DOC   ADA, dan kolomnya SUDAH LENGKAP:
--       ID CHAR(5) · OLD_ID CHAR(3) · JSON_DATA CLOB · DETAIL_DOCUMENT · DOC_TYPE_ID(100)
--       USER_EDIT · STS_INSURED(100) · DOC_COL_ID(1000) · OBJ_DOC · DOC_COL_INFO · RISK
--       OBJ_DOC_DESC · TGL_EDIT            (seluruh sisanya VARCHAR2 4000)
--
--   POOLDATA.V_LST_DET_TYPE_DOC adalah SELECT KOLOM biasa — bukan JSON, bukan join:
--       SELECT a.ID, a.OLD_ID, a.DOC_TYPE_ID, a.DETAIL_DOCUMENT, a.USER_EDIT, a.TGL_EDIT,
--              a.STS_INSURED, a.DOC_COL_ID, a.OBJ_DOC, a.DOC_COL_INFO, a.RISK,
--              a.OBJ_DOC_DESC
--         FROM LST_DET_TYPE_DOC a
--
--   POOLDATA.V_LST_DET_TYPE_DOC_BISNIS adalah JSON_TABLE atas kolom JSON_DATA tabel yang
--   SAMA, memecah `$.DFT_BISNIS_ID[*]` menjadi BISNISID, STSWAJIB, dan MINDOC.
--
-- Tiga akibat yang mengikat berkas ini:
--
--   1. `TYPE_DOCUMENT` TIDAK ADA pada view induk. Ia harus di-LEFT JOIN sendiri dari
--      POOLDATA.V_LST_DOC_TYPE — persis yang dilakukan modul MENU_ID 42. Inilah yang
--      membuat layarnya gagal memuat sebelum diperbaiki.
--   2. Kedua keterangan `DOC_COL_INFO` dan `OBJ_DOC_DESC` MEMANG KOLOM. Keputusan
--      menulisnya terbukti benar.
--   3. **TABEL ANAK BELUM ADA.** Daftar bisnis masih hidup di dalam JSON_DATA induknya,
--      dan `POOLDATA.LST_DET_TYPE_DOC_BISNIS` tidak ada di katalog. Modul ini BACA MAUPUN
--      TULIS daftar bisnis dari tabel itu (keputusan Work Owner 2026-09-23: "pakai
--      database saja, sudah tidak pakai JSON lagi"), sehingga keduanya BELUM DAPAT
--      BERJALAN sampai DBA membuatnya — lihat migrations/0009 LANGKAH 1.
--
-- # JSON_DATA tidak disentuh sama sekali
--
-- Tidak dibaca, tidak ditulis, dan tidak dikosongkan. Isinya pada baris lama dibiarkan
-- sebagai arsip — `D-66` melarang penghapusan fisik data bernilai bisnis, dan LANGKAH 1b
-- menyalin isinya ke tabel anak alih-alih memindahkannya.
--
-- `TGL_EDIT` bertipe VARCHAR2, bukan DATE, dan isinya berformat Pega:
--
--       "20231030T075651.051 GMT"
--
-- Menulis `time.Time` ke sana akan menghasilkan bentuk yang berbeda dari seluruh baris
-- yang sudah ada. Pemformatannya ada di daftardetailtipedokumen.go, satu tempat saja.
--
--
-- # SATU kolom view yang tidak ditulis, dan DUA yang ditulis meski tampak keterangan
--
-- Pembedaannya dibaca dari halaman `TempDTDoc`, bukan ditebak dari namanya.
-- `Activity/CNMSetDetailTypeDocument_act-Act.xml` mengisi 12 properti pada halaman itu:
--
--   ID · DOC_TYPE_ID · DETAIL_DOCUMENT · STS_INSURED · DOC_COL_ID · DOC_COL_INFO
--   OBJ_DOC · OBJ_DOC_DESC · RISK · TGL_EDIT · USER_EDIT · DFT_BISNIS_ID
--
-- Halaman itulah yang diserialisasi `stepPage.getJSON(false)` menjadi dokumen JSON yang
-- disimpan procedure. Jadi:
--
--   TYPE_DOCUMENT   TIDAK ADA di halaman -> tidak mungkin tersimpan -> HASIL JOIN
--   DOC_COL_INFO    ADA di halaman       -> TERSIMPAN
--   OBJ_DOC_DESC    ADA di halaman       -> TERSIMPAN
--
-- Kedua keterangan yang tersimpan itu justru ISIAN YANG DILIHAT PETUGAS:
-- `Section/BrowseListDetailTypeDocument-Section.xml` mengikat isian berlabel "Dokumen
-- kolom ID" ke `TempDTDoc.DOC_COL_INFO` dan isian "Objek Dokumen" ke
-- `TempDTDoc.OBJ_DOC_DESC`; kedua kodenya hanya target tersembunyi yang diisi
-- autocomplete lewat `pyPropertyTarget`.
--
-- Karena ketiga autocomplete ber-`pyAllowFreeFormInput=true`, keterangan di luar master
-- tetap boleh diketik dan tersimpan — dan pada keadaan itu kodenya kosong. Menyusun
-- keterangannya kembali lewat join akan MEMBUANG tepat isian itu.
--
-- Bandingkan dengan view ANAK, yang berperilaku sebaliknya: nama bisnis TIDAK ada di
-- sana, dan `RDB List/GetLbuDetType-SQL.xml` menjoin POOLDATA.BUSINESS untuk
-- mendapatkannya. Asimetri itu nyata, dan ditiru apa adanya.
--
-- # Satu tabel satu penulis (P-1)
--
-- Layar ini adalah SATU-SATUNYA penulis POOLDATA.LST_DET_TYPE_DOC di sistem lama —
-- `RDB List/UpdateDetTypeDoc-SQL.xml` satu-satunya rule yang memanggil
-- PEGA_LST_DET_TYPE_DOC, dan tidak ada rule lain yang menyentuh tabelnya. `P-1` karena
-- itu terpenuhi utuh, berbeda dari Master Penyebab Kerugian yang masih punya dua penulis.
--
-- Keempat master yang dirujuk HANYA DIBACA di sini. Tidak ada satu pun INSERT, UPDATE,
-- maupun DELETE terhadap keempatnya di berkas ini, dan tidak boleh ada.
--
-- # Yang membaca view induk, dan kenapa itu penting
--
-- 34 rule Pega membacanya, di antaranya SELURUH jalur validasi unggah dokumen:
-- `ValidationUploadDocument_act`, `ValidationUploadRegister`, `RequiredDocument_act`,
-- `RequiredDocPA`, `InsertDokumenPNC`, ditambah arsip dokumen dan pencarian klaim.
-- Selama masa paralel, apa pun yang ditulis modul ini WAJIB terbaca oleh view itu — bila
-- tidak, dokumen yang diminta pada klaim berhenti muncul TANPA satu pun galat.
--
-- Empat aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (`D-20`).
--   4. Tanpa pemanggilan stored procedure (`D-02`).


-- name: detail_list
--
-- Grid layar. Tiga kolom yang benar-benar ditampilkan
-- (`Section/BrowseListDetailTypeDocument-Section.xml`): ID, Tipe Dokumen, dan Detail
-- Dokumen — ditambah keterangan penyebab kerugian dan objek dokumen supaya daftar dapat
-- dicari dengan keduanya tanpa membuka satu per satu.
--
-- Daftar bisnis sengaja TIDAK ikut di sini: gridnya tidak menampilkannya, dan menariknya
-- untuk seluruh baris berarti satu kueri yang hasilnya tidak pernah dilihat siapa pun.
--
-- Diurutkan menurut ID, mengikuti `BrowseVLstDetTypeDoc_RD-RD.xml`.
--
-- Batas `pyMaxRecords=500` milik Report Definition itu tidak ditiru. Ia bukan aturan
-- bisnis melainkan pemotongan senyap (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2), dan
-- menirunya berarti menyembunyikan baris yang benar-benar ada dari petugas yang sedang
-- menyuntingnya.
SELECT a.ID,
       a.DOC_TYPE_ID,
       t.TYPE_DOCUMENT,
       a.DETAIL_DOCUMENT,
       a.STS_INSURED,
       a.DOC_COL_ID,
       a.DOC_COL_INFO,
       a.OBJ_DOC,
       a.OBJ_DOC_DESC,
       a.RISK
  FROM POOLDATA.V_LST_DET_TYPE_DOC a
  LEFT JOIN POOLDATA.V_LST_DOC_TYPE t ON t.ID = a.DOC_TYPE_ID
 ORDER BY a.ID

-- name: detail_get
--
-- Satu baris untuk dimuat ke form sunting. Bentuk kolomnya sama persis dengan
-- detail_list supaya keduanya dapat dibaca satu fungsi pemindai — bila keduanya berbeda,
-- satu perubahan kolom harus diingat di dua tempat.
SELECT a.ID,
       a.DOC_TYPE_ID,
       t.TYPE_DOCUMENT,
       a.DETAIL_DOCUMENT,
       a.STS_INSURED,
       a.DOC_COL_ID,
       a.DOC_COL_INFO,
       a.OBJ_DOC,
       a.OBJ_DOC_DESC,
       a.RISK
  FROM POOLDATA.V_LST_DET_TYPE_DOC a
  LEFT JOIN POOLDATA.V_LST_DOC_TYPE t ON t.ID = a.DOC_TYPE_ID
 WHERE a.ID = :1

-- name: detail_business_list
--
-- Aturan per lini bisnis milik SATU rincian dokumen.
--
-- Bentuknya mengikuti `RDB List/GetLbuDetType-SQL.xml`:
--
--   select b.ID, NOTE as "Note", STS_WAJIB, MIN_DOC
--     from V_LST_DET_TYPE_DOC_BISNIS a, BUSINESS b
--    where a.DFT_BISNIS_ID = b.ID and a.ID = {INPUTDETYPE.ID}
--
-- # Dibaca dari TABEL DASAR, bukan dari view anaknya
--
-- Keputusan Work Owner 2026-09-23: **"pakai database saja, sudah tidak pakai JSON lagi."**
--
-- View anaknya hari ini adalah `JSON_TABLE` atas `LST_DET_TYPE_DOC.JSON_DATA`, sehingga
-- membacanya berarti tetap membaca JSON. Setelah migrasi 0009 LANGKAH 1c view itu memang
-- akan membaca tabel yang sama dengan yang dibaca di sini — tetapi ia dimiliki pihak lain,
-- dan modul ini MENULIS tabelnya sendiri.
--
-- Alasan yang sama dipakai modul Master Penyebab Kerugian: membaca dari tabel dasar
-- membuat apa yang kita tulis dan apa yang kita baca kembali PASTI sama, tanpa bergantung
-- pada definisi view yang dapat berubah di luar kendali modul ini.
--
-- **Sampai LANGKAH 1 dijalankan DBA, kueri ini gagal dengan ORA-00942** — tabelnya belum
-- ada. Itu disengaja dan dilaporkan mode periksa: lebih baik gagal terang-terangan
-- daripada menampilkan daftar bisnis kosong untuk baris yang sebenarnya punya isi di JSON.
--
-- View anaknya TETAP HARUS didefinisikan ulang pada LANGKAH 1c — bukan untuk modul ini,
-- melainkan untuk **sembilan rule Pega** yang masih membacanya.
--
-- Dua penyimpangan dari kueri lama, keduanya disengaja:
--
-- 1. LEFT JOIN, bukan INNER JOIN. Kueri lama merangkai keduanya dengan koma dan
--    menyamakannya di WHERE, sehingga baris yang DFT_BISNIS_ID-nya tidak lagi ada di
--    POOLDATA.BUSINESS HILANG dari form — tanpa pesan apa pun. Baris yang hilang dari
--    layar master tidak dapat diperbaiki petugas, dan menyimpan form akan MENGHAPUSNYA
--    karena grid mengirim susunan akhir. LEFT JOIN memunculkannya kembali dengan nama
--    master kosong.
--
-- 2. Kolom yang dibaca adalah `a.DFT_BISNIS_ID`, bukan `b.ID`. Keduanya sama nilainya
--    pada INNER JOIN, tetapi pada LEFT JOIN `b.ID` menjadi NULL tepat pada baris yang
--    ingin diselamatkan penyimpangan pertama.
--
-- Diurutkan menurut nama bisnis supaya susunan grid tidak berpindah-pindah di antara dua
-- kali pemuatan — kueri lama tidak mengurutkannya sama sekali, sehingga urutannya
-- ditentukan rencana eksekusi. Grid yang berpindah urutan tanpa sebab terbaca sebagai
-- datanya berubah.
SELECT a.DFT_BISNIS_ID,
       b.NOTE,
       a.STS_WAJIB,
       a.MIN_DOC
  FROM POOLDATA.LST_DET_TYPE_DOC_BISNIS a
  LEFT JOIN POOLDATA.BUSINESS b ON b.ID = a.DFT_BISNIS_ID
 WHERE a.ID = :1
 ORDER BY b.NOTE, a.DFT_BISNIS_ID

-- name: detail_site
--
-- Kode situs, bagian pertama dari setiap ID. Meniru
-- `Database/PEGA_LST_DET_TYPE_DOC.prc:11` persis, termasuk pembandingnya yang berupa teks
-- '1' dan bukan angka.
--
-- Inilah yang membuat tiap entitas menerbitkan awalan ID-nya sendiri, dan karena itulah
-- modul ini portal-aware.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: detail_next_sequence
--
-- Nomor urut ID. Namanya DIBACA dari `Database/PEGA_LST_DET_TYPE_DOC.prc:19`, bukan
-- diturunkan dari nama tabel.
--
-- Memakai urutan yang SAMA dengan procedure lama membuat ID yang diterbitkan aplikasi ini
-- melanjutkan deret yang sudah ada dan tidak pernah bertabrakan dengan ID yang pernah
-- diterbitkan Pega.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada `ADR-0005`, satu-satunya tempat
-- lain yang dibenarkan memuat percabangan dialek.
--
-- Pemformatan nomornya menjadi ID dikerjakan di Go, bukan dengan LPAD di sini: LPAD dan
-- TO_CHAR termasuk yang dilarang `09-DATABASE-STRATEGY.md` §4 karena keduanya mengikat
-- kueri pada dialek Oracle.
SELECT POOLDATA.LST_DET_TYPE_DOC_SEQ.NEXTVAL
  FROM DUAL

-- name: detail_insert
--
-- Sebelas kolom — kesebelasnya sama persis dengan properti `TempDTDoc` yang dulu masuk ke
-- dokumen JSON, dikurangi `DFT_BISNIS_ID` yang punya tabelnya sendiri.
--
-- JSON_DATA sengaja TIDAK diisi — keputusan Work Owner 2026-09-23: penyimpanan langsung ke
-- kolom, bukan ke dokumen JSON.
--
-- Akibat yang dicatat di migrasi 0009: baris yang ditulis aplikasi ini TIDAK punya
-- dokumen JSON. Bila kedua view ternyata masih membongkar JSON alih-alih membaca kolom,
-- baris baru akan terbaca KOSONG oleh 34 rule Pega yang membacanya — tanpa satu pun
-- galat. Itulah yang dibuktikan `detail_write_check_table` di bawah, sebelum pengguna
-- pertama menekan Simpan dan bukan sesudahnya.
--
-- OLD_ID tidak diisi: penomoran lama melekat pada baris warisan dan tidak pernah
-- diberikan pada baris baru.
INSERT INTO POOLDATA.LST_DET_TYPE_DOC
       (ID, DOC_TYPE_ID, DETAIL_DOCUMENT, STS_INSURED, DOC_COL_ID, DOC_COL_INFO,
        OBJ_DOC, OBJ_DOC_DESC, RISK, TGL_EDIT, USER_EDIT)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11)

-- name: detail_update
--
-- ID tidak pernah berubah — ia dirujuk `LST_TYPE_DOC_BUSINESS.DOC_TYPE_DT_ID` milik modul
-- MENU_ID 42 dan oleh baris bisnisnya sendiri. Ia hanya dipakai sebagai penyaring WHERE.
--
-- OLD_ID tidak ikut diubah: ia jejak sejarah.
UPDATE POOLDATA.LST_DET_TYPE_DOC
   SET DOC_TYPE_ID = :1,
       DETAIL_DOCUMENT = :2,
       STS_INSURED = :3,
       DOC_COL_ID = :4,
       DOC_COL_INFO = :5,
       OBJ_DOC = :6,
       OBJ_DOC_DESC = :7,
       RISK = :8,
       TGL_EDIT = :9,
       USER_EDIT = :10
 WHERE ID = :11

-- name: detail_business_delete_all
--
-- Seluruh baris bisnis milik satu rincian dibuang sebelum susunan barunya disisipkan.
--
-- # Kenapa DELETE, bukan penandaan tidak aktif seperti modul Daftar Objek Dokumen
--
-- Karena TIDAK ADA pembacanya yang menyaring penanda aktif. `GetLbuDetType-SQL.xml`
-- membaca V_LST_DET_TYPE_DOC_BISNIS tanpa satu pun syarat selain ID induknya, dan view
-- itu dibaca Pega hari ini juga.
--
-- Akibatnya baris yang ditandai tidak aktif akan TETAP DIBERLAKUKAN sebagai aturan
-- dokumen pada klaim — persis kebalikan dari yang dimaksud petugas saat membuangnya dari
-- grid, dan tidak terlihat sebagai galat di layar mana pun. Penandaan lunak di sini lebih
-- berbahaya daripada penghapusan.
--
-- Modul Daftar Objek Dokumen memilih sebaliknya, dan perbedaannya punya sebab: tabel di
-- sana BARU — dirancang di migrasi 0008 lengkap dengan kolom penandanya, dan tidak ada
-- satu pun pembaca Pega yang melewatinya.
--
-- # Kenapa ini tidak melanggar `D-66`
--
-- `D-66` melarang penghapusan fisik DATA BERNILAI BISNIS. Baris ini adalah aturan
-- penghubung yang seluruh isinya dapat disusun ulang dari susunan barunya, dan jejak
-- perubahannya menjadi tanggung jawab `S-5` — bukan tanggung jawab tabel yang pembacanya
-- tidak dapat membedakan aktif dari tidak.
--
-- Perlakuan yang sama sudah dipakai modul Daftar Detail Dokumen Travel untuk baris
-- coverage-nya.
--
-- Yang TIDAK dilakukan: menghapus baris induknya. Rincian dokumen itu sendiri tetap tidak
-- dapat dihapus lewat jalur mana pun di modul ini.
DELETE FROM POOLDATA.LST_DET_TYPE_DOC_BISNIS
 WHERE ID = :1

-- name: detail_business_insert
--
-- STS_WAJIB diisi TEKS 'Ya' atau 'Tidak', bukan angka.
--
-- Keputusan Work Owner 2026-09-23, dan bukti export membenarkannya:
-- `Activity/SetTypePDFAdjustment-Act.xml` membandingkannya HANYA dengan `"Ya"`, sehingga
-- menulis '1' akan membuatnya berhenti mengenali dokumen wajib — tanpa satu pun galat,
-- hanya jenis PDF yang salah pilih.
--
-- Kedua activity lain (`ValidationUploadDocument_act`, `ValidationUploadRegister`)
-- menerima keempat nilai 'Ya', 'Tidak', '1', dan '0', sehingga keduanya tetap benar.
INSERT INTO POOLDATA.LST_DET_TYPE_DOC_BISNIS (ID, DFT_BISNIS_ID, STS_WAJIB, MIN_DOC)
VALUES (:1, :2, :3, :4)

-- name: document_type_choice_list
--
-- Pilihan isian ID Tipe Dokumen.
--
-- Menggantikan autocomplete pada `Section/BrowseListDetailTypeDocument-Section.xml`, yang
-- membaca kelas `ASM-FW-GCNMFW-Int-V_LST_DOC_TYPE` dengan `.ID` sebagai nilai dan
-- `.TYPE_DOCUMENT` sebagai tampilannya.
--
-- HANYA SELECT. Tabel ini dimiliki modul Daftar Tipe Dokumen, MENU_ID 40 (`P-1`).
SELECT ID,
       TYPE_DOCUMENT
  FROM POOLDATA.V_LST_DOC_TYPE
 ORDER BY TYPE_DOCUMENT

-- name: cause_of_loss_choice_list
--
-- Pilihan isian Dokumen kolom ID.
--
-- Menggantikan autocomplete atas `ASM-FW-GCNMFW-Int-V_M_CAUSE_OF_LOSS`, yang menyimpan
-- `.M_COL_ID` ke `TempDTDoc.DOC_COL_ID` dan menampilkan `.COL_DESC`.
--
-- Dibaca dari TABEL DASAR M_CAUSE_OF_LOSS, bukan dari view V_M_CAUSE_OF_LOSS — mengikuti
-- modul Master Penyebab Kerugian yang sudah berjalan, supaya apa yang ditulis modul itu
-- dan apa yang terbaca di sini pasti sama tanpa bergantung pada definisi view yang
-- dimiliki pihak lain.
--
-- HANYA SELECT. Tabel ini dimiliki modul Master Penyebab Kerugian, MENU_ID 20 (`P-1`).
SELECT M_COL_ID,
       COL_DESC
  FROM POOLDATA.M_CAUSE_OF_LOSS
 ORDER BY COL_DESC

-- name: object_document_choice_list
--
-- Pilihan isian Objek Dokumen.
--
-- Menggantikan autocomplete `BrowseVLstDocObj_RD`, yang mengambil ID, OLD_ID, dan
-- KET_DOC_OBJ. OLD_ID tidak dibawa: ia jejak sejarah yang tidak pernah ditampilkan
-- maupun disimpan layar ini.
--
-- HANYA SELECT. Tabel ini dimiliki modul Daftar Objek Dokumen, MENU_ID 43 (`P-1`).
SELECT ID,
       KET_DOC_OBJ
  FROM POOLDATA.V_LST_DOC_OBJ
 ORDER BY KET_DOC_OBJ

-- name: business_choice_list
--
-- Pilihan isian ID Bisnis pada grid.
--
-- Asal kolomnya `RDB List/GetLbuDetType-SQL.xml`, yang membaca `BUSINESS` dengan `ID` dan
-- `NOTE`.
--
-- Diurutkan menurut NOTE, bukan ID: pengguna mencari bisnis dengan namanya, dan urutan
-- kode tidak berarti apa-apa baginya. Pengurutan di basis data, bukan di Go, supaya
-- daftar panjang tidak perlu dimuat seluruhnya ke memori lebih dulu.
--
-- HANYA SELECT. Tabel ini dimiliki GISFW (`D-03`).
SELECT ID,
       NOTE
  FROM POOLDATA.BUSINESS
 ORDER BY NOTE

-- name: detail_check_table
--
-- Memastikan view induk ada dan kolomnya dapat dibaca akun aplikasi, tanpa mengambil satu
-- baris pun. Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip
-- tetapi perbaikannya berbeda jauh: objeknya tidak ada di basis data entitas itu, versus
-- akun aplikasi tidak punya hak baca atasnya.
--
-- Yang TIDAK diperiksa: urutan penerbit ID. Memeriksanya berarti MENGHABISKAN satu nomor
-- — efek samping yang tidak pantas dimiliki mode periksa.
SELECT a.ID,
       a.DOC_TYPE_ID,
       t.TYPE_DOCUMENT,
       a.DETAIL_DOCUMENT,
       a.STS_INSURED,
       a.DOC_COL_ID,
       a.DOC_COL_INFO,
       a.OBJ_DOC,
       a.OBJ_DOC_DESC,
       a.RISK
  FROM POOLDATA.V_LST_DET_TYPE_DOC a
  LEFT JOIN POOLDATA.V_LST_DOC_TYPE t ON t.ID = a.DOC_TYPE_ID
 WHERE 1 = 0

-- name: detail_business_check_table
--
-- Memastikan TABEL ANAK ada dengan bentuk yang BENAR-BENAR dipakai detail_business_list.
--
-- Tabel, bukan view: sejak keputusan Work Owner 2026-09-23 modul ini tidak lagi menyentuh
-- JSON, dan view anaknya hari ini masih `JSON_TABLE`.
SELECT ID,
       DFT_BISNIS_ID,
       STS_WAJIB,
       MIN_DOC
  FROM POOLDATA.LST_DET_TYPE_DOC_BISNIS
 WHERE 1 = 0

-- name: detail_write_check_table
--
-- Memastikan TABEL DASAR induk benar-benar punya kolom yang ditulis modul ini.
--
-- Terpisah dari pemeriksaan view dengan sengaja, dan inilah pemeriksaan terpenting di
-- berkas ini: nama kolomnya DUGAAN, diturunkan dari nama kolom view-nya. Bila tabelnya
-- ternyata masih hanya (ID, JSON_DATA) seperti yang dilakukan procedure lama, kueri ini
-- gagal dengan ORA-00904 — dan kegagalan itu terbaca di mode periksa, sebelum pengguna
-- pertama menekan Simpan.
SELECT ID,
       DOC_TYPE_ID,
       DETAIL_DOCUMENT,
       STS_INSURED,
       DOC_COL_ID,
       DOC_COL_INFO,
       OBJ_DOC,
       OBJ_DOC_DESC,
       RISK,
       TGL_EDIT,
       USER_EDIT
  FROM POOLDATA.LST_DET_TYPE_DOC
 WHERE 1 = 0

-- name: document_type_check_table
SELECT ID,
       TYPE_DOCUMENT
  FROM POOLDATA.V_LST_DOC_TYPE
 WHERE 1 = 0

-- name: cause_of_loss_check_table
SELECT M_COL_ID,
       COL_DESC
  FROM POOLDATA.M_CAUSE_OF_LOSS
 WHERE 1 = 0

-- name: object_document_check_table
SELECT ID,
       KET_DOC_OBJ
  FROM POOLDATA.V_LST_DOC_OBJ
 WHERE 1 = 0

-- name: business_check_table
SELECT ID,
       NOTE
  FROM POOLDATA.BUSINESS
 WHERE 1 = 0
