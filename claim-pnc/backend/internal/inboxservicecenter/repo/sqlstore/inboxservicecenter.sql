-- Kueri modul Inbox Service Center: klaim portal rekanan (perbaikan perangkat).
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- SELURUH tabel yang dibaca berkas ini milik sistem lama. Tidak ada satu pun pernyataan yang
-- menulis, dan memang tidak boleh ada: selama masa paralel setiap tabel hanya boleh ditulis
-- SATU sistem, dan tabel ini milik Pega (`P-1`).
--
-- ============================================================================
-- SATU RULE YANG TIDAK ADA DI EXPORT — dan bagaimana kueri ini disusun ulang
-- ============================================================================
--
-- `Activity/DataServiceCenter-Act.xml` memanggil RDB rule `GetDataServiceCenter` pada kelas
-- `ASM-FW-GCNMFW-Data-ClaimData`. Yang ikut terekspor adalah rule BERNAMA SAMA pada kelas
-- `ASM-FW-GCNMFW-Int-T_GENERAL` — kueri lain sama sekali, yang membaca
-- `pooldata.service_log_nonmbu`. Kueri grid yang sebenarnya tidak ada di export (`R-16`).
--
-- Ia TIDAK dikarang. Ketiga rule sekelas yang ADA memberi seluruh bahannya:
--
--   CountDataServiceCenter        tabelnya (POOLDATA.T_KLAIM_PORTAL_REKANAN) dan KETIGA
--                                 penyaringnya persis: {ASIS:TempSQL.AlasanKlaim}
--                                 {ASIS:TempSQL.Remark} {ASIS:TempSts.ClaimID}
--   ExportDataServiceCenter       kolom yang dibaca dari tabel yang sama
--   GetDataServiceCenter_Update   peta kolom -> alias, lengkap untuk seluruh kolom
--
-- Keenam kolom yang digambar grid ada di kedua kueri terakhir. Yang tersisa sebagai DUGAAN
-- hanyalah urutannya — lihat catatan ORDER BY di bawah.
--
-- ============================================================================
-- PEMETAAN TIGA ARAH — properti grid Pega -> kolom sebenarnya -> arti
-- ============================================================================
--
-- Alias di sistem lama nyaris seluruhnya menyesatkan; ini satu-satunya tempat ketiganya
-- dapat dibandingkan berdampingan.
--
-- Judul kolom     Properti grid   Kolom sebenarnya   Alias lama       Alias di sini
-- --------------- --------------- ------------------ ---------------- --------------
-- ID              .CaseID         ID                 "CaseID"         ID
-- Tanggal Input   .DateOfLoss (!) INPUTDATE          "DateOfLoss"     INPUT_DATE
-- No Polis        .PolicyNo       NOPOLIS            "PolicyNo"       POLICY_NUMBER
-- Nasabah         .UserName   (!) QQNAME             "NamaDokumen"    CUSTOMER_NAME
-- Tipe            .RefNo      (!) TYPE               "RefNo"          TYPE
-- PIC             .PICRekanan     PIC                "PICRekanan"     TECHNICAL_PIC
--
-- Tidak digambar sebagai kolom, tetapi dibawa:
--
-- (tidak ada)     —               REPAIRID           "NoClaim"    (!) REPAIR_ID
-- (tidak ada)     —               CLAIMNO            "ClaimNo"        CLAIM_NUMBER
-- (tidak ada)     —               IMEI               "ClientID"   (!) IMEI
-- (tidak ada)     —               STS_APPROVAL       "Keyword"    (!) APPROVAL_STATUS
-- (tidak ada)     —               STATUS             "Status"         REPAIR_STATUS
-- (tidak ada)     —               LOGIN              "Resources"  (!) OWNER
-- (tidak ada)     —               KOMITEAPPROVE      "Bgklname"   (!) COMMITTEE_APPROVER
--
-- Tanda (!) berarti nama lamanya menyebut hal yang BERBEDA dari isinya. `DateOfLoss` bukan
-- tanggal kejadian melainkan tanggal input; `UserName` bukan nama pengguna melainkan nama
-- nasabah; `NoKTP` pada kueri export justru berisi IMEI.
--
-- ============================================================================
-- PENYARING — dari mana setiap potongannya
-- ============================================================================
--
-- Sistem lama merangkainya sebagai POTONGAN SQL lewat tiga titik `{ASIS:...}`. Di sini
-- seluruhnya menjadi parameter binding; larangan merangkai nilai ke dalam teks SQL tidak
-- ikut dikecualikan oleh `P-5`, karena yang direplikasi adalah perilaku bisnis — bukan
-- celah injeksi (`03-CURRENT-ARCHITECTURE.md` §4.5).
--
--   STS_APPROVAL   Activity/DataServiceCenter-Act.xml langkah 8, 12, 15
--                  langkah  8  selalu jalan        = '<tab>'
--                  langkah 12  bila tab "2"        IN ('2','3')      <- menimpa
--                  langkah 15  bila tab ""         IS NULL           <- menimpa
--
--   PIC            langkah 13 — prakondisi `stsapprove=="komite"` dengan
--                  true=3 (lewati) dan false=2 (lanjut), sehingga ia berlaku pada SELURUH
--                  tab layar ini. Inilah yang membuatnya "inbox saya".
--
--   Cari          langkah 17 dan 18, dua potongan TERPISAH yang keduanya ikut terpasang:
--                  17: AND (ID      LIKE % OR NOPOLIS LIKE % OR QQNAME LIKE % OR IMEI LIKE %)
--                  18: AND (CLAIMNO LIKE % OR NOPOLIS LIKE % OR QQNAME LIKE % OR IMEI LIKE %)
--                  Karena keduanya digabung dengan AND, yang benar-benar dapat dicari
--                  hanyalah irisannya: NOPOLIS, QQNAME, dan IMEI. Mencari dengan ID saja
--                  atau CLAIMNO saja TIDAK menghasilkan baris.
--                  Ini cacat sistem lama. Ia DIREPLIKASI (`P-5`) dan dinyatakan terbuka
--                  lewat inboxservicecenter.Limitations, bukan diperbaiki diam-diam.
--
--   LOGIN          langkah 19 — hanya bila access group `GCNMFW:PNCServiceCenter`.
--                  TIDAK dibawa: sumber peran belum ada (`TKT-F3-004`). Penyaring PIC di
--                  atas sudah mempersempit barisnya, sehingga ketiadaannya tidak membuka
--                  antrean orang lain.
--
-- ============================================================================
-- ORDER BY — satu-satunya bagian yang masih dugaan
-- ============================================================================
--
-- Kueri grid aslinya hilang, sehingga urutannya tidak terbaca. Dua bukti tersedia dan
-- keduanya berbeda:
--
--   Activity/DataServiceCenter-Act.xml langkah 28   order by A.INPUTDATE desc
--   RDB List/ExportDataServiceCenter-SQL.xml        ORDER BY ID ASC
--
-- Yang dipakai di sini adalah yang PERTAMA, dengan dua alasan: ia grid atas tabel yang sama
-- (`T_KLAIM_PORTAL_REKANAN`), sedangkan yang kedua adalah unduhan berkas; dan antrean kerja
-- yang terbaru di atas adalah urutan yang berguna bagi petugas.
--
-- `ID` disertakan sebagai pemecah seri supaya urutan baris tidak berubah-ubah antar halaman
-- ketika beberapa baris punya INPUTDATE yang sama persis — tanpa itu, satu baris dapat
-- muncul dua kali atau terlewat saat berpindah halaman.
--
-- Bila Tim Pega kelak mengirim rule aslinya dan urutannya ternyata berbeda, yang berubah
-- hanyalah klausa ini.
--
-- ============================================================================
-- PENANDA PARAMETER
-- ============================================================================
--
-- Setiap penanda muncul TEPAT SEKALI dan bernomor urut, sesuai temuan §63 catatan
-- pengembangan: penanda berulang membuat jumlah argumen tidak lagi sama dengan jumlah
-- kemunculan, dan galatnya baru terbaca saat kueri dijalankan.
--
--   :1        'Y' bila tab ini menerima baris ber-STS_APPROVAL NULL
--   :2        'Y' bila tab ini menerima baris ber-kode tertentu
--   :3 :4     kedua kode yang diterima (tab berkode tunggal mengirim kode yang sama dua kali)
--   :5        login pemanggil, huruf besar — penyaring PIC
--   :6        kata kunci, NULL bila tidak mencari — penjaga kelompok pencarian pertama
--   :7 :8 :9 :10    pola LIKE kelompok pertama: ID, NOPOLIS, QQNAME, IMEI
--   :11       kata kunci, NULL bila tidak mencari — penjaga kelompok pencarian kedua
--   :12 :13 :14 :15 pola LIKE kelompok kedua: CLAIMNO, NOPOLIS, QQNAME, IMEI
--   :16 :17   offset dan jumlah baris — hanya pada list_claims

