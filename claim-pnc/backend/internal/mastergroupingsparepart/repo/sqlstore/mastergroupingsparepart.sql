-- Kueri modul Master Grouping Sparepart.
--
-- DUA tabel DITULIS aplikasi ini — POOLDATA.SPAREPART_HE_VIN_KEY dan
-- POOLDATA.SPAREPART_HE_VIN_GROUP — ditambah lima objek yang hanya DIBACA:
-- POOLDATA.PANEL_HE, POOLDATA.LOKASI_PANEL_HE, POOLDATA.SPAREPART_HE, branddetail, dan
-- POOLDATA.M_SPAREPART_HE_VIN_KEY. Seluruhnya milik sistem lama (ADR-0004, penulis tunggal
-- per tabel).
--
-- Kewenangan menulis kedua tabel induk berpindah dari Pega ke Go saat modulnya lulus gerbang
-- 2. Selama Pega masih penulisnya, layar ini harus dijalankan dalam modus baca saja di
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
-- KENAPA MENULIS KE SPAREPART_HE_VIN_KEY, PADAHAL PEGA MENULIS KE M_SPAREPART_HE_VIN_KEY
-- ============================================================================
--
-- Sistem lama memakai DUA penyimpanan untuk satu master, dan itu pola dua-penyimpanan yang
-- `03-CURRENT-ARCHITECTURE.md` §3.2 catat sebagai R-10:
--
--   tulis  Activity/UpdateGroupingSparepartHE_act -> InputSparepart.NAMA_SPART := @GCNM.GetPageJSONString()
--          RDB List/UpdateGroupingSparepartHE-SQL -> POOLDATA.PEGA_M_GROUPING_SPAREPART_HE(...)
--          Database/PEGA_M_GROUPING_SPAREPART_HE.prc:20
--                 -> INSERT INTO POOLDATA.m_sparepart_he_vin_key(ID, JSONDATA)
--
--   baca   RDB List/GetDataMasterGrouping-SQL          -> SPAREPART_HE_VIN_KEY + _VIN_GROUP
--          RDB List/GetNewNoGroup-SQL                  -> sparepart_he_vin_key
--          RDB List/GetNoGroup-SQL                     -> sparepart_he_vin_key
--          RDB List/CountMasterGrupSparepartManager-SQL-> SPAREPART_HE_VIN_KEY + _VIN_GROUP
--
-- Menulisnya lewat procedure dilarang D-02. Yang tersisa adalah dua pilihan, dan hanya satu
-- yang dapat dikerjakan tanpa menebak:
--
--   (a) menulis dokumen JSON ke M_SPAREPART_HE_VIN_KEY.JSONDATA dengan SQL biasa.
--       TIDAK DAPAT DIKERJAKAN: nama kunci JSON-nya diterbitkan fungsi
--       `@GCNM.GetPageJSONString()`, dan `Function/GetPageJSONString-Function.xml` HANYA
--       memuat tanda tangannya — badan fungsinya tidak ikut di export (R-16). Setiap kunci
--       yang ditulis akan menjadi tebakan.
--
--   (b) menulis kolom bernama ke SPAREPART_HE_VIN_KEY dan SPAREPART_HE_VIN_GROUP.
--       DAPAT DIKERJAKAN: keempat belas kolomnya terbaca lengkap dari
--       `GetDataMasterGrouping-SQL`, ditambah APPROVAL dari
--       `CountMasterGrupSparepartManager-SQL` dan NO_RANGKA induk dari penyaring duplikat
--       pada `UpdateGroupingSparepartHE_act`.
--
-- (b) yang dipakai. Perlakuannya sama dengan Master Bengkel, Master Panel, dan Master
-- Sparepart, yang menghadapi keluarga procedure PEGA_M_* yang sama persis dan memutuskan hal
-- yang sama (D-02, D-68): Go menjadi penulis tunggal dan berhenti menulis JSONDATA.
--
-- Satu hal lagi yang TIDAK dibawa dari procedure itu: kontrak galat berbasis string.
-- `PEGA_M_GROUPING_SPAREPART_HE.prc:21` menuliskan `ErrMsg := 'Data Sudah Disimpan dengan ID
-- : '` pada jalur SUKSES, lewat parameter yang namanya berarti pesan galat. Satu kolom
-- keluaran memikul dua arti — persis pola yang `D-68` tolak pada ADD_NEWMASTERVIRTUALACCOUNT.
--
--
-- ============================================================================
-- NO_RANGKA ADA DI KEDUA TABEL, DAN SISTEM LAMA MEMAKAI KEDUANYA
-- ============================================================================
--
-- Kolom bernama sama ada pada induk maupun pendampingnya, dan rule lama menyentuh yang
-- berbeda-beda:
--
--   RDB List/GetDataMasterGrouping-SQL        memilih  B.NO_RANGKA AS "TINGGI"
--   Activity/UpdateGroupingSparepartHE_act    menyaring A.NO_RANGKA  (pencarian duplikat)
--   RDB List/GetNoGroup-SQL                   menyaring A.NO_RANGKA  (pencarian nomor grup)
--
-- Artinya yang DITAMPILKAN daftar dan yang DIPERIKSA keunikannya bukan kolom yang sama.
-- Selama keduanya berisi nilai yang sama, perbedaannya tidak terlihat; begitu berbeda,
-- sebuah baris dapat lolos pemeriksaan duplikat sambil tampil sebagai duplikat di layar.
--
-- Yang dikerjakan di sini: penyimpanan menulis nomor rangka ke KEDUANYA dengan nilai yang
-- sama, dalam satu transaksi. Dengan begitu ketiga rule di atas selalu sepakat, dan
-- perbedaannya tidak dapat lahir lewat modul ini.
--
-- Baris LAMA yang sudah terlanjur berbeda tidak diperbaiki modul ini — jumlahnya dilaporkan
-- `claimpnc -periksa` lewat grouping_count_chassis_mismatch supaya besarnya terlihat lebih
-- dulu.
--
--
-- ============================================================================
-- TABEL PENDAMPING TIDAK PERNAH DIHAPUS BARISNYA
-- ============================================================================
--
-- Master Panel mengganti baris anaknya dengan hapus-lalu-sisip-ulang, karena baris anaknya
-- tidak punya kunci sendiri dan jumlahnya berubah-ubah.
--
-- Di sini tidak: SPAREPART_HE_VIN_GROUP berhubungan SATU-LAWAN-SATU dengan induknya
-- (`GetDataMasterGrouping` menggabungkannya dengan `A.ID=B.ID` tanpa agregasi apa pun),
-- sehingga barisnya dapat di-UPDATE di tempat. Penyimpanan karena itu mencoba UPDATE lebih
-- dulu dan hanya menyisipkan bila belum ada barisnya.
--
-- Itu sejalan dengan `D-66`: tidak ada satu pun DELETE terhadap data bernilai bisnis di
-- seluruh modul ini.
--
--
-- CATATAN TRIM pada setiap penyaring kunci. Tipe kolom belum diketahui (R-08). Bila ID
-- bertipe CHAR berlebar tetap, nilainya dipadatkan spasi tanpa tanda apa pun; Oracle
-- membandingkan CHAR dengan CHAR secara blank-padded, sehingga kueri lama yang MERANGKAI
-- nilainya tetap cocok. Parameter binding bertipe VARCHAR2, dan perbandingan CHAR dengan
-- VARCHAR2 memakai non-padded comparison: "ABC " tidak sama dengan "ABC", dan barisnya tidak
-- ketemu. Menyalin `= :1` apa adanya karena itu justru MENGUBAH perilaku. TRIM benar untuk
-- kedua kemungkinan tipe. Biayanya index atas kolom itu tidak terpakai; dapat diterima pada
-- tabel master berbaris sedikit, dan TIDAK boleh ditiru pada tabel besar.


