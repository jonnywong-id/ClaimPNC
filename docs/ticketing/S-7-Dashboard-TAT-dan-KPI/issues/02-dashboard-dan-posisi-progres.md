---
title: "TKT-S7-002 — Dashboard klaim dan posisi progres"
labels: [modul::S-7, tipe::migrasi, status::needs-info, prioritas::sedang, gelombang::6]
milestone: "Gelombang 6 — Laporan"
epic: "Migrasi Claim PNC"
---

# TKT-S7-002 — Dashboard klaim dan posisi progres

Status: needs-info
Kesiapan: **terhalang keputusan `TKT-S7-001`**
Modul: **S-7 Dashboard, TAT & KPI** · Gelombang: 6 · Bergantung pada: TKT-S7-001, TKT-S2-001
Requirement: FR-S7    Keputusan: D-18, D-49    ADR: 0011    Risiko: R-19
Rule Pega yang digantikan: harness `PNCTATReport`, `ReportKPIHarness`, `OutstandingKlaimperCabang_Harness` · `Database/GET_POSISI_PROGRESS_PNC.fnc:15` · `PROGRESS_CLAIM_PNC:7-8,40,67`
Peran penguji gerbang 2: **PncManagerAdmin** dan **PncKacab**

## Hasil yang diharapkan (dan nilai bisnisnya)

Layar yang menunjukkan **klaim mana yang masih menggantung, di tangan siapa, dan sudah berapa lama**.

Nilai bisnisnya: inilah satu-satunya tempat kepala cabang melihat pekerjaan yang tertahan. Tetapi
saat ini layar itu **menampilkan seluruh riwayat sebagai "aktif"**, karena penyaringnya keliru —
sehingga yang seharusnya menjadi alat pengawasan justru menyembunyikan keadaan sebenarnya.

## Ruang lingkup

- Dashboard klaim, KPI, dan outstanding per cabang.
- **Penyaring "aktif" memakai `PROGRESSDATEDONE IS NULL`**, bukan `STATUSPOSISI = 'On Progress'`.
- Angka TAT diambil dari **satu** fungsi (`TKT-S7-001`).
- Penegakan batas data cabang: kepala cabang melihat cabangnya (`TKT-F3-005`).

## Non-goal

- **Tidak** menentukan basis TAT — itu `TKT-S7-001`.
- **Tidak** menambah metrik baru yang belum ada di sistem lama.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Basis TAT belum ditetapkan** (`TKT-S7-001`) | **Work Owner** | Seluruh angka di dashboard ini turunan darinya |
| **Memperbaiki penyaring "aktif" akan menurunkan angka outstanding secara tajam** — diumumkan bagaimana? | **Work Owner** | Baris yang sudah selesai selama ini ikut terhitung aktif. Setelah diperbaiki, angkanya berubah besar — dan itu **terlihat sebagai penurunan mendadak** bila tidak dijelaskan |
| **Default follow-up `sysdate + 7`** — 7 hari itu aturan bisnis yang masih berlaku? | **Work Owner** | Satu-satunya angka bisnis literal di `PROGRESS_CLAIM_PNC`; ia menentukan kapan klaim muncul sebagai perlu ditindaklanjuti |
| **Siapa yang boleh melihat dashboard lintas cabang?** | **Work Owner** (`D-59`) | Menentukan batas data pada layar ini |

## Acceptance criteria

- [ ] Penyaring "aktif" memakai **`PROGRESSDATEDONE IS NULL`** — diuji dengan baris yang sudah
      selesai: **tidak** muncul sebagai aktif. Ini **perbaikan perilaku yang disengaja** (`P-5`).
- [ ] Seluruh angka TAT di dashboard berasal dari fungsi `TKT-S7-001` — tidak ada perhitungan
      tandingan di kueri dashboard.
- [ ] Kepala cabang **hanya melihat cabangnya** — diuji dengan dua peran cabang berbeda.
- [ ] Default follow-up **sesuai keputusan Work Owner**, dan angkanya **dikonfigurasi**, bukan
      tertanam di kode — diuji.
- [ ] Selisih angka terhadap Pega **dilaporkan sebagai perbaikan terencana**, bukan bug — dicatat
      per metrik lewat `S-8`.
- [ ] Dashboard **tidak memakai koneksi transaksi** saat memuat data berat (`TKT-F2-001`) — diuji.
- [ ] Gerbang 2: UAT **PncManagerAdmin** dan **PncKacab** secara terpisah.

## Dependency / Blocked by

`TKT-S7-001` · `TKT-S2-001` · `TKT-F3-005`. **Terhalang keputusan Work Owner.**

## Constraint keamanan, data, operasional

- Dashboard memuat data klaim lintas cabang bila batasnya tidak ditegakkan — ia jalan pintas yang
  sama seperti laporan (`TKT-S2-002`).
- **Angka outstanding akan turun tajam setelah penyaingnya diperbaiki.** Tanpa penjelasan,
  penurunan itu akan terbaca sebagai kesalahan sistem baru — padahal ia justru koreksi.
- Kolom `STATUSPOSISI` **tetap ada dan tetap bernilai `'On Progress'`** di data lama; memperbaiki
  penyaring **tidak memperbaiki kolomnya**.

## Migrasi skema / rollout / rollback

Tidak mengubah kolom; mengubah **cara membacanya**. Mungkin menambah index pada `PROGRESSDATEDONE`
(`TKT-F2-004`).

**Rollback:** kembali ke penyaring `STATUSPOSISI` **mengembalikan angka yang keliru**.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/dashboard/... -run TestAktifMemakaiProgressDateDone
go test ./internal/app/dashboard/... -run TestBatasDataCabang
go run ./cmd/s8 banding --modul S-7 --metrik semua --laporkan-selisih
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `STATUSPOSISI` selalu `'On Progress'`, nol pengecualian | `docs/verifikasi-bukti-adr.md` §14.8 |
| `GET_POSISI_PROGRESS_PNC.fnc:15` menyaring `'On Progress'` — cocok semua baris | idem |
| Penanda selesai sebenarnya `PROGRESSDATEDONE` (`:40`, `:67`) | idem |
| Default follow-up `sysdate + 7` (`:7-8`) | idem |
| Harness dashboard dan KPI | `docs/Steering/06-MODULE-BREAKDOWN.md:66` |

## Comments
