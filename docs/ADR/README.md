# Architecture Decision Records — Migrasi Claim PNC

Satu berkas = satu keputusan yang mahal dibalik. Keputusan yang murah dibalik **tidak** ada di
sini; ia tetap hidup di `docs/Steering/00-DECISION-LOG.md`.

**Uji kelayakan yang dipakai menyaring:** kalau keputusan ini dibalik enam bulan lagi, adakah
arsitektur yang harus dibongkar besar atau risiko bisnis yang muncul? Dari **70 keputusan** di
Decision Log, **29 lolos** dan 41 tidak.

Seluruh ADR di sini bersifat **`retrospective`** — keputusannya diambil pada Sesi 1–3
(2026-09-07 … 2026-09-14), dokumennya ditulis 2026-09-14. Itu tertulis terbuka di field `Sifat`
setiap berkas; tidak ada yang dibuat seolah ditulis lebih awal.

---

## Index

Kolom `Jenis` dan `Modul terdampak` ada di tabel ini, bukan di dalam berkas ADR — sesuai `D-31`.

| # | Judul | Status | Jenis | Modul terdampak | Pemilik keputusan |
|---|---|---|---|---|---|
| [0001](0001-modular-monolith-go-on-premise.md) | Modular monolith Go di VM on-premise, tanpa orkestrator kontainer | Accepted | turunan `D-08` | seluruh modul | Work Owner |
| [0002](0002-frontend-spa-react-disajikan-binary-go.md) | SPA React+TypeScript disajikan oleh binary Go | Accepted | turunan `D-23` | `U-1`…`U-6` | Work Owner |
| [0003](0003-peralihan-bertahap-strangler-fig.md) | Peralihan bertahap Strangler Fig per gelombang modul | Accepted | turunan `D-05` | seluruh modul | Work Owner |
| [0004](0004-database-bersama-penulis-tunggal.md) | Satu database dipakai bersama selama masa paralel, penulis tunggal per tabel | Accepted | turunan `D-21` | seluruh modul | Work Owner |
| [0005](0005-oracle-dulu-postgresql-kemudian.md) | Oracle 19c selama paralel, PostgreSQL 17+ setelah cutover, SQL portabel | Accepted | turunan `D-01` | `F-2` + seluruh modul | Work Owner |
| [0006](0006-snapshot-polis-milik-gisfw.md) | Data polis milik GISFW; Claim PNC menyimpan snapshot | Accepted | turunan `D-04` | `B-1`, `B-2` | Work Owner |
| [0007](0007-stored-procedure-naik-ke-go.md) | Logika stored procedure dinaikkan ke Go; transaksi dimiliki aplikasi | Accepted | turunan `D-02` + `D-68` | `B-4`, `B-5`, `B-9`, `B-10`, `B-12` | Work Owner |
| [0008](0008-db-link-diganti-api.md) | DB Link diganti kontrak API eksplisit | Accepted | turunan `D-25` | `S-4`, `B-7`, `B-12` | Work Owner |
| [0009](0009-penomoran-klaim-pncn.md) | Nomor klaim baru `PNCN.YY.xxxx` dari sequence; prefix Pega ditinggalkan | Accepted | turunan `D-22`, **menyupersede format pada `D-22`** (`D-71`) | `B-2` | Work Owner |
| [0010](0010-dokumen-satu-jalur-api-storage.md) | Dokumen lewat satu jalur API storage internal; DB hanya metadata | Accepted | turunan `D-16` | `S-1` | Work Owner |
| [0011](0011-pdf-excel-csv-dibangun-di-go.md) | PDF/Excel/CSV dibangun di dalam aplikasi Go | Accepted | turunan `D-11` | `S-2`, `U-5` | Work Owner |
| [0012](0012-soft-delete-menyeluruh.md) | Soft delete menyeluruh — tanpa penghapusan fisik data bernilai bisnis | Accepted | turunan `D-66`, **menyupersede `D-65`** | `B-2`, `S-5`, `S-8` | Work Owner |
| [0013](0013-pengganti-hapus-lalu-sisip-ulang.md) | Pengganti pola hapus-lalu-sisip-ulang pada konversi klaim | **Proposed** | baru | `B-2` | Work Owner + Lead Engineer |
| [0014](0014-komite-kumulatif-per-lini.md) | Komite kumulatif menurut ambang bawah; pita nilai hanya Non-MBU | Accepted | turunan `D-47` + `D-52` + `D-70` | `B-7` | Work Owner |
| [0015](0015-kurs-tanggal-kejadian.md) | Kurs pada tanggal kejadian; klaim ditolak bila kurs tidak ditemukan | Accepted | turunan `D-48` | `B-2`, `B-5` | Work Owner |
| [0016](0016-presisi-uang-dan-toleransi-spreading.md) | Uang disimpan presisi penuh; toleransi total spreading | Accepted | turunan `D-51` | `B-4`, `B-5` | Work Owner |
| [0017](0017-kebijakan-cacat-aturan-uang.md) | Sembilan cacat aturan uang diperbaiki, satu direplikasi secara sadar | Accepted | turunan `D-49` | `B-2`…`B-10` | Work Owner |
| [0018](0018-empat-konsep-status.md) | Empat konsep status dipertahankan dengan nama yang tidak tertukar | Accepted | turunan `D-18` | seluruh modul bisnis | Work Owner |
| [0019](0019-worklist-workbasket-dan-routing.md) | Worklist/Workbasket dipertahankan; router dibangun ulang sebagai aturan routing | Accepted | turunan `D-26` | `B-6` | Work Owner |
| [0020](0020-dua-basis-perhitungan-tat.md) | Dua basis perhitungan TAT dipertahankan untuk keperluan berbeda | Accepted | turunan `D-50` | `S-2`, `S-7` | Work Owner |
| [0021](0021-transisi-lateral-pengganti-ticket-rule.md) | Transisi lateral (Ticket rule) dibangun ulang sebagai perpindahan tahap eksplisit | **Proposed** | baru | `B-7`, `B-11`, `B-13`, `B-14` | Tim Pega → Work Owner |
| [0022](0022-penjadwal-job-terjadwal.md) | Job terjadwal dibangun ulang sebagai penjadwal aplikasi | **Proposed** | turunan `D-57` | `S-6` | Lead Engineer + Infra |
| [0023](0023-otorisasi-berbasis-menu.md) | Otorisasi berbasis menu ditegakkan di server; 22 peran; tanpa pemisahan tugas | Accepted | turunan `D-59` + `D-58` | `F-3` | Work Owner |
| [0024](0024-autentikasi-hcc-hcq.md) | Autentikasi didelegasikan ke HCC/HCQ; aplikasi menerbitkan sesinya sendiri | **Proposed** | turunan `D-07` | `F-3` | Tim HCC/HCQ → Work Owner |
| [0025](0025-hardcode-menjadi-master-dan-konfigurasi.md) | Seluruh hardcode menjadi master/konfigurasi; rahasia keluar dari kode | **Proposed** | turunan `D-15` + `D-40` | `F-4`, `F-5` | Tim Infra/Security |
| [0026](0026-jejak-audit-append-only.md) | Jejak audit append-only; retensi mengikuti retensi data klaim | Accepted | turunan `D-28` + `D-62` | `S-5` | Work Owner |
| [0027](0027-dua-gerbang-penerimaan.md) | Dua gerbang penerimaan; uji kesetaraan otomatis sebagai gerbang 1 | Accepted | turunan `D-42` + `D-53` + `D-54` + `D-60` | seluruh modul | Work Owner |
| [0028](0028-modul-tanpa-baseline-pega.md) | Modul tanpa baseline Pega diukur terhadap kontrak fungsional | Accepted | turunan `D-56` + `D-55` · **menyupersede `BRD §21.4`** | `F-3`, `S-5` | Work Owner |
| [0029](0029-data-nasabah-di-staging.md) | Data nasabah nyata dipakai di staging; aturan penyamaran di dokumen | Accepted | turunan `D-64` + `D-69` | `S-8` + seluruh dokumen | Work Owner |

