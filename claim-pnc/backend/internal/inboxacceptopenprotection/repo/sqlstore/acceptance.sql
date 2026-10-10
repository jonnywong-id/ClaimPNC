-- Kueri modul Inbox Accept Open Protection: antrean akseptasi permintaan proteksi.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data.
--
-- ============================================================================
-- TIGA SYARAT YANG BUKAN PILIHAN PENGGUNA
-- ============================================================================
--
-- `Report Definition/InboxOpenProtection2_RD-RD.xml` menyaring
--
--     CaseID IS NOT NULL  AND  PolicyNo IS NOT NULL  AND  AcceptStatus IS NULL
--
-- Ketiganya DEFINISI layar ini, bukan penyaring yang dapat dimatikan pemanggil: yang
-- ditampilkan hanyalah permintaan yang sudah lengkap dan belum diputuskan.
--
-- Perhatikan bedanya dengan modul `inputreqprotection`, yang penyaringnya HANYA
-- `AcceptStatus IS NULL`. Perbedaan itulah yang membuat permintaan rancangan — yang belum
-- tertaut klaim — tidak pernah sampai ke meja petugas akseptasi.
--
-- ============================================================================
-- PEMISAHAN ANTREAN PREMI / NON PREMI
-- ============================================================================
--
--     InboxOpenProtection2_RD_collection   PROTECTION_TYPE_ID  = '2'   -> PREMI
--     InboxOpenProtection2_RD              PROTECTION_TYPE_ID <> '2'   -> NON PREMI
--
-- Kedua antrean memakai KUERI YANG SAMA dengan parameter pembeda, bukan dua kueri terpisah.
-- Dua kueri yang nyaris sama akan berbeda isinya cepat atau lambat, dan yang berbeda akan
-- menampilkan antrean yang salah tanpa satu pun gejala.
--
-- Penanda pertama dan ketiga membawa penentu antrean (1 = PREMI, 0 = NON PREMI); penanda
-- kedua dan keempat membawa kode pembandingnya. Keduanya dikirim DUA KALI karena muncul
-- dua kali — lihat bagian berikut.
--
-- ============================================================================
-- SETIAP KEMUNCULAN BIND BERNOMOR SENDIRI
-- ============================================================================
--
-- Driver mengikat argumen menurut urutan KEMUNCULAN penanda, bukan menurut nomornya.
-- Memakai penanda yang sama dua kali lalu mengirim satu argumen menghasilkan
-- **ORA-01008: not all variables bound**.
--
-- Pola pencarian pun dibentuk DI GO, bukan dirangkai di SQL: teks yang memuat tanda persen
-- atau garis bawah akan menjadi wildcard tanpa disengaja.
--
-- ============================================================================
-- KOLOM YANG DITULIS MODUL INI — HANYA TIGA
-- ============================================================================
--
--     APPROVAL_STATUS   keputusan: '1' disetujui, '2' ditolak
--     RESOLVED_BY       pelakunya
--     RESOLVED_DATETIME  waktunya
--
-- Kolom pembuatan — OPEN_PROTECTION_ID, POLICY_NO, CLAIM_NO, ID_CLAIM, PROTECTION_TYPE_ID, CREATE_DATE, CREATED_BY,
-- NOTES, OLD_DATA, NEW_DATA, OBJECT_NAME, BRANCH_NAME — dimiliki modul `inputreqprotection` dan
-- TIDAK PERNAH DITULIS di sini (`P-1`). Dijaga uji di query_test.go.
--
-- ============================================================================
-- MEMBACA BUKAN MENULIS: EMPAT KOLOM DETAIL PERUBAHAN
-- ============================================================================
--
-- `P-1` mengatur siapa yang MENULIS sebuah kolom, bukan siapa yang boleh membacanya. Layar
-- akseptasi karena itu membaca empat kolom milik modul lain:
--
--     OLD_DATA      NEW_DATA      OBJECT_NAME      BRANCH_NAME
--
-- Keempatnya mengisi panel "Detail Perubahan" pada form, yang di Pega muncul bersyarat:
--
--     Section/AcceptProtectionSection-Section.xml
--       pyContainerVisibleWhen  .TypeProtection==8 || .TypeProtection==7
--         ==7 -> "Detail Perubahan DOL"              OLD_DATA -> NEW_DATA sebagai TANGGAL
--         ==8 -> "Detail Perubahan Cause Of Loss"    OLD_DATA -> NEW_DATA sebagai KODE
--
-- Bagi kedua tipe itu, melihat nilai sebelum dan sesudah ADALAH inti keputusannya. Tanpa
-- keempat kolom ini petugas menyetujui perubahan tanpa tahu apa yang diubah.
--
-- Keempatnya ikut ditarik pada `acceptance_list` meski grid TIDAK menampilkannya. Itu
-- disengaja: satu bentuk SELECT melayani satu fungsi pemindaian, dan dua SELECT yang nyaris
-- sama akan berbeda isinya cepat atau lambat — alasan yang sama dengan penyatuan kedua
-- antrean di atas.
--
-- `DISTRICT` yang juga ada di form Pega TIDAK ikut: tabelnya tidak punya kolom itu.


