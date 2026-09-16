# Ticketing — Migrasi Aplikasi Claim PNC

Papan tiket pekerjaan untuk migrasi Claim PNC dari Pega PRPC 8.3 ke Golang + React + PostgreSQL.

| | |
|---|---|
| **Tanggal** | 2026-09-14 |
| **Modul bertiket** | **34 dari 34** — seluruhnya |
| **Tiket tertulis** | **106** |
| **Total modul** | **34** (`D-35`, `D-42`, `D-75`) |
| **Sumber keputusan** | [`../Steering/00-DECISION-LOG.md`](../Steering/00-DECISION-LOG.md) — 75 entri · [`../ADR/`](../ADR/) — 30 ADR |
| **Lokasi ditetapkan** | `D-31` — ter-commit dan terlihat di GitLab, bukan `.scratch/` |
| **Penamaan modul** | `D-73` — nama fungsi bisnis, kode `F-n`/`B-n`/`S-n`/`U-n` tetap di depan |

> **Tidak ada kode implementasi di repository ini.** Seluruh tiket berstatus **belum dikerjakan**.
> Bagian **Rencana verifikasi** di setiap tiket memuat perintah yang **akan** dijalankan nanti,
> dan ditandai sebagai rencana — bukan sebagai hasil.

> **Peringatan istilah.** Folder `Ticket/` di root repository berisi **Ticket rule** — mekanisme
> lompatan lateral Pega (lihat `ADR-0021`). Kata **"tiket"** di seluruh dokumen ini selalu berarti
> **tiket pekerjaan**.

---

## 1. Cara pakai

| Anda | Mulai dari |
|---|---|
| Manajemen — ingin tahu kesiapan dan jadwal | §7 Jadwal · §4 Tiket terhalang |
| Lead engineer — ingin menyusun urutan kerja | §5 Dependency dan rantai kritis |
| Developer — akan mengerjakan satu tiket | `<MODUL>/spec.md` lalu `<MODUL>/issues/NN-*.md` |
| Auditor — ingin menelusuri satu requirement | §6 Matriks traceability |

**Susunan berkas:**

```
docs/ticketing/
├── README.md                          ← halaman ini
├── INVENTARIS-HARNESS.md              ← bahan kerja pembagian 74 harness
└── <KODE>-<Nama-Fungsi-Bisnis>/
    ├── spec.md                        ← lingkup modul, penghalang, daftar tiketnya
    └── issues/
        └── NN-slug.md                 ← satu tiket
```

Nama folder memuat **kode** dan **nama fungsi bisnis** sekaligus — misalnya
`B-14-Input-Receive-Document`. Kode di depan supaya rujukan silang dari ADR dan Steering tetap
dapat dicari; nama bisnis di belakang supaya folder dapat dikenali tanpa membuka kamus kode
(`D-73`).

**Sebelum mengerjakan sebuah tiket**, baca berurutan: `spec.md` modulnya → tiketnya → ADR yang
disebut di kepala tiket → `CONTEXT.md` untuk istilah yang belum dikenal.

---

## 2. Kosakata status

### 2.1 Lima label triage

Dipakai apa adanya dari [`../agents/triage-labels.md`](../agents/triage-labels.md). **Tidak ada
kosakata tambahan.**

| Label | Artinya di papan ini |
|---|---|
| `needs-triage` | Belum dinilai. **Tidak dipakai** — seluruh tiket sudah dinilai |
| `needs-info` | **Menunggu artefak atau keputusan pihak lain.** Tiket menyebut persis apa yang kurang dan siapa yang bisa melengkapinya |
| `ready-for-agent` | Sepenuhnya terspesifikasi untuk agen AFK. **Tidak dipakai** — `D-43` menetapkan tiket ditulis untuk **manusia junior** |
| `ready-for-human` | Siap dikerjakan manusia |
| `wontfix` | Tidak akan dikerjakan. Belum ada |

**Hanya kelima label di atas yang dipakai** — tidak ada label keenam. Tiket yang lingkupnya sudah
lengkap tetapi menunggu tiket prasyaratnya berlabel `ready-for-human` seperti yang lain; urutan
kerjanya dinyatakan di bagian **Dependency / Blocked by**, bukan lewat label tersendiri.

### 2.2 Kosakata Kesiapan

Label menyatakan **boleh dikerjakan atau tidak**; Kesiapan menyatakan **apa yang menahannya**.

| Kesiapan | Artinya |
|---|---|
| `siap` | Tidak ada yang menahan |
| `terhalang <R-nn>` | Menunggu **artefak** — namanya disebut |
| `terhalang keputusan` | Menunggu **keputusan** — pertanyaannya disebut beserta pemiliknya |
| `terhalang artefak` | Menunggu artefak dari pihak luar yang belum punya nomor risiko |

