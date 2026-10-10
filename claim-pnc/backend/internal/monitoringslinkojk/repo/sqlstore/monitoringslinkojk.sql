-- Kueri modul Monitoring SLINK OJK (`MENU_ID 78`, harness `MonitoringSLINKOJK`).
--
-- ============================================================================
-- SELURUHNYA MEMBACA — TIDAK ADA SATU PUN PERNYATAAN YANG MENULIS
-- ============================================================================
--
-- `POOLDATA.T_CLAIM_SLIK_OJK` diisi sistem lama pada jalur AKSEPTASI, bukan dari layar
-- ini: pemanggil `InsertDataSlikOJKF06-SQL.xml` adalah `InsertAdjustmentList-Act.xml` dan
-- `InsertAdjustmentListKredit-Act.xml`. Selama masa paralel, satu tabel hanya boleh
-- ditulis satu sistem (`P-1`).
--
-- Berkas ini karena itu tidak boleh memuat INSERT, UPDATE, DELETE, maupun MERGE. Larangan
-- itu DIUJI di query_test.go, bukan sekadar dituliskan di sini.
--
-- ============================================================================
-- PERANGKAIAN SQL WARISAN DIGANTI PARAMETER BINDING
-- ============================================================================
--
-- Kedua kueri lama menerima penyaringnya sebagai POTONGAN TEKS SQL yang dirangkai di
-- properti klipboard lalu disisipkan mentah:
--
--     GetDataSlinkAllFOGF06 :  ... {ASIS:DetailTempSlink.NoteKasir}
--                              ... {ASIS:DetailTempSlink.BusinessID}
--     GetDataSlinkAllFOG    :  ... {ASIS:DetailTempSlink.NoteKasir}
--
-- dan isinya dirangkai di `GetTempDataD01-Act.xml` begini:
--
--     "and to_date(B.REGISTERDATE_1,'yyyy/mm/dd')>=to_date('" + DateOfLoss + "','dd/mm/yyyy') ..."
--
-- Nilai dari layar masuk langsung ke dalam teks SQL. Itu celah injeksi sekaligus
-- penghalang portabilitas (utang teknis §4.5), dan di sini digantikan parameter binding
-- tanpa perkecualian (`08-TECHNICAL-STRATEGY.md` §4.3).
--
-- ============================================================================
-- KENAPA `TO_DATE` PADA KOLOM TANGGAL TIDAK DIBAWA
-- ============================================================================
--
-- `to_date(B.REGISTERDATE_1,'yyyy/mm/dd')` memaksa Oracle mengubah kolom DATE menjadi
-- teks memakai NLS_DATE_FORMAT sesi, lalu menguraikannya kembali. Dua akibatnya nyata:
--
--   1. Hasilnya bergantung pada NLS sesi. Sesi dengan format lain menghasilkan
--      ORA-01861 — atau, lebih buruk, tanggal yang terurai SALAH tanpa galat.
--   2. Fungsi di sisi kolom membuat index atas REGISTERDATE_1 tidak terpakai. Pada tabel
--      berisi puluhan juta baris (`D-10`), itu perbedaan antara pemindaian rentang dan
--      pemindaian penuh.
--
-- Di sini kolomnya dibandingkan LANGSUNG dengan parameter bertipe tanggal. Batas atasnya
-- dihitung sebagai "sebelum hari berikutnya" — bukan `<=` — supaya klaim yang
-- diregistrasi pada pukul berapa pun di hari "Sampai" tetap ikut. Dengan `<=` terhadap
-- tengah malam, seluruh klaim hari terakhir hilang dari laporan tanpa satu pun tanda.
--
-- ============================================================================
-- TABEL KLAIM: T_CLAIMLIST_ADMIN, BUKAN PC_ASM_FW_GCNMFW_WORK
-- ============================================================================
--
-- Kedua kueri lama menulis `DATAPEGA.PC_ASM_FW_GCNMFW_WORK b`, tetapi kolom yang
-- benar-benar dipakainya — `REGISTERDATE_1`, `BUSINESSTYPE`, `PYID`, `PZINSKEY`,
-- `POLICYNO` — adalah kolom tabel DATAR `POOLDATA.T_CLAIMLIST_ADMIN`
-- (`docs/kolom-t-claimlist-admin.md`, dibaca dari katalog Oracle).
--
-- Work Owner menetapkan tabel datar itulah yang menggantikan tabel kerja Pega beserta
-- tabel penugasannya; modul My Inbox sudah memakainya. Modul ini mengikuti.
--
-- ============================================================================
-- DUA KOLOM BERNAMA `CLAIMID` YANG BERISI HAL BERBEDA — baca sebelum menyunting join
-- ============================================================================
--
-- Kedua segmen bergabung ke tabel klaim lewat kolom yang BERBEDA, dan itu bukan
-- kelalaian — kedua kueri Pega pun begitu:
--
--     GetDataSlinkAllFOGF06 (D01) :  where a.claimid = b.pyid
--     GetDataSlinkAllFOG    (F06) :  where B.PZINSKEY = A.CLAIMID
--
-- Sebabnya: kolom bernama `CLAIMID` di kedua tabel sumber TIDAK berisi hal yang sama.
--
--     POOLDATA.T_CLAIM_SLIK_OJK.CLAIMID     berisi NOMOR KLAIM   -> cocok dengan PYID
--     POOLDATA.T_CLAIM_OBJECTLIST.CLAIMID   berisi KUNCI TEKNIS  -> cocok dengan PZINSKEY
--
-- Yang kedua berbentuk `ASM-FW-GCNMFW-WORK PNC-xxxx`: nama kelas internal Pega tertanam
-- di dalam kunci data bisnis — utang teknis §4.1 yang `D-22` dan `D-71` hapus untuk klaim
-- baru, tetapi masih melekat pada baris warisan.
--
-- Menukar keduanya TIDAK menghasilkan galat. Join-nya tetap sah, hanya mengembalikan NOL
-- BARIS — dan nol baris pada layar pemantauan laporan regulator terbaca sebagai "tidak
-- ada yang perlu dilaporkan". Itulah sebabnya keduanya ditulis terpisah di sini alih-alih
-- disatukan menjadi satu kueri berparameter.

