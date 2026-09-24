-- 0003 turun — cabut penegakan keunikan nama tipe surveyor
--
-- ============================================================================
-- BACA DULU: apa yang dikembalikan, dan apa yang TIDAK.
-- ============================================================================
--
-- Migrasi naik hanya melakukan SATU hal yang mengubah skema: membuat indeks unik
-- UX_M_SURVEYORS_DESC. Berkas ini membuangnya kembali, dan itu pembatalan yang utuh —
-- tidak ada data yang dipindahkan pada migrasi naik, sehingga tidak ada data yang perlu
-- dikembalikan.
--
-- Berbeda dari migrasi 0002 turun, yang harus mengembalikan isi master dan definisi view.
--
--
-- ## Yang TIDAK dikembalikan berkas ini, dan memang tidak bisa
--
-- Tipe surveyor yang DITAMBAHKAN atau DIUBAH lewat aplikasi Go tetap ada sebagaimana
-- adanya di kolom DESCRIPTION. Itu bukan kelalaian: kolom itu memang kolom yang dibaca
-- V_M_SURVEYORS sejak sebelum migrasi ini, dan mengosongkannya akan membuat tipe surveyor
-- hilang dari setiap layar Pega yang membacanya.
--
-- Satu akibat yang perlu disadari bila aplikasi Go kemudian dimatikan dan pengelolaan
-- dikembalikan ke layar Pega: baris yang ditulis aplikasi Go memuat DESCRIPTION tetapi
-- JSON_DATA bernilai NULL. Layar Pega membaca lewat view — jadi ia TETAP menampilkannya
-- dengan benar. Yang tidak akan berjalan benar adalah penyimpanan dari layar Pega, dan itu
-- sudah menjadi keadaannya sejak sebelum modul ini ada (lihat temuan di berkas naik).
--
--
-- ## Kapan berkas ini dijalankan
--
-- Hanya bila indeks uniknya sendiri yang bermasalah — misalnya ditemukan bahwa nama tipe
-- yang sama memang SAH dipakai dua kali pada entitas tertentu. Mematikan modul Master Tipe
-- Surveyors TIDAK menuntut migrasi turun: indeksnya tidak mengganggu siapa pun, dan
-- procedure lama pun tidak terhalang olehnya selama nama yang ditulisnya tidak ganda.


-- ---------------------------------------------------------------------------
-- Langkah 1 — buang indeks uniknya.
--
-- Sesudah ini, nama tipe surveyor ganda hanya ditolak pemeriksaan di aplikasi — dan dua
-- permintaan yang tiba bersamaan dapat sama-sama lolos.
-- ---------------------------------------------------------------------------

DROP INDEX POOLDATA.UX_M_SURVEYORS_DESC;


-- ---------------------------------------------------------------------------
-- Langkah 2 — hak akses.
--
-- GRANT yang diberikan langkah 2 migrasi naik sengaja TIDAK dicabut di sini.
--
-- Pemeriksaan 2026-09-19 menunjukkan akun aplikasi sudah memiliki hak itu di portal ASM
-- SEBELUM migrasi ini ada, sehingga mencabutnya bukan mengembalikan keadaan semula
-- melainkan mengubahnya — dan hak yang sama mungkin dipakai proses lain.
--
-- Bila hak itu memang perlu dicabut, ia permintaan tersendiri dengan pemeriksaannya
-- sendiri, bukan efek samping migrasi turun.
-- ---------------------------------------------------------------------------
