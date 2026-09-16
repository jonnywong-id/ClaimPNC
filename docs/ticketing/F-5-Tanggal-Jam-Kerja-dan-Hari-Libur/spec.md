# F-5 — Waktu & Zona Waktu

| | |
|---|---|
| **Modul** | `F-5` Waktu & Zona Waktu |
| **Gelombang** | 1 — Fondasi |
| **Ukuran** | Kecil |
| **Bergantung pada** | `F-1` |
| **Kesiapan** | **SEBAGIAN** — lingkup jelas (**118 titik di 36 activity**), empat keputusan memengaruhi AC |
| **Cakupan tiket** | **penuh** (`D-41` Opsi 1) |

## Apa yang dibangun

Satu sumber waktu untuk seluruh aplikasi, satu tempat konversi zona waktu, dan satu tempat aturan
hari kalender — menggantikan penyesuaian **7 jam manual** yang tersebar di sistem lama.

## Kenapa modul kecil ini tidak boleh ditunda

`F-5` menghapus penyesuaian 7 jam yang tersebar di 36 activity. Bila dikerjakan belakangan,
**seluruh aturan tanggal yang telanjur ditulis harus ditulis ulang** — dan aturan tanggal adalah
gerbang validasi terberat di registrasi klaim (`B-2`).

Lebih berat lagi: penyesuaian itu **tidak konsisten di dalam satu kondisi validasi**
(`Activity/InputRegister_act-Act.xml:5788` versus `:4805`), sehingga hasil validasi di batas
periode polis berbeda tergantung cabang mana yang dijalankan. Itu butir 3 pada daftar 13 perbaikan
eksplisit `P-5`.

## Yang mengikat modul ini

| Sumber | Isi |
|---|---|
| `D-49` butir 3 | Penyesuaian 7 jam asimetris **diperbaiki** — masuk daftar perbaikan eksplisit `P-5` |
| `ADR-0020` | Dua basis perhitungan TAT dipertahankan; logika jam kerja **ditulis ulang di Go**, bukan dipanggil lewat API |
| `R-12` | Zona waktu bergeser saat migrasi data — berlaku juga pada penyalinan ke staging |
| `D-50` | `GET_WORKING_HOURS` → KPI dan kronologi TAT · `GETSELISIHJAM` → satu layar inbox Compliance · `HRD_LBR` → hari libur |

## Penghalang modul ini

Seluruhnya **keputusan**, bukan artefak — lingkupnya sendiri sudah terukur.

| Pertanyaan | Pemilik | Menahan |
|---|---|---|
| **`addCalendar(…, 12, 0, 0)` di 101 titik pada 4 activity — apa maksudnya?** Menambah 12 jam, atau menetapkan tengah hari? | Work Owner + Tim Pega | `TKT-F5-002` |
| `"WIB"` (25×) versus `Asia/Jakarta` (137×) — hasilnya sama? | Lead Engineer + DBA | `TKT-F5-001` (verifikasi) |
| Adakah data produksi yang **tergeser ganda +14 jam**? | DBA | `TKT-F5-002` |
| Representasi otoritatif untuk `DateOfLoss` dan `ReportDate` — tanggal murni atau waktu penuh? | Work Owner | `TKT-F5-001` |
| Dari mana daftar hari libur diperoleh setiap tahun? | Work Owner | `TKT-F5-003` |

## Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-F5-001](issues/01-seam-clock-dan-penyimpanan-utc.md) | Seam Clock dan penyimpanan UTC | `needs-info` | terhalang keputusan |
| [TKT-F5-002](issues/02-konversi-wib-tunggal.md) | Konversi WIB tunggal dan penghapusan penyesuaian 7 jam | `needs-info` | terhalang keputusan |
| [TKT-F5-003](issues/03-kalender-hari-kerja-dan-libur.md) | Kalender hari kerja dan hari libur | `needs-info` | terhalang keputusan |
