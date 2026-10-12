-- Kueri modul Inbox Progress Claim: pemantauan progres klaim yang masih berjalan.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- SELURUH tabel yang dibaca berkas ini milik sistem lama. Tidak ada satu pun pernyataan
-- yang menulis, dan memang tidak boleh ada: selama masa paralel setiap tabel hanya boleh
-- ditulis SATU sistem, dan tabel-tabel ini milik Pega (`P-1`). Region Approval dan tombol
-- Input Progress Claim — satu-satunya bagian layar ini yang menulis — sengaja berada di
-- luar lingkup (keputusan Work Owner 2026-09-21).
--
-- ============================================================================
-- PEMETAAN TIGA ARAH — alias grid Pega -> kolom sebenarnya -> arti
-- ============================================================================
--
-- Layar ini kasus paling pekat dari utang teknis `03-CURRENT-ARCHITECTURE.md` §4.2: nomor
-- klaim dan nomor polis bahkan TERTUKAR aliasnya.
--
-- Alias Pega            Kolom sebenarnya                 Alias di sini
-- --------------------- -------------------------------- ---------------------
-- CaseID            (!) a.NOKLAIM (nomor klaim)          CLAIM_NUMBER
-- ClaimNo           (!) a.NOPOLIS (nomor polis)          POLICY_NUMBER
-- NoKTP             (!) a.NOPOLIS (nomor polis, lagi)    tidak dibawa — kembar
-- District          (!) T_CLAIM_PNC.QQNAME               INSURED_NAME
-- DateForAging      (!) a.TGLKLAIM (tgl registrasi)      REGISTER_DATE
-- DateOfLoss            a.DATEOFLOSS                     LOSS_DATE
-- Country           (!) a.LGB_NOTE                       LGB_NOTE
-- UserTeknis        (!) a.PIC                            TECHNICAL_PIC
-- City              (!) GET_POSISI_PROGRESS_PNC 'POSISI' lihat kueri `positions`
-- CityID            (!) GET_POSISI_PROGRESS_PNC 'sts_prg1' lihat kueri `positions`
-- CountryID         (!) GET_POSISI_PROGRESS_PNC 'sts_prg2' lihat kueri `positions`
-- AnalystTransferD~ (!) GET_POSISI_PROGRESS_PNC 'nextfu'  lihat kueri `positions`
-- KomiteApproveDate (!) MIN(GCNM_PROGRESS_CLAIM.NEXT_FOLLOWUP) EARLIEST_FOLLOW_UP
-- TanggalAnalystSe~ (!) a.TGL_PROSES                     PROCESS_DATE
-- ProdKe                a.PROD_KE                        PROD_KE
--
-- Rekap per PIC lebih jauh lagi — empat dari enam aliasnya menyebut atribut klaim padahal
-- seluruhnya hasil COUNT:
--
-- Alias Pega   Isi sebenarnya                              Alias di sini
-- ------------ ------------------------------------------- -----------------
-- PIC          nama petugas                                PIC
-- NOKLAIM  (!) COUNT klaim yang ditangani                  CLAIM_COUNT
-- NOAKSEP  (!) COUNT pembaruan progres, di luar AUTO%      UPDATE_COUNT
-- REINSURER(!) COUNT tindak lanjut jatuh tempo HARI INI    DUE_TODAY_COUNT
-- STSKLAIM (!) COUNT tindak lanjut TEPAT WAKTU             ON_TIME_COUNT
-- NOPOLIS  (!) COUNT tindak lanjut TERLAMBAT               LATE_COUNT
--
-- Tanda (!) menandai nama yang sama sekali tidak menyatakan isinya. **Judul kolom di layar
-- tetap memakai alias ini** atas keputusan Work Owner 2026-09-21 — lihat view.go. Yang
-- tidak dibawa adalah nama di dalam kode.
--
-- ============================================================================
-- ENAM HAL YANG BERUBAH DARI KUERI LAMA, DAN ALASANNYA
-- ============================================================================
--
-- 1. PARAMETER BINDING, bukan perangkaian nilai.
--    Ketiga kueri lama menyusun klausa WHERE-nya di activity lalu menyisipkannya sebagai
--    POTONGAN SQL: `{ASIS:tempgetpic.MCL_NAME}`, `{ASIS:TempCabang.District}`,
--    `{ASIS:TempBisnis.GROUP_PANEL}`, `{ASIS:TempBisnis.NOAKSEP}`, `{ASIS:TempBisnis.PIC}`,
--    ditambah klausa paginasinya sendiri lewat `{ASIS:Pagination.FirstRow}`.
--    Yang paling terbuka adalah kotak cari — isinya dirangkai apa adanya menjadi
--    `like '%"+TempRefresh.ClaimNo+"%'`. `08-TECHNICAL-STRATEGY.md` §4.3 melarangnya tanpa
--    perkecualian, dan larangan itu TIDAK ikut dikecualikan oleh keputusan "replikasi apa
--    adanya": yang direplikasi adalah perilaku bisnis, bukan celah injeksi.
--
-- 2. TANPA PEMANGGILAN STORED FUNCTION.
--    `GET_POSISI_PROGRESS_PNC` dipanggil EMPAT KALI untuk setiap baris grid — sekali per
--    kategori — dan setiap panggilan mengulang kursor yang sama persis. `D-02` menetapkan
--    logikanya naik ke aplikasi. Penggantinya kueri `positions` di bawah: satu kueri untuk
--    seluruh baris satu halaman, dan penggabungan antarposisi terjadi di Go.
--
-- 3. OFFSET ... FETCH NEXT, bukan ROW_NUMBER.
--    Kueri lama memaginasi dengan `ROW_NUMBER() OVER (...)` lalu menyaring `"rn"` di
--    pembungkusnya. `09-DATABASE-STRATEGY.md` §3.3 menetapkan `OFFSET ... FETCH NEXT`,
--    yang didukung Oracle 12c+ dan PostgreSQL. Hasilnya sama; yang hilang hanyalah kolom
--    bantu `"rn"` yang tidak pernah ditampilkan.
--
-- 4. TANPA TO_CHAR UNTUK MEMBANDINGKAN TANGGAL.
--    Kueri lama membandingkan tanggal dengan mengubah keduanya menjadi teks
--    (`to_char(c.next_followup,'dd/mm/yyyy') = to_char(sysdate,'dd/mm/yyyy')`). Itu
--    membuang index dan memaksa pemindaian penuh. Di sini perbandingannya antar-DATE.
--
-- 5. TANGGAL HARI INI DI-BIND, bukan dibaca dari SYSDATE.
--    `SYSDATE` adalah jam server basis data. Aturan "jatuh tempo hari ini" berbasis tanggal
--    WIB, dan sistem lama menyusunnya dengan menambahkan tujuh jam secara manual
--    (`addCalendar(...,7,0,0)`) — yang `F-5` larang. Di sini tanggalnya datang dari seam
--    Clock sebagai bind.
--
-- 6. URUTAN YANG PASTI.
--    Kueri lama mengurutkan `tgl_proses ASC` saja, sehingga baris ber-`tgl_proses` sama
--    dapat berpindah halaman antar-permintaan dan satu baris muncul dua kali sementara
--    baris lain tidak pernah muncul. Nomor klaim ditambahkan sebagai pemutus seri.
--
-- ============================================================================
-- YANG SENGAJA BELUM ADA: PENYARING CABANG
-- ============================================================================
--
-- Kueri lama menyaring cabang lewat `{ASIS:TempCabang.District}`, yang diisi
-- `RDB List/GetIDCabang-SQL.xml` — dan kueri itu menembus DB Link:
--
--     from branch a, hrdasm.v_hrd_mst@asmd.sinarmas.co.id c,
--          lst_user_asuransi@asmd.sinarmas.co.id b
--
-- DB Link `@ASMD` belum punya API pengganti (`R-03`, `D-25`). Penyaring cabang karena itu
-- TIDAK dibangun, sama seperti di Inbox Admin. Akibatnya untuk sekarang seluruh pengguna
-- melihat klaim seluruh cabang — persis seperti yang di sistem lama hanya berlaku bagi
-- pengguna kantor pusat. Keterbatasan itu dinyatakan di layar, bukan disembunyikan.


