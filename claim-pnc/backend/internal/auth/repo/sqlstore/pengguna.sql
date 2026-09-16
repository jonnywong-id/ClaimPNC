-- Kueri tabel CPNC_PENGGUNA.
--
-- Kolom selalu disebut namanya; SELECT * dilarang supaya kolom baru di basis data
-- tidak diam-diam mengubah perilaku aplikasi.

-- name: pengguna_ambil_by_identitas
SELECT IDENTITAS,
       JENIS,
       NAMA,
       LOGIN,
       EMAIL,
       PERUSAHAAN,
       CABANG,
       KODE_CABANG,
       JABATAN,
       OPERATOR_ID,
       AKTIF,
       DIBUAT_PADA,
       DIPERBARUI_PADA
  FROM CPNC_PENGGUNA
 WHERE IDENTITAS = :1

-- name: pengguna_perbarui
UPDATE CPNC_PENGGUNA
   SET JENIS           = :1,
       NAMA            = :2,
       LOGIN           = :3,
       EMAIL           = :4,
       PERUSAHAAN      = :5,
       CABANG          = :6,
       KODE_CABANG     = :7,
       JABATAN         = :8,
       DIPERBARUI_PADA = :9
 WHERE IDENTITAS = :10

-- name: pengguna_sisip
INSERT INTO CPNC_PENGGUNA (IDENTITAS, JENIS, NAMA, LOGIN, EMAIL, PERUSAHAAN, CABANG, KODE_CABANG, JABATAN, OPERATOR_ID, AKTIF, DIBUAT_PADA, DIPERBARUI_PADA)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13)

-- name: pengguna_periksa_tabel
--
-- Memastikan tabel ada dan dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
-- Dipakai mode periksa untuk memberi tahu apakah migrasi 0001 sudah dijalankan.
SELECT IDENTITAS
  FROM CPNC_PENGGUNA
 WHERE 1 = 0
