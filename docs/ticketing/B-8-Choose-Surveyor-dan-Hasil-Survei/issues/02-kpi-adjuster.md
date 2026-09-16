---
title: "TKT-B08-002 — Komponen dan skor KPI adjuster"
labels: [modul::B-8, tipe::aturan-bisnis, status::needs-info, prioritas::sedang, gelombang::4]
milestone: "Gelombang 4 — Persetujuan"
epic: "Migrasi Claim PNC"
---

# TKT-B08-002 — Komponen dan skor KPI adjuster

Status: needs-info
Kesiapan: **terhalang keputusan** — rumus skor tidak ada di mana pun
Modul: **B-8 Choose Surveyor** · Gelombang: 4 · Bergantung pada: TKT-B08-001
Requirement: FR-B8    Keputusan: D-15    ADR: 0025, 0026    Risiko: —
Rule Pega yang digantikan: `INSERT_KPIADJUSTER` — menyimpan **10 komponen**, **tanpa agregasi**
Peran penguji gerbang 2: **PncManagerAdmin** dan **PncPICTeknik**

## Hasil yang diharapkan (dan nilai bisnisnya)

Penilaian kinerja adjuster yang **dapat dipertanggungjawabkan** — komponennya tercatat, dan
skornya dihitung dengan rumus yang diketahui.

Nilai bisnisnya: KPI adjuster memengaruhi siapa yang dipilih untuk survei berikutnya, dan pada
adjuster eksternal ia menyentuh hubungan kerja. Skor yang tidak dapat dijelaskan asalnya sulit
dipakai sebagai dasar keputusan.

## Ruang lingkup

- Pencatatan **10 komponen KPI** yang sudah ada di sistem lama.
- **Perhitungan skor** dari komponen-komponen itu — setelah rumusnya ditetapkan.
- Bobot komponen sebagai **master data** (`ADR-0025`), bukan konstanta di kode.
- Riwayat skor per periode, sehingga perubahan kinerja terlihat.

## Non-goal

- **Tidak** mengarang rumus. Bila rumusnya memang tidak pernah ada, itu temuan yang dilaporkan —
  bukan celah yang ditambal dengan tebakan.
- **Tidak** membangun laporan KPI — itu `S-7`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Rumus skor KPI adjuster tidak ada di mana pun.** `INSERT_KPIADJUSTER` hanya **menyimpan 10 komponen**; tidak ada rule yang mengagregasinya menjadi skor | **Work Owner** | Tanpa rumus, yang dapat dibangun hanyalah pencatatan komponen — dan itu **bukan KPI**. Kemungkinannya: rumusnya ada di luar sistem (spreadsheet), atau memang tidak pernah ada |
| **Kunci upsert hanya `CASEID`** — sehingga **adjuster kedua pada klaim yang sama menimpa yang pertama**. Benar? | **Work Owner** | Bila salah, data KPI adjuster pertama **hilang tanpa jejak** setiap kali ada adjuster kedua |

## Acceptance criteria

> Butir 1 dan 2 berlaku apa pun jawabannya; sisanya menunggu rumus.

- [ ] Kesepuluh komponen KPI tercatat per adjuster per klaim — diuji.
- [ ] **Adjuster kedua pada klaim yang sama tidak menimpa yang pertama** — diuji dengan dua
      adjuster; keduanya tersimpan. Bila Work Owner memutuskan perilaku lama benar, AC ini dibalik
      dan alasannya dicatat.
- [ ] Skor dihitung dari komponen memakai bobot dari **master**, bukan konstanta — diuji dengan
      mengubah bobot dan memeriksa skor berubah.
- [ ] Riwayat skor per periode tersimpan; skor lama **tidak ditimpa** (`ADR-0012`).
- [ ] Perubahan bobot tercatat di jejak audit — ia mengubah penilaian orang.
- [ ] Gerbang 1: komponen yang tersimpan **sama dengan Pega**; **skor tidak dapat dibandingkan**
      karena sistem lama tidak menghitungnya — itu dinyatakan terbuka, bukan diklaim setara.

## Dependency / Blocked by

`TKT-B08-001` · `TKT-F4-001` (master bobot). **Terhalang dua keputusan Work Owner.**

## Constraint keamanan, data, operasional

- KPI menyentuh **penilaian orang**, termasuk mitra eksternal. Perubahan bobot dan skor wajib
  tercatat.
- Menimpa data adjuster pertama (perilaku sistem lama) berarti **kehilangan data tanpa jejak** —
  itu bertentangan dengan `ADR-0012`, dan karena itu perlu keputusan sadar, bukan diwarisi.

## Migrasi skema / rollout / rollback

Menambah tabel komponen dan skor KPI. Backward-compatible.

**Rollback:** komponen yang tersimpan tetap ada; perhitungan skor dinonaktifkan.

## Rencana verifikasi

> **Rencana — belum dijalankan, dan skornya belum dapat diuji.**

```bash
go test ./internal/app/kpi/... -run TestSepuluhKomponenTersimpan
go test ./internal/app/kpi/... -run TestAdjusterKeduaTidakMenimpa
go run ./cmd/tools/cek-kpi-tanpa-rumus    # laporkan komponen tanpa agregasi
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `INSERT_KPIADJUSTER` menyimpan 10 komponen tanpa agregasi | `docs/verifikasi-bukti-adr.md` §15 baris `B-8` |
| Kunci upsert hanya `CASEID` | idem |
| Nilai bisnis tidak boleh di-hardcode | `D-15` · `ADR-0025` |
| Tidak ada penghapusan fisik data bernilai bisnis | `D-66` · `ADR-0012` |

## Comments
