---
title: "TKT-U5-002 — Unduhan hasil dan antrean export besar"
labels: [modul::U-5, tipe::migrasi, status::needs-info, prioritas::sedang, gelombang::6]
milestone: "Gelombang 6 — Laporan"
epic: "Migrasi Claim PNC"
---

# TKT-U5-002 — Unduhan hasil dan antrean export besar

Status: needs-info
Kesiapan: **terhalang keputusan cara menjalankan export besar**
Modul: **U-5 Layar Laporan** · Gelombang: 6 · Bergantung pada: TKT-U5-001, TKT-S2-001
Requirement: FR-U5    Keputusan: D-11    ADR: 0011    Risiko: —
Rule Pega yang digantikan: mekanisme unduhan laporan Pega · `Activity/PrintPDFAcceptanceNote-Act.xml`
Peran penguji gerbang 2: **PNCReportClaimInternal**, **PncManagerAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Pengguna mendapatkan berkas hasil laporan — PDF, Excel, atau CSV — **tanpa layarnya membeku** saat
laporan besar dijalankan.

Nilai bisnisnya: laporan bulanan adalah laporan terbesar dan dijalankan bersamaan oleh banyak
cabang di awal bulan. Bila tiap export menahan satu koneksi dan satu layar sampai selesai, saat
itulah sistem paling lambat justru ketika paling banyak dipakai.

## Ruang lingkup

- Pengunduhan berkas hasil dari engine `TKT-S2-001`.
- Penanganan export besar sesuai keputusan Work Owner — **serentak** atau **diantrekan lalu
  diberitahukan**.
- Bila diantrekan: daftar permintaan export beserta statusnya, dan tautan unduhan saat selesai.
- Masa berlaku tautan unduhan, sejalan dengan `TKT-S1-002`.

## Non-goal

- **Tidak** membangun engine pembuat berkas — itu `TKT-S2-001`.
- **Tidak** mengubah isi laporan.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Export besar dijalankan serentak, atau diantrekan lalu diberitahukan saat selesai?** | **Work Owner + Lead Engineer** (`ADR-0011`) | Ini menentukan **ada atau tidaknya seluruh separuh tiket ini**. Bila diantrekan, dibutuhkan daftar permintaan, status, dan pemberitahuan — bila tidak, semuanya gugur |
| **Berapa baris maksimum yang wajib dilayani satu export?** | **Work Owner** | Menentukan kapan layar harus menahan permintaan |
| **Berapa lama tautan unduhan berlaku?** | **Work Owner + Keamanan Informasi** | Berkas hasil memuat data nasabah; tautan yang berlaku selamanya memperpanjang paparan (`TKT-S1-002`) |

## Acceptance criteria

- [ ] Berkas hasil dapat diunduh dalam ketiga format — diuji.
- [ ] Export besar **tidak membekukan layar** — diuji dengan export 100.000 baris; pengguna tetap
      dapat berpindah halaman.
- [ ] Export besar **tidak menghabiskan koneksi transaksi** — diuji dengan menjalankan export
      besar sambil mengirim permintaan transaksi biasa (`TKT-F2-001`).
- [ ] Tautan unduhan **kedaluwarsa** setelah masa berlaku yang ditetapkan — diuji.
- [ ] Pengguna **tidak dapat mengunduh hasil export milik pengguna lain** — diuji dengan tautan
      milik orang lain: ditolak.
- [ ] Bila antrean dipilih: pengguna melihat **status permintaannya** dan diberi tahu saat selesai
      — diuji. *Kriteria ini gugur bila Work Owner memilih export serentak.*
- [ ] Gerbang 2: UAT **PNCReportClaimInternal** pada laporan bulanan yang sebenarnya.

## Dependency / Blocked by

`TKT-U5-001` · `TKT-S2-001` · `TKT-F2-001` · `TKT-S1-002`. **Terhalang keputusan Work Owner.**

## Constraint keamanan, data, operasional

- **Berkas hasil export memuat data nasabah dalam jumlah besar dalam satu berkas.** Ia lebih
  berisiko daripada layar, karena dapat dipindahkan keluar sistem.
- Tautan unduhan tanpa masa berlaku mengulangi cacat token dokumen (`TKT-S1-002`).
- Export bersamaan di awal bulan adalah **beban puncak yang dapat diperkirakan** — dan kapasitas
  dirancang tanpa data historis yang sahih, karena angka pemakaian lama selalu terpotong di 500.

## Migrasi skema / rollout / rollback

Menambah tabel permintaan export bila antrean dipilih. Tidak menyentuh skema klaim.

**Rollback:** pengguna kembali ke unduhan Pega. Berkas yang sudah diunduh **tetap ada di komputer
pengguna** — tidak dapat ditarik.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm test -- unduhan --run TestLayarTidakMembekuSaatExportBesar
go test ./internal/app/laporan/... -run TestPoolLaporanTerpisah
go test ./internal/adapter/http/... -run TestUnduhanMilikOrangLainDitolak
go test ./internal/app/laporan/... -run TestTautanUnduhanKedaluwarsa
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Export besar asinkron dan streaming | `docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md` butir 6 |
| Pool koneksi terpisah untuk laporan | idem butir 7 |
| Cara menjalankan export besar belum diputuskan | `ADR-0011` Pertanyaan terbuka |
| Token dokumen lama tanpa masa berlaku | `docs/verifikasi-bukti-adr.md` §14.10 |
| Angka pemakaian historis terpotong di 500 | `T-12` · `ADR-0011` |

## Comments
