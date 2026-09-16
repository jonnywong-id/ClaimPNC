# 0013 — Tentukan pengganti pola hapus-lalu-sisip-ulang pada konversi klaim

Status: Proposed
Tanggal keputusan: —    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner + Lead Engineer
Jejak bukti: `D-66` | `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503`, `:497`, `:498`, `:1296`, `:1325`
Terkait: ADR-0012, ADR-0007, ADR-0027, modul `B-2`

> **Belum diputuskan.** Berkas ini memuat konteks, opsi, dan konsekuensi masing-masing opsi —
> **tanpa bagian `Keputusan`**. Jangan dijadikan dasar implementasi.

## Konteks

Sistem lama menjamin konversi klaim dapat diulang tanpa menggandakan data dengan cara yang paling
langsung: **menghapus dulu, lalu menyisipkan ulang**.
`Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503` menghapus **12 tabel** milik satu klaim,
kemudian menyisipkan ulang seluruh pohon datanya.

Pola itu **bergantung pada penghapusan fisik**, sementara ADR-0012 menetapkan tidak ada
penghapusan fisik pada data bernilai bisnis. Keduanya tidak dapat berlaku bersamaan.

Ada cacat yang sudah berjalan hari ini dan harus ikut ditutup oleh apa pun penggantinya: pada dua
tabel — `T_DLALIST` dan `T_PLALIST` — **baris `DELETE`-nya sudah dikomentari** (`:497`, `:498`)
sementara `INSERT`-nya tetap aktif (`:1296`, `:1325`). Artinya konversi ulang pada kedua tabel itu
**sudah berpotensi menduplikasi baris sekarang**, sebelum perubahan apa pun dilakukan.

## Opsi yang dipertimbangkan

**Opsi 1 — *Upsert* berbasis kunci alami.** Setiap baris dikenali dari kunci bisnisnya (misalnya
nomor klaim + objek + coverage + urutan); konversi ulang memperbarui baris yang sudah ada dan
menyisipkan yang belum.

- Menjaga jumlah baris tetap sama dengan sistem lama, sehingga uji kesetaraan lebih mudah dibaca.
- Menuntut **kunci alami yang benar-benar unik pada ke-12 tabel** — dan keunikan itu belum
  diverifikasi. Bila ada tabel tanpa kunci alami, opsi ini tidak dapat dipakai di sana.
- Tidak menyimpan riwayat: hasil konversi sebelumnya tertimpa.

**Opsi 2 — Versioning dengan penanda baris aktif.** Konversi ulang menyisipkan generasi baru dan
menandai generasi sebelumnya tidak aktif.

- Konsisten penuh dengan ADR-0012 dan ADR-0026: tidak ada yang hilang, semua perubahan terjejak.
- **Menggandakan pertumbuhan 12 tabel** setiap kali konversi diulang, di atas data historis
  puluhan juta baris (`D-10`).
- Setiap kueri pembaca ke-12 tabel harus menyaring baris aktif — 12 tabel × seluruh pembacanya.

**Opsi 3 — Konversi sekali saja, pengulangan dilarang.** Klaim yang sudah dikonversi tidak boleh
dikonversi ulang; koreksi ditempuh lewat jalur perbaikan data biasa.

- Paling sederhana dan paling murah.
- Menghapus kemampuan yang ada sekarang. Bila konversi ulang ternyata dipakai operasional untuk
  memperbaiki klaim bermasalah, opsi ini memutus jalan itu — dan **seberapa sering pengulangan
  benar-benar terjadi belum diketahui**.

## Konsekuensi bila dibiarkan tidak diputuskan

- **Tiket `B-2` tidak dapat ditulis lengkap.** Kriteria penerimaan untuk konversi ulang tidak
  dapat dirumuskan tanpa mengetahui pola penggantinya.
- Cacat duplikasi pada `T_DLALIST` dan `T_PLALIST` tetap terbuka, baik di sistem lama maupun baru.
- Uji kesetaraan `S-8` untuk `B-2` tidak dapat dirancang, karena perilaku yang dibandingkan belum
  ditentukan.

## Pertanyaan terbuka

1. **Seberapa sering konversi ulang benar-benar dijalankan di produksi, dan untuk keperluan apa?**
   Pemilik: Work Owner. Jawaban ini menentukan apakah Opsi 3 layak dipertimbangkan sama sekali.
2. **Apakah ke-12 tabel punya kunci alami yang unik?** Pemilik: DBA + Lead Engineer. Menentukan
   apakah Opsi 1 dapat dipakai seragam atau hanya sebagian.
3. **Apakah riwayat hasil konversi sebelumnya perlu disimpan?** Pemilik: Work Owner + Compliance.
   Bila ya, hanya Opsi 2 yang memenuhi.
4. Duplikasi `T_DLALIST`/`T_PLALIST` yang sudah terjadi hari ini — apakah datanya perlu
   dibersihkan sebelum migrasi? Pemilik: Work Owner + DBA.