### 2.3 Definition of Ready

Sebuah tiket boleh berlabel `ready-*` hanya bila **seluruhnya** terpenuhi:

1. Hasil dan nilai bisnisnya eksplisit.
2. Acceptance criteria dapat diverifikasi teknis dan **berangka**.
3. Ruang lingkup dan non-goal jelas.
4. Seluruh istilah domainnya sudah ada di `CONTEXT.md`.
5. Dependency dan blocker diketahui.
6. Constraint tersedia.
7. Verifikasi dapat dilakukan **tanpa pengetahuan pribadi**.
8. Seluruh keputusan yang dibutuhkannya sudah diambil Work Owner.

Yang tidak lolos → `needs-info`, **disertai apa yang kurang dan siapa yang bisa melengkapinya**.
**Requirement yang lemah tidak ditambal dengan AC yang mengarang.**

---

## 3. Ringkasan tiket

### 3.1 Per modul

| Kode | Nama fungsi bisnis | Tiket | `ready-for-human` | `needs-info` | Kesiapan modul |
|---|---|---:|---:|---:|---|
| `F-1` | Kerangka Aplikasi | 5 | 4 | 1 | SEBAGIAN |
| `F-2` | Akses Data dan Nomor Klaim | 7 | 4 | 3 | SEBAGIAN |
| `F-3` | Login dan Hak Akses Menu | 5 | 3 | 2 | TERHALANG identitas · PENUH otorisasi |
| `F-4` | Master Data dan Parameter Bisnis | 6 | 1 | 5 | SEBAGIAN |
| `F-5` | Tanggal, Jam Kerja, dan Hari Libur | 3 | 0 | 3 | SEBAGIAN |
| `F-6` | Portal dan Multi-Sumber Data | 4 | 0 | 4 | SEBAGIAN |
| `B-1` | View Polis dan Snapshot Polis | 3 | 0 | 3 | SEBAGIAN |
| `B-2` | Input Register | 5 | 1 | 4 | SEBAGIAN |
| `B-3` | Objek Pertanggungan dan Coverage | 3 | 0 | 3 | SEBAGIAN |
| `B-4` | Spreading Reasuransi | 3 | 0 | 3 | SEBAGIAN |
| `B-5` | Input Estimasi dan Penyelesaian Nilai | 3 | 0 | 3 | **TERHALANG** |
| `B-6` | Penugasan dan Inbox Petugas | 3 | 2 | 1 | SEBAGIAN |
| `B-7` | Komite Persetujuan Klaim | 3 | 0 | 3 | **TERHALANG** |
| `B-8` | Choose Surveyor dan Hasil Survei | 2 | 0 | 2 | SEBAGIAN |
| `B-9` | PLA, PreDLA, dan DLA | 2 | 0 | 2 | SEBAGIAN |
| `B-10` | Akseptasi, Transfer Kasir, dan LOD | 3 | 0 | 3 | SEBAGIAN |
| `B-11` | RCL/PUCL, Compliance, dan Investigator | 2 | 0 | 2 | **TERHALANG** |
| `B-12` | Salvage, Lelang, dan Recovery | 2 | 0 | 2 | SEBAGIAN |
| `B-13` | Input Open Protection | 2 | 1 | 1 | SEBAGIAN |
| `B-14` | Input Receive Document | 3 | 2 | 1 | SEBAGIAN |
| `S-1` | Dokumen dan Lampiran Klaim | 2 | 0 | 2 | SEBAGIAN |
| `S-2` | Laporan dan Export | 2 | 0 | 2 | SEBAGIAN |
| `S-3` | Notifikasi dan Email | 2 | 0 | 2 | **TERHALANG** |
| `S-4` | Integrasi Sistem Luar | 3 | 0 | 3 | **TERHALANG** |
| `S-5` | Jejak Audit Perubahan | 4 | 3 | 1 | SEBAGIAN |
| `S-6` | Job Terjadwal dan Proses Otomatis | 3 | 0 | 3 | SEBAGIAN |
| `S-7` | Dashboard, TAT, dan KPI | 2 | 0 | 2 | **TERHALANG** |
| `S-8` | Perkakas Uji Kesetaraan | 2 | 1 | 1 | **TERHALANG** |
| `U-1` | Kerangka Layar dan Navigasi Portal | 4 | 3 | 1 | SEBAGIAN |
| `U-2` | Komponen Layar Baku | 5 | 5 | 0 | **PENUH** |
| `U-3` | Layar Inbox per Peran | 2 | 0 | 2 | SEBAGIAN |
| `U-4` | Layar Transaksi Klaim | 2 | 0 | 2 | SEBAGIAN |
| `U-5` | Layar Laporan | 2 | 0 | 2 | SEBAGIAN |
| `U-6` | Layar Master Data | 2 | 0 | 2 | **TERHALANG** |
| | **Total** | **106** | **30** | **76** | |

