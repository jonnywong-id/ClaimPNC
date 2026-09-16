---
title: "TKT-B07-003 — Persetujuan otomatis AutoAcceptKomite"
labels: [modul::B-7, tipe::aturan-bisnis, status::needs-info, prioritas::tinggi, gelombang::4]
milestone: "Gelombang 4 — Persetujuan"
epic: "Migrasi Claim PNC"
---

# TKT-B07-003 — Persetujuan otomatis `AutoAcceptKomite`

Status: needs-info
Kesiapan: **terhalang keputusan** — perilaku ini belum pernah dibahas di dokumen mana pun
Modul: **B-7 Komite** · Gelombang: 4 · Bergantung pada: TKT-B07-002, TKT-S6-001
Requirement: FR-B7, FR-S6    Keputusan: D-57, D-59    ADR: 0022, 0023, 0026    Risiko: —
Rule Pega yang digantikan: `Job Scheduler/JOBForKomiteKlaimPNC` — **harian, jam 06:00**, menjalankan activity **`AutoAcceptKomite`**
Peran penguji gerbang 2: **PNCKomite** dan **PncManagerAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Kejelasan tentang satu proses yang berjalan setiap hari dan **menyetujui komite tanpa pengguna
sama sekali**.

Nilai bisnisnya bukan fitur baru, melainkan **kesadaran**. Perilaku ini ditemukan saat folder
`Job Scheduler/` diterima, dan ia **tidak tercatat di dokumen mana pun** sebelumnya. Sebuah proses
yang menyetujui uang klaim secara otomatis pada jam 06:00 pantas diputuskan secara sadar, bukan
diwarisi begitu saja.

## Ruang lingkup

- Penerapan kembali `AutoAcceptKomite` **bila Work Owner memutuskan perilaku ini dipertahankan**.
- Bila dipertahankan: pencatatan **jejak audit dengan pelaku "sistem"**, bukan kosong — agar
  persetujuan otomatis tetap dapat dipertanggungjawabkan.
- Bila tidak dipertahankan: klaim yang sebelumnya disetujui otomatis akan **menumpuk di inbox
  komite**, dan dampak operasionalnya harus disiapkan.
- Batasan: jenjang mana yang boleh disetujui otomatis, dan pada nilai berapa.

## Non-goal

- **Tidak** memutuskan sendiri apakah perilaku ini benar. Itu keputusan Work Owner.
- **Tidak** membangun mekanisme penjadwal — itu `S-6`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **`AutoAcceptKomite` harian jam 06:00 — perilaku yang benar?** | **Work Owner** | Ia **menyetujui komite tanpa pengguna**, sehingga **melewati kontrol menu `D-59` sepenuhnya**. Bila dipertahankan, ia adalah satu-satunya jalur yang tidak tunduk model izin |
| Jenjang dan nilai mana yang boleh disetujui otomatis | **Work Owner** | Tanpa batas, persetujuan otomatis berlaku untuk klaim nilai berapa pun |
| **Bila dua instans aplikasi berjalan, siapa yang menjalankan job ini?** | **Lead Engineer + Infra** (`ADR-0022`) | Penguncian yang salah membuat job berjalan **dua kali** — dan dua kali persetujuan komite pada klaim yang sama adalah kesalahan bernilai uang |

## Acceptance criteria

> Belum dapat diangkakan sampai perilaku diputuskan. Bentuk AC yang akan diisi:

- [ ] Bila dipertahankan: klaim yang memenuhi syarat disetujui otomatis pada jadwal yang
      ditetapkan, dan **setiap persetujuan menghasilkan baris jejak audit dengan pelaku "sistem"**.
- [ ] Job **tidak pernah berjalan dua kali** untuk hari yang sama, walau dua instans aplikasi
      hidup — diuji dengan dua instans bersamaan (`ADR-0022`).
- [ ] Klaim di luar batas jenjang dan nilai yang ditetapkan **tidak** disetujui otomatis — diuji.
- [ ] Bila **tidak** dipertahankan: klaim yang dulu disetujui otomatis **muncul di inbox komite**,
      dan jumlahnya dilaporkan sebagai angka sebelum rilis — agar beban barunya diketahui.
- [ ] Kegagalan job **terlihat** — tercatat dan dapat diketahui, bukan gagal diam-diam.

## Dependency / Blocked by

`TKT-B07-002` · `TKT-S6-001` (mekanisme penjadwal). **Terhalang tiga keputusan.**

## Constraint keamanan, data, operasional

- **Job ini melewati seluruh kontrol otorisasi.** `D-59` menetapkan izin bersatuan menu; job tidak
  punya pengguna, sehingga tidak ada menu yang dapat diperiksa. Ini **satu-satunya jalur** dengan
  sifat demikian, dan itulah alasan ia pantas diputuskan secara eksplisit.
- **Jejak audit adalah satu-satunya kontrol** yang tersisa untuk jalur ini — dan pelakunya harus
  tercatat sebagai "sistem", bukan dibiarkan kosong.
- `pyBypassActivityAuthentication=true` pada agent terkait juga menunggu keputusan (`ADR-0022`).

## Migrasi skema / rollout / rollback

Tidak menambah tabel.

**Rollback:** mematikan job. Klaim yang telanjur disetujui otomatis **tetap disetujui** — itu tidak
dapat ditarik, dan karena itu keputusan mengaktifkannya harus diambil sebelum rilis, bukan
sesudahnya.

## Rencana verifikasi

> **Rencana — belum dijalankan, dan belum dapat disusun lengkap.**

```bash
go test ./internal/app/komite/... -run TestAutoAcceptBatasJenjang
go test ./internal/app/jadwal/... -run TestJobTidakBerjalanDuaKali
go run ./cmd/tools/hitung-klaim-auto-accept --periode 90d   # beban bila fitur dihapus
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `JOBForKomiteKlaimPNC` harian 06:00 menjalankan `AutoAcceptKomite` | `D-57` · `Job Scheduler/` |
| Perilaku ini tidak tercatat di dokumen mana pun sebelumnya | `D-57` |
| Job melewati kontrol menu | `D-59` · `ADR-0023` |
| Mekanisme penjadwal dua instans belum diputuskan | `ADR-0022` `Proposed` |

## Comments
