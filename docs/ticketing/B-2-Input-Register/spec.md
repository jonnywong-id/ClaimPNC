# B-2 · Input Register — Registrasi Klaim

| | |
|---|---|
| **Nama di sistem lama** | **Input Register** — tahap `InputRegister` pada `Flow/Register_Flow.xml`, layar `InboxRegister_Harness` |
| **Kode modul** | `B-2` (dipakai di ADR dan Steering) |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | **22 activity · 137 step validasi** — modul terdalam di seluruh sistem |
| **Bergantung pada** | `B-1` View Polis · `F-1`…`F-5` |
| **Kesiapan** | **SEBAGIAN** |

## Apa yang dikerjakan modul ini

Petugas mencatat klaim yang baru masuk: data kejadian, objek yang tertimpa, jaminan yang dipakai,
pembagian reasuransi, dan estimasi awal kerugian. Di ujungnya, sistem **menerbitkan nomor klaim**.

Inilah **gerbang validasi terberat di seluruh aplikasi**. Yang diperiksa di sini antara lain:

| Yang diperiksa | Aturannya |
|---|---|
| Urutan tanggal | Tanggal kejadian ≤ tanggal lapor ≤ tanggal terima dokumen ≤ hari ini |
| Tanggal kejadian dalam periode polis | ditambah toleransi **30 hari Bonding** dan **90 hari Travel/PA** |
| Klaim ganda | Polis + Objek + Lokasi; untuk PA ditambah Penyebab Kerugian `12002` |
| Objek dan coverage | Objek tanpa coverage tidak boleh tersimpan |
| Total spreading | wajib 100% (`ADR-0016`) |
| Nomor SLIK | wajib terisi untuk lini SPK / Asuransi Kredit |
| Penyebab kerugian | wajib, kecuali lini Travel |
| Nilai estimasi | tidak boleh melebihi TSI; memicu **Notice of Large Losses** bila > Rp 1 M |

## Rule Pega yang digantikan

| Jenis | Nama |
|---|---|
| Flow | `Flow/Register_Flow.xml` — tahap `Input Register` |
| Activity | `Activity/InputRegister_act-Act.xml` (**137 step**) dan 21 activity pendukung |
| Harness | `InboxRegister_Harness` |
| Ticket rule | `setToRegister_ticket` — **hilang dari export** |

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | When rule `IsAsuransiKredit` (gerbang SLIK) · `NonMBU`, `NotPA`, `ElseRCLMSIG` — percabangan flow utama | **Tim Pega** (`R-16`) |
| **Keputusan** | Toleransi **30 hari Bonding** — mati, atau BRD-nya salah? | **Work Owner** |
| **Keputusan** | Tanggal kejadian bagian dari kunci duplikasi? | **Work Owner** |
| **Keputusan** | Lokasi tidak diperiksa untuk PA dan Travel — sengaja? | **Work Owner** |
| **Keputusan** | **Jalur API/JSON melewati aturan 7/30/90 — sengaja?** | **Work Owner** |
| **Keputusan** | Validasi KTP/HP/Email dipanggil dengan argumen salah — pernah ada keluhan? | **Work Owner** |
| **Keputusan (tiket lain)** | Format nomor klaim: sequence direset tahunan? lebar tetap? (`TKT-F2-006`) | **Work Owner** |
| **ADR `Proposed`** | `ADR-0013` — pengganti pola hapus-lalu-sisip-ulang pada konversi klaim | **Work Owner + Lead Engineer** |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B02-001](issues/01-layar-input-register-dan-penerbitan-nomor-klaim.md) | Layar Input Register dan penerbitan nomor klaim | `needs-info` |
| [TKT-B02-002](issues/02-validasi-tanggal-dan-periode-polis.md) | Validasi urutan tanggal dan periode polis | `needs-info` |
| [TKT-B02-003](issues/03-validasi-klaim-ganda.md) | Validasi klaim ganda | `needs-info` |
| [TKT-B02-004](issues/04-validasi-kelengkapan-dan-notice-of-large-losses.md) | Validasi kelengkapan dan Notice of Large Losses | `ready-for-human` |
| [TKT-B02-005](issues/05-konversi-ulang-klaim-idempoten.md) | Konversi ulang klaim yang idempoten | `needs-info` |
