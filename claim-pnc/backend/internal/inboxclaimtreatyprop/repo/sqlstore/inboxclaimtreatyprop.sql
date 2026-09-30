-- Kueri modul Inbox Claim Treaty Prop: antrean klaim treaty proporsional.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- SELURUH tabel yang dibaca berkas ini milik sistem lama. Tidak ada satu pun pernyataan
-- yang menulis, dan memang tidak boleh ada: selama masa paralel setiap tabel hanya boleh
-- ditulis SATU sistem, dan tabel-tabel ini milik Pega (`P-1`).
--
-- ============================================================================
-- SUMBERNYA REPORT DEFINITION, BUKAN LAGI RULE CONNECT-SQL
-- ============================================================================
--
-- Kedua grid di layar ini dipasok Report Definition masing-masing, diterima 2026-09-30:
--
--	Report Definition/InboxKlaimPropAdmin-RD.xml    kelas Assign-Worklist    -> tab Admin
--	Report Definition/InboxKlaimPropTeknik-RD.xml   kelas Assign-WorkBasket  -> tab Teknik
--
-- Keduanya dibuat 2026-09-09 dan TIDAK dirujuk satu pun rule di export — ia perubahan
-- produksi yang belum masuk snapshot (`R-09`). Ketiga rule Connect-SQL yang sebelumnya
-- dipakai berkas ini (`GetClaimTreaty_SQL`, `GetClaimTreatyAllAdmin_SQL`,
-- `GetClaimTreatyTeknik_SQL`) karena itu TIDAK lagi menjadi acuan.
--
-- Perbedaannya bukan kosmetik. Ketiga rule lama membaca kolom bisnis dari dokumen JSON
-- `POOLDATA.JSON_KLAIM.DATA_JSONBLOB`; kedua RD membacanya dari KOLOM TEREKSPOS pada objek
-- kerja. Keduanya dapat berbeda isinya, dan yang dilihat pengguna hari ini adalah yang kedua.
--
-- ============================================================================
-- PEMETAAN: properti RD -> kolom -> alias di sini
-- ============================================================================
--
-- Awalan `WorkPage.` pada RD adalah hasil JOIN-nya ke kelas
-- `ASM-FW-GCNMFW-Work-ClaimTreaty`; tanpa awalan berarti milik baris penugasan.
--
-- Properti pada RD                                Kolom                  Alias di sini
-- ----------------------------------------------- ---------------------- --------------------
-- .pxRefObjectKey                                 a.PXREFOBJECTKEY       WORK_KEY
-- .pzInsKey                                       a.PZINSKEY             REFERENCE
-- .pxRefObjectInsName                             a.PXREFOBJECTINSNAME   CLAIM_ID
-- .pxAssignedOperatorID                           a.PXASSIGNEDOPERATORID ASSIGNED_OPERATOR
-- WorkPage.ClaimData.IDMaster                     w.MASTERID             MASTER_ID
-- WorkPage.ClaimData.PolicyData.PolicyNo          w.POLICYNO             POLICY_NUMBER
-- WorkPage.ClaimData.DateOfLoss                   w.DATEOFLOSS_1         LOSS_DATE
-- WorkPage.ClaimData.QuotationData.BusinessName   w.BUSINESSNAME         BUSINESS_NAME
-- WorkPage.ClaimData.QuotationData.SobName        w.SOBNAME              BUSINESS_SOURCE
-- WorkPage.ClaimData.QuotationData.CedingCoName   w.CEDINGCONAME         CEDING_COMPANY
-- WorkPage.ClaimData.InsuredName                  w.INSUREDNAME          INSURED_NAME
-- WorkPage.ClaimData.IsSubjectivity               (?) belum diketahui    SUBJECTIVITY
-- WorkPage.pxUpdateOperator                       w.PXUPDATEOPERATOR     LAST_UPDATE_OPERATOR
-- WorkPage.pyStatusWork                           w.PYSTATUSWORK         CLAIM_STATUS
--
-- Nama kolomnya TIDAK tertulis di RD — Pega menyimpan pemetaan properti-ke-kolom di rule
-- Property, dan export hanya memuat SATU rule Property. Nama di atas karena itu diverifikasi
-- dengan cara lain: setiap kolom di kolom tengah benar-benar dipakai rule Pega lain terhadap
-- alias yang menunjuk `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`.
--
-- `DATEOFLOSS_1` berakhiran `_1`, BUKAN `DATEOFLOSS`, dan itu bukan salah ketik: pada seluruh
-- rule Pega yang aliasnya menunjuk tabel objek kerja, yang dipakai selalu bentuk ber-`_1`
-- (19 berkas) dan tidak pernah yang tanpa (0 berkas). Akhiran itu artefak penamaan Pega
-- ketika dua properti berbeda dipetakan ke tabel yang sama.
--
-- `IsSubjectivity` adalah SATU-SATUNYA yang nama kolomnya tidak dapat ditemukan. Ia dikirim
-- sebagai NULL, bukan ditebak: nama kolom yang salah menggagalkan SELURUH kueri dengan
-- ORA-00904 — tab yang tidak dapat dibuka sama sekali, alih-alih satu kolom yang kosong.
--
-- ============================================================================
-- JOIN-nya INNER, dan pembatas jenis klaim pindah ke PXOBJCLASS
-- ============================================================================
--
-- RD menyatakan `JOIN type=INNER` ke kelas `ASM-FW-GCNMFW-Work-ClaimTreaty` dengan kondisi
-- `WorkPage.pzInsKey = .pxRefObjectKey`. Dua hal mengikuti:
--
-- 1. INNER, bukan LEFT. Versi sebelumnya di sini memakai LEFT dengan alasan "penugasan yang
--    objek kerjanya tidak terbaca tetap muncul". Alasan itu DITARIK: yang di-LEFT-join dulu
--    adalah JSON_KLAIM — salinan yang memang boleh belum ada — sedangkan yang di sini objek
--    kerja yang DITUNJUK penugasan itu sendiri. Penugasan tanpa objek kerja adalah data
--    rusak, bukan keadaan normal, dan Pega produksi pun membuangnya.
--
-- 2. Penyaring `PXREFOBJECTKEY LIKE '%CLMP%'` DIHAPUS, diganti `PXOBJCLASS`. JOIN ke sebuah
--    kelas di Pega membatasi barisnya ke kelas itu, dan pembatas itu tegas — sementara
--    `LIKE '%CLMP%'` mencocokkan pola di tengah teks kunci, yang kebetulan bekerja.
--
-- ============================================================================
-- TIDAK ADA PENYARING OPERATOR — dan itu terlihat pengguna
-- ============================================================================
--
-- Kedua RD TIDAK punya satu pun filter selain kondisi join. Tidak ada
-- `pxAssignedOperatorID = <pemanggil>`, dan keenam parameternya (`F_CLAIMID`, `F_CLAIMNO`,
-- `F_NOPOLIS`, `F_INSURED`, `F_SOB`, `F_CEDING`) dideklarasikan tetapi TIDAK dirujuk di mana
-- pun di dalam RD.
--
-- Akibatnya tab Admin menampilkan SELURUH penugasan klaim treaty di portal itu, bukan
-- penugasan pemanggil. Itu pula sebabnya kueri "See All Claim" yang dulu ada di berkas ini
-- dihapus: ia tidak lagi punya lawan untuk ditukar.
--
-- ============================================================================
-- YANG BERUBAH DARI PEGA, DAN KENAPA
-- ============================================================================
--
-- 1. PAGINASI DIKERJAKAN BASIS DATA.
--    Kedua RD memakai `pyMaxRecords = 500` — ia MEMOTONG di 500 baris, bukan memaginasi.
--    Di sini halamannya dipotong dengan `OFFSET … FETCH NEXT … ROWS ONLY`, yang didukung
--    Oracle 12c+ dan PostgreSQL (`09-DATABASE-STRATEGY.md` §3.3). Ini PERUBAHAN PERILAKU
--    yang disadari: baris ke-501 dan seterusnya kini dapat dicapai.
--
-- 2. JUMLAH SELURUH BARIS DIHITUNG `COUNT(*) OVER ()`.
--    Satu perjalanan, bukan dua.
--
-- 3. `ORDER BY` DITETAPKAN TEGAS.
--    Kedua RD tidak menetapkan urutan (`pySortOrder` kosong pada seluruh kolom). Itu dapat
--    dibiarkan selama seluruh baris ditarik sekaligus; begitu halamannya dipotong, urutan
--    yang tidak ditetapkan membuat satu baris muncul di dua halaman sekaligus hilang dari
--    halaman lain.
--
-- 4. NILAI SELALU LEWAT PARAMETER BINDING.
--    Nama kelas objek kerja pun dikirim sebagai bind, bukan ditulis di dalam teks SQL —
--    nilainya satu tempat saja, inboxclaimtreatyprop.WorkClass.
--
-- ============================================================================
-- KE-15 ALIAS WAJIB SAMA DI SETIAP KUERI
-- ============================================================================
--
-- Urutan DAN namanya. Dua hal bergantung padanya:
--
--   * satu pemindai Go melayani kedua kueri (scanWorkItem di inboxclaimtreatyprop.go);
--   * uji query_test.go menjaganya, dan ia gagal bila ada kueri yang aliasnya berbeda.

