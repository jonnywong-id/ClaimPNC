-- Kueri modul Master Panel.
--
-- DUA tabel DITULIS aplikasi ini — POOLDATA.PANEL_HE dan POOLDATA.LOKASI_PANEL_HE —
-- ditambah POOLDATA.M_SITE_DATABASE dan sequence PANEL_HE_SEQ yang hanya DIBACA untuk
-- penomoran. Keduanya milik sistem lama (ADR-0004, penulis tunggal per tabel).
--
-- Kewenangan menulis kedua tabel itu berpindah dari Pega ke Go saat modulnya lulus
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
--
-- ============================================================================
-- KENAPA MENULIS KE PANEL_HE, PADAHAL PEGA MENULIS KE M_PANEL_HE
-- ============================================================================
--
-- Sistem lama memakai DUA tabel untuk satu master, dan itu pola dua-penyimpanan yang
-- `03-CURRENT-ARCHITECTURE.md` §3.2 catat sebagai R-10:
--
--   tulis  Activity/CNMUpdatePanelHE_act  ->  InputData.NAME := @GCNM.GetPageJSONString()
--          RDB List/UpdatePanel_HE-SQL    ->  POOLDATA.PEGA_M_PANEL_HE(Datapega, IDPanel, out)
--          Database/PEGA_M_PANEL_HE.prc:24 -> INSERT INTO POOLDATA.M_PANEL_HE(ID, JSONDATA)
--
--   baca   Report Definition/BrowseMasterPanel_HE_RD ->  POOLDATA.PANEL_HE, 14 kolom
--          RDB List/ValidationMasterPanel            ->  pooldata.panel_he
--          RDB List/CountMasterPanelManager          ->  POOLDATA.PANEL_HE
--          RDB List/GetIDDokumenPanel                ->  POOLDATA.PANEL_HE
--          RDB List/GetLokasiSisiPanel               ->  pooldata.lokasi_panel_he
--
-- Menulisnya lewat procedure dilarang D-02. Yang tersisa adalah dua pilihan, dan hanya
-- satu yang dapat dikerjakan tanpa menebak:
--
--   (a) menulis dokumen JSON ke M_PANEL_HE.JSONDATA dengan SQL biasa.
--       TIDAK DAPAT DIKERJAKAN: nama kunci JSON-nya diterbitkan fungsi
--       `@GCNM.GetPageJSONString()`, dan `Function/GetPageJSONString-Function.xml` HANYA
--       memuat tanda tangannya — badan fungsinya tidak ikut di export (R-16). Setiap
--       kunci yang ditulis akan menjadi tebakan.
--
--   (b) menulis kolom bernama ke PANEL_HE dan LOKASI_PANEL_HE.
--       DAPAT DIKERJAKAN untuk PANEL_HE: keempat belas kolomnya terbaca lengkap dari
--       `BrowseMasterPanel_HE_RD-RD.xml`. Untuk LOKASI_PANEL_HE terbaca sebagian —
--       lihat catatan kolom NAMA di bawah.
--
-- (b) yang dipakai. Perlakuannya sama dengan Master Bengkel dan Master Status Klaim, yang
-- menghadapi keluarga procedure PEGA_M_* yang sama persis dan memutuskan hal yang sama
-- (D-02, D-68): Go menjadi penulis tunggal dan berhenti menulis JSONDATA.
--
--
-- ============================================================================
-- KOLOM NAMA PADA LOKASI_PANEL_HE — ASUMSI YANG DISADARI
-- ============================================================================
--
-- Tabel anak punya SEKURANGNYA empat kolom, dan hanya tiga yang artinya pasti:
--
--   ID_PANEL      kunci induk        RDB List/GetLokasiSisiPanel-SQL.xml
--   LOKASI_PANEL  nama lokasi        idem, dialiaskan "NAME"
--   SISI_PANEL    sandi sisi         idem, dialiaskan "STS_SISI"
--   NAMA          ???                RDB List/GetDataSisiPanel-SQL.xml
--
-- Yang terakhir hanya muncul sebagai PENYARING, tidak pernah sebagai kolom yang dibaca:
--
--   select sisi_panel from pooldata.lokasi_panel_he
--    where id_panel = {...} and nama = {...}
--
-- Pemanggilnya `Activity/GetSisiPanel-Act.xml`, yang hanya dipakai lima section modul
-- **Grouping Sparepart HE** — modul lain, di luar lingkup migrasi ini. Nilai yang
-- dikirimkannya adalah sebuah nama lokasi.
--
-- Yang DITULIS di berkas ini: NAMA diisi nilai yang SAMA dengan LOKASI_PANEL. Itu satu-
-- satunya pembacaan yang konsisten dengan kedua kueri, dan satu-satunya yang menjaga
-- modul Grouping Sparepart tetap menemukan barisnya. Membiarkannya NULL akan MEMATIKAN
-- modul itu tanpa satu pun pesan galat — kelas kegagalan yang paling mahal ditemukan.
--
-- Asumsinya DAPAT DIPERIKSA, dan pemeriksaannya sudah terpasang: `claimpnc -periksa`
-- menghitung baris yang NAMA-nya berbeda dari LOKASI_PANEL. Nol berarti asumsi ini
-- benar; selain nol berarti keduanya memang dua hal berbeda, dan berkas ini harus
-- diperbaiki SEBELUM jalur tulis diaktifkan di produksi. Lihat checkPanel pada
-- cmd/claimpnc.
--
--
-- ============================================================================
-- HAPUS-LALU-SISIP-ULANG PADA TABEL ANAK
-- ============================================================================
--
-- Penyimpanan mengganti SELURUH baris lokasi sebuah panel: dibuang, lalu disisipkan
-- ulang dari daftar yang dikirim layar. Itu pola yang sama dengan yang dipakai sistem
-- lama (`Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503` melakukannya pada 12 tabel),
-- dan ia dipilih karena baris anak TIDAK PUNYA KUNCI SENDIRI — tidak ada kolom surrogate
-- yang memungkinkan satu baris dikenali lintas penyimpanan.
--
-- INI BERTENTANGAN DENGAN D-66, dan pertentangannya dinyatakan di sini alih-alih
-- disembunyikan. `D-66` menetapkan soft delete menyeluruh, dan `ADR-0013` mencatat bahwa
-- pengganti pola hapus-lalu-sisip-ulang BELUM DIPUTUSKAN. Work Owner memilih "jalankan
-- as is" pada 2026-09-19 — perlakuan yang sama dengan Master Pasal Kerugian, layar
-- pertama yang menghapus permanen.
--
-- Yang menahan akibatnya: keduanya berjalan di dalam SATU transaksi, sehingga panel tidak
-- pernah berada dalam keadaan "lokasi lama sudah dibuang, yang baru belum masuk". Sistem
-- lama tidak menjamin itu.
--
--
-- CATATAN TRIM pada setiap penyaring kunci. Tipe kolom kedua tabel belum diketahui
-- (R-08). Bila ID_PANEL bertipe CHAR berlebar tetap, nilainya dipadatkan spasi tanpa
-- tanda apa pun; Oracle membandingkan CHAR dengan CHAR secara blank-padded, sehingga
-- kueri lama yang MERANGKAI nilainya tetap cocok. Parameter binding bertipe VARCHAR2, dan
-- perbandingan CHAR dengan VARCHAR2 memakai non-padded comparison: "ABC " tidak sama
-- dengan "ABC", dan barisnya tidak ketemu. Menyalin `= :1` apa adanya karena itu justru
-- MENGUBAH perilaku. TRIM benar untuk kedua kemungkinan tipe. Biayanya index atas kolom
-- itu tidak terpakai; dapat diterima pada tabel master berbaris sedikit, dan TIDAK boleh
-- ditiru pada tabel besar.


