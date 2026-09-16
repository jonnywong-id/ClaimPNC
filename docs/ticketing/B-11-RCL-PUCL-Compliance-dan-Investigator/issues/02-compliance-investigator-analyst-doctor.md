---
title: "TKT-B11-002 — Compliance, Investigator, dan Analyst Doctor"
labels: [modul::B-11, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::4]
milestone: "Gelombang 4 — Persetujuan"
epic: "Migrasi Claim PNC"
---

# TKT-B11-002 — Compliance, Investigator, dan Analyst Doctor

Status: needs-info
Kesiapan: **terhalang artefak dan keputusan**
Modul: **B-11 RCL/PUCL** · Gelombang: 4 · Bergantung pada: TKT-B06-001, TKT-B11-001
Requirement: FR-B11, FR-R2    Keputusan: D-26, D-59    ADR: 0019, 0021, 0023    Risiko: R-16
Rule Pega yang digantikan: tahap **`Compliance`**, **`Investigator`**, **`Analyst Doctor`**, **`RCLDokter`**, **`Send To Analis`** pada `Flow/Register_Flow.xml` · harness `inboxCompliance_Harness`, `InboxInvestigator_Harness`, `inboxAnalystDoctor_Harness` · Ticket rule `CompliancePNC` (`:3844`), `SendToInvestigator` (`:3624`), `RCLDokter` (`:2954`) — **ketiganya hilang** · router `RouterRCLDokter` — **hilang**
Peran penguji gerbang 2: **PncComplience**, **PncInvestigator**, **PncAnalystDoctor**

## Hasil yang diharapkan (dan nilai bisnisnya)

Tiga jalur pemeriksaan khusus berjalan dengan inbox, kewenangan, dan jejaknya masing-masing.

Nilai bisnisnya berbeda-beda dan semuanya penting: **Compliance** mencegah klaim diselesaikan
melanggar ketentuan; **Investigator** menangani klaim mencurigakan; **Analyst Doctor** menilai
klaim yang menuntut pertimbangan medis — dan yang terakhir menyentuh **data medis**, yang aksesnya
dibatasi `FR-R2`.

## Ruang lingkup

- Tiga inbox terpisah: Compliance dan Investigator memakai **Workbasket**; Analyst Doctor dan
  RCL Dokter memakai **Worklist** (`D-26`).
- Pencatatan hasil pemeriksaan masing-masing jalur.
- **Lompatan lateral** masuk ke ketiga jalur — pengganti Ticket rule `CompliancePNC`,
  `SendToInvestigator`, dan `RCLDokter`.
- Penegakan `FR-R2`: **data medis hanya dapat diakses Analyst Doctor dan RCL Dokter**.

## Non-goal

- **Tidak** menangani penolakan umum — itu `TKT-B11-001`.
- **Tidak** membangun mekanisme penugasan — itu `B-6`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Ticket rule `CompliancePNC`, `SendToInvestigator`, `RCLDokter`** — dirujuk `Flow/Register_Flow.xml:3844`, `:3624`, `:2954`; **ketiganya tidak ada di export**, dan **tidak satu pun punya pemicu yang terverifikasi** | **Tim Pega** (`R-16`, `ADR-0021`) | **Kapan** klaim masuk ke ketiga jalur ini sama sekali tidak diketahui |
| **Router `RouterRCLDokter`** | **Tim Pega** (`R-04`) | Menentukan dokter mana yang menerima |
| **`CompliancePNC` dipakai serentak sebagai nama Ticket rule dan nama workbasket** — satu hal yang sama? | **Work Owner** | Bila berbeda, keduanya perlu dipisah di sistem baru |
| **Siapa yang boleh memicu transisi lateral?** | **Work Owner** | Sama dengan `TKT-B11-001` |

## Acceptance criteria

- [ ] Ketiga inbox memakai model yang benar: Compliance dan Investigator **Workbasket**, Analyst
      Doctor dan RCL Dokter **Worklist** — diuji keempatnya.
- [ ] Hasil pemeriksaan masing-masing jalur tersimpan dan **tercatat di jejak audit**.
- [ ] **Data medis hanya terlihat** oleh peran Analyst Doctor dan RCL Dokter — diuji dengan peran
      lain: field **tidak muncul** dan endpoint mengembalikan `403` (`FR-R2`).
- [ ] Pembatasan data medis berlaku **juga di staging** — diuji di sana, karena staging memuat data
      produksi apa adanya (`ADR-0029`).
- [ ] Lompatan lateral ke ketiga jalur berperilaku sesuai keputusan Work Owner, dan **tercatat**.
- [ ] Gerbang 1: isi inbox dan hasil pemeriksaan **sama dengan Pega** untuk peran yang sama.
- [ ] Gerbang 2: UAT oleh **ketiga peran secara terpisah** — bukan satu orang untuk semuanya,
      karena justru pemisahan aksesnya yang diuji.

## Dependency / Blocked by

`TKT-B06-001` · `TKT-B11-001` · `TKT-F3-005` (batas akses medis). **Terhalang Tim Pega dan dua
keputusan Work Owner.**

## Constraint keamanan, data, operasional

- **`FR-R2` adalah pembatasan akses paling ketat di seluruh aplikasi.** Data medis hanya untuk dua
  peran, dan pembatasan itu ditegakkan **di kueri**, bukan dengan menyembunyikan field di layar.
- Hasil investigasi memuat dugaan terhadap nasabah — ia **tidak boleh** muncul di layar peran lain
  maupun di laporan umum.
- Karena `D-59` tidak mengenal pemisahan tugas, seseorang yang memiliki menu Investigator **dan**
  menu akseptasi dapat menyelidiki sekaligus membayar klaim yang sama. Jejak audit satu-satunya
  kontrol.

## Migrasi skema / rollout / rollback

Menambah tabel hasil pemeriksaan per jalur. Backward-compatible.

**Rollback:** hasil pemeriksaan tetap ada; jalurnya kembali ditangani Pega.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/compliance/... -run TestInboxWorkbasket
go test ./internal/app/analystdoctor/... -run TestInboxWorklist
go test ./internal/adapter/http/... -run TestAksesDataMedisDibatasi
go run ./cmd/s8 banding --modul B-11 --peran 3
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Ticket rule `CompliancePNC`, `SendToInvestigator`, `RCLDokter` hilang | `Flow/Register_Flow.xml:3844`, `:3624`, `:2954` · `ADR-0021` |
| Tidak satu pun punya pemicu terverifikasi | `ADR-0021` — hanya 3 dari 17 punya pemicu |
| Pembagian Worklist versus Workbasket | `D-26` · `ADR-0019` |
| Akses data medis dibatasi dua peran | `FR-R2` · `BRD §17.2` |
| `CompliancePNC` dipakai sebagai Ticket rule dan workbasket | `docs/verifikasi-bukti-adr.md` §5 |

## Comments
