---
title: "TKT-B01-003 — Aturan penyegaran snapshot"
labels: [modul::B-1, tipe::aturan-bisnis, status::needs-info, prioritas::sedang, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B01-003 — Aturan penyegaran snapshot

Status: needs-info
Kesiapan: **terhalang keputusan** — aturan penyegaran belum ada sama sekali
Modul: **B-1 View Polis** · Gelombang: 3 · Bergantung pada: TKT-B01-001
Requirement: FR-B1    Keputusan: D-04    ADR: 0006, 0026    Risiko: —
Rule Pega yang digantikan: **tidak ada padanan** — sistem lama tidak punya aturan penyegaran yang tercatat di rule mana pun
Peran penguji gerbang 2: **PncPICTeknik** dan **PncManagerAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Aturan yang menjawab satu pertanyaan: **kapan snapshot polis boleh diperbarui, dan siapa yang
boleh melakukannya.**

Nilai bisnisnya adalah menutup utang yang lahir dari `ADR-0006` itu sendiri. Membekukan polis benar
secara bisnis, tetapi ia **menciptakan kemungkinan snapshot usang**: bila polis dikoreksi setelah
registrasi — pembatalan, endorsemen, perbaikan data tertanggung — klaim tetap memakai salinan lama
**sampai ada yang menyegarkannya secara sadar**. Aturan itu belum ada.

## Ruang lingkup

- Aturan kapan penyegaran **boleh** dilakukan, dan kapan ia **dilarang** (misalnya setelah komite
  menyetujui, atau setelah pembayaran).
- Siapa yang berwenang memicunya, mengikuti model izin menu (`D-59`).
- **Perbandingan sebelum dan sesudah** yang ditampilkan ke pengguna sebelum penyegaran disetujui —
  agar dampaknya terlihat, bukan diterapkan diam-diam.
- Pencatatan jejak audit: siapa menyegarkan, kapan, field apa yang berubah, dari nilai apa.

## Non-goal

- **Tidak** menyegarkan otomatis. Penyegaran otomatis akan mengembalikan persoalan yang justru
  dihindari `ADR-0006`.
- **Tidak** mengubah data polis di GISFW.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Kapan snapshot boleh atau harus disegarkan?** | **Work Owner** | Ini aturan bisnis yang belum pernah ada — tidak dapat disimpulkan dari sistem lama karena sistem lama tidak punya konsepnya |
| **Siapa yang berwenang memicunya?** | **Work Owner** | `D-59` menetapkan izin bersatuan menu; penyegaran mengubah dasar penilaian klaim, sehingga kewenangannya bernilai tinggi |
| **Apa yang terjadi pada nilai yang sudah dihitung** dari snapshot lama — estimasi, spreading, jenjang komite? | **Work Owner** | Menentukan apakah penyegaran memicu perhitungan ulang atau hanya memperbarui tampilan |

## Acceptance criteria

> Bentuk AC yang akan diisi setelah aturannya ditetapkan.

- [ ] Penyegaran hanya dapat dilakukan pada tahap yang diizinkan; di luar itu **ditolak** dengan
      pesan yang menyebut tahapnya.
- [ ] Sebelum disetujui, pengguna melihat **perbandingan field yang akan berubah** — nilai lama di
      samping nilai baru.
- [ ] Penyegaran menghasilkan **tepat satu baris jejak audit per field yang berubah**, berisi nilai
      sebelum dan sesudah.
- [ ] Pengguna tanpa kewenangan menerima `403` dan **tidak melihat tombolnya**.
- [ ] Snapshot yang disegarkan **tidak menghapus versi sebelumnya** — konsisten `ADR-0012`.

## Dependency / Blocked by

`TKT-B01-001` · `TKT-S5-002`. **Terhalang tiga keputusan Work Owner.**

## Constraint keamanan, data, operasional

- Penyegaran **mengubah dasar penilaian klaim yang sedang berjalan**. Ia tindakan bernilai tinggi,
  dan karena `D-59` tidak mengenal pemisahan tugas, **jejak audit adalah satu-satunya kontrolnya**.
- Snapshot lama **tidak dihapus** (`ADR-0012`) — yang lama ditandai tidak aktif, bukan dibuang.

## Migrasi skema / rollout / rollback

Menambah penanda versi pada tabel snapshot (`TKT-B01-002`) — kolom *nullable*, backward-compatible.

**Rollback:** fitur penyegaran disembunyikan dari menu. Snapshot yang sudah disegarkan tetap
berlaku; versi lamanya tetap tersimpan.

## Rencana verifikasi

> **Rencana — belum dijalankan, dan belum dapat disusun lengkap.**

```bash
go test ./internal/app/polis/... -run TestPenyegaranSnapshot
go test ./internal/app/polis/... -run TestPenyegaranTercatatDiAudit
go test ./internal/adapter/http/... -run TestIzinPenyegaran
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Snapshot dapat usang; aturan penyegaran belum ada | `ADR-0006` Negatif/utang teknis dan Pertanyaan terbuka |
| Tidak ada pemisahan tugas; audit satu-satunya kontrol | `D-59` · `ADR-0023` |
| Tidak ada penghapusan fisik | `D-66` · `ADR-0012` |

## Comments
