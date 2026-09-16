---
title: "TKT-B09-002 — Pengiriman dokumen ke reasuransi dan koasuransi"
labels: [modul::B-9, tipe::integrasi, status::needs-info, prioritas::tinggi, gelombang::5]
milestone: "Gelombang 5 — Nilai dan pihak luar"
epic: "Migrasi Claim PNC"
---

# TKT-B09-002 — Pengiriman dokumen ke reasuransi dan koasuransi

Status: needs-info
Kesiapan: terhalang keputusan dan `D-40`
Modul: **B-9 PLA/Pre-DLA/DLA** · Gelombang: 5 · Bergantung pada: TKT-B09-001, TKT-S3-001
Requirement: FR-B9    Keputusan: D-67, D-40    ADR: 0011, 0025    Risiko: R-17
Rule Pega yang digantikan: `Activity/LetterOfAssignment_Act-Act.xml:10011` (BCC broker) · `Activity/LetterOfAssignment2_Act-Act.xml:7725` (`<To>` **akun Gmail pribadi**) · `Activity/SendDLAAutoSaatGeneratedDLA-Act.xml:2706` (**fallback penerima berdasarkan hostname dev**) · `Activity/CheckerSendEmailApproveReject-Act.xml:2299`
Peran penguji gerbang 2: **PncPLADLA** dan **TreatyIn**

## Hasil yang diharapkan (dan nilai bisnisnya)

Dokumen PLA/DLA sampai ke penanggung yang benar, dengan **penerima dari master** — bukan dari
alamat yang tertanam di dalam rule.

Nilai bisnisnya: hari ini penerima sebagian di-hardcode, salah satunya **akun Gmail pribadi**, dan
ada **fallback yang mengganti penerima berdasarkan nama server**. Ketika orangnya pindah atau
server dipindahkan, dokumen ke reasuransi **berhenti sampai** tanpa ada yang menyadarinya.

## Ruang lingkup

- Pembentukan dokumen PLA/DLA sebagai **PDF** memakai engine sendiri (`ADR-0011`).
- Pengiriman lewat seam Notifier (`S-3`), dengan penerima dari **master Penerima Notifikasi**
  (`TKT-F4-003`).
- **Penghapusan fallback berbasis hostname** — perbedaan lingkungan menjadi konfigurasi, bukan
  deteksi nama server.
- Pencatatan pengiriman: tujuan, waktu, hasil.

## Non-goal

- **Tidak** membangun pengirim email — itu `S-3`.
- **Tidak** memutuskan tujuan penyimpanan kredensial SMTP — itu `D-40`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **PLA/DLA tanpa email reasuradur sah?** | **Work Owner** | Menentukan apakah dokumen boleh terbit tanpa tujuan kirim |
| **Rotasi 3 password SMTP (31 lokasi)** dan tujuan penyimpanannya | **Tim Infra/Security** (`D-40`, `R-17`) | Pengiriman tidak dapat dijalankan di produksi tanpa kredensial yang tersimpan aman |
| Mailbox fungsional pengganti untuk penerima yang kini akun pribadi | **Work Owner** (`D-67`) | Pemetaan alamat lama ke mailbox baru |

## Acceptance criteria

- [ ] Dokumen PLA/DLA terbentuk sebagai PDF dengan isi yang **sama dengan keluaran Pega** pada 10
      contoh — kriteria kesamaan (isi atau visual) mengikuti keputusan `ADR-0011`.
- [ ] Penerima diambil **dari master**; mengubah master mengubah tujuan **tanpa deployment** —
      diuji.
- [ ] **Nol alamat email di kode** — diuji pemindaian; build gagal bila muncul.
- [ ] **Nol perilaku yang bergantung pada nama host** — diuji dengan menjalankan pada dua hostname
      berbeda: penerima **sama**.
- [ ] **Nol akun pribadi** di daftar penerima — diuji terhadap daftar domain yang diizinkan.
- [ ] Kegagalan pengiriman **tercatat dan terlihat**, dan dokumen tetap tersimpan — diuji dengan
      SMTP yang sengaja dimatikan.
- [ ] Pengiriman **tidak berada di dalam transaksi database** (`TKT-F2-003`).
- [ ] Gerbang 2: UAT **PncPLADLA** memeriksa dokumen yang benar-benar diterima penanggung uji.

## Dependency / Blocked by

`TKT-B09-001` · `TKT-S3-001` · `TKT-F4-003` · `TKT-S2-001` (engine PDF). **Terhalang Work Owner dan
Tim Infra/Security.**

## Constraint keamanan, data, operasional

- Dokumen memuat **data nasabah dan nilai klaim**, dan dikirim ke **pihak luar perusahaan**.
  Salah tujuan berarti kebocoran data.
- **3 password SMTP di 31 lokasi** dan `UseSSL=false` di seluruh kemunculannya (`R-17`) — menunggu
  `D-40`.
- Alamat email **tidak pernah ditulis lengkap** di dokumen proyek (`D-69`); di tiket ini pun
  dirujuk dengan `berkas:baris`.

## Migrasi skema / rollout / rollback

Menambah tabel riwayat pengiriman. Backward-compatible.

**Rollback:** dokumen yang telanjur terkirim **tidak dapat ditarik**.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/pladla/... -run TestPenerimaDariMaster
grep -rInE "[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}" internal/app/pladla/   # HARUS 0
go test ./internal/app/pladla/... -run TestTanpaKetergantunganHostname
go test ./internal/app/pladla/... -run TestKegagalanKirimTercatat
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Akun Gmail pribadi sebagai `<To>` | `Activity/LetterOfAssignment2_Act-Act.xml:7725` |
| BCC broker eksternal | `Activity/LetterOfAssignment_Act-Act.xml:10011` |
| Fallback penerima berbasis hostname dev | `Activity/SendDLAAutoSaatGeneratedDLA-Act.xml:2706` |
| Tidak ada akun pribadi sebagai penerima | `D-67` |
| 3 password SMTP di 31 lokasi, `UseSSL=false` | `R-17` · `D-40` |
| PDF dibangun sendiri di Go | `D-11` · `ADR-0011` |

## Comments
