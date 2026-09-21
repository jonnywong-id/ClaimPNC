-- Kueri modul Master Bengkel.
--
-- Satu tabel DITULIS aplikasi ini — POOLDATA.BENGKEL_HE — dan empat tabel lain hanya
-- DIBACA: GENERAL.LST_USER_ASURANSI, LST_DET_CABANG, CITY, GENERAL.LST_BANK_GROUP,
-- ditambah POOLDATA.M_SITE_DATABASE dan sequence BENGKEL_HE_SEQ untuk penomoran.
-- Seluruhnya milik sistem lain (ADR-0004, penulis tunggal per tabel); tidak ada satu pun
-- pernyataan tulis terhadapnya di berkas ini.
--
-- Kewenangan menulis BENGKEL_HE berpindah dari Pega ke Go saat modulnya lulus gerbang 2.
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
-- TIDAK ADA DELETE di berkas ini. Sistem lama pun tidak punya satu pun terhadap tabel
-- ini, dan D-66 melarang penghapusan fisik data bernilai bisnis. Bengkel yang tidak lagi
-- dipakai ditolak (APPROVAL='2') atau ditandai lewat STS_BENGKEL, bukan dibuang.
--
--
-- ============================================================================
-- KENAPA MENULIS KE BENGKEL_HE, PADAHAL PEGA MENULIS KE M_BENGKEL_HE
-- ============================================================================
--
-- Sistem lama memakai DUA tabel untuk satu master, dan itu pola dua-penyimpanan yang
-- `03-CURRENT-ARCHITECTURE.md` §3.2 catat sebagai R-10:
--
--   tulis  Activity/UpdateBengkelHE_act  ->  InputBengkel.ACCOUNT_ID := @GCNM.GetPageJSONString()
--          RDB List/UpdateBengkelHE-SQL  ->  POOLDATA.PEGA_M_BENGKEL_HE(Datapega, IDPega, out)
--          Database/PEGA_M_BENGKEL_HE.prc:22 -> INSERT INTO POOLDATA.M_BENGKEL_HE(ID, JSONDATA)
--
--   baca   Report Definition/BrowseBengkelHE_RD  ->  POOLDATA.BENGKEL_HE, 40 kolom
--          RDB List/ValidationMasterBengkel      ->  pooldata.bengkel_he
--          RDB List/CountMasterBengkelManagee    ->  POOLDATA.BENGKEL_HE
--          RDB List/GetIDDokumenBengkel          ->  POOLDATA.BENGKEL_HE
--
-- Menulisnya lewat procedure dilarang D-02. Yang tersisa adalah dua pilihan, dan hanya
-- satu yang dapat dikerjakan tanpa menebak:
--
--   (a) menulis dokumen JSON ke M_BENGKEL_HE.JSONDATA dengan SQL biasa.
--       TIDAK DAPAT DIKERJAKAN: nama kunci JSON-nya diterbitkan fungsi
--       `@GCNM.GetPageJSONString()`, dan `Function/GetPageJSONString-Function.xml`
--       HANYA memuat tanda tangannya — badan fungsinya tidak ikut di export (R-16).
--       Setiap kunci yang ditulis akan menjadi tebakan, pada master yang menentukan
--       diskon, pajak, dan rekening tujuan pembayaran.
--
--   (b) menulis kolom bernama ke BENGKEL_HE.
--       DAPAT DIKERJAKAN: keempat puluh kolomnya terbaca lengkap dari
--       `BrowseBengkelHE_RD-RD.xml`, sehingga setiap nilai yang ditulis diketahui
--       benar, dan apa yang ditulis dapat dibaca kembali oleh layar yang sama.
--
-- (b) yang dipakai. Perlakuannya sama dengan Master Status Klaim, yang menghadapi
-- keluarga procedure PEGA_M_* yang sama persis dan memutuskan hal yang sama (D-02,
-- D-68): Go menjadi penulis tunggal dan berhenti menulis JSONDATA.
--
-- ## Satu hal yang WAJIB dipastikan DBA sebelum modul ini menulis di produksi
--
-- Apakah POOLDATA.BENGKEL_HE sebuah TABEL, atau sebuah VIEW atas
-- M_BENGKEL_HE.JSONDATA. DDL-nya tidak ada di export (R-08), dan keduanya sama-sama
-- masuk akal:
--
--   * Bila TABEL — kueri di berkas ini berjalan apa adanya.
--   * Bila VIEW  — setiap INSERT dan UPDATE di sini akan ditolak, dan penulisannya
--                  harus pindah ke M_BENGKEL_HE. Preseden bentuk itu ada:
--                  V_STS_CLAIM adalah view atas M_STS_CLAIM.JSONDATA, dan migrasi 0002
--                  yang membongkarnya menempuh prosedur D-63.
--
-- `claimpnc -periksa` melaporkan keduanya — lihat checkBengkel pada cmd/claimpnc.
-- Tidak ada satu pun pernyataan tulis yang dijalankan sebelum itu dipastikan.
--
--
-- CATATAN TRIM pada setiap penyaring kunci. Tipe kolom BENGKEL_HE belum diketahui
-- (R-08). Bila ID_BENGKEL bertipe CHAR berlebar tetap, nilainya dipadatkan spasi tanpa
-- tanda apa pun; Oracle membandingkan CHAR dengan CHAR secara blank-padded, sehingga
-- kueri lama yang MERANGKAI nilainya tetap cocok. Parameter binding bertipe VARCHAR2,
-- dan perbandingan CHAR dengan VARCHAR2 memakai non-padded comparison: "ABC " tidak sama
-- dengan "ABC", dan barisnya tidak ketemu. Menyalin `= :1` apa adanya karena itu justru
-- MENGUBAH perilaku. TRIM benar untuk kedua kemungkinan tipe. Biayanya index atas kolom
-- itu tidak terpakai; dapat diterima pada tabel master berbaris sedikit, dan TIDAK boleh
-- ditiru pada tabel besar.

