---
title: "TKT-F3-001 — Seam Identity dan adapter fake untuk pengembangan"
labels: [modul::F-3, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F3-001 — Seam Identity dan adapter fake untuk pengembangan

Status: ready-for-human
Kesiapan: siap
Modul: F-3 · Gelombang: 1 · Bergantung pada: TKT-F1-001
Requirement: FR-F3    Keputusan: D-07    ADR: 0024    Risiko: R-14
Rule Pega yang digantikan: **tidak ada padanan** — HCC/HCQ muncul **2× di seluruh export**, keduanya teks pesan galat yang menyuruh menghubungi helpdesk
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`); gerbang 1 diganti uji fungsional terhadap kontrak (`D-56`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Antarmuka autentikasi yang **implementasinya dapat diganti**, ditambah satu implementasi tiruan —
sehingga `F-3` dan seluruh modul yang bergantung padanya dapat dikerjakan **tanpa menunggu kontrak
HCC/HCQ**.

Nilai bisnisnya adalah membuka jalan buntu. `F-3` memblokir login seluruh aplikasi, dan kontraknya
dipegang pihak luar. Tanpa seam ini, seluruh papan tiket berhenti menunggu satu dokumen.

## Ruang lingkup

- Antarmuka `Identity` dideklarasikan di lapisan Domain: menerima kredensial, mengembalikan profil
  pengguna (NIK, nama, cabang, jabatan, email) atau galat.
- **Adapter fake** berbasis daftar pengguna lokal untuk pengembangan dan pengujian, yang
  **menolak berjalan di lingkungan produksi**.
- Jenis galat yang dibedakan: kredensial salah, pengguna tidak aktif, dan **sistem identitas tidak
  dapat dihubungi** — ketiganya menuntut perlakuan berbeda.

## Non-goal

- **Tidak** membangun adapter HCC/HCQ — itu `TKT-F3-002`, terhalang kontrak.
- **Tidak** menerbitkan sesi — itu `TKT-F3-003`.
- **Tidak** menangani otorisasi. Otorisasi **tidak** berada di seam ini; ia dimiliki aplikasi dan
  hidup di Domain (`ADR-0023`).

## Acceptance criteria

- [ ] Antarmuka `Identity` berada di lapisan **Domain**; adapter di lapisan Adapter — pemeriksaan
      lapisan `TKT-F1-001` **gagal** bila tertukar.
- [ ] Adapter fake **menolak start** bila lingkungan bertanda produksi — diuji, dan aplikasi gagal
      keras dengan pesan yang jelas.
- [ ] Ketiga jenis galat dapat dibedakan pemanggil tanpa memeriksa teks pesan — diuji.
- [ ] Seluruh modul yang membutuhkan identitas dapat diuji dengan adapter fake **tanpa jaringan**
      — dibuktikan dengan menjalankan uji dalam keadaan jaringan dimatikan.
- [ ] Profil pengguna yang dikembalikan memuat kelima field, dan **field yang kosong ditolak**
      sebagai galat, bukan diteruskan diam-diam.

## Dependency / Blocked by

Bergantung pada `TKT-F1-001`.

**Yang bergantung padanya:** `TKT-F3-002`, `TKT-F3-003`, dan seluruh modul yang memerlukan
identitas pengguna.

## Constraint keamanan, data, operasional

- Adapter fake **tidak boleh** dapat diaktifkan di produksi lewat konfigurasi saja — penolakannya
  ada di kode, bukan hanya di nilai konfigurasi.
- Kredensial **tidak pernah** masuk log (`TKT-F1-003`), termasuk pada jalur galat.
- Adapter fake memakai daftar pengguna contoh yang **bukan** nama pegawai nyata.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** mengembalikan versi seam; adapter fake tidak pernah berjalan
di produksi sehingga tidak ada dampak data.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/identitas/...
go test ./internal/adapter/identitas/... -run TestFakeMenolakProduksi
APP_ENV=production go run ./cmd/app     # HARUS gagal bila adapter fake aktif
go test ./... -run TestTanpaJaringan
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Autentikasi didelegasikan ke HCC/HCQ; otorisasi dimiliki aplikasi | `D-07` · `ADR-0024` |
| HCC/HCQ muncul 2× di export, keduanya teks galat | `T-2` · `docs/verifikasi-bukti-adr.md` §1.1 |
| Seam Identity dengan dua adapter | `docs/Steering/04-FUTURE-ARCHITECTURE.md` §3.5 |
| `F-3` tidak punya baseline; gerbang 1 diganti uji kontrak | `D-56` · `ADR-0028` |

## Comments
