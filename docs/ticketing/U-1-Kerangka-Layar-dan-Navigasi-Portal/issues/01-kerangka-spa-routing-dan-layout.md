---
title: "TKT-U1-001 — Kerangka SPA: routing, tata letak, dan state global"
labels: [modul::U-1, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::2]
milestone: "Gelombang 2 — Kerangka UI"
epic: "Migrasi Claim PNC"
---

# TKT-U1-001 — Kerangka SPA: routing, tata letak, dan state global

Status: ready-for-human
Kesiapan: siap
Modul: U-1 · Gelombang: 2 · Bergantung pada: TKT-F1-005
Requirement: FR-U1    Keputusan: D-13, D-23, D-09    ADR: 0002    Risiko: R-11
Rule Pega yang digantikan: portal dan kerangka navigasi Pega — `Navigation/pyCaseWorkerNavigation-Navigation.xml` (**51 item menu**) dan **74 harness**
Peran penguji gerbang 2: **PncAdmin** dan **PncPICTeknik** (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Rangka aplikasi web yang dapat dijalankan, dengan routing, tata letak, dan tempat state global —
siap diisi layar oleh `U-3`, `U-4`, `U-5`, dan `U-6`.

Nilai bisnisnya sama dengan `F-1` di sisi backend: ia menetapkan **pola yang akan diikuti ratusan
layar**. `D-09` menetapkan tim sedang belajar React; pola yang tidak ditetapkan di awal berubah
menjadi banyak gaya berbeda, dan itu biaya yang dibayar selama masa hidup aplikasi.

## Ruang lingkup

- Struktur folder frontend dan **aturan penamaan komponen**, ditulis preskriptif (`D-09`).
- Routing dengan rute bersarang: kerangka portal → daftar → detail.
- Tata letak baku: kepala, navigasi samping, area isi, dan jejak lokasi — **mengikuti susunan
  Pega** (`D-13`).
- **Navigasi yang dibangun dari izin peran**: menu yang tidak diizinkan **tidak dirender**.
- State global seperlunya: identitas pengguna, izin menu, dan notifikasi — bukan seluruh data
  aplikasi.
- Pola pengambilan data ke API dengan penanganan memuat dan galat yang seragam.

## Non-goal

- **Tidak** membangun layar bisnis apa pun.
- **Tidak** membangun komponen tabel dan form — itu `U-2`.
- **Tidak** menjadikan penyembunyian menu sebagai pengamanan. Penegakan sebenarnya ada di server
  (`TKT-F3-005`); di frontend ia **kenyamanan tampilan**.

## Acceptance criteria

- [ ] Aplikasi dapat dijalankan dan menampilkan kerangka portal dengan navigasi kosong yang
      dibangun dari daftar izin — diuji dengan dua daftar izin berbeda menghasilkan menu berbeda.
- [ ] Rute yang tidak diizinkan peran pengguna **tidak dapat dicapai** lewat URL langsung, dan
      menampilkan halaman "tidak berwenang" — **bukan** halaman kosong.
- [ ] Muat ulang halaman pada rute dalam (`/klaim/123/estimasi`) **tetap menampilkan halaman yang
      benar** — bergantung pada *fallback* `TKT-F1-005`.
- [ ] Struktur folder dan penamaan terdokumentasi cukup preskriptif untuk diikuti tanpa bertanya,
      dan **pemeriksaan otomatis** menolak berkas di luar pola.
- [ ] Pola pengambilan data menampilkan keadaan memuat, galat, dan kosong secara seragam — diuji
      pada satu rute contoh dengan ketiga keadaan.
- [ ] Bundel produksi ter-*build* dan **tersemat di binary Go** — dibuktikan dengan menjalankan
      binary dari direktori kosong (`TKT-F1-005`).
- [ ] Halaman dapat dipakai pada lebar 1366 px tanpa gulir horizontal pada tata letak utama.

## Dependency / Blocked by

Bergantung pada `TKT-F1-005` (penyajian SPA dan *fallback* rute).

**Yang bergantung padanya:** `TKT-U2-005`, `TKT-U2-001`…`004`, dan seluruh layar `U-3`…`U-6`.

## Constraint keamanan, data, operasional

- **Penyembunyian menu bukan pengamanan.** Setiap rute yang menampilkan data memanggil endpoint
  yang memeriksa kewenangan di server (`D-59`, `TKT-F3-005`).
- State global **tidak menyimpan data nasabah lebih lama dari yang dibutuhkan layar** — tidak ada
  cache persisten di peramban untuk data klaim.
- Token sesi tidak disimpan di tempat yang dapat dibaca skrip pihak ketiga.

## Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** menjalankan binary versi sebelumnya — SPA tersemat
di binary, sehingga keduanya mundur bersama.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm ci && npm run build && npm run test
npm run test -- Routing.izin
npm run test:e2e -- --grep "muat ulang rute dalam"
go build ./... && (mkdir -p /tmp/kosong && cd /tmp/kosong && /path/app)   # SPA tersaji
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 74 harness, 269 section; 51 item menu | `docs/Steering/06-MODULE-BREAKDOWN.md` §4 · `Navigation/pyCaseWorkerNavigation-Navigation.xml` |
| React + TypeScript + Vite, SPA murni tersemat di binary | `D-23` · `ADR-0002` |
| Tata letak mengikuti Pega agar tanpa pelatihan ulang | `D-13` |
| Struktur preskriptif karena tim sedang belajar | `D-09` |
| Otorisasi ditegakkan di server, bukan dengan menyembunyikan menu | `D-59` · `ADR-0023` |

## Comments
