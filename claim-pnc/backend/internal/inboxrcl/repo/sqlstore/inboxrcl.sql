-- Kueri modul Inbox RCL (`MENU_ID 62`, `RCL_Harness`).
--
-- Nama kueri berbahasa Inggris (`D-80`); nama tabel dan kolom tetap seperti aslinya.
-- Tidak ada satu pun pernyataan yang menulis (`P-1`).
--
-- ============================================================================
-- SUMBERNYA POOLDATA.T_CLAIMLIST_ADMIN — BUKAN TABEL PEGA
-- ============================================================================
--
-- Work Owner menetapkan 2026-09-27: modul ini TIDAK membaca DATAPEGA lagi. Report
-- Definition aslinya menggabung dua tabel Pega:
--
--   DATAPEGA.PC_ASM_FW_GCNMFW_WORK  INNER JOIN  DATAPEGA.PC_ASSIGN_WORKLIST
--     ON PXREFOBJECTKEY = PZINSKEY
--
-- T_CLAIMLIST_ADMIN adalah tabel DATAR — satu baris per klaim, sudah membawa pemilik
-- penugasan yang sedang berjalan (`PXASSIGNEDOPERATORID`). Gabungannya karena itu HILANG,
-- beserta dua akibat bawaannya (`docs/kolom-t-claimlist-admin.md` §A): klaim dengan dua
-- penugasan terbuka tidak lagi tampil dua kali. Itu perubahan perilaku yang sudah diterima
-- untuk tabel ini sejak Inbox Outstanding, bukan keputusan baru modul ini.
--
-- ============================================================================
-- TIGA KOLOM BELUM ADA — `migrations/0012_claimlist_admin_rcl.up.sql`
-- ============================================================================
--
-- Katalog 2026-09-27 (53 kolom) tidak memuat satu pun padanan untuk:
--
--   .ClaimData.TanggalAnalystSendRCL          -> TANGGALANALYSTSENDRCL_1   penyaring C + kolom
--   .ClaimData.NamaDokterRCL                  -> NAMADOKTERRCL_1           penyaring D
--   .ClaimData.PUCLStatus.KomentarAnalisator  -> KOMENTARANALISATOR_1      kolom "Deskripsi Analyst"
--
-- Ketiganya diajukan lewat migrasi 0012, yang DIJALANKAN DBA (`D-63`), bukan aplikasi.
-- Sampai itu terjadi, kueri di bawah gagal dengan ORA-00904 yang menyebut kolomnya.
-- Disengaja: menghilangkan penyaring C dan D supaya kuerinya jalan akan menampilkan SELURUH
-- klaim milik pemanggil — Input Register, Survey, Estimasi — sebagai antrean RCL, tanpa satu
-- pun pesan galat. `-periksa` menembak `check_columns` supaya keadaan ini diketahui lebih dulu.
--
-- Menambah kolom juga BELUM MENGISINYA. Proses pengisi T_CLAIMLIST_ADMIN harus membawa
-- ketiganya; dua di antaranya `unexposed` di Pega, sehingga harus dibaca dari blob objek
-- kerja. Isi tabel hari ini pun hanya memuat tiga tahap — Input Register, Choose Surveyor,
-- Input Estimasi — dan nol klaim di tahap RCL Dokter.
--
-- ============================================================================
-- PEMETAAN KOLOM — properti Pega -> kolom T_CLAIMLIST_ADMIN -> alias
-- ============================================================================
--
--   judul di layar         properti Pega                             kolom                      alias
--   ---------------------- ----------------------------------------- -------------------------- -----------------
--   Nomor Case             .pyID                                     k.PYID                     CASE_ID
--   No Polis               .Policy.PolicyNo                          k.POLICYNO                 POLICY_NUMBER
--   Nama Tertanggung       .Policy.QQName                            k.QQNAME                   INSURED_NAME
--   Tanggal Masuk Inbox    .ClaimData.TanggalAnalystSendRCL          k.TANGGALANALYSTSENDRCL_1  SENT_TO_RCL_AT
--   Deskripsi Analyst      .ClaimData.PUCLStatus.KomentarAnalisator  k.KOMENTARANALISATOR_1     ANALYST_NOTE
--
--   tidak digambar         .ClaimData.NamaDokterRCL                  k.NAMADOKTERRCL_1          RCL_DOCTOR
--   tidak digambar         .pxCreateDateTime                         k.PXCREATEDATETIME         REGISTERED_AT
--   tidak digambar         .pyStatusWork                             k.PYSTATUSWORK             PROCESS_STATUS
--   tidak digambar         newAssignPage.pxAssignedOperatorID        k.PXASSIGNEDOPERATORID     ASSIGNED_OPERATOR
--   tidak digambar         .pzInsKey                                 k.PZINSKEY                 REFERENCE
--
-- ============================================================================
-- CATATAN 1 — PXOBJCLASS TETAP DISARING
-- ============================================================================
--
-- T_CLAIMLIST_ADMIN menampung DUA kelas objek kerja (katalog: 2 nilai berbeda), sama seperti
-- tabel Pega asalnya. Report Definition tidak menyaringnya karena engine Pega yang melakukannya.
--
-- ============================================================================
-- CATATAN 2 — IDENTITAS YANG DIBANDINGKAN ADALAH IDENTITAS LAMA
-- ============================================================================
--
-- `assign = TempOperator.City`, diisi `GetpyUserIdentifierFromTable` dari kueri
-- `legacy_operator_for`. Identitas yang sama dipakai penyaring A (pemilik penugasan) dan D
-- (dokter RCL), dikirim sebagai dua parameter supaya tiap penanda bind dipakai sekali.
-- `UPPER(TRIM(...))` mengikuti Inbox Outstanding pada tabel yang sama: keseragaman huruf
-- identitas di sistem lama tidak terjaga (`11-SECURITY.md` §3.1).
--
-- ============================================================================
-- YANG BERUBAH / TIDAK BERUBAH DARI SISTEM LAMA
-- ============================================================================
--
-- Berubah: sumber tabel datar (di atas) · paginasi basis data, bukan `pyMaxRecords = 500` ·
-- `COUNT(*) OVER ()` satu perjalanan · kotak cari (`D-54`) · seluruh nilai lewat parameter
-- binding, termasuk grup akses yang dulu dirangkai `{Asis:TempOperator.AlasanKlaim}`.
--
-- Tidak berubah: hanya `Resolved-Completed` yang dikecualikan (`Resolved-Rejected` tetap
-- muncul) · urutan `PXCREATEDATETIME DESC, PYID DESC`, kedua `pySortType = DESC`.
--
-- CATATAN PENANDA BIND. Gaya Oracle `:n`; belum portabel ke PostgreSQL (`$n`) — utang yang
-- sudah ada sebelum modul ini.

