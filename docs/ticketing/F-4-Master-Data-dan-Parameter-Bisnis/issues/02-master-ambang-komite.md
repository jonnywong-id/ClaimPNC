---
title: "TKT-F4-002 — Master Ambang Komite dan validasi integritas tangga"
labels: [modul::F-4, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F4-002 — Master Ambang Komite dan validasi integritas tangga

Status: needs-info
Kesiapan: **terhalang keputusan** — struktur pengganti `EMAILKOMITE` belum ditetapkan
Modul: F-4 · Gelombang: 1 · Bergantung pada: TKT-F4-001
Requirement: FR-F4    Keputusan: D-14, D-47, D-52, D-70    ADR: 0014    Risiko: R-19
Rule Pega yang digantikan: `POOLDATA.EMAILKOMITE` (**21 kolom, 30 baris**) dan logika pemilihan yang tersebar di `Activity/SetEmailKomite-Act.xml`, `SetEmailKomiteAdjuster-Act.xml`, `SetEmailKomiteSalvage-Act.xml`, `SetEmailKomiteSimasnet-Act.xml` — ditambah **17 kueri** yang membacanya
Peran penguji gerbang 2: **PNCKomite** dan **PNCKomiteTeknik** (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Tangga jenjang komite menjadi **master data yang dapat divalidasi**, menggantikan aturan yang
tersebar di empat activity dan bercampur dengan **nama orang**.

Nilai bisnisnya: master ini menentukan **siapa berwenang menyetujui uang**. Satu baris yang salah
mengubah kewenangan persetujuan **tanpa perubahan kode apa pun** — itulah sebabnya validasi
integritasnya menjadi bagian tiket ini, bukan tambahan.

## Ruang lingkup

- Master ambang komite berisi, per lini bisnis: **ambang bawah**, urutan jenjang (`DEGREE`),
  penanda aktif, penanda jenis komite, dan penyetuju.
- **Validasi integritas tangga** memakai batas atas: menolak master yang rentangnya **tumpang
  tindih** atau **berlubang** antar `DEGREE` dalam satu lini + pita.
- Penghapusan aturan berbasis **nama orang** — sistem lama memaksa nilai pembanding melewati ambang
  agar jenjang tertentu ikut atau tidak ikut terpilih, dengan syarat siapa PIC Teknis-nya.
- Migrasi isi `emailkomite.csv` yang sudah diterima ke bentuk master baru.

## Non-goal

- **Tidak** membangun alur komite — itu `B-7`.
- **Tidak** mengubah model penjenjangan. Model **kumulatif** sudah ditetapkan `ADR-0014`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Apakah struktur `EMAILKOMITE` dipertahankan apa adanya**, atau dipecah? Kolom `TYPE_KOMITE` **memikul dua arti berbeda** — pita nilai di Non-MBU, varian jalur (PA reguler versus PA TKI) di lini PA | **Work Owner** | Menentukan bentuk tabel. `ADR-0014` memutuskan **tidak** memecahnya sekarang, tetapi keputusan itu diambil untuk perilaku, bukan untuk bentuk master baru |
| **PA dan Travel di atas Rp 200.000.000 tidak punya baris master** | **Work Owner** | Klaim di atas nilai itu tidak punya penyetuju — dan itu bukan sesuatu yang boleh ditebak |

**Yang sudah tidak perlu ditanyakan lagi:** aturan berbasis nama orang **dicabut** (`D-52`), dan
model kumulatif beserta cakupan pita per lini sudah ditetapkan (`D-47`, `D-70`).

## Acceptance criteria

- [ ] Master menolak disimpan bila ada **lubang** antar `DEGREE` dalam satu lini + pita — diuji
      dengan master percobaan yang sengaja berlubang.
- [ ] Master menolak disimpan bila ada **tumpang tindih** rentang — diuji.
- [ ] Validasi dijalankan **per lini**, bukan lintas lini — diuji dengan PA, yang tangganya memang
      tidak tersusun menurut pita.
- [ ] Jumlah jenjang untuk sebuah nilai klaim dihitung **kumulatif** dan hasilnya sama dengan
      tabel berikut — diuji ketujuh kasus:

      | Kasus | Jumlah penyetuju |
      |---|---|
      | PA Rp 5.000.000 | 1 |
      | PA Rp 75.000.000 | 3 |
      | PA Rp 150.000.000 | 4 |
      | Travel Rp 150.000.000 | 3 |
      | Non-MBU Rp 80.000.000 | 2 |
      | Non-MBU Rp 750.000.000 | 2 |
      | Non-MBU Rp 2.000.000.000 | 3 |

- [ ] **Nol aturan berbasis nama orang** di kode maupun master — diuji pemindaian terhadap 24
      Operator ID yang diketahui.
- [ ] Perubahan master tercatat di jejak audit (`TKT-F4-001`).

## Dependency / Blocked by

Bergantung pada `TKT-F4-001`. **Terhalang dua keputusan** di atas.

**Yang bergantung padanya:** `B-7` Komite dan `B-12` Salvage.

## Constraint keamanan, data, operasional

- **Perubahan master ini mengubah kewenangan menyetujui uang.** Ia kandidat terkuat untuk alur
  persetujuan bila `TKT-F4-001` memutuskan memakainya.
- Master memuat alamat email penyetuju — tunduk `D-67` (mailbox fungsional, bukan akun pribadi)
  dan `D-69` (email disamarkan bila dikutip ke dokumen).
- **Belum terverifikasi:** tiga kueri yang memfilter `TYPE_KOMITE` secara dinamis —
  `EmailKomiteBerjenjang_sql`, `EmailKomiteAdjuster_sql`, `EmailKomiteSalvage_sql` — menerima
  nilainya dari pemanggil lewat `tempAdj.pyMemo` dan `tempAdj.AcceptedNo`. **Lini apa saja yang
  melewati ketiga kueri itu belum ditelusuri sampai ke sumber nilainya.** Ini bagian Definition of
  Ready `B-7` dan `B-12`.

## Migrasi skema / rollout / rollback

Menambah tabel master baru; **`POOLDATA.EMAILKOMITE` tetap dibaca Pega** selama masa paralel dan
**tidak boleh diubah** (`P-1` — Pega adalah penulisnya sampai `B-7` pindah).

**Rollback:** aplikasi Go kembali membaca `EMAILKOMITE` langsung. Master baru dibiarkan.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/komite/... -run TestJumlahJenjangKumulatif
go test ./internal/app/masterdata/... -run TestValidasiTanggaBerlubang
go test ./internal/app/masterdata/... -run TestValidasiTanggaTumpangTindih
go run ./cmd/tools/cek-hardcode-operator internal/   # HARUS 0 temuan
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `EMAILKOMITE` 21 kolom, 30 baris | `D-14` · `Database/emailkomite.csv` |
| Model kumulatif; `LIMIT_BOTTOM` di 11 kueri, `LIMIT_TOP` di 0 | `D-47` · `ADR-0014` |
| Pita nilai hanya di Non-MBU | `D-70` |
| Aturan berbasis nama orang dicabut | `D-52` · `docs/Steering/20-DETAIL-KOMITE-DBLINK.md` §1.5 |
| PA dan Travel di atas Rp 200 Jt tanpa baris master | `D-55` · `ADR-0014` |
| Tujuh kasus uji jumlah penyetuju | `ADR-0014` Rationale |

## Comments