---

## Status yang dipakai

| Status | Artinya |
|---|---|
| `Accepted` | Work Owner sudah memutuskan; keputusan mengikat tiket dan implementasi |
| `Proposed` | **belum** diputuskan. Berkasnya memuat konteks, opsi, pemilik keputusan, dan apa yang diblokir — **tanpa** bagian `Keputusan`. Tidak boleh dijadikan dasar implementasi |
| `Superseded by ADR-XXXX` | digantikan ADR lain; isinya dibiarkan utuh sebagai jejak |
| `Deprecated` | tidak lagi berlaku dan tidak diganti |

**Lima ADR berstatus `Proposed`** dan apa yang masing-masing menghalangi:

| # | Yang belum diputuskan | Menghalangi |
|---|---|---|
| 0013 | pengganti pola hapus-lalu-sisip-ulang: *upsert* kunci alami atau versioning | tiket `B-2` tidak dapat ditulis lengkap |
| 0021 | 8 Ticket rule custom hilang dari export; 14 dari 17 nama tidak punya pemicu | `B-7`, `B-11`, `B-13`, `B-14` |
| 0022 | siapa yang menjalankan job saat aplikasi hidup di dua instans; nasib `AutoAcceptKomite` | `S-6` |
| 0024 | kontrak API HCC/HCQ — nol jejak di export | `F-3`, dan login seluruh aplikasi |
| 0025 | ke mana rahasia dipindahkan dan siapa pemiliknya (`D-40` OPEN) | `F-4`, `F-5`, seluruh deployment |