-- name: bengkel_list
--
-- Asal: Report Definition/BrowseBengkelHE_RD-RD.xml, dipakai ketiga tab
-- `Section/BrowseMasterHE-Section.xml` yang hanya berbeda pada nilai APPROVAL-nya.
--
-- Keempat puluh satu kolomnya disebut pada urutan yang SAMA dengan bengkel_get dan
-- bengkel_find_* — satu fungsi scanRow membaca keempatnya berdasarkan POSISI, dan satu
-- kolom yang bergeser akan menaruh nomor rekening ke kolom alamat tanpa satu pun galat.
-- `TestReaderQueriesShareColumnOrder` yang menjaganya.
--
-- DOKUMENID ikut dibaca meski `BrowseBengkelHE_RD` tidak menyebutnya: ia dibaca terpisah
-- oleh `RDB List/GetIDDokumenBengkel-SQL.xml` atas tabel yang sama. Membacanya sekaligus
-- menghindarkan satu perjalanan basis data per baris, dan membuat penyimpanan dapat
-- menulis balik nilainya apa adanya alih-alih mengosongkannya.
--
-- ORDER BY MENIRU URUTAN BAWAAN GRID PEGA, bukan dikarang.
--
-- `Section/BrowseMasterHEApprove-Section.xml` menyetel kolom PERTAMA grid — ID_BENGKEL —
-- dengan `pySortType=DESC` dan `pySortOrder=1`, sehingga daftar yang dilihat pengguna
-- hari ini menurun menurut ID: bengkel yang paling baru didaftarkan muncul di atas.
--
-- Kueri lamanya sendiri tidak punya `ORDER BY` sama sekali; yang mengurutkan adalah grid
-- di peramban, atas page list yang sudah dimuat. Menaruhnya di sini membuat urutan yang
-- terlihat sama dengan yang dilihat Pega **tanpa** bergantung pada pengurutan sisi
-- klien — dan itu satu-satunya cara halaman kedua memuat baris yang benar.
--
-- CATATAN LEBAR KUNCI. Pengurutan ini leksikografis bila ID_BENGKEL bertipe teks, dan itu
-- sama dengan pengurutan numerik HANYA selama lebarnya tetap. ID yang diterbitkan
-- berlebar tetap (kode situs + sepuluh digit bertambal nol), sehingga hari ini keduanya
-- sama. Bila nomor urut kelak melampaui sepuluh digit, kunci tumbuh lebih panjang —
-- lihat masterbengkel.ComposeID — dan urutan teks tidak lagi sama dengan urutan
-- penerbitan. Keadaan itu masih sangat jauh, tetapi ia tidak akan menimbulkan galat apa
-- pun saat tiba, sehingga layak dicatat di sini.
--
-- `pyMaxRecords=500` pada report definition lama TIDAK direplikasi. Ia memotong daftar
-- di 500 baris tanpa satu pun tanda di layar; penggantinya adalah penyaring kata kunci
-- pada bengkel_list_search, ditambah paginasi 20 baris per halaman di layar — angka yang
-- dibaca dari `pyPageSizeOther` section yang sama.
SELECT ID_BENGKEL,
       NAMA_BENGKEL,
       ALM_BENGKEL,
       TELP_BENGKEL,
       NOHP_BENGKEL,
       MAIL,
       MAIL_WO,
       CABANG_ID,
       NAMA_CABANG,
       CITY_ID,
       NAMA_KABUPATEN,
       STATUS_REKANAN,
       STS_BENGKEL,
       ALASAN_STS_BGKL,
       TGL_STATUS,
       LOGIN_APLIKASI,
       BANK_ID,
       NAMA_BANK,
       NO_ACCOUNT,
       NAMA_ACCOUNT,
       ACCOUNT_ID,
       NAMA_NPWP,
       NO_NPWP,
       ALM_NPWP,
       JENIS_PPH,
       PPN,
       DISC_JASA,
       DISC_SPART,
       PERSEN_MATERIAL,
       PCT_SELISIH_PL,
       SLA,
       STS_SUPPLY,
       SUPPLIER,
       STS_EKLAIM,
       STS_AUTO_AKSEP,
       STS_PAYMENT,
       STS_AUTOPAYMENT,
       STS_TEKNO,
       STS_ORDER,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.BENGKEL_HE
 WHERE TRIM(APPROVAL) = :1
 ORDER BY ID_BENGKEL DESC

