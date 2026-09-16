# S-4 · Integrasi Sistem Luar

| | |
|---|---|
| **Nama di sistem lama** | **Connect REST** (keluar) dan **Service REST** (masuk) · 6 DB Link (64 pemakaian) |
| **Kode modul** | `S-4` |
| **Gelombang** | 5 — Nilai dan pihak luar |
| **Ukuran** | 7 activity · **21 Connect REST** · 4 Service REST · 6 API pengganti DB Link |
| **Bergantung pada** | `F-1` Kerangka Aplikasi |
| **Kesiapan** | **TERHALANG** |

## Apa yang dikerjakan modul ini

Seluruh percakapan Claim PNC dengan sistem di luar dirinya — **dua arah**:

- **Keluar**: mengambil data premi, mengirim data bayar ke kasir, mengunggah dokumen, mengirim
  klaim ke ASO, menutup klaim Non-MBU, menetapkan DLA.
- **Masuk**: empat layanan yang **dipanggil sistem lain**, dua di antaranya **menerima persetujuan
  komite**.

## Temuan yang mengubah rencana

| Anggapan | Terverifikasi |
|---|---|
| 12 Connect REST keluar | **21 berkas** di `Connect REST/` — 12 yang terhitung sejak awal ditambah **9 yang baru ditemukan** |
| Integrasi hanya ke arah keluar | **4 Service REST masuk** — permukaan yang **belum pernah diaudit sama sekali** |
| Integrasi memakai autentikasi | **18 dari 21** Connect REST ber-`pyUseAuthentication=false` |

> **Angka 12 sudah dikoreksi menjadi 21** di `BRD §FR-S4`, `06-MODULE-BREAKDOWN.md:63`, dan
> `16-RISK-ANALYSIS.md:193` — disetujui Work Owner 2026-09-14 (`D-73`, `21-RIWAYAT-REVISI.md` §7).
> **Sembilan yang baru ditemukan belum dianalisis setara** dengan 12 yang lama.

## Kenapa modul ini TERHALANG

Yang menahan bukan kerumitan teknisnya, melainkan **tidak diketahuinya siapa lawan bicara yang
sah** — pada permukaan masuk maupun keluar.

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **6 API pengganti DB Link belum ada kontraknya** (64 pemakaian DB Link) | **Tim Infra + tim sistem lawan** (`D-25`, `R-03`) |
| **Keputusan** | **4 Service REST masuk — masuk lingkup migrasi?** Dua menerima **persetujuan komite** dari sistem lain | **Work Owner** |
| **Keputusan** | **18 dari 21 Connect REST tanpa autentikasi** — dibiarkan atau diperbaiki? | **Work Owner + Tim Infra/Security** (`R-18`) |
| **Keputusan** | `GetPremiumPaid_SPK` memakai **`http://` ke IP:port tanpa TLS** untuk data premi | **Tim Infra/Security** (`R-18`) |
| **Keputusan** | Endpoint BRI menunjuk **sandbox** — apakah itu yang berjalan di produksi? | **Work Owner + tim integrasi** (`R-18`) |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-S4-001](issues/01-klien-rest-keluar.md) | Klien REST keluar dan penanganan kegagalannya | `needs-info` |
| [TKT-S4-002](issues/02-layanan-rest-masuk.md) | Empat layanan REST masuk dan otentikasinya | `needs-info` |
| [TKT-S4-003](issues/03-pengganti-db-link.md) | Enam API pengganti DB Link | `needs-info` |
