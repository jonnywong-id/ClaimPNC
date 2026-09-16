# Future Enhancement

Perbaikan yang **sengaja tidak dikerjakan selama migrasi**, agar prinsip P-5 (perilaku
dipertahankan lebih dulu, diperbaiki kemudian) tetap terjaga.

Alasan menundanya bukan karena tidak penting, melainkan karena selama migrasi kita harus bisa
membedakan **bug** dari **perbaikan yang tidak tercatat**. Bila perilaku diperbaiki sambil jalan,
setiap perbedaan hasil antara Pega dan sistem baru menjadi tidak bisa dijelaskan — dan
verifikasi kesetaraan kehilangan maknanya.

Daftar ini ditulis sekarang justru agar temuan selama analisis tidak hilang.

---

## 1. Setelah migrasi stabil

### 1.1 Perbaikan pengalaman pengguna
D-13 menetapkan tampilan meniru Pega agar pengguna tidak perlu belajar ulang. Setelah sistem
baru stabil dan pengguna terbiasa, ruang perbaikan yang teridentifikasi:

- **Inbox berkolom banyak** terasa padat pada sebagian layar. Kolom yang dapat dipilih pengguna dengan preset per
  peran akan jauh lebih terbaca.
- **Form registrasi sangat panjang** dengan puluhan field. Dapat dipecah menjadi langkah-langkah
  dengan penyimpanan draf.
- **Pesan validasi** dapat ditampilkan langsung di sebelah field saat pengguna mengetik, bukan
  hanya setelah submit.
- **Layar surveyor** dapat dioptimalkan khusus untuk penggunaan lapangan (D-12): unggah foto
  yang tahan koneksi terputus, dan mode offline.

### 1.2 Penyederhanaan model status
D-18 menetapkan empat konsep status dipertahankan karena memang berbeda. Setelah arti kode
`1134`–`1166` diketahui (R-06 tertutup) dan pola pemakaiannya terlihat dari data nyata, layak ditinjau
apakah keempatnya masih perlu terpisah — atau sebagian dapat diturunkan dari yang lain.

Ini peninjauan berbasis data, bukan asumsi. Karena itu harus menunggu sampai ada data.

### 1.3 Membersihkan duplikasi lini bisnis
Sistem lama menggandakan pola `Browse*`, `Insert*`, `Grouping*`, `CreateCasePNC*` untuk tiap
lini bisnis (`_AsuransiKredit`, `_AutoClaim`, `_Travel`, `_Kredit_PA`). Migrasi mempertahankan
perbedaan perilakunya apa adanya.

Setelah stabil, layak ditinjau mana yang **benar-benar berbeda secara bisnis** dan mana yang
hanya hasil salin-tempel. Yang kedua dapat disatukan menjadi satu implementasi dengan parameter
lini bisnis.

---

## 2. Peluang teknis

### 2.1 Menyelesaikan perpindahan ke PostgreSQL
Rencana lengkapnya di `09-DATABASE-STRATEGY.md` §10. Karena SQL sudah portabel (D-20) dan tidak
ada pemanggilan stored procedure (D-02), perpindahan ini **tidak menyentuh logika bisnis**.

Setelah pindah, fitur PostgreSQL yang dapat dimanfaatkan: index parsial untuk inbox, index GIN
untuk pencarian di dalam JSONB snapshot polis, dan partisi deklaratif untuk tabel klaim dan
audit.

### 2.2 Read replica untuk laporan
Bila laporan mulai mengganggu transaksi pengguna, replika baca adalah langkah berikutnya yang
paling sepadan (NFR §4.2 langkah 4). Pool koneksi terpisah yang sudah dirancang sejak awal
membuat perubahan ini hanya soal konfigurasi.

### 2.3 Arsip data historis
Dengan puluhan juta baris yang terus tumbuh (D-10), pemindahan klaim lama yang sudah tutup ke
tabel arsip akan menjaga tabel aktif tetap ramping. Dilakukan berdasarkan pengukuran, bukan
dugaan.

### 2.4 Pencarian teks
Pencarian klaim saat ini memakai `LIKE '%...%'` yang tidak dapat memanfaatkan index. Setelah
pindah ke PostgreSQL, pencarian teks penuh bawaan PostgreSQL dapat menggantikannya tanpa
komponen tambahan.

---

## 3. Peluang bisnis yang terlihat dari analisis

Berikut ditemukan saat membaca source dan patut dipertimbangkan pemilik bisnis — bukan usulan
teknis.

### 3.1 Dashboard TAT yang dapat ditindaklanjuti
Sistem sudah menghitung TAT dan menyimpan `PNC_CHRONOLOGYTAT`, tetapi hanya menampilkannya
sebagai laporan. Dengan data yang sama, sistem dapat **memberi peringatan sebelum tenggat
terlampaui**, bukan melaporkan setelah terlewat.

### 3.2 Deteksi klaim mencurigakan
Sistem sudah punya peran Investigator, tabel `T_CLAIM_DATA_RESULTS_AI`, dan `M_DOMINAN_FACTOR` —
menunjukkan pernah ada upaya ke arah ini. Dengan data historis puluhan juta baris, pola klaim
mencurigakan dapat diangkat secara otomatis untuk ditinjau, bukan hanya berdasarkan kecurigaan
petugas.

### 3.3 Portal tertanggung
Saat ini status klaim hanya dapat dilihat petugas. Portal untuk tertanggung melihat status
klaimnya sendiri akan mengurangi beban Service Center. Perlu pertimbangan keamanan tersendiri
karena melibatkan akses dari luar jaringan internal.

### 3.4 Otomasi kelengkapan dokumen
Sistem sudah punya master jenis dokumen per lini bisnis (`LST_TYPE_DOC_BUSINESS`,
`V_LST_DOC_TYPE`). Pemeriksaan kelengkapan dokumen dapat berjalan otomatis dan memberi tahu
tertanggung apa yang masih kurang.

---

## 4. Utang teknis yang sudah dijadwalkan diselesaikan

Untuk kejelasan — yang berikut **tidak** ditunda, melainkan **sudah masuk scope migrasi**:

| Utang teknis | Diselesaikan oleh |
|---|---|
| Nilai bisnis di-hardcode | D-15 — master data |
| Blok `// TESTING` di jalur produksi | D-15 — tidak dibawa |
| Penambahan 7 jam manual | F-5 — modul Clock |
| Perangkaian SQL `{ASIS:...}` | Coding Standards — parameter binding wajib |
| Alias kolom menyesatkan | D-19 — penamaan ulang |
| Kunci Pega di data bisnis | D-22 + D-71 — format `PNCN.YY.xxxx` |
| Perilaku bergantung hostname | Cross-Cutting §3.4 — dilarang |
| Tanpa jejak audit | D-28 — dirancang sejak awal |
| Pemanggilan stored procedure | D-02 — logika naik ke Go |
| DB Link lintas database | D-25 — diganti API |
