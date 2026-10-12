-- 0014 down — membatalkan penanda hapus baris adjustment pada T_CLAIM_ADJUSTMENT
--
-- MENJALANKAN BERKAS INI MENGHAPUS DATA, BUKAN HANYA KOLOM: baris yang sudah ditandai terhapus akan
-- tampil kembali. Hanya sah bila kolomnya belum pernah diisi; bila sudah, hentikan pemakaiannya lebih
-- dulu dan hapus pada rilis berikutnya (P-4, penghapusan dua tahap; D-66).

ALTER TABLE POOLDATA.T_CLAIM_ADJUSTMENT DROP (DIHAPUS_PADA);
