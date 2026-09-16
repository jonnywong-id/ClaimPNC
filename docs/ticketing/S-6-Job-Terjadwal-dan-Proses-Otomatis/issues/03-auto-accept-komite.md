---
title: "TKT-S6-003 — Persetujuan komite otomatis jam 06:00"
labels: [modul::S-6, tipe::keamanan, status::needs-info, prioritas::tinggi, gelombang::7]
milestone: "Gelombang 7 — Sisa"
epic: "Migrasi Claim PNC"
---

# TKT-S6-003 — Persetujuan komite otomatis jam 06:00

Status: needs-info
Kesiapan: **terhalang keputusan Work Owner — pertanyaan belum terjawab**
Modul: **S-6 Job Terjadwal & Proses Otomatis** · Gelombang: 7 · Bergantung pada: TKT-S6-001, TKT-B07-002
Requirement: FR-S6, FR-B7    Keputusan: D-57    ADR: 0022    Risiko: —
Rule Pega yang digantikan: `Job Scheduler/JOBForKomiteKlaimPNC-Job.xml` — `Daily`, **`06:00:00`**, activity target **`AutoAcceptKomite`**
Peran penguji gerbang 2: **PncManagerAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Kejelasan lebih dulu, kode belakangan: **apakah persetujuan komite otomatis memang dikehendaki.**

Nilai bisnisnya — dan alasan tiket ini berdiri sendiri — adalah ini: setiap hari jam 06:00, sebuah
job menjalankan `AutoAcceptKomite` **tanpa pengguna sama sekali**. Seluruh kendali `B-7` — jenjang
kumulatif, pita nilai, pemisahan peran — **tidak berlaku padanya**. Bahkan kontrol berbasis menu
(`D-59`) tidak menyentuhnya, karena tidak ada menu yang dibuka.

**Perilaku ini tidak tercatat di dokumen mana pun** dan belum pernah dibahas sampai ditemukan di
export (`D-57`).

## Ruang lingkup

Ditetapkan **setelah** Work Owner menjawab. Salah satu dari tiga arah:

1. **Dipertahankan apa adanya** — dipindahkan dengan perilaku sama, dan dicatat sebagai keputusan
   sadar beserta alasannya.
2. **Dipertahankan dengan batas** — misalnya hanya untuk klaim di bawah nilai tertentu, atau hanya
   untuk klaim yang sudah menunggu sekian lama.
3. **Dihentikan** — klaim menunggu persetujuan manusia, dan dampaknya pada antrean diukur lebih dulu.

## Non-goal

- **Tidak** memilih salah satu dari ketiganya sendiri. Ini keputusan bisnis dengan dampak finansial,
  dan ia bukan milik saya.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **`AutoAcceptKomite` berjalan tiap hari 06:00 — benar demikian?** | **Work Owner** (`D-57` butir 1) | Tanpa jawaban, tiket ini tidak punya lingkup sama sekali. Menebaknya berarti memutuskan sendiri apakah klaim boleh disetujui tanpa manusia |
| **Bila dipertahankan: klaim mana yang layak disetujui otomatis?** | **Work Owner** | Menentukan penyaringnya. Tanpa itu, "otomatis" berarti **semua** |
| **Berapa banyak klaim yang benar-benar disetujui job ini per hari?** | **Work Owner + Tim Pega** | Angka ini menentukan apakah menghentikannya akan menumpuk antrean komite. Ia **tidak dapat diukur dari export** — hanya dari data produksi |
| **Apakah jejak audit Pega mencatat persetujuan ini sebagai dilakukan siapa?** | **Tim Pega** | Bila tercatat atas nama pengguna tertentu, jejak audit selama ini **menyesatkan** |

## Acceptance criteria

> **Belum dapat dituliskan.** Kriteria kelulusan tiket ini bergantung sepenuhnya pada arah yang
> dipilih Work Owner. Menuliskan daftar bercentang sekarang akan membuat tiket ini tampak siap
> dikerjakan padahal pertanyaan pokoknya belum terjawab.

Yang **sudah pasti berlaku**, apa pun arahnya:

- [ ] Setiap persetujuan otomatis **tercatat di jejak audit sebagai dilakukan oleh job**, bukan
      atas nama pengguna mana pun (`S-5`).
- [ ] Perilaku yang dipilih **tercatat sebagai keputusan di Decision Log** beserta alasannya —
      sehingga tidak ada lagi jalur persetujuan yang berjalan tanpa ada yang mengetahuinya.

## Dependency / Blocked by

`TKT-S6-001` · `TKT-B07-002` (aturan jenjang komite) · `TKT-S5-001` (jejak audit).
**Terhalang jawaban Work Owner atas `D-57` butir 1.**

## Constraint keamanan, data, operasional

- **Ini jalur persetujuan klaim yang tidak melewati kontrol mana pun.** Ia sejenis dengan
  `TKT-S4-002` (persetujuan komite lewat API): keduanya memintas `B-7`, dan keduanya baru ditemukan
  di fase verifikasi. Bila `B-7` dibangun dengan cermat sementara kedua jalur ini dibiarkan,
  kecermatan itu tidak ada artinya.
- Menghentikannya **menggeser beban ke komite manusia**; mempertahankannya **menyalin jalur tanpa
  kendali ke sistem baru**. Keduanya punya harga, dan keduanya harus dipilih dengan sadar.

## Migrasi skema / rollout / rollback

Ditetapkan setelah arahnya dipilih.

**Yang sudah dapat dikatakan:** job ini **tidak boleh aktif di Pega dan Go bersamaan** — dua
persetujuan otomatis atas klaim yang sama.

## Rencana verifikasi

> **Rencana — belum dapat disusun.** Ia mengikuti arah yang dipilih.

Yang pasti diperlukan apa pun arahnya:

```bash
go test ./internal/app/audit/... -run TestPersetujuanOtomatisTercatatSebagaiJob
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `JOBForKomiteKlaimPNC` harian 06:00 → `AutoAcceptKomite` | `D-57` tabel · `Job Scheduler/JOBForKomiteKlaimPNC-Job.xml` |
| Perilaku ini tidak tercatat di dokumen mana pun dan belum pernah dibahas | `D-57` butir 1 |
| Job melewati bahkan kontrol berbasis menu | `docs/Steering/00-DECISION-LOG.md:1634` · `D-59` |
| Jenjang komite kumulatif; pita nilai hanya Non-MBU | `D-52` · `D-70` |

## Comments
