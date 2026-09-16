# S-8 · Perkakas Uji Kesetaraan

| | |
|---|---|
| **Nama di sistem lama** | **tidak ada padanannya** — modul ini **100% baru** |
| **Kode modul** | `S-8` |
| **Gelombang** | **1 — Fondasi** (`D-42`) |
| **Ukuran** | Besar — seluruhnya baru |
| **Bergantung pada** | `F-1`, `F-2` |
| **Kesiapan** | **TERHALANG** |

## Apa yang dikerjakan modul ini

Menjalankan **kasus yang sama** di Pega staging dan di Go staging, membandingkan hasilnya, lalu
**mengklasifikasikan setiap selisih** terhadap 13 butir perbaikan `P-5` (`D-49`, `D-54`).

## Kenapa modul ini ada di gelombang 1, bukan gelombang akhir

`BRD §21.1` menjadikan uji kesetaraan otomatis sebagai **gerbang pertama setiap modul**. Selama
perkakas ini belum ada, **tidak ada satu modul pun yang dapat dinyatakan lulus** — dan Strangler
Fig (`D-05`) berhenti sebelum dimulai (`D-42`).

Artinya `S-8` bukan modul pendukung yang bisa menunggu. **Ia penghalang tunggal bagi seluruh
rencana.**

## Yang membuatnya berbeda dari modul lain

| | |
|---|---|
| **Tidak menyalin apa pun** | Tidak ada rule Pega yang digantikannya. Ia tidak tunduk `P-5` — **ia yang menegakkannya** |
| **Tidak dapat menguji dirinya sendiri** | Kebenarannya dibuktikan dengan selisih yang sengaja dibuat, bukan dengan membandingkan ke Pega |
| **Dua modul tak punya baseline** | `F-3` (HCC/HCQ nol jejak di export) dan `S-5` (sistem lama tidak punya jejak audit nilai) — `BRD §21.2` kriteria #3 **tidak dapat diberlakukan** pada keduanya (`D-42`) |

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | **Ketersediaan Pega staging yang dapat ditembak dari luar** — penghalang utama `S-8` | **Tim Pega + Infra** (`R-14`, `ADR-0027`) |
| **Keputusan** | Siapa boleh menembak Pega, di lingkungan mana, dan **apakah boleh membaca produksi read-only** | **Work Owner + Tim Infra/Security** (`D-42`) |

**Sudah diputuskan:** selisih yang cocok dengan **salah satu dari 13 butir `P-5`** lolos otomatis
dan cukup dicatat; selisih **di luar 13 butir** wajib **persetujuan Work Owner tertulis** sebelum
modul lulus gerbang 1 (`D-54`).

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-S8-001](issues/01-penembak-kasus-dan-perekam-hasil.md) | Penembak kasus dan perekam hasil | `needs-info` |
| [TKT-S8-002](issues/02-pembanding-dan-klasifikasi-selisih.md) | Pembanding hasil dan klasifikasi selisih terhadap 13 butir P-5 | `ready-for-human` |
