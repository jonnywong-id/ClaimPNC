---
title: "TKT-S5-004 — Retensi dan arsip sebagai parameter konfigurasi"
labels: [modul::S-5, tipe::fondasi, status::ready-for-human, prioritas::sedang, gelombang::2]
milestone: "Gelombang 2 — Kerangka UI"
epic: "Migrasi Claim PNC"
---

# TKT-S5-004 — Retensi dan arsip sebagai parameter konfigurasi

Status: ready-for-human
Kesiapan: siap
Modul: S-5 · Gelombang: 2 · Bergantung pada: TKT-S5-001
Requirement: FR-S5    Keputusan: D-62, D-15    ADR: 0026, 0012    Risiko: —
Rule Pega yang digantikan: **tidak ada** — sistem lama tidak punya kebijakan retensi yang tercatat di rule mana pun
Peran penguji gerbang 2: **tidak berlaku** — `D-60`

## Hasil yang diharapkan (dan nilai bisnisnya)

Retensi data audit yang **dapat diisi angkanya kelak tanpa mengubah kode**, sehingga `S-5` dapat
dibangun sekarang meski angkanya belum ada.

Nilai bisnisnya adalah menghilangkan penghalang semu. `D-62` sudah menyelesaikan pertanyaan
kebijakannya — retensi audit **mengikuti retensi data klaim yang berlaku sekarang**, satu
kebijakan untuk keduanya. Yang belum ada hanyalah **angkanya**, dan angka tidak boleh menahan
pembangunan modul.

## Ruang lingkup

- Parameter retensi sebagai konfigurasi (`TKT-F1-002`), **bukan konstanta di kode** — konsisten
  `D-15`.
- Perhitungan "baris audit ini sudah melewati masa retensi" sebagai fungsi murni yang dapat diuji.
- **Mekanisme arsip yang belum dijalankan**: kode ada, terjadwal tidak aktif, dan aktivasinya
  menuntut angka retensi serta persetujuan tertulis.
- Laporan ukuran: berapa baris dan berapa besar tabel audit per periode — supaya keputusan arsip
  kelak diambil dengan angka.

## Non-goal

- **Tidak** menghapus data apa pun. Arsip tidak dijalankan sampai angkanya ada dan disetujui.
- **Tidak** menetapkan angka retensi — itu Work Owner dan Compliance.

## Acceptance criteria

- [ ] Parameter retensi dibaca dari konfigurasi; **nilai kosong berarti tidak ada yang
      diarsipkan** — diuji, dan itulah nilai bawaannya.
- [ ] Fungsi "sudah lewat retensi" diuji pada batas: tepat di hari terakhir, satu hari sesudah,
      dan satu hari sebelum.
- [ ] Proses arsip **tidak berjalan** selama parameter kosong — diuji dengan menjalankan
      penjadwal dan memeriksa nol baris tersentuh.
- [ ] Menjalankan arsip menuntut **penanda persetujuan eksplisit** di konfigurasi; tanpa itu ia
      menolak berjalan walau parameter retensi terisi — diuji.
- [ ] Laporan ukuran tabel audit dapat dihasilkan dan memuat jumlah baris per bulan.
- [ ] Arsip **memindahkan**, tidak menghapus — baris yang diarsipkan tetap dapat ditemukan.

## Dependency / Blocked by

Bergantung pada `TKT-S5-001` dan `TKT-F1-002`.

**Tidak terhalang** meski angka retensi belum ada — itulah inti tiket ini.

## Constraint keamanan, data, operasional

- **Arsip bukan penghapusan.** `ADR-0012` menetapkan tidak ada penghapusan fisik data bernilai
  bisnis, dan jejak audit adalah kelas data yang paling tidak boleh hilang.
- Menjalankan arsip pada data audit menyentuh **bukti**. Karena itu ia menuntut penanda persetujuan
  terpisah dari sekadar parameter terisi.
- Tabel audit **tumbuh paling cepat** (`D-10`); laporan ukuran adalah cara mengetahui kapan
  keputusan arsip benar-benar mendesak.

## Migrasi skema / rollout / rollback

Menambah tabel arsip (bila arsip memindahkan baris) — tabel baru, tidak menyentuh tabel Pega.

**Rollback:** mengosongkan parameter retensi menghentikan arsip seketika. Baris yang telanjur
diarsipkan **tetap ada** dan dapat dikembalikan, karena arsip memindahkan, bukan menghapus.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/audit/... -run TestRetensiBatas
go test ./internal/app/audit/... -run TestArsipTidakBerjalanTanpaParameter
go test ./internal/app/audit/... -run TestArsipMenolakTanpaPersetujuan
go run ./cmd/tools/laporan-ukuran-audit
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Retensi audit mengikuti retensi data klaim; angkanya belum ada | `D-62` |
| Retensi dibangun sebagai parameter agar `S-5` tidak terhalang | `D-62` |
| Tidak ada penghapusan fisik data bernilai bisnis | `D-66` · `ADR-0012` |
| Nilai bisnis tidak boleh di-hardcode | `D-15` · `ADR-0025` |
| Tabel audit tumbuh paling cepat | `docs/Steering/09-DATABASE-STRATEGY.md` §8 |

## Comments
