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
--     :1              pemilik pekerjaan — identitas login, WAJIB
--     :2              pemilik pekerjaan — identitas LAMA orang yang sama
--
--   Kolom yang dibandingkan BERBEDA menurut kuerinya: `PXASSIGNEDOPERATORID` pada inbox
--   berjalan, `USERTEKNIS_1` pada Temporary Close. Lihat alasannya di bagian kueri itu.
--     :3              penentu apakah pencarian aktif
--     :4 :5 :6        pola pencarian
--     :7 :8           panel        (NULL = tidak menyaring)
--     :9 :10          RCV          (NULL = tidak menyaring)
--     :11 :12         tahap        (NULL = tidak menyaring)
--     :13 :14         cabang       (NULL = tidak menyaring)
--     :15 … :20       tab yang dipilih (NULL = tidak menyaring) — TIDAK dipakai ringkasan
--     :21 :22         offset & limit — hanya pada kueri daftar
--
--   Keenam penanda tab memuat NILAI YANG SAMA; yang membedakan hanyalah cabang `OR` mana
--   yang cocok. Bentuk itu dipilih karena go-ora mengikat menurut urutan kemunculan —
--   satu penanda dipakai berulang akan menggeser seluruh argumen sesudahnya.
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

-- ============================================================================
-- KLAIM YANG TERTUNDA TUTUP DIBUANG — dan itu BUKAN penyaring tab
-- ============================================================================
--
--     AND (k.ISPENDINGCLOSE <> 'true' OR k.ISPENDINGCLOSE IS NULL)
--
-- Di Pega ini bukan bagian kueri melainkan potongan yang ditempelkan ke `TempSearch.source`
-- oleh `Activity/SetClaimPNC-Act.xml:1284`, dan ia berlaku pada **setiap tab kecuali
-- Temporary Close**.
--
-- Bahwa pengecualiannya hanya Temporary Close terbaca dari kode aksi preconditionnya,
-- bukan dari dugaan:
--
--     :1335  pyStepsPreCondParamsWhen       TempVisibility.Email==8
--     :1342  pyStepsPreCondParamsWhenTrue   3  = LEWATI langkahnya
--     :1340  pyStepsPreCondParamsWhenFalse  2  = JALANKAN langkahnya
--
-- Dibaca terbalik ia akan tampak membuang justru pada tab Temporary Close. Karena itu
-- kodenya diuji silang terhadap langkah yang maknanya sudah pasti — langkah `Email==0`
-- di `:1459` ber-`WhenTrue=2`, yakni dijalankan saat syaratnya benar. Jadi `3` memang
-- "lewati", dan potongan ini memang berlaku di luar Temporary Close.
--
-- **Konsekuensi saat tab Temporary Close kelak dihidupkan:** penyaring ini WAJIB menjadi
-- bersyarat. Dibiarkan mutlak, tab itu akan selalu kosong — karena yang dicarinya justru
-- `ISPENDINGCLOSE = 'true'`. Hal yang sama berlaku pada `PYSTATUSWORK NOT IN (…)` di
-- atasnya, yang membuang persis populasi yang dicari tab tersebut.
--
-- Export sudah memakai penyaring yang sama sejak awal (`:259`, `:289`); yang tertinggal
-- hanyalah daftar, hitungan, dan ringkasan — ketiganya diperbaiki 2026-10-11.

