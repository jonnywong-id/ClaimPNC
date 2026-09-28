-- Kueri modul Inbox Manager — menu `MENU_ID 58`, pengganti harness `UserInbox_Harness`.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- ============================================================================
-- SUMBER DATA DASHBOARD BERPINDAH — KETETAPAN WORK OWNER 2026-09-28
-- ============================================================================
--
-- Dari seluruh kueri layar ini, hanya EMPAT yang menyentuh skema DATAPEGA, dan keempatnya
-- milik tab Outstanding:
--
--     RDB List/CountOutstandingManager      pencacah di kepala layar
--     RDB List/GetPICDashboardOS            grid "per PIC"
--     RDB List/GetBisnisGroupDashboardOS    grid "per Grup Bisnis"
--     RDB List/GetYearDashboardOS           daftar tahun untuk penyaring
--
-- Keempatnya menggabungkan dua tabel:
--
--     datapega.pc_asm_fw_gcnmfw_work c
--       JOIN datapega.pc_assign_worklist b ON c.pzinskey = b.pxrefobjectkey
--
-- Keduanya kini digantikan SATU tabel datar POOLDATA.T_CLAIMLIST_ADMIN, sehingga berkas ini
-- TIDAK menyentuh skema DATAPEGA sama sekali. Sembilan kueri lain sudah POOLDATA sejak awal
-- dan tidak berubah sumbernya.
--
-- `GetYearDashboardOS` TIDAK dibawa: ia mengisi daftar pilihan tahun, dan tab Outstanding di
-- sistem baru tidak punya penyaring periode — lihat catatan pada tab itu di tab.go. Membawanya
-- berarti menggambar penyaring yang tidak menyaring apa pun.
--
-- # Dua cacat yang ikut hilang bersama gabungannya
--
-- Gabungan lamanya `INNER`, dan bentuk itu membawa dua cacat yang tidak pernah menghasilkan
-- galat:
--
--   * klaim TANPA penugasan terbuka LENYAP dari hitungan — padahal justru itu pekerjaan yang
--     terhenti dan paling perlu dilihat penyelia;
--   * klaim dengan LEBIH DARI SATU penugasan terhitung BERKALI-KALI.
--
-- Tabel datar menyimpan satu baris per klaim, sehingga keduanya hilang dengan sendirinya. Ini
-- SELISIH PERILAKU yang disengaja — lihat inboxmanager.PlannedDifferences, dan ia wajib
-- disebut lebih dulu pada uji kesetaraan gerbang 1 (`D-54`).
--
-- ============================================================================
-- KONTRAK KOLOM: POOLDATA.T_CLAIMLIST_ADMIN — DIBACA DARI KATALOG, BUKAN DITEBAK
-- ============================================================================
--
-- Tabel ini NOL KEMUNCULAN di seluruh export Pega, jadi kolomnya tidak dapat dibaca dari sana.
-- Tiga kali pada 2026-09-27 sebuah kolom disimpulkan TIDAK ADA dari sumber yang tidak
-- membuktikannya — modul lain yang memakai kolom berbeda, atau dokumen yang tidak
-- menyebutnya — dan KETIGANYA keliru.
--
-- Karena itu daftar di bawah dibaca LANGSUNG dari `ALL_TAB_COLUMNS` pada 2026-09-28. Tabelnya
-- memuat 58 kolom, bukan 40 seperti yang tercatat 2026-09-22, dan kesepuluh kolom yang
-- dibutuhkan keempat kueri di atas SELURUHNYA SUDAH ADA:
--
--   Kolom di sini      Sumber lamanya                  Terisi (dari 1.014 baris)
--   -----------------  ------------------------------  -------------------------
--   PXOBJCLASS         c.pxobjclass                    1.014
--   PYSTATUSWORK       c.pystatuswork                  1.014
--   USERTEKNIS_1       c.userteknis_1                    354
--   PXCREATEDATETIME   c.pxcreatedatetime              1.014
--   GROUPPANEL_1       c.grouppanel_1                  1.014
--   BRANCHNAME         c.branchname                    1.014
--   BUSINESSCODE_1     c.businesscode_1                1.014
--   BUSINESSGROUPID    f.businessgroupid (lewat join)  1.014
--   PXFLOWNAME         b.pxflowname                    1.006
--   PXTASKLABEL        b.pxtasklabel                   1.006
--
-- TIDAK ADA SATU PUN KOLOM YANG PERLU DIMINTA KE DBA untuk modul ini.
--
-- Daftar kolom lengkapnya kini ada, dan ia menutup `L-3` — lihat
-- docs/kolom-t-claimlist-admin.md.
--
-- ============================================================================
-- PXOBJCLASS WAJIB DISARING — TABEL INI MEMUAT LEBIH DARI SATU KELAS KASUS
-- ============================================================================
--
-- Ini BUKAN penyaring yang boleh dihilangkan dengan alasan "tabelnya memang untuk Claim PNC".
-- Dihitung langsung pada 2026-09-28:
--
--     ASM-FW-GCNMFW-Work-PNC              872 baris
--     ASM-FW-GCNMFW-Work-ReceiveDocument  142 baris
--
-- Tanpa penyaring ini, 142 baris Receive Document ikut masuk hitungan klaim — dan karena
-- keduanya sama-sama punya PYID, tidak ada yang tampak salah.
--
-- Versi modul ini yang pernah ada (dihapus 2026-09-27) membuang penyaring itu dengan alasan
-- "isinya diandaikan klaim PNC saja" dan menandainya sendiri "ASUMSI INI PERLU DIKONFIRMASI".
-- Konfirmasinya kini ada, dan hasilnya berlawanan.
--
-- ============================================================================
-- SETIAP PENANDA BIND MUNCUL TEPAT SEKALI, DAN ITU BUKAN KERAPIAN
-- ============================================================================
--
-- Driver mengikat argumen MENURUT URUTAN KEMUNCULAN, bukan menurut nomor penandanya.
-- Penanda `:4` yang dipakai tiga kali karena itu menuntut tiga argumen, dan bila dikirim satu
-- kuerinya gagal dengan ORA-01008 — cacat yang tidak tertangkap uji memori, tidak tertangkap
-- uji teks kueri, dan tidak tertangkap `-periksa`. Ia sudah pernah menggigit modul Inbox RCL
-- (catatan pengembangan §63.2).
--
-- Seluruh kueri di berkas ini karena itu memakai penanda yang MENAIK dan MUNCUL TEPAT SEKALI,
-- dan aturan itu dikunci uji `TestSetiapPenandaBindMunculTepatSekaliDanBerurutan`.
--
-- Akibatnya terlihat pada penyaring lini bisnis di bawah: alih-alih membandingkan satu nilai
-- lima kali, Go mengirim LIMA BENDERA yang sudah dihitungnya. Bentuk itu sekaligus membuat
-- kuerinya terbaca — setiap cabang menyebut sendiri lini bisnis mana yang dilayaninya.
--
-- ============================================================================
-- TIGA CACAT KUERI LAMA YANG DIPERBAIKI
-- ============================================================================
--
-- 1. `GetBisnisGroupDashboardOS` TIDAK menyaring status kerja sama sekali, sehingga dashboard
--    berjudul "Outstanding" ikut menghitung klaim yang sudah selesai — dan angkanya tidak
--    pernah cocok dengan pencacah di kepala layar yang sama. Penyaringnya ditambahkan.
--
-- 2. `CountOutstandingManager` menggabung `t_claim_objectlist` tanpa memilih satu kolom pun
--    darinya, sehingga klaim berobjek banyak terhitung berkali-kali pada pencacah yang
--    menamai dirinya jumlah klaim. Gabungan itu tidak dibawa.
--
-- 3. Cabang TRAVEL menulis `GROUP_PANEL='005'` TANPA alias pada kueri yang tabelnya memakai
--    `grouppanel_1` — kemungkinan galat runtime, atau menunjuk kolom tabel lain. Di sini ia
--    `a.GROUPPANEL_1 = '005'`, sejalan dengan cabang PA pada kueri yang sama.
--
-- ============================================================================
-- YANG SENGAJA TIDAK DIBAWA
-- ============================================================================
--
-- * Nilai kesembilan Dashboard Klaim — `(TTLAKSEP+TTLOS)+(TTLOS-TTLAKSEP)`. Tidak ada satu
--   pun keterangan di export yang menyatakan angka itu mewakili apa, dan menampilkan jumlah
--   uang tanpa mengetahui artinya lebih buruk daripada tidak menampilkannya.
--
-- * Penyaring Leader / Member / Fac In pada Dashboard OS. Ia membaca `mst_det_sales` lewat
--   DB Link `@asmd`; penggantinya API yang belum ada (`D-25`, `R-03`).
--
-- * Penyaring `DISC2` pada `GetPICDashboardOS`. Ia mematok `TYPE_BUSINESS='NONMBU'` sebagai
--   teks tetap, bertentangan dengan pilihan lini bisnis di layarnya sendiri.
--
-- * Sub-tab "Approval Progress Klaim" beserta `ShowApproveProgressKlaim`. Kontainernya
--   bersyarat `1==2` di `Section/Sec_PaymentAkseptasiKlaimCase1-Section.xml:67064` — sebuah
--   kondisi yang tidak pernah benar — sehingga bagian itu sudah mati di Pega.

