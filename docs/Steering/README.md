# Steering — Migrasi Aplikasi Claim PNC

Blueprint migrasi aplikasi **Claim PNC** dari **Pega PRPC 8.3** ke **Golang + React**, dengan
database **Oracle 19c → PostgreSQL 17+**.

Disusun 2026-09-07, **diperbarui ke v2.0 pada 2026-09-14**, berdasarkan pembacaan menyeluruh atas
export rule XML Pega — **2.634 berkas XML** ditambah 55 `.prc` dan 8 `.fnc` — serta discovery
bersama pemilik project.

> **Apa yang berubah sejak v1.0 ada di [21-RIWAYAT-REVISI.md](21-RIWAYAT-REVISI.md).**

> **Status: menunggu persetujuan.** Belum ada satu baris kode implementasi yang ditulis, sesuai
> ketentuan bahwa implementasi baru dimulai setelah Steering disetujui.

---

## Versi gabungan untuk distribusi

Seluruh isi folder ini juga tersedia sebagai **satu dokumen utuh**, disusun menjadi **16 bab
bernomor plus 6 lampiran**, didahului bab **Riwayat Revisi**:

| Berkas | Kegunaan |
|---|---|
| [`STEERING.md`](STEERING.md) | Dokumen gabungan — Riwayat Revisi + 16 bab + Lampiran A–F |
| `Steering Document - Migrasi CLAIM PNC ke Golang.docx` | Versi Word untuk dibagikan dan direview |
| `ARSIP v1.0 (2026-09-08) - …docx` | Versi Word lama, disimpan sebagai arsip |

**Folder ini adalah satu-satunya sumber yang disunting.** `STEERING.md` **dibangun otomatis** oleh
`../tools/build-steering.js`; menyuntingnya langsung sia-sia karena perubahannya hilang pada
pembangunan berikutnya. Cara menjalankannya ada di [`../tools/README.md`](../tools/README.md).

Pemetaan bab dokumen gabungan ke berkas di folder ini:

| Bab gabungan | Sumber |
|---|---|
| — Riwayat Revisi | `21-RIWAYAT-REVISI.md` |
| 1 Pemahaman Bisnis | `02-BUSINESS-UNDERSTANDING.md` |
| 2 Arsitektur Saat Ini | `03-CURRENT-ARCHITECTURE.md` |
| 3 Arsitektur Masa Depan | `04-FUTURE-ARCHITECTURE.md` |
| 4 Domain Model dan Bounded Context | `05-DOMAIN-MODEL.md` |
| 5 Module Breakdown | `06-MODULE-BREAKDOWN.md` |
| 6 Strategi Migrasi | `07-MIGRATION-STRATEGY.md` |
| 7 Strategi Teknis, Struktur Folder, Standar Coding | `08-TECHNICAL-STRATEGY.md` |
| 8 Strategi Database | `09-DATABASE-STRATEGY.md` |
| 9 Strategi API | `10-API-STRATEGY.md` |
| 10 Keamanan, Autentikasi, Otorisasi | `11-SECURITY.md` |
| 11 Error Handling, Logging, Konfigurasi, Observability | `12-CROSSCUTTING.md` |
| 12 Strategi Deployment dan Infrastruktur | `13-DEPLOYMENT.md` |
| 13 Strategi Testing | `14-TESTING-STRATEGY.md` |
| 14 NFR, Performa, Skalabilitas, Maintainability | `15-NFR-PERFORMANCE-SCALABILITY.md` |
| 15 Analisis Risiko | `16-RISK-ANALYSIS.md` |
| 16 Future Enhancement | `17-FUTURE-ENHANCEMENT.md` |
| Lampiran A | `CONTEXT.md` |
| Lampiran B | `00-DECISION-LOG.md` |
| Lampiran C | `01-FRONTEND-ANALYSIS.md` |
| Lampiran D | `18-SKILLS-USAGE-LOG.md` |
| Lampiran E | `19-GAP-EXPORT-DETAIL.md` |
| Lampiran F | `20-DETAIL-KOMITE-DBLINK.md` |

---

## Cara membaca dokumen ini

| Anda | Mulai dari |
|---|---|
| Manajemen / pengambil keputusan | `02` Business Understanding → `16` Risk Analysis → `07` Migration Strategy |
| Arsitek / tech lead | `03` Current Architecture → `04` Future Architecture → `05` Domain Model |
| Developer backend | `08` Technical Strategy → `09` Database Strategy → `10` API Strategy |
| Developer frontend | `01` Frontend Analysis → `08` Technical Strategy §3 dan §5 |
| Siapa pun yang bertanya "kenapa diputuskan begitu?" | `00` Decision Log |

