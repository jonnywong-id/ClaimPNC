-- Kueri modul Inbox PLA, DLA, Pre DLA (`MENU_ID 44`, pengganti `Harness/InboxPLA_harness`).
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- ============================================================================
-- TABEL YANG DIBACA, DAN SIAPA PEMILIKNYA
-- ============================================================================
--
--   POOLDATA.T_CLAIM_PNC     dimiliki Pega — hanya dibaca
--   POOLDATA.T_PLALIST       dimiliki Pega — hanya dibaca
--   POOLDATA.T_DLALIST       dimiliki Pega — hanya dibaca
--   POOLDATA.T_PREDLALIST    dimiliki Pega — hanya dibaca
--   POOLDATA.BUSINESS        dimiliki Pega — hanya dibaca (penyaring tab DLA)
--
-- Berkas ini TIDAK MENULIS satu baris pun. Ketiga tabel dokumen masih ditulis Pega lewat
-- tombol "Send" yang belum dibangun di sini, sehingga `P-1` terpenuhi dengan cara yang
-- paling sederhana: kepemilikannya belum berpindah.
--
-- ============================================================================
-- TIGA DAFTAR, TIGA TABEL, TIGA PENYARING YANG BERBEDA
-- ============================================================================
--
--   list_pla      T_PLALIST     REINSCODE IS NOT NULL
--                               DAN (ISKIRIM IS NULL OR ISKIRIM = '0')
--   list_dla      T_DLALIST     (ISKIRIM IS NULL OR ISKIRIM = '0')
--                               DAN BRANCHNAME != 'ASNET'
--                               DAN bukan BUSINESSGROUPID '10008'
--   list_pre_dla  T_PREDLALIST  NOAKSEP IS NULL     <- TIDAK menyinggung ISKIRIM
--
-- Baris ketiga itu yang paling mudah "diperbaiki" menjadi salah. Yang mengeluarkan sebuah
-- Pre-DLA dari antrean adalah terbitnya Nomor Akseptasi, bukan terkirimnya surat — dan itu
-- benar secara bisnis: Pre-DLA memberitahukan nilai yang AKAN diaksep, sehingga ia
-- kehilangan gunanya begitu akseptasinya terbit. Menyeragamkan ketiganya mengubah isi
-- antrean tanpa satu pun galat.
--
-- ============================================================================
-- PEMETAAN KOLOM — kolom sebenarnya -> alias Pega -> alias di sini
-- ============================================================================
--
-- Alias Pega TIDAK dibawa (`D-19`). Tujuh dari delapan tidak menyatakan isinya, dan empat
-- menyatakan hal yang SALAH — menyalinnya akan menampilkan kolom yang keliru tanpa satu
-- pun galat:
--
--   kolom sebenarnya     alias Pega       alias di sini    judul kolom
--   -------------------- ---------------- ---------------- -------------------
--   c.CLAIMID            "BRANCH_NAME" (!) CLAIM_KEY       (tidak digambar)
--   c.CLAIMNO            "BRANCH_CODE" (!) CLAIM_NO        No Klaim
--   c.NOPOLIS            "POLICY_NO"       POLICY_NO       No Polis
--   c.QQNAME             "CUSTOMER"        INSURED         Nama Tertanggung
--   c.REGISTERDATE       "START_DATE"      REGISTER_DATE   Tanggal Register
--   c.DATEOFLOSS         "END_DATE"    (!) LOSS_DATE       Tanggal Kejadian
--   c.PICTEKNIK          "BUSINESS_NAME"(!) PIC_TEKNIK     PIC Teknik
--   MAX(TGLPLA/TGLDLA)   "pyCreateDate"(!) ADVICE_DATE     Tanggal PLA/DLA/Pre DLA
--
-- Dua alias yang paling menjebak berdampingan: `"BRANCH_NAME"` dan `"BRANCH_CODE"` dipakai
-- berurutan padahal tidak satu pun menyangkut cabang — yang pertama kunci kerja Pega, yang
-- kedua nomor klaim.
--
-- Pada grid rincian ada satu lagi yang setara berbahayanya: `TGLTERIMAPLA` beralias
-- `"POLICY_NO"`. Di seluruh modul lain alias itu berarti nomor polis; di sini ia tanggal.
--
-- ============================================================================
-- KOLOM TANGGAL ADVICE — SATU KEJANGGALAN YANG DIBAWA APA ADANYA
-- ============================================================================
--
-- Sub-kueri tanggalnya menyaring `ISKIRIM IS NULL` saja, sementara penyaring keanggotaan
-- barisnya menerima `ISKIRIM IS NULL OR ISKIRIM = '0'`. Klaim yang SELURUH dokumennya
-- ber-`ISKIRIM = '0'` karena itu muncul di daftar dengan kolom tanggal KOSONG.
--
-- Itu perilaku Pega, bukan kerusakan di sini (`P-5`). Pada tab Pre DLA kejanggalannya
-- lebih jauh: keanggotaannya ditentukan `NOAKSEP IS NULL` — yang tidak menyinggung
-- `ISKIRIM` sama sekali — sedangkan kolom tanggalnya tetap menyaring `ISKIRIM IS NULL`.
--
-- Bentuknya diubah dari `ORDER BY … DESC FETCH NEXT 1 ROW ONLY` menjadi `MAX(...)`.
-- Keduanya menghasilkan nilai yang sama, dan `MAX` tidak menuntut pengurutan per baris.
--
-- ============================================================================
-- PENYARING OPSIONAL
-- ============================================================================
--
-- Rentang tanggal dan kata kunci keduanya boleh kosong, dan satu kueri per tab melayani
-- keempat kombinasinya lewat pola `(:n IS NULL OR …)` — pola yang sudah dipakai modul
-- Inbox Close Claim dan Master Rekening.
--
-- Penanda KEHADIRAN penyaring selalu bertipe TEKS, tidak pernah bertipe tanggal. Bind
-- bertipe tanggal hanya muncul di dalam perbandingan, tempat tipenya tidak mungkin
-- ambigu. Membalikkannya — menguji `IS NULL` pada bind tanggal — menyerahkan penentuan
-- tipe kepada driver, dan itu tempat galat yang hanya muncul di Oracle.
--
-- Batas atas rentang bersifat EKSKLUSIF (`< :n`), bukan `trunc(kolom) <= :n` seperti Pega.
-- Keduanya memilih baris yang sama persis, tetapi `TRUNC` pada kolom membuat index atas
-- kolom itu tidak terpakai — pada tabel dokumen yang tumbuh terus, itu pemindaian penuh
-- setiap kali panel pencarian dipakai. Pola yang sama dipakai modul Archive Dokumen Klaim.
--
-- ============================================================================
-- PAGINASI
-- ============================================================================
--
-- `OFFSET … FETCH NEXT` di basis data, dan jumlah seluruhnya lewat `COUNT(*) OVER ()`.
--
-- Sistem lama menempuhnya berbeda: ia menyisipkan `WHERE "rn" >= … AND "rn" <= …` ke dalam
-- teks SQL lewat pola `{ASIS:Pagination.FirstRow}`, dengan batasnya dirangkai sebagai TEKS.
-- Jumlah seluruhnya pun datang dari kueri penghitung TERPISAH (`CountPNCList_PLA` dan
-- saudara-saudaranya) yang harus dijaga tetap sejalan dengan kueri daftarnya — dua tempat
-- untuk satu penyaring. `COUNT(*) OVER ()` menghapus keduanya.