-- name: grouping_list
--
-- Asal: RDB List/GetDataMasterGrouping-SQL.xml, dipakai ketiga tab
-- `Section/PNCMasterGroupingSparepartHE-Section.xml` yang hanya berbeda pada nilai APPROVAL-
-- nya — `Activity/GetDataMasterGrouping-Act.xml` merangkainya sebagai
-- `"AND A.APPROVAL=" + "'" + PARAM.Approve + "'"`.
--
-- Keenam belas kolomnya disebut pada urutan yang SAMA dengan grouping_get dan
-- grouping_find_by_key — satu fungsi scanRow membaca ketiganya berdasarkan POSISI, dan satu
-- kolom yang bergeser akan menaruh nama panel ke kolom nomor rangka tanpa satu pun galat.
-- `TestReaderQueriesShareColumnOrder` yang menjaganya.
--
-- ALIASNYA TIDAK DIBAWA. Kueri lama menamai `A.NAMA_PANEL` sebagai `"PANJANG"`,
-- `A.SISI_PANEL` sebagai `"LEBAR"`, `B.NO_RANGKA` sebagai `"TINGGI"`, dan `A.CATATAN`
-- sebagai `"MAX_STOCK"` — nama properti Master Sparepart yang dipinjam karena layar ini
-- salinan layar itu. Di sini kolomnya dibaca dengan nama aslinya.
--
-- INNER JOIN dipertahankan, bukan diganti LEFT JOIN. Baris induk yang tidak punya pendamping
-- TIDAK MUNCUL di layar lama, dan menampilkannya di sini akan memunculkan baris yang selama
-- ini tidak terlihat siapa pun — perubahan perilaku yang tidak diminta. Jumlah baris seperti
-- itu dilaporkan `claimpnc -periksa` lewat grouping_count_without_group.
--
-- ORDER BY DITAMBAHKAN. Kueri lama tidak menyebut satu pun kolom pengurut, sehingga Pega
-- menerima baris dalam urutan apa pun yang dikembalikan Oracle. Urutan yang tidak ditentukan
-- tidak dapat dipakai layar: daftar yang berubah urutan antar pemuatan membuat baris melompat
-- di bawah kursor pengguna.
--
-- `A.ID` yang dipilih, menaik, karena ia kunci yang diterbitkan pencacah — sehingga urutannya
-- sama dengan urutan penerbitan.
SELECT A.ID,
       A.NO_PART,
       A.NAMA_PART,
       A.KODE_PART,
       A.KATEGORI_SPART,
       A.TIPE_SPART,
       A.PROD_DATE,
       A.ID_PANEL,
       A.NAMA_PANEL,
       A.SISI_PANEL,
       B.NO_RANGKA,
       B.TIPE,
       A.GROUPING_DGN_RANGKA,
       A.NO_GROUP_RANGKA,
       A.CATATAN,
       A.APPROVAL
  FROM POOLDATA.SPAREPART_HE_VIN_KEY A,
       POOLDATA.SPAREPART_HE_VIN_GROUP B
 WHERE A.ID = B.ID
   AND TRIM(A.APPROVAL) = :1
 ORDER BY A.ID