---

## Daftar dokumen

| # | Dokumen | Isi |
|---|---|---|
| — | [CONTEXT.md](CONTEXT.md) | **Glossary bahasa domain.** Wajib dibaca lebih dulu — seluruh dokumen lain memakai istilah ini |
| 00 | [Decision Log](00-DECISION-LOG.md) | **72 keputusan** dalam 3 sesi: pertanyaan, pilihan yang ditawarkan, jawaban, keputusan akhir, dampaknya |
| 01 | [Frontend Analysis](01-FRONTEND-ANALYSIS.md) | Perbandingan 8 alternatif frontend beserta rekomendasi dan justifikasinya |
| 02 | [Business Understanding](02-BUSINESS-UNDERSTANDING.md) | Proses bisnis, aturan bisnis, peran pengguna, aliran nilai klaim |
| 03 | [Current Architecture](03-CURRENT-ARCHITECTURE.md) | Peta sistem Pega yang berjalan dan utang teknisnya |
| 04 | [Future Architecture](04-FUTURE-ARCHITECTURE.md) | Arsitektur sasaran, seam, modul dalam, bounded context |
| 05 | [Domain Model](05-DOMAIN-MODEL.md) | Aggregate, invarian, peristiwa domain, pemetaan ke sistem lama |
| 06 | [Module Breakdown](06-MODULE-BREAKDOWN.md) | Pembagian modul, ukuran, dan urutan ketergantungannya |
| 07 | [Migration Strategy](07-MIGRATION-STRATEGY.md) | Strangler Fig, 7 tahap, migrasi data, verifikasi kesetaraan, rollback |
| 08 | [Technical Strategy](08-TECHNICAL-STRATEGY.md) | Tumpukan teknologi, struktur folder, coding standards, penegakan otomatis |
| 09 | [Database Strategy](09-DATABASE-STRATEGY.md) | SQL portabel, tipe data, indexing, connection pool, jejak audit, migrasi skema |
| 10 | [API Strategy](10-API-STRATEGY.md) | Bentuk REST, penamaan, paginasi, kontrak, integrasi keluar |
| 11 | [Security](11-SECURITY.md) | Authentication (HCC/HCQ), Authorization, pengamanan, data sensitif |
| 12 | [Cross-Cutting](12-CROSSCUTTING.md) | Error handling, logging, configuration, observability |
| 13 | [Deployment](13-DEPLOYMENT.md) | Topologi, rolling deployment 24/7, lingkungan, backup, hidup berdampingan dengan Pega |
| 14 | [Testing Strategy](14-TESTING-STRATEGY.md) | Piramida pengujian, uji kesetaraan dengan Pega, uji beban |
| 15 | [NFR & Performance](15-NFR-PERFORMANCE-SCALABILITY.md) | Target non-fungsional, performa, scalability, maintainability |
| 16 | [Risk Analysis](16-RISK-ANALYSIS.md) | **19 risiko** beserta penanganannya, dan 15 tindakan hari pertama berstatus |
| 17 | [Future Enhancement](17-FUTURE-ENHANCEMENT.md) | Yang sengaja ditunda sampai setelah migrasi |
| 18 | [Skills Usage Log](18-SKILLS-USAGE-LOG.md) | Dokumentasi penggunaan Matt Pocock Skills |
| 19 | [Gap Export Detail](19-GAP-EXPORT-DETAIL.md) | **Daftar lengkap R-01, R-04, R-07** — 64 procedure/function, 8 router, 50 activity yang tidak ada di export |
| 20 | [Detail Komite & DB Link](20-DETAIL-KOMITE-DBLINK.md) | **Rincian D-14 dan R-03** — penjenjangan komite (termasuk **jawaban final §1.8** setelah master diterima) dan inventaris 64 pemakaian DB Link |
| 21 | [Riwayat Revisi](21-RIWAYAT-REVISI.md) | **Apa yang berubah v1.0 → v2.0 dan atas dasar apa** — angka yang dikoreksi, perubahan per bab, penghalang yang tertutup dan yang baru |

---

## Ringkasan keputusan utama

