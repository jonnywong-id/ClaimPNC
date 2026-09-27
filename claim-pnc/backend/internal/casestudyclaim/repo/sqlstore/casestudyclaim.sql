-- Kueri layar **Case Study Claim** (`Harness/PNCStudyClaim-Harness.xml`, MENU_ID 74).
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- ============================================================================
-- SUMBERNYA: RDB List/BrowseClaimStudy-SQL.xml
-- ============================================================================
--
-- Satu kueri, tiga tabel, dan delapan subkueri terkorelasi:
--
--     POOLDATA.PEGA_DASHBOARDPNC   a   tabel datar ringkasan klaim
--     POOLDATA.T_CLAIM_PNC         b   klaim yang sebenarnya
--     POOLDATA.BUSINESS            d   master kelas bisnis (COB)
--     POOLDATA.T_CLAIM_ADJUSTMENT  adj nilai settlement — diagregat per klaim
--
-- SELURUHNYA milik sistem lama. Hanya SATU pernyataan di berkas ini yang menulis, dan ia
-- menyentuh SATU kolom: `case_study_save_remark`. Lihat catatannya sendiri di bawah.
--
-- ============================================================================
-- YANG MEMBUAT SEBUAH KLAIM MASUK LAYAR INI
-- ============================================================================
--
--     EXISTS (SELECT substr(claimid,20,29) FROM pooldata.t_claim_adjustment
--              WHERE total_claim*currencyvalue > 5000000000 AND claimid = b.claimid)
--
-- Ambang **Rp 5.000.000.000**, di-hardcode di dalam rule. Di sini ia datang lewat
-- parameter binding dari `casestudyclaim.LargeClaimThreshold`, supaya terlihat dan dapat
-- diuji — lihat catatan di konstanta itu soal `D-15` dan master `F-4` yang belum ada.
--
-- `substr(claimid,20,29)` pada daftar SELECT-nya dibuang: isi daftar SELECT sebuah EXISTS
-- tidak pernah dievaluasi, dan angka 20 itu sendiri adalah panjang awalan
-- `'ASM-FW-GCNMFW-WORK '` — utang teknis `03-CURRENT-ARCHITECTURE.md` §4.1 yang tidak
-- perlu ikut dibawa hanya untuk dibuang lagi.
--
-- ============================================================================
-- TUJUH HAL YANG BERUBAH DARI KUERI LAMA, DAN ALASANNYA
-- ============================================================================
--
-- 1. PARAMETER BINDING, bukan perangkaian teks SQL.
--    Kueri lama menyisipkan DUA potongan klausa WHERE yang dirakit activity:
--    `{ASIS:TempView.City}` (status) dan `{ASIS:TempSearch.UserTeknis}` (bisnis).
--    `08-TECHNICAL-STRATEGY.md` §4.3 melarangnya tanpa perkecualian. Penggantinya pola
--    `:n IS NULL OR (:m = '<kode>' AND <syarat>)`, sehingga satu teks SQL melayani
--    seluruh gabungan penyaring tanpa satu pun nilai dirangkai.
--
-- 2. NVL -> COALESCE, dan TO_CHAR dibuang seluruhnya (§4.3, portabilitas).
--    Kolom "Bulan Klaim" di Pega adalah `to_char(b.REGISTERDATE,'mm')`. Di sini ia
--    `EXTRACT(MONTH FROM b.REGISTERDATE)`, dan nol di depannya ditambahkan di Go.
--    Bentuk itu dipilih bukan sekadar demi portabilitas: `EXTRACT` membaca nilai yang
--    TERSIMPAN, persis seperti `TO_CHAR`. Mengambil bulannya di Go dari timestamp yang
--    sudah dikonversi ke WIB akan menggeser bulan pada registrasi di sekitar pergantian
--    bulan — kelas kesalahan `R-12`, yang justru hendak dicegah aturan §4.3.
--
-- 3. JOIN eksplisit, bukan koma.
--    `FROM a, b, d WHERE a.noklaim=b.claimno AND b.businesscode=d.id` menjadi INNER JOIN.
--    Semantiknya IDENTIK — termasuk akibatnya: klaim yang `BUSINESSCODE`-nya tidak punya
--    baris di `POOLDATA.BUSINESS` TIDAK MUNCUL. Itu direplikasi apa adanya (`P-5`), dan
--    ditulis di sini karena ia cara sebuah baris menghilang tanpa satu pun galat.
--
-- 4. NILAI UANG DIKEMBALIKAN DALAM SATUAN TERKECIL, sebagai BILANGAN BULAT.
--    Setiap nilai uang dikalikan 100 lalu `ROUND`. Alasannya bukan gaya:
--    `money.FromSQLValue` MENOLAK `float64` yang berdesimal — ketepatannya tidak dapat
--    dijamin — dan seluruh nilai di sini adalah hasil `SUM` atas perkalian, yang hampir
--    selalu berdesimal. Mengembalikannya sebagai satuan terkecil membuat nilainya sampai
--    ke Go sebagai bilangan bulat, yaitu bentuk penyimpanan tipe `money.Money` itu
--    sendiri, sehingga tidak ada langkah yang dapat kehilangan ketepatan.
--    Pembulatan ke sen terjadi di Oracle, tempat aritmetikanya memang sudah dikerjakan
--    dalam NUMBER desimal.
--    Hal yang sama berlaku untuk ASM SHARE: dikali 10.000, empat desimal (`D-51`).
--
-- 5. PAGINASI, yang di Pega tidak ada di SQL.
--    Grid lama ber-`pyPageSize 20` dan memaginasi page list klipboard — seluruh hasil
--    ditarik lebih dulu (`K-34`). Di sini `OFFSET … FETCH NEXT` (§3.3) memotongnya di
--    basis data.
--    Akibat sampingannya justru menguntungkan: kedelapan subkueri terkorelasi kini hanya
--    dihitung untuk 20 baris satu halaman, bukan untuk seluruh hasil. Itulah sebabnya
--    kueri daftar berbentuk dua lapis — baris dipilih dan dipotong di lapisan dalam,
--    nilainya dihitung di lapisan luar.
--
-- 6. ORDER BY, yang di Pega TIDAK ADA sama sekali.
--    Tanpa urutan pasti, paginasi mengulang dan melewatkan baris — Oracle tidak menjamin
--    urutan yang sama antara dua eksekusi. Diurutkan menurut nomor klaim, lalu kunci
--    klaim sebagai pemutus seri. Ini PENAMBAHAN, bukan pembacaan, dan disebut di sini
--    supaya tidak terbaca sebagai selisih pada uji kesetaraan.
--
-- 7. `to_number(...)` pada kolom NILAI KLAIM NET ASM SHARE dibuang.
--    Isinya sudah NUMBER; pembungkus itu tidak mengubah apa pun dan tidak portabel.
--
-- ============================================================================
-- PENYARING RENTANG: HANYA TAHUNNYA YANG DIPAKAI
-- ============================================================================
--
--     and A.THNREGIS BETWEEN to_char(to_date({awal},'dd/mm/yyyy'), 'yyyy')
--                        AND to_char(to_date({akhir},'dd/mm/yyyy'),'yyyy')
--
-- Pengguna memilih dua TANGGAL, dan kueri membuang hari serta bulannya. Memilih
-- 01/01/2024 sampai 31/01/2024 mengembalikan **seluruh tahun 2024**.
--
-- Perilakunya direplikasi (`P-5`, keputusan Work Owner 2026-09-26). Yang berubah: tahunnya
-- diturunkan di Go — di satu tempat, dari zona waktu yang diserahkan pemanggil — lalu
-- dikirim sebagai dua teks empat digit. Dengan begitu `TO_DATE`/`TO_CHAR` hilang dari SQL
-- dan pergeseran zona waktu tidak dapat terselip (`R-12`).
--
-- `A.THNREGIS` dibandingkan terhadap TEKS, persis seperti kueri lama membandingkannya
-- terhadap keluaran `TO_CHAR`. Tipe kolomnya belum pernah diterima (`R-08`); bila ia
-- NUMBER, Oracle mengubah teksnya menjadi angka dan hasilnya sama.
--
-- ============================================================================
-- PENANDA PARAMETER
-- ============================================================================
--
-- Tiap kemunculan bernomor SENDIRI. Penanda yang dipakai ulang dengan nomor yang sama
-- ditolak Oracle dengan **ORA-01008**: driver mengikat argumen menurut urutan KEMUNCULAN
-- penanda, bukan menurut nomornya.
--
--     :1  :2         tahun awal & tahun akhir
--     :3 … :7        kode bisnis   (NULL = tidak menyaring)
--     :8 … :10       kode status   (NULL = tidak menyaring)
--     :11            ambang nilai klaim, dalam RUPIAH
--     :12 :13        offset & limit — hanya pada case_study_list

