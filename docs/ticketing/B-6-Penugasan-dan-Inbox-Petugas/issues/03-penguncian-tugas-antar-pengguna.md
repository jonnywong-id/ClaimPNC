---
title: "TKT-B06-003 — Penguncian tugas antar pengguna"
labels: [modul::B-6, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B06-003 — Penguncian tugas antar pengguna

Status: ready-for-human
Kesiapan: siap
Modul: **B-6 Penugasan & Inbox** · Gelombang: 3 · Bergantung pada: TKT-B06-001
Requirement: FR-B6    Keputusan: D-26, D-27    ADR: 0019, 0026    Risiko: —
Rule Pega yang digantikan: penguncian assignment bawaan Pega — **tidak ada rule aplikasi** yang mengaturnya; di sistem baru ia harus dibangun sendiri
Peran penguji gerbang 2: **PncRCLPUCL** dan **PncComplience** — dua peran yang bekerja dari Workbasket bersama

## Hasil yang diharapkan (dan nilai bisnisnya)

Dua petugas **tidak dapat mengerjakan tugas yang sama** dari satu Workbasket bersama tanpa
menyadarinya.

Nilai bisnisnya: Workbasket adalah antrean bersama — RCL/PUCL, Investigator, dan Compliance
mengambil dari sana. Tanpa penguncian, dua orang dapat membuka klaim yang sama, dan yang menyimpan
belakangan **menimpa pekerjaan yang pertama** tanpa peringatan.

## Ruang lingkup

- Mekanisme **ambil tugas** (claim) dari Workbasket: tugas menjadi milik pengambil sampai selesai
  atau dilepas.
- Penanganan tugas yang **ditinggalkan** — pengguna menutup peramban tanpa melepas.
- Perilaku bila tugas sudah diambil orang lain: pesan yang **menyebut siapa** yang memegangnya.
- Pelepasan tugas oleh pemegang, dan oleh peran manajer bila pemegangnya tidak hadir.

## Non-goal

- **Tidak** mengunci baris database untuk waktu lama — penguncian ini urusan alur kerja, bukan
  transaksi.
- **Tidak** menangani ketidakhadiran terjadwal — kolom `STS_ABS` pada master komite adalah
  persoalan `B-7`.

## Acceptance criteria

- [ ] Tugas yang diambil pengguna A **tidak muncul sebagai tersedia** bagi pengguna B — diuji.
- [ ] Pengguna B yang membuka tugas itu lewat URL langsung menerima pesan yang **menyebut nama
      pemegangnya** — diuji.
- [ ] Tugas yang ditinggalkan tanpa dilepas **kembali tersedia** setelah batas waktu yang
      dikonfigurasi — diuji dengan batas pendek.
- [ ] Peran manajer dapat **melepas paksa** tugas milik orang lain, dan tindakan itu **tercatat di
      jejak audit** dengan pelaku dan alasan — diuji.
- [ ] Penguncian bekerja **lintas dua instans aplikasi** — diuji dengan mengambil tugas di instans
      A dan mencoba mengambilnya di instans B: **ditolak**. Ini konsekuensi langsung `D-27`.
- [ ] Gerbang 1: **tidak ada padanan langsung di Pega** untuk dibandingkan — kelulusan bertumpu
      pada uji fungsional di atas, bukan perbandingan hasil.
- [ ] Gerbang 2: UAT **PncRCLPUCL** dan **PncComplience** dengan dua pengguna bersamaan.

## Dependency / Blocked by

`TKT-B06-001` · `TKT-S5-002` (pencatatan pelepasan paksa).

## Constraint keamanan, data, operasional

- Penguncian **tidak boleh** disimpan di memori satu instans — aplikasi wajib stateless (`D-27`),
  dan load balancer mengarahkan permintaan ke instans mana pun.
- Melepas paksa tugas orang lain adalah tindakan bernilai tinggi; karena `D-59` tidak mengenal
  pemisahan tugas, **jejak auditlah satu-satunya kontrolnya**.
- Batas waktu tugas ditinggalkan harus cukup panjang untuk pekerjaan nyata — batas yang terlalu
  pendek membuat tugas direbut saat pengguna sedang mengetik.

## Migrasi skema / rollout / rollback

Menambah kolom pemegang dan waktu pengambilan pada tabel Tugas — kolom *nullable*,
backward-compatible.

**Rollback:** kolom dibiarkan dan diabaikan; tugas kembali dapat diambil siapa pun — dengan risiko
tumpang tindih yang justru dihilangkan tiket ini.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/penugasan/... -run TestAmbilTugas
go test ./internal/app/penugasan/... -run TestTugasDitinggalkanKembaliTersedia
go test ./internal/app/penugasan/... -run TestPenguncianLintasInstans
go test ./internal/app/penugasan/... -run TestLepasPaksaTercatat
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Workbasket adalah antrean bersama | `D-26` · `CONTEXT.md` — **Workbasket** |
| Tiga tahap memakai Workbasket | `D-26` · `Flow/Register_Flow.xml` |
| Stateless, dua instans | `D-27` |
| Tidak ada pemisahan tugas; audit satu-satunya kontrol | `D-59` · `ADR-0023` |

## Comments