-- name: check_table
-- Memastikan tabel sumber dashboard terbaca dari koneksi yang dipakai.
--
-- Tidak menyentuh satu baris pun: yang diperiksa adalah hak baca dan keberadaan tabelnya.
SELECT 1
  FROM POOLDATA.T_CLAIMLIST_ADMIN
 WHERE 1 = 0

-- name: check_dashboard_snapshot
-- Memastikan tabel cuplikan dashboard terbaca.
--
-- Ia pemeriksaan TERPISAH karena tabelnya berbeda dan kegagalannya menunjuk hal yang berbeda:
-- `T_CLAIMLIST_ADMIN` memasok tab Outstanding, `PEGA_DASHBOARDPNC` memasok dua tab lainnya.
SELECT 1
  FROM POOLDATA.PEGA_DASHBOARDPNC
 WHERE 1 = 0

-- name: check_line_business
-- Memastikan kolom LINE_BUSINESS terbaca dari koneksi yang dipakai.
--
-- Ia layak diperiksa tersendiri meski kolomnya SUDAH ADA, karena namanya pernah salah
-- ditulis: migrasi `0004_login_line_business` menambahkan `LINEBUSINESS` tanpa garis bawah
-- dan DICABUT sebelum dijalankan justru karena itu (`migrations/0004_DICABUT.md`). Ejaan yang
-- keliru di sini menghasilkan ORA-00904, dan tanpa pemeriksaan ini galatnya baru muncul saat
-- penyelia membuka layar.
SELECT 1
  FROM POOLDATA.M_LOGIN_PNC
 WHERE LINE_BUSINESS IS NOT NULL
   AND 1 = 0

-- name: line_business_for
-- Lini bisnis seorang petugas — padanan `OperatorID.pyPosition` sistem lama.
--
-- Kuerinya sengaja SAMA PERSIS dengan `line_business_for` milik modul Inbox Outstanding dan
-- Inbox Manager Admin. Ketiganya membaca kolom yang sama untuk keperluan yang sama;
-- menulisnya berbeda hanya menciptakan tempat yang dapat menyimpang.
--
-- Tabelnya dimiliki modul Login; modul ini hanya MEMBACA satu kolom (`P-1`).
--
-- Bind: :1 login petugas, sudah huruf besar dan tanpa spasi tepi
SELECT p.LINE_BUSINESS
  FROM POOLDATA.M_LOGIN_PNC p
 WHERE UPPER(TRIM(p.LOGIN_ID)) = :1

-- name: dashboard_refreshed_at
-- Waktu cuplikan dashboard terakhir disegarkan.
--
-- Asal: `Activity/LastRefresh_act` → `RDB List/GetDASHBOARDPNCrefresh_sql`
--
--   select refreshdate as "DateOfLoss" from pega_DASHBOARDPNC_refresh
--
-- Alias `DateOfLoss` untuk sebuah waktu penyegaran tidak dibawa (`D-19`) — ia satu lagi nama
-- yang tidak mencerminkan isinya.
SELECT r.REFRESHDATE
  FROM POOLDATA.PEGA_DASHBOARDPNC_REFRESH r