-- name: case_study_count
-- Jumlah seluruh klaim yang cocok, untuk bilah halaman.
SELECT COUNT(*)
  FROM POOLDATA.PEGA_DASHBOARDPNC a
       INNER JOIN POOLDATA.T_CLAIM_PNC b
               ON b.CLAIMNO = a.NOKLAIM
       INNER JOIN POOLDATA.BUSINESS d
               ON d.ID = b.BUSINESSCODE
 WHERE b.CLAIMNO IS NOT NULL
   AND a.THNREGIS BETWEEN :1 AND :2
   AND ( :3 IS NULL
      OR ( :4 = '346'
           AND b.GROUPPANEL IN ('003', '004', '006', '009')
           AND b.BUSINESSCODE NOT IN ('10145', '10168', '10165', '10164', '10053') )
      OR ( :5 = '002' AND b.GROUPPANEL = '002' )
      OR ( :6 = '005' AND b.GROUPPANEL = '005' )
      OR ( :7 = '003'
           AND b.GROUPPANEL = '003'
           AND b.BUSINESSCODE IN ('10076', '10077', '10007', '10011', '10083',
                                  '10141', '10131', '10126', '10055', '10075') ) )
   AND ( :8 IS NULL
      OR ( :9  = 'progress-accept' AND a.STSKLAIM <> '3' )
      OR ( :10 = 'reject'          AND a.STSKLAIM =  '3' ) )
   AND EXISTS ( SELECT 1
                  FROM POOLDATA.T_CLAIM_ADJUSTMENT adj
                 WHERE adj.CLAIMID = b.CLAIMID
                   AND adj.TOTAL_CLAIM * adj.CURRENCYVALUE > :11 )

