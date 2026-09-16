# B-6 · Penugasan & Inbox Petugas

| | |
|---|---|
| **Nama di sistem lama** | **Worklist / Workbasket** Pega — router `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter`, `KomiteRouter`, `PNCAdminRouterRCV`; inbox `PNCInboxAdmin`, `UserTeknisInbox`, `UserInbox_Harness`, `InboxManagerAdmin_Harness` |
| **Kode modul** | `B-6` |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | 30 activity |
| **Bergantung pada** | `F-3` Login & Hak Akses |
| **Kesiapan** | **SEBAGIAN** — algoritma sudah terbaca, router perlu konfirmasi |

## Apa yang dikerjakan modul ini

Menentukan **siapa mengerjakan apa**. Setiap tahap klaim menghasilkan satu **Tugas**, dan tugas
itu berada di salah satu dari dua tempat:

| | Isi | Dipakai pada tahap |
|---|---|---|
| **Worklist** | tugas milik **satu orang tertentu** | Input Register · View Polis · Estimation · Input Estimasi · Choose Surveyor · Send To Analis · Send To PIC Teknik · RCLDokter · Analyst Doctor |
| **Workbasket** | antrean **bersama**, diambil siapa pun yang berwenang | RCL/PUCL · Investigator · Compliance |

## Algoritma pembagian beban — sudah terbaca

`R-04` semula menyatakan tiga router hilang sehingga aturan penugasan tidak diketahui. Algoritmanya
ternyata **terbaca dari tempat lain**:

```sql
ORDER BY counter_quota ASC     -- RDB List/BrowsePICRandomTeam-SQL.xml:39-40
```

Petugas dengan beban paling sedikit mendapat tugas berikutnya, lalu `AddTJobCounterPIC_SQL`
menaikkan pencacahnya. `R-04` karena itu **turun dari penghalang menjadi verifikasi**.

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter` — **tidak ada di export** · 3 `Data-Admin-WorkBasket` · 4 harness inbox | **Tim Pega** (`R-04`, `R-16`) |
| **Keputusan** | Konfirmasi ketiga router memang memakai jalur `counter_quota` | **Work Owner + Tim Pega** |
| **Keputusan** | `operator_ID != 'ELLENSUPRIYATI'` **di-hardcode di dalam SQL** — dibawa sebagai peran, atau dihapus? | **Work Owner** |
| **Catatan** | Seluruh export hanya memuat **4 workbasket**; penjenjangan komite justru menugaskan ke **operator bernama** (`T-7`) | — |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B06-001](issues/01-model-penugasan-worklist-dan-workbasket.md) | Model penugasan Worklist dan Workbasket | `ready-for-human` |
| [TKT-B06-002](issues/02-aturan-routing-dan-pembagian-beban.md) | Aturan routing dan pembagian beban | `needs-info` |
| [TKT-B06-003](issues/03-penguncian-tugas-antar-pengguna.md) | Penguncian tugas antar pengguna | `ready-for-human` |
