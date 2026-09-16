---
title: "TKT-B13-002 — Penautan proteksi ke klaim dan penandaan terpakai"
labels: [modul::B-13, tipe::migrasi, status::needs-info, prioritas::sedang, gelombang::4]
milestone: "Gelombang 4 — Persetujuan"
epic: "Migrasi Claim PNC"
---

# TKT-B13-002 — Penautan proteksi ke klaim dan penandaan terpakai

Status: needs-info
Kesiapan: terhalang keputusan (cakupan alur)
Modul: **B-13 Input Open Protection** · Gelombang: 4 · Bergantung pada: TKT-B13-001, TKT-B02-001
Requirement: FR-B13    Keputusan: D-25    ADR: 0008, 0021, 0026    Risiko: R-03
Rule Pega yang digantikan: penautan proteksi pada `Flow/CreateProtection_Flow.xml` · Ticket rule `TC_PNCInputProtection` dan `Akp_PNCInputProtection` · objek remote `GENERAL.MST_BUKA_PROTEKSI` lewat **tiga DB Link** (`@ASMD`, `@SIMASNET`, `@SMI`)
Peran penguji gerbang 2: **PncAdmin** dan **PncOPCGeneral**

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu proteksi dapat ditautkan ke klaim dan **ditandai terpakai**, sehingga tidak dapat dipakai dua
kali.

Nilai bisnisnya: proteksi yang terpakai dua kali berarti perlindungan yang sama dihitung ganda.
Penandaan inilah yang mencegahnya.

## Ruang lingkup

- Penautan satu proteksi ke satu klaim, dan penandaan **`SudahDipakaiKlaim`**.
- Penolakan bila proteksi sudah terpakai, dengan pesan yang menyebut **nomor klaim pemakainya**.
- Sinkronisasi status ke `GENERAL.MST_BUKA_PROTEKSI` — **satu-satunya objek remote yang ditulis**
  di seluruh sistem lama, dan ia dipanggil ke **tiga sistem berbeda**.
- Jejak audit pada penautan dan pelepasan.

## Non-goal

- **Tidak** membangun API pengganti DB Link — itu `S-4`.
- **Tidak** mengubah alur persetujuan proteksi (`TKT-B13-001`).

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **`CreateProtection_Flow` hanya 6 shape — cakupan sekecil itu benar?** | **Work Owner** | Bila ada bagian alur di luar flow, ia tidak akan ikut terbawa dan baru ketahuan setelah rilis |
| **API pengganti `GENERAL.MST_BUKA_PROTEKSI`** | **tim pemilik sistem** (`R-03`) | Ini **objek remote yang menulis**, ke tiga sistem. Penggantinya **wajib API idempoten** — tidak dapat dijembatani salinan berkala |

## Acceptance criteria

- [ ] Proteksi yang belum terpakai dapat ditautkan ke klaim, dan **ditandai terpakai** — diuji.
- [ ] Menautkan proteksi yang **sudah terpakai** ditolak, dengan pesan yang menyebut nomor klaim
      pemakainya — diuji.
- [ ] Melepas tautan mengembalikan proteksi menjadi tersedia, dan **tercatat di jejak audit** —
      diuji.
- [ ] Penulisan status ke sistem luar bersifat **idempoten**: mengirim dua kali tidak menghasilkan
      efek ganda — diuji dengan pengiriman berulang.
- [ ] Kegagalan penulisan ke sistem luar **tidak membatalkan penautan lokal**, tetapi masuk antrean
      untuk dicoba ulang — dan kegagalan itu **terlihat**, bukan diam.
- [ ] Pemanggilan sistem luar **tidak berada di dalam transaksi database** (`TKT-F2-003`).
- [ ] Gerbang 1: status penautan **sama dengan Pega** pada 20 kasus contoh.
- [ ] Gerbang 2: UAT **PncOPCGeneral**.

## Dependency / Blocked by

`TKT-B13-001` · `TKT-B02-001` · `TKT-S4-001` (API pengganti DB Link). **Terhalang Work Owner dan
tim pemilik sistem.**

## Constraint keamanan, data, operasional

- `GENERAL.MST_BUKA_PROTEKSI` adalah **satu-satunya objek remote yang ditulis** sistem lama, dan ia
  dipanggil ke **tiga sistem berbeda** (`@ASMD`, `@SIMASNET`, `@SMI`). Penggantinya **wajib API
  idempoten** — ini disebut eksplisit di `R-03` sebagai titik yang tidak boleh dijembatani salinan
  berkala.
- Kegagalan penulisan lintas sistem **tidak boleh** membuat proteksi tertaut di satu sisi dan tidak
  di sisi lain tanpa ada yang tahu.

## Migrasi skema / rollout / rollback

Menambah kolom penanda terpakai dan rujukan klaim. Backward-compatible.

**Rollback:** penandaan yang sudah terkirim ke sistem luar **tidak otomatis dibatalkan** — itu
menuntut pembatalan manual yang disepakati dengan tim pemilik sistem.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/proteksi/... -run TestPenautanKeKlaim
go test ./internal/app/proteksi/... -run TestTolakProteksiTerpakai
go test ./internal/adapter/eksternal/... -run TestTulisProteksiIdempoten
grep -rIn "http\." internal/app/proteksi/ | grep -i "tx\|transaksi"   # HARUS 0 baris
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `GENERAL.MST_BUKA_PROTEKSI` satu-satunya objek remote yang menulis, ke tiga sistem | `ADR-0008` · `docs/Steering/16-RISK-ANALYSIS.md` `R-03` |
| Proteksi ditandai terpakai saat ditautkan | `docs/Steering/05-DOMAIN-MODEL.md` §4 |
| Pemanggilan sistem luar tidak di dalam transaksi | `docs/Steering/10-API-STRATEGY.md` §8.2 |
| `CreateProtection_Flow` 6 shape | `docs/verifikasi-bukti-adr.md` §15 baris `B-13` |

## Comments
