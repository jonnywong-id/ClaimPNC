---
title: "TKT-B01-001 — Layar View Polis dan pembekuan snapshot"
labels: [modul::B-1, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B01-001 — Layar View Polis dan pembekuan snapshot

Status: needs-info
Kesiapan: terhalang keputusan
Modul: **B-1 View Polis** · Gelombang: 3 · Bergantung pada: TKT-F2-001, TKT-U2-001
Requirement: FR-B1    Keputusan: D-04, D-03    ADR: 0006, 0008    Risiko: R-01, R-03
Rule Pega yang digantikan: `Flow/Register_Flow.xml` tahap **`View Polis`** · `Activity/CheckViewPolis_act-Act.xml` · harness `ViewPolis`, `ViewPolis1`
Peran penguji gerbang 2: **PncAdmin** dan **PncPICTeknik**

## Hasil yang diharapkan (dan nilai bisnisnya)

Petugas dapat mencari polis, melihat isinya, dan **membekukannya** sebagai dasar klaim yang sedang
didaftarkan.

Nilai bisnisnya: inilah yang membuat nilai klaim **tidak berubah diam-diam**. Tanpa pembekuan,
endorsemen polis yang terjadi minggu depan akan mengubah dasar penilaian klaim yang sudah berjalan
hari ini — dan tidak ada yang menyadarinya.

## Ruang lingkup

- Pencarian dan tampilan polis, mengikuti susunan layar `ViewPolis`.
- **Pembekuan snapshot** pada saat klaim diregistrasi, disimpan bersama klaim dalam satu transaksi.
- Penanganan bila polis tidak ditemukan atau GISFW tidak dapat dihubungi — **pesan yang jelas**,
  bukan layar kosong.
- Menampilkan **snapshot**, bukan data polis hidup, untuk klaim yang sudah terdaftar.

## Non-goal

- **Tidak** menulis apa pun ke data polis. Kepemilikannya tetap di GISFW (`ADR-0006`).
- **Tidak** memigrasikan GISFW (`D-03`).
- **Tidak** merancang bentuk tabel snapshot — itu `TKT-B01-002`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Snapshot minimal 16 field — disengaja atau harus diperluas?** | **Work Owner** | Field yang kurang baru ketahuan saat klaim lama dibuka bertahun kemudian, dan saat itu data aslinya mungkin sudah berubah |
| **Apakah tim GISFW menyetujui Claim PNC menyimpan salinan data mereka?** | **Work Owner → Tim GISFW** | Menyentuh kepemilikan data lintas tim |
| **Cara mengambil data polis: API GISFW atau baca tabel langsung?** | **Work Owner + Tim GISFW** (`R-03`) | Menentukan apakah modul ini bergantung pada API yang belum ada |

## Acceptance criteria

- [ ] Polis dapat dicari berdasarkan nomor polis, dan hasilnya menampilkan **ketiga belas
      kelompok data** snapshot.
- [ ] Snapshot **tersimpan bersama klaim dalam satu transaksi** — kegagalan penyimpanan klaim
      **tidak meninggalkan snapshot yatim** (diuji).
- [ ] Membuka klaim yang sudah terdaftar menampilkan **snapshot**, bukan data polis hidup —
      dibuktikan dengan mengubah data polis di staging lalu membuka klaim: tampilannya **tidak
      berubah**.
- [ ] Polis tidak ditemukan menghasilkan pesan yang menyebut nomor polis yang dicari.
- [ ] GISFW tidak dapat dihubungi menghasilkan pesan **berbeda** dari polis tidak ditemukan —
      keduanya menuntut tindakan berbeda dari petugas.
- [ ] Susunan layar mengikuti `ViewPolis` — perbandingan berdampingan disetujui penguji gerbang 2.
- [ ] Gerbang 1: isi snapshot **sama dengan Pega** pada 20 polis contoh dari empat lini bisnis.
- [ ] Gerbang 2: UAT **PncAdmin** dan **PncPICTeknik**.

## Dependency / Blocked by

`TKT-F2-001` · `TKT-U2-001` · `TKT-B01-002` (bentuk tabel). Terhalang tiga keputusan di atas.

## Constraint keamanan, data, operasional

- Snapshot memuat **data nasabah**: nomor polis, nama tertanggung, NPWP. Batas data cabang dan
  lini bisnis ditegakkan di kueri (`TKT-F3-005`).
- **Claim PNC tidak pernah menulis ke data polis** — bila sebuah kueri di modul ini melakukan
  `INSERT`/`UPDATE` ke tabel polis, itu pelanggaran `P-1` dan batas bounded context sekaligus.
- Pengambilan data lintas sistem **tidak boleh berada di dalam transaksi database**
  (`TKT-F2-003`).

## Migrasi skema / rollout / rollback

Menambah tabel snapshot milik aplikasi — tidak menyentuh tabel GISFW.

**Rollback:** snapshot yang sudah terbentuk tetap ada dan tetap sah. Klaim yang memakainya tidak
terpengaruh.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/polis/... -run TestPembekuanSnapshot
go test ./internal/app/polis/... -run TestSnapshotTidakBerubahSaatPolisBerubah
grep -rInE "(INSERT|UPDATE)\s+INTO?\s+.*polis" internal/app/polis/    # HARUS 0 baris
go run ./cmd/s8 banding --modul B-1 --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Snapshot polis dibekukan saat registrasi | `D-04` · `ADR-0006` |
| Tahap `View Polis` pada flow utama | `Flow/Register_Flow.xml` |
| Isi snapshot 13 kelompok data | `docs/Steering/05-DOMAIN-MODEL.md` §1 |
| Kepemilikan data polis tetap di GISFW | `ADR-0006` |
| 141 activity GISFW ikut di export | `D-04` |

## Comments
