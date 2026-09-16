---
title: "TKT-S7-001 — Basis perhitungan TAT"
labels: [modul::S-7, tipe::aturan-bisnis, status::needs-info, prioritas::tinggi, gelombang::6]
milestone: "Gelombang 6 — Laporan"
epic: "Migrasi Claim PNC"
---

# TKT-S7-001 — Basis perhitungan TAT

Status: needs-info
Kesiapan: **terhalang keputusan aturan bisnis**
Modul: **S-7 Dashboard, TAT & KPI** · Gelombang: 6 · Bergantung pada: TKT-F5-001, TKT-S4-003
Requirement: FR-S7    Keputusan: D-25, D-49    ADR: 0017    Risiko: R-01, R-03, R-19
Rule Pega yang digantikan: `Database/GETSELISIHJAM.fnc:9-18` (algoritma) dan `:19-22` (`EXCEPTION WHEN OTHERS THEN RETURN 0`) · `DATAMINING.GET_WORKING_HOURS@ASMD` (**17 pemakaian**) · `GENERAL.HRD_LBR@ASMD`
Peran penguji gerbang 2: **PncManagerAdmin** dan **PNCReportClaimInternal**

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu rumus TAT yang disepakati, dipakai seluruh laporan dan dashboard.

Nilai bisnisnya: sekarang ada **dua** rumus. `GETSELISIHJAM` mengurangi akhir pekan saja;
`GET_WORKING_HOURS@ASMD` memakai jam kerja dan kalender libur. Keduanya dipakai di jalur berbeda,
dan **angkanya tidak akan pernah sama**. Selama itu berlangsung, "TAT klaim" bukan satu angka —
ia dua angka yang keduanya disebut TAT.

Sebagai ukuran selisihnya: SLA yang dimulai **Jumat 16:00** dan berakhir **Senin 09:00** dihitung
**17 jam** oleh `GETSELISIHJAM`, sementara menurut jam kerja ia sekitar **2 jam**.

## Ruang lingkup

- **Satu** fungsi perhitungan TAT di Go, dipakai seluruh laporan, dashboard, dan KPI.
- Perhitungan memakai kalender hari libur dan jam kerja dari `TKT-F5-001` — bukan panggilan remote
  (`R-19`: aturan bisnis ditulis ulang, tidak dipanggil).
- **Kegagalan perhitungan menghasilkan kegagalan yang terlihat**, bukan nol.
- Pemetaan angka lama ke angka baru, agar selisihnya dapat dijelaskan.

## Non-goal

- **Tidak** menggarap tampilan dashboard — itu `TKT-S7-002`.
- **Tidak** memperbaiki data historis. Memperbaiki rumus **tidak memperbaiki baris yang telanjur
  salah** (`R-19`).

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Basis TAT yang benar — akhir pekan saja, atau jam kerja + hari libur?** | **Work Owner** | Ini **aturan bisnis**, bukan pilihan teknis. Memilihnya sendiri berarti menentukan apakah cabang dinilai lulus SLA atau tidak |
| **`RETURN 0` saat error pernah menyamarkan kegagalan KPI** — angka historis perlu diperiksa ulang? | **Work Owner** | Bila pernah, sebagian angka KPI yang sudah dilaporkan **adalah nol palsu**, dan keputusan yang diambil di atasnya perlu ditinjau |
| **`weekends2` dan `POOLDATA.datediff` tidak ada di export** | **DBA** (`R-01`) | Definisi "akhir pekan" **tidak dapat diverifikasi**. Bila opsi `GETSELISIHJAM` yang dipilih, ia tidak dapat disalin tanpa source-nya |
| **Angka TAT baru akan berbeda dari yang lama — diumumkan bagaimana?** | **Work Owner** | Perubahan ini terlihat langsung oleh manajemen cabang |

## Acceptance criteria

- [ ] **Hanya ada satu** fungsi TAT di seluruh kode — dicari otomatis; tidak ada perhitungan
      tandingan di laporan mana pun.
- [ ] Kegagalan perhitungan **menghasilkan kegagalan**, bukan `0` — diuji dengan masukan rusak.
      Ini **perbaikan perilaku yang disengaja** dan masuk daftar `P-5`.
- [ ] Hasil perhitungan **sama dengan basis yang dipilih Work Owner** pada 20 kasus, termasuk kasus
      lintas akhir pekan dan lintas hari libur nasional.
- [ ] Kasus **Jumat 16:00 → Senin 09:00** menghasilkan angka yang **sesuai keputusan**, dan angka
      itu **dicatat di tiket ini** sebagai acuan.
- [ ] Selisih terhadap angka Pega **dilaporkan per laporan**, bukan disamakan diam-diam — lewat
      `S-8` dan diklasifikasikan terhadap 13 butir `P-5`.
- [ ] Gerbang 2: UAT **PncManagerAdmin**.

## Dependency / Blocked by

`TKT-F5-001` (jam kerja & hari libur) · `TKT-S4-003` (objek remote). **Terhalang Work Owner dan DBA.**

## Constraint keamanan, data, operasional

- **Angka TAT menilai kinerja cabang.** Mengubah rumusnya mengubah penilaian itu — sebagian cabang
  akan terlihat lebih baik, sebagian lebih buruk, tanpa ada yang berubah cara kerjanya.
- `P-5` menuntut kesetaraan perilaku lebih dulu, tetapi `R-19` memperingatkan bahwa **mematuhinya
  buta akan menyalin cacat hitungan uang dan waktu**. Tiket ini termasuk yang dimaksud.
- `GET_WORKING_HOURS@ASMD` adalah **objek remote** — tidak tersedia setelah pindah ke PostgreSQL.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema.

**Rollback:** mengembalikan rumus lama **mengembalikan angka lama** — termasuk `RETURN 0` yang
menyamarkan kegagalan. Ini bukan rollback yang netral.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/tat/... -run TestSatuRumusTAT
go test ./internal/app/tat/... -run TestKegagalanBukanNol
go test ./internal/app/tat/... -run TestJumat1600SeninPagi
go run ./cmd/s8 banding --modul S-7 --kasus 20 --laporkan-selisih
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Algoritma `GETSELISIHJAM`: akhir pekan saja | `Database/GETSELISIHJAM.fnc:9-18` · `docs/verifikasi-bukti-adr.md` §14.9 |
| `EXCEPTION WHEN OTHERS THEN RETURN 0` | `Database/GETSELISIHJAM.fnc:19-22` |
| `GET_WORKING_HOURS@ASMD` dipakai 17× | `D-25` tabel inventaris DB Link |
| Jumat 16:00 → Senin 09:00 dihitung 17 jam | `docs/verifikasi-bukti-adr.md` §14.9 |
| `weekends2` dan `POOLDATA.datediff` tidak ada di export | idem · `R-01` |
| Aturan bisnis remote ditulis ulang di Go | `docs/Steering/00-DECISION-LOG.md:1326` · `R-19` |

## Comments
