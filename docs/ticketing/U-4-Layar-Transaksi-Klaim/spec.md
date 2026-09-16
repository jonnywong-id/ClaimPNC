# U-4 · Layar Transaksi Klaim

| | |
|---|---|
| **Nama di sistem lama** | **Layar transaksi** — `ViewPolis`, `ReceiveDoucument_Harness`, `InputProgress`, `InputProtection_Harness`, `RCL_Harness`, `RCLPUCL_Harness`, `ProgressClaim_Harness`, `View_DetailKlaimCabang_Harness`, `ViewDetailHasilSurveyorInternal1`, `PNCSearchKlaim`, `AutoKlaim` |
| **Kode modul** | `U-4` |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | **Besar** |
| **Bergantung pada** | `U-2` Komponen Layar Baku · seluruh modul `B-*` |
| **Kesiapan** | **SEBAGIAN** |

## Apa yang dikerjakan modul ini

Layar tempat petugas benar-benar **mengerjakan klaim** — mengikuti tahapan yang sama dengan
`Flow/Register_Flow.xml`:

**Input Register** → **View Polis** → **Input Estimasi** → **Choose Surveyor** →
**Send To Analis / Send To PIC Teknik** → **RCL/PUCL** → **Compliance** → **Investigator** →
**Analyst Doctor / RCL Dokter**

Setiap tahap punya layarnya, dan `U-4` adalah kumpulan layar itu.

## Ukuran yang terverifikasi

| | |
|---|---|
| **Harness** | 74 seluruhnya; **26** bernama Inbox (milik `U-3`), sisanya terbagi antara `U-4`, `U-5`, dan `U-6` |
| **Section** | **269**, **268 di antaranya bergrid** |
| **Flow Action** | **29** |

> **BELUM DIPUTUSKAN — pertanyaan terbuka:** pembagian 48 harness non-Inbox ke `U-4`, `U-5`, dan
> `U-6` belum ditetapkan per berkas. Tanpa itu, jumlah layar yang dijanjikan tiap modul frontend
> adalah perkiraan, bukan komitmen. Pemilik: **Work Owner + Lead Engineer**.

## Hubungannya dengan modul bisnis

`U-4` **tidak memuat aturan bisnis**. Setiap layarnya adalah permukaan dari modul `B-*`
yang bersangkutan — `B-1` View Polis, `B-2` Input Register, `B-5` Input Estimasi, `B-8` Choose
Surveyor, `B-11` RCL/PUCL & Compliance, `B-14` Input Receive Document. Bila aturannya berada di
layar, ia dapat dipintas lewat pemanggilan langsung (`ADR-0023`).

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | Pembagian 48 harness non-Inbox ke `U-4`/`U-5`/`U-6` per berkas | **Work Owner + Lead Engineer** |
| **Bergantung** | Seluruh modul `B-*` yang menjadi isinya — sebagian masih TERHALANG (`B-5`, `B-7`, `B-11`) | — |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-U4-001](issues/01-kerangka-layar-transaksi-per-tahap.md) | Kerangka layar transaksi per tahap klaim | `needs-info` |
| [TKT-U4-002](issues/02-layar-pencarian-dan-detail-klaim.md) | Layar pencarian dan detail klaim | `needs-info` |

> **Daftar lengkap 74 harness beserta kelasnya:** [`../INVENTARIS-HARNESS.md`](../INVENTARIS-HARNESS.md)
> — pembagian modulnya **belum diputuskan** (`D-73`).
