-- Kueri Master Masking: POOLDATA.MST_PROTEKSI_DATA_PNC.
--
-- # Tabel mana yang dibaca dan ditulis
--
-- Tabel dasarnya, bukan view. Layar Master Masking adalah SATU-SATUNYA penulis tabel ini
-- di sistem lama, dan itu diperiksa ke seluruh export, bukan diandaikan:
--
--     Database/UPDATE_LOG_PROTEKSI.prc            satu-satunya berkas ber-INSERT/UPDATE
--     RDB List/SaveMstProteksi_SQL-SQL.xml        satu-satunya pemanggil procedure itu
--     RDB List/DeleteMstProteksi_SQL-SQL.xml      satu-satunya UPDATE lain (STS_AKTF)
--     Activity/InsermaskingDataKlaimPnc_-Act.xml  satu-satunya pemanggil rule itu
--
-- Rule lain yang menyentuh tabel ini hanya MEMBACA:
--
--     RDB List/GetTipeProteksi-SQL.xml                 baris aktif
--     RDB List/SearchMasking_SQL-SQL.xml               grid layar
--     Activity/CekmaskingDataPerLoginUserKlaim-Act.xml penegakan masking di layar klaim
--
-- Memindahkan layarnya ke sini karena itu memindahkan kepemilikan tabelnya secara utuh
-- (`P-1`). Pega berubah menjadi pembaca saja, dan tidak ada data yang kembar.
--
-- POOLDATA.BRANCH hanya DIBACA — ia dimiliki sistem lain dan tidak pernah ditulis di sini.
--
-- # Kenapa tidak lagi lewat POOLDATA.Update_Log_Proteksi
--
-- `D-02` menetapkan logika stored procedure naik ke Go. `D-68` menambahkan alasan yang
-- lebih keras, dan procedure ini contoh persisnya: ia melakukan COMMIT sendiri di TIGA
-- cabang (`:57`, `:99`, `:161`) lalu menaruh ROLLBACK sesudahnya, sehingga kegagalan di
-- tengah meninggalkan data setengah jalan. Kontrak galatnya pun berbasis teks — parameter
-- keluaran `MSG` berisi `'1'` bila berhasil dan kalimat berbahasa Indonesia bila gagal.
--
-- # Kolom PASSWORD
--
-- Diisi literal tetap, mengikuti sistem lama apa adanya — keputusan Work Owner 2026-09-20.
-- Lihat catatan lengkapnya di mastermasking.go (konstanta legacyPassword). Kolomnya TIDAK
-- pernah dibaca modul ini dan tidak pernah tampil di layar.
--
-- # Yang diperiksa langsung ke basis data pada 2026-09-20 (portal ASM, baca-saja)
--
--     25 baris          15 'AKTIF', 10 'TIDAK AKTIF'
--     STS_KTP/EMAIL/NOTELP  hanya 'Ya' dan 'Tidak' — BUKAN '1'/'0'
--     CABANG            25 dari 25 cocok ke POOLDATA.BRANCH.ID
--     CABANG+LOGIN      25 pasangan unik dari 25 baris
--     indeks            hanya MST_PROTEKSI_DATA_PNC_INDEX, NONUNIQUE
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding —
-- layar lama merangkai penyaring pencarian langsung ke teks SQL lewat `{ASIS:…}`, dan itu
-- persis celah yang `docs/Steering/07` §4.3 tutup tanpa perkecualian.