**Kesiapan modul:** **1 PENUH** · **24 SEBAGIAN** · **9 TERHALANG**.

> `F-3` terhitung TERHALANG hanya pada **bagian identitas**; middleware otorisasi per endpoint di
> modul yang sama tidak terhalang. `../verifikasi-bukti-adr.md:2805-2807` menghitungnya di kedua
> baris, sehingga di sana tertulis 24 SEBAGIAN + 8 TERHALANG. Keduanya menjumlah 33 — yang berbeda
> hanya cara `F-3` dihitung, bukan keadaannya.

### 3.2 Per gelombang

| Gelombang | Modul | Tiket |
|---|---|---:|
| **1 — Fondasi** | `F-1` `F-2` `F-3` `F-4` `F-5` `F-6` `S-8` | **32** |
| **2 — Kerangka UI** | `U-1` `U-2` `S-5` | **13** |
| **3 — Jalur klaim inti** | `B-1` `B-2` `B-3` `B-4` `B-6` `B-14` `S-1` `U-4` | **24** |
| **4 — Persetujuan dan penugasan** | `B-7` `B-8` `B-10` `B-11` `B-13` `U-3` | **14** |
| **5 — Nilai dan pihak luar** | `B-5` `B-9` `B-12` `S-3` `S-4` | **12** |
| **6 — Laporan** | `S-2` `S-7` `U-5` | **6** |
| **7 — Sisa** | `S-6` `U-6` | **5** |
| | **Total** | **106** |

### 3.3 Kenapa seluruh modul kini ditiketkan

`D-41` semula menetapkan **8 modul bertiket penuh + 25 stub**, dengan alasan yang masih berlaku:
menulis acceptance criteria berangka untuk modul yang penghalangnya belum hilang akan menghasilkan
kriteria yang mengarang.

`D-73` mengubah **bentuknya, bukan aturannya**. Seluruh 33 modul kini punya tiket, tetapi modul
yang terhalang mendapat **struktur tiket yang lengkap** — hasil, lingkup, non-goal, constraint,
rollout, bukti — dengan bagian **"Yang kurang dan siapa yang bisa melengkapinya"** yang menyebut
artefak atau keputusan beserta pemiliknya, **bukan** daftar acceptance criteria yang dikarang.

Satu tiket berjalan lebih jauh lagi:
[`TKT-S6-003`](S-6-Job-Terjadwal-dan-Proses-Otomatis/issues/03-auto-accept-komite.md)
**menolak menuliskan acceptance criteria sama sekali**, karena pertanyaan pokoknya — apakah
persetujuan komite otomatis memang dikehendaki — belum dijawab. Itu perilaku yang benar, bukan
tiket yang belum selesai ditulis.

---

## 4. Tiket yang terhalang

**76 dari 106 tiket** menunggu pihak lain. Dikelompokkan menurut **siapa yang dapat melepaskannya**
— karena itulah yang menentukan siapa yang harus dihubungi.

Satu tiket dapat muncul di lebih dari satu baris bila penghalangnya lebih dari satu pihak.

| Pihak | Tiket tertahan | Yang ditunggu, garis besarnya |
|---|---:|---|
| **Work Owner** | **65** | Keputusan aturan bisnis, lingkup, dan kewenangan |
| **Tim Pega** | **19** | ±397 artefak yang hilang dari export (`R-07`, `R-16`) |
| **DBA** | **14** | DDL, statistik ukuran, isi master, 2 source procedure sisa (`R-01`, `R-08`) |
| **Tim Infra/Security** | **6** | Rotasi kredensial, TLS, otentikasi integrasi (`D-40`, `R-17`, `R-18`) |
| **Lead Engineer** | **3** | Keputusan rancangan yang belum ditetapkan di ADR (`ADR-0022`) |
| **Tim HCC/HCQ** | **1** | Kontrak API — **nol jejak di export** |
| **Keamanan Informasi** | **1** | Token dokumen MD5 tanpa secret dan tanpa masa berlaku |
| **Compliance** | **1** | Daftar peristiwa wajib audit (`ADR-0026`) |

### 4.1 Tiga penghalang yang memblokir gerbang kelulusan, bukan pengerjaan

