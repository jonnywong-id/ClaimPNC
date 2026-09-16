---
title: "TKT-F5-003 — Kalender hari kerja dan hari libur"
labels: [modul::F-5, tipe::migrasi, status::needs-info, prioritas::sedang, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F5-003 — Kalender hari kerja dan hari libur

Status: needs-info
Kesiapan: **terhalang keputusan dan artefak** — sumber kalender belum ditetapkan; sumber `GET_WORKING_HOURS` belum diterima
Modul: F-5 · Gelombang: 1 · Bergantung pada: TKT-F5-001
Requirement: FR-F5    Keputusan: D-50, D-25    ADR: 0008, 0020    Risiko: R-03
Rule Pega yang digantikan: `DATAMINING.GET_WORKING_HOURS@ASMD` — **18 pemakaian di 7 berkas** · `HRD_LBR` (hari libur) · `GETSELISIHJAM` (dipakai satu layar inbox Compliance)
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`); angkanya diuji pengguna lewat `S-7`

## Hasil yang diharapkan (dan nilai bisnisnya)

Perhitungan **jam kerja** dan **hari libur** sebagai aturan bisnis milik aplikasi ini, bukan
panggilan ke database lain.

Nilai bisnisnya: angka TAT adalah angka yang **dilaporkan ke manajemen**. `D-50` menetapkan
logikanya ditulis ulang di Go, bukan dipanggil lewat API — karena `GET_WORKING_HOURS` dipakai
**di dalam kalkulasi laporan massal**, dan menjadikannya panggilan jaringan per baris akan
menghancurkan kinerja laporan.

## Ruang lingkup

- Perhitungan selisih **jam kerja** antara dua waktu, memperhitungkan akhir pekan dan hari libur.
- Master **hari libur** dan definisi **jam kerja** sebagai data (`F-4`), bukan konstanta di kode.
- Dua basis perhitungan dipertahankan sesuai `D-50`, dengan **batas pemakaian yang tegas**:
  jam kerja untuk KPI, laporan admin, dan kronologi TAT; selisih hari untuk satu layar inbox
  Compliance.
- Perbaikan butir 10 `D-49`: kegagalan **tidak** mengembalikan `0` yang tak terbedakan dari nol.

## Non-goal

- **Tidak** membangun laporan TAT — itu `S-7`.
- **Tidak** menyeragamkan kedua basis menjadi satu — `D-50` menolaknya, karena keduanya tidak
  pernah dipakai mengukur hal yang sama.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Sumber `DATAMINING.GET_WORKING_HOURS@ASMD`** — definisi jam kerja (mulai, selesai, istirahat) hanya ada di dalam fungsi remote yang belum dibaca | **DBA** (`R-03`) | Menulis ulang aturan yang belum pernah dibaca berarti menebak; dan angkanya dilaporkan ke manajemen |
| **Dari mana daftar hari libur diperoleh setiap tahun, dan siapa yang mengisinya?** | **Work Owner** | Menentukan apakah master hari libur diisi manual di `F-4` atau ditarik dari sistem HRD |

## Acceptance criteria

- [ ] Selisih jam kerja antara dua waktu **sama dengan** keluaran `GET_WORKING_HOURS` pada
      **30 pasangan waktu contoh** yang mencakup: akhir pekan, hari libur di tengah, lintas bulan,
      dan dua waktu di hari yang sama.
- [ ] Hari libur dibaca dari master, **bukan** dari konstanta di kode — diuji dengan menambah satu
      hari libur dan memeriksa hasil perhitungan berubah.
- [ ] Kegagalan perhitungan mengembalikan **galat**, bukan `0` — diuji dengan masukan tidak sah;
      nilai `0` hanya muncul bila selisihnya memang nol.
- [ ] `GETSELISIHJAM` hanya dipakai pada **satu layar** (inbox Compliance) — diuji pemindaian:
      pemanggilnya tepat satu.
- [ ] Perhitungan **tidak melakukan panggilan jaringan** — diuji dengan menjalankan uji tanpa
      akses jaringan.

## Dependency / Blocked by

- Bergantung pada `TKT-F5-001`.
- Membutuhkan master hari libur dan jam kerja dari `F-4`.
- **Terhalang** `R-03` (sumber fungsi remote) dan keputusan sumber kalender.

## Constraint keamanan, data, operasional

- **Tidak boleh** menjadi panggilan jaringan per baris — itu alasan utama `D-50` memilih menulis
  ulang alih-alih membungkusnya dengan API.
- Angka TAT dilaporkan ke manajemen; **selisih sekecil apa pun setelah migrasi akan terlihat** dan
  harus dapat dijelaskan.

## Migrasi skema / rollout / rollback

Menambah master hari libur dan jam kerja (tabel baru, tidak menyentuh tabel Pega).

**Rollback:** kembali memanggil `GET_WORKING_HOURS` lewat DB Link **tidak tersedia sebagai
rollback** — `ADR-0008` menghapus DB Link. Bila hasil perhitungan meragukan, yang dilakukan adalah
menahan rilis laporan, bukan mengembalikan DB Link.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/waktu/... -run TestJamKerja
# bandingkan 30 pasangan dengan keluaran fungsi lama (dijalankan DBA di staging)
go run ./cmd/tools/banding-jam-kerja pasangan.csv > hasil.csv && diff hasil.csv baseline.csv
go test ./... -run TestJamKerjaTanpaJaringan
grep -rIn "GETSELISIHJAM" internal/ | wc -l     # HARUS tepat 1 pemanggil
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `GET_WORKING_HOURS` 18 pemakaian di 7 berkas | `D-50` · `ADR-0020` |
| Dua basis dipertahankan dengan batas pemakaian | `D-50` |
| Logika ditulis ulang di Go, bukan dipanggil lewat API | `D-50` · `docs/Steering/10-API-STRATEGY.md` §8.4 |
| `GETSELISIHJAM` gagal → `RETURN 0` | `Database/GETSELISIHJAM.fnc:22` · `D-49` butir 10 |
| Kalender libur menjadi master milik aplikasi; sumbernya belum ditetapkan | `ADR-0020` pertanyaan terbuka |

## Comments