-- name: panel_list
--
-- Asal: Report Definition/BrowseMasterPanel_HE_RD-RD.xml, dipakai ketiga tab
-- `Section/BrowsePanelHE-Section.xml` yang hanya berbeda pada nilai APPROVAL-nya.
--
-- Kelima belas kolomnya disebut pada urutan yang SAMA dengan panel_get dan
-- panel_find_by_name — satu fungsi scanRow membaca ketiganya berdasarkan POSISI, dan satu
-- kolom yang bergeser akan menaruh alasan penolakan ke kolom status tanpa satu pun galat.
-- `TestReaderQueriesShareColumnOrder` yang menjaganya.
--
-- DOKUMENID ikut dibaca meski `BrowseMasterPanel_HE_RD` tidak menyebutnya: ia dibaca
-- terpisah oleh `RDB List/GetIDDokumenPanel-SQL.xml` atas tabel yang sama. Membacanya
-- sekaligus menghindarkan satu perjalanan basis data per baris, dan membuat penyimpanan
-- dapat menulis balik nilainya apa adanya alih-alih mengosongkannya.
--
-- ORDER BY ID_PANEL DESC meniru `pySortType=DESC` pada report definition lama APA ADANYA:
-- panel yang paling baru diterbitkan berada di puncak daftar.
--
-- Sempat diganti `ORDER BY NAME` dengan alasan "lebih mudah dicari mata". Itu dibatalkan
-- setelah layar Pega yang sebenarnya dibandingkan: urutan menurun berdasarkan ID adalah
-- yang dilihat petugas hari ini, dan `D-13` menuntut tata letak yang sama — bukan tata
-- letak yang menurut kami lebih baik. Pencarian dan pengurutan per kolom tetap tersedia
-- di layar, sehingga tidak ada yang hilang karenanya.
--
-- `pyMaxRecords=500` pada report definition lama TIDAK direplikasi. Ia memotong daftar di
-- 500 baris tanpa satu pun tanda di layar; penggantinya adalah penyaring kata kunci pada
-- panel_list_search.
SELECT ID_PANEL,
       NAME,
       STS_REPAIR,
       STS_EDIT_QTY,
       STS_PREMIUM_REPAIR,
       STS_PECAH,
       STS_STICKER,
       STS_SISI,
       STS_RUSAK_PARAH,
       STS_AKTIF,
       EXCLUSION_C,
       STS_APPROVAL,
       ALASAN_TOLAK,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.PANEL_HE
 WHERE TRIM(APPROVAL) = :1
 ORDER BY ID_PANEL DESC

