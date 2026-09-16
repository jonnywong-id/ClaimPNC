---
title: "TKT-B02-005 — Konversi ulang klaim yang idempoten"
labels: [modul::B-2, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B02-005 — Konversi ulang klaim yang idempoten

Status: needs-info
Kesiapan: **terhalang `ADR-0013` yang berstatus `Proposed`**
Modul: **B-2 Input Register** · Gelombang: 3 · Bergantung pada: TKT-B02-001, TKT-F2-005
Requirement: FR-B2    Keputusan: D-66    ADR: 0012, 0013    Risiko: —
Rule Pega yang digantikan: `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503` — **menghapus 12 tabel** milik satu klaim lalu menyisipkan ulang seluruh pohonnya
Peran penguji gerbang 2: **PncAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Mengulang konversi sebuah klaim **tidak menggandakan datanya**, tanpa bergantung pada penghapusan
fisik.

Nilai bisnisnya: pola lama sudah **rusak sebagian hari ini**. Pada dua tabel — `T_DLALIST` dan
`T_PLALIST` — baris `DELETE`-nya dikomentari (`:497`, `:498`) sementara `INSERT`-nya tetap aktif
(`:1296`, `:1325`). Artinya konversi ulang pada kedua tabel itu **sudah berpotensi menduplikasi
baris sekarang**, sebelum migrasi apa pun.

## Ruang lingkup

- Mekanisme idempotensi pengganti untuk pohon data satu klaim yang mencakup **12 tabel**.
- Penutupan cacat duplikasi `T_DLALIST` dan `T_PLALIST` yang sudah berjalan.
- Pencatatan jejak audit pada konversi ulang (`S-5`) — siapa mengulang, kapan, dan atas klaim mana.

## Non-goal

- **Tidak** memakai penghapusan fisik. `ADR-0012` menetapkan tidak ada `DELETE` pada data bernilai
  bisnis.
- **Tidak** memperbaiki data historis yang telanjur terduplikasi — itu pekerjaan DBA tersendiri.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Pola pengganti hapus-lalu-sisip-ulang** — *upsert* berbasis kunci alami, versioning dengan penanda baris aktif, atau melarang konversi ulang sama sekali (`ADR-0013`) | **Work Owner + Lead Engineer** | Ketiganya menghasilkan bentuk tabel dan AC yang berbeda. Menulis AC sekarang berarti menebak |
| **Seberapa sering konversi ulang benar-benar dijalankan di produksi, dan untuk apa?** | **Work Owner** | Menentukan apakah melarangnya (opsi paling sederhana) layak dipertimbangkan sama sekali |
| **Apakah ke-12 tabel punya kunci alami yang unik?** | **DBA + Lead Engineer** | Menentukan apakah *upsert* dapat dipakai seragam atau hanya sebagian |
| **Apakah riwayat hasil konversi sebelumnya perlu disimpan?** | **Work Owner + Compliance** | Bila ya, hanya versioning yang memenuhi |

## Acceptance criteria

> **Bentuk** AC yang akan diisi setelah `ADR-0013` diputuskan. Butir 1 dan 2 berlaku apa pun
> pilihannya.

- [ ] Menjalankan konversi **dua kali** atas klaim yang sama menghasilkan **jumlah baris yang sama**
      pada kesembilan belas tabel terdampak — diuji.
- [ ] `T_DLALIST` dan `T_PLALIST` **tidak menduplikasi** pada konversi kedua — inilah bukti cacat
      yang sudah berjalan tertutup.
- [ ] **Nol `DELETE` fisik** pada tabel bernilai bisnis — diuji pemindaian (`TKT-F2-005`).
- [ ] Konversi ulang tercatat di jejak audit dengan pelaku dan waktu.
- [ ] Gerbang 1: **jumlah baris tidak dibandingkan langsung** — perbandingan memakai **hasil kueri
      sesuai aturan bisnis**, karena soft delete mengubah isi tabel tanpa mengubah yang dilihat
      pengguna (`docs/Steering/14-TESTING-STRATEGY.md` §6.4).

## Dependency / Blocked by

`TKT-B02-001`, `TKT-F2-005`. **Terhalang `ADR-0013`.**

## Constraint keamanan, data, operasional

- Konversi menyentuh **pohon data satu klaim penuh** — ia harus berada di dalam satu transaksi
  (`TKT-F2-003`), atau kegagalan di tengah meninggalkan klaim setengah jadi.
- Data yang "dihapus" tetap ada dan tetap menempati kunci alaminya (`ADR-0012`) — itu yang membuat
  *upsert* menjadi kandidat, dan sekaligus yang membuatnya rumit.

## Migrasi skema / rollout / rollback

Bentuk perubahan skema bergantung pada pilihan `ADR-0013`: *upsert* menuntut constraint unik pada
kunci alami; versioning menuntut kolom penanda baris aktif pada 12 tabel.

**Rollback:** kembali ke hapus-lalu-sisip-ulang **tidak tersedia** — `ADR-0012` melarang
penghapusan fisik. Bila pola penggantinya bermasalah, yang dilakukan adalah **menghentikan
konversi ulang sementara**, bukan mengembalikan penghapusan.

## Rencana verifikasi

> **Rencana — belum dijalankan, dan belum dapat disusun lengkap.**

```bash
go test ./internal/app/konversi/... -run TestKonversiUlangIdempoten
go test ./internal/app/konversi/... -run TestDLAListTidakDuplikat
grep -rIn "DELETE FROM" internal/app/konversi/     # HARUS 0 baris
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Hapus 12 tabel lalu sisip ulang | `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503` |
| `DELETE` dikomentari, `INSERT` tetap aktif | `:497`, `:498` versus `:1296`, `:1325` |
| Soft delete menyeluruh | `D-66` · `ADR-0012` |
| Pola pengganti belum diputuskan | `ADR-0013` `Proposed` |
| Uji kesetaraan tidak membandingkan `COUNT(*)` | `docs/Steering/14-TESTING-STRATEGY.md` §6.4 |

## Comments
