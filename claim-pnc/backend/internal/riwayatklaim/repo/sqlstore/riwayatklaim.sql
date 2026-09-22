-- Kueri modul View History Claim: POOLDATA.T_CLAIM_PNC dan kerabatnya.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- ============================================================================
-- PEMETAAN TIGA ARAH — properti grid Pega -> kolom sebenarnya -> arti
-- ============================================================================
--
-- Ini satu-satunya tempat ketiganya dapat dibandingkan berdampingan, dan ia ada karena
-- nama properti grid lama MENYESATKAN SECARA AKTIF — bukan sekadar singkatan yang tidak
-- lazim. Sumbernya `Section/PNCSearchKlaim-Section.xml` (urutan kolom grid) dan kedua
-- belas berkas `RDB List/Broswse*-SQL.xml` (alias kolomnya).
--
-- Properti grid Pega   Kolom sebenarnya                  Arti bagi pengguna  Alias di sini
-- -------------------- --------------------------------- ------------------- -----------------
-- .IDPEGA              CLAIMID                           kunci teknis Pega   REFERENCE
-- .EDMNO           (!) CLAIMNO                           No Klaim            CLAIM_NUMBER
-- .NOPOLIS             NOPOLIS                           No Polis            POLICY_NUMBER
-- .QQNAME              QQNAME                            Nama Tertanggung    INSURED_NAME
-- .STARTDATE       (!) DATEOFLOSS                        Tgl Kejadian        LOSS_DATE
-- .BUSINESSNAME        BUSINESSNAME                      Bisnis              BUSINESS_NAME
-- .BRANCHNAME          BRANCHNAME                        Cabang              BRANCH_NAME
-- .STATUSBUSINESS  (!) STATUSWORK                        Status              WORK_STATUS
-- .THEINSURED      (!) V_STS_CLAIM.LSC_NOTE              Posisi Klaim        CLAIM_POSITION
-- .ENDDATE         (!) CLOSECLAIMDATE                    Tanggal Close       CLOSE_DATE
-- .FLAGEDMBATAL    (!) CLOSECLAIMNOTE                    Catatan Close       CLOSE_NOTE
-- .SOBNAME         (!) PICTEKNIK                         PIC Teknis          TECHNICAL_PIC
-- .OLDPOLICYNO     (!) DETAIL_PNC_SALVAGE.NOAKSEPTASI    No Akseptasi        ACCEPTANCE_NUMBER
-- .WARRANTYNO      (!) DETAIL_PNC_SALVAGE.IDBALAILELANG  No Balai Lelang     AUCTION_HOUSE_ID
-- .SOBLEADER1      (!) T_PERSON.FULLNAME                 Nama Objek          INSURED_ITEM_NAME
-- .EDMDATE         (!) T_PERSON.ASMDATEOFBIRTH           Tanggal Lahir       BIRTH_DATE
--
-- Tanda (!) menandai nama yang sama sekali tidak menyatakan isinya. Dua belas dari enam
-- belas. Inilah utang teknis `03-CURRENT-ARCHITECTURE.md` §4.2 dalam bentuk paling pekat
-- di seluruh export, dan alasan nama di kode ini tidak mirip nama di Pega (`D-19`).
--
-- ============================================================================
-- KEENAM BELAS ALIAS WAJIB SAMA DI SETIAP KUERI
-- ============================================================================
--
-- Urutan DAN namanya. Dua hal bergantung padanya:
--
--   * satu pemindai Go melayani kesebelas kueri (scanClaim di riwayatklaim.go);
--   * kueri dibungkus subquery untuk paginasi dan penghitungan (paged/counted di
--     query.go), dan subquery tanpa nama kolom tidak dapat dirujuk dari luar.
--
-- Kolom yang tidak berlaku bagi sebuah tipe pencarian bernilai NULL, bukan dihilangkan.
-- Layar menyembunyikannya mengikuti SearchType.ExtraColumns — bukan menampilkan kolom
-- kosong yang membuat pengguna menduga datanya hilang.
--
-- Uji query_test.go menjaga keseragaman ini; ia gagal bila ada kueri yang aliasnya
-- berbeda atau jumlah parameternya bukan satu.
--
-- ============================================================================
-- EMPAT HAL YANG BERUBAH DARI KUERI LAMA, DAN ALASANNYA
-- ============================================================================
--
-- 1. PARAMETER BINDING, bukan perangkaian nilai.
--    Kedua belas kueri lama menyisipkan nilai langsung ke teks SQL — `{InputData.CARI4}`
--    dan `{ASIS:InputData.CARI4}`. Yang kedua bahkan menyisipkan POTONGAN SQL, bukan
--    nilai. `08-TECHNICAL-STRATEGY.md` §4.3 melarang keduanya tanpa perkecualian, dan
--    larangan itu TIDAK ikut dikecualikan oleh keputusan "replikasi apa adanya": yang
--    direplikasi adalah perilaku bisnis, bukan celah injeksi.
--
-- 2. SETIAP kueri mengembalikan keenam belas kolom yang sama, sebagian NULL.
--    Lihat bagian di atas.
--
-- 3. ORDER BY ditambahkan (di query.go, saat kueri dibungkus).
--    Tidak satu pun kueri lama mengurutkan hasilnya, dan itu dapat dibiarkan selama
--    seluruh baris ditarik sekaligus. Begitu hasilnya dipaginasi, urutan yang tidak
--    ditetapkan membuat baris yang sama muncul di dua halaman sekaligus hilang dari
--    halaman lain.
--
-- 4. OFFSET ... FETCH NEXT, bukan seluruh baris sekaligus.
--    Lihat riwayatklaim.Pagination — ini perubahan perilaku yang disadari
--    (`09-DATABASE-STRATEGY.md` §6.3), bukan pemeliharaan.