-- name: acceptance_count
-- Jumlah permintaan yang menunggu keputusan pada satu antrean.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIM_OPENPROTECTION
 WHERE APPROVAL_STATUS IS NULL
   AND CLAIM_NO IS NOT NULL
   AND POLICY_NO IS NOT NULL
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')
   AND ( ( :1 = 1 AND TRIM(PROTECTION_TYPE_ID) = :2 )
         OR ( :3 = 0 AND (PROTECTION_TYPE_ID IS NULL OR TRIM(PROTECTION_TYPE_ID) <> :4) ) )
   AND ( :5 IS NULL
         OR UPPER(OPEN_PROTECTION_ID)      LIKE :6
         OR UPPER(POLICY_NO) LIKE :7
         OR UPPER(CLAIM_NO) LIKE :8 )


-- name: acceptance_list
-- Satu halaman antrean akseptasi.
--
-- Diurutkan MENURUN menurut tanggal permintaan dibuat, dengan nomor sebagai pemutus seri
-- supaya paginasi tidak menampilkan satu baris dua kali.
SELECT p.OPEN_PROTECTION_ID,
       p.POLICY_NO,
       p.CLAIM_NO,
       p.ID_CLAIM,
       p.PROTECTION_TYPE_ID,
       t.PROTECTION_TYPE_NAME,
       p.CREATE_DATE,
       p.NOTES,
       p.CREATED_BY,
       p.OLD_DATA,
       p.NEW_DATA,
       p.OBJECT_NAME,
       p.BRANCH_NAME,
       p.APPROVAL_STATUS,
       p.RESOLVED_DATETIME,
       p.RESOLVED_BY,
       p.OBJECT_ID,
       p.OBJECT_COVERAGE_ID
  FROM POOLDATA.T_CLAIM_OPENPROTECTION p
  LEFT JOIN POOLDATA.M_CLAIM_PROTECTION_TYPE t
         ON TRIM(t.PROTECTION_TYPE_ID) = TRIM(p.PROTECTION_TYPE_ID)
 WHERE p.APPROVAL_STATUS IS NULL
   AND p.CLAIM_NO IS NOT NULL
   AND p.POLICY_NO IS NOT NULL
   AND (p.STATUS_ACTIVE IS NULL OR TRIM(p.STATUS_ACTIVE) = '1')
   AND ( ( :1 = 1 AND TRIM(p.PROTECTION_TYPE_ID) = :2 )
         OR ( :3 = 0 AND (p.PROTECTION_TYPE_ID IS NULL OR TRIM(p.PROTECTION_TYPE_ID) <> :4) ) )
   AND ( :5 IS NULL
         OR UPPER(p.OPEN_PROTECTION_ID) LIKE :6
         OR UPPER(p.POLICY_NO)          LIKE :7
         OR UPPER(p.CLAIM_NO)           LIKE :8 )
 ORDER BY p.CREATE_DATE DESC, p.OPEN_PROTECTION_ID DESC
OFFSET :9 ROWS FETCH NEXT :10 ROWS ONLY


