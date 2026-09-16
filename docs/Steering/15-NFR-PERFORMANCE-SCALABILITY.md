# Non Functional Requirements, Performance, Scalability & Maintainability

---

## 1. Profil beban

Ini yang menentukan seluruh keputusan di dokumen ini:

| Aspek | Angka | Sumber |
|---|---|---|
| Pengguna aktif harian | 200–300 | Pemilik project |
| Klaim baru | Ribuan per bulan | D-10 |
| Data historis | **Puluhan juta baris** | D-10 |
| Ketersediaan | **24/7** | D-27 |
| Lingkungan | 2 VM on-premise | D-08 |

> **Profilnya: data besar, konkurensi rendah.**
>
> 200–300 pengguna bersamaan bukan beban berat bagi Go — satu instance saja mampu menanganinya
> tanpa kesulitan. Yang berat adalah **query terhadap puluhan juta baris**, terutama inbox
> berkolom banyak dan laporan lintas periode.
>
> Seluruh upaya optimasi diarahkan ke **volume data**, bukan ke jumlah permintaan per detik.
> Salah membaca ini akan membuat tim mengoptimalkan hal yang tidak menjadi hambatan.

---

## 2. Target Non Functional

### 2.1 Performa

| Jenis operasi | Target (persentil 95) | Catatan |
|---|---|---|
| Buka layar sederhana | < 1 detik | — |
| Inbox dan pencarian | < 3 detik | Terhadap data produksi penuh |
| Simpan registrasi klaim | < 3 detik | Termasuk seluruh validasi |
| Perhitungan spreading | < 1 detik | — |
| Laporan interaktif | < 10 detik | — |
| Export besar (PDF/Excel/CSV) | Asinkron | Tidak menahan pengguna; diberi tahu saat selesai |
| Pemanggilan sistem eksternal | Batas waktu 30 detik | Kegagalan ditangani, bukan menggantung |

### 2.2 Ketersediaan

| Aspek | Target | Konsekuensi |
|---|---|---|
| Ketersediaan | 24/7 (D-27) | Minimal 2 instance, rolling deployment |
| Downtime terencana | Nol untuk deployment aplikasi | Migrasi skema wajib backward-compatible |
| Downtime tak terencana | Sekecil mungkin | Kegagalan satu instance tidak menghentikan layanan |
| Pemulihan bencana | Mengikuti standar korporat (D-29) | Perlu diperiksa apakah sejalan dengan 24/7 |

### 2.3 Keamanan
Lihat `11-SECURITY.md`. Ringkasnya: autentikasi lewat HCC/HCQ, otorisasi milik aplikasi,
parameter binding tanpa perkecualian, data sensitif tidak masuk log, jejak audit append-only.

### 2.4 Auditabilitas
Wajib per D-28. Setiap perubahan bernilai bisnis tercatat permanen: siapa, kapan, nilai sebelum,
nilai sesudah. Retensi mengikuti ketentuan Compliance — masih terbuka.

---

## 3. Strategi performa

### 3.1 Hambatan yang sebenarnya

Diurutkan dari yang paling mungkin menjadi masalah:

| Peringkat | Hambatan | Penanganan |
|---|---|---|
| 1 | Query inbox terhadap puluhan juta baris | Index yang tepat + keyset pagination + index parsial |
| 2 | Laporan lintas periode panjang | Pool koneksi terpisah + eksekusi asinkron + streaming |
| 3 | Query N+1 saat memuat objek dan coverage | Muat sekaligus per batch, tidak per baris |
| 4 | Export besar dimuat seluruhnya ke memori | Streaming baris demi baris |
| 5 | Pemanggilan sistem eksternal yang lambat | Batas waktu, circuit breaker, pemanggilan asinkron |
| — | Jumlah permintaan per detik | **Bukan hambatan** pada 200–300 pengguna |

### 3.2 Aturan performa yang mengikat

1. **Paginasi selalu server-side.** Tidak pernah mengambil seluruh baris lalu memotongnya di
   aplikasi maupun di browser.
2. **Keyset pagination untuk inbox dan pencarian.** `OFFSET` bernilai besar memaksa database membaca dan
   membuang setengah juta baris.
3. **Tidak ada `COUNT(*)` atas tabel besar sebagai bagian permintaan biasa.** Frontend memakai
   pola "muat lebih banyak", bukan nomor halaman dengan total.
4. **Tidak ada query di dalam perulangan.** Muat sekaligus per batch.
5. **`SELECT` menyebutkan kolom.** Mengambil kolom `CLOB` yang tidak dipakai sangat mahal.
6. **Export dan laporan besar berjalan asinkron dan streaming.** Memori tetap datar berapa pun
   jumlah barisnya.
7. **Pool koneksi terpisah untuk laporan** agar satu laporan berat tidak menghabiskan koneksi
   transaksi.
8. **Setiap query baru wajib diperiksa rencana eksekusinya** terhadap data sebesar produksi
   sebelum merge.

> **Kapasitas laporan tidak punya dasar historis (2026-09-14).** `pyMaxRecords=500` terpasang pada
> **54 dari 56** laporan sistem lama, sehingga **kebutuhan export bervolume besar belum pernah
> benar-benar dilayani**. Menghapus batas itu adalah **penambahan kemampuan**, bukan penyalinan —
> dan rancangan kapasitasnya dibuat **tanpa data historis yang sahih**, karena angka pemakaian
> selama ini selalu terpotong di 500.
>
> Dua hal karenanya belum dapat ditetapkan secara terukur: **berapa baris maksimum yang wajib
> dilayani satu export**, dan apakah export besar dijalankan serentak dengan permintaan pengguna
> atau **diantrekan** lalu diberitahukan saat selesai. Keduanya pertanyaan terbuka `ADR-0011`, dan
> tanpa jawabannya `S-2` tidak dapat dinyatakan selesai secara terukur.
>
> Masalah nyata yang terukur justru di sisi antarmuka: **3.189 grid terikat page list klipboard**.