-- name: search_policy_number
-- Tipe 1 "No Polis" — RDB List/BroswseKlaimByPolicyNo-SQL.xml
--
-- Gabungan ke JSON_KLAIM dan saringan `STATUSCLAIM IS NOT NULL` dipertahankan apa adanya:
-- keduanya MENYARING, bukan sekadar memperkaya, sehingga membuangnya akan mengembalikan
-- baris yang tidak pernah terlihat di sistem lama.
SELECT a.CLAIMID                                        AS REFERENCE,
       a.CLAIMNO                                        AS CLAIM_NUMBER,
       a.NOPOLIS                                        AS POLICY_NUMBER,
       a.QQNAME                                         AS INSURED_NAME,
       a.DATEOFLOSS                                     AS LOSS_DATE,
       a.BUSINESSNAME                                   AS BUSINESS_NAME,
       a.BRANCHNAME                                     AS BRANCH_NAME,
       a.STATUSWORK                                     AS WORK_STATUS,
       (SELECT s.LSC_NOTE FROM POOLDATA.V_STS_CLAIM s
         WHERE s.LSC_ID = a.STATUSCLAIM)                AS CLAIM_POSITION,
       a.CLOSECLAIMDATE                                 AS CLOSE_DATE,
       a.CLOSECLAIMNOTE                                 AS CLOSE_NOTE,
       a.PICTEKNIK                                      AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(100))                      AS ACCEPTANCE_NUMBER,
       CAST(NULL AS VARCHAR2(100))                      AS AUCTION_HOUSE_ID,
       CAST(NULL AS VARCHAR2(400))                      AS INSURED_ITEM_NAME,
       CAST(NULL AS DATE)                               AS BIRTH_DATE
  FROM POOLDATA.T_CLAIM_PNC a,
       POOLDATA.JSON_KLAIM j
 WHERE a.STATUSCLAIM IS NOT NULL
   AND a.CLAIMID = j.IDPEGA
   AND a.NOPOLIS = :1

