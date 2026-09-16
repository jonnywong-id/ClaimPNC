# 0011 — Bangun pembuatan PDF, Excel, dan CSV di dalam aplikasi Go

Status: Accepted
Tanggal keputusan: 2026-09-07    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-11`, `D-10` | 56 Report Definition · 78 activity laporan | `pyMaxRecords=500` pada 54 dari 56 laporan (T-12)
Terkait: ADR-0001, ADR-0002, modul `S-2`, `S-7`, `U-5`

## Konteks

Laporan adalah bagian besar aplikasi ini: **56 Report Definition** dan **78 activity** yang
melayaninya. Di sistem lama, pembuatan PDF dan Excel dikerjakan engine reporting Pega — komponen
yang lenyap bersama platformnya.

Dua temuan Fase 1 membentuk keputusan ini:

- **`pyMaxRecords=500` pada 54 dari 56 laporan.** Laporan yang ada hari ini dibatasi 500 baris,
  bukan karena kebutuhan bisnis melainkan karena batas klipboard Pega.
- **`OFFSET` nol kemunculan** di seluruh export (T-12). Tidak ada laporan yang benar-benar
  memaginasi hasil besar; yang ada adalah pemotongan pada 500 baris.

Artinya: kebutuhan export bervolume besar **belum pernah benar-benar dilayani** sistem lama, dan
membangunnya di sistem baru adalah penambahan kemampuan, bukan penyalinan.

## Opsi yang dipertimbangkan

1. **Bangun sendiri di dalam aplikasi Go** memakai pustaka pembuat PDF/Excel.
2. **Pakai tools BI eksternal** (Metabase, Superset, atau sejenisnya).
3. **Pakai engine reporting berbayar** yang dipasang terpisah.

## Keputusan

Pembuatan **PDF, Excel, dan CSV diimplementasikan sendiri di dalam aplikasi Go**. Tidak memakai
engine reporting Pega maupun tools BI eksternal.

Export bervolume besar ditangani dengan cara yang **tidak membebani transaksi** — dialirkan
(*streaming*), bukan disusun seluruhnya di memori lebih dulu.

## Rationale

Tools BI eksternal menambah satu komponen yang harus dipasang, diamankan, dan diberi hak akses ke
database — bertentangan dengan `D-08` dan menambah permukaan yang harus dipelihara tim kecil.

Laporan di aplikasi ini bukan laporan analitis bebas bentuk; ia adalah **56 laporan dengan bentuk
tetap** yang sudah diketahui. Kebutuhan seperti itu dilayani kode biasa dengan baik, tanpa engine.

Membangun sendiri juga menjaga otorisasi tetap di satu tempat: laporan melewati pemeriksaan izin
yang sama dengan layar (ADR-0023), bukan jalur terpisah yang mudah terlupakan.

## Konsekuensi

### Positif

- Tidak ada komponen tambahan di produksi.
- Otorisasi laporan memakai mekanisme yang sama dengan seluruh aplikasi.
- Batas 500 baris dapat dihapus — laporan akhirnya dapat melayani permintaan yang sebenarnya.

### Negatif / utang teknis

- **Tata letak PDF harus dibangun satu per satu.** Untuk 56 laporan, ini pekerjaan besar yang
  mudah diremehkan, dan seluruhnya harus cocok dengan keluaran lama agar lolos gerbang 1.
- **Permintaan perubahan laporan menjadi permintaan perubahan kode**, bukan konfigurasi. Pengguna
  bisnis kehilangan kemampuan mengubah laporan sendiri — bila selama ini mereka memilikinya.
- Export besar di satu instans memakan memori dan CPU yang sama dengan pelayanan transaksi
  (ADR-0001). Tanpa pembatasan, satu export dapat memperlambat seluruh pengguna.
- **Menghapus batas 500 baris adalah perubahan perilaku**, bukan pemeliharaan. Laporan yang dulu
  terpotong kini utuh — hasilnya berbeda, dan uji kesetaraan akan menandainya sebagai selisih.

### Risiko yang diterima secara sadar

- Pustaka PDF di ekosistem Go tidak sematang engine reporting komersial; sebagian tata letak rumit
  mungkin menuntut kompromi visual.
- Volume sebenarnya yang diminta pengguna **tidak diketahui**, karena selama ini selalu terpotong
  di 500 baris. Rancangan kapasitas dibuat tanpa data historis yang sahih.

## Pertanyaan terbuka

- Berapa baris maksimum yang wajib dilayani satu export? Pemilik: Work Owner. Tanpa angka ini,
  `S-2` tidak dapat dinyatakan selesai secara terukur.
- Apakah export besar dijalankan serentak dengan permintaan pengguna, atau diantrekan dan
  diberitahukan saat selesai? Pemilik: Work Owner + Lead Engineer.
- Apakah keluaran PDF wajib identik secara visual dengan keluaran Pega, atau cukup identik secara
  isi? Pemilik: Work Owner. Ini menentukan kriteria kelulusan gerbang 1 untuk `S-2`.
