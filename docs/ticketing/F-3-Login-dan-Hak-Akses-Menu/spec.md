# F-3 — Identitas & Akses

| | |
|---|---|
| **Modul** | `F-3` Identitas & Akses |
| **Gelombang** | 1 — Fondasi |
| **Ukuran** | **Besar — tanpa baseline Pega** |
| **Bergantung pada** | `F-1`, `F-2` |
| **Kesiapan** | **TERHALANG untuk identitas · PENUH untuk middleware otorisasi per endpoint** |
| **Cakupan tiket** | penuh untuk bagian yang buktinya lengkap; `needs-info` untuk bagian terhalang (`D-41` Opsi 1) |

## Apa yang dibangun

Cara pengguna masuk, cara aplikasi mengenalinya, dan cara aplikasi memutuskan ia boleh melakukan
apa.

## Kenapa modul ini paling awal menggigit

`F-3` adalah **prasyarat login seluruh aplikasi**. Tidak ada modul yang dapat dirilis ke pengguna
tanpanya — dan justru di modul ini penghalangnya paling keras: **kontrak API HCC/HCQ tidak ada**.

`HCC` dan `HCQ` muncul **2× di seluruh export**, dan keduanya teks pesan galat yang menyuruh
pengguna menghubungi helpdesk. Tidak ada Connect REST, tidak ada pemetaan field respons, tidak ada
penanganan kegagalan. Integrasi ini **greenfield sepenuhnya**, dan `F-3` **tidak punya baseline
untuk diuji kesetaraannya** (`D-56`).

## Otorisasi: yang ini justru jelas

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

## Penghalang modul ini

| Jenis | Isi | Pemilik | Menahan |
|---|---|---|---|
| **Artefak** | **Kontrak API HCC/HCQ** — nol jejak di export | Tim HCC/HCQ | `TKT-F3-002` |
| **Artefak** | **Tidak ada folder Access Group / Role / Privilege / Operator sama sekali** di export | Tim Pega | `TKT-F3-004` sebagian |
| **Artefak** | **5 When rule peran hilang**: `IsGCNMReport`, `IsKomite`, `IsNotViewClaim`, `IsPNCBonding`, `IsSurvey` — masing-masing mengendalikan satu item menu | Tim Pega | `TKT-F3-004` |
| **Artefak** | **Penugasan operator ke peran tidak ada di database.** `POOLDATA.T_ACCESS_GROUP_PNC` hanya memetakan `OPERATOR_ID` → `OLD_OPERATOR_ID` | DBA + Work Owner | `TKT-F3-004` pengisian |
| **Keputusan** | Pemetaan 22 access group → 51 menu masih akurat? · `T_ACCESS_GROUP_PNC` sebenarnya untuk apa? · pernah ada temuan audit soal 901 dari 902 activity tanpa privilege? | Work Owner | `TKT-F3-004` |

## Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-F3-001](issues/01-seam-identity-dan-adapter-fake.md) | Seam Identity dan adapter fake untuk pengembangan | `ready-for-human` | siap |
| [TKT-F3-002](issues/02-adapter-hcc-hcq.md) | Adapter autentikasi HCC/HCQ | `needs-info` | **terhalang artefak** |
| [TKT-F3-003](issues/03-sesi-dan-token-milik-aplikasi.md) | Sesi dan token milik aplikasi | `ready-for-human` | siap |
| [TKT-F3-004](issues/04-tabel-peran-dan-izin-menu.md) | Tabel 22 peran dan 51 izin menu | `needs-info` | **terhalang artefak** |
| [TKT-F3-005](issues/05-middleware-otorisasi-per-endpoint.md) | Middleware otorisasi di setiap endpoint | `ready-for-human` | siap — bagian **PENUH** modul ini |
