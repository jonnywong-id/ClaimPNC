-- Kueri modul Master Auto Claim.
--
-- Satu tabel DITULIS aplikasi ini — POOLDATA.M_AUTO_CLAIM_PNC — dan empat tabel lain
-- hanya DIBACA: POOLDATA.AGENT, POOLDATA.CLIENT, GENERAL.LST_BANK_GROUP, dan
-- POOLDATA.EMAILKOMITE. Keempatnya milik sistem lain (ADR-0004, penulis tunggal per
-- tabel); tidak ada satu pun pernyataan tulis terhadapnya di berkas ini.
--
-- Kewenangan menulis M_AUTO_CLAIM_PNC berpindah dari Pega ke Go saat modulnya lulus
-- gerbang 2. Selama Pega masih penulisnya, layar ini harus dijalankan dalam modus baca
-- saja di produksi.
--
-- Empat aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (D-20).
--   4. Tanpa pemanggilan stored procedure (D-02).
--
-- TIDAK ADA DELETE di berkas ini. Sistem lama pun tidak punya satu pun terhadap tabel
-- ini, dan D-66 melarang penghapusan fisik data bernilai bisnis. Baris yang tidak lagi
-- dipakai ditolak komite (APPROVAL='2'), bukan dibuang.
--
-- CATATAN TRIM pada setiap penyaring kunci. Tipe kolom M_AUTO_CLAIM_PNC belum diketahui
-- — DDL-nya tidak ada di export (R-08). Bila INISIALID bertipe CHAR berlebar tetap,
-- nilainya dipadatkan spasi tanpa tanda apa pun; Oracle membandingkan CHAR dengan CHAR
-- secara blank-padded, sehingga kueri lama yang MERANGKAI nilainya tetap cocok.
-- Parameter binding bertipe VARCHAR2, dan perbandingan CHAR dengan VARCHAR2 memakai
-- non-padded comparison: "ABC " tidak sama dengan "ABC", dan barisnya tidak ketemu.
-- Menyalin `= :1` apa adanya karena itu justru MENGUBAH perilaku. TRIM benar untuk
-- kedua kemungkinan tipe. Biayanya index atas kolom itu tidak terpakai; dapat diterima
-- pada tabel master berbaris sedikit, dan TIDAK boleh ditiru pada tabel besar.

-- name: auto_claim_list
--
-- Asal: RDB List/BrowseAutoKlaim-SQL.xml
--
--   SELECT INISIALID AS "CaseID", NAMA_PENERIMA AS "City", BANK_PENERIMA AS "CityID",
--          NO_REKENING AS "District", PCT_MAX AS "Country", PIC_LAPOR AS "AlasanTerlambat",
--          EMAIL_LAPOR AS "DistrictID", CLAIM_ALLOWED AS "AnalystDoctorRemaks",
--          ALAMAT_PENERIMA AS "CountryID", KOMITE as "ProdKe"
--     FROM POOLDATA.M_AUTO_CLAIM_PNC
--    where approval={TempApproval.City}
--    {ASIS:TempApproval.CityID}
--
-- Kesepuluh alias yang menyesatkan dibuang (D-19); yang dipetakan adalah kolomnya.
--
-- DUA KOLOM DITAMBAHKAN terhadap kueri lama: CLIENTID dan CLIENTNAME. Keduanya hanya
-- ada di kueri satu-baris `UpdateAutoClaim1-SQL.xml`, tidak di daftar.
--
-- Alasannya bukan kelengkapan, melainkan menutup satu jalur rusak. Keputusan Work Owner
-- 2026-09-19 mempertahankan bentuk sistem lama: menyetujui mengirim ULANG seluruh
-- isian. Di Pega, isian itu diambil dari page form — dan page form hanya terisi bila
-- barisnya lebih dulu dimuat lewat tombol Update. Komite yang menekan Approve langsung
-- dari grid mengirim CLIENTID dan CLIENTNAME KOSONG, dan `UpdateAutoClaim-SQL.xml`
-- menulis keduanya apa adanya: data client terhapus oleh tindakan menyetujui.
--
-- Dengan kedua kolom ikut terbaca di daftar, layar selalu memegang nilai yang
-- sebenarnya, dan pengiriman ulang itu tidak dapat menghapus apa pun. Penambahan kolom
-- pada sebuah SELECT tidak mengubah satu baris pun.
--
-- APPROVAL juga ikut dibaca meski ia penyaringnya sendiri, supaya baris yang sampai ke
-- layar menyebutkan statusnya sendiri alih-alih menyandarkannya pada tab mana yang
-- sedang dibuka.
--
-- ORDER BY DITAMBAHKAN. Kueri lama tidak punya, sehingga urutan barisnya adalah apa pun
-- yang dikembalikan basis data — dapat berbeda antar pemanggilan pada tabel yang sama.
-- Ini SELISIH YANG DIRENCANAKAN terhadap Pega, dan satu-satunya yang tampak di layar
-- daftar: urutan yang tidak ditentukan digantikan urutan yang ditentukan.
SELECT INISIALID,
       NAMA_PENERIMA,
       BANK_PENERIMA,
       NO_REKENING,
       PCT_MAX,
       PIC_LAPOR,
       EMAIL_LAPOR,
       CLAIM_ALLOWED,
       ALAMAT_PENERIMA,
       KOMITE,
       APPROVAL,
       CLIENTID,
       CLIENTNAME
  FROM POOLDATA.M_AUTO_CLAIM_PNC
 WHERE TRIM(APPROVAL) = :1
 ORDER BY INISIALID

