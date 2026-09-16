# 0017 — Perbaiki sembilan cacat aturan uang; replikasi satu yang perilakunya memang benar

Status: Accepted
Tanggal keputusan: 2026-09-11    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-49`, `P-5` | lihat tabel di bawah untuk `berkas:baris` tiap butir
Terkait: ADR-0015, ADR-0016, ADR-0027, modul `B-2`…`B-10`

## Konteks

`P-5` menetapkan prinsip **kesetaraan perilaku lebih dulu**: sistem baru meniru sistem lama,
termasuk keanehannya, kecuali untuk perbaikan yang **diputuskan secara eksplisit**. Tanpa daftar
perbaikan yang disepakati di muka, setiap selisih pada uji kesetaraan menjadi perdebatan tentang
apakah ia cacat atau perbaikan.

Analisis Fase 1 menemukan **sepuluh cacat pada aturan yang menghitung atau memvalidasi uang**.
Masing-masing harus diputuskan sebelum gerbang 1 dijalankan — bukan sesudahnya.

## Opsi yang dipertimbangkan

1. Perbaiki seluruhnya.
2. Replikasi seluruhnya demi kesetaraan sempurna.
3. **Putuskan satu per satu**, dan daftarnya mengikat uji kesetaraan.

## Keputusan

**Sembilan cacat diperbaiki; satu direplikasi** karena perilakunya memang benar.

| # | Cacat | Bukti | Keputusan |
|---|---|---|---|
| 1 | Toleransi spreading berupa pencocokan substring; `199.99` dan `1100.0` lolos | `Activity/InputRegister_act-Act.xml:13183` | **PERBAIKI** (ADR-0016) |
| 2 | Ambang Rp 50 juta memakai **tiga operator berbeda** di tiga rule | `SetEmailKomite:1445` · `SetEmailKomiteAdjuster:968` · `SetEmailKomiteSalvage:888` | **PERBAIKI** — satu operator |
| 3 | Penyesuaian 7 jam diterapkan **asimetris di dalam satu kondisi validasi** | `InputRegister_act:5788` + `:4805` | **PERBAIKI** |
| 4 | Kurs mengabaikan tanggal kerugian | `GETCURRENCYSTANDARD.fnc:3` vs `:14` | **PERBAIKI** (ADR-0015) |
| 5 | Kurs tidak ditemukan → `RETURN 1` | `GETCURRENCYSTANDARD.fnc:22` | **PERBAIKI** (ADR-0015) |
| 6 | `LIMIT_TOP` diabaikan + tie-breaker acak | `EmailKomiteBerjenjangSimasnet_sql:98-99` | **PERBAIKI sebagian** (ADR-0014) — model kumulatif dipertahankan, `LIMIT_TOP` menjadi validasi master; tie-breaker acak hanya ada di 2 kueri Simasnet |
| **7** | Tanggal PLA/DLA dari pengguna dibuang, diganti `SYSDATE` | `INSERT_PLADLA.prc:67`, `:136`, `:177` | **REPLIKASI** — perilaku sekarang sudah benar |
| 8 | `NilaiSalvage` hanya ditambahkan bila baris terakhir kebetulan bertipe salvage | `ValidasiSisaTSI:817` vs `:1489` | **PERBAIKI** — salvage selalu ditambahkan |
| 9 | `INSERT_SALVAGE` menulis `IDSALVAGE = NULL` saat update | `INSERT_SALVAGE.prc:47` | **PERBAIKI** + perlu hitungan DBA atas baris rusak |
| 10 | `GETSELISIHJAM` gagal → `RETURN 0` jam, tak terbedakan dari nol | `GETSELISIHJAM.fnc:22` | **PERBAIKI** |

Butir 7 direplikasi karena tanggal PLA/DLA memang **tanggal sistem menerbitkan dokumen**, bukan
tanggal yang dipilih pengguna. Parameter `TTGLPLADLA` yang diterima lalu dibuang **bukan cacat**;
sistem baru boleh menghapus parameter itu seluruhnya.

## Rationale

Memperbaiki seluruhnya tanpa memeriksa satu per satu akan mengubah butir 7 menjadi "perbaikan"
yang justru merusak: mengizinkan pengguna memilih tanggal dokumen resmi yang dikirim ke reasuransi.

Mereplikasi seluruhnya berarti membawa `RETURN 1` pada kegagalan kurs ke sistem baru — cacat yang
membuat klaim besar lolos tanpa komite.

Jalan tengahnya adalah satu daftar yang diputuskan di muka dan mengikat pengujian.

## Konsekuensi

### Positif

- Setiap selisih pada gerbang 1 punya satu dari dua jawaban yang jelas: ia salah satu dari 13
  butir perbaikan, atau ia bug.
- Cacat yang paling berbahaya — kurs `RETURN 1`, spreading substring, `IDSALVAGE = NULL` —
  ditutup sebelum data baru bertambah di atasnya.
- Butir 7 menjadi contoh konkret bagi tiket `B-9` bahwa tidak semua keanehan adalah cacat.

### Negatif / utang teknis

- **Daftar perbaikan eksplisit `P-5` bertambah dari 4 menjadi 13 butir.** Setiap butir menambah
  satu kelas selisih yang harus dijelaskan pada setiap pengujian, untuk setiap modul terdampak.
- **Data historis tetap mengandung akibat cacat itu.** Memperbaiki aturannya tidak memperbaiki
  baris yang sudah telanjur salah — khususnya butir 9, yang menuntut hitungan DBA atas baris
  rusak dan kemungkinan perbaikan data.
- Uji kesetaraan menjadi lebih rumit dibaca: hasil "berbeda" tidak lagi otomatis berarti gagal.

### Risiko yang diterima secara sadar

- Klaim yang dulu lolos kini dapat tertolak (butir 1, 4, 5). Ini disadari dan diterima sebagai
  perbaikan, dengan dampak operasional yang harus disiapkan.
- Butir 6 hanya diperbaiki sebagian: tie-breaker acak pada dua kueri Simasnet tetap ada sampai
  ada keputusan tersendiri.

## Pertanyaan terbuka

- **Berapa banyak baris yang rusak akibat butir 9** (`IDSALVAGE = NULL`), dan apakah datanya
  diperbaiki sebelum migrasi? Pemilik: DBA + Work Owner. Menghalangi penyelesaian tiket `B-12`.
- Apakah data historis yang terdampak butir 1 (total spreading di luar toleransi) dibiarkan apa
  adanya? Pemilik: Work Owner.
