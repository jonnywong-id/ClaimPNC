---
title: "TKT-B04-002 — Fac Out, Fac Offer, dan Ex-Gratia"
labels: [modul::B-4, tipe::aturan-bisnis, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B04-002 — Fac Out, Fac Offer, dan Ex-Gratia

Status: needs-info
Kesiapan: terhalang keputusan
Modul: **B-4 Spreading Reasuransi** · Gelombang: 3 · Bergantung pada: TKT-B04-001
Requirement: FR-B4    Keputusan: D-19    ADR: 0016    Risiko: —
Rule Pega yang digantikan: aturan Fac Out dan Ex-Gratia di `Activity/InputRegister_act-Act.xml`; daftar Fac Offer pada snapshot polis (`B-1`)
Peran penguji gerbang 2: **PncPICTeknik** dan **TreatyIn**

## Hasil yang diharapkan (dan nilai bisnisnya)

Tiga aturan reasuransi yang saling berkait ditegakkan konsisten: kelengkapan Fac Offer, kelengkapan
Object Name untuk Group Panel `003`, dan perubahan otomatis treaty saat klaim ditandai Ex-Gratia.

Nilai bisnisnya: ketiganya menentukan **siapa yang menanggung dan dengan perjanjian apa**. Fac Out
tanpa Fac Offer berarti klaim dibagikan ke penanggung yang penawarannya belum tercatat.

## Ruang lingkup

- Validasi: bila ada spreading **Fac Out**, data **Fac Offer wajib ada** pada snapshot polis.
- Validasi khusus **Group Panel `003`**: Fac Offer wajib menyertakan **Object Name**.
- Perubahan otomatis: klaim bertanda **Ex-Gratia** mengubah jenis treaty **`OR` menjadi `ORS`**.
- Pencatatan jejak audit pada perubahan otomatis tersebut — karena ia mengubah data tanpa tindakan
  pengguna.

## Non-goal

- **Tidak** memvalidasi total share — itu `TKT-B04-001`.
- **Tidak** menerbitkan PLA/DLA ke penanggung — itu `B-9`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Group Panel `003`: cek semua baris Fac Offer atau hanya baris pertama?** Source hanya memeriksa satu baris | **Work Owner** | Bila seharusnya semua baris, ini celah yang sudah berjalan; bila memang satu baris, sistem baru harus menirunya |
| **Pemetaan kode `10001` / `10007` / `10015`** | **Work Owner** | Ketiganya muncul sebagai kode tanpa keterangan; artinya tidak dapat disimpulkan dari source |

## Acceptance criteria

- [ ] Spreading bertipe **Fac Out** tanpa Fac Offer pada snapshot **ditolak**, dengan pesan yang
      menyebutnya — diuji.
- [ ] Untuk **Group Panel `003`**, Fac Offer tanpa Object Name **ditolak** — diuji.
- [ ] Klaim ditandai **Ex-Gratia** mengubah treaty `OR` menjadi `ORS` **otomatis**, dan perubahan
      itu **tercatat di jejak audit** — diuji keduanya.
- [ ] Menghapus tanda Ex-Gratia **tidak** mengembalikan `ORS` menjadi `OR` secara diam-diam —
      perilakunya ditetapkan eksplisit dan diuji.
- [ ] Kode `10001`, `10007`, `10015` diperlakukan sesuai keputusan Work Owner — diuji ketiganya.
- [ ] Gerbang 1: hasil validasi **sama dengan Pega** pada 20 klaim contoh yang memuat Fac Out.
- [ ] Gerbang 2: UAT **TreatyIn**.

## Dependency / Blocked by

`TKT-B04-001` · `TKT-B01-001` (Fac Offer ada di snapshot polis). **Terhalang dua keputusan.**

## Constraint keamanan, data, operasional

- Perubahan otomatis `OR` → `ORS` mengubah data **tanpa tindakan pengguna**. Karena `D-59` tidak
  mengenal pemisahan tugas, pencatatan auditnya **wajib** — itulah satu-satunya jejak bahwa
  perubahan itu terjadi.
- Fac Offer berada di **snapshot polis**, bukan data polis hidup (`ADR-0006`) — bila Fac Offer
  ditambahkan di GISFW setelah registrasi, ia **tidak muncul** sampai snapshot disegarkan
  (`TKT-B01-003`).

## Migrasi skema / rollout / rollback

Tidak menambah tabel di luar `TKT-B04-001`.

**Rollback:** mengembalikan aturan; data yang telanjur berubah menjadi `ORS` **tidak dikembalikan
otomatis** — itu keputusan tersendiri.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/spreading/... -run TestFacOutWajibFacOffer
go test ./internal/domain/spreading/... -run TestGroupPanel003ObjectName
go test ./internal/domain/spreading/... -run TestExGratiaMengubahTreaty
go run ./cmd/s8 banding --modul B-4 --aturan fac-offer --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Fac Out wajib Fac Offer; Group Panel `003` wajib Object Name; Ex-Gratia `OR`→`ORS` | `BRD §11.3` |
| Fac Offer berada di snapshot polis | `docs/Steering/05-DOMAIN-MODEL.md` §1 |
| Istilah Fac Out, XOL, Ex-Gratia | `CONTEXT.md` |
| Pertanyaan terbuka Group Panel `003` dan kode `10001`/`10007`/`10015` | `docs/verifikasi-bukti-adr.md` §15 baris `B-4` |

## Comments
