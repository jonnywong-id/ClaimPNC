-- Kueri dua tabel Master Penolakan Klaim:
--
--   POOLDATA.MST_PENOLAKAN_KLAIM_1   Status Penolakan 1  ID_ST (kunci), NOTE_ST
--   POOLDATA.MST_PENOLAKAN_KLAIM_2   Status Penolakan 2  ID_ND (kunci), ID_ST (induk)
--
-- PERHATIKAN AKHIRAN NAMANYA. `_1` adalah induk, `_2` adalah anak, dan keduanya sama-sama
-- punya kolom ID_ST dan NOTE_ST — pada anak, kedua kolom itu adalah SALINAN milik
-- induknya. Tertukar sekali saja berarti layar ini menulis ke tabel yang salah tanpa satu
-- pun galat, karena nama kolomnya memang ada di keduanya.
--
-- Kedua tabel DITULIS aplikasi ini. Kewenangan menulisnya berpindah dari Pega ke Go saat
-- modulnya lulus gerbang 2; selama masa paralel, tepat satu sistem yang menulis
-- (ADR-0004, penulis tunggal per tabel). Selama Pega masih menjadi penulisnya, layar ini
-- harus dijalankan dalam modus baca saja di produksi.
--
-- Lima aturan yang mengikat seluruh berkas ini:
--   1. Kolom disebut namanya; SELECT * dilarang.
--   2. Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
--   3. Tanpa NVL, SYSDATE, DECODE, ROWNUM, dan TO_CHAR — SQL harus berjalan sama di
--      Oracle 19c dan PostgreSQL 17+ (D-20).
--   4. Tanpa pemanggilan stored procedure (D-02). Ketiga procedure sumber
--      (MASTERPENOLAKANKLAIM1, MASTERPENOLAKANKLAIM2, INSERTMASTERREJECTEDKOMITE)
--      logikanya naik ke Go; objeknya boleh ditinggalkan (D-68).
--   5. TIDAK ADA DELETE. Seluruh export tidak memuat satu pun DELETE terhadap kedua tabel
--      ini, dan keduanya tidak punya kolom penanda terhapus yang dapat dipakai D-66.

-- name: rejection_parent_list
--
-- Daftar Status Penolakan 1 untuk pilihan di layar.
--
-- TIDAK ADA ASALNYA DI EXPORT, dan itu bukan kelalaian pembacaan: POOLDATA.MST_PENOLAKAN_KLAIM_1
-- hanya muncul di dalam `Database/MASTERPENOLAKANKLAIM1.prc`, tidak di satu pun rule SQL.
-- Layar lama memang tidak pernah membacanya — kedua isian induknya kotak teks bebas, dan
-- setiap simpan menerbitkan baris baru. Kueri ini lahir bersama perbaikan yang diputuskan
-- Work Owner 2026-09-19 (lihat masterpenolakan.Input).
--
-- KENAPA DIURUTKAN MENURUT NAMA, bukan menurut ID seperti kueri-kueri lain di modul ini.
-- Karena tidak ada kueri lama yang urutannya harus disamai — jadi tidak ada selisih
-- kesetaraan yang mungkin timbul — sedangkan yang dibaca manusia di sini adalah namanya,
-- bukan nomornya. Pada tabel yang kemungkinan besar sudah memuat banyak nama kembar akibat
-- cacat yang baru diperbaiki itu, pengurutan menurut nama juga membuat kembarannya
-- berdampingan alih-alih tersebar.
--
-- ID ikut menjadi kunci urutan kedua supaya urutannya tetap sama pada dua pemanggilan
-- berturut-turut; tanpa itu, baris bernama sama dapat berpindah tempat di antara
-- pemuatan daftar dan pemilihan pengguna.
SELECT ID_ST,
       NOTE_ST
  FROM POOLDATA.MST_PENOLAKAN_KLAIM_1
 ORDER BY NOTE_ST ASC, ID_ST ASC

