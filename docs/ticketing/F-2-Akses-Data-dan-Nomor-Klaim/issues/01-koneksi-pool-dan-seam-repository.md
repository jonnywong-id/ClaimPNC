---
title: "TKT-F2-001 — Koneksi, connection pool, dan seam Repository"
labels: [modul::F-2, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F2-001 — Koneksi, connection pool, dan seam Repository

Status: ready-for-human
Kesiapan: siap
Modul: F-2 · Gelombang: 1 · Bergantung pada: TKT-F1-001, TKT-F1-002
Requirement: FR-F2    Keputusan: D-01, D-10, D-21    ADR: 0004, 0005    Risiko: —
Rule Pega yang digantikan: lapisan Connect-SQL Pega — **652 rule SQL**; koneksi diatur `Data-Admin-DB-Name` yang **tidak ada di export**
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu pintu masuk ke database, dengan pool yang terukur, dan **antarmuka Repository yang
dideklarasikan lapisan Domain** — sehingga aturan bisnis dapat diuji tanpa database sama sekali.

Nilai bisnisnya: `D-10` menetapkan profil sistem ini **beban data besar, konkurensi rendah** —
ribuan klaim per bulan di atas data puluhan juta baris, tetapi hanya 200–300 pengguna harian.
Pool yang dirancang untuk profil itu berbeda dari pool yang dirancang untuk lalu lintas tinggi,
dan salah merancangnya menghabiskan koneksi database yang **dibagi dengan Pega** selama masa
paralel.

## Ruang lingkup

- Pembukaan koneksi ke Oracle 19c dengan parameter dari konfigurasi (`TKT-F1-002`).
- Connection pool dengan batas yang **dapat dikonfigurasi** dan nilai baku yang sesuai profil
  `D-10`.
- **Antarmuka Repository dideklarasikan di lapisan Domain**, implementasinya di lapisan Adapter —
  sehingga arah ketergantungan tetap sesuai `ADR-0001`.
- Satu implementasi Repository contoh dan satu implementasi tiruan untuk pengujian.
- **Pool terpisah untuk laporan**, agar satu laporan berat tidak menghabiskan koneksi transaksi.

## Non-goal

- **Tidak** menulis kueri bisnis apa pun.
- **Tidak** membangun PostgreSQL sekarang — target akhir memang PostgreSQL (`ADR-0005`), tetapi
  runtime sementara Oracle.
- **Tidak** memanggil stored procedure (`ADR-0007`).

## Acceptance criteria

- [ ] Aplikasi terhubung ke Oracle memakai parameter dari konfigurasi, dan **gagal start dengan
      pesan jelas** bila parameternya salah.
- [ ] Batas pool (maksimum koneksi, idle, umur koneksi) **dapat dikonfigurasi**, dan nilai
      bakunya terdokumentasi beserta alasannya terhadap profil `D-10`.
- [ ] Antarmuka Repository berada di paket lapisan **Domain**; pemeriksaan lapisan `TKT-F1-001`
      **gagal** bila implementasinya ikut masuk ke sana — diuji dengan commit percobaan.
- [ ] Aturan bisnis contoh dapat diuji **tanpa database** memakai Repository tiruan — dibuktikan
      dengan `go test` yang lulus saat database dimatikan.
- [ ] Pool laporan terpisah dari pool transaksi, dan **menghabiskan pool laporan tidak
      memengaruhi** permintaan transaksi — diuji dengan menahan seluruh koneksi laporan.
- [ ] Setiap kueri yang dijalankan tercatat di log dengan lamanya (`TKT-F1-003`).

## Dependency / Blocked by

Bergantung pada `TKT-F1-001` dan `TKT-F1-002`.

## Constraint keamanan, data, operasional

- **Kredensial database hanya dari konfigurasi luar proses** (`ADR-0025`).
- Selama masa paralel, database **dibagi dengan Pega** (`ADR-0004`). Pool yang terlalu besar
  memakan koneksi yang dibutuhkan Pega untuk melayani produksi — batasnya harus dibicarakan
  dengan DBA sebelum dinaikkan.
- **Parameter binding tanpa perkecualian** — tidak ada perangkaian nilai ke dalam teks SQL.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** mengembalikan konfigurasi pool; tidak ada data yang berubah.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/sqlstore/...
go test ./internal/domain/... # HARUS lulus tanpa database hidup
golangci-lint run             # aturan lapisan: Domain tidak mengimpor driver SQL
# uji isolasi pool laporan:
go test ./internal/adapter/sqlstore/... -run TestPoolLaporanTidakMengganggu
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Beban data besar, konkurensi rendah; 200–300 pengguna harian | `D-10` · `docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md` |
| Database dibagi dengan Pega selama masa paralel | `D-21` · `ADR-0004` |
| Seam Repository dideklarasikan Domain | `docs/Steering/04-FUTURE-ARCHITECTURE.md` §3.1 |
| Pool terpisah untuk laporan | `docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md` butir 7 |
| 652 rule SQL sebagai lapisan yang digantikan | `docs/Steering/03-CURRENT-ARCHITECTURE.md` |

## Comments
