-- Kueri keputusan komite: POOLDATA.CPNC_KOMITE_KEPUTUSAN.
--
-- # Tabel ini MILIK APLIKASI INI sepenuhnya
--
-- Ia dibuat migrasi `0004`, tidak dibaca dan tidak ditulis Pega. `P-1` karena itu
-- terpenuhi tanpa negosiasi kepemilikan: tidak ada sistem lain yang menyentuhnya.
--
-- Inilah sebabnya berkas ini boleh memuat INSERT, sementara inbox.sql sama sekali tidak.
-- Kasus komite dibaca dari tabel warisan yang masih ditulis Pega; keputusannya ditulis ke
-- tabel sendiri. `TKT-B07-002` sudah menetapkan jalan itu pada bagian migrasi skema:
-- "Menambah tabel jejak komite. Backward-compatible."
--
--
-- # APPEND-ONLY, dan penegakannya BUKAN di sini
--
-- Tidak ada UPDATE dan tidak ada DELETE di berkas ini, dan ketiadaannya disengaja:
-- keputusan komite tidak dapat dihapus (`ADR-0012`). Perubahan pikiran dinyatakan dengan
-- keputusan baru, bukan dengan menyunting yang lama.
--
-- Tetapi aturan yang hanya ada di dalam kode dapat dilanggar oleh kode berikutnya.
-- Penegakan yang sesungguhnya ada di HAK AKSES BASIS DATA: akun aplikasi hanya diberi
-- INSERT dan SELECT pada tabel ini — tanpa UPDATE maupun DELETE
-- (`09-DATABASE-STRATEGY.md` §8). Langkahnya ada di kepala migrasi 0004.
--
--
-- # Kenapa ini penting lebih dari biasanya
--
-- `D-59` menghapus pemisahan tugas: satu orang dapat membuat, menyetujui, dan membayarkan
-- satu klaim bila perannya memiliki ketiga menu itu. Tidak ada kontrol teknis yang
-- mencegahnya, sehingga jejak inilah SATU-SATUNYA kontrol pengimbang yang tersisa.
--
-- Jejak yang dapat diubah bukan jejak.


-- name: decision_list_for_cases
--
-- Keputusan seluruh kasus yang disebut, dalam SATU perjalanan.
--
-- Penanda /*CASES*/ diganti daftar `:2, :3, …` oleh expandCases di decision.go. Yang
-- disisipkan hanyalah PENANDA PARAMETER, tidak pernah nilainya — celah `{ASIS:...}`
-- warisan tetap tertutup.
--
-- Urutannya ditetapkan di sini supaya hasil yang sama selalu datang dalam susunan yang
-- sama. Aturan urutan yang MENGIKAT tetap di Go (`komite.Evaluate`), karena kesimpulan
-- penjenjangan tidak boleh bergantung pada apa yang kebetulan dikembalikan basis data.
SELECT ID,
       CASE_ID,
       NOMOR_KLAIM,
       JENJANG,
       KEPUTUSAN,
       CATATAN,
       ACTOR_LOGIN,
       ACTOR_NAMA,
       PADA
  FROM POOLDATA.CPNC_KOMITE_KEPUTUSAN
 WHERE CASE_ID IN (/*CASES*/)
 ORDER BY CASE_ID, JENJANG, PADA, ID


-- name: decision_insert
--
-- Mencatat satu keputusan. Ia tidak pernah menimpa apa pun.
--
-- Bentrok pada CPNC_KOMITE_KEPUTUSAN_UK — satu orang memutuskan sekali per kasus —
-- diterjemahkan decision.go menjadi komite.ErrDecisionClosed, sehingga dua tab yang
-- ditekan bersamaan menghasilkan pesan yang dapat dibaca, bukan galat 500.
INSERT INTO POOLDATA.CPNC_KOMITE_KEPUTUSAN
       (ID, CASE_ID, NOMOR_KLAIM, JENJANG, KEPUTUSAN, CATATAN, ACTOR_LOGIN, ACTOR_NAMA, PADA)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9)


-- name: decision_check_table
--
-- Memastikan tabel dan seluruh kolomnya dapat dibaca akun aplikasi, tanpa mengambil satu
-- baris pun.
SELECT ID,
       CASE_ID,
       NOMOR_KLAIM,
       JENJANG,
       KEPUTUSAN,
       CATATAN,
       ACTOR_LOGIN,
       ACTOR_NAMA,
       PADA
  FROM POOLDATA.CPNC_KOMITE_KEPUTUSAN
 WHERE 1 = 0