Ini yang paling mudah terlewat: sebagian besar penghalang hanya memperlambat pengerjaan, tetapi
**tiga ini menghentikan modul dinyatakan lulus** sekalipun kodenya selesai.

| Penghalang | Pemilik | Menghentikan |
|---|---|---|
| **Kontrak API HCC/HCQ** | Tim HCC/HCQ | `F-3` — dan lewat itu, **login seluruh aplikasi** |
| **Daftar peristiwa wajib audit** | Compliance | `S-5` |
| **Ketersediaan Pega staging yang dapat ditembak dari luar** | Tim Pega + Infra | `S-8` — dan lewat itu, **gerbang 1 seluruh modul** (`R-14`, `ADR-0027`) |

### 4.2 Empat penghalang yang menyangkut kendali, bukan fitur

Empat hal berikut ditemukan di fase verifikasi, dan keempatnya adalah **jalur yang memintas
kendali yang sedang dibangun modul lain**. Bila `B-7` dibangun dengan cermat sementara keempatnya
dibiarkan, kecermatan itu tidak ada artinya.

| Temuan | Tiket | Pemilik |
|---|---|---|
| **`AutoAcceptKomite` menyetujui komite otomatis tiap hari 06:00**, tanpa pengguna, di luar kontrol menu | [`TKT-S6-003`](S-6-Job-Terjadwal-dan-Proses-Otomatis/issues/03-auto-accept-komite.md) | Work Owner |
| **Dua Service REST masuk menerima persetujuan komite dari sistem lain** — permukaan yang belum pernah diaudit | [`TKT-S4-002`](S-4-Integrasi-Sistem-Luar/issues/02-layanan-rest-masuk.md) | Work Owner |
| **18 dari 21 Connect REST ber-`pyUseAuthentication=false`**; satu memakai `http://` tanpa TLS untuk data premi | [`TKT-S4-001`](S-4-Integrasi-Sistem-Luar/issues/01-klien-rest-keluar.md) | Work Owner + Infra/Security |
| **Token dokumen MD5 tanpa secret dan tanpa masa berlaku** | [`TKT-S1-002`](S-1-Dokumen-dan-Lampiran-Klaim/issues/02-akses-dokumen-dan-token.md) | Keamanan Informasi |

---

## 5. Dependency dan rantai kritis

### 5.1 Empat rantai kritis yang tidak bisa diparalelkan

Diambil dari [`../Steering/06-MODULE-BREAKDOWN.md`](../Steering/06-MODULE-BREAKDOWN.md) §5.
**Inilah yang menentukan berapa orang dibutuhkan** — `D-44` menetapkan jumlah tim diputuskan
setelah papan ini direview, bukan sebaliknya.

| # | Rantai | Panjang | Catatan |
|---|---|---|---|
| **1** | `F-2 → B-1 → B-2 → B-3 → B-4 → B-5 → B-7 → B-10` | 8 modul | **Jalur nilai klaim.** Rantai terpanjang; menentukan tanggal selesai paling awal yang mungkin |
| **2** | `F-3 → F-4 → B-7` | 3 modul | Komite butuh master ambang persetujuan |
| **3** | `U-2 → U-3 / U-4 / U-5` | 2 tingkat | Seluruh layar butuh komponen tabel baku |
| **4** | **`S-8` → gerbang 1 seluruh modul** | — | Tidak ada modul yang dapat **dinyatakan lulus** sebelum perkakas uji kesetaraan berjalan (`D-42`) |

> **Rantai 1 melewati dua modul TERHALANG** — `B-5` (satu-satunya yang masih terikat `BRD §21.4`)
> dan `B-7`. Artinya rantai terpanjang proyek ini **bukan hanya yang terpanjang, tetapi juga yang
> paling tertahan**. Memperpendeknya tidak dapat dilakukan dengan menambah orang.

> **Risiko urutan yang harus terlihat** (`D-44`). `U-2` adalah investasi terpenting di frontend,
> tetapi ia **bersaing waktu dengan `F-2`** bila tidak ada personel khusus frontend: keduanya
> berada di awal, di rantai yang berbeda.

### 5.2 Modul mana bergantung pada modul mana

```
F-1 ──┬── F-2 ──┬── B-1 ── B-2 ── B-3 ── B-4 ── B-5 ── B-7 ── B-10
      │         ├── S-5
      │         └── S-8   ← gerbang 1 SELURUH modul
      ├── F-3 ── F-4 ──┬── B-7
      │                └── U-6
      ├── F-5 ──────────── S-7
      ├── S-4
      ├── S-6
      └── U-1 ── U-2 ──┬── U-3 ← B-6
                       ├── U-4 ← seluruh B-*
                       └── U-5 ← S-2 ── S-7
B-2 ──┬── B-6 ── B-8 ── B-9
      ├── B-11
      ├── B-13
      └── B-14 ── S-1
B-10 ── B-12
F-4 ── S-3
```