-- ============================================================================
-- SEGMEN D01 — FASILITAS KREDIT
-- ============================================================================
--
-- Sumber: `RDB List/GetDataSlinkAllFOGF06-SQL.xml` — namanya menyebut F06, isinya D01.
-- Lihat catatan "PENAMAAN DI AREA INI TERBALIK" pada columns.go.
--
-- Urutan SELECT-nya SENGAJA sama dengan urutan katalog d01ExportColumns, sehingga
-- pemetaan kolom → katalog dapat diperiksa dengan membacanya berdampingan.
--
-- `tanggalkondisi` diambil DUA KALI dengan alias berbeda. Itu ada di kueri aslinya dan
-- direplikasi: kolom layar "Tanggal Pembayaran" memang menampilkan tanggal kondisi,
-- karena tabelnya tidak menyimpan tanggal pembayaran sama sekali. Lihat d01Columns.

-- name: d01_rows
SELECT s.CLAIMID            AS NO_KLAIM,
       s.CONTRACTNO         AS CONTRACT_NO,
       s.NOREKFASILITAS     AS NOMOR_REKENING_FASILITAS,
       s.NOCIFDEBITUR       AS NO_CIF_DEBITUR,
       s.KODEJENISFASILITAS AS KODE_JENIS_FASILITAS,
       s.SUMBERDANA         AS SUMBER_DANA,
       s.TANGGALMULAI       AS START_POLIS,
       s.TANGGALAKHIR       AS END_POLIS,
       s.SUKUBUNGA          AS SUKU_BUNGA,
       s.KODEVALUTA         AS KODE_VALUTA,
       s.NILAIMATAUANGASAL  AS NILAI_MATA_UANG_ASAL,
       s.TANGGALKONDISI     AS TANGGAL_PEMBAYARAN,
       s.KODEKOLEKTABILITAS AS KODE_KOLEKTIBILITAS,
       s.TANGGALMACET       AS TANGGAL_MACET,
       s.KODESEBABMACET     AS KODE_SEBAB_MACET,
       s.TUNGGAKAN          AS TUNGGAKAN,
       s.NOMINAL            AS JUMLAH_KEWAJIBAN,
       s.JUMLAHHARITUNGGAKAN AS JUMLAH_HARI_TUNGGAKAN,
       s.TANGGALKONDISI     AS TANGGAL_KONDISI,
       s.KODEKONDISI        AS KODE_KONDISI,
       s.KETERANGAN         AS KETERANGAN,
       s.KODEKANTORCABANG   AS KODE_KANTOR_CABANG,
       s.OPERASIDATA        AS OPERASI_DATA,
       s.NOKTP              AS NO_KTP,
       s.NPWPPERUSAHAAN     AS NPWP_PERUSAHAAN,
       s.NOPOLIS            AS NO_POLIS
  FROM POOLDATA.T_CLAIM_SLIK_OJK s
  JOIN POOLDATA.T_CLAIMLIST_ADMIN k
    ON k.PYID = s.CLAIMID
 WHERE (:1 IS NULL OR k.REGISTERDATE_1 >= :2)
   AND (:3 IS NULL OR k.REGISTERDATE_1 < :4)
   AND (:5 IS NULL
        OR (:6 = 'INCLUDE' AND UPPER(TRIM(k.BUSINESSNAME)) LIKE :7)
        OR (:8 = 'EXCLUDE' AND (UPPER(TRIM(k.BUSINESSNAME)) NOT LIKE :9 OR k.BUSINESSNAME IS NULL)))
 ORDER BY k.REGISTERDATE_1 DESC, s.CLAIMID, s.CONTRACTNO
