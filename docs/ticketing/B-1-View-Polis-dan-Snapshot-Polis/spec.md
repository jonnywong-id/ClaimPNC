# B-1 · View Polis & Snapshot Polis

| | |
|---|---|
| **Nama di sistem lama** | **View Polis** — tahap `View Polis` pada `Flow/Register_Flow.xml`, layar `ViewPolis`, `ViewPolis1`, `CheckViewPolis_act` |
| **Kode modul** | `B-1` |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | 18 activity |
| **Bergantung pada** | `F-1`…`F-5` |
| **Kesiapan** | **SEBAGIAN** |

## Apa yang dikerjakan modul ini

Menarik data polis dari sistem GISFW dan **membekukannya sebagai snapshot** pada saat klaim
diregistrasi. Sejak titik itu, klaim dinilai berdasarkan **keadaan polis saat kejadian**, bukan
keadaan polis hari ini.

Isi snapshot: nomor polis, produksi ke-berapa, tertanggung, periode pertanggungan, Group Panel,
jenis bisnis, cabang, total TSI, total premi, status pembayaran premi, daftar koasuransi, daftar
Fac Offer, dan daftar alamat kirim.

## Kenapa dibekukan, bukan dibaca langsung

Dua alasan, dan keduanya sengaja (`ADR-0006`):

1. **Benar secara bisnis** — endorsemen atau pembatalan polis setelah registrasi tidak boleh
   mengubah dasar penilaian klaim yang sudah berjalan.
2. **Benar secara operasional** — klaim tetap dapat diproses ketika GISFW sedang tidak dapat
   dihubungi, sehingga tuntutan 24/7 (`D-27`) tidak bergantung pada sistem tim lain.

## Rule Pega yang digantikan

| Jenis | Nama |
|---|---|
| Flow | `Flow/Register_Flow.xml` — tahap `View Polis` |
| Activity | `Activity/CheckViewPolis_act-Act.xml` dan 17 activity pendukung |
| Harness | `ViewPolis`, `ViewPolis1` |
| Objek DB | `JSON_POLIS`, `JSON_KLAIM`, `INSERTPOLISTOJSON` di `GLADMIN.ASMD` |

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **12 dependensi procedure** — `UPDATE_LOG_KONVERSI` (**162 pemanggilan**), `GETNEWID` (42), `PKG_COUNTER_PRODUCTION` (25), `PROCESSQUEUEDIRECT` (10) | **DBA** (`R-01`) |
| **Artefak** | 4 definisi queue `DBMS_AQ` · `JSON_MBU`, `JSON_FIRE`, `JSON_POLIS_PA`, `JSON_POLIS_TRAVEL`, `JSON_POLIS_MARINE_CARGO`, `json_aneka` · pengganti `INSERTPOLISTOJSON` | **DBA** |
| **Keputusan** | Snapshot minimal **16 field polis** — disengaja, atau harus diperluas? | **Work Owner** |
| **Keputusan** | `JSON_POLIS`/`JSON_KLAIM` sumber kebenaran atau staging? | **Work Owner** |
| **Keputusan** | Kapan snapshot boleh atau harus disegarkan, dan siapa yang berwenang memicunya? | **Work Owner** (`ADR-0006`) |
| **Keputusan** | Apakah tim GISFW menyetujui Claim PNC menyimpan salinan data mereka? | **Work Owner → Tim GISFW** |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B01-001](issues/01-layar-view-polis-dan-pembekuan-snapshot.md) | Layar View Polis dan pembekuan snapshot | `needs-info` |
| [TKT-B01-002](issues/02-skema-snapshot-polis.md) | Skema snapshot polis pengganti JSON bebas bentuk | `needs-info` |
| [TKT-B01-003](issues/03-penyegaran-snapshot.md) | Aturan penyegaran snapshot | `needs-info` |
