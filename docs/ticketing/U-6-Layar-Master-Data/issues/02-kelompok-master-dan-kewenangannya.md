---
title: "TKT-U6-002 — Kelompok master dan kewenangan pengubahnya"
labels: [modul::U-6, tipe::migrasi, status::needs-info, prioritas::sedang, gelombang::7]
milestone: "Gelombang 7 — Sisa"
epic: "Migrasi Claim PNC"
---

# TKT-U6-002 — Kelompok master dan kewenangan pengubahnya

Status: needs-info
Kesiapan: **terhalang keputusan — jumlah kelompok belum pasti**
Modul: **U-6 Layar Master Data** · Gelombang: 7 · Bergantung pada: TKT-U6-001
Requirement: FR-U6    Keputusan: D-58, D-59    ADR: 0023    Risiko: —
Rule Pega yang digantikan: harness master — `MasterRekening`, `MasterSupplier`, `MasterRecovery`, `MasterPanel_HE`, `MasterLoginSurvey`, `MasterProteksiVisibilityData`, `PNC_MasterTolakKlaim`, `DetailMasterXOL`, `DetailMasterPasalRejected`, `DetailCauseOfLoss`, `DetailDominanFactor`, `GCNMCatSparepart`, `GCNMMasterSparepartType`, `GroupingSparePart_HE`, `SparePart_HE`, `BengkelHE`, `BrowseMasterDocumentTravel_Harness`, `ListDetTypeDocument`, `DetTypeDocumenBisnis`, `ListDocumentObject`, `ListDocumentTravel`
Peran penguji gerbang 2: **pemilik masing-masing kelompok master**

## Hasil yang diharapkan (dan nilai bisnisnya)

Daftar lengkap kelompok master, masing-masing dengan **satu peran yang berwenang mengubahnya**.

Nilai bisnisnya ada pada kata "satu peran yang berwenang". Master menentukan hasil hitungan klaim —
rekening tujuan, tarif sparepart, pasal penolakan, batas XOL. Saat ini kendalinya **berbasis menu**
(`D-59`), yang berarti siapa pun yang punya menu master dapat mengubah **seluruh** isinya. Tanpa
pemetaan yang eksplisit, sistem baru akan menyalin keluasan itu tanpa ada yang memutuskannya.

## Ruang lingkup

- **Daftar kelompok master lengkap**, sebagai tabel — bukan sebagai "≥29".
- Pemetaan **kelompok → peran yang berwenang mengubah**, dari 22 peran bisnis (`D-58`).
- Konfigurasi kolom dan validasi per kelompok di atas layar baku `TKT-U6-001`.
- Pemisahan mana yang **master** dan mana yang **daftar acuan tampilan** — sebagian harness
  (`ListDocumentObject`, `ListDocumentTravel`) mungkin termasuk yang kedua.

## Non-goal

- **Tidak** menambah kelompok master baru.
- **Tidak** mengubah isi master yang berlaku.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Berapa persisnya kelompok master?** Yang tercatat baru **batas bawah: ≥29** | **Work Owner** | Tiket ini menjanjikan sebuah daftar; tanpa angka pastinya, daftar itu tidak dapat dinyatakan lengkap |
| **Kelompok mana boleh diubah peran mana?** | **Work Owner** (`D-58`, `D-59`) | Kendali sekarang berbasis menu, jadi pemetaan berbutir ini **belum pernah ada**. Membuatnya adalah **pengetatan** — perubahan perilaku yang harus disepakati, bukan diselipkan |
| **Master mana yang memengaruhi hitungan uang?** | **Work Owner** | Kelompok itu layak diperlakukan lebih ketat daripada master tampilan biasa |

## Acceptance criteria

- [ ] **Daftar kelompok master lengkap** tercatat sebagai tabel, dan jumlahnya **dilaporkan sebagai
      angka pasti** — bukan "≥29".
- [ ] Setiap kelompok punya **peran yang berwenang mengubahnya**, tercatat dan ditegakkan di server
      — diuji per kelompok dengan peran yang tidak berwenang: `403`.
- [ ] Harness yang dinyatakan **bukan master** dipindahkan ke modul yang tepat, **tidak hilang** —
      dicatat per berkas.
- [ ] Isi tiap kelompok **sama dengan isi di Pega** pada saat cutover — dibandingkan lewat `S-8`.
- [ ] **Pengetatan kewenangan dilaporkan sebagai perubahan perilaku yang disengaja** (`P-5`), bukan
      sebagai perbaikan diam-diam — setiap peran yang kehilangan akses **diberi tahu lebih dulu**.
- [ ] Gerbang 2: UAT oleh **pemilik masing-masing kelompok master**, bukan satu orang untuk semuanya.

## Dependency / Blocked by

`TKT-U6-001` · `TKT-F3-002` (22 peran) · `TKT-F4-001`. **Terhalang Work Owner.**

## Constraint keamanan, data, operasional

- **Mempersempit kewenangan akan mengambil akses dari orang yang selama ini memilikinya.** Itu
  perbaikan, tetapi ia terasa sebagai kehilangan bagi yang terkena — dan harus disampaikan sebelum
  cutover, bukan ditemukan pada hari pertama.
- Master yang memengaruhi hitungan uang **mengubah nilai klaim yang dihitung sesudahnya**; jejak
  auditnya (`TKT-U6-001`) adalah satu-satunya cara menelusuri sebabnya.
- `MasterProteksiVisibilityData` menyangkut **batas keterlihatan data** — mengubahnya mengubah siapa
  melihat apa di seluruh sistem.

## Migrasi skema / rollout / rollback

Menambah tabel pemetaan kelompok master ke peran. Tidak mengubah tabel master itu sendiri.

**Rollback:** melonggarkan kembali kewenangan **mengembalikan keadaan di mana siapa pun bermenu
master dapat mengubah semuanya**. Itu bukan rollback yang netral.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/cek-master --daftar-lengkap
go test ./internal/adapter/http/... -run TestKewenanganPerKelompokMaster
go run ./cmd/s8 banding --modul U-6 --per-kelompok
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Master data ≥29 kelompok | `docs/Steering/06-MODULE-BREAKDOWN.md` koreksi ukuran 2026-09-14 |
| 22 peran bisnis, satu-untuk-satu dengan access group | `D-58` |
| Kontrol berbasis menu, bukan berbutir aksi | `D-59` |
| Harness master di export | direktori `Harness/` |
| Otorisasi diperiksa di setiap endpoint | `ADR-0023` |

## Comments
