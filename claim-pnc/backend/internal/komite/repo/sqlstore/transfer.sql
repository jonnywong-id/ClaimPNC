-- Kueri rincian "Lihat Detail Transfer" — pengganti `Section/ShowTransferDetail`.
--
-- ============================================================================
-- KEDUANYA HANYA MEMBACA.
-- ============================================================================
--
-- `POOLDATA.T_CLAIM_ADJUSTMENT` dan `POOLDATA.T_CLAIM_KOMITE_LIST` masih ditulis Pega, dan
-- `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem.
--
--
-- ============================================================================
-- KENAPA DATANYA DARI SINI, BUKAN DARI TEMPAT PEGA MEMBACANYA
-- ============================================================================
--
-- `ShowTransferDetail` membaca CLIPBOARD objek kerja, yang `OBJ-OPEN-BY-HANDLE` muat dari
-- BLOB Pega. Blob tidak dapat dibaca SQL.
--
-- Diverifikasi ke `ALL_TAB_COLUMNS`: dari 33 properti yang section itu tampilkan, hanya
-- ENAM yang benar-benar kolom `PC_ASM_FW_GCNMFW_WORK`. Dua puluh tujuh sisanya — termasuk
-- SELURUH nilai uangnya — tidak ada di sana.
--
-- Nilainya karena itu diambil dari tabel POOLDATA yang memuat data yang sama. Alasan
-- lengkapnya, beserta angka pengukurannya, ada di `internal/komite/transfer.go`.
--
--
-- ============================================================================
-- CASEIDKOMITE, BUKAN CLAIMID — DAN PERBEDAANNYA DIUKUR
-- ============================================================================
--
-- Keduanya ada di `T_CLAIM_ADJUSTMENT`, dan keduanya berarti hal yang BERBEDA. Diukur pada
-- dua belas case yang punya keduanya:
--
--     lewat CASEIDKOMITE   1 baris   pada dua belas-duanya
--     lewat CLAIMID        4–8 baris pada dua belas-duanya
--
-- Tidak satu pun sama. `CLAIMID` mengembalikan seluruh baris adjustment milik KLAIMNYA —
-- termasuk milik case komite lain. Memakainya berarti menaruh empat sampai delapan nilai
-- uang yang bukan urusan case ini pada layar tempat orang menyetujui uang.
--
-- `TestKueriTransferMemakaiKunciCaseKomite` menegakkan pilihan ini.
--
--
-- ============================================================================
-- GAYA SQL
-- ============================================================================
--
-- COALESCE bukan NVL · CASE WHEN bukan DECODE · kolom selalu disebut namanya · tanpa
-- TO_CHAR untuk tampilan. Mengikuti `09-DATABASE-STRATEGY.md` §4.


-- name: transfer_lines
--
-- Baris adjustment yang diputuskan satu case komite.
--
-- Diurutkan supaya hasilnya PASTI: dua baris pada case yang sama tidak boleh berpindah
-- tempat antar permintaan, karena pembacanya membandingkan angka uang antar baris.
--
-- # Nilai uang dibulatkan ke sen di sini
--
-- Pega menyimpan sebagian nilai berdesimal lebih dari dua (`ASM_SHARE_VALUE` `69591.261` pada
-- KMT-209620, `LOSS_ADJUSTER_FEE` `21881.317425` pada KMT-208651). money.Money bersatuan sen
-- dan MENOLAK angka yang lebih halus, sehingga tanpa ROUND case semacam itu gagal dibuka
-- sama sekali. Pembulatan ke sen adalah pembulatan tampilan (`I-12`): nilainya tidak
-- disimpan balik, dan tidak ada yang dijadikan nol.
SELECT j.CLAIMID,
       j.OBJECTID,
       j.OBJECTCOVERAGEID,
       j.NOAKSEPTASI,
       j.TGLAKSEPTASI,
       j.CURRENCY,
       j.PAYMENTTYPE,
       ROUND(j.GROSSVALUE, 2),
       ROUND(j.PROPOSE_VALUE, 2),
       ROUND(j.NILAIAKSEPTASI, 2),
       ROUND(j.NILAI_SALVAGE_A, 2),
       j.ASM_SHARE,
       ROUND(j.ASM_SHARE_VALUE, 2),
       ROUND(j.INDIVIDUAL_RISK_VALUE, 2),
       j.EXGRATIA,
       j.NOTES,
       j.CIRCUMCAUSEOFLOSS,
       -- Dua kolom di bawah ditambahkan untuk kolom kanan layar (committeesheet.go):
       -- kode mata uang untuk kolom "Currency", dan fee adjuster sebagai nilai komite
       -- Type 4/7. `CURRENCY.ID` unik (35 dari 35), jadi LEFT JOIN tidak menggandakan baris.
       m.CURRENCY,
       ROUND(j.LOSS_ADJUSTER_FEE, 2),
       -- Bahan tabel Claim Adjustment (breakdown.go). Kedua subkueri berkorelasi memakai
       -- indeks CLAIMID (T_CLAIM_ESTIMASI_IDX_01, T_CLAIM_ADJUSTMENT_IDX1) dan hanya
       -- berjalan untuk 1–3 baris satu case.
       j.ADJUSTMENTID,
       ROUND(j.TOTAL_CLAIM, 2),
       j.LOC,
       j.INDIVIDUAL_RISK_TYPE,
       j.INDIVIDUAL_RISK_PERCENT,
       (SELECT ROUND(SUM(e.ESTIMATIONVALUE), 2)
          FROM POOLDATA.T_CLAIM_ESTIMASI e
         WHERE e.CLAIMID = j.CLAIMID
           AND e.OBJECTID = j.OBJECTID
           AND e.OBJECTCOVERAGEID = j.OBJECTCOVERAGEID
           AND e.ESTIMATIONTYPE = CASE WHEN j.PAYMENTTYPE IN ('4', '7') THEN '2' ELSE '1' END),
       (SELECT ROUND(SUM(i.GROSSVALUE), 2)
          FROM POOLDATA.T_CLAIM_ADJUSTMENT i
         WHERE i.CLAIMID = j.CLAIMID
           AND i.OBJECTID = j.OBJECTID
           AND i.OBJECTCOVERAGEID = j.OBJECTCOVERAGEID
           AND i.PAYMENTTYPE = '2'
           AND i.STATUSAKSEPTASI = '1'
           AND i.ADJUSTMENTID <> j.ADJUSTMENTID)
  FROM POOLDATA.T_CLAIM_ADJUSTMENT j
  LEFT JOIN POOLDATA.CURRENCY m
         ON m.ID = j.CURRENCY
 WHERE j.CASEIDKOMITE = :1
 ORDER BY j.OBJECTID, j.OBJECTCOVERAGEID, j.ADJUSTMENTID


-- name: transfer_committee
--
-- Keputusan komite MENURUT PEGA.
--
-- Diagregasi, bukan diambil apa adanya: `KOMITE_ID` tidak dijamin unik pada tabel berisi
-- 39 juta baris itu, dan dua baris untuk satu case akan membuat kueri ini mengembalikan
-- dua hasil pada tempat yang hanya menerima satu.
--
-- MAX dipakai seragam supaya "satu nilai per case" tidak bergantung pada baris mana yang
-- kebetulan dikembalikan lebih dulu.
SELECT MAX(k.NAMAKOMITE),
       MAX(k.KOMITEKE),
       MAX(k.TYPEKOMITE),
       MAX(k.PAYMENTTYPE),
       MAX(k.NOTEKOMITE),
       ROUND(MAX(k.NILAIKLAIM), 2),
       MAX(k.SHAREASM),
       MAX(k.TANGGALKOMITE),
       MAX(k.STATUSAPPROVE),
       COUNT(1),
       -- "CREATE COMITEE DATE": seluruh anggota satu case dibentuk bersamaan, jadi yang
       -- paling awal adalah saat case komite dibuat.
       MIN(k.DATEOFCOMMITE_CREATE)
  FROM POOLDATA.T_CLAIM_KOMITE_LIST k
 WHERE k.KOMITE_ID = :1


-- name: transfer_case
--
-- Dua medan pada BARIS KERJA case komite yang dibutuhkan JUDUL layar, bukan isinya.
--
-- `Section/ShowTransfer` memasang enam label bersyarat di sebelah "CLAIM COMMITTEE". Dua
-- di antaranya bersandar pada properti yang tidak ada di `T_CLAIM_KOMITE_LIST`:
--
--     IsTravel   GroupPanel = "005"                      -> GROUPPANEL_1
--     IsHE       BusinessType in ("HE","ContractorsPM")  -> BUSINESSTYPE
--
-- # Kenapa kueri tersendiri, bukan disatukan ke transfer_committee
--
-- Sudah dicoba sebagai subkueri skalar di dalam SELECT ber-agregat tanpa GROUP BY, dan
-- Oracle menolaknya: `ORA-00937: not a single-group group function`. Jadi pemisahan ini
-- bukan selera melainkan satu-satunya bentuk yang jalan.
--
-- Pemisahan itu sekaligus membuat kedua medan ini tetap terbaca pada case yang TIDAK punya
-- baris komite sama sekali — 218 dari 610 sejak 2024 berada dalam keadaan itu.
--
-- # Yang harus disadari tentang BUSINESSTYPE
--
-- Diukur pada basis data ASM: `GROUPPANEL_1` terisi 610 dari 610, `BUSINESSTYPE` terisi
-- **0 dari 610**. Kolomnya ada, tetapi Pega tidak menulisinya untuk kelas `Work-Komite`.
-- Ia tetap dibaca supaya cabang HE benar dengan sendirinya bila kolom itu kelak terisi,
-- dan supaya layar dapat MENYATAKAN bahwa cabang itu tidak dapat dinilai.
--
-- `MAX` dipakai supaya kueri ini selalu mengembalikan tepat satu baris, termasuk ketika
-- `PYID` tidak ditemukan sama sekali.
-- `PNCCASEID` diambil APA ADANYA, lengkap dengan prefix `ASM-FW-GCNMFW-WORK `.
--
-- Itu bukan kelalaian membersihkan: ia kunci join ke `T_CLAIM_PNC`, `T_CLAIM_OBJECTLIST`,
-- dan `T_CLAIM_OBJECTCOVERAGE`, yang ketiganya menyimpan `CLAIMID` BER-PREFIX. Memangkasnya
-- lebih dulu menghasilkan nol baris pada seluruh 189 case, tanpa satu pun galat.
--
-- Pemangkasannya terjadi di lapisan domain, setelah join selesai (`D-22`).
SELECT MAX(a.GROUPPANEL_1),
       MAX(a.BUSINESSTYPE),
       MAX(a.PNCCASEID)
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
 WHERE a.PYID = :1
   AND a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-Komite'


-- name: transfer_case_new
--
-- Padanan `transfer_case` untuk case komite yang dibentuk APLIKASI INI (KMTN.*).
--
-- Case itu tidak punya baris kerja Pega — `TC_PNC_KOMITE` adalah kepalanya
-- (`docs/ddl/tc_pnc_komite.sql`). Tanpa kueri ini kunci klaimnya kosong, dan seluruh blok
-- klaim, coverage, spreading, serta lampiran tidak terbaca.
--
-- Ketiga medan diambil dengan urutan yang SAMA dengan `transfer_case`:
--
--     GROUPPANEL           IsTravel (Group Panel 005)
--     POLIS_JENIS_BISNIS   IsHE — Quotation.BusinessType yang registrasi simpan
--     CLAIMID              kunci klaim; untuk klaim PNCN ia nomor klaim itu sendiri
--
-- `MAX` supaya selalu tepat satu baris, termasuk ketika KOMITE_ID tidak ditemukan.
SELECT MAX(p.GROUPPANEL),
       MAX(p.POLIS_JENIS_BISNIS),
       MAX(k.CLAIMID)
  FROM POOLDATA.TC_PNC_KOMITE k
  LEFT JOIN POOLDATA.T_CLAIM_PNC p
         ON p.CLAIMID = k.CLAIMID
 WHERE k.KOMITE_ID = :1
   AND k.DIHAPUS_PADA IS NULL


-- name: transfer_claim
--
-- Data klaim yang `ShowTransferDetail` tampilkan lewat `.KomiteClaimData.*`.
--
-- Dikunci `CLAIMID` BER-PREFIX — lihat catatan pada `transfer_case`. Terbaca pada 189 dari
-- 189 case inbox; 189 case itu bermuara pada 72 klaim yang berbeda, karena satu klaim dapat
-- dikomitekan lebih dari sekali.
--
-- `MAX` dipakai supaya kueri ini selalu mengembalikan tepat satu baris, dan `COUNT(1)`
-- membedakan "tidak ada klaimnya" dari "ada tetapi kolomnya kosong".
SELECT MAX(p.DATEOFLOSS),
       MAX(p.REGISTERDATE),
       MAX(p.LOCATION),
       MAX(p.KRONOLOGI),
       MAX(p.STATUSCLAIM),
       MAX(p.REMARKRECOMENDATION),
       MAX(p.SHAREASM),
       MAX(p.COINSNAME),
       MAX(p.CURRENCY),
       MAX(p.EXGRATIA),
       COUNT(1),
       -- Kepala kolom kiri `ShowTransferDetail` (.Policy.* dan .CoverID). Lookup mata uang
       -- lewat LEFT JOIN, bukan subkueri skalar: subkueri di dalam SELECT beragregat tanpa
       -- GROUP BY ditolak Oracle (ORA-00937, lihat `transfer_case`).
       MAX(p.CLAIMNO),
       MAX(p.NOPOLIS),
       MAX(p.QQNAME),
       MAX(p.BUSINESSNAME),
       MAX(p.BRANCHNAME),
       MAX(p.SOBNAME),
       MAX(p.LEADER_MEMBER),
       MAX(p.GROUPPANEL),
       MAX(m.CURRENCY)
  FROM POOLDATA.T_CLAIM_PNC p
  LEFT JOIN POOLDATA.CURRENCY m
         ON m.ID = p.CURRENCY
 WHERE p.CLAIMID = :1


-- name: transfer_coverages
--
-- Blok analisis komite: di sinilah `.Komite.ExtentOfLoss`,
-- `.Komite.CircumtansesCouseOfLoss`, dan `.Komite.Remarks` sebenarnya tinggal.
--
-- Bukan di tabel kerja, dan bukan di `T_CLAIM_KOMITE_LIST` — keduanya sudah diperiksa.
--
-- # Tanpa batas baris, dan itu diukur lebih dulu
--
-- Satu case menghasilkan 1 sampai 3 baris, rerata 1,4; tidak satu pun melampaui tiga.
-- Memberi `FETCH NEXT` di sini hanya akan menyembunyikan baris ketiga tanpa alasan.
--
-- Baris yang sudah ditandai terhapus dibuang: `DIHAPUS_PADA` adalah penanda soft delete
-- pada tabel warisan ini, dan menampilkannya berarti menampilkan coverage yang sudah
-- dicabut dari klaimnya.
SELECT c.OBJECTID,
       c.OBJECTCOVERAGEID,
       c.OBJECTNAME,
       c.COVERAGENAME,
       c.CAUSEOFLOSS,
       ROUND(c.SUMTSI, 2),
       c.CURRENCY,
       c.CURICUMOFLOSS,
       c.EXTENTOFLOSS,
       c.LEGALLIABILITY,
       c.REMARKS,
       c.DIAGNOSE,
       c.INITIALNAME,
       c.TANGGALCOMITEE
  FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE c
 WHERE c.CLAIMID = :1
   AND c.DIHAPUS_PADA IS NULL
 ORDER BY c.OBJECTID, c.OBJECTCOVERAGEID


-- ============================================================================
-- KOLOM KANAN DAN TAB TAMBAHAN — lihat internal/komite/committeesheet.go
-- ============================================================================
--
-- Pega mengisi bagian ini di clipboard (`CalculatedSpredingForClaimKomite`,
-- `SetListComiteeClaimPerObjAdj`, `ShowKomiteViewContent`). Yang dibaca di sini BAHANNYA;
-- hitungannya dilakukan di Go dengan rumus yang sama.


-- name: transfer_policy
--
-- Periode pertanggungan — `.Policy.StartDateTime` S/D `.Policy.EndDateTime`.
--
-- Dokumen yang dipakai sama dengan registrasi `polis_ambil`: baris JSON_POLIS terbaru,
-- POLICYDATA lebih dulu, DATA_JSONBLOB bila POLICYDATA kosong. Nilainya teks Pega
-- (`20260801T050000.000 GMT`) dan diurai di Go.
SELECT COALESCE(JSON_VALUE(p.POLICYDATA, '$.StartDateTime'),
                JSON_VALUE(p.DATA_JSONBLOB, '$.StartDateTime')),
       COALESCE(JSON_VALUE(p.POLICYDATA, '$.EndDateTime'),
                JSON_VALUE(p.DATA_JSONBLOB, '$.EndDateTime'))
  FROM POOLDATA.JSON_POLIS p
 WHERE p.NOPOLIS = :1
   AND (p.POLICYDATA IS NOT NULL OR p.DATA_JSONBLOB IS NOT NULL)
 ORDER BY p.TGL_INPUT DESC
 FETCH FIRST 1 ROWS ONLY


-- name: transfer_coinsurance
--
-- CoinsList polis — sumber LEADER dan CO MEMBER. Salinan registrasi `polis_koasuransi`:
-- dua cabang karena POLICYDATA (CLOB) dan DATA_JSONBLOB (BLOB) tidak dapat digabung
-- COALESCE sebelum JSON_TABLE.
WITH terbaru AS (
    SELECT p.POLICYDATA, p.DATA_JSONBLOB
      FROM POOLDATA.JSON_POLIS p
     WHERE p.NOPOLIS = :1
       AND (p.POLICYDATA IS NOT NULL OR p.DATA_JSONBLOB IS NOT NULL)
     ORDER BY p.TGL_INPUT DESC
     FETCH FIRST 1 ROWS ONLY
)
SELECT jt.LEADER, jt.COINS_NAME, jt.PERCENT_SHARE
  FROM terbaru t,
       JSON_TABLE(t.POLICYDATA, '$.CoinsList[*]' COLUMNS (
           LEADER        VARCHAR(10)  PATH '$.Leader',
           COINS_NAME    VARCHAR(200) PATH '$.CoinsName',
           PERCENT_SHARE VARCHAR(50)  PATH '$.PercentShare')) jt
 WHERE t.POLICYDATA IS NOT NULL
UNION ALL
SELECT jt.LEADER, jt.COINS_NAME, jt.PERCENT_SHARE
  FROM terbaru t,
       JSON_TABLE(t.DATA_JSONBLOB, '$.CoinsList[*]' COLUMNS (
           LEADER        VARCHAR(10)  PATH '$.Leader',
           COINS_NAME    VARCHAR(200) PATH '$.CoinsName',
           PERCENT_SHARE VARCHAR(50)  PATH '$.PercentShare')) jt
 WHERE t.POLICYDATA IS NULL


-- name: transfer_fac_offer
--
-- FacOfferList polis — baris "List Reas Fac-Out". Salinan registrasi `cfs_fac_offer`.
WITH terbaru AS (
    SELECT p.POLICYDATA, p.DATA_JSONBLOB
      FROM POOLDATA.JSON_POLIS p
     WHERE p.NOPOLIS = :1
       AND (p.POLICYDATA IS NOT NULL OR p.DATA_JSONBLOB IS NOT NULL)
     ORDER BY p.TGL_INPUT DESC
     FETCH FIRST 1 ROWS ONLY
)
SELECT jt.REINSURER_NAME, jt.PCT_SHARE
  FROM terbaru t,
       JSON_TABLE(t.POLICYDATA, '$.FacOfferList[*]' COLUMNS (
           REINSURER_NAME VARCHAR(200) PATH '$.ReinsurerName',
           PCT_SHARE      VARCHAR(50)  PATH '$.PctShareForAllObj')) jt
 WHERE t.POLICYDATA IS NOT NULL
UNION ALL
SELECT jt.REINSURER_NAME, jt.PCT_SHARE
  FROM terbaru t,
       JSON_TABLE(t.DATA_JSONBLOB, '$.FacOfferList[*]' COLUMNS (
           REINSURER_NAME VARCHAR(200) PATH '$.ReinsurerName',
           PCT_SHARE      VARCHAR(50)  PATH '$.PctShareForAllObj')) jt
 WHERE t.POLICYDATA IS NULL


-- name: transfer_spreading
--
-- Spreading seluruh coverage klaim; dipilah per baris adjustment di Go.
--
-- Nama treaty dari view `REINSURANCETYPE` (ID unik, 62 dari 62) — itulah yang ditampilkan
-- dropdown Pega (`BrowseReinsuranceType_RD`, prompt `.Note`). TREATYNAME hanya cadangan:
-- ia kosong pada klaim PNCN.
SELECT s.OBJECTID,
       s.OBJECTCOVERAGEID,
       s.TREATYTYPE,
       COALESCE(t.NOTE, s.TREATYNAME),
       s.SHAREPERCENTAGE
  FROM POOLDATA.T_CLAIM_SPREADING s
  LEFT JOIN POOLDATA.REINSURANCETYPE t
         ON t.ID = s.TREATYTYPE
 WHERE s.CLAIMID = :1
 ORDER BY s.OBJECTID, s.OBJECTCOVERAGEID, s.URUTAN


-- name: transfer_dominant_factors
--
-- "Dominan Factor" — `KomiteClaimData.DominanFactorList`, diisi `ShowKomiteViewContent`.
-- Pasangan tabelnya sama dengan `GetDataDominanFactorListOS` dan modul laporan klaim.
SELECT f.NAME
  FROM POOLDATA.T_CLAIM_DOMINANFACTOR d
  JOIN POOLDATA.M_DOMINAN_FACTOR f
    ON f.ID = d.ID_DOMINANFACTOR
 WHERE d.CLAIMID = :1
 ORDER BY d.IDX_DOMINANFACTOR


-- name: transfer_attachments
--
-- Lampiran klaim untuk tab "Lampiran Dokumen". Isi berkasnya (BLOB ATTACHFILE) tidak dibaca.
--
-- Tiga bentuk kunci — apa adanya, tanpa prefix, dan ber-prefix — sama dengan registrasi
-- `lampiran_daftar`: IDPEGA ditulis dalam bentuk yang berbeda oleh Pega dan oleh aplikasi
-- ini. Tabelnya kecil (±8.900 baris) sehingga ketiadaan indeks IDPEGA tidak terasa.
SELECT a.DATAID,
       a.ATTACHNAME,
       a.ATTACHNOTE,
       a.CATEGORY,
       a.INPUTOPERATOR,
       a.INPUTDATE
  FROM POOLDATA.DATA_ATTACHFILE a
 WHERE a.IDPEGA IN (:1, :2, :3)
 ORDER BY a.INPUTDATE, a.DATAID


-- ============================================================================
-- DAFTAR KOMITE DAN HISTORY — bagian bawah ShowTransferDetail
-- ============================================================================
--
-- Keduanya baris `T_CLAIM_KOMITE_LIST` lewat indeks KOMITE_ID. Anggota ganda (9 di seluruh
-- tabel) digabung GROUP BY seperti kueri inbox; NAMAKOMITE dirapikan TRIM.


-- name: transfer_members
--
-- "Daftar Komite" — `.KomiteList` case ini: Komite, Status, Catatan, Tanggal Akseptasi.
SELECT MAX(TRIM(k.NAMAKOMITE)),
       CAST(k.KOMITEKE AS INTEGER),
       MAX(k.STATUSAPPROVE),
       MAX(k.NOTEKOMITE),
       MAX(k.TANGGALKOMITE),
       k.KOMITE_ID
  FROM POOLDATA.T_CLAIM_KOMITE_LIST k
 WHERE k.KOMITE_ID = :1
 GROUP BY k.KOMITE_ID, UPPER(TRIM(k.NAMAKOMITE)), CAST(k.KOMITEKE AS INTEGER)
 ORDER BY CAST(k.KOMITEKE AS INTEGER), MAX(TRIM(k.NAMAKOMITE))


-- name: transfer_history_legacy
--
-- "History of Previous Adjustment Committees" untuk klaim Pega: case komite lain yang
-- `PNCCASEID`-nya kunci klaim yang sama — kolom yang juga dipakai `transfer_case`.
--
-- Bukan `PXCOVERINSKEY`: kolom itu (yang berindeks) KOSONG pada seluruh baris Work-Komite —
-- diperiksa 2026-09-29. `PNCCASEID` tanpa indeks, tetapi tabel kerja hanya ±6.800 baris
-- (diukur 13 ms).
--
-- :1 kunci klaim ber-prefix, :2 case yang sedang dibuka (dikecualikan).
SELECT MAX(TRIM(k.NAMAKOMITE)),
       CAST(k.KOMITEKE AS INTEGER),
       MAX(k.STATUSAPPROVE),
       MAX(k.NOTEKOMITE),
       MAX(k.TANGGALKOMITE),
       k.KOMITE_ID
  FROM POOLDATA.T_CLAIM_KOMITE_LIST k
 WHERE k.KOMITE_ID IN (SELECT a.PYID
                         FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
                        WHERE a.PNCCASEID = :1
                          AND a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-Komite')
   AND k.KOMITE_ID <> :2
 GROUP BY k.KOMITE_ID, UPPER(TRIM(k.NAMAKOMITE)), CAST(k.KOMITEKE AS INTEGER)
 ORDER BY MIN(k.DATEOFCOMMITE_CREATE), k.KOMITE_ID, CAST(k.KOMITEKE AS INTEGER)


-- name: transfer_history_new
--
-- Padanan untuk klaim PNCN: case komitenya di TC_PNC_KOMITE (indeks CLAIMID).
--
-- :1 kunci klaim, :2 case yang sedang dibuka (dikecualikan).
SELECT MAX(TRIM(k.NAMAKOMITE)),
       CAST(k.KOMITEKE AS INTEGER),
       MAX(k.STATUSAPPROVE),
       MAX(k.NOTEKOMITE),
       MAX(k.TANGGALKOMITE),
       k.KOMITE_ID
  FROM POOLDATA.T_CLAIM_KOMITE_LIST k
 WHERE k.KOMITE_ID IN (SELECT t.KOMITE_ID
                         FROM POOLDATA.TC_PNC_KOMITE t
                        WHERE t.CLAIMID = :1
                          AND t.DIHAPUS_PADA IS NULL)
   AND k.KOMITE_ID <> :2
 GROUP BY k.KOMITE_ID, UPPER(TRIM(k.NAMAKOMITE)), CAST(k.KOMITEKE AS INTEGER)
 ORDER BY MIN(k.DATEOFCOMMITE_CREATE), k.KOMITE_ID, CAST(k.KOMITEKE AS INTEGER)


-- name: transfer_check_table
--
-- Memastikan kedua tabel beserta kolomnya dapat dibaca akun aplikasi, tanpa mengambil satu
-- baris pun.
SELECT j.CASEIDKOMITE,
       j.CLAIMID,
       j.NILAIAKSEPTASI,
       j.ASM_SHARE_VALUE,
       j.CIRCUMCAUSEOFLOSS,
       k.KOMITE_ID,
       k.NAMAKOMITE,
       k.TYPEKOMITE,
       p.CLAIMID,
       p.KRONOLOGI,
       p.REMARKRECOMENDATION,
       c.CLAIMID,
       c.COVERAGENAME,
       c.EXTENTOFLOSS,
       c.DIHAPUS_PADA
  FROM POOLDATA.T_CLAIM_ADJUSTMENT j
  LEFT JOIN POOLDATA.T_CLAIM_KOMITE_LIST k
         ON k.KOMITE_ID = j.CASEIDKOMITE
  LEFT JOIN POOLDATA.T_CLAIM_PNC p
         ON p.CLAIMID = j.CLAIMID
  LEFT JOIN POOLDATA.T_CLAIM_OBJECTCOVERAGE c
         ON c.CLAIMID = j.CLAIMID
 WHERE 1 = 0
