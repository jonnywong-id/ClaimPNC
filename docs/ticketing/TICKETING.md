# Ticketing — Migrasi Aplikasi Claim PNC

**Papan Tiket Pekerjaan · Dokumen Gabungan**

| | |
|---|---|
| **Dokumen** | Ticketing migrasi Claim PNC dari Pega PRPC 8.3 ke Golang + React + PostgreSQL |
| **Tanggal dibangun** | 2026-09-30 |
| **Jumlah modul** | **34** |
| **Jumlah tiket** | **106** — 76 `needs-info` · 30 `ready-for-human` |
| **Sumber** | `docs/ticketing/README.md` · `docs/ticketing/<modul>/spec.md` · `docs/ticketing/<modul>/issues/*.md` · `docs/ticketing/INVENTARIS-HARNESS.md` |
| **Klasifikasi** | CONFIDENTIAL |

> **Dokumen ini dibangun otomatis** oleh `docs/tools/build-ticketing.js`. Jangan disunting
> langsung — suntingan hilang pada pembangunan berikutnya. Yang disunting adalah berkas sumbernya
> di `docs/ticketing/`.

---

## Daftar Isi

1. Papan Tiket — ringkasan, status, dependency, traceability, jadwal
2. F-1 — Kerangka Aplikasi — 5 tiket
3. F-2 — Akses Data — 7 tiket
4. F-3 — Identitas & Akses — 5 tiket
5. F-4 — Master Data — 6 tiket
6. F-5 — Waktu & Zona Waktu — 3 tiket
7. F-6 · Portal & Multi-Sumber Data — 4 tiket
8. B-1 · View Polis & Snapshot Polis — 3 tiket
9. B-2 · Input Register — Registrasi Klaim — 5 tiket
10. B-3 · Objek Pertanggungan & Coverage — 3 tiket
11. B-4 · Spreading Reasuransi & Koasuransi — 3 tiket
12. B-5 · Input Estimasi & Penyelesaian Nilai Klaim — 3 tiket
13. B-6 · Penugasan & Inbox Petugas — 3 tiket
14. B-7 · Komite Persetujuan Klaim — 3 tiket
15. B-8 · Choose Surveyor & Hasil Survei — 2 tiket
16. B-9 · PLA, Pre-DLA & DLA ke Reasuransi — 2 tiket
17. B-10 · Akseptasi, Transfer Kasir & LOD — 3 tiket
18. B-11 · RCL, PUCL, Compliance, Investigator & Analyst Doctor — 2 tiket
19. B-12 · Salvage, Lelang & Recovery — 2 tiket
20. B-13 · Input Open Protection — Buka Proteksi — 2 tiket
21. B-14 · Input Receive Document — Penerimaan Dokumen Fisik — 3 tiket
22. S-1 · Dokumen & Lampiran Klaim — 2 tiket
23. S-2 · Laporan & Export — 2 tiket
24. S-3 · Notifikasi & Korespondensi Email — 2 tiket
25. S-4 · Integrasi Sistem Luar — 3 tiket
26. S-5 — Jejak Audit — 4 tiket
27. S-6 · Job Terjadwal & Proses Otomatis — 3 tiket
28. S-7 · Dashboard Klaim, TAT & KPI — 2 tiket
29. S-8 · Perkakas Uji Kesetaraan — 2 tiket
30. U-1 — Kerangka SPA — 4 tiket
31. U-2 — Pustaka Komponen — 5 tiket
32. U-3 · Layar Inbox per Peran — 2 tiket
33. U-4 · Layar Transaksi Klaim — 2 tiket
34. U-5 · Layar Laporan — 2 tiket
35. U-6 · Layar Master Data — 2 tiket

- Lampiran A — Inventaris 74 Harness
- Lampiran B — Indeks Seluruh Tiket

# 1. Papan Tiket

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

# 2. F-1 — Kerangka Aplikasi

*Modul Fondasi · folder `docs/ticketing/F-1-Kerangka-Aplikasi/`*

## Spesifikasi Modul

| | |
|---|---|
| **Modul** | `F-1` Kerangka Aplikasi |
| **Gelombang** | 1 — Fondasi |
| **Ukuran** | Kecil |
| **Bergantung pada** | — (tidak ada; ini modul paling awal) |
| **Kesiapan** | **SEBAGIAN** — lingkup jelas, satu keputusan menahan satu tiket |
| **Cakupan tiket** | **penuh** (`D-41` Opsi 1) |

### Apa yang dibangun

Rangka aplikasi Go yang menjadi tempat seluruh modul lain hidup: struktur folder dan aturan
lapisan, konfigurasi, logging, penanganan galat, health check, graceful shutdown, dan penyajian
SPA sebagai berkas statis.

**Tidak ada aturan bisnis di modul ini.** Bila sebuah tiket `F-1` mulai membicarakan klaim,
komite, atau spreading, tiket itu salah tempat.

### Kenapa ini lebih dulu

Setiap modul lain menulis kode ke dalam struktur yang ditetapkan `F-1`. Menetapkannya belakangan
berarti memindahkan seluruh kode yang telanjur ditulis — dan `D-09` menetapkan tim adalah
developer Pega yang dilatih ulang, sehingga **struktur yang preskriptif dan seragam** justru
bagian terpenting dari modul ini.

### Yang mengikat modul ini

| Sumber | Isi |
|---|---|
| `ADR-0001` | Modular monolith Go, satu binary, VM on-premise, tanpa orkestrator kontainer |
| `ADR-0002` | SPA React disajikan oleh binary Go — tidak ada runtime Node.js di produksi |
| `D-27` | 24/7 — stateless, dua instans, health check, graceful shutdown |
| `D-09` | Struktur folder, penamaan, dan pola baku ditulis preskriptif |
| `ADR-0025` | Nilai bisnis tidak boleh di-hardcode; rahasia keluar dari kode |

### Penghalang modul ini

| Jenis | Isi | Pemilik | Menahan |
|---|---|---|---|
| **Keputusan** | **720 dari 902 activity sistem lama tidak punya penanganan galat sama sekali.** Apakah kegagalan senyap itu direplikasi, atau sistem baru gagal keras? | Work Owner | `TKT-F1-004` |
| Artefak | instance Dynamic System Setting `ServiceFromTable` (1 setting, hanya rujukan) · `Data-Admin-DB-Name` tidak ada di export | Tim Pega | tidak menahan — daftar setting disusun dari nol |

### Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-F1-001](issues/01-kerangka-proyek-dan-aturan-lapisan.md) | Kerangka proyek Go dan penegakan aturan lapisan | `ready-for-human` | siap |
| [TKT-F1-002](issues/02-konfigurasi-tiga-lapis.md) | Konfigurasi tiga lapis dan gagal keras saat start | `ready-for-human` | siap |
| [TKT-F1-003](issues/03-logging-terstruktur.md) | Logging terstruktur dan ID permintaan | `ready-for-human` | siap |
| [TKT-F1-004](issues/04-penanganan-galat-terpusat.md) | Penanganan galat terpusat dan kontrak galat API | `needs-info` | terhalang keputusan |
| [TKT-F1-005](issues/05-health-check-shutdown-penyajian-spa.md) | Health check, graceful shutdown, dan penyajian SPA | `ready-for-human` | siap |

## Daftar Tiket

### TKT-F1-001 — Kerangka proyek Go dan penegakan aturan lapisan

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | F-1 · Gelombang: 1 · Bergantung pada: — |
| **Requirement** | FR-F1 |
| **Keputusan** | D-08, D-09 |
| **ADR** | 0001 |
| **Risiko** | R-11 |
| **Rule Pega yang digantikan** | — (tidak ada padanan; Pega menyediakan struktur ini lewat platform) |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi, cukup gerbang 1 + persetujuan Work Owner (`D-60`) |
| **Label** | `modul::F-1` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-1-Kerangka-Aplikasi/issues/01-kerangka-proyek-dan-aturan-lapisan.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Sebuah repository Go yang bisa di-*clone*, di-*build*, dan dijalankan, dengan **struktur folder
yang sudah final** dan aturan ketergantungan antar lapisan yang **ditegakkan perkakas, bukan
kesepakatan lisan**.

Nilai bisnisnya tidak langsung terlihat pengguna, tetapi ia yang menentukan biaya seluruh modul
berikutnya. `D-09` menetapkan tim adalah developer Pega yang sedang dilatih ulang; pada tim
seperti itu, struktur yang tidak seragam berubah menjadi 33 gaya penulisan berbeda dalam beberapa
bulan — dan itu persis kegagalan yang membuat sistem lama sulit dipelihara.

#### Ruang lingkup

- Struktur folder final beserta penjelasan **apa yang boleh dan tidak boleh berada di masing-masing
  folder**, mengikuti empat lapisan `ADR-0001`: `Domain` · `App` · `Adapter` · `Entrypoint`.
- Aturan ketergantungan antar lapisan **dipasang sebagai pemeriksaan otomatis** (`depguard` atau
  setara) yang **menggagalkan build**, bukan sekadar memperingatkan.
- Berkas `Makefile`/skrip build yang menghasilkan **satu binary** berisi SPA tersemat.
- Linter backend dengan konfigurasi yang sudah disepakati, dijalankan di pemeriksaan yang sama.
- Satu contoh modul "hello" yang menembus keempat lapisan, dipakai sebagai **acuan pola** bagi
  modul berikutnya.

#### Non-goal

- **Tidak** membangun modul bisnis apa pun.
- **Tidak** menyiapkan pipeline CI/CD — itu disiapkan tim GitLab (`D-33`).
- **Tidak** memilih pustaka frontend — itu `TKT-U2-005`.
- **Tidak** menyiapkan Docker atau orkestrator apa pun (`D-08`).

#### Acceptance criteria

- ☐ `go build ./...` menghasilkan **satu berkas binary** tanpa dependensi runtime eksternal.
- ☐ Struktur folder memuat **tepat empat lapisan** dengan nama yang disepakati, dan setiap
      lapisan punya berkas `doc.go` yang menyatakan apa yang boleh berada di dalamnya.
- ☐ Pemeriksaan ketergantungan **gagal** (exit code ≠ 0) bila lapisan `Domain` mengimpor paket
      HTTP, SQL, atau JSON — diuji dengan satu commit percobaan yang sengaja melanggar.
- ☐ Pemeriksaan ketergantungan **gagal** bila lapisan `Adapter` mengimpor lapisan `App`.
- ☐ Linter berjalan bersih (**0 temuan**) pada seluruh berkas yang ada.
- ☐ Modul contoh dapat dipanggil lewat satu endpoint HTTP dan mengembalikan respons dengan
      bentuk yang sama dengan kontrak galat `TKT-F1-004` bila gagal.
- ☐ Dokumen `CONTRIBUTING.md` atau setara menyebutkan **aturan penamaan berkas, paket, dan
      fungsi**, cukup preskriptif untuk diikuti tanpa bertanya (`D-09`).

#### Dependency / Blocked by

Tidak ada. Ini tiket paling awal di seluruh papan.

**Yang bergantung padanya:** seluruh tiket `F-2`…`F-5`, `S-5`, dan seluruh modul bisnis.

#### Constraint keamanan, data, operasional

- Binary **tidak boleh** memuat nilai rahasia apa pun. Rahasia hanya dari luar proses
  (`ADR-0025`) — dan tujuan penyimpanannya masih `OPEN` (`D-40`), sehingga tiket ini hanya
  menyiapkan **tempat membacanya**, bukan memutuskan sumbernya.
- Aplikasi wajib **stateless** (`D-27`): tidak ada state yang hanya hidup di memori satu instans.

#### Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** menghapus repository; tidak ada dampak ke sistem
yang berjalan.

#### Rencana verifikasi

> **Rencana — belum dijalankan.** Belum ada kode implementasi di repository ini.

```bash
go build ./...                      # harus menghasilkan satu binary
go vet ./...                        # harus bersih
golangci-lint run                   # harus 0 temuan
go test ./...                       # contoh modul harus lulus
### uji negatif: tambahkan import "net/http" di satu berkas Domain, lalu:
golangci-lint run                   # HARUS gagal — bila lolos, aturan lapisan tidak tegak
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Satu binary, VM on-premise, tanpa orkestrator | `ADR-0001` · `D-08` |
| Empat lapisan dan aturan ketergantungannya | `docs/Steering/04-FUTURE-ARCHITECTURE.md` §2 |
| Struktur preskriptif karena tim eks-Pega | `D-09` |
| SPA disajikan binary Go | `ADR-0002` · `D-23` |
| Stateless dan dua instans | `D-27` |

#### Comments

### TKT-F1-002 — Konfigurasi tiga lapis dan gagal keras saat start

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | F-1 · Gelombang: 1 · Bergantung pada: TKT-F1-001 |
| **Requirement** | FR-F1 |
| **Keputusan** | D-15, D-27, D-40 |
| **ADR** | 0025 |
| **Risiko** | R-17 |
| **Rule Pega yang digantikan** | — Pega memakai Dynamic System Setting; **nol DSS ditemukan di export**. Konfigurasi dinamis sistem lama berupa tabel Oracle yang dikunci per IP aplikasi |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`) |
| **Label** | `modul::F-1` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-1-Kerangka-Aplikasi/issues/02-konfigurasi-tiga-lapis.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu tempat membaca konfigurasi, dengan urutan prioritas yang jelas, dan **aplikasi yang menolak
start bila konfigurasi wajib tidak ada**.

Nilai bisnisnya: sistem lama mengubah perilaku bisnis berdasarkan **nama server** — salah satunya
mengubah ambang komite dari Rp 50.000.000 menjadi 3.500 (`ADR-0025`). Perilaku seperti itu tidak
dapat diuji dan berubah diam-diam saat server dipindahkan. Tiket ini menyiapkan penggantinya.

#### Ruang lingkup

- Tiga lapis konfigurasi, dari umum ke khusus: **nilai baku di kode → berkas YAML per lingkungan →
  variabel lingkungan**, yang paling khusus menang.
- **Validasi saat start**: seluruh konfigurasi wajib diperiksa keberadaannya dan bentuknya; bila
  ada yang kurang, aplikasi **berhenti dengan pesan yang menyebut nama setting-nya**.
- Pemisahan tegas antara **konfigurasi teknis** (alamat database, ukuran pool, batas waktu, port,
  tingkat log, masa berlaku token) dan **master data** — master data **bukan** urusan tiket ini,
  ia milik `F-4`.
- Antarmuka pembaca rahasia yang **implementasinya dapat diganti** tanpa menyentuh pemanggil.

#### Non-goal

- **Tidak** memutuskan ke mana rahasia dipindahkan — itu `D-40`, masih `OPEN`, milik Tim
  Infra/Security. Tiket ini hanya menyiapkan antarmukanya.
- **Tidak** membangun master data (`F-4`).
- **Tidak** memuat mekanisme konfigurasi yang berubah tanpa restart — konfigurasi teknis memang
  butuh restart; yang berubah tanpa restart adalah master data.

#### Acceptance criteria

- ☐ Nilai yang sama didefinisikan di ketiga lapis menghasilkan **nilai dari variabel
      lingkungan** — dibuktikan uji otomatis.
- ☐ Menjalankan aplikasi tanpa satu setting wajib menghasilkan **exit code ≠ 0** dan pesan yang
      **menyebut nama setting yang kurang** — bukan pesan umum.
- ☐ Tidak ada satu pun nilai rahasia di dalam berkas YANG MASUK repository — diuji pemindaian
      pola pada pemeriksaan build.
- ☐ **Nol perilaku yang bergantung pada nama host.** Diuji: menjalankan aplikasi dengan
      `hostname` berbeda menghasilkan perilaku identik pada seluruh uji yang ada.
- ☐ Daftar seluruh setting beserta artinya tersedia di satu berkas dokumentasi, dan
      **jumlahnya sama** dengan yang dibaca kode — diuji otomatis, bukan diperiksa manual.

#### Dependency / Blocked by

- Bergantung pada `TKT-F1-001` (struktur folder dan aturan lapisan).
- **Tidak terhalang** `D-40`: yang `OPEN` adalah tujuan penyimpanan rahasia, bukan cara membacanya.
  Bagian yang menunggu `D-40` adalah **adapter**-nya, dan itu tiket terpisah di `F-4`/deployment.

#### Constraint keamanan, data, operasional

- **Rahasia hanya dari luar proses.** Tidak pernah dari berkas YAML yang masuk repository
  (`ADR-0025`).
- Export rule memuat **3 password SMTP di 31 lokasi** dan 1 pasang kredensial OAuth, seluruhnya
  plaintext (`R-17`). Tiket ini **tidak** memindahkannya — ia menyiapkan tempatnya. Pemindahannya
  menunggu `D-40`.
- Konfigurasi **tidak boleh** dikunci per IP aplikasi seperti sistem lama — itu bertabrakan dengan
  dua instans (`D-27`).

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** mengembalikan berkas konfigurasi ke versi sebelumnya dan
menjalankan ulang; tidak ada data yang berubah.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/config/...                 # urutan prioritas tiga lapis
APP_DB_DSN= go run ./cmd/app                  # HARUS exit ≠ 0 dan menyebut nama setting
hostname-sim A go test ./...                  # perilaku identik lintas hostname
grep -rIn -E "(password|secret|api[_-]?key)\s*[:=]" config/   # HARUS 0 baris
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Tiga lapis konfigurasi dan gagal keras saat start | `docs/Steering/12-CROSSCUTTING.md` §3.1 |
| Nol Dynamic System Setting di sistem lama; konfigurasi dikunci per IP | `docs/Steering/12-CROSSCUTTING.md` §3.5 · `T-8` |
| Hostname menentukan ambang komite | `Activity/GetKomiteApproval-Act.xml:335` · `ADR-0025` |
| 3 password SMTP di 31 lokasi + 1 pasang OAuth | `R-17` · `docs/verifikasi-bukti-adr.md` §7.6 |
| Pemisahan konfigurasi teknis vs master data | `docs/Steering/12-CROSSCUTTING.md` §3.2–3.3 |

#### Comments

### TKT-F1-003 — Logging terstruktur dan ID permintaan

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | F-1 · Gelombang: 1 · Bergantung pada: TKT-F1-001, TKT-F1-002 |
| **Requirement** | FR-F1 |
| **Keputusan** | D-69 |
| **ADR** | 0026, 0029 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | — Pega menyediakan log platform; tidak ada rule aplikasi yang mengaturnya |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`) |
| **Label** | `modul::F-1` `tipe::fondasi` `status::ready-for-human` `prioritas::sedang` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-1-Kerangka-Aplikasi/issues/03-logging-terstruktur.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Log yang **dapat ditelusuri per permintaan** dan **dapat dibaca mesin**, sehingga satu keluhan
pengguna dapat dilacak sampai ke kueri yang gagal tanpa menebak.

Nilai bisnisnya paling terasa selama masa paralel: ketika Pega dan Go melayani klaim yang sama
(`ADR-0003`), pertanyaan pertama saat terjadi selisih selalu *"permintaan mana yang menghasilkan
ini"*. Tanpa ID permintaan, pertanyaan itu tidak terjawab.

#### Ruang lingkup

- Logger terstruktur (JSON) dengan tingkat log dari konfigurasi.
- **ID permintaan** dibuat di titik masuk HTTP, dibawa lewat `context`, dan muncul di **setiap**
  baris log yang lahir dari permintaan itu.
- Baris log baku untuk: permintaan masuk, permintaan selesai (beserta lamanya), galat, dan
  pemanggilan sistem eksternal (tujuan, lama, hasil).
- **Aturan penyamaran**: daftar field yang nilainya tidak boleh pernah masuk log.

#### Non-goal

- **Tidak** membangun jejak audit bisnis — itu `S-5`, dan keduanya **berbeda tujuan**: log untuk
  menelusuri masalah teknis, jejak audit untuk mempertanggungjawabkan perubahan bernilai bisnis.
- **Tidak** memasang agregator log atau dashboard — itu urusan infrastruktur.
- **Tidak** memutuskan retensi log.

#### Acceptance criteria

- ☐ Setiap baris log berformat JSON dengan field wajib: waktu (UTC), tingkat, pesan,
      **`request_id`**, dan nama modul.
- ☐ Satu permintaan HTTP yang melewati tiga lapisan menghasilkan baris log dengan
      **`request_id` yang sama persis** di ketiganya — diuji otomatis.
- ☐ Tingkat log dapat diubah lewat konfigurasi tanpa mengubah kode — diuji dengan dua nilai.
- ☐ **Nol nilai sensitif di log**, diuji dengan permintaan yang memuat nomor polis, NPWP,
      nomor rekening, dan alamat email: keempatnya **tidak muncul** di keluaran log.
- ☐ Alamat email yang terpaksa dicatat muncul **tersamar** (bagian sebelum `@` diganti),
      konsisten dengan `D-69`.
- ☐ Pemanggilan sistem eksternal tercatat dengan tujuan, lama (ms), dan hasil — diuji dengan
      satu adapter tiruan yang sengaja gagal.

#### Dependency / Blocked by

Bergantung pada `TKT-F1-001` dan `TKT-F1-002` (tingkat log dari konfigurasi).

#### Constraint keamanan, data, operasional

- **Data nasabah tidak pernah masuk log**: nomor polis, nama tertanggung, NPWP, nomor rekening,
  dan data medis. Ini berlaku **juga di staging**, karena staging memuat data produksi apa adanya
  (`ADR-0029`).
- Log **tidak boleh** menjadi tempat menyimpan bukti perubahan bernilai bisnis — bila sebuah
  perubahan perlu dipertanggungjawabkan, tempatnya jejak audit `S-5`, bukan log.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** menurunkan tingkat log atau menonaktifkan field baru lewat
konfigurasi; tidak ada data yang berubah.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/platform/log/...
### uji kebocoran data: kirim permintaan berisi nilai penanda, lalu
go test ./internal/platform/log/... -run TestTidakAdaDataSensitifDiLog
### periksa manual bentuk log satu permintaan penuh:
go run ./cmd/app & curl -s localhost:8080/api/contoh ; # satu request_id di semua baris
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Logging terstruktur, ID permintaan, baris log baku | `docs/Steering/12-CROSSCUTTING.md` §2 |
| Aturan penyamaran email dan data nasabah | `D-69` · `ADR-0029` |
| Log berbeda tujuan dari jejak audit | `ADR-0026` · `docs/Steering/09-DATABASE-STRATEGY.md` §8 |
| Staging memuat data produksi apa adanya | `D-64` · `ADR-0029` |

#### Comments

### TKT-F1-004 — Penanganan galat terpusat dan kontrak galat API

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan** — lihat "Yang kurang dan siapa yang bisa melengkapinya" |
| **Modul** | F-1 · Gelombang: 1 · Bergantung pada: TKT-F1-001, TKT-F1-003 |
| **Requirement** | FR-F1 |
| **Keputusan** | D-49 (butir 5, 10) |
| **ADR** | 0007, 0017 |
| **Risiko** | R-19 |
| **Rule Pega yang digantikan** | pola galat tersebar — **720 dari 902 activity tanpa penanganan galat sama sekali**; `Database/GETCURRENCYSTANDARD.fnc:22` (`RETURN 1`); `Database/GETSELISIHJAM.fnc:22` (`RETURN 0`); kontrak `ErrMsg` berbasis string pada 12 procedure |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`) |
| **Label** | `modul::F-1` `tipe::fondasi` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-1-Kerangka-Aplikasi/issues/04-penanganan-galat-terpusat.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu cara menangani galat di seluruh aplikasi, dan satu bentuk respons galat yang dipakai setiap
endpoint — sehingga frontend tidak perlu menebak bentuk galat per layar, dan kegagalan tidak lagi
hilang tanpa jejak.

Nilai bisnisnya sangat konkret. Di sistem lama, kegagalan **senyap** justru terjadi pada jalur
uang: fungsi kurs mengembalikan `1` ketika kurs tidak ditemukan, sehingga klaim bernilai besar
menyusut menjadi kecil lalu **lolos tanpa komite**. Tidak ada galat, tidak ada catatan, dan
angkanya tampak wajar.

#### Ruang lingkup

- Jenis galat domain yang dibedakan dari galat teknis, dan **pemetaannya ke kode status HTTP**.
- Satu bentuk respons galat untuk seluruh endpoint: kode galat, pesan untuk pengguna, dan
  pengenal permintaan (`TKT-F1-003`).
- Middleware yang menangkap panic dan mengubahnya menjadi respons galat, **dengan baris log
  lengkap**, tanpa membocorkan stack trace ke pengguna.
- **Larangan nilai bawaan yang menyamar sebagai hasil** — pola `RETURN 1` dan `RETURN 0` sistem
  lama tidak boleh punya padanan di sistem baru.

#### Non-goal

- **Tidak** menentukan pesan galat per aturan bisnis — itu milik modul bisnis masing-masing.
- **Tidak** menerjemahkan pesan galat ke bahasa lain.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan tiket ini |
|---|---|---|
| **720 dari 902 activity sistem lama tidak punya penanganan galat sama sekali.** Apakah kegagalan senyap itu **direplikasi** demi kesetaraan `P-5`, atau sistem baru **gagal keras**? | **Work Owner** | Menentukan perilaku baku seluruh aplikasi. Bila gagal keras dipilih, uji kesetaraan akan menunjukkan selisih pada setiap jalur yang dulu diam — dan selisih itu harus dinyatakan di muka sebagai perbaikan terencana, bukan ditemukan sebagai kejutan |

**Catatan yang mempersempit pertanyaannya.** Dua kasus sudah diputuskan dan **tidak perlu
ditanyakan lagi**: kurs tidak ditemukan → klaim **ditolak** (`D-48`, `D-49` butir 5), dan
`GETSELISIHJAM` gagal → **tidak** mengembalikan `0` yang tak terbedakan dari nol (`D-49` butir 10).
Yang ditanyakan adalah **perilaku baku untuk 720 activity sisanya**.

#### Acceptance criteria

> Ditulis sebagai rancangan; **belum boleh dijadikan dasar implementasi** sampai pertanyaan di
> atas terjawab, karena butir pertama berubah bentuk tergantung jawabannya.

- ☐ Seluruh endpoint mengembalikan galat dengan **bentuk yang sama**, diuji pada minimal 3
      endpoint berbeda.
- ☐ Respons galat memuat `request_id` yang sama dengan yang ada di log.
- ☐ Respons galat **tidak pernah** memuat stack trace, nama tabel, atau potongan SQL — diuji
      dengan galat database yang sengaja dipicu.
- ☐ Panic pada handler menghasilkan respons `500` dengan bentuk baku **dan** satu baris log
      bertingkat `error` — aplikasi **tetap hidup**.
- ☐ Galat domain "tidak ditemukan", "tidak berwenang", dan "masukan tidak sah" masing-masing
      memetakan ke kode status yang berbeda dan terdokumentasi.
- ☐ **Nol fungsi yang mengembalikan nilai bawaan saat gagal** — diuji lewat pemeriksaan pola
      pada kode: tidak ada `return 0, nil` atau `return 1, nil` pada jalur galat.

#### Dependency / Blocked by

- Bergantung pada `TKT-F1-001`, `TKT-F1-003`.
- **Terhalang keputusan Work Owner** di atas.

#### Constraint keamanan, data, operasional

- Pesan galat yang sampai ke pengguna **tidak boleh** membocorkan struktur internal: nama tabel,
  nama kolom, potongan SQL, atau jalur berkas.
- Galat pada jalur uang **tidak boleh** senyap. Bila nilai tidak dapat dihitung, permintaan gagal
  — bukan dilanjutkan dengan nilai bawaan.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** mengembalikan middleware ke versi sebelumnya; tidak ada data
yang berubah.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/platform/errors/...
go test ./internal/adapter/http/... -run TestKontrakGalat
grep -rIn -E "return (0|1|\"\"), nil" internal/   # HARUS 0 baris pada jalur galat
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 720 dari 902 activity tanpa penanganan galat | `docs/verifikasi-bukti-adr.md` §10.9 #3 |
| Kurs tidak ditemukan → `RETURN 1` | `Database/GETCURRENCYSTANDARD.fnc:22` · `D-49` butir 5 |
| `GETSELISIHJAM` gagal → `RETURN 0` | `Database/GETSELISIHJAM.fnc:22` · `D-49` butir 10 |
| Kontrak galat `ErrMsg` berbasis string tidak dibawa | `D-68` · `ADR-0007` |
| Bentuk respons galat baku | `docs/Steering/12-CROSSCUTTING.md` §1 · `docs/Steering/10-API-STRATEGY.md` §3 |

#### Comments

### TKT-F1-005 — Health check, graceful shutdown, dan penyajian SPA

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | F-1 · Gelombang: 1 · Bergantung pada: TKT-F1-001, TKT-F1-002 |
| **Requirement** | FR-F1 |
| **Keputusan** | D-27, D-08, D-23 |
| **ADR** | 0001, 0002 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | — Pega menyediakan ini di tingkat platform; tidak ada rule aplikasi |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`) |
| **Label** | `modul::F-1` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-1-Kerangka-Aplikasi/issues/05-health-check-shutdown-penyajian-spa.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Aplikasi yang dapat **di-update tanpa memutus pengguna yang sedang bekerja**, dan yang dapat
diketahui sehat-tidaknya oleh load balancer tanpa menebak.

Nilai bisnisnya diikat `D-27`: aplikasi harus tersedia **24/7 termasuk saat deployment**. Tanpa
health check dan graceful shutdown, setiap rilis memutus permintaan yang sedang berjalan — dan
pada jalur klaim, permintaan yang terputus di tengah berarti pekerjaan petugas hilang.

#### Ruang lingkup

- Endpoint **liveness** (proses hidup) dan **readiness** (siap menerima lalu lintas) yang
  **terpisah** — readiness memeriksa koneksi database, liveness tidak.
- **Graceful shutdown**: pada sinyal berhenti, aplikasi berhenti menerima permintaan baru,
  menyelesaikan yang sedang berjalan sampai batas waktu, lalu keluar.
- **Penyajian SPA** sebagai berkas statis yang tersemat di binary, dengan *fallback* rute SPA:
  permintaan yang bukan `/api/...` dan bukan berkas nyata dikembalikan ke `index.html`.
- Header keamanan dasar pada penyajian statis.

#### Non-goal

- **Tidak** membangun SPA-nya — itu `U-1`.
- **Tidak** menyiapkan load balancer atau konfigurasi VM — itu urusan infrastruktur.
- **Tidak** membangun metrik atau dashboard.

#### Acceptance criteria

- ☐ `GET /healthz` (liveness) mengembalikan `200` **walau database mati** — diuji dengan
      database dimatikan.
- ☐ `GET /readyz` (readiness) mengembalikan **`503` saat database mati** dan `200` saat sehat —
      diuji dengan kedua keadaan.
- ☐ Saat menerima `SIGTERM`, aplikasi **menyelesaikan permintaan yang sedang berjalan** lalu
      keluar dengan kode `0`. Diuji: permintaan lambat (≥ 2 detik) yang sedang berjalan **tetap
      menerima respons lengkap**, bukan koneksi terputus.
- ☐ Batas waktu graceful shutdown dapat dikonfigurasi, dan **habisnya batas waktu** membuat
      aplikasi tetap keluar — tidak menggantung.
- ☐ `GET /` mengembalikan `index.html`; `GET /klaim/123` (rute SPA yang tidak ada di disk) juga
      mengembalikan `index.html`; `GET /api/tidak-ada` mengembalikan **galat JSON**, bukan HTML.
- ☐ Binary berjalan **tanpa berkas pendamping apa pun** — dibuktikan dengan menjalankannya dari
      direktori kosong.
- ☐ Dua instans dapat berjalan bersamaan terhadap konfigurasi yang sama tanpa saling
      mengganggu — diuji dengan menjalankan dua proses pada port berbeda.

#### Dependency / Blocked by

Bergantung pada `TKT-F1-001` dan `TKT-F1-002`.

**Catatan:** SPA yang disajikan pada tahap ini boleh berupa halaman kosong berisi penanda versi.
Isi sebenarnya datang dari `U-1`.

#### Constraint keamanan, data, operasional

- Endpoint health **tidak boleh** membocorkan detail internal — tanpa versi pustaka, tanpa nama
  host database, tanpa jumlah koneksi.
- Aplikasi wajib **stateless** (`D-27`): sesi tidak boleh hidup di memori satu instans, karena
  dua instans dilayani load balancer bergantian.
- **Tidak ada runtime Node.js di produksi** (`ADR-0002`) — SPA disajikan binary Go.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** menjalankan binary versi sebelumnya; karena rolling
deployment, versi lama dan baru sempat hidup bersamaan — itulah alasan `P-4` mewajibkan migrasi
skema selalu backward-compatible.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/entrypoint/...
### uji shutdown: jalankan, kirim permintaan lambat, kirim SIGTERM
go run ./cmd/app & PID=$!; curl -s localhost:8080/api/lambat & sleep 1; kill -TERM $PID; wait
### uji readiness saat database mati
docker stop oracle-dev 2>/dev/null; curl -so /dev/null -w "%{http_code}" localhost:8080/readyz  # harus 503
### uji binary mandiri
mkdir -p /tmp/kosong && cd /tmp/kosong && /path/app --version
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 24/7 termasuk saat deployment; dua instans; stateless; health check; graceful shutdown | `D-27` · `docs/Steering/13-DEPLOYMENT.md` §2 |
| SPA disajikan binary Go, tanpa runtime Node.js | `ADR-0002` · `D-23` |
| Satu binary, VM on-premise | `ADR-0001` · `D-08` |
| Migrasi skema backward-compatible karena rolling deployment | `P-4` · `docs/Steering/07-MIGRATION-STRATEGY.md` |

#### Comments

# 3. F-2 — Akses Data

*Modul Fondasi · folder `docs/ticketing/F-2-Akses-Data-dan-Nomor-Klaim/`*

## Spesifikasi Modul

| | |
|---|---|
| **Modul** | `F-2` Akses Data |
| **Gelombang** | 1 — Fondasi |
| **Ukuran** | Sedang |
| **Bergantung pada** | `F-1` |
| **Kesiapan** | **SEBAGIAN** — lingkup jelas, dua penghalang menahan dua tiket |
| **Cakupan tiket** | **penuh** (`D-41` Opsi 1) |

### Apa yang dibangun

Seluruh cara aplikasi menyentuh database: koneksi dan pool, seam Repository, disiplin SQL
portabel, kepemilikan transaksi, kerangka migrasi skema, pola soft delete, dan generator nomor
klaim.

**Tidak ada kueri bisnis di modul ini.** Kueri klaim, komite, dan settlement milik modul bisnisnya
masing-masing; `F-2` menyediakan aturan dan perkakasnya.

### Kenapa ini berbahaya bila salah

Selama masa paralel, Pega dan Go menulis ke **database yang sama** (`ADR-0004`). Satu tabel yang
ditulis dua sistem menghasilkan kerusakan data yang hampir mustahil dilacak — dan tabel header
klaim `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` dibaca **116 rule Pega**. `F-2` adalah tempat aturan
pengamannya dipasang.

### Yang mengikat modul ini

| Sumber | Isi |
|---|---|
| `ADR-0004` | Database bersama · penulis tunggal per tabel (`P-1`) · skema backward-compatible (`P-4`) |
| `ADR-0005` | Satu set SQL portabel Oracle + PostgreSQL 17+, tiga pengecualian terkelola |
| `ADR-0007` | Kepemilikan transaksi pindah ke Go; stored procedure tidak dipanggil |
| `ADR-0009` | Nomor klaim `PNCN.YY.xxxx` — satu-satunya sakelar dialek |
| `ADR-0012` | Soft delete menyeluruh |
| `D-63` | Perubahan skema: permintaan tertulis → persetujuan Work Owner → DBA → uji Pega+Go bersamaan |

### Penghalang modul ini

| Jenis | Isi | Pemilik | Menahan |
|---|---|---|---|
| **Artefak** | **DDL seluruh tabel** (`R-08`) — tipe, panjang, index, constraint tidak diketahui | DBA | `TKT-F2-004` sebagian |
| **Keputusan** | **538 pemakaian `{ASIS:}`** — himpunan kolom filter dan sort yang sah belum ditetapkan | Work Owner + Lead Engineer | `TKT-F2-007` |
| Artefak | **64 Connect SQL** dan 12 dependensi procedure | DBA | tidak menahan `F-2`; menahan modul bisnis |
| Catatan | `pyParamArray` kosong → **1.646 step RDB tidak terbaca**; inventaris SQL yang ada adalah **batas bawah** | — | memengaruhi estimasi, bukan lingkup |

### Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-F2-001](issues/01-koneksi-pool-dan-seam-repository.md) | Koneksi, connection pool, dan seam Repository | `ready-for-human` | siap |
| [TKT-F2-002](issues/02-disiplin-sql-portabel.md) | Disiplin SQL portabel dan pemeriksaan pola terlarang | `ready-for-human` | siap |
| [TKT-F2-003](issues/03-kepemilikan-transaksi.md) | Kepemilikan transaksi di lapisan aplikasi | `ready-for-human` | siap |
| [TKT-F2-004](issues/04-kerangka-migrasi-skema.md) | Kerangka migrasi skema backward-compatible dan rollback | `needs-info` | terhalang `R-08` |
| [TKT-F2-005](issues/05-pola-soft-delete.md) | Soft delete sebagai pola akses data | `ready-for-human` | siap |
| [TKT-F2-006](issues/06-generator-nomor-klaim.md) | Generator nomor klaim `PNCN.YY.xxxx` | `needs-info` | terhalang keputusan |
| [TKT-F2-007](issues/07-daftar-putih-filter-dan-sort.md) | Daftar putih kolom filter dan sort pengganti `{ASIS:}` | `needs-info` | terhalang keputusan |

## Daftar Tiket

### TKT-F2-001 — Koneksi, connection pool, dan seam Repository

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | F-2 · Gelombang: 1 · Bergantung pada: TKT-F1-001, TKT-F1-002 |
| **Requirement** | FR-F2 |
| **Keputusan** | D-01, D-10, D-21 |
| **ADR** | 0004, 0005 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | lapisan Connect-SQL Pega — **652 rule SQL**; koneksi diatur `Data-Admin-DB-Name` yang **tidak ada di export** |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`) |
| **Label** | `modul::F-2` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-2-Akses-Data-dan-Nomor-Klaim/issues/01-koneksi-pool-dan-seam-repository.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu pintu masuk ke database, dengan pool yang terukur, dan **antarmuka Repository yang
dideklarasikan lapisan Domain** — sehingga aturan bisnis dapat diuji tanpa database sama sekali.

Nilai bisnisnya: `D-10` menetapkan profil sistem ini **beban data besar, konkurensi rendah** —
ribuan klaim per bulan di atas data puluhan juta baris, tetapi hanya 200–300 pengguna harian.
Pool yang dirancang untuk profil itu berbeda dari pool yang dirancang untuk lalu lintas tinggi,
dan salah merancangnya menghabiskan koneksi database yang **dibagi dengan Pega** selama masa
paralel.

#### Ruang lingkup

- Pembukaan koneksi ke Oracle 19c dengan parameter dari konfigurasi (`TKT-F1-002`).
- Connection pool dengan batas yang **dapat dikonfigurasi** dan nilai baku yang sesuai profil
  `D-10`.
- **Antarmuka Repository dideklarasikan di lapisan Domain**, implementasinya di lapisan Adapter —
  sehingga arah ketergantungan tetap sesuai `ADR-0001`.
- Satu implementasi Repository contoh dan satu implementasi tiruan untuk pengujian.
- **Pool terpisah untuk laporan**, agar satu laporan berat tidak menghabiskan koneksi transaksi.

#### Non-goal

- **Tidak** menulis kueri bisnis apa pun.
- **Tidak** membangun PostgreSQL sekarang — target akhir memang PostgreSQL (`ADR-0005`), tetapi
  runtime sementara Oracle.
- **Tidak** memanggil stored procedure (`ADR-0007`).

#### Acceptance criteria

- ☐ Aplikasi terhubung ke Oracle memakai parameter dari konfigurasi, dan **gagal start dengan
      pesan jelas** bila parameternya salah.
- ☐ Batas pool (maksimum koneksi, idle, umur koneksi) **dapat dikonfigurasi**, dan nilai
      bakunya terdokumentasi beserta alasannya terhadap profil `D-10`.
- ☐ Antarmuka Repository berada di paket lapisan **Domain**; pemeriksaan lapisan `TKT-F1-001`
      **gagal** bila implementasinya ikut masuk ke sana — diuji dengan commit percobaan.
- ☐ Aturan bisnis contoh dapat diuji **tanpa database** memakai Repository tiruan — dibuktikan
      dengan `go test` yang lulus saat database dimatikan.
- ☐ Pool laporan terpisah dari pool transaksi, dan **menghabiskan pool laporan tidak
      memengaruhi** permintaan transaksi — diuji dengan menahan seluruh koneksi laporan.
- ☐ Setiap kueri yang dijalankan tercatat di log dengan lamanya (`TKT-F1-003`).

#### Dependency / Blocked by

Bergantung pada `TKT-F1-001` dan `TKT-F1-002`.

#### Constraint keamanan, data, operasional

- **Kredensial database hanya dari konfigurasi luar proses** (`ADR-0025`).
- Selama masa paralel, database **dibagi dengan Pega** (`ADR-0004`). Pool yang terlalu besar
  memakan koneksi yang dibutuhkan Pega untuk melayani produksi — batasnya harus dibicarakan
  dengan DBA sebelum dinaikkan.
- **Parameter binding tanpa perkecualian** — tidak ada perangkaian nilai ke dalam teks SQL.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** mengembalikan konfigurasi pool; tidak ada data yang berubah.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/sqlstore/...
go test ./internal/domain/... # HARUS lulus tanpa database hidup
golangci-lint run             # aturan lapisan: Domain tidak mengimpor driver SQL
### uji isolasi pool laporan:
go test ./internal/adapter/sqlstore/... -run TestPoolLaporanTidakMengganggu
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Beban data besar, konkurensi rendah; 200–300 pengguna harian | `D-10` · `docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md` |
| Database dibagi dengan Pega selama masa paralel | `D-21` · `ADR-0004` |
| Seam Repository dideklarasikan Domain | `docs/Steering/04-FUTURE-ARCHITECTURE.md` §3.1 |
| Pool terpisah untuk laporan | `docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md` butir 7 |
| 652 rule SQL sebagai lapisan yang digantikan | `docs/Steering/03-CURRENT-ARCHITECTURE.md` |

#### Comments

### TKT-F2-002 — Disiplin SQL portabel dan pemeriksaan pola terlarang

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | F-2 · Gelombang: 1 · Bergantung pada: TKT-F2-001 |
| **Requirement** | FR-F2 |
| **Keputusan** | D-01, D-20, D-24 |
| **ADR** | 0005 |
| **Risiko** | R-19 |
| **Rule Pega yang digantikan** | seluruh 652 rule Connect-SQL — di antaranya **411 pemakaian `TO_CHAR`**, **68 pemakaian `ROWNUM`**, **538 pemakaian `{ASIS:}`** |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`) |
| **Label** | `modul::F-2` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-2-Akses-Data-dan-Nomor-Klaim/issues/02-disiplin-sql-portabel.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Aturan penulisan SQL yang **ditegakkan perkakas**, sehingga satu set kueri berjalan di Oracle
hari ini dan PostgreSQL kelak tanpa ditulis dua kali.

Nilai bisnisnya bukan hanya portabilitas. Mengeluarkan `TO_CHAR` dari SQL sekaligus **menutup
cacat yang sudah berjalan**: sistem lama mengembalikan tanggal sebagai string `'dd/mm/yyyy'`,
sehingga pengurutan tanggal menjadi pengurutan teks — `01/12/2024` dianggap lebih kecil daripada
`02/01/2020` — dan penyaringan rentang tanggal tidak dapat memakai index.

#### Ruang lingkup

- Daftar **padanan sintaks yang mengikat**, ditulis sebagai dokumen yang dapat dirujuk saat review.
- **Pemeriksaan otomatis pada build** yang menggagalkan commit bila menemukan pola terlarang:
  `TO_CHAR` untuk memformat tanggal/angka, `ROWNUM`, `NVL`, `SYSDATE` di luar generator nomor
  klaim, dan perangkaian string ke dalam SQL.
- **Pemformatan tanggal dan angka dipindahkan ke Go** — SQL mengembalikan tipe tanggal, bukan
  string.
- **Paginasi diseragamkan** ke `OFFSET … FETCH NEXT … ROWS ONLY`; untuk inbox dan pencarian
  dipakai **keyset pagination**.
- Satu berkas contoh kueri yang menunjukkan pola yang benar untuk masing-masing kasus.

#### Non-goal

- **Tidak** menulis ulang 652 kueri sistem lama — itu tersebar di modul bisnis masing-masing.
- **Tidak** menyiapkan lingkungan PostgreSQL — itu tugas infrastruktur, dan tanpa lingkungan itu
  klaim "portabel" **tidak terverifikasi**.
- **Tidak** menetapkan daftar putih kolom filter — itu `TKT-F2-007`.

#### Acceptance criteria

- ☐ Dokumen padanan sintaks memuat minimal: `NVL`→`COALESCE`, `ROWNUM`→`FETCH NEXT`,
      `SYSDATE`→parameter waktu dari seam Clock, `TO_CHAR`→pemformatan di Go, penggabungan
      string, dan tipe boolean.
- ☐ Pemeriksaan pola terlarang **menggagalkan build** pada berkas percobaan yang memuat
      `ROWNUM` — diuji dengan commit percobaan.
- ☐ Pemeriksaan **menggagalkan build** pada perangkaian string ke dalam SQL (`"… WHERE x = " +
      nilai`).
- ☐ **Nol `TO_CHAR` untuk pemformatan** di seluruh kueri yang ada — dihitung dan dilaporkan
      angkanya.
- ☐ Kueri contoh mengembalikan kolom tanggal sebagai **tipe tanggal**, bukan string —
      dibuktikan uji yang mengurutkan tiga tanggal lintas tahun dan memeriksa urutannya.
- ☐ Paginasi keyset pada kueri contoh menghasilkan **hasil yang sama** dengan `OFFSET` untuk
      1.000 baris pertama, dan **rencana eksekusinya tidak memindai seluruh tabel** — dilampirkan
      keluaran `EXPLAIN PLAN`.

#### Dependency / Blocked by

Bergantung pada `TKT-F2-001`.

**Catatan kejujuran:** portabilitas hanya dapat **dibuktikan** bila kedua mesin benar-benar diuji.
Selama lingkungan PostgreSQL 17+ belum ada, tiket ini menghasilkan **disiplin**, bukan bukti.
Itu perbedaan yang harus disebut saat tiket ini ditutup.

#### Constraint keamanan, data, operasional

- **Parameter binding tanpa perkecualian.** Ini sekaligus menutup celah warisan: sistem lama
  menyisipkan nilai pengguna langsung ke teks SQL lewat `{ASIS:}` pada **538 tempat**.
- Nama kolom untuk sort dan filter **hanya dari daftar yang diizinkan** — daftarnya ditetapkan
  `TKT-F2-007`, dan sampai itu ada, kueri baru **tidak boleh** menerima nama kolom dari pengguna.
- `SYSDATE` **dilarang** di luar generator nomor klaim; waktu berasal dari seam Clock (`F-5`).

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** menonaktifkan pemeriksaan pola; kueri yang sudah ditulis
tetap sah.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
make lint-sql                     # pemeriksaan pola terlarang
grep -rIn -E "TO_CHAR\(" internal/ | grep -v nomor_klaim   # HARUS 0 baris
grep -rIn "ROWNUM" internal/                                # HARUS 0 baris
go test ./internal/adapter/sqlstore/... -run TestPaginasiKeyset
### uji negatif: tambahkan ROWNUM di satu kueri, lalu
make lint-sql                     # HARUS gagal
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 411 `TO_CHAR`, 68 `ROWNUM`, pola paginasi sudah dipakai 35 rule | `D-20` · `ADR-0005` |
| Tanggal dikembalikan sebagai string `'dd/mm/yyyy'` → pengurutan salah | `docs/Steering/09-DATABASE-STRATEGY.md` §3.2 |
| 538 pemakaian `{ASIS:}` | `docs/verifikasi-bukti-adr.md` §15 baris `F-2` |
| PostgreSQL 17+ adalah persyaratan mengikat | `D-24` |
| `OFFSET` nol kemunculan; masalah nyata 3.189 grid page list | `T-12` · `docs/Steering/09-DATABASE-STRATEGY.md` §6.3 |

#### Comments

### TKT-F2-003 — Kepemilikan transaksi di lapisan aplikasi

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | F-2 · Gelombang: 1 · Bergantung pada: TKT-F2-001 |
| **Requirement** | FR-F2 |
| **Keputusan** | D-02, D-68 |
| **ADR** | 0007 |
| **Risiko** | R-01 |
| **Rule Pega yang digantikan** | kepemilikan transaksi yang kini berada **di dalam stored procedure** — `Database/INSERT_PLADLA.prc` ber-`COMMIT` **9×** (`:69`, `:74`, `:79`, `:138`, `:143`, `:148`, `:179`, `:184`, `:189`) dengan satu `ROLLBACK` di `:198` yang terjadi **setelah** commit; `Database/UPDATEREAS.prc` ber-`COMMIT` 4× |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`) |
| **Label** | `modul::F-2` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-2-Akses-Data-dan-Nomor-Klaim/issues/03-kepemilikan-transaksi.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Transaksi dimulai dan diakhiri **di satu tempat yang dapat dibaca** — lapisan aplikasi — sehingga
sebuah operasi bisnis berhasil seluruhnya atau gagal seluruhnya.

Nilai bisnisnya konkret dan bernilai uang. Penerbitan PLA/DLA di sistem lama menempuh **sembilan
`COMMIT`**, dan satu-satunya `ROLLBACK` berada di handler terluar yang dijalankan **setelah**
commit terjadi — sehingga tidak memulihkan apa pun. Bila proses berhenti di tengah, sebagian
pemberitahuan ke reasuransi sudah tercatat dan sebagian belum, tanpa cara membatalkannya.

#### Ruang lingkup

- Mekanisme transaksi yang dimulai dan diakhiri di lapisan `App`, **bukan** di Repository maupun
  handler HTTP.
- Aturan **satu permintaan pengguna = satu transaksi**, dengan pengecualian yang wajib
  didokumentasikan di tempatnya.
- **Larangan pemanggilan sistem eksternal di dalam transaksi database** — ditegakkan pemeriksaan,
  bukan imbauan.
- Pola meneruskan transaksi lewat `context` sehingga Repository tidak perlu tahu apakah ia sedang
  berada di dalam transaksi.

#### Non-goal

- **Tidak** menulis ulang procedure mana pun — itu milik `B-4`, `B-9`, dan modul nilai uang
  lainnya. Tiket ini menyediakan mekanismenya.
- **Tidak** memanggil stored procedure (`ADR-0007`).
- **Tidak** mematikan procedure di database — itu langkah tersendiri yang menunggu verifikasi
  `ALL_DEPENDENCIES`.

#### Acceptance criteria

- ☐ Operasi contoh yang menulis ke **tiga tabel** dan gagal di tabel ketiga **tidak
      meninggalkan satu baris pun** — diuji otomatis.
- ☐ Transaksi dimulai **hanya** di lapisan `App`; pemeriksaan lapisan **gagal** bila Repository
      atau handler memulainya — diuji dengan commit percobaan.
- ☐ Pemanggilan HTTP keluar di dalam transaksi **menggagalkan build** lewat pemeriksaan pola —
      diuji dengan commit percobaan.
- ☐ Transaksi yang dibiarkan tanpa commit maupun rollback **terdeteksi** dan menghasilkan galat
      pada uji, bukan koneksi yang menggantung.
- ☐ Lama transaksi tercatat di log (`TKT-F1-003`); transaksi melebihi ambang yang dikonfigurasi
      menghasilkan log bertingkat `warn`.

#### Dependency / Blocked by

Bergantung pada `TKT-F2-001`.

**Yang bergantung padanya:** `B-4` Spreading dan `B-9` PLA/DLA — keduanya baru dapat dibuat
atomik setelah mekanisme ini ada.

#### Constraint keamanan, data, operasional

- Selama masa paralel, **penulis tunggal per tabel** (`P-1`) tetap berlaku: transaksi yang benar
  tidak menyelamatkan tabel yang ditulis dua sistem.
- Transaksi yang lebih panjang menahan kunci baris lebih lama. Dengan 200–300 pengguna (`D-10`)
  risikonya kecil, **tetapi tidak nol pada job massal** — itu sebabnya lama transaksi dicatat.
- **Kontrak galat berbasis string `ErrMsg` tidak dibawa.** Pada enam procedure, `ErrMsg` tidak
  di-set pada jalur sukses sehingga `NULL` berarti berhasil; pada
  `Database/ADD_NEWMASTERVIRTUALACCOUNT.prc:18` kolom yang sama membawa **nomor virtual account
  sekaligus pesan galat**.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** mengembalikan pembungkus transaksi ke versi sebelumnya.

> **Perubahan perilaku yang disengaja dan harus dicatat untuk gerbang 1.** Sistem lama
> meninggalkan data setengah jalan saat gagal; sistem baru tidak meninggalkan apa pun. Kasus uji
> kesetaraan `B-4`/`B-9` harus dirancang menyadari ini, atau ia akan melaporkan **selisih palsu**.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/... -run TestTransaksiAtomik
golangci-lint run                 # aturan: transaksi hanya dimulai di lapisan App
grep -rIn -E "http\.(Get|Post|Client)" internal/app/ | grep -i "tx\|transaksi"   # HARUS 0 baris
go test ./internal/adapter/sqlstore/... -run TestTransaksiMenggantung
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 9 `COMMIT` dan `ROLLBACK` sesudahnya | `Database/INSERT_PLADLA.prc:69,74,79,138,143,148,179,184,189,198` |
| 10 dari 12 procedure melakukan `COMMIT` sendiri | `D-68` · `ADR-0007` |
| `ErrMsg` membawa nomor VA sekaligus pesan galat | `Database/ADD_NEWMASTERVIRTUALACCOUNT.prc:18` |
| Transaksi dimulai di lapisan aplikasi | `docs/Steering/08-TECHNICAL-STRATEGY.md` §4.5 |
| Claim PNC satu-satunya pemanggil procedure | `D-68` — **belum diverifikasi katalog** |

#### Comments

### TKT-F2-004 — Kerangka migrasi skema backward-compatible dan rollback

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang `R-08`** — DDL seluruh tabel tidak tersedia |
| **Modul** | F-2 · Gelombang: 1 · Bergantung pada: TKT-F2-001 |
| **Requirement** | FR-F2 |
| **Keputusan** | D-21, D-27, D-63 |
| **ADR** | 0004 |
| **Risiko** | R-08 |
| **Rule Pega yang digantikan** | — skema dikelola Pega dan DBA; tidak ada rule aplikasi yang mengaturnya. Tabel yang paling terdampak: `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` (**dibaca 116 rule**), `PC_ASSIGN_WORKLIST` (18), `PC_ASSIGN_WORKBASKET` (6), `PR_OPERATORS` (6), `PC_LINK_ATTACHMENT` (3) |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`) |
| **Label** | `modul::F-2` `tipe::fondasi` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-2-Akses-Data-dan-Nomor-Klaim/issues/04-kerangka-migrasi-skema.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Cara mengubah skema database yang **tidak pernah mematikan sistem yang sedang melayani
produksi**, beserta jalan mundurnya.

Nilai bisnisnya diikat dua hal sekaligus: `D-27` menuntut rolling deployment 24/7 sehingga versi
lama dan baru aplikasi berjalan bersamaan terhadap skema yang sama, dan `ADR-0004` menetapkan
database **dibagi dengan Pega**. Satu `ALTER` yang keliru pada tabel header klaim menghentikan
sistem yang dipakai **116 rule Pega**.

#### Ruang lingkup

- Perkakas migrasi berbasis berkas SQL polos yang dapat di-review (`golang-migrate` atau setara).
- Penamaan `NNNN_deskripsi_singkat.up.sql` dan `.down.sql`; **setiap migrasi wajib punya `down`
  yang benar-benar berfungsi**.
- Migrasi dijalankan **terpisah dari start aplikasi**, sebagai langkah deployment tersendiri —
  agar dua instans tidak mencoba bermigrasi bersamaan.
- **Urutan empat rilis** untuk perubahan yang tidak kompatibel, ditulis sebagai prosedur yang
  diikuti, bukan saran.
- Templat **permintaan perubahan skema** sesuai `D-63`: permintaan tertulis → persetujuan Work
  Owner → pelaksanaan DBA → uji Pega dan Go bersamaan.

#### Non-goal

- **Tidak** merancang skema tabel bisnis — itu milik modul bisnisnya.
- **Tidak** menjalankan migrasi apa pun di lingkungan mana pun.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **DDL seluruh tabel** — tipe kolom, panjang, index, constraint (`R-08`) | **DBA** | Tanpa DDL, `down.sql` tidak dapat ditulis benar: mengembalikan kolom ke tipe semula menuntut tahu tipe semula. Menebaknya berarti rollback yang gagal tepat saat dibutuhkan |
| **Statistik ukuran tabel** | **DBA** | Menentukan apakah sebuah `ALTER` dapat dijalankan online atau menuntut jendela pemeliharaan — dan itu mengubah prosedur, bukan hanya estimasi |

Bagian yang **tidak terhalang** dan boleh dikerjakan lebih dulu: pemilihan perkakas, penamaan,
pemisahan migrasi dari start aplikasi, dan templat permintaan `D-63`.

#### Acceptance criteria

- ☐ Migrasi dijalankan lewat perintah **terpisah** dari menjalankan aplikasi — dibuktikan
      dengan menjalankan aplikasi tanpa hak DDL dan aplikasi **tetap** start.
- ☐ Dua proses migrasi yang dijalankan bersamaan **tidak saling merusak** — satu berhasil, satu
      menunggu atau gagal bersih. Diuji.
- ☐ Setiap migrasi contoh punya `down.sql` yang **mengembalikan skema ke keadaan semula** —
      diuji `up` lalu `down` lalu bandingkan definisi skema.
- ☐ Migrasi yang **menghapus kolom dalam satu langkah ditolak** oleh pemeriksaan — penghapusan
      wajib dua tahap.
- ☐ Templat permintaan perubahan skema memuat keempat langkah `D-63` dan **bagian rollback yang
      tidak boleh kosong**.
- ☐ Prosedur empat rilis terdokumentasi dengan contoh nyata satu penggantian nama kolom.

#### Dependency / Blocked by

Bergantung pada `TKT-F2-001`. **Terhalang `R-08`** untuk bagian yang menyentuh tabel nyata.

#### Constraint keamanan, data, operasional

- **Backward-compatible tanpa perkecualian** (`P-4`): tambah kolom boleh langsung asal *nullable*
  atau ber-*default*; hapus kolom dua tahap; ganti nama **tidak pernah langsung**; ubah tipe lewat
  kolom baru.
- **Akun aplikasi tidak boleh punya hak DDL** di produksi — migrasi dijalankan akun terpisah.
- Setiap perubahan skema **wajib diuji dengan menjalankan Pega dan Go bersamaan** terhadap skema
  hasil perubahan (`D-63`). Itu tidak dapat dilakukan DBA sendirian maupun tim pengembang
  sendirian.

#### Migrasi skema / rollout / rollback

Tiket ini **adalah** mekanisme rollback-nya. Rollback tiket ini sendiri: menghapus perkakas
migrasi; tidak ada skema yang berubah karena belum ada migrasi yang dijalankan.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
migrate -path db/migrations -database "$DSN" up
migrate -path db/migrations -database "$DSN" down 1
### bandingkan definisi skema sebelum dan sesudah
./scripts/dump-schema.sh > /tmp/sesudah.sql && diff /tmp/sebelum.sql /tmp/sesudah.sql  # HARUS kosong
### uji dua migrasi bersamaan
( migrate ... up & migrate ... up & wait )   # tidak boleh merusak
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Tabel header klaim dibaca 116 rule Pega | `D-21` · `ADR-0004` |
| Rolling deployment 24/7 menuntut backward-compatible | `D-27` · `P-4` |
| Prosedur perubahan skema tiga pihak | `D-63` · `docs/Steering/09-DATABASE-STRATEGY.md` §9.1 |
| Urutan empat rilis | `docs/Steering/09-DATABASE-STRATEGY.md` §9 |
| DDL tidak tersedia | `R-08` |

#### Comments

### TKT-F2-005 — Soft delete sebagai pola akses data

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | F-2 · Gelombang: 1 · Bergantung pada: TKT-F2-001, TKT-F2-002 |
| **Requirement** | FR-F2 |
| **Keputusan** | D-66 (menyupersede D-65), D-28 |
| **ADR** | 0012, 0026 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | penghapusan fisik yang tersebar — `RDB List/InsertClaimPNC-SQL.xml:77` (`DELETE` pada `POOLDATA.JSON_KLAIM_LOG`) · `RDB List/UpdateLogServiceClaim-SQL.xml:27` (`UPDATE` pada `pooldata.claim_service_log`) · `RDB List/BrowseOldEmailCoas-SQL.xml:69` · `RDB List/UpdateMasterLoginSurvey-SQL.xml:9` · 12 step `RDB-DELETE` di 7 activity · 2 step `OBJ-DELETE` di 2 activity |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`) |
| **Label** | `modul::F-2` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-2-Akses-Data-dan-Nomor-Klaim/issues/05-pola-soft-delete.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu pola penghapusan yang dipakai seluruh aplikasi: **data bernilai bisnis tidak pernah dibuang**,
penghapusan dinyatakan lewat penanda beserta waktu dan pelakunya.

Nilai bisnisnya bertaut langsung dengan `ADR-0023`: karena satuan izin adalah menu dan **tidak ada
pemisahan tugas**, jejak audit menjadi **satu-satunya kontrol pengimbang**. Jejak audit yang
barisnya masih bisa dihapus bukan kontrol apa pun — dan sistem lama membuktikan itu terjadi: dua
tabel yang namanya log terbukti dimutasi.

#### Ruang lingkup

- Kolom penanda baku untuk seluruh tabel bernilai bisnis: penanda terhapus, waktu, dan pelaku.
- **Pembantu kueri** yang menyaring baris terhapus secara baku, sehingga menyaringnya menjadi
  perilaku bawaan dan **menampilkannya** yang menuntut niat eksplisit.
- **Pemeriksaan otomatis**: pernyataan `DELETE` pada tabel bernilai bisnis menggagalkan build.
- Pola penanganan **keunikan kunci alami** ketika baris terhapus masih menempati nilai kuncinya.
- Catatan indexing: index pada tabel bernilai bisnis memperhitungkan adanya baris mati.

#### Non-goal

- **Tidak** menetapkan siapa yang boleh melihat data terhapus dan lewat layar apa — itu keputusan
  Work Owner yang masih terbuka (`ADR-0012`).
- **Tidak** membangun arsip atau purge — itu bertaut retensi (`D-62`), dan angkanya belum ada.
- **Tidak** menyelesaikan pola pengganti hapus-lalu-sisip-ulang — itu `ADR-0013`, masih
  `Proposed`, dan menjadi tiket `B-2`.

#### Acceptance criteria

- ☐ Pembantu kueri **menyaring baris terhapus secara baku** — kueri yang ditulis tanpa
      menyebutkan apa pun **tidak** mengembalikan baris terhapus. Diuji.
- ☐ Menampilkan baris terhapus menuntut pemanggilan eksplisit — diuji bahwa jalur biasa tidak
      bisa melakukannya secara tidak sengaja.
- ☐ Pemeriksaan pola **menggagalkan build** pada `DELETE FROM` terhadap tabel bernilai bisnis —
      diuji dengan commit percobaan.
- ☐ Menghapus lalu menyisipkan ulang baris dengan kunci alami yang sama **berhasil** dan tidak
      melanggar constraint unik — diuji pada tabel contoh.
- ☐ Penanda terhapus menyimpan **waktu (UTC) dan pelaku**; keduanya wajib terisi — diuji bahwa
      penghapusan tanpa pelaku ditolak.
- ☐ Satu baris jejak audit tercatat pada setiap penghapusan (`S-5`) — diuji setelah `S-5` ada;
      sampai itu, tiket ini menyediakan titik pemanggilannya.

#### Dependency / Blocked by

Bergantung pada `TKT-F2-001` dan `TKT-F2-002`.

**Yang bergantung padanya:** seluruh modul bisnis, dan `S-8` — karena perkakas uji kesetaraan
harus tahu bahwa membandingkan jumlah baris tabel akan selalu berbeda.

#### Constraint keamanan, data, operasional

- Tabel **tumbuh permanen**. Di atas data historis puluhan juta baris (`D-10`), ini keputusan
  kapasitas, bukan sekadar kepatuhan.
- **Setiap kueri pembaca wajib menyaring** — satu kueri yang lupa akan menampilkan data yang
  seharusnya hilang. Ini **kelas cacat baru** yang tidak ada di sistem lama, dan itulah alasan
  penyaringan dijadikan perilaku bawaan, bukan tanggung jawab penulis kueri.
- Soft delete **tidak berlaku** pada berkas di storage eksternal (`ADR-0010`): menghapus metadata
  tidak menghapus berkasnya, dan itu keputusan tersendiri yang belum diambil.

#### Migrasi skema / rollout / rollback

Menambah kolom penanda ke tabel bernilai bisnis — **tambah kolom yang *nullable***, sehingga
backward-compatible dan aman bagi Pega yang masih membaca tabel yang sama (`P-4`).

**Rollback:** kolom penanda dibiarkan ada dan diabaikan; tidak ada data yang hilang. Menghapus
kolomnya menempuh prosedur dua tahap `TKT-F2-004`.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/sqlstore/... -run TestSoftDelete
grep -rIn -E "DELETE\s+FROM" internal/ | grep -v "_test.go"   # HARUS 0 baris pada tabel bisnis
go test ./internal/adapter/sqlstore/... -run TestKunciAlamiSetelahSoftDelete
make lint-sql                                                  # pemeriksaan pola DELETE
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Soft delete menyeluruh | `D-66` · `ADR-0012` |
| `DELETE` pada tabel log | `RDB List/InsertClaimPNC-SQL.xml:77` |
| `UPDATE` pada tabel log | `RDB List/UpdateLogServiceClaim-SQL.xml:27` |
| 12 step `RDB-DELETE` + 2 step `OBJ-DELETE` | `docs/verifikasi-bukti-adr.md` (D-66) |
| Jejak audit satu-satunya kontrol pengimbang | `D-59` · `ADR-0023` · `ADR-0026` |
| Uji kesetaraan tidak boleh membandingkan `COUNT(*)` | `docs/Steering/14-TESTING-STRATEGY.md` §6.4 |

#### Comments

### TKT-F2-006 — Generator nomor klaim `PNCN.YY.xxxx`

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan** — dua pertanyaan bentuk nomor belum dijawab |
| **Modul** | F-2 · Gelombang: 1 · Bergantung pada: TKT-F2-001, TKT-F2-002 |
| **Requirement** | FR-F2, DAT-03 |
| **Keputusan** | D-22, D-71, D-76 |
| **ADR** | 0005, 0009 |
| **Risiko** | R-12 |
| **Rule Pega yang digantikan** | pembentukan `pzInsKey` berformat `ASM-FW-GCNMFW-WORK PNC-xxxx` — kunci teknis Pega yang bocor menjadi identitas bisnis |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`); nomor yang dihasilkan diuji pengguna lewat `B-2` |
| **Label** | `modul::F-2` `tipe::fondasi` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-2-Akses-Data-dan-Nomor-Klaim/issues/06-generator-nomor-klaim.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu fungsi yang menerbitkan nomor klaim sistem baru, terisolasi di satu berkas, dan **satu-satunya
tempat dengan percabangan dialek database** di seluruh aplikasi.

Nilai bisnisnya: nomor klaim muncul di surat ke tertanggung, di PLA/DLA ke koasuransi dan
reasuransi, serta di pelaporan — dan **tidak dapat diubah surut**. Prefix `PNCN` juga membuat asal
sebuah klaim terbaca langsung selama masa paralel, ketika dua sistem menerbitkan klaim bersamaan.

#### Ruang lingkup

- Fungsi penerbit nomor berformat **`PNCN.YY.xxxx`** dengan sintaks Oracle yang ditetapkan `D-71`:

  ```sql
  'PNCN' || '.' || TO_CHAR(SYSDATE,'RR') || '.' || TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)
  ```

- Isolasi di satu berkas (`internal/adapter/sqlstore/nomor_klaim.go`) dengan padanan PostgreSQL
  yang siap dipakai: `nextval('pooldata.claim_no_nonpega_seq')` + `to_char(current_date,'YY')`.
- Pembacaan nomor **kedua format** — `PNCN.YY.xxxx` dan warisan `PNC-xxxx` — karena keduanya hidup
  berdampingan permanen.
- Permintaan pembuatan sequence `POOLDATA.CLAIM_NO_NONPEGA_SEQ` ke DBA lewat prosedur `D-63`.

#### Non-goal

- **Tidak** menomori ulang klaim lama. Nomor lama sudah tercetak di surat dan sudah dikirim ke
  reasuransi (`ADR-0009`).
- **Tidak** membangun alur registrasi — itu `B-2`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Akibat pada tiket ini |
|---|---|---|
| **Apakah sequence direset setiap awal tahun?** Bila tidak, nomor urut menembus pergantian tahun (`PNCN.26.8125` → `PNCN.27.8126`) dan segmen tahun menjadi penanda, bukan penghitung per tahun | **Work Owner** | Mengubah **AC** dan permintaan DDL sequence ke DBA |
| **Apakah lebar segmen terakhir dibuat tetap** (mis. `TO_CHAR(seq.NEXTVAL,'FM0000')`)? Tanpa itu, lebar berubah-ubah dan **pengurutan sebagai teks tidak sesuai urutan penerbitan** — `.10` mendahului `.9` | **Work Owner** | Mengubah AC pengurutan dan setiap layar/laporan yang mengurutkan berdasarkan nomor klaim |

`POOLDATA.CLAIM_NO_NONPEGA_SEQ` **nol kemunculan di seluruh export** — sequence-nya memang belum
ada dan akan dibuat. Itu **bukan** penghalang; ia pekerjaan DBA yang sudah diketahui bentuknya.

> **Satu pertanyaan sudah tertutup (`D-76`).** Karena `D-75` menetapkan satu database per
> entitas, sequence-nya terpisah per portal — dan Work Owner memutuskan **tidak perlu penanda
> portal**. Nomor klaim unik **di dalam satu portal**, dan **tidak dijamin unik antar portal**.
> Konsekuensinya dicatat di `D-76`.

#### Acceptance criteria

> Dua butir bertanda ⚠️ berubah tergantung jawaban di atas.

- ☐ Nomor yang diterbitkan cocok dengan pola `^PNCN\.\d{2}\.\d+$` — diuji 1.000 penerbitan
      berturut-turut.
- ☐ **Tidak ada nomor ganda** pada 1.000 penerbitan dari **dua proses bersamaan** — diuji;
      keunikan datang dari sequence, bukan dari penguncian aplikasi.
- ☐ Sakelar dialek berada di **tepat satu berkas** — diuji dengan pemindaian: `NEXTVAL` dan
      `nextval(` tidak muncul di berkas lain mana pun.
- ☐ Pembaca nomor menerima **kedua format** dan dapat menyatakan asalnya (Pega atau Go) — diuji
      dengan enam contoh nomor.
- ☐ ⚠️ Perilaku pada pergantian tahun sesuai keputusan Work Owner — diuji dengan seam Clock
      (`F-5`) yang memajukan waktu melewati 31 Desember.
- ☐ ⚠️ Pengurutan nomor klaim sesuai keputusan Work Owner — diuji dengan deret `.9`, `.10`,
      `.100`.

#### Dependency / Blocked by

- Bergantung pada `TKT-F2-001`, `TKT-F2-002`.
- **Terhalang dua keputusan Work Owner** di atas.
- Membutuhkan sequence dibuat DBA lewat prosedur `D-63`.

#### Constraint keamanan, data, operasional

- **Tahun diambil dari `SYSDATE`**, yaitu tanggal server basis data — bukan tanggal kejadian dan
  bukan tanggal registrasi. Klaim yang terbit di sekitar pergantian tahun mengambil tahun dari jam
  server, bertaut dengan `R-12` (pergeseran zona waktu).
- Nomor klaim **tidak boleh** dibentuk di lapisan Domain — ia menyentuh database, jadi tempatnya
  Adapter.

#### Migrasi skema / rollout / rollback

Menambah **satu sequence baru**; tidak mengubah tabel mana pun, sehingga tidak memengaruhi Pega.

**Rollback:** sequence dibiarkan ada dan tidak dipakai. Nomor yang telanjur terbit **tidak dapat
ditarik** — itu sebabnya kedua pertanyaan di atas harus dijawab **sebelum** klaim pertama terbit,
bukan sesudahnya.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/sqlstore/... -run TestNomorKlaim
go test ./internal/adapter/sqlstore/... -run TestNomorKlaimDuaProsesBersamaan
grep -rIn -iE "nextval" internal/ | grep -v nomor_klaim.go     # HARUS 0 baris
go test ./internal/adapter/sqlstore/... -run TestNomorKlaimPergantianTahun
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Format `PNCN.YY.xxxx` beserta sintaksnya | `D-71` · `ADR-0009` |
| Format lama `PNCN-xxxx` disupersede | `D-22` dibatasi `D-71` |
| Sequence belum ada — nol kemunculan di export | `ADR-0009` · verifikasi langsung |
| Satu-satunya sakelar dialek | `ADR-0005` §3.1 · `docs/Steering/09-DATABASE-STRATEGY.md` §3.1 |
| Prefix Pega `ASM-FW-GCNMFW-WORK` ditinggalkan | `D-22` · `CONTEXT.md` "Istilah yang sengaja tidak dipakai lagi" |

#### Comments

### TKT-F2-007 — Daftar putih kolom filter dan sort pengganti `{ASIS:}`

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan** — himpunan filter yang sah belum ditetapkan |
| **Modul** | F-2 · Gelombang: 1 · Bergantung pada: TKT-F2-002 |
| **Requirement** | FR-F2, FR-R1 |
| **Keputusan** | D-15 |
| **ADR** | 0005, 0023 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | **538 pemakaian `{ASIS:}`** di rule Connect-SQL — penyisipan nilai langsung ke teks SQL tanpa parameter binding |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`) |
| **Label** | `modul::F-2` `tipe::keamanan` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-2-Akses-Data-dan-Nomor-Klaim/issues/07-daftar-putih-filter-dan-sort.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Cara menyaring dan mengurutkan daftar yang **tidak pernah menerima nama kolom dari pengguna**,
menggantikan pola `{ASIS:}` yang menyisipkan potongan SQL apa adanya.

Nilai bisnisnya adalah menutup **satu-satunya celah injeksi SQL yang terbukti ada** di sistem
lama. Parameter binding menutup nilai; ia **tidak** menutup nama kolom dan arah pengurutan — dan
itulah yang `{ASIS:}` sisipkan.

#### Ruang lingkup

- Mekanisme daftar putih: setiap daftar mendeklarasikan **kolom mana yang boleh difilter dan
  diurutkan**, dan permintaan di luar daftar itu **ditolak**, bukan diabaikan diam-diam.
- Pemetaan **nama yang dilihat klien** ke nama kolom database, sehingga struktur tabel tidak bocor
  lewat parameter URL.
- Pemeriksaan otomatis: perangkaian apa pun ke dalam klausa `WHERE` atau `ORDER BY`
  menggagalkan build.
- Inventaris 538 pemakaian `{ASIS:}`, dikelompokkan menurut **apa yang sebenarnya disisipkan**:
  nilai (dapat langsung diparameterkan), nama kolom, potongan klausa, atau daftar `IN`.

#### Non-goal

- **Tidak** menulis ulang 538 kueri — itu tersebar di modul bisnis masing-masing. Tiket ini
  menyediakan mekanisme dan inventarisnya.
- **Tidak** menetapkan izin per peran atas kolom — itu `F-3`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Himpunan filter yang sah untuk setiap daftar** — 538 pemakaian `{ASIS:}` belum dipilah mana yang benar-benar dipakai pengguna dan mana yang sisa tambalan lama | **Work Owner** (mana yang dipakai bisnis) + **Lead Engineer** (pemilahan teknis) | Daftar putih yang dibuat dari tebakan akan **menghapus penyaringan yang dipakai orang**, dan itu baru ketahuan setelah rilis |

**Yang tidak terhalang dan boleh dikerjakan lebih dulu:** mekanisme daftar putihnya sendiri,
pemetaan nama klien → kolom, dan pemeriksaan otomatis. Yang menunggu adalah **isi** daftarnya.

#### Acceptance criteria

- ☐ Permintaan filter pada kolom **di luar daftar putih** mengembalikan galat `400` dengan pesan
      yang menyebut nama parameter — **bukan** diabaikan dan **bukan** `500`.
- ☐ Permintaan sort pada kolom di luar daftar putih ditolak dengan cara yang sama.
- ☐ Nama kolom database **tidak pernah muncul** di parameter permintaan maupun respons galat —
      diuji dengan permintaan yang sengaja salah.
- ☐ Pemeriksaan pola **menggagalkan build** pada perangkaian ke klausa `WHERE`/`ORDER BY` —
      diuji dengan commit percobaan.
- ☐ Inventaris 538 pemakaian `{ASIS:}` tersedia sebagai berkas, terbagi ke **empat kategori**,
      dengan jumlah per kategori yang dapat dihitung ulang.
- ☐ Uji injeksi: nilai `1; DROP TABLE x --` pada setiap parameter filter **tidak mengubah
      apa pun** dan menghasilkan galat yang sama dengan masukan tidak sah biasa.

#### Dependency / Blocked by

- Bergantung pada `TKT-F2-002`.
- **Terhalang keputusan** di atas untuk isi daftar putihnya.

#### Constraint keamanan, data, operasional

- **Parameter binding tanpa perkecualian** untuk nilai; daftar putih untuk nama kolom. Keduanya
  wajib, dan tidak saling menggantikan.
- Sampai daftar putih ada, kueri baru **tidak boleh** menerima nama kolom dari pengguna sama
  sekali — lebih baik daftar tanpa penyaringan daripada daftar dengan celah injeksi.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** melonggarkan daftar putih **tidak boleh** dilakukan sebagai
rollback — bila sebuah filter yang dibutuhkan ternyata tidak ada di daftar, yang ditambah adalah
daftarnya, bukan mekanismenya.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/http/... -run TestDaftarPutihFilter
go test ./internal/adapter/http/... -run TestInjeksiSQLDitolak
grep -rIn -E "(WHERE|ORDER BY).*\+" internal/adapter/sqlstore/   # HARUS 0 baris
make lint-sql
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 538 pemakaian `{ASIS:}` | `docs/verifikasi-bukti-adr.md` §15 baris `F-2` |
| Parameter binding wajib; nama kolom sort/filter hanya dari daftar yang diizinkan | `docs/Steering/11-SECURITY.md` §5 · `BRD` §17.3 |
| Perangkaian SQL dari nilai pengguna sebagai utang teknis | `docs/Steering/03-CURRENT-ARCHITECTURE.md` |
| Kriteria penerimaan #11 — tanpa perangkaian SQL | `BRD §21.2` |

#### Comments

# 4. F-3 — Identitas & Akses

*Modul Fondasi · folder `docs/ticketing/F-3-Login-dan-Hak-Akses-Menu/`*

## Spesifikasi Modul

| | |
|---|---|
| **Modul** | `F-3` Identitas & Akses |
| **Gelombang** | 1 — Fondasi |
| **Ukuran** | **Besar — tanpa baseline Pega** |
| **Bergantung pada** | `F-1`, `F-2` |
| **Kesiapan** | **TERHALANG untuk identitas · PENUH untuk middleware otorisasi per endpoint** |
| **Cakupan tiket** | penuh untuk bagian yang buktinya lengkap; `needs-info` untuk bagian terhalang (`D-41` Opsi 1) |

### Apa yang dibangun

Cara pengguna masuk, cara aplikasi mengenalinya, dan cara aplikasi memutuskan ia boleh melakukan
apa.

### Kenapa modul ini paling awal menggigit

`F-3` adalah **prasyarat login seluruh aplikasi**. Tidak ada modul yang dapat dirilis ke pengguna
tanpanya — dan justru di modul ini penghalangnya paling keras: **kontrak API HCC/HCQ tidak ada**.

`HCC` dan `HCQ` muncul **2× di seluruh export**, dan keduanya teks pesan galat yang menyuruh
pengguna menghubungi helpdesk. Tidak ada Connect REST, tidak ada pemetaan field respons, tidak ada
penanganan kegagalan. Integrasi ini **greenfield sepenuhnya**, dan `F-3` **tidak punya baseline
untuk diuji kesetaraannya** (`D-56`).

### Otorisasi: yang ini justru jelas

Berbeda dari identitas, sisi otorisasi punya bukti lengkap:

| Fakta | Angka |
|---|---|
| Access group Pega, seluruhnya terverifikasi sebagai literal `GCNMFW:<nama>` | **22** |
| Item menu yang dipetakan | **51** |
| When rule yang memuat pemetaan peran→menu | **34** |
| `pyPrivilegeName` yang terisi di seluruh 902 activity | **1** — dan itu privilege bawaan Pega untuk ekspor ruleset, bukan aturan bisnis |

Artinya otorisasi sistem lama **hanya penyembunyian menu**. `D-59` menetapkan satuan izin tetap
**menu**, tanpa pemisahan tugas — yang berubah adalah **tempat penegakannya**, dari antarmuka
menjadi server pada setiap endpoint.

### Penghalang modul ini

| Jenis | Isi | Pemilik | Menahan |
|---|---|---|---|
| **Artefak** | **Kontrak API HCC/HCQ** — nol jejak di export | Tim HCC/HCQ | `TKT-F3-002` |
| **Artefak** | **Tidak ada folder Access Group / Role / Privilege / Operator sama sekali** di export | Tim Pega | `TKT-F3-004` sebagian |
| **Artefak** | **5 When rule peran hilang**: `IsGCNMReport`, `IsKomite`, `IsNotViewClaim`, `IsPNCBonding`, `IsSurvey` — masing-masing mengendalikan satu item menu | Tim Pega | `TKT-F3-004` |
| **Artefak** | **Penugasan operator ke peran tidak ada di database.** `POOLDATA.T_ACCESS_GROUP_PNC` hanya memetakan `OPERATOR_ID` → `OLD_OPERATOR_ID` | DBA + Work Owner | `TKT-F3-004` pengisian |
| **Keputusan** | Pemetaan 22 access group → 51 menu masih akurat? · `T_ACCESS_GROUP_PNC` sebenarnya untuk apa? · pernah ada temuan audit soal 901 dari 902 activity tanpa privilege? | Work Owner | `TKT-F3-004` |

### Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-F3-001](issues/01-seam-identity-dan-adapter-fake.md) | Seam Identity dan adapter fake untuk pengembangan | `ready-for-human` | siap |
| [TKT-F3-002](issues/02-adapter-hcc-hcq.md) | Adapter autentikasi HCC/HCQ | `needs-info` | **terhalang artefak** |
| [TKT-F3-003](issues/03-sesi-dan-token-milik-aplikasi.md) | Sesi dan token milik aplikasi | `ready-for-human` | siap |
| [TKT-F3-004](issues/04-tabel-peran-dan-izin-menu.md) | Tabel 22 peran dan 51 izin menu | `needs-info` | **terhalang artefak** |
| [TKT-F3-005](issues/05-middleware-otorisasi-per-endpoint.md) | Middleware otorisasi di setiap endpoint | `ready-for-human` | siap — bagian **PENUH** modul ini |

## Daftar Tiket

### TKT-F3-001 — Seam Identity dan adapter fake untuk pengembangan

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | F-3 · Gelombang: 1 · Bergantung pada: TKT-F1-001 |
| **Requirement** | FR-F3 |
| **Keputusan** | D-07 |
| **ADR** | 0024 |
| **Risiko** | R-14 |
| **Rule Pega yang digantikan** | **tidak ada padanan** — HCC/HCQ muncul **2× di seluruh export**, keduanya teks pesan galat yang menyuruh menghubungi helpdesk |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`); gerbang 1 diganti uji fungsional terhadap kontrak (`D-56`) |
| **Label** | `modul::F-3` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-3-Login-dan-Hak-Akses-Menu/issues/01-seam-identity-dan-adapter-fake.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Antarmuka autentikasi yang **implementasinya dapat diganti**, ditambah satu implementasi tiruan —
sehingga `F-3` dan seluruh modul yang bergantung padanya dapat dikerjakan **tanpa menunggu kontrak
HCC/HCQ**.

Nilai bisnisnya adalah membuka jalan buntu. `F-3` memblokir login seluruh aplikasi, dan kontraknya
dipegang pihak luar. Tanpa seam ini, seluruh papan tiket berhenti menunggu satu dokumen.

#### Ruang lingkup

- Antarmuka `Identity` dideklarasikan di lapisan Domain: menerima kredensial, mengembalikan profil
  pengguna (NIK, nama, cabang, jabatan, email) atau galat.
- **Adapter fake** berbasis daftar pengguna lokal untuk pengembangan dan pengujian, yang
  **menolak berjalan di lingkungan produksi**.
- Jenis galat yang dibedakan: kredensial salah, pengguna tidak aktif, dan **sistem identitas tidak
  dapat dihubungi** — ketiganya menuntut perlakuan berbeda.

#### Non-goal

- **Tidak** membangun adapter HCC/HCQ — itu `TKT-F3-002`, terhalang kontrak.
- **Tidak** menerbitkan sesi — itu `TKT-F3-003`.
- **Tidak** menangani otorisasi. Otorisasi **tidak** berada di seam ini; ia dimiliki aplikasi dan
  hidup di Domain (`ADR-0023`).

#### Acceptance criteria

- ☐ Antarmuka `Identity` berada di lapisan **Domain**; adapter di lapisan Adapter — pemeriksaan
      lapisan `TKT-F1-001` **gagal** bila tertukar.
- ☐ Adapter fake **menolak start** bila lingkungan bertanda produksi — diuji, dan aplikasi gagal
      keras dengan pesan yang jelas.
- ☐ Ketiga jenis galat dapat dibedakan pemanggil tanpa memeriksa teks pesan — diuji.
- ☐ Seluruh modul yang membutuhkan identitas dapat diuji dengan adapter fake **tanpa jaringan**
      — dibuktikan dengan menjalankan uji dalam keadaan jaringan dimatikan.
- ☐ Profil pengguna yang dikembalikan memuat kelima field, dan **field yang kosong ditolak**
      sebagai galat, bukan diteruskan diam-diam.

#### Dependency / Blocked by

Bergantung pada `TKT-F1-001`.

**Yang bergantung padanya:** `TKT-F3-002`, `TKT-F3-003`, dan seluruh modul yang memerlukan
identitas pengguna.

#### Constraint keamanan, data, operasional

- Adapter fake **tidak boleh** dapat diaktifkan di produksi lewat konfigurasi saja — penolakannya
  ada di kode, bukan hanya di nilai konfigurasi.
- Kredensial **tidak pernah** masuk log (`TKT-F1-003`), termasuk pada jalur galat.
- Adapter fake memakai daftar pengguna contoh yang **bukan** nama pegawai nyata.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** mengembalikan versi seam; adapter fake tidak pernah berjalan
di produksi sehingga tidak ada dampak data.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/identitas/...
go test ./internal/adapter/identitas/... -run TestFakeMenolakProduksi
APP_ENV=production go run ./cmd/app     # HARUS gagal bila adapter fake aktif
go test ./... -run TestTanpaJaringan
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Autentikasi didelegasikan ke HCC/HCQ; otorisasi dimiliki aplikasi | `D-07` · `ADR-0024` |
| HCC/HCQ muncul 2× di export, keduanya teks galat | `T-2` · `docs/verifikasi-bukti-adr.md` §1.1 |
| Seam Identity dengan dua adapter | `docs/Steering/04-FUTURE-ARCHITECTURE.md` §3.5 |
| `F-3` tidak punya baseline; gerbang 1 diganti uji kontrak | `D-56` · `ADR-0028` |

#### Comments

### TKT-F3-002 — Adapter autentikasi HCC/HCQ

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** — kontrak API HCC/HCQ tidak ada |
| **Modul** | F-3 · Gelombang: 1 · Bergantung pada: TKT-F3-001 |
| **Requirement** | FR-F3 |
| **Keputusan** | D-07, D-56 |
| **ADR** | 0024, 0028 |
| **Risiko** | R-14 |
| **Rule Pega yang digantikan** | **tidak ada** — nol Connect REST ke HCC/HCQ di seluruh export |
| **Peran penguji gerbang 2** | **tidak berlaku** — `D-60` |
| **Label** | `modul::F-3` `tipe::integrasi` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-3-Login-dan-Hak-Akses-Menu/issues/02-adapter-hcc-hcq.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Adapter yang memvalidasi kredensial pengguna ke sistem identitas internal HCC/HCQ.

Nilai bisnisnya: **tanpa ini tidak ada pengguna yang dapat masuk**. Ini penghalang paling awal di
seluruh rencana migrasi — bukan paling besar, tetapi paling awal.

#### Ruang lingkup

- Adapter `Identity` yang memanggil API HCC/HCQ, memetakan responsnya ke profil pengguna.
- Penanganan kegagalan: timeout, kredensial salah, pengguna tidak aktif, dan **HCC/HCQ tidak dapat
  dihubungi**.
- **Pencocokan identitas HCC/HCQ dengan `OPERATOR_ID`** yang dipakai di seluruh data klaim.
- Uji fungsional terhadap kontrak — inilah pengganti gerbang 1 bagi bagian ini (`D-56`).

#### Non-goal

- **Tidak** menyimpan kata sandi. Aplikasi ini tidak pernah menjadi pemilik kredensial (`D-07`).
- **Tidak** menerbitkan sesi — itu `TKT-F3-003`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Kontrak API HCC/HCQ** — endpoint, format permintaan dan respons, kode galat, batas percobaan, endpoint refresh/validasi | **Tim HCC/HCQ** | Tidak ada satu pun jejaknya di export. Menebak bentuknya berarti menulis adapter yang hampir pasti salah |
| **Apa yang terjadi bila HCC/HCQ tidak dapat dihubungi** — seluruh aplikasi tidak dapat diakses, atau ada jalur cadangan? | **Work Owner** | Menyentuh tuntutan 24/7 `D-27`. Bila tidak ada jalur cadangan, ketersediaan aplikasi ikut bergantung pada sistem lain |
| **Cara mencocokkan identitas HCC/HCQ dengan `OPERATOR_ID`** yang dipakai di seluruh data klaim | **Work Owner + Tim HCC/HCQ** | Tanpa pemetaan ini, pengguna yang berhasil login **tetap tidak dikenali oleh data klaimnya sendiri** — ini butir yang paling mudah terlewat |
| **Masa berlaku sesi dan cara memperbaruinya** | **Work Owner + Security** | Menentukan bentuk `TKT-F3-003` |

#### Acceptance criteria

> **Belum dapat ditulis dengan angka.** Bentuk AC yang akan diisi setelah kontrak diterima:

- ☐ Login dengan kredensial benar mengembalikan profil lengkap berisi kelima field.
- ☐ Login dengan kredensial salah mengembalikan galat yang **tidak membedakan** apakah pengguna
      ada atau tidak — mencegah pencacahan pengguna.
- ☐ HCC/HCQ yang tidak merespons dalam batas waktu menghasilkan galat yang **dapat dibedakan**
      dari kredensial salah, dan perilaku aplikasi sesuai keputusan Work Owner.
- ☐ Setiap pemanggilan tercatat: tujuan, lama, hasil — **tanpa kredensial** (`TKT-F1-003`).
- ☐ Pemetaan identitas → `OPERATOR_ID` terbukti pada minimal 10 pengguna nyata di staging.
- ☐ Uji fungsional mencakup **seluruh** kode galat yang disebut kontrak — jumlahnya dilaporkan
      sebagai angka.

#### Dependency / Blocked by

Bergantung pada `TKT-F3-001`. **Terhalang Tim HCC/HCQ.**

**Yang terhalang olehnya:** login seluruh aplikasi, karenanya rilis modul apa pun ke pengguna.

#### Constraint keamanan, data, operasional

- Kredensial **tidak pernah** dicatat, tidak di log dan tidak di jejak audit.
- Kredensial aplikasi untuk memanggil HCC/HCQ (bila ada) tunduk `D-40` yang masih `OPEN`.
- Galat autentikasi **tidak boleh** membocorkan keberadaan akun.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** kembali ke adapter fake **tidak tersedia di produksi**
(`TKT-F3-001`) — bila adapter HCC/HCQ gagal di produksi, yang dilakukan adalah menunda rilis,
bukan mengaktifkan fake.

#### Rencana verifikasi

> **Rencana — belum dijalankan, dan belum dapat disusun lengkap tanpa kontrak.**

```bash
go test ./internal/adapter/identitas/... -run TestHCCHCQ
go test ./internal/adapter/identitas/... -run TestSeluruhKodeGalatKontrak
go run ./cmd/tools/cek-pemetaan-operator --sample 10
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| HCC/HCQ nol jejak; hanya 2 teks pesan galat | `T-2` · `docs/verifikasi-bukti-adr.md` §1.1 |
| Autentikasi didelegasikan; aplikasi menerbitkan sesi sendiri | `D-07` |
| Gerbang 1 diganti uji fungsional terhadap kontrak | `D-56` · `ADR-0028` |
| Empat pertanyaan terbuka | `ADR-0024` Pertanyaan terbuka |
| `OPERATOR_ID` dipakai di seluruh data klaim | `docs/Steering/11-SECURITY.md` §2.1 |

#### Comments

### TKT-F3-003 — Sesi dan token milik aplikasi

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | F-3 · Gelombang: 1 · Bergantung pada: TKT-F3-001, TKT-F2-001 |
| **Requirement** | FR-F3 |
| **Keputusan** | D-07, D-27 |
| **ADR** | 0024 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | mekanisme sesi Pega — **tidak ada rule aplikasi yang mengaturnya**; tidak ada padanan yang dapat dibandingkan |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`) |
| **Label** | `modul::F-3` `tipe::keamanan` `status::ready-for-human` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-3-Login-dan-Hak-Akses-Menu/issues/03-sesi-dan-token-milik-aplikasi.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Aplikasi menerbitkan **sesi miliknya sendiri** setelah sistem identitas memvalidasi kredensial —
bukan meneruskan token pihak lain.

Nilai bisnisnya ada di tiga hal yang menjadi milik kita: masa berlaku, pencabutan, dan isi token.
Ditambah satu yang penting secara operasional: **bila HCC/HCQ sedang bermasalah, pengguna yang
sudah masuk tetap dapat bekerja** — sejalan dengan tuntutan 24/7 (`D-27`).

#### Ruang lingkup

- Penerbitan token sesi setelah autentikasi berhasil, memuat identitas pengguna dan perannya.
- **Penyimpanan sesi aktif di database** agar dapat dicabut — bukan hanya token yang berdiri
  sendiri. Ini juga syarat aplikasi **stateless**: dua instans harus melihat sesi yang sama.
- Pembaruan sesi dan penghentian saat keluar.
- Middleware yang memuat identitas pengguna ke dalam `context` setiap permintaan.

#### Non-goal

- **Tidak** memutuskan izin — itu `TKT-F3-005`.
- **Tidak** membangun layar login — itu `U-1`.

#### Acceptance criteria

- ☐ Token yang diterbitkan **tidak memuat kredensial** dan tidak memuat data nasabah — diuji
      dengan membongkar isinya.
- ☐ Sesi tersimpan di database; **dua instans aplikasi mengenali sesi yang sama** — diuji dengan
      menerbitkan sesi di instans A dan memakainya di instans B.
- ☐ Mencabut sesi membuat permintaan berikutnya **ditolak seketika** — diuji, bukan menunggu
      masa berlaku habis.
- ☐ Token kedaluwarsa ditolak dengan galat yang **dapat dibedakan** dari token tidak sah.
- ☐ Identitas pengguna tersedia di `context` pada seluruh lapisan tanpa diteruskan sebagai
      parameter berantai — diuji.
- ☐ Masa berlaku sesi dapat dikonfigurasi; **nilai bawaan didokumentasikan beserta alasannya**.
- ☐ Keluar (logout) menghapus sesi dari database, bukan hanya dari peramban.

#### Dependency / Blocked by

Bergantung pada `TKT-F3-001` dan `TKT-F2-001`.

**Terbuka tetapi tidak menahan:** masa berlaku sesi final menunggu Work Owner dan Security
(`ADR-0024`). Nilai sementara dipakai dan ditandai di konfigurasi; mengubahnya tidak menyentuh
kode.

#### Constraint keamanan, data, operasional

- **Aplikasi wajib stateless** (`D-27`): sesi **tidak boleh** hidup di memori satu instans, karena
  load balancer mengarahkan permintaan ke instans mana pun.
- Token **tidak pernah** masuk log, termasuk sebagian isinya.
- Pencabutan sesi harus bekerja **seketika** — itulah alasan sesi disimpan, bukan sekadar token
  yang memverifikasi dirinya sendiri.
- Tabel sesi memuat identitas pengguna; ia tunduk soft delete (`ADR-0012`) dan tercatat di jejak
  audit bila dicabut oleh administrator.

#### Migrasi skema / rollout / rollback

Menambah tabel sesi — tabel baru, **tidak menyentuh tabel yang dibaca Pega**.

**Rollback:** seluruh sesi menjadi tidak sah dan pengguna harus masuk ulang. Itu gangguan yang
dapat diterima, tetapi **harus diumumkan** sebelum rilis mundur dijalankan pada jam kerja.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/sesi/...
go test ./internal/app/sesi/... -run TestSesiLintasInstans
go test ./internal/app/sesi/... -run TestPencabutanSeketika
grep -rIn "token" internal/platform/log/   # HARUS tidak ada penulisan token ke log
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Aplikasi menerbitkan sesi sendiri; alasan masa berlaku, pencabutan, isi token | `D-07` · `docs/Steering/11-SECURITY.md` §2.1 |
| Stateless, dua instans di belakang load balancer | `D-27` |
| Tabel `SessionAktif` untuk pencabutan | `docs/Steering/11-SECURITY.md` §3.3 |
| Masa berlaku sesi masih terbuka | `ADR-0024` Pertanyaan terbuka |

#### Comments

### TKT-F3-004 — Tabel 22 peran dan 51 izin menu

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** — tabel dapat dibangun, **tidak dapat diisi** |
| **Modul** | F-3 · Gelombang: 1 · Bergantung pada: TKT-F2-001, TKT-F2-004 |
| **Requirement** | FR-F3, FR-R1 |
| **Keputusan** | D-58, D-59 |
| **ADR** | 0023 |
| **Risiko** | R-16 |
| **Rule Pega yang digantikan** | **22 access group** `GCNMFW:<nama>` · **34 When rule** yang memuat pemetaan peran→menu · `Navigation/pyCaseWorkerNavigation-Navigation.xml` (51 item menu) · `POOLDATA.T_ACCESS_GROUP_PNC` |
| **Peran penguji gerbang 2** | **tidak berlaku** — `D-60`; gerbang 1 diganti uji kontrak (`D-56`) |
| **Label** | `modul::F-3` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-3-Login-dan-Hak-Akses-Menu/issues/04-tabel-peran-dan-izin-menu.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Peran dan izin menjadi **data di tabel**, bukan logika yang tersebar di 34 When rule.

Nilai bisnisnya: hari ini, menjawab *"siapa saja yang boleh menyetujui komite"* menuntut membaca
34 rule satu per satu. Setelah tiket ini, jawabannya adalah satu kueri — dan itu prasyarat bagi
audit mana pun.

#### Ruang lingkup

- Tabel: Peran (**22 baris**), Izin menu (**51 item**), PeranIzin, PenggunaPeran, dan
  BatasDataPengguna.
- **Normalisasi kapitalisasi**: tiga nama muncul dalam dua bentuk — `ViewClaimPNC`/`VIEWCLAIMPNC`
  dan `PncReceive`/`PNCRECEIVE`. Sistem baru wajib menjadikannya **satu identitas per peran**.
- Penamaan ulang 22 access group menjadi nama yang terbaca manusia, **satu-untuk-satu** — tanpa
  penggabungan dan tanpa pemecahan (`D-58`).
- Batas data: cabang, lini bisnis/Group Panel, dan organisasi (`pyOrgUnit != "Eksternal"`).

#### Non-goal

- **Tidak** membuat izin berbutir aksi. `D-59` menetapkan **satuan izin adalah menu**; membuat
  izin per tindakan berarti merancang pemisahan tugas yang tidak diminta.
- **Tidak** menegakkan izin — itu `TKT-F3-005`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Daftar operator per peran.** `POOLDATA.T_ACCESS_GROUP_PNC` hanya memetakan `OPERATOR_ID` → `OLD_OPERATOR_ID`; penugasan operator ke access group **tidak ada di database** | **Work Owner + DBA** | Tabel dapat dibangun tetapi **tidak dapat diisi**. Aplikasi yang tabel perannya kosong tidak dapat dipakai siapa pun |
| **5 When rule peran hilang dari export**: `IsGCNMReport`, `IsKomite`, `IsNotViewClaim`, `IsPNCBonding`, `IsSurvey` — masing-masing mengendalikan satu item menu | **Tim Pega** (`R-16`) | Peta peran→menu tidak lengkap; 5 dari 51 item menu tidak diketahui siapa yang boleh membukanya |
| **Tidak ada folder Access Group / Role / Privilege / Operator sama sekali** di export | **Tim Pega** | Definisi access group hanya terbaca dari **pemakaiannya** di rule, bukan dari definisinya |
| Apakah pemetaan 22 access group → 51 menu **masih akurat**? · `T_ACCESS_GROUP_PNC` sebenarnya untuk apa? · pernah ada temuan audit soal **901 dari 902 activity tanpa privilege**? | **Work Owner** | Menentukan apakah peta yang direkonstruksi dipakai apa adanya atau perlu ditinjau bisnis |

#### Acceptance criteria

> Butir yang menuntut isi data **belum dapat diangkakan** sampai daftar operator diterima.

- ☐ Tabel peran memuat **tepat 22 baris**, namanya terbaca manusia, dan pemetaannya ke nama
      access group lama tercatat — sehingga migrasi dapat ditelusuri.
- ☐ Ketiga nama yang berbeda kapitalisasi **menjadi satu baris**, bukan dua — diuji.
- ☐ Tabel izin menu memuat **51 item**, masing-masing dengan penanda yang dipakai `TKT-F3-005`.
- ☐ Pemetaan peran→menu yang direkonstruksi dari 34 When rule tersedia sebagai berkas dengan
      `berkas:baris` untuk setiap baris pemetaan.
- ☐ **5 item menu yang When rule-nya hilang ditandai eksplisit** sebagai belum diketahui —
      bukan diisi dengan tebakan.
- ☐ Batas data (cabang, lini bisnis, organisasi) tersimpan sebagai data dan dapat diuji per
      pengguna.

#### Dependency / Blocked by

Bergantung pada `TKT-F2-001`, `TKT-F2-004`. **Terhalang Tim Pega dan Work Owner.**

#### Constraint keamanan, data, operasional

- **Satuan izin adalah menu** (`D-59`). Konsekuensinya diterima sadar: orang yang sama dapat
  membuat, menyetujui, dan membayarkan satu klaim bila perannya memiliki ketiga menu itu.
- Perubahan pada tabel peran dan izin adalah **tindakan bernilai tinggi** — ia mengubah kewenangan.
  Setiap perubahannya wajib tercatat di jejak audit (`S-5`).
- Tabel ini memuat **nama pegawai**; ia tunduk aturan penulisan `D-69` bila dikutip ke dokumen.

#### Migrasi skema / rollout / rollback

Menambah lima tabel baru — **tidak menyentuh tabel yang dibaca Pega**.

**Rollback:** tabel dibiarkan ada dan tidak dipakai. Namun **rollback setelah pengguna nyata
ditugaskan ke peran akan mencabut akses mereka** — karena itu pengisian data dilakukan setelah
tabelnya stabil, bukan bersamaan.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/sqlstore/... -run TestPeranDanIzin
go run ./cmd/tools/cek-peran --harap 22        # HARUS 22 baris
go run ./cmd/tools/cek-menu --harap 51         # HARUS 51 item
go run ./cmd/tools/cek-menu --tanpa-when-rule  # HARUS melaporkan tepat 5
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 22 access group, seluruhnya terverifikasi sebagai literal `GCNMFW:<nama>` | `D-58` · `docs/Steering/11-SECURITY.md` §3.1 |
| 51 item menu dipetakan lewat 34 When rule + `Navigation/pyCaseWorkerNavigation-Navigation.xml` | idem |
| 5 When rule peran hilang | `D-58` · `R-16` |
| Kapitalisasi tidak konsisten pada 3 nama | `D-58` |
| `T_ACCESS_GROUP_PNC` hanya `OPERATOR_ID` → `OLD_OPERATOR_ID` | `D-58` · `ADR-0023` |
| Satuan izin adalah menu, tanpa pemisahan tugas | `D-59` · `ADR-0023` |

#### Comments

### TKT-F3-005 — Middleware otorisasi di setiap endpoint

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap — **inilah bagian `F-3` yang berstatus PENUH** |
| **Modul** | F-3 · Gelombang: 1 · Bergantung pada: TKT-F3-003, TKT-F3-004 |
| **Requirement** | FR-F3, FR-R1 |
| **Keputusan** | D-59 |
| **ADR** | 0023 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | **tidak ada penegakan di server sama sekali** — `pyPrivilegeName` terisi pada **1 dari 902 activity**, dan yang satu itu privilege bawaan Pega untuk ekspor ruleset, bukan aturan bisnis |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`) |
| **Label** | `modul::F-3` `tipe::keamanan` `status::ready-for-human` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-3-Login-dan-Hak-Akses-Menu/issues/05-middleware-otorisasi-per-endpoint.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Setiap endpoint memeriksa kewenangan pemanggil **di server**, dan endpoint yang lupa
mendeklarasikan kewenangannya **ditolak oleh perkakas**, bukan lolos diam-diam.

Nilai bisnisnya adalah menutup celah yang terbukti ada. Hari ini, otorisasi sistem lama hanyalah
**penyembunyian menu** — siapa pun yang mengetahui alamat sebuah endpoint dapat memanggilnya tanpa
pemeriksaan apa pun. Angkanya tegas: **1 dari 902**.

#### Ruang lingkup

- Middleware yang memeriksa: *apakah peran pemanggil memiliki menu yang memberi akses ke endpoint
  ini* (`D-59`).
- **Deklarasi kewenangan wajib pada setiap rute** — rute tanpa deklarasi **menggagalkan uji**,
  bukan berjalan sebagai publik.
- Penegakan **batas data** (cabang, lini bisnis, organisasi) di lapisan kueri, bukan dengan
  menyaring hasil setelah data terambil.
- Daftar rute publik yang eksplisit dan pendek: health check dan login.

#### Non-goal

- **Tidak** merancang pemisahan tugas. `D-59` menetapkan **tidak ada** pemisahan tugas formal.
- **Tidak** mengisi tabel peran — itu `TKT-F3-004`.

#### Acceptance criteria

- ☐ Permintaan tanpa sesi ke endpoint non-publik mengembalikan **`401`**; permintaan dengan sesi
      tetapi tanpa kewenangan mengembalikan **`403`** — keduanya diuji dan **dapat dibedakan**.
- ☐ Rute baru yang **tidak** mendeklarasikan kewenangan **menggagalkan uji** — diuji dengan rute
      percobaan; ini pemeriksaan yang mencegah endpoint lolos karena lupa.
- ☐ Daftar rute publik berisi **tepat** health check dan login — diuji; penambahan apa pun
      menggagalkan uji sampai daftar diperbarui secara sadar.
- ☐ Batas data ditegakkan **di kueri**: pengguna cabang A yang meminta daftar klaim tidak
      menerima klaim cabang B, dan **jumlah total yang dilaporkan juga tidak memuatnya** — diuji,
      karena menyaring setelah pengambilan membocorkan lewat jumlah baris.
- ☐ Pengguna `pyOrgUnit = "Eksternal"` dibatasi sesuai aturan sistem lama — diuji.
- ☐ Akses **data medis** hanya untuk peran Analyst Doctor dan RCL Dokter (`FR-R2`) — diuji, dan
      berlaku **juga di staging** karena staging memuat data nyata (`ADR-0029`).
- ☐ Setiap penolakan otorisasi tercatat di log dengan peran dan endpoint — **tanpa** data
      nasabah.

#### Dependency / Blocked by

Bergantung pada `TKT-F3-003` dan `TKT-F3-004`.

**Catatan urutan:** middleware dapat dibangun dan diuji dengan tabel peran yang **masih kosong**,
memakai data uji. Yang menunggu daftar operator nyata adalah **pengisiannya**, bukan mekanismenya.

#### Constraint keamanan, data, operasional

- **Kriteria penerimaan `BRD §21.2` #8** — otorisasi diperiksa di setiap endpoint — dipenuhi oleh
  tiket ini, dengan bacaan `D-59`: yang diperiksa adalah kepemilikan menu, bukan tindakan.
- Konsekuensi `D-59` diterima sadar: **satu orang dapat membuat, menyetujui, dan membayarkan satu
  klaim** bila perannya memiliki ketiga menu. Kontrol pengimbangnya adalah jejak audit `S-5`.
- **`AutoAcceptKomite` melewati kontrol ini sepenuhnya** — job terjadwal harian jam 06:00
  menyetujui komite tanpa pengguna sama sekali (`ADR-0022`). Middleware tidak berlaku pada jalur
  job, dan itu harus disadari saat `S-6` dirancang.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** menonaktifkan middleware **tidak tersedia sebagai rollback** —
itu membuka seluruh endpoint. Bila terjadi masalah, yang dilakukan adalah memperbaiki deklarasi
kewenangan rute yang salah, bukan mematikan pemeriksaannya.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/http/... -run TestOtorisasi
go test ./internal/adapter/http/... -run TestRuteTanpaDeklarasiGagal
go test ./internal/adapter/http/... -run TestBatasDataCabang
go run ./cmd/tools/daftar-rute --publik    # HARUS tepat 2
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `pyPrivilegeName` terisi 1 dari 902 activity | `T-10` · `docs/verifikasi-bukti-adr.md` §10.3 |
| Satuan izin adalah menu; penegakan pindah ke server | `D-59` · `ADR-0023` |
| Batas data: cabang, lini bisnis, organisasi | `docs/Steering/11-SECURITY.md` §3.2 |
| Batas data ditegakkan di kueri, bukan disaring setelahnya | idem |
| Akses data medis dibatasi dua peran | `FR-R2` · `BRD §17.2` |
| `AutoAcceptKomite` melewati kontrol menu | `D-57` · `ADR-0022` |

#### Comments

# 5. F-4 — Master Data

*Modul Fondasi · folder `docs/ticketing/F-4-Master-Data-dan-Parameter-Bisnis/`*

## Spesifikasi Modul

| | |
|---|---|
| **Modul** | `F-4` Master Data |
| **Gelombang** | 1 — Fondasi |
| **Ukuran** | **Besar** (naik dari Sedang–Besar) |
| **Bergantung pada** | `F-1`, `F-2`, `F-3` |
| **Kesiapan** | **SEBAGIAN** — kerangka siap, empat master terhalang isi atau keputusan |
| **Cakupan tiket** | penuh untuk bagian yang buktinya lengkap; `needs-info` untuk bagian terhalang (`D-41` Opsi 1) |

### Apa yang dibangun

Seluruh nilai yang di sistem lama tertanam di dalam rule, dipindahkan menjadi **data yang dapat
diubah tanpa deploy**, beserta layar pengelolanya.

### Ukuran sebenarnya

| Anggapan v1.0 | Terverifikasi |
|---|---|
| 14 kelompok master | **≥29 kelompok** |
| 10 alamat email hardcode | **66 unik**, termasuk **≥6 akun Gmail pribadi di jalur produksi** |
| 4 user ID hardcode | **24 unik**, sebagian tertanam **di dalam teks SQL** |
| 3 ambang komite hardcode | **8 ambang komite unik** + 7 ambang uang non-komite |
| 3 hostname | **3 unik, 48 perbandingan** — angkanya benar, daftarnya salah |

Angka lama benar untuk lingkupnya — dua rule saja. Memakainya untuk `FR-F4` akan membuat estimasi
meleset sekitar **enam kali lipat**.

### Yang mengikat modul ini

| Sumber | Isi |
|---|---|
| `ADR-0025` | Seluruh hardcode menjadi master/konfigurasi; rahasia keluar dari kode — **`Proposed`, `D-40` masih OPEN** |
| `ADR-0014` | Ambang komite: tangga ambang bawah per lini, diakumulasi; `LIMIT_TOP` jadi validasi master |
| `ADR-0015` | Kurs pada tanggal kejadian; tanpa kurs klaim **ditolak** |
| `D-67` | **Tidak ada akun pribadi** sebagai penerima notifikasi |
| `D-62` | Retensi sebagai master |

### Penghalang modul ini

| Jenis | Isi | Pemilik | Menahan |
|---|---|---|---|
| **Artefak** | isi **`POOLDATA.GCNM_FEE_SCALE`** (17 pita) | DBA | `B-5`, bukan `F-4` langsung |
| **Artefak** | isi **`m_currencystandard`** | DBA | `TKT-F4-004` |
| **Artefak** | **DDL 9 tabel baru** | DBA | `TKT-F4-001` sebagian |
| **Keputusan** | **Mana dari ≥29 kelompok master yang masuk lingkup?** Baru area Bengkel/Sparepart/Supplier yang diputuskan (dikecualikan, `D-34`) | Work Owner | `TKT-F4-006` |
| **Keputusan** | Struktur `EMAILKOMITE` dipertahankan apa adanya? | Work Owner | `TKT-F4-002` |
| **Keputusan** | Ambang Large Losses ada fiturnya di sistem baru? · batas tanggal per lini bisnis = kebutuhan baru? · alur persetujuan perubahan master berlaku untuk semua master? | Work Owner | `TKT-F4-001`, `TKT-F4-005` |

### Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-F4-001](issues/01-kerangka-master-data-dan-layar-crud.md) | Kerangka master data dan layar CRUD baku | `needs-info` | terhalang keputusan |
| [TKT-F4-002](issues/02-master-ambang-komite.md) | Master Ambang Komite dan validasi integritas tangga | `needs-info` | terhalang keputusan |
| [TKT-F4-003](issues/03-master-penerima-notifikasi.md) | Master Penerima Notifikasi — mailbox fungsional | `ready-for-human` | siap |
| [TKT-F4-004](issues/04-master-mata-uang-dan-kurs.md) | Master Mata Uang dan Kurs per tanggal | `needs-info` | **terhalang artefak** |
| [TKT-F4-005](issues/05-master-status-kalender-dan-ambang.md) | Master Status Klaim, kalender libur, dan ambang uang | `needs-info` | terhalang keputusan |
| [TKT-F4-006](issues/06-inventaris-kelompok-master.md) | Inventaris ≥29 kelompok master dan penetapan lingkup | `needs-info` | terhalang keputusan |

## Daftar Tiket

### TKT-F4-001 — Kerangka master data dan layar CRUD baku

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan** — alur persetujuan perubahan master belum ditetapkan |
| **Modul** | F-4 · Gelombang: 1 · Bergantung pada: TKT-F2-001, TKT-F3-005, TKT-U2-001 |
| **Requirement** | FR-F4 |
| **Keputusan** | D-15, D-59 |
| **ADR** | 0023, 0025, 0026 |
| **Risiko** | R-08 |
| **Rule Pega yang digantikan** | nilai yang tertanam di dalam activity — **66 email**, **24 Operator ID**, **8 ambang komite**, dan 7 ambang uang non-komite; ditambah layar master yang tersebar di `Section/` |
| **Peran penguji gerbang 2** | **User Admin** dan peran pemilik masing-masing master (`D-60`) |
| **Label** | `modul::F-4` `tipe::fondasi` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-4-Master-Data-dan-Parameter-Bisnis/issues/01-kerangka-master-data-dan-layar-crud.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu pola untuk seluruh master: satu tabel, satu layar daftar, satu layar ubah, satu jejak audit —
sehingga menambah master ke-30 tidak berarti menulis ulang semuanya.

Nilai bisnisnya: perubahan kebijakan bisnis — ambang, penerima notifikasi, kode status — menjadi
**pekerjaan pengguna bisnis**, bukan permintaan deployment. Itulah alasan nilai-nilai itu
di-hardcode sejak awal di sistem lama: karena mengubahnya menuntut orang teknis.

#### Ruang lingkup

- Pola generik master: definisi kolom, validasi, dan layar CRUD yang dihasilkan dari definisi itu.
- **Setiap perubahan master tercatat di jejak audit** (`S-5`) — siapa, kapan, nilai sebelum,
  nilai sesudah.
- Penegakan izin lewat `TKT-F3-005`: master yang berbeda dapat dimiliki peran yang berbeda.
- Soft delete pada seluruh master (`ADR-0012`) — master yang dinonaktifkan tetap dapat dirujuk
  data lama.

#### Non-goal

- **Tidak** mengisi master mana pun — masing-masing punya tiketnya sendiri.
- **Tidak** menetapkan daftar ≥29 kelompok — itu `TKT-F4-006`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Apakah perubahan master butuh alur persetujuan, dan berlaku untuk master yang mana?** | **Work Owner** | Menentukan apakah pola generiknya memuat status "menunggu persetujuan" atau tidak. Menambahkannya belakangan menyentuh seluruh master |
| **DDL 9 tabel baru** (`R-08`) | **DBA** | Menahan bagian yang menyentuh tabel nyata; pola generiknya sendiri tidak terhalang |

**Yang perlu disadari saat memutuskan alur persetujuan.** `D-59` menghapus pemisahan tugas —
seorang pengguna yang memiliki menu master ambang komite dapat **mengubah kewenangan persetujuan
uang sendirian**. Bila alur persetujuan hendak dipasang di suatu tempat, master inilah kandidat
paling kuat.

#### Acceptance criteria

- ☐ Menambah master baru menuntut **≤ 50 baris** definisi, tanpa menulis layar baru — dibuktikan
      dengan satu master contoh.
- ☐ Setiap perubahan master menghasilkan **tepat satu baris jejak audit** dengan nilai sebelum
      dan sesudah — diuji.
- ☐ Penghapusan master adalah **soft delete**; data lama yang merujuknya **tetap dapat
      ditampilkan** — diuji.
- ☐ Pengguna tanpa izin menu master menerima **`403`** dan **tidak melihat menunya** — keduanya
      diuji.
- ☐ Validasi per master dideklarasikan bersama definisinya, dan galatnya muncul di field yang
      benar (`TKT-U2-002`).
- ☐ Daftar master memakai komponen tabel baku `TKT-U2-001` — tidak ada tabel yang ditulis
      khusus.

#### Dependency / Blocked by

Bergantung pada `TKT-F2-001`, `TKT-F3-005`, `TKT-U2-001`, dan `TKT-S5-002` untuk pencatatan audit.
**Terhalang keputusan alur persetujuan.**

#### Constraint keamanan, data, operasional

- Master menentukan **perilaku bisnis** — ambang komite menentukan siapa berwenang menyetujui uang.
  Perubahannya adalah tindakan bernilai tinggi dan **wajib tercatat**.
- Master **tidak boleh** memuat rahasia. Kredensial dan kunci API bukan master data; tempatnya
  menunggu `D-40`.
- Nama pegawai di master tunduk aturan penulisan `D-69` bila dikutip ke dokumen.

#### Migrasi skema / rollout / rollback

Menambah tabel master baru — **tidak menyentuh tabel yang dibaca Pega**. Selama masa paralel,
**Pega tetap membaca nilai hardcode di rule-nya sendiri**; master baru hanya dipakai aplikasi Go.
Itu berarti dua sumber kebenaran sementara, dan **itu disengaja** — menyatukannya baru terjadi saat
modul yang memakainya pindah.

**Rollback:** tabel dibiarkan; aplikasi kembali memakai nilai sebelumnya.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/masterdata/...
go test ./internal/app/masterdata/... -run TestPerubahanTercatatDiAudit
go test ./internal/adapter/http/... -run TestIzinMenuMaster
npm run test -- MasterCrud
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 66 email · 24 Operator ID · 8 ambang komite · 3 hostname | `D-15` terverifikasi · `docs/Steering/12-CROSSCUTTING.md` §3.3 |
| ≥29 kelompok master | `T-13` · `docs/Steering/05-DOMAIN-MODEL.md` §5 |
| Perubahan bernilai bisnis wajib tercatat | `D-28` · `ADR-0026` · `BRD §21.2` #9 |
| Tidak ada pemisahan tugas | `D-59` · `ADR-0023` |
| DDL 9 tabel baru belum ada | `R-08` · `docs/verifikasi-bukti-adr.md` §15 baris `F-4` |

#### Comments

### TKT-F4-002 — Master Ambang Komite dan validasi integritas tangga

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan** — struktur pengganti `EMAILKOMITE` belum ditetapkan |
| **Modul** | F-4 · Gelombang: 1 · Bergantung pada: TKT-F4-001 |
| **Requirement** | FR-F4 |
| **Keputusan** | D-14, D-47, D-52, D-70 |
| **ADR** | 0014 |
| **Risiko** | R-19 |
| **Rule Pega yang digantikan** | `POOLDATA.EMAILKOMITE` (**21 kolom, 30 baris**) dan logika pemilihan yang tersebar di `Activity/SetEmailKomite-Act.xml`, `SetEmailKomiteAdjuster-Act.xml`, `SetEmailKomiteSalvage-Act.xml`, `SetEmailKomiteSimasnet-Act.xml` — ditambah **17 kueri** yang membacanya |
| **Peran penguji gerbang 2** | **PNCKomite** dan **PNCKomiteTeknik** (`D-60`) |
| **Label** | `modul::F-4` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-4-Master-Data-dan-Parameter-Bisnis/issues/02-master-ambang-komite.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Tangga jenjang komite menjadi **master data yang dapat divalidasi**, menggantikan aturan yang
tersebar di empat activity dan bercampur dengan **nama orang**.

Nilai bisnisnya: master ini menentukan **siapa berwenang menyetujui uang**. Satu baris yang salah
mengubah kewenangan persetujuan **tanpa perubahan kode apa pun** — itulah sebabnya validasi
integritasnya menjadi bagian tiket ini, bukan tambahan.

#### Ruang lingkup

- Master ambang komite berisi, per lini bisnis: **ambang bawah**, urutan jenjang (`DEGREE`),
  penanda aktif, penanda jenis komite, dan penyetuju.
- **Validasi integritas tangga** memakai batas atas: menolak master yang rentangnya **tumpang
  tindih** atau **berlubang** antar `DEGREE` dalam satu lini + pita.
- Penghapusan aturan berbasis **nama orang** — sistem lama memaksa nilai pembanding melewati ambang
  agar jenjang tertentu ikut atau tidak ikut terpilih, dengan syarat siapa PIC Teknis-nya.
- Migrasi isi `emailkomite.csv` yang sudah diterima ke bentuk master baru.

#### Non-goal

- **Tidak** membangun alur komite — itu `B-7`.
- **Tidak** mengubah model penjenjangan. Model **kumulatif** sudah ditetapkan `ADR-0014`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Apakah struktur `EMAILKOMITE` dipertahankan apa adanya**, atau dipecah? Kolom `TYPE_KOMITE` **memikul dua arti berbeda** — pita nilai di Non-MBU, varian jalur (PA reguler versus PA TKI) di lini PA | **Work Owner** | Menentukan bentuk tabel. `ADR-0014` memutuskan **tidak** memecahnya sekarang, tetapi keputusan itu diambil untuk perilaku, bukan untuk bentuk master baru |
| **PA dan Travel di atas Rp 200.000.000 tidak punya baris master** | **Work Owner** | Klaim di atas nilai itu tidak punya penyetuju — dan itu bukan sesuatu yang boleh ditebak |

**Yang sudah tidak perlu ditanyakan lagi:** aturan berbasis nama orang **dicabut** (`D-52`), dan
model kumulatif beserta cakupan pita per lini sudah ditetapkan (`D-47`, `D-70`).

#### Acceptance criteria

- ☐ Master menolak disimpan bila ada **lubang** antar `DEGREE` dalam satu lini + pita — diuji
      dengan master percobaan yang sengaja berlubang.
- ☐ Master menolak disimpan bila ada **tumpang tindih** rentang — diuji.
- ☐ Validasi dijalankan **per lini**, bukan lintas lini — diuji dengan PA, yang tangganya memang
      tidak tersusun menurut pita.
- ☐ Jumlah jenjang untuk sebuah nilai klaim dihitung **kumulatif** dan hasilnya sama dengan
      tabel berikut — diuji ketujuh kasus:

      | Kasus | Jumlah penyetuju |
      |---|---|
      | PA Rp 5.000.000 | 1 |
      | PA Rp 75.000.000 | 3 |
      | PA Rp 150.000.000 | 4 |
      | Travel Rp 150.000.000 | 3 |
      | Non-MBU Rp 80.000.000 | 2 |
      | Non-MBU Rp 750.000.000 | 2 |
      | Non-MBU Rp 2.000.000.000 | 3 |

- ☐ **Nol aturan berbasis nama orang** di kode maupun master — diuji pemindaian terhadap 24
      Operator ID yang diketahui.
- ☐ Perubahan master tercatat di jejak audit (`TKT-F4-001`).

#### Dependency / Blocked by

Bergantung pada `TKT-F4-001`. **Terhalang dua keputusan** di atas.

**Yang bergantung padanya:** `B-7` Komite dan `B-12` Salvage.

#### Constraint keamanan, data, operasional

- **Perubahan master ini mengubah kewenangan menyetujui uang.** Ia kandidat terkuat untuk alur
  persetujuan bila `TKT-F4-001` memutuskan memakainya.
- Master memuat alamat email penyetuju — tunduk `D-67` (mailbox fungsional, bukan akun pribadi)
  dan `D-69` (email disamarkan bila dikutip ke dokumen).
- **Belum terverifikasi:** tiga kueri yang memfilter `TYPE_KOMITE` secara dinamis —
  `EmailKomiteBerjenjang_sql`, `EmailKomiteAdjuster_sql`, `EmailKomiteSalvage_sql` — menerima
  nilainya dari pemanggil lewat `tempAdj.pyMemo` dan `tempAdj.AcceptedNo`. **Lini apa saja yang
  melewati ketiga kueri itu belum ditelusuri sampai ke sumber nilainya.** Ini bagian Definition of
  Ready `B-7` dan `B-12`.

#### Migrasi skema / rollout / rollback

Menambah tabel master baru; **`POOLDATA.EMAILKOMITE` tetap dibaca Pega** selama masa paralel dan
**tidak boleh diubah** (`P-1` — Pega adalah penulisnya sampai `B-7` pindah).

**Rollback:** aplikasi Go kembali membaca `EMAILKOMITE` langsung. Master baru dibiarkan.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/komite/... -run TestJumlahJenjangKumulatif
go test ./internal/app/masterdata/... -run TestValidasiTanggaBerlubang
go test ./internal/app/masterdata/... -run TestValidasiTanggaTumpangTindih
go run ./cmd/tools/cek-hardcode-operator internal/   # HARUS 0 temuan
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `EMAILKOMITE` 21 kolom, 30 baris | `D-14` · `Database/emailkomite.csv` |
| Model kumulatif; `LIMIT_BOTTOM` di 11 kueri, `LIMIT_TOP` di 0 | `D-47` · `ADR-0014` |
| Pita nilai hanya di Non-MBU | `D-70` |
| Aturan berbasis nama orang dicabut | `D-52` · `docs/Steering/20-DETAIL-KOMITE-DBLINK.md` §1.5 |
| PA dan Travel di atas Rp 200 Jt tanpa baris master | `D-55` · `ADR-0014` |
| Tujuh kasus uji jumlah penyetuju | `ADR-0014` Rationale |

#### Comments

### TKT-F4-003 — Master Penerima Notifikasi (mailbox fungsional)

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | F-4 · Gelombang: 1 · Bergantung pada: TKT-F4-001 |
| **Requirement** | FR-F4 |
| **Keputusan** | D-15, D-67 |
| **ADR** | 0025 |
| **Risiko** | R-17 |
| **Rule Pega yang digantikan** | **66 alamat email unik** yang tertanam di activity — di antaranya `Activity/InputRegister_act-Act.xml:16154` (7 email pimpinan dalam satu string), `:16297` (UW FIRE), `:16461` (UW ANEKA), `:16628` (UW MARINE); akun Gmail pribadi di `Activity/AutoEmailDownloadProposeAdjustment-Act.xml:7872`, `:8031`, `Activity/SetStsSalvagePNC_act-Act.xml:694`, `Activity/LetterOfAssignment2_Act-Act.xml:7725`, `Activity/SendEmailUnprotectPremi-Act.xml:4823` |
| **Peran penguji gerbang 2** | **User Admin** dan **PncManagerAdmin** (`D-60`) |
| **Label** | `modul::F-4` `tipe::migrasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-4-Master-Data-dan-Parameter-Bisnis/issues/03-master-penerima-notifikasi.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Penerima notifikasi menjadi master data berupa **mailbox fungsional**, dan **tidak ada satu pun
akun pribadi** yang dibawa ke sistem baru.

Nilai bisnisnya langsung: hari ini, notifikasi klaim dikirim ke **≥6 alamat Gmail pribadi di jalur
produksi**. Ketika orangnya pindah atau keluar, notifikasi itu **tetap terkirim ke sana** dan tidak
ada yang menyadarinya sampai sesuatu tidak ditindaklanjuti.

#### Ruang lingkup

- Master penerima: per **jenis peristiwa** dan per **Group Panel**, berisi alamat tujuan dan
  tembusan.
- Pemetaan 66 alamat lama ke mailbox fungsional pengganti — dikerjakan bersama Work Owner saat
  master diisi.
- **Pembersihan kategori khusus**: lima alamat Gmail yang dipakai sebagai **Operator ID** di filter
  laporan KPI — itu masalah **identitas**, bukan notifikasi, dan menyentuh `F-3` sekaligus `F-4`.
- Penghapusan blok `// TESTING` yang menimpa email produksi.

#### Non-goal

- **Tidak** membangun pengirim notifikasi — itu `S-3`.
- **Tidak** memutuskan isi pesan notifikasi.
- **Tidak** memindahkan kredensial SMTP — itu `D-40`, masih `OPEN`.

#### Acceptance criteria

- ☐ Master memuat penerima per jenis peristiwa dan per Group Panel, dan **seluruh 66 alamat
      lama terpetakan** — dihitung dan dilaporkan angkanya; alamat yang sengaja tidak dibawa
      dicatat alasannya.
- ☐ **Nol alamat email tertanam di kode** — diuji pemindaian pola; build gagal bila muncul.
- ☐ **Nol akun pribadi** di master — diuji terhadap daftar domain yang diizinkan; alamat di luar
      domain korporat ditolak saat disimpan.
- ☐ Lima alamat yang dipakai sebagai Operator ID **tidak** masuk master penerima; ia dilaporkan
      sebagai temuan untuk `F-3` — diuji bahwa keduanya tidak tertukar.
- ☐ Mengubah penerima **tidak menuntut deployment** — diuji dengan mengubah master dan memeriksa
      pengiriman berikutnya memakai alamat baru.
- ☐ Perubahan master tercatat di jejak audit (`TKT-F4-001`).

#### Dependency / Blocked by

Bergantung pada `TKT-F4-001`. Pengisian alamat pengganti dikerjakan bersama Work Owner — **itu
bagian pelaksanaan, bukan penghalang**.

#### Constraint keamanan, data, operasional

- **Alamat email tidak pernah ditulis lengkap di dokumen yang di-commit** (`D-69`) — di tiket ini
  pun alamatnya dirujuk dengan `berkas:baris`, bukan nilainya.
- Blok `// TESTING` pada `Activity/InputRegister_act-Act.xml:16693`, `:16830`, `:16998`, `:17141`
  **menimpa email produksi dengan precondition yang identik** — ia tidak boleh punya padanan apa
  pun di sistem baru.
- Kredensial SMTP **tidak** disimpan di master ini; tempatnya menunggu `D-40` (`R-17`).

#### Migrasi skema / rollout / rollback

Menambah tabel master — tidak menyentuh tabel Pega. Selama masa paralel, **Pega tetap memakai
alamat hardcode-nya sendiri**; keduanya hidup berdampingan sampai `S-3` pindah.

**Rollback:** aplikasi Go kembali ke daftar bawaan; **tidak kembali ke akun pribadi** — daftar
bawaan yang dipakai adalah mailbox fungsional.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
grep -rInE "[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}" internal/ --include=*.go   # HARUS 0
go test ./internal/app/masterdata/... -run TestPenerimaNotifikasi
go test ./internal/app/masterdata/... -run TestTolakDomainDiLuarKorporat
go run ./cmd/tools/banding-penerima daftar-lama.csv   # laporkan yang belum terpetakan
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 66 alamat email unik | `D-15` terverifikasi · `docs/verifikasi-bukti-adr.md` §7.2 |
| ≥6 akun Gmail pribadi di jalur produksi | `D-67` · lokasi tercantum di header tiket |
| 5 alamat dipakai sebagai Operator ID di filter KPI | `D-67` — menyentuh `F-3` sekaligus `F-4` |
| Blok `// TESTING` menimpa email produksi | `Activity/InputRegister_act-Act.xml:16693`, `:16830`, `:16998`, `:17141` |
| Seluruh penerima dari master, mailbox fungsional | `D-67` |

#### Comments

### TKT-F4-004 — Master Mata Uang dan Kurs per tanggal

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** — isi `m_currencystandard` belum ada |
| **Modul** | F-4 · Gelombang: 1 · Bergantung pada: TKT-F4-001, TKT-F5-001 |
| **Requirement** | FR-F4 |
| **Keputusan** | D-48, D-49 butir 4 dan 5 |
| **ADR** | 0015, 0017 |
| **Risiko** | R-19 |
| **Rule Pega yang digantikan** | `Database/GETCURRENCYSTANDARD.fnc` — memakai **kurs hari eksekusi** (`:3` versus `:14`) dan mengembalikan **`1`** ketika kurs tidak ditemukan (`:20-22`) |
| **Peran penguji gerbang 2** | **User Admin** dan **PncPICTeknik** (`D-60`) |
| **Label** | `modul::F-4` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-4-Master-Data-dan-Parameter-Bisnis/issues/04-master-mata-uang-dan-kurs.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Kurs menjadi master **per tanggal**, dan konversi memakai **kurs pada tanggal kejadian** — bukan
kurs hari klaim diproses.

Nilai bisnisnya adalah menutup cacat yang **membuat klaim besar lolos tanpa komite**. Fungsi kurs
sistem lama mengembalikan `1` ketika kurs tidak ditemukan: satu satuan valuta asing dihitung setara
satu Rupiah, sehingga klaim bernilai besar menyusut menjadi kecil, lalu **lolos di bawah seluruh
ambang komite**. Tidak ada galat, tidak ada catatan, dan angkanya tampak wajar.

#### Ruang lingkup

- Master mata uang dan **kurs per tanggal**.
- Fungsi konversi yang memakai **kurs pada tanggal kejadian** (`D-48`).
- Perilaku saat kurs tidak ditemukan: **menolak** dengan galat yang menyebutkan **mata uang dan
  tanggalnya** — tidak ada nilai bawaan dan tidak ada kurs pengganti.
- Layar pengelolaan kurs, termasuk pengisian mundur untuk tanggal yang terlewat.

#### Non-goal

- **Tidak** mengotomatiskan pengambilan kurs dari sumber luar — sumbernya belum ditetapkan.
- **Tidak** memperbaiki nilai historis yang telanjur dihitung dengan kurs `1`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Isi tabel `m_currencystandard`** | **DBA** (`R-19`) | Tanpa isinya, `B-5` tidak dapat diuji sama sekali — dan master ini tidak dapat diisi |
| **Siapa yang mengisi kurs harian dan dari sumber apa** — Bank Indonesia, kurs korporat, atau lain | **Work Owner** | Menentukan apakah layar pengisian manual cukup, atau perlu jalur pengambilan otomatis |
| **Bagaimana klaim yang tertolak karena kurs kosong diperlakukan** — ditahan dan diproses ulang otomatis, atau dikembalikan ke petugas? | **Work Owner** | Menentukan perilaku operasional yang terlihat pengguna setiap hari |

#### Acceptance criteria

- ☐ Konversi memakai kurs pada **tanggal kejadian**, bukan tanggal proses — diuji dengan klaim
      yang tanggal kejadiannya berbeda dari hari pengujian.
- ☐ Kurs tidak ditemukan menghasilkan **galat yang menyebut mata uang dan tanggal**, dan
      transaksi **ditolak** — diuji; **nilai `1` tidak boleh muncul di jalur mana pun**.
- ☐ Nilai Rupiah sebuah klaim **tidak berubah** ketika klaim yang sama diproses ulang di hari
      berbeda — diuji.
- ☐ Kurs dapat diisi mundur untuk tanggal yang terlewat, dan pengisiannya tercatat di jejak
      audit (`TKT-F4-001`).
- ☐ Perbandingan terhadap ambang komite memakai **nilai presisi penuh** hasil konversi
      (`ADR-0016`) — diuji.

#### Dependency / Blocked by

Bergantung pada `TKT-F4-001` dan `TKT-F5-001`. **Terhalang DBA dan dua keputusan Work Owner.**

**Yang bergantung padanya:** `B-5` Settlement — satu-satunya modul yang **masih terikat
`BRD §21.4`**, dan salah satu dari tiga penghalangnya adalah isi tabel ini.

#### Constraint keamanan, data, operasional

- **Dampak operasional yang harus disiapkan sebelum rilis:** klaim valuta asing yang dulu lolos
  kini **dapat ditolak** sampai kursnya dilengkapi. Ini perbaikan yang diinginkan, tetapi ia
  menghentikan pekerjaan petugas bila master kurs tidak terisi tepat waktu.
- **Uji kesetaraan akan menampilkan selisih pada seluruh data historis valuta asing**, karena
  laporan lama memakai kurs hari eksekusi. Selisih ini **sudah terdaftar** sebagai butir 8 dan 9
  pada 13 perbaikan eksplisit `P-5`, sehingga lolos otomatis di gerbang 1 (`D-54`).
- Master kurs menjadi **data kritis**: kelengkapannya per tanggal menentukan apakah klaim dapat
  diproses sama sekali.

#### Migrasi skema / rollout / rollback

Menambah tabel master kurs — tidak menyentuh tabel Pega.

**Rollback:** kembali ke `GETCURRENCYSTANDARD` **tidak tersedia** — mengembalikannya berarti
memulihkan cacat `RETURN 1`. Bila master kurs bermasalah, yang dilakukan adalah melengkapi
datanya, bukan mengembalikan fungsi lama.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/kurs/... -run TestKursTanggalKejadian
go test ./internal/domain/kurs/... -run TestKursTidakDitemukanMenolak
grep -rIn "return 1" internal/domain/kurs/     # HARUS 0 baris
go test ./internal/domain/komite/... -run TestAmbangMemakaiPresisiPenuh
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Kurs pada tanggal kejadian; tolak bila tidak ada | `D-48` · `ADR-0015` |
| `GETCURRENCYSTANDARD` memakai kurs hari eksekusi dan `RETURN 1` | `Database/GETCURRENCYSTANDARD.fnc:3`, `:14`, `:20-22` |
| Butir 4 dan 5 dari 13 perbaikan eksplisit `P-5` | `D-49` · `ADR-0017` |
| Isi `m_currencystandard` belum ada | `R-19` · `D-55` |
| `B-5` masih terikat `BRD §21.4` | `D-55` · `ADR-0028` |

#### Comments

### TKT-F4-005 — Master Status Klaim, kalender libur, dan ambang uang

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan** — dua pertanyaan lingkup belum dijawab |
| **Modul** | F-4 · Gelombang: 1 · Bergantung pada: TKT-F4-001 |
| **Requirement** | FR-F4 |
| **Keputusan** | D-15, D-18, D-50, D-62 |
| **ADR** | 0018, 0020, 0025, 0026 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | `V_STS_CLAIM` (**33 kode `1134`–`1166`**) · `HRD_LBR` lewat DB Link · ambang uang non-komite yang tertanam — `Activity/InputRegister_act-Act.xml:19110` (Notice of Large Losses `1000000000`), `Activity/CheckEstimateValue-Act.xml:9617`, `:9831` (`50000000000`), `Activity/SetListComiteeClaimAI-Act.xml:11602`, `:11742` |
| **Peran penguji gerbang 2** | **User Admin** dan **PncManagerAdmin** (`D-60`) |
| **Label** | `modul::F-4` `tipe::migrasi` `status::needs-info` `prioritas::sedang` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-4-Master-Data-dan-Parameter-Bisnis/issues/05-master-status-kalender-dan-ambang.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Tiga master yang sifatnya sama — daftar nilai referensi yang menentukan perilaku — dikerjakan
dalam satu tiket karena polanya identik: status klaim, kalender hari libur, dan ambang uang
non-komite.

Nilai bisnisnya: **33 kode status** kini diketahui artinya, dan kalender libur menentukan angka
TAT yang dilaporkan ke manajemen. Keduanya hari ini hidup di tempat yang tidak dapat diubah
pengguna bisnis.

#### Ruang lingkup

- **Master Status Klaim**: 33 kode `1134`–`1166` beserta labelnya, termasuk penomoran lama `01`–`11`
  yang melekat pada sebelas kode pertama.
- **Master Hari Libur dan Jam Kerja**: menggantikan `HRD_LBR` dan definisi jam kerja yang kini
  berada di dalam fungsi remote (`ADR-0020`).
- **Master Ambang Uang non-komite**: Notice of Large Losses (`1000000000`) dan ambang lain yang
  tertanam.
- **Master Retensi**: lama simpan data klaim dan jejak audit — satu kebijakan untuk keduanya
  (`D-62`), nilainya sebagai parameter sampai angkanya ada.

#### Non-goal

- **Tidak** mengubah arti kode status. `D-18` menetapkan keempat konsep status dipertahankan.
- **Tidak** membangun perhitungan TAT — itu `TKT-F5-003` dan `S-7`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Ambang Large Losses ada fiturnya di sistem baru?** Di sistem lama ia memicu Notice of Large Losses ke Underwriting dan jajaran pimpinan | **Work Owner** | Bila tidak dibawa, master ambang ini tidak dibuat sama sekali |
| **Batas aturan tanggal per lini bisnis (7/30/90 hari) — kebutuhan baru atau sudah ada?** | **Work Owner** | Menentukan apakah ia master atau aturan tetap di kode |
| **Dari mana daftar hari libur diperoleh setiap tahun, dan siapa mengisinya?** | **Work Owner** | Sama dengan `TKT-F5-003` — satu jawaban menutup keduanya |
| **Angka retensi data klaim yang berlaku sekarang** | **Work Owner + Compliance** | Tidak menahan pembangunan (dibuat parameter), **menahan go-live** |

#### Acceptance criteria

- ☐ Master status memuat **tepat 33 kode** `1134`–`1166` beserta labelnya — dihitung dan
      dilaporkan angkanya.
- ☐ Sebelas kode pertama (`1134`–`1144`) menyimpan **penomoran lama `01`–`11`** sebagai kolom
      terpisah, sehingga data historis tetap terbaca.
- ☐ Kode status **tidak tertanam di kode** — diuji pemindaian terhadap keempat digit yang
      diketahui.
- ☐ Master hari libur dapat diisi per tahun, dan mengubahnya **mengubah hasil perhitungan jam
      kerja** — diuji bersama `TKT-F5-003`.
- ☐ Ambang uang non-komite dibaca dari master — diuji dengan mengubah ambang Large Losses dan
      memeriksa pemicunya berubah.
- ☐ Retensi tersedia sebagai parameter dengan **nilai bawaan kosong** yang berarti tidak ada
      yang diarsipkan (`TKT-S5-004`).

#### Dependency / Blocked by

Bergantung pada `TKT-F4-001`. **Terhalang empat keputusan** di atas — tetapi bagian **Master Status
Klaim** tidak terhalang dan boleh dikerjakan lebih dulu.

#### Constraint keamanan, data, operasional

- Kode status menentukan perilaku layar dan laporan; perubahannya **wajib tercatat** di jejak
  audit (`TKT-F4-001`).
- **Arti kode status tidak boleh disimpulkan dari pemakaian.** Tiga arti yang sempat disimpulkan
  dari rule **seluruhnya salah**: `1143` bukan status awal melainkan Close Claim, `1150` bukan
  penanda terdaftar melainkan LOD Report, `1151` bukan Investigator melainkan Analyst.
- Kalender libur memengaruhi **angka yang dilaporkan ke manajemen** — perubahannya perlu terlihat.

#### Migrasi skema / rollout / rollback

Menambah tabel master baru — tidak menyentuh tabel Pega. `V_STS_CLAIM` **tetap dibaca Pega**
selama masa paralel.

**Rollback:** aplikasi Go kembali membaca `V_STS_CLAIM` langsung.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/cek-master-status --harap 33      # HARUS 33 kode
grep -rInE "\"11[3-6][0-9]\"" internal/ | grep -v masterdata   # HARUS 0 baris
go test ./internal/app/masterdata/... -run TestAmbangLargeLosses
go test ./internal/domain/waktu/... -run TestHariLiburDariMaster
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 33 kode `1134`–`1166`; sebelas pertama membawa penomoran lama | `R-06` tertutup · `CONTEXT.md` · `ADR-0018` |
| Tiga arti kode yang disimpulkan ternyata salah | `docs/Steering/16-RISK-ANALYSIS.md` `R-06` |
| Ambang Large Losses `1000000000` | `Activity/InputRegister_act-Act.xml:19110` |
| `HRD_LBR` dan jam kerja lewat DB Link; ditulis ulang di Go | `D-50` · `ADR-0020` |
| Retensi satu kebijakan untuk klaim dan audit | `D-62` |

#### Comments

### TKT-F4-006 — Inventaris ≥29 kelompok master dan penetapan lingkup

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan** — mana dari ≥29 kelompok yang masuk lingkup belum ditetapkan |
| **Modul** | F-4 · Gelombang: 1 · Bergantung pada: — |
| **Requirement** | FR-F4 |
| **Keputusan** | D-03, D-34, D-35 |
| **ADR** | 0025 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | seluruh tabel master yang dirujuk 652 rule SQL; **14 kelompok sudah terpetakan**, sisanya belum |
| **Peran penguji gerbang 2** | **tidak berlaku** — tiket analisis (`D-60`) |
| **Label** | `modul::F-4` `tipe::analisis` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-4-Master-Data-dan-Parameter-Bisnis/issues/06-inventaris-kelompok-master.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Daftar lengkap kelompok master beserta **keputusan masuk atau tidaknya ke lingkup migrasi**,
sehingga `F-4` punya batas yang jelas.

Nilai bisnisnya adalah mencegah `F-4` menjadi modul tanpa ujung. Ukurannya sudah naik dari 14 ke
**≥29 kelompok**, dan tanda **≥** itulah masalahnya: selama batas atasnya tidak diketahui, estimasi
`F-4` maupun `U-6` tidak dapat dipertanggungjawabkan.

#### Ruang lingkup

- Inventaris seluruh tabel master yang dirujuk rule, dikelompokkan menurut fungsi bisnisnya.
- Untuk setiap kelompok: **berapa rule yang membacanya**, apakah isinya berubah oleh pengguna
  bisnis, dan siapa pemiliknya.
- Usulan **masuk lingkup / tidak masuk lingkup** per kelompok, untuk diputuskan Work Owner.
- Hasil akhir: daftar tetap yang menjadi dasar tiket master berikutnya.

#### Non-goal

- **Tidak** membangun master apa pun.
- **Tidak** memutuskan sendiri mana yang masuk lingkup — usulannya disiapkan, keputusannya milik
  Work Owner.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Mana dari ≥29 kelompok master yang masuk lingkup migrasi?** Sejauh ini baru satu area yang diputuskan: **Bengkel/Sparepart/Supplier dan ruleset GKM dikecualikan** (`D-34`) | **Work Owner** | Menentukan ukuran `F-4` dan `U-6`, dan karenanya menentukan kebutuhan orang |

**Catatan yang relevan bagi keputusan itu.** `D-03` menetapkan lingkup adalah **seluruh rule yang
ada di export**. Audit menemukan **35 rule** yang tidak dimiliki modul mana pun — 31 activity
Bengkel/Sparepart berruleset `GCNMFW` dan 4 activity master Supplier berruleset `GKM`. `D-34`
mengecualikan keduanya. Pertanyaan yang tersisa adalah **apakah ada area serupa lain** di antara
kelompok master yang belum terpetakan.

#### Acceptance criteria

- ☐ Inventaris memuat **seluruh** kelompok master yang dirujuk rule, dengan jumlah yang dapat
      dihitung ulang dari export — bukan perkiraan.
- ☐ Setiap kelompok punya: nama tabel, jumlah rule pembaca, pemilik bisnis (atau **"belum
      diketahui"**), dan usulan masuk/tidak masuk lingkup.
- ☐ Kelompok yang sudah diputuskan **dikecualikan** (`D-34`) ditandai beserta rujukan
      keputusannya.
- ☐ Selisih terhadap angka 14 pada dokumen v1.0 dijelaskan: mana yang bertambah dan dari mana
      angkanya.
- ☐ Hasilnya **memperbarui `docs/Steering/05-DOMAIN-MODEL.md` §5** dan ukuran `F-4`/`U-6` di
      `06-MODULE-BREAKDOWN.md` — diajukan sebagai perubahan, bukan langsung diterapkan diam-diam.

#### Dependency / Blocked by

Tidak bergantung pada tiket lain — **ini tiket analisis yang boleh dikerjakan paling awal**, dan
hasilnya membuka `TKT-F4-005` serta memperbaiki estimasi `U-6`.

**Terhalang keputusan Work Owner** hanya pada bagian penetapan lingkupnya; **inventarisnya sendiri
tidak terhalang**.

#### Constraint keamanan, data, operasional

- Inventaris dibuat dari **pembacaan export**, bukan dari koneksi database. Tidak ada akses
  produksi.
- Nama tabel dan kolom boleh ditulis; **isi datanya tidak** (`D-69`).

#### Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema — tiket ini menghasilkan dokumen.

**Rollback:** tidak berlaku.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
### hitung tabel master yang dirujuk rule SQL, kelompokkan
go run ./cmd/tools/inventaris-master "RDB List/" > inventaris-master.csv
wc -l inventaris-master.csv
### bandingkan dengan 14 kelompok yang sudah terpetakan
go run ./cmd/tools/banding-master inventaris-master.csv docs/Steering/05-DOMAIN-MODEL.md
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| ≥29 kelompok master, bukan 14 | `T-13` · `docs/Steering/05-DOMAIN-MODEL.md` §5 |
| Lingkup adalah seluruh rule yang ada di export | `D-03` |
| Bengkel/Sparepart/Supplier dan ruleset GKM dikecualikan | `D-34` |
| 35 rule tanpa modul pemilik | `D-34` · `docs/verifikasi-bukti-adr.md` |
| Angka mengikuti bukti, bukan dokumen | `D-35` |

#### Comments

# 6. F-5 — Waktu & Zona Waktu

*Modul Fondasi · folder `docs/ticketing/F-5-Tanggal-Jam-Kerja-dan-Hari-Libur/`*

## Spesifikasi Modul

| | |
|---|---|
| **Modul** | `F-5` Waktu & Zona Waktu |
| **Gelombang** | 1 — Fondasi |
| **Ukuran** | Kecil |
| **Bergantung pada** | `F-1` |
| **Kesiapan** | **SEBAGIAN** — lingkup jelas (**118 titik di 36 activity**), empat keputusan memengaruhi AC |
| **Cakupan tiket** | **penuh** (`D-41` Opsi 1) |

### Apa yang dibangun

Satu sumber waktu untuk seluruh aplikasi, satu tempat konversi zona waktu, dan satu tempat aturan
hari kalender — menggantikan penyesuaian **7 jam manual** yang tersebar di sistem lama.

### Kenapa modul kecil ini tidak boleh ditunda

`F-5` menghapus penyesuaian 7 jam yang tersebar di 36 activity. Bila dikerjakan belakangan,
**seluruh aturan tanggal yang telanjur ditulis harus ditulis ulang** — dan aturan tanggal adalah
gerbang validasi terberat di registrasi klaim (`B-2`).

Lebih berat lagi: penyesuaian itu **tidak konsisten di dalam satu kondisi validasi**
(`Activity/InputRegister_act-Act.xml:5788` versus `:4805`), sehingga hasil validasi di batas
periode polis berbeda tergantung cabang mana yang dijalankan. Itu butir 3 pada daftar 13 perbaikan
eksplisit `P-5`.

### Yang mengikat modul ini

| Sumber | Isi |
|---|---|
| `D-49` butir 3 | Penyesuaian 7 jam asimetris **diperbaiki** — masuk daftar perbaikan eksplisit `P-5` |
| `ADR-0020` | Dua basis perhitungan TAT dipertahankan; logika jam kerja **ditulis ulang di Go**, bukan dipanggil lewat API |
| `R-12` | Zona waktu bergeser saat migrasi data — berlaku juga pada penyalinan ke staging |
| `D-50` | `GET_WORKING_HOURS` → KPI dan kronologi TAT · `GETSELISIHJAM` → satu layar inbox Compliance · `HRD_LBR` → hari libur |

### Penghalang modul ini

Seluruhnya **keputusan**, bukan artefak — lingkupnya sendiri sudah terukur.

| Pertanyaan | Pemilik | Menahan |
|---|---|---|
| **`addCalendar(…, 12, 0, 0)` di 101 titik pada 4 activity — apa maksudnya?** Menambah 12 jam, atau menetapkan tengah hari? | Work Owner + Tim Pega | `TKT-F5-002` |
| `"WIB"` (25×) versus `Asia/Jakarta` (137×) — hasilnya sama? | Lead Engineer + DBA | `TKT-F5-001` (verifikasi) |
| Adakah data produksi yang **tergeser ganda +14 jam**? | DBA | `TKT-F5-002` |
| Representasi otoritatif untuk `DateOfLoss` dan `ReportDate` — tanggal murni atau waktu penuh? | Work Owner | `TKT-F5-001` |
| Dari mana daftar hari libur diperoleh setiap tahun? | Work Owner | `TKT-F5-003` |

### Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-F5-001](issues/01-seam-clock-dan-penyimpanan-utc.md) | Seam Clock dan penyimpanan UTC | `needs-info` | terhalang keputusan |
| [TKT-F5-002](issues/02-konversi-wib-tunggal.md) | Konversi WIB tunggal dan penghapusan penyesuaian 7 jam | `needs-info` | terhalang keputusan |
| [TKT-F5-003](issues/03-kalender-hari-kerja-dan-libur.md) | Kalender hari kerja dan hari libur | `needs-info` | terhalang keputusan |

## Daftar Tiket

### TKT-F5-001 — Seam Clock dan penyimpanan UTC

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan** — representasi otoritatif tanggal kejadian belum ditetapkan |
| **Modul** | F-5 · Gelombang: 1 · Bergantung pada: TKT-F1-001 |
| **Requirement** | FR-F5 |
| **Keputusan** | D-49 butir 3 |
| **ADR** | 0017 |
| **Risiko** | R-12 |
| **Rule Pega yang digantikan** | pemakaian waktu langsung yang tersebar — `SYSDATE` di SQL, `@CurrentDateTime` di activity, dan **118 titik penyesuaian waktu di 36 activity** |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`) |
| **Label** | `modul::F-5` `tipe::fondasi` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-5-Tanggal-Jam-Kerja-dan-Hari-Libur/issues/01-seam-clock-dan-penyimpanan-utc.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu antarmuka pengambil waktu yang **dapat digantikan saat pengujian**, dan aturan tegas bahwa
waktu disimpan dalam **UTC**.

Nilai bisnisnya: aturan tanggal adalah gerbang validasi terberat di registrasi klaim — tanggal
kejadian ≤ tanggal lapor ≤ tanggal terima dokumen ≤ hari ini, ditambah batas periode polis. Aturan
seperti itu **tidak dapat diuji** bila waktunya diambil langsung dari jam sistem; kasus batas
tengah malam hanya dapat diuji dengan waktu yang dapat dikendalikan.

#### Ruang lingkup

- Antarmuka `Clock` dideklarasikan di lapisan Domain, dengan dua adapter: jam nyata dan jam
  terkendali untuk pengujian.
- **Larangan memanggil waktu sistem langsung** di luar adapter — ditegakkan pemeriksaan pola.
- Aturan penyimpanan: seluruh kolom waktu disimpan **UTC**; konversi ke WIB hanya di batas
  tampilan.
- Tipe terpisah untuk **tanggal murni** (tanpa jam) dan **waktu penuh**, agar keduanya tidak
  tertukar.

#### Non-goal

- **Tidak** melakukan konversi tampilan — itu `TKT-F5-002`.
- **Tidak** menangani kalender hari kerja — itu `TKT-F5-003`.
- **Tidak** memperbaiki data historis yang mungkin tergeser — itu pekerjaan migrasi data.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Akibat |
|---|---|---|
| **`DateOfLoss` dan `ReportDate`: tanggal murni atau waktu penuh?** | **Work Owner** | Menentukan tipe kolom dan apakah kasus tengah malam bisa terjadi sama sekali. Bila tanggal murni, seluruh kelas cacat `R-12` hilang untuk kedua field ini |
| **`"WIB"` (25 pemakaian) versus `Asia/Jakarta` (137 pemakaian) — hasilnya identik?** | **Lead Engineer + DBA** | Bila berbeda, ada data produksi yang sudah tergeser dan migrasi data harus menanganinya |

#### Acceptance criteria

- ☐ Tidak ada pemanggilan waktu sistem di luar adapter Clock — diuji pemindaian: `time.Now()`
      **0 kemunculan** di luar satu berkas adapter.
- ☐ Aturan bisnis yang bergantung waktu dapat diuji dengan waktu yang ditetapkan — dibuktikan
      satu uji yang memajukan waktu melewati tengah malam dan hasilnya **deterministik**.
- ☐ Seluruh kolom waktu disimpan UTC — diuji dengan menulis lalu membaca kembali pada dua
      pengaturan zona waktu sistem yang berbeda dan hasilnya **sama**.
- ☐ Tipe tanggal murni dan waktu penuh **tidak dapat saling ditugaskan** tanpa konversi
      eksplisit — dibuktikan kegagalan kompilasi pada uji percobaan.
- ☐ Uji zona waktu dijalankan pada `TZ=UTC` dan `TZ=Asia/Jakarta`, keduanya **lulus**.

#### Dependency / Blocked by

Bergantung pada `TKT-F1-001`. Terhalang dua keputusan di atas.

**Yang bergantung padanya:** `TKT-F2-006` (pergantian tahun pada nomor klaim), seluruh aturan
tanggal `B-2`, dan perhitungan TAT `S-7`.

#### Constraint keamanan, data, operasional

- `SYSDATE` **dilarang** di SQL kecuali di generator nomor klaim (`ADR-0009`), dan di sana ia
  memang tanggal server — bukan tanggal bisnis.
- `R-12` berlaku pada **penyalinan data ke staging**, bukan hanya migrasi akhir. Salinan yang
  bergeser menghasilkan **selisih palsu** di setiap pengujian kesetaraan.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema pada tiket ini. **Rollback:** mengembalikan adapter Clock; tidak ada data
yang berubah.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
grep -rIn "time.Now()" internal/ | grep -v adapter/clock   # HARUS 0 baris
TZ=UTC go test ./...
TZ=Asia/Jakarta go test ./...
go test ./internal/domain/... -run TestAturanTanggalTengahMalam
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 118 titik penyesuaian waktu di 36 activity | `docs/verifikasi-bukti-adr.md` §15 baris `F-5` |
| `"WIB"` 25× versus `Asia/Jakarta` 137× | idem |
| Penyesuaian 7 jam asimetris | `Activity/InputRegister_act-Act.xml:5788` dan `:4805` · `D-49` butir 3 |
| Seam Clock, penyimpanan UTC, konversi WIB tunggal | `docs/Steering/04-FUTURE-ARCHITECTURE.md` §3.2 · `06-MODULE-BREAKDOWN.md` `F-5` |
| Zona waktu bergeser saat migrasi data | `R-12` |

#### Comments

### TKT-F5-002 — Konversi WIB tunggal dan penghapusan penyesuaian 7 jam

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan** — arti `addCalendar(…, 12, 0, 0)` belum diketahui |
| **Modul** | F-5 · Gelombang: 1 · Bergantung pada: TKT-F5-001 |
| **Requirement** | FR-F5 |
| **Keputusan** | D-49 butir 3 |
| **ADR** | 0017 |
| **Risiko** | R-12 |
| **Rule Pega yang digantikan** | **118 titik penyesuaian waktu di 36 activity**, di antaranya penyesuaian 7 jam yang **asimetris di dalam satu kondisi validasi** — `Activity/InputRegister_act-Act.xml:5788` versus `:4805` — dan **101 titik `addCalendar(…, 12, 0, 0)` di 4 activity** |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`) |
| **Label** | `modul::F-5` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-5-Tanggal-Jam-Kerja-dan-Hari-Libur/issues/02-konversi-wib-tunggal.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu tempat yang mengubah UTC menjadi WIB untuk ditampilkan, dan **nol penyesuaian jam manual**
di dalam aturan bisnis.

Nilai bisnisnya adalah menutup cacat yang **mengubah hasil validasi**. Penyesuaian 7 jam di sistem
lama diterapkan tidak konsisten di dalam satu kondisi validasi yang sama, sehingga klaim dengan
tanggal kejadian di batas periode polis **bisa lolos atau ditolak tergantung cabang mana yang
dijalankan**. Ini butir 3 pada daftar 13 perbaikan eksplisit `P-5` — artinya selisihnya pada uji
kesetaraan sudah dinyatakan di muka sebagai perbaikan terencana.

#### Ruang lingkup

- Satu fungsi konversi UTC→WIB untuk tampilan dan WIB→UTC untuk masukan, dipakai seluruh aplikasi.
- **Penghapusan seluruh penyesuaian jam manual** dari aturan bisnis — tidak ada `+7`, `-7`, atau
  `addCalendar` di dalam logika validasi.
- Pemeriksaan pola yang **menggagalkan build** bila penyesuaian jam manual muncul kembali.
- Daftar 118 titik yang terdampak, dipakai sebagai daftar periksa saat modul bisnis ditulis.

#### Non-goal

- **Tidak** memperbaiki data historis yang mungkin tergeser — itu pekerjaan migrasi data, dan
  butuh jawaban DBA lebih dulu.
- **Tidak** mengubah aturan bisnis tanggalnya sendiri — hanya cara waktunya diperlakukan.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diketahui | Pemilik | Kenapa menahan |
|---|---|---|
| **Arti `addCalendar(…, 12, 0, 0)` pada 101 titik di 4 activity** — menambah 12 jam, atau menetapkan waktu ke tengah hari? | **Work Owner + Tim Pega** | Keduanya menghasilkan tanggal berbeda pada kasus batas. Menebaknya berarti mengubah hasil validasi 101 titik tanpa sadar |
| **Adakah data produksi yang tergeser ganda +14 jam?** | **DBA** | Bila ada, konversi yang benar di sistem baru akan menampilkan tanggal yang **berbeda** dari yang dilihat pengguna hari ini — dan itu harus disiapkan, bukan dikejutkan |

#### Acceptance criteria

- ☐ **Nol penyesuaian jam manual** di seluruh kode aturan bisnis — dihitung dan dilaporkan
      angkanya; pemeriksaan pola menggagalkan build bila muncul.
- ☐ Konversi WIB berada di **tepat satu berkas** — diuji pemindaian.
- ☐ Waktu yang ditampilkan untuk satu nilai UTC **sama persis** dengan yang ditampilkan Pega
      untuk nilai yang sama — diuji pada 20 nilai contoh yang mencakup tengah malam WIB dan
      pergantian bulan.
- ☐ Validasi "tanggal kejadian dalam periode polis" memberi hasil **yang sama untuk kedua
      cabang** yang di sistem lama berbeda — diuji khusus pada tanggal batas.
- ☐ Daftar 118 titik terdampak tersedia sebagai berkas dengan `berkas:baris`, dan jumlahnya
      dapat dihitung ulang.

#### Dependency / Blocked by

Bergantung pada `TKT-F5-001`. Terhalang dua pertanyaan di atas.

#### Constraint keamanan, data, operasional

- Perubahan ini **mengubah hasil validasi** pada kasus batas. Karena sudah terdaftar sebagai butir
  3 `P-5`, selisihnya pada gerbang 1 **lolos otomatis dan cukup dicatat** (`D-54`) — tetapi hanya
  bila selisihnya memang terpetakan ke butir itu.
- Tidak ada perubahan data. Bila ternyata dibutuhkan perbaikan data historis, itu **tiket
  tersendiri** dengan persetujuan Work Owner dan DBA.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** mengembalikan fungsi konversi — tetapi perlu disadari bahwa
mengembalikannya berarti **memulihkan cacat asimetris**, sehingga rollback hanya masuk akal bila
ada temuan yang lebih buruk.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
grep -rInE "(\+|\-)\s*7\s*\*\s*time\.Hour|addCalendar" internal/   # HARUS 0 baris
go test ./internal/platform/waktu/... -run TestKonversiWIB
go test ./internal/domain/... -run TestPeriodePolisBatasTanggal
diff <(go run ./cmd/tools/tampilkan-waktu contoh.json) baseline-pega.json
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Penyesuaian 7 jam asimetris di satu kondisi validasi | `Activity/InputRegister_act-Act.xml:5788` dan `:4805` |
| Butir 3 daftar 13 perbaikan eksplisit `P-5` | `D-49` · `ADR-0017` · `docs/Steering/07-MIGRATION-STRATEGY.md` |
| 101 titik `addCalendar(…,12,0,0)` di 4 activity | `docs/verifikasi-bukti-adr.md` §15 baris `F-5` |
| 118 titik di 36 activity | idem |
| Selisih terpetakan `P-5` lolos otomatis di gerbang 1 | `D-54` · `docs/Steering/14-TESTING-STRATEGY.md` §6.2 |

#### Comments

### TKT-F5-003 — Kalender hari kerja dan hari libur

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan dan artefak** — sumber kalender belum ditetapkan; sumber `GET_WORKING_HOURS` belum diterima |
| **Modul** | F-5 · Gelombang: 1 · Bergantung pada: TKT-F5-001 |
| **Requirement** | FR-F5 |
| **Keputusan** | D-50, D-25 |
| **ADR** | 0008, 0020 |
| **Risiko** | R-03 |
| **Rule Pega yang digantikan** | `DATAMINING.GET_WORKING_HOURS@ASMD` — **18 pemakaian di 7 berkas** · `HRD_LBR` (hari libur) · `GETSELISIHJAM` (dipakai satu layar inbox Compliance) |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi (`D-60`); angkanya diuji pengguna lewat `S-7` |
| **Label** | `modul::F-5` `tipe::migrasi` `status::needs-info` `prioritas::sedang` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-5-Tanggal-Jam-Kerja-dan-Hari-Libur/issues/03-kalender-hari-kerja-dan-libur.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Perhitungan **jam kerja** dan **hari libur** sebagai aturan bisnis milik aplikasi ini, bukan
panggilan ke database lain.

Nilai bisnisnya: angka TAT adalah angka yang **dilaporkan ke manajemen**. `D-50` menetapkan
logikanya ditulis ulang di Go, bukan dipanggil lewat API — karena `GET_WORKING_HOURS` dipakai
**di dalam kalkulasi laporan massal**, dan menjadikannya panggilan jaringan per baris akan
menghancurkan kinerja laporan.

#### Ruang lingkup

- Perhitungan selisih **jam kerja** antara dua waktu, memperhitungkan akhir pekan dan hari libur.
- Master **hari libur** dan definisi **jam kerja** sebagai data (`F-4`), bukan konstanta di kode.
- Dua basis perhitungan dipertahankan sesuai `D-50`, dengan **batas pemakaian yang tegas**:
  jam kerja untuk KPI, laporan admin, dan kronologi TAT; selisih hari untuk satu layar inbox
  Compliance.
- Perbaikan butir 10 `D-49`: kegagalan **tidak** mengembalikan `0` yang tak terbedakan dari nol.

#### Non-goal

- **Tidak** membangun laporan TAT — itu `S-7`.
- **Tidak** menyeragamkan kedua basis menjadi satu — `D-50` menolaknya, karena keduanya tidak
  pernah dipakai mengukur hal yang sama.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Sumber `DATAMINING.GET_WORKING_HOURS@ASMD`** — definisi jam kerja (mulai, selesai, istirahat) hanya ada di dalam fungsi remote yang belum dibaca | **DBA** (`R-03`) | Menulis ulang aturan yang belum pernah dibaca berarti menebak; dan angkanya dilaporkan ke manajemen |
| **Dari mana daftar hari libur diperoleh setiap tahun, dan siapa yang mengisinya?** | **Work Owner** | Menentukan apakah master hari libur diisi manual di `F-4` atau ditarik dari sistem HRD |

#### Acceptance criteria

- ☐ Selisih jam kerja antara dua waktu **sama dengan** keluaran `GET_WORKING_HOURS` pada
      **30 pasangan waktu contoh** yang mencakup: akhir pekan, hari libur di tengah, lintas bulan,
      dan dua waktu di hari yang sama.
- ☐ Hari libur dibaca dari master, **bukan** dari konstanta di kode — diuji dengan menambah satu
      hari libur dan memeriksa hasil perhitungan berubah.
- ☐ Kegagalan perhitungan mengembalikan **galat**, bukan `0` — diuji dengan masukan tidak sah;
      nilai `0` hanya muncul bila selisihnya memang nol.
- ☐ `GETSELISIHJAM` hanya dipakai pada **satu layar** (inbox Compliance) — diuji pemindaian:
      pemanggilnya tepat satu.
- ☐ Perhitungan **tidak melakukan panggilan jaringan** — diuji dengan menjalankan uji tanpa
      akses jaringan.

#### Dependency / Blocked by

- Bergantung pada `TKT-F5-001`.
- Membutuhkan master hari libur dan jam kerja dari `F-4`.
- **Terhalang** `R-03` (sumber fungsi remote) dan keputusan sumber kalender.

#### Constraint keamanan, data, operasional

- **Tidak boleh** menjadi panggilan jaringan per baris — itu alasan utama `D-50` memilih menulis
  ulang alih-alih membungkusnya dengan API.
- Angka TAT dilaporkan ke manajemen; **selisih sekecil apa pun setelah migrasi akan terlihat** dan
  harus dapat dijelaskan.

#### Migrasi skema / rollout / rollback

Menambah master hari libur dan jam kerja (tabel baru, tidak menyentuh tabel Pega).

**Rollback:** kembali memanggil `GET_WORKING_HOURS` lewat DB Link **tidak tersedia sebagai
rollback** — `ADR-0008` menghapus DB Link. Bila hasil perhitungan meragukan, yang dilakukan adalah
menahan rilis laporan, bukan mengembalikan DB Link.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/waktu/... -run TestJamKerja
### bandingkan 30 pasangan dengan keluaran fungsi lama (dijalankan DBA di staging)
go run ./cmd/tools/banding-jam-kerja pasangan.csv > hasil.csv && diff hasil.csv baseline.csv
go test ./... -run TestJamKerjaTanpaJaringan
grep -rIn "GETSELISIHJAM" internal/ | wc -l     # HARUS tepat 1 pemanggil
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `GET_WORKING_HOURS` 18 pemakaian di 7 berkas | `D-50` · `ADR-0020` |
| Dua basis dipertahankan dengan batas pemakaian | `D-50` |
| Logika ditulis ulang di Go, bukan dipanggil lewat API | `D-50` · `docs/Steering/10-API-STRATEGY.md` §8.4 |
| `GETSELISIHJAM` gagal → `RETURN 0` | `Database/GETSELISIHJAM.fnc:22` · `D-49` butir 10 |
| Kalender libur menjadi master milik aplikasi; sumbernya belum ditetapkan | `ADR-0020` pertanyaan terbuka |

#### Comments

# 7. F-6 · Portal & Multi-Sumber Data

*Modul Fondasi · folder `docs/ticketing/F-6-Portal-dan-Multi-Sumber-Data/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **tidak ada namanya** — dikerjakan diam-diam lewat perbandingan hostname: **48 perbandingan** terhadap `pxRequestor.pxReqServer` pada 3 hostname, ditambah When rule `IsServerSyariah` yang **hilang dari export** |
| **Kode modul** | `F-6` |
| **Gelombang** | 1 — Fondasi |
| **Ukuran** | Sedang — tetapi **melipatgandakan** beban `F-4`, `S-2`, `S-5`, `U-6`, dan `S-8` |
| **Bergantung pada** | `F-1` Kerangka Aplikasi · `F-2` Akses Data · `F-3` Login & Hak Akses |
| **Kesiapan** | **SEBAGIAN** |

### Apa yang dikerjakan modul ini

Memberi pengguna **pilihan portal** — sebuah **dropdown di dalam aplikasi**, bukan kembali ke
halaman login (`D-77`) — dan mengarahkan seluruh pembacaan serta penulisan ke **database
entitas** yang dipilih.

| Portal | Jejak di export |
|---|---|
| Asuransi Sinar Mas | entitas utama |
| Asuransi Simas Insurtech | **13** perbandingan hostname |
| Sinarmas Asuransi Syariah | `SpreadingSyariah_Act` ada · **`IsServerSyariah` hilang** |
| Timor-Leste | **1** perbandingan · **mata uang berbeda** |

### Ini menyingkap yang sudah ada, bukan menambah kemampuan

Sistem lama sudah berperilaku berbeda per entitas — dengan **membandingkan nama server**. Salah
satu perbandingan itu **mengubah ambang komite dari Rp 50.000.000 menjadi 3.500**; selisihnya bukan
salah ketik melainkan **mata uang yang berbeda**.

Tidak ada pilihan portal, tidak ada konfigurasi, dan tidak pernah tercatat sebagai rancangan.

### Temuan yang mengubah rencana

| Anggapan | Terverifikasi |
|---|---|
| 3 hostname penentu perilaku | **Bisa lebih dari 3** — `IsServerSyariah` dan `IsDevelopmentServer` **hilang dari export dan tidak tercatat di `19-GAP`** (`verifikasi-bukti-adr.md:788-791`) |
| Database bersama (`D-21`, `ADR-0004`) | **Satu database per entitas** (`D-75`) |
| Syariah hanya varian produk | Punya **logika spreading sendiri** — `Activity/SpreadingSyariah_Act-Act.xml` |

> **Jawaban Work Owner mengoreksi bukti kami.** Dokumen mencatat 3 hostname; yang benar minimal 4
> portal. Angka 3 rendah karena **rule pembedanya hilang**, bukan karena entitasnya tiga. Karena itu
> daftar portal diperlakukan sebagai **data**, bukan konstanta di kode.

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **`IsServerSyariah` dan `IsDevelopmentServer` hilang** — tidak diketahui **kapan** perilaku syariah berlaku | **Tim Pega** (`R-16`) |
| **Keputusan** | Jumlah portal sebenarnya — Work Owner menyebut *minimal* 4 | **Work Owner** |
| **Keputusan** | Nomor klaim `PNCN.YY.xxxx` memakai sequence **per database** — dua portal dapat menerbitkan nomor sama. Perlu penanda portal? | **Work Owner** (`D-71`) |
| **Keputusan** | Sejauh apa aturan bisnis syariah berbeda di luar spreading? | **Work Owner + Tim Pega** |
| **Keputusan** | Satu penyedia identitas untuk keempat entitas, atau satu per entitas? | **Work Owner** (`F-3`) |
| **Keputusan** | Mata uang, kurs, dan pembulatan per portal | **Work Owner** |

### Akibat pada modul lain — bukan detail

| Modul | Akibat |
|---|---|
| `F-2` | Pool koneksi **per portal**; migrasi skema berjalan **empat kali** |
| `F-3` | Kewenangan **per portal**, dinilai ulang saat berpindah (`R-20`) |
| `F-4` · `U-6` | Master ≥29 kelompok kini **per portal** — lingkup berlipat |
| `S-2` · `S-7` | Laporan **per portal**; konsolidasi lintas portal **tidak mungkin** tanpa keputusan baru |
| `S-5` | Jejak audit **per portal** — penyelidikan lintas entitas menuntut empat kueri |
| `S-8` | Uji kesetaraan berjalan **per portal**; portal Syariah **belum punya baseline utuh** |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-F6-001](issues/01-daftar-portal-sebagai-data.md) | Daftar portal sebagai data, bukan konstanta | `needs-info` |
| [TKT-F6-002](issues/02-koneksi-per-portal.md) | Koneksi dan pool per portal | `needs-info` |
| [TKT-F6-003](issues/03-perpindahan-portal-dan-kewenangan.md) | Perpindahan portal dan penilaian ulang kewenangan | `needs-info` |
| [TKT-F6-004](issues/04-kesekerabatan-skema-empat-database.md) | Kesekerabatan skema empat database | `needs-info` |

## Daftar Tiket

### TKT-F6-001 — Daftar portal sebagai data, bukan konstanta

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** (`IsServerSyariah`) dan keputusan mata uang |
| **Modul** | **F-6 Portal & Multi-Sumber Data** · Gelombang: 1 · Bergantung pada: TKT-F1-002, TKT-F4-001 |
| **Requirement** | FR-F6 |
| **Keputusan** | D-75, D-77 |
| **ADR** | 0030 |
| **Risiko** | R-20 |
| **Rule Pega yang digantikan** | **48 perbandingan** terhadap `pxRequestor.pxReqServer` pada 3 hostname · When rule `IsServerSyariah` dan `IsDevelopmentServer` (**keduanya hilang dari export**) |
| **Peran penguji gerbang 2** | **PncAdmin** dan **PncManagerAdmin** |
| **Label** | `modul::F-6` `tipe::fondasi` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-6-Portal-dan-Multi-Sumber-Data/issues/01-daftar-portal-sebagai-data.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Daftar portal tersimpan sebagai **data**, sehingga menambah entitas tidak menuntut perubahan kode.

Nilai bisnisnya terlihat dari cara lama: perilaku entitas ditentukan dengan **membandingkan nama
server di 48 tempat**. Menambah satu entitas berarti menambah 48 percabangan — dan melewatkan satu
saja menghasilkan entitas yang berperilaku separuh benar, tanpa ada yang tahu.

Ada bukti langsung bahwa cara lama itu memang gagal: **`IsServerSyariah` hilang dari export dan
tidak tercatat di `19-GAP`**, sehingga dokumen kami mencatat 3 hostname padahal entitasnya minimal
4. Cara lama **menyembunyikan satu entitas penuh dari inventaris kami sendiri**.

#### Ruang lingkup

- Tabel portal: kode, nama tampilan, rujukan konfigurasi database, mata uang, status aktif.
- Pembacaan daftar portal saat start, dan penyajiannya sebagai pilihan bagi pengguna yang berhak.
- **Tidak ada nama entitas yang tertanam di kode** — termasuk nama keempat portal yang sudah diketahui.
- Perilaku bila sebuah portal dinyatakan tidak aktif.

#### Non-goal

- **Tidak** membuat koneksinya — itu `TKT-F6-002`.
- **Tidak** menangani perpindahan portal dan kewenangannya — itu `TKT-F6-003`.
- **Tidak** memutuskan aturan bisnis yang berbeda per entitas.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| ~~Berapa portal sebenarnya?~~ — **TERTUTUP `D-77`**: **4 sekarang, dan bisa bertambah** | — | Menguatkan rancangan: daftar portal adalah **data**, dan penambahan portal **tidak boleh** menuntut perubahan kode |
| **`IsServerSyariah` hilang** — kapan perilaku syariah berlaku? | **Tim Pega** (`R-16`) | Tanpa rule itu, portal Syariah dapat dibuat tetapi **tidak dapat dibuktikan setara** |
| **Mata uang per portal** — Timor-Leste bukan Rupiah | **Work Owner** | Menentukan kolom mata uang di tabel portal, dan berpengaruh ke kurs serta pembulatan |

#### Acceptance criteria

- ☐ Daftar portal berasal dari **data**; menambah portal **tidak menuntut perubahan kode** —
      diuji dengan menambah satu portal lewat data saja.
- ☐ **Nol nama entitas tertanam di kode** — dicari otomatis atas keempat nama yang diketahui:
      **nol temuan**.
- ☐ Portal yang **tidak aktif** tidak muncul sebagai pilihan dan **tidak dapat dipanggil
      langsung** — diuji: `403`, bukan sekadar tersembunyi.
- ☐ Aplikasi **menolak start** bila sebuah portal aktif tidak punya konfigurasi database yang
      sah — gagal keras, bukan diam (`TKT-F1-002`).
- ☐ Mata uang portal terbaca dari data, dan dipakai pemformat terpusat (`TKT-U2-004`) — diuji
      dengan dua portal bermata uang berbeda.
- ☐ Gerbang 2: UAT **PncAdmin**.

#### Dependency / Blocked by

`TKT-F1-002` (konfigurasi) · `TKT-F4-001` (kerangka master). **Terhalang Work Owner dan Tim Pega.**

#### Constraint keamanan, data, operasional

- Konfigurasi database **empat entitas** berada di satu tempat. Kebocoran berkas konfigurasi
  berarti kebocoran **empat** basis data sekaligus, bukan satu — kredensialnya wajib dari
  penyimpanan rahasia (`D-40`, `R-17`).
- Nama entitas adalah **nama badan hukum**. Salah menampilkan portal berarti menampilkan data satu
  badan hukum di bawah nama badan hukum lain.

#### Migrasi skema / rollout / rollback

Menambah tabel portal. Tidak menyentuh tabel klaim. Backward-compatible (`P-4`).

**Rollout:** portal utama lebih dulu, entitas lain menyusul — bukan keempatnya sekaligus.

**Rollback:** menonaktifkan portal lewat data. **Menonaktifkan portal yang sedang dipakai memutus
akses pengguna entitas itu** — bukan tindakan netral.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/portal/... -run TestDaftarPortalDariData
go run ./cmd/tools/cari-nama-entitas ./internal/...   # HARUS nol temuan
go test ./internal/app/portal/... -run TestPortalTidakAktifDitolak
go test ./internal/app/portal/... -run TestGagalStartBilaKonfigurasiPortalTidakSah
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 48 perbandingan `pxRequestor.pxReqServer`, 3 hostname | `docs/Steering/03-CURRENT-ARCHITECTURE.md:180` |
| Ambang komite berubah Rp 50.000.000 → 3.500 | `docs/Steering/11-SECURITY.md:219` |
| `IsServerSyariah` dan `IsDevelopmentServer` hilang, tidak tercatat di `19-GAP` | `docs/verifikasi-bukti-adr.md:788-791` |
| Satu database per entitas, minimal 4 portal | `D-75` · `ADR-0030` |
| Logika spreading syariah terpisah | `Activity/SpreadingSyariah_Act-Act.xml` |

#### Comments

### TKT-F6-002 — Koneksi dan pool per portal

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang `D-78`** — di mana kewenangan portal disimpan. Penomoran klaim tertutup (`D-76`); jumlah portal tertutup (Work Owner 2026-09-19: **pool mengikuti database yang ada**, bukan angka tetap — sudah terpasang, lihat "Keadaan kode"). |
| **Modul** | **F-6 Portal & Multi-Sumber Data** · Gelombang: 1 · Bergantung pada: TKT-F6-001, TKT-F2-001 |
| **Requirement** | FR-F6 |
| **Keputusan** | D-75, D-71, D-76 |
| **ADR** | 0030, 0004 |
| **Risiko** | R-20 |
| **Rule Pega yang digantikan** | **tidak ada padanan** — sistem lama memakai satu koneksi dan membedakan perilaku lewat hostname |
| **Peran penguji gerbang 2** | **PncAdmin** |
| **Label** | `modul::F-6` `tipe::fondasi` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-6-Portal-dan-Multi-Sumber-Data/issues/02-koneksi-per-portal.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Setiap portal punya **pool koneksinya sendiri**, dan seluruh pembacaan serta penulisan mengikuti
portal yang sedang aktif.

Nilai bisnisnya adalah alasan `D-75` menolak database bersama: dengan koneksi terpisah, **pemisahan
data menjadi sifat struktural**. Portal A tidak dapat membaca data portal B karena koneksinya memang
berbeda — bukan karena setiap kueri ingat menyaring. Kelas kesalahan "ada satu kueri yang lupa
menyaring" **hilang seluruhnya**, dan kelas itu tidak dapat diuji habis.

#### Ruang lingkup

- Pool koneksi per portal, dibuat dari konfigurasi `TKT-F6-001`.
- Seam Repository (`TKT-F2-001`) mengambil koneksi **dari portal aktif**, bukan dari variabel global.
- Batas jumlah koneksi **per portal**, bukan satu batas dibagi rata.
- Pool laporan terpisah (`TKT-F2-001`) tetap berlaku — **per portal**.
- Perilaku bila database satu portal mati: portal lain **tetap berjalan**.

#### Non-goal

- **Tidak** menyatukan data antar portal dengan cara apa pun.
- **Tidak** membangun laporan lintas portal — `D-75` menetapkan laporan **per portal**.

#### Yang sudah ditegaskan Work Owner (2026-09-19)

| Pernyataan | Akibatnya bagi tiket ini |
|---|---|
| **Setiap portal terkoneksi ke database masing-masing** | Rancangan `D-75` dikonfirmasi; tidak ada lagi pertanyaan "apakah satu database bersama cukup" |
| **Pool dijalankan sesuai portal** | Ruang lingkup butir 1–3 dikonfirmasi apa adanya |
| **Tiap server punya `POOLDATA` sendiri**, dan aplikasi membaca milik server tempat ia berjalan | **Isi master pun berbeda antar portal** — lihat constraint di bawah. Ini yang menaikkan tiket dari "kerapian" menjadi "kebenaran data" |
| **Pool mengikuti database yang ada** — bukan angka portal yang ditetapkan di muka | Pertanyaan "berapa portal" **tertutup**: jawabannya bukan angka melainkan aturan. Perilakunya **sudah terpasang** — lihat "Keadaan kode" |

#### Keadaan kode hari ini — yang sudah ada dan yang belum

Diperiksa 2026-09-19. Pengerjaan tiket ini **tidak dimulai dari nol**:

| Bagian | Status | Letak |
|---|---|---|
| Pool per portal, dibuka dari konfigurasi | **sudah ada** | `internal/platform/db` — `KumpulanBaru`, `Untuk(alias)`, `Tersedia()`, `Utama()` |
| **Pool mengikuti database yang ada** | **sudah ada** | daftar alias dipindai dari variabel `POOLDATA_<ALIAS>_HOST` (tidak ada daftar portal di kode, `D-15`); `parameterPortal` melewati yang konfigurasinya tidak lengkap; `KumpulanBaru` melewati yang gagal dibuka dan mencatatnya — **kecuali portal utama, yang gagalnya fatal** |
| Portal yang databasenya mati **terlihat pengguna**, bukan hanya di log | **sudah ada** | `Tersedia()` → `aliasSiap` → `GET /api/portal` → `Portal.siap`; layar menandainya "belum tersedia", bukan menyembunyikannya |
| Portal yang sedang dipilih, bertahan melewati muat ulang | **sudah ada** | `frontend/src/app/portal.ts` — `gunakanPortalTerpilih`, sessionStorage |
| Daftar portal beserta kesiapannya | **sudah ada** | `internal/portal` + `GET /api/portal` |
| **Portal ikut pada setiap permintaan** | **BELUM ADA** | `frontend/src/api/klien.ts` tidak pernah mengirim portal |
| **Server memilih pool dari portal aktif** | **BELUM ADA** | seluruh repo memakai `Utama()` |

Jadi yang hilang tepat **satu mata rantai**: portal tidak pernah menyeberang dari peramban ke
server.

> **Peringatan bagi yang mengerjakan.** Mata rantai itu tampak sepele — kirim satu header, panggil
> `Untuk(alias)` — dan justru **di situ letak `R-20`**. Memilih pool dari alias yang dikirim klien
> **tanpa memeriksa kewenangan pengguna atas portal itu** membuat siapa pun dapat membaca data badan
> hukum lain hanya dengan mengganti satu header. Itu lebih buruk daripada keadaan sekarang.
>
> Pemeriksaan kewenangannya sendiri **belum dapat ditulis**, karena `D-78` meninggalkan pertanyaan
> di mana kewenangan portal disimpan: login sama untuk keempat entitas, sehingga data "siapa berhak
> atas portal mana" bersifat lintas portal dan tidak dapat tinggal di dalam database tiap portal —
> memeriksa hak atas portal B menuntut membaca database B sebelum penggunanya terbukti berhak.
>
> Urutan yang benar karena itu: `D-78` dijawab → `TKT-F6-003` (kewenangan dinilai ulang di server)
> → baru pemilihan pool di tiket ini.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Di mana kewenangan portal disimpan** (`D-78`) | **Work Owner + Keamanan Informasi** | Tanpa itu, memilih pool dari alias kiriman klien adalah kebocoran `R-20`, bukan perbaikan. **Inilah satu-satunya penghalang yang tersisa** |
| **Batas koneksi per portal berapa** | **Work Owner + Tim Infra** | Mekanismenya sudah ada (`POOLDATA_<ALIAS>_MAKS_KONEKSI`, bawaan 20/5/30m); yang belum ada angkanya. Tidak menahan pengerjaan — menahan penyetelan sebelum produksi |

**Sudah tertutup:**

- ~~**Berapa portal**, dan apakah keempat database sudah tersedia~~ — Work Owner, 2026-09-19:
  jawabannya bukan angka melainkan aturan, yaitu **pool mengikuti database yang ada**. Sudah
  terpasang; database yang belum siap dilewati, dicatat, dan ditandai "belum tersedia" di layar.
- ~~**Nomor klaim unik lintas portal?**~~ — `D-76`: tidak perlu penanda portal; nomor unik **di
  dalam portal**, tidak dijamin unik antar portal. Konsekuensinya: menyebut nomor klaim tanpa
  portalnya menjadi **ambigu**, termasuk pada LOD/PLA/DLA yang keluar ke pihak luar.

> **Penomoran klaim sudah diputuskan** (`D-76`): tanpa penanda portal. Bila kelak diputuskan ada
> laporan konsolidasi lintas portal, keputusan itu **harus ditinjau ulang lebih dulu** — menambah
> penanda setelah klaim terbit menuntut migrasi data.

#### Acceptance criteria

- ☐ Setiap portal punya pool sendiri, dan jumlah pool **sama dengan jumlah portal aktif** — diuji.
- ☐ Kueri yang dijalankan saat portal A aktif **tidak pernah menyentuh database portal B** —
      diuji dengan pencatatan koneksi pada 20 operasi berbeda: **nol kebocoran**.
- ☐ **Database satu portal mati tidak menjatuhkan portal lain** — diuji dengan mematikan satu
      database: portal lain tetap melayani.
- ☐ Batas koneksi ditegakkan **per portal**; satu portal yang sibuk **tidak menghabiskan jatah**
      portal lain — diuji dengan beban pada satu portal.
- ☐ Laporan berat di satu portal tidak menghabiskan koneksi transaksi portal itu maupun portal
      lain (`TKT-F2-001`) — diuji.
- ☐ Tidak ada jalur kode yang dapat memilih koneksi **tanpa portal aktif** — diuji: permintaan
      tanpa portal ditolak, bukan jatuh ke default.
- ☐ **Master data dibaca dari database portal aktif, bukan portal utama** — diuji pada kasus yang
      isinya benar-benar berbeda: membuka Ambang Komite di portal Simasnet mengembalikan baris
      ber-`TYPE_BUSINESS` `SIMASNET`, dan di portal ASM mengembalikan NONMBU/PA/TRAVEL/BONDING.
      Keduanya dibandingkan dalam satu sesi tanpa login ulang.
- ☐ Gerbang 2: UAT **PncAdmin** pada minimal dua portal.

#### Dependency / Blocked by

`TKT-F6-001` · `TKT-F2-001` · `TKT-F2-006` (nomor klaim). **Terhalang Work Owner dan Tim Infra.**

#### Constraint keamanan, data, operasional

- **Jatuh ke koneksi default saat portal tidak diketahui adalah kebocoran data antar badan hukum.**
  Perilaku yang benar adalah menolak permintaan, bukan menebak portalnya.
- Empat pool melipatkan kebutuhan koneksi di sisi database. Bila kapasitasnya tidak disiapkan,
  portal keempat akan gagal saat portal lain sedang sibuk.
- Kredensial keempat database tunduk `D-40` dan `R-17` — dari penyimpanan rahasia, tidak pernah
  masuk repositori.
- `P-1` (satu tabel ditulis satu sistem) kini berlaku **di dalam tiap database**, dan tetap berlaku
  penuh selama Pega dan Go berjalan berdampingan (`D-05`) — **di keempat entitas**.
- **Master data pun berbeda isinya antar portal, dan itu sudah terbukti — bukan dugaan.** Work Owner
  menegaskan 2026-09-19 bahwa tiap server punya `POOLDATA`-nya sendiri dan aplikasi membaca milik
  server tempat ia berjalan. Contoh nyatanya: `POOLDATA.EMAILKOMITE` pada server ASM memuat
  `TYPE_BUSINESS` NONMBU, PA, TRAVEL, dan BONDING, sementara baris `SIMASNET`/`SIMASNETA` hanya ada
  di server Simasnet. Aturan penjenjangan komitenya pun berbeda — Simasnet memilih **satu** penyetuju
  secara acak, yang lain **kumulatif**.
- **Membaca master dari koneksi utama menghasilkan angka yang salah tanpa satu pun tanda.** Modul
  `B-7` hari ini memasang repo ambang komite pada koneksi utama
  (`cmd/claimpnc/main.go`, dicatat di sana dan di `komite/repo/sqlstore/ambang.sql`). Begitu portal
  kedua dilayani, layar akan menampilkan jenjang persetujuan **milik entitas lain**: seluruh namanya
  masuk akal, tidak ada galat, dan yang keliru hanya *siapa* yang berwenang menyetujui uang. Bentuk
  kegagalannya sama dengan `R-20`, dan tiket inilah yang menutupnya.

#### Migrasi skema / rollout / rollback

Tidak mengubah tabel klaim. Menambah kebutuhan konfigurasi per portal.

**Rollout:** satu portal pada satu waktu. Portal yang belum dialihkan tetap memakai Pega.

**Rollback:** mengembalikan lalu lintas portal itu ke Pega. Rollback **per portal**, bukan seluruhnya.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/db/... -run TestPoolPerPortal
go test ./internal/adapter/db/... -run TestKueriTidakBocorAntarPortal
go test ./internal/adapter/db/... -run TestDatabaseSatuPortalMati
go test ./internal/adapter/db/... -run TestTanpaPortalDitolakBukanDefault
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Satu database per entitas | `D-75` · `ADR-0030` |
| Alasan menolak database bersama berkolom entitas | `ADR-0030` bagian Opsi |
| Pool laporan terpisah | `docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md` butir 7 |
| Nomor klaim `PNCN.YY.xxxx` dari sequence, tanpa penanda portal | `D-71` · `D-76` |
| Satu tabel ditulis satu sistem | `P-1` · `docs/Steering/07-MIGRATION-STRATEGY.md:15` |

#### Comments

### TKT-F6-003 — Perpindahan portal dan penilaian ulang kewenangan

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan identitas** |
| **Modul** | **F-6 Portal & Multi-Sumber Data** · Gelombang: 1 · Bergantung pada: TKT-F6-002, TKT-F3-005 |
| **Requirement** | FR-F6, FR-F3 |
| **Keputusan** | D-75, D-77, D-78, D-59 |
| **ADR** | 0030, 0023 |
| **Risiko** | **R-20** |
| **Rule Pega yang digantikan** | **tidak ada padanan** — sistem lama tidak punya perpindahan portal; entitas ditentukan server yang diakses |
| **Peran penguji gerbang 2** | **PncAdmin**, **PncManagerAdmin**, dan pemegang peran di **dua entitas berbeda** |
| **Label** | `modul::F-6` `tipe::keamanan` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-6-Portal-dan-Multi-Sumber-Data/issues/03-perpindahan-portal-dan-kewenangan.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Pengguna berpindah portal **tanpa login ulang** (`D-75`), dan setiap perpindahan **menilai ulang
kewenangannya** di portal tujuan.

Nilai bisnisnya dan bahayanya adalah hal yang sama. Tanpa login ulang, satu identitas menjangkau
**empat database milik empat badan hukum**. Bila kewenangan tidak dinilai ulang saat berpindah,
seorang pengguna dapat melihat data entitas yang bukan haknya — dan itu **kebocoran data antar badan
hukum**, bukan cacat tampilan. Inilah `R-20`.

#### Ruang lingkup

- Perpindahan portal lewat **daftar pilihan (dropdown) di dalam aplikasi** — pengguna **tidak
  kembali ke halaman login** (`D-77`). Sesi tidak diputus.
- **Penilaian ulang kewenangan di server pada setiap perpindahan** (`ADR-0023`).
- Daftar portal yang ditampilkan **hanya yang menjadi hak pengguna**.
- Portal aktif melekat pada **permintaan**, bukan pada keadaan global yang dapat tertukar antar
  permintaan bersamaan.
- Pencatatan setiap perpindahan: siapa, dari portal apa ke apa, kapan.

#### Non-goal

- **Tidak** menyatukan data antar portal.
- **Tidak** menyalin kewenangan antar portal — berhak di satu portal tidak berarti berhak di portal lain.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| ~~Satu penyedia identitas, atau satu per entitas?~~ — **TERTUTUP `D-78`**: **login sama untuk semua entitas** | — | Perpindahan tanpa login ulang **mungkin secara teknis**. Tetapi satu kredensial bocor kini menjangkau **empat badan hukum** — kewenangan portal menjadi **satu-satunya pembatas** |
| **Kewenangan portal disimpan di mana?** Karena login sama (`D-78`), data "berhak atas portal mana" bersifat **lintas portal secara alamiah** dan **tidak dapat** disimpan per portal — memeriksa hak atas portal B menuntut membaca database B sebelum orang itu terbukti berhak. Usulan: **terpusat bersama identitas**, sebagai pengecualian sadar terhadap `D-75` butir 4 | **Work Owner + Keamanan Informasi** | Ini **satu-satunya kendali** yang memisahkan empat badan hukum setelah `D-78` |
| **Bolehkah satu pengguna berhak di lebih dari satu portal?** | **Work Owner** | Bila tidak, seluruh tiket ini menyusut menjadi pemilihan portal saat login |
| **Kontrak API HCC/HCQ** masih nol jejak di export | **Tim HCC/HCQ** (`R-14`) | Penghalang yang sama dengan `TKT-F3-002` |

#### Acceptance criteria

- ☐ Perpindahan dilakukan lewat **dropdown**, dan **tidak pernah** mengembalikan pengguna ke
      halaman login — diuji.
- ☐ Pengguna hanya melihat portal yang **menjadi haknya** — diuji dengan pengguna berhak di satu
      portal: portal lain tidak muncul.
- ☐ Portal yang bukan haknya **tidak dapat dipakai lewat pemanggilan langsung** — diuji dengan
      menyisipkan kode portal pada permintaan: **`403`**, bukan data.
- ☐ Kewenangan **dinilai ulang di server pada setiap perpindahan** — diuji dengan mencabut hak
      saat sesi berjalan: perpindahan berikutnya ditolak.
- ☐ Portal aktif melekat pada permintaan — diuji dengan **permintaan bersamaan dari dua portal
      berbeda oleh satu pengguna**: tidak ada data yang tertukar.
- ☐ Setiap perpindahan **tercatat** di jejak audit portal asal **dan** portal tujuan (`S-5`).
- ☐ **Perpindahan yang ditolak mempertahankan portal sebelumnya** dan menampilkan pesan — diuji:
      pengguna **tidak** dikeluarkan, dan **tidak** berakhir tanpa portal aktif (`D-77`).
- ☐ Sesi yang kedaluwarsa **tidak dapat berpindah portal** — diuji.
- ☐ Gerbang 2: UAT dijalankan pemegang peran di **dua entitas berbeda**, bukan satu orang dengan
      hak penuh — justru pemisahannya yang diuji.

#### Dependency / Blocked by

`TKT-F6-002` · `TKT-F3-005` (middleware otorisasi) · `TKT-F3-002` (adapter HCC/HCQ) ·
`TKT-S5-002` (pencatatan). **Terhalang Work Owner, Keamanan Informasi, dan Tim HCC/HCQ.**

#### Constraint keamanan, data, operasional

- **`D-78` memperberat `R-20`.** Satu kredensial bocor kini menjangkau **empat badan hukum**;
  bila tiap entitas punya login sendiri, kebocoran terbatas pada satu entitas. Batas itu hilang.
- **`R-20` adalah risiko paling serius yang dibawa `D-75`.** Kegagalannya tidak terlihat sebagai
  galat — ia terlihat sebagai data yang tampil normal, hanya milik entitas yang salah.
- Portal aktif **tidak boleh** disimpan sebagai keadaan global di server. Dua permintaan bersamaan
  dari pengguna yang sama akan saling menimpa, dan akibatnya adalah data entitas yang tertukar.
- `D-59` menetapkan kontrol berbasis **menu**, tanpa pemisahan tugas formal. Kendali portal **tidak
  boleh** ikut bersandar pada penyembunyian menu — `ADR-0023` menuntut pemeriksaan di setiap endpoint.
- **Kegagalan perpindahan wajib mengembalikan ke portal sebelumnya** (`D-77`). Dua perilaku lain
  berbahaya: berpindah meski pemeriksaan gagal adalah `R-20` itu sendiri; berakhir tanpa portal aktif
  membuat permintaan berikutnya berisiko jatuh ke koneksi default.
- Jejak audit **per portal** (`D-75`) berarti perpindahan tercatat di dua tempat. Penyelidikan
  lintas entitas menuntut empat kueri terpisah.

#### Migrasi skema / rollout / rollback

Menambah tabel hak akses portal per pengguna. Backward-compatible (`P-4`).

**Rollback:** membatasi setiap pengguna ke satu portal. Itu **mengurangi kemampuan**, tetapi
**tidak melonggarkan kendali** — arah rollback yang benar bila `R-20` terbukti.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/http/... -run TestPortalBukanHakDitolak
go test ./internal/adapter/http/... -run TestKewenanganDinilaiUlangSaatPindah
go test ./internal/adapter/http/... -run TestPermintaanBersamaanDuaPortalTidakTertukar
go test ./internal/app/audit/... -run TestPerpindahanPortalTercatatDiKeduaPortal
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Berpindah portal tanpa login ulang | `D-75` butir 5 |
| Jejak audit per portal | `D-75` butir 6 |
| Otorisasi diperiksa di setiap endpoint | `D-59` · `ADR-0023` |
| Kontrak API HCC/HCQ nol jejak di export | `R-14` · `ADR-0024` |
| `R-20` kebocoran data antar entitas | `D-75` · `ADR-0030` |

#### Comments

### TKT-F6-004 — Kesekerabatan skema empat database

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak DDL dan keputusan syariah** |
| **Modul** | **F-6 Portal & Multi-Sumber Data** · Gelombang: 1 · Bergantung pada: TKT-F6-002, TKT-F2-004 |
| **Requirement** | FR-F6 |
| **Keputusan** | D-75 |
| **ADR** | 0030, 0005 |
| **Risiko** | R-08, R-16, R-20 |
| **Rule Pega yang digantikan** | **tidak ada padanan** — sistem lama tidak pernah menjamin keempat database sekerabat |
| **Peran penguji gerbang 2** | **PncAdmin** dan **DBA** |
| **Label** | `modul::F-6` `tipe::fondasi` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/F-6-Portal-dan-Multi-Sumber-Data/issues/04-kesekerabatan-skema-empat-database.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Keempat database tetap **sekerabat skemanya**, dan penyimpangan **terdeteksi sebelum** menjadi
kegagalan di hadapan pengguna.

Nilai bisnisnya: satu basis kode melayani empat database. Bila skema salah satunya menyimpang —
kolom tertinggal, index belum dibuat, migrasi gagal separuh — maka **kode yang sama akan gagal hanya
di portal itu**, dan biasanya baru ketahuan saat petugas entitas tersebut mengerjakan klaim.

#### Ruang lingkup

- Migrasi skema (`TKT-F2-004`) dijalankan **per portal**, dengan pencatatan versi per database.
- **Pemeriksaan kesekerabatan**: membandingkan versi skema keempat database dan melaporkan selisihnya.
- Perilaku bila sebuah portal tertinggal versi: portal itu **ditolak melayani**, bukan melayani
  dengan skema yang salah.
- Urutan penerapan migrasi dan cara menghentikannya bila gagal di tengah.

#### Non-goal

- **Tidak** menyeragamkan **isi** data — master data memang **per portal** (`D-75`).
- **Tidak** menyatukan keempat database.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **DDL seluruh tabel dan statistik ukuran** | **DBA** (`R-08`) | Penghalang yang sama dengan `TKT-F2-004`, kini **berlipat empat**: keempat database perlu diketahui keadaannya, bukan satu |
| **Apakah keempat database sekarang sudah sekerabat?** | **DBA** | Bila sudah menyimpang sejak sekarang, pekerjaan pertama adalah **menyamakannya** — dan itu tiket tersendiri yang belum ada |
| **Apakah portal syariah butuh tabel yang tidak ada di portal lain?** | **Work Owner + Tim Pega** | `SpreadingSyariah_Act` membuktikan logikanya berbeda. Bila datanya juga berbeda, "sekerabat" perlu didefinisikan ulang — mungkin **inti yang sama plus tambahan per portal** |
| **Oracle dulu atau PostgreSQL dulu, per portal?** | **Work Owner + Tim Infra** (`ADR-0005`) | Keempat portal bisa berada di tahap migrasi yang berbeda, dan itu menambah kombinasi yang harus diuji |

#### Acceptance criteria

- ☐ Versi skema tercatat **per database**, dan dapat dibaca tanpa membuka database satu per satu.
- ☐ **Selisih versi antar portal terdeteksi dan dilaporkan** — diuji dengan sengaja menahan
      migrasi di satu portal.
- ☐ Portal yang **tertinggal versi ditolak melayani**, dengan pesan yang menyebut portal dan versi
      — diuji; **tidak boleh** melayani dengan skema yang salah.
- ☐ Migrasi yang **gagal di tengah pada satu portal tidak meninggalkan portal itu setengah jadi**
      — diuji dengan migrasi yang sengaja digagalkan.
- ☐ Migrasi bersifat **backward-compatible** (`P-4`) di keempat portal — diuji dengan versi kode
      lama terhadap skema baru.
- ☐ Menambah portal baru **tidak menuntut perubahan perkakas migrasi** — diuji.
- ☐ Gerbang 2: ditinjau **DBA** bersama **PncAdmin**.

#### Dependency / Blocked by

`TKT-F6-002` · `TKT-F2-004` (kerangka migrasi skema). **Terhalang DBA (`R-08`), Work Owner, dan
Tim Pega.**

#### Constraint keamanan, data, operasional

- **Tidak ada koneksi ke produksi. Tidak ada DDL/DML** dari pekerjaan ini di lingkungan produksi —
  perkakasnya dibangun dan diuji di lingkungan uji.
- Perkakas migrasi memegang kredensial **keempat** database. Ia menjadi sasaran bernilai tinggi;
  kredensialnya wajib dari penyimpanan rahasia (`D-40`, `R-17`).
- **Menjalankan migrasi ke portal yang salah adalah perubahan struktur pada database badan hukum
  lain.** Perkakasnya wajib menyebut portal secara eksplisit dan menolak default.
- `S-8` kini berjalan **per portal**, dan portal Syariah **belum punya baseline Pega yang utuh**
  karena `IsServerSyariah` hilang — gerbang 1 di portal itu diganti ukuran lain (`D-42`, `D-75`).

#### Migrasi skema / rollout / rollback

Menambah tabel versi skema di tiap database.

**Rollout:** migrasi diterapkan portal demi portal, dimulai dari portal dengan risiko terendah.

**Rollback:** membalik migrasi **per portal**. Portal yang sudah maju dan portal yang belum
**tidak boleh dilayani basis kode yang sama** tanpa pemeriksaan versi — itulah sebab acceptance
criteria ketiga ada.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/cek-skema-portal --semua        # melaporkan versi per portal
go test ./internal/adapter/db/... -run TestPortalTertinggalVersiDitolak
go test ./internal/adapter/db/... -run TestMigrasiGagalTidakSetengahJadi
go test ./internal/adapter/db/... -run TestMigrasiBackwardCompatible
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Satu database per entitas, master per portal | `D-75` · `ADR-0030` |
| DDL dan statistik ukuran belum ada | `R-08` · `TKT-F2-004` |
| Skema backward-compatible | `P-4` · `docs/Steering/09-DATABASE-STRATEGY.md` §9.1 |
| Oracle dulu, PostgreSQL kemudian | `ADR-0005` |
| `IsServerSyariah` hilang dari export | `docs/verifikasi-bukti-adr.md:788-791` |

#### Comments

# 8. B-1 · View Polis & Snapshot Polis

*Modul Bisnis Inti · folder `docs/ticketing/B-1-View-Polis-dan-Snapshot-Polis/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **View Polis** — tahap `View Polis` pada `Flow/Register_Flow.xml`, layar `ViewPolis`, `ViewPolis1`, `CheckViewPolis_act` |
| **Kode modul** | `B-1` |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | 18 activity |
| **Bergantung pada** | `F-1`…`F-5` |
| **Kesiapan** | **SEBAGIAN** |

### Apa yang dikerjakan modul ini

Menarik data polis dari sistem GISFW dan **membekukannya sebagai snapshot** pada saat klaim
diregistrasi. Sejak titik itu, klaim dinilai berdasarkan **keadaan polis saat kejadian**, bukan
keadaan polis hari ini.

Isi snapshot: nomor polis, produksi ke-berapa, tertanggung, periode pertanggungan, Group Panel,
jenis bisnis, cabang, total TSI, total premi, status pembayaran premi, daftar koasuransi, daftar
Fac Offer, dan daftar alamat kirim.

### Kenapa dibekukan, bukan dibaca langsung

Dua alasan, dan keduanya sengaja (`ADR-0006`):

1. **Benar secara bisnis** — endorsemen atau pembatalan polis setelah registrasi tidak boleh
   mengubah dasar penilaian klaim yang sudah berjalan.
2. **Benar secara operasional** — klaim tetap dapat diproses ketika GISFW sedang tidak dapat
   dihubungi, sehingga tuntutan 24/7 (`D-27`) tidak bergantung pada sistem tim lain.

### Rule Pega yang digantikan

| Jenis | Nama |
|---|---|
| Flow | `Flow/Register_Flow.xml` — tahap `View Polis` |
| Activity | `Activity/CheckViewPolis_act-Act.xml` dan 17 activity pendukung |
| Harness | `ViewPolis`, `ViewPolis1` |
| Objek DB | `JSON_POLIS`, `JSON_KLAIM`, `INSERTPOLISTOJSON` di `GLADMIN.ASMD` |

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **12 dependensi procedure** — `UPDATE_LOG_KONVERSI` (**162 pemanggilan**), `GETNEWID` (42), `PKG_COUNTER_PRODUCTION` (25), `PROCESSQUEUEDIRECT` (10) | **DBA** (`R-01`) |
| **Artefak** | 4 definisi queue `DBMS_AQ` · `JSON_MBU`, `JSON_FIRE`, `JSON_POLIS_PA`, `JSON_POLIS_TRAVEL`, `JSON_POLIS_MARINE_CARGO`, `json_aneka` · pengganti `INSERTPOLISTOJSON` | **DBA** |
| **Keputusan** | Snapshot minimal **16 field polis** — disengaja, atau harus diperluas? | **Work Owner** |
| **Keputusan** | `JSON_POLIS`/`JSON_KLAIM` sumber kebenaran atau staging? | **Work Owner** |
| **Keputusan** | Kapan snapshot boleh atau harus disegarkan, dan siapa yang berwenang memicunya? | **Work Owner** (`ADR-0006`) |
| **Keputusan** | Apakah tim GISFW menyetujui Claim PNC menyimpan salinan data mereka? | **Work Owner → Tim GISFW** |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B01-001](issues/01-layar-view-polis-dan-pembekuan-snapshot.md) | Layar View Polis dan pembekuan snapshot | `needs-info` |
| [TKT-B01-002](issues/02-skema-snapshot-polis.md) | Skema snapshot polis pengganti JSON bebas bentuk | `needs-info` |
| [TKT-B01-003](issues/03-penyegaran-snapshot.md) | Aturan penyegaran snapshot | `needs-info` |

## Daftar Tiket

### TKT-B01-001 — Layar View Polis dan pembekuan snapshot

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | terhalang keputusan |
| **Modul** | **B-1 View Polis** · Gelombang: 3 · Bergantung pada: TKT-F2-001, TKT-U2-001 |
| **Requirement** | FR-B1 |
| **Keputusan** | D-04, D-03 |
| **ADR** | 0006, 0008 |
| **Risiko** | R-01, R-03 |
| **Rule Pega yang digantikan** | `Flow/Register_Flow.xml` tahap **`View Polis`** · `Activity/CheckViewPolis_act-Act.xml` · harness `ViewPolis`, `ViewPolis1` |
| **Peran penguji gerbang 2** | **PncAdmin** dan **PncPICTeknik** |
| **Label** | `modul::B-1` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-1-View-Polis-dan-Snapshot-Polis/issues/01-layar-view-polis-dan-pembekuan-snapshot.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Petugas dapat mencari polis, melihat isinya, dan **membekukannya** sebagai dasar klaim yang sedang
didaftarkan.

Nilai bisnisnya: inilah yang membuat nilai klaim **tidak berubah diam-diam**. Tanpa pembekuan,
endorsemen polis yang terjadi minggu depan akan mengubah dasar penilaian klaim yang sudah berjalan
hari ini — dan tidak ada yang menyadarinya.

#### Ruang lingkup

- Pencarian dan tampilan polis, mengikuti susunan layar `ViewPolis`.
- **Pembekuan snapshot** pada saat klaim diregistrasi, disimpan bersama klaim dalam satu transaksi.
- Penanganan bila polis tidak ditemukan atau GISFW tidak dapat dihubungi — **pesan yang jelas**,
  bukan layar kosong.
- Menampilkan **snapshot**, bukan data polis hidup, untuk klaim yang sudah terdaftar.

#### Non-goal

- **Tidak** menulis apa pun ke data polis. Kepemilikannya tetap di GISFW (`ADR-0006`).
- **Tidak** memigrasikan GISFW (`D-03`).
- **Tidak** merancang bentuk tabel snapshot — itu `TKT-B01-002`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Snapshot minimal 16 field — disengaja atau harus diperluas?** | **Work Owner** | Field yang kurang baru ketahuan saat klaim lama dibuka bertahun kemudian, dan saat itu data aslinya mungkin sudah berubah |
| **Apakah tim GISFW menyetujui Claim PNC menyimpan salinan data mereka?** | **Work Owner → Tim GISFW** | Menyentuh kepemilikan data lintas tim |
| **Cara mengambil data polis: API GISFW atau baca tabel langsung?** | **Work Owner + Tim GISFW** (`R-03`) | Menentukan apakah modul ini bergantung pada API yang belum ada |

#### Acceptance criteria

- ☐ Polis dapat dicari berdasarkan nomor polis, dan hasilnya menampilkan **ketiga belas
      kelompok data** snapshot.
- ☐ Snapshot **tersimpan bersama klaim dalam satu transaksi** — kegagalan penyimpanan klaim
      **tidak meninggalkan snapshot yatim** (diuji).
- ☐ Membuka klaim yang sudah terdaftar menampilkan **snapshot**, bukan data polis hidup —
      dibuktikan dengan mengubah data polis di staging lalu membuka klaim: tampilannya **tidak
      berubah**.
- ☐ Polis tidak ditemukan menghasilkan pesan yang menyebut nomor polis yang dicari.
- ☐ GISFW tidak dapat dihubungi menghasilkan pesan **berbeda** dari polis tidak ditemukan —
      keduanya menuntut tindakan berbeda dari petugas.
- ☐ Susunan layar mengikuti `ViewPolis` — perbandingan berdampingan disetujui penguji gerbang 2.
- ☐ Gerbang 1: isi snapshot **sama dengan Pega** pada 20 polis contoh dari empat lini bisnis.
- ☐ Gerbang 2: UAT **PncAdmin** dan **PncPICTeknik**.

#### Dependency / Blocked by

`TKT-F2-001` · `TKT-U2-001` · `TKT-B01-002` (bentuk tabel). Terhalang tiga keputusan di atas.

#### Constraint keamanan, data, operasional

- Snapshot memuat **data nasabah**: nomor polis, nama tertanggung, NPWP. Batas data cabang dan
  lini bisnis ditegakkan di kueri (`TKT-F3-005`).
- **Claim PNC tidak pernah menulis ke data polis** — bila sebuah kueri di modul ini melakukan
  `INSERT`/`UPDATE` ke tabel polis, itu pelanggaran `P-1` dan batas bounded context sekaligus.
- Pengambilan data lintas sistem **tidak boleh berada di dalam transaksi database**
  (`TKT-F2-003`).

#### Migrasi skema / rollout / rollback

Menambah tabel snapshot milik aplikasi — tidak menyentuh tabel GISFW.

**Rollback:** snapshot yang sudah terbentuk tetap ada dan tetap sah. Klaim yang memakainya tidak
terpengaruh.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/polis/... -run TestPembekuanSnapshot
go test ./internal/app/polis/... -run TestSnapshotTidakBerubahSaatPolisBerubah
grep -rInE "(INSERT|UPDATE)\s+INTO?\s+.*polis" internal/app/polis/    # HARUS 0 baris
go run ./cmd/s8 banding --modul B-1 --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Snapshot polis dibekukan saat registrasi | `D-04` · `ADR-0006` |
| Tahap `View Polis` pada flow utama | `Flow/Register_Flow.xml` |
| Isi snapshot 13 kelompok data | `docs/Steering/05-DOMAIN-MODEL.md` §1 |
| Kepemilikan data polis tetap di GISFW | `ADR-0006` |
| 141 activity GISFW ikut di export | `D-04` |

#### Comments

### TKT-B01-002 — Skema snapshot polis pengganti JSON bebas bentuk

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** — 12 dependensi procedure belum diterima |
| **Modul** | **B-1 View Polis** · Gelombang: 3 · Bergantung pada: TKT-F2-004 |
| **Requirement** | FR-B1 |
| **Keputusan** | D-04, D-02, D-68 |
| **ADR** | 0006, 0007 |
| **Risiko** | R-01, R-08 |
| **Rule Pega yang digantikan** | `JSON_POLIS` · `JSON_KLAIM` · `INSERTPOLISTOJSON` di `GLADMIN.ASMD` · `JSON_MBU`, `JSON_FIRE`, `JSON_POLIS_PA`, `JSON_POLIS_TRAVEL`, `JSON_POLIS_MARINE_CARGO`, `json_aneka` |
| **Peran penguji gerbang 2** | **tidak berlaku** — pekerjaan skema; diuji lewat `TKT-B01-001` |
| **Label** | `modul::B-1` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-1-View-Polis-dan-Snapshot-Polis/issues/02-skema-snapshot-polis.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Snapshot polis disimpan dalam **skema eksplisit** — kolom bernama, bertipe, dan dapat di-index —
menggantikan JSON bebas bentuk.

Nilai bisnisnya: JSON bebas bentuk berarti **tidak ada yang tahu field apa saja yang benar-benar
ada** sampai seseorang membacanya satu per satu. Skema eksplisit membuat field yang hilang
ketahuan saat penyimpanan, bukan bertahun kemudian saat klaim dibuka kembali.

#### Ruang lingkup

- Skema tabel snapshot polis beserta tabel anaknya: koasuransi, Fac Offer, dan alamat kirim.
- Pemetaan dari bentuk JSON lama ke kolom baru, **dibuat dari pembacaan pemakaian di rule** —
  karena bentuk JSON-nya tidak terdokumentasi.
- Index yang mendukung pencarian klaim berdasarkan nomor polis dan tertanggung.
- Dokumentasi field yang **sengaja tidak dibawa**, beserta alasannya.

#### Non-goal

- **Tidak** memanggil stored procedure (`ADR-0007`).
- **Tidak** memakai kolom JSON sebagai tempat menyimpan data yang belum dipetakan — itu memindahkan
  masalahnya, bukan menyelesaikannya.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **12 dependensi procedure** yang dipanggil 62 procedure yang sudah diterima — terberat `UPDATE_LOG_KONVERSI` (**162 pemanggilan**), `GETNEWID` (42), `PKG_COUNTER_PRODUCTION` (25), `PROCESSQUEUEDIRECT` (10) | **DBA** (`R-01`) | Logikanya menentukan field apa yang benar-benar dibentuk saat konversi polis |
| **Pengganti `INSERTPOLISTOJSON`** di `GLADMIN.ASMD` dan **4 definisi queue `DBMS_AQ`** | **DBA** | Menentukan apakah pembentukan snapshot sinkron atau lewat antrean |
| **`JSON_POLIS`/`JSON_KLAIM`: sumber kebenaran atau staging?** | **Work Owner** | Bila sumber kebenaran, ia harus ikut dibawa; bila staging, ia boleh ditinggalkan |
| **DDL tabel** (`R-08`) | **DBA** | Tipe dan panjang kolom sumbernya tidak diketahui |

#### Acceptance criteria

- ☐ Seluruh field snapshot punya **kolom bernama dan bertipe** — **nol kolom JSON** sebagai
      penampung sisa; diuji pemindaian definisi skema.
- ☐ Pemetaan JSON lama → kolom baru tersedia sebagai berkas, dan **setiap field yang tidak
      dibawa punya baris alasan**.
- ☐ Membaca snapshot 20 klaim lama dari staging menghasilkan **nilai yang sama** dengan yang
      ditampilkan Pega — diuji.
- ☐ Pencarian klaim berdasarkan nomor polis memakai index — dilampirkan `EXPLAIN PLAN`.
- ☐ Migrasi skema backward-compatible dan punya `down.sql` yang berfungsi (`TKT-F2-004`).
- ☐ **Nol pemanggilan stored procedure** dari modul ini — diuji pemindaian.

#### Dependency / Blocked by

`TKT-F2-004`. **Terhalang DBA (`R-01`, `R-08`) dan satu keputusan Work Owner.**

#### Constraint keamanan, data, operasional

- Snapshot memuat data nasabah; aturan akses sama dengan data klaim.
- Tabel snapshot **tumbuh satu baris per klaim** — pada ribuan klaim per bulan (`D-10`), kapasitas
  dan index-nya dirancang sejak awal, bukan diperbaiki kemudian.
- Skema baru **tidak menyentuh tabel yang dibaca Pega**; keduanya hidup berdampingan selama masa
  paralel.

#### Migrasi skema / rollout / rollback

Menambah tabel baru. **Rollback:** tabel dibiarkan; aplikasi kembali membaca `JSON_POLIS`
langsung — **hanya mungkin bila** jawaban atas pertanyaan "sumber kebenaran atau staging" adalah
*sumber kebenaran*.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
migrate -path db/migrations -database "$DSN" up
go test ./internal/adapter/sqlstore/... -run TestSnapshotPolis
go run ./cmd/tools/banding-snapshot --kasus 20 --sumber staging
grep -rIn "CALL \|EXEC " internal/app/polis/     # HARUS 0 baris
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Snapshot dengan skema eksplisit, bukan JSON bebas bentuk | `D-04` · `ADR-0006` |
| 12 dependensi procedure; `UPDATE_LOG_KONVERSI` 162× | `docs/verifikasi-bukti-adr.md` §14.2 · `R-01` |
| Tanpa pemanggilan stored procedure | `D-02` · `ADR-0007` |
| DDL tabel belum tersedia | `R-08` |

#### Comments

### TKT-B01-003 — Aturan penyegaran snapshot

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan** — aturan penyegaran belum ada sama sekali |
| **Modul** | **B-1 View Polis** · Gelombang: 3 · Bergantung pada: TKT-B01-001 |
| **Requirement** | FR-B1 |
| **Keputusan** | D-04 |
| **ADR** | 0006, 0026 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | **tidak ada padanan** — sistem lama tidak punya aturan penyegaran yang tercatat di rule mana pun |
| **Peran penguji gerbang 2** | **PncPICTeknik** dan **PncManagerAdmin** |
| **Label** | `modul::B-1` `tipe::aturan-bisnis` `status::needs-info` `prioritas::sedang` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-1-View-Polis-dan-Snapshot-Polis/issues/03-penyegaran-snapshot.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Aturan yang menjawab satu pertanyaan: **kapan snapshot polis boleh diperbarui, dan siapa yang
boleh melakukannya.**

Nilai bisnisnya adalah menutup utang yang lahir dari `ADR-0006` itu sendiri. Membekukan polis benar
secara bisnis, tetapi ia **menciptakan kemungkinan snapshot usang**: bila polis dikoreksi setelah
registrasi — pembatalan, endorsemen, perbaikan data tertanggung — klaim tetap memakai salinan lama
**sampai ada yang menyegarkannya secara sadar**. Aturan itu belum ada.

#### Ruang lingkup

- Aturan kapan penyegaran **boleh** dilakukan, dan kapan ia **dilarang** (misalnya setelah komite
  menyetujui, atau setelah pembayaran).
- Siapa yang berwenang memicunya, mengikuti model izin menu (`D-59`).
- **Perbandingan sebelum dan sesudah** yang ditampilkan ke pengguna sebelum penyegaran disetujui —
  agar dampaknya terlihat, bukan diterapkan diam-diam.
- Pencatatan jejak audit: siapa menyegarkan, kapan, field apa yang berubah, dari nilai apa.

#### Non-goal

- **Tidak** menyegarkan otomatis. Penyegaran otomatis akan mengembalikan persoalan yang justru
  dihindari `ADR-0006`.
- **Tidak** mengubah data polis di GISFW.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Kapan snapshot boleh atau harus disegarkan?** | **Work Owner** | Ini aturan bisnis yang belum pernah ada — tidak dapat disimpulkan dari sistem lama karena sistem lama tidak punya konsepnya |
| **Siapa yang berwenang memicunya?** | **Work Owner** | `D-59` menetapkan izin bersatuan menu; penyegaran mengubah dasar penilaian klaim, sehingga kewenangannya bernilai tinggi |
| **Apa yang terjadi pada nilai yang sudah dihitung** dari snapshot lama — estimasi, spreading, jenjang komite? | **Work Owner** | Menentukan apakah penyegaran memicu perhitungan ulang atau hanya memperbarui tampilan |

#### Acceptance criteria

> Bentuk AC yang akan diisi setelah aturannya ditetapkan.

- ☐ Penyegaran hanya dapat dilakukan pada tahap yang diizinkan; di luar itu **ditolak** dengan
      pesan yang menyebut tahapnya.
- ☐ Sebelum disetujui, pengguna melihat **perbandingan field yang akan berubah** — nilai lama di
      samping nilai baru.
- ☐ Penyegaran menghasilkan **tepat satu baris jejak audit per field yang berubah**, berisi nilai
      sebelum dan sesudah.
- ☐ Pengguna tanpa kewenangan menerima `403` dan **tidak melihat tombolnya**.
- ☐ Snapshot yang disegarkan **tidak menghapus versi sebelumnya** — konsisten `ADR-0012`.

#### Dependency / Blocked by

`TKT-B01-001` · `TKT-S5-002`. **Terhalang tiga keputusan Work Owner.**

#### Constraint keamanan, data, operasional

- Penyegaran **mengubah dasar penilaian klaim yang sedang berjalan**. Ia tindakan bernilai tinggi,
  dan karena `D-59` tidak mengenal pemisahan tugas, **jejak audit adalah satu-satunya kontrolnya**.
- Snapshot lama **tidak dihapus** (`ADR-0012`) — yang lama ditandai tidak aktif, bukan dibuang.

#### Migrasi skema / rollout / rollback

Menambah penanda versi pada tabel snapshot (`TKT-B01-002`) — kolom *nullable*, backward-compatible.

**Rollback:** fitur penyegaran disembunyikan dari menu. Snapshot yang sudah disegarkan tetap
berlaku; versi lamanya tetap tersimpan.

#### Rencana verifikasi

> **Rencana — belum dijalankan, dan belum dapat disusun lengkap.**

```bash
go test ./internal/app/polis/... -run TestPenyegaranSnapshot
go test ./internal/app/polis/... -run TestPenyegaranTercatatDiAudit
go test ./internal/adapter/http/... -run TestIzinPenyegaran
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Snapshot dapat usang; aturan penyegaran belum ada | `ADR-0006` Negatif/utang teknis dan Pertanyaan terbuka |
| Tidak ada pemisahan tugas; audit satu-satunya kontrol | `D-59` · `ADR-0023` |
| Tidak ada penghapusan fisik | `D-66` · `ADR-0012` |

#### Comments

# 9. B-2 · Input Register — Registrasi Klaim

*Modul Bisnis Inti · folder `docs/ticketing/B-2-Input-Register/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Input Register** — tahap `InputRegister` pada `Flow/Register_Flow.xml`, layar `InboxRegister_Harness` |
| **Kode modul** | `B-2` (dipakai di ADR dan Steering) |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | **22 activity · 137 step validasi** — modul terdalam di seluruh sistem |
| **Bergantung pada** | `B-1` View Polis · `F-1`…`F-5` |
| **Kesiapan** | **SEBAGIAN** |

### Apa yang dikerjakan modul ini

Petugas mencatat klaim yang baru masuk: data kejadian, objek yang tertimpa, jaminan yang dipakai,
pembagian reasuransi, dan estimasi awal kerugian. Di ujungnya, sistem **menerbitkan nomor klaim**.

Inilah **gerbang validasi terberat di seluruh aplikasi**. Yang diperiksa di sini antara lain:

| Yang diperiksa | Aturannya |
|---|---|
| Urutan tanggal | Tanggal kejadian ≤ tanggal lapor ≤ tanggal terima dokumen ≤ hari ini |
| Tanggal kejadian dalam periode polis | ditambah toleransi **30 hari Bonding** dan **90 hari Travel/PA** |
| Klaim ganda | Polis + Objek + Lokasi; untuk PA ditambah Penyebab Kerugian `12002` |
| Objek dan coverage | Objek tanpa coverage tidak boleh tersimpan |
| Total spreading | wajib 100% (`ADR-0016`) |
| Nomor SLIK | wajib terisi untuk lini SPK / Asuransi Kredit |
| Penyebab kerugian | wajib, kecuali lini Travel |
| Nilai estimasi | tidak boleh melebihi TSI; memicu **Notice of Large Losses** bila > Rp 1 M |

### Rule Pega yang digantikan

| Jenis | Nama |
|---|---|
| Flow | `Flow/Register_Flow.xml` — tahap `Input Register` |
| Activity | `Activity/InputRegister_act-Act.xml` (**137 step**) dan 21 activity pendukung |
| Harness | `InboxRegister_Harness` |
| Ticket rule | `setToRegister_ticket` — **hilang dari export** |

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | When rule `IsAsuransiKredit` (gerbang SLIK) · `NonMBU`, `NotPA`, `ElseRCLMSIG` — percabangan flow utama | **Tim Pega** (`R-16`) |
| **Keputusan** | Toleransi **30 hari Bonding** — mati, atau BRD-nya salah? | **Work Owner** |
| **Keputusan** | Tanggal kejadian bagian dari kunci duplikasi? | **Work Owner** |
| **Keputusan** | Lokasi tidak diperiksa untuk PA dan Travel — sengaja? | **Work Owner** |
| **Keputusan** | **Jalur API/JSON melewati aturan 7/30/90 — sengaja?** | **Work Owner** |
| **Keputusan** | Validasi KTP/HP/Email dipanggil dengan argumen salah — pernah ada keluhan? | **Work Owner** |
| **Keputusan (tiket lain)** | Format nomor klaim: sequence direset tahunan? lebar tetap? (`TKT-F2-006`) | **Work Owner** |
| **ADR `Proposed`** | `ADR-0013` — pengganti pola hapus-lalu-sisip-ulang pada konversi klaim | **Work Owner + Lead Engineer** |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B02-001](issues/01-layar-input-register-dan-penerbitan-nomor-klaim.md) | Layar Input Register dan penerbitan nomor klaim | `needs-info` |
| [TKT-B02-002](issues/02-validasi-tanggal-dan-periode-polis.md) | Validasi urutan tanggal dan periode polis | `needs-info` |
| [TKT-B02-003](issues/03-validasi-klaim-ganda.md) | Validasi klaim ganda | `needs-info` |
| [TKT-B02-004](issues/04-validasi-kelengkapan-dan-notice-of-large-losses.md) | Validasi kelengkapan dan Notice of Large Losses | `ready-for-human` |
| [TKT-B02-005](issues/05-konversi-ulang-klaim-idempoten.md) | Konversi ulang klaim yang idempoten | `needs-info` |

## Daftar Tiket

### TKT-B02-001 — Layar Input Register dan penerbitan nomor klaim

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | terhalang keputusan (format nomor klaim) |
| **Modul** | **B-2 Input Register** · Gelombang: 3 · Bergantung pada: TKT-B01-001, TKT-F2-006, TKT-U2-002 |
| **Requirement** | FR-B2 |
| **Keputusan** | D-22, D-71 |
| **ADR** | 0009, 0013, 0018 |
| **Risiko** | R-16 |
| **Rule Pega yang digantikan** | `Flow/Register_Flow.xml` tahap **`Input Register`** · `Activity/InputRegister_act-Act.xml` (137 step) · harness `InboxRegister_Harness` |
| **Peran penguji gerbang 2** | **PncAdmin** (User Admin) — peran yang mengisi layar ini setiap hari |
| **Label** | `modul::B-2` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-2-Input-Register/issues/01-layar-input-register-dan-penerbitan-nomor-klaim.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Petugas admin dapat mencatat klaim baru dari awal sampai terbit **nomor klaim**, dengan layar yang
susunannya dikenali — urutan langkah dan penempatan field mengikuti Pega (`D-13`).

Nilai bisnisnya: ini **pintu masuk seluruh klaim**. Selama layar ini belum ada, tidak ada satu pun
klaim yang dapat dimulai di sistem baru, dan `P-3` menetapkan klaim yang sudah berjalan di Pega
tetap diselesaikan di Pega — jadi sistem baru hanya bisa dimulai dari sini.

#### Ruang lingkup

- Layar Input Register: data kejadian (tanggal kejadian, tanggal lapor, tanggal terima dokumen,
  lokasi, kronologi, pelapor), pemilihan polis lewat `B-1`, dan nilai estimasi awal.
- **Penerbitan nomor klaim `PNCN.YY.xxxx`** memakai generator `TKT-F2-006`.
- Penyimpanan klaim sebagai satu transaksi utuh (`TKT-F2-003`) — objek, coverage, spreading, dan
  estimasi tersimpan bersama atau tidak sama sekali.
- Tombol **Back** yang mengembalikan klaim ke tahap sebelumnya, setara perilaku status `1146` di
  sistem lama.
- Pencatatan jejak audit pada penerbitan klaim (`S-5`).

#### Non-goal

- **Tidak** memuat aturan validasi tanggal, duplikasi, dan kelengkapan — ketiganya tiket
  tersendiri (`TKT-B02-002`, `003`, `004`) agar dapat diuji dan direview terpisah.
- **Tidak** membangun layar objek/coverage — itu `B-3`.
- **Tidak** membangun spreading — itu `B-4`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Akibat |
|---|---|---|
| **Sequence nomor klaim direset tiap awal tahun?** dan **lebar segmen terakhir dibuat tetap?** (`TKT-F2-006`) | **Work Owner** | Menentukan bentuk nomor yang **tidak dapat ditarik kembali** setelah klaim pertama terbit |
| When rule `NonMBU`, `NotPA`, `ElseRCLMSIG` — percabangan flow utama | **Tim Pega** (`R-16`) | Menentukan ke tahap mana klaim diteruskan setelah register |

#### Acceptance criteria

- ☐ Klaim baru dapat disimpan dan menerima nomor berformat `PNCN.YY.xxxx` — diuji 20 penyimpanan
      berturut-turut, **nol nomor ganda**.
- ☐ Penyimpanan yang gagal di langkah mana pun **tidak meninggalkan satu baris pun** — diuji
      dengan kegagalan yang sengaja dipicu di penyimpanan coverage.
- ☐ Urutan field dan tahapan layar **sama dengan `InboxRegister_Harness`** — dibuktikan dengan
      perbandingan tangkapan layar berdampingan yang disetujui penguji gerbang 2.
- ☐ Tombol Back mengembalikan klaim ke tahap sebelumnya tanpa kehilangan isian — diuji.
- ☐ Penerbitan klaim menghasilkan **tepat satu baris jejak audit** berisi pelaku dan waktu.
- ☐ Gerbang 1: hasil penyimpanan klaim contoh **sama dengan Pega** pada 20 kasus data staging —
      selisih yang muncul wajib terpetakan ke butir `P-5` (`D-54`).
- ☐ Gerbang 2: UAT oleh **PncAdmin** pada 10 klaim nyata dari empat lini bisnis berbeda.

#### Dependency / Blocked by

`TKT-B01-001` (snapshot polis) · `TKT-F2-006` (nomor klaim) · `TKT-F2-003` (transaksi) ·
`TKT-U2-002` (form baku) · `TKT-S5-002` (jejak audit).

#### Constraint keamanan, data, operasional

- Layar ini menampilkan **data nasabah** — nomor polis, nama tertanggung, NIK. Batas data cabang
  dan lini bisnis ditegakkan di kueri (`TKT-F3-005`), bukan disaring setelah diambil.
- **Nomor klaim tidak dapat ditarik kembali** setelah terbit; ia muncul di surat ke tertanggung
  dan di PLA/DLA ke reasuransi.
- Kepemilikan tabel: selama masa paralel, **satu sistem saja yang menulis** tabel klaim (`P-1`).
  Peralihan kewenangan terjadi saat modul ini lulus gerbang 2 — bukan sebelumnya.

#### Migrasi skema / rollout / rollback

Menambah tabel klaim milik aplikasi (`ADR-0004`) dan sequence nomor klaim. Backward-compatible;
Pega tidak membacanya.

**Rollback:** klaim yang telanjur terbit di sistem baru **tetap ada dan tetap sah** — ia tidak
dapat dipindahkan ke Pega (`P-3`). Karena itu rollback modul ini berarti **menghentikan
pendaftaran klaim baru di sistem baru**, bukan membatalkan yang sudah terbit.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/registrasi/... -run TestPenerbitanNomorKlaim
go test ./internal/app/registrasi/... -run TestSimpanAtomik
go run ./cmd/s8 banding --modul B-2 --kasus 20     # gerbang 1 lewat S-8
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Tahap `Input Register` pada flow utama | `Flow/Register_Flow.xml` |
| 137 step validasi | `Activity/InputRegister_act-Act.xml` |
| Format nomor klaim `PNCN.YY.xxxx` | `D-71` · `ADR-0009` |
| Status `1146` di-set saat pengguna menekan Back | `docs/Steering/16-RISK-ANALYSIS.md` `R-06` |
| Klaim berjalan tidak berpindah sistem | `P-3` |

#### Comments

### TKT-B02-002 — Validasi urutan tanggal dan periode polis

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | terhalang keputusan |
| **Modul** | **B-2 Input Register** · Gelombang: 3 · Bergantung pada: TKT-B02-001, TKT-F5-002 |
| **Requirement** | FR-B2 |
| **Keputusan** | D-49 butir 3 |
| **ADR** | 0017 |
| **Risiko** | R-12 |
| **Rule Pega yang digantikan** | `Activity/InputRegister_act-Act.xml` — kondisi validasi periode polis pada `:5788` dan `:4805` (**penyesuaian 7 jam yang asimetris di dalam satu kondisi yang sama**) |
| **Peran penguji gerbang 2** | **PncAdmin** |
| **Label** | `modul::B-2` `tipe::aturan-bisnis` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-2-Input-Register/issues/02-validasi-tanggal-dan-periode-polis.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Aturan tanggal klaim yang **memberi hasil sama untuk kasus yang sama**, berapa kali pun diuji.

Nilai bisnisnya: hari ini hasilnya **tidak selalu sama**. Penyesuaian 7 jam diterapkan di satu
cabang kondisi tetapi tidak di cabang lainnya **di dalam kondisi validasi yang sama**, sehingga
klaim dengan tanggal kejadian tepat di batas periode polis bisa lolos atau ditolak tergantung jalur
mana yang dijalankan. Ini butir 3 pada 13 perbaikan eksplisit `P-5`.

#### Ruang lingkup

- Aturan urutan: **tanggal kejadian ≤ tanggal lapor ≤ tanggal terima dokumen ≤ hari ini**.
- Aturan periode polis: tanggal kejadian berada di dalam periode polis, ditambah toleransi
  **30 hari untuk Bonding** dan **90 hari untuk Travel dan PA**.
- Seluruh perbandingan tanggal memakai seam Clock dan konversi WIB tunggal (`F-5`) — **nol
  penyesuaian jam manual**.
- Pesan galat yang menyebut **tanggal mana** yang salah dan **batas** yang dilanggar.

#### Non-goal

- **Tidak** mengubah angka toleransi 30 dan 90 hari — bila berubah, itu keputusan bisnis
  tersendiri.
- **Tidak** menangani validasi selain tanggal.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Toleransi 30 hari Bonding — mati, atau BRD-nya salah?** Pembacaan source tidak menemukan jalur yang benar-benar memakainya | **Work Owner** | Menentukan apakah aturannya dibawa atau dihapus. Membawanya bila sebenarnya mati berarti menambah aturan yang tidak pernah berlaku |
| **Jalur API/JSON melewati aturan 7/30/90 — sengaja?** Klaim yang masuk lewat jalur itu tidak melewati validasi ini | **Work Owner** | Bila tidak sengaja, ini celah yang sudah berjalan; bila sengaja, sistem baru harus menirunya |
| **Arti `addCalendar(…, 12, 0, 0)` pada 101 titik** (`TKT-F5-002`) | **Work Owner + Tim Pega** | Menentukan hasil perbandingan pada kasus batas |

#### Acceptance criteria

- ☐ Ketiga urutan tanggal diuji pada **kasus batas**: sama persis, selisih satu hari, dan selisih
      satu detik melewati tengah malam WIB — hasilnya **deterministik**.
- ☐ Tanggal kejadian tepat pada **hari pertama** dan **hari terakhir** periode polis diterima;
      satu hari di luar ditolak — diuji keempat batas.
- ☐ Toleransi 90 hari Travel dan PA diuji pada hari ke-90 (diterima) dan ke-91 (ditolak).
- ☐ **Kedua cabang** kondisi yang di sistem lama berbeda kini memberi **hasil identik** — diuji
      khusus pada tanggal batas; inilah bukti butir 3 `P-5` benar-benar tertutup.
- ☐ **Nol penyesuaian jam manual** di kode modul ini — diuji pemindaian.
- ☐ Pesan galat menyebut tanggal yang salah dan batas yang dilanggar — diuji ketiga jenis galat.
- ☐ Gerbang 1: selisih terhadap Pega **hanya** pada kasus batas yang terpetakan ke butir 3 `P-5`.

#### Dependency / Blocked by

`TKT-B02-001`, `TKT-F5-001`, `TKT-F5-002`.

#### Constraint keamanan, data, operasional

- Perubahan ini **mengubah hasil validasi** pada kasus batas. Ia sudah terdaftar sebagai butir 3
  `P-5`, sehingga selisihnya lolos otomatis di gerbang 1 (`D-54`) — **hanya bila** selisihnya
  memang terpetakan ke butir itu.
- `R-12`: bila data yang disalin ke staging bergeser zona waktunya, pengujian ini akan melaporkan
  selisih palsu. Salinan harus diperiksa lebih dulu.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** mengembalikan aturan berarti **memulihkan cacat asimetris** —
hanya masuk akal bila ada temuan yang lebih buruk.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/registrasi/... -run TestUrutanTanggal
go test ./internal/domain/registrasi/... -run TestPeriodePolisBatas
grep -rInE "(\+|\-)\s*7\s*\*\s*time\.Hour|addCalendar" internal/domain/registrasi/   # HARUS 0
go run ./cmd/s8 banding --modul B-2 --aturan tanggal
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Penyesuaian 7 jam asimetris dalam satu kondisi | `Activity/InputRegister_act-Act.xml:5788` dan `:4805` |
| Butir 3 dari 13 perbaikan eksplisit `P-5` | `D-49` · `ADR-0017` |
| Invarian `I-2` dan `I-3` | `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Toleransi 30 hari Bonding, 90 hari Travel/PA | `BRD §11.1` |
| 101 titik `addCalendar(…,12,0,0)` | `docs/verifikasi-bukti-adr.md` §15 baris `F-5` |

#### Comments

### TKT-B02-003 — Validasi klaim ganda

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | terhalang keputusan |
| **Modul** | **B-2 Input Register** · Gelombang: 3 · Bergantung pada: TKT-B02-001 |
| **Requirement** | FR-B2 |
| **Keputusan** | D-18 |
| **ADR** | 0018 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | pemeriksaan duplikasi di `Activity/InputRegister_act-Act.xml` — kunci **Polis + Objek + Lokasi**, ditambah **Penyebab Kerugian `12002`** khusus lini PA |
| **Peran penguji gerbang 2** | **PncAdmin** |
| **Label** | `modul::B-2` `tipe::aturan-bisnis` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-2-Input-Register/issues/03-validasi-klaim-ganda.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Sistem menolak pendaftaran klaim yang **sudah pernah didaftarkan** untuk kejadian yang sama.

Nilai bisnisnya langsung ke uang: klaim ganda yang lolos berarti **satu kerugian dibayar dua
kali**. Pemeriksaan ini adalah satu-satunya pagar terhadap hal itu pada titik masuk.

#### Ruang lingkup

- Pemeriksaan duplikasi dengan kunci **Polis + Objek Pertanggungan + Lokasi**.
- Kunci tambahan **Penyebab Kerugian `12002`** untuk lini Personal Accident.
- Perilaku saat ditemukan: **menolak** dengan pesan yang menyebut **nomor klaim yang sudah ada**,
  sehingga petugas dapat memeriksanya, bukan sekadar ditolak.
- Pemeriksaan menyertakan klaim yang dibuat **kedua sistem** selama masa paralel — klaim berformat
  `PNC-xxxx` (Pega) maupun `PNCN.YY.xxxx` (Go).

#### Non-goal

- **Tidak** menggabungkan klaim ganda yang telanjur ada di data historis.
- **Tidak** memeriksa duplikasi lintas polis.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Apakah tanggal kejadian bagian dari kunci duplikasi?** Source tidak menyertakannya; artinya dua kejadian berbeda pada objek dan lokasi yang sama **dianggap duplikat** | **Work Owner** | Menentukan apakah klaim kedua yang sah ikut tertolak. Ini perilaku yang terlihat petugas setiap hari |
| **Lokasi tidak diperiksa untuk PA dan Travel — sengaja?** | **Work Owner** | Bila sengaja, aturannya memang berbeda per lini dan harus ditiru; bila tidak, ini celah yang sudah berjalan |

#### Acceptance criteria

- ☐ Klaim kedua dengan **Polis + Objek + Lokasi** yang sama **ditolak**, dan pesannya menyebut
      nomor klaim yang sudah ada — diuji.
- ☐ Untuk lini **PA**, klaim kedua dengan kunci sama **tetapi Penyebab Kerugian berbeda dari
      `12002`** diterima — diuji.
- ☐ Pemeriksaan menemukan duplikat yang dibuat **Pega** maupun **Go** — diuji dengan satu klaim
      dari masing-masing sistem.
- ☐ Klaim yang sudah **ditandai terhapus** (soft delete, `ADR-0012`) **tidak** dihitung sebagai
      duplikat — diuji.
- ☐ Pemeriksaan memakai index; rencana eksekusinya tidak memindai seluruh tabel klaim —
      dilampirkan keluaran `EXPLAIN PLAN`.
- ☐ Gerbang 1: hasil pemeriksaan **sama dengan Pega** pada 30 kasus data staging, termasuk 10
      yang di Pega dinyatakan duplikat.

#### Dependency / Blocked by

`TKT-B02-001` · `TKT-F2-005` (soft delete memengaruhi hasil pemeriksaan).

#### Constraint keamanan, data, operasional

- Pemeriksaan berjalan pada **tabel yang ditulis dua sistem** selama masa paralel — ia membaca
  klaim Pega dan klaim Go sekaligus. Itu sah karena `P-1` membatasi **penulisan**, bukan
  pembacaan.
- Pesan galat menyebut nomor klaim; **tidak** menyebut nama tertanggung atau data nasabah lain.

#### Migrasi skema / rollout / rollback

Menambah index pada kunci duplikasi — **tambah index bersifat backward-compatible** dan tidak
mengganggu Pega, tetapi pada tabel puluhan juta baris ia menuntut jendela pemeliharaan
(`TKT-F2-004`, statistik ukuran dari DBA).

**Rollback:** menghapus index; aturan tetap berjalan, hanya lebih lambat.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/registrasi/... -run TestKlaimGanda
go test ./internal/domain/registrasi/... -run TestKlaimGandaLintasSistem
go test ./internal/domain/registrasi/... -run TestSoftDeleteTidakDihitungDuplikat
go run ./cmd/s8 banding --modul B-2 --aturan duplikasi --kasus 30
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Kunci duplikasi Polis + Objek + Lokasi; PA + `12002` | Invarian `I-8`, `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Dua format nomor klaim hidup berdampingan | `ADR-0009` |
| Soft delete menyeluruh | `D-66` · `ADR-0012` |
| Penulis tunggal per tabel membatasi penulisan, bukan pembacaan | `P-1` · `ADR-0004` |

#### Comments

### TKT-B02-004 — Validasi kelengkapan dan Notice of Large Losses

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | **B-2 Input Register** · Gelombang: 3 · Bergantung pada: TKT-B02-001, TKT-F4-005 |
| **Requirement** | FR-B2 |
| **Keputusan** | D-15 |
| **ADR** | 0025 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | aturan kelengkapan di `Activity/InputRegister_act-Act.xml` · gerbang Notice of Large Losses pada `:19110` (ambang **`1000000000`** di-hardcode) · penerima di `:16154` (7 email pimpinan dalam satu string), `:16297`, `:16461`, `:16628` |
| **Peran penguji gerbang 2** | **PncAdmin** dan **PncManagerAdmin** |
| **Label** | `modul::B-2` `tipe::aturan-bisnis` `status::ready-for-human` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-2-Input-Register/issues/04-validasi-kelengkapan-dan-notice-of-large-losses.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Klaim tidak dapat disimpan dengan data yang kurang, dan **kerugian besar otomatis diberitahukan**
ke Underwriting serta jajaran pimpinan.

Nilai bisnisnya: Notice of Large Losses adalah **pemberitahuan wajib**. Di sistem lama, ambang dan
daftar penerimanya tertanam di dalam rule — mengubahnya menuntut deployment, dan **blok
`// TESTING` menimpa daftar penerima produksi** dengan alamat penguji.

#### Ruang lingkup

- Aturan kelengkapan: Penyebab Kerugian wajib (kecuali lini Travel) · Nomor SLIK wajib untuk lini
  SPK / Asuransi Kredit · objek tanpa coverage ditolak · nilai estimasi tidak melebihi TSI.
- **Notice of Large Losses**: bila estimasi setelah konversi kurs melebihi ambang, kirim
  pemberitahuan ke penerima dari **master** (`TKT-F4-003`), bukan dari kode.
- Ambang dibaca dari **master ambang uang** (`TKT-F4-005`), bukan konstanta.
- **Nol blok TESTING** — tidak ada jalur yang menimpa penerima produksi.

#### Non-goal

- **Tidak** mengirim email sendiri — pengiriman milik `S-3`; tiket ini memicu peristiwanya.
- **Tidak** memvalidasi tanggal maupun duplikasi.

#### Acceptance criteria

- ☐ Klaim tanpa Penyebab Kerugian **ditolak**, kecuali lini Travel — diuji kedua lini.
- ☐ Klaim lini SPK tanpa Nomor SLIK **ditolak** — diuji.
- ☐ Objek tanpa coverage **tidak dapat disimpan** — diuji.
- ☐ Nilai estimasi melebihi TSI coverage **ditolak**, dengan pesan yang menyebut TSI-nya.
- ☐ Estimasi **melebihi ambang** memicu **tepat satu** peristiwa Notice of Large Losses; estimasi
      tepat **pada** ambang **tidak** memicu — diuji kedua batas.
- ☐ Penerima diambil dari master; mengubah master **mengubah penerima** tanpa deployment — diuji.
- ☐ Ambang diambil dari master; mengubahnya mengubah titik pemicu — diuji.
- ☐ **Nol alamat email di kode modul ini** — diuji pemindaian; **nol jalur yang menimpa penerima**
      seperti blok `// TESTING`.
- ☐ Konversi estimasi valuta asing memakai **kurs tanggal kejadian** (`ADR-0015`); kurs tidak
      ditemukan **menolak klaim**, bukan memakai nilai bawaan.
- ☐ Gerbang 1: hasil validasi **sama dengan Pega** pada 30 kasus, kecuali selisih kurs yang
      terpetakan ke butir 8 dan 9 `P-5`.
- ☐ Gerbang 2: UAT **PncAdmin** (pengisi) dan **PncManagerAdmin** (penerima notifikasi).

#### Dependency / Blocked by

`TKT-B02-001` · `TKT-F4-003` (master penerima) · `TKT-F4-004` (kurs) · `TKT-F4-005` (master ambang).

#### Constraint keamanan, data, operasional

- Penerima notifikasi **wajib mailbox fungsional**; **tidak ada akun pribadi** (`D-67`). Sistem
  lama memuat ≥6 akun Gmail pribadi di jalur produksi.
- Blok `// TESTING` pada `Activity/InputRegister_act-Act.xml:16693`, `:16830`, `:16998`, `:17141`
  **menimpa email produksi dengan precondition identik** — tidak boleh punya padanan apa pun.
- Klaim valuta asing **dapat tertolak** bila kurs belum diisi. Ini perbaikan yang diinginkan, dan
  dampak operasionalnya harus disiapkan sebelum rilis (`ADR-0015`).

#### Migrasi skema / rollout / rollback

Tidak menambah tabel di luar master `F-4`.

**Rollback:** ambang dan penerima kembali ke nilai sebelumnya lewat master — **tanpa deployment**.
Itu justru salah satu hasil yang dikejar tiket ini.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/registrasi/... -run TestKelengkapan
go test ./internal/domain/registrasi/... -run TestNoticeLargeLossesBatas
grep -rInE "[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}" internal/domain/registrasi/  # HARUS 0
go run ./cmd/s8 banding --modul B-2 --aturan kelengkapan --kasus 30
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Ambang Large Losses `1000000000` di-hardcode | `Activity/InputRegister_act-Act.xml:19110` |
| 7 email pimpinan dalam satu string | `:16154` |
| Blok `// TESTING` menimpa penerima produksi | `:16693`, `:16830`, `:16998`, `:17141` |
| Invarian `I-4`, `I-6`, `I-9`, `I-10` | `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Kurs tanggal kejadian; tolak bila kosong | `D-48` · `ADR-0015` |
| Tidak ada akun pribadi sebagai penerima | `D-67` |

#### Comments

### TKT-B02-005 — Konversi ulang klaim yang idempoten

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang `ADR-0013` yang berstatus `Proposed`** |
| **Modul** | **B-2 Input Register** · Gelombang: 3 · Bergantung pada: TKT-B02-001, TKT-F2-005 |
| **Requirement** | FR-B2 |
| **Keputusan** | D-66 |
| **ADR** | 0012, 0013 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503` — **menghapus 12 tabel** milik satu klaim lalu menyisipkan ulang seluruh pohonnya |
| **Peran penguji gerbang 2** | **PncAdmin** |
| **Label** | `modul::B-2` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-2-Input-Register/issues/05-konversi-ulang-klaim-idempoten.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Mengulang konversi sebuah klaim **tidak menggandakan datanya**, tanpa bergantung pada penghapusan
fisik.

Nilai bisnisnya: pola lama sudah **rusak sebagian hari ini**. Pada dua tabel — `T_DLALIST` dan
`T_PLALIST` — baris `DELETE`-nya dikomentari (`:497`, `:498`) sementara `INSERT`-nya tetap aktif
(`:1296`, `:1325`). Artinya konversi ulang pada kedua tabel itu **sudah berpotensi menduplikasi
baris sekarang**, sebelum migrasi apa pun.

#### Ruang lingkup

- Mekanisme idempotensi pengganti untuk pohon data satu klaim yang mencakup **12 tabel**.
- Penutupan cacat duplikasi `T_DLALIST` dan `T_PLALIST` yang sudah berjalan.
- Pencatatan jejak audit pada konversi ulang (`S-5`) — siapa mengulang, kapan, dan atas klaim mana.

#### Non-goal

- **Tidak** memakai penghapusan fisik. `ADR-0012` menetapkan tidak ada `DELETE` pada data bernilai
  bisnis.
- **Tidak** memperbaiki data historis yang telanjur terduplikasi — itu pekerjaan DBA tersendiri.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Pola pengganti hapus-lalu-sisip-ulang** — *upsert* berbasis kunci alami, versioning dengan penanda baris aktif, atau melarang konversi ulang sama sekali (`ADR-0013`) | **Work Owner + Lead Engineer** | Ketiganya menghasilkan bentuk tabel dan AC yang berbeda. Menulis AC sekarang berarti menebak |
| **Seberapa sering konversi ulang benar-benar dijalankan di produksi, dan untuk apa?** | **Work Owner** | Menentukan apakah melarangnya (opsi paling sederhana) layak dipertimbangkan sama sekali |
| **Apakah ke-12 tabel punya kunci alami yang unik?** | **DBA + Lead Engineer** | Menentukan apakah *upsert* dapat dipakai seragam atau hanya sebagian |
| **Apakah riwayat hasil konversi sebelumnya perlu disimpan?** | **Work Owner + Compliance** | Bila ya, hanya versioning yang memenuhi |

#### Acceptance criteria

> **Bentuk** AC yang akan diisi setelah `ADR-0013` diputuskan. Butir 1 dan 2 berlaku apa pun
> pilihannya.

- ☐ Menjalankan konversi **dua kali** atas klaim yang sama menghasilkan **jumlah baris yang sama**
      pada kesembilan belas tabel terdampak — diuji.
- ☐ `T_DLALIST` dan `T_PLALIST` **tidak menduplikasi** pada konversi kedua — inilah bukti cacat
      yang sudah berjalan tertutup.
- ☐ **Nol `DELETE` fisik** pada tabel bernilai bisnis — diuji pemindaian (`TKT-F2-005`).
- ☐ Konversi ulang tercatat di jejak audit dengan pelaku dan waktu.
- ☐ Gerbang 1: **jumlah baris tidak dibandingkan langsung** — perbandingan memakai **hasil kueri
      sesuai aturan bisnis**, karena soft delete mengubah isi tabel tanpa mengubah yang dilihat
      pengguna (`docs/Steering/14-TESTING-STRATEGY.md` §6.4).

#### Dependency / Blocked by

`TKT-B02-001`, `TKT-F2-005`. **Terhalang `ADR-0013`.**

#### Constraint keamanan, data, operasional

- Konversi menyentuh **pohon data satu klaim penuh** — ia harus berada di dalam satu transaksi
  (`TKT-F2-003`), atau kegagalan di tengah meninggalkan klaim setengah jadi.
- Data yang "dihapus" tetap ada dan tetap menempati kunci alaminya (`ADR-0012`) — itu yang membuat
  *upsert* menjadi kandidat, dan sekaligus yang membuatnya rumit.

#### Migrasi skema / rollout / rollback

Bentuk perubahan skema bergantung pada pilihan `ADR-0013`: *upsert* menuntut constraint unik pada
kunci alami; versioning menuntut kolom penanda baris aktif pada 12 tabel.

**Rollback:** kembali ke hapus-lalu-sisip-ulang **tidak tersedia** — `ADR-0012` melarang
penghapusan fisik. Bila pola penggantinya bermasalah, yang dilakukan adalah **menghentikan
konversi ulang sementara**, bukan mengembalikan penghapusan.

#### Rencana verifikasi

> **Rencana — belum dijalankan, dan belum dapat disusun lengkap.**

```bash
go test ./internal/app/konversi/... -run TestKonversiUlangIdempoten
go test ./internal/app/konversi/... -run TestDLAListTidakDuplikat
grep -rIn "DELETE FROM" internal/app/konversi/     # HARUS 0 baris
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Hapus 12 tabel lalu sisip ulang | `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503` |
| `DELETE` dikomentari, `INSERT` tetap aktif | `:497`, `:498` versus `:1296`, `:1325` |
| Soft delete menyeluruh | `D-66` · `ADR-0012` |
| Pola pengganti belum diputuskan | `ADR-0013` `Proposed` |
| Uji kesetaraan tidak membandingkan `COUNT(*)` | `docs/Steering/14-TESTING-STRATEGY.md` §6.4 |

#### Comments

# 10. B-3 · Objek Pertanggungan & Coverage

*Modul Bisnis Inti · folder `docs/ticketing/B-3-Objek-Pertanggungan-dan-Coverage/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | bagian dari layar **Input Register** — section `ObjectCoverage` (54 pemakaian), `ObjectItemList` (32), `LocationList`, `t_anekalist` |
| **Kode modul** | `B-3` |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | 44 activity |
| **Bergantung pada** | `B-2` Input Register |
| **Kesiapan** | **SEBAGIAN** |

### Apa yang dikerjakan modul ini

Mencatat **apa yang tertimpa** dan **jaminan mana yang dipakai**. Satu klaim dapat memuat banyak
Objek Pertanggungan; satu objek memuat banyak Coverage; satu coverage memuat rincian item.

Bentuk objeknya **berbeda per lini bisnis** — dan perbedaan itulah inti modul ini:

| Lini | Atribut khas objek |
|---|---|
| Kendaraan | nomor rangka, nomor mesin, merek, tipe, model |
| Personal Accident | tanggal lahir, NIK, status peserta |
| Asuransi Kredit | kolektibilitas, sebab macet, hari tunggakan, suku bunga |
| Properti / Aneka | lokasi risiko, okupasi |

### Rule Pega yang digantikan

| Jenis | Nama |
|---|---|
| Section | `ObjectCoverage` (54×) · `ObjectItemList` (32×) · `LocationList` |
| Activity | 44 activity objek dan coverage |
| Objek DB | `t_anekalist` dan tabel objek per lini |

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **When rule klasifikasi produk Aneka hilang**: `IsARBusiness` (24 pemakaian), `IsAnekaPerYear` (16), `IsMBBusiness` (16), `IsMoneyInsurance` (15), `IsElectronicEquipment` (10), dan lainnya | **Tim Pega** (`R-16`) |
| **Keputusan** | Pemetaan Group Panel `003` **berkonflik**: `LocationList` versus `t_anekalist` — mana yang benar? | **Work Owner** |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B03-001](issues/01-objek-pertanggungan-per-lini-bisnis.md) | Objek Pertanggungan per lini bisnis | `needs-info` |
| [TKT-B03-002](issues/02-coverage-penyebab-kerugian-dan-rincian-item.md) | Coverage, penyebab kerugian, dan rincian item | `needs-info` |
| [TKT-B03-003](issues/03-validasi-tsi-dan-sisa-tsi.md) | Validasi TSI dan Sisa TSI | `needs-info` |

## Daftar Tiket

### TKT-B03-001 — Objek Pertanggungan per lini bisnis

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** — When rule klasifikasi produk Aneka hilang |
| **Modul** | **B-3 Objek & Coverage** · Gelombang: 3 · Bergantung pada: TKT-B02-001, TKT-U2-002 |
| **Requirement** | FR-B3 |
| **Keputusan** | D-19 |
| **ADR** | 0018 |
| **Risiko** | R-16 |
| **Rule Pega yang digantikan** | section `ObjectCoverage` (54 pemakaian) · `ObjectItemList` (32) · `LocationList` · When rule klasifikasi produk Aneka |
| **Peran penguji gerbang 2** | **PncAdmin** dan **PncPICTeknik** per lini bisnis |
| **Label** | `modul::B-3` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-3-Objek-Pertanggungan-dan-Coverage/issues/01-objek-pertanggungan-per-lini-bisnis.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Petugas dapat mencatat objek yang tertimpa, dengan **field yang sesuai lini bisnisnya** — bukan
satu form panjang berisi semua kemungkinan.

Nilai bisnisnya: objek kendaraan, orang, kredit, dan properti menuntut data yang berbeda. Form
seragam memaksa petugas mengabaikan sebagian besar field, dan itulah jalan masuk data kosong yang
baru ketahuan saat klaim dinilai.

#### Ruang lingkup

- Pencatatan objek dengan atribut umum: kode objek, nama, lokasi objek, lokasi survei, okupasi,
  jenis surveyor, kode cabang.
- **Atribut khas per lini**: kendaraan (rangka, mesin, merek, tipe, model) · orang (tanggal lahir,
  NIK, status peserta) · kredit (kolektibilitas, sebab macet, hari tunggakan, suku bunga).
- Penentuan lini bisnis dari Group Panel dan Business Type pada **snapshot polis** (`B-1`), bukan
  dari input pengguna.
- Satu klaim dapat memuat banyak objek — penambahan dan penghapusan objek sebelum klaim disimpan.

#### Non-goal

- **Tidak** membangun coverage — itu `TKT-B03-002`.
- **Tidak** menambah lini bisnis baru; hanya memindahkan yang ada.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **When rule klasifikasi produk Aneka**: `IsARBusiness` (24×), `IsAnekaPerYear` (16×), `IsMBBusiness` (16×), `IsMoneyInsurance` (15×), `IsElectronicEquipment` (10×), dan lainnya | **Tim Pega** (`R-16`) | Aturan inilah yang menentukan **bentuk objek** untuk lini Aneka. Tanpa rule-nya, klasifikasinya harus ditebak |
| **Pemetaan Group Panel `003` berkonflik**: `LocationList` versus `t_anekalist` | **Work Owner** | Dua sumber berbeda untuk hal yang sama; salah pilih berarti objek Aneka tersimpan di tempat yang salah |

#### Acceptance criteria

- ☐ Bentuk form objek **berubah mengikuti lini bisnis** pada snapshot polis — diuji pada empat
      lini: kendaraan, PA, kredit, properti.
- ☐ Atribut khas lini **tidak muncul** pada lini yang tidak memakainya — diuji.
- ☐ Satu klaim dapat memuat **lebih dari satu objek**, dan menghapus objek kedua tidak merusak
      yang pertama — diuji.
- ☐ Lini bisnis diambil dari snapshot polis, **bukan dari isian pengguna** — diuji dengan
      mengubah isian dan memastikan bentuk form tidak ikut berubah.
- ☐ **Objek tanpa coverage tidak dapat disimpan** (invarian `I-6`) — diuji.
- ☐ Gerbang 1: objek yang tersimpan **sama dengan Pega** pada 20 klaim contoh dari empat lini.
- ☐ Gerbang 2: UAT oleh **PncPICTeknik masing-masing lini**, bukan satu orang untuk semua.

#### Dependency / Blocked by

`TKT-B02-001` · `TKT-B01-001` (snapshot menentukan lini) · `TKT-U2-002`.
**Terhalang Tim Pega dan satu keputusan Work Owner.**

#### Constraint keamanan, data, operasional

- Objek lini PA memuat **NIK dan tanggal lahir** — data pribadi; batas data medis `FR-R2` berlaku
  pada coverage-nya, bukan pada objeknya.
- Jumlah objek per klaim tidak dibatasi di sistem lama; **batas wajar** perlu ditetapkan agar satu
  klaim tidak menjadi ribuan baris tanpa disadari.

#### Migrasi skema / rollout / rollback

Menambah tabel objek dan tabel atribut per lini. Backward-compatible; Pega tidak membacanya.

**Rollback:** objek yang sudah tersimpan tetap ada; pendaftaran klaim baru dihentikan di sistem
baru (sama dengan `TKT-B02-001`).

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/objek/... -run TestBentukObjekPerLini
go test ./internal/domain/objek/... -run TestObjekTanpaCoverageDitolak
go run ./cmd/s8 banding --modul B-3 --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Section `ObjectCoverage` 54×, `ObjectItemList` 32× | `docs/verifikasi-bukti-adr.md` §15 baris `U-4` |
| Atribut objek per lini | `docs/Steering/05-DOMAIN-MODEL.md` §1 |
| When rule Aneka hilang | `docs/verifikasi-bukti-adr.md` §15 baris `B-3` · `R-16` |
| Invarian `I-6` objek tanpa coverage | `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Istilah **Objek Pertanggungan** menggantikan `Object`/`ObjectList` | `CONTEXT.md` |

#### Comments

### TKT-B03-002 — Coverage, penyebab kerugian, dan rincian item

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** |
| **Modul** | **B-3 Objek & Coverage** · Gelombang: 3 · Bergantung pada: TKT-B03-001, TKT-F4-005 |
| **Requirement** | FR-B3 |
| **Keputusan** | D-19 |
| **ADR** | 0018 |
| **Risiko** | R-16 |
| **Rule Pega yang digantikan** | section `ObjectCoverage` · `ObjectItemList` · harness `CauseOfLossInbox`, `DetailCauseOfLoss`, `CauseOfLossInboxSimasOnline` |
| **Peran penguji gerbang 2** | **PncPICTeknik** per lini bisnis |
| **Label** | `modul::B-3` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-3-Objek-Pertanggungan-dan-Coverage/issues/02-coverage-penyebab-kerugian-dan-rincian-item.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Untuk setiap objek, petugas mencatat **jaminan mana yang dipakai**, **apa penyebab kerugiannya**,
dan **rincian item** yang diklaim.

Nilai bisnisnya: coverage adalah tempat TSI berada, dan TSI-lah yang membatasi nilai yang boleh
dibayarkan. Salah mencatat coverage berarti batas pembayaran yang salah — dan itu baru ketahuan di
tahap akseptasi, jauh setelah petugas lain ikut bekerja di atasnya.

#### Ruang lingkup

- Pencatatan coverage per objek: kode coverage, nama, **TSI**, sublimit TSI, penyebab kerugian,
  dan diagnosa (untuk lini kesehatan/PA).
- Penyebab kerugian dipilih dari **master** (`TKT-F4-005`), bukan diketik bebas.
- Rincian item per coverage — sparepart, biaya, dan rincian lain.
- Aturan wajib: **penyebab kerugian wajib terisi, kecuali lini Travel** (invarian `I-10`).

#### Non-goal

- **Tidak** menghitung nilai settlement — itu `B-5`.
- **Tidak** membangun spreading — itu `B-4`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **When rule klasifikasi produk Aneka** yang menentukan coverage mana yang berlaku per produk | **Tim Pega** (`R-16`) | Tanpa rule-nya, daftar coverage yang sah per produk harus ditebak |
| **Pemetaan Group Panel `003`**: `LocationList` versus `t_anekalist` | **Work Owner** | Menentukan dari tabel mana daftar coverage Aneka dibaca |

#### Acceptance criteria

- ☐ Satu objek dapat memuat **lebih dari satu coverage**, masing-masing dengan TSI sendiri —
      diuji.
- ☐ Penyebab kerugian dipilih dari master; nilai di luar master **ditolak** — diuji.
- ☐ Coverage tanpa penyebab kerugian **ditolak**, kecuali lini Travel — diuji kedua lini.
- ☐ Rincian item dapat ditambah dan dihapus, dan **totalnya tidak melebihi TSI coverage** —
      diuji pada batas.
- ☐ Diagnosa hanya muncul untuk lini yang memakainya, dan **aksesnya dibatasi peran Analyst
      Doctor dan RCL Dokter** (`FR-R2`) — diuji dengan peran lain: field tidak terlihat.
- ☐ Gerbang 1: coverage yang tersimpan **sama dengan Pega** pada 20 klaim contoh.
- ☐ Gerbang 2: UAT **PncPICTeknik** per lini.

#### Dependency / Blocked by

`TKT-B03-001` · `TKT-F4-005` (master penyebab kerugian). **Terhalang Tim Pega dan Work Owner.**

#### Constraint keamanan, data, operasional

- **Diagnosa adalah data medis.** Aksesnya dibatasi peran Analyst Doctor dan RCL Dokter (`FR-R2`),
  dan pembatasan itu berlaku **juga di staging** karena staging memuat data produksi apa adanya
  (`ADR-0029`).
- TSI coverage menjadi batas pembayaran — mengubahnya setelah klaim berjalan adalah tindakan
  bernilai tinggi dan **wajib tercatat di jejak audit**.

#### Migrasi skema / rollout / rollback

Menambah tabel coverage dan rincian item. Backward-compatible.

**Rollback:** sama dengan `TKT-B03-001`.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/coverage/... -run TestCoveragePerObjek
go test ./internal/domain/coverage/... -run TestPenyebabKerugianWajib
go test ./internal/adapter/http/... -run TestAksesDiagnosaDibatasi
go run ./cmd/s8 banding --modul B-3 --aturan coverage --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Coverage berisi TSI dan sublimit | `docs/Steering/05-DOMAIN-MODEL.md` §1 |
| Invarian `I-10` penyebab kerugian wajib kecuali Travel | idem §2 |
| Akses data medis dibatasi dua peran | `FR-R2` · `BRD §17.2` |
| Harness penyebab kerugian | `Harness/CauseOfLossInbox-Harness.xml`, `DetailCauseOfLoss-Harness.xml` |

#### Comments

### TKT-B03-003 — Validasi TSI dan Sisa TSI

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** — `ValidasiSisaTSI` tidak ada di export |
| **Modul** | **B-3 Objek & Coverage** · Gelombang: 3 · Bergantung pada: TKT-B03-002 |
| **Requirement** | FR-B3 |
| **Keputusan** | D-49 butir 8 |
| **ADR** | 0016, 0017 |
| **Risiko** | R-07 |
| **Rule Pega yang digantikan** | `ValidasiSisaTSI` — **dipanggil tetapi tidak ada di export** (`R-07`); cacat terbukti pada `:817` versus `:1489` (**`NilaiSalvage` hanya ditambahkan bila baris terakhir kebetulan bertipe salvage**) |
| **Peran penguji gerbang 2** | **PncPICTeknik** |
| **Label** | `modul::B-3` `tipe::aturan-bisnis` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-3-Objek-Pertanggungan-dan-Coverage/issues/03-validasi-tsi-dan-sisa-tsi.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Nilai yang diusulkan untuk sebuah coverage **tidak boleh melebihi kapasitas pertanggungan yang
tersisa** — TSI dikurangi akseptasi yang masih Outstanding, ditambah Salvage.

Nilai bisnisnya langsung ke uang. Dan aturan ini **terbukti cacat hari ini**: `NilaiSalvage` hanya
ditambahkan bila baris terakhir kebetulan bertipe salvage. Artinya kapasitas yang seharusnya pulih
karena salvage **kadang dihitung, kadang tidak** — tergantung urutan baris. Ini butir 8 pada 13
perbaikan eksplisit `P-5`.

#### Ruang lingkup

- Perhitungan **Sisa TSI** per Objek Pertanggungan per Coverage:
  `TSI − akumulasi akseptasi Outstanding + Salvage`.
- Validasi: nilai usulan penyelesaian **tidak melebihi** Sisa TSI.
- **Perbaikan butir 8 `P-5`**: salvage **selalu** ditambahkan, bukan hanya bila baris terakhir
  bertipe salvage.
- Perhitungan memakai **nilai presisi penuh** (`ADR-0016`), bukan nilai yang sudah dibulatkan
  untuk tampilan.

#### Non-goal

- **Tidak** menghitung nilai settlement — itu `B-5`.
- **Tidak** mencatat salvage — itu `B-12`; modul ini membacanya.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **`ValidasiSisaTSI`** — dipanggil tetapi **tidak ada di export** | **Tim Pega** (`R-07`) | Rumus lengkapnya hanya terbaca sebagian dari pemanggilnya |
| **Salvage memulihkan TSI — apa niat bisnisnya?** | **Work Owner** | Menentukan apakah perilaku ini memang dikehendaki atau kebetulan |
| **`>` atau `>=` pada perbandingan sisa TSI?** | **Work Owner** | Menentukan apakah nilai **tepat sama** dengan sisa TSI diterima atau ditolak |

#### Acceptance criteria

- ☐ Sisa TSI dihitung dengan rumus `TSI − Outstanding + Salvage` — diuji pada 10 kombinasi,
      termasuk kasus tanpa salvage dan tanpa akseptasi.
- ☐ **Salvage selalu ikut dihitung**, apa pun urutan barisnya — diuji dengan salvage di baris
      pertama, tengah, dan terakhir: **ketiganya memberi hasil sama**. Inilah bukti butir 8 `P-5`
      tertutup.
- ☐ Nilai usulan melebihi Sisa TSI **ditolak**, dengan pesan yang menyebut angka sisa TSI-nya.
- ☐ Perhitungan memakai nilai presisi penuh — diuji dengan nilai berdesimal panjang; hasilnya
      **tidak** berubah karena pembulatan tampilan.
- ☐ Gerbang 1: selisih terhadap Pega **hanya** pada kasus salvage yang terpetakan ke butir 8
      `P-5`; selisih lain **wajib dijelaskan atau dinyatakan bug** (`D-54`).
- ☐ Gerbang 2: UAT **PncPICTeknik** pada klaim yang punya salvage.

#### Dependency / Blocked by

`TKT-B03-002` · `TKT-B12-001` (data salvage). **Terhalang Tim Pega dan dua keputusan.**

#### Constraint keamanan, data, operasional

- Perhitungan ini membatasi **berapa yang boleh dibayarkan**. Kesalahan di sini tidak terlihat di
  layar mana pun sampai tahap akseptasi.
- Perubahan pada komponen perhitungan (TSI, akseptasi, salvage) **wajib tercatat di jejak audit**.

#### Migrasi skema / rollout / rollback

Tidak menambah tabel; membaca tabel coverage, akseptasi, dan salvage.

**Rollback:** mengembalikan aturan berarti **memulihkan cacat butir 8** — hanya masuk akal bila
ada temuan yang lebih buruk.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/tsi/... -run TestSisaTSI
go test ./internal/domain/tsi/... -run TestSalvageSelaluDihitung
go test ./internal/domain/tsi/... -run TestPresisiPenuh
go run ./cmd/s8 banding --modul B-3 --aturan sisa-tsi
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `NilaiSalvage` hanya ditambahkan bila baris terakhir bertipe salvage | `ValidasiSisaTSI:817` versus `:1489` · `D-49` butir 8 |
| Definisi Sisa TSI | `CONTEXT.md` — **Sisa TSI** |
| `ValidasiSisaTSI` tidak ada di export | `R-07` |
| Uang disimpan presisi penuh | `D-51` · `ADR-0016` |

#### Comments

# 11. B-4 · Spreading Reasuransi & Koasuransi

*Modul Bisnis Inti · folder `docs/ticketing/B-4-Spreading-Reasuransi/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | bagian layar **Input Register** — blok spreading pada `Activity/InputRegister_act-Act.xml`; inbox `InboxClaimTreaty_Harness`, `InboxClaimNonProp_Harness`, `Inbox_XOL_Harness` |
| **Kode modul** | `B-4` |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | bagian dari 69 activity PLA/DLA/spreading |
| **Bergantung pada** | `B-3` Objek & Coverage |
| **Kesiapan** | **SEBAGIAN** |

### Apa yang dikerjakan modul ini

Membagi risiko satu Coverage kepada para penanggung: treaty, Fac Out, XOL, koasuransi, dan share
ASM sendiri. **Total pembagiannya wajib 100%.**

### Aturan yang mengikat

| Aturan | Isi |
|---|---|
| Total 100% | diuji `ROUND(SUM(share),4) BETWEEN 99.9999 AND 100.0001` (`ADR-0016`) |
| Fac Out | bila ada spreading Fac Out, data Fac Offer **wajib ada** |
| Group Panel `003` | Fac Offer wajib menyertakan Object Name |
| Ex-Gratia | klaim bertanda Ex-Gratia mengubah treaty `OR` menjadi `ORS` otomatis |
| Baris terhapus | spreading bertanda hapus **dibuang sebelum** perhitungan |

### Cacat yang ditutup modul ini

Toleransi 100% di sistem lama adalah **pencocokan substring**, bukan perbandingan angka:

```
@contains(local.totalspreading, 100.0) || local.totalspreading == 100 || @contains(local.totalspreading, 99.99)
```
`Activity/InputRegister_act-Act.xml:13183`

Akibatnya total **`199.99`** dan **`1100.0`** ikut lolos — keduanya memuat potongan teks yang
dicari. Ini butir 1 pada 13 perbaikan eksplisit `P-5`.

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | Berapa desimal share yang sah? | **Work Owner** |
| **Keputusan** | Group Panel `003`: cek **semua baris** Fac Offer atau hanya baris pertama? | **Work Owner** |
| **Keputusan** | Pemetaan kode `10001` / `10007` / `10015` | **Work Owner** |
| **Keputusan** | **`UPDATEREAS` ber-4 `COMMIT` — boleh ditulis ulang?** | **Work Owner** (`D-68` menjawab prinsipnya; konfirmasi per objek) |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B04-001](issues/01-pembagian-share-dan-validasi-total-100.md) | Pembagian share dan validasi total 100% | `needs-info` |
| [TKT-B04-002](issues/02-fac-out-fac-offer-dan-ex-gratia.md) | Fac Out, Fac Offer, dan Ex-Gratia | `needs-info` |
| [TKT-B04-003](issues/03-penulisan-spreading-atomik.md) | Penulisan spreading sebagai satu transaksi | `needs-info` |

## Daftar Tiket

### TKT-B04-001 — Pembagian share dan validasi total 100%

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | terhalang keputusan (jumlah desimal share) |
| **Modul** | **B-4 Spreading Reasuransi** · Gelombang: 3 · Bergantung pada: TKT-B03-002, TKT-U2-004 |
| **Requirement** | FR-B4 |
| **Keputusan** | D-49 butir 1, D-51 |
| **ADR** | 0016, 0017 |
| **Risiko** | R-19 |
| **Rule Pega yang digantikan** | `Activity/InputRegister_act-Act.xml:13183` — `@contains(local.totalspreading, 100.0) \|\| local.totalspreading == 100 \|\| @contains(local.totalspreading, 99.99)` |
| **Peran penguji gerbang 2** | **PncPICTeknik** dan **TreatyIn** |
| **Label** | `modul::B-4` `tipe::aturan-bisnis` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-4-Spreading-Reasuransi/issues/01-pembagian-share-dan-validasi-total-100.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Pembagian risiko satu Coverage kepada para penanggung, dengan validasi total **100% yang benar
secara angka**.

Nilai bisnisnya konkret: hari ini total **`199.99`** dan **`1100.0`** lolos validasi, karena yang
diperiksa adalah **apakah teksnya memuat potongan "100.0" atau "99.99"** — bukan berapa jumlahnya.
Spreading yang totalnya 199,99% berarti risiko dibagikan hampir dua kali lipat, dan itu memengaruhi
berapa yang ditagihkan ke reasuransi.

#### Ruang lingkup

- Layar pembagian share per Coverage: jenis treaty, nama treaty, persentase share, premi terbagi,
  dan penanggung.
- **Validasi total**: `ROUND(SUM(share), 4) BETWEEN 99.9999 AND 100.0001` (`ADR-0016`).
- **Baris bertanda hapus dibuang sebelum perhitungan** — soft delete (`ADR-0012`) membuat baris itu
  tetap ada, sehingga penyaringannya wajib eksplisit di perhitungan ini.
- Share disimpan **presisi penuh**; pembulatan hanya saat ditampilkan (`TKT-U2-004`).

#### Non-goal

- **Tidak** menangani Fac Offer dan Ex-Gratia — itu `TKT-B04-002`.
- **Tidak** menghitung nilai settlement — itu `B-5`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Berapa desimal share yang sah?** Toleransi validasi sudah 4 desimal, tetapi berapa desimal yang boleh **diketik** pengguna belum ditetapkan | **Work Owner** | Menentukan validasi masukan dan tampilan; salah pilih membuat total sah tertolak karena pembulatan masukan |

#### Acceptance criteria

- ☐ Total **`100.0000`** diterima · **`99.9999`** diterima · **`100.0001`** diterima — ketiganya
      diuji.
- ☐ Total **`199.99`** ditolak · **`1100.0`** ditolak — keduanya diuji. **Inilah bukti butir 1
      `P-5` tertutup.**
- ☐ Total `99.9998` dan `100.0002` ditolak — batas luar toleransi.
- ☐ Baris spreading bertanda hapus **tidak ikut dihitung** — diuji dengan satu baris dihapus dari
      tiga baris yang totalnya 100%.
- ☐ Share disimpan presisi penuh: nilai yang diketik **sama persis** dengan yang tersimpan —
      diuji dengan 4 desimal.
- ☐ Pesan galat menyebut **total yang dihitung** dan batas yang dilanggar, bukan sekadar
      "spreading tidak valid".
- ☐ Gerbang 1: selisih terhadap Pega **hanya** pada klaim yang totalnya di luar toleransi —
      terpetakan ke butir 1 `P-5` (`D-54`).
- ☐ Gerbang 2: UAT **PncPICTeknik** dan **TreatyIn**.

#### Dependency / Blocked by

`TKT-B03-002` · `TKT-U2-004` (pemformatan) · `TKT-F2-005` (soft delete).

#### Constraint keamanan, data, operasional

- **Data historis tetap mengandung akibat cacat ini.** Memperbaiki aturannya tidak memperbaiki
  klaim yang telanjur tersimpan dengan total di luar toleransi — perlakuan atas data itu adalah
  keputusan tersendiri (`ADR-0017`).
- Angka pecahan di JavaScript **tidak presisi**; share dikirim dan diterima sebagai string desimal
  (`TKT-U2-004`).

#### Migrasi skema / rollout / rollback

Menambah tabel spreading per coverage. Backward-compatible.

**Rollback:** mengembalikan validasi lama berarti **memulihkan cacat butir 1** — hanya masuk akal
bila ada temuan yang lebih buruk.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/spreading/... -run TestTotalSeratusPersen
go test ./internal/domain/spreading/... -run TestTolakSubstringPalsu   # 199.99 dan 1100.0
go test ./internal/domain/spreading/... -run TestBarisTerhapusTidakDihitung
go run ./cmd/s8 banding --modul B-4 --aturan total-share
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Toleransi berupa pencocokan substring | `Activity/InputRegister_act-Act.xml:13183` |
| Toleransi baru 4 desimal | `D-51` · `ADR-0016` |
| Butir 1 dari 13 perbaikan eksplisit `P-5` | `D-49` · `ADR-0017` |
| Invarian `I-1` | `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Baris bertanda hapus dibuang sebelum perhitungan | `BRD §11.3` |

#### Comments

### TKT-B04-002 — Fac Out, Fac Offer, dan Ex-Gratia

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | terhalang keputusan |
| **Modul** | **B-4 Spreading Reasuransi** · Gelombang: 3 · Bergantung pada: TKT-B04-001 |
| **Requirement** | FR-B4 |
| **Keputusan** | D-19 |
| **ADR** | 0016 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | aturan Fac Out dan Ex-Gratia di `Activity/InputRegister_act-Act.xml`; daftar Fac Offer pada snapshot polis (`B-1`) |
| **Peran penguji gerbang 2** | **PncPICTeknik** dan **TreatyIn** |
| **Label** | `modul::B-4` `tipe::aturan-bisnis` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-4-Spreading-Reasuransi/issues/02-fac-out-fac-offer-dan-ex-gratia.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Tiga aturan reasuransi yang saling berkait ditegakkan konsisten: kelengkapan Fac Offer, kelengkapan
Object Name untuk Group Panel `003`, dan perubahan otomatis treaty saat klaim ditandai Ex-Gratia.

Nilai bisnisnya: ketiganya menentukan **siapa yang menanggung dan dengan perjanjian apa**. Fac Out
tanpa Fac Offer berarti klaim dibagikan ke penanggung yang penawarannya belum tercatat.

#### Ruang lingkup

- Validasi: bila ada spreading **Fac Out**, data **Fac Offer wajib ada** pada snapshot polis.
- Validasi khusus **Group Panel `003`**: Fac Offer wajib menyertakan **Object Name**.
- Perubahan otomatis: klaim bertanda **Ex-Gratia** mengubah jenis treaty **`OR` menjadi `ORS`**.
- Pencatatan jejak audit pada perubahan otomatis tersebut — karena ia mengubah data tanpa tindakan
  pengguna.

#### Non-goal

- **Tidak** memvalidasi total share — itu `TKT-B04-001`.
- **Tidak** menerbitkan PLA/DLA ke penanggung — itu `B-9`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Group Panel `003`: cek semua baris Fac Offer atau hanya baris pertama?** Source hanya memeriksa satu baris | **Work Owner** | Bila seharusnya semua baris, ini celah yang sudah berjalan; bila memang satu baris, sistem baru harus menirunya |
| **Pemetaan kode `10001` / `10007` / `10015`** | **Work Owner** | Ketiganya muncul sebagai kode tanpa keterangan; artinya tidak dapat disimpulkan dari source |

#### Acceptance criteria

- ☐ Spreading bertipe **Fac Out** tanpa Fac Offer pada snapshot **ditolak**, dengan pesan yang
      menyebutnya — diuji.
- ☐ Untuk **Group Panel `003`**, Fac Offer tanpa Object Name **ditolak** — diuji.
- ☐ Klaim ditandai **Ex-Gratia** mengubah treaty `OR` menjadi `ORS` **otomatis**, dan perubahan
      itu **tercatat di jejak audit** — diuji keduanya.
- ☐ Menghapus tanda Ex-Gratia **tidak** mengembalikan `ORS` menjadi `OR` secara diam-diam —
      perilakunya ditetapkan eksplisit dan diuji.
- ☐ Kode `10001`, `10007`, `10015` diperlakukan sesuai keputusan Work Owner — diuji ketiganya.
- ☐ Gerbang 1: hasil validasi **sama dengan Pega** pada 20 klaim contoh yang memuat Fac Out.
- ☐ Gerbang 2: UAT **TreatyIn**.

#### Dependency / Blocked by

`TKT-B04-001` · `TKT-B01-001` (Fac Offer ada di snapshot polis). **Terhalang dua keputusan.**

#### Constraint keamanan, data, operasional

- Perubahan otomatis `OR` → `ORS` mengubah data **tanpa tindakan pengguna**. Karena `D-59` tidak
  mengenal pemisahan tugas, pencatatan auditnya **wajib** — itulah satu-satunya jejak bahwa
  perubahan itu terjadi.
- Fac Offer berada di **snapshot polis**, bukan data polis hidup (`ADR-0006`) — bila Fac Offer
  ditambahkan di GISFW setelah registrasi, ia **tidak muncul** sampai snapshot disegarkan
  (`TKT-B01-003`).

#### Migrasi skema / rollout / rollback

Tidak menambah tabel di luar `TKT-B04-001`.

**Rollback:** mengembalikan aturan; data yang telanjur berubah menjadi `ORS` **tidak dikembalikan
otomatis** — itu keputusan tersendiri.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/spreading/... -run TestFacOutWajibFacOffer
go test ./internal/domain/spreading/... -run TestGroupPanel003ObjectName
go test ./internal/domain/spreading/... -run TestExGratiaMengubahTreaty
go run ./cmd/s8 banding --modul B-4 --aturan fac-offer --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Fac Out wajib Fac Offer; Group Panel `003` wajib Object Name; Ex-Gratia `OR`→`ORS` | `BRD §11.3` |
| Fac Offer berada di snapshot polis | `docs/Steering/05-DOMAIN-MODEL.md` §1 |
| Istilah Fac Out, XOL, Ex-Gratia | `CONTEXT.md` |
| Pertanyaan terbuka Group Panel `003` dan kode `10001`/`10007`/`10015` | `docs/verifikasi-bukti-adr.md` §15 baris `B-4` |

#### Comments

### TKT-B04-003 — Penulisan spreading sebagai satu transaksi

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | terhalang keputusan (konfirmasi penulisan ulang `UPDATEREAS`) |
| **Modul** | **B-4 Spreading Reasuransi** · Gelombang: 3 · Bergantung pada: TKT-B04-001, TKT-F2-003 |
| **Requirement** | FR-B4 |
| **Keputusan** | D-02, D-68 |
| **ADR** | 0007 |
| **Risiko** | R-01 |
| **Rule Pega yang digantikan** | `Database/UPDATEREAS.prc` — **4 `COMMIT` sendiri**, sehingga penulisan spreading tidak pernah atomik |
| **Peran penguji gerbang 2** | **PncPICTeknik** |
| **Label** | `modul::B-4` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-4-Spreading-Reasuransi/issues/03-penulisan-spreading-atomik.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Penulisan spreading satu Coverage **berhasil seluruhnya atau gagal seluruhnya**.

Nilai bisnisnya: `UPDATEREAS` melakukan `COMMIT` **empat kali**. Bila proses berhenti di tengah,
sebagian baris spreading tersimpan dan sebagian tidak — dan totalnya **tidak lagi 100%** tanpa ada
yang menyadarinya, karena validasi total sudah lewat sebelum penulisan dimulai.

#### Ruang lingkup

- Penulisan seluruh baris spreading satu Coverage di dalam **satu transaksi** (`TKT-F2-003`).
- Penulisan ulang logika `UPDATEREAS` di Go — **tanpa memanggil procedure** (`ADR-0007`).
- Validasi total 100% dijalankan **di dalam transaksi yang sama**, sebagai pagar terakhir sebelum
  commit.

#### Non-goal

- **Tidak** mematikan `UPDATEREAS` di database — itu langkah tersendiri yang menunggu verifikasi
  `ALL_DEPENDENCIES` (`D-68`).
- **Tidak** menulis ulang procedure lain.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum dikonfirmasi | Pemilik | Kenapa menahan |
|---|---|---|
| **`UPDATEREAS` ber-4 `COMMIT` — boleh ditulis ulang?** `D-68` sudah memutuskan prinsipnya (Claim PNC satu-satunya pemanggil), tetapi konfirmasi per objek belum ada | **Work Owner + DBA** | Bila ternyata ada pemanggil lain, mematikannya merusak sistem itu. Verifikasi cukup satu kueri `ALL_DEPENDENCIES` |
| **12 dependensi procedure** yang dipanggil procedure yang sudah diterima | **DBA** (`R-01`) | Logika di dalamnya mungkin ikut dipanggil `UPDATEREAS` |

#### Acceptance criteria

- ☐ Penulisan spreading dengan **lima baris** yang gagal di baris keempat **tidak meninggalkan
      satu baris pun** — diuji.
- ☐ Validasi total 100% dijalankan **di dalam transaksi**; total yang tidak sah **membatalkan
      seluruh penulisan** — diuji.
- ☐ **Nol pemanggilan stored procedure** dari modul ini — diuji pemindaian.
- ☐ Lama transaksi tercatat; transaksi melebihi ambang menghasilkan log `warn` (`TKT-F2-003`).
- ☐ Gerbang 1: **perilaku saat gagal berbeda dengan Pega secara sengaja** — sistem lama
      meninggalkan sebagian data, sistem baru tidak. Kasus uji dirancang menyadari ini, atau ia
      melaporkan **selisih palsu** (`docs/Steering/14-TESTING-STRATEGY.md` §6.4).
- ☐ Gerbang 2: UAT **PncPICTeknik** pada klaim dengan spreading banyak baris.

#### Dependency / Blocked by

`TKT-B04-001` · `TKT-F2-003`. **Terhalang konfirmasi Work Owner dan DBA.**

#### Constraint keamanan, data, operasional

- Transaksi yang lebih panjang menahan kunci baris lebih lama. Dengan 200–300 pengguna (`D-10`)
  risikonya kecil, tetapi **tidak nol** bila spreading ditulis dalam proses massal.
- **Kontrak galat berbasis string `ErrMsg` tidak dibawa** (`ADR-0007`).

#### Migrasi skema / rollout / rollback

Tidak menambah tabel di luar `TKT-B04-001`.

**Rollback:** kembali memanggil `UPDATEREAS` **tidak tersedia** — `ADR-0007` menetapkan aplikasi
tidak memanggil stored procedure. Bila penulisan ulang bermasalah, yang dilakukan adalah menahan
rilis modul ini.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/spreading/... -run TestPenulisanAtomik
go test ./internal/app/spreading/... -run TestValidasiDidalamTransaksi
grep -rIn "CALL \|EXEC \|UPDATEREAS" internal/app/spreading/     # HARUS 0 baris
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `UPDATEREAS` melakukan 4 `COMMIT` | `Database/UPDATEREAS.prc` · `D-68` |
| `B-4` dapat dibuat atomik setelah logika naik ke Go | `D-68` · `ADR-0007` |
| Claim PNC satu-satunya pemanggil — **belum diverifikasi katalog** | `D-68` |
| Perilaku saat gagal berbeda secara sengaja | `docs/Steering/14-TESTING-STRATEGY.md` §6.4 |

#### Comments

# 12. B-5 · Input Estimasi & Penyelesaian Nilai Klaim

*Modul Bisnis Inti · folder `docs/ticketing/B-5-Input-Estimasi-dan-Penyelesaian-Nilai/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Input Estimasi** / **Estimation** — tahap `Estimation` dan `Input Estimasi` pada `Flow/Register_Flow.xml`; `AdjustmentList` di data |
| **Kode modul** | `B-5` |
| **Gelombang** | 5 — Nilai dan pihak luar |
| **Ukuran** | 29 activity |
| **Bergantung pada** | `B-3` Objek & Coverage · `B-4` Spreading |
| **Kesiapan** | **TERHALANG** — satu-satunya modul yang **masih terikat `BRD §21.4`** |

### Apa yang dikerjakan modul ini

Menjalankan perjalanan nilai uang klaim, dari perkiraan sampai dibayar:

```
Estimasi  →  Usulan (Propose)  →  Akseptasi (Accepted)  →  Dibayar (Paid)
```

Setiap baris nilai melekat pada satu **Coverage**, dan di sistem baru bernama **Settlement Line**
— menggantikan istilah lama `AdjustmentList`, yang menyesatkan karena dalam praktik asuransi
"adjusting" berarti proses penilaian kerugian, bukan nilai penyelesaian.

Termasuk di dalamnya: **salvage**, **risiko sendiri**, dan **fee adjuster**.

### Kenapa modul ini paling sensitif

Di sinilah angka yang dibayarkan ditetapkan. Tiga dari 13 perbaikan eksplisit `P-5` berada di
modul ini: basis kurs, perilaku saat kurs tidak ditemukan, dan perhitungan salvage pada sisa TSI.

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **isi `POOLDATA.GCNM_FEE_SCALE` (17 pita)** | **DBA** (`R-19`) |
| **Artefak** | **isi `m_currencystandard`** | **DBA** (`R-19`) |
| **Artefak** | `BrowseT_Claim_Adjustment_SQL` | **DBA** |
| **Keputusan** | Berapa desimal pembulatan fee? | **Work Owner** |
| **Keputusan** | Rp 1.650.000 dan 2% masih berlaku? | **Work Owner** |
| **Keputusan** | Salvage memulihkan TSI — apa niat bisnisnya? | **Work Owner** |
| **Keputusan** | `>` atau `>=` pada perbandingan sisa TSI? | **Work Owner** |

**Sudah tidak perlu ditanyakan lagi:** basis kurs adalah **tanggal kejadian**, dan kurs tidak
ditemukan berarti **klaim ditolak** (`D-48`, `ADR-0015`).

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B05-001](issues/01-layar-input-estimasi-dan-settlement-line.md) | Layar Input Estimasi dan Settlement Line | `needs-info` |
| [TKT-B05-002](issues/02-konversi-kurs-dan-perbandingan-ambang.md) | Konversi kurs dan perbandingan terhadap ambang | `needs-info` |
| [TKT-B05-003](issues/03-fee-adjuster-dan-risiko-sendiri.md) | Fee adjuster, risiko sendiri, dan salvage pada nilai | `needs-info` |

## Daftar Tiket

### TKT-B05-001 — Layar Input Estimasi dan Settlement Line

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** — `BrowseT_Claim_Adjustment_SQL` belum ada |
| **Modul** | **B-5 Input Estimasi** · Gelombang: 5 · Bergantung pada: TKT-B03-002, TKT-B04-001 |
| **Requirement** | FR-B5 |
| **Keputusan** | D-19, D-51 |
| **ADR** | 0016, 0018 |
| **Risiko** | R-01, R-19 |
| **Rule Pega yang digantikan** | tahap **`Estimation`** dan **`Input Estimasi`** pada `Flow/Register_Flow.xml` · 29 activity · `AdjustmentList` di data · `BrowseT_Claim_Adjustment_SQL` |
| **Peran penguji gerbang 2** | **PncPICTeknik** dan **PncManagerAdmin** |
| **Label** | `modul::B-5` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::5` |
| **Milestone** | Gelombang 5 — Nilai dan pihak luar |
| **Berkas sumber** | `ticketing/B-5-Input-Estimasi-dan-Penyelesaian-Nilai/issues/01-layar-input-estimasi-dan-settlement-line.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Petugas teknis mencatat nilai kerugian per Coverage, dan nilai itu berjalan melewati empat tahap
yang terlacak: **Estimasi → Usulan → Akseptasi → Dibayar**.

Nilai bisnisnya: keempat tahap itu adalah **riwayat uang klaim**. Di sistem lama ia tersimpan
sebagai `AdjustmentList` tanpa jejak perubahan nilai sama sekali (`T-14`) — sehingga pertanyaan
"siapa yang menurunkan nilai ini dari 100 juta menjadi 60 juta" **tidak dapat dijawab**.

#### Ruang lingkup

- Layar Input Estimasi per Coverage, dengan empat kolom nilai dan statusnya.
- Penyimpanan nilai **presisi penuh**; pembulatan hanya saat ditampilkan (`ADR-0016`).
- **Setiap perubahan nilai tercatat di jejak audit** dengan nilai sebelum dan sesudah (`S-5`) —
  inilah yang tidak ada di sistem lama.
- Validasi: nilai tidak melebihi **Sisa TSI** (`TKT-B03-003`) dan tidak melebihi TSI Coverage
  (invarian `I-4`).

#### Non-goal

- **Tidak** mengonversi kurs — itu `TKT-B05-002`.
- **Tidak** menghitung fee adjuster — itu `TKT-B05-003`.
- **Tidak** menjalankan komite — itu `B-7`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **`BrowseT_Claim_Adjustment_SQL`** — kueri pembaca daftar settlement line | **DBA** (`D-55`) | Bentuk data dan penyaringannya tidak diketahui |
| **`>` atau `>=` pada perbandingan sisa TSI** | **Work Owner** | Menentukan apakah nilai **tepat sama** dengan sisa TSI diterima |

#### Acceptance criteria

- ☐ Satu Coverage dapat memuat **lebih dari satu Settlement Line** — diuji.
- ☐ Keempat nilai (estimasi, usulan, akseptasi, dibayar) tersimpan terpisah dan **tidak saling
      menimpa** — diuji dengan mengisi keempatnya berurutan.
- ☐ Nilai tersimpan **presisi penuh**: nilai 4 desimal yang diketik sama persis dengan yang
      dibaca kembali — diuji.
- ☐ Setiap perubahan nilai menghasilkan **tepat satu baris jejak audit** berisi nilai sebelum,
      sesudah, pelaku, dan waktu — diuji.
- ☐ Nilai melebihi TSI Coverage **ditolak** dengan pesan yang menyebut TSI-nya.
- ☐ Nilai melebihi **Sisa TSI** ditolak (`TKT-B03-003`).
- ☐ Istilah di layar memakai **Settlement**, bukan `Adjustment` (`CONTEXT.md`).
- ☐ Gerbang 1: nilai yang tersimpan **sama dengan Pega** pada 20 klaim contoh.
- ☐ Gerbang 2: UAT **PncPICTeknik**.

#### Dependency / Blocked by

`TKT-B03-002`, `TKT-B03-003`, `TKT-B04-001`, `TKT-S5-002`. **Terhalang DBA dan satu keputusan.**

#### Constraint keamanan, data, operasional

- **Setiap perubahan nilai wajib diaudit** — `BRD §21.2` kriteria #9 berlaku tanpa pengecualian
  karena `D-59` menghapus pemisahan tugas.
- Orang yang sama dapat mengubah nilai **dan** menyetujuinya bila perannya memiliki kedua menu
  (`ADR-0023`). Tidak ada kontrol teknis yang mencegahnya; jejak audit satu-satunya pengimbang.
- Nilai uang dikirim dan diterima sebagai **string desimal** (`TKT-U2-004`).

#### Migrasi skema / rollout / rollback

Menambah tabel Settlement Line. Backward-compatible.

**Rollback:** nilai yang tersimpan tetap ada. Karena klaim tidak berpindah sistem di tengah jalan
(`P-3`), rollback berarti klaim baru tidak masuk ke sistem baru — bukan memindahkan yang ada.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/settlement/... -run TestEmpatTahapNilai
go test ./internal/domain/settlement/... -run TestPresisiPenuh
go test ./internal/app/settlement/... -run TestPerubahanNilaiTercatat
go run ./cmd/s8 banding --modul B-5 --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Perjalanan nilai empat tahap | `CONTEXT.md` — **Settlement Line** |
| Istilah `Adjustment` ditinggalkan | `CONTEXT.md` — Istilah yang sengaja tidak dipakai lagi |
| Perubahan nilai tidak punya jejak audit di sistem lama | `T-14` |
| Invarian `I-4` nilai tidak melebihi TSI | `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| `BrowseT_Claim_Adjustment_SQL` belum tersedia | `D-55` · `R-19` |

#### Comments

### TKT-B05-002 — Konversi kurs dan perbandingan terhadap ambang

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** — isi `m_currencystandard` belum ada |
| **Modul** | **B-5 Input Estimasi** · Gelombang: 5 · Bergantung pada: TKT-B05-001, TKT-F4-004 |
| **Requirement** | FR-B5 |
| **Keputusan** | D-48, D-49 butir 4 dan 5 |
| **ADR** | 0015, 0016, 0017 |
| **Risiko** | R-19 |
| **Rule Pega yang digantikan** | `Database/GETCURRENCYSTANDARD.fnc` — kurs **hari eksekusi** (`:3` versus `:14`) dan **`RETURN 1`** saat kurs tidak ditemukan (`:20-22`) |
| **Peran penguji gerbang 2** | **PncPICTeknik** dan **PNCKomite** |
| **Label** | `modul::B-5` `tipe::aturan-bisnis` `status::needs-info` `prioritas::tinggi` `gelombang::5` |
| **Milestone** | Gelombang 5 — Nilai dan pihak luar |
| **Berkas sumber** | `ticketing/B-5-Input-Estimasi-dan-Penyelesaian-Nilai/issues/02-konversi-kurs-dan-perbandingan-ambang.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Nilai klaim valuta asing dikonversi ke Rupiah memakai **kurs pada tanggal kejadian**, dan hasil
konversi itulah yang dibandingkan dengan ambang komite.

Nilai bisnisnya adalah menutup cacat yang **membuat klaim besar lolos tanpa komite**. Fungsi kurs
lama mengembalikan **`1`** ketika kurs tidak ditemukan: satu satuan valuta asing dihitung setara
satu Rupiah, sehingga klaim bernilai besar menyusut menjadi kecil dan **lolos di bawah seluruh
ambang**. Tanpa galat, tanpa catatan, dan angkanya tampak wajar.

#### Ruang lingkup

- Konversi nilai settlement ke Rupiah memakai **kurs pada tanggal kejadian** (`ADR-0015`).
- **Menolak** klaim bila kurs untuk mata uang dan tanggal itu tidak ditemukan — dengan galat yang
  menyebutkan **mata uang dan tanggalnya**; tanpa nilai bawaan.
- Hasil konversi dipakai sebagai dasar perbandingan ambang komite (`B-7`) dan ambang Notice of
  Large Losses (`B-2`).
- Perbandingan memakai **nilai presisi penuh**, bukan nilai yang sudah dibulatkan tampilan.

#### Non-goal

- **Tidak** mengelola master kurs — itu `TKT-F4-004`.
- **Tidak** menentukan jenjang komite — itu `B-7`; modul ini menyediakan angkanya.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Isi `m_currencystandard`** | **DBA** (`R-19`) | Tanpa isinya, konversi tidak dapat diuji sama sekali — dan inilah salah satu dari tiga penghalang yang membuat `B-5` masih terikat `BRD §21.4` |
| **Bagaimana klaim yang tertolak karena kurs kosong diperlakukan** — ditahan dan diproses ulang otomatis, atau dikembalikan ke petugas? | **Work Owner** | Menentukan perilaku yang dilihat petugas setiap hari |

#### Acceptance criteria

- ☐ Konversi memakai kurs **tanggal kejadian**, bukan tanggal proses — diuji dengan klaim yang
      tanggal kejadiannya berbeda dari hari pengujian.
- ☐ Nilai Rupiah sebuah klaim **tidak berubah** saat klaim yang sama diproses ulang di hari lain
      — diuji. Inilah bukti butir 4 `P-5` tertutup.
- ☐ Kurs tidak ditemukan → **klaim ditolak** dengan galat yang menyebut mata uang dan tanggal;
      **nilai `1` tidak muncul di jalur mana pun** — diuji pemindaian dan uji fungsional. Inilah
      bukti butir 5 `P-5` tertutup.
- ☐ Perbandingan terhadap ambang memakai nilai presisi penuh — diuji dengan nilai yang berbeda
      hasilnya bila dibulatkan lebih dulu.
- ☐ Gerbang 1: **selisih muncul pada seluruh data historis valuta asing** — itu **diharapkan**,
      dan wajib terpetakan ke butir 4 dan 5 `P-5` (`D-54`). Selisih pada klaim Rupiah murni
      **wajib nol**.
- ☐ Gerbang 2: UAT **PncPICTeknik** dan **PNCKomite** pada klaim valuta asing.

#### Dependency / Blocked by

`TKT-B05-001` · `TKT-F4-004` (master kurs) · `TKT-F5-001` (tanggal). **Terhalang DBA dan satu
keputusan.**

#### Constraint keamanan, data, operasional

- **Dampak operasional yang harus disiapkan sebelum rilis:** klaim valuta asing yang dulu lolos
  kini **dapat ditolak** sampai kursnya dilengkapi. Ini perbaikan yang diinginkan, tetapi ia
  menghentikan pekerjaan petugas bila master kurs tidak terisi tepat waktu.
- Hasil konversi menentukan **berapa banyak orang yang harus menyetujui** klaim (`ADR-0014`) —
  kesalahannya berpindah langsung menjadi kesalahan kewenangan.

#### Migrasi skema / rollout / rollback

Tidak menambah tabel; memakai master kurs `F-4`.

**Rollback:** kembali ke `GETCURRENCYSTANDARD` **tidak tersedia** — mengembalikannya berarti
memulihkan cacat `RETURN 1`.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/settlement/... -run TestKonversiKursTanggalKejadian
go test ./internal/domain/settlement/... -run TestKursTidakDitemukanMenolak
grep -rInE "return\s+1(\.0)?\s*,\s*nil" internal/domain/settlement/    # HARUS 0 baris
go run ./cmd/s8 banding --modul B-5 --aturan kurs
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Kurs hari eksekusi dan `RETURN 1` | `Database/GETCURRENCYSTANDARD.fnc:3`, `:14`, `:20-22` |
| Butir 4 dan 5 dari 13 perbaikan `P-5` | `D-49` · `ADR-0017` |
| Kurs tanggal kejadian; tolak bila tidak ada | `D-48` · `ADR-0015` |
| Isi `m_currencystandard` belum ada | `R-19` · `D-55` |
| Perbandingan ambang memakai presisi penuh | `D-51` · `ADR-0016` |

#### Comments

### TKT-B05-003 — Fee adjuster, risiko sendiri, dan salvage pada nilai

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** — isi `GCNM_FEE_SCALE` (17 pita) belum ada |
| **Modul** | **B-5 Input Estimasi** · Gelombang: 5 · Bergantung pada: TKT-B05-001, TKT-B12-001 |
| **Requirement** | FR-B5 |
| **Keputusan** | D-49 butir 8, D-51 |
| **ADR** | 0016, 0017 |
| **Risiko** | R-19 |
| **Rule Pega yang digantikan** | perhitungan fee adjuster dari `POOLDATA.GCNM_FEE_SCALE` (**17 pita**) · pengurangan risiko sendiri · penambahan salvage pada sisa TSI (`ValidasiSisaTSI:817` versus `:1489`) |
| **Peran penguji gerbang 2** | **PncPICTeknik** dan **PNCSurveyor** |
| **Label** | `modul::B-5` `tipe::aturan-bisnis` `status::needs-info` `prioritas::tinggi` `gelombang::5` |
| **Milestone** | Gelombang 5 — Nilai dan pihak luar |
| **Berkas sumber** | `ticketing/B-5-Input-Estimasi-dan-Penyelesaian-Nilai/issues/03-fee-adjuster-dan-risiko-sendiri.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Tiga komponen yang mengubah nilai bersih klaim dihitung konsisten: **fee adjuster**, **risiko
sendiri**, dan **salvage**.

Nilai bisnisnya: ketiganya bekerja di arah berbeda — fee menambah biaya, risiko sendiri mengurangi
yang dibayarkan, salvage memulihkan kapasitas. Salah satu saja keliru, dan angka akhir yang
dibayarkan ke tertanggung ikut salah.

#### Ruang lingkup

- Perhitungan **fee adjuster** dari pita nilai pada `GCNM_FEE_SCALE` (17 pita).
- Pengurangan **risiko sendiri** (own risk) dari nilai yang dibayarkan.
- Penambahan **salvage** pada perhitungan sisa TSI — dengan perbaikan butir 8 `P-5`: salvage
  **selalu** ditambahkan, bukan hanya bila baris terakhir kebetulan bertipe salvage.
- Seluruh perhitungan memakai **nilai presisi penuh**; pembulatan hanya saat tampil.

#### Non-goal

- **Tidak** mencatat data salvage — itu `B-12`; modul ini membacanya.
- **Tidak** menghitung TSI dan sisa TSI — itu `TKT-B03-003`; modul ini memakainya.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Isi `POOLDATA.GCNM_FEE_SCALE` (17 pita)** | **DBA** (`R-19`) | Tanpa isinya, fee adjuster tidak dapat dihitung maupun diuji. Salah satu dari tiga penghalang `BRD §21.4` |
| **Berapa desimal pembulatan fee?** | **Work Owner** | Fee dihitung dari pita; pembulatan yang berbeda menghasilkan angka berbeda pada tagihan adjuster |
| **Rp 1.650.000 dan 2% masih berlaku?** | **Work Owner** | Dua konstanta yang muncul di perhitungan; keduanya tidak dapat dikonfirmasi dari source |
| **Salvage memulihkan TSI — apa niat bisnisnya?** | **Work Owner** | Menentukan apakah perilaku itu memang dikehendaki |

#### Acceptance criteria

- ☐ Fee adjuster dihitung dari pita yang benar untuk 17 pita — diuji pada **batas bawah dan
      batas atas setiap pita**, ditambah satu nilai di tengah.
- ☐ Nilai tepat pada batas pita diperlakukan konsisten — diuji, dan aturannya (`>` atau `>=`)
      dinyatakan eksplisit.
- ☐ Risiko sendiri dikurangkan dari nilai dibayar, **bukan** dari nilai akseptasi — diuji.
- ☐ **Salvage selalu ikut dihitung** pada sisa TSI, apa pun urutan barisnya — diuji dengan
      salvage di posisi pertama, tengah, dan terakhir: **hasilnya sama**. Inilah bukti butir 8
      `P-5` tertutup.
- ☐ Seluruh perhitungan memakai presisi penuh; pembulatan **hanya** di lapisan tampilan — diuji
      dengan rantai perhitungan empat tahap.
- ☐ Gerbang 1: selisih terhadap Pega **hanya** pada kasus salvage yang terpetakan ke butir 8
      `P-5`; selisih pada fee **wajib nol** bila isi `GCNM_FEE_SCALE` sama.
- ☐ Gerbang 2: UAT **PncPICTeknik** dan **PNCSurveyor**.

#### Dependency / Blocked by

`TKT-B05-001` · `TKT-B03-003` · `TKT-B12-001`. **Terhalang DBA dan tiga keputusan Work Owner.**

#### Constraint keamanan, data, operasional

- Fee adjuster adalah **tagihan ke pihak luar** — kesalahannya terlihat oleh adjuster, bukan hanya
  internal.
- Konstanta Rp 1.650.000 dan 2% **tidak boleh di-hardcode** (`ADR-0025`); keduanya menjadi master
  di `F-4` setelah statusnya dikonfirmasi.
- Setiap perubahan komponen perhitungan **wajib tercatat di jejak audit**.

#### Migrasi skema / rollout / rollback

Memakai master fee (`F-4`) dan tabel salvage (`B-12`); tidak menambah tabel sendiri.

**Rollback:** mengembalikan perhitungan salvage berarti **memulihkan cacat butir 8**.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/settlement/... -run TestFeeAdjusterPerPita
go test ./internal/domain/settlement/... -run TestSalvageSelaluDihitung
go test ./internal/domain/settlement/... -run TestRisikoSendiri
go run ./cmd/s8 banding --modul B-5 --aturan fee
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `GCNM_FEE_SCALE` 17 pita | `D-55` · `docs/verifikasi-bukti-adr.md` §15 baris `B-5` |
| `NilaiSalvage` hanya ditambahkan bila baris terakhir bertipe salvage | `ValidasiSisaTSI:817` versus `:1489` · `D-49` butir 8 |
| Salvage memulihkan kapasitas pertanggungan | `CONTEXT.md` — **Sisa TSI** |
| Uang presisi penuh, pembulatan saat tampil | `D-51` · `ADR-0016` |
| Nilai bisnis tidak boleh di-hardcode | `D-15` · `ADR-0025` |

#### Comments

# 13. B-6 · Penugasan & Inbox Petugas

*Modul Bisnis Inti · folder `docs/ticketing/B-6-Penugasan-dan-Inbox-Petugas/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Worklist / Workbasket** Pega — router `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter`, `KomiteRouter`, `PNCAdminRouterRCV`; inbox `PNCInboxAdmin`, `UserTeknisInbox`, `UserInbox_Harness`, `InboxManagerAdmin_Harness` |
| **Kode modul** | `B-6` |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | 30 activity |
| **Bergantung pada** | `F-3` Login & Hak Akses |
| **Kesiapan** | **SEBAGIAN** — algoritma sudah terbaca, router perlu konfirmasi |

### Apa yang dikerjakan modul ini

Menentukan **siapa mengerjakan apa**. Setiap tahap klaim menghasilkan satu **Tugas**, dan tugas
itu berada di salah satu dari dua tempat:

| | Isi | Dipakai pada tahap |
|---|---|---|
| **Worklist** | tugas milik **satu orang tertentu** | Input Register · View Polis · Estimation · Input Estimasi · Choose Surveyor · Send To Analis · Send To PIC Teknik · RCLDokter · Analyst Doctor |
| **Workbasket** | antrean **bersama**, diambil siapa pun yang berwenang | RCL/PUCL · Investigator · Compliance |

### Algoritma pembagian beban — sudah terbaca

`R-04` semula menyatakan tiga router hilang sehingga aturan penugasan tidak diketahui. Algoritmanya
ternyata **terbaca dari tempat lain**:

```sql
ORDER BY counter_quota ASC     -- RDB List/BrowsePICRandomTeam-SQL.xml:39-40
```

Petugas dengan beban paling sedikit mendapat tugas berikutnya, lalu `AddTJobCounterPIC_SQL`
menaikkan pencacahnya. `R-04` karena itu **turun dari penghalang menjadi verifikasi**.

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter` — **tidak ada di export** · 3 `Data-Admin-WorkBasket` · 4 harness inbox | **Tim Pega** (`R-04`, `R-16`) |
| **Keputusan** | Konfirmasi ketiga router memang memakai jalur `counter_quota` | **Work Owner + Tim Pega** |
| **Keputusan** | `operator_ID != 'ELLENSUPRIYATI'` **di-hardcode di dalam SQL** — dibawa sebagai peran, atau dihapus? | **Work Owner** |
| **Catatan** | Seluruh export hanya memuat **4 workbasket**; penjenjangan komite justru menugaskan ke **operator bernama** (`T-7`) | — |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B06-001](issues/01-model-penugasan-worklist-dan-workbasket.md) | Model penugasan Worklist dan Workbasket | `ready-for-human` |
| [TKT-B06-002](issues/02-aturan-routing-dan-pembagian-beban.md) | Aturan routing dan pembagian beban | `needs-info` |
| [TKT-B06-003](issues/03-penguncian-tugas-antar-pengguna.md) | Penguncian tugas antar pengguna | `ready-for-human` |

## Daftar Tiket

### TKT-B06-001 — Model penugasan Worklist dan Workbasket

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | **B-6 Penugasan & Inbox** · Gelombang: 3 · Bergantung pada: TKT-F3-005, TKT-U2-001 |
| **Requirement** | FR-B6 |
| **Keputusan** | D-26 |
| **ADR** | 0019, 0023 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | tabel `DATAPEGA.PC_ASSIGN_WORKLIST` (**dibaca 18 rule**) dan `DATAPEGA.PC_ASSIGN_WORKBASKET` (6 rule) · harness `PNCInboxAdmin`, `UserTeknisInbox`, `UserInbox_Harness` |
| **Peran penguji gerbang 2** | **PncAdmin**, **PncPICTeknik**, **PncRCLPUCL** |
| **Label** | `modul::B-6` `tipe::migrasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-6-Penugasan-dan-Inbox-Petugas/issues/01-model-penugasan-worklist-dan-workbasket.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Tugas klaim muncul di tempat yang benar: **Worklist** bila sudah bertuan, **Workbasket** bila
masih antrean bersama.

Nilai bisnisnya: pembagian ini **bukan warisan kosong** — ia mencerminkan cara kerja. Tahap yang
butuh kesinambungan penanganan memakai Worklist; tahap yang dikerjakan sebuah tim memakai
Workbasket. `D-13` menetapkan alur kerja tetap sama agar pengguna tidak perlu dilatih ulang, dan
mengubah model penugasan adalah perubahan yang **paling terasa setiap hari**.

#### Ruang lingkup

- Tabel Tugas milik aplikasi, menggantikan tabel engine Pega (`ADR-0004`).
- Dua model: **Worklist** (bertuan) dan **Workbasket** (antrean bersama) — satu tugas selalu berada
  di salah satunya, tidak pernah di keduanya.
- Layar inbox per model, memakai komponen tabel baku (`TKT-U2-001`).
- Perpindahan tugas antar tahap saat klaim maju.

#### Non-goal

- **Tidak** menentukan **siapa** yang menerima tugas — itu `TKT-B06-002` (aturan routing).
- **Tidak** menangani penguncian — itu `TKT-B06-003`.
- **Tidak** menangani penugasan komite: penjenjangan komite menugaskan ke **operator bernama**,
  bukan ke workbasket (`T-7`), dan itu milik `B-7`.

#### Acceptance criteria

- ☐ Satu tugas berada **di Worklist atau di Workbasket**, tidak pernah keduanya — ditegakkan
      constraint, diuji.
- ☐ Sembilan tahap Worklist dan tiga tahap Workbasket sesuai daftar `D-26` — diuji per tahap.
- ☐ Inbox Worklist menampilkan **hanya tugas milik pengguna**; inbox Workbasket menampilkan
      antrean yang boleh diambil perannya — diuji dengan dua pengguna berbeda.
- ☐ Batas data cabang dan lini bisnis ditegakkan **di kueri** — jumlah total yang ditampilkan
      juga tidak memuat data di luar batas (`TKT-F3-005`).
- ☐ Klaim yang maju ke tahap berikutnya **memindahkan tugasnya**, dan tugas lama tertutup —
      diuji.
- ☐ Gerbang 1: isi inbox **sama dengan Pega** untuk peran dan pengguna yang sama pada data
      staging — diuji pada 5 peran.
- ☐ Gerbang 2: UAT **PncAdmin**, **PncPICTeknik**, **PncRCLPUCL**.

#### Dependency / Blocked by

`TKT-F3-005` (izin dan batas data) · `TKT-U2-001` (tabel baku) · `TKT-F2-004` (skema).

#### Constraint keamanan, data, operasional

- Tabel penugasan menggantikan tabel engine Pega. Selama masa paralel, **penulis tunggal per
  tabel** berlaku (`P-1`): Pega masih menulis tabelnya sendiri sampai `B-6` lulus gerbang 2.
- Inbox menampilkan data nasabah; penyaringan **tidak boleh** dilakukan setelah data terambil —
  itu membocorkan lewat jumlah baris.

#### Migrasi skema / rollout / rollback

Menambah tabel Tugas milik aplikasi. Backward-compatible; Pega tidak membacanya.

**Rollback:** tugas yang sudah dibuat di sistem baru **tetap ada**; pengguna kembali memakai inbox
Pega untuk tahap yang belum pindah. Tidak ada data yang hilang, tetapi **tugas dapat muncul di dua
tempat** selama masa transisi — itu konsekuensi yang harus disampaikan ke pengguna sebelum rilis.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/penugasan/... -run TestWorklistAtauWorkbasket
go test ./internal/app/penugasan/... -run TestInboxPerPengguna
go test ./internal/app/penugasan/... -run TestBatasDataCabang
go run ./cmd/s8 banding --modul B-6 --peran 5
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Dua model penugasan dan daftar tahapnya | `D-26` · `ADR-0019` · `Flow/Register_Flow.xml` |
| `PC_ASSIGN_WORKLIST` dibaca 18 rule; `PC_ASSIGN_WORKBASKET` 6 rule | `D-21` · `ADR-0004` |
| Istilah Worklist, Workbasket, Tugas | `CONTEXT.md` |
| Hanya 4 workbasket di seluruh export | `T-7` |

#### Comments

### TKT-B06-002 — Aturan routing dan pembagian beban

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** — tiga router tidak ada di export |
| **Modul** | **B-6 Penugasan & Inbox** · Gelombang: 3 · Bergantung pada: TKT-B06-001 |
| **Requirement** | FR-B6 |
| **Keputusan** | D-26, D-15 |
| **ADR** | 0019, 0025 |
| **Risiko** | R-04, R-16 |
| **Rule Pega yang digantikan** | 8 router — `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter`, `KomiteRouter`, `PNCAdminRouterRCV`, `ToCurrentOperator`, `ToWorkList`, `ToWorkbasket` · algoritma beban pada `RDB List/BrowsePICRandomTeam-SQL.xml:39-40` dan `AddTJobCounterPIC_SQL` |
| **Peran penguji gerbang 2** | **PncManagerAdmin** — peran yang paling merasakan bila pembagian beban timpang |
| **Label** | `modul::B-6` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-6-Penugasan-dan-Inbox-Petugas/issues/02-aturan-routing-dan-pembagian-beban.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Tugas baru diberikan kepada petugas yang **paling sedikit bebannya**, bukan acak dan bukan selalu
orang yang sama.

Nilai bisnisnya langsung terasa: pembagian beban yang timpang berarti sebagian petugas menumpuk
pekerjaan sementara yang lain menganggur — dan itu memanjangkan TAT tanpa alasan yang terlihat di
laporan mana pun.

#### Ruang lingkup

- Aturan routing sebagai **data**, bukan kode tersebar: untuk tiap tahap, siapa yang berhak
  menerima dan bagaimana dipilih.
- Algoritma pembagian beban: **`ORDER BY counter_quota ASC`** — yang paling sedikit bebannya
  menang; pencacah dinaikkan setelah penugasan.
- Delapan router lama dibangun ulang sebagai aturan routing terpadu.
- **Penghapusan hardcode nama orang** di dalam SQL routing.

#### Non-goal

- **Tidak** menangani penugasan komite (`B-7`).
- **Tidak** menangani penguncian (`TKT-B06-003`).

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **`PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter`** — tidak ada di export | **Tim Pega** (`R-04`, `R-16`) | Algoritma bebannya sudah terbaca, tetapi **syarat kelayakan** (siapa yang boleh masuk daftar calon) hanya ada di dalam router |
| **Konfirmasi ketiga router memang memakai jalur `counter_quota`** | **Work Owner + Tim Pega** | Rekonstruksi dari kueri lain **belum diverifikasi** terhadap router aslinya |
| **`operator_ID != 'ELLENSUPRIYATI'` di-hardcode di dalam SQL** — dibawa sebagai peran, atau dihapus? | **Work Owner** | Bila dibawa, ia menjadi baris master; bila dihapus, perilaku penugasan **berubah** dan itu harus dinyatakan sebagai perubahan sadar |
| **3 `Data-Admin-WorkBasket`** — definisi antrean bersama | **Tim Pega** | Menentukan antrean mana yang ada dan siapa yang boleh mengambilnya |

#### Acceptance criteria

- ☐ Tugas baru diberikan kepada petugas dengan **`counter_quota` terkecil** — diuji dengan tiga
      petugas berbeda beban: yang terkecil menang.
- ☐ Pencacah beban **naik tepat satu** setelah penugasan, dan **turun** saat tugas selesai —
      diuji; bila hanya naik, pembagian beban menjadi timpang permanen.
- ☐ Dua penugasan **bersamaan** tidak memberikan tugas ke petugas yang sama secara keliru —
      diuji dengan dua proses paralel.
- ☐ **Nol nama orang di kode maupun kueri routing** — diuji pemindaian terhadap 24 Operator ID
      yang diketahui (`ADR-0025`).
- ☐ Aturan routing dapat diubah lewat master **tanpa deployment** — diuji.
- ☐ Petugas yang sedang tidak aktif **tidak menerima tugas baru** — diuji.
- ☐ Gerbang 1: petugas yang terpilih **sama dengan Pega** pada 30 penugasan contoh di staging.
- ☐ Gerbang 2: UAT **PncManagerAdmin** — memeriksa sebaran beban setelah 50 penugasan.

#### Dependency / Blocked by

`TKT-B06-001` · `TKT-F4-001` (master aturan routing). **Terhalang Tim Pega dan Work Owner.**

#### Constraint keamanan, data, operasional

- Pencacah beban adalah **state yang harus dijaga konsisten**. Bila naik tanpa pernah turun, atau
  gagal naik karena galat, pembagian beban menjadi timpang **secara permanen** dan tidak ada yang
  menyadarinya sampai ada yang mengeluh.
- Hardcode nama orang di dalam SQL adalah bentuk terburuk dari `D-15`: ia **tidak terlihat** saat
  membaca kode aplikasi, hanya saat membaca teks kueri.
- Penugasan tercatat di jejak audit — ia menentukan siapa yang bertanggung jawab atas sebuah klaim.

#### Migrasi skema / rollout / rollback

Menambah master aturan routing dan kolom pencacah beban.

**Rollback:** kembali ke router Pega **tidak tersedia** untuk klaim yang sudah dipegang sistem
baru. Rollback berarti menghentikan penugasan baru di sistem baru.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/penugasan/... -run TestPembagianBebanTerkecil
go test ./internal/domain/penugasan/... -run TestPencacahNaikTurun
go test ./internal/app/penugasan/... -run TestPenugasanBersamaan
go run ./cmd/tools/cek-hardcode-operator internal/domain/penugasan/   # HARUS 0
go run ./cmd/s8 banding --modul B-6 --aturan routing --kasus 30
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `ORDER BY counter_quota ASC` | `RDB List/BrowsePICRandomTeam-SQL.xml:39-40` |
| Pencacah dinaikkan `AddTJobCounterPIC_SQL` | `ADR-0019` |
| Delapan router yang dibangun ulang | `D-26` · `ADR-0019` |
| Tiga router tidak ada di export | `R-04` |
| `operator_ID != 'ELLENSUPRIYATI'` hardcode di SQL | `docs/verifikasi-bukti-adr.md` §15 baris `B-6` |

#### Comments

### TKT-B06-003 — Penguncian tugas antar pengguna

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | **B-6 Penugasan & Inbox** · Gelombang: 3 · Bergantung pada: TKT-B06-001 |
| **Requirement** | FR-B6 |
| **Keputusan** | D-26, D-27 |
| **ADR** | 0019, 0026 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | penguncian assignment bawaan Pega — **tidak ada rule aplikasi** yang mengaturnya; di sistem baru ia harus dibangun sendiri |
| **Peran penguji gerbang 2** | **PncRCLPUCL** dan **PncComplience** — dua peran yang bekerja dari Workbasket bersama |
| **Label** | `modul::B-6` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-6-Penugasan-dan-Inbox-Petugas/issues/03-penguncian-tugas-antar-pengguna.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Dua petugas **tidak dapat mengerjakan tugas yang sama** dari satu Workbasket bersama tanpa
menyadarinya.

Nilai bisnisnya: Workbasket adalah antrean bersama — RCL/PUCL, Investigator, dan Compliance
mengambil dari sana. Tanpa penguncian, dua orang dapat membuka klaim yang sama, dan yang menyimpan
belakangan **menimpa pekerjaan yang pertama** tanpa peringatan.

#### Ruang lingkup

- Mekanisme **ambil tugas** (claim) dari Workbasket: tugas menjadi milik pengambil sampai selesai
  atau dilepas.
- Penanganan tugas yang **ditinggalkan** — pengguna menutup peramban tanpa melepas.
- Perilaku bila tugas sudah diambil orang lain: pesan yang **menyebut siapa** yang memegangnya.
- Pelepasan tugas oleh pemegang, dan oleh peran manajer bila pemegangnya tidak hadir.

#### Non-goal

- **Tidak** mengunci baris database untuk waktu lama — penguncian ini urusan alur kerja, bukan
  transaksi.
- **Tidak** menangani ketidakhadiran terjadwal — kolom `STS_ABS` pada master komite adalah
  persoalan `B-7`.

#### Acceptance criteria

- ☐ Tugas yang diambil pengguna A **tidak muncul sebagai tersedia** bagi pengguna B — diuji.
- ☐ Pengguna B yang membuka tugas itu lewat URL langsung menerima pesan yang **menyebut nama
      pemegangnya** — diuji.
- ☐ Tugas yang ditinggalkan tanpa dilepas **kembali tersedia** setelah batas waktu yang
      dikonfigurasi — diuji dengan batas pendek.
- ☐ Peran manajer dapat **melepas paksa** tugas milik orang lain, dan tindakan itu **tercatat di
      jejak audit** dengan pelaku dan alasan — diuji.
- ☐ Penguncian bekerja **lintas dua instans aplikasi** — diuji dengan mengambil tugas di instans
      A dan mencoba mengambilnya di instans B: **ditolak**. Ini konsekuensi langsung `D-27`.
- ☐ Gerbang 1: **tidak ada padanan langsung di Pega** untuk dibandingkan — kelulusan bertumpu
      pada uji fungsional di atas, bukan perbandingan hasil.
- ☐ Gerbang 2: UAT **PncRCLPUCL** dan **PncComplience** dengan dua pengguna bersamaan.

#### Dependency / Blocked by

`TKT-B06-001` · `TKT-S5-002` (pencatatan pelepasan paksa).

#### Constraint keamanan, data, operasional

- Penguncian **tidak boleh** disimpan di memori satu instans — aplikasi wajib stateless (`D-27`),
  dan load balancer mengarahkan permintaan ke instans mana pun.
- Melepas paksa tugas orang lain adalah tindakan bernilai tinggi; karena `D-59` tidak mengenal
  pemisahan tugas, **jejak auditlah satu-satunya kontrolnya**.
- Batas waktu tugas ditinggalkan harus cukup panjang untuk pekerjaan nyata — batas yang terlalu
  pendek membuat tugas direbut saat pengguna sedang mengetik.

#### Migrasi skema / rollout / rollback

Menambah kolom pemegang dan waktu pengambilan pada tabel Tugas — kolom *nullable*,
backward-compatible.

**Rollback:** kolom dibiarkan dan diabaikan; tugas kembali dapat diambil siapa pun — dengan risiko
tumpang tindih yang justru dihilangkan tiket ini.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/penugasan/... -run TestAmbilTugas
go test ./internal/app/penugasan/... -run TestTugasDitinggalkanKembaliTersedia
go test ./internal/app/penugasan/... -run TestPenguncianLintasInstans
go test ./internal/app/penugasan/... -run TestLepasPaksaTercatat
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Workbasket adalah antrean bersama | `D-26` · `CONTEXT.md` — **Workbasket** |
| Tiga tahap memakai Workbasket | `D-26` · `Flow/Register_Flow.xml` |
| Stateless, dua instans | `D-27` |
| Tidak ada pemisahan tugas; audit satu-satunya kontrol | `D-59` · `ADR-0023` |

#### Comments

# 14. B-7 · Komite Persetujuan Klaim

*Modul Bisnis Inti · folder `docs/ticketing/B-7-Komite-Persetujuan-Klaim/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Komite** — `Flow/Komite_Flow.xml`, inbox `InboxKomite_Harness`, activity `SetEmailKomite`, `SetListComiteeClaimPerObjAdj`, `GetKomiteApproval`, `AutoAcceptKomite` |
| **Kode modul** | `B-7` |
| **Gelombang** | 4 — Persetujuan |
| **Ukuran** | 34 activity |
| **Bergantung pada** | `B-5` Input Estimasi · `B-6` Penugasan · `F-4` Master Data |
| **Kesiapan** | **TERHALANG** |

### Apa yang dikerjakan modul ini

Meminta persetujuan berjenjang atas **nilai klaim** sebelum klaim dapat diakseptasi. Komite dapat
menyetujui, menolak, atau mengembalikan.

### Cara jumlah penyetuju ditentukan — kumulatif

Ini bagian yang paling mudah disalahpahami, dan sudah dua kali nyaris salah dibaca selama analisis.

**Bukan** memilih satu jenjang dari matriks. Melainkan: **setiap jenjang yang ambang bawahnya sudah
terlampaui nilai klaim ikut menyetujui.**

```
Jumlah jenjang = jumlah baris master yang LIMIT_BOTTOM <= nilai klaim
```
`Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459` · `When/IsKomiteLoop-When.xml`

Bukti pendukung: `LIMIT_BOTTOM` difilter di **11 SQL rule**; `LIMIT_TOP` di **0**.

Untuk lini **Non-MBU saja**, akumulasi didahului pemilihan **pita nilai** (≤ Rp 100 juta → pita
`1`; di atasnya → pita `2`). Lini lain tidak punya langkah itu (`D-70`).

Contoh nyata dari master yang sudah diterima:

| Kasus | Jumlah penyetuju |
|---|---|
| PA Rp 5.000.000 | 1 |
| PA Rp 75.000.000 | 3 |
| PA Rp 150.000.000 | 4 |
| Travel Rp 150.000.000 | 3 |
| Non-MBU Rp 80.000.000 | 2 |
| Non-MBU Rp 750.000.000 | 2 |
| Non-MBU Rp 2.000.000.000 | 3 |

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | Ticket rule `KomiteAssign_ticket` dan `komiteAccept_ticket` · When rule `IsKomite` | **Tim Pega** (`R-16`) |
| **Keputusan** | **`AutoAcceptKomite` menyetujui komite otomatis tiap hari jam 06:00** — perilaku yang benar? | **Work Owner** |
| **Keputusan** | Baris `DEGREE=0` maksudnya apa? | **Work Owner** |
| **Keputusan** | **PA dan Travel di atas Rp 200.000.000 tidak punya baris master** | **Work Owner** |
| **Keputusan** | `KOMITEKE` immutable? · `dbms_random.value` pada 2 kueri Simasnet disengaja? | **Work Owner** |
| **Terhalang tiket** | `TKT-F4-002` — struktur master ambang komite | — |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B07-001](issues/01-penentuan-jenjang-komite-kumulatif.md) | Penentuan jenjang komite kumulatif | `needs-info` |
| [TKT-B07-002](issues/02-layar-komite-dan-pencatatan-keputusan.md) | Layar Komite dan pencatatan keputusan | `needs-info` |
| [TKT-B07-003](issues/03-persetujuan-otomatis-autoacceptkomite.md) | Persetujuan otomatis `AutoAcceptKomite` | `needs-info` |

## Daftar Tiket

### TKT-B07-001 — Penentuan jenjang komite kumulatif

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | terhalang keputusan dan tiket `TKT-F4-002` |
| **Modul** | **B-7 Komite** · Gelombang: 4 · Bergantung pada: TKT-B05-002, TKT-F4-002 |
| **Requirement** | FR-B7 |
| **Keputusan** | D-14, D-47, D-52, D-70 |
| **ADR** | 0014 |
| **Risiko** | R-19 |
| **Rule Pega yang digantikan** | `Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459` (`KomiteLoop := pxResultCount`) · `When/IsKomiteLoop-When.xml` · **17 kueri** pembaca `POOLDATA.EMAILKOMITE` · `Activity/SetEmailKomite-Act.xml` beserta tiga varian |
| **Peran penguji gerbang 2** | **PNCKomite** dan **PNCKomiteTeknik** |
| **Label** | `modul::B-7` `tipe::aturan-bisnis` `status::needs-info` `prioritas::tinggi` `gelombang::4` |
| **Milestone** | Gelombang 4 — Persetujuan |
| **Berkas sumber** | `ticketing/B-7-Komite-Persetujuan-Klaim/issues/01-penentuan-jenjang-komite-kumulatif.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Untuk sebuah nilai klaim, sistem menetapkan **berapa banyak jenjang yang harus menyetujui** dan
**siapa saja penyetujunya** — dengan hasil yang sama setiap kali dihitung.

Nilai bisnisnya: ini menentukan **siapa berwenang menyetujui uang**. Salah hitung ke bawah berarti
klaim besar disetujui terlalu sedikit orang; salah ke atas berarti klaim kecil tertahan tanpa
alasan.

#### Ruang lingkup

- Perhitungan **kumulatif**: seluruh jenjang yang **ambang bawahnya sudah terlampaui** ikut
  menyetujui.
- Untuk lini **Non-MBU**: pemilihan **pita nilai** lebih dulu (≤ Rp 100 juta → pita `1`; di atasnya
  → pita `2`), lalu akumulasi **di dalam pita itu saja**.
- Untuk lini lain: **tanpa langkah pita** — akumulasi langsung atas seluruh jenjang lini tersebut.
- Nilai yang dibandingkan adalah hasil konversi kurs `TKT-B05-002`, memakai **presisi penuh**.
- Urutan penyetuju mengikuti `DEGREE`.

#### Non-goal

- **Tidak** memakai `LIMIT_TOP` untuk memilih baris. `LIMIT_TOP` adalah **validasi integritas
  master** (`TKT-F4-002`), bukan penyaring.
- **Tidak** membangun layar komite — itu `TKT-B07-002`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **PA dan Travel di atas Rp 200.000.000 tidak punya baris master** | **Work Owner** | Klaim di atas nilai itu **tidak punya penyetuju sama sekali** — dan itu tidak boleh ditebak |
| **Baris `DEGREE=0` maksudnya apa?** | **Work Owner** | Muncul di master tetapi tidak jelas apakah ia jenjang, penanda, atau sisa data |
| **`dbms_random.value` pada 2 kueri Simasnet disengaja?** | **Work Owner** | Bila ya, **acceptance criteria untuk jalur Simasnet tidak boleh deterministik** — dan itu mengubah cara modul ini diuji |
| **Belum ditelusuri:** lini apa saja yang melewati `EmailKomiteBerjenjang_sql`, `EmailKomiteAdjuster_sql`, `EmailKomiteSalvage_sql` — ketiganya menerima `TYPE_KOMITE` dari pemanggil lewat `tempAdj.pyMemo` dan `tempAdj.AcceptedNo` | **Lead Engineer** | Menentukan lini mana yang memakai langkah pita. Ini **Definition of Ready** tiket ini |

#### Acceptance criteria

- ☐ Jumlah penyetuju untuk **ketujuh kasus** pada `spec.md` sesuai tabel — diuji satu per satu.
- ☐ Untuk **Non-MBU**, pita dipilih lebih dulu dan akumulasi **tidak menyeberang antarpita** —
      diuji dengan Rp 80 juta (pita 1, 2 penyetuju) dan Rp 200 juta (pita 2, 1 penyetuju).
- ☐ Untuk **PA dan Travel**, **tidak ada langkah pita** — diuji: PA Rp 150 juta menghasilkan
      **4 penyetuju**, bukan 2.
- ☐ `LIMIT_TOP` **tidak dipakai menyaring baris** — diuji pemindaian kueri.
- ☐ Perbandingan memakai **nilai presisi penuh** hasil konversi kurs — diuji.
- ☐ Urutan penyetuju mengikuti `DEGREE` — diuji.
- ☐ Gerbang 1: jumlah dan identitas penyetuju **sama dengan Pega** pada 30 klaim contoh dari
      empat lini. Untuk jalur Simasnet, kriteria menyesuaikan jawaban `dbms_random`.
- ☐ Gerbang 2: UAT **PNCKomite** dan **PNCKomiteTeknik**.

#### Dependency / Blocked by

`TKT-B05-002` (nilai terkonversi) · `TKT-F4-002` (master ambang). **Terhalang tiga keputusan dan
satu penelusuran.**

#### Constraint keamanan, data, operasional

- **Menerapkan filter pita secara seragam ke semua lini akan merusak.** Dihitung dari master: PA
  Rp 5 juta menjadi **0 penyetuju** dan Travel Rp 150 juta menjadi **0 penyetuju** — klaim mandek.
  Itu sebabnya `D-70` membatasi pita **hanya** ke Non-MBU.
- **Menerapkan rentang tertutup** (`LIMIT_BOTTOM <= nilai <= LIMIT_TOP`) akan mengembalikan tepat
  satu baris → **satu jenjang berapa pun nilai klaim** → menghapus penjenjangan (`D-47`).
- Kolom `TYPE_KOMITE` **memikul dua arti**: pita nilai di Non-MBU, varian jalur di PA. Model data
  wajib mendokumentasikannya.

#### Migrasi skema / rollout / rollback

Tidak menambah tabel; membaca master `TKT-F4-002`.

**Rollback:** kembali membaca `POOLDATA.EMAILKOMITE` langsung.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/komite/... -run TestJenjangKumulatif   # 7 kasus pada spec
go test ./internal/domain/komite/... -run TestPitaHanyaNonMBU
go test ./internal/domain/komite/... -run TestLimitTopTidakMenyaring
go run ./cmd/s8 banding --modul B-7 --aturan jenjang --kasus 30
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `KomiteLoop := pxResultCount` | `Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459` |
| `KomiteCount <= KomiteLoop` | `When/IsKomiteLoop-When.xml` |
| `LIMIT_BOTTOM` di 11 rule, `LIMIT_TOP` di 0 | `ADR-0014` |
| Pita hanya Non-MBU | `D-70` |
| Tujuh kasus jumlah penyetuju | `ADR-0014` Rationale |
| Tiga kueri dinamis belum ditelusuri | `ADR-0014` Pertanyaan terbuka |

#### Comments

### TKT-B07-002 — Layar Komite dan pencatatan keputusan

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** — Ticket rule dan When rule komite |
| **Modul** | **B-7 Komite** · Gelombang: 4 · Bergantung pada: TKT-B07-001, TKT-B06-001 |
| **Requirement** | FR-B7 |
| **Keputusan** | D-26, D-59 |
| **ADR** | 0019, 0021, 0023, 0026 |
| **Risiko** | R-16 |
| **Rule Pega yang digantikan** | `Flow/Komite_Flow.xml` · harness `InboxKomite_Harness` · Ticket rule `KomiteAssign_ticket` (`:974`) dan `komiteAccept_ticket` (`:796`) — **keduanya hilang dari export** · When rule `IsKomite` — **hilang** |
| **Peran penguji gerbang 2** | **PNCKomite** dan **PNCKomiteTeknik** |
| **Label** | `modul::B-7` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::4` |
| **Milestone** | Gelombang 4 — Persetujuan |
| **Berkas sumber** | `ticketing/B-7-Komite-Persetujuan-Klaim/issues/02-layar-komite-dan-pencatatan-keputusan.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Anggota komite melihat klaim yang menunggu persetujuannya, memberi keputusan, dan keputusan itu
**tercatat permanen** beserta siapa dan kapan.

Nilai bisnisnya: persetujuan komite adalah **titik kewenangan tertinggi atas uang klaim**. Dan
karena `D-59` menghapus pemisahan tugas, pencatatan keputusan ini adalah **satu-satunya bukti**
bahwa persetujuan benar-benar diberikan orang yang berwenang.

#### Ruang lingkup

- Inbox komite per anggota, menampilkan klaim yang menunggu jenjangnya.
- Tiga keputusan: **setuju**, **tolak**, **kembalikan** — masing-masing dengan catatan.
- Pencatatan jejak komite: siapa, kapan, keputusan apa, catatan apa, pada jenjang ke berapa.
- Perpindahan ke jenjang berikutnya setelah satu jenjang menyetujui; klaim **tidak dapat
  diakseptasi** sebelum seluruh jenjang selesai (invarian `I-5`).
- **Lompatan lateral** saat seluruh komite menyetujui — kembali ke petugas estimasi untuk lini PA.

#### Non-goal

- **Tidak** menghitung jumlah jenjang — itu `TKT-B07-001`.
- **Tidak** menangani persetujuan otomatis — itu `TKT-B07-003`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Ticket rule `KomiteAssign_ticket` dan `komiteAccept_ticket`** — dirujuk `Flow/Komite_Flow.xml:974` dan `:796`, **tidak ada di export** | **Tim Pega** (`R-16`) | Menentukan **kapan** klaim melompat masuk dan keluar dari komite |
| **When rule `IsKomite`** — hilang | **Tim Pega** | Menentukan syarat sebuah klaim masuk komite sama sekali |
| **`KOMITEKE` immutable?** | **Work Owner** | Menentukan apakah jenjang yang sudah dilewati dapat diulang |
| **Siapa yang berwenang memicu lompatan lateral?** Sistem lama **nol pagar izin** | **Work Owner** | `D-59` bersatuan menu; lompatan melewati alur normal |

#### Acceptance criteria

- ☐ Anggota komite hanya melihat klaim yang **menunggu jenjangnya** — diuji dengan tiga anggota
      pada jenjang berbeda.
- ☐ Ketiga keputusan tersedia dan masing-masing menghasilkan **tepat satu baris jejak komite**
      berisi pelaku, waktu, keputusan, catatan, dan nomor jenjang.
- ☐ Klaim **tidak dapat diakseptasi** sebelum seluruh jenjang menyetujui — diuji dengan mencoba
      akseptasi di tengah jalan: **ditolak** (invarian `I-5`).
- ☐ Keputusan **tolak** dan **kembalikan** memindahkan klaim ke tahap yang benar — diuji.
- ☐ Setiap keputusan tercatat di **jejak audit** (`S-5`), terpisah dari jejak komite —
      keduanya diperiksa.
- ☐ Anggota tanpa izin menu komite menerima `403` dan **tidak melihat menunya**.
- ☐ Gerbang 1: urutan dan hasil keputusan **sama dengan Pega** pada 20 klaim contoh.
- ☐ Gerbang 2: UAT **PNCKomite** dan **PNCKomiteTeknik**.

#### Dependency / Blocked by

`TKT-B07-001` · `TKT-B06-001` · `TKT-S5-002`. **Terhalang Tim Pega dan dua keputusan.**

#### Constraint keamanan, data, operasional

- **Orang yang sama dapat mengubah nilai klaim dan menyetujuinya** bila perannya memiliki kedua
  menu (`D-59`). Tidak ada kontrol teknis yang mencegahnya; jejak audit satu-satunya pengimbang.
- Penjenjangan komite menugaskan ke **operator bernama**, bukan ke workbasket (`T-7`) — sehingga
  **ketidakhadiran seseorang dapat menghentikan klaim**. Kolom `STS_ABS` pada master tampaknya
  menjawab ini, dan perilakunya belum dirumuskan.
- Keputusan komite **tidak dapat dihapus** (`ADR-0012`) — hanya ditandai, tidak pernah dibuang.

#### Migrasi skema / rollout / rollback

Menambah tabel jejak komite. Backward-compatible.

**Rollback:** keputusan yang sudah tercatat tetap ada dan tetap sah.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/komite/... -run TestInboxPerJenjang
go test ./internal/app/komite/... -run TestAkseptasiDitolakSebelumSeluruhJenjang
go test ./internal/app/komite/... -run TestKeputusanTercatat
go run ./cmd/s8 banding --modul B-7 --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Ticket rule komite hilang | `Flow/Komite_Flow.xml:796`, `:974` · `ADR-0021` |
| Invarian `I-5` akseptasi setelah seluruh jenjang | `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Komite menugaskan ke operator bernama; hanya 4 workbasket | `T-7` |
| Tidak ada pemisahan tugas | `D-59` · `ADR-0023` |
| Nol pagar izin pada transisi lateral | `docs/verifikasi-bukti-adr.md` §15 |

#### Comments

### TKT-B07-003 — Persetujuan otomatis `AutoAcceptKomite`

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan** — perilaku ini belum pernah dibahas di dokumen mana pun |
| **Modul** | **B-7 Komite** · Gelombang: 4 · Bergantung pada: TKT-B07-002, TKT-S6-001 |
| **Requirement** | FR-B7, FR-S6 |
| **Keputusan** | D-57, D-59 |
| **ADR** | 0022, 0023, 0026 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | `Job Scheduler/JOBForKomiteKlaimPNC` — **harian, jam 06:00**, menjalankan activity **`AutoAcceptKomite`** |
| **Peran penguji gerbang 2** | **PNCKomite** dan **PncManagerAdmin** |
| **Label** | `modul::B-7` `tipe::aturan-bisnis` `status::needs-info` `prioritas::tinggi` `gelombang::4` |
| **Milestone** | Gelombang 4 — Persetujuan |
| **Berkas sumber** | `ticketing/B-7-Komite-Persetujuan-Klaim/issues/03-persetujuan-otomatis-autoacceptkomite.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Kejelasan tentang satu proses yang berjalan setiap hari dan **menyetujui komite tanpa pengguna
sama sekali**.

Nilai bisnisnya bukan fitur baru, melainkan **kesadaran**. Perilaku ini ditemukan saat folder
`Job Scheduler/` diterima, dan ia **tidak tercatat di dokumen mana pun** sebelumnya. Sebuah proses
yang menyetujui uang klaim secara otomatis pada jam 06:00 pantas diputuskan secara sadar, bukan
diwarisi begitu saja.

#### Ruang lingkup

- Penerapan kembali `AutoAcceptKomite` **bila Work Owner memutuskan perilaku ini dipertahankan**.
- Bila dipertahankan: pencatatan **jejak audit dengan pelaku "sistem"**, bukan kosong — agar
  persetujuan otomatis tetap dapat dipertanggungjawabkan.
- Bila tidak dipertahankan: klaim yang sebelumnya disetujui otomatis akan **menumpuk di inbox
  komite**, dan dampak operasionalnya harus disiapkan.
- Batasan: jenjang mana yang boleh disetujui otomatis, dan pada nilai berapa.

#### Non-goal

- **Tidak** memutuskan sendiri apakah perilaku ini benar. Itu keputusan Work Owner.
- **Tidak** membangun mekanisme penjadwal — itu `S-6`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **`AutoAcceptKomite` harian jam 06:00 — perilaku yang benar?** | **Work Owner** | Ia **menyetujui komite tanpa pengguna**, sehingga **melewati kontrol menu `D-59` sepenuhnya**. Bila dipertahankan, ia adalah satu-satunya jalur yang tidak tunduk model izin |
| Jenjang dan nilai mana yang boleh disetujui otomatis | **Work Owner** | Tanpa batas, persetujuan otomatis berlaku untuk klaim nilai berapa pun |
| **Bila dua instans aplikasi berjalan, siapa yang menjalankan job ini?** | **Lead Engineer + Infra** (`ADR-0022`) | Penguncian yang salah membuat job berjalan **dua kali** — dan dua kali persetujuan komite pada klaim yang sama adalah kesalahan bernilai uang |

#### Acceptance criteria

> Belum dapat diangkakan sampai perilaku diputuskan. Bentuk AC yang akan diisi:

- ☐ Bila dipertahankan: klaim yang memenuhi syarat disetujui otomatis pada jadwal yang
      ditetapkan, dan **setiap persetujuan menghasilkan baris jejak audit dengan pelaku "sistem"**.
- ☐ Job **tidak pernah berjalan dua kali** untuk hari yang sama, walau dua instans aplikasi
      hidup — diuji dengan dua instans bersamaan (`ADR-0022`).
- ☐ Klaim di luar batas jenjang dan nilai yang ditetapkan **tidak** disetujui otomatis — diuji.
- ☐ Bila **tidak** dipertahankan: klaim yang dulu disetujui otomatis **muncul di inbox komite**,
      dan jumlahnya dilaporkan sebagai angka sebelum rilis — agar beban barunya diketahui.
- ☐ Kegagalan job **terlihat** — tercatat dan dapat diketahui, bukan gagal diam-diam.

#### Dependency / Blocked by

`TKT-B07-002` · `TKT-S6-001` (mekanisme penjadwal). **Terhalang tiga keputusan.**

#### Constraint keamanan, data, operasional

- **Job ini melewati seluruh kontrol otorisasi.** `D-59` menetapkan izin bersatuan menu; job tidak
  punya pengguna, sehingga tidak ada menu yang dapat diperiksa. Ini **satu-satunya jalur** dengan
  sifat demikian, dan itulah alasan ia pantas diputuskan secara eksplisit.
- **Jejak audit adalah satu-satunya kontrol** yang tersisa untuk jalur ini — dan pelakunya harus
  tercatat sebagai "sistem", bukan dibiarkan kosong.
- `pyBypassActivityAuthentication=true` pada agent terkait juga menunggu keputusan (`ADR-0022`).

#### Migrasi skema / rollout / rollback

Tidak menambah tabel.

**Rollback:** mematikan job. Klaim yang telanjur disetujui otomatis **tetap disetujui** — itu tidak
dapat ditarik, dan karena itu keputusan mengaktifkannya harus diambil sebelum rilis, bukan
sesudahnya.

#### Rencana verifikasi

> **Rencana — belum dijalankan, dan belum dapat disusun lengkap.**

```bash
go test ./internal/app/komite/... -run TestAutoAcceptBatasJenjang
go test ./internal/app/jadwal/... -run TestJobTidakBerjalanDuaKali
go run ./cmd/tools/hitung-klaim-auto-accept --periode 90d   # beban bila fitur dihapus
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `JOBForKomiteKlaimPNC` harian 06:00 menjalankan `AutoAcceptKomite` | `D-57` · `Job Scheduler/` |
| Perilaku ini tidak tercatat di dokumen mana pun sebelumnya | `D-57` |
| Job melewati kontrol menu | `D-59` · `ADR-0023` |
| Mekanisme penjadwal dua instans belum diputuskan | `ADR-0022` `Proposed` |

#### Comments

# 15. B-8 · Choose Surveyor & Hasil Survei

*Modul Bisnis Inti · folder `docs/ticketing/B-8-Choose-Surveyor-dan-Hasil-Survei/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Choose Surveyor** / **InputSurveyor** — tahap pada `Flow/Register_Flow.xml`; inbox `InboxSurvey_Harness`, `SurveyorsInbox`, `DetailSurveyorsInbox`, `ViewDetailHasilSurveyorInternal1`; master `MasterLoginSurvey` |
| **Kode modul** | `B-8` |
| **Gelombang** | 4 — Persetujuan |
| **Ukuran** | 30 activity |
| **Bergantung pada** | `B-6` Penugasan |
| **Kesiapan** | **SEBAGIAN** |

### Apa yang dikerjakan modul ini

Menugaskan **surveyor atau loss adjuster** untuk meninjau kerugian di lapangan, lalu mencatat hasil
survei beserta foto dokumentasinya. Surveyor dapat internal (ASM) maupun eksternal.

Modul ini juga menyimpan komponen **KPI adjuster** — dan di sanalah kendala terbesarnya.

### Rule Pega yang digantikan

| Jenis | Nama |
|---|---|
| Flow | tahap `Choose Surveyor` / `InputSurveyor` pada `Flow/Register_Flow.xml` |
| Harness | `InboxSurvey_Harness` · `SurveyorsInbox` · `DetailSurveyorsInbox` · `ViewDetailHasilSurveyorInternal1` |
| Master | `MasterLoginSurvey` · `INSERT_SURVEYORLIST` |
| Objek DB | `INSERT_KPIADJUSTER` |

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | When rule `IsSurvey` — hilang dari export | **Tim Pega** (`R-16`) |
| **Keputusan** | **Rumus skor KPI adjuster tidak ada di mana pun.** `INSERT_KPIADJUSTER` hanya menyimpan **10 komponen tanpa agregasi** — tidak ada rule yang menghitung skornya | **Work Owner** |
| **Keputusan** | Kunci upsert hanya `CASEID` — **adjuster kedua menimpa yang pertama**. Benar? | **Work Owner** |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B08-001](issues/01-penugasan-surveyor-dan-hasil-survei.md) | Penugasan surveyor dan pencatatan hasil survei | `needs-info` |
| [TKT-B08-002](issues/02-kpi-adjuster.md) | Komponen dan skor KPI adjuster | `needs-info` |

## Daftar Tiket

### TKT-B08-001 — Penugasan surveyor dan pencatatan hasil survei

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** — When rule `IsSurvey` hilang |
| **Modul** | **B-8 Choose Surveyor** · Gelombang: 4 · Bergantung pada: TKT-B06-001, TKT-U2-003 |
| **Requirement** | FR-B8 |
| **Keputusan** | D-12, D-16 |
| **ADR** | 0010, 0019 |
| **Risiko** | R-16 |
| **Rule Pega yang digantikan** | tahap **`Choose Surveyor`** / `InputSurveyor` pada `Flow/Register_Flow.xml` · harness `InboxSurvey_Harness`, `SurveyorsInbox`, `ViewDetailHasilSurveyorInternal1` · `INSERT_SURVEYORLIST` · master `MasterLoginSurvey` |
| **Peran penguji gerbang 2** | **PNCSurveyor** dan **PncPICTeknik** |
| **Label** | `modul::B-8` `tipe::migrasi` `status::needs-info` `prioritas::sedang` `gelombang::4` |
| **Milestone** | Gelombang 4 — Persetujuan |
| **Berkas sumber** | `ticketing/B-8-Choose-Surveyor-dan-Hasil-Survei/issues/01-penugasan-surveyor-dan-hasil-survei.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Petugas teknis menugaskan surveyor, dan surveyor mencatat temuan lapangan beserta **foto
dokumentasi** — termasuk dari perangkat di lapangan.

Nilai bisnisnya: hasil survei adalah **dasar penetapan nilai kerugian**. Survei yang lambat
dicatat memanjangkan TAT, dan foto yang tidak terunggah membuat penilaian dilakukan tanpa bukti.

#### Ruang lingkup

- Pemilihan surveyor dari master, memisahkan **internal (ASM)** dan **eksternal**.
- Pencatatan hasil survei: temuan, tanggal survei, lokasi survei, dan catatan.
- **Unggah foto dokumentasi** memakai komponen unggah `TKT-U2-003`, melalui satu jalur API storage
  (`ADR-0010`).
- Dukungan **peramban ponsel** untuk pengisian di lapangan (`D-12`).

#### Non-goal

- **Tidak** menghitung KPI adjuster — itu `TKT-B08-002`.
- **Tidak** menetapkan nilai kerugian — itu `B-5`.
- **Tidak** membangun mekanisme penyimpanan dokumen — itu `S-1`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **When rule `IsSurvey`** — hilang dari export | **Tim Pega** (`R-16`) | Menentukan **kapan** sebuah klaim wajib disurvei; tanpa itu syaratnya harus ditebak |
| Jenis surveyor per objek (`jenisSurveyor` pada objek) — daftar sahnya | **Work Owner** | Menentukan isi master surveyor |

#### Acceptance criteria

- ☐ Surveyor dipilih dari master; surveyor **tidak aktif tidak muncul** sebagai pilihan — diuji.
- ☐ Surveyor internal dan eksternal **dibedakan**, dan surveyor eksternal memerlukan data
      kontak — diuji keduanya.
- ☐ Hasil survei dapat disimpan beserta **minimal satu foto**, dan foto dapat dibuka kembali —
      diuji.
- ☐ Pengisian hasil survei berfungsi pada peramban ponsel lebar **390 px**, termasuk unggah dari
      kamera — diuji.
- ☐ Kegagalan unggah foto **tidak membatalkan** hasil survei yang sudah diisi — pengguna dapat
      mengulang unggahan tanpa mengetik ulang (`ADR-0010`).
- ☐ Perubahan hasil survei tercatat di jejak audit.
- ☐ Gerbang 1: penugasan dan hasil survei **sama dengan Pega** pada 20 klaim contoh.
- ☐ Gerbang 2: UAT **PNCSurveyor** di lapangan, bukan hanya di kantor.

#### Dependency / Blocked by

`TKT-B06-001` · `TKT-U2-003` · `TKT-S1-001` (jalur penyimpanan dokumen). **Terhalang Tim Pega.**

#### Constraint keamanan, data, operasional

- Foto survei dapat memuat **data nasabah dan lokasi**; ia tunduk pembatasan akses yang sama dengan
  dokumen klaim (`S-1`).
- **Tidak ada atomisitas** antara foto di storage dan metadatanya (`ADR-0010`) — kegagalan setelah
  foto terkirim harus diberitahukan, bukan diam.
- Pengisian di lapangan berarti **jaringan tidak stabil** — kegagalan jaringan harus dapat
  dibedakan dari penolakan server (`TKT-U2-003`).

#### Migrasi skema / rollout / rollback

Menambah tabel hasil survei dan kaitan fotonya. Backward-compatible.

**Rollback:** hasil survei yang tersimpan tetap ada; foto di storage **tidak ikut terhapus**
(`ADR-0010`).

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/survei/... -run TestPenugasanSurveyor
go test ./internal/app/survei/... -run TestUnggahFotoGagalTidakMembatalkan
npm run test:e2e -- --grep "hasil survei" --device "iPhone 12"
go run ./cmd/s8 banding --modul B-8 --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Tahap `Choose Surveyor` pada flow utama | `Flow/Register_Flow.xml` |
| Surveyor internal dan eksternal | `CONTEXT.md` — **Surveyor / Loss Adjuster** |
| Pemakaian lapangan | `D-12` · `docs/Steering/06-MODULE-BREAKDOWN.md` `B-8` |
| Dokumen lewat satu jalur API storage | `D-16` · `ADR-0010` |
| `IsSurvey` hilang | `docs/verifikasi-bukti-adr.md` §15 baris `B-8` · `R-16` |

#### Comments

### TKT-B08-002 — Komponen dan skor KPI adjuster

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan** — rumus skor tidak ada di mana pun |
| **Modul** | **B-8 Choose Surveyor** · Gelombang: 4 · Bergantung pada: TKT-B08-001 |
| **Requirement** | FR-B8 |
| **Keputusan** | D-15 |
| **ADR** | 0025, 0026 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | `INSERT_KPIADJUSTER` — menyimpan **10 komponen**, **tanpa agregasi** |
| **Peran penguji gerbang 2** | **PncManagerAdmin** dan **PncPICTeknik** |
| **Label** | `modul::B-8` `tipe::aturan-bisnis` `status::needs-info` `prioritas::sedang` `gelombang::4` |
| **Milestone** | Gelombang 4 — Persetujuan |
| **Berkas sumber** | `ticketing/B-8-Choose-Surveyor-dan-Hasil-Survei/issues/02-kpi-adjuster.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Penilaian kinerja adjuster yang **dapat dipertanggungjawabkan** — komponennya tercatat, dan
skornya dihitung dengan rumus yang diketahui.

Nilai bisnisnya: KPI adjuster memengaruhi siapa yang dipilih untuk survei berikutnya, dan pada
adjuster eksternal ia menyentuh hubungan kerja. Skor yang tidak dapat dijelaskan asalnya sulit
dipakai sebagai dasar keputusan.

#### Ruang lingkup

- Pencatatan **10 komponen KPI** yang sudah ada di sistem lama.
- **Perhitungan skor** dari komponen-komponen itu — setelah rumusnya ditetapkan.
- Bobot komponen sebagai **master data** (`ADR-0025`), bukan konstanta di kode.
- Riwayat skor per periode, sehingga perubahan kinerja terlihat.

#### Non-goal

- **Tidak** mengarang rumus. Bila rumusnya memang tidak pernah ada, itu temuan yang dilaporkan —
  bukan celah yang ditambal dengan tebakan.
- **Tidak** membangun laporan KPI — itu `S-7`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Rumus skor KPI adjuster tidak ada di mana pun.** `INSERT_KPIADJUSTER` hanya **menyimpan 10 komponen**; tidak ada rule yang mengagregasinya menjadi skor | **Work Owner** | Tanpa rumus, yang dapat dibangun hanyalah pencatatan komponen — dan itu **bukan KPI**. Kemungkinannya: rumusnya ada di luar sistem (spreadsheet), atau memang tidak pernah ada |
| **Kunci upsert hanya `CASEID`** — sehingga **adjuster kedua pada klaim yang sama menimpa yang pertama**. Benar? | **Work Owner** | Bila salah, data KPI adjuster pertama **hilang tanpa jejak** setiap kali ada adjuster kedua |

#### Acceptance criteria

> Butir 1 dan 2 berlaku apa pun jawabannya; sisanya menunggu rumus.

- ☐ Kesepuluh komponen KPI tercatat per adjuster per klaim — diuji.
- ☐ **Adjuster kedua pada klaim yang sama tidak menimpa yang pertama** — diuji dengan dua
      adjuster; keduanya tersimpan. Bila Work Owner memutuskan perilaku lama benar, AC ini dibalik
      dan alasannya dicatat.
- ☐ Skor dihitung dari komponen memakai bobot dari **master**, bukan konstanta — diuji dengan
      mengubah bobot dan memeriksa skor berubah.
- ☐ Riwayat skor per periode tersimpan; skor lama **tidak ditimpa** (`ADR-0012`).
- ☐ Perubahan bobot tercatat di jejak audit — ia mengubah penilaian orang.
- ☐ Gerbang 1: komponen yang tersimpan **sama dengan Pega**; **skor tidak dapat dibandingkan**
      karena sistem lama tidak menghitungnya — itu dinyatakan terbuka, bukan diklaim setara.

#### Dependency / Blocked by

`TKT-B08-001` · `TKT-F4-001` (master bobot). **Terhalang dua keputusan Work Owner.**

#### Constraint keamanan, data, operasional

- KPI menyentuh **penilaian orang**, termasuk mitra eksternal. Perubahan bobot dan skor wajib
  tercatat.
- Menimpa data adjuster pertama (perilaku sistem lama) berarti **kehilangan data tanpa jejak** —
  itu bertentangan dengan `ADR-0012`, dan karena itu perlu keputusan sadar, bukan diwarisi.

#### Migrasi skema / rollout / rollback

Menambah tabel komponen dan skor KPI. Backward-compatible.

**Rollback:** komponen yang tersimpan tetap ada; perhitungan skor dinonaktifkan.

#### Rencana verifikasi

> **Rencana — belum dijalankan, dan skornya belum dapat diuji.**

```bash
go test ./internal/app/kpi/... -run TestSepuluhKomponenTersimpan
go test ./internal/app/kpi/... -run TestAdjusterKeduaTidakMenimpa
go run ./cmd/tools/cek-kpi-tanpa-rumus    # laporkan komponen tanpa agregasi
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `INSERT_KPIADJUSTER` menyimpan 10 komponen tanpa agregasi | `docs/verifikasi-bukti-adr.md` §15 baris `B-8` |
| Kunci upsert hanya `CASEID` | idem |
| Nilai bisnis tidak boleh di-hardcode | `D-15` · `ADR-0025` |
| Tidak ada penghapusan fisik data bernilai bisnis | `D-66` · `ADR-0012` |

#### Comments

# 16. B-9 · PLA, Pre-DLA & DLA ke Reasuransi

*Modul Bisnis Inti · folder `docs/ticketing/B-9-PLA-PreDLA-dan-DLA/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **PLA / DLA** — inbox `InboxPLADLA`, `InboxPLA_harness`; activity `LetterOfAssignment_Act`, `LetterOfAssignment2_Act`, `SendDLAAutoSaatGeneratedDLA`; procedure `INSERT_PLADLA` |
| **Kode modul** | `B-9` |
| **Gelombang** | 5 — Nilai dan pihak luar |
| **Ukuran** | 69 activity (gabungan dengan spreading) |
| **Bergantung pada** | `B-4` Spreading · `B-5` Input Estimasi |
| **Kesiapan** | **SEBAGIAN** — **lepas dari `BRD §21.4`** (`D-55`) |

### Apa yang dikerjakan modul ini

Menerbitkan **pemberitahuan bertahap** kepada koasuransi dan reasuransi tentang kerugian yang
terjadi:

| Dokumen | Kapan |
|---|---|
| **PLA** — Preliminary Loss Advice | pemberitahuan awal, berdasarkan estimasi |
| **Pre-DLA** | pemberitahuan **sebelum akseptasi** dilakukan |
| **DLA** — Definite Loss Advice | pemberitahuan final setelah nilai ditetapkan |

### Yang dilepaskan `ADR-0007`

`Database/INSERT_PLADLA.prc` melakukan **`COMMIT` sembilan kali** (`:69`, `:74`, `:79`, `:138`,
`:143`, `:148`, `:179`, `:184`, `:189`), dengan satu-satunya `ROLLBACK` di `:198` yang terjadi
**setelah** commit — sehingga tidak memulihkan apa pun.

Setelah logikanya naik ke Go, penerbitan PLA/DLA **dapat dibuat atomik**.

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | Status `VALID`/`INVALID` `GET_GROUPBUSINESS_XOL` di produksi | **DBA** |
| **Keputusan** | **Tiga perilaku berbeda untuk PLA, DLA, dan PREDLA — mana yang benar?** | **Work Owner** |
| **Keputusan** | PLA tanpa nilai sah? · PLA/DLA tanpa email reasuradur sah? | **Work Owner** |
| **Keputusan** | Kunci duplikat PLA **5 kolom** versus DLA **6 kolom** — sengaja? | **Work Owner** |
| **Keputusan** | `T_PREDLALIST` masih dipakai? | **Work Owner** |

**Sudah diputuskan:** `TTGLPLADLA` yang diterima lalu dibuang **bukan cacat** — tanggal PLA/DLA
memang tanggal sistem menerbitkan dokumen (`D-49` butir 7, **direplikasi**).

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B09-001](issues/01-penerbitan-pla-predla-dla.md) | Penerbitan PLA, Pre-DLA, dan DLA | `needs-info` |
| [TKT-B09-002](issues/02-pengiriman-ke-reasuransi-dan-koasuransi.md) | Pengiriman dokumen ke reasuransi dan koasuransi | `needs-info` |

## Daftar Tiket

### TKT-B09-001 — Penerbitan PLA, Pre-DLA, dan DLA

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | terhalang keputusan |
| **Modul** | **B-9 PLA/Pre-DLA/DLA** · Gelombang: 5 · Bergantung pada: TKT-B04-001, TKT-B05-001, TKT-F2-003 |
| **Requirement** | FR-B9 |
| **Keputusan** | D-02, D-49 butir 7, D-68 |
| **ADR** | 0007, 0017 |
| **Risiko** | R-01 |
| **Rule Pega yang digantikan** | `Database/INSERT_PLADLA.prc` — **9 `COMMIT`** (`:69`, `:74`, `:79`, `:138`, `:143`, `:148`, `:179`, `:184`, `:189`), `ROLLBACK` di `:198` **setelah** commit; parameter `TTGLPLADLA` diterima lalu **dibuang**, diganti `SYSDATE` (`:67`, `:136`, `:177`) · inbox `InboxPLADLA`, `InboxPLA_harness` |
| **Peran penguji gerbang 2** | **PncPLADLA** dan **TreatyIn** |
| **Label** | `modul::B-9` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::5` |
| **Milestone** | Gelombang 5 — Nilai dan pihak luar |
| **Berkas sumber** | `ticketing/B-9-PLA-PreDLA-dan-DLA/issues/01-penerbitan-pla-predla-dla.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Ketiga dokumen pemberitahuan dapat diterbitkan, dan penerbitannya **berhasil seluruhnya atau gagal
seluruhnya**.

Nilai bisnisnya konkret: hari ini penerbitan menempuh **sembilan `COMMIT`**, dan `ROLLBACK`-nya
dijalankan **setelah** commit — sehingga tidak memulihkan apa pun. Bila proses berhenti di tengah,
sebagian pemberitahuan ke reasuransi sudah tercatat dan sebagian belum, **tanpa cara
membatalkannya**.

#### Ruang lingkup

- Penerbitan **PLA**, **Pre-DLA**, dan **DLA** dengan data dari spreading (`B-4`) dan nilai
  settlement (`B-5`).
- Seluruh penerbitan di dalam **satu transaksi** (`TKT-F2-003`) — inilah yang dilepaskan
  `ADR-0007`.
- **Tanggal dokumen diambil dari waktu sistem**, bukan dari pilihan pengguna — perilaku ini
  **direplikasi secara sadar** (`D-49` butir 7), dan parameter `TTGLPLADLA` **dihapus seluruhnya**.
- Pemeriksaan duplikat sebelum menerbitkan.

#### Non-goal

- **Tidak** mengirim dokumen — itu `TKT-B09-002`.
- **Tidak** memanggil `INSERT_PLADLA` (`ADR-0007`).

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Tiga perilaku berbeda untuk PLA, DLA, dan PREDLA — mana yang benar?** Ketiganya diterbitkan lewat procedure yang sama tetapi dengan jalur berbeda | **Work Owner** | Menentukan aturan penerbitan masing-masing; menebak berarti salah pada salah satunya |
| **PLA tanpa nilai sah?** | **Work Owner** | Menentukan apakah PLA boleh terbit sebelum estimasi ada |
| **Kunci duplikat PLA 5 kolom versus DLA 6 kolom — sengaja?** | **Work Owner** | Bila tidak sengaja, salah satunya punya celah duplikasi |
| **`T_PREDLALIST` masih dipakai?** | **Work Owner** | Bila tidak, Pre-DLA tidak perlu dibawa sama sekali |
| Status `VALID`/`INVALID` `GET_GROUPBUSINESS_XOL` di produksi | **DBA** | Menentukan perlakuan XOL pada penerbitan |

#### Acceptance criteria

- ☐ Penerbitan yang gagal di tengah **tidak meninggalkan satu baris pun** — diuji dengan
      kegagalan sengaja pada baris ke-3 dari 5. Inilah bukti apa yang dilepaskan `ADR-0007`.
- ☐ **Tanggal dokumen sama dengan waktu sistem**, dan parameter tanggal dari pengguna **tidak
      ada sama sekali** di kontrak — diuji pemindaian.
- ☐ Pemeriksaan duplikat menolak penerbitan kedua untuk kunci yang sama — diuji untuk PLA dan
      DLA masing-masing.
- ☐ **Nol pemanggilan stored procedure** — diuji pemindaian.
- ☐ Penerbitan tercatat di jejak audit dengan pelaku dan waktu.
- ☐ Gerbang 1: isi dokumen **sama dengan Pega** pada 20 klaim contoh. **Perilaku saat gagal
      berbeda secara sengaja** — kasus uji dirancang menyadarinya, atau ia melaporkan selisih palsu
      (`docs/Steering/14-TESTING-STRATEGY.md` §6.4).
- ☐ Gerbang 2: UAT **PncPLADLA** dan **TreatyIn**.

#### Dependency / Blocked by

`TKT-B04-001`, `TKT-B05-001`, `TKT-F2-003`. **Terhalang empat keputusan dan satu artefak DBA.**

#### Constraint keamanan, data, operasional

- Dokumen ini **dikirim ke pihak luar** — koasuransi, reasuransi, broker. Kesalahan isinya terlihat
  oleh mereka, bukan hanya internal.
- **Kontrak galat berbasis string `ErrMsg` tidak dibawa** (`ADR-0007`) — pada enam procedure
  `ErrMsg` tidak di-set pada jalur sukses sehingga `NULL` berarti berhasil.
- Nomor klaim yang tercetak di dokumen memakai format baru `PNCN.YY.xxxx` untuk klaim sistem baru
  (`ADR-0009`) — pihak luar akan menerima **dua bentuk nomor**.

#### Migrasi skema / rollout / rollback

Menambah tabel PLA/DLA milik aplikasi. Backward-compatible.

**Rollback:** dokumen yang telanjur diterbitkan **tidak dapat ditarik** — ia sudah dikirim ke
pihak luar. Rollback berarti menghentikan penerbitan baru.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/pladla/... -run TestPenerbitanAtomik
go test ./internal/app/pladla/... -run TestTanggalDariWaktuSistem
go test ./internal/app/pladla/... -run TestTolakDuplikat
grep -rIn "CALL \|INSERT_PLADLA" internal/app/pladla/    # HARUS 0 baris
go run ./cmd/s8 banding --modul B-9 --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 9 `COMMIT` dan `ROLLBACK` sesudahnya | `Database/INSERT_PLADLA.prc:69,74,79,138,143,148,179,184,189,198` |
| `TTGLPLADLA` dibuang, diganti `SYSDATE` | `:67`, `:136`, `:177` · `D-49` butir 7 **REPLIKASI** |
| `B-9` dapat dibuat atomik | `D-68` · `ADR-0007` |
| Lepas dari `BRD §21.4` | `D-55` · `ADR-0028` |
| Istilah PLA, Pre-DLA, DLA | `CONTEXT.md` |

#### Comments

### TKT-B09-002 — Pengiriman dokumen ke reasuransi dan koasuransi

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | terhalang keputusan dan `D-40` |
| **Modul** | **B-9 PLA/Pre-DLA/DLA** · Gelombang: 5 · Bergantung pada: TKT-B09-001, TKT-S3-001 |
| **Requirement** | FR-B9 |
| **Keputusan** | D-67, D-40 |
| **ADR** | 0011, 0025 |
| **Risiko** | R-17 |
| **Rule Pega yang digantikan** | `Activity/LetterOfAssignment_Act-Act.xml:10011` (BCC broker) · `Activity/LetterOfAssignment2_Act-Act.xml:7725` (`<To>` **akun Gmail pribadi**) · `Activity/SendDLAAutoSaatGeneratedDLA-Act.xml:2706` (**fallback penerima berdasarkan hostname dev**) · `Activity/CheckerSendEmailApproveReject-Act.xml:2299` |
| **Peran penguji gerbang 2** | **PncPLADLA** dan **TreatyIn** |
| **Label** | `modul::B-9` `tipe::integrasi` `status::needs-info` `prioritas::tinggi` `gelombang::5` |
| **Milestone** | Gelombang 5 — Nilai dan pihak luar |
| **Berkas sumber** | `ticketing/B-9-PLA-PreDLA-dan-DLA/issues/02-pengiriman-ke-reasuransi-dan-koasuransi.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Dokumen PLA/DLA sampai ke penanggung yang benar, dengan **penerima dari master** — bukan dari
alamat yang tertanam di dalam rule.

Nilai bisnisnya: hari ini penerima sebagian di-hardcode, salah satunya **akun Gmail pribadi**, dan
ada **fallback yang mengganti penerima berdasarkan nama server**. Ketika orangnya pindah atau
server dipindahkan, dokumen ke reasuransi **berhenti sampai** tanpa ada yang menyadarinya.

#### Ruang lingkup

- Pembentukan dokumen PLA/DLA sebagai **PDF** memakai engine sendiri (`ADR-0011`).
- Pengiriman lewat seam Notifier (`S-3`), dengan penerima dari **master Penerima Notifikasi**
  (`TKT-F4-003`).
- **Penghapusan fallback berbasis hostname** — perbedaan lingkungan menjadi konfigurasi, bukan
  deteksi nama server.
- Pencatatan pengiriman: tujuan, waktu, hasil.

#### Non-goal

- **Tidak** membangun pengirim email — itu `S-3`.
- **Tidak** memutuskan tujuan penyimpanan kredensial SMTP — itu `D-40`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **PLA/DLA tanpa email reasuradur sah?** | **Work Owner** | Menentukan apakah dokumen boleh terbit tanpa tujuan kirim |
| **Rotasi 3 password SMTP (31 lokasi)** dan tujuan penyimpanannya | **Tim Infra/Security** (`D-40`, `R-17`) | Pengiriman tidak dapat dijalankan di produksi tanpa kredensial yang tersimpan aman |
| Mailbox fungsional pengganti untuk penerima yang kini akun pribadi | **Work Owner** (`D-67`) | Pemetaan alamat lama ke mailbox baru |

#### Acceptance criteria

- ☐ Dokumen PLA/DLA terbentuk sebagai PDF dengan isi yang **sama dengan keluaran Pega** pada 10
      contoh — kriteria kesamaan (isi atau visual) mengikuti keputusan `ADR-0011`.
- ☐ Penerima diambil **dari master**; mengubah master mengubah tujuan **tanpa deployment** —
      diuji.
- ☐ **Nol alamat email di kode** — diuji pemindaian; build gagal bila muncul.
- ☐ **Nol perilaku yang bergantung pada nama host** — diuji dengan menjalankan pada dua hostname
      berbeda: penerima **sama**.
- ☐ **Nol akun pribadi** di daftar penerima — diuji terhadap daftar domain yang diizinkan.
- ☐ Kegagalan pengiriman **tercatat dan terlihat**, dan dokumen tetap tersimpan — diuji dengan
      SMTP yang sengaja dimatikan.
- ☐ Pengiriman **tidak berada di dalam transaksi database** (`TKT-F2-003`).
- ☐ Gerbang 2: UAT **PncPLADLA** memeriksa dokumen yang benar-benar diterima penanggung uji.

#### Dependency / Blocked by

`TKT-B09-001` · `TKT-S3-001` · `TKT-F4-003` · `TKT-S2-001` (engine PDF). **Terhalang Work Owner dan
Tim Infra/Security.**

#### Constraint keamanan, data, operasional

- Dokumen memuat **data nasabah dan nilai klaim**, dan dikirim ke **pihak luar perusahaan**.
  Salah tujuan berarti kebocoran data.
- **3 password SMTP di 31 lokasi** dan `UseSSL=false` di seluruh kemunculannya (`R-17`) — menunggu
  `D-40`.
- Alamat email **tidak pernah ditulis lengkap** di dokumen proyek (`D-69`); di tiket ini pun
  dirujuk dengan `berkas:baris`.

#### Migrasi skema / rollout / rollback

Menambah tabel riwayat pengiriman. Backward-compatible.

**Rollback:** dokumen yang telanjur terkirim **tidak dapat ditarik**.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/pladla/... -run TestPenerimaDariMaster
grep -rInE "[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}" internal/app/pladla/   # HARUS 0
go test ./internal/app/pladla/... -run TestTanpaKetergantunganHostname
go test ./internal/app/pladla/... -run TestKegagalanKirimTercatat
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Akun Gmail pribadi sebagai `<To>` | `Activity/LetterOfAssignment2_Act-Act.xml:7725` |
| BCC broker eksternal | `Activity/LetterOfAssignment_Act-Act.xml:10011` |
| Fallback penerima berbasis hostname dev | `Activity/SendDLAAutoSaatGeneratedDLA-Act.xml:2706` |
| Tidak ada akun pribadi sebagai penerima | `D-67` |
| 3 password SMTP di 31 lokasi, `UseSSL=false` | `R-17` · `D-40` |
| PDF dibangun sendiri di Go | `D-11` · `ADR-0011` |

#### Comments

# 17. B-10 · Akseptasi, Transfer Kasir & LOD

*Modul Bisnis Inti · folder `docs/ticketing/B-10-Akseptasi-Transfer-Kasir-dan-LOD/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Akseptasi** — activity `InsertDataAkseptasiToLeader`, `InsertLogKasir_act`, `TransferCashierDataASM_act`, `HitupdateDataRekeningToKasir`; procedure `PEGA_JSON_OS_AKSEP_KLAIM`; master `MasterRekening` |
| **Kode modul** | `B-10` |
| **Gelombang** | 4 — Persetujuan |
| **Ukuran** | 31 activity |
| **Bergantung pada** | `B-5` Input Estimasi · `B-7` Komite |
| **Kesiapan** | **SEBAGIAN** — **lepas dari `BRD §21.4`** (`D-55`) |

### Apa yang dikerjakan modul ini

Menetapkan bahwa nilai klaim **disetujui untuk dibayarkan**, menerbitkan **Nomor Akseptasi**,
mengirim data pembayaran ke **kasir**, dan mencetak **LOD** — surat pemberitahuan nilai ganti rugi
kepada tertanggung.

Perbedaan yang penting dan sering tertukar:

| Istilah | Artinya |
|---|---|
| **Akseptasi** | persetujuan atas **nilai** yang akan dibayarkan |
| **LOD** | surat ke **tertanggung**, berbeda dari PLA/DLA yang ke koasuransi dan reasuransi |
| **OS — Outstanding** | akseptasi yang sudah diakui nilainya tetapi **belum selesai dibayar** |

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **`InsertDataAkseptasiToLeader`** · **`InsertLogKasir_act`** · **`TransferCashierDataASM_act`** — ketiganya dipanggil tetapi **tidak ada di export** | **Tim Pega** (`R-07`) |
| **Keputusan** | Header `Authorization` **di-hardcode di 3 Connect REST kasir** | **Tim Infra/Security** (`D-40`) |
| **Keputusan** | `PEGA_JSON_OS_AKSEP_KLAIM` memetakan `STS_PLA` ke kolom **`STS_DLA`** — benar? | **Work Owner** |
| **Catatan** | `ErrMsg` pada `ADD_NEWMASTERVIRTUALACCOUNT.prc:18` membawa **nomor virtual account sekaligus pesan galat** — kontrak itu tidak dibawa (`ADR-0007`) | — |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B10-001](issues/01-akseptasi-dan-nomor-akseptasi.md) | Akseptasi dan penerbitan Nomor Akseptasi | `needs-info` |
| [TKT-B10-002](issues/02-transfer-ke-kasir-dan-status-pembayaran.md) | Transfer ke kasir dan status pembayaran | `needs-info` |
| [TKT-B10-003](issues/03-cetak-lod.md) | Cetak LOD ke tertanggung | `needs-info` |

## Daftar Tiket

### TKT-B10-001 — Akseptasi dan penerbitan Nomor Akseptasi

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** — `InsertDataAkseptasiToLeader` tidak ada di export |
| **Modul** | **B-10 Akseptasi** · Gelombang: 4 · Bergantung pada: TKT-B05-001, TKT-B07-002 |
| **Requirement** | FR-B10 |
| **Keputusan** | D-19 |
| **ADR** | 0016, 0023, 0026 |
| **Risiko** | R-07 |
| **Rule Pega yang digantikan** | `InsertDataAkseptasiToLeader` — **dipanggil tetapi tidak ada di export** · `PEGA_JSON_OS_AKSEP_KLAIM` |
| **Peran penguji gerbang 2** | **PncManagerAdmin** dan **PNCKomite** |
| **Label** | `modul::B-10` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::4` |
| **Milestone** | Gelombang 4 — Persetujuan |
| **Berkas sumber** | `ticketing/B-10-Akseptasi-Transfer-Kasir-dan-LOD/issues/01-akseptasi-dan-nomor-akseptasi.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Nilai klaim dinyatakan **disetujui untuk dibayarkan**, dan sistem menerbitkan **Nomor Akseptasi**
sebagai penanda resminya.

Nilai bisnisnya: akseptasi adalah titik ketika perusahaan **mengikat diri untuk membayar**. Ia
tidak boleh terjadi sebelum seluruh jenjang komite menyetujui — dan itulah invarian `I-5`.

#### Ruang lingkup

- Penerbitan **Nomor Akseptasi** beserta tanggalnya, per Settlement Line.
- Penegakan invarian `I-5`: akseptasi **ditolak** bila masih ada jenjang komite yang belum
  menyetujui.
- Status **Outstanding**: akseptasi yang sudah diakui nilainya tetapi belum selesai dibayar.
- Jejak audit pada penerbitan dan pembatalan akseptasi.

#### Non-goal

- **Tidak** mengirim data ke kasir — itu `TKT-B10-002`.
- **Tidak** mencetak LOD — itu `TKT-B10-003`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **`InsertDataAkseptasiToLeader`** — dipanggil tetapi tidak ada di export | **Tim Pega** (`R-07`) | Aturan pembentukan data akseptasi ke "leader" tidak diketahui |
| **`PEGA_JSON_OS_AKSEP_KLAIM` memetakan `STS_PLA` ke kolom `STS_DLA`** — benar, atau salah pemetaan yang sudah berjalan? | **Work Owner** | Bila salah, status PLA dan DLA tertukar pada data Outstanding — dan itu memengaruhi laporan |

#### Acceptance criteria

- ☐ Akseptasi **ditolak** bila masih ada jenjang komite yang belum menyetujui — diuji pada klaim
      dengan 3 jenjang, dicoba pada jenjang ke-2 (invarian `I-5`).
- ☐ Nomor Akseptasi terbit **unik** — diuji 20 penerbitan, nol nomor ganda.
- ☐ Nilai akseptasi **tidak melebihi** nilai usulan dan tidak melebihi Sisa TSI — diuji kedua
      batas.
- ☐ Akseptasi berstatus **Outstanding** sampai pembayaran selesai — diuji perpindahan statusnya.
- ☐ Penerbitan dan pembatalan akseptasi menghasilkan **jejak audit** dengan nilai sebelum dan
      sesudah.
- ☐ Gerbang 1: nomor dan nilai akseptasi **sama dengan Pega** pada 20 klaim contoh.
- ☐ Gerbang 2: UAT **PncManagerAdmin**.

#### Dependency / Blocked by

`TKT-B05-001` · `TKT-B07-002` (keputusan komite) · `TKT-S5-002`. **Terhalang Tim Pega dan satu
keputusan.**

#### Constraint keamanan, data, operasional

- **Orang yang sama dapat menyetujui komite dan menerbitkan akseptasi** bila perannya memiliki
  kedua menu (`D-59`). Tidak ada kontrol teknis yang mencegahnya; jejak audit satu-satunya
  pengimbang — dan `BRD §21.2` kriteria #9 berlaku tanpa pengecualian.
- Nilai akseptasi memakai **presisi penuh** (`ADR-0016`).

#### Migrasi skema / rollout / rollback

Menambah kolom akseptasi pada Settlement Line dan tabel Outstanding. Backward-compatible.

**Rollback:** akseptasi yang sudah terbit **tetap sah** — ia sudah menjadi komitmen membayar.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/akseptasi/... -run TestTolakSebelumKomiteSelesai
go test ./internal/app/akseptasi/... -run TestNomorAkseptasiUnik
go run ./cmd/s8 banding --modul B-10 --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `InsertDataAkseptasiToLeader` tidak ada di export | `docs/verifikasi-bukti-adr.md` §15 baris `B-10` · `R-07` |
| `PEGA_JSON_OS_AKSEP_KLAIM` memetakan `STS_PLA` → `STS_DLA` | idem |
| Invarian `I-5` | `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Istilah Akseptasi dan Outstanding | `CONTEXT.md` |
| Lepas dari `BRD §21.4` | `D-55` |

#### Comments

### TKT-B10-002 — Transfer ke kasir dan status pembayaran

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak dan `D-40`** |
| **Modul** | **B-10 Akseptasi** · Gelombang: 4 · Bergantung pada: TKT-B10-001, TKT-S4-001 |
| **Requirement** | FR-B10 |
| **Keputusan** | D-25, D-40 |
| **ADR** | 0008, 0025 |
| **Risiko** | R-07, R-17 |
| **Rule Pega yang digantikan** | `InsertLogKasir_act` dan `TransferCashierDataASM_act` — **keduanya tidak ada di export** · `Activity/HitupdateDataRekeningToKasir-Act.xml:2151` (URL kasir sebagai konstanta) · **3 Connect REST kasir dengan header `Authorization` di-hardcode** |
| **Peran penguji gerbang 2** | **PncCollection** dan **PncManagerAdmin** |
| **Label** | `modul::B-10` `tipe::integrasi` `status::needs-info` `prioritas::tinggi` `gelombang::4` |
| **Milestone** | Gelombang 4 — Persetujuan |
| **Berkas sumber** | `ticketing/B-10-Akseptasi-Transfer-Kasir-dan-LOD/issues/02-transfer-ke-kasir-dan-status-pembayaran.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Data pembayaran klaim sampai ke sistem kasir **tepat satu kali**, dan status pembayarannya kembali
terlihat di klaim.

Nilai bisnisnya paling keras di sini: **pengiriman ganda ke kasir berarti pembayaran ganda**. Dan
integrasi ini bersifat asinkron, sehingga idempotensi bukan kenyamanan melainkan syarat.

#### Ruang lingkup

- Pengiriman data pembayaran ke sistem kasir lewat seam integrasi (`S-4`).
- **Idempotensi**: pengiriman ulang untuk akseptasi yang sama **tidak menghasilkan pembayaran
  kedua**.
- Pembaruan status pembayaran pada klaim setelah kasir memproses.
- Data rekening penerima dari master (`MasterRekening`).

#### Non-goal

- **Tidak** membangun sistem kasir.
- **Tidak** memutuskan tujuan penyimpanan kredensial — itu `D-40`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **`InsertLogKasir_act` dan `TransferCashierDataASM_act`** — tidak ada di export | **Tim Pega** (`R-07`) | Bentuk data yang dikirim ke kasir tidak diketahui |
| **Header `Authorization` di-hardcode di 3 Connect REST kasir** | **Tim Infra/Security** (`D-40`, `R-17`) | Kredensial tidak boleh dibawa apa adanya; tujuannya belum ditetapkan |
| Kontrak API kasir yang berlaku | **tim pemilik sistem kasir** | Menentukan bentuk permintaan, kode galat, dan cara idempotensi disepakati |

#### Acceptance criteria

- ☐ Pengiriman ke kasir untuk satu akseptasi **tepat satu kali**; pengiriman ulang dengan kunci
      yang sama **tidak menghasilkan pembayaran kedua** — diuji dengan tiga kali pengiriman.
- ☐ Kegagalan jaringan **masuk antrean dan dicoba ulang**, dengan jeda bertambah — diuji.
- ☐ Kegagalan yang berulang **terlihat** (tercatat dan dapat diketahui), bukan hilang diam-diam.
- ☐ Pengiriman **tidak berada di dalam transaksi database** (`TKT-F2-003`) — diuji pemindaian.
- ☐ Status pembayaran yang kembali dari kasir memperbarui klaim, dan perubahannya **tercatat di
      jejak audit**.
- ☐ **Nol kredensial di kode** — diuji pemindaian; kredensial dibaca dari konfigurasi
      (`TKT-F1-002`).
- ☐ Rekening penerima diambil dari master, bukan diketik bebas — diuji.
- ☐ Gerbang 2: UAT **PncCollection** dengan pembayaran uji yang benar-benar sampai ke kasir.

#### Dependency / Blocked by

`TKT-B10-001` · `TKT-S4-001` · `TKT-F4-001` (master rekening). **Terhalang Tim Pega, Tim
Infra/Security, dan tim kasir.**

#### Constraint keamanan, data, operasional

- **Pengiriman ganda berarti pembayaran ganda.** Idempotensi adalah syarat, bukan optimasi — ini
  disebut eksplisit pada tabel sistem eksternal Steering: *"Kasir — asinkron, idempoten wajib;
  tidak boleh kirim ganda"*.
- Data yang dikirim memuat **nomor rekening** — data nasabah; ia tidak boleh masuk log
  (`TKT-F1-003`).
- Header `Authorization` yang kini di-hardcode adalah bagian dari `R-17`.

#### Migrasi skema / rollout / rollback

Menambah tabel antrean pengiriman dan riwayatnya. Backward-compatible.

**Rollback:** pembayaran yang telanjur terkirim **tidak dapat ditarik dari sistem kasir** oleh
aplikasi ini — pembatalannya menempuh prosedur kasir.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/kasir/... -run TestKirimIdempoten
go test ./internal/adapter/kasir/... -run TestRetryDenganJedaBertambah
grep -rIn "http\." internal/app/akseptasi/ | grep -i "tx\|transaksi"     # HARUS 0 baris
grep -rInE "Authorization.*(Bearer|Basic)\s+[A-Za-z0-9]" internal/       # HARUS 0 baris
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `InsertLogKasir_act`, `TransferCashierDataASM_act` tidak ada di export | `docs/verifikasi-bukti-adr.md` §15 baris `B-10` · `R-07` |
| Header `Authorization` hardcode di 3 Connect REST kasir | idem |
| Kasir: asinkron, idempoten wajib, tidak boleh kirim ganda | `docs/Steering/10-API-STRATEGY.md` §8.3 |
| URL kasir sebagai konstanta | `Activity/HitupdateDataRekeningToKasir-Act.xml:2151` |

#### Comments

### TKT-B10-003 — Cetak LOD ke tertanggung

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | terhalang keputusan (kriteria kesamaan keluaran PDF) |
| **Modul** | **B-10 Akseptasi** · Gelombang: 4 · Bergantung pada: TKT-B10-001, TKT-S2-001 |
| **Requirement** | FR-B10 |
| **Keputusan** | D-11, D-57 |
| **ADR** | 0011, 0022 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | pencetakan LOD dan `Job Scheduler/JobSendAutoLODKlaimPersonal` (**harian, jam 20:54**, menjalankan `Act_SendAutoLODKlaimPersonal`) · `Activity/PrintPDFAcceptanceNote-Act.xml` |
| **Peran penguji gerbang 2** | **PncManagerAdmin** dan **PncAdmin** |
| **Label** | `modul::B-10` `tipe::migrasi` `status::needs-info` `prioritas::sedang` `gelombang::4` |
| **Milestone** | Gelombang 4 — Persetujuan |
| **Berkas sumber** | `ticketing/B-10-Akseptasi-Transfer-Kasir-dan-LOD/issues/03-cetak-lod.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Surat **LOD — Letter of Discharge** dapat dicetak dan dikirim ke **tertanggung**, berisi nilai
ganti rugi yang disetujui.

Nilai bisnisnya: LOD adalah dokumen yang **dipegang nasabah**. Ia berbeda dari PLA/DLA yang
ditujukan ke koasuransi dan reasuransi — dan perbedaan itu sering tertukar.

#### Ruang lingkup

- Pembentukan LOD sebagai **PDF** memakai engine sendiri (`ADR-0011`).
- Status cetak LOD pada Settlement Line, sehingga terlihat mana yang sudah dicetak.
- **Pengiriman otomatis LOD untuk lini Personal Accident** — perilaku job harian jam 20:54 yang
  ditemukan pada `D-57`.
- Pencatatan riwayat cetak dan kirim.

#### Non-goal

- **Tidak** membangun engine PDF — itu `S-2`; modul ini memakainya.
- **Tidak** memutuskan mekanisme penjadwal — itu `S-6` dan `ADR-0022`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Apakah keluaran PDF wajib identik secara visual dengan Pega, atau cukup identik secara isi?** | **Work Owner** (`ADR-0011`) | Menentukan kriteria kelulusan gerbang 1 untuk seluruh dokumen cetak |
| **Pengiriman otomatis LOD PA harian jam 20:54 — dipertahankan?** | **Work Owner** | Perilaku ini ditemukan dari `Job Scheduler/`, dan seperti `AutoAcceptKomite` ia berjalan **tanpa pengguna** |
| Mekanisme penjadwal pada dua instans | **Lead Engineer + Infra** (`ADR-0022`) | Job yang berjalan dua kali berarti **LOD terkirim dua kali** ke nasabah |

#### Acceptance criteria

- ☐ LOD terbentuk sebagai PDF berisi nilai ganti rugi yang disetujui, nomor klaim, dan data
      tertanggung — diuji pada 10 contoh.
- ☐ Kesamaan dengan keluaran Pega diuji sesuai kriteria yang ditetapkan Work Owner (isi atau
      visual) — 10 contoh.
- ☐ Status cetak tercatat; mencetak ulang **tidak menggandakan riwayat pembayaran** — diuji.
- ☐ Pengiriman otomatis LOD PA berjalan **tepat satu kali per hari**, walau dua instans hidup —
      diuji (`ADR-0022`).
- ☐ Kegagalan pembentukan PDF **tidak** membatalkan akseptasi — diuji.
- ☐ Gerbang 2: UAT **PncManagerAdmin** memeriksa LOD yang benar-benar tercetak.

#### Dependency / Blocked by

`TKT-B10-001` · `TKT-S2-001` (engine PDF) · `TKT-S6-001` (penjadwal). **Terhalang tiga keputusan.**

#### Constraint keamanan, data, operasional

- LOD memuat **data nasabah dan nilai ganti rugi**, dan dikirim ke luar perusahaan — salah tujuan
  berarti kebocoran.
- **LOD terkirim dua kali** akibat job berjalan ganda akan sampai ke nasabah; ini kesalahan yang
  terlihat pihak luar.
- Nilai pada LOD memakai pembulatan tampilan yang **sama dengan layar** (`TKT-U2-004`) — agar nasabah
  dan petugas melihat angka yang sama.

#### Migrasi skema / rollout / rollback

Menambah kolom status cetak dan tabel riwayat kirim. Backward-compatible.

**Rollback:** LOD yang telanjur terkirim **tidak dapat ditarik**.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/lod/... -run TestBentukPDF
go test ./internal/app/lod/... -run TestCetakUlangTidakMenggandakan
go test ./internal/app/jadwal/... -run TestLODOtomatisSekaliSehari
go run ./cmd/tools/banding-pdf lod-contoh/ baseline-pega/
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| LOD ditujukan ke tertanggung, berbeda dari PLA/DLA | `CONTEXT.md` — **LOD** |
| `JobSendAutoLODKlaimPersonal` harian 20:54 | `D-57` · `Job Scheduler/` |
| PDF dibangun sendiri di Go | `D-11` · `ADR-0011` |
| Kriteria kesamaan keluaran PDF belum ditetapkan | `ADR-0011` Pertanyaan terbuka |

#### Comments

# 18. B-11 · RCL, PUCL, Compliance, Investigator & Analyst Doctor

*Modul Bisnis Inti · folder `docs/ticketing/B-11-RCL-PUCL-Compliance-dan-Investigator/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | tahap **RCL/PUCL**, **Compliance**, **Investigator**, **Analyst Doctor**, **RCLDokter** pada `Flow/Register_Flow.xml`; harness `RCL_Harness`, `RCLPUCL_Harness`, `inboxCompliance_Harness`, `InboxInvestigator_Harness`, `inboxAnalystDoctor_Harness`, `PNC_MasterTolakKlaim` |
| **Kode modul** | `B-11` |
| **Gelombang** | 4 — Persetujuan |
| **Ukuran** | 6 activity + jalur alur |
| **Bergantung pada** | `B-6` Penugasan |
| **Kesiapan** | **TERHALANG** |

### Apa yang dikerjakan modul ini

Lima jalur penanganan khusus yang semuanya **keluar dari alur normal**:

| Jalur | Isi |
|---|---|
| **RCL** — Rejected Klaim | penolakan klaim |
| **PUCL** — Proses Ulang Klaim | klaim diproses ulang setelah ditolak atau ditutup |
| **Compliance** | pemeriksaan kepatuhan sebelum klaim diselesaikan |
| **Investigator** | penyelidikan klaim yang mencurigakan |
| **Analyst Doctor / RCL Dokter** | penilaian dan penolakan yang memerlukan pertimbangan medis |

Ketiganya yang pertama memakai **Workbasket** (antrean bersama); dua terakhir memakai **Worklist**.

### Kenapa modul ini TERHALANG

Modul inilah yang paling banyak bergantung pada **lompatan lateral** — dan justru di situ artefaknya
paling banyak hilang.

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | Ticket rule `SendtoPUCL` · `RCLDokter` | **Tim Pega** (`R-16`) |
| **Artefak** | Router `RouterRCLDokter` | **Tim Pega** (`R-04`) |
| **Artefak** | When rule `ElseRCLMSIG` · `IsNotViewClaim` | **Tim Pega** (`R-16`) |
| **Artefak** | Workbasket `RCLPUCL` dan `CompliancePNC` | **Tim Pega** |
| **Keputusan** | **Siapa yang boleh memicu transisi lateral?** Sistem lama **nol pagar izin** | **Work Owner** |
| **ADR `Proposed`** | `ADR-0021` — 14 dari 17 Ticket rule tanpa pemicu yang diketahui | Tim Pega → Work Owner |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B11-001](issues/01-penolakan-rcl-dan-proses-ulang-pucl.md) | Penolakan (RCL) dan proses ulang (PUCL) | `needs-info` |
| [TKT-B11-002](issues/02-compliance-investigator-analyst-doctor.md) | Compliance, Investigator, dan Analyst Doctor | `needs-info` |

## Daftar Tiket

### TKT-B11-001 — Penolakan (RCL) dan proses ulang (PUCL)

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak dan keputusan** |
| **Modul** | **B-11 RCL/PUCL** · Gelombang: 4 · Bergantung pada: TKT-B06-001 |
| **Requirement** | FR-B11 |
| **Keputusan** | D-26, D-59 |
| **ADR** | 0019, 0021, 0023, 0026 |
| **Risiko** | R-16 |
| **Rule Pega yang digantikan** | tahap **`RCL/PUCL`** pada `Flow/Register_Flow.xml` · harness `RCL_Harness`, `RCLPUCL_Harness`, `PNC_MasterTolakKlaim` · Ticket rule `SendtoPUCL` (`Flow/Register_Flow.xml:3197`) — **hilang dari export** · pemicu terverifikasi: `Activity/KomitePost_Reject-Act.xml:3172` dengan kondisi `GroupPanel=="002"` (`:3121`) |
| **Peran penguji gerbang 2** | **PncRCLPUCL** dan **PncManagerAdmin** |
| **Label** | `modul::B-11` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::4` |
| **Milestone** | Gelombang 4 — Persetujuan |
| **Berkas sumber** | `ticketing/B-11-RCL-PUCL-Compliance-dan-Investigator/issues/01-penolakan-rcl-dan-proses-ulang-pucl.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Klaim dapat **ditolak** dengan alasan yang tercatat, dan klaim yang sudah ditolak atau ditutup
dapat **diproses ulang**.

Nilai bisnisnya: penolakan adalah keputusan yang **berhadapan langsung dengan nasabah**. Alasan
penolakan yang tidak tercatat rapi membuat sengketa tidak dapat ditelusuri — dan proses ulang yang
tidak terlacak membuat klaim yang sama bisa dinilai dua kali dengan hasil berbeda.

#### Ruang lingkup

- Penolakan klaim dengan **alasan dari master** (`PNC_MasterTolakKlaim`), bukan teks bebas.
- Proses ulang klaim yang sudah ditolak atau ditutup, dengan jejak yang menghubungkan keduanya.
- Antrean **Workbasket `RCLPUCL`** — antrean bersama, diambil siapa pun yang berwenang.
- **Lompatan lateral ke PUCL** — satu-satunya pemicu yang terverifikasi ada di
  `Activity/KomitePost_Reject-Act.xml:3172` dengan kondisi `GroupPanel=="002"`.

#### Non-goal

- **Tidak** menangani penolakan medis — itu `TKT-B11-002` (RCL Dokter).
- **Tidak** membangun mekanisme penugasan — itu `B-6`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Ticket rule `SendtoPUCL`** — dirujuk `Flow/Register_Flow.xml:3197`, tidak ada di export | **Tim Pega** (`R-16`) | Satu pemicunya terverifikasi (`GroupPanel=="002"`), tetapi **apakah hanya itu** tidak diketahui |
| **When rule `ElseRCLMSIG`** dan **`IsNotViewClaim`** | **Tim Pega** | Percabangan penolakan dan pembatasan tampilan |
| **Workbasket `RCLPUCL`** — definisi antrean | **Tim Pega** | Menentukan siapa yang boleh mengambil |
| **Siapa yang boleh memicu transisi lateral?** Sistem lama **nol pagar izin** | **Work Owner** | Lompatan ke PUCL melewati alur normal; `D-59` bersatuan menu dan belum mengaturnya |

#### Acceptance criteria

- ☐ Penolakan menuntut **alasan dari master**; teks bebas tanpa alasan **ditolak** — diuji.
- ☐ Klaim yang ditolak berpindah ke status yang benar, dan **tercatat di jejak audit** dengan
      pelaku, waktu, dan alasannya.
- ☐ Proses ulang membuat kaitan ke klaim asal — riwayat keduanya **dapat ditelusuri dua arah**.
- ☐ Antrean `RCLPUCL` bersifat **Workbasket**: tugas dapat diambil siapa pun yang berwenang, dan
      **penguncian mencegah dua orang mengerjakannya** (`TKT-B06-003`).
- ☐ Lompatan lateral ke PUCL pada kondisi `GroupPanel=="002"` berperilaku sama dengan Pega —
      diuji.
- ☐ Setiap lompatan lateral **tercatat di jejak audit** dengan pelaku — karena tidak ada pagar
      izin, catatan inilah satu-satunya kontrol.
- ☐ Gerbang 1: hasil penolakan dan proses ulang **sama dengan Pega** pada 20 klaim contoh.
- ☐ Gerbang 2: UAT **PncRCLPUCL**.

#### Dependency / Blocked by

`TKT-B06-001`, `TKT-B06-003`, `TKT-S5-002`. **Terhalang Tim Pega dan Work Owner.**

#### Constraint keamanan, data, operasional

- **Nol pagar izin pada transisi lateral** di sistem lama. Karena `D-59` juga tidak mengenal
  pemisahan tugas, jejak audit adalah **satu-satunya** kontrol yang tersisa — pencatatannya wajib.
- Alasan penolakan **terlihat nasabah** dalam sengketa; ia tidak boleh memuat catatan internal.

#### Migrasi skema / rollout / rollback

Menambah tabel penolakan dan kaitan proses ulang. Backward-compatible.

**Rollback:** penolakan yang sudah tercatat tetap ada.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/rcl/... -run TestPenolakanWajibAlasanMaster
go test ./internal/app/rcl/... -run TestProsesUlangTertautKlaimAsal
go test ./internal/app/rcl/... -run TestLompatanLateralTercatat
go run ./cmd/s8 banding --modul B-11 --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Ticket rule `SendtoPUCL` hilang | `Flow/Register_Flow.xml:3197` · `ADR-0021` |
| Pemicu terverifikasi `GroupPanel=="002"` | `Activity/KomitePost_Reject-Act.xml:3172` dan `:3121` |
| `RCL/PUCL` memakai Workbasket | `D-26` · `ADR-0019` |
| Nol pagar izin pada transisi lateral | `docs/verifikasi-bukti-adr.md` §15 baris `B-11` |
| Istilah RCL dan PUCL | `CONTEXT.md` |

#### Comments

### TKT-B11-002 — Compliance, Investigator, dan Analyst Doctor

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak dan keputusan** |
| **Modul** | **B-11 RCL/PUCL** · Gelombang: 4 · Bergantung pada: TKT-B06-001, TKT-B11-001 |
| **Requirement** | FR-B11, FR-R2 |
| **Keputusan** | D-26, D-59 |
| **ADR** | 0019, 0021, 0023 |
| **Risiko** | R-16 |
| **Rule Pega yang digantikan** | tahap **`Compliance`**, **`Investigator`**, **`Analyst Doctor`**, **`RCLDokter`**, **`Send To Analis`** pada `Flow/Register_Flow.xml` · harness `inboxCompliance_Harness`, `InboxInvestigator_Harness`, `inboxAnalystDoctor_Harness` · Ticket rule `CompliancePNC` (`:3844`), `SendToInvestigator` (`:3624`), `RCLDokter` (`:2954`) — **ketiganya hilang** · router `RouterRCLDokter` — **hilang** |
| **Peran penguji gerbang 2** | **PncComplience**, **PncInvestigator**, **PncAnalystDoctor** |
| **Label** | `modul::B-11` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::4` |
| **Milestone** | Gelombang 4 — Persetujuan |
| **Berkas sumber** | `ticketing/B-11-RCL-PUCL-Compliance-dan-Investigator/issues/02-compliance-investigator-analyst-doctor.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Tiga jalur pemeriksaan khusus berjalan dengan inbox, kewenangan, dan jejaknya masing-masing.

Nilai bisnisnya berbeda-beda dan semuanya penting: **Compliance** mencegah klaim diselesaikan
melanggar ketentuan; **Investigator** menangani klaim mencurigakan; **Analyst Doctor** menilai
klaim yang menuntut pertimbangan medis — dan yang terakhir menyentuh **data medis**, yang aksesnya
dibatasi `FR-R2`.

#### Ruang lingkup

- Tiga inbox terpisah: Compliance dan Investigator memakai **Workbasket**; Analyst Doctor dan
  RCL Dokter memakai **Worklist** (`D-26`).
- Pencatatan hasil pemeriksaan masing-masing jalur.
- **Lompatan lateral** masuk ke ketiga jalur — pengganti Ticket rule `CompliancePNC`,
  `SendToInvestigator`, dan `RCLDokter`.
- Penegakan `FR-R2`: **data medis hanya dapat diakses Analyst Doctor dan RCL Dokter**.

#### Non-goal

- **Tidak** menangani penolakan umum — itu `TKT-B11-001`.
- **Tidak** membangun mekanisme penugasan — itu `B-6`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Ticket rule `CompliancePNC`, `SendToInvestigator`, `RCLDokter`** — dirujuk `Flow/Register_Flow.xml:3844`, `:3624`, `:2954`; **ketiganya tidak ada di export**, dan **tidak satu pun punya pemicu yang terverifikasi** | **Tim Pega** (`R-16`, `ADR-0021`) | **Kapan** klaim masuk ke ketiga jalur ini sama sekali tidak diketahui |
| **Router `RouterRCLDokter`** | **Tim Pega** (`R-04`) | Menentukan dokter mana yang menerima |
| **`CompliancePNC` dipakai serentak sebagai nama Ticket rule dan nama workbasket** — satu hal yang sama? | **Work Owner** | Bila berbeda, keduanya perlu dipisah di sistem baru |
| **Siapa yang boleh memicu transisi lateral?** | **Work Owner** | Sama dengan `TKT-B11-001` |

#### Acceptance criteria

- ☐ Ketiga inbox memakai model yang benar: Compliance dan Investigator **Workbasket**, Analyst
      Doctor dan RCL Dokter **Worklist** — diuji keempatnya.
- ☐ Hasil pemeriksaan masing-masing jalur tersimpan dan **tercatat di jejak audit**.
- ☐ **Data medis hanya terlihat** oleh peran Analyst Doctor dan RCL Dokter — diuji dengan peran
      lain: field **tidak muncul** dan endpoint mengembalikan `403` (`FR-R2`).
- ☐ Pembatasan data medis berlaku **juga di staging** — diuji di sana, karena staging memuat data
      produksi apa adanya (`ADR-0029`).
- ☐ Lompatan lateral ke ketiga jalur berperilaku sesuai keputusan Work Owner, dan **tercatat**.
- ☐ Gerbang 1: isi inbox dan hasil pemeriksaan **sama dengan Pega** untuk peran yang sama.
- ☐ Gerbang 2: UAT oleh **ketiga peran secara terpisah** — bukan satu orang untuk semuanya,
      karena justru pemisahan aksesnya yang diuji.

#### Dependency / Blocked by

`TKT-B06-001` · `TKT-B11-001` · `TKT-F3-005` (batas akses medis). **Terhalang Tim Pega dan dua
keputusan Work Owner.**

#### Constraint keamanan, data, operasional

- **`FR-R2` adalah pembatasan akses paling ketat di seluruh aplikasi.** Data medis hanya untuk dua
  peran, dan pembatasan itu ditegakkan **di kueri**, bukan dengan menyembunyikan field di layar.
- Hasil investigasi memuat dugaan terhadap nasabah — ia **tidak boleh** muncul di layar peran lain
  maupun di laporan umum.
- Karena `D-59` tidak mengenal pemisahan tugas, seseorang yang memiliki menu Investigator **dan**
  menu akseptasi dapat menyelidiki sekaligus membayar klaim yang sama. Jejak audit satu-satunya
  kontrol.

#### Migrasi skema / rollout / rollback

Menambah tabel hasil pemeriksaan per jalur. Backward-compatible.

**Rollback:** hasil pemeriksaan tetap ada; jalurnya kembali ditangani Pega.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/compliance/... -run TestInboxWorkbasket
go test ./internal/app/analystdoctor/... -run TestInboxWorklist
go test ./internal/adapter/http/... -run TestAksesDataMedisDibatasi
go run ./cmd/s8 banding --modul B-11 --peran 3
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Ticket rule `CompliancePNC`, `SendToInvestigator`, `RCLDokter` hilang | `Flow/Register_Flow.xml:3844`, `:3624`, `:2954` · `ADR-0021` |
| Tidak satu pun punya pemicu terverifikasi | `ADR-0021` — hanya 3 dari 17 punya pemicu |
| Pembagian Worklist versus Workbasket | `D-26` · `ADR-0019` |
| Akses data medis dibatasi dua peran | `FR-R2` · `BRD §17.2` |
| `CompliancePNC` dipakai sebagai Ticket rule dan workbasket | `docs/verifikasi-bukti-adr.md` §5 |

#### Comments

# 19. B-12 · Salvage, Lelang & Recovery

*Modul Bisnis Inti · folder `docs/ticketing/B-12-Salvage-Lelang-dan-Recovery/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Salvage** — harness `InboxSalvage`, `InboxRequestSalvage`, `MasterRecovery`; activity `SetStsSalvagePNC_act`, `Insert_salvageToGAByService`, `SetEmailKomiteSalvage`; procedure `INSERT_SALVAGE`, `INSERT_SALVAGE_DETAILS`, `ADD_NEWMASTERVIRTUALACCOUNT` |
| **Kode modul** | `B-12` |
| **Gelombang** | 5 — Nilai dan pihak luar |
| **Ukuran** | 26 activity |
| **Bergantung pada** | `B-5` Input Estimasi |
| **Kesiapan** | **SEBAGIAN** — **lepas dari `BRD §21.4`** (`D-55`) |

### Apa yang dikerjakan modul ini

Mencatat **barang sisa** dari kerugian yang masih bernilai jual, menjualnya (termasuk lewat balai
lelang), dan mencatat **pemulihan dana** dari pihak ketiga — sebagian lewat **Virtual Account**.

Salvage memengaruhi nilai klaim dua arah: ia **mengurangi nilai bersih** klaim, dan ia
**memulihkan kapasitas pertanggungan** pada perhitungan Sisa TSI (`TKT-B03-003`).

### Cacat yang ditutup modul ini

| Cacat | Bukti | Keputusan |
|---|---|---|
| `INSERT_SALVAGE` menulis **`IDSALVAGE = NULL` saat update** | `Database/INSERT_SALVAGE.prc:47` | **PERBAIKI** — butir 9 `P-5` (`D-49`) |
| `NilaiSalvage` hanya ditambahkan bila baris terakhir kebetulan bertipe salvage | `ValidasiSisaTSI:817` versus `:1489` | **PERBAIKI** — butir 8 `P-5` |

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | `SET_ATTACHFILETEMPSALVAGE` — salah satu dari dua objek yang belum diterima dari 64 | **DBA** |
| **Artefak** | **Hitungan baris rusak**: `SELECT COUNT(*) … WHERE IDSALVAGE IS NULL` | **DBA** |
| **Keputusan** | `tterjual='2'` → `STSTRANSFER='5'`: **satu item terjual mengubah status seluruh salvage induk** — benar? | **Work Owner** |
| **Keputusan** | `SISAKLAIM` **tidak dihitung**, diterima apa adanya — benar? | **Work Owner** |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B12-001](issues/01-pencatatan-salvage-dan-item-lelang.md) | Pencatatan salvage dan item lelang | `needs-info` |
| [TKT-B12-002](issues/02-recovery-dan-virtual-account.md) | Recovery dan Virtual Account | `needs-info` |

## Daftar Tiket

### TKT-B12-001 — Pencatatan salvage dan item lelang

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak dan keputusan** |
| **Modul** | **B-12 Salvage** · Gelombang: 5 · Bergantung pada: TKT-B05-001, TKT-U2-003 |
| **Requirement** | FR-B12 |
| **Keputusan** | D-49 butir 8 dan 9 |
| **ADR** | 0012, 0017 |
| **Risiko** | R-19 |
| **Rule Pega yang digantikan** | `Database/INSERT_SALVAGE.prc:47` — **menulis `IDSALVAGE = NULL` saat update** · `INSERT_SALVAGE_DETAILS` · `SET_ATTACHFILETEMPSALVAGE` (**belum diterima**) · harness `InboxSalvage`, `InboxRequestSalvage` · `Activity/SetStsSalvagePNC_act-Act.xml` |
| **Peran penguji gerbang 2** | **PncPICTeknik** dan **PncCollection** |
| **Label** | `modul::B-12` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::5` |
| **Milestone** | Gelombang 5 — Nilai dan pihak luar |
| **Berkas sumber** | `ticketing/B-12-Salvage-Lelang-dan-Recovery/issues/01-pencatatan-salvage-dan-item-lelang.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Barang sisa dicatat, dijual, dan hasilnya masuk kembali ke perhitungan klaim — **tanpa kehilangan
identitas barisnya**.

Nilai bisnisnya berupa cacat yang sudah berjalan: `INSERT_SALVAGE` **menulis `IDSALVAGE = NULL`
saat melakukan update**. Baris yang kehilangan identitasnya **tidak dapat lagi ditautkan** ke item
maupun ke klaimnya — dan berapa banyak baris yang sudah rusak **belum dihitung**.

#### Ruang lingkup

- Pencatatan salvage induk dan **item-item** di bawahnya.
- Status penjualan per item, dan status transfer pada salvage induk.
- **Perbaikan butir 9 `P-5`**: identitas baris (`IDSALVAGE`) **tidak pernah dikosongkan** saat
  update.
- Unggah lampiran salvage memakai komponen `TKT-U2-003`.

#### Non-goal

- **Tidak** menghitung sisa TSI — itu `TKT-B03-003`; modul ini menyediakan nilai salvage-nya.
- **Tidak** menangani Virtual Account — itu `TKT-B12-002`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **`SET_ATTACHFILETEMPSALVAGE`** — salah satu dari dua objek yang belum diterima dari 64 | **DBA** (`D-55`) | Aturan lampiran salvage tidak diketahui |
| **Hitungan baris rusak**: `SELECT COUNT(*) … WHERE IDSALVAGE IS NULL` | **DBA** | Menentukan besarnya data yang perlu diperbaiki sebelum migrasi — dan apakah perbaikan itu mungkin |
| **`tterjual='2'` → `STSTRANSFER='5'`: satu item terjual mengubah status seluruh salvage induk** — benar? | **Work Owner** | Bila salah, status induk berubah terlalu dini dan memengaruhi perhitungan nilai |
| **`SISAKLAIM` tidak dihitung, diterima apa adanya** — benar? | **Work Owner** | Nilai yang diterima tanpa dihitung berarti tidak ada yang memvalidasinya |

#### Acceptance criteria

- ☐ Update pada salvage **tidak pernah mengosongkan `IDSALVAGE`** — diuji dengan 10 update
      berturut-turut; identitas baris tetap. **Inilah bukti butir 9 `P-5` tertutup.**
- ☐ Satu salvage induk dapat memuat banyak item, dan menjual satu item **tidak menghapus** yang
      lain — diuji.
- ☐ Perubahan status penjualan item memengaruhi status induk **sesuai keputusan Work Owner** —
      diuji sesuai jawabannya.
- ☐ Nilai salvage yang tercatat **selalu** ikut pada perhitungan Sisa TSI (`TKT-B03-003`),
      apa pun urutan barisnya — bukti butir 8 `P-5`.
- ☐ Salvage yang dihapus adalah **soft delete** (`ADR-0012`) — diuji.
- ☐ Lampiran salvage dapat diunggah dan dibuka kembali.
- ☐ Gerbang 1: data salvage **sama dengan Pega** pada 20 klaim contoh, **kecuali** kasus
      `IDSALVAGE` yang terpetakan ke butir 9 `P-5`.
- ☐ Gerbang 2: UAT **PncPICTeknik**.

#### Dependency / Blocked by

`TKT-B05-001` · `TKT-U2-003` · `TKT-S1-001`. **Terhalang DBA dan dua keputusan Work Owner.**

#### Constraint keamanan, data, operasional

- **Data historis mungkin sudah rusak.** Baris ber-`IDSALVAGE = NULL` tidak dapat ditautkan; bila
  jumlahnya banyak, perbaikan data menjadi pekerjaan tersendiri dengan persetujuan Work Owner dan
  DBA (`ADR-0017`).
- Salvage memengaruhi **nilai yang dibayarkan**; perubahannya wajib tercatat di jejak audit.

#### Migrasi skema / rollout / rollback

Menambah tabel salvage dan item. Backward-compatible.

**Rollback:** mengembalikan perilaku lama berarti **memulihkan cacat `IDSALVAGE = NULL`** — hanya
masuk akal bila ada temuan yang lebih buruk.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/salvage/... -run TestIDSalvageTidakPernahKosong
go test ./internal/app/salvage/... -run TestItemDanInduk
go test ./internal/domain/tsi/... -run TestSalvageSelaluDihitung
go run ./cmd/s8 banding --modul B-12 --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `IDSALVAGE = NULL` saat update | `Database/INSERT_SALVAGE.prc:47` · `D-49` butir 9 |
| `NilaiSalvage` bergantung urutan baris | `ValidasiSisaTSI:817` versus `:1489` · `D-49` butir 8 |
| `SET_ATTACHFILETEMPSALVAGE` belum diterima | `D-55` |
| `tterjual='2'` → `STSTRANSFER='5'` | `docs/verifikasi-bukti-adr.md` §15 baris `B-12` |
| Salvage memulihkan kapasitas pertanggungan | `CONTEXT.md` — **Sisa TSI** |

#### Comments

### TKT-B12-002 — Recovery dan Virtual Account

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak dan `D-40`** |
| **Modul** | **B-12 Salvage** · Gelombang: 5 · Bergantung pada: TKT-B12-001, TKT-S4-001 |
| **Requirement** | FR-B12 |
| **Keputusan** | D-02, D-68, D-40 |
| **ADR** | 0007, 0008, 0025 |
| **Risiko** | R-17, R-18 |
| **Rule Pega yang digantikan** | `Database/ADD_NEWMASTERVIRTUALACCOUNT.prc:18` — **`ErrMsg` membawa nomor virtual account sekaligus pesan galat** · harness `MasterRecovery` · `Activity/Insert_salvageToGAByService-Act.xml` · Service REST `RecivedDataandAttachmentLelangASMSimasbid` |
| **Peran penguji gerbang 2** | **PncCollection** dan **PncManagerAdmin** |
| **Label** | `modul::B-12` `tipe::integrasi` `status::needs-info` `prioritas::sedang` `gelombang::5` |
| **Milestone** | Gelombang 5 — Nilai dan pihak luar |
| **Berkas sumber** | `ticketing/B-12-Salvage-Lelang-dan-Recovery/issues/02-recovery-dan-virtual-account.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Dana yang kembali dari pihak ketiga — hasil lelang, pemulihan dari pihak yang bertanggung jawab —
tercatat dan tertaut ke klaimnya, sebagian lewat **Virtual Account**.

Nilai bisnisnya: recovery **mengurangi kerugian bersih** perusahaan. Dana yang masuk tanpa tertaut
klaim menjadi selisih yang harus dicari manual saat rekonsiliasi.

#### Ruang lingkup

- Pencatatan recovery per klaim, termasuk sumbernya.
- Penerbitan dan penautan **Virtual Account** untuk penerimaan dana.
- Penerimaan data dan lampiran hasil lelang dari sistem luar (Service REST
  `RecivedDataandAttachmentLelangASMSimasbid`).
- **Kontrak galat yang bersih**: nomor Virtual Account dan pesan galat **tidak lagi berbagi satu
  kolom**.

#### Non-goal

- **Tidak** membangun sistem lelang.
- **Tidak** memanggil `ADD_NEWMASTERVIRTUALACCOUNT` (`ADR-0007`).

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Kontrak layanan Virtual Account** dan kredensialnya | **Tim Infra/Security** (`D-40`) + tim pemilik sistem | Penerbitan VA menyentuh rekening; kredensialnya belum punya tempat |
| **4 Service REST masuk — masuk lingkup?** Salah satunya `RecivedDataandAttachmentLelangASMSimasbid` | **Work Owner** (`ADR-0008`) | Permukaan masuk ini belum pernah masuk hitungan `FR-S4` |
| Otentikasi layanan masuk: **sembilan Connect REST ber-`pyUseAuthentication=false`** | **Work Owner + tim integrasi** | Menentukan siapa boleh memanggil layanan penerimaan lelang |

#### Acceptance criteria

- ☐ Recovery dapat dicatat dan **tertaut ke klaim**; recovery tanpa klaim **ditolak** — diuji.
- ☐ Penerbitan Virtual Account mengembalikan **nomor VA** dan **galat** sebagai dua hal terpisah
      — diuji; **kontrak lama yang menggabungkan keduanya tidak boleh punya padanan**.
- ☐ Penerbitan VA yang gagal **tidak meninggalkan recovery setengah jadi** — diuji.
- ☐ Layanan penerimaan data lelang **memeriksa kewenangan pemanggil** — diuji dengan permintaan
      tanpa kredensial: **ditolak**.
- ☐ Penerimaan data lelang bersifat **idempoten**: kiriman ganda tidak menghasilkan recovery
      kedua — diuji.
- ☐ **Nol pemanggilan stored procedure** dan **nol kredensial di kode** — diuji pemindaian.
- ☐ Gerbang 2: UAT **PncCollection** dengan satu penerimaan dana uji.

#### Dependency / Blocked by

`TKT-B12-001` · `TKT-S4-001` · `TKT-F1-002` (konfigurasi rahasia). **Terhalang Tim Infra/Security
dan Work Owner.**

#### Constraint keamanan, data, operasional

- Virtual Account menyentuh **rekening dan dana**; kredensialnya tunduk `D-40` yang masih `OPEN`.
- **Layanan masuk adalah permukaan serang.** Sembilan Connect REST ber-`pyUseAuthentication=false`
  dan satu memakai `http://` ke IP:port **tanpa TLS** untuk data premi (`R-18`) — pola itu tidak
  boleh dibawa.
- Nomor rekening dan VA adalah data nasabah; keduanya **tidak boleh masuk log**.

#### Migrasi skema / rollout / rollback

Menambah tabel recovery dan Virtual Account. Backward-compatible.

**Rollback:** VA yang telanjur terbit **tetap ada di sistem bank** — pembatalannya menempuh
prosedur bank, bukan aplikasi.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/recovery/... -run TestRecoveryWajibTertautKlaim
go test ./internal/adapter/va/... -run TestNomorVADanGalatTerpisah
go test ./internal/adapter/http/... -run TestLayananMasukMemeriksaKewenangan
grep -rIn "CALL \|ADD_NEWMASTERVIRTUALACCOUNT" internal/       # HARUS 0 baris
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `ErrMsg` membawa nomor VA sekaligus pesan galat | `Database/ADD_NEWMASTERVIRTUALACCOUNT.prc:18` · `ADR-0007` |
| 4 Service REST masuk, termasuk penerimaan lelang | `D-57` · `ADR-0008` §8.5 |
| 9 Connect REST tanpa autentikasi; satu `http://` tanpa TLS | `docs/verifikasi-bukti-adr.md` §15 baris `S-4` · `R-18` |
| Recovery lewat Virtual Account | `CONTEXT.md` — **Recovery** |

#### Comments

# 20. B-13 · Input Open Protection — Buka Proteksi

*Modul Bisnis Inti · folder `docs/ticketing/B-13-Input-Open-Protection/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Input Protection** / **Request Protection** — `Flow/CreateProtection_Flow.xml`, harness `InputProtection_Harness`, `InputReqProtection_Harness`, `MasterProteksiVisibilityData` |
| **Kode modul** | `B-13` |
| **Gelombang** | 4 — Persetujuan |
| **Ukuran** | 12 activity |
| **Bergantung pada** | `B-2` Input Register |
| **Kesiapan** | **SEBAGIAN** — **kendala paling sedikit di antara modul bisnis** |

### Apa yang dikerjakan modul ini

Mencatat permintaan **pembukaan proteksi** sebelum atau di luar alur klaim normal, lalu menautkannya
ke klaim dan menandainya terpakai.

### Kenapa modul ini paling siap

Berbeda dari modul bisnis lain, **artefaknya lengkap**:

| Artefak | Status |
|---|---|
| When rule `IsReqProtection` | ✅ **ada** |
| When rule `IsOpenProtectionPNC` | ✅ **ada** |
| Ticket rule `TC_PNCInputProtection` | ✅ **ada** |
| Ticket rule `Akp_PNCInputProtection` | ✅ **ada** |

Keempatnya ada di export — **satu-satunya modul bisnis dengan Ticket rule yang lengkap.**

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | `CreateProtection_Flow` hanya memuat **6 shape** — cakupan sekecil itu benar, atau ada bagian alur yang berada di luar flow? | **Work Owner** |

Tidak ada penghalang artefak.

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B13-001](issues/01-layar-permintaan-buka-proteksi.md) | Layar permintaan buka proteksi | `ready-for-human` |
| [TKT-B13-002](issues/02-penautan-proteksi-ke-klaim.md) | Penautan proteksi ke klaim dan penandaan terpakai | `needs-info` |

## Daftar Tiket

### TKT-B13-001 — Layar permintaan buka proteksi

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | **B-13 Input Open Protection** · Gelombang: 4 · Bergantung pada: TKT-U2-002, TKT-F3-005 |
| **Requirement** | FR-B13 |
| **Keputusan** | D-13 |
| **ADR** | 0002, 0026 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | `Flow/CreateProtection_Flow.xml` · harness `InputProtection_Harness`, `InputReqProtection_Harness` · When rule `IsReqProtection`, `IsOpenProtectionPNC` (**keduanya ada di export**) |
| **Peran penguji gerbang 2** | **PncAdmin** dan **PncOPCGeneral** |
| **Label** | `modul::B-13` `tipe::migrasi` `status::ready-for-human` `prioritas::sedang` `gelombang::4` |
| **Milestone** | Gelombang 4 — Persetujuan |
| **Berkas sumber** | `ticketing/B-13-Input-Open-Protection/issues/01-layar-permintaan-buka-proteksi.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Petugas dapat mengajukan **pembukaan proteksi** dan mencatat persetujuannya, sebelum atau di luar
alur klaim normal.

Nilai bisnisnya: Open Protection dipakai ketika perlindungan perlu dibuka mendahului proses klaim
biasa. Tanpa layar ini, permintaan itu ditangani di luar sistem dan **tidak terlacak**.

#### Ruang lingkup

- Layar permintaan: nomor proteksi, nomor polis, jenis proteksi, tanggal input, pemohon.
- Status akseptasi permintaan, dengan dua jalur yang sudah ada Ticket rule-nya:
  `TC_PNCInputProtection` dan `Akp_PNCInputProtection`.
- Daftar proteksi yang **belum terpakai**, memakai komponen tabel baku.
- Jejak audit pada pengajuan dan perubahan status akseptasi.

#### Non-goal

- **Tidak** menautkan ke klaim — itu `TKT-B13-002`.
- **Tidak** mengubah data polis (`ADR-0006`).

#### Acceptance criteria

- ☐ Permintaan proteksi dapat disimpan dengan kelima field, dan **nomor proteksi unik** — diuji
      20 penyimpanan, nol nomor ganda.
- ☐ Status akseptasi dapat diubah, dan setiap perubahan menghasilkan **tepat satu baris jejak
      audit** berisi nilai sebelum dan sesudah.
- ☐ Daftar menampilkan **hanya proteksi yang belum terpakai** secara baku; yang sudah terpakai
      dapat ditampilkan dengan penyaring eksplisit — diuji keduanya.
- ☐ Batas data cabang dan lini bisnis ditegakkan di kueri (`TKT-F3-005`) — diuji.
- ☐ Susunan layar mengikuti `InputProtection_Harness` — perbandingan berdampingan disetujui
      penguji gerbang 2.
- ☐ Gerbang 1: hasil penyimpanan **sama dengan Pega** pada 20 permintaan contoh.
- ☐ Gerbang 2: UAT **PncAdmin** dan **PncOPCGeneral**.

#### Dependency / Blocked by

`TKT-U2-002` · `TKT-F3-005` · `TKT-S5-002`.

#### Constraint keamanan, data, operasional

- Layar memuat nomor polis dan data tertanggung — batas data berlaku.
- Proteksi yang sudah terpakai **tidak dihapus**, hanya ditandai (`ADR-0012`) — sehingga riwayat
  pemakaiannya tetap dapat ditelusuri.

#### Migrasi skema / rollout / rollback

Menambah tabel Open Protection. Backward-compatible.

**Rollback:** permintaan yang tersimpan tetap ada; pengajuan baru kembali dilakukan di Pega.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/proteksi/... -run TestSimpanPermintaan
go test ./internal/app/proteksi/... -run TestDaftarHanyaBelumTerpakai
go run ./cmd/s8 banding --modul B-13 --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Flow dan harness Open Protection | `Flow/CreateProtection_Flow.xml` · `Harness/InputProtection_Harness-Harness.xml` |
| `IsReqProtection` dan `IsOpenProtectionPNC` ada di export | `docs/verifikasi-bukti-adr.md` §15 baris `B-13` |
| Ticket rule `TC_PNCInputProtection`, `Akp_PNCInputProtection` ada | `ADR-0021` |
| Aggregate Open Protection | `docs/Steering/05-DOMAIN-MODEL.md` §4 |

#### Comments

### TKT-B13-002 — Penautan proteksi ke klaim dan penandaan terpakai

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | terhalang keputusan (cakupan alur) |
| **Modul** | **B-13 Input Open Protection** · Gelombang: 4 · Bergantung pada: TKT-B13-001, TKT-B02-001 |
| **Requirement** | FR-B13 |
| **Keputusan** | D-25 |
| **ADR** | 0008, 0021, 0026 |
| **Risiko** | R-03 |
| **Rule Pega yang digantikan** | penautan proteksi pada `Flow/CreateProtection_Flow.xml` · Ticket rule `TC_PNCInputProtection` dan `Akp_PNCInputProtection` · objek remote `GENERAL.MST_BUKA_PROTEKSI` lewat **tiga DB Link** (`@ASMD`, `@SIMASNET`, `@SMI`) |
| **Peran penguji gerbang 2** | **PncAdmin** dan **PncOPCGeneral** |
| **Label** | `modul::B-13` `tipe::migrasi` `status::needs-info` `prioritas::sedang` `gelombang::4` |
| **Milestone** | Gelombang 4 — Persetujuan |
| **Berkas sumber** | `ticketing/B-13-Input-Open-Protection/issues/02-penautan-proteksi-ke-klaim.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu proteksi dapat ditautkan ke klaim dan **ditandai terpakai**, sehingga tidak dapat dipakai dua
kali.

Nilai bisnisnya: proteksi yang terpakai dua kali berarti perlindungan yang sama dihitung ganda.
Penandaan inilah yang mencegahnya.

#### Ruang lingkup

- Penautan satu proteksi ke satu klaim, dan penandaan **`SudahDipakaiKlaim`**.
- Penolakan bila proteksi sudah terpakai, dengan pesan yang menyebut **nomor klaim pemakainya**.
- Sinkronisasi status ke `GENERAL.MST_BUKA_PROTEKSI` — **satu-satunya objek remote yang ditulis**
  di seluruh sistem lama, dan ia dipanggil ke **tiga sistem berbeda**.
- Jejak audit pada penautan dan pelepasan.

#### Non-goal

- **Tidak** membangun API pengganti DB Link — itu `S-4`.
- **Tidak** mengubah alur persetujuan proteksi (`TKT-B13-001`).

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **`CreateProtection_Flow` hanya 6 shape — cakupan sekecil itu benar?** | **Work Owner** | Bila ada bagian alur di luar flow, ia tidak akan ikut terbawa dan baru ketahuan setelah rilis |
| **API pengganti `GENERAL.MST_BUKA_PROTEKSI`** | **tim pemilik sistem** (`R-03`) | Ini **objek remote yang menulis**, ke tiga sistem. Penggantinya **wajib API idempoten** — tidak dapat dijembatani salinan berkala |

#### Acceptance criteria

- ☐ Proteksi yang belum terpakai dapat ditautkan ke klaim, dan **ditandai terpakai** — diuji.
- ☐ Menautkan proteksi yang **sudah terpakai** ditolak, dengan pesan yang menyebut nomor klaim
      pemakainya — diuji.
- ☐ Melepas tautan mengembalikan proteksi menjadi tersedia, dan **tercatat di jejak audit** —
      diuji.
- ☐ Penulisan status ke sistem luar bersifat **idempoten**: mengirim dua kali tidak menghasilkan
      efek ganda — diuji dengan pengiriman berulang.
- ☐ Kegagalan penulisan ke sistem luar **tidak membatalkan penautan lokal**, tetapi masuk antrean
      untuk dicoba ulang — dan kegagalan itu **terlihat**, bukan diam.
- ☐ Pemanggilan sistem luar **tidak berada di dalam transaksi database** (`TKT-F2-003`).
- ☐ Gerbang 1: status penautan **sama dengan Pega** pada 20 kasus contoh.
- ☐ Gerbang 2: UAT **PncOPCGeneral**.

#### Dependency / Blocked by

`TKT-B13-001` · `TKT-B02-001` · `TKT-S4-001` (API pengganti DB Link). **Terhalang Work Owner dan
tim pemilik sistem.**

#### Constraint keamanan, data, operasional

- `GENERAL.MST_BUKA_PROTEKSI` adalah **satu-satunya objek remote yang ditulis** sistem lama, dan ia
  dipanggil ke **tiga sistem berbeda** (`@ASMD`, `@SIMASNET`, `@SMI`). Penggantinya **wajib API
  idempoten** — ini disebut eksplisit di `R-03` sebagai titik yang tidak boleh dijembatani salinan
  berkala.
- Kegagalan penulisan lintas sistem **tidak boleh** membuat proteksi tertaut di satu sisi dan tidak
  di sisi lain tanpa ada yang tahu.

#### Migrasi skema / rollout / rollback

Menambah kolom penanda terpakai dan rujukan klaim. Backward-compatible.

**Rollback:** penandaan yang sudah terkirim ke sistem luar **tidak otomatis dibatalkan** — itu
menuntut pembatalan manual yang disepakati dengan tim pemilik sistem.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/proteksi/... -run TestPenautanKeKlaim
go test ./internal/app/proteksi/... -run TestTolakProteksiTerpakai
go test ./internal/adapter/eksternal/... -run TestTulisProteksiIdempoten
grep -rIn "http\." internal/app/proteksi/ | grep -i "tx\|transaksi"   # HARUS 0 baris
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `GENERAL.MST_BUKA_PROTEKSI` satu-satunya objek remote yang menulis, ke tiga sistem | `ADR-0008` · `docs/Steering/16-RISK-ANALYSIS.md` `R-03` |
| Proteksi ditandai terpakai saat ditautkan | `docs/Steering/05-DOMAIN-MODEL.md` §4 |
| Pemanggilan sistem luar tidak di dalam transaksi | `docs/Steering/10-API-STRATEGY.md` §8.2 |
| `CreateProtection_Flow` 6 shape | `docs/verifikasi-bukti-adr.md` §15 baris `B-13` |

#### Comments

# 21. B-14 · Input Receive Document — Penerimaan Dokumen Fisik

*Modul Bisnis Inti · folder `docs/ticketing/B-14-Input-Receive-Document/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Input Receive Document** — `Flow/InputReceiveDocument.xml`, layar `ReceiveDoucument_Harness`, `InboxRCVApp_Harness`, `ViewReceiveDocument` |
| **Kode modul** | `B-14` |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | 8 activity |
| **Bergantung pada** | `F-3` Login & Hak Akses |
| **Kesiapan** | **SEBAGIAN** — kendala paling sedikit di antara modul bisnis |

### Apa yang dikerjakan modul ini

Mencatat **kedatangan dokumen fisik klaim** dari cabang: siapa mengirim, lewat ekspedisi apa,
nomor resi, berapa lembar, jenis dokumen, kapan dikirim, perkiraan tiba, dan kapan benar-benar
diterima.

Fungsinya sederhana tetapi menentukan: **tanggal terima dokumen** yang dicatat di sini menjadi
salah satu dari tiga tanggal yang divalidasi di Input Register (`TKT-B02-002`), dan menjadi titik
awal perhitungan TAT.

### Rule Pega yang digantikan

| Jenis | Nama |
|---|---|
| Flow | `Flow/InputReceiveDocument.xml` |
| Harness | `ReceiveDoucument_Harness` · `InboxRCVApp_Harness` · `ViewReceiveDocument` · `ViewTempDetailReqDocument` |
| Ticket rule | `SendToReceiveDocument` (`Flow/InputReceiveDocument.xml:690`) — **hilang dari export** |
| Router | `PNCAdminRouterRCV` |

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | Ticket rule `SendToReceiveDocument` — **dirujuk tetapi tidak ada di export** | **Tim Pega** (`R-16`) |
| **ADR `Proposed`** | `ADR-0021` — kapan lompatan lateral ke tahap Receive Document dipicu belum diketahui | Tim Pega → Work Owner |

Tidak ada penghalang keputusan bisnis. **Ini modul bisnis paling siap dikerjakan.**

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B14-001](issues/01-layar-input-receive-document.md) | Layar Input Receive Document | `ready-for-human` |
| [TKT-B14-002](issues/02-inbox-penerimaan-dan-penugasan.md) | Inbox penerimaan dokumen dan penugasannya | `needs-info` |
| [TKT-B14-003](issues/03-kaitan-dokumen-ke-klaim.md) | Kaitan dokumen ke klaim dan tanggal terima | `ready-for-human` |

## Daftar Tiket

### TKT-B14-001 — Layar Input Receive Document

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | **B-14 Input Receive Document** · Gelombang: 3 · Bergantung pada: TKT-U2-002, TKT-F3-005 |
| **Requirement** | FR-B14 |
| **Keputusan** | D-13 |
| **ADR** | 0002, 0026 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | `Flow/InputReceiveDocument.xml` · harness `ReceiveDoucument_Harness` dan `ViewReceiveDocument` |
| **Peran penguji gerbang 2** | **PncReceive** dan **PncManagerReceive** |
| **Label** | `modul::B-14` `tipe::migrasi` `status::ready-for-human` `prioritas::sedang` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-14-Input-Receive-Document/issues/01-layar-input-receive-document.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Petugas penerimaan dapat mencatat satu kiriman dokumen fisik dari cabang, dan mencatat kapan
dokumen itu **benar-benar diterima**.

Nilai bisnisnya melampaui pencatatan administratif: **tanggal terima dokumen** yang diisi di sini
dipakai sebagai batas pada validasi tanggal di Input Register, dan menjadi titik awal perhitungan
TAT yang dilaporkan ke manajemen.

#### Ruang lingkup

- Layar pencatatan kiriman: **nomor register dokumen (`RCV_ID`)**, cabang pengirim, ekspedisi,
  nomor resi, tanggal kirim, estimasi tiba, jumlah lembar, dan jenis dokumen.
- Pencatatan **tanggal terima** saat dokumen benar-benar sampai.
- Layar lihat detail kiriman (setara `ViewReceiveDocument`).
- Jejak audit pada pencatatan dan perubahan tanggal terima (`S-5`).

#### Non-goal

- **Tidak** mengunggah berkas dokumen — itu `S-1`. Modul ini mencatat **kiriman fisik**, bukan
  berkas digital.
- **Tidak** menautkan ke klaim — itu `TKT-B14-003`.
- **Tidak** membangun inbox dan penugasannya — itu `TKT-B14-002`.

#### Acceptance criteria

- ☐ Kiriman baru dapat disimpan dengan kesembilan field, dan **nomor register dokumen terbit
      unik** — diuji 20 penyimpanan berturut-turut, nol nomor ganda.
- ☐ Tanggal terima **tidak boleh lebih awal** dari tanggal kirim — diuji kedua batas.
- ☐ Tanggal terima **tidak boleh di masa depan** — diuji.
- ☐ Estimasi tiba boleh kosong; tanggal terima boleh kosong saat kiriman baru dicatat — diuji.
- ☐ Mengubah tanggal terima menghasilkan **tepat satu baris jejak audit** berisi nilai sebelum
      dan sesudah.
- ☐ Susunan field mengikuti `ReceiveDoucument_Harness` — dibuktikan perbandingan berdampingan
      yang disetujui penguji gerbang 2.
- ☐ Petugas tanpa izin menu Receive Document menerima `403` dan **tidak melihat menunya**.
- ☐ Gerbang 1: hasil penyimpanan **sama dengan Pega** pada 20 kiriman contoh di staging.
- ☐ Gerbang 2: UAT **PncReceive** pada 10 kiriman nyata.

#### Dependency / Blocked by

`TKT-U2-002` (form baku) · `TKT-F3-005` (izin menu) · `TKT-S5-002` (jejak audit).

#### Constraint keamanan, data, operasional

- Layar memuat **nomor polis dan nama tertanggung** pada kiriman yang sudah tertaut klaim — batas
  data cabang berlaku (`TKT-F3-005`).
- Tanggal terima **memengaruhi validasi klaim dan angka TAT**; mengubahnya setelah klaim berjalan
  adalah tindakan bernilai tinggi dan karena itu diaudit.
- Waktu disimpan UTC dan ditampilkan WIB lewat satu konversi (`F-5`).

#### Migrasi skema / rollout / rollback

Menambah tabel penerimaan dokumen milik aplikasi. Backward-compatible; Pega tidak membacanya.

**Rollback:** kiriman yang telanjur dicatat di sistem baru tetap ada. Karena Pega masih menangani
alurnya selama masa paralel, rollback berarti **berhenti mencatat kiriman baru di sistem baru** —
data lama tidak hilang.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/receivedoc/... -run TestSimpanKiriman
go test ./internal/app/receivedoc/... -run TestValidasiTanggalTerima
go run ./cmd/s8 banding --modul B-14 --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Flow dan harness penerimaan dokumen | `Flow/InputReceiveDocument.xml` · `Harness/ReceiveDoucument_Harness-Harness.xml` |
| Isi data: ekspedisi, resi, estimasi tiba | `CONTEXT.md` — **Receive Document** |
| Tanggal terima dipakai validasi registrasi | Invarian `I-2`, `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Tata letak mengikuti Pega | `D-13` |

#### Comments

### TKT-B14-002 — Inbox penerimaan dokumen dan penugasannya

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** — Ticket rule dan router |
| **Modul** | **B-14 Input Receive Document** · Gelombang: 3 · Bergantung pada: TKT-B14-001, TKT-B06-001 |
| **Requirement** | FR-B14, FR-W2 |
| **Keputusan** | D-26 |
| **ADR** | 0019, 0021 |
| **Risiko** | R-16 |
| **Rule Pega yang digantikan** | harness `InboxRCVApp_Harness` · router `PNCAdminRouterRCV` · Ticket rule `SendToReceiveDocument` (`Flow/InputReceiveDocument.xml:690`) |
| **Peran penguji gerbang 2** | **PncReceive** dan **PncManagerReceive** |
| **Label** | `modul::B-14` `tipe::migrasi` `status::needs-info` `prioritas::sedang` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-14-Input-Receive-Document/issues/02-inbox-penerimaan-dan-penugasan.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Kiriman dokumen yang belum ditangani muncul di inbox petugas penerimaan, dan dapat **dilompati ke
sana dari tahap lain** ketika sebuah klaim ternyata masih menunggu dokumen.

Nilai bisnisnya: tanpa inbox, kiriman yang belum ditindaklanjuti **tidak terlihat siapa pun**
sampai ada yang mencarinya. Dan lompatan lateral itulah yang membuat klaim tidak perlu dibatalkan
hanya karena dokumennya belum lengkap.

#### Ruang lingkup

- Inbox kiriman dokumen: daftar kiriman yang belum diterima atau belum tertaut klaim, memakai
  komponen tabel baku (`TKT-U2-001`).
- Penugasan kiriman ke petugas mengikuti model `Worklist`/`Workbasket` (`ADR-0019`).
- **Lompatan lateral ke tahap Receive Document** — pengganti Ticket rule `SendToReceiveDocument`.

#### Non-goal

- **Tidak** membangun mekanisme penugasan umum — itu `B-6`; modul ini memakainya.
- **Tidak** menentukan siapa yang boleh memicu lompatan lateral — itu pertanyaan lintas modul di
  `ADR-0021` dan `B-11`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Ticket rule `SendToReceiveDocument`** — dirujuk `Flow/InputReceiveDocument.xml:690` tetapi **tidak ada di export** | **Tim Pega** (`R-16`) | Tanpa rule-nya, **kapan** lompatan ini dipicu tidak diketahui |
| **Router `PNCAdminRouterRCV`** — ada di export, tetapi perlu konfirmasi bahwa ia memakai jalur `counter_quota` seperti router lain | **Work Owner + Tim Pega** | Menentukan ke siapa kiriman ditugaskan |
| **Siapa yang berwenang memicu lompatan lateral?** Sistem lama **nol pagar izin** | **Work Owner** | `D-59` menetapkan izin bersatuan menu; lompatan lateral melewati alur normal, dan kewenangannya belum dirumuskan |

#### Acceptance criteria

> Butir 3 dan 4 berubah bentuk tergantung jawaban di atas.

- ☐ Kiriman yang belum diterima muncul di inbox; yang sudah diterima **tidak** — diuji.
- ☐ Inbox menghormati batas data cabang: petugas cabang A tidak melihat kiriman cabang B, **dan
      jumlah totalnya juga tidak memuatnya** — diuji.
- ☐ Penugasan mengikuti model `Worklist`/`Workbasket` sesuai `ADR-0019` — diuji kedua model.
- ☐ Lompatan lateral memindahkan klaim ke tahap Receive Document **dan tercatat di jejak audit**
      dengan pelaku serta alasannya.
- ☐ Gerbang 1: isi inbox **sama dengan Pega** untuk peran yang sama pada data staging.
- ☐ Gerbang 2: UAT **PncReceive**.

#### Dependency / Blocked by

`TKT-B14-001` · `TKT-B06-001` (mekanisme penugasan) · `TKT-U2-001` (tabel baku).
**Terhalang Tim Pega dan satu keputusan Work Owner.**

#### Constraint keamanan, data, operasional

- Lompatan lateral **melewati urutan alur normal**. Karena `D-59` tidak mengenal pemisahan tugas,
  satu-satunya kontrol yang tersisa adalah **jejak audit** — karena itu pencatatannya wajib, bukan
  opsional.
- Inbox menampilkan data nasabah; batas data ditegakkan di kueri (`TKT-F3-005`).

#### Migrasi skema / rollout / rollback

Memakai tabel penugasan `B-6`; tidak menambah tabel sendiri.

**Rollback:** inbox disembunyikan dari menu. Kiriman tetap tercatat dan tetap dapat dicari lewat
layar `TKT-B14-001`.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/receivedoc/... -run TestInboxPenerimaan
go test ./internal/app/receivedoc/... -run TestBatasDataCabang
go test ./internal/app/receivedoc/... -run TestLompatanLateralTercatat
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Ticket rule `SendToReceiveDocument` dirujuk tanpa rule | `Flow/InputReceiveDocument.xml:690` · `ADR-0021` |
| Router `PNCAdminRouterRCV` | `D-26` · `ADR-0019` |
| Worklist versus Workbasket | `D-26` · `CONTEXT.md` |
| Nol pagar izin pada transisi lateral | `docs/verifikasi-bukti-adr.md` §15 baris `B-11` |

#### Comments

### TKT-B14-003 — Kaitan dokumen ke klaim dan tanggal terima

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | **B-14 Input Receive Document** · Gelombang: 3 · Bergantung pada: TKT-B14-001, TKT-B02-001 |
| **Requirement** | FR-B14 |
| **Keputusan** | D-50 |
| **ADR** | 0020, 0026 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | penautan `RCV_ID` ke klaim pada `Flow/InputReceiveDocument.xml` dan `Activity/InputRegister_act-Act.xml` |
| **Peran penguji gerbang 2** | **PncReceive** dan **PncAdmin** |
| **Label** | `modul::B-14` `tipe::migrasi` `status::ready-for-human` `prioritas::sedang` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/B-14-Input-Receive-Document/issues/03-kaitan-dokumen-ke-klaim.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu kiriman dokumen dapat **ditautkan ke klaim**, dan sejak saat itu tanggal terimanya menjadi
tanggal resmi klaim tersebut.

Nilai bisnisnya: tautan inilah yang membuat **TAT dapat dihitung dari titik yang benar**. Tanpa
tautan, perhitungan TAT dimulai dari tanggal registrasi — dan itu menyembunyikan waktu tunggu
dokumen, yang justru sering menjadi penyebab keterlambatan.

#### Ruang lingkup

- Penautan kiriman ke klaim, satu arah: satu klaim dapat menunjuk satu nomor register dokumen.
- **Tanggal terima dokumen mengalir ke klaim** dan menjadi batas pada validasi tanggal
  (`TKT-B02-002`).
- Perilaku bila kiriman ditautkan ke klaim yang sudah punya tautan — ditolak, dengan pesan yang
  menyebut nomor register yang sudah tertaut.
- Pencatatan jejak audit pada penautan dan pelepasan tautan.

#### Non-goal

- **Tidak** menghitung TAT — itu `S-7`; modul ini menyediakan titik awalnya.
- **Tidak** mengunggah berkas — itu `S-1`.

#### Acceptance criteria

- ☐ Kiriman dapat ditautkan ke klaim, dan tanggal terimanya **muncul pada klaim** — diuji.
- ☐ Menautkan kiriman kedua ke klaim yang sudah tertaut **ditolak**, dengan pesan yang menyebut
      nomor register yang sudah ada — diuji.
- ☐ Melepas tautan mengembalikan kiriman ke inbox dan **mengosongkan tanggal terima pada klaim**
      — diuji, dan keduanya tercatat di jejak audit.
- ☐ Mengubah tanggal terima setelah tertaut **memicu validasi ulang** urutan tanggal klaim
      (`TKT-B02-002`) — diuji dengan tanggal yang membuat urutan menjadi tidak sah: perubahan
      **ditolak**.
- ☐ Klaim tanpa kiriman tertaut tetap dapat diproses — penautan **tidak wajib** — diuji.
- ☐ Gerbang 1: kaitan yang terbentuk **sama dengan Pega** pada 20 klaim contoh di staging.
- ☐ Gerbang 2: UAT **PncReceive** dan **PncAdmin** bersama, karena tautan ini menyentuh keduanya.

#### Dependency / Blocked by

`TKT-B14-001` · `TKT-B02-001` (klaim harus ada untuk ditautkan) · `TKT-S5-002`.

#### Constraint keamanan, data, operasional

- Perubahan tanggal terima setelah klaim berjalan **mengubah angka TAT yang dilaporkan ke
  manajemen** — karena itu ia diaudit, dan perubahannya memicu validasi ulang, bukan diterima
  diam-diam.
- Penautan menyentuh dua tabel (kiriman dan klaim) — wajib dalam satu transaksi (`TKT-F2-003`).
- Selama masa paralel, **penulis tunggal per tabel** berlaku: bila tabel klaim masih dimiliki Pega,
  modul ini hanya boleh membaca (`P-1`).

#### Migrasi skema / rollout / rollback

Menambah kolom rujukan nomor register dokumen pada tabel klaim — **kolom *nullable***, sehingga
backward-compatible dan aman bagi Pega (`P-4`).

**Rollback:** kolom dibiarkan dan diabaikan; tautan yang sudah terbentuk tidak hilang.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/receivedoc/... -run TestPenautanKeKlaim
go test ./internal/app/receivedoc/... -run TestTolakTautanGanda
go test ./internal/app/receivedoc/... -run TestUbahTanggalTerimaMemicuValidasi
go run ./cmd/s8 banding --modul B-14 --aturan penautan --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `RCV_ID` sebagai nomor register dokumen | `CONTEXT.md` — **Receive Document** · `docs/Steering/05-DOMAIN-MODEL.md` §4 |
| Tanggal terima dokumen sebagai batas validasi | Invarian `I-2` |
| Dua basis perhitungan TAT | `D-50` · `ADR-0020` |
| Penulis tunggal per tabel | `P-1` · `ADR-0004` |

#### Comments

# 22. S-1 · Dokumen & Lampiran Klaim

*Modul Pendukung · folder `docs/ticketing/S-1-Dokumen-dan-Lampiran-Klaim/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Dokumen Klaim** — harness `PNCArchiveDokumen`, `ListDocumentObject`, `ListDocumentTravel`, `ListDocumentTypeInbox`, `DetTypeDocumenBisnis`, `BrowseMasterDocumentTravel_Harness`; activity `InsertDokumenPNC`, `UploadDocumentToGoogleStorage`, `SET_ATTACHMENT_64BIT` |
| **Kode modul** | `S-1` |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | 54 activity |
| **Bergantung pada** | `F-1`, `F-2`, `U-2` |
| **Kesiapan** | **SEBAGIAN** |

### Apa yang dikerjakan modul ini

Menyimpan dan mengambil **dokumen klaim** — berkas yang diunggah petugas, foto survei, lampiran
salvage, dan dokumen pendukung lain — lewat **satu jalur API storage internal** (`ADR-0010`).

Basis data hanya menyimpan **metadata dan referensi**: ID gambar, URL, masa berlaku, dan kategori.

### Tiga mekanisme lama yang disatukan

Sistem lama menyimpan dokumen lewat **tiga mekanisme berbeda** yang tumbuh berurutan dan masih
hidup bersamaan. Akibatnya tidak ada satu tempat pun yang menjawab "di mana dokumen klaim ini
berada" tanpa memeriksa ketiganya.

> **Catatan yang mencegah salah hitung.** `Activity/UploadDocumentToGoogleStorage-Act.xml`
> **bukan mekanisme keempat** — rule itu **nol `<pyMethod>`** dan mendelegasikan seluruh kerjanya
> lewat `Call InsertDokumenPNC` (`:1849`). Namanya menyesatkan, perilakunya tidak.

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | `SET_ATTACHFILETEMPSALVAGE` · `POOLDATA.base64decode` | **DBA** |
| **Keputusan** | **Token MD5 tanpa secret dan tanpa TTL — diterima Keamanan Informasi?** | **Keamanan Informasi** |
| **Keputusan** | TTL token dan URL berapa? · siapa membersihkan `GCP_IMAGE`? | **Work Owner** |
| **Keputusan** | **`COMMIT` di 4 lapis → pola outbox, atau menerima dokumen yatim?** | **Work Owner + Lead Engineer** |
| **Keputusan** | `SET_ATTACHMENT_64BIT` **menghapus pada `tCOMMAND` apa pun selain `'INSERT'`** — sengaja? | **Work Owner** |
| **Keputusan** | Dokumen lama pada tiga mekanisme — dimigrasikan, atau dibaca di tempatnya selamanya? | **Work Owner** (`ADR-0010`) |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-S1-001](issues/01-satu-jalur-unggah-dan-metadata.md) | Satu jalur unggah dan metadata dokumen | `needs-info` |
| [TKT-S1-002](issues/02-akses-dokumen-dan-token.md) | Akses dokumen, token, dan masa berlakunya | `needs-info` |

## Daftar Tiket

### TKT-S1-001 — Satu jalur unggah dan metadata dokumen

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan** |
| **Modul** | **S-1 Dokumen & Lampiran** · Gelombang: 3 · Bergantung pada: TKT-U2-003, TKT-F2-003 |
| **Requirement** | FR-S1 |
| **Keputusan** | D-16 |
| **ADR** | 0010, 0012 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | **tiga mekanisme penyimpanan** yang hidup bersamaan · `Activity/InsertDokumenPNC` · `Activity/UploadDocumentToGoogleStorage-Act.xml` (**nol `<pyMethod>`**, mendelegasikan lewat `Call InsertDokumenPNC` pada `:1849`) · `SET_ATTACHMENT_64BIT` · `DATAPEGA.PC_LINK_ATTACHMENT` |
| **Peran penguji gerbang 2** | **PncAdmin** dan **PNCSurveyor** |
| **Label** | `modul::S-1` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/S-1-Dokumen-dan-Lampiran-Klaim/issues/01-satu-jalur-unggah-dan-metadata.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Seluruh dokumen klaim masuk lewat **satu jalur**, dan basis data hanya menyimpan **metadata**.

Nilai bisnisnya: hari ini ada **tiga mekanisme** penyimpanan yang hidup bersamaan, sehingga
menjawab "di mana dokumen klaim ini" menuntut memeriksa ketiganya. Satu jalur berarti satu tempat
mencari dan satu tempat memperbaiki.

#### Ruang lingkup

- Satu jalur unggah lewat **API storage internal Sinarmas** yang sudah berjalan (`ADR-0010`).
- Metadata di basis data: ID gambar, URL, masa berlaku, kategori, kaitan ke klaim atau objek.
- Kategori dan kelengkapan dokumen per lini bisnis, dari master.
- Penanganan **dokumen yatim** — berkas tersimpan tanpa metadata, atau sebaliknya.

#### Non-goal

- **Tidak** membangun komponen unggah — itu `TKT-U2-003`.
- **Tidak** memigrasikan dokumen lama — keputusannya belum diambil.
- **Tidak** menyimpan berkas di basis data.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **`COMMIT` di 4 lapis → pola outbox, atau menerima dokumen yatim?** | **Work Owner + Lead Engineer** | `ADR-0010` mencatat **tidak ada atomisitas** antara berkas dan metadata. Pola penanganannya belum dipilih, dan itu menentukan bentuk tabel |
| **`SET_ATTACHMENT_64BIT` menghapus pada `tCOMMAND` apa pun selain `'INSERT'`** — sengaja? | **Work Owner** | Perilaku menghapus sebagai efek samping perintah yang bukan hapus adalah pola berbahaya; bila tidak sengaja, ia celah yang sudah berjalan |
| **Dokumen lama pada tiga mekanisme — dimigrasikan atau dibaca di tempatnya selamanya?** | **Work Owner** (`ADR-0010`) | Menyatukan jalur tulis **tidak menyatukan data yang sudah ada** |
| `SET_ATTACHFILETEMPSALVAGE` dan `POOLDATA.base64decode` | **DBA** | Aturan lampiran salvage dan penyandian berkas |

#### Acceptance criteria

- ☐ Seluruh unggah melewati **satu jalur**; **nol jalur alternatif** — diuji pemindaian.
- ☐ Basis data menyimpan **hanya metadata**; **nol kolom berisi isi berkas** — diuji pemeriksaan
      skema.
- ☐ Berkas terkirim tetapi metadata gagal tersimpan **terdeteksi dan dilaporkan**, bukan diam —
      diuji dengan kegagalan sengaja.
- ☐ Metadata tersimpan tetapi berkas gagal terkirim **tidak meninggalkan metadata yang menunjuk
      berkas tidak ada** — diuji.
- ☐ Penghapusan dokumen adalah **soft delete** pada metadata (`ADR-0012`); perilaku terhadap
      berkas di storage mengikuti keputusan Work Owner — diuji sesuai jawabannya.
- ☐ Kategori dokumen dan kelengkapannya dibaca dari master, bukan konstanta.
- ☐ Gerbang 1: metadata yang tersimpan **sama dengan Pega** pada 20 dokumen contoh.
- ☐ Gerbang 2: UAT **PncAdmin** dan **PNCSurveyor**.

#### Dependency / Blocked by

`TKT-U2-003` · `TKT-F2-003` · `TKT-F4-001` (master kategori dokumen). **Terhalang empat keputusan
dan DBA.**

#### Constraint keamanan, data, operasional

- Dokumen klaim memuat **data nasabah**, termasuk **data medis** pada lini PA dan Travel —
  aksesnya tunduk `FR-R2` dan berlaku **juga di staging** (`ADR-0029`).
- **Penyimpanan dokumen menjadi ketergantungan runtime pada sistem lain** (`ADR-0010`): bila API
  storage mati, unggah berhenti meski seluruh aplikasi sehat.
- Soft delete **tidak berlaku pada berkas di storage** — menghapus metadata tidak menghapus
  berkasnya, dan itu keputusan tersendiri.

#### Migrasi skema / rollout / rollback

Menambah tabel metadata dokumen. Backward-compatible; tidak menyentuh `PC_LINK_ATTACHMENT` yang
masih dibaca Pega.

**Rollback:** metadata tetap ada; berkas di storage **tidak ikut dibersihkan**.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/dokumen/... -run TestSatuJalurUnggah
go test ./internal/app/dokumen/... -run TestBerkasTerkirimMetadataGagal
go run ./cmd/tools/cek-skema-tanpa-blob    # HARUS 0 kolom berisi isi berkas
go run ./cmd/s8 banding --modul S-1 --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Tiga mekanisme penyimpanan disatukan | `D-16` · `ADR-0010` |
| `UploadDocumentToGoogleStorage` bukan mekanisme keempat | `Activity/UploadDocumentToGoogleStorage-Act.xml:1849` |
| Tidak ada atomisitas berkas dan metadata | `ADR-0010` Negatif/utang teknis |
| `SET_ATTACHMENT_64BIT` menghapus pada `tCOMMAND` selain `'INSERT'` | `docs/verifikasi-bukti-adr.md` §15 baris `S-1` |
| `PC_LINK_ATTACHMENT` dibaca 3 rule Pega | `D-21` · `ADR-0004` |

#### Comments

### TKT-S1-002 — Akses dokumen, token, dan masa berlakunya

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan Keamanan Informasi** |
| **Modul** | **S-1 Dokumen & Lampiran** · Gelombang: 3 · Bergantung pada: TKT-S1-001, TKT-F3-005 |
| **Requirement** | FR-S1, FR-R2 |
| **Keputusan** | D-16, D-64 |
| **ADR** | 0010, 0023, 0029 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | pembentukan token akses dokumen — **token MD5 tanpa secret dan tanpa TTL** · `GCP_IMAGE` · harness `PNCArchiveDokumen` |
| **Peran penguji gerbang 2** | **PncAdmin**, **PncAnalystDoctor** (untuk dokumen medis) |
| **Label** | `modul::S-1` `tipe::keamanan` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/S-1-Dokumen-dan-Lampiran-Klaim/issues/02-akses-dokumen-dan-token.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Dokumen klaim hanya dapat dibuka orang yang berwenang, dan tautannya **tidak berlaku selamanya**.

Nilai bisnisnya: dokumen klaim memuat data nasabah dan **data medis**. Token yang dibentuk **MD5
tanpa secret dan tanpa masa berlaku** berarti siapa pun yang pernah memegang tautannya — atau yang
dapat menebak polanya — dapat membukanya **kapan saja, selamanya**.

#### Ruang lingkup

- Penegakan izin pada pengambilan dokumen: peran pemanggil diperiksa **di server** (`ADR-0023`).
- Penegakan `FR-R2`: dokumen medis hanya untuk **Analyst Doctor** dan **RCL Dokter**.
- Token akses dokumen dengan **masa berlaku**, dan dibentuk dengan cara yang tidak dapat ditebak.
- Perilaku saat token kedaluwarsa: pesan yang jelas, bukan berkas tidak ditemukan.

#### Non-goal

- **Tidak** merancang skema penyimpanan — itu `TKT-S1-001`.
- **Tidak** membersihkan `GCP_IMAGE` — kepemilikannya belum ditetapkan.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Token MD5 tanpa secret dan tanpa TTL — diterima Keamanan Informasi?** | **Keamanan Informasi** | Bila tidak, mekanismenya diganti seluruhnya — dan itu mengubah bentuk tiket ini |
| **TTL token dan URL berapa?** | **Work Owner + Keamanan Informasi** | Terlalu pendek mengganggu kerja; terlalu panjang memperpanjang paparan |
| **Siapa membersihkan `GCP_IMAGE`?** | **Work Owner** | Berkas yang tidak pernah dibersihkan menumpuk; tanpa pemilik, tidak ada yang mengerjakannya |

#### Acceptance criteria

- ☐ Pengambilan dokumen **memeriksa izin di server** — permintaan tanpa kewenangan mengembalikan
      `403`, diuji lewat URL langsung.
- ☐ Dokumen **medis** hanya dapat diambil peran Analyst Doctor dan RCL Dokter — diuji dengan
      peran lain: `403`. Berlaku **juga di staging** (`ADR-0029`).
- ☐ Token **kedaluwarsa** setelah masa berlaku yang dikonfigurasi — diuji dengan TTL pendek.
- ☐ Token **tidak dapat ditebak**: dua dokumen berbeda menghasilkan token yang tidak berpola —
      diuji.
- ☐ Token kedaluwarsa menghasilkan pesan **"tautan sudah tidak berlaku"**, berbeda dari
      "dokumen tidak ditemukan" — keduanya diuji.
- ☐ Setiap pengambilan dokumen medis **tercatat** — siapa membuka, dokumen apa, kapan.
- ☐ Gerbang 2: UAT **PncAdmin** dan **PncAnalystDoctor** secara terpisah — justru pemisahan
      aksesnya yang diuji.

#### Dependency / Blocked by

`TKT-S1-001` · `TKT-F3-005`. **Terhalang Keamanan Informasi dan Work Owner.**

#### Constraint keamanan, data, operasional

- **Token MD5 tanpa secret dapat ditebak.** Bila pola pembentukannya diketahui, seluruh dokumen
  klaim dapat diakses tanpa login — ini kelas kerentanan yang tidak boleh dibawa apa adanya.
- Dokumen memuat data nasabah dan data medis; pembatasan `FR-R2` berlaku pada **berkasnya**, bukan
  hanya pada field di layar.
- Staging memuat **data produksi apa adanya** (`D-64`) — pembatasan ini berlaku di sana juga.

#### Migrasi skema / rollout / rollback

Menambah kolom token dan masa berlaku pada metadata dokumen. Backward-compatible.

**Rollback:** memperpanjang masa berlaku token **bukan rollback yang sah** — ia memperluas paparan.
Rollback yang benar adalah mengembalikan versi kode, bukan melonggarkan aturannya.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/http/... -run TestAksesDokumenMemeriksaIzin
go test ./internal/adapter/http/... -run TestDokumenMedisDibatasi
go test ./internal/app/dokumen/... -run TestTokenKedaluwarsa
go run ./cmd/tools/uji-token-tebakan --sample 1000   # tidak boleh berpola
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Token MD5 tanpa secret dan tanpa TTL | `docs/verifikasi-bukti-adr.md` §15 baris `S-1` |
| Akses data medis dibatasi dua peran | `FR-R2` · `BRD §17.2` |
| Otorisasi diperiksa di setiap endpoint | `D-59` · `ADR-0023` |
| Staging memuat data produksi apa adanya | `D-64` · `ADR-0029` |
| Masa berlaku URL sebagai metadata | `D-16` · `ADR-0010` |

#### Comments

# 23. S-2 · Laporan & Export

*Modul Pendukung · folder `docs/ticketing/S-2-Laporan-dan-Export/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Laporan** — 56 Report Definition; harness `Har_LaporanHasilAI`, `ReportKPIHarness`, `PNCTATReport`, `OutstandingKlaimperCabang_Harness`, `MonitoringSLINKOJK`, `PNCStudyClaim` |
| **Kode modul** | `S-2` |
| **Gelombang** | 6 — Laporan |
| **Ukuran** | 78 activity · 56 Report Definition |
| **Bergantung pada** | seluruh modul bisnis |
| **Kesiapan** | **SEBAGIAN** |

### Apa yang dikerjakan modul ini

Membangun **56 laporan** beserta engine pembuat **PDF, Excel, dan CSV** di dalam aplikasi Go
sendiri — tanpa engine reporting Pega dan tanpa tools BI eksternal (`ADR-0011`).

### Temuan yang mengubah rencana

| Anggapan | Terverifikasi |
|---|---|
| Masalahnya `OFFSET` besar | **`OFFSET` nol kemunculan** di seluruh export |
| Laporan memaginasi hasil besar | **`pyMaxRecords=500` pada 54 dari 56 laporan** — hasilnya **dipotong**, bukan dipaginasi |

Artinya kebutuhan export bervolume besar **belum pernah benar-benar dilayani**. Menghapus batas
500 baris adalah **penambahan kemampuan**, bukan penyalinan — dan ia **mengubah angka yang dilihat
pengguna**.

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **12 Report Definition** (`BrowseVPanel_HE_RD` 82 pemakaian) · 10 template HTML · 3 Correspondence | **Tim Pega** (`R-16`) |
| **Keputusan** | **54 dari 56 laporan memotong hasil di 500 baris — pengguna tahu?** Menampilkan seluruh baris **mengubah angka yang mereka lihat** | **Work Owner** |
| **Keputusan** | Berapa baris maksimum yang wajib dilayani satu export? | **Work Owner** (`ADR-0011`) |
| **Keputusan** | Export besar dijalankan serentak, atau diantrekan lalu diberitahukan? | **Work Owner** |
| **Keputusan** | Keluaran PDF wajib identik **visual** dengan Pega, atau cukup identik **isi**? | **Work Owner** |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-S2-001](issues/01-engine-pdf-excel-csv.md) | Engine PDF, Excel, dan CSV | `needs-info` |
| [TKT-S2-002](issues/02-56-laporan-dan-batas-500-baris.md) | 56 laporan dan penghapusan batas 500 baris | `needs-info` |

## Daftar Tiket

### TKT-S2-001 — Engine PDF, Excel, dan CSV

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | terhalang keputusan (kriteria kesamaan keluaran) |
| **Modul** | **S-2 Laporan & Export** · Gelombang: 6 · Bergantung pada: TKT-F1-005, TKT-U2-004 |
| **Requirement** | FR-S2 |
| **Keputusan** | D-11 |
| **ADR** | 0011 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | engine reporting Pega · 10 template HTML · `Activity/PrintPDFAcceptanceNote-Act.xml` · 3 Correspondence |
| **Peran penguji gerbang 2** | **PNCReportClaimInternal** dan **PncManagerAdmin** |
| **Label** | `modul::S-2` `tipe::fondasi` `status::needs-info` `prioritas::tinggi` `gelombang::6` |
| **Milestone** | Gelombang 6 — Laporan |
| **Berkas sumber** | `ticketing/S-2-Laporan-dan-Export/issues/01-engine-pdf-excel-csv.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu cara membuat PDF, Excel, dan CSV di dalam aplikasi — dipakai 56 laporan, LOD, dan dokumen
PLA/DLA.

Nilai bisnisnya: engine reporting Pega lenyap bersama platformnya. Tanpa penggantinya, **tidak
satu pun laporan dapat diterbitkan** — dan laporan adalah hal yang dilihat manajemen setiap bulan.

#### Ruang lingkup

- Pembuatan **PDF** dengan tata letak yang dapat didefinisikan ulang per dokumen.
- Pembuatan **Excel** dan **CSV**, keduanya **dialirkan (streaming)** agar memori tetap datar
  berapa pun jumlah barisnya.
- Pemformatan angka dan tanggal memakai pemformat terpusat `TKT-U2-004` — agar angka di laporan
  **sama dengan angka di layar**.
- Pool koneksi terpisah untuk laporan (`TKT-F2-001`).

#### Non-goal

- **Tidak** membangun 56 laporan — itu `TKT-S2-002`.
- **Tidak** memakai tools BI eksternal maupun engine berbayar (`ADR-0011`).

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Keluaran PDF wajib identik visual dengan Pega, atau cukup identik isi?** | **Work Owner** (`ADR-0011`) | Menentukan kriteria kelulusan gerbang 1 untuk **seluruh** dokumen cetak — termasuk LOD dan PLA/DLA |
| **Berapa baris maksimum yang wajib dilayani satu export?** | **Work Owner** | Tanpa angka, `S-2` tidak dapat dinyatakan selesai secara terukur |
| **Export besar serentak, atau diantrekan lalu diberitahukan saat selesai?** | **Work Owner + Lead Engineer** | Menentukan apakah dibutuhkan mekanisme antrean dan pemberitahuan |

#### Acceptance criteria

- ☐ PDF, Excel, dan CSV dapat dihasilkan dari satu sumber data yang sama — diuji ketiganya.
- ☐ Export **100.000 baris** berjalan dengan **memori datar** — diuji dengan pengukuran; memori
      tidak tumbuh sebanding jumlah baris.
- ☐ Angka dan tanggal pada laporan **sama persis** dengan yang ditampilkan di layar untuk data
      yang sama — diuji pada 10 nilai, termasuk uang berdesimal.
- ☐ Laporan berat **tidak menghabiskan koneksi transaksi** — diuji dengan menjalankan export
      besar sambil mengirim permintaan transaksi biasa.
- ☐ Kesamaan keluaran dengan Pega diuji sesuai kriteria yang ditetapkan Work Owner — pada 10
      dokumen contoh.
- ☐ Kegagalan pembuatan laporan **tidak menjatuhkan aplikasi** — diuji dengan data yang sengaja
      rusak.
- ☐ Gerbang 2: UAT **PNCReportClaimInternal**.

#### Dependency / Blocked by

`TKT-F1-005` · `TKT-U2-004` · `TKT-F2-001`. **Terhalang tiga keputusan Work Owner.**

#### Constraint keamanan, data, operasional

- Laporan memuat **data nasabah**; batas data cabang dan lini bisnis berlaku pada isinya
  (`TKT-F3-005`) — laporan **tidak boleh** menjadi jalan memintas pembatasan layar.
- Export besar memakan memori dan CPU yang sama dengan pelayanan transaksi (`ADR-0001`) — tanpa
  pembatasan, satu export dapat memperlambat seluruh pengguna.
- Data medis pada laporan tunduk `FR-R2`.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema.

**Rollback:** mengembalikan versi engine; laporan yang sudah dihasilkan tetap ada sebagai berkas.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/laporan/... -run TestPDFExcelCSV
go test ./internal/app/laporan/... -run TestMemoriDatarSaatExportBesar
go test ./internal/app/laporan/... -run TestPoolLaporanTerpisah
go run ./cmd/tools/banding-pdf keluaran/ baseline-pega/
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| PDF/Excel/CSV dibangun sendiri di Go | `D-11` · `ADR-0011` |
| 56 laporan, 78 activity | `docs/Steering/06-MODULE-BREAKDOWN.md` `S-2` |
| Export besar asinkron dan streaming | `docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md` butir 6 |
| Pool koneksi terpisah untuk laporan | idem butir 7 |
| Kriteria kesamaan keluaran belum ditetapkan | `ADR-0011` Pertanyaan terbuka |

#### Comments

### TKT-S2-002 — 56 laporan dan penghapusan batas 500 baris

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak dan keputusan** |
| **Modul** | **S-2 Laporan & Export** · Gelombang: 6 · Bergantung pada: TKT-S2-001 |
| **Requirement** | FR-S2 |
| **Keputusan** | D-11 |
| **ADR** | 0011 |
| **Risiko** | R-16 |
| **Rule Pega yang digantikan** | **56 Report Definition**, **54 di antaranya ber-`pyMaxRecords=500`** · **12 Report Definition hilang dari export** (`BrowseVPanel_HE_RD` **82 pemakaian**) · harness `PNCTATReport`, `ReportKPIHarness`, `OutstandingKlaimperCabang_Harness`, `MonitoringSLINKOJK` |
| **Peran penguji gerbang 2** | **PNCReportClaimInternal**, **PNCReportClaimEksternal**, **PncManagerAdmin** |
| **Label** | `modul::S-2` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::6` |
| **Milestone** | Gelombang 6 — Laporan |
| **Berkas sumber** | `ticketing/S-2-Laporan-dan-Export/issues/02-56-laporan-dan-batas-500-baris.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Kelima puluh enam laporan tersedia di sistem baru, **tanpa pemotongan diam-diam di 500 baris**.

Nilai bisnisnya sekaligus peringatannya: batas 500 baris berarti laporan yang dilihat manajemen
selama ini **mungkin tidak lengkap** — dan menghapus batas itu akan **mengubah angka yang mereka
lihat**. Perubahan itu harus disampaikan lebih dulu, bukan ditemukan sebagai kejutan.

#### Ruang lingkup

- Pemindahan 56 laporan, memakai engine `TKT-S2-001`.
- **Penghapusan batas 500 baris**, dengan penggantinya berupa batas yang disepakati dan
  **diberitahukan kepada pengguna** saat hasil melebihi.
- Penegakan batas data pada setiap laporan (`TKT-F3-005`).
- Daftar laporan beserta pemiliknya, sehingga setiap laporan punya penguji gerbang 2.

#### Non-goal

- **Tidak** menambah laporan baru.
- **Tidak** merancang ulang isi laporan.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **12 Report Definition hilang dari export**, terberat `BrowseVPanel_HE_RD` (**82 pemakaian**) · 10 template HTML · 3 Correspondence | **Tim Pega** (`R-16`) | Isi dan penyaring laporan itu tidak diketahui |
| **54 dari 56 laporan memotong di 500 baris — pengguna tahu?** | **Work Owner** | Bila tidak tahu, menghapus batas akan mengubah angka yang mereka percayai selama ini. Ini **perubahan yang harus diumumkan**, bukan diam-diam |
| **Berapa batas baru, dan bagaimana pengguna diberi tahu bila hasil melebihi?** | **Work Owner** | Tanpa itu, laporan besar kembali terpotong diam-diam — hanya pada angka berbeda |

#### Acceptance criteria

- ☐ Kelima puluh enam laporan tersedia, dan **jumlahnya dilaporkan sebagai angka** — bukan
      pernyataan "semua sudah ada".
- ☐ Laporan yang hasilnya melebihi batas **memberi tahu pengguna**, bukan memotong diam-diam —
      diuji dengan data melebihi batas.
- ☐ Batas data cabang dan lini bisnis ditegakkan pada setiap laporan — diuji pada 5 laporan
      dengan dua peran berbeda.
- ☐ Laporan yang memuat **data medis** hanya dapat dijalankan peran yang berwenang (`FR-R2`) —
      diuji.
- ☐ Gerbang 1: isi laporan **sama dengan Pega** pada 20 laporan contoh — **dengan pengecualian
      yang dinyatakan di muka**: laporan yang di Pega terpotong di 500 baris akan berbeda, dan
      selisih itu **wajib dilaporkan sebagai perbaikan terencana**, bukan bug.
- ☐ Gerbang 2: UAT **per pemilik laporan**, bukan satu orang untuk 56 laporan.

#### Dependency / Blocked by

`TKT-S2-001` · seluruh modul bisnis (sumber datanya). **Terhalang Tim Pega dan dua keputusan.**

#### Constraint keamanan, data, operasional

- **Menghapus batas 500 baris mengubah angka yang dilihat manajemen.** Ini bukan perbaikan teknis
  diam-diam — ia perubahan yang terlihat, dan harus diumumkan sebelum rilis.
- Laporan adalah jalan paling mudah untuk **memintas pembatasan data layar** bila batasnya tidak
  ditegakkan di kueri laporan juga.
- Kapasitas dirancang **tanpa data historis yang sahih**, karena angka pemakaian selama ini selalu
  terpotong di 500 (`docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md`).

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema; menambah index bila kueri laporan membutuhkannya (`TKT-F2-004`).

**Rollback:** mengembalikan batas 500 baris **mengembalikan pemotongan diam-diam** — bila
dilakukan, itu harus disampaikan ke pengguna, bukan dianggap netral.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/cek-laporan --harap 56          # HARUS 56
go test ./internal/app/laporan/... -run TestBatasDataPerLaporan
go test ./internal/app/laporan/... -run TestPemberitahuanSaatMelebihiBatas
go run ./cmd/s8 banding --modul S-2 --laporan 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `pyMaxRecords=500` pada 54 dari 56 laporan | `T-12` · `ADR-0011` |
| `OFFSET` nol kemunculan di export | `T-12` |
| 12 Report Definition hilang; `BrowseVPanel_HE_RD` 82 pemakaian | `docs/verifikasi-bukti-adr.md` §15 baris `S-2` · `R-16` |
| Menghapus batas adalah penambahan kemampuan | `ADR-0011` Negatif/utang teknis |
| Akses data medis dibatasi | `FR-R2` |

#### Comments

# 24. S-3 · Notifikasi & Korespondensi Email

*Modul Pendukung · folder `docs/ticketing/S-3-Notifikasi-dan-Email/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Notifikasi / Correspondence** — `SendEmailNotification` (**15 pemanggil**), template HTML, rule Correspondence; `Data Transform/SetDataEmail-DT.xml` |
| **Kode modul** | `S-3` |
| **Gelombang** | 5 — Nilai dan pihak luar |
| **Ukuran** | 14 activity |
| **Bergantung pada** | `F-4` Master Data |
| **Kesiapan** | **TERHALANG** |

### Apa yang dikerjakan modul ini

Mengirim pemberitahuan berdasarkan **peristiwa domain** — bukan berdasarkan perintah "kirim email
ke alamat ini". Pemanggil menyatakan **apa yang terjadi**; modul ini yang menentukan **siapa yang
perlu tahu**, dari master data.

### Kenapa modul ini TERHALANG

Tiga hal sekaligus: **activity pengirimnya hilang dari export**, **isi templatenya tidak diketahui**,
dan **kredensial SMTP-nya belum punya tempat**.

### Yang tidak diketahui tentang modul ini

- **Tidak ada satu pun direktori `Correspondence/` di export.** Seluruh rule korespondensi —
  yakni isi surat dan emailnya — **tidak tersedia**.
- Jumlah rule Correspondence **belum pasti**: `docs/Steering/06-MODULE-BREAKDOWN.md:62` menyebut
  **5**, `docs/verifikasi-bukti-adr.md:2783` menyebut **3**. Keduanya adalah hitungan dari
  *rujukan* di rule lain, bukan dari direktorinya. **BELUM DIPUTUSKAN — pertanyaan terbuka**;
  angka sebenarnya hanya dapat dipastikan **Tim Pega**.

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **`SendEmailNotification` — 15 pemanggil**, tidak ada di export (`19-GAP-EXPORT-DETAIL.md:196`) | **Tim Pega** (`R-07`) |
| **Artefak** | **Seluruh rule Correspondence dan template HTML** — direktorinya tidak ada di export | **Tim Pega** (`R-16`) |
| **Keputusan** | **Rotasi 3 password SMTP (31 lokasi, plaintext)** dan tujuan penyimpanannya; `UseSSL=false` pada seluruh 14 kemunculan sementara 16 dari 31 lokasi memakai port 587 | **Tim Infra/Security** (`D-40`, `R-17`) |
| **Keputusan** | Pemetaan penerima personal ke mailbox fungsional pengganti | **Work Owner** (`D-67`) |

**Sudah diputuskan:** **tidak ada akun pribadi** sebagai penerima; seluruhnya mailbox fungsional
dari master data (`D-67`).

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-S3-001](issues/01-seam-notifier-berbasis-peristiwa.md) | Seam Notifier berbasis peristiwa domain | `needs-info` |
| [TKT-S3-002](issues/02-template-dan-pengiriman-email.md) | Template, penerima, dan pengiriman email | `needs-info` |

## Daftar Tiket

### TKT-S3-001 — Seam Notifier berbasis peristiwa domain

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** |
| **Modul** | **S-3 Notifikasi & Email** · Gelombang: 5 · Bergantung pada: TKT-F4-001 |
| **Requirement** | FR-S3 |
| **Keputusan** | D-67 |
| **ADR** | 0016 |
| **Risiko** | R-07 |
| **Rule Pega yang digantikan** | `SendEmailNotification` — **15 pemanggil, tidak ada di export** (`docs/Steering/19-GAP-EXPORT-DETAIL.md:196`) · `Data Transform/SetDataEmail-DT.xml` |
| **Peran penguji gerbang 2** | **PncAdmin** |
| **Label** | `modul::S-3` `tipe::fondasi` `status::needs-info` `prioritas::sedang` `gelombang::5` |
| **Milestone** | Gelombang 5 — Nilai dan pihak luar |
| **Berkas sumber** | `ticketing/S-3-Notifikasi-dan-Email/issues/01-seam-notifier-berbasis-peristiwa.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu titik di dalam aplikasi tempat pemberitahuan dilepaskan, dipanggil dengan **peristiwa yang
terjadi** — bukan dengan alamat email.

Nilai bisnisnya ada pada bentuk pemanggilannya. Bila pemanggil menyebut alamat, maka alamat itu
tersebar di 15 tempat dan setiap perubahan pejabat menuntut perubahan kode. Bila pemanggil menyebut
**peristiwa**, penerimanya ditentukan satu kali dari master data — dan berpindah pejabat menjadi
perubahan data, bukan rilis.

#### Ruang lingkup

- Antarmuka `Notifier` dengan satu operasi: **melepaskan peristiwa domain** beserta konteks klaimnya.
- Penentuan penerima **dari master data** (`TKT-F4-001`), bukan dari konstanta di kode.
- Pencatatan setiap pelepasan peristiwa: peristiwa apa, klaim mana, kapan, berhasil atau gagal.
- Perilaku saat pengiriman gagal: **proses bisnis tetap berjalan**, kegagalan tercatat dan terlihat.

#### Non-goal

- **Tidak** membuat template maupun menyambung ke SMTP — itu `TKT-S3-002`.
- **Tidak** membuat notifikasi dalam-aplikasi (lonceng, inbox pesan); belum ada permintaannya.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **`SendEmailNotification` tidak ada di export**, padahal dipanggil 15 activity | **Tim Pega** (`R-07`) | Daftar peristiwa yang sebenarnya memicu email **hanya dapat dibaca dari isi rule itu**. Tanpa itu, daftar peristiwa di tiket ini adalah dugaan — dan dugaan tidak boleh menjadi acceptance criteria |
| **Satu peristiwa mengirim ke berapa penerima, dan apakah ada penerima tembusan?** | **Work Owner** | Menentukan bentuk data penerima di master |
| **Kegagalan kirim perlu dicoba ulang otomatis, atau cukup dicatat?** | **Work Owner** | Menentukan apakah dibutuhkan antrean dan berapa kali percobaan |

> Daftar peristiwa **sengaja tidak dicantumkan di sini.** Menuliskannya dari dugaan akan membuat
> tiket ini tampak siap padahal isinya belum diketahui.

#### Acceptance criteria

- ☐ Pemanggil melepaskan **peristiwa**, bukan alamat email — diuji: tidak ada satu pun alamat
      email di kode modul bisnis mana pun (dicari secara otomatis).
- ☐ Penerima diambil **dari master data**; mengubah penerima **tidak menuntut rilis** — diuji
      dengan mengubah data lalu memanggil ulang.
- ☐ **Kegagalan pengiriman tidak menggagalkan proses bisnis** — diuji dengan pengirim yang
      sengaja dibuat gagal; klaim tetap tersimpan dan berpindah status.
- ☐ Setiap pelepasan peristiwa **tercatat** dengan hasilnya — diuji.
- ☐ Daftar peristiwa yang terpasang **cocok dengan isi `SendEmailNotification`** setelah rule itu
      diterima — diperiksa satu per satu. *Kriteria ini belum dapat dijalankan sampai artefaknya ada.*
- ☐ Gerbang 2: UAT **PncAdmin**.

#### Dependency / Blocked by

`TKT-F4-001` (master data penerima). **Terhalang Tim Pega (`R-07`).**

#### Constraint keamanan, data, operasional

- Isi email memuat **data nasabah**; ia keluar dari batas sistem, sehingga apa yang boleh
  dicantumkan di dalamnya adalah keputusan bisnis, bukan teknis.
- **Tidak ada akun pribadi** sebagai penerima (`D-67`) — seluruhnya mailbox fungsional.
- Email yang memuat **data medis** tunduk `FR-R2`.

#### Migrasi skema / rollout / rollback

Menambah tabel penerima per peristiwa di master data. Backward-compatible (`P-4`).

**Rollback:** mematikan pelepasan peristiwa **menghentikan pemberitahuan tanpa terlihat** — bila
dilakukan, harus disertai pemberitahuan ke pengguna yang bergantung padanya.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/notifikasi/... -run TestPeristiwaBukanAlamat
go test ./internal/app/notifikasi/... -run TestPenerimaDariMaster
go test ./internal/app/notifikasi/... -run TestGagalKirimTidakMenggagalkanProses
go run ./cmd/tools/cari-alamat-email ./internal/app/...   # HARUS nol temuan
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `SendEmailNotification` dipanggil 15× dan tidak ada di export | `docs/Steering/19-GAP-EXPORT-DETAIL.md:196` · `R-07` |
| Seam Notifier, penerima dari master data | `docs/Steering/06-MODULE-BREAKDOWN.md:62` |
| Tidak ada penerima akun pribadi | `D-67` |
| Modul bergantung pada `F-4` | `docs/Steering/06-MODULE-BREAKDOWN.md:62` |

#### Comments

### TKT-S3-002 — Template, penerima, dan pengiriman email

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak dan keputusan keamanan** |
| **Modul** | **S-3 Notifikasi & Email** · Gelombang: 5 · Bergantung pada: TKT-S3-001 |
| **Requirement** | FR-S3 |
| **Keputusan** | D-40, D-67 |
| **ADR** | 0016 |
| **Risiko** | R-16, R-17 |
| **Rule Pega yang digantikan** | seluruh rule **Correspondence** dan template HTML email — **tidak ada satu pun direktori `Correspondence/` di export** |
| **Peran penguji gerbang 2** | **PncAdmin** dan pemilik masing-masing pemberitahuan |
| **Label** | `modul::S-3` `tipe::migrasi` `status::needs-info` `prioritas::sedang` `gelombang::5` |
| **Milestone** | Gelombang 5 — Nilai dan pihak luar |
| **Berkas sumber** | `ticketing/S-3-Notifikasi-dan-Email/issues/02-template-dan-pengiriman-email.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Email yang benar-benar terkirim: isinya sesuai template, penerimanya dari master, dan kredensial
SMTP-nya tidak berada di dalam kode maupun di dalam repositori.

Nilai bisnisnya: `TKT-S3-001` menyiapkan **kapan** email dilepaskan; tiket ini yang menentukan
**apa bunyinya** dan **bagaimana ia sampai**. Tanpa keduanya, seam Notifier tidak mengirim apa pun.

#### Ruang lingkup

- Mesin template email dengan isi yang dapat diubah tanpa rilis.
- Penyambungan SMTP, dengan kredensial diambil dari penyimpanan rahasia — **bukan** dari berkas
  konfigurasi yang masuk repositori.
- **TLS aktif** pada koneksi SMTP.
- Pencatatan pengiriman: peristiwa, penerima, waktu, hasil.

#### Non-goal

- **Tidak** merancang ulang bunyi email; isinya menyalin yang berlaku sekarang — sejauh artefaknya
  tersedia.
- **Tidak** memindahkan kredensial produksi. Rotasi dan penyimpanannya milik **Tim Infra/Security**.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Seluruh rule Correspondence dan template HTML** — direktorinya tidak ada di export. Jumlahnya pun belum pasti: `06-MODULE-BREAKDOWN.md:62` menyebut **5**, `verifikasi-bukti-adr.md:2783` menyebut **3** | **Tim Pega** (`R-16`) | **Isi email tidak diketahui sama sekali.** Tiket ini tidak dapat menyatakan satu pun template selesai |
| **Rotasi 3 password SMTP (31 lokasi, plaintext)** — masih aktif? diganti? disimpan di mana? | **Tim Infra/Security** (`D-40`, `R-17`) | Tanpa tempat penyimpanan rahasia, satu-satunya cara menyambung SMTP adalah mengulangi kesalahan yang sama |
| **`UseSSL=false` pada seluruh 14 kemunculan, sementara 16 dari 31 lokasi memakai port 587** — apakah itu yang berlaku di produksi? | **Tim Infra/Security** (`D-40`) | Bila benar, email selama ini terkirim **tanpa enkripsi**; menyalakan TLS adalah perubahan perilaku yang harus disepakati, bukan diam-diam |
| **Pemetaan penerima personal ke mailbox fungsional** | **Work Owner** (`D-67`) | `D-67` melarang akun pribadi; penggantinya belum ditunjuk |

#### Acceptance criteria

- ☐ Kredensial SMTP **tidak ada di dalam kode maupun di repositori** — diuji dengan pemindaian
      otomatis atas seluruh berkas yang di-commit: **nol temuan**.
- ☐ Koneksi SMTP memakai **TLS** — diuji; koneksi tanpa TLS ditolak.
- ☐ Isi email dapat diubah **tanpa rilis** — diuji dengan mengubah template lalu mengirim ulang.
- ☐ **Tidak ada satu pun alamat akun pribadi** sebagai penerima di data produksi (`D-67`) —
      diperiksa atas seluruh data penerima.
- ☐ Setiap pengiriman **tercatat** dengan hasilnya — diuji.
- ☐ Isi email **sama dengan yang dikirim Pega**, diperiksa per template setelah artefaknya
      diterima. *Kriteria ini belum dapat dijalankan; jumlah templatenya pun belum pasti.*
- ☐ Gerbang 2: UAT oleh **pemilik masing-masing pemberitahuan**, bukan satu orang untuk semuanya.

#### Dependency / Blocked by

`TKT-S3-001`. **Terhalang Tim Pega (`R-16`) dan Tim Infra/Security (`D-40`, `R-17`).**

#### Constraint keamanan, data, operasional

- **Export memuat 3 password SMTP di 31 lokasi dalam bentuk plaintext** (`R-17`). Nilai-nilainya
  **tidak boleh disalin ke dokumen mana pun yang di-commit**; lokasinya sudah diserahkan terpisah
  ke Tim Infra/Security.
- `UseSSL=false` pada seluruh kemunculannya berarti **email kemungkinan terkirim tanpa enkripsi
  selama ini**. Menyalakan TLS memperbaiki keadaan, tetapi ia **perubahan perilaku** dan tunduk
  `P-5` — harus dinyatakan, bukan diselipkan.
- Email membawa data nasabah ke luar batas sistem; email berisi **data medis** tunduk `FR-R2`.

#### Migrasi skema / rollout / rollback

Menambah tabel template. Tidak menyentuh tabel klaim.

**Rollback:** mengembalikan versi template. **Mematikan TLS bukan rollback yang sah** — ia
mengembalikan pengiriman tanpa enkripsi.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/pindai-rahasia .            # HARUS nol temuan
go test ./internal/adapter/smtp/... -run TestTLSWajib
go test ./internal/app/notifikasi/... -run TestTemplateBerubahTanpaRilis
go run ./cmd/tools/cek-penerima --larang-akun-pribadi
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Tidak ada direktori `Correspondence/` di export | struktur direktori export · `R-16` |
| Jumlah Correspondence berbeda antar dokumen (3 vs 5) | `docs/Steering/06-MODULE-BREAKDOWN.md:62` · `docs/verifikasi-bukti-adr.md:2783` |
| 3 password SMTP di 31 lokasi, plaintext | `D-40` · `R-17` · `docs/Steering/16-RISK-ANALYSIS.md:484` |
| `UseSSL=false` pada 14 kemunculan; 16 dari 31 lokasi port 587 | `docs/Steering/00-DECISION-LOG.md:977` |
| Tidak ada penerima akun pribadi | `D-67` |

#### Comments

# 25. S-4 · Integrasi Sistem Luar

*Modul Pendukung · folder `docs/ticketing/S-4-Integrasi-Sistem-Luar/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Connect REST** (keluar) dan **Service REST** (masuk) · 6 DB Link (64 pemakaian) |
| **Kode modul** | `S-4` |
| **Gelombang** | 5 — Nilai dan pihak luar |
| **Ukuran** | 7 activity · **21 Connect REST** · 4 Service REST · 6 API pengganti DB Link |
| **Bergantung pada** | `F-1` Kerangka Aplikasi |
| **Kesiapan** | **TERHALANG** |

### Apa yang dikerjakan modul ini

Seluruh percakapan Claim PNC dengan sistem di luar dirinya — **dua arah**:

- **Keluar**: mengambil data premi, mengirim data bayar ke kasir, mengunggah dokumen, mengirim
  klaim ke ASO, menutup klaim Non-MBU, menetapkan DLA.
- **Masuk**: empat layanan yang **dipanggil sistem lain**, dua di antaranya **menerima persetujuan
  komite**.

### Temuan yang mengubah rencana

| Anggapan | Terverifikasi |
|---|---|
| 12 Connect REST keluar | **21 berkas** di `Connect REST/` — 12 yang terhitung sejak awal ditambah **9 yang baru ditemukan** |
| Integrasi hanya ke arah keluar | **4 Service REST masuk** — permukaan yang **belum pernah diaudit sama sekali** |
| Integrasi memakai autentikasi | **18 dari 21** Connect REST ber-`pyUseAuthentication=false` |

> **Angka 12 sudah dikoreksi menjadi 21** di `BRD §FR-S4`, `06-MODULE-BREAKDOWN.md:63`, dan
> `16-RISK-ANALYSIS.md:193` — disetujui Work Owner 2026-09-14 (`D-73`, `21-RIWAYAT-REVISI.md` §7).
> **Sembilan yang baru ditemukan belum dianalisis setara** dengan 12 yang lama.

### Kenapa modul ini TERHALANG

Yang menahan bukan kerumitan teknisnya, melainkan **tidak diketahuinya siapa lawan bicara yang
sah** — pada permukaan masuk maupun keluar.

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **6 API pengganti DB Link belum ada kontraknya** (64 pemakaian DB Link) | **Tim Infra + tim sistem lawan** (`D-25`, `R-03`) |
| **Keputusan** | **4 Service REST masuk — masuk lingkup migrasi?** Dua menerima **persetujuan komite** dari sistem lain | **Work Owner** |
| **Keputusan** | **18 dari 21 Connect REST tanpa autentikasi** — dibiarkan atau diperbaiki? | **Work Owner + Tim Infra/Security** (`R-18`) |
| **Keputusan** | `GetPremiumPaid_SPK` memakai **`http://` ke IP:port tanpa TLS** untuk data premi | **Tim Infra/Security** (`R-18`) |
| **Keputusan** | Endpoint BRI menunjuk **sandbox** — apakah itu yang berjalan di produksi? | **Work Owner + tim integrasi** (`R-18`) |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-S4-001](issues/01-klien-rest-keluar.md) | Klien REST keluar dan penanganan kegagalannya | `needs-info` |
| [TKT-S4-002](issues/02-layanan-rest-masuk.md) | Empat layanan REST masuk dan otentikasinya | `needs-info` |
| [TKT-S4-003](issues/03-pengganti-db-link.md) | Enam API pengganti DB Link | `needs-info` |

## Daftar Tiket

### TKT-S4-001 — Klien REST keluar dan penanganan kegagalannya

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan keamanan** |
| **Modul** | **S-4 Integrasi Sistem Luar** · Gelombang: 5 · Bergantung pada: TKT-F1-004 |
| **Requirement** | FR-S4 |
| **Keputusan** | D-25 |
| **ADR** | 0017 |
| **Risiko** | R-18 |
| **Rule Pega yang digantikan** | **21 Connect REST** di `Connect REST/` — antara lain `GetPremiumPaid_SPK`, `SendDataPaidASMtoCashier_2`, `SendKlaimToASO`, `ServiceCloseClaimNonMBU`, `ServiceSetDLA`, `UploadDokumenPNC`, `InjectDataRekeningToKasir` |
| **Peran penguji gerbang 2** | **PncAdmin** dan pemilik masing-masing integrasi |
| **Label** | `modul::S-4` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::5` |
| **Milestone** | Gelombang 5 — Nilai dan pihak luar |
| **Berkas sumber** | `ticketing/S-4-Integrasi-Sistem-Luar/issues/01-klien-rest-keluar.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Panggilan keluar ke sistem lain berjalan dari Go, dan **kegagalan sistem lain tidak menjatuhkan
Claim PNC**.

Nilai bisnisnya ada pada kalimat kedua. Sistem yang dipanggil dimiliki pihak lain dan akan mati
tanpa memberi tahu. Bila petugas tidak dapat menyimpan registrasi hanya karena layanan premi sedang
tidak menjawab, kegagalan pihak lain menjadi kegagalan kita.

#### Ruang lingkup

- Klien untuk **21 Connect REST**, dengan **batas waktu tunggu** dan **percobaan ulang** yang
  disetel per integrasi.
- Pemutus arus (circuit breaker) agar layanan yang sedang mati tidak dipanggil berulang.
- Pencatatan setiap panggilan: tujuan, hasil, lama.
- Kredensial dan endpoint diambil dari konfigurasi lingkungan, **bukan dari kode**.

#### Non-goal

- **Tidak** menggarap layanan masuk — itu `TKT-S4-002`.
- **Tidak** menggarap pengganti DB Link — itu `TKT-S4-003`.
- **Tidak** mengubah kontrak yang dipakai sistem lawan.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **18 dari 21 Connect REST ber-`pyUseAuthentication=false`** — dibawa apa adanya atau diperbaiki? | **Work Owner + Tim Infra/Security** (`R-18`) | Memperbaikinya menuntut sistem lawan ikut berubah. Membiarkannya berarti memindahkan kerentanan ke sistem baru dengan sengaja |
| **`GetPremiumPaid_SPK` memakai `http://` ke IP:port tanpa TLS** untuk data premi | **Tim Infra/Security** (`R-18`) | Data premi melintas jaringan tanpa enkripsi. Menyalakan TLS menuntut sisi lawan menyediakannya |
| **Endpoint BRI menunjuk sandbox** — itukah yang berjalan di produksi? | **Work Owner + tim integrasi** (`R-18`) | Bila ya, integrasi itu sebenarnya **tidak pernah hidup**, dan lingkupnya berubah |
| **Berapa lama batas tunggu dan berapa kali percobaan ulang per integrasi?** | **Work Owner + tim integrasi** | Percobaan ulang pada operasi yang mengubah data dapat **menggandakan transaksi** bila lawannya tidak idempoten |

#### Acceptance criteria

- ☐ Seluruh **21** integrasi keluar terpanggil dari Go, dan **jumlahnya dilaporkan sebagai angka**.
- ☐ **Sistem lawan mati tidak menjatuhkan Claim PNC** — diuji dengan lawan yang sengaja
      dimatikan: proses bisnis tetap dapat diselesaikan atau ditunda dengan pesan yang jelas.
- ☐ Batas waktu tunggu **selalu ada**; tidak ada panggilan yang menggantung tanpa batas — diuji
      dengan lawan yang sengaja lambat.
- ☐ Percobaan ulang **tidak dilakukan pada operasi yang mengubah data** kecuali lawannya
      dinyatakan idempoten — diperiksa per integrasi.
- ☐ Endpoint dan kredensial **tidak ada di dalam kode** — pemindaian otomatis: **nol temuan**.
- ☐ Setiap panggilan **tercatat**: tujuan, hasil, lama — diuji.
- ☐ Gerbang 2: UAT oleh **pemilik masing-masing integrasi**.

#### Dependency / Blocked by

`TKT-F1-004`. **Terhalang tiga keputusan keamanan (`R-18`).**

#### Constraint keamanan, data, operasional

- **`http://` tanpa TLS untuk data premi** adalah paparan nyata, bukan catatan teknis.
- **Membawa `pyUseAuthentication=false` apa adanya** berarti memindahkan kerentanan dengan sadar.
  Bila itu yang dipilih, ia harus **tercatat sebagai keputusan**, bukan terjadi karena tidak ada
  yang menanyakannya.
- Nilai kredensial dan alamat produksi **tidak boleh masuk dokumen yang di-commit**.
- Percobaan ulang yang salah pada pengiriman ke kasir dapat menyebabkan **pembayaran ganda**.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema klaim; menambah tabel catatan panggilan.

**Rollback:** mengembalikan versi klien. Panggilan yang sudah terkirim ke sistem lain
**tidak dapat ditarik** — rollback tidak membatalkan efeknya di sana.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/rest/... -run TestBatasWaktuTungguSelaluAda
go test ./internal/adapter/rest/... -run TestLawanMatiTidakMenjatuhkanAplikasi
go run ./cmd/tools/cek-integrasi --harap 21        # HARUS 21
go run ./cmd/tools/pindai-rahasia .                # HARUS nol temuan
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 21 berkas Connect REST | direktori `Connect REST/` (dihitung langsung) |
| 18 dari 21 ber-`pyUseAuthentication=false` | `Connect REST/*.xml` (dihitung langsung) · `R-18` |
| 9 Connect REST baru ditemukan setelah inventaris v1.0 | `docs/verifikasi-bukti-adr.md:2159` |
| `GetPremiumPaid_SPK` `http://` tanpa TLS; endpoint BRI sandbox | `docs/verifikasi-bukti-adr.md:2784` |
| Angka 12 dikoreksi menjadi 21 di Steering dan BRD | `D-73` · `docs/Steering/21-RIWAYAT-REVISI.md` §7 |

#### Comments

### TKT-S4-002 — Empat layanan REST masuk dan otentikasinya

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan lingkup** |
| **Modul** | **S-4 Integrasi Sistem Luar** · Gelombang: 5 · Bergantung pada: TKT-F1-004, TKT-B07-002 |
| **Requirement** | FR-S4 |
| **Keputusan** | D-25 |
| **ADR** | 0017, 0023 |
| **Risiko** | R-18 |
| **Rule Pega yang digantikan** | `Service REST/KomiteAcceptAdjustment-REST.xml` · `Service REST/KomiteAcceptAdjustmentPA-REST.xml` · `Service REST/RecivedDataandAttachmentLelangASMSimasbid-REST.xml` · `Service REST/RequestCreateClaimCredit2-REST.xml` |
| **Peran penguji gerbang 2** | **PncManagerAdmin** (komite) dan **PncAdmin** |
| **Label** | `modul::S-4` `tipe::keamanan` `status::needs-info` `prioritas::tinggi` `gelombang::5` |
| **Milestone** | Gelombang 5 — Nilai dan pihak luar |
| **Berkas sumber** | `ticketing/S-4-Integrasi-Sistem-Luar/issues/02-layanan-rest-masuk.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Empat pintu masuk yang selama ini ada tetapi **belum pernah diaudit** menjadi pintu yang diketahui:
siapa boleh mengetuk, apa yang boleh diminta, dan apa yang tercatat.

Nilai bisnisnya terletak pada dua dari empat layanan itu: **`KomiteAcceptAdjustment` dan
`KomiteAcceptAdjustmentPA` menerima persetujuan komite dari sistem lain.** Artinya keputusan
persetujuan klaim — inti kendali `B-7` — **dapat masuk dari luar aplikasi**. Siapa pun yang dapat
memanggil dua layanan itu dapat menyetujui klaim.

#### Ruang lingkup

- Empat endpoint masuk dengan kontrak yang **setara dengan yang berlaku sekarang**.
- **Otentikasi dan otorisasi pemanggil**, diperiksa di server (`ADR-0023`).
- Pencatatan setiap panggilan masuk: siapa memanggil, klaim mana, apa yang diubah, kapan —
  tersambung ke jejak audit `S-5`.
- Penegakan aturan komite yang sama seperti jalur layar: persetujuan lewat API **tidak boleh
  melewati** jenjang kumulatif `B-7`.

#### Non-goal

- **Tidak** menambah layanan masuk baru.
- **Tidak** mengubah kontrak yang sudah dipakai sistem lawan tanpa kesepakatan.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Keempat layanan masuk ini masuk lingkup migrasi?** | **Work Owner** | Permukaan ini **belum pernah masuk hitungan** `FR-S4` maupun `D-25`. Bila masuk, lingkup `S-4` bertambah; bila tidak, harus jelas apa yang terjadi pada pemanggilnya saat Pega dimatikan |
| **Siapa sistem pemanggil keempat layanan itu, dan atas dasar apa mereka berwenang?** | **Work Owner + tim integrasi** | Tanpa daftar pemanggil yang sah, otorisasi tidak dapat dirumuskan |
| **Persetujuan komite dari luar tunduk jenjang kumulatif yang sama?** | **Work Owner** (`D-52`, `D-70`) | Bila tidak, ada **dua aturan persetujuan berbeda** untuk klaim yang sama — dan yang satu memintas yang lain |
| **Layanan masuk memakai otentikasi apa sekarang?** | **Tim Infra/Security** (`R-18`) | Rule-nya ada di export, tetapi kelayakan mekanismenya belum dinilai |

#### Acceptance criteria

- ☐ Keempat endpoint masuk berfungsi dengan kontrak setara — diuji per endpoint.
- ☐ Panggilan **tanpa otentikasi ditolak** — diuji pada keempatnya: bukan `200`.
- ☐ Panggilan dari pemanggil yang tidak berwenang ditolak — diuji.
- ☐ **Persetujuan komite lewat API tunduk jenjang kumulatif yang sama dengan layar** — diuji:
      klaim yang belum cukup jenjangnya **tidak berpindah status** walau API memanggilnya.
- ☐ Setiap panggilan masuk **tercatat di jejak audit** dengan identitas pemanggil — diuji.
- ☐ Panggilan berulang dengan isi sama **tidak menggandakan efeknya** — diuji.
- ☐ Gerbang 2: UAT **PncManagerAdmin** untuk dua layanan komite, terpisah dari dua lainnya.

#### Dependency / Blocked by

`TKT-F1-004` · `TKT-B07-002` (aturan jenjang komite) · `TKT-S5-001` (jejak audit).
**Terhalang keputusan lingkup Work Owner.**

#### Constraint keamanan, data, operasional

- **Dua layanan ini dapat menyetujui klaim dari luar aplikasi.** Ini permukaan dengan dampak
  finansial langsung, dan ia **belum pernah diaudit**.
- Otorisasi wajib diperiksa **di server** (`D-59`, `ADR-0023`) — tidak boleh bersandar pada
  anggapan bahwa hanya sistem internal yang tahu alamatnya.
- Selama Pega dan Go berjalan berdampingan (`D-05`), **kedua sistem dapat menerima panggilan yang
  sama**. Aturan penulis tunggal (`P-1`) harus tetap berlaku pada tabel yang mereka ubah.

#### Migrasi skema / rollout / rollback

Tidak mengubah skema; memakai tabel klaim dan jejak audit yang ada.

**Rollout:** pengalihan pemanggil dari Pega ke Go menuntut **sistem lawan mengubah alamat** —
ini koordinasi dengan pihak luar, bukan pekerjaan sepihak.

**Rollback:** mengembalikan alamat ke Pega. Persetujuan yang sudah masuk lewat Go **tetap ada** di
data — rollback tidak membatalkannya.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/http/inbound/... -run TestTanpaOtentikasiDitolak
go test ./internal/adapter/http/inbound/... -run TestKomiteAPITundukJenjangKumulatif
go test ./internal/adapter/http/inbound/... -run TestPanggilanBerulangTidakMenggandakan
go test ./internal/app/audit/... -run TestPanggilanMasukTercatat
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Empat layanan REST masuk | direktori `Service REST/` (4 berkas) |
| Dua di antaranya menerima persetujuan komite | `docs/verifikasi-bukti-adr.md:2707-2715` |
| Permukaan masuk belum pernah masuk hitungan `FR-S4`/`D-25` | `docs/verifikasi-bukti-adr.md:2715` |
| Jenjang komite kumulatif; pita nilai hanya Non-MBU | `D-52` · `D-70` |
| Otorisasi diperiksa di server | `D-59` · `ADR-0023` |

#### Comments

### TKT-S4-003 — Enam API pengganti DB Link

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak — kontrak API belum ada** |
| **Modul** | **S-4 Integrasi Sistem Luar** · Gelombang: 5 · Bergantung pada: TKT-S4-001 |
| **Requirement** | FR-S4 |
| **Keputusan** | D-25 |
| **ADR** | 0017 |
| **Risiko** | R-03, R-19 |
| **Rule Pega yang digantikan** | **64 pemakaian DB Link** ke 6 database lain — `@ASMD` (55×), `@SIMASNET` (3×), `@SMI` (2×), `@OPJAVA` (2×), `@PROD_ASM` (1×), `@PROD_TKA` (1×) |
| **Peran penguji gerbang 2** | **PncAdmin** dan pemilik masing-masing sistem sumber |
| **Label** | `modul::S-4` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::5` |
| **Milestone** | Gelombang 5 — Nilai dan pihak luar |
| **Berkas sumber** | `ticketing/S-4-Integrasi-Sistem-Luar/issues/03-pengganti-db-link.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Data dari enam database lain diambil lewat **kontrak yang eksplisit**, bukan lewat sambungan
langsung antar database.

Nilai bisnisnya: PostgreSQL tidak punya DB Link, jadi ini bukan pilihan gaya. Tetapi nilai yang
sebenarnya adalah **hilangnya kopling tersembunyi** — selama ini enam sistem lain dapat mengubah
tabelnya dan memecahkan Claim PNC tanpa ada yang tahu sampai kejadian.

#### Ruang lingkup

- Enam klien API menggantikan 64 pemakaian DB Link, sesuai `D-25`.
- **Penulisan ulang aturan bisnis yang selama ini bersembunyi di objek remote** — `GET_WORKING_HOURS`
  (17×) dan `HRD_LBR` adalah **perhitungan jam kerja dan kalender libur**, yaitu aturan bisnis;
  keduanya ditulis ulang di Go, **bukan dipanggil lewat API** (`D-…` konsekuensi `R-19`).
- Perilaku saat sistem sumber tidak menjawab.

#### Non-goal

- **Tidak** membangun API di sisi sistem pemilik data — itu pekerjaan tim mereka.
- **Tidak** memakai Foreign Data Wrapper maupun replikasi; keduanya ditolak di `D-25`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Keenam API kemungkinan besar belum ada** — belum ada satu pun kontrak | **Tim pemilik masing-masing sistem** (`R-03`) | Ini bukan pekerjaan yang dapat diselesaikan sepihak. Setiap satu menuntut kesepakatan dengan tim lain, dan **jadwalnya bukan milik proyek ini** |
| **Apa yang terjadi bila sistem sumber tidak menjawab** — klaim ditahan, atau lanjut dengan data kosong? | **Work Owner** | Keduanya sah, tetapi akibatnya sangat berbeda bagi petugas di cabang |
| **Data mana yang boleh disimpan sementara (cache) dan berapa lama?** | **Work Owner + pemilik data** | Data polis dan HRD yang basi dapat menghasilkan keputusan klaim yang salah |

#### Acceptance criteria

- ☐ **Nol pemakaian DB Link** tersisa di kode Go — dicari secara otomatis.
- ☐ Keenam sumber data terlayani, dan **jumlahnya dilaporkan sebagai angka**.
- ☐ **`GET_WORKING_HOURS` dan kalender libur berjalan sebagai aturan di Go**, bukan panggilan
      remote — diuji terhadap kasus yang sama dengan `TKT-F5-001`.
- ☐ Sistem sumber tidak menjawab **ditangani sesuai keputusan Work Owner**, dan perilakunya
      **terlihat oleh petugas** — bukan diam.
- ☐ Hasil pengambilan data **sama dengan hasil DB Link** pada 20 kasus per sumber — dibandingkan
      lewat `S-8`.
- ☐ Gerbang 2: UAT per **pemilik sistem sumber**.

#### Dependency / Blocked by

`TKT-S4-001` · `TKT-F5-001` (jam kerja & hari libur). **Terhalang tim pemilik sistem (`R-03`) —
penghalang di luar kendali proyek.**

#### Constraint keamanan, data, operasional

- **`R-03` adalah penghalang yang bergantung pada pihak lain.** Jadwal `S-4` tidak dapat dijanjikan
  selama kontrak keenam API belum ada — menuliskan tanggal untuk pekerjaan ini akan menyesatkan.
- Data HRD dan polis adalah **data pribadi**; kontrak API harus membatasi apa yang boleh diambil,
  bukan menyalin akses tabel penuh yang selama ini dimiliki DB Link.
- Selama peralihan, Pega masih memakai DB Link sementara Go memakai API — **dua jalur ke data yang
  sama**, dan selisihnya harus terpantau (`P-1`, `S-8`).

#### Migrasi skema / rollout / rollback

Tidak menambah tabel. Bila cache disepakati, ia tabel baru dan backward-compatible (`P-4`).

**Rollback:** **tidak dapat kembali ke DB Link di PostgreSQL** — mekanismenya tidak ada di sana.
Rollback yang tersedia hanya mengembalikan lalu lintas ke Pega.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/cari-dblink ./internal/...      # HARUS nol temuan
go test ./internal/app/jamkerja/... -run TestGetWorkingHoursDiGo
go run ./cmd/s8 banding --modul S-4 --kasus 20 --per-sumber
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 64 pemakaian DB Link ke 6 database | `D-25` tabel inventaris · `docs/Steering/00-DECISION-LOG.md:605-612` |
| `@ASMD` 55×, `GET_WORKING_HOURS` 17× | idem |
| API pengganti kemungkinan belum ada | `D-25` · `R-03` |
| `GET_WORKING_HOURS` dan `HRD_LBR` ditulis ulang di Go, bukan dipanggil | `docs/Steering/00-DECISION-LOG.md:1326` · `R-19` |
| FDW dan replikasi ditolak | `D-25` pilihan 2 dan 3 tidak dipilih |

#### Comments

# 26. S-5 — Jejak Audit

*Modul Pendukung · folder `docs/ticketing/S-5-Jejak-Audit-Perubahan/`*

## Spesifikasi Modul

| | |
|---|---|
| **Modul** | `S-5` Jejak Audit |
| **Gelombang** | 2 |
| **Ukuran** | Sedang — **kemampuan baru 100%, tanpa baseline Pega** |
| **Bergantung pada** | `F-2` |
| **Kesiapan** | **SEBAGIAN** — lingkup jelas, daftar peristiwa `needs-info` |
| **Cakupan tiket** | **penuh** (`D-41` Opsi 1) |

### Apa yang dibangun

Pencatatan permanen setiap perubahan bernilai bisnis: **siapa, kapan, nilai sebelum, nilai
sesudah** — bersifat append-only dan tidak dapat diubah oleh jalur aplikasi mana pun.

### Kenapa modul ini naik derajat

Dua hal terjadi bersamaan:

1. **Sistem lama tidak punya jejak audit atas nilai uang klaim sama sekali** (`T-14`). Yang paling
   mendekati, `Database/PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc:7`, hanya mencatat **empat kolom**.
   Dan dua tabel yang namanya log **terbukti dimutasi** — `UPDATE` pada `claim_service_log`,
   `DELETE` pada `JSON_KLAIM_LOG`.
2. **`D-59` menghapus pemisahan tugas.** Satuan izin adalah menu; seorang pengguna yang memiliki
   tiga menu dapat membuat, menyetujui, dan membayarkan satu klaim. Tidak ada kontrol teknis yang
   mencegahnya.

Akibatnya `S-5` bukan modul pendukung, melainkan **satu-satunya kontrol pengimbang yang tersisa**.

### Konsekuensi: tidak ada yang bisa dibandingkan

Karena tidak ada baseline, `S-5` **tidak dapat melewati gerbang 1** dalam bentuk uji kesetaraan.
`D-56` menggantinya dengan **uji fungsional terhadap kontrak** — dan kontrak itu adalah **daftar
peristiwa wajib audit dari Compliance**, yang **belum ada**.

### Yang mengikat modul ini

| Sumber | Isi |
|---|---|
| `ADR-0026` | Append-only, ditegakkan hak akses database · retensi mengikuti retensi data klaim |
| `ADR-0012` | Soft delete menyeluruh — tidak ada penghapusan fisik |
| `ADR-0023` | Jejak audit satu-satunya kontrol pengimbang |
| `ADR-0028` | Gerbang 1 diganti uji fungsional terhadap kontrak |
| `D-62` | Retensi mengikuti retensi data klaim; **angkanya belum ada**, dibuat sebagai parameter |

### Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-S5-001](issues/01-skema-jejak-audit-append-only.md) | Skema jejak audit append-only dan hak akses database | `ready-for-human` | siap |
| [TKT-S5-002](issues/02-pencatatan-otomatis-perubahan.md) | Pencatatan otomatis pada perubahan bernilai bisnis | `ready-for-human` | siap |
| [TKT-S5-003](issues/03-daftar-peristiwa-wajib-audit.md) | Daftar peristiwa wajib audit | `needs-info` | **terhalang Compliance** |
| [TKT-S5-004](issues/04-retensi-sebagai-parameter.md) | Retensi dan arsip sebagai parameter konfigurasi | `ready-for-human` | siap |

## Daftar Tiket

### TKT-S5-001 — Skema jejak audit append-only dan hak akses database

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | S-5 · Gelombang: 2 · Bergantung pada: TKT-F2-001, TKT-F2-004 |
| **Requirement** | FR-S5 |
| **Keputusan** | D-28, D-62 |
| **ADR** | 0026 |
| **Risiko** | R-14 |
| **Rule Pega yang digantikan** | **tidak ada padanan** — sistem lama tidak mencatat perubahan nilai. Yang paling mendekati: `Database/PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc:7` (**hanya 4 kolom**). Anti-pola yang tidak dibawa: `RDB List/UpdateLogServiceClaim-SQL.xml:27` (`UPDATE` pada log), `RDB List/InsertClaimPNC-SQL.xml:77` (`DELETE` pada log) |
| **Peran penguji gerbang 2** | **tidak berlaku** — `D-60`; gerbang 1 diganti uji fungsional terhadap kontrak (`D-56`) |
| **Label** | `modul::S-5` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::2` |
| **Milestone** | Gelombang 2 — Kerangka UI |
| **Berkas sumber** | `ticketing/S-5-Jejak-Audit-Perubahan/issues/01-skema-jejak-audit-append-only.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Tabel jejak audit yang **secara struktural tidak dapat diubah atau dihapus** oleh aplikasi, dengan
kolom yang cukup untuk menjawab *"siapa mengubah nilai ini, dari berapa menjadi berapa, kapan"*.

Nilai bisnisnya: pertanyaan itu **tidak dapat dijawab hari ini**. Dan karena `D-59` menghapus
pemisahan tugas, ia satu-satunya kontrol yang tersisa atas kewenangan menyetujui uang.

#### Ruang lingkup

- Skema tabel jejak audit: entitas, id entitas, jenis aksi, nilai sebelum, nilai sesudah, pelaku,
  waktu (UTC), dan id permintaan (`TKT-F1-003`).
- **Penegakan append-only di tingkat hak akses database**: akun aplikasi hanya diberi `INSERT`
  dan `SELECT` pada tabel audit — tanpa `UPDATE` maupun `DELETE`.
- Index yang mendukung pertanyaan nyata: per entitas+id, per pelaku, per rentang waktu.
- Catatan kapasitas dan kandidat partisi per tahun, **disiapkan tetapi belum diterapkan**.

#### Non-goal

- **Tidak** menentukan peristiwa apa saja yang wajib dicatat — itu `TKT-S5-003`, menunggu
  Compliance.
- **Tidak** membangun layar penelusuran audit — belum diputuskan apakah pengguna bisnis perlu
  melihatnya (`ADR-0026` pertanyaan terbuka).
- **Tidak** menerapkan partisi di awal.

#### Acceptance criteria

- ☐ `UPDATE` pada tabel audit memakai akun aplikasi **ditolak database** — diuji langsung,
      bukan diasumsikan dari kode.
- ☐ `DELETE` pada tabel audit memakai akun aplikasi **ditolak database** — diuji.
- ☐ Satu baris audit memuat kedelapan kolom wajib, dan **nilai sebelum serta sesudah tidak
      boleh keduanya kosong** — ditegakkan constraint, diuji.
- ☐ `request_id` pada baris audit **sama** dengan `request_id` di log untuk permintaan yang sama
      — diuji.
- ☐ Kueri "seluruh perubahan pada klaim X" dan "seluruh perubahan oleh pelaku Y dalam rentang
      tanggal" **memakai index** — dilampirkan keluaran `EXPLAIN PLAN` keduanya.
- ☐ Migrasi skema tabel audit backward-compatible dan punya `down.sql` yang berfungsi
      (`TKT-F2-004`).

#### Dependency / Blocked by

Bergantung pada `TKT-F2-001` dan `TKT-F2-004`. Membutuhkan **DBA** untuk menetapkan hak akses
akun aplikasi — itu bagian prosedur `D-63`.

#### Constraint keamanan, data, operasional

- **Append-only ditegakkan hak akses database, bukan hanya kode.** Aturan yang hanya ada di kode
  bisa dilanggar oleh kode berikutnya; aturan yang ada di hak akses tidak.
- Nilai sebelum dan sesudah dapat memuat **data nasabah**. Tabel audit karena itu tunduk pada
  pembatasan akses yang sama dengan data aslinya — termasuk pembatasan data medis (`FR-R2`).
- Tabel ini **tumbuh paling cepat** di seluruh basis data (`D-10`).

#### Migrasi skema / rollout / rollback

Menambah tabel baru — **tidak menyentuh tabel mana pun yang dibaca Pega**, sehingga aman bagi
masa paralel.

**Rollback:** tabel dibiarkan ada dan tidak ditulisi. **Tabel audit tidak pernah di-`DROP` sebagai
rollback** — itu menghapus bukti, dan bertentangan dengan tujuan modul ini.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
migrate -path db/migrations -database "$DSN" up
### uji hak akses dengan akun aplikasi
psql/sqlplus -U app -c "UPDATE jejak_audit SET pelaku='x' WHERE id=1;"   # HARUS ditolak
psql/sqlplus -U app -c "DELETE FROM jejak_audit WHERE id=1;"             # HARUS ditolak
go test ./internal/adapter/sqlstore/... -run TestJejakAuditAppendOnly
EXPLAIN PLAN FOR SELECT * FROM jejak_audit WHERE entitas='klaim' AND entitas_id=:1;
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Sistem lama tidak punya jejak audit atas nilai | `T-14` · `docs/verifikasi-bukti-adr.md` §10.8 |
| History hanya 4 kolom | `Database/PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc:7` |
| Tabel log terbukti dimutasi | `RDB List/UpdateLogServiceClaim-SQL.xml:27` · `RDB List/InsertClaimPNC-SQL.xml:77` |
| Append-only ditegakkan hak akses DB | `docs/Steering/09-DATABASE-STRATEGY.md` §8 · `ADR-0026` |
| Jejak audit satu-satunya kontrol pengimbang | `D-59` · `ADR-0023` |

#### Comments

### TKT-S5-002 — Pencatatan otomatis pada perubahan bernilai bisnis

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | S-5 · Gelombang: 2 · Bergantung pada: TKT-S5-001, TKT-F2-003 |
| **Requirement** | FR-S5 |
| **Keputusan** | D-28, D-59 |
| **ADR** | 0023, 0026 |
| **Risiko** | R-14 |
| **Rule Pega yang digantikan** | **tidak ada padanan** — pencatatan di sistem lama bersifat sebagian dan dapat dimutasi |
| **Peran penguji gerbang 2** | **tidak berlaku** — `D-60` |
| **Label** | `modul::S-5` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::2` |
| **Milestone** | Gelombang 2 — Kerangka UI |
| **Berkas sumber** | `ticketing/S-5-Jejak-Audit-Perubahan/issues/02-pencatatan-otomatis-perubahan.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Mekanisme yang membuat pencatatan audit **terjadi karena strukturnya**, bukan karena penulis kode
ingat menuliskannya.

Nilai bisnisnya sederhana dan keras: kontrol yang bergantung pada ingatan **akan terlewat**. Dan
di sistem ini, yang terlewat berarti perubahan nilai uang klaim yang tidak dapat
dipertanggungjawabkan — pada aplikasi yang **tidak punya pemisahan tugas** (`D-59`).

#### Ruang lingkup

- Titik pencatatan yang menyatu dengan mekanisme transaksi (`TKT-F2-003`): baris audit ditulis
  **di dalam transaksi yang sama** dengan perubahannya, sehingga keduanya berhasil bersama atau
  gagal bersama.
- Cara menyatakan "entitas ini diaudit" **sekali di satu tempat**, bukan di setiap pemanggil.
- Perhitungan **nilai sebelum dan sesudah** secara otomatis dari perubahan yang terjadi.
- Pemeriksaan otomatis: perubahan pada entitas yang ditandai diaudit **tanpa** baris audit
  menggagalkan uji.

#### Non-goal

- **Tidak** menentukan daftar entitas dan peristiwa yang diaudit — itu `TKT-S5-003`. Tiket ini
  menyediakan mekanismenya, dan daftar sementara memakai **daftar minimum** `ADR-0026`.
- **Tidak** mencatat pembacaan data — hanya perubahan.

#### Acceptance criteria

- ☐ Perubahan nilai pada entitas yang ditandai diaudit **selalu** menghasilkan tepat satu baris
      audit — diuji pada 5 jenis perubahan berbeda.
- ☐ Baris audit ditulis **di dalam transaksi yang sama**: transaksi yang gagal **tidak
      meninggalkan baris audit** — diuji dengan kegagalan yang sengaja dipicu.
- ☐ Perubahan yang gagal ditulis auditnya **menggagalkan seluruh transaksi** — audit tidak boleh
      "best effort".
- ☐ Nilai sebelum dan sesudah terisi benar untuk perubahan sebagian — diuji dengan mengubah satu
      field dari tiga.
- ☐ Uji otomatis **gagal** bila ada entitas bertanda diaudit yang jalur perubahannya tidak
      mencatat — diuji dengan entitas percobaan.
- ☐ Daftar minimum `ADR-0026` tercakup: nilai estimasi, nilai settlement, akseptasi, keputusan
      komite, penolakan, proses ulang, perubahan status klaim, dan pembayaran.

#### Dependency / Blocked by

Bergantung pada `TKT-S5-001` dan `TKT-F2-003`.

**Catatan urutan:** tiket ini boleh selesai sebelum `TKT-S5-003`. Bila Compliance kelak menambah
peristiwa, yang bertambah adalah **daftar**, bukan mekanismenya.

#### Constraint keamanan, data, operasional

- Audit **tidak boleh** dapat dimatikan lewat konfigurasi. Bila sebuah lingkungan perlu
  mematikannya, itu keputusan yang menuntut perubahan kode dan review — bukan sakelar.
- Pencatatan menambah satu operasi tulis pada setiap perubahan bernilai bisnis; dampaknya pada
  jalur transaksi **diukur**, bukan diasumsikan kecil.
- Baris audit memuat data nasabah pada kolom nilai — tunduk pembatasan akses yang sama.

#### Migrasi skema / rollout / rollback

Tidak menambah skema di luar `TKT-S5-001`.

**Rollback:** menonaktifkan pencatatan **tidak tersedia** — lihat constraint di atas. Rollback yang
sah adalah mengembalikan versi kode sebelumnya secara keseluruhan.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/... -run TestAuditTercatat
go test ./internal/app/... -run TestAuditIkutGagalSaatTransaksiGagal
go test ./internal/app/... -run TestEntitasDiauditTanpaPencatatanGagal
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Daftar minimum perubahan yang wajib diaudit | `D-28` · `ADR-0026` |
| Tidak ada pemisahan tugas; audit satu-satunya kontrol | `D-59` · `ADR-0023` |
| Kriteria penerimaan #9 wajib tanpa pengecualian | `BRD §21.2` |
| Kepemilikan transaksi di lapisan aplikasi | `ADR-0007` · `TKT-F2-003` |

#### Comments

### TKT-S5-003 — Daftar peristiwa wajib audit

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak** — kontrak dari Compliance belum ada |
| **Modul** | S-5 · Gelombang: 2 · Bergantung pada: TKT-S5-002 |
| **Requirement** | FR-S5 |
| **Keputusan** | D-28, D-56, D-62 |
| **ADR** | 0026, 0028 |
| **Risiko** | R-14 |
| **Rule Pega yang digantikan** | **tidak ada** — sistem lama tidak punya daftar seperti ini |
| **Peran penguji gerbang 2** | **tidak berlaku** — `D-60` |
| **Label** | `modul::S-5` `tipe::kepatuhan` `status::needs-info` `prioritas::tinggi` `gelombang::2` |
| **Milestone** | Gelombang 2 — Kerangka UI |
| **Berkas sumber** | `ticketing/S-5-Jejak-Audit-Perubahan/issues/03-daftar-peristiwa-wajib-audit.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Daftar resmi peristiwa yang **wajib** masuk jejak audit beserta field yang harus tercatat pada
masing-masing — disepakati Compliance, bukan disusun tim teknis.

Nilai bisnisnya melampaui kelengkapan. `D-56` menetapkan daftar ini **menggantikan gerbang 1**
bagi `S-5`, karena tidak ada baseline Pega untuk dibandingkan. Artinya: **tanpa daftar ini, `S-5`
tidak dapat dinyatakan lulus gerbang apa pun**, sekalipun kodenya selesai dan bekerja.

#### Ruang lingkup

- Daftar peristiwa wajib audit, per entitas, beserta field yang harus tercatat.
- Pemetaan setiap peristiwa ke **titik kode** tempat ia terjadi.
- Uji fungsional yang memeriksa daftar itu terpenuhi seluruhnya — inilah pengganti gerbang 1.

#### Non-goal

- **Tidak** menyusun daftarnya sendiri. Tim teknis dapat **mengusulkan**, tetapi yang mengikat
  adalah kesepakatan Compliance.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Daftar peristiwa wajib audit beserta field yang harus tercatat** | **Compliance** | Ia adalah **kontrak pengganti gerbang 1** (`D-56`). Tanpanya, tidak ada ukuran kelulusan |
| **Angka retensi data klaim yang berlaku sekarang** | **Work Owner + Compliance** | `D-62` menetapkan retensi audit mengikuti retensi data klaim, dan kebijakannya sudah ada — yang belum ada **angkanya**. Ini tidak menahan pembangunan (`TKT-S5-004` membuatnya parameter), tetapi menahan **go-live** |
| **Data historis tidak punya kolom nilai sebelum dan sesudah** — apakah diterima apa adanya? | **Compliance** | Menentukan apakah jejak audit dimulai dari nol pada tanggal cutover, atau menuntut rekonstruksi retrospektif yang **tidak mungkin dilakukan** dari data yang ada |

**Usulan awal yang dapat dipakai sebagai bahan pembicaraan** — bukan keputusan: daftar minimum
`ADR-0026` (nilai estimasi klaim, nilai settlement, akseptasi, keputusan komite, penolakan,
proses ulang, perubahan status klaim, pembayaran), ditambah tiga kandidat yang muncul dari analisis
— perubahan master ambang komite, perubahan master penerima notifikasi, dan perubahan peran
pengguna. Ketiganya diusulkan karena `D-59` menghapus pemisahan tugas, sehingga **perubahan pada
master yang menentukan kewenangan** menjadi tindakan bernilai tinggi.

#### Acceptance criteria

> Tidak dapat ditulis dengan angka sampai daftarnya diterima. Butir di bawah adalah **bentuk** AC
> yang akan diisi, bukan AC final.

- ☐ Setiap peristiwa dalam daftar Compliance punya **uji fungsional** yang membuktikan ia
      tercatat beserta seluruh field yang diminta.
- ☐ Jumlah peristiwa yang diuji **sama dengan** jumlah peristiwa dalam daftar — dilaporkan
      sebagai angka, bukan pernyataan.
- ☐ Peristiwa yang ada di kode tetapi **tidak** ada di daftar dilaporkan sebagai selisih, dan
      diputuskan satu per satu: ditambahkan ke daftar, atau dihentikan pencatatannya.

#### Dependency / Blocked by

Bergantung pada `TKT-S5-002`. **Terhalang Compliance.**

**Yang terhalang olehnya:** kelulusan seluruh modul `S-5`.

#### Constraint keamanan, data, operasional

- Daftar ini **mengikat**; menambah atau mengurangi isinya kelak adalah perubahan yang menuntut
  persetujuan Compliance, bukan keputusan teknis.
- `BRD §21.2` kriteria #9 berlaku **tanpa pengecualian** pada seluruh modul bisnis karena `D-59` —
  daftar ini yang menjabarkannya.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema. Bila daftar menuntut field yang belum ada di tabel `TKT-S5-001`,
penambahannya menempuh `TKT-F2-004` (backward-compatible, dua tahap).

#### Rencana verifikasi

> **Rencana — belum dijalankan, dan belum dapat disusun lengkap.**

```bash
go test ./internal/app/... -run TestPeristiwaWajibAudit
go run ./cmd/tools/banding-daftar-audit daftar-compliance.yaml   # selisih kode vs daftar
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Daftar peristiwa menggantikan gerbang 1 bagi `S-5` | `D-56` · `ADR-0028` |
| Daftar minimum yang wajib diaudit | `D-28` · `ADR-0026` |
| Retensi mengikuti retensi data klaim; angkanya belum ada | `D-62` |
| Sistem lama tidak punya jejak audit atas nilai | `T-14` |
| Audit satu-satunya kontrol pengimbang | `D-59` · `ADR-0023` |

#### Comments

### TKT-S5-004 — Retensi dan arsip sebagai parameter konfigurasi

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | S-5 · Gelombang: 2 · Bergantung pada: TKT-S5-001 |
| **Requirement** | FR-S5 |
| **Keputusan** | D-62, D-15 |
| **ADR** | 0026, 0012 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | **tidak ada** — sistem lama tidak punya kebijakan retensi yang tercatat di rule mana pun |
| **Peran penguji gerbang 2** | **tidak berlaku** — `D-60` |
| **Label** | `modul::S-5` `tipe::fondasi` `status::ready-for-human` `prioritas::sedang` `gelombang::2` |
| **Milestone** | Gelombang 2 — Kerangka UI |
| **Berkas sumber** | `ticketing/S-5-Jejak-Audit-Perubahan/issues/04-retensi-sebagai-parameter.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Retensi data audit yang **dapat diisi angkanya kelak tanpa mengubah kode**, sehingga `S-5` dapat
dibangun sekarang meski angkanya belum ada.

Nilai bisnisnya adalah menghilangkan penghalang semu. `D-62` sudah menyelesaikan pertanyaan
kebijakannya — retensi audit **mengikuti retensi data klaim yang berlaku sekarang**, satu
kebijakan untuk keduanya. Yang belum ada hanyalah **angkanya**, dan angka tidak boleh menahan
pembangunan modul.

#### Ruang lingkup

- Parameter retensi sebagai konfigurasi (`TKT-F1-002`), **bukan konstanta di kode** — konsisten
  `D-15`.
- Perhitungan "baris audit ini sudah melewati masa retensi" sebagai fungsi murni yang dapat diuji.
- **Mekanisme arsip yang belum dijalankan**: kode ada, terjadwal tidak aktif, dan aktivasinya
  menuntut angka retensi serta persetujuan tertulis.
- Laporan ukuran: berapa baris dan berapa besar tabel audit per periode — supaya keputusan arsip
  kelak diambil dengan angka.

#### Non-goal

- **Tidak** menghapus data apa pun. Arsip tidak dijalankan sampai angkanya ada dan disetujui.
- **Tidak** menetapkan angka retensi — itu Work Owner dan Compliance.

#### Acceptance criteria

- ☐ Parameter retensi dibaca dari konfigurasi; **nilai kosong berarti tidak ada yang
      diarsipkan** — diuji, dan itulah nilai bawaannya.
- ☐ Fungsi "sudah lewat retensi" diuji pada batas: tepat di hari terakhir, satu hari sesudah,
      dan satu hari sebelum.
- ☐ Proses arsip **tidak berjalan** selama parameter kosong — diuji dengan menjalankan
      penjadwal dan memeriksa nol baris tersentuh.
- ☐ Menjalankan arsip menuntut **penanda persetujuan eksplisit** di konfigurasi; tanpa itu ia
      menolak berjalan walau parameter retensi terisi — diuji.
- ☐ Laporan ukuran tabel audit dapat dihasilkan dan memuat jumlah baris per bulan.
- ☐ Arsip **memindahkan**, tidak menghapus — baris yang diarsipkan tetap dapat ditemukan.

#### Dependency / Blocked by

Bergantung pada `TKT-S5-001` dan `TKT-F1-002`.

**Tidak terhalang** meski angka retensi belum ada — itulah inti tiket ini.

#### Constraint keamanan, data, operasional

- **Arsip bukan penghapusan.** `ADR-0012` menetapkan tidak ada penghapusan fisik data bernilai
  bisnis, dan jejak audit adalah kelas data yang paling tidak boleh hilang.
- Menjalankan arsip pada data audit menyentuh **bukti**. Karena itu ia menuntut penanda persetujuan
  terpisah dari sekadar parameter terisi.
- Tabel audit **tumbuh paling cepat** (`D-10`); laporan ukuran adalah cara mengetahui kapan
  keputusan arsip benar-benar mendesak.

#### Migrasi skema / rollout / rollback

Menambah tabel arsip (bila arsip memindahkan baris) — tabel baru, tidak menyentuh tabel Pega.

**Rollback:** mengosongkan parameter retensi menghentikan arsip seketika. Baris yang telanjur
diarsipkan **tetap ada** dan dapat dikembalikan, karena arsip memindahkan, bukan menghapus.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/audit/... -run TestRetensiBatas
go test ./internal/app/audit/... -run TestArsipTidakBerjalanTanpaParameter
go test ./internal/app/audit/... -run TestArsipMenolakTanpaPersetujuan
go run ./cmd/tools/laporan-ukuran-audit
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Retensi audit mengikuti retensi data klaim; angkanya belum ada | `D-62` |
| Retensi dibangun sebagai parameter agar `S-5` tidak terhalang | `D-62` |
| Tidak ada penghapusan fisik data bernilai bisnis | `D-66` · `ADR-0012` |
| Nilai bisnis tidak boleh di-hardcode | `D-15` · `ADR-0025` |
| Tabel audit tumbuh paling cepat | `docs/Steering/09-DATABASE-STRATEGY.md` §8 |

#### Comments

# 27. S-6 · Job Terjadwal & Proses Otomatis

*Modul Pendukung · folder `docs/ticketing/S-6-Job-Terjadwal-dan-Proses-Otomatis/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Job Scheduler** (5 job) dan **Agent** (1) — `Job Scheduler/`, `Agents/TATReportAgent-Agents.xml` |
| **Kode modul** | `S-6` |
| **Gelombang** | 7 — Sisa |
| **Ukuran** | 5 job · 1 agent · 8 activity target |
| **Bergantung pada** | `F-1` Kerangka Aplikasi |
| **Kesiapan** | **SEBAGIAN** |

### Apa yang dikerjakan modul ini

Lima pekerjaan yang berjalan sendiri pada jam tertentu, tanpa ada orang yang menekan tombol,
ditambah satu agent yang berjalan tiap 30 menit.

| Job | Frekuensi | Jam | Yang dikerjakan |
|---|---|---|---|
| `JOBForKomiteKlaimPNC` | Harian | **06:00** | **Menyetujui komite secara otomatis** (`AutoAcceptKomite`) |
| `JobHitungDeadlineToTemporaryCLose` | **Mingguan** | 23:43 | Menutup sementara klaim yang lewat tenggat |
| `JobSendAutoLODKlaimPersonal` | Harian | 20:54 | Mengirim LOD klaim personal otomatis |
| `PNCMyReportKlaim3` | Harian | 08:00 | Laporan AI |
| `ProcessClaimKredit` | Harian | 10:00 | Membuat klaim kredit |
| `TATReportAgent` (agent) | tiap **30 menit** | — | Alert kasir belum transfer · pengalihan kasus tak tertugaskan |

Seluruhnya `pyIsEnabled=true`, `pyApplicableTo=Cluster` (`D-57`).

### Temuan yang mengubah rencana

| Anggapan | Terverifikasi |
|---|---|
| Daftar job tidak diketahui (`R-02`) | **Lengkap dan terverifikasi** — `R-02` tertutup pada tingkat artefak (`D-57`) |
| Persetujuan komite selalu dilakukan manusia | **`AutoAcceptKomite` berjalan tiap hari jam 06:00 tanpa pengguna sama sekali** |
| Agent berjalan dengan identitas pengguna | **`pyBypassActivityAuthentication=true`** — agent **melewati pemeriksaan otentikasi** |

### Kenapa modul ini SEBAGIAN, bukan SIAP

Artefaknya lengkap. Yang belum ada adalah **jawaban atas apa yang artefak itu ungkapkan** — terutama
bahwa ada jalur persetujuan klaim yang **tidak melewati kontrol mana pun**.

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | **`AutoAcceptKomite` menyetujui komite otomatis tiap hari 06:00** — benar demikian? Perilaku ini **tidak tercatat di dokumen mana pun** | **Work Owner** (`D-57` butir 1) |
| **Keputusan** | Tiga job ber-`pyDescription = "job jalan 2 menit"` sementara konfigurasinya `Daily`/`Weekly` — **selisih 720× sampai 5.040×**. Mana yang benar? | **Work Owner + Tim Pega** (`D-57` butir 2) |
| **Keputusan** | **`pyBypassActivityAuthentication=true`** pada agent — diterima untuk sistem baru? | **Work Owner + Tim Infra/Security** (`D-57` butir 3) |
| **Keputusan** | Rancangan penjadwal di sistem baru (satu node atau seluruh node) | **Lead Engineer** (`ADR-0022`) |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-S6-001](issues/01-penjadwal-dan-kunci-satu-pelaksana.md) | Penjadwal dan kunci satu pelaksana | `needs-info` |
| [TKT-S6-002](issues/02-lima-job-dan-agent.md) | Lima job terjadwal dan satu agent | `needs-info` |
| [TKT-S6-003](issues/03-auto-accept-komite.md) | Persetujuan komite otomatis jam 06:00 | `needs-info` |

## Daftar Tiket

### TKT-S6-001 — Penjadwal dan kunci satu pelaksana

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan rancangan (`ADR-0022`)** |
| **Modul** | **S-6 Job Terjadwal & Proses Otomatis** · Gelombang: 7 · Bergantung pada: TKT-F1-002 |
| **Requirement** | FR-S6 |
| **Keputusan** | D-57 |
| **ADR** | 0022 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | mekanisme Job Scheduler Pega — `pyApplicableTo=Cluster`, `pyNodeTypesText=BackgroundProcessing` |
| **Peran penguji gerbang 2** | **PncAdmin** |
| **Label** | `modul::S-6` `tipe::fondasi` `status::needs-info` `prioritas::sedang` `gelombang::7` |
| **Milestone** | Gelombang 7 — Sisa |
| **Berkas sumber** | `ticketing/S-6-Job-Terjadwal-dan-Proses-Otomatis/issues/01-penjadwal-dan-kunci-satu-pelaksana.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Penjadwal yang menjalankan pekerjaan pada jamnya, dan menjalankannya **tepat satu kali** walau
aplikasi berjalan di beberapa node.

Nilai bisnisnya persis pada kata "tepat satu kali". Bila `JobSendAutoLODKlaimPersonal` berjalan di
tiga node sekaligus, nasabah menerima **tiga LOD**. Bila `ProcessClaimKredit` berjalan rangkap,
klaim kredit terbentuk ganda.

#### Ruang lingkup

- Penjadwal dengan jadwal harian dan mingguan pada jam yang dapat dikonfigurasi.
- **Kunci agar satu job hanya berjalan di satu node** — setara `pyApplicableTo=Cluster` di Pega.
- Pencatatan tiap eksekusi: job apa, mulai kapan, selesai kapan, berhasil atau gagal.
- Perilaku saat job gagal: **tidak mengulang diam-diam** tanpa disadari.
- Cara **mematikan satu job** tanpa mematikan yang lain, dan tanpa rilis.

#### Non-goal

- **Tidak** memindahkan isi kelima job — itu `TKT-S6-002`.
- **Tidak** memutuskan nasib `AutoAcceptKomite` — itu `TKT-S6-003`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Penjadwal berjalan di dalam aplikasi atau sebagai proses terpisah?** | **Lead Engineer** (`ADR-0022`) | Menentukan bentuk kuncinya dan cara job dipantau. `ADR-0022` masih `Proposed` |
| **Job yang terlewat karena aplikasi mati — dikejar atau dilewatkan?** | **Work Owner** | Mengejar `JobSendAutoLODKlaimPersonal` yang terlewat dua hari berarti mengirim LOD ganda |

#### Acceptance criteria

- ☐ Job berjalan **tepat satu kali** walau tiga node hidup bersamaan — diuji dengan tiga
      instance: catatan eksekusi menunjukkan **satu**, bukan tiga.
- ☐ Job berjalan pada **jam yang dikonfigurasi**, bukan jam yang tertanam di kode — diuji dengan
      mengubah konfigurasi.
- ☐ Satu job dapat **dimatikan tanpa rilis** dan tanpa memengaruhi job lain — diuji.
- ☐ Kegagalan job **tercatat dan terlihat** — diuji dengan job yang sengaja gagal; kegagalannya
      tidak diam.
- ☐ Job yang berjalan lama **tidak menghalangi** job berikutnya berjalan — diuji.
- ☐ Node yang mati di tengah eksekusi **tidak meninggalkan kunci yang menggantung selamanya** —
      diuji dengan mematikan node paksa.
- ☐ Gerbang 2: UAT **PncAdmin**.

#### Dependency / Blocked by

`TKT-F1-002`. **Terhalang `ADR-0022` yang masih `Proposed`.**

#### Constraint keamanan, data, operasional

- Job berjalan **tanpa pengguna**. Setiap perubahan data olehnya tetap wajib masuk jejak audit
  (`S-5`) dengan identitas job — bukan tanpa pelaku.
- Selama Pega dan Go berjalan berdampingan (`D-05`), **job yang sama dapat aktif di kedua sistem**.
  Itu akan menggandakan efeknya. Pengalihan job wajib mematikan sisi Pega-nya lebih dulu, dan itu
  bagian dari rencana rollout — bukan hal yang bisa diserahkan ke kebetulan.

#### Migrasi skema / rollout / rollback

Menambah tabel kunci dan tabel catatan eksekusi job. Tidak menyentuh tabel klaim. Backward-compatible.

**Rollout:** satu job dialihkan pada satu waktu, **setelah job yang sama dimatikan di Pega**.

**Rollback:** mematikan job di Go dan menyalakan kembali di Pega. Keduanya **tidak boleh menyala
bersamaan**.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/penjadwal/... -run TestSatuJobSatuNode
go test ./internal/app/penjadwal/... -run TestKunciTidakMenggantungSaatNodeMati
go test ./internal/app/penjadwal/... -run TestJobDapatDimatikanTanpaRilis
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 5 job, seluruhnya `pyApplicableTo=Cluster`, `pyIsEnabled=true` | `D-57` · direktori `Job Scheduler/` |
| Jam dan frekuensi tiap job | `D-57` tabel |
| Rancangan penjadwal belum diputuskan | `ADR-0022` (status `Proposed`) |
| Pega dan Go berjalan berdampingan | `D-05` |

#### Comments

### TKT-S6-002 — Lima job terjadwal dan satu agent

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan** |
| **Modul** | **S-6 Job Terjadwal & Proses Otomatis** · Gelombang: 7 · Bergantung pada: TKT-S6-001 |
| **Requirement** | FR-S6 |
| **Keputusan** | D-57 |
| **ADR** | 0022 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | `Job Scheduler/JobHitungDeadlineToTemporaryClose-Job.xml` · `JobSendAutoLODKlaimPersonal-Job.xml` · `PNCMyReportKlaim3-Job.xml` · `ProcessClaimKredit-Job.xml` · `Agents/TATReportAgent-Agents.xml` |
| **Peran penguji gerbang 2** | **PncAdmin**, **PncKasir** (alert kasir), **PNCReportClaimInternal** (laporan) |
| **Label** | `modul::S-6` `tipe::migrasi` `status::needs-info` `prioritas::sedang` `gelombang::7` |
| **Milestone** | Gelombang 7 — Sisa |
| **Berkas sumber** | `ticketing/S-6-Job-Terjadwal-dan-Proses-Otomatis/issues/02-lima-job-dan-agent.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Empat job dan satu agent berjalan di sistem baru dengan hasil yang sama.

Nilai bisnisnya tidak merata di antara keenamnya: `JobHitungDeadlineToTemporaryCLose` **mengubah
status klaim**, `JobSendAutoLODKlaimPersonal` **mengirim dokumen ke nasabah**, dan
`TransferAllCaseNotAssigned` **memindahkan pekerjaan antar petugas**. Ketiganya berakibat nyata
tanpa ada orang yang menyetujuinya saat itu.

> `JOBForKomiteKlaimPNC` **tidak termasuk di sini** — ia digarap terpisah di `TKT-S6-003` karena
> dampaknya adalah persetujuan klaim.

#### Ruang lingkup

- `JobHitungDeadlineToTemporaryCLose` — mingguan 23:43 → `JobTemporaryCloseClaimPNC`.
- `JobSendAutoLODKlaimPersonal` — harian 20:54 → `Act_SendAutoLODKlaimPersonal`.
- `PNCMyReportKlaim3` — harian 08:00 → `ReportAI_Act`.
- `ProcessClaimKredit` — harian 10:00 → `CreateClaimCredit_Table`.
- `TATReportAgent` — tiap 30 menit → alert kasir belum transfer · `TransferAllCaseNotAssigned` ·
  `CreateCasePNCAgent_ActButton`.

#### Non-goal

- **Tidak** mengubah jam maupun frekuensi — kecuali butir "job jalan 2 menit" diputuskan lain.
- **Tidak** menambah job baru.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Tiga job ber-`pyDescription = "job jalan 2 menit"` sementara konfigurasinya `Daily`/`Weekly`** — selisih **720× sampai 5.040×** | **Work Owner + Tim Pega** (`D-57` butir 2) | Salah satu dari keduanya salah. Menyalin yang keliru berarti job berjalan 720 kali lebih sering atau lebih jarang daripada yang dimaksud |
| **`pyBypassActivityAuthentication=true` pada agent** — dibawa ke sistem baru? | **Work Owner + Tim Infra/Security** (`D-57` butir 3) | Agent ini **memindahkan pekerjaan antar petugas** tanpa pemeriksaan otentikasi. Membawanya apa adanya berarti menyalin jalur yang tidak terkontrol |
| **`TransferAllCaseNotAssigned` memindahkan kasus berdasarkan aturan apa?** | **Work Owner** | Isi activity-nya ada, tetapi **maksud bisnisnya** — kasus mana yang layak dipindah dan ke siapa — tidak tertulis di mana pun |

#### Acceptance criteria

- ☐ Keempat job dan agent berjalan pada jadwal yang ditetapkan — diuji per job.
- ☐ Hasil tiap job **sama dengan hasil Pega** atas data uji yang sama — dibandingkan lewat `S-8`
      pada minimal 10 kasus per job.
- ☐ `JobSendAutoLODKlaimPersonal` **tidak mengirim LOD ganda** bila dijalankan ulang — diuji.
- ☐ `JobHitungDeadlineToTemporaryCLose` mengubah status **hanya klaim yang memenuhi syarat** —
      diuji dengan klaim di kedua sisi batas tenggat.
- ☐ Setiap perubahan data oleh job **tercatat di jejak audit** dengan identitas job — diuji.
- ☐ Agent yang memindahkan kasus **mencatat asal dan tujuan penugasan** — diuji.
- ☐ Gerbang 2: UAT oleh peran yang menerima akibatnya — **PncKasir** untuk alert,
      **PNCReportClaimInternal** untuk laporan.

#### Dependency / Blocked by

`TKT-S6-001` · `TKT-B06-002` (penugasan) · `TKT-S1-001` (LOD) · `TKT-S5-001` (jejak audit).
**Terhalang tiga keputusan `D-57`.**

#### Constraint keamanan, data, operasional

- Job mengubah data **tanpa pengguna yang menyetujui saat itu**. Jejak auditnya adalah satu-satunya
  cara mengetahui apa yang terjadi.
- `JobSendAutoLODKlaimPersonal` **mengirim dokumen keluar ke nasabah** — kesalahan di sini terlihat
  oleh pihak luar dan tidak dapat ditarik kembali.
- Job aktif di Pega **dan** Go bersamaan akan menggandakan efeknya (`P-1`).

#### Migrasi skema / rollout / rollback

Tidak menambah tabel bisnis; memakai catatan eksekusi dari `TKT-S6-001`.

**Rollout:** satu job pada satu waktu, dimulai dari yang **tidak mengirim apa pun keluar**
(`PNCMyReportKlaim3`), dan yang mengirim ke nasabah paling akhir.

**Rollback:** mematikan job di Go, menyalakan di Pega. **LOD yang telanjur terkirim tidak dapat
ditarik.**

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/job/... -run TestLODTidakGandaSaatDijalankanUlang
go test ./internal/app/job/... -run TestTemporaryCloseHanyaYangLewatTenggat
go run ./cmd/s8 banding --modul S-6 --per-job 10
go test ./internal/app/audit/... -run TestPerubahanOlehJobTercatat
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Jam, frekuensi, dan activity target tiap job | `D-57` tabel · direktori `Job Scheduler/` |
| Agent `pyTriggerInterval=1800`, `pyBypassActivityAuthentication=true` | `D-57` · `Agents/TATReportAgent-Agents.xml` |
| Agent merujuk `TransferAllCaseNotAssigned` dan alert kasir | `D-57` |
| Tiga job berdeskripsi "job jalan 2 menit" | `D-57` butir 2 · `docs/verifikasi-bukti-adr.md:2702-2705` |

#### Comments

### TKT-S6-003 — Persetujuan komite otomatis jam 06:00

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan Work Owner — pertanyaan belum terjawab** |
| **Modul** | **S-6 Job Terjadwal & Proses Otomatis** · Gelombang: 7 · Bergantung pada: TKT-S6-001, TKT-B07-002 |
| **Requirement** | FR-S6, FR-B7 |
| **Keputusan** | D-57 |
| **ADR** | 0022 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | `Job Scheduler/JOBForKomiteKlaimPNC-Job.xml` — `Daily`, **`06:00:00`**, activity target **`AutoAcceptKomite`** |
| **Peran penguji gerbang 2** | **PncManagerAdmin** |
| **Label** | `modul::S-6` `tipe::keamanan` `status::needs-info` `prioritas::tinggi` `gelombang::7` |
| **Milestone** | Gelombang 7 — Sisa |
| **Berkas sumber** | `ticketing/S-6-Job-Terjadwal-dan-Proses-Otomatis/issues/03-auto-accept-komite.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Kejelasan lebih dulu, kode belakangan: **apakah persetujuan komite otomatis memang dikehendaki.**

Nilai bisnisnya — dan alasan tiket ini berdiri sendiri — adalah ini: setiap hari jam 06:00, sebuah
job menjalankan `AutoAcceptKomite` **tanpa pengguna sama sekali**. Seluruh kendali `B-7` — jenjang
kumulatif, pita nilai, pemisahan peran — **tidak berlaku padanya**. Bahkan kontrol berbasis menu
(`D-59`) tidak menyentuhnya, karena tidak ada menu yang dibuka.

**Perilaku ini tidak tercatat di dokumen mana pun** dan belum pernah dibahas sampai ditemukan di
export (`D-57`).

#### Ruang lingkup

Ditetapkan **setelah** Work Owner menjawab. Salah satu dari tiga arah:

1. **Dipertahankan apa adanya** — dipindahkan dengan perilaku sama, dan dicatat sebagai keputusan
   sadar beserta alasannya.
2. **Dipertahankan dengan batas** — misalnya hanya untuk klaim di bawah nilai tertentu, atau hanya
   untuk klaim yang sudah menunggu sekian lama.
3. **Dihentikan** — klaim menunggu persetujuan manusia, dan dampaknya pada antrean diukur lebih dulu.

#### Non-goal

- **Tidak** memilih salah satu dari ketiganya sendiri. Ini keputusan bisnis dengan dampak finansial,
  dan ia bukan milik saya.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **`AutoAcceptKomite` berjalan tiap hari 06:00 — benar demikian?** | **Work Owner** (`D-57` butir 1) | Tanpa jawaban, tiket ini tidak punya lingkup sama sekali. Menebaknya berarti memutuskan sendiri apakah klaim boleh disetujui tanpa manusia |
| **Bila dipertahankan: klaim mana yang layak disetujui otomatis?** | **Work Owner** | Menentukan penyaringnya. Tanpa itu, "otomatis" berarti **semua** |
| **Berapa banyak klaim yang benar-benar disetujui job ini per hari?** | **Work Owner + Tim Pega** | Angka ini menentukan apakah menghentikannya akan menumpuk antrean komite. Ia **tidak dapat diukur dari export** — hanya dari data produksi |
| **Apakah jejak audit Pega mencatat persetujuan ini sebagai dilakukan siapa?** | **Tim Pega** | Bila tercatat atas nama pengguna tertentu, jejak audit selama ini **menyesatkan** |

#### Acceptance criteria

> **Belum dapat dituliskan.** Kriteria kelulusan tiket ini bergantung sepenuhnya pada arah yang
> dipilih Work Owner. Menuliskan daftar bercentang sekarang akan membuat tiket ini tampak siap
> dikerjakan padahal pertanyaan pokoknya belum terjawab.

Yang **sudah pasti berlaku**, apa pun arahnya:

- ☐ Setiap persetujuan otomatis **tercatat di jejak audit sebagai dilakukan oleh job**, bukan
      atas nama pengguna mana pun (`S-5`).
- ☐ Perilaku yang dipilih **tercatat sebagai keputusan di Decision Log** beserta alasannya —
      sehingga tidak ada lagi jalur persetujuan yang berjalan tanpa ada yang mengetahuinya.

#### Dependency / Blocked by

`TKT-S6-001` · `TKT-B07-002` (aturan jenjang komite) · `TKT-S5-001` (jejak audit).
**Terhalang jawaban Work Owner atas `D-57` butir 1.**

#### Constraint keamanan, data, operasional

- **Ini jalur persetujuan klaim yang tidak melewati kontrol mana pun.** Ia sejenis dengan
  `TKT-S4-002` (persetujuan komite lewat API): keduanya memintas `B-7`, dan keduanya baru ditemukan
  di fase verifikasi. Bila `B-7` dibangun dengan cermat sementara kedua jalur ini dibiarkan,
  kecermatan itu tidak ada artinya.
- Menghentikannya **menggeser beban ke komite manusia**; mempertahankannya **menyalin jalur tanpa
  kendali ke sistem baru**. Keduanya punya harga, dan keduanya harus dipilih dengan sadar.

#### Migrasi skema / rollout / rollback

Ditetapkan setelah arahnya dipilih.

**Yang sudah dapat dikatakan:** job ini **tidak boleh aktif di Pega dan Go bersamaan** — dua
persetujuan otomatis atas klaim yang sama.

#### Rencana verifikasi

> **Rencana — belum dapat disusun.** Ia mengikuti arah yang dipilih.

Yang pasti diperlukan apa pun arahnya:

```bash
go test ./internal/app/audit/... -run TestPersetujuanOtomatisTercatatSebagaiJob
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `JOBForKomiteKlaimPNC` harian 06:00 → `AutoAcceptKomite` | `D-57` tabel · `Job Scheduler/JOBForKomiteKlaimPNC-Job.xml` |
| Perilaku ini tidak tercatat di dokumen mana pun dan belum pernah dibahas | `D-57` butir 1 |
| Job melewati bahkan kontrol berbasis menu | `docs/Steering/00-DECISION-LOG.md:1634` · `D-59` |
| Jenjang komite kumulatif; pita nilai hanya Non-MBU | `D-52` · `D-70` |

#### Comments

# 28. S-7 · Dashboard Klaim, TAT & KPI

*Modul Pendukung · folder `docs/ticketing/S-7-Dashboard-TAT-dan-KPI/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Dashboard & Monitoring** — harness `PNCTATReport`, `ReportKPIHarness`, `OutstandingKlaimperCabang_Harness` · `Database/GETSELISIHJAM.fnc` · `Database/GET_POSISI_PROGRESS_PNC.fnc` · `PROGRESS_CLAIM_PNC` |
| **Kode modul** | `S-7` |
| **Gelombang** | 6 — Laporan |
| **Ukuran** | bagian dari 78 activity `S-2` |
| **Bergantung pada** | `S-2` Laporan & Export |
| **Kesiapan** | **TERHALANG** |

### Apa yang dikerjakan modul ini

Menampilkan **berapa lama klaim diproses** dan **berapa yang masih menggantung** — TAT, KPI, dan
posisi progres klaim per cabang.

### Kenapa modul ini TERHALANG

Karena angka yang ditampilkannya **saat ini tidak dapat dipercaya**, dan itu terbukti dari
source-nya, bukan dari dugaan.

### Temuan yang mengubah rencana

| Anggapan | Terverifikasi |
|---|---|
| TAT dihitung dengan satu rumus | **Dua basis yang saling eksklusif** — `GETSELISIHJAM` (akhir pekan saja) dan `GET_WORKING_HOURS@ASMD` (dipakai **17×**). Angka dari kedua jalur **tidak akan pernah sama** |
| TAT memperhitungkan hari libur dan jam kerja | **Tidak.** `GETSELISIHJAM` hanya mengurangi akhir pekan. SLA Jumat 16:00 → Senin 09:00 dihitung **17 jam**, bukan ~2 jam kerja |
| Kegagalan hitungan terlihat | **`EXCEPTION WHEN OTHERS THEN RETURN 0`** — kegagalan **tak terbedakan dari nol jam** |
| "Posisi aktif" disaring lewat status | **`STATUSPOSISI` selalu `'On Progress'`, nol pengecualian.** Filternya cocok dengan **semua** baris. Penanda selesai yang sebenarnya adalah **`PROGRESSDATEDONE`** |

> Artinya angka TAT dan KPI yang dilihat manajemen selama ini **mungkin keliru**, dan sebagian
> kegagalan hitungan **tersamar sebagai nol**. Memperbaikinya akan **mengubah angka** — dan itu
> harus disampaikan, bukan diselipkan (`P-5`, `R-19`).

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | **Basis TAT yang benar: `GETSELISIHJAM` (akhir pekan saja) atau `GET_WORKING_HOURS@ASMD`?** | **Work Owner** |
| **Keputusan** | **`RETURN 0` saat error pernah menyamarkan kegagalan KPI** — angka historis perlu diperiksa ulang? | **Work Owner** |
| **Artefak** | `weekends2` dan `POOLDATA.datediff` **tidak ada di export** — definisi "akhir pekan" tidak dapat diverifikasi | **DBA** (`R-01`) |
| **Keputusan** | `GET_WORKING_HOURS@ASMD` dan `HRD_LBR@ASMD` adalah objek **remote** — masuk `D-25`/`R-03`, dan logikanya **ditulis ulang di Go** | **Tim Infra + Work Owner** (`R-19`) |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-S7-001](issues/01-basis-perhitungan-tat.md) | Basis perhitungan TAT | `needs-info` |
| [TKT-S7-002](issues/02-dashboard-dan-posisi-progres.md) | Dashboard klaim dan posisi progres | `needs-info` |

## Daftar Tiket

### TKT-S7-001 — Basis perhitungan TAT

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan aturan bisnis** |
| **Modul** | **S-7 Dashboard, TAT & KPI** · Gelombang: 6 · Bergantung pada: TKT-F5-001, TKT-S4-003 |
| **Requirement** | FR-S7 |
| **Keputusan** | D-25, D-49 |
| **ADR** | 0017 |
| **Risiko** | R-01, R-03, R-19 |
| **Rule Pega yang digantikan** | `Database/GETSELISIHJAM.fnc:9-18` (algoritma) dan `:19-22` (`EXCEPTION WHEN OTHERS THEN RETURN 0`) · `DATAMINING.GET_WORKING_HOURS@ASMD` (**17 pemakaian**) · `GENERAL.HRD_LBR@ASMD` |
| **Peran penguji gerbang 2** | **PncManagerAdmin** dan **PNCReportClaimInternal** |
| **Label** | `modul::S-7` `tipe::aturan-bisnis` `status::needs-info` `prioritas::tinggi` `gelombang::6` |
| **Milestone** | Gelombang 6 — Laporan |
| **Berkas sumber** | `ticketing/S-7-Dashboard-TAT-dan-KPI/issues/01-basis-perhitungan-tat.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu rumus TAT yang disepakati, dipakai seluruh laporan dan dashboard.

Nilai bisnisnya: sekarang ada **dua** rumus. `GETSELISIHJAM` mengurangi akhir pekan saja;
`GET_WORKING_HOURS@ASMD` memakai jam kerja dan kalender libur. Keduanya dipakai di jalur berbeda,
dan **angkanya tidak akan pernah sama**. Selama itu berlangsung, "TAT klaim" bukan satu angka —
ia dua angka yang keduanya disebut TAT.

Sebagai ukuran selisihnya: SLA yang dimulai **Jumat 16:00** dan berakhir **Senin 09:00** dihitung
**17 jam** oleh `GETSELISIHJAM`, sementara menurut jam kerja ia sekitar **2 jam**.

#### Ruang lingkup

- **Satu** fungsi perhitungan TAT di Go, dipakai seluruh laporan, dashboard, dan KPI.
- Perhitungan memakai kalender hari libur dan jam kerja dari `TKT-F5-001` — bukan panggilan remote
  (`R-19`: aturan bisnis ditulis ulang, tidak dipanggil).
- **Kegagalan perhitungan menghasilkan kegagalan yang terlihat**, bukan nol.
- Pemetaan angka lama ke angka baru, agar selisihnya dapat dijelaskan.

#### Non-goal

- **Tidak** menggarap tampilan dashboard — itu `TKT-S7-002`.
- **Tidak** memperbaiki data historis. Memperbaiki rumus **tidak memperbaiki baris yang telanjur
  salah** (`R-19`).

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Basis TAT yang benar — akhir pekan saja, atau jam kerja + hari libur?** | **Work Owner** | Ini **aturan bisnis**, bukan pilihan teknis. Memilihnya sendiri berarti menentukan apakah cabang dinilai lulus SLA atau tidak |
| **`RETURN 0` saat error pernah menyamarkan kegagalan KPI** — angka historis perlu diperiksa ulang? | **Work Owner** | Bila pernah, sebagian angka KPI yang sudah dilaporkan **adalah nol palsu**, dan keputusan yang diambil di atasnya perlu ditinjau |
| **`weekends2` dan `POOLDATA.datediff` tidak ada di export** | **DBA** (`R-01`) | Definisi "akhir pekan" **tidak dapat diverifikasi**. Bila opsi `GETSELISIHJAM` yang dipilih, ia tidak dapat disalin tanpa source-nya |
| **Angka TAT baru akan berbeda dari yang lama — diumumkan bagaimana?** | **Work Owner** | Perubahan ini terlihat langsung oleh manajemen cabang |

#### Acceptance criteria

- ☐ **Hanya ada satu** fungsi TAT di seluruh kode — dicari otomatis; tidak ada perhitungan
      tandingan di laporan mana pun.
- ☐ Kegagalan perhitungan **menghasilkan kegagalan**, bukan `0` — diuji dengan masukan rusak.
      Ini **perbaikan perilaku yang disengaja** dan masuk daftar `P-5`.
- ☐ Hasil perhitungan **sama dengan basis yang dipilih Work Owner** pada 20 kasus, termasuk kasus
      lintas akhir pekan dan lintas hari libur nasional.
- ☐ Kasus **Jumat 16:00 → Senin 09:00** menghasilkan angka yang **sesuai keputusan**, dan angka
      itu **dicatat di tiket ini** sebagai acuan.
- ☐ Selisih terhadap angka Pega **dilaporkan per laporan**, bukan disamakan diam-diam — lewat
      `S-8` dan diklasifikasikan terhadap 13 butir `P-5`.
- ☐ Gerbang 2: UAT **PncManagerAdmin**.

#### Dependency / Blocked by

`TKT-F5-001` (jam kerja & hari libur) · `TKT-S4-003` (objek remote). **Terhalang Work Owner dan DBA.**

#### Constraint keamanan, data, operasional

- **Angka TAT menilai kinerja cabang.** Mengubah rumusnya mengubah penilaian itu — sebagian cabang
  akan terlihat lebih baik, sebagian lebih buruk, tanpa ada yang berubah cara kerjanya.
- `P-5` menuntut kesetaraan perilaku lebih dulu, tetapi `R-19` memperingatkan bahwa **mematuhinya
  buta akan menyalin cacat hitungan uang dan waktu**. Tiket ini termasuk yang dimaksud.
- `GET_WORKING_HOURS@ASMD` adalah **objek remote** — tidak tersedia setelah pindah ke PostgreSQL.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema.

**Rollback:** mengembalikan rumus lama **mengembalikan angka lama** — termasuk `RETURN 0` yang
menyamarkan kegagalan. Ini bukan rollback yang netral.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/tat/... -run TestSatuRumusTAT
go test ./internal/app/tat/... -run TestKegagalanBukanNol
go test ./internal/app/tat/... -run TestJumat1600SeninPagi
go run ./cmd/s8 banding --modul S-7 --kasus 20 --laporkan-selisih
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Algoritma `GETSELISIHJAM`: akhir pekan saja | `Database/GETSELISIHJAM.fnc:9-18` · `docs/verifikasi-bukti-adr.md` §14.9 |
| `EXCEPTION WHEN OTHERS THEN RETURN 0` | `Database/GETSELISIHJAM.fnc:19-22` |
| `GET_WORKING_HOURS@ASMD` dipakai 17× | `D-25` tabel inventaris DB Link |
| Jumat 16:00 → Senin 09:00 dihitung 17 jam | `docs/verifikasi-bukti-adr.md` §14.9 |
| `weekends2` dan `POOLDATA.datediff` tidak ada di export | idem · `R-01` |
| Aturan bisnis remote ditulis ulang di Go | `docs/Steering/00-DECISION-LOG.md:1326` · `R-19` |

#### Comments

### TKT-S7-002 — Dashboard klaim dan posisi progres

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan `TKT-S7-001`** |
| **Modul** | **S-7 Dashboard, TAT & KPI** · Gelombang: 6 · Bergantung pada: TKT-S7-001, TKT-S2-001 |
| **Requirement** | FR-S7 |
| **Keputusan** | D-18, D-49 |
| **ADR** | 0011 |
| **Risiko** | R-19 |
| **Rule Pega yang digantikan** | harness `PNCTATReport`, `ReportKPIHarness`, `OutstandingKlaimperCabang_Harness` · `Database/GET_POSISI_PROGRESS_PNC.fnc:15` · `PROGRESS_CLAIM_PNC:7-8,40,67` |
| **Peran penguji gerbang 2** | **PncManagerAdmin** dan **PncKacab** |
| **Label** | `modul::S-7` `tipe::migrasi` `status::needs-info` `prioritas::sedang` `gelombang::6` |
| **Milestone** | Gelombang 6 — Laporan |
| **Berkas sumber** | `ticketing/S-7-Dashboard-TAT-dan-KPI/issues/02-dashboard-dan-posisi-progres.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Layar yang menunjukkan **klaim mana yang masih menggantung, di tangan siapa, dan sudah berapa lama**.

Nilai bisnisnya: inilah satu-satunya tempat kepala cabang melihat pekerjaan yang tertahan. Tetapi
saat ini layar itu **menampilkan seluruh riwayat sebagai "aktif"**, karena penyaringnya keliru —
sehingga yang seharusnya menjadi alat pengawasan justru menyembunyikan keadaan sebenarnya.

#### Ruang lingkup

- Dashboard klaim, KPI, dan outstanding per cabang.
- **Penyaring "aktif" memakai `PROGRESSDATEDONE IS NULL`**, bukan `STATUSPOSISI = 'On Progress'`.
- Angka TAT diambil dari **satu** fungsi (`TKT-S7-001`).
- Penegakan batas data cabang: kepala cabang melihat cabangnya (`TKT-F3-005`).

#### Non-goal

- **Tidak** menentukan basis TAT — itu `TKT-S7-001`.
- **Tidak** menambah metrik baru yang belum ada di sistem lama.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Basis TAT belum ditetapkan** (`TKT-S7-001`) | **Work Owner** | Seluruh angka di dashboard ini turunan darinya |
| **Memperbaiki penyaring "aktif" akan menurunkan angka outstanding secara tajam** — diumumkan bagaimana? | **Work Owner** | Baris yang sudah selesai selama ini ikut terhitung aktif. Setelah diperbaiki, angkanya berubah besar — dan itu **terlihat sebagai penurunan mendadak** bila tidak dijelaskan |
| **Default follow-up `sysdate + 7`** — 7 hari itu aturan bisnis yang masih berlaku? | **Work Owner** | Satu-satunya angka bisnis literal di `PROGRESS_CLAIM_PNC`; ia menentukan kapan klaim muncul sebagai perlu ditindaklanjuti |
| **Siapa yang boleh melihat dashboard lintas cabang?** | **Work Owner** (`D-59`) | Menentukan batas data pada layar ini |

#### Acceptance criteria

- ☐ Penyaring "aktif" memakai **`PROGRESSDATEDONE IS NULL`** — diuji dengan baris yang sudah
      selesai: **tidak** muncul sebagai aktif. Ini **perbaikan perilaku yang disengaja** (`P-5`).
- ☐ Seluruh angka TAT di dashboard berasal dari fungsi `TKT-S7-001` — tidak ada perhitungan
      tandingan di kueri dashboard.
- ☐ Kepala cabang **hanya melihat cabangnya** — diuji dengan dua peran cabang berbeda.
- ☐ Default follow-up **sesuai keputusan Work Owner**, dan angkanya **dikonfigurasi**, bukan
      tertanam di kode — diuji.
- ☐ Selisih angka terhadap Pega **dilaporkan sebagai perbaikan terencana**, bukan bug — dicatat
      per metrik lewat `S-8`.
- ☐ Dashboard **tidak memakai koneksi transaksi** saat memuat data berat (`TKT-F2-001`) — diuji.
- ☐ Gerbang 2: UAT **PncManagerAdmin** dan **PncKacab** secara terpisah.

#### Dependency / Blocked by

`TKT-S7-001` · `TKT-S2-001` · `TKT-F3-005`. **Terhalang keputusan Work Owner.**

#### Constraint keamanan, data, operasional

- Dashboard memuat data klaim lintas cabang bila batasnya tidak ditegakkan — ia jalan pintas yang
  sama seperti laporan (`TKT-S2-002`).
- **Angka outstanding akan turun tajam setelah penyaingnya diperbaiki.** Tanpa penjelasan,
  penurunan itu akan terbaca sebagai kesalahan sistem baru — padahal ia justru koreksi.
- Kolom `STATUSPOSISI` **tetap ada dan tetap bernilai `'On Progress'`** di data lama; memperbaiki
  penyaring **tidak memperbaiki kolomnya**.

#### Migrasi skema / rollout / rollback

Tidak mengubah kolom; mengubah **cara membacanya**. Mungkin menambah index pada `PROGRESSDATEDONE`
(`TKT-F2-004`).

**Rollback:** kembali ke penyaring `STATUSPOSISI` **mengembalikan angka yang keliru**.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/dashboard/... -run TestAktifMemakaiProgressDateDone
go test ./internal/app/dashboard/... -run TestBatasDataCabang
go run ./cmd/s8 banding --modul S-7 --metrik semua --laporkan-selisih
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `STATUSPOSISI` selalu `'On Progress'`, nol pengecualian | `docs/verifikasi-bukti-adr.md` §14.8 |
| `GET_POSISI_PROGRESS_PNC.fnc:15` menyaring `'On Progress'` — cocok semua baris | idem |
| Penanda selesai sebenarnya `PROGRESSDATEDONE` (`:40`, `:67`) | idem |
| Default follow-up `sysdate + 7` (`:7-8`) | idem |
| Harness dashboard dan KPI | `docs/Steering/06-MODULE-BREAKDOWN.md:66` |

#### Comments

# 29. S-8 · Perkakas Uji Kesetaraan

*Modul Pendukung · folder `docs/ticketing/S-8-Perkakas-Uji-Kesetaraan/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **tidak ada padanannya** — modul ini **100% baru** |
| **Kode modul** | `S-8` |
| **Gelombang** | **1 — Fondasi** (`D-42`) |
| **Ukuran** | Besar — seluruhnya baru |
| **Bergantung pada** | `F-1`, `F-2` |
| **Kesiapan** | **TERHALANG** |

### Apa yang dikerjakan modul ini

Menjalankan **kasus yang sama** di Pega staging dan di Go staging, membandingkan hasilnya, lalu
**mengklasifikasikan setiap selisih** terhadap 13 butir perbaikan `P-5` (`D-49`, `D-54`).

### Kenapa modul ini ada di gelombang 1, bukan gelombang akhir

`BRD §21.1` menjadikan uji kesetaraan otomatis sebagai **gerbang pertama setiap modul**. Selama
perkakas ini belum ada, **tidak ada satu modul pun yang dapat dinyatakan lulus** — dan Strangler
Fig (`D-05`) berhenti sebelum dimulai (`D-42`).

Artinya `S-8` bukan modul pendukung yang bisa menunggu. **Ia penghalang tunggal bagi seluruh
rencana.**

### Yang membuatnya berbeda dari modul lain

| | |
|---|---|
| **Tidak menyalin apa pun** | Tidak ada rule Pega yang digantikannya. Ia tidak tunduk `P-5` — **ia yang menegakkannya** |
| **Tidak dapat menguji dirinya sendiri** | Kebenarannya dibuktikan dengan selisih yang sengaja dibuat, bukan dengan membandingkan ke Pega |
| **Dua modul tak punya baseline** | `F-3` (HCC/HCQ nol jejak di export) dan `S-5` (sistem lama tidak punya jejak audit nilai) — `BRD §21.2` kriteria #3 **tidak dapat diberlakukan** pada keduanya (`D-42`) |

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | **Ketersediaan Pega staging yang dapat ditembak dari luar** — penghalang utama `S-8` | **Tim Pega + Infra** (`R-14`, `ADR-0027`) |
| **Keputusan** | Siapa boleh menembak Pega, di lingkungan mana, dan **apakah boleh membaca produksi read-only** | **Work Owner + Tim Infra/Security** (`D-42`) |

**Sudah diputuskan:** selisih yang cocok dengan **salah satu dari 13 butir `P-5`** lolos otomatis
dan cukup dicatat; selisih **di luar 13 butir** wajib **persetujuan Work Owner tertulis** sebelum
modul lulus gerbang 1 (`D-54`).

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-S8-001](issues/01-penembak-kasus-dan-perekam-hasil.md) | Penembak kasus dan perekam hasil | `needs-info` |
| [TKT-S8-002](issues/02-pembanding-dan-klasifikasi-selisih.md) | Pembanding hasil dan klasifikasi selisih terhadap 13 butir P-5 | `ready-for-human` |

## Daftar Tiket

### TKT-S8-001 — Penembak kasus dan perekam hasil

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang ketersediaan Pega staging** |
| **Modul** | **S-8 Perkakas Uji Kesetaraan** · Gelombang: 1 · Bergantung pada: TKT-F1-001, TKT-F2-001 |
| **Requirement** | FR-S8 |
| **Keputusan** | D-42 |
| **ADR** | 0027 |
| **Risiko** | R-14 |
| **Rule Pega yang digantikan** | **tidak ada** — modul ini 100% baru |
| **Peran penguji gerbang 2** | **Lead Engineer** (modul ini tidak dipakai peran bisnis) |
| **Label** | `modul::S-8` `tipe::fondasi` `status::needs-info` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/S-8-Perkakas-Uji-Kesetaraan/issues/01-penembak-kasus-dan-perekam-hasil.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Perkakas yang menjalankan satu daftar kasus di **dua sistem** — Pega staging dan Go staging — lalu
merekam hasil keduanya dalam bentuk yang dapat dibandingkan.

Nilai bisnisnya adalah nilai seluruh proyek: `BRD §21.1` menjadikan uji kesetaraan sebagai gerbang
pertama setiap modul. **Tanpa perkakas ini, tidak ada satu modul pun yang dapat dinyatakan lulus** —
dan tidak ada satu pun bagian Pega yang boleh dimatikan (`D-42`).

#### Ruang lingkup

- Definisi **kasus uji** dalam bentuk data, bukan kode — agar penambahannya tidak menuntut rilis.
- Penjalan kasus ke **Pega staging** dan ke **Go staging**.
- Perekaman hasil kedua sisi dalam bentuk yang stabil dan dapat dibandingkan ulang.
- Penjalanan ulang kasus yang sama menghasilkan rekaman yang sama bila sistemnya tidak berubah.

#### Non-goal

- **Tidak** membandingkan maupun mengklasifikasikan selisih — itu `TKT-S8-002`.
- **Tidak** menyentuh produksi. Tidak ada koneksi, tidak ada DDL, tidak ada DML.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Pega staging yang dapat ditembak dari luar — ada?** | **Tim Pega + Infra** (`R-14`, `ADR-0027`) | Ini **penghalang utama `S-8`**, dan karenanya penghalang gerbang 1 seluruh modul. Tanpa sisi Pega, tidak ada yang dapat dibandingkan |
| **Siapa boleh menembak Pega, di lingkungan mana?** | **Work Owner + Tim Infra/Security** (`D-42`) | Belum pernah ditanyakan sampai `D-42` |
| **Boleh membaca produksi read-only untuk mengambil kasus nyata?** | **Work Owner + Tim Infra/Security** (`D-42`) | Menentukan apakah kasus uji berasal dari data nyata atau data buatan — dan keduanya menghasilkan kepercayaan yang berbeda |
| **Kasus uji diambil dari mana, dan berapa banyak per modul?** | **Work Owner + Lead Engineer** | Tanpa jumlah minimum, "lulus gerbang 1" tidak punya arti terukur |

#### Acceptance criteria

- ☐ Satu kasus dapat dijalankan di **kedua sistem** dan hasilnya terekam — diuji dari ujung ke ujung.
- ☐ Kasus didefinisikan **sebagai data**; menambah kasus **tidak menuntut rilis** — diuji.
- ☐ Menjalankan kasus yang sama dua kali pada sistem yang tidak berubah menghasilkan rekaman
      **yang sama** — diuji; bila tidak, pembanding akan melaporkan selisih palsu.
- ☐ Perkakas **tidak pernah menulis ke produksi** — dipastikan lewat konfigurasi yang menolak
      alamat produksi, dan diuji dengan mencoba mengarahkannya ke sana: **ditolak**.
- ☐ Kegagalan menembak salah satu sisi **dilaporkan sebagai kegagalan**, bukan sebagai "tidak ada
      selisih" — diuji dengan mematikan salah satu sisi.
- ☐ Gerbang 2: ditinjau **Lead Engineer**; modul ini tidak dipakai peran bisnis.

#### Dependency / Blocked by

`TKT-F1-001` · `TKT-F2-001`. **Terhalang Tim Pega + Infra (`R-14`) — di luar kendali proyek.**

#### Constraint keamanan, data, operasional

- **Tidak ada koneksi ke produksi. Tidak ada DDL/DML.** Batas ini mutlak dan tidak dapat dilonggarkan
  demi kemudahan pengujian.
- Staging memuat **data produksi apa adanya** (`D-64`) — kasus uji dan rekaman hasilnya karena itu
  memuat **data nasabah nyata**. Rekaman **tidak boleh di-commit**, dan pembatasan data medis
  (`FR-R2`) berlaku pada isinya.
- Menembak Pega staging menambah beban ke sistem yang masih dipakai untuk pengujian lain.

#### Migrasi skema / rollout / rollback

Menambah tabel kasus uji dan rekaman hasil **di lingkungan uji**, bukan di skema klaim.

**Rollback:** perkakas ini tidak dipakai pengguna; mengembalikan versinya tidak berdampak ke bisnis.
Tetapi **mematikannya menghentikan gerbang 1 seluruh modul**.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./cmd/s8/... -run TestKasusSebagaiData
go test ./cmd/s8/... -run TestRekamanStabilSaatDiulang
go test ./cmd/s8/... -run TestMenolakAlamatProduksi
go test ./cmd/s8/... -run TestSisiMatiDilaporkanSebagaiGagal
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `S-8` modul 100% baru, gelombang 1 | `D-42` · `docs/Steering/06-MODULE-BREAKDOWN.md:67` |
| `S-8` memblokir gerbang 1 seluruh modul | `D-42` alasan · `BRD §21.1` |
| Ketersediaan Pega staging belum dipastikan | `R-14` · `docs/Steering/16-RISK-ANALYSIS.md:562` · `ADR-0027` |
| Kewenangan menembak Pega belum ditanyakan | `D-42` bagian Terbuka |
| Staging memuat data produksi apa adanya | `D-64` · `ADR-0029` |

#### Comments

### TKT-S8-002 — Pembanding hasil dan klasifikasi selisih terhadap 13 butir P-5

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap — menunggu urutan kerja `TKT-S8-001`, bukan pihak lain |
| **Modul** | **S-8 Perkakas Uji Kesetaraan** · Gelombang: 1 · Bergantung pada: TKT-S8-001 |
| **Requirement** | FR-S8 |
| **Keputusan** | D-42, D-49, D-54 |
| **ADR** | 0017, 0027 |
| **Risiko** | R-14, R-19 |
| **Rule Pega yang digantikan** | **tidak ada** — modul ini 100% baru |
| **Peran penguji gerbang 2** | **Lead Engineer**; keluarannya dibaca **Work Owner** |
| **Label** | `modul::S-8` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::1` |
| **Milestone** | Gelombang 1 — Fondasi |
| **Berkas sumber** | `ticketing/S-8-Perkakas-Uji-Kesetaraan/issues/02-pembanding-dan-klasifikasi-selisih.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Perkakas yang membandingkan hasil kedua sisi dan **memilah setiap selisih menjadi dua tumpukan**:
yang **terpetakan** ke salah satu dari 13 butir perbaikan `P-5`, dan yang **tidak terpetakan**.

Nilai bisnisnya terletak pada pemilahan itu, bukan pada pembandingannya. `D-54` menetapkan selisih
yang cocok dengan 13 butir itu **lolos otomatis**, sementara selisih di luar itu **wajib persetujuan
Work Owner tertulis**. Tanpa klasifikasi otomatis, setiap selisih akan menumpuk di meja Work Owner —
dan gerbang 1 berubah dari kendali menjadi hambatan.

Sekaligus inilah pagar terhadap `R-19`: `P-5` yang dipatuhi buta akan **menyalin cacat hitungan uang
dan waktu**. Perkakas ini yang memaksa setiap selisih dijawab, bukan dilewati.

#### Ruang lingkup

- Pembandingan hasil kedua sisi, termasuk **nilai uang berdesimal** dan **tanggal/jam**.
- Klasifikasi tiap selisih terhadap **13 butir `P-5`** (`D-49`) — keluarannya menyebut **butir mana**.
- Antrean selisih **tidak terpetakan** sebagai daftar yang menunggu persetujuan Work Owner tertulis.
- Laporan per modul: berapa kasus dijalankan, berapa setara, berapa terpetakan, berapa menunggu.
- Penanganan dua modul **tanpa baseline**: `F-3` dan `S-5` dilaporkan sebagai **tidak dapat diuji
  setara**, bukan sebagai lulus (`D-42`).

#### Non-goal

- **Tidak** memutuskan selisih mana yang dapat diterima. Selisih di luar 13 butir adalah
  **keputusan Work Owner**, dan perkakas ini hanya menyajikannya.
- **Tidak** memperbaiki kode yang menghasilkan selisih.

#### Acceptance criteria

- ☐ Selisih yang cocok dengan salah satu dari **13 butir `P-5`** ditandai **beserta nomor
      butirnya** — diuji pada minimal satu kasus per butir, ketiga belasnya.
- ☐ Selisih **di luar 13 butir** masuk antrean persetujuan dan **tidak pernah ditandai lolos** —
      diuji dengan selisih yang sengaja dibuat di luar daftar.
- ☐ Modul dengan selisih tak terpetakan yang belum disetujui **dilaporkan belum lulus gerbang 1** —
      diuji.
- ☐ Perbandingan **nilai uang** benar sampai desimal terkecil yang dipakai — diuji dengan nilai
      yang berbeda hanya di desimal: **terdeteksi sebagai selisih**, bukan dibulatkan sama.
- ☐ Perbandingan **tanggal dan jam** memperhitungkan zona waktu dan pembulatan — diuji dengan
      selisih detik.
- ☐ **`F-3` dan `S-5` dilaporkan sebagai "tidak dapat diuji setara"**, bukan sebagai lulus —
      diuji; keduanya tidak punya baseline Pega (`D-42`).
- ☐ Laporan per modul memuat **angka**: kasus dijalankan, setara, terpetakan, menunggu persetujuan.
- ☐ Gerbang 2: ditinjau **Lead Engineer**, dan keluarannya dibaca **Work Owner** pada satu modul
      percobaan sebelum dipakai untuk seluruh modul.

#### Dependency / Blocked by

`TKT-S8-001`. **Tidak terhalang keputusan** — `D-49` dan `D-54` sudah menetapkan aturannya lengkap.
Yang menahan hanyalah urutan pekerjaan.

#### Constraint keamanan, data, operasional

- Rekaman selisih memuat **data nasabah nyata** (staging memuat data produksi apa adanya, `D-64`).
  Laporan selisih **tidak boleh di-commit** dan tidak boleh dikirim ke luar tanpa penyamaran.
- **Perkakas ini tidak boleh melonggarkan penilaiannya sendiri.** Menambahkan toleransi agar lebih
  banyak kasus tampak setara akan membatalkan gunanya — dan kesetaraan yang dilaporkannya menjadi
  tidak berarti.
- Dua modul tanpa baseline **tidak boleh dihitung lulus** hanya karena tidak ada pembandingnya.

#### Migrasi skema / rollout / rollback

Menambah tabel hasil pembandingan di lingkungan uji. Tidak menyentuh skema klaim.

**Rollback:** mengembalikan versi perkakas. Persetujuan Work Owner yang sudah diberikan atas selisih
**tetap berlaku** dan tidak perlu diulang.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./cmd/s8/... -run TestKlasifikasi13ButirP5
go test ./cmd/s8/... -run TestSelisihDiLuarDaftarTidakPernahLolos
go test ./cmd/s8/... -run TestSelisihDesimalTerdeteksi
go test ./cmd/s8/... -run TestModulTanpaBaselineTidakDihitungLulus
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Selisih cocok 13 butir lolos otomatis; di luar itu wajib persetujuan tertulis | `D-54` |
| Perkakas wajib **mengklasifikasikan**, bukan hanya melaporkan | `D-54` bagian Konsekuensi |
| Daftar perbaikan `P-5` berjumlah 13 butir | `D-49` · `ADR-0017` · `R-19` |
| `F-3` dan `S-5` tidak punya baseline Pega | `D-42` bagian Pengecualian |
| Uji kesetaraan adalah gerbang pertama tiap modul | `BRD §21.1` |

#### Comments

# 30. U-1 — Kerangka SPA

*Modul Frontend · folder `docs/ticketing/U-1-Kerangka-Layar-dan-Navigasi-Portal/`*

## Spesifikasi Modul

| | |
|---|---|
| **Modul** | `U-1` Kerangka SPA |
| **Gelombang** | 2 — Kerangka UI |
| **Ukuran** | Sedang |
| **Bergantung pada** | `F-1` (penyajian SPA), `F-3` (autentikasi) |
| **Kesiapan** | **SEBAGIAN** — kerangka siap, peta rute terhalang artefak |
| **Cakupan tiket** | penuh untuk bagian yang buktinya lengkap; `needs-info` untuk peta rute (`D-41` Opsi 1) |

### Apa yang dibangun

Rangka aplikasi satu halaman: routing, tata letak, alur masuk, state global, dan penanganan galat
di sisi peramban.

### Yang mengikat modul ini

| Sumber | Isi |
|---|---|
| `ADR-0002` | React + TypeScript + Vite, SPA murni disajikan binary Go — tanpa runtime Node.js di produksi |
| `D-13` | Alur kerja, urutan langkah, penempatan field, dan tata letak **mengikuti Pega** |
| `ADR-0024` | Autentikasi lewat HCC/HCQ — kontraknya belum ada |
| `D-59` | Menu yang terlihat mengikuti izin peran; penyembunyian menu **bukan** pengamanan |

### Penghalang modul ini

| Jenis | Isi | Pemilik | Menahan |
|---|---|---|---|
| **Artefak** | **7 harness portal hilang dari export**: `InboxServiceCenter` · `InboxCloseClaim_Harness` · `InboxRequestSalvage` · `PNCViewClaim` · `ReportProduksiPA_harnes` · `InboxOutstanding_Harness` · `LostAdjuster_harness` | Tim Pega (`R-16`) | `TKT-U1-004` |
| **Keputusan** | **Ketujuh layar itu masih aktif di produksi?** Bila tidak, 7 rute SPA dihapus dari lingkup | Work Owner | `TKT-U1-004` |
| **Keputusan** | Dari **38 harness tanpa entri menu**, mana yang rute berdiri sendiri dan mana yang modal? `pyHarnessPurpose` **tidak ada** di export | Work Owner + Tim Pega | `TKT-U1-004` |
| **Artefak** | Kontrak API HCC/HCQ | Tim HCC/HCQ | `TKT-U1-002` sebagian |

### Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-U1-001](issues/01-kerangka-spa-routing-dan-layout.md) | Kerangka SPA: routing, tata letak, dan state global | `ready-for-human` | siap |
| [TKT-U1-002](issues/02-alur-masuk-dan-sesi-di-frontend.md) | Alur masuk dan penanganan sesi di frontend | `ready-for-human` | siap |
| [TKT-U1-003](issues/03-penanganan-galat-dan-notifikasi-global.md) | Penanganan galat dan notifikasi global | `ready-for-human` | siap |
| [TKT-U1-004](issues/04-peta-rute-dari-74-harness.md) | Peta rute dari 74 harness dan 51 item menu | `needs-info` | **terhalang artefak** |

## Daftar Tiket

### TKT-U1-001 — Kerangka SPA: routing, tata letak, dan state global

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | U-1 · Gelombang: 2 · Bergantung pada: TKT-F1-005 |
| **Requirement** | FR-U1 |
| **Keputusan** | D-13, D-23, D-09 |
| **ADR** | 0002 |
| **Risiko** | R-11 |
| **Rule Pega yang digantikan** | portal dan kerangka navigasi Pega — `Navigation/pyCaseWorkerNavigation-Navigation.xml` (**51 item menu**) dan **74 harness** |
| **Peran penguji gerbang 2** | **PncAdmin** dan **PncPICTeknik** (`D-60`) |
| **Label** | `modul::U-1` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::2` |
| **Milestone** | Gelombang 2 — Kerangka UI |
| **Berkas sumber** | `ticketing/U-1-Kerangka-Layar-dan-Navigasi-Portal/issues/01-kerangka-spa-routing-dan-layout.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Rangka aplikasi web yang dapat dijalankan, dengan routing, tata letak, dan tempat state global —
siap diisi layar oleh `U-3`, `U-4`, `U-5`, dan `U-6`.

Nilai bisnisnya sama dengan `F-1` di sisi backend: ia menetapkan **pola yang akan diikuti ratusan
layar**. `D-09` menetapkan tim sedang belajar React; pola yang tidak ditetapkan di awal berubah
menjadi banyak gaya berbeda, dan itu biaya yang dibayar selama masa hidup aplikasi.

#### Ruang lingkup

- Struktur folder frontend dan **aturan penamaan komponen**, ditulis preskriptif (`D-09`).
- Routing dengan rute bersarang: kerangka portal → daftar → detail.
- Tata letak baku: kepala, navigasi samping, area isi, dan jejak lokasi — **mengikuti susunan
  Pega** (`D-13`).
- **Navigasi yang dibangun dari izin peran**: menu yang tidak diizinkan **tidak dirender**.
- State global seperlunya: identitas pengguna, izin menu, dan notifikasi — bukan seluruh data
  aplikasi.
- Pola pengambilan data ke API dengan penanganan memuat dan galat yang seragam.

#### Non-goal

- **Tidak** membangun layar bisnis apa pun.
- **Tidak** membangun komponen tabel dan form — itu `U-2`.
- **Tidak** menjadikan penyembunyian menu sebagai pengamanan. Penegakan sebenarnya ada di server
  (`TKT-F3-005`); di frontend ia **kenyamanan tampilan**.

#### Acceptance criteria

- ☐ Aplikasi dapat dijalankan dan menampilkan kerangka portal dengan navigasi kosong yang
      dibangun dari daftar izin — diuji dengan dua daftar izin berbeda menghasilkan menu berbeda.
- ☐ Rute yang tidak diizinkan peran pengguna **tidak dapat dicapai** lewat URL langsung, dan
      menampilkan halaman "tidak berwenang" — **bukan** halaman kosong.
- ☐ Muat ulang halaman pada rute dalam (`/klaim/123/estimasi`) **tetap menampilkan halaman yang
      benar** — bergantung pada *fallback* `TKT-F1-005`.
- ☐ Struktur folder dan penamaan terdokumentasi cukup preskriptif untuk diikuti tanpa bertanya,
      dan **pemeriksaan otomatis** menolak berkas di luar pola.
- ☐ Pola pengambilan data menampilkan keadaan memuat, galat, dan kosong secara seragam — diuji
      pada satu rute contoh dengan ketiga keadaan.
- ☐ Bundel produksi ter-*build* dan **tersemat di binary Go** — dibuktikan dengan menjalankan
      binary dari direktori kosong (`TKT-F1-005`).
- ☐ Halaman dapat dipakai pada lebar 1366 px tanpa gulir horizontal pada tata letak utama.

#### Dependency / Blocked by

Bergantung pada `TKT-F1-005` (penyajian SPA dan *fallback* rute).

**Yang bergantung padanya:** `TKT-U2-005`, `TKT-U2-001`…`004`, dan seluruh layar `U-3`…`U-6`.

#### Constraint keamanan, data, operasional

- **Penyembunyian menu bukan pengamanan.** Setiap rute yang menampilkan data memanggil endpoint
  yang memeriksa kewenangan di server (`D-59`, `TKT-F3-005`).
- State global **tidak menyimpan data nasabah lebih lama dari yang dibutuhkan layar** — tidak ada
  cache persisten di peramban untuk data klaim.
- Token sesi tidak disimpan di tempat yang dapat dibaca skrip pihak ketiga.

#### Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** menjalankan binary versi sebelumnya — SPA tersemat
di binary, sehingga keduanya mundur bersama.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm ci && npm run build && npm run test
npm run test -- Routing.izin
npm run test:e2e -- --grep "muat ulang rute dalam"
go build ./... && (mkdir -p /tmp/kosong && cd /tmp/kosong && /path/app)   # SPA tersaji
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 74 harness, 269 section; 51 item menu | `docs/Steering/06-MODULE-BREAKDOWN.md` §4 · `Navigation/pyCaseWorkerNavigation-Navigation.xml` |
| React + TypeScript + Vite, SPA murni tersemat di binary | `D-23` · `ADR-0002` |
| Tata letak mengikuti Pega agar tanpa pelatihan ulang | `D-13` |
| Struktur preskriptif karena tim sedang belajar | `D-09` |
| Otorisasi ditegakkan di server, bukan dengan menyembunyikan menu | `D-59` · `ADR-0023` |

#### Comments

### TKT-U1-002 — Alur masuk dan penanganan sesi di frontend

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | U-1 · Gelombang: 2 · Bergantung pada: TKT-U1-001, TKT-F3-003 |
| **Requirement** | FR-U1, FR-F3 |
| **Keputusan** | D-07, D-27 |
| **ADR** | 0024 |
| **Risiko** | R-14 |
| **Rule Pega yang digantikan** | layar masuk portal Pega — **tidak ada rule aplikasi** yang mengaturnya; HCC/HCQ muncul hanya sebagai 2 teks pesan galat |
| **Peran penguji gerbang 2** | **seluruh peran** — ini layar yang dilalui semua orang (`D-60`) |
| **Label** | `modul::U-1` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::2` |
| **Milestone** | Gelombang 2 — Kerangka UI |
| **Berkas sumber** | `ticketing/U-1-Kerangka-Layar-dan-Navigasi-Portal/issues/02-alur-masuk-dan-sesi-di-frontend.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Layar masuk, penyimpanan sesi di peramban, dan perilaku yang benar ketika sesi habis di tengah
pekerjaan.

Nilai bisnisnya ada pada butir terakhir. Pengguna sistem klaim mengisi form panjang; sesi yang
habis di tengah pengisian **tanpa peringatan** berarti pekerjaan hilang — dan itu keluhan yang
paling cepat menghapus kepercayaan pada sistem baru.

#### Ruang lingkup

- Layar masuk dengan penanganan tiga jenis galat yang **dibedakan**: kredensial salah, pengguna
  tidak aktif, dan **sistem identitas tidak dapat dihubungi**.
- Penyimpanan sesi di peramban dan pengirimannya pada setiap permintaan.
- **Peringatan sebelum sesi habis**, dan perilaku saat sesi ditolak server di tengah pekerjaan:
  pengguna diarahkan ke layar masuk **dengan isian yang belum tersimpan tidak hilang begitu saja**.
- Keluar (logout) yang menghapus sesi di server, bukan hanya di peramban.

#### Non-goal

- **Tidak** memvalidasi kredensial — itu server (`TKT-F3-002`).
- **Tidak** menyimpan kata sandi di peramban dalam bentuk apa pun.

#### Acceptance criteria

- ☐ Ketiga jenis galat masuk menampilkan pesan **berbeda dan dapat ditindaklanjuti** — diuji.
- ☐ Galat kredensial **tidak membedakan** apakah pengguna ada atau tidak — diuji dengan pengguna
      yang tidak ada dan pengguna yang ada berkata sandi salah: **pesannya sama**.
- ☐ Sesi yang ditolak server di tengah permintaan mengarahkan ke layar masuk, dan setelah masuk
      kembali pengguna **kembali ke halaman terakhir** — diuji.
- ☐ Peringatan muncul sebelum sesi habis, dengan pilihan memperpanjang — diuji dengan masa
      berlaku pendek.
- ☐ Keluar menghapus sesi di server — diuji dengan memakai token lama setelah keluar: **ditolak**.
- ☐ Token sesi **tidak muncul** di URL, di log peramban, maupun di pesan galat — diuji.

#### Dependency / Blocked by

Bergantung pada `TKT-U1-001` dan `TKT-F3-003`.

**Catatan:** tiket ini **tidak terhalang** ketiadaan kontrak HCC/HCQ. Selama pengembangan ia
memakai adapter fake `TKT-F3-001`; yang berubah kelak adalah sisi server, bukan layar ini. Yang
**terhalang** hanyalah pesan galat yang spesifik terhadap kode galat HCC/HCQ.

#### Constraint keamanan, data, operasional

- Galat autentikasi **tidak boleh membocorkan keberadaan akun**.
- Bila HCC/HCQ tidak dapat dihubungi, perilaku aplikasi mengikuti keputusan Work Owner yang
  **masih terbuka** (`ADR-0024`). Sampai itu diputuskan, layar menampilkan pesan "sistem identitas
  tidak dapat dihubungi" dan **tidak** menawarkan jalur lain.
- Isian yang belum tersimpan **tidak dikirim ke mana pun** saat sesi habis — ia tetap di memori
  peramban sampai pengguna menyimpannya.

#### Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** menjalankan bundel sebelumnya; pengguna mungkin
perlu masuk ulang.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm run test -- Masuk
npm run test -- Sesi.habis
npm run test:e2e -- --grep "sesi habis di tengah form"
npm run test:e2e -- --grep "keluar mencabut sesi"
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Aplikasi menerbitkan sesi sendiri; pengguna tetap bekerja bila HCC/HCQ bermasalah | `D-07` · `docs/Steering/11-SECURITY.md` §2.1 |
| HCC/HCQ nol jejak di export | `T-2` · `ADR-0024` |
| Perilaku saat HCC/HCQ tidak dapat dihubungi belum diputuskan | `ADR-0024` Pertanyaan terbuka |
| Form panjang bernested repeat | `docs/Steering/01-FRONTEND-ANALYSIS.md` |
| Stateless; sesi dikenali lintas instans | `D-27` · `TKT-F3-003` |

#### Comments

### TKT-U1-003 — Penanganan galat dan notifikasi global

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | U-1 · Gelombang: 2 · Bergantung pada: TKT-U1-001 |
| **Requirement** | FR-U1 |
| **Keputusan** | D-13 |
| **ADR** | 0002 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | penyajian galat Pega yang tersebar di `Section/` dan `Harness/` — tidak ada pola terpusat yang dapat dirujuk |
| **Peran penguji gerbang 2** | **PncAdmin** (`D-60`) |
| **Label** | `modul::U-1` `tipe::fondasi` `status::ready-for-human` `prioritas::sedang` `gelombang::2` |
| **Milestone** | Gelombang 2 — Kerangka UI |
| **Berkas sumber** | `ticketing/U-1-Kerangka-Layar-dan-Navigasi-Portal/issues/03-penanganan-galat-dan-notifikasi-global.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu cara menampilkan galat dan pemberitahuan di seluruh aplikasi, sehingga pengguna belajar
sekali dan berlaku di semua layar.

Nilai bisnisnya paling terasa pada layar registrasi, yang menghadapkan pengguna pada belasan
aturan validasi sekaligus. Galat yang tampil dengan cara berbeda di setiap layar membuat pengguna
harus menebak apa yang salah — dan itu bertentangan dengan `D-13`, yang justru mempertahankan
tata letak agar **tidak ada pelatihan ulang**.

#### Ruang lingkup

- Penangkap galat tingkat aplikasi yang mencegah satu galat merender **halaman kosong**.
- Pemetaan kontrak galat API (`TKT-F1-004`) ke tampilan: galat field, galat form, galat halaman,
  dan galat sistem.
- Komponen pemberitahuan (toast/banner) untuk hasil tindakan: berhasil, peringatan, gagal.
- Penampilan **pengenal permintaan** pada galat sistem, agar keluhan pengguna dapat ditelusuri ke
  log (`TKT-F1-003`).

#### Non-goal

- **Tidak** menentukan teks galat per aturan bisnis — itu milik modul bisnisnya.
- **Tidak** mengirim laporan galat ke layanan pihak ketiga.

#### Acceptance criteria

- ☐ Galat tak tertangani pada satu komponen **tidak membuat seluruh halaman kosong** — diuji
      dengan komponen yang sengaja melempar galat.
- ☐ Keempat jenis galat ditampilkan berbeda dan **dapat dibedakan** pengguna — diuji keempatnya.
- ☐ Galat sistem menampilkan **pengenal permintaan** yang dapat disalin pengguna, dan pengenal
      itu **sama** dengan yang ada di log server — diuji ujung ke ujung.
- ☐ Galat **tidak pernah** menampilkan stack trace, nama tabel, atau potongan SQL — diuji dengan
      galat database yang sengaja dipicu.
- ☐ Pemberitahuan berhasil hilang sendiri; pemberitahuan gagal **bertahan** sampai ditutup
      pengguna — diuji.
- ☐ Pemberitahuan dapat dibaca pembaca layar — diuji aksesibilitas.

#### Dependency / Blocked by

Bergantung pada `TKT-U1-001`. Bentuk galat mengikuti `TKT-F1-004` yang berstatus `needs-info` —
**bentuk kontraknya sudah cukup jelas** untuk dipakai; yang belum diputuskan di sana adalah
perilaku baku 720 activity, bukan bentuk responsnya.

#### Constraint keamanan, data, operasional

- Pesan galat yang sampai ke pengguna **tidak boleh membocorkan struktur internal**.
- Pengenal permintaan yang ditampilkan **bukan** data sensitif — ia justru dirancang untuk
  dibagikan pengguna saat melapor.
- Pemberitahuan **tidak menampilkan data nasabah** di luar konteks layar yang memang menampilkannya.

#### Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** menjalankan bundel sebelumnya.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm run test -- Galat
npm run test -- Notifikasi
npm run test:e2e -- --grep "request id terlihat di galat sistem"
npm run test:a11y -- --grep "notifikasi"
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Bentuk kontrak galat API | `docs/Steering/10-API-STRATEGY.md` §3 · `TKT-F1-004` |
| ID permintaan di log dan respons | `docs/Steering/12-CROSSCUTTING.md` §2 |
| Tata letak mengikuti Pega; tanpa pelatihan ulang | `D-13` |
| Registrasi adalah gerbang validasi terberat | `docs/Steering/06-MODULE-BREAKDOWN.md` `B-2` |

#### Comments

### TKT-U1-004 — Peta rute dari 74 harness dan 51 item menu

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang artefak dan keputusan** |
| **Modul** | U-1 · Gelombang: 2 · Bergantung pada: TKT-U1-001 |
| **Requirement** | FR-U1 |
| **Keputusan** | D-13, D-58 |
| **ADR** | 0002, 0023 |
| **Risiko** | R-16 |
| **Rule Pega yang digantikan** | **74 harness** dan `Navigation/pyCaseWorkerNavigation-Navigation.xml` (**51 item menu**) |
| **Peran penguji gerbang 2** | **seluruh 22 peran** — peta ini menentukan apa yang masing-masing lihat (`D-60`) |
| **Label** | `modul::U-1` `tipe::analisis` `status::needs-info` `prioritas::tinggi` `gelombang::2` |
| **Milestone** | Gelombang 2 — Kerangka UI |
| **Berkas sumber** | `ticketing/U-1-Kerangka-Layar-dan-Navigasi-Portal/issues/04-peta-rute-dari-74-harness.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Daftar tetap rute SPA beserta pemetaannya ke harness lama dan ke izin menu — dasar bagi seluruh
layar `U-3`, `U-4`, `U-5`, dan `U-6`.

Nilai bisnisnya: tanpa peta ini, jumlah layar yang harus dibangun **tidak diketahui**. Dan angka
itulah yang menentukan ukuran empat modul frontend sekaligus.

#### Ruang lingkup

- Pemetaan **74 harness → rute SPA**, dengan pembedaan: rute berdiri sendiri, modal di atas rute
  lain, atau tidak dibawa.
- Pemetaan **rute → izin menu** (`TKT-F3-004`), sehingga navigasi dibangun dari izin.
- Penandaan eksplisit untuk harness yang **hilang dari export** dan untuk yang statusnya belum
  diputuskan.
- Hasilnya memperbarui ukuran `U-3`, `U-4`, `U-5`, `U-6` di `06-MODULE-BREAKDOWN.md` — diajukan
  sebagai perubahan, bukan diterapkan diam-diam.

#### Non-goal

- **Tidak** membangun layar apa pun.
- **Tidak** merancang ulang navigasi. `D-13` menetapkan susunan mengikuti Pega.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **7 harness portal hilang dari export**: `InboxServiceCenter` · `InboxCloseClaim_Harness` · `InboxRequestSalvage` · `PNCViewClaim` · `ReportProduksiPA_harnes` · `InboxOutstanding_Harness` · `LostAdjuster_harness` | **Tim Pega** (`R-16`) | Tujuh layar tidak dapat dipetakan isinya sama sekali |
| **Apakah ketujuh layar itu masih aktif di produksi?** | **Work Owner** | Bila tidak, **7 rute dihapus dari lingkup** — dan itu memperkecil `U-3` dan `U-5` |
| **Dari 38 harness tanpa entri menu, mana rute berdiri sendiri dan mana modal?** `pyHarnessPurpose` **tidak ada** di export | **Work Owner + Tim Pega** | Menentukan apakah 38 layar itu menambah rute atau tidak — selisih terbesar dalam estimasi frontend |
| **5 When rule peran hilang** yang masing-masing mengendalikan satu item menu | **Tim Pega** (`R-16`) | 5 dari 51 item menu tidak diketahui siapa yang boleh membukanya (`TKT-F3-004`) |

#### Acceptance criteria

> Beberapa butir belum dapat diangkakan sampai artefak dan keputusan di atas tersedia.

- ☐ Setiap dari **74 harness** punya satu baris pemetaan: nama harness, rute SPA (atau
      **"tidak dibawa"**), jenis (rute/modal), modul pemilik, dan izin menu yang dibutuhkan.
- ☐ **7 harness yang hilang ditandai eksplisit** sebagai belum diketahui — **bukan** dipetakan
      dengan tebakan.
- ☐ **38 harness tanpa entri menu** diklasifikasikan, dan jumlah yang masih belum jelas
      dilaporkan sebagai angka.
- ☐ Jumlah rute akhir dilaporkan, dan perbedaannya terhadap 74 dijelaskan baris per baris.
- ☐ Setiap rute punya izin menu yang menaunginya; rute tanpa izin **menggagalkan pemeriksaan**
      — tidak ada rute yang terbuka tanpa sengaja.
- ☐ Usulan pembaruan ukuran `U-3`…`U-6` disiapkan beserta angka barunya.

#### Dependency / Blocked by

Bergantung pada `TKT-U1-001` dan `TKT-F3-004`. **Terhalang Tim Pega dan Work Owner.**

**Yang terhalang olehnya:** estimasi dan penulisan tiket `U-3`, `U-4`, `U-5`, `U-6`.

#### Constraint keamanan, data, operasional

- Setiap rute yang menampilkan data **wajib** punya izin menu yang menaunginya, dan penegakannya
  ada di server (`TKT-F3-005`) — bukan di peta ini.
- Peta ini menyebut nama harness dan nama rule; **tidak** menyebut data nasabah.

#### Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema — tiket ini menghasilkan dokumen dan konfigurasi rute.

**Rollback:** tidak berlaku.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/inventaris-harness "Harness/" > peta-harness.csv
wc -l peta-harness.csv                       # HARUS 74 baris + header
go run ./cmd/tools/banding-menu peta-harness.csv "Navigation/pyCaseWorkerNavigation-Navigation.xml"
npm run test -- Routing.setiapRutePunyaIzin
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 74 harness, 269 section | `docs/Steering/06-MODULE-BREAKDOWN.md` §4 |
| 51 item menu di `pyCaseWorkerNavigation` | `D-58` · `ADR-0023` |
| 7 harness portal hilang | `docs/verifikasi-bukti-adr.md` §15 baris `U-1` · `R-16` |
| 38 harness tanpa entri menu; `pyHarnessPurpose` tidak ada | idem |
| 5 When rule peran hilang | `D-58` · `R-16` |

#### Comments

# 31. U-2 — Pustaka Komponen

*Modul Frontend · folder `docs/ticketing/U-2-Komponen-Layar-Baku/`*

## Spesifikasi Modul

| | |
|---|---|
| **Modul** | `U-2` Pustaka Komponen |
| **Gelombang** | 2 — Kerangka UI |
| **Ukuran** | **Sedang** (turun dari Besar — lihat di bawah) |
| **Bergantung pada** | `U-1` untuk dipakai; **dapat dimulai paralel** |
| **Kesiapan** | **PENUH** — satu-satunya modul tanpa penghalang artefak maupun keputusan yang menahan |
| **Cakupan tiket** | **penuh** (`D-41` Opsi 1) |

### Apa yang dibangun

Komponen antarmuka baku yang dipakai berulang oleh seluruh layar: tabel, form, unggah berkas,
pemilih tanggal, dan pemformatan angka serta tanggal.

### Kenapa modul ini paling berdaya ungkit

**268 dari 269 section** sistem lama memakai pola grid yang sama. Satu komponen tabel yang benar
dipakai ratusan kali, dan satu perbaikan di dalamnya memperbaiki seluruh layar. Melewatkannya
berarti 268 implementasi tabel yang berbeda-beda — persis kegagalan yang `D-09` peringatkan pada
tim yang sedang belajar teknologi baru.

### Koreksi ukuran yang mengubah rencana

| Anggapan v1.0 | Terverifikasi |
|---|---|
| Tabel baku **18–27 kolom** | **median 6 kolom** |
| Grid berat karena banyak fitur | **tiga fitur termahal tidak dipakai sama sekali**: tambah baris inline, hapus baris inline, resize kolom |
| Masalahnya `OFFSET` besar | **`OFFSET` nol kemunculan**; masalah nyatanya **3.189 grid terikat page list klipboard** dan `pyMaxRecords=500` pada **54 dari 56** laporan |

Akibatnya `U-2` **turun dari Besar menjadi Sedang**, dan alasan memilih pustaka tabel kelas berat
ikut melemah — itulah yang membuat `TKT-U2-005` ada.

### Dua pertanyaan terbuka yang **tidak** menahan tiket mana pun

Keduanya memperluas atau mempersempit lingkup, bukan memblokirnya:

| Pertanyaan | Pemilik | Bila jawabannya "ya" |
|---|---|---|
| Grid 39 kolom dan 25 kolom — seluruh kolomnya benar-benar dilihat pengguna? | Work Owner | `TKT-U2-001` menambah fitur pemilih kolom |
| Tambah dan hapus baris inline diinginkan di sistem baru? | Work Owner | `TKT-U2-001` bertambah; hari ini **tidak dipakai sama sekali** di sistem lama |

### Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-U2-005](issues/05-pemilihan-pustaka-tabel.md) | Pemilihan pustaka tabel — TanStack Table versus AG Grid | `ready-for-human` | siap · **dikerjakan lebih dulu** |
| [TKT-U2-001](issues/01-komponen-tabel-baku.md) | Komponen tabel baku dengan paginasi keyset server-side | `ready-for-human` | siap |
| [TKT-U2-002](issues/02-komponen-form-baku.md) | Komponen form baku dan penyajian galat validasi | `ready-for-human` | siap |
| [TKT-U2-003](issues/03-komponen-unggah-berkas.md) | Komponen unggah berkas | `ready-for-human` | siap |
| [TKT-U2-004](issues/04-pemformatan-tanggal-dan-uang.md) | Pemilih tanggal dan pemformatan tanggal serta uang terpusat | `ready-for-human` | siap |

## Daftar Tiket

### TKT-U2-001 — Komponen tabel baku dengan paginasi keyset server-side

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | U-2 · Gelombang: 2 · Bergantung pada: TKT-U2-005, TKT-U1-001 |
| **Requirement** | FR-U2 |
| **Keputusan** | D-13, D-23 |
| **ADR** | 0002 |
| **Risiko** | R-11 |
| **Rule Pega yang digantikan** | pola grid pada **268 dari 269 section** — di antaranya **3.189 grid terikat page list klipboard** |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi UI (`D-60`); perilakunya diuji pengguna lewat `U-3` dan `U-4` |
| **Label** | `modul::U-2` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::2` |
| **Milestone** | Gelombang 2 — Kerangka UI |
| **Berkas sumber** | `ticketing/U-2-Komponen-Layar-Baku/issues/01-komponen-tabel-baku.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu komponen tabel yang dipakai seluruh layar daftar, dengan paginasi, pengurutan, dan
penyaringan yang **dikerjakan di server**.

Nilai bisnisnya berlipat: satu komponen dipakai ratusan kali, dan satu perbaikan di dalamnya
memperbaiki seluruh layar. Ia juga tempat satu-satunya yang menegakkan aturan keamanan daftar —
nama kolom sort dan filter hanya dari daftar putih (`TKT-F2-007`), bukan dari apa yang dikirim
peramban.

#### Ruang lingkup

- Komponen tabel dengan: paginasi **keyset** server-side, pengurutan server-side, penyaringan
  server-side, keadaan kosong, keadaan memuat, dan keadaan galat.
- Kolom dideklarasikan sebagai data, bukan JSX berulang — sehingga 268 layar mendeklarasikan
  kolom, bukan menulis ulang tabel.
- Pemformatan sel memakai pemformat terpusat `TKT-U2-004` — **tidak ada** pemformatan tanggal
  atau uang yang ditulis ulang per layar.
- Perilaku pada **grid lebar** (39 kolom): gulir horizontal yang benar dengan kolom identitas yang
  tetap terlihat.

#### Non-goal

- **Tidak** menambah baris atau menghapus baris **inline** — ketiga fitur itu **tidak dipakai
  sama sekali** di sistem lama, dan menambahkannya berarti menambah kemampuan yang tidak diminta.
  Bila Work Owner menginginkannya, itu tiket baru.
- **Tidak** membangun layar apa pun — itu `U-3`, `U-4`, `U-5`.
- **Tidak** melakukan penyaringan di sisi klien atas data yang sudah diambil.

#### Acceptance criteria

- ☐ Tabel menampilkan 10.000 baris lewat paginasi server-side **tanpa memuat seluruhnya** —
      diuji dengan memeriksa jumlah baris yang diminta per permintaan.
- ☐ Paginasi memakai **keyset**, bukan `OFFSET` — diuji dengan memeriksa parameter permintaan
      halaman kedua memuat penanda baris terakhir, bukan angka lompatan.
- ☐ Permintaan sort pada kolom **di luar daftar putih** ditolak dan ditampilkan sebagai galat
      yang dapat dibaca pengguna — bukan tabel kosong tanpa penjelasan.
- ☐ Keadaan kosong, memuat, dan galat masing-masing punya tampilan yang berbeda dan **dapat
      dibedakan** — diuji ketiganya.
- ☐ Grid **39 kolom** dapat digulir horizontal dengan kolom identitas tetap terlihat — diuji
      pada lebar layar 1366 px.
- ☐ Mendeklarasikan tabel baru untuk 6 kolom membutuhkan **≤ 30 baris kode** — dibuktikan
      dengan satu contoh nyata.
- ☐ Tidak ada pemformatan tanggal atau uang di dalam komponen layar — diuji pemindaian:
      pemanggilan pemformat hanya lewat `TKT-U2-004`.

#### Dependency / Blocked by

- Bergantung pada `TKT-U2-005` (pustaka tabel harus dipilih lebih dulu).
- Bergantung pada `TKT-U1-001` (kerangka SPA).
- Bergantung secara kontrak pada `TKT-F2-007` (daftar putih kolom) — komponen ini yang
  menegakkannya di sisi klien; server tetap menjadi penegak sebenarnya.

#### Constraint keamanan, data, operasional

- **Penyaringan dan pengurutan tidak pernah dikerjakan di klien** atas data yang sudah diambil —
  itu membocorkan baris yang seharusnya tidak terlihat lewat jumlah baris (batas data `F-3`).
- Nama kolom database **tidak boleh muncul** di parameter URL maupun di DOM.
- Menghapus batas 500 baris sistem lama adalah **penambahan kemampuan**, bukan penyalinan —
  kapasitas maksimum satu daftar mengikuti keputusan `ADR-0011` yang masih terbuka.

#### Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** mengembalikan versi komponen; layar yang
memakainya ikut kembali karena hanya ada satu implementasi.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm run test -- DataTable
npm run test -- DataTable.keyset
npm run test:e2e -- --grep "tabel 39 kolom"
npx bundlesize   # ukuran bundel setelah komponen ditambahkan
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 268 dari 269 section bergrid | `docs/Steering/06-MODULE-BREAKDOWN.md` §4 |
| 3.189 grid terikat page list klipboard | `T-12` · `docs/Steering/09-DATABASE-STRATEGY.md` §6.3 |
| Tambah/hapus baris inline dan resize **tidak dipakai** | `T-11` |
| Median 6 kolom; kasus terberat 39 dan 25 kolom | `T-11` · `docs/verifikasi-bukti-adr.md` §15 |
| Paginasi keyset adalah perubahan perilaku | `docs/Steering/09-DATABASE-STRATEGY.md` §6.3 |
| Tata letak mengikuti Pega | `D-13` |

#### Comments

### TKT-U2-002 — Komponen form baku dan penyajian galat validasi

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | U-2 · Gelombang: 2 · Bergantung pada: TKT-U1-001 |
| **Requirement** | FR-U2 |
| **Keputusan** | D-13, D-09 |
| **ADR** | 0002 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | pola form pada `Section/` dan `Harness/` — form panjang bernested repeat, terberat registrasi klaim (`InputRegister_act`, **137 step validasi**) |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi UI (`D-60`) |
| **Label** | `modul::U-2` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::2` |
| **Milestone** | Gelombang 2 — Kerangka UI |
| **Berkas sumber** | `ticketing/U-2-Komponen-Layar-Baku/issues/02-komponen-form-baku.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu cara membangun form dan **satu cara menampilkan galat validasi**, dipakai seluruh layar
transaksi.

Nilai bisnisnya terletak pada galat. Registrasi klaim adalah gerbang validasi terberat di seluruh
sistem; pengguna di sana menghadapi belasan aturan sekaligus. Bila setiap layar menampilkan galat
dengan caranya sendiri, pengguna harus belajar ulang di setiap layar — dan `D-13` justru
menetapkan alur dan tata letak dipertahankan agar **tidak ada pelatihan ulang**.

#### Ruang lingkup

- Komponen field baku: teks, angka, uang, tanggal, pilihan, pencarian, dan area teks.
- **Satu pola penyajian galat**: galat per field ditampilkan di field-nya, galat lintas field
  ditampilkan di kepala form, dan keduanya **menunjuk ke field yang salah**.
- Pola form bernested repeat — objek pertanggungan berisi coverage berisi settlement line —
  karena itulah bentuk nyata layar registrasi.
- Keadaan **sedang menyimpan** dan pencegahan kiriman ganda.

#### Non-goal

- **Tidak** menulis aturan validasi bisnis — itu milik modul bisnisnya, dan **server tetap
  penegak sebenarnya**. Validasi di klien hanya kenyamanan.
- **Tidak** merancang ulang tata letak layar (`D-13`).

#### Acceptance criteria

- ☐ Galat dari server yang menyebut nama field **otomatis muncul di field itu** — diuji dengan
      respons galat berbentuk kontrak `TKT-F1-004`.
- ☐ Galat lintas field muncul di kepala form dan menautkan ke field pertama yang salah — diuji.
- ☐ Menekan simpan dua kali cepat **hanya mengirim satu permintaan** — diuji.
- ☐ Form bernested tiga tingkat dapat menambah dan menghapus baris pada tingkat terdalam tanpa
      kehilangan isian tingkat di atasnya — diuji.
- ☐ Field uang menolak masukan bukan angka dan **tidak membulatkan** nilai yang diketik —
      pembulatan hanya saat tampil (`ADR-0016`).
- ☐ Seluruh field dapat dioperasikan dengan papan ketik saja — diuji pada satu form penuh.

#### Dependency / Blocked by

Bergantung pada `TKT-U1-001`. Bentuk galat mengikuti kontrak `TKT-F1-004`, yang berstatus
`needs-info` — **tetapi bentuk kontraknya sudah cukup jelas** untuk dipakai; yang belum diputuskan
di sana adalah perilaku baku 720 activity, bukan bentuk responsnya.

#### Constraint keamanan, data, operasional

- **Validasi klien bukan pengaman.** Server memvalidasi ulang seluruhnya; klien hanya mempercepat
  umpan balik.
- Nilai uang **tidak dibulatkan saat diketik maupun dikirim** — presisi penuh disimpan, pembulatan
  hanya saat ditampilkan (`D-51`, `ADR-0016`).
- Angka pecahan di JavaScript **tidak presisi** — nilai uang tidak boleh melewati batas itu sebagai
  angka; dikirim dan diterima sebagai string desimal.

#### Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** mengembalikan versi komponen.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm run test -- Form
npm run test -- Form.galat
npm run test -- Form.nested
npm run test:a11y -- --grep "form registrasi"
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Registrasi adalah gerbang validasi terberat (137 step) | `docs/Steering/06-MODULE-BREAKDOWN.md` `B-2` |
| Tata letak dan alur mengikuti Pega; tanpa pelatihan ulang | `D-13` |
| Form panjang bernested repeat | `docs/Steering/01-FRONTEND-ANALYSIS.md` |
| Uang disimpan presisi penuh, dibulatkan saat tampil | `D-51` · `ADR-0016` |
| Bentuk kontrak galat API | `docs/Steering/10-API-STRATEGY.md` §3 |

#### Comments

### TKT-U2-003 — Komponen unggah berkas

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | U-2 · Gelombang: 2 · Bergantung pada: TKT-U1-001, TKT-U2-002 |
| **Requirement** | FR-U2 |
| **Keputusan** | D-16 |
| **ADR** | 0010 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | pola unggah pada `Section/` dokumen klaim dan foto survei; **tiga mekanisme penyimpanan lama** yang disatukan menjadi satu jalur |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi UI (`D-60`); perilakunya diuji pengguna lewat `S-1` dan `B-8` |
| **Label** | `modul::U-2` `tipe::fondasi` `status::ready-for-human` `prioritas::sedang` `gelombang::2` |
| **Milestone** | Gelombang 2 — Kerangka UI |
| **Berkas sumber** | `ticketing/U-2-Komponen-Layar-Baku/issues/03-komponen-unggah-berkas.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu komponen unggah berkas yang dipakai dokumen klaim maupun foto survei, dengan umpan balik
kemajuan dan penanganan kegagalan yang jelas.

Nilai bisnisnya: unggah dokumen adalah titik yang **paling sering gagal** karena melibatkan sistem
lain (`ADR-0010`). Kegagalan yang tidak dijelaskan membuat petugas mengulang unggahan berkali-kali
dan meninggalkan berkas yatim di storage.

#### Ruang lingkup

- Komponen unggah: pilih berkas, seret-lepas, kemajuan, batal, dan hapus sebelum disimpan.
- Batas jenis dan ukuran berkas **dari konfigurasi**, bukan konstanta di kode.
- Penanganan kegagalan yang membedakan: berkas ditolak (jenis/ukuran), gagal jaringan, dan
  **storage tidak dapat dihubungi** — ketiganya butuh tindakan berbeda dari pengguna.
- Dukungan **peramban ponsel** untuk kasus foto survei lapangan (`D-12`).

#### Non-goal

- **Tidak** membangun alur dokumen klaim — itu `S-1`.
- **Tidak** memutuskan penyimpanan — dokumen disimpan lewat API storage internal (`ADR-0010`);
  komponen ini hanya mengirimkannya.
- **Tidak** menangani berkas yatim — pembersihannya milik `S-1`, dan mekanismenya belum dirancang.

#### Acceptance criteria

- ☐ Berkas dengan jenis di luar daftar yang diizinkan **ditolak sebelum diunggah**, dengan pesan
      yang menyebut jenis yang diizinkan.
- ☐ Berkas melebihi batas ukuran ditolak dengan pesan yang **menyebut batasnya dalam MB**.
- ☐ Kemajuan unggah terlihat dan **dapat dibatalkan**; pembatalan menghentikan permintaan,
      bukan hanya menyembunyikan indikatornya — diuji dengan memeriksa permintaan jaringan.
- ☐ Ketiga jenis kegagalan menghasilkan pesan yang **berbeda dan dapat ditindaklanjuti** —
      diuji dengan tiga skenario tiruan.
- ☐ Unggah dari peramban ponsel (lebar 390 px) dapat memakai kamera — diuji pada satu perangkat
      atau emulator.
- ☐ Batas jenis dan ukuran dibaca dari konfigurasi — diuji dengan mengubah nilainya dan
      memeriksa perilaku berubah tanpa mengubah kode.

#### Dependency / Blocked by

Bergantung pada `TKT-U1-001` dan `TKT-U2-002`.

#### Constraint keamanan, data, operasional

- Nama berkas dari pengguna **tidak pernah dipakai apa adanya** sebagai jalur penyimpanan.
- Komponen **tidak menyimpan berkas di peramban** lebih lama dari yang dibutuhkan untuk unggah.
- Dokumen klaim memuat **data nasabah** — pratinjau di klien tidak boleh dicache oleh peramban
  secara persisten.
- **Tidak ada atomisitas** antara berkas dan metadata (`ADR-0010`): bila metadata gagal disimpan
  setelah berkas terkirim, komponen wajib memberi tahu pengguna dengan jelas, bukan diam.

#### Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** mengembalikan versi komponen.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm run test -- Unggah
npm run test -- Unggah.kegagalan
npm run test:e2e -- --grep "unggah dokumen" --device "iPhone 12"
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Dokumen lewat satu jalur API storage internal; DB hanya metadata | `D-16` · `ADR-0010` |
| Tiga mekanisme penyimpanan lama disatukan | `D-16` |
| Tidak ada atomisitas berkas dan metadata | `ADR-0010` Negatif/utang teknis |
| Pemakaian lapangan untuk survei | `D-12` · `docs/Steering/06-MODULE-BREAKDOWN.md` `B-8` |

#### Comments

### TKT-U2-004 — Pemilih tanggal dan pemformatan tanggal serta uang terpusat

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | U-2 · Gelombang: 2 · Bergantung pada: TKT-U1-001 |
| **Requirement** | FR-U2 |
| **Keputusan** | D-51, D-49 butir 3 |
| **ADR** | 0016, 0017 |
| **Risiko** | R-12 |
| **Rule Pega yang digantikan** | pemformatan yang tersebar — **411 pemakaian `TO_CHAR`** di SQL yang mengembalikan tanggal sebagai string `'dd/mm/yyyy'` |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi UI (`D-60`) |
| **Label** | `modul::U-2` `tipe::fondasi` `status::ready-for-human` `prioritas::tinggi` `gelombang::2` |
| **Milestone** | Gelombang 2 — Kerangka UI |
| **Berkas sumber** | `ticketing/U-2-Komponen-Layar-Baku/issues/04-pemformatan-tanggal-dan-uang.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu tempat yang menentukan **bagaimana tanggal dan uang terlihat**, dipakai seluruh layar dan
seluruh laporan.

Nilai bisnisnya: `ADR-0016` menetapkan uang **disimpan presisi penuh dan dibulatkan hanya saat
ditampilkan**. Konsekuensinya, angka yang tersimpan tidak selalu sama dengan angka yang dilihat
pengguna — dan itu **aman hanya bila pembulatan tampilan dilakukan di satu tempat**. Satu layar
yang membulatkan dengan caranya sendiri akan menampilkan angka berbeda untuk data yang sama, dan
itu jenis laporan bug yang paling mahal ditelusuri.

#### Ruang lingkup

- Fungsi pemformat tunggal untuk: tanggal, tanggal+waktu, nilai uang Rupiah, nilai uang valuta
  asing, dan persentase share reasuransi.
- Komponen **pemilih tanggal** yang mengirim tanggal dalam bentuk yang tidak ambigu, dan
  menampilkannya dalam WIB.
- Aturan pembulatan tampilan: **berapa desimal** untuk Rupiah, untuk valuta asing, dan untuk
  persentase share.
- Pemeriksaan otomatis: pemformatan tanggal atau uang **di luar** modul ini menggagalkan build.

#### Non-goal

- **Tidak** melakukan perhitungan uang — itu milik server; klien hanya menampilkan.
- **Tidak** menangani konversi kurs — itu `B-5` dan master kurs `F-4`.

#### Acceptance criteria

- ☐ Pemformatan tanggal dan uang berada di **tepat satu modul** — diuji pemindaian:
      `toLocaleString`, `Intl.NumberFormat`, dan pemformatan manual **0 kemunculan** di luar modul
      itu.
- ☐ Nilai uang diterima sebagai **string desimal** dari API dan ditampilkan tanpa kehilangan
      presisi — diuji dengan nilai 15 digit dan 4 desimal.
- ☐ Pembulatan hanya terjadi saat menampilkan; nilai yang dikirim kembali ke server **sama
      persis** dengan yang diterima bila pengguna tidak mengubahnya — diuji.
- ☐ Tanggal ditampilkan dalam WIB dan dikirim dalam bentuk tak ambigu — diuji dengan menjalankan
      peramban pada dua zona waktu berbeda dan hasilnya **sama**.
- ☐ Persentase share ditampilkan dengan **4 desimal**, konsisten dengan toleransi validasi
      `ROUND(SUM(share),4) BETWEEN 99.9999 AND 100.0001`.
- ☐ Nilai kosong, nol, dan negatif masing-masing punya tampilan yang **dapat dibedakan** —
      diuji ketiganya.

#### Dependency / Blocked by

Bergantung pada `TKT-U1-001`.

**Terbuka tetapi tidak menahan:** berapa desimal untuk Rupiah pada tampilan belum ditetapkan Work
Owner (`ADR-0016` pertanyaan terbuka). Nilai sementara dipakai dan **ditandai di kode**; mengubahnya
kelak menyentuh satu berkas.

#### Constraint keamanan, data, operasional

- **Angka pecahan di JavaScript tidak presisi.** Nilai uang tidak boleh melewati batas itu sebagai
  angka — diterima, disimpan di state, dan dikirim sebagai **string desimal**.
- Tampilan WIB memakai satu fungsi konversi; klien **tidak** menerapkan penyesuaian jam sendiri
  (`ADR-0017` butir 3).

#### Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** mengembalikan aturan pembulatan; karena
terpusat, perubahannya seragam di seluruh layar.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm run test -- Format
grep -rInE "toLocaleString|Intl\.NumberFormat" src/ | grep -v src/format/   # HARUS 0 baris
TZ=UTC npm run test -- Format.tanggal
TZ=Asia/Jakarta npm run test -- Format.tanggal
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Uang disimpan presisi penuh, dibulatkan hanya saat tampil | `D-51` · `ADR-0016` |
| Toleransi share 4 desimal `99.9999`–`100.0001` | `D-51` |
| 411 `TO_CHAR` mengembalikan tanggal sebagai string | `D-20` · `ADR-0005` §3.2 |
| Angka pecahan JavaScript tidak presisi | `ADR-0016` Negatif/utang teknis |
| Konversi WIB tunggal | `ADR-0017` butir 3 · `F-5` |

#### Comments

### TKT-U2-005 — Pemilihan pustaka tabel: TanStack Table versus AG Grid

| | |
|---|---|
| **Status** | ready-for-human |
| **Kesiapan** | siap |
| **Modul** | U-2 · Gelombang: 2 · Bergantung pada: TKT-U1-001 |
| **Requirement** | FR-U2 |
| **Keputusan** | D-23, D-09 |
| **ADR** | 0002 |
| **Risiko** | R-11 |
| **Rule Pega yang digantikan** | grid bawaan Pega pada **268 dari 269 section**; `Section/` seluruhnya |
| **Peran penguji gerbang 2** | **tidak berlaku** — modul fondasi UI (`D-60`) |
| **Label** | `modul::U-2` `tipe::keputusan-teknis` `status::ready-for-human` `prioritas::tinggi` `gelombang::2` |
| **Milestone** | Gelombang 2 — Kerangka UI |
| **Berkas sumber** | `ticketing/U-2-Komponen-Layar-Baku/issues/05-pemilihan-pustaka-tabel.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu keputusan tertulis tentang pustaka tabel mana yang dipakai, **beserta alasannya dan bukti
pengukurannya** — bukan preferensi.

Nilai bisnisnya: pilihan ini mengikat 268 layar. Menggantinya setelah 50 layar ditulis adalah
pekerjaan berbulan-bulan. `ADR-0002` sengaja meninggalkannya terbuka karena koreksi ukuran
(median 6 kolom, bukan 18–27) **melemahkan alasan memilih pustaka kelas berat** — dan keputusan
itu pantas diambil dengan angka, bukan dengan kesan.

#### Ruang lingkup

- Prototipe kecil untuk **kedua** pustaka, memuat kasus terberat yang benar-benar ada:
  grid **39 kolom**, grid **25 kolom**, dan grid dengan 10.000 baris hasil paginasi server-side.
- Pengukuran yang dibandingkan: waktu render awal, waktu ganti halaman, ukuran bundel yang
  ditambahkan, dan **jumlah baris kode** yang dibutuhkan untuk mencapai perilaku yang sama.
- Pemeriksaan **lisensi**: AG Grid punya edisi komersial; bila fitur yang dibutuhkan ada di edisi
  berbayar, biayanya diangkat sebagai keputusan Work Owner.
- Keluaran akhir: **pembaruan `ADR-0002`** dengan keputusan dan angkanya, bukan berkas ADR baru.

#### Non-goal

- **Tidak** membangun komponen tabel produksi — itu `TKT-U2-001`, dan menunggu tiket ini.
- **Tidak** membandingkan seluruh pustaka tabel yang ada. `ADR-0002` sudah mempersempit ke dua.

#### Acceptance criteria

- ☐ Kedua prototipe dapat dijalankan dari repository dan **menampilkan data yang sama**.
- ☐ Tabel pengukuran memuat keempat metrik untuk kedua pustaka, dengan **angka**, bukan kesan.
- ☐ Pengukuran dijalankan pada kasus **39 kolom**, **25 kolom**, dan **10.000 baris** —
      ketiganya diambil dari layar nyata, disebutkan nama harness-nya.
- ☐ Kebutuhan lisensi dinyatakan tegas: fitur apa yang menuntut edisi berbayar, atau **nihil**.
- ☐ `ADR-0002` diperbarui: pilihan final, alasannya, dan angka pendukungnya; bagian
      **Negatif / utang teknis** ikut diperbarui dengan konsekuensi pilihan itu.
- ☐ Keputusan mempertimbangkan `D-09` secara eksplisit — pustaka yang menuntut lebih sedikit
      konsep baru bagi tim eks-Pega diberi bobot, dan bobot itu ditulis, bukan disiratkan.

#### Dependency / Blocked by

Bergantung pada `TKT-U1-001` (kerangka SPA agar prototipe dapat dijalankan).

**Yang bergantung padanya:** `TKT-U2-001`, dan lewat itu seluruh layar `U-3`, `U-4`, `U-5`.

#### Constraint keamanan, data, operasional

- Prototipe **tidak boleh** memakai data produksi. Data contoh dibuat sendiri — meski staging
  memuat data nyata (`ADR-0029`), prototipe tidak membutuhkannya.
- Bila pilihan jatuh pada edisi berbayar, **keputusan biaya milik Work Owner**, bukan tim teknis.

#### Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** membuang prototipe; tidak ada dampak.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
cd prototipe/tanstack && npm ci && npm run build && npm run bench
cd prototipe/aggrid   && npm ci && npm run build && npm run bench
### bandingkan ukuran bundel
du -sh prototipe/*/dist
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 268 dari 269 section bergrid | `docs/Steering/06-MODULE-BREAKDOWN.md` §4 |
| Median 6 kolom; tiga fitur grid termahal tidak dipakai | `T-11` · `docs/Steering/01-FRONTEND-ANALYSIS.md` banner koreksi |
| Pilihan TanStack versus AG Grid masih terbuka | `ADR-0002` pertanyaan terbuka |
| Tim eks-Pega, utamakan sedikit konsep baru | `D-09` |
| Grid 39 kolom dan 25 kolom | `docs/verifikasi-bukti-adr.md` §15 baris `U-2` |

#### Comments

# 32. U-3 · Layar Inbox per Peran

*Modul Frontend · folder `docs/ticketing/U-3-Layar-Inbox-per-Peran/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Inbox** — `InboxRegister_Harness`, `InboxKomite_Harness`, `InboxSurvey_Harness`, `InboxPLADLA`, `InboxSalvage`, `InboxInvestigator_Harness`, `inboxCompliance_Harness`, `inboxAnalystDoctor_Harness`, `InboxTKA_Harness`, `UserTeknisInbox`, `PNCInboxAdmin`, dan seterusnya |
| **Kode modul** | `U-3` |
| **Gelombang** | 4 — Inbox dan penugasan |
| **Ukuran** | **Besar** |
| **Bergantung pada** | `U-2` Komponen Layar Baku · `B-6` Penugasan & Inbox Petugas |
| **Kesiapan** | **SEBAGIAN** |

### Apa yang dikerjakan modul ini

Layar tempat setiap petugas melihat **pekerjaan yang menjadi tanggung jawabnya** — daftar klaim
yang menunggu tindakannya, sesuai perannya.

Inilah layar pertama yang dibuka petugas setiap pagi. Bila ia salah, petugas tidak tahu apa yang
harus dikerjakan hari itu.

### Ukuran yang terverifikasi

**26 harness** memuat kata *Inbox* dalam namanya (dihitung langsung dari direktori `Harness/`;
total harness 74).

> `06-MODULE-BREAKDOWN.md:85` menyebut **"~20 inbox berbasis peran"**. Selisihnya bukan
> pertentangan: sebagian dari 26 itu tampaknya **bukan inbox peran** melainkan daftar pilihan —
> misalnya `ListDocumentTypeInbox`, `CauseOfLossInbox`, `StatusClaimInbox`.
> **Definisinya kini ada (`D-79`):** Inbox = layar berisi **daftar pekerjaan milik pengguna** —
> Tugas dari Worklist/Workbasket. Layar data acuan **bukan** Inbox, sekalipun namanya mengandung
> kata itu. Dengan definisi ini **tujuh** dari 26 diusulkan pindah ke `U-6` karena bersidik jari
> layar master. Yang tersisa: **8 usulan `DUGAAN` dan 2 `JANGGAL`** menunggu koreksi Work Owner.

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | ~~Mana yang benar-benar inbox peran?~~ — **definisinya tertutup `D-79`**; sisa: koreksi 8 `DUGAAN` + 2 `JANGGAL` | **Work Owner** |
| **Keputusan** | Satu layar inbox yang menyesuaikan diri per peran, atau 20-an layar terpisah? | **Work Owner + Lead Engineer** |
| **Bergantung** | Model penugasan `B-6` harus lebih dulu ada | — (`D-26`) |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-U3-001](issues/01-layar-inbox-baku.md) | Layar inbox baku dan penyaringnya | `needs-info` |
| [TKT-U3-002](issues/02-inbox-per-peran-dan-pemetaannya.md) | Inbox per peran dan pemetaan dari harness | `needs-info` |

> **Daftar lengkap 74 harness beserta kelasnya:** [`../INVENTARIS-HARNESS.md`](../INVENTARIS-HARNESS.md)
> — pembagian modulnya **belum diputuskan** (`D-73`).

## Daftar Tiket

### TKT-U3-001 — Layar inbox baku dan penyaringnya

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan rancangan** |
| **Modul** | **U-3 Layar Inbox per Peran** · Gelombang: 4 · Bergantung pada: TKT-U2-001, TKT-B06-002 |
| **Requirement** | FR-U3 |
| **Keputusan** | D-26 |
| **ADR** | 0012 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | pola bersama **26 harness Inbox** — antara lain `InboxRegister_Harness`, `InboxSurvey_Harness`, `UserInbox_Harness` |
| **Peran penguji gerbang 2** | **PncRegister**, **PncSurveyor**, **PncAdmin** |
| **Label** | `modul::U-3` `tipe::fondasi` `status::needs-info` `prioritas::tinggi` `gelombang::4` |
| **Milestone** | Gelombang 4 — Inbox dan penugasan |
| **Berkas sumber** | `ticketing/U-3-Layar-Inbox-per-Peran/issues/01-layar-inbox-baku.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu layar inbox yang benar, dipakai ulang oleh seluruh peran.

Nilai bisnisnya sama dengan alasan `U-2` menjadi investasi terpenting di frontend: bila tiap inbox
dibuat sendiri-sendiri, ada dua puluhan implementasi yang berbeda perilakunya — dan memperbaiki satu
bug menuntut dua puluh perbaikan. Membuat satu yang benar berarti perbaikan di satu tempat
memperbaiki semuanya.

#### Ruang lingkup

- Layar inbox baku di atas komponen tabel `TKT-U2-001`: **paginasi dari server**, penyortiran,
  penyaringan.
- Kolom dan penyaring **didefinisikan per peran sebagai konfigurasi**, bukan sebagai layar terpisah.
- Penandaan pekerjaan yang **sudah lewat tenggat**, karena itu yang dicari petugas lebih dulu.
- Tindakan dari inbox: membuka klaim, dan mengambil pekerjaan dari workbasket (`B-6`).

#### Non-goal

- **Tidak** memetakan inbox mana untuk peran mana — itu `TKT-U3-002`.
- **Tidak** membangun model penugasannya; itu `B-6` (`D-26`).

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Satu layar yang menyesuaikan diri per peran, atau layar terpisah per peran?** | **Work Owner + Lead Engineer** | Menentukan bentuk seluruh modul. Satu layar berarti perbedaan antar peran menjadi data; layar terpisah berarti dua puluhan halaman yang harus dirawat sendiri-sendiri |
| **Kolom apa yang wajib ada di setiap inbox?** | **Work Owner** | Petugas memilih pekerjaan berdasarkan kolom yang terlihat. Salah kolom berarti mereka membuka klaim satu per satu untuk mencari |
| **Urutan baku: paling lama menunggu, atau paling dekat tenggat?** | **Work Owner** | Menentukan pekerjaan mana yang dikerjakan lebih dulu setiap hari |

#### Acceptance criteria

- ☐ Layar inbox memakai komponen tabel baku `TKT-U2-001` — **bukan** tabel yang dibuat khusus.
- ☐ Paginasi dilakukan **di server**; layar **tidak memuat seluruh baris** lalu memotongnya —
      diuji dengan inbox berisi 5.000 baris.
- ☐ Kolom dan penyaring per peran berasal dari **konfigurasi**; menambah peran **tidak menuntut
      halaman baru** — diuji.
- ☐ Pekerjaan yang **lewat tenggat tertandai** dan dapat diurutkan lebih dulu — diuji.
- ☐ Petugas **hanya melihat pekerjaan yang menjadi haknya** — diuji dengan dua peran berbeda;
      penyaringan ditegakkan **di server** (`ADR-0023`), bukan di layar.
- ☐ Layar tetap terpakai pada **lebar layar yang dipakai petugas cabang** — diuji.
- ☐ Gerbang 2: UAT **PncRegister** dan **PncSurveyor** secara terpisah.

#### Dependency / Blocked by

`TKT-U2-001` · `TKT-B06-002` · `TKT-F3-005`. **Terhalang tiga keputusan.**

#### Constraint keamanan, data, operasional

- **Penyaringan inbox adalah kendali akses, bukan kenyamanan.** Bila dilakukan di layar saja,
  data pekerjaan peran lain tetap terkirim ke browser.
- Inbox adalah layar yang paling sering dibuka — beban kuerinya paling besar di seluruh aplikasi.
- Sebagian inbox memuat klaim dengan **data medis**; `FR-R2` berlaku pada kolom yang ditampilkan.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema; menambah tabel konfigurasi kolom per peran. Backward-compatible.

**Rollback:** mengembalikan versi layar. Petugas kembali ke inbox Pega selama peralihan (`D-05`).

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm test -- inbox --run TestPaginasiDariServer
npm test -- inbox --run TestKolomDariKonfigurasi
go test ./internal/adapter/http/... -run TestPenyaringanInboxDiServer
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 26 harness bernama Inbox dari 74 harness | direktori `Harness/` (dihitung langsung) |
| Model penugasan: worklist dan workbasket | `D-26` · `docs/Steering/CONTEXT.md` |
| Tabel baku median 6 kolom, paginasi server-side | `docs/Steering/06-MODULE-BREAKDOWN.md:76` |
| 3.189 grid terikat page list klipboard | `docs/Steering/06-MODULE-BREAKDOWN.md` koreksi ukuran |
| Otorisasi diperiksa di server | `D-59` · `ADR-0023` |

#### Comments

### TKT-U3-002 — Inbox per peran dan pemetaan dari harness

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan lingkup — jumlah layar belum pasti** |
| **Modul** | **U-3 Layar Inbox per Peran** · Gelombang: 4 · Bergantung pada: TKT-U3-001 |
| **Requirement** | FR-U3 |
| **Keputusan** | D-26, D-58, D-79 |
| **ADR** | 0012 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | **26 harness Inbox** di `Harness/` — `InboxRegister_Harness`, `InboxKomite_Harness`, `InboxSurvey_Harness`, `InboxPLADLA`, `InboxPLA_harness`, `InboxSalvage`, `InboxInvestigator_Harness`, `inboxCompliance_Harness`, `inboxAnalystDoctor_Harness`, `InboxTKA_Harness`, `InboxRCVApp_Harness`, `InboxClaimTreaty_Harness`, `InboxClaimNonProp_Harness`, `Inbox_XOL_Harness`, `InboxManagerAdmin_Harness`, `InboxKomunikasiCabang`, `InboxAutoClaim`, `PNCInboxAdmin`, `UserInbox_Harness`, `UserTeknisInbox`, `SurveyorsInbox`, `DetailSurveyorsInbox`, `StatusClaimInbox`, `CauseOfLossInbox`, `CauseOfLossInboxSimasOnline`, `ListDocumentTypeInbox` |
| **Peran penguji gerbang 2** | **satu peran per inbox** — bukan satu orang untuk semuanya |
| **Label** | `modul::U-3` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::4` |
| **Milestone** | Gelombang 4 — Inbox dan penugasan |
| **Berkas sumber** | `ticketing/U-3-Layar-Inbox-per-Peran/issues/02-inbox-per-peran-dan-pemetaannya.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Setiap peran punya inbox yang menampilkan pekerjaannya, dan **tidak ada peran yang kehilangan
inbox-nya** saat Pega dimatikan.

Nilai bisnisnya terletak pada kata "tidak ada yang kehilangan". Ada 22 peran bisnis (`D-58`) dan
26 harness bernama Inbox. Bila pemetaannya tidak dibuat eksplisit, peran yang terlewat baru akan
ketahuan **setelah** Pega mati — saat petugasnya tidak punya layar untuk bekerja.

#### Ruang lingkup

- **Pemetaan eksplisit**: harness Inbox mana melayani peran mana, satu per satu.
- Pemisahan mana yang **inbox peran** dan mana yang **daftar pilihan** — `ListDocumentTypeInbox`,
  `CauseOfLossInbox`, `StatusClaimInbox` tampaknya termasuk yang kedua.
- Konfigurasi kolom dan penyaring per inbox di atas layar baku `TKT-U3-001`.

#### Non-goal

- **Tidak** menambah inbox untuk peran yang sekarang tidak punya.
- **Tidak** menggabungkan dua inbox menjadi satu tanpa persetujuan Work Owner.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| ~~Mana yang benar-benar inbox peran?~~ — **definisi tertutup `D-79`**: Inbox = daftar **pekerjaan** milik pengguna; layar data acuan **bukan** Inbox. Tujuh dari 26 diusulkan pindah ke `U-6` | **Work Owner** | Sisa yang menahan: **8 usulan `DUGAAN`** dan **2 `JANGGAL`** pada inventaris harness |
| **Pemetaan 22 peran bisnis ke inbox-nya** | **Work Owner** (`D-58`) | Tanpa ini tidak dapat dipastikan setiap peran punya tempat bekerja |
| **Ada peran yang berbagi satu inbox, atau punya lebih dari satu?** | **Work Owner** | Menentukan apakah pemetaannya satu-ke-satu |

#### Acceptance criteria

- ☐ **Setiap dari 22 peran bisnis punya inbox**, dan pemetaannya tercatat sebagai tabel — tidak
      ada peran tanpa inbox, diperiksa satu per satu.
- ☐ Jumlah inbox yang dibangun **dilaporkan sebagai angka** dan cocok dengan daftar yang
      disetujui Work Owner.
- ☐ Harness yang dinyatakan **bukan inbox peran** dipindahkan ke modul yang tepat, **tidak
      hilang begitu saja** — dicatat per berkas.
- ☐ Tiap inbox menampilkan **pekerjaan yang sama dengan inbox Pega** untuk peran yang sama —
      dibandingkan lewat `S-8` pada 10 kasus per inbox.
- ☐ Peran **tidak dapat membuka inbox peran lain** lewat URL langsung — diuji: `403`.
- ☐ Gerbang 2: UAT **oleh pemegang masing-masing peran**, bukan satu orang untuk semuanya.

#### Dependency / Blocked by

`TKT-U3-001` · `TKT-F3-002` (22 peran) · `TKT-B06-002`. **Terhalang Work Owner.**

#### Constraint keamanan, data, operasional

- **Peran yang terlewat baru terlihat setelah Pega mati.** Itulah mengapa pemetaannya harus
  eksplisit sebelum cutover, bukan sesudahnya.
- `inboxAnalystDoctor_Harness` dan inbox RCL Dokter memuat **data medis** — `FR-R2` berlaku, dan
  kedua inbox itu tidak boleh terbuka bagi peran lain.
- `InboxKomite_Harness` menampilkan klaim yang menunggu persetujuan — isinya menentukan apa yang
  dilihat komite sebelum memutuskan (`B-7`).

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema; menambah baris konfigurasi per inbox.

**Rollout:** per peran, bukan sekaligus. Peran yang inbox-nya belum dialihkan **tetap memakai Pega**
(`D-05`).

**Rollback:** peran dikembalikan ke inbox Pega satu per satu.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/cek-inbox-per-peran --harap-semua-22
go test ./internal/adapter/http/... -run TestInboxPeranLainDitolak
go run ./cmd/s8 banding --modul U-3 --per-inbox 10
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 26 harness bernama Inbox | direktori `Harness/` (dihitung langsung) |
| Dokumen menyebut "~20 inbox berbasis peran" | `docs/Steering/06-MODULE-BREAKDOWN.md:85` |
| 22 peran bisnis, satu-untuk-satu dengan access group | `D-58` |
| Model penugasan worklist/workbasket | `D-26` |
| Akses data medis dibatasi dua peran | `FR-R2` |

#### Comments

# 33. U-4 · Layar Transaksi Klaim

*Modul Frontend · folder `docs/ticketing/U-4-Layar-Transaksi-Klaim/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Layar transaksi** — `ViewPolis`, `ReceiveDoucument_Harness`, `InputProgress`, `InputProtection_Harness`, `RCL_Harness`, `RCLPUCL_Harness`, `ProgressClaim_Harness`, `View_DetailKlaimCabang_Harness`, `ViewDetailHasilSurveyorInternal1`, `PNCSearchKlaim`, `AutoKlaim` |
| **Kode modul** | `U-4` |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | **Besar** |
| **Bergantung pada** | `U-2` Komponen Layar Baku · seluruh modul `B-*` |
| **Kesiapan** | **SEBAGIAN** |

### Apa yang dikerjakan modul ini

Layar tempat petugas benar-benar **mengerjakan klaim** — mengikuti tahapan yang sama dengan
`Flow/Register_Flow.xml`:

**Input Register** → **View Polis** → **Input Estimasi** → **Choose Surveyor** →
**Send To Analis / Send To PIC Teknik** → **RCL/PUCL** → **Compliance** → **Investigator** →
**Analyst Doctor / RCL Dokter**

Setiap tahap punya layarnya, dan `U-4` adalah kumpulan layar itu.

### Ukuran yang terverifikasi

| | |
|---|---|
| **Harness** | 74 seluruhnya; **26** bernama Inbox (milik `U-3`), sisanya terbagi antara `U-4`, `U-5`, dan `U-6` |
| **Section** | **269**, **268 di antaranya bergrid** |
| **Flow Action** | **29** |

> **BELUM DIPUTUSKAN — pertanyaan terbuka:** pembagian 48 harness non-Inbox ke `U-4`, `U-5`, dan
> `U-6` belum ditetapkan per berkas. Tanpa itu, jumlah layar yang dijanjikan tiap modul frontend
> adalah perkiraan, bukan komitmen. Pemilik: **Work Owner + Lead Engineer**.

### Hubungannya dengan modul bisnis

`U-4` **tidak memuat aturan bisnis**. Setiap layarnya adalah permukaan dari modul `B-*`
yang bersangkutan — `B-1` View Polis, `B-2` Input Register, `B-5` Input Estimasi, `B-8` Choose
Surveyor, `B-11` RCL/PUCL & Compliance, `B-14` Input Receive Document. Bila aturannya berada di
layar, ia dapat dipintas lewat pemanggilan langsung (`ADR-0023`).

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | Pembagian 48 harness non-Inbox ke `U-4`/`U-5`/`U-6` per berkas | **Work Owner + Lead Engineer** |
| **Bergantung** | Seluruh modul `B-*` yang menjadi isinya — sebagian masih TERHALANG (`B-5`, `B-7`, `B-11`) | — |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-U4-001](issues/01-kerangka-layar-transaksi-per-tahap.md) | Kerangka layar transaksi per tahap klaim | `needs-info` |
| [TKT-U4-002](issues/02-layar-pencarian-dan-detail-klaim.md) | Layar pencarian dan detail klaim | `needs-info` |

> **Daftar lengkap 74 harness beserta kelasnya:** [`../INVENTARIS-HARNESS.md`](../INVENTARIS-HARNESS.md)
> — pembagian modulnya **belum diputuskan** (`D-73`).

## Daftar Tiket

### TKT-U4-001 — Kerangka layar transaksi per tahap klaim

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan pembagian layar** |
| **Modul** | **U-4 Layar Transaksi Klaim** · Gelombang: 3 · Bergantung pada: TKT-U2-001, TKT-U2-002, TKT-U1-004 |
| **Requirement** | FR-U4 |
| **Keputusan** | D-05 |
| **ADR** | 0012, 0023 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | `Flow/Register_Flow.xml` (tahapan) · 29 Flow Action · harness `InputProgress`, `ProgressClaim_Harness`, `StatusProgress`, `StatusProgress2` |
| **Peran penguji gerbang 2** | **PncRegister**, **PncSurveyor**, **PncAnalis** |
| **Label** | `modul::U-4` `tipe::fondasi` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/U-4-Layar-Transaksi-Klaim/issues/01-kerangka-layar-transaksi-per-tahap.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Kerangka layar yang mengikuti tahapan klaim, sehingga petugas melihat **di mana klaim berada** dan
**apa tindakan yang tersedia baginya di tahap itu**.

Nilai bisnisnya: tahapan klaim di Pega terbentuk dari Flow dan Flow Action, dan petugas terbiasa
dengan urutannya. Bila layar baru tidak menampilkan tahapan yang sama, petugas kehilangan orientasi
— bukan karena datanya salah, melainkan karena mereka tidak tahu lagi sedang di mana.

#### Ruang lingkup

- Kerangka layar per tahap `Register_Flow`: Input Register, View Polis, Input Estimasi,
  Choose Surveyor, Send To Analis / PIC Teknik, RCL/PUCL, Compliance, Investigator,
  Analyst Doctor / RCL Dokter.
- Penanda tahap berjalan dan riwayat tahap yang sudah dilalui.
- **Tindakan yang tersedia ditentukan server**, bukan disembunyikan di layar (`ADR-0023`).
- Penyimpanan draft agar petugas tidak kehilangan isian saat berpindah tahap.

#### Non-goal

- **Tidak** memuat aturan bisnis apa pun. Seluruhnya milik modul `B-*` yang bersangkutan.
- **Tidak** membangun layar pencarian dan detail — itu `TKT-U4-002`.
- **Tidak** mengubah urutan tahapan.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Pembagian 48 harness non-Inbox ke `U-4`, `U-5`, dan `U-6` per berkas** | **Work Owner + Lead Engineer** | Jumlah layar yang dijanjikan modul ini belum dapat dinyatakan sebagai angka |
| **Tahapan ditampilkan sebagai jalur bertahap, atau sebagai satu halaman panjang?** | **Work Owner** | Mengubah cara petugas bekerja sehari-hari, dan `P-5` menuntut kesetaraan perilaku lebih dulu |
| **Lompatan lateral (Ticket rule) muncul sebagai apa di layar?** | **Work Owner** | Di Pega klaim dapat melompat antar tahap lewat Ticket rule (`FR-W2`). Bila layar tidak menunjukkannya, petugas akan melihat klaim berpindah tanpa sebab yang terlihat |

#### Acceptance criteria

- ☐ Tahapan yang ditampilkan **sama dengan tahapan `Register_Flow`** — diperiksa satu per satu.
- ☐ **Tindakan yang tersedia ditentukan server**; menyembunyikan tombol di layar **tidak menjadi
      satu-satunya penghalang** — diuji dengan memanggil endpoint langsung: `403`.
- ☐ Klaim yang berpindah lewat **lompatan lateral** tetap terbaca posisinya di layar — diuji.
- ☐ Isian yang belum disimpan **tidak hilang** saat petugas berpindah tahap atau sesi terputus —
      diuji.
- ☐ Seluruh tabel di layar ini memakai komponen `TKT-U2-001`, dan seluruh form memakai
      `TKT-U2-002` — diperiksa; tidak ada tabel atau form buatan sendiri.
- ☐ Angka uang dan tanggal dipformat lewat `TKT-U2-004` — **sama dengan yang tampil di laporan**.
- ☐ Gerbang 2: UAT **PncRegister**, **PncSurveyor**, dan **PncAnalis** — masing-masing pada
      tahapnya sendiri.

#### Dependency / Blocked by

`TKT-U2-001` · `TKT-U2-002` · `TKT-U1-004` (peta rute dari 74 harness) · modul `B-*` yang menjadi
isinya. **Terhalang keputusan pembagian layar.**

#### Constraint keamanan, data, operasional

- **Layar bukan tempat menegakkan kewenangan.** `D-59` menetapkan kontrol berbasis menu, dan
  `ADR-0023` menuntut pemeriksaan di setiap endpoint. Tombol yang disembunyikan **bukan kendali**.
- Layar tahap Analyst Doctor dan RCL Dokter memuat **data medis** (`FR-R2`).
- Selama peralihan, klaim yang sama dapat dibuka di Pega **dan** di layar baru (`D-05`) — aturan
  penulis tunggal (`P-1`) menentukan sisi mana yang boleh menyimpan.

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema; menambah tabel draft isian bila disepakati. Backward-compatible.

**Rollback:** petugas dikembalikan ke layar Pega per tahap. Draft yang tersimpan di sistem baru
**tidak terbawa** ke Pega.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm test -- transaksi --run TestTahapanSesuaiRegisterFlow
go test ./internal/adapter/http/... -run TestTindakanDitentukanServer
npm test -- transaksi --run TestDraftTidakHilang
go run ./cmd/s8 banding --modul U-4 --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Tahapan klaim | `Flow/Register_Flow.xml` |
| 29 Flow Action | direktori `Flow Action/` (dihitung langsung) |
| 269 section, 268 bergrid | `docs/Steering/06-MODULE-BREAKDOWN.md:88` |
| Lompatan lateral lewat Ticket rule | `FR-W2` · `docs/Steering/CONTEXT.md` |
| Otorisasi diperiksa di server, kontrol berbasis menu | `D-59` · `ADR-0023` |

#### Comments

### TKT-U4-002 — Layar pencarian dan detail klaim

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan batas data** |
| **Modul** | **U-4 Layar Transaksi Klaim** · Gelombang: 3 · Bergantung pada: TKT-U4-001, TKT-F3-005 |
| **Requirement** | FR-U4 |
| **Keputusan** | D-59, D-71 |
| **ADR** | 0012, 0023 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | harness `PNCSearchKlaim`, `View_DetailKlaimCabang_Harness`, `ViewTempDetailClaim`, `ViewTempDetailAllCase`, `ProgressClaim_Harness`, `StatusProgress`, `StatusProgress2` |
| **Peran penguji gerbang 2** | **PncRegister**, **PncKacab**, **PncManagerAdmin** |
| **Label** | `modul::U-4` `tipe::migrasi` `status::needs-info` `prioritas::tinggi` `gelombang::3` |
| **Milestone** | Gelombang 3 — Jalur klaim inti |
| **Berkas sumber** | `ticketing/U-4-Layar-Transaksi-Klaim/issues/02-layar-pencarian-dan-detail-klaim.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Layar untuk **menemukan klaim** dan **melihat seluruh keadaannya dalam satu tempat**.

Nilai bisnisnya: pencarian adalah pintu masuk ke hampir semua pekerjaan. Petugas datang dengan
nomor klaim atau nomor polis dari telepon nasabah, dan bila pencariannya lambat atau tidak
menemukan, seluruh percakapan itu tertahan.

#### Ruang lingkup

- Pencarian klaim berdasarkan **nomor klaim** (format `PNCN.YY.xxxx`, `D-71`), nomor polis,
  nama tertanggung, tanggal, cabang, dan status.
- Layar detail klaim: ringkasan polis, objek pertanggungan, estimasi, riwayat tahap, dokumen.
- **Batas data ditegakkan di server**: petugas cabang menemukan klaim cabangnya (`TKT-F3-005`).
- Paginasi hasil pencarian **dari server**.

#### Non-goal

- **Tidak** membangun layar per tahap — itu `TKT-U4-001`.
- **Tidak** menambah kriteria pencarian yang tidak ada di sistem lama.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Siapa boleh mencari klaim lintas cabang?** | **Work Owner** (`D-59`) | Kontrol sekarang berbasis menu, bukan berbutir aksi — sehingga batas data pencarian **belum tentu sama dengan batas menu**. Salah menetapkannya membuka seluruh portofolio klaim kepada peran cabang |
| **Pencarian tanpa kriteria mengembalikan apa?** | **Work Owner** | Di Pega, laporan memotong di 500 baris tanpa memberi tahu. Bila pencarian mengembalikan semuanya, ia menjadi cara mengunduh seluruh basis klaim lewat layar |
| **Nomor klaim lama berformat berbeda — tetap dapat dicari?** | **Work Owner** (`D-71`) | `D-71` menetapkan format `PNCN.YY.xxxx` untuk yang baru; klaim lama tetap ada di data |

#### Acceptance criteria

- ☐ Pencarian menemukan klaim berdasarkan seluruh kriteria yang ada di `PNCSearchKlaim` —
      diperiksa per kriteria.
- ☐ **Klaim berformat nomor lama tetap ditemukan** — diuji dengan kedua format (`D-71`).
- ☐ **Batas data ditegakkan di server** — diuji: peran cabang A memanggil endpoint pencarian
      untuk klaim cabang B dan **tidak menerima datanya**, bukan sekadar tidak melihatnya.
- ☐ Paginasi hasil dilakukan **di server**; hasil besar **tidak dipotong diam-diam** — bila
      melebihi batas, pengguna **diberi tahu**.
- ☐ Layar detail menampilkan data yang **sama dengan Pega** untuk klaim yang sama — dibandingkan
      lewat `S-8` pada 20 klaim, termasuk klaim dengan banyak objek pertanggungan.
- ☐ Angka uang dan tanggal memakai `TKT-U2-004` — sama dengan tampilan di laporan.
- ☐ Klaim dengan **data medis** hanya menampilkan bagian itu kepada peran yang berwenang
      (`FR-R2`) — diuji dengan peran lain.
- ☐ Gerbang 2: UAT **PncRegister** dan **PncKacab** secara terpisah — pemisahan batas datanya
      yang diuji.

#### Dependency / Blocked by

`TKT-U4-001` · `TKT-F3-005` · `TKT-B01-001` (snapshot polis). **Terhalang keputusan batas data.**

#### Constraint keamanan, data, operasional

- **Pencarian adalah cara termudah memintas batas data** bila penyaringnya hanya di layar.
- Pencarian tanpa batas atas dapat menjadi **jalan mengunduh seluruh basis klaim** — masalah yang
  sama dengan `TKT-S2-002`, hanya lewat pintu berbeda.
- Pencarian berjalan di atas tabel klaim yang sama yang dipakai transaksi; kueri yang buruk di sini
  memperlambat seluruh pengguna (`ADR-0001`).

#### Migrasi skema / rollout / rollback

Tidak menyentuh kolom; kemungkinan menambah index pencarian (`TKT-F2-004`) — penambahan index
bersifat backward-compatible (`P-4`).

**Rollback:** petugas kembali memakai pencarian Pega (`D-05`).

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/http/... -run TestBatasDataPencarianDiServer
go test ./internal/app/klaim/... -run TestNomorKlaimFormatLamaDanBaru
npm test -- pencarian --run TestPaginasiDariServer
go run ./cmd/s8 banding --modul U-4 --kasus 20
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Harness pencarian dan detail klaim | direktori `Harness/` |
| Format nomor klaim `PNCN.YY.xxxx` | `D-71` |
| Kontrol berbasis menu, bukan berbutir aksi | `D-59` |
| Otorisasi diperiksa di setiap endpoint | `ADR-0023` |
| Laporan lama memotong di 500 baris tanpa pemberitahuan | `T-12` · `ADR-0011` |

#### Comments

# 34. U-5 · Layar Laporan

*Modul Frontend · folder `docs/ticketing/U-5-Layar-Laporan/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Layar laporan** — harness `Har_LaporanHasilAI`, `ReportKPIHarness`, `PNCTATReport`, `OutstandingKlaimperCabang_Harness`, `MonitoringSLINKOJK`, `PNCStudyClaim`, `DashboardClaim_Harness` |
| **Kode modul** | `U-5` |
| **Gelombang** | 6 — Laporan |
| **Ukuran** | **Besar** |
| **Bergantung pada** | `U-2` Komponen Layar Baku · `S-2` Laporan & Export · `S-7` Dashboard |
| **Kesiapan** | **SEBAGIAN** |

### Apa yang dikerjakan modul ini

Layar tempat pengguna **memilih laporan, mengisi parameternya, menjalankannya, dan mengunduh
hasilnya**.

`S-2` membangun laporannya; `U-5` adalah cara orang sampai ke laporan itu.

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Bergantung** | Seluruh penghalang `S-2` berlaku di sini — batas 500 baris, kriteria kesamaan keluaran, 12 Report Definition hilang | **Work Owner + Tim Pega** (`R-16`) |
| **Keputusan** | Export besar dijalankan serentak atau diantrekan — menentukan apakah layar butuh daftar unduhan | **Work Owner** |
| **Keputusan** | Pembagian 48 harness non-Inbox ke `U-4`/`U-5`/`U-6` per berkas | **Work Owner + Lead Engineer** |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-U5-001](issues/01-layar-pemilih-laporan-dan-parameter.md) | Layar pemilih laporan dan parameternya | `needs-info` |
| [TKT-U5-002](issues/02-unduhan-dan-antrean-export.md) | Unduhan hasil dan antrean export besar | `needs-info` |

> **Daftar lengkap 74 harness beserta kelasnya:** [`../INVENTARIS-HARNESS.md`](../INVENTARIS-HARNESS.md)
> — pembagian modulnya **belum diputuskan** (`D-73`).

## Daftar Tiket

### TKT-U5-001 — Layar pemilih laporan dan parameternya

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang penghalang `S-2`** |
| **Modul** | **U-5 Layar Laporan** · Gelombang: 6 · Bergantung pada: TKT-U2-002, TKT-S2-002 |
| **Requirement** | FR-U5 |
| **Keputusan** | D-11, D-59 |
| **ADR** | 0011, 0023 |
| **Risiko** | R-16 |
| **Rule Pega yang digantikan** | harness `Har_LaporanHasilAI`, `ReportKPIHarness`, `PNCTATReport`, `OutstandingKlaimperCabang_Harness`, `MonitoringSLINKOJK`, `PNCStudyClaim` |
| **Peran penguji gerbang 2** | **PNCReportClaimInternal**, **PNCReportClaimEksternal** |
| **Label** | `modul::U-5` `tipe::migrasi` `status::needs-info` `prioritas::sedang` `gelombang::6` |
| **Milestone** | Gelombang 6 — Laporan |
| **Berkas sumber** | `ticketing/U-5-Layar-Laporan/issues/01-layar-pemilih-laporan-dan-parameter.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Layar tempat pengguna menemukan laporan yang dibutuhkannya, mengisi parameternya, dan
menjalankannya.

Nilai bisnisnya: ada **56 laporan**. Bila daftarnya tidak tersusun dan parameternya tidak jelas,
pengguna akan menjalankan laporan yang salah atau menyerah dan meminta data lewat jalur lain.

#### Ruang lingkup

- Daftar laporan **yang dapat diakses peran yang sedang masuk** — laporan yang tidak berwenang
  **tidak muncul dan tidak dapat dijalankan**.
- Form parameter per laporan, memakai komponen `TKT-U2-002` dan pemilih tanggal baku.
- Menjalankan laporan dan menampilkan hasilnya dengan tabel baku `TKT-U2-001`.
- **Pemberitahuan bila hasil melebihi batas** — bukan pemotongan diam-diam (`TKT-S2-002`).

#### Non-goal

- **Tidak** membangun laporannya — itu `S-2`.
- **Tidak** membangun antrean unduhan — itu `TKT-U5-002`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **12 Report Definition hilang dari export** | **Tim Pega** (`R-16`) | Parameter laporan-laporan itu tidak diketahui, sehingga formnya tidak dapat dibuat |
| **Laporan mana untuk peran mana?** | **Work Owner** (`D-59`) | Kontrol sekarang berbasis menu; pemetaan laporan ke peran **belum tertulis** |
| **Berapa batas baru dan bagaimana pengguna diberi tahu?** | **Work Owner** | Layar inilah yang menyampaikannya; tanpa keputusan, ia tidak punya pesan untuk ditampilkan |

#### Acceptance criteria

- ☐ Daftar laporan yang muncul **sesuai kewenangan peran** — diuji dengan dua peran berbeda.
- ☐ Laporan yang tidak berwenang **tidak dapat dijalankan lewat pemanggilan langsung** — diuji:
      `403` (`ADR-0023`).
- ☐ Parameter tiap laporan **sama dengan parameter di Pega** — diperiksa per laporan.
- ☐ Hasil yang melebihi batas **memunculkan pemberitahuan**, bukan tabel yang diam-diam terpotong
      — diuji dengan data melebihi batas.
- ☐ Form parameter memakai `TKT-U2-002`; tanggal memakai pemilih tanggal baku — diperiksa.
- ☐ Laporan berisi **data medis** hanya muncul bagi peran yang berwenang (`FR-R2`) — diuji.
- ☐ Gerbang 2: UAT **PNCReportClaimInternal** dan **PNCReportClaimEksternal** secara terpisah.

#### Dependency / Blocked by

`TKT-U2-002` · `TKT-S2-002` · `TKT-F3-005`. **Terhalang Tim Pega (`R-16`) dan Work Owner.**

#### Constraint keamanan, data, operasional

- **Daftar laporan yang disaring hanya di layar bukan kendali.** Endpoint penjalan laporan wajib
  memeriksa kewenangan sendiri.
- Laporan adalah jalan memintas batas data layar bila kueri di baliknya tidak ikut dibatasi
  (`TKT-S2-002`).

#### Migrasi skema / rollout / rollback

Tidak menyentuh skema; menambah tabel pemetaan laporan ke peran.

**Rollback:** pengguna kembali ke layar laporan Pega (`D-05`).

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/http/... -run TestLaporanTanpaKewenanganDitolak
npm test -- laporan --run TestPemberitahuanSaatMelebihiBatas
npm test -- laporan --run TestFormParameterMemakaiKomponenBaku
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 56 laporan | `docs/Steering/06-MODULE-BREAKDOWN.md` `S-2` |
| 12 Report Definition hilang dari export | `docs/verifikasi-bukti-adr.md:2782` · `R-16` |
| `pyMaxRecords=500` pada 54 dari 56 laporan | `T-12` · `ADR-0011` |
| Kontrol berbasis menu | `D-59` |
| Otorisasi diperiksa di setiap endpoint | `ADR-0023` |

#### Comments

### TKT-U5-002 — Unduhan hasil dan antrean export besar

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan cara menjalankan export besar** |
| **Modul** | **U-5 Layar Laporan** · Gelombang: 6 · Bergantung pada: TKT-U5-001, TKT-S2-001 |
| **Requirement** | FR-U5 |
| **Keputusan** | D-11 |
| **ADR** | 0011 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | mekanisme unduhan laporan Pega · `Activity/PrintPDFAcceptanceNote-Act.xml` |
| **Peran penguji gerbang 2** | **PNCReportClaimInternal**, **PncManagerAdmin** |
| **Label** | `modul::U-5` `tipe::migrasi` `status::needs-info` `prioritas::sedang` `gelombang::6` |
| **Milestone** | Gelombang 6 — Laporan |
| **Berkas sumber** | `ticketing/U-5-Layar-Laporan/issues/02-unduhan-dan-antrean-export.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Pengguna mendapatkan berkas hasil laporan — PDF, Excel, atau CSV — **tanpa layarnya membeku** saat
laporan besar dijalankan.

Nilai bisnisnya: laporan bulanan adalah laporan terbesar dan dijalankan bersamaan oleh banyak
cabang di awal bulan. Bila tiap export menahan satu koneksi dan satu layar sampai selesai, saat
itulah sistem paling lambat justru ketika paling banyak dipakai.

#### Ruang lingkup

- Pengunduhan berkas hasil dari engine `TKT-S2-001`.
- Penanganan export besar sesuai keputusan Work Owner — **serentak** atau **diantrekan lalu
  diberitahukan**.
- Bila diantrekan: daftar permintaan export beserta statusnya, dan tautan unduhan saat selesai.
- Masa berlaku tautan unduhan, sejalan dengan `TKT-S1-002`.

#### Non-goal

- **Tidak** membangun engine pembuat berkas — itu `TKT-S2-001`.
- **Tidak** mengubah isi laporan.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Export besar dijalankan serentak, atau diantrekan lalu diberitahukan saat selesai?** | **Work Owner + Lead Engineer** (`ADR-0011`) | Ini menentukan **ada atau tidaknya seluruh separuh tiket ini**. Bila diantrekan, dibutuhkan daftar permintaan, status, dan pemberitahuan — bila tidak, semuanya gugur |
| **Berapa baris maksimum yang wajib dilayani satu export?** | **Work Owner** | Menentukan kapan layar harus menahan permintaan |
| **Berapa lama tautan unduhan berlaku?** | **Work Owner + Keamanan Informasi** | Berkas hasil memuat data nasabah; tautan yang berlaku selamanya memperpanjang paparan (`TKT-S1-002`) |

#### Acceptance criteria

- ☐ Berkas hasil dapat diunduh dalam ketiga format — diuji.
- ☐ Export besar **tidak membekukan layar** — diuji dengan export 100.000 baris; pengguna tetap
      dapat berpindah halaman.
- ☐ Export besar **tidak menghabiskan koneksi transaksi** — diuji dengan menjalankan export
      besar sambil mengirim permintaan transaksi biasa (`TKT-F2-001`).
- ☐ Tautan unduhan **kedaluwarsa** setelah masa berlaku yang ditetapkan — diuji.
- ☐ Pengguna **tidak dapat mengunduh hasil export milik pengguna lain** — diuji dengan tautan
      milik orang lain: ditolak.
- ☐ Bila antrean dipilih: pengguna melihat **status permintaannya** dan diberi tahu saat selesai
      — diuji. *Kriteria ini gugur bila Work Owner memilih export serentak.*
- ☐ Gerbang 2: UAT **PNCReportClaimInternal** pada laporan bulanan yang sebenarnya.

#### Dependency / Blocked by

`TKT-U5-001` · `TKT-S2-001` · `TKT-F2-001` · `TKT-S1-002`. **Terhalang keputusan Work Owner.**

#### Constraint keamanan, data, operasional

- **Berkas hasil export memuat data nasabah dalam jumlah besar dalam satu berkas.** Ia lebih
  berisiko daripada layar, karena dapat dipindahkan keluar sistem.
- Tautan unduhan tanpa masa berlaku mengulangi cacat token dokumen (`TKT-S1-002`).
- Export bersamaan di awal bulan adalah **beban puncak yang dapat diperkirakan** — dan kapasitas
  dirancang tanpa data historis yang sahih, karena angka pemakaian lama selalu terpotong di 500.

#### Migrasi skema / rollout / rollback

Menambah tabel permintaan export bila antrean dipilih. Tidak menyentuh skema klaim.

**Rollback:** pengguna kembali ke unduhan Pega. Berkas yang sudah diunduh **tetap ada di komputer
pengguna** — tidak dapat ditarik.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm test -- unduhan --run TestLayarTidakMembekuSaatExportBesar
go test ./internal/app/laporan/... -run TestPoolLaporanTerpisah
go test ./internal/adapter/http/... -run TestUnduhanMilikOrangLainDitolak
go test ./internal/app/laporan/... -run TestTautanUnduhanKedaluwarsa
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Export besar asinkron dan streaming | `docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md` butir 6 |
| Pool koneksi terpisah untuk laporan | idem butir 7 |
| Cara menjalankan export besar belum diputuskan | `ADR-0011` Pertanyaan terbuka |
| Token dokumen lama tanpa masa berlaku | `docs/verifikasi-bukti-adr.md` §14.10 |
| Angka pemakaian historis terpotong di 500 | `T-12` · `ADR-0011` |

#### Comments

# 35. U-6 · Layar Master Data

*Modul Frontend · folder `docs/ticketing/U-6-Layar-Master-Data/`*

## Spesifikasi Modul

| | |
|---|---|
| **Nama di sistem lama** | **Layar master** — `MasterRekening`, `MasterSupplier`, `MasterRecovery`, `MasterPanel_HE`, `MasterLoginSurvey`, `MasterProteksiVisibilityData`, `PNC_MasterTolakKlaim`, `DetailMasterXOL`, `DetailMasterPasalRejected`, `DetailCauseOfLoss`, `DetailDominanFactor`, `GCNMCatSparepart`, `GCNMMasterSparepartType`, `GroupingSparePart_HE`, `SparePart_HE`, `BengkelHE`, `BrowseMasterDocumentTravel_Harness`, dan seterusnya |
| **Kode modul** | `U-6` |
| **Gelombang** | 7 — Sisa |
| **Ukuran** | **Besar** — naik dari Sedang setelah verifikasi |
| **Bergantung pada** | `U-2` Komponen Layar Baku · `F-4` Master Data & Parameter Bisnis |
| **Kesiapan** | **TERHALANG** |

### Apa yang dikerjakan modul ini

Layar tempat administrator **mengelola data acuan** yang dipakai seluruh sistem: rekening,
supplier, bengkel, sparepart, panel, sebab kerugian, pasal penolakan, XOL, dan seterusnya.

### Temuan yang mengubah rencana

| Anggapan | Terverifikasi |
|---|---|
| Master data berjumlah 14 kelompok | **≥29 kelompok** — `U-6` naik dari **Sedang menjadi Besar**, dan `F-4` ikut naik |

### Kenapa modul ini TERHALANG

Karena **jumlah kelompok master masih dinyatakan sebagai "≥29"**, bukan sebagai angka pasti. Modul
yang tidak tahu berapa layar yang harus dibuatnya tidak dapat dinyatakan siap.

### Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | **Berapa persisnya kelompok master?** Angka yang ada baru batas bawah (**≥29**) | **Work Owner** |
| **Keputusan** | Siapa berwenang mengubah tiap kelompok master — kontrol sekarang **berbasis menu**, bukan berbutir aksi | **Work Owner** (`D-59`) |
| **Keputusan** | Pembagian 48 harness non-Inbox ke `U-4`/`U-5`/`U-6` per berkas | **Work Owner + Lead Engineer** |
| **Bergantung** | Penghapusan lunak dan riwayat perubahan master (`F-4`, `S-5`) | — (`D-…` §8.1 `09-DATABASE`) |

### Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-U6-001](issues/01-layar-master-baku.md) | Layar master baku (CRUD seragam) | `needs-info` |
| [TKT-U6-002](issues/02-kelompok-master-dan-kewenangannya.md) | Kelompok master dan kewenangan pengubahnya | `needs-info` |

> **Daftar lengkap 74 harness beserta kelasnya:** [`../INVENTARIS-HARNESS.md`](../INVENTARIS-HARNESS.md)
> — pembagian modulnya **belum diputuskan** (`D-73`).

## Daftar Tiket

### TKT-U6-001 — Layar master baku (CRUD seragam)

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan jumlah kelompok master** |
| **Modul** | **U-6 Layar Master Data** · Gelombang: 7 · Bergantung pada: TKT-U2-001, TKT-U2-002, TKT-F4-001 |
| **Requirement** | FR-U6 |
| **Keputusan** | D-59 |
| **ADR** | 0012, 0023 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | pola bersama harness master — `MasterRekening`, `MasterSupplier`, `MasterRecovery`, `MasterPanel_HE`, `DetailCauseOfLoss`, `DetailDominanFactor`, `GCNMCatSparepart`, `SparePart_HE`, `BengkelHE` |
| **Peran penguji gerbang 2** | **PncAdmin**, **PncManagerAdmin** |
| **Label** | `modul::U-6` `tipe::fondasi` `status::needs-info` `prioritas::sedang` `gelombang::7` |
| **Milestone** | Gelombang 7 — Sisa |
| **Berkas sumber** | `ticketing/U-6-Layar-Master-Data/issues/01-layar-master-baku.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Satu layar pengelolaan master yang benar, dipakai ulang **≥29 kali**.

Nilai bisnisnya sama dengan `U-2` dan `U-3`: dengan sedikitnya 29 kelompok master, membuat tiap
layar sendiri-sendiri berarti 29 halaman yang perilakunya berbeda-beda. Satu layar baku berarti
menambah kelompok master menjadi **pekerjaan konfigurasi**, bukan pekerjaan pemrograman.

#### Ruang lingkup

- Layar baku: daftar bergrid, tambah, ubah, dan **penghapusan lunak** (`09-DATABASE` §8.1) —
  master tidak dihapus permanen karena data klaim lama merujuknya.
- Definisi kolom, validasi, dan kewenangan per kelompok sebagai **konfigurasi**.
- Setiap perubahan master **tercatat di jejak audit** (`S-5`) — siapa mengubah, dari apa menjadi apa.
- Pencarian dan paginasi dari server.

#### Non-goal

- **Tidak** mendaftar kelompok master satu per satu — itu `TKT-U6-002`.
- **Tidak** memutuskan aturan bisnis isi master; itu `F-4`.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Berapa persisnya kelompok master?** Yang ada baru **batas bawah: ≥29** | **Work Owner** | Modul yang tidak tahu berapa layar yang harus dibuatnya tidak dapat dinyatakan selesai secara terukur |
| **Setiap master butuh persetujuan sebelum berlaku, atau berlaku langsung?** | **Work Owner** | Sebagian master menentukan **hasil hitungan uang** — mengubahnya tanpa persetujuan mengubah nilai klaim yang dihitung sesudahnya |
| **Perubahan master berlaku surut ke klaim yang sudah ada?** | **Work Owner** | Menentukan apakah dibutuhkan versi bertanggal pada master |

#### Acceptance criteria

- ☐ Layar master baku dipakai seluruh kelompok — tidak ada layar master buatan sendiri,
      diperiksa.
- ☐ Menambah kelompok master baru **tidak menuntut halaman baru** — diuji dengan menambah satu
      kelompok lewat konfigurasi saja.
- ☐ **Penghapusan bersifat lunak**; baris yang dirujuk klaim lama **tetap terbaca** — diuji
      dengan menghapus master yang dipakai klaim lama: klaim itu tetap tampil utuh.
- ☐ Setiap perubahan **tercatat di jejak audit** dengan nilai sebelum dan sesudah — diuji.
- ☐ Peran yang tidak berwenang **tidak dapat mengubah** master lewat pemanggilan langsung —
      diuji: `403` (`ADR-0023`).
- ☐ Tabel memakai `TKT-U2-001` dan form memakai `TKT-U2-002` — diperiksa.
- ☐ Gerbang 2: UAT **PncAdmin** dan **PncManagerAdmin**.

#### Dependency / Blocked by

`TKT-U2-001` · `TKT-U2-002` · `TKT-F4-001` · `TKT-S5-001`. **Terhalang Work Owner.**

#### Constraint keamanan, data, operasional

- **Master data menentukan hasil hitungan klaim.** Mengubah satu baris master dapat mengubah nilai
  yang dibayarkan — karena itu jejak auditnya bukan tambahan, melainkan syarat.
- **Penghapusan permanen akan merusak klaim lama** yang merujuk baris itu (`09-DATABASE` §8.1).
- Kontrol sekarang **berbasis menu** (`D-59`) — artinya siapa pun yang punya menu master dapat
  mengubah **seluruh** kelompok di dalamnya, bukan sebagian.

#### Migrasi skema / rollout / rollback

Menambah kolom penanda hapus lunak pada tabel master yang belum punya. Backward-compatible (`P-4`).

**Rollback:** mengembalikan versi layar. Perubahan master yang telanjur dilakukan **tetap ada** —
dan telah memengaruhi klaim yang dihitung sesudahnya.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm test -- master --run TestSatuLayarBakuUntukSemuaKelompok
go test ./internal/app/master/... -run TestHapusLunakTidakMerusakKlaimLama
go test ./internal/app/audit/... -run TestPerubahanMasterTercatat
go test ./internal/adapter/http/... -run TestUbahMasterTanpaKewenanganDitolak
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Master data ≥29 kelompok; `U-6` naik menjadi Besar | `docs/Steering/06-MODULE-BREAKDOWN.md` koreksi ukuran 2026-09-14 |
| Penghapusan lunak pada master | `docs/Steering/09-DATABASE-STRATEGY.md` §8.1 |
| Kontrol berbasis menu | `D-59` |
| Otorisasi diperiksa di setiap endpoint | `ADR-0023` |
| Tabel baku median 6 kolom | `docs/Steering/06-MODULE-BREAKDOWN.md:76` |

#### Comments

### TKT-U6-002 — Kelompok master dan kewenangan pengubahnya

| | |
|---|---|
| **Status** | needs-info |
| **Kesiapan** | **terhalang keputusan — jumlah kelompok belum pasti** |
| **Modul** | **U-6 Layar Master Data** · Gelombang: 7 · Bergantung pada: TKT-U6-001 |
| **Requirement** | FR-U6 |
| **Keputusan** | D-58, D-59 |
| **ADR** | 0023 |
| **Risiko** | — |
| **Rule Pega yang digantikan** | harness master — `MasterRekening`, `MasterSupplier`, `MasterRecovery`, `MasterPanel_HE`, `MasterLoginSurvey`, `MasterProteksiVisibilityData`, `PNC_MasterTolakKlaim`, `DetailMasterXOL`, `DetailMasterPasalRejected`, `DetailCauseOfLoss`, `DetailDominanFactor`, `GCNMCatSparepart`, `GCNMMasterSparepartType`, `GroupingSparePart_HE`, `SparePart_HE`, `BengkelHE`, `BrowseMasterDocumentTravel_Harness`, `ListDetTypeDocument`, `DetTypeDocumenBisnis`, `ListDocumentObject`, `ListDocumentTravel` |
| **Peran penguji gerbang 2** | **pemilik masing-masing kelompok master** |
| **Label** | `modul::U-6` `tipe::migrasi` `status::needs-info` `prioritas::sedang` `gelombang::7` |
| **Milestone** | Gelombang 7 — Sisa |
| **Berkas sumber** | `ticketing/U-6-Layar-Master-Data/issues/02-kelompok-master-dan-kewenangannya.md` |

#### Hasil yang diharapkan (dan nilai bisnisnya)

Daftar lengkap kelompok master, masing-masing dengan **satu peran yang berwenang mengubahnya**.

Nilai bisnisnya ada pada kata "satu peran yang berwenang". Master menentukan hasil hitungan klaim —
rekening tujuan, tarif sparepart, pasal penolakan, batas XOL. Saat ini kendalinya **berbasis menu**
(`D-59`), yang berarti siapa pun yang punya menu master dapat mengubah **seluruh** isinya. Tanpa
pemetaan yang eksplisit, sistem baru akan menyalin keluasan itu tanpa ada yang memutuskannya.

#### Ruang lingkup

- **Daftar kelompok master lengkap**, sebagai tabel — bukan sebagai "≥29".
- Pemetaan **kelompok → peran yang berwenang mengubah**, dari 22 peran bisnis (`D-58`).
- Konfigurasi kolom dan validasi per kelompok di atas layar baku `TKT-U6-001`.
- Pemisahan mana yang **master** dan mana yang **daftar acuan tampilan** — sebagian harness
  (`ListDocumentObject`, `ListDocumentTravel`) mungkin termasuk yang kedua.

#### Non-goal

- **Tidak** menambah kelompok master baru.
- **Tidak** mengubah isi master yang berlaku.

#### Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Berapa persisnya kelompok master?** Yang tercatat baru **batas bawah: ≥29** | **Work Owner** | Tiket ini menjanjikan sebuah daftar; tanpa angka pastinya, daftar itu tidak dapat dinyatakan lengkap |
| **Kelompok mana boleh diubah peran mana?** | **Work Owner** (`D-58`, `D-59`) | Kendali sekarang berbasis menu, jadi pemetaan berbutir ini **belum pernah ada**. Membuatnya adalah **pengetatan** — perubahan perilaku yang harus disepakati, bukan diselipkan |
| **Master mana yang memengaruhi hitungan uang?** | **Work Owner** | Kelompok itu layak diperlakukan lebih ketat daripada master tampilan biasa |

#### Acceptance criteria

- ☐ **Daftar kelompok master lengkap** tercatat sebagai tabel, dan jumlahnya **dilaporkan sebagai
      angka pasti** — bukan "≥29".
- ☐ Setiap kelompok punya **peran yang berwenang mengubahnya**, tercatat dan ditegakkan di server
      — diuji per kelompok dengan peran yang tidak berwenang: `403`.
- ☐ Harness yang dinyatakan **bukan master** dipindahkan ke modul yang tepat, **tidak hilang** —
      dicatat per berkas.
- ☐ Isi tiap kelompok **sama dengan isi di Pega** pada saat cutover — dibandingkan lewat `S-8`.
- ☐ **Pengetatan kewenangan dilaporkan sebagai perubahan perilaku yang disengaja** (`P-5`), bukan
      sebagai perbaikan diam-diam — setiap peran yang kehilangan akses **diberi tahu lebih dulu**.
- ☐ Gerbang 2: UAT oleh **pemilik masing-masing kelompok master**, bukan satu orang untuk semuanya.

#### Dependency / Blocked by

`TKT-U6-001` · `TKT-F3-002` (22 peran) · `TKT-F4-001`. **Terhalang Work Owner.**

#### Constraint keamanan, data, operasional

- **Mempersempit kewenangan akan mengambil akses dari orang yang selama ini memilikinya.** Itu
  perbaikan, tetapi ia terasa sebagai kehilangan bagi yang terkena — dan harus disampaikan sebelum
  cutover, bukan ditemukan pada hari pertama.
- Master yang memengaruhi hitungan uang **mengubah nilai klaim yang dihitung sesudahnya**; jejak
  auditnya (`TKT-U6-001`) adalah satu-satunya cara menelusuri sebabnya.
- `MasterProteksiVisibilityData` menyangkut **batas keterlihatan data** — mengubahnya mengubah siapa
  melihat apa di seluruh sistem.

#### Migrasi skema / rollout / rollback

Menambah tabel pemetaan kelompok master ke peran. Tidak mengubah tabel master itu sendiri.

**Rollback:** melonggarkan kembali kewenangan **mengembalikan keadaan di mana siapa pun bermenu
master dapat mengubah semuanya**. Itu bukan rollback yang netral.

#### Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/cek-master --daftar-lengkap
go test ./internal/adapter/http/... -run TestKewenanganPerKelompokMaster
go run ./cmd/s8 banding --modul U-6 --per-kelompok
```

#### Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Master data ≥29 kelompok | `docs/Steering/06-MODULE-BREAKDOWN.md` koreksi ukuran 2026-09-14 |
| 22 peran bisnis, satu-untuk-satu dengan access group | `D-58` |
| Kontrol berbasis menu, bukan berbutir aksi | `D-59` |
| Harness master di export | direktori `Harness/` |
| Otorisasi diperiksa di setiap endpoint | `ADR-0023` |

#### Comments

# Lampiran A — Inventaris 74 Harness

| | |
|---|---|
| **Tabel lengkapnya** | [`../Steering/22-INVENTARIS-HARNESS.md`](../Steering/22-INVENTARIS-HARNESS.md) |
| **Sumber** | direktori `Harness/` — dibaca langsung oleh `docs/tools/build-inventaris-harness.js` |
| **Dasar** | `D-73` · pertanyaan terbuka `U-3` dan `U-4`/`U-5`/`U-6` |

> **Tabel di bawah dibangkitkan generator yang sama** dengan lampiran Steering> (`docs/tools/build-inventaris-harness.js`), langsung dari direktori `Harness/`. Keduanya karena> itu **tidak dapat berbeda**: yang disunting tangan hanya pengantar di atas tabel, bukan tabelnya.
## Kenapa pemisahan ini penting bagi tiket

Empat tiket bergantung langsung pada pembagian yang belum diputuskan:

| Tiket | Bergantung pada |
|---|---|
| [TKT-U3-002](U-3-Layar-Inbox-per-Peran/issues/02-inbox-per-peran-dan-pemetaannya.md) | Mana dari 26 harness bernama *Inbox* yang benar-benar **inbox peran** |
| [TKT-U4-001](U-4-Layar-Transaksi-Klaim/issues/01-kerangka-layar-transaksi-per-tahap.md) | Pembagian harness non-Inbox ke `U-4`/`U-5`/`U-6` |
| [TKT-U5-001](U-5-Layar-Laporan/issues/01-layar-pemilih-laporan-dan-parameter.md) | idem |
| [TKT-U6-002](U-6-Layar-Master-Data/issues/02-kelompok-master-dan-kewenangannya.md) | idem |

## Pemisahan berdasarkan NAMA ternyata bukan pembeda yang berguna

Diuji terhadap export, dan **gagal**:

| Yang diuji | Hasil |
|---|---|
| Harness bernama *Inbox* memakai model penugasan? | **Tidak terbukti.** Penanda yang dipakai ternyata menangkap boilerplate Pega (`pyDashboardMyWorkList`); `InboxRegister_Harness` — inbox sungguhan — justru **nol kecocokan** |
| Nama *Inbox* sejalan dengan kelas harness? | **Tidak.** 23 dari 26 bernama Inbox berkelas `Data-Portal` — sama dengan 35 yang bukan bernama Inbox |

## Yang benar-benar berpengaruh pada implementasi

| Kelas | Jumlah | Akibatnya pada kode |
|---|---:|---|
| **`ASM-FW-GCNMFW-Work*`** | **7** | Terikat pada **satu klaim**; rutenya memuat nomor klaim, dan layarnya tidak berarti tanpa klaim yang dimuat. **Sudah pasti milik `U-4`** |
| **`@baseclass`** | **9** | Generik; konteks ditentukan pemanggil |
| **`Data-Portal`** | **58** | Berdiri sendiri di portal — daftar, master, laporan. Rute tidak memerlukan klaim |

Ketujuh yang sudah pasti `U-4`:

```
InputProgress · ViewDetailHasilSurveyorInternal1 · ViewReceiveDocument
ViewTempDetailAllCase · ViewTempDetailClaim · ViewTempDetailReqDocument
View_DetailKlaimCabang_Harness
```

## Apa yang berubah bila pembagiannya salah

| Terpengaruh | Bagaimana |
|---|---|
| **Ukuran `U-3`** | `U-3` membangun **satu layar inbox dipakai ulang**. Bila yang benar-benar inbox peran hanya 14 dan bukan 26, komponennya lebih kecil dan AC-nya berubah |
| **Urutan kerja** | Layar yang membaca model penugasan **tidak dapat dibangun sebelum `B-6`** |
| **Bentuk rute** | Layar terikat klaim butuh nomor klaim di rutenya; layar portal tidak |
| **Penguji gerbang 2** | Tiap layar diuji pemegang perannya — salah modul berarti salah penguji |
| **Kendali akses** | Pada inbox, penyaringan "hanya pekerjaan saya" **adalah kendali akses**; pada master, kendalinya "siapa boleh mengubah" |

## Definisi Inbox kini menjadi dasar penggolongan (`D-79`)

> **Inbox** = layar berisi **daftar pekerjaan milik pengguna** — Tugas dari Worklist/Workbasket.
> Layar **data acuan** bukan Inbox, sekalipun dapat dicari dan sekalipun namanya mengandung
> kata "Inbox".

Sebelum `D-79`, kolom **Usulan modul** bersandar pada **sidik jari berkas** — dasar yang
korelasional. Sekarang dasarnya **isi layar**; sidik jari tetap dipakai sebagai **alat baca**,
karena isi layar tidak dapat dibaca langsung dari XML. Yang berubah adalah kedudukannya: sidik
jari menjadi cara **menerapkan** definisi, bukan **pengganti** definisi.

Akibatnya **tujuh harness bernama *Inbox* diusulkan ke `U-6`**.

## Yang perlu diputuskan

1. ~~Mana yang benar-benar inbox peran.~~ — **definisinya tertutup `D-79`**. Yang tersisa
   adalah **koreksi atas usulan**: 27 bertanda `DUGAAN` dan 5 bertanda `JANGGAL`.
2. **Pemilik modul untuk 67 harness selain ketujuh yang berkelas Work.**

Kolom **Usulan modul** kini terisi — lihat bagian **Tingkat keyakinan** di atas. Setiap baris
ditandai PASTI / KUAT / DUGAAN / JANGGAL, sehingga usulan tidak dapat tersamar sebagai fakta.

## Sidik jari berkas mempersempit keputusan dari 74 menjadi segelintir

Ukuran berkas harness dan jumlah rujukan Report Definition di dalamnya ternyata mengelompok tajam.
Ini **bukti korelasional, bukan pernyataan tujuan** — tetapi cukup untuk menyingkirkan sebagian
besar dugaan.

| Kelompok | Jumlah | Sidik jari | Tafsiran |
|---|---:|---|---|
| Bernama *Inbox*, berkas **besar** | **10** | 971–1.885 KB · RD 69–276 | **Kandidat kuat inbox peran** |
| Bernama *Inbox*, ukuran **sedang** | 7 | 412–805 KB · RD 27–84 | Perlu ditinjau |
| Bernama *Inbox*, berkas **kecil** | 7 | **267–273 KB · RD 15** | **Hampir pasti BUKAN inbox peran** |
| Pembanding: `MasterRekening` (master murni) | — | **273 KB · RD 15** | — |

Ketujuh yang berkas kecil bersidik jari **nyaris identik dengan layar master**, bukan dengan
`InboxRegister_Harness` (1.330 KB · RD 144):

```
CauseOfLossInbox · CauseOfLossInboxSimasOnline · DetailSurveyorsInbox
ListDocumentTypeInbox · StatusClaimInbox · SurveyorsInbox · UserTeknisInbox
```

**Dua yang menyimpang dari pola dan perlu ditinjau khusus:**

| Harness | Sidik jari | Kenapa janggal |
|---|---|---|
| `inboxCompliance_Harness` | **144 KB · RD 0** | Terkecil dari seluruh 74 dan **tanpa satu pun Report Definition**. Layar antrean kerja tanpa sumber data adalah hal yang tidak masuk akal |
| `inboxAnalystDoctor_Harness` | 348 KB · RD 12 | Di bawah ambang kelompok master, tetapi ia menyangkut **data medis** (`FR-R2`) sehingga salah golong berakibat pada kendali akses |

Enam harness **tidak** bernama *Inbox* tetapi berkas besar — layar kerja sungguhan yang penamaannya
tidak memberi petunjuk: `PNCArchiveDokumen` · `ViewReceiveDocument` · `ReportKPIHarness` ·
`PNCTATReport` · `ProgressClaim_Harness` · `MasterProteksiVisibilityData`.

## Tingkat keyakinan pada kolom usulan

Kolom **Usulan modul** diisi oleh aturan di `docs/tools/build-inventaris-harness.js` (fungsi
`usul()`), bukan diketik tangan — sehingga dasarnya dapat diperiksa dan hasilnya dapat dibangun
ulang. **Seluruhnya usulan, bukan keputusan.**

| Tanda | Artinya | Jumlah |
|---|---|---:|
| **PASTI** | Bukti **struktural** dari export: kelas harness `ASM-FW-GCNMFW-Work*`. Tidak bergantung nama sama sekali | **7** |
| **KUAT** | **Dua bukti sejalan** — topik dari nama *dan* sidik jari berkas menunjuk arah yang sama | **35** |
| **DUGAAN** | **Satu bukti saja**, atau bukti yang saling bertentangan | **27** |
| **JANGGAL** | Sidik jarinya menyimpang dari pola mana pun — **wajib ditinjau manusia** | **5** |

**Cara nama berkas dipakai — dan tidak dipakai.** Nama dipakai untuk menduga **topik** layar
(rekening, sparepart, laporan). Nama **tidak** dipakai untuk menduga **pola interaksi**, karena itu
sudah diuji dan gagal. Pola diambil dari kelas harness dan sidik jari berkas.

Akibatnya, tujuh harness bernama *Inbox* justru diusulkan ke **`U-6`**, bukan `U-3` — sidik jarinya
sama dengan layar master, bukan dengan layar antrean kerja.

### Lima yang JANGGAL — tidak saya beri usulan sama sekali

| Harness | Sidik jari | Kenapa tidak diusulkan |
|---|---|---|
| `inboxCompliance_Harness` | 145 KB · RD 0 | Bernama inbox tetapi **tanpa satu pun sumber data**. Antrean kerja tanpa sumber data tidak masuk akal |
| `RCLPUCL_Harness` | 184 KB · RD 0 | `B-11` menyebutnya layar utama RCL/PUCL, tetapi isinya nyaris kosong |
| `DashboardClaim_Harness` | 155 KB · RD 0 | Dashboard tanpa sumber data |
| `ViewPolis1` | 122 KB · RD 0 | Kembar dengan `ViewPolis`; salah satunya mungkin sudah mati |
| `GCNMCatSparepart` | 85 KB · RD 0 | **Terkecil dari 74** |

Kelimanya berbagi satu pola: **`RD = 0`**. Kemungkinan besar cangkang tipis — layar yang isinya
dipasok dari tempat lain, atau layar yang sudah tidak dipakai. Keduanya berakibat berbeda pada
lingkup, dan **BELUM DIPUTUSKAN — pertanyaan terbuka** (pemilik: **Work Owner + Tim Pega**).

### Yang tetap tidak berubah

Usulan ini **tidak menutup** pertanyaan terbuka `U-3` maupun `U-4`/`U-5`/`U-6`. Ia hanya mengubah
bentuk pekerjaan Work Owner: dari **mengklasifikasi 74 berkas** menjadi **mengoreksi 27 DUGAAN dan
memutuskan 5 JANGGAL**. Yang bertanda **PASTI** tidak perlu ditinjau; yang **KUAT** cukup dibaca
sekilas.

## Inventaris

| Harness | Kelas | KB | RD | Bernama *Inbox* | **Usulan modul** | Keyakinan |
|---|---|---:|---:|---|---|---|
| `AutoKlaim` | Data-Portal | 276 | 9 | — | **U-4** | DUGAAN |
| `BengkelHE` | Data-Portal | 267 | 15 | — | **U-6** | KUAT |
| `BrowseMasterDocumentTravel_Harness` | Data-Portal | 278 | 21 | — | **U-6** | DUGAAN |
| `CauseOfLossInbox` | Data-Portal | 268 | 15 | ya | **U-6** | KUAT |
| `CauseOfLossInboxSimasOnline` | Data-Portal | 274 | 15 | ya | **U-6** | KUAT |
| `DashboardClaim_Harness` | @baseclass | 155 | 0 | — | **?** | JANGGAL |
| `DetailCauseOfLoss` | Data-Portal | 268 | 15 | — | **U-6** | KUAT |
| `DetailDominanFactor` | Data-Portal | 316 | 27 | — | **U-6** | DUGAAN |
| `DetailMasterPasalRejected` | Data-Portal | 274 | 15 | — | **U-6** | KUAT |
| `DetailMasterXOL` | Data-Portal | 163 | 9 | — | **U-6** | DUGAAN |
| `DetailSurveyorsInbox` | Data-Portal | 268 | 15 | ya | **U-6** | KUAT |
| `DetTypeDocumenBisnis` | Data-Portal | 268 | 15 | — | **U-6** | KUAT |
| `GCNMCatSparepart` | Data-Portal | 85 | 0 | — | **?** | JANGGAL |
| `GCNMMasterSparepartType` | Data-Portal | 419 | 30 | — | **U-6** | DUGAAN |
| `GroupingSparePart_HE` | Data-Portal | 269 | 15 | — | **U-6** | KUAT |
| `Har_LaporanHasilAI` | Data-Portal | 466 | 45 | — | **U-5** | KUAT |
| `Inbox_XOL_Harness` | Data-Portal | 971 | 69 | ya | **U-3** | KUAT |
| `inboxAnalystDoctor_Harness` | Data-Portal | 348 | 12 | ya | **U-3** | DUGAAN |
| `InboxAutoClaim` | Data-Portal | 1664 | 183 | ya | **U-3** | KUAT |
| `InboxClaimNonProp_Harness` | Data-Portal | 1284 | 120 | ya | **U-3** | KUAT |
| `InboxClaimTreaty_Harness` | Data-Portal | 1538 | 69 | ya | **U-3** | KUAT |
| `inboxCompliance_Harness` | Data-Portal | 145 | 0 | ya | **?** | JANGGAL |
| `InboxInvestigator_Harness` | Data-Portal | 456 | 33 | ya | **U-3** | DUGAAN |
| `InboxKomite_Harness` | Data-Portal | 1759 | 216 | ya | **U-3** | KUAT |
| `InboxKomunikasiCabang` | @baseclass | 682 | 84 | ya | **U-3** | DUGAAN |
| `InboxManagerAdmin_Harness` | Data-Portal | 482 | 30 | ya | **U-3** | DUGAAN |
| `InboxPLA_harness` | Data-Portal | 706 | 27 | ya | **U-3** | DUGAAN |
| `InboxPLADLA` | Data-Portal | 1599 | 126 | ya | **U-3** | KUAT |
| `InboxRCVApp_Harness` | @baseclass | 1126 | 126 | ya | **U-3** | KUAT |
| `InboxRegister_Harness` | Data-Portal | 1331 | 144 | ya | **U-3** | KUAT |
| `InboxSalvage` | Data-Portal | 413 | 42 | ya | **U-3** | DUGAAN |
| `InboxSurvey_Harness` | Data-Portal | 1796 | 150 | ya | **U-3** | KUAT |
| `InboxTKA_Harness` | Data-Portal | 774 | 51 | ya | **U-3** | DUGAAN |
| `InputProgress` | Work (konteks klaim) | 251 | 12 | — | **U-4** | PASTI |
| `InputProtection_Harness` | Data-Portal | 483 | 54 | — | **U-4** | DUGAAN |
| `InputReqProtection_Harness` | Data-Portal | 232 | 21 | — | **U-4** | DUGAAN |
| `ListDetTypeDocument` | Data-Portal | 268 | 15 | — | **U-6** | KUAT |
| `ListDocumentObject` | Data-Portal | 268 | 15 | — | **U-6** | KUAT |
| `ListDocumentTravel` | Data-Portal | 268 | 15 | — | **U-6** | KUAT |
| `ListDocumentTypeInbox` | Data-Portal | 268 | 15 | ya | **U-6** | KUAT |
| `MasterLoginSurvey` | Data-Portal | 268 | 15 | — | **U-6** | KUAT |
| `MasterPanel_HE` | Data-Portal | 273 | 15 | — | **U-6** | KUAT |
| `MasterProteksiVisibilityData` | Data-Portal | 1011 | 66 | — | **U-6** | DUGAAN |
| `MasterRecovery` | Data-Portal | 361 | 27 | — | **U-6** | DUGAAN |
| `MasterRekening` | Data-Portal | 274 | 15 | — | **U-6** | KUAT |
| `MasterSupplier` | Data-Portal | 322 | 15 | — | **U-6** | DUGAAN |
| `MonitoringSLINKOJK` | Data-Portal | 198 | 3 | — | **U-5** | DUGAAN |
| `OutstandingKlaimperCabang_Harness` | Data-Portal | 469 | 18 | — | **U-5** | KUAT |
| `PNC_MasterTolakKlaim` | Data-Portal | 276 | 15 | — | **U-6** | KUAT |
| `PNCArchiveDokumen` | Data-Portal | 1282 | 135 | — | **U-4** | DUGAAN |
| `PNCInboxAdmin` | @baseclass | 1885 | 276 | ya | **U-3** | KUAT |
| `PNCSearchKlaim` | @baseclass | 574 | 36 | — | **U-4** | DUGAAN |
| `PNCStudyClaim` | Data-Portal | 476 | 42 | — | **U-5** | KUAT |
| `PNCTATReport` | @baseclass | 1127 | 114 | — | **U-5** | KUAT |
| `ProgressClaim_Harness` | @baseclass | 1024 | 93 | — | **U-4** | DUGAAN |
| `RCL_Harness` | Data-Portal | 214 | 9 | — | **U-4** | DUGAAN |
| `RCLPUCL_Harness` | Data-Portal | 184 | 0 | — | **?** | JANGGAL |
| `ReceiveDoucument_Harness` | Data-Portal | 502 | 15 | — | **U-4** | DUGAAN |
| `ReportKPIHarness` | Data-Portal | 1620 | 78 | — | **U-5** | KUAT |
| `SparePart_HE` | Data-Portal | 268 | 15 | — | **U-6** | KUAT |
| `StatusClaimInbox` | Data-Portal | 267 | 15 | ya | **U-6** | KUAT |
| `StatusProgress` | Data-Portal | 267 | 15 | — | **U-5** | DUGAAN |
| `StatusProgress2` | Data-Portal | 267 | 15 | — | **U-5** | DUGAAN |
| `SurveyorsInbox` | Data-Portal | 268 | 15 | ya | **U-6** | KUAT |
| `UserInbox_Harness` | Data-Portal | 805 | 30 | ya | **U-3** | DUGAAN |
| `UserTeknisInbox` | Data-Portal | 270 | 15 | ya | **U-6** | KUAT |
| `View_DetailKlaimCabang_Harness` | Work (konteks klaim) | 734 | 54 | — | **U-4** | PASTI |
| `ViewDetailHasilSurveyorInternal1` | Work (konteks klaim) | 516 | 33 | — | **U-4** | PASTI |
| `ViewPolis` | @baseclass | 431 | 27 | — | **U-4** | DUGAAN |
| `ViewPolis1` | @baseclass | 122 | 0 | — | **?** | JANGGAL |
| `ViewReceiveDocument` | Work (konteks klaim) | 1238 | 132 | — | **U-4** | PASTI |
| `ViewTempDetailAllCase` | Work (konteks klaim) | 493 | 0 | — | **U-4** | PASTI |
| `ViewTempDetailClaim` | Work (konteks klaim) | 537 | 3 | — | **U-4** | PASTI |
| `ViewTempDetailReqDocument` | Work (konteks klaim) | 533 | 0 | — | **U-4** | PASTI |

**Total: 74 harness** — 7 berkelas Work · 9 `@baseclass` · 58 `Data-Portal` · 26 di antaranya bernama *Inbox*.

> Tabel di atas **dibangun ulang** oleh `docs/tools/build-inventaris-harness.js` langsung dari
> direktori `Harness/`, termasuk kolom **Usulan modul** dan **Keyakinan** yang dihitung dari
> aturan di `usul()`. Jangan menyuntingnya dengan tangan — koreksi Work Owner dicatat di
> Decision Log, lalu aturannya disesuaikan di sini agar tabel tetap dapat dibangun ulang.

# Lampiran B — Indeks Seluruh Tiket

| Modul | Tiket | Status |
|---|---|---|
| F-1 | TKT-F1-001 — Kerangka proyek Go dan penegakan aturan lapisan | ready-for-human |
| F-1 | TKT-F1-002 — Konfigurasi tiga lapis dan gagal keras saat start | ready-for-human |
| F-1 | TKT-F1-003 — Logging terstruktur dan ID permintaan | ready-for-human |
| F-1 | TKT-F1-004 — Penanganan galat terpusat dan kontrak galat API | needs-info |
| F-1 | TKT-F1-005 — Health check, graceful shutdown, dan penyajian SPA | ready-for-human |
| F-2 | TKT-F2-001 — Koneksi, connection pool, dan seam Repository | ready-for-human |
| F-2 | TKT-F2-002 — Disiplin SQL portabel dan pemeriksaan pola terlarang | ready-for-human |
| F-2 | TKT-F2-003 — Kepemilikan transaksi di lapisan aplikasi | ready-for-human |
| F-2 | TKT-F2-004 — Kerangka migrasi skema backward-compatible dan rollback | needs-info |
| F-2 | TKT-F2-005 — Soft delete sebagai pola akses data | ready-for-human |
| F-2 | TKT-F2-006 — Generator nomor klaim `PNCN.YY.xxxx` | needs-info |
| F-2 | TKT-F2-007 — Daftar putih kolom filter dan sort pengganti `{ASIS:}` | needs-info |
| F-3 | TKT-F3-001 — Seam Identity dan adapter fake untuk pengembangan | ready-for-human |
| F-3 | TKT-F3-002 — Adapter autentikasi HCC/HCQ | needs-info |
| F-3 | TKT-F3-003 — Sesi dan token milik aplikasi | ready-for-human |
| F-3 | TKT-F3-004 — Tabel 22 peran dan 51 izin menu | needs-info |
| F-3 | TKT-F3-005 — Middleware otorisasi di setiap endpoint | ready-for-human |
| F-4 | TKT-F4-001 — Kerangka master data dan layar CRUD baku | needs-info |
| F-4 | TKT-F4-002 — Master Ambang Komite dan validasi integritas tangga | needs-info |
| F-4 | TKT-F4-003 — Master Penerima Notifikasi (mailbox fungsional) | ready-for-human |
| F-4 | TKT-F4-004 — Master Mata Uang dan Kurs per tanggal | needs-info |
| F-4 | TKT-F4-005 — Master Status Klaim, kalender libur, dan ambang uang | needs-info |
| F-4 | TKT-F4-006 — Inventaris ≥29 kelompok master dan penetapan lingkup | needs-info |
| F-5 | TKT-F5-001 — Seam Clock dan penyimpanan UTC | needs-info |
| F-5 | TKT-F5-002 — Konversi WIB tunggal dan penghapusan penyesuaian 7 jam | needs-info |
| F-5 | TKT-F5-003 — Kalender hari kerja dan hari libur | needs-info |
| F-6 | TKT-F6-001 — Daftar portal sebagai data, bukan konstanta | needs-info |
| F-6 | TKT-F6-002 — Koneksi dan pool per portal | needs-info |
| F-6 | TKT-F6-003 — Perpindahan portal dan penilaian ulang kewenangan | needs-info |
| F-6 | TKT-F6-004 — Kesekerabatan skema empat database | needs-info |
| B-1 | TKT-B01-001 — Layar View Polis dan pembekuan snapshot | needs-info |
| B-1 | TKT-B01-002 — Skema snapshot polis pengganti JSON bebas bentuk | needs-info |
| B-1 | TKT-B01-003 — Aturan penyegaran snapshot | needs-info |
| B-2 | TKT-B02-001 — Layar Input Register dan penerbitan nomor klaim | needs-info |
| B-2 | TKT-B02-002 — Validasi urutan tanggal dan periode polis | needs-info |
| B-2 | TKT-B02-003 — Validasi klaim ganda | needs-info |
| B-2 | TKT-B02-004 — Validasi kelengkapan dan Notice of Large Losses | ready-for-human |
| B-2 | TKT-B02-005 — Konversi ulang klaim yang idempoten | needs-info |
| B-3 | TKT-B03-001 — Objek Pertanggungan per lini bisnis | needs-info |
| B-3 | TKT-B03-002 — Coverage, penyebab kerugian, dan rincian item | needs-info |
| B-3 | TKT-B03-003 — Validasi TSI dan Sisa TSI | needs-info |
| B-4 | TKT-B04-001 — Pembagian share dan validasi total 100% | needs-info |
| B-4 | TKT-B04-002 — Fac Out, Fac Offer, dan Ex-Gratia | needs-info |
| B-4 | TKT-B04-003 — Penulisan spreading sebagai satu transaksi | needs-info |
| B-5 | TKT-B05-001 — Layar Input Estimasi dan Settlement Line | needs-info |
| B-5 | TKT-B05-002 — Konversi kurs dan perbandingan terhadap ambang | needs-info |
| B-5 | TKT-B05-003 — Fee adjuster, risiko sendiri, dan salvage pada nilai | needs-info |
| B-6 | TKT-B06-001 — Model penugasan Worklist dan Workbasket | ready-for-human |
| B-6 | TKT-B06-002 — Aturan routing dan pembagian beban | needs-info |
| B-6 | TKT-B06-003 — Penguncian tugas antar pengguna | ready-for-human |
| B-7 | TKT-B07-001 — Penentuan jenjang komite kumulatif | needs-info |
| B-7 | TKT-B07-002 — Layar Komite dan pencatatan keputusan | needs-info |
| B-7 | TKT-B07-003 — Persetujuan otomatis `AutoAcceptKomite` | needs-info |
| B-8 | TKT-B08-001 — Penugasan surveyor dan pencatatan hasil survei | needs-info |
| B-8 | TKT-B08-002 — Komponen dan skor KPI adjuster | needs-info |
| B-9 | TKT-B09-001 — Penerbitan PLA, Pre-DLA, dan DLA | needs-info |
| B-9 | TKT-B09-002 — Pengiriman dokumen ke reasuransi dan koasuransi | needs-info |
| B-10 | TKT-B10-001 — Akseptasi dan penerbitan Nomor Akseptasi | needs-info |
| B-10 | TKT-B10-002 — Transfer ke kasir dan status pembayaran | needs-info |
| B-10 | TKT-B10-003 — Cetak LOD ke tertanggung | needs-info |
| B-11 | TKT-B11-001 — Penolakan (RCL) dan proses ulang (PUCL) | needs-info |
| B-11 | TKT-B11-002 — Compliance, Investigator, dan Analyst Doctor | needs-info |
| B-12 | TKT-B12-001 — Pencatatan salvage dan item lelang | needs-info |
| B-12 | TKT-B12-002 — Recovery dan Virtual Account | needs-info |
| B-13 | TKT-B13-001 — Layar permintaan buka proteksi | ready-for-human |
| B-13 | TKT-B13-002 — Penautan proteksi ke klaim dan penandaan terpakai | needs-info |
| B-14 | TKT-B14-001 — Layar Input Receive Document | ready-for-human |
| B-14 | TKT-B14-002 — Inbox penerimaan dokumen dan penugasannya | needs-info |
| B-14 | TKT-B14-003 — Kaitan dokumen ke klaim dan tanggal terima | ready-for-human |
| S-1 | TKT-S1-001 — Satu jalur unggah dan metadata dokumen | needs-info |
| S-1 | TKT-S1-002 — Akses dokumen, token, dan masa berlakunya | needs-info |
| S-2 | TKT-S2-001 — Engine PDF, Excel, dan CSV | needs-info |
| S-2 | TKT-S2-002 — 56 laporan dan penghapusan batas 500 baris | needs-info |
| S-3 | TKT-S3-001 — Seam Notifier berbasis peristiwa domain | needs-info |
| S-3 | TKT-S3-002 — Template, penerima, dan pengiriman email | needs-info |
| S-4 | TKT-S4-001 — Klien REST keluar dan penanganan kegagalannya | needs-info |
| S-4 | TKT-S4-002 — Empat layanan REST masuk dan otentikasinya | needs-info |
| S-4 | TKT-S4-003 — Enam API pengganti DB Link | needs-info |
| S-5 | TKT-S5-001 — Skema jejak audit append-only dan hak akses database | ready-for-human |
| S-5 | TKT-S5-002 — Pencatatan otomatis pada perubahan bernilai bisnis | ready-for-human |
| S-5 | TKT-S5-003 — Daftar peristiwa wajib audit | needs-info |
| S-5 | TKT-S5-004 — Retensi dan arsip sebagai parameter konfigurasi | ready-for-human |
| S-6 | TKT-S6-001 — Penjadwal dan kunci satu pelaksana | needs-info |
| S-6 | TKT-S6-002 — Lima job terjadwal dan satu agent | needs-info |
| S-6 | TKT-S6-003 — Persetujuan komite otomatis jam 06:00 | needs-info |
| S-7 | TKT-S7-001 — Basis perhitungan TAT | needs-info |
| S-7 | TKT-S7-002 — Dashboard klaim dan posisi progres | needs-info |
| S-8 | TKT-S8-001 — Penembak kasus dan perekam hasil | needs-info |
| S-8 | TKT-S8-002 — Pembanding hasil dan klasifikasi selisih terhadap 13 butir P-5 | ready-for-human |
| U-1 | TKT-U1-001 — Kerangka SPA: routing, tata letak, dan state global | ready-for-human |
| U-1 | TKT-U1-002 — Alur masuk dan penanganan sesi di frontend | ready-for-human |
| U-1 | TKT-U1-003 — Penanganan galat dan notifikasi global | ready-for-human |
| U-1 | TKT-U1-004 — Peta rute dari 74 harness dan 51 item menu | needs-info |
| U-2 | TKT-U2-001 — Komponen tabel baku dengan paginasi keyset server-side | ready-for-human |
| U-2 | TKT-U2-002 — Komponen form baku dan penyajian galat validasi | ready-for-human |
| U-2 | TKT-U2-003 — Komponen unggah berkas | ready-for-human |
| U-2 | TKT-U2-004 — Pemilih tanggal dan pemformatan tanggal serta uang terpusat | ready-for-human |
| U-2 | TKT-U2-005 — Pemilihan pustaka tabel: TanStack Table versus AG Grid | ready-for-human |
| U-3 | TKT-U3-001 — Layar inbox baku dan penyaringnya | needs-info |
| U-3 | TKT-U3-002 — Inbox per peran dan pemetaan dari harness | needs-info |
| U-4 | TKT-U4-001 — Kerangka layar transaksi per tahap klaim | needs-info |
| U-4 | TKT-U4-002 — Layar pencarian dan detail klaim | needs-info |
| U-5 | TKT-U5-001 — Layar pemilih laporan dan parameternya | needs-info |
| U-5 | TKT-U5-002 — Unduhan hasil dan antrean export besar | needs-info |
| U-6 | TKT-U6-001 — Layar master baku (CRUD seragam) | needs-info |
| U-6 | TKT-U6-002 — Kelompok master dan kewenangan pengubahnya | needs-info |