-- name: auto_claim_list_by_committee
--
-- Sama dengan auto_claim_list, ditambah penyaring komite — tab "Komite Approval".
--
-- Asal penyaringnya: Activity/BrowseAutoKlaim_act step 3, yang MERANGKAI teks SQL
--
--   TempApproval.CityID := "and KOMITE = '" + OperatorID.pyUserIdentifier + "'"
--
-- lalu menyisipkannya lewat `{ASIS:TempApproval.CityID}`. Itu pola yang
-- `08-TECHNICAL-STRATEGY.md` §4.3 larang tanpa perkecualian: nilai dirangkai langsung
-- ke teks SQL, dan operator ID yang memuat tanda kutip tunggal akan mengubah bentuk
-- kuerinya.
--
-- Di sini ia menjadi kueri TERSENDIRI dengan parameter terikat, bukan satu kueri yang
-- potongan klausanya ditempel. Dua kueri pendek yang keduanya utuh lebih mudah dibaca
-- — dan lebih mudah dibuktikan aman — daripada satu kueri yang bentuk akhirnya baru
-- diketahui saat berjalan.
SELECT INISIALID,
       NAMA_PENERIMA,
       BANK_PENERIMA,
       NO_REKENING,
       PCT_MAX,
       PIC_LAPOR,
       EMAIL_LAPOR,
       CLAIM_ALLOWED,
       ALAMAT_PENERIMA,
       KOMITE,
       APPROVAL,
       CLIENTID,
       CLIENTNAME
  FROM POOLDATA.M_AUTO_CLAIM_PNC
 WHERE TRIM(APPROVAL) = :1
   AND TRIM(KOMITE) = :2
 ORDER BY INISIALID

-- name: auto_claim_get
--
-- Asal: RDB List/UpdateAutoClaim1-SQL.xml — kueri yang memuat satu baris ke form.
-- Namanya di Pega berawalan "Update" padahal ia SELECT; nama di sini menyebutkan apa
-- yang benar-benar dilakukannya.
--
-- Kueri lama tidak membaca APPROVAL; di sini ia ikut, supaya baris yang dimuat ke form
-- membawa statusnya sendiri. Tanpa itu layar harus mengingat dari tab mana ia datang.
SELECT INISIALID,
       NAMA_PENERIMA,
       BANK_PENERIMA,
       NO_REKENING,
       PCT_MAX,
       PIC_LAPOR,
       EMAIL_LAPOR,
       CLAIM_ALLOWED,
       ALAMAT_PENERIMA,
       KOMITE,
       APPROVAL,
       CLIENTID,
       CLIENTNAME
  FROM POOLDATA.M_AUTO_CLAIM_PNC
 WHERE TRIM(INISIALID) = :1