-- name: list_pla
-- Antrean PLA — klaim yang punya PLA sudah ber-reasuradur tetapi belum terkirim.
--
-- Bind: :1 penanda pencarian · :2 pola pencarian · :3 penanda "dari" · :4 tanggal "dari"
--       :5 penanda "sampai" · :6 tanggal "sampai" (eksklusif) · :7 offset · :8 jumlah baris
--
-- `REINSCODE IS NOT NULL` hanya ada di tab ini: PLA yang belum menunjuk reasuradur belum
-- dapat dikirim ke siapa pun, sehingga ia belum menjadi pekerjaan.
SELECT c.CLAIMID                         AS CLAIM_KEY,
       c.CLAIMNO                         AS CLAIM_NO,
       c.NOPOLIS                         AS POLICY_NO,
       c.QQNAME                          AS INSURED,
       c.REGISTERDATE                    AS REGISTER_DATE,
       c.DATEOFLOSS                      AS LOSS_DATE,
       c.PICTEKNIK                       AS PIC_TEKNIK,
       (SELECT MAX(p.TGLPLA)
          FROM POOLDATA.T_PLALIST p
         WHERE p.CLAIMID = c.CLAIMID
           AND p.ISKIRIM IS NULL)        AS ADVICE_DATE,
       COUNT(*) OVER ()                  AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE c.GROUPPANEL <> '002'
   AND c.GROUPPANEL <> '005'
   AND (:1 IS NULL OR UPPER(c.CLAIMID) LIKE :2 ESCAPE '\')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_PLALIST a
                WHERE a.CLAIMID = c.CLAIMID
                  AND a.REINSCODE IS NOT NULL
                  AND (a.ISKIRIM IS NULL OR a.ISKIRIM = '0')
                  AND (:3 IS NULL OR a.TGLPLA >= :4)
                  AND (:5 IS NULL OR a.TGLPLA < :6))
 ORDER BY c.REGISTERDATE, c.CLAIMNO
OFFSET :7 ROWS FETCH NEXT :8 ROWS ONLY