-- name: masking_list
--
-- Satu kueri melayani KEEMPAT tipe pencarian, dan setiap bind muncul TEPAT SEKALI.
--
-- Keempatnya diambil apa adanya dari `Activity/SearchDataMasking-Act.xml`, yang
-- menyusun satu penyaring `InputSearch.CARI1` menurut `SearchData.Type`:
--
--     Type "1"  tidak menyaring apa pun
--     Type "2"  CABANG IN (SELECT ID FROM POOLDATA.BRANCH WHERE BRANCHNAME LIKE '%…%')
--     Type "3"  LOGIN LIKE '%…%'
--     Type "4"  STS_AKTF = '<nilai>'
--
-- Kata kunci sengaja diikat dua kali sebagai :2 dan :3 — bukan satu bind yang diulang —
-- supaya kueri ini tidak bergantung pada perilaku driver terhadap penanda berulang.
-- Pemanggil mengirim nilai yang sama untuk keduanya; :4 membawa nilai status.
--
-- Penyaring cabang menelusuri NAMA cabang, bukan kodenya. Di sini ia menjadi syarat atas
-- hasil LEFT JOIN, yang mengembalikan baris yang sama tanpa subkueri terpisah.
--
-- Penyaring status cocok PERSIS, bukan sebagian — layar lama memakai `=`, bukan `LIKE`.
--
-- LEFT JOIN, bukan INNER: baris yang cabangnya tidak dikenal HARUS tetap terlihat. Tidak
-- ada yang menggantung hari ini, tetapi menyembunyikannya bila kelak terjadi berarti
-- sebuah kewenangan melihat data pribadi hidup tanpa pernah tampil di layar mana pun.
--
-- Urutan mengikuti SearchMasking_SQL: TANGGALINPUT menurun. ID_MST dipakai sebagai
-- pemecah seri supaya urutannya tetap sama antarpemuatan — TANGGALINPUT bertipe DATE
-- (presisi detik), sehingga dua baris yang disimpan pada detik yang sama akan berpindah
-- tempat tanpa itu.
SELECT m.ID_MST,
       m.CABANG,
       b.BRANCHNAME,
       m.LOGIN,
       m.MODUL,
       m.SUBMODUL,
       m.LOGSEARCH,
       m.LOGSEEN,
       m.STS_KTP,
       m.STS_EMAIL,
       m.STS_NOTELP,
       m.STS_AKTF,
       m.USERINPUT,
       m.TANGGALINPUT
  FROM POOLDATA.MST_PROTEKSI_DATA_PNC m
  LEFT JOIN POOLDATA.BRANCH b ON b.ID = m.CABANG
 WHERE (CASE :1
          WHEN 'login'  THEN CASE WHEN UPPER(m.LOGIN) LIKE :2 THEN 1 ELSE 0 END
          WHEN 'cabang' THEN CASE WHEN UPPER(b.BRANCHNAME) LIKE :3 THEN 1 ELSE 0 END
          WHEN 'status' THEN CASE WHEN UPPER(TRIM(m.STS_AKTF)) = :4 THEN 1 ELSE 0 END
          ELSE 1
        END) = 1
 ORDER BY m.TANGGALINPUT DESC, m.ID_MST DESC

-- name: masking_get
SELECT m.ID_MST,
       m.CABANG,
       b.BRANCHNAME,
       m.LOGIN,
       m.MODUL,
       m.SUBMODUL,
       m.LOGSEARCH,
       m.LOGSEEN,
       m.STS_KTP,
       m.STS_EMAIL,
       m.STS_NOTELP,
       m.STS_AKTF,
       m.USERINPUT,
       m.TANGGALINPUT
  FROM POOLDATA.MST_PROTEKSI_DATA_PNC m
  LEFT JOIN POOLDATA.BRANCH b ON b.ID = m.CABANG
 WHERE m.ID_MST = :1

-- name: masking_find_pair
--
-- Pasangan CABANG+LOGIN adalah kunci alami tabel ini — procedure lama menolak insert bila
-- pasangannya sudah ada (`:29-31`) dan mencocokkan pasangan itu saat update (`:74-76`).
--
-- Dibandingkan dengan UPPER(TRIM(…)) di kedua sisi, sejalan dengan mastermasking.PairKey.
-- Perbandingan yang peka huruf akan membiarkan `BUDI` dan `budi` hidup berdampingan
-- sebagai dua kewenangan terpisah, dan yang satu dapat luput saat dicabut.
SELECT m.ID_MST,
       m.CABANG,
       b.BRANCHNAME,
       m.LOGIN,
       m.MODUL,
       m.SUBMODUL,
       m.LOGSEARCH,
       m.LOGSEEN,
       m.STS_KTP,
       m.STS_EMAIL,
       m.STS_NOTELP,
       m.STS_AKTF,
       m.USERINPUT,
       m.TANGGALINPUT
  FROM POOLDATA.MST_PROTEKSI_DATA_PNC m
  LEFT JOIN POOLDATA.BRANCH b ON b.ID = m.CABANG
 WHERE UPPER(TRIM(m.CABANG)) = :1
   AND UPPER(TRIM(m.LOGIN))  = :2
 ORDER BY m.ID_MST
 FETCH FIRST 1 ROW ONLY

-- name: masking_next_id
--
-- Nomor berikutnya dibentuk `MAX(TO_NUMBER(ID_MST))+1`, persis procedure lama
-- (`Database/UPDATE_LOG_PROTEKSI.prc:33-34`) — keputusan Work Owner 2026-09-20, supaya
-- penomorannya tetap bersambung dengan yang dibuat Pega selama masa paralel.
--
-- Bahaya yang DISADARI dan diterima: dua penyimpanan bersamaan dapat membaca nilai yang
-- sama. Tabel ini tidak punya indeks unik yang menolaknya, sehingga penanganannya ada di
-- Go — nomor dibaca DI DALAM transaksi yang sama dengan penyisipannya, dan bentrokan
-- dicoba ulang. Itu mempersempit peluangnya, bukan menutupnya. Indeks unik atas ID_MST
-- diusulkan ke DBA bersama migrasi modul ini.
--
-- COALESCE, bukan NVL — `docs/Steering/09-DATABASE-STRATEGY.md` §4 menetapkan padanan
-- portabel supaya kueri yang sama berjalan di PostgreSQL 17+ kelak.
SELECT COALESCE(MAX(TO_NUMBER(ID_MST)), 0) + 1
  FROM POOLDATA.MST_PROTEKSI_DATA_PNC

