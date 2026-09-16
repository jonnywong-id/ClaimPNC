---
title: "TKT-F1-004 — Penanganan galat terpusat dan kontrak galat API"
labels: [modul::F-1, tipe::fondasi, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F1-004 — Penanganan galat terpusat dan kontrak galat API

Status: needs-info
Kesiapan: **terhalang keputusan** — lihat "Yang kurang dan siapa yang bisa melengkapinya"
Modul: F-1 · Gelombang: 1 · Bergantung pada: TKT-F1-001, TKT-F1-003
Requirement: FR-F1    Keputusan: D-49 (butir 5, 10)    ADR: 0007, 0017    Risiko: R-19
Rule Pega yang digantikan: pola galat tersebar — **720 dari 902 activity tanpa penanganan galat sama sekali**; `Database/GETCURRENCYSTANDARD.fnc:22` (`RETURN 1`); `Database/GETSELISIHJAM.fnc:22` (`RETURN 0`); kontrak `ErrMsg` berbasis string pada 12 procedure
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu cara menangani galat di seluruh aplikasi, dan satu bentuk respons galat yang dipakai setiap
endpoint — sehingga frontend tidak perlu menebak bentuk galat per layar, dan kegagalan tidak lagi
hilang tanpa jejak.

Nilai bisnisnya sangat konkret. Di sistem lama, kegagalan **senyap** justru terjadi pada jalur
uang: fungsi kurs mengembalikan `1` ketika kurs tidak ditemukan, sehingga klaim bernilai besar
menyusut menjadi kecil lalu **lolos tanpa komite**. Tidak ada galat, tidak ada catatan, dan
angkanya tampak wajar.

## Ruang lingkup

- Jenis galat domain yang dibedakan dari galat teknis, dan **pemetaannya ke kode status HTTP**.
- Satu bentuk respons galat untuk seluruh endpoint: kode galat, pesan untuk pengguna, dan
  pengenal permintaan (`TKT-F1-003`).
- Middleware yang menangkap panic dan mengubahnya menjadi respons galat, **dengan baris log
  lengkap**, tanpa membocorkan stack trace ke pengguna.
- **Larangan nilai bawaan yang menyamar sebagai hasil** — pola `RETURN 1` dan `RETURN 0` sistem
  lama tidak boleh punya padanan di sistem baru.

## Non-goal

- **Tidak** menentukan pesan galat per aturan bisnis — itu milik modul bisnis masing-masing.
- **Tidak** menerjemahkan pesan galat ke bahasa lain.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan tiket ini |
|---|---|---|
| **720 dari 902 activity sistem lama tidak punya penanganan galat sama sekali.** Apakah kegagalan senyap itu **direplikasi** demi kesetaraan `P-5`, atau sistem baru **gagal keras**? | **Work Owner** | Menentukan perilaku baku seluruh aplikasi. Bila gagal keras dipilih, uji kesetaraan akan menunjukkan selisih pada setiap jalur yang dulu diam — dan selisih itu harus dinyatakan di muka sebagai perbaikan terencana, bukan ditemukan sebagai kejutan |

**Catatan yang mempersempit pertanyaannya.** Dua kasus sudah diputuskan dan **tidak perlu
ditanyakan lagi**: kurs tidak ditemukan → klaim **ditolak** (`D-48`, `D-49` butir 5), dan
`GETSELISIHJAM` gagal → **tidak** mengembalikan `0` yang tak terbedakan dari nol (`D-49` butir 10).
Yang ditanyakan adalah **perilaku baku untuk 720 activity sisanya**.

## Acceptance criteria

> Ditulis sebagai rancangan; **belum boleh dijadikan dasar implementasi** sampai pertanyaan di
> atas terjawab, karena butir pertama berubah bentuk tergantung jawabannya.

- [ ] Seluruh endpoint mengembalikan galat dengan **bentuk yang sama**, diuji pada minimal 3
      endpoint berbeda.
- [ ] Respons galat memuat `request_id` yang sama dengan yang ada di log.
- [ ] Respons galat **tidak pernah** memuat stack trace, nama tabel, atau potongan SQL — diuji
      dengan galat database yang sengaja dipicu.
- [ ] Panic pada handler menghasilkan respons `500` dengan bentuk baku **dan** satu baris log
      bertingkat `error` — aplikasi **tetap hidup**.
- [ ] Galat domain "tidak ditemukan", "tidak berwenang", dan "masukan tidak sah" masing-masing
      memetakan ke kode status yang berbeda dan terdokumentasi.
- [ ] **Nol fungsi yang mengembalikan nilai bawaan saat gagal** — diuji lewat pemeriksaan pola
      pada kode: tidak ada `return 0, nil` atau `return 1, nil` pada jalur galat.

## Dependency / Blocked by

- Bergantung pada `TKT-F1-001`, `TKT-F1-003`.
- **Terhalang keputusan Work Owner** di atas.

## Constraint keamanan, data, operasional

- Pesan galat yang sampai ke pengguna **tidak boleh** membocorkan struktur internal: nama tabel,
  nama kolom, potongan SQL, atau jalur berkas.
- Galat pada jalur uang **tidak boleh** senyap. Bila nilai tidak dapat dihitung, permintaan gagal
  — bukan dilanjutkan dengan nilai bawaan.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** mengembalikan middleware ke versi sebelumnya; tidak ada data
yang berubah.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/platform/errors/...
go test ./internal/adapter/http/... -run TestKontrakGalat
grep -rIn -E "return (0|1|\"\"), nil" internal/   # HARUS 0 baris pada jalur galat
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 720 dari 902 activity tanpa penanganan galat | `docs/verifikasi-bukti-adr.md` §10.9 #3 |
| Kurs tidak ditemukan → `RETURN 1` | `Database/GETCURRENCYSTANDARD.fnc:22` · `D-49` butir 5 |
| `GETSELISIHJAM` gagal → `RETURN 0` | `Database/GETSELISIHJAM.fnc:22` · `D-49` butir 10 |
| Kontrak galat `ErrMsg` berbasis string tidak dibawa | `D-68` · `ADR-0007` |
| Bentuk respons galat baku | `docs/Steering/12-CROSSCUTTING.md` §1 · `docs/Steering/10-API-STRATEGY.md` §3 |

## Comments
