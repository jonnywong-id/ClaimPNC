---
title: "TKT-F5-001 — Seam Clock dan penyimpanan UTC"
labels: [modul::F-5, tipe::fondasi, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F5-001 — Seam Clock dan penyimpanan UTC

Status: needs-info
Kesiapan: **terhalang keputusan** — representasi otoritatif tanggal kejadian belum ditetapkan
Modul: F-5 · Gelombang: 1 · Bergantung pada: TKT-F1-001
Requirement: FR-F5    Keputusan: D-49 butir 3    ADR: 0017    Risiko: R-12
Rule Pega yang digantikan: pemakaian waktu langsung yang tersebar — `SYSDATE` di SQL, `@CurrentDateTime` di activity, dan **118 titik penyesuaian waktu di 36 activity**
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu antarmuka pengambil waktu yang **dapat digantikan saat pengujian**, dan aturan tegas bahwa
waktu disimpan dalam **UTC**.

Nilai bisnisnya: aturan tanggal adalah gerbang validasi terberat di registrasi klaim — tanggal
kejadian ≤ tanggal lapor ≤ tanggal terima dokumen ≤ hari ini, ditambah batas periode polis. Aturan
seperti itu **tidak dapat diuji** bila waktunya diambil langsung dari jam sistem; kasus batas
tengah malam hanya dapat diuji dengan waktu yang dapat dikendalikan.

## Ruang lingkup

- Antarmuka `Clock` dideklarasikan di lapisan Domain, dengan dua adapter: jam nyata dan jam
  terkendali untuk pengujian.
- **Larangan memanggil waktu sistem langsung** di luar adapter — ditegakkan pemeriksaan pola.
- Aturan penyimpanan: seluruh kolom waktu disimpan **UTC**; konversi ke WIB hanya di batas
  tampilan.
- Tipe terpisah untuk **tanggal murni** (tanpa jam) dan **waktu penuh**, agar keduanya tidak
  tertukar.

## Non-goal

- **Tidak** melakukan konversi tampilan — itu `TKT-F5-002`.
- **Tidak** menangani kalender hari kerja — itu `TKT-F5-003`.
- **Tidak** memperbaiki data historis yang mungkin tergeser — itu pekerjaan migrasi data.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Akibat |
|---|---|---|
| **`DateOfLoss` dan `ReportDate`: tanggal murni atau waktu penuh?** | **Work Owner** | Menentukan tipe kolom dan apakah kasus tengah malam bisa terjadi sama sekali. Bila tanggal murni, seluruh kelas cacat `R-12` hilang untuk kedua field ini |
| **`"WIB"` (25 pemakaian) versus `Asia/Jakarta` (137 pemakaian) — hasilnya identik?** | **Lead Engineer + DBA** | Bila berbeda, ada data produksi yang sudah tergeser dan migrasi data harus menanganinya |

## Acceptance criteria

- [ ] Tidak ada pemanggilan waktu sistem di luar adapter Clock — diuji pemindaian: `time.Now()`
      **0 kemunculan** di luar satu berkas adapter.
- [ ] Aturan bisnis yang bergantung waktu dapat diuji dengan waktu yang ditetapkan — dibuktikan
      satu uji yang memajukan waktu melewati tengah malam dan hasilnya **deterministik**.
- [ ] Seluruh kolom waktu disimpan UTC — diuji dengan menulis lalu membaca kembali pada dua
      pengaturan zona waktu sistem yang berbeda dan hasilnya **sama**.
- [ ] Tipe tanggal murni dan waktu penuh **tidak dapat saling ditugaskan** tanpa konversi
      eksplisit — dibuktikan kegagalan kompilasi pada uji percobaan.
- [ ] Uji zona waktu dijalankan pada `TZ=UTC` dan `TZ=Asia/Jakarta`, keduanya **lulus**.

## Dependency / Blocked by

Bergantung pada `TKT-F1-001`. Terhalang dua keputusan di atas.

**Yang bergantung padanya:** `TKT-F2-006` (pergantian tahun pada nomor klaim), seluruh aturan
tanggal `B-2`, dan perhitungan TAT `S-7`.

## Constraint keamanan, data, operasional

- `SYSDATE` **dilarang** di SQL kecuali di generator nomor klaim (`ADR-0009`), dan di sana ia
  memang tanggal server — bukan tanggal bisnis.
- `R-12` berlaku pada **penyalinan data ke staging**, bukan hanya migrasi akhir. Salinan yang
  bergeser menghasilkan **selisih palsu** di setiap pengujian kesetaraan.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema pada tiket ini. **Rollback:** mengembalikan adapter Clock; tidak ada data
yang berubah.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
grep -rIn "time.Now()" internal/ | grep -v adapter/clock   # HARUS 0 baris
TZ=UTC go test ./...
TZ=Asia/Jakarta go test ./...
go test ./internal/domain/... -run TestAturanTanggalTengahMalam
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 118 titik penyesuaian waktu di 36 activity | `docs/verifikasi-bukti-adr.md` §15 baris `F-5` |
| `"WIB"` 25× versus `Asia/Jakarta` 137× | idem |
| Penyesuaian 7 jam asimetris | `Activity/InputRegister_act-Act.xml:5788` dan `:4805` · `D-49` butir 3 |
| Seam Clock, penyimpanan UTC, konversi WIB tunggal | `docs/Steering/04-FUTURE-ARCHITECTURE.md` §3.2 · `06-MODULE-BREAKDOWN.md` `F-5` |
| Zona waktu bergeser saat migrasi data | `R-12` |

## Comments