Aturan terakhir yang paling sering diabaikan, padahal paling murah. Query yang cepat terhadap
seribu baris bisa sangat lambat terhadap sepuluh juta — dan perbedaannya tidak akan terlihat di
lingkungan pengembangan.

### 3.3 Caching

Sistem lama berjalan **tanpa caching sama sekali** (7 data page seluruhnya `refresh=never`).
Kita tidak menambahkan kerumitan caching kecuali terbukti perlu.

| Data | Strategi |
|---|---|
| Master data (cabang, penyebab kerugian, mata uang, jenis treaty) | **Cache in-process**, disegarkan berkala atau saat diubah. Jarang berubah, sering dibaca |
| Izin pengguna | Cache in-process berumur pendek (menit) agar pencabutan hak cepat berlaku |
| Data klaim | **Tidak di-cache.** Harus selalu mutakhir |
| Hasil laporan | Tidak di-cache di awal; dipertimbangkan bila terbukti perlu |

**Tidak memakai cache terdistribusi (Redis).** Dengan hanya dua instance dan master data yang
kecil, cache in-process di masing-masing instance sudah memadai dan jauh lebih sederhana untuk
dioperasikan di VM on-premise.

---

## 4. Scalability

### 4.1 Skala saat ini
Dua instance sudah lebih dari cukup untuk 200–300 pengguna. Keduanya ada **karena tuntutan
ketersediaan 24/7** (D-27), bukan karena kebutuhan kapasitas.

### 4.2 Bila beban bertambah

Urutan langkah, dari yang termurah:

| Langkah | Kapan | Biaya |
|---|---|---|
| 1. Perbaiki query dan index | Selalu lebih dulu | Rendah |
| 2. Tambah cache master data | Bila master sering dibaca | Rendah |
| 3. Tambah instance aplikasi | Bila CPU aplikasi jenuh | Rendah — aplikasi stateless |
| 4. Read replica untuk laporan | Bila laporan mengganggu transaksi | Sedang |
| 5. Partisi tabel per tahun | Bila tabel klaim/audit sangat besar | Sedang |
| 6. Arsip data lama | Bila data historis menghambat | Sedang |

**Aplikasi stateless sejak awal** membuat langkah 3 hanya soal menambah VM — tidak perlu
perubahan kode. Inilah manfaat nyata dari syarat stateless yang sudah dituntut D-27.

### 4.3 Yang sengaja tidak disiapkan
Sharding, microservices, message broker, autoscaling. Seluruhnya menambah kerumitan operasional
yang nyata tanpa menyelesaikan masalah yang kita punya. Bila kelak dibutuhkan, seam yang sudah
ada (Future Architecture §3) membuatnya bisa ditambahkan tanpa membongkar domain.

---

## 5. Maintainability

Ini yang menentukan apakah sistem masih bisa dirawat tiga tahun lagi. Untuk tim di D-09,
maintainability lebih penting daripada kecanggihan.

### 5.1 Yang membuatnya terawat

| Hal | Bagaimana | Menghilangkan masalah lama |
|---|---|---|
| **Satu aturan hidup di satu tempat** | Aturan ketergantungan antar lapisan | Aturan tersebar di activity, SQL, dan stored procedure |
| **Bahasa yang seragam** | `CONTEXT.md` dipakai di kode, API, dan UI | Alias kolom menyesatkan; `Adjustment` yang berarti nilai penyelesaian |
| **Nilai bisnis dapat diubah tanpa deploy** | Master data (D-15) | Email dan ambang di-hardcode |
| **Satu komponen tabel** | Pustaka komponen baku (U-2) | 268 grid dengan pola berulang |
| **Struktur yang dapat ditebak** | Struktur folder preskriptif | — |
| **Aturan ditegakkan otomatis** | Lint, `depguard`, pemeriksaan pola SQL | Konvensi yang hanya ada di dokumen akan dilanggar |
| **Test sebagai dokumentasi aturan** | Setiap aturan bisnis punya test bernama kalimat bisnis | Aturan hanya diketahui dari membaca 137 step |

### 5.2 Ukuran yang dipantau

| Ukuran | Batas | Tindakan bila terlampaui |
|---|---|---|
| Panjang fungsi | ~80 baris | Pecah |
| Panjang berkas | ~400 baris (Go), ~200 baris (komponen React) | Pecah |
| Ketergantungan antar lapisan | Nol pelanggaran | Merge diblokir |
| Cakupan test aturan bisnis | Seluruh aturan terdokumentasi punya test | Lengkapi sebelum merge |
| Jumlah dependensi pihak ketiga | Ditinjau setiap penambahan | Butuh alasan tertulis |

### 5.3 Dokumentasi yang harus tetap hidup

| Dokumen | Diperbarui saat |
|---|---|
| `CONTEXT.md` | Ada istilah domain baru atau berubah artinya |
| `00-DECISION-LOG.md` | Ada keputusan arsitektur baru |
| Kontrak OpenAPI | Ada perubahan API |
| Dokumen Steering terkait | Ada perubahan strategi |

Dokumentasi yang tidak diperbarui lebih buruk daripada tidak ada dokumentasi, karena orang
mempercayainya. Karena itu daftar di atas sengaja pendek — hanya yang benar-benar akan dibaca.
