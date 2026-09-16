---
title: "TKT-S3-002 — Template, penerima, dan pengiriman email"
labels: [modul::S-3, tipe::migrasi, status::needs-info, prioritas::sedang, gelombang::5]
milestone: "Gelombang 5 — Nilai dan pihak luar"
epic: "Migrasi Claim PNC"
---

# TKT-S3-002 — Template, penerima, dan pengiriman email

Status: needs-info
Kesiapan: **terhalang artefak dan keputusan keamanan**
Modul: **S-3 Notifikasi & Email** · Gelombang: 5 · Bergantung pada: TKT-S3-001
Requirement: FR-S3    Keputusan: D-40, D-67    ADR: 0016    Risiko: R-16, R-17
Rule Pega yang digantikan: seluruh rule **Correspondence** dan template HTML email — **tidak ada satu pun direktori `Correspondence/` di export**
Peran penguji gerbang 2: **PncAdmin** dan pemilik masing-masing pemberitahuan

## Hasil yang diharapkan (dan nilai bisnisnya)

Email yang benar-benar terkirim: isinya sesuai template, penerimanya dari master, dan kredensial
SMTP-nya tidak berada di dalam kode maupun di dalam repositori.

Nilai bisnisnya: `TKT-S3-001` menyiapkan **kapan** email dilepaskan; tiket ini yang menentukan
**apa bunyinya** dan **bagaimana ia sampai**. Tanpa keduanya, seam Notifier tidak mengirim apa pun.

## Ruang lingkup

- Mesin template email dengan isi yang dapat diubah tanpa rilis.
- Penyambungan SMTP, dengan kredensial diambil dari penyimpanan rahasia — **bukan** dari berkas
  konfigurasi yang masuk repositori.
- **TLS aktif** pada koneksi SMTP.
- Pencatatan pengiriman: peristiwa, penerima, waktu, hasil.

## Non-goal

- **Tidak** merancang ulang bunyi email; isinya menyalin yang berlaku sekarang — sejauh artefaknya
  tersedia.
- **Tidak** memindahkan kredensial produksi. Rotasi dan penyimpanannya milik **Tim Infra/Security**.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Seluruh rule Correspondence dan template HTML** — direktorinya tidak ada di export. Jumlahnya pun belum pasti: `06-MODULE-BREAKDOWN.md:62` menyebut **5**, `verifikasi-bukti-adr.md:2783` menyebut **3** | **Tim Pega** (`R-16`) | **Isi email tidak diketahui sama sekali.** Tiket ini tidak dapat menyatakan satu pun template selesai |
| **Rotasi 3 password SMTP (31 lokasi, plaintext)** — masih aktif? diganti? disimpan di mana? | **Tim Infra/Security** (`D-40`, `R-17`) | Tanpa tempat penyimpanan rahasia, satu-satunya cara menyambung SMTP adalah mengulangi kesalahan yang sama |
| **`UseSSL=false` pada seluruh 14 kemunculan, sementara 16 dari 31 lokasi memakai port 587** — apakah itu yang berlaku di produksi? | **Tim Infra/Security** (`D-40`) | Bila benar, email selama ini terkirim **tanpa enkripsi**; menyalakan TLS adalah perubahan perilaku yang harus disepakati, bukan diam-diam |
| **Pemetaan penerima personal ke mailbox fungsional** | **Work Owner** (`D-67`) | `D-67` melarang akun pribadi; penggantinya belum ditunjuk |

## Acceptance criteria

- [ ] Kredensial SMTP **tidak ada di dalam kode maupun di repositori** — diuji dengan pemindaian
      otomatis atas seluruh berkas yang di-commit: **nol temuan**.
- [ ] Koneksi SMTP memakai **TLS** — diuji; koneksi tanpa TLS ditolak.
- [ ] Isi email dapat diubah **tanpa rilis** — diuji dengan mengubah template lalu mengirim ulang.
- [ ] **Tidak ada satu pun alamat akun pribadi** sebagai penerima di data produksi (`D-67`) —
      diperiksa atas seluruh data penerima.
- [ ] Setiap pengiriman **tercatat** dengan hasilnya — diuji.
- [ ] Isi email **sama dengan yang dikirim Pega**, diperiksa per template setelah artefaknya
      diterima. *Kriteria ini belum dapat dijalankan; jumlah templatenya pun belum pasti.*
- [ ] Gerbang 2: UAT oleh **pemilik masing-masing pemberitahuan**, bukan satu orang untuk semuanya.

## Dependency / Blocked by

`TKT-S3-001`. **Terhalang Tim Pega (`R-16`) dan Tim Infra/Security (`D-40`, `R-17`).**

## Constraint keamanan, data, operasional

- **Export memuat 3 password SMTP di 31 lokasi dalam bentuk plaintext** (`R-17`). Nilai-nilainya
  **tidak boleh disalin ke dokumen mana pun yang di-commit**; lokasinya sudah diserahkan terpisah
  ke Tim Infra/Security.
- `UseSSL=false` pada seluruh kemunculannya berarti **email kemungkinan terkirim tanpa enkripsi
  selama ini**. Menyalakan TLS memperbaiki keadaan, tetapi ia **perubahan perilaku** dan tunduk
  `P-5` — harus dinyatakan, bukan diselipkan.
- Email membawa data nasabah ke luar batas sistem; email berisi **data medis** tunduk `FR-R2`.

## Migrasi skema / rollout / rollback

Menambah tabel template. Tidak menyentuh tabel klaim.

**Rollback:** mengembalikan versi template. **Mematikan TLS bukan rollback yang sah** — ia
mengembalikan pengiriman tanpa enkripsi.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/pindai-rahasia .            # HARUS nol temuan
go test ./internal/adapter/smtp/... -run TestTLSWajib
go test ./internal/app/notifikasi/... -run TestTemplateBerubahTanpaRilis
go run ./cmd/tools/cek-penerima --larang-akun-pribadi
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Tidak ada direktori `Correspondence/` di export | struktur direktori export · `R-16` |
| Jumlah Correspondence berbeda antar dokumen (3 vs 5) | `docs/Steering/06-MODULE-BREAKDOWN.md:62` · `docs/verifikasi-bukti-adr.md:2783` |
| 3 password SMTP di 31 lokasi, plaintext | `D-40` · `R-17` · `docs/Steering/16-RISK-ANALYSIS.md:484` |
| `UseSSL=false` pada 14 kemunculan; 16 dari 31 lokasi port 587 | `docs/Steering/00-DECISION-LOG.md:977` |
| Tidak ada penerima akun pribadi | `D-67` |

## Comments
