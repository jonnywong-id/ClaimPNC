---
title: "TKT-U4-001 — Kerangka layar transaksi per tahap klaim"
labels: [modul::U-4, tipe::fondasi, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-U4-001 — Kerangka layar transaksi per tahap klaim

Status: needs-info
Kesiapan: **terhalang keputusan pembagian layar**
Modul: **U-4 Layar Transaksi Klaim** · Gelombang: 3 · Bergantung pada: TKT-U2-001, TKT-U2-002, TKT-U1-004
Requirement: FR-U4    Keputusan: D-05    ADR: 0012, 0023    Risiko: —
Rule Pega yang digantikan: `Flow/Register_Flow.xml` (tahapan) · 29 Flow Action · harness `InputProgress`, `ProgressClaim_Harness`, `StatusProgress`, `StatusProgress2`
Peran penguji gerbang 2: **PncRegister**, **PncSurveyor**, **PncAnalis**

## Hasil yang diharapkan (dan nilai bisnisnya)

Kerangka layar yang mengikuti tahapan klaim, sehingga petugas melihat **di mana klaim berada** dan
**apa tindakan yang tersedia baginya di tahap itu**.

Nilai bisnisnya: tahapan klaim di Pega terbentuk dari Flow dan Flow Action, dan petugas terbiasa
dengan urutannya. Bila layar baru tidak menampilkan tahapan yang sama, petugas kehilangan orientasi
— bukan karena datanya salah, melainkan karena mereka tidak tahu lagi sedang di mana.

## Ruang lingkup

- Kerangka layar per tahap `Register_Flow`: Input Register, View Polis, Input Estimasi,
  Choose Surveyor, Send To Analis / PIC Teknik, RCL/PUCL, Compliance, Investigator,
  Analyst Doctor / RCL Dokter.
- Penanda tahap berjalan dan riwayat tahap yang sudah dilalui.
- **Tindakan yang tersedia ditentukan server**, bukan disembunyikan di layar (`ADR-0023`).
- Penyimpanan draft agar petugas tidak kehilangan isian saat berpindah tahap.

## Non-goal

- **Tidak** memuat aturan bisnis apa pun. Seluruhnya milik modul `B-*` yang bersangkutan.
- **Tidak** membangun layar pencarian dan detail — itu `TKT-U4-002`.
- **Tidak** mengubah urutan tahapan.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Pembagian 48 harness non-Inbox ke `U-4`, `U-5`, dan `U-6` per berkas** | **Work Owner + Lead Engineer** | Jumlah layar yang dijanjikan modul ini belum dapat dinyatakan sebagai angka |
| **Tahapan ditampilkan sebagai jalur bertahap, atau sebagai satu halaman panjang?** | **Work Owner** | Mengubah cara petugas bekerja sehari-hari, dan `P-5` menuntut kesetaraan perilaku lebih dulu |
| **Lompatan lateral (Ticket rule) muncul sebagai apa di layar?** | **Work Owner** | Di Pega klaim dapat melompat antar tahap lewat Ticket rule (`FR-W2`). Bila layar tidak menunjukkannya, petugas akan melihat klaim berpindah tanpa sebab yang terlihat |

## Acceptance criteria

- [ ] Tahapan yang ditampilkan **sama dengan tahapan `Register_Flow`** — diperiksa satu per satu.
- [ ] **Tindakan yang tersedia ditentukan server**; menyembunyikan tombol di layar **tidak menjadi
      satu-satunya penghalang** — diuji dengan memanggil endpoint langsung: `403`.
- [ ] Klaim yang berpindah lewat **lompatan lateral** tetap terbaca posisinya di layar — diuji.
- [ ] Isian yang belum disimpan **tidak hilang** saat petugas berpindah tahap atau sesi terputus —
      diuji.
- [ ] Seluruh tabel di layar ini memakai komponen `TKT-U2-001`, dan seluruh form memakai
      `TKT-U2-002` — diperiksa; tidak ada tabel atau form buatan sendiri.
- [ ] Angka uang dan tanggal dipformat lewat `TKT-U2-004` — **sama dengan yang tampil di laporan**.
- [ ] Gerbang 2: UAT **PncRegister**, **PncSurveyor**, dan **PncAnalis** — masing-masing pada
      tahapnya sendiri.

## Dependency / Blocked by

`TKT-U2-001` · `TKT-U2-002` · `TKT-U1-004` (peta rute dari 74 harness) · modul `B-*` yang menjadi
isinya. **Terhalang keputusan pembagian layar.**

## Constraint keamanan, data, operasional

- **Layar bukan tempat menegakkan kewenangan.** `D-59` menetapkan kontrol berbasis menu, dan
  `ADR-0023` menuntut pemeriksaan di setiap endpoint. Tombol yang disembunyikan **bukan kendali**.
- Layar tahap Analyst Doctor dan RCL Dokter memuat **data medis** (`FR-R2`).
- Selama peralihan, klaim yang sama dapat dibuka di Pega **dan** di layar baru (`D-05`) — aturan
  penulis tunggal (`P-1`) menentukan sisi mana yang boleh menyimpan.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema; menambah tabel draft isian bila disepakati. Backward-compatible.

**Rollback:** petugas dikembalikan ke layar Pega per tahap. Draft yang tersimpan di sistem baru
**tidak terbawa** ke Pega.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm test -- transaksi --run TestTahapanSesuaiRegisterFlow
go test ./internal/adapter/http/... -run TestTindakanDitentukanServer
npm test -- transaksi --run TestDraftTidakHilang
go run ./cmd/s8 banding --modul U-4 --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Tahapan klaim | `Flow/Register_Flow.xml` |
| 29 Flow Action | direktori `Flow Action/` (dihitung langsung) |
| 269 section, 268 bergrid | `docs/Steering/06-MODULE-BREAKDOWN.md:88` |
| Lompatan lateral lewat Ticket rule | `FR-W2` · `docs/Steering/CONTEXT.md` |
| Otorisasi diperiksa di server, kontrol berbasis menu | `D-59` · `ADR-0023` |

## Comments
