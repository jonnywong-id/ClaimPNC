---
title: "TKT-S8-001 — Penembak kasus dan perekam hasil"
labels: [modul::S-8, tipe::fondasi, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-S8-001 — Penembak kasus dan perekam hasil

Status: needs-info
Kesiapan: **terhalang ketersediaan Pega staging**
Modul: **S-8 Perkakas Uji Kesetaraan** · Gelombang: 1 · Bergantung pada: TKT-F1-001, TKT-F2-001
Requirement: FR-S8    Keputusan: D-42    ADR: 0027    Risiko: R-14
Rule Pega yang digantikan: **tidak ada** — modul ini 100% baru
Peran penguji gerbang 2: **Lead Engineer** (modul ini tidak dipakai peran bisnis)

## Hasil yang diharapkan (dan nilai bisnisnya)

Perkakas yang menjalankan satu daftar kasus di **dua sistem** — Pega staging dan Go staging — lalu
merekam hasil keduanya dalam bentuk yang dapat dibandingkan.

Nilai bisnisnya adalah nilai seluruh proyek: `BRD §21.1` menjadikan uji kesetaraan sebagai gerbang
pertama setiap modul. **Tanpa perkakas ini, tidak ada satu modul pun yang dapat dinyatakan lulus** —
dan tidak ada satu pun bagian Pega yang boleh dimatikan (`D-42`).

## Ruang lingkup

- Definisi **kasus uji** dalam bentuk data, bukan kode — agar penambahannya tidak menuntut rilis.
- Penjalan kasus ke **Pega staging** dan ke **Go staging**.
- Perekaman hasil kedua sisi dalam bentuk yang stabil dan dapat dibandingkan ulang.
- Penjalanan ulang kasus yang sama menghasilkan rekaman yang sama bila sistemnya tidak berubah.

## Non-goal

- **Tidak** membandingkan maupun mengklasifikasikan selisih — itu `TKT-S8-002`.
- **Tidak** menyentuh produksi. Tidak ada koneksi, tidak ada DDL, tidak ada DML.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Pega staging yang dapat ditembak dari luar — ada?** | **Tim Pega + Infra** (`R-14`, `ADR-0027`) | Ini **penghalang utama `S-8`**, dan karenanya penghalang gerbang 1 seluruh modul. Tanpa sisi Pega, tidak ada yang dapat dibandingkan |
| **Siapa boleh menembak Pega, di lingkungan mana?** | **Work Owner + Tim Infra/Security** (`D-42`) | Belum pernah ditanyakan sampai `D-42` |
| **Boleh membaca produksi read-only untuk mengambil kasus nyata?** | **Work Owner + Tim Infra/Security** (`D-42`) | Menentukan apakah kasus uji berasal dari data nyata atau data buatan — dan keduanya menghasilkan kepercayaan yang berbeda |
| **Kasus uji diambil dari mana, dan berapa banyak per modul?** | **Work Owner + Lead Engineer** | Tanpa jumlah minimum, "lulus gerbang 1" tidak punya arti terukur |

## Acceptance criteria

- [ ] Satu kasus dapat dijalankan di **kedua sistem** dan hasilnya terekam — diuji dari ujung ke ujung.
- [ ] Kasus didefinisikan **sebagai data**; menambah kasus **tidak menuntut rilis** — diuji.
- [ ] Menjalankan kasus yang sama dua kali pada sistem yang tidak berubah menghasilkan rekaman
      **yang sama** — diuji; bila tidak, pembanding akan melaporkan selisih palsu.
- [ ] Perkakas **tidak pernah menulis ke produksi** — dipastikan lewat konfigurasi yang menolak
      alamat produksi, dan diuji dengan mencoba mengarahkannya ke sana: **ditolak**.
- [ ] Kegagalan menembak salah satu sisi **dilaporkan sebagai kegagalan**, bukan sebagai "tidak ada
      selisih" — diuji dengan mematikan salah satu sisi.
- [ ] Gerbang 2: ditinjau **Lead Engineer**; modul ini tidak dipakai peran bisnis.

## Dependency / Blocked by

`TKT-F1-001` · `TKT-F2-001`. **Terhalang Tim Pega + Infra (`R-14`) — di luar kendali proyek.**

## Constraint keamanan, data, operasional

- **Tidak ada koneksi ke produksi. Tidak ada DDL/DML.** Batas ini mutlak dan tidak dapat dilonggarkan
  demi kemudahan pengujian.
- Staging memuat **data produksi apa adanya** (`D-64`) — kasus uji dan rekaman hasilnya karena itu
  memuat **data nasabah nyata**. Rekaman **tidak boleh di-commit**, dan pembatasan data medis
  (`FR-R2`) berlaku pada isinya.
- Menembak Pega staging menambah beban ke sistem yang masih dipakai untuk pengujian lain.

## Migrasi skema / rollout / rollback

Menambah tabel kasus uji dan rekaman hasil **di lingkungan uji**, bukan di skema klaim.

**Rollback:** perkakas ini tidak dipakai pengguna; mengembalikan versinya tidak berdampak ke bisnis.
Tetapi **mematikannya menghentikan gerbang 1 seluruh modul**.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./cmd/s8/... -run TestKasusSebagaiData
go test ./cmd/s8/... -run TestRekamanStabilSaatDiulang
go test ./cmd/s8/... -run TestMenolakAlamatProduksi
go test ./cmd/s8/... -run TestSisiMatiDilaporkanSebagaiGagal
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `S-8` modul 100% baru, gelombang 1 | `D-42` · `docs/Steering/06-MODULE-BREAKDOWN.md:67` |
| `S-8` memblokir gerbang 1 seluruh modul | `D-42` alasan · `BRD §21.1` |
| Ketersediaan Pega staging belum dipastikan | `R-14` · `docs/Steering/16-RISK-ANALYSIS.md:562` · `ADR-0027` |
| Kewenangan menembak Pega belum ditanyakan | `D-42` bagian Terbuka |
| Staging memuat data produksi apa adanya | `D-64` · `ADR-0029` |

## Comments