-- name: panel_list_search
--
-- Sama dengan panel_list, ditambah penyaring kata kunci atas nama panel.
--
-- KUERI TERSENDIRI, bukan satu kueri yang klausanya ditempel. Sistem lama menempuh cara
-- yang kedua di banyak tempat lewat pola `{ASIS:...}` — nilai dirangkai langsung ke teks
-- SQL — dan `08-TECHNICAL-STRATEGY.md` §4.3 melarangnya tanpa perkecualian.
--
-- UPPER dipasang pada KOLOMNYA juga, bukan hanya pada kata kuncinya: nama panel tersimpan
-- dengan besar-kecil huruf apa adanya, dan pencarian yang hanya meng-uppercase kata kunci
-- tidak akan pernah menemukan baris yang namanya huruf kecil.
--
-- `ESCAPE '\'` disebut eksplisit karena Oracle TIDAK punya karakter pelolos bawaan pada
-- LIKE. PostgreSQL memakai backslash sebagai bawaan; menyebutkannya eksplisit membuat
-- kedua basis data berperilaku sama (D-20).
SELECT ID_PANEL,
       NAME,
       STS_REPAIR,
       STS_EDIT_QTY,
       STS_PREMIUM_REPAIR,
       STS_PECAH,
       STS_STICKER,
       STS_SISI,
       STS_RUSAK_PARAH,
       STS_AKTIF,
       EXCLUSION_C,
       STS_APPROVAL,
       ALASAN_TOLAK,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.PANEL_HE
 WHERE TRIM(APPROVAL) = :1
   AND UPPER(NAME) LIKE :2 ESCAPE '\'
 ORDER BY ID_PANEL DESC

-- name: panel_get
--
-- Kolomnya sama dan pada urutan yang sama dengan panel_list.
SELECT ID_PANEL,
       NAME,
       STS_REPAIR,
       STS_EDIT_QTY,
       STS_PREMIUM_REPAIR,
       STS_PECAH,
       STS_STICKER,
       STS_SISI,
       STS_RUSAK_PARAH,
       STS_AKTIF,
       EXCLUSION_C,
       STS_APPROVAL,
       ALASAN_TOLAK,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.PANEL_HE
 WHERE TRIM(ID_PANEL) = :1

-- name: panel_find_by_name
--
-- Padanan `RDB List/ValidationMasterPanel-SQL.xml`:
--
--   select id_panel as "ID_PANEL" from pooldata.panel_he
--    where upper(trim(name)) = upper(trim({TempInputPanelHE.CaseID}))
--
-- Yang berubah hanyalah cara nilainya sampai — parameter binding menggantikan perangkaian
-- teks — dan kolom yang dibaca, supaya pemanggil dapat menyebut panel mana yang memakai
-- nama itu alih-alih hanya mengatakan "sudah dipakai".
--
-- Kolomnya sama dan pada urutan yang sama dengan panel_list.
SELECT ID_PANEL,
       NAME,
       STS_REPAIR,
       STS_EDIT_QTY,
       STS_PREMIUM_REPAIR,
       STS_PECAH,
       STS_STICKER,
       STS_SISI,
       STS_RUSAK_PARAH,
       STS_AKTIF,
       EXCLUSION_C,
       STS_APPROVAL,
       ALASAN_TOLAK,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.PANEL_HE
 WHERE UPPER(TRIM(NAME)) = :1

