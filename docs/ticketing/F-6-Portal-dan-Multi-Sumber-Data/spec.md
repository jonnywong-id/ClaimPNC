# F-6 · Portal & Multi-Sumber Data

| | |
|---|---|
| **Nama di sistem lama** | **tidak ada namanya** — dikerjakan diam-diam lewat perbandingan hostname: **48 perbandingan** terhadap `pxRequestor.pxReqServer` pada 3 hostname, ditambah When rule `IsServerSyariah` yang **hilang dari export** |
| **Kode modul** | `F-6` |
| **Gelombang** | 1 — Fondasi |
| **Ukuran** | Sedang — tetapi **melipatgandakan** beban `F-4`, `S-2`, `S-5`, `U-6`, dan `S-8` |
| **Bergantung pada** | `F-1` Kerangka Aplikasi · `F-2` Akses Data · `F-3` Login & Hak Akses |
| **Kesiapan** | **SEBAGIAN** |

## Apa yang dikerjakan modul ini

Memberi pengguna **pilihan portal** — sebuah **dropdown di dalam aplikasi**, bukan kembali ke
halaman login (`D-77`) — dan mengarahkan seluruh pembacaan serta penulisan ke **database
entitas** yang dipilih.

| Portal | Jejak di export |
|---|---|
| Asuransi Sinar Mas | entitas utama |
| Asuransi Simas Insurtech | **13** perbandingan hostname |
| Sinarmas Asuransi Syariah | `SpreadingSyariah_Act` ada · **`IsServerSyariah` hilang** |
| Timor-Leste | **1** perbandingan · **mata uang berbeda** |

## Ini menyingkap yang sudah ada, bukan menambah kemampuan

Sistem lama sudah berperilaku berbeda per entitas — dengan **membandingkan nama server**. Salah
satu perbandingan itu **mengubah ambang komite dari Rp 50.000.000 menjadi 3.500**; selisihnya bukan
salah ketik melainkan **mata uang yang berbeda**.

Tidak ada pilihan portal, tidak ada konfigurasi, dan tidak pernah tercatat sebagai rancangan.

## Temuan yang mengubah rencana

| Anggapan | Terverifikasi |
|---|---|
| 3 hostname penentu perilaku | **Bisa lebih dari 3** — `IsServerSyariah` dan `IsDevelopmentServer` **hilang dari export dan tidak tercatat di `19-GAP`** (`verifikasi-bukti-adr.md:788-791`) |
| Database bersama (`D-21`, `ADR-0004`) | **Satu database per entitas** (`D-75`) |
| Syariah hanya varian produk | Punya **logika spreading sendiri** — `Activity/SpreadingSyariah_Act-Act.xml` |

> **Jawaban Work Owner mengoreksi bukti kami.** Dokumen mencatat 3 hostname; yang benar minimal 4
> portal. Angka 3 rendah karena **rule pembedanya hilang**, bukan karena entitasnya tiga. Karena itu
> daftar portal diperlakukan sebagai **data**, bukan konstanta di kode.

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | **`IsServerSyariah` dan `IsDevelopmentServer` hilang** — tidak diketahui **kapan** perilaku syariah berlaku | **Tim Pega** (`R-16`) |
| **Keputusan** | Jumlah portal sebenarnya — Work Owner menyebut *minimal* 4 | **Work Owner** |
| **Keputusan** | Nomor klaim `PNCN.YY.xxxx` memakai sequence **per database** — dua portal dapat menerbitkan nomor sama. Perlu penanda portal? | **Work Owner** (`D-71`) |
| **Keputusan** | Sejauh apa aturan bisnis syariah berbeda di luar spreading? | **Work Owner + Tim Pega** |
| **Keputusan** | Satu penyedia identitas untuk keempat entitas, atau satu per entitas? | **Work Owner** (`F-3`) |
| **Keputusan** | Mata uang, kurs, dan pembulatan per portal | **Work Owner** |

## Akibat pada modul lain — bukan detail

| Modul | Akibat |
|---|---|
| `F-2` | Pool koneksi **per portal**; migrasi skema berjalan **empat kali** |
| `F-3` | Kewenangan **per portal**, dinilai ulang saat berpindah (`R-20`) |
| `F-4` · `U-6` | Master ≥29 kelompok kini **per portal** — lingkup berlipat |
| `S-2` · `S-7` | Laporan **per portal**; konsolidasi lintas portal **tidak mungkin** tanpa keputusan baru |
| `S-5` | Jejak audit **per portal** — penyelidikan lintas entitas menuntut empat kueri |
| `S-8` | Uji kesetaraan berjalan **per portal**; portal Syariah **belum punya baseline utuh** |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-F6-001](issues/01-daftar-portal-sebagai-data.md) | Daftar portal sebagai data, bukan konstanta | `needs-info` |
| [TKT-F6-002](issues/02-koneksi-per-portal.md) | Koneksi dan pool per portal | `needs-info` |
| [TKT-F6-003](issues/03-perpindahan-portal-dan-kewenangan.md) | Perpindahan portal dan penilaian ulang kewenangan | `needs-info` |
| [TKT-F6-004](issues/04-kesekerabatan-skema-empat-database.md) | Kesekerabatan skema empat database | `needs-info` |