-- name: list_dla
-- Antrean DLA — klaim yang punya DLA belum terkirim.
--
-- Bind: sama urutannya dengan list_pla.
--
-- DUA penyaring tambahan yang tidak punya pasangan di tab lain, keduanya dibawa apa adanya
-- (`P-5`) karena alasan bisnisnya tidak tertulis di mana pun di export:
--
--   BRANCHNAME <> 'ASNET'
--   tidak ada di BUSINESS ber-BUSINESSGROUPID '10008'
--
-- `BRANCHNAME <> 'ASNET'` ditulis persis seperti Pega, TANPA `OR BRANCHNAME IS NULL`.
-- Akibatnya klaim yang cabangnya kosong ikut tersaring keluar — perbandingan dengan NULL
-- menghasilkan UNKNOWN, bukan TRUE. Itu perilaku sistem lama, dan memperbaikinya akan
-- MENAMBAH baris ke antrean tanpa ada yang memintanya.
SELECT c.CLAIMID                         AS CLAIM_KEY,
       c.CLAIMNO                         AS CLAIM_NO,
       c.NOPOLIS                         AS POLICY_NO,
       c.QQNAME                          AS INSURED,
       c.REGISTERDATE                    AS REGISTER_DATE,
       c.DATEOFLOSS                      AS LOSS_DATE,
       c.PICTEKNIK                       AS PIC_TEKNIK,
       (SELECT MAX(d.TGLDLA)
          FROM POOLDATA.T_DLALIST d
         WHERE d.CLAIMID = c.CLAIMID
           AND d.ISKIRIM IS NULL)        AS ADVICE_DATE,
       COUNT(*) OVER ()                  AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE c.GROUPPANEL <> '002'
   AND c.GROUPPANEL <> '005'
   AND c.BRANCHNAME <> 'ASNET'
   AND NOT EXISTS (SELECT 1
                     FROM POOLDATA.BUSINESS b
                    WHERE b.ID = c.BUSINESSCODE
                      AND b.BUSINESSGROUPID = '10008')
   AND (:1 IS NULL OR UPPER(c.CLAIMID) LIKE :2 ESCAPE '\')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_DLALIST a
                WHERE a.CLAIMID = c.CLAIMID
                  AND (a.ISKIRIM IS NULL OR a.ISKIRIM = '0')
                  AND (:3 IS NULL OR a.TGLDLA >= :4)
                  AND (:5 IS NULL OR a.TGLDLA < :6))
 ORDER BY c.REGISTERDATE, c.CLAIMNO
OFFSET :7 ROWS FETCH NEXT :8 ROWS ONLY

