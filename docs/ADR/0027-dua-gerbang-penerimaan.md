# 0027 — Luluskan setiap modul lewat dua gerbang: uji kesetaraan otomatis, lalu UAT bisnis

Status: Accepted
Tanggal keputusan: 2026-09-10 … 2026-09-12    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-42`, `D-53`, `D-54`, `D-60`, `P-5`, `BRD §21.1` | `docs/verifikasi-bukti-adr.md` §15
Terkait: ADR-0003, ADR-0017, ADR-0028, ADR-0029, seluruh modul

## Konteks

`BRD §21.1` menetapkan dua gerbang untuk setiap modul: **uji kesetaraan otomatis**, lalu **UAT
pengguna bisnis**. Gerbang pertama itu dilayani satu modul perkakas — `S-8`.

`S-8` adalah **satu-satunya modul yang memblokir cutover setiap modul lain**. Menaruhnya di
gelombang akhir berarti tidak ada satu modul pun dapat lulus gerbang 1 sampai gelombang itu tiba,
yang membatalkan gagasan Strangler Fig (ADR-0003). Karena itu `D-42` memindahkannya ke
**gelombang 1**, dan total modul menjadi **33**, bukan 32.

## Opsi yang dipertimbangkan

Untuk lingkungan pengujian:
1. Tembak produksi langsung — dilarang aturan kerja proyek.
2. **Salinan data produksi di staging**, Pega staging vs Go staging.
3. Data uji buatan.

Untuk kewenangan menyetujui selisih:
1. Setiap selisih disetujui Work Owner.
2. **Selisih yang cocok dengan 13 butir `P-5` lolos otomatis**; sisanya menuntut persetujuan.
3. Tim pengembang memutuskan sendiri.

## Keputusan

**Gerbang 1 — uji kesetaraan otomatis.** Dijalankan atas **salinan data produksi di lingkungan
staging**, dengan **Pega staging** dan **Go staging** sebagai kedua sisi pembanding, dijalankan
oleh **tim pengembang**.

**Klasifikasi selisih:** selisih yang **cocok dengan salah satu dari 13 butir `P-5`** (ADR-0017)
lolos otomatis dan cukup dicatat. Selisih **di luar 13 butir itu** wajib **persetujuan Work Owner
secara tertulis** sebelum modul dinyatakan lulus. Perkakas `S-8` karenanya wajib
**mengklasifikasikan** setiap selisih, bukan sekadar melaporkannya.

**Gerbang 2 — UAT pengguna bisnis**, berlaku menurut kelompok modul:

| Kelompok | Gerbang 2 | Penguji |
|---|---|---|
| **Modul bisnis** (`B-1`…`B-14`, `S-1`…`S-4`, `S-6`, `S-7`, `U-1`, `U-3`…`U-6`) | **berlaku** | peran bisnis pemakai inbox/layar modul itu |
| **Modul fondasi** (`F-1`…`F-5`, `S-5`, `S-8`, `U-2`) | **tidak berlaku** | gerbang 1 + **persetujuan Work Owner** |

Memaksakan gerbang 2 pada modul yang tidak punya peran pemakai akan membuat tiket fondasi macet
di gerbang yang tidak dapat dilewati siapa pun.

## Rationale

Data uji buatan tidak memadai, dan buktinya konkret: cacat yang ditemukan pada Fase 1 — toleransi
spreading berupa pencocokan substring, kurs `RETURN 1`, `IDSALVAGE = NULL` — **muncul dari data
nyata yang tidak akan terpikir dibuat**.

Meminta persetujuan untuk selisih yang sudah diputuskan eksplisit di `D-49` menambah beban tanpa
menambah kendali. Yang benar-benar menuntut perhatian adalah selisih **yang tidak terduga** —
itulah yang dipagari.

## Konsekuensi

### Positif

- Setiap modul punya kriteria lulus yang sama dan dapat diperiksa ulang.
- Selisih tak terduga tidak dapat lolos diam-diam; ia menjadi antrean persetujuan yang terlihat.
- Modul fondasi tidak tersandera gerbang yang tidak punya penguji.

### Negatif / utang teknis

- **`S-8` harus selesai sebelum modul mana pun dapat lulus.** Ia menjadi jalur kritis paling awal,
  dan keterlambatannya menunda seluruh migrasi.
- **`S-8` menuntut kemampuan menembak Pega dari luar** untuk membandingkan hasil. Siapa yang boleh,
  di lingkungan mana, dan apakah Pega staging tersedia — **belum dikonfirmasi**. Tiket `S-8`
  karenanya berstatus **TERHALANG** meski modulnya di gelombang 1.
- **Salinan data produksi membawa data nasabah nyata ke staging** (ADR-0029), dengan seluruh
  kewajiban pengamanan yang menyertainya.
- `R-12` — pergeseran zona waktu saat migrasi data — berlaku pada **proses penyalinan itu
  sendiri**, bukan hanya pada migrasi akhir. Salinan yang bergeser akan menghasilkan selisih palsu
  di setiap pengujian.
- Dua modul **tidak punya baseline Pega** untuk diuji setara: `F-3` dan `S-5` (ADR-0028).

### Risiko yang diterima secara sadar

- **Waktu pengguna bisnis untuk UAT belum dialokasikan resmi** (`C-9`, `R-15`). Gerbang 2
  bergantung pada ketersediaan orang yang sedang menjalankan operasional harian.
- Persetujuan tertulis Work Owner untuk setiap selisih tak terduga membuat Work Owner menjadi
  titik tunggal yang dapat menghambat kelulusan modul.

## Pertanyaan terbuka

- **Apakah Pega staging yang dapat ditembak dari luar tersedia?** Pemilik: Tim Pega + Tim Infra.
  Ini penghalang utama `S-8`.
- Kapan waktu pengguna bisnis untuk UAT dialokasikan resmi? Pemilik: manajemen (`R-15`).
- Apakah salinan data produksi disegarkan berkala, dan berapa lama disimpan? Pemilik: Work Owner
  + Infra (bertaut dengan ADR-0029).
