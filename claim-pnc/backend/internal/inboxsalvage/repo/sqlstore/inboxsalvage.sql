-- Kueri modul Inbox Salvage (`MENU_ID 71`, pengganti `Harness/InboxSalvage`).
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- ============================================================================
-- TABEL YANG DIBACA, DAN SIAPA PEMILIKNYA
-- ============================================================================
--
--   POOLDATA.T_CLAIM_PNC         dimiliki Pega   — hanya dibaca
--   POOLDATA.T_CLAIM_OBJECTLIST  dimiliki Pega   — hanya dibaca
--   POOLDATA.T_CLAIM_ADJUSTMENT  dimiliki Pega   — hanya dibaca
--   POOLDATA.PNC_SALVAGE         dimiliki MODUL INI — dibaca dan DITULIS
--   POOLDATA.DETAIL_PNC_SALVAGE  dimiliki MODUL INI — dibaca dan DITULIS
--
-- Kedua tabel terakhir adalah satu-satunya yang ditulis berkas ini, dan `P-1` terpenuhi:
-- tidak ada layar Pega lain yang menulisinya — seluruh penulisnya adalah layar Inbox
-- Salvage, yang modul ini gantikan.
--
-- ============================================================================
-- TIGA KELUARGA KUERI UNTUK TIGA BELAS DAFTAR
-- ============================================================================
--
--   list_claim         Salvage Outstanding                         <- GcnmSalvageData_OS_SQL
--   list_claim_object  Ekonomis · TBA · Tidak Ekonomis ·
--                      Tidak Ada Salvage                           <- GcnmSalvageData_ekonomisdanTba
--   list_claim_buyback Salvage Buyback                             <- idem, penyaring berbeda
--   list_salvage       ketujuh daftar berbasis PNC_SALVAGE         <- GcnmSalvageData_CloseOs_SQL
--
-- Keempatnya memakai penyaring lini bisnis yang SAMA, dan pasangan itu muncul kata demi
-- kata di sepuluh rule Pega berbeda:
--
--   GROUPPANEL IN ('003','004','006','009')
--   BUSINESSCODE NOT IN ('10145','10168')
--
-- ============================================================================
-- PEMETAAN KOLOM — kolom sebenarnya -> alias Pega -> alias di sini
-- ============================================================================
--
-- Alias Pega TIDAK dibawa (`D-19`). Tidak satu pun menyatakan isinya, dan dua di antaranya
-- menyatakan hal yang SALAH — menyalinnya akan menampilkan kolom yang keliru tanpa satu pun
-- galat:
--
--   kolom sebenarnya   alias Pega     alias di sini      judul kolom
--   ------------------ -------------- ------------------ -------------------------
--   NOKLAIM            "CaseID"       CLAIM_NO           No Klaim
--   IDSALVAGE          "ClaimNo"  (!) SALVAGE_ID         (tidak digambar)
--   TGLINPUT           "DateOfLoss"(!) INPUT_DATE        Tanggal Input
--   PIC                "UserTeknis"   PIC                PIC
--   JENISSALVAGE       "NewEmail"     SALVAGE_TYPE       Jenis Salvage
--   LOKASISALVAGE      "Location"     SALVAGE_LOCATION   Lokasi
--   QUANTITYSALVAGE    "KomiteCount"  QUANTITY           (tidak digambar)
--   ESTIMASINILAI      "District"     ESTIMATE_VALUE     Nilai Pengajuan PIC
--   EMAIL              "Email"        EMAIL              Email
--   REMARK             "AlasanKlaim"  REMARK             Keterangan PIC
--   NOAKSEPTASI        "NIK"          ACCEPTANCE_NO      (tidak digambar)
--   STSTRANSFER        "Password"     TRANSFER_STATUS    (tidak digambar)
--   CASE NILAIAKSEP    "TreatyName"   AUCTION_STATUS     Status Lelang
--
-- Dua tanda (!) itu adalah tempat paling mudah salah di seluruh berkas ini:
--
--   "ClaimNo"    di kueri ini berarti ID SALVAGE, bukan nomor klaim
--   "DateOfLoss" di kueri ini berarti TANGGAL INPUT SALVAGE, bukan tanggal kejadian —
--                sementara alias yang SAMA pada list_claim memang berarti tanggal kejadian
--
-- ============================================================================
-- PAGINASI
-- ============================================================================
--
-- `OFFSET … FETCH NEXT` di basis data, dan jumlah seluruhnya lewat `COUNT(*) OVER ()`.
--
-- Sistem lama menempuhnya berbeda: ia menyisipkan `WHERE RN >= '…' AND RN <= '…'` ke dalam
-- teks SQL lewat pola `{ASIS:…}`, dengan batas atas-bawah dirangkai sebagai TEKS. Angkanya
-- lalu dibandingkan dengan kolom `ROWNUM` bertipe angka, sehingga Oracle mengonversinya
-- diam-diam. Pola itu tidak dibawa (`11-SECURITY.md` §5).
--
-- Jumlah seluruhnya pun berbeda sumbernya: sistem lama memakai kueri penghitung TERPISAH
-- (`GetCountSalvage_OS`, `GetCountSalvage_All`, …), dan salah satunya menghitung populasi
-- yang berbeda dari daftarnya. Lihat catatan pada count_outstanding_legacy.

-- name: list_claim
-- Daftar Salvage Outstanding — klaim yang belum ditandai punya salvage.
--
-- Bind: :1 status kerja dikecualikan pertama · :2 kedua · :3 offset · :4 jumlah baris
--
-- Penyaring `STSSALVAGE IS NULL` ditulis TETAP di sini, bukan diikat, karena ia satu-satunya
-- bentuknya: hanya daftar ini yang menyaring kekosongan, dan `IS NULL` tidak dapat
-- dinyatakan lewat bind.
SELECT c.CLAIMNO                        AS CLAIM_NO,
       c.PICTEKNIK                      AS PIC,
       c.BUSINESSNAME                   AS BUSINESS_NAME,
       c.DATEOFLOSS                     AS LOSS_DATE,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE c.GROUPPANEL IN ('003', '004', '006', '009')
   AND c.BUSINESSCODE NOT IN ('10145', '10168')
   AND c.STATUSWORK NOT IN (:1, :2)
   AND c.STSSALVAGE IS NULL
 ORDER BY c.CLAIMNO
OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY

-- name: list_claim_search
-- Sama seperti di atas, disaring nomor klaim yang MENGANDUNG kata kunci.
--
-- Bind: :1 status kerja dikecualikan pertama · :2 kedua · :3 kata kunci · :4 offset
--       :5 jumlah baris
--
-- Kata kuncinya DIIKAT, bukan dirangkai. Sistem lama menyisipkannya ke dalam teks SQL lewat
-- `{ASIS:TempTes.Country}` — apa yang diketik pengguna, langsung ke dalam kueri.
--
-- `UPPER` di kedua sisi: kueri lama membandingkan apa adanya sehingga pencarian di sana peka
-- huruf besar-kecil. Nomor klaim selalu huruf kapital di basis data, sehingga satu-satunya
-- selisih yang mungkin adalah pengguna yang mengetik huruf kecil — yang di Pega tidak
-- menemukan apa-apa, dan di sini menemukan barisnya.
SELECT c.CLAIMNO                        AS CLAIM_NO,
       c.PICTEKNIK                      AS PIC,
       c.BUSINESSNAME                   AS BUSINESS_NAME,
       c.DATEOFLOSS                     AS LOSS_DATE,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE c.GROUPPANEL IN ('003', '004', '006', '009')
   AND c.BUSINESSCODE NOT IN ('10145', '10168')
   AND c.STATUSWORK NOT IN (:1, :2)
   AND c.STSSALVAGE IS NULL
   AND UPPER(c.CLAIMNO) LIKE UPPER(:3) ESCAPE '\'
 ORDER BY c.CLAIMNO
OFFSET :4 ROWS FETCH NEXT :5 ROWS ONLY

-- name: list_claim_object
-- Daftar Ekonomis · TBA · Tidak Ekonomis · Tidak Ada Salvage.
--
-- Bind: :1 nilai STSSALVAGE · :2 offset · :3 jumlah baris
--
-- Nama objek diambil dari objek PERTAMA klaim. Kueri lama menulisnya
-- `fetch next 1 row only` TANPA `ORDER BY`, sehingga objek mana yang terpilih tidak
-- ditentukan — baris yang sama dapat menampilkan objek berbeda pada dua pembukaan. Urutan
-- yang pasti DITAMBAHKAN di sini; lihat PlannedDifferences.
--
-- Daftar ini TIDAK menyaring status kerja, sehingga ia memuat klaim yang sudah selesai. Itu
-- perilaku sistem lama apa adanya — hanya list_claim yang menyaringnya.
SELECT c.CLAIMNO                        AS CLAIM_NO,
       c.PICTEKNIK                      AS PIC,
       c.BUSINESSNAME                   AS BUSINESS_NAME,
       (SELECT o.OBJECTNAME
          FROM POOLDATA.T_CLAIM_OBJECTLIST o
         WHERE o.CLAIMID = c.CLAIMID
         ORDER BY o.OBJECTNAME
         FETCH NEXT 1 ROW ONLY)         AS OBJECT_NAME,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE c.GROUPPANEL IN ('003', '004', '006', '009')
   AND c.BUSINESSCODE NOT IN ('10145', '10168')
   AND c.STSSALVAGE = :1
 ORDER BY c.CLAIMNO
OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY

-- name: list_claim_object_search
-- Sama seperti di atas, disaring nomor klaim yang mengandung kata kunci.
--
-- Bind: :1 nilai STSSALVAGE · :2 kata kunci · :3 offset · :4 jumlah baris
SELECT c.CLAIMNO                        AS CLAIM_NO,
       c.PICTEKNIK                      AS PIC,
       c.BUSINESSNAME                   AS BUSINESS_NAME,
       (SELECT o.OBJECTNAME
          FROM POOLDATA.T_CLAIM_OBJECTLIST o
         WHERE o.CLAIMID = c.CLAIMID
         ORDER BY o.OBJECTNAME
         FETCH NEXT 1 ROW ONLY)         AS OBJECT_NAME,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE c.GROUPPANEL IN ('003', '004', '006', '009')
   AND c.BUSINESSCODE NOT IN ('10145', '10168')
   AND c.STSSALVAGE = :1
   AND UPPER(c.CLAIMNO) LIKE UPPER(:2) ESCAPE '\'
 ORDER BY c.CLAIMNO
OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY

-- name: list_claim_buyback
-- Daftar Salvage Buyback — klaim yang punya baris adjustment ber-nilai salvage terisi.
--
-- Bind: :1 offset · :2 jumlah baris
--
-- Penyaringnya ditulis `EXISTS`, bukan `IN (SELECT DISTINCT …)` seperti kueri lama.
-- Keduanya menghasilkan baris yang sama; `EXISTS` berhenti pada baris pertama yang cocok
-- alih-alih menyusun seluruh daftar nomor klaim lebih dulu.
SELECT c.CLAIMNO                        AS CLAIM_NO,
       c.PICTEKNIK                      AS PIC,
       c.BUSINESSNAME                   AS BUSINESS_NAME,
       (SELECT o.OBJECTNAME
          FROM POOLDATA.T_CLAIM_OBJECTLIST o
         WHERE o.CLAIMID = c.CLAIMID
         ORDER BY o.OBJECTNAME
         FETCH NEXT 1 ROW ONLY)         AS OBJECT_NAME,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE c.GROUPPANEL IN ('003', '004', '006', '009')
   AND c.BUSINESSCODE NOT IN ('10145', '10168')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_CLAIM_ADJUSTMENT a
                WHERE a.CLAIMID = c.CLAIMID
                  AND a.NILAI_SALVAGE_A IS NOT NULL)
 ORDER BY c.CLAIMNO
