---
title: "TKT-F4-006 — Inventaris ≥29 kelompok master dan penetapan lingkup"
labels: [modul::F-4, tipe::analisis, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F4-006 — Inventaris ≥29 kelompok master dan penetapan lingkup

Status: needs-info
Kesiapan: **terhalang keputusan** — mana dari ≥29 kelompok yang masuk lingkup belum ditetapkan
Modul: F-4 · Gelombang: 1 · Bergantung pada: —
Requirement: FR-F4    Keputusan: D-03, D-34, D-35    ADR: 0025    Risiko: —
Rule Pega yang digantikan: seluruh tabel master yang dirujuk 652 rule SQL; **14 kelompok sudah terpetakan**, sisanya belum
Peran penguji gerbang 2: **tidak berlaku** — tiket analisis (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Daftar lengkap kelompok master beserta **keputusan masuk atau tidaknya ke lingkup migrasi**,
sehingga `F-4` punya batas yang jelas.

Nilai bisnisnya adalah mencegah `F-4` menjadi modul tanpa ujung. Ukurannya sudah naik dari 14 ke
**≥29 kelompok**, dan tanda **≥** itulah masalahnya: selama batas atasnya tidak diketahui, estimasi
`F-4` maupun `U-6` tidak dapat dipertanggungjawabkan.

## Ruang lingkup

- Inventaris seluruh tabel master yang dirujuk rule, dikelompokkan menurut fungsi bisnisnya.
- Untuk setiap kelompok: **berapa rule yang membacanya**, apakah isinya berubah oleh pengguna
  bisnis, dan siapa pemiliknya.
- Usulan **masuk lingkup / tidak masuk lingkup** per kelompok, untuk diputuskan Work Owner.
- Hasil akhir: daftar tetap yang menjadi dasar tiket master berikutnya.

## Non-goal

- **Tidak** membangun master apa pun.
- **Tidak** memutuskan sendiri mana yang masuk lingkup — usulannya disiapkan, keputusannya milik
  Work Owner.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Mana dari ≥29 kelompok master yang masuk lingkup migrasi?** Sejauh ini baru satu area yang diputuskan: **Bengkel/Sparepart/Supplier dan ruleset GKM dikecualikan** (`D-34`) | **Work Owner** | Menentukan ukuran `F-4` dan `U-6`, dan karenanya menentukan kebutuhan orang |

**Catatan yang relevan bagi keputusan itu.** `D-03` menetapkan lingkup adalah **seluruh rule yang
ada di export**. Audit menemukan **35 rule** yang tidak dimiliki modul mana pun — 31 activity
Bengkel/Sparepart berruleset `GCNMFW` dan 4 activity master Supplier berruleset `GKM`. `D-34`
mengecualikan keduanya. Pertanyaan yang tersisa adalah **apakah ada area serupa lain** di antara
kelompok master yang belum terpetakan.

## Acceptance criteria

- [ ] Inventaris memuat **seluruh** kelompok master yang dirujuk rule, dengan jumlah yang dapat
      dihitung ulang dari export — bukan perkiraan.
- [ ] Setiap kelompok punya: nama tabel, jumlah rule pembaca, pemilik bisnis (atau **"belum
      diketahui"**), dan usulan masuk/tidak masuk lingkup.
- [ ] Kelompok yang sudah diputuskan **dikecualikan** (`D-34`) ditandai beserta rujukan
      keputusannya.
- [ ] Selisih terhadap angka 14 pada dokumen v1.0 dijelaskan: mana yang bertambah dan dari mana
      angkanya.
- [ ] Hasilnya **memperbarui `docs/Steering/05-DOMAIN-MODEL.md` §5** dan ukuran `F-4`/`U-6` di
      `06-MODULE-BREAKDOWN.md` — diajukan sebagai perubahan, bukan langsung diterapkan diam-diam.

## Dependency / Blocked by

Tidak bergantung pada tiket lain — **ini tiket analisis yang boleh dikerjakan paling awal**, dan
hasilnya membuka `TKT-F4-005` serta memperbaiki estimasi `U-6`.

**Terhalang keputusan Work Owner** hanya pada bagian penetapan lingkupnya; **inventarisnya sendiri
tidak terhalang**.

## Constraint keamanan, data, operasional

- Inventaris dibuat dari **pembacaan export**, bukan dari koneksi database. Tidak ada akses
  produksi.
- Nama tabel dan kolom boleh ditulis; **isi datanya tidak** (`D-69`).

## Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema — tiket ini menghasilkan dokumen.

**Rollback:** tidak berlaku.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
# hitung tabel master yang dirujuk rule SQL, kelompokkan
go run ./cmd/tools/inventaris-master "RDB List/" > inventaris-master.csv
wc -l inventaris-master.csv
# bandingkan dengan 14 kelompok yang sudah terpetakan
go run ./cmd/tools/banding-master inventaris-master.csv docs/Steering/05-DOMAIN-MODEL.md
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| ≥29 kelompok master, bukan 14 | `T-13` · `docs/Steering/05-DOMAIN-MODEL.md` §5 |
| Lingkup adalah seluruh rule yang ada di export | `D-03` |
| Bengkel/Sparepart/Supplier dan ruleset GKM dikecualikan | `D-34` |
| 35 rule tanpa modul pemilik | `D-34` · `docs/verifikasi-bukti-adr.md` |
| Angka mengikuti bukti, bukan dokumen | `D-35` |

## Comments
