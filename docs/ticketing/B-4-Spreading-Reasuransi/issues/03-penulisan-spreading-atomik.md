---
title: "TKT-B04-003 — Penulisan spreading sebagai satu transaksi"
labels: [modul::B-4, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B04-003 — Penulisan spreading sebagai satu transaksi

Status: needs-info
Kesiapan: terhalang keputusan (konfirmasi penulisan ulang `UPDATEREAS`)
Modul: **B-4 Spreading Reasuransi** · Gelombang: 3 · Bergantung pada: TKT-B04-001, TKT-F2-003
Requirement: FR-B4    Keputusan: D-02, D-68    ADR: 0007    Risiko: R-01
Rule Pega yang digantikan: `Database/UPDATEREAS.prc` — **4 `COMMIT` sendiri**, sehingga penulisan spreading tidak pernah atomik
Peran penguji gerbang 2: **PncPICTeknik**

## Hasil yang diharapkan (dan nilai bisnisnya)

Penulisan spreading satu Coverage **berhasil seluruhnya atau gagal seluruhnya**.

Nilai bisnisnya: `UPDATEREAS` melakukan `COMMIT` **empat kali**. Bila proses berhenti di tengah,
sebagian baris spreading tersimpan dan sebagian tidak — dan totalnya **tidak lagi 100%** tanpa ada
yang menyadarinya, karena validasi total sudah lewat sebelum penulisan dimulai.

## Ruang lingkup

- Penulisan seluruh baris spreading satu Coverage di dalam **satu transaksi** (`TKT-F2-003`).
- Penulisan ulang logika `UPDATEREAS` di Go — **tanpa memanggil procedure** (`ADR-0007`).
- Validasi total 100% dijalankan **di dalam transaksi yang sama**, sebagai pagar terakhir sebelum
  commit.

## Non-goal

- **Tidak** mematikan `UPDATEREAS` di database — itu langkah tersendiri yang menunggu verifikasi
  `ALL_DEPENDENCIES` (`D-68`).
- **Tidak** menulis ulang procedure lain.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum dikonfirmasi | Pemilik | Kenapa menahan |
|---|---|---|
| **`UPDATEREAS` ber-4 `COMMIT` — boleh ditulis ulang?** `D-68` sudah memutuskan prinsipnya (Claim PNC satu-satunya pemanggil), tetapi konfirmasi per objek belum ada | **Work Owner + DBA** | Bila ternyata ada pemanggil lain, mematikannya merusak sistem itu. Verifikasi cukup satu kueri `ALL_DEPENDENCIES` |
| **12 dependensi procedure** yang dipanggil procedure yang sudah diterima | **DBA** (`R-01`) | Logika di dalamnya mungkin ikut dipanggil `UPDATEREAS` |

## Acceptance criteria

- [ ] Penulisan spreading dengan **lima baris** yang gagal di baris keempat **tidak meninggalkan
      satu baris pun** — diuji.
- [ ] Validasi total 100% dijalankan **di dalam transaksi**; total yang tidak sah **membatalkan
      seluruh penulisan** — diuji.
- [ ] **Nol pemanggilan stored procedure** dari modul ini — diuji pemindaian.
- [ ] Lama transaksi tercatat; transaksi melebihi ambang menghasilkan log `warn` (`TKT-F2-003`).
- [ ] Gerbang 1: **perilaku saat gagal berbeda dengan Pega secara sengaja** — sistem lama
      meninggalkan sebagian data, sistem baru tidak. Kasus uji dirancang menyadari ini, atau ia
      melaporkan **selisih palsu** (`docs/Steering/14-TESTING-STRATEGY.md` §6.4).
- [ ] Gerbang 2: UAT **PncPICTeknik** pada klaim dengan spreading banyak baris.

## Dependency / Blocked by

`TKT-B04-001` · `TKT-F2-003`. **Terhalang konfirmasi Work Owner dan DBA.**

## Constraint keamanan, data, operasional

- Transaksi yang lebih panjang menahan kunci baris lebih lama. Dengan 200–300 pengguna (`D-10`)
  risikonya kecil, tetapi **tidak nol** bila spreading ditulis dalam proses massal.
- **Kontrak galat berbasis string `ErrMsg` tidak dibawa** (`ADR-0007`).

## Migrasi skema / rollout / rollback

Tidak menambah tabel di luar `TKT-B04-001`.

**Rollback:** kembali memanggil `UPDATEREAS` **tidak tersedia** — `ADR-0007` menetapkan aplikasi
tidak memanggil stored procedure. Bila penulisan ulang bermasalah, yang dilakukan adalah menahan
rilis modul ini.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/spreading/... -run TestPenulisanAtomik
go test ./internal/app/spreading/... -run TestValidasiDidalamTransaksi
grep -rIn "CALL \|EXEC \|UPDATEREAS" internal/app/spreading/     # HARUS 0 baris
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `UPDATEREAS` melakukan 4 `COMMIT` | `Database/UPDATEREAS.prc` · `D-68` |
| `B-4` dapat dibuat atomik setelah logika naik ke Go | `D-68` · `ADR-0007` |
| Claim PNC satu-satunya pemanggil — **belum diverifikasi katalog** | `D-68` |
| Perilaku saat gagal berbeda secara sengaja | `docs/Steering/14-TESTING-STRATEGY.md` §6.4 |

## Comments
