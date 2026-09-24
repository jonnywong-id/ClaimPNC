# Kolom yang dibutuhkan `POOLDATA.T_CLAIMLIST_ADMIN`

Untuk menggantikan `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` dan `DATAPEGA.PC_ASSIGN_WORKLIST` —
**sebagai tujuan tulis**, bukan hanya sebagai sumber baca.

| | |
|---|---|
| **Tanggal** | 2026-09-22 |
| **Untuk** | DBA dan Work Owner |
| **Sumber** | Katalog Oracle (`ALL_TAB_COLUMNS`, `ALL_TABLES`) dibaca langsung — **bukan** disimpulkan dari SQL |

---

## Keadaan hari ini

| Tabel | Kolom | Kolom **terisi** | Baris |
|---|---:|---:|---:|
| `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` | 186 | **123** | 7.703 |
| `DATAPEGA.PC_ASSIGN_WORKLIST` | 56 | **51** | 259.011 |
| `POOLDATA.T_CLAIMLIST_ADMIN` | **40** | 38 | 1.014 |

> "Terisi" = `NUM_DISTINCT > 0` pada statistik optimizer. Kolom tanpa satu pun nilai berbeda
> tidak pernah diisi.

**Dua angka yang menentukan rancangan:**

1. **259.011 assignment untuk 7.703 klaim** — rata-rata **34 assignment per klaim**. Tabel
   datar menyimpan satu baris per klaim, sehingga ia **hanya dapat membawa assignment yang
   sedang berjalan**, bukan riwayatnya. Itu keputusan yang sudah terkandung dalam
   pendataran, dan perlu disadari sebelum riwayat penugasan dibutuhkan modul lain.
2. **1.014 dari 7.703 klaim** ada di tabel datar — **13%**. Isinya belum lengkap.

---

## A. `PXREFOBJECTKEY` dan `PXREFOBJECTINSNAME` tidak perlu ditambahkan

Keduanya kunci join `worklist → work object`. Karena kedua tabel sudah menyatu:

```
PXREFOBJECTKEY      -> PZINSKEY
PXREFOBJECTINSNAME  -> PYID
```

Ini bukan sekadar penghematan kolom. Join lamanya `INNER`, dan itu membawa dua cacat yang
ikut hilang: klaim tanpa assignment **lenyap dari layar**, dan klaim dengan lebih dari satu
assignment **tampil berkali-kali**.

---

## B. Kolom yang harus ditambahkan

### B.1 Menghidupkan tab layar Inbox — prioritas tertinggi

Ketujuh tab layar rujukan yang belum dapat dibangun, seluruhnya terhalang kolom di bawah.
**Semuanya ADA di `PC_ASM_FW_GCNMFW_WORK`** — hanya belum dibawa ke tabel datar.

| Kolom | Tipe | Nilai berbeda | Menghidupkan |
|---|---|---:|---|
| **`DOKUMENLENGKAP_1`** | VARCHAR2 | 2 | **Complete documents** · **Documents not complete** |
| **`ISPENDINGCLOSE`** | VARCHAR2 | 2 | **Temporary Close** |
| **`SURVEYORTYPE_1`** | VARCHAR2 | 4 | **Internal Surveyor** · **Loss Adjuster** |
| **`ADJUSTERPIC_1`** | VARCHAR2 | 93 | **Loss Adjuster** |
| **`STATUSKOMUNIKASI_1`** | VARCHAR2 | 2 | **Not Answered** · **Replied** |
| `TANGGALDOKLENGKAP` | TIMESTAMP | 181 | tanggal dokumen lengkap |
| `SURVEYORNAME_1` | VARCHAR2 | 101 | nama surveyor |
| `SURVEYORNAMEMARINE_1` | VARCHAR2 | 4 | surveyor lini Marine |
| `ADJUSTERSTATUS_1` | VARCHAR2 | 19 | status adjuster |

Dari `PC_ASSIGN_WORKLIST`, untuk **Deadline To Temporary Close**:

| Kolom | Tipe | Nilai berbeda |
|---|---|---:|
| **`PXDEADLINETIME`** | DATE | 53 |
| `PXGOALTIME` | DATE | 53 |

### B.2 Memperbaiki kolom yang sudah tampil tetapi kosong

| Kolom | Nilai berbeda | Kenapa |
|---|---:|---|
| **`STATUSCLAIM_1`** | **24** | Kode status klaim, dicari ke `v_sts_claim.lsc_id` untuk label **"Status ASM"**. Tabel datar punya `STATUSLOCK_1`, tetapi kolom itu **kosong di seluruh 1.014 baris** — kolom di layar karena itu selalu hampa |

### B.3 Data bisnis yang dipakai laporan