-- name: auto_claim_list_initial_locked
--
-- Mengunci baris yang kunci alaminya sama, dipakai sebelum menyisipkan.
--
-- Asal pemeriksaannya: RDB List/ValidasiAutoClaim-SQL.xml
--
--   select inisialid as "CaseID" from pooldata.m_auto_claim_pnc
--    where inisialid={TempValidasi.CaseID}
--
-- Di Pega ia dijalankan sebagai langkah 4 dari sebuah activity, sedangkan INSERT-nya
-- langkah 8. Di antara keduanya tidak ada apa pun yang menghalangi penambahan lain
-- masuk lebih dulu.
--
-- FOR UPDATE di sini MEMPERSEMPIT jarak itu, tetapi TIDAK MENUTUPNYA — dan itu harus
-- dinyatakan terang-terangan. Mengunci baris yang belum ada tidak mungkin: bila kedua
-- penambahan sama-sama menemukan nol baris, keduanya lolos. Penutup yang sebenarnya
-- adalah constraint unik pada INISIALID, dan itu menunggu DDL (R-08) beserta prosedur
-- perubahan skema (D-63).
SELECT INISIALID
  FROM POOLDATA.M_AUTO_CLAIM_PNC
 WHERE TRIM(INISIALID) = :1
 FOR UPDATE

-- name: auto_claim_insert
--
-- Asal: RDB List/InsertAutoClaim-SQL.xml — empat belas kolom, urutan dipertahankan.
--
-- Pemetaan page klipboard lama ke kolom, dicatat karena ia satu-satunya cara memeriksa
-- ulang bahwa tidak ada dua isian yang tertukar:
--
--   INISIALID       {TempInputAutoClaim.CaseID}
--   NAMA_PENERIMA   {TempInputAutoClaim.CauseOfLoss}
--   BANK_PENERIMA   {TempInputAutoClaim.City}
--   NO_REKENING     {TempInputAutoClaim.DistrictID}
--   PCT_MAX         {TempInputAutoClaim.CountryID}
--   PIC_LAPOR       {TempInputAutoClaim.Country}
--   EMAIL_LAPOR     {TempInputAutoClaim.CityID}
--   CLAIM_ALLOWED   {TempInputAutoClaim.District}    <- selalu "1", ditimpa step 1
--   ALAMAT_PENERIMA {TempInputAutoClaim.Conveyance}
--   USERINPUT       {TempInputAutoClaim.Email}       <- OperatorID.pyUserIdentifier
--   KOMITE          {TempInputAutoClaim.FlagASO}     <- hasil GetKomiteAutoKlaim
--   APPROVAL        {TempInputAutoClaim.PolicyNo}    <- Param.Approval, selalu "0"
--   CLIENTID        {TempInputAutoClaim.FlagReject}
--   CLIENTNAME      {TempInputAutoClaim.EmailTertanggung}
--
-- Tabel ini tidak punya kolom pencatat kapan. Jejak audit perubahan master (D-28, modul
-- S-5) karena itu belum dapat disandarkan padanya — dicatat sebagai keterbatasan, bukan
-- ditambal dengan kolom yang dikarang.
INSERT INTO POOLDATA.M_AUTO_CLAIM_PNC
       (INISIALID, NAMA_PENERIMA, BANK_PENERIMA, NO_REKENING, PCT_MAX,
        PIC_LAPOR, EMAIL_LAPOR, CLAIM_ALLOWED, ALAMAT_PENERIMA, USERINPUT,
        KOMITE, APPROVAL, CLIENTID, CLIENTNAME)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13, :14)

-- name: auto_claim_update
--
-- Asal: RDB List/UpdateAutoClaim-SQL.xml — dua belas kolom.
--
-- NAMA_PENERIMA SENGAJA TIDAK DISEBUT, persis seperti kueri lama. Keputusan Work Owner
-- 2026-09-19: nama penerima tidak dapat diperbarui lewat layar. Kolomnya hanya ditulis
-- sekali, saat baris dibuat dari lookup Sumber Bisnis.
--
-- INISIALID juga tidak disebut — ia kunci baris, bukan isian. Memindahkan sebuah baris
-- ke kode sumber bisnis lain berarti menambah baris baru, bukan mengubah yang ada.
UPDATE POOLDATA.M_AUTO_CLAIM_PNC
   SET BANK_PENERIMA = :1,
       NO_REKENING = :2,
       PCT_MAX = :3,
       PIC_LAPOR = :4,
       EMAIL_LAPOR = :5,
       CLAIM_ALLOWED = :6,
       ALAMAT_PENERIMA = :7,
       APPROVAL = :8,
       USERINPUT = :9,
       KOMITE = :10,
       CLIENTID = :11,
       CLIENTNAME = :12
 WHERE TRIM(INISIALID) = :13