-- name: search_insured_name
-- Tipe 2 "Nama Customer" — RDB List/BroswseKlaimByName-SQL.xml
--
-- Pencocokan sebagian di kedua sisi, persis `QQNAME like '%…%'` yang lama. ESCAPE
-- dipasang supaya tanda `%` dan `_` yang diketik pengguna dicari sebagai huruf biasa,
-- bukan sebagai pola — tanpa itu satu tanda `%` mengembalikan seluruh tabel.
SELECT a.CLAIMID                                        AS REFERENCE,
       a.CLAIMNO                                        AS CLAIM_NUMBER,
       a.NOPOLIS                                        AS POLICY_NUMBER,
       a.QQNAME                                         AS INSURED_NAME,
       a.DATEOFLOSS                                     AS LOSS_DATE,
       a.BUSINESSNAME                                   AS BUSINESS_NAME,
       a.BRANCHNAME                                     AS BRANCH_NAME,
       a.STATUSWORK                                     AS WORK_STATUS,
       (SELECT s.LSC_NOTE FROM POOLDATA.V_STS_CLAIM s
         WHERE s.LSC_ID = a.STATUSCLAIM)                AS CLAIM_POSITION,
       a.CLOSECLAIMDATE                                 AS CLOSE_DATE,
       a.CLOSECLAIMNOTE                                 AS CLOSE_NOTE,
       a.PICTEKNIK                                      AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(100))                      AS ACCEPTANCE_NUMBER,
       CAST(NULL AS VARCHAR2(100))                      AS AUCTION_HOUSE_ID,
       CAST(NULL AS VARCHAR2(400))                      AS INSURED_ITEM_NAME,
       CAST(NULL AS DATE)                               AS BIRTH_DATE
  FROM POOLDATA.T_CLAIM_PNC a
 WHERE UPPER(a.QQNAME) LIKE '%' || UPPER(:1) || '%' ESCAPE '\'

-- name: search_insured_item_name
-- Tipe 3 "Nama Objek" — RDB List/BroswseKlaimByObjectName-SQL.xml
--
-- CACAT YANG DIREPLIKASI: CLAIM_POSITION bernilai NULL di sini. Kueri lama adalah
-- SATU-SATUNYA dari kedua belas yang tidak membawa subquery V_STS_CLAIM, sehingga kolom
-- Posisi Klaim kosong tanpa alasan bisnis. Work Owner memutuskan 2026-09-20 ia
-- direplikasi apa adanya demi kesetaraan `P-5`, bukan diperbaiki.
SELECT a.CLAIMID                                        AS REFERENCE,
       a.CLAIMNO                                        AS CLAIM_NUMBER,
       a.NOPOLIS                                        AS POLICY_NUMBER,
       a.QQNAME                                         AS INSURED_NAME,
       a.DATEOFLOSS                                     AS LOSS_DATE,
       a.BUSINESSNAME                                   AS BUSINESS_NAME,
       a.BRANCHNAME                                     AS BRANCH_NAME,
       a.STATUSWORK                                     AS WORK_STATUS,
       CAST(NULL AS VARCHAR2(400))                      AS CLAIM_POSITION,
       a.CLOSECLAIMDATE                                 AS CLOSE_DATE,
       a.CLOSECLAIMNOTE                                 AS CLOSE_NOTE,
       a.PICTEKNIK                                      AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(100))                      AS ACCEPTANCE_NUMBER,
       CAST(NULL AS VARCHAR2(100))                      AS AUCTION_HOUSE_ID,
       CAST(NULL AS VARCHAR2(400))                      AS INSURED_ITEM_NAME,
       CAST(NULL AS DATE)                               AS BIRTH_DATE
  FROM POOLDATA.T_CLAIM_PNC a,
       POOLDATA.T_CLAIM_OBJECTLIST o
 WHERE o.CLAIMID = a.CLAIMID
   AND UPPER(o.OBJECTNAME) LIKE '%' || UPPER(:1) || '%' ESCAPE '\'

