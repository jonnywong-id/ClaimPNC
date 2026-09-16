---
title: "TKT-B02-004 — Validasi kelengkapan dan Notice of Large Losses"
labels: [modul::B-2, tipe::aturan-bisnis, status::ready-for-human, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B02-004 — Validasi kelengkapan dan Notice of Large Losses

Status: ready-for-human
Kesiapan: siap
Modul: **B-2 Input Register** · Gelombang: 3 · Bergantung pada: TKT-B02-001, TKT-F4-005
Requirement: FR-B2    Keputusan: D-15    ADR: 0025    Risiko: —
Rule Pega yang digantikan: aturan kelengkapan di `Activity/InputRegister_act-Act.xml` · gerbang Notice of Large Losses pada `:19110` (ambang **`1000000000`** di-hardcode) · penerima di `:16154` (7 email pimpinan dalam satu string), `:16297`, `:16461`, `:16628`
Peran penguji gerbang 2: **PncAdmin** dan **PncManagerAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Klaim tidak dapat disimpan dengan data yang kurang, dan **kerugian besar otomatis diberitahukan**
ke Underwriting serta jajaran pimpinan.

Nilai bisnisnya: Notice of Large Losses adalah **pemberitahuan wajib**. Di sistem lama, ambang dan
daftar penerimanya tertanam di dalam rule — mengubahnya menuntut deployment, dan **blok
`// TESTING` menimpa daftar penerima produksi** dengan alamat penguji.

## Ruang lingkup

- Aturan kelengkapan: Penyebab Kerugian wajib (kecuali lini Travel) · Nomor SLIK wajib untuk lini
  SPK / Asuransi Kredit · objek tanpa coverage ditolak · nilai estimasi tidak melebihi TSI.
- **Notice of Large Losses**: bila estimasi setelah konversi kurs melebihi ambang, kirim
  pemberitahuan ke penerima dari **master** (`TKT-F4-003`), bukan dari kode.
- Ambang dibaca dari **master ambang uang** (`TKT-F4-005`), bukan konstanta.
- **Nol blok TESTING** — tidak ada jalur yang menimpa penerima produksi.

## Non-goal

- **Tidak** mengirim email sendiri — pengiriman milik `S-3`; tiket ini memicu peristiwanya.
- **Tidak** memvalidasi tanggal maupun duplikasi.

## Acceptance criteria

- [ ] Klaim tanpa Penyebab Kerugian **ditolak**, kecuali lini Travel — diuji kedua lini.
- [ ] Klaim lini SPK tanpa Nomor SLIK **ditolak** — diuji.
- [ ] Objek tanpa coverage **tidak dapat disimpan** — diuji.
- [ ] Nilai estimasi melebihi TSI coverage **ditolak**, dengan pesan yang menyebut TSI-nya.
- [ ] Estimasi **melebihi ambang** memicu **tepat satu** peristiwa Notice of Large Losses; estimasi
      tepat **pada** ambang **tidak** memicu — diuji kedua batas.
- [ ] Penerima diambil dari master; mengubah master **mengubah penerima** tanpa deployment — diuji.
- [ ] Ambang diambil dari master; mengubahnya mengubah titik pemicu — diuji.
- [ ] **Nol alamat email di kode modul ini** — diuji pemindaian; **nol jalur yang menimpa penerima**
      seperti blok `// TESTING`.
- [ ] Konversi estimasi valuta asing memakai **kurs tanggal kejadian** (`ADR-0015`); kurs tidak
      ditemukan **menolak klaim**, bukan memakai nilai bawaan.
- [ ] Gerbang 1: hasil validasi **sama dengan Pega** pada 30 kasus, kecuali selisih kurs yang
      terpetakan ke butir 8 dan 9 `P-5`.
- [ ] Gerbang 2: UAT **PncAdmin** (pengisi) dan **PncManagerAdmin** (penerima notifikasi).

## Dependency / Blocked by

`TKT-B02-001` · `TKT-F4-003` (master penerima) · `TKT-F4-004` (kurs) · `TKT-F4-005` (master ambang).

## Constraint keamanan, data, operasional

- Penerima notifikasi **wajib mailbox fungsional**; **tidak ada akun pribadi** (`D-67`). Sistem
  lama memuat ≥6 akun Gmail pribadi di jalur produksi.
- Blok `// TESTING` pada `Activity/InputRegister_act-Act.xml:16693`, `:16830`, `:16998`, `:17141`
  **menimpa email produksi dengan precondition identik** — tidak boleh punya padanan apa pun.
- Klaim valuta asing **dapat tertolak** bila kurs belum diisi. Ini perbaikan yang diinginkan, dan
  dampak operasionalnya harus disiapkan sebelum rilis (`ADR-0015`).

## Migrasi skema / rollout / rollback

Tidak menambah tabel di luar master `F-4`.

**Rollback:** ambang dan penerima kembali ke nilai sebelumnya lewat master — **tanpa deployment**.
Itu justru salah satu hasil yang dikejar tiket ini.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/registrasi/... -run TestKelengkapan
go test ./internal/domain/registrasi/... -run TestNoticeLargeLossesBatas
grep -rInE "[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}" internal/domain/registrasi/  # HARUS 0
go run ./cmd/s8 banding --modul B-2 --aturan kelengkapan --kasus 30
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Ambang Large Losses `1000000000` di-hardcode | `Activity/InputRegister_act-Act.xml:19110` |
| 7 email pimpinan dalam satu string | `:16154` |
| Blok `// TESTING` menimpa penerima produksi | `:16693`, `:16830`, `:16998`, `:17141` |
| Invarian `I-4`, `I-6`, `I-9`, `I-10` | `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Kurs tanggal kejadian; tolak bila kosong | `D-48` · `ADR-0015` |
| Tidak ada akun pribadi sebagai penerima | `D-67` |

## Comments
