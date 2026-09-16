# S-2 · Laporan & Export

| | |
|---|---|
| **Nama di sistem lama** | **Laporan** — 56 Report Definition; harness `Har_LaporanHasilAI`, `ReportKPIHarness`, `PNCTATReport`, `OutstandingKlaimperCabang_Harness`, `MonitoringSLINKOJK`, `PNCStudyClaim` |
| **Kode modul** | `S-2` |
| **Gelombang** | 6 — Laporan |
| **Ukuran** | 78 activity · 56 Report Definition |
| **Bergantung pada** | seluruh modul bisnis |
| **Kesiapan** | **SEBAGIAN** |

## Apa yang dikerjakan modul ini

Membangun **56 laporan** beserta engine pembuat **PDF, Excel, dan CSV** di dalam aplikasi Go
sendiri — tanpa engine reporting Pega dan tanpa tools BI eksternal (`ADR-0011`).

## Temuan yang mengubah rencana

| Anggapan | Terverifikasi |
|---|---|
| Masalahnya `OFFSET` besar | **`OFFSET` nol kemunculan** di seluruh export |
| Laporan memaginasi hasil besar | **`pyMaxRecords=500` pada 54 dari 56 laporan** — hasilnya **dipotong**, bukan dipaginasi |

Artinya kebutuhan export bervolume besar **belum pernah benar-benar dilayani**. Menghapus batas
500 baris adalah **penambahan kemampuan**, bukan penyalinan — dan ia **mengubah angka yang dilihat
pengguna**.

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **12 Report Definition** (`BrowseVPanel_HE_RD` 82 pemakaian) · 10 template HTML · 3 Correspondence | **Tim Pega** (`R-16`) |
| **Keputusan** | **54 dari 56 laporan memotong hasil di 500 baris — pengguna tahu?** Menampilkan seluruh baris **mengubah angka yang mereka lihat** | **Work Owner** |
| **Keputusan** | Berapa baris maksimum yang wajib dilayani satu export? | **Work Owner** (`ADR-0011`) |
| **Keputusan** | Export besar dijalankan serentak, atau diantrekan lalu diberitahukan? | **Work Owner** |
| **Keputusan** | Keluaran PDF wajib identik **visual** dengan Pega, atau cukup identik **isi**? | **Work Owner** |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-S2-001](issues/01-engine-pdf-excel-csv.md) | Engine PDF, Excel, dan CSV | `needs-info` |
| [TKT-S2-002](issues/02-56-laporan-dan-batas-500-baris.md) | 56 laporan dan penghapusan batas 500 baris | `needs-info` |
