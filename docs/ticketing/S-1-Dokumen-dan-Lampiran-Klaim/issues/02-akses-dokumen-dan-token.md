---
title: "TKT-S1-002 — Akses dokumen, token, dan masa berlakunya"
labels: [modul::S-1, tipe::keamanan, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-S1-002 — Akses dokumen, token, dan masa berlakunya

Status: needs-info
Kesiapan: **terhalang keputusan Keamanan Informasi**
Modul: **S-1 Dokumen & Lampiran** · Gelombang: 3 · Bergantung pada: TKT-S1-001, TKT-F3-005
Requirement: FR-S1, FR-R2    Keputusan: D-16, D-64    ADR: 0010, 0023, 0029    Risiko: —
Rule Pega yang digantikan: pembentukan token akses dokumen — **token MD5 tanpa secret dan tanpa TTL** · `GCP_IMAGE` · harness `PNCArchiveDokumen`
Peran penguji gerbang 2: **PncAdmin**, **PncAnalystDoctor** (untuk dokumen medis)

## Hasil yang diharapkan (dan nilai bisnisnya)

Dokumen klaim hanya dapat dibuka orang yang berwenang, dan tautannya **tidak berlaku selamanya**.

Nilai bisnisnya: dokumen klaim memuat data nasabah dan **data medis**. Token yang dibentuk **MD5
tanpa secret dan tanpa masa berlaku** berarti siapa pun yang pernah memegang tautannya — atau yang
dapat menebak polanya — dapat membukanya **kapan saja, selamanya**.

## Ruang lingkup

- Penegakan izin pada pengambilan dokumen: peran pemanggil diperiksa **di server** (`ADR-0023`).
- Penegakan `FR-R2`: dokumen medis hanya untuk **Analyst Doctor** dan **RCL Dokter**.
- Token akses dokumen dengan **masa berlaku**, dan dibentuk dengan cara yang tidak dapat ditebak.
- Perilaku saat token kedaluwarsa: pesan yang jelas, bukan berkas tidak ditemukan.

## Non-goal

- **Tidak** merancang skema penyimpanan — itu `TKT-S1-001`.
- **Tidak** membersihkan `GCP_IMAGE` — kepemilikannya belum ditetapkan.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Token MD5 tanpa secret dan tanpa TTL — diterima Keamanan Informasi?** | **Keamanan Informasi** | Bila tidak, mekanismenya diganti seluruhnya — dan itu mengubah bentuk tiket ini |
| **TTL token dan URL berapa?** | **Work Owner + Keamanan Informasi** | Terlalu pendek mengganggu kerja; terlalu panjang memperpanjang paparan |
| **Siapa membersihkan `GCP_IMAGE`?** | **Work Owner** | Berkas yang tidak pernah dibersihkan menumpuk; tanpa pemilik, tidak ada yang mengerjakannya |

## Acceptance criteria

- [ ] Pengambilan dokumen **memeriksa izin di server** — permintaan tanpa kewenangan mengembalikan
      `403`, diuji lewat URL langsung.
- [ ] Dokumen **medis** hanya dapat diambil peran Analyst Doctor dan RCL Dokter — diuji dengan
      peran lain: `403`. Berlaku **juga di staging** (`ADR-0029`).
- [ ] Token **kedaluwarsa** setelah masa berlaku yang dikonfigurasi — diuji dengan TTL pendek.
- [ ] Token **tidak dapat ditebak**: dua dokumen berbeda menghasilkan token yang tidak berpola —
      diuji.
- [ ] Token kedaluwarsa menghasilkan pesan **"tautan sudah tidak berlaku"**, berbeda dari
      "dokumen tidak ditemukan" — keduanya diuji.
- [ ] Setiap pengambilan dokumen medis **tercatat** — siapa membuka, dokumen apa, kapan.
- [ ] Gerbang 2: UAT **PncAdmin** dan **PncAnalystDoctor** secara terpisah — justru pemisahan
      aksesnya yang diuji.

## Dependency / Blocked by

`TKT-S1-001` · `TKT-F3-005`. **Terhalang Keamanan Informasi dan Work Owner.**

## Constraint keamanan, data, operasional

- **Token MD5 tanpa secret dapat ditebak.** Bila pola pembentukannya diketahui, seluruh dokumen
  klaim dapat diakses tanpa login — ini kelas kerentanan yang tidak boleh dibawa apa adanya.
- Dokumen memuat data nasabah dan data medis; pembatasan `FR-R2` berlaku pada **berkasnya**, bukan
  hanya pada field di layar.
- Staging memuat **data produksi apa adanya** (`D-64`) — pembatasan ini berlaku di sana juga.

## Migrasi skema / rollout / rollback

Menambah kolom token dan masa berlaku pada metadata dokumen. Backward-compatible.

**Rollback:** memperpanjang masa berlaku token **bukan rollback yang sah** — ia memperluas paparan.
Rollback yang benar adalah mengembalikan versi kode, bukan melonggarkan aturannya.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/http/... -run TestAksesDokumenMemeriksaIzin
go test ./internal/adapter/http/... -run TestDokumenMedisDibatasi
go test ./internal/app/dokumen/... -run TestTokenKedaluwarsa
go run ./cmd/tools/uji-token-tebakan --sample 1000   # tidak boleh berpola
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Token MD5 tanpa secret dan tanpa TTL | `docs/verifikasi-bukti-adr.md` §15 baris `S-1` |
| Akses data medis dibatasi dua peran | `FR-R2` · `BRD §17.2` |
| Otorisasi diperiksa di setiap endpoint | `D-59` · `ADR-0023` |
| Staging memuat data produksi apa adanya | `D-64` · `ADR-0029` |
| Masa berlaku URL sebagai metadata | `D-16` · `ADR-0010` |

## Comments
