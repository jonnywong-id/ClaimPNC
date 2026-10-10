-- Kueri master Status Klaim: POOLDATA.M_STS_CLAIM.
--
-- # Tabel mana yang dibaca, dan tabel mana yang ditulis
--
-- Keduanya M_STS_CLAIM — tabel dasarnya, bukan view POOLDATA.V_STS_CLAIM. View itu
-- tetap ada dan tetap dibaca 23 rule Pega selama masa paralel; aplikasi ini tidak
-- menyentuhnya. Membaca dari tabel dasar membuat apa yang kita tulis dan apa yang kita
-- baca kembali pasti sama, tanpa bergantung pada definisi view yang dimiliki pihak lain.
--
-- # Kenapa tidak lagi lewat PEGA_M_STS_CLAIM
--
-- `D-02` menetapkan logika stored procedure naik ke Go dan aplikasi tidak memanggil
-- procedure. `D-68` menambahkan alasan yang lebih keras: kontrak galat berbasis string
-- procedure itu tidak dapat dipakai — parameter keluarannya bernama `ErrMsg` tetapi
-- pada jalur BERHASIL ia berisi kalimat "Data Sudah Disimpan dengan ID : 1167", sehingga
-- pemanggil tidak dapat membedakan berhasil dari gagal tanpa membaca teks.
--
-- Keputusan Work Owner 2026-09-17: Go menjadi penulis tunggal M_STS_CLAIM, tidak lagi
-- menulis JSONDATA, dan isinya pindah ke kolom LSC_NOTE dan OLD_LSC_ID yang ditambahkan
-- migrasi 0002.
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding.

-- name: status_klaim_daftar
SELECT LSC_ID,
       LSC_NOTE,
       OLD_LSC_ID
  FROM POOLDATA.M_STS_CLAIM
 ORDER BY LSC_ID

-- name: status_klaim_ambil
SELECT LSC_ID,
       LSC_NOTE,
       OLD_LSC_ID
  FROM POOLDATA.M_STS_CLAIM
 WHERE LSC_ID = :1

-- name: status_klaim_situs
--
-- Kode situs, bagian pertama dari setiap LSC_ID. Meniru
-- `Database/PEGA_M_STS_CLAIM.prc:11` persis, termasuk pembandingnya yang berupa teks
-- '1' dan bukan angka.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: status_klaim_urutan_berikutnya
--
-- Urutan yang SAMA dengan yang dipakai procedure lama, supaya kode yang diterbitkan
-- aplikasi ini melanjutkan deret yang sudah ada dan tidak pernah bertabrakan dengan
-- kode yang pernah diterbitkan Pega.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia
-- tidak terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada `ADR-0005`, satu-satunya tempat
-- lain yang dibenarkan memuat percabangan dialek.
SELECT POOLDATA.M_STS_CLAIM_SEQ.NEXTVAL
  FROM DUAL

-- name: status_klaim_sisip
--
-- OLD_LSC_ID sengaja tidak diisi: penomoran lama `01`–`11` hanya melekat pada sebelas
-- kode pertama dan tidak pernah diberikan pada status baru.
INSERT INTO POOLDATA.M_STS_CLAIM (LSC_ID, LSC_NOTE)
VALUES (:1, :2)

-- name: status_klaim_perbarui
--
-- Hanya LSC_NOTE yang diubah. LSC_ID tidak pernah berubah — mengubahnya akan memutus
-- setiap klaim lama yang menyimpan kode itu — dan OLD_LSC_ID adalah jejak sejarah.
UPDATE POOLDATA.M_STS_CLAIM
   SET LSC_NOTE = :1
 WHERE LSC_ID = :2

-- name: status_klaim_periksa_tabel
--
-- Memastikan kolom baru migrasi 0002 sudah ada dan dapat dibaca akun aplikasi, tanpa
-- mengambil satu baris pun. Dipakai mode periksa untuk membedakan dua sebab kegagalan
-- yang tampak mirip: migrasi belum dijalankan versus tidak punya hak baca.
SELECT LSC_ID,
       LSC_NOTE,
       OLD_LSC_ID
  FROM POOLDATA.M_STS_CLAIM
 WHERE 1 = 0
