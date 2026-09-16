---
title: "TKT-U3-001 — Layar inbox baku dan penyaringnya"
labels: [modul::U-3, tipe::fondasi, status::needs-info, prioritas::tinggi, gelombang::4]
milestone: "Gelombang 4 — Inbox dan penugasan"
epic: "Migrasi Claim PNC"
---

# TKT-U3-001 — Layar inbox baku dan penyaringnya

Status: needs-info
Kesiapan: **terhalang keputusan rancangan**
Modul: **U-3 Layar Inbox per Peran** · Gelombang: 4 · Bergantung pada: TKT-U2-001, TKT-B06-002
Requirement: FR-U3    Keputusan: D-26    ADR: 0012    Risiko: —
Rule Pega yang digantikan: pola bersama **26 harness Inbox** — antara lain `InboxRegister_Harness`, `InboxSurvey_Harness`, `UserInbox_Harness`
Peran penguji gerbang 2: **PncRegister**, **PncSurveyor**, **PncAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu layar inbox yang benar, dipakai ulang oleh seluruh peran.

Nilai bisnisnya sama dengan alasan `U-2` menjadi investasi terpenting di frontend: bila tiap inbox
dibuat sendiri-sendiri, ada dua puluhan implementasi yang berbeda perilakunya — dan memperbaiki satu
bug menuntut dua puluh perbaikan. Membuat satu yang benar berarti perbaikan di satu tempat
memperbaiki semuanya.

## Ruang lingkup

- Layar inbox baku di atas komponen tabel `TKT-U2-001`: **paginasi dari server**, penyortiran,
  penyaringan.
- Kolom dan penyaring **didefinisikan per peran sebagai konfigurasi**, bukan sebagai layar terpisah.
- Penandaan pekerjaan yang **sudah lewat tenggat**, karena itu yang dicari petugas lebih dulu.
- Tindakan dari inbox: membuka klaim, dan mengambil pekerjaan dari workbasket (`B-6`).

## Non-goal

- **Tidak** memetakan inbox mana untuk peran mana — itu `TKT-U3-002`.
- **Tidak** membangun model penugasannya; itu `B-6` (`D-26`).

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Satu layar yang menyesuaikan diri per peran, atau layar terpisah per peran?** | **Work Owner + Lead Engineer** | Menentukan bentuk seluruh modul. Satu layar berarti perbedaan antar peran menjadi data; layar terpisah berarti dua puluhan halaman yang harus dirawat sendiri-sendiri |
| **Kolom apa yang wajib ada di setiap inbox?** | **Work Owner** | Petugas memilih pekerjaan berdasarkan kolom yang terlihat. Salah kolom berarti mereka membuka klaim satu per satu untuk mencari |
| **Urutan baku: paling lama menunggu, atau paling dekat tenggat?** | **Work Owner** | Menentukan pekerjaan mana yang dikerjakan lebih dulu setiap hari |

## Acceptance criteria

- [ ] Layar inbox memakai komponen tabel baku `TKT-U2-001` — **bukan** tabel yang dibuat khusus.
- [ ] Paginasi dilakukan **di server**; layar **tidak memuat seluruh baris** lalu memotongnya —
      diuji dengan inbox berisi 5.000 baris.
- [ ] Kolom dan penyaring per peran berasal dari **konfigurasi**; menambah peran **tidak menuntut
      halaman baru** — diuji.
- [ ] Pekerjaan yang **lewat tenggat tertandai** dan dapat diurutkan lebih dulu — diuji.
- [ ] Petugas **hanya melihat pekerjaan yang menjadi haknya** — diuji dengan dua peran berbeda;
      penyaringan ditegakkan **di server** (`ADR-0023`), bukan di layar.
- [ ] Layar tetap terpakai pada **lebar layar yang dipakai petugas cabang** — diuji.
- [ ] Gerbang 2: UAT **PncRegister** dan **PncSurveyor** secara terpisah.

## Dependency / Blocked by

`TKT-U2-001` · `TKT-B06-002` · `TKT-F3-005`. **Terhalang tiga keputusan.**

## Constraint keamanan, data, operasional

- **Penyaringan inbox adalah kendali akses, bukan kenyamanan.** Bila dilakukan di layar saja,
  data pekerjaan peran lain tetap terkirim ke browser.
- Inbox adalah layar yang paling sering dibuka — beban kuerinya paling besar di seluruh aplikasi.
- Sebagian inbox memuat klaim dengan **data medis**; `FR-R2` berlaku pada kolom yang ditampilkan.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema; menambah tabel konfigurasi kolom per peran. Backward-compatible.

**Rollback:** mengembalikan versi layar. Petugas kembali ke inbox Pega selama peralihan (`D-05`).

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm test -- inbox --run TestPaginasiDariServer
npm test -- inbox --run TestKolomDariKonfigurasi
go test ./internal/adapter/http/... -run TestPenyaringanInboxDiServer
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 26 harness bernama Inbox dari 74 harness | direktori `Harness/` (dihitung langsung) |
| Model penugasan: worklist dan workbasket | `D-26` · `docs/Steering/CONTEXT.md` |
| Tabel baku median 6 kolom, paginasi server-side | `docs/Steering/06-MODULE-BREAKDOWN.md:76` |
| 3.189 grid terikat page list klipboard | `docs/Steering/06-MODULE-BREAKDOWN.md` koreksi ukuran |
| Otorisasi diperiksa di server | `D-59` · `ADR-0023` |

## Comments