-- name: list_pre_dla
-- Antrean Pre DLA — klaim yang punya Pre-DLA belum ber-Nomor Akseptasi.
--
-- Bind: sama urutannya dengan list_pla.
--
-- Perhatikan penyaring keanggotaannya: `NOAKSEP IS NULL`, BUKAN `ISKIRIM`. Lihat catatan
-- di kepala berkas ini.
SELECT c.CLAIMID                         AS CLAIM_KEY,
       c.CLAIMNO                         AS CLAIM_NO,
       c.NOPOLIS                         AS POLICY_NO,
       c.QQNAME                          AS INSURED,
       c.REGISTERDATE                    AS REGISTER_DATE,
       c.DATEOFLOSS                      AS LOSS_DATE,
       c.PICTEKNIK                       AS PIC_TEKNIK,
       (SELECT MAX(p.TGLDLA)
          FROM POOLDATA.T_PREDLALIST p
         WHERE p.CLAIMID = c.CLAIMID
           AND p.ISKIRIM IS NULL)        AS ADVICE_DATE,
       COUNT(*) OVER ()                  AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE c.GROUPPANEL <> '002'
   AND c.GROUPPANEL <> '005'
   AND (:1 IS NULL OR UPPER(c.CLAIMID) LIKE :2 ESCAPE '\')
   AND EXISTS (SELECT 1
                 FROM POOLDATA.T_PREDLALIST a
                WHERE a.CLAIMID = c.CLAIMID
                  AND a.NOAKSEP IS NULL
                  AND (:3 IS NULL OR a.TGLDLA >= :4)
                  AND (:5 IS NULL OR a.TGLDLA < :6))
 ORDER BY c.REGISTERDATE, c.CLAIMNO
OFFSET :7 ROWS FETCH NEXT :8 ROWS ONLY

-- name: documents_pla
-- Grid "Detail PLA List" — SELURUH PLA milik satu klaim, terkirim maupun belum.
--
-- Bind: :1 kunci klaim (`T_CLAIM_PNC.CLAIMID`)
--
-- Ia sengaja TIDAK menyaring `ISKIRIM`: grid ini adalah riwayat pemberitahuan klaim itu,
-- dan kolom "Terkirim" beserta "Tanggal Kirim" justru ada supaya perbedaannya terlihat.
-- Kueri lama pun tidak menyaringnya.
--
-- `ORDER BY` DITAMBAHKAN. `GetPLAList` tidak punya satu pun, sehingga urutan barisnya di
-- Pega tidak ditentukan — dua pembukaan layar yang sama dapat menampilkan urutan yang
-- berbeda. Urutan yang tidak ditentukan bukan perilaku yang layak ditiru; ia diurutkan
-- menurut tanggal dokumen, lalu nomornya sebagai pemutus seri.
SELECT a.NOPLA                           AS ADVICE_NO,
       a.PLAREINSURER                    AS REINSURER,
       a.TIPEPLA                         AS ADVICE_TYPE,
       a.REVISI                          AS REVISION,
       a.TGLPLA                          AS ADVICE_DATE,
       a.ISKIRIM                         AS SENT,
       a.TGLKIRIM                        AS SENT_DATE,
       a.TGLTERIMAPLA                    AS RECEIVED_DATE,
       a.NOTES                           AS NOTES,
       a.EMAILPLA                        AS EMAIL,
       NULL                              AS ACCEPTANCE_NO
  FROM POOLDATA.T_PLALIST a
 WHERE a.CLAIMID = :1
 ORDER BY a.TGLPLA, a.NOPLA

-- name: documents_dla
-- Grid "Detail DLA List" — SELURUH DLA milik satu klaim.
--
-- Bind: :1 kunci klaim (`T_CLAIM_PNC.CLAIMID`)
--
-- Berbeda dari PLA pada dua kolom: `REVISI` tidak diambil `GetDLAList` sama sekali, dan
-- `NOAKSEP` hanya ada di sini. Keduanya tetap berada di posisi yang SAMA pada senarai
-- kolom supaya satu pemindai melayani kedua kueri — kolom yang tidak ada diisi NULL,
-- bukan dihilangkan.
SELECT a.NODLA                           AS ADVICE_NO,
       a.DLAREINSURER                    AS REINSURER,
       a.TIPEDLA                         AS ADVICE_TYPE,
       NULL                              AS REVISION,
       a.TGLDLA                          AS ADVICE_DATE,
       a.ISKIRIM                         AS SENT,
       a.TGLKIRIM                        AS SENT_DATE,
       a.TGLTERIMADLA                    AS RECEIVED_DATE,
       a.NOTES                           AS NOTES,
       a.EMAILDLA                        AS EMAIL,
       a.NOAKSEP                         AS ACCEPTANCE_NO
  FROM POOLDATA.T_DLALIST a
 WHERE a.CLAIMID = :1
 ORDER BY a.TGLDLA, a.NODLA

-- name: claim_exists
-- Memastikan kunci klaimnya benar-benar ada pada entitas ini.
--
-- Bind: :1 kunci klaim
--
-- # Kenapa satu kueri TERSENDIRI, dan bukan disimpulkan dari daftar dokumen yang kosong
--
-- Karena keduanya berarti hal yang BERBEDA, dan hanya satu yang merupakan kekeliruan:
--
--   klaim ada, dokumen kosong    keadaan yang sah — jawabannya daftar kosong
--   klaim tidak ada              kunci yang salah, atau portal yang salah -> 404
--
-- Yang kedua paling sering terjadi bukan karena klaimnya benar-benar tidak ada, melainkan
-- karena kunci yang benar dibuka pada portal yang keliru — keadaan yang tidak menghasilkan
-- satu pun tanda lain (`R-20`). Menyamakan keduanya menjadi "daftar kosong" akan
-- menyembunyikannya sepenuhnya.
SELECT 1
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE c.CLAIMID = :1
 FETCH NEXT 1 ROWS ONLY

-- name: print_pre_dla
-- Panel "Print Pre DLA" — Pre-DLA satu klaim yang dokumennya SUDAH terlampir.
--
-- Bind: :1 kunci klaim (`T_CLAIM_PNC.CLAIMID`)
--
-- Sumber: `RDB List/GetPreDLAList-SQL.xml`, dikirim Work Owner 2026-09-26.
--
-- ============================================================================
-- BARIS DI SINI LEBIH SEDIKIT DARIPADA PRE-DLA YANG BENAR-BENAR ADA
-- ============================================================================
--
-- Gabungan ke kedua tabel lampiran Pega bukan hiasan — ia MENYARING. Pre-DLA yang belum
-- punya berkas lampiran berkategori `DLA`, atau yang nama berkasnya tidak berpola
-- seperti di bawah, tidak muncul sama sekali.
--
-- Itu memang guna panelnya: daftar Pre-DLA yang dokumennya sudah siap dikirim, bukan
-- daftar seluruh Pre-DLA. Perilakunya dibawa apa adanya (`P-5`), dan selisih jumlah
-- barisnya terhadap tab Pre DLA dinyatakan di layar supaya tidak terbaca sebagai
-- kerusakan.
--
-- ============================================================================
-- PENCOCOKAN NAMA BERKAS MEMAKAI POSISI KARAKTER, DAN ITU RAPUH
-- ============================================================================
--
--   SUBSTR(PXATTACHNAME, -15, 11) = NODLA
--
-- Sebelas karakter, dihitung mundur lima belas karakter dari ujung nama berkas. Pola itu
-- pecah begitu konvensi penamaan lampiran berubah — dan pecahnya TIDAK menghasilkan
-- galat, hanya panel yang kosong. Ia dibawa apa adanya karena mengubahnya mengubah baris
-- mana yang muncul, dan itu selisih yang belum diminta siapa pun.
--
-- `PXATTACHNAME` sengaja TIDAK dikualifikasi nama tabelnya, persis seperti kueri lama.
-- Kolom itu ada di salah satu dari kedua tabel lampiran, dan export tidak memuat DDL
-- keduanya (`R-08`). Menebak tabelnya berisiko memilih yang salah; membiarkannya
-- membuat Oracle menyelesaikannya persis seperti hari ini.
--
-- ============================================================================
-- PEMETAAN KOLOM — alias Pega -> alias di sini
-- ============================================================================
--
--   NODLA          "NO_DLA"        ADVICE_NO        NO DLA
--   DLAREINSURER   "DLAReinsurer"  REINSURER        DLA REINSURER
--   TIPEDLA        "DLAType"       ADVICE_TYPE      TIPE DLA
--   TGLKIRIM       "TglDLA"     (!) SENT_DATE       Tgl Kirim
--   NVL(ISKIRIM)   "IsDLA"      (!) SENT            Terkirim
--   a.PZINSKEY     "Currency"   (!) ATTACHMENT_KEY  (tidak digambar)
--
-- Tiga tanda (!): `"TglDLA"` berarti tanggal KIRIM, bukan tanggal Pre-DLA; `"IsDLA"`
-- berarti penanda terkirim; dan `"Currency"` berarti kunci lampiran, bukan mata uang.
--
-- Tiga kolom yang diambil kueri lama TIDAK dibawa — `objectid`, `objectcoverageid`, dan
-- `adjustmentid`, beralias `"pyID"`, `"DLAStream"`, dan `"Count"`. Tidak satu pun
-- digambar panelnya, dan ketiganya hanya dipakai jalur tulis yang belum dibangun.
SELECT c.NODLA                          AS ADVICE_NO,
       c.DLAREINSURER                   AS REINSURER,
       c.TIPEDLA                        AS ADVICE_TYPE,
       c.TGLKIRIM                       AS SENT_DATE,
       NVL(c.ISKIRIM, '0')              AS SENT,
       a.PZINSKEY                       AS ATTACHMENT_KEY
  FROM POOLDATA.T_PREDLALIST c
  JOIN DATAPEGA.PC_LINK_ATTACHMENT b
    ON b.PXLINKEDREFFROM = c.CLAIMID
  JOIN DATAPEGA.PC_DATA_WORKATTACH a
    ON a.PZINSKEY = b.PXLINKEDREFTO
   AND a.PXREFOBJECTKEY = c.CLAIMID
 WHERE c.CLAIMID = :1
   AND b.PYCATEGORY = 'DLA'
   AND SUBSTR(PXATTACHNAME, -15, 11) = c.NODLA
 ORDER BY c.NODLA

-- name: print_pre_dla_diagnosa
-- Menghitung berapa baris Pre-DLA yang LOLOS tiap tahap penyaring panel "Print Pre DLA".
--
-- Bind: :1 kunci klaim
--
-- Ia dipakai HANYA perintah pemeriksaan kesiapan, tidak oleh layar. Alasannya satu: panel
-- yang kosong punya empat sebab berbeda yang semuanya tampak sama, sehingga menebak
-- sebabnya memakan waktu lebih lama daripada menghitungnya.
--
-- ============================================================================
-- TAHAP TERAKHIR YANG PALING SERING MENGGUGURKAN SEMUANYA
-- ============================================================================
--
--   SUBSTR(PXATTACHNAME, -15, 11) = NODLA
--
-- Ia hanya dapat cocok bila `NODLA` tepat **11 karakter** DAN nama berkasnya berakhir
-- dengan `NODLA` ditambah tepat **4 karakter** — misalnya `.pdf`:
--
--   nodla 11 kar + ".pdf"   -> cocok
--   nodla 11 kar + ".jpeg"  -> TIDAK (ekornya 5 karakter)
--   nodla 12 kar + ".pdf"   -> TIDAK (tergeser satu karakter)
--   nodla 13 kar + ".pdf"   -> TIDAK
--   "… (1).pdf"             -> TIDAK
--
-- Ketidakcocokan itu **tidak menghasilkan galat**. Ia hanya menghasilkan panel kosong.
-- Karena itu panjang `NODLA` ikut dihitung di sini: bila kolom NODLA_11_KARAKTER bernilai
-- nol sementara PRE_DLA_ADA tidak, sebabnya sudah pasti dan tidak perlu ditelusuri lagi.
--
-- `FROM DUAL` sengaja DIHINDARI meski kueri ini hanya diagnosa (`D-20`): satu
-- pengecualian portabilitas yang dibiarkan masuk akan diikuti yang berikutnya.
SELECT COUNT(*)                                            AS PRE_DLA_ADA,
       COUNT(CASE WHEN LENGTH(c.NODLA) = 11
                  THEN 1 END)                              AS NODLA_11_KARAKTER,
       COUNT(CASE WHEN EXISTS (SELECT 1
                                 FROM DATAPEGA.PC_LINK_ATTACHMENT b
                                WHERE b.PXLINKEDREFFROM = c.CLAIMID)
                  THEN 1 END)                              AS PUNYA_LAMPIRAN,
       COUNT(CASE WHEN EXISTS (SELECT 1
                                 FROM DATAPEGA.PC_LINK_ATTACHMENT b
                                WHERE b.PXLINKEDREFFROM = c.CLAIMID
                                  AND b.PYCATEGORY = 'DLA')
                  THEN 1 END)                              AS KATEGORI_DLA
  FROM POOLDATA.T_PREDLALIST c
 WHERE c.CLAIMID = :1

-- name: mark_pre_dla_sent
-- Menandai SATU Pre-DLA sebagai terkirim beserta tanggalnya.
--
-- Bind: :1 kunci klaim · :2 nomor Pre-DLA
--
-- Sumber: `RDB List/GetPreDLAList-SQL.xml` bagian `pySaveSQL`, dijalankan
-- `Activity/SetTglKirimPreDLA_Act-Act.xml` lewat `RDB-Save`.
--
-- ============================================================================
-- SATU-SATUNYA KUERI TULIS DI MODUL INI — DAN IA MENYENTUH TABEL MILIK PEGA
-- ============================================================================
--
-- `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel.
-- `POOLDATA.T_PREDLALIST` masih ditulis Pega, sehingga kueri ini membuat keduanya menjadi
-- penulis. Itu keputusan Work Owner 2026-09-27, diambil setelah keberatannya disampaikan.
--
-- Yang mengimbanginya ada di klausa `WHERE`: baris yang SUDAH terkirim tidak disentuh.
-- Dengan begitu penandaan dari sini tidak pernah menimpa tanggal kirim yang sudah dicatat
-- Pega, dan menekan tombol dua kali tidak menghasilkan dua tanggal yang berbeda.
--
-- ============================================================================
-- TIGA PENYIMPANGAN DARI KUERI PEGA, SELURUHNYA DISENGAJA
-- ============================================================================
--
-- **1. `CLAIMID` ikut menyaring.** Kueri Pega hanya menyaring `WHERE NODLA = …`, tanpa
--    klaimnya. Bila satu nomor Pre-DLA dipakai dua klaim, penandaan itu mengenai
--    KEDUANYA — dan kerusakannya tidak menghasilkan galat apa pun.
--
--    Ini bukan rekaan bentuk baru: `RDB List/UpdatePREDLAList-SQL.xml` menulis tabel yang
--    SAMA dengan `WHERE claimid = … AND nodla = …`. Pega sendiri melakukannya di jalur
--    lain; yang di sini yang menyimpang.
--
-- **2. `TGLKIRIM` diisi `CURRENT_TIMESTAMP`, bukan nilai yang dikirim layar.** Kueri Pega
--    menerima `{updatetgldla.Date}`, dan `Section/PrintPreDLA-Section.xml:104789`
--    menunjukkan layar mengirim `tmptglkirim = .TglDLA` — yaitu **tanggal kirim yang baru
--    saja ditampilkan**. Pada Pre-DLA yang belum pernah dikirim, kolom itu KOSONG.
--
--    Jadi menekan "Kirim Pre DLA" pada Pre-DLA baru akan menandainya terkirim dengan
--    tanggal kirim KOSONG. Itu kehilangan informasi yang tidak dapat dipulihkan, dan
--    tidak ditiru. `UpdatePREDLAList` pada tabel yang sama memakai `sysdate`; pola itulah
--    yang diikuti.
--
-- **3. `CURRENT_TIMESTAMP`, bukan `SYSDATE`.** Portabilitas (`D-20`). Keduanya sama di
--    Oracle.
--
-- Waktunya diambil dari BASIS DATA, bukan dari jam aplikasi. Kolom yang sama ditulis Pega
-- dengan `sysdate`; mengisi sebagian barisnya dengan jam aplikasi akan membuat satu kolom
-- memuat waktu dari dua jam yang berbeda, dan selisih antarkeduanya tidak terlihat.
UPDATE POOLDATA.T_PREDLALIST
   SET ISKIRIM  = '1',
       TGLKIRIM = CURRENT_TIMESTAMP
 WHERE CLAIMID = :1
   AND NODLA   = :2
   AND (ISKIRIM IS NULL OR ISKIRIM <> '1')

-- name: advice_for_sending_pla
-- Satu PLA beserta keterangan reasuradur dan klaimnya, untuk menyusun suratnya.
--
-- Bind: :1 kunci klaim · :2 nomor PLA
--
-- `TYPE` pada gabungan ke `T_REINSURER` adalah **huruf pertama nomor dokumen**. Itu bukan
-- tebakan: `RDB List/GetDataPreDLA-SQL.xml` memakai `substr(a.NODLA,0,1) = TYPE` pada
-- sub-kueri yang membaca `LOGIN` dan `COUNTRY` dari tabel yang sama.
--
-- Gabungannya `LEFT JOIN`, bukan `JOIN`. Reasuradur yang belum terdaftar di master tidak
-- boleh menghentikan pengiriman — alamatnya ada di baris dokumennya sendiri, dan master
-- itu justru yang akan DIISI setelah suratnya terkirim.
SELECT p.NOPLA                       AS ADVICE_NO,
       p.TIPEPLA                     AS ADVICE_TYPE,
       p.PLAREINSURER                AS REINSURER,
       p.REINSCODE                   AS REINSURER_ID,
       p.EMAILPLA                    AS EMAIL,
       NVL(p.ISKIRIM, '0')           AS SENT,
       r.LOGIN                       AS REINSURER_LOGIN,
       r.COUNTRY                     AS COUNTRY,
       c.CLAIMNO                     AS CLAIM_NO,
       c.NOPOLIS                     AS POLICY_NO,
       c.QQNAME                      AS INSURED,
       c.BUSINESSNAME                AS BUSINESS,
       c.DATEOFLOSS                  AS LOSS_DATE
  FROM POOLDATA.T_PLALIST p
  JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = p.CLAIMID
  LEFT JOIN POOLDATA.T_REINSURER r
    ON r.REINSURERID = p.REINSCODE
   AND r.TYPE = SUBSTR(p.NOPLA, 1, 1)
 WHERE p.CLAIMID = :1
   AND p.NOPLA   = :2

-- name: advice_for_sending_dla
-- Sama dengan advice_for_sending_pla, untuk tabel DLA.
--
-- Urutan dan nama aliasnya WAJIB sama persis: satu pemindai melayani keduanya, dan kolom
-- yang bergeser pada salah satunya tidak menghasilkan galat — hanya isi yang tertukar.
SELECT d.NODLA                       AS ADVICE_NO,
       d.TIPEDLA                     AS ADVICE_TYPE,
       d.DLAREINSURER                AS REINSURER,
       d.REINSCODE                   AS REINSURER_ID,
       d.EMAILDLA                    AS EMAIL,
       NVL(d.ISKIRIM, '0')           AS SENT,
       r.LOGIN                       AS REINSURER_LOGIN,
       r.COUNTRY                     AS COUNTRY,
       c.CLAIMNO                     AS CLAIM_NO,
       c.NOPOLIS                     AS POLICY_NO,
       c.QQNAME                      AS INSURED,
       c.BUSINESSNAME                AS BUSINESS,
       c.DATEOFLOSS                  AS LOSS_DATE
  FROM POOLDATA.T_DLALIST d
  JOIN POOLDATA.T_CLAIM_PNC c
    ON c.CLAIMID = d.CLAIMID
  LEFT JOIN POOLDATA.T_REINSURER r
    ON r.REINSURERID = d.REINSCODE
   AND r.TYPE = SUBSTR(d.NODLA, 1, 1)
 WHERE d.CLAIMID = :1
   AND d.NODLA   = :2

-- name: attachments_for_claim
-- Berkas lampiran satu klaim pada satu kategori, beserta ISINYA.
--
-- Bind: :1 kunci klaim · :2 kategori (`PLA` atau `DLA`).
--
-- Sumbernya `POOLDATA.DATA_ATTACHFILE` — tabel yang dibaca `UpdateDetailPLA2` lewat
-- `GetAttachmentFromDB_Sql` (`tempAttachment2`: ATTACHNAME, ATTACHMIMETYPE, ATTACHFILE).
--
-- Bukan `DATAPEGA.PC_DATA_WORKATTACH`: tabel itu tidak punya kolom isi berkas — isinya ada di
-- blob internal Pega `PZPVSTREAM` yang tidak dapat dibaca SQL. Kueri sebelumnya memakai
-- `PYATTACHSTREAM` dan ditolak ORA-00904 (SEND ALL PLA PNCN.26.43, 2026-10-10). Berkas
-- kategori PLA di tabel Pega itu adalah PDF PLA hasil `AttachAsPDFC`; di sistem ini PDF-nya
-- dibuat ulang dan dilampirkan oleh pemanggil (SEND ALL PLA), bukan dibaca dari sini.
--
-- **Isinya ikut terbaca.** Itu disengaja dan berbeda dari panel "Print Pre DLA", yang
-- hanya mengambil kuncinya: di sini isinya memang dibutuhkan, karena ia yang dilampirkan
-- ke surat. Batasnya disadari — satu surat memuat dokumen pendukung satu klaim, bukan
-- arsip.
SELECT a.ATTACHNAME                  AS NAME,
       a.ATTACHMIMETYPE              AS MIME_TYPE,
       a.ATTACHFILE                  AS CONTENT
  FROM POOLDATA.DATA_ATTACHFILE a
 WHERE a.IDPEGA = :1
   AND UPPER(TRIM(a.CATEGORY)) = UPPER(:2)
   AND a.ATTACHFILE IS NOT NULL
 ORDER BY a.ATTACHNAME, a.DATAID

-- name: mark_advice_sent_pla
-- Menandai satu PLA terkirim beserta tanggal dan alamat tujuannya.
--
-- Bind: :1 kunci klaim · :2 nomor PLA · :3 alamat tujuan
--
-- Sumber: `RDB List/UpdatePLAList-SQL.xml`. Bentuknya ditiru, termasuk pengisian
-- `TGLTERIMAPLA` dengan waktu yang sama — Pega memakai `sysdate` untuk keduanya.
--
-- Penyaring `ISKIRIM <> '1'` DITAMBAHKAN, dengan alasan yang sama seperti pada Pre-DLA:
-- selama Pega masih menulis tabel ini (`P-1`), penandaan dari sini tidak boleh menimpa
-- tanggal kirim yang sudah tercatat.
UPDATE POOLDATA.T_PLALIST
   SET ISKIRIM      = '1',
       TGLKIRIM     = CURRENT_TIMESTAMP,
       TGLTERIMAPLA = CURRENT_TIMESTAMP,
       EMAILPLA     = :3
 WHERE CLAIMID = :1
   AND NOPLA   = :2
   AND (ISKIRIM IS NULL OR ISKIRIM <> '1')

-- name: mark_advice_sent_dla
-- Menandai satu DLA terkirim. Bind dan bentuknya sama dengan jalur PLA.
UPDATE POOLDATA.T_DLALIST
   SET ISKIRIM      = '1',
       TGLKIRIM     = CURRENT_TIMESTAMP,
       TGLTERIMADLA = CURRENT_TIMESTAMP,
       EMAILDLA     = :3
 WHERE CLAIMID = :1
   AND NODLA   = :2
   AND (ISKIRIM IS NULL OR ISKIRIM <> '1')

-- name: update_reinsurer_email
-- Memperbarui alamat surel satu reasuradur pada master.
--
-- Bind: :1 alamat · :2 REINSURERID · :3 REINSURERNAME · :4 TYPE
--
-- ============================================================================
-- MENGGANTIKAN `Database/UPDATEREAS.prc` — TIGA CABANGNYA MENJADI SATU
-- ============================================================================
--
-- Procedure lamanya bercabang tiga, dan ketiganya berakhir pada satu hal yang sama:
-- alamat surel reasuradur itu diperbarui.
--
--   1  baris (ID, NAMA, TYPE) ada          -> UPDATE alamatnya
--   2  ada dengan TYPE '1'                 -> UPDATE alamat DAN pindahkan TYPE-nya
--   3  tidak ada sama sekali               -> INSERT baris baru
--
-- Cabang 2 dan 3 TIDAK dibawa, dan itu keputusan yang perlu disebut alasannya.
--
-- **Cabang 2 mengubah `TYPE` baris yang sudah ada.** `TYPE` adalah bagian kunci alami
-- baris itu — huruf pertama nomor dokumen — dan memindahkannya berarti baris yang semula
-- melayani satu jenis dokumen kini melayani jenis lain. Pengiriman surat bukan tempat
-- yang tepat untuk mengubah kunci master.
--
-- **Cabang 3 membuat baris master baru.** Master reasuransi diisi `F-4`, bukan oleh
-- tombol kirim. Membuat baris master sebagai efek samping pengiriman surat berarti
-- mengisi master dari jalur yang tidak punya validasi master sama sekali — termasuk
-- `COUNTRY` yang di procedure lama dicari lewat `SELECT ID INTO` tanpa penanganan bila
-- negaranya tidak ditemukan.
--
-- Akibat yang diterima: reasuradur yang belum terdaftar di master tidak akan tercatat
-- alamatnya. Suratnya TETAP terkirim — alamatnya diambil dari baris dokumennya, bukan
-- dari master — dan nol baris terpengaruh di sini bukan galat.
--
-- Empat `COMMIT` di dalam procedure lamanya juga tidak dibawa: pernyataan ini berjalan di
-- dalam transaksi pemanggilnya (`D-68`).
UPDATE POOLDATA.T_REINSURER
   SET EMAIL = :1
 WHERE REINSURERID   = :2
   AND REINSURERNAME = :3
   AND TYPE          = :4
