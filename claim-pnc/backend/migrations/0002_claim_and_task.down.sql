-- Pembatalan 0002.
--
-- Urutannya kebalikan dari pembuatan: tabel anak lebih dulu, karena foreign key-nya
-- menahan penghapusan induk.
--
-- PERINGATAN. Menjalankan berkas ini MENGHAPUS SELURUH KLAIM yang terbit di sistem baru,
-- beserta jejak auditnya. Klaim yang telanjur terbit TIDAK DAPAT dipindahkan ke Pega
-- (P-3), sehingga rollback modul ini yang benar adalah MENGHENTIKAN PENDAFTARAN KLAIM
-- BARU — bukan menjalankan berkas ini. Ia disediakan untuk lingkungan uji, bukan untuk
-- produksi.

DROP TABLE CPNC_NOTIFIKASI;
DROP TABLE CPNC_JEJAK_AUDIT;
DROP TABLE CPNC_NOMOR_KLAIM;
DROP TABLE CPNC_TUGAS;
DROP TABLE CPNC_KLAIM_SPREADING;
DROP TABLE CPNC_KLAIM_COVERAGE;
DROP TABLE CPNC_KLAIM_OBJEK;
DROP TABLE CPNC_KLAIM;