OFFSET :1 ROWS FETCH NEXT :2 ROWS ONLY

-- name: list_claim_buyback_search
-- Sama seperti di atas, disaring nomor klaim yang mengandung kata kunci.
--
-- Bind: :1 kata kunci · :2 offset · :3 jumlah baris
SELECT c.CLAIMNO                        AS CLAIM_NO,
       c.PICTEKNIK                      AS PIC,
       c.BUSINESSNAME                   AS BUSINESS_NAME,
       (SELECT o.OBJECTNAME
          FROM POOLDATA.T_CLAIM_OBJECTLIST o
         WHERE o.CLAIMID = c.CLAIMID
         ORDER BY o.OBJECTNAME
         FETCH NEXT 1 ROW ONLY)         AS OBJECT_NAME,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE c.GROUPPANEL IN ('003', '004', '006', '009')
   AND c.BUSINESSCODE NOT IN ('10145', '10168')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_CLAIM_ADJUSTMENT a
                WHERE a.CLAIMID = c.CLAIMID
                  AND a.NILAI_SALVAGE_A IS NOT NULL)
   AND UPPER(c.CLAIMNO) LIKE UPPER(:1) ESCAPE '\'
 ORDER BY c.CLAIMNO
OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY

-- name: list_salvage
-- Ketujuh daftar berbasis PNC_SALVAGE, dibedakan penyaringnya sendiri.
--
-- Bind: :1 awalan kunci Pega · :2 STSTRANSFER (`%` berarti seluruhnya)
--       :3 PIC pemilik (`%` berarti seluruhnya)
--       :4 kata kunci nomor klaim (`%` berarti seluruhnya)
--       :5 kata kunci PIC (`%` berarti seluruhnya) · :6 offset · :7 jumlah baris
--
-- Nomor bind mengikuti urutan MUNCULNYA di dalam teks, supaya susunan argumen di Go dapat
-- diperiksa dengan membaca kueri dari atas ke bawah.
--
-- ============================================================================
-- KENAPA PENYARINGNYA MEMAKAI `LIKE`, BUKAN `=`
-- ============================================================================
--
-- Supaya SATU kueri melayani ketujuh daftar beserta keempat ragam pencariannya, tanpa
-- merangkai teks SQL sama sekali. Nilai `%` berarti "jangan saring kolom ini".
--
-- Kedua bind pencarian digabung dengan **ATAU**, bukan DAN. Itu bentuk penyaring tab
-- Checker di sistem lama — `and (a.noklaim = '…' or a.pic = '…')` — dan satu kotak di sana
-- memang mencari dua kolom sekaligus. Tab yang mencari nomor klaim saja mengirim pola yang
-- tidak pernah cocok pada bind PIC, sehingga cabang keduanya mati.
--
-- Karakter `%` dan `_` yang DIKETIK pengguna dilepaskan di sisi Go, dan `ESCAPE '\'`
-- memberitahu Oracle penanda pelolosannya. Tanpa itu, pencarian "cocok persis" pada tab
-- Checker akan meloloskan `%` sebagai wildcard — padahal kueri lama memakai `=`, yang
-- memperlakukannya sebagai huruf biasa.
--
-- ============================================================================
-- KENAPA `TO_CHAR` DI SEKITAR STSTRANSFER
-- ============================================================================
--
-- Karena `POOLDATA.PNC_SALVAGE.STSTRANSFER` bertipe **NUMBER**, dan `COALESCE` menuntut
-- seluruh argumennya bertipe sama. Mencampurnya dengan sentinel `'~'` yang bertipe CHAR
-- menghasilkan:
--
--   ORA-00932: inconsistent datatypes: expected NUMBER got CHAR
--
-- Galat itu nyata, bukan teoretis: ia menjatuhkan KETUJUH daftar keluarga C sekaligus pada
-- 2026-09-25, sementara pencacah tetap berjalan — karena pencacah memakai `IN (:1, :2)`,
-- yang justru mengonversi ke arah sebaliknya tanpa keberatan.
--
-- `TO_CHAR` dipakai, bukan mengganti sentinel menjadi angka, karena kolom yang sama
-- dibandingkan dengan literal TEKS di seluruh rule Pega (`and a.STSTRANSFER ='3'`). Nilai
-- penyaring di sisi Go karena itu tetap teks, dan `TO_CHAR` membuat kedua sisi bertemu di
-- tipe yang sama — apa pun tipe kolomnya kelak, sebab `TO_CHAR` menerima masukan teks pula.
--
-- Pelajaran yang berlaku di luar kueri ini: tipe kolom `PNC_SALVAGE` TIDAK diketahui dari
-- export — DDL-nya belum pernah diterima (`R-08`), dan rule Pega membandingkan kolom angka
-- dengan literal teks di mana-mana sehingga tipenya tidak terbaca dari sana. Setiap kolom
-- yang dipakai di dalam fungsi yang peka tipe harus dibuat aman lebih dulu.
--
-- Yang harus disadari: `LIKE '%'` TIDAK cocok dengan `NULL`. Itu disengaja dan benar di
-- sini — baris `PNC_SALVAGE` yang `STSTRANSFER`-nya kosong tidak pernah muncul di daftar
-- mana pun di Pega, karena setiap penyaringnya membandingkan dengan nilai. Satu-satunya
-- daftar yang tidak menyaring sama sekali adalah Histori Salvage, dan di sana kueri lama
-- pun menghasilkan hal yang sama: gabungan ke `T_CLAIM_PNC` yang membuang baris tanpa
-- pasangan. `COALESCE` dipakai supaya baris ber-PIC kosong tetap terhitung pada daftar yang
-- tidak menyaring PIC.
--
-- ============================================================================
-- GABUNGAN KE T_CLAIM_PNC, DAN KUNCI PEGA YANG BOCOR
-- ============================================================================
--
--   b.CLAIMID = 'ASM-FW-GCNMFW-WORK ' || a.NOKLAIM
--
-- Nama kelas internal Pega tertanam di dalam kunci data bisnis — utang teknis §4.1 yang
-- `D-22` dan `D-71` hapus untuk klaim baru. Gabungan ini dibawa apa adanya karena data
-- lamanya memang begitu; awalannya diikat (`:7`) alih-alih ditulis di dalam SQL, supaya ia
-- terbaca sebagai nilai yang kelak berubah, bukan sebagai bagian kuerinya.
--
-- Gabungan ini pula yang MEMBUANG baris salvage yang klaimnya tidak ada. Itu perilaku
-- sistem lama apa adanya.
--
-- Awalannya diikat sebagai :1 dan tampil pertama di dalam teks kueri.
--
-- ============================================================================
-- AGREGAT DETAIL — SATU kueri, bukan satu per baris
-- ============================================================================
--
-- `NILAI_REQUEST` dan `NOTE_REQUEST` diambil lewat sub-kueri berkorelasi, sekali untuk
-- seluruh halaman. Sistem lama menjalankan `PNCSalvageGetChekerDataKlaimAllData` sekali
-- untuk SETIAP baris — dua puluh satu kueri untuk satu halaman berisi dua puluh baris.
-- Angkanya sama persis; yang berubah hanya jumlah perjalanan ke basis data.
--
-- Penyaring `statusterjual is null or statusterjual = '3'` dibawa apa adanya dari kueri itu.
SELECT a.IDSALVAGE                      AS SALVAGE_ID,
       a.NOKLAIM                        AS CLAIM_NO,
       a.TGLINPUT                       AS INPUT_DATE,
       a.PIC                            AS PIC,
       a.JENISSALVAGE                   AS SALVAGE_TYPE,
       a.LOKASISALVAGE                  AS SALVAGE_LOCATION,
       a.QUANTITYSALVAGE                AS QUANTITY,
       a.ESTIMASINILAI                  AS ESTIMATE_VALUE,
       a.EMAIL                          AS EMAIL,
       a.REMARK                         AS REMARK,
       a.NOAKSEPTASI                    AS ACCEPTANCE_NO,
       a.STSTRANSFER                    AS TRANSFER_STATUS,
       a.NILAIAKSEP                     AS ACCEPTED_VALUE,
       (SELECT MAX(d.NILAI_REQUEST)
          FROM POOLDATA.DETAIL_PNC_SALVAGE d
         WHERE d.NOKLAIM = a.NOKLAIM
           AND d.IDSALVAGE = a.IDSALVAGE
           AND (d.STATUSTERJUAL IS NULL OR d.STATUSTERJUAL = '3'))
                                        AS REQUEST_VALUE,
       (SELECT MAX(d.NOTE_REQUEST)
          FROM POOLDATA.DETAIL_PNC_SALVAGE d
         WHERE d.NOKLAIM = a.NOKLAIM
           AND d.IDSALVAGE = a.IDSALVAGE
           AND (d.STATUSTERJUAL IS NULL OR d.STATUSTERJUAL = '3'))
                                        AS REQUEST_NOTE,
       COUNT(*) OVER ()                 AS TOTAL_ROWS
  FROM POOLDATA.PNC_SALVAGE a
       INNER JOIN POOLDATA.T_CLAIM_PNC b
               ON b.CLAIMID = :1 || a.NOKLAIM
 WHERE COALESCE(TO_CHAR(a.STSTRANSFER), '~') LIKE :2 ESCAPE '\'
   AND COALESCE(UPPER(a.PIC), '~') LIKE UPPER(:3) ESCAPE '\'
   AND (   UPPER(a.NOKLAIM) LIKE UPPER(:4) ESCAPE '\'
        OR COALESCE(UPPER(a.PIC), '~') LIKE UPPER(:5) ESCAPE '\')
 ORDER BY a.TGLINPUT DESC, a.IDSALVAGE
OFFSET :6 ROWS FETCH NEXT :7 ROWS ONLY

-- name: count_claim_status
-- Pencacah baris keluarga klaim: Ekonomis · TBA · Tidak Ekonomis · Tidak Ada Salvage,
-- dan baris "Outstanding" yang menghitung DUA nilai sekaligus.
--
-- Bind: :1 nilai STSSALVAGE pertama · :2 kedua
--
-- Kedua bind menerima nilai yang SAMA bila hanya satu yang dihitung. Menulisnya begitu
-- membuat satu kueri melayani kelima baris tanpa merangkai teks SQL.
--
-- Perhatikan baris "Outstanding" menghitung `STSSALVAGE` 3 atau 5 — BUKAN yang kosong,
-- yang justru ditampilkan daftarnya. Selisih itu ada di Pega dan direplikasi (`P-5`).
SELECT COUNT(*)                         AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE c.GROUPPANEL IN ('003', '004', '006', '009')
   AND c.BUSINESSCODE NOT IN ('10145', '10168')
   AND c.STSSALVAGE IN (:1, :2)

-- name: count_claim_buyback
-- Pencacah baris "Salvage Buyback".
SELECT COUNT(*)                         AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE c.GROUPPANEL IN ('003', '004', '006', '009')
   AND c.BUSINESSCODE NOT IN ('10145', '10168')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_CLAIM_ADJUSTMENT a
                WHERE a.CLAIMID = c.CLAIMID
                  AND a.NILAI_SALVAGE_A IS NOT NULL)

