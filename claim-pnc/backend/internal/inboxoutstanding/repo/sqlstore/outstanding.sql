-- Kueri layar **My Inbox** (`Harness/InboxRegister_Harness-Harness.xml`, MENU_ID 51).
--
-- ============================================================================
-- KUERI INI SEMPAT DIAMBIL DARI LAYAR YANG SALAH
-- ============================================================================
--
-- Semula dibangun dari `RDB List/BrowseInboxOutstanding1-SQL.xml` dan
-- `Activity/InboxOutstanding_Act-Act.xml`, karena keduanya bernama "Outstanding" dan
-- section rujukan memang berjudul "Inbox Outstanding" di dalamnya (`:2150`).
--
-- Penelusuran membuktikan keduanya **tidak pernah dipanggil** harness maupun section ini:
--
--     BrowseInboxOutstanding1  <- SetDashboardClaim, ExportOutstanding, AlertPendingPLADLA
--     InboxOutstanding_Act     <- idem
--     InboxRegister_Section    -> InboxRegister_RD, SetClaimPNC   <- yang BENAR
--
-- Akibatnya layar ini sempat berperilaku sebagai "semua klaim outstanding, disaring lini
-- bisnis" — yaitu layar Dashboard — bukan sebagai "pekerjaan saya".
--
-- ============================================================================
-- SUMBER YANG BENAR: Report Definition/InboxRegister_RD-RD.xml
-- ============================================================================
--
-- RD itu menyaring LIMA syarat, seluruhnya ber-AND (`pyFilterLogic: A AND B AND D AND C AND E`):
--
--     A  Operator ID   =   Param.assign          <- PXASSIGNEDOPERATORID
--     B  GroupPanel    =   Param.panel           <- GROUPPANEL_1
--     C  RCV_ID        =   Param.idRCV           <- PNCCASEID
--     D  Work Status  !=   "Resolved-Completed"  <- PYSTATUSWORK
--     E  Work Status  !=   "Resolved-Rejected"   <- PYSTATUSWORK
--
-- **Syarat A adalah yang membuatnya "My" Inbox.** `Section/InboxRegister_Section-Section.xml`
-- mengisi parameternya dengan `OperatorID.pyUserIdentifier` — pengguna yang sedang masuk.
--
-- `PXASSIGNEDOPERATORID` terisi pada 1.004 dari 1.012 baris, 65 operator berbeda.
--
-- ============================================================================
-- TIGA PENYARING YANG DIBUANG DARI DAFTAR — DAN KE MANA PERGINYA
-- ============================================================================
--
-- Ketiganya berasal dari `BrowseInboxOutstanding1`, dan RD tidak memilikinya:
--
--     AND b.pxflowname  NOT IN ('FixCorrespondence', 'Register_Flow_1')
--     AND b.PXTASKLABEL NOT IN ('FixCorrespondence')
--     AND a.branchname != 'ASNET'
--
-- Membawanya ke DAFTAR berarti menyaring lebih ketat daripada layar aslinya — klaim yang
-- di Pega terlihat akan hilang di sini, tanpa satu pun galat yang menandainya.
--
-- Tetapi ketiganya **bukan milik siapa-siapa**: ketiganya milik **EXPORT**. Penelusuran
-- `RDB List/ExportDataDetailKlaim-SQL.xml` menemukan kedua yang pertama apa adanya, dan
-- yang ketiga di dalam cakupan NONMBU. Lihat bagian EXPORT di bawah.
--
-- Batas data per lini bisnis juga dibuang DARI DAFTAR, dan alasannya sama: RD
-- memperlakukan panel sebagai PARAMETER, bukan sebagai pita tetap per pengguna. Pada
-- export ia justru penentu cakupan.
--
-- ============================================================================
-- PEMETAAN KOLOM — alias lama TIDAK dibawa
-- ============================================================================
--
-- Judul kolom tetap dari `Section/InboxRegister_Section-Section.xml` (`D-13`); bagian itu
-- sejak awal bersumber benar.
--
--   kolom layar      properti section   kolom di T_CLAIMLIST_ADMIN
--   ---------------  -----------------  --------------------------
--   Claim no         .ClaimNo           PYID
--   Policy no        .District (!)      POLICYNO
--   Insured name     .CountryID (!)     QQNAME
--   Business Name    .Country (!)       BUSINESSNAME
--   Business source  .CityID (!)        SOBNAME
--   Branch name      .City (!)          BRANCHNAME
--   Admin name       .ReporterName      PXCREATEOPERATOR
--   Register Date    .pxCreateDateTime  REGISTERDATE_1
--   Date of loss     .DateOfLoss        DATEOFLOSS_1
--   Aging            .DateForAging      AGING
--   Claim status     .StatusClaim       PYSTATUSWORK
--   Status ASM       .LSC_ID            STATUSLOCK_1   (kosong di seluruh baris)
--   ASM PIC          —                  USERTEKNIS_1
--   Total Aging      —                  dihitung dari REGISTERDATE_1
--
-- Tanda (!) menandai alias yang artinya BERLAWANAN dengan isinya.
--
-- `AGING` dibaca APA ADANYA, tidak dihitung ulang — `DATEFORAGING_1` tampak menjadi
-- acuannya tetapi artinya belum dipastikan.
--
-- ============================================================================
-- PENANDA PARAMETER
-- ============================================================================
--
-- Tiap kemunculan bernomor SENDIRI. Keduanya sempat memakai nomor yang sama
-- (`:5 IS NULL OR … = :5`) dan Oracle menolaknya dengan **ORA-01008**: driver mengikat
-- argumen menurut urutan KEMUNCULAN penanda, bukan menurut nomornya.
--
--     :1              PXASSIGNEDOPERATORID — identitas login, WAJIB
--     :2              PXASSIGNEDOPERATORID — identitas LAMA orang yang sama
--     :3              penentu apakah pencarian aktif
--     :4 :5 :6        pola pencarian
--     :7 :8           panel        (NULL = tidak menyaring)
--     :9 :10          RCV          (NULL = tidak menyaring)
--     :11 :12         tahap        (NULL = tidak menyaring)
--     :13 :14         cabang       (NULL = tidak menyaring)
--     :15 :16 :17     status dokumen (NULL = tidak menyaring) — TIDAK dipakai ringkasan
--     :18 :19         offset & limit — hanya pada my_inbox_list
--
-- ============================================================================
-- DUA IDENTITAS UNTUK SATU ORANG
-- ============================================================================
--
-- `BrowseInboxPicTeknik-SQL.xml:21` menyaring **DAFTAR**, bukan satu nilai:
--
--     AND b.pxassignedoperatorid IN {ASIS:TempOperator.CityID}
--
-- Isinya dirangkai `Activity/SetClaimPNC-Act.xml:554`:
--
--     "('" + TempOPID.pxResults(1).City + "', '" + OperatorID.pyUserIdentifier + "')"
--
-- Yaitu **identitas lama** dan **identitas sekarang** orang yang sama. Yang lama dibaca
-- `RDB List/GetOperatorID-SQL.xml` dari `POOLDATA.T_ACCESS_GROUP_PNC` — dan kolom
-- penampungnya dialias `"City"`, alias yang tidak ada hubungannya dengan isinya.
--
-- **Tanpa ini, login HCC yang berbentuk email tidak cocok dengan satu baris pun.** Klaim
-- warisan tertugas ke nama operator Pega, bukan ke email. Terukur pada data ASM:
-- 20 dari 29 operator punya identitas lama yang berbeda, dan salah satunya memegang
-- **79 klaim berjalan yang seluruhnya tersimpan di identitas lamanya**.
--
-- Kegagalannya tidak menghasilkan galat: layarnya tampil rapi dan kosong.