-- name: auto_claim_check_table
--
-- Memastikan tabel ada dan dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
-- Dipakai mode periksa.
SELECT INISIALID
  FROM POOLDATA.M_AUTO_CLAIM_PNC
 WHERE 1 = 0

-- name: auto_claim_business_source_search
--
-- Asal: RDB List/GetClientName-SQL.xml
--
--   select id as "Country", clientname as "CountryID" from pooldata.agent
--    where replace(clientname,'.','') like '%{ASIS:TempDcol.ATASAN}%'
--       or id = {TempDcol.ATASAN}
--
-- PERHATIKAN NAMA KOLOMNYA: `CLIENTNAME` pada tabel AGENT adalah nama SUMBER BISNIS,
-- bukan nama client. Master client yang sebenarnya adalah POOLDATA.CLIENT, dan ia
-- dibaca kueri berikutnya. Tertukar sekali saja berarti lookup Sumber Bisnis
-- menampilkan daftar tertanggung.
--
-- TIGA PERUBAHAN terhadap kueri lama, ketiganya disengaja:
--
--  1. `{ASIS:...}` menjadi parameter terikat. Kata kunci berisi tanda kutip tunggal
--     atau tanda persen mengubah bentuk kueri lama; di sini ia hanya nilai.
--  2. UPPER dipasang pada KOLOMNYA juga. Pega meng-uppercase kata kuncinya saja
--     (`@toUpperCase` pada Activity/GetClientName step 3), sehingga baris yang namanya
--     tidak tersimpan huruf besar TIDAK PERNAH ketemu. Memasang UPPER di kedua sisi
--     membuat pencarian menemukan lebih banyak baris — perubahan yang hanya menambah
--     hasil, tidak pernah mengurangi.
--  3. `or id = <kata kunci>` menjadi `CAST(ID AS VARCHAR(64)) = :2`. Bila ID bertipe
--     angka, perbandingan lama memaksa Oracle mengubah kata kunci menjadi angka — dan
--     kata kunci berupa nama akan menghasilkan ORA-01722, menggagalkan SELURUH
--     pencarian alih-alih hanya bagian itu. CAST memindahkan konversinya ke sisi kolom,
--     yang selalu berhasil. Tipe ID sendiri belum diketahui (R-08). Biayanya index atas
--     ID tidak terpakai pada cabang itu; dapat diterima untuk kotak pencarian.
--
-- Pembuangan titik dipertahankan: ia membuat "PT. ABC" dan "PT ABC" sama-sama ketemu.
--
-- FETCH NEXT membatasi hasil. Kueri lama tidak membatasinya sama sekali, dan kata kunci
-- sependek dua huruf dapat menarik ribuan baris ke memori aplikasi dan ke peramban
-- untuk daftar yang hanya akan dipilih satu.
--
-- `ESCAPE '\'` disebut eksplisit karena Oracle TIDAK punya karakter pelolos bawaan pada
-- LIKE. Tanpa itu, tanda persen yang diketik pengguna tetap berlaku sebagai wildcard
-- meski sudah diloloskan di Go — dan hasil pencariannya tidak dapat dijelaskan kepada
-- yang mengetiknya. PostgreSQL memakai backslash sebagai bawaan; menyebutkannya
-- eksplisit membuat kedua basis data berperilaku sama (D-20).
SELECT ID,
       CLIENTNAME
  FROM POOLDATA.AGENT
 WHERE UPPER(REPLACE(CLIENTNAME, '.', '')) LIKE :1 ESCAPE '\'
    OR CAST(ID AS VARCHAR(64)) = :2
 ORDER BY CLIENTNAME
 FETCH NEXT 50 ROWS ONLY