OFFSET :10 ROWS FETCH NEXT :11 ROWS ONLY

-- Jumlah SELURUH baris yang cocok — untuk bilah halaman.
--
-- Penyaringnya WAJIB sama persis dengan d01_rows. Satu syarat yang berbeda membuat bilah
-- halaman menjanjikan halaman yang isinya kosong, dan tidak ada apa pun yang menandainya.
-- query_test.go memeriksa keduanya memuat rangkaian syarat yang sama.

-- name: d01_count
SELECT COUNT(1) AS TOTAL
  FROM POOLDATA.T_CLAIM_SLIK_OJK s
  JOIN POOLDATA.T_CLAIMLIST_ADMIN k
    ON k.PYID = s.CLAIMID
 WHERE (:1 IS NULL OR k.REGISTERDATE_1 >= :2)
   AND (:3 IS NULL OR k.REGISTERDATE_1 < :4)
   AND (:5 IS NULL
        OR (:6 = 'INCLUDE' AND UPPER(TRIM(k.BUSINESSNAME)) LIKE :7)
        OR (:8 = 'EXCLUDE' AND (UPPER(TRIM(k.BUSINESSNAME)) NOT LIKE :9 OR k.BUSINESSNAME IS NULL)))

-- Ekspor segmen D01 — kueri yang SAMA tanpa paginasi.
--
-- Ia berkas tersendiri, bukan d01_rows dengan halaman raksasa: `OFFSET … FETCH` memaksa
-- Oracle menyusun urutan lengkap lebih dulu, sedangkan ekspor membacanya mengalir.

