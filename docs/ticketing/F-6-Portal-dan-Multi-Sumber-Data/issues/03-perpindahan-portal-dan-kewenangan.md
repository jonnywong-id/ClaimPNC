---
title: "TKT-F6-003 — Perpindahan portal dan penilaian ulang kewenangan"
labels: [modul::F-6, tipe::keamanan, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F6-003 — Perpindahan portal dan penilaian ulang kewenangan

Status: needs-info
Kesiapan: **terhalang keputusan identitas**
Modul: **F-6 Portal & Multi-Sumber Data** · Gelombang: 1 · Bergantung pada: TKT-F6-002, TKT-F3-005
Requirement: FR-F6, FR-F3    Keputusan: D-75, D-77, D-78, D-59    ADR: 0030, 0023    Risiko: **R-20**
Rule Pega yang digantikan: **tidak ada padanan** — sistem lama tidak punya perpindahan portal; entitas ditentukan server yang diakses
Peran penguji gerbang 2: **PncAdmin**, **PncManagerAdmin**, dan pemegang peran di **dua entitas berbeda**

## Hasil yang diharapkan (dan nilai bisnisnya)

Pengguna berpindah portal **tanpa login ulang** (`D-75`), dan setiap perpindahan **menilai ulang
kewenangannya** di portal tujuan.

Nilai bisnisnya dan bahayanya adalah hal yang sama. Tanpa login ulang, satu identitas menjangkau
**empat database milik empat badan hukum**. Bila kewenangan tidak dinilai ulang saat berpindah,
seorang pengguna dapat melihat data entitas yang bukan haknya — dan itu **kebocoran data antar badan
hukum**, bukan cacat tampilan. Inilah `R-20`.

## Ruang lingkup

- Perpindahan portal lewat **daftar pilihan (dropdown) di dalam aplikasi** — pengguna **tidak
  kembali ke halaman login** (`D-77`). Sesi tidak diputus.
- **Penilaian ulang kewenangan di server pada setiap perpindahan** (`ADR-0023`).
- Daftar portal yang ditampilkan **hanya yang menjadi hak pengguna**.
- Portal aktif melekat pada **permintaan**, bukan pada keadaan global yang dapat tertukar antar
  permintaan bersamaan.
- Pencatatan setiap perpindahan: siapa, dari portal apa ke apa, kapan.

## Non-goal

- **Tidak** menyatukan data antar portal.
- **Tidak** menyalin kewenangan antar portal — berhak di satu portal tidak berarti berhak di portal lain.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| ~~Satu penyedia identitas, atau satu per entitas?~~ — **TERTUTUP `D-78`**: **login sama untuk semua entitas** | — | Perpindahan tanpa login ulang **mungkin secara teknis**. Tetapi satu kredensial bocor kini menjangkau **empat badan hukum** — kewenangan portal menjadi **satu-satunya pembatas** |
| **Kewenangan portal disimpan di mana?** Karena login sama (`D-78`), data "berhak atas portal mana" bersifat **lintas portal secara alamiah** dan **tidak dapat** disimpan per portal — memeriksa hak atas portal B menuntut membaca database B sebelum orang itu terbukti berhak. Usulan: **terpusat bersama identitas**, sebagai pengecualian sadar terhadap `D-75` butir 4 | **Work Owner + Keamanan Informasi** | Ini **satu-satunya kendali** yang memisahkan empat badan hukum setelah `D-78` |
| **Bolehkah satu pengguna berhak di lebih dari satu portal?** | **Work Owner** | Bila tidak, seluruh tiket ini menyusut menjadi pemilihan portal saat login |
| **Kontrak API HCC/HCQ** masih nol jejak di export | **Tim HCC/HCQ** (`R-14`) | Penghalang yang sama dengan `TKT-F3-002` |

## Acceptance criteria

- [ ] Perpindahan dilakukan lewat **dropdown**, dan **tidak pernah** mengembalikan pengguna ke
      halaman login — diuji.
- [ ] Pengguna hanya melihat portal yang **menjadi haknya** — diuji dengan pengguna berhak di satu
      portal: portal lain tidak muncul.
- [ ] Portal yang bukan haknya **tidak dapat dipakai lewat pemanggilan langsung** — diuji dengan
      menyisipkan kode portal pada permintaan: **`403`**, bukan data.
- [ ] Kewenangan **dinilai ulang di server pada setiap perpindahan** — diuji dengan mencabut hak
      saat sesi berjalan: perpindahan berikutnya ditolak.
- [ ] Portal aktif melekat pada permintaan — diuji dengan **permintaan bersamaan dari dua portal
      berbeda oleh satu pengguna**: tidak ada data yang tertukar.
- [ ] Setiap perpindahan **tercatat** di jejak audit portal asal **dan** portal tujuan (`S-5`).
- [ ] **Perpindahan yang ditolak mempertahankan portal sebelumnya** dan menampilkan pesan — diuji:
      pengguna **tidak** dikeluarkan, dan **tidak** berakhir tanpa portal aktif (`D-77`).
- [ ] Sesi yang kedaluwarsa **tidak dapat berpindah portal** — diuji.
- [ ] Gerbang 2: UAT dijalankan pemegang peran di **dua entitas berbeda**, bukan satu orang dengan
      hak penuh — justru pemisahannya yang diuji.

## Dependency / Blocked by

`TKT-F6-002` · `TKT-F3-005` (middleware otorisasi) · `TKT-F3-002` (adapter HCC/HCQ) ·
`TKT-S5-002` (pencatatan). **Terhalang Work Owner, Keamanan Informasi, dan Tim HCC/HCQ.**

## Constraint keamanan, data, operasional

- **`D-78` memperberat `R-20`.** Satu kredensial bocor kini menjangkau **empat badan hukum**;
  bila tiap entitas punya login sendiri, kebocoran terbatas pada satu entitas. Batas itu hilang.
- **`R-20` adalah risiko paling serius yang dibawa `D-75`.** Kegagalannya tidak terlihat sebagai
  galat — ia terlihat sebagai data yang tampil normal, hanya milik entitas yang salah.
- Portal aktif **tidak boleh** disimpan sebagai keadaan global di server. Dua permintaan bersamaan
  dari pengguna yang sama akan saling menimpa, dan akibatnya adalah data entitas yang tertukar.
- `D-59` menetapkan kontrol berbasis **menu**, tanpa pemisahan tugas formal. Kendali portal **tidak
  boleh** ikut bersandar pada penyembunyian menu — `ADR-0023` menuntut pemeriksaan di setiap endpoint.
- **Kegagalan perpindahan wajib mengembalikan ke portal sebelumnya** (`D-77`). Dua perilaku lain
  berbahaya: berpindah meski pemeriksaan gagal adalah `R-20` itu sendiri; berakhir tanpa portal aktif
  membuat permintaan berikutnya berisiko jatuh ke koneksi default.
- Jejak audit **per portal** (`D-75`) berarti perpindahan tercatat di dua tempat. Penyelidikan
  lintas entitas menuntut empat kueri terpisah.

## Migrasi skema / rollout / rollback

Menambah tabel hak akses portal per pengguna. Backward-compatible (`P-4`).

**Rollback:** membatasi setiap pengguna ke satu portal. Itu **mengurangi kemampuan**, tetapi
**tidak melonggarkan kendali** — arah rollback yang benar bila `R-20` terbukti.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/http/... -run TestPortalBukanHakDitolak
go test ./internal/adapter/http/... -run TestKewenanganDinilaiUlangSaatPindah
go test ./internal/adapter/http/... -run TestPermintaanBersamaanDuaPortalTidakTertukar
go test ./internal/app/audit/... -run TestPerpindahanPortalTercatatDiKeduaPortal
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Berpindah portal tanpa login ulang | `D-75` butir 5 |
| Jejak audit per portal | `D-75` butir 6 |
| Otorisasi diperiksa di setiap endpoint | `D-59` · `ADR-0023` |
| Kontrak API HCC/HCQ nol jejak di export | `R-14` · `ADR-0024` |
| `R-20` kebocoran data antar entitas | `D-75` · `ADR-0030` |

## Comments