-- name: acceptance_get
-- Satu permintaan untuk form akseptasi.
--
-- TIDAK menyaring APPROVAL_STATUS, dan itu disengaja: layar ini antrean BERSAMA
-- (`Flow/CreateProtection_Flow.xml` menempatkannya di workbasket ProtectionPNC), sehingga
-- sebuah baris dapat diputuskan petugas lain kapan saja. Form harus tetap terbuka supaya
-- pesannya dapat menyatakan keputusan siapa dan kapan — bukan sekadar "tidak ditemukan".
--
-- ============================================================================
-- NAMA TERTANGGUNG, START DATE TIME, DAN END DATE TIME DARI T_GENERAL
-- ============================================================================
--
-- Ketiganya sebelumnya DIBIARKAN KOSONG — adapter mencatat bahwa ia "milik snapshot polis
-- yang modulnya belum terpasang". Work Owner menunjuk sumbernya 2026-09-27:
--
--     select startdate, enddate, theinsured from pooldata.t_general
--      where nopolis = [nopolis opc]
--      order by to_number(prodke) desc fetch first 1 rows only
--
-- Terverifikasi: `T_GENERAL` 201.566 baris, `THEINSURED` terisi 200.586, `STARTDATE` dan
-- `ENDDATE` masing-masing 201.501. Dari 8 proteksi ber-`POLICY_NO`, 7 polisnya ketemu.
--
-- ============================================================================
-- URUTANNYA NUMERIK, SEBAGAIMANA MESTINYA
-- ============================================================================
--
-- Work Owner menetapkan 2026-09-27: *"prodke adalah varchar/char, namun isinya pasti angka,
-- gunakan order by sebagaimana mestinya agar data yang diambil selalu prodke paling baru
-- (angka terbesar)"*.
--
-- Karena itu urutannya `CAST(TRIM(g.PRODKE) AS NUMERIC) DESC` — perbandingan ANGKA, bukan
-- teks. Kolomnya memang `VARCHAR2(100)`, tetapi yang menentukan perpanjangan terbaru adalah
-- nilainya sebagai bilangan.
--
-- # Kenapa BUKAN `to_number(prodke)` seperti yang dituliskan
--
-- `to_number(x)` berargumen satu sah di Oracle, **tidak sah di PostgreSQL** — di sana
-- `to_number` menuntut format mask. Ia karena itu tidak berpindah, dan `D-20` menetapkan
-- satu set SQL untuk kedua basis data.
--
-- `CAST(... AS NUMERIC)` adalah bentuk ANSI dari hal yang sama, dan Oracle menerimanya —
-- diuji langsung, bukan diandaikan:
--
--     CAST('0012' AS NUMERIC)    OK -> 12
--     CAST('0012' AS NUMBER)     OK         (khas Oracle, tidak dipakai)
--     CAST('0012' AS BIGINT)     ORA-00902  (tidak didukung)
--
-- # Kenapa BUKAN trik teks
--
-- Versi pertama memakai `LPAD(TRIM(PRODKE), 10, '0')`, yang menghasilkan urutan sama hari
-- ini. Ia ditinggalkan karena ia **trik**, bukan pernyataan maksud: lebar 10 dipilih dari
-- data hari ini, dan `PRODKE` yang kelak melewati sepuluh digit akan salah urut DIAM-DIAM.
--
-- Urutan teks POLOS lebih salah lagi — 13.181 baris ber-`PRODKE` berawalan nol.
--
-- # Yang diukur sebelum diganti
--
--     baris ber-PRODKE                           201.540   CAST berhasil atas SEMUANYA
--     PRODKE bukan angka                               0
--     polis dengan lebih dari satu PRODKE          6.639
--     yang urutannya BERBEDA dari to_number             0
--
-- Baris yang terpilih karena itu identik dengan yang `to_number(prodke) desc` pilih, pada
-- seluruh polis yang punya lebih dari satu perpanjangan.
--
-- # Risiko yang diterima
--
-- `CAST` adalah konversi: satu baris ber-`PRODKE` bukan angka akan menggagalkan kueri ini.
-- Work Owner menyatakan isinya pasti angka, dan hitungan membenarkannya (nol non-angka dari
-- 201.540). Bila kelak muncul, yang terjadi adalah galat yang TERLIHAT — bukan urutan salah
-- yang diam. Untuk nilai yang menentukan perpanjangan mana yang dipakai, gagal keras lebih
-- baik daripada salah diam.
--
-- ============================================================================
-- TIGA SUBKUERI, BUKAN SATU JOIN
-- ============================================================================
--
-- Satu `LEFT JOIN ... FETCH FIRST` tidak dapat ditulis per baris induk tanpa lateral join,
-- yang sintaksnya berbeda antara Oracle dan PostgreSQL. Tiga subkueri berkorelasi portabel
-- di keduanya, dan `NOPOLIS` terindeks sehingga ketiganya menempuh jalur yang sama.
--
-- Konsekuensinya diterima: bila sebuah polis punya beberapa PRODKE, ketiganya HARUS
-- memilih baris yang sama — dan itu dijaga oleh `ORDER BY` yang identik. Mengubah salah
-- satunya akan menggabungkan nama tertanggung dari satu perpanjangan dengan tanggal dari
-- perpanjangan lain, tanpa satu pun gejala.
SELECT p.OPEN_PROTECTION_ID,
       p.POLICY_NO,
       p.CLAIM_NO,
       p.ID_CLAIM,
       p.PROTECTION_TYPE_ID,
       t.PROTECTION_TYPE_NAME,
       p.CREATE_DATE,
       p.NOTES,
       p.CREATED_BY,
       p.OLD_DATA,
       p.NEW_DATA,
       p.OBJECT_NAME,
       p.BRANCH_NAME,
       p.APPROVAL_STATUS,
       p.RESOLVED_DATETIME,
       p.RESOLVED_BY,
       p.OBJECT_ID,
       p.OBJECT_COVERAGE_ID,
       (SELECT g.THEINSURED FROM POOLDATA.T_GENERAL g
         WHERE UPPER(TRIM(g.NOPOLIS)) = UPPER(TRIM(p.POLICY_NO))
         ORDER BY CAST(TRIM(g.PRODKE) AS NUMERIC) DESC
         FETCH FIRST 1 ROW ONLY),
       (SELECT g.STARTDATE FROM POOLDATA.T_GENERAL g
         WHERE UPPER(TRIM(g.NOPOLIS)) = UPPER(TRIM(p.POLICY_NO))
         ORDER BY CAST(TRIM(g.PRODKE) AS NUMERIC) DESC
         FETCH FIRST 1 ROW ONLY),
       (SELECT g.ENDDATE FROM POOLDATA.T_GENERAL g
         WHERE UPPER(TRIM(g.NOPOLIS)) = UPPER(TRIM(p.POLICY_NO))
         ORDER BY CAST(TRIM(g.PRODKE) AS NUMERIC) DESC
         FETCH FIRST 1 ROW ONLY)
  FROM POOLDATA.T_CLAIM_OPENPROTECTION p
  LEFT JOIN POOLDATA.M_CLAIM_PROTECTION_TYPE t
         ON TRIM(t.PROTECTION_TYPE_ID) = TRIM(p.PROTECTION_TYPE_ID)
 WHERE UPPER(TRIM(p.OPEN_PROTECTION_ID)) = :1
   AND (p.STATUS_ACTIVE IS NULL OR TRIM(p.STATUS_ACTIVE) = '1')


