---
title: "TKT-F3-005 — Middleware otorisasi di setiap endpoint"
labels: [modul::F-3, tipe::keamanan, status::ready-for-human, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F3-005 — Middleware otorisasi di setiap endpoint

Status: ready-for-human
Kesiapan: siap — **inilah bagian `F-3` yang berstatus PENUH**
Modul: F-3 · Gelombang: 1 · Bergantung pada: TKT-F3-003, TKT-F3-004
Requirement: FR-F3, FR-R1    Keputusan: D-59    ADR: 0023    Risiko: —
Rule Pega yang digantikan: **tidak ada penegakan di server sama sekali** — `pyPrivilegeName` terisi pada **1 dari 902 activity**, dan yang satu itu privilege bawaan Pega untuk ekspor ruleset, bukan aturan bisnis
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Setiap endpoint memeriksa kewenangan pemanggil **di server**, dan endpoint yang lupa
mendeklarasikan kewenangannya **ditolak oleh perkakas**, bukan lolos diam-diam.

Nilai bisnisnya adalah menutup celah yang terbukti ada. Hari ini, otorisasi sistem lama hanyalah
**penyembunyian menu** — siapa pun yang mengetahui alamat sebuah endpoint dapat memanggilnya tanpa
pemeriksaan apa pun. Angkanya tegas: **1 dari 902**.

## Ruang lingkup

- Middleware yang memeriksa: *apakah peran pemanggil memiliki menu yang memberi akses ke endpoint
  ini* (`D-59`).
- **Deklarasi kewenangan wajib pada setiap rute** — rute tanpa deklarasi **menggagalkan uji**,
  bukan berjalan sebagai publik.
- Penegakan **batas data** (cabang, lini bisnis, organisasi) di lapisan kueri, bukan dengan
  menyaring hasil setelah data terambil.
- Daftar rute publik yang eksplisit dan pendek: health check dan login.

## Non-goal

- **Tidak** merancang pemisahan tugas. `D-59` menetapkan **tidak ada** pemisahan tugas formal.
- **Tidak** mengisi tabel peran — itu `TKT-F3-004`.

## Acceptance criteria

- [ ] Permintaan tanpa sesi ke endpoint non-publik mengembalikan **`401`**; permintaan dengan sesi
      tetapi tanpa kewenangan mengembalikan **`403`** — keduanya diuji dan **dapat dibedakan**.
- [ ] Rute baru yang **tidak** mendeklarasikan kewenangan **menggagalkan uji** — diuji dengan rute
      percobaan; ini pemeriksaan yang mencegah endpoint lolos karena lupa.
- [ ] Daftar rute publik berisi **tepat** health check dan login — diuji; penambahan apa pun
      menggagalkan uji sampai daftar diperbarui secara sadar.
- [ ] Batas data ditegakkan **di kueri**: pengguna cabang A yang meminta daftar klaim tidak
      menerima klaim cabang B, dan **jumlah total yang dilaporkan juga tidak memuatnya** — diuji,
      karena menyaring setelah pengambilan membocorkan lewat jumlah baris.
- [ ] Pengguna `pyOrgUnit = "Eksternal"` dibatasi sesuai aturan sistem lama — diuji.
- [ ] Akses **data medis** hanya untuk peran Analyst Doctor dan RCL Dokter (`FR-R2`) — diuji, dan
      berlaku **juga di staging** karena staging memuat data nyata (`ADR-0029`).
- [ ] Setiap penolakan otorisasi tercatat di log dengan peran dan endpoint — **tanpa** data
      nasabah.

## Dependency / Blocked by

Bergantung pada `TKT-F3-003` dan `TKT-F3-004`.

**Catatan urutan:** middleware dapat dibangun dan diuji dengan tabel peran yang **masih kosong**,
memakai data uji. Yang menunggu daftar operator nyata adalah **pengisiannya**, bukan mekanismenya.

## Constraint keamanan, data, operasional

- **Kriteria penerimaan `BRD §21.2` #8** — otorisasi diperiksa di setiap endpoint — dipenuhi oleh
  tiket ini, dengan bacaan `D-59`: yang diperiksa adalah kepemilikan menu, bukan tindakan.
- Konsekuensi `D-59` diterima sadar: **satu orang dapat membuat, menyetujui, dan membayarkan satu
  klaim** bila perannya memiliki ketiga menu. Kontrol pengimbangnya adalah jejak audit `S-5`.
- **`AutoAcceptKomite` melewati kontrol ini sepenuhnya** — job terjadwal harian jam 06:00
  menyetujui komite tanpa pengguna sama sekali (`ADR-0022`). Middleware tidak berlaku pada jalur
  job, dan itu harus disadari saat `S-6` dirancang.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** menonaktifkan middleware **tidak tersedia sebagai rollback** —
itu membuka seluruh endpoint. Bila terjadi masalah, yang dilakukan adalah memperbaiki deklarasi
kewenangan rute yang salah, bukan mematikan pemeriksaannya.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/http/... -run TestOtorisasi
go test ./internal/adapter/http/... -run TestRuteTanpaDeklarasiGagal
go test ./internal/adapter/http/... -run TestBatasDataCabang
go run ./cmd/tools/daftar-rute --publik    # HARUS tepat 2
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `pyPrivilegeName` terisi 1 dari 902 activity | `T-10` · `docs/verifikasi-bukti-adr.md` §10.3 |
| Satuan izin adalah menu; penegakan pindah ke server | `D-59` · `ADR-0023` |
| Batas data: cabang, lini bisnis, organisasi | `docs/Steering/11-SECURITY.md` §3.2 |
| Batas data ditegakkan di kueri, bukan disaring setelahnya | idem |
| Akses data medis dibatasi dua peran | `FR-R2` · `BRD §17.2` |
| `AutoAcceptKomite` melewati kontrol menu | `D-57` · `ADR-0022` |

## Comments
