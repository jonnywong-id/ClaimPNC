---
title: "TKT-S8-002 — Pembanding hasil dan klasifikasi selisih terhadap 13 butir P-5"
labels: [modul::S-8, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-S8-002 — Pembanding hasil dan klasifikasi selisih terhadap 13 butir P-5

Status: ready-for-human
Kesiapan: siap — menunggu urutan kerja `TKT-S8-001`, bukan pihak lain
Modul: **S-8 Perkakas Uji Kesetaraan** · Gelombang: 1 · Bergantung pada: TKT-S8-001
Requirement: FR-S8    Keputusan: D-42, D-49, D-54    ADR: 0017, 0027    Risiko: R-14, R-19
Rule Pega yang digantikan: **tidak ada** — modul ini 100% baru
Peran penguji gerbang 2: **Lead Engineer**; keluarannya dibaca **Work Owner**

## Hasil yang diharapkan (dan nilai bisnisnya)

Perkakas yang membandingkan hasil kedua sisi dan **memilah setiap selisih menjadi dua tumpukan**:
yang **terpetakan** ke salah satu dari 13 butir perbaikan `P-5`, dan yang **tidak terpetakan**.

Nilai bisnisnya terletak pada pemilahan itu, bukan pada pembandingannya. `D-54` menetapkan selisih
yang cocok dengan 13 butir itu **lolos otomatis**, sementara selisih di luar itu **wajib persetujuan
Work Owner tertulis**. Tanpa klasifikasi otomatis, setiap selisih akan menumpuk di meja Work Owner —
dan gerbang 1 berubah dari kendali menjadi hambatan.

Sekaligus inilah pagar terhadap `R-19`: `P-5` yang dipatuhi buta akan **menyalin cacat hitungan uang
dan waktu**. Perkakas ini yang memaksa setiap selisih dijawab, bukan dilewati.

## Ruang lingkup

- Pembandingan hasil kedua sisi, termasuk **nilai uang berdesimal** dan **tanggal/jam**.
- Klasifikasi tiap selisih terhadap **13 butir `P-5`** (`D-49`) — keluarannya menyebut **butir mana**.
- Antrean selisih **tidak terpetakan** sebagai daftar yang menunggu persetujuan Work Owner tertulis.
- Laporan per modul: berapa kasus dijalankan, berapa setara, berapa terpetakan, berapa menunggu.
- Penanganan dua modul **tanpa baseline**: `F-3` dan `S-5` dilaporkan sebagai **tidak dapat diuji
  setara**, bukan sebagai lulus (`D-42`).

## Non-goal

- **Tidak** memutuskan selisih mana yang dapat diterima. Selisih di luar 13 butir adalah
  **keputusan Work Owner**, dan perkakas ini hanya menyajikannya.
- **Tidak** memperbaiki kode yang menghasilkan selisih.

## Acceptance criteria

- [ ] Selisih yang cocok dengan salah satu dari **13 butir `P-5`** ditandai **beserta nomor
      butirnya** — diuji pada minimal satu kasus per butir, ketiga belasnya.
- [ ] Selisih **di luar 13 butir** masuk antrean persetujuan dan **tidak pernah ditandai lolos** —
      diuji dengan selisih yang sengaja dibuat di luar daftar.
- [ ] Modul dengan selisih tak terpetakan yang belum disetujui **dilaporkan belum lulus gerbang 1** —
      diuji.
- [ ] Perbandingan **nilai uang** benar sampai desimal terkecil yang dipakai — diuji dengan nilai
      yang berbeda hanya di desimal: **terdeteksi sebagai selisih**, bukan dibulatkan sama.
- [ ] Perbandingan **tanggal dan jam** memperhitungkan zona waktu dan pembulatan — diuji dengan
      selisih detik.
- [ ] **`F-3` dan `S-5` dilaporkan sebagai "tidak dapat diuji setara"**, bukan sebagai lulus —
      diuji; keduanya tidak punya baseline Pega (`D-42`).
- [ ] Laporan per modul memuat **angka**: kasus dijalankan, setara, terpetakan, menunggu persetujuan.
- [ ] Gerbang 2: ditinjau **Lead Engineer**, dan keluarannya dibaca **Work Owner** pada satu modul
      percobaan sebelum dipakai untuk seluruh modul.

## Dependency / Blocked by

`TKT-S8-001`. **Tidak terhalang keputusan** — `D-49` dan `D-54` sudah menetapkan aturannya lengkap.
Yang menahan hanyalah urutan pekerjaan.

## Constraint keamanan, data, operasional

- Rekaman selisih memuat **data nasabah nyata** (staging memuat data produksi apa adanya, `D-64`).
  Laporan selisih **tidak boleh di-commit** dan tidak boleh dikirim ke luar tanpa penyamaran.
- **Perkakas ini tidak boleh melonggarkan penilaiannya sendiri.** Menambahkan toleransi agar lebih
  banyak kasus tampak setara akan membatalkan gunanya — dan kesetaraan yang dilaporkannya menjadi
  tidak berarti.
- Dua modul tanpa baseline **tidak boleh dihitung lulus** hanya karena tidak ada pembandingnya.

## Migrasi skema / rollout / rollback

Menambah tabel hasil pembandingan di lingkungan uji. Tidak menyentuh skema klaim.

**Rollback:** mengembalikan versi perkakas. Persetujuan Work Owner yang sudah diberikan atas selisih
**tetap berlaku** dan tidak perlu diulang.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./cmd/s8/... -run TestKlasifikasi13ButirP5
go test ./cmd/s8/... -run TestSelisihDiLuarDaftarTidakPernahLolos
go test ./cmd/s8/... -run TestSelisihDesimalTerdeteksi
go test ./cmd/s8/... -run TestModulTanpaBaselineTidakDihitungLulus
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Selisih cocok 13 butir lolos otomatis; di luar itu wajib persetujuan tertulis | `D-54` |
| Perkakas wajib **mengklasifikasikan**, bukan hanya melaporkan | `D-54` bagian Konsekuensi |
| Daftar perbaikan `P-5` berjumlah 13 butir | `D-49` · `ADR-0017` · `R-19` |
| `F-3` dan `S-5` tidak punya baseline Pega | `D-42` bagian Pengecualian |
| Uji kesetaraan adalah gerbang pertama tiap modul | `BRD §21.1` |

## Comments