-- name: count_salvage_status
-- Pencacah baris keluarga salvage: Checker · Salvage Diterima · Salvage Ditolak ·
-- Rejected Checker · Balai Lelang · Request Balai Lelang · Histori Salvage.
--
-- Bind: :1 STSTRANSFER pertama · :2 kedua · :3 PIC (`%` berarti seluruhnya)
--
-- Kueri lama menghitungnya lewat LIMA rule berbeda, tiga di antaranya `SUM(CASE WHEN …)`
-- atas GABUNGAN SILANG `T_CLAIM_PNC` dan `PNC_SALVAGE` tanpa syarat gabungan di `WHERE` —
-- syaratnya hanya ada di dalam `CASE`. Hasilnya benar tetapi biayanya adalah perkalian
-- kedua tabel. Di sini tidak ada gabungan sama sekali, karena tidak satu pun dari kelima
-- hitungan itu membutuhkan kolom klaim.
--
-- Konsekuensinya HARUS disadari: baris salvage yang klaimnya tidak ada IKUT terhitung di
-- sini, sementara ia TIDAK tampil di daftar mana pun — daftar menggabungkannya ke klaim.
-- Perilaku itu sama dengan dua dari lima kueri lama (`CountSalvageDiterimaSalvage` dan
-- `CountSalvageDitolakSalvage`, yang juga tidak menggabung), dan berbeda dari tiga lainnya.
-- Menyeragamkannya ke arah sebaliknya akan mengubah angka yang dibaca orang setiap hari.
SELECT COUNT(*)                         AS TOTAL_ROWS
  FROM POOLDATA.PNC_SALVAGE a
 WHERE a.STSTRANSFER IN (:1, :2)
   AND COALESCE(UPPER(a.PIC), '~') LIKE UPPER(:3)

