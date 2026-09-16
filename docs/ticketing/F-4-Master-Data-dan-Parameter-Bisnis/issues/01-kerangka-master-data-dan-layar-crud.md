---
title: "TKT-F4-001 — Kerangka master data dan layar CRUD baku"
labels: [modul::F-4, tipe::fondasi, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F4-001 — Kerangka master data dan layar CRUD baku

Status: needs-info
Kesiapan: **terhalang keputusan** — alur persetujuan perubahan master belum ditetapkan
Modul: F-4 · Gelombang: 1 · Bergantung pada: TKT-F2-001, TKT-F3-005, TKT-U2-001
Requirement: FR-F4    Keputusan: D-15, D-59    ADR: 0023, 0025, 0026    Risiko: R-08
Rule Pega yang digantikan: nilai yang tertanam di dalam activity — **66 email**, **24 Operator ID**, **8 ambang komite**, dan 7 ambang uang non-komite; ditambah layar master yang tersebar di `Section/`
Peran penguji gerbang 2: **User Admin** dan peran pemilik masing-masing master (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu pola untuk seluruh master: satu tabel, satu layar daftar, satu layar ubah, satu jejak audit —
sehingga menambah master ke-30 tidak berarti menulis ulang semuanya.

Nilai bisnisnya: perubahan kebijakan bisnis — ambang, penerima notifikasi, kode status — menjadi
**pekerjaan pengguna bisnis**, bukan permintaan deployment. Itulah alasan nilai-nilai itu
di-hardcode sejak awal di sistem lama: karena mengubahnya menuntut orang teknis.

## Ruang lingkup

- Pola generik master: definisi kolom, validasi, dan layar CRUD yang dihasilkan dari definisi itu.
- **Setiap perubahan master tercatat di jejak audit** (`S-5`) — siapa, kapan, nilai sebelum,
  nilai sesudah.
- Penegakan izin lewat `TKT-F3-005`: master yang berbeda dapat dimiliki peran yang berbeda.
- Soft delete pada seluruh master (`ADR-0012`) — master yang dinonaktifkan tetap dapat dirujuk
  data lama.

## Non-goal

- **Tidak** mengisi master mana pun — masing-masing punya tiketnya sendiri.
- **Tidak** menetapkan daftar ≥29 kelompok — itu `TKT-F4-006`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Apakah perubahan master butuh alur persetujuan, dan berlaku untuk master yang mana?** | **Work Owner** | Menentukan apakah pola generiknya memuat status "menunggu persetujuan" atau tidak. Menambahkannya belakangan menyentuh seluruh master |
| **DDL 9 tabel baru** (`R-08`) | **DBA** | Menahan bagian yang menyentuh tabel nyata; pola generiknya sendiri tidak terhalang |

**Yang perlu disadari saat memutuskan alur persetujuan.** `D-59` menghapus pemisahan tugas —
seorang pengguna yang memiliki menu master ambang komite dapat **mengubah kewenangan persetujuan
uang sendirian**. Bila alur persetujuan hendak dipasang di suatu tempat, master inilah kandidat
paling kuat.

## Acceptance criteria

- [ ] Menambah master baru menuntut **≤ 50 baris** definisi, tanpa menulis layar baru — dibuktikan
      dengan satu master contoh.
- [ ] Setiap perubahan master menghasilkan **tepat satu baris jejak audit** dengan nilai sebelum
      dan sesudah — diuji.
- [ ] Penghapusan master adalah **soft delete**; data lama yang merujuknya **tetap dapat
      ditampilkan** — diuji.
- [ ] Pengguna tanpa izin menu master menerima **`403`** dan **tidak melihat menunya** — keduanya
      diuji.
- [ ] Validasi per master dideklarasikan bersama definisinya, dan galatnya muncul di field yang
      benar (`TKT-U2-002`).
- [ ] Daftar master memakai komponen tabel baku `TKT-U2-001` — tidak ada tabel yang ditulis
      khusus.

## Dependency / Blocked by

Bergantung pada `TKT-F2-001`, `TKT-F3-005`, `TKT-U2-001`, dan `TKT-S5-002` untuk pencatatan audit.
**Terhalang keputusan alur persetujuan.**

## Constraint keamanan, data, operasional

- Master menentukan **perilaku bisnis** — ambang komite menentukan siapa berwenang menyetujui uang.
  Perubahannya adalah tindakan bernilai tinggi dan **wajib tercatat**.
- Master **tidak boleh** memuat rahasia. Kredensial dan kunci API bukan master data; tempatnya
  menunggu `D-40`.
- Nama pegawai di master tunduk aturan penulisan `D-69` bila dikutip ke dokumen.

## Migrasi skema / rollout / rollback

Menambah tabel master baru — **tidak menyentuh tabel yang dibaca Pega**. Selama masa paralel,
**Pega tetap membaca nilai hardcode di rule-nya sendiri**; master baru hanya dipakai aplikasi Go.
Itu berarti dua sumber kebenaran sementara, dan **itu disengaja** — menyatukannya baru terjadi saat
modul yang memakainya pindah.

**Rollback:** tabel dibiarkan; aplikasi kembali memakai nilai sebelumnya.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/masterdata/...
go test ./internal/app/masterdata/... -run TestPerubahanTercatatDiAudit
go test ./internal/adapter/http/... -run TestIzinMenuMaster
npm run test -- MasterCrud
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 66 email · 24 Operator ID · 8 ambang komite · 3 hostname | `D-15` terverifikasi · `docs/Steering/12-CROSSCUTTING.md` §3.3 |
| ≥29 kelompok master | `T-13` · `docs/Steering/05-DOMAIN-MODEL.md` §5 |
| Perubahan bernilai bisnis wajib tercatat | `D-28` · `ADR-0026` · `BRD §21.2` #9 |
| Tidak ada pemisahan tugas | `D-59` · `ADR-0023` |
| DDL 9 tabel baru belum ada | `R-08` · `docs/verifikasi-bukti-adr.md` §15 baris `F-4` |

## Comments
