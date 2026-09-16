---
title: "TKT-B14-001 — Layar Input Receive Document"
labels: [modul::B-14, tipe::migrasi, status::ready-for-human, prioritas::sedang, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B14-001 — Layar Input Receive Document

Status: ready-for-human
Kesiapan: siap
Modul: **B-14 Input Receive Document** · Gelombang: 3 · Bergantung pada: TKT-U2-002, TKT-F3-005
Requirement: FR-B14    Keputusan: D-13    ADR: 0002, 0026    Risiko: —
Rule Pega yang digantikan: `Flow/InputReceiveDocument.xml` · harness `ReceiveDoucument_Harness` dan `ViewReceiveDocument`
Peran penguji gerbang 2: **PncReceive** dan **PncManagerReceive**

## Hasil yang diharapkan (dan nilai bisnisnya)

Petugas penerimaan dapat mencatat satu kiriman dokumen fisik dari cabang, dan mencatat kapan
dokumen itu **benar-benar diterima**.

Nilai bisnisnya melampaui pencatatan administratif: **tanggal terima dokumen** yang diisi di sini
dipakai sebagai batas pada validasi tanggal di Input Register, dan menjadi titik awal perhitungan
TAT yang dilaporkan ke manajemen.

## Ruang lingkup

- Layar pencatatan kiriman: **nomor register dokumen (`RCV_ID`)**, cabang pengirim, ekspedisi,
  nomor resi, tanggal kirim, estimasi tiba, jumlah lembar, dan jenis dokumen.
- Pencatatan **tanggal terima** saat dokumen benar-benar sampai.
- Layar lihat detail kiriman (setara `ViewReceiveDocument`).
- Jejak audit pada pencatatan dan perubahan tanggal terima (`S-5`).

## Non-goal

- **Tidak** mengunggah berkas dokumen — itu `S-1`. Modul ini mencatat **kiriman fisik**, bukan
  berkas digital.
- **Tidak** menautkan ke klaim — itu `TKT-B14-003`.
- **Tidak** membangun inbox dan penugasannya — itu `TKT-B14-002`.

## Acceptance criteria

- [ ] Kiriman baru dapat disimpan dengan kesembilan field, dan **nomor register dokumen terbit
      unik** — diuji 20 penyimpanan berturut-turut, nol nomor ganda.
- [ ] Tanggal terima **tidak boleh lebih awal** dari tanggal kirim — diuji kedua batas.
- [ ] Tanggal terima **tidak boleh di masa depan** — diuji.
- [ ] Estimasi tiba boleh kosong; tanggal terima boleh kosong saat kiriman baru dicatat — diuji.
- [ ] Mengubah tanggal terima menghasilkan **tepat satu baris jejak audit** berisi nilai sebelum
      dan sesudah.
- [ ] Susunan field mengikuti `ReceiveDoucument_Harness` — dibuktikan perbandingan berdampingan
      yang disetujui penguji gerbang 2.
- [ ] Petugas tanpa izin menu Receive Document menerima `403` dan **tidak melihat menunya**.
- [ ] Gerbang 1: hasil penyimpanan **sama dengan Pega** pada 20 kiriman contoh di staging.
- [ ] Gerbang 2: UAT **PncReceive** pada 10 kiriman nyata.

## Dependency / Blocked by

`TKT-U2-002` (form baku) · `TKT-F3-005` (izin menu) · `TKT-S5-002` (jejak audit).

## Constraint keamanan, data, operasional

- Layar memuat **nomor polis dan nama tertanggung** pada kiriman yang sudah tertaut klaim — batas
  data cabang berlaku (`TKT-F3-005`).
- Tanggal terima **memengaruhi validasi klaim dan angka TAT**; mengubahnya setelah klaim berjalan
  adalah tindakan bernilai tinggi dan karena itu diaudit.
- Waktu disimpan UTC dan ditampilkan WIB lewat satu konversi (`F-5`).

## Migrasi skema / rollout / rollback

Menambah tabel penerimaan dokumen milik aplikasi. Backward-compatible; Pega tidak membacanya.

**Rollback:** kiriman yang telanjur dicatat di sistem baru tetap ada. Karena Pega masih menangani
alurnya selama masa paralel, rollback berarti **berhenti mencatat kiriman baru di sistem baru** —
data lama tidak hilang.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/receivedoc/... -run TestSimpanKiriman
go test ./internal/app/receivedoc/... -run TestValidasiTanggalTerima
go run ./cmd/s8 banding --modul B-14 --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Flow dan harness penerimaan dokumen | `Flow/InputReceiveDocument.xml` · `Harness/ReceiveDoucument_Harness-Harness.xml` |
| Isi data: ekspedisi, resi, estimasi tiba | `CONTEXT.md` — **Receive Document** |
| Tanggal terima dipakai validasi registrasi | Invarian `I-2`, `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Tata letak mengikuti Pega | `D-13` |

## Comments
