-- 0005 turun — kembalikan Master Penyebab Kerugian ke pembacaan JSON
--
-- ============================================================================
-- BACA DULU: apa yang dapat dikembalikan, dan apa yang tidak.
-- ============================================================================
--
-- Migrasi naik melakukan dua hal yang mengubah keadaan. Keduanya dapat dibatalkan —
-- tetapi yang dipulihkan adalah BENTUKNYA, bukan seluruh ISINYA.
--
--   Langkah naik                          Dapat dibatalkan?
--   -----------------------------------   ------------------------------------------
--   1. Mengisi kolom COL_DESC             Ya — dikosongkan kembali (langkah 3 di bawah)
--   3. Mendefinisikan ulang view          Ya — definisinya dipulihkan (langkah 2)
--
-- Yang TIDAK dapat dikembalikan adalah perubahan master yang dibuat pengguna lewat
-- aplikasi Go setelah migrasi naik. Perubahan itu hanya hidup di kolom COL_DESC;
-- JSON_DATA tidak pernah ikut diperbarui — itu memang maksud migrasi naik. Mengembalikan
-- view ke JSON_DATA berarti mengembalikan pula isi master seperti saat migrasi naik
-- dijalankan.
--
-- Lebih jauh dari itu: baris yang DITAMBAHKAN lewat aplikasi Go tidak punya JSON_DATA
-- sama sekali. Setelah view dikembalikan, baris-baris itu akan tampil TANPA KETERANGAN
-- di seluruh 19 rule pembaca — tidak hilang, tetapi kosong. Langkah 0 menyelamatkan
-- isinya lebih dulu supaya setidaknya tidak lenyap tanpa jejak.
--
--
-- ## Satu berkas yang harus sudah ada sebelum migrasi turun dijalankan
--
-- Definisi view yang asli TIDAK ADA di repository ini: ia objek basis data, bukan rule
-- Pega. Migrasi naik mewajibkan DBA menyimpannya lebih dulu pada LANGKAH 0a.
--
-- Berbeda dari migrasi 0002 — yang bentuk view aslinya sudah pernah dibaca dari katalog
-- pada 2026-09-17 sehingga dapat dituliskan sebagai cadangan — bentuk asli view ini
-- **belum pernah dibaca siapa pun di tim ini**. Yang ditulis pada langkah 2 di bawah
-- adalah DUGAAN berdasarkan pola V_STS_CLAIM.
--
-- **Bila berkas simpanan DBA tidak ada, JANGAN jalankan langkah 2.** Memulihkan view
-- dengan bentuk yang ditebak berisiko mengubah urutan kolom atau kunci JSON-nya, dan
-- akibatnya justru lebih buruk daripada membiarkan view yang sekarang.
--
--
-- ## Satu hal yang TIDAK perlu dibatalkan
--
-- Migrasi naik tidak membuat indeks unik apa pun — berbeda dari 0002 dan 0003 — karena
-- keterangan ganda memang diterima. Jadi tidak ada indeks yang perlu dibuang di sini.
--
-- Kolom COL_DESC juga TIDAK dibuang, meski migrasi naik mungkin yang menambahkannya.
-- Membuang kolom adalah operasi yang tidak dapat dibatalkan dan menghapus satu-satunya
-- salinan perubahan yang dibuat lewat aplikasi Go. Ia dibiarkan kosong; kolom kosong
-- tidak mengganggu siapa pun.


-- ---------------------------------------------------------------------------
-- Langkah 0 — SELAMATKAN dulu apa yang akan hilang artinya.
--
-- Jalankan dan SIMPAN HASILNYA sebelum apa pun disentuh. Kueri pertama adalah seluruh
-- isi master menurut kolom; kueri kedua adalah baris yang akan menjadi kosong setelah
-- view dikembalikan — yakni yang tidak punya JSON_DATA.
-- ---------------------------------------------------------------------------

--     SELECT M_COL_ID, OLD_M_COL_ID, COL_DESC
--       FROM POOLDATA.M_CAUSE_OF_LOSS ORDER BY M_COL_ID;
--
--     SELECT M_COL_ID, COL_DESC
--       FROM POOLDATA.M_CAUSE_OF_LOSS
--      WHERE JSON_DATA IS NULL
--      ORDER BY M_COL_ID;


-- ---------------------------------------------------------------------------
-- Langkah 1 — hentikan penulisan dari aplikasi Go.
--
-- Dilakukan SEBELUM view dikembalikan, bukan sesudah. Bila urutannya dibalik, ada
-- jendela waktu ketika aplikasi masih menulis ke kolom sementara view sudah membaca
-- JSON — dan perubahan yang dibuat pengguna di jendela itu hilang tanpa jejak, termasuk
-- dari langkah 0 yang sudah terlanjur dijalankan.
--
-- Cara yang paling sederhana adalah mencabut hak tulisnya. Ganti <AKUN_APLIKASI>.
-- ---------------------------------------------------------------------------

-- REVOKE INSERT, UPDATE ON POOLDATA.M_CAUSE_OF_LOSS FROM <AKUN_APLIKASI>;


-- ---------------------------------------------------------------------------
-- Langkah 2 — kembalikan view supaya membaca JSON.
--
-- ⚠ PAKAI BERKAS SIMPANAN DBA DARI LANGKAH 0a MIGRASI NAIK. Yang di bawah adalah
--   DUGAAN berdasarkan pola V_STS_CLAIM, dan belum pernah diverifikasi ke katalog.
--   Urutan kolom dan kunci JSON-nya khususnya belum pasti.
--
-- CREATE OR REPLACE, bukan DROP lalu CREATE — yang pertama mempertahankan seluruh grant.
-- ---------------------------------------------------------------------------

-- CREATE OR REPLACE VIEW POOLDATA.V_M_CAUSE_OF_LOSS (M_COL_ID, OLD_M_COL_ID, COL_DESC) AS
-- SELECT M_COL_ID,
--        OLD_M_COL_ID,
--        JSON_VALUE(JSON_DATA, '$.COL_DESC')
--   FROM POOLDATA.M_CAUSE_OF_LOSS;


-- ---------------------------------------------------------------------------
-- Langkah 3 — kosongkan kembali kolom keterangan.
--
-- Dijalankan PALING AKHIR, dan hanya setelah langkah 2 terbukti berhasil. Selama kolom
-- masih terisi, ia salinan cadangan yang masih dapat dibaca bila langkah 2 gagal.
--
-- Dikosongkan, bukan dibiarkan: kolom yang terisi tetapi tidak dibaca siapa pun akan
-- menjadi USANG dan menyesatkan siapa pun yang membacanya kemudian — persis kekeliruan
-- yang ditinggalkan JSON_DATA sesudah migrasi naik.
--
-- Barisnya dibatasi pada yang punya JSON_DATA, supaya baris tulisan aplikasi Go — yang
-- keterangannya hanya ada di kolom ini — tidak ikut dihapus isinya. Ia akan tampil
-- kosong di view, tetapi setidaknya nilainya masih dapat ditemukan di tabel.
-- ---------------------------------------------------------------------------

-- UPDATE POOLDATA.M_CAUSE_OF_LOSS
--    SET COL_DESC = NULL
--  WHERE JSON_DATA IS NOT NULL;
--
-- COMMIT;
