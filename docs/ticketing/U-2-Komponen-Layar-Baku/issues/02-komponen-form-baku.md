---
title: "TKT-U2-002 — Komponen form baku dan penyajian galat validasi"
labels: [modul::U-2, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::2]
milestone: "Gelombang 2 — Kerangka UI"
epic: "Migrasi Claim PNC"
---

# TKT-U2-002 — Komponen form baku dan penyajian galat validasi

Status: ready-for-human
Kesiapan: siap
Modul: U-2 · Gelombang: 2 · Bergantung pada: TKT-U1-001
Requirement: FR-U2    Keputusan: D-13, D-09    ADR: 0002    Risiko: —
Rule Pega yang digantikan: pola form pada `Section/` dan `Harness/` — form panjang bernested repeat, terberat registrasi klaim (`InputRegister_act`, **137 step validasi**)
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi UI (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu cara membangun form dan **satu cara menampilkan galat validasi**, dipakai seluruh layar
transaksi.

Nilai bisnisnya terletak pada galat. Registrasi klaim adalah gerbang validasi terberat di seluruh
sistem; pengguna di sana menghadapi belasan aturan sekaligus. Bila setiap layar menampilkan galat
dengan caranya sendiri, pengguna harus belajar ulang di setiap layar — dan `D-13` justru
menetapkan alur dan tata letak dipertahankan agar **tidak ada pelatihan ulang**.

## Ruang lingkup

- Komponen field baku: teks, angka, uang, tanggal, pilihan, pencarian, dan area teks.
- **Satu pola penyajian galat**: galat per field ditampilkan di field-nya, galat lintas field
  ditampilkan di kepala form, dan keduanya **menunjuk ke field yang salah**.
- Pola form bernested repeat — objek pertanggungan berisi coverage berisi settlement line —
  karena itulah bentuk nyata layar registrasi.
- Keadaan **sedang menyimpan** dan pencegahan kiriman ganda.

## Non-goal

- **Tidak** menulis aturan validasi bisnis — itu milik modul bisnisnya, dan **server tetap
  penegak sebenarnya**. Validasi di klien hanya kenyamanan.
- **Tidak** merancang ulang tata letak layar (`D-13`).

## Acceptance criteria

- [ ] Galat dari server yang menyebut nama field **otomatis muncul di field itu** — diuji dengan
      respons galat berbentuk kontrak `TKT-F1-004`.
- [ ] Galat lintas field muncul di kepala form dan menautkan ke field pertama yang salah — diuji.
- [ ] Menekan simpan dua kali cepat **hanya mengirim satu permintaan** — diuji.
- [ ] Form bernested tiga tingkat dapat menambah dan menghapus baris pada tingkat terdalam tanpa
      kehilangan isian tingkat di atasnya — diuji.
- [ ] Field uang menolak masukan bukan angka dan **tidak membulatkan** nilai yang diketik —
      pembulatan hanya saat tampil (`ADR-0016`).
- [ ] Seluruh field dapat dioperasikan dengan papan ketik saja — diuji pada satu form penuh.

## Dependency / Blocked by

Bergantung pada `TKT-U1-001`. Bentuk galat mengikuti kontrak `TKT-F1-004`, yang berstatus
`needs-info` — **tetapi bentuk kontraknya sudah cukup jelas** untuk dipakai; yang belum diputuskan
di sana adalah perilaku baku 720 activity, bukan bentuk responsnya.

## Constraint keamanan, data, operasional

- **Validasi klien bukan pengaman.** Server memvalidasi ulang seluruhnya; klien hanya mempercepat
  umpan balik.
- Nilai uang **tidak dibulatkan saat diketik maupun dikirim** — presisi penuh disimpan, pembulatan
  hanya saat ditampilkan (`D-51`, `ADR-0016`).
- Angka pecahan di JavaScript **tidak presisi** — nilai uang tidak boleh melewati batas itu sebagai
  angka; dikirim dan diterima sebagai string desimal.

## Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** mengembalikan versi komponen.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm run test -- Form
npm run test -- Form.galat
npm run test -- Form.nested
npm run test:a11y -- --grep "form registrasi"
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Registrasi adalah gerbang validasi terberat (137 step) | `docs/Steering/06-MODULE-BREAKDOWN.md` `B-2` |
| Tata letak dan alur mengikuti Pega; tanpa pelatihan ulang | `D-13` |
| Form panjang bernested repeat | `docs/Steering/01-FRONTEND-ANALYSIS.md` |
| Uang disimpan presisi penuh, dibulatkan saat tampil | `D-51` · `ADR-0016` |
| Bentuk kontrak galat API | `docs/Steering/10-API-STRATEGY.md` §3 |

## Comments
