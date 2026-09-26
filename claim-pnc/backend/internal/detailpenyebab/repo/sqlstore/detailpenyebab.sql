-- Kueri modul Detail Penyebab Kerugian.
--
-- SATU tabel DITULIS aplikasi ini — POOLDATA.D_CAUSE_OF_LOSS — dan LIMA objek lain hanya
-- DIBACA: V_D_CAUSE_OF_LOSS, V_D_CAUSE_OF_LOSS_BUSINESS, V_M_CAUSE_OF_LOSS, BUSINESS,
-- ditambah POOLDATA.M_SITE_DATABASE dan sequence D_CAUSE_SEQ untuk penomoran.
--
-- Kewenangan menulis D_CAUSE_OF_LOSS berpindah dari Pega ke Go saat modulnya lulus
-- gerbang 2. Selama Pega masih penulisnya, layar ini harus dijalankan dalam modus baca
-- saja di produksi (P-1, penulis tunggal per tabel).
--
-- Empat aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (D-20).
--   4. Tanpa pemanggilan stored procedure (D-02).
--
-- TIDAK ADA DELETE di berkas ini. Layar lamanya memang tidak punya tombolnya — lihat doc
-- comment detailpenyebab.Repo — dan D-66 melarang penghapusan fisik data bernilai bisnis.
-- Detail yang tidak lagi dipakai ditandai lewat STS_AKTIF.
--
--
-- ============================================================================
-- MENULIS KE TABEL, MEMBACA DARI VIEW — DAN KENAPA KEDUANYA BERBEDA
-- ============================================================================
--
-- Ini bagian yang paling mudah salah dibaca, jadi dinyatakan di muka.
--
-- `Database/PEGA_D_CAUSE_OF_LOSS.prc:22,33` membuktikan tabel fisiknya hanya punya DUA
-- kolom:
--
--   INSERT INTO POOLDATA.D_CAUSE_OF_LOSS(D_COL_ID,JSONDATA) VALUES(id_dcol_ins, …);
--   UPDATE POOLDATA.D_CAUSE_OF_LOSS SET JSONDATA = DataPega WHERE D_COL_ID = IDPega;
--
-- sedangkan `RDB List/QueryGetAllDataCauseOfLoss-SQL.xml:92-100` membaca ENAM kolom dari
-- `POOLDATA.V_D_CAUSE_OF_LOSS`. Yang kedua adalah view yang membentangkan JSONDATA menjadi
-- kolom.
--
-- Karena itu: seluruh SELECT di berkas ini menyebut VIEW, dan seluruh INSERT/UPDATE
-- menyebut TABEL. Menukarnya akan gagal — view berkolom itu hampir pasti tidak dapat
-- ditulis langsung.
--
-- KONSEKUENSI YANG WAJIB DISADARI SEBELUM MODUL DINYATAKAN LULUS. Definisi view-nya tidak
-- ada di export (`R-08`), sehingga PEMETAAN kunci JSON ke kolom view adalah REKONSTRUKSI.
-- Dasarnya: nama kolom view sama persis dengan nama properti page `TempDcol` yang
-- diserialkan `@GCNM.GetPageJSONString()`, dan itu pola yang sama yang sudah terbukti pada
-- Master Pasal Kerugian (`RDB List/GetDataCOLByPasalBisnis_Sql-SQL.xml` membaca JSONPASAL
-- dengan `json_value` memakai nama properti sebagai kunci).
--
-- Yang harus diuji terhadap basis data sungguhan: simpan satu baris lewat layar ini, lalu
-- pastikan keenam kolom view-nya terisi. Bila salah satu kosong, kunci JSON-nya berbeda
-- dari yang diduga di sini — dan itu TIDAK akan menghasilkan galat, hanya kolom kosong.
--
--
-- ============================================================================
-- LINI BISNIS: DITULIS DI DALAM JSON, DIBACA DARI VIEW TERPISAH
-- ============================================================================
--
-- `V_D_CAUSE_OF_LOSS_BUSINESS` **tidak pernah ditulis** oleh satu pun rule di export —
-- diperiksa atas seluruh *.xml dan *.prc, nol INSERT, nol UPDATE, nol DELETE terhadapnya.
-- Yang ada hanyalah pembacaan (`RDB List/GetLBUID_SQL-SQL.xml:33`).
--
-- Artinya ia view turunan atas `D_CAUSE_OF_LOSS.JSONDATA`, membentangkan `$.BISNISID[*]`
-- menjadi baris. Menyimpan lini bisnis karena itu dilakukan dengan menulis ULANG seluruh
-- dokumen JSON — bukan dengan menyisipkan baris ke view itu.