-- name: my_inbox_list
-- Satu halaman pekerjaan milik pemanggil.
SELECT k.PZINSKEY,
       k.PYID,
       k.POLICYNO,
       k.QQNAME,
       k.BUSINESSNAME,
       k.SOBNAME,
       k.BRANCHNAME,
       k.GROUPPANEL_1,
       k.PNCCASEID,
       k.REGISTERDATE_1,
       k.PXCREATEDATETIME,
       k.DATEOFLOSS_1,
       k.REPORTDATE_1,
       k.AGING,
       k.PYSTATUSWORK,
       k.STATUSLOCK_1,
       k.STATUSPROGRESS1,
       k.USERTEKNIS_1,
       k.PXCREATEOPERATOR,
       k.PXTASKLABEL,
       k.PXASSIGNEDOPERATORID,
       k.DOKUMENLENGKAP_1
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
 WHERE k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND UPPER(TRIM(k.PXASSIGNEDOPERATORID)) IN (:1, :2)
   AND k.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (:3 IS NULL
        OR UPPER(k.PYID) LIKE :4 ESCAPE '\'
        OR UPPER(k.POLICYNO) LIKE :5 ESCAPE '\'
        OR UPPER(k.USERTEKNIS_1) LIKE :6 ESCAPE '\')
   AND (:7 IS NULL OR UPPER(TRIM(k.GROUPPANEL_1)) = :8)
   AND (:9 IS NULL OR UPPER(TRIM(k.PNCCASEID)) = :10)
   AND (:11 IS NULL OR UPPER(TRIM(k.PXTASKLABEL)) = :12)
   AND (:13 IS NULL OR UPPER(TRIM(k.BRANCHNAME)) = :14)
   AND (:15 IS NULL
        OR (:16 = 'LENGKAP' AND k.DOKUMENLENGKAP_1 = '1')
        OR (:17 = 'BELUM'
            AND (k.DOKUMENLENGKAP_1 = '0' OR k.DOKUMENLENGKAP_1 IS NULL)))
 ORDER BY k.PXCREATEDATETIME DESC, k.PZINSKEY
OFFSET :18 ROWS FETCH NEXT :19 ROWS ONLY

-- ============================================================================
-- EXPORT — kueri yang BERBEDA, bukan daftar yang dipanggil ulang
-- ============================================================================
--
-- Sumbernya `RDB List/ExportDataDetailKlaim-SQL.xml`, dan perbedaannya dengan daftar
-- BUKAN perkara detail:
--
--     WHERE c.pxobjclass = 'ASM-FW-GCNMFW-Work-PNC'
--       AND c.pystatuswork NOT IN ('Resolved-Completed', 'Resolved-Rejected')
--       AND b.pxflowname  NOT IN ('FixCorrespondence', 'Register_Flow_1')
--       AND b.PXTASKLABEL NOT IN ('FixCorrespondence')
--       AND (c.ISPENDINGCLOSE != 'true' or c.ISPENDINGCLOSE IS NULL)
--       {ASIS:TempBisnis.BUSINESSTYPE}   <- cakupan lini bisnis
--       {ASIS:TempBisnis.REGISTERID}     <- rentang tanggal
--
-- **Tidak ada satu pun penyaring operator di sana.** Export sebelumnya memanggil ulang
-- daftar, sehingga ikut terkena `PXASSIGNEDOPERATORID` dan mengembalikan berkas kosong
-- bagi petugas yang inbox-nya kosong — padahal di Pega berkasnya tetap berisi.
--
-- Cakupan lini bisnis dipilih `Activity/ExportDataDetailKlaim-Act.xml` dari
-- `OperatorID.pyPosition`; di sini dari `M_LOGIN_PNC.LINE_BUSINESS`:
--
--     PA       and c.GROUPPANEL_1 in ('002')
--     TRAVEL   and c.GROUPPANEL_1 in ('005')
--     BONDING  and f.businessgroupid IN ('10008','10010','10015','10023')
--     NONMBU   and c.GROUPPANEL_1 in ('003','004','006')
--              AND f.businessgroupid NOT IN (...bonding...)
--              and c.branchname!='ASNET' AND c.USERTEKNIS_1 is not null
--     lainnya  ""   <- tanpa cakupan
--
-- Join `business`/`businessgroup` TIDAK dibawa: `BUSINESSGROUPID` sudah ada sebagai kolom
-- di `T_CLAIMLIST_ADMIN`, terisi pada seluruh 1.012 baris dengan 10 nilai berbeda.
--
-- Keempat kode grup bonding masih TERTANAM di sini, sama seperti di Pega. Memindahkannya
-- ke master data adalah pekerjaan `F-4` (`D-15`) dan di luar lingkup layar ini; yang
-- dilakukan sekarang hanyalah menyebut asalnya supaya ia tidak tampak sebagai angka yang
-- muncul begitu saja.
--
-- Hitungan terhadap 1.012 baris produksi: dasar 862 · NONMBU 354 · PA 1 · TRAVEL 0 ·
-- BONDING 0. `ISPENDINGCLOSE` NULL pada seluruh baris, jadi penyaringnya belum pernah
-- menggigit; ia tetap dibawa supaya setara saat kolomnya mulai terisi.
--
-- # Batas atas tanggal memakai `<`, bukan `TRUNC(...) <=`
--
-- Pega membandingkan `trunc(c.pxcreatedatetime) <= to_date(ENDDATE)`. `TRUNC` pada kolom
-- mematikan index-nya. `ts < awal hari BERIKUTNYA` memberi himpunan baris yang sama persis
-- dan tetap dapat memakai index; pemanggil yang menambahkan satu harinya.
--
-- # Penanda parameter — bernomor sendiri, sama seperti daftar
--
--     :1 :2 :3 :4      lini bisnis, diuji empat kali
--     :5               lini bisnis, cabang "tanpa cakupan"
--     :6 :7            tanggal mulai (NULL = tidak menyaring)
--     :8 :9            tanggal akhir, EKSKLUSIF (NULL = tidak menyaring)
--     :10 :11          offset & limit — hanya pada my_inbox_export

-- name: my_inbox_export
-- Sekumpulan baris untuk unduhan CSV. TANPA penyaring pemilik pekerjaan.
SELECT k.PZINSKEY,
       k.PYID,
       k.POLICYNO,
       k.QQNAME,
       k.BUSINESSNAME,
       k.SOBNAME,
       k.BRANCHNAME,
       k.GROUPPANEL_1,
       k.PNCCASEID,
       k.REGISTERDATE_1,
       k.PXCREATEDATETIME,
       k.DATEOFLOSS_1,
       k.REPORTDATE_1,
       k.AGING,
       k.PYSTATUSWORK,
       k.STATUSLOCK_1,
       k.STATUSPROGRESS1,
       k.USERTEKNIS_1,
       k.PXCREATEOPERATOR,
       k.PXTASKLABEL,
       k.PXASSIGNEDOPERATORID,
       k.DOKUMENLENGKAP_1
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
 WHERE k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND k.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (k.PXFLOWNAME IS NULL
        OR k.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1'))
   AND (k.PXTASKLABEL IS NULL
        OR k.PXTASKLABEL NOT IN ('FixCorrespondence'))
   AND (k.ISPENDINGCLOSE IS NULL OR k.ISPENDINGCLOSE <> 'true')
   AND ((:1 = 'PA' AND k.GROUPPANEL_1 IN ('002'))
        OR (:2 = 'TRAVEL' AND k.GROUPPANEL_1 IN ('005'))
        OR (:3 = 'BONDING'
            AND k.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023'))
        OR (:4 = 'NONMBU'
            AND k.GROUPPANEL_1 IN ('003', '004', '006')
            AND (k.BUSINESSGROUPID IS NULL
                 OR k.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023'))
            AND (k.BRANCHNAME IS NULL OR k.BRANCHNAME <> 'ASNET')
            AND k.USERTEKNIS_1 IS NOT NULL)
        OR COALESCE(:5, '-') NOT IN ('PA', 'TRAVEL', 'BONDING', 'NONMBU'))
   AND (:6 IS NULL OR k.PXCREATEDATETIME >= :7)
   AND (:8 IS NULL OR k.PXCREATEDATETIME < :9)
 ORDER BY k.PXCREATEDATETIME DESC, k.PZINSKEY
OFFSET :10 ROWS FETCH NEXT :11 ROWS ONLY

-- name: my_inbox_export_count
-- Menghitung SELURUH baris yang akan terbawa unduhan.
--
-- Syarat WHERE-nya wajib sama persis dengan my_inbox_export, dengan alasan yang sama
-- seperti pada pasangan daftar: penghentian pengambilan bertumpu pada angka ini.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
 WHERE k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND k.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (k.PXFLOWNAME IS NULL
        OR k.PXFLOWNAME NOT IN ('FixCorrespondence', 'Register_Flow_1'))
   AND (k.PXTASKLABEL IS NULL
        OR k.PXTASKLABEL NOT IN ('FixCorrespondence'))
   AND (k.ISPENDINGCLOSE IS NULL OR k.ISPENDINGCLOSE <> 'true')
   AND ((:1 = 'PA' AND k.GROUPPANEL_1 IN ('002'))
        OR (:2 = 'TRAVEL' AND k.GROUPPANEL_1 IN ('005'))
        OR (:3 = 'BONDING'
            AND k.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023'))
        OR (:4 = 'NONMBU'
            AND k.GROUPPANEL_1 IN ('003', '004', '006')
            AND (k.BUSINESSGROUPID IS NULL
                 OR k.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023'))
            AND (k.BRANCHNAME IS NULL OR k.BRANCHNAME <> 'ASNET')
            AND k.USERTEKNIS_1 IS NOT NULL)
        OR COALESCE(:5, '-') NOT IN ('PA', 'TRAVEL', 'BONDING', 'NONMBU'))
   AND (:6 IS NULL OR k.PXCREATEDATETIME >= :7)
   AND (:8 IS NULL OR k.PXCREATEDATETIME < :9)

-- name: legacy_operator_for
-- Identitas LAMA seorang petugas — padanan `RDB List/GetOperatorID-SQL.xml`.
--
-- Aslinya berbunyi:
--
--     SELECT OLD_OPERATOR_ID AS "City" FROM POOLDATA.T_ACCESS_GROUP_PNC
--      WHERE OPERATOR_ID = {OperatorID.pyUserIdentifier} AND STS_AKTIF = '1'
--
-- Fragmen `{Asis:TempOperator.AlasanKlaim}` yang mengekornya TIDAK pernah diisi pada jalur
-- ini, sehingga tidak dibawa.
--
-- # Kenapa MAX, bukan baris pertama
--
-- Pega memakai `pxResults(1)` — baris PERTAMA, yang urutannya tidak ditentukan kueri mana
-- pun. Tabelnya memuat satu baris per ACCESS_GROUP, sehingga satu orang punya banyak baris:
-- 208 baris untuk 29 operator, dengan 41 grup akses.
--
-- Yang membuat `pxResults(1)` aman ternyata bukan urutannya melainkan datanya: **setiap
-- dari 29 operator hanya punya SATU `OLD_OPERATOR_ID` yang berbeda** — diperiksa langsung
-- ke basis data. `MAX` karena itu mengembalikan nilai yang sama persis, dan ia
-- deterministik sedangkan `pxResults(1)` tidak.
--
-- Bila kelak ada operator dengan lebih dari satu identitas lama, keduanya akan berbeda —
-- dan itu keadaan yang memang harus ditanyakan, bukan dipilih diam-diam.
SELECT MAX(UPPER(TRIM(g.OLD_OPERATOR_ID)))
  FROM POOLDATA.T_ACCESS_GROUP_PNC g
 WHERE UPPER(TRIM(g.OPERATOR_ID)) = :1
   AND g.STS_AKTIF = '1'

-- name: line_business_for
-- Lini bisnis seorang petugas — penentu cakupan export.
--
-- Padanan `OperatorID.pyPosition` pada `ExportDataDetailKlaim-Act`. Petugas tanpa baris
-- di sini BUKAN galat: ia diperlakukan sebagai tanpa cakupan, persis seperti Pega
-- memperlakukan pyPosition yang tidak cocok satu pun.
--
-- Tabelnya dimiliki modul Login; modul ini hanya MEMBACA satu kolom (`P-1`).
SELECT p.LINE_BUSINESS
  FROM POOLDATA.M_LOGIN_PNC p
 WHERE UPPER(TRIM(p.LOGIN_ID)) = :1

-- name: my_inbox_count
-- Menghitung SELURUH pekerjaan pemanggil yang cocok, bukan baris pada halaman ini.
--
-- Syarat WHERE-nya wajib sama persis dengan my_inbox_list. Bila keduanya menyimpang,
-- pengguna melihat "247 baris cocok" lalu menemukan jumlah yang berbeda saat menelusuri
-- halamannya — dan tidak ada galat yang muncul.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
 WHERE k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND UPPER(TRIM(k.PXASSIGNEDOPERATORID)) IN (:1, :2)
   AND k.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (:3 IS NULL
        OR UPPER(k.PYID) LIKE :4 ESCAPE '\'
        OR UPPER(k.POLICYNO) LIKE :5 ESCAPE '\'
        OR UPPER(k.USERTEKNIS_1) LIKE :6 ESCAPE '\')
   AND (:7 IS NULL OR UPPER(TRIM(k.GROUPPANEL_1)) = :8)
   AND (:9 IS NULL OR UPPER(TRIM(k.PNCCASEID)) = :10)
   AND (:11 IS NULL OR UPPER(TRIM(k.PXTASKLABEL)) = :12)
   AND (:13 IS NULL OR UPPER(TRIM(k.BRANCHNAME)) = :14)
   AND (:15 IS NULL
        OR (:16 = 'LENGKAP' AND k.DOKUMENLENGKAP_1 = '1')
        OR (:17 = 'BELUM'
            AND (k.DOKUMENLENGKAP_1 = '0' OR k.DOKUMENLENGKAP_1 IS NULL)))

-- name: my_inbox_document_status
-- Ringkasan isi inbox per status kelengkapan dokumen — sumber donut.
--
-- # Kenapa satu kueri ber-CASE, bukan dua kueri COUNT
--
-- Dua kueri terpisah membaca tabel dua kali dan, yang lebih buruk, dapat melihat keadaan
-- yang BERBEDA bila ada yang mengubah data di antaranya — sehingga jumlah irisan tidak
-- sama dengan totalnya, tanpa satu pun galat yang menandainya.
--
-- # Penyaring status TIDAK ikut di sini
--
-- Ringkasan wajib tetap memuat SELURUH status supaya irisan yang sedang dipilih tetap
-- terlihat dan dapat dibatalkan. Menghormati penyaringnya akan membuat donut menyusut
-- menjadi satu irisan begitu pengguna mengekliknya, dan tidak ada jalan kembali selain
-- memuat ulang halaman. Karena itu penomoran di sini berhenti di :14.
--
-- # Aturannya disalin dari fragmen Pega
--
--     lengkap   AND A.DOKUMENLENGKAP_1 = '1'               (SetClaimPNC-Act.xml:1799)
--     belum     AND (A.DOKUMENLENGKAP_1 = '0' OR IS NULL)  (SetClaimPNC-Act.xml:1630)
--
-- Syarat `AND (A.TKA_1 != '1' OR A.TKA_1 IS NULL)` yang mengekor keduanya **tidak dibawa**:
-- kolom `TKA_1` TIDAK ADA di `T_CLAIMLIST_ADMIN`. Akibatnya klaim TKA — yang di Pega
-- dikeluarkan dari kedua tab dan punya tabnya sendiri — di sini ikut terhitung "belum
-- lengkap". Selisih yang disadari, dan hilang sendiri begitu kolomnya ada.
--
-- Keadaan data 2026-09-24: `DOKUMENLENGKAP_1` NULL pada SELURUH 1.012 baris, sehingga
-- donutnya satu irisan utuh sampai sistem lama mulai mengisinya.
SELECT SUM(CASE WHEN k.DOKUMENLENGKAP_1 = '1' THEN 1 ELSE 0 END) AS LENGKAP,
       SUM(CASE WHEN k.DOKUMENLENGKAP_1 = '0' OR k.DOKUMENLENGKAP_1 IS NULL
                THEN 1 ELSE 0 END) AS BELUM,
       COUNT(*) AS TOTAL
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
 WHERE k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND UPPER(TRIM(k.PXASSIGNEDOPERATORID)) IN (:1, :2)
   AND k.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (:3 IS NULL
        OR UPPER(k.PYID) LIKE :4 ESCAPE '\'
        OR UPPER(k.POLICYNO) LIKE :5 ESCAPE '\'
        OR UPPER(k.USERTEKNIS_1) LIKE :6 ESCAPE '\')
   AND (:7 IS NULL OR UPPER(TRIM(k.GROUPPANEL_1)) = :8)
   AND (:9 IS NULL OR UPPER(TRIM(k.PNCCASEID)) = :10)
   AND (:11 IS NULL OR UPPER(TRIM(k.PXTASKLABEL)) = :12)
   AND (:13 IS NULL OR UPPER(TRIM(k.BRANCHNAME)) = :14)