-- ============================================================================
-- DUA TAB SURVEI — keduanya dari SURVEYORTYPE_1, dan hanya SATU yang menyalin Pega
-- ============================================================================
--
--     Internal Surveyor   SURVEYORTYPE_1 = '1'
--     Loss Adjuster       SURVEYORTYPE_1 = '2'
--
-- **Internal Surveyor menyalin Pega apa adanya.** `CountInternalSurveyorPIC-SQL.xml`
-- menyaring `a.SURVEYORTYPE_1='1'` pada baris klaim itu sendiri, tanpa menyentuh objek
-- survei sama sekali.
--
-- **Loss Adjuster adalah PEMBACAAN, bukan salinan.** `CountOSLossAdjusterPIC-SQL.xml`
-- membaca `C.SURVEYORTYPE_1='2'` pada baris berkelas `Work-SurveyClaim` — tetapi alias `C`
-- di sana **tidak punya satu pun syarat penghubung** ke klaim maupun ke penugasan. Itu
-- cross join: hasilnya (klaim milik saya) × (seluruh survei terbuka di sistem). Berlaku di
-- kueri `Count` MAUPUN `Browse`.
--
-- Perilaku itu **tidak direplikasi** — ia masuk kategori cacat `P-5`, bukan perilaku yang
-- dipertahankan. Yang dipakai adalah kolom pada baris klaim, yang membawa domain nilai yang
-- sama dan diisi dari pilihan yang sama.
--
-- # Kenapa BUKAN T_SURVEYORLIST
--
-- Tabel itu dipertimbangkan dan diukur (2026-10-11, DEV):
--
--     T_CLAIMLIST_ADMIN.SURVEYORTYPE_1   satuannya KLAIM           674 klaim
--     T_SURVEYORLIST.SURVEYTYPE          satuannya LANGKAH SURVEI  280 baris = 113 klaim
--
-- Keduanya menjawab pertanyaan yang berbeda: kolom klaim menyatakan **jenis surveyor yang
-- dipilih**, sedangkan tabel survei menyatakan **survei yang benar-benar tercatat**. Pada
-- klaim yang masih berjalan, hanya **41 dari 513** klaim bertipe `1` punya baris survei.
--
-- Memakainya akan menampilkan 41 alih-alih 513 — bukan lebih akurat, melainkan **bukan isi
-- tab itu**. Keputusan Work Owner 2026-10-11: keduanya dari `SURVEYORTYPE_1`.

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
   AND (k.ISPENDINGCLOSE <> 'true' OR k.ISPENDINGCLOSE IS NULL)
   AND (:3 IS NULL
        OR UPPER(k.PYID) LIKE :4 ESCAPE '\'
        OR UPPER(k.POLICYNO) LIKE :5 ESCAPE '\'
        OR UPPER(k.USERTEKNIS_1) LIKE :6 ESCAPE '\')
   AND (:7 IS NULL OR UPPER(TRIM(k.GROUPPANEL_1)) = :8)
   AND (:9 IS NULL OR UPPER(TRIM(k.PNCCASEID)) = :10)
   AND (:11 IS NULL OR UPPER(TRIM(k.PXTASKLABEL)) = :12)
   AND (:13 IS NULL OR UPPER(TRIM(k.BRANCHNAME)) = :14)
   AND (:15 IS NULL
        OR (:16 = 'LENGKAP' AND k.DOKUMENLENGKAP_1 = '1'
            AND (k.TKA_1 <> '1' OR k.TKA_1 IS NULL))
        OR (:17 = 'BELUM'
            AND (k.DOKUMENLENGKAP_1 = '0' OR k.DOKUMENLENGKAP_1 IS NULL)
            AND (k.TKA_1 <> '1' OR k.TKA_1 IS NULL))
        OR (:18 = 'SURVEYOR-INTERNAL' AND k.SURVEYORTYPE_1 = '1')
        OR (:19 = 'LOSS-ADJUSTER' AND k.SURVEYORTYPE_1 = '2')
        OR (:20 = 'TKA' AND k.TKA_1 = '1'))
 ORDER BY k.PXCREATEDATETIME DESC, k.PZINSKEY
OFFSET :21 ROWS FETCH NEXT :22 ROWS ONLY

-- ============================================================================
-- TEMPORARY CLOSE — kueri TERSENDIRI, bukan daftar yang dilonggarkan
-- ============================================================================
--
-- Tab ini mencari populasi yang justru DIBUANG daftar biasa:
--
--     daftar biasa       PYSTATUSWORK NOT IN (Resolved-*)  dan  ISPENDINGCLOSE <> 'true'
--     Temporary Close    PYSTATUSWORK  =  'Resolved-Completed'  dan  ISPENDINGCLOSE = 'true'
--
-- Keduanya saling meniadakan. Menambal daftar biasa dengan syarat bersyarat akan membuat
-- satu `WHERE` memikul dua maksud yang berlawanan — dan Pega sendiri tidak melakukannya:
-- `Activity/SetClaimPNC-Act.xml:4669` memanggil rule yang BERBEDA,
-- `BrowseClaimTemporaryClose`, dengan `WHERE` miliknya sendiri
-- (`RDB List/CountClaimTemporaryClose-SQL.xml`).
--
-- Jadi bentuk ini bukan pilihan gaya: ia mengikuti Pega, dan ia yang membuat kedua maksud
-- tetap terbaca terpisah.
--
-- # Parameternya SENGAJA sama persis dengan my_inbox_list
--
-- Urutan dan jumlah bind identik (:1 sampai :19), sehingga `filterArgs` yang sama dipakai
-- tanpa cabang. Dengan go-ora, bind terikat menurut URUTAN KEMUNCULAN — bukan menurut
-- nomornya — sehingga dua kueri yang berbagi satu penyusun argumen WAJIB sama bentuknya.
-- Menyimpangkan salah satunya tidak menghasilkan galat; ia menggeser seluruh nilai.
--
-- Blok status :15–:17 tetap ditulis walau tab ini tidak memakainya. Adapter mengosongkan
-- status sebelum memanggil — tab ini SUDAH menjadi penyaringnya sendiri — sehingga :15
-- bernilai NULL dan bloknya lolos. Menghapus bloknya akan menggeser :18 dan :19.

-- ============================================================================
-- PEMILIKNYA DIBACA DARI USERTEKNIS_1, BUKAN DARI PENUGASAN
-- ============================================================================
--
-- `RDB List/CountClaimTemporaryClose-SQL.xml` menyaring `a.userteknis_1`, dan **tidak
-- menyentuh `PC_ASSIGN_WORKLIST` sama sekali** — berbeda dari inbox biasa yang menyaring
-- `b.pxassignedoperatorid` lewat join ke worklist.
--
-- Itu bukan ketidakkonsistenan Pega melainkan keharusan: klaim yang sudah
-- `Resolved-Completed` **tidak punya penugasan terbuka lagi**, sehingga
-- `PXASSIGNEDOPERATORID` kosong untuk seluruhnya. Terukur di DEV 2026-10-11: dari 10 klaim
-- tertunda tutup, **nol** punya baris di `PC_ASSIGN_WORKLIST`.
--
-- Menyalin penyaring inbox biasa ke sini karena itu menghasilkan tab yang **selalu kosong**,
-- dan kegagalannya tidak menghasilkan galat apa pun — layarnya tampil rapi, hanya tidak
-- pernah ada isinya. Versi pertama kueri ini melakukan tepat itu, dan tertangkap hanya
-- karena dijalankan terhadap data nyata.

-- name: my_inbox_temporary_close_list
-- Satu halaman klaim pemanggil yang sudah selesai tetapi tertunda penutupannya.
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
   AND UPPER(TRIM(k.USERTEKNIS_1)) IN (:1, :2)
   AND k.PYSTATUSWORK = 'Resolved-Completed'
   AND k.ISPENDINGCLOSE = 'true'
   AND (:3 IS NULL
        OR UPPER(k.PYID) LIKE :4 ESCAPE '\'
        OR UPPER(k.POLICYNO) LIKE :5 ESCAPE '\'
        OR UPPER(k.USERTEKNIS_1) LIKE :6 ESCAPE '\')
   AND (:7 IS NULL OR UPPER(TRIM(k.GROUPPANEL_1)) = :8)
   AND (:9 IS NULL OR UPPER(TRIM(k.PNCCASEID)) = :10)
   AND (:11 IS NULL OR UPPER(TRIM(k.PXTASKLABEL)) = :12)
   AND (:13 IS NULL OR UPPER(TRIM(k.BRANCHNAME)) = :14)
   AND (:15 IS NULL OR :16 IS NOT NULL OR :17 IS NOT NULL
        OR :18 IS NOT NULL OR :19 IS NOT NULL OR :20 IS NOT NULL)
 ORDER BY k.PXCREATEDATETIME DESC, k.PZINSKEY
OFFSET :21 ROWS FETCH NEXT :22 ROWS ONLY

-- name: my_inbox_temporary_close_count
-- Pasangan hitungan my_inbox_temporary_close_list — WHERE-nya wajib sama persis.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
 WHERE k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND UPPER(TRIM(k.USERTEKNIS_1)) IN (:1, :2)
   AND k.PYSTATUSWORK = 'Resolved-Completed'
   AND k.ISPENDINGCLOSE = 'true'
   AND (:3 IS NULL
        OR UPPER(k.PYID) LIKE :4 ESCAPE '\'
        OR UPPER(k.POLICYNO) LIKE :5 ESCAPE '\'
        OR UPPER(k.USERTEKNIS_1) LIKE :6 ESCAPE '\')
   AND (:7 IS NULL OR UPPER(TRIM(k.GROUPPANEL_1)) = :8)
   AND (:9 IS NULL OR UPPER(TRIM(k.PNCCASEID)) = :10)
   AND (:11 IS NULL OR UPPER(TRIM(k.PXTASKLABEL)) = :12)
   AND (:13 IS NULL OR UPPER(TRIM(k.BRANCHNAME)) = :14)
   AND (:15 IS NULL OR :16 IS NOT NULL OR :17 IS NOT NULL
        OR :18 IS NOT NULL OR :19 IS NOT NULL OR :20 IS NOT NULL)

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
   AND (k.ISPENDINGCLOSE <> 'true' OR k.ISPENDINGCLOSE IS NULL)
   AND (:3 IS NULL
        OR UPPER(k.PYID) LIKE :4 ESCAPE '\'
        OR UPPER(k.POLICYNO) LIKE :5 ESCAPE '\'
        OR UPPER(k.USERTEKNIS_1) LIKE :6 ESCAPE '\')
   AND (:7 IS NULL OR UPPER(TRIM(k.GROUPPANEL_1)) = :8)
   AND (:9 IS NULL OR UPPER(TRIM(k.PNCCASEID)) = :10)
   AND (:11 IS NULL OR UPPER(TRIM(k.PXTASKLABEL)) = :12)
   AND (:13 IS NULL OR UPPER(TRIM(k.BRANCHNAME)) = :14)
   AND (:15 IS NULL
        OR (:16 = 'LENGKAP' AND k.DOKUMENLENGKAP_1 = '1'
            AND (k.TKA_1 <> '1' OR k.TKA_1 IS NULL))
        OR (:17 = 'BELUM'
            AND (k.DOKUMENLENGKAP_1 = '0' OR k.DOKUMENLENGKAP_1 IS NULL)
            AND (k.TKA_1 <> '1' OR k.TKA_1 IS NULL))
        OR (:18 = 'SURVEYOR-INTERNAL' AND k.SURVEYORTYPE_1 = '1')
        OR (:19 = 'LOSS-ADJUSTER' AND k.SURVEYORTYPE_1 = '2')
        OR (:20 = 'TKA' AND k.TKA_1 = '1'))

-- ============================================================================
-- DEADLINE TO TEMPORARY CLOSE — kueri tersendiri, menempuh tabel LAIN
-- ============================================================================
--
-- Sumbernya `RDB List/BrowseClaimDeadlineToTemporaryClose-SQL.xml`, dan tiga hal di sana
-- menentukan bentuk kueri ini:
--
--     AND A.PZINSKEY IN (SELECT claimid FROM pooldata.Temporary_CloseSementaraPNC t
--                         WHERE t.PIC = a.userteknis_1)
--     AND a.userteknis_1 IN {ASIS:TempOperator.CityID}
--     {ASIS:TempSearch.source}
--
-- 1. **Pemiliknya `USERTEKNIS_1`**, sama seperti Temporary Close dan dengan alasan yang
--    sama — tab ini tidak bersandar pada penugasan terbuka.
-- 2. **Subkuerinya BERKORELASI**: baris tenggat hanya dihitung bila `PIC`-nya sama dengan
--    PIC teknis klaimnya. Melepas korelasi itu akan meloloskan klaim milik orang lain yang
--    kebetulan punya baris tenggat.
-- 3. **TIDAK ada penyaring `PYSTATUSWORK`** — klaim yang sudah selesai pun ikut. Tetapi
--    `TempSearch.source` ikut ditempelkan, sehingga pengecualian `ISPENDINGCLOSE` tetap
--    berlaku.
--
-- Butir 3 membuatnya berbeda dari KEDUA kueri lain di berkas ini, dan itu disengaja Pega:
-- inbox berjalan membuang klaim selesai, Temporary Close justru mencarinya, dan tab ini
-- tidak peduli.
--
-- # `EXISTS`, bukan `IN (subquery)`
--
-- Setara secara hasil dan lebih terbaca; korelasinya dipertahankan apa adanya.
--
-- # Perbandingan PIC dibiarkan MENTAH
--
-- `t.PIC = k.USERTEKNIS_1` tanpa `UPPER(TRIM(...))`, persis seperti Pega. Keduanya kolom
-- yang sama-sama berasal dari Pega, jadi menormalkannya justru dapat meloloskan pasangan
-- yang di Pega tidak cocok. Penyaring pemilik di atasnya tetap dinormalkan, karena di sana
-- yang dibandingkan adalah identitas login — dan itu memang berbeda bentuknya.
--
-- # Keadaan data
--
-- `POOLDATA.TEMPORARY_CLOSESEMENTARAPNC` memuat **1 baris** di DEV (2026-10-11). Tab ini
-- karenanya hampir selalu kosong di sini; yang diuji adalah kuerinya, bukan isinya.

-- name: my_inbox_deadline_list
-- Satu halaman klaim pemanggil yang punya tenggat penutupan sementara.
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
   AND UPPER(TRIM(k.USERTEKNIS_1)) IN (:1, :2)
   AND (k.ISPENDINGCLOSE <> 'true' OR k.ISPENDINGCLOSE IS NULL)
   AND EXISTS (SELECT 1
                 FROM POOLDATA.TEMPORARY_CLOSESEMENTARAPNC t
                WHERE t.CLAIMID = k.PZINSKEY
                  AND t.PIC = k.USERTEKNIS_1)
   AND (:3 IS NULL
        OR UPPER(k.PYID) LIKE :4 ESCAPE '\'
        OR UPPER(k.POLICYNO) LIKE :5 ESCAPE '\'
        OR UPPER(k.USERTEKNIS_1) LIKE :6 ESCAPE '\')
   AND (:7 IS NULL OR UPPER(TRIM(k.GROUPPANEL_1)) = :8)
   AND (:9 IS NULL OR UPPER(TRIM(k.PNCCASEID)) = :10)
   AND (:11 IS NULL OR UPPER(TRIM(k.PXTASKLABEL)) = :12)
   AND (:13 IS NULL OR UPPER(TRIM(k.BRANCHNAME)) = :14)
   AND (:15 IS NULL OR :16 IS NOT NULL OR :17 IS NOT NULL
        OR :18 IS NOT NULL OR :19 IS NOT NULL OR :20 IS NOT NULL)
 ORDER BY k.PXCREATEDATETIME DESC, k.PZINSKEY
OFFSET :21 ROWS FETCH NEXT :22 ROWS ONLY

-- name: my_inbox_deadline_count
-- Pasangan hitungan my_inbox_deadline_list — WHERE-nya wajib sama persis.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
 WHERE k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND UPPER(TRIM(k.USERTEKNIS_1)) IN (:1, :2)
   AND (k.ISPENDINGCLOSE <> 'true' OR k.ISPENDINGCLOSE IS NULL)
   AND EXISTS (SELECT 1
                 FROM POOLDATA.TEMPORARY_CLOSESEMENTARAPNC t
                WHERE t.CLAIMID = k.PZINSKEY
                  AND t.PIC = k.USERTEKNIS_1)
   AND (:3 IS NULL
        OR UPPER(k.PYID) LIKE :4 ESCAPE '\'
        OR UPPER(k.POLICYNO) LIKE :5 ESCAPE '\'
        OR UPPER(k.USERTEKNIS_1) LIKE :6 ESCAPE '\')
   AND (:7 IS NULL OR UPPER(TRIM(k.GROUPPANEL_1)) = :8)
   AND (:9 IS NULL OR UPPER(TRIM(k.PNCCASEID)) = :10)
   AND (:11 IS NULL OR UPPER(TRIM(k.PXTASKLABEL)) = :12)
   AND (:13 IS NULL OR UPPER(TRIM(k.BRANCHNAME)) = :14)
   AND (:15 IS NULL OR :16 IS NOT NULL OR :17 IS NOT NULL
        OR :18 IS NOT NULL OR :19 IS NOT NULL OR :20 IS NOT NULL)

-- ============================================================================
-- COMMUNICATION — dasarnya SAMA dengan inbox berjalan, yang ditambahkan satu EXISTS
-- ============================================================================
--
-- Berbeda dari Temporary Close dan Deadline, tab ini TIDAK punya populasi sendiri. Dasarnya
-- persis inbox berjalan — pemiliknya `PXASSIGNEDOPERATORID`, klaim selesai dibuang,
-- tertunda tutup dibuang. Yang membedakannya hanya satu syarat tambahan.
--
-- # Apa yang sebenarnya dicarinya
--
-- `Activity/SetClaimPNC-Act.xml` cabang `TempVisibility.Email==5` menyetel DUA parameter:
--
--     TempView.ClaimNo = "0"                                -> kom.komunikasistatus = '0'
--     TempView.CaseID  = "and kom.sender NOT IN <saya>"
--
-- Jadi isinya **komunikasi MASUK yang belum dijawab**: pesan berstatus `0` yang pengirimnya
-- BUKAN pemanggil. Bukan sekadar "klaim ini punya komunikasi".
--
-- Tiga tab bersaudara memakai tabel yang sama dengan kombinasi berbeda — `Email==6` status
-- `0` dari saya, `Email==7` status `1` dari saya — dan keduanya tidak ada di layar ini.
--
-- # SENDER berisi Operator ID, BUKAN email
--
--     SENDER terisi                               574 dari 580
--     cocok sebuah PXASSIGNEDOPERATORID           450
--     berbentuk email                               3
--
-- Karena itu `NOT IN (:21, :22)` memakai **dua identitas**, sama seperti penyaring pemilik.
-- Dibandingkan dengan email login saja, syarat "bukan saya" tidak akan pernah cocok —
-- setiap pesan terbaca sebagai datang dari orang lain, dan tabnya membengkak tanpa galat.
--
-- # Penyambungan ke klaim: DUA jalur langsung, DITAMBAH jembatan survei
--
-- Pega menyambung lewat TIGA kolom:
--
--     (a.pnccaseid = kom.caseid OR a.caseid_1 = kom.caseid OR a.pzinskey = kom.caseid)
--
-- `CASEID_1` **tidak ada** di `T_CLAIMLIST_ADMIN`, dan isinya terbukti **kunci objek
-- survei** — polanya `ASM-FW-GCNMFW-WORK SRV-…` pada seluruh 146 baris yang terisi.
--
-- `POOLDATA.T_SURVEYORLIST` adalah peta SRV -> PNC: `CASEID` berisi SRV (279 dari 280),
-- `PNCCASEID` berisi kunci klaim (280 dari 280). Ia karena itu dipakai sebagai jembatan,
-- dan jangkauannya nyaris setara:
--
--     dua kolom langsung saja                    176 dari 580 pesan
--     ditambah jembatan T_SURVEYORLIST           336
--     Pega dengan ketiga kolomnya                338
--
-- Selisih 2 pesan. **Permintaan kolom baru ke DBA karena itu tidak diperlukan.**
--
-- # Yang TIDAK terjangkau siapa pun
--
-- `M_KOMUNIKASI_PNC.CASEID` menampung beberapa jenis pengenal sekaligus: SRV (332), PNC
-- (130), angka telanjang (38), nomor klaim format baru `PNCN.YY.xxxx` (19), dan RCV (15).
-- Dua jenis terakhir tidak dijangkau Pega maupun kueri ini — ketiga kolom Pega hanya
-- mencakup SRV dan PNC.

-- name: my_inbox_communication_list
-- Satu halaman klaim pemanggil yang punya komunikasi masuk belum terjawab.
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
   AND (k.ISPENDINGCLOSE <> 'true' OR k.ISPENDINGCLOSE IS NULL)
   AND (:3 IS NULL
        OR UPPER(k.PYID) LIKE :4 ESCAPE '\'
        OR UPPER(k.POLICYNO) LIKE :5 ESCAPE '\'
        OR UPPER(k.USERTEKNIS_1) LIKE :6 ESCAPE '\')
   AND (:7 IS NULL OR UPPER(TRIM(k.GROUPPANEL_1)) = :8)
   AND (:9 IS NULL OR UPPER(TRIM(k.PNCCASEID)) = :10)
   AND (:11 IS NULL OR UPPER(TRIM(k.PXTASKLABEL)) = :12)
   AND (:13 IS NULL OR UPPER(TRIM(k.BRANCHNAME)) = :14)
   AND (:15 IS NULL OR :16 IS NOT NULL OR :17 IS NOT NULL
        OR :18 IS NOT NULL OR :19 IS NOT NULL OR :20 IS NOT NULL)
   AND EXISTS (SELECT 1
                 FROM POOLDATA.M_KOMUNIKASI_PNC kom
                WHERE kom.KOMUNIKASISTATUS = '0'
                  AND UPPER(TRIM(kom.SENDER)) NOT IN (:21, :22)
                  AND (kom.CASEID = k.PZINSKEY
                       OR kom.CASEID = k.PNCCASEID
                       OR EXISTS (SELECT 1
                                    FROM POOLDATA.T_SURVEYORLIST srv
                                   WHERE srv.CASEID = kom.CASEID
                                     AND srv.PNCCASEID = k.PZINSKEY)))
 ORDER BY k.PXCREATEDATETIME DESC, k.PZINSKEY
OFFSET :23 ROWS FETCH NEXT :24 ROWS ONLY

-- name: my_inbox_communication_count
-- Pasangan hitungan my_inbox_communication_list — WHERE-nya wajib sama persis.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
 WHERE k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND UPPER(TRIM(k.PXASSIGNEDOPERATORID)) IN (:1, :2)
   AND k.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (k.ISPENDINGCLOSE <> 'true' OR k.ISPENDINGCLOSE IS NULL)
   AND (:3 IS NULL
        OR UPPER(k.PYID) LIKE :4 ESCAPE '\'
        OR UPPER(k.POLICYNO) LIKE :5 ESCAPE '\'
        OR UPPER(k.USERTEKNIS_1) LIKE :6 ESCAPE '\')
   AND (:7 IS NULL OR UPPER(TRIM(k.GROUPPANEL_1)) = :8)
   AND (:9 IS NULL OR UPPER(TRIM(k.PNCCASEID)) = :10)
   AND (:11 IS NULL OR UPPER(TRIM(k.PXTASKLABEL)) = :12)
   AND (:13 IS NULL OR UPPER(TRIM(k.BRANCHNAME)) = :14)
   AND (:15 IS NULL OR :16 IS NOT NULL OR :17 IS NOT NULL
        OR :18 IS NOT NULL OR :19 IS NOT NULL OR :20 IS NOT NULL)
   AND EXISTS (SELECT 1
                 FROM POOLDATA.M_KOMUNIKASI_PNC kom
                WHERE kom.KOMUNIKASISTATUS = '0'
                  AND UPPER(TRIM(kom.SENDER)) NOT IN (:21, :22)
                  AND (kom.CASEID = k.PZINSKEY
                       OR kom.CASEID = k.PNCCASEID
                       OR EXISTS (SELECT 1
                                    FROM POOLDATA.T_SURVEYORLIST srv
                                   WHERE srv.CASEID = kom.CASEID
                                     AND srv.PNCCASEID = k.PZINSKEY)))

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
-- Syarat `AND (A.TKA_1 != '1' OR A.TKA_1 IS NULL)` yang mengekor keduanya **kini dibawa**.
-- Kolom `TKA_1` ditambahkan Work Owner pada 2026-10-11 dan diisi dari Pega, sehingga klaim
-- TKA tidak lagi ikut terhitung "belum lengkap" — ia punya tabnya sendiri.
--
-- Akibatnya **LENGKAP + BELUM tidak selalu sama dengan TOTAL**, dan itu memang benar:
-- selisihnya adalah klaim TKA. `BuildSummary` tidak pernah mengandaikan keduanya berjumlah
-- total, jadi tidak ada yang perlu disesuaikan di sisi Go.
--
-- Keadaan data 2026-10-11 (DEV): `TKA_1` bernilai `'1'` pada 14 klaim dari 2.747.
SELECT SUM(CASE WHEN k.DOKUMENLENGKAP_1 = '1'
                 AND (k.TKA_1 <> '1' OR k.TKA_1 IS NULL)
                THEN 1 ELSE 0 END) AS LENGKAP,
       SUM(CASE WHEN (k.DOKUMENLENGKAP_1 = '0' OR k.DOKUMENLENGKAP_1 IS NULL)
                 AND (k.TKA_1 <> '1' OR k.TKA_1 IS NULL)
                THEN 1 ELSE 0 END) AS BELUM,
       SUM(CASE WHEN k.SURVEYORTYPE_1 = '1' THEN 1 ELSE 0 END) AS SURVEYOR_INTERNAL,
       SUM(CASE WHEN k.SURVEYORTYPE_1 = '2' THEN 1 ELSE 0 END) AS LOSS_ADJUSTER,
       SUM(CASE WHEN k.TKA_1 = '1' THEN 1 ELSE 0 END)          AS TKA,
       COUNT(*) AS TOTAL
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
 WHERE k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND UPPER(TRIM(k.PXASSIGNEDOPERATORID)) IN (:1, :2)
   AND k.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (k.ISPENDINGCLOSE <> 'true' OR k.ISPENDINGCLOSE IS NULL)
   AND (:3 IS NULL
        OR UPPER(k.PYID) LIKE :4 ESCAPE '\'
        OR UPPER(k.POLICYNO) LIKE :5 ESCAPE '\'
        OR UPPER(k.USERTEKNIS_1) LIKE :6 ESCAPE '\')
   AND (:7 IS NULL OR UPPER(TRIM(k.GROUPPANEL_1)) = :8)
   AND (:9 IS NULL OR UPPER(TRIM(k.PNCCASEID)) = :10)
   AND (:11 IS NULL OR UPPER(TRIM(k.PXTASKLABEL)) = :12)
   AND (:13 IS NULL OR UPPER(TRIM(k.BRANCHNAME)) = :14)
