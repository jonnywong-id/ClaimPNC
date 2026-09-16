---
title: "TKT-U5-001 — Layar pemilih laporan dan parameternya"
labels: [modul::U-5, tipe::migrasi, status::needs-info, prioritas::sedang, gelombang::6]
milestone: "Gelombang 6 — Laporan"
epic: "Migrasi Claim PNC"
---

# TKT-U5-001 — Layar pemilih laporan dan parameternya

Status: needs-info
Kesiapan: **terhalang penghalang `S-2`**
Modul: **U-5 Layar Laporan** · Gelombang: 6 · Bergantung pada: TKT-U2-002, TKT-S2-002
Requirement: FR-U5    Keputusan: D-11, D-59    ADR: 0011, 0023    Risiko: R-16
Rule Pega yang digantikan: harness `Har_LaporanHasilAI`, `ReportKPIHarness`, `PNCTATReport`, `OutstandingKlaimperCabang_Harness`, `MonitoringSLINKOJK`, `PNCStudyClaim`
Peran penguji gerbang 2: **PNCReportClaimInternal**, **PNCReportClaimEksternal**

## Hasil yang diharapkan (dan nilai bisnisnya)

Layar tempat pengguna menemukan laporan yang dibutuhkannya, mengisi parameternya, dan
menjalankannya.

Nilai bisnisnya: ada **56 laporan**. Bila daftarnya tidak tersusun dan parameternya tidak jelas,
pengguna akan menjalankan laporan yang salah atau menyerah dan meminta data lewat jalur lain.

## Ruang lingkup

- Daftar laporan **yang dapat diakses peran yang sedang masuk** — laporan yang tidak berwenang
  **tidak muncul dan tidak dapat dijalankan**.
- Form parameter per laporan, memakai komponen `TKT-U2-002` dan pemilih tanggal baku.
- Menjalankan laporan dan menampilkan hasilnya dengan tabel baku `TKT-U2-001`.
- **Pemberitahuan bila hasil melebihi batas** — bukan pemotongan diam-diam (`TKT-S2-002`).

## Non-goal

- **Tidak** membangun laporannya — itu `S-2`.
- **Tidak** membangun antrean unduhan — itu `TKT-U5-002`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **12 Report Definition hilang dari export** | **Tim Pega** (`R-16`) | Parameter laporan-laporan itu tidak diketahui, sehingga formnya tidak dapat dibuat |
| **Laporan mana untuk peran mana?** | **Work Owner** (`D-59`) | Kontrol sekarang berbasis menu; pemetaan laporan ke peran **belum tertulis** |
| **Berapa batas baru dan bagaimana pengguna diberi tahu?** | **Work Owner** | Layar inilah yang menyampaikannya; tanpa keputusan, ia tidak punya pesan untuk ditampilkan |

## Acceptance criteria

- [ ] Daftar laporan yang muncul **sesuai kewenangan peran** — diuji dengan dua peran berbeda.
- [ ] Laporan yang tidak berwenang **tidak dapat dijalankan lewat pemanggilan langsung** — diuji:
      `403` (`ADR-0023`).
- [ ] Parameter tiap laporan **sama dengan parameter di Pega** — diperiksa per laporan.
- [ ] Hasil yang melebihi batas **memunculkan pemberitahuan**, bukan tabel yang diam-diam terpotong
      — diuji dengan data melebihi batas.
- [ ] Form parameter memakai `TKT-U2-002`; tanggal memakai pemilih tanggal baku — diperiksa.
- [ ] Laporan berisi **data medis** hanya muncul bagi peran yang berwenang (`FR-R2`) — diuji.
- [ ] Gerbang 2: UAT **PNCReportClaimInternal** dan **PNCReportClaimEksternal** secara terpisah.

## Dependency / Blocked by

`TKT-U2-002` · `TKT-S2-002` · `TKT-F3-005`. **Terhalang Tim Pega (`R-16`) dan Work Owner.**

## Constraint keamanan, data, operasional

- **Daftar laporan yang disaring hanya di layar bukan kendali.** Endpoint penjalan laporan wajib
  memeriksa kewenangan sendiri.
- Laporan adalah jalan memintas batas data layar bila kueri di baliknya tidak ikut dibatasi
  (`TKT-S2-002`).

## Migrasi skema / rollout / rollback

Tidak menyentuh skema; menambah tabel pemetaan laporan ke peran.

**Rollback:** pengguna kembali ke layar laporan Pega (`D-05`).

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/http/... -run TestLaporanTanpaKewenanganDitolak
npm test -- laporan --run TestPemberitahuanSaatMelebihiBatas
npm test -- laporan --run TestFormParameterMemakaiKomponenBaku
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 56 laporan | `docs/Steering/06-MODULE-BREAKDOWN.md` `S-2` |
| 12 Report Definition hilang dari export | `docs/verifikasi-bukti-adr.md:2782` · `R-16` |
| `pyMaxRecords=500` pada 54 dari 56 laporan | `T-12` · `ADR-0011` |
| Kontrol berbasis menu | `D-59` |
| Otorisasi diperiksa di setiap endpoint | `ADR-0023` |

## Comments