-- name: search_pla_number
-- Tipe 4 "No PLA" — RDB List/BroswseKlaimByPLANo-SQL.xml
SELECT a.CLAIMID                                        AS REFERENCE,
       a.CLAIMNO                                        AS CLAIM_NUMBER,
       a.NOPOLIS                                        AS POLICY_NUMBER,
       a.QQNAME                                         AS INSURED_NAME,
       a.DATEOFLOSS                                     AS LOSS_DATE,
       a.BUSINESSNAME                                   AS BUSINESS_NAME,
       a.BRANCHNAME                                     AS BRANCH_NAME,
       a.STATUSWORK                                     AS WORK_STATUS,
       (SELECT s.LSC_NOTE FROM POOLDATA.V_STS_CLAIM s
         WHERE s.LSC_ID = a.STATUSCLAIM)                AS CLAIM_POSITION,
       a.CLOSECLAIMDATE                                 AS CLOSE_DATE,
       a.CLOSECLAIMNOTE                                 AS CLOSE_NOTE,
       a.PICTEKNIK                                      AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(100))                      AS ACCEPTANCE_NUMBER,
       CAST(NULL AS VARCHAR2(100))                      AS AUCTION_HOUSE_ID,
       CAST(NULL AS VARCHAR2(400))                      AS INSURED_ITEM_NAME,
       CAST(NULL AS DATE)                               AS BIRTH_DATE
  FROM POOLDATA.T_CLAIM_PNC a,
       POOLDATA.T_PLALIST p
 WHERE p.CLAIMID = a.CLAIMID
   AND UPPER(p.NOPLA) LIKE '%' || UPPER(:1) || '%' ESCAPE '\'

-- name: search_dla_number
-- Tipe 5 "No DLA" — RDB List/BroswseKlaimByDLANo-SQL.xml
SELECT a.CLAIMID                                        AS REFERENCE,
       a.CLAIMNO                                        AS CLAIM_NUMBER,
       a.NOPOLIS                                        AS POLICY_NUMBER,
       a.QQNAME                                         AS INSURED_NAME,
       a.DATEOFLOSS                                     AS LOSS_DATE,
       a.BUSINESSNAME                                   AS BUSINESS_NAME,
       a.BRANCHNAME                                     AS BRANCH_NAME,
       a.STATUSWORK                                     AS WORK_STATUS,
       (SELECT s.LSC_NOTE FROM POOLDATA.V_STS_CLAIM s
         WHERE s.LSC_ID = a.STATUSCLAIM)                AS CLAIM_POSITION,
       a.CLOSECLAIMDATE                                 AS CLOSE_DATE,
       a.CLOSECLAIMNOTE                                 AS CLOSE_NOTE,
       a.PICTEKNIK                                      AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(100))                      AS ACCEPTANCE_NUMBER,
       CAST(NULL AS VARCHAR2(100))                      AS AUCTION_HOUSE_ID,
       CAST(NULL AS VARCHAR2(400))                      AS INSURED_ITEM_NAME,
       CAST(NULL AS DATE)                               AS BIRTH_DATE
  FROM POOLDATA.T_CLAIM_PNC a,
       POOLDATA.T_DLALIST d
 WHERE d.CLAIMID = a.CLAIMID
   AND UPPER(d.NODLA) LIKE '%' || UPPER(:1) || '%' ESCAPE '\'

-- name: search_loss_date
-- Tipe 6 "Tgl Kejadian" — RDB List/BroswseKlaimByDOL-SQL.xml
--
-- `TRUNC` dipertahankan: kolomnya bertipe DATE Oracle yang membawa jam, dan tanpa
-- pemotongan hanya baris yang jamnya tepat tengah malam yang akan cocok. Tanggalnya
-- dikirim sebagai parameter DATE, bukan sebagai teks yang di-`TO_DATE` — sehingga format
-- tanggal tidak lagi menjadi kontrak tersembunyi antara Go dan SQL.
SELECT a.CLAIMID                                        AS REFERENCE,
       a.CLAIMNO                                        AS CLAIM_NUMBER,
       a.NOPOLIS                                        AS POLICY_NUMBER,
       a.QQNAME                                         AS INSURED_NAME,
       a.DATEOFLOSS                                     AS LOSS_DATE,
       a.BUSINESSNAME                                   AS BUSINESS_NAME,
       a.BRANCHNAME                                     AS BRANCH_NAME,
       a.STATUSWORK                                     AS WORK_STATUS,
       (SELECT s.LSC_NOTE FROM POOLDATA.V_STS_CLAIM s
         WHERE s.LSC_ID = a.STATUSCLAIM)                AS CLAIM_POSITION,
       a.CLOSECLAIMDATE                                 AS CLOSE_DATE,
       a.CLOSECLAIMNOTE                                 AS CLOSE_NOTE,
       a.PICTEKNIK                                      AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(100))                      AS ACCEPTANCE_NUMBER,
       CAST(NULL AS VARCHAR2(100))                      AS AUCTION_HOUSE_ID,
       CAST(NULL AS VARCHAR2(400))                      AS INSURED_ITEM_NAME,
       CAST(NULL AS DATE)                               AS BIRTH_DATE
  FROM POOLDATA.T_CLAIM_PNC a
 WHERE TRUNC(a.DATEOFLOSS) = TRUNC(CAST(:1 AS DATE))

