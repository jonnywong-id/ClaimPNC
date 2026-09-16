---
title: "TKT-F3-003 — Sesi dan token milik aplikasi"
labels: [modul::F-3, tipe::keamanan, status::ready-for-human, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F3-003 — Sesi dan token milik aplikasi

Status: ready-for-human
Kesiapan: siap
Modul: F-3 · Gelombang: 1 · Bergantung pada: TKT-F3-001, TKT-F2-001
Requirement: FR-F3    Keputusan: D-07, D-27    ADR: 0024    Risiko: —
Rule Pega yang digantikan: mekanisme sesi Pega — **tidak ada rule aplikasi yang mengaturnya**; tidak ada padanan yang dapat dibandingkan
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Aplikasi menerbitkan **sesi miliknya sendiri** setelah sistem identitas memvalidasi kredensial —
bukan meneruskan token pihak lain.

Nilai bisnisnya ada di tiga hal yang menjadi milik kita: masa berlaku, pencabutan, dan isi token.
Ditambah satu yang penting secara operasional: **bila HCC/HCQ sedang bermasalah, pengguna yang
sudah masuk tetap dapat bekerja** — sejalan dengan tuntutan 24/7 (`D-27`).

## Ruang lingkup

- Penerbitan token sesi setelah autentikasi berhasil, memuat identitas pengguna dan perannya.
- **Penyimpanan sesi aktif di database** agar dapat dicabut — bukan hanya token yang berdiri
  sendiri. Ini juga syarat aplikasi **stateless**: dua instans harus melihat sesi yang sama.
- Pembaruan sesi dan penghentian saat keluar.
- Middleware yang memuat identitas pengguna ke dalam `context` setiap permintaan.

## Non-goal

- **Tidak** memutuskan izin — itu `TKT-F3-005`.
- **Tidak** membangun layar login — itu `U-1`.

## Acceptance criteria

- [ ] Token yang diterbitkan **tidak memuat kredensial** dan tidak memuat data nasabah — diuji
      dengan membongkar isinya.
- [ ] Sesi tersimpan di database; **dua instans aplikasi mengenali sesi yang sama** — diuji dengan
      menerbitkan sesi di instans A dan memakainya di instans B.
- [ ] Mencabut sesi membuat permintaan berikutnya **ditolak seketika** — diuji, bukan menunggu
      masa berlaku habis.
- [ ] Token kedaluwarsa ditolak dengan galat yang **dapat dibedakan** dari token tidak sah.
- [ ] Identitas pengguna tersedia di `context` pada seluruh lapisan tanpa diteruskan sebagai
      parameter berantai — diuji.
- [ ] Masa berlaku sesi dapat dikonfigurasi; **nilai bawaan didokumentasikan beserta alasannya**.
- [ ] Keluar (logout) menghapus sesi dari database, bukan hanya dari peramban.

## Dependency / Blocked by

Bergantung pada `TKT-F3-001` dan `TKT-F2-001`.

**Terbuka tetapi tidak menahan:** masa berlaku sesi final menunggu Work Owner dan Security
(`ADR-0024`). Nilai sementara dipakai dan ditandai di konfigurasi; mengubahnya tidak menyentuh
kode.

## Constraint keamanan, data, operasional

- **Aplikasi wajib stateless** (`D-27`): sesi **tidak boleh** hidup di memori satu instans, karena
  load balancer mengarahkan permintaan ke instans mana pun.
- Token **tidak pernah** masuk log, termasuk sebagian isinya.
- Pencabutan sesi harus bekerja **seketika** — itulah alasan sesi disimpan, bukan sekadar token
  yang memverifikasi dirinya sendiri.
- Tabel sesi memuat identitas pengguna; ia tunduk soft delete (`ADR-0012`) dan tercatat di jejak
  audit bila dicabut oleh administrator.

## Migrasi skema / rollout / rollback

Menambah tabel sesi — tabel baru, **tidak menyentuh tabel yang dibaca Pega**.

**Rollback:** seluruh sesi menjadi tidak sah dan pengguna harus masuk ulang. Itu gangguan yang
dapat diterima, tetapi **harus diumumkan** sebelum rilis mundur dijalankan pada jam kerja.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/sesi/...
go test ./internal/app/sesi/... -run TestSesiLintasInstans
go test ./internal/app/sesi/... -run TestPencabutanSeketika
grep -rIn "token" internal/platform/log/   # HARUS tidak ada penulisan token ke log
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Aplikasi menerbitkan sesi sendiri; alasan masa berlaku, pencabutan, isi token | `D-07` · `docs/Steering/11-SECURITY.md` §2.1 |
| Stateless, dua instans di belakang load balancer | `D-27` |
| Tabel `SessionAktif` untuk pencabutan | `docs/Steering/11-SECURITY.md` §3.3 |
| Masa berlaku sesi masih terbuka | `ADR-0024` Pertanyaan terbuka |

## Comments
