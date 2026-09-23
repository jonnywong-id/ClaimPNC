-- Penerjemah login petugas menjadi kode cabang klaimnya.
--
-- ============================================================================
-- KENAPA IA TIDAK DAPAT DIAMBIL DARI PROFIL HCC/HCQ
-- ============================================================================
--
-- Karena kode cabang yang dipakai layar Inbox Laporan Klaim adalah `POOLDATA.BRANCH.ID`,
-- dan profil HCC/HCQ mengirim `EmpResponse.Placement.BranchCode` — sistem kode yang
-- berbeda. Memakai yang kedua sebagai penyaring terhadap `kodecabang_1` membuat tidak
-- satu pun baris cocok, dan daftar tampil KOSONG tanpa satu pun pesan galat.
--
-- Sistem lama menurunkannya lewat tiga tabel dan dua DB Link:
--
--	login petugas -> HRDASM.V_HRD_MST.login_aplikasi
--	              -> NIK
--	              -> LST_USER_ASURANSI.cab_id
--	              -> BRANCH.oldid
--	              -> BRANCH.id
--
-- Perhatikan sambungan terakhir: ia lewat `oldid`, BUKAN `id`. Tidak ada jalan pintas.
--
-- ============================================================================
-- DUA DB LINK YANG MASIH DIPAKAI, DAN KENAPA
-- ============================================================================
--
-- `D-25` menetapkan seluruh DB Link diganti pemanggilan API, dan `R-03` mencatat API-nya
-- kemungkinan belum ada. Selama itu belum tiba, kueri ini memakai DB Link yang sama dengan
-- sistem lama — ia membaca objek yang sama, lewat sambungan yang sama, dari basis data
-- yang sama.
--
-- Yang membuatnya dapat diganti tanpa menyentuh aturan modul: ia berada di balik seam
-- `inboxlaporanklaim.BranchResolver`. Saat API penggantinya tiba, yang berubah hanya
-- pengisi seam itu.
--
-- Empat aturan berkas .sql tetap berlaku: kolom disebut namanya, nilai lewat parameter
-- binding, tanpa NVL/SYSDATE/DECODE/ROWNUM/TO_CHAR, dan tanpa pemanggilan procedure.

-- name: branch_of_login
--
-- Asal: `RDB List/GetIDCabang-SQL.xml`, disalin apa adanya kecuali dua hal.
--
--   select a.id AS "KodeCabang", a.branchname as "Remark", b.LUS_ID as "Keyword"
--     from branch a,
--          hrdasm.v_hrd_mst@asmd.sinarmas.co.id c,
--          lst_user_asuransi@asmd.sinarmas.co.id b
--    where c.login_aplikasi = {TempCabang.UserAdmin}
--      and c.nik = b.nik
--      and b.cab_id = a.oldid
--
-- Yang diubah:
--
--  1. Gabungan gaya lama (koma pada FROM) menjadi JOIN eksplisit. `09-DATABASE-STRATEGY.md`
--     §4 menuntutnya demi portabilitas, dan bentuk ini juga membuat arah sambungannya
--     terbaca — tiga tabel yang digabung lewat koma menyembunyikan mana yang menyaring apa.
--
--  2. Dua kolom yang tidak dipakai dibuang. Kueri lama ikut mengambil `branchname` dan
--     `LUS_ID` karena satu RDB List melayani beberapa pemanggil; yang dibutuhkan di sini
--     hanya kodenya.
--
-- Hasil lebih dari satu baris DIMUNGKINKAN bila seorang petugas terdaftar di lebih dari
-- satu cabang. Kueri lama mengambil `pxResults(1)` — baris pertama, tanpa urutan yang
-- ditetapkan. Di sini urutannya DITEGASKAN supaya dua pemanggilan yang sama menghasilkan
-- cabang yang sama; tanpa itu, batas data seorang petugas dapat berpindah-pindah di antara
-- dua permintaan tanpa sebab yang terlihat.
SELECT a.id
  FROM POOLDATA.BRANCH a
  JOIN LST_USER_ASURANSI@asmd.sinarmas.co.id b ON b.cab_id = a.oldid
  JOIN HRDASM.V_HRD_MST@asmd.sinarmas.co.id c ON c.nik = b.nik
 WHERE c.login_aplikasi = :1
 ORDER BY a.id
 FETCH NEXT 1 ROWS ONLY