-- name: panel_lock_by_name
--
-- Dipakai di dalam transaksi penambahan. FOR UPDATE mengunci baris yang SUDAH ADA
-- sehingga dua penambahan atas nama yang sama tidak dapat berjalan berdampingan.
--
-- SEBERAPA JAUH LUBANGNYA TERTUTUP: tidak sepenuhnya. FOR UPDATE tidak dapat mengunci
-- baris yang belum ada, sehingga dua penambahan yang sama-sama menemukan nol baris tetap
-- lolos berdampingan. Yang benar-benar menutupnya adalah constraint unik pada NAME, dan
-- itu menunggu DDL (R-08) serta prosedur perubahan skema (D-63).
SELECT ID_PANEL
  FROM POOLDATA.PANEL_HE
 WHERE UPPER(TRIM(NAME)) = :1
 FOR UPDATE

-- name: panel_insert
--
-- Kelima belas kolomnya pada urutan yang sama dengan insertArguments. Urutan itu WAJIB
-- sama; `TestInsertArgumentsMatchColumnOrder` yang menjaganya.
INSERT INTO POOLDATA.PANEL_HE
       (ID_PANEL, NAME, STS_REPAIR, STS_EDIT_QTY, STS_PREMIUM_REPAIR,
        STS_PECAH, STS_STICKER, STS_SISI, STS_RUSAK_PARAH, STS_AKTIF,
        EXCLUSION_C, STS_APPROVAL, ALASAN_TOLAK, DOKUMENID, APPROVAL)
VALUES (:1, :2, :3, :4, :5,
        :6, :7, :8, :9, :10,
        :11, :12, :13, :14, :15)

-- name: panel_update
--
-- ID_PANEL TIDAK disebut sebagai kolom yang ditulis — ia penyaring WHERE, dan berada di
-- posisi terakhir pada updateArguments. Kunci baris tidak pernah berpindah.
UPDATE POOLDATA.PANEL_HE
   SET NAME = :1,
       STS_REPAIR = :2,
       STS_EDIT_QTY = :3,
       STS_PREMIUM_REPAIR = :4,
       STS_PECAH = :5,
       STS_STICKER = :6,
       STS_SISI = :7,
       STS_RUSAK_PARAH = :8,
       STS_AKTIF = :9,
       EXCLUSION_C = :10,
       STS_APPROVAL = :11,
       ALASAN_TOLAK = :12,
       DOKUMENID = :13,
       APPROVAL = :14
 WHERE TRIM(ID_PANEL) = :15

-- name: panel_set_status
--
-- Satu pernyataan per baris, bukan satu pernyataan dengan daftar kunci yang panjangnya
-- berubah-ubah. Daftar `IN (:1, :2, …)` yang panjangnya mengikuti jumlah baris menghasilkan
-- teks SQL yang berbeda setiap kali dipanggil — setiap bentuk menempati satu slot pada
-- shared pool Oracle, dan tabel master yang diputuskan borongan akan menghasilkan puluhan
-- bentuk berbeda dari satu operasi yang sama.
--
-- `AND TRIM(APPROVAL) <> :1` membuat baris yang sudah berstatus itu TIDAK terhitung
-- sebagai berubah — itulah yang membuat jumlah yang dilaporkan ke layar bermakna.
--
-- ALASAN_TOLAK ikut ditulis. Pemanggil yang mengosongkannya pada keputusan selain Reject;
-- lihat usecase.Service.Decide.
UPDATE POOLDATA.PANEL_HE
   SET APPROVAL = :1,
       ALASAN_TOLAK = :2
 WHERE TRIM(ID_PANEL) = :3
   AND TRIM(APPROVAL) <> :1