-- name: d01_export
SELECT s.CLAIMID            AS NO_KLAIM,
       s.CONTRACTNO         AS CONTRACT_NO,
       s.NOREKFASILITAS     AS NOMOR_REKENING_FASILITAS,
       s.NOCIFDEBITUR       AS NO_CIF_DEBITUR,
       s.KODEJENISFASILITAS AS KODE_JENIS_FASILITAS,
       s.SUMBERDANA         AS SUMBER_DANA,
       s.TANGGALMULAI       AS START_POLIS,
       s.TANGGALAKHIR       AS END_POLIS,
       s.SUKUBUNGA          AS SUKU_BUNGA,
       s.KODEVALUTA         AS KODE_VALUTA,
       s.NILAIMATAUANGASAL  AS NILAI_MATA_UANG_ASAL,
       s.TANGGALKONDISI     AS TANGGAL_PEMBAYARAN,
       s.KODEKOLEKTABILITAS AS KODE_KOLEKTIBILITAS,
       s.TANGGALMACET       AS TANGGAL_MACET,
       s.KODESEBABMACET     AS KODE_SEBAB_MACET,
       s.TUNGGAKAN          AS TUNGGAKAN,
       s.NOMINAL            AS JUMLAH_KEWAJIBAN,
       s.JUMLAHHARITUNGGAKAN AS JUMLAH_HARI_TUNGGAKAN,
       s.TANGGALKONDISI     AS TANGGAL_KONDISI,
       s.KODEKONDISI        AS KODE_KONDISI,
       s.KETERANGAN         AS KETERANGAN,
       s.KODEKANTORCABANG   AS KODE_KANTOR_CABANG,
       s.OPERASIDATA        AS OPERASI_DATA,
       s.NOKTP              AS NO_KTP,
       s.NPWPPERUSAHAAN     AS NPWP_PERUSAHAAN,
       s.NOPOLIS            AS NO_POLIS
  FROM POOLDATA.T_CLAIM_SLIK_OJK s
  JOIN POOLDATA.T_CLAIMLIST_ADMIN k
    ON k.PYID = s.CLAIMID
 WHERE (:1 IS NULL OR k.REGISTERDATE_1 >= :2)
   AND (:3 IS NULL OR k.REGISTERDATE_1 < :4)
   AND (:5 IS NULL
        OR (:6 = 'INCLUDE' AND UPPER(TRIM(k.BUSINESSNAME)) LIKE :7)
        OR (:8 = 'EXCLUDE' AND (UPPER(TRIM(k.BUSINESSNAME)) NOT LIKE :9 OR k.BUSINESSNAME IS NULL)))
 ORDER BY k.REGISTERDATE_1 DESC, s.CLAIMID, s.CONTRACTNO

