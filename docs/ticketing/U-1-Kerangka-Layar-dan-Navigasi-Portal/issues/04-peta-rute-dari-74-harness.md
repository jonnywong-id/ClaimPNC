---
title: "TKT-U1-004 — Peta rute dari 74 harness dan 51 item menu"
labels: [modul::U-1, tipe::analisis, status::needs-info, prioritas::tinggi, gelombang::2]
milestone: "Gelombang 2 — Kerangka UI"
epic: "Migrasi Claim PNC"
---

# TKT-U1-004 — Peta rute dari 74 harness dan 51 item menu

Status: needs-info
Kesiapan: **terhalang artefak dan keputusan**
Modul: U-1 · Gelombang: 2 · Bergantung pada: TKT-U1-001
Requirement: FR-U1    Keputusan: D-13, D-58    ADR: 0002, 0023    Risiko: R-16
Rule Pega yang digantikan: **74 harness** dan `Navigation/pyCaseWorkerNavigation-Navigation.xml` (**51 item menu**)
Peran penguji gerbang 2: **seluruh 22 peran** — peta ini menentukan apa yang masing-masing lihat (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Daftar tetap rute SPA beserta pemetaannya ke harness lama dan ke izin menu — dasar bagi seluruh
layar `U-3`, `U-4`, `U-5`, dan `U-6`.

Nilai bisnisnya: tanpa peta ini, jumlah layar yang harus dibangun **tidak diketahui**. Dan angka
itulah yang menentukan ukuran empat modul frontend sekaligus.

## Ruang lingkup

- Pemetaan **74 harness → rute SPA**, dengan pembedaan: rute berdiri sendiri, modal di atas rute
  lain, atau tidak dibawa.
- Pemetaan **rute → izin menu** (`TKT-F3-004`), sehingga navigasi dibangun dari izin.
- Penandaan eksplisit untuk harness yang **hilang dari export** dan untuk yang statusnya belum
  diputuskan.
- Hasilnya memperbarui ukuran `U-3`, `U-4`, `U-5`, `U-6` di `06-MODULE-BREAKDOWN.md` — diajukan
  sebagai perubahan, bukan diterapkan diam-diam.

## Non-goal

- **Tidak** membangun layar apa pun.
- **Tidak** merancang ulang navigasi. `D-13` menetapkan susunan mengikuti Pega.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **7 harness portal hilang dari export**: `InboxServiceCenter` · `InboxCloseClaim_Harness` · `InboxRequestSalvage` · `PNCViewClaim` · `ReportProduksiPA_harnes` · `InboxOutstanding_Harness` · `LostAdjuster_harness` | **Tim Pega** (`R-16`) | Tujuh layar tidak dapat dipetakan isinya sama sekali |
| **Apakah ketujuh layar itu masih aktif di produksi?** | **Work Owner** | Bila tidak, **7 rute dihapus dari lingkup** — dan itu memperkecil `U-3` dan `U-5` |
| **Dari 38 harness tanpa entri menu, mana rute berdiri sendiri dan mana modal?** `pyHarnessPurpose` **tidak ada** di export | **Work Owner + Tim Pega** | Menentukan apakah 38 layar itu menambah rute atau tidak — selisih terbesar dalam estimasi frontend |
| **5 When rule peran hilang** yang masing-masing mengendalikan satu item menu | **Tim Pega** (`R-16`) | 5 dari 51 item menu tidak diketahui siapa yang boleh membukanya (`TKT-F3-004`) |

## Acceptance criteria

> Beberapa butir belum dapat diangkakan sampai artefak dan keputusan di atas tersedia.

- [ ] Setiap dari **74 harness** punya satu baris pemetaan: nama harness, rute SPA (atau
      **"tidak dibawa"**), jenis (rute/modal), modul pemilik, dan izin menu yang dibutuhkan.
- [ ] **7 harness yang hilang ditandai eksplisit** sebagai belum diketahui — **bukan** dipetakan
      dengan tebakan.
- [ ] **38 harness tanpa entri menu** diklasifikasikan, dan jumlah yang masih belum jelas
      dilaporkan sebagai angka.
- [ ] Jumlah rute akhir dilaporkan, dan perbedaannya terhadap 74 dijelaskan baris per baris.
- [ ] Setiap rute punya izin menu yang menaunginya; rute tanpa izin **menggagalkan pemeriksaan**
      — tidak ada rute yang terbuka tanpa sengaja.
- [ ] Usulan pembaruan ukuran `U-3`…`U-6` disiapkan beserta angka barunya.

## Dependency / Blocked by

Bergantung pada `TKT-U1-001` dan `TKT-F3-004`. **Terhalang Tim Pega dan Work Owner.**

**Yang terhalang olehnya:** estimasi dan penulisan tiket `U-3`, `U-4`, `U-5`, `U-6`.

## Constraint keamanan, data, operasional

- Setiap rute yang menampilkan data **wajib** punya izin menu yang menaunginya, dan penegakannya
  ada di server (`TKT-F3-005`) — bukan di peta ini.
- Peta ini menyebut nama harness dan nama rule; **tidak** menyebut data nasabah.

## Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema — tiket ini menghasilkan dokumen dan konfigurasi rute.

**Rollback:** tidak berlaku.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/inventaris-harness "Harness/" > peta-harness.csv
wc -l peta-harness.csv                       # HARUS 74 baris + header
go run ./cmd/tools/banding-menu peta-harness.csv "Navigation/pyCaseWorkerNavigation-Navigation.xml"
npm run test -- Routing.setiapRutePunyaIzin
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 74 harness, 269 section | `docs/Steering/06-MODULE-BREAKDOWN.md` §4 |
| 51 item menu di `pyCaseWorkerNavigation` | `D-58` · `ADR-0023` |
| 7 harness portal hilang | `docs/verifikasi-bukti-adr.md` §15 baris `U-1` · `R-16` |
| 38 harness tanpa entri menu; `pyHarnessPurpose` tidak ada | idem |
| 5 When rule peran hilang | `D-58` · `R-16` |

## Comments