-- name: case_study_list
--
-- Satu halaman klaim beserta seluruh nilai telaahnya.
--
-- Dua lapis, dan itu disengaja — lihat butir 5 di kepala berkas. Lapisan DALAM memilih dan
-- memotong barisnya; lapisan LUAR menghitung kedelapan agregat, sehingga agregat itu hanya
-- dihitung untuk baris yang benar-benar dikirim.
--
-- Kedelapan agregat dipertahankan sebagai subkueri TERKORELASI, tidak disatukan menjadi
-- satu `GROUP BY` yang di-join. Menyatukannya berarti mengagregat SELURUH
-- `T_CLAIM_ADJUSTMENT` lebih dulu — tabel nilai settlement seluruh klaim — untuk
-- mendapatkan 20 baris. Bentuk terkorelasi menembak indeks `CLAIMID` sebanyak 20 kali dan
-- hasilnya identik dengan kueri lama, baris demi baris.
SELECT base.CLAIM_NUMBER,
       base.POLICY_NUMBER,
       base.INSURED_NAME,
       base.BUSINESS_NAME,
       base.POLICY_PERIOD,
       base.CLAIM_MONTH,
       base.LOSS_DATE,
       base.BUSINESS_SOURCE,
       base.REINSURER_CODE,
       base.CAUSE_OF_LOSS,
       ROUND(base.TSI * 100)                                          AS TSI_MINOR,
       base.BRANCH_NAME,
       base.CLAIM_STATUS_CODE,
       base.CHRONOLOGY,
       base.REMARK,

       -- ASM SHARE — persentase dikali 10.000 (empat desimal, `D-51`).
       ( SELECT ROUND(MAX(adj.ASM_SHARE) * 10000)
           FROM POOLDATA.T_CLAIM_ADJUSTMENT adj
          WHERE adj.CLAIMID = base.CLAIM_KEY )                        AS ASM_SHARE_E4,

       -- Deductible <- SUM(INDIVIDUAL_RISK_VALUE * CURRENCYVALUE)
       ( SELECT ROUND(SUM(adj.INDIVIDUAL_RISK_VALUE * adj.CURRENCYVALUE) * 100)
           FROM POOLDATA.T_CLAIM_ADJUSTMENT adj
          WHERE adj.CLAIMID = base.CLAIM_KEY )                        AS DEDUCTIBLE_MINOR,

       -- Nilai share ASM — perhatikan ASM_SHARE dipakai PER BARIS di dalam SUM, berbeda
       -- dari kolom NET ASM SHARE di bawah yang memakai MAX atas seluruh baris. Keduanya
       -- memang berbeda di kueri lama, dan perbedaannya dipertahankan.
       ( SELECT ROUND(SUM(adj.TOTAL_CLAIM * adj.CURRENCYVALUE * (adj.ASM_SHARE / 100)) * 100)
           FROM POOLDATA.T_CLAIM_ADJUSTMENT adj
          WHERE adj.CLAIMID = base.CLAIM_KEY )                        AS ASM_SHARE_VALUE_MINOR,

       -- NILAI KLAIM 100% <- SUM(TOTAL_CLAIM * CURRENCYVALUE)
       ( SELECT ROUND(SUM(adj.TOTAL_CLAIM * adj.CURRENCYVALUE) * 100)
           FROM POOLDATA.T_CLAIM_ADJUSTMENT adj
          WHERE adj.CLAIMID = base.CLAIM_KEY )                        AS CLAIM_VALUE_MINOR,

       -- ADJUSTER FEE 100% SHARE <- SUM(GROSSVALUE * CURRENCYVALUE)
       ( SELECT ROUND(SUM(adj.GROSSVALUE * adj.CURRENCYVALUE) * 100)
           FROM POOLDATA.T_CLAIM_ADJUSTMENT adj
          WHERE adj.CLAIMID = base.CLAIM_KEY )                        AS ADJUSTER_FEE_MINOR,

       -- LACK OF DOC / SALVAGE / RECOVERY / SUBROGARATION
       --   SUM(COALESCE(LOC,0)/100 * TOTAL_CLAIM * CV) + SUM(COALESCE(NILAI_SALVAGE_A*CV,0))
       ( SELECT ROUND(( SUM((COALESCE(adj.LOC, 0) / 100) * adj.TOTAL_CLAIM * adj.CURRENCYVALUE)
                      + SUM(COALESCE(adj.NILAI_SALVAGE_A * adj.CURRENCYVALUE, 0)) ) * 100)
           FROM POOLDATA.T_CLAIM_ADJUSTMENT adj
          WHERE adj.CLAIMID = base.CLAIM_KEY )                        AS LACK_OF_DOC_MINOR,

       -- NILAI KLAIM NET 100%
       --   total - own retention - lack of doc + salvage - adjuster fee
       ( SELECT ROUND(( SUM(adj.TOTAL_CLAIM * adj.CURRENCYVALUE)
                      - SUM(adj.INDIVIDUAL_RISK_VALUE * adj.CURRENCYVALUE)
                      - SUM((COALESCE(adj.LOC, 0) / 100) * adj.TOTAL_CLAIM * adj.CURRENCYVALUE)
                      + SUM(COALESCE(adj.NILAI_SALVAGE_A * adj.CURRENCYVALUE, 0))
                      - SUM(adj.GROSSVALUE * adj.CURRENCYVALUE) ) * 100)
           FROM POOLDATA.T_CLAIM_ADJUSTMENT adj
          WHERE adj.CLAIMID = base.CLAIM_KEY )                        AS NET_CLAIM_MINOR,

       -- NILAI KLAIM NET ASM SHARE — rumus net di atas dikali MAX(ASM_SHARE/100).
       ( SELECT ROUND(( SUM(adj.TOTAL_CLAIM * adj.CURRENCYVALUE)
                      - SUM(adj.INDIVIDUAL_RISK_VALUE * adj.CURRENCYVALUE)
                      - SUM((COALESCE(adj.LOC, 0) / 100) * adj.TOTAL_CLAIM * adj.CURRENCYVALUE)
                      + SUM(COALESCE(adj.NILAI_SALVAGE_A * adj.CURRENCYVALUE, 0))
                      - SUM(adj.GROSSVALUE * adj.CURRENCYVALUE) )
                      * MAX(adj.ASM_SHARE / 100) * 100)
           FROM POOLDATA.T_CLAIM_ADJUSTMENT adj
          WHERE adj.CLAIMID = base.CLAIM_KEY )                        AS NET_CLAIM_ASM_MINOR

  FROM ( SELECT a.NOKLAIM                             AS CLAIM_NUMBER,
                b.CLAIMID                             AS CLAIM_KEY,
                a.NOPOLIS                             AS POLICY_NUMBER,
                b.QQNAME                              AS INSURED_NAME,
                d.NOTE                                AS BUSINESS_NAME,
                a.THNREGIS                            AS POLICY_PERIOD,
                EXTRACT(MONTH FROM b.REGISTERDATE)    AS CLAIM_MONTH,
                a.DATEOFLOSS                          AS LOSS_DATE,
                b.SOBNAME                             AS BUSINESS_SOURCE,
                a.REINSURER                           AS REINSURER_CODE,
                a.COL_DESC                            AS CAUSE_OF_LOSS,
                a.TSI                                 AS TSI,
                a.CABANG                              AS BRANCH_NAME,
                a.STSKLAIM                            AS CLAIM_STATUS_CODE,
                b.KRONOLOGI                           AS CHRONOLOGY,
                b.REMARKRECOMENDATION                 AS REMARK
           FROM POOLDATA.PEGA_DASHBOARDPNC a
                INNER JOIN POOLDATA.T_CLAIM_PNC b
                        ON b.CLAIMNO = a.NOKLAIM
                INNER JOIN POOLDATA.BUSINESS d
                        ON d.ID = b.BUSINESSCODE
          WHERE b.CLAIMNO IS NOT NULL
            AND a.THNREGIS BETWEEN :1 AND :2
            AND ( :3 IS NULL
               OR ( :4 = '346'
                    AND b.GROUPPANEL IN ('003', '004', '006', '009')
                    AND b.BUSINESSCODE NOT IN ('10145', '10168', '10165', '10164', '10053') )
               OR ( :5 = '002' AND b.GROUPPANEL = '002' )
               OR ( :6 = '005' AND b.GROUPPANEL = '005' )
               OR ( :7 = '003'
                    AND b.GROUPPANEL = '003'
                    AND b.BUSINESSCODE IN ('10076', '10077', '10007', '10011', '10083',
                                           '10141', '10131', '10126', '10055', '10075') ) )
            AND ( :8 IS NULL
               OR ( :9  = 'progress-accept' AND a.STSKLAIM <> '3' )
               OR ( :10 = 'reject'          AND a.STSKLAIM =  '3' ) )
            AND EXISTS ( SELECT 1
                           FROM POOLDATA.T_CLAIM_ADJUSTMENT adj
                          WHERE adj.CLAIMID = b.CLAIMID
                            AND adj.TOTAL_CLAIM * adj.CURRENCYVALUE > :11 )
          ORDER BY a.NOKLAIM, b.CLAIMID
         OFFSET :12 ROWS FETCH NEXT :13 ROWS ONLY ) base
 ORDER BY base.CLAIM_NUMBER, base.CLAIM_KEY

