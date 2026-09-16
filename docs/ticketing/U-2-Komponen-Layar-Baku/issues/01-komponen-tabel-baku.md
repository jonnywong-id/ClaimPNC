---
title: "TKT-U2-001 — Komponen tabel baku dengan paginasi keyset server-side"
labels: [modul::U-2, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::2]
milestone: "Gelombang 2 — Kerangka UI"
epic: "Migrasi Claim PNC"
---

# TKT-U2-001 — Komponen tabel baku dengan paginasi keyset server-side

Status: ready-for-human
Kesiapan: siap
Modul: U-2 · Gelombang: 2 · Bergantung pada: TKT-U2-005, TKT-U1-001
Requirement: FR-U2    Keputusan: D-13, D-23    ADR: 0002    Risiko: R-11
Rule Pega yang digantikan: pola grid pada **268 dari 269 section** — di antaranya **3.189 grid terikat page list klipboard**
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi UI (`D-60`); perilakunya diuji pengguna lewat `U-3` dan `U-4`

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu komponen tabel yang dipakai seluruh layar daftar, dengan paginasi, pengurutan, dan
penyaringan yang **dikerjakan di server**.

Nilai bisnisnya berlipat: satu komponen dipakai ratusan kali, dan satu perbaikan di dalamnya
memperbaiki seluruh layar. Ia juga tempat satu-satunya yang menegakkan aturan keamanan daftar —
nama kolom sort dan filter hanya dari daftar putih (`TKT-F2-007`), bukan dari apa yang dikirim
peramban.

## Ruang lingkup

- Komponen tabel dengan: paginasi **keyset** server-side, pengurutan server-side, penyaringan
  server-side, keadaan kosong, keadaan memuat, dan keadaan galat.
- Kolom dideklarasikan sebagai data, bukan JSX berulang — sehingga 268 layar mendeklarasikan
  kolom, bukan menulis ulang tabel.
- Pemformatan sel memakai pemformat terpusat `TKT-U2-004` — **tidak ada** pemformatan tanggal
  atau uang yang ditulis ulang per layar.
- Perilaku pada **grid lebar** (39 kolom): gulir horizontal yang benar dengan kolom identitas yang
  tetap terlihat.

## Non-goal

- **Tidak** menambah baris atau menghapus baris **inline** — ketiga fitur itu **tidak dipakai
  sama sekali** di sistem lama, dan menambahkannya berarti menambah kemampuan yang tidak diminta.
  Bila Work Owner menginginkannya, itu tiket baru.
- **Tidak** membangun layar apa pun — itu `U-3`, `U-4`, `U-5`.
- **Tidak** melakukan penyaringan di sisi klien atas data yang sudah diambil.

## Acceptance criteria

- [ ] Tabel menampilkan 10.000 baris lewat paginasi server-side **tanpa memuat seluruhnya** —
      diuji dengan memeriksa jumlah baris yang diminta per permintaan.
- [ ] Paginasi memakai **keyset**, bukan `OFFSET` — diuji dengan memeriksa parameter permintaan
      halaman kedua memuat penanda baris terakhir, bukan angka lompatan.
- [ ] Permintaan sort pada kolom **di luar daftar putih** ditolak dan ditampilkan sebagai galat
      yang dapat dibaca pengguna — bukan tabel kosong tanpa penjelasan.
- [ ] Keadaan kosong, memuat, dan galat masing-masing punya tampilan yang berbeda dan **dapat
      dibedakan** — diuji ketiganya.
- [ ] Grid **39 kolom** dapat digulir horizontal dengan kolom identitas tetap terlihat — diuji
      pada lebar layar 1366 px.
- [ ] Mendeklarasikan tabel baru untuk 6 kolom membutuhkan **≤ 30 baris kode** — dibuktikan
      dengan satu contoh nyata.
- [ ] Tidak ada pemformatan tanggal atau uang di dalam komponen layar — diuji pemindaian:
      pemanggilan pemformat hanya lewat `TKT-U2-004`.

## Dependency / Blocked by

- Bergantung pada `TKT-U2-005` (pustaka tabel harus dipilih lebih dulu).
- Bergantung pada `TKT-U1-001` (kerangka SPA).
- Bergantung secara kontrak pada `TKT-F2-007` (daftar putih kolom) — komponen ini yang
  menegakkannya di sisi klien; server tetap menjadi penegak sebenarnya.

## Constraint keamanan, data, operasional

- **Penyaringan dan pengurutan tidak pernah dikerjakan di klien** atas data yang sudah diambil —
  itu membocorkan baris yang seharusnya tidak terlihat lewat jumlah baris (batas data `F-3`).
- Nama kolom database **tidak boleh muncul** di parameter URL maupun di DOM.
- Menghapus batas 500 baris sistem lama adalah **penambahan kemampuan**, bukan penyalinan —
  kapasitas maksimum satu daftar mengikuti keputusan `ADR-0011` yang masih terbuka.

## Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** mengembalikan versi komponen; layar yang
memakainya ikut kembali karena hanya ada satu implementasi.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm run test -- DataTable
npm run test -- DataTable.keyset
npm run test:e2e -- --grep "tabel 39 kolom"
npx bundlesize   # ukuran bundel setelah komponen ditambahkan
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 268 dari 269 section bergrid | `docs/Steering/06-MODULE-BREAKDOWN.md` §4 |
| 3.189 grid terikat page list klipboard | `T-12` · `docs/Steering/09-DATABASE-STRATEGY.md` §6.3 |
| Tambah/hapus baris inline dan resize **tidak dipakai** | `T-11` |
| Median 6 kolom; kasus terberat 39 dan 25 kolom | `T-11` · `docs/verifikasi-bukti-adr.md` §15 |
| Paginasi keyset adalah perubahan perilaku | `docs/Steering/09-DATABASE-STRATEGY.md` §6.3 |
| Tata letak mengikuti Pega | `D-13` |

## Comments
