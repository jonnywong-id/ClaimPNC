---
title: "TKT-B13-001 — Layar permintaan buka proteksi"
labels: [modul::B-13, tipe::migrasi, status::ready-for-human, prioritas::sedang, gelombang::4]
milestone: "Gelombang 4 — Persetujuan"
epic: "Migrasi Claim PNC"
---

# TKT-B13-001 — Layar permintaan buka proteksi

Status: ready-for-human
Kesiapan: siap
Modul: **B-13 Input Open Protection** · Gelombang: 4 · Bergantung pada: TKT-U2-002, TKT-F3-005
Requirement: FR-B13    Keputusan: D-13    ADR: 0002, 0026    Risiko: —
Rule Pega yang digantikan: `Flow/CreateProtection_Flow.xml` · harness `InputProtection_Harness`, `InputReqProtection_Harness` · When rule `IsReqProtection`, `IsOpenProtectionPNC` (**keduanya ada di export**)
Peran penguji gerbang 2: **PncAdmin** dan **PncOPCGeneral**

## Hasil yang diharapkan (dan nilai bisnisnya)

Petugas dapat mengajukan **pembukaan proteksi** dan mencatat persetujuannya, sebelum atau di luar
alur klaim normal.

Nilai bisnisnya: Open Protection dipakai ketika perlindungan perlu dibuka mendahului proses klaim
biasa. Tanpa layar ini, permintaan itu ditangani di luar sistem dan **tidak terlacak**.

## Ruang lingkup

- Layar permintaan: nomor proteksi, nomor polis, jenis proteksi, tanggal input, pemohon.
- Status akseptasi permintaan, dengan dua jalur yang sudah ada Ticket rule-nya:
  `TC_PNCInputProtection` dan `Akp_PNCInputProtection`.
- Daftar proteksi yang **belum terpakai**, memakai komponen tabel baku.
- Jejak audit pada pengajuan dan perubahan status akseptasi.

## Non-goal

- **Tidak** menautkan ke klaim — itu `TKT-B13-002`.
- **Tidak** mengubah data polis (`ADR-0006`).

## Acceptance criteria

- [ ] Permintaan proteksi dapat disimpan dengan kelima field, dan **nomor proteksi unik** — diuji
      20 penyimpanan, nol nomor ganda.
- [ ] Status akseptasi dapat diubah, dan setiap perubahan menghasilkan **tepat satu baris jejak
      audit** berisi nilai sebelum dan sesudah.
- [ ] Daftar menampilkan **hanya proteksi yang belum terpakai** secara baku; yang sudah terpakai
      dapat ditampilkan dengan penyaring eksplisit — diuji keduanya.
- [ ] Batas data cabang dan lini bisnis ditegakkan di kueri (`TKT-F3-005`) — diuji.
- [ ] Susunan layar mengikuti `InputProtection_Harness` — perbandingan berdampingan disetujui
      penguji gerbang 2.
- [ ] Gerbang 1: hasil penyimpanan **sama dengan Pega** pada 20 permintaan contoh.
- [ ] Gerbang 2: UAT **PncAdmin** dan **PncOPCGeneral**.

## Dependency / Blocked by

`TKT-U2-002` · `TKT-F3-005` · `TKT-S5-002`.

## Constraint keamanan, data, operasional

- Layar memuat nomor polis dan data tertanggung — batas data berlaku.
- Proteksi yang sudah terpakai **tidak dihapus**, hanya ditandai (`ADR-0012`) — sehingga riwayat
  pemakaiannya tetap dapat ditelusuri.

## Migrasi skema / rollout / rollback

Menambah tabel Open Protection. Backward-compatible.

**Rollback:** permintaan yang tersimpan tetap ada; pengajuan baru kembali dilakukan di Pega.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/proteksi/... -run TestSimpanPermintaan
go test ./internal/app/proteksi/... -run TestDaftarHanyaBelumTerpakai
go run ./cmd/s8 banding --modul B-13 --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Flow dan harness Open Protection | `Flow/CreateProtection_Flow.xml` · `Harness/InputProtection_Harness-Harness.xml` |
| `IsReqProtection` dan `IsOpenProtectionPNC` ada di export | `docs/verifikasi-bukti-adr.md` §15 baris `B-13` |
| Ticket rule `TC_PNCInputProtection`, `Akp_PNCInputProtection` ada | `ADR-0021` |
| Aggregate Open Protection | `docs/Steering/05-DOMAIN-MODEL.md` §4 |

## Comments
