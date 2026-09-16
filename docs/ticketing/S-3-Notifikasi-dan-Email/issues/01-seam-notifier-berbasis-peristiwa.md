---
title: "TKT-S3-001 — Seam Notifier berbasis peristiwa domain"
labels: [modul::S-3, tipe::fondasi, status::needs-info, prioritas::sedang, gelombang::5]
milestone: "Gelombang 5 — Nilai dan pihak luar"
epic: "Migrasi Claim PNC"
---

# TKT-S3-001 — Seam Notifier berbasis peristiwa domain

Status: needs-info
Kesiapan: **terhalang artefak**
Modul: **S-3 Notifikasi & Email** · Gelombang: 5 · Bergantung pada: TKT-F4-001
Requirement: FR-S3    Keputusan: D-67    ADR: 0016    Risiko: R-07
Rule Pega yang digantikan: `SendEmailNotification` — **15 pemanggil, tidak ada di export** (`docs/Steering/19-GAP-EXPORT-DETAIL.md:196`) · `Data Transform/SetDataEmail-DT.xml`
Peran penguji gerbang 2: **PncAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu titik di dalam aplikasi tempat pemberitahuan dilepaskan, dipanggil dengan **peristiwa yang
terjadi** — bukan dengan alamat email.

Nilai bisnisnya ada pada bentuk pemanggilannya. Bila pemanggil menyebut alamat, maka alamat itu
tersebar di 15 tempat dan setiap perubahan pejabat menuntut perubahan kode. Bila pemanggil menyebut
**peristiwa**, penerimanya ditentukan satu kali dari master data — dan berpindah pejabat menjadi
perubahan data, bukan rilis.

## Ruang lingkup

- Antarmuka `Notifier` dengan satu operasi: **melepaskan peristiwa domain** beserta konteks klaimnya.
- Penentuan penerima **dari master data** (`TKT-F4-001`), bukan dari konstanta di kode.
- Pencatatan setiap pelepasan peristiwa: peristiwa apa, klaim mana, kapan, berhasil atau gagal.
- Perilaku saat pengiriman gagal: **proses bisnis tetap berjalan**, kegagalan tercatat dan terlihat.

## Non-goal

- **Tidak** membuat template maupun menyambung ke SMTP — itu `TKT-S3-002`.
- **Tidak** membuat notifikasi dalam-aplikasi (lonceng, inbox pesan); belum ada permintaannya.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **`SendEmailNotification` tidak ada di export**, padahal dipanggil 15 activity | **Tim Pega** (`R-07`) | Daftar peristiwa yang sebenarnya memicu email **hanya dapat dibaca dari isi rule itu**. Tanpa itu, daftar peristiwa di tiket ini adalah dugaan — dan dugaan tidak boleh menjadi acceptance criteria |
| **Satu peristiwa mengirim ke berapa penerima, dan apakah ada penerima tembusan?** | **Work Owner** | Menentukan bentuk data penerima di master |
| **Kegagalan kirim perlu dicoba ulang otomatis, atau cukup dicatat?** | **Work Owner** | Menentukan apakah dibutuhkan antrean dan berapa kali percobaan |

> Daftar peristiwa **sengaja tidak dicantumkan di sini.** Menuliskannya dari dugaan akan membuat
> tiket ini tampak siap padahal isinya belum diketahui.

## Acceptance criteria

- [ ] Pemanggil melepaskan **peristiwa**, bukan alamat email — diuji: tidak ada satu pun alamat
      email di kode modul bisnis mana pun (dicari secara otomatis).
- [ ] Penerima diambil **dari master data**; mengubah penerima **tidak menuntut rilis** — diuji
      dengan mengubah data lalu memanggil ulang.
- [ ] **Kegagalan pengiriman tidak menggagalkan proses bisnis** — diuji dengan pengirim yang
      sengaja dibuat gagal; klaim tetap tersimpan dan berpindah status.
- [ ] Setiap pelepasan peristiwa **tercatat** dengan hasilnya — diuji.
- [ ] Daftar peristiwa yang terpasang **cocok dengan isi `SendEmailNotification`** setelah rule itu
      diterima — diperiksa satu per satu. *Kriteria ini belum dapat dijalankan sampai artefaknya ada.*
- [ ] Gerbang 2: UAT **PncAdmin**.

## Dependency / Blocked by

`TKT-F4-001` (master data penerima). **Terhalang Tim Pega (`R-07`).**

## Constraint keamanan, data, operasional

- Isi email memuat **data nasabah**; ia keluar dari batas sistem, sehingga apa yang boleh
  dicantumkan di dalamnya adalah keputusan bisnis, bukan teknis.
- **Tidak ada akun pribadi** sebagai penerima (`D-67`) — seluruhnya mailbox fungsional.
- Email yang memuat **data medis** tunduk `FR-R2`.

## Migrasi skema / rollout / rollback

Menambah tabel penerima per peristiwa di master data. Backward-compatible (`P-4`).

**Rollback:** mematikan pelepasan peristiwa **menghentikan pemberitahuan tanpa terlihat** — bila
dilakukan, harus disertai pemberitahuan ke pengguna yang bergantung padanya.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/notifikasi/... -run TestPeristiwaBukanAlamat
go test ./internal/app/notifikasi/... -run TestPenerimaDariMaster
go test ./internal/app/notifikasi/... -run TestGagalKirimTidakMenggagalkanProses
go run ./cmd/tools/cari-alamat-email ./internal/app/...   # HARUS nol temuan
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `SendEmailNotification` dipanggil 15× dan tidak ada di export | `docs/Steering/19-GAP-EXPORT-DETAIL.md:196` · `R-07` |
| Seam Notifier, penerima dari master data | `docs/Steering/06-MODULE-BREAKDOWN.md:62` |
| Tidak ada penerima akun pribadi | `D-67` |
| Modul bergantung pada `F-4` | `docs/Steering/06-MODULE-BREAKDOWN.md:62` |

## Comments
