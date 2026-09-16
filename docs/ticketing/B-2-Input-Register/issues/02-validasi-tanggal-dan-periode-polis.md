---
title: "TKT-B02-002 — Validasi urutan tanggal dan periode polis"
labels: [modul::B-2, tipe::aturan-bisnis, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B02-002 — Validasi urutan tanggal dan periode polis

Status: needs-info
Kesiapan: terhalang keputusan
Modul: **B-2 Input Register** · Gelombang: 3 · Bergantung pada: TKT-B02-001, TKT-F5-002
Requirement: FR-B2    Keputusan: D-49 butir 3    ADR: 0017    Risiko: R-12
Rule Pega yang digantikan: `Activity/InputRegister_act-Act.xml` — kondisi validasi periode polis pada `:5788` dan `:4805` (**penyesuaian 7 jam yang asimetris di dalam satu kondisi yang sama**)
Peran penguji gerbang 2: **PncAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Aturan tanggal klaim yang **memberi hasil sama untuk kasus yang sama**, berapa kali pun diuji.

Nilai bisnisnya: hari ini hasilnya **tidak selalu sama**. Penyesuaian 7 jam diterapkan di satu
cabang kondisi tetapi tidak di cabang lainnya **di dalam kondisi validasi yang sama**, sehingga
klaim dengan tanggal kejadian tepat di batas periode polis bisa lolos atau ditolak tergantung jalur
mana yang dijalankan. Ini butir 3 pada 13 perbaikan eksplisit `P-5`.

## Ruang lingkup

- Aturan urutan: **tanggal kejadian ≤ tanggal lapor ≤ tanggal terima dokumen ≤ hari ini**.
- Aturan periode polis: tanggal kejadian berada di dalam periode polis, ditambah toleransi
  **30 hari untuk Bonding** dan **90 hari untuk Travel dan PA**.
- Seluruh perbandingan tanggal memakai seam Clock dan konversi WIB tunggal (`F-5`) — **nol
  penyesuaian jam manual**.
- Pesan galat yang menyebut **tanggal mana** yang salah dan **batas** yang dilanggar.

## Non-goal

- **Tidak** mengubah angka toleransi 30 dan 90 hari — bila berubah, itu keputusan bisnis
  tersendiri.
- **Tidak** menangani validasi selain tanggal.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Toleransi 30 hari Bonding — mati, atau BRD-nya salah?** Pembacaan source tidak menemukan jalur yang benar-benar memakainya | **Work Owner** | Menentukan apakah aturannya dibawa atau dihapus. Membawanya bila sebenarnya mati berarti menambah aturan yang tidak pernah berlaku |
| **Jalur API/JSON melewati aturan 7/30/90 — sengaja?** Klaim yang masuk lewat jalur itu tidak melewati validasi ini | **Work Owner** | Bila tidak sengaja, ini celah yang sudah berjalan; bila sengaja, sistem baru harus menirunya |
| **Arti `addCalendar(…, 12, 0, 0)` pada 101 titik** (`TKT-F5-002`) | **Work Owner + Tim Pega** | Menentukan hasil perbandingan pada kasus batas |

## Acceptance criteria

- [ ] Ketiga urutan tanggal diuji pada **kasus batas**: sama persis, selisih satu hari, dan selisih
      satu detik melewati tengah malam WIB — hasilnya **deterministik**.
- [ ] Tanggal kejadian tepat pada **hari pertama** dan **hari terakhir** periode polis diterima;
      satu hari di luar ditolak — diuji keempat batas.
- [ ] Toleransi 90 hari Travel dan PA diuji pada hari ke-90 (diterima) dan ke-91 (ditolak).
- [ ] **Kedua cabang** kondisi yang di sistem lama berbeda kini memberi **hasil identik** — diuji
      khusus pada tanggal batas; inilah bukti butir 3 `P-5` benar-benar tertutup.
- [ ] **Nol penyesuaian jam manual** di kode modul ini — diuji pemindaian.
- [ ] Pesan galat menyebut tanggal yang salah dan batas yang dilanggar — diuji ketiga jenis galat.
- [ ] Gerbang 1: selisih terhadap Pega **hanya** pada kasus batas yang terpetakan ke butir 3 `P-5`.

## Dependency / Blocked by

`TKT-B02-001`, `TKT-F5-001`, `TKT-F5-002`.

## Constraint keamanan, data, operasional

- Perubahan ini **mengubah hasil validasi** pada kasus batas. Ia sudah terdaftar sebagai butir 3
  `P-5`, sehingga selisihnya lolos otomatis di gerbang 1 (`D-54`) — **hanya bila** selisihnya
  memang terpetakan ke butir itu.
- `R-12`: bila data yang disalin ke staging bergeser zona waktunya, pengujian ini akan melaporkan
  selisih palsu. Salinan harus diperiksa lebih dulu.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** mengembalikan aturan berarti **memulihkan cacat asimetris** —
hanya masuk akal bila ada temuan yang lebih buruk.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/registrasi/... -run TestUrutanTanggal
go test ./internal/domain/registrasi/... -run TestPeriodePolisBatas
grep -rInE "(\+|\-)\s*7\s*\*\s*time\.Hour|addCalendar" internal/domain/registrasi/   # HARUS 0
go run ./cmd/s8 banding --modul B-2 --aturan tanggal
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Penyesuaian 7 jam asimetris dalam satu kondisi | `Activity/InputRegister_act-Act.xml:5788` dan `:4805` |
| Butir 3 dari 13 perbaikan eksplisit `P-5` | `D-49` · `ADR-0017` |
| Invarian `I-2` dan `I-3` | `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Toleransi 30 hari Bonding, 90 hari Travel/PA | `BRD §11.1` |
| 101 titik `addCalendar(…,12,0,0)` | `docs/verifikasi-bukti-adr.md` §15 baris `F-5` |

## Comments
