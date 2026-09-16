---
title: "TKT-B14-002 — Inbox penerimaan dokumen dan penugasannya"
labels: [modul::B-14, tipe::migrasi, status::needs-info, prioritas::sedang, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B14-002 — Inbox penerimaan dokumen dan penugasannya

Status: needs-info
Kesiapan: **terhalang artefak** — Ticket rule dan router
Modul: **B-14 Input Receive Document** · Gelombang: 3 · Bergantung pada: TKT-B14-001, TKT-B06-001
Requirement: FR-B14, FR-W2    Keputusan: D-26    ADR: 0019, 0021    Risiko: R-16
Rule Pega yang digantikan: harness `InboxRCVApp_Harness` · router `PNCAdminRouterRCV` · Ticket rule `SendToReceiveDocument` (`Flow/InputReceiveDocument.xml:690`)
Peran penguji gerbang 2: **PncReceive** dan **PncManagerReceive**

## Hasil yang diharapkan (dan nilai bisnisnya)

Kiriman dokumen yang belum ditangani muncul di inbox petugas penerimaan, dan dapat **dilompati ke
sana dari tahap lain** ketika sebuah klaim ternyata masih menunggu dokumen.

Nilai bisnisnya: tanpa inbox, kiriman yang belum ditindaklanjuti **tidak terlihat siapa pun**
sampai ada yang mencarinya. Dan lompatan lateral itulah yang membuat klaim tidak perlu dibatalkan
hanya karena dokumennya belum lengkap.

## Ruang lingkup

- Inbox kiriman dokumen: daftar kiriman yang belum diterima atau belum tertaut klaim, memakai
  komponen tabel baku (`TKT-U2-001`).
- Penugasan kiriman ke petugas mengikuti model `Worklist`/`Workbasket` (`ADR-0019`).
- **Lompatan lateral ke tahap Receive Document** — pengganti Ticket rule `SendToReceiveDocument`.

## Non-goal

- **Tidak** membangun mekanisme penugasan umum — itu `B-6`; modul ini memakainya.
- **Tidak** menentukan siapa yang boleh memicu lompatan lateral — itu pertanyaan lintas modul di
  `ADR-0021` dan `B-11`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Ticket rule `SendToReceiveDocument`** — dirujuk `Flow/InputReceiveDocument.xml:690` tetapi **tidak ada di export** | **Tim Pega** (`R-16`) | Tanpa rule-nya, **kapan** lompatan ini dipicu tidak diketahui |
| **Router `PNCAdminRouterRCV`** — ada di export, tetapi perlu konfirmasi bahwa ia memakai jalur `counter_quota` seperti router lain | **Work Owner + Tim Pega** | Menentukan ke siapa kiriman ditugaskan |
| **Siapa yang berwenang memicu lompatan lateral?** Sistem lama **nol pagar izin** | **Work Owner** | `D-59` menetapkan izin bersatuan menu; lompatan lateral melewati alur normal, dan kewenangannya belum dirumuskan |

## Acceptance criteria

> Butir 3 dan 4 berubah bentuk tergantung jawaban di atas.

- [ ] Kiriman yang belum diterima muncul di inbox; yang sudah diterima **tidak** — diuji.
- [ ] Inbox menghormati batas data cabang: petugas cabang A tidak melihat kiriman cabang B, **dan
      jumlah totalnya juga tidak memuatnya** — diuji.
- [ ] Penugasan mengikuti model `Worklist`/`Workbasket` sesuai `ADR-0019` — diuji kedua model.
- [ ] Lompatan lateral memindahkan klaim ke tahap Receive Document **dan tercatat di jejak audit**
      dengan pelaku serta alasannya.
- [ ] Gerbang 1: isi inbox **sama dengan Pega** untuk peran yang sama pada data staging.
- [ ] Gerbang 2: UAT **PncReceive**.

## Dependency / Blocked by

`TKT-B14-001` · `TKT-B06-001` (mekanisme penugasan) · `TKT-U2-001` (tabel baku).
**Terhalang Tim Pega dan satu keputusan Work Owner.**

## Constraint keamanan, data, operasional

- Lompatan lateral **melewati urutan alur normal**. Karena `D-59` tidak mengenal pemisahan tugas,
  satu-satunya kontrol yang tersisa adalah **jejak audit** — karena itu pencatatannya wajib, bukan
  opsional.
- Inbox menampilkan data nasabah; batas data ditegakkan di kueri (`TKT-F3-005`).

## Migrasi skema / rollout / rollback

Memakai tabel penugasan `B-6`; tidak menambah tabel sendiri.

**Rollback:** inbox disembunyikan dari menu. Kiriman tetap tercatat dan tetap dapat dicari lewat
layar `TKT-B14-001`.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/receivedoc/... -run TestInboxPenerimaan
go test ./internal/app/receivedoc/... -run TestBatasDataCabang
go test ./internal/app/receivedoc/... -run TestLompatanLateralTercatat
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Ticket rule `SendToReceiveDocument` dirujuk tanpa rule | `Flow/InputReceiveDocument.xml:690` · `ADR-0021` |
| Router `PNCAdminRouterRCV` | `D-26` · `ADR-0019` |
| Worklist versus Workbasket | `D-26` · `CONTEXT.md` |
| Nol pagar izin pada transisi lateral | `docs/verifikasi-bukti-adr.md` §15 baris `B-11` |

## Comments
