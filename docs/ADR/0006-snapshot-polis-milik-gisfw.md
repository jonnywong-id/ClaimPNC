# 0006 — Bekukan data polis sebagai snapshot saat registrasi; kepemilikannya tetap di GISFW

Status: Accepted
Tanggal keputusan: 2026-09-07    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-04`, `D-03`, `D-34` | `JSON_POLIS`, `JSON_KLAIM` | 141 activity GISFW pada export
Terkait: CONTEXT.md#Snapshot-Polis, ADR-0008, modul `B-1`, `B-2`

## Konteks

Data polis dimiliki **GISFW**, aplikasi tim lain. Meski begitu, **141 activity GISFW ikut berada
di export ini** dan dipakai saat klaim berjalan — batas kepemilikannya kabur di tingkat kode.

`D-03` dan `D-34` sudah mempersempit lingkup migrasi: hanya ruleset `GCNMFW`; area
bengkel/sparepart/supplier dan ruleset GKM **dikecualikan**. Persoalan yang tersisa adalah data
polis, yang tidak bisa dikecualikan karena setiap klaim bertumpu padanya.

## Opsi yang dipertimbangkan

1. **Simpan snapshot polis** saat klaim diregistrasi.
2. **Panggil API GISFW secara real-time** setiap kali data polis dibutuhkan.
3. **Ikut memigrasikan GISFW** dalam proyek ini.
4. Tunda — perlu koordinasi lebih dulu dengan tim GISFW.

## Keputusan

Domain Klaim **menyimpan snapshot polis pada saat klaim diregistrasi** — melanjutkan pola
`JSON_POLIS`/`JSON_KLAIM` yang sudah berjalan, tetapi dengan **skema eksplisit**, bukan JSON
bebas bentuk.

**Kepemilikan data polis tetap di GISFW.** Claim PNC tidak pernah menulis ke data polis; ia hanya
membekukan salinannya pada satu titik waktu.

## Rationale

Klaim adalah peristiwa yang dinilai menurut keadaan polis **pada saat kejadian dan registrasi** —
bukan menurut keadaan polis hari ini. Membaca polis secara real-time justru salah secara bisnis:
endorsemen yang terjadi setelah registrasi akan mengubah dasar penilaian klaim yang sudah
berjalan.

Snapshot juga membuat pemrosesan klaim **kebal terhadap ketersediaan sistem tim lain saat
runtime** — klaim tetap dapat diproses ketika GISFW sedang tidak dapat dihubungi.

Memigrasikan GISFW sekaligus akan melipatgandakan lingkup proyek dan menyeret tim lain ke dalam
jadwal yang bukan jadwal mereka.

## Konsekuensi

### Positif

- Nilai klaim tidak berubah diam-diam karena perubahan polis setelah registrasi.
- Ketergantungan runtime pada sistem tim lain hilang pada jalur pemrosesan klaim.
- Batas bounded context menjadi konkret dan dapat diuji: Claim PNC tidak punya jalur tulis ke
  data polis.

### Negatif / utang teknis

- **Snapshot dapat usang.** Bila polis dikoreksi setelah registrasi — pembatalan, endorsemen,
  perbaikan data tertanggung — klaim tetap memakai salinan lama sampai ada yang menyegarkannya
  secara sadar. Aturan penyegaran itu **belum ada**.
- **Duplikasi data** antara GISFW dan Claim PNC, dengan seluruh biaya penyimpanan dan kebingungan
  "mana yang benar" yang menyertainya.
- Merancang skema eksplisit pengganti `JSON_POLIS` menuntut pemahaman penuh atas bentuk JSON yang
  ada sekarang — dan bentuk itu tidak terdokumentasi, hanya tersirat dari cara rule memakainya.
- 141 activity GISFW di export harus dipilah satu per satu: mana yang benar-benar milik tim lain
  dan mana yang sudah menjadi logika klaim.

### Risiko yang diterima secara sadar

- Bila kelak GISFW menyediakan API polis yang andal, keputusan ini tetap dipertahankan — karena
  alasannya bisnis (membekukan dasar penilaian), bukan teknis (menghindari kopling).
- Ketidaksesuaian antara snapshot dan polis asli akan muncul dalam sengketa klaim, dan yang
  dipakai adalah snapshot.

## Pertanyaan terbuka

- Kapan sebuah snapshot boleh atau harus disegarkan, dan siapa yang berwenang memicunya?
  Pemilik: Work Owner. Menghalangi penyelesaian tiket `B-1`.
- Field polis apa saja yang wajib masuk snapshot? Daftar ini hanya dapat disusun dengan membaca
  seluruh pemakaian `JSON_POLIS` di rule, dan belum dikerjakan. Pemilik: Lead Engineer.
- Apakah tim GISFW menyetujui bahwa Claim PNC menyimpan salinan data mereka? Pemilik: Work Owner
  → Tim GISFW.