-- name: search_claim_number
-- Tipe 7 "No Klaim" — RDB List/BroswseKlaimByKlaimNo-SQL.xml
--
-- SATU PERUBAHAN TERHADAP KUERI LAMA, dan ia dituntut `D-22`.
--
-- Kueri lama mencari `CLAIMID = 'ASM-FW-GCNMFW-WORK ' || <nomor>` — merangkai nama kelas
-- internal Pega ke dalam kunci pencarian. Klaim yang diterbitkan sistem baru berformat
-- `PNCN.YY.xxxx` dan TIDAK PERNAH menulis awalan itu lagi (`D-22`, `D-71`), sehingga
-- kueri lama tidak akan pernah menemukannya.
--
-- Yang dicari di sini adalah CLAIMNO, dan itu setara untuk baris warisan: awalannya tepat
-- 19 karakter, dan kueri lama sendiri memperlakukan `SUBSTR(CLAIMID,20)` sebagai nomor
-- klaim pada `BroswseKlaimByPolicyNo`. Baris yang sama tetap ditemukan dengan masukan
-- yang sama; yang bertambah hanyalah baris terbitan sistem baru.
SELECT a.CLAIMID                                        AS REFERENCE,
       a.CLAIMNO                                        AS CLAIM_NUMBER,
       a.NOPOLIS                                        AS POLICY_NUMBER,
       a.QQNAME                                         AS INSURED_NAME,
       a.DATEOFLOSS                                     AS LOSS_DATE,
       a.BUSINESSNAME                                   AS BUSINESS_NAME,
       a.BRANCHNAME                                     AS BRANCH_NAME,
       a.STATUSWORK                                     AS WORK_STATUS,
       (SELECT s.LSC_NOTE FROM POOLDATA.V_STS_CLAIM s
         WHERE s.LSC_ID = a.STATUSCLAIM)                AS CLAIM_POSITION,
       a.CLOSECLAIMDATE                                 AS CLOSE_DATE,
       a.CLOSECLAIMNOTE                                 AS CLOSE_NOTE,
       a.PICTEKNIK                                      AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(100))                      AS ACCEPTANCE_NUMBER,
       CAST(NULL AS VARCHAR2(100))                      AS AUCTION_HOUSE_ID,
       CAST(NULL AS VARCHAR2(400))                      AS INSURED_ITEM_NAME,
       CAST(NULL AS DATE)                               AS BIRTH_DATE
  FROM POOLDATA.T_CLAIM_PNC a
 WHERE UPPER(a.CLAIMNO) = UPPER(:1)

