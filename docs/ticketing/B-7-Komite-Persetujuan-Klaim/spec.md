# B-7 · Komite Persetujuan Klaim

| | |
|---|---|
| **Nama di sistem lama** | **Komite** — `Flow/Komite_Flow.xml`, inbox `InboxKomite_Harness`, activity `SetEmailKomite`, `SetListComiteeClaimPerObjAdj`, `GetKomiteApproval`, `AutoAcceptKomite` |
| **Kode modul** | `B-7` |
| **Gelombang** | 4 — Persetujuan |
| **Ukuran** | 34 activity |
| **Bergantung pada** | `B-5` Input Estimasi · `B-6` Penugasan · `F-4` Master Data |
| **Kesiapan** | **TERHALANG** |

## Apa yang dikerjakan modul ini

Meminta persetujuan berjenjang atas **nilai klaim** sebelum klaim dapat diakseptasi. Komite dapat
menyetujui, menolak, atau mengembalikan.

## Cara jumlah penyetuju ditentukan — kumulatif

Ini bagian yang paling mudah disalahpahami, dan sudah dua kali nyaris salah dibaca selama analisis.

**Bukan** memilih satu jenjang dari matriks. Melainkan: **setiap jenjang yang ambang bawahnya sudah
terlampaui nilai klaim ikut menyetujui.**

```
Jumlah jenjang = jumlah baris master yang LIMIT_BOTTOM <= nilai klaim
```
`Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459` · `When/IsKomiteLoop-When.xml`

Bukti pendukung: `LIMIT_BOTTOM` difilter di **11 SQL rule**; `LIMIT_TOP` di **0**.

Untuk lini **Non-MBU saja**, akumulasi didahului pemilihan **pita nilai** (≤ Rp 100 juta → pita
`1`; di atasnya → pita `2`). Lini lain tidak punya langkah itu (`D-70`).

Contoh nyata dari master yang sudah diterima:

| Kasus | Jumlah penyetuju |
|---|---|
| PA Rp 5.000.000 | 1 |
| PA Rp 75.000.000 | 3 |
| PA Rp 150.000.000 | 4 |
| Travel Rp 150.000.000 | 3 |
| Non-MBU Rp 80.000.000 | 2 |
| Non-MBU Rp 750.000.000 | 2 |
| Non-MBU Rp 2.000.000.000 | 3 |

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | Ticket rule `KomiteAssign_ticket` dan `komiteAccept_ticket` · When rule `IsKomite` | **Tim Pega** (`R-16`) |
| **Keputusan** | **`AutoAcceptKomite` menyetujui komite otomatis tiap hari jam 06:00** — perilaku yang benar? | **Work Owner** |
| **Keputusan** | Baris `DEGREE=0` maksudnya apa? | **Work Owner** |
| **Keputusan** | **PA dan Travel di atas Rp 200.000.000 tidak punya baris master** | **Work Owner** |
| **Keputusan** | `KOMITEKE` immutable? · `dbms_random.value` pada 2 kueri Simasnet disengaja? | **Work Owner** |
| **Terhalang tiket** | `TKT-F4-002` — struktur master ambang komite | — |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B07-001](issues/01-penentuan-jenjang-komite-kumulatif.md) | Penentuan jenjang komite kumulatif | `needs-info` |
| [TKT-B07-002](issues/02-layar-komite-dan-pencatatan-keputusan.md) | Layar Komite dan pencatatan keputusan | `needs-info` |
| [TKT-B07-003](issues/03-persetujuan-otomatis-autoacceptkomite.md) | Persetujuan otomatis `AutoAcceptKomite` | `needs-info` |
