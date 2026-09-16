---
title: "TKT-U4-002 — Layar pencarian dan detail klaim"
labels: [modul::U-4, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-U4-002 — Layar pencarian dan detail klaim

Status: needs-info
Kesiapan: **terhalang keputusan batas data**
Modul: **U-4 Layar Transaksi Klaim** · Gelombang: 3 · Bergantung pada: TKT-U4-001, TKT-F3-005
Requirement: FR-U4    Keputusan: D-59, D-71    ADR: 0012, 0023    Risiko: —
Rule Pega yang digantikan: harness `PNCSearchKlaim`, `View_DetailKlaimCabang_Harness`, `ViewTempDetailClaim`, `ViewTempDetailAllCase`, `ProgressClaim_Harness`, `StatusProgress`, `StatusProgress2`
Peran penguji gerbang 2: **PncRegister**, **PncKacab**, **PncManagerAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Layar untuk **menemukan klaim** dan **melihat seluruh keadaannya dalam satu tempat**.

Nilai bisnisnya: pencarian adalah pintu masuk ke hampir semua pekerjaan. Petugas datang dengan
nomor klaim atau nomor polis dari telepon nasabah, dan bila pencariannya lambat atau tidak
menemukan, seluruh percakapan itu tertahan.

## Ruang lingkup

- Pencarian klaim berdasarkan **nomor klaim** (format `PNCN.YY.xxxx`, `D-71`), nomor polis,
  nama tertanggung, tanggal, cabang, dan status.
- Layar detail klaim: ringkasan polis, objek pertanggungan, estimasi, riwayat tahap, dokumen.
- **Batas data ditegakkan di server**: petugas cabang menemukan klaim cabangnya (`TKT-F3-005`).
- Paginasi hasil pencarian **dari server**.

## Non-goal

- **Tidak** membangun layar per tahap — itu `TKT-U4-001`.
- **Tidak** menambah kriteria pencarian yang tidak ada di sistem lama.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Siapa boleh mencari klaim lintas cabang?** | **Work Owner** (`D-59`) | Kontrol sekarang berbasis menu, bukan berbutir aksi — sehingga batas data pencarian **belum tentu sama dengan batas menu**. Salah menetapkannya membuka seluruh portofolio klaim kepada peran cabang |
| **Pencarian tanpa kriteria mengembalikan apa?** | **Work Owner** | Di Pega, laporan memotong di 500 baris tanpa memberi tahu. Bila pencarian mengembalikan semuanya, ia menjadi cara mengunduh seluruh basis klaim lewat layar |
| **Nomor klaim lama berformat berbeda — tetap dapat dicari?** | **Work Owner** (`D-71`) | `D-71` menetapkan format `PNCN.YY.xxxx` untuk yang baru; klaim lama tetap ada di data |

## Acceptance criteria

- [ ] Pencarian menemukan klaim berdasarkan seluruh kriteria yang ada di `PNCSearchKlaim` —
      diperiksa per kriteria.
- [ ] **Klaim berformat nomor lama tetap ditemukan** — diuji dengan kedua format (`D-71`).
- [ ] **Batas data ditegakkan di server** — diuji: peran cabang A memanggil endpoint pencarian
      untuk klaim cabang B dan **tidak menerima datanya**, bukan sekadar tidak melihatnya.
- [ ] Paginasi hasil dilakukan **di server**; hasil besar **tidak dipotong diam-diam** — bila
      melebihi batas, pengguna **diberi tahu**.
- [ ] Layar detail menampilkan data yang **sama dengan Pega** untuk klaim yang sama — dibandingkan
      lewat `S-8` pada 20 klaim, termasuk klaim dengan banyak objek pertanggungan.
- [ ] Angka uang dan tanggal memakai `TKT-U2-004` — sama dengan tampilan di laporan.
- [ ] Klaim dengan **data medis** hanya menampilkan bagian itu kepada peran yang berwenang
      (`FR-R2`) — diuji dengan peran lain.
- [ ] Gerbang 2: UAT **PncRegister** dan **PncKacab** secara terpisah — pemisahan batas datanya
      yang diuji.

## Dependency / Blocked by

`TKT-U4-001` · `TKT-F3-005` · `TKT-B01-001` (snapshot polis). **Terhalang keputusan batas data.**

## Constraint keamanan, data, operasional

- **Pencarian adalah cara termudah memintas batas data** bila penyaringnya hanya di layar.
- Pencarian tanpa batas atas dapat menjadi **jalan mengunduh seluruh basis klaim** — masalah yang
  sama dengan `TKT-S2-002`, hanya lewat pintu berbeda.
- Pencarian berjalan di atas tabel klaim yang sama yang dipakai transaksi; kueri yang buruk di sini
  memperlambat seluruh pengguna (`ADR-0001`).

## Migrasi skema / rollout / rollback

Tidak menyentuh kolom; kemungkinan menambah index pencarian (`TKT-F2-004`) — penambahan index
bersifat backward-compatible (`P-4`).

**Rollback:** petugas kembali memakai pencarian Pega (`D-05`).

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/http/... -run TestBatasDataPencarianDiServer
go test ./internal/app/klaim/... -run TestNomorKlaimFormatLamaDanBaru
npm test -- pencarian --run TestPaginasiDariServer
go run ./cmd/s8 banding --modul U-4 --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Harness pencarian dan detail klaim | direktori `Harness/` |
| Format nomor klaim `PNCN.YY.xxxx` | `D-71` |
| Kontrol berbasis menu, bukan berbutir aksi | `D-59` |
| Otorisasi diperiksa di setiap endpoint | `ADR-0023` |
| Laporan lama memotong di 500 baris tanpa pemberitahuan | `T-12` · `ADR-0011` |

## Comments
