---
title: "TKT-B02-001 — Layar Input Register dan penerbitan nomor klaim"
labels: [modul::B-2, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B02-001 — Layar Input Register dan penerbitan nomor klaim

Status: needs-info
Kesiapan: terhalang keputusan (format nomor klaim)
Modul: **B-2 Input Register** · Gelombang: 3 · Bergantung pada: TKT-B01-001, TKT-F2-006, TKT-U2-002
Requirement: FR-B2    Keputusan: D-22, D-71    ADR: 0009, 0013, 0018    Risiko: R-16
Rule Pega yang digantikan: `Flow/Register_Flow.xml` tahap **`Input Register`** · `Activity/InputRegister_act-Act.xml` (137 step) · harness `InboxRegister_Harness`
Peran penguji gerbang 2: **PncAdmin** (User Admin) — peran yang mengisi layar ini setiap hari

## Hasil yang diharapkan (dan nilai bisnisnya)

Petugas admin dapat mencatat klaim baru dari awal sampai terbit **nomor klaim**, dengan layar yang
susunannya dikenali — urutan langkah dan penempatan field mengikuti Pega (`D-13`).

Nilai bisnisnya: ini **pintu masuk seluruh klaim**. Selama layar ini belum ada, tidak ada satu pun
klaim yang dapat dimulai di sistem baru, dan `P-3` menetapkan klaim yang sudah berjalan di Pega
tetap diselesaikan di Pega — jadi sistem baru hanya bisa dimulai dari sini.

## Ruang lingkup

- Layar Input Register: data kejadian (tanggal kejadian, tanggal lapor, tanggal terima dokumen,
  lokasi, kronologi, pelapor), pemilihan polis lewat `B-1`, dan nilai estimasi awal.
- **Penerbitan nomor klaim `PNCN.YY.xxxx`** memakai generator `TKT-F2-006`.
- Penyimpanan klaim sebagai satu transaksi utuh (`TKT-F2-003`) — objek, coverage, spreading, dan
  estimasi tersimpan bersama atau tidak sama sekali.
- Tombol **Back** yang mengembalikan klaim ke tahap sebelumnya, setara perilaku status `1146` di
  sistem lama.
- Pencatatan jejak audit pada penerbitan klaim (`S-5`).

## Non-goal

- **Tidak** memuat aturan validasi tanggal, duplikasi, dan kelengkapan — ketiganya tiket
  tersendiri (`TKT-B02-002`, `003`, `004`) agar dapat diuji dan direview terpisah.
- **Tidak** membangun layar objek/coverage — itu `B-3`.
- **Tidak** membangun spreading — itu `B-4`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Akibat |
|---|---|---|
| **Sequence nomor klaim direset tiap awal tahun?** dan **lebar segmen terakhir dibuat tetap?** (`TKT-F2-006`) | **Work Owner** | Menentukan bentuk nomor yang **tidak dapat ditarik kembali** setelah klaim pertama terbit |
| When rule `NonMBU`, `NotPA`, `ElseRCLMSIG` — percabangan flow utama | **Tim Pega** (`R-16`) | Menentukan ke tahap mana klaim diteruskan setelah register |

## Acceptance criteria

- [ ] Klaim baru dapat disimpan dan menerima nomor berformat `PNCN.YY.xxxx` — diuji 20 penyimpanan
      berturut-turut, **nol nomor ganda**.
- [ ] Penyimpanan yang gagal di langkah mana pun **tidak meninggalkan satu baris pun** — diuji
      dengan kegagalan yang sengaja dipicu di penyimpanan coverage.
- [ ] Urutan field dan tahapan layar **sama dengan `InboxRegister_Harness`** — dibuktikan dengan
      perbandingan tangkapan layar berdampingan yang disetujui penguji gerbang 2.
- [ ] Tombol Back mengembalikan klaim ke tahap sebelumnya tanpa kehilangan isian — diuji.
- [ ] Penerbitan klaim menghasilkan **tepat satu baris jejak audit** berisi pelaku dan waktu.
- [ ] Gerbang 1: hasil penyimpanan klaim contoh **sama dengan Pega** pada 20 kasus data staging —
      selisih yang muncul wajib terpetakan ke butir `P-5` (`D-54`).
- [ ] Gerbang 2: UAT oleh **PncAdmin** pada 10 klaim nyata dari empat lini bisnis berbeda.

## Dependency / Blocked by

`TKT-B01-001` (snapshot polis) · `TKT-F2-006` (nomor klaim) · `TKT-F2-003` (transaksi) ·
`TKT-U2-002` (form baku) · `TKT-S5-002` (jejak audit).

## Constraint keamanan, data, operasional

- Layar ini menampilkan **data nasabah** — nomor polis, nama tertanggung, NIK. Batas data cabang
  dan lini bisnis ditegakkan di kueri (`TKT-F3-005`), bukan disaring setelah diambil.
- **Nomor klaim tidak dapat ditarik kembali** setelah terbit; ia muncul di surat ke tertanggung
  dan di PLA/DLA ke reasuransi.
- Kepemilikan tabel: selama masa paralel, **satu sistem saja yang menulis** tabel klaim (`P-1`).
  Peralihan kewenangan terjadi saat modul ini lulus gerbang 2 — bukan sebelumnya.

## Migrasi skema / rollout / rollback

Menambah tabel klaim milik aplikasi (`ADR-0004`) dan sequence nomor klaim. Backward-compatible;
Pega tidak membacanya.

**Rollback:** klaim yang telanjur terbit di sistem baru **tetap ada dan tetap sah** — ia tidak
dapat dipindahkan ke Pega (`P-3`). Karena itu rollback modul ini berarti **menghentikan
pendaftaran klaim baru di sistem baru**, bukan membatalkan yang sudah terbit.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/registrasi/... -run TestPenerbitanNomorKlaim
go test ./internal/app/registrasi/... -run TestSimpanAtomik
go run ./cmd/s8 banding --modul B-2 --kasus 20     # gerbang 1 lewat S-8
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Tahap `Input Register` pada flow utama | `Flow/Register_Flow.xml` |
| 137 step validasi | `Activity/InputRegister_act-Act.xml` |
| Format nomor klaim `PNCN.YY.xxxx` | `D-71` · `ADR-0009` |
| Status `1146` di-set saat pengguna menekan Back | `docs/Steering/16-RISK-ANALYSIS.md` `R-06` |
| Klaim berjalan tidak berpindah sistem | `P-3` |

## Comments
