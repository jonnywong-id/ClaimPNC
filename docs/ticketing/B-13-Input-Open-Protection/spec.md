# B-13 · Input Open Protection — Buka Proteksi

| | |
|---|---|
| **Nama di sistem lama** | **Input Protection** / **Request Protection** — `Flow/CreateProtection_Flow.xml`, harness `InputProtection_Harness`, `InputReqProtection_Harness`, `MasterProteksiVisibilityData` |
| **Kode modul** | `B-13` |
| **Gelombang** | 4 — Persetujuan |
| **Ukuran** | 12 activity |
| **Bergantung pada** | `B-2` Input Register |
| **Kesiapan** | **SEBAGIAN** — **kendala paling sedikit di antara modul bisnis** |

## Apa yang dikerjakan modul ini

Mencatat permintaan **pembukaan proteksi** sebelum atau di luar alur klaim normal, lalu menautkannya
ke klaim dan menandainya terpakai.

## Kenapa modul ini paling siap

Berbeda dari modul bisnis lain, **artefaknya lengkap**:

| Artefak | Status |
|---|---|
| When rule `IsReqProtection` | ✅ **ada** |
| When rule `IsOpenProtectionPNC` | ✅ **ada** |
| Ticket rule `TC_PNCInputProtection` | ✅ **ada** |
| Ticket rule `Akp_PNCInputProtection` | ✅ **ada** |

Keempatnya ada di export — **satu-satunya modul bisnis dengan Ticket rule yang lengkap.**

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | `CreateProtection_Flow` hanya memuat **6 shape** — cakupan sekecil itu benar, atau ada bagian alur yang berada di luar flow? | **Work Owner** |

Tidak ada penghalang artefak.

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B13-001](issues/01-layar-permintaan-buka-proteksi.md) | Layar permintaan buka proteksi | `ready-for-human` |
| [TKT-B13-002](issues/02-penautan-proteksi-ke-klaim.md) | Penautan proteksi ke klaim dan penandaan terpakai | `needs-info` |
