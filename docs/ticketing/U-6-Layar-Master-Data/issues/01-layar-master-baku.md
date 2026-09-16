---
title: "TKT-U6-001 — Layar master baku (CRUD seragam)"
labels: [modul::U-6, tipe::fondasi, status::needs-info, prioritas::sedang, gelombang::7]
milestone: "Gelombang 7 — Sisa"
epic: "Migrasi Claim PNC"
---

# TKT-U6-001 — Layar master baku (CRUD seragam)

Status: needs-info
Kesiapan: **terhalang keputusan jumlah kelompok master**
Modul: **U-6 Layar Master Data** · Gelombang: 7 · Bergantung pada: TKT-U2-001, TKT-U2-002, TKT-F4-001
Requirement: FR-U6    Keputusan: D-59    ADR: 0012, 0023    Risiko: —
Rule Pega yang digantikan: pola bersama harness master — `MasterRekening`, `MasterSupplier`, `MasterRecovery`, `MasterPanel_HE`, `DetailCauseOfLoss`, `DetailDominanFactor`, `GCNMCatSparepart`, `SparePart_HE`, `BengkelHE`
Peran penguji gerbang 2: **PncAdmin**, **PncManagerAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu layar pengelolaan master yang benar, dipakai ulang **≥29 kali**.

Nilai bisnisnya sama dengan `U-2` dan `U-3`: dengan sedikitnya 29 kelompok master, membuat tiap
layar sendiri-sendiri berarti 29 halaman yang perilakunya berbeda-beda. Satu layar baku berarti
menambah kelompok master menjadi **pekerjaan konfigurasi**, bukan pekerjaan pemrograman.

## Ruang lingkup

- Layar baku: daftar bergrid, tambah, ubah, dan **penghapusan lunak** (`09-DATABASE` §8.1) —
  master tidak dihapus permanen karena data klaim lama merujuknya.
- Definisi kolom, validasi, dan kewenangan per kelompok sebagai **konfigurasi**.
- Setiap perubahan master **tercatat di jejak audit** (`S-5`) — siapa mengubah, dari apa menjadi apa.
- Pencarian dan paginasi dari server.

## Non-goal

- **Tidak** mendaftar kelompok master satu per satu — itu `TKT-U6-002`.
- **Tidak** memutuskan aturan bisnis isi master; itu `F-4`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Berapa persisnya kelompok master?** Yang ada baru **batas bawah: ≥29** | **Work Owner** | Modul yang tidak tahu berapa layar yang harus dibuatnya tidak dapat dinyatakan selesai secara terukur |
| **Setiap master butuh persetujuan sebelum berlaku, atau berlaku langsung?** | **Work Owner** | Sebagian master menentukan **hasil hitungan uang** — mengubahnya tanpa persetujuan mengubah nilai klaim yang dihitung sesudahnya |
| **Perubahan master berlaku surut ke klaim yang sudah ada?** | **Work Owner** | Menentukan apakah dibutuhkan versi bertanggal pada master |

## Acceptance criteria

- [ ] Layar master baku dipakai seluruh kelompok — tidak ada layar master buatan sendiri,
      diperiksa.
- [ ] Menambah kelompok master baru **tidak menuntut halaman baru** — diuji dengan menambah satu
      kelompok lewat konfigurasi saja.
- [ ] **Penghapusan bersifat lunak**; baris yang dirujuk klaim lama **tetap terbaca** — diuji
      dengan menghapus master yang dipakai klaim lama: klaim itu tetap tampil utuh.
- [ ] Setiap perubahan **tercatat di jejak audit** dengan nilai sebelum dan sesudah — diuji.
- [ ] Peran yang tidak berwenang **tidak dapat mengubah** master lewat pemanggilan langsung —
      diuji: `403` (`ADR-0023`).
- [ ] Tabel memakai `TKT-U2-001` dan form memakai `TKT-U2-002` — diperiksa.
- [ ] Gerbang 2: UAT **PncAdmin** dan **PncManagerAdmin**.

## Dependency / Blocked by

`TKT-U2-001` · `TKT-U2-002` · `TKT-F4-001` · `TKT-S5-001`. **Terhalang Work Owner.**

## Constraint keamanan, data, operasional

- **Master data menentukan hasil hitungan klaim.** Mengubah satu baris master dapat mengubah nilai
  yang dibayarkan — karena itu jejak auditnya bukan tambahan, melainkan syarat.
- **Penghapusan permanen akan merusak klaim lama** yang merujuk baris itu (`09-DATABASE` §8.1).
- Kontrol sekarang **berbasis menu** (`D-59`) — artinya siapa pun yang punya menu master dapat
  mengubah **seluruh** kelompok di dalamnya, bukan sebagian.

## Migrasi skema / rollout / rollback

Menambah kolom penanda hapus lunak pada tabel master yang belum punya. Backward-compatible (`P-4`).

**Rollback:** mengembalikan versi layar. Perubahan master yang telanjur dilakukan **tetap ada** —
dan telah memengaruhi klaim yang dihitung sesudahnya.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm test -- master --run TestSatuLayarBakuUntukSemuaKelompok
go test ./internal/app/master/... -run TestHapusLunakTidakMerusakKlaimLama
go test ./internal/app/audit/... -run TestPerubahanMasterTercatat
go test ./internal/adapter/http/... -run TestUbahMasterTanpaKewenanganDitolak
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Master data ≥29 kelompok; `U-6` naik menjadi Besar | `docs/Steering/06-MODULE-BREAKDOWN.md` koreksi ukuran 2026-09-14 |
| Penghapusan lunak pada master | `docs/Steering/09-DATABASE-STRATEGY.md` §8.1 |
| Kontrol berbasis menu | `D-59` |
| Otorisasi diperiksa di setiap endpoint | `ADR-0023` |
| Tabel baku median 6 kolom | `docs/Steering/06-MODULE-BREAKDOWN.md:76` |

## Comments
