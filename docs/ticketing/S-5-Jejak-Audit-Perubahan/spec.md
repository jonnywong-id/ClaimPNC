# S-5 — Jejak Audit

| | |
|---|---|
| **Modul** | `S-5` Jejak Audit |
| **Gelombang** | 2 |
| **Ukuran** | Sedang — **kemampuan baru 100%, tanpa baseline Pega** |
| **Bergantung pada** | `F-2` |
| **Kesiapan** | **SEBAGIAN** — lingkup jelas, daftar peristiwa `needs-info` |
| **Cakupan tiket** | **penuh** (`D-41` Opsi 1) |

## Apa yang dibangun

Pencatatan permanen setiap perubahan bernilai bisnis: **siapa, kapan, nilai sebelum, nilai
sesudah** — bersifat append-only dan tidak dapat diubah oleh jalur aplikasi mana pun.

## Kenapa modul ini naik derajat

Dua hal terjadi bersamaan:

1. **Sistem lama tidak punya jejak audit atas nilai uang klaim sama sekali** (`T-14`). Yang paling
   mendekati, `Database/PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc:7`, hanya mencatat **empat kolom**.
   Dan dua tabel yang namanya log **terbukti dimutasi** — `UPDATE` pada `claim_service_log`,
   `DELETE` pada `JSON_KLAIM_LOG`.
2. **`D-59` menghapus pemisahan tugas.** Satuan izin adalah menu; seorang pengguna yang memiliki
   tiga menu dapat membuat, menyetujui, dan membayarkan satu klaim. Tidak ada kontrol teknis yang
   mencegahnya.

Akibatnya `S-5` bukan modul pendukung, melainkan **satu-satunya kontrol pengimbang yang tersisa**.

## Konsekuensi: tidak ada yang bisa dibandingkan

Karena tidak ada baseline, `S-5` **tidak dapat melewati gerbang 1** dalam bentuk uji kesetaraan.
`D-56` menggantinya dengan **uji fungsional terhadap kontrak** — dan kontrak itu adalah **daftar
peristiwa wajib audit dari Compliance**, yang **belum ada**.

## Yang mengikat modul ini

| Sumber | Isi |
|---|---|
| `ADR-0026` | Append-only, ditegakkan hak akses database · retensi mengikuti retensi data klaim |
| `ADR-0012` | Soft delete menyeluruh — tidak ada penghapusan fisik |
| `ADR-0023` | Jejak audit satu-satunya kontrol pengimbang |
| `ADR-0028` | Gerbang 1 diganti uji fungsional terhadap kontrak |
| `D-62` | Retensi mengikuti retensi data klaim; **angkanya belum ada**, dibuat sebagai parameter |

## Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-S5-001](issues/01-skema-jejak-audit-append-only.md) | Skema jejak audit append-only dan hak akses database | `ready-for-human` | siap |
| [TKT-S5-002](issues/02-pencatatan-otomatis-perubahan.md) | Pencatatan otomatis pada perubahan bernilai bisnis | `ready-for-human` | siap |
| [TKT-S5-003](issues/03-daftar-peristiwa-wajib-audit.md) | Daftar peristiwa wajib audit | `needs-info` | **terhalang Compliance** |
| [TKT-S5-004](issues/04-retensi-sebagai-parameter.md) | Retensi dan arsip sebagai parameter konfigurasi | `ready-for-human` | siap |