-- name: rejection_parent_get
--
-- Membaca satu Status Penolakan 1, dipakai saat menyalin namanya ke baris tingkat 2.
--
-- TRIM pada penyaringnya sama alasannya dengan modul Master Status Progres: kolom kunci
-- di tabel-tabel warisan ini bertipe teks yang lebarnya belum diketahui (R-08), dan bila
-- ia CHAR berlebar tetap, nilainya dipadatkan dengan spasi tanpa memberi tanda apa pun.
-- Oracle membandingkan CHAR dengan CHAR secara blank-padded — sehingga rangkaian teks
-- `= '12'` pada kueri lama tetap cocok dengan "12 ". Tetapi parameter binding bertipe
-- VARCHAR2, dan perbandingan CHAR dengan VARCHAR2 memakai non-padded comparison: "12 "
-- tidak sama dengan "12", dan barisnya tidak ketemu.
--
-- Jadi menyalin `= :1` apa adanya justru MENGUBAH perilaku, bukan mempertahankannya.
-- Biayanya index atas ID_ST tidak terpakai; dapat diterima pada tabel master berbaris
-- sedikit, dan TIDAK boleh ditiru pada tabel besar.
SELECT ID_ST,
       NOTE_ST
  FROM POOLDATA.MST_PENOLAKAN_KLAIM_1
 WHERE TRIM(ID_ST) = :1

-- name: rejection_parent_list_id_locked
--
-- Mengunci seluruh baris tingkat 1, lalu mengembalikan ID-nya untuk menurunkan nomor
-- berikutnya.
--
-- KENAPA FOR UPDATE. Nomor berikutnya diturunkan dari isi tabel — di procedure lama lewat
-- `select nvl(max(to_number(ID_ST)),0)+1` (`MASTERPENOLAKANKLAIM1.prc:6`) yang dijalankan
-- sebagai pernyataan lepas, lalu hasilnya dipakai INSERT beberapa baris kemudian. Di
-- antara keduanya tidak ada apa pun yang menghalangi penambahan lain masuk lebih dulu,
-- sehingga dua petugas yang menambah bersamaan dapat menerima nomor yang sama.
--
-- FOR UPDATE membuat penambahan kedua menunggu sampai yang pertama selesai, lalu membaca
-- ulang termasuk baris yang baru masuk. Didukung Oracle maupun PostgreSQL (D-20).
SELECT ID_ST
  FROM POOLDATA.MST_PENOLAKAN_KLAIM_1
 FOR UPDATE

-- name: rejection_parent_insert
--
-- Asal: Database/MASTERPENOLAKANKLAIM1.prc:9
--
--   INSERT INTO POOLDATA.MST_PENOLAKAN_KLAIM_1 (ID_ST,NOTE_ST) VALUES (id_mst,tnotest);
--
-- Kedua kolom itu saja — tabelnya memang tidak punya kolom lain yang disentuh procedure
-- mana pun. Tidak ada kolom pencatat siapa dan kapan, sehingga jejak audit perubahan
-- master (D-28, modul S-5) belum dapat disandarkan padanya. Dicatat sebagai keterbatasan,
-- bukan ditambal dengan kolom yang dikarang: menambah kolom menuntut persetujuan Work
-- Owner dan pelaksanaan DBA (D-63).
INSERT INTO POOLDATA.MST_PENOLAKAN_KLAIM_1 (ID_ST, NOTE_ST)
VALUES (:1, :2)

-- name: rejection_parent_check_table
--
-- Memastikan tabel ada dan dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
SELECT ID_ST
  FROM POOLDATA.MST_PENOLAKAN_KLAIM_1
 WHERE 1 = 0

