---
title: "TKT-F2-007 — Daftar putih kolom filter dan sort pengganti {ASIS:}"
labels: [modul::F-2, tipe::keamanan, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F2-007 — Daftar putih kolom filter dan sort pengganti `{ASIS:}`

Status: needs-info
Kesiapan: **terhalang keputusan** — himpunan filter yang sah belum ditetapkan
Modul: F-2 · Gelombang: 1 · Bergantung pada: TKT-F2-002
Requirement: FR-F2, FR-R1    Keputusan: D-15    ADR: 0005, 0023    Risiko: —
Rule Pega yang digantikan: **538 pemakaian `{ASIS:}`** di rule Connect-SQL — penyisipan nilai langsung ke teks SQL tanpa parameter binding
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Cara menyaring dan mengurutkan daftar yang **tidak pernah menerima nama kolom dari pengguna**,
menggantikan pola `{ASIS:}` yang menyisipkan potongan SQL apa adanya.

Nilai bisnisnya adalah menutup **satu-satunya celah injeksi SQL yang terbukti ada** di sistem
lama. Parameter binding menutup nilai; ia **tidak** menutup nama kolom dan arah pengurutan — dan
itulah yang `{ASIS:}` sisipkan.

## Ruang lingkup

- Mekanisme daftar putih: setiap daftar mendeklarasikan **kolom mana yang boleh difilter dan
  diurutkan**, dan permintaan di luar daftar itu **ditolak**, bukan diabaikan diam-diam.
- Pemetaan **nama yang dilihat klien** ke nama kolom database, sehingga struktur tabel tidak bocor
  lewat parameter URL.
- Pemeriksaan otomatis: perangkaian apa pun ke dalam klausa `WHERE` atau `ORDER BY`
  menggagalkan build.
- Inventaris 538 pemakaian `{ASIS:}`, dikelompokkan menurut **apa yang sebenarnya disisipkan**:
  nilai (dapat langsung diparameterkan), nama kolom, potongan klausa, atau daftar `IN`.

## Non-goal

- **Tidak** menulis ulang 538 kueri — itu tersebar di modul bisnis masing-masing. Tiket ini
  menyediakan mekanisme dan inventarisnya.
- **Tidak** menetapkan izin per peran atas kolom — itu `F-3`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Himpunan filter yang sah untuk setiap daftar** — 538 pemakaian `{ASIS:}` belum dipilah mana yang benar-benar dipakai pengguna dan mana yang sisa tambalan lama | **Work Owner** (mana yang dipakai bisnis) + **Lead Engineer** (pemilahan teknis) | Daftar putih yang dibuat dari tebakan akan **menghapus penyaringan yang dipakai orang**, dan itu baru ketahuan setelah rilis |

**Yang tidak terhalang dan boleh dikerjakan lebih dulu:** mekanisme daftar putihnya sendiri,
pemetaan nama klien → kolom, dan pemeriksaan otomatis. Yang menunggu adalah **isi** daftarnya.

## Acceptance criteria

- [ ] Permintaan filter pada kolom **di luar daftar putih** mengembalikan galat `400` dengan pesan
      yang menyebut nama parameter — **bukan** diabaikan dan **bukan** `500`.
- [ ] Permintaan sort pada kolom di luar daftar putih ditolak dengan cara yang sama.
- [ ] Nama kolom database **tidak pernah muncul** di parameter permintaan maupun respons galat —
      diuji dengan permintaan yang sengaja salah.
- [ ] Pemeriksaan pola **menggagalkan build** pada perangkaian ke klausa `WHERE`/`ORDER BY` —
      diuji dengan commit percobaan.
- [ ] Inventaris 538 pemakaian `{ASIS:}` tersedia sebagai berkas, terbagi ke **empat kategori**,
      dengan jumlah per kategori yang dapat dihitung ulang.
- [ ] Uji injeksi: nilai `1; DROP TABLE x --` pada setiap parameter filter **tidak mengubah
      apa pun** dan menghasilkan galat yang sama dengan masukan tidak sah biasa.

## Dependency / Blocked by

- Bergantung pada `TKT-F2-002`.
- **Terhalang keputusan** di atas untuk isi daftar putihnya.

## Constraint keamanan, data, operasional

- **Parameter binding tanpa perkecualian** untuk nilai; daftar putih untuk nama kolom. Keduanya
  wajib, dan tidak saling menggantikan.
- Sampai daftar putih ada, kueri baru **tidak boleh** menerima nama kolom dari pengguna sama
  sekali — lebih baik daftar tanpa penyaringan daripada daftar dengan celah injeksi.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** melonggarkan daftar putih **tidak boleh** dilakukan sebagai
rollback — bila sebuah filter yang dibutuhkan ternyata tidak ada di daftar, yang ditambah adalah
daftarnya, bukan mekanismenya.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/http/... -run TestDaftarPutihFilter
go test ./internal/adapter/http/... -run TestInjeksiSQLDitolak
grep -rIn -E "(WHERE|ORDER BY).*\+" internal/adapter/sqlstore/   # HARUS 0 baris
make lint-sql
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 538 pemakaian `{ASIS:}` | `docs/verifikasi-bukti-adr.md` §15 baris `F-2` |
| Parameter binding wajib; nama kolom sort/filter hanya dari daftar yang diizinkan | `docs/Steering/11-SECURITY.md` §5 · `BRD` §17.3 |
| Perangkaian SQL dari nilai pengguna sebagai utang teknis | `docs/Steering/03-CURRENT-ARCHITECTURE.md` |
| Kriteria penerimaan #11 — tanpa perangkaian SQL | `BRD §21.2` |

## Comments