-- name: masking_insert
--
-- TANGGALINPUT diisi dari aplikasi, bukan SYSDATE.
--
-- Itu perbedaan yang disengaja dari procedure lama, dan alasannya ada di
-- `docs/Steering/07` §4.4: waktu datang dari satu tempat (seam Clock, `F-5`), bukan dari
-- jam basis data yang tidak dapat dikendalikan maupun diuji. Ia juga menutup `R-12` —
-- pergeseran zona waktu yang tidak terdeteksi karena penulis dan pembacanya memakai jam
-- yang berbeda.
INSERT INTO POOLDATA.MST_PROTEKSI_DATA_PNC
       (ID_MST, CABANG, LOGIN, MODUL, SUBMODUL,
        LOGSEARCH, LOGSEEN, STS_KTP, STS_EMAIL, STS_NOTELP,
        STS_AKTF, PASSWORD, USERINPUT, TANGGALINPUT)
VALUES (:1, :2, :3, :4, :5,
        :6, :7, :8, :9, :10,
        :11, :12, :13, :14)

-- name: masking_update
--
-- Kolom yang diubah sama persis dengan cabang `T_ACTION = 'update'` pada
-- `Database/UPDATE_LOG_PROTEKSI.prc:81-95` — TERMASUK STS_AKTF, karena layar lama memang
-- memuat isian "STATUS" pada form (`MasterProteksi_Sec:22449`, terikat
-- `InputData.BranchID` yang dipetakan ke `T_STSAKTF`).
--
-- Satu perbedaan yang disengaja: procedure lama mencocokkan baris dengan
-- `WHERE CABANG = … AND LOGIN = …`, sehingga MENGUBAH cabang atau login sebuah baris
-- mustahil — pernyataannya tidak akan menemukan barisnya. Di sini pencocokannya memakai
-- ID_MST, yang membuat keduanya dapat disunting sebagaimana form memang menampilkannya.
--
-- ID_MST sendiri tidak pernah ikut berubah; ia penanda baris.
UPDATE POOLDATA.MST_PROTEKSI_DATA_PNC
   SET CABANG       = :1,
       LOGIN        = :2,
       MODUL        = :3,
       SUBMODUL     = :4,
       LOGSEARCH    = :5,
       LOGSEEN      = :6,
       STS_KTP      = :7,
       STS_EMAIL    = :8,
       STS_NOTELP   = :9,
       STS_AKTF     = :10,
       PASSWORD     = :11,
       USERINPUT    = :12,
       TANGGALINPUT = :13
 WHERE ID_MST = :14

-- name: masking_set_active
--
-- Inilah "hapus" layar lama. `RDB List/DeleteMstProteksi_SQL-SQL.xml` menjalankan
-- pernyataan yang sama persis, hanya tanpa mencatat pelakunya.
--
-- Pelaku dan waktunya IKUT dicatat di sini, dan itu penambahan yang disengaja: mencabut
-- kewenangan melihat data pribadi adalah peristiwa yang harus dapat ditelusuri, dan
-- `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang yang tersisa.
UPDATE POOLDATA.MST_PROTEKSI_DATA_PNC
   SET STS_AKTF     = :1,
       USERINPUT    = :2,
       TANGGALINPUT = :3
 WHERE ID_MST = :4

-- name: branch_list
--
-- Pilihan cabang untuk isian CABANG. POOLDATA.BRANCH hanya DIBACA.
--
-- Kata kunci dicocokkan ke NAMA maupun KODE, karena pengguna yang hafal kodenya mengetik
-- kode, dan yang tidak mengetik nama. Kata kunci kosong mengembalikan awal daftar —
-- itulah yang tampil saat form baru dibuka.
--
-- Dibatasi FETCH FIRST supaya 803 baris tidak dikirim seluruhnya ke peramban. Angkanya
-- datang dari pemanggil, bukan dipatok di sini.
SELECT ID,
       BRANCHNAME
  FROM POOLDATA.BRANCH
 WHERE (:1 IS NULL
        OR UPPER(BRANCHNAME) LIKE :2
        OR UPPER(ID) LIKE :3)
 ORDER BY BRANCHNAME
 FETCH FIRST :4 ROWS ONLY

-- name: branch_exists
SELECT COUNT(*)
  FROM POOLDATA.BRANCH
 WHERE ID = :1
