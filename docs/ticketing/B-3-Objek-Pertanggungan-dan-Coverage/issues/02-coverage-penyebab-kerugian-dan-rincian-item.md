---
title: "TKT-B03-002 — Coverage, penyebab kerugian, dan rincian item"
labels: [modul::B-3, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B03-002 — Coverage, penyebab kerugian, dan rincian item

Status: needs-info
Kesiapan: **terhalang artefak**
Modul: **B-3 Objek & Coverage** · Gelombang: 3 · Bergantung pada: TKT-B03-001, TKT-F4-005
Requirement: FR-B3    Keputusan: D-19    ADR: 0018    Risiko: R-16
Rule Pega yang digantikan: section `ObjectCoverage` · `ObjectItemList` · harness `CauseOfLossInbox`, `DetailCauseOfLoss`, `CauseOfLossInboxSimasOnline`
Peran penguji gerbang 2: **PncPICTeknik** per lini bisnis

## Hasil yang diharapkan (dan nilai bisnisnya)

Untuk setiap objek, petugas mencatat **jaminan mana yang dipakai**, **apa penyebab kerugiannya**,
dan **rincian item** yang diklaim.

Nilai bisnisnya: coverage adalah tempat TSI berada, dan TSI-lah yang membatasi nilai yang boleh
dibayarkan. Salah mencatat coverage berarti batas pembayaran yang salah — dan itu baru ketahuan di
tahap akseptasi, jauh setelah petugas lain ikut bekerja di atasnya.

## Ruang lingkup

- Pencatatan coverage per objek: kode coverage, nama, **TSI**, sublimit TSI, penyebab kerugian,
  dan diagnosa (untuk lini kesehatan/PA).
- Penyebab kerugian dipilih dari **master** (`TKT-F4-005`), bukan diketik bebas.
- Rincian item per coverage — sparepart, biaya, dan rincian lain.
- Aturan wajib: **penyebab kerugian wajib terisi, kecuali lini Travel** (invarian `I-10`).

## Non-goal

- **Tidak** menghitung nilai settlement — itu `B-5`.
- **Tidak** membangun spreading — itu `B-4`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **When rule klasifikasi produk Aneka** yang menentukan coverage mana yang berlaku per produk | **Tim Pega** (`R-16`) | Tanpa rule-nya, daftar coverage yang sah per produk harus ditebak |
| **Pemetaan Group Panel `003`**: `LocationList` versus `t_anekalist` | **Work Owner** | Menentukan dari tabel mana daftar coverage Aneka dibaca |

## Acceptance criteria

- [ ] Satu objek dapat memuat **lebih dari satu coverage**, masing-masing dengan TSI sendiri —
      diuji.
- [ ] Penyebab kerugian dipilih dari master; nilai di luar master **ditolak** — diuji.
- [ ] Coverage tanpa penyebab kerugian **ditolak**, kecuali lini Travel — diuji kedua lini.
- [ ] Rincian item dapat ditambah dan dihapus, dan **totalnya tidak melebihi TSI coverage** —
      diuji pada batas.
- [ ] Diagnosa hanya muncul untuk lini yang memakainya, dan **aksesnya dibatasi peran Analyst
      Doctor dan RCL Dokter** (`FR-R2`) — diuji dengan peran lain: field tidak terlihat.
- [ ] Gerbang 1: coverage yang tersimpan **sama dengan Pega** pada 20 klaim contoh.
- [ ] Gerbang 2: UAT **PncPICTeknik** per lini.

## Dependency / Blocked by

`TKT-B03-001` · `TKT-F4-005` (master penyebab kerugian). **Terhalang Tim Pega dan Work Owner.**

## Constraint keamanan, data, operasional

- **Diagnosa adalah data medis.** Aksesnya dibatasi peran Analyst Doctor dan RCL Dokter (`FR-R2`),
  dan pembatasan itu berlaku **juga di staging** karena staging memuat data produksi apa adanya
  (`ADR-0029`).
- TSI coverage menjadi batas pembayaran — mengubahnya setelah klaim berjalan adalah tindakan
  bernilai tinggi dan **wajib tercatat di jejak audit**.

## Migrasi skema / rollout / rollback

Menambah tabel coverage dan rincian item. Backward-compatible.

**Rollback:** sama dengan `TKT-B03-001`.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/coverage/... -run TestCoveragePerObjek
go test ./internal/domain/coverage/... -run TestPenyebabKerugianWajib
go test ./internal/adapter/http/... -run TestAksesDiagnosaDibatasi
go run ./cmd/s8 banding --modul B-3 --aturan coverage --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Coverage berisi TSI dan sublimit | `docs/Steering/05-DOMAIN-MODEL.md` §1 |
| Invarian `I-10` penyebab kerugian wajib kecuali Travel | idem §2 |
| Akses data medis dibatasi dua peran | `FR-R2` · `BRD §17.2` |
| Harness penyebab kerugian | `Harness/CauseOfLossInbox-Harness.xml`, `DetailCauseOfLoss-Harness.xml` |

## Comments
