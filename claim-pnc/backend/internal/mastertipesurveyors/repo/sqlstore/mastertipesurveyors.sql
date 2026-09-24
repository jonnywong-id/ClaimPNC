-- Kueri master Tipe Surveyor: POOLDATA.M_SURVEYORS.
--
-- # Tabel mana yang dibaca, dan tabel mana yang ditulis
--
-- Keduanya M_SURVEYORS — tabel dasarnya, bukan view POOLDATA.V_M_SURVEYORS. View itu
-- tetap ada dan tetap dibaca rule Pega selama masa paralel; aplikasi ini tidak
-- menyentuhnya. Membaca dari tabel dasar membuat apa yang kita tulis dan apa yang kita
-- baca kembali pasti sama, tanpa bergantung pada definisi view yang dimiliki pihak lain.
--
-- # Kenapa TIDAK ADA migrasi yang mengubah view di modul ini
--
-- Berbeda dari Master Status Klaim. Di sana view membaca label lewat
-- `JSON_VALUE(JSONDATA, '$.LSC_NOTE')`, sehingga menulis ke kolom menuntut view
-- didefinisikan ulang (migrasi 0002, langkah 3) — pernyataan yang menyentuh 23 rule Pega.
--
-- Di sini view SUDAH membaca kolom. Dibaca langsung dari ALL_VIEWS pada 2026-09-19:
--
--     SELECT M_SURVEY_ID, OLD_M_SURVEY_ID, DESCRIPTION FROM M_SURVEYORS
--
-- Artinya menulis ke kolom DESCRIPTION langsung terlihat oleh setiap rule Pega yang
-- membaca V_M_SURVEYORS, tanpa satu pun perubahan pada view. Migrasi 0003 karena itu
-- hanya membuat satu indeks unik, dan modul ini tetap bekerja bila migrasi itu belum
-- dijalankan.
--
-- # Kenapa tidak lagi lewat PEGA_M_SURVEYORS
--
-- `D-02` menetapkan logika stored procedure naik ke Go dan aplikasi tidak memanggil
-- procedure. `D-68` menambahkan alasan yang lebih keras: kontrak galat berbasis string
-- procedure itu tidak dapat dipakai — parameter keluarannya bernama `ErrMsg` tetapi pada
-- jalur BERHASIL ia berisi kalimat "Data Sudah Disimpan dengan ID : 1011", sehingga
-- pemanggil tidak dapat membedakan berhasil dari gagal tanpa membaca teks.
--
-- Keputusan Work Owner 2026-09-19: Go menjadi penulis tunggal M_SURVEYORS, dan yang
-- ditulis adalah kolom DESCRIPTION — JSON_DATA tidak dipakai lagi.
--
-- # Yang HARUS diketahui tentang JSON_DATA
--
-- Baris baru yang ditulis modul ini meninggalkan JSON_DATA bernilai NULL. Itu SAH:
-- constraint VALID_JSON_DATA1 berbunyi `JSON_DATA IS JSON (STRICT)`, dan pemeriksaan
-- semacam itu menghasilkan UNKNOWN — bukan FALSE — untuk NULL, sehingga barisnya
-- diterima.
--
-- Akibat yang disadari: empat baris lama memuat dokumen JSON, baris baru tidak. Tidak ada
-- satu pun rule Pega yang membaca kolom itu — satu-satunya yang menyentuhnya adalah
-- procedure PEGA_M_SURVEYORS, yang sejak sekarang tidak dipanggil siapa pun.
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding.

-- name: surveyor_type_list
SELECT M_SURVEY_ID,
       DESCRIPTION,
       OLD_M_SURVEY_ID
  FROM POOLDATA.M_SURVEYORS
 ORDER BY M_SURVEY_ID

-- name: surveyor_type_get
SELECT M_SURVEY_ID,
       DESCRIPTION,
       OLD_M_SURVEY_ID
  FROM POOLDATA.M_SURVEYORS
 WHERE M_SURVEY_ID = :1

-- name: surveyor_type_site
--
-- Kode situs, bagian pertama dari setiap M_SURVEY_ID. Meniru
-- `Database/PEGA_M_SURVEYORS.prc:11` persis, termasuk pembandingnya yang berupa teks '1'
-- dan bukan angka.
SELECT ID
  FROM POOLDATA.M_SITE_DATABASE
 WHERE CURRENT_SITE = '1'

-- name: surveyor_type_next_sequence
--
-- Urutan yang SAMA dengan yang dipakai procedure lama, supaya kode yang diterbitkan
-- aplikasi ini melanjutkan deret yang sudah ada dan tidak pernah bertabrakan dengan kode
-- yang pernah diterbitkan Pega.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada `ADR-0005`, satu-satunya tempat
-- lain yang dibenarkan memuat percabangan dialek.
SELECT POOLDATA.M_SURVEYORS_SEQ.NEXTVAL
  FROM DUAL

-- name: surveyor_type_insert
--
-- Dua kolom saja yang diisi.
--
-- OLD_M_SURVEY_ID sengaja tidak diisi: ia jejak penomoran sistem sebelumnya dan kosong
-- pada seluruh empat baris yang ada. JSON_DATA juga tidak diisi — lihat catatan di kepala
-- berkas ini.
INSERT INTO POOLDATA.M_SURVEYORS (M_SURVEY_ID, DESCRIPTION)
VALUES (:1, :2)

-- name: surveyor_type_update
--
-- Hanya DESCRIPTION yang diubah. M_SURVEY_ID tidak pernah berubah — tiga kueri Pega
-- mematok nilainya langsung (`m_survey_id in ('1002')` dan seterusnya) dan setiap baris
-- D_SURVEYORS menyimpannya — dan OLD_M_SURVEY_ID adalah jejak sejarah.
--
-- JSON_DATA sengaja TIDAK ikut dikosongkan pada baris lama: ia dibiarkan berisi nilai
-- terakhir yang ditulis Pega sebagai bahan pembanding. Sejak baris itu diubah di sini,
-- isinya menjadi usang — dan itu disadari, bukan terlewat.
UPDATE POOLDATA.M_SURVEYORS
   SET DESCRIPTION = :1
 WHERE M_SURVEY_ID = :2

-- name: surveyor_type_check_table
--
-- Memastikan ketiga kolom yang dipakai modul ini ada dan dapat dibaca akun aplikasi,
-- tanpa mengambil satu baris pun. Aman dijalankan terhadap produksi, dan dipakai untuk
-- membedakan dua sebab kegagalan yang tampak mirip: tabelnya tidak ada di portal itu,
-- versus tidak punya hak baca.
SELECT M_SURVEY_ID,
       DESCRIPTION,
       OLD_M_SURVEY_ID
  FROM POOLDATA.M_SURVEYORS
 WHERE 1 = 0