-- ============================================================================
-- SEGMEN F06 — DEBITUR INDIVIDU
-- ============================================================================
--
-- Sumber: `RDB List/GetDataSlinkAllFOG-SQL.xml` — namanya menyebut FOG, isinya F06.
--
-- Ia membaca data klaim SUMBERNYA (`T_CLAIM_OBJECTLIST`), bukan tabel SLIK yang sudah
-- terisi. Itulah sebab kedua segmen punya penyaring yang sama tetapi isi yang berbeda:
-- D01 memantau apa yang SUDAH tersusun, F06 memantau apa yang AKAN disusun.
--
-- ---------------------------------------------------------------------------
-- KOREKSI 2026-09-27 — kueri ini sempat dibangun jauh terlalu kecil
-- ---------------------------------------------------------------------------
--
-- Sebelumnya ia hanya memilih **9 kolom**. `GetDataSlinkAllFOG-SQL.xml` memilih **30**,
-- dan sebagian besar di antaranya adalah subkueri berkorelasi ke `T_GENERAL`,
-- `T_CLAIM_PNC`, `T_CLAIM_ADJUSTMENT`, dan `T_CLAIM_OBJECTCOVERAGE` — bukan kolom
-- `T_CLAIM_OBJECTLIST` begitu saja. Kedua puluh satu kolom yang hilang itulah yang
-- membuat grid F06 kosong hampir seluruhnya.
--
-- Tabel klaimnya tetap `POOLDATA.T_CLAIMLIST_ADMIN`, bukan tabel kerja Pega yang ditulis
-- kueri lama — lihat catatan "TABEL KLAIM" di atas. Yang dikoreksi adalah daftar
-- kolomnya, bukan keputusan tabelnya.
--
-- Ketiga puluh kolom itu kini dibawa utuh. Dua puluh di antaranya tampil di grid — daftar
-- yang SAMA dengan segmen D01, terbukti dari layar Pega yang berjalan — dan sisanya
-- dipakai berkas ekspor.
--
-- ---------------------------------------------------------------------------
-- TIDAK ADA PENYARING JENIS BISNIS DI SINI — dan itu bukan kelalaian
-- ---------------------------------------------------------------------------
--
-- Layar F06 MENAMPILKAN dropdown "Business Name", tetapi `GetAllDataSumbisSlink` tidak
-- pernah menurunkannya menjadi `DetailTempSlink.BusinessID`, dan kueri ini tidak memuat
-- placeholder-nya sama sekali — hanya `{ASIS:DetailTempSlink.NoteKasir}`, yaitu rentang
-- tanggal. Bandingkan dengan `GetDataSlinkAllFOGF06` (segmen D01) yang memuat keduanya.
--
-- Jadi di Pega, mengubah dropdown itu pada segmen F06 **tidak mengubah hasil apa pun**.
-- Perilakunya direplikasi: dropdown tetap tampil, dan tetap tidak menyaring.
--
-- ---------------------------------------------------------------------------
-- DUA CACAT WARISAN YANG DIREPLIKASI, DAN KENAPA
-- ---------------------------------------------------------------------------
--
-- 1. `'001' AS KODE_KANTOR_CABANG` — kode kantor cabang adalah KONSTANTA di dalam kueri
--    lama. Ia melanggar `D-15` (tidak ada nilai bisnis yang di-hardcode) dan tempatnya
--    adalah master Cabang pada `F-4`. Ia direplikasi karena mengubahnya berarti mengubah
--    isi kolom laporan regulator atas dasar tebakan — dan nilainya diangkat menjadi
--    parameter repo (lihat DefaultBranchCode) supaya ketika masternya tiba, yang berubah
--    hanya satu tempat.
--
-- 2. `OPERASI_DATA` diturunkan dari pencocokan teks `LIKE '%EDM%'` atas `IDPEGA` polis:
--    'U' bila polisnya berasal dari EDM, 'C' bila bukan. Pencocokan substring atas kunci
--    teknis adalah pola yang sama rapuhnya dengan toleransi spreading yang `D-49` butir 1
--    perbaiki. Ia direplikasi dengan alasan yang sama seperti butir 1 di atas.
--
-- Keduanya dicatat sebagai calon butir `P-5` di `docs/keputusan-implementasi.md`.
--
-- ---------------------------------------------------------------------------
-- TIGA SUBKUERI BERSARANG PADA KUERI LAMA TIDAK DIBAWA
-- ---------------------------------------------------------------------------
--
-- Kueri lama memanggil `T_GENERAL` di dalam `CASE` yang di dalamnya ada `CASE` lagi,
-- masing-masing dengan subkueri berkorelasi — lima pembacaan tabel polis untuk MENGISI
-- SATU kolom, per baris. Di sini bentuknya diratakan menjadi satu subkueri skalar yang
-- menghasilkan nilai yang sama.
--
-- Yang tidak dibawa sama sekali: `rownum AS "JumlahHariTunggakan"`. Kolom itu mengisi
-- "Jumlah Hari Tunggakan" dengan NOMOR URUT BARIS — angka yang berubah setiap kali
-- urutannya berubah dan tidak ada hubungannya dengan tunggakan. Ia tidak tampil di grid
-- F06 maupun di berkas ekspornya, sehingga tidak ada yang hilang dengan membuangnya.
-- `D-20` juga melarang `ROWNUM`, sehingga membawanya menuntut padanan portabel untuk
-- angka yang tidak berarti apa pun.

