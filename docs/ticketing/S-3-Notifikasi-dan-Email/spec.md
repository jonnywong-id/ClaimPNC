# S-3 · Notifikasi & Korespondensi Email

| | |
|---|---|
| **Nama di sistem lama** | **Notifikasi / Correspondence** — `SendEmailNotification` (**15 pemanggil**), template HTML, rule Correspondence; `Data Transform/SetDataEmail-DT.xml` |
| **Kode modul** | `S-3` |
| **Gelombang** | 5 — Nilai dan pihak luar |
| **Ukuran** | 14 activity |
| **Bergantung pada** | `F-4` Master Data |
| **Kesiapan** | **TERHALANG** |

## Apa yang dikerjakan modul ini

Mengirim pemberitahuan berdasarkan **peristiwa domain** — bukan berdasarkan perintah "kirim email
ke alamat ini". Pemanggil menyatakan **apa yang terjadi**; modul ini yang menentukan **siapa yang
perlu tahu**, dari master data.

## Kenapa modul ini TERHALANG

Tiga hal sekaligus: **activity pengirimnya hilang dari export**, **isi templatenya tidak diketahui**,
dan **kredensial SMTP-nya belum punya tempat**.

## Yang tidak diketahui tentang modul ini

- **Tidak ada satu pun direktori `Correspondence/` di export.** Seluruh rule korespondensi —
  yakni isi surat dan emailnya — **tidak tersedia**.
- Jumlah rule Correspondence **belum pasti**: `docs/Steering/06-MODULE-BREAKDOWN.md:62` menyebut
  **5**, `docs/verifikasi-bukti-adr.md:2783` menyebut **3**. Keduanya adalah hitungan dari
  *rujukan* di rule lain, bukan dari direktorinya. **BELUM DIPUTUSKAN — pertanyaan terbuka**;
  angka sebenarnya hanya dapat dipastikan **Tim Pega**.

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **`SendEmailNotification` — 15 pemanggil**, tidak ada di export (`19-GAP-EXPORT-DETAIL.md:196`) | **Tim Pega** (`R-07`) |
| **Artefak** | **Seluruh rule Correspondence dan template HTML** — direktorinya tidak ada di export | **Tim Pega** (`R-16`) |
| **Keputusan** | **Rotasi 3 password SMTP (31 lokasi, plaintext)** dan tujuan penyimpanannya; `UseSSL=false` pada seluruh 14 kemunculan sementara 16 dari 31 lokasi memakai port 587 | **Tim Infra/Security** (`D-40`, `R-17`) |
| **Keputusan** | Pemetaan penerima personal ke mailbox fungsional pengganti | **Work Owner** (`D-67`) |

**Sudah diputuskan:** **tidak ada akun pribadi** sebagai penerima; seluruhnya mailbox fungsional
dari master data (`D-67`).

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-S3-001](issues/01-seam-notifier-berbasis-peristiwa.md) | Seam Notifier berbasis peristiwa domain | `needs-info` |
| [TKT-S3-002](issues/02-template-dan-pengiriman-email.md) | Template, penerima, dan pengiriman email | `needs-info` |
