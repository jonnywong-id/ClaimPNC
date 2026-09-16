# B-14 · Input Receive Document — Penerimaan Dokumen Fisik

| | |
|---|---|
| **Nama di sistem lama** | **Input Receive Document** — `Flow/InputReceiveDocument.xml`, layar `ReceiveDoucument_Harness`, `InboxRCVApp_Harness`, `ViewReceiveDocument` |
| **Kode modul** | `B-14` |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | 8 activity |
| **Bergantung pada** | `F-3` Login & Hak Akses |
| **Kesiapan** | **SEBAGIAN** — kendala paling sedikit di antara modul bisnis |

## Apa yang dikerjakan modul ini

Mencatat **kedatangan dokumen fisik klaim** dari cabang: siapa mengirim, lewat ekspedisi apa,
nomor resi, berapa lembar, jenis dokumen, kapan dikirim, perkiraan tiba, dan kapan benar-benar
diterima.

Fungsinya sederhana tetapi menentukan: **tanggal terima dokumen** yang dicatat di sini menjadi
salah satu dari tiga tanggal yang divalidasi di Input Register (`TKT-B02-002`), dan menjadi titik
awal perhitungan TAT.

## Rule Pega yang digantikan

| Jenis | Nama |
|---|---|
| Flow | `Flow/InputReceiveDocument.xml` |
| Harness | `ReceiveDoucument_Harness` · `InboxRCVApp_Harness` · `ViewReceiveDocument` · `ViewTempDetailReqDocument` |
| Ticket rule | `SendToReceiveDocument` (`Flow/InputReceiveDocument.xml:690`) — **hilang dari export** |
| Router | `PNCAdminRouterRCV` |

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | Ticket rule `SendToReceiveDocument` — **dirujuk tetapi tidak ada di export** | **Tim Pega** (`R-16`) |
| **ADR `Proposed`** | `ADR-0021` — kapan lompatan lateral ke tahap Receive Document dipicu belum diketahui | Tim Pega → Work Owner |

Tidak ada penghalang keputusan bisnis. **Ini modul bisnis paling siap dikerjakan.**

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B14-001](issues/01-layar-input-receive-document.md) | Layar Input Receive Document | `ready-for-human` |
| [TKT-B14-002](issues/02-inbox-penerimaan-dan-penugasan.md) | Inbox penerimaan dokumen dan penugasannya | `needs-info` |
| [TKT-B14-003](issues/03-kaitan-dokumen-ke-klaim.md) | Kaitan dokumen ke klaim dan tanggal terima | `ready-for-human` |
