# 0020 — Pertahankan dua basis perhitungan TAT untuk keperluan yang berbeda

Status: Accepted
Tanggal keputusan: 2026-09-11    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-50`, `D-49` butir 10, `D-25` | `DATAMINING.GET_WORKING_HOURS@ASMD` (18 pemakaian di 7 berkas) | `GETSELISIHJAM.fnc:22` | `HRD_LBR`
Terkait: CONTEXT.md#TAT, ADR-0008, ADR-0017, modul `S-2`, `S-7`, `B-8`

## Konteks

TAT — lama penyelesaian klaim — dihitung dengan **dua fungsi berbeda** yang tidak akan pernah
menghasilkan angka sama:

| Fungsi | Keluaran | Dipakai di |
|---|---|---|
| `DATAMINING.GET_WORKING_HOURS@ASMD` | jam kerja, memperhitungkan kalender libur | seluruh KPI dan laporan admin, serta penulisan kronologi TAT |
| `GETSELISIHJAM` | selisih jam mentah, dibagi 24 menjadi hari | **satu layar saja** — inbox Compliance |
| `HRD_LBR` | pemeriksaan hari libur | dipakai terpisah |

Dugaan awal bahwa ini adalah ketidakkonsistenan yang harus diseragamkan ternyata **menyesatkan**:
benar secara aritmetika, tetapi keduanya **tidak pernah dipakai mengukur hal yang sama**. Batas
pemakaiannya tidak tumpang tindih.

## Opsi yang dipertimbangkan

1. Seragamkan ke satu fungsi — `GET_WORKING_HOURS` untuk semua.
2. Seragamkan ke `GETSELISIHJAM` karena lebih sederhana.
3. **Pertahankan keduanya** dengan batas pemakaian yang sudah terbukti.

## Keputusan

Kedua basis **dipertahankan**, dengan batas pemakaian yang sudah terbukti dari sumber:

- **`GET_WORKING_HOURS`** → seluruh **KPI dan laporan admin**, ditambah **penulisan kronologi
  TAT**. Ini jalur yang angkanya dilaporkan ke manajemen.
- **`GETSELISIHJAM`** → **satu layar saja**, inbox Compliance, dengan keluaran **hari**.
- **`HRD_LBR`** → pemeriksaan hari libur, dipakai terpisah.

Karena `GET_WORKING_HOURS` dan `HRD_LBR` adalah objek **remote lewat DB Link**, keduanya masuk
lingkup ADR-0008 — dan logikanya **ditulis ulang di Go**, bukan sekadar dipanggil lewat API:
perhitungan jam kerja dan kalender libur adalah **aturan bisnis**, bukan pengambilan data.

Butir 10 `D-49` tetap berlaku: `GETSELISIHJAM.fnc:22` yang mengembalikan `0` saat galat —
tak terbedakan dari nol hari — **diperbaiki** (ADR-0017).

## Rationale

Menyeragamkan akan mengubah angka pada salah satu dari dua jalur. Bila yang berubah adalah jalur
KPI, angka yang dilaporkan ke manajemen berubah tanpa ada perubahan kinerja nyata. Bila yang
berubah adalah inbox Compliance, satuan yang dilihat petugas berubah dari hari menjadi jam kerja.

Keduanya adalah perubahan perilaku yang tidak diminta siapa pun.

Menulis ulang logika jam kerja di Go — alih-alih memanggilnya lewat API — diperlukan karena
fungsi itu dipanggil di dalam kalkulasi laporan massal. Menjadikannya panggilan jaringan per baris
akan menghancurkan kinerja laporan.

## Konsekuensi

### Positif

- Tidak ada angka TAT yang berubah tanpa diminta.
- Kalender libur dan jam kerja menjadi aturan bisnis milik aplikasi, dapat diuji tanpa
  ketergantungan pada database lain.
- Menghapus 18 pemanggilan lintas DB Link dari jalur laporan.

### Negatif / utang teknis

- **Dua basis perhitungan hidup permanen**, dan perbedaannya akan terus menimbulkan pertanyaan
  "mengapa angkanya berbeda" dari pengguna yang membandingkan dua layar.
- **Kalender libur harus dimiliki dan dipelihara aplikasi ini.** Sumbernya hari ini adalah
  `HRD_LBR` milik sistem HRD; menulis ulang logikanya berarti ikut memutuskan dari mana daftar
  hari libur diperoleh setiap tahun — dan itu **belum ditetapkan**.
- Menulis ulang perhitungan jam kerja berisiko menghasilkan selisih dengan implementasi lama pada
  kasus batas (hari libur di tengah, lembur, akhir pekan berturut-turut) — dan kasus batas itulah
  yang paling sering muncul pada klaim bermasalah.

### Risiko yang diterima secara sadar

- Angka TAT adalah angka yang dilaporkan ke manajemen. Selisih sekecil apa pun setelah migrasi
  akan terlihat dan harus dapat dijelaskan.
- Definisi jam kerja (mulai, selesai, istirahat) tidak terdokumentasi; ia hanya ada di dalam
  fungsi remote yang sumbernya belum dibaca.

## Pertanyaan terbuka

- Sumber `DATAMINING.GET_WORKING_HOURS@ASMD` — kapan tersedia untuk dibaca? Tanpa itu aturan jam
  kerjanya tidak dapat ditulis ulang. Pemilik: DBA (`R-03`). Menghalangi tiket `S-7`.
- Dari mana daftar hari libur diperoleh setiap tahun di sistem baru, dan siapa yang mengisinya?
  Pemilik: Work Owner. Menghalangi penyelesaian tiket `F-4` dan `F-5`.
