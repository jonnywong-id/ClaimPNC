-- Kueri master Penyebab Kerugian: POOLDATA.M_CAUSE_OF_LOSS.
--
-- # Tabel mana yang dibaca, dan tabel mana yang ditulis
--
-- Keduanya M_CAUSE_OF_LOSS — tabel dasarnya, bukan view POOLDATA.V_M_CAUSE_OF_LOSS.
-- View itu tetap ada dan tetap dibaca 19 rule Pega selama masa paralel; aplikasi ini
-- tidak menyentuhnya. Membaca dari tabel dasar membuat apa yang kita tulis dan apa yang
-- kita baca kembali pasti sama, tanpa bergantung pada definisi view yang dimiliki pihak
-- lain.
--
-- # Kenapa tidak lagi lewat PEGA_M_CAUSE_OF_LOSS
--
-- `D-02` menetapkan logika stored procedure naik ke Go dan aplikasi tidak memanggil
-- procedure. `D-68` menambahkan alasan yang lebih keras: kontrak galat berbasis string
-- procedure itu tidak dapat dipakai — parameter keluarannya bernama `ErrMsg` tetapi pada
-- jalur BERHASIL ia berisi kalimat "Data Sudah Disimpan dengan ID : 1011", sehingga
-- pemanggil tidak dapat membedakan berhasil dari gagal tanpa membaca teks.
--
-- Keputusan Work Owner 2026-09-20: Go memakai KOLOM yang dibaca Pega dan tidak lagi
-- menulis JSON_DATA. Isinya pindah ke kolom COL_DESC lewat migrasi 0005.
--
-- # Satu hal yang HARUS dibaca sebelum menyetujui migrasi 0005
--
-- Berbeda dari Master Status Klaim, `P-1` di sini BELUM terpenuhi utuh: ada DUA layar
-- Pega yang menulis tabel ini lewat `RDB List/UpdateMCauseOfLoss-SQL.xml` —
-- `CauseOfLossInbox` (MENU_ID 20, yang digantikan modul ini) dan
-- `CauseOfLossInboxSimasOnline` (MENU_ID 21, yang BELUM digantikan). Selama layar kedua
-- masih hidup, baris yang ditulisnya hanya mengisi JSON_DATA dan kolomnya tetap kosong.
-- Rinciannya ada di berkas migrasi; ia bukan sesuatu yang layak ditambal diam-diam.
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding.

-- name: cause_of_loss_list
SELECT M_COL_ID,
       COL_DESC,
       OLD_M_COL_ID
  FROM POOLDATA.M_CAUSE_OF_LOSS

-- name: cause_of_loss_get
SELECT M_COL_ID,
       COL_DESC,
       OLD_M_COL_ID
  FROM POOLDATA.M_CAUSE_OF_LOSS
 WHERE M_COL_ID = :1

-- name: cause_of_loss_site
--
-- Kode situs, bagian pertama dari setiap M_COL_ID. Meniru
-- `Database/PEGA_M_CAUSE_OF_LOSS.prc:11` persis, termasuk pembandingnya yang berupa teks
-- '1' dan bukan angka.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: cause_of_loss_next_sequence
--
-- Urutan yang SAMA dengan yang dipakai procedure lama, supaya ID yang diterbitkan
-- aplikasi ini melanjutkan deret yang sudah ada dan tidak pernah bertabrakan dengan ID
-- yang pernah diterbitkan Pega.
--
-- Procedure lama menyebutnya `M_CAUSE_SEQ` tanpa nama skema, sehingga skemanya mengikuti
-- pemilik procedure. Di sini ia dikualifikasi POOLDATA mengikuti perlakuan yang sama pada
-- M_STS_CLAIM_SEQ di modul Master Status Klaim — dan itulah satu hal yang WAJIB
-- dikonfirmasi DBA sebelum migrasi 0005 dijalankan.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada `ADR-0005`, satu-satunya tempat
-- lain yang dibenarkan memuat percabangan dialek.
SELECT POOLDATA.M_CAUSE_SEQ.NEXTVAL
  FROM DUAL

-- name: cause_of_loss_insert
--
-- OLD_M_COL_ID sengaja tidak diisi: penomoran lama melekat pada baris warisan dan tidak
-- pernah diberikan pada golongan baru.
--
-- JSON_DATA juga sengaja tidak diisi. Konsekuensinya disadari dan dicatat di migrasi
-- 0005: baris yang ditulis aplikasi ini TIDAK punya dokumen JSON, sehingga siapa pun yang
-- membacanya lewat JSON — hanya procedure lama, yang sejak sekarang tidak dipanggil
-- siapa pun — akan mendapat NULL.
INSERT INTO POOLDATA.M_CAUSE_OF_LOSS (M_COL_ID, COL_DESC)
VALUES (:1, :2)

-- name: cause_of_loss_update
--
-- Hanya COL_DESC yang diubah. M_COL_ID tidak pernah berubah — mengubahnya akan memutus
-- setiap baris D_CAUSE_OF_LOSS yang bernaung di bawahnya — dan OLD_M_COL_ID adalah jejak
-- sejarah.
UPDATE POOLDATA.M_CAUSE_OF_LOSS
   SET COL_DESC = :1
 WHERE M_COL_ID = :2

-- name: cause_of_loss_check_table
--
-- Memastikan kolom yang dipakai modul ini sudah ada dan dapat dibaca akun aplikasi, tanpa
-- mengambil satu baris pun. Dipakai mode periksa untuk membedakan dua sebab kegagalan
-- yang tampak mirip: migrasi 0005 belum dijalankan versus tidak punya hak baca.
SELECT M_COL_ID,
       COL_DESC,
       OLD_M_COL_ID
  FROM POOLDATA.M_CAUSE_OF_LOSS
 WHERE 1 = 0

-- name: cause_of_loss_count_pending_json
--
-- Berapa baris yang dokumen JSON-nya ada tetapi kolom keterangannya masih kosong.
--
-- Angka ini menjawab satu pertanyaan yang tidak dapat dijawab kueri lain: apakah langkah
-- pemindahan isi pada migrasi 0005 sudah berjalan, DAN apakah layar Simas Online (MENU_ID
-- 21) sudah menulis baris baru yang belum ikut dipindahkan.
--
-- Nol berarti seluruh baris siap. Angka yang naik dari waktu ke waktu berarti penulis
-- kedua masih hidup dan barisnya perlu dipindahkan lagi — persis keadaan yang diperingatkan
-- di kepala berkas ini.
--
-- Dipakai HANYA oleh mode periksa, tidak pernah oleh jalur yang melayani pengguna.
SELECT COUNT(*)
  FROM POOLDATA.M_CAUSE_OF_LOSS
 WHERE JSON_DATA IS NOT NULL
   AND COL_DESC IS NULL