| Kolom | Nilai berbeda | | Kolom | Nilai berbeda |
|---|---:|---|---|---:|
| `PXINSNAME` | 6.810 | | `CASEID_1` | 264 |
| `RECEIVEDDATE_1` | 2.827 | | `REFNO_1` | 192 |
| `DATEOFCOMITEE_1` | 991 | | `BOOKNO_1` | 109 |
| `DATEOFSENTDOCUMENT_1` | 970 | | `SOURCEOFBUSINESS` | 118 |
| `LOCATION_1` | 546 | | `BUSINESSTYPE` | 40 |
| `CLOSECLAIMDATE_1` | 514 | | `CLOSECLAIMNOTE_1` | 126 |
| `ASMCLIENTID_1` | 266 | | `PNCSTATUS_1` | 5 |
| `RESCHEDULEDATE_1` | 320 | | `RESCHEDULELOCATION_1` | 108 |

Alur PUCL/RCL: `RCL_PUCL_1` (3) · `TANGGALKIRIMPUCL_1` (88) ·
`TANGGALCETAKDOKUMENPUCL_1` (74) · `KOMENTARPUCL_1` (30) · `PUCLAPPROVE_1` (2) ·
`KOMENTARANALISATOR_1` (45)

Lain-lain: `STATUSCASE_1` (3) · `STATUSKLAIM_1` (3) · `ASMSTATUS_1` (2) · `EXGRATIA_1` (2) ·
`TKA_1` (2) · `TYPEOFCLAIM_1` (2) · `TYPEPROTECTION` (9) · `NUMBEROFDOCUMENT_1` (8) ·
`ADJUSTERACCEPT_1` (2) · `LAMAKLAIM_1` (86) · `SURVEYDATE_1` (29) ·
`APPOINTMENTDATE_1` (254) · `KETERANGAN_1` (49) · `INPUTDATE` (108)

### B.4 Jejak siapa-kapan

`PXSAVEDATETIME` · `PXUPDATEDATETIME` · `PXCOMMITDATETIME` · `PXUPDATEOPERATOR` (111) ·
`PXUPDATEOPNAME` (111) · `PYORIGUSERID` (90) · `PYRESOLVEDUSERID` (56) ·
`PYRESOLVEDTIMESTAMP` · `PYREOPENTIMESTAMP` (47) · `PYREOPENCOUNT` (7)

Dari worklist: `PYASSIGNMENTSTATUS` (344) · `PXASSIGNEDUSERNAME` (981) · `PYLABEL` (16.156) ·
`PXTASKNAME` (73) · `PYERRORMESSAGE` (100)

### B.5 Data nasabah — perlu keputusan tersendiri

`NOKTP_1` (325) · `EMAIL_1` (212) · `SENDER_1` (181) · `SUBJECTEMAIL_1` (12) ·
`TANGGALSELESAIRAWATINAP_1` (14)

Membawanya ke tabel baru memperluas permukaan data pribadi. `D-64` menetapkan staging
memakai data produksi apa adanya, sehingga penambahan ini ikut tersalin ke staging.

---

## C. Yang sebaiknya **tidak** dibawa

Kolom internal engine Pega — tidak bermakna di luar Pega dan tidak dirujuk satu pun aturan
bisnis:

`PXCOVERINSKEY` · `PXCOVEREDCOUNT` · `PXCOVEREDCOUNTOPEN` · `PXCURRENTSTAGELABEL` ·
`PXCREATESYSTEMID` · `PXUPDATESYSTEMID` · `PXAPPLICATION` · `PXFLOWINSKEY` ·
`PXREFOBJECTCLASS` · `PXREFQUEUEKEY` · `PYINTERESTPAGECLASS` · `PYELAPSED*` ·
`PYSTATUSCUSTOMERSAT` · seluruh `PYORIG*`/`PYOWNER*`/`PYRESOLVED*` yang berupa
org/division/workgroup

---

## D. Pertanyaan yang harus dijawab sebelum menambah kolom

**1. Siapa yang mengisi `T_CLAIMLIST_ADMIN`?**

Tidak satu pun rule atau procedure di export menyentuhnya — **nol kemunculan** di 2.634 XML,
55 `.prc`, dan 8 `.fnc`. Statistik optimizer-nya mencatat 1 baris sementara isinya 1.014,
menandakan tabel yang baru dibuat dan belum pernah di-*gather* ulang.

Selama pengisinya tidak diketahui, menambahkan kolom berisiko: kolomnya ada, tetapi tidak
ada yang mengisinya — persis keadaan `STATUSLOCK_1` dan `REQUESTSURVEY_1` hari ini, yang
**kosong di seluruh 1.014 baris**.

**2. Satu penulis, atau dua?**

`P-1` dan `ADR-0004` menetapkan satu tabel ditulis satu sistem. Bila Go mulai menulis
`T_CLAIMLIST_ADMIN` sementara proses pengisi yang ada juga menulisnya, keduanya akan saling
menimpa — dan kegagalannya tidak menghasilkan galat, hanya data yang berubah sendiri.