-- name: rejection_list
--
-- Asal: RDB List/BrowseStatusPenolakanKlaim2-SQL.xml
--
--   SELECT A.ID_ST AS "CaseID", A.NOTE_ST AS "City", A.NOTE_ND AS "CityID",
--          A.ID_ND AS "District", A.STATUS AS "DistrictID", USER_INPUT as "UserTeknis",
--          case when A.STATUS='1' then 'APPROVED' when A.STATUS='2' then 'REJECTED'
--               ELSE 'MENUNGGU' end as "AnaylstRemarks",
--          A.NOTEAPPROVED as "NoteKasir"
--     FROM POOLDATA.MST_PENOLAKAN_KLAIM_2 a
--   {ASIS:MasterCheckerPenolakan.RemakApprove}
--    ORDER BY A.ID_ST ASC
--
-- TIGA hal yang berbeda dari kueri lama, ketiganya disengaja:
--
-- 1. POTONGAN `{ASIS:...}` TIDAK DIBAWA. Ia merangkai teks SQL dari nilai klipboard —
--    persis celah yang §4.3 `08-TECHNICAL-STRATEGY.md` tutup. Yang mengisinya adalah
--    `Activity/BrowseStatusPenolakanKlaim_2-Act.xml` langkah 2, dan langkah itu
--    BERPRASYARAT `Param.master=="1"` yang hanya dikirim layar checker pada Inbox
--    Manager. Dari layar Master Penolakan Klaim potongan itu tetap kosong, sehingga
--    kueri ini memang menampilkan seluruh baris — bukan hanya yang berstatus menunggu.
--
--    Membawanya juga berarti membawa kebocoran antarlayar: nilai yang tertinggal dari
--    layar checker dalam sesi yang sama akan diam-diam menyaring layar ini.
--
-- 2. DERIVASI `case when ... end` TIDAK DIBAWA. Ia pemformatan untuk tampilan, dan
--    §4.3 menetapkan pemformatan dilakukan di Go. Yang dibaca adalah kolom STATUS apa
--    adanya; penerjemahannya ada di masterpenolakan.ApprovalStatus.Label, dengan teks
--    yang sama persis termasuk huruf besarnya.
--
-- 3. TANGGALKIRIM IKUT DIBACA meski kueri lama tidak membacanya. Kolomnya memang ditulis
--    procedure (`MASTERPENOLAKANKLAIM2.prc:10`), dan tanpa itu layar tidak dapat
--    menyebutkan kapan sebuah baris diajukan — satu-satunya penanda urutan pengajuan
--    yang dimiliki tabel ini.
--
-- ORDER BY ID_ST dipertahankan supaya urutan baris di layar sama dengan urutan di Pega
-- saat uji kesetaraan. Ia pengurutan TEKS pada kolom bertipe teks — "10" mendahului "9" —
-- dan itu perilaku kueri lama yang tidak diperbaiki di sini: memperbaikinya mengubah
-- urutan yang dilihat pengguna, dan itu selisih yang tidak diminta.
--
-- ID_ND ditambahkan sebagai kunci urutan KEDUA. Beberapa baris tingkat 2 berbagi satu
-- ID_ST — memang itu gunanya induk — sehingga kueri lama tidak menentukan urutan di
-- antara mereka sama sekali dan basis data bebas memulangkannya dalam urutan apa pun.
-- Menambahkan kunci kedua tidak bertentangan dengan urutan lama; ia hanya membuat yang
-- tadinya sembarang menjadi tetap, sehingga daftar tidak berubah-ubah di antara dua
-- pemuatan.
SELECT ID_ND,
       NOTE_ND,
       ID_ST,
       NOTE_ST,
       STATUS,
       USER_INPUT,
       TANGGALKIRIM,
       APPROVEBY,
       TANGGAL_APPROVE,
       NOTEAPPROVED
  FROM POOLDATA.MST_PENOLAKAN_KLAIM_2
 ORDER BY ID_ST ASC, ID_ND ASC

-- name: rejection_get
--
-- Asal: RDB List/UpdateStatusPenolakanKlaim2-SQL.xml — kueri yang memuat satu baris ke
-- modal. Namanya di Pega berawalan "Update" padahal ia SELECT; nama di sini menyebutkan
-- apa yang benar-benar dilakukannya.
--
-- Penyaringnya `where A.ID_ND={TempSearchStatusProgress2.CaseID}` — perhatikan ia
-- menyaring ID_ND, kunci baris ini sendiri, meskipun nama page-nya berbunyi "CaseID".
-- TRIM ditambahkan dengan alasan yang sama seperti rejection_parent_get.
SELECT ID_ND,
       NOTE_ND,
       ID_ST,
       NOTE_ST,
       STATUS,
       USER_INPUT,
       TANGGALKIRIM,
       APPROVEBY,
       TANGGAL_APPROVE,
       NOTEAPPROVED
  FROM POOLDATA.MST_PENOLAKAN_KLAIM_2
 WHERE TRIM(ID_ND) = :1

-- name: rejection_list_id_locked
--
-- Mengunci seluruh baris tingkat 2, lalu mengembalikan ID-nya untuk menurunkan nomor
-- berikutnya. Alasannya sama dengan rejection_parent_list_id_locked; asal nomornya
-- `MASTERPENOLAKANKLAIM2.prc:6`.
SELECT ID_ND
  FROM POOLDATA.MST_PENOLAKAN_KLAIM_2
 FOR UPDATE

