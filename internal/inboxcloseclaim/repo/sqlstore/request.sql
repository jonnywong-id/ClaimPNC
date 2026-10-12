-- Permintaan ReOpen dan Copy Klaim — POOLDATA.CPNC_PERMINTAAN_KLAIM.
--
-- ============================================================================
-- TABEL INI MILIK APLIKASI INI SEPENUHNYA
-- ============================================================================
--
-- Dibuat migrasi `0006`. Tidak dibaca dan tidak ditulis Pega, sehingga `P-1` terpenuhi
-- tanpa negosiasi kepemilikan.
--
-- Itulah sebabnya ia ada. `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` — tabel yang benar-benar harus
-- berubah saat sebuah klaim dibuka kembali — hari ini ditulis Pega, dan `P-1`/`ADR-0004`
-- menetapkan satu tabel hanya ditulis satu sistem. Work Owner memutuskan 2026-09-23
-- aplikasi ini mencatat PERMINTAANNYA; eksekusinya tetap di Pega.
--
-- ============================================================================
-- APLIKASI INI HANYA INSERT DAN SELECT
-- ============================================================================
--
-- Tidak ada satu pun UPDATE maupun DELETE di berkas ini, dan ketiadaannya bukan kelalaian:
-- akun aplikasi TIDAK diberi hak untuk keduanya (lihat kepala `migrations/0006`). Yang
-- memindahkan STATUS dari 'menunggu' ke 'dijalankan' adalah pihak yang benar-benar
-- mengeksekusi, dengan hak aksesnya sendiri.
--
-- Aturan yang hanya ada di dalam kode dapat dilanggar oleh kode berikutnya; aturan yang ada
-- di hak akses tidak.

-- name: request_insert
-- Mencatat satu permintaan.
--
-- Bind: :1 ID · :2 JENIS · :3 CASE_ID · :4 NOMOR_KLAIM · :5 ALASAN · :6 STATUS
--       :7 EFEK_STATUS_KERJA · :8 EFEK_STATUS_KLAIM · :9 LINGKUP_SALIN
--       :10 ACTOR_LOGIN · :11 ACTOR_NAMA · :12 PADA
INSERT INTO POOLDATA.CPNC_PERMINTAAN_KLAIM (
    ID, JENIS, CASE_ID, NOMOR_KLAIM, ALASAN, STATUS,
    EFEK_STATUS_KERJA, EFEK_STATUS_KLAIM, LINGKUP_SALIN,
    ACTOR_LOGIN, ACTOR_NAMA, PADA
) VALUES (
    :1, :2, :3, :4, :5, :6,
    :7, :8, :9,
    :10, :11, :12
)

-- name: request_pending_for
-- Permintaan yang MASIH MENUNGGU untuk sekumpulan klaim sekaligus.
--
-- Banyak klaim dalam satu kueri, bukan satu kueri per baris layar: kueri di dalam
-- perulangan adalah hambatan peringkat ketiga pada `15-NFR` §3.2.
--
-- Penanda /*CLAIMS*/ diganti di Go dengan daftar PENANDA parameter — `:1, :2, …` — bukan
-- nilainya. Seluruh nilai tetap dikirim terpisah, sehingga celah `{ASIS:…}` warisan tetap
-- tertutup.
--
-- Hanya yang 'menunggu' yang dibaca. Permintaan yang sudah dijalankan atau dibatalkan tidak
-- menghalangi permintaan baru, dan tidak perlu ditampilkan sebagai peringatan di layar.
SELECT ID,
       JENIS,
       CASE_ID,
       NOMOR_KLAIM,
       ALASAN,
       STATUS,
       EFEK_STATUS_KERJA,
       EFEK_STATUS_KLAIM,
       LINGKUP_SALIN,
       ACTOR_LOGIN,
       ACTOR_NAMA,
       PADA
  FROM POOLDATA.CPNC_PERMINTAAN_KLAIM
 WHERE STATUS = 'menunggu'
   AND CASE_ID IN (/*CLAIMS*/)
 ORDER BY PADA DESC

-- name: request_check_table
-- Memastikan tabel permintaan dapat dibaca akun aplikasi.
--
-- Dipakai perintah `-periksa` saat start. Ia sengaja tidak mengembalikan baris: yang
-- diperiksa adalah keberadaan tabel beserta hak SELECT-nya, bukan isinya.
SELECT ID
  FROM POOLDATA.CPNC_PERMINTAAN_KLAIM
 WHERE 1 = 0
