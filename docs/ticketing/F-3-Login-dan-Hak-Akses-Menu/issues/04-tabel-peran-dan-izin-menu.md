---
title: "TKT-F3-004 — Tabel 22 peran dan 51 izin menu"
labels: [modul::F-3, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F3-004 — Tabel 22 peran dan 51 izin menu

Status: needs-info
Kesiapan: **terhalang artefak** — tabel dapat dibangun, **tidak dapat diisi**
Modul: F-3 · Gelombang: 1 · Bergantung pada: TKT-F2-001, TKT-F2-004
Requirement: FR-F3, FR-R1    Keputusan: D-58, D-59    ADR: 0023    Risiko: R-16
Rule Pega yang digantikan: **22 access group** `GCNMFW:<nama>` · **34 When rule** yang memuat pemetaan peran→menu · `Navigation/pyCaseWorkerNavigation-Navigation.xml` (51 item menu) · `POOLDATA.T_ACCESS_GROUP_PNC`
Peran penguji gerbang 2: **tidak berlaku** — `D-60`; gerbang 1 diganti uji kontrak (`D-56`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Peran dan izin menjadi **data di tabel**, bukan logika yang tersebar di 34 When rule.

Nilai bisnisnya: hari ini, menjawab *"siapa saja yang boleh menyetujui komite"* menuntut membaca
34 rule satu per satu. Setelah tiket ini, jawabannya adalah satu kueri — dan itu prasyarat bagi
audit mana pun.

## Ruang lingkup

- Tabel: Peran (**22 baris**), Izin menu (**51 item**), PeranIzin, PenggunaPeran, dan
  BatasDataPengguna.
- **Normalisasi kapitalisasi**: tiga nama muncul dalam dua bentuk — `ViewClaimPNC`/`VIEWCLAIMPNC`
  dan `PncReceive`/`PNCRECEIVE`. Sistem baru wajib menjadikannya **satu identitas per peran**.
- Penamaan ulang 22 access group menjadi nama yang terbaca manusia, **satu-untuk-satu** — tanpa
  penggabungan dan tanpa pemecahan (`D-58`).
- Batas data: cabang, lini bisnis/Group Panel, dan organisasi (`pyOrgUnit != "Eksternal"`).

## Non-goal

- **Tidak** membuat izin berbutir aksi. `D-59` menetapkan **satuan izin adalah menu**; membuat
  izin per tindakan berarti merancang pemisahan tugas yang tidak diminta.
- **Tidak** menegakkan izin — itu `TKT-F3-005`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Daftar operator per peran.** `POOLDATA.T_ACCESS_GROUP_PNC` hanya memetakan `OPERATOR_ID` → `OLD_OPERATOR_ID`; penugasan operator ke access group **tidak ada di database** | **Work Owner + DBA** | Tabel dapat dibangun tetapi **tidak dapat diisi**. Aplikasi yang tabel perannya kosong tidak dapat dipakai siapa pun |
| **5 When rule peran hilang dari export**: `IsGCNMReport`, `IsKomite`, `IsNotViewClaim`, `IsPNCBonding`, `IsSurvey` — masing-masing mengendalikan satu item menu | **Tim Pega** (`R-16`) | Peta peran→menu tidak lengkap; 5 dari 51 item menu tidak diketahui siapa yang boleh membukanya |
| **Tidak ada folder Access Group / Role / Privilege / Operator sama sekali** di export | **Tim Pega** | Definisi access group hanya terbaca dari **pemakaiannya** di rule, bukan dari definisinya |
| Apakah pemetaan 22 access group → 51 menu **masih akurat**? · `T_ACCESS_GROUP_PNC` sebenarnya untuk apa? · pernah ada temuan audit soal **901 dari 902 activity tanpa privilege**? | **Work Owner** | Menentukan apakah peta yang direkonstruksi dipakai apa adanya atau perlu ditinjau bisnis |

## Acceptance criteria

> Butir yang menuntut isi data **belum dapat diangkakan** sampai daftar operator diterima.

- [ ] Tabel peran memuat **tepat 22 baris**, namanya terbaca manusia, dan pemetaannya ke nama
      access group lama tercatat — sehingga migrasi dapat ditelusuri.
- [ ] Ketiga nama yang berbeda kapitalisasi **menjadi satu baris**, bukan dua — diuji.
- [ ] Tabel izin menu memuat **51 item**, masing-masing dengan penanda yang dipakai `TKT-F3-005`.
- [ ] Pemetaan peran→menu yang direkonstruksi dari 34 When rule tersedia sebagai berkas dengan
      `berkas:baris` untuk setiap baris pemetaan.
- [ ] **5 item menu yang When rule-nya hilang ditandai eksplisit** sebagai belum diketahui —
      bukan diisi dengan tebakan.
- [ ] Batas data (cabang, lini bisnis, organisasi) tersimpan sebagai data dan dapat diuji per
      pengguna.

## Dependency / Blocked by

Bergantung pada `TKT-F2-001`, `TKT-F2-004`. **Terhalang Tim Pega dan Work Owner.**

## Constraint keamanan, data, operasional

- **Satuan izin adalah menu** (`D-59`). Konsekuensinya diterima sadar: orang yang sama dapat
  membuat, menyetujui, dan membayarkan satu klaim bila perannya memiliki ketiga menu itu.
- Perubahan pada tabel peran dan izin adalah **tindakan bernilai tinggi** — ia mengubah kewenangan.
  Setiap perubahannya wajib tercatat di jejak audit (`S-5`).
- Tabel ini memuat **nama pegawai**; ia tunduk aturan penulisan `D-69` bila dikutip ke dokumen.

## Migrasi skema / rollout / rollback

Menambah lima tabel baru — **tidak menyentuh tabel yang dibaca Pega**.

**Rollback:** tabel dibiarkan ada dan tidak dipakai. Namun **rollback setelah pengguna nyata
ditugaskan ke peran akan mencabut akses mereka** — karena itu pengisian data dilakukan setelah
tabelnya stabil, bukan bersamaan.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/sqlstore/... -run TestPeranDanIzin
go run ./cmd/tools/cek-peran --harap 22        # HARUS 22 baris
go run ./cmd/tools/cek-menu --harap 51         # HARUS 51 item
go run ./cmd/tools/cek-menu --tanpa-when-rule  # HARUS melaporkan tepat 5
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 22 access group, seluruhnya terverifikasi sebagai literal `GCNMFW:<nama>` | `D-58` · `docs/Steering/11-SECURITY.md` §3.1 |
| 51 item menu dipetakan lewat 34 When rule + `Navigation/pyCaseWorkerNavigation-Navigation.xml` | idem |
| 5 When rule peran hilang | `D-58` · `R-16` |
| Kapitalisasi tidak konsisten pada 3 nama | `D-58` |
| `T_ACCESS_GROUP_PNC` hanya `OPERATOR_ID` → `OLD_OPERATOR_ID` | `D-58` · `ADR-0023` |
| Satuan izin adalah menu, tanpa pemisahan tugas | `D-59` · `ADR-0023` |

## Comments
