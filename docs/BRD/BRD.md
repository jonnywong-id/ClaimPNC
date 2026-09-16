# Business Requirement Document

## Migrasi Aplikasi Claim PNC — Pega PRPC 8.3 → Golang + React

| | |
|---|---|
| **Dokumen** | Business Requirement Document (BRD) |
| **Aplikasi** | Claim PNC — Penanganan Klaim Asuransi Umum Lini Non-Motor |
| **Organisasi** | PT Asuransi Sinar Mas |
| **Versi** | **2.0** |
| **Tanggal** | **2026-09-14** |
| **Status** | Untuk direview dan disetujui manajemen |
| **Sumber** | [`../Steering/STEERING.md`](../Steering/STEERING.md) · 24 berkas [`../Steering/`](../Steering/) · **29 ADR** di [`../ADR/`](../ADR/) · discovery bersama pemilik bisnis 2026-09-07 … 2026-09-14 |

**Dasar penyusunan.** Seluruh isi dokumen ini diturunkan dari pembacaan menyeluruh atas export
rule XML aplikasi Pega — **2.634 berkas XML**, ditambah **55 `.prc` dan 8 `.fnc`** di folder
`Database/` yang diterima kemudian — mencakup 902 activity berisi 15.063 step, 652 rule SQL,
269 section, 74 harness, 70 when rule, dan 4 flow, ditambah discovery langsung dengan pemilik
bisnis. **Tidak ada isi dokumen ini yang berupa asumsi.** Setiap hal yang belum diketahui
ditandai terbuka, bukan ditebak.

> **Versi 2.0 — apa yang berubah.** Versi 1.0 disusun atas snapshot **2.167 rule** dan Decision
> Log `D-01`…`D-30`. Versi ini menyerap **42 keputusan baru** (`D-31`…`D-73`), **14 temuan
> verifikasi bukti**, dan **29 ADR**. Ringkasan lengkap perubahan ada di bab **Riwayat Revisi**
> pada `../Steering/STEERING.md`. Yang paling berdampak bagi pembaca BRD:
>
> | Hal | v1.0 | **v2.0** |
> |---|---|---|
> | Jumlah modul | 32 | **33** — `S-8` Perkakas Uji Kesetaraan masuk gelombang 1 |
> | Jumlah risiko | 15 | **19** — `R-16`…`R-19` ditambahkan |
> | Format nomor klaim | `PNCN-xxxx` | **`PNCN.YY.xxxx`** |
> | Aturan komite | matriks nilai × jenis bisnis | **kumulatif** — setiap jenjang yang ambangnya terlampaui ikut menyetujui |
> | Satuan izin | berbutir aksi | **menu**, tanpa pemisahan tugas formal |
> | Hardcode yang harus dihapus | 10 email · 4 user ID · 3 ambang | **66 · 24 · 8** |
> | Kode Status Klaim | `1142`–`1151` (10) | **`1134`–`1166` (33)** |
> | Bab 21.4 | mengikat 5 modul | **dicabut untuk `B-7`, `B-9`, `B-10`, `B-12`**; tetap untuk `B-5` |
> | Integrasi keluar | 12 Connect REST | **21** — 9 baru ditemukan; ditambah **4 Service REST masuk** (`D-73`) |
> | Cakupan tiket | 8 modul penuh + 25 stub | **33 modul, 102 tiket** (`D-73`) |

---

## Daftar Isi

