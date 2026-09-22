-- Kueri tabel CPNC_TUGAS.
--
-- Tugas tidak pernah dihapus: yang selesai ditandai SELESAI_PADA. Riwayat siapa
-- mengerjakan apa adalah bagian jejak audit klaim, bukan sampah yang dibersihkan.

-- name: tugas_perbarui
-- Pengambilan tugas dari Workbasket dijaga di sini, bukan hanya di dalam kode.
--
-- Dua orang yang menekan "ambil" pada tugas yang sama adalah kejadian biasa — sebuah
-- antrean bersama memang dilihat banyak orang sekaligus. Syarat
-- `(PEMILIK IS NULL OR PEMILIK = :pemilik)` membuat yang kedua tidak mengubah satu baris
-- pun, dan pemanggil menerjemahkan nol baris itu menjadi "sudah diambil orang lain".
-- Tanpa syarat itu, yang terakhir menulis akan menang tanpa ada yang tahu (TKT-B06-003).
UPDATE CPNC_TUGAS
   SET NOMOR_KLAIM    = :1,
       PEMILIK        = :2,
       DIAMBIL_PADA   = :3,
       SELESAI_PADA   = :4,
       ALASAN_SELESAI = :5
 WHERE ID = :6
   AND SELESAI_PADA IS NULL
   AND (PEMILIK IS NULL OR PEMILIK = :7)

-- name: tugas_sisip
INSERT INTO CPNC_TUGAS (NOMOR_KLAIM, PEMILIK, DIAMBIL_PADA, SELESAI_PADA, ALASAN_SELESAI,
                        ID, KLAIM_ID, TAHAP, ANTREAN, WORKBASKET, DIBUAT_PADA)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11)

-- name: tugas_ambil
SELECT ID, KLAIM_ID, NOMOR_KLAIM, TAHAP, ANTREAN, WORKBASKET, PEMILIK,
       DIBUAT_PADA, DIAMBIL_PADA, SELESAI_PADA, ALASAN_SELESAI
  FROM CPNC_TUGAS
 WHERE ID = :1

-- name: tugas_terbuka_klaim
SELECT ID, KLAIM_ID, NOMOR_KLAIM, TAHAP, ANTREAN, WORKBASKET, PEMILIK,
       DIBUAT_PADA, DIAMBIL_PADA, SELESAI_PADA, ALASAN_SELESAI
  FROM CPNC_TUGAS
 WHERE KLAIM_ID = :1
   AND SELESAI_PADA IS NULL
 ORDER BY DIBUAT_PADA, ID

-- name: tugas_inbox_milik_saya
-- Kelompok pertama isi Inbox: tugas yang sudah menjadi milik pemanggil, apa pun
-- antreannya. Tugas Workbasket yang sudah ia ambil termasuk di sini.
SELECT ID, KLAIM_ID, NOMOR_KLAIM, TAHAP, ANTREAN, WORKBASKET, PEMILIK,
       DIBUAT_PADA, DIAMBIL_PADA, SELESAI_PADA, ALASAN_SELESAI
  FROM CPNC_TUGAS
 WHERE PEMILIK = :1
   AND SELESAI_PADA IS NULL
 ORDER BY DIBUAT_PADA, ID

-- name: tugas_inbox_antrean
-- Kelompok kedua: tugas antrean bersama yang BELUM bertuan. Ia belum menjadi pekerjaan
-- siapa pun — ia tawaran.
SELECT ID, KLAIM_ID, NOMOR_KLAIM, TAHAP, ANTREAN, WORKBASKET, PEMILIK,
       DIBUAT_PADA, DIAMBIL_PADA, SELESAI_PADA, ALASAN_SELESAI
  FROM CPNC_TUGAS
 WHERE WORKBASKET = :1
   AND PEMILIK IS NULL
   AND SELESAI_PADA IS NULL
 ORDER BY DIBUAT_PADA, ID
