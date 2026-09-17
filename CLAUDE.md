# STEERING DOCUMENT

## Migrasi Aplikasi CLAIM PNC
### Dari Pega PRPC 8.3 ke Golang + React + PostgreSQL

| | |
|---|---|
| **Status** | Draft v2.0 — Menunggu Review |
| **Owner** | PT. Asuransi Sinar Mas — Claim PNC Migration Project |
| **Tanggal** | 14 September 2026 |
| **Versi** | 2.0 |
| **Klasifikasi** | CONFIDENTIAL |
| **Traceability** | Seluruh keputusan bersumber dari Lampiran B — Decision Log (`D-01` … `D-72`) |
| **Revisi** | v2.0 — menyerap 41 keputusan Sesi 3, 14 temuan verifikasi bukti, dan 29 ADR. Ringkasan perubahan ada di bab **Riwayat Revisi** |
| **Basis Teknis** | Seluruh klaim teknis bersumber dari pembacaan langsung export rule Pega: **2.634 berkas XML**, ditambah 55 `.prc` dan 8 `.fnc` |
| **Dokumen terkait** | `docs/ADR/` — 29 Architecture Decision Record · `docs/BRD/BRD.md` · `docs/ticketing/` |

— CONFIDENTIAL —

Dokumen ini adalah blueprint migrasi. Tidak berisi source code, API schema, atau database
migration script — hal tersebut disusun setelah Steering ini disetujui.

> **Dokumen ini dibangun otomatis** dari berkas sumber di `docs/Steering/` oleh
> `docs/tools/build-steering.js`. **Jangan disunting langsung** — suntingan akan hilang pada
> pembangunan berikutnya. Yang disunting adalah bab sumbernya.

---

## Daftar Isi

**Riwayat Revisi** — apa yang berubah sejak v1.0 dan atas dasar apa

1. [Pemahaman Bisnis (Business Understanding)](#1-pemahaman-bisnis-business-understanding)
2. [Arsitektur Saat Ini (As-Is)](#2-arsitektur-saat-ini-as-is)
3. [Arsitektur Masa Depan (To-Be)](#3-arsitektur-masa-depan-to-be)
4. [Domain Model dan Bounded Context](#4-domain-model-dan-bounded-context)
5. [Module Breakdown dan Urutan Migrasi](#5-module-breakdown-dan-urutan-migrasi)
6. [Strategi Migrasi](#6-strategi-migrasi)
7. [Strategi Teknis, Struktur Folder, dan Standar Coding](#7-strategi-teknis-struktur-folder-dan-standar-coding)
8. [Strategi Database](#8-strategi-database)
9. [Strategi API](#9-strategi-api)
10. [Keamanan, Autentikasi, dan Otorisasi](#10-keamanan-autentikasi-dan-otorisasi)
11. [Error Handling, Logging, Konfigurasi, dan Observability](#11-error-handling-logging-konfigurasi-dan-observability)
12. [Strategi Deployment dan Infrastruktur](#12-strategi-deployment-dan-infrastruktur)
13. [Strategi Testing](#13-strategi-testing)
14. [Non-Functional Requirements, Performa, Skalabilitas, dan Maintainability](#14-non-functional-requirements-performa-skalabilitas-dan-maintainability)
15. [Analisis Risiko](#15-analisis-risiko)
16. [Future Enhancement](#16-future-enhancement)

- [Lampiran A — Glossary Domain (Ubiquitous Language)](#lampiran-a-glossary-domain-ubiquitous-language)
- [Lampiran B — Decision Log](#lampiran-b-decision-log)
- [Lampiran C — Analisis Pemilihan Frontend](#lampiran-c-analisis-pemilihan-frontend)
- [Lampiran D — Catatan Penggunaan Matt Pocock Skills](#lampiran-d-catatan-penggunaan-matt-pocock-skills)
- [Lampiran E — Rincian Gap Export](#lampiran-e-rincian-gap-export)
- [Lampiran F — Rincian Penjenjangan Komite dan DB Link](#lampiran-f-rincian-penjenjangan-komite-dan-db-link)
- [Lampiran G — Inventaris 74 Harness](#lampiran-g-inventaris-74-harness)
- [Lampiran H — Status 40 Usulan Revisi](#lampiran-h-status-40-usulan-revisi)

---

## Riwayat Revisi — v1.0 → v2.0

Bab ini menjawab satu pertanyaan bagi siapa pun yang sudah pernah membaca atau menyetujui versi
sebelumnya: **apa yang berubah, dan atas dasar apa.**

| | |
|---|---|
| **Versi sebelumnya** | v1.0 — 2026-09-07, bersumber dari Decision Log `D-01`…`D-30` |
| **Versi ini** | **v2.0 — 2026-09-14**, bersumber dari `D-01`…`D-72` |
| **Dasar perubahan** | 41 keputusan baru (`D-31`…`D-72`) · 14 temuan verifikasi (`T-1`…`T-14`) · 40 kontradiksi (`K-1`…`K-40`) · 29 ADR |
| **Kewenangan menyunting** | `D-72` — larangan menyunting `STEERING.md` dan berkas `.docx` dicabut |

---

## 1. Tiga hal yang paling berubah

### 1.1 Export bertambah dua kali, dan sebagian penghalang terbesar hilang

Snapshot v1.0 memakai **2.167 rule**. Pada 2026-09-09 export bertambah dan **tiga folder yang
sebelumnya tidak ada** muncul — `Database/`, `Job Scheduler/`, `Agents/`, ditambah
`Service REST/`. Snapshot yang berlaku sekarang memuat **2.634 berkas XML**, ditambah **55 `.prc`
dan 8 `.fnc`**.

Akibatnya, empat penghalang lama tertutup — tetapi **tujuh penghalang baru muncul**, dan tiga di
antaranya memblokir gerbang kelulusan, bukan sekadar memperlambat pengerjaan. Rinciannya di §4.

### 1.2 Aturan komite ternyata dibaca terbalik

Versi v1.0 menyebut jumlah jenjang komite ditentukan **matriks nilai × jenis bisnis** yang memilih
satu baris. Kenyataannya **kumulatif**: setiap jenjang yang ambang bawahnya sudah terlampaui
**ikut menyetujui**.

Ini nyaris "diperbaiki" menjadi salah. Usulan mengganti penyaring menjadi rentang tertutup akan
mengembalikan tepat satu baris — **satu jenjang persetujuan berapa pun nilai klaim** — dan
menghapus penjenjangan yang menjadi inti `D-14`. Koreksinya diambil sebelum jawaban diterapkan
(`D-47`).

### 1.3 Otorisasi: satuan izin adalah menu, bukan aksi

Versi v1.0 menetapkan izin **berbutir aksi** (`klaim.akseptasi`, `komite.setujui`, …). `D-59`
memutuskan sebaliknya: **satuan izin adalah menu**, dan **tidak ada pemisahan tugas formal**.

Yang tetap berubah dari sistem lama adalah **tempat penegakannya** — dari penyembunyian menu di
antarmuka menjadi pemeriksaan di server pada setiap endpoint. Konsekuensinya diterima secara
sadar, dan menaikkan `S-5` Jejak Audit dari modul pendukung menjadi **satu-satunya kontrol
pengimbang yang tersisa**.

---

## 2. Angka yang dikoreksi

Seluruhnya terverifikasi langsung dari export. Angka lama tidak salah untuk lingkupnya; ia
**terlalu sempit** atau bersumber dari dokumen, bukan dari bukti.

| Hal | v1.0 | **v2.0** | Sumber |
|---|---|---|---|
| Snapshot export | 2.167 rule | **2.634 berkas XML** + 55 `.prc` + 8 `.fnc` | `D-45` |
| Jumlah modul | 32 | **33** — `S-8` Perkakas Uji Kesetaraan | `D-42` |
| Jumlah risiko | 12 | **19** — menyerap `R-13`…`R-15`, menambah `R-16`…`R-19` | `D-38` |
| Format nomor klaim | `PNCN-xxxx` | **`PNCN.YY.xxxx`** | `D-71` |
| Kelompok master data | 14 | **≥29** | `T-13` |
| Kolom grid inbox | 18–27 | **median 6** | `T-11` |
| Alamat email hardcode | 10 | **66 unik** | `D-15` terverifikasi |
| Operator ID hardcode | 4 | **24 unik** | idem |
| Ambang komite hardcode | 3 | **8 unik** + 7 ambang uang non-komite | idem |
| Hostname penentu perilaku | 3 | **3** — angkanya benar, **daftarnya salah** | idem |
| Domain kode Status Klaim | `1142`–`1151` (10 kode) | **`1134`–`1166` (33 kode)** | `R-06` tertutup |
| Gap export | puluhan rule | **±242 rule**, termasuk **137 When rule** | `R-16` |
| Ticket rule | "11 ticket" | **17 dirujuk · 8 ada · 9 tanpa rule** | koreksi `FR-W2` |
| Objek database diminta | 86 procedure | **64 objek** diminta, **62 diterima** | `D-45` |
| Perbaikan eksplisit `P-5` | 4 butir | **13 butir** | `D-49` |
| Batas hasil laporan | — | `pyMaxRecords=500` pada **54 dari 56** laporan | `T-12` |
| `OFFSET 500000` | dipakai sebagai premis | **tidak berdasar** — `OFFSET` nol kemunculan | `T-12` |

---

## 3. Perubahan per bab

| Bab sumber | Yang berubah |
|---|---|
| `01-FRONTEND-ANALYSIS.md` | Banner koreksi: "18–27 kolom" → **median 6**; pilihan React tidak berubah, tetapi **ukuran `U-2` turun** dan pilihan pustaka tabel menjadi terbuka. Rujukan `D-20` di kepala dokumen dikoreksi menjadi **`D-23`** — keputusan frontend, bukan dialek SQL |
| `02-BUSINESS-UNDERSTANDING.md` | Alur Komite ditulis ulang sebagai **kumulatif**; ambang komite dikoreksi menjadi **8 unik** |
| `03-CURRENT-ARCHITECTURE.md` | Snapshot export diperbarui; `R-02` dan `R-06` ditandai **tertutup**; format nomor klaim |
| `04-FUTURE-ARCHITECTURE.md` | Seam **Identity** naik derajat — kontrak HCC/HCQ belum ada; pernyataan "scheduler cukup di dalam aplikasi" diturunkan menjadi **belum diputuskan** |
| `05-DOMAIN-MODEL.md` | **Status Klaim 33 kode** · aggregate **Komite ditulis ulang kumulatif** · invarian baru **`I-11`…`I-13`** (kurs, presisi uang, soft delete) · `I-1` memakai toleransi numerik tegas · master **≥29 kelompok** |
| `06-MODULE-BREAKDOWN.md` | **`S-8` ditambahkan, total 33 modul** · `S-8` masuk gelombang 1 · ukuran `F-3`, `F-4`, `U-2`, `U-6` dikoreksi · **rantai kritis menjadi empat** · tabel modul terhalang diperbarui menyeluruh |
| `07-MIGRATION-STRATEGY.md` | `P-5` **13 butir** · Tahap 0 diperbarui dengan status nyata tiap permintaan · verifikasi kesetaraan diarahkan ke `S-8` |
| `08-TECHNICAL-STRATEGY.md` | **Kepemilikan transaksi pindah ke Go**; `B-4`/`B-9` dapat dibuat atomik; kontrak galat `ErrMsg` tidak dibawa |
| `09-DATABASE-STRATEGY.md` | Generator nomor klaim **`PNCN.YY.xxxx`** · **§8.1 soft delete menyeluruh** · retensi audit mengikuti retensi klaim · **§9.1 prosedur perubahan skema** dan **§9.2 penulis tunggal** · premis `OFFSET` dikoreksi |
| `10-API-STRATEGY.md` | **§8.5 permukaan REST masuk** — 4 layanan, dua di antaranya menerima persetujuan komite dari sistem lain · `GET_WORKING_HOURS` **ditulis ulang di Go**, bukan dipanggil lewat API · status kontrak HCC/HCQ |
| `11-SECURITY.md` | **Satuan izin menjadi menu** (`D-59`) · **22 peran** (`D-58`) · HCC/HCQ **nol jejak** · daftar §5 diperluas dan angkanya dikoreksi · **§6 baru: data nasabah di lingkungan non-produksi** |
| `12-CROSSCUTTING.md` | Master data diperluas dengan ukuran terverifikasi · **§3.4 larangan hostname** dirinci · **§3.5 nol Dynamic System Setting** · **§3.6 rahasia masih terbuka** |
| `13-DEPLOYMENT.md` | **Staging memakai data produksi apa adanya, tanpa penyamaran** (`D-64`) — mengoreksi kewajiban penyamaran pada v1.0 · **§9 baru: job terjadwal pada dua instans** |
| `14-TESTING-STRATEGY.md` | **§6.1–§6.4** perkakas `S-8`, kewenangan menyetujui selisih, selisih yang sudah dapat diperkirakan, dan **dua hal yang tidak boleh dibandingkan** · **§10 dua gerbang penerimaan**, modul tanpa baseline, pencabutan `BRD §21.4` |
| `15-NFR-PERFORMANCE-SCALABILITY.md` | Premis `OFFSET` dikoreksi · **kapasitas laporan tidak punya dasar historis** karena batas 500 baris |
| `16-RISK-ANALYSIS.md` | **19 risiko** · `R-02` dan `R-06` **tertutup** · `R-01` sebagian tertutup · `R-04` **turun** menjadi verifikasi · `R-13`…`R-19` ditambahkan · **ringkasan tindakan hari pertama diperbarui** menjadi 15 baris berstatus |
| `17-FUTURE-ENHANCEMENT.md` | Rujukan kode status dan format nomor klaim disesuaikan |
| `18-SKILLS-USAGE-LOG.md` | **Sesi 3 ditambahkan** — `grilling`, `domain-modeling`, beserta **empat kesalahan sendiri dan tiga koreksi atas laporan sub-agen** |
| `19-GAP-EXPORT-DETAIL.md` | Banner: dokumen ini **jauh dari lengkap** — tujuh tipe rule tidak diauditnya sama sekali; penggantinya adalah **export ulang berbasis Product rule** |
| `20-DETAIL-KOMITE-DBLINK.md` | **§1.8 baru** — jawaban final setelah master diterima: mekanisme kumulatif, `TYPE_KOMITE` sebagai pita **hanya di Non-MBU**, dan tabel dampak bila diberlakukan seragam |
| `CONTEXT.md` | 61 → **70 istilah**; tidak ada `[TERBUKA]` tersisa |
| `00-DECISION-LOG.md` | `D-31`…`D-72` ditambahkan sebagai **Sesi 3**. **Entri `D-01`…`D-30` tidak disunting sama sekali** |

---

## 4. Penghalang: empat tertutup, tujuh baru

**Tertutup atau turun derajat:**

| Penghalang | Status |
|---|---|
| `R-02` job terjadwal tidak diketahui | **tertutup** — 5 job + 1 agent (`D-57`) |
| `R-06` arti kode status | **tertutup** — 33 kode diterima |
| `D-14` isi master komite | **tertutup** — 21 kolom, 30 baris |
| `R-01` source procedure | **sebagian** — 62 dari 64 diterima |
| `R-04` router penugasan | **turun** menjadi verifikasi — algoritma beban terbaca dari kueri lain |

**Baru, dan tiga di antaranya memblokir gerbang kelulusan:**

| Penghalang | Menghalangi | Pemilik |
|---|---|---|
| **Kontrak API HCC/HCQ** — nol jejak di export | `F-3`, dan login seluruh aplikasi | Tim HCC/HCQ |
| **Daftar peristiwa wajib audit** | `S-5` | Compliance |
| **Pega staging yang dapat ditembak dari luar** | `S-8`, karenanya **gerbang 1 seluruh modul** | Tim Pega + Infra |
| **12 dependensi** yang dipanggil 62 procedure (`UPDATE_LOG_KONVERSI` 162×) | `B-1` dan modul nilai uang | DBA |
| **±242 rule hilang**, 137 di antaranya When rule | percabangan bisnis hampir semua modul | Tim Pega |
| **Tujuan penyimpanan rahasia** (`D-40` masih `OPEN`) | `F-4`, `F-5`, seluruh deployment | Tim Infra / Security |
| **8 Ticket rule custom hilang**; 14 dari 17 nama tanpa pemicu | `B-7`, `B-11`, `B-13`, `B-14` | Tim Pega |

---

## 5. Yang **tidak** berubah

Dicatat agar tidak perlu dibaca ulang:

| Hal | Keterangan |
|---|---|
| Pilihan teknologi | Go modular monolith · React + TypeScript + Vite · PostgreSQL 17+ sebagai target · Oracle 19c sementara |
| Strategi migrasi | Strangler Fig, modul dialihkan bertahap, database bersama selama masa paralel |
| Batas bounded context | Data polis milik GISFW; Claim PNC menyimpan snapshot |
| Empat konsep status | **tetap empat** — pemilik bisnis menegaskan keempatnya memang berbeda; usulan menggabungkan **ditarik** |
| Worklist dan Workbasket | tetap dua model penugasan |
| Jadwal | **tidak berubah** (`D-61`) — selisih antara jadwal dan kesiapan menjadi tanggung jawab manajemen |
| Entri `D-01`…`D-30` | **tidak disunting sama sekali**; perubahan pikiran ditulis sebagai keputusan baru yang menyebut entri yang disupersede |

---

## 6. Cara dokumen ini dibangun

Sejak v2.0, **`STEERING.md` tidak disunting langsung**. Ia dibangun dari berkas sumber di
`docs/Steering/` oleh `docs/tools/build-steering.js`, sehingga tidak ada lagi dua versi pernyataan
yang sama yang bisa berbeda (`D-72`, Q35 Opsi 1).

**Suntingan manual pada `STEERING.md` akan hilang pada pembangunan berikutnya.** Yang disunting
adalah bab sumbernya.

Satu akibat langsung yang menutup keusangan terbesar: **Lampiran B kini memuat `D-01`…`D-73`**,
bukan berhenti di `D-30` seperti sebelumnya.

Berkas `.docx` lama disimpan sebagai arsip bertanda versi dan **tidak dihapus** (`D-72`, Q36
Opsi 3).

---

## 7. Koreksi angka setelah v2.0 — disetujui 2026-09-14

Ketiga angka berikut ditemukan keliru saat seluruh modul ditiketkan (`D-73`), dan **disetujui Work
Owner untuk diterapkan** ke Steering dan BRD.

| Angka | Tertulis sebelumnya | Terverifikasi | Cara memastikannya |
|---|---|---|---|
| Pemanggil `SendEmailNotification` | **17** | **15** | `19-GAP-EXPORT-DETAIL.md:196` menyebut kelima belas pemanggilnya satu per satu |
| Lokasi 3 password SMTP | **44** | **31** | `D-40` · `16-RISK-ANALYSIS.md:484` |
| Connect REST keluar | **12** | **21** | direktori `Connect REST/` dihitung langsung — 12 yang terhitung sejak awal ditambah **9 yang baru ditemukan** |

**Berkas yang disunting:** `06-MODULE-BREAKDOWN.md` (baris modul `S-4` dan baris risikonya) ·
`16-RISK-ANALYSIS.md` (tabel ukuran) · `BRD.md` (ringkasan ukuran, permukaan yang harus dibangun,
`FR-S4`, dan dasar perkiraan `§20.1`) · `verifikasi-bukti-adr.md` (dua baris `S-3`, satu baris
`S-4`) · `migration-readiness.md` · `requirement-summary.md`.

**Dua tempat sengaja TIDAK disunting**, karena keduanya adalah rekaman, bukan pernyataan fakta
yang berlaku:

| Tempat | Alasan |
|---|---|
| `00-DECISION-LOG.md:729` | Merekam **keberatan yang benar-benar diajukan** pada saat itu, beserta angka yang dipakai saat itu. Mengubahnya berarti memalsukan catatan rapat |
| `00-DECISION-LOG.md` tabel koreksi `D-73` | Tabel itu **justru berisi pasangan angka lama → angka baru**. Menggantinya akan menghapus jejak koreksinya sendiri |

**Yang berubah bukan hanya angka.** Koreksi ketiga **menambah lingkup `S-4`**: dari 12 integrasi
keluar menjadi 21, ditambah 4 layanan masuk yang belum pernah dihitung. Sembilan yang baru
ditemukan itu **belum dianalisis setara dengan 12 yang lama**, dan seluruhnya
ber-`pyUseAuthentication=false`. Penambahan ini tercermin di `BRD §20.1` sebagai dasar perkiraan,
bukan disembunyikan sebagai detail teknis.

---

# 1. Pemahaman Bisnis (Business Understanding)

Seluruh isi dokumen ini diturunkan langsung dari source aplikasi Pega (902 activity, 652 rule
SQL, 70 when rule, 4 flow), dikonfirmasi oleh pemilik bisnis pada 2026-09-07. Istilah mengikuti
`CONTEXT.md`.

---

## 1. Aplikasi ini menangani apa

**Claim PNC** adalah sistem penanganan klaim asuransi umum untuk lini bisnis **Non-Motor
(Non-MBU)** di Asuransi Sinar Mas. Perjalanannya dari laporan kerugian oleh tertanggung sampai
pembayaran ganti rugi dan pemberitahuan ke koasuransi/reasuransi.

Lini bisnis yang ditangani, dikenali lewat **Group Panel**:

| Group Panel | Lini | Karakter penanganan |
|---|---|---|
| `002` | Personal Accident (PA) | Perlu penilaian medis; ada peran Analyst Doctor dan RCL Dokter |
| `003` · `009` | Aneka | Paling beragam; Fac Out sering dipakai |
| `004` | Marine Cargo | Terkait pengangkutan, rute, dan kemasan |
| `005` | Travel | Batas waktu lapor paling longgar (90 hari) |
| `006` | Fire / Property | Terkait lokasi risiko dan okupasi bangunan |

Lini khusus lain yang muncul di kode: **TKA** (Tenaga Kerja Asing), **SPK** (Sinarmas Penjaminan
Kredit / Asuransi Kredit — wajib Nomor SLIK), **Bonding**, **HE** (Heavy Equipment), dan
**Contractors PM**.

---

## 2. Proses bisnis inti

### 2.1 Alur utama — Register Flow

Diturunkan dari `Register_Flow` (23 shape, 30 konektor, 13 assignment, 7 decision).

```
                          [Mulai]
                             │
                      ┌──────▼──────┐
                      │  View Polis │  ambil & kunci snapshot polis
                      └──────┬──────┘
                             │
                    ┌────────▼────────┐
                    │ Input Register  │  ← gerbang validasi terberat
                    └────────┬────────┘
                             │
                        ◇ Kembali? ──ya──► kembali ke View Polis
                             │tidak
                        ◇ Apakah PA?
                   ya ───────┴─────── tidak
                    │                   │
             ┌──────▼──────┐       ◇ Apakah Travel?
             │ Estimation  │      ya ───┴─── tidak
             │    (PA)     │       │           │
             └──────┬──────┘  ┌────▼────┐ ┌────▼─────┐
                    │         │ Input   │ │ Input    │
                ◇ Kembali?    │Estimasi │ │Estimasi  │
                    │         │(Travel) │ │(Non-MBU) │
             ┌──────▼──────┐  └────┬────┘ └────┬─────┘
             │ Investigator│       │           │
             │ (workbasket)│   ◇ Kembali?  ◇ Kembali?
             └──────┬──────┘       │           │
                    │        ┌─────▼─────┐┌────▼──────────┐
                    │        │Send To PIC││Choose Surveyor│
                    │        │  Teknik   ││               │
                    │        └─────┬─────┘└────┬──────────┘
                    │              │           │
             ┌──────▼──────────────▼───┐       └──► [Selesai]
             │     Send To Analis      │
             └────────────┬────────────┘
                          │
        ┌─────────────────▼──────────────────┐
        │  Percabangan: Compliance / PUCL /  │
        │  Analyst Doctor / RCL Dokter       │
        └──┬──────┬──────────┬───────────┬───┘
           │      │          │           │
    Compliance  RCL/PUCL  Analyst    RCL Dokter
    (basket)    (basket)  Doctor
           │      │          │           │
           └──────┴──────────┴───────────┴──► [Selesai]
```

**Delapan ticket** memungkinkan lompatan langsung ke tahap tertentu tanpa melewati urutan di
atas: `setToRegister_ticket`, `SendToEstAdmin`, `SendToEstTravel`, `SendToEstimatorPA`,
`SendToPICTravel`, `SendtoAnalysator`, `SendToInvestigator`, `CompliancePNC`, `AcceptanceKomite`,
`RCLDokter`, `SendtoPUCL`.

> **Konsekuensi penting untuk desain:** alur nyata **bukan rangkaian linear**. Setiap tahap bisa
> dimasuki dari luar. Model status baru harus mengizinkan transisi lateral ini, bukan memaksakan
> urutan kaku yang akan langsung ditolak user.

### 2.2 Tiga proses pendukung

**Open Protection** (`CreateProtection_Flow`)
`Input Protection → Akseptasi Protection → ◇ Diterima? → Selesai / Ditolak`
Permintaan pembukaan proteksi sebelum atau di luar alur klaim normal. Satu Open Protection
dapat ditautkan ke klaim dan ditandai terpakai (`IsUsedPNC = "1"`) saat registrasi.

**Receive Document** (`ReceiveDocument_Flow`)
`Input Receive Document → Selesai`
Pencatatan penerimaan dokumen fisik, termasuk pengiriman antar cabang (ekspedisi, nomor resi,
estimasi tiba). Menghasilkan `RCV_ID` yang ditautkan ke klaim.

**Komite** (`Komite_Flow`)
`Komite Router → Lihat Detail Transfer → ◇ Masih ada level berikutnya? → ulang / Selesai`
Persetujuan berjenjang. Setiap putaran menaikkan `KomiteCount` dan mereset `AcceptStatus`, lalu
meneruskan ke penyetuju berikutnya.

> **Koreksi v2.0 — jumlah jenjang tidak dipilih dari matriks, melainkan diakumulasi.** Jumlah
> jenjang **sama dengan jumlah baris master yang ambang bawahnya sudah terlampaui nilai klaim**
> (`D-47`): setiap jenjang yang terlampaui **ikut menyetujui**, bukan memilih satu jenjang tunggal.
> Klaim kecil melewati sedikit jenjang, klaim besar melewati banyak.
>
> Batas **4 level** adalah akibat isi master hari ini, **bukan aturan** — secara mekanisme jumlah
> jenjang mengikuti jumlah baris. Untuk lini **Non-MBU** saja, akumulasi didahului pemilihan
> **pita nilai** (sampai Rp 100.000.000 → pita `1`; di atasnya → pita `2`); lini lain tidak punya
> langkah pendahuluan itu (`D-52`, `D-70`). Rinciannya di `20-DETAIL-KOMITE-DBLINK.md` §1.8.

---

## 3. Aturan bisnis yang berhasil diekstraksi

Diambil terutama dari `InputRegister_act` (137 step) dan 70 when rule. Ini bukan daftar lengkap,
tapi ini aturan yang **paling sering menolak input user** sehingga wajib benar di sistem baru.

### 3.1 Aturan tanggal

| Aturan | Berlaku untuk |
|---|---|
| Tanggal Kejadian (DOL) harus di dalam periode polis | Semua |
| DOL boleh sampai 30 hari setelah polis berakhir | Bonding |
| Tanggal cetak boleh sampai 90 hari setelah polis berakhir | Travel & PA |
| Tanggal Lapor ≤ DOL + 7 hari | Semua **kecuali** PA |
| Tanggal Terima Dokumen ≤ DOL + 90 hari | Travel |
| DOL ≤ Tanggal Lapor | Semua (sama hari diperbolehkan) |
| Tanggal Lapor ≤ Tanggal Terima Dokumen | Semua |
| Tidak boleh melebihi tanggal hari ini | DOL, Tanggal Lapor, Tanggal Terima Dokumen |

### 3.2 Aturan duplikasi

- Klaim ditolak bila sudah ada klaim lain dengan **Nomor Polis + Objek + Lokasi** yang sama.
- Untuk **PA**, pengecekan diperketat: **Nomor Polis + Objek + Penyebab Kerugian `12002` + Lokasi**.
- Pesan yang dikembalikan menyertakan nomor klaim yang sudah ada.

### 3.3 Aturan reasuransi

- **Total spreading wajib 100%**, **toleransi 4 desimal, `99,9999`–`100,0001`** (`D-51`). Bila tidak, submit ditolak. Menggantikan pencocokan substring lama yang meloloskan `199.99` dan `1100.0`.
- Bila ada spreading **Fac Out** (`TreatyType = 10015`), data Fac Offer wajib ada.
- Untuk **Group Panel `003`**, Fac Offer wajib menyertakan Object Name.
- Bila klaim ditandai **Ex-Gratia**, treaty `OR` (`10001`) otomatis berubah menjadi `ORS` (`10007`).
- Spreading yang bertanda `FlagDelete = "1"` dibuang sebelum perhitungan.

### 3.4 Aturan nilai dan notifikasi

- Nilai estimasi dikonversi ke IDR memakai **kurs standar** sebelum dibandingkan.
- Estimasi **> Rp 1.000.000.000** memicu **Notice of Large Losses** ke Underwriting sesuai Group
  Panel dan ke jajaran pimpinan.
- Persetujuan komite dipicu ambang **Rp 50.000.000** (umum) dan **Rp 30.000.000** (PA/Travel).
  Verifikasi menemukan **8 ambang komite unik**, bukan dua — termasuk `3.500` yang dipicu
  **hostname entitas Timor-Leste** dan `7.000`, `20.000.000`, `100.000.000` pada jalur lain. Satu
  ambang yang sama (`50.000.000`) bahkan dibandingkan dengan **tiga operator berbeda** di tiga
  rule; itu masuk daftar perbaikan eksplisit `P-5` (`D-49` #2).
- Bila premi belum lunas (`AgingAmount > 1`), notifikasi premi tertunggak dikirim.

### 3.5 Aturan kelengkapan

- **Penyebab Kerugian wajib diisi**, kecuali lini Travel.
- **Nomor SLIK wajib diisi** untuk SPK / Asuransi Kredit.
- Objek tanpa Coverage dibuang otomatis dari daftar.
- Bila hubungan tertanggung dipilih "lain-lain" (`7`), keterangannya wajib diisi.
- Polis Deklarasi tidak dapat diklaim, kecuali lini Aneka.
- `ObjectName` dipotong pada 3.800 karakter karena batas kolom database.

---

## 4. Peran pengguna

22 access group ditemukan di when rule, mengendalikan **47 harness target unik** lewat **51 item menu aksi**; **11 dari 47 harness tidak ada di export** (`K-33`).

| Peran | Tanggung jawab utama |
|---|---|
| **PncAdmin** | Registrasi klaim, input data, unggah dokumen |
| **PncManagerAdmin** | Persetujuan tingkat admin, pemantauan |
| **PncPICTeknik** | Penanggung jawab teknis sesuai lini bisnis |
| **PNCKomiteTeknik** · **PNCKomite** | Persetujuan nilai klaim berjenjang |
| **CaseManager** | Pengawasan lintas kasus |
| **PncRCLPUCL** | Penanganan penolakan dan proses ulang klaim |
| **PncAnalystDoctor** | Penilaian medis klaim PA |
| **PncComplience** | Pemeriksaan kepatuhan |
| **PncInvestigator** | Penyelidikan klaim mencurigakan |
| **PNCSurveyor** | Survei lapangan, unggah foto dan hasil survei |
| **PncPLADLA** | Pengelolaan pemberitahuan ke koasuransi/reasuransi |
| **PncReceive** · **PncManagerReceive** | Penerimaan dokumen fisik |
| **PncCollection** · **PncOPCGeneral** | Open Protection |
| **PNCServiceCenter** | Layanan pelanggan |
| **TreatyIn** | Klaim treaty masuk |
| **ViewClaimPNC** | Akses baca saja |
| **PNCReportClaimInternal** · **PNCReportClaimEksternal** | Akses laporan; eksternal terbatas |
| **Administrators** | Akses penuh |

---

## 5. Aliran nilai uang klaim

Ini tulang punggung bisnisnya, dan urutan inilah yang menentukan model data settlement.

```
Estimasi Klaim                    saat registrasi
      │                           dipakai untuk PLA & ambang Large Losses
      ▼
Usulan Nilai (Propose)            hasil survei / penilaian adjuster
      │                           dipakai untuk Pre-DLA
      ▼
Persetujuan Komite                berjenjang 1–4 level bila melewati ambang
      │
      ▼
Akseptasi (Accepted)              terbit Nomor Akseptasi
      │                           dipakai untuk DLA & LOD ke tertanggung
      ▼
Transfer ke Kasir                 TransferCashierStatus
      │
      ▼
Pembayaran                        ClaimPaidStatus
```

Di setiap tahap, nilai dikurangi **Salvage** dan **Recovery** bila ada, lalu dibagi ke para
penanggung sesuai **Spreading** dan **Koasuransi**.

---

## 6. Integrasi dengan dunia luar

| Sistem | Arah | Keperluan |
|---|---|---|
| **BRI Surf** (`partner.api.bri.co.id`) | Keluar | OAuth token + kirim umpan balik klaim |
| **Arsip Dokumen** (`app8/asm-archive`) | Keluar | Injeksi data arsip dokumen klaim |
| **Storage Dokumen** (`app13/api`) | Dua arah | Unggah dan ambil dokumen klaim |
| **Konversi Gambar** (`aiimage`) | Keluar | Konversi format AVIF |
| **History Payment** (WebLogic internal) | Masuk | Riwayat pembayaran produksi |
| **HCC / HCQ** | Masuk | Autentikasi user (D-07) |
| **@ASMD** dan 5 database lain | Masuk | HRD, GL payment, master sales, polis, jam kerja (D-25) |
| **Kasir** | Keluar | Data rekening dan permintaan pembayaran |
| **SLIK OJK** | Keluar | Pelaporan regulator untuk lini SPK |
| **Email SMTP** | Keluar | 5 correspondence: notifikasi register, large losses, VA, laporan klaim, error produksi |

---

## 7. Karakter beban kerja

| Aspek | Angka | Sumber |
|---|---|---|
| User aktif harian | 200–300 | Pemilik project |
| Klaim baru | Ribuan per bulan | D-10 |
| Data historis | Puluhan juta baris | D-10 |
| Ketersediaan | 24/7 | D-27 |
| Layar | 74 harness, 269 section | Source |
| Laporan | 56 report definition | Source |

**Profil bebannya: data besar, konkurensi rendah.** 200–300 user bersamaan bukan beban berat
untuk Go. Yang berat adalah **query terhadap puluhan juta baris**, terutama pada inbox
berkolom banyak dan laporan lintas periode. Optimasi harus diarahkan ke sana, bukan ke jumlah
request per detik.

# 2. Arsitektur Saat Ini (As-Is)

Peta arsitektur sistem yang ada sekarang, beserta utang teknis yang harus dijawab oleh desain
baru. Seluruh angka diukur langsung dari export rule XML Pega. **Snapshot v1.0 memakai 2.167
rule; snapshot berlaku sejak 2026-09-14 memuat 2.634 berkas XML**, ditambah 55 `.prc` dan 8
`.fnc` di folder `Database/` yang baru diterima.

---

## 1. Gambaran umum

```
┌──────────────────────────────────────────────────────────────────┐
│                      Browser (Portal Pega)                       │
│         74 Harness · 269 Section · Navigasi 47 menu              │
└────────────────────────────┬─────────────────────────────────────┘
                             │ HTTP (server-rendered)
┌────────────────────────────▼─────────────────────────────────────┐
│                  Pega PRPC 8.3 (JBoss / WebLogic)                │
│                                                                  │
│  Case Types:  Work-PNC · Work-Komite · Work-OpenProtection       │
│               Work-ReceiveDocument                               │
│                                                                  │
│  Rule:  902 Activity (15.063 step) · 80 Data Transform           │
│         70 When · 29 Flow Action · 56 Report Definition          │
│         652 Connect-SQL · 21 Connect-REST · 7 Data Page          │
│                                                                  │
│  Framework: GCNMFW (Claim) ── di atas ── GISFW (Policy/UW)       │
└──────┬────────────────────────────────────────────┬──────────────┘
       │ JDBC                                       │ HTTPS
┌──────▼──────────────────────────────┐   ┌─────────▼──────────────┐
│         Oracle Database              │   │  Sistem Eksternal      │
│                                      │   │                        │
│  DATAPEGA  — tabel engine Pega       │   │  BRI Surf              │
│  POOLDATA  — data bisnis inti        │   │  Arsip (app8)          │
│  GENERAL   — storage & token         │   │  Storage (app13)       │
│  GL · COLLECTION · MBU · ANEKA       │   │  AI Image              │
│                                      │   │  History Payment       │
│  245 tabel · 70 proc/function        │   │  SMTP                  │
└──────┬───────────────────────────────┘   └────────────────────────┘
       │ DB Link (64 pemakaian)
┌──────▼───────────────────────────────────────────────────────────┐
│  @ASMD (55×) · @SIMASNET · @SMI · @OPJAVA · @PROD_ASM · @PROD_TKA │
│  HRD · GL Payment · Master Sales · Polis · Jam Kerja · Treaty     │
└──────────────────────────────────────────────────────────────────┘
```

---

## 2. Lapisan aplikasi

### 2.1 Lapisan presentasi
- **74 Harness** = layar utuh (inbox, form, laporan, dashboard).
- **269 Section** = komponen UI yang dirakit ke dalam harness. **268 di antaranya memakai
  repeat/grid.**
- **29 Flow Action** = aksi yang bisa dijalankan pengguna pada satu tahap alur, masing-masing
  memetakan ke satu Section plus pre/post processing.
- Semua dirender di **sisi server** oleh Pega. Tidak ada API terpisah; UI dan logika menyatu.

### 2.2 Lapisan proses
- **4 Flow** mendefinisikan alur kerja per case type.
- **Assignment** menempatkan tugas ke **Worklist** (per orang) atau **Workbasket** (antrean bersama).
- **Router** menentukan penerima tugas: `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter`,
  `KomiteRouter`, `PNCAdminRouterRCV`, `ToCurrentOperator`, `ToWorkList`, `ToWorkbasket`.
- **Ticket** memungkinkan lompatan ke tahap mana pun, di luar urutan alur.

### 2.3 Lapisan logika
- **902 Activity** dengan **15.063 step**, rata-rata 17,9 step per activity.
- Distribusi ruleset: **GCNMFW 633** · **GISFW 141** · Pega OOTB 124 · GKM 4.
- Metode yang paling sering dipakai: `Property-Set` (6.953), `RDB-List` (1.144), `Page-New` (938),
  `Page-Copy` (662), `Java` (253 + 69), `Obj-Save` (109), `Commit` (103), `Connect-REST` (56).
- **335 step berisi Java mentah** (326 KB) — logika yang menembus abstraksi Pega.

### 2.4 Lapisan data
- **652 Connect-SQL** berisi **534 KB SQL mentah**: 636 browse, 43 save, 16 delete, 2 open.
- **7 Data Page**, seluruhnya `refresh=never` — **praktis tidak ada lapisan caching**.
- **56 Report Definition** untuk inbox dan laporan.

---

## 3. Struktur database

### 3.1 Pembagian schema

| Schema | Peran | Contoh tabel |
|---|---|---|
| `POOLDATA` | Data bisnis inti | `T_CLAIM_PNC` (124×), `GCNM_PROGRESS_CLAIM` (119×), `T_CLAIM_ADJUSTMENT` (92×), `M_KOMUNIKASI_PNC` (46×), `T_CLAIM_KOMITE_LIST` (33×), `JSON_KLAIM`, `JSON_POLIS` |
| `DATAPEGA` | Tabel engine Pega | `PC_ASM_FW_GCNMFW_WORK` (116×), `PC_ASSIGN_WORKLIST` (18×), `PR_OPERATORS`, `PC_LINK_ATTACHMENT` |
| `GENERAL` | Storage & token | `T_STORAGE_IMAGE`, `MST_BUKA_PROTEKSI` |
| `GL` · `COLLECTION` · `MBU` · `ANEKA` · `HRDASM` | Domain lain | diakses lewat DB Link |

**245 tabel berbeda** teridentifikasi dari SQL.

### 3.2 Dua model penyimpanan yang hidup berdampingan

Yang membingungkan: **data klaim yang sama disimpan dua kali** dengan bentuk berbeda.

1. **Tabel relasional `POOLDATA`** — `T_CLAIM_PNC`, `T_CLAIM_OBJECTLIST`,
   `T_CLAIM_OBJECTCOVERAGE`, `T_CLAIM_ADJUSTMENT`, dan seterusnya.
2. **Dokumen JSON** — `JSON_KLAIM.DATA_JSONBLOB` dan `JSON_POLIS.DATA_JSONBLOB`, dibaca dengan
   `JSON_TABLE` / `JSON_VALUE` (222 pemakaian di 29 rule).

Sinkronisasi antara keduanya dilakukan **stored procedure** (`PEGA_JSON_KLAIM_PNC`,
`PEGA_CONVERT_JSONKLAIM_PNC`, `CONVERTJSONPRODUCTION`). Bila prosedur gagal atau tidak dipanggil,
kedua representasi menjadi tidak konsisten — dan tidak ada mekanisme yang mendeteksinya.

### 3.3 Konstruksi Oracle yang dipakai

| Konstruksi | Volume |
|---|---|
| `TO_CHAR` | 411× di 79 rule |
| `CASE WHEN` | 317× |
| `JSON_VALUE` / `JSON_QUERY` | 195× di 15 rule |
| `TRUNC` | 150× |
| `NVL` | 100× |
| `SYSDATE` | 69× |
| `ROWNUM` | 68× di 44 rule |
| DB Link `@` | 64× di 27 rule |
| Blok PL/SQL `BEGIN…END` | 44 rule |
| `JSON_TABLE` | 27× di 14 rule |
| `LISTAGG` · `(+)` · `CONNECT BY` · `KEEP DENSE_RANK` | sedikit tapi ada |

### 3.4 Stored procedure dan function database

Logika bisnis nyata hidup di dalam database. Yang teridentifikasi antara lain:
`PEGA_JSON_KLAIM_PNC`, `INSERT_PLADLA`, `INSERT_SURVEYORLIST`, `INSERT_KPIADJUSTER`,
`INSERT_SALVAGE_DETAILS`, `CONVERTJSONPRODUCTION`, `INSERTDATAAIKLAIMPNC`,
`INSERTDATAKOMITELIST`, `ADD_NEWMASTERVIRTUALACCOUNT`, `PNC_INSERT_EMAIL_ADJUSTER`,
`GENERAL.GET_TOKEN_STORAGE`, `COLLECTION.P_GET_DATA_REFUND`, `GET_POSISI_PROGRESS_PNC`,
`BASE64ENCODE`, dan puluhan lainnya.

> **Source procedure ini tidak ada di export XML.** Isinya belum pernah dilihat. → **R-01**

---

## 4. Utang teknis yang harus dijawab desain baru

### 4.1 Kunci teknis Pega bocor ke data bisnis

```sql
CLAIMID = 'ASM-FW-GCNMFW-WORK ' || {no_klaim}
```

Nama kelas internal Pega tertanam sebagai bagian dari primary key di tabel bisnis. Akibatnya
data bisnis tidak bisa dibaca tanpa mengetahui konvensi internal Pega, dan sistem apa pun yang
menggantikan Pega tetap terikat pada nama itu.
→ Dijawab oleh **D-22** dan **D-71** (format `PNCN.YY.xxxx` tanpa prefix Pega).

### 4.2 Alias kolom yang menyesatkan

Contoh nyata dari `BroswseKlaimByRegisterDate-SQL`:

```sql
a.LOCATION      AS "RISKLOCATION"
b.NOPOLIS       AS "NoKTP"
a.picteknik     AS "UserAdmin"
a.ttlos         AS "CaseID"
a.BUSINESSNAME  AS "NOPOLIS"      -- nama bisnis dialiaskan jadi "nomor polis"
a.leader_member AS "BUSINESSTYPE"
```

Penyebabnya: developer memaksa nama kolom agar cocok dengan property clipboard Pega yang sudah
ada, alih-alih membuat property baru. Akibatnya **nama tidak lagi mencerminkan isi**, dan
membaca query berarti menebak.
→ Dijawab oleh **D-19** (penamaan ulang menyeluruh mengikuti `CONTEXT.md`).

### 4.3 Nilai bisnis di-hardcode di dalam logika

Ditemukan di `InputRegister_act` dan `GetKomiteApproval`:

Angka di bawah adalah hasil verifikasi terhadap **seluruh export**, bukan terhadap dua rule saja
seperti pada v1.0. Nilai sensitif **tidak direproduksi** sesuai aturan penulisan `D-69`; lokasinya
dirujuk dengan `berkas:baris` pada `docs/verifikasi-bukti-adr.md` §7.

| Jenis | Jumlah terverifikasi | Catatan |
|---|---|---|
| **Alamat email** | **66 unik** | termasuk **≥6 akun Gmail pribadi di jalur produksi** dan **5 alamat yang dipakai sebagai Operator ID** di filter laporan KPI |
| **Operator ID** | **24 unik** | antara lain `MORASOTARDODOTARIGAN`, `ELLENSUPRIYATI`, `IRMANOPITAPURBA_1`, `RATNAGUSNITASARI`; sebagian tertanam **di dalam teks SQL** |
| **Ambang komite** | **8 unik** | antara lain `50000000`, `30000000`, `20000000`, `7000`, `3500`; ditambah 7 ambang uang non-komite |
| **Hostname penentu perilaku** | **3 unik, 48 perbandingan** | host **dev** (34), host **entitas Timor-Leste** (1), host **entitas Insurtech** (13) |

> **Koreksi daftar hostname.** Versi v1.0 menyebut host produksi utama sebagai salah satu dari
> tiga. Itu **tidak akurat** — host itu hanya muncul sebagai **konstanta URL**, tidak pernah
> dibandingkan. Hostname ketiga yang benar-benar menentukan perilaku adalah host **entitas
> Insurtech**. Angka 3 kebetulan tetap benar; daftarnya tidak.

**Yang paling serius:** `Activity/InputRegister_act-Act.xml` memuat **empat step** bertanda
`// TESTING` (`:16693`, `:16830`, `:16998`, `:17141`) yang **menimpa email Underwriting dan
pimpinan dengan alamat penguji** — salah satunya alamat Gmail pribadi — lalu tetap berada di jalur
produksi.

Teori bahwa blok itu nonaktif karena bertanda `pyStepsBlockName=//` sudah diuji dan **gugur**:
`//` dipakai **952× di 279 activity** berdampingan dengan label bermakna (`ERR`, `EXT`, `END`,
`SKIP`), dan format export ini **tidak memiliki elemen aktif/nonaktif sama sekali**. Perbandingan
langsung membuktikan step TESTING punya **precondition identik** dengan step produksinya dan
berada **sesudahnya**.

→ Dijawab oleh **D-15** (semua jadi konfigurasi; blok TESTING tidak dibawa) dan **`ADR-0025`**,
yang masih `Proposed` karena tujuan penyimpanan rahasia belum ditetapkan (`D-40`).

### 4.4 Zona waktu ditangani manual

Tanggal disimpan dalam GMT, lalu **+7 jam ditambahkan secara manual** di setiap tempat yang
membutuhkannya:

```
@DateTime.addCalendar(.ClaimData.DateOfLoss, 0,0,0,0, 7, 0,0)
```

Ada activity khusus bernama `Set7Hours` untuk keperluan ini. Bila satu tempat lupa memanggilnya,
tanggal bergeser 7 jam tanpa ada yang menyadari — dan pada aturan seperti "Tanggal Lapor ≤ DOL
+ 7 hari", pergeseran ini mengubah hasil validasi.

### 4.5 SQL dirangkai dari string

Pola `{ASIS:...}` menyisipkan nilai **langsung ke dalam teks SQL tanpa parameter binding**:

```sql
WHERE claimid = ... {ASIS:InputData.CARI4}
AND b.pxassignedoperatorid IN {ASIS:TempOperator.CityID}
```

Bahkan potongan klausa SQL disimpan sebagai nilai property:

```
.StatusClaim := "and a.STSTRANSFER ='3'"
.StatusClaim := "and trunc(a.TGLAKSEPTASI)>=to_date('...','dd/mm/yyyy') ..."
```

Ini **risiko SQL injection sekaligus penghalang portabilitas**. → Dijawab di Coding Standards:
seluruh query memakai parameter binding, tanpa perkecualian.

### 4.6 Duplikasi masif per lini bisnis

Pola `Browse*`, `Insert*`, `Grouping*`, `CreateCasePNC*`, `GetObjectFromTable*` digandakan untuk
tiap lini bisnis: `_AsuransiKredit`, `_AutoClaim`, `_Travel`, `_Kredit_PA`. Satu perubahan aturan
harus diterapkan di empat tempat — dan sering hanya diterapkan di sebagian.

### 4.7 Nama rule mengandung salah ketik yang dipertahankan

`Broswse*` (11 rule, seharusnya `Browse`), `Complience` (seharusnya `Compliance`),
`Proccedure`, `SALAVAGEDOCUMENT`, `CATRGORY`. Ini menyulitkan pencarian dan menandakan tidak
adanya proses review penamaan.

### 4.8 Sistem masih aktif berubah

Distribusi tahun perubahan terakhir activity:

| Tahun | 2016 | 2017 | 2018 | 2019 | 2020 | 2021 | 2022 | 2023 | 2024 | 2025 | 2026 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| Jumlah | 2 | 35 | 127 | 43 | 108 | 58 | 58 | 97 | **180** | 70 | **124** |

**124 activity berubah pada 2026** — sistem sumber adalah sasaran bergerak. Strategi Strangler
Fig (D-05) harus memperhitungkan bahwa Pega akan terus berubah selama migrasi berjalan.

---

## 5. Yang tidak ada di export dan harus dilengkapi

| Yang hilang | Dampak | Risiko |
|---|---|---|
| Source 64 procedure & function database | Logika bisnis tidak diketahui | **R-01** |
| ~~Rule Agent / Queue Processor~~ | ✅ **diterima 2026-09-09** — 5 job + 1 agent (`D-57`) | ~~R-02~~ tertutup |
| ~~Isi master `V_STS_CLAIM`~~ | ✅ **diterima** — **33 kode `1134`–`1166`**, bukan 10 kode | ~~R-06~~ tertutup |
| `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter` | Aturan penugasan tidak diketahui | **R-04** |
| 50 activity yang dipanggil tapi tidak diekspor | Termasuk `SendEmailNotification` (dipanggil 15×), `generatePDF`, `SetTicket` | **R-07** |
| DDL tabel (tipe kolom, index, constraint) | Desain skema harus menebak | **R-08** |

# 3. Arsitektur Masa Depan (To-Be)

Arsitektur sasaran, batas modul, dan seam. Dirancang memakai kosakata dari skill
`mattpocock-skills:codebase-design`: **module** (punya interface dan implementation),
**interface** (segala yang harus diketahui pemanggil), **depth** (banyak perilaku di balik
interface kecil), **seam** (tempat perilaku bisa diganti tanpa mengedit di tempat itu), dan
**adapter** (yang mengisi seam).

Prinsip yang dipegang: **satu adapter berarti seam hipotetis; dua adapter berarti seam nyata.**
Kita hanya membuat seam bila ada yang benar-benar bervariasi di sana.

---

## 1. Gambaran umum

```
┌───────────────────────────────────────────────────────────────────────┐
│  Browser — React 18 + TypeScript + Vite (SPA, berkas statis)          │
│  TanStack Table / AG Grid · TanStack Query · React Router             │
└───────────────────────────────┬───────────────────────────────────────┘
                                │ HTTPS · JSON · Bearer token
┌───────────────────────────────▼───────────────────────────────────────┐
│                     Load Balancer (2+ instance, D-27)                  │
└───────────────────────────────┬───────────────────────────────────────┘
                                │
┌───────────────────────────────▼───────────────────────────────────────┐
│  Aplikasi Go (satu binary — menyajikan API dan berkas statis SPA)     │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  Transport      HTTP handler · routing · autentikasi         │    │
│  │                 validasi request · serialisasi response       │    │
│  └──────────────────────────┬───────────────────────────────────┘    │
│  ┌──────────────────────────▼───────────────────────────────────┐    │
│  │  Domain         Modul bisnis — aturan, invarian, alur        │    │
│  │                 Tidak tahu apa pun soal HTTP maupun SQL       │    │
│  └──────────────────────────┬───────────────────────────────────┘    │
│  ┌──────────────────────────▼───────────────────────────────────┐    │
│  │  Seam           Interface yang dideklarasikan Domain          │    │
│  │                 Repository · Clock · Storage · Notifier ·     │    │
│  │                 ExternalSystem · Identity                     │    │
│  └──────────────────────────┬───────────────────────────────────┘    │
│  ┌──────────────────────────▼───────────────────────────────────┐    │
│  │  Adapter        Implementasi nyata di balik tiap seam         │    │
│  │                 SQL · HTTP client · SMTP · jam sistem         │    │
│  └──────────────────────────┬───────────────────────────────────┘    │
└─────────────────────────────┼─────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
┌───────▼────────┐  ┌─────────▼─────────┐  ┌────────▼──────────┐
│ Oracle 19c     │  │ Storage Dokumen   │  │ HCC / HCQ         │
│ → PostgreSQL   │  │ (app13 / app8)    │  │ Autentikasi       │
│    17+ (D-24)  │  └───────────────────┘  └───────────────────┘
│ SQL portabel   │  ┌───────────────────┐  ┌───────────────────┐
│ tanpa procedure│  │ BRI Surf · SLIK   │  │ 6 API pengganti   │
└────────────────┘  │ Kasir · SMTP      │  │ DB Link (D-25)    │
                    └───────────────────┘  └───────────────────┘
```

**Satu binary Go** yang menyajikan API sekaligus berkas statis SPA. Tidak ada runtime Node.js di
production (D-23). Ini menjaga operasional tetap sederhana di VM on-premise (D-08).

---

## 2. Empat lapisan dan aturan ketergantungannya

Aturannya satu kalimat, dan ini **wajib ditegakkan di code review**:

> **Ketergantungan hanya boleh mengarah ke dalam. Domain tidak boleh mengimpor apa pun dari
> Transport maupun Adapter.**

| Lapisan | Boleh tahu | Dilarang tahu |
|---|---|---|
| **Transport** | Domain | SQL, tabel, detail sistem eksternal |
| **Domain** | Interface seam yang ia deklarasikan sendiri | HTTP, SQL, JSON wire format, nama tabel |
| **Seam** | Tipe domain | apa pun tentang implementasi |
| **Adapter** | Interface seam, teknologi | aturan bisnis |

**Kenapa ini penting khusus untuk project ini:** aturan bisnis Claim PNC saat ini tersebar di
tiga tempat — activity Pega, SQL, dan stored procedure — sehingga satu perubahan aturan harus
dicari di tiga tempat dan sering hanya ditemukan di dua. Aturan ketergantungan ini memaksa
**locality**: satu aturan bisnis hidup di satu tempat.

---

## 3. Seam yang kita buat, dan alasannya

Setiap seam di bawah punya **minimal dua adapter nyata** — kalau tidak, ia tidak dibuat.

### 3.1 Repository — seam ke database

**Interface:** dideklarasikan oleh Domain, satu per aggregate.
**Adapter:** (1) implementasi SQL portabel, (2) fake di memori untuk pengujian.

Meskipun D-20 menetapkan satu set SQL portabel untuk Oracle dan PostgreSQL, seam ini tetap
nyata karena adapter keduanya adalah **fake untuk pengujian**. Itu yang membuat aturan bisnis
bisa diuji tanpa database sama sekali.

Interface dijaga tetap **dangkal secara permukaan, dalam secara isi**: ia bicara dalam bahasa
domain (`FindByNomorKlaim`, `SimpanSettlement`), bukan bahasa SQL (`Query`, `Exec`, `Scan`).

### 3.2 Clock — seam ke waktu

**Interface:** `Now()` dan konversi zona waktu.
**Adapter:** (1) jam sistem, (2) jam tetap untuk pengujian.

Ini **menjawab langsung utang teknis 4.4**. Saat ini `+7 jam` ditambahkan manual di puluhan
tempat lewat `addCalendar(...,0,0,0,0,7,0,0)` dan activity `Set7Hours`. Satu tempat yang lupa
memanggilnya menggeser tanggal 7 jam tanpa terdeteksi — dan pada aturan "Tanggal Lapor ≤ DOL +
7 hari", pergeseran itu mengubah hasil validasi.

Aturan baru yang mengikat:
- Semua waktu disimpan sebagai **UTC** di database.
- Konversi ke **WIB (Asia/Jakarta)** hanya terjadi di **satu tempat**: modul Clock.
- Aturan bisnis berbasis *hari kalender* (batas 7 hari, 30 hari, 90 hari) dihitung terhadap
  **tanggal WIB**, bukan timestamp UTC.
- **Tidak ada satu pun penambahan 7 jam manual** di kode baru.

Seam ini juga membuat seluruh aturan tanggal di bagian 3.1 Business Understanding bisa diuji
secara deterministik.

### 3.3 DocumentStore — seam ke penyimpanan dokumen

**Interface:** unggah, ambil, hapus, daftar — dalam istilah domain (dokumen klaim, kategori,
pemilik), bukan istilah HTTP.
**Adapter:** (1) API storage internal Sinarmas (D-16), (2) fake untuk pengujian.

Menyatukan **tiga mekanisme yang sekarang hidup berdampingan** (BLOB Oracle, tabel Pega, API
eksternal) menjadi satu jalur. Database aplikasi hanya menyimpan metadata dan referensi.

### 3.4 ExternalSystem — seam ke sistem lain

**Interface:** satu interface sempit per sistem, bicara dalam istilah domain.
**Adapter:** (1) HTTP client nyata, (2) fake untuk pengujian.

Menggantikan **64 pemakaian** DB Link (`D-25`) dan **21 Connect-REST** (`D-73`). Setiap sistem eksternal punya
seam sendiri — **bukan satu interface raksasa untuk semuanya**, karena kegagalan dan aturan
retry-nya berbeda-beda.

### 3.5 Identity — seam ke autentikasi

**Interface:** verifikasi kredensial, kembalikan profil pengguna.
**Adapter:** (1) API HCC/HCQ (D-07), (2) fake untuk pengujian dan pengembangan lokal.

Otorisasi **tidak** berada di seam ini — otorisasi dimiliki aplikasi dan hidup di Domain.

> **Seam ini naik derajat (2026-09-14).** `HCC` dan `HCQ` muncul **2× di seluruh export**, keduanya
> teks pesan galat — **tidak ada Connect REST, tidak ada pemetaan field, tidak ada penanganan
> kegagalan**. Adapter pertama karena itu dibangun terhadap **kontrak yang belum ada**
> (`ADR-0024`, `Proposed`), dan `F-3` tidak punya baseline untuk diuji kesetaraannya.
>
> Justru karena itu seam ini **wajib ada sejak awal**: ia satu-satunya cara `F-3` dapat dikerjakan
> sebelum kontrak HCC/HCQ tiba, dengan adapter fake sebagai penopang sementara. Konsekuensinya
> disadari — jalur yang dipakai saat pengembangan bukan jalur yang dipakai di produksi, sehingga
> kelas cacat integrasi baru muncul terlambat.

### 3.6 Notifier — seam ke pemberitahuan

**Interface:** kirim pemberitahuan sesuai peristiwa domain (`NotifikasiKerugianBesar`,
`NotifikasiPremiTertunggak`), bukan `SendEmail(to, subject, body)`.
**Adapter:** (1) SMTP, (2) fake yang merekam.

Perbedaan ini penting: dengan interface berbasis peristiwa domain, **siapa penerimanya adalah
urusan konfigurasi** (D-15), bukan urusan pemanggil. Inilah yang menghapus hardcode email UW
dan pimpinan sekaligus menutup kemungkinan blok "TESTING" terulang.

---

## 4. Modul domain yang dalam

Tiap modul di bawah menyembunyikan banyak perilaku di balik interface kecil. Ukurannya diambil
dari sistem lama agar terlihat **leverage**-nya.

### 4.1 Registrasi Klaim — modul terdalam

| | |
|---|---|
| **Interface** | Terima data registrasi, kembalikan klaim yang sah atau daftar kesalahan validasi |
| **Menyembunyikan** | 8 aturan tanggal · 2 aturan duplikasi · aturan spreading 100% · kelengkapan penyebab kerugian dan Nomor SLIK · penautan Open Protection · konversi kurs · ambang Large Losses · pembentukan snapshot polis |
| **Asal di sistem lama** | `InputRegister_act` — **137 step** |
| **Leverage** | Satu pemanggilan menggantikan 137 step yang tersebar |
| **Locality** | Seluruh aturan registrasi berubah di satu tempat |

**Uji deletion:** kalau modul ini dihapus, kompleksitasnya muncul kembali di setiap pemanggil —
API registrasi, impor batch, dan koreksi data. Modul ini jelas membayar dirinya sendiri.

### 4.2 Spreading Reasuransi

| | |
|---|---|
| **Interface** | Terima daftar spreading sebuah coverage, kembalikan hasil perhitungan atau pelanggaran aturan |
| **Menyembunyikan** | Aturan total 100% (**toleransi 4 desimal, `99,9999`–`100,0001`** (`D-51`)) · penanganan Fac Out dan kelengkapan Fac Offer · aturan Group Panel `003` · konversi Ex-Gratia `OR`→`ORS` |
| **Kenapa dalam** | Aturan ini sekarang tersebar di `InputRegister_act`, `CallSpreadingView`, dan 45 activity bernama `*Spreading*`/`*Reas*` |

### 4.3 Penjenjangan Komite

| | |
|---|---|
| **Interface** | Terima klaim, kembalikan langkah persetujuan berikutnya (atau: sudah selesai) |
| **Menyembunyikan** | Matriks nilai klaim × jenis bisnis (D-14) · penambahan `KomiteCount` · reset `AcceptStatus` · penentuan workbasket tujuan · jejak persetujuan |
| **Catatan** | Matriks adalah **master data**, bukan kode (D-15). Modul membaca matriks, tidak memuatnya. |

### 4.4 Settlement Klaim

| | |
|---|---|
| **Interface** | Ajukan nilai, setujui nilai, catat pembayaran |
| **Menyembunyikan** | Perjalanan Estimasi → Usulan → Akseptasi → Dibayar · penerbitan Nomor Akseptasi · pengurangan Salvage dan Recovery · pembagian ke koasuransi · jejak audit (D-28) |
| **Invarian yang dijaga** | Nilai tidak boleh melebihi TSI · akseptasi hanya sah setelah komite selesai · setiap perubahan nilai tercatat |

### 4.5 Pemberitahuan Reasuransi (PLA / Pre-DLA / DLA)

| | |
|---|---|
| **Interface** | Terbitkan pemberitahuan sesuai tahap klaim |
| **Menyembunyikan** | Pemilihan tahap yang tepat · perhitungan nilai per penanggung · pembuatan dokumen · pengiriman ke koasuransi/reasuransi |
| **Asal di sistem lama** | 69 activity bernama `*DLA*`/`*PLA*`/`*Treaty*`/`*XOL*`/`*Coins*` |

### 4.6 Penugasan (Worklist & Workbasket)

| | |
|---|---|
| **Interface** | Tugaskan, ambil dari antrean, lepaskan, pindahkan |
| **Menyembunyikan** | Aturan routing per tahap · pembedaan Worklist dan Workbasket (D-26) · penguncian agar dua orang tidak mengerjakan tugas yang sama |
| **Menggantikan** | `PC_ASSIGN_WORKLIST`, `PC_ASSIGN_WORKBASKET`, `PR_SYS_LOCKS`, dan 5 router |

### 4.7 Otorisasi

| | |
|---|---|
| **Interface** | Boleh atau tidak pengguna ini melakukan aksi ini pada klaim ini |
| **Menyembunyikan** | Pemetaan pengguna → peran → menu → izin (D-07) · aturan visibilitas berdasarkan cabang dan lini bisnis |
| **Menggantikan** | 22 access group dan puluhan when rule berbasis `AccessGroup.pyAccessGroup` |

### 4.8 Pembuatan Dokumen (PDF / Excel / CSV)

| | |
|---|---|
| **Interface** | Hasilkan dokumen dari data laporan |
| **Menyembunyikan** | Tata letak · penomoran halaman · streaming untuk data besar · pembatasan memori |
| **Sesuai** | D-11 — dibuat sendiri di Go, bukan engine Pega maupun BI eksternal |

---

## 5. Bounded Context

Empat konteks, dengan **Domain Klaim sebagai inti**. Batasnya ditentukan oleh **siapa pemilik
datanya**, bukan oleh kemiripan teknis.

```
┌──────────────────────────────────────────────────────────────┐
│                    DOMAIN KLAIM (inti)                        │
│                       kita miliki                             │
│                                                               │
│  Klaim · Objek Pertanggungan · Coverage · Settlement          │
│  Spreading · Komite · Survey · Salvage · Recovery             │
│  Open Protection · Receive Document · Penugasan               │
└───┬──────────────┬─────────────────┬─────────────────┬────────┘
    │ snapshot     │ referensi       │ referensi       │ kirim
    │ (D-04)       │                 │                 │
┌───▼──────────┐ ┌─▼─────────────┐ ┌─▼──────────────┐ ┌▼──────────────┐
│ POLIS        │ │ IDENTITAS     │ │ MASTER DATA    │ │ KOASURANSI &  │
│ (GISFW)      │ │ & AKSES       │ │                │ │ REASURANSI    │
│              │ │               │ │                │ │               │
│ tim lain     │ │ HCC/HCQ +     │ │ kita miliki    │ │ pihak luar    │
│              │ │ kita miliki   │ │                │ │               │
│ hanya baca   │ │               │ │ Cabang · Bisnis│ │ PLA · Pre-DLA │
│ lewat        │ │ Auth: HCC/HCQ │ │ Penyebab Rugi  │ │ DLA · LOD     │
│ snapshot     │ │ Authz: kita   │ │ Surveyor · Bank│ │               │
│              │ │               │ │ Ambang komite  │ │               │
└──────────────┘ └───────────────┘ └────────────────┘ └───────────────┘
```

### Kontrak antar konteks

| Dari → Ke | Bentuk | Aturan |
|---|---|---|
| Polis → Klaim | **Snapshot** saat registrasi (D-04) | Klaim **tidak pernah** membaca polis secara langsung setelah registrasi. Perubahan polis tidak mengubah klaim yang sudah berjalan. |
| Identitas → Klaim | Autentikasi via HCC/HCQ, profil dikembalikan | Otorisasi **tidak** didelegasikan — dimiliki Domain Klaim (D-07) |
| Master Data → Klaim | Referensi berdasarkan kode | Master data dimiliki aplikasi dan dapat diubah tanpa deploy (D-15) |
| Klaim → Koasuransi/Reasuransi | Dokumen pemberitahuan | Satu arah keluar; tidak ada ketergantungan runtime |

**Kenapa snapshot, bukan pemanggilan langsung:** GISFW dikembangkan tim lain dengan jadwal
sendiri (D-03). Bila Domain Klaim memanggilnya saat runtime, ketersediaan klaim menjadi
bergantung pada ketersediaan sistem tim lain — dan target 24/7 (D-27) menjadi mustahil dipenuhi
secara sepihak. Snapshot memutus ketergantungan itu, sekaligus benar secara bisnis: **klaim
harus dinilai berdasarkan kondisi polis pada saat kejadian, bukan kondisi hari ini.**

---

## 6. Yang sengaja tidak dibangun

Menyatakan yang tidak dibangun sama pentingnya dengan yang dibangun.

| Tidak dibangun | Alasan |
|---|---|
| **Microservices** | 200–300 user harian, satu tim, satu database. Microservices menambah kerumitan jaringan, deployment, dan penelusuran tanpa menyelesaikan satu pun masalah nyata kita. Satu binary jauh lebih cocok untuk VM on-premise. |
| **Message broker (Kafka/RabbitMQ)** | Tidak ada kebutuhan pemrosesan asinkron bervolume tinggi. Bila kelak dibutuhkan, seam Notifier sudah siap menerimanya. **Catatan:** pernyataan "job terjadwal cukup ditangani scheduler di dalam aplikasi" **belum menjadi keputusan** — dengan dua instans di belakang load balancer, mekanismenya masih dipilih di `ADR-0022` (`Proposed`). |
| **Cache terdistribusi (Redis)** | Sistem lama berjalan **tanpa caching sama sekali** (7 data page semuanya `refresh=never`). Masalah performa ada di query terhadap puluhan juta baris — itu diselesaikan index dan paginasi, bukan cache. Cache in-process untuk master data sudah memadai. |
| **Event sourcing / CQRS** | Kebutuhan jejak audit (D-28) dijawab tabel riwayat append-only yang jauh lebih sederhana dan bisa dipahami tim eks-Pega. |
| **GraphQL** | Klien hanya satu (SPA kita sendiri). REST dengan endpoint yang dirancang untuk kebutuhan layar lebih sederhana dan lebih mudah di-cache. |
| **ORM penuh** | D-20 menetapkan SQL ditulis langsung. Query warisan Pega terlalu kompleks untuk ORM, dan tim akan kesulitan men-debug SQL hasil generate. |
| **Kubernetes** | D-08 menetapkan VM on-premise. Rolling deployment 24/7 (D-27) bisa dicapai dengan dua instance di belakang load balancer. |

# 4. Domain Model dan Bounded Context

Model domain sasaran. Istilah mengikuti `CONTEXT.md` tanpa perkecualian. Struktur diturunkan
dari analisis 1.381 referensi property `ClaimData.*` dan 245 tabel di sistem lama, lalu dinamai
ulang sesuai D-19.

Dokumen ini menjelaskan **konsep dan hubungannya**, bukan DDL. Skema fisik ada di
`09-DATABASE-STRATEGY.md`.

> **Diperbarui v2.0 (2026-09-14).** Yang berubah: **Status Klaim menjadi 33 kode `1134`–`1166`**
> (`R-06` tertutup) · **aggregate Komite ditulis ulang sebagai penjenjangan kumulatif** (`D-47`,
> `D-52`, `D-70`) · tiga invarian baru `I-11`…`I-13` (kurs, presisi uang, soft delete) · `I-1`
> memakai toleransi numerik yang tegas · master data dinyatakan **≥29 kelompok**, bukan 14 · dan
> nomor klaim menjadi **`PNCN.YY.xxxx`** (`D-71`).

---

## 1. Aggregate utama: Klaim

**Klaim** adalah aggregate root. Semua di bawahnya hanya boleh diubah lewat Klaim — inilah yang
menjaga invarian seperti "total spreading harus 100%" tetap benar.

```
Klaim  ◄── aggregate root
│
├─ Identitas
│    NomorKlaim            PNCN.YY.xxxx (baru) atau PNC-xxxx (warisan) — D-22, D-71
│    NomorRegisterDokumen  kaitan ke Receive Document
│
├─ SnapshotPolis  ◄── nilai beku saat registrasi (D-04)
│    NomorPolis · ProdKe · Tertanggung · PeriodePertanggungan
│    GroupPanel · BusinessType · SourceOfBusiness · Cabang
│    TotalTSI · TotalPremi · StatusPembayaranPremi
│    DaftarKoasuransi[]     ← pembagian antar perusahaan asuransi
│    DaftarFacOffer[]       ← penawaran fakultatif
│    DaftarAlamatKirim[]
│
├─ DataKerugian
│    TanggalKejadian (DOL) · TanggalLapor · TanggalTerimaDokumen
│    LokasiKejadian · Kronologi · NilaiEstimasi · MataUang
│    HubunganTertanggung · Pelapor · KontakPelapor
│    ExGratia · FlagASO · NomorSLIK
│
├─ StatusProses          alur kerja        ──┐
├─ StatusKlaim           33 kode 1134–1166   ├─ empat konsep berbeda (D-18)
├─ FlagKlaim             penanda biner       │
├─ StatusPosisiProgres   tahapan progres   ──┘
│
├─ ObjekPertanggungan[]  ◄── entity
│  │   KodeObjek · NamaObjek · LokasiObjek · LokasiSurvei
│  │   Okupasi · JenisSurveyor · KodeCabang
│  │   AtributKendaraan   (rangka, mesin, merek, tipe, model)
│  │   AtributOrang       (tanggal lahir, NIK, status peserta)
│  │   AtributKredit      (kolektibilitas, sebab macet, hari tunggakan, suku bunga)
│  │
│  ├─ Coverage[]  ◄── entity
│  │  │   KodeCoverage · NamaCoverage · TSI · SublimitTSI
│  │  │   PenyebabKerugian · Diagnosa
│  │  │
│  │  ├─ SettlementLine[]  ◄── entity  (dulu: AdjustmentList)
│  │  │  │   NilaiEstimasi → NilaiUsulan → NilaiAkseptasi → NilaiDibayar
│  │  │  │   NomorAkseptasi · TanggalAkseptasi · MataUang
│  │  │  │   NilaiSalvage · NilaiRisikoSendiri · ShareASM
│  │  │  │   StatusAkseptasi · StatusTransferKasir · StatusPembayaran
│  │  │  │   PenerimaPembayaran · JenisPembayaran
│  │  │  │   StatusCetakLOD
│  │  │  ├─ JejakKomite[]           persetujuan berjenjang
│  │  │  ├─ PemberitahuanDLA[]      DLA yang diterbitkan
│  │  │  └─ Dokumen[]
│  │  │
│  │  ├─ RincianItem[]      sparepart, biaya, rincian per item
│  │  └─ Spreading[]        pembagian risiko — TOTAL WAJIB 100%
│  │        JenisTreaty · NamaTreaty · PersentaseShare
│  │        PremiTerbagi · Penanggung
│  │
│  └─ RincianItem[]
│
├─ HasilSurvei[]           surveyor, tanggal, lokasi, temuan, foto
├─ Komite[]                daftar dan keputusan komite
├─ Dokumen[]               metadata + referensi ke storage eksternal (D-16)
├─ PenerimaGantiRugi[]     nama, alamat, jenis pembayaran
├─ OpenProtectionTerkait[] Open Protection yang dipakai klaim ini
├─ Salvage[]               barang sisa dan hasil lelang
├─ Recovery[]              pemulihan dari pihak ketiga
├─ RiwayatKomunikasi[]     korespondensi dengan tertanggung
└─ JejakAudit[]            append-only, tidak dapat diubah (D-28)
```

---

## 2. Invarian aggregate

Aturan yang **wajib benar setiap saat**. Inilah alasan Klaim menjadi aggregate root — kalau
Coverage bisa diubah langsung tanpa lewat Klaim, invarian ini tidak bisa dijamin.

| # | Invarian | Kapan diperiksa |
|---|---|---|
| I-1 | Total Spreading setiap Coverage = 100%, diuji **`ROUND(SUM(share),4) BETWEEN 99.9999 AND 100.0001`** (`D-51`) | Saat registrasi dan setiap perubahan spreading |
| I-2 | `TanggalKejadian ≤ TanggalLapor ≤ TanggalTerimaDokumen ≤ hari ini` | Saat registrasi dan setiap perubahan tanggal |
| I-3 | `TanggalKejadian` berada dalam periode polis (+30 hari Bonding, +90 hari Travel/PA) | Saat registrasi |
| I-4 | Nilai Settlement tidak melebihi TSI Coverage | Setiap perubahan nilai |
| I-5 | Akseptasi hanya sah bila seluruh jenjang komite yang diwajibkan sudah selesai | Saat akseptasi |
| I-6 | Objek tanpa Coverage tidak boleh tersimpan | Saat registrasi |
| I-7 | Setiap perubahan nilai bisnis menghasilkan satu baris JejakAudit | Setiap perubahan |
| I-8 | Tidak ada klaim ganda: Polis + Objek + Lokasi (PA: + Penyebab Kerugian `12002`) | Saat registrasi |
| I-9 | Nomor SLIK terisi bila lini bisnis SPK / Asuransi Kredit | Saat registrasi |
| I-10 | Penyebab Kerugian terisi, kecuali lini Travel | Saat registrasi |
| **I-11** | Klaim valuta asing hanya sah bila **kurs pada tanggal kejadian** tersedia; bila tidak, klaim **ditolak** — tidak ada nilai bawaan (`D-48`) | Saat registrasi dan setiap perubahan nilai |
| **I-12** | Nilai uang disimpan **presisi penuh**; pembulatan hanya saat ditampilkan, dan perbandingan terhadap ambang memakai nilai presisi penuh (`D-51`) | Setiap perhitungan |
| **I-13** | **Tidak ada penghapusan fisik** data bernilai bisnis; penghapusan dinyatakan lewat penanda (`D-66`) | Setiap penghapusan |

> **I-1 dan I-11 menutup cacat yang berjalan hari ini.** Toleransi spreading sistem lama adalah
> **pencocokan substring**, bukan perbandingan numerik — total `199.99` dan `1100.0` ikut lolos.
> Dan fungsi kurs mengembalikan `1` ketika kurs tidak ditemukan, sehingga klaim bernilai besar
> menyusut menjadi kecil lalu **lolos tanpa komite**. Keduanya masuk daftar 13 perbaikan eksplisit
> `P-5` (`D-49`, `ADR-0017`).

---

## 3. Empat konsep status

Dikonfirmasi memang berbeda (D-18). Di sistem lama keempatnya bernama mirip dan tersimpan
berdampingan sehingga sering tertukar. Di sistem baru namanya dibuat **tidak mungkin tertukar**.

| Konsep | Nama lama | Nilai | Menjawab pertanyaan |
|---|---|---|---|
| **Status Proses** | `StatusWork` | `Berjalan`, `Selesai`, `Ditolak` | Klaim ini masih dikerjakan atau sudah tuntas? |
| **Status Klaim** | `StatusClaim` | **33 kode, `1134`–`1166`** | Klaim ini berada di keadaan bisnis apa? |
| **Flag Klaim** | `ClaimStatus` | `0` / `1` | *(makna persisnya masih perlu dikonfirmasi)* |
| **Status Posisi Progres** | `StatusPosisi` | `On Progress`, `Done` | Tahapan progres mana yang sudah dilalui? |

**`R-06` tertutup.** Isi master diterima (`v_sts_claim.csv`) dan **memperbaiki premisnya sendiri**:
domain `StatusClaim` bukan 10 kode `1142`–`1151`, melainkan **33 kode `1134`–`1166`**. Sebelas
kode pertama (`1134`–`1144`) membawa penomoran lama `01`–`11`; sisanya hanya bernomor baru.

Contoh: `1142` Rejected Claim · `1143` Close Claim for this object · `1144` Cancelled Claim ·
`1147` Register · `1149` Claim Committee · `1163` Paid · `1164` Reopen Claim.

> **Rentang `1142`–`1151` yang dipakai dokumen-dokumen awal ternyata sebagian dari domainnya,
> bukan keseluruhannya.** Tiga arti kode yang sempat disimpulkan dari pemakaiannya di rule
> **seluruhnya salah** — `1143` bukan status awal melainkan Close Claim, `1150` bukan penanda
> terdaftar melainkan LOD Report, `1151` bukan Investigator melainkan Analyst. Arti kode status
> tidak boleh disimpulkan dari pemakaian.

**Dua catatan yang membatasi dua dari empat konsep:**

- **`ClaimStatus` milik ruleset GISFW**, bukan Claim PNC — salah satu dari "empat konsep" berada
  di bounded context tim lain, dan hubungannya dengan snapshot polis (`D-04`) belum dirumuskan.
- **`StatusPosisi` hanya pernah muncul dengan satu nilai** (`'On Progress'`) pada 21 titik panggil
  di export. Kueri `SELECT DISTINCT STATUSPOSISI` pada produksi masih ada di daftar permintaan DBA
  untuk menyelesaikannya secara empiris. Pemilik bisnis menegaskan keempat konsep memang berbeda,
  sehingga keempatnya tetap dipertahankan (`D-18`, `ADR-0018`).

---

## 4. Aggregate lain

### Komite
Aggregate tersendiri karena punya siklus hidup sendiri dan **berjenjang secara kumulatif**.

```
Komite
├─ NomorKomite · KlaimTerkait · JenisKomite
├─ JenjangDibutuhkan[]  ← seluruh jenjang yang ambang bawahnya sudah terlampaui (D-47)
├─ JenjangTerpenuhi     ← pencacah persetujuan yang sudah masuk
├─ AnggotaKomite[]      nama, peran, urutan (DEGREE)
├─ Keputusan[]          setuju / tolak / kembalikan · catatan · waktu · oleh siapa
└─ StatusPersetujuan
```

**Invarian:** klaim tidak boleh diakseptasi sebelum **seluruh jenjang yang dibutuhkan** menyetujui.

**Cara jumlah jenjang dihitung** (`D-47`, `D-52`, `D-70` — `ADR-0014`):

1. **Untuk lini Non-MBU saja**, pita nilai dipilih lebih dulu: sampai **Rp 100.000.000** → pita
   `1`; di atasnya → pita `2`. Akumulasi kemudian berjalan **di dalam pita itu saja**.
2. **Akumulasi menurut ambang bawah.** Setiap jenjang yang **ambang bawahnya sudah terlampaui**
   nilai klaim ikut menyetujui — bukan memilih satu jenjang tunggal. Jumlah penyetuju **bertambah**
   seiring besarnya klaim.

> **Ini bukan "matriks nilai × jenis bisnis" yang memilih satu baris.** Di sistem lama, jumlah
> jenjang = **jumlah baris yang dikembalikan kueri**
> (`Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459`), dan kueri itu menyaring **hanya
> dengan batas bawah**. `LIMIT_BOTTOM` muncul di 11 SQL rule; **`LIMIT_TOP` muncul di 0**.
> Menerapkan rentang tertutup akan mengembalikan tepat satu baris — **satu jenjang persetujuan
> berapa pun nilai klaim** — dan menghapus penjenjangan yang menjadi inti `D-14`.

**`LIMIT_TOP` tetap dipakai, tetapi sebagai validasi integritas master**: sistem baru menolak
master yang rentangnya tumpang tindih atau berlubang antar `DEGREE` dalam satu `TYPE_BUSINESS` +
`TYPE_KOMITE`. Karena tangga PA tidak tersusun menurut pita, validasi itu **hanya dapat dijalankan
per lini**, bukan lintas lini.

**Peringatan model data:** kolom `TYPE_KOMITE` **memikul dua arti berbeda** — pita nilai di
Non-MBU, dan **varian jalur** (PA reguler versus PA TKI) di lini PA. Pemisahannya menjadi dua kolom
dipertimbangkan dan **tidak diambil**, sehingga arti gandanya wajib didokumentasikan di master
baru.

**Catatan penugasan:** penjenjangan komite menugaskan ke **operator bernama**, bukan ke workbasket
— seluruh export hanya memuat **4 workbasket**. Model penugasan komite karena itu tidak mengikuti
pola Worklist/Workbasket pada aggregate Penugasan di bawah.

### Open Protection
```
OpenProtection
├─ NomorProteksi · NomorPolis · JenisProteksi
├─ TanggalInput · Pemohon
├─ StatusAkseptasi
└─ SudahDipakaiKlaim   ← ditandai saat ditautkan ke klaim
```

### Receive Document
```
ReceiveDocument
├─ NomorRegisterDokumen (RCV_ID)
├─ CabangPengirim · Ekspedisi · NomorResi
├─ TanggalKirim · EstimasiTiba · TanggalTerima
├─ JumlahLembar · JenisDokumen
└─ KlaimTerkait
```

### Penugasan
```
Penugasan
├─ KlaimTerkait · Tahap · JenisPenugasan (Worklist | Workbasket)   ← D-26
├─ DitugaskanKe        diisi bila Worklist
├─ Workbasket          diisi bila Workbasket
├─ DiambilOleh · WaktuDiambil    penguncian antar-pengguna
└─ StatusPenugasan
```

---

## 5. Master data

Semua yang berikut adalah **data yang dapat diubah tanpa deploy** (D-15). Di sistem lama
sebagian di-hardcode di dalam activity.

> **Koreksi jumlah (2026-09-14).** Verifikasi terhadap export menemukan **≥29 kelompok master**,
> bukan 14. Daftar di bawah adalah **14 kelompok yang sudah terpetakan**; sisanya diinventarisasi
> saat tiket `F-4` ditulis. Akibatnya `F-4` naik menjadi **Besar** dan `U-6` dari Sedang menjadi
> **Besar**.

| Master | Isi | Menggantikan hardcode |
|---|---|---|
| Lini Bisnis | Group Panel, Business Type | — |
| Penyebab Kerugian | kode, deskripsi, kaitan lini bisnis | — |
| Cabang & Wilayah | kode, nama, kanwil | — |
| Surveyor & Adjuster | internal/eksternal, spesialisasi | — |
| Jenis Treaty | OR, ORS, Fac Out, XOL, BPPDAN | — |
| **Mata Uang & Kurs** | kode, **kurs per tanggal** — konversi memakai kurs **tanggal kejadian**; bila tidak ada, klaim **ditolak** tanpa nilai bawaan (`D-48`) | ✅ `GETCURRENCYSTANDARD` yang mengembalikan `1` saat kurs tidak ditemukan |
| Bank & Rekening | untuk pembayaran | — |
| Jenis Dokumen | per lini bisnis, wajib/opsional | — |
| **Ambang Komite** | **tangga ambang bawah per lini** (`TYPE_BUSINESS` + `TYPE_KOMITE` + `LIMIT_BOTTOM` + `DEGREE`), diakumulasi (`D-47`, `D-70`) | ✅ **8 ambang komite unik**, antara lain `50000000`, `30000000`, `20000000`, `7000`, `3500` |
| **Penerima Notifikasi** | per jenis peristiwa dan Group Panel — **mailbox fungsional saja, tidak ada akun pribadi** (`D-67`) | ✅ **66 alamat email**, termasuk ≥6 akun Gmail pribadi di jalur produksi |
| **Ambang Large Losses** | nilai pemicu Notice of Large Losses | ✅ `1000000000` |
| **Batas Aturan Tanggal** | 7 hari, 30 hari, 90 hari per lini bisnis | ✅ tertanam di validasi |
| Peran & Izin Menu | **22 peran** → 51 item menu (`D-58`, `D-59`) | ✅ 22 access group + **24 Operator ID hardcode** |
| Master Status Klaim | **33 kode `1134`–`1166`** dan artinya | — |
| **Hari Libur & Jam Kerja** | kalender libur dan definisi jam kerja untuk perhitungan TAT (`D-50`) | ✅ `DATAMINING.GET_WORKING_HOURS@ASMD` dan `HRD_LBR` lewat DB Link |
| **Retensi Data** | lama simpan data klaim **dan** jejak audit — satu kebijakan (`D-62`) | ✅ tidak ada; belum pernah ditetapkan |

---

## 6. Peristiwa domain

Peristiwa yang memicu akibat lain. Dipakai sebagai interface seam Notifier (bagian 3.6 Future
Architecture) — pemanggil menyatakan **apa yang terjadi**, bukan **siapa yang harus diberi tahu**.

| Peristiwa | Akibat |
|---|---|
| `KlaimDiregistrasi` | Notifikasi register · penautan Open Protection · pencatatan progres |
| `EstimasiMelebihiAmbangBesar` | Notice of Large Losses ke UW sesuai Group Panel dan pimpinan |
| `PremiBelumLunas` | Notifikasi premi tertunggak |
| `KlaimDiestimasi` | Penerbitan **PLA** ke koasuransi/reasuransi |
| `NilaiDiusulkan` | Penerbitan **Pre-DLA** |
| `KomiteMenyetujui` | Lanjut ke jenjang berikutnya, atau buka akseptasi |
| `KomiteMenolak` | Klaim masuk jalur **RCL** |
| `KlaimDiakseptasi` | Penerbitan **DLA** dan **LOD** ke tertanggung |
| `KlaimDitransferKeKasir` | Permintaan pembayaran |
| `KlaimDibayar` | Penutupan klaim · pencatatan progres |
| `KlaimDitolak` (RCL) | Surat penolakan · penutupan |
| `KlaimDiprosesUlang` (PUCL) | Pembukaan kembali klaim yang sudah ditutup |

---

## 7. Pemetaan ke sistem lama

Rujukan saat implementasi, agar penelusuran ke source Pega tetap mungkin.

| Konsep baru | Kelas Pega | Tabel Oracle |
|---|---|---|
| Klaim | `ASM-FW-GCNMFW-Work-PNC` | `POOLDATA.T_CLAIM_PNC` + `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` |
| DataKerugian | `ASM-FW-GCNMFW-Data-ClaimData` | kolom di `T_CLAIM_PNC` |
| SnapshotPolis | `ASM-FW-GISFW-Data-Policy` | `POOLDATA.JSON_POLIS.DATA_JSONBLOB` |
| ObjekPertanggungan | `ASM-FW-GCNMFW-Data-Object` | `POOLDATA.T_CLAIM_OBJECTLIST` |
| Coverage | `ASM-FW-GCNMFW-Data-ObjectCoverage` | `POOLDATA.T_CLAIM_OBJECTCOVERAGE` |
| SettlementLine | `ASM-FW-GCNMFW-Data-Adjustment` | `POOLDATA.T_CLAIM_ADJUSTMENT` |
| Spreading | `ASM-FW-GISFW-Data-SpreadingRisk` | tertanam di JSON dan tabel reas |
| Komite | `ASM-FW-GCNMFW-Work-Komite` | `POOLDATA.T_CLAIM_KOMITE_LIST` |
| OpenProtection | `ASM-FW-GCNMFW-Work-OpenProtection` | `POOLDATA.MST_PROTEKSI_DATA_PNC` |
| ReceiveDocument | `ASM-FW-GCNMFW-Work-ReceiveDocument` | tabel receive |
| HasilSurvei | `ASM-FW-GCNMFW-Data-Survey` | `POOLDATA.T_SURVEYORLIST`, `T_DOC_SURVEY` |
| Dokumen | `Data-WorkAttach-File` | `POOLDATA.DATA_ATTACHFILE`, `GENERAL.T_STORAGE_IMAGE` |
| Penugasan | `Assign-Worklist` / `Assign-Workbasket` | `DATAPEGA.PC_ASSIGN_*` |
| StatusPosisiProgres | — | `POOLDATA.GCNM_PROGRESS_CLAIM` + `GCNM_MST_PROGRESS_KLAIM` |
| Salvage | — | `POOLDATA.PNC_SALVAGE`, `DETAIL_PNC_SALVAGE` |
| PLA / DLA | `ASM-FW-GCNMFW-Data-PLA` / `-DLA` | `POOLDATA.T_PLALIST`, `T_DLALIST`, `T_PREDLALIST` |

# 5. Module Breakdown dan Urutan Migrasi

Pembagian modul beserta ukuran pekerjaannya. Ukuran diambil dari pemetaan 778 activity custom
(di luar 124 activity bawaan Pega), 652 rule SQL, 74 harness, dan 56 report definition.

Kolom **Bergantung pada** menentukan urutan pengerjaan: modul tidak bisa dimulai sebelum
dependensinya selesai.

> **Diperbarui v2.0 (2026-09-14).** Jumlah modul menjadi **33**, bukan 32 — `FR-S8` menjadi modul
> **`S-8` Perkakas Uji Kesetaraan** dan masuk **gelombang 1** (`D-42`), karena ia satu-satunya
> modul yang memblokir gerbang 1 setiap modul lain. Tiga ukuran dikoreksi terhadap bukti:
> **master data ≥29 kelompok** (bukan 14), **median kolom grid 6** (bukan 18–27), dan **daftar job
> terjadwal sudah diketahui** (`R-02` tertutup). Ringkasan di
> [`21-RIWAYAT-REVISI.md`](21-RIWAYAT-REVISI.md).

---

## 1. Modul fondasi — wajib selesai lebih dulu

Tidak ada modul bisnis yang bisa jalan tanpa lima modul ini.

| # | Modul | Isi | Ukuran | Bergantung pada |
|---|---|---|---|---|
| F-1 | **Kerangka Aplikasi** | Struktur folder, konfigurasi, logging, error handling, health check, graceful shutdown, penyajian SPA | Kecil | — |
| F-2 | **Akses Data** | Koneksi, connection pool, transaksi, seam Repository, SQL portabel, migrasi skema | Sedang | F-1 |
| F-3 | **Identitas & Akses** | Login via HCC/HCQ, session/token, **22 peran** dan izin menu, middleware otorisasi di **setiap endpoint** (D-58, D-59) | **Besar — tanpa baseline Pega** | F-1, F-2 |
| F-4 | **Master Data** | **≥29 kelompok master** (lihat Domain Model §5) + layar pengelolanya. Menghapus seluruh hardcode: **66 email · 24 user ID · 8 ambang komite · 3 hostname** (D-15) | **Besar** | F-1, F-2, F-3 |
| F-5 | **Waktu & Zona Waktu** | Seam Clock, penyimpanan UTC, konversi WIB tunggal, aturan hari kalender | **Sedang** — seam-nya kecil, tetapi migrasinya menyentuh **118 titik +7 jam di 36 activity** dan **101 titik +12 jam** | F-1 |
| F-6 | **Portal & Multi-Sumber Data** | Pilihan portal, koneksi per entitas, kewenangan per portal (`D-75`) — **4 portal**, satu database per entitas | Sedang | F-1, F-2, F-3 |

> **F-5 kecil tapi tidak boleh dilewati.** Ia menghapus 7-jam-manual yang tersebar di sistem
> lama. Bila dikerjakan belakangan, seluruh aturan tanggal harus ditulis ulang.

---

## 2. Modul bisnis inti

| # | Modul | Cakupan | Activity lama | Bergantung pada |
|---|---|---|---|---|
| B-1 | **Polis & Snapshot** | Ambil data polis, bekukan sebagai snapshot saat registrasi (D-04) | 18 | F-1…F-5 |
| B-2 | **Registrasi Klaim** | Validasi tanggal, duplikasi, kelengkapan; pembentukan objek dan coverage. Modul terdalam (137 step) | 22 | B-1 |
| B-3 | **Objek & Coverage** | Objek pertanggungan per lini bisnis, coverage, penyebab kerugian, rincian item | 44 | B-2 |
| B-4 | **Spreading Reasuransi** | Aturan total 100%, Fac Out, Ex-Gratia OR→ORS, share ASM | (bagian dari 69) | B-3 |
| B-5 | **Estimasi & Settlement** | Estimasi → Usulan → Akseptasi → Dibayar; salvage; risiko sendiri | 29 | B-3, B-4 |
| B-6 | **Penugasan & Inbox** | Worklist, Workbasket, routing, penguncian (D-26) | 30 | F-3 |
| B-7 | **Komite** | Penjenjangan 1–4 level, matriks nilai × jenis bisnis, jejak persetujuan | 34 | B-5, B-6, F-4 |
| B-8 | **Survey & Adjuster** | Penugasan surveyor, hasil survei, foto lapangan (mobile-friendly per D-12) | 30 | B-6 |
| B-9 | **PLA / Pre-DLA / DLA** | Pemberitahuan bertahap ke koasuransi/reasuransi | 69 (gabungan) | B-4, B-5 |
| B-10 | **Akseptasi & Pembayaran** | Nomor akseptasi, transfer kasir, status pembayaran, LOD | 31 | B-5, B-7 |
| B-11 | **RCL / PUCL / Compliance** | Penolakan, proses ulang, pemeriksaan kepatuhan, investigator, analyst doctor | 6 + jalur alur | B-6 |
| B-12 | **Salvage & Recovery** | Barang sisa, lelang, pemulihan, virtual account | 26 | B-5 |
| B-13 | **Open Protection** | Alur buka proteksi dan penautannya ke klaim | 12 | B-2 |
| B-14 | **Receive Document** | Penerimaan dokumen fisik, ekspedisi, resi | 8 | F-3 |

---

## 3. Modul pendukung

| # | Modul | Cakupan | Ukuran | Bergantung pada |
|---|---|---|---|---|
| S-1 | **Dokumen & Lampiran** | Unggah/ambil lewat API storage internal (D-16), kategori, kelengkapan dokumen | 54 activity | F-1, F-2 |
| S-2 | **Laporan & Export** | 56 laporan + engine PDF/Excel/CSV buatan sendiri (D-11) | 78 activity, 56 RD | seluruh modul bisnis |
| S-3 | **Notifikasi** | Seam Notifier, 5 correspondence, penerima dari master data | 14 activity | F-4 |
| S-4 | **Integrasi Eksternal** | **21 Connect REST keluar** + 6 API pengganti DB Link (D-25) + **4 layanan REST masuk** (`Service REST/`) | 7 activity + **21 REST** + 4 service | F-1 |
| S-5 | **Jejak Audit** | Pencatatan append-only seluruh perubahan bernilai bisnis (D-28) | — (baru, **tanpa baseline Pega**) | F-2 |
| S-6 | **Penjadwalan (Scheduler)** | **5 job terjadwal + 1 agent**, seluruhnya terverifikasi (D-57) | 5 job · 1 agent · 8 activity target | F-1 |
| S-7 | **Dashboard & Monitoring Bisnis** | Dashboard klaim, TAT, KPI, progres | (bagian dari 78) | S-2 |
| **S-8** | **Perkakas Uji Kesetaraan** | Menjalankan kasus yang sama di Pega staging dan Go staging, membandingkan hasil, **dan mengklasifikasikan setiap selisih** terhadap 13 butir `P-5` (D-42, D-54) | Besar — **baru 100%** | F-1, F-2 |

> **`S-8` masuk gelombang 1, bukan gelombang akhir.** Ia satu-satunya modul yang **memblokir
> cutover setiap modul lain**: gerbang 1 setiap modul adalah uji kesetaraan otomatis, sehingga
> menaruh `S-8` di belakang berarti tidak ada satu modul pun dapat lulus sampai gelombang itu tiba
> — membatalkan gagasan Strangler Fig (`D-05`). Lihat `ADR-0027`.
>
> **Dua modul tidak dapat memakai `S-8`**: `F-3` dan `S-5` tidak punya baseline Pega, sehingga
> gerbang 1 keduanya diganti uji fungsional terhadap kontrak (`D-56`, `ADR-0028`).

---

## 4. Frontend

| # | Modul | Cakupan | Ukuran |
|---|---|---|---|
| U-1 | **Kerangka SPA** | Routing, layout, autentikasi, penanganan error, state global | Sedang |
| U-2 | **Pustaka Komponen** | Tabel baku (**median 6 kolom**, paginasi server-side), form baku, unggah berkas, pemilih tanggal | **Sedang** |
| U-3 | **Layar Inbox** | **26** harness bernama *Inbox*, tetapi **Inbox = daftar pekerjaan pengguna** (`D-79`) — **7 di antaranya layar data acuan** dan diusulkan ke `U-6` | Besar |
| U-4 | **Layar Transaksi** | Registrasi, estimasi, survei, komite, akseptasi | Besar |
| U-5 | **Layar Laporan** | 56 laporan + unduhan | Besar |
| U-6 | **Layar Master Data** | Pengelolaan **≥29 kelompok master** | **Besar** |

**Total permukaan UI:** 74 harness, 269 section, 268 di antaranya bergrid.

> **U-2 adalah investasi terpenting di frontend.** 268 section memakai pola grid yang sama.
> Membuat satu komponen tabel baku yang benar akan dipakai ratusan kali (**leverage**), dan
> perbaikan bug di satu tempat memperbaiki seluruh layar (**locality**). Melewatkan U-2 berarti
> 268 implementasi tabel yang berbeda-beda — persis kegagalan yang harus dicegah pada tim di D-09.

> **Koreksi ukuran terhadap bukti (2026-09-14).**
>
> - **`U-2` turun dari Besar menjadi Sedang.** Spesifikasi "tabel baku 18–27 kolom" salah sasaran:
>   **median kolom sebenarnya 6**, dan tiga fitur grid yang biasanya paling mahal — tambah baris
>   inline, hapus baris inline, dan resize kolom — **tidak dipakai sama sekali** di seluruh export.
> - **`U-6` naik dari Sedang menjadi Besar**, dan `F-4` ikut naik: master data berjumlah
>   **≥29 kelompok**, bukan 14.
> - **Masalah paginasi bukan `OFFSET` besar.** `OFFSET` **nol kemunculan** di seluruh export;
>   yang nyata adalah **3.189 grid terikat page list klipboard** dan `pyMaxRecords=500` pada
>   **54 dari 56** laporan. Paginasi keyset karena itu **perubahan perilaku**, bukan pemeliharaan,
>   dan harus diuji per layar.

---

## 5. Urutan pengerjaan berdasarkan ketergantungan

```
Gelombang 1 — Fondasi          F-1 → F-2 → F-5 → F-3 → F-4
                               S-8 (perkakas uji kesetaraan)
                                       ↓
Gelombang 2 — Kerangka UI      U-1 → U-2          S-5 (jejak audit)
                                       ↓
Gelombang 3 — Jalur klaim      B-1 → B-2 → B-3 → B-4 → B-5
                               B-6 (penugasan) · B-14 (receive) · S-1 (dokumen)
                                       ↓
Gelombang 4 — Persetujuan      B-7 (komite) → B-10 (akseptasi) → B-11 (RCL/PUCL)
                               B-8 (survey) · B-13 (open protection)
                                       ↓
Gelombang 5 — Nilai & luar     B-9 (PLA/DLA) · B-12 (salvage) · S-3 · S-4
                                       ↓
Gelombang 6 — Laporan          S-2 · S-7 · U-5
                                       ↓
Gelombang 7 — Sisa             S-6 (scheduler) · U-6
```

**Empat rantai kritis** yang tidak bisa diparalelkan:
1. `F-2 → B-1 → B-2 → B-3 → B-4 → B-5 → B-7 → B-10` — jalur nilai klaim
2. `F-3 → F-4 → B-7` — komite butuh master ambang persetujuan
3. `U-2 → U-3/U-4/U-5` — semua layar butuh komponen tabel baku
4. **`S-8` → gerbang 1 seluruh modul** — tidak ada modul yang dapat dinyatakan lulus sebelum
   perkakas uji kesetaraan berjalan (`D-42`)

**`S-6` tidak lagi menunggu `R-02`.** Daftar job sudah lengkap (`D-57`); yang tersisa adalah
keputusan rancangan di `ADR-0022` — siapa yang menjalankan job saat aplikasi hidup di dua instans,
dan apakah `AutoAcceptKomite` yang menyetujui komite otomatis tiap hari jam 06:00 dipertahankan.

---

## 6. Modul yang terhalang informasi

Modul berikut **tidak dapat diselesaikan** sampai informasi yang hilang dilengkapi.

Diperbarui 2026-09-14 setelah export bertambah dan `Database/` diterima (`D-45`).

| Modul | Terhalang oleh | Risiko | Status |
|---|---|---|---|
| B-7 Komite | Isi tabel `POOLDATA.EMAILKOMITE` belum diambil | D-14 | ✅ **lepas** — 21 kolom, 30 baris diterima; model kumulatif ditetapkan (`ADR-0014`) |
| S-6 Scheduler | Rule Agent/Queue Processor tidak ada di export | R-02 | ✅ **lepas** — 5 job + 1 agent (`D-57`); sisa keputusan rancangan di `ADR-0022` |
| Seluruh modul berstatus | Arti kode status `1142`–`1151` belum ada | R-06 | ✅ **lepas** — 33 kode `1134`–`1166` diterima |
| B-6 Penugasan | `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter` tidak ada di export | R-04 | **Turun** — algoritma beban terbaca dari `BrowsePICRandomTeam-SQL.xml:39-40`; tinggal verifikasi |
| B-9, B-10, B-12 | Source procedure & function database belum ada | R-01 | **Lepas dari `BRD §21.4`** (`D-55`); sisa penghalang spesifik per modul |
| **B-5 Settlement** | isi `POOLDATA.GCNM_FEE_SCALE` (17 pita) · isi `m_currencystandard` · `BrowseT_Claim_Adjustment_SQL` | R-01, R-19 | **Masih terikat `BRD §21.4`** |
| Seluruh modul | **12 dependensi** yang dipanggil 62 procedure yang sudah diterima — terberat `UPDATE_LOG_KONVERSI` (162×) | R-01 | **Terbuka** |
| S-4 Integrasi | API pengganti 6 DB Link kemungkinan belum ada | R-03 | Terbuka · **lingkup bertambah**: 4 layanan REST masuk belum pernah dihitung, dan **9 Connect REST keluar baru ditemukan** — totalnya 21, bukan 12 (`D-73`) |
| S-3 Notifikasi | `SendEmailNotification` (dipanggil 15×) tidak ada di export | R-07 | Terbuka |
| **F-3 Identitas** | **kontrak API HCC/HCQ — nol jejak di export** | R-14, R-16 | **Terbuka** — `ADR-0024`; menghalangi login seluruh aplikasi |
| **F-3 Identitas** | penugasan operator ke peran **tidak ada di database**; 5 When rule menu hilang | R-16 | **Terbuka** — tabel izin dapat dibangun tetapi **tidak dapat diisi** |
| **F-4 Master Data** | tujuan penyimpanan rahasia belum ditetapkan | R-17 | **Terbuka** — `D-40` |
| **S-5 Jejak Audit** | **daftar peristiwa wajib audit** dari Compliance | R-14 | **Terbuka** — `ADR-0026` |
| **S-8 Perkakas Uji** | ketersediaan **Pega staging yang dapat ditembak dari luar** | R-14 | **Terbuka** — penghalang gerbang 1 seluruh modul |
| B-7, B-11, B-13, B-14 | **8 Ticket rule custom hilang**; 14 dari 17 nama tanpa pemicu di export | R-16 | **Terbuka** — `ADR-0021` |
| **B-2 Registrasi** | pengganti pola hapus-lalu-sisip-ulang belum diputuskan | — | **Terbuka** — `ADR-0013` |

> Ini bukan alasan menunda mulai — modul fondasi dan sebagian besar jalur klaim tidak terhalang.
> Tapi daftar ini menunjukkan **apa yang harus diminta sekarang juga**, karena waktu tunggunya
> ada di luar kendali tim pengembang.
>
> **Yang berubah sejak v1.0:** empat penghalang tertutup (`D-14`, `R-02`, `R-06`, sebagian `R-01`),
> satu turun derajat (`R-04`), dan **tujuh penghalang baru** muncul dari verifikasi bukti —
> seluruhnya menyangkut artefak atau keputusan yang sebelumnya tidak diketahui hilang.

# 6. Strategi Migrasi

Strategi pemindahan dari Pega ke Go + React. Mengikuti **Strangler Fig** (D-05): Pega dan Go
berjalan paralel, modul dialihkan satu per satu.

> **Diperbarui v2.0 (2026-09-14).** Daftar perbaikan eksplisit `P-5` bertambah dari **4 menjadi
> 13 butir** (`D-49`) · Tahap 0 diperbarui dengan status nyata tiap permintaan, empat tertutup dan
> enam baru · nomor klaim menjadi `PNCN.YY.xxxx` (`D-71`) · dan verifikasi kesetaraan kini
> dikerjakan modul **`S-8`** yang masuk gelombang 1 (`D-42`).

---

## 1. Prinsip yang mengikat

### P-1 · Satu tabel hanya boleh ditulis satu sistem
Selama masa paralel, setiap tabel punya **tepat satu pemilik**. Modul yang sudah pindah ke Go
memiliki tabelnya dan Pega hanya membaca; modul yang belum pindah tetap dimiliki Pega dan Go
hanya membaca.

**Tidak ada sinkronisasi dua arah.** Dua sistem yang sama-sama menulis ke tabel yang sama akan
menghasilkan konflik data yang hampir mustahil dilacak, apalagi ketika keduanya punya aturan
validasi yang berbeda.

### P-2 · Batas modul mengikuti batas kepemilikan tabel
Konsekuensi langsung dari P-1: urutan migrasi tidak bebas. Sebuah modul hanya bisa dipindahkan
bila **seluruh tabel yang ia tulis** bisa ikut berpindah kepemilikan bersamanya.

### P-3 · Klaim yang sedang berjalan tidak berpindah sistem di tengah jalan
Klaim yang sudah dimulai di Pega **diselesaikan di Pega**. Klaim baru dimulai di Go setelah
modul registrasi dialihkan. Nomor `PNCN.YY.xxxx` versus `PNC-xxxx` (D-22, D-71) membuat asal setiap klaim
langsung terbaca tanpa tabel pemetaan.

Ini menghindari kelas bug terburuk dalam migrasi bertahap: klaim setengah jalan yang datanya
ditulis dua sistem dengan aturan berbeda.

### P-4 · Migrasi skema selalu backward-compatible
Karena target 24/7 (D-27) menuntut rolling deployment, versi lama dan baru aplikasi berjalan
bersamaan terhadap skema yang sama. Karena itu:

- Menambah kolom: boleh langsung, harus *nullable* atau punya *default*.
- Menghapus kolom: **dua tahap** — berhenti dipakai lebih dulu, dihapus pada rilis berikutnya.
- Mengganti nama kolom: **tidak pernah langsung** — tambah kolom baru, tulis ke keduanya,
  pindahkan pembacaan, baru hapus yang lama.
- Mengubah tipe kolom: lewat kolom baru, tidak pernah `ALTER` di tempat.

### P-5 · Perilaku dipertahankan lebih dulu, diperbaiki kemudian
Selama migrasi, **hasil yang benar adalah hasil yang sama dengan Pega** — kecuali untuk hal-hal
yang secara eksplisit diputuskan diperbaiki.

Perbaikan lain dicatat sebagai Future Enhancement, tidak dikerjakan sambil jalan. Alasannya:
bila hasil berbeda, kita harus bisa memastikan itu **bug**, bukan **perbaikan yang tidak
tercatat**.

**Daftar perbaikan eksplisit — 13 butir** (naik dari 4; `D-49`). Setiap selisih yang muncul pada
uji kesetaraan **wajib dapat dipetakan ke salah satu butir ini, atau dinyatakan sebagai bug**.

| # | Perbaikan | Sumber |
|---|---|---|
| 1 | Hardcode menjadi konfigurasi dan master data | `D-15` |
| 2 | Blok `// TESTING` yang menimpa email produksi dihapus | `D-15` |
| 3 | Penanganan zona waktu terpusat di `F-5` | `D-13`, `R-12` |
| 4 | Penamaan domain mengikuti `CONTEXT.md` | `D-19` |
| 5 | Toleransi spreading: pencocokan substring → `ROUND(SUM(share),4) BETWEEN 99.9999 AND 100.0001` | `D-49` #1, `D-51` |
| 6 | Ambang Rp 50 juta: **tiga operator berbeda** di tiga rule → satu operator | `D-49` #2 |
| 7 | Penyesuaian 7 jam yang **asimetris di dalam satu kondisi validasi** diperbaiki | `D-49` #3 |
| 8 | Kurs memakai **tanggal kejadian**, bukan hari eksekusi | `D-49` #4, `D-48` |
| 9 | Kurs tidak ditemukan → **klaim ditolak**, bukan `RETURN 1` | `D-49` #5, `D-48` |
| 10 | `LIMIT_TOP` menjadi **validasi integritas master**; tie-breaker acak dibatasi | `D-49` #6, `D-47` |
| 11 | `NilaiSalvage` **selalu** ditambahkan, bukan hanya bila baris terakhir kebetulan bertipe salvage | `D-49` #8 |
| 12 | `INSERT_SALVAGE` tidak lagi menulis `IDSALVAGE = NULL` saat update | `D-49` #9 |
| 13 | `GETSELISIHJAM` gagal tidak lagi mengembalikan `0` jam yang tak terbedakan dari nol | `D-49` #10 |

**Satu cacat sengaja direplikasi.** Tanggal PLA/DLA diambil dari **waktu sistem**, bukan dari
tanggal yang dipilih pengguna — parameter `TTGLPLADLA` yang diterima lalu dibuang **bukan cacat**,
melainkan perilaku yang memang benar (`D-49` #7). Sistem baru boleh menghapus parameter itu
seluruhnya.

**Dua perubahan yang bukan "perbaikan hasil" tetapi tetap mengubah keluaran**, dan harus
diperlakukan di sisi perkakas pembanding, bukan sebagai selisih: **soft delete menyeluruh**
(`D-66`) mengubah isi tabel tanpa mengubah apa yang dilihat pengguna, dan **transaksi atomik pada
`B-4`/`B-9`** (`D-68`) mengubah keadaan akhir saat terjadi kegagalan.

---

## 2. Tahapan

### Tahap 0 — Persiapan (tidak menghasilkan fitur, tapi memblokir semuanya)

Diperbarui 2026-09-14. Tanda ✅ menandai kegiatan yang **sudah selesai**.

| Kegiatan | Keluaran | Status |
|---|---|---|
| Minta source 64 procedure & function dari DBA | Logika bisnis yang tersembunyi terbaca (**R-01**) | ✅ **62 dari 64 diterima** |
| Minta **12 dependensi** yang dipanggil 62 procedure itu | `UPDATE_LOG_KONVERSI` (162×), `GETNEWID` (42×), … | **Terbuka** |
| Minta export Rule-Agent / Queue Processor dari Pega | Daftar job terjadwal (**R-02**) | ✅ **5 job + 1 agent** (`D-57`) |
| Minta DDL lengkap tabel `POOLDATA` dan `DATAPEGA` | Tipe kolom, index, constraint (**R-08**) | Terbuka |
| Ambil isi master `V_STS_CLAIM` | Arti kode status (**R-06**) | ✅ **33 kode `1134`–`1166`** |
| Ambil isi tabel `POOLDATA.EMAILKOMITE` + konfirmasi aturan `SetEmailKomite` | Tangga penjenjangan komite (D-14) | ✅ **21 kolom, 30 baris**; aturan berbasis nama orang **dicabut** (`D-52`, `D-70`) |
| **Permintaan export ulang berbasis Product rule** | ±242 rule hilang, **137 di antaranya When rule** (**R-16**) | **Terbuka** (`D-39`) |
| Inventarisasi 6 API pengganti DB Link bersama tim pemilik | Kontrak integrasi (**R-03**) | Terbuka |
| **Minta kontrak API HCC/HCQ** | Pengganti gerbang 1 untuk `F-3` (**R-14**) | **Terbuka** — `ADR-0024` |
| **Minta daftar peristiwa wajib audit dari Compliance** | Pengganti gerbang 1 untuk `S-5` | **Terbuka** — `ADR-0026` |
| **Konfirmasi Pega staging yang dapat ditembak dari luar** | Prasyarat `S-8` dan gerbang 1 seluruh modul | **Terbuka** — `ADR-0027` |
| **Tetapkan tujuan penyimpanan rahasia dan keputusan rotasi** | Menutup **R-17** | **Terbuka** — `D-40` |
| Sediakan lingkungan dev, staging, production | Pipeline deployment | Terbuka |
| Pelatihan tim: Go, React, TypeScript | Kesiapan tim (D-09) | Terbuka |

> Tahap ini **harus dimulai hari pertama** karena waktu tunggunya ada di luar kendali tim
> pengembang. Empat penghalang lama sudah tertutup; **enam penghalang baru** muncul dari
> verifikasi bukti, dan tiga di antaranya — kontrak HCC/HCQ, daftar peristiwa audit, dan Pega
> staging — **memblokir gerbang kelulusan**, bukan sekadar memperlambat pengerjaan.

### Tahap 1 — Fondasi
Modul F-1 sampai F-5, ditambah U-1 dan U-2 (kerangka SPA dan pustaka komponen).
**Belum ada perubahan bagi pengguna.** Pega masih menangani seluruh pekerjaan.
Keluaran: aplikasi Go yang bisa login, membaca database, dan menampilkan satu layar contoh.

### Tahap 2 — Baca dulu, tulis belakangan
Modul yang **hanya membaca** dialihkan lebih dulu: inbox, pencarian klaim, tampilan detail,
dan laporan.

Ini pilihan yang disengaja: modul baca **tidak melanggar P-1** karena tidak menulis apa pun,
sehingga bisa berjalan berdampingan dengan Pega tanpa risiko konflik data. Sekaligus menjadi
pembuktian arsitektur yang nyata — tim belajar Go dan React pada pekerjaan yang kesalahannya
tidak merusak data.

Bila hasil di Go berbeda dengan Pega pada layar yang sama, itu **bug yang harus diperbaiki
sebelum lanjut** — dan inilah cara paling murah menemukannya.

### Tahap 3 — Jalur klaim inti
`B-1 → B-2 → B-3 → B-4 → B-5`, ditambah B-6 (penugasan), B-14 (receive document), dan S-1
(dokumen).

Sejak titik ini, klaim baru bernomor `PNCN.YY.xxxx` dibuat di Go. Klaim `PNC-xxxx` yang sedang
berjalan tetap diselesaikan di Pega (P-3).

### Tahap 4 — Persetujuan dan penyelesaian
B-7 (komite), B-10 (akseptasi & pembayaran), B-11 (RCL/PUCL/compliance), B-8 (survey),
B-13 (open protection).

### Tahap 5 — Nilai dan pihak luar
B-9 (PLA/Pre-DLA/DLA), B-12 (salvage & recovery), S-3 (notifikasi), S-4 (integrasi eksternal).

### Tahap 6 — Laporan dan penutup
S-2 (56 laporan + engine dokumen), S-7 (dashboard), U-5, U-6, S-6 (scheduler).

### Tahap 7 — Penonaktifan Pega
Dilakukan hanya setelah **seluruh klaim `PNC-xxxx` yang masih berjalan selesai** atau
dipindahkan secara sadar. Pega dijadikan read-only lebih dulu selama satu periode pengamatan,
baru dimatikan.

---

## 3. Migrasi data

### 3.1 Data yang tidak dipindahkan
Data historis tetap di tabel `POOLDATA` yang sama — **satu database bersama** (D-21). Tidak ada
migrasi massal data klaim.

### 3.2 Data yang dipindahkan
Yang berpindah adalah data yang selama ini hidup di **tabel milik engine Pega**, ke tabel baru
milik aplikasi (D-21):

| Dari | Ke | Catatan |
|---|---|---|
| `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` | Tabel header klaim baru | Dibaca 116 rule; ini tabel klaim yang sebenarnya |
| `DATAPEGA.PC_ASSIGN_WORKLIST` | Tabel penugasan baru | Worklist (D-26) |
| `DATAPEGA.PC_ASSIGN_WORKBASKET` | Tabel antrean baru | Workbasket (D-26) |
| `DATAPEGA.PR_OPERATORS` | Tabel pengguna baru | Digabung dengan profil dari HCC/HCQ (D-07) |
| `DATAPEGA.PC_LINK_ATTACHMENT` + `PC_DATA_WORKATTACH` | Tabel metadata dokumen | Berkasnya ke storage internal (D-16) |

### 3.3 Penanganan kunci warisan
`CLAIMID = 'ASM-FW-GCNMFW-WORK ' || no_klaim` tetap ada pada data lama. Aturan:
- Data lama **dibaca apa adanya**, prefix dipangkas saat dibaca ke domain.
- Data baru **tidak pernah** menulis prefix (D-22).
- Satu fungsi tunggal menangani konversi ini — tidak tersebar di banyak query.

### 3.4 Migrasi Oracle → PostgreSQL (kelak)
Terpisah dari migrasi aplikasi dan dilakukan **setelah** aplikasi Go stabil di Oracle.

Karena SQL sudah portabel (D-20) dan tidak ada pemanggilan stored procedure (D-02), langkahnya:
1. Siapkan PostgreSQL 17+ (D-24) dengan skema yang setara.
2. Pindahkan data.
3. Ganti konfigurasi driver dan generator nomor klaim (satu-satunya sakelar dialek, D-22).
4. Jalankan seluruh test suite terhadap PostgreSQL.
5. Cutover.

**Inilah imbalan dari SQL portabel:** perpindahan database tidak menyentuh satu baris pun logika
bisnis.

---

## 4. Verifikasi kesetaraan

Karena P-5 menuntut perilaku identik, kita butuh cara membuktikannya — bukan sekadar berharap.

**Pelaksananya adalah modul `S-8`**, yang masuk **gelombang 1** karena memblokir gerbang 1 setiap
modul lain (`D-42`). Lingkungannya **Pega staging vs Go staging** atas salinan data produksi, dan
keluarannya wajib **mengklasifikasikan** setiap selisih terhadap 13 butir `P-5` — bukan sekadar
melaporkannya (`D-53`, `D-54`). Rinciannya di `14-TESTING-STRATEGY.md` §6.

| Cara | Kapan | Isi |
|---|---|---|
| **Perbandingan hasil baca** | Tahap 2 | Jalankan query yang sama di Pega dan Go, bandingkan hasilnya baris per baris pada data produksi (disalin ke staging) |
| **Uji kasus aturan bisnis** | Setiap modul | Setiap aturan di Business Understanding §3 punya test dengan kasus lolos dan kasus ditolak |
| **Uji paralel transaksi** | Tahap 3+ | Klaim contoh diproses di kedua sistem, hasil akhirnya dibandingkan |
| **Rekonsiliasi harian** | Selama masa paralel | Jumlah klaim, total nilai akseptasi, dan jumlah penugasan dibandingkan antar sistem |

---

## 5. Rencana mundur (rollback)

| Tingkat | Cara | Waktu pemulihan |
|---|---|---|
| Satu rilis bermasalah | Kembalikan versi binary sebelumnya di load balancer | Menit |
| Satu modul bermasalah | Arahkan kembali menu modul itu ke Pega; kepemilikan tabel dikembalikan | Jam |
| Kegagalan menyeluruh | Pega dijadikan sistem utama kembali; Go dimatikan | Jam |

**Syarat agar rollback benar-benar mungkin:** Pega **tidak boleh dinonaktifkan** sebelum Tahap 7.
Selama masa paralel, Pega harus tetap dapat menjalankan seluruh fungsinya. Ini biaya nyata dari
Strangler Fig — dan biaya itu adalah harga dari kemampuan mundur.

---

## 6. Catatan atas target waktu

Target yang ditetapkan manajemen adalah **seluruh modul selesai akhir September 2026** (D-30).
Migration Strategy ini disusun untuk target tersebut, dengan urutan yang meletakkan pekerjaan
berisiko rendah lebih dulu sehingga bila waktu tidak mencukupi, yang tertinggal adalah bagian
yang paling sedikit dampaknya.

Perbedaan antara target itu dan ukuran pekerjaan yang terukur didokumentasikan di **R-05**
(Risk Analysis) — bukan untuk menolak target, melainkan agar keputusan pemangkasan scope atau
penambahan sumber daya di kemudian hari punya dasar angka yang jelas.

**Tiga hal yang tidak bisa dipercepat oleh penambahan orang**, dan karena itu harus dimulai
sekarang juga:
1. Menunggu source 64 procedure & function dari DBA (**R-01**)
2. Menunggu 6 API pengganti DB Link dibangun tim lain (**R-03**)
3. Waktu belajar tim atas Go, React, dan TypeScript (D-09)

# 7. Strategi Teknis, Struktur Folder, dan Standar Coding

Dokumen paling preskriptif dalam Steering ini. Ketegasannya disengaja: D-09 menetapkan tim
adalah developer Pega yang belum terbiasa Go maupun JavaScript modern, dan D-23 memilih React
yang tidak opinionated. Gabungan keduanya punya satu mode kegagalan yang sangat mungkin terjadi:
**setiap layar ditulis dengan gaya berbeda oleh orang berbeda**, lalu tidak ada yang bisa
merawatnya.

Aturan di sini **mengikat**, bukan anjuran. Pelanggaran ditolak di code review.

---

## 1. Tumpukan teknologi

### Backend

| Bagian | Pilihan | Alasan |
|---|---|---|
| Bahasa | **Go 1.22+** | Ditetapkan pemilik project |
| Router HTTP | **`net/http` + `chi`** | `chi` tipis, idiomatik, tanpa konsep baru di luar `net/http` bawaan. Framework besar seperti Gin/Echo memperkenalkan konteks dan middleware sendiri — beban belajar tambahan yang tidak perlu |
| Akses database | **`database/sql` + driver** | SQL ditulis langsung (D-20). Tanpa ORM |
| Driver Oracle | **`godror`** | Paling matang untuk Oracle |
| Driver PostgreSQL | **`pgx`** (mode `database/sql`) | Standar de-facto |
| Migrasi skema | **`golang-migrate`** | Berkas SQL polos, mudah dibaca dan di-review |
| Logging | **`log/slog`** (pustaka standar) | Terstruktur, tanpa dependensi tambahan |
| Konfigurasi | **Variabel lingkungan + berkas YAML** | — |
| Validasi | **Kode eksplisit di modul domain** | Bukan tag struct. Aturan bisnis harus terbaca sebagai kalimat, bukan tersembunyi di anotasi |
| Pengujian | **`testing` bawaan + `testify/require`** | Tanpa framework BDD |
| PDF | **`maroto`** atau **`gopdf`** | D-11 |
| Excel | **`excelize`** | D-11 |
| Penjadwalan | **`robfig/cron`** | Sederhana, cukup untuk kebutuhan S-6 |

> **Prinsip pemilihan pustaka:** utamakan pustaka standar. Setiap dependensi pihak ketiga adalah
> satu hal lagi yang harus dipelajari tim, dipantau keamanannya, dan bisa ditinggalkan
> pemeliharanya.

### Frontend

| Bagian | Pilihan | Alasan |
|---|---|---|
| Framework | **React 18 + TypeScript** | D-23 |
| Build | **Vite** | Cepat, konfigurasi sederhana |
| Routing | **React Router** | Standar de-facto |
| Data server | **TanStack Query** | Menyeragamkan pengambilan data, cache, dan status loading. **Wajib** — inilah yang mencegah 74 layar punya cara berbeda memanggil API |
| Tabel | **TanStack Table** atau **AG Grid** | D-23. Satu pilihan saja, tidak keduanya |
| Form | **React Hook Form + Zod** | Validasi terketik, sinkron dengan kontrak API |
| State global | **Zustand** | Hanya untuk state yang benar-benar global (pengguna, izin). Selebihnya state server ditangani TanStack Query |
| Styling | **Tailwind CSS** | Konsisten tanpa berdebat penamaan kelas CSS |
| Pengujian | **Vitest + Testing Library** | — |

> **Yang dilarang di frontend:** memanggil `fetch` langsung di komponen (harus lewat TanStack
> Query), menyimpan data server di Zustand, dan membuat komponen tabel baru di luar pustaka
> komponen baku (U-2).

---

## 2. Struktur folder — Backend

```
claim-pnc/
├── cmd/
│   └── server/                  titik masuk aplikasi — sangat tipis
│
├── internal/                    seluruh kode aplikasi; tidak bisa diimpor project lain
│   │
│   ├── domain/                  ── LAPISAN DOMAIN ──
│   │   │                        aturan bisnis murni
│   │   │                        DILARANG mengimpor: HTTP, SQL, driver, JSON wire
│   │   ├── klaim/
│   │   │   ├── klaim.go             tipe agregat & invarian
│   │   │   ├── registrasi.go        modul Registrasi Klaim
│   │   │   ├── validasi_tanggal.go  aturan tanggal (Business Understanding §3.1)
│   │   │   ├── validasi_duplikat.go aturan duplikasi (§3.2)
│   │   │   ├── status.go            empat konsep status (D-18)
│   │   │   ├── errors.go            kesalahan domain
│   │   │   └── repository.go        SEAM — interface, bukan implementasi
│   │   ├── polis/                   snapshot polis (D-04)
│   │   ├── objek/                   objek pertanggungan & coverage
│   │   ├── spreading/               aturan 100%, Fac Out, Ex-Gratia
│   │   ├── settlement/              estimasi → usulan → akseptasi → bayar
│   │   ├── komite/                  penjenjangan 1–4 level
│   │   ├── penugasan/               worklist & workbasket (D-26)
│   │   ├── survei/
│   │   ├── reasuransi/              PLA · Pre-DLA · DLA
│   │   ├── salvage/
│   │   ├── dokumen/
│   │   ├── otorisasi/               izin menu & aksi (D-07)
│   │   ├── masterdata/
│   │   └── audit/                   jejak audit append-only (D-28)
│   │
│   ├── app/                     ── LAPISAN APLIKASI ──
│   │   │                        orkestrasi lintas modul domain + transaksi
│   │   ├── registrasiklaim/
│   │   ├── prosesakseptasi/
│   │   └── ...
│   │
│   ├── adapter/                 ── LAPISAN ADAPTER ──
│   │   ├── sqlstore/                implementasi Repository (SQL portabel)
│   │   │   ├── klaim.go
│   │   │   ├── klaim.sql            ← query terpisah dari kode Go
│   │   │   ├── penugasan.go
│   │   │   ├── penugasan.sql
│   │   │   └── nomor_klaim.go       ← SATU-SATUNYA sakelar dialek (D-22)
│   │   ├── hccclient/               autentikasi HCC/HCQ (D-07)
│   │   ├── docstore/                storage dokumen internal (D-16)
│   │   ├── brisurf/                 integrasi BRI Surf
│   │   ├── kasir/
│   │   ├── slikojk/
│   │   ├── smtp/                    adapter Notifier
│   │   └── clock/                   adapter Clock (F-5)
│   │
│   ├── transport/               ── LAPISAN TRANSPORT ──
│   │   └── http/
│   │       ├── handler/             satu berkas per sumber daya
│   │       ├── middleware/          auth · logging · recovery · request ID
│   │       ├── dto/                 bentuk request & response — TERPISAH dari tipe domain
│   │       └── router.go
│   │
│   ├── report/                      engine PDF · Excel · CSV (D-11)
│   ├── scheduler/                   job terjadwal (S-6)
│   └── platform/
│       ├── config/
│       ├── logging/
│       ├── database/                koneksi & connection pool
│       └── errs/                    tipe kesalahan bersama
│
├── migrations/                      berkas SQL golang-migrate
├── web/                             hasil build SPA React (disematkan ke binary)
├── docs/
└── Steering/                        dokumen ini
```

### Aturan struktur yang mengikat

1. **`domain/` tidak boleh mengimpor `adapter/`, `transport/`, maupun pustaka database.**
   Ditegakkan otomatis oleh linter (`depguard`), bukan hanya oleh kesepakatan.
2. **Interface dideklarasikan di paket yang memakainya**, bukan di paket yang mengimplementasikannya.
   `domain/klaim/repository.go` mendeklarasikan apa yang dibutuhkan Klaim; `adapter/sqlstore`
   memenuhinya. Inilah yang membuat seam berada di tempat yang benar.
3. **Satu berkas = satu tanggung jawab.** Berkas melebihi ~400 baris adalah tanda modul perlu
   dipecah.
4. **DTO transport tidak sama dengan tipe domain.** Memakai tipe domain langsung sebagai bentuk
   JSON membuat perubahan internal bocor ke klien dan sebaliknya.
5. **SQL berada di berkas `.sql` terpisah**, bukan sebagai string di tengah kode Go. Query bisa
   dibaca, di-review, dan diuji langsung terhadap database.

---

## 3. Struktur folder — Frontend

```
web/src/
├── app/                     kerangka: router, provider, layout, guard
├── shared/
│   ├── api/                 klien HTTP + tipe hasil generate dari kontrak API
│   ├── components/          ── PUSTAKA KOMPONEN BAKU (U-2) ──
│   │   ├── DataTable/           satu-satunya tabel di seluruh aplikasi
│   │   ├── Form/                field, label, pesan kesalahan
│   │   ├── FileUpload/
│   │   ├── DatePicker/
│   │   └── ...
│   ├── hooks/
│   └── lib/                 format tanggal, angka, mata uang — SATU tempat
│
└── features/                satu folder per modul bisnis
    ├── registrasi-klaim/
    │   ├── api/                 hook TanStack Query untuk fitur ini
    │   ├── components/          komponen khusus fitur ini
    │   ├── pages/
    │   └── types.ts
    ├── inbox/
    ├── komite/
    ├── akseptasi/
    ├── survei/
    ├── laporan/
    └── master-data/
```

**Aturan mengikat:**
1. **Fitur tidak boleh mengimpor dari fitur lain.** Kebutuhan bersama naik ke `shared/`.
2. **Semua tabel memakai `shared/components/DataTable`.** Tidak ada `<table>` mentah di
   folder `features/`. Ini yang mengubah 268 grid menjadi satu implementasi.
3. **Semua pemanggilan API lewat hook TanStack Query** di `features/*/api/`. Tidak ada `fetch`
   di dalam komponen.
4. **Semua format tanggal, angka, dan mata uang lewat `shared/lib`.** Bukan format lokal per
   komponen — inilah yang mencegah terulangnya masalah `TO_CHAR` tersebar (utang teknis 4.4).

---

## 4. Coding Standards — Backend

### 4.1 Penamaan

| Hal | Aturan | Contoh |
|---|---|---|
| Paket | Kata benda tunggal, huruf kecil, tanpa garis bawah | `klaim`, `spreading`, `settlement` |
| Berkas | `snake_case.go` | `validasi_tanggal.go` |
| Tipe & fungsi ekspor | `PascalCase` | `Klaim`, `RegistrasiKlaim` |
| Interface | Nama peran, bukan berakhiran `Interface` | `KlaimRepository`, bukan `IKlaimRepository` |
| Istilah domain | **Ikuti `CONTEXT.md` tanpa perkecualian** | `ObjekPertanggungan`, bukan `Object` · `SettlementLine`, bukan `Adjustment` |

**Bahasa penamaan:** istilah domain memakai **bahasa Indonesia** sesuai `CONTEXT.md`, karena
itulah bahasa yang dipakai bisnis dan tim. Istilah teknis memakai **bahasa Inggris** mengikuti
konvensi Go. Contoh: `type Klaim struct` dengan method `Validate()`.

Alasannya: mencampur istilah domain berbahasa Inggris yang salah terjemah (seperti `Adjustment`
yang ternyata berarti nilai penyelesaian) adalah tepat sumber kekacauan yang sedang kita
perbaiki.

### 4.2 Penanganan kesalahan

- Kesalahan selalu dikembalikan, **tidak pernah** `panic` di jalur normal.
- Setiap kesalahan **dibungkus dengan konteks** saat naik ke pemanggil, sehingga jejaknya
  terbaca dari pesan.
- Kesalahan domain adalah **tipe tersendiri**, bukan string. Transport memetakannya ke kode HTTP.
- Kesalahan validasi bisnis **dikumpulkan seluruhnya**, tidak berhenti pada yang pertama.
  Ini meniru perilaku Pega yang menampilkan semua pesan sekaligus — mengubahnya jadi
  satu-per-satu akan sangat menyiksa pengguna pada form registrasi yang panjang.
- Kesalahan **tidak pernah** ditelan diam-diam. Tidak ada `_ = err`.

### 4.3 Aturan SQL

Ini yang membuat D-20 bisa dijalankan.

| Aturan | Alasan |
|---|---|
| **Selalu parameter binding**, tidak pernah merangkai SQL dari string | Menutup celah SQL injection yang ada di pola `{ASIS:...}` warisan (utang teknis 4.5) |
| `COALESCE`, bukan `NVL` | Portabilitas |
| `CURRENT_TIMESTAMP`, bukan `SYSDATE` | Portabilitas |
| `CASE WHEN`, bukan `DECODE` | Portabilitas |
| `OFFSET … FETCH NEXT … ROWS ONLY`, bukan `ROWNUM` | Portabilitas; pola ini sudah dipakai di 35 rule lama |
| `POSITION`, bukan `INSTR` | Portabilitas |
| `STRING_AGG`, bukan `LISTAGG` | Portabilitas |
| `LEFT JOIN`, bukan `(+)` | Portabilitas |
| Hilangkan `FROM DUAL` | Portabilitas |
| **Tanpa `TO_CHAR` untuk pemformatan tampilan** | Format tanggal dan angka dilakukan di Go |
| **Tanpa pemanggilan stored procedure** | D-02 |
| `JSON_VALUE` / `JSON_TABLE` boleh dipakai | Portabel bila PostgreSQL 17+ (D-24) |
| Satu-satunya sakelar dialek: generator nomor klaim | D-22 |

**Kolom yang dipilih harus disebutkan namanya.** `SELECT *` dilarang — kolom baru di database
tidak boleh diam-diam mengubah perilaku aplikasi.

### 4.4 Waktu

- Disimpan **UTC** di database.
- Ditampilkan **WIB (Asia/Jakarta)**.
- Konversi hanya di modul Clock (F-5).
- **Tidak ada penambahan 7 jam manual di mana pun.** Ini pelanggaran yang otomatis ditolak
  review.
- Aturan berbasis hari kalender dihitung terhadap tanggal WIB.

### 4.5 Transaksi

- Transaksi dimulai dan diakhiri di **lapisan aplikasi** (`internal/app/`), bukan di dalam
  repository maupun handler.
- Satu permintaan pengguna = satu transaksi, kecuali ada alasan yang didokumentasikan.
- Pemanggilan sistem eksternal **tidak boleh berada di dalam transaksi database** — kegagalan
  jaringan tidak boleh menahan kunci baris.

**Kepemilikan transaksi berpindah sepenuhnya ke Go** (`D-68`). Ini bukan penegasan gaya kode
melainkan perbaikan nyata: **10 dari 12 procedure** yang dibaca melakukan `COMMIT` sendiri —
`Database/INSERT_PLADLA.prc` **sembilan kali** (`:69`, `:74`, `:79`, `:138`, `:143`, `:148`,
`:179`, `:184`, `:189`), dengan satu-satunya `ROLLBACK` di handler terluar (`:198`) yang
**terjadi setelah commit** sehingga tidak memulihkan apa pun.

| Yang ini lepaskan | Isi |
|---|---|
| **`B-4` Spreading dan `B-9` PLA/DLA dapat dibuat atomik** | penerbitan yang dulu menempuh sembilan `COMMIT` dan bisa berhenti setengah jalan kini dibungkus satu transaksi |
| **Kontrak galat berbasis string `ErrMsg` tidak dibawa** | pada enam procedure `ErrMsg` **tidak di-set pada jalur sukses** sehingga `NULL` berarti berhasil; pada `ADD_NEWMASTERVIRTUALACCOUNT.prc:18` kolom yang sama membawa **nomor virtual account sekaligus pesan galat** |
| **Procedure boleh ditinggalkan** | Claim PNC adalah satu-satunya pemanggilnya (`D-68`) |

> **Konsekuensi untuk uji kesetaraan:** perubahan dari sembilan commit menjadi satu transaksi
> **mengubah perilaku saat gagal** — sistem lama meninggalkan sebagian data, sistem baru tidak
> meninggalkan apa pun. Kasus uji harus dirancang menyadari ini, atau ia akan melaporkan selisih
> palsu.
>
> **Yang belum diverifikasi:** klaim "Claim PNC satu-satunya pemanggil" **belum dibuktikan dengan
> kueri katalog**. Satu kueri `ALL_DEPENDENCIES` cukup, dan harus dijalankan **sebelum** procedure
> benar-benar dinonaktifkan — bukan sebelum logikanya ditulis ulang.

### 4.6 Yang dilarang

| Dilarang | Alasan |
|---|---|
| Variabel global yang bisa diubah | Menghancurkan kemampuan uji dan aman-konkuren |
| Nilai bisnis di-hardcode | D-15 |
| Komentar `// TESTING` di jalur produksi | Utang teknis 4.3 tidak boleh terulang |
| Kredensial di dalam kode | Keamanan |
| Fungsi melebihi ~80 baris | Tanda ada modul yang belum dipisahkan |
| `interface{}` / `any` tanpa alasan tertulis | Menghilangkan manfaat tipe |
| Membaca konteks pengguna dari variabel global | Harus lewat `context.Context` |

---

## 5. Coding Standards — Frontend

| Aturan | Alasan |
|---|---|
| **TypeScript mode ketat**, `any` dilarang tanpa alasan tertulis | 74 layar dikerjakan tim yang sedang belajar — tipe adalah jaring pengaman |
| Tipe API **dihasilkan dari kontrak backend**, tidak ditulis tangan | Backend dan frontend tidak boleh berbeda persepsi |
| Komponen adalah fungsi, tanpa class component | Satu cara saja |
| Seluruh data server lewat **TanStack Query** | Menyeragamkan loading, error, cache, dan refetch di 74 layar |
| Seluruh tabel lewat **`shared/components/DataTable`** | 268 grid menjadi satu implementasi |
| Seluruh form lewat **React Hook Form + Zod** | Validasi seragam dan terketik |
| Paginasi **selalu server-side** | D-10: puluhan juta baris |
| Berkas komponen melebihi ~200 baris harus dipecah | Keterbacaan |
| Tanpa CSS inline; memakai Tailwind | Konsistensi |

---

## 6. Penegakan otomatis

Aturan yang hanya ada di dokumen akan dilanggar. Yang berikut **wajib berjalan di CI dan
memblokir merge**:

| Alat | Menegakkan |
|---|---|
| `gofmt` + `goimports` | Format Go |
| `golangci-lint` | Kualitas kode Go |
| `depguard` | **Aturan ketergantungan antar lapisan** (§2 aturan 1) |
| `go vet` | Kesalahan umum |
| `go test ./...` | Seluruh test lulus |
| ESLint + `@typescript-eslint` | Kualitas kode frontend |
| `tsc --noEmit` | Tidak ada kesalahan tipe |
| Prettier | Format frontend |
| Pemeriksaan pola SQL terlarang | `SELECT *`, `NVL`, `ROWNUM`, `SYSDATE`, `TO_CHAR`, perangkaian string SQL |

> Pemeriksaan pola SQL terlarang tampak sepele, tapi inilah yang benar-benar menjaga D-20 tetap
> berlaku setelah bulan ketiga — ketika tekanan jadwal membuat orang menempuh jalan pintas.

# 8. Strategi Database

Mengacu pada **PostgreSQL 17+ sebagai desain kanonikal** (D-01, D-24), berjalan di **Oracle 19c
untuk sementara** dengan **satu set SQL portabel** (D-20), **tanpa pemanggilan stored procedure**
(D-02).

> **Diperbarui v2.0 (2026-09-14).** Yang berubah di bab ini: format nomor klaim menjadi
> **`PNCN.YY.xxxx`** (`D-71`) · **soft delete menyeluruh** ditambahkan sebagai §8.1 (`D-66`) ·
> retensi jejak audit **mengikuti retensi data klaim** (`D-62`) · prosedur perubahan skema dan
> aturan penulis tunggal per tabel ditetapkan sebagai §9.1 dan §9.2 (`D-63`, `P-1`) · premis
> `OFFSET 500000` **dikoreksi** karena tidak berdasar · dan **stored procedure boleh ditinggalkan**
> setelah logikanya naik ke Go (`D-68`).

---

## 1. Aturan dasar

| # | Aturan | Sumber |
|---|---|---|
| DB-1 | Desain skema, tipe data, indexing, dan gaya SQL mengacu PostgreSQL | D-01 |
| DB-2 | Tidak ada pemanggilan stored procedure dari aplikasi | D-02 |
| DB-3 | Satu set SQL yang berjalan di Oracle 19c dan PostgreSQL 17+ | D-20 |
| DB-4 | Target PostgreSQL **17 atau lebih baru** — persyaratan mengikat | D-24 |
| DB-5 | Satu database bersama dengan Pega selama masa paralel | D-21 |
| DB-6 | Satu tabel hanya boleh ditulis satu sistem | D-21 |
| DB-7 | Migrasi skema selalu backward-compatible | D-27 |
| DB-8 | Seluruh waktu disimpan UTC | F-5 |
| DB-9 | Jejak audit append-only, tidak dapat diubah aplikasi | D-28 |

---

## 2. Kenapa PostgreSQL 17 wajib

Ini bukan preferensi versi, melainkan syarat agar D-20 bisa dijalankan sama sekali.

Sistem lama memakai **222 pemanggilan SQL/JSON** di 29 rule terhadap `JSON_KLAIM.DATA_JSONBLOB`
dan `JSON_POLIS.DATA_JSONBLOB`:

| Fungsi | Volume | Oracle 12c+ | PostgreSQL ≤16 | PostgreSQL 17+ |
|---|---|---|---|---|
| `JSON_VALUE` | 195× | ✅ | ❌ | ✅ |
| `JSON_QUERY` | (termasuk di atas) | ✅ | ❌ | ✅ |
| `JSON_TABLE` | 27× | ✅ | ❌ | ✅ |

PostgreSQL 17 mengimplementasikan fungsi SQL/JSON standar dengan sintaks yang sama seperti
Oracle. Dengan itu, seluruh query JSON portabel apa adanya.

Pada PostgreSQL 16 ke bawah, ke-29 rule harus ditulis ulang memakai operator `jsonb` khas
PostgreSQL (`->`, `->>`, `jsonb_path_query`), sehingga akan ada **dua versi query** — dan
keputusan "satu set SQL portabel" gugur.

---

## 3. Tiga pengecualian portabilitas

Diakui secara sadar, dikelola, dan tidak boleh bertambah tanpa keputusan tertulis.

### 3.1 Generator nomor klaim

Format nomor klaim sistem baru adalah **`PNCN.YY.xxxx`** — tiga segmen dipisahkan titik
(`D-71`, menyupersede bentuk `PNCN-xxxx` pada `D-22`). Sintaks Oracle yang ditetapkan:

```sql
'PNCN' || '.' || TO_CHAR(SYSDATE,'RR') || '.' || TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)
```

| Segmen | Isi | Sumber |
|---|---|---|
| `PNCN` | penanda tetap asal sistem baru | literal |
| `YY` | dua digit tahun | `TO_CHAR(SYSDATE,'RR')` |
| `xxxx` | nomor urut | `TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)` |

**Sequence-nya belum ada** — `POOLDATA.CLAIM_NO_NONPEGA_SEQ` nol kemunculan di seluruh export; ia
akan dibuat, bukan dipakai ulang.

Sejak `D-71`, sakelar dialek di sini membungkus **dua** perbedaan sekaligus, bukan satu: sequence
(Oracle `SEQ.NEXTVAL` versus PostgreSQL `nextval('seq')`) **dan** pemformatan tahun (`TO_CHAR(SYSDATE,'RR')`
versus `to_char(current_date,'YY')`). **Diisolasi di satu berkas**:
`internal/adapter/sqlstore/nomor_klaim.go`. Ini satu-satunya tempat dengan percabangan dialek di
seluruh aplikasi.

**Dua hal yang mengikuti dari sintaks ini dan belum diputuskan** (`ADR-0009`):

1. **Apakah sequence direset tiap awal tahun?** Bila tidak, nomor urut menembus pergantian tahun
   (`PNCN.26.8125` → `PNCN.27.8126`) dan segmen tahun menjadi penanda, bukan penghitung per tahun.
2. **`TO_CHAR(...NEXTVAL)` tanpa format mask tidak memberi nol di depan**, sehingga lebar segmen
   terakhir berubah-ubah dan **pengurutan sebagai teks tidak sesuai urutan penerbitan** (`.10`
   mendahului `.9`). Setiap layar dan laporan yang mengurutkan berdasarkan nomor klaim harus
   menyadarinya.

Catatan ketiga: **tahun diambil dari `SYSDATE`**, yaitu tanggal server basis data — bukan tanggal
kejadian dan bukan tanggal registrasi. Klaim yang terbit di sekitar pergantian tahun mengambil
tahun dari jam server, bertaut dengan `R-12`.

### 3.2 Pemformatan tanggal dan angka
411 pemakaian `TO_CHAR` **dihapus dari SQL**, pemformatan pindah ke Go.

Ini bukan sekadar demi portabilitas. Pola sekarang mengembalikan tanggal sebagai string
`'dd/mm/yyyy'` dari database, sehingga:
- pengurutan tanggal menjadi pengurutan **teks** — `01/12/2024` dianggap lebih kecil dari
  `02/01/2020`;
- penyaringan rentang tanggal tidak bisa memakai index;
- perbandingan tanggal salah tanpa ada yang menyadari.

Memindahkan pemformatan ke Go memperbaiki ketiganya sekaligus.

### 3.3 Paginasi
68 pemakaian `ROWNUM` diganti `OFFSET … FETCH NEXT … ROWS ONLY`, yang didukung Oracle 12c+ dan
PostgreSQL. Pola ini **sudah dipakai di 35 rule** pada sistem lama, jadi bukan hal baru bagi tim.

---

## 4. Padanan sintaks yang mengikat

| Jangan pakai | Pakai | Volume di sistem lama |
|---|---|---|
| `NVL(a, b)` | `COALESCE(a, b)` | 100× |
| `SYSDATE` | `CURRENT_TIMESTAMP` | 69× |
| `DECODE(...)` | `CASE WHEN … END` | 16× |
| `ROWNUM` | `OFFSET … FETCH NEXT … ROWS ONLY` | 68× |
| `INSTR(a, b)` | `POSITION(b IN a)` | 11× |
| `LISTAGG(...)` | `STRING_AGG(...)` | 4× |
| `a = b(+)` | `LEFT JOIN` | 2× |
| `FROM DUAL` | hilangkan klausa `FROM` | 12× |
| `ADD_MONTHS(d, n)` | `d + INTERVAL` | 18× |
| `MONTHS_BETWEEN(a, b)` | hitung di Go | 2× |
| `TRUNC(date)` | `CAST(x AS DATE)` | 150× |
| `TO_CHAR(...)` untuk tampilan | format di Go | 411× |
| `SELECT *` | sebutkan nama kolom | — |
| Perangkaian string SQL (`{ASIS:...}`) | parameter binding | — |

---

## 5. Tipe data

Dipilih agar sama-sama sah di Oracle dan PostgreSQL, dengan PostgreSQL sebagai acuan.

| Kegunaan | PostgreSQL | Oracle | Catatan |
|---|---|---|---|
| Teks pendek | `VARCHAR(n)` | `VARCHAR2(n)` | Panjang eksplisit |
| Teks panjang | `TEXT` | `CLOB` | Kronologi, catatan |
| Bilangan bulat | `BIGINT` | `NUMBER(19)` | — |
| **Nilai uang** | `NUMERIC(18,2)` | `NUMBER(18,2)` | **Tidak pernah** `float`/`double` |
| Persentase share | `NUMERIC(9,6)` | `NUMBER(9,6)` | Presisi cukup untuk aturan 100% |
| Waktu | `TIMESTAMPTZ` | `TIMESTAMP WITH TIME ZONE` | Disimpan UTC (DB-8) |
| Tanggal murni | `DATE` | `DATE` | DOL, tanggal lapor |
| Boolean | `BOOLEAN` | `NUMBER(1)` | Dipetakan di adapter |
| Dokumen JSON | `JSONB` | `CLOB` + `IS JSON` | Snapshot polis |

**Aturan nilai uang tidak bisa ditawar.** Sistem menghitung pembagian share reasuransi dengan
aturan total harus 100%, **toleransi 4 desimal, `99,9999`–`100,0001`** (`D-51`). Pembulatan floating point akan membuat validasi ini
gagal secara acak dan tidak dapat direproduksi.

---

## 6. Strategi indexing

Profil beban: **data besar, konkurensi rendah** (D-10). Optimasi diarahkan ke volume.

### 6.1 Index wajib

| Tabel | Index | Untuk |
|---|---|---|
| Klaim | `nomor_klaim` (unik) | Pencarian utama |
| Klaim | `nomor_polis, prod_ke` | Pencarian per polis, cek duplikat |
| Klaim | `tanggal_kejadian` | Filter dan laporan periode |
| Klaim | `status_klaim, group_panel` | Filter inbox |
| Klaim | `kode_cabang` | Batas visibilitas per cabang |
| Klaim | `(status_proses, dibuat_pada DESC)` | Urutan inbox |
| Penugasan | `(ditugaskan_ke, status)` | Worklist per orang |
| Penugasan | `(workbasket, status)` | Antrean bersama |
| ObjekPertanggungan | `klaim_id` | Muat anak |
| Coverage | `objek_id` | Muat anak |
| SettlementLine | `coverage_id` | Muat anak |
| SettlementLine | `nomor_akseptasi` | Pencarian akseptasi |
| JejakAudit | `(entitas, entitas_id, terjadi_pada DESC)` | Penelusuran riwayat |

### 6.2 Index parsial (PostgreSQL)
Inbox hanya menampilkan klaim yang belum selesai, sedangkan mayoritas baris adalah klaim lama
yang sudah selesai. Index parsial atas klaim yang masih berjalan membuat ukurannya tetap kecil
walau tabel berisi puluhan juta baris.

Oracle tidak punya index parsial — padanannya adalah function-based index. Karena ini urusan
DDL dan bukan query, perbedaannya **tidak melanggar D-20**.

### 6.3 Paginasi
- Inbox dan pencarian: **keyset pagination** (`WHERE (kolom_urut, id) < (nilai, id)`), bukan
  `OFFSET` besar. Pada puluhan juta baris, `OFFSET` bernilai besar memaksa database membaca dan
  membuang seluruh baris sebelum halaman yang diminta.
- `OFFSET … FETCH` hanya untuk halaman-halaman awal atau data yang sudah tersaring sempit.

> **Koreksi premis (2026-09-14).** Angka `OFFSET 500000` yang dipakai dokumen versi sebelumnya
> **tidak berdasar**: `OFFSET` **nol kemunculan** di seluruh export. Masalah nyatanya berbeda dan
> lebih berat — **3.189 grid terikat page list klipboard** Pega, dan `pyMaxRecords=500` pada
> **54 dari 56** laporan. Artinya sistem lama tidak memaginasi hasil besar sama sekali; ia
> **memotongnya di 500 baris**.
>
> Konsekuensinya bagi uji kesetaraan: **paginasi keyset adalah perubahan perilaku, bukan
> pemeliharaan.** Grid yang hari ini memuat seluruh page list akan berperilaku berbeda saat
> dipaginasi di server, dan laporan yang dulu terpotong kini utuh. Keduanya harus diuji per layar,
> bukan diasumsikan setara.

### 6.4 Partisi
Disiapkan tapi **belum diterapkan** di awal.

Tabel Klaim dan JejakAudit adalah kandidat partisi per tahun berdasarkan tanggal registrasi.
Diterapkan hanya bila pengukuran nyata menunjukkan kebutuhannya — bukan di awal. Menerapkan
partisi tanpa data pengukuran adalah menambah kerumitan tanpa bukti manfaat.

---

## 7. Connection pool

Untuk 200–300 pengguna aktif harian dengan dua instance aplikasi (D-27):

| Parameter | Nilai awal | Alasan |
|---|---|---|
| Koneksi maksimum per instance | **20** | 2 instance × 20 = 40 koneksi. Cukup untuk beban ini; koneksi berlebih justru membebani database |
| Koneksi idle | 5 | Menghindari biaya pembukaan koneksi berulang |
| Umur maksimum koneksi | 30 menit | Mencegah koneksi basi di balik firewall/load balancer |
| Idle maksimum | 5 menit | — |
| Batas waktu query | 30 detik (transaksi), 5 menit (laporan) | Laporan besar tidak boleh menahan pool transaksi |

**Pool terpisah untuk laporan.** Query laporan berjalan lama dan bervolume besar. Bila memakai
pool yang sama, satu laporan berat bisa menghabiskan seluruh koneksi dan membuat pengguna lain
tidak bisa bertransaksi. Pool laporan diberi batas koneksi lebih kecil dan batas waktu lebih
panjang.

---

## 8. Jejak audit

Wajib per D-28. Dirancang sejak awal, bukan ditambal.

**Isi setiap baris:** entitas, id entitas, jenis aksi, nilai sebelum, nilai sesudah, pelaku,
waktu (UTC), id permintaan.

**Yang wajib diaudit:** nilai estimasi klaim, nilai settlement pada setiap tahap, akseptasi dan
nomornya, keputusan komite, penolakan (RCL), proses ulang (PUCL), perubahan status klaim, dan
pembayaran.

**Sifat append-only ditegakkan di database**, bukan hanya di kode: akun aplikasi hanya diberi
hak `INSERT` dan `SELECT` pada tabel audit — tanpa `UPDATE` maupun `DELETE`. Aturan yang hanya
ada di kode bisa dilanggar oleh kode berikutnya; aturan yang ada di hak akses database tidak.

**Retensi:** **mengikuti retensi data klaim yang berlaku sekarang** (`D-62`) — satu kebijakan
untuk keduanya, bukan kebijakan terpisah. Ini menutup pertanyaan terbuka `D-28`: kebijakannya
sudah ada, yang dibutuhkan hanyalah **angkanya**. Sampai angka itu masuk, `S-5` dibangun dengan
**retensi sebagai parameter konfigurasi**, sehingga modulnya tidak terhalang.

**Kenapa bab ini naik derajat.** `D-59` menetapkan satuan izin adalah menu dan **tidak ada
pemisahan tugas** — satu orang dapat membuat, menyetujui, dan membayarkan satu klaim bila perannya
memiliki ketiga menu itu. Tidak ada kontrol teknis yang mencegahnya, sehingga **jejak audit
menjadi satu-satunya kontrol pengimbang yang tersisa** (`ADR-0023`, `ADR-0026`).

**Dua anti-pola dari sistem lama yang tidak dibawa**, keduanya terbukti di source: `UPDATE` pada
`pooldata.claim_service_log` dan `DELETE` pada `POOLDATA.JSON_KLAIM_LOG`. Log yang dapat diubah
dan dihapus bukan log.

---

## 8.1 Penghapusan data — soft delete menyeluruh

`D-66` menetapkan **tidak ada `DELETE` fisik pada data bernilai bisnis**. Penghapusan dinyatakan
lewat penanda — kolom flag beserta waktu dan pelakunya — bukan lewat pembuangan baris. Dengan
begitu, jejak audit append-only dan kebijakan penghapusan berdiri di atas prinsip yang sama.

**Konsekuensi yang mengikat desain:**

| Hal | Ketetapan |
|---|---|
| Setiap kueri pembaca | **wajib menyaring baris bertanda terhapus**. Satu kueri yang lupa akan menampilkan data yang seharusnya hilang — kelas cacat baru yang tidak ada di sistem lama |
| Keunikan kunci alami | baris yang "terhapus" **masih menempati nilai kuncinya**; constraint unik harus memperhitungkan penanda |
| Indexing dan partisi | dirancang menyadari adanya **baris mati** di atas data historis puluhan juta baris |
| Uji kesetaraan | **tidak boleh** membandingkan `COUNT(*)` tabel — lihat Testing Strategy §6.4 |

**Satu pola lama yang gugur karenanya.** `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503`
memakai **hapus-lalu-sisip-ulang** pada **12 tabel** sebagai mekanisme idempotensi konversi klaim.
Pola itu bergantung pada penghapusan fisik dan **tidak dapat dipertahankan**. Penggantinya —
*upsert* berbasis kunci alami, versioning, atau melarang konversi ulang — **belum diputuskan**
(`ADR-0013`, berstatus `Proposed`).

Catatan pendukung: dua tabel pada blok itu (`T_DLALIST`, `T_PLALIST`) **delete-nya sudah
dikomentari** di sumber (`:497`, `:498`) sementara insert-nya tetap aktif (`:1296`, `:1325`) —
sehingga konversi ulang pada kedua tabel itu **sudah berpotensi menduplikasi baris hari ini**.

---

## 9. Migrasi skema

- Dikelola `golang-migrate` dengan berkas SQL polos yang bisa di-review.
- **Selalu backward-compatible** (DB-7) karena rolling deployment 24/7.
- Penamaan: `NNNN_deskripsi_singkat.up.sql` dan `.down.sql`.
- Setiap migrasi wajib punya `down` yang benar-benar berfungsi.
- Migrasi dijalankan **terpisah dari start aplikasi**, sebagai langkah deployment tersendiri.
  Menjalankannya saat start akan menyebabkan dua instance mencoba bermigrasi bersamaan.

**Urutan untuk perubahan yang tidak kompatibel:**
1. Rilis N: tambah struktur baru, tulis ke lama dan baru, baca dari lama.
2. Rilis N+1: baca dari baru.
3. Rilis N+2: berhenti menulis ke lama.
4. Rilis N+3: hapus struktur lama.

### 9.1 Siapa yang menjalankan perubahan skema

`D-63` menetapkan prosedurnya, dan ia **menempuh tiga pihak**:

| Langkah | Pelaku |
|---|---|
| Permintaan **tertulis** | tim pengembang |
| Persetujuan | **Work Owner** |
| Pelaksanaan | **DBA** |
| Verifikasi | **wajib diuji dengan menjalankan Pega dan Go bersamaan** terhadap skema hasil perubahan |

**Alasannya bukan birokrasi.** `P-4` mewajibkan verifikasi backward-compatible dengan menjalankan
versi lama dan baru bersamaan — itu tidak dapat dilakukan DBA sendirian maupun tim pengembang
sendirian. Dan karena `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` dibaca **116 rule Pega**, satu `ALTER` yang
keliru menghentikan sistem yang sedang melayani produksi.

**Konsekuensi yang diterima:** iterasi melambat selama seluruh masa paralel. Setiap tiket yang
menyentuh skema wajib memuat bagian rollback yang **tidak kosong**.

### 9.2 Penulis tunggal per tabel selama masa paralel

Selama Pega dan Go berjalan bersamaan di atas satu database (`D-21`, `ADR-0004`), berlaku `P-1`:
**untuk setiap tabel, tepat satu sistem berwenang menulis**; yang lain hanya membaca. Kewenangan
berpindah saat modul pemiliknya lulus gerbang 2 — bukan sebelum itu.

> **Aturan ini tidak ditegakkan mesin mana pun.** Ia disiplin manusia, dan pelanggarannya baru
> terlihat sebagai data rusak. Cara mendeteksinya — pemeriksaan berkala, trigger audit, atau tidak
> sama sekali — **belum diputuskan**.

---

## 10. Rencana perpindahan ke PostgreSQL

Dilakukan **setelah** aplikasi Go stabil di Oracle. Terpisah dari migrasi aplikasi.

1. Siapkan PostgreSQL 17+ dengan skema setara (DDL berbeda, query sama).
2. Pindahkan data. Perhatikan: tipe boolean (`NUMBER(1)` → `BOOLEAN`), `CLOB` → `TEXT`/`JSONB`,
   dan zona waktu.
3. Buat sequence `claim_no_nonpega_seq` dengan nilai awal melanjutkan Oracle.
4. Ganti konfigurasi driver dan sakelar dialek generator nomor klaim (§3.1) — **dua perbedaan**:
   `nextval('pooldata.claim_no_nonpega_seq')` dan `to_char(current_date,'YY')`.
5. Jalankan seluruh test suite terhadap PostgreSQL.
6. Jalankan paralel sementara untuk membandingkan hasil.
7. Cutover.

**Yang tidak perlu disentuh:** logika bisnis, query, handler, dan frontend. Inilah imbalan dari
D-20 — dan alasan mengapa disiplin SQL portabel di §4 harus dijaga sejak baris pertama, bukan
diperbaiki menjelang cutover.

# 9. Strategi API

Kontrak antara SPA React dan backend Go, serta antara Claim PNC dan sistem luar.

---

## 1. Bentuk API

**REST over HTTPS, JSON.** Bukan GraphQL, bukan gRPC.

Alasan: klien hanya satu (SPA kita sendiri), tim belum terbiasa dengan konsep baru (D-09), dan
REST paling mudah ditelusuri dengan alat biasa saat menelusuri masalah di production.

**Endpoint dirancang untuk kebutuhan layar, bukan untuk mencerminkan tabel.** Satu layar
sebaiknya dilayani satu permintaan. Ini penting karena Pega merender seluruh layar sekaligus;
memecah satu layar menjadi delapan panggilan API akan membuat aplikasi baru terasa lebih lambat
dari yang lama meski backend-nya lebih cepat.

---

## 2. Aturan penamaan

| Aturan | Contoh |
|---|---|
| Awalan versi | `/api/v1/...` |
| Sumber daya berbentuk jamak, `kebab-case` | `/api/v1/klaim`, `/api/v1/open-protection` |
| Istilah domain mengikuti `CONTEXT.md` | `/api/v1/klaim/{nomor}/settlement`, bukan `/adjustments` |
| Sub-sumber daya bersarang maksimal 2 tingkat | `/api/v1/klaim/{nomor}/objek` ✅ · `/klaim/{n}/objek/{o}/coverage/{c}/settlement` ❌ |
| Aksi yang bukan CRUD sebagai sub-sumber daya | `POST /api/v1/klaim/{nomor}/akseptasi` |
| Nama field JSON `camelCase` | `nomorKlaim`, `tanggalKejadian` |

**Aksi bisnis dimodelkan sebagai peristiwa, bukan pembaruan field.**
`POST /klaim/{nomor}/akseptasi` — bukan `PATCH /klaim/{nomor}` dengan `{"status": "accepted"}`.
Alasannya: akseptasi punya invarian (I-5: komite harus selesai), memicu peristiwa domain
(penerbitan DLA dan LOD), dan wajib tercatat di jejak audit. Pembaruan field generik akan
melewatkan ketiganya.

---

## 3. Bentuk respons

Seluruh respons memakai amplop yang sama, sehingga frontend punya **satu** cara menangani hasil
dan kesalahan — bukan 74 cara berbeda.

**Berhasil:** objek `data`, ditambah `meta` bila berupa daftar berhalaman.
**Gagal:** objek `error` berisi `code` (dapat dibaca mesin), `message` (dapat dibaca manusia,
bahasa Indonesia), dan `details` berupa daftar kesalahan per field.

**Kesalahan validasi dikembalikan seluruhnya sekaligus**, tidak satu per satu. Ini meniru
perilaku Pega yang menampilkan semua pesan bersamaan — pada form registrasi dengan puluhan field,
mengembalikan satu kesalahan per permintaan akan sangat menyiksa pengguna.

---

## 4. Paginasi, penyaringan, pengurutan

| Aspek | Aturan |
|---|---|
| Paginasi | **Keyset** untuk inbox dan pencarian (D-10: puluhan juta baris). Parameter `cursor` dan `limit` |
| Batas | `limit` maksimum 100. Permintaan lebih besar ditolak, bukan dipenuhi |
| Penyaringan | Parameter query eksplisit per field, **bukan** bahasa filter generik |
| Pengurutan | `sort` dengan daftar kolom yang diizinkan — **tidak pernah** nama kolom mentah dari klien |
| Total baris | Tidak dikembalikan secara baku pada data besar. `COUNT(*)` atas puluhan juta baris mahal; frontend memakai pola "muat lebih banyak" |

> Larangan filter generik dan kolom sort mentah bukan soal kerapian — itu yang menutup celah
> SQL injection yang ada pada pola `{ASIS:...}` warisan (utang teknis 4.5).

---

## 5. Kode status HTTP

| Kode | Dipakai untuk |
|---|---|
| `200` | Berhasil |
| `201` | Sumber daya dibuat |
| `400` | Permintaan tidak valid secara bentuk |
| `401` | Belum login atau token kedaluwarsa |
| `403` | Sudah login tapi tidak berwenang |
| `404` | Tidak ditemukan |
| `409` | Melanggar aturan bisnis atau konflik konkurensi |
| `422` | Validasi bisnis gagal (mis. total spreading bukan 100%) |
| `429` | Terlalu banyak permintaan |
| `500` | Kesalahan tak terduga — **detail internal tidak pernah dibocorkan ke klien** |

Pembedaan `422` dan `400` disengaja: `400` berarti klien salah membentuk permintaan (bug
frontend), `422` berarti permintaannya benar tapi melanggar aturan bisnis (kesalahan pengguna).
Frontend menanganinya berbeda.

---

## 6. Kontrak dan pembuatan tipe

- Kontrak API ditulis sebagai **OpenAPI 3**, disimpan di repository, dan menjadi **sumber
  kebenaran**.
- **Tipe TypeScript dihasilkan otomatis** dari kontrak ini. Ditulis tangan dilarang.
- Perubahan kontrak yang merusak kompatibilitas memerlukan versi baru.

Alasan aturan ini penting untuk tim di D-09: tanpa tipe hasil generate, backend dan frontend
akan berbeda persepsi tentang bentuk data, dan kesalahannya baru muncul saat runtime di layar
pengguna.

---

## 7. Idempotensi

Aksi yang menimbulkan akibat di luar sistem — akseptasi, transfer ke kasir, penerbitan DLA,
pengiriman notifikasi — wajib **idempoten**.

Klien mengirim kunci idempotensi; permintaan ulang dengan kunci yang sama mengembalikan hasil
yang sama tanpa mengulang akibatnya. Ini mencegah pembayaran ganda ketika pengguna menekan
tombol dua kali atau jaringan terputus setelah permintaan terkirim.

---

## 8. Integrasi keluar

### 8.1 Prinsip
Setiap sistem eksternal punya **seam sendiri** (Future Architecture §3.4) — bukan satu interface
raksasa. Kegagalan dan aturan retry setiap sistem berbeda, dan menyatukannya akan memaksa
perlakuan yang sama untuk hal yang tidak sama.

### 8.2 Aturan pemanggilan keluar

| Aturan | Alasan |
|---|---|
| Batas waktu **wajib** di setiap pemanggilan | Tanpa batas waktu, satu sistem yang menggantung akan menghabiskan seluruh koneksi kita |
| Retry hanya untuk operasi idempoten, dengan jeda bertambah | Retry pada operasi non-idempoten menyebabkan pengiriman ganda |
| **Tidak pernah di dalam transaksi database** | Kegagalan jaringan tidak boleh menahan kunci baris |
| Circuit breaker untuk sistem yang sering gagal | Berhenti mencoba ketika jelas sedang bermasalah |
| Setiap pemanggilan dicatat: tujuan, lama, hasil | Tanpa ini, menelusuri masalah integrasi mustahil |
| Kredensial dari konfigurasi, tidak pernah dari kode | Keamanan |

### 8.3 Sistem eksternal

| Sistem | Arah | Sifat | Bila gagal |
|---|---|---|---|
| **HCC/HCQ** | Keluar | Sinkron, menghalangi login · **kontraknya belum ada — nol jejak di export** | Pengguna tidak bisa masuk. **Apakah ada jalur cadangan belum diputuskan** (`ADR-0024`) |
| **Storage Dokumen** (app13/app8) | Dua arah | Sinkron saat unggah | Unggah gagal, klaim tetap tersimpan; dokumen bisa diulang |
| **BRI Surf** | Keluar | Asinkron | Antrekan dan coba lagi |
| **Kasir** | Keluar | Asinkron, idempoten wajib | Antrekan; **tidak boleh kirim ganda** |
| **SLIK OJK** | Keluar | Batch | Catat kegagalan, laporkan |
| **6 API pengganti DB Link** | Masuk | Sinkron | Lihat §8.4 |
| **SMTP** | Keluar | Asinkron | Antrekan dan coba lagi |

### 8.4 Pengganti DB Link (D-25)

| API baru | Menggantikan | Data |
|---|---|---|
| API HRD | `HRDASM.V_HRD_MST@ASMD` | Data pegawai |
| API Jam Kerja | `DATAMINING.GET_WORKING_HOURS@ASMD` (17×) | Perhitungan TAT hari kerja |
| API Pembayaran GL | `GL.T_ALL_PAYMENT@ASMD` | Riwayat pembayaran |
| API Master Sales | `MST_DET_SALES@ASMD`, `MST_SALES@ASMD` | Agen, MO, cabang |
| API Buka Proteksi | `GENERAL.MST_BUKA_PROTEKSI@ASMD/@SIMASNET/@SMI` | Status proteksi |
| API Pengguna Lintas Sistem | `NEW_GENERAL.M_USER@OPJAVA` | Pengguna sistem lain |

**Risiko:** API-API ini kemungkinan belum ada dan harus dibangun tim lain (**R-03**).

**Mitigasi selama menunggu:** untuk data yang jarang berubah — master sales, cabang, agen —
gunakan **salinan yang disegarkan berkala** sebagai jembatan, dengan seam yang sama sehingga
penggantian ke API nyata nanti tidak menyentuh kode domain. Untuk data yang harus mutakhir,
modul yang bergantung padanya tertahan sampai API tersedia.

> **Satu pengecualian yang tidak boleh diganti panggilan jaringan.**
> `DATAMINING.GET_WORKING_HOURS@ASMD` dipakai **18 kali di 7 berkas** di dalam kalkulasi laporan
> massal. Menjadikannya panggilan jaringan per baris akan menghancurkan kinerja laporan. `D-50`
> karena itu menetapkan **logikanya ditulis ulang di Go**, bukan dipanggil lewat API —
> perhitungan jam kerja dan kalender libur adalah **aturan bisnis**, bukan pengambilan data. Hal
> yang sama berlaku untuk `HRD_LBR`.
>
> Konsekuensinya: **kalender libur menjadi master data milik aplikasi ini** (`F-4`), dan dari mana
> daftar hari libur diperoleh setiap tahun **belum ditetapkan**.

### 8.5 Permukaan masuk yang belum pernah dihitung

Seluruh analisis integrasi sebelumnya hanya melihat Connect REST **keluar**. Folder
`Service REST/` yang diterima pada 2026-09-09 memuat **empat layanan REST masuk**:

| Layanan | Isi |
|---|---|
| `KomiteAcceptAdjustment` | **menerima persetujuan komite dari sistem lain** |
| `KomiteAcceptAdjustmentPA` | **menerima persetujuan komite dari sistem lain** (jalur PA) |
| `RecivedDataandAttachmentLelangASMSimasbid` | menerima data dan lampiran hasil lelang |
| `RequestCreateClaimCredit2` | permintaan pembuatan klaim kredit |

**Dua di antaranya menerima persetujuan komite dari luar**, sehingga menyentuh langsung kewenangan
menyetujui uang (`B-7`) dan model otorisasi `D-59`.

> **Lingkup `S-4` lebih besar daripada yang disetujui.** Permukaan masuk ini belum pernah masuk
> hitungan `FR-S4` maupun `D-25`, dan otentikasi setiap layanan masuk harus ditetapkan — siapa
> boleh memanggilnya, dan dengan kredensial apa. Diajukan sebagai pertanyaan terbuka pada
> `ADR-0008`.

# 10. Keamanan, Autentikasi, dan Otorisasi

Mengacu pada D-07: **autentikasi didelegasikan ke API internal HCC/HCQ**, **otorisasi dimiliki
sepenuhnya oleh aplikasi Claim PNC**. Ditambah kewajiban jejak audit dari D-28.

> **Diperbarui v2.0 (2026-09-14).** Tiga perubahan besar di bab ini: **satuan izin adalah menu,
> bukan aksi** (`D-59` — mengoreksi §3.1 versi sebelumnya) · **peran ditetapkan 22, satu-untuk-satu
> dengan access group** (`D-58`) · dan **HCC/HCQ ternyata nol jejak di export**, sehingga
> autentikasi menjadi integrasi greenfield tanpa baseline (`T-2`, `ADR-0024`). Ditambah bab baru
> **§6 Data nasabah di lingkungan non-produksi** (`D-64`, `D-69`), dan daftar §5 yang angkanya
> dikoreksi terhadap bukti.

---

## 1. Pemisahan yang mendasar

| | Authentication | Authorization |
|---|---|---|
| Menjawab | Siapa pengguna ini? | Boleh melakukan apa? |
| Pemilik | **HCC/HCQ** (sistem lain) | **Claim PNC** (kita) |
| Sumber data | API HCC/HCQ | Tabel milik aplikasi |
| Bila sistem lain mati | Pengguna baru tidak bisa login | Tidak terpengaruh |

Pemisahan ini adalah keputusan yang baik dan patut dipertahankan. Sistem lama menggabungkan
keduanya di Pega lewat 22 access group, sehingga menambah satu peran berarti mengubah konfigurasi
platform. Dengan pemisahan ini, peran dan izin menjadi **data biasa yang bisa diubah tanpa
deploy** (D-15).

---

## 2. Authentication

### 2.1 Alur

1. Pengguna memasukkan username dan password di SPA.
2. Backend meneruskannya ke **API HCC/HCQ**.
3. HCC/HCQ mengembalikan **profil lengkap**: NIK, nama, cabang, jabatan, email (D-07).
4. Backend mencari atau membuat catatan pengguna lokal berdasarkan NIK.
5. Backend menerbitkan **token session miliknya sendiri**.
6. Token dipakai untuk seluruh permintaan berikutnya.

**Backend menerbitkan token sendiri, bukan meneruskan token HCC/HCQ.** Alasannya: masa berlaku,
pencabutan, dan isi token menjadi kendali kita; aplikasi tidak bergantung pada HCC/HCQ untuk
setiap permintaan; dan bila HCC/HCQ sedang bermasalah, pengguna yang sudah login tetap bisa
bekerja.

> **Temuan yang mengubah derajat alur ini (2026-09-14).** `HCC` dan `HCQ` muncul **2× di seluruh
> export**, dan **keduanya teks pesan galat** yang menyuruh pengguna menghubungi helpdesk. Tidak
> ada Connect REST ke HCC/HCQ, tidak ada pemetaan field respons, tidak ada penanganan kegagalan
> autentikasi, dan tidak ada mekanisme sesi yang dapat dijadikan pembanding.
>
> Artinya alur enam langkah di atas **bukan pemindahan perilaku lama, melainkan integrasi
> greenfield sepenuhnya** — dan `F-3` tidak punya baseline untuk diuji kesetaraannya (`D-56`).
> Empat hal karenanya belum dapat ditetapkan dan menunggu Tim HCC/HCQ (`ADR-0024`, `Proposed`):
> bentuk kontraknya, perilaku saat HCC/HCQ tidak dapat dihubungi, **cara mencocokkan identitas
> HCC/HCQ dengan `OPERATOR_ID` yang dipakai di seluruh data klaim**, dan masa berlaku sesi.
>
> Butir ketiga yang paling mudah terlewat: tanpa pemetaan itu, pengguna yang berhasil login tetap
> **tidak dikenali oleh data klaimnya sendiri**.

### 2.2 Aturan token

| Aturan | Nilai |
|---|---|
| Bentuk | JWT bertanda tangan, atau token opaque + penyimpanan session |
| Masa berlaku access token | 30–60 menit |
| Refresh token | Ada, dengan rotasi |
| Penyimpanan di browser | **Cookie `HttpOnly` + `Secure` + `SameSite=Strict`** |
| Isi token | NIK, id session, waktu kedaluwarsa. **Tanpa izin** |
| Pencabutan | Daftar session aktif di server; logout mencabut |

**Kenapa izin tidak dimasukkan ke dalam token:** izin bisa berubah kapan saja lewat layar master
data. Bila izin tertanam di token, pencabutan hak akses baru berlaku setelah token kedaluwarsa —
bisa satu jam kemudian. Izin dibaca dari database (dengan cache in-process berumur pendek) agar
perubahan berlaku hampir seketika.

**Kenapa cookie, bukan `localStorage`:** token di `localStorage` dapat dibaca JavaScript
sehingga satu kerentanan XSS langsung berarti pencurian session. Cookie `HttpOnly` tidak dapat
dibaca JavaScript.

### 2.3 Kegagalan HCC/HCQ
Bila API tidak dapat dihubungi, pesan yang ditampilkan harus **membedakan** antara "sistem
autentikasi sedang bermasalah" dan "username atau password salah". Pesan yang sama untuk
keduanya membuat pengguna mencoba berulang kali dan membanjiri sistem yang sedang bermasalah.

Pengguna yang **sudah** login tidak terpengaruh, karena token diterbitkan aplikasi sendiri.

---

## 3. Authorization

### 3.1 Model

Tiga tingkat, dari kasar ke halus:

```
Pengguna  ──►  Peran  ──►  Izin  ──►  akses Menu dan Aksi
                             │
                             └──►  Batas Data (cabang · lini bisnis)
```

**Peran berjumlah 22, satu-untuk-satu dengan access group Pega** (`D-58`), hanya dinamai ulang
agar terbaca manusia. Tidak ada penggabungan dan tidak ada pemecahan. Seluruhnya terverifikasi ada
di export sebagai literal `GCNMFW:<nama>`: PncAdmin, PncManagerAdmin, PncPICTeknik,
PNCKomiteTeknik, PNCKomite, CaseManager, PncRCLPUCL, PncAnalystDoctor, PncComplience,
PncInvestigator, PNCSurveyor, PncPLADLA, PncReceive, PncManagerReceive, PncCollection,
PncOPCGeneral, PNCServiceCenter, TreatyIn, ViewClaimPNC, PNCReportClaimInternal,
PNCReportClaimEksternal, Administrators.

> **Koreksi v2.0 — satuan izin adalah menu, bukan aksi.** Versi sebelumnya menetapkan izin
> berbutir aksi (`klaim.akseptasi`, `komite.setujui`, …). **`D-59` memutuskan sebaliknya:** satuan
> izin adalah **menu**, dan pengguna yang memiliki akses ke sebuah menu berwenang atas **seluruh
> tindakan yang dijangkau menu itu** — termasuk membatalkan klaim, mengubah nilai setelah
> persetujuan komite, dan menyetujui komite. **Tidak ada pemisahan tugas formal.**

**Yang tetap berubah dari sistem lama adalah tempat penegakannya.** Sistem lama hanya
menyembunyikan menu — `pyPrivilegeName` terisi pada **1 dari 902 activity**, dan yang satu itu
privilege bawaan Pega untuk ekspor ruleset, bukan aturan bisnis. Di sistem baru, **setiap endpoint
memeriksa di server**: *apakah peran pemanggil memiliki menu yang memberi akses ke endpoint ini*.
Menyembunyikan menu hanyalah kenyamanan tampilan. Dengan bacaan itu, `BRD §21.2` kriteria #8 dan
`FR-R1` tetap terpenuhi.

**Konsekuensi yang diterima secara sadar** (`D-59`):

1. **Orang yang sama dapat membuat, menyetujui, dan membayarkan satu klaim** bila perannya
   memiliki ketiga menu itu. Tidak ada kontrol teknis yang mencegahnya.
2. **Jejak audit menjadi satu-satunya kontrol pengimbang.** Ini menaikkan `S-5` dari modul
   pendukung menjadi kontrol utama — dan `S-5` adalah kemampuan **baru 100% tanpa baseline**.
3. **`AutoAcceptKomite` melewati kontrol apa pun.** Job terjadwal harian jam 06:00 menyetujui
   komite tanpa pengguna sama sekali, sehingga bahkan kontrol berbasis menu tidak berlaku padanya.
4. Bila kemudian ada temuan audit atau pentest yang menuntut pemisahan tugas, perubahannya
   menyentuh model izin `F-3` — bukan penyesuaian kecil.

**Yang menjadi wajib karenanya:** `BRD §21.2` kriteria #9 — jejak audit untuk setiap perubahan
bernilai bisnis — berlaku **tanpa pengecualian** pada seluruh tiket modul bisnis.

**Tiga hal yang harus ditangani `F-3` saat membangun tabel peran:**

| Hal | Isi |
|---|---|
| **Kapitalisasi tidak konsisten** | tiga nama muncul dalam dua bentuk — `ViewClaimPNC`/`VIEWCLAIMPNC` dan `PncReceive`/`PNCRECEIVE`. Sistem baru wajib menormalkannya menjadi satu identitas per peran |
| **Peta peran → menu hidup di rule, bukan data** | pemetaan ke **51 item menu** ada di **34 When rule** + `Navigation/pyCaseWorkerNavigation-Navigation.xml`. **Lima di antaranya hilang dari export**: `IsGCNMReport`, `IsKomite`, `IsNotViewClaim`, `IsPNCBonding`, `IsSurvey` |
| **Penugasan operator ke peran tidak ada di database** | `POOLDATA.T_ACCESS_GROUP_PNC` hanya memetakan `OPERATOR_ID` → `OLD_OPERATOR_ID`. Tanpa artefak ini, `F-3` dapat membangun tabelnya tetapi **tidak dapat mengisinya** |

### 3.2 Batas data

Selain "boleh melakukan aksi apa", ada "boleh melihat data siapa". Dari analisis when rule
sistem lama, batasnya berdasarkan:
- **Cabang** — pengguna cabang hanya melihat klaim cabangnya
- **Lini bisnis / Group Panel** — PIC Teknik PA tidak melihat klaim Marine
- **Organisasi** — pengguna `Eksternal` dibatasi tegas (`OperatorID.pyOrgUnit != "Eksternal"`)

**Batas data ditegakkan di lapisan query**, bukan disaring setelah data terambil. Menyaring
setelah pengambilan berarti data yang tidak boleh dilihat sempat berada di memori aplikasi dan
ikut terhitung pada paginasi — bocor lewat jumlah baris.

### 3.3 Tabel baru yang dibutuhkan

Belum ada (D-07), dirancang dalam project ini:

| Tabel | Isi |
|---|---|
| Pengguna | NIK, nama, email, cabang, jabatan, status aktif |
| Peran | kode, nama, deskripsi — **22 baris** (`D-58`) |
| PenggunaPeran | pemetaan pengguna → peran |
| Izin | **kode menu** — 51 item (`D-59`) |
| PeranIzin | pemetaan peran → menu |
| BatasDataPengguna | cabang dan lini bisnis yang boleh diakses |
| SessionAktif | untuk pencabutan |

> **Tabel dapat dibangun, tetapi belum dapat diisi.** Dua isian menunggu pihak lain: **daftar
> operator per peran** (tidak ada di database) dan **kontrak API HCC/HCQ** yang menentukan bentuk
> identitas penggunanya (`ADR-0024`, berstatus `Proposed`).

---

## 4. Pengamanan lain

### 4.1 Masukan pengguna

| Ancaman | Penangkal |
|---|---|
| SQL injection | **Parameter binding tanpa perkecualian.** Nama kolom sort dan filter hanya dari daftar yang diizinkan. Ini yang menutup celah `{ASIS:...}` warisan |
| XSS | React meng-escape secara baku. `dangerouslySetInnerHTML` **dilarang** kecuali disetujui tertulis |
| CSRF | Cookie `SameSite=Strict` + token CSRF pada permintaan yang mengubah data |
| Unggahan berkas berbahaya | Validasi jenis berkas dan ukuran; nama berkas dihasilkan sistem, tidak pernah dari pengguna |
| Path traversal | Tidak ada akses berkas berdasarkan jalur dari pengguna |

### 4.2 Data sensitif

Sistem menyimpan NIK, nomor telepon, alamat, data medis (klaim PA), dan nomor rekening.

- Sistem lama sudah punya konsep **masking** (`FlagMaskingKTP`, `TelpMasking`,
  `NoKTPMasking`, `SetMaskingCIF_DT`) — ini **dipertahankan**.
- Data sensitif **tidak pernah** ditulis ke log (lihat §12 Cross-Cutting).
- Akses ke data medis dibatasi peran Analyst Doctor dan RCL Dokter.
- Sambungan ke database dan sistem eksternal wajib terenkripsi.

### 4.3 Kredensial
Tidak pernah di dalam kode maupun di repository. Diambil dari variabel lingkungan atau
pengelola rahasia. Berkas konfigurasi contoh hanya memuat nilai kosong.

### 4.4 Jejak audit sebagai kendali keamanan
Jejak audit (D-28) bukan hanya kebutuhan bisnis — ia juga kendali keamanan. Sifat append-only
ditegakkan lewat **hak akses database** (aplikasi hanya diberi `INSERT` dan `SELECT`), sehingga
tidak ada jalur di dalam aplikasi yang bisa menghapus jejaknya sendiri.

---

## 5. Yang harus dihilangkan dari sistem lama

| Masalah di sistem lama | Ukuran terverifikasi | Perbaikan |
|---|---|---|
| Perangkaian SQL `{ASIS:...}` dari nilai pengguna | 538 kemunculan | Parameter binding wajib |
| Potongan klausa SQL disimpan sebagai nilai property | — | Dilarang sepenuhnya |
| **User ID di-hardcode sebagai penentu perilaku** (`MORASOTARDODOTARIGAN`, `ELLENSUPRIYATI`, `IRMANOPITAPURBA_1`, `RATNAGUSNITASARI`, …) | **24 unik**, ditambah belasan tertanam di dalam teks SQL | Peran dan izin dari master data (`D-15`) |
| **Alamat email di-hardcode** sebagai penerima notifikasi | **66 unik**, termasuk **≥6 akun Gmail pribadi di jalur produksi** dan **5 alamat dipakai sebagai Operator ID** | Seluruh penerima dari master **Penerima Notifikasi**, berupa **mailbox fungsional**; **tidak ada akun pribadi** (`D-67`) |
| **Ambang uang di-hardcode** | **8 ambang komite unik** + 7 ambang uang non-komite | Master Ambang Komite (`F-4`) |
| **Hostname server menentukan perilaku bisnis** — salah satunya mengubah ambang komite dari Rp 50.000.000 menjadi 3.500 | **3 hostname, 48 perbandingan** terhadap `pxRequestor.pxReqServer` | Konfigurasi per lingkungan, bukan deteksi hostname. **Cara entitas dikenali di sistem baru belum diputuskan** (`ADR-0025`) |
| **Blok `// TESTING` menimpa email produksi** | 4 step di `Activity/InputRegister_act-Act.xml` (`:16693`, `:16830`, `:16998`, `:17141`), precondition **identik** dengan step produksi | Dilarang; ditolak di code review |
| **Kredensial plaintext di dalam rule** | **3 password SMTP di 31 lokasi** + 1 pasang kredensial OAuth; `UseSSL=false` di seluruh kemunculannya | Dipindahkan keluar dari kode — **tujuan penyimpanannya masih `OPEN`** (`D-40`, `R-17`) |
| **Dua integrasi menunjuk host sandbox** di ruleset produksi | 2 Connect REST | Endpoint per lingkungan sebagai konfigurasi (`R-18`) |
| Otorisasi bergantung pada penyembunyian menu | `pyPrivilegeName` terisi di **1 dari 902** activity | Pemeriksaan izin **di setiap endpoint backend** (`D-59`) |
| **Nol Dynamic System Setting** — konfigurasi dinamis berupa tabel Oracle yang dikunci **per IP aplikasi** | — | Konfigurasi aplikasi yang tidak terikat IP; bertabrakan dengan tuntutan dua instans (`D-27`) |

> **Nilai sensitif tidak direproduksi di dokumen ini.** Hostname produksi, alamat email,
> kredensial, dan data nasabah dirujuk dengan `berkas:baris` + nama elemen saja (`D-69`). Lokasi
> lengkap kredensial sudah diserahkan ke Tim Infra/Security lewat dokumen terpisah **di luar
> repository**.

---

## 6. Data nasabah di lingkungan non-produksi

`D-64` menetapkan data produksi disalin ke **staging apa adanya, tanpa penyamaran**, dengan **hak
akses staging diperketat** sebagai kontrol penggantinya. Alasannya ada di Testing Strategy §6.1:
cacat yang ditemukan pada verifikasi Fase 1 muncul dari data nyata yang tidak akan terpikir
dibuat.

**Konsekuensi yang diterima secara sadar:**

1. **Staging memuat data nasabah nyata** — nomor polis, nama tertanggung, NPWP, nomor rekening,
   dan **data medis** pada lini PA dan Travel. Perlindungannya sepenuhnya bergantung pada hak
   akses, bukan pada sifat datanya.
2. **Klasifikasi staging naik setara produksi** untuk keperluan keamanan.
3. Pembatasan akses data medis yang `FR-R2` terapkan pada peran Analyst Doctor dan RCL Dokter
   **berlaku juga di staging**.

**Masih terbuka:** siapa yang menyetujui akses staging, berapa lama salinan disimpan, dan
prosedur pemusnahannya — ditujukan ke pihak yang sama dengan `D-40`.

**Aturan penulisan dokumen** (`D-69`), berlaku untuk seluruh artefak yang di-commit:

| Jenis nilai | Perlakuan |
|---|---|
| Nama Operator ID | **boleh ditulis lengkap** — diperlukan agar tiket dapat menunjuk hardcode mana yang dihapus |
| Alamat email | **selalu disamarkan** |
| Data nasabah — nomor polis, nama tertanggung, nomor klaim asli, NPWP, nomor rekening | **tidak pernah ditulis**; dirujuk dengan `berkas:baris` atau nama kolom |
| Kredensial, kunci API, hostname/IP produksi | **tidak pernah ditulis** |

# 11. Error Handling, Logging, Konfigurasi, dan Observability

Hal-hal yang menyentuh seluruh modul. Bila tidak diseragamkan sejak awal, setiap modul akan
melakukannya dengan caranya sendiri — dan pada tim di D-09 dengan 74 layar, itu pasti terjadi.

---

## 1. Error Handling

### 1.1 Tiga jenis kesalahan

| Jenis | Contoh | Penanganan | HTTP |
|---|---|---|---|
| **Kesalahan validasi bisnis** | Total spreading bukan 100% · DOL di luar periode polis · Nomor SLIK kosong | Kumpulkan **semua**, kembalikan bersamaan | `422` |
| **Pelanggaran aturan / konflik** | Akseptasi sebelum komite selesai · penugasan sudah diambil orang lain | Kembalikan satu kesalahan yang jelas | `409` |
| **Kegagalan teknis** | Database mati · sistem eksternal tidak merespons · bug | Catat lengkap, kembalikan pesan umum | `500` |

### 1.2 Aturan

1. **Kesalahan validasi dikumpulkan seluruhnya, tidak berhenti pada yang pertama.**
   `InputRegister_act` sistem lama memeriksa belasan aturan dan menampilkan semuanya sekaligus.
   Pada form registrasi berisi puluhan field, mengembalikan satu kesalahan per percobaan akan
   membuat pengguna menyerah. Ini bukan preferensi — ini kesetaraan perilaku (P-5).

2. **Kesalahan domain adalah tipe, bukan string.** Transport yang memetakannya ke kode HTTP.
   Domain tidak boleh tahu tentang HTTP.

3. **Setiap kesalahan dibungkus dengan konteks saat naik**, sehingga pesan akhirnya menceritakan
   jalurnya — bukan sekadar "record not found" tanpa keterangan record apa.

4. **Kesalahan tidak pernah ditelan.** Tidak ada `_ = err`. Bila memang sengaja diabaikan,
   wajib disertai komentar yang menjelaskan alasannya.

5. **Detail internal tidak pernah bocor ke klien.** Pesan `500` yang dikirim ke pengguna hanya
   memuat id permintaan; detail lengkapnya ada di log. Membocorkan struktur database atau
   jejak tumpukan ke browser adalah celah keamanan.

6. **Pesan untuk pengguna berbahasa Indonesia dan menjelaskan cara memperbaiki**, mengikuti gaya
   sistem lama yang sudah dikenal pengguna:
   > "Tanggal Lapor tidak boleh lebih dari 7 hari setelah Tanggal Kejadian."
   > "Nomor Polis sudah terdaftar dengan nomor klaim PNC-1865."
   > "Total spreading harus 100%."

7. **`panic` hanya untuk kondisi yang tidak mungkin terjadi**, ditangkap oleh middleware
   recovery, dicatat lengkap, dan dikembalikan sebagai `500`. Tidak pernah dipakai sebagai alur
   kendali.

---

## 2. Logging

### 2.1 Bentuk
**Terstruktur dalam JSON** memakai `log/slog`. Bukan teks bebas — log teks bebas tidak bisa
dicari, disaring, maupun diagregasi ketika sedang menelusuri masalah di production.

### 2.2 Tingkat

| Tingkat | Untuk | Contoh |
|---|---|---|
| `ERROR` | Butuh perhatian manusia | Database tidak dapat dihubungi · kegagalan tak terduga |
| `WARN` | Tidak normal tapi tertangani | Sistem eksternal gagal lalu berhasil saat dicoba ulang |
| `INFO` | Peristiwa bisnis penting | Klaim diregistrasi · akseptasi diterbitkan · pengguna login |
| `DEBUG` | Penelusuran mendalam | Mati di production, dinyalakan sementara saat dibutuhkan |

**Kegagalan validasi bisnis bukan `ERROR`.** Pengguna salah mengisi form adalah hal normal.
Mencatatnya sebagai `ERROR` akan menenggelamkan kesalahan sungguhan di antara ribuan baris
kesalahan pengisian.

### 2.3 Field wajib di setiap baris log
Waktu (UTC), tingkat, pesan, **id permintaan**, id pengguna, metode dan jalur HTTP.
Untuk peristiwa bisnis, tambahkan nomor klaim.

**Id permintaan** dibuat di middleware paling luar, dibawa lewat `context.Context`, dan muncul
di setiap baris log dari permintaan itu — juga dikembalikan ke klien pada respons `500`.
Tanpa ini, menelusuri "apa yang terjadi pada permintaan pengguna tadi" hampir mustahil.

### 2.4 Yang dilarang masuk log

Password · token dan kunci API · **NIK** · nomor telepon · alamat · **data medis** · nomor
rekening · isi dokumen.

Bila diperlukan untuk penelusuran, catat **referensinya** (nomor klaim, id dokumen), bukan
isinya. Log biasanya disimpan lebih longgar daripada database, sehingga data sensitif di dalam
log adalah kebocoran yang mudah terlewat.

### 2.5 Log pemanggilan sistem eksternal
Setiap pemanggilan keluar dicatat: sistem tujuan, operasi, lama, hasil, dan percobaan ke berapa.
Ini yang membuat pertanyaan "apakah lambatnya karena kita atau karena sistem mereka" bisa
dijawab dengan data.

---

## 3. Configuration

### 3.1 Tiga lapis, dari umum ke khusus

```
Nilai baku di kode  →  berkas YAML per lingkungan  →  variabel lingkungan
       (paling umum)                                      (paling menang)
```

- **Rahasia hanya dari variabel lingkungan** — tidak pernah dari berkas YAML yang masuk repository.
- **Aplikasi gagal saat start bila konfigurasi wajib tidak ada.** Gagal keras di awal jauh lebih
  baik daripada gagal diam-diam saat pengguna sedang bekerja.

### 3.2 Konfigurasi teknis (butuh restart)
Alamat database dan ukuran pool · alamat sistem eksternal · batas waktu · port · tingkat log ·
masa berlaku token.

### 3.3 Master data (dapat diubah tanpa restart) — D-15

Ini yang menggantikan seluruh hardcode di sistem lama:

| Master | Menggantikan hardcode | Ukuran terverifikasi |
|---|---|---|
| Ambang komite per lini bisnis | `50000000` · `30000000` · `20000000` · `7000` · `3500` · … | **8 ambang komite unik** + 7 ambang uang non-komite |
| Ambang Large Losses | `1000000000` | 1 |
| Penerima notifikasi per peristiwa dan Group Panel | email UW, pimpinan, komite, broker | **66 alamat unik**, termasuk **≥6 akun Gmail pribadi di jalur produksi** |
| Batas aturan tanggal per lini bisnis | 7 hari · 30 hari · 90 hari | — |
| Peran dan izin menu | 22 access group | 22 peran → 51 item menu |
| Operator ID penentu perilaku | `MORASOTARDODOTARIGAN`, `ELLENSUPRIYATI`, `IRMANOPITAPURBA_1`, `RATNAGUSNITASARI`, … | **24 unik** |
| Master status klaim | — | **33 kode `1134`–`1166`** (`R-06` tertutup) |
| Kurs per tanggal | `GETCURRENCYSTANDARD` yang mengembalikan `1` saat kurs tidak ada | — (`D-48`) |
| Hari libur dan jam kerja | `GET_WORKING_HOURS` dan `HRD_LBR` lewat DB Link | 18 pemakaian di 7 berkas |
| Retensi data | tidak ada | satu kebijakan untuk data klaim **dan** jejak audit (`D-62`) |

> **Angka pada versi sebelumnya terlalu kecil.** `docs/BRD` menyebut 10 email, 4 user ID, dan
> 3 ambang; verifikasi terhadap seluruh export memberi **66 / 24 / 8**. Angka lama benar untuk
> lingkupnya — dua rule saja — dan memakainya untuk `FR-F4` akan membuat estimasi master data
> meleset sekitar **enam kali lipat**.
>
> **Tidak ada akun pribadi yang dibawa** (`D-67`): seluruh penerima notifikasi berasal dari master
> Penerima Notifikasi berupa **mailbox fungsional**. Lima alamat Gmail yang dipakai sebagai
> **Operator ID** di filter laporan KPI adalah masalah **identitas**, bukan notifikasi — ia
> menyentuh `F-3` sekaligus `F-4`.

**Perbedaan yang menentukan:** konfigurasi teknis milik tim infrastruktur dan berubah saat
deployment. Master data milik pengguna bisnis dan berubah saat kebijakan berubah. Menaruh ambang
komite di berkas konfigurasi berarti setiap perubahan kebijakan membutuhkan deployment — itu
persis masalah yang membuatnya di-hardcode sejak awal.

### 3.4 Larangan perilaku berdasarkan hostname

Sistem lama membandingkan `pxRequestor.pxReqServer` terhadap **3 hostname** pada **48 titik**, dan
perbandingan itu **menentukan perilaku bisnis**, bukan sekadar tampilan:

| Host | Perbandingan | Perilaku yang dipicu |
|---|---|---|
| host **dev** | 34 | melewati atau mengganti step; mengganti penerima email; menentukan `IsServiceCenterPNC` |
| host **entitas Timor-Leste** | 1 | **mengubah ambang komite dari Rp 50.000.000 menjadi 3.500** |
| host **entitas Insurtech** | 13 | mengganti kode entitas, status investigator, filter view klaim |

Nilai hostname **tidak direproduksi di sini** sesuai aturan penulisan `D-69`; lokasinya dirujuk
dengan `berkas:baris` pada `docs/verifikasi-bukti-adr.md` §7.6.

**Ini dilarang.** Perbedaan antar lingkungan dan antar negara dinyatakan sebagai konfigurasi
eksplisit, bukan disimpulkan dari nama server. Perilaku yang bergantung pada hostname tidak dapat
diuji, tidak terlihat saat membaca kode, dan berubah diam-diam ketika server dipindahkan.

> **Yang menggantikannya belum diputuskan.** Bagaimana entitas — Indonesia, Timor-Leste,
> Insurtech — dikenali di sistem baru adalah pertanyaan terbuka pada `ADR-0025`, dan jawabannya
> menentukan **ambang komite mana yang berlaku** untuk sebuah klaim.

### 3.5 Tidak ada Dynamic System Setting yang bisa ditiru

Sistem lama **tidak memiliki satu pun Dynamic System Setting**. Konfigurasi dinamis yang nyata
berupa **tabel Oracle yang dikunci per IP aplikasi** — pola yang bertabrakan langsung dengan
tuntutan **dua instans di belakang load balancer** (`D-27`).

Artinya tidak ada mekanisme konfigurasi lama yang dapat disalin; lapisan konfigurasi §3.1 adalah
**kemampuan baru**, bukan pemindahan.

### 3.6 Rahasia — masih terbuka

Export memuat **3 password SMTP di 31 lokasi** dan **1 pasang kredensial OAuth**, seluruhnya
plaintext, dengan `UseSSL=false` di seluruh kemunculannya (`R-17`).

Aturan §3.1 — *"rahasia hanya dari variabel lingkungan"* — adalah **arah**, bukan keputusan final.
**Ke mana rahasia dipindahkan, siapa pemiliknya, dan apakah kredensial yang telanjur terekspos
harus dirotasi masih `OPEN`** (`D-40`, `ADR-0025`). Sampai itu dijawab, `F-4` dan `F-5` tidak dapat
ditulis lengkap, dan cara repository ini disimpan ikut menjadi persoalan keamanan hari ini.

---

## 4. Observability

Dijaga sederhana dan sepadan dengan skala 200–300 pengguna di VM on-premise (D-08). Tanpa
platform observability besar.

### 4.1 Health check

| Endpoint | Menjawab | Dipakai oleh |
|---|---|---|
| `/health/live` | Proses masih hidup? | Load balancer |
| `/health/ready` | Siap menerima trafik? (database terhubung, migrasi selesai) | Load balancer saat rolling deployment |

Pembedaan keduanya penting untuk D-27: saat rolling deployment, instance baru harus dinyatakan
*belum siap* sampai benar-benar siap, agar load balancer tidak mengirim trafik ke instance yang
sedang start.

### 4.2 Metrik
Diekspos dalam format Prometheus, dikumpulkan bila sudah ada infrastrukturnya.

| Kelompok | Isi |
|---|---|
| HTTP | Jumlah permintaan, lama, kode status per endpoint |
| Database | Koneksi terpakai, koneksi menunggu, lama query |
| Sistem eksternal | Jumlah pemanggilan, lama, tingkat kegagalan per sistem |
| Bisnis | Klaim diregistrasi, klaim diakseptasi, tugas menunggu per workbasket |
| Runtime | Memori, goroutine, GC |

Metrik bisnis sengaja dimasukkan: "jumlah tugas menunggu di workbasket komite" adalah tanda
peringatan operasional yang jauh lebih berguna daripada penggunaan CPU.

### 4.3 Pemberitahuan yang layak membangunkan orang

| Kondisi | Alasan |
|---|---|
| Tingkat kesalahan `500` melebihi ambang | Ada yang rusak |
| Database tidak dapat dihubungi | Aplikasi tidak berfungsi |
| Health check gagal pada satu instance | Kapasitas berkurang; 24/7 terancam (D-27) |
| Sistem eksternal gagal terus-menerus | Fungsi tertentu lumpuh |
| Antrean notifikasi menumpuk | Pemberitahuan tidak terkirim |

**Tidak** memicu pemberitahuan: kegagalan validasi pengguna, `404`, dan lonjakan trafik yang
wajar. Pemberitahuan yang terlalu sering membuat orang berhenti memperhatikannya — dan saat itu
terjadi, pemberitahuan menjadi tidak berguna.

### 4.4 Penelusuran jejak
Id permintaan yang dibawa lewat `context.Context` dan muncul di seluruh log sudah memadai untuk
skala ini. Distributed tracing tidak diperlukan karena hanya ada satu layanan.

# 12. Strategi Deployment dan Infrastruktur

Mengacu pada **VM/bare metal on-premise** (D-08) dengan tuntutan **ketersediaan 24/7** (D-27),
untuk 200–300 pengguna aktif harian.

---

## 1. Topologi

```
                    ┌────────────────────┐
                    │   Load Balancer    │  health check aktif
                    │  (nginx / HAProxy) │  sticky session TIDAK diperlukan
                    └─────────┬──────────┘
              ┌───────────────┴───────────────┐
    ┌─────────▼─────────┐          ┌──────────▼────────┐
    │  VM Aplikasi 1    │          │  VM Aplikasi 2    │
    │                   │          │                   │
    │  binary Go        │          │  binary Go        │
    │  + SPA tersemat   │          │  + SPA tersemat   │
    │  (satu proses)    │          │  (satu proses)    │
    └─────────┬─────────┘          └──────────┬────────┘
              └───────────────┬───────────────┘
                    ┌─────────▼──────────┐
                    │  Oracle 19c        │
                    │  → PostgreSQL 17+  │
                    │  (dibagi dgn Pega  │
                    │   selama paralel)  │
                    └────────────────────┘
```

**Minimal dua instance aplikasi** — bukan pilihan, melainkan konsekuensi langsung dari D-27.
Dengan satu instance, setiap deployment berarti downtime.

**Satu proses per VM.** Binary Go menyajikan API sekaligus berkas statis SPA. Tidak ada
Node.js, tidak ada web server terpisah untuk berkas statis. Ini konsekuensi dari menolak
Next.js/Nuxt di D-23, dan inilah manfaat nyatanya: satu hal untuk di-deploy, dipantau, dan
ditambal keamanannya.

---

## 2. Syarat teknis untuk 24/7

| Syarat | Alasan |
|---|---|
| **Aplikasi stateless** | Session di memori satu instance akan hilang saat instance itu di-restart, dan pengguna terlempar keluar di tengah pekerjaan |
| **Graceful shutdown** | Saat instance dimatikan, permintaan yang sedang berjalan harus diselesaikan lebih dulu, bukan diputus |
| **Health check `live` dan `ready` terpisah** | Load balancer harus tahu kapan instance baru benar-benar siap menerima trafik |
| **Migrasi skema backward-compatible** | Saat rolling deployment, versi lama dan baru berjalan bersamaan terhadap skema yang sama (DB-7) |
| **Migrasi dijalankan terpisah dari start aplikasi** | Bila dijalankan saat start, dua instance akan bermigrasi bersamaan |
| **Job terjadwal hanya berjalan di satu instance** | Tanpa ini, job berjalan dua kali. Ditegakkan dengan kunci di database, bukan konfigurasi manual |

Poin terakhir sering terlewat dan akibatnya serius: job pengiriman notifikasi atau transfer ke
kasir yang berjalan dua kali berarti pengiriman ganda.

---

## 3. Rolling deployment

1. Migrasi skema dijalankan (backward-compatible, aman terhadap versi lama).
2. Instance 1 ditandai *tidak siap*; load balancer berhenti mengirim trafik.
3. Instance 1 menyelesaikan permintaan yang sedang berjalan, lalu berhenti.
4. Instance 1 dijalankan dengan versi baru; menunggu sampai `/health/ready` sukses.
5. Load balancer mengembalikan trafik ke Instance 1.
6. Ulangi untuk Instance 2.

**Tanpa downtime.** Selama proses, satu instance selalu melayani.

**Bila gagal:** kembalikan binary ke versi sebelumnya dan ulangi langkah yang sama. Karena
migrasi skema backward-compatible, versi lama tetap berfungsi terhadap skema baru.

---

## 4. Lingkungan

| Lingkungan | Kegunaan | Database |
|---|---|---|
| **Development** | Pengembangan lokal | Database lokal |
| **Staging** | Pengujian, **verifikasi kesetaraan dengan Pega** | **Salinan produksi apa adanya, tanpa penyamaran** (`D-64`) |
| **Production** | Operasional | Oracle produksi (dibagi dengan Pega selama paralel) |

**Staging wajib memakai salinan data produksi.** Verifikasi kesetaraan hanya bermakna bila
dijalankan terhadap data nyata — data uji buatan tidak akan memicu kasus tepi yang justru paling
sering menjadi sumber perbedaan hasil. Buktinya konkret: cacat yang ditemukan pada verifikasi
Fase 1 — toleransi spreading berupa pencocokan substring, kurs yang mengembalikan `1`,
`IDSALVAGE = NULL` — **muncul dari data nyata yang tidak akan terpikir dibuat**.

> **Koreksi v2.0 — data tidak disamarkan.** Versi sebelumnya mewajibkan penyamaran NIK, nomor
> telepon, alamat, data medis, dan nomor rekening. **`D-64` memutuskan sebaliknya:** data disalin
> **apa adanya**, dan **hak akses lingkungan staging diperketat** sebagai kontrol penggantinya.
> Penyamaran akan menghilangkan justru sifat data yang membuatnya berguna untuk uji kesetaraan.

**Konsekuensi yang diterima secara sadar:**

1. **Staging memuat data nasabah nyata** — nomor polis, nama tertanggung, NPWP, nomor rekening,
   dan **data medis** pada lini PA dan Travel.
2. **Klasifikasi staging naik setara produksi** untuk keperluan keamanan.
3. Pembatasan akses data medis (`FR-R2`) berlaku juga di staging, bukan hanya produksi.

**Masih terbuka:** siapa yang menyetujui akses staging, berapa lama salinan disimpan, dan prosedur
pemusnahannya — ditujukan ke pihak yang sama dengan `D-40`.

**Satu prasyarat yang belum dikonfirmasi:** uji kesetaraan menuntut **Pega staging yang dapat
ditembak dari luar** sebagai pembanding. Ketersediaannya **belum dipastikan**, dan tanpa itu
gerbang 1 seluruh modul tidak dapat dijalankan (`ADR-0027`).

**Satu risiko yang berlaku pada penyalinannya sendiri:** `R-12` — pergeseran zona waktu — berlaku
pada **proses menyalin data ke staging**, bukan hanya pada migrasi akhir. Salinan yang bergeser
akan menghasilkan selisih palsu di setiap pengujian.

---

## 5. Rilis

- **Artefak:** satu berkas binary Go dengan SPA tersemat, ditandai versi dan commit.
- Binary yang sama dipromosikan dari staging ke production — **tidak pernah dibangun ulang**
  untuk production. Build ulang berarti yang diuji bukan yang dijalankan.
- Konfigurasi berasal dari lingkungan, bukan dari binary.
- Setiap rilis dicatat: versi, commit, isi perubahan, waktu, pelaksana.

---

## 6. Backup dan pemulihan

Mengikuti **standar backup korporat data center Sinarmas** (D-29).

**Yang harus diperiksa terhadap standar itu**, dan diangkat ke tim infra bila belum sejalan:

| Pertanyaan | Kenapa penting |
|---|---|
| Apakah RPO standar sejalan dengan sifat data klaim? | Data klaim menyangkut nilai uang dan kewajiban ke tertanggung |
| Apakah RTO standar sejalan dengan tuntutan 24/7 (D-27)? | Standar backup menjawab pemulihan bencana, bukan operasional harian — keduanya berbeda |
| Apakah pemulihan pernah benar-benar diuji? | Backup yang tidak pernah diuji pemulihannya bukan backup |
| Apakah retensi memenuhi kewajiban audit (D-28)? | Retensi audit bisa lebih panjang dari retensi backup biasa |

---

## 7. Infrastruktur yang diperlukan

| Komponen | Keterangan |
|---|---|
| 2 VM aplikasi | CPU dan memori sedang; Go hemat sumber daya |
| 1 load balancer | nginx atau HAProxy dengan health check aktif |
| Database | Oracle 19c yang ada; kelak PostgreSQL 17+ (D-24) |
| Sertifikat TLS | Wajib; termasuk untuk sambungan internal |
| Pengumpul log | Terpusat agar log dari kedua instance dapat dicari bersamaan |
| Pemantauan | Prometheus + Grafana bila tersedia; minimal pemantauan health check |
| Penyimpanan dokumen | API internal yang sudah ada (D-16) — tidak perlu infrastruktur baru |

**Yang sengaja tidak diperlukan:** Kubernetes (D-08), message broker, cache terdistribusi,
service mesh. Seluruhnya dijelaskan di Future Architecture §6.

---

## 8. Hidup berdampingan dengan Pega

Selama masa paralel (D-05):

- Pega dan aplikasi Go berjalan bersamaan, memakai database yang sama (D-21).
- Pengguna diarahkan ke sistem yang tepat berdasarkan modul — lewat menu portal atau aturan
  routing di load balancer.
- **Pega tidak boleh dinonaktifkan sebelum Tahap 7** (Migration Strategy §2). Kemampuan mundur
  bergantung sepenuhnya pada Pega yang masih berfungsi penuh.
- Beban database naik karena dua sistem membacanya. Ini harus dipantau sejak awal masa paralel,
  bukan setelah pengguna mengeluh.
- **Perubahan skema menempuh tiga pihak** — permintaan tertulis tim pengembang, persetujuan Work
  Owner, pelaksanaan DBA, lalu **wajib diuji dengan menjalankan Pega dan Go bersamaan** (`D-63`).
  Satu `ALTER` yang keliru pada tabel yang dibaca 116 rule Pega menghentikan produksi.

---

## 9. Job terjadwal pada dua instans — belum diputuskan

Sistem lama memiliki **5 job terjadwal + 1 agent**, seluruhnya `pyApplicableTo=Cluster` —
mekanisme Pega yang menjamin satu job berjalan **sekali saja** meski ada banyak node (`D-57`).

Go **tidak punya padanan otomatis**, sementara §1 menuntut minimal dua instans di belakang load
balancer. Tiga pilihan dipertimbangkan di `ADR-0022` dan **belum diputuskan**: penjadwal di dalam
binary dengan penguncian lewat database, cron sistem operasi di satu VM, atau instans penjadwal
terpisah dari binary yang sama.

> **Kenapa ini bukan detail teknis kecil.** Bila penguncian salah dirancang,
> **`AutoAcceptKomite` dapat berjalan dua kali** — menyetujui komite dua kali pada klaim yang
> sama. Itu kesalahan bernilai uang, bukan gangguan operasional.

Dua pertanyaan lain menunggu Work Owner, bukan tim teknis: apakah `AutoAcceptKomite` yang
menyetujui komite otomatis tiap hari **jam 06:00 tanpa pengguna sama sekali** dipertahankan, dan
apakah `pyBypassActivityAuthentication=true` pada agent diterima.

**Bagaimana kegagalan job diketahui** — notifikasi, catatan, atau dashboard — juga belum
ditetapkan; di sistem lama tidak ada bukti mekanisme apa pun untuk itu.

# 13. Strategi Testing

Pengujian di project ini punya satu tugas yang lebih penting dari biasanya: **membuktikan bahwa
sistem baru berperilaku sama dengan Pega** (Migration Strategy P-5). Tanpa itu, setiap perbedaan
hasil akan diperdebatkan tanpa cara menyelesaikannya.

> **Diperbarui v2.0 (2026-09-14).** Bab ini menyerap tujuh keputusan baru: perkakas uji kesetaraan
> menjadi modul `S-8` di gelombang 1 (`D-42`), lingkungan dan pelaksananya ditetapkan (`D-53`),
> kewenangan menyetujui selisih dipagari (`D-54`), daftar perbaikan eksplisit `P-5` bertambah dari
> 4 menjadi **13 butir** (`D-49`), dua modul tanpa baseline Pega diukur dengan kontrak (`D-56`),
> `BRD §21.4` dicabut untuk empat modul (`D-55`), dan peran penguji gerbang 2 ditetapkan per
> kelompok modul (`D-60`).

---

## 1. Kenapa pengujian tidak bisa ditawar di sini

Tiga alasan yang khas project ini:

1. **Tim sedang belajar tiga hal baru sekaligus** (D-09). Test adalah jaring pengaman yang
   menangkap kesalahan sebelum sampai ke pengguna.
2. **Perilaku wajib setara dengan Pega.** Satu-satunya cara membuktikannya adalah menuliskan
   aturan bisnis sebagai test yang bisa dijalankan berulang.
3. **Database akan berpindah dari Oracle ke PostgreSQL** (D-01). Test suite yang lengkap adalah
   yang membuktikan perpindahan itu tidak mengubah apa pun. Tanpanya, cutover database menjadi
   lompatan tanpa jaring.

---

## 2. Bentuk piramida

```
        ╱╲          Uji Ujung-ke-Ujung  — sedikit, alur terpenting saja
       ╱──╲
      ╱────╲        Uji Integrasi       — sedang, terhadap database nyata
     ╱──────╲
    ╱────────╲      Uji Aturan Bisnis   — BANYAK, cepat, tanpa infrastruktur
   ╱──────────╲
```

Lapisan terbesar sengaja bukan "unit test" melainkan **uji aturan bisnis**. Perbedaannya bukan
istilah: unit test cenderung menguji fungsi kecil apa pun, sedangkan uji aturan bisnis menguji
**aturan yang bisa disebutkan dalam kalimat bisnis**. Yang kedua jauh lebih berguna saat
membuktikan kesetaraan dengan Pega.

---

## 3. Uji aturan bisnis

**Sasaran:** setiap aturan di `02-BUSINESS-UNDERSTANDING.md` §3 dan setiap invarian di
`05-DOMAIN-MODEL.md` §2 punya test tersendiri.

Berjalan **tanpa database, tanpa jaringan, tanpa berkas** — memakai fake di balik seam
(Future Architecture §3). Ini yang membuatnya cepat dan bisa dijalankan setiap kali menyimpan
berkas.

### 3.1 Cakupan wajib

| Kelompok | Kasus minimum |
|---|---|
| Aturan tanggal | 8 aturan × (lolos, ditolak, tepat di batas) |
| Aturan duplikasi | umum dan varian PA dengan penyebab kerugian `12002` |
| Spreading | tepat 100% · `99,9999` (lolos) · `99,99` (**ditolak** — selisih terencana `D-49` butir 1) · `199.99` (**ditolak**) · kurang · lebih · Fac Out tanpa Fac Offer · Group Panel `003` tanpa Object Name · Ex-Gratia mengubah `OR`→`ORS` |
| Kelengkapan | penyebab kerugian (dan pengecualian Travel) · Nomor SLIK untuk SPK · hubungan tertanggung "lain-lain" |
| Ambang nilai | Large Losses > 1 miliar · ambang komite per lini bisnis |
| Penjenjangan komite | setiap kombinasi matriks nilai × jenis bisnis |
| Status | seluruh transisi yang sah, dan penolakan transisi yang tidak sah |
| Otorisasi | setiap peran terhadap setiap aksi |
| Waktu | perhitungan hari kalender di sekitar tengah malam WIB |

**Kasus "tepat di batas" tidak boleh dilewatkan.** Aturan seperti "Tanggal Lapor ≤ DOL + 7 hari"
paling sering salah tepat pada hari ketujuh — dan sistem lama menangani ini dengan penambahan
7 jam manual yang membuat batasnya bergeser. Inilah tempat perbedaan hasil paling mungkin muncul.

### 3.2 Gaya penulisan
Nama test menyebutkan **aturannya**, bukan nama fungsinya. Nama yang baik terbaca sebagai
kalimat bisnis, sehingga daftar test menjadi dokumentasi aturan yang selalu mutakhir.

---

## 4. Uji integrasi

**Sasaran:** membuktikan lapisan adapter benar-benar bekerja — SQL sah, pemetaan tipe benar,
transaksi berperilaku sesuai harapan.

| Aspek | Aturan |
|---|---|
| Database | **Database nyata**, bukan tiruan. SQL yang tidak dijalankan terhadap database nyata tidak terbukti sah |
| Dijalankan terhadap | **Oracle dan PostgreSQL keduanya** |
| Data awal | Disiapkan dan dibersihkan per test |
| Cakupan | Setiap query di berkas `.sql` minimal dijalankan sekali |

**Menjalankan uji integrasi terhadap kedua database adalah inti dari D-20.** Inilah satu-satunya
mekanisme yang benar-benar membuktikan SQL portabel — bukan disiplin penulisan, bukan review,
melainkan test yang gagal ketika seseorang menulis `NVL`. Bila hanya diuji terhadap Oracle,
ketidakportabelan baru ditemukan saat cutover, dan pada saat itu memperbaikinya sangat mahal.

Uji integrasi juga mencakup adapter sistem eksternal terhadap server tiruan, untuk memastikan
penanganan batas waktu, retry, dan kegagalan berperilaku benar.

---

## 5. Uji ujung-ke-ujung

Sedikit saja, hanya untuk alur yang bila rusak berarti sistem tidak bisa dipakai:

1. Login sampai membuka inbox
2. Registrasi klaim lengkap sampai tersimpan
3. Estimasi sampai akseptasi melewati komite
4. Unggah dokumen
5. Menjalankan laporan dan mengunduh hasilnya

Dijalankan terhadap staging. Sengaja dibatasi jumlahnya karena uji ujung-ke-ujung lambat dan
rapuh; menambah banyak akan membuat tim mengabaikan hasilnya ketika sering gagal karena alasan
yang tidak berhubungan.

---

## 6. Uji kesetaraan dengan Pega

Khas project migrasi, dan inilah yang menjawab P-5.

| Cara | Kapan | Isi |
|---|---|---|
| **Perbandingan hasil baca** | Tahap 2 migrasi | Query yang sama dijalankan di Pega dan Go terhadap data staging; hasilnya dibandingkan baris per baris. Perbedaan wajib dijelaskan sebagai bug atau sebagai perbaikan yang disengaja |
| **Perbandingan hasil validasi** | Tahap 3 | Data registrasi yang sama dimasukkan ke kedua sistem; daftar pesan kesalahannya dibandingkan |
| **Perbandingan perhitungan** | Tahap 3–5 | Perhitungan spreading, konversi kurs, dan nilai settlement dibandingkan hasilnya |
| **Rekonsiliasi harian** | Selama masa paralel | Jumlah klaim, total nilai akseptasi, dan jumlah penugasan dibandingkan antar sistem |

**Setiap perbedaan wajib punya kesimpulan tertulis:** bug di sistem baru, atau perbaikan yang
disengaja terhadap perilaku lama. Tidak boleh ada perbedaan yang dibiarkan tanpa penjelasan —
karena satu perbedaan yang tidak dijelaskan akan menjadi alasan meragukan seluruh hasil
perbandingan.

### 6.1 Perkakas pelaksananya: modul `S-8`

Uji kesetaraan tidak dijalankan manual. Ia dikerjakan **modul `S-8` Perkakas Uji Kesetaraan**,
yang masuk **gelombang 1** karena memblokir gerbang 1 setiap modul lain (`D-42`).

| Hal | Ketetapan |
|---|---|
| **Lingkungan** | **Pega staging vs Go staging**, atas **salinan data produksi** — bukan data buatan, dan **tidak pernah menembak produksi** (`D-53`) |
| **Pelaksana** | tim pengembang |
| **Keluaran wajib** | setiap selisih **diklasifikasikan**, bukan sekadar dilaporkan: terpetakan ke butir `P-5` yang mana, atau tidak terpetakan (`D-54`) |

**Alasan memakai data produksi, bukan data buatan:** cacat yang ditemukan pada verifikasi Fase 1 —
toleransi spreading berupa pencocokan substring, kurs yang mengembalikan `1`, `IDSALVAGE = NULL` —
**muncul dari data nyata yang tidak akan terpikir dibuat**.

### 6.2 Kewenangan menyetujui selisih

| Jenis selisih | Perlakuan |
|---|---|
| Cocok dengan salah satu dari **13 butir `P-5`** | **lolos otomatis**, cukup dicatat |
| **Di luar 13 butir itu** | wajib **persetujuan Work Owner secara tertulis** sebelum modul dinyatakan lulus |

Ketiga belas butir sudah diputuskan eksplisit di `D-49`; meminta persetujuan ulang per modul
menambah beban tanpa menambah kendali. Yang menuntut perhatian adalah selisih **yang tidak
terduga** — itulah yang dipagari.

### 6.3 Selisih yang sudah dapat diperkirakan sekarang

Tiga perubahan yang diputuskan akan **pasti** memunculkan selisih, dan harus dinyatakan di muka
sebagai perbaikan terencana — bukan ditemukan sebagai kejutan:

| Selisih | Sebab | Rujukan |
|---|---|---|
| Seluruh data historis **valuta asing** berbeda | basis kurs berubah menjadi **kurs tanggal kejadian** | `D-48`, `ADR-0015` |
| Klaim dengan total spreading di luar toleransi kini **ditolak** | pencocokan substring diganti `ROUND(SUM(share),4) BETWEEN 99.9999 AND 100.0001` | `D-51`, `ADR-0016` |
| Laporan yang dulu terpotong **500 baris** kini utuh | batas `pyMaxRecords=500` dihapus | `ADR-0011` |

### 6.4 Dua hal yang **tidak boleh** dibandingkan begitu saja

1. **Jumlah baris tabel.** `D-66` menetapkan **soft delete menyeluruh** — yang terhapus tetap
   tidak muncul bagi pengguna, tetapi barisnya tetap ada. Perbandingan berbasis `COUNT(*)` pada
   tabel akan **selalu berbeda dan bukan indikasi cacat**. Yang dibandingkan adalah **hasil kueri
   sesuai aturan bisnis**.
2. **Perilaku saat gagal pada `B-4` dan `B-9`.** Sistem lama menempuh sembilan `COMMIT` dan dapat
   meninggalkan data setengah jalan; sistem baru membungkusnya satu transaksi (`D-68`,
   `ADR-0007`). Kegagalan menghasilkan keadaan akhir yang berbeda **secara sengaja**.

---

## 7. Pengujian frontend

| Jenis | Cakupan |
|---|---|
| Komponen | Pustaka komponen baku (U-2) — terutama `DataTable` dan komponen form. Dipakai ratusan kali, jadi kesalahan di sini berlipat ganda |
| Integrasi | Alur per fitur dengan API tiruan |
| Tipe | `tsc --noEmit` di CI — mode ketat, `any` dilarang |

Komponen khusus fitur diuji lebih ringan; investasi pengujian diarahkan ke komponen bersama
karena di situlah **leverage**-nya.

---

## 8. Uji beban

Dijalankan sebelum go-live, terhadap staging dengan data sebesar produksi (D-10: puluhan juta
baris).

| Skenario | Sasaran |
|---|---|
| Inbox dengan data penuh | Waktu tampil di bawah target NFR |
| Pencarian klaim rentang lebar | Tidak menghabiskan connection pool |
| Laporan besar bersamaan | Tidak mengganggu transaksi pengguna lain |
| 300 pengguna bersamaan | Waktu respons tetap dalam target |
| Export besar | Memori tidak meledak — harus streaming, bukan dimuat seluruhnya |

**Uji beban wajib memakai data sebesar produksi.** Query yang cepat terhadap seribu baris bisa
sangat lambat terhadap sepuluh juta baris, dan perbedaannya tidak akan terlihat pada data uji
yang kecil. Ini risiko terbesar pada profil beban D-10.

---

## 9. Yang wajib lulus sebelum merge

| Pemeriksaan | Blokir merge |
|---|---|
| Seluruh uji aturan bisnis lulus | Ya |
| Seluruh uji integrasi lulus (Oracle **dan** PostgreSQL) | Ya |
| Lint backend dan frontend bersih | Ya |
| Aturan ketergantungan antar lapisan tidak dilanggar (`depguard`) | Ya |
| Tidak ada pola SQL terlarang | Ya |
| Tidak ada kesalahan tipe TypeScript | Ya |
| Uji ujung-ke-ujung lulus | Sebelum rilis, tidak setiap merge |

---

## 10. Dua gerbang penerimaan per modul

Setiap modul melewati **dua gerbang** sebelum dinyatakan pindah:

| Gerbang | Isi | Berlaku untuk |
|---|---|---|
| **Gerbang 1** | uji kesetaraan otomatis oleh `S-8` | seluruh modul, **kecuali** `F-3` dan `S-5` |
| **Gerbang 2** | UAT pengguna bisnis | **modul bisnis saja** |

**Peran penguji gerbang 2** (`D-60`):

| Kelompok modul | Gerbang 2 | Penguji |
|---|---|---|
| **Modul bisnis** — `B-1`…`B-14`, `S-1`…`S-4`, `S-6`, `S-7`, `U-1`, `U-3`…`U-6` | **berlaku** | peran bisnis pemakai inbox/layar modul itu |
| **Modul fondasi** — `F-1`…`F-5`, `S-5`, `S-8`, `U-2` | **tidak berlaku** | gerbang 1 + **persetujuan Work Owner** |

Memaksakan gerbang 2 pada modul yang tidak punya peran pemakai akan membuat tiket fondasi macet
di gerbang yang tidak dapat dilewati siapa pun — tidak ada pengguna bisnis yang membuka layar
"Akses Data".

### 10.1 Dua modul yang tidak punya baseline Pega

`F-3` dan `S-5` **tidak dapat diuji kesetaraannya** karena tidak ada yang bisa dibandingkan:
HCC/HCQ tidak meninggalkan jejak apa pun di export, dan sistem lama tidak mencatat perubahan nilai
sama sekali. Untuk keduanya, gerbang 1 diganti **uji fungsional terhadap kontrak** (`D-56`):

| Modul | Kontrak yang menjadi acuan | Pemilik kontrak | Status kontrak |
|---|---|---|---|
| **F-3** Identitas & Akses | kontrak API HCC/HCQ — field request/response login, kode galat, timeout, endpoint refresh/validasi | pemilik API HCC/HCQ | **belum ada** |
| **S-5** Jejak Audit | daftar peristiwa wajib audit beserta field yang harus tercatat | Compliance | **belum ada** |

Menghapus gerbang 1 begitu saja **ditolak**: keduanya justru modul paling sensitif — `F-3` adalah
otorisasi, `S-5` adalah jejak audit — dan UAT tidak memeriksa hal yang tidak terlihat di layar.

**Konsekuensi yang harus diterima:** sampai kedua kontrak diterima, `F-3` dan `S-5` **tidak dapat
lulus gerbang apa pun**.

### 10.2 `BRD §21.4` dicabut untuk empat modul

Setelah `Database/` diterima, penghalang yang mendasari `BRD §21.4` sudah tidak berlaku untuk
sebagian modul (`D-55`):

| Modul | Status | Sisa penghalang |
|---|---|---|
| **B-7** Komite | **lepas** | PA dan Travel di atas Rp 200.000.000 tidak punya baris master |
| **B-9** PLA/DLA | **lepas** | status `VALID`/`INVALID` `GET_GROUPBUSINESS_XOL`; penulisan ulang procedure ber-9-`COMMIT` |
| **B-10** Akseptasi | **lepas** | 3 activity hilang: `InsertDataAkseptasiToLeader`, `InsertLogKasir_act`, `TransferCashierDataASM_act` |
| **B-12** Salvage | **lepas** | `SET_ATTACHFILETEMPSALVAGE`; hitungan DBA atas baris ber-`IDSALVAGE` NULL |
| **B-5** Settlement | **tetap terikat** | isi `POOLDATA.GCNM_FEE_SCALE` (17 pita) · isi `m_currencystandard` · `BrowseT_Claim_Adjustment_SQL` |

Usulan revisi `BRD §21.4` menunggu persetujuan Work Owner; pencabutannya dinyatakan lewat
`ADR-0028`, bukan dengan menyunting BRD diam-diam.

---

## 11. Catatan atas tekanan jadwal

Target D-30 sangat ketat, dan pada project seperti ini pengujian adalah hal pertama yang
tergoda untuk dikorbankan.

Bila pengujian dipangkas, yang **paling akhir** boleh dikorbankan adalah:
1. **Uji aturan bisnis** — inilah yang membuktikan kesetaraan dengan Pega; tanpanya tidak ada
   dasar untuk menyatakan migrasi berhasil.
2. **Uji integrasi terhadap kedua database** — tanpanya, janji SQL portabel (D-20) tidak
   terbukti dan cutover ke PostgreSQL menjadi taruhan.

Uji ujung-ke-ujung dan sebagian uji komponen frontend jauh lebih bisa ditunda tanpa kehilangan
dasar pembuktian.

# 14. Non-Functional Requirements, Performa, Skalabilitas, dan Maintainability

---

## 1. Profil beban

Ini yang menentukan seluruh keputusan di dokumen ini:

| Aspek | Angka | Sumber |
|---|---|---|
| Pengguna aktif harian | 200–300 | Pemilik project |
| Klaim baru | Ribuan per bulan | D-10 |
| Data historis | **Puluhan juta baris** | D-10 |
| Ketersediaan | **24/7** | D-27 |
| Lingkungan | 2 VM on-premise | D-08 |

> **Profilnya: data besar, konkurensi rendah.**
>
> 200–300 pengguna bersamaan bukan beban berat bagi Go — satu instance saja mampu menanganinya
> tanpa kesulitan. Yang berat adalah **query terhadap puluhan juta baris**, terutama inbox
> berkolom banyak dan laporan lintas periode.
>
> Seluruh upaya optimasi diarahkan ke **volume data**, bukan ke jumlah permintaan per detik.
> Salah membaca ini akan membuat tim mengoptimalkan hal yang tidak menjadi hambatan.

---

## 2. Target Non Functional

### 2.1 Performa

| Jenis operasi | Target (persentil 95) | Catatan |
|---|---|---|
| Buka layar sederhana | < 1 detik | — |
| Inbox dan pencarian | < 3 detik | Terhadap data produksi penuh |
| Simpan registrasi klaim | < 3 detik | Termasuk seluruh validasi |
| Perhitungan spreading | < 1 detik | — |
| Laporan interaktif | < 10 detik | — |
| Export besar (PDF/Excel/CSV) | Asinkron | Tidak menahan pengguna; diberi tahu saat selesai |
| Pemanggilan sistem eksternal | Batas waktu 30 detik | Kegagalan ditangani, bukan menggantung |

### 2.2 Ketersediaan

| Aspek | Target | Konsekuensi |
|---|---|---|
| Ketersediaan | 24/7 (D-27) | Minimal 2 instance, rolling deployment |
| Downtime terencana | Nol untuk deployment aplikasi | Migrasi skema wajib backward-compatible |
| Downtime tak terencana | Sekecil mungkin | Kegagalan satu instance tidak menghentikan layanan |
| Pemulihan bencana | Mengikuti standar korporat (D-29) | Perlu diperiksa apakah sejalan dengan 24/7 |

### 2.3 Keamanan
Lihat `11-SECURITY.md`. Ringkasnya: autentikasi lewat HCC/HCQ, otorisasi milik aplikasi,
parameter binding tanpa perkecualian, data sensitif tidak masuk log, jejak audit append-only.

### 2.4 Auditabilitas
Wajib per D-28. Setiap perubahan bernilai bisnis tercatat permanen: siapa, kapan, nilai sebelum,
nilai sesudah. Retensi mengikuti ketentuan Compliance — masih terbuka.

---

## 3. Strategi performa

### 3.1 Hambatan yang sebenarnya

Diurutkan dari yang paling mungkin menjadi masalah:

| Peringkat | Hambatan | Penanganan |
|---|---|---|
| 1 | Query inbox terhadap puluhan juta baris | Index yang tepat + keyset pagination + index parsial |
| 2 | Laporan lintas periode panjang | Pool koneksi terpisah + eksekusi asinkron + streaming |
| 3 | Query N+1 saat memuat objek dan coverage | Muat sekaligus per batch, tidak per baris |
| 4 | Export besar dimuat seluruhnya ke memori | Streaming baris demi baris |
| 5 | Pemanggilan sistem eksternal yang lambat | Batas waktu, circuit breaker, pemanggilan asinkron |
| — | Jumlah permintaan per detik | **Bukan hambatan** pada 200–300 pengguna |

### 3.2 Aturan performa yang mengikat

1. **Paginasi selalu server-side.** Tidak pernah mengambil seluruh baris lalu memotongnya di
   aplikasi maupun di browser.
2. **Keyset pagination untuk inbox dan pencarian.** `OFFSET` bernilai besar memaksa database membaca dan
   membuang setengah juta baris.
3. **Tidak ada `COUNT(*)` atas tabel besar sebagai bagian permintaan biasa.** Frontend memakai
   pola "muat lebih banyak", bukan nomor halaman dengan total.
4. **Tidak ada query di dalam perulangan.** Muat sekaligus per batch.
5. **`SELECT` menyebutkan kolom.** Mengambil kolom `CLOB` yang tidak dipakai sangat mahal.
6. **Export dan laporan besar berjalan asinkron dan streaming.** Memori tetap datar berapa pun
   jumlah barisnya.
7. **Pool koneksi terpisah untuk laporan** agar satu laporan berat tidak menghabiskan koneksi
   transaksi.
8. **Setiap query baru wajib diperiksa rencana eksekusinya** terhadap data sebesar produksi
   sebelum merge.

> **Kapasitas laporan tidak punya dasar historis (2026-09-14).** `pyMaxRecords=500` terpasang pada
> **54 dari 56** laporan sistem lama, sehingga **kebutuhan export bervolume besar belum pernah
> benar-benar dilayani**. Menghapus batas itu adalah **penambahan kemampuan**, bukan penyalinan —
> dan rancangan kapasitasnya dibuat **tanpa data historis yang sahih**, karena angka pemakaian
> selama ini selalu terpotong di 500.
>
> Dua hal karenanya belum dapat ditetapkan secara terukur: **berapa baris maksimum yang wajib
> dilayani satu export**, dan apakah export besar dijalankan serentak dengan permintaan pengguna
> atau **diantrekan** lalu diberitahukan saat selesai. Keduanya pertanyaan terbuka `ADR-0011`, dan
> tanpa jawabannya `S-2` tidak dapat dinyatakan selesai secara terukur.
>
> Masalah nyata yang terukur justru di sisi antarmuka: **3.189 grid terikat page list klipboard**.

Aturan terakhir yang paling sering diabaikan, padahal paling murah. Query yang cepat terhadap
seribu baris bisa sangat lambat terhadap sepuluh juta — dan perbedaannya tidak akan terlihat di
lingkungan pengembangan.

### 3.3 Caching

Sistem lama berjalan **tanpa caching sama sekali** (7 data page seluruhnya `refresh=never`).
Kita tidak menambahkan kerumitan caching kecuali terbukti perlu.

| Data | Strategi |
|---|---|
| Master data (cabang, penyebab kerugian, mata uang, jenis treaty) | **Cache in-process**, disegarkan berkala atau saat diubah. Jarang berubah, sering dibaca |
| Izin pengguna | Cache in-process berumur pendek (menit) agar pencabutan hak cepat berlaku |
| Data klaim | **Tidak di-cache.** Harus selalu mutakhir |
| Hasil laporan | Tidak di-cache di awal; dipertimbangkan bila terbukti perlu |

**Tidak memakai cache terdistribusi (Redis).** Dengan hanya dua instance dan master data yang
kecil, cache in-process di masing-masing instance sudah memadai dan jauh lebih sederhana untuk
dioperasikan di VM on-premise.

---

## 4. Scalability

### 4.1 Skala saat ini
Dua instance sudah lebih dari cukup untuk 200–300 pengguna. Keduanya ada **karena tuntutan
ketersediaan 24/7** (D-27), bukan karena kebutuhan kapasitas.

### 4.2 Bila beban bertambah

Urutan langkah, dari yang termurah:

| Langkah | Kapan | Biaya |
|---|---|---|
| 1. Perbaiki query dan index | Selalu lebih dulu | Rendah |
| 2. Tambah cache master data | Bila master sering dibaca | Rendah |
| 3. Tambah instance aplikasi | Bila CPU aplikasi jenuh | Rendah — aplikasi stateless |
| 4. Read replica untuk laporan | Bila laporan mengganggu transaksi | Sedang |
| 5. Partisi tabel per tahun | Bila tabel klaim/audit sangat besar | Sedang |
| 6. Arsip data lama | Bila data historis menghambat | Sedang |

**Aplikasi stateless sejak awal** membuat langkah 3 hanya soal menambah VM — tidak perlu
perubahan kode. Inilah manfaat nyata dari syarat stateless yang sudah dituntut D-27.

### 4.3 Yang sengaja tidak disiapkan
Sharding, microservices, message broker, autoscaling. Seluruhnya menambah kerumitan operasional
yang nyata tanpa menyelesaikan masalah yang kita punya. Bila kelak dibutuhkan, seam yang sudah
ada (Future Architecture §3) membuatnya bisa ditambahkan tanpa membongkar domain.

---

## 5. Maintainability

Ini yang menentukan apakah sistem masih bisa dirawat tiga tahun lagi. Untuk tim di D-09,
maintainability lebih penting daripada kecanggihan.

### 5.1 Yang membuatnya terawat

| Hal | Bagaimana | Menghilangkan masalah lama |
|---|---|---|
| **Satu aturan hidup di satu tempat** | Aturan ketergantungan antar lapisan | Aturan tersebar di activity, SQL, dan stored procedure |
| **Bahasa yang seragam** | `CONTEXT.md` dipakai di kode, API, dan UI | Alias kolom menyesatkan; `Adjustment` yang berarti nilai penyelesaian |
| **Nilai bisnis dapat diubah tanpa deploy** | Master data (D-15) | Email dan ambang di-hardcode |
| **Satu komponen tabel** | Pustaka komponen baku (U-2) | 268 grid dengan pola berulang |
| **Struktur yang dapat ditebak** | Struktur folder preskriptif | — |
| **Aturan ditegakkan otomatis** | Lint, `depguard`, pemeriksaan pola SQL | Konvensi yang hanya ada di dokumen akan dilanggar |
| **Test sebagai dokumentasi aturan** | Setiap aturan bisnis punya test bernama kalimat bisnis | Aturan hanya diketahui dari membaca 137 step |

### 5.2 Ukuran yang dipantau

| Ukuran | Batas | Tindakan bila terlampaui |
|---|---|---|
| Panjang fungsi | ~80 baris | Pecah |
| Panjang berkas | ~400 baris (Go), ~200 baris (komponen React) | Pecah |
| Ketergantungan antar lapisan | Nol pelanggaran | Merge diblokir |
| Cakupan test aturan bisnis | Seluruh aturan terdokumentasi punya test | Lengkapi sebelum merge |
| Jumlah dependensi pihak ketiga | Ditinjau setiap penambahan | Butuh alasan tertulis |

### 5.3 Dokumentasi yang harus tetap hidup

| Dokumen | Diperbarui saat |
|---|---|
| `CONTEXT.md` | Ada istilah domain baru atau berubah artinya |
| `00-DECISION-LOG.md` | Ada keputusan arsitektur baru |
| Kontrak OpenAPI | Ada perubahan API |
| Dokumen Steering terkait | Ada perubahan strategi |

Dokumentasi yang tidak diperbarui lebih buruk daripada tidak ada dokumentasi, karena orang
mempercayainya. Karena itu daftar di atas sengaja pendek — hanya yang benar-benar akan dibaca.

# 15. Analisis Risiko

Risiko yang teridentifikasi dari analisis source dan hasil discovery. Diurutkan berdasarkan
dampak terhadap keberhasilan project.

Skala: **Dampak** dan **Kemungkinan** = Rendah · Sedang · Tinggi

> **Diperbarui v2.0 (2026-09-14).** Jumlah risiko menjadi **19**, bukan 12: `R-13`…`R-15` lahir
> dari penyusunan BRD, dan `R-16`…`R-19` dari verifikasi bukti Fase 2 (`D-38`). Tiga risiko
> berubah status setelah export bertambah pada 2026-09-09 (`D-45`): **`R-02` dan `R-06` tertutup**,
> **`R-01` sebagian tertutup**, **`R-04` turun** dari penghalang menjadi verifikasi. Ringkasan
> perubahan ada di [`21-RIWAYAT-REVISI.md`](21-RIWAYAT-REVISI.md).

---

## R-01 · Source procedure & function database tidak tersedia

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Terjadi — sudah pasti, bukan kemungkinan |
| **Status** | **Sebagian tertutup (2026-09-14)** — 62 dari 64 objek sudah diterima |

**Masalahnya.** D-02 menetapkan seluruh logika stored procedure ditulis ulang di Go. Tetapi
**source-nya tidak ada di export XML** — kita hanya melihat nama dan daftar parameternya dari
pemanggilan di SQL. Isinya belum pernah dilihat siapa pun di tim ini.

Jumlah terverifikasi: **56 stored procedure + 8 function lokal = 64 objek** yang harus diminta ke
DBA. Ditambah 6 function remote lewat DB Link yang ditangani terpisah di R-03.

**Yang sudah berubah (`D-45`).** Folder `Database/` diterima pada 2026-09-09 dan kini memuat
**55 berkas `.prc` + 8 berkas `.fnc`**. **62 dari 64 objek** yang diminta sudah ada. Dua yang
belum: **`SET_ATTACHFILETEMPSALVAGE`** (menghalangi `B-12`) dan **`UPDATEPREMIUMTEMPLATE`**
(menghalangi `B-9`).

**Yang membuat risiko ini belum tertutup penuh.** 62 berkas yang datang **memanggil 12 objek
lain yang tidak ikut dikirim**, terberat **`UPDATE_LOG_KONVERSI` (162 pemanggilan)**, `GETNEWID`
(42), `PKG_COUNTER_PRODUCTION` (25), `PROCESSQUEUEDIRECT` (10). Logikanya tetap harus dibaca
untuk ditulis ulang, meski objeknya kelak ditinggalkan (`D-68`, `ADR-0007`).

> **Daftar lengkapnya ada di [`19-GAP-EXPORT-DETAIL.md`](19-GAP-EXPORT-DETAIL.md)** — beserta
> jumlah parameter, rule pemanggil, modul terdampak, dan query siap pakai untuk menarik source-nya.

Yang terdampak antara lain `PEGA_JSON_KLAIM_PNC`, `INSERT_PLADLA`, `INSERT_SURVEYORLIST`,
`CONVERTJSONPRODUCTION`, `INSERTDATAKOMITELIST`, `INSERT_SALVAGE_DETAILS`,
`PNC_INSERT_EMAIL_ADJUSTER`, `GET_POSISI_PROGRESS_PNC`, `ADD_NEWMASTERVIRTUALACCOUNT`.

**Akibatnya.** Modul B-5 (settlement), B-7 (komite), B-9 (PLA/DLA), B-10 (akseptasi), dan B-12
(salvage) **tidak dapat diselesaikan** sampai isinya diketahui. Ini bukan penundaan kecil —
kelimanya adalah jantung nilai uang klaim.

**Penanganan.**
1. Minta source lengkap ke DBA **pada hari pertama** — ini di jalur kritis.
2. Sambil menunggu, kerjakan modul yang tidak bergantung: fondasi, registrasi, inbox, dokumen.
3. Bila tetap tidak tersedia, satu-satunya jalan adalah membaca perilakunya lewat perbandingan
   masukan dan keluaran di lingkungan staging — jauh lebih lambat dan tidak menjamin kelengkapan
   kasus tepi.

---

## R-02 · Job terjadwal tidak diketahui

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Terjadi |
| **Status** | **TERTUTUP (2026-09-09)** — `D-57` |

**Masalahnya (saat risiko ini ditulis).** Pemilik project mengonfirmasi ada job terjadwal, tetapi
belum tahu detailnya (D-17). Export XML **tidak memuat rule Agent maupun Queue Processor Pega**,
sehingga tidak dapat dianalisis dari source sama sekali.

**Bagaimana ditutup.** Pada 2026-09-09 export bertambah dan tiga folder yang sebelumnya tidak ada
muncul: `Job Scheduler/` (5 berkas), `Agents/` (1), `Service REST/` (4). Daftar lengkap job
terjadwal, terverifikasi dari `pyRuleName` dan `pyActivityName` masing-masing berkas:

| Job | Frekuensi | Jam mulai | Activity target | Target ada? |
|---|---|---|---|---|
| `JOBForKomiteKlaimPNC` | Daily | **06:00:00** | **`AutoAcceptKomite`** | ✅ |
| `JobHitungDeadlineToTemporaryCLose` | **Weekly** | 23:43:00 | `JobTemporaryCloseClaimPNC` | ✅ |
| `JobSendAutoLODKlaimPersonal` | Daily | 20:54:00 | `Act_SendAutoLODKlaimPersonal` | ✅ |
| `PNCMyReportKlaim3` | Daily | 08:00:00 | `ReportAI_Act` | ✅ |
| `ProcessClaimKredit` | Daily | 10:00:00 | `CreateClaimCredit_Table` | ✅ |

Seluruhnya `pyIsEnabled=true`, `pyApplicableTo=Cluster`, `pyNodeTypesText=BackgroundProcessing`.
Ditambah satu agent: `Agents/TATReportAgent-Agents.xml`, `pyEnable=true`,
`pyTriggerInterval=1800` (30 menit), `pyBypassActivityAuthentication=true`.

**Konsekuensi.** Modul `S-6` Penjadwalan naik dari **TERHALANG** menjadi **SEBAGIAN**. Yang
tersisa bukan lagi soal artefak melainkan soal rancangan dan kewenangan — dipindahkan ke
`ADR-0022`, yang masih berstatus `Proposed`:

1. Siapa yang menjalankan job bila aplikasi hidup di dua instans (`D-27`), karena Go tidak punya
   padanan otomatis untuk `pyApplicableTo=Cluster`.
2. **`AutoAcceptKomite` menyetujui komite otomatis setiap hari jam 06:00**, melewati kontrol menu
   `D-59` — apakah dipertahankan.
3. Tiga job ber-`pyDescription = "job jalan 2 menit"` sementara konfigurasinya `Daily`/`Weekly`.

---

## R-03 · Enam API pengganti DB Link belum ada

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Tinggi |
| **Status** | Terbuka |

**Masalahnya.** D-25 mengganti 64 pemakaian DB Link dengan pemanggilan API. Tetapi API-API itu
kemungkinan besar belum ada, dan yang membangunnya adalah **tim lain dengan jadwal sendiri**.

Terdampak: data HRD, jam kerja, pembayaran GL, master sales, status buka proteksi, dan pengguna
lintas sistem.

Angka terverifikasi: **64 pemakaian · 6 DB Link · 28 objek remote · 27 rule**.

> **Inventaris lengkapnya ada di [`20-DETAIL-KOMITE-DBLINK.md`](20-DETAIL-KOMITE-DBLINK.md)** —
> objek remote per DB Link, proses bisnis yang terdampak, 27 rule pemakai, usulan 6 API pengganti,
> serta mana yang boleh dijembatani salinan berkala dan mana yang wajib API sungguhan.

Dua titik paling kritis: **`GetIDCabang`** (dipanggil 13 aktivitas, termasuk jalur registrasi) dan
**`GENERAL.MST_BUKA_PROTEKSI`** — satu-satunya objek remote yang **menulis**, dipanggil ke tiga
sistem berbeda, sehingga penggantinya wajib API idempoten dan tidak bisa disalin berkala.

**Akibatnya.** Ketergantungan pada pihak di luar kendali tim, tepat pada project dengan jadwal
sangat ketat (D-30).

**Penanganan.**
1. Inventarisasi kebutuhan per sistem dan sampaikan ke tim pemilik **sekarang juga**.
2. Untuk data yang jarang berubah — master sales, cabang, agen — pakai **salinan yang
   disegarkan berkala** sebagai jembatan, di balik seam yang sama sehingga penggantian ke API
   nyata nanti tidak menyentuh kode domain.
3. Untuk data yang harus mutakhir, modul yang bergantung padanya tertahan — dan itu harus
   disampaikan ke manajemen sebagai konsekuensi jadwal, bukan diserap diam-diam.

---

## R-04 · Tiga router penugasan tidak ada di export

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Terjadi |
| **Status** | **Turun menjadi verifikasi (2026-09-14)** — algoritmanya sudah dapat direkonstruksi |

**Masalahnya.** `PNCAdminRouter`, `PNCTeknikRouter`, dan `RouterRCLDokter` dirujuk oleh
Register_Flow tetapi **tidak ada di export**. Ketiganya buatan sendiri (ruleset GCNMFW) dan
menentukan **siapa menerima tugas apa** — inti dari modul B-6.

**Yang sudah berubah.** Algoritma pembagian beban **ditemukan di tempat lain** dan tidak lagi
perlu ditebak: `RDB List/BrowsePICRandomTeam-SQL.xml:39-40` memilih petugas dengan
**`ORDER BY counter_quota ASC`** — yang paling sedikit bebannya — lalu `AddTJobCounterPIC_SQL`
menaikkan pencacahnya. Risiko ini karena itu **turun dari penghalang menjadi verifikasi**: yang
dibutuhkan bukan lagi artefaknya, melainkan konfirmasi bahwa rekonstruksi ini memang aturan yang
berlaku (`ADR-0019`).

**Catatan yang mempersempit lingkup `B-6`.** Penjenjangan komite ternyata menugaskan ke
**operator bernama**, bukan ke workbasket, dan seluruh export hanya memuat **4 workbasket**
(`T-7`). Model penugasan `B-7` karena itu tidak seluruhnya mengikuti pola Worklist/Workbasket.

Satu router lain juga tidak ada di export, yaitu `ToWorkList`, tetapi itu **bawaan Pega**
(`Pega-ProcessEngine`) dengan perilaku baku — bukan gap nyata dan tidak perlu diminta.

Yang tersedia hanya `KomiteRouter` dan `PNCAdminRouterRCV`, sehingga polanya bisa ditebak
(pemilihan berdasarkan lini bisnis, cabang, dan beban kerja), tapi aturan persisnya tidak.

> **Daftar lengkap 8 router beserta shape yang memakainya ada di
> [`19-GAP-EXPORT-DETAIL.md`](19-GAP-EXPORT-DETAIL.md).**

**Penanganan.** Minta export ketiganya. Bila tidak tersedia, gali aturannya dari tim operasional
dan buktikan dengan membandingkan hasil penugasan pada data historis.

---

## R-05 · Jadwal tidak sepadan dengan ukuran pekerjaan

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Tinggi |
| **Status** | Diterima manajemen (D-30) |

**Masalahnya.** Target seluruh modul selesai akhir September 2026 (D-30), sekitar 3 minggu sejak
analisis ini. Ukuran pekerjaan yang terukur dari source:

| Yang harus dibangun | Ukuran |
|---|---|
| Layar | 74 harness, 269 section, 268 bergrid |
| Logika bisnis | 15.063 step activity |
| Query | 652 rule SQL, 534 KB |
| Procedure & function di-Go-kan | 64 — source belum ada (R-01) |
| Laporan + engine dokumen | 56 report definition |
| Integrasi | **21 REST** + 6 API baru (R-03) |
| Belum ada sama sekali | Tabel auth/authz, tabel penugasan, modul master data, daftar job (R-02) |

Ditambah tim yang harus belajar **Go, React, dan TypeScript sekaligus** (D-09).

Perkiraan berdasarkan ukuran di atas: **9–15 bulan dengan tim 5–6 orang.**

**Status.** Keberatan sudah disampaikan beserta angkanya. Pemilik project menegaskan target
tetap berlaku dan seluruh modul harus pindah. Keputusan ini dihormati dan Migration Strategy
disusun untuknya.

**Penanganan.**
1. Urutan pengerjaan disusun agar pekerjaan berisiko rendah berjalan lebih dulu — sehingga bila
   waktu tidak mencukupi, yang tertinggal adalah bagian yang paling sedikit dampaknya.
2. Tiga hal yang **tidak bisa dipercepat oleh penambahan orang** harus dimulai hari ini:
   menunggu source procedure & function (R-01), menunggu API tim lain (R-03), dan waktu belajar tim.
3. Bila pemangkasan menjadi perlu, dokumen ini menyediakan dasar angka untuk memutuskan bagian
   mana yang ditunda.

---

## R-06 · Arti kode status 1142–1151 tidak diketahui

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Terjadi |
| **Status** | **TERTUTUP (2026-09-09)** — isi master diterima |

**Masalahnya (saat risiko ini ditulis).** `StatusClaim` memakai kode `1142`–`1151` yang artinya
tersimpan di view `V_STS_CLAIM.LSC_NOTE`. **Isi view itu tidak ada di export.** Yang berhasil
terbaca dari kode hanya `1146`, yang di-set ketika pengguna menekan "Back" pada tahap registrasi.

**Bagaimana ditutup.** Isi master diterima sebagai `v_sts_claim.csv`. Hasilnya **memperbaiki
premisnya sendiri**: domain `StatusClaim` bukan 10 kode `1142`–`1151`, melainkan **33 kode
`1134`–`1166`**. Sebelas kode pertama (`1134`–`1144`) membawa penomoran lama `01`–`11`; sisanya
hanya bernomor baru. Contoh: `1142` Rejected Claim · `1143` Close Claim for this object ·
`1144` Cancelled Claim · `1147` Register · `1149` Claim Committee · `1163` Paid ·
`1164` Reopen Claim.

Rentang `1142`–`1151` yang dipakai dokumen-dokumen awal ternyata **sebagian** dari domainnya,
bukan keseluruhannya. Definisi resmi ada di [`CONTEXT.md`](CONTEXT.md) dan `ADR-0018`.

**Catatan atas tebakan yang gugur.** Sebelum master diterima, tiga arti kode sempat disimpulkan
dari pemakaiannya di rule dan **ketiganya salah**: `1143` bukan "status awal" melainkan Close
Claim, `1150` bukan penanda terdaftar melainkan LOD Report, dan `1151` bukan Investigator
melainkan Analyst. Ini alasan konkret mengapa arti kode status tidak boleh disimpulkan dari
pemakaian.

---

## R-07 · 50 activity dipanggil tapi tidak ada di export

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Terjadi |
| **Status** | Terbuka |

**Masalahnya.** Dari 554 activity yang dipanggil, 50 tidak ada di export. Sebagian adalah
bawaan Pega dan tidak masalah, tetapi beberapa jelas berisi logika bisnis:

`SendEmailNotification` (dipanggil **15×**), `SetTicket` (4×), `SetKasir_Act` (3×),
`generatePDF`, `generatePD4ML`, `PNCSalvageHistorySemuaKlaim`, `SendFilePendukungLelangKeSimasBit`,
`InsertDataSlinkOJKIndividu`, `UploadDocumentToGoogleStorage`, `ValidasiSisaTSI`,
`GCNMGetLossAdjuster_Act`, `SendUpdateCIF_act`.

> **Daftar lengkap 50 activity — 45 buatan sendiri dan 5 bawaan Pega — beserta activity pemanggil
> dan modul terdampak ada di [`19-GAP-EXPORT-DETAIL.md`](19-GAP-EXPORT-DETAIL.md).**

`ValidasiSisaTSI` patut diperhatikan khusus — namanya menunjukkan aturan validasi sisa TSI yang
tidak muncul di analisis aturan bisnis manapun.

**Penanganan.** Minta export ke-50 activity tersebut. Prioritaskan `SendEmailNotification`,
`ValidasiSisaTSI`, `SetKasir_Act`, dan `SetTicket`.

---

## R-08 · DDL tabel tidak tersedia

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Terjadi |
| **Status** | Terbuka |

**Masalahnya.** 245 tabel teridentifikasi dari SQL, tetapi **tipe kolom, panjang, index,
constraint, dan foreign key tidak diketahui**. Desain skema baru harus menebak.

Contoh nyata dampaknya: kode lama memotong `ObjectName` pada 3.800 karakter "karena melebihi
batas database" — batas sebenarnya tidak diketahui.

**Penanganan.** Minta DDL lengkap schema `POOLDATA`, `DATAPEGA`, dan `GENERAL`. Sekaligus minta
statistik ukuran tabel — itu yang menentukan strategi index dan partisi (Database Strategy §6).

---

## R-09 · Sistem sumber masih aktif berubah

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Tinggi |
| **Status** | Melekat pada Strangler Fig |

**Masalahnya.** 124 activity diubah pada 2026 dan 180 pada 2024. Pega adalah **sasaran bergerak**:
perubahan yang dibuat tim Pega selama migrasi tidak otomatis ikut ke sistem baru.

**Penanganan.**
1. Sepakati **pembekuan perubahan** pada modul yang sedang dimigrasi.
2. Perubahan mendesak yang tetap harus dilakukan di Pega **wajib dicatat** dan diterapkan juga
   di sistem baru.
3. Rekonsiliasi berkala membandingkan perilaku kedua sistem (Testing Strategy §6).

---

## R-10 · Data ganda antara tabel relasional dan JSON

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Sedang |
| **Status** | Terbuka |

**Masalahnya.** Data klaim yang sama disimpan dua kali: tabel relasional `POOLDATA` dan dokumen
JSON (`JSON_KLAIM`, `JSON_POLIS`). Sinkronisasi dilakukan stored procedure. **Tidak ada mekanisme
yang memeriksa apakah keduanya konsisten.**

**Akibatnya.** Bila keduanya berbeda pada data historis, sistem baru harus memutuskan mana yang
benar — dan keputusan itu memengaruhi hasil laporan dan perhitungan.

**Penanganan.**
1. Jalankan pemeriksaan konsistensi pada data produksi sebelum migrasi.
2. Tetapkan sumber kebenaran secara eksplisit per jenis data.
3. Di sistem baru, satu data hanya disimpan satu kali — snapshot polis sebagai JSON (D-04),
   data klaim sebagai tabel relasional.

---

## R-11 · Tim mempelajari tiga teknologi sekaligus

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Tinggi |
| **Status** | Melekat pada D-09 dan D-23 |

**Masalahnya.** Tim adalah developer Pega yang harus menguasai Go, React, dan TypeScript
bersamaan. D-23 memilih React yang tidak opinionated, sehingga risiko **kode berbeda gaya di
setiap layar** menjadi nyata — inilah alasan React bukan rekomendasi awal saya.

**Penanganan.**
1. Coding Standards yang **mengikat**, bukan anjuran (dokumen 08).
2. Penegakan otomatis lewat lint, `depguard`, dan pemeriksaan pola SQL — bukan hanya code review.
3. Pustaka komponen baku (U-2) dibangun lebih dulu, sehingga layar-layar berikutnya merakit,
   bukan mencipta.
4. Code review wajib pada masa awal, terutama untuk layar pertama tiap jenis.
5. Modul baca dikerjakan lebih dulu (Migration Strategy Tahap 2) — tim belajar pada pekerjaan
   yang kesalahannya tidak merusak data.

---

## R-12 · Zona waktu bergeser saat migrasi data

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Sedang |
| **Status** | Terbuka |

**Masalahnya.** Sistem lama menyimpan waktu dalam GMT dan menambahkan 7 jam secara manual di
banyak tempat. Sistem baru menyimpan UTC dan mengonversi di satu tempat (F-5).

Bila ada data lama yang **sudah** tersimpan dengan 7 jam tertambah — karena satu jalur kode
menyimpannya setelah konversi — data itu akan bergeser lagi saat dibaca sistem baru.

**Akibatnya.** Tanggal kejadian dan tanggal lapor bergeser satu hari pada kasus di sekitar
tengah malam, dan itu **mengubah hasil validasi** aturan seperti "Tanggal Lapor ≤ DOL + 7 hari".

**Penanganan.**
1. Periksa contoh data produksi untuk memastikan konvensi penyimpanan yang sebenarnya.
2. Uji khusus untuk kasus di sekitar tengah malam WIB (Testing Strategy §3.1).
3. Bandingkan tanggal yang ditampilkan Pega dan sistem baru pada klaim yang sama.

---

## R-13 · Tenggat tidak punya pemaksa eksternal

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Terjadi |
| **Status** | Terbuka · pemilik **manajemen** |

**Masalahnya.** Pendorong migrasi adalah kemandirian teknologi dan maintainability — **bukan**
lisensi yang berakhir, **bukan** biaya lisensi, dan **bukan** arahan standardisasi grup. Target
akhir September 2026 karena itu adalah **target internal**, tanpa tanggal yang dipaksakan pihak
luar dan **tanpa penalti kontraktual**.

**Kenapa ini risiko, bukan kabar baik.** Menerima `R-05` apa adanya akan wajar bila ada tenggat
eksternal yang tidak dapat digeser. Di sini tidak ada, sehingga risiko yang diterima **tidak
dibayar oleh manfaat apa pun** — yang tersisa hanya biayanya: mutu yang dikompromikan pada sistem
yang menangani uang klaim.

**Status keputusan.** `D-61` menetapkan **jadwal seluruh modul tidak berubah**, dan selisih antara
jadwal dan kesiapan menjadi **tanggung jawab manajemen**. Risiko ini karena itu tetap terbuka
secara sadar, bukan karena belum dibahas.

---

## R-14 · Ukuran keberhasilan tidak dapat dibuktikan untuk lima modul

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Terjadi |
| **Status** | **Membaik** seiring `R-01` — 62 dari 64 objek sudah diterima |

**Masalahnya.** Keberhasilan diukur sebagai kesetaraan fungsional dengan Pega, dan gerbang pertama
cutover adalah uji kesetaraan otomatis. Keduanya menuntut satu hal yang sama: **mengetahui apa
hasil yang benar.** Untuk `B-5`, `B-7`, `B-9`, `B-10`, dan `B-12`, hasil yang benar ditentukan
oleh procedure yang logikanya belum pernah dibaca (`R-01`).

> **Kesetaraan atas logika yang tidak diketahui tidak dapat dibuktikan, hanya diduga.**

**Yang sudah berubah.** Setelah `Database/` diterima, `D-55` mencabut `BRD §21.4` untuk **`B-7`,
`B-9`, `B-10`, dan `B-12`**; hanya **`B-5`** yang masih terikat, dengan tiga penghalang tersisa —
isi `POOLDATA.GCNM_FEE_SCALE` (17 pita), isi `m_currencystandard`, dan
`BrowseT_Claim_Adjustment_SQL`.

**Dua modul yang persoalannya berbeda.** `F-3` dan `S-5` **tidak punya baseline Pega sama sekali**
— HCC/HCQ nol jejak di export (`T-2`), dan sistem lama tidak punya jejak audit atas nilai
(`T-14`). Untuk keduanya, gerbang 1 diganti **uji fungsional terhadap kontrak** (`D-56`,
`ADR-0028`), dan kedua kontrak itu **belum ada**.

---

## R-15 · Waktu user bisnis untuk UAT belum dialokasikan

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Terjadi |
| **Status** | Terbuka · pemilik **manajemen** |

**Masalahnya.** Gerbang 2 setiap modul adalah UAT pengguna bisnis, untuk sekitar 20 inbox berbasis
peran dan 5 layar transaksi utama. Pengguna yang sama sedang menjalankan **operasional klaim
harian**, dan alokasi waktunya **belum tercatat di mana pun** serta **tidak punya ruang pada
jadwal `C-2`**.

**Penanganan.** Alokasikan resmi per modul dan perlakukan sebagai **constraint project** (`C-9`),
bukan detail pelaksanaan yang diasumsikan tersedia. `D-60` menetapkan modul fondasi tidak melewati
gerbang 2 — itu mengurangi beban, tetapi tidak menghapusnya untuk modul bisnis.

---

## R-16 · Export Pega tidak lengkap

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Terjadi |
| **Status** | Terbuka · pemilik **Tim Pega** · **memburuk** |

**Masalahnya.** Sekitar **242 rule dirujuk tetapi tidak ada di export** — jauh di atas angka yang
diaudit sebelumnya. Tujuh tipe rule **tidak diaudit `19-GAP` sama sekali**: When, Flow Action,
Section, Data Transform, Ticket, Report Definition, dan Function library.

**Yang memburuk.** Jumlah **When rule buatan sendiri yang hilang naik dari 116 menjadi 137**
setelah penghitungan ulang terhadap snapshot baru (`D-45`). When rule adalah **kategori ketiga**
di luar `R-01` (objek database) dan `R-07` (activity), dan dampaknya lebih luas: ia memblokir
**percabangan bisnis di hampir semua modul**, bukan lima modul.

**Contoh konkret yang sudah menggigit.** Lima When rule yang mengendalikan item menu hilang —
`IsGCNMReport`, `IsKomite`, `IsNotViewClaim`, `IsPNCBonding`, `IsSurvey` — sehingga peta peran →
menu pada `F-3` tidak dapat disusun lengkap (`ADR-0023`). Delapan Ticket rule custom juga hilang
(`ADR-0021`).

**Penanganan.** Permintaan export ulang berbasis **Product rule**, bukan pemilihan manual per tipe
(`D-39`).

---

## R-17 · Kredensial aktif berada di dalam export

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | **Sudah terjadi** |
| **Status** | Terbuka · pemilik **Tim Infra / Security** |

**Masalahnya.** Export memuat **3 password SMTP di 31 lokasi** ditambah **satu pasang kredensial
OAuth**, seluruhnya plaintext, dan `UseSSL=false` pada seluruh kemunculannya.

**Kenapa ini berbeda dari risiko lain di dokumen ini.** Ia **bukan risiko migrasi**: ia hidup
sekarang, bahkan bila migrasi dibatalkan hari ini. Yang dipertaruhkan juga bukan jadwal, melainkan
**cara repository ini disimpan dan dibagikan**.

**Penanganan.** `D-40` masih **OPEN** — ke mana rahasia dipindahkan, siapa pemiliknya, dan apakah
kredensial yang telanjur terekspos harus dirotasi (`ADR-0025`). Lokasi lengkapnya sudah diserahkan
ke Tim Infra/Security lewat dokumen terpisah **di luar repository**; tidak ada satu nilai pun
tertulis di dokumen yang di-commit (`D-69`).

---

## R-18 · Dua integrasi BRI menunjuk host sandbox

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | **Sudah terjadi** |
| **Status** | Terbuka · pemilik **Work Owner + tim integrasi** |

**Masalahnya.** Dua Connect REST ke BRI menunjuk **host sandbox** di dalam ruleset produksi.
Entah integrasi itu memang tidak aktif, atau ia aktif dan mengarah ke lingkungan yang salah —
keduanya harus dijawab sebelum integrasi dibawa ke sistem baru.

**Penanganan.** Konfirmasi status kedua integrasi ke tim integrasi, dan tetapkan endpoint per
lingkungan sebagai konfigurasi (`ADR-0025`), bukan konstanta di dalam rule.

---

## R-19 · Cacat aturan uang direplikasi bila `P-5` dipatuhi buta

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Terjadi bila tidak ditangani |
| **Status** | **Ditangani** — `D-49` · pemilik **Work Owner** |

**Masalahnya.** `P-5` menetapkan kesetaraan perilaku lebih dulu, termasuk meniru keanehan sistem
lama. Dipatuhi buta, prinsip itu akan **membawa serta cacat pada aturan yang menghitung uang** —
antara lain toleransi spreading berupa pencocokan substring (`199.99` dan `1100.0` lolos), ambang
Rp 50 juta dengan **tiga operator berbeda** di tiga rule, penyesuaian 7 jam yang asimetris di
dalam satu kondisi validasi, dan kurs yang mengembalikan `1` saat tidak ditemukan.

**Bagaimana ditangani.** `D-49` memutuskan sepuluh cacat satu per satu: **sembilan diperbaiki, satu
direplikasi** karena perilakunya memang benar (tanggal PLA/DLA diambil dari waktu sistem, bukan
dari pilihan pengguna). Daftar perbaikan eksplisit `P-5` bertambah dari **4 menjadi 13 butir**
(`ADR-0017`).

**Yang tersisa.** Data historis tetap mengandung akibat cacat itu — memperbaiki aturannya tidak
memperbaiki baris yang telanjur salah, khususnya `IDSALVAGE = NULL` yang menuntut hitungan DBA.

---

## R-20 · Kebocoran data antar badan hukum saat berpindah portal

| | |
|---|---|
| **Dampak** | **Sangat tinggi** — lintas badan hukum |
| **Kemungkinan** | Terjadi bila tidak ditangani |
| **Status** | **Terbuka** — `D-75` · `ADR-0030` · pemilik **Work Owner + Keamanan Informasi** |

**Masalahnya.** `D-75` menetapkan empat portal dengan **satu database per entitas**, dan
perpindahan portal **tanpa login ulang**. `D-78` kemudian menetapkan **login yang sama untuk
keempat entitas**, sehingga **satu kredensial bocor menjangkau empat badan hukum** — batas yang
tadinya ada bila tiap entitas punya login sendiri kini **hilang**, dan yang tersisa sebagai
pembatas hanyalah **kewenangan portal per pengguna**. Artinya **satu identitas menjangkau empat database milik
empat badan hukum**. Bila kewenangan tidak dinilai ulang pada saat perpindahan, seorang pengguna
dapat melihat data entitas yang bukan haknya.

**Kenapa ini lebih berbahaya daripada cacat akses biasa.** Kegagalannya **tidak terlihat sebagai
galat**. Layar tampil normal, angkanya masuk akal, dan tidak ada pesan kesalahan — yang salah hanya
*milik siapa* data itu. Cacat seperti ini dapat berjalan lama tanpa ada yang menyadarinya.

Dua jalur kegagalan yang sudah terlihat di rancangan:

1. **Jatuh ke koneksi default** ketika portal tidak diketahui, alih-alih menolak permintaan.
2. **Portal aktif disimpan sebagai keadaan global** di server — dua permintaan bersamaan dari
   pengguna yang sama akan saling menimpa, dan data kedua entitas tertukar.

**Bagaimana ditangani.** `TKT-F6-003` menuntut kewenangan **dinilai ulang di server pada setiap
perpindahan** (`ADR-0023`), portal yang bukan hak **tidak muncul dan tidak dapat dipanggil
langsung**, dan portal aktif **melekat pada permintaan**, bukan pada keadaan global. `TKT-F6-002`
menuntut permintaan tanpa portal **ditolak**, bukan jatuh ke default.

**Yang tersisa.** Tiga hal belum diputuskan dan menahan penanganan: apakah penyedia identitas satu
untuk keempat entitas atau satu per entitas · di mana kewenangan portal disimpan (bila di database
tiap portal, memeriksa hak atas portal B menuntut membaca database B **sebelum** pengguna berhak
membacanya) · dan apakah satu pengguna boleh berhak di lebih dari satu portal. Ditambah `D-59` yang
menetapkan kontrol berbasis **menu** tanpa pemisahan tugas formal — kendali portal **tidak boleh**
ikut bersandar pada penyembunyian menu.

---

## Ringkasan tindakan yang harus dimulai hari pertama

Diperbarui 2026-09-14. Tanda ✅ menandai tindakan yang **sudah selesai**.

| # | Tindakan | Menutup risiko | Kepada | Status |
|---|---|---|---|---|
| 1 | Minta source 64 procedure & function (daftar: `19-GAP-EXPORT-DETAIL.md`) | R-01 | DBA | ✅ **62 dari 64 diterima** — sisa `SET_ATTACHFILETEMPSALVAGE`, `UPDATEPREMIUMTEMPLATE` |
| 1b | Minta **12 dependensi** yang dipanggil 62 procedure itu (terberat `UPDATE_LOG_KONVERSI` 162×) | R-01 | DBA | **Terbuka** |
| 2 | Minta export Rule-Agent / Queue Processor | R-02 | Tim Pega | ✅ **selesai** — 5 job + 1 agent (`D-57`) |
| 3 | Minta DDL + statistik ukuran tabel | R-08 | DBA | Terbuka |
| 4 | Ambil isi master `V_STS_CLAIM` | R-06 | DBA | ✅ **selesai** — 33 kode `1134`–`1166` |
| 5 | Minta export 3 router + activity yang hilang | R-04, R-07 | Tim Pega | Terbuka — `R-04` turun menjadi verifikasi |
| 5b | **Permintaan export ulang berbasis Product rule** — ±242 rule hilang, 137 di antaranya When rule | R-16 | Tim Pega | **Terbuka** (`D-39`) |
| 6 | Sampaikan kebutuhan 6 API pengganti DB Link | R-03 | Tim pemilik sistem | Terbuka |
| 7 | Ambil isi tabel `POOLDATA.EMAILKOMITE` (matriks komite) | D-14 | DBA | ✅ **selesai** — 21 kolom, 30 baris |
| 7b | Konfirmasi aturan `SetEmailKomite` yang berbasis nama orang | D-14 | Tim bisnis | ✅ **selesai** — dicabut, diganti pita nilai (`D-52`, `D-70`) |
| 8 | Konfirmasi retensi audit | D-28 | Compliance | ✅ **terjawab** — mengikuti retensi data klaim (`D-62`); **angkanya** masih diminta |
| 9 | Sepakati pembekuan perubahan Pega | R-09 | Tim Pega + manajemen | Terbuka |
| 10 | Mulai pelatihan Go, React, TypeScript | R-11 | Tim pengembang | Terbuka |
| 11 | **Tetapkan tujuan penyimpanan rahasia dan keputusan rotasi** | **R-17** | Tim Infra / Security | **Terbuka** (`D-40`) |
| 12 | Konfirmasi status dua integrasi BRI bersandbox | **R-18** | Tim integrasi | **Terbuka** |
| 13 | Sediakan **kontrak API HCC/HCQ** | R-14 | Tim HCC/HCQ | **Terbuka** (`ADR-0024`) |
| 14 | Sediakan **daftar peristiwa wajib audit** | R-14 | Compliance | **Terbuka** (`ADR-0026`) |
| 15 | Konfirmasi ketersediaan **Pega staging yang dapat ditembak dari luar** | R-14 | Tim Pega + Infra | **Terbuka** — penghalang utama `S-8` (`ADR-0027`) |

**Sebagian besar tindakan ini bergantung pada pihak di luar tim pengembang.** Karena itu
seluruhnya harus dimulai hari pertama — waktu tunggunya tidak bisa dipercepat dengan menambah
developer.

# 16. Future Enhancement

Perbaikan yang **sengaja tidak dikerjakan selama migrasi**, agar prinsip P-5 (perilaku
dipertahankan lebih dulu, diperbaiki kemudian) tetap terjaga.

Alasan menundanya bukan karena tidak penting, melainkan karena selama migrasi kita harus bisa
membedakan **bug** dari **perbaikan yang tidak tercatat**. Bila perilaku diperbaiki sambil jalan,
setiap perbedaan hasil antara Pega dan sistem baru menjadi tidak bisa dijelaskan — dan
verifikasi kesetaraan kehilangan maknanya.

Daftar ini ditulis sekarang justru agar temuan selama analisis tidak hilang.

---

## 1. Setelah migrasi stabil

### 1.1 Perbaikan pengalaman pengguna
D-13 menetapkan tampilan meniru Pega agar pengguna tidak perlu belajar ulang. Setelah sistem
baru stabil dan pengguna terbiasa, ruang perbaikan yang teridentifikasi:

- **Inbox berkolom banyak** terasa padat pada sebagian layar. Kolom yang dapat dipilih pengguna dengan preset per
  peran akan jauh lebih terbaca.
- **Form registrasi sangat panjang** dengan puluhan field. Dapat dipecah menjadi langkah-langkah
  dengan penyimpanan draf.
- **Pesan validasi** dapat ditampilkan langsung di sebelah field saat pengguna mengetik, bukan
  hanya setelah submit.
- **Layar surveyor** dapat dioptimalkan khusus untuk penggunaan lapangan (D-12): unggah foto
  yang tahan koneksi terputus, dan mode offline.

### 1.2 Penyederhanaan model status
D-18 menetapkan empat konsep status dipertahankan karena memang berbeda. Setelah arti kode
`1134`–`1166` diketahui (R-06 tertutup) dan pola pemakaiannya terlihat dari data nyata, layak ditinjau
apakah keempatnya masih perlu terpisah — atau sebagian dapat diturunkan dari yang lain.

Ini peninjauan berbasis data, bukan asumsi. Karena itu harus menunggu sampai ada data.

### 1.3 Membersihkan duplikasi lini bisnis
Sistem lama menggandakan pola `Browse*`, `Insert*`, `Grouping*`, `CreateCasePNC*` untuk tiap
lini bisnis (`_AsuransiKredit`, `_AutoClaim`, `_Travel`, `_Kredit_PA`). Migrasi mempertahankan
perbedaan perilakunya apa adanya.

Setelah stabil, layak ditinjau mana yang **benar-benar berbeda secara bisnis** dan mana yang
hanya hasil salin-tempel. Yang kedua dapat disatukan menjadi satu implementasi dengan parameter
lini bisnis.

---

## 2. Peluang teknis

### 2.1 Menyelesaikan perpindahan ke PostgreSQL
Rencana lengkapnya di `09-DATABASE-STRATEGY.md` §10. Karena SQL sudah portabel (D-20) dan tidak
ada pemanggilan stored procedure (D-02), perpindahan ini **tidak menyentuh logika bisnis**.

Setelah pindah, fitur PostgreSQL yang dapat dimanfaatkan: index parsial untuk inbox, index GIN
untuk pencarian di dalam JSONB snapshot polis, dan partisi deklaratif untuk tabel klaim dan
audit.

### 2.2 Read replica untuk laporan
Bila laporan mulai mengganggu transaksi pengguna, replika baca adalah langkah berikutnya yang
paling sepadan (NFR §4.2 langkah 4). Pool koneksi terpisah yang sudah dirancang sejak awal
membuat perubahan ini hanya soal konfigurasi.

### 2.3 Arsip data historis
Dengan puluhan juta baris yang terus tumbuh (D-10), pemindahan klaim lama yang sudah tutup ke
tabel arsip akan menjaga tabel aktif tetap ramping. Dilakukan berdasarkan pengukuran, bukan
dugaan.

### 2.4 Pencarian teks
Pencarian klaim saat ini memakai `LIKE '%...%'` yang tidak dapat memanfaatkan index. Setelah
pindah ke PostgreSQL, pencarian teks penuh bawaan PostgreSQL dapat menggantikannya tanpa
komponen tambahan.

---

## 3. Peluang bisnis yang terlihat dari analisis

Berikut ditemukan saat membaca source dan patut dipertimbangkan pemilik bisnis — bukan usulan
teknis.

### 3.1 Dashboard TAT yang dapat ditindaklanjuti
Sistem sudah menghitung TAT dan menyimpan `PNC_CHRONOLOGYTAT`, tetapi hanya menampilkannya
sebagai laporan. Dengan data yang sama, sistem dapat **memberi peringatan sebelum tenggat
terlampaui**, bukan melaporkan setelah terlewat.

### 3.2 Deteksi klaim mencurigakan
Sistem sudah punya peran Investigator, tabel `T_CLAIM_DATA_RESULTS_AI`, dan `M_DOMINAN_FACTOR` —
menunjukkan pernah ada upaya ke arah ini. Dengan data historis puluhan juta baris, pola klaim
mencurigakan dapat diangkat secara otomatis untuk ditinjau, bukan hanya berdasarkan kecurigaan
petugas.

### 3.3 Portal tertanggung
Saat ini status klaim hanya dapat dilihat petugas. Portal untuk tertanggung melihat status
klaimnya sendiri akan mengurangi beban Service Center. Perlu pertimbangan keamanan tersendiri
karena melibatkan akses dari luar jaringan internal.

### 3.4 Otomasi kelengkapan dokumen
Sistem sudah punya master jenis dokumen per lini bisnis (`LST_TYPE_DOC_BUSINESS`,
`V_LST_DOC_TYPE`). Pemeriksaan kelengkapan dokumen dapat berjalan otomatis dan memberi tahu
tertanggung apa yang masih kurang.

---

## 4. Utang teknis yang sudah dijadwalkan diselesaikan

Untuk kejelasan — yang berikut **tidak** ditunda, melainkan **sudah masuk scope migrasi**:

| Utang teknis | Diselesaikan oleh |
|---|---|
| Nilai bisnis di-hardcode | D-15 — master data |
| Blok `// TESTING` di jalur produksi | D-15 — tidak dibawa |
| Penambahan 7 jam manual | F-5 — modul Clock |
| Perangkaian SQL `{ASIS:...}` | Coding Standards — parameter binding wajib |
| Alias kolom menyesatkan | D-19 — penamaan ulang |
| Kunci Pega di data bisnis | D-22 + D-71 — format `PNCN.YY.xxxx` |
| Perilaku bergantung hostname | Cross-Cutting §3.4 — dilarang |
| Tanpa jejak audit | D-28 — dirancang sejak awal |
| Pemanggilan stored procedure | D-02 — logika naik ke Go |
| DB Link lintas database | D-25 — diganti API |

# Lampiran A — Glossary Domain (Ubiquitous Language)

Glossary bahasa domain untuk sistem Claim PNC. Dokumen ini **hanya** kamus istilah — tanpa
detail implementasi, tanpa nama tabel, tanpa keputusan teknis. Keputusan teknis ada di
`00-DECISION-LOG.md` dan dokumen Steering lainnya.

Sumber istilah: 902 activity, 652 SQL rule, 70 when rule, 4 flow Pega — dikonfirmasi langsung
oleh pemilik bisnis pada 2026-09-07.

Setiap istilah punya penanda asal:
`[BISNIS]` = dikonfirmasi pemilik bisnis · `[KODE]` = disimpulkan dari source · `[TERBUKA]` = belum pasti

---

## Lini Bisnis & Segmentasi

**MBU — Motor Business Unit** `[BISNIS]`
Lini bisnis kendaraan bermotor. Bukan cakupan utama aplikasi ini.

**Non-MBU** `[BISNIS]`
Seluruh lini bisnis di luar kendaraan bermotor. **Non-MBU inilah yang disebut PNC.**

**PNC** `[BISNIS]`
Sinonim dari Non-MBU. Nama aplikasi "Claim PNC" berarti *penanganan klaim untuk lini bisnis
non-motor*. Jangan mengartikan PNC sebagai singkatan teknis lain.

**Group Panel** `[KODE]`
Kode segmentasi lini bisnis yang menyetir hampir seluruh percabangan aturan — validasi,
routing, estimasi, komite, dan spreading:

| Kode | Lini |
|---|---|
| `002` | Personal Accident (PA) |
| `003` | Aneka |
| `004` | Marine Cargo |
| `005` | Travel |
| `006` | Fire / Property |
| `009` | Aneka (varian lain) |

**Business Type** `[KODE]`
Klasifikasi lebih rinci di dalam Group Panel: `MBUCar`, `MBUMotorCycle`, `Fire`, `MarineCargo`,
`HE`, `ContractorsPM`, `Bonding`, `BondingKBG`, `PAYDI_PA`, `Travel`.

**TKA — Tenaga Kerja Asing** `[BISNIS]`
Lini bisnis asuransi untuk pekerja warga negara asing.

**SPK — Sinarmas Penjaminan Kredit** `[BISNIS]`
Lini bisnis penjaminan kredit. Di kode muncul sebagai "Asuransi Kredit".
Punya kewajiban khusus: **Nomor SLIK wajib diisi** saat registrasi.

---

## Inti Klaim

**Klaim (Claim)** `[BISNIS]`
Pengajuan ganti rugi atas satu polis akibat satu peristiwa kerugian. Satu klaim mencakup satu
atau lebih Objek Pertanggungan.

**Polis (Policy)** `[BISNIS]`
Kontrak asuransi. Dimiliki oleh domain lain (GISFW, tim berbeda). Domain Klaim hanya menyimpan
**Snapshot Polis**.

**Snapshot Polis** `[BISNIS]` (D-04)
Salinan data polis pada saat klaim diregistrasi. Setelah snapshot diambil, klaim tidak lagi
terpengaruh perubahan polis. Ini yang membuat domain Klaim berdiri sendiri.

**Objek Pertanggungan (Insured Item)** `[KODE]`
Barang atau orang yang dipertanggungkan dan terdampak kerugian — kendaraan, orang, properti,
kargo, atau lokasi. Di sistem lama bernama `ObjectList`.
→ Istilah `Object` sengaja **dihindari** di sistem baru karena bertabrakan dengan makna
pemrograman.

**Coverage (Jaminan)** `[KODE]`
Jenis jaminan yang melekat pada satu Objek Pertanggungan, beserta nilai TSI dan Penyebab
Kerugian yang berlaku.

**TSI — Total Sum Insured** `[KODE]`
Nilai pertanggungan. Nilai klaim tidak boleh melebihi TSI — divalidasi saat registrasi.

**Sisa TSI** `[BISNIS]`
Bagian TSI yang belum terpakai untuk satu Objek Pertanggungan pada satu Coverage: TSI dikurangi
akumulasi nilai akseptasi yang masih **Outstanding**, lalu **ditambah** nilai **Salvage** —
salvage memulihkan kapasitas pertanggungan. Nilai usulan penyelesaian tidak boleh melebihi sisa
TSI. Berlaku pada lini Personal Accident.

**Cause of Loss (Penyebab Kerugian)** `[KODE]`
Sebab terjadinya kerugian. Wajib diisi kecuali untuk lini Travel.

**Date of Loss (DOL) — Tanggal Kejadian** `[KODE]`
Tanggal peristiwa kerugian terjadi. Titik acuan hampir seluruh aturan tanggal.

**Report Date — Tanggal Lapor** `[KODE]`
Tanggal tertanggung melaporkan kerugian.

**Date Received — Tanggal Terima Dokumen** `[KODE]`
Tanggal dokumen klaim diterima.

> Urutan wajib: **DOL ≤ Tanggal Lapor ≤ Tanggal Terima Dokumen ≤ hari ini**

---

## Nilai & Penyelesaian

**Estimasi Klaim (Claim Estimate)** `[KODE]`
Perkiraan awal nilai kerugian saat registrasi. Dipakai untuk PLA dan untuk memicu notifikasi
kerugian besar.

**Settlement Line** `[KODE]`
Baris penyelesaian nilai untuk satu Coverage. Di sistem lama bernama `AdjustmentList`.
Menyimpan perjalanan nilai uang klaim:

`Estimasi → Usulan (Propose) → Akseptasi (Accepted) → Dibayar (Paid)`

> Istilah lama `Adjustment` menyesatkan karena dalam praktik asuransi "adjusting" berarti proses
> penilaian kerugian, sedangkan di sini isinya adalah **nilai penyelesaian**. Di sistem baru
> dipakai istilah **Settlement**.

**Akseptasi** `[BISNIS]`
Persetujuan atas **nilai** klaim yang akan dibayarkan. Menghasilkan **Nomor Akseptasi**.
Berbeda dari persetujuan bahwa klaim itu dijamin (liability).

**OS — Outstanding** `[BISNIS]`
Klaim yang sudah diakui nilainya tetapi belum selesai dibayar. "OS Akseptasi" = akseptasi
yang masih menggantung.

**Ex-Gratia** `[KODE]`
Pembayaran kebijakan di luar kewajiban polis. Bila klaim ditandai ex-gratia, tipe treaty
`OR` otomatis berubah menjadi `ORS`.

**Salvage** `[KODE]`
Nilai sisa barang rusak yang bisa dijual kembali (termasuk lewat balai lelang). Mengurangi
nilai bersih klaim.

**Recovery** `[KODE]`
Pemulihan dana dari pihak ketiga, termasuk lewat Virtual Account.

**Notice of Large Losses** `[KODE]`
Pemberitahuan wajib ke Underwriting dan jajaran pimpinan bila estimasi klaim (setelah konversi
kurs) melebihi **Rp 1.000.000.000**.

**Kurs Standar** `[BISNIS]` (D-48)
Nilai tukar yang dipakai untuk mengubah nilai klaim mata uang asing menjadi Rupiah. Kurs yang
berlaku adalah kurs **pada tanggal kejadian** — bukan kurs saat klaim diproses atau dibayar,
sehingga nilai Rupiah sebuah klaim tidak berubah karena keterlambatan proses. Bila kurs untuk
tanggal itu tidak tersedia, klaim **ditolak dengan pesan yang menyebutkan mata uang dan
tanggalnya** — tidak ada kurs pengganti dan tidak ada nilai bawaan.

---

## Reasuransi & Koasuransi

**Spreading** `[KODE]`
Pembagian risiko satu Coverage ke para penanggung. **Total share wajib 100%** (toleransi
Pembagian risiko satu Coverage ke para penanggung. **Total share wajib 100%**, **toleransi 4 desimal, `99,9999`–`100,0001`** (`D-51`) —
divalidasi saat registrasi dan menolak submit bila tidak terpenuhi. Angka `99,99` yang dulu
lolos kini **ditolak** — selisih yang direncanakan (`D-49` butir 1).
**Koasuransi (Coins)** `[KODE]`
Pembagian risiko antar sesama perusahaan asuransi. Ada peran Leader dan Member.

**Reasuransi (Reinsurance)** `[KODE]`
Pengalihan risiko ke perusahaan reasuransi.

**Treaty** `[KODE]`
Perjanjian reasuransi otomatis. Tipe yang muncul: `OR`, `ORS`, Fac Out (`10015`), Quota Share,
XOL, BPPDAN.

**BPPDAN — Badan Pengelola Pusat Data Asuransi Nasional** `[BISNIS]`
Salah satu jenis treaty asuransi.

**Fac Out (Facultative Outward)** `[KODE]`
Pengalihan risiko per kasus, bukan otomatis. Bila ada spreading Fac Out, data Fac Offer wajib
lengkap — termasuk Object Name untuk Group Panel `003`.

**XOL — Excess of Loss** `[KODE]`
Treaty non-proporsional yang menanggung kerugian di atas batas tertentu.

**PLA — Preliminary Loss Advice** `[BISNIS]`
Pemberitahuan **nilai estimasi** klaim kepada koasuransi/reasuransi.

**Pre-DLA** `[BISNIS]`
Pemberitahuan nilai klaim yang **akan** diakseptasi kepada koasuransi/reasuransi, dikirim
**sebelum** akseptasi dilakukan.

**DLA — Definite Loss Advice** `[BISNIS]`
Pemberitahuan **nilai akseptasi** klaim kepada koasuransi/reasuransi.

> Urutan pemberitahuan: **PLA** (estimasi) → **Pre-DLA** (akan diaksep) → **DLA** (sudah diaksep)

**KBRU — Kali Besar Raya Utama** `[BISNIS]`
Salah satu broker asuransi yang berintegrasi dengan sistem.

---

## Proses & Peran

**Register (Registrasi Klaim)** `[KODE]`
Tahap awal: pencatatan klaim, validasi tanggal, cek duplikat, penetapan objek dan coverage,
serta spreading. Gerbang validasi terberat di seluruh sistem.

**Open Protection (Buka Proteksi)** `[KODE]`
Permintaan pembukaan proteksi sebelum/di luar alur klaim normal. Satu Open Protection dapat
ditautkan ke klaim dan ditandai terpakai.

**Receive Document** `[KODE]`
Pencatatan penerimaan dokumen fisik klaim, termasuk pengiriman antar cabang (ekspedisi, no.
resi, estimasi tiba).

**Surveyor / Loss Adjuster** `[KODE]`
Pihak yang meninjau kerugian di lapangan. Bisa internal (ASM) atau eksternal. Menghasilkan
hasil survei dan foto dokumentasi.

**Komite (Committee)** `[BISNIS]`
Forum persetujuan berjenjang atas nilai klaim. Jumlah jenjang ditentukan **kombinasi nilai klaim
dan jenis bisnis** (D-14), dan bersifat **kumulatif**: makin besar nilai klaim, makin banyak
jenjang yang harus menyetujui (D-47). Komite dapat menyetujui, menolak, atau mengembalikan.

**Pita Nilai Komite** `[BISNIS]` (D-52, D-70)
Pengelompokan klaim **Non-MBU** menurut besar nilainya, yang menentukan **rangkaian jenjang
mana** yang menangani klaim itu. Dua pita: sampai **Rp 100.000.000**, dan di atas
**Rp 100.000.000**. Pita dipilih **lebih dulu**, sebelum jenjang mana pun dihitung, dan setiap
pita punya rangkaian jenjangnya sendiri. Di sistem lama pita ini dipilih berdasarkan nama orang;
di sistem baru diturunkan dari nilai klaim.

> Istilah ini **hanya berlaku di lini Non-MBU** (D-70). Lini lain tidak mengenal pita nilai —
> jenjangnya satu rangkaian utuh, dibedakan hanya oleh ambang bawah.

**Jenjang Kumulatif** `[BISNIS]` (D-47, D-70)
Cara menghitung berapa banyak persetujuan yang dibutuhkan sebuah klaim. Setiap jenjang punya
**ambang bawah** — nilai klaim minimum yang membuatnya ikut campur. Semua jenjang yang ambang
bawahnya **sudah terlampaui** nilai klaim harus menyetujui, bukan hanya jenjang tertinggi.
Akibatnya jumlah penyetuju **bertambah** seiring besarnya klaim, bukan berpindah orang: klaim
yang melampaui satu ambang butuh satu persetujuan, klaim yang melampaui tiga ambang butuh tiga.

Di lini **Non-MBU** akumulasi itu didahului satu langkah: **pita dipilih lebih dulu**, lalu
akumulasi berjalan **di dalam pita itu saja** dan tidak menyeberang ke pita lain. Di lini lain
tidak ada langkah pendahuluan — akumulasi langsung berjalan atas seluruh jenjang lini tersebut.

**Batas atas** setiap jenjang tetap dicatat di master, tetapi hanya untuk memastikan pita dan
jenjang tidak bertumpang tindih dan tidak berlubang saat master diisi — batas atas tidak pernah
menentukan siapa yang menyetujui sebuah klaim.

**RCL — Rejected Klaim** `[BISNIS]`
Klaim yang ditolak.

**RCL Dokter** `[KODE]`
Penolakan yang memerlukan pertimbangan medis, dipakai pada lini Personal Accident.

**PUCL — Proses Ulang Klaim** `[BISNIS]`
Klaim yang diproses ulang setelah sebelumnya ditolak atau ditutup.

**Compliance Check** `[KODE]`
Pemeriksaan kepatuhan sebelum klaim diselesaikan.

**Investigator** `[KODE]`
Peran yang menyelidiki klaim yang mencurigakan.

**Analyst Doctor** `[KODE]`
Peran tenaga medis yang menilai klaim kesehatan/kecelakaan diri.

**PIC Teknik / User Teknis** `[KODE]`
Penanggung jawab teknis klaim dari sisi keahlian lini bisnis.

**User Admin** `[KODE]`
Petugas administrasi yang melakukan registrasi dan input data klaim.

**LOD — Letter of Discharge** `[BISNIS]`
Surat pemberitahuan nilai ganti rugi yang disetujui asuransi kepada **tertanggung**.
Berbeda dari PLA/DLA yang ditujukan ke koasuransi/reasuransi.

**TAT — Turn Around Time** `[KODE]`
Waktu penyelesaian klaim, dipantau lewat laporan TAT.

**SLIK OJK** `[KODE]`
Pelaporan ke Sistem Layanan Informasi Keuangan OJK. Wajib untuk lini SPK / Asuransi Kredit —
Nomor SLIK tidak boleh kosong.

**Tugas (Assignment)** `[BISNIS]` (D-26)
Satu pekerjaan yang menunggu dikerjakan orang pada satu tahap klaim. Sebuah klaim melewati
banyak tugas berturut-turut; satu tugas selalu berada di **Worklist** atau di **Workbasket**,
tidak pernah di keduanya.

**Worklist** `[BISNIS]` (D-26)
Daftar tugas milik **satu orang tertentu**. Tugas di dalamnya sudah punya pemilik dan hanya
dikerjakan orang itu. Dipakai untuk tahap yang butuh kesinambungan penanganan — antara lain
Input Register, Estimasi, pemilihan surveyor, dan penilaian Analyst Doctor.

**Workbasket** `[BISNIS]` (D-26)
Antrean tugas **bersama** yang belum bertuan: siapa pun yang berwenang atas antrean itu boleh
mengambilnya. Dipakai untuk tahap yang dikerjakan sebuah tim, antara lain RCL/PUCL,
Investigator, dan Compliance.

**Inbox** `[BISNIS]` (D-79)
**Layar yang menampilkan daftar pekerjaan milik seorang pengguna** — yakni **Tugas** dari
**Worklist** atau **Workbasket** yang menunggu dikerjakan.

Yang membuatnya Inbox bukan bentuknya, melainkan isinya. Empat ciri yang membedakannya dari layar
daftar biasa: barisnya adalah **pekerjaan**, bukan data acuan · baris **hilang** setelah selesai
dikerjakan · "hanya milik saya" adalah **aturan kewenangan**, bukan sekadar penyaring · dan
barisnya punya **tenggat**.

**Layar yang menampilkan data acuan (master) bukan Inbox**, sekalipun dapat dicari dan sekalipun
namanya di sistem lama mengandung kata "Inbox". Dari 26 harness bernama *Inbox* di export,
sebagian bersidik jari layar master — lihat Lampiran Inventaris 74 Harness.

---

## Status — empat konsep berbeda

Dikonfirmasi pemilik bisnis (D-18): keempatnya **memang berbeda** dan semuanya dipertahankan.
Di sistem baru masing-masing diberi nama yang tidak lagi bisa tertukar.

**Status Proses** — sistem lama: `StatusWork` `[BISNIS]`
Posisi klaim dalam alur kerja: sedang berjalan, selesai, atau ditolak.

**Status Klaim** — sistem lama: `StatusClaim`, kode `1134`–`1166` `[BISNIS]`
Status bisnis klaim. **33 kode**, masing-masing berlabel di master status. Sebelas kode pertama
(`1134`–`1144`) membawa penomoran lama `01`–`11`; sisanya hanya bernomor baru.
Contoh: `1142` Rejected Claim · `1143` Close Claim for this object · `1144` Cancelled Claim ·
`1147` Register · `1149` Claim Committee · `1163` Paid · `1164` Reopen Claim.

> Rentang `1142`–`1151` yang dipakai dokumen-dokumen awal ternyata **sebagian** dari domainnya,
> bukan keseluruhannya.

**Flag Klaim** — sistem lama: `ClaimStatus` (`0`/`1`) `[BISNIS]`
Penanda biner. Makna persisnya masih perlu dikonfirmasi.

**Status Posisi Progres** — sistem lama: `StatusPosisi` (`On Progress` / `Done`) `[BISNIS]`
Posisi klaim pada rangkaian tahapan progres yang dicatat terpisah dari alur kerja utama.

---

## Istilah yang mudah tertukar

**Lompatan Lateral (Lateral Transition)** `[KODE]` — sistem lama: **Ticket rule**
Perpindahan klaim langsung ke sebuah tahap yang **bukan** tahap berikutnya dalam urutan normal —
misalnya klaim dikembalikan dari Komite ke petugas estimasi begitu seluruh anggota komite
menyetujui, atau klaim berpindah ke PUCL saat ditolak pada Group Panel tertentu. Di sistem lama
mekanisme ini bernama **Ticket rule**: sebuah nama tujuan dipasang pada tahap tertentu, lalu
dipicu dari tempat lain sehingga klaim "melompat" ke sana.

> **Jangan tertukar dengan tiket pekerjaan.** Di seluruh dokumen proyek ini, kata **"tiket"**
> tanpa keterangan lain berarti **tiket pekerjaan** — satuan pekerjaan yang dikerjakan di
> `docs/ticketing/`. Mekanisme Pega di atas **selalu** disebut lengkap sebagai **"Ticket rule"**,
> tidak pernah sekadar "ticket" atau "tiket". Folder `Ticket/` pada export berisi Ticket rule,
> bukan tiket pekerjaan.

**Tiket Pekerjaan** `[BISNIS]` (D-31)
Satuan pekerjaan migrasi yang dapat dikerjakan dan dinyatakan selesai secara mandiri, disimpan
di `docs/ticketing/`. Tidak ada hubungannya dengan klaim maupun dengan Ticket rule.

**Master Data** `[BISNIS]` (D-73)
Data acuan yang dipakai seluruh sistem dan berubah jauh lebih jarang daripada data klaim — antara
lain rekening, supplier, bengkel, sparepart, panel, sebab kerugian, pasal penolakan, ambang komite,
dan kurs standar. Berjumlah **sekurang-kurangnya 29 kelompok**; jumlah pastinya
**BELUM DIPUTUSKAN — pertanyaan terbuka** (pemilik: Work Owner). Sebagian di antaranya menentukan
hasil hitungan uang, sehingga perubahannya berakibat pada nilai klaim yang dihitung sesudahnya.

**Correspondence** `[KODE]` (D-73)
Nama mekanisme Pega untuk **isi surat dan email** yang dikirim sistem. Tidak ada satu pun direktori
`Correspondence/` di export, sehingga seluruh isinya tidak diketahui (`R-16`). Jumlahnya pun belum
pasti — dua dokumen internal menyebut angka berbeda (3 dan 5).

**Notifier** `[KODE]` (D-73)
Nama seam di sistem baru yang melepaskan **peristiwa domain** — bukan perintah "kirim email ke
alamat ini". Pemanggil menyatakan *apa yang terjadi*; penerimanya ditentukan dari Master Data.
Menggantikan `SendEmailNotification` yang dipanggil 15 activity dan **tidak ada di export**.

**Job Terjadwal** `[BISNIS]` (D-57, D-73)
Pekerjaan yang berjalan sendiri pada jam tertentu **tanpa ada orang yang menekan tombol**. Ada
lima di sistem lama, ditambah satu **Agent** yang berjalan tiap 30 menit. Karena tidak ada pengguna
yang bertindak, perubahan data olehnya hanya dapat ditelusuri lewat jejak audit.

**Uji Kesetaraan** `[BISNIS]` (D-42, D-54)
Menjalankan **kasus yang sama** di Pega dan di sistem baru, lalu membandingkan hasilnya. Setiap
selisih wajib **diklasifikasikan** terhadap 13 butir perbaikan `P-5`: yang cocok lolos otomatis,
yang di luar itu menunggu persetujuan Work Owner tertulis. Dikerjakan modul `S-8`.

**Gerbang 1** `[BISNIS]` (BRD §21.1, D-42)
Gerbang kelulusan pertama sebuah modul: hasilnya **setara dengan Pega** menurut Uji Kesetaraan.
Tidak ada modul yang dapat dinyatakan lulus sebelum perkakas `S-8` berjalan. Dua modul **tidak
punya baseline Pega** untuk diuji — `F-3` (identitas) dan `S-5` (jejak audit) — sehingga bagi
keduanya gerbang ini diganti ukuran lain.

**Gerbang 2** `[BISNIS]` (BRD §21.1)
Gerbang kelulusan kedua: **UAT oleh peran bisnis** yang benar-benar memakai modul itu, bukan oleh
pengembang. Setiap tiket menyebut peran pengujinya di kepala tiket.

**Portal** `[BISNIS]` (D-75)
Pilihan **entitas** di dalam satu aplikasi. Setiap portal menampilkan data dari **database
entitasnya sendiri** — satu database per entitas, bukan satu database bersama dengan penanda
entitas. Minimal empat: Asuransi Sinar Mas, Asuransi Simas Insurtech, Sinarmas Asuransi Syariah,
dan Timor-Leste. Jumlah pastinya **BELUM DIPUTUSKAN — pertanyaan terbuka** (pemilik: Work Owner),
karena itu daftarnya diperlakukan sebagai **data**, bukan konstanta di kode.

Di sistem lama tidak ada istilah maupun pilihan ini: perilaku entitas ditentukan dengan
**membandingkan nama server** (48 perbandingan terhadap `pxRequestor.pxReqServer`). Lihat
[[uji-kesetaraan]] — portal Syariah belum punya baseline utuh karena rule pembedanya hilang.

---

## Istilah yang sengaja tidak dipakai lagi

| Istilah lama | Alasan ditinggalkan | Pengganti |
|---|---|---|
| `Object` / `ObjectList` | Bertabrakan dengan makna pemrograman | Objek Pertanggungan / Insured Item |
| `Adjustment` / `AdjustmentList` | Menyesatkan; isinya nilai penyelesaian, bukan proses adjusting | Settlement Line |
| `CaseID` | Dipakai untuk 5 hal berbeda di SQL yang berbeda | Nama eksplisit per konteks |
| `pzInsKey` / `CLAIMID` berformat `ASM-FW-GCNMFW-WORK PNC-xxxx` | Kunci teknis Pega bocor ke data bisnis | Nomor Klaim sebagai identitas bisnis |
| Alias kolom SQL warisan (`LOCATION AS "RISKLOCATION"`, `NOPOLIS AS "NoKTP"`, `picteknik AS "UserAdmin"`) | Nama tidak mencerminkan isi; dipaksa agar cocok dengan clipboard Pega | Penamaan ulang menyeluruh |

# Lampiran B — Decision Log

Semua pertanyaan discovery, pilihan yang ditawarkan, jawaban yang dipilih, dan keputusan akhir
yang dihasilkan. Setiap keputusan punya ID (`D-xx`) yang direferensikan dari dokumen Steering lain.

Status: `DECIDED` = final · `OPEN` = masih perlu konfirmasi pihak lain · `ASSUMED` = asumsi kerja, wajib divalidasi

---

## Sesi 1 — 2026-09-07 — Discovery Awal

### D-01 · Strategi Database
**Status:** DECIDED

**Pertanyaan:** Database POOLDATA/GENERAL dipakai bersama sistem lain (ASMD, SIMASNET, SMI, OPJAVA lewat DB Link) dan berisi 86 stored procedure. Bagaimana nasibnya di migrasi ke PostgreSQL?

**Pilihan yang ditawarkan:**
1. Hanya data milik Claim PNC yang pindah ke PostgreSQL
2. Seluruh POOLDATA + GENERAL ikut pindah ke PostgreSQL
3. PostgreSQL hanya untuk data baru, Oracle tetap dibaca (hybrid jangka panjang)
4. Belum diputuskan — perlu dibahas dengan tim DBA/infra

**Jawaban (Other):**
> "Saya ingin seluruhnya. Dan untuk model pembuatannya nanti buat menjadi postgre friendly, seperti procedurenya di tulis langsung di scriptnya, bukan di panggil seperti oracle. Note : untuk sementara memang masih menggunakan oracle, perpindahan ke postgre masih blm tahu kapan namun nantinya pasti bakal pindah ke postgre"

**Keputusan akhir:**
- Target akhir adalah **PostgreSQL untuk seluruh data**, bukan hanya subset Claim PNC.
- **Tidak ada pemanggilan stored procedure** dari aplikasi. Seluruh SQL ditulis langsung di kode Go.
- Desain skema, tipe data, indexing, dan gaya SQL mengacu ke **PostgreSQL sebagai kanonikal**.
- **Runtime sementara tetap Oracle**, tanggal cutover belum ditentukan. Aplikasi wajib bisa berjalan di Oracle tanpa mengorbankan desain PostgreSQL-first.

**Dampak ke Steering:** Database Strategy · Technical Strategy · Folder Structure · Migration Strategy · Risk Analysis

---

### D-02 · Perlakuan Stored Procedure Oracle
**Status:** DECIDED

**Pertanyaan:** Ada 86 stored procedure/package Oracle (INSERT_PLADLA, PEGA_JSON_KLAIM_PNC, CONVERTJSONPRODUCTION, dll) yang berisi logika bisnis nyata. Bagaimana perlakuannya?

**Pilihan yang ditawarkan:**
1. Ditulis ulang jadi logika Go di application layer
2. Diport ke PL/pgSQL apa adanya
3. Hybrid: yang murni data-access ke Go, yang transaksional berat tetap di PL/pgSQL
4. Belum tahu — isi procedure-nya sendiri belum pernah dilihat

**Jawaban:** Ditulis ulang jadi logika Go di application layer

**Keputusan akhir:** Seluruh logika bisnis yang kini berada di dalam stored procedure dinaikkan ke application layer Go. Database menjadi penyimpanan murni.

**Konsekuensi terbuka:** Source procedure **tidak ada** di export XML ini. Diperlukan DDL/source dari DBA sebelum modul yang bergantung padanya bisa diimplementasikan → **R-01** di Risk Analysis.

**Koreksi jumlah (2026-09-08):** angka "86 stored procedure" pada pertanyaan di atas adalah hitungan awal yang terlalu tinggi — pemindaian pertama ikut menangkap nama tabel pada `INSERT INTO tabel (kolom…)` sebagai pemanggilan procedure. Angka terverifikasi: **56 procedure + 8 function lokal = 64 objek** yang harus diminta ke DBA, ditambah 6 function remote lewat DB Link yang masuk lingkup D-25/R-03. Teks pertanyaan sengaja **tidak diubah**, karena Decision Log mencatat apa yang benar-benar ditanyakan saat itu. Daftar lengkapnya di [`19-GAP-EXPORT-DETAIL.md`](19-GAP-EXPORT-DETAIL.md).

**Dampak ke Steering:** Technical Strategy · Module Breakdown · Risk Analysis · Migration Strategy

---

### D-03 · Scope Migrasi (GCNMFW vs GISFW)
**Status:** DECIDED

**Pertanyaan:** Seberapa besar scope migrasi? Ada 2 framework: GCNMFW (Claim PNC, 633 activity) dan GISFW (policy/underwriting, 141 activity) yang saling terikat.

**Pilihan yang ditawarkan:**
1. Hanya GCNMFW (Claim PNC) — GISFW jadi dependensi eksternal
2. GCNMFW + GISFW sekaligus
3. GCNMFW dulu, GISFW menyusul di fase berikutnya
4. Hanya sebagian modul Claim PNC sebagai pilot

**Jawaban (Other):**
> "buat yg ada dulu, semua sudah sy lampirkan di xmlnya. Memang benar untuk GISFW berisi data polis yang dibutuhkan saat claim, namun GISFW sedang di develop tim lain."

**Keputusan akhir:** Scope = seluruh rule yang ada di export XML ini. GISFW **tidak dimiliki** tim ini (sedang dikembangkan tim lain), tetapi struktur data polis yang dipakai saat klaim tetap masuk scope sebagai data yang dikonsumsi. Batas kepemilikan diatur di **D-04**.

**Dampak ke Steering:** Bounded Context · Module Breakdown · Domain Model

---

### D-04 · Batas Bounded Context terhadap Data Polis (GISFW)
**Status:** DECIDED

**Pertanyaan:** GISFW (data polis) dikembangkan tim lain, tapi 141 activity-nya ada di export ini dan dipakai saat klaim. Bagaimana batas kepemilikan datanya?

**Pilihan yang ditawarkan:**
1. Claim PNC menyimpan salinan snapshot polis saat klaim dibuat
2. Claim PNC memanggil API GISFW tim lain secara real-time
3. Kita ikut migrasikan GISFW juga dalam project ini
4. Belum jelas — perlu koordinasi dengan tim GISFW

**Jawaban:** Claim PNC menyimpan salinan snapshot polis saat klaim dibuat

**Keputusan akhir:** Domain Klaim menyimpan **snapshot polis** pada saat klaim diregistrasi — melanjutkan pola `JSON_POLIS`/`JSON_KLAIM` yang sudah ada, tetapi dengan skema eksplisit. Klaim menjadi kebal terhadap perubahan polis setelah registrasi dan tidak bergantung pada ketersediaan sistem tim lain saat runtime.

**Dampak ke Steering:** Domain Model · Bounded Context · Database Strategy · Integration Strategy

---

### D-05 · Strategi Cutover
**Status:** DECIDED

**Pertanyaan:** Aplikasi lama masih aktif dikembangkan (124 activity diubah tahun 2026). Bagaimana strategi cutover-nya?

**Pilihan yang ditawarkan:**
1. Strangler Fig — modul dipindah bertahap, dua sistem jalan paralel
2. Big Bang — bangun penuh, lalu potong sekali waktu
3. Per case type — Register dulu, lalu Komite, OpenProtection, ReceiveDocument
4. Belum diputuskan

**Jawaban:** Strangler Fig — modul dipindah bertahap, dua sistem jalan paralel

**Keputusan akhir:** Migrasi bertahap. Pega dan aplikasi Go berjalan paralel; modul dialihkan satu per satu. Konsekuensi: dibutuhkan strategi berbagi data/status antar kedua sistem selama masa paralel.

**Dampak ke Steering:** Migration Strategy · Deployment Strategy · Risk Analysis · Database Strategy

---

### D-06 · Penanganan Dialek SQL (Oracle sekarang → PostgreSQL nanti)
**Status:** SUPERSEDED oleh **D-20** — pertanyaan ini diangkat ulang dengan bukti terukur pada sesi grilling dan diputuskan di sana

**Pertanyaan:** Go harus jalan di Oracle dulu, lalu pindah ke PostgreSQL. SQL ditulis langsung di kode. Bagaimana menangani perbedaan dialek (NVL vs COALESCE, ROWNUM vs LIMIT, JSON_TABLE vs jsonb, TO_CHAR, sequence)?

**Pilihan yang ditawarkan:**
1. Repository interface, dua implementasi SQL (Oracle & PostgreSQL) berdampingan
2. Tulis SQL portabel (ANSI) sebisa mungkin, dialek khusus hanya di sedikit tempat
3. Pakai query builder / ORM yang menangani dialek otomatis
4. Bangun langsung untuk PostgreSQL saja, Oracle diakses lewat lapisan sementara

**Jawaban (Other):**
> "buat langsung postgre friendly, maksudnya adalah sudah tidak ada call procedure seperti sistem oracle, dan lain sebagainya. Note : untuk sementara masih pakai database oracle."

**Catatan:** Jawaban menegaskan *tidak ada call procedure* dan *PostgreSQL-friendly*, tetapi belum memilih mekanisme teknis untuk menjembatani dua dialek.

**Tindak lanjut:** pertanyaan diajukan ulang pada sesi grilling setelah seluruh 652 rule SQL dianalisis dan dikategorikan, sehingga pilihannya dapat disajikan dengan volume yang terukur. Hasilnya → **D-20**.

**Dampak ke Steering:** Database Strategy · Technical Strategy · Coding Standards · Testing Strategy

---

### D-07 · Authentication & Authorization
**Status:** DECIDED

**Pertanyaan:** Autentikasi saat ini memakai operator Pega (DATAPEGA.PR_OPERATORS) dengan 22 access group. Di sistem baru pakai apa?

**Pilihan yang ditawarkan:**
1. SSO korporat Sinarmas (LDAP / Active Directory / OIDC)
2. User & password sendiri di aplikasi baru (JWT)
3. Migrasi tabel operator Pega apa adanya ke sistem baru
4. Belum diputuskan

**Jawaban (Other):**
> "autentikasi nantinya akan di buat baru. gambaran singkatnya, user akan login memasukkan username dan pwd yg terdaftar di internal API hcc/hcq. Jika sudah berhasil login, hak akses menu akan di buat terpisah lagi nantinya di aplikasi klaim ini, artinya akan ada tabel baru yg menyimpan user bisa akses menu apa saja. Tapi tabelnya belum di buat"

**Pertanyaan lanjutan:** Untuk API HCC/HCQ, informasi apa yang dikembalikan setelah login berhasil?

**Pilihan yang ditawarkan:**
1. Hanya status berhasil/gagal — data user lain diambil dari tabel aplikasi
2. Mengembalikan token yang harus dipakai untuk request berikutnya
3. Mengembalikan profil lengkap user (NIK, nama, cabang, jabatan, email)
4. Saya belum tahu detail responsnya

**Jawaban:** Mengembalikan profil lengkap user (NIK, nama, cabang, jabatan, email)

**Keputusan akhir:**
- **Authentication** didelegasikan ke **API internal HCC/HCQ** (username + password). Respons memuat profil lengkap: NIK, nama, cabang, jabatan, email.
- **Authorization** dimiliki dan dikelola sepenuhnya oleh aplikasi Claim PNC lewat tabel baru yang memetakan user → menu/permission. Tabel belum ada, akan dirancang dalam project ini.
- Aplikasi menerbitkan session/token miliknya sendiri setelah HCC/HCQ memvalidasi kredensial.

**Dampak ke Steering:** Security · Authentication · Authorization · Domain Model · API Strategy

---

### D-08 · Deployment Target
**Status:** DECIDED

**Pertanyaan:** Di mana sistem baru akan dijalankan?

**Pilihan yang ditawarkan:**
1. On-premise VM / bare metal (data center Sinarmas)
2. On-premise Kubernetes / container platform
3. Cloud (AWS / GCP / Azure)
4. Belum diputuskan / masih dibahas tim infra

**Jawaban:** On-premise VM / bare metal (data center Sinarmas)

**Keputusan akhir:** Deployment ke VM/bare metal on-premise di data center Sinarmas. Tanpa asumsi Kubernetes. Strategi deployment, monitoring, dan scaling harus realistis untuk lingkungan VM.

**Dampak ke Steering:** Deployment Strategy · Scalability · Performance Strategy · Monitoring · Infrastructure

---

### D-09 · Latar Belakang Tim Pengembang
**Status:** DECIDED

**Pertanyaan:** Siapa yang akan membangun dan merawat sistem baru ini?

**Pilihan yang ditawarkan:**
1. Tim internal yang sekarang pegang Pega (belum terbiasa Go/JS modern)
2. Developer Golang yang sudah berpengalaman
3. Developer frontend JavaScript/TypeScript berpengalaman (React/Vue)
4. Vendor / konsultan eksternal

**Jawaban:** Tim internal yang sekarang pegang Pega (belum terbiasa Go/JS modern)

**Keputusan akhir:** Tim adalah developer Pega internal yang akan dilatih ulang. Ini menjadi **batasan utama** dalam pemilihan frontend dan gaya kode backend:
- Utamakan learning curve landai dan jumlah konsep yang sedikit, bukan yang paling canggih.
- Steering harus sangat preskriptif: struktur folder, penamaan, dan pola baku yang seragam.
- Hindari abstraksi berat, generic tingkat lanjut, dan "magic" framework.

**Dampak ke Steering:** Frontend Decision · Coding Standards · Folder Structure · Maintainability · Testing Strategy

---

### D-10 · Volume Data
**Status:** DECIDED (angka pasti masih OPEN)

**Pertanyaan:** Berapa volume data dan beban puncaknya?

**Pilihan yang ditawarkan:**
1. Saya tahu perkiraan angkanya — akan saya isi
2. Belum tahu, perlu ditanyakan ke tim DBA
3. Skala kecil — ratusan klaim per bulan, data historis < 1 juta baris
4. Skala besar — ribuan klaim per bulan, data historis puluhan juta baris

**Jawaban:** Skala besar — ribuan klaim per bulan, data historis puluhan juta baris

**Keputusan akhir:** Sistem menangani **ribuan klaim per bulan** dengan **data historis puluhan juta baris**, tetapi hanya **200–300 user aktif harian**. Karakteristiknya: *beban data besar, konkurensi rendah*. Optimasi diarahkan ke volume data (indexing, paginasi keyset, partisi, arsip) — **bukan** ke konkurensi tinggi.

**Dampak ke Steering:** Performance Strategy · Database Strategy · Scalability · Non Functional Requirements

---

### D-11 · Strategi Reporting & Export
**Status:** DECIDED

**Pertanyaan:** Ada 56 Report Definition + banyak export CSV/Excel (pxConvertResultsToCSV dipanggil 95x). Seberapa penting fitur laporan di sistem baru?

**Pilihan yang ditawarkan:**
1. Sangat penting — semua laporan harus ada sejak hari pertama
2. Penting, tapi bisa menyusul setelah proses inti jalan
3. Sebaiknya dipindah ke tools BI terpisah (Metabase, Superset, Power BI)
4. Belum diputuskan

**Jawaban (Other):**
> "buat script sendiri untuk generate pdf / excel / csv, karena kita sudah tidak lagi menggunaan pega."

**Keputusan akhir:** Pembuatan PDF, Excel, dan CSV **diimplementasikan sendiri di dalam aplikasi Go**. Tidak memakai engine reporting Pega maupun tools BI eksternal. Steering memerlukan bab Reporting & Document Generation Strategy tersendiri, termasuk penanganan export bervolume besar agar tidak membebani transaksi.

**Dampak ke Steering:** Module Breakdown · Performance Strategy · Technical Strategy · Non Functional Requirements

---

### D-12 · Penggunaan Mobile / Lapangan
**Status:** DECIDED

**Pertanyaan:** Ada kategori foto survey (FotoDepan, FotoRangka, FotoSelfieKendaraan, FotoLokasiDebitur) dan role PNCSurveyor. Apakah ada user yang memakai aplikasi ini dari lapangan/HP?

**Pilihan yang ditawarkan:**
1. Ya — surveyor/adjuster memakai HP di lapangan untuk upload foto
2. Tidak — semua user memakai desktop di kantor
3. Mayoritas desktop, tapi surveyor kadang pakai tablet/HP
4. Surveyor pakai aplikasi terpisah, bukan aplikasi ini

**Jawaban:** Mayoritas desktop, tapi surveyor kadang pakai tablet/HP

**Keputusan akhir:** Frontend **desktop-first namun responsive**. Layar padat data (inbox, report, grid akseptasi) dioptimalkan untuk desktop. Layar yang dipakai surveyor — terutama upload foto dan input hasil survei — wajib nyaman di tablet/HP dan tahan jaringan lambat.

**Dampak ke Steering:** Frontend Decision · Non Functional Requirements · Performance Strategy

---

### D-13 · Ekspektasi UI/UX
**Status:** DECIDED

**Pertanyaan:** Bagaimana ekspektasi tampilan sistem baru dibanding Pega sekarang (74 layar, 269 section)?

**Pilihan yang ditawarkan:**
1. Tiru alur & tata letak Pega — supaya user tidak perlu belajar ulang
2. Desain ulang total dengan UX yang lebih baik
3. Alur bisnis sama, tampilan dimodernkan
4. Belum dipikirkan — fokus dulu ke fungsionalitas

**Jawaban:** Tiru alur & tata letak Pega — supaya user tidak perlu belajar ulang

**Keputusan akhir:** Alur kerja, urutan langkah, penempatan field, dan tata letak layar **mengikuti Pega yang ada**. Perubahan tampilan dijaga seminimal mungkin agar user tidak perlu pelatihan ulang.

**Konsekuensi yang harus disadari:**
- Aplikasi mewarisi UX Pega yang padat (grid lebar, form panjang, banyak tab).
- Frontend harus kuat menangani **tabel/grid kompleks** dan **form panjang bernested repeat** — ini jadi kriteria penilaian utama pemilihan framework.
- Kesempatan perbaikan UX ditunda ke fase pasca-migrasi.

**Dampak ke Steering:** Frontend Decision · Module Breakdown · Future Enhancement

---

### D-14 · Aturan Penjenjangan Komite
**Status:** DECIDED (matriks detail: OPEN)

**Pertanyaan:** Komite bisa berjenjang sampai 4 level (KomiteCount 1–4 → workbasket komitepnc..komitepnc4). Apa yang menentukan berapa level yang dibutuhkan?

**Pilihan yang ditawarkan:**
1. Berdasarkan nilai klaim — makin besar makin banyak level
2. Berdasarkan jenis bisnis / GroupPanel
3. Kombinasi nilai klaim dan jenis bisnis
4. Ditentukan manual oleh user, bukan otomatis

**Jawaban:** Kombinasi nilai klaim dan jenis bisnis

**Keputusan akhir:** Jenjang komite ditentukan **matriks (nilai klaim × jenis bisnis)**. Matriks ini menjadi **master data yang dapat dikonfigurasi**, bukan kode. Ambang yang ditemukan di kode lama (50.000.000 umum; 30.000.000 untuk PA/Travel; 3.500 untuk server Timor-Leste) diperlakukan sebagai *nilai awal* yang harus divalidasi ulang oleh tim bisnis.

**Terbuka:** Matriks lengkapnya belum ada di tangan kita.

**Temuan lanjutan (2026-09-08):** matriks ini ternyata **sudah berupa master data** di tabel `POOLDATA.EMAILKOMITE`. Jumlah jenjang = jumlah baris hasil query (`KomiteLoop := pxResultCount`), urutan jenjang = kolom `DEGREE`, ambang nilai = kolom `LIMIT_BOTTOM`. Yang belum kita punya hanyalah **isi tabelnya** — cukup satu query ke DBA. Yang tetap perlu dikonfirmasi ke tim bisnis adalah logika pemilihan di `SetEmailKomite`, yang bercampur dengan nama orang tertentu dan percabangan hostname server. Rincian lengkap: [`20-DETAIL-KOMITE-DBLINK.md`](20-DETAIL-KOMITE-DBLINK.md).

**Dampak ke Steering:** Domain Model · Module Breakdown · Configuration · Risk Analysis

---

### D-15 · Nilai Hardcode → Konfigurasi
**Status:** DECIDED

**Pertanyaan:** Di kode ada nilai hardcode: email pimpinan/UW, user ID tertentu (MORASOTARDODOTARIGAN, ELLENSUPRIYATI), ambang komite, dan hostname server. Bahkan ada blok bertanda "TESTING" yang menimpa email UW asli di alur produksi. Bagaimana perlakuannya di sistem baru?

**Pilihan yang ditawarkan:**
1. Semua jadi konfigurasi / master data yang bisa diubah tanpa deploy
2. Konfigurasi untuk yang sering berubah, konstanta kode untuk yang jarang
3. Salin apa adanya dulu supaya perilaku identik, dibersihkan belakangan
4. Perlu ditinjau satu per satu bersama tim bisnis

**Jawaban:** Semua jadi konfigurasi / master data yang bisa diubah tanpa deploy

**Keputusan akhir:** **Tidak ada nilai bisnis yang boleh di-hardcode.** Semua ambang nilai, penerima notifikasi, pemetaan role, alamat endpoint, dan aturan penjenjangan disimpan sebagai konfigurasi atau master data yang dapat diubah tanpa deploy ulang.

**Konsekuensi:** Dibutuhkan **modul Administrasi/Master Data** beserta layar pengelolanya — tambahan scope yang harus masuk perencanaan sejak awal.

**Blok "TESTING" di `InputRegister_act` step 106–109 tidak dibawa ke sistem baru.** Perilaku benar yang dipakai adalah email UW asli per Group Panel.

**Dampak ke Steering:** Configuration · Module Breakdown · Security · Coding Standards · Migration Strategy

---

### D-16 · Penyimpanan Dokumen
**Status:** DECIDED

**Pertanyaan:** Dokumen/lampiran sekarang tersimpan di 3 tempat (BLOB Oracle, tabel Pega, object storage eksternal via API). Di sistem baru bagaimana?

**Pilihan yang ditawarkan:**
1. Object storage (MinIO / S3-compatible) on-premise, DB hanya simpan metadata
2. Tetap pakai API storage internal yang sudah ada (app13/app8 Sinarmas)
3. Simpan sebagai BLOB di database seperti sekarang
4. Belum diputuskan

**Jawaban:** Tetap pakai API storage internal yang sudah ada (app13/app8 Sinarmas)

**Keputusan akhir:** Dokumen disimpan lewat **API storage internal Sinarmas yang sudah berjalan**. Database aplikasi hanya menyimpan **metadata dan referensi** (ID gambar, URL, masa berlaku, kategori). Tiga mekanisme penyimpanan yang ada sekarang **disatukan menjadi satu jalur**.

**Konsekuensi:** Ketersediaan dokumen bergantung pada layanan tim lain. Steering harus mengatur penanganan kegagalan, retry, dan masa berlaku URL.

**Dampak ke Steering:** Document Storage Strategy · Integration · Error Handling · Risk Analysis

---

### D-17 · Proses Terjadwal / Batch
**Status:** OPEN — gap discovery

**Pertanyaan:** Apakah ada proses terjadwal / batch di sistem sekarang (reminder SLA, kirim email otomatis, sinkronisasi ke kasir/GL, generate laporan harian)?

**Pilihan yang ditawarkan:**
1. Ya — ada job terjadwal, tapi belum tahu detail lengkapnya
2. Ada reminder / notifikasi email otomatis berbasis SLA
3. Ada sinkronisasi/pengiriman data otomatis ke sistem lain
4. Tidak ada — semua proses dipicu aksi user

**Jawaban:** Ya — ada job terjadwal, tapi saya belum tahu detail lengkapnya

**Keputusan akhir:** Sistem baru **membutuhkan komponen scheduler**. Namun daftar job-nya belum diketahui.

**Penyebab gap:** Export XML ini **tidak memuat rule Agent maupun Queue Processor Pega**, sehingga job terjadwal tidak dapat dianalisis dari source. → **R-02** di Risk Analysis.

**Tindak lanjut:** Minta export `Rule-Agent-Queue`, `Rule-Async-QueueProcessor`, dan `Rule-Job-Scheduler` dari sistem Pega.

**Dampak ke Steering:** Module Breakdown · Deployment Strategy · Risk Analysis · Future Architecture

---

### D-18 · Empat Konsep Status
**Status:** DECIDED (arti kode 1142–1151: OPEN)

**Pertanyaan:** Ada 4 konsep status yang tumpang tindih: StatusWork (Pega), StatusClaim (kode 1142–1151), ClaimStatus (0/1), dan StatusPosisi (On Progress/Done). Apakah ini memang 4 hal berbeda?

**Pilihan yang ditawarkan:**
1. Ya, 4 hal berbeda — semuanya harus dipertahankan
2. Sebenarnya duplikat — hasil penumpukan bertahun-tahun, boleh disederhanakan
3. StatusClaim yang utama, sisanya turunan/teknis
4. Saya perlu cek dulu ke user bisnis

**Jawaban:** Ya, 4 hal berbeda — semuanya harus dipertahankan

**Keputusan akhir:** Keempat status dimodelkan sebagai **konsep terpisah dengan nama yang tidak bisa tertukar** (lihat `CONTEXT.md` bagian Status):
- Status Proses (alur kerja)
- Status Klaim (status bisnis, kode 1142–1151)
- Flag Klaim (penanda biner)
- Status Posisi Progres (tahapan progres)

**Terbuka:** Label dan arti tiap kode `1142`–`1151` tersimpan di master status `V_STS_CLAIM` yang **tidak ada di export XML**. Harus diambil dari database sebelum state machine klaim difinalkan.

**Dampak ke Steering:** Domain Model · Database Strategy · Risk Analysis

---

### D-19 · Ubiquitous Language (Glossary Domain)
**Status:** DECIDED

**Pertanyaan:** Arti singkatan domain yang dipakai di seluruh kode.

**Jawaban (Other) — dikonfirmasi langsung oleh pemilik bisnis:**
> RCL = Rejected klaim
> PUCL = Proses Ulang klaim
> PLA = Preliminary Loss Advice, untuk memberitahu nilai estimasi klaim kepada koasuransi/reasuransi
> DLA = Definite Loss Advice, untuk memberitahu nilai akseptasi klaim kepada koasuransi/reasuransi
> Pre DLA = untuk memberitahu nilai klaim yang akan di aksep kepada koasuransi/reasuransi sebelum akseptasi dilakukan
> LOD = Letter of Discharge, surat pemberitahuan nilai ganti rugi klaim yang disetujui asuransi kepada tertanggung
> TKA = jenis bisnis untuk asuransi Tenaga Kerja Asing
> KBRU = Kali Besar Raya Utama, salah satu broker asuransi
> BPPDAN = Badan Pengelola Pusat Data Asuransi Nasional, salah satu jenis treaty asuransi
> MBU vs Non MBU = Motor Business Unit vs Non Motor Business Unit, Non MBU = PNC
> SPK = Sinarmas Penjaminan Kredit
> OS = OutStanding

**Keputusan akhir:** Glossary lengkap ditulis di **`CONTEXT.md`** dan menjadi **ubiquitous language wajib** — dipakai konsisten di nama tabel, struct Go, endpoint API, label UI, dan dokumentasi.

**Temuan penting:** **Non-MBU = PNC.** Aplikasi "Claim PNC" menangani klaim **Non-Motor**. Ini mengoreksi asumsi awal dan mempengaruhi penamaan bounded context.

**Penamaan yang sengaja diubah** (lihat `CONTEXT.md`):
- `Object`/`ObjectList` → **Objek Pertanggungan / Insured Item** (menghindari tabrakan makna pemrograman)
- `Adjustment`/`AdjustmentList` → **Settlement Line** (isinya nilai penyelesaian, bukan proses adjusting)
- Alias kolom SQL warisan dinamai ulang menyeluruh

**Dampak ke Steering:** Domain Model · Bounded Context · Coding Standards · Database Strategy · API Strategy

---

## Sesi 2 — 2026-09-07 — Grilling (skill `mattpocock-skills:grilling`)

### D-20 · Mekanisme Dialek SQL
**Status:** DECIDED

**Pertanyaan (Grilling Q1):** Bagaimana mekanisme menangani dua dialek SQL (Oracle sekarang, PostgreSQL nanti)?

**Pilihan yang ditawarkan:**
1. Repository interface + SQL per dialek, PostgreSQL kanonikal *(rekomendasi saya)*
2. Satu set SQL portabel yang jalan di kedua database
3. Tulis SQL PostgreSQL saja, Oracle ditangani lapisan penerjemah
4. Pakai query builder yang menangani dialek otomatis

**Jawaban:** Satu set SQL portabel yang jalan di kedua database
*(berbeda dari rekomendasi saya — keputusan pemilik project yang diikuti)*

**Bukti terukur yang dikumpulkan sebelum keputusan** (dari 652 rule SQL, 534 KB):

| Kategori | Volume |
|---|---|
| Sudah portabel | `CASE WHEN` 317× · `\|\|` 191× · `FETCH FIRST` 72× · `SUBSTR` 69× · `ROW_NUMBER()` 14× · CTE 8× |
| Terjemahan mekanis | `TRUNC` 150× · `NVL` 100× · `SYSDATE` 69× · `DECODE` 16× · `DUAL` 12× · `INSTR` 11× · `LISTAGG` 4× · `(+)` 2× |
| Tidak portabel | `JSON_VALUE/QUERY` 195× · `JSON_TABLE` 27× · DB Link 64× · `TO_CHAR` 411× · `SEQ.NEXTVAL` |

**Keputusan akhir:** **Satu set SQL portabel** yang berjalan di Oracle 19c dan PostgreSQL 17+, dengan **tiga pengecualian terkelola**:

1. **Pemformatan tanggal dan angka dikeluarkan dari SQL ke Go.** Menghapus 411 pemakaian `TO_CHAR` sekaligus memperbaiki bug senyap: SQL saat ini mengembalikan tanggal sebagai string `'dd/mm/yyyy'`, sehingga pengurutan dan penyaringan tanggal salah secara diam-diam.
2. **Generator nomor klaim** adalah satu-satunya tempat dengan sakelar dialek eksplisit (lihat D-23).
3. **Paginasi diseragamkan ke `OFFSET … FETCH NEXT … ROWS ONLY`**, menggantikan 68 pemakaian `ROWNUM`. Pola ini sudah dipakai di 35 rule pada codebase yang ada, jadi bukan hal baru bagi tim.

**Prasyarat wajib:** PostgreSQL 17+ (lihat D-24). Tanpa itu keputusan ini tidak bisa dijalankan.

**Dampak ke Steering:** Database Strategy · Coding Standards · Technical Strategy · Testing Strategy

---

### D-21 · Berbagi Data Selama Jalan Paralel
**Status:** DECIDED (desain tabel baru: OPEN)

**Pertanyaan (Grilling Q2):** Selama Pega dan Go jalan paralel, bagaimana keduanya berbagi data klaim?

**Pilihan yang ditawarkan:**
1. Satu database bersama, kepemilikan tabel dibagi per modul *(rekomendasi saya)*
2. Database terpisah dengan sinkronisasi dua arah
3. Go jadi satu-satunya penulis, Pega diubah agar hanya membaca
4. Perlu dibahas dulu dengan tim DBA dan tim Pega

**Jawaban (Other):**
> "memang 1 database namun untuk beberapa data yg saat ini masih disimpan di pega nantinya akan dibuatkan kemudian ditampung di tabel baru. untuk sementara tabelnya belum ada"

**Keputusan akhir:** **Satu database bersama.** Data yang saat ini hidup di tabel milik engine Pega dipindahkan ke **tabel baru milik aplikasi Claim PNC**. Tabel-tabel ini belum ada dan harus dirancang dalam project ini.

**Cakupan yang terukur dari source — tabel Pega yang harus digantikan:**

| Tabel Pega | Dibaca oleh | Isi |
|---|---|---|
| `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` | **116 rule** | Tabel header klaim (pyID, policyno, qqname, statusclaim, statuswork, dateofloss, grouppanel, surveyortype, …) |
| `DATAPEGA.PC_ASSIGN_WORKLIST` | 18 rule | Penugasan per orang |
| `DATAPEGA.PC_ASSIGN_WORKBASKET` | 6 rule | Antrean bersama |
| `DATAPEGA.PR_OPERATORS` | 6 rule | Master user |
| `DATAPEGA.PC_LINK_ATTACHMENT` | 3 rule | Kaitan lampiran |
| `DATAPEGA.PC_DATA_WORKATTACH` | 2 rule | Lampiran |
| `DATAPEGA.PR_SYS_LOCKS` | 1 rule | Penguncian record |

**Aturan kepemilikan selama masa paralel:** setiap tabel hanya boleh ditulis oleh **satu** sistem. Modul yang sudah pindah ke Go memiliki tabelnya, Pega hanya membaca; sebaliknya untuk modul yang belum pindah. **Tidak ada sinkronisasi dua arah.**

**Dampak ke Steering:** Migration Strategy · Database Strategy · Domain Model · Risk Analysis

---

### D-22 · Penomoran Klaim
**Status:** DECIDED

**Pertanyaan (Grilling Q3):** Bagaimana penomoran klaim dan penanganan data historis?

**Pilihan yang ditawarkan:**
1. Pertahankan `PNC-xxxx`, buang prefix `ASM-FW-GCNMFW-WORK` *(rekomendasi saya)*
2. Pertahankan format lama sepenuhnya termasuk prefix
3. Mulai penomoran baru, data lama dipetakan
4. Perlu dicek dulu sistem lain mana saja yang memakai format ini

**Jawaban (Other):**
> "Akan gunakan format PNCN-xxxx tanpa prefix ASM-FW-GCNMFW-WORK dan penomorannya gunakan sequence di oracle yang akan diberi nama POOLDATA.CLAIM_NO_NONPEGA_SEQ dengan syntax 'PNCN-' || TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)"

**Keputusan akhir:** Klaim yang dibuat sistem baru memakai format **`PNCN-xxxx`**, dihasilkan dari sequence **`POOLDATA.CLAIM_NO_NONPEGA_SEQ`**. Prefix `ASM-FW-GCNMFW-WORK` **tidak dipakai lagi**.

**Kenapa ini lebih baik dari rekomendasi awal saya:** prefix `PNCN` membuat klaim buatan sistem baru **langsung bisa dibedakan** dari klaim warisan Pega (`PNC-xxxx`) tanpa perlu tabel pemetaan. Selama masa paralel yang panjang, kemampuan membedakan asal data ini sangat berharga untuk penelusuran dan rekonsiliasi.

**Pengecualian portabilitas yang diterima secara sadar:** sintaks sequence berbeda antara Oracle (`SEQ.NEXTVAL`) dan PostgreSQL (`nextval('seq')`). Ini **satu-satunya** tempat dengan sakelar dialek, diisolasi di satu generator nomor klaim. Memaksakan portabilitas di sini akan mengorbankan jaminan keunikan nomor — pertukaran yang tidak sepadan.

**Klaim warisan `PNC-xxxx` tetap dapat dibaca dan diproses**, hanya tidak lagi dibuat baru.

**Dampak ke Steering:** Domain Model · Database Strategy · Migration Strategy

---

### D-23 · Pilihan Frontend
**Status:** DECIDED

**Pertanyaan (Grilling Q4):** Frontend mana yang dipilih?

**Analisis lengkap:** `01-FRONTEND-ANALYSIS.md` — membandingkan React, Next.js, Vue 3, Nuxt, Angular, Svelte, SolidJS, dan Go+templ+HTMX terhadap kelebihan, kekurangan, performa, maintainability, kecocokan dengan Go, skalabilitas, ekosistem, dan learning curve.

**Pilihan yang ditawarkan:**
1. Vue 3 + TypeScript + Vite + PrimeVue *(rekomendasi saya)*
2. React + TypeScript + Vite + TanStack Table / AG Grid
3. Angular
4. Go + templ + HTMX

**Jawaban:** **React + TypeScript + Vite + TanStack Table / AG Grid**
*(berbeda dari rekomendasi saya — keputusan pemilik project yang diikuti)*

**Keputusan akhir:** Frontend memakai **React + TypeScript + Vite**, dengan **TanStack Table** atau **AG Grid** untuk kebutuhan grid berat (268 dari 269 section bergrid; inbox utama 18–27 kolom). Dibangun sebagai **SPA murni** yang disajikan sebagai berkas statis oleh binary Go — tidak ada runtime Node.js di production.

**Yang gugur dan alasannya:**
- **Next.js dan Nuxt** — keunggulan utamanya SSR/SEO, sementara aplikasi ini ada di balik login tanpa akses publik sehingga manfaatnya nol; yang tersisa hanya biayanya, yaitu runtime Node.js kedua di VM on-premise.
- **Svelte dan SolidJS** — ekosistem DataTable enterprise belum memadai untuk kebutuhan inti kita.
- **Angular** — paling terstruktur, tapi RxJS dan Dependency Injection menghantam langsung batasan terkuat kita (D-09).

**Mitigasi wajib atas risiko yang diketahui:** React tidak opinionated, dan tim di D-09 belum terbiasa JS modern. Risiko nyatanya adalah **kode berbeda gaya di setiap layar**. Karena itu Steering menetapkan konvensi yang mengikat untuk struktur folder, pengambilan data, pengelolaan state, penanganan form, dan pola tabel — bukan sekadar anjuran. Ini bukan formalitas: tanpa itu, kelemahan yang jadi alasan saya tidak merekomendasikan React akan benar-benar terjadi.

**Dampak ke Steering:** Frontend Architecture · Coding Standards · Folder Structure · Testing Strategy · Maintainability

---

### D-24 · Versi PostgreSQL
**Status:** DECIDED

**Pertanyaan (Grilling Q5):** SQL portabel untuk JSON hanya mungkin bila PostgreSQL 17+. Ada 222 pemakaian JSON_VALUE/JSON_TABLE di 29 rule. Versi PostgreSQL apa yang jadi target?

**Pilihan yang ditawarkan:**
1. PostgreSQL 17 atau lebih baru *(rekomendasi saya)*
2. PostgreSQL 15 atau 16
3. Menyesuaikan versi standar Sinarmas
4. Belum ditentukan — tim infra yang memutuskan

**Jawaban:** PostgreSQL 17 atau lebih baru

**Keputusan akhir:** Target database adalah **PostgreSQL 17 atau lebih baru**. Ini **persyaratan teknis mengikat**, bukan preferensi.

**Alasannya:** PostgreSQL 17 mendukung fungsi SQL/JSON standar `JSON_TABLE`, `JSON_VALUE`, dan `JSON_QUERY` dengan sintaks yang sama seperti Oracle. Ini membuat **seluruh 222 query JSON portabel apa adanya**. PostgreSQL 16 ke bawah belum punya `JSON_TABLE`, sehingga 29 rule harus ditulis ulang memakai operator `jsonb` khas PostgreSQL — dan keputusan D-20 (SQL portabel) menjadi tidak bisa dijalankan.

**Dampak ke Steering:** Database Strategy · Infrastructure · Risk Analysis

---

### D-25 · Pengganti DB Link
**Status:** DECIDED (kontrak API per sistem: OPEN)

**Pertanyaan (Grilling Q6):** PostgreSQL tidak punya DB Link. Ada 64 pemakaian ke 6 database lain untuk ambil data HRD, pembayaran GL, master sales, dan polis. Bagaimana penggantinya?

**Pilihan yang ditawarkan:**
1. Ganti dengan pemanggilan API ke sistem pemilik data *(rekomendasi saya)*
2. Replikasi data yang dibutuhkan ke database Claim PNC
3. Pakai Foreign Data Wrapper PostgreSQL
4. Perlu dibahas per sistem

**Jawaban:** Ganti dengan pemanggilan API ke sistem pemilik data

**Keputusan akhir:** Seluruh akses lintas database lewat DB Link **diganti pemanggilan API** ke sistem pemilik data. Kopling tersembunyi antar database dihapus dan diganti kontrak yang eksplisit.

**Inventaris DB Link yang harus digantikan:**

| DB Link | Pemakaian | Data yang diambil |
|---|---|---|
| `@ASMD` | **55×** | `DATAMINING.GET_WORKING_HOURS` (17×), `HRDASM.V_HRD_MST`, `MST_DET_SALES`, `GENERAL.MST_BUKA_PROTEKSI`, `GL.T_ALL_PAYMENT`, `LST_USER_ASURANSI`, `GENERAL.LST_MITRA`, `TREATY_LOSS`, `MBU.T_CADANGAN_KLAIM_KREDIT`, fungsi `GET_NAMA_*` |
| `@SIMASNET` | 3× | `GENERAL.MST_BUKA_PROTEKSI`, `V_KLAIM` |
| `@SMI` | 2× | `GENERAL.MST_BUKA_PROTEKSI`, `V_KLAIM` |
| `@OPJAVA` | 2× | `NEW_GENERAL.M_USER`, `M_USER_JOB` |
| `@PROD_ASM` | 1× | `POOLDATA.AGENT` |
| `@PROD_TKA` | 1× | `ANEKA.MST_SHARE_PU` |

**Risiko yang harus disadari:** API-API ini **kemungkinan besar belum ada**. Setiap satu memerlukan koordinasi dengan tim pemilik sistem. → **R-03** di Risk Analysis.

**Dampak ke Steering:** Integration Strategy · API Strategy · Risk Analysis · Migration Strategy

---

### D-26 · Model Penugasan (Inbox)
**Status:** DECIDED

**Pertanyaan (Grilling Q7):** Sistem baru harus punya model penugasan sendiri menggantikan tabel inbox Pega. Konsep apa yang dipertahankan?

**Pilihan yang ditawarkan:**
1. Pertahankan konsep Worklist (per orang) dan Workbasket (antrean bersama) *(rekomendasi saya)*
2. Sederhanakan jadi satu model penugasan saja
3. Rancang ulang total sesuai kebutuhan bisnis sekarang
4. Perlu dibahas dengan tim bisnis dulu

**Jawaban:** Pertahankan konsep Worklist (per orang) dan Workbasket (antrean bersama)

**Keputusan akhir:** Model penugasan mempertahankan **dua konsep**:
- **Worklist** — tugas ditugaskan ke satu orang tertentu
- **Workbasket** — antrean bersama, diambil siapa pun yang berwenang

**Dasar dari source (Register_Flow):** pembagian ini nyata dipakai, bukan warisan kosong.
- **Workbasket** (`impl=WorkBasket`, router `ToWorkbasket`): `RCL/PUCL`, `Investigator`, `Compliance`
- **Worklist** (`impl=WorkList`): `Input Register`, `View Polis`, `Estimation`, `Input Estimasi`, `Choose Surveyor`, `Send To Analis`, `Send To PIC Teknik`, `RCLDokter`, `Analyst Doctor`

Mengubahnya akan mengubah cara kerja user, sedangkan D-13 menetapkan alur kerja tetap sama.

**Router yang harus dibuat ulang sebagai aturan routing:** `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter`, `KomiteRouter`, `PNCAdminRouterRCV`, `ToCurrentOperator`, `ToWorkList`, `ToWorkbasket`.
Catatan: `PNCAdminRouter`, `PNCTeknikRouter`, dan `RouterRCLDokter` **tidak ada di export XML** — logikanya harus digali ulang. → **R-04**.

**Dampak ke Steering:** Domain Model · Module Breakdown · Database Strategy

---

### D-27 · Ketersediaan (Availability)
**Status:** DECIDED

**Pertanyaan (Grilling Q8a):** Kapan aplikasi harus bisa dipakai?

**Pilihan yang ditawarkan:**
1. Jam kerja kantor saja (Senin–Jumat, 07.00–18.00 WIB)
2. Jam kerja diperpanjang, termasuk Sabtu atau lembur
3. Harus hidup terus 24/7
4. Belum ada ketentuan resmi

**Jawaban:** Harus hidup terus 24/7

**Keputusan akhir:** Aplikasi harus tersedia **24/7**, termasuk saat deployment.

**Konsekuensi arsitektur yang mengikat:**
- Minimal **dua instance aplikasi** di belakang load balancer, di-update bergantian (rolling deployment).
- Aplikasi wajib **stateless** — tidak boleh ada session tersimpan di memori satu instance.
- **Migrasi skema database harus backward-compatible**: selama rolling deployment, versi lama dan baru berjalan bersamaan terhadap skema yang sama. Kolom dihapus dalam dua tahap, tidak pernah sekali jalan.
- Perlu **health check endpoint** dan graceful shutdown agar request yang sedang berjalan tidak terputus.

**Dampak ke Steering:** Deployment Strategy · Scalability · Technical Strategy · Database Strategy · NFR

---

### D-28 · Audit Trail dan Retensi Data
**Status:** DECIDED (lama retensi: OPEN)

**Pertanyaan (Grilling Q8b):** Apakah ada kewajiban dari OJK atau audit internal soal jejak perubahan data dan lama penyimpanan?

**Pilihan yang ditawarkan:**
1. Ya — ada kewajiban jejak audit dan retensi data
2. Tidak ada kewajiban formal, tapi jejak perubahan tetap diinginkan
3. Belum tahu — perlu tanya tim Compliance
4. Tidak perlu jejak audit khusus

**Jawaban:** Ya — ada kewajiban jejak audit dan retensi data

**Keputusan akhir:** Jejak audit adalah **persyaratan wajib yang dirancang sejak awal**, bukan tambahan.

Setiap perubahan yang bernilai bisnis harus tercatat permanen — **siapa, kapan, nilai sebelum, nilai sesudah** — minimal untuk: nilai estimasi klaim, nilai settlement, akseptasi, keputusan komite, penolakan (RCL), proses ulang (PUCL), perubahan status klaim, dan pembayaran.

Data audit bersifat **append-only**: tidak boleh diubah atau dihapus oleh jalur aplikasi mana pun.

**Terbuka:** lama retensi belum ditentukan; harus dikonfirmasi ke tim Compliance sebelum go-live.

**Dampak ke Steering:** Security · Database Strategy · Domain Model · NFR · Performance Strategy

---

### D-29 · Backup dan Pemulihan
**Status:** DECIDED (angka RPO/RTO: mengikuti standar infra)

**Pertanyaan (Grilling Q8c):** Kalau server database rusak, berapa banyak data yang boleh hilang dan berapa lama sistem boleh mati?

**Pilihan yang ditawarkan:**
1. Ikuti standar backup yang sudah berlaku di data center Sinarmas
2. Tidak boleh ada data hilang sama sekali
3. Boleh hilang maksimal beberapa jam, sistem boleh mati sehari
4. Belum tahu — tolong usulkan angka yang wajar

**Jawaban:** Ikuti standar backup yang sudah berlaku di data center Sinarmas

**Keputusan akhir:** Target RPO dan RTO **mengikuti kebijakan backup korporat** yang sudah berlaku untuk database Oracle saat ini. Aplikasi tidak menetapkan targetnya sendiri.

**Catatan penting:** keputusan ini perlu dibaca bersama D-27. Standar backup korporat menjawab *pemulihan saat bencana*, sedangkan tuntutan 24/7 menjawab *operasional harian*. Keduanya berbeda, dan Steering harus memastikan standar korporat yang ada memang sejalan dengan tuntutan 24/7 — bila tidak, selisihnya harus diangkat ke tim infra.

**Dampak ke Steering:** Deployment Strategy · NFR · Risk Analysis

---

### D-30 · Timeline dan Scope
**Status:** DECIDED oleh manajemen — dicatat dengan risiko terbuka

**Pertanyaan (Grilling Q9):** Kapan project ditargetkan mulai dan selesai?

**Jawaban awal (Other):**
> "dimulai awal September 2026 dan ditargetkan selesai akhir September 2026"

**Keberatan yang diajukan beserta buktinya:** disampaikan bahwa target ~3 minggu tidak realistis, dengan ukuran terukur dari source: 74 harness, 269 section, 15.063 step activity, 652 rule SQL, 86 stored procedure yang source-nya belum ada, 56 report, 12 REST + 6 DB Link, ditambah tabel auth/authz dan penugasan yang belum ada sama sekali, serta tim yang harus belajar Go, React, dan TypeScript sekaligus. Perkiraan yang diajukan: **9–15 bulan dengan tim 5–6 orang**.

**Pertanyaan lanjutan:** Target akhir September itu untuk apa tepatnya?

**Pilihan yang ditawarkan:**
1. Dokumen Steering yang selesai, implementasi menyusul
2. Satu modul percontohan yang jalan di produksi
3. Memang seluruh aplikasi, jadwal sudah ditetapkan manajemen
4. Perlu didiskusikan ulang setelah melihat angka

**Jawaban:** Memang seluruh aplikasi, jadwal sudah ditetapkan manajemen

**Pertanyaan lanjutan:** Bila scope harus dipangkas, mana yang paling mendesak?

**Jawaban (Other):** "semua modul sudah harus pindah"

**Keputusan akhir:** Target **seluruh modul selesai akhir September 2026** diterima sebagai keputusan manajemen dan menjadi dasar penyusunan Migration Strategy. Tidak ada pemangkasan scope.

**Risiko jadwal tetap didokumentasikan** beserta angkanya di **R-05**, bukan untuk menolak target, melainkan agar keputusan pemangkasan atau penambahan sumber daya di kemudian hari punya dasar ukuran yang jelas.

**Dampak ke Steering:** Migration Strategy · Risk Analysis · Testing Strategy

---

## Sesi 3 — 2026-09-08 — Penyusunan ADR dan Ticketing

Sesi ini menurunkan Steering dan BRD yang sudah disetujui menjadi dua artefak yang mengikat
pelaksanaan: ADR (`docs/adr/`) dan tiket pekerjaan (`docs/ticketing/`). Tidak ada kode
implementasi yang ditulis.

Dua kelompok keputusan: **D-31…D-35** dari GATE 0 (audit lingkungan dan rekonsiliasi angka), dan
**D-36…D-40** dari Ronde 1 grilling (Lapis 1 — status keputusan terbuka dan risiko).

Atas permintaan pemilik project, pertanyaan grilling disusun memakai kerangka **5W + 1H**
(Apa · Siapa · Kapan · Di mana · Mengapa · Bagaimana) dan selalu menyertakan pilihan jawaban.

> Catatan penomoran: `docs/interview-history.md` sudah memakai label "Sesi 3" untuk sesi
> Discovery BRD 2026-09-08. Di Decision Log ini, Sesi 3 adalah sesi ADR & Ticketing — secara
> kronologis sesi wawancara keempat. Penomoran `D-nn` tetap berurutan dan tidak bertabrakan.

---

### D-31 · Lokasi dan Format Artefak ADR & Tiket
**Status:** DECIDED

**Pertanyaan:** Di mana ADR dan tiket pekerjaan disimpan, dan format ADR mana yang dipakai? `docs/agents/domain.md:21` menyebut `docs/adr/` dibuat *lazily*; `docs/agents/issue-tracker.md:3` menetapkan `.scratch/` sebagai lokasi tiket, tetapi tiket ini akan ditinjau manajemen, harus ter-commit, dan diimpor ke GitLab.

**Pilihan yang ditawarkan:**
1. ADR di `docs/adr/`, tiket di `docs/ticketing/` — ter-commit dan terlihat di GitLab
2. Tiket di `.scratch/<fitur>/issues/` sesuai `issue-tracker.md` sekarang
3. Format ADR mengikuti template brief tugas (dengan field `Jenis` dan `Modul terdampak`)
4. Format ADR mengikuti `Sample/ADR Ticketing.md` apa adanya

**Jawaban:**
> "a. lokasi adr di docs/adr/
> b. lokasi ticketing di docs/ticketing/
> c. tulis apa adanya sesuai dengan format ADR yang ada di folder Sample"

**Keputusan akhir:** ADR di **`docs/adr/`**, tiket pekerjaan di **`docs/ticketing/`**. Format tiap ADR mengikuti `Sample/ADR Ticketing.md:280-297` apa adanya: `Status` · `Tanggal keputusan`/`Tanggal dokumen` · `Sifat` · `Pemilik keputusan` · `Jejak bukti` · `Terkait`, lalu `Konteks` · `Opsi yang dipertimbangkan` · `Keputusan` · `Rationale` · `Konsekuensi` (Positif / Negatif–utang teknis / Risiko yang diterima sadar) · `Pertanyaan terbuka`.

**Dua penyesuaian yang tidak terhindarkan:** (a) `Sample` menulis `Jejak bukti: decision-log Qnn` karena proyek TEKNO memakai skema `Qnn`; di sini dipakai `D-nn`, `FR-xx`, `R-nn` sesuai skema proyek ini. (b) Format `Sample` tidak punya field `Jenis` (baru / turunan `D-nn` / supersede `D-nn`) maupun `Modul terdampak`; keduanya ditaruh sebagai **kolom di tabel index `docs/adr/README.md`** agar tidak hilang.

**Konsekuensi:** `docs/agents/issue-tracker.md` harus disesuaikan dari `.scratch/` ke `docs/ticketing/` (dikerjakan di Fase 6).

**Dampak ke Steering:** tidak ada — ini keputusan tata letak artefak, bukan arsitektur.

---

### D-32 · Log Penggunaan Skill Dibiarkan Utuh
**Status:** DECIDED

**Pertanyaan:** Dua log skill hidup berdampingan — `docs/skill-usage.md` (sesi BRD 2026-09-08) dan `docs/Steering/18-SKILLS-USAGE-LOG.md` (sesi Steering 2026-09-07, sekaligus Lampiran D dokumen gabungan). Mana yang kanonikal untuk sesi ini?

**Pilihan yang ditawarkan:**
1. `docs/skill-usage.md` menjadi kanonikal; `18-SKILLS-USAGE-LOG.md` dibiarkan sebagai catatan sesi Steering
2. Gabungkan keduanya menjadi satu log
3. Biarkan keduanya utuh apa adanya

**Jawaban:** "e. biarkan utuh"

**Keputusan akhir:** Kedua log **dibiarkan utuh**, tidak digabung dan tidak direstrukturisasi. Keduanya sudah saling merujuk (`skill-usage.md:10` menyebut `18-SKILLS-USAGE-LOG.md` sebagai log sesi sebelumnya), sehingga hubungannya berurutan, bukan bertentangan.

**Terbuka:** di mana entri sesi ADR & Ticketing ini dicatat — ditanyakan pada Fase 6.

**Dampak ke Steering:** tidak ada.

---

### D-33 · Tanpa Version Control di Repository Ini
**Status:** DECIDED

**Pertanyaan:** Repo ini bukan repository VCS apa pun — `.git`, `.hg`, `.svn`, dan `.gitignore` semuanya tidak ada. Selama Fase 2–5 akan ditulis banyak dokumen; tanpa VCS tidak ada cara memperlihatkan diff untuk direview di setiap gate. Apakah `git init` dijalankan sekarang?

**Pilihan yang ditawarkan:**
1. `git init` sekarang, dengan `.gitignore` yang mengecualikan 297 MiB export XML
2. Repo git hanya untuk `docs/`, export XML tetap di luar
3. Tidak perlu VCS sekarang

**Jawaban:** "d. tidak perlu karena nantinya akan disiapkan CI/CD dari team gitlab"

**Keputusan akhir:** **Tidak ada inisialisasi VCS** oleh tim ini. Version control dan CI/CD disiapkan **Tim GitLab** pada tahap berikutnya. Repo dibiarkan apa adanya.

**Konsekuensi yang harus disadari:** (a) tidak ada diff yang dapat ditunjukkan antar gate — review dilakukan atas berkas utuh; (b) tidak ada perlindungan terhadap kehilangan pekerjaan; (c) keputusan mengecualikan export XML dari riwayat git berpindah menjadi tanggung jawab Tim GitLab, dan **harus disampaikan pada mereka sebelum commit pertama** — lihat **D-40**, karena export memuat kredensial plaintext.

**Dampak ke Steering:** Deployment Strategy — bagian pipeline perlu menyebut kepemilikan Tim GitLab.

---

### D-34 · Scope: Area Bengkel/Sparepart/Supplier dan Ruleset GKM Dikecualikan
**Status:** DECIDED

**Pertanyaan:** Audit GATE 0 menemukan **35 rule** yang tidak dimiliki modul mana pun: 31 activity bernama Bengkel/Sparepart (`ChooseCabangBengkelHE`, `UpdateSparepartHE_act`, `ValidasiSparepart`, `ValidationLoginBengkel_act`, dan lainnya) berruleset **GCNMFW** — ruleset Claim PNC sendiri — dan 4 activity master Supplier berruleset **GKM**, ruleset ketiga yang tidak pernah dibahas di D-03 maupun dokumen mana pun. Domain ini tidak ada di `CONTEXT.md`, tidak ada di 14 master, tidak ada di 32 modul, tidak ada di 39 `FR-*`. Padahal D-03 menetapkan bahwa scope adalah seluruh rule yang ada di export XML ini.

**Pilihan yang ditawarkan:**
1. Masuk scope — butuh modul dan `FR-*` baru
2. Keluar scope — butuh entri Decision Log yang membatasi D-03
3. Belum diputuskan — tiket terkait ditandai `needs-info`

**Jawaban:** "f. abaikan saja, tidak masuk dalam scope"

**Keputusan akhir:** Area **Bengkel, Sparepart, dan Supplier dikeluarkan dari scope migrasi**, termasuk seluruh 4 activity berruleset **GKM**. Keputusan ini **membatasi D-03** secara eksplisit: "seluruh rule yang ada di export" tidak mencakup area ini.

**Cakupan yang dikecualikan, terukur dari export:** 35 activity; 8 tabel database (`BENGKEL_HE`, `SPAREPART_HE`, `SPAREPART_HE_VIN_KEY`, `GCNM_M_SPAREPART_CATEGORY`, `GCNM_M_SPAREPART_TYPE`, `NOTIF_RANGKA_HE`, `PANEL_HE`, `LOKASI_PANEL_HE`); 5 harness; 20 section — termasuk alur persetujuan 4 tahap (Approval, lalu Approve atau Reject). Rincian di `docs/verifikasi-bukti-adr.md` §10.4.

**Konsekuensi terbuka:** sebagian activity yang dikecualikan adalah **pemanggil `SendEmailNotification`** yang sudah masuk perhitungan R-07 (`19-GAP-EXPORT-DETAIL.md`). Angka R-07 karena itu memuat pemanggil dari area di luar scope, dan perlu diperiksa saat R-07 direvisi. Juga: **siapa yang memiliki 35 rule ini setelah Pega dimatikan belum ditentukan.**

**Dampak ke Steering:** Module Breakdown · Domain Model · Risk Analysis (R-07) · CONTEXT.md (istilah Bengkel, Sparepart, Supplier sengaja tidak dimasukkan)

---

### D-35 · Angka Modul dan Requirement Mengikuti Bukti, Bukan Dokumen
**Status:** DECIDED

**Pertanyaan:** Rekonsiliasi GATE 0 menemukan angka yang bertentangan antar dokumen. Yang terpenting: `docs/BRD.md:71`, `:1290`, `:1478` dan `docs/understanding-log.md:69` menyebut **27 modul**, padahal enumerasi di kalimat yang sama menghasilkan **32** (F-1 sampai F-5 = 5, B-1 sampai B-14 = 14, S-1 sampai S-7 = 7, U-1 sampai U-6 = 6). `BRD §21.3` kriteria pertama — gerbang cutover tingkat sistem — memakai angka 27 itu. Selain itu `FR-S8` (Perkakas Uji Kesetaraan) **tidak punya baris modul** di `06-MODULE-BREAKDOWN.md`, dan tidak ada modul `S-8` di berkas mana pun. Angka mana yang dipakai untuk menulis ADR dan tiket?

**Pilihan yang ditawarkan:**
1. Pakai angka dokumen (27) supaya konsisten dengan BRD yang sudah disetujui
2. Pakai angka hasil verifikasi bukti, dan ajukan koreksi BRD sebagai usulan revisi
3. Tunda sampai BRD direvisi

**Jawaban:** "g. buat apa adanya sesuai dengan bukti yg kamu temukan, jangan ngasal"

**Keputusan akhir:** **Seluruh angka di ADR dan tiket mengikuti hasil verifikasi bukti**, bukan angka dokumen, dan setiap selisih dicatat sebagai usulan revisi yang menunggu persetujuan pemilik project. Berlaku untuk seluruh 40 kontradiksi di `docs/verifikasi-bukti-adr.md` §11 — di antaranya: **32 modul** (bukan 27); `FR-S8` tanpa modul pemilik; hardcode **66 email, 24 user ID, 8 ambang** (bukan 10, 4, 3); **26** inbox berbasis peran (bukan sekitar 20); **177** objek tabel (bukan 245); grid median **6 kolom** (bukan 18 sampai 27); dan `OFFSET 500000` yang **nol kemunculan** di export.

**Konsekuensi:** ADR dan tiket akan berisi angka yang berbeda dari BRD di beberapa tempat. Tiap perbedaan wajib menyebut kedua rujukan berdampingan agar tidak terbaca sebagai kelalaian.

**Dampak ke Steering:** Module Breakdown · Domain Model · NFR & Performance · Risk Analysis · dan BRD Bab 9, 20, 21, 23 — seluruhnya sebagai **usulan revisi**, bukan perubahan yang sudah dieksekusi.

---

### D-36 · Tanggal Komitmen dan Jalur Eskalasi ke Pihak Luar
**Status:** DECIDED

**Pertanyaan:** `I-05` (`docs/interview-history.md:202`) sudah menetapkan seluruh permintaan artefak **sudah dikirim tetapi belum satu pun diterima**, dan mencatat sendiri bahwa **tanggal komitmen dan jalur eskalasi belum ada** — sehingga "sudah diminta" dan "belum diminta" punya dampak jadwal identik. Bagaimana mekanisme yang mengubah "sudah diminta" menjadi tanggal yang bisa dijadwalkan? Pihaknya kini **enam**, bukan lima: DBA, Tim Pega, Tim pemilik sistem, Compliance, Tim Infra, dan **pemilik API HCC/HCQ** — yang terakhir baru muncul dari Fase 1 dan belum pernah dihubungi.

**Pilihan yang ditawarkan:**
1. Work owner meminta tanggal komitmen tertulis dari keenam pihak minggu ini, dan eskalasi ke Sponsor Project bila lewat
2. Sudah ada tanggal komitmen — akan disebutkan per pihak
3. Tidak ada mekanisme tanggal komitmen; jalan dengan apa yang ada
4. Eskalasi bukan wewenang work owner — harus lewat Sponsor Project sejak awal

**Jawaban:** Opsi 1

**Keputusan akhir:** Pemilik project meminta **tanggal komitmen tertulis** dari keenam pihak luar dalam minggu ini (mulai 2026-09-08), dengan **eskalasi ke Sponsor Project** bila tanggal itu terlewat.

**Konsekuensi untuk ADR dan tiket:** sampai tanggal komitmen diterima, seluruh ADR yang bergantung artefak pihak luar berstatus **`Proposed`** (bukan `Accepted`), dan tiket terkait berstatus **`needs-info`** dengan pemilik yang disebut eksplisit. Status ini diperbarui begitu tanggal komitmen masuk.

**Terbuka:** pemilik API HCC/HCQ belum pernah dihubungi — ini pihak luar ketujuh dalam daftar tindakan hari pertama yang sekarang berisi sepuluh butir.

**Dampak ke Steering:** Risk Analysis, bagian Ringkasan tindakan hari pertama · Migration Strategy Tahap 0

---

### D-37 · Kebijakan Modul yang Artefaknya Tidak Pernah Datang
**Status:** DECIDED

**Pertanyaan:** `BRD §21.4` menyatakan **FR-B5, FR-B7, FR-B9, FR-B10, FR-B12 tidak boleh dinyatakan diterima** sampai R-01 tertutup, dan `BRD:1210` menegaskan itu keputusan manajemen, bukan kompromi diam-diam oleh tim. Asumsi `A-2` (`BRD:1059`) mencatat bila source 64 procedure tidak pernah ada, R-01 berubah dari penundaan menjadi **kebuntuan**. Tiket kelima modul itu ditulis sekarang dengan status terhalang, atau ditunda seluruhnya?

**Pilihan yang ditawarkan:**
1. Tiket ditulis sekarang dengan `Kesiapan: terhalang R-nn`, tidak boleh `ready-for-agent`; ukuran penerimaan pengganti diputuskan manajemen tertulis bila artefak tidak datang sampai tanggal tertentu
2. Tiket kelima modul ditunda seluruhnya — tidak ditulis sampai artefak ada
3. Jalan dengan menyimpulkan perilaku dari perbandingan masukan-keluaran di staging, tanpa jaminan kelengkapan kasus tepi
4. Belum dapat diputuskan — harus dibawa ke manajemen dulu

**Jawaban:** Opsi 1

**Keputusan akhir:** Tiket untuk modul yang artefaknya belum ada **ditulis sekarang** dengan penanda `Kesiapan: terhalang <R-nn>`, dan **tidak boleh** berstatus `ready-for-agent`. Bila artefak tidak datang sampai tanggal yang ditetapkan, **ukuran penerimaan pengganti diputuskan manajemen secara eksplisit dan tertulis** — bukan diputuskan tim pengembang.

**Alasan menulis sekarang alih-alih menunda:** menulis tiketnya tidak berbiaya, dan justru membuat besarnya pekerjaan yang tertahan terlihat oleh manajemen. Menundanya menyembunyikan masalah.

**Dampak ke Steering:** Testing Strategy · Migration Strategy · Risk Analysis (R-01, R-14)

---

### D-38 · Empat Risiko Baru — R-16 sampai R-19
**Status:** DECIDED

**Pertanyaan:** Fase 1 memunculkan empat hal yang tidak tercakup risiko mana pun yang sudah ada. Penomoran risiko sekarang: `docs/Steering/16-RISK-ANALYSIS.md` memuat R-01 sampai R-12, sementara `docs/BRD.md` §20 dan `docs/migration-readiness.md` sudah menambah R-13, R-14, R-15 — yang **belum diserap Steering**. Apakah keempat kandidat diterima sebagai risiko bernomor, dan siapa pemiliknya?

**Pilihan yang ditawarkan:**
1. Terima keempatnya sebagai R-16, R-17, R-18, R-19, meneruskan penomoran BRD
2. Terima sebagian saja
3. Jangan buat risiko baru; masukkan sebagai catatan di bawah R-01 dan R-07
4. Tunda sampai Tim Pega menjawab soal export ulang

**Jawaban:** Opsi 1

**Keputusan akhir:** Empat risiko baru diterima, meneruskan penomoran dari R-15:

| ID | Isi | Pemilik |
|---|---|---|
| **R-16** | **Export Pega tidak lengkap** — sekitar 242 rule dirujuk tetapi tidak ada di export, terberat **116 When rule buatan sendiri**. Tujuh tipe rule tidak diaudit `19-GAP` sama sekali: When, Flow Action, Section, Data Transform, Ticket, Report Definition, Function library | Tim Pega |
| **R-17** | **Kredensial aktif berada di dalam export** — 3 password SMTP di 31 lokasi ditambah kredensial OAuth, semuanya plaintext; ditambah `UseSSL=false` pada seluruh kemunculannya | Tim Infra / Security |
| **R-18** | **Dua integrasi BRI menunjuk host sandbox** di ruleset produksi | Work Owner dan tim integrasi |
| **R-19** | **Cacat aturan uang direplikasi bila P-5 dipatuhi buta** — toleransi spreading berupa pencocokan substring, ambang Rp 50 juta dengan tiga operator berbeda, penyesuaian 7 jam asimetris di dalam satu kondisi validasi | Work Owner |

**Alasan tidak digabung ke risiko yang ada:** R-01 mencakup objek database dan R-07 mencakup activity; **116 When rule adalah kategori ketiga** yang tidak tercakup keduanya, dan dampaknya lebih luas — memblokir percabangan bisnis di hampir semua modul, bukan lima modul. R-17 bukan risiko migrasi sama sekali: ia hidup sekarang bahkan bila migrasi dibatalkan. Menggabungkan keempatnya akan menyembunyikan pemilik yang berbeda.

**Dampak ke Steering:** Risk Analysis — perlu menyerap R-13, R-14, R-15 dari BRD **dan** menambah R-16 sampai R-19; total menjadi **19 risiko**, bukan 12. Diajukan sebagai usulan revisi.

---

### D-39 · Permintaan Export Ulang Pega Berbasis Product Rule
**Status:** DECIDED

**Pertanyaan:** Permintaan yang tercatat sekarang adalah mengirim 40 activity dan 3 router. Fase 1 membuktikan gapnya sekitar 242 rule, dan daftar itu sendiri hanya **batas bawah** — 65 step `Apply-DataTransform` punya `pyParamArray` kosong sehingga nama Data Transform yang dipanggil tidak selalu terbaca. Ditambah dugaan cacat proses export: `Activity/SendEmailNotification-Act.xml` berisi rule `CompressImage_Act` dan `Activity/NotificationPAYDI-Act.xml` berisi `JobSendEmailNotificationPAYDI`. Dan tidak ada direktori sama sekali untuk Access Group, Role, Privilege, Operator, Decision Table, Agent, maupun Job Scheduler — padahal `FR-F3` bertumpu pada 22 access group dan R-02 pada job terjadwal. Bagaimana bentuk permintaan yang benar?

**Pilihan yang ditawarkan:**
1. Ubah permintaan menjadi export ulang berbasis Product rule dengan dependensi, dan minta konfirmasi cacat export
2. Minta keduanya: export ulang **dan** daftar 242 rule sebagai jaring pengaman
3. Tetap minta daftar rule saja
4. Tim Pega tidak akan menuruti; jalan dengan export yang ada dan gali aturan bisnis dari wawancara

**Jawaban:** Opsi 2

**Keputusan akhir:** Permintaan ke Tim Pega berisi **dua bagian sekaligus**: (a) **export ulang lengkap** memakai *Product rule* atau application-based export dengan opsi *include dependent rules*, dan (b) **daftar sekitar 242 rule** yang dirujuk tetapi tidak ada, sebagai **alat verifikasi** bahwa export ulangnya benar-benar lengkap. Sekalian diminta konfirmasi apakah dua berkas yang berisi rule lain menandakan **seluruh 902 berkas perlu divalidasi ulang**.

**Alasan meminta keduanya:** export ulang adalah cara yang benar, tetapi bila Tim Pega hanya dapat memberi sebagian, daftar 242 rule menjadi satu-satunya cara membuktikan apa yang masih kurang.

**Konsekuensi metode:** inventaris rule untuk seluruh pekerjaan berikutnya dibangun dari elemen `pyRuleName`, **bukan dari nama berkas** — karena nama berkas terbukti tidak dapat dipercaya.

**Dampak ke Steering:** Risk Analysis (R-16, R-07, R-04) · `19-GAP-EXPORT-DETAIL.md` perlu bagian Cara Meminta ke Tim Pega yang setara dengan bagian untuk DBA

---

### D-40 · Kredensial di Dalam Export — Menunggu Tim Infra/Security
**Status:** OPEN — menunggu Tim Infra / Security

**Pertanyaan:** Export memuat **3 password SMTP unik di 31 lokasi** dan sepasang kredensial OAuth mitra, semuanya plaintext, di berkas yang direncanakan masuk GitLab. Ditambah `UseSSL` bernilai `false` pada **seluruh 14** kemunculan elemen itu sementara 16 dari 31 lokasi memakai port 587. Bagaimana perlakuannya?

**Pilihan yang ditawarkan:**
1. Kredensial dianggap aktif sampai terbukti sebaliknya: minta rotasi, dan export dikecualikan dari repo git lewat `.gitignore` sebelum commit pertama
2. Kredensial sudah mati — tidak perlu tindakan, boleh disebut di ADR dengan `berkas:baris`
3. Perlu dicek dulu ke Tim Infra/Security sebelum memutuskan
4. Jangan tulis temuan ini di dokumen yang akan di-commit sama sekali

**Jawaban:** Opsi 3, disertai permintaan lampiran detail lokasi temuan password SMTP

**Keputusan akhir:** **BELUM DIPUTUSKAN — pertanyaan terbuka.** Keputusan menunggu Tim Infra/Security. Pemilik keputusan: **Tim Infra / Security**, diteruskan oleh Work Owner.

Daftar lengkap lokasi disusun sebagai berkas serah-terima terpisah, **sengaja tidak disimpan di dalam repository** karena ia memuat peta lokasi kredensial. Nilai kredensial **tidak direproduksi** di berkas mana pun — hanya `berkas:baris`, nama elemen, panjang karakter, dan sidik-ringkas untuk membedakan satu nilai dari yang lain.

**Yang diblokir oleh keputusan ini:**
- ADR proses tentang penanganan secrets dan konfigurasi tiga lapis — dapat ditulis, tetapi berstatus `Proposed`.
- Keputusan apakah `docs/verifikasi-bukti-adr.md` §7.6, yang sudah memuat lokasi tanpa nilai, boleh tetap ada saat repo di-commit.
- Instruksi ke Tim GitLab soal `.gitignore` sebelum commit pertama — lihat **D-33**.

**Enam hal yang perlu dijawab Tim Infra/Security:** apakah 3 password SMTP masih aktif · apakah `UseSSL=false` pada port 587 masih berlaku di produksi · apakah kredensial OAuth BRI itu sandbox atau produksi · bagaimana export 297 MiB boleh disimpan dan apakah boleh masuk GitLab · siapa pemilik Authentication Profile `ServiceRekKlaimPNC` dan `SERVICEKLAIMKBRU` · apakah lokasi temuan boleh dicantumkan di dokumen arsitektur yang di-commit.

**Dampak ke Steering:** Security · Cross-Cutting (Configuration) · Deployment Strategy · Risk Analysis (R-17)

---

### D-41 · Cakupan Tiket Penuh
**Status:** DECIDED

**Pertanyaan (Ronde 2 Q6):** Modul mana yang ditiketkan penuh dengan acceptance criteria berangka, dan mana yang cukup stub backlog? Menulis AC berangka untuk 33 modul saat 19 risiko masih terbuka akan menghasilkan tiket yang mengarang.

**Pilihan yang ditawarkan:**
1. Lima modul ditiketkan penuh (`F-1`, `F-2`, `F-5`, `U-2`, `S-5`); `F-3`, `F-4`, `U-1` penuh untuk bagian yang buktinya lengkap dan `needs-info` untuk bagian terhalang; 24 modul sisanya stub backlog
2. Kedelapan modul gelombang 1 ditiketkan penuh apa adanya
3. Hanya lima modul yang benar-benar tidak terhalang; `F-3`, `F-4`, `U-1` seluruhnya stub backlog
4. Seluruh 33 modul ditiketkan penuh

**Jawaban:** Opsi 1, dengan tiga permintaan lampiran — *"untuk F-3 tolong list access group apa saja yang perlu saya lampirkan"* · *"untuk F-4 sudah dilampirkan isi table EMAILKOMITE pada file emailkomite.csv"* · *"untuk U-3 tolong list harness yang belum ada dan beritahu call darimana"*

**Jawaban lanjutan (bersyarat):** *"tetapi Opsi 1 jika masih ada halangan, tapi lampirkan detail kendalanya. Jika sudah tidak ada halangan, Opsi 4."*

**Keputusan akhir:** Syarat diperiksa dan **halangan masih ada di 32 dari 33 modul**, sehingga **Opsi 1 berlaku**. Matriks kendala per modul disusun sebagai `docs/verifikasi-bukti-adr.md` §15, memisahkan **kendala artefak** (dengan nama artefaknya) dari **kendala keputusan** (dengan pertanyaannya).

Rekapitulasi verdict: **PENUH 1 modul** (`U-2`) · **SEBAGIAN 24 modul** · **TERHALANG 8 modul** (`F-3` identitas, `B-5`, `B-7`, `B-11`, `S-3`, `S-4`, `S-7`, `S-8`, `U-6`).

Tiga lampiran yang diminta sudah diserahkan: daftar 22 access group + 5 When rule peran + penugasan operator→grup (F-3) · analisis 21 kolom `POOLDATA.EMAILKOMITE` beserta tangga jenjang PA/TRAVEL/NONMBU/BONDING (F-4) · daftar 7 harness portal beserta baris rujukan di `Navigation/pyCaseWorkerNavigation-Navigation.xml` (U-1/U-3).

**Temuan yang mengubah premis:** **R-01 bukan lagi penghalang utama.** Dari lima modul yang `BRD §21.4` nyatakan tidak dapat diterima sampai R-01 tertutup, `B-9`/`B-10`/`B-12` naik ke SEBAGIAN karena source procedure-nya datang. `B-5` dan `B-7` tetap terhalang, tetapi oleh **isi dua tabel master** (`GCNM_FEE_SCALE`, `m_currencystandard`) dan oleh **keputusan bisnis yang belum diambil** — bukan oleh R-01. Penghalang utama sekarang **R-16** (±397 artefak, terberat 137 When rule) dan keputusan bisnis.

**Dampak ke Steering:** Module Breakdown · Migration Strategy · Risk Analysis (R-01, R-16)

---

### D-42 · Modul S-8 — Perkakas Uji Kesetaraan Masuk Gelombang 1
**Status:** DECIDED

**Pertanyaan (Ronde 2 Q7):** `FR-S8` punya requirement tetapi **tidak punya baris modul** — tidak ada `S-8` di berkas mana pun, dan `06-MODULE-BREAKDOWN.md` tidak menyebut kata "kesetaraan" sekali pun. Beri nomor modul apa, dan masuk gelombang mana?

**Pilihan yang ditawarkan:**
1. Beri nomor `S-8`, masuk gelombang 1 bersama modul fondasi
2. Beri nomor `S-8`, tetap di gelombang 6 bersama laporan
3. Bukan modul terpisah — jadikan bagian dari `F-2`
4. Belum diputuskan

**Jawaban:** Opsi 1

**Keputusan akhir:** `FR-S8` menjadi modul **`S-8` Perkakas Uji Kesetaraan**, masuk **gelombang 1** bersama modul fondasi. Total modul menjadi **33**, bukan 32.

**Alasan:** ia satu-satunya modul yang **memblokir cutover setiap modul lain**. `BRD §21.1` menjadikan uji kesetaraan otomatis sebagai gerbang pertama setiap modul; menaruhnya di gelombang 6 berarti tidak ada satu modul pun dapat lulus gerbang 1 sampai gelombang 6, yang membatalkan gagasan Strangler Fig (D-05).

**Terbuka:** perkakas ini menuntut kemampuan **menembak Pega dari luar** untuk membandingkan hasil. Siapa yang boleh melakukannya, di lingkungan mana, dan apakah boleh membaca produksi read-only — belum ditanyakan (Lapis 4). Karena itu tiket `S-8` berstatus **TERHALANG** meski modulnya di gelombang 1.

**Pengecualian yang harus dicatat:** dua modul **tidak punya baseline Pega** untuk diuji setara — `F-3` (HCC/HCQ nol jejak di export) dan `S-5` (sistem lama tidak punya jejak audit nilai). Untuk keduanya, `BRD §21.2` kriteria #3 tidak dapat diberlakukan dan harus diganti ukuran lain.

**Dampak ke Steering:** Module Breakdown §3 dan §5 · Testing Strategy · BRD Bab 9 dan 21

---

### D-43 · Pembaca Tiket — Manusia Junior
**Status:** DECIDED

**Pertanyaan (Ronde 2 Q8):** Tiket ditulis untuk siapa? Ini menentukan kedalaman acceptance criteria dan seberapa eksplisit constraint teknis ditulis. `D-09` menetapkan tim adalah developer Pega internal yang dilatih ulang.

**Pilihan yang ditawarkan:**
1. Keduanya — badan tiket untuk manusia junior, bagian verifikasi dan constraint cukup eksplisit untuk agent AFK
2. Manusia junior saja
3. Agent AFK saja
4. Manusia junior sekarang; format agent ditambahkan nanti

**Jawaban:** Opsi 2

**Keputusan akhir:** Tiket ditulis **untuk manusia junior** yang sedang belajar Go, sesuai `D-09`. Konsekuensinya: badan tiket menyertakan **konteks dan alasan**, bukan hanya perintah; istilah domain dijelaskan atau dirujuk ke `CONTEXT.md`; dan constraint teknis ditulis preskriptif sesuai tuntutan `D-09` (struktur folder, penamaan, pola baku yang seragam).

**Yang tidak dilakukan:** tiket **tidak** dioptimalkan untuk agent AFK. Bagian `Rencana verifikasi` tetap memuat perintah nyata yang dapat dijalankan, tetapi karena syarat Definition of Ready menuntut verifikasi dapat dilakukan **tanpa pengetahuan pribadi** — bukan karena menyasar agent.

**Dampak ke Steering:** tidak ada — ini keputusan bentuk artefak.

---

### D-44 · Tim Pelaksana — Ditentukan Setelah Tiket Direview
**Status:** OPEN — menunggu review tiket

**Pertanyaan (Ronde 2 Q9):** Berapa developer dan bagaimana komposisinya? `D-09` menetapkan latar belakang tim tetapi **tidak ada satu pun dokumen yang menyebut jumlahnya**. `06-MODULE-BREAKDOWN.md:103-106` mencatat tiga rantai kritis yang tidak dapat diparalelkan, sehingga jumlah orang menentukan bentuk papan tiket.

**Pilihan yang ditawarkan:**
1. Angka disebutkan — berapa backend, berapa frontend, berapa penuh waktu
2. Belum ditetapkan; tiket ditulis netral terhadap jumlah orang
3. Sama dengan tim Pega yang sekarang merawat aplikasi ini
4. Akan ditambah atau dikurangi tergantung hasil review tiket ini

**Jawaban:** Opsi 4

**Keputusan akhir:** Jumlah dan komposisi tim **ditentukan setelah tiket direview** — papan tiket menjadi bahan keputusan sumber daya, bukan sebaliknya.

**Konsekuensi yang mengikat bentuk tiket:** tiket harus ditulis **netral terhadap jumlah orang**, dan `docs/ticketing/README.md` wajib menampilkan **tiga rantai kritis** (`06-MODULE-BREAKDOWN.md:103-106`) beserta dependency antar tiket secara eksplisit — karena itulah yang memungkinkan pembaca menyimpulkan berapa orang dibutuhkan untuk paralelisasi.

**Risiko yang harus disadari:** `06-MODULE-BREAKDOWN.md:76` menyatakan `U-2` adalah investasi terpenting di frontend. Bila tidak ada personel khusus frontend, `U-2` bersaing waktu dengan `F-2` di rantai kritis ketiga (`U-2 → U-3/U-4/U-5`). Ini harus terlihat di papan tiket sebagai risiko urutan.

**Dampak ke Steering:** Migration Strategy · Risk Analysis (R-05, R-11)

---

### D-45 · Fase 1 Dijalankan Ulang terhadap Snapshot Baru
**Status:** DECIDED

**Pertanyaan (Q11):** `docs/verifikasi-bukti-adr.md` sahih untuk snapshot 2.167 rule. Pada 2026-09-09 export bertambah menjadi 2.389 rule (kemudian terhitung 2.634 berkas setelah `InboxAutoClaim/` diekstrak) dan folder `Database/` muncul dengan 63 objek Oracle + 2 master CSV yang belum pernah dibaca. Fase 1 dijalankan ulang atau tidak?

**Pilihan yang ditawarkan:**
1. Jalankan ulang audit ketersediaan dan area yang paling terdampak saja, lalu perbarui dokumen dan lanjut
2. Jalankan ulang seluruh Fase 1 dari nol terhadap snapshot baru
3. Lanjut dengan dokumen yang ada, tandai setiap angka sebagai "per snapshot 2026-09-08"
4. Tunggu sampai Tim Pega menyatakan pengirimannya lengkap

**Jawaban:** Opsi 1

**Keputusan akhir:** Audit ketersediaan dan area paling terdampak dijalankan ulang. Hasilnya `docs/verifikasi-bukti-adr.md` §14 dan §15. Bagian §0–§13 **dibiarkan apa adanya** sebagai catatan keadaan snapshot 2.167, dengan penanda koreksi di tempat yang dirujuk balik.

**Hasil yang paling berdampak:** **R-02 tertutup** (5 job + 1 agent ditemukan di folder `Job Scheduler/` dan `Agents/` yang sebelumnya tidak ada, seluruh activity target-nya ada) · **R-06 tertutup** (`v_sts_claim.csv`, 33 kode `1134`–`1166`, bukan 10 kode `1142`–`1151`) · **D-14 terjawab** (`emailkomite.csv`, 21 kolom) · **R-01 62 dari 64 berkas** tetapi 12 dependensinya belum ikut · **R-16 memburuk** dari 116 menjadi 137 When rule hilang.

**Enam kesalahan saya sendiri terkoreksi** dan tercatat di §14.2 dan §15.1, di antaranya satu klaim yang dicabut seluruhnya (mekanisme penyimpanan dokumen tetap **tiga**, bukan empat) dan tiga kesimpulan arti kode status yang terbukti salah dari sembilan.

**Alasan tidak memilih opsi 4:** `R-09` mencatat sistem sumber masih aktif berubah — export bertambah dua kali selama sesi ini. "Lengkap" mungkin tidak pernah tiba.

**Dampak ke Steering:** Risk Analysis (R-01, R-02, R-06, R-16) · Gap Export Detail

---

### D-46 · Kewenangan Review dan Pernyataan Selesai Tiket
**Status:** DECIDED oleh work owner — dicatat dengan konsekuensi terbuka

**Pertanyaan (Ronde 2 Q10):** Siapa reviewer, dan siapa yang berwenang menyatakan tiket selesai? Untuk gerbang 2 UAT, `I-06` sudah menetapkan user bisnis per peran + Business Owner. Tetapi modul fondasi (`F-1`, `F-2`, `F-5`, `S-5`) **tidak punya peran pemakai** — tidak ada user bisnis yang membuka layar "Akses Data". Format tiket mewajibkan field `Peran penguji gerbang 2`, yang akan kosong tanpa jawaban.

**Pilihan yang ditawarkan:**
1. Lead engineer me-review teknis; Work Owner menyatakan tiket selesai. Untuk modul fondasi, gerbang 2 diganti "diterima Lead engineer + Work Owner"
2. Reviewer dan yang menyatakan selesai adalah orang yang sama
3. Work Owner me-review dan menyatakan selesai untuk semua tiket
4. Belum ada mekanisme review; ditetapkan saat implementasi mulai

**Jawaban:** Menolak keempat pilihan.
> *"Saya tidak ingin opsi 1 sampai 4, yang saya mau : tiketnya langsung kamu review sendiri dan nyatakan selesai, namun kamu harus ketat berarti kamu harus testing hasilnya dari awal sampai akhir, sampai statusnya hijau/aman."*

**Keberatan yang diajukan beserta buktinya.** Disampaikan tiga konsekuensi, masing-masing dengan rujukan:

1. **Tidak ada kode untuk diuji.** `docs/AGENTS.md:9` dan aturan tugas ini menetapkan repo belum punya kode implementasi dan penulisan kode dilarang. "Testing hasilnya sampai hijau" tidak punya objek uji berupa sistem berjalan; yang dapat diuji adalah dokumennya.
2. **`BRD §21.2` kriteria #4** mewajibkan gerbang 2 berupa UAT oleh user bisnis dari peran yang memakai modul, dan `I-06` — jawaban work owner sendiri — sudah menetapkan user bisnis per peran + Business Owner sebagai penanggung jawabnya.
3. **`BRD §21.2` kriteria #9 dan #12** menuntut jejak audit dan kepemilikan tunggal; keduanya kehilangan makna bila pelaksana menyatakan pekerjaannya sendiri selesai.

Ditambah satu bukti dari sesi ini: **sepanjang 2026-09-08 sampai 2026-09-10, sepuluh kesimpulan saya terbukti salah** — tiga arti kode status, satu klaim mekanisme penyimpanan dokumen, satu keberadaan When rule, satu status R-01, satu arah parameter, satu pemetaan job-activity, dan dua positif palsu metode. Yang mengoreksi seluruhnya adalah master data dan source yang datang kemudian, bukan pembacaan ulang yang lebih teliti.

**Penegasan ulang work owner:** pertanyaan diajukan ulang dalam bentuk terpisah (`selesai-ditulis` versus `selesai-dikerjakan`) dan work owner **menegaskan jawaban yang sama**. Keputusan dihormati.

**Keputusan akhir:** **Lead engineer (agen) menjadi reviewer sekaligus yang menyatakan tiket selesai.** Kewajiban yang mengikat sebagai gantinya: setiap tiket dan ADR wajib melewati **suite verifikasi yang benar-benar dijalankan**, dengan hasil dilaporkan **hijau atau merah per butir beserta angkanya** — bukan penilaian kualitatif. Tidak ada artefak dinyatakan selesai selama ada satu butir merah.

Butir yang diuji, seluruhnya dapat dijalankan tanpa kode implementasi:

| # | Butir | Cara uji |
|---|---|---|
| 1 | Setiap `berkas:baris` yang dikutip benar-benar ada dan isinya sesuai | diverifikasi ulang satu per satu terhadap export terbaru |
| 2 | Traceability dua arah: setiap `FR-xx` dalam cakupan punya ≥1 tiket; setiap tiket punya ≥1 `FR-xx` **dan** rule Pega yang digantikan | dihitung, dilaporkan sebagai angka |
| 3 | Setiap tiket ber-`ready-*` lolos seluruh butir Definition of Ready | daftar periksa per tiket |
| 4 | Setiap istilah domain di tiket sudah ada di `CONTEXT.md` | selisih istilah |
| 5 | Nol nilai sensitif di dokumen yang akan di-commit | pemindaian pola kredensial, email, hostname, data nasabah |
| 6 | Setiap `D-nn`, `R-nn`, `FR-xx` yang dirujuk benar-benar ada dan statusnya mutakhir | selisih terhadap Decision Log dan Risk Analysis |
| 7 | Setiap tiket TERHALANG menyebut penghalangnya dengan nama artefak atau `berkas:baris` | pemindaian |
| 8 | Nol tiket mengklaim status penyelesaian pekerjaan | pemindaian |

**Konsekuensi yang diterima secara sadar, dicatat sebagai keputusan work owner:**
- Tidak ada mata manusia di antara tiket yang ditulis agen dan pekerjaan yang dijalankan atasnya.
- Gerbang 2 (`BRD §21.2` #4) tidak diberlakukan pada tahap penulisan tiket; bila ia tetap diberlakukan pada tahap implementasi, itu keputusan terpisah yang belum diambil.
- Field `Peran penguji gerbang 2` pada setiap tiket diisi sesuai peran bisnis yang teridentifikasi di Lapis 5, dan menjadi **rencana** untuk tahap implementasi — bukan pernyataan bahwa gerbang itu sudah dilewati.

**Terbuka:** siapa yang menyatakan **pekerjaan implementasi** selesai — belum diputuskan, dan tidak dapat diputuskan sekarang karena kodenya belum ada.

**Dampak ke Steering:** Testing Strategy · BRD Bab 21 (usulan revisi: pengecualian gerbang untuk tahap penulisan artefak)

---

### D-47 · Model Penjenjangan Komite — Kumulatif, `LIMIT_TOP` sebagai Validasi Master
**Status:** DECIDED

**Pertanyaan (Ronde 3 Q12):** Klaim Rp 75.000.000 NONMBU masuk jenjang komite mana? Query `RDB List/EmailKomiteBerjenjangSimasnet_sql-SQL.xml:98` hanya memfilter `LIMIT_BOTTOM <= nilai` — kolom `LIMIT_TOP` yang ada di master tidak dipakai — dan rentang NONMBU di `POOLDATA.EMAILKOMITE` saat itu tumpang tindih, sehingga Rp 75.000.000 memenuhi dua baris dan Rp 50.000.000 memenuhi empat.

**Pilihan yang ditawarkan (versi pertama):**
1. Rentang tertutup `LIMIT_BOTTOM <= nilai <= LIMIT_TOP`, master NONMBU dirapikan, tie-breaker dibuang
2. Hanya batas bawah, dengan aturan "DEGREE tertinggi yang memenuhi menang"
3. Kumulatif — semua jenjang yang memenuhi harus menyetujui
4. `dbms_random.value` memang disengaja — pertahankan, dan AC `B-7` tidak boleh deterministik

**Jawaban pertama:** Opsi 1, disertai keterangan *"sudah diperbaiki isinya pada emailkomite.csv"*

**Kerangka pertanyaan dikoreksi sebelum jawaban dipakai.** Setelah jawaban diterima, penelusuran lanjutan membuktikan **kerangka pertanyaannya salah**: `LIMIT_BOTTOM <=` bukan cacat, melainkan **mekanisme penjenjangan kumulatif**.

| Bukti | Isi |
|---|---|
| `Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459` | `childPageKomite.KomiteLoop := TempRDBSearchEmailKomite.pxResultCount` |
| `When/IsKomiteLoop-When.xml` | `.AcceptStatus = "1"` **dan** `.KomiteCount <= .KomiteLoop` |

Jumlah jenjang persetujuan **sama dengan jumlah baris yang dikembalikan query**, dan flow memutar `KomiteCount` dari 1 sampai `KomiteLoop`. Jadi `LIMIT_BOTTOM <= nilai` mengembalikan seluruh jenjang sampai tingkat nilai itu — klaim kecil sedikit approver, klaim besar banyak approver. `LIMIT_TOP` tidak dibutuhkan untuk semantik kumulatif.

**Menerapkan Opsi 1 (rentang tertutup) akan mengembalikan tepat satu baris → `KomiteLoop = 1` → hanya satu jenjang persetujuan berapa pun nilai klaim** — menghapus penjenjangan yang menjadi inti D-14 dan `BRD §11.4`.

**Koreksi kedua atas laporan agen:** `dbms_random.value` **tidak** berlaku umum. Dari **17 query** yang membaca `EMAILKOMITE`: 8 memfilter `LIMIT_BOTTOM`, **0 memfilter `LIMIT_TOP`**, 4 memfilter `TYPE_KOMITE`, dan **hanya 2** memakai `dbms_random` — keduanya jalur Simasnet (`EmailKomiteBerjenjangSimasnet_sql`, `GetEmailKomiteSimasnet_Reject`). Jalur utama (`EmailKomiteBerjenjang_sql`, PA, Travel, Adjuster, Salvage) memakai `ORDER BY DEGREE` saja — **deterministik**. `EmailKomiteBerjenjangBonding_sql` tidak memfilter `LIMIT` sama sekali.

**Pilihan yang ditawarkan (versi kedua):**
1. Pertahankan model kumulatif; `LIMIT_TOP` menjadi dokumentasi, tidak dipakai query
2. Kumulatif, **dan** `LIMIT_TOP` ikut divalidasi sebagai pemeriksaan integritas master — menolak master yang rentangnya bolong atau tumpang tindih — tetapi tidak dipakai memilih baris
3. Tetap rentang tertutup — satu jenjang per klaim, penjenjangan dihapus
4. Lain

**Jawaban:** Opsi 2

**Keputusan akhir:** **Model kumulatif dipertahankan.** Jumlah jenjang persetujuan = jumlah baris `POOLDATA.EMAILKOMITE` yang memenuhi `LIMIT_BOTTOM <= nilai klaim` (ditambah filter `STS_ADJ`, `STS_AKTIF`, `TYPE_BUSINESS`, dan `TYPE_KOMITE` pada query yang memakainya), diurutkan `DEGREE`. Perilaku identik dengan Pega, sehingga aman untuk gerbang 1.

**`LIMIT_TOP` dipakai sebagai validasi integritas master data**, bukan untuk memilih baris: sistem baru menolak master yang rentangnya tumpang tindih atau berlubang antar `DEGREE` dalam satu `TYPE_BUSINESS` + `TYPE_KOMITE`.

**Isi master sudah diperbaiki work owner** (`Database/emailkomite.csv`, 2026-09-11). Tangga kumulatif yang dihasilkan, terverifikasi:

| Lini | Batas bawah per jenjang | Jenjang maksimum |
|---|---|---|
| **PA** | `0` · `10.000.001` · `50.000.001` · `100.000.001` | **4** |
| **TRAVEL** | `0` · `50.000.001` · `100.000.001` | **3** |
| **NONMBU** `TYPE_KOMITE=2` | `0` · `100.000.001` · `500.000.001` · `1.000.000.001` | **4** |
| **NONMBU** `TYPE_KOMITE=1` | `50.000.001` | 1 |
| **BONDING** | — query tidak memfilter `LIMIT` | 1 |

Acceptance criteria yang dapat langsung dipakai: **PA Rp 10.000.000 → 1 jenjang; Rp 10.000.001 → 2 jenjang; Rp 50.000.001 → 3 jenjang; Rp 100.000.001 → 4 jenjang.** Ambiguitas Rp 75.000.000 hilang karena perbaikan data, bukan karena perubahan aturan.

**Terbuka — belum dijawab, dibawa ke ronde berikutnya:**
- `TYPE_KOMITE=2` NONMBU tidak punya baris ber-batas-bawah antara `0` dan `100.000.001`, sehingga Rp 50 juta–Rp 100 juta hanya 1 jenjang di jalur itu, sementara `TYPE_KOMITE=1` punya baris `50.000.001`. Benar begitu?
- Di atas **Rp 200.000.000** tidak ada baris untuk PA dan TRAVEL.
- `TYPE_KOMITE` dipilih di `SetEmailKomite` lewat **syarat bernama orang** — dibawa sebagai peran atau dibuang?

**Dampak ke Steering:** Domain Model (master Ambang Komite) · Module Breakdown B-7 · BRD Bab 11.4 · `20-DETAIL-KOMITE-DBLINK.md` §1.2 dan §1.4 (struktur 21 kolom, bukan 14)

---

### D-48 · Basis Kurs dan Perilaku Saat Kurs Tidak Ditemukan
**Status:** DECIDED

**Pertanyaan (Ronde 3 Q13):** `Database/GETCURRENCYSTANDARD.fnc` menerima parameter tanggal `i_tgl_kurs` (`:3`) tetapi **tidak pernah memakainya** — yang dipakai `TRUNC(sysdate)` (`:14`). Pemanggilnya mengirim **tanggal kerugian** (`RDB List/GetDataTrytyInwardFromUploadData-SQL.xml:34`, `:38`; `Activity/BrowseDataDetailKlaimXOLAndReas-Act.xml:3289`, `:3320`, `:3331`). Dan bila kurs tidak ditemukan, fungsi mengembalikan **`1`** (`:20-22`) — valuta asing diperlakukan 1:1 terhadap rupiah, sehingga ambang komite Rp 50 juta/Rp 30 juta dan Notice of Large Losses Rp 1 miliar **tidak terpicu**.

**Pilihan yang ditawarkan — (a) basis kurs:**
1. Kurs pada tanggal kerugian
2. Kurs hari eksekusi
3. Kurs pada tanggal registrasi klaim

**Pilihan yang ditawarkan — (b) bila kurs tidak ditemukan:**
1. Tolak transaksi dengan galat eksplisit
2. Pakai kurs terakhir yang tersedia berapa pun tanggalnya
3. Pertahankan `RETURN 1`

**Jawaban:** (a) Opsi 1 · (b) Opsi 1

**Keputusan akhir:** Konversi ke IDR memakai **kurs yang berlaku pada tanggal kerugian**, bukan kurs hari eksekusi. Bila kurs untuk mata uang dan tanggal itu tidak ditemukan, transaksi **ditolak dengan galat eksplisit** — tidak ada nilai default.

**Konsekuensi yang harus disetujui sebelum gerbang 1 dijalankan:** memperbaiki basis kurs membuat uji kesetaraan **menampilkan selisih pada seluruh data historis valuta asing**, karena laporan lama memakai kurs hari eksekusi. Selisih itu wajib dinyatakan lebih dulu sebagai perbaikan yang direncanakan, bukan ditemukan sebagai kejutan saat pengujian.

**Konsekuensi kedua:** menolak transaksi saat kurs tidak ada akan **menampakkan kegagalan yang selama ini senyap**. Klaim valuta asing yang dulu lolos tanpa komite kini akan ditolak sampai kursnya dilengkapi. Ini perbaikan yang diinginkan, tetapi berdampak operasional dan perlu disiapkan.

**Terbuka:** isi tabel `m_currencystandard` belum ada di repo — masih diminta ke DBA. Tanpa itu `B-5` tetap tidak dapat diuji.

**Dampak ke Steering:** Domain Model (master Mata Uang & Kurs) · BRD Bab 11.4 (status ✅ pada baris konversi kurs perlu diturunkan) · Risk Analysis (R-19)

---

### D-49 · Kebijakan atas Sepuluh Cacat Aturan Uang
**Status:** DECIDED

**Pertanyaan (Ronde 3 Q14):** `P-5` mewajibkan hasil identik Pega **kecuali perbaikan yang diputuskan eksplisit**. Daftar itu berisi empat butir (`requirement-summary.md:26`). Verifikasi bukti menemukan **sepuluh cacat aturan uang**. Mana yang direplikasi, mana yang diperbaiki?

**Pilihan yang ditawarkan:**
1. Setujui seluruh usulan; butir 7 ditanyakan terpisah
2. Setujui sebagian — sebutkan nomor mana yang direplikasi apa adanya
3. Replikasi seluruh sepuluh; perbaikan ditunda sebagai Future Enhancement
4. Bawa ke manajemen dulu karena menyentuh angka historis

**Jawaban:** Opsi 1, dan untuk butir 7: *"sudah benar"*

**Keputusan akhir:** Sembilan cacat **diperbaiki**; satu **direplikasi** karena perilakunya memang benar.

| # | Cacat | Bukti | Keputusan |
|---|---|---|---|
| 1 | Toleransi spreading berupa pencocokan substring; `199.99` dan `1100.0` lolos | `Activity/InputRegister_act-Act.xml:13183` | **PERBAIKI** |
| 2 | Ambang Rp 50 juta memakai tiga operator berbeda di tiga rule | `SetEmailKomite:1445` · `SetEmailKomiteAdjuster:968` · `SetEmailKomiteSalvage:888` | **PERBAIKI** — satu operator |
| 3 | Penyesuaian 7 jam asimetris di dalam satu kondisi validasi | `InputRegister_act:5788` + `:4805` | **PERBAIKI** |
| 4 | Kurs mengabaikan tanggal kerugian | `GETCURRENCYSTANDARD.fnc:3` vs `:14` | **PERBAIKI** — lihat D-48 |
| 5 | Kurs tidak ditemukan → `RETURN 1` | `GETCURRENCYSTANDARD.fnc:22` | **PERBAIKI** — lihat D-48 |
| 6 | `LIMIT_TOP` diabaikan + tie-breaker acak | `EmailKomiteBerjenjangSimasnet_sql:98-99` | **PERBAIKI sebagian** — lihat D-47: model kumulatif dipertahankan, `LIMIT_TOP` jadi validasi master; tie-breaker acak hanya ada di 2 query Simasnet |
| **7** | **Tanggal PLA/DLA dari pengguna dibuang, diganti `SYSDATE`** | `INSERT_PLADLA.prc:67`, `:136`, `:177` | **REPLIKASI — perilaku sekarang sudah benar.** Tanggal PLA/DLA adalah tanggal sistem menerbitkan dokumen, bukan tanggal yang dipilih pengguna |
| 8 | `NilaiSalvage` hanya ditambahkan bila baris terakhir kebetulan bertipe salvage | `ValidasiSisaTSI:817` vs `:1489` | **PERBAIKI** — salvage selalu ditambahkan |
| 9 | `INSERT_SALVAGE` menulis `IDSALVAGE = NULL` saat update | `INSERT_SALVAGE.prc:47` | **PERBAIKI** + perlu hitungan DBA atas baris rusak |
| 10 | `GETSELISIHJAM` gagal → `RETURN 0` jam, tak terbedakan dari nol | `GETSELISIHJAM.fnc:22` | **PERBAIKI** |

**Konsekuensi yang mengikat gerbang 1:** daftar "perbaikan yang diputuskan eksplisit" (`P-5`) bertambah dari **4 menjadi 13** butir. Setiap selisih yang muncul pada uji kesetaraan wajib dapat dipetakan ke salah satu dari 13 butir itu, atau dinyatakan sebagai bug.

Butir 7 menjadi contoh penting untuk tiket `B-9`: parameter `TTGLPLADLA` yang diterima lalu dibuang **bukan cacat** — sistem baru boleh menghapus parameter itu seluruhnya, dan AC-nya menyatakan tanggal dokumen diisi waktu sistem.

**Dampak ke Steering:** Migration Strategy `P-5` · Testing Strategy (gerbang 1) · `requirement-summary.md` §1 · BRD Bab 11 dan 21 · Risk Analysis (R-19)

---

### D-50 · Basis Perhitungan TAT — Dua Fungsi Dipertahankan untuk Keperluan Berbeda
**Status:** DECIDED

**Pertanyaan (Ronde 3 Q15):** `Database/GETSELISIHJAM.fnc` menghitung selisih dengan memotong akhir pekan saja — **nol hari libur, nol jam kerja** — sementara `DATAMINING.GET_WORKING_HOURS@ASMD` adalah objek remote lewat DB Link. Mana basis TAT yang benar?

**Pilihan yang ditawarkan:**
1. `GET_WORKING_HOURS@ASMD` menjadi satu-satunya basis; `GETSELISIHJAM` dipensiunkan
2. `GETSELISIHJAM` menjadi satu-satunya basis
3. Keduanya dipertahankan untuk keperluan berbeda
4. Belum tahu — perlu dicek ke pemilik KPI

**Jawaban:** Opsi 3, disertai permintaan daftar pemakaian kedua fungsi untuk cross-check

**Daftar pemakaian yang diserahkan** (pencarian case-insensitive; pemakaiannya huruf kecil):

| Fungsi | Pemakaian | Berkas |
|---|---|---|
| **`GET_WORKING_HOURS`** | **18** di **7 berkas** | `RDB List/GetDataKPIAdmin-SQL.xml` (4) · `GetDataKPIAdminPA-SQL.xml` (4) · `GetDataKPIAdminPA_khususPA-SQL.xml` (4) · `BrowseDataKPIAdmin_PA-SQL.xml` (2) · `ExportDataKomitesKlaimNONMBU-SQL.xml` (2) · `BrowseDataKPIAdmin-SQL.xml` (1) · **`Database/INSERT_PNCCHRONOLOGYTAT.prc` (1)** |
| **`GETSELISIHJAM`** | **1 jalur** | `RDB List/GetSelisihJam_sql-SQL.xml` — hasilnya **dibagi 24**, jadi satuannya **hari**; satu-satunya pemanggil `Activity/GetInboxRegisterCompliance-Act.xml` |
| **`HRD_LBR`** | **1** | `RDB List/CheckHoliday_SQL-SQL.xml` |

**Keputusan akhir:** Kedua fungsi dipertahankan dengan batas pemakaian yang sudah terbukti:

- **`GET_WORKING_HOURS`** → seluruh **KPI dan laporan admin**, ditambah **penulisan kronologi TAT**. Ini jalur yang angkanya dilaporkan ke manajemen.
- **`GETSELISIHJAM`** → **satu layar saja**, inbox Compliance, dengan keluaran **hari** (dibagi 24), bukan jam kerja.
- **`HRD_LBR`** → pemeriksaan hari libur, dipakai terpisah.

**Temuan yang mendukung keputusan ini:** framing awal agen ("dua basis yang tidak akan pernah menghasilkan angka sama") benar secara aritmetika tetapi **menyesatkan** — keduanya tidak pernah dipakai mengukur hal yang sama. Batas pemakaiannya tidak tumpang tindih.

**Konsekuensi:** `GET_WORKING_HOURS` dan `HRD_LBR` adalah objek **remote lewat DB Link**, sehingga keduanya masuk lingkup **D-25/R-03** dan logikanya harus **ditulis ulang di Go**, bukan sekadar dipanggil lewat API — karena perhitungan jam kerja dan kalender libur adalah aturan bisnis, bukan pengambilan data.

**Tetap diperbaiki apa pun pilihannya:** `GETSELISIHJAM.fnc:22` mengembalikan `0` saat error, tak terbedakan dari nol hari — butir 10 pada D-49.

**Dampak ke Steering:** Module Breakdown S-7 dan B-8 · Integration Strategy (D-25) · Risk Analysis (R-03, R-19)

---

### D-51 · Pembulatan Uang dan Toleransi Total Spreading
**Status:** DECIDED

**Pertanyaan (Ronde 3 Q16):** Sistem lama **tidak punya kebijakan pembulatan di mana pun** — nol `ROUND`/`TRUNC`/`CEIL`/`FLOOR`/`setScale` pada nilai uang, baik di lapisan Pega (`InputRegister_act`, 1,1 MiB) maupun di 12 procedure Oracle yang dibaca. `Database/GET_INTERPOLASIPNC.fnc` melakukan dua pembagian (`:25`, `:37`) tanpa pembulatan, dan `ehasil` dideklarasi `number` polos tanpa presisi. Berapa desimal untuk nilai uang, berapa untuk share spreading, dan kapan dibulatkan?

**Pilihan yang ditawarkan — (a) nilai uang klaim:**
1. Bulat ke rupiah · 2. Dua desimal · 3. Potong ke bawah · 4. Simpan penuh, bulatkan hanya saat tampil

**Pilihan yang ditawarkan — (b) share spreading:**
1. Dua desimal, toleransi `>= 99,99 && <= 100,01` · 2. Empat desimal · 3. Bilangan bulat persen

**Pilihan yang ditawarkan — (c) kapan dibulatkan:**
1. Setiap langkah · 2. Hanya pada nilai akhir sebelum disimpan · 3. Hanya saat ditampilkan

**Jawaban:**
> *"Kalau yang dimaksud untuk hal yang sama terkait total spreading 100% maka (a) Opsi 4 (simpan penuh, bulatkan hanya saat tampil) dengan validasi SUM(share) harus ada toleransi 99,9999 atau 100,0001 dengan pembulatan 4 desimal"*

**Keputusan akhir:**

**Nilai uang** disimpan **presisi penuh**; pembulatan dilakukan **hanya saat ditampilkan**, tidak saat menyimpan dan tidak per langkah perhitungan. Perbandingan terhadap ambang memakai nilai presisi penuh.

**Total spreading reasuransi** divalidasi dengan **pembulatan 4 desimal** dan **toleransi `99,9999` sampai `100,0001`**:

```
ROUND(SUM(share), 4) BETWEEN 99.9999 AND 100.0001
```

Ini menggantikan pencocokan substring `@contains(local.totalspreading, 99.99)` di
`Activity/InputRegister_act-Act.xml:13183` — butir 1 pada D-49.

**Acceptance criteria yang dapat langsung dipakai untuk `B-4`:**

| Total share | Hasil | Alasan |
|---|---|---|
| `100.0000` | **diterima** | tepat |
| `99.9999` | **diterima** | batas bawah toleransi |
| `100.0001` | **diterima** | batas atas toleransi |
| `99.9998` | **ditolak** | di luar toleransi |
| `100.0002` | **ditolak** | di luar toleransi |
| `99.99` | **ditolak** | perilaku lama menerimanya — **selisih yang direncanakan**, butir 1 D-49 |
| `199.99` | **ditolak** | perilaku lama menerimanya — selisih yang direncanakan |
| `1100.0` | **ditolak** | perilaku lama menerimanya — selisih yang direncanakan |

Tiga baris terakhir adalah **selisih yang akan muncul pada gerbang 1** dan sudah disetujui lebih dulu lewat D-49 butir 1.

**Konsekuensi untuk fee adjuster:** `GET_INTERPOLASIPNC` menghasilkan pecahan rupiah tak terbatas dari dua pembagian. Dengan keputusan ini, hasilnya **disimpan penuh** dan dibulatkan hanya saat ditampilkan — sehingga akumulasi lintas adjustment tidak kehilangan presisi.

**Terbuka:** apakah `LIMIT_BOTTOM` dan ambang uang lain di master bertipe numerik atau teks — DDL belum ada (R-08). Bila bertipe `VARCHAR2`, perbandingan menjadi leksikografis dan seluruh AC di atas tidak berlaku.

**Dampak ke Steering:** Domain Model · Database Strategy (tipe kolom nilai uang) · BRD Bab 11.3 · Testing Strategy (gerbang 1)

---

### D-52 · `TYPE_KOMITE` adalah Pemilih Pita Nilai — Melengkapi D-47
**Status:** DECIDED

**Pertanyaan (Ronde 3 Q17):** Setelah `Database/emailkomite.csv` dikoreksi kedua kalinya (2026-09-11 19:06, satu karakter: baris `ID 7` `TYPE_KOMITE` dari `2` menjadi `1`), gabungan kedua sub-tangga menjadi bersambung tanpa celah. Tetapi query `RDB List/EmailKomiteBerjenjang_sql-SQL.xml` memfilter `trim(TYPE_KOMITE) = trim({tempAdj.AcceptedNo})`, sehingga satu klaim hanya melihat satu sub-tangga. Akibatnya `TYPE_KOMITE=2` tidak lagi punya baris di bawah Rp 100.000.001. Bagaimana `TYPE_KOMITE` dipilih, dan apakah pemisahan ini disengaja?

**Pilihan yang ditawarkan:**
1. Gabungkan menjadi satu tangga — `TYPE_KOMITE` tidak lagi memfilter; kelima baris jadi satu deret kumulatif 1–5 jenjang
2. Pertahankan dua tangga, `TYPE_KOMITE` ditentukan aturan bisnis berbasis data (lini bisnis, cabang, jenis klaim)
3. Pertahankan dua tangga dengan pemilihan berbasis peran pengganti ketiga nama orang
4. `TYPE_KOMITE=2` memang seharusnya tidak menangani klaim di bawah Rp 100.000.001 — celah itu disengaja

**Jawaban:** Opsi 4 — *"karena untuk komite sampai 100 Jt pakai type_komite=1"*

**Keputusan akhir:** **`TYPE_KOMITE` adalah pemilih pita nilai klaim**, bukan penanda orang dan bukan penanda organisasi:

| Nilai klaim (setelah konversi IDR per D-48) | `TYPE_KOMITE` |
|---|---|
| Rp 0 sampai **Rp 100.000.000** | **`1`** |
| **di atas Rp 100.000.000** | **`2`** |

Celah pada `TYPE_KOMITE=2` di bawah Rp 100.000.001 **disengaja** dan bukan cacat — pita itu ditangani `TYPE_KOMITE=1`.

**Konsekuensi yang menutup satu butir `D-15`:** karena `TYPE_KOMITE` kini diturunkan dari **nilai klaim**, syarat bernama orang di `Activity/SetEmailKomite-Act.xml` step 10, 12, dan 14 (`ELLENSUPRIYATI` → `1`, `INDRAGUNAWAN` → `2`, `YOHANESRAYMONDADIKARTA` → `NONMBUAB`/`0`) **tidak dibawa ke sistem baru**. Begitu pula pola "nilai pembanding sengaja dinaikkan melewati ambang" yang menyertainya (`20-DETAIL-KOMITE-DBLINK.md:110-120`). Keduanya digantikan satu aturan berbasis nilai.

Ini menjadikan **tiga dari 24 user ID hardcode** hilang dari jalur komite — bagian dari `FR-F4`.

**Acceptance criteria final untuk B-7 — jumlah jenjang persetujuan NONMBU**

Model kumulatif per D-47: jumlah jenjang = jumlah baris `EMAILKOMITE` yang memenuhi
`STS_ADJ='1' AND STS_AKTIF='1' AND TYPE_BUSINESS='NONMBU' AND TYPE_KOMITE=<pita> AND LIMIT_BOTTOM <= nilai`, diurutkan `DEGREE`.

| Nilai klaim | `TYPE_KOMITE` | Baris yang cocok | **Jenjang** |
|---|---|---|---|
| Rp 50.000.000 | 1 | ID 7 | **1** |
| Rp 50.000.001 | 1 | ID 7, ID 1 | **2** |
| Rp 100.000.000 | 1 | ID 7, ID 1 | **2** |
| Rp 100.000.001 | 2 | ID 2 | **1** |
| Rp 500.000.001 | 2 | ID 2, ID 3 | **2** |
| Rp 1.000.000.001 | 2 | ID 2, ID 3, ID 4 | **3** |

**Diskontinuitas yang diterima secara sadar:** menyeberangi Rp 100.000.000 **menurunkan** jumlah approver dari 2 menjadi 1, karena komite berganti badan — dari komite pita rendah (2 anggota) ke komite pita tinggi (3 anggota) yang mulai dari jenjang pertamanya sendiri. Ini konsekuensi langsung dari memperlakukan keduanya sebagai **dua komite terpisah**, bukan satu tangga bersambung. Dicatat agar tidak dibaca sebagai cacat saat uji kesetaraan.

**Catatan penomoran `DEGREE`:** nilai `DEGREE` di master (1,1,2,3,4) **tidak sama dengan** jumlah jenjang. `DEGREE` hanya menentukan urutan (`ORDER BY DEGREE`); jumlah approver ditentukan `pxResultCount`. Pernyataan "berjenjang sampai 4 level" di D-14 dan `BRD §11.4` merujuk nilai `DEGREE`, bukan jumlah approver — jumlah approver maksimum yang benar-benar dapat terjadi adalah **2** (pita rendah) dan **3** (pita tinggi).

**Masih terbuka dari D-47:** di atas **Rp 200.000.000** tidak ada baris untuk **PA** dan **TRAVEL** (maksimum `LIMIT_TOP` keduanya Rp 200.000.000). Klaim PA atau Travel di atas nilai itu tetap mendapat 4 dan 3 jenjang karena model kumulatif, tetapi tidak ada baris yang secara eksplisit mencakupnya — validasi `LIMIT_TOP` per D-47 Opsi 2 akan menandainya sebagai lubang master.

**Dampak ke Steering:** Domain Model (master Ambang Komite) · Module Breakdown B-7 · BRD Bab 11.4 · `20-DETAIL-KOMITE-DBLINK.md` §1.4 dan §1.5 (logika bernama orang dicabut) · `FR-F4` (3 user ID hardcode hilang)

---

### D-53 · Data dan Lingkungan Uji Kesetaraan
**Status:** DECIDED

**Pertanyaan (Ronde 4 Q18):** `BRD §21.1` gerbang 1 menuntut perbandingan atas "data historis yang sama". Data apa, di lingkungan mana, dijalankan siapa? Tanpa ini `S-8` tidak punya acceptance criteria, dan ia gelombang 1 yang memblokir cutover setiap modul.

**Pilihan yang ditawarkan:**
1. Salinan data produksi di staging; Pega staging dan Go staging; dijalankan tim pengembang
2. Produksi read-only untuk Pega + Go staging
3. Data uji buatan yang mencakup kasus tepi yang sudah teridentifikasi
4. Belum dapat diputuskan — perlu tim infra

**Jawaban:** Opsi 1

**Keputusan akhir:** Uji kesetaraan dijalankan atas **salinan data produksi di lingkungan staging**, dengan **Pega staging** dan **Go staging** sebagai kedua sisi pembanding, dan dijalankan oleh **tim pengembang**.

**Alasan yang mengikat:** aturan kerja project melarang akses produksi, dan `D-21` menetapkan satu database bersama selama masa paralel — menembak produksi membawa risiko menulis tanpa sengaja. Data uji buatan (opsi 3) tidak memadai: cacat yang ditemukan pada Fase 1 — toleransi spreading berupa pencocokan substring, kurs `RETURN 1`, `IDSALVAGE = NULL` — justru muncul dari data nyata yang tidak akan terpikir dibuat.

**Konsekuensi yang harus disiapkan:**
- Dibutuhkan **Pega staging yang dapat ditembak dari luar** — belum ada konfirmasi bahwa lingkungan itu tersedia. Ini dependensi ke Tim Pega dan Tim Infra.
- Salinan data produksi membawa **data nasabah nyata** ke staging. Perlakuan penyamaran, hak akses, dan retensi salinan itu masuk lingkup `D-40` dan Compliance.
- `R-12` (zona waktu bergeser saat migrasi data) berlaku pada proses penyalinan itu sendiri, bukan hanya pada migrasi akhir.

**Dampak ke Steering:** Testing Strategy · Deployment Strategy (lingkungan) · Risk Analysis (R-12) · Module Breakdown S-8

---

### D-54 · Kewenangan Menyetujui Selisih pada Gerbang 1
**Status:** DECIDED

**Pertanyaan (Ronde 4 Q19):** Daftar perbaikan eksplisit `P-5` kini berisi **13 butir** (D-49). Setiap selisih yang muncul pada uji kesetaraan wajib dipetakan ke salah satunya atau dinyatakan bug. Siapa yang memutuskan?

**Pilihan yang ditawarkan:**
1. Selisih yang cocok dengan 13 butir `P-5` lolos otomatis; selisih di luar itu wajib persetujuan Work Owner tertulis
2. Setiap selisih tanpa kecuali butuh persetujuan Work Owner
3. Lead engineer memutuskan; Work Owner diberi tahu
4. Belum ditetapkan

**Jawaban:** Opsi 1

**Keputusan akhir:** Selisih yang **cocok dengan salah satu dari 13 butir `P-5`** lolos otomatis dan cukup dicatat. Selisih **di luar 13 butir itu** wajib **persetujuan Work Owner secara tertulis** sebelum modul dinyatakan lulus gerbang 1.

**Alasan:** ketiga belas butir sudah diputuskan eksplisit pada D-49, sehingga meminta persetujuan ulang per modul menambah beban tanpa menambah kendali. Yang benar-benar menuntut perhatian adalah selisih **yang tidak terduga** — dan itulah yang dipagari.

**Konsekuensi untuk perkakas `S-8`:** ia wajib **mengklasifikasikan** setiap selisih, bukan hanya melaporkannya. Keluarannya minimal: selisih yang terpetakan ke butir `P-5` mana, dan selisih yang tidak terpetakan. Yang kedua menjadi antrean persetujuan Work Owner.

**Dampak ke Steering:** Testing Strategy · Migration Strategy `P-5` · Module Breakdown S-8

---

### D-55 · Pencabutan `BRD §21.4` untuk Empat Modul
**Status:** DECIDED — usulan revisi BRD, menunggu persetujuan formal

**Pertanyaan (Ronde 4 Q20):** `BRD §21.4` menyatakan `FR-B5`, `B7`, `B9`, `B10`, `B12` tidak dapat dinyatakan diterima sampai R-01 tertutup. Setelah 62 dari 64 source procedure datang, status kelimanya berubah dan penghalangnya kini berbeda per modul. Apakah §21.4 masih berlaku untuk kelimanya?

**Pilihan yang ditawarkan:**
1. Cabut §21.4 untuk `B-7`, `B-9`, `B-10`, `B-12`; pertahankan hanya untuk `B-5`
2. Pertahankan §21.4 apa adanya sampai seluruh artefak lengkap
3. Cabut seluruhnya — R-01 sudah tidak relevan
4. Bawa ke manajemen karena §21.4 sudah disetujui

**Jawaban:** Opsi 1

**Keputusan akhir:** `BRD §21.4` **dicabut untuk `B-7`, `B-9`, `B-10`, dan `B-12`**, dan **dipertahankan hanya untuk `B-5`**.

Status dan sisa penghalang per modul:

| Modul | Status | Sisa penghalang |
|---|---|---|
| **B-7** Komite | lepas dari §21.4 | hanya sisa: PA dan TRAVEL di atas Rp 200.000.000 tidak punya baris master |
| **B-9** PLA/DLA | lepas dari §21.4 | status `VALID`/`INVALID` `GET_GROUPBUSINESS_XOL` di produksi; keputusan menulis ulang procedure ber-9-commit |
| **B-10** Akseptasi | lepas dari §21.4 | 3 activity hilang: `InsertDataAkseptasiToLeader`, `InsertLogKasir_act`, `TransferCashierDataASM_act` |
| **B-12** Salvage | lepas dari §21.4 | `SET_ATTACHFILETEMPSALVAGE`; hitungan DBA atas baris ber-`IDSALVAGE` NULL |
| **B-5** Settlement | **tetap terikat §21.4** | isi `POOLDATA.GCNM_FEE_SCALE` (17 pita) · isi `m_currencystandard` · `BrowseT_Claim_Adjustment_SQL` |

**Alasan:** §21.4 ditulis ketika R-01 memblokir kelimanya secara seragam. Penghalangnya sekarang berbeda per modul dan sebagian sudah lepas; menahan empat modul atas dasar yang tidak lagi berlaku menunda pekerjaan yang sebenarnya dapat berjalan.

**Ini usulan revisi BRD, bukan perubahan yang dieksekusi agen.** `docs/BRD.md` tidak disentuh; revisinya masuk daftar usulan Fase 6.

**Dampak ke Steering:** BRD Bab 21.4 · Risk Analysis (R-01, R-14) · Module Breakdown

---

### D-56 · Ukuran Penerimaan untuk Modul Tanpa Baseline Pega
**Status:** DECIDED

**Pertanyaan (Ronde 4 Q21):** Dua modul **tidak punya pembanding Pega** sehingga gerbang 1 tidak dapat diberlakukan — `F-3` karena HCC/HCQ nol jejak di export (`docs/verifikasi-bukti-adr.md` §1.1), dan `S-5` karena sistem lama tidak punya jejak audit nilai sama sekali (§14.13). Apa penggantinya?

**Pilihan yang ditawarkan:**
1. Gerbang 1 diganti "lolos uji fungsional terhadap kontrak yang disepakati"
2. Kedua modul hanya melewati gerbang 2 (UAT), tanpa gerbang 1
3. Tunda keduanya sampai ada baseline
4. Belum ditetapkan

**Jawaban:** Opsi 1

**Keputusan akhir:** Untuk `F-3` dan `S-5`, gerbang 1 diganti **uji fungsional terhadap kontrak yang disepakati**:

| Modul | Kontrak yang menjadi acuan | Pemilik kontrak |
|---|---|---|
| **F-3** Identitas & Akses | kontrak API **HCC/HCQ** — field request/response login, kode error, timeout, endpoint refresh/validasi | pemilik API HCC/HCQ |
| **S-5** Jejak Audit | **daftar peristiwa wajib audit** beserta field yang harus tercatat | Compliance |

**Alasan menolak opsi 2:** menghapus gerbang 1 justru pada dua modul yang paling sensitif — `F-3` adalah otorisasi, `S-5` adalah jejak audit — menyisakan hanya UAT sebagai pemeriksaan, dan UAT tidak memeriksa hal yang tidak terlihat di layar.

**Ketergantungan baru yang diciptakan keputusan ini:** kedua kontrak **belum ada**. Sampai keduanya diterima:
- `F-3` tidak dapat lulus gerbang apa pun, dan tiketnya tetap `needs-info`.
- `S-5` dapat ditulis dan dikerjakan, tetapi tidak dapat dinyatakan lulus.

Keduanya masuk daftar tagihan pihak luar pada `D-36`, dan pemilik API HCC/HCQ adalah pihak yang **belum pernah dihubungi**.

**Dampak ke Steering:** Testing Strategy · BRD Bab 21.2 (pengecualian kriteria #3 untuk dua modul) · Risk Analysis (R-16)

---

### D-57 · Daftar Job Terjadwal Ditemukan — Menutup D-17 dan R-02
**Status:** DECIDED — menutup **D-17** yang berstatus `OPEN — gap discovery`

**Pertanyaan asal (D-17, Sesi 1):** Apa saja proses terjadwal / batch yang berjalan di sistem lama? Saat itu rule Agent dan Queue Processor tidak ada di export, sehingga daftar job tidak diketahui (**R-02**).

**Yang berubah:** pada 2026-09-09 export bertambah dan **tiga folder yang sebelumnya tidak ada** muncul — `Job Scheduler/` (5 berkas), `Agents/` (1), `Service REST/` (4).

**Keputusan akhir:** **D-17 ditutup dan R-02 tertutup pada tingkat artefak.** Daftar lengkap job terjadwal, terverifikasi dari `pyRuleName` dan `pyActivityName` masing-masing berkas:

| Job | Frekuensi | Jam mulai | Activity target | Target ada? |
|---|---|---|---|---|
| `JOBForKomiteKlaimPNC` | Daily, interval 1 | **06:00:00** | **`AutoAcceptKomite`** | ✅ |
| `JobHitungDeadlineToTemporaryCLose` | **Weekly** | **23:43:00** | `JobTemporaryCloseClaimPNC` | ✅ |
| `JobSendAutoLODKlaimPersonal` | Daily, interval 1 | **20:54:00** | `Act_SendAutoLODKlaimPersonal` | ✅ |
| `PNCMyReportKlaim3` | Daily, interval 1 | **08:00:00** | `ReportAI_Act` | ✅ |
| `ProcessClaimKredit` | Daily, interval 1 | **10:00:00** | `CreateClaimCredit_Table` | ✅ |

Seluruhnya `pyIsEnabled=true`, `pyApplicableTo=Cluster`, `pyNodeTypesText=BackgroundProcessing`.

**Agent:** `Agents/TATReportAgent-Agents.xml` — `pyEnable=true`, **`pyTriggerInterval=1800`** (30 menit), **`pyBypassActivityAuthentication=true`**. Merujuk `CreateCasePNCAgent_ActButton`, `AlertAgentKasirBlmTransferKlaim`, `TransferAllCaseNotAssigned` — **ketiganya ada**.

**Konsekuensi: `S-6` Penjadwalan naik dari TERHALANG menjadi SEBAGIAN.**

**Tiga hal yang tetap perlu jawaban Work Owner:**
1. **`JOBForKomiteKlaimPNC` menjalankan `AutoAcceptKomite` setiap hari jam 06:00** — job yang **menyetujui komite secara otomatis**. Perilaku otorisasi ini tidak tercatat di dokumen mana pun dan belum pernah dibahas. Benar demikian?
2. Tiga job ber-`pyDescription = "job jalan 2 menit"` sementara konfigurasinya `Daily`/`Weekly` — selisih 720× sampai 5.040×. Mana yang benar?
3. `pyBypassActivityAuthentication=true` pada agent — diterima untuk sistem baru?

**Penemuan lain dari folder baru:** `Service REST/` memuat **empat layanan masuk** — `KomiteAcceptAdjustment`, `KomiteAcceptAdjustmentPA`, `RecivedDataandAttachmentLelangASMSimasbid`, `RequestCreateClaimCredit2`. Dua di antaranya **menerima persetujuan komite dari sistem lain**. Seluruh analisis integrasi sebelumnya hanya melihat Connect REST **keluar**; permukaan masuk ini **menambah lingkup `S-4`** dan belum pernah masuk hitungan `FR-S4` maupun `D-25`.

**Dampak ke Steering:** Risk Analysis (R-02 ditutup) · Module Breakdown S-6 dan S-4 · `19-GAP-EXPORT-DETAIL.md` · BRD Bab 15 dan 21.3 kriteria #3

---

### D-58 · Peran Bisnis = 22 Access Group, Dinamai Ulang
**Status:** DECIDED

**Pertanyaan (Ronde 5 Q22):** Tiket wajib mengisi field `Peran penguji gerbang 2`, dan `F-3` harus membangun tabel peran dari nol. Yang tersedia hanya 22 nama access group Pega, dan `I-03` sudah menetapkan BRD memakai jabatan generik — bukan nama peran operasional. Berapa peran bisnis yang sebenarnya?

**Pilihan yang ditawarkan:**
1. Peran bisnis = 22 access group apa adanya, hanya dinamai ulang agar terbaca manusia
2. Lebih sedikit dari 22 — beberapa access group adalah varian teknis dari peran yang sama
3. Lebih banyak dari 22 — ada peran bisnis yang berbagi satu access group
4. Work owner menyebutkan daftarnya sendiri

**Jawaban:** Opsi 1

**Keputusan akhir:** **Peran bisnis di sistem baru berjumlah 22, satu-untuk-satu dengan access group Pega**, hanya dinamai ulang agar terbaca manusia. Tidak ada penggabungan dan tidak ada pemecahan.

Daftar 22 access group, seluruhnya terverifikasi ada di export sebagai literal `GCNMFW:<nama>`:

`Administrators` · `CaseManager` · `PncAdmin` · `PncManagerAdmin` · `PncPICTeknik` · `PNCKomiteTeknik` · `PNCKomite` · `PncRCLPUCL` · `PncAnalystDoctor` · `PncComplience` · `PncInvestigator` · `PNCSurveyor` · `PncPLADLA` · `PncReceive` · `PncManagerReceive` · `PncCollection` · `PncOPCGeneral` · `PNCServiceCenter` · `TreatyIn` · `ViewClaimPNC` · `PNCReportClaimInternal` · `PNCReportClaimEksternal`

**Konsekuensi yang harus ditangani di `F-3`:**
- **Tiga nama muncul dalam dua kapitalisasi** — `ViewClaimPNC`/`VIEWCLAIMPNC` dan `PncReceive`/`PNCRECEIVE`. Perbandingan access group di rule lama tidak konsisten soal huruf besar-kecil. Sistem baru wajib menormalkannya menjadi satu identitas per peran.
- Pemetaan **peran → 51 item menu** hidup hanya di dalam **34 When rule + `Navigation/pyCaseWorkerNavigation-Navigation.xml`**, bukan sebagai data. Sumber untuk mengisi tabel izin adalah membaca ke-34 When rule itu satu per satu — **5 di antaranya hilang dari export** (`IsGCNMReport`, `IsKomite`, `IsNotViewClaim`, `IsPNCBonding`, `IsSurvey`).
- **Penugasan operator ke peran tidak ada di database.** `POOLDATA.T_ACCESS_GROUP_PNC` hanya memetakan `OPERATOR_ID` → `OLD_OPERATOR_ID`. Tanpa artefak ini, `F-3` dapat membangun tabelnya tetapi tidak dapat mengisinya.

**Dampak ke Steering:** Domain Model (master Peran & Izin Menu) · Module Breakdown F-3 dan U-6 · Security

---

### D-59 · Satuan Izin adalah Menu, Bukan Tindakan — Tanpa Pemisahan Tugas
**Status:** DECIDED oleh work owner — dicatat dengan konsekuensi terbuka

**Pertanyaan (Ronde 5 Q23):** Siapa yang boleh **membatalkan klaim**, **mengubah nilai estimasi/akseptasi setelah komite menyetujui**, dan **menyetujui komite**? Export membuktikan otorisasi sistem lama adalah penyembunyian menu semata: `pyPrivilegeName` terisi hanya pada **1 dari 902** activity, dan yang satu itu privilege bawaan Pega untuk ekspor ruleset, bukan aturan bisnis (`docs/verifikasi-bukti-adr.md` §10.3).

**Pilihan yang ditawarkan:**
1. Ada pemisahan tugas nyata — pelaksana tidak boleh menyetujui; peran disebutkan per tindakan
2. Tidak ada pemisahan formal; kontrolnya prosedural
3. Sama dengan sistem lama — siapa pun yang punya akses menu bisa melakukannya
4. Belum pernah ditetapkan

**Jawaban:** Opsi 3

**Keputusan akhir:** **Satuan izin adalah menu, bukan tindakan individual.** Seorang pengguna yang memiliki akses ke sebuah menu berwenang atas seluruh tindakan yang dijangkau menu itu — termasuk membatalkan klaim, mengubah nilai setelah persetujuan komite, dan menyetujui komite. **Tidak ada pemisahan tugas formal.**

**Ini tetap memenuhi `BRD §21.2` kriteria #8 dan `FR-R1`,** dengan bacaan berikut yang disampaikan ke work owner dan tidak dikoreksi: yang diperiksa di setiap endpoint adalah **"apakah peran pemanggil memiliki menu yang memberi akses ke endpoint ini"**. Perbedaan dengan sistem lama bukan pada satuan izinnya, melainkan pada **tempat penegakannya** — dulu hanya disembunyikan di antarmuka, sekarang ditegakkan di server pada setiap endpoint.

**Konsekuensi yang diterima secara sadar:**

1. **Orang yang sama dapat membuat, menyetujui, dan membayarkan satu klaim** bila perannya memiliki ketiga menu itu. Tidak ada kontrol teknis yang mencegahnya.
2. **Jejak audit menjadi satu-satunya kontrol pengimbang.** Karena tidak ada pencegahan, yang tersisa hanyalah pencatatan. Ini **menaikkan** kepentingan `S-5` dari modul pendukung menjadi kontrol utama — dan `S-5` adalah kemampuan baru 100% tanpa baseline (D-56).
3. **`AutoAcceptKomite` melewati kontrol apa pun.** Job terjadwal harian jam 06:00 (D-57) menyetujui komite tanpa pengguna sama sekali, sehingga bahkan kontrol berbasis menu tidak berlaku padanya.
4. Bila kemudian ada temuan audit atau pentest yang menuntut pemisahan tugas, perubahannya menyentuh model izin di `F-3` — bukan penyesuaian kecil.

**Yang tetap berlaku:** `BRD §21.2` kriteria #9 (jejak audit untuk setiap perubahan bernilai bisnis) menjadi **wajib tanpa pengecualian** pada seluruh tiket modul bisnis, karena ia satu-satunya kontrol yang tersisa.

**Dampak ke Steering:** Security (Authorization) · Module Breakdown F-3 dan S-5 · BRD Bab 17 dan 21.2 · Risk Analysis

---

### D-60 · Peran Penguji Gerbang 2 per Modul
**Status:** DECIDED

**Pertanyaan (Ronde 5 Q24):** Setiap tiket wajib mengisi field `Peran penguji gerbang 2`. Untuk modul bisnis, peran penguji dapat diturunkan dari inbox yang dipakai. Untuk **modul fondasi** tidak bisa — tidak ada user bisnis yang membuka layar "Akses Data".

**Pilihan yang ditawarkan:**
1. Modul bisnis → peran pemakai inbox terkait; modul fondasi → tidak melewati gerbang 2, cukup gerbang 1 ditambah persetujuan Work Owner
2. Modul fondasi tetap melewati gerbang 2 dengan penguji yang ditunjuk Work Owner
3. Seluruh modul diuji peran yang sama — satu tim UAT terpusat
4. Belum ditetapkan

**Jawaban:** Opsi 1

**Keputusan akhir:**

| Kelompok modul | Gerbang 2 | Penguji |
|---|---|---|
| **Modul bisnis** (`B-1`…`B-14`, `S-1`…`S-4`, `S-6`, `S-7`, `U-1`, `U-3`…`U-6`) | **berlaku** | peran bisnis pemakai inbox/layar modul itu, diturunkan dari `Navigation/pyCaseWorkerNavigation-Navigation.xml` dan When rule peran |
| **Modul fondasi** (`F-1`, `F-2`, `F-3`, `F-4`, `F-5`, `S-5`, `S-8`, `U-2`) | **tidak berlaku** | gerbang 1 ditambah **persetujuan Work Owner** |

**Alasan:** memaksakan gerbang 2 pada modul yang tidak punya peran pemakai akan membuat tiket fondasi macet di gerbang yang tidak dapat dilewati siapa pun.

**Catatan penting terhadap `D-46`:** keputusan ini mengatur **tahap implementasi**, bukan tahap penulisan tiket. Pada tahap penulisan, `D-46` menetapkan agen yang me-review dan menyatakan tiket selesai. Field `Peran penguji gerbang 2` pada setiap tiket karena itu diisi sebagai **rencana untuk tahap implementasi**, bukan pernyataan bahwa gerbang itu sudah dilewati.

**Pengecualian yang mengikuti dari D-56:** `F-3` dan `S-5` tidak punya baseline Pega, sehingga gerbang 1 mereka diganti uji fungsional terhadap kontrak. Untuk kedua modul itu, keseluruhan kelulusan bertumpu pada kontrak yang **belum ada**.

**Terbuka — belum dijawab:** `C-9` dan `R-15` mencatat bahwa **waktu user bisnis untuk UAT belum dialokasikan resmi**. Pertanyaan itu diajukan bersama Q24 dan belum terjawab. Tanpa alokasi resmi, gerbang 2 bergantung pada ketersediaan user yang sedang menjalankan operasional harian — dan itu risiko jadwal yang sudah tercatat tetapi belum ditangani.

**Dampak ke Steering:** Testing Strategy · BRD Bab 19 (C-9) dan 21 · Risk Analysis (R-15)

---

### D-61 · Jadwal Tetap Seluruh Modul — Selisih Menjadi Tanggung Jawab Manajemen
**Status:** DECIDED oleh work owner — dicatat dengan risiko terbuka

**Pertanyaan (Ronde 6 Q25):** `D-30` menetapkan seluruh modul selesai akhir September 2026. Hari ini **14 September 2026**, tersisa **16 hari kalender**, dan pemeriksaan `find` untuk `*.go`, `go.mod`, `package.json` menghasilkan **nihil** — belum ada satu baris kode implementasi. Apa yang sebenarnya harus jadi akhir September?

**Yang berubah sejak `D-30` ditetapkan dan belum pernah dibawa kembali:** jumlah modul menjadi **33** (bukan 27, per `D-35` dan `D-42`) · **±397 artefak** masih diminta ke Tim Pega (`R-16`) · tiga pihak luar belum menjawab (`D-36`) · dan `D-44` menetapkan jumlah tim baru ditentukan **setelah** tiket direview.

**Pilihan yang ditawarkan:**
1. Target akhir September berlaku untuk artefak perencanaan — ADR, tiket, Steering yang direvisi — bukan untuk kode
2. Target tetap seluruh modul; tiket ditulis apa adanya dan selisihnya menjadi tanggung jawab manajemen
3. Scope dipangkas — work owner menyebutkan modul mana yang wajib akhir September
4. Dibawa ke Sponsor Project dengan papan tiket sebagai bahan

**Jawaban:** Opsi 2

**Keputusan akhir:** **Target `D-30` tidak berubah** — seluruh modul akhir September 2026. Tiket ditulis **apa adanya**, mencerminkan cakupan dan kendala yang sebenarnya, dan **selisih antara cakupan itu dengan tenggat menjadi tanggung jawab manajemen**.

**Konsekuensi yang mengikat bentuk artefak:**
- Tiket **tidak ditulis** dengan estimasi atau urutan yang berpura-pura tenggat ini terpenuhi. Cakupan ditulis sesuai bukti.
- `docs/ticketing/README.md` **wajib memuat satu bagian jujur tentang jadwal**: jumlah tiket, jumlah modul yang belum tersentuh, dan sisa waktu ke target, berdampingan. Angka tidak disajikan tanpa konteks.
- Papan tiket menjadi bahan keputusan manajemen — konsisten dengan `D-44` yang menjadikan hasil review tiket sebagai dasar penentuan jumlah tim.

**Risiko yang tetap terbuka dan tidak berubah oleh keputusan ini:** `R-05` (jadwal tidak sepadan dengan ukuran pekerjaan, diterima manajemen) dan `R-13` (tenggat tidak punya pemaksa eksternal — `I-01` menetapkan tidak ada lisensi berakhir, tidak ada penalti kontraktual, sehingga penjadwalan ulang sepenuhnya di dalam kendali manajemen Sinarmas).

**Dampak ke Steering:** Migration Strategy · Risk Analysis (R-05, R-13) · `docs/ticketing/README.md`

---

### D-62 · Retensi Jejak Audit Mengikuti Retensi Data Klaim
**Status:** DECIDED — angka masih perlu diambil; menutup arah `D-28`

**Pertanyaan (Ronde 6 Q26):** `D-28` menetapkan jejak audit append-only tetapi **lama retensinya belum ditentukan**. `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang karena tidak ada pemisahan tugas. Dan `BRD §21.3` kriteria #8 menjadikan retensi sesuai ketentuan Compliance sebagai syarat mematikan Pega.

**Pilihan yang ditawarkan:**
1. Ikuti ketentuan OJK untuk dokumen klaim asuransi
2. Sama dengan retensi data klaim yang berlaku sekarang
3. Belum tahu — harus ke Compliance, tiket `S-5` ditandai `needs-info`
4. Angka tertentu yang disebutkan work owner

**Jawaban:** Opsi 2

**Keputusan akhir:** **Retensi jejak audit mengikuti retensi data klaim yang berlaku sekarang.** Tidak ada kebijakan retensi terpisah untuk data audit — satu kebijakan berlaku untuk keduanya.

**Yang ini selesaikan:** `S-5` tidak perlu menunggu keputusan kebijakan baru dari Compliance. Kebijakannya sudah ada; yang dibutuhkan hanyalah **angkanya**.

**Yang masih perlu diambil:** angka retensi data klaim yang berlaku sekarang belum ada di repo maupun di dokumen proyek mana pun. Ia diambil dari kebijakan yang sudah berjalan — bukan diputuskan ulang. Sampai angkanya masuk, `S-5` dibangun dengan **retensi sebagai parameter konfigurasi** (konsisten `D-15`: tidak ada nilai bisnis yang di-hardcode), sehingga modulnya tidak terhalang.

**Dampak ke Steering:** Security · NFR (auditabilitas) · Database Strategy · BRD Bab 21.3 kriteria #8 · Module Breakdown S-5

---

### D-63 · Kewenangan Perubahan Skema Selama Masa Paralel
**Status:** DECIDED

**Pertanyaan (Ronde 6 Q27):** `D-21` menetapkan satu database bersama dan `P-1` satu penulis per tabel; `P-4` mewajibkan migrasi skema backward-compatible. Tetapi **siapa yang boleh menjalankan perubahan skema** belum ditetapkan — padahal `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` dibaca **116 rule** Pega yang sedang melayani produksi.

**Pilihan yang ditawarkan:**
1. DBA menjalankan, atas permintaan tertulis tim pengembang, dengan persetujuan work owner — dan setiap perubahan diuji menjalankan Pega dan Go bersamaan lebih dulu
2. Tim pengembang punya akses langsung ke skema aplikasi sendiri; tabel bersama lewat DBA
3. Hanya DBA, tanpa keterlibatan tim pengembang
4. Belum ditetapkan

**Jawaban:** Opsi 1

**Keputusan akhir:** Perubahan skema dijalankan **DBA**, atas **permintaan tertulis** tim pengembang, dengan **persetujuan Work Owner**. Setiap perubahan **wajib diuji lebih dulu dengan menjalankan Pega dan Go bersamaan** terhadap skema hasil perubahan.

**Alasan:** `P-4` mewajibkan verifikasi backward-compatible dengan menjalankan versi lama dan baru bersamaan — itu tidak dapat dilakukan DBA sendirian maupun tim pengembang sendirian. Dan karena ada tabel yang dibaca 116 rule Pega, satu `ALTER` yang keliru menghentikan sistem yang sedang melayani produksi.

**Konsekuensi untuk acceptance criteria:** `BRD §21.2` kriteria #13 (migrasi skema backward-compatible, diverifikasi dengan menjalankan versi lama dan baru bersamaan) menjadi **dapat diuji**, karena prosedurnya kini punya pemilik dan langkah. Setiap tiket yang menyentuh skema wajib memuat bagian rollback yang tidak kosong, sesuai `P-4`.

**Konsekuensi operasional:** setiap perubahan skema menempuh tiga pihak. Ini memperlambat iterasi, dan konsekuensi itu diterima demi melindungi produksi yang masih dilayani Pega.

**Dampak ke Steering:** Database Strategy (migrasi skema) · Deployment Strategy · Migration Strategy `P-4` · BRD Bab 21.2 kriteria #13

---

### D-64 · Data Nasabah Disalin Apa Adanya ke Staging
**Status:** DECIDED oleh work owner — dicatat dengan konsekuensi terbuka

**Pertanyaan (Ronde 6 Q28a):** `D-53` menetapkan uji kesetaraan dijalankan atas salinan data produksi di staging. Apakah data nasabah disamarkan sebelum disalin?

**Pilihan yang ditawarkan:**
1. Disamarkan sebelum disalin — nomor polis, nama tertanggung, NPWP, rekening
2. Disalin apa adanya, dengan hak akses staging diperketat
3. Perlu ke Compliance dulu

**Jawaban:** Opsi 2

**Keputusan akhir:** Data produksi disalin ke staging **apa adanya, tanpa penyamaran**, dengan **hak akses lingkungan staging diperketat** sebagai kontrol penggantinya.

**Konsekuensi yang diterima secara sadar:**
1. **Staging menjadi lingkungan yang memuat data nasabah nyata** — nomor polis, nama tertanggung, NPWP, nomor rekening, data medis pada lini PA dan Travel. Perlindungannya sepenuhnya bergantung pada hak akses, bukan pada sifat datanya.
2. Klasifikasi staging **naik setara produksi** untuk keperluan keamanan: siapa yang punya akses, berapa lama salinan disimpan, dan kapan dimusnahkan menjadi pertanyaan yang harus dijawab — dan **belum dijawab**.
3. Akses medis yang `FR-R2` batasi pada peran Analyst Doctor dan RCL Dokter berlaku juga di staging, bukan hanya produksi.
4. Keputusan ini **berdampingan dengan `D-40`** yang masih terbuka: export rule sudah memuat kredensial plaintext, dan kini salinan data nasabah menyusul ke lingkungan yang sama-sama belum ditetapkan pengamanannya.

**Yang tidak berubah:** aturan kerja project tetap melarang **menampilkan atau menyalin data nasabah ke dokumen yang akan di-commit**. Keputusan ini mengatur isi lingkungan staging, bukan isi dokumen. ADR dan tiket tetap merujuk dengan nama kolom dan `berkas:baris`, tanpa nilai.

**Terbuka:** siapa yang menyetujui akses staging, berapa lama salinan disimpan, dan prosedur pemusnahannya. Ditujukan ke pihak yang sama dengan `D-40`.

**Dampak ke Steering:** Security (data sensitif) · Testing Strategy · Deployment Strategy (lingkungan) · Risk Analysis (R-17)

---

### D-65 · Kebijakan Penghapusan Data Mengikuti Sistem Lama — dengan Pengecualian Jejak Audit
**Status:** DECIDED oleh work owner — **dengan satu pertentangan terhadap `D-28` yang perlu diselesaikan**

**Pertanyaan (Ronde 6 Q28b):** Bagaimana penghapusan data di sistem baru? Sistem lama melakukan `DELETE` dan `UPDATE` nyata pada tabel log.

**Pilihan yang ditawarkan:**
1. Soft delete di mana pun — tidak ada `DELETE` fisik pada data bernilai bisnis
2. Hard delete untuk data tertentu — work owner menyebutkan mana
3. Sama dengan sistem lama

**Jawaban:** Opsi 3

**Keputusan akhir untuk data bisnis:** kebijakan penghapusan **mengikuti sistem lama**. Pola yang terbukti di export dan menjadi acuan: `RDB-DELETE` pada 12 step di 7 activity, `OBJ-DELETE` pada 2 step di 2 activity, dan `DELETE` pada rule SQL — termasuk `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503` yang menghapus lalu menyisipkan ulang seluruh pohon klaim sebagai pola idempoten.

**Pertentangan yang harus diselesaikan.** `D-28` (Sesi 2) menetapkan:

> *"Data audit bersifat append-only: tidak boleh diubah atau dihapus oleh jalur aplikasi mana pun."*

Sementara sistem lama **melanggar itu pada tabel lognya sendiri**, terbukti:

| Operasi | Tabel | `berkas:baris` |
|---|---|---|
| **`UPDATE`** | `pooldata.claim_service_log` — `set jsonout = … where id = …` | `RDB List/UpdateLogServiceClaim-SQL.xml:27` |
| **`DELETE`** | `POOLDATA.JSON_KLAIM_LOG` — `delete … where a.idpega=… and a.tgl_input=…` | `RDB List/InsertClaimPNC-SQL.xml:77` |
| `UPDATE` | `T_LOGINCOAS` | `RDB List/BrowseOldEmailCoas-SQL.xml:69` |
| `UPDATE` | `pooldata.mst_login_surveyor` | `RDB List/UpdateMasterLoginSurvey-SQL.xml:9` |

Menerapkan "sama dengan sistem lama" secara harfiah pada tabel-tabel itu akan **mereplikasi pelanggaran `D-28`** — dan `D-59` baru saja menjadikan jejak audit **satu-satunya kontrol pengimbang** karena tidak ada pemisahan tugas.

**Resolusi yang diusulkan, menunggu persetujuan work owner:**

| Kelas data | Kebijakan | Dasar |
|---|---|---|
| **Data bisnis** (klaim, objek, coverage, estimasi, adjustment, master) | **mengikuti sistem lama** — hard delete di tempat sistem lama melakukannya | D-65 |
| **Jejak audit `S-5`** (modul baru) | **append-only mutlak** — tidak ada `UPDATE`, tidak ada `DELETE` | D-28, dikuatkan D-59 |
| **Tabel log warisan** (`claim_service_log`, `json_klaim_log`, `T_LOGINCOAS`, `mst_login_surveyor`) | **bukan jejak audit** — diperlakukan sebagai data bisnis, boleh mengikuti sistem lama | konsekuensi pemisahan di atas |

Dengan pemisahan ini, `D-65` dan `D-28` tidak bertentangan: yang "mengikuti sistem lama" adalah data bisnis dan tabel log warisan; yang append-only adalah jejak audit baru yang dibangun `S-5`.

**Bila resolusi ini tidak disetujui**, salah satu dari dua hal harus dipilih secara eksplisit: `D-28` dicabut untuk tabel log warisan, atau `D-65` dikecualikan untuk tabel log warisan. Tiket `S-5` dan `B-1` tidak dapat ditulis sebelum ini jelas.

**Dampak ke Steering:** Database Strategy · Security · Domain Model · Module Breakdown S-5 dan B-1 · BRD Bab 17.5 dan 21.2 kriteria #9

---

### D-66 · Soft Delete Menyeluruh — Menyupersede D-65
**Status:** DECIDED — **menyupersede `D-65`**

**Latar:** `D-65` mencatat jawaban Ronde 6 Q28(b) Opsi 3 ("sama dengan sistem lama") dan mengangkat bahwa jawaban itu bertentangan dengan `D-28` (jejak audit append-only), lalu mengusulkan pemisahan tiga kelas data sebagai resolusi. Work owner **merevisi jawabannya menjadi Opsi 1** sebelum resolusi itu disetujui.

**Pertanyaan (Ronde 6 Q28b):** Bagaimana penghapusan data di sistem baru?

**Pilihan yang ditawarkan:**
1. Soft delete di mana pun — tidak ada `DELETE` fisik pada data bernilai bisnis
2. Hard delete untuk data tertentu — work owner menyebutkan mana
3. Sama dengan sistem lama

**Jawaban pertama:** Opsi 3
**Jawaban revisi:** Opsi 1

**Keputusan akhir:** **Soft delete berlaku menyeluruh.** Tidak ada `DELETE` fisik pada data bernilai bisnis di sistem baru. Penghapusan dinyatakan lewat penanda (mis. kolom flag beserta waktu dan pelaku), bukan lewat pembuangan baris.

**Pertentangan dengan `D-28` hilang.** Karena tidak ada penghapusan fisik pada data bernilai bisnis, jejak audit append-only (`D-28`) dan kebijakan penghapusan (`D-66`) berdiri di atas prinsip yang sama. Pemisahan tiga kelas data yang diusulkan `D-65` **tidak diperlukan dan tidak dipakai**.

**Yang ini perbaiki dari sistem lama.** Sistem lama melakukan penghapusan fisik di banyak tempat, termasuk pada tabel lognya sendiri:

| Operasi | Objek | `berkas:baris` |
|---|---|---|
| `UPDATE` | `pooldata.claim_service_log` | `RDB List/UpdateLogServiceClaim-SQL.xml:27` |
| `DELETE` | `POOLDATA.JSON_KLAIM_LOG` | `RDB List/InsertClaimPNC-SQL.xml:77` |
| `UPDATE` | `T_LOGINCOAS` | `RDB List/BrowseOldEmailCoas-SQL.xml:69` |
| `UPDATE` | `pooldata.mst_login_surveyor` | `RDB List/UpdateMasterLoginSurvey-SQL.xml:9` |
| `RDB-DELETE` · `OBJ-DELETE` | berbagai | 12 step di 7 activity · 2 step di 2 activity |

Seluruhnya tidak dibawa ke sistem baru.

**Konsekuensi desain yang harus ditangani di `B-1`.** Sistem lama memakai pola **hapus-lalu-sisip-ulang** sebagai mekanisme idempotensi: `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503` menghapus **12 tabel** milik satu klaim lalu menyisipkan ulang seluruh pohonnya. Pola itu **bergantung pada penghapusan fisik** dan tidak dapat dipertahankan apa adanya.

Penggantinya harus diputuskan saat menulis tiket `B-1` — kandidatnya *upsert* berbasis kunci alami, atau versioning dengan penanda baris aktif. Catatan pendukung: dua tabel pada blok itu (`T_DLALIST`, `T_PLALIST`) **delete-nya sudah dikomentari** di sumber (`:497`, `:498`) sementara insert-nya tetap aktif (`:1296`, `:1325`) — sehingga konversi ulang pada kedua tabel itu **sudah berpotensi menduplikasi baris hari ini**. Pola penggantinya sekaligus menutup cacat yang sudah ada.

**Konsekuensi untuk uji kesetaraan.** Soft delete **tidak mengubah hasil yang dilihat pengguna** — yang terhapus tetap tidak muncul. Tetapi ia **mengubah isi tabel**. Perkakas `S-8` karena itu membandingkan **hasil kueri sesuai aturan bisnis**, bukan jumlah baris mentah; perbandingan berbasis `COUNT(*)` pada tabel akan selalu berbeda dan bukan indikasi cacat.

**Konsekuensi untuk `P-5`.** Soft delete **tidak ditambahkan** ke daftar 13 perbaikan eksplisit `D-49`, karena ia tidak mengubah hasil yang terlihat — ia mengubah cara penyimpanan. Bila pada uji kesetaraan muncul selisih yang berasal dari perbedaan ini, ia diperlakukan sebagai **selisih metode pembandingan**, bukan selisih hasil, dan ditangani di sisi perkakas `S-8`.

**Dampak ke Steering:** Database Strategy · Domain Model · Security · Module Breakdown B-1, S-5, S-8 · BRD Bab 17.5 dan 21.2 kriteria #9

---

### D-67 · Tidak Ada Akun Pribadi sebagai Penerima Notifikasi
**Status:** DECIDED

**Pertanyaan (Ronde 7 Q29):** Export memuat sedikitnya enam alamat Gmail pribadi sebagai penerima notifikasi di jalur produksi, ditambah lima alamat Gmail huruf besar yang dipakai sebagai **Operator ID** di filter laporan KPI.

**Pilihan yang ditawarkan:**
1. Ganti seluruhnya dengan mailbox fungsional dari master Penerima Notifikasi
2. Sebagian masih sah — work owner menyebutkan mana
3. Orangnya sudah tidak bekerja di sini; hapus saja
4. Perlu dicek dulu

**Jawaban:** Opsi 1

**Keputusan akhir:** **Tidak ada satu pun alamat pribadi yang dibawa ke sistem baru.** Seluruh penerima notifikasi berasal dari master **Penerima Notifikasi** (`F-4`), berupa mailbox fungsional. Pemetaan konkret alamat lama ke mailbox pengganti ditetapkan saat master diisi.

Lokasi yang harus dibersihkan, seluruhnya terverifikasi:

| Jenis | `berkas:baris` |
|---|---|
| Gmail pribadi sebagai penerima | `Activity/AutoEmailDownloadProposeAdjustment-Act.xml:7872`, `:8031` · `Activity/SetStsSalvagePNC_act-Act.xml:694` · `Activity/LetterOfAssignment2_Act-Act.xml:7725` (elemen `<To>`) · `Activity/SendEmailUnprotectPremi-Act.xml:4823` |
| Gmail huruf besar sebagai **Operator ID** di filter KPI | `Activity/PNCReportKPI_act-Act.xml:7913`, `:9564` · `Activity/GetNextFUdata_act-Act.xml:3522` · `Activity/EksportDataAllKPIPICKlaim-Act.xml:3790` |

**Catatan yang memperluas cakupan `F-4`:** lima alamat Gmail yang dipakai sebagai Operator ID **bukan** masalah notifikasi melainkan masalah **identitas** — mereka muncul di klausa `pic in (…)` pada SQL laporan. Pembersihannya menyentuh `F-3` (identitas) sekaligus `F-4` (master), bukan hanya master penerima notifikasi.

**Dampak ke Steering:** Domain Model (master Penerima Notifikasi) · Module Breakdown F-3 dan F-4 · Security · `FR-F4`

---

### D-68 · Claim PNC Satu-satunya Pemanggil Stored Procedure — B-4 dan B-9 Dapat Dibuat Atomik
**Status:** DECIDED

**Pertanyaan (Ronde 7 Q30):** `D-02` memutuskan aplikasi baru tidak memanggil stored procedure. Tetapi **10 dari 12 procedure** yang dibaca melakukan `COMMIT` sendiri — `Database/INSERT_PLADLA.prc` sembilan kali (`:69`, `:74`, `:79`, `:138`, `:143`, `:148`, `:179`, `:184`, `:189`) dengan hanya satu `ROLLBACK` di handler terluar (`:198`), dan `Database/UPDATEREAS.prc` empat kali. Selama procedure itu dipanggil apa adanya, `B-4` dan `B-9` tidak dapat dibuat atomik. Menulis ulangnya di Go menyelesaikan itu — tetapi hanya bila Claim PNC satu-satunya pemanggilnya.

**Pilihan yang ditawarkan:**
1. Claim PNC satu-satunya pemanggil — procedure boleh ditinggalkan setelah logikanya naik ke Go
2. Ada sistem lain yang memanggilnya — procedure harus tetap hidup
3. Belum tahu — perlu ke DBA untuk memeriksa dependensi
4. Semua procedure tetap hidup apa pun jawabannya

**Jawaban:** Opsi 1

**Keputusan akhir:** **Claim PNC adalah satu-satunya pemanggil** stored procedure yang dibahas. Setelah logikanya dinaikkan ke Go sesuai `D-02`, procedure tersebut **boleh ditinggalkan** — tidak perlu dipelihara demi sistem lain.

**Yang ini lepaskan:**
- **`B-4` Spreading Reasuransi** dan **`B-9` PLA/DLA** dapat dibuat **atomik**. Penerbitan PLA/DLA yang di sistem lama menempuh sembilan `COMMIT` dan berpotensi meninggalkan state setengah jalan (karena `ROLLBACK`-nya terjadi setelah commit) kini dapat dibungkus satu transaksi.
- Kepemilikan transaksi berpindah sepenuhnya ke lapisan Go, konsisten dengan `D-02` ("database menjadi penyimpanan murni").
- Kontrak galat berbasis string `ErrMsg` — yang pada enam procedure **tidak di-set pada jalur sukses** sehingga `NULL` berarti sukses, dan pada `ADD_NEWMASTERVIRTUALACCOUNT.prc:18` bahkan **membawa nomor virtual account** sekaligus pesan galat — tidak ikut dibawa.

**Yang tetap harus diminta ke DBA meski keputusan ini diambil:** **12 dependensi** yang dipanggil oleh 62 procedure yang sudah diterima, terberat `UPDATE_LOG_KONVERSI` (**162 pemanggilan**), `GETNEWID` (42), `PKG_COUNTER_PRODUCTION` (25), `PROCESSQUEUEDIRECT` (10). Logikanya tetap perlu dibaca untuk ditulis ulang, meski objeknya nanti ditinggalkan.

**Catatan verifikasi:** keputusan ini dapat dikonfirmasi DBA dengan satu kueri katalog (`ALL_DEPENDENCIES`) bila kelak diperlukan bukti tertulis sebelum procedure benar-benar dinonaktifkan.

**Dampak ke Steering:** Technical Strategy (transaksi) · Database Strategy · Module Breakdown B-4 dan B-9 · `D-02`

---

### D-69 · Perlakuan Nama Orang dan Alamat Email di ADR dan Tiket
**Status:** DECIDED

**Pertanyaan (Ronde 7 Q31):** Apakah nama Operator ID dan alamat email boleh tertulis lengkap di ADR dan tiket yang akan di-commit? Nama seperti `MORASOTARDODOTARIGAN`, `ELLENSUPRIYATI`, dan `INDRAGUNAWAN` **sudah tertulis apa adanya** di dokumen yang disetujui (`D-15`, `20-DETAIL-KOMITE-DBLINK.md` §1.5.4); alamat email tidak pernah.

**Pilihan yang ditawarkan:**
1. Nama Operator ID boleh ditulis; alamat email selalu disamarkan; data nasabah tidak pernah
2. Keduanya disamarkan mulai sekarang, termasuk nama Operator ID
3. Keduanya boleh ditulis lengkap
4. Perlu ke Compliance

**Jawaban:** Opsi 1

**Keputusan akhir:** Aturan penulisan untuk seluruh dokumen yang akan di-commit:

| Jenis nilai | Perlakuan |
|---|---|
| **Nama Operator ID** | **boleh ditulis lengkap** — preseden sudah ada di Steering yang disetujui, dan diperlukan agar tiket dapat menunjuk hardcode mana yang harus dihapus |
| **Alamat email** | **selalu disamarkan** — bagian sebelum `@` diganti, domain boleh tampak |
| **Data nasabah** — nomor polis, nama tertanggung, nomor klaim asli, NPWP, nomor rekening | **tidak pernah ditulis**; dirujuk dengan `berkas:baris` atau nama kolom saja |
| **Kredensial, kunci API, hostname/IP produksi** | **tidak pernah ditulis**; dirujuk dengan `berkas:baris` + nama elemen (lihat `D-40`) |

**Alasan:** nama Operator ID diperlukan untuk ketelusuran — tanpanya tiket `F-4` tidak dapat menunjuk hardcode mana yang dihapus. Alamat email tidak menambah ketelusuran apa pun, sehingga tidak ada alasan menuliskannya.

**Yang tidak berubah:** `D-64` menetapkan data nasabah nyata disalin ke staging. Keputusan ini mengatur **isi dokumen**, bukan isi lingkungan — keduanya berdiri sendiri.

**Berlaku surut:** aturan ini sudah dijalankan sepanjang Fase 1 dan Fase 2 pada `docs/verifikasi-bukti-adr.md` dan seluruh entri Decision Log Sesi 3.

**Dampak ke Steering:** tidak ada — ini aturan penulisan artefak, bukan arsitektur.

---

### D-70 · `TYPE_KOMITE` adalah Pita Nilai **hanya** di Non-MBU — Membatasi D-52
**Status:** DECIDED — **membatasi cakupan `D-52`** (tidak menyupersede; isi `D-52` tetap benar untuk Non-MBU)

**Pertanyaan (Ronde 8 Q32):** Saat menajamkan definisi "kumulatif" di `CONTEXT.md`, work owner menegaskan bahwa akumulasi jenjang didahului filter `TYPE_KOMITE` (`'1'` atau `'2'`) untuk menetapkan jenjang awal. Verifikasi ke master membuktikan aturan itu **tidak berlaku seragam di semua lini**.

**Bukti yang memicu pertanyaan:**

| Bukti | Isi |
|---|---|
| 17 SQL rule membaca `EMAILKOMITE`; **4** memfilter `TYPE_KOMITE` | `EmailKomiteBerjenjang_sql-SQL.xml:52` · `EmailKomiteBerjenjangPATKI_sql-SQL.xml:82` · `EmailKomiteAdjuster_sql-SQL.xml:101` · `EmailKomiteSalvage_sql-SQL.xml:96` |
| Jalur utama **PA** dan **Travel** **tidak** memfilternya | `EmailKomiteBerjenjangPA_sql-SQL.xml:5` · `EmailKomiteBerjenjangTravel_sql-SQL.xml:103` |
| `LIMIT_TOP` tetap **0 kemunculan** di seluruh `RDB List/` | menegaskan kembali `D-47` |

Isi master `Database/emailkomite.csv` (baris aktif, `STS_AKTIF='1'` dan `STS_ADJ='1'`):

| Lini | Ambang bawah per jenjang | `DEGREE` | `TYPE_KOMITE` | `berkas:baris` |
|---|---|---|---|---|
| **NONMBU** | `0` · `50.000.001` | 1 · 1 | **1** · **1** | `emailkomite.csv:7`, `:5` |
| **NONMBU** | `100.000.001` · `500.000.001` · `1.000.000.001` | 2 · 3 · 4 | **2** · **2** · **2** | `emailkomite.csv:2`, `:3`, `:4` |
| **PA** | `0` · `10.000.001` · `50.000.001` · `100.000.001` | 1 · 2 · 3 · 4 | **2** · **1** · **1** · **2** | `emailkomite.csv:24-27` |
| **TRAVEL** | `0` · `50.000.001` · `100.000.001` | 1 · 2 · 3 | **1** · **1** · **1** | `emailkomite.csv:21-23` |

Pada **Non-MBU** nilai `TYPE_KOMITE` memang memisahkan pita ≤ Rp 100 Jt dari pita di atasnya. Pada **PA** nilainya berselang-seling **2 · 1 · 1 · 2** menaiki tangga — ia membedakan **jalur PA reguler dari jalur PA TKI** (`EmailKomiteBerjenjangPATKI_sql-SQL.xml:82` mematok `type_komite='2'`), bukan pita nilai. Pada **Travel** seluruh jenjang aktif bernilai `1`, termasuk jenjang di atas Rp 100 Jt.

**Akibat bila filter pita diberlakukan seragam** (dihitung dari master di atas):

| Kasus | Perilaku sistem lama | Bila pita disaring dulu |
|---|---|---|
| PA Rp 5.000.000 | 1 penyetuju | **0 penyetuju** — klaim mandek |
| PA Rp 75.000.000 | 3 penyetuju | 2 penyetuju |
| PA Rp 150.000.000 | 4 penyetuju | 2 penyetuju |
| Travel Rp 150.000.000 | 3 penyetuju | **0 penyetuju** — klaim mandek |
| Non-MBU, seluruh nilai | — | tidak berubah |

**Pilihan yang ditawarkan:**
1. Filter pita berlaku **per lini sesuai kueri hari ini**; `D-52` dinyatakan berlaku khusus Non-MBU
2. Filter pita berlaku seragam, master PA dan Travel diperbaiki sebelum migrasi
3. `TYPE_KOMITE` dipecah menjadi dua kolom di sistem baru: pita nilai dan varian jalur
4. Perlu dikonfirmasi dulu ke pengguna bisnis PA dan Travel

**Jawaban:** Opsi 1 — *"filter type_komite khusus Non MBU"*

**Keputusan akhir:** **`TYPE_KOMITE` diperlakukan sebagai pita nilai hanya pada lini Non-MBU.**

- **Non-MBU** — pita ditetapkan **lebih dulu** dari nilai klaim (≤ Rp 100.000.000 → pita `1`; di atasnya → pita `2`, sesuai `D-52`), lalu jenjang diakumulasi **di dalam pita itu saja**. Akumulasi tidak menyeberang antarpita.
- **Lini lain** (PA, Travel, Simasnet, Bonding, NONMBUAB, NONMBUC) — **tidak ada** langkah pendahuluan. Akumulasi `LIMIT_BOTTOM <= nilai klaim` berjalan atas seluruh jenjang lini tersebut, persis seperti kuerinya hari ini.
- **PA TKI** tetap jalur tersendiri yang membatasi diri pada `TYPE_KOMITE = '2'`, dan itu **bukan** pita nilai melainkan penanda varian jalur.

**Cakupan `D-52` dibatasi, bukan dibatalkan.** Isi `D-52` — pita ≤ Rp 100 Jt memakai `TYPE_KOMITE=1` — tetap benar dan tetap berlaku, tetapi **hanya untuk Non-MBU**. Di luar Non-MBU, `TYPE_KOMITE` tidak boleh dibaca sebagai pita nilai.

**Mengapa ini yang benar terhadap `P-5`:** perilaku sistem lama dipertahankan persis di setiap lini, sehingga uji kesetaraan gerbang 1 pada `B-7` dapat dijalankan tanpa pengecualian. Menyeragamkan filter justru akan menghasilkan klaim tanpa penyetuju sama sekali pada dua kasus di atas — perubahan perilaku, bukan perbaikan.

**Konsekuensi yang harus ditangani saat menulis tiket:**

1. **`B-7` (Komite) wajib menguji per lini, bukan satu model tunggal.** Kasus uji minimum: PA Rp 5 Jt (1 penyetuju), PA Rp 75 Jt (3), PA Rp 150 Jt (4), Travel Rp 150 Jt (3), Non-MBU Rp 80 Jt (2), Non-MBU Rp 750 Jt (2), Non-MBU Rp 2 M (3).
2. **Satu kolom memikul dua arti berbeda.** `TYPE_KOMITE` berarti pita nilai di Non-MBU dan varian jalur di PA. Opsi 3 (memecahnya menjadi dua kolom) **tidak diambil**; karena itu model data baru wajib mendokumentasikan arti gandanya, dan validasi master `LIMIT_TOP` dari `D-47` hanya dapat dijalankan **per lini** — bukan lintas lini.
3. **Belum terverifikasi — dibawa ke tiket `B-7` dan `B-12`:** tiga kueri yang memfilter `TYPE_KOMITE` secara dinamis — `EmailKomiteBerjenjang_sql` (dipanggil `Activity/SetEmailKomite-Act.xml`), `EmailKomiteAdjuster_sql` (`Activity/SetEmailKomiteAdjuster-Act.xml`), `EmailKomiteSalvage_sql` (`Activity/SetEmailKomiteSalvage-Act.xml`) — menerima nilai `TYPE_BUSINESS` dan `TYPE_KOMITE` dari pemanggilnya lewat `tempAdj.pyMemo` dan `tempAdj.AcceptedNo`. **Lini apa saja yang benar-benar melewati ketiga kueri itu belum ditelusuri sampai ke sumber nilainya.** Bila ternyata ada lini selain Non-MBU yang melewatinya, perilaku lini tersebut mengikuti kuerinya apa adanya — bukan aturan pita. Penelusuran ini menjadi bagian Definition of Ready tiket `B-7` dan `B-12`.

**Dampak ke Steering:** Domain Model (Pita Nilai Komite, Jenjang Kumulatif) · Module Breakdown B-7 dan B-12 · Testing Strategy · `D-52` · `D-47`

---

### D-71 · Format Nomor Klaim Menjadi `PNCN.YY.xxxx` — Menyupersede Bagian Format pada D-22
**Status:** DECIDED — **menyupersede `D-22` pada bagian format nomor**; sisa isi `D-22` tetap berlaku

**Latar:** `D-22` menetapkan format `PNCN-xxxx` dengan sintaks `'PNCN-' || TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)`. Format itu tidak memuat unsur tahun, sehingga pertanyaan "berapa digit `xxxx`, dan apakah memuat unsur tahun" tercatat sebagai pertanyaan terbuka pada `ADR-0009`.

**Jawaban work owner:**
> "koreksi dulu penomoran klaim dari format PNCN-xxxx menjadi PNCN.YY.xxxx menggunakan sequence di oracle yang akan diberi nama POOLDATA.CLAIM_NO_NONPEGA_SEQ dengan syntax `'PNCN' || '.' || TO_CHAR(SYSDATE,'RR') || '.' || TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)`"

**Keputusan akhir:** Nomor klaim yang diterbitkan sistem baru berformat **`PNCN.YY.xxxx`** — tiga segmen dipisahkan **titik**:

| Segmen | Isi | Sumber |
|---|---|---|
| `PNCN` | penanda tetap asal sistem baru | literal |
| `YY` | dua digit tahun | `TO_CHAR(SYSDATE,'RR')` |
| `xxxx` | nomor urut | `TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)` |

Sintaks Oracle yang ditetapkan, apa adanya:

```sql
'PNCN' || '.' || TO_CHAR(SYSDATE,'RR') || '.' || TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)
```

**Yang tetap berlaku dari `D-22`** — tidak disupersede:
- Prefix `ASM-FW-GCNMFW-WORK` **tidak dipakai lagi**.
- Nomor klaim lama **dibiarkan apa adanya**, tidak dinomori ulang; kedua format hidup berdampingan permanen.
- Prefix `PNCN` membuat asal sebuah klaim terbaca langsung dari nomornya tanpa tabel pemetaan.

**Yang disupersede:** bentuk `PNCN-xxxx` (tanda hubung, tanpa segmen tahun) beserta sintaks lamanya.

**Fakta yang diverifikasi ulang saat keputusan ini diambil:** `POOLDATA.CLAIM_NO_NONPEGA_SEQ` **nol kemunculan di seluruh export** — sequence itu memang **belum ada** dan akan dibuat, sesuai kalimat *"yang akan diberi nama"* pada `D-22` maupun `D-71`. Catatan pada `ADR-0009` yang menyebut "pemakaiannya oleh sistem lain belum diverifikasi" karena itu **dikoreksi**: bukan belum diverifikasi, melainkan objeknya belum ada.

**Tiga hal yang mengikuti dari sintaks ini dan perlu diputuskan sebelum `B-2` ditulis lengkap — tidak saya isi sendiri:**

1. **Apakah sequence direset setiap awal tahun?** Sintaksnya memakai satu sequence global. Bila **tidak** direset, nomor urut terus bertambah melewati pergantian tahun — `PNCN.26.8125` diikuti `PNCN.27.8126` — sehingga segmen tahun bersifat penanda, bukan penghitung per tahun. Bila **direset**, nomor urut mulai dari 1 tiap tahun dan keunikan lintas tahun dijamin segmen `YY`. Keduanya sah; pilihannya mengubah cara nomor dibaca orang.
2. **`TO_CHAR(...NEXTVAL)` tanpa format mask tidak memberi angka nol di depan**, sehingga lebar segmen terakhir berubah-ubah: `PNCN.26.9` lalu `PNCN.26.10` lalu `PNCN.26.1000`. Akibatnya **pengurutan sebagai teks tidak sesuai urutan penerbitan** (`.10` mendahului `.9`). Bila lebar tetap dikehendaki, mask-nya perlu ditetapkan (mis. `TO_CHAR(seq.NEXTVAL,'FM0000')`). Sintaks yang ditetapkan work owner dipakai apa adanya sampai ada keputusan lain.
3. **`SYSDATE` adalah tanggal server basis data**, bukan tanggal kejadian maupun tanggal registrasi. Klaim yang diterbitkan di sekitar pergantian tahun akan mengambil tahun dari jam server — bertaut dengan `R-12` (pergeseran zona waktu) dan dengan temuan penyesuaian 7 jam pada `D-49` butir 3.

**Catatan teknis netral:** pada `TO_CHAR`, format `'RR'` menghasilkan dua digit tahun yang **identik** dengan `'YY'`; perbedaan perilaku `RR` hanya berlaku saat menafsirkan masukan (`TO_DATE`), bukan saat mengeluarkan teks. Sintaks work owner dipakai apa adanya.

**Konsekuensi untuk `ADR-0005`:** generator nomor klaim tetap menjadi **satu-satunya tempat dengan sakelar dialek eksplisit**. Padanan PostgreSQL-nya menyentuh dua hal sekaligus — `nextval('pooldata.claim_no_nonpega_seq')` dan `to_char(current_date,'YY')` — sehingga sakelar itu kini membungkus dua perbedaan dialek, bukan satu.

**Dokumen yang memuat format lama dan perlu direvisi — diajukan sebagai usulan, belum dikerjakan:**

| Dokumen | Lokasi |
|---|---|
| `docs/BRD.md` | `:246`, `:296`, `:311`, `:322`, `:770`, `:883`, `:1436` |
| `docs/requirement-summary.md` | `:24` (`P-3`), `:296` (`DAT-03`) |
| `docs/prompt-adr-ticketing.md` | `:193`, `:448` |

`docs/interview-history.md:65` **tidak diusulkan berubah** — berkas itu catatan riwayat wawancara, dan isinya memang merekam jawaban yang berlaku saat itu.

**Yang sudah dikerjakan atas keputusan ini:** `docs/adr/0009-penomoran-klaim-pncn.md` dan `docs/adr/README.md` dikoreksi, lalu `docs/ADR.md` dan `docs/ADR.docx` dibangun ulang.

**Dampak ke Steering:** Domain Model · Database Strategy · Module Breakdown B-2 · `D-22` · `ADR-0005`, `ADR-0009`

---

### D-72 · Steering dan BRD Diperbarui Langsung — Larangan Menyunting Dicabut
**Status:** DECIDED — **mencabut batasan kerja awal**, bukan menyupersede keputusan arsitektur mana pun

**Latar:** aturan kerja proyek sejak awal berbunyi *"JANGAN mengedit `docs/STEERING.md` maupun berkas `.docx`"*; temuan yang menuntut revisi Steering ditulis sebagai **usulan** yang menunggu persetujuan. Sepanjang Sesi 3 terkumpul **41 keputusan** (`D-31`…`D-71`), **37 di antaranya berdampak ke Steering**, ditambah 14 temuan Fase 1 (`T-1`…`T-14`) dan 40 kontradiksi (`K-1`…`K-40`) yang membuat sejumlah pernyataan di Steering dan BRD tidak lagi benar.

**Pertanyaan (Ronde 9 Q34–Q36):** bentuk pembaruan · perlakuan `STEERING.md` terhadap 20 bab sumber · perlakuan kedua berkas `.docx`.

**Jawaban work owner:**
> "Saya pindahkan file Steering.md beserta format .docx dan file BRD.md beserta format .docx ke folder masing2. Update juga semua perubahan ke dokumen Steering dan BRD ini."
> "Q34 Opsi 1 / Q35 Opsi 1 / Q36 Opsi 3"

**Keputusan akhir:**

| Hal | Keputusan |
|---|---|
| **Kewenangan menyunting** | `docs/Steering/STEERING.md`, `docs/BRD/BRD.md`, dan kedua berkas `.docx` **boleh diperbarui langsung**. Larangan lama dicabut. |
| **Bentuk pembaruan (Q34 Opsi 1)** | Sunting di tempat menyeluruh, **ditambah satu bab baru "Riwayat Revisi"** di depan yang meringkas apa yang berubah dan atas dasar `D-nn` mana |
| **Sumber tunggal (Q35 Opsi 1)** | Bab sumber di `docs/Steering/` menjadi satu-satunya tempat menyunting; `STEERING.md` **dibangun ulang otomatis** oleh `docs/tools/build-steering.js` |
| **Berkas `.docx` (Q36 Opsi 3)** | Dibangun ulang dengan pipeline yang sama seperti `ADR.docx`; berkas `.docx` lama **disimpan sebagai arsip** dengan nama bertanda versi lama |

**Struktur berkas baru, ditetapkan work owner:** `docs/ADR/` (ADR + `ADR.md` + `ADR.docx`) · `docs/BRD/` (`BRD.md` + `.docx`) · `docs/Steering/` (20 bab sumber + `CONTEXT.md` + `00-DECISION-LOG.md` + `STEERING.md` + `.docx`).

**Koreksi atas premis yang saya pakai menyusun Q35 — disampaikan sebelum pekerjaan dimulai.** Saya menyebut `STEERING.md` sebagai "dokumen gabungan dari 20 bab". Pemeriksaan membuktikan hubungannya **tidak seragam**:

| Bagian `STEERING.md` | Hubungan dengan berkas sumber | Bukti |
|---|---|---|
| **Lampiran A, C, D, E, F** | **salinan verbatim** dari `CONTEXT.md`, `01-FRONTEND-ANALYSIS.md`, `18-SKILLS-USAGE-LOG.md`, `19-GAP-EXPORT-DETAIL.md`, `20-DETAIL-KOMITE-DBLINK.md` | teks identik kata per kata |
| **Lampiran B** | **salinan verbatim `00-DECISION-LOG.md` yang berhenti di `D-30`** — 752 baris, 30 entri, sementara sumbernya kini 2.070 baris dan 71 entri | `STEERING.md:2615-3366` vs `00-DECISION-LOG.md:1-752` |
| **Bab 1–26** | **tulisan ulang yang diringkas**, bukan salinan | Bab 21 memuat `R-01`…`R-12` yang sama dengan `16-RISK-ANALYSIS.md` tetapi dalam **prosa 149 baris**, sedangkan sumbernya **tabel 347 baris** |

**Konsekuensi yang mengikuti dari Q35 Opsi 1, dan diterima:** karena bab 1–26 akan dibangun dari berkas sumber, **kata-kata hasil peringkasan pada bab 1–26 tidak dipertahankan** — yang berlaku adalah teks bab sumber. Akibatnya `STEERING.md` menjadi **lebih panjang dan lebih rinci**, dan susunannya mengikuti berkas sumber. Sebagai gantinya:

1. Tidak ada lagi dua versi pernyataan yang sama yang bisa berbeda.
2. **Lampiran B ikut terbarui dengan sendirinya** dari `D-30` menjadi `D-01`…`D-72` — salah satu keusangan terbesar di dokumen itu tertutup tanpa pekerjaan terpisah.
3. Setiap revisi berikutnya cukup menyunting satu tempat.

**Bab baru yang dibuat:** `docs/Steering/21-RIWAYAT-REVISI.md` — berisi ringkasan perubahan v1.0 → v2.0 beserta dasar `D-nn`-nya. Nomor `21` adalah **nomor berkas**, bukan ID modul/risiko/requirement, sehingga tidak melanggar aturan penomoran proyek.

**Yang tetap tidak berubah:** rule XML Pega tetap **baca-saja**. Larangan yang dicabut hanya menyangkut dokumen proyek.

**Dampak ke Steering:** seluruh bab · `README.md` · `docs/tools/`

---

### D-73 · Seluruh Modul Ditiketkan dan Dinamai Menurut Fungsi Bisnis
**Status:** DECIDED — **membatasi `D-41`**, tidak mencabutnya

**Pertanyaan:** Work Owner menyatakan kode modul `F-1`, `F-2`, dan seterusnya **tidak dapat dipahami** saat membaca papan tiket, dan meminta modul dinamai seperti yang dikenalnya sehari-hari — *"Saya ingin seperti contoh: Input Receive Document, Input Register, dll"* — serta meminta **tiket lengkap untuk masing-masing modul**, dengan kode tetap di depan agar rujukan silang ke ADR tetap dapat dicari.

**Keputusan akhir — dua bagian.**

**1. Penamaan.** Setiap folder modul dinamai `<KODE>-<Nama-Fungsi-Bisnis>`, misalnya `B-14-Input-Receive-Document`, `B-2-Input-Register`, `S-7-Dashboard-TAT-dan-KPI`. Nama bisnisnya **tidak dikarang**: diambil dari tahapan `Flow/Register_Flow.xml` dan dari nama harness di export, sehingga yang terbaca Work Owner adalah istilah yang memang dipakai sistem berjalan. Kode tetap di depan supaya rujukan dari ADR, Steering, dan BRD tidak putus.

**2. Cakupan.** **Seluruh 33 modul kini punya tiket** — 102 tiket. Ini melampaui `D-41` Opsi 1 (8 modul penuh + 25 stub), dan karena itu dicatat sebagai keputusan tersendiri alih-alih dijalankan diam-diam.

**Yang TIDAK berubah — dan inilah bagian terpenting entri ini.** `D-41` menolak Opsi 4 dengan alasan yang **masih berlaku sepenuhnya**: menulis acceptance criteria berangka untuk modul yang penghalangnya belum hilang akan menghasilkan kriteria yang mengarang. Aturan itu **tetap ditegakkan**. Yang berubah hanya **bentuk** tiketnya, bukan izin untuk mengarang:

- Modul terhalang mendapat **struktur tiket lengkap** — hasil dan nilai bisnis, lingkup, non-goal, constraint, rollout/rollback, bukti ruang lingkup.
- Bagian acceptance criteria-nya **tidak diisi dengan angka yang dikarang**. Sebagai gantinya, setiap tiket terhalang memuat bagian **"Yang kurang dan siapa yang bisa melengkapinya"** yang menyebut artefak atau keputusan yang hilang **beserta pemiliknya** dan **alasan ia menahan**.
- Satu tiket **menolak menuliskan acceptance criteria sama sekali**: `TKT-S6-003` (persetujuan komite otomatis jam 06:00), karena pertanyaan pokoknya — apakah perilaku itu memang dikehendaki — belum dijawab Work Owner. Menuliskan daftar bercentang di sana akan membuat tiket itu tampak siap dikerjakan padahal lingkupnya belum ada.

**Hasil terukur:** 102 tiket · **30** `ready-for-human` · **72** `needs-info` (**71%**). Kesiapan modul tidak berubah dari `D-41`: **1 PENUH · 23 SEBAGIAN · 9 TERHALANG** (`F-3` terhalang hanya pada bagian identitas). Bertambahnya jumlah tiket **tidak menaikkan kesiapan satu modul pun** — ia hanya membuat lingkup yang selama ini tersembunyi di balik stub menjadi terlihat.

**Temuan yang muncul justru karena penulisan ini dilakukan.** Empat hal berikut tidak akan terlihat bila 25 modul dibiarkan sebagai stub, dan ketiga yang pertama **memintas kendali yang sedang dibangun modul lain**:

1. **`JOBForKomiteKlaimPNC` menjalankan `AutoAcceptKomite` tiap hari 06:00** — menyetujui komite **tanpa pengguna**, di luar jangkauan kontrol berbasis menu (`D-59`).
2. **Dua Service REST masuk menerima persetujuan komite dari sistem lain** — permukaan yang belum pernah diaudit.
3. **18 dari 21 Connect REST ber-`pyUseAuthentication=false`**, satu di antaranya memakai `http://` tanpa TLS untuk data premi.
4. **`GETSELISIHJAM.fnc` dan `GET_WORKING_HOURS@ASMD` adalah dua basis TAT yang saling eksklusif**, ditambah `RETURN 0` yang menyamarkan kegagalan sebagai nol jam.

Bila `B-7` dibangun dengan cermat sementara ketiga jalur pertama dibiarkan, kecermatan itu tidak ada artinya.

**Koreksi angka yang ditemukan saat penulisan:**

| Angka lama | Terverifikasi | Sumber |
|---|---|---|
| `SendEmailNotification` **17 pemanggil** (`verifikasi-bukti-adr.md:2783`) | **15**, dengan kelima belas pemanggilnya disebut namanya | `19-GAP-EXPORT-DETAIL.md:196` |
| Rotasi SMTP **44 lokasi** (`verifikasi-bukti-adr.md:2783`) | **31 lokasi** | `D-40` · `16-RISK-ANALYSIS.md:484` |
| **12 Connect REST keluar** (`06-MODULE-BREAKDOWN.md:63`, `16-RISK-ANALYSIS.md:193`, `BRD §FR-S4`) | **21 berkas** di `Connect REST/` — 12 yang terhitung sejak awal ditambah **9 yang baru ditemukan** | direktori `Connect REST/` (dihitung langsung) |

Ketiganya semula **diusulkan sebagai revisi**. **Work Owner menyetujui penerapannya pada 2026-09-14**, dan ketiganya sudah diterapkan ke `06-MODULE-BREAKDOWN.md`, `16-RISK-ANALYSIS.md`, `BRD.md`, `verifikasi-bukti-adr.md`, `migration-readiness.md`, dan `requirement-summary.md`. Rinciannya — termasuk **dua tempat yang sengaja tidak disunting karena berupa rekaman** — dicatat di `21-RIWAYAT-REVISI.md` §7.

**Dua hal yang masih perlu jawaban Work Owner, dan sengaja tidak saya putuskan sendiri:**

1. **Mana dari 26 harness bernama *Inbox* yang benar-benar inbox peran?** `06-MODULE-BREAKDOWN.md:85` menyebut "~20"; direktori `Harness/` memuat 26, sebagian tampaknya daftar pilihan (`ListDocumentTypeInbox`, `CauseOfLossInbox`, `StatusClaimInbox`). Jumlah layar yang dijanjikan `U-3` bergantung padanya.
2. **Pembagian 48 harness non-Inbox ke `U-4`, `U-5`, dan `U-6` per berkas.** Tanpa itu, jumlah layar ketiga modul frontend adalah perkiraan, bukan komitmen.

**`BACKLOG.md` dicabut.** Kedua puluh lima modul stub di dalamnya kini menjadi modul bertiket, sehingga berkas itu tidak lagi punya isi yang tidak ada di tempat lain. Isinya tidak hilang — ia terserap ke `spec.md` masing-masing modul.

**Dampak ke Steering:** `06-MODULE-BREAKDOWN.md` (angka Connect REST) · `16-RISK-ANALYSIS.md` (R-18) · BRD `§FR-S4` · `docs/verifikasi-bukti-adr.md` §15 · `docs/ticketing/README.md`

---

### D-74 · Fase 6 — Status 40 Usulan Revisi, dan Dua Pernyataan Lama yang Menjadi Usang
**Status:** DECIDED — **melengkapi `D-39`**, dan mengoreksi pernyataan di `D-55`

**Pertanyaan:** `D-39` menetapkan setiap selisih antara dokumen dan bukti dicatat sebagai usulan revisi, dan daftarnya disusun sebagai 40 kontradiksi `K-1`…`K-40` di `verifikasi-bukti-adr.md` §11. Daftar itu **tidak punya penanda status sama sekali**, sehingga tidak ada cara mengetahui mana yang sudah diterapkan tanpa memeriksa satu per satu. Berapa yang sebenarnya masih menggantung?

**Keputusan akhir:** sapuan dijalankan atas keempat puluhnya, hasilnya dicatat di lampiran baru **`23-STATUS-USULAN-REVISI.md`**, dan lampiran itu menjadi tempat status dilacak — **bukan** dengan menyunting §11, yang merupakan rekaman temuan.

| Status | Jumlah |
|---|---:|
| Sudah diterapkan | **11** |
| **Masih menggantung** | **11** |
| Tidak perlu tindakan (bukti mendukung dokumen) | **2** |
| Perlu keputusan, bukan koreksi teks | **11** |
| Belum tuntas diverifikasi — **tidak dinyatakan bersih** | **5** |

**Dua temuan yang lebih berat daripada koreksi angka:**

1. **`K-31` — `NFR-13` berdiri di atas premis yang tidak ada.** Requirement itu dibenarkan oleh `OFFSET 500000`, sementara `OFFSET` **nol kemunculan** di seluruh export. Ia perlu **ditulis ulang**, bukan diperbaiki angkanya.
2. **`K-16` dan `K-17` menyangkut aturan yang menentukan uang** — ambang PA/Travel yang ternyata dipicu **jabatan operator** (bukan lini bisnis), dan batas nilai klaim ≤ TSI yang **mengecualikan PA** tanpa tercatat di BRD. Keduanya ada di kelompok **belum tuntas diverifikasi**, dan layak diprioritaskan di atas koreksi angka mana pun.

**Pernyataan lama yang menjadi usang — dicatat, bukan disunting.** `D-55` berbunyi *"`docs/BRD.md` tidak disentuh; revisinya masuk daftar usulan Fase 6"*. Pernyataan itu benar saat ditulis, tetapi **`D-72` kemudian mencabut larangan menyunting BRD**, dan pencabutan `§21.4` kini benar-benar ada di `BRD.md:1373`. Entri `D-55` **tidak diubah** — aturan proyek melarang menyunting entri lama, dan mengubahnya akan menghapus jejak bahwa keadaannya pernah berbeda. Yang berlaku sekarang adalah entri ini.

**Yang belum diputuskan dan menunggu Work Owner:**

1. **Kesebelas butir "masih menggantung"** — boleh diterapkan seperti tiga koreksi pada `D-73`, atau dipilih sebagian.
2. **Kesebelas butir "perlu keputusan"** — masing-masing menuntut jawaban, bukan suntingan. Empat di antaranya menyangkut keputusan lama yang premisnya berubah: `D-15` (`K-22`), `D-18` (`K-7`), `D-20` (`K-38`), `D-26` (`K-23`).
3. **Definisi istilah "Inbox"** untuk `CONTEXT.md` — diusulkan pada sesi ini karena ternyata kata itu dipakai di seluruh dokumen **tanpa pernah didefinisikan**, sementara `Worklist` dan `Workbasket` sudah punya entri sejak `D-26`. Ini sebab langsung kebingungan penggolongan `U-3` versus `U-6`.

**Dampak ke Steering:** lampiran baru `23-STATUS-USULAN-REVISI.md` · `18-SKILLS-USAGE-LOG.md` · `21-RIWAYAT-REVISI.md`

---

### D-75 · Portal Multi-Entitas — Satu Aplikasi, Satu Database per Entitas
**Status:** DECIDED — **membatasi `D-21`/`ADR-0004`**, dan menambah pertanyaan terbuka pada `D-71`

**Pertanyaan:** Work Owner meminta aplikasi punya **pilihan portal**, sehingga tiap portal menampilkan data dari sumber yang berbeda. Berapa portal, satu database per entitas atau satu database bersama, dan bagaimana laporan, master data, perpindahan portal, serta jejak auditnya?

**Yang ditemukan sebelum pertanyaan dijawab — perilaku ini sudah ada, tetapi tersembunyi:** export memuat **48 perbandingan** terhadap `pxRequestor.pxReqServer` pada **3 hostname** — host dev (34), host entitas Insurtech (13), host entitas Timor-Leste (1). Salah satunya **mengubah ambang komite dari Rp 50.000.000 menjadi 3.500**; selisih itu bukan salah ketik melainkan **mata uang yang berbeda**. Jadi sistem lama sudah melayani beberapa entitas, dengan cara membandingkan nama server — tidak terlihat pengguna, dan tidak pernah tercatat sebagai rancangan di dokumen mana pun.

**Jawaban Work Owner:**

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Berapa portal | **Minimal 4**: Asuransi Sinar Mas · Asuransi Simas Insurtech · Sinarmas Asuransi Syariah · Timor-Leste |
| 2 | Bentuk penyimpanan | **Satu database per entitas** |
| 3 | Laporan | **Per portal** |
| 4 | Master data | **Per portal** |
| 5 | Perpindahan portal | **Tanpa login ulang** |
| 6 | Jejak audit | **Per portal** |

**Keputusan akhir:** **satu aplikasi Go, satu basis kode, empat portal, satu database per entitas**, dengan kelima ketetapan di atas. Dirinci di **`ADR-0030`**. Modul baru **`F-6` Portal & Multi-Sumber Data**, requirement **`FR-F6`**, risiko baru **`R-20`**. Total modul menjadi **34**, bukan 33.

**Jawaban Work Owner mengoreksi bukti kami, bukan sebaliknya.** Dokumen mencatat 3 hostname; Work Owner menyebut 4 portal termasuk **Syariah**. Pemeriksaan membenarkan Work Owner: **`IsServerSyariah` hilang dari export dan tidak tercatat di `19-GAP`** (dirujuk `Data Transform/SetDataEmail-DT.xml:147`, `:411` dan `Activity/SpreadingDataProtection-Act.xml:1294`, `:4153`), sebagaimana `IsDevelopmentServer`. `docs/verifikasi-bukti-adr.md:791` bahkan sudah menyimpulkan **"jumlah hostname sebenarnya bisa lebih dari 3"**. Angka 3 rendah karena **rule pembedanya hilang**. Karena itu kata **"minimal"** diperlakukan harfiah: **daftar portal adalah data, bukan konstanta di kode.**

**Alasan menolak satu database bersama dengan kolom penanda entitas:** pemisahan data akan bergantung pada **tidak adanya satu pun kueri yang lupa menyaring** — kelas kesalahan yang tidak dapat diuji habis. Dengan database terpisah, pemisahan menjadi **sifat struktural**: portal A tidak dapat membaca data portal B karena koneksinya memang berbeda.

**Konsekuensi yang harus disadari, bukan ditemukan belakangan:**

1. **`R-20` — berpindah portal tanpa login ulang berarti satu identitas menjangkau empat database.** Bila kewenangan tidak dinilai ulang saat perpindahan, pengguna dapat melihat data entitas yang bukan haknya. Itu **kebocoran data antar badan hukum**, bukan cacat tampilan.
2. **Empat database harus tetap sekerabat skemanya.** Migrasi skema (`TKT-F2-004`) berjalan empat kali; gagal di salah satunya membuat portal itu tertinggal versi.
3. **Lingkup `F-4` dan `U-6` berlipat** — master ≥29 kelompok kini per portal. Demikian pula beban `S-2`, `S-5`, `S-8`, pencadangan, dan rotasi kredensial.
4. **Laporan konsolidasi lintas portal menjadi tidak mungkin** tanpa keputusan baru. Bila manajemen memintanya kelak, itu keputusan tersendiri.
5. **Portal Syariah belum punya baseline Pega yang utuh.** `SpreadingSyariah_Act` ada, tetapi rule yang menentukan **kapan** ia berlaku hilang. Sampai Tim Pega mengirimkannya, gerbang 1 untuk portal itu **diganti ukuran lain** — sama seperti `F-3` dan `S-5` pada `D-42`, dan **tidak boleh dianggap lulus**.

**Lima pertanyaan yang sengaja dibiarkan terbuka:** jumlah portal yang sebenarnya · perlukah penanda portal di dalam nomor klaim `PNCN.YY.xxxx` karena sequence-nya per database (`D-71`) · sejauh apa aturan bisnis syariah berbeda · satu penyedia identitas untuk keempat entitas atau satu per entitas · mata uang dan pembulatan per portal.

**Dampak ke Steering:** `06-MODULE-BREAKDOWN.md` (modul `F-6`, total 34) · `16-RISK-ANALYSIS.md` (`R-20`) · `09-DATABASE-STRATEGY.md` · `11-SECURITY.md` · `ADR-0004` dibatasi · BRD `FR-F6`

---

### D-76 · Nomor Klaim Tidak Memuat Penanda Portal
**Status:** DECIDED — **menutup pertanyaan terbuka `D-71`** yang dibuka oleh `D-75`

**Pertanyaan:** `D-71` menetapkan nomor klaim `PNCN.YY.xxxx` yang diterbitkan dari sequence database. `D-75` kemudian menetapkan **satu database per entitas**, sehingga sequence-nya pun terpisah per portal. Akibatnya **dua portal dapat menerbitkan nomor klaim yang sama**. Perlukah penanda portal di dalam nomornya?

**Jawaban:** **Tidak perlu penanda portal.**

**Keputusan akhir:** format nomor klaim **tetap `PNCN.YY.xxxx`** persis seperti `D-71`, tanpa penambahan segmen portal. Nomor klaim **unik di dalam satu portal**, dan **tidak dijamin unik antar portal**.

**Konsekuensi yang dicatat supaya menjadi pilihan sadar, bukan kejutan:**

1. **Nomor klaim bukan lagi identitas tunggal di seluruh perusahaan.** Menyebut sebuah nomor klaim tanpa menyebut portalnya menjadi **ambigu**. Setiap kali nomor klaim dipakai di luar konteks satu portal — percakapan telepon, tiket dukungan, surat — portalnya harus ikut disebut.
2. **Dokumen yang keluar ke pihak luar membawa nomor itu.** LOD, PLA, dan DLA dicetak dengan nomor klaim. Nasabah, reasuradur, atau koasuradur yang menerima dokumen dari dua entitas Sinarmas dapat menerima **dua dokumen bernomor sama untuk klaim yang berbeda**. Ini akibat yang terlihat di luar sistem, bukan detail internal.
3. **Keputusan ini sejalan dengan `D-75` butir 3 dan 6**: laporan **per portal** dan jejak audit **per portal**. Selama tidak ada satu pun keluaran yang menggabungkan portal, ambiguitasnya tidak pernah muncul di dalam sistem.
4. **Bila kelak diputuskan ada laporan konsolidasi lintas portal**, keputusan ini **harus ditinjau ulang lebih dulu** — menggabungkan dua portal yang bernomor sama menghasilkan baris yang tidak dapat dibedakan. Menambah penanda portal **setelah** klaim terbit menuntut migrasi data, dan itu jauh lebih mahal daripada memutuskannya sekarang.

**Dampak pada tiket:** `TKT-F2-006` (generator nomor klaim) — salah satu pertanyaan terbukanya tertutup; sequence tetap per database tanpa penanda portal. `TKT-F6-002` — butir "penomoran klaim" pada bagian *Yang kurang* tertutup.

**Dampak ke Steering:** `09-DATABASE-STRATEGY.md` · `ADR-0009` · `ADR-0030` · BRD `FR-F2`

---

### D-77 · Portal — Perilaku Saat Perpindahan Gagal, dan Jumlah Portal
**Status:** DECIDED — melengkapi `D-75`

**Pertanyaan:** `D-75` menetapkan perpindahan portal **tanpa login ulang**, dan `R-20` menandai bahwa satu identitas kini menjangkau empat database milik empat badan hukum. Dua hal belum ditetapkan: apa yang terjadi bila perpindahan **gagal**, dan berapa portal yang harus dilayani.

**Jawaban Work Owner:**

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Bila perpindahan portal gagal | **Tampilkan pesan, dan tetap di portal sebelumnya** |
| 2 | Jumlah portal | **Saat ini 4, dan bisa bertambah** |

**Keputusan akhir:**

1. **Kegagalan perpindahan portal tidak boleh meninggalkan pengguna dalam keadaan menggantung.** Sesi tetap berada di **portal sebelumnya**, dengan pesan yang menjelaskan kegagalannya. Pengguna **tidak** dikeluarkan dari sistem, dan **tidak** dibiarkan di layar tanpa portal aktif.
2. **Empat portal sekarang, dan daftarnya harus dapat bertambah tanpa perubahan kode** — menguatkan `TKT-F6-001` yang sudah memperlakukan daftar portal sebagai **data**.

**Kenapa butir 1 penting untuk `R-20`, bukan sekadar kenyamanan.** Ada tiga kemungkinan perilaku saat perpindahan gagal, dan dua di antaranya berbahaya:

| Perilaku | Akibat |
|---|---|
| Tetap di portal sebelumnya *(dipilih)* | Aman — portal aktif tidak pernah kosong maupun salah |
| Pindah ke portal tujuan meski pemeriksaan gagal | **Kebocoran data antar badan hukum** — persis `R-20` |
| Tidak berada di portal mana pun | Permintaan berikutnya **tidak punya portal aktif**; bila kode jatuh ke koneksi default, akibatnya sama dengan baris di atas |

Keputusan ini karena itu menutup satu jalur kegagalan `R-20`, dan menjadi **acceptance criteria** di `TKT-F6-003`: perpindahan yang ditolak wajib **mempertahankan portal sebelumnya**, bukan mengosongkannya.

**Yang masih terbuka dan tidak tertutup oleh entri ini:** apakah penyedia identitas satu untuk keempat entitas atau satu per entitas · di mana kewenangan portal disimpan · apakah satu pengguna boleh berhak di lebih dari satu portal · sejauh apa aturan bisnis syariah berbeda · mata uang dan pembulatan per portal.

**Dampak ke Steering:** `ADR-0030` · `16-RISK-ANALYSIS.md` (`R-20`) · `TKT-F6-001`, `TKT-F6-003`

---

### D-78 · Satu Identitas Login untuk Keempat Portal
**Status:** DECIDED — melengkapi `D-75` dan `D-77`, menutup satu pertanyaan terbuka `ADR-0030`

**Pertanyaan:** `D-75` menetapkan perpindahan portal tanpa login ulang, dan `D-77` menetapkan mekanismenya berupa **dropdown** tanpa kembali ke halaman login. Yang belum jelas: apakah satu pengguna memakai **user ID yang sama** di keempat entitas, atau tiap entitas punya daftar penggunanya sendiri? Ini menentukan apakah perpindahan tanpa login ulang **mungkin secara teknis**.

**Jawaban:** **Login sama untuk semua entitas.**

**Keputusan akhir:** **satu identitas berlaku di keempat portal.** Pengguna masuk sekali, lalu berpindah portal lewat dropdown tanpa autentikasi ulang. **Portal adalah dimensi cakupan data, bukan batas identitas.**

**Akibat yang menyederhanakan:**

1. Perpindahan portal tidak perlu memetakan empat identitas ke satu sesi — pekerjaan yang jauh lebih besar bila jawabannya sebaliknya.
2. **Autentikasi** (`F-3`, HCC/HCQ) tetap **satu jalur**, tidak berlipat empat. Penghalang `R-14` juga tidak berlipat.
3. Jejak audit dapat menyebut pengguna dengan pengenal yang sama di keempat portal, sehingga penelusuran lintas entitas atas satu orang **tetap mungkin** meski auditnya tersimpan terpisah (`D-75` butir 6).

**Akibat yang memusatkan risiko — dicatat supaya menjadi pilihan sadar:**

**Satu kredensial yang bocor kini menjangkau data empat badan hukum.** Bila tiap entitas punya login sendiri, kebocoran satu akun terbatas pada satu entitas. Dengan keputusan ini, batas itu hilang — yang tersisa sebagai pembatas hanyalah **kewenangan portal per pengguna**. Konsekuensinya, kewenangan portal berhenti menjadi soal kenyamanan dan menjadi **satu-satunya kendali yang memisahkan empat badan hukum**. Ini memperberat `R-20`, bukan meringankannya.

**Satu hal yang menjadi terang justru karena jawaban ini — dan belum diputuskan.** Bila login sama, maka data **"pengguna ini berhak atas portal mana"** bersifat **lintas portal secara alamiah**. Ia tidak dapat disimpan "per portal" seperti master data lain (`D-75` butir 4), karena untuk memeriksa apakah seseorang berhak atas portal B, sistem harus membaca database B **sebelum** orang itu terbukti berhak membacanya — masalah ayam-telur.

Karena itu **kewenangan portal harus tersimpan di satu tempat bersama identitas**, bukan di dalam masing-masing database entitas. Ini **pengecualian yang disengaja** terhadap `D-75` butir 4, dan perlu ditegaskan Work Owner: apakah pengecualian ini diterima, dan di mana tepatnya data itu tinggal.

**Dampak ke Steering:** `ADR-0030` (satu pertanyaan terbuka tertutup) · `16-RISK-ANALYSIS.md` (`R-20` diperberat) · `11-SECURITY.md` · `TKT-F6-003` · `TKT-F3-002`

---

### D-79 · Definisi "Inbox" — Daftar Pekerjaan Milik Pengguna
**Status:** DECIDED — menutup pertanyaan terbuka `U-3` versus `U-6` pada `D-73`

**Pertanyaan:** Kata **"Inbox"** dipakai di seluruh dokumen proyek — README ticketing, Module Breakdown, dan nama modul `U-3` — tetapi **tidak pernah punya entri di `CONTEXT.md`**, sementara `Worklist` dan `Workbasket` sudah terdefinisi sejak `D-26`. Akibatnya penggolongan layar berputar-putar: apakah layar master yang menampilkan data dan dapat dicari boleh disebut Inbox?

**Jawaban:** **Inbox adalah daftar pekerjaan user.**

**Keputusan akhir:** **Inbox** = layar yang menampilkan **daftar pekerjaan milik seorang pengguna** — yakni **Tugas** dari **Worklist** atau **Workbasket** yang menunggu dikerjakan. Definisinya disandarkan pada istilah yang sudah disepakati di `D-26`, bukan pada sumbu baru.

**Empat ciri pembeda terhadap layar daftar biasa:**

| Ciri | Inbox | Layar master |
|---|---|---|
| Isi baris | **pekerjaan** | data acuan |
| Setelah ditindaklanjuti | baris **hilang** | baris **tetap ada** |
| "Hanya milik saya" | **aturan kewenangan** | tidak bermakna |
| Tenggat | ada | tidak ada |

**Layar yang menampilkan data acuan bukan Inbox** — sekalipun dapat dicari, dan sekalipun namanya di sistem lama mengandung kata "Inbox".

**Kenapa ini bukan sekadar soal istilah.** Penggolongan menentukan hal yang nyata: modul mana yang memilikinya (`U-3` atau `U-6`), bergantung pada `B-6` (penugasan) atau `F-4` (master), gelombang berapa, siapa penguji gerbang 2, dan bentuk kendali aksesnya — pada Inbox, penyaringan "hanya pekerjaan saya" **adalah kendali akses**; pada master, kendalinya "siapa boleh mengubah".

**Akibat langsung pada `U-3`.** Sebelumnya `06-MODULE-BREAKDOWN.md` menyebut "~20 inbox berbasis peran", sementara direktori memuat **26 harness bernama *Inbox***. Dengan definisi ini, **tujuh** di antaranya diusulkan pindah ke `U-6` karena bersidik jari layar master — sidik jarinya nyaris identik dengan `MasterRekening` (267–273 KB · 15 rujukan Report Definition), bukan dengan `InboxRegister_Harness` (1.330 KB · 144). Aturan penggolongan di `docs/tools/build-inventaris-harness.js` fungsi `usul()` **sudah menerapkan definisi ini**, sehingga usulannya kini berdiri di atas definisi yang disepakati, bukan sekadar heuristik ukuran berkas.

**Yang tetap terbuka:** **8 usulan bertanda `DUGAAN`** dan **5 bertanda `JANGGAL`** pada inventaris harness masih menunggu koreksi Work Owner. Dua yang paling patut ditinjau: `inboxCompliance_Harness` (145 KB, **nol** Report Definition — antrean kerja tanpa sumber data tidak masuk akal) dan `inboxAnalystDoctor_Harness` (348 KB, menyangkut **data medis** `FR-R2`, sehingga salah golong berakibat pada kendali akses).

**Dampak ke Steering:** `CONTEXT.md` (entri baru) · `06-MODULE-BREAKDOWN.md` `U-3` · `22-INVENTARIS-HARNESS.md` · `docs/ticketing/U-3-*` dan `U-6-*` · `docs/tools/build-inventaris-harness.js`

# Lampiran C — Analisis Pemilihan Frontend

Dokumen pendukung untuk keputusan **D-23**. Berisi perbandingan alternatif frontend modern,
dinilai terhadap batasan nyata project ini, beserta rekomendasi final dan justifikasinya.

> **Koreksi salah satu batasan (2026-09-14) — keputusannya tidak berubah.**
>
> Dokumen ini berkali-kali memakai angka **"18–27 kolom"** sebagai ukuran beratnya kebutuhan grid.
> Verifikasi terhadap export membuktikan angka itu **salah sasaran**: **median kolom sebenarnya 6**,
> dan tiga fitur grid yang biasanya paling mahal — **tambah baris inline, hapus baris inline, dan
> resize kolom** — **tidak dipakai sama sekali** di seluruh 269 section.
>
> Isi analisis di bawah **dibiarkan apa adanya** sebagai catatan pertimbangan saat keputusan
> diambil. Yang berubah adalah **konsekuensinya**, bukan pilihannya:
>
> | Hal | Sebelum | Sesudah |
> |---|---|---|
> | Pilihan framework | React + TypeScript + Vite | **tidak berubah** (`D-23`) |
> | Ukuran modul `U-2` | Besar | **Sedang** |
> | Alasan memilih pustaka tabel kelas berat | kebutuhan inti | **melemah** — kebutuhan grid lebih ringan daripada yang diperkirakan |
>
> Pilihan antara **TanStack Table** dan **AG Grid** karena itu **belum final** dan menjadi
> pertanyaan terbuka di `ADR-0002`. Masalah nyata pada grid ternyata bukan jumlah kolom melainkan
> **3.189 grid yang terikat page list klipboard** Pega dan `pyMaxRecords=500` pada 54 dari 56
> laporan — persoalan paginasi, bukan persoalan lebar tabel.

---

## 1. Batasan yang menentukan (bukan preferensi)

Pilihan frontend di sini **tidak** ditentukan oleh framework mana yang paling canggih, melainkan
oleh lima batasan keras yang sudah diputuskan:

| # | Batasan | Sumber | Konsekuensi |
|---|---|---|---|
| B1 | Tim adalah developer Pega, belum terbiasa Go/JS modern | D-09 | **Learning curve adalah faktor bobot tertinggi.** Framework dengan banyak konsep akan gagal di tangan tim ini. |
| B2 | Tampilan meniru Pega — alur dan tata letak sama | D-13 | Beban terberat ada di **grid padat dan form panjang**, bukan animasi atau halaman konten. |
| B3 | Desktop-first, tapi surveyor pakai tablet/HP | D-12 | Wajib responsive, tapi tidak perlu *mobile-first*. |
| B4 | Deploy ke VM on-premise, bukan Kubernetes | D-08 | **Menambah runtime Node.js di production adalah beban operasional nyata.** |
| B5 | Aplikasi internal di balik login, tanpa akses publik | D-07 | **SEO dan SSR tidak punya nilai sama sekali di sini.** |

Ditambah fakta terukur dari source:
- **74 layar**, **269 section**, **268 di antaranya memakai repeat/grid**
- Inbox utama punya **18–27 kolom** per tabel
- **56 laporan**, export PDF/Excel/CSV **dibuat di sisi Go** (D-11) — bukan tugas frontend
- Volume data besar (D-10) → grid wajib mendukung **paginasi server-side**, bukan memuat semua baris

> **Implikasi B5 yang sering terlewat:** karena SEO dan SSR tidak bernilai, keunggulan utama
> Next.js dan Nuxt hilang seluruhnya. Yang tersisa dari keduanya justru kerugiannya (B4).

---

## 2. Perbandingan alternatif

### React (SPA, dengan Vite)

**Kelebihan** — Ekosistem terbesar di dunia frontend. Untuk kebutuhan grid berat, pilihannya
paling matang: AG Grid dan TanGrid/TanStack Table adalah standar industri untuk tabel puluhan
kolom dengan paginasi server-side. Materi belajar berlimpah, termasuk dalam bahasa Indonesia.
Pasar tenaga kerja paling besar — paling mudah mencari pengganti bila ada developer keluar.

**Kekurangan** — Tidak opinionated. Tidak ada cara resmi untuk routing, state, form, maupun
pengambilan data; semuanya keputusan tim. Untuk tim di B1, "banyak jalan menuju Roma" berubah
menjadi **kode yang berbeda gaya di setiap layar**. Konsep yang harus dikuasai relatif banyak:
JSX, hooks, dependency array, aturan re-render, memoization. Kesalahan pada `useEffect` dan
dependency array adalah sumber bug paling umum bagi pemula.

**Performa** — Sangat baik untuk skala ini. Virtual DOM lebih lambat dibanding pendekatan
kompilasi (Svelte/Solid), tapi pada 200–300 user dengan grid ber-paginasi, perbedaannya tidak
akan terasa.

**Maintenance** — Baik bila konvensi ditegakkan; buruk bila tidak. Sangat bergantung pada
disiplin Steering.

**Kecocokan dengan Golang** — Sempurna. SPA murni, Go menyajikan JSON dan berkas statis.

**Skalabilitas** — Terbukti pada aplikasi jauh lebih besar dari ini.

**Ekosistem** — Terbaik di antara semua kandidat.

**Learning curve** — Menengah. Mudah dimulai, sulit dikuasai dengan benar.

---

### Next.js

**Kelebihan** — Kerangka lengkap di atas React: routing berbasis berkas, SSR/SSG, optimasi
gambar, server actions.

**Kekurangan** — **Seluruh keunggulan utamanya tidak relevan di sini** (B5: aplikasi internal
di balik login, tanpa SEO). Yang tersisa adalah biayanya: **butuh Node.js berjalan permanen di
VM production** (B4), model App Router dengan Server Components menambah satu lapisan konsep
berat (server vs client component, caching, streaming) yang justru paling berbahaya bagi tim B1.

**Kecocokan dengan Golang** — Buruk secara arsitektur. Kita jadi punya **dua backend runtime**
(Go dan Node) untuk satu aplikasi — dua hal yang harus di-deploy, dimonitor, ditambal
keamanannya, dan di-restart.

**Alasan tidak dipilih** — Membayar seluruh biaya SSR tanpa memperoleh satu pun manfaatnya.

---

### Vue 3 (SPA, dengan Vite)

**Kelebihan** — Learning curve paling landai di antara framework yang benar-benar matang.
Sintaks template adalah **HTML dengan direktif** (`v-if`, `v-for`, `v-model`) — secara konsep
**paling dekat dengan cara Pega Section bekerja**, sehingga tim B1 membaca kode baru dengan
model mental yang sudah mereka punya. Single-File Component menyatukan template, logika, dan
gaya dalam satu berkas yang mudah ditelusuri. Punya **pustaka resmi** untuk routing (Vue Router)
dan state (Pinia) — mengurangi keputusan yang harus dibuat tim. `v-model` menjadikan form dua
arah jauh lebih ringkas dibanding React, dan **B2 adalah aplikasi yang didominasi form**.

**Kekurangan** — Ekosistem lebih kecil dari React. Pasar tenaga kerja di Indonesia lebih kecil
walau tetap sehat. Beberapa pustaka niche hanya tersedia untuk React.

**Performa** — Setara React, pada beberapa kasus sedikit lebih baik. Lebih dari cukup.

**Maintenance** — Sangat baik. Struktur SFC memaksa keseragaman, dan pustaka resmi mencegah
perpecahan gaya antar developer.

**Kecocokan dengan Golang** — Sempurna. SPA murni.

**Skalabilitas** — Terbukti pada aplikasi enterprise berskala besar.

**Ekosistem** — Cukup. Untuk kebutuhan spesifik kita, **PrimeVue** menyediakan DataTable dengan
paginasi server-side, filter per kolom, kolom beku, kolom yang bisa diatur ulang, seleksi baris,
dan penyuntingan sel — yaitu **persis daftar kebutuhan grid Pega**. Element Plus dan Naive UI
adalah alternatif setara.

**Learning curve** — Paling landai. Ini keunggulan terbesarnya untuk project ini.

---

### Nuxt

**Kelebihan** — Setara Next.js untuk ekosistem Vue.

**Kekurangan** — Persis sama dengan Next.js: manfaat SSR tidak terpakai (B5), tapi kewajiban
menjalankan Node.js di VM tetap ada (B4).

**Alasan tidak dipilih** — Sama dengan Next.js.

---

### Angular

**Kelebihan** — Paling opinionated dari semua kandidat, dan itu **sebenarnya cocok** dengan
kebutuhan B1 akan struktur yang seragam. Semuanya resmi dan satu jalan: routing, HTTP client,
form (reactive forms sangat kuat untuk form panjang seperti milik kita), validasi, testing,
dependency injection. TypeScript wajib sejak awal. Terbukti pada aplikasi *line-of-business*
besar di lingkungan korporat. Rilis punya jadwal panjang yang jelas.

**Kekurangan** — **Learning curve paling curam.** Tim B1 harus menyerap Dependency Injection,
dekorator, modul (atau standalone component), dan terutama **RxJS** — pemrograman reaktif
berbasis stream yang merupakan hambatan besar bagi developer yang belum pernah menyentuhnya.
Kode jauh lebih panjang untuk hasil yang sama. Ukuran bundel paling besar.

**Performa** — Baik, tapi paling berat di antara kandidat.

**Maintenance** — Sangat baik dalam jangka panjang, **jika** tim berhasil melewati fase belajar.

**Kecocokan dengan Golang** — Sempurna. SPA murni.

**Alasan tidak dipilih** — Kekuatannya (struktur ketat) bisa kita peroleh dari Steering yang
preskriptif, sedangkan kelemahannya (RxJS + DI + verbositas) langsung menghantam batasan
terkuat kita, yaitu B1. Risiko tim tidak pernah benar-benar produktif terlalu besar.

---

### Svelte / SvelteKit

**Kelebihan** — Model mental paling sederhana dari semuanya; sintaks paling sedikit boilerplate.
Dikompilasi, tanpa Virtual DOM — bundel terkecil dan performa runtime terbaik.

**Kekurangan** — Ekosistem paling kecil di antara kandidat arus utama. **Pilihan pustaka
DataTable enterprise sangat terbatas** — padahal itu kebutuhan inti kita (18–27 kolom, paginasi
server-side, filter per kolom). Pasar tenaga kerja di Indonesia sempit. SvelteKit juga membawa
persoalan runtime Node yang sama seperti Next/Nuxt bila dipakai penuh.

**Alasan tidak dipilih** — Kesederhanaannya menarik untuk B1, tapi kami akan **membangun sendiri
komponen grid** yang di ekosistem lain sudah tersedia matang. Untuk 74 layar dengan tim yang
masih belajar, itu risiko yang tidak sepadan.

---

### SolidJS

**Kelebihan** — Performa terbaik secara benchmark. Reaktivitas granular tanpa Virtual DOM.
Sintaks mirip React sehingga materi React sebagian bisa dipakai.

**Kekurangan** — Ekosistem paling kecil dan komunitas paling sedikit. Nyaris tidak ada pustaka
komponen enterprise yang matang. Pasar tenaga kerja sangat sempit.

**Alasan tidak dipilih** — Terlalu berisiko untuk aplikasi bisnis inti berumur panjang yang
dikerjakan tim yang sedang belajar. Keunggulan performanya tidak menjawab masalah nyata kita —
hambatan kita adalah volume data di sisi database, bukan kecepatan render.

---

### Alternatif lain: Go + `templ` + HTMX (server-rendered)

Layak dipertimbangkan serius, bukan sekadar pelengkap daftar.

**Kelebihan** — **Satu bahasa untuk seluruh aplikasi**, dan itu menjawab B1 secara paling
langsung: tim hanya perlu belajar Go, bukan Go *dan* satu ekosistem JavaScript. Satu artefak
deploy, satu proses yang dimonitor — sangat cocok dengan B4. Secara konseptual **paling dekat
dengan Pega**, yang juga merender HTML di sisi server. Tanpa proses build frontend, tanpa
`node_modules`.

**Kekurangan** — Titik lemahnya persis di tempat aplikasi kita paling berat. Grid dengan
penyuntingan inline, baris yang bisa ditambah/dihapus, dan **perhitungan langsung di layar**
(total spreading wajib 100%, nilai settlement yang berubah seketika) memerlukan interaksi sisi
klien yang kaya. Dengan HTMX, setiap interaksi kecil menjadi perjalanan bolak-balik ke server —
pada form panjang milik Pega ini akan terasa lambat dan rapuh, terutama untuk surveyor di
jaringan seluler (B3). Menambal ini dengan Alpine.js berarti tetap menulis JavaScript, hanya
dengan alat yang jauh lebih terbatas.

**Alasan tidak dipilih** — Tepat menyelesaikan masalah tim, tapi tepat gagal pada karakter UI
yang harus kita tiru (B2). Bila UI boleh disederhanakan, opsi ini akan menjadi rekomendasi
utama — namun D-13 menutup kemungkinan itu.

---

## 3. Tabel ringkas

Bobot mencerminkan batasan project, bukan mutu framework secara umum.

| Kriteria | Bobot | React | Next.js | **Vue 3** | Nuxt | Angular | Svelte | SolidJS | Go+HTMX |
|---|---|---|---|---|---|---|---|---|---|
| Learning curve untuk tim eks-Pega | ★★★★★ | 3 | 2 | **5** | 2 | 1 | 4 | 3 | 5 |
| Kesiapan grid/tabel enterprise | ★★★★★ | 5 | 5 | **5** | 5 | 5 | 2 | 1 | 2 |
| Kemudahan form panjang & kompleks | ★★★★☆ | 3 | 3 | **5** | 5 | 5 | 4 | 3 | 2 |
| Kecocokan dengan backend Go | ★★★★☆ | 5 | 2 | **5** | 2 | 5 | 5 | 5 | 5 |
| Kesederhanaan operasional di VM | ★★★★☆ | 5 | 1 | **5** | 1 | 5 | 5 | 5 | 5 |
| Keseragaman kode / anti-perpecahan gaya | ★★★★☆ | 2 | 3 | **4** | 4 | 5 | 3 | 2 | 4 |
| Ekosistem & pustaka | ★★★☆☆ | 5 | 5 | **4** | 4 | 4 | 2 | 1 | 3 |
| Ketersediaan SDM di Indonesia | ★★★☆☆ | 5 | 5 | **4** | 3 | 3 | 2 | 1 | 3 |
| Performa untuk 200–300 user | ★★☆☆☆ | 4 | 4 | **4** | 4 | 3 | 5 | 5 | 4 |
| Nilai SSR/SEO | — | — | 0 | — | 0 | — | — | — | — |
| Responsif untuk surveyor (B3) | ★★★☆☆ | 5 | 5 | **5** | 5 | 4 | 5 | 5 | 3 |

---

## 4. Rekomendasi final

### **Vue 3 + TypeScript + Vite + PrimeVue**, sebagai SPA murni yang disajikan oleh binary Go.

**Justifikasi teknis:**

1. **Menjawab batasan terkuat secara langsung (B1).** Learning curve paling landai di antara
   opsi yang matang. Sintaks template Vue adalah HTML dengan direktif — model mental yang sudah
   dimiliki tim dari Pega Section. Ini menghemat berbulan-bulan produktivitas dibanding React
   (hooks, re-render) atau Angular (RxJS, DI).

2. **Kuat tepat di titik terberat aplikasi (B2).** Aplikasi ini didominasi form panjang dan grid
   padat. `v-model` membuat form dua arah jauh lebih ringkas dan lebih sulit disalahgunakan
   dibanding pola controlled component React. PrimeVue DataTable menyediakan paginasi
   server-side, filter per kolom, kolom beku, dan penyuntingan sel — persis kebutuhan grid
   18–27 kolom kita, tanpa membangun sendiri.

3. **Mengurangi keputusan yang harus dibuat tim.** Vue Router dan Pinia adalah pustaka resmi.
   Tim tidak perlu memilih di antara lima pustaka routing dan tujuh pustaka state. Untuk tim
   yang sedang belajar, **lebih sedikit keputusan berarti lebih sedikit ketidakseragaman** —
   dan ini melengkapi D-09 yang menuntut Steering preskriptif.

4. **Menjaga kesederhanaan operasional (B4, B8).** SPA murni dikompilasi menjadi berkas statis
   yang bisa **disajikan langsung oleh binary Go**. Di VM production hanya ada **satu proses**
   untuk di-deploy dan dimonitor. Next.js dan Nuxt akan menambah runtime Node.js kedua tanpa
   memberi manfaat apa pun (B5).

5. **TypeScript wajib, bukan opsional.** Pada 74 layar yang dikerjakan tim yang sedang belajar,
   TypeScript menangkap kesalahan saat kompilasi yang jika tidak akan lolos ke production.
   Tambahan beban belajarnya nyata namun terbayar berkali lipat, dan tipe dapat **dihasilkan
   otomatis dari kontrak API Go** sehingga backend dan frontend tidak pernah berbeda persepsi.

**Alternatif kedua: React + TypeScript + Vite + TanStack Table / AG Grid.**
Dipilih **jika** faktor penentu bergeser ke ketersediaan SDM jangka panjang. Ekosistem dan
pasar kerja React lebih besar, dan itu keunggulan nyata untuk aplikasi berumur 10 tahun. Yang
dikorbankan: learning curve lebih curam dan risiko ketidakseragaman kode lebih tinggi — yang
harus ditebus dengan Coding Standards dan code review yang jauh lebih ketat.

**Yang secara tegas tidak direkomendasikan:** Next.js dan Nuxt (membayar biaya SSR tanpa
manfaatnya, serta menambah runtime kedua di VM), Svelte dan SolidJS (ekosistem grid enterprise
belum memadai untuk kebutuhan inti kita).

# Lampiran D — Catatan Penggunaan Matt Pocock Skills

Dokumentasi setiap skill yang dipakai: tahapan, alasan pemilihan, tujuan, hasil, dan dampaknya
terhadap keputusan Steering.

---

## Catatan awal: ketersediaan skill di environment ini

Diperiksa pada 2026-09-07 di `~/.claude/skills`, `~/.claude/plugins`, dan registry skill sesi.

**Tidak terpasang** — sehingga tidak dapat dipanggil meskipun disebut dalam instruksi:
`/setup-matt-pocock-skills`, `/ask-matt`, `/grill-me`, `/grill-with-docs`, `/to-spec`,
`/to-tickets`, `/triage`, `/implement`, `/wayfinder`, `/improve-codebase-architecture`,
`/teach`, `/handoff`.

Karena `/setup-matt-pocock-skills` tidak ada, instruksi memanggilnya lebih dulu tidak dapat
dijalankan. Hal ini disampaikan ke pemilik project sebelum pekerjaan dimulai.

**Terpasang dan tersedia:**
`mattpocock-skills:domain-modeling` · `grilling` · `codebase-design` · `research` · `tdd` ·
`prototype` · `diagnosing-bugs` · `code-review` · `resolving-merge-conflicts` · `wizard` ·
`writing-for-agents`

Dari yang tersedia, **tiga dipakai** karena relevan dengan tahapan yang dikerjakan. Sisanya
sengaja tidak dipakai — alasannya di bagian akhir dokumen ini.

---

## 1. `mattpocock-skills:domain-modeling`

### Tahapan penggunaan
Tahap 2 — Deep Discovery, setelah pembacaan source selesai dan sebelum Domain Model disusun.

### Alasan memilih skill ini
Setelah membaca 902 activity dan 652 rule SQL, masalah terbesar yang muncul **bukan** kerumitan
teknis, melainkan **bahasa domain yang kacau**. Ditemukan tiga gejala konkret:

1. Istilah yang menyesatkan — `AdjustmentList` ternyata berisi nilai penyelesaian, bukan proses
   loss adjusting.
2. Istilah yang bertabrakan — `Object` dipakai untuk objek pertanggungan, bertabrakan dengan
   makna pemrograman.
3. Empat konsep status bernama mirip (`StatusWork`, `StatusClaim`, `ClaimStatus`, `StatusPosisi`)
   yang tidak jelas apakah benar-benar berbeda.

Skill ini secara khusus menangani hal itu: mempertajam bahasa yang kabur, menantang istilah,
menyilangkan pernyataan dengan kode, dan menuliskan glossary saat istilah terselesaikan.

### Tujuan penggunaan
Menghasilkan **ubiquitous language** yang dipakai konsisten di nama tabel, struct Go, endpoint
API, label UI, dan dokumentasi — agar kekacauan penamaan sistem lama tidak ikut berpindah.

### Yang dilakukan mengikuti skill

**Mempertajam istilah kabur.** Untuk setiap istilah yang meragukan, dibuat usulan istilah
kanonikal beserta alasannya, lalu diuji ke pemilik bisnis. Hasilnya: `Object` → **Objek
Pertanggungan**, `Adjustment` → **Settlement Line**.

**Menyilangkan pernyataan dengan kode.** Empat konsep status tidak diterima begitu saja sebagai
duplikasi maupun sebagai konsep berbeda. Dilacak dulu ke source: `StatusWork` berisi
`Resolved-Completed`/`Resolved-Rejected`, `StatusClaim` berisi kode `1142`–`1151` yang di-lookup
ke `V_STS_CLAIM.LSC_ID`, `StatusPosisi` berisi `On Progress`/`Done` di `GCNM_PROGRESS_CLAIM`.
Baru setelah bukti terkumpul, pertanyaannya diajukan ke pemilik bisnis.

**Menulis glossary saat itu juga.** `CONTEXT.md` ditulis segera setelah pemilik bisnis
menjelaskan arti singkatan, tidak ditunda sampai akhir — sesuai anjuran skill agar istilah tidak
menguap.

**Menjaga `CONTEXT.md` bebas detail implementasi.** Tidak ada nama tabel, tipe data, maupun
keputusan teknis di dalamnya. Semua itu ditempatkan di dokumen Steering lain.

### Hasil yang diperoleh

**`CONTEXT.md`** — glossary lengkap dengan penanda asal setiap istilah:
`[BISNIS]` dikonfirmasi pemilik bisnis · `[KODE]` disimpulkan dari source · `[TERBUKA]` belum pasti.

**Arti 12 singkatan yang sebelumnya tidak diketahui**, dikonfirmasi langsung: RCL, PUCL, PLA,
DLA, Pre-DLA, LOD, TKA, KBRU, BPPDAN, MBU/Non-MBU, SPK, OS.

**Temuan paling menentukan: Non-MBU = PNC.** Aplikasi "Claim PNC" ternyata menangani klaim
**Non-Motor**. Ini mengoreksi asumsi awal dan mengubah cara bounded context dinamai.

**Empat konsep status terbukti memang berbeda**, bukan duplikasi — dikonfirmasi pemilik bisnis
setelah bukti dari kode disajikan.

### Dampak terhadap keputusan Steering

| Dampak | Ke mana |
|---|---|
| Ubiquitous language menjadi mengikat di kode, API, dan UI | D-19, Coding Standards §4.1 |
| `Object` → Objek Pertanggungan, `Adjustment` → Settlement Line | Domain Model, Database Strategy |
| Empat status dimodelkan terpisah dengan nama yang tidak bisa tertukar | D-18, Domain Model §3 |
| Alias kolom warisan dinamai ulang menyeluruh | D-19, Current Architecture §4.2 |
| Urutan PLA → Pre-DLA → DLA menjadi jelas, membentuk modul B-9 | Module Breakdown, Domain Model §6 |
| Arti `1142`–`1151` yang masih hilang diangkat sebagai risiko | R-06 |

**Tanpa skill ini**, Domain Model kemungkinan besar akan menyalin penamaan Pega apa adanya —
termasuk `Adjustment` yang salah arti dan empat status yang mudah tertukar. Kekacauan yang justru
menjadi alasan utama migrasi akan ikut berpindah ke sistem baru.

---

## 2. `mattpocock-skills:grilling`

### Tahapan penggunaan
Tahap 2 — Deep Discovery, setelah keputusan awal terkumpul dan sebelum Steering ditulis.

### Alasan memilih skill ini
Instruksi project secara eksplisit meminta grilling. Lebih dari itu, beberapa jawaban discovery
awal **saling bertentangan tanpa disadari** — dan bila dituliskan apa adanya ke dalam Steering,
tim implementasi akan menemukan kontradiksinya di tengah pengerjaan.

Skill ini menyediakan disiplin yang tepat: memetakan keputusan sebagai pohon, mengerjakan
*frontier* (keputusan yang prasyaratnya sudah selesai) per ronde, menyertakan rekomendasi di
setiap pertanyaan, dan **mencari fakta sendiri** alih-alih menanyakannya ke pengguna.

### Tujuan penggunaan
Menekan setiap asumsi arsitektur sampai patah atau terbukti kuat, sebelum menjadi blueprint yang
diikuti tim selama berbulan-bulan.

### Yang dilakukan mengikuti skill

**Fakta dicari sendiri, keputusan diserahkan ke pemilik project.** Sebelum menanyakan strategi
dialek SQL, seluruh 652 rule SQL dianalisis dan dikategorikan ke portabel / terjemahan ringan /
tidak portabel, lengkap dengan volumenya. Pertanyaan diajukan dengan angka, bukan dengan
"bagaimana menurut Bapak".

**Setiap pertanyaan disertai rekomendasi.** Sesuai format skill, setiap pertanyaan punya
jawaban yang saya rekomendasikan beserta alasannya — termasuk ketika rekomendasi itu akhirnya
tidak dipilih.

**Frontier dikerjakan per ronde.** Pertanyaan yang jawabannya bergantung pada pertanyaan lain
yang masih terbuka **ditunda ke ronde berikutnya**, tidak diajukan bersamaan. Contoh: versi
PostgreSQL baru ditanyakan setelah strategi SQL portabel diputuskan, karena barulah versi
menjadi relevan.

**Kontradiksi diangkat, tidak diserap diam-diam.** Ketika Q1 memilih SQL portabel sementara Q3
menetapkan `POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL` yang khas Oracle, pertentangannya disampaikan
terbuka beserta usulan penyelesaiannya.

### Hasil yang diperoleh

**Kontradiksi SQL portabel versus sequence Oracle terungkap** dan diselesaikan sebagai
pengecualian terkelola yang diisolasi di satu berkas (D-22 §3.1).

**Prasyarat tersembunyi ditemukan.** Keputusan "SQL portabel" ternyata **hanya bisa dijalankan
bila PostgreSQL 17+**, karena 222 pemanggilan `JSON_VALUE`/`JSON_TABLE` tidak punya padanan di
PostgreSQL 16 ke bawah. Tanpa grilling, keputusan ini akan tertulis di Steering sebagai janji
yang tidak bisa ditepati, dan baru ketahuan saat implementasi.

**Blocker sesungguhnya teridentifikasi.** Yang paling menghalangi portabilitas bukan `ROWNUM`
(yang sepele — 35 rule sudah memakai `FETCH FIRST`), melainkan **64 pemakaian DB Link ke 6
database lain**, yang tidak punya padanan di PostgreSQL sama sekali. Ini menghasilkan D-25 dan
risiko R-03.

**Lubang di keputusan Strangler Fig tertutup.** Pertanyaan "bagaimana dua sistem berbagi data"
belum terjawab saat D-05 diambil. Grilling memaksanya terjawab, menghasilkan D-21 beserta aturan
"satu tabel hanya boleh ditulis satu sistem", dan mengungkap bahwa **116 query membaca tabel
klaim milik engine Pega** yang harus digantikan tabel baru.

**Ukuran pekerjaan versus jadwal terkuantifikasi.** Pertanyaan timeline menghasilkan angka
konkret, keberatan yang disampaikan dengan bukti, dan — setelah pemilik project menegaskan
targetnya — pencatatannya sebagai R-05 dengan dasar angka yang jelas untuk keputusan berikutnya.

**Empat pertanyaan NFR yang awalnya terlalu abstrak** diperjelas setelah pemilik project meminta
klarifikasi, lalu dipecah menjadi pertanyaan konkret per topik. Ini menghasilkan D-27 (24/7),
D-28 (audit trail), dan D-29 (backup).

### Dampak terhadap keputusan Steering

| Dampak | Ke mana |
|---|---|
| Prasyarat PostgreSQL 17+ menjadi mengikat | D-24, Database Strategy §2 |
| Tiga pengecualian portabilitas diakui dan dikelola | D-20, Database Strategy §3 |
| DB Link diganti API + inventaris 6 sistem | D-25, API Strategy §8.4 |
| Aturan kepemilikan tabel selama masa paralel | D-21, Migration Strategy P-1 |
| Tabel penugasan baru + konsep Worklist/Workbasket dipertahankan | D-26 |
| Ketersediaan 24/7 menghasilkan syarat stateless, rolling deployment, migrasi backward-compatible | D-27, Deployment §2 |
| Jejak audit dirancang sejak awal, bukan ditambal | D-28, Database Strategy §8 |
| Risiko jadwal terdokumentasi dengan angka | R-05 |

**Tanpa skill ini**, Steering akan memuat janji "SQL portabel" tanpa prasyarat versi, strategi
Strangler Fig tanpa aturan kepemilikan data, dan target waktu tanpa dasar ukuran.

---

## 3. `mattpocock-skills:codebase-design`

### Tahapan penggunaan
Tahap 3 — penyusunan Future Architecture, Module Breakdown, dan Technical Strategy.

### Alasan memilih skill ini
Dua hal membuat skill ini tepat di titik ini:

1. **Tim adalah developer Pega yang belum terbiasa Go maupun JS modern (D-09).** Mereka
   membutuhkan struktur yang preskriptif dan kosakata bersama untuk membahas desain — bukan
   kebebasan.
2. **Sistem lama tidak punya batas modul sama sekali.** Logika bisnis tersebar di activity, SQL,
   dan stored procedure sekaligus, sehingga satu aturan harus dicari di tiga tempat.

Skill ini menyediakan kosakata yang tepat — module, interface, depth, seam, adapter, leverage,
locality — dan prinsip yang mencegah kesalahan paling umum saat merancang batas.

### Tujuan penggunaan
Menetapkan batas modul dan letak seam berdasarkan alasan yang dapat diuji, bukan berdasarkan
selera; serta menghasilkan struktur yang cukup preskriptif untuk dijalankan tim yang sedang
belajar.

### Yang dilakukan mengikuti skill

**Prinsip "satu adapter berarti seam hipotetis, dua adapter berarti seam nyata" diterapkan
sebagai penyaring.** Setiap seam yang diusulkan harus punya minimal dua adapter nyata. Ini
mencegah kelebihan abstraksi yang justru berbahaya bagi tim di D-09.

Contoh penerapannya: meskipun D-20 menetapkan satu set SQL portabel — sehingga seolah hanya ada
satu adapter database — seam Repository tetap dibuat karena adapter keduanya adalah **fake untuk
pengujian**. Itulah yang membuat aturan bisnis dapat diuji tanpa database.

**Uji deletion dipakai untuk membenarkan modul.** Untuk modul Registrasi Klaim: bila dihapus,
kompleksitas 137 step akan muncul kembali di setiap pemanggil — API registrasi, impor batch,
koreksi data. Modul ini terbukti membayar dirinya sendiri.

**Depth diukur sebagai leverage, bukan sebagai jumlah baris.** Modul Registrasi Klaim
menyembunyikan 137 step di balik satu pemanggilan; modul Spreading menyembunyikan aturan yang
kini tersebar di 45 activity.

**"Interface adalah permukaan pengujian" menjadi dasar Testing Strategy.** Karena pemanggil dan
test melewati seam yang sama, uji aturan bisnis dapat berjalan tanpa database, tanpa jaringan,
dan tanpa berkas.

**Interface dideklarasikan di paket yang memakainya**, bukan di paket yang mengimplementasikannya
— agar seam berada di tempat yang benar dan arah ketergantungan tetap ke dalam.

### Hasil yang diperoleh

**Enam seam yang dibenarkan**, masing-masing dengan minimal dua adapter: Repository, Clock,
DocumentStore, ExternalSystem, Identity, Notifier.

**Seam Clock adalah temuan yang paling berdampak.** Skill ini mendorong pertanyaan "apa yang
sebenarnya bervariasi di sini", dan jawabannya mengungkap bahwa penanganan waktu adalah sumber
bug tersembunyi di sistem lama: `+7 jam` ditambahkan manual di puluhan tempat, dan satu tempat
yang lupa memanggilnya menggeser tanggal tanpa terdeteksi. Seam Clock menjadikannya satu tempat,
sekaligus membuat seluruh aturan tanggal dapat diuji secara deterministik.

**Seam Notifier dirumuskan berbasis peristiwa domain**, bukan `SendEmail(to, subject, body)`.
Perbedaan ini yang menjadikan penerima notifikasi sebagai konfigurasi (D-15) — sekaligus menutup
kemungkinan blok "TESTING" yang menimpa email produksi terulang kembali.

**Delapan modul domain yang dalam** dengan interface kecil, masing-masing disertai ukuran dari
sistem lama sebagai bukti leverage-nya.

**Aturan ketergantungan antar lapisan** yang ditegakkan otomatis oleh `depguard`, bukan hanya
oleh kesepakatan.

**Konsep leverage dan locality diterapkan ke frontend.** 268 dari 269 section memakai pola grid
yang sama — satu komponen `DataTable` baku dipakai ratusan kali (leverage) dan perbaikan bug di
satu tempat memperbaiki seluruh layar (locality). Ini menjadikan U-2 sebagai investasi frontend
yang paling menentukan.

### Dampak terhadap keputusan Steering

| Dampak | Ke mana |
|---|---|
| Enam seam beserta pembenarannya | Future Architecture §3 |
| Delapan modul domain dalam dengan ukuran leverage | Future Architecture §4 |
| Seam Clock menjadi modul fondasi F-5 | Module Breakdown §1 |
| Struktur folder mengikuti arah ketergantungan | Technical Strategy §2 |
| Aturan ketergantungan ditegakkan `depguard` | Technical Strategy §6 |
| Uji aturan bisnis lewat seam, tanpa infrastruktur | Testing Strategy §3 |
| Pustaka komponen baku U-2 sebagai prioritas frontend | Module Breakdown §4 |
| Daftar tegas yang **tidak** dibangun | Future Architecture §6 |

**Tanpa skill ini**, batas modul kemungkinan besar dibagi menurut lapisan teknis (semua handler,
semua service, semua repository) — pembagian yang justru menyebarkan satu aturan bisnis ke
banyak tempat, mengulang persis masalah sistem lama.

---

## Skill yang tersedia tetapi sengaja tidak dipakai

Instruksi project meminta skill dipakai hanya bila relevan, bukan dipanggil karena tersedia.

| Skill | Alasan tidak dipakai |
|---|---|
| `tdd` | Tahap ini melarang penulisan kode. Skill ini akan relevan saat implementasi dimulai — dan Testing Strategy sudah disiapkan untuk menerimanya |
| `prototype` | Tidak ada pertanyaan desain yang membutuhkan prototipe. Pertanyaan yang muncul terjawab dengan membaca source dan bertanya ke pemilik bisnis |
| `diagnosing-bugs` | Tidak ada bug yang sedang didiagnosis. Utang teknis yang ditemukan dicatat sebagai temuan, bukan didiagnosis |
| `code-review` | Tidak ada perubahan kode untuk di-review. Akan relevan saat implementasi berjalan |
| `research` | Seluruh fakta tersedia di source dalam repository. Tidak ada yang perlu diteliti dari sumber eksternal |
| `resolving-merge-conflicts` | Tidak ada konflik merge; project ini bahkan belum berupa repository git |
| `wizard` | Tidak ada langkah provisioning yang harus dipandu manusia pada tahap ini |
| `writing-for-agents` | Dokumen Steering ditujukan untuk dibaca manusia, bukan sebagai skill atau `AGENTS.md` |

---

## Sesi 3 — Penyusunan ADR dan Ticketing (2026-09-08 … 2026-09-14)

Sesi ini menurunkan Steering dan BRD yang sudah disetujui menjadi dua artefak yang mengikat:
ADR di `docs/ADR/` dan tiket pekerjaan di `docs/ticketing/`.

## Ketersediaan skill pada sesi ini

| Skill | Status | Keterangan |
|---|---|---|
| `mattpocock-skills:grilling` | **terpasang, dipakai** | Fase 2 |
| `mattpocock-skills:domain-modeling` | **terpasang, dipakai** | Fase 3–4 |
| `mattpocock-skills:writing-for-agents` | terpasang | dievaluasi, dipakai terbatas |
| `mattpocock-skills:codebase-design` | terpasang | dievaluasi untuk batas modul |
| `/setup-matt-pocock-skills` | **tidak terpasang** | dievaluasi, tidak dijalankan |

## `grilling` — Fase 2

**Tujuan.** Menguji setiap premis yang dipakai Steering dan BRD terhadap bukti di export, lalu
mengangkat yang tidak terbukti sebagai pertanyaan kepada Work Owner — bukan mengisinya dengan
asumsi.

**Hasil terukur.** 36 pertanyaan dalam 9 ronde, seluruhnya dijawab Work Owner, menghasilkan
**41 keputusan baru** (`D-31`…`D-71`).

**Empat kesalahan saya sendiri yang tertangkap oleh disiplin skill ini**, dicatat karena
pola kesalahannya lebih berguna daripada kesalahannya:

| # | Kesalahan | Bagaimana tertangkap |
|---|---|---|
| 1 | **Kerangka pertanyaan Q12 salah.** `LIMIT_BOTTOM <=` disajikan sebagai cacat, dan rentang tertutup ditawarkan sebagai perbaikannya | Penelusuran lanjutan menemukan `KomiteLoop := pxResultCount` — itu **mekanisme penjenjangan**, bukan cacat. Perbaikan yang ditawarkan akan menurunkan **setiap** klaim menjadi satu penyetuju. Pertanyaan diajukan ulang sebagai Q12-ulang **sebelum** jawaban pertama dipakai |
| 2 | **Klaim Google Cloud Storage sebagai mekanisme dokumen keempat** | `Activity/UploadDocumentToGoogleStorage-Act.xml` ternyata **nol `<pyMethod>`** dan mendelegasikan lewat `Call InsertDokumenPNC`. Temuan ditarik; `D-16` tetap benar |
| 3 | **Tiga arti kode status disimpulkan dari pemakaian** | Master `v_sts_claim.csv` membuktikan **ketiganya salah** — `1143`, `1150`, `1151` |
| 4 | **`SetTicket` dilaporkan sebagai rule hilang** | Ternyata bawaan Pega (`pyRuleSet=Pega-ProcessEngine`); ditarik dari daftar permintaan ke Tim Pega |

**Tiga laporan sub-agen yang dikoreksi**, karena hasil agen tidak diterima begitu saja:

1. Klaim bahwa nilai `client_id`/`client_secret` tidak ada di export — agen mencari di
   `pyParameterName`, sedangkan nilainya ada di `pyMapFromKey` sebelumnya. Temuan asli bertahan.
2. Dugaan blok `// TESTING` tidak aktif karena `pyStepsBlockName=//` — dibantah dengan sensus:
   `//` dipakai **952× di 279 activity** berdampingan dengan label bermakna, dan format export ini
   **tidak punya elemen aktif/nonaktif sama sekali**.
3. Klaim `CompareDates` sebagai fungsi custom yang hilang — sensus menunjukkan
   `@DateTime.CompareDates` berada di pustaka bawaan yang sama dengan `addCalendar`. **Delapan
   pertanyaan ditarik** dari daftar.

**Dampak ke Steering:** seluruh bab. Lihat `21-RIWAYAT-REVISI.md`.

## `domain-modeling` — Fase 3 dan 4

**Tujuan.** Menajamkan bahasa domain sebelum ADR ditulis, agar istilah di ADR dan tiket tidak
bertabrakan dengan `CONTEXT.md`.

**Hasil pada `CONTEXT.md`:** dari 61 menjadi **70 istilah**; **tidak ada istilah `[TERBUKA]` yang
tersisa**.

| Istilah | Isi |
|---|---|
| **Status Klaim** | dinaikkan dari `[TERBUKA]` menjadi `[BISNIS]`; domain dikoreksi menjadi **33 kode `1134`–`1166`** |
| **Komite** | ditulis ulang sebagai **kumulatif** |
| **Pita Nilai Komite** · **Jenjang Kumulatif** | baru — menjelaskan dua langkah penentuan jumlah penyetuju |
| **Sisa TSI** · **Kurs Standar** | baru |
| **Tugas · Worklist · Workbasket** | baru — inti `D-26`, sebelumnya tidak ada sama sekali di glossary |
| **Lompatan Lateral (Ticket rule)** · **Tiket Pekerjaan** | baru — seksi "Istilah yang mudah tertukar" |

**Satu usulan yang ditolak Work Owner, dan penolakannya benar.** Saya mengusulkan menggabungkan
dua dari empat konsep status karena bukti pemakaiannya lemah. Work Owner menegaskan *"4 status
tersebut memang berbeda"*. Usulan ditarik, `D-18` dibiarkan utuh, dan catatan buktinya disimpan di
dokumen verifikasi — bukan dinaikkan menjadi definisi.

**Dampak ke Steering:** `CONTEXT.md` · Domain Model · seluruh ADR.

## `writing-for-agents` dan `codebase-design`

| Skill | Pemakaian |
|---|---|
| `writing-for-agents` | dipakai terbatas pada `docs/tools/README.md` dan berkas `docs/agents/`, yang memang ditujukan untuk dibaca agen. Dokumen Steering, ADR, dan BRD tetap ditulis untuk manusia |
| `codebase-design` | dievaluasi untuk menentukan batas modul dan seam pada ADR-0001 dan ADR-0004; tidak menghasilkan artefak tersendiri |

## Skill yang tersedia tetapi tidak dipakai pada sesi ini

Alasannya sama dengan sesi sebelumnya dan tidak berubah: tahap ini **melarang penulisan kode**,
sehingga `tdd`, `prototype`, `diagnosing-bugs`, `code-review`, dan `resolving-merge-conflicts`
belum relevan. `research` tetap tidak dipakai karena seluruh fakta tersedia di dalam repository —
tidak ada satu pun klaim di sesi ini yang bersumber dari luar.

---

## Sesi 4 — restrukturisasi ticketing, perbaikan pipeline dokumen, dan Fase 6

| Skill | Pemakaian pada sesi ini |
|---|---|
| `domain-modeling` | Dipakai paling berat di sesi ini. Menambah **7 istilah** ke `CONTEXT.md` (`Master Data`, `Correspondence`, `Notifier`, `Job Terjadwal`, `Uji Kesetaraan`, `Gerbang 1`, `Gerbang 2`) setelah butir 4 suite verifikasi **merah** — ketujuhnya dipakai di tiket sebelum didefinisikan. Menjelang akhir sesi menemukan lubang yang lebih mendasar: **kata "Inbox" tidak pernah punya entri** padahal dipakai di seluruh dokumen, sementara `Worklist` dan `Workbasket` sudah ada sejak `D-26`. Itulah sebab langsung kebingungan `U-3` versus `U-6` |
| `grilling` | Tidak diminta secara eksplisit, tetapi disiplinnya dipakai pada diri sendiri: setiap angka yang dipakai di 102 tiket diuji ke source lebih dulu. Itu yang memunculkan koreksi `15 pemanggil` (bukan 17), `31 lokasi` (bukan 44), dan `21 Connect REST` (bukan 12) |
| `writing-for-agents` | `docs/tools/README.md` diperluas dengan sebab dan cara memeriksa ulang lebar tabel `.docx` |
| `codebase-design` | Dipakai menetapkan bahwa aturan penggolongan harness tinggal di **satu fungsi** (`usul()`) di generator, bukan tersebar sebagai isian tangan di 74 baris tabel |

## Tiga kesalahan saya sendiri pada sesi ini — dicatat supaya tidak terulang

Ketiganya punya pola yang sama: **alat ukur dipercaya sebelum divalidasi.**

| Kesalahan | Bagaimana ketahuan | Pelajaran |
|---|---|---|
| Melaporkan `SendEmailNotification` punya **17 pemanggil** | Angka itu saya salin dari `verifikasi-bukti-adr.md`, bukan dari sumbernya. `19-GAP-EXPORT-DETAIL.md:196` menyebut **15** beserta nama kelima belasnya | Dokumen turunan bukan sumber. Yang menyebut nama satu per satu lebih kuat daripada yang hanya menyebut angka |
| Mengukur lebar tabel `.docx` lewat `Cell.Width` Word COM, lalu melaporkan "11 dari 64 melebihi" dan "memburuk jadi 16" | Angkanya **identik** meski HTML-nya berubah total. Dokumen hasil impor HTML dibuka dalam **Web Layout**, dan nilainya bukan lebar cetak | Angka yang tidak bergerak saat masukannya berubah adalah **alat ukur yang rusak**, bukan temuan |
| Menguji korelasi nama *Inbox* dengan model penugasan memakai penanda `worklist`/`Assign-` | Penandanya menangkap boilerplate Pega (`pyDashboardMyWorkList`); `InboxRegister_Harness` — inbox sungguhan — justru **nol kecocokan** | Sebelum memakai penanda, uji dulu ia menyala pada kasus yang jelas benar |

Ketiganya dilaporkan apa adanya kepada Work Owner saat ditemukan, bukan dirapikan diam-diam.

## Skill yang tersedia tetapi tetap tidak dipakai

Alasannya tidak berubah: tahap ini **melarang penulisan kode implementasi**, sehingga `tdd`,
`prototype`, `diagnosing-bugs`, `code-review`, dan `resolving-merge-conflicts` belum relevan.
`research` tetap tidak dipakai — seluruh fakta sesi ini berasal dari dalam repository, dan tidak
ada satu pun klaim yang bersumber dari luar.

# Lampiran E — Rincian Gap Export

Daftar lengkap artefak yang **dirujuk oleh aplikasi tetapi tidak ada di export XML**. Dokumen
ini melengkapi `16-RISK-ANALYSIS.md`, yang hanya menyebut jumlah dan beberapa contoh.

Gunakan daftar ini apa adanya sebagai lampiran permintaan ke DBA dan tim Pega.

Seluruh isi diekstraksi langsung dari 652 rule Connect-SQL dan 902 Activity di export.

> **Diperbarui v2.0 (2026-09-14) — dokumen ini ternyata jauh dari lengkap.**
>
> Audit ulang terhadap snapshot baru menemukan **±242 rule dirujuk tetapi tidak ada di export**,
> bukan puluhan. Yang lebih penting: **tujuh tipe rule tidak diaudit dokumen ini sama sekali** —
> **When, Flow Action, Section, Data Transform, Ticket, Report Definition, dan Function library**.
>
> | Kategori | Diaudit dokumen ini | Temuan v2.0 |
> |---|---|---|
> | Objek database (`R-01`) | ✅ 64 objek | ✅ **62 diterima**; sisa `SET_ATTACHFILETEMPSALVAGE`, `UPDATEPREMIUMTEMPLATE` — **plus 12 dependensi baru** yang dipanggil 62 berkas itu |
> | Router (`R-04`) | ✅ 8 router | algoritma beban **terbaca dari kueri lain**; risiko turun menjadi verifikasi |
> | Activity (`R-07`) | ✅ 50 activity | tetap berlaku |
> | **When rule** | ❌ tidak diaudit | **137 When rule buatan sendiri hilang** — kategori terbesar |
> | **Ticket rule** | ❌ tidak diaudit | **17 nama dirujuk, 8 ada, 9 tanpa rule** (1 di antaranya bawaan Pega) |
> | Flow Action · Section · Data Transform · Report Definition · Function | ❌ tidak diaudit | tercakup dalam ±242 |
>
> Konsekuensinya dicatat sebagai **`R-16`**, dan permintaan penggantinya bukan lagi daftar manual
> per tipe melainkan **export ulang berbasis Product rule** (`D-39`) — karena daftar manual
> terbukti melewatkan tujuh tipe sekaligus.
>
> **Isi §-§ di bawah tetap sahih** untuk tiga kategori yang memang diauditnya; ia kurang lengkap,
> bukan salah.

---

## Koreksi jumlah stored procedure

Angka **86 stored procedure** yang tercantum di dokumen Steering lain adalah **hasil hitungan
yang terlalu tinggi**. Pemindaian awal menangkap pola `SKEMA.NAMA(` tanpa memeriksa kata di
depannya, sehingga nama tabel pada `INSERT INTO POOLDATA.T_CLAIM_SLIK_OJK (kolom, kolom, …)`
ikut terhitung sebagai pemanggilan procedure.

Setelah pemindaian ulang dengan pemeriksaan posisi sintaksis, angka terverifikasi:

| Kategori | Jumlah | Keterangan |
|---|---|---|
| **Stored procedure** | 56 | Dipanggil sebagai statement di dalam blok PL/SQL |
| **Function lokal** | 8 | Dipanggil sebagai ekspresi di dalam SELECT/WHERE |
| **Function remote (DB Link)** | 6 | Milik database lain — masuk lingkup **R-03**, bukan R-01 |
| **Total** | 70 |  |

> **Yang harus diminta ke DBA untuk R-01 adalah 64 objek** —
> 56 procedure dan 8 function lokal. Yang 6 lagi berada di database lain
> dan ditangani lewat penggantian DB Link (D-25).

---

## R-01 · Stored Procedure yang Source-nya Dibutuhkan

**56 procedure.** Kolom *Param* adalah jumlah argumen pada pemanggilan — berguna untuk
memverifikasi bahwa source yang diterima memang versi yang dipakai aplikasi.

| No | Objek database | Param | Rule pemanggil | Modul terdampak |
|---|---|---|---|---|
| 1 | `COLLECTION.P_GET_DATA_REFUND` | 5 | `BrowseDataRefund` | perlu ditentukan |
| 2 | `GENERAL.GET_TOKEN_STORAGE` | 4 | `GenerateTokenPNCDokumen` | S-1 Dokumen |
| 3 | `KONVERSIT_VEHICLELIST` | 2 | `KonversiT_Vehiclelist_SQL` | B-1 Polis/Snapshot |
| 4 | `POOLDATA.ADD_NEWMASTERVIRTUALACCOUNT` | 8 | `InsertDataMSTVirtualAccount` | B-10 Akseptasi |
| 5 | `POOLDATA.CONVERTJSONPRODUCTION` | 3 | `ConvertJSONProduction_SQL` | B-1 Polis/Snapshot |
| 6 | `POOLDATA.DOCTRAVEL_CVG` | 4 | `UpdateMstDocTravel` | S-1 Dokumen |
| 7 | `POOLDATA.INSERT_KPIADJUSTER` | 14 | `CallProcedureInsertKPISurvey` | B-8 Survey |
| 8 | `POOLDATA.INSERT_PLADLA` | 28 | `CallProccedureInsertPLADLA2` | B-9 PLA/DLA |
| 9 | `POOLDATA.INSERT_PNCCHRONOLOGYTAT` | 13 | `CallProccedureInsertMitra` | S-7 Progres/TAT |
| 10 | `POOLDATA.INSERT_SALVAGE` | 29 | `InsertNewSalvage` | B-12 Salvage |
| 11 | `POOLDATA.INSERT_SALVAGE_DETAILS` | 18 | `InsertDetailsSalvageperObject` | B-12 Salvage |
| 12 | `POOLDATA.INSERT_SURVEYORLIST` | 17 | `CallProcedureInsertSurvey` | B-8 Survey |
| 13 | `POOLDATA.INSERT_UPDATE_MST_XOL` | 18 | `SetMasterXOL` | B-9 PLA/DLA |
| 14 | `POOLDATA.INSERTDATAAIKLAIMPNC` | 30 | `InsertDataAIToTableFlatKlaimPA` | perlu ditentukan |
| 15 | `POOLDATA.INSERTDATAKLAIMCABANG` | 15 | `InsertDataCabang` | perlu ditentukan |
| 16 | `POOLDATA.INSERTDATAKOMITELIST` | 21 | `InsertDataToFlatTableKomiteList` | B-7 Komite |
| 17 | `POOLDATA.INSERTDATASFILLINGARCHIVE` | 18 | `InsertToClaimArchive` | S-1 Dokumen |
| 18 | `POOLDATA.INSERTMASTERRECOVERYKLAIM` | 21 | `InsertMasterRecoveryKlaimASM` | B-12 Salvage |
| 19 | `POOLDATA.INSERTMASTERREJECTEDKOMITE` | 3 | `MasterRejectedKlaimPNC` | B-7 Komite |
| 20 | `POOLDATA.INSERTOBJECTBACKUP` | 6 | `INSERTJSON_OBJECTPERSONPEGA` | B-1 Polis/Snapshot |
| 21 | `POOLDATA.INSERTPOLISTOJSON` | 2 | `InsertPolicyToJSON` | B-1 Polis/Snapshot |
| 22 | `POOLDATA.INSERTT_PERSONLIST` | 7 | `INSERTJSON_OBJECTPERSONPEGA` | B-1 Polis/Snapshot |
| 23 | `POOLDATA.MASTERPENOLAKANKLAIM1` | 3 | `SaveDataPenolakanKlaimMaster` | F-4 Master Data |
| 24 | `POOLDATA.MASTERPENOLAKANKLAIM2` | 6 | `SaveDataPenolakanKlaimMaster2` | F-4 Master Data |
| 25 | `POOLDATA.PEGA_CONVERT_JSONKLAIM_PNC` | 2 | `RunConvertJSONKLAIM` | B-1 Polis/Snapshot |
| 26 | `POOLDATA.PEGA_D_CAUSE_OF_LOSS` | 3 | `UpdateDCauseOfLoss` | F-4 Master Data |
| 27 | `POOLDATA.PEGA_D_PASAL_MASTER` | 4 | `UpdateDPasalDataMaster` | F-4 Master Data |
| 28 | `POOLDATA.PEGA_D_SURVEYORS` | 3 | `UpdateDetailSurveyors` | B-8 Survey |
| 29 | `POOLDATA.PEGA_JSON_INSERT_HISTORY_CLAIM_PNC` | 4 | `InsertHistoryClaimPNC` | B-1 Polis/Snapshot |
| 30 | `POOLDATA.PEGA_JSON_KLAIM_PNC` | 7 | `InsertClaimPNC` | B-1 Polis/Snapshot |
| 31 | `POOLDATA.PEGA_JSON_OS_AKSEP_KLAIM` | 7 | `InsertOSAkseptasiClaimPNC` | B-10 Akseptasi |
| 32 | `POOLDATA.PEGA_LOGJSON_LOG` | 3 | `Loging_InsertClaimPNC` | B-1 Polis/Snapshot |
| 33 | `POOLDATA.PEGA_LST_DET_TYPE_DOC` | 3 | `UpdateDetTypeDoc` | S-1 Dokumen |
| 34 | `POOLDATA.PEGA_LST_DET_TYPE_DOC_BUSINESS` | 14 | `UpdateDetTypeDocBusiness_SQL` | S-1 Dokumen |
| 35 | `POOLDATA.PEGA_LST_DOC_TYPE` | 3 | `UpdateLstDocType` | S-1 Dokumen |
| 36 | `POOLDATA.PEGA_M_BENGKEL_HE` | 3 | `UpdateBengkelHE` | F-4 Master Data |
| 37 | `POOLDATA.PEGA_M_CAUSE_OF_LOSS` | 3 | `UpdateMCauseOfLoss` | F-4 Master Data |
| 38 | `POOLDATA.PEGA_M_DOMINAN_FACTOR` | 4 | `InsertDominanfactor` | F-4 Master Data |
| 39 | `POOLDATA.PEGA_M_GROUPING_SPAREPART_HE` | 3 | `UpdateGroupingSparepartHE` | F-4 Master Data |
| 40 | `POOLDATA.PEGA_M_PANEL_HE` | 3 | `UpdatePanel_HE` | F-4 Master Data |
| 41 | `POOLDATA.PEGA_M_SPAREPART_HE` | 3 | `UpdateSparepartHE` | F-4 Master Data |
| 42 | `POOLDATA.PEGA_M_STS_CLAIM` | 3 | `UpdateStsClaim` | F-4 Master Data |
| 43 | `POOLDATA.PEGA_M_SUPPLIER` | 5 | `KonversiMasterSupplier_SQL` | F-4 Master Data |
| 44 | `POOLDATA.PEGA_M_SURVEYORS` | 3 | `UpdateMSurveyors` | B-8 Survey |
| 45 | `POOLDATA.PEGA_MST_USER_TEKNIS` | 10 | `UpdateMasterUserTeknis` | F-4 Master Data |
| 46 | `POOLDATA.PNC_INSERT_EMAIL_ADJUSTER` | 9 | `InsertEmailAdjuster_PNC` | B-8 Survey |
| 47 | `POOLDATA.PROC_JOBPERSONALACCIDENTKLAIM` | 13 | `ProcLODPersonalAccident_job` | perlu ditentukan |
| 48 | `POOLDATA.PROCINSERTDATARECIVEDKLAIM` | 27 | `Rcv_ProcInsertRecivedDocument` | B-14 Receive Doc |
| 49 | `POOLDATA.PROGRESS_CLAIM_PNC` | 16 | `InsertStatusProgress` | S-7 Progres/TAT |
| 50 | `POOLDATA.SET_ATTACHFILETEMPSALVAGE` | 10 | `SaveAttachmentToDBTemp_Sql` | B-12 Salvage |
| 51 | `POOLDATA.SET_ATTACHMENT_64BIT` | 11 | `SaveAttachmentToDB_Sql` | S-1 Dokumen |
| 52 | `POOLDATA.UPDATE_LOG_PROTEKSI` | 15 | `SaveMstProteksi_SQL` | B-13 Open Protection |
| 53 | `POOLDATA.UPDATEINSERT_PENGKINIANDATA` | 12 | `InsertPengkinianData_SQL` | B-2 Registrasi |
| 54 | `POOLDATA.UPDATEPREMIUMTEMPLATE` | 2 | `UpdateTemplatePKS` | perlu ditentukan |
| 55 | `POOLDATA.UPDATEREAS` | 7 | `UpdateEmailReas` | B-9 PLA/DLA |
| 56 | `PROCESSNEWEDMOBJECTDATA` | 3 | `ProcessNewEDMObjectData` | B-1 Polis/Snapshot |

### Function lokal

**8 function.** Dipanggil di dalam SELECT sehingga logikanya ikut menentukan hasil query —
harus ikut ditulis ulang di Go (D-02).

| No | Objek database | Param | Dipakai di rule | Modul terdampak |
|---|---|---|---|---|
| 1 | `MBU.F_VALIDASI_KLAIM_PENGKINIAN` | 3 | 1 | B-2 Registrasi |
| 2 | `POOLDATA.BASE64ENCODE` | 1 | 5 | S-1 Dokumen |
| 3 | `POOLDATA.GET_GROUPBUSINESS_XOL` | 3 | 1 | B-9 PLA/DLA |
| 4 | `POOLDATA.GET_INTERPOLASIPNC` | 1 | 4 | B-5 Settlement |
| 5 | `POOLDATA.GET_POSISI_PROGRESS_PNC` | 2 | 2 | S-7 Progres/TAT |
| 6 | `POOLDATA.GET_POSISI_PROGRESS2` | 3 | 6 | S-7 Progres/TAT |
| 7 | `POOLDATA.GETCURRENCYSTANDARD` | 2 | 2 | B-5 Settlement |
| 8 | `POOLDATA.GETSELISIHJAM` | 2 | 1 | S-7 Progres/TAT |

### Function remote via DB Link — lingkup R-03

**6 function** milik database lain. Tidak diminta ke DBA POOLDATA; digantikan API sesuai D-25.

| No | Function | DB Link | Param | Dipakai di rule |
|---|---|---|---|---|
| 1 | `DATAMINING.GET_WORKING_HOURS` | `asmd.sinarmas.co.id` | 2 | 6 |
| 2 | `GET_NAMA_AGEN` | `asmd.sinarmas.co.id` | 1 | 2 |
| 3 | `GET_NAMA_BISNIS` | `asmd.sinarmas.co.id` | 1 | 2 |
| 4 | `GET_NAMA_CABANG` | `asmd.sinarmas.co.id` | 1 | 2 |
| 5 | `GET_NAMA_CLIENT` | `asmd.sinarmas.co.id` | 1 | 2 |
| 6 | `GET_NAMA_MO` | `asmd.sinarmas.co.id` | 1 | 2 |

---

## R-04 · Router Penugasan

Seluruh router yang dirujuk oleh 4 flow, beserta status ketersediaannya di export.

| Router | Status | Dipakai oleh shape | Dampak bila hilang |
|---|---|---|---|
| `PNCAdminRouter` | **HILANG** | Register / Estimation · Register / Input Estimasi · Register / Input Register · Register / Input Estimasi | Menentukan penerima tugas |
| `PNCTeknikRouter` | **HILANG** | Register / Send To Analis · Register / Choose Surveyor · Register / Send To PIC Teknik | Menentukan penerima tugas |
| `RouterRCLDokter` | **HILANG** | Register / RCLDokter | Menentukan penerima tugas |
| `ToWorkList` | **HILANG** | Register / Analyst Doctor | Menentukan penerima tugas |
| `KomiteRouter` | Ada | Komite / KomiteRouter | — |
| `PNCAdminRouterRCV` | Ada | ReceiveDocument / InputReceiveDocument | — |
| `ToCurrentOperator` | Ada | CreateProtection / InputProtection · Register / View Polis | — |
| `ToWorkbasket` | Ada | CreateProtection / AksepProtection · Register / RCL/PUCL · Register / Investigator · Register / Compliance | — |

**4 router hilang**, tetapi hanya **3 yang benar-benar kritis**:

- `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter` — **buatan sendiri (ruleset GCNMFW)**,
  berisi aturan bisnis penentuan penerima tugas. Tanpa ini, modul B-6 tidak dapat diselesaikan.
- `ToWorkList` — **bawaan Pega** (`Pega-ProcessEngine`), sama seperti `ToWorkbasket` dan
  `ToCurrentOperator` yang ada di export. Perilakunya baku: menugaskan ke worklist operator.
  **Bukan gap nyata** — tidak perlu diminta.

Sebagai pembanding, dua router buatan sendiri yang **ada** di export dapat dipakai untuk menebak
pola ketiganya:

- `KomiteRouter` (GCNMFW 01-01-50) — memilih **worklist** `komitepnc` (tanpa angka), `komitepnc2`, `komitepnc3`, `komitepnc4` berdasarkan `KomiteCount`. **Worklist, bukan workbasket** — dua model penugasan yang berbeda (`D-26`).
- `PNCAdminRouterRCV` (GCNMFW 01-01-86) — menugaskan ke `ReceiveDocument.UserAdmin`, jatuh ke
  pembuat kasus bila kosong.

> Pola keduanya sederhana: menetapkan `Param.AssignTo` berdasarkan kondisi. Kemungkinan besar
> ketiga router yang hilang juga demikian — memilih PIC berdasarkan lini bisnis, cabang, dan
> beban kerja. Tapi ini **dugaan**, bukan fakta, dan tidak boleh dipakai sebagai dasar implementasi.

---

## R-07 · Activity yang Dipanggil tapi Tidak Ada di Export

**50 activity** dipanggil oleh activity lain tetapi tidak ikut diekspor:
**45 buatan sendiri** dan **5 bawaan Pega**.

### Buatan sendiri — perlu diminta

| No | Activity | Dipanggil | Oleh | Modul terdampak |
|---|---|---|---|---|
| 1 | `SendEmailNotification` | 15× | `ASMSendsEmailAttachments`, `ASMSendsEmailAttachments_PDF`, `CNMUpdatePanelHE_act`, `InputSurveyorBCAF_PostAct`, `InputSurveyorBRIF_PostAct`, `JobSendEmailNotificationPAYDI`, `SendWorkMailReport_Act`, `SubmitTanggalLengkapTKA`, `UpdateBengkelHE_act`, `UpdateGroupingSparepartHE_act`, `UpdateKategoriSparepart2_act2`, `UpdateKategoriSparepart_act2`, `UpdateSparepartHE_act`, `UpdateTypeSparepartReject_act2`, `UpdateTypeSparepart_act2` | S-3 Notifikasi |
| 2 | `CreateWorkPage` | 5× | `CreateNewMasterSupplier_post`, `EditMasterSupplier_post`, `KomitePost_Survey`, `SetChildKomitePerAdjustment_act` | perlu ditentukan |
| 3 | `SetTicket` | 4× | `AllCoveredResolved`, `KomitePost_Adjustment`, `KomitePost_Reject`, `SetListComiteeClaimPerObjAdj` | perlu ditentukan |
| 4 | `SetKasir_Act` | 3× | `CreateCasePNC_AsKredit`, `InsertCasePNC_AsuransiKredit`, `InsertCasePNC_Kredit_PA` | B-10 Akseptasi |
| 5 | `generatePD4ML` | 2× | `CreatePDFPolisAdjuster_act`, `GenerateViewPolisClaim` | perlu ditentukan |
| 6 | `generatePDF` | 2× | `CreatePDFPolisAdjuster_act`, `GenerateViewPolisClaim` | perlu ditentukan |
| 7 | `getRandomTeam_act` | 2× | `CallActivityInputRegister`, `InsertObjekAneka` | perlu ditentukan |
| 8 | `PNCSalvageHistorySemuaKlaim` | 2× | `SetDataSalavage_act`, `SetDataSalavage_act_ASI` | B-12 Salvage |
| 9 | `SetPageErrors` | 2× | `createWorkPage`, `pzCheckFieldSecurity` | perlu ditentukan |
| 10 | `AddCoveredDefaults` | 1× | `AddCoveredWork` | perlu ditentukan |
| 11 | `addWork` | 1× | `svcAddWorkObject` | perlu ditentukan |
| 12 | `CopyCoverage_act` | 1× | `ValidateInputEstimate_act` | perlu ditentukan |
| 13 | `ExportToExcel` | 1× | `ExportProduktivitasClaim_Act` | perlu ditentukan |
| 14 | `GCNMGetInternalSurveyor_Act` | 1× | `SetDashboardClaim` | B-8 Survey |
| 15 | `GCNMGetLossAdjuster_Act` | 1× | `SetDashboardClaim` | B-8 Survey |
| 16 | `GetAttchmentBase64CsvOrFile` | 1× | `ExportHasilDataAIKlaim` | S-1 Dokumen |
| 17 | `GetEmailSenderInfo` | 1× | `SendSimpleEmail` | S-3 Notifikasi |
| 18 | `GetKomunikasiRequestDokumen_Act` | 1× | `SetDataViewKlaimRequestDoc_Act` | B-10 Akseptasi |
| 19 | `InsertDataSlinkOJKIndividu` | 1× | `InsertAdjustmentListKredit` | S-4 Integrasi |
| 20 | `InsertJsonClaimNonMBUforKomite_act` | 1× | `KomitePost_Reject` | B-7 Komite |
| 21 | `InsertToSalvageDocument` | 1× | `SetStsSalvagePNC_act` | B-12 Salvage |
| 22 | `NotificationPAYDI` | 1× | `JobSendEmailNotificationPAYDI` | S-3 Notifikasi |
| 23 | `OsAkseptasiKlaim` | 1× | `KomitePost_Reject` | B-10 Akseptasi |
| 24 | `PartyNewSetup` | 1× | `createWorkPage` | perlu ditentukan |
| 25 | `PerformFlowAction` | 1× | `pzProcessIndividualDepAssignment` | perlu ditentukan |
| 26 | `RejectedCaseSurvey_Act` | 1× | `InputSurveyorBCAF_PostAct` | B-8 Survey |
| 27 | `RemoveFromCover` | 1× | `AddCoveredWork` | perlu ditentukan |
| 28 | `ReportKPILoginAdjuster_act` | 1× | `GetReportKPIAdjuster` | B-8 Survey |
| 29 | `runReport` | 1× | `pxShowReport` | perlu ditentukan |
| 30 | `SendEmailAlertPIC` | 1× | `ValidationUploadRegister` | S-3 Notifikasi |
| 31 | `SendFilePendukungLelangKeSimasBit` | 1× | `Insert_salvageToSimasBid` | B-12 Salvage |
| 32 | `SendUpdateCIF_act` | 1× | `InputRegister_act` | B-2 Registrasi |
| 33 | `SendWorkMail_Act` | 1× | `ServiceGetProductionData` | perlu ditentukan |
| 34 | `setCauseOfLossID_act` | 1× | `CallActivityInputRegister` | perlu ditentukan |
| 35 | `SetFlagactivityleaderKomite` | 1× | `SetAssignmentKomiteAI` | B-7 Komite |
| 36 | `SetListRegistKlaimPA` | 1× | `GetReportClaimRegistList` | perlu ditentukan |
| 37 | `SetspreadingtoCoverage` | 1× | `CallActivityInputRegister` | perlu ditentukan |
| 38 | `SetSumOfOutgo_Act` | 1× | `SetSumOfTSIPremium_Act` | perlu ditentukan |
| 39 | `SetSumOfTSIPerLocationList_Act` | 1× | `SetSumOfTSIPremium_Act` | perlu ditentukan |
| 40 | `SetViewAttachmentReas` | 1× | `SetDataViewKlaimReas_Act` | B-9 PLA/DLA |
| 41 | `ShowCoverage` | 1× | `CallActivityInputRegister` | perlu ditentukan |
| 42 | `TransferToKasir_act_Leader` | 1× | `SaveApprovalAkseptasiPaymentLeader_Act` | B-10 Akseptasi |
| 43 | `UpdateStatusMasterKomitexol` | 1× | `InsertUpdateMasterXOL` | B-9 PLA/DLA |
| 44 | `UploadDocumentToGoogleStorage` | 1× | `SetStsSalvagePNC_act` | S-1 Dokumen |
| 45 | `ValidasiSisaTSI` | 1× | `SetNilaiResikoSendiri` | perlu ditentukan |

### Bawaan Pega — tidak perlu diminta

Berawalan `px`/`py`/`pz`, milik ruleset Pega. Perilakunya baku dan akan digantikan mekanisme
setara di sistem baru, bukan diporting.

| Activity | Dipanggil | Oleh |
|---|---|---|
| `pxAddChildWork` | 1× | `SetChildKomitePerAdjustment_act` |
| `pxChooseBestRuleSet` | 1× | `pxShowReport` |
| `pxTransferAssignment` | 1× | `GCNMTransferDataKlaim_act` |
| `pxUploadCSVResults` | 1× | `PNCUploadAutoClaimSlikOJK` |
| `pyDeleteAttachmentContent` | 1× | `DeleteAttachment` |

---

## Cara Meminta ke DBA

Daftar di atas dapat dikirim apa adanya. Untuk mempercepat, berikut query yang dapat dijalankan
DBA guna menarik seluruh source sekaligus.

**Query penarikan source:**

```sql
SELECT owner, name, type, line, text
  FROM all_source
 WHERE (owner, name) IN (
       ('COLLECTION', 'P_GET_DATA_REFUND'),
       ('GENERAL', 'GET_TOKEN_STORAGE'),
       ('POOLDATA', 'KONVERSIT_VEHICLELIST'),
       ('POOLDATA', 'ADD_NEWMASTERVIRTUALACCOUNT'),
       ('POOLDATA', 'CONVERTJSONPRODUCTION'),
       ('POOLDATA', 'DOCTRAVEL_CVG'),
       ('POOLDATA', 'INSERT_KPIADJUSTER'),
       ('POOLDATA', 'INSERT_PLADLA'),
       ('POOLDATA', 'INSERT_PNCCHRONOLOGYTAT'),
       ('POOLDATA', 'INSERT_SALVAGE'),
       ('POOLDATA', 'INSERT_SALVAGE_DETAILS'),
       ('POOLDATA', 'INSERT_SURVEYORLIST'),
       ('POOLDATA', 'INSERT_UPDATE_MST_XOL'),
       ('POOLDATA', 'INSERTDATAAIKLAIMPNC'),
       ('POOLDATA', 'INSERTDATAKLAIMCABANG'),
       ('POOLDATA', 'INSERTDATAKOMITELIST'),
       ('POOLDATA', 'INSERTDATASFILLINGARCHIVE'),
       ('POOLDATA', 'INSERTMASTERRECOVERYKLAIM'),
       ('POOLDATA', 'INSERTMASTERREJECTEDKOMITE'),
       ('POOLDATA', 'INSERTPOLISTOJSON'),
       ('POOLDATA', 'MASTERPENOLAKANKLAIM1'),
       ('POOLDATA', 'MASTERPENOLAKANKLAIM2'),
       ('POOLDATA', 'PEGA_CONVERT_JSONKLAIM_PNC'),
       ('POOLDATA', 'PEGA_D_CAUSE_OF_LOSS'),
       ('POOLDATA', 'PEGA_D_PASAL_MASTER'),
       ('POOLDATA', 'PEGA_D_SURVEYORS'),
       ('POOLDATA', 'PEGA_JSON_INSERT_HISTORY_CLAIM_PNC'),
       ('POOLDATA', 'PEGA_JSON_KLAIM_PNC'),
       ('POOLDATA', 'PEGA_JSON_OS_AKSEP_KLAIM'),
       ('POOLDATA', 'PEGA_LOGJSON_LOG'),
       ('POOLDATA', 'PEGA_LST_DET_TYPE_DOC'),
       ('POOLDATA', 'PEGA_LST_DET_TYPE_DOC_BUSINESS'),
       ('POOLDATA', 'PEGA_LST_DOC_TYPE'),
       ('POOLDATA', 'PEGA_M_BENGKEL_HE'),
       ('POOLDATA', 'PEGA_M_CAUSE_OF_LOSS'),
       ('POOLDATA', 'PEGA_M_DOMINAN_FACTOR'),
       ('POOLDATA', 'PEGA_M_GROUPING_SPAREPART_HE'),
       ('POOLDATA', 'PEGA_M_PANEL_HE'),
       ('POOLDATA', 'PEGA_M_SPAREPART_HE'),
       ('POOLDATA', 'PEGA_M_STS_CLAIM'),
       ('POOLDATA', 'PEGA_M_SUPPLIER'),
       ('POOLDATA', 'PEGA_M_SURVEYORS'),
       ('POOLDATA', 'PEGA_MST_USER_TEKNIS'),
       ('POOLDATA', 'PNC_INSERT_EMAIL_ADJUSTER'),
       ('POOLDATA', 'PROC_JOBPERSONALACCIDENTKLAIM'),
       ('POOLDATA', 'PROCINSERTDATARECIVEDKLAIM'),
       ('POOLDATA', 'PROGRESS_CLAIM_PNC'),
       ('POOLDATA', 'SET_ATTACHFILETEMPSALVAGE'),
       ('POOLDATA', 'SET_ATTACHMENT_64BIT'),
       ('POOLDATA', 'UPDATE_LOG_PROTEKSI'),
       ('POOLDATA', 'UPDATEINSERT_PENGKINIANDATA'),
       ('POOLDATA', 'UPDATEPREMIUMTEMPLATE'),
       ('POOLDATA', 'UPDATEREAS'),
       ('POOLDATA', 'PROCESSNEWEDMOBJECTDATA'),
       ('POOLDATA', 'INSERTOBJECTBACKUP'),
       ('POOLDATA', 'INSERTT_PERSONLIST'),
       ('MBU', 'F_VALIDASI_KLAIM_PENGKINIAN'),
       ('POOLDATA', 'BASE64ENCODE'),
       ('POOLDATA', 'GET_GROUPBUSINESS_XOL'),
       ('POOLDATA', 'GET_INTERPOLASIPNC'),
       ('POOLDATA', 'GET_POSISI_PROGRESS_PNC'),
       ('POOLDATA', 'GET_POSISI_PROGRESS2'),
       ('POOLDATA', 'GETCURRENCYSTANDARD'),
       ('POOLDATA', 'GETSELISIHJAM')
 )
 ORDER BY owner, name, type, line;
```

**Sebaran per skema:**

| Skema | Jumlah objek |
|---|---|
| `POOLDATA` | 59 |
| `(tanpa skema)` | 2 |
| `COLLECTION` | 1 |
| `GENERAL` | 1 |
| `MBU` | 1 |

> Catatan: objek `KONVERSIT_VEHICLELIST` dan `PROCESSNEWEDMOBJECTDATA`
> dipanggil **tanpa nama skema**, sehingga skemanya mengikuti user koneksi aplikasi. DBA perlu
> memastikan skema sebenarnya saat menarik source.

**Yang juga perlu diminta bersamaan** — tanpa ini source procedure saja belum cukup:

- **Definisi tabel** yang disentuh tiap procedure (R-08), agar perilaku insert/update dapat dibaca utuh.
- **Grant dan sinonim** — beberapa objek dipanggil tanpa skema, kemungkinan lewat sinonim publik.
- **Isi master `V_STS_CLAIM`** (R-06), karena beberapa procedure menulis kolom status.

# Lampiran F — Rincian Penjenjangan Komite dan DB Link

Dokumen ini menjawab dua pertanyaan yang belum terjawab rinci di Steering:

1. **D-14** — aturan penjenjangan komite saat ini berada di mana saja?
2. **R-03** — DB Link dipakai pada proses apa saja, dan menyentuh objek apa saja?

Seluruh isi diekstraksi langsung dari 652 rule Connect-SQL dan 902 Activity di export XML.

> **Diperbarui v2.0 (2026-09-14) — Bagian 1 terjawab.** Isi `POOLDATA.EMAILKOMITE` sudah diterima
> (**21 kolom, 30 baris**), dan mekanisme penjenjangannya kini dipahami penuh. **Dua hal di bawah
> yang dibaca sebagai masalah ternyata bukan masalah**, dan satu di antaranya nyaris "diperbaiki"
> menjadi salah. Ringkasannya di §1.8 yang baru; isi §1.1–§1.7 dibiarkan sebagai catatan
> penelusuran saat itu.

---

## Bagian 1 — D-14: Penjenjangan Komite

## 1.1 Temuan utama: matriksnya sudah berupa master data

Pada Decision Log, D-14 dicatat sebagai *"matriks lengkapnya belum ada, harus dikumpulkan dari
tim bisnis"*. Setelah ditelusuri ke source, **anggapan itu tidak sepenuhnya benar**.

Matriks penjenjangan komite **sudah ada sebagai data**, tersimpan di tabel
`POOLDATA.EMAILKOMITE`. Yang belum kita punya hanyalah **isi tabelnya** — dan itu satu query.

Buktinya ada di `SetListComiteeClaimPerObjAdj` step 98:

```
childPageKomite.KomiteLoop  := TempRDBSearchEmailKomite.pxResultCount
childPageKomite.KomiteCount := 1
```

**Jumlah jenjang komite = jumlah baris yang dikembalikan query ke `EMAILKOMITE`.**
Bukan angka tetap, bukan aturan di dalam kode. Urutan jenjangnya ditentukan kolom `DEGREE`
(`ORDER BY DEGREE` di semua query).

## 1.2 Struktur tabel `POOLDATA.EMAILKOMITE`

Kolom berikut disimpulkan dari 11 query berbeda yang membacanya. Tipe dan panjangnya belum
diketahui (butuh DDL — R-08).

| Kolom | Perannya dalam penjenjangan |
|---|---|
| `DEGREE` | **Urutan jenjang.** Semua query `ORDER BY DEGREE`. Inilah yang menentukan siapa komite ke-1, ke-2, dst. |
| `TYPE_BUSINESS` | **Lini bisnis.** Nilai yang ditemukan: `NONMBU`, `NONMBUAB`, `NONMBUC`, `BONDING`, `TRAVEL`, `PA` |
| `TYPE_KOMITE` | **Kelompok komite.** Nilai `0`–`3`. Ditentukan oleh logika di aktivitas (lihat §1.4) |
| `LIMIT_BOTTOM` | **Ambang nilai bawah.** Filter `LIMIT_BOTTOM <= nilai klaim` — komite dengan ambang di atas nilai klaim tidak ikut terpilih |
| `LIMIT_BOTTOM_EXGRATIA` | Ambang khusus untuk klaim **Ex-Gratia** pada lini Travel |
| `OPERATOR_ID` | User komite — menjadi tujuan penugasan (`Param.AssignTo`) |
| `EMAIL` · `CC` | Alamat notifikasi komite |
| `STS_AKTIF` | Aktif/tidak. Semua query menyaring `= 1` |
| `STS_ADJ` | Berlaku untuk komite **adjustment** (penetapan nilai) |
| `STS_REG` | Berlaku untuk notifikasi saat **registrasi** |
| `STS_REJECT` | Berlaku untuk komite **penolakan** |
| `STS_EXGRATIA` | Berlaku untuk klaim **Ex-Gratia** |
| `STS_ABS` | Penanda user sedang tidak aktif (absen) — dicek terpisah |
| `ID` | Kunci baris; dipakai langsung pada kasus PA PHK (`WHERE ID = 2`) |

## 1.3 Sebelas query yang membaca tabel ini

Masing-masing melayani skenario berbeda. Perbedaannya ada pada kombinasi filter status dan
ambang nilai — inilah bentuk nyata "matriks" yang dimaksud D-14.

| Query (RDB rule) | Filter yang membedakan | Dipakai untuk |
|---|---|---|
| `EmailKomiteBerjenjang_sql` | `STS_ADJ=1` + `TYPE_BUSINESS` + `TYPE_KOMITE` + `LIMIT_BOTTOM` | Komite adjustment umum (Non-MBU) |
| `EmailKomiteBerjenjangBonding_sql` | `STS_ADJ=1` + `TYPE_BUSINESS` saja — **tanpa ambang nilai** | Bonding |
| `EmailKomiteBerjenjangTravel_sql` | `TYPE_BUSINESS=TRAVEL` + `LIMIT_BOTTOM` | Travel |
| `EmailKomiteBerjenjangTravelExGratia_sql` | `STS_EXGRATIA` + `LIMIT_BOTTOM_EXGRATIA` | Travel Ex-Gratia |
| `EmailKomiteBerjenjangPA_sql` | `STS_ADJ=1` + `TYPE_BUSINESS` + `LIMIT_BOTTOM` | Personal Accident |
| `EmailKomiteBerjenjangPATKI_sql` | ditambah `TYPE_KOMITE=2` | PA untuk Tenaga Kerja Asing |
| `EmailKomiteBerjenjangPAPHK_sql` | **`WHERE ID = 2`** — baris tetap, tanpa filter lain | PA kasus PHK |
| `EmailKomiteBerjenjangSimasnet_sql` | ditambah pengacakan `dbms_random.value` | Simasnet |
| `EmailKomiteAdjuster_sql` | `STS_AKTIF` + `TYPE_BUSINESS` + `TYPE_KOMITE` + `LIMIT_BOTTOM` | Komite fee adjuster |
| `EmailKomiteSalvage_sql` | sama, konteks salvage | Komite salvage |
| `GetEmailKomiteSimasnet_Reject` | `STS_REJECT=1` | Komite penolakan |
| `getEmailKomite_Register` | `STS_REG=1` + `TYPE_BUSINESS` | Notifikasi saat registrasi (bukan penjenjangan) |

> Perhatikan `EmailKomiteBerjenjangPAPHK_sql` yang memakai `WHERE ID = 2` — satu baris tetap
> yang di-hardcode di dalam query. Bila baris itu berpindah ID, alur PA PHK diam-diam salah.

## 1.4 Di mana logika pemilihannya berada

Sebelum query dijalankan, tiga parameter harus ditetapkan lebih dulu. Di sinilah aturan bisnis
yang sebenarnya berada — dan di sinilah masalahnya.

| Parameter | Dipetakan ke kolom | Ditetapkan di |
|---|---|---|
| `tempAdj.ConvertAdjustmentValue` | `LIMIT_BOTTOM` | Nilai settlement × kurs — `SetListComiteeClaimAI` / `SetListComiteeClaimPerObjAdj` step 61–63 |
| `tempAdj.pyMemo` | `TYPE_BUSINESS` | `SetEmailKomite` step 8–23 |
| `tempAdj.AcceptedNo` | `TYPE_KOMITE` | `SetEmailKomite` step 8–22 |

Nilai yang dibandingkan dengan `LIMIT_BOTTOM` berbeda menurut jenis pembayaran:

| Payment type | Nilai yang dipakai |
|---|---|
| `1` Final · `2` Interim · `5` Adjustment · Ex-Gratia | `AdjustmentValue` × kurs |
| `3` Salvage | `SalvageValue` × kurs |
| lainnya | `AdjusterFeeValue` × kurs |

**Empat aktivitas menetapkan parameter ini, masing-masing dengan aturan sendiri:**

| Aktivitas | Konteks |
|---|---|
| `SetEmailKomite` | Jalur utama — 32 step, menentukan `TYPE_BUSINESS` dan `TYPE_KOMITE` untuk Non-MBU, Bonding, Travel, PA |
| `SetEmailKomiteAdjuster` | Komite fee adjuster |
| `SetEmailKomiteSalvage` | Komite salvage |
| `SetEmailKomiteSimasnet` | Jalur Simasnet |

## 1.5 Masalah pada logika pemilihan

Aturan di keempat aktivitas itu **bercampur dengan nama orang**. Contoh nyata dari
`SetEmailKomite`:

```
step 10  jika nilai <= 50.000.000 DAN UserTeknis = "ELLENSUPRIYATI"
         maka TYPE_KOMITE := "1" dan nilai pembanding dipaksa jadi 50.000.001
         // komentar asli: "kalo ellen < 50jt, komite start dari INDRA"

step 14  jika nilai <= 100.000.000 DAN UserTeknis = "INDRAGUNAWAN..."
         maka TYPE_KOMITE := "2" dan nilai pembanding dipaksa jadi 100.000.001

step 12  jika nilai <= 50.000.000 DAN UserTeknis = "YOHANES..."
         maka TYPE_BUSINESS := "NONMBUAB", TYPE_KOMITE := "0"
```

Polanya: **nilai pembanding sengaja dinaikkan melewati ambang** agar jenjang komite tertentu
ikut atau tidak ikut terpilih, dengan syarat siapa PIC Teknis-nya.

Ditambah percabangan berdasarkan hostname server:

```
GetKomiteApproval step 1:
  ConvertAdjustmentValue := jika server = <hostname entitas Timor-Leste>
                           maka 3.500  selain itu  50.000.000
GetKomiteApproval step 2:
  jika jabatan user = "PA" atau "TRAVEL" maka 30.000.000
```

Angka `3.500` versus `50.000.000` adalah **mata uang berbeda** — entitas USD memakai 3.500.

## 1.5.1 Telusur lengkap ambang `3.500`

Angka `3.500` hanya muncul di **dua aktivitas, tiga tempat**. Tidak ada satu pun di SQL, Section,
Harness, When, Data Transform, Flow Action, maupun Report Definition — seluruh kemunculan di sana
terbukti hanya timestamp, `pyAutomationID`, dan checksum.

| # | Lokasi | Bentuk | Pemicu |
|---|---|---|---|
| 1 | `GetKomiteApproval` step 1 | `ConvertAdjustmentValue := @If(pxRequestor.pxReqServer == <hostname entitas Timor-Leste>, 3500, 50000000)` | **Hostname server** |
| 2 | `SetEmailKomite` step 11 | syarat `ConvertAdjustmentValue <= 3500` → set `TYPE_KOMITE = 1`, nilai dipaksa `3501` | `LSC_ID = "SMI"` **dan** `UserTeknis = "ELLENSUPRIYATI"` |
| 3 | `SetEmailKomite` step 13 | syarat `ConvertAdjustmentValue <= 3500` → set `TYPE_KOMITE = 0` | `LSC_ID = "SMI"` |

Pasangan USD untuk ambang `100.000.000` adalah `7.000`, di tempat yang sama:

| # | Lokasi | Bentuk | Pemicu |
|---|---|---|---|
| 4 | `SetEmailKomite` step 15 | syarat `<= 7000` → set `TYPE_KOMITE = 2`, nilai dipaksa `7001` | `LSC_ID = "SMI"` **dan** `UserTeknis = "INDRAGUNAWAN"` |
| 5 | `SetEmailKomite` step 18 | `TYPE_KOMITE := @If(ConvertAdjustmentValue > 7000, 2, 1)` | `LSC_ID = "SMI"` |

Rasio kedua pasangan konsisten: `50.000.000 ÷ 3.500 ≈ 100.000.000 ÷ 7.000 ≈ 14.285`. Ini kurs
IDR/USD yang **dibekukan ke dalam kode** — bukan diambil dari master kurs yang sudah dipakai di
tempat lain untuk mengonversi nilai klaim.

> Catatan: `CompressImage_Act` juga memuat angka `7000`, tetapi itu ambang ukuran berkas untuk
> kompresi gambar — **tidak berhubungan** dengan komite.

## 1.5.2 Dua mekanisme berbeda untuk hal yang sama

Ini temuan yang paling perlu diperhatikan. Kedua lokasi di atas menentukan "entitas mana ini"
dengan cara yang **berbeda dan dikelola terpisah**:

| | `GetKomiteApproval` | `SetEmailKomite` |
|---|---|---|
| Cara mengenali entitas | Membandingkan **hostname** dengan teks literal | Membaca `TempGetApp.LSC_ID` |
| Sumber nilainya | Ditulis di dalam kode | Hasil query tabel `POOLDATA.DB_LINK_PEGA` |
| Dipakai di | 1 tempat | 22 tempat |

`LSC_ID` berasal dari `GetLinkAppClaim`, yang menjalankan:

```sql
SELECT app AS "LSC_ID", NPP AS "LSC_NOTE"
  FROM pooldata.db_link_pega
 WHERE appip LIKE '%' || <hostname server> || '%'
```

Artinya **pemetaan hostname → entitas sudah berupa master data** di tabel `DB_LINK_PEGA`, dengan
nilai yang ditemukan: `ASM`, `SIMASNET`, `SMI`, `PEGAKREDIT`. Bila lookup gagal, `GetLinkAppClaim`
step 3 jatuh ke nilai baku `ASM`.

**Akibat dari ketidakkonsistenan ini:** bila server entitas USD dipindah atau namanya diubah,
`SetEmailKomite` tetap benar karena membaca tabel, sedangkan `GetKomiteApproval` diam-diam jatuh
ke `50.000.000` — ambang **rupiah** dipakai untuk klaim **dolar**. Selisihnya sekitar 14.000 kali
lipat, dan tidak ada pesan kesalahan yang muncul.

## 1.5.3 Kapan masing-masing ambang benar-benar dipakai

Ambang di `GetKomiteApproval` **tidak dipakai di jalur klaim normal**. Rantai pemanggilannya:

```
ValidationTypePaymentAdj
  ├─ s43 → SetListComiteeClaimPerObjAdj      ← JALUR KLAIM NORMAL
  │         s61–63  ConvertAdjustmentValue := nilai settlement × kurs
  │         s86–89  → SetEmailKomiteSimasnet / SetEmailKomite /
  │                    SetEmailKomiteSalvage / SetEmailKomiteAdjuster
  │         s98     KomiteLoop := jumlah baris hasil query
  └─ s45 → SetListComiteeClaimAI             ← JALUR AI, pola sama

GetKomiteApproval                            ← JALUR TERPISAH
  dipanggil hanya oleh: CNMInsertDetailSurveyors_act s7
                        CNMUpdateMasterRekening_act s31
  s1  ConvertAdjustmentValue := 3500 / 50.000.000   ← ambang tetap
  s2  := 30.000.000  bila jabatan user = PA atau TRAVEL
  s11 → SetEmailKomite
```

Jadi:

- **Jalur klaim normal** — `ConvertAdjustmentValue` berisi **nilai settlement** hasil konversi
  kurs. Angka 3.500 hanya muncul sebagai **batas pembanding** di syarat step 11 dan 13.
- **Jalur `GetKomiteApproval`** — dipakai dua aktivitas master data (input detail surveyor dan
  pembaruan master rekening). Di sini `ConvertAdjustmentValue` berisi **ambang tetap**, bukan
  nilai klaim.

**Satu properti dipakai untuk dua makna berbeda.** Kadang berarti "nilai klaim", kadang berarti
"ambang". Yang menentukan maknanya adalah aktivitas mana yang berjalan terakhir — dan itu tidak
terlihat saat membaca salah satu aktivitas saja.

## 1.5.4 Yang harus dilakukan di sistem baru

| Masalah | Perlakuan |
|---|---|
| Ambang `3.500` / `7.000` / `50jt` / `100jt` / `30jt` di kode | Master data per entitas dan per mata uang (D-15) |
| Kurs 14.285 dibekukan lewat pasangan angka | Pakai master kurs yang sudah ada, jangan bekukan pasangan ambang |
| Pengenalan entitas lewat hostname literal | Selalu lewat pemetaan entitas — jangan pernah membandingkan hostname di kode (§14.4) |
| `ConvertAdjustmentValue` berarti dua hal | Pisahkan: `NilaiSettlement` dan `AmbangKomite` sebagai dua konsep berbeda |
| Nama orang jadi syarat (`ELLENSUPRIYATI`, `INDRAGUNAWAN`, `YOHANESRAYMONDADIKARTA`) | Peran dan izin dari master data, bukan identitas orang |

**Yang perlu dikonfirmasi ke tim bisnis:** apakah entitas USD memang hanya satu (`SMI`), dan
apakah ambang 3.500/7.000 masih berlaku — mengingat kurs 14.285 sudah tidak mencerminkan kurs
saat ini.

> Seluruh perilaku ini masuk lingkup **D-15**: nama orang, ambang nilai, dan hostname menjadi
> master data yang dapat diubah tanpa deploy. Yang perlu dipastikan ke tim bisnis bukan
> "berapa jenjangnya", melainkan **mana aturan yang masih berlaku dan mana yang sisa tambalan lama**.

## 1.6 Mekanisme perputaran jenjang

```
SetListComiteeClaim*   KomiteLoop  := jumlah baris hasil query EMAILKOMITE
                       KomiteCount := 1
        │
        ▼
KomiteRouter           KomiteCount = 1 → workbasket "komitepnc"
                       KomiteCount = 2 → workbasket "komitepnc2"
                       KomiteCount = 3 → workbasket "komitepnc3"
                       KomiteCount = 4 → workbasket "komitepnc4"
                       KomiteCount := KomiteCount + 1 ; AcceptStatus := ""
        │
        ▼
Komite mengambil keputusan  AcceptStatus = 1 (setuju) / 2 (tolak)
        │
        ▼
IsKomiteLoop (when)    AcceptStatus = 1  DAN  KomiteCount <= KomiteLoop
                       ├─ benar → ulang ke KomiteRouter (jenjang berikutnya)
                       └─ salah → selesai
```

Aturan tambahan yang ditemukan:

- **Menolak = langsung selesai.** `KomitePost_Adjustment` step 65 dan `KomitePost_LiableKlaim`
  step 22–23: bila `AcceptStatus = 2`, `KomiteCount` dipaksa sama dengan `KomiteLoop` sehingga
  perulangan berhenti. Penolakan satu jenjang membatalkan seluruh sisa jenjang.
- **Persetujuan penuh baru diakui di jenjang terakhir.** `IsKomiteApprove` di-set `1` hanya
  ketika `KomiteCount == KomiteLoop` **dan** `AcceptStatus = 1`.
- **`KomitePost_Reject` step 2** menetapkan `KomiteLoop := -1` — penanda alur penolakan.
- **Batas 4 jenjang berasal dari router**, bukan dari data. Bila `EMAILKOMITE` mengembalikan
  lebih dari 4 baris, `KomiteRouter` tidak punya cabang untuk jenjang ke-5.

## 1.7 Yang perlu diminta untuk melengkapi D-14

| # | Yang diminta | Kepada | Kenapa |
|---|---|---|---|
| 1 | Isi tabel `POOLDATA.EMAILKOMITE` (seluruh baris, seluruh kolom) | DBA | Inilah matriks penjenjangan yang sebenarnya |
| 2 | DDL tabel `EMAILKOMITE` | DBA | Tipe kolom, panjang, constraint (bagian dari R-08) |
| 3 | Konfirmasi aturan berbasis nama orang di `SetEmailKomite` step 8–23 | Tim bisnis | Mana yang masih berlaku, mana sisa tambalan lama |
| 4 | Konfirmasi ambang 50jt / 30jt / 100jt / 3.500 | Tim bisnis | Apakah masih berlaku, dan apakah 3.500 memang USD Timor-Leste |
| 5 | Apakah jenjang bisa lebih dari 4 | Tim bisnis | Batas 4 saat ini berasal dari router, bukan dari data |

**Query untuk permintaan nomor 1:**

```sql
SELECT ID, DEGREE, TYPE_BUSINESS, TYPE_KOMITE,
       LIMIT_BOTTOM, LIMIT_BOTTOM_EXGRATIA,
       OPERATOR_ID, EMAIL, CC,
       STS_AKTIF, STS_ADJ, STS_REG, STS_REJECT, STS_EXGRATIA, STS_ABS
  FROM POOLDATA.EMAILKOMITE
 ORDER BY TYPE_BUSINESS, TYPE_KOMITE, LIMIT_BOTTOM, DEGREE;
```

> **Dampak ke status D-14:** dari *"matriks belum ada"* menjadi *"matriks ada sebagai master
> data, tinggal diambil isinya"*. Ini menurunkan bobot risikonya secara berarti — dan sekaligus
> membuktikan bahwa keputusan D-15 (nilai bisnis jadi master data) memang arah yang benar,
> karena sistem lama pun sudah setengah jalan ke sana.

---

## 1.8 Jawaban final setelah master diterima (2026-09-14)

Isi `POOLDATA.EMAILKOMITE` diterima: **21 kolom, 30 baris**. Seluruh pertanyaan §1.7 terjawab, dan
**dua pembacaan di §1.5 terbukti keliru**.

### 1.8.1 `LIMIT_BOTTOM <=` bukan cacat — itu mekanisme penjenjangannya

| Bukti | Isi |
|---|---|
| `Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459` | `childPageKomite.KomiteLoop := TempRDBSearchEmailKomite.pxResultCount` |
| `When/IsKomiteLoop-When.xml` | `.AcceptStatus = "1"` **dan** `.KomiteCount <= .KomiteLoop` |
| Sebaran kolom di 17 kueri `EMAILKOMITE` | `LIMIT_BOTTOM` difilter di **11**; `LIMIT_TOP` di **0** |

**Jumlah jenjang persetujuan = jumlah baris yang dikembalikan kueri**, dan flow memutar
`KomiteCount` dari 1 sampai `KomiteLoop`. Karena penyaringnya hanya batas bawah, seluruh jenjang
sampai tingkat nilai klaim ikut menyetujui: **klaim kecil sedikit penyetuju, klaim besar banyak
penyetuju**.

> **Perbaikan yang nyaris diterapkan dan akan merusak.** Mengganti penyaring menjadi rentang
> tertutup `LIMIT_BOTTOM <= nilai <= LIMIT_TOP` akan mengembalikan **tepat satu baris** →
> `KomiteLoop = 1` → **satu jenjang persetujuan berapa pun nilai klaim**. Itu menghapus
> penjenjangan yang menjadi inti `D-14` dan `BRD §11.4`. Kerangka pertanyaannya dikoreksi sebelum
> jawaban diterapkan (`D-47`).

**Keputusan:** model kumulatif **dipertahankan**; `LIMIT_TOP` dipakai sebagai **validasi
integritas master** — menolak master yang rentangnya tumpang tindih atau berlubang antar `DEGREE`
dalam satu `TYPE_BUSINESS` + `TYPE_KOMITE`.

### 1.8.2 `TYPE_KOMITE` adalah pita nilai **hanya** di Non-MBU

Baris aktif (`STS_AKTIF='1'`, `STS_ADJ='1'`) pada master yang diterima:

| Lini | Ambang bawah per jenjang | `DEGREE` | `TYPE_KOMITE` |
|---|---|---|---|
| **NONMBU** | `0` · `50.000.001` | 1 · 1 | `1` · `1` |
| **NONMBU** | `100.000.001` · `500.000.001` · `1.000.000.001` | 2 · 3 · 4 | `2` · `2` · `2` |
| **PA** | `0` · `10.000.001` · `50.000.001` · `100.000.001` | 1 · 2 · 3 · 4 | **`2` · `1` · `1` · `2`** |
| **TRAVEL** | `0` · `50.000.001` · `100.000.001` | 1 · 2 · 3 | `1` · `1` · `1` |

Pada **Non-MBU**, `TYPE_KOMITE` memang memisahkan pita ≤ Rp 100.000.000 dari pita di atasnya.
Pada **PA** nilainya berselang-seling **2 · 1 · 1 · 2** menaiki tangga — di sana ia membedakan
**jalur PA reguler dari PA TKI** (`EmailKomiteBerjenjangPATKI_sql-SQL.xml:82` mematok
`type_komite='2'`), bukan pita nilai. Pada **Travel**, seluruh jenjang aktif bernilai `1`, termasuk
jenjang di atas Rp 100.000.000.

**Akibat bila filter pita diberlakukan seragam ke semua lini** — dihitung dari master di atas:

| Kasus | Perilaku sistem lama | Bila pita disaring seragam |
|---|---|---|
| PA Rp 5.000.000 | 1 penyetuju | **0 penyetuju** — klaim mandek |
| PA Rp 75.000.000 | 3 penyetuju | 2 penyetuju |
| PA Rp 150.000.000 | 4 penyetuju | 2 penyetuju |
| Travel Rp 150.000.000 | 3 penyetuju | **0 penyetuju** — klaim mandek |
| Non-MBU, seluruh nilai | — | tidak berubah |

**Keputusan (`D-70`):** filter pita berlaku **khusus Non-MBU**; lini lain mengikuti kuerinya apa
adanya. Ini membatasi cakupan `D-52`, tidak membatalkannya.

### 1.8.3 Sisa pertanyaan §1.7

| # | Pertanyaan | Status |
|---|---|---|
| 1 | Isi tabel | ✅ diterima — 21 kolom, 30 baris |
| 2 | DDL tabel | Terbuka — bagian `R-08` |
| 3 | Aturan berbasis nama orang di `SetEmailKomite` | ✅ **dicabut** — diganti pita nilai yang diturunkan dari nilai klaim (`D-52`) |
| 4 | Konfirmasi ambang | ✅ **8 ambang komite unik** terverifikasi; ambang `3.500` dipicu **hostname entitas Timor-Leste** |
| 5 | Apakah jenjang bisa lebih dari 4 | ✅ **ya secara mekanisme** — jumlah jenjang mengikuti jumlah baris master, bukan batas tetap. Maksimum saat ini 4 karena isi master, bukan karena aturan |

**Yang belum terjawab dan dibawa ke tiket `B-7`/`B-12`:** tiga kueri yang memfilter `TYPE_KOMITE`
secara dinamis — `EmailKomiteBerjenjang_sql`, `EmailKomiteAdjuster_sql`, `EmailKomiteSalvage_sql` —
menerima nilainya dari pemanggil lewat `tempAdj.pyMemo` dan `tempAdj.AcceptedNo`. **Lini apa saja
yang benar-benar melewati ketiga kueri itu belum ditelusuri sampai ke sumber nilainya.**

Ditambah satu kekosongan data: **PA dan Travel di atas Rp 200.000.000 tidak punya baris master**.

---

## Bagian 2 — R-03: Inventaris DB Link

## 2.1 Ringkasan

| Ukuran | Jumlah |
|---|---|
| Total pemakaian DB Link | **64** |
| DB Link berbeda | **6** |
| Objek remote unik | **28** |
| Rule Connect-SQL yang memakainya | **27** |

| DB Link | Pemakaian | Objek | Rule | Sistem sumber |
|---|---|---|---|---|
| `@ASMD` | 55 | 20 | 24 | Database inti ASM (produksi) |
| `@SIMASNET` | 3 | 2 | 1 | Simasnet |
| `@SMI` | 2 | 2 | 1 | Sinarmas MSIG |
| `@OPJAVA` | 2 | 2 | 1 | Sistem OPJAVA |
| `@PROD_ASM` | 1 | 1 | 1 | Produksi ASM |
| `@PROD_TKA` | 1 | 1 | 1 | Sistem TKA |

> `@ASMD` menyumbang **86% dari seluruh pemakaian**. Bila hanya satu integrasi yang bisa
> diprioritaskan, itu adalah ASMD.

## 2.2 Objek remote — apa dan untuk apa

| DB Link | Objek remote | Jenis | Rule | Untuk apa |
|---|---|---|---|---|
| `@ASMD` | `DATAMINING.GET_WORKING_HOURS` | function | 6 | Menghitung selisih **jam kerja** antara dua waktu — dipakai untuk TAT dan KPI, agar akhir pekan dan hari libur tidak ikut terhitung |
| `@ASMD` | `HRDASM.V_HRD_MST` | tabel/view | 5 | Master pegawai HRD — mencari cabang, atasan, dan unit kerja seorang user |
| `@ASMD` | `LST_USER_ASURANSI` | tabel/view | 3 | Pemetaan user aplikasi ke kode cabang asuransi |
| `@ASMD` | `MST_DET_SALES` | tabel/view | 3 | Detail data sales/agen per polis — dipakai saat recovery dan penelusuran sumber bisnis |
| `@ASMD` | `GENERAL.LST_MITRA` | tabel/view | 2 | Master mitra (bengkel, investigator, surveyor eksternal) beserta akun login-nya |
| `@ASMD` | `GENERAL.MST_BUKA_PROTEKSI` | function | 2 | Procedure pembukaan proteksi di sistem sumber — dipanggil saat Open Protection dibuat |
| `@ASMD` | `GET_NAMA_AGEN` | function | 2 | Function: kode agen → nama agen |
| `@ASMD` | `GET_NAMA_BISNIS` | function | 2 | Function: kode bisnis → nama lini bisnis |
| `@ASMD` | `GET_NAMA_CABANG` | function | 2 | Function: kode cabang → nama cabang |
| `@ASMD` | `GET_NAMA_CLIENT` | function | 2 | Function: ID client → nama tertanggung |
| `@ASMD` | `GET_NAMA_MO` | function | 2 | Function: kode MO → nama Marketing Officer |
| `@ASMD` | `MST_SALES` | tabel/view | 2 | Master sales — dipakai bersama GET_NAMA_AGEN pada layar polis |
| `@ASMD` | `COLLECTION.MST_DET_SALES` | tabel/view | 1 | Sama seperti di atas, diakses lewat skema COLLECTION untuk laporan tanggal registrasi |
| `@ASMD` | `COLLECTION.TEMP_PREMI_WOM` | tabel/view | 1 | Data premi WOM Finance — dipakai menghitung total premi |
| `@ASMD` | `GENERAL.HRD_LBR` | tabel/view | 1 | Kalender **hari libur** perusahaan — dasar perhitungan hari kerja |
| `@ASMD` | `GL.T_ALL_PAYMENT` | tabel/view | 1 | Riwayat pembayaran di General Ledger — dicari berdasarkan nomor rekening |
| `@ASMD` | `LST_DET_CABANG` | tabel/view | 1 | Detail cabang |
| `@ASMD` | `MBU.T_CADANGAN_KLAIM_KREDIT_04203` | tabel/view | 1 | Cadangan klaim kredit Mandala Finance — dipakai menghitung total premi |
| `@ASMD` | `TREATY_LOSS` | tabel/view | 1 | Data kerugian treaty — dipakai laporan outstanding per cabang |
| `@ASMD` | `V_KLAIM` | tabel/view | 1 | View klaim di sistem sumber — dicek saat membuka proteksi agar tidak dobel |
| `@OPJAVA` | `NEW_GENERAL.M_USER` | tabel/view | 1 | Master user sistem OPJAVA |
| `@OPJAVA` | `NEW_GENERAL.M_USER_JOB` | tabel/view | 1 | Beban kerja (job count) per user — dipakai menyeimbangkan penugasan |
| `@PROD_ASM` | `POOLDATA.AGENT` | tabel/view | 1 | Master agen di database produksi ASM |
| `@PROD_TKA` | `ANEKA.MST_SHARE_PU` | tabel/view | 1 | Master share Penanggung Utama untuk lini TKA — dipakai perhitungan spreading |
| `@SIMASNET` | `GENERAL.MST_BUKA_PROTEKSI` | function | 1 | Procedure pembukaan proteksi di sistem sumber — dipanggil saat Open Protection dibuat |
| `@SIMASNET` | `V_KLAIM` | tabel/view | 1 | View klaim di sistem sumber — dicek saat membuka proteksi agar tidak dobel |
| `@SMI` | `GENERAL.MST_BUKA_PROTEKSI` | function | 1 | Procedure pembukaan proteksi di sistem sumber — dipanggil saat Open Protection dibuat |
| `@SMI` | `V_KLAIM` | tabel/view | 1 | View klaim di sistem sumber — dicek saat membuka proteksi agar tidak dobel |

## 2.3 Proses bisnis yang terdampak

Dikelompokkan menurut proses, bukan menurut rule — inilah yang menentukan modul mana yang
tertahan bila API penggantinya belum ada.

| Proses bisnis | Objek remote yang dipakai | Modul terdampak |
|---|---|---|
| Pencarian & tampilan Polis | `GET_NAMA_AGEN` · `GET_NAMA_BISNIS` · `GET_NAMA_CABANG` · `GET_NAMA_CLIENT` · `GET_NAMA_MO` · `MST_SALES` · `LST_DET_CABANG` | B-1 |
| Open Protection | `MST_BUKA_PROTEKSI` · `V_KLAIM` · `MST_BUKA_PROTEKSI` · `V_KLAIM` · `MST_BUKA_PROTEKSI` · `V_KLAIM` | B-13 |
| Registrasi · Komite · Master user | `V_HRD_MST` · `LST_USER_ASURANSI` | B-2 · B-7 · F-4 |
| Spreading TKA | `AGENT` · `MST_SHARE_PU` | B-4 · B-9 |
| Laporan TAT & KPI | `GET_WORKING_HOURS` | S-2 · S-7 |
| Salvage & Recovery | `MST_DET_SALES` | B-12 |
| Investigator & Laporan Mitra | `LST_MITRA` | B-11 · S-2 |
| Laporan Compliance & TAT | `MST_DET_SALES` | S-2 |
| Perhitungan Premi | `TEMP_PREMI_WOM` | B-5 |
| Perhitungan hari kerja (TAT) | `HRD_LBR` | S-7 |
| Pencarian klaim by rekening | `T_ALL_PAYMENT` | B-10 |
| Perhitungan Premi (Asuransi Kredit) | `T_CADANGAN_KLAIM_KREDIT_04203` | B-5 |
| Laporan Outstanding per Cabang | `TREATY_LOSS` | S-2 |
| Master User Teknis | `M_USER` | F-4 |
| Master User Teknis (beban kerja) | `M_USER_JOB` | F-4 · B-6 |

## 2.4 Daftar lengkap 27 rule pemakai

| Rule Connect-SQL | DB Link | Objek remote | Dipanggil aktivitas |
|---|---|---|---|
| `AmbilDataKlaimDenganNoRekening` | `@ASMD` | `T_ALL_PAYMENT` | `PNCSearchHistoryKlaim_Act` |
| `BrowseNonMBUUsers` | `@ASMD` | `V_HRD_MST` | `PNCCallRDBName_act` |
| `ExportDataKomitesKlaimNONMBU` | `@ASMD` | `GET_WORKING_HOURS` | `PNCReportDataKomites_act` |
| `GetIDCabang1` | `@ASMD` | `V_HRD_MST` · `LST_USER_ASURANSI` | `GetLostAdjuster_act`, `KomitePost_Survey`, `SetTempLostAdjuster` |
| `ExportDetailMitraReport` | `@ASMD` | `LST_MITRA` | `PNCMitraReport_Act` |
| `GetDataOutstandingperCabangExport` | `@ASMD` | `TREATY_LOSS` | `ExportDataOSCabang` |
| `GetDataCabangToEmail` | `@ASMD` | `V_HRD_MST` · `LST_USER_ASURANSI` | `SendDataDariCabangKeKantorPusat_ACT` |
| `GetDataMitraLoginInvest` | `@ASMD` | `LST_MITRA` | `InjectDataInvetigatorMitra` |
| `SetTotalJobMstUserTeknis` | `@OPJAVA` `@ASMD` | `V_HRD_MST` · `M_USER` · `M_USER_JOB` | `SetTotalJob_act` |
| `CheckProtectTable` | `@ASMD` | `MST_BUKA_PROTEKSI` | `InsertOpenProtectionCase` |
| `InsertOpenPortectionCase` | `@SIMASNET` | `MST_BUKA_PROTEKSI` · `V_KLAIM` | `InsertOpenProtectionCase` |
| `InsertOpenPortectionCaseASM` | `@ASMD` | `MST_BUKA_PROTEKSI` · `V_KLAIM` | `InsertOpenProtectionCase` |
| `InsertOpenPortectionCaseSMI` | `@SMI` | `MST_BUKA_PROTEKSI` · `V_KLAIM` | `InsertOpenProtectionCase` |
| `CheckHoliday_SQL` | `@ASMD` | `HRD_LBR` | `GCNMTimeDifferenceWorkCalender_Act` |
| `BrowseDataPolicyRNWAllFilter_SQL` | `@ASMD` | `GET_NAMA_AGEN` · `GET_NAMA_BISNIS` · `GET_NAMA_CABANG` · `GET_NAMA_CLIENT` · `GET_NAMA_MO` · `MST_DET_SALES` · `MST_SALES` | `NextPageGrid_Act` |
| `BrowseDataPolicyRNWRTTAllFilter_SQL` | `@ASMD` | `GET_NAMA_AGEN` · `GET_NAMA_BISNIS` · `GET_NAMA_CABANG` · `GET_NAMA_CLIENT` · `GET_NAMA_MO` · `LST_DET_CABANG` · `MST_DET_SALES` · `MST_SALES` | `NextPageGrid_Act` |
| `GetMandalaFinancePremi_SQL` | `@ASMD` | `T_CADANGAN_KLAIM_KREDIT_04203` | `GetTotalPremi` |
| `GetwomPremi` | `@ASMD` | `TEMP_PREMI_WOM` | `GetTotalPremi` |
| `GetShareTKA` | `@PROD_ASM` `@PROD_TKA` | `MST_SHARE_PU` · `AGENT` | `SetNilaiResikoSendiri`, `SpreadingTKA` |
| `BroswseKlaimByRegisterDate` | `@ASMD` | `MST_DET_SALES` | `PNCComplianceReport_Act`, `PNCTATReport1_Act` |
| `GetIDCabang` | `@ASMD` | `V_HRD_MST` · `LST_USER_ASURANSI` | `AchiveDocument_klaimAdmin`, `CheckViewPolis_act`, `CreateInputKlaim_PNC` (+10) |
| `GetRecoveryClaimData` | `@ASMD` | `MST_DET_SALES` | `Insert_mst_recoveryKlaimASM` |
| `BrowseDataKPIAdmin` | `@ASMD` | `GET_WORKING_HOURS` | `PNCReportKPIAdmin_Act` |
| `BrowseDataKPIAdmin_PA` | `@ASMD` | `GET_WORKING_HOURS` | `PNCReportKPIAdmin_Act` |
| `GetDataKPIAdmin` | `@ASMD` | `GET_WORKING_HOURS` | `PNCReportKPIAdmin_Act` |
| `GetDataKPIAdminPA` | `@ASMD` | `GET_WORKING_HOURS` | `PNCReportKPIAdmin_Act` |
| `GetDataKPIAdminPA_khususPA` | `@ASMD` | `GET_WORKING_HOURS` | `PNCReportKPIAdmin_Act_khususPA` |

## 2.5 Enam API pengganti yang dibutuhkan

Digabung menurut sistem pemilik, bukan menurut objek — supaya jumlah pihak yang harus
dikoordinasikan sesedikit mungkin.

| # | API yang dibutuhkan | Menggantikan | Prioritas |
|---|---|---|---|
| 1 | **API Jam Kerja & Hari Libur** | `DATAMINING.GET_WORKING_HOURS` (6 rule) · `GENERAL.HRD_LBR` | Tinggi — dipakai seluruh laporan TAT dan KPI |
| 2 | **API Pegawai & Cabang** | `HRDASM.V_HRD_MST` (5 rule) · `LST_USER_ASURANSI` (3) · `LST_DET_CABANG` | Tinggi — `GetIDCabang` dipanggil 13 aktivitas, termasuk registrasi |
| 3 | **API Open Protection** | `GENERAL.MST_BUKA_PROTEKSI` + `V_KLAIM` di **3 sistem** (@ASMD, @SIMASNET, @SMI) | Tinggi — modul B-13 tidak jalan tanpa ini |
| 4 | **API Master Sales & Agen** | `MST_DET_SALES` · `MST_SALES` · `POOLDATA.AGENT` · fungsi `GET_NAMA_*` (5) | Sedang — sebagian besar hanya kode → nama, bisa disalin berkala |
| 5 | **API Premi & Pembayaran** | `GL.T_ALL_PAYMENT` · `COLLECTION.TEMP_PREMI_WOM` · `MBU.T_CADANGAN_KLAIM_KREDIT_04203` | Sedang |
| 6 | **API Mitra & User Lintas Sistem** | `GENERAL.LST_MITRA` · `NEW_GENERAL.M_USER` · `M_USER_JOB` · `ANEKA.MST_SHARE_PU` · `TREATY_LOSS` | Rendah–Sedang |

## 2.6 Mana yang bisa dijembatani salinan berkala

Tidak semua harus menunggu API. Pembedaannya: apakah data harus mutakhir saat itu juga.

| Boleh disalin berkala | Wajib real-time |
|---|---|
| `MST_SALES` · `MST_DET_SALES` · `POOLDATA.AGENT` | `GENERAL.MST_BUKA_PROTEKSI` — procedure yang **menulis** ke sistem sumber |
| `LST_DET_CABANG` · `LST_USER_ASURANSI` | `V_KLAIM` — cek duplikat proteksi harus melihat kondisi terkini |
| `GENERAL.HRD_LBR` (kalender libur) | `GL.T_ALL_PAYMENT` — status pembayaran berubah setiap saat |
| `ANEKA.MST_SHARE_PU` · `TREATY_LOSS` | `NEW_GENERAL.M_USER_JOB` — beban kerja untuk penugasan |
| fungsi `GET_NAMA_*` → salin sebagai tabel referensi | `HRDASM.V_HRD_MST` — bila dipakai saat registrasi |

> **Catatan penting tentang `GENERAL.MST_BUKA_PROTEKSI`:** ini satu-satunya objek remote yang
> **bukan pembacaan**, melainkan **procedure yang menulis** ke tiga sistem berbeda (@ASMD,
> @SIMASNET, @SMI). Penggantinya wajib API sungguhan dan wajib **idempoten** — salinan berkala
> tidak bisa dipakai, dan kegagalan di tengah tidak boleh menghasilkan proteksi ganda.

## 2.7 Risiko tersembunyi

- **Tiga sistem, satu proses.** Open Protection memanggil procedure yang sama di @ASMD,
  @SIMASNET, dan @SMI lewat tiga rule terpisah. Bila ketiganya diganti API, kontraknya harus
  seragam — kalau tidak, satu proses bisnis akan punya tiga perilaku berbeda.
- **`GetIDCabang` adalah titik paling kritis.** Dipanggil **13 aktivitas** termasuk jalur
  registrasi klaim. Bila API pegawai belum siap, modul B-2 ikut tertahan.
- **Perhitungan TAT bergantung penuh pada sistem lain.** `GET_WORKING_HOURS` dan `HRD_LBR`
  ada di @ASMD. Tanpa keduanya, seluruh laporan TAT dan KPI tidak dapat dihitung — dan itu
  laporan yang dipakai harian.
- **Fungsi `GET_NAMA_*` tampak sepele tapi ada di query pencarian polis** yang dipakai
  di layar. Mengganti dengan join ke tabel salinan mengubah rencana eksekusi query — perlu
  diukur ulang pada data sebesar produksi.

# Lampiran G — Inventaris 74 Harness

| | |
|---|---|
| **Tanggal** | 2026-09-14 |
| **Sumber** | direktori `Harness/` — **dihitung dan dibaca langsung**, bukan dari dokumen turunan |
| **Dasar** | `D-73` · pertanyaan terbuka `U-3` dan `U-4`/`U-5`/`U-6` |
| **Status** | Kolom **Usulan modul** terisi **usulan bertanda keyakinan** — bukan keputusan |

## Kenapa lampiran ini ada

Sebelum ini, nama ke-74 harness tersebar di **sembilan berkas** dan tidak ada satu pun yang memuat
seluruhnya. Dokumen yang ada bahkan sempat menyebut *"~20 inbox"* padahal direktorinya memuat
**26 berkas bernama Inbox**. Lampiran ini menjadikan direktori `Harness/` sebagai satu-satunya
sumber yang dirujuk.

## Koreksi: pemisahan berdasarkan NAMA bukan pembeda yang berguna

Pemisahan "Inbox versus non-Inbox" yang saya pakai sebelumnya **berdasarkan nama berkas**. Itu
pembeda yang lemah, dan buktinya dari export sendiri:

| Yang diuji | Hasil |
|---|---|
| Apakah harness bernama *Inbox* memakai model penugasan? | **Tidak terbukti.** Penanda yang saya pakai ternyata menangkap boilerplate Pega (`pyDashboardMyWorkList`), dan `InboxRegister_Harness` — inbox sungguhan — justru **nol kecocokan** |
| Apakah nama *Inbox* sejalan dengan kelas harness? | **Tidak.** 23 dari 26 bernama Inbox berkelas `Data-Portal` — **sama** dengan 35 harness yang bukan bernama Inbox |

**Kesimpulan:** nama berkas tidak memberi tahu apa pun yang tidak diberitahu kelasnya dengan lebih
baik. Nama tetap dicantumkan di kolom terakhir sebagai alat cari, **bukan** sebagai dasar pembagian.

## Pembeda yang benar-benar berpengaruh pada implementasi

| Kelas harness | Jumlah | Artinya bagi implementasi |
|---|---:|---|
| **`ASM-FW-GCNMFW-Work*`** | **7** | Terikat pada **satu klaim**. Layar hanya berarti bila sebuah klaim sudah dimuat — menentukan bentuk rute (`/klaim/{nomor}/…`), dan menempatkannya di **`U-4`** |
| **`@baseclass`** | **9** | Generik; konteksnya ditentukan pemanggil |
| **`Data-Portal`** | **58** | Layar berdiri sendiri di portal — daftar, master, laporan. Rutenya tidak memerlukan klaim |

Ketujuh harness berkelas Work adalah satu-satunya kelompok yang **sudah pasti** milik `U-4`:

```
InputProgress · ViewDetailHasilSurveyorInternal1 · ViewReceiveDocument
ViewTempDetailAllCase · ViewTempDetailClaim · ViewTempDetailReqDocument
View_DetailKlaimCabang_Harness
```

## Apa yang berubah bila pembagian ini salah

| Yang terpengaruh | Bagaimana |
|---|---|
| **Komponen yang dibangun `U-3`** | `U-3` membangun **satu layar inbox yang dipakai ulang**. Bila yang benar-benar inbox peran hanya 14 dan bukan 26, komponennya lebih kecil dan acceptance criteria-nya berubah |
| **Urutan kerja** | Layar yang membaca model penugasan **tidak dapat dibangun sebelum `B-6`**. Yang tidak membacanya, bisa lebih awal |
| **Bentuk rute** | Layar terikat klaim butuh nomor klaim di rutenya; layar portal tidak |
| **Siapa menguji gerbang 2** | Tiap layar diuji pemegang perannya. Salah modul berarti salah penguji |
| **Kendali akses** | Pada inbox, penyaringan "hanya pekerjaan saya" adalah **kendali akses**; pada master, kendalinya "siapa boleh mengubah" — dua hal berbeda |

**Jadi jawabannya: ya, pembagiannya berpengaruh pada implementasi — tetapi bukan pembagian
berdasarkan nama.** Yang berpengaruh adalah kelas harness, pola interaksinya, dan apakah ia
membaca model penugasan.

### Definisi Inbox kini menjadi dasar aturan penggolongan (`D-79`)

Sebelum `D-79`, usulan di kolom **Usulan modul** bersandar pada **sidik jari berkas** — dasar yang
korelasional. Dengan definisi yang disepakati, dasarnya berpindah menjadi **isi layar**:

> **Inbox** = layar berisi **daftar pekerjaan milik pengguna** — Tugas dari Worklist/Workbasket.
> Layar **data acuan** bukan Inbox, sekalipun dapat dicari dan sekalipun namanya mengandung
> kata "Inbox".

Aturan di `usul()` tidak berubah — sidik jari tetap dipakai sebagai **alat baca**, karena isi layar
tidak dapat dibaca langsung dari XML. Yang berubah adalah **kedudukannya**: sidik jari kini menjadi
cara menerapkan definisi, bukan pengganti definisi.

Akibatnya **tujuh harness bernama *Inbox* diusulkan ke `U-6`** — sidik jarinya (267–273 KB · RD 15)
nyaris identik dengan `MasterRekening`, bukan dengan `InboxRegister_Harness` (1.330 KB · RD 144).

## Yang masih perlu diputuskan Work Owner

1. ~~Mana yang benar-benar inbox peran.~~ — **definisinya tertutup `D-79`**. Yang tersisa adalah
   **koreksi atas usulan**: 27 bertanda `DUGAAN` dan 5 bertanda `JANGGAL`.
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

# Lampiran H — Status 40 Usulan Revisi

| | |
|---|---|
| **Tanggal sapuan** | 2026-09-15 |
| **Sumber daftar** | `docs/verifikasi-bukti-adr.md` §11 — 40 kontradiksi `K-1`…`K-40` |
| **Dasar** | `D-39` — seluruh angka mengikuti hasil verifikasi bukti, selisihnya jadi usulan revisi |
| **Status lampiran** | **hasil pemeriksaan, bukan perubahan** — belum ada yang diterapkan dari sapuan ini |

## Kenapa lampiran ini ada

`D-39` memutuskan setiap selisih antara dokumen dan bukti dicatat sebagai usulan revisi. Daftarnya
ada di `verifikasi-bukti-adr.md` §11 — tetapi **tanpa penanda status sama sekali**. Akibatnya tidak
ada cara mengetahui mana yang sudah diterapkan dan mana yang masih menggantung, selain memeriksa
satu per satu ke dokumennya. Itulah yang dikerjakan sapuan ini.

## Ringkasan

| Status | Jumlah | Artinya |
|---|---:|---|
| **Sudah diterapkan** | **10** | Dokumen sudah memuat angka/fakta terverifikasi |
| **Diterapkan pada 2026-09-15** | **10** | Disetujui Work Owner; `K-13` sengaja ditahan |
| **Ditahan** | **1** | `K-13` — mengubah aturan penolakan klaim ganda, menunggu konfirmasi tim bisnis |
| **BERTENTANGAN** | **1** | `K-6` — dua rekaman verifikasi tidak sepakat |
| **Tidak perlu tindakan** | **2** | Bukti justru mendukung klaim dokumen |
| **Perlu keputusan, bukan koreksi teks** | **11** | Menyangkut keputusan bisnis/arsitektur, bukan salah ketik angka |
| **Belum tuntas diverifikasi** | **5** | Perlu pembacaan rule lebih dalam; **tidak saya nyatakan bersih** |

## 1. Masih menggantung — 11 butir

Inilah yang dapat langsung diperbaiki bila disetujui. Semuanya sudah diverifikasi ke berkasnya.

| # | Klaim dokumen sekarang | Terverifikasi | Lokasi |
|---|---|---|---|
| `K-1` | 45 activity hilang | **40** (5 positif palsu, bawaan Pega) | `BRD.md:446`, `BRD.md:1289` |
| `K-3` | workbasket `komitepnc1..4` | **`komitepnc`** tanpa angka, dan **worklist**, bukan workbasket | `19-GAP-EXPORT-DETAIL.md:177` |
| `K-8` | 10 email · 4 user ID · 3 ambang | **66 · 24 · 8** | `BRD.md:267` |
| `K-10` | `InputRegister_act` step 103 dan 107 | **sub-step 2 dan 6 di dalam step 53** | `BRD.md:268` |
| `K-13` | kunci duplikasi = Polis + Objek + Lokasi | **juga DOL**; lokasi hanya untuk non-PA/non-Travel | `BRD.md:646` |
| `K-18` | ~20 inbox berbasis peran | **26** harness bernama *Inbox* | `06-MODULE-BREAKDOWN.md:85`, `BRD.md:526`, `BRD.md:1136` |
| `K-26` | 27 modul | **34** (32 enumerasi + `FR-S8` pada `D-42` + `FR-F6` pada `D-75`) | `BRD.md:90`, `BRD.md:1355` |
| `K-30` | 245 tabel di 5+ skema | **177** objek di **9 skema** | `BRD.md:82` |
| `K-31` | `OFFSET 500000` membaca setengah juta baris | **`OFFSET` nol kemunculan** di seluruh export | `requirement-summary.md:168` |
| `K-33` | 47 menu portal | **47 harness target** ✔ tetapi **51 item menu**; 11 dari 47 harness tidak ada di export | `requirement-summary.md:121`, `02-BUSINESS-UNDERSTANDING.md:179` |
| `K-37` | `F-5` berukuran **Kecil** | "Kecil" hanya benar untuk seam Clock — migrasinya **118 titik +7 jam di 36 activity** ditambah 101 titik +12 jam | `06-MODULE-BREAKDOWN.md:28` |

> **`K-31` bukan sekadar angka salah.** `OFFSET 500000` dipakai sebagai **dasar** `NFR-13`. Karena
> `OFFSET` tidak pernah muncul sekali pun di export, requirement itu berdiri di atas premis yang
> tidak ada — dan perlu **ditulis ulang**, bukan diperbaiki angkanya.

## 2. Sudah diterapkan — 11 butir

`K-2` · `K-9` · `K-11` · `K-12` · `K-19` · `K-24` · `K-25` · `K-27` · `K-32` · `K-36`

Contoh yang diperiksa langsung: `K-12` (toleransi spreading kini `ROUND(SUM(share),4) BETWEEN
99.9999 AND 100.0001` per `D-51`, menggantikan pencocokan substring) · `K-9` (`03-CURRENT-ARCHITECTURE.md:182`
sudah memuat "Koreksi daftar hostname") · `K-32` dan `K-36` (koreksi ukuran `U-2`, `F-4`, `U-6`
sudah tercatat di Module Breakdown).

## 3. Tidak perlu tindakan — 2 butir

| # | Sebabnya |
|---|---|
| `K-39` | **Cocok persis** — 68 pemakaian `ROWNUM` di 44 berkas; angka dokumen benar |
| `K-40` | Klaim didukung angka — `pyPrivilegeName` non-kosong **1 dari 902** activity |

## 4. Perlu keputusan, bukan koreksi teks — 11 butir

Kesebelas ini **tidak bisa diselesaikan dengan menyunting angka**. Masing-masing menyangkut
keputusan yang pemiliknya bukan saya.

| # | Pokok persoalan | Pemilik |
|---|---|---|
| `K-5` | Lingkup audit export: **7 tipe rule belum diaudit**; gap ±242 versus 43 yang tercatat | **Work Owner + Tim Pega** |
| `K-7` | `D-18` mempertahankan empat konsep status, tetapi `ClaimStatus` milik **GISFW** dan `StatusPosisi` bernilai tunggal | **Work Owner** — perlu keputusan baru yang menyebut `D-18` |
| `K-20` | Mekanisme penyimpanan dokumen ternyata **empat**, bukan tiga — GCS yang keempat, rule-nya hilang | **Tim Pega + Work Owner** |
| `K-21` | HCC/HCQ **nol jejak di export** → ini **integrasi baru**, bukan migrasi; gerbang 1 tidak berlaku bagi `F-3` | **Work Owner** |
| `K-22` | `D-15` konfigurasi tiga lapis: **nol DSS**; pola lama terkunci per IP dan bertabrakan dengan `D-27` | **Lead Engineer + Work Owner** |
| `K-23` | `D-26` mempertahankan model penugasan, padahal penguncian **nol kustomisasi** → sistem baru bebas memilih | **Work Owner** |
| `K-28` | **Jalur validasi API/JSON lebih longgar** daripada jalur layar — aturan 7/30/90 tidak berlaku di sana | **Work Owner** |
| `K-29` | Perangkaian SQL dari nilai pengguna — melanggar `BRD §21.2` #11, **skalanya jauh lebih besar** (538 `{ASIS:}`) | **Work Owner + Keamanan Informasi** |
| `K-34` | Paginasi sistem lama praktis **client-side** (3.189 grid page list) → `NFR-12`/`NFR-13` adalah **perubahan perilaku**, bukan penyalinan | **Work Owner** |
| `K-35` | Matriks jenjang komite **sudah berupa master data**, bukan hardcode yang perlu dipindah | **Work Owner** |
| `K-38` | `D-20` mengganti `ROWNUM` → `OFFSET`; angkanya benar tetapi **mayoritas `FETCH NEXT 1 ROW ONLY`** — bukan paginasi | **Lead Engineer** |

## 5. Belum tuntas diverifikasi — 5 butir

Saya **tidak menyatakan kelimanya bersih**. Memastikannya menuntut pembacaan rule lebih dalam
daripada yang dilakukan sapuan ini.

`K-4` (router `ReceiveDocument.UserAdmin` — properti operator, bukan workbasket) ·
`K-14` (kunci duplikasi PA di SQL) · `K-15` (Group Panel `003` hanya baris (1)(1)) ·
`K-16` (ambang PA/Travel dipicu **jabatan operator**, bukan lini bisnis) ·
`K-17` (nilai klaim ≤ TSI, **PA dikecualikan** — belum ada di BRD)

> `K-16` dan `K-17` menyangkut **aturan yang menentukan uang**. Keduanya layak diprioritaskan di
> atas koreksi angka mana pun di bagian 1.

## Yang tidak saya sentuh, dan alasannya

| Tempat | Alasan |
|---|---|
| Entri lama `00-DECISION-LOG.md` (`D-15`, `D-18`, `D-20`, `D-26`) | Aturan proyek melarang menyunting entri lama. Perubahan pikiran ditulis sebagai keputusan baru yang menyebut ID yang disupersede |
| `verifikasi-bukti-adr.md` §11 | Ia **rekaman temuan**. Statusnya dilacak di lampiran ini, bukan dengan mengubah temuannya |
| Tabel ringkasan v1.0 → v2.0 di `BRD.md:34` | Isinya memang pasangan **angka lama → angka baru**. Menggantinya akan menghapus jejak koreksinya sendiri |

---

## 6. Pembaruan 2026-09-15 — sepuluh diterapkan, satu ditahan, satu bertentangan

Work Owner menyetujui penerapan. Hasilnya:

### Sepuluh diterapkan

`K-1` · `K-3` · `K-8` · `K-10` · `K-18` · `K-26` · `K-30` · `K-31` · `K-33` · `K-37`

Berkas yang disunting: `BRD.md` (10 baris) · `06-MODULE-BREAKDOWN.md` (2) ·
`19-GAP-EXPORT-DETAIL.md` (1) · `02-BUSINESS-UNDERSTANDING.md` (1) · `requirement-summary.md` (3).

### `K-13` ditahan atas saran saya

Ia **bukan koreksi angka** melainkan **aturan penolakan klaim ganda**: kuncinya ternyata menyertakan
**DOL**, dan lokasi hanya berlaku untuk non-PA/non-Travel. Mengubahnya mengubah klaim mana yang
ditolak sebagai duplikat — layak dikonfirmasi ke tim bisnis lebih dulu.

### `K-6` BERTENTANGAN — tidak diterapkan

Dua rekaman verifikasi **tidak sepakat** tentang jumlah pemakaian DB Link:

| Sumber | Angka |
|---|---|
| `verifikasi-bukti-adr.md` §11 `K-6` | **71 pemakaian · 22 objek `@ASMD`** |
| `16-RISK-ANALYSIS.md:115` | **64 pemakaian · 6 DB Link · 28 objek remote · 27 rule** — ditulis sebagai *"angka terverifikasi"* |

Keduanya mengklaim hasil verifikasi, dan keduanya tidak dapat benar bersamaan. Saya sempat
menerapkan angka `K-6` ke tiga berkas, lalu **membatalkannya** begitu pertentangan ini terlihat —
karena memilih salah satu berarti menegaskan angka yang belum tentu benar.

**Perlu dihitung ulang langsung dari export**, bukan dipilih dari salah satu dokumen.
Pemilik: **Lead Engineer** (perhitungan) lalu **Work Owner** (penetapan).

### Tiga verdict "sudah diterapkan" saya ternyata salah

Ketiganya salah karena sebab yang sama: **saya memeriksa BRD saja, bukan seluruh dokumen.**

| # | Yang terlewat |
|---|---|
| `K-12` | Toleransi spreading `99,99%` masih tertulis di **enam berkas** — `02-BUSINESS-UNDERSTANDING`, `04-FUTURE-ARCHITECTURE`, `09-DATABASE-STRATEGY`, `14-TESTING-STRATEGY`, `CONTEXT.md`, `requirement-summary`. Sudah diperbaiki ke **4 desimal `99,9999`–`100,0001`** (`D-51`) |
| `K-6` | Lihat di atas — bukan hanya terlewat, tetapi **bertentangan** |
| `K-32` | Angka `18–27 kolom` masih ada di badan `01-FRONTEND-ANALYSIS.md`. **Dibiarkan dengan sengaja**: dokumen itu punya catatan koreksi di kepalanya, dan badannya adalah analisis historis yang menghasilkan keputusan — diperlakukan sama seperti entri Decision Log lama |

Ditemukan pula **dua berkas yang terlewat dari koreksi Connect REST kemarin** (`D-73`):
`03-CURRENT-ARCHITECTURE.md:26` dan `04-FUTURE-ARCHITECTURE.md:136` masih menulis
`12 Connect-REST`. Keduanya sudah diperbaiki menjadi **21**.

---

## 7. Hasil verifikasi lanjutan 2026-09-15 — `K-6`, `K-16`, `K-17`

Diminta Work Owner. Dihitung dan dibaca langsung dari export, bukan dari dokumen turunan.

### `K-6` — kedua angka yang bertentangan **sama-sama tidak cocok**

Hitungan langsung atas seluruh export (`*.xml`, `*.prc`, `*.fnc`), pola `@ASMD|@SIMASNET|@SMI|@OPJAVA|@PROD_ASM|@PROD_TKA`:

| Satuan hitung | Hasil |
|---|---:|
| Kemunculan mentah | **203** |
| Berkas/rule yang memakainya | **37** |
| Objek remote unik (`SKEMA.OBJEK@LINK`) | **49** |
| — di antaranya `@ASMD` | **34** |

| Angka dokumen | Cocok? |
|---|---|
| `16-RISK-ANALYSIS.md:115` — 64 pemakaian · 28 objek remote | **tidak** |
| `verifikasi-bukti-adr` `K-6` — 71 pemakaian · 22 objek | **tidak** |

**Kenapa ketiganya berbeda: satuan hitungnya berbeda, dan dua berkas mendominasi.**

| Berkas | Kemunculan |
|---|---:|
| `RDB List/GetTotalKlaimCreditValue_NGPW-SQL.xml` | **68** (seluruhnya `@SIMASNET`, 9 objek unik) |
| `Database/CONVERTJSONPRODUCTION.prc` | **51** |
| 35 berkas lainnya | 84 |

Menghitung **hanya `RDB List` tanpa berkas pencilan pertama** menghasilkan **66** — sangat dekat
dengan angka 64 yang tercatat. Dugaan terkuat: angka lama dihitung **sebelum export bertambah**
dari 2.167 menjadi 2.634 berkas, dan tidak pernah dihitung ulang.

**Dua akibat yang lebih penting daripada angkanya sendiri:**

1. **Inventaris `D-25` salah besar untuk `@SIMASNET`.** Tabel `D-25` mencatatnya **3×**; hitungan
   sebenarnya **71**, dan **68 di antaranya ada di satu rule** — `GetTotalKlaimCreditValue_NGPW`,
   sebuah kueri lintas database ke **9 objek remote**. `@SIMASNET` dicatat sebagai link kecil,
   padahal ia yang terberat kedua.
2. **`DATAMINING.GET_KURS_STANDARD@SIMASNET` adalah objek remote.** Kurs standar — yang menentukan
   konversi nilai klaim — diambil lintas database. Ini menyambung langsung ke `R-19` dan
   `TKT-F4-004` (isi `m_currencystandard` belum ada dari DBA).

**Status `K-6`: tetap BERTENTANGAN, kini dengan angka ketiga.** Penetapannya milik **Work Owner**,
setelah menyepakati **satuan hitung** — "pemakaian" bisa berarti kemunculan, rule, atau objek unik,
dan ketiganya menghasilkan angka yang jauh berbeda.

### `K-16` — terkonfirmasi sebagian, dan yang ditemukan lebih berat

`Activity/GetKomiteApproval-Act.xml` membandingkan:

| Properti | Dibandingkan dengan | Jumlah |
|---|---|---:|
| `pyWorkPage.Policy.Quotation.BusinessType` | `"PA"` · `"Travel"` · `"Bonding"` | 3 |
| `TempRDBSearchEmailKomite.pxResults(1).BUSINESS_CODE` | — | 3 |
| `tempAdj.ConvertAdjustmentValue` | `30000000` dan ambang berpindah menurut hostname | 2 |
| `OperatorID.pyUserIdentifier` | **nama satu orang, tertanam di rule** | 1 |
| `pyWorkPage.ClaimData.UserTeknis` | idem | 1 |

**Klaim `K-16` — "dipicu jabatan operator, bukan lini bisnis" — tidak sepenuhnya tepat: keduanya
dipakai.** Lini bisnis memang dibandingkan (3×).

**Yang jauh lebih berat, dan tidak tercatat di mana pun:** persetujuan komite bercabang pada
**identitas satu orang tertentu** yang namanya **tertanam di dalam rule**, bukan pada peran maupun
jabatan. Bila orang itu berpindah tugas atau keluar, perilaku komite berubah — dan tidak ada
dokumen yang menjelaskan mengapa. Penanganannya termasuk dalam penghapusan hardcode (`66 email ·
24 user ID · 8 ambang`), tetapi **akibatnya pada aturan komite belum pernah dibahas**.

Ambang `Rp 50.000.000` juga terbukti berpindah menjadi **3.500** lewat perbandingan hostname —
bukti langsung untuk `D-75`/`ADR-0030`. *(Nilai hostname-nya sengaja tidak disalin ke dokumen ini.)*

### `K-17` — **tidak terkonfirmasi**

`Activity/ValidasiSisaTSI-Act.xml` dibaca seluruhnya pada bagian pembandingnya. Aturannya:

```
sisa TSI = local.sumTSI − local.NilaiAkseptasiKlaim (+ local.NilaiSalvage)
tolak bila nilai adjustment melebihi sisa itu
```

**Tidak ada satu pun perbandingan terhadap `"PA"` di dalam rule itu.** Pemanggilnya
(`SetNilaiResikoSendiri-Act.xml`) hanya dijaga `.ConfirmationAnswer`, juga tanpa cabang PA.

Jadi klaim *"PA dikecualikan"* **tidak terbukti dari isi rule**. Bila pengecualian itu nyata, ia
terjadi karena **klaim PA tidak pernah sampai ke jalur ini** — dan itu pertanyaan **alur**, bukan
isi rule. Memastikannya menuntut penelusuran `Flow` dan pemanggil-pemanggilnya.

**Status `K-17`: turun dari "belum tuntas diverifikasi" menjadi TIDAK TERBUKTI pada tingkat rule;
sisa pemeriksaannya ada di tingkat alur.**

### Yang masih belum diverifikasi — 3 butir

`K-4` (router `ReceiveDocument.UserAdmin` — properti operator, bukan workbasket) ·
`K-14` (kunci duplikasi PA di SQL: `policyno` + `objectid` + `coverageid='10009'`, tanpa lokasi dan
tanpa cause of loss) · `K-15` (Group Panel `003` — hanya baris `(1)(1)` yang diperiksa).

Ketiganya menuntut pembacaan rule yang lebih dalam daripada pemeriksaan pembanding. **Tidak satu
pun saya nyatakan bersih.**

