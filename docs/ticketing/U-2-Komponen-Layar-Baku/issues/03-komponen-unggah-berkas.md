---
title: "TKT-U2-003 — Komponen unggah berkas"
labels: [modul::U-2, tipe::fondasi, status::ready-for-human, prioritas::sedang, gelombang::2]
milestone: "Gelombang 2 — Kerangka UI"
epic: "Migrasi Claim PNC"
---

# TKT-U2-003 — Komponen unggah berkas

Status: ready-for-human
Kesiapan: siap
Modul: U-2 · Gelombang: 2 · Bergantung pada: TKT-U1-001, TKT-U2-002
Requirement: FR-U2    Keputusan: D-16    ADR: 0010    Risiko: —
Rule Pega yang digantikan: pola unggah pada `Section/` dokumen klaim dan foto survei; **tiga mekanisme penyimpanan lama** yang disatukan menjadi satu jalur
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi UI (`D-60`); perilakunya diuji pengguna lewat `S-1` dan `B-8`

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu komponen unggah berkas yang dipakai dokumen klaim maupun foto survei, dengan umpan balik
kemajuan dan penanganan kegagalan yang jelas.

Nilai bisnisnya: unggah dokumen adalah titik yang **paling sering gagal** karena melibatkan sistem
lain (`ADR-0010`). Kegagalan yang tidak dijelaskan membuat petugas mengulang unggahan berkali-kali
dan meninggalkan berkas yatim di storage.

## Ruang lingkup

- Komponen unggah: pilih berkas, seret-lepas, kemajuan, batal, dan hapus sebelum disimpan.
- Batas jenis dan ukuran berkas **dari konfigurasi**, bukan konstanta di kode.
- Penanganan kegagalan yang membedakan: berkas ditolak (jenis/ukuran), gagal jaringan, dan
  **storage tidak dapat dihubungi** — ketiganya butuh tindakan berbeda dari pengguna.
- Dukungan **peramban ponsel** untuk kasus foto survei lapangan (`D-12`).

## Non-goal

- **Tidak** membangun alur dokumen klaim — itu `S-1`.
- **Tidak** memutuskan penyimpanan — dokumen disimpan lewat API storage internal (`ADR-0010`);
  komponen ini hanya mengirimkannya.
- **Tidak** menangani berkas yatim — pembersihannya milik `S-1`, dan mekanismenya belum dirancang.

## Acceptance criteria

- [ ] Berkas dengan jenis di luar daftar yang diizinkan **ditolak sebelum diunggah**, dengan pesan
      yang menyebut jenis yang diizinkan.
- [ ] Berkas melebihi batas ukuran ditolak dengan pesan yang **menyebut batasnya dalam MB**.
- [ ] Kemajuan unggah terlihat dan **dapat dibatalkan**; pembatalan menghentikan permintaan,
      bukan hanya menyembunyikan indikatornya — diuji dengan memeriksa permintaan jaringan.
- [ ] Ketiga jenis kegagalan menghasilkan pesan yang **berbeda dan dapat ditindaklanjuti** —
      diuji dengan tiga skenario tiruan.
- [ ] Unggah dari peramban ponsel (lebar 390 px) dapat memakai kamera — diuji pada satu perangkat
      atau emulator.
- [ ] Batas jenis dan ukuran dibaca dari konfigurasi — diuji dengan mengubah nilainya dan
      memeriksa perilaku berubah tanpa mengubah kode.

## Dependency / Blocked by

Bergantung pada `TKT-U1-001` dan `TKT-U2-002`.

## Constraint keamanan, data, operasional

- Nama berkas dari pengguna **tidak pernah dipakai apa adanya** sebagai jalur penyimpanan.
- Komponen **tidak menyimpan berkas di peramban** lebih lama dari yang dibutuhkan untuk unggah.
- Dokumen klaim memuat **data nasabah** — pratinjau di klien tidak boleh dicache oleh peramban
  secara persisten.
- **Tidak ada atomisitas** antara berkas dan metadata (`ADR-0010`): bila metadata gagal disimpan
  setelah berkas terkirim, komponen wajib memberi tahu pengguna dengan jelas, bukan diam.

## Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** mengembalikan versi komponen.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm run test -- Unggah
npm run test -- Unggah.kegagalan
npm run test:e2e -- --grep "unggah dokumen" --device "iPhone 12"
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Dokumen lewat satu jalur API storage internal; DB hanya metadata | `D-16` · `ADR-0010` |
| Tiga mekanisme penyimpanan lama disatukan | `D-16` |
| Tidak ada atomisitas berkas dan metadata | `ADR-0010` Negatif/utang teknis |
| Pemakaian lapangan untuk survei | `D-12` · `docs/Steering/06-MODULE-BREAKDOWN.md` `B-8` |

## Comments