-- name: rejection_insert
--
-- Asal: Database/MASTERPENOLAKANKLAIM2.prc:9
--
--   INSERT INTO POOLDATA.MST_PENOLAKAN_KLAIM_2 a
--          (A.ID_ST,A.NOTE_ST,A.ID_ND,A.NOTE_ND,A.USER_INPUT,A.TANGGALKIRIM,A.STATUS)
--   VALUES (ID_1,note_1,id_mst,note_2,user_input,sysdate,'0');
--
-- Ketujuh kolom itu saja. Tiga kolom persetujuan — APPROVEBY, TANGGAL_APPROVE,
-- NOTEAPPROVED — sengaja TIDAK disebut, persis seperti procedure lama, sehingga basis
-- data mengisinya dengan default kolomnya sendiri. Yang mengisinya adalah layar checker
-- pada Inbox Manager, bukan layar ini.
--
-- STATUS diikat sebagai parameter (:7) alih-alih ditulis '0' di dalam teks: nilainya
-- berasal dari masterpenolakan.StatusPending, sehingga hanya ada SATU tempat yang tahu
-- sandi status — dan bila kelak sandinya berubah, ia berubah di satu tempat saja.
--
-- TANGGALKIRIM diisi dari aplikasi (:6), bukan dari SYSDATE seperti procedure lama.
-- Alasannya mengikat seluruh aplikasi: waktu berasal dari satu seam (platform/clock)
-- sehingga dapat diuji deterministik, dan disimpan UTC. SYSDATE mengambil zona waktu
-- server basis data, yang di sistem lama justru menjadi sumber penanganan zona waktu
-- manual yang §4.4 `08-TECHNICAL-STRATEGY.md` larang.
--
-- KONSEKUENSI YANG DISADARI: selama masa paralel, baris yang ditulis Pega memuat waktu
-- server (WIB) sedangkan baris yang ditulis aplikasi ini memuat UTC, sehingga keduanya
-- terpaut tujuh jam pada kolom yang sama. Itu wujud nyata R-12 pada tabel ini. Ia tidak
-- dapat dihindari tanpa melanggar §4.4, dan terbatas pada satu kolom yang tidak dipakai
-- perhitungan mana pun — hanya ditampilkan. Layar karena itu menyebut waktunya sebagai
-- waktu pengajuan, bukan sebagai penentu urutan.
INSERT INTO POOLDATA.MST_PENOLAKAN_KLAIM_2
       (ID_ST, NOTE_ST, ID_ND, NOTE_ND, USER_INPUT, TANGGALKIRIM, STATUS)
VALUES (:1, :2, :3, :4, :5, :6, :7)

-- name: rejection_update
--
-- Asal: Database/MASTERPENOLAKANKLAIM2.prc:14
--
--   update POOLDATA.MST_PENOLAKAN_KLAIM_2 a
--      set A.ID_ST=ID_1, A.NOTE_ST=note_1, A.NOTE_ND=note_2, A.USER_INPUT=user_input,
--          A.TANGGALKIRIM=sysdate, A.STATUS='0'
--    where A.ID_ND=ID_2;
--
-- Keenam kolom yang di-SET direplikasi apa adanya, TERMASUK `STATUS='0'`. Artinya
-- mengubah teks penolakan MENGEMBALIKAN barisnya ke antrean persetujuan: persetujuan
-- yang sudah ada batal, dan checker harus memutuskannya lagi.
--
-- Perilaku itu dipertahankan atas keputusan Work Owner 2026-09-19. Ia masuk akal secara
-- bisnis — teks yang sudah disetujui tidak boleh berubah diam-diam — dan mereplikasinya
-- berarti uji kesetaraan gerbang 1 tidak melihat selisih.
--
-- TIGA KOLOM PERSETUJUAN TIDAK IKUT DIBERSIHKAN, juga persis seperti procedure lama.
-- Akibatnya baris berstatus MENUNGGU masih memuat nama penyetuju dan tanggal keputusan
-- sebelumnya. Itu jejak keputusan yang pernah ada, bukan keadaan yang berlaku; layar
-- menyebutnya demikian alih-alih menyembunyikannya.
--
-- ID_ND tidak pernah ikut di-SET: ia kunci baris, dan procedure lama pun hanya memakainya
-- sebagai penyaring WHERE. TRIM pada penyaring, alasannya sama seperti rejection_get.
UPDATE POOLDATA.MST_PENOLAKAN_KLAIM_2
   SET ID_ST        = :1,
       NOTE_ST      = :2,
       NOTE_ND      = :3,
       USER_INPUT   = :4,
       TANGGALKIRIM = :5,
       STATUS       = :6
 WHERE TRIM(ID_ND) = :7

-- name: rejection_check_table
--
-- Memastikan tabel ada dan dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
SELECT ID_ND
  FROM POOLDATA.MST_PENOLAKAN_KLAIM_2
 WHERE 1 = 0
