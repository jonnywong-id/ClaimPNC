---
title: "TKT-B03-001 — Objek Pertanggungan per lini bisnis"
labels: [modul::B-3, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B03-001 — Objek Pertanggungan per lini bisnis

Status: needs-info
Kesiapan: **terhalang artefak** — When rule klasifikasi produk Aneka hilang
Modul: **B-3 Objek & Coverage** · Gelombang: 3 · Bergantung pada: TKT-B02-001, TKT-U2-002
Requirement: FR-B3    Keputusan: D-19    ADR: 0018    Risiko: R-16
Rule Pega yang digantikan: section `ObjectCoverage` (54 pemakaian) · `ObjectItemList` (32) · `LocationList` · When rule klasifikasi produk Aneka
Peran penguji gerbang 2: **PncAdmin** dan **PncPICTeknik** per lini bisnis

## Hasil yang diharapkan (dan nilai bisnisnya)

Petugas dapat mencatat objek yang tertimpa, dengan **field yang sesuai lini bisnisnya** — bukan
satu form panjang berisi semua kemungkinan.

Nilai bisnisnya: objek kendaraan, orang, kredit, dan properti menuntut data yang berbeda. Form
seragam memaksa petugas mengabaikan sebagian besar field, dan itulah jalan masuk data kosong yang
baru ketahuan saat klaim dinilai.

## Ruang lingkup

- Pencatatan objek dengan atribut umum: kode objek, nama, lokasi objek, lokasi survei, okupasi,
  jenis surveyor, kode cabang.
- **Atribut khas per lini**: kendaraan (rangka, mesin, merek, tipe, model) · orang (tanggal lahir,
  NIK, status peserta) · kredit (kolektibilitas, sebab macet, hari tunggakan, suku bunga).
- Penentuan lini bisnis dari Group Panel dan Business Type pada **snapshot polis** (`B-1`), bukan
  dari input pengguna.
- Satu klaim dapat memuat banyak objek — penambahan dan penghapusan objek sebelum klaim disimpan.

## Non-goal

- **Tidak** membangun coverage — itu `TKT-B03-002`.
- **Tidak** menambah lini bisnis baru; hanya memindahkan yang ada.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **When rule klasifikasi produk Aneka**: `IsARBusiness` (24×), `IsAnekaPerYear` (16×), `IsMBBusiness` (16×), `IsMoneyInsurance` (15×), `IsElectronicEquipment` (10×), dan lainnya | **Tim Pega** (`R-16`) | Aturan inilah yang menentukan **bentuk objek** untuk lini Aneka. Tanpa rule-nya, klasifikasinya harus ditebak |
| **Pemetaan Group Panel `003` berkonflik**: `LocationList` versus `t_anekalist` | **Work Owner** | Dua sumber berbeda untuk hal yang sama; salah pilih berarti objek Aneka tersimpan di tempat yang salah |

## Acceptance criteria

- [ ] Bentuk form objek **berubah mengikuti lini bisnis** pada snapshot polis — diuji pada empat
      lini: kendaraan, PA, kredit, properti.
- [ ] Atribut khas lini **tidak muncul** pada lini yang tidak memakainya — diuji.
- [ ] Satu klaim dapat memuat **lebih dari satu objek**, dan menghapus objek kedua tidak merusak
      yang pertama — diuji.
- [ ] Lini bisnis diambil dari snapshot polis, **bukan dari isian pengguna** — diuji dengan
      mengubah isian dan memastikan bentuk form tidak ikut berubah.
- [ ] **Objek tanpa coverage tidak dapat disimpan** (invarian `I-6`) — diuji.
- [ ] Gerbang 1: objek yang tersimpan **sama dengan Pega** pada 20 klaim contoh dari empat lini.
- [ ] Gerbang 2: UAT oleh **PncPICTeknik masing-masing lini**, bukan satu orang untuk semua.

## Dependency / Blocked by

`TKT-B02-001` · `TKT-B01-001` (snapshot menentukan lini) · `TKT-U2-002`.
**Terhalang Tim Pega dan satu keputusan Work Owner.**

## Constraint keamanan, data, operasional

- Objek lini PA memuat **NIK dan tanggal lahir** — data pribadi; batas data medis `FR-R2` berlaku
  pada coverage-nya, bukan pada objeknya.
- Jumlah objek per klaim tidak dibatasi di sistem lama; **batas wajar** perlu ditetapkan agar satu
  klaim tidak menjadi ribuan baris tanpa disadari.

## Migrasi skema / rollout / rollback

Menambah tabel objek dan tabel atribut per lini. Backward-compatible; Pega tidak membacanya.

**Rollback:** objek yang sudah tersimpan tetap ada; pendaftaran klaim baru dihentikan di sistem
baru (sama dengan `TKT-B02-001`).

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/objek/... -run TestBentukObjekPerLini
go test ./internal/domain/objek/... -run TestObjekTanpaCoverageDitolak
go run ./cmd/s8 banding --modul B-3 --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Section `ObjectCoverage` 54×, `ObjectItemList` 32× | `docs/verifikasi-bukti-adr.md` §15 baris `U-4` |
| Atribut objek per lini | `docs/Steering/05-DOMAIN-MODEL.md` §1 |
| When rule Aneka hilang | `docs/verifikasi-bukti-adr.md` §15 baris `B-3` · `R-16` |
| Invarian `I-6` objek tanpa coverage | `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Istilah **Objek Pertanggungan** menggantikan `Object`/`ObjectList` | `CONTEXT.md` |

## Comments
