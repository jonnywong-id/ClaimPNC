# S-7 · Dashboard Klaim, TAT & KPI

| | |
|---|---|
| **Nama di sistem lama** | **Dashboard & Monitoring** — harness `PNCTATReport`, `ReportKPIHarness`, `OutstandingKlaimperCabang_Harness` · `Database/GETSELISIHJAM.fnc` · `Database/GET_POSISI_PROGRESS_PNC.fnc` · `PROGRESS_CLAIM_PNC` |
| **Kode modul** | `S-7` |
| **Gelombang** | 6 — Laporan |
| **Ukuran** | bagian dari 78 activity `S-2` |
| **Bergantung pada** | `S-2` Laporan & Export |
| **Kesiapan** | **TERHALANG** |

## Apa yang dikerjakan modul ini

Menampilkan **berapa lama klaim diproses** dan **berapa yang masih menggantung** — TAT, KPI, dan
posisi progres klaim per cabang.

## Kenapa modul ini TERHALANG

Karena angka yang ditampilkannya **saat ini tidak dapat dipercaya**, dan itu terbukti dari
source-nya, bukan dari dugaan.

## Temuan yang mengubah rencana

| Anggapan | Terverifikasi |
|---|---|
| TAT dihitung dengan satu rumus | **Dua basis yang saling eksklusif** — `GETSELISIHJAM` (akhir pekan saja) dan `GET_WORKING_HOURS@ASMD` (dipakai **17×**). Angka dari kedua jalur **tidak akan pernah sama** |
| TAT memperhitungkan hari libur dan jam kerja | **Tidak.** `GETSELISIHJAM` hanya mengurangi akhir pekan. SLA Jumat 16:00 → Senin 09:00 dihitung **17 jam**, bukan ~2 jam kerja |
| Kegagalan hitungan terlihat | **`EXCEPTION WHEN OTHERS THEN RETURN 0`** — kegagalan **tak terbedakan dari nol jam** |
| "Posisi aktif" disaring lewat status | **`STATUSPOSISI` selalu `'On Progress'`, nol pengecualian.** Filternya cocok dengan **semua** baris. Penanda selesai yang sebenarnya adalah **`PROGRESSDATEDONE`** |

> Artinya angka TAT dan KPI yang dilihat manajemen selama ini **mungkin keliru**, dan sebagian
> kegagalan hitungan **tersamar sebagai nol**. Memperbaikinya akan **mengubah angka** — dan itu
> harus disampaikan, bukan diselipkan (`P-5`, `R-19`).

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | **Basis TAT yang benar: `GETSELISIHJAM` (akhir pekan saja) atau `GET_WORKING_HOURS@ASMD`?** | **Work Owner** |
| **Keputusan** | **`RETURN 0` saat error pernah menyamarkan kegagalan KPI** — angka historis perlu diperiksa ulang? | **Work Owner** |
| **Artefak** | `weekends2` dan `POOLDATA.datediff` **tidak ada di export** — definisi "akhir pekan" tidak dapat diverifikasi | **DBA** (`R-01`) |
| **Keputusan** | `GET_WORKING_HOURS@ASMD` dan `HRD_LBR@ASMD` adalah objek **remote** — masuk `D-25`/`R-03`, dan logikanya **ditulis ulang di Go** | **Tim Infra + Work Owner** (`R-19`) |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-S7-001](issues/01-basis-perhitungan-tat.md) | Basis perhitungan TAT | `needs-info` |
| [TKT-S7-002](issues/02-dashboard-dan-posisi-progres.md) | Dashboard klaim dan posisi progres | `needs-info` |
