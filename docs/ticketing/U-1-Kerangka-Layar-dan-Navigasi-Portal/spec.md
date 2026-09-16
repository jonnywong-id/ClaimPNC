# U-1 — Kerangka SPA

| | |
|---|---|
| **Modul** | `U-1` Kerangka SPA |
| **Gelombang** | 2 — Kerangka UI |
| **Ukuran** | Sedang |
| **Bergantung pada** | `F-1` (penyajian SPA), `F-3` (autentikasi) |
| **Kesiapan** | **SEBAGIAN** — kerangka siap, peta rute terhalang artefak |
| **Cakupan tiket** | penuh untuk bagian yang buktinya lengkap; `needs-info` untuk peta rute (`D-41` Opsi 1) |

## Apa yang dibangun

Rangka aplikasi satu halaman: routing, tata letak, alur masuk, state global, dan penanganan galat
di sisi peramban.

## Yang mengikat modul ini

| Sumber | Isi |
|---|---|
| `ADR-0002` | React + TypeScript + Vite, SPA murni disajikan binary Go — tanpa runtime Node.js di produksi |
| `D-13` | Alur kerja, urutan langkah, penempatan field, dan tata letak **mengikuti Pega** |
| `ADR-0024` | Autentikasi lewat HCC/HCQ — kontraknya belum ada |
| `D-59` | Menu yang terlihat mengikuti izin peran; penyembunyian menu **bukan** pengamanan |

## Penghalang modul ini

| Jenis | Isi | Pemilik | Menahan |
|---|---|---|---|
| **Artefak** | **7 harness portal hilang dari export**: `InboxServiceCenter` · `InboxCloseClaim_Harness` · `InboxRequestSalvage` · `PNCViewClaim` · `ReportProduksiPA_harnes` · `InboxOutstanding_Harness` · `LostAdjuster_harness` | Tim Pega (`R-16`) | `TKT-U1-004` |
| **Keputusan** | **Ketujuh layar itu masih aktif di produksi?** Bila tidak, 7 rute SPA dihapus dari lingkup | Work Owner | `TKT-U1-004` |
| **Keputusan** | Dari **38 harness tanpa entri menu**, mana yang rute berdiri sendiri dan mana yang modal? `pyHarnessPurpose` **tidak ada** di export | Work Owner + Tim Pega | `TKT-U1-004` |
| **Artefak** | Kontrak API HCC/HCQ | Tim HCC/HCQ | `TKT-U1-002` sebagian |

## Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-U1-001](issues/01-kerangka-spa-routing-dan-layout.md) | Kerangka SPA: routing, tata letak, dan state global | `ready-for-human` | siap |
| [TKT-U1-002](issues/02-alur-masuk-dan-sesi-di-frontend.md) | Alur masuk dan penanganan sesi di frontend | `ready-for-human` | siap |
| [TKT-U1-003](issues/03-penanganan-galat-dan-notifikasi-global.md) | Penanganan galat dan notifikasi global | `ready-for-human` | siap |
| [TKT-U1-004](issues/04-peta-rute-dari-74-harness.md) | Peta rute dari 74 harness dan 51 item menu | `needs-info` | **terhalang artefak** |