-- name: list_claims
-- Satu halaman klaim portal rekanan milik pemanggil pada satu tab.
SELECT k.ID,
       k.REPAIRID,
       k.CLAIMNO,
       k.NOPOLIS,
       k.QQNAME,
       k.TYPE,
       k.PIC,
       k.INPUTDATE,
       k.IMEI,
       k.STS_APPROVAL,
       k.STATUS,
       k.LOGIN,
       k.KOMITEAPPROVE
  FROM POOLDATA.T_KLAIM_PORTAL_REKANAN k
 WHERE ( (:1 = 'Y' AND k.STS_APPROVAL IS NULL)
      OR (:2 = 'Y' AND TRIM(k.STS_APPROVAL) IN (:3, :4)) )
   AND UPPER(TRIM(k.PIC)) = :5
   AND (:6 IS NULL
        OR UPPER(k.ID) LIKE :7 ESCAPE '\'
        OR UPPER(k.NOPOLIS) LIKE :8 ESCAPE '\'
        OR UPPER(k.QQNAME) LIKE :9 ESCAPE '\'
        OR UPPER(k.IMEI) LIKE :10 ESCAPE '\')
   AND (:11 IS NULL
        OR UPPER(k.CLAIMNO) LIKE :12 ESCAPE '\'
        OR UPPER(k.NOPOLIS) LIKE :13 ESCAPE '\'
        OR UPPER(k.QQNAME) LIKE :14 ESCAPE '\'
        OR UPPER(k.IMEI) LIKE :15 ESCAPE '\')
 ORDER BY k.INPUTDATE DESC, k.ID
OFFSET :16 ROWS FETCH NEXT :17 ROWS ONLY

-- name: count_claims
-- Jumlah seluruh baris yang cocok, tanpa paginasi.
--
-- Penyaringnya WAJIB sama persis dengan list_claims. Keduanya dijaga query_test.go, karena
-- penyaring yang berbeda menghasilkan "halaman 1 dari 7" yang halaman ketujuhnya kosong.
--
-- Sumbernya `RDB List/CountDataServiceCenter-SQL.xml`, yang di sistem lama pun kueri
-- TERSENDIRI dengan ketiga `{ASIS:...}` yang sama.
SELECT COUNT(*)
  FROM POOLDATA.T_KLAIM_PORTAL_REKANAN k
 WHERE ( (:1 = 'Y' AND k.STS_APPROVAL IS NULL)
      OR (:2 = 'Y' AND TRIM(k.STS_APPROVAL) IN (:3, :4)) )
   AND UPPER(TRIM(k.PIC)) = :5
   AND (:6 IS NULL
        OR UPPER(k.ID) LIKE :7 ESCAPE '\'
        OR UPPER(k.NOPOLIS) LIKE :8 ESCAPE '\'
        OR UPPER(k.QQNAME) LIKE :9 ESCAPE '\'
        OR UPPER(k.IMEI) LIKE :10 ESCAPE '\')
   AND (:11 IS NULL
        OR UPPER(k.CLAIMNO) LIKE :12 ESCAPE '\'
        OR UPPER(k.NOPOLIS) LIKE :13 ESCAPE '\'
        OR UPPER(k.QQNAME) LIKE :14 ESCAPE '\'
        OR UPPER(k.IMEI) LIKE :15 ESCAPE '\')

-- name: check_table
-- Memastikan tabel inti modul ini terbaca dari koneksi yang dipakai.
--
-- Ia dipanggil perintah `-periksa` saat aplikasi start, dan sengaja tidak menyentuh baris
-- mana pun: yang diperiksa adalah HAK BACA dan keberadaan tabelnya, bukan isinya.
--
-- `COUNT(*)`, bukan `SELECT 1`: dengan `WHERE 1 = 0` yang pertama tetap mengembalikan satu
-- baris berisi nol, sedangkan yang kedua tidak mengembalikan baris sama sekali — dan
-- pemindainya akan melaporkan "tidak ada baris" seolah tabelnya bermasalah.
SELECT COUNT(*)
  FROM POOLDATA.T_KLAIM_PORTAL_REKANAN
 WHERE 1 = 0

-- name: find_detail
-- Satu klaim beserta SELURUH isian layar rinciannya.
--
-- Kolomnya dibaca dari `RDB List/GetDataServiceCenter_Update-SQL.xml` dan disilangkan dengan
-- daftar kolom yang benar-benar ditulis `Database/PEGA_PORTAL_REKANAN.prc`. Procedure itu
-- menjadi penengah ketika keduanya berselisih, sebab ia menyebut kolom apa adanya tanpa alias.
--
-- # Dua perbedaan yang DISENGAJA terhadap kueri lama
--
--   1. `OTHER_FEE` dan `CANCELLED_REASON` di sana sama-sama dialiaskan `"DistrictID"`,
--      sehingga yang belakangan menimpa yang duluan dan salah satunya selalu hilang di
--      klipboard. Di sini keduanya kolom terpisah.
--   2. Penyaring `PIC` ditambahkan. Kueri lama tidak punya — di Pega rincian hanya dapat
--      dicapai lewat klik pada baris yang SUDAH tersaring, sehingga tidak dibutuhkan. Pada
--      API yang dapat dipanggil langsung jaminan itu hilang, dan `ID` di tabel ini berurutan.
--
-- Keduanya dinyatakan lewat inboxservicecenter.Limitations, bukan disembunyikan.
--
--   :1  ID klaim, huruf besar
--   :2  login pemanggil, huruf besar — penyaring PIC
SELECT k.ID,
       k.REPAIRID,
       k.CLAIMNO,
       k.TYPE,
       k.LOGIN,
       k.INPUTDATE,
       k.NOPOLIS,
       k.INSURANCE,
       k.STARTDATE,
       k.ENDDATE,
       k.CUST_NAME,
       k.QQNAME,
       k.NOHP,
       k.NOKTP,
       k.PRINCIPAL_BILL_NO,
       k.INSURANCE_BILL_NO,
       k.QUOTATIONNO,
       k.QUOTATION_AMOUNT,
       k.PROD_GROUP,
       k.PROD_CATEGORY,
       k.BRAND,
       k.MODEL,
       k.COLOUR,
       k.DEVICE,
       k.IMEI,
       k.SERIALNO,
       k.ITEMWARRANTY,
       k.OBJECT,
       k.COLLECT_POINT,
       k.REPAIR_POINT,
       k.IS_DELIVERY,
       k.SYMPTOM_CODE,
       k.SYMPTOM_DESC,
       k.ANALISA,
       k.PIC,
       k.STATUS,
       k.REASON,
       k.CANCELLED_REASON,
       k.CUST_ARRIVAL,
       k.ACKNOWLEDGE_DATE,
       k.ASSIGNED_DATE,
       k.COMPLETED_DATE,
       k.RELEASE_DATE,
       k.INVOICE_DATE,
       k.ESTIMATED_PICKUP_DATE,
       k.PICKUP_COURIER_DATE,
       k.DOWNPAYMENT,
       k.DP_NO,
       k.DP_METHOD,
       k.STS_APPROVAL,
       k.REMARK,
       k.KOMITEAPPROVE,
       k.DETAILPART,
       k.ACCESSORIESLAINYA,
       k.SERVICE_FEE,
       k.SPAREPART_FEE,
       k.SUKUCADANG,
       k.SUKUCADANGAPPROVE,
       k.TAX_FEE,
       k.TAX_FEEAPPROVE,
       k.PPN,
       k.PPNAPPROVE,
       k.DELIVERY_FEE,
       k.DELIVERY_FEEAPPROVE,
       k.OTHER_FEE,
       k.EXCESS,
       k.EXCESSAPPROVE,
       k.DEDUCTIBLE,
       k.DEDUCAPPROVE,
       k.TOTAL_FEE,
       k.TOTAL_FEEAPPROVE,
       k.DESKCHARGER,
       k.CGARGERCABLE,
       k.CARKIT,
       k.REMOVABLEANTENNA,
       k.HEADSET,
       k.BATTERY,
       k.SIMCARD,
       k.EXTRACOVER,
       k.BATTERYCOVER,
       k.LCD_TEXT,
       k.CASE,
       k.BOXUNIT
  FROM POOLDATA.T_KLAIM_PORTAL_REKANAN k
 WHERE UPPER(TRIM(k.ID)) = :1
   AND UPPER(TRIM(k.PIC)) = :2

-- name: list_progress
-- Riwayat catatan progres satu klaim.
--
-- Sumbernya `RDB List/GetListDataServiceCenter-SQL.xml`, apa adanya termasuk urutannya:
-- `INSERTDATE ASC`, terlama di atas.
--
-- Dikunci `REPAIRID`, BUKAN `ID` — keduanya kolom berbeda pada tabel klaimnya, dan hanya yang
-- pertama yang tersimpan di tabel progres ini.
--
--   :1  REPAIRID
SELECT p.REPAIRID,
       p.INSERTDATE,
       p.NOTEPROGRESS,
       p.USERUPDATE
  FROM POOLDATA.PROGRESS_SERVICECENTER_CLAIM p
 WHERE TRIM(p.REPAIRID) = :1
 ORDER BY p.INSERTDATE ASC