-- name: f06_rows
SELECT k.PYID              AS NO_KLAIM,
       o.CONTRACTNO        AS CONTRACT_NO,
       o.OBJECTNAME        AS OBJECT_NAME,
       o.NOMORREKENINGFASILITAS AS NOMOR_REKENING_FASILITAS,
       o.NOMORCIFDEBITUR   AS NO_CIF_DEBITUR,
       o.KODEJENISFASILITAS AS KODE_JENIS_FASILITAS,
       o.SUMBERDANA        AS SUMBER_DANA,
       (SELECT MAX(g.STARTDATE) FROM POOLDATA.T_GENERAL g
         WHERE g.NOPOLIS = k.POLICYNO)                     AS START_POLIS,
       (SELECT MAX(g.ENDDATE) FROM POOLDATA.T_GENERAL g
         WHERE g.NOPOLIS = k.POLICYNO)                     AS END_POLIS,
       o.SUKUBUNGA         AS SUKU_BUNGA,
       (SELECT MAX(pnc.CURRENCY) FROM POOLDATA.T_CLAIM_PNC pnc
         WHERE pnc.CLAIMID = k.PZINSKEY
           AND pnc.NOPOLIS = k.POLICYNO)                   AS KODE_VALUTA,
       (SELECT MAX(ad.TGLAKSEPTASI) FROM POOLDATA.T_CLAIM_ADJUSTMENT ad
         WHERE ad.CLAIMID = o.CLAIMID
           AND ad.OBJECTID = o.OBJECTID
           AND ad.CONTRACTNO = o.CONTRACTNO)               AS TANGGAL_PEMBAYARAN,
       (SELECT MAX(cc.CURRENCY) FROM POOLDATA.CURRENCY cc
         WHERE cc.ID = (SELECT MAX(c.CURRENCY)
                          FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE c
                         WHERE c.CLAIMID = o.CLAIMID
                           AND c.OBJECTID = o.OBJECTID
                           AND c.CONTRACTNO = o.CONTRACTNO))
       || ' ' ||
       (SELECT MAX(c.SUMTSI) FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE c
         WHERE c.CLAIMID = o.CLAIMID
           AND c.OBJECTID = o.OBJECTID
           AND c.CONTRACTNO = o.CONTRACTNO)                AS NILAI_MATA_UANG_ASAL,
       o.KODEKOLEKTIBILITAS AS KODE_KOLEKTIBILITAS,
       (SELECT MAX(pnc.REGISTERDATE) FROM POOLDATA.T_CLAIM_PNC pnc
         WHERE pnc.CLAIMID = k.PZINSKEY
           AND pnc.NOPOLIS = k.POLICYNO)                   AS TANGGAL_MACET,
       o.KODESEBABMACET    AS KODE_SEBAB_MACET,
       (SELECT SUM(ad.NILAIAKSEPTASI) FROM POOLDATA.T_CLAIM_ADJUSTMENT ad
         WHERE ad.CLAIMID = o.CLAIMID
           AND ad.OBJECTID = o.OBJECTID
           AND ad.CONTRACTNO = o.CONTRACTNO)               AS TUNGGAKAN,
       (SELECT MAX(c.SUMTSI) FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE c
         WHERE c.CLAIMID = o.CLAIMID
           AND c.OBJECTID = o.OBJECTID
           AND c.CONTRACTNO = o.CONTRACTNO)                AS JUMLAH_KEWAJIBAN,
       o.KODEKONDISI       AS KODE_KONDISI,
       o.KETERANGAN        AS KETERANGAN,
       (SELECT MAX(pnc.REGISTERDATE) FROM POOLDATA.T_CLAIM_PNC pnc
         WHERE pnc.CLAIMID = k.PZINSKEY
           AND pnc.NOPOLIS = k.POLICYNO)                   AS TANGGAL_KONDISI,
       :1                  AS KODE_KANTOR_CABANG,
       CASE
         WHEN EXISTS (SELECT 1 FROM POOLDATA.T_GENERAL g
                       WHERE g.NOPOLIS = k.POLICYNO
                         AND g.IDPEGA LIKE '%EDM%')
         THEN 'U'
         ELSE 'C'
       END                 AS OPERASI_DATA,
       o.CUSTOMERTYPE      AS CUSTOMER_TYPE,
       o.OBJECTGENDER      AS JENIS_KELAMIN,
       o.DATEOFBIRTH       AS TANGGAL_LAHIR,
       o.ASMZIPCODE        AS KODE_POS,
       o.ASMADDRESS        AS ALAMAT,
       o.TELFAXNUMBER      AS TELEPON
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
  JOIN POOLDATA.T_CLAIM_OBJECTLIST o
    ON k.PZINSKEY = o.CLAIMID
 WHERE (:2 IS NULL OR k.REGISTERDATE_1 >= :3)
   AND (:4 IS NULL OR k.REGISTERDATE_1 < :5)
 ORDER BY k.REGISTERDATE_1 DESC, o.CLAIMID, o.OBJECTID, o.CONTRACTNO
