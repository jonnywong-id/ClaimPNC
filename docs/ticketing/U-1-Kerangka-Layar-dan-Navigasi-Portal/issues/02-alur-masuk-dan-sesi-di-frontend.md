---
title: "TKT-U1-002 — Alur masuk dan penanganan sesi di frontend"
labels: [modul::U-1, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::2]
milestone: "Gelombang 2 — Kerangka UI"
epic: "Migrasi Claim PNC"
---

# TKT-U1-002 — Alur masuk dan penanganan sesi di frontend

Status: ready-for-human
Kesiapan: siap
Modul: U-1 · Gelombang: 2 · Bergantung pada: TKT-U1-001, TKT-F3-003
Requirement: FR-U1, FR-F3    Keputusan: D-07, D-27    ADR: 0024    Risiko: R-14
Rule Pega yang digantikan: layar masuk portal Pega — **tidak ada rule aplikasi** yang mengaturnya; HCC/HCQ muncul hanya sebagai 2 teks pesan galat
Peran penguji gerbang 2: **seluruh peran** — ini layar yang dilalui semua orang (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Layar masuk, penyimpanan sesi di peramban, dan perilaku yang benar ketika sesi habis di tengah
pekerjaan.

Nilai bisnisnya ada pada butir terakhir. Pengguna sistem klaim mengisi form panjang; sesi yang
habis di tengah pengisian **tanpa peringatan** berarti pekerjaan hilang — dan itu keluhan yang
paling cepat menghapus kepercayaan pada sistem baru.

## Ruang lingkup

- Layar masuk dengan penanganan tiga jenis galat yang **dibedakan**: kredensial salah, pengguna
  tidak aktif, dan **sistem identitas tidak dapat dihubungi**.
- Penyimpanan sesi di peramban dan pengirimannya pada setiap permintaan.
- **Peringatan sebelum sesi habis**, dan perilaku saat sesi ditolak server di tengah pekerjaan:
  pengguna diarahkan ke layar masuk **dengan isian yang belum tersimpan tidak hilang begitu saja**.
- Keluar (logout) yang menghapus sesi di server, bukan hanya di peramban.

## Non-goal

- **Tidak** memvalidasi kredensial — itu server (`TKT-F3-002`).
- **Tidak** menyimpan kata sandi di peramban dalam bentuk apa pun.

## Acceptance criteria

- [ ] Ketiga jenis galat masuk menampilkan pesan **berbeda dan dapat ditindaklanjuti** — diuji.
- [ ] Galat kredensial **tidak membedakan** apakah pengguna ada atau tidak — diuji dengan pengguna
      yang tidak ada dan pengguna yang ada berkata sandi salah: **pesannya sama**.
- [ ] Sesi yang ditolak server di tengah permintaan mengarahkan ke layar masuk, dan setelah masuk
      kembali pengguna **kembali ke halaman terakhir** — diuji.
- [ ] Peringatan muncul sebelum sesi habis, dengan pilihan memperpanjang — diuji dengan masa
      berlaku pendek.
- [ ] Keluar menghapus sesi di server — diuji dengan memakai token lama setelah keluar: **ditolak**.
- [ ] Token sesi **tidak muncul** di URL, di log peramban, maupun di pesan galat — diuji.

## Dependency / Blocked by

Bergantung pada `TKT-U1-001` dan `TKT-F3-003`.

**Catatan:** tiket ini **tidak terhalang** ketiadaan kontrak HCC/HCQ. Selama pengembangan ia
memakai adapter fake `TKT-F3-001`; yang berubah kelak adalah sisi server, bukan layar ini. Yang
**terhalang** hanyalah pesan galat yang spesifik terhadap kode galat HCC/HCQ.

## Constraint keamanan, data, operasional

- Galat autentikasi **tidak boleh membocorkan keberadaan akun**.
- Bila HCC/HCQ tidak dapat dihubungi, perilaku aplikasi mengikuti keputusan Work Owner yang
  **masih terbuka** (`ADR-0024`). Sampai itu diputuskan, layar menampilkan pesan "sistem identitas
  tidak dapat dihubungi" dan **tidak** menawarkan jalur lain.
- Isian yang belum tersimpan **tidak dikirim ke mana pun** saat sesi habis — ia tetap di memori
  peramban sampai pengguna menyimpannya.

## Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** menjalankan bundel sebelumnya; pengguna mungkin
perlu masuk ulang.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm run test -- Masuk
npm run test -- Sesi.habis
npm run test:e2e -- --grep "sesi habis di tengah form"
npm run test:e2e -- --grep "keluar mencabut sesi"
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Aplikasi menerbitkan sesi sendiri; pengguna tetap bekerja bila HCC/HCQ bermasalah | `D-07` · `docs/Steering/11-SECURITY.md` §2.1 |
| HCC/HCQ nol jejak di export | `T-2` · `ADR-0024` |
| Perilaku saat HCC/HCQ tidak dapat dihubungi belum diputuskan | `ADR-0024` Pertanyaan terbuka |
| Form panjang bernested repeat | `docs/Steering/01-FRONTEND-ANALYSIS.md` |
| Stateless; sesi dikenali lintas instans | `D-27` · `TKT-F3-003` |

## Comments
