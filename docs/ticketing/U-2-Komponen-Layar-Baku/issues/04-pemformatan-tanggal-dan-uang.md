---
title: "TKT-U2-004 — Pemilih tanggal dan pemformatan tanggal serta uang terpusat"
labels: [modul::U-2, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::2]
milestone: "Gelombang 2 — Kerangka UI"
epic: "Migrasi Claim PNC"
---

# TKT-U2-004 — Pemilih tanggal dan pemformatan tanggal serta uang terpusat

Status: ready-for-human
Kesiapan: siap
Modul: U-2 · Gelombang: 2 · Bergantung pada: TKT-U1-001
Requirement: FR-U2    Keputusan: D-51, D-49 butir 3    ADR: 0016, 0017    Risiko: R-12
Rule Pega yang digantikan: pemformatan yang tersebar — **411 pemakaian `TO_CHAR`** di SQL yang mengembalikan tanggal sebagai string `'dd/mm/yyyy'`
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi UI (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu tempat yang menentukan **bagaimana tanggal dan uang terlihat**, dipakai seluruh layar dan
seluruh laporan.

Nilai bisnisnya: `ADR-0016` menetapkan uang **disimpan presisi penuh dan dibulatkan hanya saat
ditampilkan**. Konsekuensinya, angka yang tersimpan tidak selalu sama dengan angka yang dilihat
pengguna — dan itu **aman hanya bila pembulatan tampilan dilakukan di satu tempat**. Satu layar
yang membulatkan dengan caranya sendiri akan menampilkan angka berbeda untuk data yang sama, dan
itu jenis laporan bug yang paling mahal ditelusuri.

## Ruang lingkup

- Fungsi pemformat tunggal untuk: tanggal, tanggal+waktu, nilai uang Rupiah, nilai uang valuta
  asing, dan persentase share reasuransi.
- Komponen **pemilih tanggal** yang mengirim tanggal dalam bentuk yang tidak ambigu, dan
  menampilkannya dalam WIB.
- Aturan pembulatan tampilan: **berapa desimal** untuk Rupiah, untuk valuta asing, dan untuk
  persentase share.
- Pemeriksaan otomatis: pemformatan tanggal atau uang **di luar** modul ini menggagalkan build.

## Non-goal

- **Tidak** melakukan perhitungan uang — itu milik server; klien hanya menampilkan.
- **Tidak** menangani konversi kurs — itu `B-5` dan master kurs `F-4`.

## Acceptance criteria

- [ ] Pemformatan tanggal dan uang berada di **tepat satu modul** — diuji pemindaian:
      `toLocaleString`, `Intl.NumberFormat`, dan pemformatan manual **0 kemunculan** di luar modul
      itu.
- [ ] Nilai uang diterima sebagai **string desimal** dari API dan ditampilkan tanpa kehilangan
      presisi — diuji dengan nilai 15 digit dan 4 desimal.
- [ ] Pembulatan hanya terjadi saat menampilkan; nilai yang dikirim kembali ke server **sama
      persis** dengan yang diterima bila pengguna tidak mengubahnya — diuji.
- [ ] Tanggal ditampilkan dalam WIB dan dikirim dalam bentuk tak ambigu — diuji dengan menjalankan
      peramban pada dua zona waktu berbeda dan hasilnya **sama**.
- [ ] Persentase share ditampilkan dengan **4 desimal**, konsisten dengan toleransi validasi
      `ROUND(SUM(share),4) BETWEEN 99.9999 AND 100.0001`.
- [ ] Nilai kosong, nol, dan negatif masing-masing punya tampilan yang **dapat dibedakan** —
      diuji ketiganya.

## Dependency / Blocked by

Bergantung pada `TKT-U1-001`.

**Terbuka tetapi tidak menahan:** berapa desimal untuk Rupiah pada tampilan belum ditetapkan Work
Owner (`ADR-0016` pertanyaan terbuka). Nilai sementara dipakai dan **ditandai di kode**; mengubahnya
kelak menyentuh satu berkas.

## Constraint keamanan, data, operasional

- **Angka pecahan di JavaScript tidak presisi.** Nilai uang tidak boleh melewati batas itu sebagai
  angka — diterima, disimpan di state, dan dikirim sebagai **string desimal**.
- Tampilan WIB memakai satu fungsi konversi; klien **tidak** menerapkan penyesuaian jam sendiri
  (`ADR-0017` butir 3).

## Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** mengembalikan aturan pembulatan; karena
terpusat, perubahannya seragam di seluruh layar.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm run test -- Format
grep -rInE "toLocaleString|Intl\.NumberFormat" src/ | grep -v src/format/   # HARUS 0 baris
TZ=UTC npm run test -- Format.tanggal
TZ=Asia/Jakarta npm run test -- Format.tanggal
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Uang disimpan presisi penuh, dibulatkan hanya saat tampil | `D-51` · `ADR-0016` |
| Toleransi share 4 desimal `99.9999`–`100.0001` | `D-51` |
| 411 `TO_CHAR` mengembalikan tanggal sebagai string | `D-20` · `ADR-0005` §3.2 |
| Angka pecahan JavaScript tidak presisi | `ADR-0016` Negatif/utang teknis |
| Konversi WIB tunggal | `ADR-0017` butir 3 · `F-5` |

## Comments
