-- 0002 turun — kembalikan Master Status Klaim ke pembacaan JSON
--
-- ============================================================================
-- BACA DULU: apa yang dapat dikembalikan, dan apa yang tidak.
-- ============================================================================
--
-- Migrasi naik melakukan tiga hal. Dua dapat dibatalkan seluruhnya; satu tidak.
--
--   Langkah naik                          Dapat dibatalkan?
--   -----------------------------------   ------------------------------------------
--   1. Mengisi kolom LSC_NOTE             Ya — dikosongkan kembali (langkah 3 di bawah)
--   3. Mendefinisikan ulang view          Ya — definisinya dipulihkan (langkah 2)
--   4. Membuat indeks unik UX_...LABEL    Ya — dibuang (langkah 1)
--
-- Yang TIDAK dapat dikembalikan adalah perubahan master yang dibuat pengguna lewat
-- aplikasi Go setelah migrasi naik. Perubahan itu hanya hidup di kolom LSC_NOTE;
-- JSONDATA tidak pernah ikut diperbarui — itu memang maksud migrasi naik. Mengembalikan
-- view ke JSONDATA berarti mengembalikan pula isi master seperti saat migrasi naik
-- dijalankan.
--
-- Langkah 0 menyelamatkannya lebih dulu supaya setidaknya tidak lenyap tanpa jejak.
--
--
-- ## Satu berkas yang harus sudah ada sebelum migrasi turun dijalankan
--
-- Definisi view yang asli TIDAK ADA di repository ini: ia objek basis data, bukan rule
-- Pega. Migrasi naik mewajibkan DBA menyimpannya lebih dulu.
--
-- Bentuknya sudah diketahui dari pembacaan katalog 2026-09-17, dan dituliskan di
-- langkah 2 sebagai cadangan. TETAPI yang dipakai tetap berkas simpanan DBA — bentuk di
-- bawah berasal dari ALL_VIEWS.TEXT, yang TIDAK menyimpan daftar nama kolom view.
-- Memakainya apa adanya berisiko mengubah nama kolom ketiga.


-- ---------------------------------------------------------------------------
-- Langkah 0 — selamatkan perubahan yang dibuat aplikasi Go.
--
-- Jalankan dan SIMPAN HASILNYA sebelum melanjutkan. Nol baris berarti tidak ada
-- perubahan yang akan hilang.
-- ---------------------------------------------------------------------------

-- SELECT LSC_ID,
--        LSC_NOTE                          AS LABEL_SEKARANG,
--        JSON_VALUE(JSONDATA,'$.LSC_NOTE') AS LABEL_DI_JSON
--   FROM POOLDATA.M_STS_CLAIM
--  WHERE NVL(TRIM(LSC_NOTE), '~') <> NVL(TRIM(JSON_VALUE(JSONDATA,'$.LSC_NOTE')), '~');


-- ---------------------------------------------------------------------------
-- Langkah 1 — buang indeks unik.
--
-- Dijalankan lebih dulu: langkah 3 mengosongkan kolom yang diindeks ini, dan indeks unik
-- atas kolom yang seluruhnya NULL memang tidak bermasalah di Oracle — tetapi
-- membuangnya lebih awal membuat urutannya tidak bergantung pada perilaku itu.
-- ---------------------------------------------------------------------------

DROP INDEX POOLDATA.UX_M_STS_CLAIM_LABEL;


-- ---------------------------------------------------------------------------
-- Langkah 2 — kembalikan definisi view yang asli.
--
-- UTAMAKAN berkas DDL yang disimpan DBA sebelum migrasi naik:
--
--     @definisi_v_sts_claim_sebelum_0002.sql
--
-- Bila berkas itu tidak ada, bentuk di bawah adalah rekonstruksi dari pembacaan katalog
-- 2026-09-17. Daftar nama kolom ditulis eksplisit karena ALL_VIEWS.TEXT tidak
-- menyimpannya, dan tanpa itu kolom ketiga akan bernama
-- "JSON_VALUE(JSONDATA,'$.LSC_NOTE')" — yang membuat 23 rule Pega kehilangan LSC_NOTE.
-- ---------------------------------------------------------------------------

CREATE OR REPLACE VIEW POOLDATA.V_STS_CLAIM (LSC_ID, OLD_LSC_ID, LSC_NOTE) AS
SELECT LSC_ID,
       OLD_LSC_ID,
       JSON_VALUE(JSONDATA, '$.LSC_NOTE')
  FROM POOLDATA.M_STS_CLAIM;


-- ---------------------------------------------------------------------------
-- Langkah 3 — kosongkan kembali kolom LSC_NOTE.
--
-- Kolomnya TIDAK dibuang: ia sudah ada SEBELUM migrasi naik dijalankan, dalam keadaan
-- kosong pada seluruh 32 baris. Membuangnya berarti meninggalkan basis data dalam
-- keadaan yang BERBEDA dari sebelum migrasi — bukan memulihkannya.
--
-- Langkah ini opsional. Membiarkan kolomnya terisi tidak merusak apa pun setelah
-- langkah 2 berjalan, karena view tidak lagi membacanya. Kosongkan hanya bila keadaan
-- semula ingin dipulihkan seutuhnya.
-- ---------------------------------------------------------------------------

-- UPDATE POOLDATA.M_STS_CLAIM SET LSC_NOTE = NULL;
-- COMMIT;


-- ---------------------------------------------------------------------------
-- Langkah 4 — hak akses.
--
-- Cabut hak tulis yang diberikan migrasi naik; hak baca dibiarkan bila aplikasi masih
-- perlu menampilkan daftarnya.
-- ---------------------------------------------------------------------------

-- REVOKE INSERT, UPDATE ON POOLDATA.M_STS_CLAIM FROM <AKUN_APLIKASI>;


-- ---------------------------------------------------------------------------
-- Langkah 5 — procedure PEGA_M_STS_CLAIM.
--
-- Ia TIDAK PERNAH DIHAPUS migrasi naik, jadi tidak ada yang perlu dikembalikan di sini.
-- Membiarkannya berdiri adalah yang membuat rollback ini mungkin sama sekali: begitu
-- view kembali membaca JSONDATA, layar Pega lama dapat langsung dipakai lagi tanpa
-- perubahan apa pun di sisi Pega.
-- ---------------------------------------------------------------------------
