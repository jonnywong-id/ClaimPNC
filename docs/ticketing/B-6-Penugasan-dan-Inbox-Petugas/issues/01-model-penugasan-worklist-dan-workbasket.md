---
title: "TKT-B06-001 — Model penugasan Worklist dan Workbasket"
labels: [modul::B-6, tipe::migrasi, status::ready-for-human, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B06-001 — Model penugasan Worklist dan Workbasket

Status: ready-for-human
Kesiapan: siap
Modul: **B-6 Penugasan & Inbox** · Gelombang: 3 · Bergantung pada: TKT-F3-005, TKT-U2-001
Requirement: FR-B6    Keputusan: D-26    ADR: 0019, 0023    Risiko: —
Rule Pega yang digantikan: tabel `DATAPEGA.PC_ASSIGN_WORKLIST` (**dibaca 18 rule**) dan `DATAPEGA.PC_ASSIGN_WORKBASKET` (6 rule) · harness `PNCInboxAdmin`, `UserTeknisInbox`, `UserInbox_Harness`
Peran penguji gerbang 2: **PncAdmin**, **PncPICTeknik**, **PncRCLPUCL**

## Hasil yang diharapkan (dan nilai bisnisnya)

Tugas klaim muncul di tempat yang benar: **Worklist** bila sudah bertuan, **Workbasket** bila
masih antrean bersama.

Nilai bisnisnya: pembagian ini **bukan warisan kosong** — ia mencerminkan cara kerja. Tahap yang
butuh kesinambungan penanganan memakai Worklist; tahap yang dikerjakan sebuah tim memakai
Workbasket. `D-13` menetapkan alur kerja tetap sama agar pengguna tidak perlu dilatih ulang, dan
mengubah model penugasan adalah perubahan yang **paling terasa setiap hari**.

## Ruang lingkup

- Tabel Tugas milik aplikasi, menggantikan tabel engine Pega (`ADR-0004`).
- Dua model: **Worklist** (bertuan) dan **Workbasket** (antrean bersama) — satu tugas selalu berada
  di salah satunya, tidak pernah di keduanya.
- Layar inbox per model, memakai komponen tabel baku (`TKT-U2-001`).
- Perpindahan tugas antar tahap saat klaim maju.

## Non-goal

- **Tidak** menentukan **siapa** yang menerima tugas — itu `TKT-B06-002` (aturan routing).
- **Tidak** menangani penguncian — itu `TKT-B06-003`.
- **Tidak** menangani penugasan komite: penjenjangan komite menugaskan ke **operator bernama**,
  bukan ke workbasket (`T-7`), dan itu milik `B-7`.

## Acceptance criteria

- [ ] Satu tugas berada **di Worklist atau di Workbasket**, tidak pernah keduanya — ditegakkan
      constraint, diuji.
- [ ] Sembilan tahap Worklist dan tiga tahap Workbasket sesuai daftar `D-26` — diuji per tahap.
- [ ] Inbox Worklist menampilkan **hanya tugas milik pengguna**; inbox Workbasket menampilkan
      antrean yang boleh diambil perannya — diuji dengan dua pengguna berbeda.
- [ ] Batas data cabang dan lini bisnis ditegakkan **di kueri** — jumlah total yang ditampilkan
      juga tidak memuat data di luar batas (`TKT-F3-005`).
- [ ] Klaim yang maju ke tahap berikutnya **memindahkan tugasnya**, dan tugas lama tertutup —
      diuji.
- [ ] Gerbang 1: isi inbox **sama dengan Pega** untuk peran dan pengguna yang sama pada data
      staging — diuji pada 5 peran.
- [ ] Gerbang 2: UAT **PncAdmin**, **PncPICTeknik**, **PncRCLPUCL**.

## Dependency / Blocked by

`TKT-F3-005` (izin dan batas data) · `TKT-U2-001` (tabel baku) · `TKT-F2-004` (skema).

## Constraint keamanan, data, operasional

- Tabel penugasan menggantikan tabel engine Pega. Selama masa paralel, **penulis tunggal per
  tabel** berlaku (`P-1`): Pega masih menulis tabelnya sendiri sampai `B-6` lulus gerbang 2.
- Inbox menampilkan data nasabah; penyaringan **tidak boleh** dilakukan setelah data terambil —
  itu membocorkan lewat jumlah baris.

## Migrasi skema / rollout / rollback

Menambah tabel Tugas milik aplikasi. Backward-compatible; Pega tidak membacanya.

**Rollback:** tugas yang sudah dibuat di sistem baru **tetap ada**; pengguna kembali memakai inbox
Pega untuk tahap yang belum pindah. Tidak ada data yang hilang, tetapi **tugas dapat muncul di dua
tempat** selama masa transisi — itu konsekuensi yang harus disampaikan ke pengguna sebelum rilis.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/penugasan/... -run TestWorklistAtauWorkbasket
go test ./internal/app/penugasan/... -run TestInboxPerPengguna
go test ./internal/app/penugasan/... -run TestBatasDataCabang
go run ./cmd/s8 banding --modul B-6 --peran 5
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Dua model penugasan dan daftar tahapnya | `D-26` · `ADR-0019` · `Flow/Register_Flow.xml` |
| `PC_ASSIGN_WORKLIST` dibaca 18 rule; `PC_ASSIGN_WORKBASKET` 6 rule | `D-21` · `ADR-0004` |
| Istilah Worklist, Workbasket, Tugas | `CONTEXT.md` |
| Hanya 4 workbasket di seluruh export | `T-7` |

## Comments
