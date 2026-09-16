---
title: "TKT-B10-003 — Cetak LOD ke tertanggung"
labels: [modul::B-10, tipe::migrasi, status::needs-info, prioritas::sedang, gelombang::4]
milestone: "Gelombang 4 — Persetujuan"
epic: "Migrasi Claim PNC"
---

# TKT-B10-003 — Cetak LOD ke tertanggung

Status: needs-info
Kesiapan: terhalang keputusan (kriteria kesamaan keluaran PDF)
Modul: **B-10 Akseptasi** · Gelombang: 4 · Bergantung pada: TKT-B10-001, TKT-S2-001
Requirement: FR-B10    Keputusan: D-11, D-57    ADR: 0011, 0022    Risiko: —
Rule Pega yang digantikan: pencetakan LOD dan `Job Scheduler/JobSendAutoLODKlaimPersonal` (**harian, jam 20:54**, menjalankan `Act_SendAutoLODKlaimPersonal`) · `Activity/PrintPDFAcceptanceNote-Act.xml`
Peran penguji gerbang 2: **PncManagerAdmin** dan **PncAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Surat **LOD — Letter of Discharge** dapat dicetak dan dikirim ke **tertanggung**, berisi nilai
ganti rugi yang disetujui.

Nilai bisnisnya: LOD adalah dokumen yang **dipegang nasabah**. Ia berbeda dari PLA/DLA yang
ditujukan ke koasuransi dan reasuransi — dan perbedaan itu sering tertukar.

## Ruang lingkup

- Pembentukan LOD sebagai **PDF** memakai engine sendiri (`ADR-0011`).
- Status cetak LOD pada Settlement Line, sehingga terlihat mana yang sudah dicetak.
- **Pengiriman otomatis LOD untuk lini Personal Accident** — perilaku job harian jam 20:54 yang
  ditemukan pada `D-57`.
- Pencatatan riwayat cetak dan kirim.

## Non-goal

- **Tidak** membangun engine PDF — itu `S-2`; modul ini memakainya.
- **Tidak** memutuskan mekanisme penjadwal — itu `S-6` dan `ADR-0022`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Apakah keluaran PDF wajib identik secara visual dengan Pega, atau cukup identik secara isi?** | **Work Owner** (`ADR-0011`) | Menentukan kriteria kelulusan gerbang 1 untuk seluruh dokumen cetak |
| **Pengiriman otomatis LOD PA harian jam 20:54 — dipertahankan?** | **Work Owner** | Perilaku ini ditemukan dari `Job Scheduler/`, dan seperti `AutoAcceptKomite` ia berjalan **tanpa pengguna** |
| Mekanisme penjadwal pada dua instans | **Lead Engineer + Infra** (`ADR-0022`) | Job yang berjalan dua kali berarti **LOD terkirim dua kali** ke nasabah |

## Acceptance criteria

- [ ] LOD terbentuk sebagai PDF berisi nilai ganti rugi yang disetujui, nomor klaim, dan data
      tertanggung — diuji pada 10 contoh.
- [ ] Kesamaan dengan keluaran Pega diuji sesuai kriteria yang ditetapkan Work Owner (isi atau
      visual) — 10 contoh.
- [ ] Status cetak tercatat; mencetak ulang **tidak menggandakan riwayat pembayaran** — diuji.
- [ ] Pengiriman otomatis LOD PA berjalan **tepat satu kali per hari**, walau dua instans hidup —
      diuji (`ADR-0022`).
- [ ] Kegagalan pembentukan PDF **tidak** membatalkan akseptasi — diuji.
- [ ] Gerbang 2: UAT **PncManagerAdmin** memeriksa LOD yang benar-benar tercetak.

## Dependency / Blocked by

`TKT-B10-001` · `TKT-S2-001` (engine PDF) · `TKT-S6-001` (penjadwal). **Terhalang tiga keputusan.**

## Constraint keamanan, data, operasional

- LOD memuat **data nasabah dan nilai ganti rugi**, dan dikirim ke luar perusahaan — salah tujuan
  berarti kebocoran.
- **LOD terkirim dua kali** akibat job berjalan ganda akan sampai ke nasabah; ini kesalahan yang
  terlihat pihak luar.
- Nilai pada LOD memakai pembulatan tampilan yang **sama dengan layar** (`TKT-U2-004`) — agar nasabah
  dan petugas melihat angka yang sama.

## Migrasi skema / rollout / rollback

Menambah kolom status cetak dan tabel riwayat kirim. Backward-compatible.

**Rollback:** LOD yang telanjur terkirim **tidak dapat ditarik**.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/lod/... -run TestBentukPDF
go test ./internal/app/lod/... -run TestCetakUlangTidakMenggandakan
go test ./internal/app/jadwal/... -run TestLODOtomatisSekaliSehari
go run ./cmd/tools/banding-pdf lod-contoh/ baseline-pega/
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| LOD ditujukan ke tertanggung, berbeda dari PLA/DLA | `CONTEXT.md` — **LOD** |
| `JobSendAutoLODKlaimPersonal` harian 20:54 | `D-57` · `Job Scheduler/` |
| PDF dibangun sendiri di Go | `D-11` · `ADR-0011` |
| Kriteria kesamaan keluaran PDF belum ditetapkan | `ADR-0011` Pertanyaan terbuka |

## Comments
