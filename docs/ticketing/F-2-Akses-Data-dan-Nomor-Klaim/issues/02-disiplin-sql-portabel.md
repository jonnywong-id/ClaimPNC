---
title: "TKT-F2-002 — Disiplin SQL portabel dan pemeriksaan pola terlarang"
labels: [modul::F-2, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F2-002 — Disiplin SQL portabel dan pemeriksaan pola terlarang

Status: ready-for-human
Kesiapan: siap
Modul: F-2 · Gelombang: 1 · Bergantung pada: TKT-F2-001
Requirement: FR-F2    Keputusan: D-01, D-20, D-24    ADR: 0005    Risiko: R-19
Rule Pega yang digantikan: seluruh 652 rule Connect-SQL — di antaranya **411 pemakaian `TO_CHAR`**, **68 pemakaian `ROWNUM`**, **538 pemakaian `{ASIS:}`**
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Aturan penulisan SQL yang **ditegakkan perkakas**, sehingga satu set kueri berjalan di Oracle
hari ini dan PostgreSQL kelak tanpa ditulis dua kali.

Nilai bisnisnya bukan hanya portabilitas. Mengeluarkan `TO_CHAR` dari SQL sekaligus **menutup
cacat yang sudah berjalan**: sistem lama mengembalikan tanggal sebagai string `'dd/mm/yyyy'`,
sehingga pengurutan tanggal menjadi pengurutan teks — `01/12/2024` dianggap lebih kecil daripada
`02/01/2020` — dan penyaringan rentang tanggal tidak dapat memakai index.

## Ruang lingkup

- Daftar **padanan sintaks yang mengikat**, ditulis sebagai dokumen yang dapat dirujuk saat review.
- **Pemeriksaan otomatis pada build** yang menggagalkan commit bila menemukan pola terlarang:
  `TO_CHAR` untuk memformat tanggal/angka, `ROWNUM`, `NVL`, `SYSDATE` di luar generator nomor
  klaim, dan perangkaian string ke dalam SQL.
- **Pemformatan tanggal dan angka dipindahkan ke Go** — SQL mengembalikan tipe tanggal, bukan
  string.
- **Paginasi diseragamkan** ke `OFFSET … FETCH NEXT … ROWS ONLY`; untuk inbox dan pencarian
  dipakai **keyset pagination**.
- Satu berkas contoh kueri yang menunjukkan pola yang benar untuk masing-masing kasus.

## Non-goal

- **Tidak** menulis ulang 652 kueri sistem lama — itu tersebar di modul bisnis masing-masing.
- **Tidak** menyiapkan lingkungan PostgreSQL — itu tugas infrastruktur, dan tanpa lingkungan itu
  klaim "portabel" **tidak terverifikasi**.
- **Tidak** menetapkan daftar putih kolom filter — itu `TKT-F2-007`.

## Acceptance criteria

- [ ] Dokumen padanan sintaks memuat minimal: `NVL`→`COALESCE`, `ROWNUM`→`FETCH NEXT`,
      `SYSDATE`→parameter waktu dari seam Clock, `TO_CHAR`→pemformatan di Go, penggabungan
      string, dan tipe boolean.
- [ ] Pemeriksaan pola terlarang **menggagalkan build** pada berkas percobaan yang memuat
      `ROWNUM` — diuji dengan commit percobaan.
- [ ] Pemeriksaan **menggagalkan build** pada perangkaian string ke dalam SQL (`"… WHERE x = " +
      nilai`).
- [ ] **Nol `TO_CHAR` untuk pemformatan** di seluruh kueri yang ada — dihitung dan dilaporkan
      angkanya.
- [ ] Kueri contoh mengembalikan kolom tanggal sebagai **tipe tanggal**, bukan string —
      dibuktikan uji yang mengurutkan tiga tanggal lintas tahun dan memeriksa urutannya.
- [ ] Paginasi keyset pada kueri contoh menghasilkan **hasil yang sama** dengan `OFFSET` untuk
      1.000 baris pertama, dan **rencana eksekusinya tidak memindai seluruh tabel** — dilampirkan
      keluaran `EXPLAIN PLAN`.

## Dependency / Blocked by

Bergantung pada `TKT-F2-001`.

**Catatan kejujuran:** portabilitas hanya dapat **dibuktikan** bila kedua mesin benar-benar diuji.
Selama lingkungan PostgreSQL 17+ belum ada, tiket ini menghasilkan **disiplin**, bukan bukti.
Itu perbedaan yang harus disebut saat tiket ini ditutup.

## Constraint keamanan, data, operasional

- **Parameter binding tanpa perkecualian.** Ini sekaligus menutup celah warisan: sistem lama
  menyisipkan nilai pengguna langsung ke teks SQL lewat `{ASIS:}` pada **538 tempat**.
- Nama kolom untuk sort dan filter **hanya dari daftar yang diizinkan** — daftarnya ditetapkan
  `TKT-F2-007`, dan sampai itu ada, kueri baru **tidak boleh** menerima nama kolom dari pengguna.
- `SYSDATE` **dilarang** di luar generator nomor klaim; waktu berasal dari seam Clock (`F-5`).

## Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** menonaktifkan pemeriksaan pola; kueri yang sudah ditulis
tetap sah.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
make lint-sql                     # pemeriksaan pola terlarang
grep -rIn -E "TO_CHAR\(" internal/ | grep -v nomor_klaim   # HARUS 0 baris
grep -rIn "ROWNUM" internal/                                # HARUS 0 baris
go test ./internal/adapter/sqlstore/... -run TestPaginasiKeyset
# uji negatif: tambahkan ROWNUM di satu kueri, lalu
make lint-sql                     # HARUS gagal
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 411 `TO_CHAR`, 68 `ROWNUM`, pola paginasi sudah dipakai 35 rule | `D-20` · `ADR-0005` |
| Tanggal dikembalikan sebagai string `'dd/mm/yyyy'` → pengurutan salah | `docs/Steering/09-DATABASE-STRATEGY.md` §3.2 |
| 538 pemakaian `{ASIS:}` | `docs/verifikasi-bukti-adr.md` §15 baris `F-2` |
| PostgreSQL 17+ adalah persyaratan mengikat | `D-24` |
| `OFFSET` nol kemunculan; masalah nyata 3.189 grid page list | `T-12` · `docs/Steering/09-DATABASE-STRATEGY.md` §6.3 |

## Comments
