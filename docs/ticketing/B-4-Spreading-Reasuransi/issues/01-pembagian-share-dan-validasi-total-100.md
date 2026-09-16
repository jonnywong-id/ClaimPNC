---
title: "TKT-B04-001 — Pembagian share dan validasi total 100%"
labels: [modul::B-4, tipe::aturan-bisnis, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B04-001 — Pembagian share dan validasi total 100%

Status: needs-info
Kesiapan: terhalang keputusan (jumlah desimal share)
Modul: **B-4 Spreading Reasuransi** · Gelombang: 3 · Bergantung pada: TKT-B03-002, TKT-U2-004
Requirement: FR-B4    Keputusan: D-49 butir 1, D-51    ADR: 0016, 0017    Risiko: R-19
Rule Pega yang digantikan: `Activity/InputRegister_act-Act.xml:13183` — `@contains(local.totalspreading, 100.0) || local.totalspreading == 100 || @contains(local.totalspreading, 99.99)`
Peran penguji gerbang 2: **PncPICTeknik** dan **TreatyIn**

## Hasil yang diharapkan (dan nilai bisnisnya)

Pembagian risiko satu Coverage kepada para penanggung, dengan validasi total **100% yang benar
secara angka**.

Nilai bisnisnya konkret: hari ini total **`199.99`** dan **`1100.0`** lolos validasi, karena yang
diperiksa adalah **apakah teksnya memuat potongan "100.0" atau "99.99"** — bukan berapa jumlahnya.
Spreading yang totalnya 199,99% berarti risiko dibagikan hampir dua kali lipat, dan itu memengaruhi
berapa yang ditagihkan ke reasuransi.

## Ruang lingkup

- Layar pembagian share per Coverage: jenis treaty, nama treaty, persentase share, premi terbagi,
  dan penanggung.
- **Validasi total**: `ROUND(SUM(share), 4) BETWEEN 99.9999 AND 100.0001` (`ADR-0016`).
- **Baris bertanda hapus dibuang sebelum perhitungan** — soft delete (`ADR-0012`) membuat baris itu
  tetap ada, sehingga penyaringannya wajib eksplisit di perhitungan ini.
- Share disimpan **presisi penuh**; pembulatan hanya saat ditampilkan (`TKT-U2-004`).

## Non-goal

- **Tidak** menangani Fac Offer dan Ex-Gratia — itu `TKT-B04-002`.
- **Tidak** menghitung nilai settlement — itu `B-5`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Berapa desimal share yang sah?** Toleransi validasi sudah 4 desimal, tetapi berapa desimal yang boleh **diketik** pengguna belum ditetapkan | **Work Owner** | Menentukan validasi masukan dan tampilan; salah pilih membuat total sah tertolak karena pembulatan masukan |

## Acceptance criteria

- [ ] Total **`100.0000`** diterima · **`99.9999`** diterima · **`100.0001`** diterima — ketiganya
      diuji.
- [ ] Total **`199.99`** ditolak · **`1100.0`** ditolak — keduanya diuji. **Inilah bukti butir 1
      `P-5` tertutup.**
- [ ] Total `99.9998` dan `100.0002` ditolak — batas luar toleransi.
- [ ] Baris spreading bertanda hapus **tidak ikut dihitung** — diuji dengan satu baris dihapus dari
      tiga baris yang totalnya 100%.
- [ ] Share disimpan presisi penuh: nilai yang diketik **sama persis** dengan yang tersimpan —
      diuji dengan 4 desimal.
- [ ] Pesan galat menyebut **total yang dihitung** dan batas yang dilanggar, bukan sekadar
      "spreading tidak valid".
- [ ] Gerbang 1: selisih terhadap Pega **hanya** pada klaim yang totalnya di luar toleransi —
      terpetakan ke butir 1 `P-5` (`D-54`).
- [ ] Gerbang 2: UAT **PncPICTeknik** dan **TreatyIn**.

## Dependency / Blocked by

`TKT-B03-002` · `TKT-U2-004` (pemformatan) · `TKT-F2-005` (soft delete).

## Constraint keamanan, data, operasional

- **Data historis tetap mengandung akibat cacat ini.** Memperbaiki aturannya tidak memperbaiki
  klaim yang telanjur tersimpan dengan total di luar toleransi — perlakuan atas data itu adalah
  keputusan tersendiri (`ADR-0017`).
- Angka pecahan di JavaScript **tidak presisi**; share dikirim dan diterima sebagai string desimal
  (`TKT-U2-004`).

## Migrasi skema / rollout / rollback

Menambah tabel spreading per coverage. Backward-compatible.

**Rollback:** mengembalikan validasi lama berarti **memulihkan cacat butir 1** — hanya masuk akal
bila ada temuan yang lebih buruk.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/spreading/... -run TestTotalSeratusPersen
go test ./internal/domain/spreading/... -run TestTolakSubstringPalsu   # 199.99 dan 1100.0
go test ./internal/domain/spreading/... -run TestBarisTerhapusTidakDihitung
go run ./cmd/s8 banding --modul B-4 --aturan total-share
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Toleransi berupa pencocokan substring | `Activity/InputRegister_act-Act.xml:13183` |
| Toleransi baru 4 desimal | `D-51` · `ADR-0016` |
| Butir 1 dari 13 perbaikan eksplisit `P-5` | `D-49` · `ADR-0017` |
| Invarian `I-1` | `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Baris bertanda hapus dibuang sebelum perhitungan | `BRD §11.3` |

## Comments
