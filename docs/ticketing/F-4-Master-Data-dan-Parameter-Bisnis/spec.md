# F-4 — Master Data

| | |
|---|---|
| **Modul** | `F-4` Master Data |
| **Gelombang** | 1 — Fondasi |
| **Ukuran** | **Besar** (naik dari Sedang–Besar) |
| **Bergantung pada** | `F-1`, `F-2`, `F-3` |
| **Kesiapan** | **SEBAGIAN** — kerangka siap, empat master terhalang isi atau keputusan |
| **Cakupan tiket** | penuh untuk bagian yang buktinya lengkap; `needs-info` untuk bagian terhalang (`D-41` Opsi 1) |

## Apa yang dibangun

Seluruh nilai yang di sistem lama tertanam di dalam rule, dipindahkan menjadi **data yang dapat
diubah tanpa deploy**, beserta layar pengelolanya.

## Ukuran sebenarnya

| Anggapan v1.0 | Terverifikasi |
|---|---|
| 14 kelompok master | **≥29 kelompok** |
| 10 alamat email hardcode | **66 unik**, termasuk **≥6 akun Gmail pribadi di jalur produksi** |
| 4 user ID hardcode | **24 unik**, sebagian tertanam **di dalam teks SQL** |
| 3 ambang komite hardcode | **8 ambang komite unik** + 7 ambang uang non-komite |
| 3 hostname | **3 unik, 48 perbandingan** — angkanya benar, daftarnya salah |

Angka lama benar untuk lingkupnya — dua rule saja. Memakainya untuk `FR-F4` akan membuat estimasi
meleset sekitar **enam kali lipat**.

## Yang mengikat modul ini

| Sumber | Isi |
|---|---|
| `ADR-0025` | Seluruh hardcode menjadi master/konfigurasi; rahasia keluar dari kode — **`Proposed`, `D-40` masih OPEN** |
| `ADR-0014` | Ambang komite: tangga ambang bawah per lini, diakumulasi; `LIMIT_TOP` jadi validasi master |
| `ADR-0015` | Kurs pada tanggal kejadian; tanpa kurs klaim **ditolak** |
| `D-67` | **Tidak ada akun pribadi** sebagai penerima notifikasi |
| `D-62` | Retensi sebagai master |

## Penghalang modul ini

| Jenis | Isi | Pemilik | Menahan |
|---|---|---|---|
| **Artefak** | isi **`POOLDATA.GCNM_FEE_SCALE`** (17 pita) | DBA | `B-5`, bukan `F-4` langsung |
| **Artefak** | isi **`m_currencystandard`** | DBA | `TKT-F4-004` |
| **Artefak** | **DDL 9 tabel baru** | DBA | `TKT-F4-001` sebagian |
| **Keputusan** | **Mana dari ≥29 kelompok master yang masuk lingkup?** Baru area Bengkel/Sparepart/Supplier yang diputuskan (dikecualikan, `D-34`) | Work Owner | `TKT-F4-006` |
| **Keputusan** | Struktur `EMAILKOMITE` dipertahankan apa adanya? | Work Owner | `TKT-F4-002` |
| **Keputusan** | Ambang Large Losses ada fiturnya di sistem baru? · batas tanggal per lini bisnis = kebutuhan baru? · alur persetujuan perubahan master berlaku untuk semua master? | Work Owner | `TKT-F4-001`, `TKT-F4-005` |

## Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-F4-001](issues/01-kerangka-master-data-dan-layar-crud.md) | Kerangka master data dan layar CRUD baku | `needs-info` | terhalang keputusan |
| [TKT-F4-002](issues/02-master-ambang-komite.md) | Master Ambang Komite dan validasi integritas tangga | `needs-info` | terhalang keputusan |
| [TKT-F4-003](issues/03-master-penerima-notifikasi.md) | Master Penerima Notifikasi — mailbox fungsional | `ready-for-human` | siap |
| [TKT-F4-004](issues/04-master-mata-uang-dan-kurs.md) | Master Mata Uang dan Kurs per tanggal | `needs-info` | **terhalang artefak** |
| [TKT-F4-005](issues/05-master-status-kalender-dan-ambang.md) | Master Status Klaim, kalender libur, dan ambang uang | `needs-info` | terhalang keputusan |
| [TKT-F4-006](issues/06-inventaris-kelompok-master.md) | Inventaris ≥29 kelompok master dan penetapan lingkup | `needs-info` | terhalang keputusan |
