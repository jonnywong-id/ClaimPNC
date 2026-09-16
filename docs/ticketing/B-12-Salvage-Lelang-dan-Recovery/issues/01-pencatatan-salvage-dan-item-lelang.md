---
title: "TKT-B12-001 — Pencatatan salvage dan item lelang"
labels: [modul::B-12, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::5]
milestone: "Gelombang 5 — Nilai dan pihak luar"
epic: "Migrasi Claim PNC"
---

# TKT-B12-001 — Pencatatan salvage dan item lelang

Status: needs-info
Kesiapan: **terhalang artefak dan keputusan**
Modul: **B-12 Salvage** · Gelombang: 5 · Bergantung pada: TKT-B05-001, TKT-U2-003
Requirement: FR-B12    Keputusan: D-49 butir 8 dan 9    ADR: 0012, 0017    Risiko: R-19
Rule Pega yang digantikan: `Database/INSERT_SALVAGE.prc:47` — **menulis `IDSALVAGE = NULL` saat update** · `INSERT_SALVAGE_DETAILS` · `SET_ATTACHFILETEMPSALVAGE` (**belum diterima**) · harness `InboxSalvage`, `InboxRequestSalvage` · `Activity/SetStsSalvagePNC_act-Act.xml`
Peran penguji gerbang 2: **PncPICTeknik** dan **PncCollection**

## Hasil yang diharapkan (dan nilai bisnisnya)

Barang sisa dicatat, dijual, dan hasilnya masuk kembali ke perhitungan klaim — **tanpa kehilangan
identitas barisnya**.

Nilai bisnisnya berupa cacat yang sudah berjalan: `INSERT_SALVAGE` **menulis `IDSALVAGE = NULL`
saat melakukan update**. Baris yang kehilangan identitasnya **tidak dapat lagi ditautkan** ke item
maupun ke klaimnya — dan berapa banyak baris yang sudah rusak **belum dihitung**.

## Ruang lingkup

- Pencatatan salvage induk dan **item-item** di bawahnya.
- Status penjualan per item, dan status transfer pada salvage induk.
- **Perbaikan butir 9 `P-5`**: identitas baris (`IDSALVAGE`) **tidak pernah dikosongkan** saat
  update.
- Unggah lampiran salvage memakai komponen `TKT-U2-003`.

## Non-goal

- **Tidak** menghitung sisa TSI — itu `TKT-B03-003`; modul ini menyediakan nilai salvage-nya.
- **Tidak** menangani Virtual Account — itu `TKT-B12-002`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **`SET_ATTACHFILETEMPSALVAGE`** — salah satu dari dua objek yang belum diterima dari 64 | **DBA** (`D-55`) | Aturan lampiran salvage tidak diketahui |
| **Hitungan baris rusak**: `SELECT COUNT(*) … WHERE IDSALVAGE IS NULL` | **DBA** | Menentukan besarnya data yang perlu diperbaiki sebelum migrasi — dan apakah perbaikan itu mungkin |
| **`tterjual='2'` → `STSTRANSFER='5'`: satu item terjual mengubah status seluruh salvage induk** — benar? | **Work Owner** | Bila salah, status induk berubah terlalu dini dan memengaruhi perhitungan nilai |
| **`SISAKLAIM` tidak dihitung, diterima apa adanya** — benar? | **Work Owner** | Nilai yang diterima tanpa dihitung berarti tidak ada yang memvalidasinya |

## Acceptance criteria

- [ ] Update pada salvage **tidak pernah mengosongkan `IDSALVAGE`** — diuji dengan 10 update
      berturut-turut; identitas baris tetap. **Inilah bukti butir 9 `P-5` tertutup.**
- [ ] Satu salvage induk dapat memuat banyak item, dan menjual satu item **tidak menghapus** yang
      lain — diuji.
- [ ] Perubahan status penjualan item memengaruhi status induk **sesuai keputusan Work Owner** —
      diuji sesuai jawabannya.
- [ ] Nilai salvage yang tercatat **selalu** ikut pada perhitungan Sisa TSI (`TKT-B03-003`),
      apa pun urutan barisnya — bukti butir 8 `P-5`.
- [ ] Salvage yang dihapus adalah **soft delete** (`ADR-0012`) — diuji.
- [ ] Lampiran salvage dapat diunggah dan dibuka kembali.
- [ ] Gerbang 1: data salvage **sama dengan Pega** pada 20 klaim contoh, **kecuali** kasus
      `IDSALVAGE` yang terpetakan ke butir 9 `P-5`.
- [ ] Gerbang 2: UAT **PncPICTeknik**.

## Dependency / Blocked by

`TKT-B05-001` · `TKT-U2-003` · `TKT-S1-001`. **Terhalang DBA dan dua keputusan Work Owner.**

## Constraint keamanan, data, operasional

- **Data historis mungkin sudah rusak.** Baris ber-`IDSALVAGE = NULL` tidak dapat ditautkan; bila
  jumlahnya banyak, perbaikan data menjadi pekerjaan tersendiri dengan persetujuan Work Owner dan
  DBA (`ADR-0017`).
- Salvage memengaruhi **nilai yang dibayarkan**; perubahannya wajib tercatat di jejak audit.

## Migrasi skema / rollout / rollback

Menambah tabel salvage dan item. Backward-compatible.

**Rollback:** mengembalikan perilaku lama berarti **memulihkan cacat `IDSALVAGE = NULL`** — hanya
masuk akal bila ada temuan yang lebih buruk.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/salvage/... -run TestIDSalvageTidakPernahKosong
go test ./internal/app/salvage/... -run TestItemDanInduk
go test ./internal/domain/tsi/... -run TestSalvageSelaluDihitung
go run ./cmd/s8 banding --modul B-12 --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `IDSALVAGE = NULL` saat update | `Database/INSERT_SALVAGE.prc:47` · `D-49` butir 9 |
| `NilaiSalvage` bergantung urutan baris | `ValidasiSisaTSI:817` versus `:1489` · `D-49` butir 8 |
| `SET_ATTACHFILETEMPSALVAGE` belum diterima | `D-55` |
| `tterjual='2'` → `STSTRANSFER='5'` | `docs/verifikasi-bukti-adr.md` §15 baris `B-12` |
| Salvage memulihkan kapasitas pertanggungan | `CONTEXT.md` — **Sisa TSI** |

## Comments
