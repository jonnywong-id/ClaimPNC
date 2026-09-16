# 0018 — Pertahankan empat konsep status, masing-masing dengan nama yang tidak dapat tertukar

Status: Accepted
Tanggal keputusan: 2026-09-07 (`D-18`), ditegaskan ulang 2026-09-14    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-18`, `D-19` | `docs/verifikasi-bukti-adr.md` §4 (T-5, T-6) | master status: 33 kode `1134`–`1166`
Terkait: CONTEXT.md#Status, ADR-0006, seluruh modul bisnis

## Konteks

Sistem lama membawa **empat penanda status berbeda** pada klaim yang sama:

| Sistem lama | Isi |
|---|---|
| `StatusWork` | posisi klaim dalam alur kerja |
| `StatusClaim` | status bisnis klaim — **33 kode**, `1134`–`1166` |
| `ClaimStatus` | penanda biner `0`/`1` |
| `StatusPosisi` | posisi pada rangkaian tahapan progres |

Nama-nama itu **hampir tidak dapat dibedakan satu sama lain**: `StatusClaim` dan `ClaimStatus`
berbeda hanya pada urutan kata. Kesalahan membaca keempatnya adalah sumber kekeliruan yang mudah
terjadi dan sulit terdeteksi.

Dua temuan Fase 1 sempat meragukan keempatnya:

- **T-5** — `ClaimStatus` ternyata milik ruleset **GISFW**, bukan Claim PNC.
- **T-6** — `StatusPosisi` bukan properti dan hanya pernah muncul dengan satu nilai
  (`'On Progress'`) pada 21 titik panggil di export.

## Opsi yang dipertimbangkan

1. **Pertahankan keempatnya**, beri nama yang jelas berbeda.
2. Gabungkan menjadi lebih sedikit konsep, karena dua di antaranya tidak terbukti punya domain
   nilai yang kaya.
3. Rancang ulang model status dari kebutuhan bisnis hari ini.

## Keputusan

**Keempat konsep dipertahankan** — dikonfirmasi ulang oleh pemilik bisnis: *"4 status tersebut
memang berbeda"*. Di sistem baru masing-masing diberi nama yang **tidak lagi bisa tertukar**:

| Nama baru | Sistem lama | Isi |
|---|---|---|
| **Status Proses** | `StatusWork` | sedang berjalan, selesai, atau ditolak |
| **Status Klaim** | `StatusClaim` | status bisnis, 33 kode berlabel di master |
| **Flag Klaim** | `ClaimStatus` | penanda biner |
| **Status Posisi Progres** | `StatusPosisi` | posisi pada rangkaian tahapan progres |

Domain **Status Klaim** dikoreksi menjadi rentang penuh `1134`–`1166` — **33 kode**, bukan
`1142`–`1151` seperti yang dipakai dokumen-dokumen awal. Sebelas kode pertama (`1134`–`1144`)
membawa penomoran lama `01`–`11`.

## Rationale

Menggabungkan konsep status adalah perubahan makna bisnis, bukan penyederhanaan teknis — dan
pemilik bisnis menyatakan keempatnya memang berbeda. Menggabungkannya akan menghilangkan
perbedaan yang dipakai orang dalam bekerja.

Yang benar-benar menjadi sumber kesalahan bukan jumlah konsepnya, melainkan **namanya**. Itulah
yang diperbaiki.

## Konsekuensi

### Positif

- Kekeliruan membaca `StatusClaim` vs `ClaimStatus` tidak mungkin lagi terjadi, karena namanya
  tidak lagi mirip.
- Domain 33 kode terdokumentasi untuk pertama kalinya; sebelumnya hanya sebagian yang diketahui.
- Model status tetap sesuai cara pengguna bekerja hari ini, sejalan dengan `D-13`.

### Negatif / utang teknis

- **Empat konsep status pada satu entitas tetap rumit**, apa pun namanya. Setiap layar dan laporan
  harus jelas menyatakan status mana yang ditampilkannya.
- **`ClaimStatus` milik bounded context tim lain** (T-5). Salah satu dari "empat konsep" sebenarnya
  berada di luar batas kepemilikan Claim PNC, dan hubungannya dengan ADR-0006 belum dirumuskan.
- **`StatusPosisi` tidak terbukti punya lebih dari satu nilai** di export (T-6). Bila di produksi
  ia memang hanya bernilai `'On Progress'`, sistem baru akan membawa kolom yang tidak pernah
  berubah.
- Pemetaan 33 kode lama ke penamaan baru harus lengkap dan tidak boleh meleset satu pun; kode yang
  tidak terpetakan akan membuat klaim historis tidak terbaca.

### Risiko yang diterima secara sadar

- Membawa empat konsep berarti membawa kerumitannya ke sistem baru selama masa hidup aplikasi,
  bukan hanya masa migrasi.
- Bukti untuk dua dari empat konsep lebih lemah daripada dua lainnya, dan keputusan tetap diambil
  atas dasar pernyataan pemilik bisnis.

## Pertanyaan terbuka

- **`SELECT DISTINCT STATUSPOSISI`** pada data produksi — masih ada di daftar permintaan ke DBA,
  untuk menyelesaikan T-6 secara empiris. Pemilik: DBA. Tidak menghalangi tiket mana pun, tetapi
  menentukan apakah kolom itu perlu dibawa.
- Bagaimana `Flag Klaim` yang dimiliki GISFW diperlakukan — dibaca dari snapshot (ADR-0006), atau
  dibaca langsung? Pemilik: Work Owner + Tim GISFW.
- Apa makna persis `0` dan `1` pada Flag Klaim? Pemilik: Work Owner. Belum terjawab sejak `D-18`.