-- name: acceptance_decide
-- Menuliskan keputusan akseptasi.
--
-- `APPROVAL_STATUS IS NULL` ada DI DALAM WHERE, bukan hanya diperiksa lebih dulu di Go.
-- Itulah yang benar-benar menahan petugas KEDUA pada antrean bersama: pemeriksaan di Go
-- dilewati keduanya bila mereka menekan tombol bersamaan.
--
-- Kedua syarat kelengkapan ikut diulang. Sebuah permintaan yang belum tertaut klaim tidak
-- muncul di antrean, tetapi tautan yang disimpan masih dapat membukanya — dan keputusan
-- atasnya harus ditolak, bukan diterima diam-diam.
UPDATE POOLDATA.T_CLAIM_OPENPROTECTION
   SET APPROVAL_STATUS  = :1,
       RESOLVED_BY      = :2,
       RESOLVED_DATETIME = :3
 WHERE UPPER(TRIM(OPEN_PROTECTION_ID)) = :4
   AND APPROVAL_STATUS IS NULL
   AND CLAIM_NO IS NOT NULL
   AND POLICY_NO IS NOT NULL
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')


-- name: claim_apply_loss_date
-- Menerapkan Tanggal Kejadian baru ke data klaim.
--
-- ============================================================================
-- SASARANNYA T_CLAIM_PNC, BUKAN T_CLAIMLIST_ADMIN
-- ============================================================================
--
-- Di Pega, menyetujui permintaan tipe '7' mengubah `ClaimData.DateOfLoss` pada work object
-- klaim.
--
-- Sasaran di sistem baru ditetapkan Work Owner 2026-09-26, setelah sempat keliru mengarah ke
-- tabel datar:
--
--   > "ternyata tabel t_claimlist_admin hanya untuk dashboard, jadi datanya tidak
--   >  diganti-ganti. untuk perubahan data dari open proteksi, update data di t_claim_pnc
--   >  saja."
--
-- `T_CLAIMLIST_ADMIN` adalah tabel BACA untuk dashboard dan daftar. Menulisinya akan membuat
-- dua sumber kebenaran untuk hal yang sama, dan yang satu akan menyimpang dari yang lain
-- tanpa gejala.
--
-- ============================================================================
-- KUNCINYA CLAIMID, BUKAN NOMOR KLAIM
-- ============================================================================
--
-- `T_CLAIM_PNC` dikunci `CLAIMID`, yang berbentuk `ASM-FW-GCNMFW-WORK PNC-1865` — terbukti
-- dari join yang dipakai modul `inputreqprotection`: `c.CLAIMID = w.PZINSKEY`.
--
-- Nilai itulah yang tersimpan pada `T_CLAIM_OPENPROTECTION.ID_CLAIM` (lihat
-- `inputreqprotection.ClaimReferenceOf`). Memakai `CLAIM_NO` di sini TIDAK akan menemukan
-- baris mana pun untuk klaim warisan — dan tidak menemukan apa pun bukan galat, melainkan
-- diam.
--
-- ============================================================================
-- JUMLAH BARIS TERSENTUH DIPERIKSA PEMANGGIL
-- ============================================================================
--
-- `UPDATE` yang tidak menyentuh apa pun BUKAN galat basis data: ia keadaan yang harus
-- dijawab pemanggil, bukan disembunyikan.
UPDATE POOLDATA.T_CLAIM_PNC
   SET DATEOFLOSS = :1
 WHERE UPPER(TRIM(CLAIMID)) = :2


