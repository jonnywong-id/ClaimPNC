---
title: "TKT-S2-001 — Engine PDF, Excel, dan CSV"
labels: [modul::S-2, tipe::fondasi, status::needs-info, prioritas::tinggi, gelombang::6]
milestone: "Gelombang 6 — Laporan"
epic: "Migrasi Claim PNC"
---

# TKT-S2-001 — Engine PDF, Excel, dan CSV

Status: needs-info
Kesiapan: terhalang keputusan (kriteria kesamaan keluaran)
Modul: **S-2 Laporan & Export** · Gelombang: 6 · Bergantung pada: TKT-F1-005, TKT-U2-004
Requirement: FR-S2    Keputusan: D-11    ADR: 0011    Risiko: —
Rule Pega yang digantikan: engine reporting Pega · 10 template HTML · `Activity/PrintPDFAcceptanceNote-Act.xml` · 3 Correspondence
Peran penguji gerbang 2: **PNCReportClaimInternal** dan **PncManagerAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu cara membuat PDF, Excel, dan CSV di dalam aplikasi — dipakai 56 laporan, LOD, dan dokumen
PLA/DLA.

Nilai bisnisnya: engine reporting Pega lenyap bersama platformnya. Tanpa penggantinya, **tidak
satu pun laporan dapat diterbitkan** — dan laporan adalah hal yang dilihat manajemen setiap bulan.

## Ruang lingkup

- Pembuatan **PDF** dengan tata letak yang dapat didefinisikan ulang per dokumen.
- Pembuatan **Excel** dan **CSV**, keduanya **dialirkan (streaming)** agar memori tetap datar
  berapa pun jumlah barisnya.
- Pemformatan angka dan tanggal memakai pemformat terpusat `TKT-U2-004` — agar angka di laporan
  **sama dengan angka di layar**.
- Pool koneksi terpisah untuk laporan (`TKT-F2-001`).

## Non-goal

- **Tidak** membangun 56 laporan — itu `TKT-S2-002`.
- **Tidak** memakai tools BI eksternal maupun engine berbayar (`ADR-0011`).

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Keluaran PDF wajib identik visual dengan Pega, atau cukup identik isi?** | **Work Owner** (`ADR-0011`) | Menentukan kriteria kelulusan gerbang 1 untuk **seluruh** dokumen cetak — termasuk LOD dan PLA/DLA |
| **Berapa baris maksimum yang wajib dilayani satu export?** | **Work Owner** | Tanpa angka, `S-2` tidak dapat dinyatakan selesai secara terukur |
| **Export besar serentak, atau diantrekan lalu diberitahukan saat selesai?** | **Work Owner + Lead Engineer** | Menentukan apakah dibutuhkan mekanisme antrean dan pemberitahuan |

## Acceptance criteria

- [ ] PDF, Excel, dan CSV dapat dihasilkan dari satu sumber data yang sama — diuji ketiganya.
- [ ] Export **100.000 baris** berjalan dengan **memori datar** — diuji dengan pengukuran; memori
      tidak tumbuh sebanding jumlah baris.
- [ ] Angka dan tanggal pada laporan **sama persis** dengan yang ditampilkan di layar untuk data
      yang sama — diuji pada 10 nilai, termasuk uang berdesimal.
- [ ] Laporan berat **tidak menghabiskan koneksi transaksi** — diuji dengan menjalankan export
      besar sambil mengirim permintaan transaksi biasa.
- [ ] Kesamaan keluaran dengan Pega diuji sesuai kriteria yang ditetapkan Work Owner — pada 10
      dokumen contoh.
- [ ] Kegagalan pembuatan laporan **tidak menjatuhkan aplikasi** — diuji dengan data yang sengaja
      rusak.
- [ ] Gerbang 2: UAT **PNCReportClaimInternal**.

## Dependency / Blocked by

`TKT-F1-005` · `TKT-U2-004` · `TKT-F2-001`. **Terhalang tiga keputusan Work Owner.**

## Constraint keamanan, data, operasional

- Laporan memuat **data nasabah**; batas data cabang dan lini bisnis berlaku pada isinya
  (`TKT-F3-005`) — laporan **tidak boleh** menjadi jalan memintas pembatasan layar.
- Export besar memakan memori dan CPU yang sama dengan pelayanan transaksi (`ADR-0001`) — tanpa
  pembatasan, satu export dapat memperlambat seluruh pengguna.
- Data medis pada laporan tunduk `FR-R2`.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema.

**Rollback:** mengembalikan versi engine; laporan yang sudah dihasilkan tetap ada sebagai berkas.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/laporan/... -run TestPDFExcelCSV
go test ./internal/app/laporan/... -run TestMemoriDatarSaatExportBesar
go test ./internal/app/laporan/... -run TestPoolLaporanTerpisah
go run ./cmd/tools/banding-pdf keluaran/ baseline-pega/
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| PDF/Excel/CSV dibangun sendiri di Go | `D-11` · `ADR-0011` |
| 56 laporan, 78 activity | `docs/Steering/06-MODULE-BREAKDOWN.md` `S-2` |
| Export besar asinkron dan streaming | `docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md` butir 6 |
| Pool koneksi terpisah untuk laporan | idem butir 7 |
| Kriteria kesamaan keluaran belum ditetapkan | `ADR-0011` Pertanyaan terbuka |

## Comments
