---
title: "TKT-B05-002 — Konversi kurs dan perbandingan terhadap ambang"
labels: [modul::B-5, tipe::aturan-bisnis, status::needs-info, prioritas::tinggi, gelombang::5]
milestone: "Gelombang 5 — Nilai dan pihak luar"
epic: "Migrasi Claim PNC"
---

# TKT-B05-002 — Konversi kurs dan perbandingan terhadap ambang

Status: needs-info
Kesiapan: **terhalang artefak** — isi `m_currencystandard` belum ada
Modul: **B-5 Input Estimasi** · Gelombang: 5 · Bergantung pada: TKT-B05-001, TKT-F4-004
Requirement: FR-B5    Keputusan: D-48, D-49 butir 4 dan 5    ADR: 0015, 0016, 0017    Risiko: R-19
Rule Pega yang digantikan: `Database/GETCURRENCYSTANDARD.fnc` — kurs **hari eksekusi** (`:3` versus `:14`) dan **`RETURN 1`** saat kurs tidak ditemukan (`:20-22`)
Peran penguji gerbang 2: **PncPICTeknik** dan **PNCKomite**

## Hasil yang diharapkan (dan nilai bisnisnya)

Nilai klaim valuta asing dikonversi ke Rupiah memakai **kurs pada tanggal kejadian**, dan hasil
konversi itulah yang dibandingkan dengan ambang komite.

Nilai bisnisnya adalah menutup cacat yang **membuat klaim besar lolos tanpa komite**. Fungsi kurs
lama mengembalikan **`1`** ketika kurs tidak ditemukan: satu satuan valuta asing dihitung setara
satu Rupiah, sehingga klaim bernilai besar menyusut menjadi kecil dan **lolos di bawah seluruh
ambang**. Tanpa galat, tanpa catatan, dan angkanya tampak wajar.

## Ruang lingkup

- Konversi nilai settlement ke Rupiah memakai **kurs pada tanggal kejadian** (`ADR-0015`).
- **Menolak** klaim bila kurs untuk mata uang dan tanggal itu tidak ditemukan — dengan galat yang
  menyebutkan **mata uang dan tanggalnya**; tanpa nilai bawaan.
- Hasil konversi dipakai sebagai dasar perbandingan ambang komite (`B-7`) dan ambang Notice of
  Large Losses (`B-2`).
- Perbandingan memakai **nilai presisi penuh**, bukan nilai yang sudah dibulatkan tampilan.

## Non-goal

- **Tidak** mengelola master kurs — itu `TKT-F4-004`.
- **Tidak** menentukan jenjang komite — itu `B-7`; modul ini menyediakan angkanya.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Isi `m_currencystandard`** | **DBA** (`R-19`) | Tanpa isinya, konversi tidak dapat diuji sama sekali — dan inilah salah satu dari tiga penghalang yang membuat `B-5` masih terikat `BRD §21.4` |
| **Bagaimana klaim yang tertolak karena kurs kosong diperlakukan** — ditahan dan diproses ulang otomatis, atau dikembalikan ke petugas? | **Work Owner** | Menentukan perilaku yang dilihat petugas setiap hari |

## Acceptance criteria

- [ ] Konversi memakai kurs **tanggal kejadian**, bukan tanggal proses — diuji dengan klaim yang
      tanggal kejadiannya berbeda dari hari pengujian.
- [ ] Nilai Rupiah sebuah klaim **tidak berubah** saat klaim yang sama diproses ulang di hari lain
      — diuji. Inilah bukti butir 4 `P-5` tertutup.
- [ ] Kurs tidak ditemukan → **klaim ditolak** dengan galat yang menyebut mata uang dan tanggal;
      **nilai `1` tidak muncul di jalur mana pun** — diuji pemindaian dan uji fungsional. Inilah
      bukti butir 5 `P-5` tertutup.
- [ ] Perbandingan terhadap ambang memakai nilai presisi penuh — diuji dengan nilai yang berbeda
      hasilnya bila dibulatkan lebih dulu.
- [ ] Gerbang 1: **selisih muncul pada seluruh data historis valuta asing** — itu **diharapkan**,
      dan wajib terpetakan ke butir 4 dan 5 `P-5` (`D-54`). Selisih pada klaim Rupiah murni
      **wajib nol**.
- [ ] Gerbang 2: UAT **PncPICTeknik** dan **PNCKomite** pada klaim valuta asing.

## Dependency / Blocked by

`TKT-B05-001` · `TKT-F4-004` (master kurs) · `TKT-F5-001` (tanggal). **Terhalang DBA dan satu
keputusan.**

## Constraint keamanan, data, operasional

- **Dampak operasional yang harus disiapkan sebelum rilis:** klaim valuta asing yang dulu lolos
  kini **dapat ditolak** sampai kursnya dilengkapi. Ini perbaikan yang diinginkan, tetapi ia
  menghentikan pekerjaan petugas bila master kurs tidak terisi tepat waktu.
- Hasil konversi menentukan **berapa banyak orang yang harus menyetujui** klaim (`ADR-0014`) —
  kesalahannya berpindah langsung menjadi kesalahan kewenangan.

## Migrasi skema / rollout / rollback

Tidak menambah tabel; memakai master kurs `F-4`.

**Rollback:** kembali ke `GETCURRENCYSTANDARD` **tidak tersedia** — mengembalikannya berarti
memulihkan cacat `RETURN 1`.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/settlement/... -run TestKonversiKursTanggalKejadian
go test ./internal/domain/settlement/... -run TestKursTidakDitemukanMenolak
grep -rInE "return\s+1(\.0)?\s*,\s*nil" internal/domain/settlement/    # HARUS 0 baris
go run ./cmd/s8 banding --modul B-5 --aturan kurs
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Kurs hari eksekusi dan `RETURN 1` | `Database/GETCURRENCYSTANDARD.fnc:3`, `:14`, `:20-22` |
| Butir 4 dan 5 dari 13 perbaikan `P-5` | `D-49` · `ADR-0017` |
| Kurs tanggal kejadian; tolak bila tidak ada | `D-48` · `ADR-0015` |
| Isi `m_currencystandard` belum ada | `R-19` · `D-55` |
| Perbandingan ambang memakai presisi penuh | `D-51` · `ADR-0016` |

## Comments
