# B-12 · Salvage, Lelang & Recovery

| | |
|---|---|
| **Nama di sistem lama** | **Salvage** — harness `InboxSalvage`, `InboxRequestSalvage`, `MasterRecovery`; activity `SetStsSalvagePNC_act`, `Insert_salvageToGAByService`, `SetEmailKomiteSalvage`; procedure `INSERT_SALVAGE`, `INSERT_SALVAGE_DETAILS`, `ADD_NEWMASTERVIRTUALACCOUNT` |
| **Kode modul** | `B-12` |
| **Gelombang** | 5 — Nilai dan pihak luar |
| **Ukuran** | 26 activity |
| **Bergantung pada** | `B-5` Input Estimasi |
| **Kesiapan** | **SEBAGIAN** — **lepas dari `BRD §21.4`** (`D-55`) |

## Apa yang dikerjakan modul ini

Mencatat **barang sisa** dari kerugian yang masih bernilai jual, menjualnya (termasuk lewat balai
lelang), dan mencatat **pemulihan dana** dari pihak ketiga — sebagian lewat **Virtual Account**.

Salvage memengaruhi nilai klaim dua arah: ia **mengurangi nilai bersih** klaim, dan ia
**memulihkan kapasitas pertanggungan** pada perhitungan Sisa TSI (`TKT-B03-003`).

## Cacat yang ditutup modul ini

| Cacat | Bukti | Keputusan |
|---|---|---|
| `INSERT_SALVAGE` menulis **`IDSALVAGE = NULL` saat update** | `Database/INSERT_SALVAGE.prc:47` | **PERBAIKI** — butir 9 `P-5` (`D-49`) |
| `NilaiSalvage` hanya ditambahkan bila baris terakhir kebetulan bertipe salvage | `ValidasiSisaTSI:817` versus `:1489` | **PERBAIKI** — butir 8 `P-5` |

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | `SET_ATTACHFILETEMPSALVAGE` — salah satu dari dua objek yang belum diterima dari 64 | **DBA** |
| **Artefak** | **Hitungan baris rusak**: `SELECT COUNT(*) … WHERE IDSALVAGE IS NULL` | **DBA** |
| **Keputusan** | `tterjual='2'` → `STSTRANSFER='5'`: **satu item terjual mengubah status seluruh salvage induk** — benar? | **Work Owner** |
| **Keputusan** | `SISAKLAIM` **tidak dihitung**, diterima apa adanya — benar? | **Work Owner** |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B12-001](issues/01-pencatatan-salvage-dan-item-lelang.md) | Pencatatan salvage dan item lelang | `needs-info` |
| [TKT-B12-002](issues/02-recovery-dan-virtual-account.md) | Recovery dan Virtual Account | `needs-info` |
