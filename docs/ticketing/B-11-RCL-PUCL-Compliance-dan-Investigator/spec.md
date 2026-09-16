# B-11 · RCL, PUCL, Compliance, Investigator & Analyst Doctor

| | |
|---|---|
| **Nama di sistem lama** | tahap **RCL/PUCL**, **Compliance**, **Investigator**, **Analyst Doctor**, **RCLDokter** pada `Flow/Register_Flow.xml`; harness `RCL_Harness`, `RCLPUCL_Harness`, `inboxCompliance_Harness`, `InboxInvestigator_Harness`, `inboxAnalystDoctor_Harness`, `PNC_MasterTolakKlaim` |
| **Kode modul** | `B-11` |
| **Gelombang** | 4 — Persetujuan |
| **Ukuran** | 6 activity + jalur alur |
| **Bergantung pada** | `B-6` Penugasan |
| **Kesiapan** | **TERHALANG** |

## Apa yang dikerjakan modul ini

Lima jalur penanganan khusus yang semuanya **keluar dari alur normal**:

| Jalur | Isi |
|---|---|
| **RCL** — Rejected Klaim | penolakan klaim |
| **PUCL** — Proses Ulang Klaim | klaim diproses ulang setelah ditolak atau ditutup |
| **Compliance** | pemeriksaan kepatuhan sebelum klaim diselesaikan |
| **Investigator** | penyelidikan klaim yang mencurigakan |
| **Analyst Doctor / RCL Dokter** | penilaian dan penolakan yang memerlukan pertimbangan medis |

Ketiganya yang pertama memakai **Workbasket** (antrean bersama); dua terakhir memakai **Worklist**.

## Kenapa modul ini TERHALANG

Modul inilah yang paling banyak bergantung pada **lompatan lateral** — dan justru di situ artefaknya
paling banyak hilang.

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | Ticket rule `SendtoPUCL` · `RCLDokter` | **Tim Pega** (`R-16`) |
| **Artefak** | Router `RouterRCLDokter` | **Tim Pega** (`R-04`) |
| **Artefak** | When rule `ElseRCLMSIG` · `IsNotViewClaim` | **Tim Pega** (`R-16`) |
| **Artefak** | Workbasket `RCLPUCL` dan `CompliancePNC` | **Tim Pega** |
| **Keputusan** | **Siapa yang boleh memicu transisi lateral?** Sistem lama **nol pagar izin** | **Work Owner** |
| **ADR `Proposed`** | `ADR-0021` — 14 dari 17 Ticket rule tanpa pemicu yang diketahui | Tim Pega → Work Owner |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B11-001](issues/01-penolakan-rcl-dan-proses-ulang-pucl.md) | Penolakan (RCL) dan proses ulang (PUCL) | `needs-info` |
| [TKT-B11-002](issues/02-compliance-investigator-analyst-doctor.md) | Compliance, Investigator, dan Analyst Doctor | `needs-info` |