-- name: legacy_operator_for
-- Identitas LAMA seorang petugas — padanan `RDB List/GetOperatorID-SQL.xml` sebagaimana
-- dijalankan `Activity/GetpyUserIdentifierFromTable-Act.xml`:
--
--     SELECT OLD_OPERATOR_ID AS "City" FROM POOLDATA.T_ACCESS_GROUP_PNC
--      WHERE OPERATOR_ID = {OperatorID.pyUserIdentifier} AND STS_AKTIF = '1'
--        AND ACCESS_GROUP IN ('GCNMFW:Administrators','GCNMFW:PNCKomite','GCNMFW:CaseManager')
--        AND ACCESS_GROUP != 'GCNMFW:ViewClaimPNC'
--
-- Tabelnya POOLDATA, bukan DATAPEGA — tidak terkena keputusan 2026-09-27.
--
-- `MAX`, bukan `pxResults(1)`: diukur 2026-09-27, 17 operator aktif pada ketiga grup dan NOL
-- di antaranya punya lebih dari satu OLD_OPERATOR_ID, sehingga hasilnya sama dan deterministik.
-- Kueri milik `inboxoutstanding` TIDAK dipakai karena ia tidak menyaring grup akses.
--
-- Bind:
--   :1  login pemanggil (huruf besar, dipangkas)
--   :2  'GCNMFW:Administrators'
--   :3  'GCNMFW:PNCKomite'
--   :4  'GCNMFW:CaseManager'
--   :5  'GCNMFW:ViewClaimPNC'
SELECT MAX(UPPER(TRIM(g.OLD_OPERATOR_ID)))
  FROM POOLDATA.T_ACCESS_GROUP_PNC g
 WHERE UPPER(TRIM(g.OPERATOR_ID)) = :1
   AND g.STS_AKTIF = '1'
   AND g.ACCESS_GROUP IN (:2, :3, :4)
   AND g.ACCESS_GROUP <> :5