-- name: cause_of_loss_describe
-- Mencari DESKRIPSI sebuah Penyebab Kerugian dari kodenya.
--
-- ============================================================================
-- PASANGAN KOLOMNYA DIBUKTIKAN, BUKAN DIANDAIKAN
-- ============================================================================
--
-- `T_CLAIM_OBJECTCOVERAGE` menyimpan DUA kolom berdampingan — `CAUSEOFLOSSID` dan
-- `CAUSEOFLOSS` — dan menerapkan perubahan menuntut mengetahui mana yang berisi apa.
-- Dua bukti menetapkannya, dan hanya satu pemasangan yang konsisten dengan keduanya:
--
--   1. Data produksi, klaim `PNC-1452`: `CAUSEOFLOSSID = 12003`, `CAUSEOFLOSS = WINDSTORM`.
--
--   2. Rule Pega `RDB List/GetCauseofLossDesc-SQL.xml`, kueri lengkapnya:
--
--        SELECT DESCRIPTION as "LSC_NOTE" FROM POOLDATA.V_D_CAUSE_OF_LOSS
--         WHERE D_COL_ID={InputParam.CauseOfLoss}
--
--      Yaitu: diberi KODE, kembalikan DESKRIPSI. Itulah pencarian kanoniknya di sistem lama.
--
-- Jadi `D_COL_ID` -> `CAUSEOFLOSSID`, dan `DESCRIPTION` -> `CAUSEOFLOSS`. Bentuk nilainya
-- membenarkan: contoh isi master "11997 FIRE - OPEN FLAME" dan "12033 WRECK REMOVAL" sebentuk
-- dengan `WINDSTORM`, `HURRICANE`, `ILLNESS`, `STORM` yang tersimpan di coverage.
--
-- ============================================================================
-- TABEL INDUK, BUKAN VIEW
-- ============================================================================
--
-- Pega membaca `V_D_CAUSE_OF_LOSS`; di sini tabel induknya, mengikuti ketetapan Work Owner
-- 2026-10-05 ("menggunakan D_CAUSE_OF_LOSS jangan view"). Dropdown yang menawarkan pilihan
-- dan penerapan yang menuliskannya WAJIB membaca sumber yang sama — kalau tidak, sebuah kode
-- dapat tampil di dropdown lalu ditolak saat diterapkan.
--
-- ============================================================================
-- NOL BARIS BUKAN GALAT BASIS DATA
-- ============================================================================
--
-- Kode yang tidak ada di master menghasilkan nol baris, dan pemanggillah yang menjawabnya
-- (`ErrUnknownCauseOfLoss`). Menuliskannya lewat subkueri skalar di dalam UPDATE akan
-- menghasilkan `CAUSEOFLOSS = NULL` tanpa satu pun gejala — baris yang kodenya terisi dan
-- namanya kosong.
SELECT d.DESCRIPTION
  FROM POOLDATA.D_CAUSE_OF_LOSS d
 WHERE TRIM(d.D_COL_ID) = :1


