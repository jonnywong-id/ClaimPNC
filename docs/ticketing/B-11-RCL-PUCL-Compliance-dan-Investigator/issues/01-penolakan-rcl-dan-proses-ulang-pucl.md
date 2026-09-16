---
title: "TKT-B11-001 — Penolakan (RCL) dan proses ulang (PUCL)"
labels: [modul::B-11, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::4]
milestone: "Gelombang 4 — Persetujuan"
epic: "Migrasi Claim PNC"
---

# TKT-B11-001 — Penolakan (RCL) dan proses ulang (PUCL)

Status: needs-info
Kesiapan: **terhalang artefak dan keputusan**
Modul: **B-11 RCL/PUCL** · Gelombang: 4 · Bergantung pada: TKT-B06-001
Requirement: FR-B11    Keputusan: D-26, D-59    ADR: 0019, 0021, 0023, 0026    Risiko: R-16
Rule Pega yang digantikan: tahap **`RCL/PUCL`** pada `Flow/Register_Flow.xml` · harness `RCL_Harness`, `RCLPUCL_Harness`, `PNC_MasterTolakKlaim` · Ticket rule `SendtoPUCL` (`Flow/Register_Flow.xml:3197`) — **hilang dari export** · pemicu terverifikasi: `Activity/KomitePost_Reject-Act.xml:3172` dengan kondisi `GroupPanel=="002"` (`:3121`)
Peran penguji gerbang 2: **PncRCLPUCL** dan **PncManagerAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Klaim dapat **ditolak** dengan alasan yang tercatat, dan klaim yang sudah ditolak atau ditutup
dapat **diproses ulang**.

Nilai bisnisnya: penolakan adalah keputusan yang **berhadapan langsung dengan nasabah**. Alasan
penolakan yang tidak tercatat rapi membuat sengketa tidak dapat ditelusuri — dan proses ulang yang
tidak terlacak membuat klaim yang sama bisa dinilai dua kali dengan hasil berbeda.

## Ruang lingkup

- Penolakan klaim dengan **alasan dari master** (`PNC_MasterTolakKlaim`), bukan teks bebas.
- Proses ulang klaim yang sudah ditolak atau ditutup, dengan jejak yang menghubungkan keduanya.
- Antrean **Workbasket `RCLPUCL`** — antrean bersama, diambil siapa pun yang berwenang.
- **Lompatan lateral ke PUCL** — satu-satunya pemicu yang terverifikasi ada di
  `Activity/KomitePost_Reject-Act.xml:3172` dengan kondisi `GroupPanel=="002"`.

## Non-goal

- **Tidak** menangani penolakan medis — itu `TKT-B11-002` (RCL Dokter).
- **Tidak** membangun mekanisme penugasan — itu `B-6`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Ticket rule `SendtoPUCL`** — dirujuk `Flow/Register_Flow.xml:3197`, tidak ada di export | **Tim Pega** (`R-16`) | Satu pemicunya terverifikasi (`GroupPanel=="002"`), tetapi **apakah hanya itu** tidak diketahui |
| **When rule `ElseRCLMSIG`** dan **`IsNotViewClaim`** | **Tim Pega** | Percabangan penolakan dan pembatasan tampilan |
| **Workbasket `RCLPUCL`** — definisi antrean | **Tim Pega** | Menentukan siapa yang boleh mengambil |
| **Siapa yang boleh memicu transisi lateral?** Sistem lama **nol pagar izin** | **Work Owner** | Lompatan ke PUCL melewati alur normal; `D-59` bersatuan menu dan belum mengaturnya |

## Acceptance criteria

- [ ] Penolakan menuntut **alasan dari master**; teks bebas tanpa alasan **ditolak** — diuji.
- [ ] Klaim yang ditolak berpindah ke status yang benar, dan **tercatat di jejak audit** dengan
      pelaku, waktu, dan alasannya.
- [ ] Proses ulang membuat kaitan ke klaim asal — riwayat keduanya **dapat ditelusuri dua arah**.
- [ ] Antrean `RCLPUCL` bersifat **Workbasket**: tugas dapat diambil siapa pun yang berwenang, dan
      **penguncian mencegah dua orang mengerjakannya** (`TKT-B06-003`).
- [ ] Lompatan lateral ke PUCL pada kondisi `GroupPanel=="002"` berperilaku sama dengan Pega —
      diuji.
- [ ] Setiap lompatan lateral **tercatat di jejak audit** dengan pelaku — karena tidak ada pagar
      izin, catatan inilah satu-satunya kontrol.
- [ ] Gerbang 1: hasil penolakan dan proses ulang **sama dengan Pega** pada 20 klaim contoh.
- [ ] Gerbang 2: UAT **PncRCLPUCL**.

## Dependency / Blocked by

`TKT-B06-001`, `TKT-B06-003`, `TKT-S5-002`. **Terhalang Tim Pega dan Work Owner.**

## Constraint keamanan, data, operasional

- **Nol pagar izin pada transisi lateral** di sistem lama. Karena `D-59` juga tidak mengenal
  pemisahan tugas, jejak audit adalah **satu-satunya** kontrol yang tersisa — pencatatannya wajib.
- Alasan penolakan **terlihat nasabah** dalam sengketa; ia tidak boleh memuat catatan internal.

## Migrasi skema / rollout / rollback

Menambah tabel penolakan dan kaitan proses ulang. Backward-compatible.

**Rollback:** penolakan yang sudah tercatat tetap ada.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/rcl/... -run TestPenolakanWajibAlasanMaster
go test ./internal/app/rcl/... -run TestProsesUlangTertautKlaimAsal
go test ./internal/app/rcl/... -run TestLompatanLateralTercatat
go run ./cmd/s8 banding --modul B-11 --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Ticket rule `SendtoPUCL` hilang | `Flow/Register_Flow.xml:3197` · `ADR-0021` |
| Pemicu terverifikasi `GroupPanel=="002"` | `Activity/KomitePost_Reject-Act.xml:3172` dan `:3121` |
| `RCL/PUCL` memakai Workbasket | `D-26` · `ADR-0019` |
| Nol pagar izin pada transisi lateral | `docs/verifikasi-bukti-adr.md` §15 baris `B-11` |
| Istilah RCL dan PUCL | `CONTEXT.md` |

## Comments
