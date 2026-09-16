# 0008 — Ganti seluruh akses DB Link dengan pemanggilan API ke sistem pemilik data

Status: Accepted
Tanggal keputusan: 2026-09-07    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-25`, `D-57`, `R-03` | 64 pemakaian DB Link pada export | `Service REST/` (4 layanan masuk)
Terkait: ADR-0006, ADR-0022, modul `S-4`, `B-7`, `B-12`

## Konteks

Sistem lama menjangkau enam database lain lewat **DB Link Oracle** — kopling yang tidak terlihat
dari kode aplikasi mana pun, hanya dari teks SQL:

| DB Link | Pemakaian | Data yang diambil |
|---|---|---|
| `@ASMD` | **55×** | `DATAMINING.GET_WORKING_HOURS` (17×), `HRDASM.V_HRD_MST`, `MST_DET_SALES`, `GENERAL.MST_BUKA_PROTEKSI`, `GL.T_ALL_PAYMENT`, `LST_USER_ASURANSI`, `GENERAL.LST_MITRA`, `TREATY_LOSS`, `MBU.T_CADANGAN_KLAIM_KREDIT`, fungsi `GET_NAMA_*` |
| `@SIMASNET` | 3× | `GENERAL.MST_BUKA_PROTEKSI`, `V_KLAIM` |
| `@SMI` | 2× | `GENERAL.MST_BUKA_PROTEKSI`, `V_KLAIM` |
| `@OPJAVA` | 2× | `NEW_GENERAL.M_USER`, `M_USER_JOB` |
| `@PROD_ASM` | 1× | `POOLDATA.AGENT` |
| `@PROD_TKA` | 1× | `ANEKA.MST_SHARE_PU` |

Konsekuensi bentuk ini: perubahan skema di database mana pun di atas dapat merusak Claim PNC
**tanpa peringatan**, karena tidak ada kontrak apa pun di antaranya.

## Opsi yang dipertimbangkan

1. **Ganti seluruhnya dengan pemanggilan API** ke sistem pemilik data.
2. **Pertahankan DB Link** — paling murah, kopling tetap.
3. **Replikasi data** yang dibutuhkan ke database Claim PNC.

## Keputusan

Seluruh akses lintas database lewat DB Link **diganti pemanggilan API** ke sistem pemilik data.
Kopling tersembunyi antar database dihapus dan diganti **kontrak yang eksplisit**.

## Rationale

DB Link menyembunyikan ketergantungan justru pada titik yang paling mahal bila putus. Kontrak API
memaksa ketergantungan itu menjadi terlihat: ada pemilik, ada versi, ada perilaku saat gagal.

Replikasi (opsi 3) menukar satu masalah dengan masalah lain — keusangan data — dan menambah
komponen yang harus dipantau, bertentangan dengan `D-08`.

## Konsekuensi

### Positif

- Ketergantungan lintas sistem menjadi eksplisit, bernama, dan dapat diuji tiruannya.
- Perubahan skema di sistem lain tidak lagi merusak Claim PNC secara diam-diam.
- Sejalan dengan ADR-0005: DB Link adalah fitur Oracle yang tidak ada padanannya di PostgreSQL.

### Negatif / utang teknis

- **API penggantinya kemungkinan besar belum ada** (`R-03`). Yang diganti bukan cara memanggil,
  melainkan kontrak yang harus dibangun tim lain lebih dulu.
- Panggilan jaringan menggantikan join database: lebih lambat, dan **memperkenalkan mode gagal
  yang sebelumnya tidak ada**. Setiap titik integrasi butuh timeout, percobaan ulang, dan
  perilaku saat sistem lawan mati.
- `DATAMINING.GET_WORKING_HOURS` dipanggil **17×** untuk perhitungan TAT (ADR-0020). Mengubahnya
  menjadi panggilan jaringan di dalam kalkulasi yang dipakai laporan massal berisiko pada kinerja
  — kemungkinan besar menuntut caching, yang membawa persoalan kebaruan datanya sendiri.
- Enam sistem berarti enam tim, enam jadwal, dan enam kesepakatan — seluruhnya di luar kendali
  proyek ini.

### Risiko yang diterima secara sadar

- Bila satu API tidak pernah datang, modul yang bergantung padanya berhenti. Mitigasinya adalah
  kebijakan `D-37` untuk modul yang artefaknya tidak pernah tiba — bukan solusi teknis.
- Ketersediaan Claim PNC menjadi ikut bergantung pada ketersediaan sistem lain saat runtime, hal
  yang justru dihindari ADR-0006 untuk data polis.

## Pertanyaan terbuka

- API pengganti untuk enam DB Link — sudah ada, atau harus dibangun tim pemiliknya? Pemilik: Work
  Owner → tim masing-masing sistem. Ini `R-03`, dan menghalangi tiket `S-4`.
- **Ditemukan setelah `D-25` diputuskan:** `Service REST/` memuat **empat layanan masuk** —
  `KomiteAcceptAdjustment`, `KomiteAcceptAdjustmentPA`,
  `RecivedDataandAttachmentLelangASMSimasbid`, `RequestCreateClaimCredit2`. Dua di antaranya
  **menerima persetujuan komite dari sistem lain**. Permukaan masuk ini belum pernah masuk
  hitungan `FR-S4` maupun `D-25`, dan lingkup `S-4` karenanya **lebih besar daripada yang
  disetujui**. Pemilik: Work Owner.
