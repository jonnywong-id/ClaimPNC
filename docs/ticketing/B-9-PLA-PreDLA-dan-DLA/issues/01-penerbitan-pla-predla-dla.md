---
title: "TKT-B09-001 — Penerbitan PLA, Pre-DLA, dan DLA"
labels: [modul::B-9, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::5]
milestone: "Gelombang 5 — Nilai dan pihak luar"
epic: "Migrasi Claim PNC"
---

# TKT-B09-001 — Penerbitan PLA, Pre-DLA, dan DLA

Status: needs-info
Kesiapan: terhalang keputusan
Modul: **B-9 PLA/Pre-DLA/DLA** · Gelombang: 5 · Bergantung pada: TKT-B04-001, TKT-B05-001, TKT-F2-003
Requirement: FR-B9    Keputusan: D-02, D-49 butir 7, D-68    ADR: 0007, 0017    Risiko: R-01
Rule Pega yang digantikan: `Database/INSERT_PLADLA.prc` — **9 `COMMIT`** (`:69`, `:74`, `:79`, `:138`, `:143`, `:148`, `:179`, `:184`, `:189`), `ROLLBACK` di `:198` **setelah** commit; parameter `TTGLPLADLA` diterima lalu **dibuang**, diganti `SYSDATE` (`:67`, `:136`, `:177`) · inbox `InboxPLADLA`, `InboxPLA_harness`
Peran penguji gerbang 2: **PncPLADLA** dan **TreatyIn**

## Hasil yang diharapkan (dan nilai bisnisnya)

Ketiga dokumen pemberitahuan dapat diterbitkan, dan penerbitannya **berhasil seluruhnya atau gagal
seluruhnya**.

Nilai bisnisnya konkret: hari ini penerbitan menempuh **sembilan `COMMIT`**, dan `ROLLBACK`-nya
dijalankan **setelah** commit — sehingga tidak memulihkan apa pun. Bila proses berhenti di tengah,
sebagian pemberitahuan ke reasuransi sudah tercatat dan sebagian belum, **tanpa cara
membatalkannya**.

## Ruang lingkup

- Penerbitan **PLA**, **Pre-DLA**, dan **DLA** dengan data dari spreading (`B-4`) dan nilai
  settlement (`B-5`).
- Seluruh penerbitan di dalam **satu transaksi** (`TKT-F2-003`) — inilah yang dilepaskan
  `ADR-0007`.
- **Tanggal dokumen diambil dari waktu sistem**, bukan dari pilihan pengguna — perilaku ini
  **direplikasi secara sadar** (`D-49` butir 7), dan parameter `TTGLPLADLA` **dihapus seluruhnya**.
- Pemeriksaan duplikat sebelum menerbitkan.

## Non-goal

- **Tidak** mengirim dokumen — itu `TKT-B09-002`.
- **Tidak** memanggil `INSERT_PLADLA` (`ADR-0007`).

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Tiga perilaku berbeda untuk PLA, DLA, dan PREDLA — mana yang benar?** Ketiganya diterbitkan lewat procedure yang sama tetapi dengan jalur berbeda | **Work Owner** | Menentukan aturan penerbitan masing-masing; menebak berarti salah pada salah satunya |
| **PLA tanpa nilai sah?** | **Work Owner** | Menentukan apakah PLA boleh terbit sebelum estimasi ada |
| **Kunci duplikat PLA 5 kolom versus DLA 6 kolom — sengaja?** | **Work Owner** | Bila tidak sengaja, salah satunya punya celah duplikasi |
| **`T_PREDLALIST` masih dipakai?** | **Work Owner** | Bila tidak, Pre-DLA tidak perlu dibawa sama sekali |
| Status `VALID`/`INVALID` `GET_GROUPBUSINESS_XOL` di produksi | **DBA** | Menentukan perlakuan XOL pada penerbitan |

## Acceptance criteria

- [ ] Penerbitan yang gagal di tengah **tidak meninggalkan satu baris pun** — diuji dengan
      kegagalan sengaja pada baris ke-3 dari 5. Inilah bukti apa yang dilepaskan `ADR-0007`.
- [ ] **Tanggal dokumen sama dengan waktu sistem**, dan parameter tanggal dari pengguna **tidak
      ada sama sekali** di kontrak — diuji pemindaian.
- [ ] Pemeriksaan duplikat menolak penerbitan kedua untuk kunci yang sama — diuji untuk PLA dan
      DLA masing-masing.
- [ ] **Nol pemanggilan stored procedure** — diuji pemindaian.
- [ ] Penerbitan tercatat di jejak audit dengan pelaku dan waktu.
- [ ] Gerbang 1: isi dokumen **sama dengan Pega** pada 20 klaim contoh. **Perilaku saat gagal
      berbeda secara sengaja** — kasus uji dirancang menyadarinya, atau ia melaporkan selisih palsu
      (`docs/Steering/14-TESTING-STRATEGY.md` §6.4).
- [ ] Gerbang 2: UAT **PncPLADLA** dan **TreatyIn**.

## Dependency / Blocked by

`TKT-B04-001`, `TKT-B05-001`, `TKT-F2-003`. **Terhalang empat keputusan dan satu artefak DBA.**

## Constraint keamanan, data, operasional

- Dokumen ini **dikirim ke pihak luar** — koasuransi, reasuransi, broker. Kesalahan isinya terlihat
  oleh mereka, bukan hanya internal.
- **Kontrak galat berbasis string `ErrMsg` tidak dibawa** (`ADR-0007`) — pada enam procedure
  `ErrMsg` tidak di-set pada jalur sukses sehingga `NULL` berarti berhasil.
- Nomor klaim yang tercetak di dokumen memakai format baru `PNCN.YY.xxxx` untuk klaim sistem baru
  (`ADR-0009`) — pihak luar akan menerima **dua bentuk nomor**.

## Migrasi skema / rollout / rollback

Menambah tabel PLA/DLA milik aplikasi. Backward-compatible.

**Rollback:** dokumen yang telanjur diterbitkan **tidak dapat ditarik** — ia sudah dikirim ke
pihak luar. Rollback berarti menghentikan penerbitan baru.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/pladla/... -run TestPenerbitanAtomik
go test ./internal/app/pladla/... -run TestTanggalDariWaktuSistem
go test ./internal/app/pladla/... -run TestTolakDuplikat
grep -rIn "CALL \|INSERT_PLADLA" internal/app/pladla/    # HARUS 0 baris
go run ./cmd/s8 banding --modul B-9 --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 9 `COMMIT` dan `ROLLBACK` sesudahnya | `Database/INSERT_PLADLA.prc:69,74,79,138,143,148,179,184,189,198` |
| `TTGLPLADLA` dibuang, diganti `SYSDATE` | `:67`, `:136`, `:177` · `D-49` butir 7 **REPLIKASI** |
| `B-9` dapat dibuat atomik | `D-68` · `ADR-0007` |
| Lepas dari `BRD §21.4` | `D-55` · `ADR-0028` |
| Istilah PLA, Pre-DLA, DLA | `CONTEXT.md` |

## Comments
