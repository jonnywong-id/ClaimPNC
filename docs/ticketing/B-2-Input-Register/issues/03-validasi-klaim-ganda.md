---
title: "TKT-B02-003 — Validasi klaim ganda"
labels: [modul::B-2, tipe::aturan-bisnis, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B02-003 — Validasi klaim ganda

Status: needs-info
Kesiapan: terhalang keputusan
Modul: **B-2 Input Register** · Gelombang: 3 · Bergantung pada: TKT-B02-001
Requirement: FR-B2    Keputusan: D-18    ADR: 0018    Risiko: —
Rule Pega yang digantikan: pemeriksaan duplikasi di `Activity/InputRegister_act-Act.xml` — kunci **Polis + Objek + Lokasi**, ditambah **Penyebab Kerugian `12002`** khusus lini PA
Peran penguji gerbang 2: **PncAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Sistem menolak pendaftaran klaim yang **sudah pernah didaftarkan** untuk kejadian yang sama.

Nilai bisnisnya langsung ke uang: klaim ganda yang lolos berarti **satu kerugian dibayar dua
kali**. Pemeriksaan ini adalah satu-satunya pagar terhadap hal itu pada titik masuk.

## Ruang lingkup

- Pemeriksaan duplikasi dengan kunci **Polis + Objek Pertanggungan + Lokasi**.
- Kunci tambahan **Penyebab Kerugian `12002`** untuk lini Personal Accident.
- Perilaku saat ditemukan: **menolak** dengan pesan yang menyebut **nomor klaim yang sudah ada**,
  sehingga petugas dapat memeriksanya, bukan sekadar ditolak.
- Pemeriksaan menyertakan klaim yang dibuat **kedua sistem** selama masa paralel — klaim berformat
  `PNC-xxxx` (Pega) maupun `PNCN.YY.xxxx` (Go).

## Non-goal

- **Tidak** menggabungkan klaim ganda yang telanjur ada di data historis.
- **Tidak** memeriksa duplikasi lintas polis.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Apakah tanggal kejadian bagian dari kunci duplikasi?** Source tidak menyertakannya; artinya dua kejadian berbeda pada objek dan lokasi yang sama **dianggap duplikat** | **Work Owner** | Menentukan apakah klaim kedua yang sah ikut tertolak. Ini perilaku yang terlihat petugas setiap hari |
| **Lokasi tidak diperiksa untuk PA dan Travel — sengaja?** | **Work Owner** | Bila sengaja, aturannya memang berbeda per lini dan harus ditiru; bila tidak, ini celah yang sudah berjalan |

## Acceptance criteria

- [ ] Klaim kedua dengan **Polis + Objek + Lokasi** yang sama **ditolak**, dan pesannya menyebut
      nomor klaim yang sudah ada — diuji.
- [ ] Untuk lini **PA**, klaim kedua dengan kunci sama **tetapi Penyebab Kerugian berbeda dari
      `12002`** diterima — diuji.
- [ ] Pemeriksaan menemukan duplikat yang dibuat **Pega** maupun **Go** — diuji dengan satu klaim
      dari masing-masing sistem.
- [ ] Klaim yang sudah **ditandai terhapus** (soft delete, `ADR-0012`) **tidak** dihitung sebagai
      duplikat — diuji.
- [ ] Pemeriksaan memakai index; rencana eksekusinya tidak memindai seluruh tabel klaim —
      dilampirkan keluaran `EXPLAIN PLAN`.
- [ ] Gerbang 1: hasil pemeriksaan **sama dengan Pega** pada 30 kasus data staging, termasuk 10
      yang di Pega dinyatakan duplikat.

## Dependency / Blocked by

`TKT-B02-001` · `TKT-F2-005` (soft delete memengaruhi hasil pemeriksaan).

## Constraint keamanan, data, operasional

- Pemeriksaan berjalan pada **tabel yang ditulis dua sistem** selama masa paralel — ia membaca
  klaim Pega dan klaim Go sekaligus. Itu sah karena `P-1` membatasi **penulisan**, bukan
  pembacaan.
- Pesan galat menyebut nomor klaim; **tidak** menyebut nama tertanggung atau data nasabah lain.

## Migrasi skema / rollout / rollback

Menambah index pada kunci duplikasi — **tambah index bersifat backward-compatible** dan tidak
mengganggu Pega, tetapi pada tabel puluhan juta baris ia menuntut jendela pemeliharaan
(`TKT-F2-004`, statistik ukuran dari DBA).

**Rollback:** menghapus index; aturan tetap berjalan, hanya lebih lambat.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/registrasi/... -run TestKlaimGanda
go test ./internal/domain/registrasi/... -run TestKlaimGandaLintasSistem
go test ./internal/domain/registrasi/... -run TestSoftDeleteTidakDihitungDuplikat
go run ./cmd/s8 banding --modul B-2 --aturan duplikasi --kasus 30
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Kunci duplikasi Polis + Objek + Lokasi; PA + `12002` | Invarian `I-8`, `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Dua format nomor klaim hidup berdampingan | `ADR-0009` |
| Soft delete menyeluruh | `D-66` · `ADR-0012` |
| Penulis tunggal per tabel membatasi penulisan, bukan pembacaan | `P-1` · `ADR-0004` |

## Comments