-- name: count_outstanding
-- Pencacah "Outstanding" di kepala layar.
--
-- Asal: `RDB List/CountOutstandingManager-SQL.xml`, dengan dua tabel DATAPEGA diganti satu
-- tabel datar dan gabungan `t_claim_objectlist` dibuang (lihat cacat #2 di kepala berkas).
--
-- # Kenapa penyaring alur DIBIARKAN LOLOS pada baris tanpa penugasan
--
-- `PXFLOWNAME NOT IN (…)` bernilai UNKNOWN saat kolomnya NULL, sehingga baris itu akan
-- terbuang. Delapan dari 1.014 baris berkolom NULL, dan NULL di sini berarti klaim itu belum
-- punya penugasan terbuka — persis baris yang gabungan `INNER` lama buang dan yang sengaja
-- kita pertahankan. Karena itu ditulis eksplisit sebagai `IS NULL OR NOT IN`.
--
-- Bind: :1..:5 bendera lini bisnis (1 = cabang ini berlaku), dihitung di Go
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIMLIST_ADMIN a
  JOIN POOLDATA.BUSINESS f
    ON a.BUSINESSCODE_1 = f.ID
 WHERE a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND a.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (a.PXFLOWNAME IS NULL OR a.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1'))
   AND (a.PXTASKLABEL IS NULL OR a.PXTASKLABEL NOT IN ('FixCorrespondence'))
   AND a.USERTEKNIS_1 IS NOT NULL
   AND ( :1 = 1
      OR ( :2 = 1
           AND a.GROUPPANEL_1 IN ('003', '004', '006')
           AND f.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023')
           AND a.BRANCHNAME <> 'ASNET' )
      OR ( :3 = 1
           AND f.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023') )
      OR ( :4 = 1 AND a.GROUPPANEL_1 = '002' )
      OR ( :5 = 1 AND a.GROUPPANEL_1 = '005' ) )

-- name: dashboard_os_pic
-- Grid "Outstanding per PIC".
--
-- Asal: `RDB List/GetPICDashboardOS-SQL.xml`. Bentuk `UNION ALL` dengan baris 'ALL' di atas
-- DIBAWA apa adanya — ia baris ringkasan yang memang tampil di layar lama, bukan kelebihan.
--
-- Bind: :1..:5 dan :6..:10 bendera lini bisnis, dikirim dua kali karena kueri ini dua bagian
SELECT 'ALL'            AS DIMENSION,
       COUNT(*)         AS TOTAL
  FROM POOLDATA.T_CLAIMLIST_ADMIN a
  JOIN POOLDATA.BUSINESS f
    ON a.BUSINESSCODE_1 = f.ID
 WHERE a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND a.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (a.PXFLOWNAME IS NULL OR a.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1'))
   AND (a.PXTASKLABEL IS NULL OR a.PXTASKLABEL NOT IN ('FixCorrespondence'))
   AND a.USERTEKNIS_1 IS NOT NULL
   AND ( :1 = 1
      OR ( :2 = 1
           AND a.GROUPPANEL_1 IN ('003', '004', '006')
           AND f.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023')
           AND a.BRANCHNAME <> 'ASNET' )
      OR ( :3 = 1
           AND f.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023') )
      OR ( :4 = 1 AND a.GROUPPANEL_1 = '002' )
      OR ( :5 = 1 AND a.GROUPPANEL_1 = '005' ) )
UNION ALL
SELECT a.USERTEKNIS_1   AS DIMENSION,
       COUNT(*)         AS TOTAL
  FROM POOLDATA.T_CLAIMLIST_ADMIN a
  JOIN POOLDATA.BUSINESS f
    ON a.BUSINESSCODE_1 = f.ID
 WHERE a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND a.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (a.PXFLOWNAME IS NULL OR a.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1'))
   AND (a.PXTASKLABEL IS NULL OR a.PXTASKLABEL NOT IN ('FixCorrespondence'))
   AND a.USERTEKNIS_1 IS NOT NULL
   AND ( :6 = 1
      OR ( :7 = 1
           AND a.GROUPPANEL_1 IN ('003', '004', '006')
           AND f.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023')
           AND a.BRANCHNAME <> 'ASNET' )
      OR ( :8 = 1
           AND f.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023') )
      OR ( :9 = 1 AND a.GROUPPANEL_1 = '002' )
      OR ( :10 = 1 AND a.GROUPPANEL_1 = '005' ) )
 GROUP BY a.USERTEKNIS_1
 ORDER BY 1

-- name: dashboard_os_business_group
-- Grid "Outstanding per Grup Bisnis".
--
-- Asal: `RDB List/GetBisnisGroupDashboardOS-SQL.xml`.
--
-- Penyaring status kerja DITAMBAHKAN — kueri lamanya tidak punya sama sekali, sehingga
-- dashboard berjudul "Outstanding" ikut menghitung klaim yang sudah selesai (cacat #1 di
-- kepala berkas).
--
-- Bind: :1..:5 bendera lini bisnis
SELECT e.NOTE     AS DIMENSION,
       COUNT(*)   AS TOTAL
  FROM POOLDATA.T_CLAIMLIST_ADMIN a
  JOIN POOLDATA.BUSINESS f
    ON a.BUSINESSCODE_1 = f.ID
  JOIN POOLDATA.BUSINESSGROUP e
    ON f.BUSINESSGROUPID = e.ID
 WHERE a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND a.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (a.PXFLOWNAME IS NULL OR a.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1'))
   AND (a.PXTASKLABEL IS NULL OR a.PXTASKLABEL NOT IN ('FixCorrespondence'))
   AND a.USERTEKNIS_1 IS NOT NULL
   AND ( :1 = 1
      OR ( :2 = 1
           AND a.GROUPPANEL_1 IN ('003', '004', '006')
           AND f.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023')
           AND a.BRANCHNAME <> 'ASNET' )
      OR ( :3 = 1
           AND f.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023') )
      OR ( :4 = 1 AND a.GROUPPANEL_1 = '002' )
      OR ( :5 = 1 AND a.GROUPPANEL_1 = '005' ) )
 GROUP BY e.NOTE
 ORDER BY e.NOTE ASC

-- name: dashboard_produktivitas_business
-- Grid "Produktivitas per Grup Bisnis".
--
-- Asal: `RDB List/GetSumBusinessDashboardProduktivitas_SQL-SQL.xml`.
--
-- # Kedelapan pencacahnya adalah EMPAT KERANJANG × DUA PERIODE
--
-- Alias kolom kueri lamanya tidak menyebut isinya sama sekali — `BRANCHNAME`, `BUSINESSCODE`,
-- `CLIENTID`, `EDMNO`, `FLAGEDMBATAL`, `FOLLOWEDPOLICY`, `IDPEGA`. Artinya dibaca dari
-- `Activity/PNCGetDashboardProduktivitasInbox_Act`, yang menyusun kedua predikatnya:
-- `TempLaporan.LokasiSurveyor` adalah periode yang dipilih, dan `TempLaporan.NamaSurveyor`
-- periode yang sama SATU TAHUN sebelumnya (`- INTERVAL '1' YEAR`).
--
-- # Arti status klaimnya diperiksa ke DATA, bukan disimpulkan
--
-- Pada 2026-09-28, dari 66.972 baris: status `1` (50.074 baris) punya NOAKSEP terisi pada
-- 49.198 di antaranya dan TGL_REJECT nol — ia AKSEPTASI. Status `3` (7.936) punya TGL_REJECT
-- terisi pada 7.904 — ia PENOLAKAN. Sisanya status `0` (8.710) — masih berjalan.
--
-- Status `2` (252 baris) TIDAK masuk keranjang mana pun, dan itu bentuk kueri lamanya yang
-- dibawa apa adanya: keranjang ketiga berbunyi "selain 1, 2, dan 3". Akibatnya ketiga
-- keranjang tidak berjumlah sama dengan kolom Total — selisih terencana, bukan cacat.
--
-- # Penyaring periode menjadi selang setengah terbuka
--
-- Kueri lamanya menyisipkan `to_char(tglklaim,'mm-rrrr')=…` atau `trunc(tglklaim)>=…` lewat
-- pola `ASIS`. Keduanya menyentuh kolomnya pada setiap baris sehingga index tidak terpakai,
-- dan keduanya dilarang `08-TECHNICAL-STRATEGY.md` §4.3. Kedua batas dihitung di Go.
--
-- Bind: :1..:16 batas periode — pasangan (dari, sampai) delapan kali, bergantian antara
--       periode berjalan dan periode tahun lalu, seurutan dengan kedelapan pencacahnya
--       :17..:21 bendera lini bisnis
SELECT z.*
  FROM (
        SELECT d.LGB_NOTE                                                         AS DIMENSION,
               COUNT(CASE WHEN d.TGLKLAIM >= :1 AND d.TGLKLAIM < :2 THEN 1 END)   AS TOTAL_NOW,
               COUNT(CASE WHEN d.TGLKLAIM >= :3 AND d.TGLKLAIM < :4 THEN 1 END)   AS TOTAL_PRIOR,
               COUNT(CASE WHEN d.STSKLAIM = '1'
                           AND d.TGLKLAIM >= :5 AND d.TGLKLAIM < :6 THEN 1 END)   AS ACCEPTED_NOW,
               COUNT(CASE WHEN d.STSKLAIM = '1'
                           AND d.TGLKLAIM >= :7 AND d.TGLKLAIM < :8 THEN 1 END)   AS ACCEPTED_PRIOR,
               COUNT(CASE WHEN d.STSKLAIM = '3'
                           AND d.TGLKLAIM >= :9 AND d.TGLKLAIM < :10 THEN 1 END)  AS REJECTED_NOW,
               COUNT(CASE WHEN d.STSKLAIM = '3'
                           AND d.TGLKLAIM >= :11 AND d.TGLKLAIM < :12 THEN 1 END) AS REJECTED_PRIOR,
               COUNT(CASE WHEN d.STSKLAIM NOT IN ('1', '2', '3')
                           AND d.TGLKLAIM >= :13 AND d.TGLKLAIM < :14 THEN 1 END) AS OUTSTANDING_NOW,
               COUNT(CASE WHEN d.STSKLAIM NOT IN ('1', '2', '3')
                           AND d.TGLKLAIM >= :15 AND d.TGLKLAIM < :16 THEN 1 END) AS OUTSTANDING_PRIOR
          FROM POOLDATA.PEGA_DASHBOARDPNC d
         WHERE d.PIC IS NOT NULL
           AND ( :17 = 1
              OR ( :18 = 1
                   AND d.GROUP_PANEL IN ('003', '004', '006')
                   AND d.GROUPBISNISID NOT IN ('09', '11', '16', '25') )
              OR ( :19 = 1 AND d.GROUPBISNISID IN ('09', '11', '16', '25') )
              OR ( :20 = 1 AND d.GROUP_PANEL = '002' )
              OR ( :21 = 1 AND d.GROUP_PANEL = '005' ) )
         GROUP BY d.LGB_NOTE
       ) z
 WHERE z.TOTAL_NOW + z.TOTAL_PRIOR + z.ACCEPTED_NOW + z.ACCEPTED_PRIOR
     + z.REJECTED_NOW + z.REJECTED_PRIOR + z.OUTSTANDING_NOW + z.OUTSTANDING_PRIOR <> 0
 ORDER BY z.DIMENSION ASC

-- name: dashboard_produktivitas_pic
-- Grid "Produktivitas per PIC".
--
-- Asal: `RDB List/GetSumPICDashboardProduktivitas_SQL-SQL.xml`. Bentuknya sama dengan grid di
-- atas, hanya berdimensi `PIC` alih-alih `LGB_NOTE`, dan diawali baris ringkasan 'ALL' —
-- persis seperti grid "per PIC" pada Dashboard OS.
--
-- Baris 'ALL' kueri lamanya TIDAK menyaring `(jumlah)<>0`, sementara bagian keduanya
-- menyaring. Bentuk itu dibawa apa adanya: baris ringkasan memang layak tampil meski nol,
-- karena ia yang menyatakan "tidak ada apa-apa pada periode ini".
--
-- Bind: :1..:16 batas periode bagian ALL · :17..:21 bendera lini bisnis bagian ALL
--       :22..:37 batas periode bagian per-PIC · :38..:42 bendera lini bisnis bagian per-PIC
SELECT 'ALL'                                                                  AS DIMENSION,
       COUNT(CASE WHEN d.TGLKLAIM >= :1 AND d.TGLKLAIM < :2 THEN 1 END)       AS TOTAL_NOW,
       COUNT(CASE WHEN d.TGLKLAIM >= :3 AND d.TGLKLAIM < :4 THEN 1 END)       AS TOTAL_PRIOR,
       COUNT(CASE WHEN d.STSKLAIM = '1'
                   AND d.TGLKLAIM >= :5 AND d.TGLKLAIM < :6 THEN 1 END)       AS ACCEPTED_NOW,
       COUNT(CASE WHEN d.STSKLAIM = '1'
                   AND d.TGLKLAIM >= :7 AND d.TGLKLAIM < :8 THEN 1 END)       AS ACCEPTED_PRIOR,
       COUNT(CASE WHEN d.STSKLAIM = '3'
                   AND d.TGLKLAIM >= :9 AND d.TGLKLAIM < :10 THEN 1 END)      AS REJECTED_NOW,
       COUNT(CASE WHEN d.STSKLAIM = '3'
                   AND d.TGLKLAIM >= :11 AND d.TGLKLAIM < :12 THEN 1 END)     AS REJECTED_PRIOR,
       COUNT(CASE WHEN d.STSKLAIM NOT IN ('1', '2', '3')
                   AND d.TGLKLAIM >= :13 AND d.TGLKLAIM < :14 THEN 1 END)     AS OUTSTANDING_NOW,
       COUNT(CASE WHEN d.STSKLAIM NOT IN ('1', '2', '3')
                   AND d.TGLKLAIM >= :15 AND d.TGLKLAIM < :16 THEN 1 END)     AS OUTSTANDING_PRIOR
  FROM POOLDATA.PEGA_DASHBOARDPNC d
 WHERE d.PIC IS NOT NULL
   AND ( :17 = 1
      OR ( :18 = 1
           AND d.GROUP_PANEL IN ('003', '004', '006')
           AND d.GROUPBISNISID NOT IN ('09', '11', '16', '25') )
      OR ( :19 = 1 AND d.GROUPBISNISID IN ('09', '11', '16', '25') )
      OR ( :20 = 1 AND d.GROUP_PANEL = '002' )
      OR ( :21 = 1 AND d.GROUP_PANEL = '005' ) )
UNION ALL
SELECT z.*
  FROM (
        SELECT d.PIC                                                              AS DIMENSION,
               COUNT(CASE WHEN d.TGLKLAIM >= :22 AND d.TGLKLAIM < :23 THEN 1 END) AS TOTAL_NOW,
               COUNT(CASE WHEN d.TGLKLAIM >= :24 AND d.TGLKLAIM < :25 THEN 1 END) AS TOTAL_PRIOR,
               COUNT(CASE WHEN d.STSKLAIM = '1'
                           AND d.TGLKLAIM >= :26 AND d.TGLKLAIM < :27 THEN 1 END) AS ACCEPTED_NOW,
               COUNT(CASE WHEN d.STSKLAIM = '1'
                           AND d.TGLKLAIM >= :28 AND d.TGLKLAIM < :29 THEN 1 END) AS ACCEPTED_PRIOR,
               COUNT(CASE WHEN d.STSKLAIM = '3'
                           AND d.TGLKLAIM >= :30 AND d.TGLKLAIM < :31 THEN 1 END) AS REJECTED_NOW,
               COUNT(CASE WHEN d.STSKLAIM = '3'
                           AND d.TGLKLAIM >= :32 AND d.TGLKLAIM < :33 THEN 1 END) AS REJECTED_PRIOR,
               COUNT(CASE WHEN d.STSKLAIM NOT IN ('1', '2', '3')
                           AND d.TGLKLAIM >= :34 AND d.TGLKLAIM < :35 THEN 1 END) AS OUTSTANDING_NOW,
               COUNT(CASE WHEN d.STSKLAIM NOT IN ('1', '2', '3')
                           AND d.TGLKLAIM >= :36 AND d.TGLKLAIM < :37 THEN 1 END) AS OUTSTANDING_PRIOR
          FROM POOLDATA.PEGA_DASHBOARDPNC d
         WHERE d.PIC IS NOT NULL
           AND ( :38 = 1
              OR ( :39 = 1
                   AND d.GROUP_PANEL IN ('003', '004', '006')
                   AND d.GROUPBISNISID NOT IN ('09', '11', '16', '25') )
              OR ( :40 = 1 AND d.GROUPBISNISID IN ('09', '11', '16', '25') )
              OR ( :41 = 1 AND d.GROUP_PANEL = '002' )
              OR ( :42 = 1 AND d.GROUP_PANEL = '005' ) )
         GROUP BY d.PIC
       ) z
 WHERE z.TOTAL_NOW + z.TOTAL_PRIOR + z.ACCEPTED_NOW + z.ACCEPTED_PRIOR
     + z.REJECTED_NOW + z.REJECTED_PRIOR + z.OUTSTANDING_NOW + z.OUTSTANDING_PRIOR <> 0

-- name: dashboard_klaim_business
-- Grid "Total Klaim Bisnis".
--
-- Asal: `RDB List/BrowseCaseClaim-SQL.xml`.
--
-- Nilai uang dipilih sebagai NUMBER apa adanya dan dibaca ke Go sebagai TEKS presisi penuh —
-- tidak pernah melewati float (`I-12`, `09-DATABASE-STRATEGY.md` §5).
--
-- Kolom kesembilan kueri lama — `(TTLAKSEP+TTLOS)+(TTLOS-TTLAKSEP)` — TIDAK dibawa. Tidak ada
-- keterangan di export yang menyatakan angka itu mewakili apa.
--
-- # Periode boleh KOSONG di sini, berbeda dari Dashboard Produktivitas
--
-- Kueri lamanya menyisipkan penyaring periode lewat `{ASIS:TempPeriode.AnaylstRemarks}`, dan
-- pola itu memang dapat berisi teks kosong — artinya seluruh periode. Dashboard Produktivitas
-- tidak dapat: kedelapan pencacahnya DIBANGUN dari perbandingan dua periode, sehingga
-- predikat kosong di sana akan menghasilkan SQL yang tidak sah.
--
-- Karena itu di sini periodenya bersyarat bendera, dan di sana wajib.
--
-- Bind: :1 bendera "seluruh periode" · :2..:3 periode · :4..:8 bendera lini bisnis
SELECT d.LGB_NOTE                                                  AS DIMENSION,
       COUNT(*)                                                    AS TOTAL_CLAIM,
       COUNT(CASE WHEN d.STSKLAIM = '1' THEN 1 END)                AS ACCEPTED_COUNT,
       COUNT(CASE WHEN d.STSKLAIM = '3' THEN 1 END)                AS REJECTED_COUNT,
       COUNT(CASE WHEN d.STSKLAIM NOT IN ('1','2','3') THEN 1 END) AS OUTSTANDING_COUNT,
       SUM(CASE WHEN d.STSKLAIM = '1' THEN d.TTLAKSEP ELSE 0 END)  AS ACCEPTED_AMOUNT,
       SUM(CASE WHEN d.STSKLAIM = '3' THEN d.TTLOS ELSE 0 END)     AS REJECTED_AMOUNT,
       SUM(CASE WHEN d.STSKLAIM NOT IN ('1','2','3')
                THEN d.TTLOS ELSE 0 END)                           AS OUTSTANDING_AMOUNT
  FROM POOLDATA.PEGA_DASHBOARDPNC d
 WHERE ( :1 = 1 OR (d.TGLKLAIM >= :2 AND d.TGLKLAIM < :3) )
   AND ( :4 = 1
      OR ( :5 = 1
           AND d.GROUP_PANEL IN ('003', '004', '006')
           AND d.GROUPBISNISID NOT IN ('09', '11', '16', '25') )
      OR ( :6 = 1 AND d.GROUPBISNISID IN ('09', '11', '16', '25') )
      OR ( :7 = 1 AND d.GROUP_PANEL = '002' )
      OR ( :8 = 1 AND d.GROUP_PANEL = '005' ) )
 GROUP BY d.LGB_NOTE
 ORDER BY d.LGB_NOTE ASC

-- name: dashboard_klaim_cause
-- Grid "Total Klaim CauseOfLoss".
--
-- Asal: `RDB List/BrowseCaseClaimPerCauseOfLoss-SQL.xml` — kueri yang sama dengan grid di
-- atas ditambah satu dimensi `COL_DESC`, yakni penyebab kerugian.
--
-- Bind: :1 bendera "seluruh periode" · :2..:3 periode · :4..:8 bendera lini bisnis
SELECT d.LGB_NOTE                                                  AS DIMENSION,
       d.COL_DESC                                                  AS CAUSE,
       COUNT(*)                                                    AS TOTAL_CLAIM,
       COUNT(CASE WHEN d.STSKLAIM = '1' THEN 1 END)                AS ACCEPTED_COUNT,
       COUNT(CASE WHEN d.STSKLAIM = '3' THEN 1 END)                AS REJECTED_COUNT,
       COUNT(CASE WHEN d.STSKLAIM NOT IN ('1','2','3') THEN 1 END) AS OUTSTANDING_COUNT,
       SUM(CASE WHEN d.STSKLAIM = '1' THEN d.TTLAKSEP ELSE 0 END)  AS ACCEPTED_AMOUNT,
       SUM(CASE WHEN d.STSKLAIM = '3' THEN d.TTLOS ELSE 0 END)     AS REJECTED_AMOUNT,
       SUM(CASE WHEN d.STSKLAIM NOT IN ('1','2','3')
                THEN d.TTLOS ELSE 0 END)                           AS OUTSTANDING_AMOUNT
  FROM POOLDATA.PEGA_DASHBOARDPNC d
 WHERE ( :1 = 1 OR (d.TGLKLAIM >= :2 AND d.TGLKLAIM < :3) )
   AND ( :4 = 1
      OR ( :5 = 1
           AND d.GROUP_PANEL IN ('003', '004', '006')
           AND d.GROUPBISNISID NOT IN ('09', '11', '16', '25') )
      OR ( :6 = 1 AND d.GROUPBISNISID IN ('09', '11', '16', '25') )
      OR ( :7 = 1 AND d.GROUP_PANEL = '002' )
      OR ( :8 = 1 AND d.GROUP_PANEL = '005' ) )
 GROUP BY d.LGB_NOTE, d.COL_DESC
 ORDER BY d.LGB_NOTE ASC, d.COL_DESC ASC

-- ============================================================================
-- SEMBILAN ANTREAN PERSETUJUAN
-- ============================================================================
--
-- Setiap antrean punya TIGA kueri dengan syarat `WHERE` yang SAMA: pencacah, daftar, dan
-- pernyataan keputusan. Ketiganya ditulis berdampingan supaya syaratnya dapat dibaca
-- bersamaan — dan uji `TestSyaratAntreanSamaPadaPencacahDanDaftar` menjaganya tetap sama.
--
-- Pernyataan keputusan SELALU ikut menyaring status menunggu di samping kuncinya. Tanpa itu,
-- penyelia yang membuka daftar lama akan MENIMPA keputusan orang lain, dan jumlah baris yang
-- berubah akan tetap sama dengan jumlah yang dipilih sehingga tidak ada yang tahu. Dengan itu,
-- selisihnya terlihat dan dinyatakan di layar.
--
-- Nilai `'0'` menunggu, `'1'` disetujui, `'2'` ditolak. Ketiganya terbukti dari export DAN
-- dari isi basis data — lihat catatan pada konstanta di decision.go.

-- name: count_bengkel
-- Asal: `RDB List/CountMasterBengkelManagee-SQL.xml`
--
-- Alias `City` untuk sebuah pencacah tidak dibawa (`D-19`), dan `ORDER BY 1 DESC` atas kueri
-- yang selalu mengembalikan tepat satu baris juga tidak — ia tidak berakibat apa pun.
SELECT COUNT(*)
  FROM POOLDATA.BENGKEL_HE
 WHERE TRIM(APPROVAL) = '0'

-- name: queue_bengkel
-- Antrean Master Bengkel.
SELECT TRIM(b.ID_BENGKEL) AS ROW_KEY,
       b.ID_BENGKEL       AS COL1,
       b.NAMA_BENGKEL     AS COL2,
       b.NAMA_CABANG      AS COL3,
       b.NAMA_KABUPATEN   AS COL4,
       b.ALM_BENGKEL      AS COL5
  FROM POOLDATA.BENGKEL_HE b
 WHERE TRIM(b.APPROVAL) = '0'
 ORDER BY b.ID_BENGKEL

-- name: decide_bengkel
-- Bind: :1 status baru · :2 alasan · :3 kunci baris
UPDATE POOLDATA.BENGKEL_HE
   SET APPROVAL = :1,
       ALASAN_STS_BGKL = :2
 WHERE TRIM(ID_BENGKEL) = :3
   AND TRIM(APPROVAL) = '0'

-- name: count_panel
-- Asal: `RDB List/CountMasterPanelManager-SQL.xml`
SELECT COUNT(*)
  FROM POOLDATA.PANEL_HE
 WHERE TRIM(APPROVAL) = '0'

-- name: queue_panel
-- Nama panel ada di kolom `NAME`, bukan `NAMA_PANEL` — nama yang terakhir itu justru milik
-- `SPAREPART_HE_VIN_KEY`. Keduanya terverifikasi dari kueri modul Master yang berjalan.
SELECT TRIM(p.ID_PANEL) AS ROW_KEY,
       p.ID_PANEL       AS COL1,
       p.NAME           AS COL2
  FROM POOLDATA.PANEL_HE p
 WHERE TRIM(p.APPROVAL) = '0'
 ORDER BY p.ID_PANEL

-- name: decide_panel
-- Bind: :1 status baru · :2 alasan · :3 kunci baris
UPDATE POOLDATA.PANEL_HE
   SET APPROVAL = :1,
       ALASAN_TOLAK = :2
 WHERE TRIM(ID_PANEL) = :3
   AND TRIM(APPROVAL) = '0'

-- name: count_nomor_rangka
-- Asal: `RDB List/CountNomorRangkaManager-SQL.xml`
--
-- Menunggu dinyatakan `STS_AKSEP IS NULL` di sini, BUKAN `= '0'`, dan itu bentuk sistem
-- lama yang dibawa apa adanya. Isi kolomnya pada 2026-09-28 membenarkannya: `1` (10 baris),
-- `2` (5), NULL (2) — tidak ada satu pun baris bernilai `0`.
SELECT COUNT(*)
  FROM POOLDATA.NOTIF_RANGKA_HE
 WHERE STS_AKSEP IS NULL

-- name: queue_nomor_rangka
-- Asal: `RDB List/GetDataKonfirmasiHE-SQL.xml`
--
-- Alias kueri lamanya menyesatkan seluruhnya — `noklaim AS "BRANCH_CODE"`,
-- `merk AS "BUSINESS_NAME"`, `no_rangka_bengkel AS "POLICY_NO"` — dan tidak dibawa (`D-19`).
-- Judul yang dibaca pengguna diambil dari section, bukan dari alias ini.
--
-- # Kuncinya GABUNGAN EMPAT KOLOM, dan itu bukan pilihan modul ini
--
-- `RDB List/SetStatusAksepNoRangka_SQL.xml` menyaring dengan `noklaim`, `model`,
-- `no_rangka_user`, DAN `no_rangka_bengkel` sekaligus — tabelnya memang tidak punya kunci
-- tunggal. Keempatnya karena itu dirangkai menjadi satu kunci baris, dan dipecah kembali saat
-- keputusan dituliskan.
SELECT n.NOKLAIM || '|' || n.MODEL || '|' ||
       n.NO_RANGKA_USER || '|' || n.NO_RANGKA_BENGKEL AS ROW_KEY,
       n.NOKLAIM                                      AS COL1,
       n.PENGIRIM                                     AS COL2,
       n.MERK                                         AS COL3,
       n.MODEL                                        AS COL4,
       n.TIPE                                         AS COL5,
       n.NO_RANGKA_USER                               AS COL6,
       n.NO_RANGKA_BENGKEL                            AS COL7
  FROM POOLDATA.NOTIF_RANGKA_HE n
 WHERE n.STS_AKSEP IS NULL
 ORDER BY n.NOKLAIM

-- name: decide_nomor_rangka
-- Asal: `RDB List/SetStatusAksepNoRangka_SQL.xml`, apa adanya kecuali penjaga status menunggu.
--
-- Bind: :1 status baru · :2 noklaim · :3 model · :4 no_rangka_user · :5 no_rangka_bengkel
UPDATE POOLDATA.NOTIF_RANGKA_HE
   SET STS_AKSEP = :1
 WHERE NOKLAIM = :2
   AND MODEL = :3
   AND NO_RANGKA_USER = :4
   AND NO_RANGKA_BENGKEL = :5
   AND STS_AKSEP IS NULL

-- name: count_sparepart
-- Asal: `RDB List/CountMasterSparepartManager-SQL.xml`
--
-- # Sumbernya RUSAK di basis data, dan itu bukan cacat modul ini
--
-- `POOLDATA.SPAREPART_HE` adalah VIEW berstatus INVALID saat diperiksa 2026-09-28. Ia
-- membaca `JSON_VALUE(a.JSONDATA, '$.…')` dari `POOLDATA.M_SPAREPART_HE`, sementara kolom
-- `JSONDATA` sudah tidak ada lagi di tabel itu — isinya kini 18 kolom bernama MERK, TYPE,
-- SUBTYPE, PARTNO, PARTNAME, dan seterusnya. Setiap pembacaan view-nya gagal ORA-04063.
--
-- Kuerinya TETAP ditulis terhadap view itu, bukan diakali: ia sumber yang sama dengan Pega
-- dan dengan modul Master Sparepart yang sudah ada. Yang dibutuhkan adalah DBA memperbaiki
-- view-nya; begitu itu terjadi, tab ini hidup tanpa satu baris kode pun berubah.
SELECT COUNT(*)
  FROM POOLDATA.SPAREPART_HE
 WHERE TRIM(APPROVAL) = '0'

-- name: queue_sparepart
SELECT TRIM(s.ID)      AS ROW_KEY,
       s.ID            AS COL1,
       s.NAMA_SPART    AS COL2,
       s.KATEGORI_SPART AS COL3,
       s.NO_SPART      AS COL4
  FROM POOLDATA.SPAREPART_HE s
 WHERE TRIM(s.APPROVAL) = '0'
 ORDER BY s.ID

-- name: decide_sparepart
-- Bind: :1 status baru · :2 petugas yang memutuskan · :3 kunci baris
--
-- `USER_UPDATE` ikut ditulis karena kolom itulah yang di sistem lama menyimpan siapa terakhir
-- menyentuh barisnya. Ia satu-satunya jejak pelaku pada antrean ini.
UPDATE POOLDATA.SPAREPART_HE
   SET APPROVAL = :1,
       USER_UPDATE = :2
 WHERE TRIM(ID) = :3
   AND TRIM(APPROVAL) = '0'

-- name: count_kategori_sparepart
-- Asal: `RDB List/CountMasterKatSparepartManager-SQL.xml`
SELECT COUNT(*)
  FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY
 WHERE TRIM(APPROVAL) = '0'

-- name: queue_kategori_sparepart
SELECT TRIM(k.PART_CATEGORY_ID) AS ROW_KEY,
       k.PART_CATEGORY_ID       AS COL1,
       k.PART_CATEGORY_NAME     AS COL2
  FROM POOLDATA.GCNM_M_SPAREPART_CATEGORY k
 WHERE TRIM(k.APPROVAL) = '0'
 ORDER BY k.PART_CATEGORY_ID

-- name: decide_kategori_sparepart
-- Bind: :1 status baru · :2 kunci baris
UPDATE POOLDATA.GCNM_M_SPAREPART_CATEGORY
   SET APPROVAL = :1
 WHERE TRIM(PART_CATEGORY_ID) = :2
   AND TRIM(APPROVAL) = '0'

-- name: count_tipe_sparepart
-- Asal: `RDB List/CountMasterTipeSparepartManager-SQL.xml`
SELECT COUNT(*)
  FROM POOLDATA.GCNM_M_SPAREPART_TYPE
 WHERE TRIM(APPROVAL) = '0'

-- name: queue_tipe_sparepart
SELECT TRIM(t.PART_SECTION_ID) AS ROW_KEY,
       t.PART_SECTION_ID       AS COL1,
       t.PART_SECTION_NAME     AS COL2,
       t.PART_CATEGORY_ID      AS COL3
  FROM POOLDATA.GCNM_M_SPAREPART_TYPE t
 WHERE TRIM(t.APPROVAL) = '0'
 ORDER BY t.PART_SECTION_ID

-- name: decide_tipe_sparepart
-- Bind: :1 status baru · :2 kunci baris
UPDATE POOLDATA.GCNM_M_SPAREPART_TYPE
   SET APPROVAL = :1
 WHERE TRIM(PART_SECTION_ID) = :2
   AND TRIM(APPROVAL) = '0'

-- name: count_grouping_sparepart
-- Asal: `RDB List/CountMasterGrupSparepartManager-SQL.xml`
--
-- Kueri lamanya menggabungkan `SPAREPART_HE_VIN_KEY A` dengan `SPAREPART_HE_VIN_GROUP B`
-- pada `A.ID=B.ID` tanpa memilih satu kolom pun dari B. Gabungan itu DIBAWA karena ia
-- menyaring: baris yang tidak punya pasangan di B tidak terhitung.
SELECT COUNT(*)
  FROM POOLDATA.SPAREPART_HE_VIN_KEY a
  JOIN POOLDATA.SPAREPART_HE_VIN_GROUP b
    ON a.ID = b.ID
 WHERE TRIM(a.APPROVAL) = '0'

-- name: queue_grouping_sparepart
-- Nomor rangka ada di tabel GROUP (alias `b`), sedangkan nama dan nomor sparepart ada di
-- tabel KEY (alias `a`). Pembagian itu terverifikasi dari kueri modul Master Grouping
-- Sparepart yang berjalan, dan mudah tertukar karena keduanya bernama mirip.
SELECT TRIM(a.ID)  AS ROW_KEY,
       a.ID        AS COL1,
       b.NO_RANGKA AS COL2,
       a.NAMA_PART AS COL3,
       a.NO_PART   AS COL4
  FROM POOLDATA.SPAREPART_HE_VIN_KEY a
  JOIN POOLDATA.SPAREPART_HE_VIN_GROUP b
    ON a.ID = b.ID
 WHERE TRIM(a.APPROVAL) = '0'
 ORDER BY a.ID

-- name: decide_grouping_sparepart
-- Bind: :1 status baru · :2 kunci baris
UPDATE POOLDATA.SPAREPART_HE_VIN_KEY
   SET APPROVAL = :1
 WHERE TRIM(ID) = :2
   AND TRIM(APPROVAL) = '0'

-- name: count_payment_akseptasi
-- Asal: `RDB List/CountPaymentKlaimAkseptasiManager-SQL.xml`
--
-- # Syaratnya DIPERSEMPIT, dan alasannya ada di pernyataan keputusannya sendiri
--
-- Kueri lamanya berbunyi `STSAPP NOT IN ('1','2') OR STSAPP IS NULL`, sementara
-- `SaveApproveAkseptasiPaymentLeader_Sql` — pernyataan yang menuliskan keputusannya —
-- menyaring `STSAPP = '0'`. Keduanya tidak sama: baris ber-STSAPP NULL akan MASUK antrean
-- tetapi TIDAK PERNAH dapat diputuskan, dan penyelia tidak akan tahu kenapa.
--
-- Yang dipakai di sini adalah syarat pernyataan keputusannya, sehingga daftar dan keputusan
-- selalu sepakat. Isi kolomnya pada 2026-09-28 membuat selisih itu nol dalam praktik: `1`
-- (42 baris), `2` (33), `0` (8), dan TIDAK ADA satu pun NULL.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIM_AKSEPTASI_CHECKER
 WHERE TRIM(STSAPP) = '0'

-- name: queue_payment_akseptasi
-- Asal: `RDB List/GetDataAkseptasiLeaderKlaimNonMBU-SQL.xml`
--
-- Subkueri nomor klaim dibawa apa adanya — ia satu-satunya cara tabel checker mengetahui
-- nomor klaimnya, dan menggantinya dengan gabungan akan mengubah baris yang tampil bila
-- `T_CLAIM_PNC` tidak punya pasangannya.
--
-- `TO_CHAR(tglinput,'dd/mm/yyyy')` kueri lama TIDAK dibawa: pemformatan tanggal dilakukan di
-- Go (`08-TECHNICAL-STRATEGY.md` §4.3), dan mengembalikannya sebagai teks membuat
-- pengurutannya menjadi pengurutan teks.
SELECT TRIM(a.NOAKSEPTASI) AS ROW_KEY,
       (SELECT x.CLAIMNO
          FROM POOLDATA.T_CLAIM_PNC x
         WHERE x.CLAIMID = a.CLAIMID) AS COL1,
       a.NOAKSEPTASI                  AS COL2,
       a.PIC                          AS COL3,
       a.TGLINPUT                     AS COL4
  FROM POOLDATA.T_CLAIM_AKSEPTASI_CHECKER a
 WHERE TRIM(a.STSAPP) = '0'
 ORDER BY a.TGLINPUT DESC, a.NOAKSEPTASI

-- name: decide_payment_akseptasi
-- Asal: `RDB List/SaveApproveAkseptasiPaymentLeader_Sql_SQL.xml`
--
-- # Hanya jalur TOLAK yang memakai pernyataan ini
--
-- Di Pega, `Activity/SaveApprovalAkseptasiPaymentLeader_Act:89539` memanggil
-- `TransferToKasir_act_Leader` pada precondition `AcceptanceStatus=="1"` — yakni hanya pada
-- PERSETUJUAN. Rantai itu belum lengkap, sehingga jalur setuju ditahan di lapisan domain.
-- Lihat catatan panjang pada tab Payment Klaim Akseptasi di tab.go.
--
-- `TGLAPPROVE` memakai CURRENT_TIMESTAMP, bukan `sysdate` seperti kueri lama: `sysdate`
-- dilarang demi portabilitas (`08-TECHNICAL-STRATEGY.md` §4.3, `09-DATABASE-STRATEGY.md` §4).
--
-- # `PICNOTES` TIDAK ikut ditulis, dan itu keputusan
--
-- Pernyataan lamanya menulis DUA kolom catatan dari DUA isian layar yang berbeda:
-- `NOTEAPPROVE` dari `RemarkManager`, dan `PICNOTES` dari `RemarkAcceptedLeader` — yang
-- terakhir kemudian dibaca `TransferToKasir_act_Leader` sebagai `Local.Keteranganatasan`.
--
-- Layar ini punya SATU isian alasan. Menuliskannya ke kedua kolom berarti mengarang isi
-- kolom kedua, dan menimpa apa pun yang sudah ada di sana. Yang ditulis karena itu hanya
-- `NOTEAPPROVE` — kolom yang namanya memang menyebut catatan persetujuan.
--
-- Bind: :1 status baru · :2 catatan atasan · :3 nomor akseptasi
UPDATE POOLDATA.T_CLAIM_AKSEPTASI_CHECKER
   SET STSAPP = :1,
       NOTEAPPROVE = :2,
       TGLAPPROVE = CURRENT_TIMESTAMP
 WHERE TRIM(NOAKSEPTASI) = :3
   AND TRIM(STSAPP) = '0'

-- name: count_penolakan_klaim
-- Asal: `RDB List/CountCheckerRejectNotes-SQL.xml`
SELECT COUNT(*)
  FROM POOLDATA.MST_PENOLAKAN_KLAIM_2
 WHERE TRIM(STATUS) = '0'

-- name: queue_penolakan_klaim
-- Asal: `RDB List/BrowseStatusPenolakanKlaim2-SQL.xml`
--
-- Kolom terjemahan status kueri lama (`case when A.STATUS='1' then 'APPROVED' …`) TIDAK
-- dibawa: daftar ini hanya memuat baris berstatus menunggu, sehingga kolom itu akan berbunyi
-- "MENUNGGU" pada setiap baris.
SELECT TRIM(a.ID_ST) AS ROW_KEY,
       a.ID_ST       AS COL1,
       a.NOTE_ST     AS COL2,
       a.NOTE_ND     AS COL3,
       a.USER_INPUT  AS COL4
  FROM POOLDATA.MST_PENOLAKAN_KLAIM_2 a
 WHERE TRIM(a.STATUS) = '0'
 ORDER BY a.ID_ST ASC

-- name: decide_penolakan_klaim
-- Bind: :1 status baru · :2 petugas yang memutuskan · :3 catatan · :4 kunci baris
--
-- `APPROVEBY` dan `TANGGAL_APPROVE` diisi di sini, dan itu memang tujuannya:
-- `internal/masterpenolakan/masterpenolakan.go` mencatat kolom `APPROVEBY` sebagai "diisi
-- layar Inbox Manager, baca-saja di sini". Layar itu adalah layar ini.
UPDATE POOLDATA.MST_PENOLAKAN_KLAIM_2
   SET STATUS = :1,
       APPROVEBY = :2,
       TANGGAL_APPROVE = CURRENT_TIMESTAMP,
       NOTEAPPROVED = :3
 WHERE TRIM(ID_ST) = :4
   AND TRIM(STATUS) = '0'
