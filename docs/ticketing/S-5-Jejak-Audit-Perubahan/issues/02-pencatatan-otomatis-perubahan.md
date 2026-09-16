---
title: "TKT-S5-002 — Pencatatan otomatis pada perubahan bernilai bisnis"
labels: [modul::S-5, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::2]
milestone: "Gelombang 2 — Kerangka UI"
epic: "Migrasi Claim PNC"
---

# TKT-S5-002 — Pencatatan otomatis pada perubahan bernilai bisnis

Status: ready-for-human
Kesiapan: siap
Modul: S-5 · Gelombang: 2 · Bergantung pada: TKT-S5-001, TKT-F2-003
Requirement: FR-S5    Keputusan: D-28, D-59    ADR: 0023, 0026    Risiko: R-14
Rule Pega yang digantikan: **tidak ada padanan** — pencatatan di sistem lama bersifat sebagian dan dapat dimutasi
Peran penguji gerbang 2: **tidak berlaku** — `D-60`

## Hasil yang diharapkan (dan nilai bisnisnya)

Mekanisme yang membuat pencatatan audit **terjadi karena strukturnya**, bukan karena penulis kode
ingat menuliskannya.

Nilai bisnisnya sederhana dan keras: kontrol yang bergantung pada ingatan **akan terlewat**. Dan
di sistem ini, yang terlewat berarti perubahan nilai uang klaim yang tidak dapat
dipertanggungjawabkan — pada aplikasi yang **tidak punya pemisahan tugas** (`D-59`).

## Ruang lingkup

- Titik pencatatan yang menyatu dengan mekanisme transaksi (`TKT-F2-003`): baris audit ditulis
  **di dalam transaksi yang sama** dengan perubahannya, sehingga keduanya berhasil bersama atau
  gagal bersama.
- Cara menyatakan "entitas ini diaudit" **sekali di satu tempat**, bukan di setiap pemanggil.
- Perhitungan **nilai sebelum dan sesudah** secara otomatis dari perubahan yang terjadi.
- Pemeriksaan otomatis: perubahan pada entitas yang ditandai diaudit **tanpa** baris audit
  menggagalkan uji.

## Non-goal

- **Tidak** menentukan daftar entitas dan peristiwa yang diaudit — itu `TKT-S5-003`. Tiket ini
  menyediakan mekanismenya, dan daftar sementara memakai **daftar minimum** `ADR-0026`.
- **Tidak** mencatat pembacaan data — hanya perubahan.

## Acceptance criteria

- [ ] Perubahan nilai pada entitas yang ditandai diaudit **selalu** menghasilkan tepat satu baris
      audit — diuji pada 5 jenis perubahan berbeda.
- [ ] Baris audit ditulis **di dalam transaksi yang sama**: transaksi yang gagal **tidak
      meninggalkan baris audit** — diuji dengan kegagalan yang sengaja dipicu.
- [ ] Perubahan yang gagal ditulis auditnya **menggagalkan seluruh transaksi** — audit tidak boleh
      "best effort".
- [ ] Nilai sebelum dan sesudah terisi benar untuk perubahan sebagian — diuji dengan mengubah satu
      field dari tiga.
- [ ] Uji otomatis **gagal** bila ada entitas bertanda diaudit yang jalur perubahannya tidak
      mencatat — diuji dengan entitas percobaan.
- [ ] Daftar minimum `ADR-0026` tercakup: nilai estimasi, nilai settlement, akseptasi, keputusan
      komite, penolakan, proses ulang, perubahan status klaim, dan pembayaran.

## Dependency / Blocked by

Bergantung pada `TKT-S5-001` dan `TKT-F2-003`.

**Catatan urutan:** tiket ini boleh selesai sebelum `TKT-S5-003`. Bila Compliance kelak menambah
peristiwa, yang bertambah adalah **daftar**, bukan mekanismenya.

## Constraint keamanan, data, operasional

- Audit **tidak boleh** dapat dimatikan lewat konfigurasi. Bila sebuah lingkungan perlu
  mematikannya, itu keputusan yang menuntut perubahan kode dan review — bukan sakelar.
- Pencatatan menambah satu operasi tulis pada setiap perubahan bernilai bisnis; dampaknya pada
  jalur transaksi **diukur**, bukan diasumsikan kecil.
- Baris audit memuat data nasabah pada kolom nilai — tunduk pembatasan akses yang sama.

## Migrasi skema / rollout / rollback

Tidak menambah skema di luar `TKT-S5-001`.

**Rollback:** menonaktifkan pencatatan **tidak tersedia** — lihat constraint di atas. Rollback yang
sah adalah mengembalikan versi kode sebelumnya secara keseluruhan.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/... -run TestAuditTercatat
go test ./internal/app/... -run TestAuditIkutGagalSaatTransaksiGagal
go test ./internal/app/... -run TestEntitasDiauditTanpaPencatatanGagal
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Daftar minimum perubahan yang wajib diaudit | `D-28` · `ADR-0026` |
| Tidak ada pemisahan tugas; audit satu-satunya kontrol | `D-59` · `ADR-0023` |
| Kriteria penerimaan #9 wajib tanpa pengecualian | `BRD §21.2` |
| Kepemilikan transaksi di lapisan aplikasi | `ADR-0007` · `TKT-F2-003` |

## Comments