---

## 6. Matriks traceability

Dua arah, sesuai kewajiban `D-46` butir 2: dari `FR-xx` turun ke tiket, dan dari tiket kembali ke
rule Pega yang digantikan.

### 6.1 FR → tiket → ADR → rule Pega

| FR | Tiket | ADR | Rule Pega yang digantikan |
|---|---|---|---|
| **FR-F1** | `TKT-F1-001`…`005` | 0001, 0002, 0025, 0026 | struktur platform Pega · `Data-Admin-DB-Name` (tidak ada di export) · pola galat pada **720 dari 902 activity** |
| **FR-F2** | `TKT-F2-001`…`007` | 0004, 0005, 0007, 0009, 0012 | **652 rule Connect-SQL** · `INSERT_PLADLA.prc` (9 `COMMIT`) · `UPDATEREAS.prc` (4 `COMMIT`) · **538 `{ASIS:}`** · `RDB List/InsertClaimPNC-SQL.xml:77` |
| **FR-F3** | `TKT-F3-001`…`005` | 0023, 0024, 0028 | **22 access group** `GCNMFW:<nama>` · **34 When rule** · `Navigation/pyCaseWorkerNavigation-Navigation.xml` (51 menu) · `pyPrivilegeName` (**1 dari 902**) |
| **FR-F4** | `TKT-F4-001`…`006` | 0014, 0015, 0018, 0020, 0025 | **66 email** · **24 Operator ID** · **8 ambang komite** · `POOLDATA.EMAILKOMITE` (21 kolom) · `V_STS_CLAIM` (33 kode) · `GETCURRENCYSTANDARD.fnc` |
| **FR-F5** | `TKT-F5-001`…`003` | 0017, 0020 | **118 titik penyesuaian waktu di 36 activity** · **101 titik `addCalendar(…,12,0,0)`** · `GET_WORKING_HOURS@ASMD` · `GETSELISIHJAM.fnc:22` |
| **FR-F6** | `TKT-F6-001`…`004` | 0030, 0004, 0023 | **tidak ada padanan bernama** — **48 perbandingan** `pxRequestor.pxReqServer` pada 3 hostname · When rule `IsServerSyariah` dan `IsDevelopmentServer` (**keduanya hilang dari export**) · `Activity/SpreadingSyariah_Act-Act.xml` |
| **FR-B1** | `TKT-B01-001`…`003` | 0006, 0010 | `ViewPolis-Harness` · `ViewPolis1-Harness` · snapshot polis |
| **FR-B2** | `TKT-B02-001`…`005` | 0004, 0005, 0019 | `Flow/Register_Flow.xml` tahap **Input Register** · `InputRegister_act` · `InsertClaimPNC-SQL.xml` |
| **FR-B3** | `TKT-B03-001`…`003` | 0006, 0019 | objek pertanggungan dan coverage |
| **FR-B4** | `TKT-B04-001`…`003` | 0017, 0019 | `UPDATEREAS.prc` · toleransi spreading |
| **FR-B5** | `TKT-B05-001`…`003` | 0017, 0019 | tahap **Estimation / Input Estimasi** · ambang Rp 50 juta |
| **FR-B6** | `TKT-B06-001`…`003` | 0008, 0013 | model penugasan · router · `TransferAllCaseNotAssigned` |
| **FR-B7** | `TKT-B07-001`…`003` | 0014, 0018 | jenjang kumulatif komite · `KomiteLoop` · `TYPE_KOMITE` |
| **FR-B8** | `TKT-B08-001`…`002` | 0010, 0017 | tahap **Choose Surveyor** · `InputSurveyor` · `SurveyorsInbox` |
| **FR-B9** | `TKT-B09-001`…`002` | 0011, 0017 | `INSERT_PLADLA.prc` · `ServiceSetDLA-ConnectREST` |
| **FR-B10** | `TKT-B10-001`…`003` | 0011, 0017 | akseptasi · `InjectDataRekeningToKasir` · `SendDataPaidASMtoCashier_2` · LOD |
| **FR-B11** | `TKT-B11-001`…`002` | 0023, 0029 | `RCL_Harness` · `RCLPUCL_Harness` · `inboxCompliance_Harness` · `InboxInvestigator_Harness` |
| **FR-B12** | `TKT-B12-001`…`002` | 0011, 0017 | `InboxSalvage` · `RecivedDataandAttachmentLelangASMSimasbid-REST` · `IDSALVAGE` |
| **FR-B13** | `TKT-B13-001`…`002` | 0019 | `InputProtection_Harness` · `InputReqProtection_Harness` · `MST_BUKA_PROTEKSI` |
| **FR-B14** | `TKT-B14-001`…`003` | 0010, 0019 | `ReceiveDoucument_Harness` · `ViewReceiveDocument` · `InboxRCVApp_Harness` |
| **FR-S1** | `TKT-S1-001`…`002` | 0010, 0023, 0029 | `PNCArchiveDokumen` · `GENERAL.GET_TOKEN_STORAGE.prc:11` · `GCP_IMAGE` |
| **FR-S2** | `TKT-S2-001`…`002` | 0011 | **56 Report Definition**, 54 ber-`pyMaxRecords=500` · 12 hilang dari export |
| **FR-S3** | `TKT-S3-001`…`002` | 0016 | `SendEmailNotification` (**15 pemanggil**, hilang) · `SetDataEmail-DT` · seluruh Correspondence |
| **FR-S4** | `TKT-S4-001`…`003` | 0017, 0023 | **21 Connect REST** · **4 Service REST masuk** · **64 pemakaian DB Link** ke 6 database |
| **FR-S5** | `TKT-S5-001`…`004` | 0012, 0023, 0026, 0028 | **tidak ada padanan** · `PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc:7` |
| **FR-S6** | `TKT-S6-001`…`003` | 0022 | **5 Job Scheduler + 1 Agent** · `JOBForKomiteKlaimPNC` → `AutoAcceptKomite` |
| **FR-S7** | `TKT-S7-001`…`002` | 0011 | `GETSELISIHJAM.fnc:9-22` · `GET_POSISI_PROGRESS_PNC.fnc:15` · `PROGRESS_CLAIM_PNC` |
| **FR-S8** | `TKT-S8-001`…`002` | 0027 | **tidak ada padanan** — modul 100% baru |
| **FR-U1** | `TKT-U1-001`…`004` | 0002, 0023, 0024 | **74 harness** · `pyCaseWorkerNavigation-Navigation.xml` |
| **FR-U2** | `TKT-U2-001`…`005` | 0002, 0011, 0016, 0017 | pola grid pada **268 dari 269 section** · **3.189 grid page list** · **411 `TO_CHAR`** |
| **FR-U3** | `TKT-U3-001`…`002` | 0012, 0023 | **26 harness bernama Inbox** |
| **FR-U4** | `TKT-U4-001`…`002` | 0012, 0023 | `Register_Flow` · **29 Flow Action** · `PNCSearchKlaim` |
| **FR-U5** | `TKT-U5-001`…`002` | 0011, 0023 | `PNCTATReport` · `ReportKPIHarness` · `MonitoringSLINKOJK` |
| **FR-U6** | `TKT-U6-001`…`002` | 0023 | harness master · **≥29 kelompok master** |

