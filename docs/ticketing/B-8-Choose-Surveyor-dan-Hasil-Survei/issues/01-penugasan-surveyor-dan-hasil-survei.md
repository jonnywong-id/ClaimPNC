---
title: "TKT-B08-001 — Penugasan surveyor dan pencatatan hasil survei"
labels: [modul::B-8, tipe::migrasi, status::needs-info, prioritas::sedang, gelombang::4]
milestone: "Gelombang 4 — Persetujuan"
epic: "Migrasi Claim PNC"
---

# TKT-B08-001 — Penugasan surveyor dan pencatatan hasil survei

Status: needs-info
Kesiapan: **terhalang artefak** — When rule `IsSurvey` hilang
Modul: **B-8 Choose Surveyor** · Gelombang: 4 · Bergantung pada: TKT-B06-001, TKT-U2-003
Requirement: FR-B8    Keputusan: D-12, D-16    ADR: 0010, 0019    Risiko: R-16
Rule Pega yang digantikan: tahap **`Choose Surveyor`** / `InputSurveyor` pada `Flow/Register_Flow.xml` · harness `InboxSurvey_Harness`, `SurveyorsInbox`, `ViewDetailHasilSurveyorInternal1` · `INSERT_SURVEYORLIST` · master `MasterLoginSurvey`
Peran penguji gerbang 2: **PNCSurveyor** dan **PncPICTeknik**

## Hasil yang diharapkan (dan nilai bisnisnya)

Petugas teknis menugaskan surveyor, dan surveyor mencatat temuan lapangan beserta **foto
dokumentasi** — termasuk dari perangkat di lapangan.

Nilai bisnisnya: hasil survei adalah **dasar penetapan nilai kerugian**. Survei yang lambat
dicatat memanjangkan TAT, dan foto yang tidak terunggah membuat penilaian dilakukan tanpa bukti.

## Ruang lingkup

- Pemilihan surveyor dari master, memisahkan **internal (ASM)** dan **eksternal**.
- Pencatatan hasil survei: temuan, tanggal survei, lokasi survei, dan catatan.
- **Unggah foto dokumentasi** memakai komponen unggah `TKT-U2-003`, melalui satu jalur API storage
  (`ADR-0010`).
- Dukungan **peramban ponsel** untuk pengisian di lapangan (`D-12`).

## Non-goal

- **Tidak** menghitung KPI adjuster — itu `TKT-B08-002`.
- **Tidak** menetapkan nilai kerugian — itu `B-5`.
- **Tidak** membangun mekanisme penyimpanan dokumen — itu `S-1`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **When rule `IsSurvey`** — hilang dari export | **Tim Pega** (`R-16`) | Menentukan **kapan** sebuah klaim wajib disurvei; tanpa itu syaratnya harus ditebak |
| Jenis surveyor per objek (`jenisSurveyor` pada objek) — daftar sahnya | **Work Owner** | Menentukan isi master surveyor |

## Acceptance criteria

- [ ] Surveyor dipilih dari master; surveyor **tidak aktif tidak muncul** sebagai pilihan — diuji.
- [ ] Surveyor internal dan eksternal **dibedakan**, dan surveyor eksternal memerlukan data
      kontak — diuji keduanya.
- [ ] Hasil survei dapat disimpan beserta **minimal satu foto**, dan foto dapat dibuka kembali —
      diuji.
- [ ] Pengisian hasil survei berfungsi pada peramban ponsel lebar **390 px**, termasuk unggah dari
      kamera — diuji.
- [ ] Kegagalan unggah foto **tidak membatalkan** hasil survei yang sudah diisi — pengguna dapat
      mengulang unggahan tanpa mengetik ulang (`ADR-0010`).
- [ ] Perubahan hasil survei tercatat di jejak audit.
- [ ] Gerbang 1: penugasan dan hasil survei **sama dengan Pega** pada 20 klaim contoh.
- [ ] Gerbang 2: UAT **PNCSurveyor** di lapangan, bukan hanya di kantor.

## Dependency / Blocked by

`TKT-B06-001` · `TKT-U2-003` · `TKT-S1-001` (jalur penyimpanan dokumen). **Terhalang Tim Pega.**

## Constraint keamanan, data, operasional

- Foto survei dapat memuat **data nasabah dan lokasi**; ia tunduk pembatasan akses yang sama dengan
  dokumen klaim (`S-1`).
- **Tidak ada atomisitas** antara foto di storage dan metadatanya (`ADR-0010`) — kegagalan setelah
  foto terkirim harus diberitahukan, bukan diam.
- Pengisian di lapangan berarti **jaringan tidak stabil** — kegagalan jaringan harus dapat
  dibedakan dari penolakan server (`TKT-U2-003`).

## Migrasi skema / rollout / rollback

Menambah tabel hasil survei dan kaitan fotonya. Backward-compatible.

**Rollback:** hasil survei yang tersimpan tetap ada; foto di storage **tidak ikut terhapus**
(`ADR-0010`).

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/survei/... -run TestPenugasanSurveyor
go test ./internal/app/survei/... -run TestUnggahFotoGagalTidakMembatalkan
npm run test:e2e -- --grep "hasil survei" --device "iPhone 12"
go run ./cmd/s8 banding --modul B-8 --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Tahap `Choose Surveyor` pada flow utama | `Flow/Register_Flow.xml` |
| Surveyor internal dan eksternal | `CONTEXT.md` — **Surveyor / Loss Adjuster** |
| Pemakaian lapangan | `D-12` · `docs/Steering/06-MODULE-BREAKDOWN.md` `B-8` |
| Dokumen lewat satu jalur API storage | `D-16` · `ADR-0010` |
| `IsSurvey` hilang | `docs/verifikasi-bukti-adr.md` §15 baris `B-8` · `R-16` |

## Comments
