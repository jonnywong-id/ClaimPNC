---
title: "TKT-B05-001 — Layar Input Estimasi dan Settlement Line"
labels: [modul::B-5, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::5]
milestone: "Gelombang 5 — Nilai dan pihak luar"
epic: "Migrasi Claim PNC"
---

# TKT-B05-001 — Layar Input Estimasi dan Settlement Line

Status: needs-info
Kesiapan: **terhalang artefak** — `BrowseT_Claim_Adjustment_SQL` belum ada
Modul: **B-5 Input Estimasi** · Gelombang: 5 · Bergantung pada: TKT-B03-002, TKT-B04-001
Requirement: FR-B5    Keputusan: D-19, D-51    ADR: 0016, 0018    Risiko: R-01, R-19
Rule Pega yang digantikan: tahap **`Estimation`** dan **`Input Estimasi`** pada `Flow/Register_Flow.xml` · 29 activity · `AdjustmentList` di data · `BrowseT_Claim_Adjustment_SQL`
Peran penguji gerbang 2: **PncPICTeknik** dan **PncManagerAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Petugas teknis mencatat nilai kerugian per Coverage, dan nilai itu berjalan melewati empat tahap
yang terlacak: **Estimasi → Usulan → Akseptasi → Dibayar**.

Nilai bisnisnya: keempat tahap itu adalah **riwayat uang klaim**. Di sistem lama ia tersimpan
sebagai `AdjustmentList` tanpa jejak perubahan nilai sama sekali (`T-14`) — sehingga pertanyaan
"siapa yang menurunkan nilai ini dari 100 juta menjadi 60 juta" **tidak dapat dijawab**.

## Ruang lingkup

- Layar Input Estimasi per Coverage, dengan empat kolom nilai dan statusnya.
- Penyimpanan nilai **presisi penuh**; pembulatan hanya saat ditampilkan (`ADR-0016`).
- **Setiap perubahan nilai tercatat di jejak audit** dengan nilai sebelum dan sesudah (`S-5`) —
  inilah yang tidak ada di sistem lama.
- Validasi: nilai tidak melebihi **Sisa TSI** (`TKT-B03-003`) dan tidak melebihi TSI Coverage
  (invarian `I-4`).

## Non-goal

- **Tidak** mengonversi kurs — itu `TKT-B05-002`.
- **Tidak** menghitung fee adjuster — itu `TKT-B05-003`.
- **Tidak** menjalankan komite — itu `B-7`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **`BrowseT_Claim_Adjustment_SQL`** — kueri pembaca daftar settlement line | **DBA** (`D-55`) | Bentuk data dan penyaringannya tidak diketahui |
| **`>` atau `>=` pada perbandingan sisa TSI** | **Work Owner** | Menentukan apakah nilai **tepat sama** dengan sisa TSI diterima |

## Acceptance criteria

- [ ] Satu Coverage dapat memuat **lebih dari satu Settlement Line** — diuji.
- [ ] Keempat nilai (estimasi, usulan, akseptasi, dibayar) tersimpan terpisah dan **tidak saling
      menimpa** — diuji dengan mengisi keempatnya berurutan.
- [ ] Nilai tersimpan **presisi penuh**: nilai 4 desimal yang diketik sama persis dengan yang
      dibaca kembali — diuji.
- [ ] Setiap perubahan nilai menghasilkan **tepat satu baris jejak audit** berisi nilai sebelum,
      sesudah, pelaku, dan waktu — diuji.
- [ ] Nilai melebihi TSI Coverage **ditolak** dengan pesan yang menyebut TSI-nya.
- [ ] Nilai melebihi **Sisa TSI** ditolak (`TKT-B03-003`).
- [ ] Istilah di layar memakai **Settlement**, bukan `Adjustment` (`CONTEXT.md`).
- [ ] Gerbang 1: nilai yang tersimpan **sama dengan Pega** pada 20 klaim contoh.
- [ ] Gerbang 2: UAT **PncPICTeknik**.

## Dependency / Blocked by

`TKT-B03-002`, `TKT-B03-003`, `TKT-B04-001`, `TKT-S5-002`. **Terhalang DBA dan satu keputusan.**

## Constraint keamanan, data, operasional

- **Setiap perubahan nilai wajib diaudit** — `BRD §21.2` kriteria #9 berlaku tanpa pengecualian
  karena `D-59` menghapus pemisahan tugas.
- Orang yang sama dapat mengubah nilai **dan** menyetujuinya bila perannya memiliki kedua menu
  (`ADR-0023`). Tidak ada kontrol teknis yang mencegahnya; jejak audit satu-satunya pengimbang.
- Nilai uang dikirim dan diterima sebagai **string desimal** (`TKT-U2-004`).

## Migrasi skema / rollout / rollback

Menambah tabel Settlement Line. Backward-compatible.

**Rollback:** nilai yang tersimpan tetap ada. Karena klaim tidak berpindah sistem di tengah jalan
(`P-3`), rollback berarti klaim baru tidak masuk ke sistem baru — bukan memindahkan yang ada.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/settlement/... -run TestEmpatTahapNilai
go test ./internal/domain/settlement/... -run TestPresisiPenuh
go test ./internal/app/settlement/... -run TestPerubahanNilaiTercatat
go run ./cmd/s8 banding --modul B-5 --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Perjalanan nilai empat tahap | `CONTEXT.md` — **Settlement Line** |
| Istilah `Adjustment` ditinggalkan | `CONTEXT.md` — Istilah yang sengaja tidak dipakai lagi |
| Perubahan nilai tidak punya jejak audit di sistem lama | `T-14` |
| Invarian `I-4` nilai tidak melebihi TSI | `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| `BrowseT_Claim_Adjustment_SQL` belum tersedia | `D-55` · `R-19` |

## Comments
