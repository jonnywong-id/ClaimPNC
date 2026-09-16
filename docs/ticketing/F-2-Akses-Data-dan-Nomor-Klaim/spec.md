# F-2 — Akses Data

| | |
|---|---|
| **Modul** | `F-2` Akses Data |
| **Gelombang** | 1 — Fondasi |
| **Ukuran** | Sedang |
| **Bergantung pada** | `F-1` |
| **Kesiapan** | **SEBAGIAN** — lingkup jelas, dua penghalang menahan dua tiket |
| **Cakupan tiket** | **penuh** (`D-41` Opsi 1) |

## Apa yang dibangun

Seluruh cara aplikasi menyentuh database: koneksi dan pool, seam Repository, disiplin SQL
portabel, kepemilikan transaksi, kerangka migrasi skema, pola soft delete, dan generator nomor
klaim.

**Tidak ada kueri bisnis di modul ini.** Kueri klaim, komite, dan settlement milik modul bisnisnya
masing-masing; `F-2` menyediakan aturan dan perkakasnya.

## Kenapa ini berbahaya bila salah

Selama masa paralel, Pega dan Go menulis ke **database yang sama** (`ADR-0004`). Satu tabel yang
ditulis dua sistem menghasilkan kerusakan data yang hampir mustahil dilacak — dan tabel header
klaim `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` dibaca **116 rule Pega**. `F-2` adalah tempat aturan
pengamannya dipasang.

## Yang mengikat modul ini

| Sumber | Isi |
|---|---|
| `ADR-0004` | Database bersama · penulis tunggal per tabel (`P-1`) · skema backward-compatible (`P-4`) |
| `ADR-0005` | Satu set SQL portabel Oracle + PostgreSQL 17+, tiga pengecualian terkelola |
| `ADR-0007` | Kepemilikan transaksi pindah ke Go; stored procedure tidak dipanggil |
| `ADR-0009` | Nomor klaim `PNCN.YY.xxxx` — satu-satunya sakelar dialek |
| `ADR-0012` | Soft delete menyeluruh |
| `D-63` | Perubahan skema: permintaan tertulis → persetujuan Work Owner → DBA → uji Pega+Go bersamaan |

## Penghalang modul ini

| Jenis | Isi | Pemilik | Menahan |
|---|---|---|---|
| **Artefak** | **DDL seluruh tabel** (`R-08`) — tipe, panjang, index, constraint tidak diketahui | DBA | `TKT-F2-004` sebagian |
| **Keputusan** | **538 pemakaian `{ASIS:}`** — himpunan kolom filter dan sort yang sah belum ditetapkan | Work Owner + Lead Engineer | `TKT-F2-007` |
| Artefak | **64 Connect SQL** dan 12 dependensi procedure | DBA | tidak menahan `F-2`; menahan modul bisnis |
| Catatan | `pyParamArray` kosong → **1.646 step RDB tidak terbaca**; inventaris SQL yang ada adalah **batas bawah** | — | memengaruhi estimasi, bukan lingkup |

## Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-F2-001](issues/01-koneksi-pool-dan-seam-repository.md) | Koneksi, connection pool, dan seam Repository | `ready-for-human` | siap |
| [TKT-F2-002](issues/02-disiplin-sql-portabel.md) | Disiplin SQL portabel dan pemeriksaan pola terlarang | `ready-for-human` | siap |
| [TKT-F2-003](issues/03-kepemilikan-transaksi.md) | Kepemilikan transaksi di lapisan aplikasi | `ready-for-human` | siap |
| [TKT-F2-004](issues/04-kerangka-migrasi-skema.md) | Kerangka migrasi skema backward-compatible dan rollback | `needs-info` | terhalang `R-08` |
| [TKT-F2-005](issues/05-pola-soft-delete.md) | Soft delete sebagai pola akses data | `ready-for-human` | siap |
| [TKT-F2-006](issues/06-generator-nomor-klaim.md) | Generator nomor klaim `PNCN.YY.xxxx` | `needs-info` | terhalang keputusan |
| [TKT-F2-007](issues/07-daftar-putih-filter-dan-sort.md) | Daftar putih kolom filter dan sort pengganti `{ASIS:}` | `needs-info` | terhalang keputusan |
