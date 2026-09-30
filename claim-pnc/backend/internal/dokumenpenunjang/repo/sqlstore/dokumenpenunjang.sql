-- Kueri metadata dokumen penunjang.
--
-- ============================================================================
-- SELURUH TABELNYA BERADA DI BASIS DATA LAIN
-- ============================================================================
--
-- Diperiksa langsung ke katalog 2026-09-26: skema `GENERAL` **tidak ada** di basis data
-- Claim PNC — `ALL_USERS` nol baris untuk nama itu. Work Owner menegaskan tempatnya:
--
--     "general.t_storage_image@asmd.sinarmas.co.id (BUKAN POOLDATA)"
--
-- Terverifikasi lewat DB link tersebut:
--
--     GENERAL.T_FOLDER_STORAGE      10 baris
--     GENERAL.T_STORAGE_IMAGE  128.379 baris · 2.050 di antaranya APPNAME 'klaimpnc'
--     GENERAL.GCP_IMAGE             33 baris
--
-- Hak tulisnya ADA: `UPDATE ... WHERE 1 = 0` lewat link itu diterima, bukan ditolak.
--
-- ============================================================================
-- INI MELANGGAR ARAH `D-25`, DAN ITU DISADARI
-- ============================================================================
--
-- `D-25` mengganti seluruh DB link dengan pemanggilan API, dan `R-03` mencatat API-nya
-- belum ada. Menunggu API berarti layar unggah tidak dapat dibangun sama sekali.
--
-- Jalan tengahnya bukan kompromi diam-diam melainkan **seam**: kueri di berkas ini adalah
-- SATU adapter di balik `dokumenpenunjang.Repo`. Ketika API penggantinya tiba, yang berubah
-- hanyalah adapter — tidak satu baris pun aturan domain.
--
-- ============================================================================
-- JANGAN PERNAH MENGAGREGASI TABEL INI LEWAT DB LINK
-- ============================================================================
--
-- `SELECT ... GROUP BY` atas 128.379 baris lewat link ini **tidak selesai dalam 5 menit**
-- saat diukur. Seluruh kueri di sini karena itu menyaring `NO_CLAIM` atau `IMAGEID` lebih
-- dulu — keduanya selektif — dan tidak satu pun mengagregasi.


-- name: folder_aplikasi
-- Mencari nama folder penyimpanan sebuah aplikasi.
--
-- Salinan `RDB List/GetAppFolder-SQL.xml`:
--
--     select NAMA_FOLDER AS CARI1 FROM general.T_FOLDER_STORAGE WHERE APLIKASI = {TempApp.CARI1}
--
-- Hasilnya yang dikirim ke layanan penyimpanan sebagai nama aplikasi — BUKAN kunci
-- pencariannya. Keduanya mudah tertukar, dan tertukarnya menaruh berkas di folder yang
-- salah tanpa satu pun galat.
SELECT f.NAMA_FOLDER
  FROM GENERAL.T_FOLDER_STORAGE@asmd.sinarmas.co.id f
 WHERE UPPER(TRIM(f.APLIKASI)) = :1


-- name: catat_akses_unggah
-- Mencatat satu izin unggah.
--
-- Pega menempuhnya lewat `GENERAL.GET_TOKEN_STORAGE`, yang sumbernya sudah diterima
-- (`Database/GENERAL.GET_TOKEN_STORAGE.prc`) dan isinya hanya dua pernyataan:
--
--     select standard_hash('ASMAPP'||SYSTIMESTAMP,'MD5') into VAKSESTOKEN from dual;
--     INSERT INTO GENERAL.GCP_IMAGE(APPNAME,KODEAKSES,USERINPUT,INPUTDATE)
--          values (VAPPNAME, VAKSESTOKEN, VUSERINPUT, sysdate);
--
-- `D-02` melarang memanggil stored procedure, jadi logikanya naik ke Go: tokennya dibentuk
-- di Go dan dikirim sebagai `:2`.
--
-- # Tokennya IKUT terkirim ke layanan penyimpanan (koreksi 2026-09-30)
--
-- `RDB List/GenerateTokenPNCDokumen-SQL.xml:44` menulis OUT procedure ke
-- `{DocAPI.KodeString OUT}`, dan muatan unggah adalah `@GetPageJSONString()` halaman DocAPI
-- (`InsertDokumenPNC` :3903) — jadi `KodeString` ada di JSON. Catatan sebelumnya yang
-- menyatakan sebaliknya salah baca; layanan menolak muatan tanpanya dengan
-- "can't Kodestring null value in json data". Token yang dikirim harus SAMA dengan yang
-- disisipkan di sini.
--
-- Ia tetap ditulis, dan bukan karena meniru: `D-59` menjadikan jejak audit satu-satunya
-- kontrol pengimbang, dan ini satu-satunya jejak yang tercatat sebelum berkasnya terkirim.
INSERT INTO GENERAL.GCP_IMAGE@asmd.sinarmas.co.id
    (APPNAME, KODEAKSES, USERINPUT, INPUTDATE)
VALUES (:1, :2, :3, CURRENT_TIMESTAMP)


