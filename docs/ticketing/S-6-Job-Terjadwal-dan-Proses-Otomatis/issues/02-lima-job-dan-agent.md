---
title: "TKT-S6-002 — Lima job terjadwal dan satu agent"
labels: [modul::S-6, tipe::migrasi, status::needs-info, prioritas::sedang, gelombang::7]
milestone: "Gelombang 7 — Sisa"
epic: "Migrasi Claim PNC"
---

# TKT-S6-002 — Lima job terjadwal dan satu agent

Status: needs-info
Kesiapan: **terhalang keputusan**
Modul: **S-6 Job Terjadwal & Proses Otomatis** · Gelombang: 7 · Bergantung pada: TKT-S6-001
Requirement: FR-S6    Keputusan: D-57    ADR: 0022    Risiko: —
Rule Pega yang digantikan: `Job Scheduler/JobHitungDeadlineToTemporaryClose-Job.xml` · `JobSendAutoLODKlaimPersonal-Job.xml` · `PNCMyReportKlaim3-Job.xml` · `ProcessClaimKredit-Job.xml` · `Agents/TATReportAgent-Agents.xml`
Peran penguji gerbang 2: **PncAdmin**, **PncKasir** (alert kasir), **PNCReportClaimInternal** (laporan)

## Hasil yang diharapkan (dan nilai bisnisnya)

Empat job dan satu agent berjalan di sistem baru dengan hasil yang sama.

Nilai bisnisnya tidak merata di antara keenamnya: `JobHitungDeadlineToTemporaryCLose` **mengubah
status klaim**, `JobSendAutoLODKlaimPersonal` **mengirim dokumen ke nasabah**, dan
`TransferAllCaseNotAssigned` **memindahkan pekerjaan antar petugas**. Ketiganya berakibat nyata
tanpa ada orang yang menyetujuinya saat itu.

> `JOBForKomiteKlaimPNC` **tidak termasuk di sini** — ia digarap terpisah di `TKT-S6-003` karena
> dampaknya adalah persetujuan klaim.

## Ruang lingkup

- `JobHitungDeadlineToTemporaryCLose` — mingguan 23:43 → `JobTemporaryCloseClaimPNC`.
- `JobSendAutoLODKlaimPersonal` — harian 20:54 → `Act_SendAutoLODKlaimPersonal`.
- `PNCMyReportKlaim3` — harian 08:00 → `ReportAI_Act`.
- `ProcessClaimKredit` — harian 10:00 → `CreateClaimCredit_Table`.
- `TATReportAgent` — tiap 30 menit → alert kasir belum transfer · `TransferAllCaseNotAssigned` ·
  `CreateCasePNCAgent_ActButton`.

## Non-goal

- **Tidak** mengubah jam maupun frekuensi — kecuali butir "job jalan 2 menit" diputuskan lain.
- **Tidak** menambah job baru.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Tiga job ber-`pyDescription = "job jalan 2 menit"` sementara konfigurasinya `Daily`/`Weekly`** — selisih **720× sampai 5.040×** | **Work Owner + Tim Pega** (`D-57` butir 2) | Salah satu dari keduanya salah. Menyalin yang keliru berarti job berjalan 720 kali lebih sering atau lebih jarang daripada yang dimaksud |
| **`pyBypassActivityAuthentication=true` pada agent** — dibawa ke sistem baru? | **Work Owner + Tim Infra/Security** (`D-57` butir 3) | Agent ini **memindahkan pekerjaan antar petugas** tanpa pemeriksaan otentikasi. Membawanya apa adanya berarti menyalin jalur yang tidak terkontrol |
| **`TransferAllCaseNotAssigned` memindahkan kasus berdasarkan aturan apa?** | **Work Owner** | Isi activity-nya ada, tetapi **maksud bisnisnya** — kasus mana yang layak dipindah dan ke siapa — tidak tertulis di mana pun |

## Acceptance criteria

- [ ] Keempat job dan agent berjalan pada jadwal yang ditetapkan — diuji per job.
- [ ] Hasil tiap job **sama dengan hasil Pega** atas data uji yang sama — dibandingkan lewat `S-8`
      pada minimal 10 kasus per job.
- [ ] `JobSendAutoLODKlaimPersonal` **tidak mengirim LOD ganda** bila dijalankan ulang — diuji.
- [ ] `JobHitungDeadlineToTemporaryCLose` mengubah status **hanya klaim yang memenuhi syarat** —
      diuji dengan klaim di kedua sisi batas tenggat.
- [ ] Setiap perubahan data oleh job **tercatat di jejak audit** dengan identitas job — diuji.
- [ ] Agent yang memindahkan kasus **mencatat asal dan tujuan penugasan** — diuji.
- [ ] Gerbang 2: UAT oleh peran yang menerima akibatnya — **PncKasir** untuk alert,
      **PNCReportClaimInternal** untuk laporan.

## Dependency / Blocked by

`TKT-S6-001` · `TKT-B06-002` (penugasan) · `TKT-S1-001` (LOD) · `TKT-S5-001` (jejak audit).
**Terhalang tiga keputusan `D-57`.**

## Constraint keamanan, data, operasional

- Job mengubah data **tanpa pengguna yang menyetujui saat itu**. Jejak auditnya adalah satu-satunya
  cara mengetahui apa yang terjadi.
- `JobSendAutoLODKlaimPersonal` **mengirim dokumen keluar ke nasabah** — kesalahan di sini terlihat
  oleh pihak luar dan tidak dapat ditarik kembali.
- Job aktif di Pega **dan** Go bersamaan akan menggandakan efeknya (`P-1`).

## Migrasi skema / rollout / rollback

Tidak menambah tabel bisnis; memakai catatan eksekusi dari `TKT-S6-001`.

**Rollout:** satu job pada satu waktu, dimulai dari yang **tidak mengirim apa pun keluar**
(`PNCMyReportKlaim3`), dan yang mengirim ke nasabah paling akhir.

**Rollback:** mematikan job di Go, menyalakan di Pega. **LOD yang telanjur terkirim tidak dapat
ditarik.**

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/job/... -run TestLODTidakGandaSaatDijalankanUlang
go test ./internal/app/job/... -run TestTemporaryCloseHanyaYangLewatTenggat
go run ./cmd/s8 banding --modul S-6 --per-job 10
go test ./internal/app/audit/... -run TestPerubahanOlehJobTercatat
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Jam, frekuensi, dan activity target tiap job | `D-57` tabel · direktori `Job Scheduler/` |
| Agent `pyTriggerInterval=1800`, `pyBypassActivityAuthentication=true` | `D-57` · `Agents/TATReportAgent-Agents.xml` |
| Agent merujuk `TransferAllCaseNotAssigned` dan alert kasir | `D-57` |
| Tiga job berdeskripsi "job jalan 2 menit" | `D-57` butir 2 · `docs/verifikasi-bukti-adr.md:2702-2705` |

## Comments
