---
title: "TKT-S6-001 — Penjadwal dan kunci satu pelaksana"
labels: [modul::S-6, tipe::fondasi, status::needs-info, prioritas::sedang, gelombang::7]
milestone: "Gelombang 7 — Sisa"
epic: "Migrasi Claim PNC"
---

# TKT-S6-001 — Penjadwal dan kunci satu pelaksana

Status: needs-info
Kesiapan: **terhalang keputusan rancangan (`ADR-0022`)**
Modul: **S-6 Job Terjadwal & Proses Otomatis** · Gelombang: 7 · Bergantung pada: TKT-F1-002
Requirement: FR-S6    Keputusan: D-57    ADR: 0022    Risiko: —
Rule Pega yang digantikan: mekanisme Job Scheduler Pega — `pyApplicableTo=Cluster`, `pyNodeTypesText=BackgroundProcessing`
Peran penguji gerbang 2: **PncAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Penjadwal yang menjalankan pekerjaan pada jamnya, dan menjalankannya **tepat satu kali** walau
aplikasi berjalan di beberapa node.

Nilai bisnisnya persis pada kata "tepat satu kali". Bila `JobSendAutoLODKlaimPersonal` berjalan di
tiga node sekaligus, nasabah menerima **tiga LOD**. Bila `ProcessClaimKredit` berjalan rangkap,
klaim kredit terbentuk ganda.

## Ruang lingkup

- Penjadwal dengan jadwal harian dan mingguan pada jam yang dapat dikonfigurasi.
- **Kunci agar satu job hanya berjalan di satu node** — setara `pyApplicableTo=Cluster` di Pega.
- Pencatatan tiap eksekusi: job apa, mulai kapan, selesai kapan, berhasil atau gagal.
- Perilaku saat job gagal: **tidak mengulang diam-diam** tanpa disadari.
- Cara **mematikan satu job** tanpa mematikan yang lain, dan tanpa rilis.

## Non-goal

- **Tidak** memindahkan isi kelima job — itu `TKT-S6-002`.
- **Tidak** memutuskan nasib `AutoAcceptKomite` — itu `TKT-S6-003`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Penjadwal berjalan di dalam aplikasi atau sebagai proses terpisah?** | **Lead Engineer** (`ADR-0022`) | Menentukan bentuk kuncinya dan cara job dipantau. `ADR-0022` masih `Proposed` |
| **Job yang terlewat karena aplikasi mati — dikejar atau dilewatkan?** | **Work Owner** | Mengejar `JobSendAutoLODKlaimPersonal` yang terlewat dua hari berarti mengirim LOD ganda |

## Acceptance criteria

- [ ] Job berjalan **tepat satu kali** walau tiga node hidup bersamaan — diuji dengan tiga
      instance: catatan eksekusi menunjukkan **satu**, bukan tiga.
- [ ] Job berjalan pada **jam yang dikonfigurasi**, bukan jam yang tertanam di kode — diuji dengan
      mengubah konfigurasi.
- [ ] Satu job dapat **dimatikan tanpa rilis** dan tanpa memengaruhi job lain — diuji.
- [ ] Kegagalan job **tercatat dan terlihat** — diuji dengan job yang sengaja gagal; kegagalannya
      tidak diam.
- [ ] Job yang berjalan lama **tidak menghalangi** job berikutnya berjalan — diuji.
- [ ] Node yang mati di tengah eksekusi **tidak meninggalkan kunci yang menggantung selamanya** —
      diuji dengan mematikan node paksa.
- [ ] Gerbang 2: UAT **PncAdmin**.

## Dependency / Blocked by

`TKT-F1-002`. **Terhalang `ADR-0022` yang masih `Proposed`.**

## Constraint keamanan, data, operasional

- Job berjalan **tanpa pengguna**. Setiap perubahan data olehnya tetap wajib masuk jejak audit
  (`S-5`) dengan identitas job — bukan tanpa pelaku.
- Selama Pega dan Go berjalan berdampingan (`D-05`), **job yang sama dapat aktif di kedua sistem**.
  Itu akan menggandakan efeknya. Pengalihan job wajib mematikan sisi Pega-nya lebih dulu, dan itu
  bagian dari rencana rollout — bukan hal yang bisa diserahkan ke kebetulan.

## Migrasi skema / rollout / rollback

Menambah tabel kunci dan tabel catatan eksekusi job. Tidak menyentuh tabel klaim. Backward-compatible.

**Rollout:** satu job dialihkan pada satu waktu, **setelah job yang sama dimatikan di Pega**.

**Rollback:** mematikan job di Go dan menyalakan kembali di Pega. Keduanya **tidak boleh menyala
bersamaan**.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/penjadwal/... -run TestSatuJobSatuNode
go test ./internal/app/penjadwal/... -run TestKunciTidakMenggantungSaatNodeMati
go test ./internal/app/penjadwal/... -run TestJobDapatDimatikanTanpaRilis
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 5 job, seluruhnya `pyApplicableTo=Cluster`, `pyIsEnabled=true` | `D-57` · direktori `Job Scheduler/` |
| Jam dan frekuensi tiap job | `D-57` tabel |
| Rancangan penjadwal belum diputuskan | `ADR-0022` (status `Proposed`) |
| Pega dan Go berjalan berdampingan | `D-05` |

## Comments
