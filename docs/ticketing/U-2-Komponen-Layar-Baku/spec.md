# U-2 — Pustaka Komponen

| | |
|---|---|
| **Modul** | `U-2` Pustaka Komponen |
| **Gelombang** | 2 — Kerangka UI |
| **Ukuran** | **Sedang** (turun dari Besar — lihat di bawah) |
| **Bergantung pada** | `U-1` untuk dipakai; **dapat dimulai paralel** |
| **Kesiapan** | **PENUH** — satu-satunya modul tanpa penghalang artefak maupun keputusan yang menahan |
| **Cakupan tiket** | **penuh** (`D-41` Opsi 1) |

## Apa yang dibangun

Komponen antarmuka baku yang dipakai berulang oleh seluruh layar: tabel, form, unggah berkas,
pemilih tanggal, dan pemformatan angka serta tanggal.

## Kenapa modul ini paling berdaya ungkit

**268 dari 269 section** sistem lama memakai pola grid yang sama. Satu komponen tabel yang benar
dipakai ratusan kali, dan satu perbaikan di dalamnya memperbaiki seluruh layar. Melewatkannya
berarti 268 implementasi tabel yang berbeda-beda — persis kegagalan yang `D-09` peringatkan pada
tim yang sedang belajar teknologi baru.

## Koreksi ukuran yang mengubah rencana

| Anggapan v1.0 | Terverifikasi |
|---|---|
| Tabel baku **18–27 kolom** | **median 6 kolom** |
| Grid berat karena banyak fitur | **tiga fitur termahal tidak dipakai sama sekali**: tambah baris inline, hapus baris inline, resize kolom |
| Masalahnya `OFFSET` besar | **`OFFSET` nol kemunculan**; masalah nyatanya **3.189 grid terikat page list klipboard** dan `pyMaxRecords=500` pada **54 dari 56** laporan |

Akibatnya `U-2` **turun dari Besar menjadi Sedang**, dan alasan memilih pustaka tabel kelas berat
ikut melemah — itulah yang membuat `TKT-U2-005` ada.

## Dua pertanyaan terbuka yang **tidak** menahan tiket mana pun

Keduanya memperluas atau mempersempit lingkup, bukan memblokirnya:

| Pertanyaan | Pemilik | Bila jawabannya "ya" |
|---|---|---|
| Grid 39 kolom dan 25 kolom — seluruh kolomnya benar-benar dilihat pengguna? | Work Owner | `TKT-U2-001` menambah fitur pemilih kolom |
| Tambah dan hapus baris inline diinginkan di sistem baru? | Work Owner | `TKT-U2-001` bertambah; hari ini **tidak dipakai sama sekali** di sistem lama |

## Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-U2-005](issues/05-pemilihan-pustaka-tabel.md) | Pemilihan pustaka tabel — TanStack Table versus AG Grid | `ready-for-human` | siap · **dikerjakan lebih dulu** |
| [TKT-U2-001](issues/01-komponen-tabel-baku.md) | Komponen tabel baku dengan paginasi keyset server-side | `ready-for-human` | siap |
| [TKT-U2-002](issues/02-komponen-form-baku.md) | Komponen form baku dan penyajian galat validasi | `ready-for-human` | siap |
| [TKT-U2-003](issues/03-komponen-unggah-berkas.md) | Komponen unggah berkas | `ready-for-human` | siap |
| [TKT-U2-004](issues/04-pemformatan-tanggal-dan-uang.md) | Pemilih tanggal dan pemformatan tanggal serta uang terpusat | `ready-for-human` | siap |