-- name: panel_location_by_status
--
-- Seluruh baris lokasi milik panel pada satu status, dibaca SEKALI untuk seluruh daftar.
--
-- Bukan satu kueri per panel. Pola yang kedua adalah N+1 yang
-- `15-NFR-PERFORMANCE-SCALABILITY.md` §3.1 letakkan di peringkat ketiga hambatan nyata,
-- dan pada layar yang memuat ratusan panel ia berarti ratusan perjalanan basis data untuk
-- menggambar satu tabel.
--
-- ORDER BY menentukan urutan lokasi di dalam setiap panel, dan ia ikut menentukan apa
-- yang dilihat pengguna: daftar lokasi yang urutannya berubah antar pemuatan membuat
-- layar tampak berkedip tanpa sebab.
SELECT L.ID_PANEL,
       L.LOKASI_PANEL,
       L.SISI_PANEL
  FROM POOLDATA.LOKASI_PANEL_HE L
  JOIN POOLDATA.PANEL_HE P
    ON TRIM(P.ID_PANEL) = TRIM(L.ID_PANEL)
 WHERE TRIM(P.APPROVAL) = :1
 ORDER BY L.ID_PANEL, L.LOKASI_PANEL, L.SISI_PANEL

-- name: panel_location_by_status_search
--
-- Sama dengan panel_location_by_status, dengan penyaring kata kunci yang SAMA PERSIS
-- dengan panel_list_search.
--
-- Kesamaan itu penting: bila kedua penyaring berbeda, daftar induk dan daftar anaknya
-- akan berisi panel yang berbeda, dan sebagian baris akan tampil tanpa lokasinya tanpa
-- satu pun tanda bahwa ada yang salah.
SELECT L.ID_PANEL,
       L.LOKASI_PANEL,
       L.SISI_PANEL
  FROM POOLDATA.LOKASI_PANEL_HE L
  JOIN POOLDATA.PANEL_HE P
    ON TRIM(P.ID_PANEL) = TRIM(L.ID_PANEL)
 WHERE TRIM(P.APPROVAL) = :1
   AND UPPER(P.NAME) LIKE :2 ESCAPE '\'
 ORDER BY L.ID_PANEL, L.LOKASI_PANEL, L.SISI_PANEL

-- name: panel_location_get
--
-- Baris lokasi satu panel. Padanan `RDB List/GetLokasiSisiPanel-SQL.xml`:
--
--   select LOKASI_PANEL as "NAME", SISI_PANEL as "STS_SISI"
--     from pooldata.lokasi_panel_he WHERE ID_PANEL = {TempStsClaim.ID_PANEL}
--
-- Alias `"STS_SISI"` pada kueri asli TIDAK dibawa: nama itu sudah dipakai kolom lain pada
-- tabel INDUK dengan arti yang sama sekali berbeda. Lihat catatan paket domain.
SELECT LOKASI_PANEL,
       SISI_PANEL
  FROM POOLDATA.LOKASI_PANEL_HE
 WHERE TRIM(ID_PANEL) = :1
 ORDER BY LOKASI_PANEL, SISI_PANEL

-- name: panel_location_clear
--
-- Membuang seluruh baris lokasi sebuah panel, selalu di dalam transaksi yang sama dengan
-- penyisipan penggantinya. Lihat banner berkas ini untuk alasan pola ini dipakai dan apa
-- pertentangannya dengan D-66.
DELETE FROM POOLDATA.LOKASI_PANEL_HE
 WHERE TRIM(ID_PANEL) = :1

-- name: panel_location_insert
--
-- NAMA diisi nilai yang SAMA dengan LOKASI_PANEL. Itu asumsi yang disadari; lihat banner
-- berkas ini, dan pemeriksaannya di `claimpnc -periksa`.
INSERT INTO POOLDATA.LOKASI_PANEL_HE
       (ID_PANEL, LOKASI_PANEL, SISI_PANEL, NAMA)
VALUES (:1, :2, :3, :4)

-- name: panel_count_pending
SELECT COUNT(ID_PANEL)
  FROM POOLDATA.PANEL_HE
 WHERE TRIM(APPROVAL) = :1

-- name: panel_count_all
SELECT COUNT(ID_PANEL)
  FROM POOLDATA.PANEL_HE

-- name: panel_check_table
--
-- Tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi. Yang
-- dibuktikannya hanyalah tabelnya ada dan kelima belas kolomnya dapat dibaca akun
-- aplikasi.
SELECT ID_PANEL,
       NAME,
       STS_REPAIR,
       STS_EDIT_QTY,
       STS_PREMIUM_REPAIR,
       STS_PECAH,
       STS_STICKER,
       STS_SISI,
       STS_RUSAK_PARAH,
       STS_AKTIF,
       EXCLUSION_C,
       STS_APPROVAL,
       ALASAN_TOLAK,
       DOKUMENID,
       APPROVAL
  FROM POOLDATA.PANEL_HE
 WHERE 1 = 0