-- name: search_acceptance_number
-- Tipe 8 "No Akseptasi" — RDB List/BroswseKlaimByAcceptedNo-SQL.xml
SELECT a.CLAIMID                                        AS REFERENCE,
       a.CLAIMNO                                        AS CLAIM_NUMBER,
       a.NOPOLIS                                        AS POLICY_NUMBER,
       a.QQNAME                                         AS INSURED_NAME,
       a.DATEOFLOSS                                     AS LOSS_DATE,
       a.BUSINESSNAME                                   AS BUSINESS_NAME,
       a.BRANCHNAME                                     AS BRANCH_NAME,
       a.STATUSWORK                                     AS WORK_STATUS,
       (SELECT s.LSC_NOTE FROM POOLDATA.V_STS_CLAIM s
         WHERE s.LSC_ID = a.STATUSCLAIM)                AS CLAIM_POSITION,
       a.CLOSECLAIMDATE                                 AS CLOSE_DATE,
       a.CLOSECLAIMNOTE                                 AS CLOSE_NOTE,
       a.PICTEKNIK                                      AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(100))                      AS ACCEPTANCE_NUMBER,
       CAST(NULL AS VARCHAR2(100))                      AS AUCTION_HOUSE_ID,
       CAST(NULL AS VARCHAR2(400))                      AS INSURED_ITEM_NAME,
       CAST(NULL AS DATE)                               AS BIRTH_DATE
  FROM POOLDATA.T_CLAIM_PNC a,
       POOLDATA.T_CLAIM_ADJUSTMENT j
 WHERE j.CLAIMID = a.CLAIMID
   AND UPPER(j.NOAKSEPTASI) = UPPER(:1)

-- name: search_birth_date
-- Tipe 9 "Tanggal Lahir" — RDB List/BroswseKlaimByBirthDate-SQL.xml
--
-- Satu-satunya kueri yang membawa Nama Objek dan Tanggal Lahir peserta.
--
-- CACAT YANG DIREPLIKASI: tanggal yang sampai ke sini BUKAN yang diketik pengguna,
-- melainkan isian "Tanggal Pencarian" yang tersembunyi untuk tipe ini — karena itu selalu
-- kosong. Kueri ini karena itu tidak pernah mengembalikan baris. Lihat
-- riwayatklaim.Criteria.QueryValue; perbandingan dengan NULL menghasilkan UNKNOWN, bukan
-- galat, sehingga hasilnya kosong tanpa pesan apa pun — persis sistem lama.
--
-- `{ASIS:InputData.CARI103}` yang menempel di ujung kueri lama TIDAK dibawa: ia
-- penyisipan potongan SQL mentah, dan properti CARI103 tidak pernah diisi activity mana
-- pun sehingga nilainya selalu kosong.
SELECT a.CLAIMID                                        AS REFERENCE,
       a.CLAIMNO                                        AS CLAIM_NUMBER,
       a.NOPOLIS                                        AS POLICY_NUMBER,
       a.QQNAME                                         AS INSURED_NAME,
       a.DATEOFLOSS                                     AS LOSS_DATE,
       a.BUSINESSNAME                                   AS BUSINESS_NAME,
       a.BRANCHNAME                                     AS BRANCH_NAME,
       a.STATUSWORK                                     AS WORK_STATUS,
       (SELECT s.LSC_NOTE FROM POOLDATA.V_STS_CLAIM s
         WHERE s.LSC_ID = a.STATUSCLAIM)                AS CLAIM_POSITION,
       a.CLOSECLAIMDATE                                 AS CLOSE_DATE,
       a.CLOSECLAIMNOTE                                 AS CLOSE_NOTE,
       a.PICTEKNIK                                      AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(100))                      AS ACCEPTANCE_NUMBER,
       CAST(NULL AS VARCHAR2(100))                      AS AUCTION_HOUSE_ID,
       p.FULLNAME                                       AS INSURED_ITEM_NAME,
       p.ASMDATEOFBIRTH                                 AS BIRTH_DATE
  FROM POOLDATA.T_CLAIM_PNC a,
       POOLDATA.T_PERSON p,
       POOLDATA.T_CLAIM_OBJECTLIST o
 WHERE a.NOPOLIS = p.NOPOLIS
   AND a.CLAIMID = o.CLAIMID
   AND o.OBJECTID = TO_CHAR(p.INDEXCOUNT)
   AND TRUNC(p.ASMDATEOFBIRTH) = TRUNC(CAST(:1 AS DATE))

