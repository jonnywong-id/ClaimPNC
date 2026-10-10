-- Mengembalikan 0015.
--
-- Index dibuang LEBIH DULU: membuang kolomnya lebih dahulu membuang index itu diam-diam,
-- dan urutan yang eksplisit membuat kegagalan separuh jalan terbaca dari pesannya.
--
-- PERINGATAN. Menjalankan ini MENGHAPUS data kasus survei yang sudah masuk — ketiga kolom
-- itu tidak ada duanya di tabel ini. Sumbernya masih ada di DATAPEGA selama masa paralel,
-- jadi pengisiannya dapat diulang; setelah Pega dimatikan, tidak lagi.
--
-- Baris kasus survei itu sendiri (PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim') TIDAK
-- dihapus di sini. Membuangnya adalah keputusan data, bukan pembatalan perubahan skema,
-- dan menggabungkan keduanya membuat rollback skema ikut menghapus baris bisnis.

DROP INDEX POOLDATA.IX_CLAIMLIST_ADMIN_CASEID;

ALTER TABLE POOLDATA.T_CLAIMLIST_ADMIN DROP (
  CASEID_1,
  RESCHEDULEDATE_1,
  RESCHEDULELOCATION_1
);