-- name: count_salvage_detail_unsold
-- Pencacah baris "Tidak Terjual".
--
-- Bind: :1 nilai STATUSTERJUAL
--
-- Baris ini tidak menuju daftar mana pun — tidak ada tab yang menerima kodenya di Pega.
-- Ia tetap dihitung karena activity pencacah menghitungnya tanpa syarat apa pun.
SELECT COUNT(*)                         AS TOTAL_ROWS
  FROM POOLDATA.DETAIL_PNC_SALVAGE d
 WHERE d.STATUSTERJUAL = :1

-- name: next_salvage_id
-- Nomor ID salvage berikutnya.
--
-- `MAX(TO_NUMBER(IDSALVAGE)) + 1`, sama dengan `Database/INSERT_SALVAGE.prc:20`.
--
-- Pola ini MEWARISI cacat yang sama dengan procedure-nya: dua penyimpanan yang berjalan
-- bersamaan dapat membaca nilai maksimum yang sama lalu menerbitkan ID yang sama. Ia tetap
-- dibawa karena mengganti ke sequence berarti mengubah cara nomor terbit, dan nomor yang
-- sudah ada di produksi tidak dapat dipindahkan ke sequence tanpa keputusan tersendiri.
-- Penyimpanan menutupnya sejauh yang dapat ditutup dari sisi ini — lihat catatan pada
-- Repo.Create.
SELECT COALESCE(MAX(TO_NUMBER(a.IDSALVAGE)), 0) + 1 AS NEXT_ID
  FROM POOLDATA.PNC_SALVAGE a

-- name: insert_salvage
-- Menyisipkan satu pengajuan salvage.
--
-- Bind: :1 NOKLAIM · :2 TGLINPUT · :3 JENISSALVAGE · :4 QUANTITYSALVAGE
--       :5 ESTIMASINILAI · :6 LOKASISALVAGE · :7 STSTRANSFER · :8 REMARK · :9 PIC
--       :10 IDSALVAGE · :11 IDOBJECT · :12 OBJECTNAME · :13 IDCOVERAGE · :14 COVERAGENAME
--       :15 CURRENCY · :16 NILAIAKSEP · :17 EMAIL · :18 NILAIPENAWARAN · :19 PICSURVEY
--       :20 EMAILSURVEY · :21 NOTELP · :22 ISJABODATABEK
--
-- Kolom yang SENGAJA tidak diisi, beserta alasannya:
--
--   TGLTRANSFERGA   diisi saat pengajuan benar-benar dikirim ke bagian umum, bukan saat
--                   dibuat. Procedure lama mengisinya dengan waktu penyimpanan, dan itu
--                   membuat setiap pengajuan tampak sudah ditransfer sejak lahir.
--   TGLAKSEPTASI    diisi saat akseptasi terbit.
--   NOAKSEPTASI     idem.
--   PEMENANGNAME    diisi setelah lelang selesai.
--   TANGGALLELANG   idem.
--   TANGGALTERIMA   idem.
--
-- Keenamnya ADA di parameter procedure lama, dan form Tambah tidak mengisi satu pun —
-- seluruhnya berasal dari isian yang hanya tergambar pada mode ubah.
INSERT INTO POOLDATA.PNC_SALVAGE
       (NOKLAIM, TGLINPUT, JENISSALVAGE, QUANTITYSALVAGE, ESTIMASINILAI,
        LOKASISALVAGE, STSTRANSFER, REMARK, PIC, IDSALVAGE,
        IDOBJECT, OBJECTNAME, IDCOVERAGE, COVERAGENAME, CURRENCY,
        NILAIAKSEP, EMAIL, NILAIPENAWARAN, PICSURVEY, EMAILSURVEY,
        NOTELP, ISJABODATABEK)
VALUES (:1, TO_DATE(:2, 'YYYY-MM-DD'), :3, TO_NUMBER(:4), TO_NUMBER(:5),
        :6, :7, :8, :9, :10,
        :11, :12, :13, :14, :15,
        TO_NUMBER(:16), :17, TO_NUMBER(:18), :19, :20,
        :21, :22)