-- name: bengkel_list_search
--
-- Sama dengan bengkel_list, ditambah penyaring kata kunci atas nama bengkel, kota, dan
-- cabang.
--
-- KUERI TERSENDIRI, bukan satu kueri yang klausanya ditempel. Sistem lama menempuh cara
-- yang kedua di banyak tempat lewat pola `{ASIS:...}` — nilai dirangkai langsung ke teks
-- SQL — dan `08-TECHNICAL-STRATEGY.md` §4.3 melarangnya tanpa perkecualian. Dua kueri
-- pendek yang keduanya utuh lebih mudah dibaca, dan lebih mudah dibuktikan aman,
-- daripada satu kueri yang bentuk akhirnya baru diketahui saat berjalan.
--
-- UPPER dipasang pada KOLOMNYA juga, bukan hanya pada kata kuncinya: nama bengkel
-- tersimpan dengan besar-kecil huruf apa adanya, dan pencarian yang hanya meng-uppercase
-- kata kunci tidak akan pernah menemukan baris yang namanya huruf kecil.
--
-- `ESCAPE '\'` disebut eksplisit karena Oracle TIDAK punya karakter pelolos bawaan pada
-- LIKE. Tanpa itu, tanda persen yang diketik pengguna tetap berlaku sebagai wildcard
-- meski sudah diloloskan di Go — dan hasil pencariannya tidak dapat dijelaskan kepada
-- yang mengetiknya. PostgreSQL memakai backslash sebagai bawaan; menyebutkannya
-- eksplisit membuat kedua basis data berperilaku sama (D-20).
SELECT ID_BENGKEL,
       NAMA_BENGKEL,
       ALM_BENGKEL,
       TELP_BENGKEL,
       NOHP_BENGKEL,
       MAIL,
       MAIL_WO,
       CABANG_ID,
       NAMA_CABANG,
       CITY_ID,
       NAMA_KABUPATEN,
       STATUS_REKANAN,
       STS_BENGKEL,
       ALASAN_STS_BGKL,
       TGL_STATUS,
       LOGIN_APLIKASI,
       BANK_ID,
       NAMA_BANK,
       NO_ACCOUNT,
       NAMA_ACCOUNT,
       ACCOUNT_ID,
       NAMA_NPWP,
       NO_NPWP,
       ALM_NPWP,
       JENIS_PPH,
       PPN,
       DISC_JASA,
       DISC_SPART,
       PERSEN_MATERIAL,
       PCT_SELISIH_PL,
       SLA,
       STS_SUPPLY,
       SUPPLIER,
       STS_EKLAIM,
       STS_AUTO_AKSEP,
       STS_PAYMENT,
       STS_AUTOPAYMENT,
       STS_TEKNO,
       STS_ORDER,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.BENGKEL_HE
 WHERE TRIM(APPROVAL) = :1
   AND (UPPER(NAMA_BENGKEL) LIKE :2 ESCAPE '\'
        OR UPPER(NAMA_KABUPATEN) LIKE :2 ESCAPE '\'
        OR UPPER(NAMA_CABANG) LIKE :2 ESCAPE '\')
 ORDER BY ID_BENGKEL DESC

-- name: bengkel_get
--
-- Satu baris, untuk dimuat ke form penyuntingan. Urutan kolomnya sama dengan
-- bengkel_list.
SELECT ID_BENGKEL,
       NAMA_BENGKEL,
       ALM_BENGKEL,
       TELP_BENGKEL,
       NOHP_BENGKEL,
       MAIL,
       MAIL_WO,
       CABANG_ID,
       NAMA_CABANG,
       CITY_ID,
       NAMA_KABUPATEN,
       STATUS_REKANAN,
       STS_BENGKEL,
       ALASAN_STS_BGKL,
       TGL_STATUS,
       LOGIN_APLIKASI,
       BANK_ID,
       NAMA_BANK,
       NO_ACCOUNT,
       NAMA_ACCOUNT,
       ACCOUNT_ID,
       NAMA_NPWP,
       NO_NPWP,
       ALM_NPWP,
       JENIS_PPH,
       PPN,
       DISC_JASA,
       DISC_SPART,
       PERSEN_MATERIAL,
       PCT_SELISIH_PL,
       SLA,
       STS_SUPPLY,
       SUPPLIER,
       STS_EKLAIM,
       STS_AUTO_AKSEP,
       STS_PAYMENT,
       STS_AUTOPAYMENT,
       STS_TEKNO,
       STS_ORDER,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.BENGKEL_HE
 WHERE TRIM(ID_BENGKEL) = :1

-- name: bengkel_find_by_name
--
-- Asal: RDB List/ValidationMasterBengkel-SQL.xml
--
--   select id_bengkel as "ID_BENGKEL", login_aplikasi as "LOGIN_APLIKASI"
--     from pooldata.bengkel_he
--    where upper(trim(nama_bengkel)) = upper(trim({TempInputBengkelHE.CaseID}))
--
-- Perlakuan `upper(trim(...))` di KEDUA sisi ditiru apa adanya — ia yang membuat
-- "Bengkel Jaya" dan "BENGKEL JAYA " dikenali sebagai nama yang sama.
--
-- Alias `CaseID` pada kueri lama TIDAK dibawa: ia nama klipboard yang tidak mencerminkan
-- isinya sama sekali (D-19). Yang dipetakan adalah kolomnya.
--
-- Kueri lama hanya mengambil dua kolom. Di sini seluruh baris dibaca dengan urutan yang
-- sama seperti bengkel_list, supaya pemanggil dapat menyebutkan ID DAN nama bengkel yang
-- sudah memakai nama itu di dalam pesan galatnya — "sudah dipakai" tanpa menyebut yang
-- mana memaksa pengguna mencarinya sendiri.
SELECT ID_BENGKEL,
       NAMA_BENGKEL,
       ALM_BENGKEL,
       TELP_BENGKEL,
       NOHP_BENGKEL,
       MAIL,
       MAIL_WO,
       CABANG_ID,
       NAMA_CABANG,
       CITY_ID,
       NAMA_KABUPATEN,
       STATUS_REKANAN,
       STS_BENGKEL,
       ALASAN_STS_BGKL,
       TGL_STATUS,
       LOGIN_APLIKASI,
       BANK_ID,
       NAMA_BANK,
       NO_ACCOUNT,
       NAMA_ACCOUNT,
       ACCOUNT_ID,
       NAMA_NPWP,
       NO_NPWP,
       ALM_NPWP,
       JENIS_PPH,
       PPN,
       DISC_JASA,
       DISC_SPART,
       PERSEN_MATERIAL,
       PCT_SELISIH_PL,
       SLA,
       STS_SUPPLY,
       SUPPLIER,
       STS_EKLAIM,
       STS_AUTO_AKSEP,
       STS_PAYMENT,
       STS_AUTOPAYMENT,
       STS_TEKNO,
       STS_ORDER,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.BENGKEL_HE
 WHERE UPPER(TRIM(NAMA_BENGKEL)) = :1
 FETCH NEXT 1 ROW ONLY

-- name: bengkel_find_by_login
--
-- Padanan pemeriksaan `Activity/ValidationLoginBengkel_act` step 5–7, yang mencari login
-- yang sama lalu menolak dengan pesan
-- "Login aplikasi tersebut talah dipakai, tolong ubah login aplikasi".
--
-- YANG DICARI BERBEDA, dan itu tidak terhindarkan. Pega mencarinya di daftar operator
-- Pega (`Data-Admin-Operator-ID` lewat report `GCNMGetListOfOperators`), karena di sana
-- login bengkel memang berupa operator. Sistem baru tidak punya operator Pega, dan
-- kontrak identitasnya belum ada (F-3, R-14) — sehingga keunikannya diperiksa terhadap
-- kolom LOGIN_APLIKASI pada tabel ini.
--
-- Akibat yang harus disadari: login yang bertabrakan dengan operator Pega yang BUKAN
-- bengkel tidak lagi tertangkap. Itu selisih terencana, dan tertutup begitu F-3 punya
-- kontrak.
--
-- Baris ber-LOGIN_APLIKASI kosong sengaja tidak pernah cocok: bengkel non-rekanan boleh
-- tidak punya login, dan tanpa penyaring ini bengkel non-rekanan kedua akan ditolak
-- karena "login kosong sudah dipakai".
SELECT ID_BENGKEL,
       NAMA_BENGKEL,
       ALM_BENGKEL,
       TELP_BENGKEL,
       NOHP_BENGKEL,
       MAIL,
       MAIL_WO,
       CABANG_ID,
       NAMA_CABANG,
       CITY_ID,
       NAMA_KABUPATEN,
       STATUS_REKANAN,
       STS_BENGKEL,
       ALASAN_STS_BGKL,
       TGL_STATUS,
       LOGIN_APLIKASI,
       BANK_ID,
       NAMA_BANK,
       NO_ACCOUNT,
       NAMA_ACCOUNT,
       ACCOUNT_ID,
       NAMA_NPWP,
       NO_NPWP,
       ALM_NPWP,
       JENIS_PPH,
       PPN,
       DISC_JASA,
       DISC_SPART,
       PERSEN_MATERIAL,
       PCT_SELISIH_PL,
       SLA,
       STS_SUPPLY,
       SUPPLIER,
       STS_EKLAIM,
       STS_AUTO_AKSEP,
       STS_PAYMENT,
       STS_AUTOPAYMENT,
       STS_TEKNO,
       STS_ORDER,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.BENGKEL_HE
 WHERE UPPER(TRIM(LOGIN_APLIKASI)) = :1
   AND TRIM(LOGIN_APLIKASI) IS NOT NULL
 FETCH NEXT 1 ROW ONLY

-- name: bengkel_lock_by_name
--
-- Mengunci baris yang namanya sama, dipakai sebelum menyisipkan.
--
-- Di Pega, `ValidateMasterBengkel` dijalankan dari layar sementara penyimpanannya
-- terjadi jauh sesudahnya lewat `UpdateBengkelHE_act`. Di antara keduanya tidak ada apa
-- pun yang menghalangi penambahan lain masuk lebih dulu.
--
-- FOR UPDATE di sini MEMPERSEMPIT jarak itu, tetapi TIDAK MENUTUPNYA — dan itu harus
-- dinyatakan terang-terangan. Mengunci baris yang belum ada tidak mungkin: bila kedua
-- penambahan sama-sama menemukan nol baris, keduanya lolos. Penutup yang sebenarnya
-- adalah constraint unik pada NAMA_BENGKEL, dan itu menunggu DDL (R-08) beserta prosedur
-- perubahan skema (D-63).
SELECT ID_BENGKEL
  FROM POOLDATA.BENGKEL_HE
 WHERE UPPER(TRIM(NAMA_BENGKEL)) = :1
 FOR UPDATE

-- name: bengkel_lock_by_login
--
-- Sama dengan bengkel_lock_by_name, untuk LOGIN_APLIKASI. Keterbatasannya sama persis.
SELECT ID_BENGKEL
  FROM POOLDATA.BENGKEL_HE
 WHERE UPPER(TRIM(LOGIN_APLIKASI)) = :1
   AND TRIM(LOGIN_APLIKASI) IS NOT NULL
 FOR UPDATE

-- name: bengkel_insert
--
-- Keempat puluh satu kolom disebut namanya. Tidak ada kolom yang dibiarkan terisi
-- default: tabel ini tidak punya default yang diketahui (R-08), dan kolom yang tidak
-- disebut akan bernilai NULL — yang pada kolom persentase berarti "belum ditentukan"
-- alih-alih nol, dan dibaca berbeda oleh setiap hitungan di hilirnya.
--
-- ID_BENGKEL TIDAK diterbitkan basis data. Ia dibentuk di Go dari kode situs dan
-- sequence, persis seperti `Database/PEGA_M_BENGKEL_HE.prc:19` — lihat bengkel_site dan
-- bengkel_next_sequence.
INSERT INTO POOLDATA.BENGKEL_HE
       (ID_BENGKEL, NAMA_BENGKEL, ALM_BENGKEL, TELP_BENGKEL, NOHP_BENGKEL,
        MAIL, MAIL_WO, CABANG_ID, NAMA_CABANG, CITY_ID,
        NAMA_KABUPATEN, STATUS_REKANAN, STS_BENGKEL, ALASAN_STS_BGKL, TGL_STATUS,
        LOGIN_APLIKASI, BANK_ID, NAMA_BANK, NO_ACCOUNT, NAMA_ACCOUNT,
        ACCOUNT_ID, NAMA_NPWP, NO_NPWP, ALM_NPWP, JENIS_PPH,
        PPN, DISC_JASA, DISC_SPART, PERSEN_MATERIAL, PCT_SELISIH_PL,
        SLA, STS_SUPPLY, SUPPLIER, STS_EKLAIM, STS_AUTO_AKSEP,
        STS_PAYMENT, STS_AUTOPAYMENT, STS_TEKNO, STS_ORDER, DOKUMENID,
        APPROVAL)
VALUES (:1, :2, :3, :4, :5,
        :6, :7, :8, :9, :10,
        :11, :12, :13, :14, :15,
        :16, :17, :18, :19, :20,
        :21, :22, :23, :24, :25,
        :26, :27, :28, :29, :30,
        :31, :32, :33, :34, :35,
        :36, :37, :38, :39, :40,
        :41)

-- name: bengkel_update
--
-- Empat puluh kolom; ID_BENGKEL tidak disebut karena ia kunci baris, bukan isian.
-- Memindahkan sebuah baris ke ID lain berarti menambah baris baru, bukan mengubah yang
-- ada.
--
-- APPROVAL IKUT DITULIS, dan selalu bernilai "0". Itu bukan kelalaian melainkan langkah
-- tersendiri di sistem lama: `Activity/UpdateBengkelHE_act` step 7 menetapkannya tanpa
-- syarat apa pun, sehingga menyunting bengkel yang sudah disetujui mengembalikannya ke
-- antrean persetujuan.
--
-- DOKUMENID juga ikut, tetapi nilainya dibaca dari baris yang tersimpan dan ditulis
-- kembali apa adanya — lihat usecase.Service.Save untuk alasan kenapa jalur Pega yang
-- mengosongkannya tidak dibawa.
UPDATE POOLDATA.BENGKEL_HE
   SET NAMA_BENGKEL = :1,
       ALM_BENGKEL = :2,
       TELP_BENGKEL = :3,
       NOHP_BENGKEL = :4,
       MAIL = :5,
       MAIL_WO = :6,
       CABANG_ID = :7,
       NAMA_CABANG = :8,
       CITY_ID = :9,
       NAMA_KABUPATEN = :10,
       STATUS_REKANAN = :11,
       STS_BENGKEL = :12,
       ALASAN_STS_BGKL = :13,
       TGL_STATUS = :14,
       LOGIN_APLIKASI = :15,
       BANK_ID = :16,
       NAMA_BANK = :17,
       NO_ACCOUNT = :18,
       NAMA_ACCOUNT = :19,
       ACCOUNT_ID = :20,
       NAMA_NPWP = :21,
       NO_NPWP = :22,
       ALM_NPWP = :23,
       JENIS_PPH = :24,
       PPN = :25,
       DISC_JASA = :26,
       DISC_SPART = :27,
       PERSEN_MATERIAL = :28,
       PCT_SELISIH_PL = :29,
       SLA = :30,
       STS_SUPPLY = :31,
       SUPPLIER = :32,
       STS_EKLAIM = :33,
       STS_AUTO_AKSEP = :34,
       STS_PAYMENT = :35,
       STS_AUTOPAYMENT = :36,
       STS_TEKNO = :37,
       STS_ORDER = :38,
       DOKUMENID = :39,
       APPROVAL = :40
 WHERE TRIM(ID_BENGKEL) = :41

-- name: bengkel_set_status
--
-- Menetapkan APPROVAL SATU baris, tanpa menyentuh kolom lain.
--
-- Asal: Activity/SetApprovalAllMaster-Act.xml, yang menelusuri baris bercentang
-- (`.pySelected=="true"`) lalu menetapkan `InputBengkel.APPROVAL := Param.approval` pada
-- masing-masing. Rule SQL yang menjalankannya TIDAK ADA di export (R-16) — yang terbaca
-- hanyalah kolom yang disentuhnya.
--
-- Satu baris per pernyataan, dijalankan berulang di dalam SATU transaksi — bukan satu
-- pernyataan dengan daftar kunci yang panjangnya berubah-ubah. Alasannya: daftar
-- placeholder yang dibentuk saat berjalan membuat teks SQL tidak lagi tetap, dan itu
-- persis bentuk yang §4.3 larang. Biayanya beberapa perjalanan tambahan pada operasi
-- yang jarang dan berbaris sedikit; yang diperoleh adalah teks kueri yang dapat dibaca
-- utuh di berkas ini.
--
-- MAIL SENGAJA TIDAK IKUT. Sistem lama menulisi `InputBengkel.MAIL := .USER_UPDATE` —
-- menimpa kolom surel bengkel dengan identitas petugas yang menyetujui. Itu satu lagi
-- kolom berarti ganda, dan membawanya berarti menghapus alamat surel bengkel setiap kali
-- ia disetujui.
UPDATE POOLDATA.BENGKEL_HE
   SET APPROVAL = :1
 WHERE TRIM(ID_BENGKEL) = :2
   AND TRIM(APPROVAL) <> :1

-- name: bengkel_count_pending
--
-- Asal: RDB List/CountMasterBengkelManagee-SQL.xml
--
--   SELECT COUNT(ID_BENGKEL) AS "City" FROM POOLDATA.BENGKEL_HE
--    WHERE APPROVAL = '0' ORDER BY 1 DESC
--
-- Alias `City` untuk sebuah pencacah tidak dibawa (D-19); `ORDER BY 1 DESC` atas kueri
-- yang selalu mengembalikan tepat satu baris juga tidak — ia tidak berakibat apa pun.
--
-- Dipakai mode periksa, bukan layar: ia menjawab "berapa banyak yang menunggu" sebelum
-- layarnya dibuka.
SELECT COUNT(ID_BENGKEL)
  FROM POOLDATA.BENGKEL_HE
 WHERE TRIM(APPROVAL) = :1

-- name: bengkel_check_table
--
-- Memastikan tabel ada dan dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
SELECT ID_BENGKEL
  FROM POOLDATA.BENGKEL_HE
 WHERE 1 = 0

-- name: bengkel_check_json_mirror
--
-- Memastikan POOLDATA.M_BENGKEL_HE — tabel JSON yang ditulis Pega — dapat dibaca, tanpa
-- mengambil satu baris pun.
--
-- Ia TIDAK dipakai jalur mana pun selain mode periksa, dan itu memang tujuannya:
-- membandingkan jumlah barisnya dengan BENGKEL_HE adalah cara termurah mengetahui apakah
-- keduanya satu sumber (view) atau dua sumber yang disinkronkan. Lihat banner berkas ini.
SELECT ID
  FROM POOLDATA.M_BENGKEL_HE
 WHERE 1 = 0

-- name: bengkel_count_json_mirror
--
-- Jumlah baris pada tabel JSON milik Pega. Dibandingkan dengan jumlah baris BENGKEL_HE
-- di mode periksa.
SELECT COUNT(ID)
  FROM POOLDATA.M_BENGKEL_HE

-- name: bengkel_count_all
--
-- Jumlah seluruh baris BENGKEL_HE, tanpa penyaring status.
SELECT COUNT(ID_BENGKEL)
  FROM POOLDATA.BENGKEL_HE

-- name: bengkel_site
--
-- Kode situs, bagian pertama dari setiap ID_BENGKEL. Meniru
-- `Database/PEGA_M_BENGKEL_HE.prc:11` persis, termasuk pembandingnya yang berupa TEKS
-- '1' dan bukan angka.
--
-- Tabel yang sama dibaca Master Status Klaim lewat kueri `claim_status_site`-nya
-- sendiri. Keduanya sengaja tidak dipakai bersama: modul tidak saling mengimpor, dan
-- kueri bersama akan membuat perubahan di satu modul menyeret modul lain.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: bengkel_next_sequence
--
-- Urutan yang SAMA dengan yang dipakai procedure lama, supaya ID yang diterbitkan
-- aplikasi ini melanjutkan deret yang sudah ada dan tidak pernah bertabrakan dengan ID
-- yang pernah diterbitkan Pega.
--
-- `Database/PEGA_M_BENGKEL_HE.prc:19` menyebutnya tanpa skema (`BENGKEL_HE_SEQ`),
-- sehingga yang terpakai bergantung pada skema bawaan akun koneksi — dan itu berbeda
-- antar lingkungan. Di sini ia dilengkapi POOLDATA, skema yang sama dengan tabel yang
-- disisipi procedure itu.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada ADR-0005, satu-satunya tempat lain
-- yang dibenarkan memuat percabangan dialek.
SELECT POOLDATA.BENGKEL_HE_SEQ.NEXTVAL
  FROM DUAL

-- name: bengkel_branch_list
--
-- Asal: RDB List/BrowseCabangBengkelHE-SQL.xml
--
--   select ldc_nama as BRANCHNAME, cab_id as BRANCH
--     from general.lst_user_asuransi a, lst_det_cabang b
--    where b.ldc_id = a.cab_id
--      and b.ldc_id = a.rep_cab_id
--      and a.ldi_id = '0076'
--      and a.sts_aktif = '1'
--
-- DUA PERUBAHAN terhadap kueri lama, keduanya tidak mengubah baris yang dikembalikan:
--
--  1. Gabungan gaya lama (dua tabel pada FROM, syarat gabung di WHERE) menjadi
--     JOIN ... ON. Bentuknya sah di kedua basis data; yang berubah hanya keterbacaannya
--     — syarat gabung tidak lagi bercampur dengan penyaring.
--  2. DISTINCT ditambahkan. `lst_user_asuransi` berisi satu baris per PENGGUNA, sehingga
--     cabang yang punya banyak pengguna aktif muncul berkali-kali pada daftar pilihan.
--     Kueri lama tidak membuangnya, dan dropdown-nya memang memuat nama yang berulang.
--
-- Penyaring `ldi_id = '0076'` DIREPLIKASI APA ADANYA. Ia kode aplikasi yang tertanam di
-- kueri — persis bentuk hardcode yang D-15 perintahkan menjadi konfigurasi — tetapi tidak
-- ada satu pun keterangan di export tentang artinya, sehingga mengangkatnya menjadi
-- konfigurasi berarti menebak nilainya untuk entitas selain yang sekarang. Ia literal,
-- bukan masukan pengguna, sehingga bukan celah injeksi. Dicatat sebagai utang.
SELECT DISTINCT b.LDC_NAMA,
       a.CAB_ID
  FROM GENERAL.LST_USER_ASURANSI a
  JOIN LST_DET_CABANG b
    ON b.LDC_ID = a.CAB_ID
   AND b.LDC_ID = a.REP_CAB_ID
 WHERE a.LDI_ID = '0076'
   AND a.STS_AKTIF = '1'
 ORDER BY b.LDC_NAMA

-- name: bengkel_city_search
--
-- Asal: Report Definition/BrowseCity_RD-RD.xml atas kelas ASM-FW-GISFW-Int-CITY, yang
-- menyuapi autocomplete "NAMA KOTA".
--
-- PERHATIKAN NAMA KOLOMNYA: nama kota tersimpan di kolom bernama NOTE, bukan NAME
-- maupun CITYNAME. Itu terbaca dari kueri lain atas tabel yang sama, misalnya
-- `RDB List/BrowseRW_SQL-SQL.xml`: `(select distinct a.note from city a where a.id = b.cityid)`.
--
-- Nama tabelnya disebut TANPA skema, persis seperti seluruh kueri lama yang membacanya —
-- sehingga yang terbaca mengikuti skema bawaan akun koneksi. Melengkapinya dengan skema
-- berarti menebak, dan tebakan yang salah membuat lookup Kota kosong tanpa galat yang
-- menjelaskan sebabnya.
--
-- FETCH NEXT membatasi hasil; kueri lama tidak membatasinya sama sekali.
SELECT ID,
       NOTE
  FROM CITY
 WHERE UPPER(NOTE) LIKE :1 ESCAPE '\'
    OR UPPER(TRIM(ID)) = :2
 ORDER BY NOTE
 FETCH NEXT 50 ROWS ONLY

-- name: bengkel_bank_list
--
-- Asal: Report Definition/BrowseBankGroup-RD.xml atas GENERAL.LST_BANK_GROUP, yang
-- menyuapi autocomplete "NAMA BANK".
--
-- Tabel yang sama dibaca modul Master Rekening dan Master Auto Claim lewat kueri
-- masing-masing. Ketiganya sengaja tidak dipakai bersama: modul tidak saling mengimpor,
-- dan kueri bersama akan membuat perubahan di satu modul menyeret modul lain.
--
-- Berbeda dari Master Auto Claim yang hanya menyimpan nama banknya, tabel bengkel punya
-- kolom kodenya sendiri (BANK_ID) — sehingga keduanya dipakai, bukan hanya namanya.
SELECT LBG_ID,
       BANK_GROUP
  FROM GENERAL.LST_BANK_GROUP
 ORDER BY BANK_GROUP