-- name: list_tasks
-- Satu halaman antrean RCL Dokter milik sebuah identitas lama.
--
-- SETIAP penanda bind muncul TEPAT SATU KALI. godror mengikat parameter menurut URUTAN
-- KEMUNCULAN, bukan menurut nomornya — penanda yang dipakai ulang (`:4` tiga kali) gagal
-- dengan ORA-01008 terhadap Oracle sungguhan, dan uji memori tidak dapat menangkapnya.
-- Terbukti 2026-09-27. Karena itu kata kunci dikirim sebagai tiga parameter: penanda NULL,
-- dan dua pola LIKE yang sudah dibentuk dan di-escape di Go (preseden `inboxoutstanding`).
--
-- Bind:
--   :1  identitas lama — penyaring A, pemilik penugasan
--   :2  status kerja yang DIKECUALIKAN — "Resolved-Completed"
--   :3  identitas lama — penyaring D, nama dokter RCL
--   :4  kata kunci, atau NULL bila kotak carinya kosong
--   :5  pola LIKE untuk PYID     ('%KATA%', huruf besar, karakter khusus di-escape)
--   :6  pola LIKE untuk POLICYNO (sama dengan :5)
--   :7  offset
--   :8  jumlah baris
SELECT k.PZINSKEY                    AS REFERENCE,
       k.PYID                        AS CASE_ID,
       k.POLICYNO                    AS POLICY_NUMBER,
       k.QQNAME                      AS INSURED_NAME,
       k.TANGGALANALYSTSENDRCL_1     AS SENT_TO_RCL_AT,
       k.KOMENTARANALISATOR_1        AS ANALYST_NOTE,
       k.NAMADOKTERRCL_1             AS RCL_DOCTOR,
       k.PXCREATEDATETIME            AS REGISTERED_AT,
       k.PYSTATUSWORK                AS PROCESS_STATUS,
       k.PXASSIGNEDOPERATORID        AS ASSIGNED_OPERATOR,
       COUNT(*) OVER ()              AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
 WHERE k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND UPPER(TRIM(k.PXASSIGNEDOPERATORID)) = UPPER(:1)
   AND k.PYSTATUSWORK <> :2
   AND k.TANGGALANALYSTSENDRCL_1 IS NOT NULL
   AND UPPER(TRIM(k.NAMADOKTERRCL_1)) = UPPER(:3)
   AND (:4 IS NULL
        OR UPPER(k.PYID) LIKE :5 ESCAPE '\'
        OR UPPER(k.POLICYNO) LIKE :6 ESCAPE '\')
 ORDER BY k.PXCREATEDATETIME DESC, k.PYID DESC
OFFSET :7 ROWS FETCH NEXT :8 ROWS ONLY

-- name: check_tables
-- Dipakai `-periksa`: memastikan KEDUA tabel terbaca dari koneksi yang dipakai.
-- `WHERE 1 = 0` — yang diperiksa hak baca dan keberadaan tabel, bukan isinya.
SELECT COUNT(*) AS PROBE
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
       CROSS JOIN POOLDATA.T_ACCESS_GROUP_PNC g
 WHERE 1 = 0

-- name: check_columns
-- Dipakai `-periksa`: memastikan KETIGA kolom migrasi 0012 memang ada.
--
-- Terpisah dari check_tables karena sebab gagalnya berbeda — hak baca lawan migrasi yang
-- belum dijalankan. `WHERE 1 = 0` tetap membuat Oracle MEM-PARSE ketiga kolom. Oracle hanya
-- menyebut identifier PERTAMA yang gagal, sehingga galatnya tidak membuktikan kolom lain ada.
SELECT COUNT(k.TANGGALANALYSTSENDRCL_1) AS PROBE_SENT,
       COUNT(k.NAMADOKTERRCL_1)         AS PROBE_DOCTOR,
       COUNT(k.KOMENTARANALISATOR_1)    AS PROBE_NOTE
  FROM POOLDATA.T_CLAIMLIST_ADMIN k
 WHERE 1 = 0
