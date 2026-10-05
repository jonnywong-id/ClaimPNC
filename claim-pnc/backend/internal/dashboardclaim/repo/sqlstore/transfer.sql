-- Kueri permintaan TRANSFER penugasan klaim.
--
-- Tabelnya `POOLDATA.CPNC_PERMINTAAN_TRANSFER`, milik aplikasi ini sendiri — dibuat migrasi
-- `0014`, yang BELUM dijalankan DBA di lingkungan mana pun.
--
-- Ia satu-satunya berkas .sql di modul ini yang MENULIS. Seluruh berkas lain hanya membaca,
-- karena tabelnya milik Pega selama masa paralel (`P-1`). Yang ditulis di sini bukan klaim
-- melainkan PERMINTAAN atas klaim.

-- name: transfer_insert
-- Mencatat satu permintaan transfer.
--
-- Status tidak dikirim sebagai parameter: aplikasi hanya pernah menulis 'menunggu', dan
-- menjadikannya parameter membuka kemungkinan menulis 'dijalankan' dari sini — padahal yang
-- berhak menuliskannya adalah pelaksana.
INSERT INTO POOLDATA.CPNC_PERMINTAAN_TRANSFER (
    ID, LINGKUP, CASE_ID, NOMOR_KLAIM,
    OPERATOR_ASAL, OPERATOR_TUJUAN, TIPE_PENGGUNA, ALASAN,
    STATUS, PEMOHON, PEMOHON_NAMA
) VALUES (
    :1, :2, :3, :4,
    :5, :6, :7, :8,
    'menunggu', :9, :10
)

-- name: transfer_pending
-- Membaca permintaan yang masih menunggu atas sekumpulan klaim.
--
-- Penanda /*CLAIMS*/ diganti daftar parameter sepanjang klaim yang diminta — bukan dirangkai
-- dari nilainya. Jumlahnya berubah tiap permintaan; yang tidak pernah berubah adalah bahwa
-- nilainya lewat parameter binding.
SELECT ID, LINGKUP, CASE_ID, NOMOR_KLAIM,
       OPERATOR_ASAL, OPERATOR_TUJUAN, TIPE_PENGGUNA, ALASAN,
       STATUS, PEMOHON, PEMOHON_NAMA, DIBUAT_PADA
  FROM POOLDATA.CPNC_PERMINTAAN_TRANSFER
 WHERE STATUS = 'menunggu'
   AND CASE_ID IN (/*CLAIMS*/)
 ORDER BY DIBUAT_PADA
