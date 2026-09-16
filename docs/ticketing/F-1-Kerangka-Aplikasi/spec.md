# F-1 — Kerangka Aplikasi

| | |
|---|---|
| **Modul** | `F-1` Kerangka Aplikasi |
| **Gelombang** | 1 — Fondasi |
| **Ukuran** | Kecil |
| **Bergantung pada** | — (tidak ada; ini modul paling awal) |
| **Kesiapan** | **SEBAGIAN** — lingkup jelas, satu keputusan menahan satu tiket |
| **Cakupan tiket** | **penuh** (`D-41` Opsi 1) |

## Apa yang dibangun

Rangka aplikasi Go yang menjadi tempat seluruh modul lain hidup: struktur folder dan aturan
lapisan, konfigurasi, logging, penanganan galat, health check, graceful shutdown, dan penyajian
SPA sebagai berkas statis.

**Tidak ada aturan bisnis di modul ini.** Bila sebuah tiket `F-1` mulai membicarakan klaim,
komite, atau spreading, tiket itu salah tempat.

## Kenapa ini lebih dulu

Setiap modul lain menulis kode ke dalam struktur yang ditetapkan `F-1`. Menetapkannya belakangan
berarti memindahkan seluruh kode yang telanjur ditulis — dan `D-09` menetapkan tim adalah
developer Pega yang dilatih ulang, sehingga **struktur yang preskriptif dan seragam** justru
bagian terpenting dari modul ini.

## Yang mengikat modul ini

| Sumber | Isi |
|---|---|
| `ADR-0001` | Modular monolith Go, satu binary, VM on-premise, tanpa orkestrator kontainer |
| `ADR-0002` | SPA React disajikan oleh binary Go — tidak ada runtime Node.js di produksi |
| `D-27` | 24/7 — stateless, dua instans, health check, graceful shutdown |
| `D-09` | Struktur folder, penamaan, dan pola baku ditulis preskriptif |
| `ADR-0025` | Nilai bisnis tidak boleh di-hardcode; rahasia keluar dari kode |

## Penghalang modul ini

| Jenis | Isi | Pemilik | Menahan |
|---|---|---|---|
| **Keputusan** | **720 dari 902 activity sistem lama tidak punya penanganan galat sama sekali.** Apakah kegagalan senyap itu direplikasi, atau sistem baru gagal keras? | Work Owner | `TKT-F1-004` |
| Artefak | instance Dynamic System Setting `ServiceFromTable` (1 setting, hanya rujukan) · `Data-Admin-DB-Name` tidak ada di export | Tim Pega | tidak menahan — daftar setting disusun dari nol |

## Daftar tiket

| Tiket | Judul | Status | Kesiapan |
|---|---|---|---|
| [TKT-F1-001](issues/01-kerangka-proyek-dan-aturan-lapisan.md) | Kerangka proyek Go dan penegakan aturan lapisan | `ready-for-human` | siap |
| [TKT-F1-002](issues/02-konfigurasi-tiga-lapis.md) | Konfigurasi tiga lapis dan gagal keras saat start | `ready-for-human` | siap |
| [TKT-F1-003](issues/03-logging-terstruktur.md) | Logging terstruktur dan ID permintaan | `ready-for-human` | siap |
| [TKT-F1-004](issues/04-penanganan-galat-terpusat.md) | Penanganan galat terpusat dan kontrak galat API | `needs-info` | terhalang keputusan |
| [TKT-F1-005](issues/05-health-check-shutdown-penyajian-spa.md) | Health check, graceful shutdown, dan penyajian SPA | `ready-for-human` | siap |
