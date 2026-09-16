---
title: "TKT-F1-002 — Konfigurasi tiga lapis dan gagal keras saat start"
labels: [modul::F-1, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F1-002 — Konfigurasi tiga lapis dan gagal keras saat start

Status: ready-for-human
Kesiapan: siap
Modul: F-1 · Gelombang: 1 · Bergantung pada: TKT-F1-001
Requirement: FR-F1    Keputusan: D-15, D-27, D-40    ADR: 0025    Risiko: R-17
Rule Pega yang digantikan: — Pega memakai Dynamic System Setting; **nol DSS ditemukan di export**. Konfigurasi dinamis sistem lama berupa tabel Oracle yang dikunci per IP aplikasi
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu tempat membaca konfigurasi, dengan urutan prioritas yang jelas, dan **aplikasi yang menolak
start bila konfigurasi wajib tidak ada**.

Nilai bisnisnya: sistem lama mengubah perilaku bisnis berdasarkan **nama server** — salah satunya
mengubah ambang komite dari Rp 50.000.000 menjadi 3.500 (`ADR-0025`). Perilaku seperti itu tidak
dapat diuji dan berubah diam-diam saat server dipindahkan. Tiket ini menyiapkan penggantinya.

## Ruang lingkup

- Tiga lapis konfigurasi, dari umum ke khusus: **nilai baku di kode → berkas YAML per lingkungan →
  variabel lingkungan**, yang paling khusus menang.
- **Validasi saat start**: seluruh konfigurasi wajib diperiksa keberadaannya dan bentuknya; bila
  ada yang kurang, aplikasi **berhenti dengan pesan yang menyebut nama setting-nya**.
- Pemisahan tegas antara **konfigurasi teknis** (alamat database, ukuran pool, batas waktu, port,
  tingkat log, masa berlaku token) dan **master data** — master data **bukan** urusan tiket ini,
  ia milik `F-4`.
- Antarmuka pembaca rahasia yang **implementasinya dapat diganti** tanpa menyentuh pemanggil.

## Non-goal

- **Tidak** memutuskan ke mana rahasia dipindahkan — itu `D-40`, masih `OPEN`, milik Tim
  Infra/Security. Tiket ini hanya menyiapkan antarmukanya.
- **Tidak** membangun master data (`F-4`).
- **Tidak** memuat mekanisme konfigurasi yang berubah tanpa restart — konfigurasi teknis memang
  butuh restart; yang berubah tanpa restart adalah master data.

## Acceptance criteria

- [ ] Nilai yang sama didefinisikan di ketiga lapis menghasilkan **nilai dari variabel
      lingkungan** — dibuktikan uji otomatis.
- [ ] Menjalankan aplikasi tanpa satu setting wajib menghasilkan **exit code ≠ 0** dan pesan yang
      **menyebut nama setting yang kurang** — bukan pesan umum.
- [ ] Tidak ada satu pun nilai rahasia di dalam berkas YANG MASUK repository — diuji pemindaian
      pola pada pemeriksaan build.
- [ ] **Nol perilaku yang bergantung pada nama host.** Diuji: menjalankan aplikasi dengan
      `hostname` berbeda menghasilkan perilaku identik pada seluruh uji yang ada.
- [ ] Daftar seluruh setting beserta artinya tersedia di satu berkas dokumentasi, dan
      **jumlahnya sama** dengan yang dibaca kode — diuji otomatis, bukan diperiksa manual.

## Dependency / Blocked by

- Bergantung pada `TKT-F1-001` (struktur folder dan aturan lapisan).
- **Tidak terhalang** `D-40`: yang `OPEN` adalah tujuan penyimpanan rahasia, bukan cara membacanya.
  Bagian yang menunggu `D-40` adalah **adapter**-nya, dan itu tiket terpisah di `F-4`/deployment.

## Constraint keamanan, data, operasional

- **Rahasia hanya dari luar proses.** Tidak pernah dari berkas YAML yang masuk repository
  (`ADR-0025`).
- Export rule memuat **3 password SMTP di 31 lokasi** dan 1 pasang kredensial OAuth, seluruhnya
  plaintext (`R-17`). Tiket ini **tidak** memindahkannya — ia menyiapkan tempatnya. Pemindahannya
  menunggu `D-40`.
- Konfigurasi **tidak boleh** dikunci per IP aplikasi seperti sistem lama — itu bertabrakan dengan
  dua instans (`D-27`).

## Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** mengembalikan berkas konfigurasi ke versi sebelumnya dan
menjalankan ulang; tidak ada data yang berubah.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/config/...                 # urutan prioritas tiga lapis
APP_DB_DSN= go run ./cmd/app                  # HARUS exit ≠ 0 dan menyebut nama setting
hostname-sim A go test ./...                  # perilaku identik lintas hostname
grep -rIn -E "(password|secret|api[_-]?key)\s*[:=]" config/   # HARUS 0 baris
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Tiga lapis konfigurasi dan gagal keras saat start | `docs/Steering/12-CROSSCUTTING.md` §3.1 |
| Nol Dynamic System Setting di sistem lama; konfigurasi dikunci per IP | `docs/Steering/12-CROSSCUTTING.md` §3.5 · `T-8` |
| Hostname menentukan ambang komite | `Activity/GetKomiteApproval-Act.xml:335` · `ADR-0025` |
| 3 password SMTP di 31 lokasi + 1 pasang OAuth | `R-17` · `docs/verifikasi-bukti-adr.md` §7.6 |
| Pemisahan konfigurasi teknis vs master data | `docs/Steering/12-CROSSCUTTING.md` §3.2–3.3 |

## Comments
