---
title: "TKT-U1-003 — Penanganan galat dan notifikasi global"
labels: [modul::U-1, tipe::fondasi, status::ready-for-human, prioritas::sedang, gelombang::2]
milestone: "Gelombang 2 — Kerangka UI"
epic: "Migrasi Claim PNC"
---

# TKT-U1-003 — Penanganan galat dan notifikasi global

Status: ready-for-human
Kesiapan: siap
Modul: U-1 · Gelombang: 2 · Bergantung pada: TKT-U1-001
Requirement: FR-U1    Keputusan: D-13    ADR: 0002    Risiko: —
Rule Pega yang digantikan: penyajian galat Pega yang tersebar di `Section/` dan `Harness/` — tidak ada pola terpusat yang dapat dirujuk
Peran penguji gerbang 2: **PncAdmin** (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu cara menampilkan galat dan pemberitahuan di seluruh aplikasi, sehingga pengguna belajar
sekali dan berlaku di semua layar.

Nilai bisnisnya paling terasa pada layar registrasi, yang menghadapkan pengguna pada belasan
aturan validasi sekaligus. Galat yang tampil dengan cara berbeda di setiap layar membuat pengguna
harus menebak apa yang salah — dan itu bertentangan dengan `D-13`, yang justru mempertahankan
tata letak agar **tidak ada pelatihan ulang**.

## Ruang lingkup

- Penangkap galat tingkat aplikasi yang mencegah satu galat merender **halaman kosong**.
- Pemetaan kontrak galat API (`TKT-F1-004`) ke tampilan: galat field, galat form, galat halaman,
  dan galat sistem.
- Komponen pemberitahuan (toast/banner) untuk hasil tindakan: berhasil, peringatan, gagal.
- Penampilan **pengenal permintaan** pada galat sistem, agar keluhan pengguna dapat ditelusuri ke
  log (`TKT-F1-003`).

## Non-goal

- **Tidak** menentukan teks galat per aturan bisnis — itu milik modul bisnisnya.
- **Tidak** mengirim laporan galat ke layanan pihak ketiga.

## Acceptance criteria

- [ ] Galat tak tertangani pada satu komponen **tidak membuat seluruh halaman kosong** — diuji
      dengan komponen yang sengaja melempar galat.
- [ ] Keempat jenis galat ditampilkan berbeda dan **dapat dibedakan** pengguna — diuji keempatnya.
- [ ] Galat sistem menampilkan **pengenal permintaan** yang dapat disalin pengguna, dan pengenal
      itu **sama** dengan yang ada di log server — diuji ujung ke ujung.
- [ ] Galat **tidak pernah** menampilkan stack trace, nama tabel, atau potongan SQL — diuji dengan
      galat database yang sengaja dipicu.
- [ ] Pemberitahuan berhasil hilang sendiri; pemberitahuan gagal **bertahan** sampai ditutup
      pengguna — diuji.
- [ ] Pemberitahuan dapat dibaca pembaca layar — diuji aksesibilitas.

## Dependency / Blocked by

Bergantung pada `TKT-U1-001`. Bentuk galat mengikuti `TKT-F1-004` yang berstatus `needs-info` —
**bentuk kontraknya sudah cukup jelas** untuk dipakai; yang belum diputuskan di sana adalah
perilaku baku 720 activity, bukan bentuk responsnya.

## Constraint keamanan, data, operasional

- Pesan galat yang sampai ke pengguna **tidak boleh membocorkan struktur internal**.
- Pengenal permintaan yang ditampilkan **bukan** data sensitif — ia justru dirancang untuk
  dibagikan pengguna saat melapor.
- Pemberitahuan **tidak menampilkan data nasabah** di luar konteks layar yang memang menampilkannya.

## Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** menjalankan bundel sebelumnya.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
npm run test -- Galat
npm run test -- Notifikasi
npm run test:e2e -- --grep "request id terlihat di galat sistem"
npm run test:a11y -- --grep "notifikasi"
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Bentuk kontrak galat API | `docs/Steering/10-API-STRATEGY.md` §3 · `TKT-F1-004` |
| ID permintaan di log dan respons | `docs/Steering/12-CROSSCUTTING.md` §2 |
| Tata letak mengikuti Pega; tanpa pelatihan ulang | `D-13` |
| Registrasi adalah gerbang validasi terberat | `docs/Steering/06-MODULE-BREAKDOWN.md` `B-2` |

## Comments
