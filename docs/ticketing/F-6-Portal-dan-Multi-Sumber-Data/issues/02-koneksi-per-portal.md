---
title: "TKT-F6-002 — Koneksi dan pool per portal"
labels: [modul::F-6, tipe::fondasi, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F6-002 — Koneksi dan pool per portal

Status: needs-info
Kesiapan: **terhalang keputusan jumlah portal** (penomoran klaim tertutup )
Modul: **F-6 Portal & Multi-Sumber Data** · Gelombang: 1 · Bergantung pada: TKT-F6-001, TKT-F2-001
Requirement: FR-F6    Keputusan: D-75, D-71, D-76    ADR: 0030, 0004    Risiko: R-20
Rule Pega yang digantikan: **tidak ada padanan** — sistem lama memakai satu koneksi dan membedakan perilaku lewat hostname
Peran penguji gerbang 2: **PncAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Setiap portal punya **pool koneksinya sendiri**, dan seluruh pembacaan serta penulisan mengikuti
portal yang sedang aktif.

Nilai bisnisnya adalah alasan `D-75` menolak database bersama: dengan koneksi terpisah, **pemisahan
data menjadi sifat struktural**. Portal A tidak dapat membaca data portal B karena koneksinya memang
berbeda — bukan karena setiap kueri ingat menyaring. Kelas kesalahan "ada satu kueri yang lupa
menyaring" **hilang seluruhnya**, dan kelas itu tidak dapat diuji habis.

## Ruang lingkup

- Pool koneksi per portal, dibuat dari konfigurasi `TKT-F6-001`.
- Seam Repository (`TKT-F2-001`) mengambil koneksi **dari portal aktif**, bukan dari variabel global.
- Batas jumlah koneksi **per portal**, bukan satu batas dibagi rata.
- Pool laporan terpisah (`TKT-F2-001`) tetap berlaku — **per portal**.
- Perilaku bila database satu portal mati: portal lain **tetap berjalan**.

## Non-goal

- **Tidak** menyatukan data antar portal dengan cara apa pun.
- **Tidak** membangun laporan lintas portal — `D-75` menetapkan laporan **per portal**.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Berapa portal**, dan apakah keempat database sudah tersedia | **Work Owner + Tim Infra** | Menentukan jumlah pool, kebutuhan memori, dan apa yang dapat diuji |
| ~~Nomor klaim unik lintas portal?~~ — **TERTUTUP `D-76`**: tidak perlu penanda portal; nomor unik **di dalam portal**, tidak dijamin unik antar portal | — | Konsekuensinya dicatat di `D-76`: menyebut nomor klaim tanpa portalnya menjadi **ambigu**, termasuk pada LOD/PLA/DLA yang keluar ke pihak luar |
| **Batas koneksi per portal berapa?** | **Work Owner + Tim Infra** | Empat pool sekaligus melipatkan kebutuhan koneksi di sisi database |

> **Penomoran klaim sudah diputuskan** (`D-76`): tanpa penanda portal. Bila kelak diputuskan ada
> laporan konsolidasi lintas portal, keputusan itu **harus ditinjau ulang lebih dulu** — menambah
> penanda setelah klaim terbit menuntut migrasi data.

## Acceptance criteria

- [ ] Setiap portal punya pool sendiri, dan jumlah pool **sama dengan jumlah portal aktif** — diuji.
- [ ] Kueri yang dijalankan saat portal A aktif **tidak pernah menyentuh database portal B** —
      diuji dengan pencatatan koneksi pada 20 operasi berbeda: **nol kebocoran**.
- [ ] **Database satu portal mati tidak menjatuhkan portal lain** — diuji dengan mematikan satu
      database: portal lain tetap melayani.
- [ ] Batas koneksi ditegakkan **per portal**; satu portal yang sibuk **tidak menghabiskan jatah**
      portal lain — diuji dengan beban pada satu portal.
- [ ] Laporan berat di satu portal tidak menghabiskan koneksi transaksi portal itu maupun portal
      lain (`TKT-F2-001`) — diuji.
- [ ] Tidak ada jalur kode yang dapat memilih koneksi **tanpa portal aktif** — diuji: permintaan
      tanpa portal ditolak, bukan jatuh ke default.
- [ ] Gerbang 2: UAT **PncAdmin** pada minimal dua portal.

## Dependency / Blocked by

`TKT-F6-001` · `TKT-F2-001` · `TKT-F2-006` (nomor klaim). **Terhalang Work Owner dan Tim Infra.**

## Constraint keamanan, data, operasional

- **Jatuh ke koneksi default saat portal tidak diketahui adalah kebocoran data antar badan hukum.**
  Perilaku yang benar adalah menolak permintaan, bukan menebak portalnya.
- Empat pool melipatkan kebutuhan koneksi di sisi database. Bila kapasitasnya tidak disiapkan,
  portal keempat akan gagal saat portal lain sedang sibuk.
- Kredensial keempat database tunduk `D-40` dan `R-17` — dari penyimpanan rahasia, tidak pernah
  masuk repositori.
- `P-1` (satu tabel ditulis satu sistem) kini berlaku **di dalam tiap database**, dan tetap berlaku
  penuh selama Pega dan Go berjalan berdampingan (`D-05`) — **di keempat entitas**.

## Migrasi skema / rollout / rollback

Tidak mengubah tabel klaim. Menambah kebutuhan konfigurasi per portal.

**Rollout:** satu portal pada satu waktu. Portal yang belum dialihkan tetap memakai Pega.

**Rollback:** mengembalikan lalu lintas portal itu ke Pega. Rollback **per portal**, bukan seluruhnya.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/db/... -run TestPoolPerPortal
go test ./internal/adapter/db/... -run TestKueriTidakBocorAntarPortal
go test ./internal/adapter/db/... -run TestDatabaseSatuPortalMati
go test ./internal/adapter/db/... -run TestTanpaPortalDitolakBukanDefault
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Satu database per entitas | `D-75` · `ADR-0030` |
| Alasan menolak database bersama berkolom entitas | `ADR-0030` bagian Opsi |
| Pool laporan terpisah | `docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md` butir 7 |
| Nomor klaim `PNCN.YY.xxxx` dari sequence, tanpa penanda portal | `D-71` · `D-76` |
| Satu tabel ditulis satu sistem | `P-1` · `docs/Steering/07-MIGRATION-STRATEGY.md:15` |

## Comments