-- name: claim_apply_cause_of_loss
-- Menerapkan Penyebab Kerugian baru ke SATU baris coverage klaim.
--
-- ============================================================================
-- SASARANNYA SATU BARIS, DITUNJUK PEMOHON
-- ============================================================================
--
-- Kuncinya `CLAIMID` + `OBJECTID` + `OBJECTCOVERAGEID` — bentuk yang sama dipakai
-- `T_CLAIM_SPREADING`. Ketiganya diperlukan: pada klaim `PNC-1452`, `JackHugh / Resiko A`
-- muncul TIGA KALI dengan Penyebab Kerugian berbeda, sehingga menyaring dengan nama akan
-- mengubah baris yang salah dua dari tiga kali.
--
-- `OBJECTID` dan `OBJECT_COVERAGE_ID` datang dari panel pemilih pada form permintaan, dan
-- disimpan di kolom `OBJECT_ID` dan `OBJECT_COVERAGE_ID` milik `T_CLAIM_OPENPROTECTION`
-- (ditambahkan 2026-10-05, lihat `migrations/0015_openprotection_sasaran_coverage.up.sql`).
--
-- ============================================================================
-- KEDUA KOLOM DIUBAH BERSAMAAN
-- ============================================================================
--
-- Work Owner, 2026-10-05: *"ingat ganti cause of loss itu ganti causeoflossid juga"*.
--
-- Mengubah salah satunya saja menghasilkan baris yang namanya berkata satu hal dan kodenya
-- berkata hal lain — dan laporan yang mengelompokkan menurut kode akan menghitungnya ke
-- golongan lama sementara layar menampilkan yang baru.
--
-- Deskripsinya DITERIMA SEBAGAI PARAMETER, hasil `cause_of_loss_describe` yang dijalankan
-- lebih dulu di dalam transaksi yang sama. Lihat kueri itu untuk alasannya.
--
-- ============================================================================
-- SATU SELISIH DARI PEGA, DIAMBIL SADAR
-- ============================================================================
--
-- Pega MEMBAWA keduanya di dalam permintaan: `Activity/InsertOpenProtectionCase-Act.xml`
-- menyalin `.ClaimDataProtect.CauseOfLoss` ke kolom teks (`:2956`) dan `.CauseOfLossID` ke
-- kolom kode (`:3003`), keduanya dari permintaan yang disimpan.
--
-- Di sini hanya KODENYA yang disimpan — Work Owner, 2026-10-05: *"old data new data simpan
-- idcol aja"* — dan teksnya dicari saat menerapkan. Akibatnya permintaan yang dibuat bulan
-- lalu lalu disetujui hari ini menuliskan teks yang berlaku HARI INI, bukan teks yang
-- kebetulan tersimpan saat permintaan dibuat.
--
-- Konsekuensi yang diterima: kode yang DIHAPUS dari master antara meminta dan menyetujui
-- membuat keputusannya ditolak (`ErrUnknownCauseOfLoss`), sedangkan Pega akan tetap
-- menuliskannya memakai teks lama. Itu memang yang dikehendaki — menuliskan penyebab kerugian
-- yang sudah tidak berlaku lebih buruk daripada menolaknya.
--
-- ============================================================================
-- BARIS TERHAPUS TIDAK IKUT BERUBAH
-- ============================================================================
--
-- `DIHAPUS_PADA IS NULL` menjaga penghapusan lunak (`D-66`): coverage yang sudah dibuang dari
-- klaim tidak boleh berubah karena persetujuan yang menunjuknya. Penyaring yang sama dipakai
-- `claim_coverages` saat menawarkan pilihannya, sehingga yang dapat dipilih dan yang dapat
-- diubah adalah himpunan yang sama.
--
-- Akibatnya nol baris tersentuh ketika coverage-nya dibuang setelah permintaan diajukan —
-- keadaan yang dijawab pemanggil sebagai `ErrClaimNotSynced`, bukan diabaikan.
UPDATE POOLDATA.T_CLAIM_OBJECTCOVERAGE
   SET CAUSEOFLOSSID = :1,
       CAUSEOFLOSS   = :2
 WHERE UPPER(TRIM(CLAIMID)) = :3
   AND TRIM(OBJECTID) = :4
   AND TRIM(OBJECTCOVERAGEID) = :5
   AND DIHAPUS_PADA IS NULL
