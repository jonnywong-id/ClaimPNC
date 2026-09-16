# B-5 · Input Estimasi & Penyelesaian Nilai Klaim

| | |
|---|---|
| **Nama di sistem lama** | **Input Estimasi** / **Estimation** — tahap `Estimation` dan `Input Estimasi` pada `Flow/Register_Flow.xml`; `AdjustmentList` di data |
| **Kode modul** | `B-5` |
| **Gelombang** | 5 — Nilai dan pihak luar |
| **Ukuran** | 29 activity |
| **Bergantung pada** | `B-3` Objek & Coverage · `B-4` Spreading |
| **Kesiapan** | **TERHALANG** — satu-satunya modul yang **masih terikat `BRD §21.4`** |

## Apa yang dikerjakan modul ini

Menjalankan perjalanan nilai uang klaim, dari perkiraan sampai dibayar:

```
Estimasi  →  Usulan (Propose)  →  Akseptasi (Accepted)  →  Dibayar (Paid)
```

Setiap baris nilai melekat pada satu **Coverage**, dan di sistem baru bernama **Settlement Line**
— menggantikan istilah lama `AdjustmentList`, yang menyesatkan karena dalam praktik asuransi
"adjusting" berarti proses penilaian kerugian, bukan nilai penyelesaian.

Termasuk di dalamnya: **salvage**, **risiko sendiri**, dan **fee adjuster**.

## Kenapa modul ini paling sensitif

Di sinilah angka yang dibayarkan ditetapkan. Tiga dari 13 perbaikan eksplisit `P-5` berada di
modul ini: basis kurs, perilaku saat kurs tidak ditemukan, dan perhitungan salvage pada sisa TSI.

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **isi `POOLDATA.GCNM_FEE_SCALE` (17 pita)** | **DBA** (`R-19`) |
| **Artefak** | **isi `m_currencystandard`** | **DBA** (`R-19`) |
| **Artefak** | `BrowseT_Claim_Adjustment_SQL` | **DBA** |
| **Keputusan** | Berapa desimal pembulatan fee? | **Work Owner** |
| **Keputusan** | Rp 1.650.000 dan 2% masih berlaku? | **Work Owner** |
| **Keputusan** | Salvage memulihkan TSI — apa niat bisnisnya? | **Work Owner** |
| **Keputusan** | `>` atau `>=` pada perbandingan sisa TSI? | **Work Owner** |

**Sudah tidak perlu ditanyakan lagi:** basis kurs adalah **tanggal kejadian**, dan kurs tidak
ditemukan berarti **klaim ditolak** (`D-48`, `ADR-0015`).

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B05-001](issues/01-layar-input-estimasi-dan-settlement-line.md) | Layar Input Estimasi dan Settlement Line | `needs-info` |
| [TKT-B05-002](issues/02-konversi-kurs-dan-perbandingan-ambang.md) | Konversi kurs dan perbandingan terhadap ambang | `needs-info` |
| [TKT-B05-003](issues/03-fee-adjuster-dan-risiko-sendiri.md) | Fee adjuster, risiko sendiri, dan salvage pada nilai | `needs-info` |