| Bab | Judul | | Bab | Judul |
|---|---|---|---|---|
| 1 | [Executive Summary](#1-executive-summary) | | 13 | [User Journey](#13-user-journey) |
| 2 | [Business Background](#2-business-background) | | 14 | [Data Flow](#14-data-flow) |
| 3 | [Business Objectives](#3-business-objectives) | | 15 | [Integration](#15-integration) |
| 4 | [Current Process (As-Is)](#4-current-process-as-is) | | 16 | [Error Handling](#16-error-handling) |
| 5 | [Future Process (To-Be)](#5-future-process-to-be) | | 17 | [Security](#17-security) |
| 6 | [Scope](#6-scope) | | 18 | [Assumptions](#18-assumptions) |
| 7 | [Out of Scope](#7-out-of-scope) | | 19 | [Constraints](#19-constraints) |
| 8 | [Stakeholders](#8-stakeholders) | | 20 | [Risks](#20-risks) |
| 9 | [Functional Requirements](#9-functional-requirements) | | 21 | [Acceptance Criteria](#21-acceptance-criteria) |
| 10 | [Non Functional Requirements](#10-non-functional-requirements) | | 22 | [Success Metrics](#22-success-metrics) |
| 11 | [Business Rules](#11-business-rules) | | 23 | [Appendix](#23-appendix) |
| 12 | [Process Flow](#12-process-flow) | | | |

---

# 1. Executive Summary

**Claim PNC** adalah sistem penanganan klaim asuransi umum untuk lini bisnis **Non-Motor
(Non-MBU)** di Asuransi Sinar Mas — mencakup Personal Accident, Aneka, Marine Cargo, Travel,
Fire/Property, serta lini khusus Tenaga Kerja Asing, Sinarmas Penjaminan Kredit, Bonding, Heavy
Equipment, dan Contractors PM. Sistem ini menangani perjalanan klaim dari laporan kerugian oleh
tertanggung sampai pembayaran ganti rugi dan pemberitahuan ke koasuransi/reasuransi.

Sistem berjalan di atas **Pega PRPC 8.3** — platform berlisensi yang menyatukan antarmuka dan
logika bisnis dalam satu model rule, tanpa test otomatis, dan tanpa CI/CD. Project ini
memindahkan seluruh kapabilitasnya ke **Golang + React + PostgreSQL**.

### Ukuran pekerjaan

| Artefak | Jumlah |
|---|---|
| Rule XML | **2.634 berkas**, ditambah 55 `.prc` + 8 `.fnc` |
| Logika bisnis | **15.063 step** dalam 902 activity |
| Layar | **74 harness, 269 section** — 268 di antaranya bergrid |
| Query | **652 rule SQL** (534 KB) |
| Laporan | **56 report definition** |
| Tabel database | **177** objek di **9 skema** |
| Stored procedure & function | **70** |
| Integrasi | **21 Connect REST keluar** + **4 Service REST masuk** + **6 DB Link** (64 pemakaian) |
| Peran pengguna | **22 access group** mengendalikan **47 harness target** lewat **51 item menu** |

### Yang sudah selesai

Pemahaman bisnis dan teknis **sudah lengkap dan terdokumentasi**: 26 bab Steering, glossary
domain, domain model, 30 keputusan arsitektur bercatat, pembagian 34 modul beserta urutan
ketergantungannya, dan **19 risiko** teridentifikasi.

### Tiga hal yang perlu keputusan manajemen

Dokumen ini disusun untuk menyampaikan tiga hal yang tidak dapat diselesaikan tim pengembang:

**Pertama — lima modul yang mengurus nilai uang klaim tidak dapat diselesaikan saat ini.**
Logika bisnis untuk settlement, komite, PLA/DLA, akseptasi, dan salvage berada di dalam
**64 stored procedure dan function database yang source-nya tidak ada di export Pega, dan belum
pernah dilihat siapa pun di tim ini.** Ini bukan penundaan kecil — kelimanya adalah jantung
aliran nilai uang klaim (Risiko R-01).

**Kedua — ukuran keberhasilan yang dipilih belum dapat dibuktikan untuk kelima modul itu.**
Keberhasilan project ditetapkan sebagai **kesetaraan fungsional dengan Pega** (Bab 22). Tetapi
kesetaraan menuntut kita mengetahui apa hasil yang benar — dan untuk kelima modul itu, hasil yang
benar ditentukan oleh logika yang belum pernah dibaca. **Kesetaraan atas logika yang tidak
diketahui tidak dapat dibuktikan, hanya diduga** (Risiko R-14).

**Ketiga — jadwal dan ukuran pekerjaan tidak sepadan, dan tenggatnya tidak dipaksa pihak luar.**
Target akhir September 2026 telah ditetapkan manajemen dan dihormati sepenuhnya dalam dokumen
ini. Namun perkiraan berbasis ukuran terukur di atas adalah **9–15 bulan dengan tim 5–6 orang**
(Risiko R-05). Discovery juga mengonfirmasi bahwa pendorong migrasi adalah kemandirian teknologi
— **bukan** lisensi yang berakhir dan **bukan** biaya lisensi — sehingga **tidak ada tenggat
eksternal maupun penalti kontraktual** yang mengikat tanggal tersebut (Risiko R-13).

### Rekomendasi

**Mulai sekarang** pekerjaan yang tidak terhalang: modul fondasi, kerangka antarmuka, seluruh
modul baca-saja (inbox, pencarian, laporan), jalur klaim awal, perkakas uji kesetaraan, dan
pelatihan tim. Cukup untuk berbulan-bulan dan **sepenuhnya di dalam kendali tim.**

**Kejar sepuluh artefak** yang ditunggu dari DBA, Tim Pega, dan tim pemilik enam sistem sampai
masing-masing punya **tanggal komitmen**. Permintaan sudah dikirim; yang belum ada adalah
tanggal. Empat di antaranya hanya berupa query database — dan **dua di antaranya masing-masing
satu query** yang memblokir seluruh modul komite dan setiap layar berstatus.

**Jangan mulai dengan menebak** lima modul jantung nilai uang klaim. Pada sistem yang menangani
uang, kode yang kelihatan selesai tetapi tidak dapat dibuktikan setara lebih buruk daripada kode
yang belum dikerjakan — karena kesalahannya baru terlihat setelah uang berpindah.

---

# 2. Business Background

## 2.1 Bisnis yang dilayani

Claim PNC melayani penanganan klaim untuk seluruh lini bisnis **di luar kendaraan bermotor**.
Istilah **PNC** adalah sinonim **Non-MBU** (Non-Motor Business Unit) — bukan singkatan teknis.

Lini bisnis dikenali lewat kode **Group Panel**, yang menyetir hampir seluruh percabangan
aturan — validasi, routing, estimasi, komite, dan spreading:

| Group Panel | Lini | Karakter penanganan |
|---|---|---|
| `002` | Personal Accident (PA) | Perlu penilaian medis; ada peran Analyst Doctor dan RCL Dokter |
| `003` · `009` | Aneka | Paling beragam; Fac Out sering dipakai |
| `004` | Marine Cargo | Terkait pengangkutan, rute, dan kemasan |
| `005` | Travel | Batas waktu lapor paling longgar (90 hari) |
| `006` | Fire / Property | Terkait lokasi risiko dan okupasi bangunan |

Lini khusus lain: **TKA** (Tenaga Kerja Asing), **SPK** (Sinarmas Penjaminan Kredit / Asuransi
Kredit — wajib Nomor SLIK), **Bonding**, **HE** (Heavy Equipment), dan **Contractors PM**.

## 2.2 Karakter beban kerja

| Aspek | Angka |
|---|---|
| Pengguna aktif harian | **200–300** |
| Klaim baru | **Ribuan per bulan** |
| Data historis | **Puluhan juta baris** |
| Ketersediaan yang dituntut | **24/7** |

> **Profil bebannya: data besar, konkurensi rendah.** 200–300 pengguna bersamaan bukan beban
> berat. Yang berat adalah **query terhadap puluhan juta baris** — terutama inbox berkolom banyak
> dan laporan lintas periode. Seluruh upaya optimasi diarahkan ke sana.

## 2.3 Kenapa sistem ini perlu dipindahkan

Sistem berjalan di **Pega PRPC 8.3**, platform berlisensi yang:

- **Menyatukan antarmuka dan logika bisnis** dalam satu model rule, sehingga perubahan tampilan
  dan perubahan aturan tidak dapat dipisahkan.
- **Tidak memiliki test otomatis maupun CI/CD.** Tidak ada jaring pengaman ketika aturan diubah.
- **Menyebarkan satu aturan bisnis ke tiga tempat sekaligus** — activity, SQL, dan stored
  procedure — sehingga untuk mengetahui satu aturan, tim harus mencarinya di tiga tempat.
- **Mengikat organisasi pada keahlian yang semakin langka** dan pada batasan model rule vendor.

Kondisi itu sudah menimbulkan masalah nyata yang terbukti di dalam source, bukan dugaan:
nilai bisnis di-hardcode di dalam logika, alamat email penerima notifikasi tertanam di kode,
SQL dirangkai dari string tanpa parameter binding, zona waktu ditangani manual di puluhan tempat,
dan satu blok kode bertanda `// TESTING` **menimpa alamat email Underwriting yang asli dengan
email personal seorang tester — dan blok itu tetap berada di jalur produksi.**

Rincian lengkap: Bab 4.4.

---

# 3. Business Objectives

## 3.1 Tujuan utama

Dikonfirmasi pemilik project pada 2026-09-08 sebagai **satu-satunya pendorong utama**:

> ### Kemandirian teknologi dan maintainability
>
> Melepas ketergantungan pada platform berlisensi dan pada vendor, sehingga tim internal dapat
> mengembangkan dan merawat sistem sendiri tanpa keterbatasan model rule Pega.

## 3.2 Tujuan pendukung

| # | Tujuan | Dasar |
|---|---|---|
| O-1 | Memindahkan **seluruh** kapabilitas operasional ke platform yang **testable dan maintainable** — Golang + React + PostgreSQL | Tujuan utama |
| O-2 | **Menjaga kelangsungan bisnis.** Pega tetap menjadi sistem produksi dan tidak dinonaktifkan sampai sistem baru sepenuhnya divalidasi | Prinsip migrasi |
| O-3 | **Menutup technical debt dan celah keamanan yang sudah diketahui, bukan mereplikasinya** | Bab 4.4, Bab 17 |
| O-4 | **Menyatukan bahasa domain** agar satu aturan hidup di satu tempat dan dapat ditemukan | Bab 23.1 |
| O-5 | **Menyesuaikan arsitektur dengan skala aktual** — menghindari under-engineering maupun over-engineering | Bab 2.2 |
| O-6 | Menjadikan **seluruh nilai bisnis dapat diubah tanpa deploy** — ambang, penerima notifikasi, pemetaan peran | Bab 11.6 |

## 3.3 Yang secara eksplisit **bukan** tujuan

Ditegaskan agar tidak muncul sebagai ekspektasi yang tidak akan dipenuhi:

| Bukan tujuan | Penjelasan |
|---|---|
| **Penghematan biaya lisensi** | Bukan pendorong migrasi. **Tidak ada angka penghematan biaya** yang menjadi justifikasi project ini |
| **Memenuhi tenggat lisensi atau dukungan vendor** | Tidak ada lisensi yang berakhir dan tidak ada dukungan vendor yang habis. **Tidak ada tenggat eksternal** — lihat Bab 20, R-13 |
| **Perbaikan Turn Around Time klaim** | Bukan ukuran keberhasilan project ini (Bab 22). Karena itu tidak ada kewajiban mengambil baseline TAT Pega |
| **Reengineering proses bisnis** | Ini **reimplementasi**. Aturan bisnis yang ada dipertahankan — lihat Bab 7 |
| **Desain ulang antarmuka** | Alur, urutan langkah, dan tata letak layar mengikuti Pega agar pengguna tidak perlu pelatihan ulang |

---

# 4. Current Process (As-Is)

## 4.1 Gambaran sistem

Aplikasi Pega PRPC 8.3 dengan dua framework yang saling terikat: **GCNMFW** (Claim PNC,
633 activity) dan **GISFW** (polis/underwriting, 141 activity — dikembangkan tim lain).
Empat case type: Register, Open Protection, Receive Document, Komite.

## 4.2 Struktur database

| Schema | Peran | Tabel utama |
|---|---|---|
| `POOLDATA` | Data bisnis inti | `T_CLAIM_PNC` (124×), `GCNM_PROGRESS_CLAIM` (119×), `T_CLAIM_ADJUSTMENT` (92×), `M_KOMUNIKASI_PNC` (46×), `T_CLAIM_KOMITE_LIST` (33×) |
| `DATAPEGA` | **Tabel milik engine Pega** | `PC_ASM_FW_GCNMFW_WORK` (**116×**), `PC_ASSIGN_WORKLIST` (18×), `PR_OPERATORS`, `PC_LINK_ATTACHMENT` |
| `GENERAL` | Storage & token | `T_STORAGE_IMAGE`, `MST_BUKA_PROTEKSI` |
| `GL` · `COLLECTION` · `MBU` · `ANEKA` · `HRDASM` | Domain lain | Diakses lewat DB Link |

**Dua model penyimpanan hidup berdampingan.** Data klaim yang sama disimpan **dua kali** — sebagai
tabel relasional `POOLDATA` dan sebagai dokumen JSON (`JSON_KLAIM`, `JSON_POLIS`, 222 pemakaian
`JSON_VALUE`/`JSON_TABLE` di 29 rule). Sinkronisasi dilakukan stored procedure, dan **tidak ada
mekanisme yang memeriksa apakah keduanya konsisten.**

## 4.3 Empat konsep status yang berbeda

Dikonfirmasi pemilik bisnis: keempatnya **memang berbeda**, bukan duplikasi, dan semuanya
dipertahankan.

| Konsep | Sistem lama | Isi |
|---|---|---|
| **Status Proses** | `StatusWork` | Posisi klaim dalam alur kerja: berjalan, selesai, ditolak |
| **Status Klaim** | `StatusClaim`, **33 kode `1134`–`1166`** | Status bisnis klaim — arti tiap kode **sudah diketahui** sejak master `v_sts_claim` diterima |
| **Flag Klaim** | `ClaimStatus` (`0`/`1`) | Penanda biner |
| **Status Posisi Progres** | `StatusPosisi` | `On Progress` / `Done`, dicatat terpisah dari alur utama |

## 4.4 Technical debt yang terbukti dari source

Delapan temuan. Kolom terakhir menunjukkan bahwa setiap temuan **sudah punya jawaban desain** —
tidak ada yang dibiarkan.

| # | Temuan | Bukti dari source | Jawaban |
|---|---|---|---|
| 1 | **Kunci teknis Pega bocor ke data bisnis** | `CLAIMID = 'ASM-FW-GCNMFW-WORK ' \|\| no_klaim` — data bisnis tidak bisa dibaca tanpa mengetahui konvensi internal Pega | Nomor klaim `PNCN.YY.xxxx` |
| 2 | **Alias kolom menyesatkan** | `BUSINESSNAME AS "NOPOLIS"` (nama bisnis dialiaskan jadi nomor polis), `NOPOLIS AS "NoKTP"`, `picteknik AS "UserAdmin"`. Membaca query berarti menebak | Penamaan ulang menyeluruh |
| 3 | **Nilai bisnis di-hardcode** | **66** alamat email, **24** user ID, **8** ambang komite, dan **3 hostname server sebagai penentu perilaku bisnis** | Seluruhnya jadi master data |
| 4 | **Blok `// TESTING` di jalur produksi** | `InputRegister_act` **sub-step 2 dan 6 di dalam step 53** — sub-step 2 menetapkan email UW asli, **sub-step 6 menimpanya dengan email personal tester** — dan tetap ada di produksi | Dilarang; ditolak di code review |
| 5 | **Zona waktu ditangani manual** | `+7 jam` ditambahkan di puluhan tempat, ada activity khusus `Set7Hours`. **Satu tempat yang lupa memanggilnya menggeser tanggal tanpa terdeteksi** — dan pada aturan "Tanggal Lapor ≤ DOL + 7 hari", pergeseran itu **mengubah hasil validasi** | Satu modul waktu tunggal |
| 6 | **SQL dirangkai dari string** | `WHERE claimid = ... {ASIS:InputData.CARI4}`. Bahkan potongan klausa SQL disimpan sebagai nilai property. **Risiko SQL injection sekaligus penghalang portabilitas** | Parameter binding wajib |
| 7 | **Duplikasi masif per lini bisnis** | Pola `Browse*`/`Insert*` digandakan `_AsuransiKredit`, `_AutoClaim`, `_Travel`, `_Kredit_PA`. Satu perubahan aturan harus diterapkan di empat tempat — **dan sering hanya diterapkan di sebagian** | Modul bersama |
| 8 | **Salah ketik dipertahankan di nama rule** | `Broswse*` (11 rule), `Complience`, `Proccedure`, `SALAVAGEDOCUMENT` | Penamaan ulang |

> **Temuan #4 dan #6 adalah yang paling serius.** #4 berarti notifikasi Underwriting untuk
> kerugian besar **kemungkinan tidak pernah sampai ke penerima yang benar** di produksi.
> #6 adalah celah keamanan yang nyata.

## 4.5 Sistem sumber masih aktif berubah

| Tahun perubahan terakhir activity | 2018 | 2020 | 2023 | **2024** | 2025 | **2026** |
|---|---|---|---|---|---|---|
| Jumlah | 127 | 108 | 97 | **180** | 70 | **124** |

**124 activity diubah pada 2026.** Pega adalah **sasaran bergerak** — perubahan yang dibuat tim
Pega selama migrasi tidak otomatis ikut ke sistem baru.

---

# 5. Future Process (To-Be)

## 5.1 Bentuk sistem baru

| Lapisan | Teknologi | Keputusan |
|---|---|---|
| Frontend | **React 18 + TypeScript + Vite**, TanStack Table / AG Grid. **SPA murni** disajikan sebagai berkas statis oleh binary Go — tanpa runtime Node.js di production | Pemilik project |
| Backend | **Go 1.22+**, `chi`, `database/sql`, **tanpa ORM**. Modular monolith empat lapisan | Pemilik project |
| Database sasaran | **PostgreSQL 17 atau lebih baru** — persyaratan teknis **mengikat**, bukan preferensi | Pemilik project |
| Database sementara | Oracle 19c, dengan **satu set SQL portabel** yang berjalan di keduanya | Pemilik project |
| Deployment | **2 VM on-premise** di belakang load balancer, rolling deployment | Pemilik project |

> **Kenapa PostgreSQL 17 mengikat.** Hanya versi 17 ke atas yang mendukung fungsi SQL/JSON
> standar `JSON_TABLE`, `JSON_VALUE`, dan `JSON_QUERY` **dengan sintaks yang sama seperti
> Oracle** — sehingga seluruh 222 query JSON menjadi portabel apa adanya. Pada PostgreSQL 16 ke
> bawah, 29 rule harus ditulis ulang dan keputusan "satu set SQL portabel" **tidak dapat
> dijalankan sama sekali.**

## 5.2 Perubahan mendasar dari sistem lama

| Aspek | Sistem lama | Sistem baru |
|---|---|---|
| Logika bisnis | Tersebar di activity, SQL, **dan stored procedure** | **Seluruhnya di Go.** Database menjadi penyimpanan murni; tidak ada pemanggilan stored procedure |
| Data polis | Dibaca berulang dari domain lain | **Snapshot dibekukan saat registrasi** — klaim kebal perubahan polis dan tidak bergantung ketersediaan sistem tim lain saat runtime |
| Akses lintas database | 64 pemakaian **DB Link** ke 6 database | **Pemanggilan API** ke sistem pemilik data — kopling tersembunyi diganti kontrak eksplisit |
| Nilai bisnis | Di-hardcode di dalam logika | **Master data**, dapat diubah tanpa deploy |
| Nomor klaim | `ASM-FW-GCNMFW-WORK PNC-xxxx` | **`PNCN.YY.xxxx`** — asal dan tahun setiap klaim langsung terbaca tanpa tabel pemetaan |
| Otorisasi | Bergantung pada **penyembunyian menu** | **Diperiksa di setiap endpoint backend** |
| Jejak audit | Tidak dirancang | **Append-only, dirancang sejak awal**, ditegakkan lewat hak akses database |
| Waktu | GMT + 7 jam manual di puluhan tempat | **UTC, konversi WIB di satu tempat** |
| Laporan | Engine reporting Pega | **Engine PDF/Excel/CSV dibuat sendiri di Go**, export besar asinkron |
| Test | Tidak ada | **Setiap aturan bisnis punya test bernama kalimat bisnis** |

## 5.3 Strategi peralihan — Strangler Fig

Pega dan aplikasi Go **berjalan paralel**; modul dialihkan satu per satu. Lima prinsip mengikat:

| # | Prinsip |
|---|---|
| **P-1** | **Satu tabel hanya boleh ditulis satu sistem.** Modul yang sudah pindah memiliki tabelnya, Pega hanya membaca — dan sebaliknya. **Tidak ada sinkronisasi dua arah**, karena dua sistem yang sama-sama menulis dengan aturan validasi berbeda menghasilkan konflik data yang hampir mustahil dilacak |
| **P-2** | Batas modul mengikuti batas kepemilikan tabel — sehingga urutan migrasi **tidak bebas** |
| **P-3** | **Klaim yang sedang berjalan tidak berpindah sistem di tengah jalan.** Klaim `PNC-xxxx` diselesaikan di Pega; klaim baru `PNCN.YY.xxxx` dimulai di Go. Ini menghindari kelas bug terburuk dalam migrasi bertahap |
| **P-4** | **Migrasi skema selalu backward-compatible** — karena tuntutan 24/7 membuat versi lama dan baru berjalan bersamaan terhadap skema yang sama |
| **P-5** | **Perilaku dipertahankan lebih dulu, diperbaiki kemudian.** Hasil yang benar adalah hasil yang sama dengan Pega, kecuali empat perbaikan yang diputuskan eksplisit. Alasannya: bila hasil berbeda, kita harus bisa memastikan itu **bug**, bukan perbaikan yang tidak tercatat |

## 5.4 Tahapan peralihan

| Tahap | Isi | Perubahan bagi pengguna |
|---|---|---|
| **0 — Persiapan** | Mengejar 10 artefak dari pihak luar; menyiapkan lingkungan; pelatihan tim | Tidak ada — **tetapi memblokir semuanya** |
| **1 — Fondasi** | Kerangka aplikasi, akses data, identitas, master data, waktu, kerangka SPA, pustaka komponen | Tidak ada. Pega masih menangani seluruh pekerjaan |
| **2 — Baca dulu** | Inbox, pencarian, detail, laporan | Layar baca berpindah ke sistem baru |
| **3 — Jalur klaim inti** | Polis & snapshot, registrasi, objek & coverage, spreading, settlement, penugasan, receive document, dokumen | **Klaim baru `PNCN.YY.xxxx` mulai dibuat di sistem baru** |
| **4 — Persetujuan** | Komite, akseptasi & pembayaran, RCL/PUCL/compliance, survey, open protection | Persetujuan dan penyelesaian berpindah |
| **5 — Nilai & pihak luar** | PLA/Pre-DLA/DLA, salvage & recovery, notifikasi, integrasi eksternal | Pemberitahuan ke koasuransi/reasuransi berpindah |
| **6 — Laporan & penutup** | 56 laporan, dashboard, layar master data, scheduler | Seluruh laporan berpindah |
| **7 — Penonaktifan Pega** | Hanya setelah seluruh klaim `PNC-xxxx` selesai. **Pega dijadikan read-only lebih dulu selama satu periode pengamatan**, baru dimatikan | Pega tidak lagi dipakai |

> **Tahap 2 didahulukan secara sengaja, dengan tiga alasan sekaligus:** modul baca tidak
> melanggar P-1 karena tidak menulis apa pun; ia menjadi pembuktian arsitektur yang nyata; dan
> **tim belajar Go dan React pada pekerjaan yang kesalahannya tidak merusak data.**

---

# 6. Scope

## 6.1 Yang masuk scope

**Seluruh rule yang ada di export XML ini** — **2.634 berkas XML** ditambah 55 `.prc` dan 8
`.fnc`, tanpa pemangkasan. Ditegaskan pemilik project: *"semua modul sudah harus pindah."*

> **Catatan v2.0.** Angka ini **bergerak dua kali** selama analisis: 2.167 → 2.389 → 2.634 berkas.
> Dan ia tetap bukan batas atas — **±242 rule dirujuk aplikasi tetapi tidak ada di export**
> (`R-16`). Scope yang sebenarnya karena itu lebih besar daripada yang terlihat dari jumlah berkas.

| Kelompok | Cakupan |
|---|---|
| **Modul fondasi** (5) | Kerangka aplikasi · akses data · identitas & akses · master data · waktu & zona waktu |
| **Modul bisnis inti** (14) | Polis & snapshot · registrasi klaim · objek & coverage · spreading reasuransi · estimasi & settlement · penugasan & inbox · komite · survey & adjuster · PLA/Pre-DLA/DLA · akseptasi & pembayaran · RCL/PUCL/compliance · salvage & recovery · open protection · receive document |
| **Modul pendukung** (8) | Dokumen & lampiran · laporan & export · notifikasi · integrasi eksternal · jejak audit · penjadwalan · dashboard bisnis · **perkakas uji kesetaraan** |
| **Frontend** (6) | Kerangka SPA · pustaka komponen · layar inbox · layar transaksi · layar laporan · layar master data |
| **Yang belum ada sama sekali** | Tabel autentikasi & otorisasi · tabel penugasan · tabel pengganti tabel engine Pega · modul master data · jejak audit |

**Permukaan yang harus dibangun:** 74 layar, 269 komponen, 56 laporan, **21 integrasi keluar**,
6 API pengganti DB Link, dan 15.063 step logika bisnis.

## 6.2 Data yang masuk scope

| Kegiatan | Cakupan |
|---|---|
| Data klaim historis | **Tidak dipindahkan.** Tetap di tabel `POOLDATA` yang sama — satu database bersama |
| Data di tabel engine Pega | **Dipindahkan** ke tabel baru milik aplikasi: `PC_ASM_FW_GCNMFW_WORK` (dibaca 116 rule), `PC_ASSIGN_WORKLIST`, `PC_ASSIGN_WORKBASKET`, `PR_OPERATORS`, `PC_LINK_ATTACHMENT`, `PC_DATA_WORKATTACH`, `PR_SYS_LOCKS` |
| Klaim warisan `PNC-xxxx` | **Tetap dapat dibaca dan diproses**, hanya tidak lagi dibuat baru |
| Perpindahan Oracle → PostgreSQL | **Langkah terpisah setelah aplikasi stabil.** Tanggalnya belum ditentukan |

---

# 7. Out of Scope

| # | Di luar scope | Alasan |
|---|---|---|
| 1 | **Reengineering proses bisnis** | Ini reimplementasi. Aturan bisnis dipertahankan kecuali ada keputusan eksplisit sebaliknya |
| 2 | **Desain ulang antarmuka** | Alur, urutan langkah, dan tata letak layar mengikuti Pega agar pengguna tidak perlu pelatihan ulang. Perbaikan UX ditunda ke pasca-migrasi |
| 3 | **GISFW — domain polis dan underwriting** | Tidak dimiliki project ini; sedang dikembangkan tim lain. Claim PNC hanya menyimpan snapshot polis |
| 4 | **Perpindahan database ke PostgreSQL** | Tidak dilakukan bersamaan dengan migrasi aplikasi. Aplikasi Go berjalan di Oracle lebih dulu |
| 5 | **Pembangunan 6 API pengganti DB Link** | Dibangun **tim pemilik masing-masing sistem**, bukan tim ini. Yang masuk scope adalah konsumsinya |
| 6 | **Penulisan ulang stored procedure sebagai PL/pgSQL** | Logikanya naik ke Go; database menjadi penyimpanan murni |
| 7 | **Migrasi massal data klaim historis** | Satu database bersama membuatnya tidak perlu |
| 8 | **Tools BI eksternal** (Metabase, Superset, Power BI) | Engine PDF/Excel/CSV dibuat sendiri di Go |
| 9 | **Sharding, microservices, message broker, autoscaling, cache terdistribusi (Redis)** | Menambah kerumitan operasional nyata tanpa menyelesaikan masalah yang ada pada skala 200–300 pengguna dan 2 VM |
| 10 | **Perbaikan yang tidak diputuskan eksplisit** | Dicatat sebagai Future Enhancement. Bila hasil berbeda dari Pega, kita harus bisa memastikan itu bug — bukan perbaikan yang tidak tercatat (P-5) |
| 11 | **Baseline dan perbaikan Turn Around Time klaim** | Bukan ukuran keberhasilan project ini (Bab 22) |
| 12 | **Aplikasi mobile native** | Frontend responsive sudah memenuhi kebutuhan surveyor |

---

# 8. Stakeholders

> **Catatan yang harus ditindaklanjuti sebelum presentasi.** Atas permintaan pemilik project,
> tabel ini memakai **jabatan generik tanpa nama orang**. Kolom nama dan tanda tangan **wajib
> diisi manual** sebelum dokumen dipresentasikan — tanpa nama, tidak ada pihak yang benar-benar
> terikat pada komitmen yang tercatat di kolom "Yang dibutuhkan dari mereka".

## 8.1 Pengambil keputusan

| Peran | Kepentingan | Yang dibutuhkan dari mereka |
|---|---|---|
| **Sponsor Project** | Keberhasilan dan pembiayaan project | Persetujuan BRD · keputusan atas tiga hal di Bab 20.4 |
| **Business Owner Claim PNC** | Kelangsungan operasional klaim | Persetujuan cutover per modul · penetapan ambang komite · alokasi waktu user untuk UAT |
| **Manajemen Teknologi Informasi** | Arsitektur dan sumber daya teknis | Persetujuan arsitektur · penetapan jadwal · penyediaan tim |

## 8.2 Pemilik proses bisnis (pengguna sistem)

22 access group; yang paling terdampak:

| Peran | Tanggung jawab | Modul yang dipakai |
|---|---|---|
| **PncAdmin** · **PncManagerAdmin** | Registrasi klaim, input data, unggah dokumen, persetujuan tingkat admin | Registrasi, dokumen, inbox |
| **PncPICTeknik** | Penanggung jawab teknis sesuai lini bisnis | Estimasi, settlement |
| **PNCKomite** · **PNCKomiteTeknik** | Persetujuan nilai klaim berjenjang | Komite |
| **PncPLADLA** | Pemberitahuan ke koasuransi/reasuransi | PLA/Pre-DLA/DLA |
| **PNCSurveyor** | Survei lapangan, unggah foto dan hasil survei | Survey — **dari tablet/HP di lapangan** |
| **PncAnalystDoctor** | Penilaian medis klaim PA | RCL Dokter, analyst doctor |
| **PncComplience** · **PncInvestigator** | Pemeriksaan kepatuhan, penyelidikan klaim mencurigakan | Compliance, investigator |
| **PncRCLPUCL** | Penanganan penolakan dan proses ulang klaim | RCL/PUCL |
| **PncReceive** · **PncManagerReceive** | Penerimaan dokumen fisik | Receive document |
| **CaseManager** | Pengawasan lintas kasus | Dashboard, laporan |

## 8.3 Pihak eksternal yang menjadi jalur kritis

**Ini bagian terpenting dari bab ini.** Sepuluh dari dua belas tindakan yang memblokir project
bergantung pada pihak-pihak di bawah — dan **waktu tunggunya tidak dapat dipercepat dengan
menambah developer.**

| Pihak | Yang dibutuhkan dari mereka | Memblokir | Risiko |
|---|---|---|---|
| **Tim DBA** | Source **64 procedure & function** · **DDL lengkap** + statistik ukuran tabel · isi master **`V_STS_CLAIM`** · isi **`POOLDATA.EMAILKOMITE`** | **5 modul jantung nilai uang klaim** · desain skema · seluruh layar berstatus · modul komite | **R-01, R-06, R-08** |
| **Tim Pega** | Export **Rule-Agent / Queue Processor** · **3 router penugasan** · **40 activity** yang hilang · kesepakatan **pembekuan perubahan** | Scheduler · modul penugasan · notifikasi · SLIK OJK | **R-02, R-04, R-07, R-09** |
| **Tim pemilik 6 sistem** | Kontrak dan ketersediaan **6 API pengganti DB Link** | Integrasi eksternal · jalur registrasi (`GetIDCabang`) | **R-03** |
| **Tim GISFW** | Struktur data polis untuk snapshot | Modul polis & snapshot | — |
| **Tim Compliance** | Lama **retensi data audit** | Strategi arsip dan partisi | **D-28** |
| **Tim Infra** | Angka **RPO / RTO** · penyediaan 2 VM · load balancer | Strategi deployment dan pemulihan | **D-29** |
| **Tim Kasir** | Kontrak data rekening dan permintaan pembayaran | Modul akseptasi & pembayaran | — |

## 8.4 Pelaksana

| Peran | Tanggung jawab | Catatan |
|---|---|---|
| **Tim Pengembang** | Pembangunan seluruh modul backend dan frontend | **Developer Pega internal** yang harus mempelajari Go, React, dan TypeScript sekaligus — ini batasan utama seluruh keputusan teknis (Bab 19) |
| **Technical Architect / Tech Lead** | Penegakan coding standards, review layar pertama tiap jenis | Wajib pada masa awal |

---

# 9. Functional Requirements

Daftar lengkap bernomor: [`requirement-summary.md`](../requirement-summary.md) §2.
Di sini disajikan ringkasannya beserta status kesiapan.

**Penanda:** `✅` siap diimplementasikan · `⚠️` menunggu artefak pihak luar · `🆕` tambahan dari discovery 2026-09-08

## 9.1 Modul fondasi — wajib selesai lebih dulu

| ID | Requirement | Status |
|---|---|---|
| FR-F1 | **Kerangka Aplikasi** — konfigurasi tiga lapis, logging terstruktur, error handling, health check, graceful shutdown, penyajian SPA | ✅ |
| FR-F2 | **Akses Data** — connection pool (terpisah untuk laporan), transaksi, satu set SQL portabel, migrasi skema versioned | ✅ |
| FR-F3 | **Identitas & Akses** — login via HCC/HCQ, session/token milik aplikasi, tabel peran & izin menu **yang belum ada**, middleware otorisasi | ✅ |
| FR-F4 | **Master Data** — **≥29 kelompok master** + layar pengelolanya. Menghapus **seluruh** hardcode: **66 email, 24 user ID, 8 ambang komite, 3 hostname** | ✅ |
| FR-F5 | **Waktu & Zona Waktu** — penyimpanan UTC, konversi WIB di **satu** tempat | ✅ |

> **FR-F5 kecil tetapi tidak boleh ditunda.** Bila dikerjakan belakangan, **seluruh aturan
> tanggal harus ditulis ulang.**

## 9.2 Modul bisnis inti

| ID | Requirement | Ukuran lama | Status |
|---|---|---|---|
| FR-B1 | **Polis & Snapshot** | 18 activity | ✅ |
| FR-B2 | **Registrasi Klaim** — gerbang validasi terberat, **137 step** | 22 activity | ✅ |
| FR-B3 | **Objek & Coverage** | 44 activity | ✅ |
| FR-B4 | **Spreading Reasuransi** — total 100%, Fac Out, Ex-Gratia | bagian dari 69 | ✅ |
| FR-B5 | **Estimasi & Settlement** | 29 activity | ⚠️ **R-01** |
| FR-B6 | **Penugasan & Inbox** — Worklist & Workbasket, routing, penguncian | 30 activity | ⚠️ **R-04** |
| FR-B7 | **Komite** — 1–4 level, matriks nilai × jenis bisnis | 34 activity | ⚠️ **R-01, D-14** |
| FR-B8 | **Survey & Adjuster** — foto lapangan, wajib nyaman di tablet/HP | 30 activity | ✅ |
| FR-B9 | **PLA / Pre-DLA / DLA** | 69 gabungan | ⚠️ **R-01** |
| FR-B10 | **Akseptasi & Pembayaran** — nomor akseptasi, transfer kasir, LOD | 31 activity | ⚠️ **R-01** |
| FR-B11 | **RCL / PUCL / Compliance** | 6 + jalur alur | ✅ |
| FR-B12 | **Salvage & Recovery** — balai lelang, virtual account | 26 activity | ⚠️ **R-01** |
| FR-B13 | **Open Protection** | 12 activity | ✅ |
| FR-B14 | **Receive Document** | 8 activity | ✅ |

## 9.3 Modul pendukung

| ID | Requirement | Status |
|---|---|---|
| FR-S1 | **Dokumen & Lampiran** — **tiga mekanisme penyimpanan lama disatukan menjadi satu jalur** | ✅ |
| FR-S2 | **Laporan & Export** — 56 laporan + engine PDF/Excel/CSV sendiri; export besar **asinkron dan streaming** | ✅ |
| FR-S3 | **Notifikasi** — berbasis **peristiwa domain**, penerima dari master data | ⚠️ **R-07** |
| FR-S4 | **Integrasi Eksternal** — **21 Connect REST keluar** + 6 API pengganti DB Link + **4 layanan REST masuk** yang baru ditemukan | ⚠️ **R-03** · lingkup bertambah |
| FR-S5 | **Jejak Audit** — append-only; **satu-satunya kontrol pengimbang** karena tidak ada pemisahan tugas (D-59) | ⚠️ **tanpa baseline Pega** — lihat §21.5 |
| FR-S6 | **Penjadwalan** — **5 job terjadwal + 1 agent**, seluruhnya terverifikasi | ✅ artefak · ⚠️ rancangan dua instans belum diputuskan |
| FR-S7 | **Dashboard & Monitoring Bisnis** | ✅ |
| FR-S8 🆕 | **Perkakas Uji Kesetaraan** — menjalankan permintaan yang sama ke Pega dan ke Go atas data historis yang sama, membandingkan hasilnya, **dan mengklasifikasikan setiap selisih** terhadap 13 butir `P-5` | ⚠️ **Pega staging belum dikonfirmasi** — lihat §21.6 |

> **FR-S8 adalah tambahan dari discovery 2026-09-08.** Ia menjadi modul karena keberhasilan
> project diukur sebagai kesetaraan fungsional dengan Pega (Bab 22), dan uji kesetaraan otomatis
> adalah **gerbang pertama cutover setiap modul** (Bab 21). **Tanpa perkakasnya, ukuran
> keberhasilan yang dipilih tidak dapat dibuktikan** — dan bila dibangun belakangan, cutover
> modul pertama akan tertahan olehnya.

## 9.4 Frontend

| ID | Requirement | Status |
|---|---|---|
| FR-U1 | **Kerangka SPA** | ✅ |
| FR-U2 | **Pustaka Komponen** — tabel baku (**median 6 kolom**) paginasi server-side, form baku, unggah berkas | ✅ |
| FR-U3 | **Layar Inbox** — **26** harness bernama *Inbox* | ✅ |
| FR-U4 | **Layar Transaksi** — registrasi, estimasi, survei, komite, akseptasi | ✅ |
| FR-U5 | **Layar Laporan** — 56 laporan + unduhan | ✅ |
| FR-U6 | **Layar Master Data** — pengelolaan **≥29 kelompok master** | ✅ |

> **FR-U2 adalah investasi paling menentukan di frontend.** 268 dari 269 section memakai pola
> grid yang sama. Satu komponen tabel baku yang benar dipakai ratusan kali, dan perbaikan bug di
> satu tempat memperbaiki seluruh layar. Melewatkannya berarti **268 implementasi tabel yang
> berbeda-beda** — persis kegagalan yang harus dicegah pada tim yang sedang belajar.

## 9.5 Alur kerja yang wajib didukung

| ID | Requirement | Status |
|---|---|---|
| FR-W1 | **Register Flow** — 23 shape, 13 assignment, 7 decision | ✅ |
| FR-W2 | **Transisi lateral wajib diizinkan** — 11 *ticket* memungkinkan lompatan langsung ke tahap mana pun | ✅ |
| FR-W3 | **Open Protection Flow** | ✅ |
| FR-W4 | **Receive Document Flow** | ✅ |
| FR-W5 | **Komite Flow** — berjenjang sampai 4 level | ⚠️ **D-14** |
| FR-W6 | **Empat konsep status dipertahankan terpisah** | ⚠️ **R-06** |

---

# 10. Non Functional Requirements

Daftar lengkap: [`requirement-summary.md`](../requirement-summary.md) §3.

## 10.1 Performa — target persentil 95

| Operasi | Target |
|---|---|
| Buka layar sederhana | **< 1 detik** |
| Inbox dan pencarian | **< 3 detik** terhadap data produksi penuh |
| Simpan registrasi klaim | **< 3 detik** termasuk seluruh validasi |
| Perhitungan spreading | **< 1 detik** |
| Laporan interaktif | **< 10 detik** |
| Export besar PDF/Excel/CSV | **Asinkron** — tidak menahan pengguna |
| Pemanggilan sistem eksternal | Batas waktu **30 detik**; kegagalan ditangani, bukan menggantung |

**Hambatan yang sebenarnya**, diurutkan dari yang paling mungkin menjadi masalah:

| # | Hambatan | Penanganan |
|---|---|---|
| 1 | Query inbox terhadap puluhan juta baris | Index tepat + **keyset pagination** + index parsial |
| 2 | Laporan lintas periode panjang | Pool koneksi terpisah + eksekusi asinkron + streaming |
| 3 | Query N+1 saat memuat objek dan coverage | Muat sekaligus per batch |
| 4 | Export besar dimuat seluruhnya ke memori | Streaming baris demi baris |
| 5 | Pemanggilan sistem eksternal yang lambat | Batas waktu, circuit breaker |
| — | Jumlah permintaan per detik | **Bukan hambatan** pada 200–300 pengguna |

## 10.2 Ketersediaan

| Aspek | Target | Konsekuensi yang mengikat |
|---|---|---|
| Ketersediaan | **24/7**, termasuk saat deployment | Minimal 2 instance di belakang load balancer |
| Downtime terencana | **Nol** untuk deployment aplikasi | Migrasi skema **wajib backward-compatible**; kolom dihapus dalam dua tahap |
| Downtime tak terencana | Sekecil mungkin | Aplikasi **wajib stateless**; kegagalan satu instance tidak menghentikan layanan |
| Pemulihan bencana | Mengikuti standar korporat | ⚠️ **Angka RPO/RTO belum ada** |

> **Perlu diperiksa tim infra.** Standar backup korporat menjawab *pemulihan saat bencana*,
> sedangkan tuntutan 24/7 menjawab *operasional harian*. **Keduanya berbeda**, dan perlu
> dipastikan standar yang ada memang sejalan dengan tuntutan 24/7 — bila tidak, selisihnya harus
> diangkat.

## 10.3 Auditabilitas

Kewajiban, dirancang sejak awal — bukan tambahan.

| Requirement |
|---|
| Setiap perubahan bernilai bisnis tercatat permanen: **siapa, kapan, nilai sebelum, nilai sesudah** |
| Cakupan minimal: nilai estimasi, nilai settlement, akseptasi, keputusan komite, penolakan (RCL), proses ulang (PUCL), perubahan status klaim, pembayaran |
| Data audit **append-only**, ditegakkan lewat **hak akses database** — aplikasi hanya diberi `INSERT` dan `SELECT`, sehingga **tidak ada jalur di dalam aplikasi yang bisa menghapus jejaknya sendiri** |
| ⚠️ **Lama retensi belum ditentukan** — harus dikonfirmasi Compliance sebelum go-live |

## 10.4 Maintainability

Untuk tim yang sedang mempelajari tiga teknologi sekaligus, **maintainability lebih penting
daripada kecanggihan.**

| Ukuran | Batas | Tindakan bila terlampaui |
|---|---|---|
| Panjang fungsi | ~80 baris | Pecah |
| Panjang berkas | ~400 baris Go · ~200 baris komponen React | Pecah |
| Pelanggaran aturan ketergantungan antar lapisan | **Nol** | **Merge diblokir** — ditegakkan otomatis, bukan oleh kesepakatan |
| Cakupan test aturan bisnis | Seluruh aturan terdokumentasi punya test **bernama kalimat bisnis** | Lengkapi sebelum merge |
| Dependensi pihak ketiga | Ditinjau setiap penambahan | Butuh alasan tertulis |

## 10.5 Antarmuka

| Requirement |
|---|
| Frontend **desktop-first namun responsive** |
| Layar surveyor — unggah foto dan input hasil survei — wajib nyaman di tablet/HP dan **tahan jaringan lambat** |
| Alur kerja, urutan langkah, penempatan field, dan tata letak layar **mengikuti Pega** |
| Pesan kesalahan **berbahasa Indonesia** dan **menjelaskan cara memperbaiki** |

---

# 11. Business Rules

Daftar lengkap bernomor: [`requirement-summary.md`](../requirement-summary.md) §4.
Aturan di bawah adalah yang **paling sering menolak input pengguna**, sehingga wajib benar.

## 11.1 Aturan tanggal

| Aturan | Berlaku untuk |
|---|---|
| Tanggal Kejadian (DOL) harus di dalam periode polis | Semua |
| DOL boleh sampai **30 hari** setelah polis berakhir | Bonding |
| Tanggal cetak boleh sampai **90 hari** setelah polis berakhir | Travel & PA |
| Tanggal Lapor ≤ DOL + **7 hari** | Semua **kecuali PA** |
| Tanggal Terima Dokumen ≤ DOL + **90 hari** | Travel |

> **Urutan wajib: DOL ≤ Tanggal Lapor ≤ Tanggal Terima Dokumen ≤ hari ini.**

## 11.2 Aturan duplikasi

| Aturan |
|---|
| Klaim **ditolak** bila sudah ada klaim lain dengan **Nomor Polis + Objek + Lokasi** yang sama |
| Untuk **PA**, diperketat: **Nomor Polis + Objek + Penyebab Kerugian `12002` + Lokasi** |
| Pesan penolakan **menyertakan nomor klaim yang sudah ada** |

## 11.3 Aturan reasuransi

| Aturan |
|---|
| **Total spreading wajib 100%**, diuji `ROUND(SUM(share),4) BETWEEN 99.9999 AND 100.0001` (D-51). Bila tidak, **submit ditolak** |
| Bila ada spreading **Fac Out**, data Fac Offer **wajib ada** |
| Untuk **Group Panel `003`**, Fac Offer wajib menyertakan Object Name |
| Bila klaim ditandai **Ex-Gratia**, treaty `OR` **otomatis berubah** menjadi `ORS` |
| Spreading bertanda hapus **dibuang sebelum** perhitungan |

## 11.4 Aturan nilai dan notifikasi

| Aturan | Status |
|---|---|
| Nilai estimasi **dikonversi ke IDR memakai kurs pada tanggal kejadian** sebelum dibandingkan dengan ambang apa pun; bila kurs tidak ada, **klaim ditolak** tanpa nilai bawaan (D-48) | ⚠️ isi `m_currencystandard` belum ada |
| Estimasi **> Rp 1.000.000.000** memicu **Notice of Large Losses** ke Underwriting sesuai Group Panel dan ke jajaran pimpinan | ✅ |
| Persetujuan komite dipicu ambang bertingkat — **8 ambang komite unik** terverifikasi, bukan dua | ✅ terverifikasi dari master |
| Bila premi belum lunas, notifikasi premi tertunggak dikirim | ✅ |
| Jumlah jenjang komite **diakumulasi**: setiap jenjang yang ambang bawahnya sudah terlampaui nilai klaim **ikut menyetujui** (D-47). Untuk lini **Non-MBU** saja, pita nilai dipilih lebih dulu (D-70) | ✅ master diterima — 21 kolom, 30 baris |

> **Ambang Rp 3.500 untuk server Timor-Leste tidak dibawa sebagai aturan.** Di sistem lama ia
> ditentukan oleh **hostname server** — yang dilarang di sistem baru. Bila aturan ini masih
> berlaku secara bisnis, ia menjadi baris master data per entitas, dan itu **harus dikonfirmasi
> tim bisnis** — tidak boleh disalin apa adanya.

## 11.5 Aturan kelengkapan

| Aturan | Status |
|---|---|
| **Penyebab Kerugian wajib diisi**, kecuali lini Travel | ✅ |
| **Nomor SLIK wajib diisi** untuk SPK / Asuransi Kredit | ✅ |
| Nilai klaim **tidak boleh melebihi TSI** | ✅ |
| Objek **tanpa Coverage dibuang otomatis** dari daftar | ✅ |
| Hubungan tertanggung "lain-lain" → **keterangan wajib diisi** | ✅ |
| **Polis Deklarasi tidak dapat diklaim**, kecuali lini Aneka | ✅ |
| Validasi sisa TSI | ⚠️ **aturannya tidak diketahui sama sekali** |

> **Perlu diperhatikan khusus.** Sistem lama memanggil sebuah proses bernama `ValidasiSisaTSI`
> yang **tidak ada di export**. Namanya menunjukkan adanya **aturan validasi sisa TSI yang tidak
> muncul di analisis aturan bisnis manapun** — ini bukan sekadar detail yang hilang, melainkan
> petunjuk bahwa **masih ada aturan bisnis yang belum kita ketahui keberadaannya.**

## 11.6 Aturan yang mengikat sistem baru

| # | Aturan |
|---|---|
| 1 | **Tidak ada nilai bisnis yang boleh di-hardcode.** Seluruh ambang, penerima notifikasi, pemetaan peran, dan endpoint adalah master data yang dapat diubah tanpa deploy |
| 2 | **Seluruh kesalahan validasi dikumpulkan dan dikembalikan bersamaan**, tidak berhenti pada yang pertama. Pada form berisi puluhan field, mengembalikan satu kesalahan per percobaan akan membuat pengguna menyerah. Ini **kesetaraan perilaku**, bukan preferensi |
| 3 | **Blok `// TESTING` yang menimpa email produksi tidak dibawa.** Perilaku benar yang dipakai adalah email Underwriting asli per Group Panel |
| 4 | **Bahasa domain dipakai konsisten** di nama tabel, struct, endpoint API, label UI, dan dokumentasi |

---

# 12. Process Flow

## 12.1 Alur utama — Register Flow

```
                          [Mulai]
                             │
                      ┌──────▼──────┐
                      │  View Polis │  ambil & kunci snapshot polis
                      └──────┬──────┘
                             │
                    ┌────────▼────────┐
                    │ Input Register  │  ← gerbang validasi terberat (137 step)
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

> ### Konsekuensi terpenting untuk desain
>
> **Alur nyata bukan rangkaian linear.** Sebelas *ticket* memungkinkan lompatan langsung ke tahap
> tertentu tanpa melewati urutan di atas — `setToRegister`, `SendToEstAdmin`, `SendToEstTravel`,
> `SendToEstimatorPA`, `SendToPICTravel`, `SendtoAnalysator`, `SendToInvestigator`,
> `CompliancePNC`, `AcceptanceKomite`, `RCLDokter`, `SendtoPUCL`.
>
> **Setiap tahap bisa dimasuki dari luar.** Model status di sistem baru harus mengizinkan
> transisi lateral ini — memaksakan urutan kaku akan **langsung ditolak pengguna.**

## 12.2 Tiga proses pendukung

| Proses | Alur | Keterangan |
|---|---|---|
| **Open Protection** | Input Protection → Akseptasi Protection → ◇ Diterima? → Selesai / Ditolak | Permintaan pembukaan proteksi sebelum atau di luar alur klaim normal. Satu Open Protection dapat **ditautkan ke klaim dan ditandai terpakai** saat registrasi |
| **Receive Document** | Input Receive Document → Selesai | Pencatatan penerimaan dokumen fisik, termasuk pengiriman antar cabang (ekspedisi, nomor resi, estimasi tiba). Menghasilkan `RCV_ID` yang ditautkan ke klaim |
| **Komite** | Komite Router → Lihat Detail Transfer → ◇ Masih ada level berikutnya? → ulang / Selesai | Persetujuan berjenjang sampai **4 level**. Setiap putaran menaikkan jenjang dan mereset status persetujuan, lalu meneruskan ke antrean berikutnya. Komite dapat **menyetujui, menolak, atau mengembalikan** |

## 12.3 Model penugasan

Dua konsep dipertahankan karena terbukti dipakai nyata, bukan warisan kosong:

| Konsep | Arti | Dipakai di tahap |
|---|---|---|
| **Worklist** | Tugas ditugaskan ke **satu orang tertentu** | Input Register, View Polis, Estimation, Input Estimasi, Choose Surveyor, Send To Analis, Send To PIC Teknik, RCL Dokter, Analyst Doctor |
| **Workbasket** | **Antrean bersama**, diambil siapa pun yang berwenang | RCL/PUCL, Investigator, Compliance |

⚠️ **Aturan routing untuk tiga penentu penugasan tidak ada di export** — `PNCAdminRouter`,
`PNCTeknikRouter`, `RouterRCLDokter`. Ketiganya menentukan **siapa menerima tugas apa** (R-04).

---

# 13. User Journey

## 13.1 Perjalanan klaim dari sudut pandang pengguna

| # | Tahap | Peran | Yang dilakukan | Keluaran |
|---|---|---|---|---|
| 1 | **Terima laporan** | PncAdmin | Menerima laporan kerugian dari tertanggung; mencari polis | Snapshot polis diambil dan dibekukan |
| 2 | **Registrasi** | PncAdmin | Mengisi data kejadian, objek pertanggungan, coverage, spreading, estimasi. **Menghadapi gerbang validasi terberat** | **Nomor klaim `PNCN.YY.xxxx`** · notifikasi register · **Notice of Large Losses** bila estimasi > Rp 1 M |
| 3 | **Terima dokumen fisik** | PncReceive | Mencatat penerimaan dokumen, ekspedisi, nomor resi | `RCV_ID` ditautkan ke klaim |
| 4 | **Estimasi** | PncPICTeknik / Estimator | Menetapkan nilai estimasi sesuai lini bisnis | **PLA** dikirim ke koasuransi/reasuransi |
| 5 | **Survei lapangan** | **PNCSurveyor** | Meninjau kerugian, mengunggah foto, mengisi hasil survei — **dari tablet/HP di lapangan, sering pada jaringan lambat** | Hasil survei + foto dokumentasi |
| 6 | **Usulan nilai** | PncPICTeknik / Adjuster | Menetapkan usulan nilai berdasarkan hasil survei | **Pre-DLA** dikirim sebelum akseptasi |
| 7 | **Pemeriksaan khusus** | PncComplience · PncInvestigator · PncAnalystDoctor | Kepatuhan · penyelidikan klaim mencurigakan · penilaian medis untuk PA | Rekomendasi lanjut / tolak |
| 8 | **Persetujuan komite** | PNCKomite · PNCKomiteTeknik | Menyetujui, menolak, atau mengembalikan nilai klaim — **berjenjang sampai 4 level** bila melewati ambang | Keputusan komite tercatat di jejak audit |
| 9 | **Akseptasi** | PncPICTeknik / Manager | Mengakseptasi nilai klaim | **Nomor Akseptasi** · **DLA** ke koasuransi/reasuransi · **LOD** ke tertanggung |
| 10 | **Transfer & bayar** | Kasir | Transfer ke kasir, pembayaran ganti rugi | Status pembayaran · laporan **SLIK OJK** untuk lini SPK |
| 11 | **Salvage & recovery** | PncPICTeknik | Mengelola barang sisa (termasuk balai lelang) dan pemulihan dana | Nilai bersih klaim berkurang |

## 13.2 Jalur alternatif

| Jalur | Peran | Kapan |
|---|---|---|
| **RCL — Klaim ditolak** | PncRCLPUCL | Klaim tidak dijamin polis atau ditolak komite |
| **RCL Dokter** | PncAnalystDoctor | Penolakan yang memerlukan pertimbangan medis — lini PA |
| **PUCL — Proses ulang** | PncRCLPUCL | Klaim diproses ulang setelah sebelumnya ditolak atau ditutup |
| **Open Protection** | PncCollection · PncOPCGeneral | Permintaan pembukaan proteksi di luar alur klaim normal |

## 13.3 Yang tidak berubah bagi pengguna

Ekspektasi yang **wajib dipenuhi**: alur kerja, urutan langkah, penempatan field, dan tata letak
layar **mengikuti Pega yang ada**. Perubahan tampilan dijaga seminimal mungkin agar
**pengguna tidak perlu pelatihan ulang.**

Konsekuensinya harus disadari: sistem baru **mewarisi antarmuka Pega yang padat** — grid lebar
grid lebar, form panjang, banyak tab. Kesempatan perbaikan pengalaman pengguna **ditunda ke
pasca-migrasi**, bukan dihilangkan.

---

# 14. Data Flow

## 14.1 Aliran nilai uang klaim

**Ini tulang punggung bisnisnya**, dan urutan inilah yang menentukan model data penyelesaian.

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
Transfer ke Kasir
      │
      ▼
Pembayaran
```

Di setiap tahap, nilai **dikurangi Salvage dan Recovery** bila ada, lalu **dibagi ke para
penanggung** sesuai Spreading dan Koasuransi.

## 14.2 Urutan pemberitahuan yang mengikat

| Pemberitahuan | Isi | Ditujukan ke |
|---|---|---|
| **PLA** — Preliminary Loss Advice | Nilai **estimasi** | Koasuransi / reasuransi |
| **Pre-DLA** | Nilai yang **akan** diakseptasi, dikirim **sebelum** akseptasi | Koasuransi / reasuransi |
| **DLA** — Definite Loss Advice | Nilai **akseptasi** | Koasuransi / reasuransi |
| **LOD** — Letter of Discharge | Nilai ganti rugi yang disetujui | **Tertanggung** — sasaran berbeda dari ketiga di atas |

## 14.3 Aliran data masuk dan keluar

```
        ┌──────────────┐
        │  GISFW       │──── data polis ────►┐
        │  (tim lain)  │                     │
        └──────────────┘                     ▼
                                      ┌─────────────┐
        ┌──────────────┐              │  SNAPSHOT   │  dibekukan saat registrasi
        │  HCC / HCQ   │── profil ───►│    POLIS    │  klaim kebal perubahan polis
        └──────────────┘   user       └──────┬──────┘
                                             │
        ┌──────────────┐                     ▼
        │  6 API       │              ┌─────────────────┐
        │  pengganti   │── HRD, GL, ─►│                 │──► Kasir (rekening, bayar)
        │  DB Link ⚠️  │   sales,     │   DOMAIN        │──► SLIK OJK (lini SPK)
        └──────────────┘   cabang     │   KLAIM         │──► BRI Surf (umpan balik)
                                      │                 │──► Email (5 correspondence)
        ┌──────────────┐              │   • Klaim       │──► PLA/Pre-DLA/DLA (koas/reas)
        │  Storage     │◄── unggah ──►│   • Objek       │──► LOD (tertanggung)
        │  Dokumen     │    & ambil   │   • Coverage    │
        │  (app13)     │              │   • Settlement  │
        └──────────────┘              │   • Komite      │
                                      │   • Spreading   │
        ┌──────────────┐              │                 │
        │  Arsip       │◄── injeksi ──│                 │
        │  (app8)      │              └────────┬────────┘
        └──────────────┘                       │
                                               ▼
                                      ┌─────────────────┐
                                      │  JEJAK AUDIT    │  append-only
                                      │  siapa · kapan  │  hanya INSERT & SELECT
                                      │  sebelum·sesudah│
                                      └─────────────────┘
```

## 14.4 Kepemilikan data selama masa paralel

| Aturan |
|---|
| **Satu database bersama.** Tidak ada migrasi massal data klaim historis |
| Setiap tabel punya **tepat satu pemilik**: modul yang sudah pindah ke Go memiliki tabelnya dan Pega hanya membaca — dan sebaliknya |
| **Tidak ada sinkronisasi dua arah** |
| Data yang kini hidup di **tabel milik engine Pega** dipindahkan ke tabel baru milik aplikasi — terpenting `PC_ASM_FW_GCNMFW_WORK`, yang **dibaca 116 rule** |
| Klaim `PNCN.YY.xxxx` versus `PNC-xxxx` membuat **asal setiap klaim langsung terbaca** tanpa tabel pemetaan |

## 14.5 Satu data disimpan satu kali

Sistem lama menyimpan data klaim **dua kali** — relasional dan JSON — disinkronkan stored
procedure, **tanpa mekanisme yang memeriksa konsistensinya.**

Sistem baru: **snapshot polis** sebagai JSON, **data klaim** sebagai tabel relasional. Satu data
satu tempat.

⚠️ **Sebelum migrasi**, pemeriksaan konsistensi wajib dijalankan pada data produksi, dan **sumber
kebenaran ditetapkan eksplisit per jenis data** — karena bila keduanya berbeda pada data
historis, pilihan itu **memengaruhi hasil laporan dan perhitungan.**

---

# 15. Integration

## 15.1 Integrasi yang sudah berjalan

| Sistem | Arah | Keperluan | Status |
|---|---|---|---|
| **HCC / HCQ** | Masuk | Autentikasi username + password; respons memuat **profil lengkap** (NIK, nama, cabang, jabatan, email) | ✅ |
| **Storage Dokumen** (`app13/api`) | Dua arah | Unggah dan ambil dokumen klaim; database hanya menyimpan **metadata dan referensi** | ✅ |
| **Arsip Dokumen** (`app8/asm-archive`) | Keluar | Injeksi data arsip dokumen klaim | ✅ |
| **BRI Surf** (`partner.api.bri.co.id`) | Keluar | OAuth token + kirim umpan balik klaim | ✅ |
| **Konversi Gambar** (`aiimage`) | Keluar | Konversi format AVIF | ✅ |
| **History Payment** (WebLogic internal) | Masuk | Riwayat pembayaran produksi | ✅ |
| **Kasir** | Keluar | Data rekening dan permintaan pembayaran | ✅ |
| **SLIK OJK** | Keluar | Pelaporan regulator untuk lini SPK | ⚠️ **R-07** |
| **Email SMTP** | Keluar | 5 correspondence: notifikasi register, large losses, virtual account, laporan klaim, error produksi | ⚠️ **R-07** |

## 15.2 Enam API pengganti DB Link — belum ada

Sistem lama mengakses **6 database lain langsung lewat DB Link** — 64 pemakaian di 27 rule.
PostgreSQL tidak memiliki DB Link, dan kopling tersembunyi antar database ini akan diganti
**kontrak API yang eksplisit.**

| DB Link | Pemakaian | Data yang diambil |
|---|---|---|
| `@ASMD` | **55×** | Jam kerja · master HRD · master sales · buka proteksi · GL payment · pengguna asuransi · mitra · treaty loss · cadangan klaim kredit |
| `@SIMASNET` | 3× | Buka proteksi · data klaim |
| `@SMI` | 2× | Buka proteksi · data klaim |
| `@OPJAVA` | 2× | Master user · jabatan user |
| `@PROD_ASM` | 1× | Master agen |
| `@PROD_TKA` | 1× | Master share pertanggungan |

### Dua titik paling kritis

| Objek | Kenapa kritis |
|---|---|
| **`GetIDCabang`** | Dipanggil **13 activity, termasuk jalur registrasi.** Tanpa penggantinya, registrasi klaim tidak dapat berjalan |
| **`GENERAL.MST_BUKA_PROTEKSI`** | **Satu-satunya objek remote yang *menulis*** — dan menulis ke **tiga sistem berbeda**. Penggantinya **wajib API idempoten** dan **tidak boleh dijembatani salinan berkala** |

### Strategi bertahap

| Jenis data | Penanganan |
|---|---|
| Jarang berubah — master sales, cabang, agen | **Salinan yang disegarkan berkala** boleh dipakai sebagai jembatan — **di balik seam yang sama**, sehingga penggantian ke API nyata nanti **tidak menyentuh kode domain** |
| Harus mutakhir | Modul yang bergantung padanya **tertahan** — dan itu harus disampaikan ke manajemen sebagai konsekuensi jadwal, **bukan diserap diam-diam** |
| Menulis | **Wajib API sungguhan yang idempoten.** Tidak ada jalan pintas |

> ⚠️ **Risiko R-03.** API-API ini kemungkinan besar **belum ada**, dan yang membangunnya adalah
> **tim lain dengan jadwal sendiri.** Kebutuhannya sudah disampaikan; yang belum ada adalah
> tanggal ketersediaan.

## 15.3 Aturan integrasi yang mengikat

| # | Aturan |
|---|---|
| 1 | Setiap pemanggilan keluar punya **batas waktu 30 detik**, penanganan kegagalan, dan retry di mana relevan. Kegagalan **ditangani, bukan menggantung** |
| 2 | Log pemanggilan sistem eksternal dicatat — **tanpa data sensitif** |
| 3 | Ketersediaan dokumen bergantung pada layanan tim lain, sehingga penanganan kegagalan, retry, dan **masa berlaku URL** wajib diatur |
| 4 | Operasi yang mengubah nilai uang wajib **idempoten** |

---

# 16. Error Handling

## 16.1 Tiga jenis kesalahan

| Jenis | Contoh | Penanganan | HTTP |
|---|---|---|---|
| **Kesalahan validasi bisnis** | Total spreading bukan 100% · DOL di luar periode polis · Nomor SLIK kosong | **Kumpulkan semua, kembalikan bersamaan** | `422` |
| **Pelanggaran aturan / konflik** | Akseptasi sebelum komite selesai · penugasan sudah diambil orang lain | Kembalikan **satu** kesalahan yang jelas | `409` |
| **Kegagalan teknis** | Database mati · sistem eksternal tidak merespons · bug | Catat lengkap, kembalikan **pesan umum** | `500` |

## 16.2 Aturan yang mengikat

| # | Aturan |
|---|---|
| 1 | **Kesalahan validasi dikumpulkan seluruhnya, tidak berhenti pada yang pertama.** Sistem lama memeriksa belasan aturan dan menampilkan semuanya sekaligus. Pada form registrasi berisi puluhan field, mengembalikan satu kesalahan per percobaan akan membuat pengguna menyerah. Ini **kesetaraan perilaku**, bukan preferensi |
| 2 | **Kesalahan domain adalah tipe, bukan string.** Lapisan transport yang memetakannya ke kode HTTP; domain **tidak boleh tahu tentang HTTP** |
| 3 | Setiap kesalahan **dibungkus dengan konteks saat naik**, sehingga pesan akhirnya menceritakan jalurnya — bukan sekadar "record not found" tanpa keterangan record apa |
| 4 | **Kesalahan tidak pernah ditelan.** Bila memang sengaja diabaikan, **wajib disertai komentar** yang menjelaskan alasannya |
| 5 | **Detail internal tidak pernah bocor ke klien.** Pesan `500` hanya memuat id permintaan; detail lengkapnya di log. Membocorkan struktur database atau jejak tumpukan ke browser adalah **celah keamanan** |
| 6 | **Pesan untuk pengguna berbahasa Indonesia dan menjelaskan cara memperbaiki**, mengikuti gaya sistem lama yang sudah dikenal |
| 7 | `panic` **hanya untuk kondisi yang tidak mungkin terjadi**, ditangkap middleware recovery. **Tidak pernah sebagai alur kendali** |

## 16.3 Contoh pesan yang menjadi acuan

> "Tanggal Lapor tidak boleh lebih dari 7 hari setelah Tanggal Kejadian."
> "Nomor Polis sudah terdaftar dengan nomor klaim PNC-1865."
> "Total spreading harus 100%."

---

# 17. Security

## 17.1 Authentication

| Aspek | Ketentuan |
|---|---|
| Kredensial | Diverifikasi **API internal HCC/HCQ** — username + password |
| Respons | Memuat **profil lengkap**: NIK, nama, cabang, jabatan, email |
| Session | Aplikasi menerbitkan **session/token miliknya sendiri** setelah HCC/HCQ memvalidasi |
| Kegagalan HCC/HCQ | Ditangani eksplisit — tidak menggantung, tidak membocorkan penyebab kegagalan ke pengguna |

## 17.2 Authorization

| Aspek | Ketentuan |
|---|---|
| Kepemilikan | **Dimiliki dan dikelola sepenuhnya aplikasi Claim PNC**, bukan didelegasikan |
| Mekanisme | Tabel peran dan izin menu **yang belum ada** dan akan dirancang dalam project ini |
| Penegakan | **Diperiksa di setiap endpoint backend** — **tidak boleh** bergantung pada penyembunyian menu seperti sistem lama |
| Batas data | Akses **data medis dibatasi** peran Analyst Doctor dan RCL Dokter |
| Akses laporan | Peran laporan eksternal mendapat akses **terbatas** |
| **Jumlah peran** | **22**, satu-untuk-satu dengan access group Pega, hanya dinamai ulang (`D-58`) |
| **Satuan izin** | **menu, bukan tindakan individual** (`D-59`) |
| **Pemisahan tugas** | **tidak ada** secara formal |

> **Pembaruan v2.0 — satuan izin ditetapkan sebagai menu.** Pengguna yang memiliki akses ke sebuah
> menu berwenang atas **seluruh tindakan yang dijangkau menu itu**, termasuk membatalkan klaim,
> mengubah nilai setelah persetujuan komite, dan menyetujui komite.
>
> Yang tetap berubah dari sistem lama adalah **tempat penegakannya** — dari penyembunyian menu di
> antarmuka menjadi pemeriksaan di server pada setiap endpoint. Dengan bacaan itu, §21.2 kriteria
> #8 dan `FR-R1` tetap terpenuhi: yang diperiksa adalah *apakah peran pemanggil memiliki menu yang
> memberi akses ke endpoint ini*.
>
> **Konsekuensi yang diterima secara sadar:**
>
> 1. **Orang yang sama dapat membuat, menyetujui, dan membayarkan satu klaim** bila perannya
>    memiliki ketiga menu itu. Tidak ada kontrol teknis yang mencegahnya.
> 2. **Jejak audit menjadi satu-satunya kontrol pengimbang** — menaikkan `FR-S5` dari modul
>    pendukung menjadi kontrol utama, padahal ia kemampuan **baru 100% tanpa baseline**.
> 3. **`AutoAcceptKomite` melewati kontrol apa pun** — job terjadwal harian jam 06:00 menyetujui
>    komite tanpa pengguna sama sekali.
> 4. §21.2 kriteria #9 (jejak audit untuk setiap perubahan bernilai bisnis) menjadi **wajib tanpa
>    pengecualian** pada seluruh modul bisnis.
>
> **Dua hal yang menghalangi `FR-F3` mengisi tabelnya:** kontrak API HCC/HCQ **belum ada** (nol
> jejak di export), dan **penugasan operator ke peran tidak ada di database** —
> `POOLDATA.T_ACCESS_GROUP_PNC` hanya memetakan `OPERATOR_ID` → `OLD_OPERATOR_ID`.

## 17.3 Perlindungan masukan pengguna

| Ancaman | Penangkal |
|---|---|
| **SQL injection** | **Parameter binding tanpa perkecualian.** Nama kolom sort dan filter **hanya dari daftar yang diizinkan.** Ini yang menutup celah warisan yang menyisipkan nilai pengguna langsung ke teks SQL |
| **XSS** | React meng-escape secara baku. Penyisipan HTML mentah **dilarang** kecuali disetujui tertulis |
| **CSRF** | Cookie `SameSite=Strict` + token CSRF pada permintaan yang mengubah data |
| **Unggahan berkas berbahaya** | Validasi jenis dan ukuran; **nama berkas dihasilkan sistem, tidak pernah dari pengguna** |
| **Path traversal** | Tidak ada akses berkas berdasarkan jalur dari pengguna |

## 17.4 Data sensitif

Sistem menyimpan **NIK, nomor telepon, alamat, data medis (klaim PA), dan nomor rekening**.

| Ketentuan |
|---|
| Konsep **masking yang sudah ada di sistem lama dipertahankan** |
| Data sensitif **tidak pernah** ditulis ke log |
| Sambungan ke database dan sistem eksternal **wajib terenkripsi** |
| Kredensial **tidak pernah** di dalam kode maupun repository — dari variabel lingkungan atau pengelola rahasia. Berkas konfigurasi contoh hanya memuat nilai kosong |

## 17.5 Jejak audit sebagai kendali keamanan

Jejak audit bukan hanya kebutuhan bisnis — ia juga **kendali keamanan**. Sifat append-only
ditegakkan lewat **hak akses database**: aplikasi hanya diberi `INSERT` dan `SELECT`, sehingga
**tidak ada jalur di dalam aplikasi yang bisa menghapus jejaknya sendiri.**

## 17.6 Yang harus dihilangkan dari sistem lama

| Masalah di sistem lama | Perbaikan |
|---|---|
| **Perangkaian SQL dari nilai pengguna** | Parameter binding wajib |
| **Potongan klausa SQL disimpan sebagai nilai property** | Dilarang sepenuhnya |
| **User ID di-hardcode sebagai penentu perilaku** — 4 nama orang tertentu | Peran dan izin dari master data |
| **Hostname server menentukan perilaku bisnis** | Konfigurasi per lingkungan |
| **Blok `// TESTING` menimpa email produksi** | Dilarang; **ditolak di code review** |
| **Otorisasi bergantung pada penyembunyian menu** | Pemeriksaan izin di setiap endpoint backend |

---

# 18. Assumptions

Asumsi kerja yang menjadi dasar dokumen ini. **Setiap asumsi disertai konsekuensi bila salah** —
karena asumsi yang tidak diuji adalah risiko yang belum tercatat.

| # | Asumsi | Bila salah |
|---|---|---|
| A-1 | Export XML ini **mewakili seluruh aplikasi Claim PNC** yang berjalan di produksi | **Asumsi ini sudah gugur sebagian** — ±242 rule dirujuk tetapi tidak ada di export (R-16), dan export bertambah dua kali selama analisis |
| A-2 | **Source 64 procedure & function masih ada** di database dan dapat diberikan DBA | R-01 berubah dari penundaan menjadi **kebuntuan** — perilaku harus disimpulkan dari perbandingan masukan-keluaran di staging, tanpa jaminan kelengkapan kasus tepi |
| A-3 | Tabel-tabel yang dirujuk 652 rule SQL **masih ada dengan struktur yang sama** | Desain skema harus diulang |
| A-4 | **API HCC/HCQ stabil** dan tersedia untuk seluruh pengguna Claim PNC | Modul identitas & akses tertahan; seluruh sistem tidak dapat diakses |
| A-5 | **API storage internal (app13/app8) tetap berjalan** dan kontraknya tidak berubah | Modul dokumen tertahan; dokumen klaim tidak dapat diunggah atau diambil |
| A-6 | **Tim infra menyediakan 2 VM + load balancer** sebelum tahap deployment | Tuntutan 24/7 tidak dapat dipenuhi |
| A-7 | **PostgreSQL 17+ akan tersedia** ketika perpindahan database dilakukan | Keputusan "satu set SQL portabel" **tidak dapat dijalankan**; 29 rule JSON harus ditulis ulang |
| A-8 | **Tim GISFW menyediakan struktur data polis** yang dibutuhkan untuk snapshot | Modul polis & snapshot tertahan |
| A-9 | Ambang yang ditemukan di kode lama (Rp 50 juta, Rp 30 juta, Rp 1 miliar) **masih berlaku secara bisnis** | Perhitungan komite dan notifikasi salah — **karena itu wajib divalidasi ulang tim bisnis, bukan disalin** |
| A-10 | Tanggal di data historis **tersimpan dengan konvensi yang konsisten** | Tanggal bergeser saat dibaca sistem baru, dan pada kasus di sekitar tengah malam ini **mengubah hasil validasi** aturan tanggal — **wajib diperiksa terhadap contoh data produksi** |
| A-11 | 22 access group dan pemetaannya ke 47 menu **masih mencerminkan kebutuhan bisnis saat ini** | Tabel otorisasi baru salah rancang |
| A-12 | **Tim Pega bersedia menyepakati pembekuan perubahan** pada modul yang sedang dimigrasi | Sistem sumber tetap bergerak; **124 activity berubah pada 2026** menunjukkan ini nyata |
| A-13 | **User bisnis tersedia untuk UAT** per modul | Gerbang kedua cutover tidak dapat dilewati — lihat Bab 19, C-9 |

---

# 19. Constraints

| ID | Batasan | Konsekuensi |
|---|---|---|
| **C-1** | **Tim adalah developer Pega internal** yang harus mempelajari Go, React, dan TypeScript **sekaligus** | **Batasan utama seluruh keputusan teknis.** Mengutamakan learning curve landai di atas kecanggihan; Steering wajib preskriptif; abstraksi berat dan "magic" framework dihindari; **waktu belajar tidak dapat dipercepat dengan menambah orang** |
| **C-2** | **Target seluruh modul selesai akhir September 2026** — ditetapkan manajemen, tanpa pemangkasan scope | Lihat Bab 20, R-05 dan R-13 |
| **C-3** | **Ketersediaan 24/7** termasuk saat deployment | Aplikasi **wajib stateless**; minimal 2 instance; rolling deployment; **migrasi skema wajib backward-compatible** — kolom dihapus dalam dua tahap, nama kolom tidak pernah diganti langsung |
| **C-4** | **Runtime sementara tetap Oracle 19c**, sasaran akhir PostgreSQL 17+ | Satu set SQL portabel yang berjalan di keduanya, dengan tiga pengecualian terkelola |
| **C-5** | **Deployment on-premise VM**, tanpa Kubernetes maupun cloud | Strategi deployment, monitoring, dan scaling harus realistis untuk lingkungan VM |
| **C-6** | **Satu database bersama** dengan Pega selama masa paralel | Satu tabel hanya boleh ditulis satu sistem; tidak ada sinkronisasi dua arah; **urutan migrasi tidak bebas** |
| **C-7** | **Alur dan tata letak layar mengikuti Pega** | Mewarisi antarmuka padat: grid lebar, form panjang bernested repeat. Perbaikan pengalaman pengguna ditunda ke pasca-migrasi |
| **C-8** | **Sepuluh artefak ditunggu dari pihak luar** — DBA, Tim Pega, tim pemilik 6 sistem, Compliance, Infra | **Sepuluh dari dua belas tindakan pembuka tidak dapat dipercepat dengan menambah developer.** Lima modul jantung nilai uang klaim tertahan |
| **C-9** | **Waktu user bisnis untuk UAT** per modul — untuk 26 harness bernama Inbox dan 5 layar transaksi utama | User yang sama sedang menjalankan **operasional klaim harian**. Alokasinya **belum tercatat di manapun** dan **tidak punya ruang pada jadwal C-2**. Lihat Bab 20, R-15 |
| **C-10** | **Pega tetap sistem produksi** sampai sistem baru sepenuhnya divalidasi | Tidak boleh ada gangguan pada operasional klaim yang sedang berjalan |
| **C-11** | **GISFW dikembangkan tim lain** dengan jadwal sendiri | Batas kepemilikan data polis ditetapkan lewat snapshot agar Claim PNC tidak bergantung pada jadwal tim lain saat runtime |
| **C-12** | **Sistem sumber masih aktif berubah** — 124 activity diubah pada 2026 | Butuh kesepakatan pembekuan perubahan + pencatatan perubahan mendesak + rekonsiliasi berkala |

---

# 20. Risks

**19 risiko**. Skala: **Rendah · Sedang · Tinggi**. Rincian lengkap:
[`../Steering/16-RISK-ANALYSIS.md`](../Steering/16-RISK-ANALYSIS.md) dan
[`migration-readiness.md`](../migration-readiness.md).

## 20.1 Risiko berdampak tinggi

| ID | Risiko | Dampak | Kemungkinan | Status |
|---|---|---|---|---|
| **R-01** | **Source 64 procedure & function tidak tersedia** | Tinggi | **Terjadi** | Terbuka — sudah diminta |
| **R-02** | **Job terjadwal tidak diketahui** | Tinggi | **Terjadi** | Terbuka — sudah diminta |
| **R-03** | **6 API pengganti DB Link belum ada** | Tinggi | Tinggi | Terbuka — sudah disampaikan |
| **R-05** | **Jadwal tidak sepadan dengan ukuran pekerjaan** | Tinggi | Tinggi | Diterima manajemen |
| **R-11** | **Tim mempelajari 3 teknologi sekaligus** | Tinggi | Tinggi | Melekat pada pilihan tim & frontend |
| **R-14** 🆕 | **Ukuran keberhasilan tidak dapat dibuktikan untuk 5 modul** | Tinggi | **Terjadi** | **Baru** |

### R-01 · Source 64 procedure & function tidak tersedia

Seluruh logika stored procedure ditulis ulang di Go — tetapi **source-nya tidak ada di export
XML.** Yang terlihat hanya nama dan daftar parameternya dari pemanggilan di SQL. **Isinya belum
pernah dilihat siapa pun di tim ini.**

**Akibatnya: lima modul tidak dapat diselesaikan** — settlement (FR-B5), komite (FR-B7),
PLA/DLA (FR-B9), akseptasi & pembayaran (FR-B10), salvage (FR-B12). **Kelimanya adalah jantung
aliran nilai uang klaim.**

**Penanganan:** minta source lengkap ke DBA (daftar dan query siap pakai sudah disiapkan);
sementara itu kerjakan modul yang tidak bergantung. Bila tetap tidak tersedia, satu-satunya jalan
adalah menyimpulkan perilaku dari perbandingan masukan-keluaran di staging — **jauh lebih lambat
dan tidak menjamin kelengkapan kasus tepi.**

### R-02 · Job terjadwal tidak diketahui

Ada job terjadwal yang berjalan hari ini, tetapi **detailnya belum diketahui** dan export XML
**tidak memuat rule Agent maupun Queue Processor**, sehingga tidak dapat dianalisis dari source
sama sekali.

Kandidat yang paling mungkin: pengingat SLA, eskalasi tugas menggantung, penutupan klaim yang
lewat tenggat, pengiriman batch ke kasir, dan pelaporan SLIK OJK.

> **Ini kelas risiko paling berbahaya: kegagalan yang tidak terlihat.** Sistem baru akan tampak
> berjalan normal sementara ada proses otomatis yang diam-diam **tidak lagi berjalan** — tanpa
> ada yang menyadarinya sampai sesuatu tidak terkirim.

### R-05 · Jadwal tidak sepadan dengan ukuran pekerjaan

Target akhir September 2026 versus ukuran terukur dari source: 74 layar, 269 komponen,
15.063 step logika, 652 query, 64 procedure tanpa source, 56 laporan, **21 integrasi + 4 layanan masuk** + 6 API baru,
ditambah tabel autentikasi/otorisasi dan penugasan yang **belum ada sama sekali**, dan tim yang
harus belajar tiga teknologi sekaligus.

**Perkiraan berbasis ukuran di atas: 9–15 bulan dengan tim 5–6 orang.**

**Status:** keberatan sudah disampaikan beserta angkanya. Pemilik project menegaskan target tetap
berlaku dan seluruh modul harus pindah. **Keputusan ini dihormati** dan strategi migrasi disusun
untuknya — urutan pengerjaan disusun agar pekerjaan berisiko rendah berjalan lebih dulu, sehingga
bila waktu tidak mencukupi, **yang tertinggal adalah bagian yang paling sedikit dampaknya.**

### R-14 🆕 · Ukuran keberhasilan tidak dapat dibuktikan untuk 5 modul

Keberhasilan project diukur sebagai **kesetaraan fungsional dengan Pega** (Bab 22), dan gerbang
pertama cutover adalah **uji kesetaraan otomatis** (Bab 21). Keduanya menuntut satu hal yang
sama: **mengetahui apa hasil yang benar.**

Untuk FR-B5, FR-B7, FR-B9, FR-B10, dan FR-B12, hasil yang benar ditentukan oleh **64 procedure
yang logikanya belum pernah dibaca** (R-01).

> **Kesetaraan atas logika yang tidak diketahui tidak dapat dibuktikan, hanya diduga.**

Ini menaikkan derajat R-01: ia bukan sekadar penghambat jadwal, melainkan **penghambat kemampuan
menyatakan project ini berhasil menurut ukuran yang dipilih manajemen sendiri.**

**Penanganan:** angkat R-01 ke manajemen bukan sebagai permintaan teknis kepada DBA, melainkan
sebagai **prasyarat ukuran keberhasilan.** Bila tidak dapat dipenuhi, ukuran keberhasilan untuk
kelima modul harus **diubah secara sadar** — bukan diam-diam diturunkan menjadi "tampaknya jalan".

## 20.2 Risiko berdampak sedang

| ID | Risiko | Penanganan ringkas |
|---|---|---|
| **R-04** | **3 router penugasan tidak ada di export** — menentukan siapa menerima tugas apa | Minta export; bila tidak tersedia, gali dari tim operasional **dan buktikan** dengan membandingkan hasil penugasan pada data historis |
| ~~**R-06**~~ | ~~Arti kode status tidak diketahui~~ — **TERTUTUP**: master diterima, **33 kode `1134`–`1166`** | — |
| **R-07** | **50 activity dipanggil tapi tidak ada di export** — termasuk notifikasi email (dipanggil 15×) dan **validasi sisa TSI** | Minta export; prioritaskan notifikasi, validasi sisa TSI, kasir, dan ticket |
| **R-08** | **DDL tabel tidak tersedia** — desain skema harus menebak tipe, panjang, index, constraint | Minta DDL + statistik ukuran tabel; statistik yang menentukan strategi index dan partisi |
| **R-09** | **Sistem sumber masih aktif berubah** — 124 activity diubah pada 2026 | Pembekuan perubahan + pencatatan perubahan mendesak + rekonsiliasi berkala |
| **R-10** | **Data ganda relasional versus JSON**, tanpa mekanisme pemeriksa konsistensi | Pemeriksaan konsistensi pada data produksi **sebelum** migrasi; sumber kebenaran ditetapkan eksplisit |
| **R-12** | **Zona waktu bergeser saat migrasi data** — **mengubah hasil validasi** aturan tanggal | Periksa contoh data produksi; uji khusus kasus di sekitar tengah malam WIB; bandingkan tanggal yang ditampilkan kedua sistem pada klaim yang sama |
| **R-13** 🆕 | **Tenggat tidak punya pemaksa eksternal** | Lihat §20.3 |
| **R-15** 🆕 | **Waktu user bisnis untuk UAT belum dialokasikan** | Alokasikan resmi per modul; masukkan sebagai constraint project (C-9), bukan detail pelaksanaan yang diasumsikan tersedia |

## 20.3 R-13 · Tenggat tidak punya pemaksa eksternal

Discovery 2026-09-08 mengonfirmasi pendorong migrasi adalah **kemandirian teknologi dan
maintainability** — dan secara eksplisit **bukan** lisensi yang berakhir, **bukan** biaya
lisensi, dan **bukan** arahan standardisasi grup.

Artinya target akhir September 2026 adalah **target internal manajemen**, tanpa tanggal yang
dipaksakan pihak luar dan **tanpa penalti kontraktual.** Penjadwalan ulang sepenuhnya berada di
dalam kendali manajemen Sinarmas.

**Kenapa ini dicatat sebagai risiko, bukan sebagai kabar baik.** Menerima risiko R-05 apa adanya
akan wajar bila ada tenggat eksternal yang tidak dapat digeser — mengejar tanggal dengan mutu
yang dikompromikan kadang memang pilihan yang benar. Tetapi di sini **tidak ada tenggat seperti
itu**, sehingga risiko yang diterima **tidak dibayar oleh manfaat apa pun.** Yang tersisa hanya
biayanya: mutu yang dikompromikan pada sistem yang menangani uang klaim.

**Penanganan.** Sampaikan ke manajemen bahwa penjadwalan ulang **tidak berbiaya eksternal**,
disertai angka R-05 dan urutan pengerjaan berbasis dependensi sebagai dasar keputusan. **Bukan
untuk menolak target** — yang sudah ditegaskan ulang dan tetap menjadi dasar strategi migrasi —
melainkan agar keputusan mempertahankannya diambil dengan mengetahui bahwa **alternatifnya
tersedia tanpa penalti.**

## 20.3b Empat risiko baru — R-16 sampai R-19

Ditambahkan v2.0 atas dasar `D-38`, dari verifikasi bukti terhadap export.

| ID | Risiko | Dampak | Pemilik | Kenapa tidak digabung ke risiko yang ada |
|---|---|---|---|---|
| **R-16** | **Export Pega tidak lengkap** — **±242 rule** dirujuk tetapi tidak ada, terberat **137 When rule buatan sendiri**. Tujuh tipe rule tidak pernah diaudit sama sekali | Tinggi | Tim Pega | `R-01` mencakup objek database, `R-07` mencakup activity. **When rule adalah kategori ketiga**, dan dampaknya lebih luas — memblokir percabangan bisnis di hampir semua modul |
| **R-17** | **Kredensial aktif di dalam export** — 3 password SMTP di 31 lokasi + kredensial OAuth, plaintext, `UseSSL=false` | Tinggi | Tim Infra / Security | **Bukan risiko migrasi** — ia hidup sekarang bahkan bila migrasi dibatalkan |
| **R-18** | **Dua integrasi BRI menunjuk host sandbox** di ruleset produksi | Sedang | Work Owner + tim integrasi | Menyangkut benar-tidaknya integrasi berjalan hari ini, bukan pemindahannya |
| **R-19** | **Cacat aturan uang direplikasi bila `P-5` dipatuhi buta** | Tinggi | Work Owner | Ini risiko **dari prinsip yang kita pilih sendiri**, bukan dari kekurangan artefak |

`R-19` sudah **ditangani**: `D-49` memutuskan sepuluh cacat satu per satu — **sembilan diperbaiki,
satu direplikasi** karena perilakunya memang benar. Daftar perbaikan eksplisit `P-5` naik dari
**4 menjadi 13 butir**.

## 20.4 Tiga hal yang perlu keputusan manajemen

| # | Yang diputuskan | Kenapa manajemen, bukan tim teknis |
|---|---|---|
| 1 | **R-01 sebagai prasyarat ukuran keberhasilan** | Bila source 64 procedure tidak pernah datang, ukuran keberhasilan untuk 5 modul harus diubah secara sadar. Itu keputusan manajemen — **bukan kompromi yang diambil diam-diam oleh tim** |
| 2 | **Penegasan atau penyesuaian jadwal** (R-05 + R-13) | Penjadwalan ulang tersedia tanpa penalti kontraktual. Manajemen berhak mengetahui itu sebelum menegaskan target kembali |
| 3 | **Alokasi waktu user bisnis untuk UAT** (R-15) | Menuntut waktu orang yang sedang menjalankan operasional klaim harian. **Hanya manajemen yang dapat mengalokasikannya** |

## 20.5 Sepuluh tindakan pembuka

Status: **sudah diminta, menunggu jawaban.**

| # | Tindakan | Menutup | Kepada |
|---|---|---|---|
| 1 | Minta **source 64 procedure & function** | R-01, R-14 | **DBA** |
| 2 | Minta export **Rule-Agent / Queue Processor** | R-02 | **Tim Pega** |
| 3 | Minta **DDL + statistik ukuran tabel** | R-08 | **DBA** |
| 4 | Ambil isi master **`V_STS_CLAIM`** — *satu query* | R-06 | **DBA** |
| 5 | Minta export **3 router + 40 activity** | R-04, R-07 | **Tim Pega** |
| 6 | Sampaikan kebutuhan **6 API pengganti DB Link** | R-03 | **Tim pemilik sistem** |
| 7 | Ambil isi **`POOLDATA.EMAILKOMITE`** — *satu query* + konfirmasi aturannya | Matriks komite | **DBA + tim bisnis** |
| 8 | Konfirmasi **retensi audit** | NFR audit | **Compliance** |
| 9 | Sepakati **pembekuan perubahan Pega** | R-09 | **Tim Pega + manajemen** |
| 10 | Mulai **pelatihan Go, React, TypeScript** | R-11 | **Tim pengembang** |

> **Sembilan dari sepuluh bergantung pada pihak di luar tim pengembang.** Waktu tunggunya
> **tidak dapat dipercepat dengan menambah developer** — karena itu seluruhnya harus dimulai
> hari pertama.
>
> **Karena permintaan sudah terkirim, yang menentukan sekarang bukan mengirim ulang, melainkan
> menetapkan tanggal komitmen per artefak dan jalur eskalasi bila tanggal itu tidak dipenuhi.**
> Tanpa tanggal komitmen, "sudah diminta" punya dampak jadwal yang **sama persis** dengan
> "belum diminta".
>
> **Prioritaskan #4 dan #7** — masing-masing satu query, dan keduanya memblokir seluruh modul
> komite dan setiap layar berstatus. Biaya pemenuhannya paling kecil, dampaknya paling besar.

---

# 21. Acceptance Criteria

## 21.1 Dua gerbang berurutan per modul

Ditetapkan pada discovery 2026-09-08. Setiap modul melewati **dua gerbang, berurutan**:

| Gerbang | Isi | Penanggung jawab |
|---|---|---|
| **1 — Uji kesetaraan otomatis** | Perbandingan hasil sistem baru versus Pega atas **data historis yang sama**. **Setiap selisih harus dijelaskan** sebagai bug yang diperbaiki, atau sebagai salah satu dari empat perbaikan yang sudah diputuskan eksplisit | **Tim Pengembang** |
| **2 — UAT user bisnis** | User bisnis dari **peran yang memakai modul tersebut** menguji dan menyetujui | **User bisnis per peran + Business Owner** |

> **Kenapa dua gerbang, bukan satu.** Gerbang 1 menangkap perbedaan hasil yang tidak terlihat mata
> — pada data uang klaim, satu selisih pembulatan yang lolos akan berlipat di ribuan klaim.
> Gerbang 2 menangkap hal yang tidak dapat diukur otomatis: apakah pengguna benar-benar dapat
> menyelesaikan pekerjaannya di layar baru.
>
> Ini pilihan **paling aman untuk data uang klaim — dan sekaligus paling lambat.** Konsekuensinya
> tercatat sebagai C-9 dan R-15: **waktu user bisnis harus dialokasikan resmi.**

## 21.2 Kriteria penerimaan per modul

Sebuah modul diterima bila **seluruh** kriteria berikut terpenuhi:

| # | Kriteria |
|---|---|
| 1 | **Seluruh functional requirement modul terpenuhi** sesuai [`requirement-summary.md`](../requirement-summary.md) §2 |
| 2 | **Seluruh business rule dan validation rule** yang berlaku pada modul **punya test bernama kalimat bisnis** dan lulus |
| 3 | **Uji kesetaraan otomatis lulus** — hasil identik Pega atas data historis yang sama, atau setiap selisih dijelaskan dan disetujui |
| 4 | **UAT oleh user bisnis dari peran yang memakai modul disetujui** |
| 5 | **Target performa terpenuhi terhadap data sebesar produksi** — bukan terhadap data pengembangan |
| 6 | **Rencana eksekusi setiap query baru sudah diperiksa** terhadap data sebesar produksi |
| 7 | **Nol pelanggaran aturan ketergantungan antar lapisan** — ditegakkan otomatis |
| 8 | **Otorisasi diperiksa di setiap endpoint** modul, bukan hanya disembunyikan di menu |
| 9 | **Jejak audit tercatat** untuk setiap perubahan bernilai bisnis pada modul |
| 10 | **Tidak ada nilai bisnis yang di-hardcode** — seluruhnya master data |
| 11 | **Tidak ada perangkaian SQL dari nilai pengguna** — parameter binding tanpa perkecualian |
| 12 | **Kepemilikan tabel jelas dan tunggal** — modul menulis hanya ke tabel yang dimilikinya |
| 13 | **Migrasi skema backward-compatible** — diverifikasi dengan menjalankan versi lama dan baru bersamaan |

## 21.3 Kriteria penerimaan tingkat sistem

Sebelum Pega dapat dinonaktifkan:

| # | Kriteria |
|---|---|
| 1 | **Seluruh 34 modul** melewati kedua gerbang |
| 2 | **Seluruh klaim `PNC-xxxx` yang masih berjalan selesai**, atau dipindahkan secara sadar |
| 3 | **Daftar job terjadwal lengkap dan seluruhnya berjalan** di sistem baru — atau dinyatakan tidak lagi diperlukan oleh business owner |
| 4 | **Seluruh 6 API pengganti DB Link berjalan** — atau jembatan sementaranya disetujui secara sadar sebagai kondisi tetap |
| 5 | **Rolling deployment terbukti tanpa downtime** pada lingkungan produksi |
| 6 | **Pemulihan dari backup terbukti** memenuhi RPO/RTO yang ditetapkan |
| 7 | **Pega dijalankan read-only selama satu periode pengamatan** sebelum dimatikan |
| 8 | **Retensi jejak audit sesuai ketentuan Compliance** |

## 21.4 Kriteria yang belum dapat dinyatakan

> ⚠️ Untuk **FR-B5, FR-B7, FR-B9, FR-B10, dan FR-B12**, kriteria #3 pada §21.2 — uji kesetaraan
> otomatis — **belum dapat dinyatakan terpenuhi maupun tidak terpenuhi**, karena pembanding yang
> benar berada di dalam 64 procedure yang belum pernah dibaca (R-01, R-14).
>
> **Kelima modul ini tidak boleh dinyatakan diterima** sampai R-01 tertutup, **atau** sampai
> manajemen menetapkan ukuran penerimaan pengganti secara eksplisit dan tertulis.

### Pembaruan v2.0 — pasal ini dicabut untuk empat modul

Setelah folder `Database/` diterima pada 2026-09-09, dasar pasal di atas **sudah tidak berlaku**
untuk sebagian modul. `D-55` mencabutnya untuk empat modul dan mempertahankannya untuk satu:

| Modul | Status §21.4 | Sisa penghalang |
|---|---|---|
| **FR-B7** Komite | **dicabut** | PA dan Travel di atas Rp 200.000.000 tidak punya baris master |
| **FR-B9** PLA/DLA | **dicabut** | status `VALID`/`INVALID` `GET_GROUPBUSINESS_XOL`; penulisan ulang procedure ber-9-`COMMIT` |
| **FR-B10** Akseptasi | **dicabut** | 3 activity hilang: `InsertDataAkseptasiToLeader`, `InsertLogKasir_act`, `TransferCashierDataASM_act` |
| **FR-B12** Salvage | **dicabut** | `SET_ATTACHFILETEMPSALVAGE`; hitungan DBA atas baris ber-`IDSALVAGE` NULL |
| **FR-B5** Settlement | **tetap berlaku** | isi `POOLDATA.GCNM_FEE_SCALE` (17 pita) · isi `m_currencystandard` · `BrowseT_Claim_Adjustment_SQL` |

### 21.5 Dua modul yang tidak punya pembanding sama sekali

Persoalan berbeda dari §21.4, dan **tidak dapat diselesaikan dengan menunggu artefak**:

| Modul | Sebab | Pengganti gerbang 1 | Pemilik kontrak |
|---|---|---|---|
| **FR-F3** Identitas & Akses | **HCC/HCQ nol jejak di export** — hanya 2 teks pesan galat | uji fungsional terhadap **kontrak API HCC/HCQ** | Tim HCC/HCQ |
| **FR-S5** Jejak Audit | sistem lama **tidak mencatat perubahan nilai sama sekali** | uji fungsional terhadap **daftar peristiwa wajib audit** | Compliance |

Menghapus gerbang 1 begitu saja **ditolak** — keduanya justru modul paling sensitif, dan UAT tidak
memeriksa hal yang tidak terlihat di layar (`D-56`).

> **Kedua kontrak belum ada.** Sampai diterima, `FR-F3` dan `FR-S5` **tidak dapat lulus gerbang apa
> pun**. `FR-F3` adalah prasyarat login seluruh aplikasi, sehingga ini penghalang paling awal di
> seluruh rencana.

### 21.6 Prasyarat yang memblokir gerbang 1 seluruh modul

Uji kesetaraan dijalankan modul **`S-8`** atas **Pega staging versus Go staging** dengan salinan
data produksi (`D-42`, `D-53`). **Ketersediaan Pega staging yang dapat ditembak dari luar belum
dikonfirmasi** — dan tanpa itu, gerbang 1 **tidak dapat dijalankan untuk modul mana pun**.

Kewenangan menyetujui selisih ditetapkan `D-54`: selisih yang cocok dengan salah satu dari
**13 butir `P-5`** lolos otomatis; selisih di luar itu menuntut **persetujuan Work Owner secara
tertulis**.

---

# 22. Success Metrics

## 22.1 Ukuran keberhasilan utama

Ditetapkan pemilik project pada 2026-09-08 sebagai **satu-satunya ukuran keberhasilan**:

> ### Kesetaraan fungsional dengan Pega
>
> Seluruh modul berjalan di sistem baru dengan **hasil identik Pega**, sampai **Pega dapat
> dimatikan**. Ukurannya **biner per modul**: modul sudah pindah dan terbukti setara, atau belum.

**Ini konsisten penuh dengan prinsip migrasi P-5** — "hasil yang benar adalah hasil yang sama
dengan Pega". Ukuran keberhasilan dan prinsip pelaksanaan tidak bertentangan.

## 22.2 Pengukuran

| # | Ukuran | Cara mengukur | Target |
|---|---|---|---|
| M-1 | **Modul yang setara dan sudah pindah** | Jumlah modul yang lulus kedua gerbang Bab 21 | **27 dari 27** |
| M-2 | **Selisih hasil versus Pega** | Laporan perkakas uji kesetaraan (FR-S8) per modul | **Nol selisih yang tidak dijelaskan** |
| M-3 | **Klaim `PNC-xxxx` yang masih berjalan** | Query jumlah klaim warisan yang belum selesai | **Nol** sebelum Pega dimatikan |
| M-4 | **Pega dinonaktifkan** | Pega read-only, lalu mati | **Tercapai** |

## 22.3 Ukuran pendukung — bukan penentu keberhasilan

Ukuran teknis di bawah **wajib dipenuhi** sebagai bagian kriteria penerimaan (Bab 21), tetapi
**bukan** ukuran keberhasilan project di mata manajemen:

| Ukuran | Target |
|---|---|
| Performa persentil 95 terhadap data produksi penuh | Sesuai Bab 10.1 |
| Ketersediaan | 24/7, nol downtime terencana untuk deployment aplikasi |
| Cakupan test aturan bisnis | Seluruh aturan terdokumentasi punya test |
| Pelanggaran aturan ketergantungan antar lapisan | Nol |
| Nilai bisnis yang masih di-hardcode | Nol |
| Perangkaian SQL dari nilai pengguna | Nol |

## 22.4 Yang secara eksplisit tidak diukur

| Tidak diukur | Alasan |
|---|---|
| **Penghematan biaya lisensi** | Bukan pendorong migrasi; tidak ada angka penghematan yang menjadi justifikasi project |
| **Perbaikan Turn Around Time klaim** | Bukan ukuran keberhasilan. **Karena itu tidak ada kewajiban mengambil baseline TAT Pega sebelum cutover** |
| **Kepuasan pengguna atas tampilan baru** | Tampilan sengaja dibuat sama agar pengguna tidak perlu pelatihan ulang |

## 22.5 Peringatan atas ukuran yang dipilih

> **Ukuran keberhasilan ini belum dapat dibuktikan untuk lima modul.** Kesetaraan menuntut kita
> mengetahui apa hasil yang benar. Untuk FR-B5, FR-B7, FR-B9, FR-B10, dan FR-B12, hasil yang
> benar ditentukan oleh **64 procedure yang logikanya belum pernah dibaca siapa pun di tim ini.**
>
> **Kesetaraan atas logika yang tidak diketahui tidak dapat dibuktikan, hanya diduga** (R-14).
>
> Karena itu **menutup R-01 bukan permintaan teknis kepada DBA — ia prasyarat agar project ini
> dapat dinyatakan berhasil menurut ukuran yang dipilih manajemen sendiri.**

---

# 23. Appendix

## 23.1 Glossary istilah domain

Glossary lengkap: [`../Steering/CONTEXT.md`](../Steering/CONTEXT.md). Istilah yang paling sering
dipakai di dokumen ini:

| Istilah | Arti |
|---|---|
| **PNC / Non-MBU** | Non-Motor Business Unit. **PNC adalah sinonim Non-MBU** — bukan singkatan teknis lain |
| **Group Panel** | Kode segmentasi lini bisnis yang menyetir hampir seluruh percabangan aturan |
| **Objek Pertanggungan** | Barang atau orang yang dipertanggungkan dan terdampak kerugian |
| **Coverage / Jaminan** | Jenis jaminan pada satu Objek Pertanggungan, beserta TSI dan penyebab kerugian yang berlaku |
| **TSI** | Total Sum Insured — nilai pertanggungan. Nilai klaim tidak boleh melebihinya |
| **DOL** | Date of Loss / Tanggal Kejadian — **titik acuan hampir seluruh aturan tanggal** |
| **Snapshot Polis** | Salinan data polis pada saat klaim diregistrasi. **Ini yang membuat domain Klaim berdiri sendiri** |
| **Settlement Line** | Baris penyelesaian nilai untuk satu Coverage — menyimpan perjalanan nilai uang klaim |
| **Akseptasi** | Persetujuan atas **nilai** klaim yang akan dibayarkan. **Berbeda** dari persetujuan bahwa klaim dijamin |
| **Spreading** | Pembagian risiko satu Coverage ke para penanggung. **Total wajib 100%** |
| **PLA / Pre-DLA / DLA** | Pemberitahuan nilai estimasi / akan diaksep / sudah diaksep — ke **koasuransi/reasuransi** |
| **LOD** | Letter of Discharge — pemberitahuan nilai ganti rugi kepada **tertanggung** |
| **Komite** | Forum persetujuan berjenjang atas nilai klaim, sampai 4 level |
| **RCL / PUCL** | Rejected Klaim / Proses Ulang Klaim |
| **Ex-Gratia** | Pembayaran kebijakan di luar kewajiban polis |
| **Salvage / Recovery** | Nilai sisa barang rusak yang dapat dijual / pemulihan dana dari pihak ketiga |
| **OS** | Outstanding — klaim yang sudah diakui nilainya tetapi belum selesai dibayar |
| **SLIK OJK** | Pelaporan ke regulator, wajib untuk lini SPK |
| **TAT** | Turn Around Time — waktu penyelesaian klaim |

### Istilah yang sengaja tidak dipakai lagi

| Istilah lama | Alasan ditinggalkan | Pengganti |
|---|---|---|
| `Object` / `ObjectList` | Bertabrakan dengan makna pemrograman | Objek Pertanggungan |
| `Adjustment` / `AdjustmentList` | **Menyesatkan** — isinya nilai penyelesaian, bukan proses loss adjusting | Settlement Line |
| `CaseID` | **Dipakai untuk 5 hal berbeda** di SQL yang berbeda | Nama eksplisit per konteks |
| `pzInsKey` / `CLAIMID` berprefix Pega | Kunci teknis Pega bocor ke data bisnis | Nomor Klaim sebagai identitas bisnis |
| Alias kolom SQL warisan | Nama tidak mencerminkan isi | Penamaan ulang menyeluruh |

## 23.2 Dokumen pendukung

| Dokumen | Isi |
|---|---|
| [`../Steering/STEERING.md`](../Steering/STEERING.md) | **Blueprint teknis lengkap** v2.0 — Riwayat Revisi + 16 bab + Lampiran A–F |
| [`../Steering/README.md`](../Steering/README.md) | Peta 24 dokumen Steering dan cara membacanya per peran |
| [`../Steering/CONTEXT.md`](../Steering/CONTEXT.md) | Glossary domain lengkap — **70 istilah**, wajib dibaca lebih dulu |
| [`../Steering/00-DECISION-LOG.md`](../Steering/00-DECISION-LOG.md) | **72 keputusan** dalam 3 sesi: pertanyaan, pilihan, jawaban, keputusan akhir, dampaknya |
| [`../ADR/README.md`](../ADR/README.md) | **29 Architecture Decision Record** — 24 `Accepted`, 5 `Proposed` |
| [`../Steering/21-RIWAYAT-REVISI.md`](../Steering/21-RIWAYAT-REVISI.md) | **Apa yang berubah v1.0 → v2.0 dan atas dasar apa** |
| [`../Steering/02-BUSINESS-UNDERSTANDING.md`](../Steering/02-BUSINESS-UNDERSTANDING.md) | Proses bisnis, aturan, peran, aliran nilai klaim |
| [`../Steering/16-RISK-ANALYSIS.md`](../Steering/16-RISK-ANALYSIS.md) | **19 risiko** beserta penanganannya |
| [`../Steering/19-GAP-EXPORT-DETAIL.md`](../Steering/19-GAP-EXPORT-DETAIL.md) | **Daftar lengkap yang hilang dari export** — 64 procedure/function, 8 router, 50 activity, beserta query siap pakai |
| [`../Steering/20-DETAIL-KOMITE-DBLINK.md`](../Steering/20-DETAIL-KOMITE-DBLINK.md) | Rincian matriks komite + **inventaris 64 pemakaian DB Link** dan usulan 6 API penggantinya |
| [`understanding-log.md`](../understanding-log.md) | Hasil analisis pemahaman per 19 aspek |
| [`requirement-summary.md`](../requirement-summary.md) | **162 requirement bernomor** beserta status validasinya |
| [`interview-history.md`](../interview-history.md) | Riwayat lengkap 3 sesi discovery |
| [`migration-readiness.md`](../migration-readiness.md) | **Verdict kesiapan migrasi** beserta checklist dan ambang READY |
| [`skill-usage.md`](../skill-usage.md) | Catatan penggunaan skill sepanjang pekerjaan |

## 23.3 Keputusan arsitektur utama

| Aspek | Keputusan |
|---|---|
| Backend | Go 1.22+, `chi`, `database/sql`, **tanpa ORM** |
| Frontend | React 18 + TypeScript + Vite + TanStack Table/AG Grid, **SPA murni** |
| Database sasaran | **PostgreSQL 17+** — mengikat |
| Database sementara | Oracle 19c dengan **satu set SQL portabel** |
| Stored procedure | **Tidak dipanggil sama sekali**; logikanya naik ke Go |
| Data polis | **Snapshot** saat registrasi, bukan pemanggilan runtime |
| Cutover | **Strangler Fig**, satu database bersama, kepemilikan tabel per modul |
| Nomor klaim | **`PNCN.YY.xxxx`** dari sequence |
| Authentication | API internal **HCC/HCQ** |
| Authorization | **Dimiliki aplikasi** — tabel peran dan izin baru |
| Deployment | 2 VM on-premise di belakang load balancer |
| Ketersediaan | **24/7**, rolling deployment tanpa downtime |
| Nilai bisnis | **Tidak ada hardcode** — seluruhnya master data |
| Jejak audit | **Append-only**, dirancang sejak awal |
| Laporan | PDF/Excel/CSV **dibuat sendiri di Go** |
| Tampilan | **Meniru alur dan tata letak Pega** |

## 23.4 Keputusan yang masih terbuka

| Perlu dilengkapi | Kepada |
|---|---|
| Matriks penjenjangan komite (nilai × jenis bisnis) | DBA + tim bisnis |
| Daftar job terjadwal | Tim Pega |
| ~~Arti kode status~~ — **sudah diterima**, 33 kode `1134`–`1166` | ~~DBA~~ |
| Kontrak 6 API pengganti DB Link | Tim pemilik sistem |
| Lama retensi data audit | Compliance |
| Angka RPO / RTO korporat | Tim infra |
| Aturan 3 router penugasan | Tim Pega |
| Aturan validasi sisa TSI | Tim Pega + tim bisnis |

---

## Lembar Persetujuan

> **Wajib diisi sebelum dokumen dipresentasikan.** Atas permintaan pemilik project, dokumen ini
> memakai jabatan generik tanpa nama orang (Bab 8). Nama dan tanda tangan harus dilengkapi agar
> komitmen yang tercatat di Bab 8.3 benar-benar mengikat pihak yang bersangkutan.

| Peran | Nama | Tanggal | Tanda tangan |
|---|---|---|---|
| **Sponsor Project** | | | |
| **Business Owner Claim PNC** | | | |
| **Manajemen Teknologi Informasi** | | | |
| **Technical Architect / Tech Lead** | | | |

### Yang disetujui dengan menandatangani dokumen ini

| # | Butir persetujuan |
|---|---|
| 1 | **Scope** (Bab 6) dan **Out of Scope** (Bab 7) — seluruh 34 modul pindah, tanpa pemangkasan |
| 2 | **Functional dan Non Functional Requirement** (Bab 9, 10) beserta 9 requirement yang belum dapat difinalkan |
| 3 | **Business Rules** (Bab 11), termasuk kewajiban **memvalidasi ulang ambang komite** dan mengklarifikasi aturan validasi sisa TSI |
| 4 | **Dua gerbang penerimaan per modul** (Bab 21) dan konsekuensinya pada **alokasi waktu user bisnis** |
| 5 | **Ukuran keberhasilan: kesetaraan fungsional dengan Pega** (Bab 22) — beserta peringatan bahwa ia **belum dapat dibuktikan untuk 5 modul** |
| 6 | **19 risiko** (Bab 20), khususnya **R-01, R-05, R-13, R-14, R-16, R-17** |
| 7 | **Tiga hal yang perlu keputusan manajemen** (Bab 20.4) |
| 8 | **Sepuluh tindakan pembuka** (Bab 20.5) — beserta kewajiban menetapkan **tanggal komitmen** dan **jalur eskalasi** untuk masing-masing |
