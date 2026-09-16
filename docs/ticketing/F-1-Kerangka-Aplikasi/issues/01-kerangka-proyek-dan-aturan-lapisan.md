---
title: "TKT-F1-001 — Kerangka proyek Go dan penegakan aturan lapisan"
labels: [modul::F-1, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F1-001 — Kerangka proyek Go dan penegakan aturan lapisan

Status: ready-for-human
Kesiapan: siap
Modul: F-1 · Gelombang: 1 · Bergantung pada: —
Requirement: FR-F1    Keputusan: D-08, D-09    ADR: 0001    Risiko: R-11
Rule Pega yang digantikan: — (tidak ada padanan; Pega menyediakan struktur ini lewat platform)
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi, cukup gerbang 1 + persetujuan Work Owner (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Sebuah repository Go yang bisa di-*clone*, di-*build*, dan dijalankan, dengan **struktur folder
yang sudah final** dan aturan ketergantungan antar lapisan yang **ditegakkan perkakas, bukan
kesepakatan lisan**.

Nilai bisnisnya tidak langsung terlihat pengguna, tetapi ia yang menentukan biaya seluruh modul
berikutnya. `D-09` menetapkan tim adalah developer Pega yang sedang dilatih ulang; pada tim
seperti itu, struktur yang tidak seragam berubah menjadi 33 gaya penulisan berbeda dalam beberapa
bulan — dan itu persis kegagalan yang membuat sistem lama sulit dipelihara.

## Ruang lingkup

- Struktur folder final beserta penjelasan **apa yang boleh dan tidak boleh berada di masing-masing
  folder**, mengikuti empat lapisan `ADR-0001`: `Domain` · `App` · `Adapter` · `Entrypoint`.
- Aturan ketergantungan antar lapisan **dipasang sebagai pemeriksaan otomatis** (`depguard` atau
  setara) yang **menggagalkan build**, bukan sekadar memperingatkan.
- Berkas `Makefile`/skrip build yang menghasilkan **satu binary** berisi SPA tersemat.
- Linter backend dengan konfigurasi yang sudah disepakati, dijalankan di pemeriksaan yang sama.
- Satu contoh modul "hello" yang menembus keempat lapisan, dipakai sebagai **acuan pola** bagi
  modul berikutnya.

## Non-goal

- **Tidak** membangun modul bisnis apa pun.
- **Tidak** menyiapkan pipeline CI/CD — itu disiapkan tim GitLab (`D-33`).
- **Tidak** memilih pustaka frontend — itu `TKT-U2-005`.
- **Tidak** menyiapkan Docker atau orkestrator apa pun (`D-08`).

## Acceptance criteria

- [ ] `go build ./...` menghasilkan **satu berkas binary** tanpa dependensi runtime eksternal.
- [ ] Struktur folder memuat **tepat empat lapisan** dengan nama yang disepakati, dan setiap
      lapisan punya berkas `doc.go` yang menyatakan apa yang boleh berada di dalamnya.
- [ ] Pemeriksaan ketergantungan **gagal** (exit code ≠ 0) bila lapisan `Domain` mengimpor paket
      HTTP, SQL, atau JSON — diuji dengan satu commit percobaan yang sengaja melanggar.
- [ ] Pemeriksaan ketergantungan **gagal** bila lapisan `Adapter` mengimpor lapisan `App`.
- [ ] Linter berjalan bersih (**0 temuan**) pada seluruh berkas yang ada.
- [ ] Modul contoh dapat dipanggil lewat satu endpoint HTTP dan mengembalikan respons dengan
      bentuk yang sama dengan kontrak galat `TKT-F1-004` bila gagal.
- [ ] Dokumen `CONTRIBUTING.md` atau setara menyebutkan **aturan penamaan berkas, paket, dan
      fungsi**, cukup preskriptif untuk diikuti tanpa bertanya (`D-09`).

## Dependency / Blocked by

Tidak ada. Ini tiket paling awal di seluruh papan.

**Yang bergantung padanya:** seluruh tiket `F-2`…`F-5`, `S-5`, dan seluruh modul bisnis.

## Constraint keamanan, data, operasional

- Binary **tidak boleh** memuat nilai rahasia apa pun. Rahasia hanya dari luar proses
  (`ADR-0025`) — dan tujuan penyimpanannya masih `OPEN` (`D-40`), sehingga tiket ini hanya
  menyiapkan **tempat membacanya**, bukan memutuskan sumbernya.
- Aplikasi wajib **stateless** (`D-27`): tidak ada state yang hanya hidup di memori satu instans.

## Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** menghapus repository; tidak ada dampak ke sistem
yang berjalan.

## Rencana verifikasi

> **Rencana — belum dijalankan.** Belum ada kode implementasi di repository ini.

```bash
go build ./...                      # harus menghasilkan satu binary
go vet ./...                        # harus bersih
golangci-lint run                   # harus 0 temuan
go test ./...                       # contoh modul harus lulus
# uji negatif: tambahkan import "net/http" di satu berkas Domain, lalu:
golangci-lint run                   # HARUS gagal — bila lolos, aturan lapisan tidak tegak
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Satu binary, VM on-premise, tanpa orkestrator | `ADR-0001` · `D-08` |
| Empat lapisan dan aturan ketergantungannya | `docs/Steering/04-FUTURE-ARCHITECTURE.md` §2 |
| Struktur preskriptif karena tim eks-Pega | `D-09` |
| SPA disajikan binary Go | `ADR-0002` · `D-23` |
| Stateless dan dua instans | `D-27` |

## Comments