-- name: list_worklist
-- Tab "Prop Treaty-in Admin" — Report Definition/InboxKlaimPropAdmin-RD.xml
--
-- Bind: :1 kelas objek kerja · :2 offset · :3 jumlah baris
SELECT a.PXREFOBJECTKEY                                          AS WORK_KEY,
       a.PZINSKEY                                                AS REFERENCE,
       a.PXREFOBJECTINSNAME                                      AS CLAIM_ID,
       a.PXASSIGNEDOPERATORID                                    AS ASSIGNED_OPERATOR,
       w.MASTERID                                                AS MASTER_ID,
       w.POLICYNO                                                AS POLICY_NUMBER,
       w.DATEOFLOSS_1                                            AS LOSS_DATE,
       w.BUSINESSNAME                                            AS BUSINESS_NAME,
       w.SOBNAME                                                 AS BUSINESS_SOURCE,
       w.CEDINGCONAME                                            AS CEDING_COMPANY,
       w.INSUREDNAME                                             AS INSURED_NAME,
       CAST(NULL AS VARCHAR2(100))                               AS SUBJECTIVITY,
       w.PXUPDATEOPERATOR                                        AS LAST_UPDATE_OPERATOR,
       w.PYSTATUSWORK                                            AS CLAIM_STATUS,
       COUNT(*) OVER ()                                          AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       INNER JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
               ON w.PZINSKEY = a.PXREFOBJECTKEY
 WHERE w.PXOBJCLASS = :1
 ORDER BY a.PXCREATEDATETIME DESC, a.PXREFOBJECTINSNAME
OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY

-- name: list_workbasket
-- Tab "Prop Treaty-in Teknik" — Report Definition/InboxKlaimPropTeknik-RD.xml
--
-- Ia membaca TABEL YANG BERBEDA: PC_ASSIGN_WORKBASKET, bukan PC_ASSIGN_WORKLIST. Itulah
-- satu-satunya hal yang membedakannya dari kueri di atas, dan itu pula yang membedakan
-- antrean bersama dari penugasan perorangan (`D-26`) — persis seperti kedua RD-nya, yang
-- berbeda hanya pada `pyClassName`.
--
-- Nama akun antrean `TreatyinPNCTeknik` TIDAK lagi disaring di sini: RD-nya tidak
-- menyaringnya, sehingga seluruh antrean bersama klaim treaty ikut tampil.
--
-- Bind: :1 kelas objek kerja · :2 offset · :3 jumlah baris
SELECT a.PXREFOBJECTKEY                                          AS WORK_KEY,
       a.PZINSKEY                                                AS REFERENCE,
       a.PXREFOBJECTINSNAME                                      AS CLAIM_ID,
       a.PXASSIGNEDOPERATORID                                    AS ASSIGNED_OPERATOR,
       w.MASTERID                                                AS MASTER_ID,
       w.POLICYNO                                                AS POLICY_NUMBER,
       w.DATEOFLOSS_1                                            AS LOSS_DATE,
       w.BUSINESSNAME                                            AS BUSINESS_NAME,
       w.SOBNAME                                                 AS BUSINESS_SOURCE,
       w.CEDINGCONAME                                            AS CEDING_COMPANY,
       w.INSUREDNAME                                             AS INSURED_NAME,
       CAST(NULL AS VARCHAR2(100))                               AS SUBJECTIVITY,
       w.PXUPDATEOPERATOR                                        AS LAST_UPDATE_OPERATOR,
       w.PYSTATUSWORK                                            AS CLAIM_STATUS,
       COUNT(*) OVER ()                                          AS TOTAL_ROWS
  FROM DATAPEGA.PC_ASSIGN_WORKBASKET a
       INNER JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
               ON w.PZINSKEY = a.PXREFOBJECTKEY
 WHERE w.PXOBJCLASS = :1
 ORDER BY a.PXCREATEDATETIME DESC, a.PXREFOBJECTINSNAME
OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY

-- name: check_worklist
-- Dipakai perintah `-periksa`: memastikan tabel penugasan perorangan DAN gabungannya ke
-- tabel objek kerja terbaca dari koneksi yang dipakai.
--
-- Ia tidak menyentuh satu baris pun — yang diperiksa adalah hak baca dan keberadaan
-- tabelnya, bukan isinya. Gabungannya ikut diperiksa karena kegagalan yang paling mungkin
-- terjadi bukan "tabel tidak ada" melainkan "hak baca hanya diberikan pada salah satunya".
SELECT COUNT(*) AS PROBE
  FROM DATAPEGA.PC_ASSIGN_WORKLIST a
       INNER JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
               ON w.PZINSKEY = a.PXREFOBJECTKEY
 WHERE 1 = 0

-- name: check_workbasket
-- Dipakai perintah `-periksa`: memastikan tabel antrean bersama terbaca.
--
-- Ia terpisah dari check_worklist karena tabelnya memang berbeda, dan hak baca atas yang
-- satu tidak menyatakan apa pun tentang yang lain. Tab "Prop Treaty-in Teknik" bergantung
-- HANYA pada tabel ini.
SELECT COUNT(*) AS PROBE
  FROM DATAPEGA.PC_ASSIGN_WORKBASKET
 WHERE 1 = 0