-- name: detail_list
--
-- Daftar Detail Penyebab Kerugian, dengan sebutan induknya.
--
-- # Rule aslinya HILANG dari export
--
-- Grid layar lama diisi `BrowseVDCauseOfLoss_RD`, dan Report Definition itu TIDAK ADA di
-- export (`R-16`) — lihat banner paket detailpenyebab. Kueri ini rekonstruksi dari dua
-- rule atas kelas yang sama:
--
--   RDB List/QueryGetAllDataCauseOfLoss-SQL.xml:92-100   keenam kolom, `order by DESCRIPTION`
--   Report Definition/SelectVDCauseOfLoss_RD-RD.xml      keenam properti, pyMaxRecords=500
--
-- Sub-kueri sebutan induk diambil dari `RDB List/BrowseCOLByBisnis_Sql-SQL.xml:64`:
--
--   (select col_desc from v_m_cause_of_loss where m_col_id=a.m_col_id) as "pyNote"
--
-- LEFT JOIN dipakai, bukan sub-kueri, karena keduanya berperilaku sama untuk satu baris
-- induk dan LEFT JOIN dapat memanfaatkan index. Barisnya tetap muncul meski induknya tidak
-- ada — tidak ada foreign key yang diketahui (`R-08`), sehingga baris yatim mungkin ada,
-- dan menyembunyikannya akan membuatnya hilang dari layar tanpa satu pun tanda.
--
-- # Ketiga penyaingnya opsional
--
-- Pola `(:n IS NULL OR …)` dipakai supaya satu teks kueri melayani seluruh kombinasi
-- penyaring. Merangkai WHERE secara dinamis akan menghasilkan teks kueri yang berbeda-beda
-- dan menghapus manfaat cache rencana eksekusi.
--
-- UPPER dipasang di KEDUA sisi pencarian supaya hasilnya tidak bergantung pada besar-kecil
-- huruf yang kebetulan tersimpan. `ESCAPE '\'` disebut eksplisit karena Oracle TIDAK punya
-- karakter pelolos bawaan pada LIKE; PostgreSQL memakai backslash sebagai bawaan, dan
-- menyebutkannya membuat keduanya berperilaku sama (D-20).
--
-- Penyaring lini bisnis memakai EXISTS, meniru `BrowseCOLByBisnis_Sql` apa adanya. Yang
-- TIDAK dibawa adalah `{ASIS:TempSearchBisnis.DESCRIPTION}` di dalamnya — potongan klausa
-- SQL yang dirangkai dari nilai property, persis utang teknis 4.5 pada Steering.
SELECT D.D_COL_ID,
       D.OLD_D_COL_ID,
       D.M_COL_ID,
       D.DESCRIPTION,
       D.LOSS_CODE,
       D.STS_AKTIF,
       M.COL_DESC
  FROM POOLDATA.V_D_CAUSE_OF_LOSS D
  LEFT JOIN POOLDATA.V_M_CAUSE_OF_LOSS M
    ON TRIM(M.M_COL_ID) = TRIM(D.M_COL_ID)
 WHERE (:1 IS NULL
        OR UPPER(D.DESCRIPTION) LIKE :2 ESCAPE '\'
        OR UPPER(D.LOSS_CODE) LIKE :3 ESCAPE '\'
        OR UPPER(D.D_COL_ID) LIKE :4 ESCAPE '\')
   AND (:5 IS NULL OR TRIM(D.M_COL_ID) = :6)
   AND (:7 IS NULL
        OR EXISTS (SELECT 1
                     FROM POOLDATA.V_D_CAUSE_OF_LOSS_BUSINESS B
                    WHERE TRIM(B.D_COL_ID) = TRIM(D.D_COL_ID)
                      AND TRIM(CAST(B.BISNISID AS VARCHAR(64))) = :8))
 ORDER BY D.DESCRIPTION

-- name: detail_get
--
-- Satu baris untuk dimuat ke form.
--
-- Asal: `Report Definition/SelectVDCauseOfLoss_RD-RD.xml`, yang dipanggil
-- `Activity/CNMSetDetailCauseOfLoss_act-Act.xml:857-884` lewat `pxShowReport`, lalu
-- keenam propertinya disalin ke `TempDcol` (`:1193-1320`).
--
-- TRIM pada penyaing: D_COL_ID bertipe teks dengan lebar yang tidak diketahui (`R-08`).
-- Bila ia CHAR alih-alih VARCHAR2, nilainya dipadatkan spasi di kanan dan perbandingan
-- langsung tidak akan pernah cocok.
SELECT D.D_COL_ID,
       D.OLD_D_COL_ID,
       D.M_COL_ID,
       D.DESCRIPTION,
       D.LOSS_CODE,
       D.STS_AKTIF,
       M.COL_DESC
  FROM POOLDATA.V_D_CAUSE_OF_LOSS D
  LEFT JOIN POOLDATA.V_M_CAUSE_OF_LOSS M
    ON TRIM(M.M_COL_ID) = TRIM(D.M_COL_ID)
 WHERE TRIM(D.D_COL_ID) = :1

-- name: detail_business_list
--
-- Lini bisnis yang menempel pada satu detail.
--
-- Asal: `RDB List/GetLBUID_SQL-SQL.xml:33` apa adanya —
--
--   select ID, NOTE as "Note"
--     from V_D_CAUSE_OF_LOSS_BUSINESS a, BUSINESS b
--    where a.BISNISID=b.ID and D_COL_ID={InputCOL.D_COL_ID}
--
-- Dua hal yang diubah, keduanya tanpa mengubah hasilnya:
--
--   1. Join lama berupa koma tanpa ON; di sini JOIN … ON, supaya syarat join terbaca
--      terpisah dari syarat penyaring (D-20 melarang `(+)`, dan koma-join adalah kerabat
--      dekatnya yang sama sulitnya dibaca).
--   2. `ORDER BY` ditambahkan. Kueri lama tidak punya, sehingga urutan barisnya tidak
--      ditentukan apa pun — dua pemuatan form yang sama dapat menampilkan urutan berbeda.
--
-- CAST pada perbandingan BISNISID: tipe kolomnya tidak diketahui (`R-08`). Bila ia angka,
-- perbandingan dengan teks memaksa konversi implisit yang gagal dengan ORA-01722 pada nilai
-- yang bukan angka, menggagalkan SELURUH pemuatan alih-alih satu baris.
SELECT B.BISNISID,
       S.NOTE
  FROM POOLDATA.V_D_CAUSE_OF_LOSS_BUSINESS B
  JOIN POOLDATA.BUSINESS S
    ON TRIM(CAST(S.ID AS VARCHAR(64))) = TRIM(CAST(B.BISNISID AS VARCHAR(64)))
 WHERE TRIM(B.D_COL_ID) = :1
 ORDER BY S.NOTE ASC

-- name: detail_document
--
-- Dokumen JSON mentah satu baris, dibaca dari TABEL fisiknya.
--
-- # Kenapa ia ada, padahal keenam kolomnya sudah dapat dibaca dari view
--
-- Karena MENYIMPAN berarti menulis ulang SELURUH dokumen, dan dokumen itu dapat memuat
-- kunci yang TIDAK dibentangkan view mana pun. `@GCNM.GetPageJSONString()` menyerialkan
-- seluruh properti page `TempDcol` apa adanya — termasuk apa pun yang pernah ditaruh di
-- sana oleh versi layar sebelumnya.
--
-- Menyusun dokumen baru hanya dari keenam kolom yang dikenal akan **membuang kunci yang
-- tidak dikenal itu, diam-diam, pada setiap penyimpanan**. Dengan membaca dokumen aslinya
-- lebih dulu lalu menimpa kunci yang memang disunting, kunci lain tetap utuh.
--
-- Ini bukan kehati-hatian yang mengada-ada: `TempDcol` terbukti juga menampung `pyNote` dan
-- `pyLabel` (`Activity/CNMInsertDetailCauseOfLoss_act-Act.xml:741,790`), dan keduanya ikut
-- terserialkan ke dalam JSON oleh langkah sebelumnya.
SELECT JSONDATA
  FROM POOLDATA.D_CAUSE_OF_LOSS
 WHERE TRIM(D_COL_ID) = :1

-- name: detail_insert
--
-- Asal: `Database/PEGA_D_CAUSE_OF_LOSS.prc:22`
--
--   INSERT INTO POOLDATA.D_CAUSE_OF_LOSS(D_COL_ID,JSONDATA)
--   VALUES(id_dcol_ins, replace(DataPega,'UnknownID',id_dcol_ins));
--
-- `replace(…,'UnknownID',…)` TIDAK dibawa, dan itu disengaja. Di sistem lama, ID belum
-- diterbitkan ketika dokumen JSON disusun, sehingga dokumen itu memuat teks harfiah
-- "UnknownID" yang lalu ditimpa procedure. Di sini ID sudah diterbitkan SEBELUM dokumen
-- disusun, sehingga penggantian itu tidak ada gunanya.
--
-- Dan ia lebih dari sekadar tidak berguna: `replace` bekerja atas SELURUH dokumen, bukan
-- hanya kunci ID. Sebuah deskripsi kerugian yang kebetulan memuat kata "UnknownID" akan
-- ikut tertimpa nomor baris. Tidak membawanya menutup cacat itu.
--
-- JSONDATA bertipe CLOB dan diikat sebagai teks biasa. Nilai yang lebih panjang dari 4000
-- karakter dapat ditolak driver dengan ORA-01461 bila ia mengikatnya sebagai VARCHAR2.
-- Belum dapat dibuktikan di lingkungan ini karena tidak ada Oracle untuk diuji; dicatat
-- sebagai hal yang WAJIB dicoba pada basis data sungguhan sebelum modul dinyatakan lulus,
-- dengan satu detail berdaftar lini bisnis panjang.
--
-- Tabelnya tidak punya kolom pencatat siapa dan kapan, sehingga jejak audit perubahan
-- master (D-28, modul S-5) belum dapat disandarkan padanya. Dicatat sebagai keterbatasan,
-- bukan ditambal dengan kolom yang dikarang: menambah kolom menuntut persetujuan Work Owner
-- dan pelaksanaan DBA (D-63).
INSERT INTO POOLDATA.D_CAUSE_OF_LOSS (D_COL_ID, JSONDATA)
VALUES (:1, :2)

-- name: detail_update
--
-- Asal: `Database/PEGA_D_CAUSE_OF_LOSS.prc:33`
--
--   UPDATE POOLDATA.D_CAUSE_OF_LOSS SET JSONDATA = DataPega WHERE D_COL_ID = IDPega;
--
-- D_COL_ID tidak pernah ikut di-SET: ia kunci baris, dan procedure lama pun hanya
-- memakainya sebagai penyaring WHERE. TRIM pada penyaring, alasannya sama seperti
-- detail_get.
UPDATE POOLDATA.D_CAUSE_OF_LOSS
   SET JSONDATA = :1
 WHERE TRIM(D_COL_ID) = :2

-- name: detail_exists
--
-- Memastikan sebuah D_COL_ID belum dipakai, sebelum menyisipkan.
--
-- Ia membaca TABEL, bukan view: yang ditanyakan adalah apakah kuncinya sudah terpakai, dan
-- kunci itu milik tabel. View dapat saja menyembunyikan baris — misalnya bila definisinya
-- menyaring dokumen yang tidak dapat diurai — dan baris tersembunyi yang kuncinya terpakai
-- tetap akan menolak penyisipan.
SELECT D_COL_ID
  FROM POOLDATA.D_CAUSE_OF_LOSS
 WHERE TRIM(D_COL_ID) = :1

-- name: detail_site
--
-- Kode situs, bagian pertama dari setiap D_COL_ID. Meniru
-- `Database/PEGA_D_CAUSE_OF_LOSS.prc:11` persis, termasuk pembandingnya yang berupa TEKS
-- '1' dan bukan angka.
--
-- Tabel yang sama dibaca Master Supplier, Master Bengkel, dan Master Status Klaim lewat
-- kueri masing-masing. Keempatnya sengaja tidak dipakai bersama: modul tidak saling
-- mengimpor, dan kueri bersama akan membuat perubahan di satu modul menyeret modul lain.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: detail_next_sequence
--
-- Urutan yang SAMA dengan yang dipakai procedure lama, supaya ID yang diterbitkan aplikasi
-- ini melanjutkan deret yang sudah ada dan tidak pernah bertabrakan dengan ID yang pernah
-- diterbitkan Pega.
--
-- `Database/PEGA_D_CAUSE_OF_LOSS.prc:19` menyebutnya tanpa skema (`D_CAUSE_SEQ`), sehingga
-- yang terpakai bergantung pada skema bawaan akun koneksi. Ia dibiarkan tanpa skema di sini
-- juga.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada ADR-0005 dan dengan Master Supplier,
-- satu-satunya tempat lain yang dibenarkan memuat percabangan dialek.
SELECT D_CAUSE_SEQ.NEXTVAL
  FROM DUAL

-- name: detail_check_table
--
-- Memastikan view utama ada dan dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
SELECT D_COL_ID
  FROM POOLDATA.V_D_CAUSE_OF_LOSS
 WHERE 1 = 0

-- name: detail_check_writable
--
-- Memastikan TABEL fisiknya ada dan dapat dibaca akun aplikasi.
--
-- Terpisah dari detail_check_table karena keduanya objek yang berbeda: hak baca atas view
-- TIDAK menyiratkan hak tulis atas tabel di belakangnya, dan modul ini butuh keduanya.
SELECT D_COL_ID
  FROM POOLDATA.D_CAUSE_OF_LOSS
 WHERE 1 = 0

-- name: detail_check_business_view
--
-- Memastikan view lini bisnis ada dan dapat dibaca akun aplikasi.
SELECT D_COL_ID
  FROM POOLDATA.V_D_CAUSE_OF_LOSS_BUSINESS
 WHERE 1 = 0

-- name: master_search
--
-- Daftar pilihan pada isian "ID Master Kerugian".
--
-- Asal: `Report Definition/BrowseVMCauseOfLoss_RD-RD.xml` atas kelas
-- `ASM-FW-GCNMFW-Int-V_M_CAUSE_OF_LOSS`, yang menyuapi autocomplete pada
-- `Section/BrowseDetailCauseOfLoss-Section.xml:1814-1819`. Nilai yang disetel `.M_COL_ID`,
-- yang ditampilkan `.COL_DESC`.
--
-- Report Definition itu tidak menyaring apa pun dan memakai `pyMaxRecords=500`: ia menarik
-- lima ratus baris pertama lalu menyaringnya di peramban. Di sini penyaringnya pindah ke
-- basis data, sehingga yang dikirim hanyalah yang benar-benar cocok.
--
-- `OLD_M_COL_ID` ikut dibaca Report Definition itu tetapi TIDAK dipakai isiannya, dan tidak
-- dibawa ke sini: kolom yang tidak ditampilkan dan tidak disimpan hanya menambah lebar
-- baris yang dikirim.
SELECT M_COL_ID,
       COL_DESC
  FROM POOLDATA.V_M_CAUSE_OF_LOSS
 WHERE UPPER(COL_DESC) LIKE :1 ESCAPE '\'
    OR TRIM(CAST(M_COL_ID AS VARCHAR(64))) = :2
 ORDER BY COL_DESC ASC
 FETCH NEXT 50 ROWS ONLY

-- name: master_get
--
-- Membaca sebutan sebuah Master Penyebab Kerugian dari kodenya.
--
-- Dipakai saat satu baris dibuka, supaya isian "ID Master Kerugian" dapat menampilkan
-- sebutannya tanpa menunggu pengguna mengetik di autocomplete.
SELECT M_COL_ID,
       COL_DESC
  FROM POOLDATA.V_M_CAUSE_OF_LOSS
 WHERE TRIM(CAST(M_COL_ID AS VARCHAR(64))) = :1

-- name: master_check_table
--
-- Memastikan master induk ada dan dapat dibaca akun aplikasi.
SELECT M_COL_ID
  FROM POOLDATA.V_M_CAUSE_OF_LOSS
 WHERE 1 = 0

-- name: business_search
--
-- Daftar pilihan lini bisnis pada grid "Bisnis".
--
-- Asal: autocomplete `.Note` pada `Section/BrowseDetailCauseOfLoss-Section.xml:3819`, yang
-- bersumber `Report Definition/BrowseBusiness_RD-RD.xml` atas kelas
-- `ASM-FW-GISFW-Int-BUSINESS` — yaitu POOLDATA.BUSINESS.
--
-- Kueri yang sama bentuknya dipakai Master Pasal Kerugian. Keduanya sengaja tidak berbagi
-- berkas: modul tidak saling mengimpor, dan satu kueri bersama akan membuat perubahan di
-- satu modul menyeret modul lain.
--
-- `CAST(ID AS VARCHAR(64))` pada cabang kedua: bila ID bertipe angka, perbandingan langsung
-- memaksa basis data mengubah kata kunci menjadi angka — dan kata kunci berupa nama akan
-- menghasilkan ORA-01722, menggagalkan SELURUH pencarian alih-alih hanya bagian itu. Tipe
-- ID sendiri belum diketahui (R-08).
SELECT ID,
       NOTE
  FROM POOLDATA.BUSINESS
 WHERE UPPER(NOTE) LIKE :1 ESCAPE '\'
    OR TRIM(CAST(ID AS VARCHAR(64))) = :2
 ORDER BY NOTE ASC
 FETCH NEXT 50 ROWS ONLY

-- name: business_check_table
--
-- Memastikan master lini bisnis ada dan dapat dibaca akun aplikasi.
SELECT ID
  FROM POOLDATA.BUSINESS
 WHERE 1 = 0
