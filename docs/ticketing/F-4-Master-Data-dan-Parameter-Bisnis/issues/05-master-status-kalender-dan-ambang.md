---
title: "TKT-F4-005 — Master Status Klaim, kalender libur, dan ambang uang"
labels: [modul::F-4, tipe::migrasi, status::needs-info, prioritas::sedang, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F4-005 — Master Status Klaim, kalender libur, dan ambang uang

Status: needs-info
Kesiapan: **terhalang keputusan** — dua pertanyaan lingkup belum dijawab
Modul: F-4 · Gelombang: 1 · Bergantung pada: TKT-F4-001
Requirement: FR-F4    Keputusan: D-15, D-18, D-50, D-62    ADR: 0018, 0020, 0025, 0026    Risiko: —
Rule Pega yang digantikan: `V_STS_CLAIM` (**33 kode `1134`–`1166`**) · `HRD_LBR` lewat DB Link · ambang uang non-komite yang tertanam — `Activity/InputRegister_act-Act.xml:19110` (Notice of Large Losses `1000000000`), `Activity/CheckEstimateValue-Act.xml:9617`, `:9831` (`50000000000`), `Activity/SetListComiteeClaimAI-Act.xml:11602`, `:11742`
Peran penguji gerbang 2: **User Admin** dan **PncManagerAdmin** (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Tiga master yang sifatnya sama — daftar nilai referensi yang menentukan perilaku — dikerjakan
dalam satu tiket karena polanya identik: status klaim, kalender hari libur, dan ambang uang
non-komite.

Nilai bisnisnya: **33 kode status** kini diketahui artinya, dan kalender libur menentukan angka
TAT yang dilaporkan ke manajemen. Keduanya hari ini hidup di tempat yang tidak dapat diubah
pengguna bisnis.

## Ruang lingkup

- **Master Status Klaim**: 33 kode `1134`–`1166` beserta labelnya, termasuk penomoran lama `01`–`11`
  yang melekat pada sebelas kode pertama.
- **Master Hari Libur dan Jam Kerja**: menggantikan `HRD_LBR` dan definisi jam kerja yang kini
  berada di dalam fungsi remote (`ADR-0020`).
- **Master Ambang Uang non-komite**: Notice of Large Losses (`1000000000`) dan ambang lain yang
  tertanam.
- **Master Retensi**: lama simpan data klaim dan jejak audit — satu kebijakan untuk keduanya
  (`D-62`), nilainya sebagai parameter sampai angkanya ada.

## Non-goal

- **Tidak** mengubah arti kode status. `D-18` menetapkan keempat konsep status dipertahankan.
- **Tidak** membangun perhitungan TAT — itu `TKT-F5-003` dan `S-7`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Ambang Large Losses ada fiturnya di sistem baru?** Di sistem lama ia memicu Notice of Large Losses ke Underwriting dan jajaran pimpinan | **Work Owner** | Bila tidak dibawa, master ambang ini tidak dibuat sama sekali |
| **Batas aturan tanggal per lini bisnis (7/30/90 hari) — kebutuhan baru atau sudah ada?** | **Work Owner** | Menentukan apakah ia master atau aturan tetap di kode |
| **Dari mana daftar hari libur diperoleh setiap tahun, dan siapa mengisinya?** | **Work Owner** | Sama dengan `TKT-F5-003` — satu jawaban menutup keduanya |
| **Angka retensi data klaim yang berlaku sekarang** | **Work Owner + Compliance** | Tidak menahan pembangunan (dibuat parameter), **menahan go-live** |

## Acceptance criteria

- [ ] Master status memuat **tepat 33 kode** `1134`–`1166` beserta labelnya — dihitung dan
      dilaporkan angkanya.
- [ ] Sebelas kode pertama (`1134`–`1144`) menyimpan **penomoran lama `01`–`11`** sebagai kolom
      terpisah, sehingga data historis tetap terbaca.
- [ ] Kode status **tidak tertanam di kode** — diuji pemindaian terhadap keempat digit yang
      diketahui.
- [ ] Master hari libur dapat diisi per tahun, dan mengubahnya **mengubah hasil perhitungan jam
      kerja** — diuji bersama `TKT-F5-003`.
- [ ] Ambang uang non-komite dibaca dari master — diuji dengan mengubah ambang Large Losses dan
      memeriksa pemicunya berubah.
- [ ] Retensi tersedia sebagai parameter dengan **nilai bawaan kosong** yang berarti tidak ada
      yang diarsipkan (`TKT-S5-004`).

## Dependency / Blocked by

Bergantung pada `TKT-F4-001`. **Terhalang empat keputusan** di atas — tetapi bagian **Master Status
Klaim** tidak terhalang dan boleh dikerjakan lebih dulu.

## Constraint keamanan, data, operasional

- Kode status menentukan perilaku layar dan laporan; perubahannya **wajib tercatat** di jejak
  audit (`TKT-F4-001`).
- **Arti kode status tidak boleh disimpulkan dari pemakaian.** Tiga arti yang sempat disimpulkan
  dari rule **seluruhnya salah**: `1143` bukan status awal melainkan Close Claim, `1150` bukan
  penanda terdaftar melainkan LOD Report, `1151` bukan Investigator melainkan Analyst.
- Kalender libur memengaruhi **angka yang dilaporkan ke manajemen** — perubahannya perlu terlihat.

## Migrasi skema / rollout / rollback

Menambah tabel master baru — tidak menyentuh tabel Pega. `V_STS_CLAIM` **tetap dibaca Pega**
selama masa paralel.

**Rollback:** aplikasi Go kembali membaca `V_STS_CLAIM` langsung.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/cek-master-status --harap 33      # HARUS 33 kode
grep -rInE "\"11[3-6][0-9]\"" internal/ | grep -v masterdata   # HARUS 0 baris
go test ./internal/app/masterdata/... -run TestAmbangLargeLosses
go test ./internal/domain/waktu/... -run TestHariLiburDariMaster
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 33 kode `1134`–`1166`; sebelas pertama membawa penomoran lama | `R-06` tertutup · `CONTEXT.md` · `ADR-0018` |
| Tiga arti kode yang disimpulkan ternyata salah | `docs/Steering/16-RISK-ANALYSIS.md` `R-06` |
| Ambang Large Losses `1000000000` | `Activity/InputRegister_act-Act.xml:19110` |
| `HRD_LBR` dan jam kerja lewat DB Link; ditulis ulang di Go | `D-50` · `ADR-0020` |
| Retensi satu kebijakan untuk klaim dan audit | `D-62` |

## Comments