-- name: case_study_save_remark
--
-- Menuliskan catatan telaah satu klaim.
--
-- # SATU-SATUNYA pernyataan yang menulis di modul ini
--
-- Sumbernya `RDB List/SaveRemarksRecommendation_sql-SQL.xml` apa adanya:
--
--     update POOLDATA.T_CLAIM_PNC
--        set REMARKRECOMENDATION = {TEMPSaveRemark.City}
--      where claimno = {TEMPSaveRemark.CaseID}
--
-- Alias `City` pada properti penampungnya tidak ada hubungannya dengan isinya — salah satu
-- dari sekian alias menyesatkan pada layar ini.
--
-- # Tabelnya MILIK PEGA
--
-- `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem. Penulisan ini karena itu
-- menuntut serah-terima kepemilikan tulis atas SATU KOLOM lewat `D-63`: permintaan
-- tertulis, persetujuan Work Owner, pelaksanaan DBA. Presedennya sudah berjalan — modul
-- Inbox Receive TKA menulis `TGLDOKLENGKAP` pada tabel yang sama.
--
-- # Penyaringnya CLAIMNO, bukan CLAIMID
--
-- Berbeda dari modul Inbox Receive TKA, yang memakai `CLAIMID` karena kunci itulah yang
-- baru saja dibacanya dari tabel kerja Pega. Di sini `CLAIMNO` dipakai karena itulah yang
-- dikirim tombol Save di Pega (`Param.CASE` <- `.CaseID` <- `A.NOKLAIM`), dan mengubahnya
-- menjadi kunci lain berarti memperbaiki baris yang berbeda dari yang diperbaiki Pega.
--
-- Keunikan `CLAIMNO` tidak dibuktikan DDL mana pun (`R-08`). Jumlah baris terpengaruh
-- karena itu DIPERIKSA di Go: nol berarti klaimnya tidak ada, dan lebih dari satu berarti
-- nomor klaim ternyata tidak unik — keadaan yang harus terlihat, bukan lewat begitu saja.
UPDATE POOLDATA.T_CLAIM_PNC
   SET REMARKRECOMENDATION = :1
 WHERE CLAIMNO = :2

-- name: case_study_check_table
--
-- Membuktikan keempat tabel beserta kolom yang dipakai modul ini ada dan dapat dibaca.
--
-- Dipakai `claimpnc -periksa`. `FETCH FIRST 0 ROWS ONLY`: yang diperiksa adalah apakah
-- pernyataannya dapat diurai dan dijalankan, bukan isinya — menarik satu baris berarti
-- membaca nama tertanggung tanpa keperluan.
--
-- Berharga justru karena ia menyentuh keempat tabel sekaligus, termasuk
-- `T_CLAIM_ADJUSTMENT` yang hanya muncul di dalam subkueri. Hak baca yang kurang pada
-- salah satunya baru terlihat saat pengguna menekan "Lihat Data" — kecuali diperiksa
-- lebih dulu di sini.
SELECT a.NOKLAIM,
       a.NOPOLIS,
       a.THNREGIS,
       a.DATEOFLOSS,
       a.COL_DESC,
       a.CABANG,
       a.TSI,
       a.STSKLAIM,
       a.REINSURER,
       b.CLAIMID,
       b.CLAIMNO,
       b.QQNAME,
       b.SOBNAME,
       b.KRONOLOGI,
       b.REGISTERDATE,
       b.REMARKRECOMENDATION,
       b.GROUPPANEL,
       b.BUSINESSCODE,
       d.ID,
       d.NOTE,
       ( SELECT SUM(adj.TOTAL_CLAIM * adj.CURRENCYVALUE
                    + adj.GROSSVALUE
                    + adj.INDIVIDUAL_RISK_VALUE
                    + adj.ASM_SHARE
                    + COALESCE(adj.LOC, 0)
                    + COALESCE(adj.NILAI_SALVAGE_A, 0))
           FROM POOLDATA.T_CLAIM_ADJUSTMENT adj
          WHERE adj.CLAIMID = b.CLAIMID ) AS ADJUSTMENT_COLUMNS
  FROM POOLDATA.PEGA_DASHBOARDPNC a
       INNER JOIN POOLDATA.T_CLAIM_PNC b
               ON b.CLAIMNO = a.NOKLAIM
       INNER JOIN POOLDATA.BUSINESS d
               ON d.ID = b.BUSINESSCODE
 FETCH FIRST 0 ROWS ONLY
