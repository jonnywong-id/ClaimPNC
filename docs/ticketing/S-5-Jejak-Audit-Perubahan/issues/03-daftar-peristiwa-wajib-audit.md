---
title: "TKT-S5-003 — Daftar peristiwa wajib audit"
labels: [modul::S-5, tipe::kepatuhan, status::needs-info, prioritas::tinggi, gelombang::2]
milestone: "Gelombang 2 — Kerangka UI"
epic: "Migrasi Claim PNC"
---

# TKT-S5-003 — Daftar peristiwa wajib audit

Status: needs-info
Kesiapan: **terhalang artefak** — kontrak dari Compliance belum ada
Modul: S-5 · Gelombang: 2 · Bergantung pada: TKT-S5-002
Requirement: FR-S5    Keputusan: D-28, D-56, D-62    ADR: 0026, 0028    Risiko: R-14
Rule Pega yang digantikan: **tidak ada** — sistem lama tidak punya daftar seperti ini
Peran penguji gerbang 2: **tidak berlaku** — `D-60`

## Hasil yang diharapkan (dan nilai bisnisnya)

Daftar resmi peristiwa yang **wajib** masuk jejak audit beserta field yang harus tercatat pada
masing-masing — disepakati Compliance, bukan disusun tim teknis.

Nilai bisnisnya melampaui kelengkapan. `D-56` menetapkan daftar ini **menggantikan gerbang 1**
bagi `S-5`, karena tidak ada baseline Pega untuk dibandingkan. Artinya: **tanpa daftar ini, `S-5`
tidak dapat dinyatakan lulus gerbang apa pun**, sekalipun kodenya selesai dan bekerja.

## Ruang lingkup

- Daftar peristiwa wajib audit, per entitas, beserta field yang harus tercatat.
- Pemetaan setiap peristiwa ke **titik kode** tempat ia terjadi.
- Uji fungsional yang memeriksa daftar itu terpenuhi seluruhnya — inilah pengganti gerbang 1.

## Non-goal

- **Tidak** menyusun daftarnya sendiri. Tim teknis dapat **mengusulkan**, tetapi yang mengikat
  adalah kesepakatan Compliance.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Daftar peristiwa wajib audit beserta field yang harus tercatat** | **Compliance** | Ia adalah **kontrak pengganti gerbang 1** (`D-56`). Tanpanya, tidak ada ukuran kelulusan |
| **Angka retensi data klaim yang berlaku sekarang** | **Work Owner + Compliance** | `D-62` menetapkan retensi audit mengikuti retensi data klaim, dan kebijakannya sudah ada — yang belum ada **angkanya**. Ini tidak menahan pembangunan (`TKT-S5-004` membuatnya parameter), tetapi menahan **go-live** |
| **Data historis tidak punya kolom nilai sebelum dan sesudah** — apakah diterima apa adanya? | **Compliance** | Menentukan apakah jejak audit dimulai dari nol pada tanggal cutover, atau menuntut rekonstruksi retrospektif yang **tidak mungkin dilakukan** dari data yang ada |

**Usulan awal yang dapat dipakai sebagai bahan pembicaraan** — bukan keputusan: daftar minimum
`ADR-0026` (nilai estimasi klaim, nilai settlement, akseptasi, keputusan komite, penolakan,
proses ulang, perubahan status klaim, pembayaran), ditambah tiga kandidat yang muncul dari analisis
— perubahan master ambang komite, perubahan master penerima notifikasi, dan perubahan peran
pengguna. Ketiganya diusulkan karena `D-59` menghapus pemisahan tugas, sehingga **perubahan pada
master yang menentukan kewenangan** menjadi tindakan bernilai tinggi.

## Acceptance criteria

> Tidak dapat ditulis dengan angka sampai daftarnya diterima. Butir di bawah adalah **bentuk** AC
> yang akan diisi, bukan AC final.

- [ ] Setiap peristiwa dalam daftar Compliance punya **uji fungsional** yang membuktikan ia
      tercatat beserta seluruh field yang diminta.
- [ ] Jumlah peristiwa yang diuji **sama dengan** jumlah peristiwa dalam daftar — dilaporkan
      sebagai angka, bukan pernyataan.
- [ ] Peristiwa yang ada di kode tetapi **tidak** ada di daftar dilaporkan sebagai selisih, dan
      diputuskan satu per satu: ditambahkan ke daftar, atau dihentikan pencatatannya.

## Dependency / Blocked by

Bergantung pada `TKT-S5-002`. **Terhalang Compliance.**

**Yang terhalang olehnya:** kelulusan seluruh modul `S-5`.

## Constraint keamanan, data, operasional

- Daftar ini **mengikat**; menambah atau mengurangi isinya kelak adalah perubahan yang menuntut
  persetujuan Compliance, bukan keputusan teknis.
- `BRD §21.2` kriteria #9 berlaku **tanpa pengecualian** pada seluruh modul bisnis karena `D-59` —
  daftar ini yang menjabarkannya.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema. Bila daftar menuntut field yang belum ada di tabel `TKT-S5-001`,
penambahannya menempuh `TKT-F2-004` (backward-compatible, dua tahap).

## Rencana verifikasi

> **Rencana — belum dijalankan, dan belum dapat disusun lengkap.**

```bash
go test ./internal/app/... -run TestPeristiwaWajibAudit
go run ./cmd/tools/banding-daftar-audit daftar-compliance.yaml   # selisih kode vs daftar
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Daftar peristiwa menggantikan gerbang 1 bagi `S-5` | `D-56` · `ADR-0028` |
| Daftar minimum yang wajib diaudit | `D-28` · `ADR-0026` |
| Retensi mengikuti retensi data klaim; angkanya belum ada | `D-62` |
| Sistem lama tidak punya jejak audit atas nilai | `T-14` |
| Audit satu-satunya kontrol pengimbang | `D-59` · `ADR-0023` |

## Comments