-- name: update_salvage
-- Memperbarui satu pengajuan salvage.
--
-- Bind: sama seperti insert_salvage, dengan :10 IDSALVAGE sebagai kunci pencarinya.
--
-- ============================================================================
-- CACAT YANG DIPERBAIKI DI SINI — butir 12 daftar perbaikan `P-5` (`D-49` #9)
-- ============================================================================
--
-- `Database/INSERT_SALVAGE.prc:43` menulis pada cabang pembaruan:
--
--   UPDATE POOLDATA.PNC_SALVAGE SET …, IDSALVAGE = idsalvage, … WHERE IDSALVAGE = tID;
--
-- Variabel `idsalvage` hanya diisi di cabang PENYISIPAN, tidak pernah di cabang ini —
-- sehingga setiap pembaruan menimpa kunci barisnya sendiri dengan nilai kosong. Baris itu
-- lalu hilang dari seluruh daftar, karena setiap kueri mencarinya lewat `IDSALVAGE`.
--
-- Kueri ini TIDAK menulis `IDSALVAGE` sama sekali. Kunci barisnya tidak berubah, dan itu
-- memang yang dimaksud penulis procedure aslinya.
UPDATE POOLDATA.PNC_SALVAGE
   SET NOKLAIM = :1,
       TGLINPUT = TO_DATE(:2, 'YYYY-MM-DD'),
       JENISSALVAGE = :3,
       QUANTITYSALVAGE = TO_NUMBER(:4),
       ESTIMASINILAI = TO_NUMBER(:5),
       LOKASISALVAGE = :6,
       STSTRANSFER = :7,
       REMARK = :8,
       PIC = :9,
       IDOBJECT = :11,
       OBJECTNAME = :12,
       IDCOVERAGE = :13,
       COVERAGENAME = :14,
       CURRENCY = :15,
       NILAIAKSEP = TO_NUMBER(:16),
       EMAIL = :17,
       NILAIPENAWARAN = TO_NUMBER(:18),
       PICSURVEY = :19,
       EMAILSURVEY = :20,
       NOTELP = :21,
       ISJABODATABEK = :22
 WHERE IDSALVAGE = :10

-- name: count_salvage_detail_for
-- Jumlah baris detail yang sudah ada untuk satu pengajuan.
--
-- Bind: :1 IDSALVAGE · :2 NOKLAIM
--
-- Dipakai menyusun `IDDETAILSALVAGE`, yang berbentuk `noklaim/idsalvage/urutan` —
-- `Database/INSERT_SALVAGE_DETAILS.prc:23`.
SELECT COUNT(*)                         AS TOTAL_ROWS
  FROM POOLDATA.DETAIL_PNC_SALVAGE d
 WHERE d.IDSALVAGE = :1
   AND d.NOKLAIM = :2

-- name: insert_salvage_detail
-- Menyisipkan satu baris Detail Item Salvage.
--
-- Bind: :1 IDSALVAGE · :2 NOKLAIM · :3 IDDETAILSALVAGE · :4 NAMABARANG · :5 SATUAN
--       :6 NOTE_ITEM · :7 HARGAITEM · :8 IDOBJECT · :9 REMARK · :10 STATUSTERJUAL
--       :11 NOAKSEPTASI
--
-- Nilai `STATUSTERJUAL` dan `NOAKSEPTASI` diikat, bukan ditulis tetap seperti di procedure
-- lama (`'3'` dan `'-'`). Keduanya nilai bisnis, dan `D-15` melarangnya tertanam di dalam
-- kode.
--
-- `HARGAITEM` menerima JUMLAH item, bukan harga. Itu keadaan di procedure lama, tempat
-- parameter bernama `tTOTALHARGA` menerima isian berjudul "Jumlah Item / Qty". Nama
-- kolomnya tidak diubah — perubahan nama kolom menempuh `D-63`.
INSERT INTO POOLDATA.DETAIL_PNC_SALVAGE
       (INSERTDATE, IDSALVAGE, NOKLAIM, IDDETAILSALVAGE, NAMABARANG,
        SATUAN, NOTE_ITEM, HARGAITEM, IDOBJECT, REMARK,
        STATUSTERJUAL, NOAKSEPTASI)
VALUES (CURRENT_TIMESTAMP, :1, :2, :3, :4,
        :5, :6, TO_NUMBER(:7), :8, :9,
        :10, :11)

-- name: check_salvage_columns
-- Pemeriksaan kesiapan: memastikan kolom yang dibaca modul ini memang ada.
--
-- Dipakai perintah `check`, bukan jalur permintaan. Ia menjawab pertanyaan yang paling
-- sering muncul saat layar kosong: apakah kolomnya tidak ada, atau datanya yang tidak ada.
SELECT COUNT(*)                         AS TOTAL_ROWS
  FROM ALL_TAB_COLUMNS t
 WHERE t.OWNER = 'POOLDATA'
   AND t.TABLE_NAME = 'PNC_SALVAGE'
   AND t.COLUMN_NAME IN ('NOKLAIM', 'IDSALVAGE', 'TGLINPUT', 'PIC', 'JENISSALVAGE',
                         'LOKASISALVAGE', 'ESTIMASINILAI', 'STSTRANSFER', 'NILAIAKSEP')

-- name: detail_header
-- Kepala panel "Detail Salvage" — satu pengajuan.
--
-- Bind: :1 awalan kunci Pega · :2 IDSALVAGE
--
-- ============================================================================
-- CACAT KUERI LAMA YANG TIDAK DIBAWA: SATU ALIAS UNTUK DUA KOLOM
-- ============================================================================
--
-- `GcnmSetSalvageData_SQL` memakai alias `"AreaClaimId"` DUA KALI:
--
--   b.QUANTITYSALVAGE                              AS "AreaClaimId"
--   case when trunc(b.tglinput) < to_date(...)     AS "AreaClaimId"
--
-- Keduanya sah menurut Oracle, dan yang sampai ke layar hanya SATU. Akibatnya isian
-- "Quantity Salvage" pada panel detail menampilkan penanda 1/0 — bukan jumlah barang —
-- atau sebaliknya penanda itu yang hilang; mana yang menang bergantung pada urutan
-- pemetaan, bukan pada keputusan siapa pun.
--
-- Kelas cacat yang sama sudah tercatat pada modul Inbox Komunikasi Cabang (§48.4), tempat
-- satu alias `"UserName"` dipakai untuk nama pengirim DAN kode asal.
--
-- Di sini keduanya diberi alias TERSENDIRI, sehingga keduanya terbaca. Itu perbaikan atas
-- kode yang tidak pernah berjalan sebagaimana dimaksud penulisnya — bukan perubahan
-- perilaku yang disengaja pengguna.
--
-- Penanda tanggalnya sendiri dibawa apa adanya meski ARTINYA tidak diketahui: tidak ada
-- satu pun rule di export yang memakainya selain menggambarnya.
SELECT b.IDSALVAGE                      AS SALVAGE_ID,
       b.NOKLAIM                        AS CLAIM_NO,
       b.TGLINPUT                       AS INPUT_DATE,
       b.JENISSALVAGE                   AS SALVAGE_TYPE,
       b.QUANTITYSALVAGE                AS QUANTITY,
       b.ESTIMASINILAI                  AS ESTIMATE_VALUE,
       b.LOKASISALVAGE                  AS SALVAGE_LOCATION,
       b.TGLTRANSFERGA                  AS TRANSFER_GA_DATE,
       b.STSTRANSFER                    AS TRANSFER_STATUS,
       b.TGLAKSEPTASI                   AS ACCEPTANCE_DATE,
       b.NOAKSEPTASI                    AS ACCEPTANCE_NO,
       b.REMARK                         AS REMARK,
       b.CURRENCY                       AS CURRENCY,
       b.OBJECTNAME                     AS OBJECT_NAME,
       b.IDOBJECT                       AS OBJECT_ID,
       b.COVERAGENAME                   AS COVERAGE_NAME,
       b.IDCOVERAGE                     AS COVERAGE_ID,
       b.NILAIAKSEP                     AS ACCEPTED_VALUE,
       b.EMAIL                          AS EMAIL,
       b.NILAIPENAWARAN                 AS OFFER_VALUE,
       b.PEMENANGNAME                   AS WINNER_NAME,
       b.TANGGALLELANG                  AS AUCTION_DATE,
       b.PICSURVEY                      AS SURVEYOR_NAME,
       b.NOTELP                         AS SURVEYOR_PHONE,
       b.EMAILSURVEY                    AS SURVEYOR_EMAIL,
       b.ISJABODATABEK                  AS IN_JABODETABEK,
       CASE
           WHEN TRUNC(b.TGLINPUT) < DATE '2023-07-17' THEN '1'
           ELSE '0'
       END                              AS LEGACY_FLAG,
       (SELECT a.BUSINESSNAME
          FROM POOLDATA.T_CLAIM_PNC a
         WHERE a.CLAIMID = :1 || b.NOKLAIM)
                                        AS BUSINESS_NAME
  FROM POOLDATA.PNC_SALVAGE b
 WHERE b.IDSALVAGE = :2

-- name: detail_items
-- Daftar barang pada satu pengajuan.
--
-- Bind: :1 NOKLAIM · :2 IDSALVAGE
--
-- Barisnya DIKELOMPOKKAN menurut nama barang dan satuan, persis seperti kueri lama —
-- sehingga dua baris `DETAIL_PNC_SALVAGE` bernama sama menyatu menjadi satu baris
-- berjumlah dua.
--
-- Dua hal yang dibawa apa adanya dari `GetDetailSalvage`:
--
--   * `SUM(HARGAITEM)` — kolom yang namanya menyebut HARGA tetapi diisi JUMLAH ITEM oleh
--     procedure penyimpannya. Nama kolomnya tidak diubah; perubahan nama kolom menempuh
--     `D-63`.
--   * `COUNT(NAMABARANG)` sebagai jumlah baris yang menyatu. Ia dikirim sebagai ANGKA
--     tersendiri di sini, bukan dirangkai dengan satuannya di dalam SQL seperti kueri
--     lama — supaya jumlah dan satuan tetap dapat dibaca sendiri-sendiri.
--
-- `STATUSTERJUAL` dikirim sebagai KODE, bukan diterjemahkan di dalam SQL. Penerjemahannya
-- ada di inboxsalvage.SoldStatusOf, tempat pemetaannya dapat diuji — dan tempat
-- pertentangannya dengan kueri ekspor tercatat.
SELECT d.NAMABARANG                     AS ITEM_NAME,
       COUNT(d.NAMABARANG)              AS ITEM_COUNT,
       d.SATUAN                         AS UNIT,
       SUM(d.HARGAITEM)                 AS TOTAL_VALUE,
       d.STATUSTERJUAL                  AS SOLD_STATUS,
       d.PEMENANGSALVAGE                AS WINNER_NAME,
       d.NOAKSEPTASI                    AS ACCEPTANCE_NO,
       d.NILAIAKSEPTASI                 AS ACCEPTED_VALUE,
       d.REMARK                         AS REMARK
  FROM POOLDATA.DETAIL_PNC_SALVAGE d
 WHERE d.NOKLAIM = :1
   AND d.IDSALVAGE = :2
 GROUP BY d.NAMABARANG, d.SATUAN, d.STATUSTERJUAL, d.PEMENANGSALVAGE,
          d.NOAKSEPTASI, d.NILAIAKSEPTASI, d.REMARK
 ORDER BY d.NAMABARANG

-- name: latest_salvage_of_claim
-- ID pengajuan salvage TERAKHIR milik sebuah klaim.
--
-- Bind: :1 NOKLAIM
--
-- ============================================================================
-- KENAPA KUERI TERSENDIRI, BUKAN SUB-KUERI DI DALAM detail_header
-- ============================================================================
--
-- Layar lama menempuhnya sebagai sub-kueri:
--
--   Activity/SetDataDetailSalvage_act-Act.xml langkah 18 (param.tipe == 1)
--     tempQuery.NewEmail :=
--       "B.IDSALVAGE in (select max(IDSALVAGE)
--                          from POOLDATA.DETAIL_PNC_SALVAGE
--                         where noklaim = '" + Param.CaseeID + "')"
--
-- Aturannya dibawa apa adanya; yang tidak dibawa adalah perangkaiannya dari nilai
-- pengguna (`{ASIS:}`) — di sini ia parameter terikat.
--
-- Dipisah karena dua hal, dan keduanya menyangkut apa yang terbaca layar:
--
--   1. Sebagai sub-kueri, klaim yang BELUM pernah punya pengajuan menghasilkan NOL BARIS
--      pada kepala panel — tidak dapat dibedakan dari ID pengajuan yang salah. Dipisah,
--      keduanya menjadi dua keadaan yang berbeda: "klaim ini belum punya pengajuan" dan
--      "pengajuan tidak ditemukan".
--   2. Daftar kolom kepala panel tetap hidup di SATU tempat. Menyalinnya menjadi dua
--      kueri yang berbeda hanya pada klausa WHERE berarti menunggu keduanya bergeser.
--
-- Yang dikembalikan boleh NULL — itulah jawabannya untuk klaim tanpa pengajuan.
SELECT MAX(d.IDSALVAGE)                 AS SALVAGE_ID
  FROM POOLDATA.DETAIL_PNC_SALVAGE d
 WHERE d.NOKLAIM = :1

