# B-3 · Objek Pertanggungan & Coverage

| | |
|---|---|
| **Nama di sistem lama** | bagian dari layar **Input Register** — section `ObjectCoverage` (54 pemakaian), `ObjectItemList` (32), `LocationList`, `t_anekalist` |
| **Kode modul** | `B-3` |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | 44 activity |
| **Bergantung pada** | `B-2` Input Register |
| **Kesiapan** | **SEBAGIAN** |

## Apa yang dikerjakan modul ini

Mencatat **apa yang tertimpa** dan **jaminan mana yang dipakai**. Satu klaim dapat memuat banyak
Objek Pertanggungan; satu objek memuat banyak Coverage; satu coverage memuat rincian item.

Bentuk objeknya **berbeda per lini bisnis** — dan perbedaan itulah inti modul ini:

| Lini | Atribut khas objek |
|---|---|
| Kendaraan | nomor rangka, nomor mesin, merek, tipe, model |
| Personal Accident | tanggal lahir, NIK, status peserta |
| Asuransi Kredit | kolektibilitas, sebab macet, hari tunggakan, suku bunga |
| Properti / Aneka | lokasi risiko, okupasi |

## Rule Pega yang digantikan

| Jenis | Nama |
|---|---|
| Section | `ObjectCoverage` (54×) · `ObjectItemList` (32×) · `LocationList` |
| Activity | 44 activity objek dan coverage |
| Objek DB | `t_anekalist` dan tabel objek per lini |

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **When rule klasifikasi produk Aneka hilang**: `IsARBusiness` (24 pemakaian), `IsAnekaPerYear` (16), `IsMBBusiness` (16), `IsMoneyInsurance` (15), `IsElectronicEquipment` (10), dan lainnya | **Tim Pega** (`R-16`) |
| **Keputusan** | Pemetaan Group Panel `003` **berkonflik**: `LocationList` versus `t_anekalist` — mana yang benar? | **Work Owner** |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B03-001](issues/01-objek-pertanggungan-per-lini-bisnis.md) | Objek Pertanggungan per lini bisnis | `needs-info` |
| [TKT-B03-002](issues/02-coverage-penyebab-kerugian-dan-rincian-item.md) | Coverage, penyebab kerugian, dan rincian item | `needs-info` |
| [TKT-B03-003](issues/03-validasi-tsi-dan-sisa-tsi.md) | Validasi TSI dan Sisa TSI | `needs-info` |
