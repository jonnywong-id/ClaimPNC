-- 0013 down — membatalkan kolom isian Persetujuan / Akseptasi pada T_CLAIM_ADJUSTMENT
--
-- MENJALANKAN BERKAS INI MENGHAPUS DATA, BUKAN HANYA KOLOM. Hanya sah bila kolomnya belum
-- pernah diisi; bila sudah, hentikan pemakaiannya lebih dulu dan hapus pada rilis berikutnya
-- (P-4, penghapusan dua tahap; D-66).

ALTER TABLE POOLDATA.T_CLAIM_ADJUSTMENT DROP (
    TANGGALBOLEHBAYAR, RECEIVEDATEANALIST, ACCEPTANCEVALUELOD, TIPEAKSEPTASI,
    KOMITEACCEPTED, REMARKACCEPTED, UPLOADNOTELOD
);
