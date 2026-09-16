---
title: "TKT-F6-004 — Kesekerabatan skema empat database"
labels: [modul::F-6, tipe::fondasi, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F6-004 — Kesekerabatan skema empat database

Status: needs-info
Kesiapan: **terhalang artefak DDL dan keputusan syariah**
Modul: **F-6 Portal & Multi-Sumber Data** · Gelombang: 1 · Bergantung pada: TKT-F6-002, TKT-F2-004
Requirement: FR-F6    Keputusan: D-75    ADR: 0030, 0005    Risiko: R-08, R-16, R-20
Rule Pega yang digantikan: **tidak ada padanan** — sistem lama tidak pernah menjamin keempat database sekerabat
Peran penguji gerbang 2: **PncAdmin** dan **DBA**

## Hasil yang diharapkan (dan nilai bisnisnya)

Keempat database tetap **sekerabat skemanya**, dan penyimpangan **terdeteksi sebelum** menjadi
kegagalan di hadapan pengguna.

Nilai bisnisnya: satu basis kode melayani empat database. Bila skema salah satunya menyimpang —
kolom tertinggal, index belum dibuat, migrasi gagal separuh — maka **kode yang sama akan gagal hanya
di portal itu**, dan biasanya baru ketahuan saat petugas entitas tersebut mengerjakan klaim.

## Ruang lingkup

- Migrasi skema (`TKT-F2-004`) dijalankan **per portal**, dengan pencatatan versi per database.
- **Pemeriksaan kesekerabatan**: membandingkan versi skema keempat database dan melaporkan selisihnya.
- Perilaku bila sebuah portal tertinggal versi: portal itu **ditolak melayani**, bukan melayani
  dengan skema yang salah.
- Urutan penerapan migrasi dan cara menghentikannya bila gagal di tengah.

## Non-goal

- **Tidak** menyeragamkan **isi** data — master data memang **per portal** (`D-75`).
- **Tidak** menyatukan keempat database.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **DDL seluruh tabel dan statistik ukuran** | **DBA** (`R-08`) | Penghalang yang sama dengan `TKT-F2-004`, kini **berlipat empat**: keempat database perlu diketahui keadaannya, bukan satu |
| **Apakah keempat database sekarang sudah sekerabat?** | **DBA** | Bila sudah menyimpang sejak sekarang, pekerjaan pertama adalah **menyamakannya** — dan itu tiket tersendiri yang belum ada |
| **Apakah portal syariah butuh tabel yang tidak ada di portal lain?** | **Work Owner + Tim Pega** | `SpreadingSyariah_Act` membuktikan logikanya berbeda. Bila datanya juga berbeda, "sekerabat" perlu didefinisikan ulang — mungkin **inti yang sama plus tambahan per portal** |
| **Oracle dulu atau PostgreSQL dulu, per portal?** | **Work Owner + Tim Infra** (`ADR-0005`) | Keempat portal bisa berada di tahap migrasi yang berbeda, dan itu menambah kombinasi yang harus diuji |

## Acceptance criteria

- [ ] Versi skema tercatat **per database**, dan dapat dibaca tanpa membuka database satu per satu.
- [ ] **Selisih versi antar portal terdeteksi dan dilaporkan** — diuji dengan sengaja menahan
      migrasi di satu portal.
- [ ] Portal yang **tertinggal versi ditolak melayani**, dengan pesan yang menyebut portal dan versi
      — diuji; **tidak boleh** melayani dengan skema yang salah.
- [ ] Migrasi yang **gagal di tengah pada satu portal tidak meninggalkan portal itu setengah jadi**
      — diuji dengan migrasi yang sengaja digagalkan.
- [ ] Migrasi bersifat **backward-compatible** (`P-4`) di keempat portal — diuji dengan versi kode
      lama terhadap skema baru.
- [ ] Menambah portal baru **tidak menuntut perubahan perkakas migrasi** — diuji.
- [ ] Gerbang 2: ditinjau **DBA** bersama **PncAdmin**.

## Dependency / Blocked by

`TKT-F6-002` · `TKT-F2-004` (kerangka migrasi skema). **Terhalang DBA (`R-08`), Work Owner, dan
Tim Pega.**

## Constraint keamanan, data, operasional

- **Tidak ada koneksi ke produksi. Tidak ada DDL/DML** dari pekerjaan ini di lingkungan produksi —
  perkakasnya dibangun dan diuji di lingkungan uji.
- Perkakas migrasi memegang kredensial **keempat** database. Ia menjadi sasaran bernilai tinggi;
  kredensialnya wajib dari penyimpanan rahasia (`D-40`, `R-17`).
- **Menjalankan migrasi ke portal yang salah adalah perubahan struktur pada database badan hukum
  lain.** Perkakasnya wajib menyebut portal secara eksplisit dan menolak default.
- `S-8` kini berjalan **per portal**, dan portal Syariah **belum punya baseline Pega yang utuh**
  karena `IsServerSyariah` hilang — gerbang 1 di portal itu diganti ukuran lain (`D-42`, `D-75`).

## Migrasi skema / rollout / rollback

Menambah tabel versi skema di tiap database.

**Rollout:** migrasi diterapkan portal demi portal, dimulai dari portal dengan risiko terendah.

**Rollback:** membalik migrasi **per portal**. Portal yang sudah maju dan portal yang belum
**tidak boleh dilayani basis kode yang sama** tanpa pemeriksaan versi — itulah sebab acceptance
criteria ketiga ada.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/cek-skema-portal --semua        # melaporkan versi per portal
go test ./internal/adapter/db/... -run TestPortalTertinggalVersiDitolak
go test ./internal/adapter/db/... -run TestMigrasiGagalTidakSetengahJadi
go test ./internal/adapter/db/... -run TestMigrasiBackwardCompatible
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Satu database per entitas, master per portal | `D-75` · `ADR-0030` |
| DDL dan statistik ukuran belum ada | `R-08` · `TKT-F2-004` |
| Skema backward-compatible | `P-4` · `docs/Steering/09-DATABASE-STRATEGY.md` §9.1 |
| Oracle dulu, PostgreSQL kemudian | `ADR-0005` |
| `IsServerSyariah` hilang dari export | `docs/verifikasi-bukti-adr.md:788-791` |

## Comments