-- name: claim_header
-- Kepala panel untuk baris yang berupa KLAIM, bukan pengajuan.
--
-- Bind: :1 CLAIMNO
--
-- Kolomnya sama dengan yang dibaca `GcnmSalvageData_OS_SQL` — kueri yang dijalankan
-- `SetDataDetailSalvage_act` langkah 15 pada jalur `param.tipe == 1`.
--
-- ============================================================================
-- SATU SELISIH YANG DISENGAJA: PENYARING POPULASI TIDAK IKUT
-- ============================================================================
--
-- `GcnmSalvageData_OS_SQL` menyaring `STATUSWORK`, `GROUPPANEL`, dan `BUSINESSCODE`
-- sebelum menambahkan `and claimno = ...`. Penyaring itu menyusun POPULASI daftar
-- Salvage Outstanding — dan di Pega ia ikut terbawa ke detail semata karena jalur
-- `tipe == 1` hanya dipanggil dari grid itu.
--
-- Di sini panel detail dibuka dari KEENAM daftar berbasis klaim, sehingga membawa
-- penyaring populasi satu daftar akan membuat baris yang tampil di daftar LAIN menjawab
-- "tidak ditemukan" — padahal barisnya baru saja digambar di layar yang sama.
--
-- Barisnya sudah pasti ada; yang dibutuhkan panel adalah isinya, bukan pengujian ulang
-- keanggotaannya pada sebuah daftar.
SELECT c.CLAIMNO                        AS CLAIM_NO,
       c.PICTEKNIK                      AS PIC,
       c.BUSINESSNAME                   AS BUSINESS_NAME,
       c.DATEOFLOSS                     AS LOSS_DATE
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE c.CLAIMNO = :1

