# S-6 · Job Terjadwal & Proses Otomatis

| | |
|---|---|
| **Nama di sistem lama** | **Job Scheduler** (5 job) dan **Agent** (1) — `Job Scheduler/`, `Agents/TATReportAgent-Agents.xml` |
| **Kode modul** | `S-6` |
| **Gelombang** | 7 — Sisa |
| **Ukuran** | 5 job · 1 agent · 8 activity target |
| **Bergantung pada** | `F-1` Kerangka Aplikasi |
| **Kesiapan** | **SEBAGIAN** |

## Apa yang dikerjakan modul ini

Lima pekerjaan yang berjalan sendiri pada jam tertentu, tanpa ada orang yang menekan tombol,
ditambah satu agent yang berjalan tiap 30 menit.

| Job | Frekuensi | Jam | Yang dikerjakan |
|---|---|---|---|
| `JOBForKomiteKlaimPNC` | Harian | **06:00** | **Menyetujui komite secara otomatis** (`AutoAcceptKomite`) |
| `JobHitungDeadlineToTemporaryCLose` | **Mingguan** | 23:43 | Menutup sementara klaim yang lewat tenggat |
| `JobSendAutoLODKlaimPersonal` | Harian | 20:54 | Mengirim LOD klaim personal otomatis |
| `PNCMyReportKlaim3` | Harian | 08:00 | Laporan AI |
| `ProcessClaimKredit` | Harian | 10:00 | Membuat klaim kredit |
| `TATReportAgent` (agent) | tiap **30 menit** | — | Alert kasir belum transfer · pengalihan kasus tak tertugaskan |

Seluruhnya `pyIsEnabled=true`, `pyApplicableTo=Cluster` (`D-57`).

## Temuan yang mengubah rencana

| Anggapan | Terverifikasi |
|---|---|
| Daftar job tidak diketahui (`R-02`) | **Lengkap dan terverifikasi** — `R-02` tertutup pada tingkat artefak (`D-57`) |
| Persetujuan komite selalu dilakukan manusia | **`AutoAcceptKomite` berjalan tiap hari jam 06:00 tanpa pengguna sama sekali** |
| Agent berjalan dengan identitas pengguna | **`pyBypassActivityAuthentication=true`** — agent **melewati pemeriksaan otentikasi** |

## Kenapa modul ini SEBAGIAN, bukan SIAP

Artefaknya lengkap. Yang belum ada adalah **jawaban atas apa yang artefak itu ungkapkan** — terutama
bahwa ada jalur persetujuan klaim yang **tidak melewati kontrol mana pun**.

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | **`AutoAcceptKomite` menyetujui komite otomatis tiap hari 06:00** — benar demikian? Perilaku ini **tidak tercatat di dokumen mana pun** | **Work Owner** (`D-57` butir 1) |
| **Keputusan** | Tiga job ber-`pyDescription = "job jalan 2 menit"` sementara konfigurasinya `Daily`/`Weekly` — **selisih 720× sampai 5.040×**. Mana yang benar? | **Work Owner + Tim Pega** (`D-57` butir 2) |
| **Keputusan** | **`pyBypassActivityAuthentication=true`** pada agent — diterima untuk sistem baru? | **Work Owner + Tim Infra/Security** (`D-57` butir 3) |
| **Keputusan** | Rancangan penjadwal di sistem baru (satu node atau seluruh node) | **Lead Engineer** (`ADR-0022`) |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-S6-001](issues/01-penjadwal-dan-kunci-satu-pelaksana.md) | Penjadwal dan kunci satu pelaksana | `needs-info` |
| [TKT-S6-002](issues/02-lima-job-dan-agent.md) | Lima job terjadwal dan satu agent | `needs-info` |
| [TKT-S6-003](issues/03-auto-accept-komite.md) | Persetujuan komite otomatis jam 06:00 | `needs-info` |
