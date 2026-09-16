# U-5 · Layar Laporan

| | |
|---|---|
| **Nama di sistem lama** | **Layar laporan** — harness `Har_LaporanHasilAI`, `ReportKPIHarness`, `PNCTATReport`, `OutstandingKlaimperCabang_Harness`, `MonitoringSLINKOJK`, `PNCStudyClaim`, `DashboardClaim_Harness` |
| **Kode modul** | `U-5` |
| **Gelombang** | 6 — Laporan |
| **Ukuran** | **Besar** |
| **Bergantung pada** | `U-2` Komponen Layar Baku · `S-2` Laporan & Export · `S-7` Dashboard |
| **Kesiapan** | **SEBAGIAN** |

## Apa yang dikerjakan modul ini

Layar tempat pengguna **memilih laporan, mengisi parameternya, menjalankannya, dan mengunduh
hasilnya**.

`S-2` membangun laporannya; `U-5` adalah cara orang sampai ke laporan itu.

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Bergantung** | Seluruh penghalang `S-2` berlaku di sini — batas 500 baris, kriteria kesamaan keluaran, 12 Report Definition hilang | **Work Owner + Tim Pega** (`R-16`) |
| **Keputusan** | Export besar dijalankan serentak atau diantrekan — menentukan apakah layar butuh daftar unduhan | **Work Owner** |
| **Keputusan** | Pembagian 48 harness non-Inbox ke `U-4`/`U-5`/`U-6` per berkas | **Work Owner + Lead Engineer** |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-U5-001](issues/01-layar-pemilih-laporan-dan-parameter.md) | Layar pemilih laporan dan parameternya | `needs-info` |
| [TKT-U5-002](issues/02-unduhan-dan-antrean-export.md) | Unduhan hasil dan antrean export besar | `needs-info` |

> **Daftar lengkap 74 harness beserta kelasnya:** [`../INVENTARIS-HARNESS.md`](../INVENTARIS-HARNESS.md)
> — pembagian modulnya **belum diputuskan** (`D-73`).