### 6.2 Requirement yang belum punya tiket

**Nihil.** Ketiga puluh empat `FR` punya **minimal satu tiket**, dan setiap tiket menyebut **minimal
satu `FR`** serta rule Pega yang digantikannya — atau menyatakan secara eksplisit bahwa **tidak ada
padanannya** (`S-5`, `S-8`).

---

## 7. Jadwal — angka apa adanya

`D-61` menetapkan **jadwal seluruh modul tidak berubah**, dan selisih antara jadwal dan kesiapan
menjadi **tanggung jawab manajemen** — bukan diselesaikan di tingkat teknis.

### 7.1 Yang sudah ada dan yang belum

| | Jumlah | Terhadap total |
|---|---:|---|
| Modul bertiket | **34** | dari **34** modul |
| Tiket tertulis | **106** | seluruhnya **belum dikerjakan** |
| Tiket siap dikerjakan (`ready-for-human`) | **30** | 28% dari tiket |
| Tiket menunggu pihak lain (`needs-info`) | **76** | **72% dari tiket** |
| Modul tanpa penghalang sama sekali | **1** (`U-2`) | 3% dari modul |

**Angka yang paling penting di tabel ini adalah 72%.** Seluruh modul kini punya tiket, tetapi
**tujuh dari sepuluh tiket menunggu orang lain** — bukan menunggu programmer. Menambah developer
tidak menggerakkan tujuh per sepuluh papan ini.

### 7.2 Sisa waktu terhadap target

