---
title: "TKT-U3-002 — Inbox per peran dan pemetaan dari harness"
labels: [modul::U-3, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::4]
milestone: "Gelombang 4 — Inbox dan penugasan"
epic: "Migrasi Claim PNC"
---

# TKT-U3-002 — Inbox per peran dan pemetaan dari harness

Status: needs-info
Kesiapan: **terhalang keputusan lingkup — jumlah layar belum pasti**
Modul: **U-3 Layar Inbox per Peran** · Gelombang: 4 · Bergantung pada: TKT-U3-001
Requirement: FR-U3    Keputusan: D-26, D-58, D-79    ADR: 0012    Risiko: —
Rule Pega yang digantikan: **26 harness Inbox** di `Harness/` — `InboxRegister_Harness`, `InboxKomite_Harness`, `InboxSurvey_Harness`, `InboxPLADLA`, `InboxPLA_harness`, `InboxSalvage`, `InboxInvestigator_Harness`, `inboxCompliance_Harness`, `inboxAnalystDoctor_Harness`, `InboxTKA_Harness`, `InboxRCVApp_Harness`, `InboxClaimTreaty_Harness`, `InboxClaimNonProp_Harness`, `Inbox_XOL_Harness`, `InboxManagerAdmin_Harness`, `InboxKomunikasiCabang`, `InboxAutoClaim`, `PNCInboxAdmin`, `UserInbox_Harness`, `UserTeknisInbox`, `SurveyorsInbox`, `DetailSurveyorsInbox`, `StatusClaimInbox`, `CauseOfLossInbox`, `CauseOfLossInboxSimasOnline`, `ListDocumentTypeInbox`
Peran penguji gerbang 2: **satu peran per inbox** — bukan satu orang untuk semuanya

## Hasil yang diharapkan (dan nilai bisnisnya)

Setiap peran punya inbox yang menampilkan pekerjaannya, dan **tidak ada peran yang kehilangan
inbox-nya** saat Pega dimatikan.

Nilai bisnisnya terletak pada kata "tidak ada yang kehilangan". Ada 22 peran bisnis (`D-58`) dan
26 harness bernama Inbox. Bila pemetaannya tidak dibuat eksplisit, peran yang terlewat baru akan
ketahuan **setelah** Pega mati — saat petugasnya tidak punya layar untuk bekerja.

## Ruang lingkup

- **Pemetaan eksplisit**: harness Inbox mana melayani peran mana, satu per satu.
- Pemisahan mana yang **inbox peran** dan mana yang **daftar pilihan** — `ListDocumentTypeInbox`,
  `CauseOfLossInbox`, `StatusClaimInbox` tampaknya termasuk yang kedua.
- Konfigurasi kolom dan penyaring per inbox di atas layar baku `TKT-U3-001`.

## Non-goal

- **Tidak** menambah inbox untuk peran yang sekarang tidak punya.
- **Tidak** menggabungkan dua inbox menjadi satu tanpa persetujuan Work Owner.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| ~~Mana yang benar-benar inbox peran?~~ — **definisi tertutup `D-79`**: Inbox = daftar **pekerjaan** milik pengguna; layar data acuan **bukan** Inbox. Tujuh dari 26 diusulkan pindah ke `U-6` | **Work Owner** | Sisa yang menahan: **8 usulan `DUGAAN`** dan **2 `JANGGAL`** pada inventaris harness |
| **Pemetaan 22 peran bisnis ke inbox-nya** | **Work Owner** (`D-58`) | Tanpa ini tidak dapat dipastikan setiap peran punya tempat bekerja |
| **Ada peran yang berbagi satu inbox, atau punya lebih dari satu?** | **Work Owner** | Menentukan apakah pemetaannya satu-ke-satu |

## Acceptance criteria

- [ ] **Setiap dari 22 peran bisnis punya inbox**, dan pemetaannya tercatat sebagai tabel — tidak
      ada peran tanpa inbox, diperiksa satu per satu.
- [ ] Jumlah inbox yang dibangun **dilaporkan sebagai angka** dan cocok dengan daftar yang
      disetujui Work Owner.
- [ ] Harness yang dinyatakan **bukan inbox peran** dipindahkan ke modul yang tepat, **tidak
      hilang begitu saja** — dicatat per berkas.
- [ ] Tiap inbox menampilkan **pekerjaan yang sama dengan inbox Pega** untuk peran yang sama —
      dibandingkan lewat `S-8` pada 10 kasus per inbox.
- [ ] Peran **tidak dapat membuka inbox peran lain** lewat URL langsung — diuji: `403`.
- [ ] Gerbang 2: UAT **oleh pemegang masing-masing peran**, bukan satu orang untuk semuanya.

## Dependency / Blocked by

`TKT-U3-001` · `TKT-F3-002` (22 peran) · `TKT-B06-002`. **Terhalang Work Owner.**

## Constraint keamanan, data, operasional

- **Peran yang terlewat baru terlihat setelah Pega mati.** Itulah mengapa pemetaannya harus
  eksplisit sebelum cutover, bukan sesudahnya.
- `inboxAnalystDoctor_Harness` dan inbox RCL Dokter memuat **data medis** — `FR-R2` berlaku, dan
  kedua inbox itu tidak boleh terbuka bagi peran lain.
- `InboxKomite_Harness` menampilkan klaim yang menunggu persetujuan — isinya menentukan apa yang
  dilihat komite sebelum memutuskan (`B-7`).

## Migrasi skema / rollout / rollback

Tidak menyentuh skema; menambah baris konfigurasi per inbox.

**Rollout:** per peran, bukan sekaligus. Peran yang inbox-nya belum dialihkan **tetap memakai Pega**
(`D-05`).

**Rollback:** peran dikembalikan ke inbox Pega satu per satu.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go run ./cmd/tools/cek-inbox-per-peran --harap-semua-22
go test ./internal/adapter/http/... -run TestInboxPeranLainDitolak
go run ./cmd/s8 banding --modul U-3 --per-inbox 10
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 26 harness bernama Inbox | direktori `Harness/` (dihitung langsung) |
| Dokumen menyebut "~20 inbox berbasis peran" | `docs/Steering/06-MODULE-BREAKDOWN.md:85` |
| 22 peran bisnis, satu-untuk-satu dengan access group | `D-58` |
| Model penugasan worklist/workbasket | `D-26` |
| Akses data medis dibatasi dua peran | `FR-R2` |

## Comments