-- name: grouping_list_search
--
-- Sama dengan grouping_list, ditambah penyaring kata kunci.
--
-- KUERI TERSENDIRI, bukan satu kueri yang klausanya ditempel. Sistem lama menempuh cara yang
-- kedua justru di modul ini — `Activity/GetDataMasterGrouping-Act.xml` merangkai penyaringnya
-- sebagai teks lalu menyisipkannya lewat `{ASIS:TempData.APPROVAL}` dan `{ASIS:TempData.ID}`
-- — dan `08-TECHNICAL-STRATEGY.md` §4.3 melarangnya tanpa perkecualian.
--
-- EMPAT kolom dicari sekaligus: nomor sparepart, nama sparepart, nama panel, DAN nomor
-- rangka. Keempatnya cara orang mencari baris di layar ini; nomor rangka yang paling sering,
-- karena pertanyaan sehari-harinya adalah "panel apa saja yang terpasang di rangka ini".
--
-- Nomor rangkanya dicari pada `B.NO_RANGKA`, kolom yang sama dengan yang DITAMPILKAN —
-- mencari pada kolom yang tidak terlihat akan menghasilkan baris yang kata kuncinya tidak
-- ada di mana pun di layar.
--
-- UPPER dipasang pada KOLOMNYA juga, bukan hanya pada kata kuncinya: keempatnya tersimpan
-- dengan besar-kecil huruf apa adanya, dan pencarian yang hanya meng-uppercase kata kunci
-- tidak akan pernah menemukan baris yang isinya huruf kecil.
--
-- `ESCAPE '\'` disebut eksplisit karena Oracle TIDAK punya karakter pelolos bawaan pada LIKE.
-- PostgreSQL memakai backslash sebagai bawaan; menyebutkannya eksplisit membuat kedua basis
-- data berperilaku sama (D-20).
--
-- Parameter kata kuncinya disebut EMPAT KALI dengan nilai yang sama, bukan sekali. Oracle
-- memperlakukan setiap penanda posisi sebagai bind terpisah, sehingga menyebut `:2` empat
-- kali akan menuntut driver mengirim satu nilai untuk empat posisi — perilaku yang berbeda
-- antar driver.
SELECT A.ID,
       A.NO_PART,
       A.NAMA_PART,
       A.KODE_PART,
       A.KATEGORI_SPART,
       A.TIPE_SPART,
       A.PROD_DATE,
       A.ID_PANEL,
       A.NAMA_PANEL,
       A.SISI_PANEL,
       B.NO_RANGKA,
       B.TIPE,
       A.GROUPING_DGN_RANGKA,
       A.NO_GROUP_RANGKA,
       A.CATATAN,
       A.APPROVAL
  FROM POOLDATA.SPAREPART_HE_VIN_KEY A,
       POOLDATA.SPAREPART_HE_VIN_GROUP B
 WHERE A.ID = B.ID
   AND TRIM(A.APPROVAL) = :1
   AND (UPPER(A.NO_PART) LIKE :2 ESCAPE '\'
        OR UPPER(A.NAMA_PART) LIKE :3 ESCAPE '\'
        OR UPPER(A.NAMA_PANEL) LIKE :4 ESCAPE '\'
        OR UPPER(B.NO_RANGKA) LIKE :5 ESCAPE '\')
 ORDER BY A.ID

-- name: grouping_get
--
-- Kolomnya sama dan pada urutan yang sama dengan grouping_list.
SELECT A.ID,
       A.NO_PART,
       A.NAMA_PART,
       A.KODE_PART,
       A.KATEGORI_SPART,
       A.TIPE_SPART,
       A.PROD_DATE,
       A.ID_PANEL,
       A.NAMA_PANEL,
       A.SISI_PANEL,
       B.NO_RANGKA,
       B.TIPE,
       A.GROUPING_DGN_RANGKA,
       A.NO_GROUP_RANGKA,
       A.CATATAN,
       A.APPROVAL
  FROM POOLDATA.SPAREPART_HE_VIN_KEY A,
       POOLDATA.SPAREPART_HE_VIN_GROUP B
 WHERE A.ID = B.ID
   AND TRIM(A.ID) = :1

-- name: grouping_find_by_key
--
-- Padanan pencarian duplikat pada `Activity/UpdateGroupingSparepartHE_act-Act.xml` langkah 3,
-- yang merangkai penyaringnya sebagai teks SQL:
--
--   AND A.NO_PART= '<nomor>'AND A.NAMA_PANEL ='<panel>'AND A.NO_RANGKA ='<rangka>' AND A.SISI_PANEL= '<sisi>'
--
-- PERHATIKAN NOMOR RANGKANYA: `A.NO_RANGKA`, kolom pada tabel INDUK — bukan `B.NO_RANGKA`
-- yang ditampilkan daftar. Ditiru apa adanya; lihat banner berkas ini.
--
-- Yang berubah selain itu hanyalah cara nilainya sampai — parameter binding menggantikan
-- perangkaian teks — dan penambahan UPPER serta TRIM pada keempat kolomnya. Kueri lama
-- membandingkannya apa adanya, sehingga nomor sparepart bertuliskan huruf kecil lolos sebagai
-- baris yang berbeda. Itu SELISIH YANG DIRENCANAKAN: ia menolak baris yang di sistem lama
-- diterima, dan baris lama tetap dibaca apa adanya.
--
-- Kolomnya sama dan pada urutan yang sama dengan grouping_list.
SELECT A.ID,
       A.NO_PART,
       A.NAMA_PART,
       A.KODE_PART,
       A.KATEGORI_SPART,
       A.TIPE_SPART,
       A.PROD_DATE,
       A.ID_PANEL,
       A.NAMA_PANEL,
       A.SISI_PANEL,
       B.NO_RANGKA,
       B.TIPE,
       A.GROUPING_DGN_RANGKA,
       A.NO_GROUP_RANGKA,
       A.CATATAN,
       A.APPROVAL
  FROM POOLDATA.SPAREPART_HE_VIN_KEY A,
       POOLDATA.SPAREPART_HE_VIN_GROUP B
 WHERE A.ID = B.ID
   AND UPPER(TRIM(A.NO_PART)) = :1
   AND UPPER(TRIM(A.NAMA_PANEL)) = :2
   AND UPPER(TRIM(A.NO_RANGKA)) = :3
   AND UPPER(TRIM(A.SISI_PANEL)) = :4

-- name: grouping_lock_by_key
--
-- Dipakai di dalam transaksi penambahan. FOR UPDATE mengunci baris yang SUDAH ADA sehingga
-- dua penambahan atas kunci yang sama tidak dapat berjalan berdampingan.
--
-- TANPA join ke tabel pendamping, berbeda dari grouping_find_by_key: `FOR UPDATE` atas join
-- mengunci baris di KEDUA tabel, dan mengunci pendamping yang tidak akan disentuh
-- pemeriksaan ini hanya memperlebar apa yang ditahan transaksinya. Keempat kolom kuncinya
-- seluruhnya ada di tabel induk, sehingga join memang tidak diperlukan.
--
-- SEBERAPA JAUH LUBANGNYA TERTUTUP: tidak sepenuhnya. FOR UPDATE tidak dapat mengunci baris
-- yang belum ada, sehingga dua penambahan yang sama-sama menemukan nol baris tetap lolos
-- berdampingan. Yang benar-benar menutupnya adalah constraint unik pada keempat kolom, dan
-- itu menunggu DDL (R-08) serta prosedur perubahan skema (D-63).
SELECT ID
  FROM POOLDATA.SPAREPART_HE_VIN_KEY
 WHERE UPPER(TRIM(NO_PART)) = :1
   AND UPPER(TRIM(NAMA_PANEL)) = :2
   AND UPPER(TRIM(NO_RANGKA)) = :3
   AND UPPER(TRIM(SISI_PANEL)) = :4
 FOR UPDATE

-- name: grouping_insert
--
-- Kelima belas kolomnya pada urutan yang sama dengan insertArguments. Urutan itu WAJIB sama;
-- `TestInsertArgumentsMatchColumnOrder` yang menjaganya.
--
-- NO_RANGKA ikut ditulis di sini, bukan hanya di tabel pendamping; lihat banner berkas ini.
INSERT INTO POOLDATA.SPAREPART_HE_VIN_KEY
       (ID, NO_PART, NAMA_PART, KODE_PART, KATEGORI_SPART,
        TIPE_SPART, PROD_DATE, ID_PANEL, NAMA_PANEL, SISI_PANEL,
        NO_RANGKA, GROUPING_DGN_RANGKA, NO_GROUP_RANGKA, CATATAN, APPROVAL)
VALUES (:1, :2, :3, :4, :5,
        :6, :7, :8, :9, :10,
        :11, :12, :13, :14, :15)

-- name: grouping_update
--
-- ID TIDAK disebut sebagai kolom yang ditulis — ia penyaring WHERE, dan berada di posisi
-- terakhir pada updateArguments. Kunci baris tidak pernah berpindah.
UPDATE POOLDATA.SPAREPART_HE_VIN_KEY
   SET NO_PART = :1,
       NAMA_PART = :2,
       KODE_PART = :3,
       KATEGORI_SPART = :4,
       TIPE_SPART = :5,
       PROD_DATE = :6,
       ID_PANEL = :7,
       NAMA_PANEL = :8,
       SISI_PANEL = :9,
       NO_RANGKA = :10,
       GROUPING_DGN_RANGKA = :11,
       NO_GROUP_RANGKA = :12,
       CATATAN = :13,
       APPROVAL = :14
 WHERE TRIM(ID) = :15

-- name: grouping_group_insert
--
-- Baris pendamping. Ketiga kolomnya adalah seluruh isi tabel itu sejauh yang terbaca dari
-- `GetDataMasterGrouping-SQL`.
INSERT INTO POOLDATA.SPAREPART_HE_VIN_GROUP
       (ID, NO_RANGKA, TIPE)
VALUES (:1, :2, :3)

-- name: grouping_group_update
--
-- Dicoba LEBIH DULU pada penyimpanan; grouping_group_insert hanya dijalankan bila ini tidak
-- menyentuh satu baris pun. Lihat banner berkas ini — tidak ada DELETE di seluruh modul ini
-- (D-66).
UPDATE POOLDATA.SPAREPART_HE_VIN_GROUP
   SET NO_RANGKA = :1,
       TIPE = :2
 WHERE TRIM(ID) = :3

-- name: grouping_set_status
--
-- Satu pernyataan per baris, bukan satu pernyataan dengan daftar kunci yang panjangnya
-- berubah-ubah. Daftar `IN (:1, :2, …)` yang panjangnya mengikuti jumlah baris menghasilkan
-- teks SQL yang berbeda setiap kali dipanggil — setiap bentuk menempati satu slot pada shared
-- pool Oracle, dan tabel master yang diputuskan borongan akan menghasilkan puluhan bentuk
-- berbeda dari satu operasi yang sama.
--
-- `AND TRIM(APPROVAL) <> :1` membuat baris yang sudah berstatus itu TIDAK terhitung sebagai
-- berubah — itulah yang membuat jumlah yang dilaporkan ke layar bermakna.
--
-- HANYA APPROVAL yang disentuh. Sistem lama menyimpan ulang SELURUH kolomnya saat menyetujui,
-- karena layar persetujuannya memanggil activity penyimpanan yang sama dengan
-- `Param.Approval` yang berbeda. Tidak ditiru; alasannya pada usecase.Service.Decide.
UPDATE POOLDATA.SPAREPART_HE_VIN_KEY
   SET APPROVAL = :1
 WHERE TRIM(ID) = :2
   AND TRIM(APPROVAL) <> :1

-- name: grouping_max_id
--
-- Padanan `Database/PEGA_M_GROUPING_SPAREPART_HE.prc:11`:
--
--   SELECT NVL(MAX(ID),0) + 1 INTO id_count from POOLDATA.m_sparepart_he_vin_key;
--
-- Dua hal berbeda, dan keduanya disengaja.
--
-- PERTAMA, tabelnya. Procedure membacanya dari penyimpanan JSON yang TIDAK LAGI DITULIS
-- aplikasi ini; membaca MAX dari sana akan membuat nomornya berhenti bertambah, dan setiap
-- penambahan berikutnya menerbitkan ID yang sama. Yang dibaca di sini adalah tabel yang
-- benar-benar ditulis. Penyimpanan JSON tetap ikut dibaca lewat grouping_max_id_mirror; lihat
-- NextID pada adapter.
--
-- KEDUA, penambahannya dilakukan di Go, bukan di SQL. `MAX(ID) + 1` di dalam kueri menuntut
-- kolomnya bertipe angka; bila ia ternyata teks, Oracle memaksa konversi implisit yang gagal
-- pada baris yang isinya bukan angka. Membaca MAX-nya saja lalu menambahkannya di Go membuat
-- kegagalan itu terbaca sebagai kalimat, bukan sebagai ORA-01722.
SELECT MAX(ID)
  FROM POOLDATA.SPAREPART_HE_VIN_KEY

-- name: grouping_max_id_mirror
--
-- MAX(ID) pada penyimpanan JSON milik Pega.
--
-- Ia dibaca BERSAMA grouping_max_id, dan yang dipakai adalah yang lebih besar. Alasannya
-- masa paralel (D-21, ADR-0004): selama Pega masih menulis, ID yang diterbitkannya hanya
-- muncul di sana. Mengabaikannya berarti kedua sistem dapat menerbitkan ID yang sama pada
-- hari yang sama.
SELECT MAX(ID)
  FROM POOLDATA.M_SPAREPART_HE_VIN_KEY

-- name: grouping_group_numbers
--
-- Seluruh nomor grup yang sudah dipakai, tanpa duplikat.
--
-- # Kenapa BUKAN MAX(), padahal kueri lamanya memakainya
--
-- `RDB List/GetNewNoGroup-SQL.xml`:
--
--   select to_number(nvl(max(NO_GROUP_RANGKA),0)+1) AS "ID" from POOLDATA.sparepart_he_vin_key
--
-- `MAX` atas kolom TEKS adalah maksimum LEKSIKOGRAFIS, dan nomor grup ditulis berawalan nol
-- oleh `UpdateGroupingSparepartHE_act` langkah 7 (`"000" + nomor`). Akibatnya "0009" lebih
-- besar daripada "00010" — sehingga setelah grup kesembilan, kueri itu SELALU mengembalikan
-- 10 dan setiap grup baru menerima nomor yang sama.
--
-- Itu bukan ketidakrapian melainkan cacat yang menghasilkan nomor grup ganda, dan meniru
-- cacat yang menghasilkan kunci ganda bukan kesetaraan perilaku melainkan kerusakan data.
-- Nomor terbesarnya karena itu dihitung SECARA ANGKA di Go; lihat NextGroupNumber pada
-- adapter.
--
-- Nilai yang tidak dapat diurai sebagai angka dilewati di sana, bukan menggagalkan
-- penambahan: satu baris warisan yang isinya rusak tidak boleh menghentikan seluruh layar.
--
-- DISTINCT dipakai karena satu grup dipakai banyak baris — tanpa itu, sebuah grup berisi
-- lima puluh panel akan terbaca lima puluh kali.
SELECT DISTINCT NO_GROUP_RANGKA
  FROM POOLDATA.SPAREPART_HE_VIN_KEY
 WHERE NO_GROUP_RANGKA IS NOT NULL

-- name: grouping_group_by_chassis
--
-- Padanan `RDB List/GetNoGroup-SQL.xml`:
--
--   select TO_NUMBER(NO_GROUP_RANGKA) AS "ID" from POOLDATA.sparepart_he_vin_key
--    where NO_RANGKA = {TempSparepart.BERAT}
--
-- `TO_NUMBER` TIDAK dibawa: ia membuang awalan nol, sehingga baris yang BERGABUNG ke sebuah
-- grup menyimpan bentuk teks yang berbeda dari baris yang MEMBUKA grup itu. Alasan lengkapnya
-- pada usecase.Service.resolveGroupNumber, beserta catatan bahwa ia menuntut persetujuan
-- `D-54`.
--
-- UPPER dan TRIM ditambahkan pada penyaringnya dengan alasan yang sama seperti
-- grouping_find_by_key.
--
-- ORDER BY DITAMBAHKAN dan hasilnya diambil satu. Kueri lama tidak menyebutnya sementara
-- pemanggilnya hanya memakai `pxResults(1)`; bila sebuah nomor rangka terdaftar pada dua
-- nomor grup — keadaan yang sah menurut struktur tabelnya — baris mana yang menang menjadi
-- bergantung pada urutan yang dikembalikan Oracle. Di sini yang menang selalu nomor grup
-- terkecil, sehingga hasilnya sama setiap kali dipanggil.
SELECT NO_GROUP_RANGKA
  FROM POOLDATA.SPAREPART_HE_VIN_KEY
 WHERE UPPER(TRIM(NO_RANGKA)) = :1
   AND NO_GROUP_RANGKA IS NOT NULL
 ORDER BY NO_GROUP_RANGKA
 FETCH FIRST 1 ROWS ONLY

-- name: grouping_panel_list
--
-- Padanan autocomplete "Nama Panel" pada
-- `Section/MasterGroupingSparepartHEApproval-Section.xml`, yang membaca
-- `Report Definition/BrowseMasterPanel_HE_RD-RD.xml` dengan parameter `APPROVAL = "1"` lalu
-- menampilkan `.NAME` dan menyalin `.ID_PANEL` ke isian tersembunyi.
--
-- `pyMaxRecords=500` pada report definition itu TIDAK direplikasi sebagai pemotongan diam-
-- diam; batasnya dipasang di adapter dan jumlahnya dilaporkan. Lihat MaxLookupRows.
--
-- ORDER BY DITAMBAHKAN. Nama yang dipilih, bukan ID, karena inilah daftar yang dipindai mata
-- pengguna.
SELECT ID_PANEL,
       NAME
  FROM POOLDATA.PANEL_HE
 WHERE TRIM(APPROVAL) = :1
 ORDER BY NAME

-- name: grouping_side_list
--
-- Padanan `RDB List/GetDataSisiPanel-SQL.xml`:
--
--   select sisi_panel AS "NAME" from pooldata.lokasi_panel_he
--    where id_panel = {TempSparepart.MIN_STOCK} and nama = {TempSparepart.PANJANG}
--
-- KEDUA penyaringnya dipertahankan. Ketidakpastian soal arti kolom `NAMA` — nama lokasi atau
-- nama panel — dinyatakan pada doc comment LookupRepo.ListSides, dan dijawab empiris oleh
-- grouping_count_child_name_as_panel serta grouping_count_child_name_as_location di bawah.
--
-- DISTINCT dipakai karena satu panel dapat punya beberapa lokasi bersisi sama, dan yang
-- dibutuhkan layar adalah daftar SISI — bukan daftar lokasi.
--
-- ORDER BY DITAMBAHKAN supaya urutan pilihan tidak berubah antar pemuatan.
SELECT DISTINCT SISI_PANEL
  FROM POOLDATA.LOKASI_PANEL_HE
 WHERE TRIM(ID_PANEL) = :1
   AND UPPER(TRIM(NAMA)) = :2
 ORDER BY SISI_PANEL

-- name: grouping_vehicle_type_list
--
-- Padanan `RDB List/BrowseTypeHE_Sql-SQL.xml`, lewat data page `D_TypeHEList`:
--
--   select id as "BANK_ID", TYPENAME as "NAMA_BANK"
--     from branddetail where type = 'ANEKA' and ACTIVESTATUS=1
--
-- Alias `"BANK_ID"` dan `"NAMA_BANK"` TIDAK dibawa; lihat catatan pada
-- mastergroupingsparepart.VehicleType.
--
-- NAMA TABELNYA TANPA SKEMA, persis seperti kueri aslinya. Dengan begitu yang terpakai adalah
-- skema bawaan akun koneksi — sama seperti di Pega hari ini. Melengkapinya dengan POOLDATA
-- akan menjadi tebakan: tidak satu pun rule di export menyebut skema tabel itu.
--
-- `ACTIVESTATUS = 1` DIBIARKAN sebagai literal angka, bukan diikat sebagai teks. Ia konstanta
-- di dalam kueri, bukan masukan pengguna, sehingga tidak ada celah yang ditutup dengan
-- mengikatnya — sedangkan mengubahnya menjadi teks akan memaksa konversi yang perilakunya
-- berbeda antar basis data bila kolomnya ternyata bertipe angka.
--
-- `TYPE` diikat karena ia yang membedakan lini bisnis, dan nilainya berada di adapter supaya
-- terbaca sebagai keputusan alih-alih tersembunyi di dalam teks SQL.
--
-- ORDER BY DITAMBAHKAN; kueri lama tidak menyebut satu pun.
SELECT ID,
       TYPENAME
  FROM branddetail
 WHERE TRIM(TYPE) = :1
   AND ACTIVESTATUS = 1
 ORDER BY TYPENAME

-- name: grouping_part_find
--
-- Padanan `RDB List/GetDataSparepart-SQL.xml`:
--
--   select no_spart, nama_spart, kategori_spart, tipe_spart, PROD_DATE, KODE_SPART
--     from pooldata.sparepart_he where no_spart = {TempSparepart.NO_SPART}
--
-- TANPA penyaring APPROVAL, persis seperti kueri aslinya. Menambahkannya akan menolak nomor
-- sparepart yang hari ini diterima — termasuk sparepart yang sedang menunggu persetujuan.
--
-- UPPER dan TRIM DITAMBAHKAN pada penyaringnya, mengikuti modul Master Sparepart yang sudah
-- mengambil keputusan yang sama atas kolom yang sama.
--
-- Bila sebuah nomor dipakai lebih dari satu baris — keadaan yang mungkin karena tabelnya
-- tidak punya constraint unik yang diketahui (R-08) — yang menang adalah yang ID-nya terkecil,
-- sehingga hasilnya sama setiap kali dipanggil. Kueri lama menyerahkannya pada urutan yang
-- dikembalikan Oracle.
SELECT NO_SPART,
       NAMA_SPART,
       KATEGORI_SPART,
       TIPE_SPART,
       KODE_SPART,
       PROD_DATE
  FROM POOLDATA.SPAREPART_HE
 WHERE UPPER(TRIM(NO_SPART)) = :1
 ORDER BY ID
 FETCH FIRST 1 ROWS ONLY

-- name: grouping_count_by_status
--
-- Padanan `RDB List/CountMasterGrupSparepartManager-SQL.xml`:
--
--   SELECT COUNT(A.ID) AS "City" FROM POOLDATA.SPAREPART_HE_VIN_KEY A,
--          POOLDATA.SPAREPART_HE_VIN_GROUP B WHERE A.ID=B.ID AND A.APPROVAL=0
--
-- PERHATIKAN PEMBANDINGNYA pada kueri lama: angka `0`, bukan teks `'0'` — sementara ketiga
-- section tab membandingkannya sebagai teks. Bila kolomnya bertipe teks, perbandingan dengan
-- angka memaksa konversi implisit yang perilakunya berbeda antar basis data, dan pada baris
-- yang isinya bukan angka ia melempar ORA-01722.
--
-- Di sini ia selalu teks, sama dengan seluruh kueri lain di berkas ini.
--
-- Alias `"City"` juga tidak dibawa — satu lagi nama yang tidak ada hubungannya dengan isinya.
SELECT COUNT(A.ID)
  FROM POOLDATA.SPAREPART_HE_VIN_KEY A,
       POOLDATA.SPAREPART_HE_VIN_GROUP B
 WHERE A.ID = B.ID
   AND TRIM(A.APPROVAL) = :1

-- name: grouping_count_all
--
-- Seluruh baris INDUK, tanpa join. Perbandingannya dengan grouping_count_by_status yang
-- dijumlahkan memperlihatkan berapa baris yang hilang karena join — lihat
-- grouping_count_without_group.
SELECT COUNT(ID)
  FROM POOLDATA.SPAREPART_HE_VIN_KEY

-- name: grouping_count_without_group
--
-- Baris induk yang TIDAK punya pendamping di SPAREPART_HE_VIN_GROUP.
--
-- Seluruhnya TIDAK MUNCUL di layar, baik di Pega maupun di sini, karena kedua sistem
-- menggabungkannya dengan INNER JOIN. Jumlahnya menentukan apakah daftar yang tampil memang
-- seluruh isi tabelnya — dan bila tidak nol, ia menjelaskan selisih yang akan muncul pada uji
-- kesetaraan sebelum ada yang mengiranya cacat.
SELECT COUNT(A.ID)
  FROM POOLDATA.SPAREPART_HE_VIN_KEY A
 WHERE NOT EXISTS (
       SELECT 1
         FROM POOLDATA.SPAREPART_HE_VIN_GROUP B
        WHERE TRIM(B.ID) = TRIM(A.ID))

-- name: grouping_count_chassis_mismatch
--
-- Baris yang NO_RANGKA induknya BERBEDA dari NO_RANGKA pendampingnya.
--
-- Sistem lama menampilkan yang satu dan memeriksa keunikan atas yang lain; lihat banner
-- berkas ini. Selama nilainya sama, perbedaannya tidak terlihat. Angka ini yang mengukurnya.
--
-- Baris yang KEDUANYA kosong tidak terhitung: keduanya sama, dan sama-sama kosong.
SELECT COUNT(A.ID)
  FROM POOLDATA.SPAREPART_HE_VIN_KEY A,
       POOLDATA.SPAREPART_HE_VIN_GROUP B
 WHERE A.ID = B.ID
   AND (UPPER(TRIM(A.NO_RANGKA)) <> UPPER(TRIM(B.NO_RANGKA))
        OR (A.NO_RANGKA IS NULL AND B.NO_RANGKA IS NOT NULL)
        OR (A.NO_RANGKA IS NOT NULL AND B.NO_RANGKA IS NULL))

-- name: grouping_count_child_name_as_panel
--
-- Baris LOKASI_PANEL_HE yang kolom NAMA-nya sama dengan NAME panel induknya.
--
-- Ia separuh dari jawaban atas pertanyaan yang belum tertutup pada
-- LookupRepo.ListSides: apakah `NAMA` berisi nama PANEL atau nama LOKASI. Separuh lainnya
-- ada di grouping_count_child_name_as_location, dan `claimpnc -periksa` menyandingkan
-- keduanya.
--
-- Bila yang ini yang mendekati jumlah seluruh barisnya, layar Sisi modul ini akan bekerja dan
-- asumsi penulisan Master Panel perlu ditinjau. Bila yang satunya, sebaliknya.
SELECT COUNT(L.ID_PANEL)
  FROM POOLDATA.LOKASI_PANEL_HE L,
       POOLDATA.PANEL_HE P
 WHERE TRIM(L.ID_PANEL) = TRIM(P.ID_PANEL)
   AND UPPER(TRIM(L.NAMA)) = UPPER(TRIM(P.NAME))

-- name: grouping_count_child_name_as_location
--
-- Baris LOKASI_PANEL_HE yang kolom NAMA-nya sama dengan LOKASI_PANEL-nya sendiri.
SELECT COUNT(ID_PANEL)
  FROM POOLDATA.LOKASI_PANEL_HE
 WHERE UPPER(TRIM(NAMA)) = UPPER(TRIM(LOKASI_PANEL))

-- name: grouping_count_child_rows
--
-- Seluruh baris LOKASI_PANEL_HE, sebagai penyebut kedua pencacah di atas.
SELECT COUNT(ID_PANEL)
  FROM POOLDATA.LOKASI_PANEL_HE

-- name: grouping_count_orphan_part
--
-- Baris yang NO_PART-nya tidak ada di POOLDATA.SPAREPART_HE.
--
-- Ia tidak dapat lahir lewat modul ini — penyimpanan menolak nomor yang tidak ketemu — tetapi
-- dapat sudah ada di data warisan, dan jumlahnya menentukan berapa baris lama yang TIDAK
-- DAPAT DISIMPAN ULANG tanpa lebih dulu memperbaiki nomornya.
--
-- Baris yang kolomnya kosong ikut terhitung: nomor sparepart wajib diisi di modul ini,
-- sehingga baris tanpa nomor sama-sama tidak dapat disimpan ulang.
SELECT COUNT(A.ID)
  FROM POOLDATA.SPAREPART_HE_VIN_KEY A
 WHERE NOT EXISTS (
       SELECT 1
         FROM POOLDATA.SPAREPART_HE S
        WHERE UPPER(TRIM(S.NO_SPART)) = UPPER(TRIM(A.NO_PART)))

-- name: grouping_count_orphan_panel
--
-- Baris yang NAMA_PANEL-nya tidak ada di POOLDATA.PANEL_HE.
--
-- Alasannya sama: daftar pilihan hanya menawarkan panel yang disetujui, sehingga baris lama
-- yang menunjuk panel lain tidak dapat disimpan ulang tanpa memilih panel yang lain.
--
-- Perhatikan bahwa penyaring di sini TIDAK memandang APPROVAL, sementara daftar pilihan
-- memandangnya. Selisih keduanya justru yang ingin diketahui: baris yang panelnya ADA tetapi
-- BELUM DISETUJUI tidak terhitung di sini, dan tetap tidak dapat dipilih ulang di layar.
SELECT COUNT(A.ID)
  FROM POOLDATA.SPAREPART_HE_VIN_KEY A
 WHERE A.NAMA_PANEL IS NOT NULL
   AND TRIM(A.NAMA_PANEL) <> ''
   AND NOT EXISTS (
       SELECT 1
         FROM POOLDATA.PANEL_HE P
        WHERE UPPER(TRIM(P.NAME)) = UPPER(TRIM(A.NAMA_PANEL)))

-- name: grouping_check_table
--
-- Tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi. Yang
-- dibuktikannya hanyalah tabelnya ada dan kelima belas kolomnya dapat dibaca akun aplikasi.
SELECT ID,
       NO_PART,
       NAMA_PART,
       KODE_PART,
       KATEGORI_SPART,
       TIPE_SPART,
       PROD_DATE,
       ID_PANEL,
       NAMA_PANEL,
       SISI_PANEL,
       NO_RANGKA,
       GROUPING_DGN_RANGKA,
       NO_GROUP_RANGKA,
       CATATAN,
       APPROVAL
  FROM POOLDATA.SPAREPART_HE_VIN_KEY
 WHERE 1 = 0

-- name: grouping_check_group_table
SELECT ID,
       NO_RANGKA,
       TIPE
  FROM POOLDATA.SPAREPART_HE_VIN_GROUP
 WHERE 1 = 0

-- name: grouping_check_vehicle_type_table
--
-- Tabel tanpa skema, persis seperti kueri aslinya; lihat grouping_vehicle_type_list.
SELECT ID,
       TYPENAME,
       TYPE,
       ACTIVESTATUS
  FROM branddetail
 WHERE 1 = 0

-- name: grouping_check_child_name_column
--
-- Kolom NAMA pada tabel anak Master Panel — satu-satunya kolom di luar modul ini yang
-- keberadaannya menentukan apakah daftar Sisi dapat terisi sama sekali.
SELECT ID_PANEL,
       LOKASI_PANEL,
       SISI_PANEL,
       NAMA
  FROM POOLDATA.LOKASI_PANEL_HE
 WHERE 1 = 0

-- name: grouping_check_json_mirror
--
-- Penyimpanan JSON milik Pega. Perbandingan jumlah barisnya dengan SPAREPART_HE_VIN_KEY
-- adalah cara termurah mengetahui apakah keduanya satu sumber; lihat banner berkas ini.
SELECT ID
  FROM POOLDATA.M_SPAREPART_HE_VIN_KEY
 WHERE 1 = 0

-- name: grouping_count_json_mirror
SELECT COUNT(ID)
  FROM POOLDATA.M_SPAREPART_HE_VIN_KEY