-- name: search_survey_number
-- Tipe 11 "No Survey" — RDB List/BroswseKlaimByNoSurvey-SQL.xml
--
-- Awalan `'ASM-FW-GCNMFW-WORK ' ||` DIPERTAHANKAN di sini, berbeda dari tipe 7.
-- Alasannya: yang dirangkai bukan nomor klaim melainkan CASEID milik baris surveyor, dan
-- kolom itu memang menyimpan kunci berformat Pega. Tidak ada kolom lain yang setara untuk
-- menggantikannya, dan menebak ada akan menghasilkan pencarian yang diam-diam kosong.
--
-- `FETCH NEXT 1 ROW ONLY` di dalam subquery dipertahankan: tanpanya, satu nomor survei
-- yang menunjuk lebih dari satu baris membuat `=` gagal dengan ORA-01427.
SELECT a.CLAIMID                                        AS REFERENCE,
       a.CLAIMNO                                        AS CLAIM_NUMBER,
       a.NOPOLIS                                        AS POLICY_NUMBER,
       a.QQNAME                                         AS INSURED_NAME,
       a.DATEOFLOSS                                     AS LOSS_DATE,
       a.BUSINESSNAME                                   AS BUSINESS_NAME,
       a.BRANCHNAME                                     AS BRANCH_NAME,
       a.STATUSWORK                                     AS WORK_STATUS,
       (SELECT s.LSC_NOTE FROM POOLDATA.V_STS_CLAIM s
         WHERE s.LSC_ID = a.STATUSCLAIM)                AS CLAIM_POSITION,
       a.CLOSECLAIMDATE                                 AS CLOSE_DATE,
       a.CLOSECLAIMNOTE                                 AS CLOSE_NOTE,
       a.PICTEKNIK                                      AS TECHNICAL_PIC,
       CAST(NULL AS VARCHAR2(100))                      AS ACCEPTANCE_NUMBER,
       CAST(NULL AS VARCHAR2(100))                      AS AUCTION_HOUSE_ID,
       CAST(NULL AS VARCHAR2(400))                      AS INSURED_ITEM_NAME,
       CAST(NULL AS DATE)                               AS BIRTH_DATE
  FROM POOLDATA.T_CLAIM_PNC a
 WHERE a.CLAIMID = (SELECT v.PNCCASEID
                      FROM POOLDATA.T_SURVEYORLIST v
                     WHERE v.CASEID = 'ASM-FW-GCNMFW-WORK ' || :1
                     FETCH NEXT 1 ROW ONLY)

-- name: search_auction_house_id
-- Tipe 13 "ID Balai Lelang" — RDB List/BroswseKlaimByIdBalaiLelang-SQL.xml
--
-- Satu-satunya kueri yang membawa No Akseptasi dan No Balai Lelang. Gabungannya lewat
-- CLAIMNO, bukan CLAIMID — begitu pula di kueri lama.
SELECT a.CLAIMID                                        AS REFERENCE,
       a.CLAIMNO                                        AS CLAIM_NUMBER,
       a.NOPOLIS                                        AS POLICY_NUMBER,
       a.QQNAME                                         AS INSURED_NAME,
       a.DATEOFLOSS                                     AS LOSS_DATE,
       a.BUSINESSNAME                                   AS BUSINESS_NAME,
       a.BRANCHNAME                                     AS BRANCH_NAME,
       a.STATUSWORK                                     AS WORK_STATUS,
       (SELECT s.LSC_NOTE FROM POOLDATA.V_STS_CLAIM s
         WHERE s.LSC_ID = a.STATUSCLAIM)                AS CLAIM_POSITION,
       a.CLOSECLAIMDATE                                 AS CLOSE_DATE,
       a.CLOSECLAIMNOTE                                 AS CLOSE_NOTE,
       a.PICTEKNIK                                      AS TECHNICAL_PIC,
       v.NOAKSEPTASI                                    AS ACCEPTANCE_NUMBER,
       v.IDBALAILELANG                                  AS AUCTION_HOUSE_ID,
       CAST(NULL AS VARCHAR2(400))                      AS INSURED_ITEM_NAME,
       CAST(NULL AS DATE)                               AS BIRTH_DATE
  FROM POOLDATA.T_CLAIM_PNC a,
       POOLDATA.DETAIL_PNC_SALVAGE v
 WHERE a.CLAIMNO = v.NOKLAIM
   AND UPPER(v.IDBALAILELANG) = UPPER(:1)