-- name: panel_check_location_table
--
-- Keempat kolom disebut, TERMASUK NAMA — justru kolom itulah yang perlu dipastikan ada
-- sebelum jalur tulis dipakai.
SELECT ID_PANEL,
       LOKASI_PANEL,
       SISI_PANEL,
       NAMA
  FROM POOLDATA.LOKASI_PANEL_HE
 WHERE 1 = 0

-- name: panel_count_location
SELECT COUNT(ID_PANEL)
  FROM POOLDATA.LOKASI_PANEL_HE

-- name: panel_count_location_orphan
--
-- Baris lokasi yang induknya tidak ada.
--
-- Ia tidak dapat terjadi lewat modul ini — penyisipannya selalu berdampingan dengan
-- induknya di dalam satu transaksi — tetapi dapat sudah ada di data warisan, dan
-- jumlahnya menentukan apakah tabel anak ini dapat dipercaya sebagai cerminan tabel
-- induknya.
SELECT COUNT(L.ID_PANEL)
  FROM POOLDATA.LOKASI_PANEL_HE L
 WHERE NOT EXISTS (
       SELECT 1
         FROM POOLDATA.PANEL_HE P
        WHERE TRIM(P.ID_PANEL) = TRIM(L.ID_PANEL))

-- name: panel_count_location_name_mismatch
--
-- Baris yang NAMA-nya BERBEDA dari LOKASI_PANEL.
--
-- Inilah pemeriksaan yang menentukan apakah asumsi pada banner berkas ini benar. Nol
-- berarti keduanya memang memuat hal yang sama, dan panel_location_insert boleh dipakai
-- apa adanya. Selain nol berarti keduanya dua hal berbeda, dan berkas ini harus diperbaiki
-- SEBELUM jalur tulis diaktifkan di produksi.
--
-- Baris yang salah satunya NULL ikut terhitung sebagai berbeda: `NULL <> 'X'` bernilai
-- UNKNOWN dan tidak akan pernah lolos WHERE, sehingga keadaan itu harus disebut sendiri.
SELECT COUNT(ID_PANEL)
  FROM POOLDATA.LOKASI_PANEL_HE
 WHERE TRIM(LOKASI_PANEL) <> TRIM(NAMA)
    OR (LOKASI_PANEL IS NULL AND NAMA IS NOT NULL)
    OR (LOKASI_PANEL IS NOT NULL AND NAMA IS NULL)

-- name: panel_check_json_mirror
--
-- Tabel JSON milik Pega. Perbandingan jumlah barisnya dengan PANEL_HE adalah cara
-- termurah mengetahui apakah keduanya satu sumber; lihat banner berkas ini.
SELECT ID
  FROM POOLDATA.M_PANEL_HE
 WHERE 1 = 0

-- name: panel_count_json_mirror
SELECT COUNT(ID)
  FROM POOLDATA.M_PANEL_HE

-- name: panel_site
--
-- Kode situs, dibaca persis seperti `Database/PEGA_M_PANEL_HE.prc:12`:
--
--   SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
--
-- Pembandingnya TEKS '1', bukan angka 1, dan itu ditiru apa adanya: bila kolomnya bertipe
-- teks, perbandingan dengan angka akan memaksa konversi implisit yang perilakunya berbeda
-- antar basis data.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: panel_next_sequence
--
-- Sequence yang SAMA dengan yang dipakai procedure lama, supaya ID yang diterbitkan
-- aplikasi ini melanjutkan deret yang sudah ada dan tidak pernah bertabrakan dengan ID
-- yang pernah diterbitkan Pega.
--
-- `Database/PEGA_M_PANEL_HE.prc:21` menyebutnya tanpa skema (`PANEL_HE_SEQ`), sehingga
-- yang terpakai bergantung pada skema bawaan akun koneksi — dan itu berbeda antar
-- lingkungan. Di sini ia dilengkapi POOLDATA, skema yang sama dengan tabel yang disisipi
-- procedure itu.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada ADR-0005, satu-satunya tempat lain
-- yang dibenarkan memuat percabangan dialek.
SELECT POOLDATA.PANEL_HE_SEQ.NEXTVAL
  FROM DUAL
