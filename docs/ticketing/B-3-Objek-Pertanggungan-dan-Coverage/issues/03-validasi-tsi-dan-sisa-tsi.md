---
title: "TKT-B03-003 — Validasi TSI dan Sisa TSI"
labels: [modul::B-3, tipe::aturan-bisnis, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B03-003 — Validasi TSI dan Sisa TSI

Status: needs-info
Kesiapan: **terhalang artefak** — `ValidasiSisaTSI` tidak ada di export
Modul: **B-3 Objek & Coverage** · Gelombang: 3 · Bergantung pada: TKT-B03-002
Requirement: FR-B3    Keputusan: D-49 butir 8    ADR: 0016, 0017    Risiko: R-07
Rule Pega yang digantikan: `ValidasiSisaTSI` — **dipanggil tetapi tidak ada di export** (`R-07`); cacat terbukti pada `:817` versus `:1489` (**`NilaiSalvage` hanya ditambahkan bila baris terakhir kebetulan bertipe salvage**)
Peran penguji gerbang 2: **PncPICTeknik**

## Hasil yang diharapkan (dan nilai bisnisnya)

Nilai yang diusulkan untuk sebuah coverage **tidak boleh melebihi kapasitas pertanggungan yang
tersisa** — TSI dikurangi akseptasi yang masih Outstanding, ditambah Salvage.

Nilai bisnisnya langsung ke uang. Dan aturan ini **terbukti cacat hari ini**: `NilaiSalvage` hanya
ditambahkan bila baris terakhir kebetulan bertipe salvage. Artinya kapasitas yang seharusnya pulih
karena salvage **kadang dihitung, kadang tidak** — tergantung urutan baris. Ini butir 8 pada 13
perbaikan eksplisit `P-5`.

## Ruang lingkup

- Perhitungan **Sisa TSI** per Objek Pertanggungan per Coverage:
  `TSI − akumulasi akseptasi Outstanding + Salvage`.
- Validasi: nilai usulan penyelesaian **tidak melebihi** Sisa TSI.
- **Perbaikan butir 8 `P-5`**: salvage **selalu** ditambahkan, bukan hanya bila baris terakhir
  bertipe salvage.
- Perhitungan memakai **nilai presisi penuh** (`ADR-0016`), bukan nilai yang sudah dibulatkan
  untuk tampilan.

## Non-goal

- **Tidak** menghitung nilai settlement — itu `B-5`.
- **Tidak** mencatat salvage — itu `B-12`; modul ini membacanya.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **`ValidasiSisaTSI`** — dipanggil tetapi **tidak ada di export** | **Tim Pega** (`R-07`) | Rumus lengkapnya hanya terbaca sebagian dari pemanggilnya |
| **Salvage memulihkan TSI — apa niat bisnisnya?** | **Work Owner** | Menentukan apakah perilaku ini memang dikehendaki atau kebetulan |
| **`>` atau `>=` pada perbandingan sisa TSI?** | **Work Owner** | Menentukan apakah nilai **tepat sama** dengan sisa TSI diterima atau ditolak |

## Acceptance criteria

- [ ] Sisa TSI dihitung dengan rumus `TSI − Outstanding + Salvage` — diuji pada 10 kombinasi,
      termasuk kasus tanpa salvage dan tanpa akseptasi.
- [ ] **Salvage selalu ikut dihitung**, apa pun urutan barisnya — diuji dengan salvage di baris
      pertama, tengah, dan terakhir: **ketiganya memberi hasil sama**. Inilah bukti butir 8 `P-5`
      tertutup.
- [ ] Nilai usulan melebihi Sisa TSI **ditolak**, dengan pesan yang menyebut angka sisa TSI-nya.
- [ ] Perhitungan memakai nilai presisi penuh — diuji dengan nilai berdesimal panjang; hasilnya
      **tidak** berubah karena pembulatan tampilan.
- [ ] Gerbang 1: selisih terhadap Pega **hanya** pada kasus salvage yang terpetakan ke butir 8
      `P-5`; selisih lain **wajib dijelaskan atau dinyatakan bug** (`D-54`).
- [ ] Gerbang 2: UAT **PncPICTeknik** pada klaim yang punya salvage.

## Dependency / Blocked by

`TKT-B03-002` · `TKT-B12-001` (data salvage). **Terhalang Tim Pega dan dua keputusan.**

## Constraint keamanan, data, operasional

- Perhitungan ini membatasi **berapa yang boleh dibayarkan**. Kesalahan di sini tidak terlihat di
  layar mana pun sampai tahap akseptasi.
- Perubahan pada komponen perhitungan (TSI, akseptasi, salvage) **wajib tercatat di jejak audit**.

## Migrasi skema / rollout / rollback

Tidak menambah tabel; membaca tabel coverage, akseptasi, dan salvage.

**Rollback:** mengembalikan aturan berarti **memulihkan cacat butir 8** — hanya masuk akal bila
ada temuan yang lebih buruk.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/tsi/... -run TestSisaTSI
go test ./internal/domain/tsi/... -run TestSalvageSelaluDihitung
go test ./internal/domain/tsi/... -run TestPresisiPenuh
go run ./cmd/s8 banding --modul B-3 --aturan sisa-tsi
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `NilaiSalvage` hanya ditambahkan bila baris terakhir bertipe salvage | `ValidasiSisaTSI:817` versus `:1489` · `D-49` butir 8 |
| Definisi Sisa TSI | `CONTEXT.md` — **Sisa TSI** |
| `ValidasiSisaTSI` tidak ada di export | `R-07` |
| Uang disimpan presisi penuh | `D-51` · `ADR-0016` |

## Comments
