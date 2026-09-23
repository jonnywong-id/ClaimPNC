-- Kueri layar Inbox Close Claim (`MENU_ID 59`, harness `InboxCloseClaim_Harness`).
--
-- ============================================================================
-- HARNESS-NYA TIDAK ADA DI EXPORT
-- ============================================================================
--
-- Yang direkonstruksi di sini bukan tebakan atas layar yang hilang, melainkan salinan dua
-- kueri yang MEMANG ADA beserta activity yang mengisinya:
--
--     RDB List/GcnmBrowseReopenCase_SQL-SQL.xml       kueri daftar
--     RDB List/GCNMCountCloseClaim-SQL.xml            kueri hitung
--     Activity/GCNMGetManagerReopenCase_Act-Act.xml   seluruh penyaringnya
--
-- ============================================================================
-- SUMBER DATA — tiga tabel, tanpa join ke worklist
-- ============================================================================
--
--     DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
--     POOLDATA.BUSINESS              c
--     POOLDATA.BUSINESSGROUP         d
--
-- Persis seperti kueri lamanya, dan sama dengan yang dipakai `inboxadmin`,
-- `inboxlaporanklaim`, `inboxmanagerreceivepucl`, dan `komite`.
--
-- `POOLDATA.T_CLAIMLIST_ADMIN` sengaja TIDAK dipakai (keputusan Work Owner 2026-09-23):
-- tabel datar itu baru terisi 13% dari klaim dan BELUM punya `STATUSCLAIM_1` sampai
-- migrasi `0005` tahap 1 dijalankan DBA — padahal kolom itulah yang dibutuhkan penyaring
-- Status Bayar di bawah.
--
-- Join ke `BUSINESSGROUP d` DIPERTAHANKAN meski tak satu pun kolomnya dibawa. Ia INNER
-- JOIN di kueri lama, sehingga ia MENYARING: klaim yang lini bisnisnya tidak punya baris
-- di `BUSINESS` atau kelompoknya tidak punya baris di `BUSINESSGROUP` tidak muncul.
-- Membuangnya akan menambah baris tanpa satu pun pesan galat.
--
-- ============================================================================
-- PEMETAAN KOLOM — alias lama TIDAK dibawa
-- ============================================================================
--
-- Judul kolom diambil dari `Section/InboxManagerReopen1_Sec-Section.xml`, yang di dalamnya
-- sendiri berjudul "Inbox Close Claim".
--
--   kolom layar         properti section          alias kueri lama        kolom
--   ------------------  ------------------------  ----------------------  -----------------
--   No Klaim            .CaseIDView               CaseIDView              PYID
--   Pilih               .IsSelected / .CaseID     CaseID                  PZINSKEY
--   No Polis            .POLICYNO                 policyno                POLICYNO
--   Nama Tertanggung    .CustomerName             CustomerName (!)        QQNAME
--   Nama Bisnis         .BUSINESSNAME             businessname            BUSINESSNAME
--   Sumber Bisnis       .SourceOfBusinessName     SourceOfBusinessName    SOBNAME
--   Nama Cabang         .BRANCHNAME               branchname              BRANCHNAME
--   Tanggal Pendaftaran .pxCreateDateTime         pxCreateDateTime        PXCREATEDATETIME
--   Lama Waktu Klaim    .pxCreateDateTime (!)     —                       dihitung di Go
--   PIC Teknik          .ReinsurerName            ReinsurerName (!)       USERTEKNIS_1
--   Admin PNC           .MOName                   MOName (!)              PXCREATEOPNAME
--
-- Tanda (!) menandai alias yang artinya BERLAWANAN dengan isinya: "ReinsurerName" berisi
-- PIC Teknik, "MOName" berisi Admin PNC, dan "CoverNo" — yang tidak ditampilkan — berisi
-- TANGGAL KEJADIAN.
--
-- Baris "Lama Waktu Klaim" bertanda (!) karena alasan yang berbeda: di Pega ia terikat
-- `.pxCreateDateTime` dengan format `pxDateTime`, yaitu TANGGAL YANG SAMA PERSIS dengan
-- kolom di sebelahnya. Kolom berjudul durasi yang isinya tanggal. Work Owner memutuskan
-- 2026-09-23 kolom itu diisi umur dalam hari — lihat `ClosedClaim.DurationDays`.
--
-- ============================================================================
-- DUA KOLOM YANG DITAMBAHKAN, DAN SATU PEMFORMATAN YANG DIBUANG
-- ============================================================================
--
-- `CLOSECLAIMDATE_1` dan `PYRESOLVEDTIMESTAMP` TIDAK ada di kueri lama. Keduanya
-- ditambahkan karena "Lama Waktu Klaim" membutuhkan titik akhir: klaim yang tutup tiga
-- tahun lalu bukan klaim berumur seribu hari. Keduanya memang ada di tabel — 514 dan 3.142
-- nilai berbeda (`migrations/0005`).
--
-- `to_char(TRUNC(dateofloss_1),'dd-mm-yyyy')` pada kueri lama DIBUANG. `D-20` menetapkan
-- pemformatan tanggal keluar dari SQL: tanggal yang dikembalikan sebagai teks membuat
-- pengurutan menjadi pengurutan TEKS — `01-12-2024` terbaca lebih kecil daripada
-- `02-01-2020` — dan penyaringan rentang tidak dapat memakai index.
--
-- ============================================================================
-- PENYARING — enam, seluruhnya dari activity
-- ============================================================================
--
-- Panel penyaringnya (`FilterDashboardClaimclose`) TIDAK ADA di export, tetapi activity
-- yang membacanya menyebut keenamnya satu per satu:
--
--   properti activity          penanda di kueri lama        di sini
--   -------------------------  ---------------------------  ----------------------------
--   param.filter2              {ASIS:TempView.pyNote}       :1 :2 :3   kotak cari gabungan
--   param.filterNOPOLIS        {ASIS:TempFilter.CaseID}     :4 :5      No Polis
--   param.filterNOKLAIM        {ASIS:TempFilter.City}       :6 :7      No Klaim
--   param.filterPIC            {ASIS:TempFilter.CityID}     :8 :9      PIC Teknik
--   TempView2.Remark           {ASIS:TempFilter.Country}    :10 … :14  lini bisnis
--   param.filterSTSTRF         {ASIS:TempFilter.District}   :15 :16 :17 status transfer
--   param.filterSTSBAYAR       {ASIS:TempFilter.DistrictID} :18 … :22  status bayar
--
-- Seluruh penanda `{ASIS:…}` itu adalah PERANGKAIAN SQL dari nilai properti klipboard —
-- 538 kemunculan di seluruh export (`K-29`) — dan tidak satu pun dibawa. Di sini setiap
-- nilai lewat parameter binding.
--
-- ============================================================================
-- `STATUSCLAIM_1 <> :22` MEMBUANG BARIS BER-NULL — DAN ITU MEMANG DIKEHENDAKI
-- ============================================================================
--
-- Di Oracle, `NULL <> '1163'` bernilai UNKNOWN, dan baris ber-UNKNOWN dibuang WHERE.
-- Artinya klaim yang `STATUSCLAIM_1`-nya kosong TIDAK muncul saat penyaring "BELUM LUNAS"
-- dipakai.
--
-- Itu perilaku sistem lama, dan **dikonfirmasi Work Owner 2026-09-23 sebagai perilaku yang
-- BENAR** — bukan sekadar ditiru karena `P-5`. Klaim tanpa kode status bukan "belum lunas";
-- ia klaim yang keadaan bayarnya belum diketahui, dan menempatkannya di bawah penyaring
-- "Belum Lunas" akan menyatakan sesuatu yang tidak diketahui sebagai fakta.
--
-- JANGAN "memperbaikinya" menjadi `(… IS NULL OR … <> :22)`. Itu akan menambah baris yang
-- justru tidak boleh ada di sana, dan penambahannya tidak menghasilkan galat apa pun.
--
-- ============================================================================
-- PENOMORAN BIND — tiap KEMUNCULAN punya nomornya sendiri
-- ============================================================================
--
-- Nilai lini bisnis muncul lima kali, status transfer tiga kali, status bayar lima kali.
-- Masing-masing kemunculan diberi NOMOR TERSENDIRI, dan nilainya dikirim berulang dari
-- `filterArgs`.
--
-- Ini mengikuti pola yang TERBUKTI JALAN terhadap Oracle pada modul Inbox Outstanding,
-- satu-satunya modul sejenis yang kuerinya sudah benar-benar menyentuh basis data. Di sana
-- penanda yang diulang dengan nomor yang sama menghasilkan **ORA-01008 not all variables
-- bound** (`inboxoutstanding/repo/sqlstore/outstanding.sql`).
--
-- Nomornya ditulis MENAIK sesuai urutan kemunculan di dalam teks, sehingga penafsiran
-- driver mana pun — menurut nomor atau menurut urutan kemunculan — menghasilkan
-- pengikatan yang sama. Kesesuaian jumlahnya dijaga `query_test.go`.
--
--     close_claim_list   :1 … :22 penyaring · :23 offset · :24 limit
--     close_claim_count  :1 … :22 penyaring
--
-- Kedua kueri WAJIB memakai syarat WHERE yang sama persis — lihat catatan pada
-- close_claim_count.