-- name: auto_claim_client_search
--
-- Asal: RDB List/GetClientName2-SQL.xml
--
--   select id as "Country", name as "CountryID" from pooldata.client
--    where replace(name,'.','') like '%{ASIS:TempDcol.ATASAN}%' or id = {TempDcol.ATASAN}
--
-- Ketiga perubahan yang sama dengan auto_claim_business_source_search berlaku di sini.
SELECT ID,
       NAME
  FROM POOLDATA.CLIENT
 WHERE UPPER(REPLACE(NAME, '.', '')) LIKE :1 ESCAPE '\'
    OR CAST(ID AS VARCHAR(64)) = :2
 ORDER BY NAME
 FETCH NEXT 50 ROWS ONLY

-- name: auto_claim_bank_list
--
-- Asal: DataPage/D_BankGroup-DataPage.xml, yang menyuapi autocomplete "Bank Penerima"
-- pada Section/BrowseAutoKlaim-Section.xml.
--
-- Tabel yang sama dibaca modul Master Rekening lewat kueri `bank_list`-nya sendiri.
-- Keduanya sengaja tidak dipakai bersama: modul tidak saling mengimpor, dan kueri
-- bersama akan membuat perubahan di satu modul menyeret modul lain.
SELECT LBG_ID,
       BANK_GROUP
  FROM GENERAL.LST_BANK_GROUP
 ORDER BY BANK_GROUP

-- name: auto_claim_bank_by_name
--
-- Mencocokkan nama bank yang akan tersimpan ke master bank.
--
-- Ia padanan pemeriksaan "Nama bank jangan diketik manual" pada
-- Activity/InsertMstAutoClaim_act step 3. Pega memeriksa bahwa autocomplete-nya sempat
-- menghasilkan sebuah kode; di sini diperiksa bahwa NAMANYA benar-benar ada.
--
-- UPPER dan TRIM di kedua sisi: BANK_GROUP dapat bertipe CHAR berlebar tetap, dan nama
-- bank yang dikirim layar berasal dari daftar yang sama sehingga besar-kecil hurufnya
-- seharusnya cocok — "seharusnya" bukan jaminan yang layak dipegang pada pemeriksaan
-- yang menolak penyimpanan.
SELECT LBG_ID,
       BANK_GROUP
  FROM GENERAL.LST_BANK_GROUP
 WHERE UPPER(TRIM(BANK_GROUP)) = :1

-- name: auto_claim_committee
--
-- Asal: RDB List/GetKomiteAutoKlaim-SQL.xml
--
--   select operator_id as "UserAdmin" from emailkomite
--    where type_business='BONDING' and sts_aktif='1'
--
-- lalu Activity/InsertMstAutoClaim_act step 7 mengambil BARIS PERTAMA saja.
--
-- DIREPLIKASI APA ADANYA atas keputusan Work Owner 2026-09-19, termasuk kedua
-- keanehannya:
--
--   * `TYPE_BUSINESS = 'BONDING'` tertanam di kueri, padahal modul ini bukan lini
--     Bonding. Ia persis bentuk hardcode yang D-15 perintahkan menjadi master atau
--     konfigurasi; dicatat sebagai utang teknis, bukan diperbaiki sepihak di sini.
--     Ia literal, bukan masukan pengguna, sehingga bukan celah injeksi.
--   * Tanpa ORDER BY. Bila barisnya lebih dari satu, penyetujunya ditentukan urutan
--     yang tidak dijamin apa pun dan dapat berbeda antar pemanggilan.
--
-- Satu tempat SAJA yang berbeda: nama tabelnya dilengkapi skema menjadi
-- POOLDATA.EMAILKOMITE. Kueri lama menyebutnya tanpa skema, sehingga yang terbaca
-- bergantung pada skema bawaan akun koneksi — dan itu berbeda antar lingkungan.
-- Dokumen `20-DETAIL-KOMITE-DBLINK.md` menyebut tabelnya sebagai POOLDATA.EMAILKOMITE,
-- dan seluruh kueri lain di export yang membacanya memakai skema itu.
--
-- FETCH NEXT 1 ROW ONLY setara dengan mengambil baris pertama dari hasil yang memang
-- tidak berurut: keduanya mengembalikan satu baris sembarang dari himpunan yang sama.
SELECT OPERATOR_ID
  FROM POOLDATA.EMAILKOMITE
 WHERE TYPE_BUSINESS = 'BONDING'
   AND STS_AKTIF = '1'
 FETCH NEXT 1 ROW ONLY