OFFSET :6 ROWS FETCH NEXT :7 ROWS ONLY

-- name: f06_count
SELECT COUNT(1) AS TOTAL
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
  JOIN POOLDATA.T_CLAIM_OBJECTLIST o
    ON k.PZINSKEY = o.CLAIMID
 WHERE (:1 IS NULL OR k.REGISTERDATE_1 >= :2)
   AND (:3 IS NULL OR k.REGISTERDATE_1 < :4)

-- name: f06_export
SELECT k.PYID              AS NO_KLAIM,
       o.CONTRACTNO        AS CONTRACT_NO,
       o.OBJECTNAME        AS OBJECT_NAME,
       o.NOMORREKENINGFASILITAS AS NOMOR_REKENING_FASILITAS,
       o.NOMORCIFDEBITUR   AS NO_CIF_DEBITUR,
       o.KODEJENISFASILITAS AS KODE_JENIS_FASILITAS,
       o.SUMBERDANA        AS SUMBER_DANA,
       (SELECT MAX(g.STARTDATE) FROM POOLDATA.T_GENERAL g
         WHERE g.NOPOLIS = k.POLICYNO)                     AS START_POLIS,
       (SELECT MAX(g.ENDDATE) FROM POOLDATA.T_GENERAL g
         WHERE g.NOPOLIS = k.POLICYNO)                     AS END_POLIS,
       o.SUKUBUNGA         AS SUKU_BUNGA,
       (SELECT MAX(pnc.CURRENCY) FROM POOLDATA.T_CLAIM_PNC pnc
         WHERE pnc.CLAIMID = k.PZINSKEY
           AND pnc.NOPOLIS = k.POLICYNO)                   AS KODE_VALUTA,
       (SELECT MAX(ad.TGLAKSEPTASI) FROM POOLDATA.T_CLAIM_ADJUSTMENT ad
         WHERE ad.CLAIMID = o.CLAIMID
           AND ad.OBJECTID = o.OBJECTID
           AND ad.CONTRACTNO = o.CONTRACTNO)               AS TANGGAL_PEMBAYARAN,
       (SELECT MAX(cc.CURRENCY) FROM POOLDATA.CURRENCY cc
         WHERE cc.ID = (SELECT MAX(c.CURRENCY)
                          FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE c
                         WHERE c.CLAIMID = o.CLAIMID
                           AND c.OBJECTID = o.OBJECTID
                           AND c.CONTRACTNO = o.CONTRACTNO))
       || ' ' ||
       (SELECT MAX(c.SUMTSI) FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE c
         WHERE c.CLAIMID = o.CLAIMID
           AND c.OBJECTID = o.OBJECTID
           AND c.CONTRACTNO = o.CONTRACTNO)                AS NILAI_MATA_UANG_ASAL,
       o.KODEKOLEKTIBILITAS AS KODE_KOLEKTIBILITAS,
       (SELECT MAX(pnc.REGISTERDATE) FROM POOLDATA.T_CLAIM_PNC pnc
         WHERE pnc.CLAIMID = k.PZINSKEY
           AND pnc.NOPOLIS = k.POLICYNO)                   AS TANGGAL_MACET,
       o.KODESEBABMACET    AS KODE_SEBAB_MACET,
       (SELECT SUM(ad.NILAIAKSEPTASI) FROM POOLDATA.T_CLAIM_ADJUSTMENT ad
         WHERE ad.CLAIMID = o.CLAIMID
           AND ad.OBJECTID = o.OBJECTID
           AND ad.CONTRACTNO = o.CONTRACTNO)               AS TUNGGAKAN,
       (SELECT MAX(c.SUMTSI) FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE c
         WHERE c.CLAIMID = o.CLAIMID
           AND c.OBJECTID = o.OBJECTID
           AND c.CONTRACTNO = o.CONTRACTNO)                AS JUMLAH_KEWAJIBAN,
       o.KODEKONDISI       AS KODE_KONDISI,
       o.KETERANGAN        AS KETERANGAN,
       (SELECT MAX(pnc.REGISTERDATE) FROM POOLDATA.T_CLAIM_PNC pnc
         WHERE pnc.CLAIMID = k.PZINSKEY
           AND pnc.NOPOLIS = k.POLICYNO)                   AS TANGGAL_KONDISI,
       :1                  AS KODE_KANTOR_CABANG,
       CASE
         WHEN EXISTS (SELECT 1 FROM POOLDATA.T_GENERAL g
                       WHERE g.NOPOLIS = k.POLICYNO
                         AND g.IDPEGA LIKE '%EDM%')
         THEN 'U'
         ELSE 'C'
       END                 AS OPERASI_DATA,
       o.CUSTOMERTYPE      AS CUSTOMER_TYPE,
       o.OBJECTGENDER      AS JENIS_KELAMIN,
       o.DATEOFBIRTH       AS TANGGAL_LAHIR,
       o.ASMZIPCODE        AS KODE_POS,
       o.ASMADDRESS        AS ALAMAT,
       o.TELFAXNUMBER      AS TELEPON
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
  JOIN POOLDATA.T_CLAIM_OBJECTLIST o
    ON k.PZINSKEY = o.CLAIMID
 WHERE (:2 IS NULL OR k.REGISTERDATE_1 >= :3)
   AND (:4 IS NULL OR k.REGISTERDATE_1 < :5)
 ORDER BY k.REGISTERDATE_1 DESC, o.CLAIMID, o.OBJECTID, o.CONTRACTNO