-- name: close_claim_list
SELECT A.PZINSKEY,
       A.PYID,
       A.POLICYNO,
       A.QQNAME,
       A.BUSINESSNAME,
       A.SOBNAME,
       A.BRANCHNAME,
       A.GROUPPANEL_1,
       c.BUSINESSGROUPID,
       A.PXCREATEDATETIME,
       A.DATEOFLOSS_1,
       A.CLOSECLAIMDATE_1,
       A.PYRESOLVEDTIMESTAMP,
       A.PYSTATUSWORK,
       A.STATUSCLAIM_1,
       (SELECT s.LSC_NOTE FROM POOLDATA.V_STS_CLAIM s
         WHERE s.LSC_ID = A.STATUSCLAIM_1)            AS STATUS_KLAIM_LABEL,
       A.USERTEKNIS_1,
       A.PXCREATEOPNAME,
       CASE WHEN EXISTS (SELECT 1 FROM POOLDATA.T_CLAIM_ADJUSTMENT B
                          WHERE B.TRANSFER_CASHIER_DATE IS NOT NULL
                            AND B.CLAIMID = A.PZINSKEY)
            THEN 1 ELSE 0 END                         AS SUDAH_TRANSFER
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.PYSTATUSWORK IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (A.BRANCHNAME <> 'ASNET' OR A.BRANCHNAME IS NULL)
   AND (:1 IS NULL
        OR UPPER(A.POLICYNO) LIKE :2 ESCAPE '\'
        OR UPPER(A.PYID) LIKE :3 ESCAPE '\')
   AND (:4 IS NULL OR UPPER(A.POLICYNO) LIKE :5 ESCAPE '\')
   AND (:6 IS NULL OR UPPER(A.PYID) LIKE :7 ESCAPE '\')
   AND (:8 IS NULL OR UPPER(A.USERTEKNIS_1) LIKE :9 ESCAPE '\')
   AND (:10 = 'ALL'
        OR (:11 = 'NONMBU'
            AND A.GROUPPANEL_1 IN ('003', '004', '006')
            AND c.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023'))
        OR (:12 = 'BONDING'
            AND c.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023'))
        OR (:13 = 'PA' AND A.GROUPPANEL_1 = '002')
        OR (:14 = 'TRAVEL' AND A.GROUPPANEL_1 = '005'))
   AND (:15 IS NULL
        OR (:16 = 'SUDAH TRANSFER'
            AND EXISTS (SELECT 1 FROM POOLDATA.T_CLAIM_ADJUSTMENT B
                         WHERE B.TRANSFER_CASHIER_DATE IS NOT NULL
                           AND B.CLAIMID = A.PZINSKEY))
        OR (:17 = 'BELUM TRANSFER'
            AND NOT EXISTS (SELECT 1 FROM POOLDATA.T_CLAIM_ADJUSTMENT B
                             WHERE B.TRANSFER_CASHIER_DATE IS NOT NULL
                               AND B.CLAIMID = A.PZINSKEY)))
   AND (:18 IS NULL
        OR (:19 = 'LUNAS' AND A.STATUSCLAIM_1 = :20)
        OR (:21 = 'BELUM LUNAS' AND A.STATUSCLAIM_1 <> :22))
 ORDER BY A.PXCREATEDATETIME ASC, A.PZINSKEY
OFFSET :23 ROWS FETCH NEXT :24 ROWS ONLY

-- name: close_claim_count
-- Menghitung SELURUH baris yang cocok, bukan baris pada halaman ini.
--
-- ============================================================================
-- SATU PENYARING YANG HILANG DI PEGA, DAN DIKEMBALIKAN DI SINI
-- ============================================================================
--
-- `RDB List/GCNMCountCloseClaim-SQL.xml` memuat lima penanda `{ASIS:…}`; kueri daftarnya
-- memuat enam. Yang hilang adalah **`{ASIS:TempFilter.DistrictID}`** — penyaring
-- **Status Bayar**.
--
-- Akibatnya di sistem lama: begitu penyaring Status Bayar dipakai, jumlah total yang
-- ditampilkan TIDAK cocok dengan baris yang benar-benar dapat ditelusuri, dan tidak ada
-- galat yang muncul. Pengguna membaca "247 baris cocok" lalu menemukan jumlah yang lain.
--
-- Work Owner memutuskan 2026-09-23 keduanya disamakan. Ini **selisih terencana** terhadap
-- Pega: angka totalnya akan BERBEDA saat penyaring itu dipakai, dan itu memang yang
-- dikehendaki (`D-54`).
--
-- Syarat WHERE di bawah wajib sama persis dengan close_claim_list. Bila keduanya menyimpang
-- lagi, cacat yang sama kembali — dan `query_test.go` menjaganya baris per baris.
SELECT COUNT(*)
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
       INNER JOIN POOLDATA.BUSINESS c
               ON A.BUSINESSCODE_1 = c.ID
       INNER JOIN POOLDATA.BUSINESSGROUP d
               ON c.BUSINESSGROUPID = d.ID
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.PYSTATUSWORK IN ('Resolved-Completed', 'Resolved-Rejected')
   AND (A.BRANCHNAME <> 'ASNET' OR A.BRANCHNAME IS NULL)
   AND (:1 IS NULL
        OR UPPER(A.POLICYNO) LIKE :2 ESCAPE '\'
        OR UPPER(A.PYID) LIKE :3 ESCAPE '\')
   AND (:4 IS NULL OR UPPER(A.POLICYNO) LIKE :5 ESCAPE '\')
   AND (:6 IS NULL OR UPPER(A.PYID) LIKE :7 ESCAPE '\')
   AND (:8 IS NULL OR UPPER(A.USERTEKNIS_1) LIKE :9 ESCAPE '\')
   AND (:10 = 'ALL'
        OR (:11 = 'NONMBU'
            AND A.GROUPPANEL_1 IN ('003', '004', '006')
            AND c.BUSINESSGROUPID NOT IN ('10008', '10010', '10015', '10023'))
        OR (:12 = 'BONDING'
            AND c.BUSINESSGROUPID IN ('10008', '10010', '10015', '10023'))
        OR (:13 = 'PA' AND A.GROUPPANEL_1 = '002')
        OR (:14 = 'TRAVEL' AND A.GROUPPANEL_1 = '005'))
   AND (:15 IS NULL
        OR (:16 = 'SUDAH TRANSFER'
            AND EXISTS (SELECT 1 FROM POOLDATA.T_CLAIM_ADJUSTMENT B
                         WHERE B.TRANSFER_CASHIER_DATE IS NOT NULL
                           AND B.CLAIMID = A.PZINSKEY))
        OR (:17 = 'BELUM TRANSFER'
            AND NOT EXISTS (SELECT 1 FROM POOLDATA.T_CLAIM_ADJUSTMENT B
                             WHERE B.TRANSFER_CASHIER_DATE IS NOT NULL
                               AND B.CLAIMID = A.PZINSKEY)))
   AND (:18 IS NULL
        OR (:19 = 'LUNAS' AND A.STATUSCLAIM_1 = :20)
        OR (:21 = 'BELUM LUNAS' AND A.STATUSCLAIM_1 <> :22))

-- name: close_claim_exists
-- Memastikan sebuah klaim benar-benar ADA dan benar-benar SUDAH TUTUP.
--
-- Dipanggil sebelum permintaan ReOpen atau Copy Klaim dicatat. Tanpa pemeriksaan ini,
-- permintaan dapat diajukan atas kunci klaim apa pun yang diketik pemanggil — termasuk
-- klaim yang masih berjalan, yang justru tidak boleh dibuka kembali karena belum tutup.
--
-- Syarat tutupnya SAMA PERSIS dengan kedua kueri di atas; itulah yang membuat "ada di
-- layar ini" dan "boleh diajukan" tidak dapat berselisih.
--
-- Bind: :1 PZINSKEY
SELECT A.PYID
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK A
 WHERE A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND A.PYSTATUSWORK IN ('Resolved-Completed', 'Resolved-Rejected')
   AND A.PZINSKEY = :1
