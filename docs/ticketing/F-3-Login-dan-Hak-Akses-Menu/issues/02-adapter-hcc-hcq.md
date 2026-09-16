---
title: "TKT-F3-002 — Adapter autentikasi HCC/HCQ"
labels: [modul::F-3, tipe::integrasi, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F3-002 — Adapter autentikasi HCC/HCQ

Status: needs-info
Kesiapan: **terhalang artefak** — kontrak API HCC/HCQ tidak ada
Modul: F-3 · Gelombang: 1 · Bergantung pada: TKT-F3-001
Requirement: FR-F3    Keputusan: D-07, D-56    ADR: 0024, 0028    Risiko: R-14
Rule Pega yang digantikan: **tidak ada** — nol Connect REST ke HCC/HCQ di seluruh export
Peran penguji gerbang 2: **tidak berlaku** — `D-60`

## Hasil yang diharapkan (dan nilai bisnisnya)

Adapter yang memvalidasi kredensial pengguna ke sistem identitas internal HCC/HCQ.

Nilai bisnisnya: **tanpa ini tidak ada pengguna yang dapat masuk**. Ini penghalang paling awal di
seluruh rencana migrasi — bukan paling besar, tetapi paling awal.

## Ruang lingkup

- Adapter `Identity` yang memanggil API HCC/HCQ, memetakan responsnya ke profil pengguna.
- Penanganan kegagalan: timeout, kredensial salah, pengguna tidak aktif, dan **HCC/HCQ tidak dapat
  dihubungi**.
- **Pencocokan identitas HCC/HCQ dengan `OPERATOR_ID`** yang dipakai di seluruh data klaim.
- Uji fungsional terhadap kontrak — inilah pengganti gerbang 1 bagi bagian ini (`D-56`).

## Non-goal

- **Tidak** menyimpan kata sandi. Aplikasi ini tidak pernah menjadi pemilik kredensial (`D-07`).
- **Tidak** menerbitkan sesi — itu `TKT-F3-003`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Kontrak API HCC/HCQ** — endpoint, format permintaan dan respons, kode galat, batas percobaan, endpoint refresh/validasi | **Tim HCC/HCQ** | Tidak ada satu pun jejaknya di export. Menebak bentuknya berarti menulis adapter yang hampir pasti salah |
| **Apa yang terjadi bila HCC/HCQ tidak dapat dihubungi** — seluruh aplikasi tidak dapat diakses, atau ada jalur cadangan? | **Work Owner** | Menyentuh tuntutan 24/7 `D-27`. Bila tidak ada jalur cadangan, ketersediaan aplikasi ikut bergantung pada sistem lain |
| **Cara mencocokkan identitas HCC/HCQ dengan `OPERATOR_ID`** yang dipakai di seluruh data klaim | **Work Owner + Tim HCC/HCQ** | Tanpa pemetaan ini, pengguna yang berhasil login **tetap tidak dikenali oleh data klaimnya sendiri** — ini butir yang paling mudah terlewat |
| **Masa berlaku sesi dan cara memperbaruinya** | **Work Owner + Security** | Menentukan bentuk `TKT-F3-003` |

## Acceptance criteria

> **Belum dapat ditulis dengan angka.** Bentuk AC yang akan diisi setelah kontrak diterima:

- [ ] Login dengan kredensial benar mengembalikan profil lengkap berisi kelima field.
- [ ] Login dengan kredensial salah mengembalikan galat yang **tidak membedakan** apakah pengguna
      ada atau tidak — mencegah pencacahan pengguna.
- [ ] HCC/HCQ yang tidak merespons dalam batas waktu menghasilkan galat yang **dapat dibedakan**
      dari kredensial salah, dan perilaku aplikasi sesuai keputusan Work Owner.
- [ ] Setiap pemanggilan tercatat: tujuan, lama, hasil — **tanpa kredensial** (`TKT-F1-003`).
- [ ] Pemetaan identitas → `OPERATOR_ID` terbukti pada minimal 10 pengguna nyata di staging.
- [ ] Uji fungsional mencakup **seluruh** kode galat yang disebut kontrak — jumlahnya dilaporkan
      sebagai angka.

## Dependency / Blocked by

Bergantung pada `TKT-F3-001`. **Terhalang Tim HCC/HCQ.**

**Yang terhalang olehnya:** login seluruh aplikasi, karenanya rilis modul apa pun ke pengguna.

## Constraint keamanan, data, operasional

- Kredensial **tidak pernah** dicatat, tidak di log dan tidak di jejak audit.
- Kredensial aplikasi untuk memanggil HCC/HCQ (bila ada) tunduk `D-40` yang masih `OPEN`.
- Galat autentikasi **tidak boleh** membocorkan keberadaan akun.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** kembali ke adapter fake **tidak tersedia di produksi**
(`TKT-F3-001`) — bila adapter HCC/HCQ gagal di produksi, yang dilakukan adalah menunda rilis,
bukan mengaktifkan fake.

## Rencana verifikasi

> **Rencana — belum dijalankan, dan belum dapat disusun lengkap tanpa kontrak.**

```bash
go test ./internal/adapter/identitas/... -run TestHCCHCQ
go test ./internal/adapter/identitas/... -run TestSeluruhKodeGalatKontrak
go run ./cmd/tools/cek-pemetaan-operator --sample 10
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| HCC/HCQ nol jejak; hanya 2 teks pesan galat | `T-2` · `docs/verifikasi-bukti-adr.md` §1.1 |
| Autentikasi didelegasikan; aplikasi menerbitkan sesi sendiri | `D-07` |
| Gerbang 1 diganti uji fungsional terhadap kontrak | `D-56` · `ADR-0028` |
| Empat pertanyaan terbuka | `ADR-0024` Pertanyaan terbuka |
| `OPERATOR_ID` dipakai di seluruh data klaim | `docs/Steering/11-SECURITY.md` §2.1 |

## Comments
