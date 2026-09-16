---
title: "TKT-B07-002 — Layar Komite dan pencatatan keputusan"
labels: [modul::B-7, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::4]
milestone: "Gelombang 4 — Persetujuan"
epic: "Migrasi Claim PNC"
---

# TKT-B07-002 — Layar Komite dan pencatatan keputusan

Status: needs-info
Kesiapan: **terhalang artefak** — Ticket rule dan When rule komite
Modul: **B-7 Komite** · Gelombang: 4 · Bergantung pada: TKT-B07-001, TKT-B06-001
Requirement: FR-B7    Keputusan: D-26, D-59    ADR: 0019, 0021, 0023, 0026    Risiko: R-16
Rule Pega yang digantikan: `Flow/Komite_Flow.xml` · harness `InboxKomite_Harness` · Ticket rule `KomiteAssign_ticket` (`:974`) dan `komiteAccept_ticket` (`:796`) — **keduanya hilang dari export** · When rule `IsKomite` — **hilang**
Peran penguji gerbang 2: **PNCKomite** dan **PNCKomiteTeknik**

## Hasil yang diharapkan (dan nilai bisnisnya)

Anggota komite melihat klaim yang menunggu persetujuannya, memberi keputusan, dan keputusan itu
**tercatat permanen** beserta siapa dan kapan.

Nilai bisnisnya: persetujuan komite adalah **titik kewenangan tertinggi atas uang klaim**. Dan
karena `D-59` menghapus pemisahan tugas, pencatatan keputusan ini adalah **satu-satunya bukti**
bahwa persetujuan benar-benar diberikan orang yang berwenang.

## Ruang lingkup

- Inbox komite per anggota, menampilkan klaim yang menunggu jenjangnya.
- Tiga keputusan: **setuju**, **tolak**, **kembalikan** — masing-masing dengan catatan.
- Pencatatan jejak komite: siapa, kapan, keputusan apa, catatan apa, pada jenjang ke berapa.
- Perpindahan ke jenjang berikutnya setelah satu jenjang menyetujui; klaim **tidak dapat
  diakseptasi** sebelum seluruh jenjang selesai (invarian `I-5`).
- **Lompatan lateral** saat seluruh komite menyetujui — kembali ke petugas estimasi untuk lini PA.

## Non-goal

- **Tidak** menghitung jumlah jenjang — itu `TKT-B07-001`.
- **Tidak** menangani persetujuan otomatis — itu `TKT-B07-003`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Ticket rule `KomiteAssign_ticket` dan `komiteAccept_ticket`** — dirujuk `Flow/Komite_Flow.xml:974` dan `:796`, **tidak ada di export** | **Tim Pega** (`R-16`) | Menentukan **kapan** klaim melompat masuk dan keluar dari komite |
| **When rule `IsKomite`** — hilang | **Tim Pega** | Menentukan syarat sebuah klaim masuk komite sama sekali |
| **`KOMITEKE` immutable?** | **Work Owner** | Menentukan apakah jenjang yang sudah dilewati dapat diulang |
| **Siapa yang berwenang memicu lompatan lateral?** Sistem lama **nol pagar izin** | **Work Owner** | `D-59` bersatuan menu; lompatan melewati alur normal |

## Acceptance criteria

- [ ] Anggota komite hanya melihat klaim yang **menunggu jenjangnya** — diuji dengan tiga anggota
      pada jenjang berbeda.
- [ ] Ketiga keputusan tersedia dan masing-masing menghasilkan **tepat satu baris jejak komite**
      berisi pelaku, waktu, keputusan, catatan, dan nomor jenjang.
- [ ] Klaim **tidak dapat diakseptasi** sebelum seluruh jenjang menyetujui — diuji dengan mencoba
      akseptasi di tengah jalan: **ditolak** (invarian `I-5`).
- [ ] Keputusan **tolak** dan **kembalikan** memindahkan klaim ke tahap yang benar — diuji.
- [ ] Setiap keputusan tercatat di **jejak audit** (`S-5`), terpisah dari jejak komite —
      keduanya diperiksa.
- [ ] Anggota tanpa izin menu komite menerima `403` dan **tidak melihat menunya**.
- [ ] Gerbang 1: urutan dan hasil keputusan **sama dengan Pega** pada 20 klaim contoh.
- [ ] Gerbang 2: UAT **PNCKomite** dan **PNCKomiteTeknik**.

## Dependency / Blocked by

`TKT-B07-001` · `TKT-B06-001` · `TKT-S5-002`. **Terhalang Tim Pega dan dua keputusan.**

## Constraint keamanan, data, operasional

- **Orang yang sama dapat mengubah nilai klaim dan menyetujuinya** bila perannya memiliki kedua
  menu (`D-59`). Tidak ada kontrol teknis yang mencegahnya; jejak audit satu-satunya pengimbang.
- Penjenjangan komite menugaskan ke **operator bernama**, bukan ke workbasket (`T-7`) — sehingga
  **ketidakhadiran seseorang dapat menghentikan klaim**. Kolom `STS_ABS` pada master tampaknya
  menjawab ini, dan perilakunya belum dirumuskan.
- Keputusan komite **tidak dapat dihapus** (`ADR-0012`) — hanya ditandai, tidak pernah dibuang.

## Migrasi skema / rollout / rollback

Menambah tabel jejak komite. Backward-compatible.

**Rollback:** keputusan yang sudah tercatat tetap ada dan tetap sah.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/komite/... -run TestInboxPerJenjang
go test ./internal/app/komite/... -run TestAkseptasiDitolakSebelumSeluruhJenjang
go test ./internal/app/komite/... -run TestKeputusanTercatat
go run ./cmd/s8 banding --modul B-7 --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Ticket rule komite hilang | `Flow/Komite_Flow.xml:796`, `:974` · `ADR-0021` |
| Invarian `I-5` akseptasi setelah seluruh jenjang | `docs/Steering/05-DOMAIN-MODEL.md` §2 |
| Komite menugaskan ke operator bernama; hanya 4 workbasket | `T-7` |
| Tidak ada pemisahan tugas | `D-59` · `ADR-0023` |
| Nol pagar izin pada transisi lateral | `docs/verifikasi-bukti-adr.md` §15 |

## Comments
