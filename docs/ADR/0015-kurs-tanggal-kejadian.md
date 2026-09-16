# 0015 — Konversi mata uang memakai kurs tanggal kejadian; tolak klaim bila kurs tidak ditemukan

Status: Accepted
Tanggal keputusan: 2026-09-11    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-48`, `R-19` | `Database/GETCURRENCYSTANDARD.fnc:3`, `:14`, `:20-22` | `m_currencystandard`
Terkait: CONTEXT.md#Kurs-Standar, ADR-0014, ADR-0017, ADR-0027, modul `B-2`, `B-5`

## Konteks

Nilai klaim dalam valuta asing dikonversi ke Rupiah sebelum dibandingkan dengan ambang komite
(ADR-0014) dan sebelum masuk laporan. Basis kurs karenanya menentukan **berapa banyak orang yang
harus menyetujui sebuah klaim** — bukan sekadar angka tampilan.

Fungsi konversi yang ada, `Database/GETCURRENCYSTANDARD.fnc`, memakai kurs **hari eksekusi**, dan
pada jalur tertentu (`:20-22`) mengembalikan `1` ketika kurs tidak ditemukan. Nilai `1` berarti
satu satuan valuta asing dihitung setara satu Rupiah: klaim bernilai besar menyusut menjadi
kecil, lalu **lolos tanpa komite**.

Kegagalan itu senyap. Tidak ada galat, tidak ada catatan, dan hasilnya tampak seperti angka wajar.

## Opsi yang dipertimbangkan

1. **Kurs pada tanggal kejadian**, tolak bila tidak ditemukan.
2. **Kurs hari eksekusi** — sama dengan sistem lama.
3. Kurs tanggal kejadian, dengan **nilai bawaan** bila kurs tidak ada.

## Keputusan

Konversi ke IDR memakai **kurs yang berlaku pada tanggal kerugian**, bukan kurs hari eksekusi.

Bila kurs untuk mata uang dan tanggal itu **tidak ditemukan**, transaksi **ditolak dengan galat
eksplisit** yang menyebutkan mata uang dan tanggalnya. **Tidak ada nilai bawaan, dan tidak ada
kurs pengganti.**

## Rationale

Klaim dinilai menurut keadaan pada saat kejadian — sama seperti polis dibekukan pada saat
registrasi (ADR-0006). Memakai kurs hari eksekusi membuat nilai Rupiah sebuah klaim **berubah
hanya karena prosesnya terlambat**, dan dengan itu mengubah jenjang komite yang harus
menyetujuinya.

Nilai bawaan `1` adalah bentuk terburuk dari kegagalan senyap: ia mengubah klaim besar menjadi
kecil tepat pada titik yang menentukan kewenangan persetujuan. Menolak secara eksplisit menukar
kesalahan yang tak terlihat dengan gangguan yang terlihat — dan gangguan yang terlihat dapat
diperbaiki.

## Konsekuensi

### Positif

- Nilai Rupiah sebuah klaim menjadi stabil: tidak berubah karena keterlambatan proses.
- Jenjang komite menjadi konsisten dan dapat diulang hasilnya.
- Kegagalan data kurs menjadi terlihat, bukan tersembunyi di balik angka yang tampak wajar.

### Negatif / utang teknis

- **Uji kesetaraan akan menampilkan selisih pada seluruh data historis valuta asing**, karena
  laporan lama memakai kurs hari eksekusi. Selisih ini **wajib dinyatakan lebih dulu sebagai
  perbaikan yang direncanakan** (ADR-0017), bukan ditemukan sebagai kejutan saat pengujian.
- **Klaim valuta asing yang dulu lolos kini dapat ditolak** sampai kursnya dilengkapi. Ini
  perbaikan yang diinginkan, tetapi berdampak operasional dan perlu disiapkan sebelum gerbang 1
  dijalankan.
- Master kurs menjadi data kritis: kelengkapannya per tanggal kini menentukan apakah klaim dapat
  diproses sama sekali. Proses pengisiannya — siapa, kapan, dari sumber apa — **belum ada**.
- Klaim dengan tanggal kejadian jauh di masa lalu menuntut kurs historis yang mungkin tidak
  pernah tercatat.

### Risiko yang diterima secara sadar

- Penolakan pada saat kurs tidak tersedia dapat menghentikan pemrosesan klaim di jam kerja, dan
  pemulihannya bergantung pada pihak yang mengisi master kurs.
- Perubahan basis kurs mengubah nilai historis dalam laporan; angka lama dan baru untuk klaim yang
  sama tidak akan cocok.

## Pertanyaan terbuka

- **Isi tabel `m_currencystandard` belum ada di repo** — masih diminta ke DBA (`R-19`). Tanpa itu
  `B-5` tidak dapat diuji. Pemilik: DBA.
- Siapa yang mengisi kurs harian dan dari sumber apa (Bank Indonesia, kurs korporat, atau lain)?
  Pemilik: Work Owner. Menghalangi penyelesaian tiket `F-4`.
- Bagaimana klaim yang tertolak karena kurs kosong diperlakukan — ditahan dan diproses ulang
  otomatis, atau dikembalikan ke petugas? Pemilik: Work Owner.
