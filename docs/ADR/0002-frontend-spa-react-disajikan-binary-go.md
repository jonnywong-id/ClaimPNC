# 0002 — Bangun antarmuka sebagai SPA React yang disajikan oleh binary Go

Status: Accepted
Tanggal keputusan: 2026-09-07    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-23`, `D-13`, `D-09`, `D-12` | `docs/verifikasi-bukti-adr.md` §10.7 (T-11, T-12)
Terkait: ADR-0001, ADR-0011, modul `U-1`…`U-6`

## Konteks

Antarmuka Pega adalah antarmuka **padat grid**: dari 269 section pada export, **268 memuat
grid**. `D-13` menetapkan tata letak, urutan langkah, dan penempatan field **mengikuti Pega yang
ada**, supaya pengguna tidak perlu dilatih ulang — sehingga kepadatan itu ikut terwarisi.

Dua temuan Fase 1 mengoreksi gambaran awal tentang beratnya grid:

- **T-11** — spesifikasi "tabel baku 18–27 kolom" salah sasaran. **Median kolom sebenarnya 6**,
  dan tiga fitur grid yang biasanya mahal — tambah baris inline, hapus baris inline, dan resize
  kolom — **tidak dipakai sama sekali**. `U-2` lebih ringan daripada yang diperkirakan.
- **T-12** — masalah paginasi bukan `OFFSET` besar (`OFFSET` **nol kemunculan** di export),
  melainkan **3.189 grid yang terikat page list klipboard** dan `pyMaxRecords=500` pada 54 dari
  56 laporan.

`D-09` membatasi pilihan: tim adalah developer Pega yang dilatih ulang, sehingga jumlah konsep
baru harus sedikit. `D-08` melarang mengasumsikan infrastruktur tambahan.

## Opsi yang dipertimbangkan

1. **SPA React + TypeScript + Vite**, disajikan sebagai berkas statis oleh binary Go.
2. **Server-side rendering** (Next.js atau sejenis) — menuntut runtime Node.js di produksi.
3. **Template HTML dari Go** (`html/template`) + JavaScript seperlunya.

## Keputusan

Antarmuka dibangun sebagai **SPA React + TypeScript + Vite**, dikompilasi menjadi berkas statis
dan **disajikan langsung oleh binary Go** (ADR-0001). **Tidak ada runtime Node.js di produksi.**

Kebutuhan grid berat ditangani satu pustaka tabel yang dipilih di awal — **TanStack Table** atau
**AG Grid** — dan dipakai seragam lewat `U-2` Pustaka Komponen. Tidak ada modul yang membangun
tabelnya sendiri.

## Rationale

Menyajikan berkas statis dari binary Go menghapus satu runtime, satu proses, dan satu rantai
pembaruan keamanan dari lingkungan produksi — konsisten dengan `D-08`.

React dipilih bukan karena paling canggih, melainkan karena kumpulan pustaka tabelnya paling
matang untuk kebutuhan yang benar-benar dimiliki aplikasi ini, dan karena materi belajarnya
paling melimpah bagi tim yang sedang berpindah dari Pega.

Satu pustaka tabel yang dipakai seragam adalah konsekuensi langsung dari `D-09`: 268 layar
bergrid yang masing-masing menafsirkan tabelnya sendiri akan menjadi beban pemeliharaan
terbesar aplikasi ini.

## Konsekuensi

### Positif

- Produksi hanya menjalankan satu proses: binary Go.
- `U-2` dapat dibangun lebih ramping daripada rencana awal berkat T-11.
- TypeScript memberi pemeriksaan kontrak antara layar dan API sejak waktu kompilasi — penting
  bagi tim yang belum terbiasa dengan JavaScript dinamis.

### Negatif / utang teknis

- **UX Pega yang padat ikut terwarisi** (`D-13`): grid lebar, form panjang, banyak tab. Kesempatan
  perbaikan UX ditunda ke fase pasca-migrasi, dan penundaan itu akan terasa oleh pengguna.
- **Paginasi keyset adalah perubahan perilaku, bukan pemeliharaan** (T-12). Grid yang hari ini
  memuat seluruh page list klipboard akan berperilaku berbeda saat dipaginasi di server; ini
  harus diuji per layar, bukan diasumsikan setara.
- Rendering awal bergantung pada JavaScript. Tidak ada halaman yang dapat dibaca tanpa SPA
  termuat lebih dulu.
- Memilih AG Grid versi komersial akan menambah lisensi; memilih TanStack Table menambah
  pekerjaan membangun perilaku grid sendiri. Pilihan finalnya **belum dibuat**.

### Risiko yang diterima secara sadar

- Tim menanggung dua kurva belajar sekaligus — Go dan React — pada proyek yang sama.
- Menyalin tata letak Pega berarti ikut menyalin kekakuannya; beberapa layar akan terasa tidak
  wajar di web dan tetap dibiarkan demi menghindari pelatihan ulang.

## Pertanyaan terbuka

- TanStack Table atau AG Grid? Pemilik: Lead Engineer, dengan persetujuan Work Owner bila
  berbiaya lisensi. Menghalangi penyelesaian tiket `U-2`.
- `D-12` menetapkan pemakaian lapangan untuk survei. Apakah `U-3`/`U-4` wajib berfungsi penuh di
  peramban ponsel, atau cukup layar survei `B-8` saja? Pemilik: Work Owner.
