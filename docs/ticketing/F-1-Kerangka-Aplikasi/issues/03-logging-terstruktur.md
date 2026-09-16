---
title: "TKT-F1-003 — Logging terstruktur dan ID permintaan"
labels: [modul::F-1, tipe::fondasi, status::ready-for-human, prioritas::sedang, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F1-003 — Logging terstruktur dan ID permintaan

Status: ready-for-human
Kesiapan: siap
Modul: F-1 · Gelombang: 1 · Bergantung pada: TKT-F1-001, TKT-F1-002
Requirement: FR-F1    Keputusan: D-69    ADR: 0026, 0029    Risiko: —
Rule Pega yang digantikan: — Pega menyediakan log platform; tidak ada rule aplikasi yang mengaturnya
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Log yang **dapat ditelusuri per permintaan** dan **dapat dibaca mesin**, sehingga satu keluhan
pengguna dapat dilacak sampai ke kueri yang gagal tanpa menebak.

Nilai bisnisnya paling terasa selama masa paralel: ketika Pega dan Go melayani klaim yang sama
(`ADR-0003`), pertanyaan pertama saat terjadi selisih selalu *"permintaan mana yang menghasilkan
ini"*. Tanpa ID permintaan, pertanyaan itu tidak terjawab.

## Ruang lingkup

- Logger terstruktur (JSON) dengan tingkat log dari konfigurasi.
- **ID permintaan** dibuat di titik masuk HTTP, dibawa lewat `context`, dan muncul di **setiap**
  baris log yang lahir dari permintaan itu.
- Baris log baku untuk: permintaan masuk, permintaan selesai (beserta lamanya), galat, dan
  pemanggilan sistem eksternal (tujuan, lama, hasil).
- **Aturan penyamaran**: daftar field yang nilainya tidak boleh pernah masuk log.

## Non-goal

- **Tidak** membangun jejak audit bisnis — itu `S-5`, dan keduanya **berbeda tujuan**: log untuk
  menelusuri masalah teknis, jejak audit untuk mempertanggungjawabkan perubahan bernilai bisnis.
- **Tidak** memasang agregator log atau dashboard — itu urusan infrastruktur.
- **Tidak** memutuskan retensi log.

## Acceptance criteria

- [ ] Setiap baris log berformat JSON dengan field wajib: waktu (UTC), tingkat, pesan,
      **`request_id`**, dan nama modul.
- [ ] Satu permintaan HTTP yang melewati tiga lapisan menghasilkan baris log dengan
      **`request_id` yang sama persis** di ketiganya — diuji otomatis.
- [ ] Tingkat log dapat diubah lewat konfigurasi tanpa mengubah kode — diuji dengan dua nilai.
- [ ] **Nol nilai sensitif di log**, diuji dengan permintaan yang memuat nomor polis, NPWP,
      nomor rekening, dan alamat email: keempatnya **tidak muncul** di keluaran log.
- [ ] Alamat email yang terpaksa dicatat muncul **tersamar** (bagian sebelum `@` diganti),
      konsisten dengan `D-69`.
- [ ] Pemanggilan sistem eksternal tercatat dengan tujuan, lama (ms), dan hasil — diuji dengan
      satu adapter tiruan yang sengaja gagal.

## Dependency / Blocked by

Bergantung pada `TKT-F1-001` dan `TKT-F1-002` (tingkat log dari konfigurasi).

## Constraint keamanan, data, operasional

- **Data nasabah tidak pernah masuk log**: nomor polis, nama tertanggung, NPWP, nomor rekening,
  dan data medis. Ini berlaku **juga di staging**, karena staging memuat data produksi apa adanya
  (`ADR-0029`).
- Log **tidak boleh** menjadi tempat menyimpan bukti perubahan bernilai bisnis — bila sebuah
  perubahan perlu dipertanggungjawabkan, tempatnya jejak audit `S-5`, bukan log.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** menurunkan tingkat log atau menonaktifkan field baru lewat
konfigurasi; tidak ada data yang berubah.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/platform/log/...
# uji kebocoran data: kirim permintaan berisi nilai penanda, lalu
go test ./internal/platform/log/... -run TestTidakAdaDataSensitifDiLog
# periksa manual bentuk log satu permintaan penuh:
go run ./cmd/app & curl -s localhost:8080/api/contoh ; # satu request_id di semua baris
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Logging terstruktur, ID permintaan, baris log baku | `docs/Steering/12-CROSSCUTTING.md` §2 |
| Aturan penyamaran email dan data nasabah | `D-69` · `ADR-0029` |
| Log berbeda tujuan dari jejak audit | `ADR-0026` · `docs/Steering/09-DATABASE-STRATEGY.md` §8 |
| Staging memuat data produksi apa adanya | `D-64` · `ADR-0029` |

## Comments
