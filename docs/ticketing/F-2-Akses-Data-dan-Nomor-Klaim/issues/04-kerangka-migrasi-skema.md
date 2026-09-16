---
title: "TKT-F2-004 — Kerangka migrasi skema backward-compatible dan rollback"
labels: [modul::F-2, tipe::fondasi, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F2-004 — Kerangka migrasi skema backward-compatible dan rollback

Status: needs-info
Kesiapan: **terhalang `R-08`** — DDL seluruh tabel tidak tersedia
Modul: F-2 · Gelombang: 1 · Bergantung pada: TKT-F2-001
Requirement: FR-F2    Keputusan: D-21, D-27, D-63    ADR: 0004    Risiko: R-08
Rule Pega yang digantikan: — skema dikelola Pega dan DBA; tidak ada rule aplikasi yang mengaturnya. Tabel yang paling terdampak: `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` (**dibaca 116 rule**), `PC_ASSIGN_WORKLIST` (18), `PC_ASSIGN_WORKBASKET` (6), `PR_OPERATORS` (6), `PC_LINK_ATTACHMENT` (3)
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Cara mengubah skema database yang **tidak pernah mematikan sistem yang sedang melayani
produksi**, beserta jalan mundurnya.

Nilai bisnisnya diikat dua hal sekaligus: `D-27` menuntut rolling deployment 24/7 sehingga versi
lama dan baru aplikasi berjalan bersamaan terhadap skema yang sama, dan `ADR-0004` menetapkan
database **dibagi dengan Pega**. Satu `ALTER` yang keliru pada tabel header klaim menghentikan
sistem yang dipakai **116 rule Pega**.

## Ruang lingkup

- Perkakas migrasi berbasis berkas SQL polos yang dapat di-review (`golang-migrate` atau setara).
- Penamaan `NNNN_deskripsi_singkat.up.sql` dan `.down.sql`; **setiap migrasi wajib punya `down`
  yang benar-benar berfungsi**.
- Migrasi dijalankan **terpisah dari start aplikasi**, sebagai langkah deployment tersendiri —
  agar dua instans tidak mencoba bermigrasi bersamaan.
- **Urutan empat rilis** untuk perubahan yang tidak kompatibel, ditulis sebagai prosedur yang
  diikuti, bukan saran.
- Templat **permintaan perubahan skema** sesuai `D-63`: permintaan tertulis → persetujuan Work
  Owner → pelaksanaan DBA → uji Pega dan Go bersamaan.

## Non-goal

- **Tidak** merancang skema tabel bisnis — itu milik modul bisnisnya.
- **Tidak** menjalankan migrasi apa pun di lingkungan mana pun.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **DDL seluruh tabel** — tipe kolom, panjang, index, constraint (`R-08`) | **DBA** | Tanpa DDL, `down.sql` tidak dapat ditulis benar: mengembalikan kolom ke tipe semula menuntut tahu tipe semula. Menebaknya berarti rollback yang gagal tepat saat dibutuhkan |
| **Statistik ukuran tabel** | **DBA** | Menentukan apakah sebuah `ALTER` dapat dijalankan online atau menuntut jendela pemeliharaan — dan itu mengubah prosedur, bukan hanya estimasi |

Bagian yang **tidak terhalang** dan boleh dikerjakan lebih dulu: pemilihan perkakas, penamaan,
pemisahan migrasi dari start aplikasi, dan templat permintaan `D-63`.

## Acceptance criteria

- [ ] Migrasi dijalankan lewat perintah **terpisah** dari menjalankan aplikasi — dibuktikan
      dengan menjalankan aplikasi tanpa hak DDL dan aplikasi **tetap** start.
- [ ] Dua proses migrasi yang dijalankan bersamaan **tidak saling merusak** — satu berhasil, satu
      menunggu atau gagal bersih. Diuji.
- [ ] Setiap migrasi contoh punya `down.sql` yang **mengembalikan skema ke keadaan semula** —
      diuji `up` lalu `down` lalu bandingkan definisi skema.
- [ ] Migrasi yang **menghapus kolom dalam satu langkah ditolak** oleh pemeriksaan — penghapusan
      wajib dua tahap.
- [ ] Templat permintaan perubahan skema memuat keempat langkah `D-63` dan **bagian rollback yang
      tidak boleh kosong**.
- [ ] Prosedur empat rilis terdokumentasi dengan contoh nyata satu penggantian nama kolom.

## Dependency / Blocked by

Bergantung pada `TKT-F2-001`. **Terhalang `R-08`** untuk bagian yang menyentuh tabel nyata.

## Constraint keamanan, data, operasional

- **Backward-compatible tanpa perkecualian** (`P-4`): tambah kolom boleh langsung asal *nullable*
  atau ber-*default*; hapus kolom dua tahap; ganti nama **tidak pernah langsung**; ubah tipe lewat
  kolom baru.
- **Akun aplikasi tidak boleh punya hak DDL** di produksi — migrasi dijalankan akun terpisah.
- Setiap perubahan skema **wajib diuji dengan menjalankan Pega dan Go bersamaan** terhadap skema
  hasil perubahan (`D-63`). Itu tidak dapat dilakukan DBA sendirian maupun tim pengembang
  sendirian.

## Migrasi skema / rollout / rollback

Tiket ini **adalah** mekanisme rollback-nya. Rollback tiket ini sendiri: menghapus perkakas
migrasi; tidak ada skema yang berubah karena belum ada migrasi yang dijalankan.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
migrate -path db/migrations -database "$DSN" up
migrate -path db/migrations -database "$DSN" down 1
# bandingkan definisi skema sebelum dan sesudah
./scripts/dump-schema.sh > /tmp/sesudah.sql && diff /tmp/sebelum.sql /tmp/sesudah.sql  # HARUS kosong
# uji dua migrasi bersamaan
( migrate ... up & migrate ... up & wait )   # tidak boleh merusak
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Tabel header klaim dibaca 116 rule Pega | `D-21` · `ADR-0004` |
| Rolling deployment 24/7 menuntut backward-compatible | `D-27` · `P-4` |
| Prosedur perubahan skema tiga pihak | `D-63` · `docs/Steering/09-DATABASE-STRATEGY.md` §9.1 |
| Urutan empat rilis | `docs/Steering/09-DATABASE-STRATEGY.md` §9 |
| DDL tidak tersedia | `R-08` |

## Comments