**3. Riwayat assignment dibuang atau disimpan terpisah?**

34 assignment per klaim tidak muat dalam satu baris. Bila riwayatnya dibutuhkan — TAT per
tahap, siapa memegang apa dan kapan — ia menuntut tabel tersendiri.

---

## E. Rule yang harus ditulis ulang — 111

Dihitung dari direktori `RDB List/`:

| Kelompok | Jumlah | Akibat penggabungan |
|---|---:|---|
| Memakai **kedua** tabel | **45** | join-nya **hilang** — beserta dua cacat bawaannya |
| Hanya tabel kerja | 63 | `FROM` diganti, kolom sebagian besar sudah tersedia |
| Hanya worklist | 3 | perlu diperiksa: apakah butuh riwayat assignment |

Menurut jenisnya: **44 `Get*`** · **27 `Browse*`** · **17 `Count*`** · 7 `View*` ·
4 `Export*` · 12 lainnya.

Pasangan `Browse*` dan `Count*` biasanya memakai syarat `WHERE` yang sama — itu menurunkan
pekerjaan nyata di bawah 111, karena keduanya ditulis ulang sekali lalu dipakai dua kali.
Modul Inbox Outstanding sudah memperlakukannya begitu: satu berkas `.sql`, dua kueri, syarat
`WHERE` yang dijaga uji agar tetap sama.

### 45 rule yang join-nya hilang

```
BrowseAllInboxPicTeknik          BrowseCaseNotAssigned            BrowseClaimALL
BrowseClaimALLKomunikasi         BrowseClaimNotRegistAll          BrowseInboxOutstanding1
BrowseInboxPicTeknik             BrowseInternalSurveyor           BrowseInternalSurveyorPIC
BrowseKomunikasiPicTeknik        BrowseLossAdjuster               BrowseOSLossAdjusterPIC
BrowseRequestSurvey              CountCaseNotAssigned             CountClaimRegist
CountClaimRegistTravPA           CountClaimRegistTravelClaim      CountInAllInboxPicTeknik
CountInboxPicTeknik              CountInternalSurveyorPIC         CountKomunikasiDokumenCabang
CountKomunikasiPicTeknik         CountOSLossAdjusterPIC           CountOutstandingClaim
CountOutstandingManager          ExportDataDetailKlaim            GcnmBrowseCase_SQL
GcnmCountDataCase_SQL            GetAllCaseAdmin                  GetAllDataTemporaryClose
GetBisnisGroupDashboardOS        GetDataRCVallKlaimPATravel       GetExportDataDetailKlaim
GetExportDataDetailKlaimTempClose                                 GetKlaimCabang
GetKlaimNonPropAdminALL_SQL      GetKlaimNonPropAdminTBA_SQL      GetKlaimNonPropAdmin_SQL
GetKomitePAOutstanding           GetPICDashboardOS                GetProgressAllYearDashboarOS
GetYearDashboardOS               Get_CountInternalSurveyor        Get_CountLostAdjusterClaim
SearchDataClaim
```

`BrowseInboxOutstanding1` sudah selesai — ia yang menjadi modul Inbox Outstanding.

---

## F. Urutan yang disarankan

**1. Jalankan `migrations/0005` tahap 1** — 13 kolom.

Berdiri sendiri, dan langsung membuka tujuh tab layar Inbox plus memperbaiki kolom
"Status ASM" yang hari ini selalu kosong.

**2. Perluas proses pengisi tabel** — dan inilah yang sesungguhnya menahan.

Menurut Work Owner, pengisi `T_CLAIMLIST_ADMIN` **baru berjalan di lingkungan testing**.
Sampai ia mengisi kolom-kolom baru di produksi, kolomnya ada tetapi kosong — persis keadaan
`STATUSLOCK_1` dan `REQUESTSURVEY_1` hari ini.

**3. Tetapkan penulis tunggalnya** sebelum modul Go mana pun mulai menulis.

`P-1` dan `ADR-0004` menetapkan satu tabel ditulis satu sistem. Bila proses pengisi dan
aplikasi Go sama-sama menulis, keduanya saling menimpa **tanpa menghasilkan galat**.

**4. Tulis ulang rule per layar, bukan per rule.**

Pasangan `Browse*`/`Count*` dikerjakan bersama. Urutan yang paling murah: layar yang sudah
punya modul (Inbox Outstanding ✅), lalu dashboard `Count*` yang syaratnya paling sederhana,
baru laporan `Export*` yang kolomnya paling banyak.

**5. Isi tabelnya sampai penuh.** Sekarang 1.014 dari 7.703 klaim — **13%**. Uji kesetaraan
gerbang 1 tidak bermakna selama sumbernya belum selengkap pembandingnya.