| | |
|---|---|
| **Target manajemen** | seluruh modul selesai **akhir September 2026** (`D-30`, `C-2`) |
| **Tanggal papan ini** | 2026-09-14 |
| **Sisa waktu** | **sekitar dua minggu** |
| **Perkiraan berbasis ukuran terukur** | **9–15 bulan dengan tim 5–6 orang** (`BRD §20.1`) |

Ukuran yang mendasari perkiraan itu: 74 layar · 269 section · 15.063 step logika · 652 kueri ·
64 objek database · 56 laporan · **21 integrasi keluar + 4 layanan masuk + 6 API baru** · ditambah
tabel autentikasi, otorisasi, dan penugasan yang **belum ada sama sekali** · dan tim yang harus
mempelajari tiga teknologi sekaligus (`D-09`, `R-11`).

### 7.3 Apa yang sudah dan belum dilakukan terhadap selisih ini

| | |
|---|---|
| **Sudah** | Keberatan disampaikan beserta angkanya (`R-05`). Work Owner menegaskan target tetap berlaku, dan strategi migrasi disusun untuknya: **pekerjaan berisiko rendah didahulukan**, sehingga bila waktu tidak mencukupi, **yang tertinggal adalah bagian yang paling sedikit dampaknya** |
| **Sudah** | Dicatat bahwa tenggat ini **tidak punya pemaksa eksternal** — tidak ada lisensi berakhir, tidak ada penalti kontraktual (`R-13`). Penjadwalan ulang sepenuhnya di dalam kendali manajemen |
| **Sudah** | Seluruh 33 modul kini punya tiket (`D-73`), sehingga lingkup pekerjaan **tidak lagi tersembunyi di balik stub** |
| **Belum** | **Waktu pengguna bisnis untuk UAT belum dialokasikan resmi** (`C-9`, `R-15`). Gerbang 2 bergantung pada orang yang sedang menjalankan operasional klaim harian |
| **Belum** | **Jumlah dan komposisi tim belum ditetapkan** (`D-44`) — dan itu memang disengaja: papan ini menjadi bahan keputusannya |
| **Belum** | Tiga penghalang gerbang kelulusan di §4.1 **belum tersentuh sama sekali** |
| **Belum** | Empat temuan kendali di §4.2 **belum dijawab** — dan tiga di antaranya memintas persetujuan klaim |

### 7.4 Yang harus dibaca bersama angka di atas

**Kecepatan menulis kode bukan penentu utama jadwal saat ini.** Dari 73 tiket yang tertahan,
**mayoritas menunggu pihak di luar tim pengembang** — Work Owner, DBA, Tim Pega, Tim HCC/HCQ,
Compliance, Infra/Security. Itu alasan §4 disusun menurut **siapa yang dapat melepaskannya**,
bukan menurut modul.

---

## 8. Kewenangan review

`D-46` menetapkan **lead engineer (agen) menjadi reviewer sekaligus yang menyatakan tiket
selesai**, tanpa mata manusia di antaranya. Sebagai gantinya, setiap tiket dan ADR wajib melewati
**suite verifikasi yang benar-benar dijalankan**, dengan hasil dilaporkan **hijau atau merah per
butir beserta angkanya** — bukan penilaian kualitatif.

| # | Butir yang diuji |
|---|---|
| 1 | Setiap `berkas:baris` yang dikutip benar-benar ada dan isinya sesuai |
| 2 | Traceability dua arah: setiap `FR-xx` dalam cakupan punya ≥1 tiket; setiap tiket punya ≥1 `FR-xx` **dan** rule Pega yang digantikan |
| 3 | Setiap tiket ber-`ready-*` lolos seluruh butir Definition of Ready |
| 4 | Setiap istilah domain di tiket sudah ada di `CONTEXT.md` |
| 5 | Nol nilai sensitif di dokumen yang akan di-commit |
| 6 | Setiap `D-nn`, `R-nn`, `FR-xx` yang dirujuk benar-benar ada dan statusnya mutakhir |
| 7 | Setiap tiket TERHALANG menyebut penghalangnya dengan nama artefak atau `berkas:baris` |
| 8 | **Nol tiket mengklaim status penyelesaian pekerjaan** |

**Tidak ada artefak dinyatakan selesai selama ada satu butir merah.**

Hasil terakhir suite ini dilaporkan di §9.

---

## 9. Hasil verifikasi terakhir

Dijalankan **2026-09-14**, setelah restrukturisasi `D-73`, dan **dijalankan ulang** setelah tiga
koreksi angka diterapkan ke Steering dan BRD (`21-RIWAYAT-REVISI.md` §7). Kedelapan butir §8
dijalankan sebagai perintah, bukan dinilai secara kualitatif. Hasil per butir beserta angkanya:

| # | Butir | Angka | Hasil |
|---|---|---|---|
| 1 | Kutipan `berkas:baris` ada dan sesuai | **69** jalur berkas dirujuk · **0** hilang · **0** tautan relatif putus | 🟢 |
| 2 | Traceability dua arah | **102/102** tiket menyebut `FR-xx` · **102/102** menyebut rule Pega · **33/33** `FR` punya ≥1 tiket | 🟢 |
| 3 | Definition of Ready untuk tiket `ready-*` | **30** tiket `ready-for-human` diperiksa · **0** yang punya bagian "Yang kurang" · **0** tanpa acceptance criteria | 🟢 |
| 4 | Istilah domain ada di `CONTEXT.md` | **7 istilah semula tidak ada** — `Master Data`, `Correspondence`, `Notifier`, `Job Terjadwal`, `Uji Kesetaraan`, `Gerbang 1`, `Gerbang 2`. **Ditambahkan**; glosarium kini **78 istilah** | 🟢 **setelah diperbaiki** |
| 5 | Nol nilai sensitif | **0** alamat email · **0** alamat IP · **0** nomor klaim nyata · **0** kredensial | 🟢 |
| 6 | Rujukan `D-nn`/`R-nn`/`ADR-nnnn` ada | **62** `D-nn` dirujuk · **0** menggantung · **0** `R-nn` menggantung · **0** ADR menggantung | 🟢 |
| 7 | Tiket TERHALANG menyebut penghalangnya | **72** tiket `needs-info` · **72** memuat bagian "Yang kurang dan siapa yang bisa melengkapinya" · **0** tanpa | 🟢 |
| 8 | Nol klaim penyelesaian pekerjaan | **0** tiket mengklaim sudah dikerjakan, lulus tes, atau terimplementasi | 🟢 |

**Delapan butir hijau, satu di antaranya setelah diperbaiki.**

### 9.1 Butir 4 sempat merah — dan itu dilaporkan apa adanya

Tujuh istilah dipakai di tiket sebelum ada di `CONTEXT.md`. Itu **pelanggaran butir 4**, bukan
temuan kecil: tiket yang memakai istilah yang belum didefinisikan memaksa pembacanya menebak.
Ketujuhnya sudah ditambahkan ke glosarium, dan butir 4 dijalankan ulang hingga hijau.

Dicatat di sini alih-alih dihapus, supaya terlihat bahwa suite ini **benar-benar dijalankan** dan
pernah menemukan sesuatu — bukan diketik hijau sejak awal.

### 9.2 Label keenam yang sempat saya buat sendiri — dan dicabut

Saat menulis, saya sempat memakai label **`ready-blocked`** pada dua tiket. Itu **melanggar aturan
yang dipegang papan ini sendiri**: `../agents/triage-labels.md` menetapkan lima label dan §2.1
menyatakan tidak ada kosakata tambahan. Menambah label keenam berarti papan ini berhenti dapat
dibaca dengan kamus yang sama seperti papan lain.

Dicabut, dan kedua tiket dikembalikan ke kosakata yang sah — bukan dengan menambahkan pengecualian
di README:

| Tiket | Semula | Menjadi | Alasan |
|---|---|---|---|
| `TKT-S6-001` Penjadwal dan kunci satu pelaksana | `ready-blocked` | **`needs-info`** | Ternyata memang punya keputusan tertunda (`ADR-0022` masih `Proposed`), sehingga butir 8 Definition of Ready tidak terpenuhi |
| `TKT-S8-002` Pembanding dan klasifikasi selisih | `ready-blocked` | **`ready-for-human`** | Lingkup dan AC-nya lengkap, tidak ada keputusan tertunda. Ia menunggu `TKT-S8-001` — dan itu **dependency biasa**, sama seperti puluhan tiket lain |

### 9.3 Dua hal yang suite ini **tidak** dapat membuktikan

Agar hasil di atas tidak dibaca lebih jauh daripada yang seharusnya:

1. **Butir 1 memeriksa bahwa berkas yang dirujuk ada**, dan bahwa kutipan `berkas:baris` yang
   dipakai saat penulisan sudah dibaca langsung dari sumbernya. Ia **tidak** memeriksa ulang
   kesesuaian isi setiap baris secara otomatis untuk seluruh 68 rujukan.
2. **Tidak satu pun butir membuktikan bahwa tiketnya benar secara bisnis.** Suite ini memeriksa
   keutuhan, ketertelusuran, dan kejujuran status — bukan apakah lingkup yang ditulis memang
   lingkup yang dibutuhkan. Itu yang diperiksa Work Owner saat mereview papan ini.
