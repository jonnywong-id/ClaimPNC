# B-10 · Akseptasi, Transfer Kasir & LOD

| | |
|---|---|
| **Nama di sistem lama** | **Akseptasi** — activity `InsertDataAkseptasiToLeader`, `InsertLogKasir_act`, `TransferCashierDataASM_act`, `HitupdateDataRekeningToKasir`; procedure `PEGA_JSON_OS_AKSEP_KLAIM`; master `MasterRekening` |
| **Kode modul** | `B-10` |
| **Gelombang** | 4 — Persetujuan |
| **Ukuran** | 31 activity |
| **Bergantung pada** | `B-5` Input Estimasi · `B-7` Komite |
| **Kesiapan** | **SEBAGIAN** — **lepas dari `BRD §21.4`** (`D-55`) |

## Apa yang dikerjakan modul ini

Menetapkan bahwa nilai klaim **disetujui untuk dibayarkan**, menerbitkan **Nomor Akseptasi**,
mengirim data pembayaran ke **kasir**, dan mencetak **LOD** — surat pemberitahuan nilai ganti rugi
kepada tertanggung.

Perbedaan yang penting dan sering tertukar:

| Istilah | Artinya |
|---|---|
| **Akseptasi** | persetujuan atas **nilai** yang akan dibayarkan |
| **LOD** | surat ke **tertanggung**, berbeda dari PLA/DLA yang ke koasuransi dan reasuransi |
| **OS — Outstanding** | akseptasi yang sudah diakui nilainya tetapi **belum selesai dibayar** |

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **`InsertDataAkseptasiToLeader`** · **`InsertLogKasir_act`** · **`TransferCashierDataASM_act`** — ketiganya dipanggil tetapi **tidak ada di export** | **Tim Pega** (`R-07`) |
| **Keputusan** | Header `Authorization` **di-hardcode di 3 Connect REST kasir** | **Tim Infra/Security** (`D-40`) |
| **Keputusan** | `PEGA_JSON_OS_AKSEP_KLAIM` memetakan `STS_PLA` ke kolom **`STS_DLA`** — benar? | **Work Owner** |
| **Catatan** | `ErrMsg` pada `ADD_NEWMASTERVIRTUALACCOUNT.prc:18` membawa **nomor virtual account sekaligus pesan galat** — kontrak itu tidak dibawa (`ADR-0007`) | — |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B10-001](issues/01-akseptasi-dan-nomor-akseptasi.md) | Akseptasi dan penerbitan Nomor Akseptasi | `needs-info` |
| [TKT-B10-002](issues/02-transfer-ke-kasir-dan-status-pembayaran.md) | Transfer ke kasir dan status pembayaran | `needs-info` |
| [TKT-B10-003](issues/03-cetak-lod.md) | Cetak LOD ke tertanggung | `needs-info` |
