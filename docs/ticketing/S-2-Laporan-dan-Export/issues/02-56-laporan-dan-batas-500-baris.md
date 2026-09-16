---
title: "TKT-S2-002 — 56 laporan dan penghapusan batas 500 baris"
labels: [modul::S-2, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::6]
milestone: "Gelombang 6 — Laporan"
epic: "Migrasi Claim PNC"
---

# TKT-S2-002 — 56 laporan dan penghapusan batas 500 baris

Status: needs-info
Kesiapan: **terhalang artefak dan keputusan**
Modul: **S-2 Laporan & Export** · Gelombang: 6 · Bergantung pada: TKT-S2-001
Requirement: FR-S2    Keputusan: D-11    ADR: 0011    Risiko: R-16
Rule Pega yang digantikan: **56 Report Definition**, **54 di antaranya ber-`pyMaxRecords=500`** · **12 Report Definition hilang dari export** (`BrowseVPanel_HE_RD` **82 pemakaian**) · harness `PNCTATReport`, `ReportKPIHarness`, `OutstandingKlaimperCabang_Harness`, `MonitoringSLINKOJK`
Peran penguji gerbang 2: **PNCReportClaimInternal**, **PNCReportClaimEksternal**, **PncManagerAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Kelima puluh enam laporan tersedia di sistem baru, **tanpa pemotongan diam-diam di 500 baris**.

Nilai bisnisnya sekaligus peringatannya: batas 500 baris berarti laporan yang dilihat manajemen
selama ini **mungkin tidak lengkap** — dan menghapus batas itu akan **mengubah angka yang mereka
lihat**. Perubahan itu harus disampaikan lebih dulu, bukan ditemukan sebagai kejutan.

## Ruang lingkup

- Pemindahan 56 laporan, memakai engine `TKT-S2-001`.
- **Penghapusan batas 500 baris**, dengan penggantinya berupa batas yang disepakati dan
  **diberitahukan kepada pengguna** saat hasil melebihi.
- Penegakan batas data pada setiap laporan (`TKT-F3-005`).
- Daftar laporan beserta pemiliknya, sehingga setiap laporan punya penguji gerbang 2.

## Non-goal

- **Tidak** menambah laporan baru.
- **Tidak** merancang ulang isi laporan.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **12 Report Definition hilang dari export**, terberat `BrowseVPanel_HE_RD` (**82 pemakaian**) · 10 template HTML · 3 Correspondence | **Tim Pega** (`R-16`) | Isi dan penyaring laporan itu tidak diketahui |
| **54 dari 56 laporan memotong di 500 baris — pengguna tahu?** | **Work Owner** | Bila tidak tahu, menghapus batas akan mengubah angka yang mereka percayai selama ini. Ini **perubahan yang harus diumumkan**, bukan diam-diam |
| **Berapa batas baru, dan bagaimana pengguna diberi tahu bila hasil melebihi?** | **Work Owner** | Tanpa itu, laporan besar kembali terpotong diam-diam — hanya pada angka berbeda |

## Acceptance criteria

- [ ] Kelima puluh enam laporan tersedia, dan **jumlahnya dilaporkan sebagai angka** — bukan
      pernyataan "semua sudah ada".
- [ ] Laporan yang hasilnya melebihi batas **memberi tahu pengguna**, bukan memotong diam-diam —
      diuji dengan data melebihi batas.
- [ ] Batas data cabang dan lini bisnis ditegakkan pada setiap laporan — diuji pada 5 laporan
      dengan dua peran berbeda.
- [ ] Laporan yang memuat **data medis** hanya dapat dijalankan peran yang berwenang (`FR-R2`) —
      diuji.
- [ ] Gerbang 1: isi laporan **sama dengan Pega** pada 20 laporan contoh — **dengan pengecualian
      yang dinyatakan di muka**: laporan yang di Pega terpotong di 500 baris akan berbeda, dan
      selisih itu **wajib dilaporkan sebagai perbaikan terencana**, bukan bug.
- [ ] Gerbang 2: UAT **per pemilik laporan**, bukan satu orang untuk 56 laporan.

## Dependency / Blocked by

`TKT-S2-001` · seluruh modul bisnis (sumber datanya). **Terhalang Tim Pega dan dua keputusan.**

## Constraint keamanan, data, operasional

- **Menghapus batas 500 baris mengubah angka yang dilihat manajemen.** Ini bukan perbaikan teknis
  diam-diam — ia perubahan yang terlihat, dan harus diumumkan sebelum rilis.
- Laporan adalah jalan paling mudah untuk **memintas pembatasan data layar** bila batasnya tidak
  ditegakkan di kueri laporan juga.
- Kapasitas dirancang **tanpa data historis yang sahih**, karena angka pemakaian selama ini selalu
  terpotong di 500 (`docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md`).

## Migrasi skema / rollout / rollback

Tidak menyentuh skema; menambah index bila kueri laporan membutuhkannya (`TKT-F2-004`).

**Rollback:** mengembalikan batas 500 baris **mengembalikan pemotongan diam-diam** — bila
dilakukan, itu harus disampaikan ke pengguna, bukan dianggap netral.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/cek-laporan --harap 56          # HARUS 56
go test ./internal/app/laporan/... -run TestBatasDataPerLaporan
go test ./internal/app/laporan/... -run TestPemberitahuanSaatMelebihiBatas
go run ./cmd/s8 banding --modul S-2 --laporan 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `pyMaxRecords=500` pada 54 dari 56 laporan | `T-12` · `ADR-0011` |
| `OFFSET` nol kemunculan di export | `T-12` |
| 12 Report Definition hilang; `BrowseVPanel_HE_RD` 82 pemakaian | `docs/verifikasi-bukti-adr.md` §15 baris `S-2` · `R-16` |
| Menghapus batas adalah penambahan kemampuan | `ADR-0011` Negatif/utang teknis |
| Akses data medis dibatasi | `FR-R2` |

## Comments