---

## Hubungan dengan dokumen lain

| Dokumen | Perannya |
|---|---|
| `docs/Steering/00-DECISION-LOG.md` | sumber seluruh `D-nn`. ADR **tidak** menggantikannya |
| `docs/Steering/CONTEXT.md` | kamus istilah. ADR memakai istilahnya, tidak mendefinisikan ulang |
| `docs/verifikasi-bukti-adr.md` | bukti Fase 1 — `berkas:baris`, angka, dan kontradiksi `K-nn` |
| `docs/BRD.md` | sumber `FR-xx` |
| `docs/ticketing/` | tiket pekerjaan; setiap tiket merujuk ADR yang mengikatnya |

> **Peringatan istilah.** Folder `Ticket/` pada export Pega berisi **Ticket rule** — mekanisme
> lompatan lateral (lihat ADR 0021). Kata **"tiket"** di seluruh dokumen proyek ini selalu
> berarti **tiket pekerjaan** di `docs/ticketing/`.

---

## Template ADR kosong

```markdown
# NNNN — <judul keputusan, kalimat aktif>

Status: Proposed | Accepted | Superseded by ADR-XXXX | Deprecated
Tanggal keputusan: YYYY-MM-DD    Tanggal dokumen: YYYY-MM-DD
Sifat: original | retrospective
Pemilik keputusan: <peran, mis. Work Owner / Lead Engineer / Tim Infra>
Jejak bukti: D-nn | FR-xx | R-nn | path/berkas:baris | nama objek DB
Terkait: CONTEXT.md#<istilah>, ADR-XXXX, modul <kode>

## Konteks
## Opsi yang dipertimbangkan
## Keputusan
## Rationale
## Konsekuensi
### Positif
### Negatif / utang teknis
### Risiko yang diterima secara sadar
## Pertanyaan terbuka
```

**Aturan menulis yang mengikat:**

1. Satu berkas = satu keputusan yang mahal dibalik.
2. `Sifat: retrospective` wajib ditulis terbuka bila keputusan mendahului dokumennya.
3. **Bagian `Negatif / utang teknis` tidak boleh kosong.** ADR yang hanya memuat keuntungan
   adalah dokumen jualan, bukan ADR.
4. Jangan menulis `Keputusan` untuk hal yang belum diputuskan Work Owner — pakai `Proposed`,
   sebutkan pemilik keputusan dan apa yang diblokir.
5. Bila ADR bertentangan dengan Steering atau BRD, tulis **`Menyupersede …`** secara eksplisit
   dan ajukan revisinya sebagai pertanyaan terbuka. Jangan mengedit Steering diam-diam.
6. Nilai sensitif tidak pernah ditulis: kredensial, kunci API, hostname/IP produksi, dan data
   nasabah dirujuk dengan `berkas:baris` + nama elemen saja. Alamat email disamarkan (`D-69`).