-- name: salvage_history_of_claim
-- Grid "Detail History Salvage" — SELURUH pengajuan salvage milik satu klaim.
--
-- Bind: :1 NOKLAIM
--
-- Menggantikan `RDB List/GetHistoriKlaimPNCSalvage_hist-SQL.xml`, yang merangkai nomor
-- klaim ke dalam teks SQL (`WHERE A.NOKLAIM = {tempQuery.RWID}`). Di sini ia parameter
-- terikat.
--
-- ============================================================================
-- KENAPA STSTRANSFER DIKIRIM SEBAGAI KODE, BUKAN DITERJEMAHKAN DI SINI
-- ============================================================================
--
-- Kueri lama menerjemahkannya di dalam `CASE WHEN` menjadi kalimat — "Sudah Aksep
-- Checker", "Salvage Waive", dan seterusnya. Pemetaan itu dipindahkan ke
-- inboxsalvage.HistoryPositionOf, dan alasannya bukan kerapian:
--
--   * `STSTRANSFER` sudah memikul TIGA arti yang berbeda di layar ini — penyaring daftar,
--     "Posisi Salvage" pada panel rincian, dan kalimat riwayat ini. Ketiganya TIDAK sama,
--     dan yang ketiga bahkan memecah kode `2` menjadi dua kalimat menurut terisi-tidaknya
--     nomor akseptasi.
--   * Pemetaan yang hidup di dalam SQL tidak dapat diuji tanpa basis data, sementara
--     pemetaan yang salah di sini menampilkan posisi yang keliru pada layar tanpa satu pun
--     galat.
--
-- Karena kode `2` bercabang menurut nomor akseptasi, kolom itu ikut dikirim sebagai
-- penanda terisi — bukan nilainya, karena yang dibutuhkan hanya ada-tidaknya.
SELECT a.TGLINPUT                       AS INPUT_DATE,
       a.NOKLAIM                        AS CLAIM_NO,
       a.PIC                            AS PIC,
       a.ESTIMASINILAI                  AS MINIMUM_VALUE,
       a.STSTRANSFER                    AS TRANSFER_STATUS,
       CASE
           WHEN a.NOAKSEPTASI IS NULL THEN '0'
           ELSE '1'
       END                              AS HAS_ACCEPTANCE_NO,
       a.IDSALVAGE                      AS SALVAGE_ID
  FROM POOLDATA.PNC_SALVAGE a
 WHERE a.NOKLAIM = :1
 ORDER BY a.TGLINPUT DESC, a.IDSALVAGE DESC
