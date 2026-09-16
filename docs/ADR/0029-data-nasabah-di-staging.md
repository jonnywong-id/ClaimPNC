# 0029 — Salin data nasabah apa adanya ke staging; samarkan email dan jangan pernah tulis data nasabah di dokumen

Status: Accepted
Tanggal keputusan: 2026-09-13 (`D-64`), 2026-09-14 (`D-69`)    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-64`, `D-69`, `D-40` (OPEN), `D-53`, `FR-R2` | `docs/verifikasi-bukti-adr.md` §7.4
Terkait: ADR-0025, ADR-0027, modul `S-8` + seluruh dokumen proyek

## Konteks

ADR-0027 menetapkan uji kesetaraan dijalankan atas **salinan data produksi di staging** — karena
data uji buatan terbukti tidak mampu memunculkan cacat yang ditemukan Fase 1. Konsekuensinya:
**data nasabah nyata berpindah ke lingkungan staging**.

Persoalan kedua terpisah tetapi sering tertukar dengan yang pertama: **apa yang boleh tertulis di
ADR dan tiket** yang akan di-commit. Nama seperti `MORASOTARDODOTARIGAN`, `ELLENSUPRIYATI`, dan
`INDRAGUNAWAN` **sudah tertulis apa adanya** di dokumen yang sudah disetujui (`D-15`,
`20-DETAIL-KOMITE-DBLINK.md` §1.5.4); alamat email tidak pernah.

## Opsi yang dipertimbangkan

Untuk isi lingkungan staging:
1. **Salin apa adanya**, hak akses staging diperketat sebagai kontrol pengganti.
2. Samarkan data nasabah sebelum disalin.
3. Pakai data uji buatan.

Untuk isi dokumen:
1. **Nama Operator ID boleh ditulis; alamat email selalu disamarkan; data nasabah tidak pernah.**
2. Keduanya disamarkan, termasuk nama Operator ID.
3. Keduanya boleh ditulis lengkap.

## Keputusan

**Isi lingkungan staging:** data produksi disalin **apa adanya, tanpa penyamaran**, dengan **hak
akses lingkungan staging diperketat** sebagai kontrol penggantinya.

**Isi dokumen yang di-commit:**

| Jenis nilai | Perlakuan |
|---|---|
| **Nama Operator ID** | **boleh ditulis lengkap** — preseden sudah ada di Steering yang disetujui, dan diperlukan agar tiket dapat menunjuk hardcode mana yang harus dihapus |
| **Alamat email** | **selalu disamarkan** — bagian sebelum `@` diganti, domain boleh tampak |
| **Data nasabah** — nomor polis, nama tertanggung, nomor klaim asli, NPWP, nomor rekening | **tidak pernah ditulis**; dirujuk dengan `berkas:baris` atau nama kolom saja |
| **Kredensial, kunci API, hostname/IP produksi** | **tidak pernah ditulis**; dirujuk dengan `berkas:baris` + nama elemen (ADR-0025) |

Kedua keputusan berdiri sendiri: yang pertama mengatur **isi lingkungan**, yang kedua mengatur
**isi dokumen**.

## Rationale

Penyamaran data sebelum disalin akan menghilangkan justru sifat data yang membuatnya berguna
untuk uji kesetaraan — nilai ekstrem, karakter tak terduga, dan kombinasi yang tidak akan terpikir
dibuat. Cacat seperti toleransi spreading substring hanya terlihat pada data yang nyata.

Untuk dokumen, nama Operator ID diperlukan demi ketelusuran: tanpanya, tiket `F-4` tidak dapat
menunjuk hardcode mana yang harus dihapus. Alamat email **tidak menambah ketelusuran apa pun**,
sehingga tidak ada alasan menuliskannya.

## Konsekuensi

### Positif

- Uji kesetaraan berjalan atas data yang benar-benar mewakili produksi.
- Aturan penulisan dokumen menjadi tegas dan dapat diperiksa — tidak ada penilaian kasus per kasus.
- Tiket tetap dapat menunjuk hardcode secara spesifik tanpa membocorkan data nasabah.

### Negatif / utang teknis

- **Staging menjadi lingkungan yang memuat data nasabah nyata** — nomor polis, nama tertanggung,
  NPWP, nomor rekening, dan **data medis** pada lini PA dan Travel. Perlindungannya sepenuhnya
  bergantung pada hak akses, bukan pada sifat datanya.
- **Klasifikasi staging naik setara produksi** untuk keperluan keamanan. Siapa yang punya akses,
  berapa lama salinan disimpan, dan kapan dimusnahkan menjadi pertanyaan yang harus dijawab — dan
  **belum dijawab**.
- Pembatasan akses data medis yang `FR-R2` terapkan pada peran Analyst Doctor dan RCL Dokter
  **berlaku juga di staging**, bukan hanya produksi. Ini menambah pekerjaan pada `F-3` di
  lingkungan yang biasanya dianggap bebas.
- Keputusan ini **berdampingan dengan `D-40` yang masih terbuka**: export sudah memuat kredensial
  plaintext, dan kini salinan data nasabah menyusul ke lingkungan yang sama-sama belum ditetapkan
  pengamanannya.

### Risiko yang diterima secara sadar

- Kebocoran dari staging akan setara dengan kebocoran dari produksi. Tidak ada mitigasi teknis
  tambahan — hanya hak akses.
- Nama pegawai tertulis lengkap di dokumen yang di-commit dan akan dibaca banyak orang. Ini
  diterima demi ketelusuran, dengan kesadaran penuh bahwa ia data pribadi.

## Pertanyaan terbuka

- **Siapa yang menyetujui akses staging, berapa lama salinan disimpan, dan bagaimana prosedur
  pemusnahannya?** Pemilik: Tim Infra/Security + Compliance — pihak yang sama dengan `D-40`.
- Apakah salinan data disegarkan berkala, dan apakah setiap penyegaran menuntut persetujuan ulang?
  Pemilik: Work Owner.
- Apakah data medis di staging perlu perlakuan lebih ketat daripada data nasabah lainnya?
  Pemilik: Compliance.
