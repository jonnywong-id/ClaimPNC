# U-3 · Layar Inbox per Peran

| | |
|---|---|
| **Nama di sistem lama** | **Inbox** — `InboxRegister_Harness`, `InboxKomite_Harness`, `InboxSurvey_Harness`, `InboxPLADLA`, `InboxSalvage`, `InboxInvestigator_Harness`, `inboxCompliance_Harness`, `inboxAnalystDoctor_Harness`, `InboxTKA_Harness`, `UserTeknisInbox`, `PNCInboxAdmin`, dan seterusnya |
| **Kode modul** | `U-3` |
| **Gelombang** | 4 — Inbox dan penugasan |
| **Ukuran** | **Besar** |
| **Bergantung pada** | `U-2` Komponen Layar Baku · `B-6` Penugasan & Inbox Petugas |
| **Kesiapan** | **SEBAGIAN** |

## Apa yang dikerjakan modul ini

Layar tempat setiap petugas melihat **pekerjaan yang menjadi tanggung jawabnya** — daftar klaim
yang menunggu tindakannya, sesuai perannya.

Inilah layar pertama yang dibuka petugas setiap pagi. Bila ia salah, petugas tidak tahu apa yang
harus dikerjakan hari itu.

## Ukuran yang terverifikasi

**26 harness** memuat kata *Inbox* dalam namanya (dihitung langsung dari direktori `Harness/`;
total harness 74).

> `06-MODULE-BREAKDOWN.md:85` menyebut **"~20 inbox berbasis peran"**. Selisihnya bukan
> pertentangan: sebagian dari 26 itu tampaknya **bukan inbox peran** melainkan daftar pilihan —
> misalnya `ListDocumentTypeInbox`, `CauseOfLossInbox`, `StatusClaimInbox`.
> **Definisinya kini ada (`D-79`):** Inbox = layar berisi **daftar pekerjaan milik pengguna** —
> Tugas dari Worklist/Workbasket. Layar data acuan **bukan** Inbox, sekalipun namanya mengandung
> kata itu. Dengan definisi ini **tujuh** dari 26 diusulkan pindah ke `U-6` karena bersidik jari
> layar master. Yang tersisa: **8 usulan `DUGAAN` dan 2 `JANGGAL`** menunggu koreksi Work Owner.

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | ~~Mana yang benar-benar inbox peran?~~ — **definisinya tertutup `D-79`**; sisa: koreksi 8 `DUGAAN` + 2 `JANGGAL` | **Work Owner** |
| **Keputusan** | Satu layar inbox yang menyesuaikan diri per peran, atau 20-an layar terpisah? | **Work Owner + Lead Engineer** |
| **Bergantung** | Model penugasan `B-6` harus lebih dulu ada | — (`D-26`) |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-U3-001](issues/01-layar-inbox-baku.md) | Layar inbox baku dan penyaringnya | `needs-info` |
| [TKT-U3-002](issues/02-inbox-per-peran-dan-pemetaannya.md) | Inbox per peran dan pemetaan dari harness | `needs-info` |

> **Daftar lengkap 74 harness beserta kelasnya:** [`../INVENTARIS-HARNESS.md`](../INVENTARIS-HARNESS.md)
> — pembagian modulnya **belum diputuskan** (`D-73`).
