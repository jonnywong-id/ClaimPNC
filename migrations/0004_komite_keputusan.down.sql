-- 0004 — pembatalan: menghapus tabel jejak keputusan komite.
--
-- ============================================================================
-- BACA INI SEBELUM MENJALANKANNYA. IA MENGHAPUS JEJAK YANG TIDAK DAPAT DIPULIHKAN.
-- ============================================================================
--
-- # Apa yang hilang, dan kenapa itu lebih berat daripada migrasi lain
--
-- Isi tabel ini adalah **satu-satunya catatan** bahwa seseorang menyetujui, menolak, atau
-- mengembalikan sebuah klaim di sistem baru. Ia tidak ada duanya di mana pun: keputusan
-- ini TIDAK ditulis ke Pega, karena `P-1` melarangnya.
--
-- Dan `D-59` menghapus pemisahan tugas — satu orang dapat membuat, menyetujui, dan
-- membayarkan satu klaim — sehingga jejak inilah satu-satunya kontrol pengimbang yang
-- tersisa. Menghapusnya berarti menghapus bukti kewenangan atas uang klaim.
--
-- # Yang WAJIB dikerjakan lebih dulu
--
-- Ambil salinannya, dan simpan di luar basis data ini:
--
--     SELECT ID, CASE_ID, NOMOR_KLAIM, JENJANG, KEPUTUSAN, CATATAN,
--            ACTOR_LOGIN, ACTOR_NAMA, PADA
--       FROM POOLDATA.CPNC_KOMITE_KEPUTUSAN
--      ORDER BY PADA;
--
-- Bila tabelnya masih kosong — modul belum pernah dipakai — langkah itu tidak diperlukan,
-- dan pembatalan ini aman sepenuhnya.
--
-- # Yang TIDAK tersentuh
--
-- Tidak ada satu pun objek milik sistem lama. Pembatalan ini hanya membuang apa yang
-- migrasi 0004 buat; Pega tidak pernah tahu tabel ini ada.

DROP INDEX POOLDATA.IX_CPNC_KOMITE_KEPUTUSAN_ACTOR;
DROP INDEX POOLDATA.IX_CPNC_KOMITE_KEPUTUSAN_CASE;

DROP TABLE POOLDATA.CPNC_KOMITE_KEPUTUSAN;