-- ============================================================================
-- PEMERIKSA `-periksa`
-- ============================================================================
--
-- Ketiga tabel di bawah TIDAK dibaca modul lain mana pun, sehingga ketiadaannya baru
-- ketahuan ketika pengguna membuka layar ini dan mendapat galat 500. Pemeriksa
-- menjadikannya ketahuan saat start.
--
-- Sentinel bertipe TEKS dipakai pada kolom yang bertipe teks. Mengirim sentinel teks ke
-- kolom NUMBER menghasilkan ORA-01722, dan pemeriksa yang selalu gagal sama tidak
-- bergunanya dengan pemeriksa yang tidak ada — cacat yang sudah pernah terjadi dua kali
-- di repo ini (`catatan-pengembangan.md` §54.2).

-- name: probe_slik_table
SELECT COUNT(1) AS TOTAL
  FROM POOLDATA.T_CLAIM_SLIK_OJK
 WHERE CLAIMID = :1

-- name: probe_objectlist_table
SELECT COUNT(1) AS TOTAL
  FROM POOLDATA.T_CLAIM_OBJECTLIST
 WHERE CLAIMID = :1

-- name: probe_general_table
SELECT COUNT(1) AS TOTAL
  FROM POOLDATA.T_GENERAL
 WHERE NOPOLIS = :1

-- Tiga tabel berikut hanya dipakai kueri SUMBER (tombol "Proses Data Klaim"), sehingga
-- ketiadaannya tidak menghalangi pemantauan — hanya penyusunan laporannya.
--
-- Dilaporkan terpisah supaya perbedaannya terbaca: layar yang dapat memantau tetapi tidak
-- dapat menyusun adalah keadaan yang berbeda dari layar yang sama sekali tidak berfungsi.

-- name: probe_claim_pnc_table
SELECT COUNT(1) AS TOTAL
  FROM POOLDATA.T_CLAIM_PNC
 WHERE CLAIMID = :1

-- name: probe_adjustment_table
SELECT COUNT(1) AS TOTAL
  FROM POOLDATA.T_CLAIM_ADJUSTMENT
 WHERE CLAIMID = :1

-- name: probe_coverage_table
SELECT COUNT(1) AS TOTAL
  FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE
 WHERE CLAIMID = :1

-- Tabel pengiriman ke SLIK, dipakai tombol "SLIK OJK".

-- name: probe_submission_table
SELECT COUNT(1) AS TOTAL
  FROM POOLDATA.T_CLAIM_SLINK_INDIVIDU
 WHERE NO_KLAIM = :1