| Aspek | Keputusan | Ref |
|---|---|---|
| Backend | Go 1.22+, `chi`, `database/sql`, tanpa ORM | — |
| Frontend | React 18 + TypeScript + Vite + TanStack Table/AG Grid, SPA murni | D-23 |
| Database sasaran | PostgreSQL **17+** (mengikat) | D-24 |
| Database sementara | Oracle 19c, dengan **satu set SQL portabel** | D-20 |
| Stored procedure | Tidak dipanggil sama sekali; logikanya naik ke Go | D-02 |
| Data polis | **Snapshot** saat registrasi, bukan pemanggilan runtime | D-04 |
| Cutover | **Strangler Fig**, satu database bersama, kepemilikan tabel per modul | D-05, D-21 |
| Nomor klaim | `PNCN.YY.xxxx` dari `POOLDATA.CLAIM_NO_NONPEGA_SEQ` | D-22, D-71 |
| Authentication | API internal **HCC/HCQ** | D-07 |
| Authorization | Dimiliki aplikasi, tabel peran dan izin baru | D-07 |
| Deployment | 2 VM on-premise di belakang load balancer | D-08, D-27 |
| Ketersediaan | **24/7**, rolling deployment tanpa downtime | D-27 |
| Nilai bisnis | Tidak ada hardcode — seluruhnya master data | D-15 |
| Jejak audit | Append-only, dirancang sejak awal | D-28 |
| Laporan | PDF/Excel/CSV dibuat sendiri di Go | D-11 |
| Tampilan | Meniru alur dan tata letak Pega | D-13 |
| Target waktu | Seluruh modul akhir September 2026 (ditetapkan manajemen) | D-30, R-05 |

---

## Ukuran sistem yang dimigrasi

| Artefak | Jumlah |
|---|---|
| Rule XML | **2.634 berkas** + 55 `.prc` + 8 `.fnc` |
| Activity | 902 — **15.063 step** (778 custom) |
| Connect-SQL | 652 — **534 KB SQL** |
| Section (komponen UI) | 269 — 268 bergrid |
| Harness (layar) | 74 |
| Report Definition | 56 |
| Data Transform | 80 |
| When (business rule) | 70 |
| Flow Action | 29 |
| Connect-REST | 12 |
| Flow (case type) | 4 |
| Tabel database | 245 di 5+ schema |
| Stored procedure & function database | 70 |
| DB Link ke sistem lain | 6 |
| Access group | 22 |
| Menu portal | 47 |

---

## Yang harus dilakukan sebelum implementasi dimulai

Sepuluh tindakan di [Risk Analysis §Ringkasan](16-RISK-ANALYSIS.md). **Sembilan di antaranya
bergantung pada pihak di luar tim pengembang**, sehingga waktu tunggunya tidak dapat dipercepat
dengan menambah developer — dan karena itu harus dimulai hari pertama.

Yang paling mendesak:

1. **Source 64 procedure & function** dari DBA — memblokir 5 modul inti (R-01). Daftar lengkap: [`19-GAP-EXPORT-DETAIL.md`](19-GAP-EXPORT-DETAIL.md)
2. ~~**Isi tabel `POOLDATA.EMAILKOMITE`**~~ — ✅ **diterima**, 21 kolom, 30 baris (D-14)
3. ~~**Export Rule-Agent / Queue Processor**~~ — ✅ **diterima**, 5 job + 1 agent (R-02 tertutup)
4. ~~**Isi master `V_STS_CLAIM`**~~ — ✅ **diterima**, 33 kode `1134`–`1166` (R-06 tertutup)
5. **Kebutuhan 6 API pengganti DB Link** ke tim pemilik sistem (R-03)
6. **DDL lengkap + statistik ukuran tabel** (R-08)
7. **Export ulang berbasis Product rule** — ±242 rule hilang, 137 di antaranya When rule (R-16)
8. **Kontrak API HCC/HCQ** (ADR-0024) · **daftar peristiwa wajib audit** (ADR-0026) · **konfirmasi Pega staging** (ADR-0027)
9. **Tujuan penyimpanan rahasia dan keputusan rotasi** (D-40, R-17)

---

## Keputusan yang masih terbuka

| ID | Perlu dilengkapi | Kepada |
|---|---|---|
| ~~D-14~~ | ✅ **tertutup** — master diterima; model kumulatif ditetapkan (D-47, D-70) | — |
| ~~D-17~~ | ✅ **tertutup** — 5 job + 1 agent (D-57) | — |
| ~~D-18~~ | ✅ **tertutup** — 33 kode `1134`–`1166` | — |
| **D-40** | **Tujuan penyimpanan rahasia dan keputusan rotasi** | **Tim Infra / Security** |
| D-25 | Kontrak 6 API pengganti DB Link | Tim pemilik sistem |
| D-28 | Lama retensi data audit | Compliance |
| D-29 | Angka RPO/RTO standar korporat | Tim infra |
