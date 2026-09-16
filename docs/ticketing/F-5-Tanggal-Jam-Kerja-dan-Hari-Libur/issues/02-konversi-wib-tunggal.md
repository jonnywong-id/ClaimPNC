---
title: "TKT-F5-002 — Konversi WIB tunggal dan penghapusan penyesuaian 7 jam"
labels: [modul::F-5, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F5-002 — Konversi WIB tunggal dan penghapusan penyesuaian 7 jam

Status: needs-info
Kesiapan: **terhalang keputusan** — arti `addCalendar(…, 12, 0, 0)` belum diketahui
Modul: F-5 · Gelombang: 1 · Bergantung pada: TKT-F5-001
Requirement: FR-F5    Keputusan: D-49 butir 3    ADR: 0017    Risiko: R-12
Rule Pega yang digantikan: **118 titik penyesuaian waktu di 36 activity**, di antaranya penyesuaian 7 jam yang **asimetris di dalam satu kondisi validasi** — `Activity/InputRegister_act-Act.xml:5788` versus `:4805` — dan **101 titik `addCalendar(…, 12, 0, 0)` di 4 activity**
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu tempat yang mengubah UTC menjadi WIB untuk ditampilkan, dan **nol penyesuaian jam manual**
di dalam aturan bisnis.

Nilai bisnisnya adalah menutup cacat yang **mengubah hasil validasi**. Penyesuaian 7 jam di sistem
lama diterapkan tidak konsisten di dalam satu kondisi validasi yang sama, sehingga klaim dengan
tanggal kejadian di batas periode polis **bisa lolos atau ditolak tergantung cabang mana yang
dijalankan**. Ini butir 3 pada daftar 13 perbaikan eksplisit `P-5` — artinya selisihnya pada uji
kesetaraan sudah dinyatakan di muka sebagai perbaikan terencana.

## Ruang lingkup

- Satu fungsi konversi UTC→WIB untuk tampilan dan WIB→UTC untuk masukan, dipakai seluruh aplikasi.
- **Penghapusan seluruh penyesuaian jam manual** dari aturan bisnis — tidak ada `+7`, `-7`, atau
  `addCalendar` di dalam logika validasi.
- Pemeriksaan pola yang **menggagalkan build** bila penyesuaian jam manual muncul kembali.
- Daftar 118 titik yang terdampak, dipakai sebagai daftar periksa saat modul bisnis ditulis.

## Non-goal

- **Tidak** memperbaiki data historis yang mungkin tergeser — itu pekerjaan migrasi data, dan
  butuh jawaban DBA lebih dulu.
- **Tidak** mengubah aturan bisnis tanggalnya sendiri — hanya cara waktunya diperlakukan.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diketahui | Pemilik | Kenapa menahan |
|---|---|---|
| **Arti `addCalendar(…, 12, 0, 0)` pada 101 titik di 4 activity** — menambah 12 jam, atau menetapkan waktu ke tengah hari? | **Work Owner + Tim Pega** | Keduanya menghasilkan tanggal berbeda pada kasus batas. Menebaknya berarti mengubah hasil validasi 101 titik tanpa sadar |
| **Adakah data produksi yang tergeser ganda +14 jam?** | **DBA** | Bila ada, konversi yang benar di sistem baru akan menampilkan tanggal yang **berbeda** dari yang dilihat pengguna hari ini — dan itu harus disiapkan, bukan dikejutkan |

## Acceptance criteria

- [ ] **Nol penyesuaian jam manual** di seluruh kode aturan bisnis — dihitung dan dilaporkan
      angkanya; pemeriksaan pola menggagalkan build bila muncul.
- [ ] Konversi WIB berada di **tepat satu berkas** — diuji pemindaian.
- [ ] Waktu yang ditampilkan untuk satu nilai UTC **sama persis** dengan yang ditampilkan Pega
      untuk nilai yang sama — diuji pada 20 nilai contoh yang mencakup tengah malam WIB dan
      pergantian bulan.
- [ ] Validasi "tanggal kejadian dalam periode polis" memberi hasil **yang sama untuk kedua
      cabang** yang di sistem lama berbeda — diuji khusus pada tanggal batas.
- [ ] Daftar 118 titik terdampak tersedia sebagai berkas dengan `berkas:baris`, dan jumlahnya
      dapat dihitung ulang.

## Dependency / Blocked by

Bergantung pada `TKT-F5-001`. Terhalang dua pertanyaan di atas.

## Constraint keamanan, data, operasional

- Perubahan ini **mengubah hasil validasi** pada kasus batas. Karena sudah terdaftar sebagai butir
  3 `P-5`, selisihnya pada gerbang 1 **lolos otomatis dan cukup dicatat** (`D-54`) — tetapi hanya
  bila selisihnya memang terpetakan ke butir itu.
- Tidak ada perubahan data. Bila ternyata dibutuhkan perbaikan data historis, itu **tiket
  tersendiri** dengan persetujuan Work Owner dan DBA.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** mengembalikan fungsi konversi — tetapi perlu disadari bahwa
mengembalikannya berarti **memulihkan cacat asimetris**, sehingga rollback hanya masuk akal bila
ada temuan yang lebih buruk.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
grep -rInE "(\+|\-)\s*7\s*\*\s*time\.Hour|addCalendar" internal/   # HARUS 0 baris
go test ./internal/platform/waktu/... -run TestKonversiWIB
go test ./internal/domain/... -run TestPeriodePolisBatasTanggal
diff <(go run ./cmd/tools/tampilkan-waktu contoh.json) baseline-pega.json
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Penyesuaian 7 jam asimetris di satu kondisi validasi | `Activity/InputRegister_act-Act.xml:5788` dan `:4805` |
| Butir 3 daftar 13 perbaikan eksplisit `P-5` | `D-49` · `ADR-0017` · `docs/Steering/07-MIGRATION-STRATEGY.md` |
| 101 titik `addCalendar(…,12,0,0)` di 4 activity | `docs/verifikasi-bukti-adr.md` §15 baris `F-5` |
| 118 titik di 36 activity | idem |
| Selisih terpetakan `P-5` lolos otomatis di gerbang 1 | `D-54` · `docs/Steering/14-TESTING-STRATEGY.md` §6.2 |

## Comments
