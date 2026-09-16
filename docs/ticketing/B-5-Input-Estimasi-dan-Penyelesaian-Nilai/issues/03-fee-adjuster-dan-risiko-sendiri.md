---
title: "TKT-B05-003 — Fee adjuster, risiko sendiri, dan salvage pada nilai"
labels: [modul::B-5, tipe::aturan-bisnis, status::needs-info, prioritas::tinggi, gelombang::5]
milestone: "Gelombang 5 — Nilai dan pihak luar"
epic: "Migrasi Claim PNC"
---

# TKT-B05-003 — Fee adjuster, risiko sendiri, dan salvage pada nilai

Status: needs-info
Kesiapan: **terhalang artefak** — isi `GCNM_FEE_SCALE` (17 pita) belum ada
Modul: **B-5 Input Estimasi** · Gelombang: 5 · Bergantung pada: TKT-B05-001, TKT-B12-001
Requirement: FR-B5    Keputusan: D-49 butir 8, D-51    ADR: 0016, 0017    Risiko: R-19
Rule Pega yang digantikan: perhitungan fee adjuster dari `POOLDATA.GCNM_FEE_SCALE` (**17 pita**) · pengurangan risiko sendiri · penambahan salvage pada sisa TSI (`ValidasiSisaTSI:817` versus `:1489`)
Peran penguji gerbang 2: **PncPICTeknik** dan **PNCSurveyor**

## Hasil yang diharapkan (dan nilai bisnisnya)

Tiga komponen yang mengubah nilai bersih klaim dihitung konsisten: **fee adjuster**, **risiko
sendiri**, dan **salvage**.

Nilai bisnisnya: ketiganya bekerja di arah berbeda — fee menambah biaya, risiko sendiri mengurangi
yang dibayarkan, salvage memulihkan kapasitas. Salah satu saja keliru, dan angka akhir yang
dibayarkan ke tertanggung ikut salah.

## Ruang lingkup

- Perhitungan **fee adjuster** dari pita nilai pada `GCNM_FEE_SCALE` (17 pita).
- Pengurangan **risiko sendiri** (own risk) dari nilai yang dibayarkan.
- Penambahan **salvage** pada perhitungan sisa TSI — dengan perbaikan butir 8 `P-5`: salvage
  **selalu** ditambahkan, bukan hanya bila baris terakhir kebetulan bertipe salvage.
- Seluruh perhitungan memakai **nilai presisi penuh**; pembulatan hanya saat tampil.

## Non-goal

- **Tidak** mencatat data salvage — itu `B-12`; modul ini membacanya.
- **Tidak** menghitung TSI dan sisa TSI — itu `TKT-B03-003`; modul ini memakainya.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Isi `POOLDATA.GCNM_FEE_SCALE` (17 pita)** | **DBA** (`R-19`) | Tanpa isinya, fee adjuster tidak dapat dihitung maupun diuji. Salah satu dari tiga penghalang `BRD §21.4` |
| **Berapa desimal pembulatan fee?** | **Work Owner** | Fee dihitung dari pita; pembulatan yang berbeda menghasilkan angka berbeda pada tagihan adjuster |
| **Rp 1.650.000 dan 2% masih berlaku?** | **Work Owner** | Dua konstanta yang muncul di perhitungan; keduanya tidak dapat dikonfirmasi dari source |
| **Salvage memulihkan TSI — apa niat bisnisnya?** | **Work Owner** | Menentukan apakah perilaku itu memang dikehendaki |

## Acceptance criteria

- [ ] Fee adjuster dihitung dari pita yang benar untuk 17 pita — diuji pada **batas bawah dan
      batas atas setiap pita**, ditambah satu nilai di tengah.
- [ ] Nilai tepat pada batas pita diperlakukan konsisten — diuji, dan aturannya (`>` atau `>=`)
      dinyatakan eksplisit.
- [ ] Risiko sendiri dikurangkan dari nilai dibayar, **bukan** dari nilai akseptasi — diuji.
- [ ] **Salvage selalu ikut dihitung** pada sisa TSI, apa pun urutan barisnya — diuji dengan
      salvage di posisi pertama, tengah, dan terakhir: **hasilnya sama**. Inilah bukti butir 8
      `P-5` tertutup.
- [ ] Seluruh perhitungan memakai presisi penuh; pembulatan **hanya** di lapisan tampilan — diuji
      dengan rantai perhitungan empat tahap.
- [ ] Gerbang 1: selisih terhadap Pega **hanya** pada kasus salvage yang terpetakan ke butir 8
      `P-5`; selisih pada fee **wajib nol** bila isi `GCNM_FEE_SCALE` sama.
- [ ] Gerbang 2: UAT **PncPICTeknik** dan **PNCSurveyor**.

## Dependency / Blocked by

`TKT-B05-001` · `TKT-B03-003` · `TKT-B12-001`. **Terhalang DBA dan tiga keputusan Work Owner.**

## Constraint keamanan, data, operasional

- Fee adjuster adalah **tagihan ke pihak luar** — kesalahannya terlihat oleh adjuster, bukan hanya
  internal.
- Konstanta Rp 1.650.000 dan 2% **tidak boleh di-hardcode** (`ADR-0025`); keduanya menjadi master
  di `F-4` setelah statusnya dikonfirmasi.
- Setiap perubahan komponen perhitungan **wajib tercatat di jejak audit**.

## Migrasi skema / rollout / rollback

Memakai master fee (`F-4`) dan tabel salvage (`B-12`); tidak menambah tabel sendiri.

**Rollback:** mengembalikan perhitungan salvage berarti **memulihkan cacat butir 8**.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/settlement/... -run TestFeeAdjusterPerPita
go test ./internal/domain/settlement/... -run TestSalvageSelaluDihitung
go test ./internal/domain/settlement/... -run TestRisikoSendiri
go run ./cmd/s8 banding --modul B-5 --aturan fee
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `GCNM_FEE_SCALE` 17 pita | `D-55` · `docs/verifikasi-bukti-adr.md` §15 baris `B-5` |
| `NilaiSalvage` hanya ditambahkan bila baris terakhir bertipe salvage | `ValidasiSisaTSI:817` versus `:1489` · `D-49` butir 8 |
| Salvage memulihkan kapasitas pertanggungan | `CONTEXT.md` — **Sisa TSI** |
| Uang presisi penuh, pembulatan saat tampil | `D-51` · `ADR-0016` |
| Nilai bisnis tidak boleh di-hardcode | `D-15` · `ADR-0025` |

## Comments