-- name: claims_outstanding_list
--
-- Region Outstanding — RDB List/DataProgressClaim-SQL.xml
--
-- Seluruh klaim yang belum tutup. `STSKLAIM NOT IN ('1','2','3')` dibawa apa adanya; ia
-- penanda klaim selesai, ditolak, dan dibatalkan.
--
-- Bind: :1 kata kunci (boleh NULL) · :2 offset · :3 jumlah baris
SELECT a.NOKLAIM                                         AS CLAIM_NUMBER,
       a.NOPOLIS                                         AS POLICY_NUMBER,
       (SELECT MAX(c.QQNAME)
          FROM POOLDATA.T_CLAIM_PNC c
         WHERE c.CLAIMNO = a.NOKLAIM)                    AS INSURED_NAME,
       a.TGLKLAIM                                        AS REGISTER_DATE,
       a.DATEOFLOSS                                      AS LOSS_DATE,
       a.LGB_NOTE                                        AS LGB_NOTE,
       a.PIC                                             AS TECHNICAL_PIC,
       (SELECT MIN(f.NEXT_FOLLOWUP)
          FROM POOLDATA.GCNM_PROGRESS_CLAIM f
         WHERE f.PNCCASEID = a.NOKLAIM)                  AS EARLIEST_FOLLOW_UP,
       a.TGL_PROSES                                      AS PROCESS_DATE,
       a.PROD_KE                                         AS PROD_KE
  FROM POOLDATA.PEGA_DASHBOARDPNC a
 WHERE a.STSKLAIM NOT IN ('1', '2', '3')
   AND (:1 IS NULL
        OR UPPER(a.NOKLAIM) LIKE '%' || UPPER(:1) || '%' ESCAPE '\'
        OR UPPER(a.NOPOLIS) LIKE '%' || UPPER(:1) || '%' ESCAPE '\'
        OR UPPER(a.PIC) LIKE '%' || UPPER(:1) || '%' ESCAPE '\')
 ORDER BY a.TGL_PROSES ASC, a.NOKLAIM ASC
OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY


-- name: claims_outstanding_count
--
-- Banyaknya baris yang cocok SEBELUM dipotong paginasi —
-- RDB List/GcnmCountProgressClaim_SQL-SQL.xml
--
-- Penyaringnya WAJIB sama persis dengan claims_outstanding_list. Bila keduanya berbeda,
-- layar menampilkan jumlah halaman yang tidak sesuai isinya — dan halaman terakhir menjadi
-- kosong tanpa sebab yang terlihat.
--
-- Bind: :1 kata kunci (boleh NULL)
SELECT COUNT(1) AS TOTAL_ROWS
  FROM POOLDATA.PEGA_DASHBOARDPNC a
 WHERE a.STSKLAIM NOT IN ('1', '2', '3')
   AND (:1 IS NULL
        OR UPPER(a.NOKLAIM) LIKE '%' || UPPER(:1) || '%' ESCAPE '\'
        OR UPPER(a.NOPOLIS) LIKE '%' || UPPER(:1) || '%' ESCAPE '\'
        OR UPPER(a.PIC) LIKE '%' || UPPER(:1) || '%' ESCAPE '\')


-- name: claims_next_fu_list
--
-- Region Next Follow Up — Activity/GetNextFUdata_act-Act.xml
--
-- Sama dengan Outstanding, ditambah satu saringan: tindak lanjut TERAKHIR yang dicatat PIC
-- klaim itu sendiri sudah jatuh tempo hari ini atau sebelumnya.
--
-- Perhatikan `b.USER_INPUT = a.PIC` — saringannya hanya melihat catatan yang dibuat PIC
-- klaim, bukan catatan siapa pun. Klaim yang tindak lanjut terakhirnya dicatat orang lain
-- karena itu TIDAK muncul di sini, dan itu perilaku sistem lama yang dibawa apa adanya.
--
-- Bind: :1 kata kunci (boleh NULL) · :2 tanggal hari ini · :3 offset · :4 jumlah baris
SELECT a.NOKLAIM                                         AS CLAIM_NUMBER,
       a.NOPOLIS                                         AS POLICY_NUMBER,
       (SELECT MAX(c.QQNAME)
          FROM POOLDATA.T_CLAIM_PNC c
         WHERE c.CLAIMNO = a.NOKLAIM)                    AS INSURED_NAME,
       a.TGLKLAIM                                        AS REGISTER_DATE,
       a.DATEOFLOSS                                      AS LOSS_DATE,
       a.LGB_NOTE                                        AS LGB_NOTE,
       a.PIC                                             AS TECHNICAL_PIC,
       (SELECT MIN(f.NEXT_FOLLOWUP)
          FROM POOLDATA.GCNM_PROGRESS_CLAIM f
         WHERE f.PNCCASEID = a.NOKLAIM)                  AS EARLIEST_FOLLOW_UP,
       a.TGL_PROSES                                      AS PROCESS_DATE,
       a.PROD_KE                                         AS PROD_KE
  FROM POOLDATA.PEGA_DASHBOARDPNC a
 WHERE a.STSKLAIM NOT IN ('1', '2', '3')
   AND (:1 IS NULL
        OR UPPER(a.NOKLAIM) LIKE '%' || UPPER(:1) || '%' ESCAPE '\'
        OR UPPER(a.NOPOLIS) LIKE '%' || UPPER(:1) || '%' ESCAPE '\'
        OR UPPER(a.PIC) LIKE '%' || UPPER(:1) || '%' ESCAPE '\')
   AND (SELECT CAST(MAX(b.NEXT_FOLLOWUP) AS DATE)
          FROM POOLDATA.GCNM_PROGRESS_CLAIM b
         WHERE b.PNCCASEID = a.NOKLAIM
           AND b.USER_INPUT = a.PIC) <= :2
 ORDER BY a.TGL_PROSES ASC, a.NOKLAIM ASC
OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY


-- name: claims_next_fu_count
--
-- Penyaringnya WAJIB sama persis dengan claims_next_fu_list.
--
-- Bind: :1 kata kunci (boleh NULL) · :2 tanggal hari ini
SELECT COUNT(1) AS TOTAL_ROWS
  FROM POOLDATA.PEGA_DASHBOARDPNC a
 WHERE a.STSKLAIM NOT IN ('1', '2', '3')
   AND (:1 IS NULL
        OR UPPER(a.NOKLAIM) LIKE '%' || UPPER(:1) || '%' ESCAPE '\'
        OR UPPER(a.NOPOLIS) LIKE '%' || UPPER(:1) || '%' ESCAPE '\'
        OR UPPER(a.PIC) LIKE '%' || UPPER(:1) || '%' ESCAPE '\')
   AND (SELECT CAST(MAX(b.NEXT_FOLLOWUP) AS DATE)
          FROM POOLDATA.GCNM_PROGRESS_CLAIM b
         WHERE b.PNCCASEID = a.NOKLAIM
           AND b.USER_INPUT = a.PIC) <= :2


-- name: positions
--
-- PENGGANTI Database/GET_POSISI_PROGRESS_PNC.fnc, satu kueri untuk seluruh baris satu
-- halaman.
--
-- # Apa yang digantikannya
--
-- Fungsi lama menerima nomor klaim dan sebuah kategori, lalu memutar kursor atas setiap
-- posisi berjalan dan menggabungkan nilainya dengan koma. Ia dipanggil EMPAT KALI untuk
-- setiap baris grid — `POSISI`, `sts_prg1`, `sts_prg2`, `nextfu` — dan setiap panggilan
-- mengulang kursor yang sama. Pada satu halaman 15 baris itu 60 pemanggilan.
--
-- Di sini keempat nilainya datang dari satu baris hasil, penggabungan antarposisi terjadi
-- di Go, dan seluruh halaman selesai dalam satu perjalanan.
--
-- # Yang TIDAK dibawa: pemformatan tanggalnya
--
-- Fungsi lama mengembalikan tenggat sebagai teks lewat
-- `to_char(max(next_followup),'DD-MM-YYYY hh:mm:ss')`. Pada Oracle, `mm` di dalam bagian
-- jam berarti BULAN — menitnya seharusnya `mi` — sehingga jam yang selama ini tampil
-- berbunyi `jam:BULAN:detik`. Di sini ia dikembalikan sebagai DATE dan pemformatannya
-- terjadi di satu tempat di lapisan transport.
--
-- # Ketiga nilai per posisi dibiarkan sebagai subkueri skalar
--
-- Menjadikannya JOIN akan MELIPATGANDAKAN baris bila satu posisi punya beberapa catatan
-- progres, dan jumlah posisi sebuah klaim adalah hal yang ditampilkan. `MAX` di dalam
-- subkueri itu bawaan sistem lama, bukan tambahan.
--
-- Bind: :1..:N nomor klaim pada halaman yang sedang dibuka. Daftar bind-nya disusun
-- sebanyak baris halaman — lihat catatan `inList` di query.go; yang disusun hanyalah
-- PENANDA bind, tidak pernah nilainya.
SELECT q.CLAIMNO                                         AS CLAIM_NUMBER,
       q.POSISI                                          AS POSITION_NAME,
       (SELECT MAX(m1.STS_PROGRESS1)
          FROM POOLDATA.GCNM_PROGRESS_CLAIM g1
               INNER JOIN POOLDATA.GCNM_MST_PROGRESS_KLAIM m1
                       ON g1.STATUS_PROGRESS1 = m1.ID_PROGRESS
         WHERE g1.PNCCASEID = q.CLAIMNO
           AND g1.POSISIID = q.ID)                       AS PROGRESS_STATUS_1,
       (SELECT MAX(m2.STS_PROGRESS2)
          FROM POOLDATA.GCNM_PROGRESS_CLAIM g2
               INNER JOIN POOLDATA.GCNM_MST_PROGRESS m2
                       ON g2.STATUS_PROGRESS2 = m2.ID_MST
         WHERE g2.PNCCASEID = q.CLAIMNO
           AND g2.POSISIID = q.ID)                       AS PROGRESS_STATUS_2,
       (SELECT MAX(g3.NEXT_FOLLOWUP)
          FROM POOLDATA.GCNM_PROGRESS_CLAIM g3
         WHERE g3.PNCCASEID = q.CLAIMNO
           AND g3.POSISIID = q.ID)                       AS NEXT_FOLLOW_UP
  FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC q
 WHERE q.STATUSPOSISI = 'On Progress'
   AND q.CLAIMNO IN (%s)
 ORDER BY q.CLAIMNO ASC, q.ID ASC


-- name: pic_summary
--
-- Region Progress Klaim per PIC — RDB List/GetProgressPIC-SQL.xml
--
-- Satu baris berisi lima pencacah tentang beban dan ketepatan tindak lanjut seorang
-- petugas.
--
-- # Kenapa barisnya hanya milik pemanggil
--
-- Karena begitulah di sistem lama. Langkah pertama `GetProgressPerPIC` — bernama "Progress
-- Claim per User" — menyusun `and a.pic = '<pengguna yang login>'` dan langkah itu TIDAK
-- punya prakondisi sama sekali, sehingga selalu berjalan. Judulnya menyebut "per PIC",
-- tetapi isinya selalu satu petugas: yang sedang membuka layar.
--
-- # Lini bisnis WAJIB, dan itu bukan pilihan kami
--
-- `MST_USER_TEKNIK.TYPE_BUSINESS` dicocokkan dengan lini bisnis yang dipilih. Tanpa lini
-- bisnis, tidak ada satu pun petugas yang cocok dan hasilnya selalu kosong.
--
-- Di sistem lama lini bisnis itu TIDAK dipilih pengguna melainkan dibaca dari
-- `OperatorID.pyPosition` — jabatan pada catatan operator Pega, yang rupanya diisi
-- "NONMBU", "TRAVEL", "BONDING", atau "PA". Nilai itu tidak tersedia di sistem baru:
-- HCC/HCQ mengembalikan jabatan sebenarnya (`Placement.PositionName`), bukan lini bisnis.
-- Sampai pemetaan pengguna ke lini bisnis menjadi master data (`F-4`), lini bisnis dipilih
-- pengguna lewat dropdown. Keterbatasan itu dinyatakan di layar.
--
-- # Pencacah yang menyatakan ketepatan waktu
--
-- ON_TIME_COUNT dan LATE_COUNT membandingkan tenggat sebuah catatan dengan tanggal input
-- catatan BERIKUTNYA (`id_update + 1`). Bila catatan berikutnya belum ada, pembandingnya
-- tenggat itu sendiri — sehingga catatan terakhir selalu terhitung tepat waktu. Perilaku
-- itu dibawa apa adanya.
--
-- Bind: :1 lini bisnis · :2 login pemanggil · :3 tanggal hari ini ·
--       :4 tanggal awal (boleh NULL) · :5 tanggal akhir (boleh NULL)
SELECT a.PIC                                             AS PIC,
       COUNT(a.NOKLAIM)                                  AS CLAIM_COUNT,
       (SELECT COUNT(c.ID_UPDATE)
          FROM POOLDATA.GCNM_PROGRESS_CLAIM c
               INNER JOIN POOLDATA.PEGA_DASHBOARDPNC b
                       ON c.PNCCASEID = b.NOKLAIM
                      AND b.PIC = c.USER_INPUT
         WHERE b.PIC = a.PIC
           AND UPPER(c.KETERANGAN) NOT LIKE 'AUTO%' ESCAPE '\'
           AND (:4 IS NULL OR CAST(b.TGLKLAIM AS DATE) >= :4)
           AND (:5 IS NULL OR CAST(b.TGLKLAIM AS DATE) <= :5)) AS UPDATE_COUNT,
       (SELECT COUNT(c.NEXT_FOLLOWUP)
          FROM POOLDATA.GCNM_PROGRESS_CLAIM c
               INNER JOIN POOLDATA.PEGA_DASHBOARDPNC b
                       ON c.PNCCASEID = b.NOKLAIM
                      AND b.PIC = c.USER_INPUT
         WHERE b.PIC = a.PIC
           AND b.STSKLAIM NOT IN ('1', '2', '3')
           AND UPPER(c.KETERANGAN) NOT LIKE 'AUTO%' ESCAPE '\'
           AND CAST(c.NEXT_FOLLOWUP AS DATE) = :3
           AND c.ID_UPDATE = (SELECT MAX(r.ID_UPDATE)
                                FROM POOLDATA.GCNM_PROGRESS_CLAIM r
                               WHERE r.PNCCASEID = c.PNCCASEID)
           AND (:4 IS NULL OR CAST(b.TGLKLAIM AS DATE) >= :4)
           AND (:5 IS NULL OR CAST(b.TGLKLAIM AS DATE) <= :5)) AS DUE_TODAY_COUNT,
       (SELECT COUNT(t.NEXT_FOLLOWUP)
          FROM POOLDATA.GCNM_PROGRESS_CLAIM t
               INNER JOIN POOLDATA.PEGA_DASHBOARDPNC b
                       ON t.PNCCASEID = b.NOKLAIM
                      AND b.PIC = t.USER_INPUT
         WHERE b.PIC = a.PIC
           AND UPPER(t.KETERANGAN) NOT LIKE 'AUTO%' ESCAPE '\'
           AND CAST(t.NEXT_FOLLOWUP AS DATE) + 1 >= COALESCE(
                 (SELECT COALESCE(CAST(rt.TGL_INPUT AS DATE), :3)
                    FROM POOLDATA.GCNM_PROGRESS_CLAIM rt
                   WHERE rt.PNCCASEID = t.PNCCASEID
                     AND rt.ID_UPDATE = t.ID_UPDATE + 1),
                 CAST(t.NEXT_FOLLOWUP AS DATE))
           AND (:4 IS NULL OR CAST(b.TGLKLAIM AS DATE) >= :4)
           AND (:5 IS NULL OR CAST(b.TGLKLAIM AS DATE) <= :5)) AS ON_TIME_COUNT,
       (SELECT COUNT(t.NEXT_FOLLOWUP)
          FROM POOLDATA.GCNM_PROGRESS_CLAIM t
               INNER JOIN POOLDATA.PEGA_DASHBOARDPNC b
                       ON t.PNCCASEID = b.NOKLAIM
                      AND b.PIC = t.USER_INPUT
         WHERE b.PIC = a.PIC
           AND UPPER(t.KETERANGAN) NOT LIKE 'AUTO%' ESCAPE '\'
           AND CAST(t.NEXT_FOLLOWUP AS DATE) + 1 < (SELECT COALESCE(CAST(rt.TGL_INPUT AS DATE), :3)
                                                      FROM POOLDATA.GCNM_PROGRESS_CLAIM rt
                                                     WHERE rt.PNCCASEID = t.PNCCASEID
                                                       AND rt.ID_UPDATE = t.ID_UPDATE + 1)
           AND (:4 IS NULL OR CAST(b.TGLKLAIM AS DATE) >= :4)
           AND (:5 IS NULL OR CAST(b.TGLKLAIM AS DATE) <= :5)) AS LATE_COUNT
  FROM POOLDATA.PEGA_DASHBOARDPNC a
 WHERE a.PIC IS NOT NULL
   AND a.PIC <> 'ASNET'
   AND a.STSKLAIM NOT IN ('1', '2', '3')
   AND a.PIC = :2
   AND a.PIC IN (SELECT u.OPERATOR_ID
                   FROM POOLDATA.MST_USER_TEKNIK u
                  WHERE u.TYPE_BUSINESS = :1
                    AND u.STS_AKTIF = '1')
   -- Keempat cabang lini bisnis, apa adanya dari Activity/GetProgressPerPIC-Act.xml
   -- langkah 2–5. Ia ditulis sebagai satu predikat ber-bind tunggal, bukan empat potongan
   -- SQL yang dirangkai di aplikasi seperti di sistem lama.
   --
   -- Perhatikan NONMBU: ia bukan "selain PA, Travel, dan Bonding", melainkan tiga Group
   -- Panel tertentu DIKURANGI kelompok Bonding. Klaim ber-Group Panel di luar keempat
   -- kelompok itu tidak muncul pada pilihan mana pun.
   AND ((:1 = 'NONMBU'
         AND a.GROUP_PANEL IN ('003', '004', '006')
         AND a.GROUPBISNISID NOT IN ('09', '11', '16', '25'))
     OR (:1 = 'TRAVEL' AND a.GROUP_PANEL IN ('005'))
     OR (:1 = 'BONDING' AND a.GROUPBISNISID IN ('09', '11', '16', '25'))
     OR (:1 = 'PA' AND a.GROUP_PANEL IN ('002')))
   AND (:4 IS NULL OR CAST(a.TGLKLAIM AS DATE) >= :4)
   AND (:5 IS NULL OR CAST(a.TGLKLAIM AS DATE) <= :5)
 GROUP BY a.PIC
 ORDER BY a.PIC ASC


-- name: check_table
--
-- Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun: yang diperiksa adalah
-- hak baca dan keberadaan tabel inti modul ini.
SELECT COUNT(1) AS TABLE_READABLE
  FROM POOLDATA.PEGA_DASHBOARDPNC
 WHERE 1 = 0
