# 0003 — Alihkan modul satu per satu dengan Pega dan Go berjalan paralel

Status: Accepted
Tanggal keputusan: 2026-09-07    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-05`, `D-30`, `D-61`, `D-42` | `docs/verifikasi-bukti-adr.md` §10, §15
Terkait: ADR-0004, ADR-0027, seluruh modul

## Konteks

Claim PNC adalah sistem yang sedang melayani produksi. Menggantinya sekaligus berarti satu
tanggal ketika seluruh alur klaim — registrasi, estimasi, komite, akseptasi, pembayaran — berpindah
bersamaan, tanpa jalan kembali.

`D-05` menolak pendekatan itu: **Pega dan aplikasi Go berjalan paralel, modul dialihkan satu per
satu.** Konsekuensi yang sudah dicatat sejak keputusan itu diambil: dibutuhkan strategi berbagi
data dan status antara kedua sistem selama masa paralel — ditangani ADR-0004.

## Opsi yang dipertimbangkan

1. **Strangler Fig** — modul dialihkan bertahap, Pega menyusut sampai habis.
2. **Big bang cutover** — satu tanggal, seluruh aplikasi berpindah.
3. **Jalan paralel penuh** — kedua sistem menerima input yang sama, hasilnya dibandingkan
   terus-menerus sampai kepercayaan terbentuk.

## Keputusan

Migrasi memakai pola **Strangler Fig**: aplikasi Go mengambil alih modul **satu per satu dalam
gelombang**, sementara Pega tetap melayani modul yang belum dialihkan. Kedua sistem berjalan di
atas **database yang sama** (ADR-0004) selama masa paralel.

Sebuah modul dinyatakan pindah hanya setelah melewati **dua gerbang** (ADR-0027): uji kesetaraan
otomatis, lalu UAT pengguna bisnis.

## Rationale

Pola ini membuat setiap langkah dapat dibatalkan. Bila satu modul gagal di gerbang mana pun,
yang dikembalikan hanya modul itu — bukan seluruh migrasi, dan bukan pula operasional harian
yang sedang berjalan.

Big bang tidak dapat dipertanggungjawabkan pada sistem yang menangani uang klaim dengan **tanpa
jejak audit atas perubahan nilai di sistem lama** (T-14): bila ada selisih setelah cutover, tidak
ada sumber untuk menelusurinya.

Jalan paralel penuh — kedua sistem menerima input yang sama — akan menggandakan beban kerja
pengguna operasional, dan `C-9`/`R-15` sudah mencatat bahwa **waktu pengguna bisnis untuk UAT
saja belum dialokasikan resmi**.

## Konsekuensi

### Positif

- Setiap gelombang memiliki titik kembali yang jelas.
- Risiko terdistribusi ke banyak rilis kecil, bukan menumpuk di satu tanggal.
- Tim belajar pada modul berisiko rendah sebelum menyentuh modul bernilai uang.

### Negatif / utang teknis

- **Dua sistem harus dipelihara bersamaan** selama seluruh masa transisi, termasuk memperbaiki
  cacat di Pega yang sebenarnya akan segera ditinggalkan.
- **Masa paralel menambah pekerjaan yang tidak menghasilkan fitur**: sinkronisasi status, aturan
  penulis tunggal per tabel, dan perkakas uji kesetaraan `S-8`.
- Skema database tidak boleh berubah bebas selama masa paralel (`D-63`) — setiap perubahan
  menempuh tiga pihak dan memperlambat iterasi.
- Semakin lama masa paralel, semakin besar biayanya. Tanpa tanggal akhir yang mengikat, pola ini
  dapat berlangsung jauh lebih lama daripada rencana.

### Risiko yang diterima secara sadar

- `D-61` menetapkan jadwal seluruh modul **tidak berubah** meski hasil analisis Fase 1
  menunjukkan 32 dari 33 modul belum berstatus siap penuh (§15). Selisih antara jadwal dan
  kesiapan itu **diterima sebagai tanggung jawab manajemen**, bukan diselesaikan di tingkat
  teknis.
- Pengguna bekerja di dua aplikasi sekaligus selama masa transisi, dengan tampilan yang mirip
  tetapi tidak identik.

## Pertanyaan terbuka

- Apa tanggal akhir yang mengikat bagi masa paralel — kapan Pega dimatikan? Pemilik: Work Owner.
  Tanpa ini, biaya pemeliharaan ganda tidak berbatas.
- Bila sebuah gelombang gagal di gerbang 2, apakah gelombang berikutnya tetap berjalan sesuai
  jadwal `D-61`? Pemilik: Work Owner + manajemen.
