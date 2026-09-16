---
title: "TKT-B07-001 — Penentuan jenjang komite kumulatif"
labels: [modul::B-7, tipe::aturan-bisnis, status::needs-info, prioritas::tinggi, gelombang::4]
milestone: "Gelombang 4 — Persetujuan"
epic: "Migrasi Claim PNC"
---

# TKT-B07-001 — Penentuan jenjang komite kumulatif

Status: needs-info
Kesiapan: terhalang keputusan dan tiket `TKT-F4-002`
Modul: **B-7 Komite** · Gelombang: 4 · Bergantung pada: TKT-B05-002, TKT-F4-002
Requirement: FR-B7    Keputusan: D-14, D-47, D-52, D-70    ADR: 0014    Risiko: R-19
Rule Pega yang digantikan: `Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459` (`KomiteLoop := pxResultCount`) · `When/IsKomiteLoop-When.xml` · **17 kueri** pembaca `POOLDATA.EMAILKOMITE` · `Activity/SetEmailKomite-Act.xml` beserta tiga varian
Peran penguji gerbang 2: **PNCKomite** dan **PNCKomiteTeknik**

## Hasil yang diharapkan (dan nilai bisnisnya)

Untuk sebuah nilai klaim, sistem menetapkan **berapa banyak jenjang yang harus menyetujui** dan
**siapa saja penyetujunya** — dengan hasil yang sama setiap kali dihitung.

Nilai bisnisnya: ini menentukan **siapa berwenang menyetujui uang**. Salah hitung ke bawah berarti
klaim besar disetujui terlalu sedikit orang; salah ke atas berarti klaim kecil tertahan tanpa
alasan.

## Ruang lingkup

- Perhitungan **kumulatif**: seluruh jenjang yang **ambang bawahnya sudah terlampaui** ikut
  menyetujui.
- Untuk lini **Non-MBU**: pemilihan **pita nilai** lebih dulu (≤ Rp 100 juta → pita `1`; di atasnya
  → pita `2`), lalu akumulasi **di dalam pita itu saja**.
- Untuk lini lain: **tanpa langkah pita** — akumulasi langsung atas seluruh jenjang lini tersebut.
- Nilai yang dibandingkan adalah hasil konversi kurs `TKT-B05-002`, memakai **presisi penuh**.
- Urutan penyetuju mengikuti `DEGREE`.

## Non-goal

- **Tidak** memakai `LIMIT_TOP` untuk memilih baris. `LIMIT_TOP` adalah **validasi integritas
  master** (`TKT-F4-002`), bukan penyaring.
- **Tidak** membangun layar komite — itu `TKT-B07-002`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **PA dan Travel di atas Rp 200.000.000 tidak punya baris master** | **Work Owner** | Klaim di atas nilai itu **tidak punya penyetuju sama sekali** — dan itu tidak boleh ditebak |
| **Baris `DEGREE=0` maksudnya apa?** | **Work Owner** | Muncul di master tetapi tidak jelas apakah ia jenjang, penanda, atau sisa data |
| **`dbms_random.value` pada 2 kueri Simasnet disengaja?** | **Work Owner** | Bila ya, **acceptance criteria untuk jalur Simasnet tidak boleh deterministik** — dan itu mengubah cara modul ini diuji |
| **Belum ditelusuri:** lini apa saja yang melewati `EmailKomiteBerjenjang_sql`, `EmailKomiteAdjuster_sql`, `EmailKomiteSalvage_sql` — ketiganya menerima `TYPE_KOMITE` dari pemanggil lewat `tempAdj.pyMemo` dan `tempAdj.AcceptedNo` | **Lead Engineer** | Menentukan lini mana yang memakai langkah pita. Ini **Definition of Ready** tiket ini |

## Acceptance criteria

- [ ] Jumlah penyetuju untuk **ketujuh kasus** pada `spec.md` sesuai tabel — diuji satu per satu.
- [ ] Untuk **Non-MBU**, pita dipilih lebih dulu dan akumulasi **tidak menyeberang antarpita** —
      diuji dengan Rp 80 juta (pita 1, 2 penyetuju) dan Rp 200 juta (pita 2, 1 penyetuju).
- [ ] Untuk **PA dan Travel**, **tidak ada langkah pita** — diuji: PA Rp 150 juta menghasilkan
      **4 penyetuju**, bukan 2.
- [ ] `LIMIT_TOP` **tidak dipakai menyaring baris** — diuji pemindaian kueri.
- [ ] Perbandingan memakai **nilai presisi penuh** hasil konversi kurs — diuji.
- [ ] Urutan penyetuju mengikuti `DEGREE` — diuji.
- [ ] Gerbang 1: jumlah dan identitas penyetuju **sama dengan Pega** pada 30 klaim contoh dari
      empat lini. Untuk jalur Simasnet, kriteria menyesuaikan jawaban `dbms_random`.
- [ ] Gerbang 2: UAT **PNCKomite** dan **PNCKomiteTeknik**.

## Dependency / Blocked by

`TKT-B05-002` (nilai terkonversi) · `TKT-F4-002` (master ambang). **Terhalang tiga keputusan dan
satu penelusuran.**

## Constraint keamanan, data, operasional

- **Menerapkan filter pita secara seragam ke semua lini akan merusak.** Dihitung dari master: PA
  Rp 5 juta menjadi **0 penyetuju** dan Travel Rp 150 juta menjadi **0 penyetuju** — klaim mandek.
  Itu sebabnya `D-70` membatasi pita **hanya** ke Non-MBU.
- **Menerapkan rentang tertutup** (`LIMIT_BOTTOM <= nilai <= LIMIT_TOP`) akan mengembalikan tepat
  satu baris → **satu jenjang berapa pun nilai klaim** → menghapus penjenjangan (`D-47`).
- Kolom `TYPE_KOMITE` **memikul dua arti**: pita nilai di Non-MBU, varian jalur di PA. Model data
  wajib mendokumentasikannya.

## Migrasi skema / rollout / rollback

Tidak menambah tabel; membaca master `TKT-F4-002`.

**Rollback:** kembali membaca `POOLDATA.EMAILKOMITE` langsung.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/komite/... -run TestJenjangKumulatif   # 7 kasus pada spec
go test ./internal/domain/komite/... -run TestPitaHanyaNonMBU
go test ./internal/domain/komite/... -run TestLimitTopTidakMenyaring
go run ./cmd/s8 banding --modul B-7 --aturan jenjang --kasus 30
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `KomiteLoop := pxResultCount` | `Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459` |
| `KomiteCount <= KomiteLoop` | `When/IsKomiteLoop-When.xml` |
| `LIMIT_BOTTOM` di 11 rule, `LIMIT_TOP` di 0 | `ADR-0014` |
| Pita hanya Non-MBU | `D-70` |
| Tujuh kasus jumlah penyetuju | `ADR-0014` Rationale |
| Tiga kueri dinamis belum ditelusuri | `ADR-0014` Pertanyaan terbuka |

## Comments
