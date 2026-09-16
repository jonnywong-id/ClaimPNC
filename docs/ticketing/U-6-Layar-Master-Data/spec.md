# U-6 · Layar Master Data

| | |
|---|---|
| **Nama di sistem lama** | **Layar master** — `MasterRekening`, `MasterSupplier`, `MasterRecovery`, `MasterPanel_HE`, `MasterLoginSurvey`, `MasterProteksiVisibilityData`, `PNC_MasterTolakKlaim`, `DetailMasterXOL`, `DetailMasterPasalRejected`, `DetailCauseOfLoss`, `DetailDominanFactor`, `GCNMCatSparepart`, `GCNMMasterSparepartType`, `GroupingSparePart_HE`, `SparePart_HE`, `BengkelHE`, `BrowseMasterDocumentTravel_Harness`, dan seterusnya |
| **Kode modul** | `U-6` |
| **Gelombang** | 7 — Sisa |
| **Ukuran** | **Besar** — naik dari Sedang setelah verifikasi |
| **Bergantung pada** | `U-2` Komponen Layar Baku · `F-4` Master Data & Parameter Bisnis |
| **Kesiapan** | **TERHALANG** |

## Apa yang dikerjakan modul ini

Layar tempat administrator **mengelola data acuan** yang dipakai seluruh sistem: rekening,
supplier, bengkel, sparepart, panel, sebab kerugian, pasal penolakan, XOL, dan seterusnya.

## Temuan yang mengubah rencana

| Anggapan | Terverifikasi |
|---|---|
| Master data berjumlah 14 kelompok | **≥29 kelompok** — `U-6` naik dari **Sedang menjadi Besar**, dan `F-4` ikut naik |

## Kenapa modul ini TERHALANG

Karena **jumlah kelompok master masih dinyatakan sebagai "≥29"**, bukan sebagai angka pasti. Modul
yang tidak tahu berapa layar yang harus dibuatnya tidak dapat dinyatakan siap.

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | **Berapa persisnya kelompok master?** Angka yang ada baru batas bawah (**≥29**) | **Work Owner** |
| **Keputusan** | Siapa berwenang mengubah tiap kelompok master — kontrol sekarang **berbasis menu**, bukan berbutir aksi | **Work Owner** (`D-59`) |
| **Keputusan** | Pembagian 48 harness non-Inbox ke `U-4`/`U-5`/`U-6` per berkas | **Work Owner + Lead Engineer** |
| **Bergantung** | Penghapusan lunak dan riwayat perubahan master (`F-4`, `S-5`) | — (`D-…` §8.1 `09-DATABASE`) |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-U6-001](issues/01-layar-master-baku.md) | Layar master baku (CRUD seragam) | `needs-info` |
| [TKT-U6-002](issues/02-kelompok-master-dan-kewenangannya.md) | Kelompok master dan kewenangan pengubahnya | `needs-info` |

> **Daftar lengkap 74 harness beserta kelasnya:** [`../INVENTARIS-HARNESS.md`](../INVENTARIS-HARNESS.md)
> — pembagian modulnya **belum diputuskan** (`D-73`).
