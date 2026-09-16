# ADR — Migrasi Aplikasi Claim PNC ke Golang

**Architecture Decision Records · Dokumen Gabungan**

| | |
|---|---|
| **Dokumen** | Architecture Decision Records (ADR) — Migrasi Claim PNC dari Pega PRPC 8.3 ke Golang + React + PostgreSQL |
| **Tanggal dokumen** | 2026-09-14 |
| **Pemilik keputusan** | Work Owner |
| **Jumlah keputusan** | **30** — 25 `Accepted`, 5 `Proposed` |
| **Sumber keputusan** | `docs/Steering/00-DECISION-LOG.md` (70 entri, D-01…D-70) |
| **Sumber bukti** | `docs/verifikasi-bukti-adr.md` · export rule Pega (baca-saja) |
| **Sifat seluruh ADR** | `retrospective` — keputusan diambil 2026-09-07 … 2026-09-14, dokumen ditulis 2026-09-14 |

> **Dokumen ini dibangun otomatis** dari `docs/ADR/README.md` dan `docs/ADR/00NN-*.md`.
> Berkas sumber tetap menjadi acuan; dokumen gabungan ini untuk dibaca dan diedarkan.

---

## Daftar Isi

1. [Ringkasan Eksekutif](#1-ringkasan-eksekutif)
2. [Index Keputusan](#2-index-keputusan)
3. [Kosakata Status](#3-kosakata-status)
4. [Hubungan dengan Dokumen Lain](#4-hubungan-dengan-dokumen-lain)
5. [Kelompok A — Fondasi & Topologi](#5-kelompok-a-fondasi-topologi)
    - [0001 — Bangun satu modular monolith Go yang dijalankan di VM on-premise](#0001-bangun-satu-modular-monolith-go-yang-dijalankan-di-vm-on-premise)
    - [0002 — Bangun antarmuka sebagai SPA React yang disajikan oleh binary Go](#0002-bangun-antarmuka-sebagai-spa-react-yang-disajikan-oleh-binary-go)
    - [0003 — Alihkan modul satu per satu dengan Pega dan Go berjalan paralel](#0003-alihkan-modul-satu-per-satu-dengan-pega-dan-go-berjalan-paralel)
    - [0004 — Pakai satu database bersama selama masa paralel, dengan penulis tunggal per tabel](#0004-pakai-satu-database-bersama-selama-masa-paralel-dengan-penulis-tunggal-per-tabel)
    - [0005 — Tulis satu set SQL portabel: jalan di Oracle sekarang, PostgreSQL 17+ kemudian](#0005-tulis-satu-set-sql-portabel-jalan-di-oracle-sekarang-postgresql-17-kemudian)
    - [0006 — Bekukan data polis sebagai snapshot saat registrasi; kepemilikannya tetap di GISFW](#0006-bekukan-data-polis-sebagai-snapshot-saat-registrasi-kepemilikannya-tetap-di-gisfw)
6. [Kelompok B — Data, Integrasi, dan Artefak](#6-kelompok-b-data-integrasi-dan-artefak)
    - [0007 — Naikkan logika stored procedure ke Go dan pindahkan kepemilikan transaksi ke aplikasi](#0007-naikkan-logika-stored-procedure-ke-go-dan-pindahkan-kepemilikan-transaksi-ke-aplikasi)
    - [0008 — Ganti seluruh akses DB Link dengan pemanggilan API ke sistem pemilik data](#0008-ganti-seluruh-akses-db-link-dengan-pemanggilan-api-ke-sistem-pemilik-data)
    - [0009 — Terbitkan nomor klaim baru berformat `PNCN.YY.xxxx` dari sequence database](#0009-terbitkan-nomor-klaim-baru-berformat-pncnyyxxxx-dari-sequence-database)
    - [0010 — Simpan dokumen lewat satu jalur API storage internal; database hanya menyimpan metadata](#0010-simpan-dokumen-lewat-satu-jalur-api-storage-internal-database-hanya-menyimpan-metadata)
    - [0011 — Bangun pembuatan PDF, Excel, dan CSV di dalam aplikasi Go](#0011-bangun-pembuatan-pdf-excel-dan-csv-di-dalam-aplikasi-go)
    - [0012 — Terapkan soft delete menyeluruh; tidak ada penghapusan fisik data bernilai bisnis](#0012-terapkan-soft-delete-menyeluruh-tidak-ada-penghapusan-fisik-data-bernilai-bisnis)
    - [0013 — Tentukan pengganti pola hapus-lalu-sisip-ulang pada konversi klaim](#0013-tentukan-pengganti-pola-hapus-lalu-sisip-ulang-pada-konversi-klaim)
7. [Kelompok C — Aturan Bisnis Bernilai Uang dan Kewenangan](#7-kelompok-c-aturan-bisnis-bernilai-uang-dan-kewenangan)
    - [0014 — Hitung jenjang komite secara kumulatif menurut ambang bawah, dengan pita nilai hanya di Non-MBU](#0014-hitung-jenjang-komite-secara-kumulatif-menurut-ambang-bawah-dengan-pita-nilai-hanya-di-non-mbu)
    - [0015 — Konversi mata uang memakai kurs tanggal kejadian; tolak klaim bila kurs tidak ditemukan](#0015-konversi-mata-uang-memakai-kurs-tanggal-kejadian-tolak-klaim-bila-kurs-tidak-ditemukan)
    - [0016 — Simpan nilai uang presisi penuh; validasi total spreading dengan toleransi empat desimal](#0016-simpan-nilai-uang-presisi-penuh-validasi-total-spreading-dengan-toleransi-empat-desimal)
    - [0017 — Perbaiki sembilan cacat aturan uang; replikasi satu yang perilakunya memang benar](#0017-perbaiki-sembilan-cacat-aturan-uang-replikasi-satu-yang-perilakunya-memang-benar)
    - [0018 — Pertahankan empat konsep status, masing-masing dengan nama yang tidak dapat tertukar](#0018-pertahankan-empat-konsep-status-masing-masing-dengan-nama-yang-tidak-dapat-tertukar)
    - [0019 — Pertahankan Worklist dan Workbasket; bangun ulang router sebagai aturan routing](#0019-pertahankan-worklist-dan-workbasket-bangun-ulang-router-sebagai-aturan-routing)
    - [0020 — Pertahankan dua basis perhitungan TAT untuk keperluan yang berbeda](#0020-pertahankan-dua-basis-perhitungan-tat-untuk-keperluan-yang-berbeda)
8. [Kelompok D — Mekanisme Pega yang Harus Diganti](#8-kelompok-d-mekanisme-pega-yang-harus-diganti)
    - [0021 — Bangun ulang transisi lateral (Ticket rule) sebagai perpindahan tahap yang eksplisit](#0021-bangun-ulang-transisi-lateral-ticket-rule-sebagai-perpindahan-tahap-yang-eksplisit)
    - [0022 — Tentukan cara menjalankan job terjadwal di aplikasi yang hidup di dua instans](#0022-tentukan-cara-menjalankan-job-terjadwal-di-aplikasi-yang-hidup-di-dua-instans)
9. [Kelompok E — Keamanan, Konfigurasi, dan Audit](#9-kelompok-e-keamanan-konfigurasi-dan-audit)
    - [0023 — Tegakkan otorisasi berbasis menu di server, dengan 22 peran dan tanpa pemisahan tugas](#0023-tegakkan-otorisasi-berbasis-menu-di-server-dengan-22-peran-dan-tanpa-pemisahan-tugas)
    - [0024 — Delegasikan autentikasi ke HCC/HCQ dan terbitkan sesi milik aplikasi sendiri](#0024-delegasikan-autentikasi-ke-hcchcq-dan-terbitkan-sesi-milik-aplikasi-sendiri)
    - [0025 — Pindahkan seluruh nilai hardcode ke master data dan konfigurasi, dan keluarkan rahasia dari kode](#0025-pindahkan-seluruh-nilai-hardcode-ke-master-data-dan-konfigurasi-dan-keluarkan-rahasia-dari-kode)
    - [0026 — Catat setiap perubahan bernilai bisnis sebagai jejak audit append-only](#0026-catat-setiap-perubahan-bernilai-bisnis-sebagai-jejak-audit-append-only)
10. [Kelompok F — Mutu dan Data Uji](#10-kelompok-f-mutu-dan-data-uji)
    - [0027 — Luluskan setiap modul lewat dua gerbang: uji kesetaraan otomatis, lalu UAT bisnis](#0027-luluskan-setiap-modul-lewat-dua-gerbang-uji-kesetaraan-otomatis-lalu-uat-bisnis)
    - [0028 — Ukur modul tanpa baseline Pega dengan kontrak fungsional, dan cabut `BRD §21.4` untuk empat modul](#0028-ukur-modul-tanpa-baseline-pega-dengan-kontrak-fungsional-dan-cabut-brd-214-untuk-empat-modul)
    - [0029 — Salin data nasabah apa adanya ke staging; samarkan email dan jangan pernah tulis data nasabah di dokumen](#0029-salin-data-nasabah-apa-adanya-ke-staging-samarkan-email-dan-jangan-pernah-tulis-data-nasabah-di-dokumen)

- [Lampiran A — Template ADR Kosong](#lampiran-a--template-adr-kosong)
- [Lampiran B — Aturan Menulis yang Mengikat](#lampiran-b--aturan-menulis-yang-mengikat)
- [Lampiran C — Rekapitulasi Seluruh Pertanyaan Terbuka](#lampiran-c--rekapitulasi-seluruh-pertanyaan-terbuka)

---

## 1. Ringkasan Eksekutif

### 1.1 Apa isi dokumen ini

Satu berkas ADR = satu keputusan yang **mahal dibalik**. Uji kelayakan yang dipakai menyaring:
*kalau keputusan ini dibalik enam bulan lagi, adakah arsitektur yang harus dibongkar besar atau
risiko bisnis yang muncul?* Dari **70 keputusan** di Decision Log, **30 lolos** dan 41 tidak —
yang tidak lolos tetap hidup sebagai `D-nn` di `docs/Steering/00-DECISION-LOG.md`.

### 1.2 Papan status

| Kelompok | Jumlah | `Accepted` | `Proposed` |
|---|---:|---:|---:|
| **A** — Fondasi & Topologi | 6 | 6 | 0 |
| **B** — Data, Integrasi, dan Artefak | 7 | 6 | 1 |
| **C** — Aturan Bisnis Bernilai Uang dan Kewenangan | 7 | 7 | 0 |
| **D** — Mekanisme Pega yang Harus Diganti | 2 | 0 | 2 |
| **E** — Keamanan, Konfigurasi, dan Audit | 4 | 2 | 2 |
| **F** — Mutu dan Data Uji | 3 | 3 | 0 |
| **Total** | **30** | **25** | **5** |

### 1.3 Lima keputusan yang belum diambil, dan apa yang terhalang

Kelima ADR berikut berstatus `Proposed`: memuat konteks, opsi, dan pemilik keputusan — **tanpa
bagian `Keputusan`**. Tidak satu pun boleh dijadikan dasar implementasi.

| ADR | Yang belum diputuskan | Pemilik keputusan | Menghalangi |
|---|---|---|---|
| **0013** | Pengganti pola hapus-lalu-sisip-ulang pada konversi klaim | Work Owner + Lead Engineer | tiket `B-2` |
| **0021** | 8 Ticket rule custom hilang; 14 dari 17 nama tanpa pemicu | Tim Pega → Work Owner | `B-7`, `B-11`, `B-13`, `B-14` |
| **0022** | Mekanisme penjadwal saat aplikasi hidup di dua instans; nasib `AutoAcceptKomite` | Lead Engineer + Infra · Work Owner | `S-6` |
| **0024** | Kontrak API HCC/HCQ — nol jejak di export | Tim HCC/HCQ → Work Owner | `F-3`, dan login seluruh aplikasi |
| **0025** | Ke mana rahasia dipindahkan dan siapa pemiliknya (`D-40` OPEN) | Tim Infra/Security | `F-4`, `F-5`, seluruh deployment |

### 1.4 Keputusan yang menyupersede dokumen lain

| ADR | Menyupersede | Keterangan |
|---|---|---|
| **0012** | `D-65` | Soft delete menyeluruh menggantikan kebijakan penghapusan yang mengikuti sistem lama |
| **0028** | **`BRD §21.4`** untuk `B-7`, `B-9`, `B-10`, `B-12` | Pasalnya dicabut lewat ADR; **BRD tidak diedit** — usulan revisinya menunggu persetujuan Work Owner |
| **0014** | membatasi cakupan `D-52` | `TYPE_KOMITE` adalah pita nilai **hanya** di lini Non-MBU |
| **0009** | bagian format pada `D-22` | Format nomor klaim `PNCN-xxxx` direvisi menjadi `PNCN.YY.xxxx` oleh `D-71`; sisa isi `D-22` tetap berlaku |

### 1.5 Tiga keputusan paling berisiko

| ADR | Mengapa paling berisiko |
|---|---|
| **0004** Database bersama, penulis tunggal per tabel | Satu-satunya keputusan yang dapat **merusak data produksi** bila salah; mengikat seluruh masa paralel dan hampir mustahil dibalik setelah gelombang 1 berjalan |
| **0007** Stored procedure naik ke Go | Menyentuh 62 procedure + 12 dependensi, memindahkan kepemilikan transaksi, dan menjadi dasar `B-4`/`B-9` dapat dibuat atomik |
| **0014** Komite kumulatif per lini | Menentukan **siapa berwenang menyetujui uang**; mekanismenya terbukti dua kali hampir salah dibaca selama analisis |

---

## 2. Index Keputusan

Kolom `Jenis` dan `Modul terdampak` berada di tabel ini, bukan di dalam berkas ADR — sesuai `D-31`.

| # | Judul | Status | Jenis | Modul terdampak | Pemilik keputusan |
|---|---|---|---|---|---|
| [0001](#0001-bangun-satu-modular-monolith-go-yang-dijalankan-di-vm-on-premise) | Modular monolith Go di VM on-premise, tanpa orkestrator kontainer | Accepted | turunan `D-08` | seluruh modul | Work Owner |
| [0002](#0002-bangun-antarmuka-sebagai-spa-react-yang-disajikan-oleh-binary-go) | SPA React+TypeScript disajikan oleh binary Go | Accepted | turunan `D-23` | `U-1`…`U-6` | Work Owner |
| [0003](#0003-alihkan-modul-satu-per-satu-dengan-pega-dan-go-berjalan-paralel) | Peralihan bertahap Strangler Fig per gelombang modul | Accepted | turunan `D-05` | seluruh modul | Work Owner |
| [0004](#0004-pakai-satu-database-bersama-selama-masa-paralel-dengan-penulis-tunggal-per-tabel) | Satu database dipakai bersama selama masa paralel, penulis tunggal per tabel | Accepted | turunan `D-21` | seluruh modul | Work Owner |
| [0005](#0005-tulis-satu-set-sql-portabel-jalan-di-oracle-sekarang-postgresql-17-kemudian) | Oracle 19c selama paralel, PostgreSQL 17+ setelah cutover, SQL portabel | Accepted | turunan `D-01` | `F-2` + seluruh modul | Work Owner |
| [0006](#0006-bekukan-data-polis-sebagai-snapshot-saat-registrasi-kepemilikannya-tetap-di-gisfw) | Data polis milik GISFW; Claim PNC menyimpan snapshot | Accepted | turunan `D-04` | `B-1`, `B-2` | Work Owner |
| [0007](#0007-naikkan-logika-stored-procedure-ke-go-dan-pindahkan-kepemilikan-transaksi-ke-aplikasi) | Logika stored procedure dinaikkan ke Go; transaksi dimiliki aplikasi | Accepted | turunan `D-02` + `D-68` | `B-4`, `B-5`, `B-9`, `B-10`, `B-12` | Work Owner |
| [0008](#0008-ganti-seluruh-akses-db-link-dengan-pemanggilan-api-ke-sistem-pemilik-data) | DB Link diganti kontrak API eksplisit | Accepted | turunan `D-25` | `S-4`, `B-7`, `B-12` | Work Owner |
| [0009](#0009-terbitkan-nomor-klaim-baru-berformat-pncnyyxxxx-dari-sequence-database) | Nomor klaim baru `PNCN.YY.xxxx` dari sequence; prefix Pega ditinggalkan | Accepted | turunan `D-22`, **menyupersede format pada `D-22`** (`D-71`) | `B-2` | Work Owner |
| [0010](#0010-simpan-dokumen-lewat-satu-jalur-api-storage-internal-database-hanya-menyimpan-metadata) | Dokumen lewat satu jalur API storage internal; DB hanya metadata | Accepted | turunan `D-16` | `S-1` | Work Owner |
| [0011](#0011-bangun-pembuatan-pdf-excel-dan-csv-di-dalam-aplikasi-go) | PDF/Excel/CSV dibangun di dalam aplikasi Go | Accepted | turunan `D-11` | `S-2`, `U-5` | Work Owner |
| [0012](#0012-terapkan-soft-delete-menyeluruh-tidak-ada-penghapusan-fisik-data-bernilai-bisnis) | Soft delete menyeluruh — tanpa penghapusan fisik data bernilai bisnis | Accepted | turunan `D-66`, **menyupersede `D-65`** | `B-2`, `S-5`, `S-8` | Work Owner |
| [0013](#0013-tentukan-pengganti-pola-hapus-lalu-sisip-ulang-pada-konversi-klaim) | Pengganti pola hapus-lalu-sisip-ulang pada konversi klaim | **Proposed** | baru | `B-2` | Work Owner + Lead Engineer |
| [0014](#0014-hitung-jenjang-komite-secara-kumulatif-menurut-ambang-bawah-dengan-pita-nilai-hanya-di-non-mbu) | Komite kumulatif menurut ambang bawah; pita nilai hanya Non-MBU | Accepted | turunan `D-47` + `D-52` + `D-70` | `B-7` | Work Owner |
| [0015](#0015-konversi-mata-uang-memakai-kurs-tanggal-kejadian-tolak-klaim-bila-kurs-tidak-ditemukan) | Kurs pada tanggal kejadian; klaim ditolak bila kurs tidak ditemukan | Accepted | turunan `D-48` | `B-2`, `B-5` | Work Owner |
| [0016](#0016-simpan-nilai-uang-presisi-penuh-validasi-total-spreading-dengan-toleransi-empat-desimal) | Uang disimpan presisi penuh; toleransi total spreading | Accepted | turunan `D-51` | `B-4`, `B-5` | Work Owner |
| [0017](#0017-perbaiki-sembilan-cacat-aturan-uang-replikasi-satu-yang-perilakunya-memang-benar) | Sembilan cacat aturan uang diperbaiki, satu direplikasi secara sadar | Accepted | turunan `D-49` | `B-2`…`B-10` | Work Owner |
| [0018](#0018-pertahankan-empat-konsep-status-masing-masing-dengan-nama-yang-tidak-dapat-tertukar) | Empat konsep status dipertahankan dengan nama yang tidak tertukar | Accepted | turunan `D-18` | seluruh modul bisnis | Work Owner |
| [0019](#0019-pertahankan-worklist-dan-workbasket-bangun-ulang-router-sebagai-aturan-routing) | Worklist/Workbasket dipertahankan; router dibangun ulang sebagai aturan routing | Accepted | turunan `D-26` | `B-6` | Work Owner |
| [0020](#0020-pertahankan-dua-basis-perhitungan-tat-untuk-keperluan-yang-berbeda) | Dua basis perhitungan TAT dipertahankan untuk keperluan berbeda | Accepted | turunan `D-50` | `S-2`, `S-7` | Work Owner |
| [0021](#0021-bangun-ulang-transisi-lateral-ticket-rule-sebagai-perpindahan-tahap-yang-eksplisit) | Transisi lateral (Ticket rule) dibangun ulang sebagai perpindahan tahap eksplisit | **Proposed** | baru | `B-7`, `B-11`, `B-13`, `B-14` | Tim Pega → Work Owner |
| [0022](#0022-tentukan-cara-menjalankan-job-terjadwal-di-aplikasi-yang-hidup-di-dua-instans) | Job terjadwal dibangun ulang sebagai penjadwal aplikasi | **Proposed** | turunan `D-57` | `S-6` | Lead Engineer + Infra |
| [0023](#0023-tegakkan-otorisasi-berbasis-menu-di-server-dengan-22-peran-dan-tanpa-pemisahan-tugas) | Otorisasi berbasis menu ditegakkan di server; 22 peran; tanpa pemisahan tugas | Accepted | turunan `D-59` + `D-58` | `F-3` | Work Owner |
| [0024](#0024-delegasikan-autentikasi-ke-hcchcq-dan-terbitkan-sesi-milik-aplikasi-sendiri) | Autentikasi didelegasikan ke HCC/HCQ; aplikasi menerbitkan sesinya sendiri | **Proposed** | turunan `D-07` | `F-3` | Tim HCC/HCQ → Work Owner |
| [0025](#0025-pindahkan-seluruh-nilai-hardcode-ke-master-data-dan-konfigurasi-dan-keluarkan-rahasia-dari-kode) | Seluruh hardcode menjadi master/konfigurasi; rahasia keluar dari kode | **Proposed** | turunan `D-15` + `D-40` | `F-4`, `F-5` | Tim Infra/Security |
| [0026](#0026-catat-setiap-perubahan-bernilai-bisnis-sebagai-jejak-audit-append-only) | Jejak audit append-only; retensi mengikuti retensi data klaim | Accepted | turunan `D-28` + `D-62` | `S-5` | Work Owner |
| [0027](#0027-luluskan-setiap-modul-lewat-dua-gerbang-uji-kesetaraan-otomatis-lalu-uat-bisnis) | Dua gerbang penerimaan; uji kesetaraan otomatis sebagai gerbang 1 | Accepted | turunan `D-42` + `D-53` + `D-54` + `D-60` | seluruh modul | Work Owner |
| [0028](#0028-ukur-modul-tanpa-baseline-pega-dengan-kontrak-fungsional-dan-cabut-brd-214-untuk-empat-modul) | Modul tanpa baseline Pega diukur terhadap kontrak fungsional | Accepted | turunan `D-56` + `D-55` · **menyupersede `BRD §21.4`** | `F-3`, `S-5` | Work Owner |
| [0029](#0029-salin-data-nasabah-apa-adanya-ke-staging-samarkan-email-dan-jangan-pernah-tulis-data-nasabah-di-dokumen) | Data nasabah nyata dipakai di staging; aturan penyamaran di dokumen | Accepted | turunan `D-64` + `D-69` | `S-8` + seluruh dokumen | Work Owner |

---

## 3. Kosakata Status

| Status | Artinya |
|---|---|
| `Accepted` | Work Owner sudah memutuskan; keputusan mengikat tiket dan implementasi |
| `Proposed` | **Belum** diputuskan. Berkasnya memuat konteks, opsi, pemilik keputusan, dan apa yang diblokir — **tanpa** bagian `Keputusan`. Tidak boleh dijadikan dasar implementasi |
| `Superseded by ADR-XXXX` | Digantikan ADR lain; isinya dibiarkan utuh sebagai jejak |
| `Deprecated` | Tidak lagi berlaku dan tidak diganti |

---

## 4. Hubungan dengan Dokumen Lain

| Dokumen | Perannya |
|---|---|
| `docs/Steering/00-DECISION-LOG.md` | Sumber seluruh `D-nn`. ADR **tidak** menggantikannya |
| `docs/Steering/CONTEXT.md` | Kamus istilah. ADR memakai istilahnya, tidak mendefinisikan ulang |
| `docs/verifikasi-bukti-adr.md` | Bukti Fase 1 — `berkas:baris`, angka, dan kontradiksi `K-nn` |
| `docs/BRD.md` | Sumber `FR-xx` |
| `docs/ticketing/` | Tiket pekerjaan; setiap tiket merujuk ADR yang mengikatnya |

> **Peringatan istilah.** Folder `Ticket/` pada export Pega berisi **Ticket rule** — mekanisme
> lompatan lateral (lihat ADR 0021). Kata **"tiket"** di seluruh dokumen proyek ini selalu berarti
> **tiket pekerjaan** di `docs/ticketing/`.

---

## 5. Kelompok A — Fondasi & Topologi

Keputusan yang menentukan bentuk aplikasi secara keseluruhan: bahasa, bentuk artefak, tempat ia berjalan, cara ia menggantikan sistem lama, dan di mana batas kepemilikan datanya.

| # | Judul | Status |
|---|---|---|
| 0001 | Bangun satu modular monolith Go yang dijalankan di VM on-premise | Accepted |
| 0002 | Bangun antarmuka sebagai SPA React yang disajikan oleh binary Go | Accepted |
| 0003 | Alihkan modul satu per satu dengan Pega dan Go berjalan paralel | Accepted |
| 0004 | Pakai satu database bersama selama masa paralel, dengan penulis tunggal per tabel | Accepted |
| 0005 | Tulis satu set SQL portabel: jalan di Oracle sekarang, PostgreSQL 17+ kemudian | Accepted |
| 0006 | Bekukan data polis sebagai snapshot saat registrasi; kepemilikannya tetap di GISFW | Accepted |

### 0001 — Bangun satu modular monolith Go yang dijalankan di VM on-premise

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-07 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-08`, `D-09`, `D-10`, `D-27`, `D-29` · `docs/verifikasi-bukti-adr.md` §9, §10 |
| **Terkait** | ADR-0002, ADR-0003, ADR-0004, ADR-0022, modul `F-1` |

#### Konteks

Aplikasi lama adalah Pega PRPC 8.3 — satu platform yang menyatukan alur kerja, antarmuka,
integrasi, dan penjadwalan. Penggantinya harus berjalan di tempat yang sama: **data center
Sinarmas**, bukan cloud publik.

Tiga fakta membentuk keputusan ini:

- **Tidak ada Kubernetes dan tidak boleh diasumsikan ada** (`D-08`). Target deployment adalah
  VM atau bare metal.
- **Tim adalah developer Pega internal yang dilatih ulang** (`D-09`). Prioritasnya learning curve
  landai dan pola seragam, bukan arsitektur paling canggih.
- **Beban data besar, konkurensi rendah** (`D-10`): ribuan klaim per bulan, data historis puluhan
  juta baris, tetapi hanya **200–300 pengguna aktif harian**.

Sementara itu `D-27` menuntut ketersediaan **24/7 termasuk saat deployment**, yang berarti
minimal dua instans di belakang load balancer dan aplikasi yang **stateless**.

#### Opsi yang dipertimbangkan

1. **Modular monolith Go**, satu binary, modul dipisah lewat batas paket dan seam repository.
2. **Microservices** per domain (klaim, komite, dokumen, laporan).
3. **Monolith tanpa batas modul** — paling cepat ditulis, paling cepat rusak.

#### Keputusan

Aplikasi dibangun sebagai **satu modular monolith Go**, dikompilasi menjadi **satu binary** yang
dijalankan di VM on-premise, **tanpa mengasumsikan orkestrator kontainer apa pun**.

Batas modul ditegakkan di dalam kode — bukan lewat jaringan — mengikuti pembagian `F-*`, `B-*`,
`S-*`, `U-*` pada Module Breakdown. Aplikasi wajib **stateless**, menyediakan **health check
endpoint** dan **graceful shutdown**, sehingga dua instans dapat di-update bergantian.

#### Rationale

Konkurensi rendah dengan volume data besar adalah profil yang **tidak** menuntut pemisahan
proses. Microservices akan menambah biaya operasional (service discovery, tracing terdistribusi,
transaksi lintas layanan) tanpa menyelesaikan masalah yang benar-benar dimiliki sistem ini —
yaitu kueri berat di atas data puluhan juta baris.

Satu binary juga satu-satunya bentuk yang realistis dijalankan tim yang baru meninggalkan Pega:
satu artefak untuk di-deploy, satu proses untuk dipantau, satu tempat untuk mencari galat.

Batas modul tetap ditegakkan agar kelak, bila benar-benar dibutuhkan, satu modul dapat dipisah
tanpa membongkar seluruhnya.

#### Konsekuensi

##### Positif

- Deployment sederhana: salin satu binary, jalankan ulang bergantian di dua VM.
- Transaksi database lintas modul tetap dapat dibuat **atomik dalam satu proses** — prasyarat
  bagi ADR-0007.
- Tidak ada biaya belajar orkestrator bagi tim yang sedang belajar Go dan React sekaligus.

##### Negatif / utang teknis

- **Seluruh modul di-deploy bersama.** Perbaikan kecil di satu laporan tetap menuntut rilis
  seluruh aplikasi.
- **Skala hanya vertikal per instans.** Satu modul yang boros memori — pembuatan Excel bervolume
  besar pada `S-2` — memengaruhi seluruh proses.
- Batas modul hanya sekuat disiplin tim. Tanpa pemeriksaan otomatis atas ketergantungan
  antarpaket, monolith modular berubah menjadi monolith biasa tanpa ada yang menyadarinya.
- Penjadwalan job tidak lagi mendapat mekanisme cluster bawaan seperti `pyApplicableTo=Cluster`
  milik Pega — konsekuensi ini ditangani ADR-0022, dan **belum terselesaikan**.

##### Risiko yang diterima secara sadar

- Bila kelak satu modul benar-benar membutuhkan skala terpisah, pemisahannya adalah pekerjaan
  rekayasa tersendiri, bukan konfigurasi.
- `D-29` menetapkan RPO/RTO mengikuti kebijakan backup korporat, sementara `D-27` menuntut 24/7.
  Keduanya menjawab hal berbeda, dan **kesesuaiannya belum diverifikasi** ke tim infra.

#### Pertanyaan terbuka

- Apakah kebijakan backup korporat yang berlaku hari ini memang sejalan dengan tuntutan 24/7
  (`D-29`)? Pemilik: Tim Infra. Selama belum dijawab, target pemulihan aplikasi tidak dapat
  dinyatakan di tiket mana pun.
- Bagaimana dua instans berbagi berkas sementara (hasil export besar) bila tidak ada storage
  bersama? Pemilik: Lead Engineer + Infra. Menghalangi bagian export bervolume besar di `S-2`.

### 0002 — Bangun antarmuka sebagai SPA React yang disajikan oleh binary Go

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-07 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-23`, `D-13`, `D-09`, `D-12` · `docs/verifikasi-bukti-adr.md` §10.7 (T-11, T-12) |
| **Terkait** | ADR-0001, ADR-0011, modul `U-1`…`U-6` |

#### Konteks

Antarmuka Pega adalah antarmuka **padat grid**: dari 269 section pada export, **268 memuat
grid**. `D-13` menetapkan tata letak, urutan langkah, dan penempatan field **mengikuti Pega yang
ada**, supaya pengguna tidak perlu dilatih ulang — sehingga kepadatan itu ikut terwarisi.

Dua temuan Fase 1 mengoreksi gambaran awal tentang beratnya grid:

- **T-11** — spesifikasi "tabel baku 18–27 kolom" salah sasaran. **Median kolom sebenarnya 6**,
  dan tiga fitur grid yang biasanya mahal — tambah baris inline, hapus baris inline, dan resize
  kolom — **tidak dipakai sama sekali**. `U-2` lebih ringan daripada yang diperkirakan.
- **T-12** — masalah paginasi bukan `OFFSET` besar (`OFFSET` **nol kemunculan** di export),
  melainkan **3.189 grid yang terikat page list klipboard** dan `pyMaxRecords=500` pada 54 dari
  56 laporan.

`D-09` membatasi pilihan: tim adalah developer Pega yang dilatih ulang, sehingga jumlah konsep
baru harus sedikit. `D-08` melarang mengasumsikan infrastruktur tambahan.

#### Opsi yang dipertimbangkan

1. **SPA React + TypeScript + Vite**, disajikan sebagai berkas statis oleh binary Go.
2. **Server-side rendering** (Next.js atau sejenis) — menuntut runtime Node.js di produksi.
3. **Template HTML dari Go** (`html/template`) + JavaScript seperlunya.

#### Keputusan

Antarmuka dibangun sebagai **SPA React + TypeScript + Vite**, dikompilasi menjadi berkas statis
dan **disajikan langsung oleh binary Go** (ADR-0001). **Tidak ada runtime Node.js di produksi.**

Kebutuhan grid berat ditangani satu pustaka tabel yang dipilih di awal — **TanStack Table** atau
**AG Grid** — dan dipakai seragam lewat `U-2` Pustaka Komponen. Tidak ada modul yang membangun
tabelnya sendiri.

#### Rationale

Menyajikan berkas statis dari binary Go menghapus satu runtime, satu proses, dan satu rantai
pembaruan keamanan dari lingkungan produksi — konsisten dengan `D-08`.

React dipilih bukan karena paling canggih, melainkan karena kumpulan pustaka tabelnya paling
matang untuk kebutuhan yang benar-benar dimiliki aplikasi ini, dan karena materi belajarnya
paling melimpah bagi tim yang sedang berpindah dari Pega.

Satu pustaka tabel yang dipakai seragam adalah konsekuensi langsung dari `D-09`: 268 layar
bergrid yang masing-masing menafsirkan tabelnya sendiri akan menjadi beban pemeliharaan
terbesar aplikasi ini.

#### Konsekuensi

##### Positif

- Produksi hanya menjalankan satu proses: binary Go.
- `U-2` dapat dibangun lebih ramping daripada rencana awal berkat T-11.
- TypeScript memberi pemeriksaan kontrak antara layar dan API sejak waktu kompilasi — penting
  bagi tim yang belum terbiasa dengan JavaScript dinamis.

##### Negatif / utang teknis

- **UX Pega yang padat ikut terwarisi** (`D-13`): grid lebar, form panjang, banyak tab. Kesempatan
  perbaikan UX ditunda ke fase pasca-migrasi, dan penundaan itu akan terasa oleh pengguna.
- **Paginasi keyset adalah perubahan perilaku, bukan pemeliharaan** (T-12). Grid yang hari ini
  memuat seluruh page list klipboard akan berperilaku berbeda saat dipaginasi di server; ini
  harus diuji per layar, bukan diasumsikan setara.
- Rendering awal bergantung pada JavaScript. Tidak ada halaman yang dapat dibaca tanpa SPA
  termuat lebih dulu.
- Memilih AG Grid versi komersial akan menambah lisensi; memilih TanStack Table menambah
  pekerjaan membangun perilaku grid sendiri. Pilihan finalnya **belum dibuat**.

##### Risiko yang diterima secara sadar

- Tim menanggung dua kurva belajar sekaligus — Go dan React — pada proyek yang sama.
- Menyalin tata letak Pega berarti ikut menyalin kekakuannya; beberapa layar akan terasa tidak
  wajar di web dan tetap dibiarkan demi menghindari pelatihan ulang.

#### Pertanyaan terbuka

- TanStack Table atau AG Grid? Pemilik: Lead Engineer, dengan persetujuan Work Owner bila
  berbiaya lisensi. Menghalangi penyelesaian tiket `U-2`.
- `D-12` menetapkan pemakaian lapangan untuk survei. Apakah `U-3`/`U-4` wajib berfungsi penuh di
  peramban ponsel, atau cukup layar survei `B-8` saja? Pemilik: Work Owner.

### 0003 — Alihkan modul satu per satu dengan Pega dan Go berjalan paralel

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-07 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-05`, `D-30`, `D-61`, `D-42` · `docs/verifikasi-bukti-adr.md` §10, §15 |
| **Terkait** | ADR-0004, ADR-0027, seluruh modul |

#### Konteks

Claim PNC adalah sistem yang sedang melayani produksi. Menggantinya sekaligus berarti satu
tanggal ketika seluruh alur klaim — registrasi, estimasi, komite, akseptasi, pembayaran — berpindah
bersamaan, tanpa jalan kembali.

`D-05` menolak pendekatan itu: **Pega dan aplikasi Go berjalan paralel, modul dialihkan satu per
satu.** Konsekuensi yang sudah dicatat sejak keputusan itu diambil: dibutuhkan strategi berbagi
data dan status antara kedua sistem selama masa paralel — ditangani ADR-0004.

#### Opsi yang dipertimbangkan

1. **Strangler Fig** — modul dialihkan bertahap, Pega menyusut sampai habis.
2. **Big bang cutover** — satu tanggal, seluruh aplikasi berpindah.
3. **Jalan paralel penuh** — kedua sistem menerima input yang sama, hasilnya dibandingkan
   terus-menerus sampai kepercayaan terbentuk.

#### Keputusan

Migrasi memakai pola **Strangler Fig**: aplikasi Go mengambil alih modul **satu per satu dalam
gelombang**, sementara Pega tetap melayani modul yang belum dialihkan. Kedua sistem berjalan di
atas **database yang sama** (ADR-0004) selama masa paralel.

Sebuah modul dinyatakan pindah hanya setelah melewati **dua gerbang** (ADR-0027): uji kesetaraan
otomatis, lalu UAT pengguna bisnis.

#### Rationale

Pola ini membuat setiap langkah dapat dibatalkan. Bila satu modul gagal di gerbang mana pun,
yang dikembalikan hanya modul itu — bukan seluruh migrasi, dan bukan pula operasional harian
yang sedang berjalan.

Big bang tidak dapat dipertanggungjawabkan pada sistem yang menangani uang klaim dengan **tanpa
jejak audit atas perubahan nilai di sistem lama** (T-14): bila ada selisih setelah cutover, tidak
ada sumber untuk menelusurinya.

Jalan paralel penuh — kedua sistem menerima input yang sama — akan menggandakan beban kerja
pengguna operasional, dan `C-9`/`R-15` sudah mencatat bahwa **waktu pengguna bisnis untuk UAT
saja belum dialokasikan resmi**.

#### Konsekuensi

##### Positif

- Setiap gelombang memiliki titik kembali yang jelas.
- Risiko terdistribusi ke banyak rilis kecil, bukan menumpuk di satu tanggal.
- Tim belajar pada modul berisiko rendah sebelum menyentuh modul bernilai uang.

##### Negatif / utang teknis

- **Dua sistem harus dipelihara bersamaan** selama seluruh masa transisi, termasuk memperbaiki
  cacat di Pega yang sebenarnya akan segera ditinggalkan.
- **Masa paralel menambah pekerjaan yang tidak menghasilkan fitur**: sinkronisasi status, aturan
  penulis tunggal per tabel, dan perkakas uji kesetaraan `S-8`.
- Skema database tidak boleh berubah bebas selama masa paralel (`D-63`) — setiap perubahan
  menempuh tiga pihak dan memperlambat iterasi.
- Semakin lama masa paralel, semakin besar biayanya. Tanpa tanggal akhir yang mengikat, pola ini
  dapat berlangsung jauh lebih lama daripada rencana.

##### Risiko yang diterima secara sadar

- `D-61` menetapkan jadwal seluruh modul **tidak berubah** meski hasil analisis Fase 1
  menunjukkan 32 dari 33 modul belum berstatus siap penuh (§15). Selisih antara jadwal dan
  kesiapan itu **diterima sebagai tanggung jawab manajemen**, bukan diselesaikan di tingkat
  teknis.
- Pengguna bekerja di dua aplikasi sekaligus selama masa transisi, dengan tampilan yang mirip
  tetapi tidak identik.

#### Pertanyaan terbuka

- Apa tanggal akhir yang mengikat bagi masa paralel — kapan Pega dimatikan? Pemilik: Work Owner.
  Tanpa ini, biaya pemeliharaan ganda tidak berbatas.
- Bila sebuah gelombang gagal di gerbang 2, apakah gelombang berikutnya tetap berjalan sesuai
  jadwal `D-61`? Pemilik: Work Owner + manajemen.

### 0004 — Pakai satu database bersama selama masa paralel, dengan penulis tunggal per tabel

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-07 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-21`, `D-63`, `D-05`, `P-1`, `P-4` · `docs/verifikasi-bukti-adr.md` §14 |
| **Terkait** | ADR-0003, ADR-0005, ADR-0007, ADR-0012, ADR-0027, seluruh modul |

#### Konteks

Selama masa paralel (ADR-0003), Pega dan aplikasi Go melayani klaim yang sama pada hari yang
sama. Keduanya harus melihat data yang sama — status klaim, penugasan, nilai estimasi — tanpa
jeda sinkronisasi.

Data itu hari ini hidup di **tabel milik engine Pega**, dan sebagian dibaca sangat luas:

| Tabel Pega | Dibaca oleh | Isi |
|---|---|---|
| `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` | **116 rule** | header klaim |
| `DATAPEGA.PC_ASSIGN_WORKLIST` | 18 rule | penugasan per orang |
| `DATAPEGA.PC_ASSIGN_WORKBASKET` | 6 rule | antrean bersama |
| `DATAPEGA.PR_OPERATORS` | 6 rule | master pengguna |
| `DATAPEGA.PC_LINK_ATTACHMENT` | 3 rule | kaitan lampiran |

Angka 116 itu yang menentukan sifat keputusan ini: satu `ALTER` yang keliru pada tabel header
klaim menghentikan sistem yang sedang melayani produksi.

#### Opsi yang dipertimbangkan

1. **Satu database bersama**, dengan tabel baru milik Claim PNC menggantikan tabel engine Pega
   secara bertahap.
2. **Database terpisah + sinkronisasi dua arah** (CDC, replikasi, atau antrean pesan).
3. **Database terpisah + integrasi lewat API** antara Pega dan Go.

#### Keputusan

Kedua sistem memakai **satu database yang sama** selama masa paralel. Data yang hari ini hidup di
tabel engine Pega dipindahkan ke **tabel baru milik aplikasi Claim PNC** — tabel yang belum ada
dan dirancang di proyek ini.

Tiga aturan mengikat berlaku sepanjang masa paralel:

1. **Penulis tunggal per tabel (`P-1`).** Untuk setiap tabel, tepat satu sistem berwenang
   menulis. Sistem yang lain hanya membaca. Kewenangan berpindah saat modul pemiliknya lulus
   gerbang 2, bukan sebelum itu.
2. **Perubahan skema wajib backward-compatible (`P-4`).** Kolom dihapus dalam dua tahap, tidak
   pernah sekali jalan. Setiap tiket yang menyentuh skema memuat bagian rollback yang tidak
   kosong.
3. **Perubahan skema dijalankan DBA** atas permintaan tertulis tim pengembang dengan persetujuan
   Work Owner, dan **wajib diuji dengan menjalankan Pega dan Go bersamaan** terhadap skema hasil
   perubahan (`D-63`).

#### Rationale

Sinkronisasi dua arah antar database adalah sumber selisih data yang paling sulit ditelusuri, dan
sistem ini menangani uang klaim **tanpa jejak audit atas perubahan nilai di sistem lama** (T-14).
Selisih yang muncul tidak akan punya sumber pembanding.

Integrasi lewat API antara Pega dan Go menuntut perubahan besar di sisi Pega — tepatnya di
aplikasi yang sedang ditinggalkan, dikerjakan oleh tim yang sedang berpindah teknologi.

Database bersama memindahkan seluruh persoalan konsistensi ke satu tempat yang sudah memiliki
mekanisme untuk itu: transaksi database. Harganya adalah kopling, dan kopling itu dijinakkan
dengan aturan penulis tunggal.

#### Konsekuensi

##### Positif

- Tidak ada jeda sinkronisasi dan tidak ada konflik penggabungan data.
- Uji kesetaraan gerbang 1 (ADR-0027) dapat membandingkan hasil di atas **data yang benar-benar
  sama**, bukan dua salinan yang mungkin berbeda.
- Peralihan kewenangan per tabel menjadi penanda kemajuan migrasi yang konkret dan dapat diaudit.

##### Negatif / utang teknis

- **Kopling terkuat yang mungkin ada** antara sistem lama dan baru. Selama masa paralel, kedua
  aplikasi tersandera skema yang sama.
- **Iterasi melambat secara permanen selama masa paralel**: tiga pihak untuk setiap perubahan
  skema (`D-63`).
- Aplikasi Go tidak dapat merancang skemanya secara ideal; ia harus hidup berdampingan dengan
  tabel yang bentuknya ditentukan engine Pega.
- Aturan penulis tunggal **tidak ditegakkan mesin mana pun**. Ia hanya disiplin manusia, dan
  pelanggarannya baru terlihat sebagai data rusak.

##### Risiko yang diterima secara sadar

- Satu `ALTER` keliru pada `PC_ASM_FW_GCNMFW_WORK` menghentikan produksi yang dilayani 116 rule
  Pega. Mitigasinya adalah prosedur `D-63`, bukan mekanisme teknis.
- Tabel baru milik Claim PNC **belum ada sama sekali** dan harus dirancang dari nol, sementara
  bentuk tabel Pega hanya dapat dibaca dari cara rule memakainya — bukan dari dokumentasi.

#### Pertanyaan terbuka

- Bagaimana pelanggaran aturan penulis tunggal dideteksi — pemeriksaan berkala, trigger audit,
  atau tidak sama sekali? Pemilik: Lead Engineer + DBA. Tanpa jawaban, `P-1` hanya imbauan.
- Apakah aplikasi Go mendapat skema (schema) Oracle sendiri, atau menumpang skema yang ada?
  Pemilik: DBA + Work Owner. Menghalangi penulisan tiket `F-2`.

### 0005 — Tulis satu set SQL portabel: jalan di Oracle sekarang, PostgreSQL 17+ kemudian

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-07 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-01`, `D-20` (menyupersede `D-06`), `D-24`, `D-02` · 652 rule SQL pada export |
| **Terkait** | ADR-0004, ADR-0007, ADR-0009, modul `F-2` |

#### Konteks

`D-01` menetapkan dua hal yang tampak bertentangan: target akhir adalah **PostgreSQL untuk
seluruh data**, sementara **runtime sementara tetap Oracle 19c** dan tanggal cutover belum
ditentukan. Aplikasi harus berjalan di Oracle hari ini tanpa mengorbankan desain PostgreSQL-first.

`D-06` sempat mengangkat persoalan ini tetapi belum memilih mekanismenya; pertanyaan diajukan
ulang setelah seluruh **652 rule SQL** dianalisis, dan diputuskan sebagai `D-20`.

Volume yang terukur dari export menentukan bentuk keputusannya: **411 pemakaian `TO_CHAR`** dan
**68 pemakaian `ROWNUM`**.

#### Opsi yang dipertimbangkan

1. **Repository interface dengan dua implementasi SQL** berdampingan (Oracle dan PostgreSQL).
2. **Satu set SQL portabel**, dialek khusus hanya di sedikit tempat yang dikelola.
3. **Query builder / ORM** yang menangani dialek otomatis.
4. **Bangun untuk PostgreSQL saja**, Oracle diakses lewat lapisan kompatibilitas sementara.

#### Keputusan

**Satu set SQL portabel** yang berjalan di Oracle 19c dan PostgreSQL 17+, dengan **tiga
pengecualian yang dikelola secara sadar**:

1. **Pemformatan tanggal dan angka dikeluarkan dari SQL ke Go**, menghapus 411 pemakaian
   `TO_CHAR`.
2. **Generator nomor klaim** menjadi satu-satunya tempat dengan sakelar dialek eksplisit
   (ADR-0009).
3. **Paginasi diseragamkan ke `OFFSET … FETCH NEXT … ROWS ONLY`**, menggantikan 68 pemakaian
   `ROWNUM`. Pola ini sudah dipakai di 35 rule pada codebase yang ada.

**PostgreSQL 17 atau lebih baru adalah persyaratan teknis mengikat** (`D-24`), bukan preferensi.
Tanpa itu keputusan ini tidak dapat dijalankan.

Desain skema, tipe data, indexing, dan gaya SQL mengacu ke **PostgreSQL sebagai kanonikal**.

#### Rationale

Dua implementasi SQL berdampingan berarti setiap kueri ditulis dan diuji dua kali, oleh tim yang
sedang belajar Go. Biaya itu berlangsung sepanjang masa paralel yang belum bertanggal akhir.

ORM menyembunyikan SQL justru pada aplikasi yang inti kerumitannya **ada di SQL** — 652 rule,
sebagian dengan 97 predikat join. Menyembunyikannya memindahkan kerumitan, bukan menguranginya.

Mengeluarkan `TO_CHAR` ke Go bukan sekadar demi portabilitas: SQL yang ada **mengembalikan
tanggal sebagai string `'dd/mm/yyyy'`**, sehingga pengurutan dan penyaringan tanggal salah secara
diam-diam. Satu keputusan portabilitas sekaligus menutup cacat yang sudah berjalan.

#### Konsekuensi

##### Positif

- Satu kueri, satu tempat diuji, dua mesin database.
- Perpindahan ke PostgreSQL kelak menjadi peristiwa infrastruktur, bukan penulisan ulang aplikasi.
- Tanggal berpindah sebagai tipe tanggal, bukan string — memperbaiki pengurutan dan penyaringan
  yang selama ini salah.

##### Negatif / utang teknis

- **Fitur khas Oracle tidak boleh dipakai** meski tersedia dan kadang lebih cepat: `result_cache`,
  `JSON_OBJECT_T`, `DBMS_AQ`, hierarki `CONNECT BY`.
- Portabilitas hanya dapat dibuktikan bila **kedua mesin benar-benar diuji**. Tanpa lingkungan
  PostgreSQL sejak awal, klaim "portabel" tidak terverifikasi sampai cutover — dan saat itu sudah
  terlambat.
- Perubahan `ROWNUM` → `OFFSET … FETCH NEXT` mengubah rencana eksekusi pada beberapa kueri; ini
  **perubahan kinerja yang harus diukur**, bukan penggantian sintaks.
- `D-24` mengunci versi minimum PostgreSQL. Bila infrastruktur hanya menyediakan versi lebih
  rendah, seluruh ADR ini gugur.

##### Risiko yang diterima secara sadar

- Menulis SQL portabel di atas Oracle berarti sebagian optimasi yang wajar untuk Oracle
  ditinggalkan hari ini demi keuntungan yang baru datang saat cutover.
- Tanggal cutover belum ditentukan (`D-01`), sehingga masa "membayar biaya portabilitas tanpa
  menikmati hasilnya" tidak berbatas.

#### Pertanyaan terbuka

- Kapan lingkungan PostgreSQL 17+ tersedia untuk pengujian portabilitas? Pemilik: Tim Infra.
  Selama belum ada, tidak ada tiket yang boleh menyatakan SQL-nya "terbukti portabel".
- Apakah 97 predikat join pada kueri terberat tetap berkinerja wajar di PostgreSQL? Pemilik:
  DBA + Lead Engineer. Menghalangi kepastian NFR kinerja.

### 0006 — Bekukan data polis sebagai snapshot saat registrasi; kepemilikannya tetap di GISFW

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-07 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-04`, `D-03`, `D-34` · `JSON_POLIS`, `JSON_KLAIM` · 141 activity GISFW pada export |
| **Terkait** | CONTEXT.md#Snapshot-Polis, ADR-0008, modul `B-1`, `B-2` |

#### Konteks

Data polis dimiliki **GISFW**, aplikasi tim lain. Meski begitu, **141 activity GISFW ikut berada
di export ini** dan dipakai saat klaim berjalan — batas kepemilikannya kabur di tingkat kode.

`D-03` dan `D-34` sudah mempersempit lingkup migrasi: hanya ruleset `GCNMFW`; area
bengkel/sparepart/supplier dan ruleset GKM **dikecualikan**. Persoalan yang tersisa adalah data
polis, yang tidak bisa dikecualikan karena setiap klaim bertumpu padanya.

#### Opsi yang dipertimbangkan

1. **Simpan snapshot polis** saat klaim diregistrasi.
2. **Panggil API GISFW secara real-time** setiap kali data polis dibutuhkan.
3. **Ikut memigrasikan GISFW** dalam proyek ini.
4. Tunda — perlu koordinasi lebih dulu dengan tim GISFW.

#### Keputusan

Domain Klaim **menyimpan snapshot polis pada saat klaim diregistrasi** — melanjutkan pola
`JSON_POLIS`/`JSON_KLAIM` yang sudah berjalan, tetapi dengan **skema eksplisit**, bukan JSON
bebas bentuk.

**Kepemilikan data polis tetap di GISFW.** Claim PNC tidak pernah menulis ke data polis; ia hanya
membekukan salinannya pada satu titik waktu.

#### Rationale

Klaim adalah peristiwa yang dinilai menurut keadaan polis **pada saat kejadian dan registrasi** —
bukan menurut keadaan polis hari ini. Membaca polis secara real-time justru salah secara bisnis:
endorsemen yang terjadi setelah registrasi akan mengubah dasar penilaian klaim yang sudah
berjalan.

Snapshot juga membuat pemrosesan klaim **kebal terhadap ketersediaan sistem tim lain saat
runtime** — klaim tetap dapat diproses ketika GISFW sedang tidak dapat dihubungi.

Memigrasikan GISFW sekaligus akan melipatgandakan lingkup proyek dan menyeret tim lain ke dalam
jadwal yang bukan jadwal mereka.

#### Konsekuensi

##### Positif

- Nilai klaim tidak berubah diam-diam karena perubahan polis setelah registrasi.
- Ketergantungan runtime pada sistem tim lain hilang pada jalur pemrosesan klaim.
- Batas bounded context menjadi konkret dan dapat diuji: Claim PNC tidak punya jalur tulis ke
  data polis.

##### Negatif / utang teknis

- **Snapshot dapat usang.** Bila polis dikoreksi setelah registrasi — pembatalan, endorsemen,
  perbaikan data tertanggung — klaim tetap memakai salinan lama sampai ada yang menyegarkannya
  secara sadar. Aturan penyegaran itu **belum ada**.
- **Duplikasi data** antara GISFW dan Claim PNC, dengan seluruh biaya penyimpanan dan kebingungan
  "mana yang benar" yang menyertainya.
- Merancang skema eksplisit pengganti `JSON_POLIS` menuntut pemahaman penuh atas bentuk JSON yang
  ada sekarang — dan bentuk itu tidak terdokumentasi, hanya tersirat dari cara rule memakainya.
- 141 activity GISFW di export harus dipilah satu per satu: mana yang benar-benar milik tim lain
  dan mana yang sudah menjadi logika klaim.

##### Risiko yang diterima secara sadar

- Bila kelak GISFW menyediakan API polis yang andal, keputusan ini tetap dipertahankan — karena
  alasannya bisnis (membekukan dasar penilaian), bukan teknis (menghindari kopling).
- Ketidaksesuaian antara snapshot dan polis asli akan muncul dalam sengketa klaim, dan yang
  dipakai adalah snapshot.

#### Pertanyaan terbuka

- Kapan sebuah snapshot boleh atau harus disegarkan, dan siapa yang berwenang memicunya?
  Pemilik: Work Owner. Menghalangi penyelesaian tiket `B-1`.
- Field polis apa saja yang wajib masuk snapshot? Daftar ini hanya dapat disusun dengan membaca
  seluruh pemakaian `JSON_POLIS` di rule, dan belum dikerjakan. Pemilik: Lead Engineer.
- Apakah tim GISFW menyetujui bahwa Claim PNC menyimpan salinan data mereka? Pemilik: Work Owner
  → Tim GISFW.

---

## 6. Kelompok B — Data, Integrasi, dan Artefak

Keputusan tentang cara data ditulis, dibaca, dipertukarkan dengan sistem lain, dan diterbitkan sebagai dokumen.

| # | Judul | Status |
|---|---|---|
| 0007 | Naikkan logika stored procedure ke Go dan pindahkan kepemilikan transaksi ke aplikasi | Accepted |
| 0008 | Ganti seluruh akses DB Link dengan pemanggilan API ke sistem pemilik data | Accepted |
| 0009 | Terbitkan nomor klaim baru berformat `PNCN.YY.xxxx` dari sequence database | Accepted |
| 0010 | Simpan dokumen lewat satu jalur API storage internal; database hanya menyimpan metadata | Accepted |
| 0011 | Bangun pembuatan PDF, Excel, dan CSV di dalam aplikasi Go | Accepted |
| 0012 | Terapkan soft delete menyeluruh; tidak ada penghapusan fisik data bernilai bisnis | Accepted |
| 0013 | Tentukan pengganti pola hapus-lalu-sisip-ulang pada konversi klaim | Proposed |

### 0007 — Naikkan logika stored procedure ke Go dan pindahkan kepemilikan transaksi ke aplikasi

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-07 (`D-02`), 2026-09-14 (`D-68`) |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-02`, `D-68`, `D-01` · `Database/INSERT_PLADLA.prc:69,74,79,138,143,148,179,184,189,198` · `Database/UPDATEREAS.prc` · `Database/ADD_NEWMASTERVIRTUALACCOUNT.prc:18` |
| **Terkait** | ADR-0004, ADR-0005, ADR-0013, modul `B-4`, `B-5`, `B-9`, `B-10`, `B-12` |

#### Konteks

`D-01` dan `D-02` menetapkan **tidak ada pemanggilan stored procedure** dari aplikasi baru;
seluruh SQL ditulis di kode Go dan database menjadi penyimpanan murni.

Pembacaan sumber procedure membuktikan keputusan itu bukan sekadar soal gaya. Dari 12 procedure
yang dibaca, **10 melakukan `COMMIT` sendiri**:

| Procedure | `COMMIT` | Catatan |
|---|---|---|
| `Database/INSERT_PLADLA.prc` | **9×** (`:69`, `:74`, `:79`, `:138`, `:143`, `:148`, `:179`, `:184`, `:189`) | `ROLLBACK` satu-satunya ada di handler terluar `:198` — **terjadi setelah commit**, sehingga tidak memulihkan apa pun |
| `Database/UPDATEREAS.prc` | 4× | — |

Selama procedure itu dipanggil apa adanya, **`B-4` dan `B-9` tidak dapat dibuat atomik**:
penerbitan PLA/DLA yang menempuh sembilan `COMMIT` dapat berhenti di tengah dan meninggalkan
state setengah jalan yang tidak dapat dibatalkan.

Kontrak galatnya juga bermasalah. Galat disampaikan lewat string `ErrMsg`, dan pada enam
procedure **`ErrMsg` tidak di-set pada jalur sukses** — sehingga `NULL` berarti berhasil. Pada
`Database/ADD_NEWMASTERVIRTUALACCOUNT.prc:18`, kolom yang sama membawa **nomor virtual account
sekaligus pesan galat**.

Syarat untuk meninggalkan procedure adalah memastikan tidak ada sistem lain yang memanggilnya —
dijawab `D-68`: **Claim PNC satu-satunya pemanggil.**

#### Opsi yang dipertimbangkan

1. **Naikkan seluruh logika ke Go**, procedure ditinggalkan setelah modulnya pindah.
2. **Pertahankan procedure** dan panggil dari Go — perilaku identik, tetapi non-atomik ikut
   terwarisi.
3. **Tulis ulang procedure** agar tidak melakukan `COMMIT` sendiri, lalu tetap dipanggil dari Go.

#### Keputusan

Seluruh logika stored procedure **dinaikkan ke Go**. Database menjadi penyimpanan murni, dan
**kepemilikan transaksi berpindah sepenuhnya ke lapisan Go**.

Karena Claim PNC adalah satu-satunya pemanggil (`D-68`), procedure tersebut **boleh ditinggalkan**
setelah modul pemiliknya lulus gerbang — tidak perlu dipelihara demi sistem lain.

Kontrak galat berbasis string `ErrMsg` **tidak ikut dibawa**. Galat disampaikan sebagai galat
bahasa Go, bukan sebagai nilai kolom.

#### Rationale

Ini satu-satunya opsi yang menyelesaikan tiga persoalan sekaligus: ketergantungan pada dialek
Oracle (ADR-0005), ketiadaan atomisitas pada `B-4`/`B-9`, dan kontrak galat yang tidak dapat
dibedakan dari data.

Menulis ulang procedure agar tidak ber-`COMMIT` (opsi 3) memindahkan pekerjaan ke PL/SQL — bahasa
yang justru sedang ditinggalkan — dan tetap menyisakan logika bisnis di dua tempat.

#### Konsekuensi

##### Positif

- **`B-4` Spreading Reasuransi dan `B-9` PLA/DLA dapat dibuat atomik.** Penerbitan yang di sistem
  lama menempuh sembilan `COMMIT` kini dapat dibungkus satu transaksi.
- Logika bisnis berada di satu tempat, dapat diuji dengan uji otomatis biasa.
- Menghapus ketergantungan pada PL/SQL sejalan dengan target PostgreSQL (ADR-0005).

##### Negatif / utang teknis

- **Logika yang sudah teruji bertahun-tahun ditulis ulang.** Setiap procedure yang dinaikkan
  adalah kesempatan baru untuk salah, pada modul yang menghitung uang.
- **62 procedure yang sudah diterima memanggil 12 objek lain yang belum diserahkan**, terberat
  `UPDATE_LOG_KONVERSI` (**162 pemanggilan**), `GETNEWID` (42), `PKG_COUNTER_PRODUCTION` (25),
  `PROCESSQUEUEDIRECT` (10). Sumbernya tetap harus diminta untuk dibaca meski objeknya kelak
  ditinggalkan.
- Perubahan dari sembilan commit menjadi satu transaksi **mengubah perilaku saat gagal**: sistem
  lama meninggalkan sebagian data, sistem baru tidak meninggalkan apa pun. Kasus uji kesetaraan
  harus dirancang menyadari ini, atau ia akan melaporkan selisih palsu.
- Transaksi yang lebih panjang menahan kunci lebih lama. Dengan 200–300 pengguna (`D-10`) risiko
  ini kecil, tetapi tidak nol pada job massal.

##### Risiko yang diterima secara sadar

- `D-68` menyatakan Claim PNC satu-satunya pemanggil **tanpa verifikasi katalog**. Bila ternyata
  ada sistem lain yang memanggil, mematikan procedure akan merusaknya. Verifikasinya cukup satu
  kueri `ALL_DEPENDENCIES`, dan **belum dijalankan**.
- Cacat yang selama ini tersembunyi di balik `COMMIT` beruntun akan tampak sebagai kegagalan
  transaksi penuh — lebih benar, tetapi lebih terlihat oleh pengguna.

#### Pertanyaan terbuka

- Kapan kueri `ALL_DEPENDENCIES` dijalankan untuk membuktikan `D-68` sebelum procedure
  dinonaktifkan? Pemilik: DBA. Menghalangi langkah mematikan procedure, bukan penulisan ulangnya.
- Sumber 12 objek dependensi yang belum diserahkan — kapan tersedia? Pemilik: Tim Pega/DBA.
  Menghalangi penyelesaian tiket `B-5`, `B-9`, `B-10`, `B-12` (`R-01`).

### 0008 — Ganti seluruh akses DB Link dengan pemanggilan API ke sistem pemilik data

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-07 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-25`, `D-57`, `R-03` · 64 pemakaian DB Link pada export · `Service REST/` (4 layanan masuk) |
| **Terkait** | ADR-0006, ADR-0022, modul `S-4`, `B-7`, `B-12` |

#### Konteks

Sistem lama menjangkau enam database lain lewat **DB Link Oracle** — kopling yang tidak terlihat
dari kode aplikasi mana pun, hanya dari teks SQL:

| DB Link | Pemakaian | Data yang diambil |
|---|---|---|
| `@ASMD` | **55×** | `DATAMINING.GET_WORKING_HOURS` (17×), `HRDASM.V_HRD_MST`, `MST_DET_SALES`, `GENERAL.MST_BUKA_PROTEKSI`, `GL.T_ALL_PAYMENT`, `LST_USER_ASURANSI`, `GENERAL.LST_MITRA`, `TREATY_LOSS`, `MBU.T_CADANGAN_KLAIM_KREDIT`, fungsi `GET_NAMA_*` |
| `@SIMASNET` | 3× | `GENERAL.MST_BUKA_PROTEKSI`, `V_KLAIM` |
| `@SMI` | 2× | `GENERAL.MST_BUKA_PROTEKSI`, `V_KLAIM` |
| `@OPJAVA` | 2× | `NEW_GENERAL.M_USER`, `M_USER_JOB` |
| `@PROD_ASM` | 1× | `POOLDATA.AGENT` |
| `@PROD_TKA` | 1× | `ANEKA.MST_SHARE_PU` |

Konsekuensi bentuk ini: perubahan skema di database mana pun di atas dapat merusak Claim PNC
**tanpa peringatan**, karena tidak ada kontrak apa pun di antaranya.

#### Opsi yang dipertimbangkan

1. **Ganti seluruhnya dengan pemanggilan API** ke sistem pemilik data.
2. **Pertahankan DB Link** — paling murah, kopling tetap.
3. **Replikasi data** yang dibutuhkan ke database Claim PNC.

#### Keputusan

Seluruh akses lintas database lewat DB Link **diganti pemanggilan API** ke sistem pemilik data.
Kopling tersembunyi antar database dihapus dan diganti **kontrak yang eksplisit**.

#### Rationale

DB Link menyembunyikan ketergantungan justru pada titik yang paling mahal bila putus. Kontrak API
memaksa ketergantungan itu menjadi terlihat: ada pemilik, ada versi, ada perilaku saat gagal.

Replikasi (opsi 3) menukar satu masalah dengan masalah lain — keusangan data — dan menambah
komponen yang harus dipantau, bertentangan dengan `D-08`.

#### Konsekuensi

##### Positif

- Ketergantungan lintas sistem menjadi eksplisit, bernama, dan dapat diuji tiruannya.
- Perubahan skema di sistem lain tidak lagi merusak Claim PNC secara diam-diam.
- Sejalan dengan ADR-0005: DB Link adalah fitur Oracle yang tidak ada padanannya di PostgreSQL.

##### Negatif / utang teknis

- **API penggantinya kemungkinan besar belum ada** (`R-03`). Yang diganti bukan cara memanggil,
  melainkan kontrak yang harus dibangun tim lain lebih dulu.
- Panggilan jaringan menggantikan join database: lebih lambat, dan **memperkenalkan mode gagal
  yang sebelumnya tidak ada**. Setiap titik integrasi butuh timeout, percobaan ulang, dan
  perilaku saat sistem lawan mati.
- `DATAMINING.GET_WORKING_HOURS` dipanggil **17×** untuk perhitungan TAT (ADR-0020). Mengubahnya
  menjadi panggilan jaringan di dalam kalkulasi yang dipakai laporan massal berisiko pada kinerja
  — kemungkinan besar menuntut caching, yang membawa persoalan kebaruan datanya sendiri.
- Enam sistem berarti enam tim, enam jadwal, dan enam kesepakatan — seluruhnya di luar kendali
  proyek ini.

##### Risiko yang diterima secara sadar

- Bila satu API tidak pernah datang, modul yang bergantung padanya berhenti. Mitigasinya adalah
  kebijakan `D-37` untuk modul yang artefaknya tidak pernah tiba — bukan solusi teknis.
- Ketersediaan Claim PNC menjadi ikut bergantung pada ketersediaan sistem lain saat runtime, hal
  yang justru dihindari ADR-0006 untuk data polis.

#### Pertanyaan terbuka

- API pengganti untuk enam DB Link — sudah ada, atau harus dibangun tim pemiliknya? Pemilik: Work
  Owner → tim masing-masing sistem. Ini `R-03`, dan menghalangi tiket `S-4`.
- **Ditemukan setelah `D-25` diputuskan:** `Service REST/` memuat **empat layanan masuk** —
  `KomiteAcceptAdjustment`, `KomiteAcceptAdjustmentPA`,
  `RecivedDataandAttachmentLelangASMSimasbid`, `RequestCreateClaimCredit2`. Dua di antaranya
  **menerima persetujuan komite dari sistem lain**. Permukaan masuk ini belum pernah masuk
  hitungan `FR-S4` maupun `D-25`, dan lingkup `S-4` karenanya **lebih besar daripada yang
  disetujui**. Pemilik: Work Owner.

### 0009 — Terbitkan nomor klaim baru berformat `PNCN.YY.xxxx` dari sequence database

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-07 (`D-22`), format direvisi 2026-09-14 (`D-71`) |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-22`, `D-71` (menyupersede bagian format pada `D-22`), `D-20` · `POOLDATA.CLAIM_NO_NONPEGA_SEQ` · `docs/verifikasi-bukti-adr.md` §2 |
| **Terkait** | CONTEXT.md#Klaim, ADR-0004, ADR-0005, modul `B-2` |

#### Konteks

Identitas klaim di sistem lama adalah `pzInsKey` berformat
`ASM-FW-GCNMFW-WORK PNC-xxxx` — **kunci teknis Pega yang bocor menjadi identitas bisnis**. Format
itu memuat nama aplikasi, nama ruleset, dan nama kelas kerja Pega: seluruhnya hal yang akan
lenyap bersama Pega.

Nomor klaim bukan sekadar kunci internal. Ia muncul di surat ke tertanggung, di PLA/DLA ke
koasuransi dan reasuransi, serta di pelaporan. Ia **tidak dapat diubah surut**.

Selama masa paralel (ADR-0003), kedua sistem menerbitkan klaim baru ke database yang sama
(ADR-0004), sehingga nomor dari kedua sistem harus dapat hidup berdampingan dan dibedakan.

**Sequence-nya belum ada.** `POOLDATA.CLAIM_NO_NONPEGA_SEQ` **nol kemunculan di seluruh export** —
ia akan dibuat, bukan dipakai ulang.

#### Opsi yang dipertimbangkan

1. **Format baru berprefix `PNCN`** dari sequence khusus, prefix Pega ditinggalkan.
2. **Pertahankan format lama** agar tidak ada yang berubah bagi pengguna dan pihak luar.
3. **Penomoran ulang seluruh klaim lama** ke format baru saat migrasi data.

Setelah opsi 1 dipilih, bentuk segmennya masih terbuka: dengan atau tanpa unsur tahun. `D-71`
menutupnya.

#### Keputusan

Klaim yang dibuat sistem baru memakai format **`PNCN.YY.xxxx`** — tiga segmen dipisahkan
**titik**:

| Segmen | Isi | Sumber |
|---|---|---|
| `PNCN` | penanda tetap asal sistem baru | literal |
| `YY` | dua digit tahun | `TO_CHAR(SYSDATE,'RR')` |
| `xxxx` | nomor urut | `TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)` |

Sintaks Oracle yang ditetapkan, apa adanya:

```sql
'PNCN' || '.' || TO_CHAR(SYSDATE,'RR') || '.' || TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)
```

Prefix `ASM-FW-GCNMFW-WORK` **tidak dipakai lagi**.

Nomor klaim lama **dibiarkan apa adanya** — tidak dinomori ulang. Kedua format hidup berdampingan
secara permanen.

Generator nomor klaim adalah **satu-satunya tempat dengan sakelar dialek eksplisit** dalam
kebijakan SQL portabel (ADR-0005). Sejak `D-71`, sakelar itu membungkus **dua** perbedaan dialek
sekaligus — sequence dan pemformatan tahun — bukan satu.

> **`D-71` menyupersede `D-22` hanya pada bentuk nomornya** (`PNCN-xxxx` → `PNCN.YY.xxxx`). Sisa
> isi `D-22` — prefix Pega ditinggalkan, nomor lama tidak dinomori ulang — tetap berlaku.

#### Rationale

Membawa `ASM-FW-GCNMFW-WORK` ke sistem yang tidak lagi memakai Pega berarti mengabadikan nama
platform yang sudah tiada di dalam data bisnis selamanya.

Penomoran ulang klaim lama (opsi 3) mustahil dipertanggungjawabkan: nomor itu sudah tercetak di
surat, sudah dikirim ke reasuransi, dan sudah dipakai pihak luar untuk merujuk klaim yang sama.

Prefix `PNCN` membuat asal sebuah klaim dapat dibaca langsung dari nomornya — berguna justru
selama masa paralel, ketika dua sistem menerbitkan klaim bersamaan.

Segmen tahun membuat **usia sebuah klaim terbaca tanpa membuka datanya** — berguna pada berkas
fisik, surat, dan percakapan dengan pihak luar, dan itu praktik lazim pada penomoran dokumen
asuransi.

#### Konsekuensi

##### Positif

- Identitas bisnis lepas dari kunci teknis platform.
- Asal klaim (Pega atau Go) **dan tahun terbitnya** terbaca dari nomornya tanpa melihat data lain.
- Sequence database menjamin keunikan tanpa koordinasi antar instans aplikasi (`D-27`).

##### Negatif / utang teknis

- **Dua format nomor klaim hidup permanen.** Setiap pencarian, laporan, dan integrasi harus
  menerima keduanya. Ini bukan keadaan sementara — klaim lama tidak akan pernah berubah format.
- **Segmen terakhir tidak berlebar tetap.** `TO_CHAR(...NEXTVAL)` tanpa format mask tidak memberi
  angka nol di depan, sehingga nomor tumbuh `PNCN.26.9` → `PNCN.26.10` → `PNCN.26.1000`.
  Akibatnya **pengurutan sebagai teks tidak sesuai urutan penerbitan** — `.10` mendahului `.9`.
  Setiap layar dan laporan yang mengurutkan berdasarkan nomor klaim harus menyadarinya.
- **Tahun diambil dari `SYSDATE`**, yaitu tanggal server basis data — bukan tanggal kejadian dan
  bukan tanggal registrasi. Klaim yang diterbitkan di sekitar pergantian tahun mengambil tahun
  dari jam server, bertaut dengan `R-12` (pergeseran zona waktu) dan dengan penyesuaian 7 jam pada
  ADR-0017 butir 3.
- Pengurutan berdasarkan nomor klaim tidak bermakna kronologis lintas kedua format.
- Pihak luar — koasuransi, reasuransi, broker — akan menerima dua bentuk nomor dari perusahaan
  yang sama, dan sebagian sistem mereka mungkin memvalidasi formatnya.
- Titik sebagai pemisah lebih rawan daripada tanda hubung pada perkakas yang memperlakukan titik
  sebagai pemisah ekstensi berkas atau pemisah desimal — khususnya saat nomor klaim dipakai
  sebagai nama berkas atau ditempel ke lembar kerja.

##### Risiko yang diterima secara sadar

- Segmen `xxxx` bertambah lebar seiring waktu; tidak ada batas panjang yang ditetapkan, sehingga
  kolom penyimpan dan bidang tampilan harus disiapkan longgar sejak awal.
- Sequence adalah satu-satunya pengecualian dialek yang disengaja; ia harus diuji di dua mesin
  database, bukan satu.
- Sintaks work owner dipakai **apa adanya**. Catatan netral: pada `TO_CHAR`, format `'RR'`
  menghasilkan dua digit tahun yang identik dengan `'YY'` — perbedaan perilaku `RR` hanya berlaku
  saat menafsirkan masukan (`TO_DATE`), bukan saat mengeluarkan teks.

#### Pertanyaan terbuka

- **Apakah sequence direset setiap awal tahun?** Bila tidak, nomor urut terus bertambah melewati
  pergantian tahun (`PNCN.26.8125` → `PNCN.27.8126`) dan segmen tahun menjadi penanda, bukan
  penghitung per tahun. Bila ya, nomor mulai dari 1 tiap tahun. Keduanya sah; pilihannya mengubah
  cara nomor dibaca orang. Pemilik: Work Owner. Menghalangi penyelesaian tiket `B-2`.
- **Apakah lebar segmen terakhir perlu dibuat tetap** (mis. `TO_CHAR(seq.NEXTVAL,'FM0000')`) agar
  pengurutan teks benar? Pemilik: Work Owner. Sampai dijawab, sintaks yang ditetapkan dipakai apa
  adanya.
- Apakah pihak luar (reasuransi, broker, BPPDAN) perlu diberi tahu perubahan format sebelum klaim
  pertama terbit dari sistem baru? Pemilik: Work Owner.

### 0010 — Simpan dokumen lewat satu jalur API storage internal; database hanya menyimpan metadata

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-07 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-16` · `Activity/UploadDocumentToGoogleStorage-Act.xml:1849` · `DATAPEGA.PC_LINK_ATTACHMENT` · `docs/verifikasi-bukti-adr.md` §3, §14.2 |
| **Terkait** | ADR-0004, ADR-0026, modul `S-1` |

#### Konteks

Sistem lama menyimpan dokumen klaim lewat **tiga mekanisme berbeda** yang tumbuh berurutan, dan
ketiganya masih hidup bersamaan. Akibatnya tidak ada satu tempat pun yang menjawab "di mana
dokumen klaim ini berada" tanpa memeriksa ketiganya.

Satu koreksi penting atas analisis awal: `Activity/UploadDocumentToGoogleStorage-Act.xml` **bukan
mekanisme keempat**. Rule itu tidak memiliki satu pun `<pyMethod>` dan mendelegasikan seluruh
kerjanya lewat `Call InsertDokumenPNC` (`:1849`) — namanya menyesatkan, perilakunya tidak.
Hitungan tiga mekanisme pada `D-16` benar.

#### Opsi yang dipertimbangkan

1. **Satu jalur lewat API storage internal Sinarmas** yang sudah berjalan; database hanya
   menyimpan metadata.
2. **Simpan berkas di database** sebagai BLOB — satu tempat, satu transaksi.
3. **Simpan di filesystem bersama** milik aplikasi.
4. Pertahankan ketiga mekanisme yang ada.

#### Keputusan

Dokumen disimpan lewat **API storage internal Sinarmas yang sudah berjalan**. Database aplikasi
hanya menyimpan **metadata dan referensi**: ID gambar, URL, masa berlaku, dan kategori.

**Ketiga mekanisme yang ada disatukan menjadi satu jalur.** Tidak ada modul yang boleh menulis
dokumen dengan cara lain.

#### Rationale

Menyimpan berkas di database (opsi 2) memindahkan beban penyimpanan ke tempat yang paling mahal
untuk di-backup dan direplikasi, pada sistem yang datanya sudah puluhan juta baris (`D-10`).

Filesystem bersama (opsi 3) menuntut storage bersama antar dua instans aplikasi (`D-27`) —
komponen infrastruktur tambahan yang bertentangan dengan `D-08`.

API storage internal sudah ada, sudah dipakai, dan sudah punya pemilik. Memakainya menghapus
seluruh kelas persoalan penyimpanan berkas dari lingkup proyek ini.

#### Konsekuensi

##### Positif

- Satu jalur, satu tempat mencari, satu tempat memperbaiki.
- Database tetap ramping dan cepat di-backup.
- Aplikasi tetap stateless — prasyarat `D-27`.

##### Negatif / utang teknis

- **Penyimpanan dokumen menjadi ketergantungan runtime pada sistem lain.** Bila API storage mati,
  unggah dokumen berhenti meski seluruh aplikasi sehat.
- **Tidak ada atomisitas antara dokumen dan metadata.** Berkas tersimpan di sistem lain sementara
  barisnya di database, sehingga selalu mungkin terjadi berkas yatim (tersimpan tanpa metadata)
  atau metadata yatim (baris tanpa berkas). Mekanisme pembersihannya **harus dirancang** dan
  belum ada.
- **Dokumen lama tersebar di tiga mekanisme.** Menyatukan jalur tulis tidak menyatukan data yang
  sudah ada; pembacaan harus tetap menjangkau ketiganya sampai ada migrasi data dokumen —
  pekerjaan yang belum dijadwalkan.
- Soft delete (ADR-0012) tidak berlaku pada berkas di sistem lain. Menghapus metadata tidak
  menghapus berkasnya, dan itu keputusan tersendiri yang belum diambil.

##### Risiko yang diterima secara sadar

- Kontrak API storage internal dimiliki tim lain dan dapat berubah di luar kendali proyek ini.
- Masa berlaku URL yang disimpan sebagai metadata berarti referensi dokumen dapat kedaluwarsa;
  perilaku saat itu terjadi belum ditentukan.

#### Pertanyaan terbuka

- Bagaimana dokumen lama pada tiga mekanisme lama diperlakukan — dimigrasikan, atau dibaca di
  tempatnya selamanya? Pemilik: Work Owner. Menghalangi penyelesaian tiket `S-1`.
- Saat metadata dokumen dihapus secara soft delete, apakah berkas di storage ikut dihapus?
  Pemilik: Work Owner + Compliance.
- Berapa masa berlaku URL dokumen, dan apa yang terjadi setelah lewat? Pemilik: Tim pemilik API
  storage.

### 0011 — Bangun pembuatan PDF, Excel, dan CSV di dalam aplikasi Go

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-07 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-11`, `D-10` · 56 Report Definition · 78 activity laporan · `pyMaxRecords=500` pada 54 dari 56 laporan (T-12) |
| **Terkait** | ADR-0001, ADR-0002, modul `S-2`, `S-7`, `U-5` |

#### Konteks

Laporan adalah bagian besar aplikasi ini: **56 Report Definition** dan **78 activity** yang
melayaninya. Di sistem lama, pembuatan PDF dan Excel dikerjakan engine reporting Pega — komponen
yang lenyap bersama platformnya.

Dua temuan Fase 1 membentuk keputusan ini:

- **`pyMaxRecords=500` pada 54 dari 56 laporan.** Laporan yang ada hari ini dibatasi 500 baris,
  bukan karena kebutuhan bisnis melainkan karena batas klipboard Pega.
- **`OFFSET` nol kemunculan** di seluruh export (T-12). Tidak ada laporan yang benar-benar
  memaginasi hasil besar; yang ada adalah pemotongan pada 500 baris.

Artinya: kebutuhan export bervolume besar **belum pernah benar-benar dilayani** sistem lama, dan
membangunnya di sistem baru adalah penambahan kemampuan, bukan penyalinan.

#### Opsi yang dipertimbangkan

1. **Bangun sendiri di dalam aplikasi Go** memakai pustaka pembuat PDF/Excel.
2. **Pakai tools BI eksternal** (Metabase, Superset, atau sejenisnya).
3. **Pakai engine reporting berbayar** yang dipasang terpisah.

#### Keputusan

Pembuatan **PDF, Excel, dan CSV diimplementasikan sendiri di dalam aplikasi Go**. Tidak memakai
engine reporting Pega maupun tools BI eksternal.

Export bervolume besar ditangani dengan cara yang **tidak membebani transaksi** — dialirkan
(*streaming*), bukan disusun seluruhnya di memori lebih dulu.

#### Rationale

Tools BI eksternal menambah satu komponen yang harus dipasang, diamankan, dan diberi hak akses ke
database — bertentangan dengan `D-08` dan menambah permukaan yang harus dipelihara tim kecil.

Laporan di aplikasi ini bukan laporan analitis bebas bentuk; ia adalah **56 laporan dengan bentuk
tetap** yang sudah diketahui. Kebutuhan seperti itu dilayani kode biasa dengan baik, tanpa engine.

Membangun sendiri juga menjaga otorisasi tetap di satu tempat: laporan melewati pemeriksaan izin
yang sama dengan layar (ADR-0023), bukan jalur terpisah yang mudah terlupakan.

#### Konsekuensi

##### Positif

- Tidak ada komponen tambahan di produksi.
- Otorisasi laporan memakai mekanisme yang sama dengan seluruh aplikasi.
- Batas 500 baris dapat dihapus — laporan akhirnya dapat melayani permintaan yang sebenarnya.

##### Negatif / utang teknis

- **Tata letak PDF harus dibangun satu per satu.** Untuk 56 laporan, ini pekerjaan besar yang
  mudah diremehkan, dan seluruhnya harus cocok dengan keluaran lama agar lolos gerbang 1.
- **Permintaan perubahan laporan menjadi permintaan perubahan kode**, bukan konfigurasi. Pengguna
  bisnis kehilangan kemampuan mengubah laporan sendiri — bila selama ini mereka memilikinya.
- Export besar di satu instans memakan memori dan CPU yang sama dengan pelayanan transaksi
  (ADR-0001). Tanpa pembatasan, satu export dapat memperlambat seluruh pengguna.
- **Menghapus batas 500 baris adalah perubahan perilaku**, bukan pemeliharaan. Laporan yang dulu
  terpotong kini utuh — hasilnya berbeda, dan uji kesetaraan akan menandainya sebagai selisih.

##### Risiko yang diterima secara sadar

- Pustaka PDF di ekosistem Go tidak sematang engine reporting komersial; sebagian tata letak rumit
  mungkin menuntut kompromi visual.
- Volume sebenarnya yang diminta pengguna **tidak diketahui**, karena selama ini selalu terpotong
  di 500 baris. Rancangan kapasitas dibuat tanpa data historis yang sahih.

#### Pertanyaan terbuka

- Berapa baris maksimum yang wajib dilayani satu export? Pemilik: Work Owner. Tanpa angka ini,
  `S-2` tidak dapat dinyatakan selesai secara terukur.
- Apakah export besar dijalankan serentak dengan permintaan pengguna, atau diantrekan dan
  diberitahukan saat selesai? Pemilik: Work Owner + Lead Engineer.
- Apakah keluaran PDF wajib identik secara visual dengan keluaran Pega, atau cukup identik secara
  isi? Pemilik: Work Owner. Ini menentukan kriteria kelulusan gerbang 1 untuk `S-2`.

### 0012 — Terapkan soft delete menyeluruh; tidak ada penghapusan fisik data bernilai bisnis

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-13 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-66` (menyupersede `D-65`), `D-28`, `D-62` · `RDB List/UpdateLogServiceClaim-SQL.xml:27` · `RDB List/InsertClaimPNC-SQL.xml:77` · `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503` |
| **Terkait** | ADR-0013, ADR-0026, ADR-0027, modul `B-2`, `S-5`, `S-8` |

#### Konteks

Sistem lama menghapus baris secara fisik di banyak tempat — **termasuk pada tabel lognya sendiri**:

| Operasi | Objek | `berkas:baris` |
|---|---|---|
| `UPDATE` | `pooldata.claim_service_log` | `RDB List/UpdateLogServiceClaim-SQL.xml:27` |
| `DELETE` | `POOLDATA.JSON_KLAIM_LOG` | `RDB List/InsertClaimPNC-SQL.xml:77` |
| `UPDATE` | `T_LOGINCOAS` | `RDB List/BrowseOldEmailCoas-SQL.xml:69` |
| `UPDATE` | `pooldata.mst_login_surveyor` | `RDB List/UpdateMasterLoginSurvey-SQL.xml:9` |
| `RDB-DELETE` · `OBJ-DELETE` | berbagai | 12 step di 7 activity · 2 step di 2 activity |

Log yang dapat diubah dan dihapus bukan log. Ini bertabrakan langsung dengan `D-28`, yang
menetapkan jejak audit bersifat append-only.

`D-65` semula mencatat jawaban "sama dengan sistem lama" dan mengusulkan pemisahan tiga kelas
data sebagai jalan keluar. Work Owner **merevisi jawabannya** sebelum usulan itu disetujui, dan
revisi itulah yang menjadi `D-66`.

#### Opsi yang dipertimbangkan

1. **Soft delete di mana pun** — tidak ada `DELETE` fisik pada data bernilai bisnis.
2. **Hard delete untuk data tertentu** yang disebutkan Work Owner.
3. **Sama dengan sistem lama.**

#### Keputusan

**Soft delete berlaku menyeluruh.** Tidak ada `DELETE` fisik pada data bernilai bisnis di sistem
baru. Penghapusan dinyatakan lewat penanda — kolom flag beserta waktu dan pelakunya — bukan lewat
pembuangan baris.

Seluruh operasi `UPDATE`/`DELETE` pada tabel log di atas **tidak dibawa** ke sistem baru.

ADR ini menyupersede `D-65`. Pemisahan tiga kelas data yang diusulkan `D-65` **tidak diperlukan
dan tidak dipakai**.

#### Rationale

Dengan tidak adanya penghapusan fisik, jejak audit append-only (`D-28`) dan kebijakan penghapusan
berdiri di atas prinsip yang sama — pertentangan yang membuat `D-65` rumit hilang dengan
sendirinya.

Ini juga satu-satunya kebijakan yang konsisten dengan **T-14**: perubahan nilai uang klaim di
sistem lama **tidak punya jejak audit sama sekali**. Menambahkan jejak audit sambil tetap
mengizinkan penghapusan fisik akan menghasilkan jejak yang tetap bisa dihilangkan.

#### Konsekuensi

##### Positif

- Tidak ada data bernilai bisnis yang dapat hilang tanpa jejak.
- Jejak audit (ADR-0026) menjadi kontrol yang benar-benar mengikat — penting karena `D-59`
  menghapus pemisahan tugas, sehingga audit adalah satu-satunya kontrol pengimbang.
- Retensi data menjadi keputusan kebijakan (`D-62`), bukan efek samping operasi harian.

##### Negatif / utang teknis

- **Setiap kueri harus menyaring baris yang ditandai terhapus.** Satu kueri yang lupa
  menyaringnya akan menampilkan data yang seharusnya hilang — kelas cacat baru yang tidak ada
  di sistem lama.
- **Tabel tumbuh permanen.** Dengan data historis puluhan juta baris (`D-10`), indeks dan rencana
  eksekusi harus dirancang menyadari adanya baris mati.
- **Pola hapus-lalu-sisip-ulang tidak dapat dipertahankan.** `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503`
  menghapus **12 tabel** milik satu klaim lalu menyisipkan ulang seluruh pohonnya sebagai
  mekanisme idempotensi. Penggantinya belum diputuskan — ADR-0013.
- Keunikan kunci alami menjadi rumit: baris yang "terhapus" masih menempati nilai kuncinya.

##### Risiko yang diterima secara sadar

- **Uji kesetaraan tidak boleh membandingkan jumlah baris.** Soft delete tidak mengubah hasil yang
  dilihat pengguna, tetapi mengubah isi tabel. Perkakas `S-8` membandingkan **hasil kueri sesuai
  aturan bisnis**; perbandingan berbasis `COUNT(*)` akan selalu berbeda dan **bukan** indikasi
  cacat.
- Soft delete **tidak** ditambahkan ke daftar perbaikan sadar `D-49` (ADR-0017), karena ia
  mengubah cara penyimpanan, bukan hasil yang terlihat.
- Dua tabel pada blok konversi (`T_DLALIST`, `T_PLALIST`) **sudah berpotensi menduplikasi baris
  hari ini** — delete-nya dikomentari di `:497`, `:498` sementara insert-nya tetap aktif di
  `:1296`, `:1325`. Cacat itu sudah ada sebelum keputusan ini.

#### Pertanyaan terbuka

- Siapa yang berwenang melihat data yang sudah ditandai terhapus, dan lewat layar apa? Pemilik:
  Work Owner. Tanpa ini, soft delete hanya menyembunyikan data tanpa manfaat pemulihan.
- Apakah ada titik waktu ketika baris bertanda terhapus benar-benar dibuang (arsip atau purge),
  mengikuti `D-62`? Pemilik: Work Owner + Compliance.

### 0013 — Tentukan pengganti pola hapus-lalu-sisip-ulang pada konversi klaim

| | |
|---|---|
| **Status** | Proposed |
| **Tanggal keputusan** | — |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner + Lead Engineer |
| **Jejak bukti** | `D-66` · `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503`, `:497`, `:498`, `:1296`, `:1325` |
| **Terkait** | ADR-0012, ADR-0007, ADR-0027, modul `B-2` |

> **Belum diputuskan.** Berkas ini memuat konteks, opsi, dan konsekuensi masing-masing opsi —
> **tanpa bagian `Keputusan`**. Jangan dijadikan dasar implementasi.

#### Konteks

Sistem lama menjamin konversi klaim dapat diulang tanpa menggandakan data dengan cara yang paling
langsung: **menghapus dulu, lalu menyisipkan ulang**.
`Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503` menghapus **12 tabel** milik satu klaim,
kemudian menyisipkan ulang seluruh pohon datanya.

Pola itu **bergantung pada penghapusan fisik**, sementara ADR-0012 menetapkan tidak ada
penghapusan fisik pada data bernilai bisnis. Keduanya tidak dapat berlaku bersamaan.

Ada cacat yang sudah berjalan hari ini dan harus ikut ditutup oleh apa pun penggantinya: pada dua
tabel — `T_DLALIST` dan `T_PLALIST` — **baris `DELETE`-nya sudah dikomentari** (`:497`, `:498`)
sementara `INSERT`-nya tetap aktif (`:1296`, `:1325`). Artinya konversi ulang pada kedua tabel itu
**sudah berpotensi menduplikasi baris sekarang**, sebelum perubahan apa pun dilakukan.

#### Opsi yang dipertimbangkan

**Opsi 1 — *Upsert* berbasis kunci alami.** Setiap baris dikenali dari kunci bisnisnya (misalnya
nomor klaim + objek + coverage + urutan); konversi ulang memperbarui baris yang sudah ada dan
menyisipkan yang belum.

- Menjaga jumlah baris tetap sama dengan sistem lama, sehingga uji kesetaraan lebih mudah dibaca.
- Menuntut **kunci alami yang benar-benar unik pada ke-12 tabel** — dan keunikan itu belum
  diverifikasi. Bila ada tabel tanpa kunci alami, opsi ini tidak dapat dipakai di sana.
- Tidak menyimpan riwayat: hasil konversi sebelumnya tertimpa.

**Opsi 2 — Versioning dengan penanda baris aktif.** Konversi ulang menyisipkan generasi baru dan
menandai generasi sebelumnya tidak aktif.

- Konsisten penuh dengan ADR-0012 dan ADR-0026: tidak ada yang hilang, semua perubahan terjejak.
- **Menggandakan pertumbuhan 12 tabel** setiap kali konversi diulang, di atas data historis
  puluhan juta baris (`D-10`).
- Setiap kueri pembaca ke-12 tabel harus menyaring baris aktif — 12 tabel × seluruh pembacanya.

**Opsi 3 — Konversi sekali saja, pengulangan dilarang.** Klaim yang sudah dikonversi tidak boleh
dikonversi ulang; koreksi ditempuh lewat jalur perbaikan data biasa.

- Paling sederhana dan paling murah.
- Menghapus kemampuan yang ada sekarang. Bila konversi ulang ternyata dipakai operasional untuk
  memperbaiki klaim bermasalah, opsi ini memutus jalan itu — dan **seberapa sering pengulangan
  benar-benar terjadi belum diketahui**.

#### Konsekuensi bila dibiarkan tidak diputuskan

- **Tiket `B-2` tidak dapat ditulis lengkap.** Kriteria penerimaan untuk konversi ulang tidak
  dapat dirumuskan tanpa mengetahui pola penggantinya.
- Cacat duplikasi pada `T_DLALIST` dan `T_PLALIST` tetap terbuka, baik di sistem lama maupun baru.
- Uji kesetaraan `S-8` untuk `B-2` tidak dapat dirancang, karena perilaku yang dibandingkan belum
  ditentukan.

#### Pertanyaan terbuka

1. **Seberapa sering konversi ulang benar-benar dijalankan di produksi, dan untuk keperluan apa?**
   Pemilik: Work Owner. Jawaban ini menentukan apakah Opsi 3 layak dipertimbangkan sama sekali.
2. **Apakah ke-12 tabel punya kunci alami yang unik?** Pemilik: DBA + Lead Engineer. Menentukan
   apakah Opsi 1 dapat dipakai seragam atau hanya sebagian.
3. **Apakah riwayat hasil konversi sebelumnya perlu disimpan?** Pemilik: Work Owner + Compliance.
   Bila ya, hanya Opsi 2 yang memenuhi.
4. Duplikasi `T_DLALIST`/`T_PLALIST` yang sudah terjadi hari ini — apakah datanya perlu
   dibersihkan sebelum migrasi? Pemilik: Work Owner + DBA.

---

## 7. Kelompok C — Aturan Bisnis Bernilai Uang dan Kewenangan

Keputusan yang langsung menentukan berapa nilai sebuah klaim, siapa yang berwenang menyetujuinya, dan bagaimana pekerjaan dibagikan. Kelompok paling sensitif dalam dokumen ini.

| # | Judul | Status |
|---|---|---|
| 0014 | Hitung jenjang komite secara kumulatif menurut ambang bawah, dengan pita nilai hanya di Non-MBU | Accepted |
| 0015 | Konversi mata uang memakai kurs tanggal kejadian; tolak klaim bila kurs tidak ditemukan | Accepted |
| 0016 | Simpan nilai uang presisi penuh; validasi total spreading dengan toleransi empat desimal | Accepted |
| 0017 | Perbaiki sembilan cacat aturan uang; replikasi satu yang perilakunya memang benar | Accepted |
| 0018 | Pertahankan empat konsep status, masing-masing dengan nama yang tidak dapat tertukar | Accepted |
| 0019 | Pertahankan Worklist dan Workbasket; bangun ulang router sebagai aturan routing | Accepted |
| 0020 | Pertahankan dua basis perhitungan TAT untuk keperluan yang berbeda | Accepted |

### 0014 — Hitung jenjang komite secara kumulatif menurut ambang bawah, dengan pita nilai hanya di Non-MBU

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-11 (`D-47`, `D-52`), 2026-09-14 (`D-70`) |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-47`, `D-52`, `D-70`, `D-14` · `Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459`, `:16501` · `When/IsKomiteLoop-When.xml` · `RDB List/EmailKomiteBerjenjang_sql-SQL.xml:52` · `Database/emailkomite.csv` |
| **Terkait** | CONTEXT.md#Jenjang-Kumulatif, CONTEXT.md#Pita-Nilai-Komite, ADR-0017, ADR-0027, modul `B-7` |

#### Konteks

Komite menyetujui nilai klaim secara berjenjang. Cara jumlah jenjang ditentukan **tidak
terdokumentasi di mana pun** dan hanya dapat dibaca dari kode.

Mekanismenya, terverifikasi:

| Bukti | Isi |
|---|---|
| `Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459` | `childPageKomite.KomiteLoop := TempRDBSearchEmailKomite.pxResultCount` |
| `When/IsKomiteLoop-When.xml` | `.AcceptStatus = "1"` **dan** `.KomiteCount <= .KomiteLoop` |

**Jumlah jenjang persetujuan sama dengan jumlah baris yang dikembalikan kueri.** Dan kueri itu
menyaring **hanya dengan ambang bawah**:

```sql
... AND LIMIT_BOTTOM <= {tempAdj.ConvertAdjustmentValue}
```

`LIMIT_BOTTOM` muncul di **11 SQL rule**; `LIMIT_TOP` muncul di **0**. Batas atas tercatat di
master tetapi tidak pernah dipakai menyaring — itulah sebabnya hasilnya banyak baris, bukan satu.

Dari 17 kueri yang membaca `EMAILKOMITE`, hanya **4** yang memfilter `TYPE_KOMITE`. Jalur utama
**PA** (`EmailKomiteBerjenjangPA_sql-SQL.xml:5`) dan **Travel**
(`EmailKomiteBerjenjangTravel_sql-SQL.xml:103`) tidak memfilternya — dan isi master menjelaskan
mengapa:

| Lini | Ambang bawah per jenjang | `TYPE_KOMITE` |
|---|---|---|
| **NONMBU** | `0` · `50.000.001` | `1` · `1` |
| **NONMBU** | `100.000.001` · `500.000.001` · `1.000.000.001` | `2` · `2` · `2` |
| **PA** | `0` · `10.000.001` · `50.000.001` · `100.000.001` | **`2` · `1` · `1` · `2`** |
| **TRAVEL** | `0` · `50.000.001` · `100.000.001` | `1` · `1` · `1` |

Pada Non-MBU, `TYPE_KOMITE` memang memisahkan pita nilai. Pada PA ia berselang-seling menaiki
tangga — di sana ia membedakan **jalur PA reguler dari PA TKI**
(`EmailKomiteBerjenjangPATKI_sql-SQL.xml:82` mematok `type_komite='2'`), bukan pita nilai.

#### Opsi yang dipertimbangkan

1. **Rentang tertutup** `LIMIT_BOTTOM <= nilai <= LIMIT_TOP` — satu jenjang per klaim.
2. Hanya batas bawah, dengan aturan "DEGREE tertinggi yang memenuhi menang".
3. **Kumulatif** — seluruh jenjang yang memenuhi harus menyetujui, `LIMIT_TOP` menjadi validasi
   integritas master.
4. Untuk filter pita: berlaku seragam di semua lini, atau per lini sesuai kuerinya hari ini.

#### Keputusan

**Model kumulatif dipertahankan.** Jumlah jenjang persetujuan = jumlah baris `EMAILKOMITE` yang
memenuhi `LIMIT_BOTTOM <= nilai klaim` (ditambah filter `STS_ADJ`, `STS_AKTIF`, `TYPE_BUSINESS`),
diurutkan `DEGREE`.

**`LIMIT_TOP` dipakai sebagai validasi integritas master data**, bukan untuk memilih baris:
sistem baru menolak master yang rentangnya tumpang tindih atau berlubang antar `DEGREE` dalam satu
`TYPE_BUSINESS` + `TYPE_KOMITE`.

**Filter pita nilai (`TYPE_KOMITE`) berlaku khusus lini Non-MBU** (`D-70`). Di sana pita dipilih
lebih dulu — ≤ Rp 100.000.000 → `1`, di atasnya → `2` — lalu akumulasi berjalan di dalam pita itu
saja. **Di lini lain tidak ada langkah pendahuluan**; akumulasi berjalan atas seluruh jenjang lini
tersebut, persis seperti kuerinya hari ini.

`D-70` **membatasi cakupan `D-52`**, tidak membatalkannya: isi `D-52` tetap benar untuk Non-MBU.

#### Rationale

Menerapkan rentang tertutup akan mengembalikan tepat satu baris → `KomiteLoop = 1` → **hanya satu
jenjang persetujuan berapa pun nilai klaim**, menghapus penjenjangan yang menjadi inti `D-14` dan
`BRD §11.4`.

Menyeragamkan filter pita ke semua lini terbukti lebih buruk lagi. Dihitung dari master:

| Kasus | Perilaku sistem lama | Bila pita disaring seragam |
|---|---|---|
| PA Rp 5.000.000 | 1 penyetuju | **0 penyetuju** — klaim mandek |
| PA Rp 75.000.000 | 3 penyetuju | 2 penyetuju |
| PA Rp 150.000.000 | 4 penyetuju | 2 penyetuju |
| Travel Rp 150.000.000 | 3 penyetuju | **0 penyetuju** — klaim mandek |
| Non-MBU seluruh nilai | — | tidak berubah |

Dua kasus menghasilkan klaim tanpa penyetuju sama sekali. Itu perubahan perilaku, bukan
perbaikan, dan melanggar `P-5`.

#### Konsekuensi

##### Positif

- Perilaku sistem lama dipertahankan persis di setiap lini, sehingga gerbang 1 `B-7` dapat
  dijalankan tanpa pengecualian.
- Penjenjangan menjadi terdokumentasi untuk pertama kalinya.
- `LIMIT_TOP` yang selama ini mati mendapat kegunaan nyata: menjaga master tetap konsisten saat
  diisi.

##### Negatif / utang teknis

- **Satu kolom memikul dua arti berbeda.** `TYPE_KOMITE` berarti pita nilai di Non-MBU dan varian
  jalur di PA. Pemisahannya menjadi dua kolom dipertimbangkan dan **tidak diambil**, sehingga arti
  gandanya wajib didokumentasikan di model data baru.
- **Validasi `LIMIT_TOP` hanya dapat dijalankan per lini**, tidak lintas lini — karena tangga PA
  memang tidak tersusun menurut pita.
- Aturan penjenjangan berbeda antarlini membuat `B-7` tidak dapat ditulis sebagai satu model
  tunggal; ia adalah beberapa model yang berbagi mekanisme.
- Jumlah penyetuju bergantung penuh pada isi master. Satu baris master yang salah mengubah
  kewenangan persetujuan tanpa ada perubahan kode.

##### Risiko yang diterima secara sadar

- Kerangka pertanyaan atas mekanisme ini **pernah salah dua kali** selama analisis (Q12 dan Q32).
  Ia mudah disalahpahami, dan setiap perubahan di masa depan berisiko mengulang kesalahan yang
  sama.
- Dua kueri jalur Simasnet memakai `dbms_random.value` dalam pemilihan komite; jalur utama
  memakai `ORDER BY DEGREE` dan deterministik. Ketidakseragaman ini dibawa apa adanya.

#### Pertanyaan terbuka

- **Belum terverifikasi:** tiga kueri yang memfilter `TYPE_KOMITE` secara dinamis —
  `EmailKomiteBerjenjang_sql` (`Activity/SetEmailKomite-Act.xml`), `EmailKomiteAdjuster_sql`
  (`Activity/SetEmailKomiteAdjuster-Act.xml`), `EmailKomiteSalvage_sql`
  (`Activity/SetEmailKomiteSalvage-Act.xml`) — menerima nilainya dari pemanggil lewat
  `tempAdj.pyMemo` dan `tempAdj.AcceptedNo`. **Lini apa saja yang benar-benar melewati ketiga
  kueri itu belum ditelusuri sampai ke sumber nilainya.** Pemilik: Lead Engineer. Ini bagian
  Definition of Ready tiket `B-7` dan `B-12`.
- Apakah `TYPE_KOMITE` kelak dipecah menjadi dua kolom saat master diisi ulang di `F-4`?
  Pemilik: Work Owner.

### 0015 — Konversi mata uang memakai kurs tanggal kejadian; tolak klaim bila kurs tidak ditemukan

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-11 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-48`, `R-19` · `Database/GETCURRENCYSTANDARD.fnc:3`, `:14`, `:20-22` · `m_currencystandard` |
| **Terkait** | CONTEXT.md#Kurs-Standar, ADR-0014, ADR-0017, ADR-0027, modul `B-2`, `B-5` |

#### Konteks

Nilai klaim dalam valuta asing dikonversi ke Rupiah sebelum dibandingkan dengan ambang komite
(ADR-0014) dan sebelum masuk laporan. Basis kurs karenanya menentukan **berapa banyak orang yang
harus menyetujui sebuah klaim** — bukan sekadar angka tampilan.

Fungsi konversi yang ada, `Database/GETCURRENCYSTANDARD.fnc`, memakai kurs **hari eksekusi**, dan
pada jalur tertentu (`:20-22`) mengembalikan `1` ketika kurs tidak ditemukan. Nilai `1` berarti
satu satuan valuta asing dihitung setara satu Rupiah: klaim bernilai besar menyusut menjadi
kecil, lalu **lolos tanpa komite**.

Kegagalan itu senyap. Tidak ada galat, tidak ada catatan, dan hasilnya tampak seperti angka wajar.

#### Opsi yang dipertimbangkan

1. **Kurs pada tanggal kejadian**, tolak bila tidak ditemukan.
2. **Kurs hari eksekusi** — sama dengan sistem lama.
3. Kurs tanggal kejadian, dengan **nilai bawaan** bila kurs tidak ada.

#### Keputusan

Konversi ke IDR memakai **kurs yang berlaku pada tanggal kerugian**, bukan kurs hari eksekusi.

Bila kurs untuk mata uang dan tanggal itu **tidak ditemukan**, transaksi **ditolak dengan galat
eksplisit** yang menyebutkan mata uang dan tanggalnya. **Tidak ada nilai bawaan, dan tidak ada
kurs pengganti.**

#### Rationale

Klaim dinilai menurut keadaan pada saat kejadian — sama seperti polis dibekukan pada saat
registrasi (ADR-0006). Memakai kurs hari eksekusi membuat nilai Rupiah sebuah klaim **berubah
hanya karena prosesnya terlambat**, dan dengan itu mengubah jenjang komite yang harus
menyetujuinya.

Nilai bawaan `1` adalah bentuk terburuk dari kegagalan senyap: ia mengubah klaim besar menjadi
kecil tepat pada titik yang menentukan kewenangan persetujuan. Menolak secara eksplisit menukar
kesalahan yang tak terlihat dengan gangguan yang terlihat — dan gangguan yang terlihat dapat
diperbaiki.

#### Konsekuensi

##### Positif

- Nilai Rupiah sebuah klaim menjadi stabil: tidak berubah karena keterlambatan proses.
- Jenjang komite menjadi konsisten dan dapat diulang hasilnya.
- Kegagalan data kurs menjadi terlihat, bukan tersembunyi di balik angka yang tampak wajar.

##### Negatif / utang teknis

- **Uji kesetaraan akan menampilkan selisih pada seluruh data historis valuta asing**, karena
  laporan lama memakai kurs hari eksekusi. Selisih ini **wajib dinyatakan lebih dulu sebagai
  perbaikan yang direncanakan** (ADR-0017), bukan ditemukan sebagai kejutan saat pengujian.
- **Klaim valuta asing yang dulu lolos kini dapat ditolak** sampai kursnya dilengkapi. Ini
  perbaikan yang diinginkan, tetapi berdampak operasional dan perlu disiapkan sebelum gerbang 1
  dijalankan.
- Master kurs menjadi data kritis: kelengkapannya per tanggal kini menentukan apakah klaim dapat
  diproses sama sekali. Proses pengisiannya — siapa, kapan, dari sumber apa — **belum ada**.
- Klaim dengan tanggal kejadian jauh di masa lalu menuntut kurs historis yang mungkin tidak
  pernah tercatat.

##### Risiko yang diterima secara sadar

- Penolakan pada saat kurs tidak tersedia dapat menghentikan pemrosesan klaim di jam kerja, dan
  pemulihannya bergantung pada pihak yang mengisi master kurs.
- Perubahan basis kurs mengubah nilai historis dalam laporan; angka lama dan baru untuk klaim yang
  sama tidak akan cocok.

#### Pertanyaan terbuka

- **Isi tabel `m_currencystandard` belum ada di repo** — masih diminta ke DBA (`R-19`). Tanpa itu
  `B-5` tidak dapat diuji. Pemilik: DBA.
- Siapa yang mengisi kurs harian dan dari sumber apa (Bank Indonesia, kurs korporat, atau lain)?
  Pemilik: Work Owner. Menghalangi penyelesaian tiket `F-4`.
- Bagaimana klaim yang tertolak karena kurs kosong diperlakukan — ditahan dan diproses ulang
  otomatis, atau dikembalikan ke petugas? Pemilik: Work Owner.

### 0016 — Simpan nilai uang presisi penuh; validasi total spreading dengan toleransi empat desimal

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-11 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-51`, `D-49` · `Activity/InputRegister_act-Act.xml:13183` |
| **Terkait** | CONTEXT.md#Spreading, ADR-0014, ADR-0017, modul `B-4`, `B-5` |

#### Konteks

Spreading reasuransi membagi risiko sebuah klaim menjadi beberapa bagian yang totalnya harus
**100%**. Sistem lama memvalidasinya begini:

```
@contains(local.totalspreading, 100.0) || local.totalspreading == 100 || @contains(local.totalspreading, 99.99)
```
`Activity/InputRegister_act-Act.xml:13183`

Ini **pencocokan substring, bukan perbandingan numerik** (T-3). Akibatnya total `199.99` dan
`1100.0` ikut lolos — keduanya memuat potongan teks yang dicari — sementara maksud aturannya
adalah "mendekati seratus". Tidak ada pembulatan sama sekali dalam perhitungannya.

Persoalan kedua: sampai berapa desimal nilai uang disimpan, dan kapan dibulatkan. Pembulatan per
langkah perhitungan menumpuk selisih; pembulatan saat menyimpan menghilangkan informasi secara
permanen.

#### Opsi yang dipertimbangkan

1. Bulatkan saat menyimpan, sehingga angka tersimpan sama dengan angka tampil.
2. Bulatkan di setiap langkah perhitungan.
3. Pertahankan perilaku lama apa adanya.
4. **Simpan presisi penuh, bulatkan hanya saat ditampilkan.**

#### Keputusan

**Nilai uang disimpan presisi penuh.** Pembulatan dilakukan **hanya saat ditampilkan** — tidak
saat menyimpan, dan tidak per langkah perhitungan. Perbandingan terhadap ambang (termasuk ambang
komite, ADR-0014) memakai **nilai presisi penuh**.

**Total spreading reasuransi** divalidasi dengan pembulatan 4 desimal dan toleransi:

```sql
ROUND(SUM(share), 4) BETWEEN 99.9999 AND 100.0001
```

Ini menggantikan pencocokan substring di `InputRegister_act-Act.xml:13183` — butir pertama pada
daftar perbaikan `D-49` (ADR-0017).

Kriteria penerimaan `B-4` yang langsung dapat dipakai: `100.0000` diterima · `99.9999` diterima
(batas bawah) · `100.0001` diterima (batas atas) · `199.99` **ditolak** · `1100.0` **ditolak**.

#### Rationale

Toleransi dibutuhkan karena share reasuransi memang kerap tidak berjumlah tepat 100% setelah
pembagian — itu kenyataan bisnis, bukan cacat. Yang salah pada sistem lama bukan adanya toleransi,
melainkan **cara toleransi itu diperiksa**.

Presisi penuh saat menyimpan menjaga agar pembulatan tidak menumpuk sepanjang rantai
`Estimasi → Usulan → Akseptasi → Dibayar`. Membulatkan saat menyimpan berarti setiap tahap
mewarisi selisih tahap sebelumnya.

#### Konsekuensi

##### Positif

- `199.99` dan `1100.0` tidak lagi lolos validasi. Kelas cacat yang sudah berjalan tertutup.
- Nilai yang tersimpan dapat direkonsiliasi dengan sumbernya tanpa selisih pembulatan.
- Ambang komite dibandingkan terhadap angka yang sama persis setiap kali — jenjang persetujuan
  menjadi dapat diulang hasilnya.

##### Negatif / utang teknis

- **Angka tersimpan tidak selalu sama dengan angka yang dilihat pengguna.** Setiap laporan,
  export, dan layar harus konsisten menerapkan aturan pembulatan tampilan — dan satu tempat yang
  lupa akan menampilkan angka berbeda untuk data yang sama.
- **Uji kesetaraan akan menemukan selisih pada data yang sebelumnya lolos**: klaim historis dengan
  total spreading `199.99` valid menurut sistem lama dan tidak valid menurut sistem baru. Selisih
  itu harus dinyatakan sebagai perbaikan yang direncanakan.
- Tipe data desimal presisi penuh menuntut kedisiplinan di seluruh lapisan — database, Go, JSON
  API, dan JavaScript. **Angka pecahan di JavaScript tidak presisi**, sehingga nilai uang tidak
  boleh melewati batas itu sebagai angka.

##### Risiko yang diterima secara sadar

- Toleransi `99,9999`–`100,0001` adalah batas yang dipilih; klaim dengan total di luar batas itu
  akan ditolak meski secara bisnis mungkin masih wajar.
- Pengguna yang terbiasa melihat angka bulat akan menemukan selisih tampilan pada beberapa
  laporan, dan itu akan dilaporkan sebagai "bug" sebelum dipahami sebagai perbaikan.

#### Pertanyaan terbuka

- Berapa desimal yang dipakai pada tampilan untuk nilai Rupiah dan untuk persentase share?
  Pemilik: Work Owner. Menghalangi penyelesaian tiket `U-2`.
- Berapa presisi yang disepakati untuk kolom uang di skema baru? Pemilik: Lead Engineer + DBA.

### 0017 — Perbaiki sembilan cacat aturan uang; replikasi satu yang perilakunya memang benar

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-11 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-49`, `P-5` · lihat tabel di bawah untuk `berkas:baris` tiap butir |
| **Terkait** | ADR-0015, ADR-0016, ADR-0027, modul `B-2`…`B-10` |

#### Konteks

`P-5` menetapkan prinsip **kesetaraan perilaku lebih dulu**: sistem baru meniru sistem lama,
termasuk keanehannya, kecuali untuk perbaikan yang **diputuskan secara eksplisit**. Tanpa daftar
perbaikan yang disepakati di muka, setiap selisih pada uji kesetaraan menjadi perdebatan tentang
apakah ia cacat atau perbaikan.

Analisis Fase 1 menemukan **sepuluh cacat pada aturan yang menghitung atau memvalidasi uang**.
Masing-masing harus diputuskan sebelum gerbang 1 dijalankan — bukan sesudahnya.

#### Opsi yang dipertimbangkan

1. Perbaiki seluruhnya.
2. Replikasi seluruhnya demi kesetaraan sempurna.
3. **Putuskan satu per satu**, dan daftarnya mengikat uji kesetaraan.

#### Keputusan

**Sembilan cacat diperbaiki; satu direplikasi** karena perilakunya memang benar.

| # | Cacat | Bukti | Keputusan |
|---|---|---|---|
| 1 | Toleransi spreading berupa pencocokan substring; `199.99` dan `1100.0` lolos | `Activity/InputRegister_act-Act.xml:13183` | **PERBAIKI** (ADR-0016) |
| 2 | Ambang Rp 50 juta memakai **tiga operator berbeda** di tiga rule | `SetEmailKomite:1445` · `SetEmailKomiteAdjuster:968` · `SetEmailKomiteSalvage:888` | **PERBAIKI** — satu operator |
| 3 | Penyesuaian 7 jam diterapkan **asimetris di dalam satu kondisi validasi** | `InputRegister_act:5788` + `:4805` | **PERBAIKI** |
| 4 | Kurs mengabaikan tanggal kerugian | `GETCURRENCYSTANDARD.fnc:3` vs `:14` | **PERBAIKI** (ADR-0015) |
| 5 | Kurs tidak ditemukan → `RETURN 1` | `GETCURRENCYSTANDARD.fnc:22` | **PERBAIKI** (ADR-0015) |
| 6 | `LIMIT_TOP` diabaikan + tie-breaker acak | `EmailKomiteBerjenjangSimasnet_sql:98-99` | **PERBAIKI sebagian** (ADR-0014) — model kumulatif dipertahankan, `LIMIT_TOP` menjadi validasi master; tie-breaker acak hanya ada di 2 kueri Simasnet |
| **7** | Tanggal PLA/DLA dari pengguna dibuang, diganti `SYSDATE` | `INSERT_PLADLA.prc:67`, `:136`, `:177` | **REPLIKASI** — perilaku sekarang sudah benar |
| 8 | `NilaiSalvage` hanya ditambahkan bila baris terakhir kebetulan bertipe salvage | `ValidasiSisaTSI:817` vs `:1489` | **PERBAIKI** — salvage selalu ditambahkan |
| 9 | `INSERT_SALVAGE` menulis `IDSALVAGE = NULL` saat update | `INSERT_SALVAGE.prc:47` | **PERBAIKI** + perlu hitungan DBA atas baris rusak |
| 10 | `GETSELISIHJAM` gagal → `RETURN 0` jam, tak terbedakan dari nol | `GETSELISIHJAM.fnc:22` | **PERBAIKI** |

Butir 7 direplikasi karena tanggal PLA/DLA memang **tanggal sistem menerbitkan dokumen**, bukan
tanggal yang dipilih pengguna. Parameter `TTGLPLADLA` yang diterima lalu dibuang **bukan cacat**;
sistem baru boleh menghapus parameter itu seluruhnya.

#### Rationale

Memperbaiki seluruhnya tanpa memeriksa satu per satu akan mengubah butir 7 menjadi "perbaikan"
yang justru merusak: mengizinkan pengguna memilih tanggal dokumen resmi yang dikirim ke reasuransi.

Mereplikasi seluruhnya berarti membawa `RETURN 1` pada kegagalan kurs ke sistem baru — cacat yang
membuat klaim besar lolos tanpa komite.

Jalan tengahnya adalah satu daftar yang diputuskan di muka dan mengikat pengujian.

#### Konsekuensi

##### Positif

- Setiap selisih pada gerbang 1 punya satu dari dua jawaban yang jelas: ia salah satu dari 13
  butir perbaikan, atau ia bug.
- Cacat yang paling berbahaya — kurs `RETURN 1`, spreading substring, `IDSALVAGE = NULL` —
  ditutup sebelum data baru bertambah di atasnya.
- Butir 7 menjadi contoh konkret bagi tiket `B-9` bahwa tidak semua keanehan adalah cacat.

##### Negatif / utang teknis

- **Daftar perbaikan eksplisit `P-5` bertambah dari 4 menjadi 13 butir.** Setiap butir menambah
  satu kelas selisih yang harus dijelaskan pada setiap pengujian, untuk setiap modul terdampak.
- **Data historis tetap mengandung akibat cacat itu.** Memperbaiki aturannya tidak memperbaiki
  baris yang sudah telanjur salah — khususnya butir 9, yang menuntut hitungan DBA atas baris
  rusak dan kemungkinan perbaikan data.
- Uji kesetaraan menjadi lebih rumit dibaca: hasil "berbeda" tidak lagi otomatis berarti gagal.

##### Risiko yang diterima secara sadar

- Klaim yang dulu lolos kini dapat tertolak (butir 1, 4, 5). Ini disadari dan diterima sebagai
  perbaikan, dengan dampak operasional yang harus disiapkan.
- Butir 6 hanya diperbaiki sebagian: tie-breaker acak pada dua kueri Simasnet tetap ada sampai
  ada keputusan tersendiri.

#### Pertanyaan terbuka

- **Berapa banyak baris yang rusak akibat butir 9** (`IDSALVAGE = NULL`), dan apakah datanya
  diperbaiki sebelum migrasi? Pemilik: DBA + Work Owner. Menghalangi penyelesaian tiket `B-12`.
- Apakah data historis yang terdampak butir 1 (total spreading di luar toleransi) dibiarkan apa
  adanya? Pemilik: Work Owner.

### 0018 — Pertahankan empat konsep status, masing-masing dengan nama yang tidak dapat tertukar

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-07 (`D-18`), ditegaskan ulang 2026-09-14 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-18`, `D-19` · `docs/verifikasi-bukti-adr.md` §4 (T-5, T-6) · master status: 33 kode `1134`–`1166` |
| **Terkait** | CONTEXT.md#Status, ADR-0006, seluruh modul bisnis |

#### Konteks

Sistem lama membawa **empat penanda status berbeda** pada klaim yang sama:

| Sistem lama | Isi |
|---|---|
| `StatusWork` | posisi klaim dalam alur kerja |
| `StatusClaim` | status bisnis klaim — **33 kode**, `1134`–`1166` |
| `ClaimStatus` | penanda biner `0`/`1` |
| `StatusPosisi` | posisi pada rangkaian tahapan progres |

Nama-nama itu **hampir tidak dapat dibedakan satu sama lain**: `StatusClaim` dan `ClaimStatus`
berbeda hanya pada urutan kata. Kesalahan membaca keempatnya adalah sumber kekeliruan yang mudah
terjadi dan sulit terdeteksi.

Dua temuan Fase 1 sempat meragukan keempatnya:

- **T-5** — `ClaimStatus` ternyata milik ruleset **GISFW**, bukan Claim PNC.
- **T-6** — `StatusPosisi` bukan properti dan hanya pernah muncul dengan satu nilai
  (`'On Progress'`) pada 21 titik panggil di export.

#### Opsi yang dipertimbangkan

1. **Pertahankan keempatnya**, beri nama yang jelas berbeda.
2. Gabungkan menjadi lebih sedikit konsep, karena dua di antaranya tidak terbukti punya domain
   nilai yang kaya.
3. Rancang ulang model status dari kebutuhan bisnis hari ini.

#### Keputusan

**Keempat konsep dipertahankan** — dikonfirmasi ulang oleh pemilik bisnis: *"4 status tersebut
memang berbeda"*. Di sistem baru masing-masing diberi nama yang **tidak lagi bisa tertukar**:

| Nama baru | Sistem lama | Isi |
|---|---|---|
| **Status Proses** | `StatusWork` | sedang berjalan, selesai, atau ditolak |
| **Status Klaim** | `StatusClaim` | status bisnis, 33 kode berlabel di master |
| **Flag Klaim** | `ClaimStatus` | penanda biner |
| **Status Posisi Progres** | `StatusPosisi` | posisi pada rangkaian tahapan progres |

Domain **Status Klaim** dikoreksi menjadi rentang penuh `1134`–`1166` — **33 kode**, bukan
`1142`–`1151` seperti yang dipakai dokumen-dokumen awal. Sebelas kode pertama (`1134`–`1144`)
membawa penomoran lama `01`–`11`.

#### Rationale

Menggabungkan konsep status adalah perubahan makna bisnis, bukan penyederhanaan teknis — dan
pemilik bisnis menyatakan keempatnya memang berbeda. Menggabungkannya akan menghilangkan
perbedaan yang dipakai orang dalam bekerja.

Yang benar-benar menjadi sumber kesalahan bukan jumlah konsepnya, melainkan **namanya**. Itulah
yang diperbaiki.

#### Konsekuensi

##### Positif

- Kekeliruan membaca `StatusClaim` vs `ClaimStatus` tidak mungkin lagi terjadi, karena namanya
  tidak lagi mirip.
- Domain 33 kode terdokumentasi untuk pertama kalinya; sebelumnya hanya sebagian yang diketahui.
- Model status tetap sesuai cara pengguna bekerja hari ini, sejalan dengan `D-13`.

##### Negatif / utang teknis

- **Empat konsep status pada satu entitas tetap rumit**, apa pun namanya. Setiap layar dan laporan
  harus jelas menyatakan status mana yang ditampilkannya.
- **`ClaimStatus` milik bounded context tim lain** (T-5). Salah satu dari "empat konsep" sebenarnya
  berada di luar batas kepemilikan Claim PNC, dan hubungannya dengan ADR-0006 belum dirumuskan.
- **`StatusPosisi` tidak terbukti punya lebih dari satu nilai** di export (T-6). Bila di produksi
  ia memang hanya bernilai `'On Progress'`, sistem baru akan membawa kolom yang tidak pernah
  berubah.
- Pemetaan 33 kode lama ke penamaan baru harus lengkap dan tidak boleh meleset satu pun; kode yang
  tidak terpetakan akan membuat klaim historis tidak terbaca.

##### Risiko yang diterima secara sadar

- Membawa empat konsep berarti membawa kerumitannya ke sistem baru selama masa hidup aplikasi,
  bukan hanya masa migrasi.
- Bukti untuk dua dari empat konsep lebih lemah daripada dua lainnya, dan keputusan tetap diambil
  atas dasar pernyataan pemilik bisnis.

#### Pertanyaan terbuka

- **`SELECT DISTINCT STATUSPOSISI`** pada data produksi — masih ada di daftar permintaan ke DBA,
  untuk menyelesaikan T-6 secara empiris. Pemilik: DBA. Tidak menghalangi tiket mana pun, tetapi
  menentukan apakah kolom itu perlu dibawa.
- Bagaimana `Flag Klaim` yang dimiliki GISFW diperlakukan — dibaca dari snapshot (ADR-0006), atau
  dibaca langsung? Pemilik: Work Owner + Tim GISFW.
- Apa makna persis `0` dan `1` pada Flag Klaim? Pemilik: Work Owner. Belum terjawab sejak `D-18`.

### 0019 — Pertahankan Worklist dan Workbasket; bangun ulang router sebagai aturan routing

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-07 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-26`, `R-04`, `D-13` · `Flow/Register_Flow.xml` · `RDB List/BrowsePICRandomTeam-SQL.xml:39-40` · `AddTJobCounterPIC_SQL` |
| **Terkait** | CONTEXT.md#Worklist, CONTEXT.md#Workbasket, ADR-0021, ADR-0023, modul `B-6` |

#### Konteks

Penugasan di sistem lama memakai mekanisme Pega: tabel `PC_ASSIGN_WORKLIST` (18 rule) dan
`PC_ASSIGN_WORKBASKET` (6 rule). Keduanya lenyap bersama platformnya, sehingga sistem baru harus
punya model penugasannya sendiri.

Pembagian dua model itu **nyata dipakai**, bukan warisan kosong — terbaca dari
`Flow/Register_Flow.xml`:

| Model | Tahap |
|---|---|
| **Workbasket** (`impl=WorkBasket`, router `ToWorkbasket`) | `RCL/PUCL`, `Investigator`, `Compliance` |
| **Worklist** (`impl=WorkList`) | `Input Register`, `View Polis`, `Estimation`, `Input Estimasi`, `Choose Surveyor`, `Send To Analis`, `Send To PIC Teknik`, `RCLDokter`, `Analyst Doctor` |

Algoritma penugasannya juga sudah dapat direkonstruksi: `RDB List/BrowsePICRandomTeam-SQL.xml:39-40`
memilih petugas dengan **`ORDER BY counter_quota ASC`** — yang paling sedikit bebannya — lalu
`AddTJobCounterPIC_SQL` menaikkan pencacahnya.

#### Opsi yang dipertimbangkan

1. **Pertahankan kedua konsep** — Worklist (per orang) dan Workbasket (antrean bersama).
2. Sederhanakan menjadi satu model penugasan.
3. Rancang ulang total sesuai kebutuhan bisnis sekarang.

#### Keputusan

Model penugasan mempertahankan **dua konsep**:

- **Worklist** — tugas ditugaskan ke satu orang tertentu.
- **Workbasket** — antrean bersama, diambil siapa pun yang berwenang.

Delapan router Pega dibangun ulang sebagai **aturan routing milik aplikasi**: `PNCAdminRouter`,
`PNCTeknikRouter`, `RouterRCLDokter`, `KomiteRouter`, `PNCAdminRouterRCV`, `ToCurrentOperator`,
`ToWorkList`, `ToWorkbasket`.

#### Rationale

`D-13` menetapkan alur kerja tetap sama agar pengguna tidak perlu dilatih ulang. Mengubah model
penugasan mengubah cara orang bekerja setiap hari — perubahan paling terasa yang bisa dilakukan
tanpa mengubah satu pun aturan bisnis.

Pembagian dua model juga bukan kebetulan: tahap yang butuh kesinambungan penanganan memakai
Worklist, tahap yang dikerjakan sebuah tim memakai Workbasket. Pembagian itu mencerminkan cara
kerja, bukan keterbatasan platform.

#### Konsekuensi

##### Positif

- Cara kerja pengguna tidak berubah saat modulnya berpindah.
- Algoritma pembagian beban terbaca dari sumber, bukan ditebak: yang paling sedikit bebannya
  mendapat tugas berikutnya.
- Tabel penugasan milik aplikasi menggantikan tabel engine Pega secara bersih (ADR-0004).

##### Negatif / utang teknis

- **Tiga router tidak ada di export**: `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter`
  (`R-04`). Logikanya harus digali ulang — bukan disalin.
- Pencacah beban (`counter_quota`) adalah **state yang harus dijaga konsisten**; bila naik tanpa
  turun, atau gagal naik karena galat, pembagian beban menjadi timpang secara permanen.
- Penugasan menjadi tabel milik aplikasi yang harus dikunci dengan benar saat dua orang mengambil
  tugas yang sama dari satu Workbasket — persoalan konkurensi yang di sistem lama ditangani engine.
- **Hanya ada 4 workbasket di seluruh export** (T-7), sementara penjenjangan komite justru
  menugaskan ke **operator bernama**, bukan ke workbasket. Model penugasan `B-7` karenanya tidak
  seluruhnya mengikuti pola ini.

##### Risiko yang diterima secara sadar

- Membawa model Pega berarti membawa keterbatasannya, termasuk ketiadaan mekanisme "ambil kembali"
  tugas yang sudah ditugaskan ke seseorang yang kemudian tidak masuk kerja.
- Penugasan ke operator bernama (T-7) membuat ketidakhadiran seseorang dapat menghentikan klaim —
  hal yang tampaknya dijawab kolom `STS_ABS` pada master komite, dan perilakunya belum dirumuskan.

#### Pertanyaan terbuka

- Logika tiga router yang hilang (`R-04`) — digali dari Tim Pega, atau ditetapkan ulang oleh Work
  Owner sebagai aturan baru? Pemilik: Work Owner. Menghalangi penyelesaian tiket `B-6`.
- Apa yang terjadi pada tugas milik seseorang yang sedang tidak hadir (`STS_ABS`)? Pemilik: Work
  Owner.
- Apakah pencacah beban direset berkala? Pemilik: Work Owner. Tanpa jawaban, perilaku jangka
  panjang pembagian beban tidak dapat ditiru.

### 0020 — Pertahankan dua basis perhitungan TAT untuk keperluan yang berbeda

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-11 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-50`, `D-49` butir 10, `D-25` · `DATAMINING.GET_WORKING_HOURS@ASMD` (18 pemakaian di 7 berkas) · `GETSELISIHJAM.fnc:22` · `HRD_LBR` |
| **Terkait** | CONTEXT.md#TAT, ADR-0008, ADR-0017, modul `S-2`, `S-7`, `B-8` |

#### Konteks

TAT — lama penyelesaian klaim — dihitung dengan **dua fungsi berbeda** yang tidak akan pernah
menghasilkan angka sama:

| Fungsi | Keluaran | Dipakai di |
|---|---|---|
| `DATAMINING.GET_WORKING_HOURS@ASMD` | jam kerja, memperhitungkan kalender libur | seluruh KPI dan laporan admin, serta penulisan kronologi TAT |
| `GETSELISIHJAM` | selisih jam mentah, dibagi 24 menjadi hari | **satu layar saja** — inbox Compliance |
| `HRD_LBR` | pemeriksaan hari libur | dipakai terpisah |

Dugaan awal bahwa ini adalah ketidakkonsistenan yang harus diseragamkan ternyata **menyesatkan**:
benar secara aritmetika, tetapi keduanya **tidak pernah dipakai mengukur hal yang sama**. Batas
pemakaiannya tidak tumpang tindih.

#### Opsi yang dipertimbangkan

1. Seragamkan ke satu fungsi — `GET_WORKING_HOURS` untuk semua.
2. Seragamkan ke `GETSELISIHJAM` karena lebih sederhana.
3. **Pertahankan keduanya** dengan batas pemakaian yang sudah terbukti.

#### Keputusan

Kedua basis **dipertahankan**, dengan batas pemakaian yang sudah terbukti dari sumber:

- **`GET_WORKING_HOURS`** → seluruh **KPI dan laporan admin**, ditambah **penulisan kronologi
  TAT**. Ini jalur yang angkanya dilaporkan ke manajemen.
- **`GETSELISIHJAM`** → **satu layar saja**, inbox Compliance, dengan keluaran **hari**.
- **`HRD_LBR`** → pemeriksaan hari libur, dipakai terpisah.

Karena `GET_WORKING_HOURS` dan `HRD_LBR` adalah objek **remote lewat DB Link**, keduanya masuk
lingkup ADR-0008 — dan logikanya **ditulis ulang di Go**, bukan sekadar dipanggil lewat API:
perhitungan jam kerja dan kalender libur adalah **aturan bisnis**, bukan pengambilan data.

Butir 10 `D-49` tetap berlaku: `GETSELISIHJAM.fnc:22` yang mengembalikan `0` saat galat —
tak terbedakan dari nol hari — **diperbaiki** (ADR-0017).

#### Rationale

Menyeragamkan akan mengubah angka pada salah satu dari dua jalur. Bila yang berubah adalah jalur
KPI, angka yang dilaporkan ke manajemen berubah tanpa ada perubahan kinerja nyata. Bila yang
berubah adalah inbox Compliance, satuan yang dilihat petugas berubah dari hari menjadi jam kerja.

Keduanya adalah perubahan perilaku yang tidak diminta siapa pun.

Menulis ulang logika jam kerja di Go — alih-alih memanggilnya lewat API — diperlukan karena
fungsi itu dipanggil di dalam kalkulasi laporan massal. Menjadikannya panggilan jaringan per baris
akan menghancurkan kinerja laporan.

#### Konsekuensi

##### Positif

- Tidak ada angka TAT yang berubah tanpa diminta.
- Kalender libur dan jam kerja menjadi aturan bisnis milik aplikasi, dapat diuji tanpa
  ketergantungan pada database lain.
- Menghapus 18 pemanggilan lintas DB Link dari jalur laporan.

##### Negatif / utang teknis

- **Dua basis perhitungan hidup permanen**, dan perbedaannya akan terus menimbulkan pertanyaan
  "mengapa angkanya berbeda" dari pengguna yang membandingkan dua layar.
- **Kalender libur harus dimiliki dan dipelihara aplikasi ini.** Sumbernya hari ini adalah
  `HRD_LBR` milik sistem HRD; menulis ulang logikanya berarti ikut memutuskan dari mana daftar
  hari libur diperoleh setiap tahun — dan itu **belum ditetapkan**.
- Menulis ulang perhitungan jam kerja berisiko menghasilkan selisih dengan implementasi lama pada
  kasus batas (hari libur di tengah, lembur, akhir pekan berturut-turut) — dan kasus batas itulah
  yang paling sering muncul pada klaim bermasalah.

##### Risiko yang diterima secara sadar

- Angka TAT adalah angka yang dilaporkan ke manajemen. Selisih sekecil apa pun setelah migrasi
  akan terlihat dan harus dapat dijelaskan.
- Definisi jam kerja (mulai, selesai, istirahat) tidak terdokumentasi; ia hanya ada di dalam
  fungsi remote yang sumbernya belum dibaca.

#### Pertanyaan terbuka

- Sumber `DATAMINING.GET_WORKING_HOURS@ASMD` — kapan tersedia untuk dibaca? Tanpa itu aturan jam
  kerjanya tidak dapat ditulis ulang. Pemilik: DBA (`R-03`). Menghalangi tiket `S-7`.
- Dari mana daftar hari libur diperoleh setiap tahun di sistem baru, dan siapa yang mengisinya?
  Pemilik: Work Owner. Menghalangi penyelesaian tiket `F-4` dan `F-5`.

---

## 8. Kelompok D — Mekanisme Pega yang Harus Diganti

Dua mekanisme platform yang tidak punya padanan otomatis di luar Pega, dan keduanya belum dapat diputuskan.

| # | Judul | Status |
|---|---|---|
| 0021 | Bangun ulang transisi lateral (Ticket rule) sebagai perpindahan tahap yang eksplisit | Proposed |
| 0022 | Tentukan cara menjalankan job terjadwal di aplikasi yang hidup di dua instans | Proposed |

### 0021 — Bangun ulang transisi lateral (Ticket rule) sebagai perpindahan tahap yang eksplisit

| | |
|---|---|
| **Status** | Proposed |
| **Tanggal keputusan** | — |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Tim Pega (artefak) → Work Owner (perilaku) |
| **Jejak bukti** | `FR-W2` (`docs/BRD.md:518`) · `docs/verifikasi-bukti-adr.md` §5 · lihat tabel untuk `berkas:baris` |
| **Terkait** | CONTEXT.md#Lompatan-Lateral, ADR-0019, modul `B-7`, `B-11`, `B-13`, `B-14` |

> **Belum diputuskan.** Berkas ini memuat konteks, opsi, dan apa yang terhalang — **tanpa bagian
> `Keputusan`**. Jangan dijadikan dasar implementasi.

> **Peringatan istilah.** "Ticket rule" di sini adalah mekanisme Pega. Kata **"tiket"** di seluruh
> dokumen proyek ini berarti **tiket pekerjaan** di `docs/ticketing/`.

#### Konteks

Alur klaim tidak selalu berjalan maju. Pada keadaan tertentu klaim **melompat** ke tahap yang
bukan tahap berikutnya: kembali dari Komite ke petugas estimasi begitu seluruh anggota komite
menyetujui, atau berpindah ke PUCL saat ditolak pada Group Panel tertentu.

Di Pega mekanisme ini bernama **Ticket rule**: sebuah nama tujuan dipasang pada satu tahap, lalu
dipicu dari tempat lain.

Empat flow merujuk **17 nama Ticket rule**; folder `Ticket/` hanya berisi **8 rule**.

**Delapan yang ada:** `AcceptanceKomite` · `Akp_PNCInputProtection` · `SendToEstAdmin` ·
`SendToEstTravel` · `SendToEstimatorPA` · `SendToPICTravel` · `SendtoAnalysator` ·
`TC_PNCInputProtection`

**Sembilan yang dirujuk tanpa rule:**

| Ticket rule | Dirujuk dari | Klasifikasi |
|---|---|---|
| `SendToReceiveDocument` | `Flow/InputReceiveDocument.xml:690` | minta ke Tim Pega |
| `komiteAccept_ticket` | `Flow/Komite_Flow.xml:796` | minta ke Tim Pega |
| `KomiteAssign_ticket` | `Flow/Komite_Flow.xml:974` | minta ke Tim Pega |
| `RCLDokter` | `Flow/Register_Flow.xml:2954` | minta ke Tim Pega |
| `setToRegister_ticket` | `Flow/Register_Flow.xml:3052` | minta ke Tim Pega |
| `SendtoPUCL` | `Flow/Register_Flow.xml:3197` | minta ke Tim Pega |
| `SendToInvestigator` | `Flow/Register_Flow.xml:3624` | minta ke Tim Pega |
| `CompliancePNC` | `Flow/Register_Flow.xml:3844` | minta ke Tim Pega |
| `Status-Resolved` | `Flow/Register_Flow.xml:3004` | **bawaan Pega** — dipicu activity OOTB `Work-.Resolve` (`Activity/Resolve-Act.xml:1331`, nama ticket `:1343`); tidak perlu diminta |

Jadi yang benar-benar hilang dan harus digali ulang: **8 rule custom**.

**Masalah kedua lebih berat daripada yang pertama.** Dari 17 nama itu, **hanya tiga transisi yang
punya pemicu** di seluruh 902 activity + 29 flow action:

| Ticket rule | Dipicu dari | Kondisi |
|---|---|---|
| `SendToEstimatorPA` | `Activity/KomitePost_Adjustment-Act.xml:15083` | precondition `IsPA` (`:15033`); deskripsi *"Pindahin ke inputor apabila semua komite sudah aksep"* (`:14993`) |
| `SendToEstimatorPA` | `Activity/SetListComiteeClaimPerObjAdj-Act.xml:28380` | precondition `IsPA` (`:28426`); *"Back to estimator"* |
| `SendtoPUCL` | `Activity/KomitePost_Reject-Act.xml:3172` | `pyWorkCover.Policy.Quotation.GroupPanel=="002"` (`:3121`) |

**Empat belas nama sisanya tidak punya pemicu apa pun di export.** Metode pemicu yang dicari dan
jumlah temuannya: `Obj-Set-Tickets` **1** (hanya `Activity/Resolve-Act.xml:1331`, dan Pega
menandainya *deprecated* di `:153`) · `call SetTicket` **4** · `<Ticket>` sebagai parameter **4** ·
`<SetTicketNames>` **1** · rujukan ticket di `Flow Action/` **0**.

Artinya: meski Tim Pega mengirim 8 rule yang hilang, **kapan** masing-masing dipicu tetap tidak
diketahui.

**Koreksi atas `FR-W2`.** Angka "11 ticket" di `docs/BRD.md:518` benar untuk `Register_Flow` saja
— 12 shape ticket dikurangi `Status-Resolved` yang bawaan. Angka untuk seluruh aplikasi:
**17 dirujuk, 8 ada, 9 tanpa rule.**

#### Opsi yang dipertimbangkan

**Opsi 1 — Minta artefak lengkap ke Tim Pega lebih dulu**, baru rancang penggantinya.
Paling akurat; bergantung penuh pada pihak luar dan pada apakah pemicunya memang ada di ruleset
yang tidak ikut export.

**Opsi 2 — Gali ulang perilakunya bersama pengguna bisnis**, lalu tetapkan aturan perpindahan
tahap sebagai aturan baru yang eksplisit.
Tidak bergantung pada Tim Pega, tetapi **melepas jaminan kesetaraan**: perilaku hasil galian belum
tentu sama dengan yang berjalan di produksi, dan gerbang 1 kehilangan pembandingnya.

**Opsi 3 — Bawa hanya tiga transisi yang terbukti punya pemicu**, sisanya dianggap tidak aktif
sampai ada bukti sebaliknya.
Paling jujur terhadap bukti, tetapi berisiko menghapus jalur yang ternyata dipakai — dan
kegagalannya baru terlihat sebagai klaim yang mandek di produksi.

#### Konsekuensi bila dibiarkan tidak diputuskan

- Tiket `B-7`, `B-11`, `B-13`, dan `B-14` tidak dapat ditulis lengkap: perpindahan tahap adalah
  bagian inti alur keempat modul itu.
- Uji kesetaraan untuk keempat modul tidak dapat dirancang, karena perilaku yang dibandingkan
  belum diketahui seluruhnya.
- Bila ternyata ada transisi aktif yang tidak terbawa, akibatnya adalah klaim yang berhenti di
  satu tahap tanpa ada yang tahu mengapa — kelas cacat yang paling mahal ditemukan setelah rilis.

#### Pertanyaan terbuka

1. **Apakah 8 Ticket rule custom yang hilang tersedia di sistem Pega produksi?** Pemilik: Tim Pega.
   Ini bagian dari permintaan export ulang `D-39`.
2. **Di mana pemicu 14 nama yang tidak punya pemicu berada** — di ruleset lain yang tidak ikut
   export, atau memang sudah mati? Pemilik: Tim Pega.
3. **Bila jawabannya "sudah mati", apakah Work Owner menyetujui transisi itu tidak dibawa?**
   Pemilik: Work Owner.
4. Apakah `CompliancePNC` yang dipakai serentak sebagai **nama Ticket rule dan nama workbasket**
   memang satu hal yang sama? Pemilik: Work Owner.

### 0022 — Tentukan cara menjalankan job terjadwal di aplikasi yang hidup di dua instans

| | |
|---|---|
| **Status** | Proposed |
| **Tanggal keputusan** | — |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Lead Engineer + Tim Infra (mekanisme) · Work Owner (perilaku `AutoAcceptKomite`) |
| **Jejak bukti** | `D-57` (menutup `D-17`, `R-02`), `D-27`, `D-59` · `Job Scheduler/` (5 berkas) · `Agents/TATReportAgent-Agents.xml` · T-8 |
| **Terkait** | ADR-0001, ADR-0008, ADR-0023, modul `S-6` |

> **Belum diputuskan.** Berkas ini memuat konteks, opsi, dan apa yang terhalang — **tanpa bagian
> `Keputusan`**. Jangan dijadikan dasar implementasi.

#### Konteks

Daftar job terjadwal sempat tidak diketahui sama sekali (`R-02`). Pada 2026-09-09 export bertambah
dan folder `Job Scheduler/`, `Agents/`, serta `Service REST/` muncul. `D-57` menutup persoalan
inventarisnya:

| Job | Frekuensi | Jam mulai | Activity target |
|---|---|---|---|
| `JOBForKomiteKlaimPNC` | harian | **06:00:00** | **`AutoAcceptKomite`** |
| `JobHitungDeadlineToTemporaryCLose` | **mingguan** | 23:43:00 | `JobTemporaryCloseClaimPNC` |
| `JobSendAutoLODKlaimPersonal` | harian | 20:54:00 | `Act_SendAutoLODKlaimPersonal` |
| `PNCMyReportKlaim3` | harian | 08:00:00 | `ReportAI_Act` |
| `ProcessClaimKredit` | harian | 10:00:00 | `CreateClaimCredit_Table` |

Seluruhnya `pyIsEnabled=true`, `pyApplicableTo=Cluster`, `pyNodeTypesText=BackgroundProcessing`.
Seluruh activity targetnya ada di export.

Ditambah satu agent: `Agents/TATReportAgent-Agents.xml` — `pyEnable=true`,
**`pyTriggerInterval=1800`** (30 menit), **`pyBypassActivityAuthentication=true`**, merujuk
`CreateCasePNCAgent_ActButton`, `AlertAgentKasirBlmTransferKlaim`, `TransferAllCaseNotAssigned`
(ketiganya ada).

**Inventarisnya selesai; cara menjalankannya yang belum.** Pega menyediakan
`pyApplicableTo=Cluster` — jaminan bahwa satu job berjalan sekali meski ada banyak node. Go tidak
punya padanan otomatis, sementara `D-27` menuntut **minimal dua instans** di belakang load
balancer.

Temuan **T-8** mempertajam persoalan ini: sistem lama **tidak punya satu pun Dynamic System
Setting**. Konfigurasi dinamisnya berupa tabel Oracle yang **dikunci per IP aplikasi** — pola yang
justru bertabrakan dengan dua instans.

#### Opsi yang dipertimbangkan

**Opsi 1 — Penjadwal di dalam binary Go, dengan penguncian lewat database.** Setiap instans
menjalankan penjadwal; yang berhasil mengambil kunci menjalankan job.

- Satu artefak, konsisten dengan ADR-0001; tidak ada komponen baru di produksi.
- Menuntut penguncian yang benar-benar aman, termasuk saat instans mati di tengah job — kelas
  kesalahan yang sulit diuji dan sulit terlihat.

**Opsi 2 — Cron sistem operasi di satu VM**, memanggil endpoint atau subperintah aplikasi.

- Sederhana dan mudah dipahami; tidak ada risiko job ganda.
- Menjadikan satu VM sebagai titik tunggal kegagalan untuk seluruh penjadwalan, dan menempatkan
  jadwal di luar aplikasi — tidak ikut ter-commit, tidak ikut ter-review.

**Opsi 3 — Instans penjadwal terpisah**: satu proses yang sama binary-nya, dijalankan dengan
sakelar sebagai penjadwal.

- Memisahkan beban job dari pelayanan pengguna, dan tetap satu artefak.
- Menambah satu proses yang harus dipantau dan di-deploy, meski binary-nya sama.

#### Yang harus diputuskan Work Owner, bukan teknis

1. **`AutoAcceptKomite` menyetujui komite otomatis setiap hari jam 06:00** — tanpa pengguna sama
   sekali. `D-59` menetapkan satuan izin adalah menu; job ini **melewati kontrol menu apa pun**.
   Apakah perilaku ini dibawa ke sistem baru?
2. **Tiga job ber-`pyDescription = "job jalan 2 menit"`** sementara konfigurasinya `Daily`/`Weekly`
   — selisih 720× sampai 5.040×. Mana yang benar: deskripsinya atau konfigurasinya?
3. **`pyBypassActivityAuthentication=true`** pada agent — diterima untuk sistem baru?

#### Konsekuensi bila dibiarkan tidak diputuskan

- Tiket `S-6` tidak dapat ditulis lengkap.
- Bila penguncian salah dirancang, `AutoAcceptKomite` dapat berjalan **dua kali** — menyetujui
  komite dua kali pada klaim yang sama. Ini kesalahan bernilai uang, bukan gangguan operasional.
- Jadwal sebenarnya tidak dapat dipastikan selama pertanyaan 2 belum dijawab, sehingga uji
  kesetaraan `S-6` tidak dapat dirancang.

#### Pertanyaan terbuka

- Ketiga pertanyaan Work Owner di atas (`AutoAcceptKomite`, deskripsi vs konfigurasi, bypass
  autentikasi).
- Mekanisme mana — Opsi 1, 2, atau 3? Pemilik: Lead Engineer + Infra.
- Bagaimana kegagalan job diketahui (notifikasi, catatan, dashboard)? Di sistem lama tidak ada
  bukti mekanisme apa pun untuk ini. Pemilik: Lead Engineer + Work Owner.

---

## 9. Kelompok E — Keamanan, Konfigurasi, dan Audit

Keputusan tentang siapa boleh melakukan apa, di mana nilai dan rahasia disimpan, dan apa yang tercatat.

| # | Judul | Status |
|---|---|---|
| 0023 | Tegakkan otorisasi berbasis menu di server, dengan 22 peran dan tanpa pemisahan tugas | Accepted |
| 0024 | Delegasikan autentikasi ke HCC/HCQ dan terbitkan sesi milik aplikasi sendiri | Proposed |
| 0025 | Pindahkan seluruh nilai hardcode ke master data dan konfigurasi, dan keluarkan rahasia dari kode | Proposed |
| 0026 | Catat setiap perubahan bernilai bisnis sebagai jejak audit append-only | Accepted |

### 0023 — Tegakkan otorisasi berbasis menu di server, dengan 22 peran dan tanpa pemisahan tugas

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-12 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-58`, `D-59`, `D-07`, `FR-R1` · `docs/verifikasi-bukti-adr.md` §10.3 (T-10) · `Navigation/pyCaseWorkerNavigation-Navigation.xml` · `POOLDATA.T_ACCESS_GROUP_PNC` |
| **Terkait** | ADR-0024, ADR-0026, modul `F-3`, `U-6` |

#### Konteks

Otorisasi di sistem lama adalah **penyembunyian menu semata**. `pyPrivilegeName` terisi pada
**1 dari 902 activity**, dan yang satu itu privilege bawaan Pega untuk ekspor ruleset — bukan
aturan bisnis (T-10). Tidak ada satu pun pemeriksaan izin di sisi server untuk tindakan bisnis.

Yang tersedia sebagai bahan adalah **22 access group Pega**, seluruhnya terverifikasi ada di
export sebagai literal `GCNMFW:<nama>`:

`Administrators` · `CaseManager` · `PncAdmin` · `PncManagerAdmin` · `PncPICTeknik` ·
`PNCKomiteTeknik` · `PNCKomite` · `PncRCLPUCL` · `PncAnalystDoctor` · `PncComplience` ·
`PncInvestigator` · `PNCSurveyor` · `PncPLADLA` · `PncReceive` · `PncManagerReceive` ·
`PncCollection` · `PncOPCGeneral` · `PNCServiceCenter` · `TreatyIn` · `ViewClaimPNC` ·
`PNCReportClaimInternal` · `PNCReportClaimEksternal`

Pemetaan peran → **51 item menu** tidak hidup sebagai data, melainkan di dalam **34 When rule** +
`Navigation/pyCaseWorkerNavigation-Navigation.xml`.

#### Opsi yang dipertimbangkan

Untuk jumlah peran:
1. **22 peran, satu-untuk-satu dengan access group**, hanya dinamai ulang.
2. Lebih sedikit — sebagian access group adalah varian teknis dari peran yang sama.
3. Lebih banyak — ada peran bisnis yang berbagi satu access group.

Untuk satuan izin:
1. Pemisahan tugas nyata — pelaksana tidak boleh menyetujui.
2. Tidak ada pemisahan formal; kontrolnya prosedural.
3. **Sama dengan sistem lama** — siapa pun yang punya akses menu dapat melakukannya.

#### Keputusan

**Peran bisnis berjumlah 22, satu-untuk-satu dengan access group Pega**, hanya dinamai ulang agar
terbaca manusia. Tidak ada penggabungan dan tidak ada pemecahan.

**Satuan izin adalah menu, bukan tindakan individual.** Pengguna yang memiliki akses ke sebuah
menu berwenang atas seluruh tindakan yang dijangkau menu itu — termasuk membatalkan klaim,
mengubah nilai setelah persetujuan komite, dan menyetujui komite. **Tidak ada pemisahan tugas
formal.**

Perbedaan dengan sistem lama bukan pada satuan izinnya, melainkan pada **tempat penegakannya**:
dulu hanya disembunyikan di antarmuka, sekarang **ditegakkan di server pada setiap endpoint** —
yang diperiksa adalah "apakah peran pemanggil memiliki menu yang memberi akses ke endpoint ini".
Dengan bacaan itu, `BRD §21.2` kriteria #8 dan `FR-R1` tetap terpenuhi.

#### Rationale

Merancang pemisahan tugas dari nol berarti menetapkan aturan kewenangan yang belum pernah ada,
pada saat yang sama dengan memigrasikan seluruh aplikasi. Itu dua perubahan besar sekaligus, dan
yang kedua tidak diminta siapa pun.

Menegakkan di server tetap menutup celah terpenting yang ada sekarang: hari ini, siapa pun yang
mengetahui alamat sebuah endpoint dapat memanggilnya tanpa pemeriksaan apa pun.

#### Konsekuensi

##### Positif

- Celah "menu disembunyikan tetapi endpoint terbuka" tertutup sepenuhnya.
- Model izin tetap dikenali pengguna dan administrator — tidak ada konsep baru yang harus
  dipelajari.
- Peran menjadi data di tabel, bukan logika tersebar di 34 When rule.

##### Negatif / utang teknis

- **Orang yang sama dapat membuat, menyetujui, dan membayarkan satu klaim** bila perannya memiliki
  ketiga menu itu. Tidak ada kontrol teknis yang mencegahnya.
- **Jejak audit menjadi satu-satunya kontrol pengimbang.** Ini menaikkan `S-5` dari modul pendukung
  menjadi kontrol utama — dan `S-5` adalah kemampuan **baru 100% tanpa baseline** (ADR-0028).
- **Tiga nama access group muncul dalam dua kapitalisasi** — `ViewClaimPNC`/`VIEWCLAIMPNC` dan
  `PncReceive`/`PNCRECEIVE`. Perbandingan di rule lama tidak konsisten soal huruf besar-kecil;
  sistem baru wajib menormalkannya menjadi satu identitas per peran.
- **Lima When rule hilang dari export** — `IsGCNMReport`, `IsKomite`, `IsNotViewClaim`,
  `IsPNCBonding`, `IsSurvey` — masing-masing mengendalikan satu item menu.
- **Penugasan operator ke peran tidak ada di database.** `POOLDATA.T_ACCESS_GROUP_PNC` hanya
  memetakan `OPERATOR_ID` → `OLD_OPERATOR_ID`. Tabel izin dapat dibangun tetapi **tidak dapat
  diisi** tanpa artefak ini.

##### Risiko yang diterima secara sadar

- `AutoAcceptKomite` (ADR-0022) menyetujui komite tanpa pengguna sama sekali, sehingga bahkan
  kontrol berbasis menu tidak berlaku padanya.
- Bila kelak ada temuan audit atau pentest yang menuntut pemisahan tugas, perubahannya menyentuh
  model izin `F-3` — bukan penyesuaian kecil.
- `BRD §21.2` kriteria #9 (jejak audit untuk setiap perubahan bernilai bisnis) menjadi **wajib
  tanpa pengecualian** pada seluruh tiket modul bisnis, karena ia satu-satunya kontrol yang
  tersisa.

#### Pertanyaan terbuka

- **Dari mana daftar operator per peran diperoleh?** Tanpa sumbernya, `F-3` dapat membangun
  tabelnya tetapi tidak dapat mengisinya. Pemilik: Work Owner + DBA.
- Lima When rule yang hilang — diminta ke Tim Pega, atau aturannya ditetapkan ulang? Pemilik:
  Work Owner.

### 0024 — Delegasikan autentikasi ke HCC/HCQ dan terbitkan sesi milik aplikasi sendiri

| | |
|---|---|
| **Status** | Proposed |
| **Tanggal keputusan** | — |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Tim HCC/HCQ (kontrak) → Work Owner (keputusan akhir) |
| **Jejak bukti** | `D-07`, T-2 · `docs/verifikasi-bukti-adr.md` §1.1 · `D-56` |
| **Terkait** | ADR-0023, ADR-0028, modul `F-3` |

> **Belum dapat dinyatakan `Accepted`.** `D-07` sudah memutuskan arahnya, tetapi **kontrak yang
> menjadi tumpuan keputusan itu tidak dapat diverifikasi sama sekali** dari bahan yang ada.
> Berkas ini memuat arah keputusan beserta apa yang menghalanginya — jangan dijadikan dasar
> implementasi sampai pertanyaan di bawah terjawab.

#### Konteks

`D-07` menetapkan pembagian berikut:

- **Authentication** didelegasikan ke **API internal HCC/HCQ** (username + password). Responsnya
  memuat profil lengkap: NIK, nama, cabang, jabatan, email.
- **Authorization** dimiliki dan dikelola sepenuhnya oleh Claim PNC lewat tabel yang memetakan
  pengguna → menu/izin (ADR-0023).
- Aplikasi menerbitkan **session/token miliknya sendiri** setelah HCC/HCQ memvalidasi kredensial.

Pemisahan itu sehat: identitas dimiliki sistem identitas, kewenangan dimiliki aplikasi yang
memahami bisnisnya.

**Yang menghalangi:** `HCC` dan `HCQ` muncul **2×** di seluruh export, dan **keduanya teks pesan
galat** yang menyuruh pengguna menghubungi helpdesk (T-2). Tidak ada satu pun:

- Connect REST ke HCC/HCQ,
- pemetaan field respons,
- penanganan kegagalan autentikasi,
- mekanisme sesi yang dapat dijadikan pembanding.

Artinya integrasi ini adalah **greenfield sepenuhnya**. `F-3` tidak punya baseline Pega untuk
diuji kesetaraannya (ADR-0028), sehingga **seluruh kelulusannya bertumpu pada kontrak yang belum
ada**.

#### Opsi yang dipertimbangkan

**Opsi 1 — Jalankan `D-07` apa adanya** begitu kontrak HCC/HCQ diperoleh.
Sesuai keputusan yang sudah diambil; seluruh jadwal `F-3` bergantung pada pihak luar.

**Opsi 2 — Rancang lapisan autentikasi dengan antarmuka yang dapat diganti**, dan implementasi
sementara berbasis tabel pengguna lokal untuk pengembangan dan pengujian.
`F-3` dapat berjalan tanpa menunggu; tetapi jalur yang dipakai di pengembangan bukan jalur yang
dipakai di produksi, sehingga kelas cacat integrasi baru muncul terlambat.

**Opsi 3 — Autentikasi lokal sepenuhnya**, tidak memakai HCC/HCQ.
Menghapus ketergantungan pada pihak luar; tetapi memindahkan penyimpanan dan pengelolaan kata sandi
ke aplikasi ini — tanggung jawab keamanan yang jauh lebih besar dan bertentangan dengan `D-07`.

#### Konsekuensi bila dibiarkan tidak diputuskan

- **Tiket `F-3` tidak dapat ditulis lengkap**, dan `F-3` adalah prasyarat bagi `F-4` dan seluruh
  modul bisnis yang memerlukan identitas pengguna.
- Tanpa autentikasi, **tidak ada satu pun modul yang dapat dirilis ke pengguna** — ini jalur kritis
  paling awal di seluruh rencana migrasi.
- Kriteria kelulusan `F-3` tidak dapat dirumuskan: ADR-0028 menggantikan gerbang 1 dengan uji
  fungsional terhadap kontrak, dan kontrak itulah yang tidak ada.

#### Pertanyaan terbuka

1. **Apakah API HCC/HCQ benar-benar ada hari ini, atau harus dibangun?** Pemilik: Tim HCC/HCQ.
   Ini pertanyaan pertama yang harus dijawab sebelum `F-3` dijadwalkan.
2. **Bagaimana bentuk kontraknya** — endpoint, format permintaan dan respons, kode galat, batas
   percobaan? Pemilik: Tim HCC/HCQ.
3. **Apa yang terjadi bila HCC/HCQ tidak dapat dihubungi?** Seluruh aplikasi tidak dapat diakses,
   atau ada jalur cadangan? Pemilik: Work Owner. Ini menyentuh tuntutan 24/7 `D-27`.
4. **Bagaimana pengguna dicocokkan** antara identitas HCC/HCQ dan `OPERATOR_ID` yang dipakai di
   seluruh data klaim? Pemilik: Work Owner + Tim HCC/HCQ. Tanpa pemetaan ini, pengguna yang
   berhasil login tetap tidak dikenali oleh data klaimnya sendiri.
5. Berapa lama sesi berlaku, dan bagaimana ia diperbarui? Pemilik: Work Owner + Security.

### 0025 — Pindahkan seluruh nilai hardcode ke master data dan konfigurasi, dan keluarkan rahasia dari kode

| | |
|---|---|
| **Status** | Proposed |
| **Tanggal keputusan** | — |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Tim Infra/Security (`D-40`) → Work Owner |
| **Jejak bukti** | `D-15`, `D-40` (OPEN), `D-67`, T-8, T-9 · `docs/verifikasi-bukti-adr.md` §7 |
| **Terkait** | ADR-0014, ADR-0015, ADR-0022, modul `F-4`, `F-5` |

> **Arah keputusan sudah ada (`D-15`), tetapi bagian rahasianya belum.** `D-40` masih berstatus
> **OPEN** menunggu Tim Infra/Security. Sampai itu terjawab, berkas ini **tidak boleh** dijadikan
> dasar implementasi `F-4`/`F-5` secara utuh.

#### Konteks

`D-15` menetapkan seluruh nilai hardcode menjadi konfigurasi atau master data. Verifikasi Fase 1
menunjukkan populasinya **jauh lebih besar** daripada yang tercatat di `docs/BRD.md:453`:

| Jenis | Klaim BRD | Terverifikasi di seluruh export |
|---|---|---|
| Alamat email | 10 | **66 unik** |
| User ID | 4 | **24 unik** + belasan tertanam di dalam teks SQL |
| Ambang komite | 3 | **8 unik** + 7 ambang uang non-komite |
| Hostname penentu perilaku | 3 | **3 unik, 48 perbandingan** — angkanya benar, **daftarnya salah** |

Angka BRD benar untuk lingkupnya (dua rule saja), tetapi memakainya untuk `FR-F4` akan membuat
estimasi master data meleset sekitar **enam kali lipat**.

**Tiga hal membuat ini bukan sekadar memindahkan konstanta:**

1. **Hostname menentukan uang.** `Activity/GetKomiteApproval-Act.xml:335` mengubah ambang komite
   dari **50.000.000 menjadi 3.500** berdasarkan nama server — cara entitas Timor-Leste dibedakan.
   Menghapus hardcode di sini berarti mengubah cara entitas dikenali, bukan memindahkan angka.
2. **Blok `// TESTING` menimpa email produksi.** Empat step di
   `Activity/InputRegister_act-Act.xml:16693`, `:16830`, `:16998`, `:17141` menimpa email pimpinan
   dan Underwriting; precondition-nya **identik** dengan step produksi dan posisinya sesudah, dan
   salah satu penimpanya **alamat Gmail pribadi**. Teori "blok ini nonaktif" sudah diuji dan
   gugur: `//` adalah label blok (dipakai 952× di 279 activity), dan format export ini **tidak
   memiliki elemen aktif/nonaktif sama sekali**.
3. **Kredensial plaintext ada di dalam export** (T-9): **3 password SMTP di 31 lokasi** + **1
   pasang kredensial OAuth**. Lokasinya sudah diserahkan lengkap ke Tim Infra/Security lewat
   dokumen terpisah **di luar repo**; tidak ada satu nilai pun tertulis di dokumen yang di-commit
   (`D-69`).

Temuan **T-8** menutup jalan yang tampaknya paling mudah: sistem lama **tidak punya satu pun
Dynamic System Setting**. Konfigurasi dinamisnya berupa tabel Oracle yang **dikunci per IP
aplikasi** — pola yang bertabrakan langsung dengan dua instans (`D-27`).

`D-67` menambahkan satu aturan yang sudah diputuskan: **tidak ada akun pribadi sebagai penerima
notifikasi**; seluruhnya berasal dari master Penerima Notifikasi berupa mailbox fungsional.

#### Opsi yang dipertimbangkan

**Untuk nilai bisnis** (ambang, email penerima, entitas):
1. Master data di database, dikelola lewat layar `U-6`.
2. Berkas konfigurasi aplikasi.
3. Campuran — nilai yang diubah pengguna bisnis di master, nilai teknis di konfigurasi.

**Untuk rahasia** (password SMTP, kredensial OAuth) — inilah yang menunggu `D-40`:
1. Secret manager korporat, bila ada.
2. Variabel lingkungan yang dikelola tim infra.
3. Berkas konfigurasi terenkripsi di luar repo.

#### Konsekuensi bila dibiarkan tidak diputuskan

- **Tiket `F-4` dan `F-5` tidak dapat ditulis lengkap.** `F-4` adalah prasyarat modul bisnis yang
  memakai ambang dan penerima notifikasi.
- **Repo ini menyimpan kredensial plaintext sampai ada keputusan** tentang ke mana ia dipindahkan
  — yang berarti cara repo disimpan dan dibagikan ikut menjadi persoalan keamanan hari ini, bukan
  nanti.
- Pola konfigurasi dinamis belum ditentukan, sehingga ADR-0022 (penjadwal dua instans) ikut
  tertahan.

#### Pertanyaan terbuka

1. **Ke mana rahasia dipindahkan, siapa pemiliknya, dan bagaimana rotasinya?** Pemilik: Tim
   Infra/Security. Ini `D-40`, dan ia menghalangi `F-4`, `F-5`, serta seluruh deployment.
2. **Apakah kredensial yang sudah telanjur plaintext di export harus dirotasi** sebelum sistem
   baru jalan? Pemilik: Tim Infra/Security.
3. **Bagaimana entitas (Indonesia, Timor-Leste, Insurtech) dikenali di sistem baru**, kalau bukan
   dari hostname? Pemilik: Work Owner. Ini menentukan ambang komite mana yang berlaku — ADR-0014.
4. Apakah blok `// TESTING` memang tidak seharusnya ada di produksi, dan sejak kapan ia aktif?
   Pemilik: Work Owner. Jawaban "sudah lama aktif" berarti email produksi selama ini tertimpa, dan
   itu temuan operasional tersendiri.
5. Mana nilai yang boleh diubah pengguna bisnis lewat layar, dan mana yang hanya boleh diubah tim
   teknis? Pemilik: Work Owner. Menentukan pembagian antara `F-4` dan `F-5`.

### 0026 — Catat setiap perubahan bernilai bisnis sebagai jejak audit append-only

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-07 (`D-28`), 2026-09-13 (`D-62`) |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-28`, `D-62`, `D-59`, T-14 · `Database/PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc:7` · `RDB List/UpdateLogServiceClaim-SQL.xml:27` · `RDB List/InsertClaimPNC-SQL.xml:77` |
| **Terkait** | ADR-0012, ADR-0023, ADR-0028, modul `S-5` |

#### Konteks

**Perubahan nilai uang klaim di sistem lama tidak punya jejak audit sama sekali** (T-14). Yang
paling mendekati, `Database/PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc:7`, hanya mencatat **empat
kolom** — jauh dari cukup untuk menjawab "siapa mengubah nilai ini, dari berapa menjadi berapa".

Lebih buruk lagi, dua tabel yang namanya log **terbukti dimutasi**:
`RDB List/UpdateLogServiceClaim-SQL.xml:27` melakukan `UPDATE` pada `claim_service_log`, dan
`RDB List/InsertClaimPNC-SQL.xml:77` melakukan `DELETE` pada `JSON_KLAIM_LOG`.

Kepentingan jejak audit **naik drastis** karena ADR-0023: `D-59` menghapus pemisahan tugas, dan
satuan izin adalah menu. Tidak ada kontrol teknis yang mencegah satu orang membuat, menyetujui,
dan membayarkan satu klaim. **Jejak audit menjadi satu-satunya kontrol pengimbang yang tersisa.**

#### Opsi yang dipertimbangkan

1. **Jejak audit sebagai persyaratan wajib yang dirancang sejak awal**, append-only.
2. Tambahkan jejak audit setelah modul bisnis selesai.
3. Ikuti sistem lama — catat seadanya.

#### Keputusan

Jejak audit adalah **persyaratan wajib yang dirancang sejak awal**, bukan tambahan.

Setiap perubahan bernilai bisnis tercatat permanen — **siapa, kapan, nilai sebelum, nilai
sesudah** — minimal untuk: nilai estimasi klaim, nilai settlement, akseptasi, keputusan komite,
penolakan (RCL), proses ulang (PUCL), perubahan status klaim, dan pembayaran.

Data audit bersifat **append-only**: **tidak boleh diubah atau dihapus oleh jalur aplikasi mana
pun**. Kedua anti-pola di atas tidak dibawa ke sistem baru.

**Retensi jejak audit mengikuti retensi data klaim yang berlaku sekarang** (`D-62`) — satu
kebijakan untuk keduanya, bukan kebijakan terpisah. Sampai angkanya diperoleh, `S-5` dibangun
dengan **retensi sebagai parameter konfigurasi** (konsisten ADR-0025), sehingga modulnya tidak
terhalang.

#### Rationale

Menambahkan jejak audit belakangan berarti setiap modul bisnis harus dibongkar ulang untuk
menyisipkan pencatatan pada setiap titik perubahan — pekerjaan yang lebih besar daripada
merancangnya sejak awal, dan hampir pasti menyisakan titik yang terlewat.

Append-only bukan kehati-hatian berlebihan: sistem lama membuktikan bahwa log yang dapat diubah
**memang diubah**. Sifat append-only harus ditegakkan struktur, bukan diserahkan pada disiplin.

`D-62` menghapus satu penghalang yang tampak besar — `D-28` menggantungkan retensi pada Compliance.
Ternyata kebijakannya sudah ada; yang dibutuhkan hanya angkanya.

#### Konsekuensi

##### Positif

- Pertanyaan "siapa mengubah nilai ini" dapat dijawab — untuk pertama kalinya dalam sejarah
  aplikasi ini.
- Kontrol pengimbang atas ketiadaan pemisahan tugas (ADR-0023) benar-benar ada.
- Soft delete (ADR-0012) dan append-only berdiri di atas prinsip yang sama, tanpa pertentangan.

##### Negatif / utang teknis

- **Volume data audit dapat melampaui data bisnisnya sendiri.** Di atas data historis puluhan juta
  baris (`D-10`), ini keputusan kapasitas, bukan sekadar keputusan kepatuhan.
- Pencatatan menambah satu operasi tulis pada setiap perubahan bernilai bisnis — memengaruhi
  seluruh jalur transaksi.
- **`S-5` adalah kemampuan baru 100% tanpa baseline Pega** (ADR-0028). Tidak ada yang dapat
  dibandingkan untuk membuktikannya benar; kelulusannya bertumpu pada **daftar peristiwa wajib
  audit** yang harus disepakati Compliance dan **belum ada**.
- Append-only menuntut penegakan di tingkat hak akses database, bukan hanya di kode aplikasi —
  dan itu menyentuh kewenangan DBA, bukan tim pengembang.

##### Risiko yang diterima secara sadar

- Angka retensi diambil dari kebijakan yang sudah berjalan dan **belum masuk ke repo maupun
  dokumen proyek mana pun**. `S-5` dibangun dengan parameter, dan angkanya diisi kemudian.
- Selama daftar peristiwa wajib audit belum disepakati, cakupan "perubahan bernilai bisnis" adalah
  daftar minimum di atas — bukan daftar final.

#### Pertanyaan terbuka

- **Angka retensi data klaim yang berlaku sekarang** — berapa? Pemilik: Work Owner + Compliance.
  Tidak menghalangi pembangunan `S-5`, tetapi menghalangi go-live-nya.
- **Daftar peristiwa wajib audit beserta field yang harus tercatat** — pemilik: Compliance
  (`D-56`). Ini kontrak yang menggantikan gerbang 1 bagi `S-5`; tanpanya `S-5` tidak dapat lulus
  gerbang apa pun.
- Apakah jejak audit perlu dapat dibaca pengguna bisnis lewat layar, atau cukup tersedia untuk
  audit? Pemilik: Work Owner.

---

## 10. Kelompok F — Mutu dan Data Uji

Keputusan tentang cara sebuah modul dinyatakan lulus, dan lingkungan tempat pembuktiannya dijalankan.

| # | Judul | Status |
|---|---|---|
| 0027 | Luluskan setiap modul lewat dua gerbang: uji kesetaraan otomatis, lalu UAT bisnis | Accepted |
| 0028 | Ukur modul tanpa baseline Pega dengan kontrak fungsional, dan cabut `BRD §21.4` untuk empat modul | Accepted |
| 0029 | Salin data nasabah apa adanya ke staging; samarkan email dan jangan pernah tulis data nasabah di dokumen | Accepted |

### 0027 — Luluskan setiap modul lewat dua gerbang: uji kesetaraan otomatis, lalu UAT bisnis

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-10 … 2026-09-12 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-42`, `D-53`, `D-54`, `D-60`, `P-5`, `BRD §21.1` · `docs/verifikasi-bukti-adr.md` §15 |
| **Terkait** | ADR-0003, ADR-0017, ADR-0028, ADR-0029, seluruh modul |

#### Konteks

`BRD §21.1` menetapkan dua gerbang untuk setiap modul: **uji kesetaraan otomatis**, lalu **UAT
pengguna bisnis**. Gerbang pertama itu dilayani satu modul perkakas — `S-8`.

`S-8` adalah **satu-satunya modul yang memblokir cutover setiap modul lain**. Menaruhnya di
gelombang akhir berarti tidak ada satu modul pun dapat lulus gerbang 1 sampai gelombang itu tiba,
yang membatalkan gagasan Strangler Fig (ADR-0003). Karena itu `D-42` memindahkannya ke
**gelombang 1**, dan total modul menjadi **33**, bukan 32.

#### Opsi yang dipertimbangkan

Untuk lingkungan pengujian:
1. Tembak produksi langsung — dilarang aturan kerja proyek.
2. **Salinan data produksi di staging**, Pega staging vs Go staging.
3. Data uji buatan.

Untuk kewenangan menyetujui selisih:
1. Setiap selisih disetujui Work Owner.
2. **Selisih yang cocok dengan 13 butir `P-5` lolos otomatis**; sisanya menuntut persetujuan.
3. Tim pengembang memutuskan sendiri.

#### Keputusan

**Gerbang 1 — uji kesetaraan otomatis.** Dijalankan atas **salinan data produksi di lingkungan
staging**, dengan **Pega staging** dan **Go staging** sebagai kedua sisi pembanding, dijalankan
oleh **tim pengembang**.

**Klasifikasi selisih:** selisih yang **cocok dengan salah satu dari 13 butir `P-5`** (ADR-0017)
lolos otomatis dan cukup dicatat. Selisih **di luar 13 butir itu** wajib **persetujuan Work Owner
secara tertulis** sebelum modul dinyatakan lulus. Perkakas `S-8` karenanya wajib
**mengklasifikasikan** setiap selisih, bukan sekadar melaporkannya.

**Gerbang 2 — UAT pengguna bisnis**, berlaku menurut kelompok modul:

| Kelompok | Gerbang 2 | Penguji |
|---|---|---|
| **Modul bisnis** (`B-1`…`B-14`, `S-1`…`S-4`, `S-6`, `S-7`, `U-1`, `U-3`…`U-6`) | **berlaku** | peran bisnis pemakai inbox/layar modul itu |
| **Modul fondasi** (`F-1`…`F-5`, `S-5`, `S-8`, `U-2`) | **tidak berlaku** | gerbang 1 + **persetujuan Work Owner** |

Memaksakan gerbang 2 pada modul yang tidak punya peran pemakai akan membuat tiket fondasi macet
di gerbang yang tidak dapat dilewati siapa pun.

#### Rationale

Data uji buatan tidak memadai, dan buktinya konkret: cacat yang ditemukan pada Fase 1 — toleransi
spreading berupa pencocokan substring, kurs `RETURN 1`, `IDSALVAGE = NULL` — **muncul dari data
nyata yang tidak akan terpikir dibuat**.

Meminta persetujuan untuk selisih yang sudah diputuskan eksplisit di `D-49` menambah beban tanpa
menambah kendali. Yang benar-benar menuntut perhatian adalah selisih **yang tidak terduga** —
itulah yang dipagari.

#### Konsekuensi

##### Positif

- Setiap modul punya kriteria lulus yang sama dan dapat diperiksa ulang.
- Selisih tak terduga tidak dapat lolos diam-diam; ia menjadi antrean persetujuan yang terlihat.
- Modul fondasi tidak tersandera gerbang yang tidak punya penguji.

##### Negatif / utang teknis

- **`S-8` harus selesai sebelum modul mana pun dapat lulus.** Ia menjadi jalur kritis paling awal,
  dan keterlambatannya menunda seluruh migrasi.
- **`S-8` menuntut kemampuan menembak Pega dari luar** untuk membandingkan hasil. Siapa yang boleh,
  di lingkungan mana, dan apakah Pega staging tersedia — **belum dikonfirmasi**. Tiket `S-8`
  karenanya berstatus **TERHALANG** meski modulnya di gelombang 1.
- **Salinan data produksi membawa data nasabah nyata ke staging** (ADR-0029), dengan seluruh
  kewajiban pengamanan yang menyertainya.
- `R-12` — pergeseran zona waktu saat migrasi data — berlaku pada **proses penyalinan itu
  sendiri**, bukan hanya pada migrasi akhir. Salinan yang bergeser akan menghasilkan selisih palsu
  di setiap pengujian.
- Dua modul **tidak punya baseline Pega** untuk diuji setara: `F-3` dan `S-5` (ADR-0028).

##### Risiko yang diterima secara sadar

- **Waktu pengguna bisnis untuk UAT belum dialokasikan resmi** (`C-9`, `R-15`). Gerbang 2
  bergantung pada ketersediaan orang yang sedang menjalankan operasional harian.
- Persetujuan tertulis Work Owner untuk setiap selisih tak terduga membuat Work Owner menjadi
  titik tunggal yang dapat menghambat kelulusan modul.

#### Pertanyaan terbuka

- **Apakah Pega staging yang dapat ditembak dari luar tersedia?** Pemilik: Tim Pega + Tim Infra.
  Ini penghalang utama `S-8`.
- Kapan waktu pengguna bisnis untuk UAT dialokasikan resmi? Pemilik: manajemen (`R-15`).
- Apakah salinan data produksi disegarkan berkala, dan berapa lama disimpan? Pemilik: Work Owner
  + Infra (bertaut dengan ADR-0029).

### 0028 — Ukur modul tanpa baseline Pega dengan kontrak fungsional, dan cabut `BRD §21.4` untuk empat modul

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-12 |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-56`, `D-55`, `D-42`, T-2, T-14 · `BRD §21.2` kriteria #3 · `BRD §21.4` |
| **Terkait** | ADR-0024, ADR-0026, ADR-0027, modul `F-3`, `S-5`, `B-5`, `B-7`, `B-9`, `B-10`, `B-12` |

> **Menyupersede `BRD §21.4`** untuk `B-7`, `B-9`, `B-10`, dan `B-12`. Revisi BRD-nya diajukan
> sebagai pertanyaan terbuka di bawah — BRD **tidak** diedit oleh ADR ini.

#### Konteks

Gerbang 1 (ADR-0027) membandingkan perilaku sistem baru dengan sistem lama. Dua modul **tidak
punya sistem lama untuk dibandingkan**:

- **`F-3` Identitas & Akses** — HCC/HCQ **nol jejak di export** (T-2). Tidak ada mekanisme
  autentikasi yang dapat dijadikan pembanding.
- **`S-5` Jejak Audit** — sistem lama **tidak punya jejak audit atas nilai** (T-14). Tidak ada
  keluaran yang dapat dibandingkan.

Untuk keduanya, `BRD §21.2` kriteria #3 (uji kesetaraan) **tidak dapat diberlakukan**.

Persoalan kedua datang dari sisi lain: `BRD §21.4` menahan sejumlah modul sampai penghalangnya
hilang. Setelah export bertambah pada 2026-09-09 dan `Database/` diterima, sebagian penghalang itu
**sudah tidak ada lagi** — sementara pasalnya masih mengikat.

#### Opsi yang dipertimbangkan

1. **Ganti gerbang 1 dengan uji fungsional terhadap kontrak** yang disepakati.
2. Hapus gerbang 1 untuk kedua modul, cukup UAT.
3. Tunda kedua modul sampai ada pembanding.

#### Keputusan

**Untuk `F-3` dan `S-5`, gerbang 1 diganti uji fungsional terhadap kontrak yang disepakati:**

| Modul | Kontrak yang menjadi acuan | Pemilik kontrak |
|---|---|---|
| **F-3** | kontrak API **HCC/HCQ** — field request/response login, kode galat, timeout, endpoint refresh/validasi | pemilik API HCC/HCQ |
| **S-5** | **daftar peristiwa wajib audit** beserta field yang harus tercatat | Compliance |

**`BRD §21.4` dicabut untuk `B-7`, `B-9`, `B-10`, dan `B-12`**, dan **dipertahankan hanya untuk
`B-5`**. Sisa penghalang per modul:

| Modul | Status | Sisa penghalang |
|---|---|---|
| **B-7** Komite | lepas dari §21.4 | PA dan Travel di atas Rp 200.000.000 tidak punya baris master |
| **B-9** PLA/DLA | lepas dari §21.4 | status `VALID`/`INVALID` `GET_GROUPBUSINESS_XOL` di produksi; penulisan ulang procedure ber-9-commit (ADR-0007) |
| **B-10** Akseptasi | lepas dari §21.4 | 3 activity hilang: `InsertDataAkseptasiToLeader`, `InsertLogKasir_act`, `TransferCashierDataASM_act` |
| **B-12** Salvage | lepas dari §21.4 | `SET_ATTACHFILETEMPSALVAGE`; hitungan DBA atas baris ber-`IDSALVAGE` NULL |
| **B-5** Settlement | **tetap terikat §21.4** | isi `POOLDATA.GCNM_FEE_SCALE` (17 pita) · isi `m_currencystandard` · `BrowseT_Claim_Adjustment_SQL` |

#### Rationale

Menghapus gerbang 1 (opsi 2) justru pada dua modul **paling sensitif** — `F-3` adalah otorisasi,
`S-5` adalah jejak audit — menyisakan hanya UAT sebagai pemeriksaan. Dan UAT tidak memeriksa hal
yang tidak terlihat di layar: tidak ada penguji bisnis yang dapat melihat apakah sebuah perubahan
nilai benar-benar tercatat.

Menunda kedua modul (opsi 3) mustahil: `F-3` adalah prasyarat seluruh modul yang memerlukan
identitas pengguna.

Mempertahankan `§21.4` untuk modul yang penghalangnya sudah hilang berarti menahan pekerjaan atas
alasan yang tidak lagi berlaku — persis jenis penundaan yang `D-35` tolak (angka mengikuti bukti,
bukan dokumen).

#### Konsekuensi

##### Positif

- `F-3` dan `S-5` punya ukuran kelulusan yang nyata, bukan sekadar dikecualikan.
- Empat modul bisnis dapat mulai dikerjakan tanpa menunggu pencabutan pasal secara formal.
- Penghalang yang tersisa per modul menjadi eksplisit dan dapat ditelusuri satu per satu.

##### Negatif / utang teknis

- **Kedua kontrak belum ada.** Sampai keduanya diterima, `F-3` tidak dapat lulus gerbang apa pun
  dan tiketnya tetap `needs-info` — begitu pula `S-5`. Keputusan ini **menciptakan ketergantungan
  baru** kepada dua pihak luar.
- Uji terhadap kontrak hanya sekuat kontraknya. Kontrak yang ditulis buru-buru akan meloloskan
  modul yang sebenarnya belum benar.
- Mencabut `§21.4` untuk empat modul berarti **BRD dan kenyataan kini berbeda** sampai revisinya
  disetujui. Dokumen yang tidak sinkron adalah sumber kebingungan bagi pembaca baru.
- `B-5` tetap tertahan, dan tiga penghalangnya seluruhnya bergantung pada data yang harus diminta
  ke DBA.

##### Risiko yang diterima secara sadar

- Dua modul paling sensitif diuji dengan cara yang berbeda dari 31 modul lainnya — perbedaan yang
  harus diingat setiap kali hasil pengujian dibaca.
- Pencabutan `§21.4` dilakukan lewat ADR, sementara BRD-nya menunggu revisi. Bila revisi itu tidak
  pernah disetujui, terdapat dua dokumen yang saling bertentangan.

#### Pertanyaan terbuka

- **Usulan revisi `BRD §21.4`** agar sesuai keputusan ini — menunggu persetujuan Work Owner.
  Diajukan di Fase 6, tidak dikerjakan diam-diam.
- Kapan kontrak API HCC/HCQ tersedia? Pemilik: Tim HCC/HCQ (ADR-0024).
- Kapan daftar peristiwa wajib audit tersedia? Pemilik: Compliance (ADR-0026).
- Baris master komite untuk PA dan Travel di atas Rp 200.000.000 — memang tidak ada, atau belum
  diisi? Pemilik: Work Owner (ADR-0014).

### 0029 — Salin data nasabah apa adanya ke staging; samarkan email dan jangan pernah tulis data nasabah di dokumen

| | |
|---|---|
| **Status** | Accepted |
| **Tanggal keputusan** | 2026-09-13 (`D-64`), 2026-09-14 (`D-69`) |
| **Tanggal dokumen** | 2026-09-14 |
| **Sifat** | retrospective |
| **Pemilik keputusan** | Work Owner |
| **Jejak bukti** | `D-64`, `D-69`, `D-40` (OPEN), `D-53`, `FR-R2` · `docs/verifikasi-bukti-adr.md` §7.4 |
| **Terkait** | ADR-0025, ADR-0027, modul `S-8` + seluruh dokumen proyek |

#### Konteks

ADR-0027 menetapkan uji kesetaraan dijalankan atas **salinan data produksi di staging** — karena
data uji buatan terbukti tidak mampu memunculkan cacat yang ditemukan Fase 1. Konsekuensinya:
**data nasabah nyata berpindah ke lingkungan staging**.

Persoalan kedua terpisah tetapi sering tertukar dengan yang pertama: **apa yang boleh tertulis di
ADR dan tiket** yang akan di-commit. Nama seperti `MORASOTARDODOTARIGAN`, `ELLENSUPRIYATI`, dan
`INDRAGUNAWAN` **sudah tertulis apa adanya** di dokumen yang sudah disetujui (`D-15`,
`20-DETAIL-KOMITE-DBLINK.md` §1.5.4); alamat email tidak pernah.

#### Opsi yang dipertimbangkan

Untuk isi lingkungan staging:
1. **Salin apa adanya**, hak akses staging diperketat sebagai kontrol pengganti.
2. Samarkan data nasabah sebelum disalin.
3. Pakai data uji buatan.

Untuk isi dokumen:
1. **Nama Operator ID boleh ditulis; alamat email selalu disamarkan; data nasabah tidak pernah.**
2. Keduanya disamarkan, termasuk nama Operator ID.
3. Keduanya boleh ditulis lengkap.

#### Keputusan

**Isi lingkungan staging:** data produksi disalin **apa adanya, tanpa penyamaran**, dengan **hak
akses lingkungan staging diperketat** sebagai kontrol penggantinya.

**Isi dokumen yang di-commit:**

| Jenis nilai | Perlakuan |
|---|---|
| **Nama Operator ID** | **boleh ditulis lengkap** — preseden sudah ada di Steering yang disetujui, dan diperlukan agar tiket dapat menunjuk hardcode mana yang harus dihapus |
| **Alamat email** | **selalu disamarkan** — bagian sebelum `@` diganti, domain boleh tampak |
| **Data nasabah** — nomor polis, nama tertanggung, nomor klaim asli, NPWP, nomor rekening | **tidak pernah ditulis**; dirujuk dengan `berkas:baris` atau nama kolom saja |
| **Kredensial, kunci API, hostname/IP produksi** | **tidak pernah ditulis**; dirujuk dengan `berkas:baris` + nama elemen (ADR-0025) |

Kedua keputusan berdiri sendiri: yang pertama mengatur **isi lingkungan**, yang kedua mengatur
**isi dokumen**.

#### Rationale

Penyamaran data sebelum disalin akan menghilangkan justru sifat data yang membuatnya berguna
untuk uji kesetaraan — nilai ekstrem, karakter tak terduga, dan kombinasi yang tidak akan terpikir
dibuat. Cacat seperti toleransi spreading substring hanya terlihat pada data yang nyata.

Untuk dokumen, nama Operator ID diperlukan demi ketelusuran: tanpanya, tiket `F-4` tidak dapat
menunjuk hardcode mana yang harus dihapus. Alamat email **tidak menambah ketelusuran apa pun**,
sehingga tidak ada alasan menuliskannya.

#### Konsekuensi

##### Positif

- Uji kesetaraan berjalan atas data yang benar-benar mewakili produksi.
- Aturan penulisan dokumen menjadi tegas dan dapat diperiksa — tidak ada penilaian kasus per kasus.
- Tiket tetap dapat menunjuk hardcode secara spesifik tanpa membocorkan data nasabah.

##### Negatif / utang teknis

- **Staging menjadi lingkungan yang memuat data nasabah nyata** — nomor polis, nama tertanggung,
  NPWP, nomor rekening, dan **data medis** pada lini PA dan Travel. Perlindungannya sepenuhnya
  bergantung pada hak akses, bukan pada sifat datanya.
- **Klasifikasi staging naik setara produksi** untuk keperluan keamanan. Siapa yang punya akses,
  berapa lama salinan disimpan, dan kapan dimusnahkan menjadi pertanyaan yang harus dijawab — dan
  **belum dijawab**.
- Pembatasan akses data medis yang `FR-R2` terapkan pada peran Analyst Doctor dan RCL Dokter
  **berlaku juga di staging**, bukan hanya produksi. Ini menambah pekerjaan pada `F-3` di
  lingkungan yang biasanya dianggap bebas.
- Keputusan ini **berdampingan dengan `D-40` yang masih terbuka**: export sudah memuat kredensial
  plaintext, dan kini salinan data nasabah menyusul ke lingkungan yang sama-sama belum ditetapkan
  pengamanannya.

##### Risiko yang diterima secara sadar

- Kebocoran dari staging akan setara dengan kebocoran dari produksi. Tidak ada mitigasi teknis
  tambahan — hanya hak akses.
- Nama pegawai tertulis lengkap di dokumen yang di-commit dan akan dibaca banyak orang. Ini
  diterima demi ketelusuran, dengan kesadaran penuh bahwa ia data pribadi.

#### Pertanyaan terbuka

- **Siapa yang menyetujui akses staging, berapa lama salinan disimpan, dan bagaimana prosedur
  pemusnahannya?** Pemilik: Tim Infra/Security + Compliance — pihak yang sama dengan `D-40`.
- Apakah salinan data disegarkan berkala, dan apakah setiap penyegaran menuntut persetujuan ulang?
  Pemilik: Work Owner.
- Apakah data medis di staging perlu perlakuan lebih ketat daripada data nasabah lainnya?
  Pemilik: Compliance.

---

## Lampiran A — Template ADR Kosong

```markdown
# NNNN — <judul keputusan, kalimat aktif>

Status: Proposed | Accepted | Superseded by ADR-XXXX | Deprecated
Tanggal keputusan: YYYY-MM-DD    Tanggal dokumen: YYYY-MM-DD
Sifat: original | retrospective
Pemilik keputusan: <peran, mis. Work Owner / Lead Engineer / Tim Infra>
Jejak bukti: D-nn | FR-xx | R-nn | path/berkas:baris | nama objek DB
Terkait: CONTEXT.md#<istilah>, ADR-XXXX, modul <kode>

## Konteks
## Opsi yang dipertimbangkan
## Keputusan
## Rationale
## Konsekuensi
### Positif
### Negatif / utang teknis
### Risiko yang diterima secara sadar
## Pertanyaan terbuka
```

---

## Lampiran B — Aturan Menulis yang Mengikat

1. Satu berkas = satu keputusan yang mahal dibalik.
2. `Sifat: retrospective` wajib ditulis terbuka bila keputusan mendahului dokumennya.
3. **Bagian `Negatif / utang teknis` tidak boleh kosong.** ADR yang hanya memuat keuntungan adalah dokumen jualan, bukan ADR.
4. Jangan menulis `Keputusan` untuk hal yang belum diputuskan Work Owner — pakai `Proposed`, sebutkan pemilik keputusan dan apa yang diblokir.
5. Bila ADR bertentangan dengan Steering atau BRD, tulis **`Menyupersede …`** secara eksplisit dan ajukan revisinya sebagai pertanyaan terbuka. Jangan mengedit Steering diam-diam.
6. Nilai sensitif tidak pernah ditulis: kredensial, kunci API, hostname/IP produksi, dan data nasabah dirujuk dengan `berkas:baris` + nama elemen saja. Alamat email disamarkan (`D-69`).

---

## Lampiran C — Rekapitulasi Seluruh Pertanyaan Terbuka

Dikumpulkan otomatis dari bagian **Pertanyaan terbuka** setiap ADR. Ini daftar kerja untuk
Work Owner dan pihak luar — bukan bagian dari keputusan yang sudah diambil.

### 0001 — Bangun satu modular monolith Go yang dijalankan di VM on-premise

- Apakah kebijakan backup korporat yang berlaku hari ini memang sejalan dengan tuntutan 24/7
  (`D-29`)? Pemilik: Tim Infra. Selama belum dijawab, target pemulihan aplikasi tidak dapat
  dinyatakan di tiket mana pun.
- Bagaimana dua instans berbagi berkas sementara (hasil export besar) bila tidak ada storage
  bersama? Pemilik: Lead Engineer + Infra. Menghalangi bagian export bervolume besar di `S-2`.

### 0002 — Bangun antarmuka sebagai SPA React yang disajikan oleh binary Go

- TanStack Table atau AG Grid? Pemilik: Lead Engineer, dengan persetujuan Work Owner bila
  berbiaya lisensi. Menghalangi penyelesaian tiket `U-2`.
- `D-12` menetapkan pemakaian lapangan untuk survei. Apakah `U-3`/`U-4` wajib berfungsi penuh di
  peramban ponsel, atau cukup layar survei `B-8` saja? Pemilik: Work Owner.

### 0003 — Alihkan modul satu per satu dengan Pega dan Go berjalan paralel

- Apa tanggal akhir yang mengikat bagi masa paralel — kapan Pega dimatikan? Pemilik: Work Owner.
  Tanpa ini, biaya pemeliharaan ganda tidak berbatas.
- Bila sebuah gelombang gagal di gerbang 2, apakah gelombang berikutnya tetap berjalan sesuai
  jadwal `D-61`? Pemilik: Work Owner + manajemen.

### 0004 — Pakai satu database bersama selama masa paralel, dengan penulis tunggal per tabel

- Bagaimana pelanggaran aturan penulis tunggal dideteksi — pemeriksaan berkala, trigger audit,
  atau tidak sama sekali? Pemilik: Lead Engineer + DBA. Tanpa jawaban, `P-1` hanya imbauan.
- Apakah aplikasi Go mendapat skema (schema) Oracle sendiri, atau menumpang skema yang ada?
  Pemilik: DBA + Work Owner. Menghalangi penulisan tiket `F-2`.

### 0005 — Tulis satu set SQL portabel: jalan di Oracle sekarang, PostgreSQL 17+ kemudian

- Kapan lingkungan PostgreSQL 17+ tersedia untuk pengujian portabilitas? Pemilik: Tim Infra.
  Selama belum ada, tidak ada tiket yang boleh menyatakan SQL-nya "terbukti portabel".
- Apakah 97 predikat join pada kueri terberat tetap berkinerja wajar di PostgreSQL? Pemilik:
  DBA + Lead Engineer. Menghalangi kepastian NFR kinerja.

### 0006 — Bekukan data polis sebagai snapshot saat registrasi; kepemilikannya tetap di GISFW

- Kapan sebuah snapshot boleh atau harus disegarkan, dan siapa yang berwenang memicunya?
  Pemilik: Work Owner. Menghalangi penyelesaian tiket `B-1`.
- Field polis apa saja yang wajib masuk snapshot? Daftar ini hanya dapat disusun dengan membaca
  seluruh pemakaian `JSON_POLIS` di rule, dan belum dikerjakan. Pemilik: Lead Engineer.
- Apakah tim GISFW menyetujui bahwa Claim PNC menyimpan salinan data mereka? Pemilik: Work Owner
  → Tim GISFW.

### 0007 — Naikkan logika stored procedure ke Go dan pindahkan kepemilikan transaksi ke aplikasi

- Kapan kueri `ALL_DEPENDENCIES` dijalankan untuk membuktikan `D-68` sebelum procedure
  dinonaktifkan? Pemilik: DBA. Menghalangi langkah mematikan procedure, bukan penulisan ulangnya.
- Sumber 12 objek dependensi yang belum diserahkan — kapan tersedia? Pemilik: Tim Pega/DBA.
  Menghalangi penyelesaian tiket `B-5`, `B-9`, `B-10`, `B-12` (`R-01`).

### 0008 — Ganti seluruh akses DB Link dengan pemanggilan API ke sistem pemilik data

- API pengganti untuk enam DB Link — sudah ada, atau harus dibangun tim pemiliknya? Pemilik: Work
  Owner → tim masing-masing sistem. Ini `R-03`, dan menghalangi tiket `S-4`.
- **Ditemukan setelah `D-25` diputuskan:** `Service REST/` memuat **empat layanan masuk** —
  `KomiteAcceptAdjustment`, `KomiteAcceptAdjustmentPA`,
  `RecivedDataandAttachmentLelangASMSimasbid`, `RequestCreateClaimCredit2`. Dua di antaranya
  **menerima persetujuan komite dari sistem lain**. Permukaan masuk ini belum pernah masuk
  hitungan `FR-S4` maupun `D-25`, dan lingkup `S-4` karenanya **lebih besar daripada yang
  disetujui**. Pemilik: Work Owner.

### 0009 — Terbitkan nomor klaim baru berformat `PNCN.YY.xxxx` dari sequence database

- **Apakah sequence direset setiap awal tahun?** Bila tidak, nomor urut terus bertambah melewati
  pergantian tahun (`PNCN.26.8125` → `PNCN.27.8126`) dan segmen tahun menjadi penanda, bukan
  penghitung per tahun. Bila ya, nomor mulai dari 1 tiap tahun. Keduanya sah; pilihannya mengubah
  cara nomor dibaca orang. Pemilik: Work Owner. Menghalangi penyelesaian tiket `B-2`.
- **Apakah lebar segmen terakhir perlu dibuat tetap** (mis. `TO_CHAR(seq.NEXTVAL,'FM0000')`) agar
  pengurutan teks benar? Pemilik: Work Owner. Sampai dijawab, sintaks yang ditetapkan dipakai apa
  adanya.
- Apakah pihak luar (reasuransi, broker, BPPDAN) perlu diberi tahu perubahan format sebelum klaim
  pertama terbit dari sistem baru? Pemilik: Work Owner.

### 0010 — Simpan dokumen lewat satu jalur API storage internal; database hanya menyimpan metadata

- Bagaimana dokumen lama pada tiga mekanisme lama diperlakukan — dimigrasikan, atau dibaca di
  tempatnya selamanya? Pemilik: Work Owner. Menghalangi penyelesaian tiket `S-1`.
- Saat metadata dokumen dihapus secara soft delete, apakah berkas di storage ikut dihapus?
  Pemilik: Work Owner + Compliance.
- Berapa masa berlaku URL dokumen, dan apa yang terjadi setelah lewat? Pemilik: Tim pemilik API
  storage.

### 0011 — Bangun pembuatan PDF, Excel, dan CSV di dalam aplikasi Go

- Berapa baris maksimum yang wajib dilayani satu export? Pemilik: Work Owner. Tanpa angka ini,
  `S-2` tidak dapat dinyatakan selesai secara terukur.
- Apakah export besar dijalankan serentak dengan permintaan pengguna, atau diantrekan dan
  diberitahukan saat selesai? Pemilik: Work Owner + Lead Engineer.
- Apakah keluaran PDF wajib identik secara visual dengan keluaran Pega, atau cukup identik secara
  isi? Pemilik: Work Owner. Ini menentukan kriteria kelulusan gerbang 1 untuk `S-2`.

### 0012 — Terapkan soft delete menyeluruh; tidak ada penghapusan fisik data bernilai bisnis

- Siapa yang berwenang melihat data yang sudah ditandai terhapus, dan lewat layar apa? Pemilik:
  Work Owner. Tanpa ini, soft delete hanya menyembunyikan data tanpa manfaat pemulihan.
- Apakah ada titik waktu ketika baris bertanda terhapus benar-benar dibuang (arsip atau purge),
  mengikuti `D-62`? Pemilik: Work Owner + Compliance.

### 0013 — Tentukan pengganti pola hapus-lalu-sisip-ulang pada konversi klaim

1. **Seberapa sering konversi ulang benar-benar dijalankan di produksi, dan untuk keperluan apa?**
   Pemilik: Work Owner. Jawaban ini menentukan apakah Opsi 3 layak dipertimbangkan sama sekali.
2. **Apakah ke-12 tabel punya kunci alami yang unik?** Pemilik: DBA + Lead Engineer. Menentukan
   apakah Opsi 1 dapat dipakai seragam atau hanya sebagian.
3. **Apakah riwayat hasil konversi sebelumnya perlu disimpan?** Pemilik: Work Owner + Compliance.
   Bila ya, hanya Opsi 2 yang memenuhi.
4. Duplikasi `T_DLALIST`/`T_PLALIST` yang sudah terjadi hari ini — apakah datanya perlu
   dibersihkan sebelum migrasi? Pemilik: Work Owner + DBA.

### 0014 — Hitung jenjang komite secara kumulatif menurut ambang bawah, dengan pita nilai hanya di Non-MBU

- **Belum terverifikasi:** tiga kueri yang memfilter `TYPE_KOMITE` secara dinamis —
  `EmailKomiteBerjenjang_sql` (`Activity/SetEmailKomite-Act.xml`), `EmailKomiteAdjuster_sql`
  (`Activity/SetEmailKomiteAdjuster-Act.xml`), `EmailKomiteSalvage_sql`
  (`Activity/SetEmailKomiteSalvage-Act.xml`) — menerima nilainya dari pemanggil lewat
  `tempAdj.pyMemo` dan `tempAdj.AcceptedNo`. **Lini apa saja yang benar-benar melewati ketiga
  kueri itu belum ditelusuri sampai ke sumber nilainya.** Pemilik: Lead Engineer. Ini bagian
  Definition of Ready tiket `B-7` dan `B-12`.
- Apakah `TYPE_KOMITE` kelak dipecah menjadi dua kolom saat master diisi ulang di `F-4`?
  Pemilik: Work Owner.

### 0015 — Konversi mata uang memakai kurs tanggal kejadian; tolak klaim bila kurs tidak ditemukan

- **Isi tabel `m_currencystandard` belum ada di repo** — masih diminta ke DBA (`R-19`). Tanpa itu
  `B-5` tidak dapat diuji. Pemilik: DBA.
- Siapa yang mengisi kurs harian dan dari sumber apa (Bank Indonesia, kurs korporat, atau lain)?
  Pemilik: Work Owner. Menghalangi penyelesaian tiket `F-4`.
- Bagaimana klaim yang tertolak karena kurs kosong diperlakukan — ditahan dan diproses ulang
  otomatis, atau dikembalikan ke petugas? Pemilik: Work Owner.

### 0016 — Simpan nilai uang presisi penuh; validasi total spreading dengan toleransi empat desimal

- Berapa desimal yang dipakai pada tampilan untuk nilai Rupiah dan untuk persentase share?
  Pemilik: Work Owner. Menghalangi penyelesaian tiket `U-2`.
- Berapa presisi yang disepakati untuk kolom uang di skema baru? Pemilik: Lead Engineer + DBA.

### 0017 — Perbaiki sembilan cacat aturan uang; replikasi satu yang perilakunya memang benar

- **Berapa banyak baris yang rusak akibat butir 9** (`IDSALVAGE = NULL`), dan apakah datanya
  diperbaiki sebelum migrasi? Pemilik: DBA + Work Owner. Menghalangi penyelesaian tiket `B-12`.
- Apakah data historis yang terdampak butir 1 (total spreading di luar toleransi) dibiarkan apa
  adanya? Pemilik: Work Owner.

### 0018 — Pertahankan empat konsep status, masing-masing dengan nama yang tidak dapat tertukar

- **`SELECT DISTINCT STATUSPOSISI`** pada data produksi — masih ada di daftar permintaan ke DBA,
  untuk menyelesaikan T-6 secara empiris. Pemilik: DBA. Tidak menghalangi tiket mana pun, tetapi
  menentukan apakah kolom itu perlu dibawa.
- Bagaimana `Flag Klaim` yang dimiliki GISFW diperlakukan — dibaca dari snapshot (ADR-0006), atau
  dibaca langsung? Pemilik: Work Owner + Tim GISFW.
- Apa makna persis `0` dan `1` pada Flag Klaim? Pemilik: Work Owner. Belum terjawab sejak `D-18`.

### 0019 — Pertahankan Worklist dan Workbasket; bangun ulang router sebagai aturan routing

- Logika tiga router yang hilang (`R-04`) — digali dari Tim Pega, atau ditetapkan ulang oleh Work
  Owner sebagai aturan baru? Pemilik: Work Owner. Menghalangi penyelesaian tiket `B-6`.
- Apa yang terjadi pada tugas milik seseorang yang sedang tidak hadir (`STS_ABS`)? Pemilik: Work
  Owner.
- Apakah pencacah beban direset berkala? Pemilik: Work Owner. Tanpa jawaban, perilaku jangka
  panjang pembagian beban tidak dapat ditiru.

### 0020 — Pertahankan dua basis perhitungan TAT untuk keperluan yang berbeda

- Sumber `DATAMINING.GET_WORKING_HOURS@ASMD` — kapan tersedia untuk dibaca? Tanpa itu aturan jam
  kerjanya tidak dapat ditulis ulang. Pemilik: DBA (`R-03`). Menghalangi tiket `S-7`.
- Dari mana daftar hari libur diperoleh setiap tahun di sistem baru, dan siapa yang mengisinya?
  Pemilik: Work Owner. Menghalangi penyelesaian tiket `F-4` dan `F-5`.

### 0021 — Bangun ulang transisi lateral (Ticket rule) sebagai perpindahan tahap yang eksplisit

1. **Apakah 8 Ticket rule custom yang hilang tersedia di sistem Pega produksi?** Pemilik: Tim Pega.
   Ini bagian dari permintaan export ulang `D-39`.
2. **Di mana pemicu 14 nama yang tidak punya pemicu berada** — di ruleset lain yang tidak ikut
   export, atau memang sudah mati? Pemilik: Tim Pega.
3. **Bila jawabannya "sudah mati", apakah Work Owner menyetujui transisi itu tidak dibawa?**
   Pemilik: Work Owner.
4. Apakah `CompliancePNC` yang dipakai serentak sebagai **nama Ticket rule dan nama workbasket**
   memang satu hal yang sama? Pemilik: Work Owner.

### 0022 — Tentukan cara menjalankan job terjadwal di aplikasi yang hidup di dua instans

- Ketiga pertanyaan Work Owner di atas (`AutoAcceptKomite`, deskripsi vs konfigurasi, bypass
  autentikasi).
- Mekanisme mana — Opsi 1, 2, atau 3? Pemilik: Lead Engineer + Infra.
- Bagaimana kegagalan job diketahui (notifikasi, catatan, dashboard)? Di sistem lama tidak ada
  bukti mekanisme apa pun untuk ini. Pemilik: Lead Engineer + Work Owner.

### 0023 — Tegakkan otorisasi berbasis menu di server, dengan 22 peran dan tanpa pemisahan tugas

- **Dari mana daftar operator per peran diperoleh?** Tanpa sumbernya, `F-3` dapat membangun
  tabelnya tetapi tidak dapat mengisinya. Pemilik: Work Owner + DBA.
- Lima When rule yang hilang — diminta ke Tim Pega, atau aturannya ditetapkan ulang? Pemilik:
  Work Owner.

### 0024 — Delegasikan autentikasi ke HCC/HCQ dan terbitkan sesi milik aplikasi sendiri

1. **Apakah API HCC/HCQ benar-benar ada hari ini, atau harus dibangun?** Pemilik: Tim HCC/HCQ.
   Ini pertanyaan pertama yang harus dijawab sebelum `F-3` dijadwalkan.
2. **Bagaimana bentuk kontraknya** — endpoint, format permintaan dan respons, kode galat, batas
   percobaan? Pemilik: Tim HCC/HCQ.
3. **Apa yang terjadi bila HCC/HCQ tidak dapat dihubungi?** Seluruh aplikasi tidak dapat diakses,
   atau ada jalur cadangan? Pemilik: Work Owner. Ini menyentuh tuntutan 24/7 `D-27`.
4. **Bagaimana pengguna dicocokkan** antara identitas HCC/HCQ dan `OPERATOR_ID` yang dipakai di
   seluruh data klaim? Pemilik: Work Owner + Tim HCC/HCQ. Tanpa pemetaan ini, pengguna yang
   berhasil login tetap tidak dikenali oleh data klaimnya sendiri.
5. Berapa lama sesi berlaku, dan bagaimana ia diperbarui? Pemilik: Work Owner + Security.

### 0025 — Pindahkan seluruh nilai hardcode ke master data dan konfigurasi, dan keluarkan rahasia dari kode

1. **Ke mana rahasia dipindahkan, siapa pemiliknya, dan bagaimana rotasinya?** Pemilik: Tim
   Infra/Security. Ini `D-40`, dan ia menghalangi `F-4`, `F-5`, serta seluruh deployment.
2. **Apakah kredensial yang sudah telanjur plaintext di export harus dirotasi** sebelum sistem
   baru jalan? Pemilik: Tim Infra/Security.
3. **Bagaimana entitas (Indonesia, Timor-Leste, Insurtech) dikenali di sistem baru**, kalau bukan
   dari hostname? Pemilik: Work Owner. Ini menentukan ambang komite mana yang berlaku — ADR-0014.
4. Apakah blok `// TESTING` memang tidak seharusnya ada di produksi, dan sejak kapan ia aktif?
   Pemilik: Work Owner. Jawaban "sudah lama aktif" berarti email produksi selama ini tertimpa, dan
   itu temuan operasional tersendiri.
5. Mana nilai yang boleh diubah pengguna bisnis lewat layar, dan mana yang hanya boleh diubah tim
   teknis? Pemilik: Work Owner. Menentukan pembagian antara `F-4` dan `F-5`.

### 0026 — Catat setiap perubahan bernilai bisnis sebagai jejak audit append-only

- **Angka retensi data klaim yang berlaku sekarang** — berapa? Pemilik: Work Owner + Compliance.
  Tidak menghalangi pembangunan `S-5`, tetapi menghalangi go-live-nya.
- **Daftar peristiwa wajib audit beserta field yang harus tercatat** — pemilik: Compliance
  (`D-56`). Ini kontrak yang menggantikan gerbang 1 bagi `S-5`; tanpanya `S-5` tidak dapat lulus
  gerbang apa pun.
- Apakah jejak audit perlu dapat dibaca pengguna bisnis lewat layar, atau cukup tersedia untuk
  audit? Pemilik: Work Owner.

### 0027 — Luluskan setiap modul lewat dua gerbang: uji kesetaraan otomatis, lalu UAT bisnis

- **Apakah Pega staging yang dapat ditembak dari luar tersedia?** Pemilik: Tim Pega + Tim Infra.
  Ini penghalang utama `S-8`.
- Kapan waktu pengguna bisnis untuk UAT dialokasikan resmi? Pemilik: manajemen (`R-15`).
- Apakah salinan data produksi disegarkan berkala, dan berapa lama disimpan? Pemilik: Work Owner
  + Infra (bertaut dengan ADR-0029).

### 0028 — Ukur modul tanpa baseline Pega dengan kontrak fungsional, dan cabut `BRD §21.4` untuk empat modul

- **Usulan revisi `BRD §21.4`** agar sesuai keputusan ini — menunggu persetujuan Work Owner.
  Diajukan di Fase 6, tidak dikerjakan diam-diam.
- Kapan kontrak API HCC/HCQ tersedia? Pemilik: Tim HCC/HCQ (ADR-0024).
- Kapan daftar peristiwa wajib audit tersedia? Pemilik: Compliance (ADR-0026).
- Baris master komite untuk PA dan Travel di atas Rp 200.000.000 — memang tidak ada, atau belum
  diisi? Pemilik: Work Owner (ADR-0014).

### 0029 — Salin data nasabah apa adanya ke staging; samarkan email dan jangan pernah tulis data nasabah di dokumen

- **Siapa yang menyetujui akses staging, berapa lama salinan disimpan, dan bagaimana prosedur
  pemusnahannya?** Pemilik: Tim Infra/Security + Compliance — pihak yang sama dengan `D-40`.
- Apakah salinan data disegarkan berkala, dan apakah setiap penyegaran menuntut persetujuan ulang?
  Pemilik: Work Owner.
- Apakah data medis di staging perlu perlakuan lebih ketat daripada data nasabah lainnya?
  Pemilik: Compliance.

### 0030 — Satu aplikasi, empat portal, satu database per entitas

1. **Berapa portal sebenarnya?** Work Owner menyebut *minimal* 4. Jumlah pastinya **BELUM
   DIPUTUSKAN — pertanyaan terbuka** (pemilik: **Work Owner**).
2. **Nomor klaim `PNCN.YY.xxxx` (`D-71`) memakai sequence per database.** Dua portal dapat
   menerbitkan nomor yang sama. Perlukah penanda portal di dalam nomornya? (**Work Owner**)
3. **Aturan bisnis syariah berbeda sejauh apa?** `SpreadingSyariah_Act` membuktikan spreading-nya
   berbeda; apakah estimasi, komite, dan penyelesaian juga? (**Work Owner + Tim Pega**)
4. ~~Satu penyedia identitas untuk keempat entitas?~~ — **TERTUTUP `D-78`: login sama untuk semua
   entitas.** Menyisakan pertanyaan baru: **di mana kewenangan portal disimpan**, karena data itu
   lintas portal secara alamiah dan tidak dapat tinggal di masing-masing database (**Work Owner +
   Keamanan Informasi**)
5. **Mata uang per portal** — Timor-Leste bukan Rupiah. Kurs, pembulatan, dan format angka
   mengikuti portal? (**Work Owner**)

