---
title: "TKT-F6-001 — Daftar portal sebagai data, bukan konstanta"
labels: [modul::F-6, tipe::fondasi, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F6-001 — Daftar portal sebagai data, bukan konstanta

Status: needs-info
Kesiapan: **terhalang artefak** (`IsServerSyariah`) dan keputusan mata uang
Modul: **F-6 Portal & Multi-Sumber Data** · Gelombang: 1 · Bergantung pada: TKT-F1-002, TKT-F4-001
Requirement: FR-F6    Keputusan: D-75, D-77    ADR: 0030    Risiko: R-20
Rule Pega yang digantikan: **48 perbandingan** terhadap `pxRequestor.pxReqServer` pada 3 hostname · When rule `IsServerSyariah` dan `IsDevelopmentServer` (**keduanya hilang dari export**)
Peran penguji gerbang 2: **PncAdmin** dan **PncManagerAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Daftar portal tersimpan sebagai **data**, sehingga menambah entitas tidak menuntut perubahan kode.

Nilai bisnisnya terlihat dari cara lama: perilaku entitas ditentukan dengan **membandingkan nama
server di 48 tempat**. Menambah satu entitas berarti menambah 48 percabangan — dan melewatkan satu
saja menghasilkan entitas yang berperilaku separuh benar, tanpa ada yang tahu.

Ada bukti langsung bahwa cara lama itu memang gagal: **`IsServerSyariah` hilang dari export dan
tidak tercatat di `19-GAP`**, sehingga dokumen kami mencatat 3 hostname padahal entitasnya minimal
4. Cara lama **menyembunyikan satu entitas penuh dari inventaris kami sendiri**.

## Ruang lingkup

- Tabel portal: kode, nama tampilan, rujukan konfigurasi database, mata uang, status aktif.
- Pembacaan daftar portal saat start, dan penyajiannya sebagai pilihan bagi pengguna yang berhak.
- **Tidak ada nama entitas yang tertanam di kode** — termasuk nama keempat portal yang sudah diketahui.
- Perilaku bila sebuah portal dinyatakan tidak aktif.

## Non-goal

- **Tidak** membuat koneksinya — itu `TKT-F6-002`.
- **Tidak** menangani perpindahan portal dan kewenangannya — itu `TKT-F6-003`.
- **Tidak** memutuskan aturan bisnis yang berbeda per entitas.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| ~~Berapa portal sebenarnya?~~ — **TERTUTUP `D-77`**: **4 sekarang, dan bisa bertambah** | — | Menguatkan rancangan: daftar portal adalah **data**, dan penambahan portal **tidak boleh** menuntut perubahan kode |
| **`IsServerSyariah` hilang** — kapan perilaku syariah berlaku? | **Tim Pega** (`R-16`) | Tanpa rule itu, portal Syariah dapat dibuat tetapi **tidak dapat dibuktikan setara** |
| **Mata uang per portal** — Timor-Leste bukan Rupiah | **Work Owner** | Menentukan kolom mata uang di tabel portal, dan berpengaruh ke kurs serta pembulatan |

## Acceptance criteria

- [ ] Daftar portal berasal dari **data**; menambah portal **tidak menuntut perubahan kode** —
      diuji dengan menambah satu portal lewat data saja.
- [ ] **Nol nama entitas tertanam di kode** — dicari otomatis atas keempat nama yang diketahui:
      **nol temuan**.
- [ ] Portal yang **tidak aktif** tidak muncul sebagai pilihan dan **tidak dapat dipanggil
      langsung** — diuji: `403`, bukan sekadar tersembunyi.
- [ ] Aplikasi **menolak start** bila sebuah portal aktif tidak punya konfigurasi database yang
      sah — gagal keras, bukan diam (`TKT-F1-002`).
- [ ] Mata uang portal terbaca dari data, dan dipakai pemformat terpusat (`TKT-U2-004`) — diuji
      dengan dua portal bermata uang berbeda.
- [ ] Gerbang 2: UAT **PncAdmin**.

## Dependency / Blocked by

`TKT-F1-002` (konfigurasi) · `TKT-F4-001` (kerangka master). **Terhalang Work Owner dan Tim Pega.**

## Constraint keamanan, data, operasional

- Konfigurasi database **empat entitas** berada di satu tempat. Kebocoran berkas konfigurasi
  berarti kebocoran **empat** basis data sekaligus, bukan satu — kredensialnya wajib dari
  penyimpanan rahasia (`D-40`, `R-17`).
- Nama entitas adalah **nama badan hukum**. Salah menampilkan portal berarti menampilkan data satu
  badan hukum di bawah nama badan hukum lain.

## Migrasi skema / rollout / rollback

Menambah tabel portal. Tidak menyentuh tabel klaim. Backward-compatible (`P-4`).

**Rollout:** portal utama lebih dulu, entitas lain menyusul — bukan keempatnya sekaligus.

**Rollback:** menonaktifkan portal lewat data. **Menonaktifkan portal yang sedang dipakai memutus
akses pengguna entitas itu** — bukan tindakan netral.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/portal/... -run TestDaftarPortalDariData
go run ./cmd/tools/cari-nama-entitas ./internal/...   # HARUS nol temuan
go test ./internal/app/portal/... -run TestPortalTidakAktifDitolak
go test ./internal/app/portal/... -run TestGagalStartBilaKonfigurasiPortalTidakSah
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 48 perbandingan `pxRequestor.pxReqServer`, 3 hostname | `docs/Steering/03-CURRENT-ARCHITECTURE.md:180` |
| Ambang komite berubah Rp 50.000.000 → 3.500 | `docs/Steering/11-SECURITY.md:219` |
| `IsServerSyariah` dan `IsDevelopmentServer` hilang, tidak tercatat di `19-GAP` | `docs/verifikasi-bukti-adr.md:788-791` |
| Satu database per entitas, minimal 4 portal | `D-75` · `ADR-0030` |
| Logika spreading syariah terpisah | `Activity/SpreadingSyariah_Act-Act.xml` |

## Comments