-- name: simpan_metadata
-- Mencatat metadata satu dokumen yang sudah terunggah.
--
-- Salinan `RDB List/InsertDataPNCStorage-SQL.xml`, dengan tiga perbedaan yang disengaja:
--
--  1. `COMMIT` di dalam pernyataan DIBUANG. `D-68` memindahkan kepemilikan transaksi ke Go;
--     procedure dan blok PL/SQL yang commit sendiri adalah pola yang justru ditinggalkan.
--  2. `To_date({exp}, 'DD/MM/YYYY HH24:MI:SS')` diganti bind bertipe waktu. Pega mengubah
--     waktu menjadi TEKS lalu mengurainya kembali; setiap langkah itu tempat pergeseran
--     zona waktu (`R-12`), dan tidak satu pun diperlukan.
--  3. `TGL_UPLOAD` ikut diisi. Pega meninggalkannya NULL, sehingga urutan "terbaru lebih
--     dulu" pada daftar tidak punya dasar. Kolomnya sudah ada dan nullable — mengisinya
--     tidak mengubah apa pun bagi Pega, dan memberi daftar kita urutan yang benar.
INSERT INTO GENERAL.T_STORAGE_IMAGE@asmd.sinarmas.co.id
    (IMAGEID, URLPUBLIC, APPFOLDER, EXPDATE, FILENAME,
     APPNAME, STORAGE, TYPEIMAGE, NO_CLAIM, TGL_UPLOAD)
VALUES (:1, :2, :3, :4, :5,
        :6, :7, :8, :9, :10)


-- name: dokumen_per_klaim
-- Daftar dokumen sebuah klaim, terbaru lebih dulu.
--
-- # Kenapa disaring APPNAME juga, bukan NO_CLAIM saja
--
-- Tabel ini dipakai BERSAMA sembilan aplikasi — `klaimasmtest` 113.449 baris, `klaim`
-- 6.781, `jtutest` 4.614, `klaimpnc` 2.050, dan seterusnya. Nomor klaim tidak dijamin unik
-- antar aplikasi; tanpa saringan `APPNAME`, daftar sebuah klaim dapat memuat dokumen milik
-- aplikasi lain yang kebetulan bernomor sama.
--
-- # Baris yang sudah ditandai hapus atau arsip disembunyikan
--
-- `DELETE_DATE` dan `TGL_ARCHIVE` adalah penanda soft delete milik tabel ini. Kueri
-- pembaca WAJIB menyaringnya (`D-66` §8.1) — satu kueri yang lupa akan menampilkan dokumen
-- yang seharusnya sudah hilang.
--
-- Urutannya `TGL_UPLOAD DESC` dengan `IMAGEID` sebagai pemecah seri, supaya dua unggahan
-- pada detik yang sama tetap berurutan tetap — bukan berpindah-pindah tiap muat.
-- `NULLS LAST` menaruh baris warisan tanpa `TGL_UPLOAD` di bawah, bukan di atas.
SELECT s.IMAGEID,
       s.FILENAME,
       s.URLPUBLIC,
       s.APPFOLDER,
       s.EXPDATE,
       s.TYPEIMAGE,
       s.NO_CLAIM,
       s.TGL_UPLOAD
  FROM GENERAL.T_STORAGE_IMAGE@asmd.sinarmas.co.id s
 WHERE UPPER(TRIM(s.NO_CLAIM)) = :1
   AND UPPER(TRIM(s.APPNAME)) = :2
   AND s.DELETE_DATE IS NULL
   AND s.TGL_ARCHIVE IS NULL
 ORDER BY s.TGL_UPLOAD DESC NULLS LAST, s.IMAGEID DESC


-- name: dokumen_menurut_imageid
-- Membaca kembali satu dokumen sesudah metadatanya tercatat.
--
-- Salinan `RDB List/GetURLAndEXPDate-SQL.xml`, yang Pega jalankan tepat sesudah insert:
--
--     select urlpublic AS "URLImage", EXPDATE AS "exp", APPFOLDER AS "appfolder",
--            FILENAME AS "NamaFile"
--       from general.t_storage_image WHERE IMAGEID = {ImageID.CARI1}
--
-- Alasannya tetap berlaku: respons unggah memuat `ImageID` dan `exp`, tetapi TIDAK memuat
-- `URLPUBLIC`. Tanpa membaca kembali, dokumen yang baru diunggah tidak punya alamat yang
-- dapat dibuka sampai halaman dimuat ulang.
--
-- Kolomnya dibuat SAMA PERSIS dengan dokumen_per_klaim supaya satu fungsi pemindai melayani
-- keduanya. Dua pemindai untuk bentuk yang sama adalah dua tempat yang dapat berbeda.
SELECT s.IMAGEID,
       s.FILENAME,
       s.URLPUBLIC,
       s.APPFOLDER,
       s.EXPDATE,
       s.TYPEIMAGE,
       s.NO_CLAIM,
       s.TGL_UPLOAD
  FROM GENERAL.T_STORAGE_IMAGE@asmd.sinarmas.co.id s
 WHERE TRIM(s.IMAGEID) = :1
