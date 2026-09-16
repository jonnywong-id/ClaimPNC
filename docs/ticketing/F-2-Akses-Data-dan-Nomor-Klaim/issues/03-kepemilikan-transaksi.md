---
title: "TKT-F2-003 — Kepemilikan transaksi di lapisan aplikasi"
labels: [modul::F-2, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F2-003 — Kepemilikan transaksi di lapisan aplikasi

Status: ready-for-human
Kesiapan: siap
Modul: F-2 · Gelombang: 1 · Bergantung pada: TKT-F2-001
Requirement: FR-F2    Keputusan: D-02, D-68    ADR: 0007    Risiko: R-01
Rule Pega yang digantikan: kepemilikan transaksi yang kini berada **di dalam stored procedure** — `Database/INSERT_PLADLA.prc` ber-`COMMIT` **9×** (`:69`, `:74`, `:79`, `:138`, `:143`, `:148`, `:179`, `:184`, `:189`) dengan satu `ROLLBACK` di `:198` yang terjadi **setelah** commit; `Database/UPDATEREAS.prc` ber-`COMMIT` 4×
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Transaksi dimulai dan diakhiri **di satu tempat yang dapat dibaca** — lapisan aplikasi — sehingga
sebuah operasi bisnis berhasil seluruhnya atau gagal seluruhnya.

Nilai bisnisnya konkret dan bernilai uang. Penerbitan PLA/DLA di sistem lama menempuh **sembilan
`COMMIT`**, dan satu-satunya `ROLLBACK` berada di handler terluar yang dijalankan **setelah**
commit terjadi — sehingga tidak memulihkan apa pun. Bila proses berhenti di tengah, sebagian
pemberitahuan ke reasuransi sudah tercatat dan sebagian belum, tanpa cara membatalkannya.

## Ruang lingkup

- Mekanisme transaksi yang dimulai dan diakhiri di lapisan `App`, **bukan** di Repository maupun
  handler HTTP.
- Aturan **satu permintaan pengguna = satu transaksi**, dengan pengecualian yang wajib
  didokumentasikan di tempatnya.
- **Larangan pemanggilan sistem eksternal di dalam transaksi database** — ditegakkan pemeriksaan,
  bukan imbauan.
- Pola meneruskan transaksi lewat `context` sehingga Repository tidak perlu tahu apakah ia sedang
  berada di dalam transaksi.

## Non-goal

- **Tidak** menulis ulang procedure mana pun — itu milik `B-4`, `B-9`, dan modul nilai uang
  lainnya. Tiket ini menyediakan mekanismenya.
- **Tidak** memanggil stored procedure (`ADR-0007`).
- **Tidak** mematikan procedure di database — itu langkah tersendiri yang menunggu verifikasi
  `ALL_DEPENDENCIES`.

## Acceptance criteria

- [ ] Operasi contoh yang menulis ke **tiga tabel** dan gagal di tabel ketiga **tidak
      meninggalkan satu baris pun** — diuji otomatis.
- [ ] Transaksi dimulai **hanya** di lapisan `App`; pemeriksaan lapisan **gagal** bila Repository
      atau handler memulainya — diuji dengan commit percobaan.
- [ ] Pemanggilan HTTP keluar di dalam transaksi **menggagalkan build** lewat pemeriksaan pola —
      diuji dengan commit percobaan.
- [ ] Transaksi yang dibiarkan tanpa commit maupun rollback **terdeteksi** dan menghasilkan galat
      pada uji, bukan koneksi yang menggantung.
- [ ] Lama transaksi tercatat di log (`TKT-F1-003`); transaksi melebihi ambang yang dikonfigurasi
      menghasilkan log bertingkat `warn`.

## Dependency / Blocked by

Bergantung pada `TKT-F2-001`.

**Yang bergantung padanya:** `B-4` Spreading dan `B-9` PLA/DLA — keduanya baru dapat dibuat
atomik setelah mekanisme ini ada.

## Constraint keamanan, data, operasional

- Selama masa paralel, **penulis tunggal per tabel** (`P-1`) tetap berlaku: transaksi yang benar
  tidak menyelamatkan tabel yang ditulis dua sistem.
- Transaksi yang lebih panjang menahan kunci baris lebih lama. Dengan 200–300 pengguna (`D-10`)
  risikonya kecil, **tetapi tidak nol pada job massal** — itu sebabnya lama transaksi dicatat.
- **Kontrak galat berbasis string `ErrMsg` tidak dibawa.** Pada enam procedure, `ErrMsg` tidak
  di-set pada jalur sukses sehingga `NULL` berarti berhasil; pada
  `Database/ADD_NEWMASTERVIRTUALACCOUNT.prc:18` kolom yang sama membawa **nomor virtual account
  sekaligus pesan galat**.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** mengembalikan pembungkus transaksi ke versi sebelumnya.

> **Perubahan perilaku yang disengaja dan harus dicatat untuk gerbang 1.** Sistem lama
> meninggalkan data setengah jalan saat gagal; sistem baru tidak meninggalkan apa pun. Kasus uji
> kesetaraan `B-4`/`B-9` harus dirancang menyadari ini, atau ia akan melaporkan **selisih palsu**.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/... -run TestTransaksiAtomik
golangci-lint run                 # aturan: transaksi hanya dimulai di lapisan App
grep -rIn -E "http\.(Get|Post|Client)" internal/app/ | grep -i "tx\|transaksi"   # HARUS 0 baris
go test ./internal/adapter/sqlstore/... -run TestTransaksiMenggantung
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 9 `COMMIT` dan `ROLLBACK` sesudahnya | `Database/INSERT_PLADLA.prc:69,74,79,138,143,148,179,184,189,198` |
| 10 dari 12 procedure melakukan `COMMIT` sendiri | `D-68` · `ADR-0007` |
| `ErrMsg` membawa nomor VA sekaligus pesan galat | `Database/ADD_NEWMASTERVIRTUALACCOUNT.prc:18` |
| Transaksi dimulai di lapisan aplikasi | `docs/Steering/08-TECHNICAL-STRATEGY.md` §4.5 |
| Claim PNC satu-satunya pemanggil procedure | `D-68` — **belum diverifikasi katalog** |

## Comments
