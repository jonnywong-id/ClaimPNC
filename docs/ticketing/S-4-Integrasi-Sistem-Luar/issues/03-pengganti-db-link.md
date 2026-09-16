---
title: "TKT-S4-003 — Enam API pengganti DB Link"
labels: [modul::S-4, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::5]
milestone: "Gelombang 5 — Nilai dan pihak luar"
epic: "Migrasi Claim PNC"
---

# TKT-S4-003 — Enam API pengganti DB Link

Status: needs-info
Kesiapan: **terhalang artefak — kontrak API belum ada**
Modul: **S-4 Integrasi Sistem Luar** · Gelombang: 5 · Bergantung pada: TKT-S4-001
Requirement: FR-S4    Keputusan: D-25    ADR: 0017    Risiko: R-03, R-19
Rule Pega yang digantikan: **64 pemakaian DB Link** ke 6 database lain — `@ASMD` (55×), `@SIMASNET` (3×), `@SMI` (2×), `@OPJAVA` (2×), `@PROD_ASM` (1×), `@PROD_TKA` (1×)
Peran penguji gerbang 2: **PncAdmin** dan pemilik masing-masing sistem sumber

## Hasil yang diharapkan (dan nilai bisnisnya)

Data dari enam database lain diambil lewat **kontrak yang eksplisit**, bukan lewat sambungan
langsung antar database.

Nilai bisnisnya: PostgreSQL tidak punya DB Link, jadi ini bukan pilihan gaya. Tetapi nilai yang
sebenarnya adalah **hilangnya kopling tersembunyi** — selama ini enam sistem lain dapat mengubah
tabelnya dan memecahkan Claim PNC tanpa ada yang tahu sampai kejadian.

## Ruang lingkup

- Enam klien API menggantikan 64 pemakaian DB Link, sesuai `D-25`.
- **Penulisan ulang aturan bisnis yang selama ini bersembunyi di objek remote** — `GET_WORKING_HOURS`
  (17×) dan `HRD_LBR` adalah **perhitungan jam kerja dan kalender libur**, yaitu aturan bisnis;
  keduanya ditulis ulang di Go, **bukan dipanggil lewat API** (`D-…` konsekuensi `R-19`).
- Perilaku saat sistem sumber tidak menjawab.

## Non-goal

- **Tidak** membangun API di sisi sistem pemilik data — itu pekerjaan tim mereka.
- **Tidak** memakai Foreign Data Wrapper maupun replikasi; keduanya ditolak di `D-25`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Keenam API kemungkinan besar belum ada** — belum ada satu pun kontrak | **Tim pemilik masing-masing sistem** (`R-03`) | Ini bukan pekerjaan yang dapat diselesaikan sepihak. Setiap satu menuntut kesepakatan dengan tim lain, dan **jadwalnya bukan milik proyek ini** |
| **Apa yang terjadi bila sistem sumber tidak menjawab** — klaim ditahan, atau lanjut dengan data kosong? | **Work Owner** | Keduanya sah, tetapi akibatnya sangat berbeda bagi petugas di cabang |
| **Data mana yang boleh disimpan sementara (cache) dan berapa lama?** | **Work Owner + pemilik data** | Data polis dan HRD yang basi dapat menghasilkan keputusan klaim yang salah |

## Acceptance criteria

- [ ] **Nol pemakaian DB Link** tersisa di kode Go — dicari secara otomatis.
- [ ] Keenam sumber data terlayani, dan **jumlahnya dilaporkan sebagai angka**.
- [ ] **`GET_WORKING_HOURS` dan kalender libur berjalan sebagai aturan di Go**, bukan panggilan
      remote — diuji terhadap kasus yang sama dengan `TKT-F5-001`.
- [ ] Sistem sumber tidak menjawab **ditangani sesuai keputusan Work Owner**, dan perilakunya
      **terlihat oleh petugas** — bukan diam.
- [ ] Hasil pengambilan data **sama dengan hasil DB Link** pada 20 kasus per sumber — dibandingkan
      lewat `S-8`.
- [ ] Gerbang 2: UAT per **pemilik sistem sumber**.

## Dependency / Blocked by

`TKT-S4-001` · `TKT-F5-001` (jam kerja & hari libur). **Terhalang tim pemilik sistem (`R-03`) —
penghalang di luar kendali proyek.**

## Constraint keamanan, data, operasional

- **`R-03` adalah penghalang yang bergantung pada pihak lain.** Jadwal `S-4` tidak dapat dijanjikan
  selama kontrak keenam API belum ada — menuliskan tanggal untuk pekerjaan ini akan menyesatkan.
- Data HRD dan polis adalah **data pribadi**; kontrak API harus membatasi apa yang boleh diambil,
  bukan menyalin akses tabel penuh yang selama ini dimiliki DB Link.
- Selama peralihan, Pega masih memakai DB Link sementara Go memakai API — **dua jalur ke data yang
  sama**, dan selisihnya harus terpantau (`P-1`, `S-8`).

## Migrasi skema / rollout / rollback

Tidak menambah tabel. Bila cache disepakati, ia tabel baru dan backward-compatible (`P-4`).

**Rollback:** **tidak dapat kembali ke DB Link di PostgreSQL** — mekanismenya tidak ada di sana.
Rollback yang tersedia hanya mengembalikan lalu lintas ke Pega.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/cari-dblink ./internal/...      # HARUS nol temuan
go test ./internal/app/jamkerja/... -run TestGetWorkingHoursDiGo
go run ./cmd/s8 banding --modul S-4 --kasus 20 --per-sumber
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 64 pemakaian DB Link ke 6 database | `D-25` tabel inventaris · `docs/Steering/00-DECISION-LOG.md:605-612` |
| `@ASMD` 55×, `GET_WORKING_HOURS` 17× | idem |
| API pengganti kemungkinan belum ada | `D-25` · `R-03` |
| `GET_WORKING_HOURS` dan `HRD_LBR` ditulis ulang di Go, bukan dipanggil | `docs/Steering/00-DECISION-LOG.md:1326` · `R-19` |
| FDW dan replikasi ditolak | `D-25` pilihan 2 dan 3 tidak dipilih |

## Comments
