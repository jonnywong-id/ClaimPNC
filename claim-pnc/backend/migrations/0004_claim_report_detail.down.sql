-- 0004 — Pembatalan kolom isian form Input Receive Document (Oracle 19c)
--
-- Membuang kesepuluh kolom yang ditambahkan 0004, beserta index nomor polis. Tabelnya
-- sendiri TIDAK dibuang — ia milik 0003, dan membatalkannya adalah urusan berkas itu.
--
-- YANG HILANG BILA INI DIJALANKAN: seluruh isian form pada setiap berkas laporan — nama
-- pengirim, kronologis, rincian kerusakan, dan sisanya. Yang tetap ada hanyalah kepala
-- berkasnya: nomor, cabang, tanggal aging, dan jejak pembuatannya.
--
-- Karena itu berkas ini hanya benar dijalankan ketika 0004 baru saja dipasang dan belum
-- ada satu berkas pun yang diisi petugas. Setelah pemakaian nyata dimulai, membatalkannya
-- berarti membuang data bisnis — dan itu menuntut keputusan tersendiri, bukan sekadar
-- menjalankan berkas down.
--
-- Urutannya: index lebih dulu, lalu kolomnya. Membuang kolom yang masih diindeks
-- memaksa Oracle membuang index-nya diam-diam, dan pembatalan yang menyentuh lebih
-- banyak daripada yang disebutnya bukan pembatalan yang dapat dipercaya.

DROP INDEX IX_CPNC_LAPORAN_POLIS;

ALTER TABLE CPNC_LAPORAN_KLAIM DROP (
    TGL_TERIMA_DOKUMEN,
    EMAIL_PELAPOR,
    TLP_PELAPOR,
    NAMA_KURIR,
    NILAI_ESTIMASI,
    LOKASI_KEJADIAN,
    KRONOLOGIS,
    RINCIAN_KERUSAKAN,
    KET_BELUM_REGISTRASI,
    JUMLAH_DOKUMEN
);
