---
title: "TKT-B01-002 — Skema snapshot polis pengganti JSON bebas bentuk"
labels: [modul::B-1, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B01-002 — Skema snapshot polis pengganti JSON bebas bentuk

Status: needs-info
Kesiapan: **terhalang artefak** — 12 dependensi procedure belum diterima
Modul: **B-1 View Polis** · Gelombang: 3 · Bergantung pada: TKT-F2-004
Requirement: FR-B1    Keputusan: D-04, D-02, D-68    ADR: 0006, 0007    Risiko: R-01, R-08
Rule Pega yang digantikan: `JSON_POLIS` · `JSON_KLAIM` · `INSERTPOLISTOJSON` di `GLADMIN.ASMD` · `JSON_MBU`, `JSON_FIRE`, `JSON_POLIS_PA`, `JSON_POLIS_TRAVEL`, `JSON_POLIS_MARINE_CARGO`, `json_aneka`
Peran penguji gerbang 2: **tidak berlaku** — pekerjaan skema; diuji lewat `TKT-B01-001`

## Hasil yang diharapkan (dan nilai bisnisnya)

Snapshot polis disimpan dalam **skema eksplisit** — kolom bernama, bertipe, dan dapat di-index —
menggantikan JSON bebas bentuk.

Nilai bisnisnya: JSON bebas bentuk berarti **tidak ada yang tahu field apa saja yang benar-benar
ada** sampai seseorang membacanya satu per satu. Skema eksplisit membuat field yang hilang
ketahuan saat penyimpanan, bukan bertahun kemudian saat klaim dibuka kembali.

## Ruang lingkup

- Skema tabel snapshot polis beserta tabel anaknya: koasuransi, Fac Offer, dan alamat kirim.
- Pemetaan dari bentuk JSON lama ke kolom baru, **dibuat dari pembacaan pemakaian di rule** —
  karena bentuk JSON-nya tidak terdokumentasi.
- Index yang mendukung pencarian klaim berdasarkan nomor polis dan tertanggung.
- Dokumentasi field yang **sengaja tidak dibawa**, beserta alasannya.

## Non-goal

- **Tidak** memanggil stored procedure (`ADR-0007`).
- **Tidak** memakai kolom JSON sebagai tempat menyimpan data yang belum dipetakan — itu memindahkan
  masalahnya, bukan menyelesaikannya.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **12 dependensi procedure** yang dipanggil 62 procedure yang sudah diterima — terberat `UPDATE_LOG_KONVERSI` (**162 pemanggilan**), `GETNEWID` (42), `PKG_COUNTER_PRODUCTION` (25), `PROCESSQUEUEDIRECT` (10) | **DBA** (`R-01`) | Logikanya menentukan field apa yang benar-benar dibentuk saat konversi polis |
| **Pengganti `INSERTPOLISTOJSON`** di `GLADMIN.ASMD` dan **4 definisi queue `DBMS_AQ`** | **DBA** | Menentukan apakah pembentukan snapshot sinkron atau lewat antrean |
| **`JSON_POLIS`/`JSON_KLAIM`: sumber kebenaran atau staging?** | **Work Owner** | Bila sumber kebenaran, ia harus ikut dibawa; bila staging, ia boleh ditinggalkan |
| **DDL tabel** (`R-08`) | **DBA** | Tipe dan panjang kolom sumbernya tidak diketahui |

## Acceptance criteria

- [ ] Seluruh field snapshot punya **kolom bernama dan bertipe** — **nol kolom JSON** sebagai
      penampung sisa; diuji pemindaian definisi skema.
- [ ] Pemetaan JSON lama → kolom baru tersedia sebagai berkas, dan **setiap field yang tidak
      dibawa punya baris alasan**.
- [ ] Membaca snapshot 20 klaim lama dari staging menghasilkan **nilai yang sama** dengan yang
      ditampilkan Pega — diuji.
- [ ] Pencarian klaim berdasarkan nomor polis memakai index — dilampirkan `EXPLAIN PLAN`.
- [ ] Migrasi skema backward-compatible dan punya `down.sql` yang berfungsi (`TKT-F2-004`).
- [ ] **Nol pemanggilan stored procedure** dari modul ini — diuji pemindaian.

## Dependency / Blocked by

`TKT-F2-004`. **Terhalang DBA (`R-01`, `R-08`) dan satu keputusan Work Owner.**

## Constraint keamanan, data, operasional

- Snapshot memuat data nasabah; aturan akses sama dengan data klaim.
- Tabel snapshot **tumbuh satu baris per klaim** — pada ribuan klaim per bulan (`D-10`), kapasitas
  dan index-nya dirancang sejak awal, bukan diperbaiki kemudian.
- Skema baru **tidak menyentuh tabel yang dibaca Pega**; keduanya hidup berdampingan selama masa
  paralel.

## Migrasi skema / rollout / rollback

Menambah tabel baru. **Rollback:** tabel dibiarkan; aplikasi kembali membaca `JSON_POLIS`
langsung — **hanya mungkin bila** jawaban atas pertanyaan "sumber kebenaran atau staging" adalah
*sumber kebenaran*.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
migrate -path db/migrations -database "$DSN" up
go test ./internal/adapter/sqlstore/... -run TestSnapshotPolis
go run ./cmd/tools/banding-snapshot --kasus 20 --sumber staging
grep -rIn "CALL \|EXEC " internal/app/polis/     # HARUS 0 baris
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Snapshot dengan skema eksplisit, bukan JSON bebas bentuk | `D-04` · `ADR-0006` |
| 12 dependensi procedure; `UPDATE_LOG_KONVERSI` 162× | `docs/verifikasi-bukti-adr.md` §14.2 · `R-01` |
| Tanpa pemanggilan stored procedure | `D-02` · `ADR-0007` |
| DDL tabel belum tersedia | `R-08` |

## Comments
