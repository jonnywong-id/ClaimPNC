---
title: "TKT-B10-001 — Akseptasi dan penerbitan Nomor Akseptasi"
labels: [modul::B-10, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::4]
milestone: "Gelombang 4 — Persetujuan"
epic: "Migrasi Claim PNC"
---

# TKT-B10-001 — Akseptasi dan penerbitan Nomor Akseptasi

Status: needs-info
Kesiapan: **terhalang artefak** — `InsertDataAkseptasiToLeader` tidak ada di export
Modul: **B-10 Akseptasi** · Gelombang: 4 · Bergantung pada: TKT-B05-001, TKT-B07-002
Requirement: FR-B10    Keputusan: D-19    ADR: 0016, 0023, 0026    Risiko: R-07
Rule Pega yang digantikan: `InsertDataAkseptasiToLeader` — **dipanggil tetapi tidak ada di export** · `PEGA_JSON_OS_AKSEP_KLAIM`
Peran penguji gerbang 2: **PncManagerAdmin** dan **PNCKomite**

## Hasil yang diharapkan (dan nilai bisnisnya)

Nilai klaim dinyatakan **disetujui untuk dibayarkan**, dan sistem menerbitkan **Nomor Akseptasi**
sebagai penanda resminya.

Nilai bisnisnya: akseptasi adalah titik ketika perusahaan **mengikat diri untuk membayar**. Ia
tidak boleh terjadi sebelum seluruh jenjang komite menyetujui — dan itulah invarian `I-5`.

## Ruang lingkup

- Penerbitan **Nomor Akseptasi** beserta tanggalnya, per Settlement Line.
- Penegakan invarian `I-5`: akseptasi **ditolak** bila masih ada jenjang komite yang belum
  menyetujui.
- Status **Outstanding**: akseptasi yang sudah diakui nilainya tetapi belum selesai dibayar.
- Jejak audit pada penerbitan dan pembatalan akseptasi.

## Non-goal

- **Tidak** mengirim data ke kasir — itu `TKT-B10-002`.
- **Tidak** mencetak LOD — itu `TKT-B10-003`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **`InsertDataAkseptasiToLeader`** — dipanggil tetapi tidak ada di export | **Tim Pega** (`R-07`) | Aturan pembentukan data akseptasi ke "leader" tidak diketahui |
| **`PEGA_JSON_OS_AKSEP_KLAIM` memetakan `STS_PLA` ke kolom `STS_DLA`** — benar, atau salah pemetaan yang sudah berjalan? | **Work Owner** | Bila salah, status PLA dan DLA tertukar pada data Outstanding — dan itu memengaruhi laporan |

## Acceptance criteria

- [ ] Akseptasi **ditolak** bila masih ada jenjang komite yang belum menyetujui — diuji pada klaim
      dengan 3 jenjang, dicoba pada jenjang ke-2 (invarian `I-5`).
- [ ] Nomor Akseptasi terbit **unik** — diuji 20 penerbitan, nol nomor ganda.
- [ ] Nilai akseptasi **tidak melebihi** nilai usulan dan tidak melebihi Sisa TSI — diuji kedua
      batas.
- [ ] Akseptasi berstatus **Outstanding** sampai pembayaran selesai — diuji perpindahan statusnya.
- [ ] Penerbitan dan pembatalan akseptasi menghasilkan **jejak audit** dengan nilai sebelum dan
      sesudah.
- [ ] Gerbang 1: nomor dan nilai akseptasi **sama dengan Pega** pada 20 klaim contoh.
- [ ] Gerbang 2: UAT **PncManagerAdmin**.

## Dependency / Blocked by

`TKT-B05-001` · `TKT-B07-002` (keputusan komite) · `TKT-S5-002`. **Terhalang Tim Pega dan satu
keputusan.**

## Constraint keamanan, data, operasional

- **Orang yang sama dapat menyetujui komite dan menerbitkan akseptasi** bila perannya memiliki
  kedua menu (`D-59`). Tidak ada kontrol teknis yang mencegahnya; jejak audit satu-satunya
  pengimbang — dan `BRD §21.2` kriteria #9 berlaku tanpa pengecualian.
- Nilai akseptasi memakai **presisi penuh** (`ADR-0016`).

## Migrasi skema / rollout / rollback

Menambah kolom akseptasi pada Settlement Line dan tabel Outstanding. Backward-compatible.

**Rollback:** akseptasi yang sudah terbit **tetap sah** — ia sudah menjadi komitmen membayar.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/akseptasi/... -run TestTolakSebelumKomiteSelesai
go test ./internal/app/akseptasi/... -run TestNomorAkseptasiUnik
go run ./cmd/s8 banding --modul B-10 --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `InsertDataAkseptasiToLeader` tidak ada di export | `docs/verifikasi-bukti-adr.md` §15 baris `B-10` · `R-07` |
| `PEGA_JSON_OS_AKSEP_KLAIM` memetakan `STS_PLA` → `STS_DLA` | idem |
| Invarian `I-5` | `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Istilah Akseptasi dan Outstanding | `CONTEXT.md` |
| Lepas dari `BRD §21.4` | `D-55` |

## Comments
