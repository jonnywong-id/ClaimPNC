---
title: "TKT-F4-004 — Master Mata Uang dan Kurs per tanggal"
labels: [modul::F-4, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F4-004 — Master Mata Uang dan Kurs per tanggal

Status: needs-info
Kesiapan: **terhalang artefak** — isi `m_currencystandard` belum ada
Modul: F-4 · Gelombang: 1 · Bergantung pada: TKT-F4-001, TKT-F5-001
Requirement: FR-F4    Keputusan: D-48, D-49 butir 4 dan 5    ADR: 0015, 0017    Risiko: R-19
Rule Pega yang digantikan: `Database/GETCURRENCYSTANDARD.fnc` — memakai **kurs hari eksekusi** (`:3` versus `:14`) dan mengembalikan **`1`** ketika kurs tidak ditemukan (`:20-22`)
Peran penguji gerbang 2: **User Admin** dan **PncPICTeknik** (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Kurs menjadi master **per tanggal**, dan konversi memakai **kurs pada tanggal kejadian** — bukan
kurs hari klaim diproses.

Nilai bisnisnya adalah menutup cacat yang **membuat klaim besar lolos tanpa komite**. Fungsi kurs
sistem lama mengembalikan `1` ketika kurs tidak ditemukan: satu satuan valuta asing dihitung setara
satu Rupiah, sehingga klaim bernilai besar menyusut menjadi kecil, lalu **lolos di bawah seluruh
ambang komite**. Tidak ada galat, tidak ada catatan, dan angkanya tampak wajar.

## Ruang lingkup

- Master mata uang dan **kurs per tanggal**.
- Fungsi konversi yang memakai **kurs pada tanggal kejadian** (`D-48`).
- Perilaku saat kurs tidak ditemukan: **menolak** dengan galat yang menyebutkan **mata uang dan
  tanggalnya** — tidak ada nilai bawaan dan tidak ada kurs pengganti.
- Layar pengelolaan kurs, termasuk pengisian mundur untuk tanggal yang terlewat.

## Non-goal

- **Tidak** mengotomatiskan pengambilan kurs dari sumber luar — sumbernya belum ditetapkan.
- **Tidak** memperbaiki nilai historis yang telanjur dihitung dengan kurs `1`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Isi tabel `m_currencystandard`** | **DBA** (`R-19`) | Tanpa isinya, `B-5` tidak dapat diuji sama sekali — dan master ini tidak dapat diisi |
| **Siapa yang mengisi kurs harian dan dari sumber apa** — Bank Indonesia, kurs korporat, atau lain | **Work Owner** | Menentukan apakah layar pengisian manual cukup, atau perlu jalur pengambilan otomatis |
| **Bagaimana klaim yang tertolak karena kurs kosong diperlakukan** — ditahan dan diproses ulang otomatis, atau dikembalikan ke petugas? | **Work Owner** | Menentukan perilaku operasional yang terlihat pengguna setiap hari |

## Acceptance criteria

- [ ] Konversi memakai kurs pada **tanggal kejadian**, bukan tanggal proses — diuji dengan klaim
      yang tanggal kejadiannya berbeda dari hari pengujian.
- [ ] Kurs tidak ditemukan menghasilkan **galat yang menyebut mata uang dan tanggal**, dan
      transaksi **ditolak** — diuji; **nilai `1` tidak boleh muncul di jalur mana pun**.
- [ ] Nilai Rupiah sebuah klaim **tidak berubah** ketika klaim yang sama diproses ulang di hari
      berbeda — diuji.
- [ ] Kurs dapat diisi mundur untuk tanggal yang terlewat, dan pengisiannya tercatat di jejak
      audit (`TKT-F4-001`).
- [ ] Perbandingan terhadap ambang komite memakai **nilai presisi penuh** hasil konversi
      (`ADR-0016`) — diuji.

## Dependency / Blocked by

Bergantung pada `TKT-F4-001` dan `TKT-F5-001`. **Terhalang DBA dan dua keputusan Work Owner.**

**Yang bergantung padanya:** `B-5` Settlement — satu-satunya modul yang **masih terikat
`BRD §21.4`**, dan salah satu dari tiga penghalangnya adalah isi tabel ini.

## Constraint keamanan, data, operasional

- **Dampak operasional yang harus disiapkan sebelum rilis:** klaim valuta asing yang dulu lolos
  kini **dapat ditolak** sampai kursnya dilengkapi. Ini perbaikan yang diinginkan, tetapi ia
  menghentikan pekerjaan petugas bila master kurs tidak terisi tepat waktu.
- **Uji kesetaraan akan menampilkan selisih pada seluruh data historis valuta asing**, karena
  laporan lama memakai kurs hari eksekusi. Selisih ini **sudah terdaftar** sebagai butir 8 dan 9
  pada 13 perbaikan eksplisit `P-5`, sehingga lolos otomatis di gerbang 1 (`D-54`).
- Master kurs menjadi **data kritis**: kelengkapannya per tanggal menentukan apakah klaim dapat
  diproses sama sekali.

## Migrasi skema / rollout / rollback

Menambah tabel master kurs — tidak menyentuh tabel Pega.

**Rollback:** kembali ke `GETCURRENCYSTANDARD` **tidak tersedia** — mengembalikannya berarti
memulihkan cacat `RETURN 1`. Bila master kurs bermasalah, yang dilakukan adalah melengkapi
datanya, bukan mengembalikan fungsi lama.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/kurs/... -run TestKursTanggalKejadian
go test ./internal/domain/kurs/... -run TestKursTidakDitemukanMenolak
grep -rIn "return 1" internal/domain/kurs/     # HARUS 0 baris
go test ./internal/domain/komite/... -run TestAmbangMemakaiPresisiPenuh
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Kurs pada tanggal kejadian; tolak bila tidak ada | `D-48` · `ADR-0015` |
| `GETCURRENCYSTANDARD` memakai kurs hari eksekusi dan `RETURN 1` | `Database/GETCURRENCYSTANDARD.fnc:3`, `:14`, `:20-22` |
| Butir 4 dan 5 dari 13 perbaikan eksplisit `P-5` | `D-49` · `ADR-0017` |
| Isi `m_currencystandard` belum ada | `R-19` · `D-55` |
| `B-5` masih terikat `BRD §21.4` | `D-55` · `ADR-0028` |

## Comments
