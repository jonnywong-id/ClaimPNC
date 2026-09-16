---
title: "TKT-B14-003 — Kaitan dokumen ke klaim dan tanggal terima"
labels: [modul::B-14, tipe::migrasi, status::ready-for-human, prioritas::sedang, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B14-003 — Kaitan dokumen ke klaim dan tanggal terima

Status: ready-for-human
Kesiapan: siap
Modul: **B-14 Input Receive Document** · Gelombang: 3 · Bergantung pada: TKT-B14-001, TKT-B02-001
Requirement: FR-B14    Keputusan: D-50    ADR: 0020, 0026    Risiko: —
Rule Pega yang digantikan: penautan `RCV_ID` ke klaim pada `Flow/InputReceiveDocument.xml` dan `Activity/InputRegister_act-Act.xml`
Peran penguji gerbang 2: **PncReceive** dan **PncAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu kiriman dokumen dapat **ditautkan ke klaim**, dan sejak saat itu tanggal terimanya menjadi
tanggal resmi klaim tersebut.

Nilai bisnisnya: tautan inilah yang membuat **TAT dapat dihitung dari titik yang benar**. Tanpa
tautan, perhitungan TAT dimulai dari tanggal registrasi — dan itu menyembunyikan waktu tunggu
dokumen, yang justru sering menjadi penyebab keterlambatan.

## Ruang lingkup

- Penautan kiriman ke klaim, satu arah: satu klaim dapat menunjuk satu nomor register dokumen.
- **Tanggal terima dokumen mengalir ke klaim** dan menjadi batas pada validasi tanggal
  (`TKT-B02-002`).
- Perilaku bila kiriman ditautkan ke klaim yang sudah punya tautan — ditolak, dengan pesan yang
  menyebut nomor register yang sudah tertaut.
- Pencatatan jejak audit pada penautan dan pelepasan tautan.

## Non-goal

- **Tidak** menghitung TAT — itu `S-7`; modul ini menyediakan titik awalnya.
- **Tidak** mengunggah berkas — itu `S-1`.

## Acceptance criteria

- [ ] Kiriman dapat ditautkan ke klaim, dan tanggal terimanya **muncul pada klaim** — diuji.
- [ ] Menautkan kiriman kedua ke klaim yang sudah tertaut **ditolak**, dengan pesan yang menyebut
      nomor register yang sudah ada — diuji.
- [ ] Melepas tautan mengembalikan kiriman ke inbox dan **mengosongkan tanggal terima pada klaim**
      — diuji, dan keduanya tercatat di jejak audit.
- [ ] Mengubah tanggal terima setelah tertaut **memicu validasi ulang** urutan tanggal klaim
      (`TKT-B02-002`) — diuji dengan tanggal yang membuat urutan menjadi tidak sah: perubahan
      **ditolak**.
- [ ] Klaim tanpa kiriman tertaut tetap dapat diproses — penautan **tidak wajib** — diuji.
- [ ] Gerbang 1: kaitan yang terbentuk **sama dengan Pega** pada 20 klaim contoh di staging.
- [ ] Gerbang 2: UAT **PncReceive** dan **PncAdmin** bersama, karena tautan ini menyentuh keduanya.

## Dependency / Blocked by

`TKT-B14-001` · `TKT-B02-001` (klaim harus ada untuk ditautkan) · `TKT-S5-002`.

## Constraint keamanan, data, operasional

- Perubahan tanggal terima setelah klaim berjalan **mengubah angka TAT yang dilaporkan ke
  manajemen** — karena itu ia diaudit, dan perubahannya memicu validasi ulang, bukan diterima
  diam-diam.
- Penautan menyentuh dua tabel (kiriman dan klaim) — wajib dalam satu transaksi (`TKT-F2-003`).
- Selama masa paralel, **penulis tunggal per tabel** berlaku: bila tabel klaim masih dimiliki Pega,
  modul ini hanya boleh membaca (`P-1`).

## Migrasi skema / rollout / rollback

Menambah kolom rujukan nomor register dokumen pada tabel klaim — **kolom *nullable***, sehingga
backward-compatible dan aman bagi Pega (`P-4`).

**Rollback:** kolom dibiarkan dan diabaikan; tautan yang sudah terbentuk tidak hilang.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/receivedoc/... -run TestPenautanKeKlaim
go test ./internal/app/receivedoc/... -run TestTolakTautanGanda
go test ./internal/app/receivedoc/... -run TestUbahTanggalTerimaMemicuValidasi
go run ./cmd/s8 banding --modul B-14 --aturan penautan --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `RCV_ID` sebagai nomor register dokumen | `CONTEXT.md` — **Receive Document** · `docs/Steering/05-DOMAIN-MODEL.md` §4 |
| Tanggal terima dokumen sebagai batas validasi | Invarian `I-2` |
| Dua basis perhitungan TAT | `D-50` · `ADR-0020` |
| Penulis tunggal per tabel | `P-1` · `ADR-0004` |

## Comments
