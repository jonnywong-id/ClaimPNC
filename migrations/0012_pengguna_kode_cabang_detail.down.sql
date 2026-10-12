-- 0012 — pembatalan: melepas kolom KODE_CABANG_DETAIL dari CPNC_PENGGUNA.
--
-- ============================================================================
-- BACA SEBELUM MENJALANKAN
-- ============================================================================
--
-- Melepas kolom MENGHAPUS ISINYA, dan isinya tidak dapat dipulihkan dari tempat lain di
-- basis data ini — ia datang dari HCQ pada setiap kali pengguna masuk. Setelah dilepas,
-- nilainya kosong sampai setiap pengguna masuk kembali.
--
-- `P-4` menuntut perubahan skema backward-compatible: versi lama dan baru aplikasi berjalan
-- bersamaan saat rolling deployment. Menjalankan berkas ini selagi versi yang MEMBACA kolom
-- itu masih hidup akan membuat setiap pembacaan pengguna gagal — bukan hanya layar OS per
-- cabang, melainkan SELURUH jalur masuk.
--
-- Urutan yang benar: kembalikan binary ke versi yang tidak membaca kolom ini LEBIH DULU,
-- baru jalankan berkas ini.
--
-- Aman diulang: kolom yang sudah tidak ada menghasilkan ORA-00904, yang ditangkap di bawah.

DECLARE
    tidak_ada EXCEPTION;
    PRAGMA EXCEPTION_INIT(tidak_ada, -904);
BEGIN
    EXECUTE IMMEDIATE 'ALTER TABLE CPNC_PENGGUNA DROP COLUMN KODE_CABANG_DETAIL';
EXCEPTION
    WHEN tidak_ada THEN NULL;
END;
/
