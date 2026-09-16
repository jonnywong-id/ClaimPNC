# Migration Strategy — Claim PNC

Strategi pemindahan dari Pega ke Go + React. Mengikuti **Strangler Fig** (D-05): Pega dan Go
berjalan paralel, modul dialihkan satu per satu.

> **Diperbarui v2.0 (2026-09-14).** Daftar perbaikan eksplisit `P-5` bertambah dari **4 menjadi
> 13 butir** (`D-49`) · Tahap 0 diperbarui dengan status nyata tiap permintaan, empat tertutup dan
> enam baru · nomor klaim menjadi `PNCN.YY.xxxx` (`D-71`) · dan verifikasi kesetaraan kini
> dikerjakan modul **`S-8`** yang masuk gelombang 1 (`D-42`).

---

## 1. Prinsip yang mengikat

### P-1 · Satu tabel hanya boleh ditulis satu sistem
Selama masa paralel, setiap tabel punya **tepat satu pemilik**. Modul yang sudah pindah ke Go
memiliki tabelnya dan Pega hanya membaca; modul yang belum pindah tetap dimiliki Pega dan Go
hanya membaca.

**Tidak ada sinkronisasi dua arah.** Dua sistem yang sama-sama menulis ke tabel yang sama akan
menghasilkan konflik data yang hampir mustahil dilacak, apalagi ketika keduanya punya aturan
validasi yang berbeda.

### P-2 · Batas modul mengikuti batas kepemilikan tabel
Konsekuensi langsung dari P-1: urutan migrasi tidak bebas. Sebuah modul hanya bisa dipindahkan
bila **seluruh tabel yang ia tulis** bisa ikut berpindah kepemilikan bersamanya.

### P-3 · Klaim yang sedang berjalan tidak berpindah sistem di tengah jalan
Klaim yang sudah dimulai di Pega **diselesaikan di Pega**. Klaim baru dimulai di Go setelah
modul registrasi dialihkan. Nomor `PNCN.YY.xxxx` versus `PNC-xxxx` (D-22, D-71) membuat asal setiap klaim
langsung terbaca tanpa tabel pemetaan.

Ini menghindari kelas bug terburuk dalam migrasi bertahap: klaim setengah jalan yang datanya
ditulis dua sistem dengan aturan berbeda.

### P-4 · Migrasi skema selalu backward-compatible
Karena target 24/7 (D-27) menuntut rolling deployment, versi lama dan baru aplikasi berjalan
bersamaan terhadap skema yang sama. Karena itu:

- Menambah kolom: boleh langsung, harus *nullable* atau punya *default*.
- Menghapus kolom: **dua tahap** — berhenti dipakai lebih dulu, dihapus pada rilis berikutnya.
- Mengganti nama kolom: **tidak pernah langsung** — tambah kolom baru, tulis ke keduanya,
  pindahkan pembacaan, baru hapus yang lama.
- Mengubah tipe kolom: lewat kolom baru, tidak pernah `ALTER` di tempat.

### P-5 · Perilaku dipertahankan lebih dulu, diperbaiki kemudian
Selama migrasi, **hasil yang benar adalah hasil yang sama dengan Pega** — kecuali untuk hal-hal
yang secara eksplisit diputuskan diperbaiki.

Perbaikan lain dicatat sebagai Future Enhancement, tidak dikerjakan sambil jalan. Alasannya:
bila hasil berbeda, kita harus bisa memastikan itu **bug**, bukan **perbaikan yang tidak
tercatat**.

**Daftar perbaikan eksplisit — 13 butir** (naik dari 4; `D-49`). Setiap selisih yang muncul pada
uji kesetaraan **wajib dapat dipetakan ke salah satu butir ini, atau dinyatakan sebagai bug**.

| # | Perbaikan | Sumber |
|---|---|---|
| 1 | Hardcode menjadi konfigurasi dan master data | `D-15` |
| 2 | Blok `// TESTING` yang menimpa email produksi dihapus | `D-15` |
| 3 | Penanganan zona waktu terpusat di `F-5` | `D-13`, `R-12` |
| 4 | Penamaan domain mengikuti `CONTEXT.md` | `D-19` |
| 5 | Toleransi spreading: pencocokan substring → `ROUND(SUM(share),4) BETWEEN 99.9999 AND 100.0001` | `D-49` #1, `D-51` |
| 6 | Ambang Rp 50 juta: **tiga operator berbeda** di tiga rule → satu operator | `D-49` #2 |
| 7 | Penyesuaian 7 jam yang **asimetris di dalam satu kondisi validasi** diperbaiki | `D-49` #3 |
| 8 | Kurs memakai **tanggal kejadian**, bukan hari eksekusi | `D-49` #4, `D-48` |
| 9 | Kurs tidak ditemukan → **klaim ditolak**, bukan `RETURN 1` | `D-49` #5, `D-48` |
| 10 | `LIMIT_TOP` menjadi **validasi integritas master**; tie-breaker acak dibatasi | `D-49` #6, `D-47` |
| 11 | `NilaiSalvage` **selalu** ditambahkan, bukan hanya bila baris terakhir kebetulan bertipe salvage | `D-49` #8 |
| 12 | `INSERT_SALVAGE` tidak lagi menulis `IDSALVAGE = NULL` saat update | `D-49` #9 |
| 13 | `GETSELISIHJAM` gagal tidak lagi mengembalikan `0` jam yang tak terbedakan dari nol | `D-49` #10 |

**Satu cacat sengaja direplikasi.** Tanggal PLA/DLA diambil dari **waktu sistem**, bukan dari
tanggal yang dipilih pengguna — parameter `TTGLPLADLA` yang diterima lalu dibuang **bukan cacat**,
melainkan perilaku yang memang benar (`D-49` #7). Sistem baru boleh menghapus parameter itu
seluruhnya.

**Dua perubahan yang bukan "perbaikan hasil" tetapi tetap mengubah keluaran**, dan harus
diperlakukan di sisi perkakas pembanding, bukan sebagai selisih: **soft delete menyeluruh**
(`D-66`) mengubah isi tabel tanpa mengubah apa yang dilihat pengguna, dan **transaksi atomik pada
`B-4`/`B-9`** (`D-68`) mengubah keadaan akhir saat terjadi kegagalan.

---

## 2. Tahapan

### Tahap 0 — Persiapan (tidak menghasilkan fitur, tapi memblokir semuanya)

Diperbarui 2026-09-14. Tanda ✅ menandai kegiatan yang **sudah selesai**.

| Kegiatan | Keluaran | Status |
|---|---|---|
| Minta source 64 procedure & function dari DBA | Logika bisnis yang tersembunyi terbaca (**R-01**) | ✅ **62 dari 64 diterima** |
| Minta **12 dependensi** yang dipanggil 62 procedure itu | `UPDATE_LOG_KONVERSI` (162×), `GETNEWID` (42×), … | **Terbuka** |
| Minta export Rule-Agent / Queue Processor dari Pega | Daftar job terjadwal (**R-02**) | ✅ **5 job + 1 agent** (`D-57`) |
| Minta DDL lengkap tabel `POOLDATA` dan `DATAPEGA` | Tipe kolom, index, constraint (**R-08**) | Terbuka |
| Ambil isi master `V_STS_CLAIM` | Arti kode status (**R-06**) | ✅ **33 kode `1134`–`1166`** |
| Ambil isi tabel `POOLDATA.EMAILKOMITE` + konfirmasi aturan `SetEmailKomite` | Tangga penjenjangan komite (D-14) | ✅ **21 kolom, 30 baris**; aturan berbasis nama orang **dicabut** (`D-52`, `D-70`) |
| **Permintaan export ulang berbasis Product rule** | ±242 rule hilang, **137 di antaranya When rule** (**R-16**) | **Terbuka** (`D-39`) |
| Inventarisasi 6 API pengganti DB Link bersama tim pemilik | Kontrak integrasi (**R-03**) | Terbuka |
| **Minta kontrak API HCC/HCQ** | Pengganti gerbang 1 untuk `F-3` (**R-14**) | **Terbuka** — `ADR-0024` |
| **Minta daftar peristiwa wajib audit dari Compliance** | Pengganti gerbang 1 untuk `S-5` | **Terbuka** — `ADR-0026` |
| **Konfirmasi Pega staging yang dapat ditembak dari luar** | Prasyarat `S-8` dan gerbang 1 seluruh modul | **Terbuka** — `ADR-0027` |
| **Tetapkan tujuan penyimpanan rahasia dan keputusan rotasi** | Menutup **R-17** | **Terbuka** — `D-40` |
| Sediakan lingkungan dev, staging, production | Pipeline deployment | Terbuka |
| Pelatihan tim: Go, React, TypeScript | Kesiapan tim (D-09) | Terbuka |

> Tahap ini **harus dimulai hari pertama** karena waktu tunggunya ada di luar kendali tim
> pengembang. Empat penghalang lama sudah tertutup; **enam penghalang baru** muncul dari
> verifikasi bukti, dan tiga di antaranya — kontrak HCC/HCQ, daftar peristiwa audit, dan Pega
> staging — **memblokir gerbang kelulusan**, bukan sekadar memperlambat pengerjaan.

### Tahap 1 — Fondasi
Modul F-1 sampai F-5, ditambah U-1 dan U-2 (kerangka SPA dan pustaka komponen).
**Belum ada perubahan bagi pengguna.** Pega masih menangani seluruh pekerjaan.
Keluaran: aplikasi Go yang bisa login, membaca database, dan menampilkan satu layar contoh.

### Tahap 2 — Baca dulu, tulis belakangan
Modul yang **hanya membaca** dialihkan lebih dulu: inbox, pencarian klaim, tampilan detail,
dan laporan.

Ini pilihan yang disengaja: modul baca **tidak melanggar P-1** karena tidak menulis apa pun,
sehingga bisa berjalan berdampingan dengan Pega tanpa risiko konflik data. Sekaligus menjadi
pembuktian arsitektur yang nyata — tim belajar Go dan React pada pekerjaan yang kesalahannya
tidak merusak data.

Bila hasil di Go berbeda dengan Pega pada layar yang sama, itu **bug yang harus diperbaiki
sebelum lanjut** — dan inilah cara paling murah menemukannya.

### Tahap 3 — Jalur klaim inti
`B-1 → B-2 → B-3 → B-4 → B-5`, ditambah B-6 (penugasan), B-14 (receive document), dan S-1
(dokumen).

Sejak titik ini, klaim baru bernomor `PNCN.YY.xxxx` dibuat di Go. Klaim `PNC-xxxx` yang sedang
berjalan tetap diselesaikan di Pega (P-3).

### Tahap 4 — Persetujuan dan penyelesaian
B-7 (komite), B-10 (akseptasi & pembayaran), B-11 (RCL/PUCL/compliance), B-8 (survey),
B-13 (open protection).

### Tahap 5 — Nilai dan pihak luar
B-9 (PLA/Pre-DLA/DLA), B-12 (salvage & recovery), S-3 (notifikasi), S-4 (integrasi eksternal).

### Tahap 6 — Laporan dan penutup
S-2 (56 laporan + engine dokumen), S-7 (dashboard), U-5, U-6, S-6 (scheduler).

### Tahap 7 — Penonaktifan Pega
Dilakukan hanya setelah **seluruh klaim `PNC-xxxx` yang masih berjalan selesai** atau
dipindahkan secara sadar. Pega dijadikan read-only lebih dulu selama satu periode pengamatan,
baru dimatikan.

---

## 3. Migrasi data

### 3.1 Data yang tidak dipindahkan
Data historis tetap di tabel `POOLDATA` yang sama — **satu database bersama** (D-21). Tidak ada
migrasi massal data klaim.

### 3.2 Data yang dipindahkan
Yang berpindah adalah data yang selama ini hidup di **tabel milik engine Pega**, ke tabel baru
milik aplikasi (D-21):

| Dari | Ke | Catatan |
|---|---|---|
| `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` | Tabel header klaim baru | Dibaca 116 rule; ini tabel klaim yang sebenarnya |
| `DATAPEGA.PC_ASSIGN_WORKLIST` | Tabel penugasan baru | Worklist (D-26) |
| `DATAPEGA.PC_ASSIGN_WORKBASKET` | Tabel antrean baru | Workbasket (D-26) |
| `DATAPEGA.PR_OPERATORS` | Tabel pengguna baru | Digabung dengan profil dari HCC/HCQ (D-07) |
| `DATAPEGA.PC_LINK_ATTACHMENT` + `PC_DATA_WORKATTACH` | Tabel metadata dokumen | Berkasnya ke storage internal (D-16) |

### 3.3 Penanganan kunci warisan
`CLAIMID = 'ASM-FW-GCNMFW-WORK ' || no_klaim` tetap ada pada data lama. Aturan:
- Data lama **dibaca apa adanya**, prefix dipangkas saat dibaca ke domain.
- Data baru **tidak pernah** menulis prefix (D-22).
- Satu fungsi tunggal menangani konversi ini — tidak tersebar di banyak query.

### 3.4 Migrasi Oracle → PostgreSQL (kelak)
Terpisah dari migrasi aplikasi dan dilakukan **setelah** aplikasi Go stabil di Oracle.

Karena SQL sudah portabel (D-20) dan tidak ada pemanggilan stored procedure (D-02), langkahnya:
1. Siapkan PostgreSQL 17+ (D-24) dengan skema yang setara.
2. Pindahkan data.
3. Ganti konfigurasi driver dan generator nomor klaim (satu-satunya sakelar dialek, D-22).
4. Jalankan seluruh test suite terhadap PostgreSQL.
5. Cutover.

**Inilah imbalan dari SQL portabel:** perpindahan database tidak menyentuh satu baris pun logika
bisnis.

---

## 4. Verifikasi kesetaraan

Karena P-5 menuntut perilaku identik, kita butuh cara membuktikannya — bukan sekadar berharap.

**Pelaksananya adalah modul `S-8`**, yang masuk **gelombang 1** karena memblokir gerbang 1 setiap
modul lain (`D-42`). Lingkungannya **Pega staging vs Go staging** atas salinan data produksi, dan
keluarannya wajib **mengklasifikasikan** setiap selisih terhadap 13 butir `P-5` — bukan sekadar
melaporkannya (`D-53`, `D-54`). Rinciannya di `14-TESTING-STRATEGY.md` §6.

| Cara | Kapan | Isi |
|---|---|---|
| **Perbandingan hasil baca** | Tahap 2 | Jalankan query yang sama di Pega dan Go, bandingkan hasilnya baris per baris pada data produksi (disalin ke staging) |
| **Uji kasus aturan bisnis** | Setiap modul | Setiap aturan di Business Understanding §3 punya test dengan kasus lolos dan kasus ditolak |
| **Uji paralel transaksi** | Tahap 3+ | Klaim contoh diproses di kedua sistem, hasil akhirnya dibandingkan |
| **Rekonsiliasi harian** | Selama masa paralel | Jumlah klaim, total nilai akseptasi, dan jumlah penugasan dibandingkan antar sistem |

---

## 5. Rencana mundur (rollback)

| Tingkat | Cara | Waktu pemulihan |
|---|---|---|
| Satu rilis bermasalah | Kembalikan versi binary sebelumnya di load balancer | Menit |
| Satu modul bermasalah | Arahkan kembali menu modul itu ke Pega; kepemilikan tabel dikembalikan | Jam |
| Kegagalan menyeluruh | Pega dijadikan sistem utama kembali; Go dimatikan | Jam |

**Syarat agar rollback benar-benar mungkin:** Pega **tidak boleh dinonaktifkan** sebelum Tahap 7.
Selama masa paralel, Pega harus tetap dapat menjalankan seluruh fungsinya. Ini biaya nyata dari
Strangler Fig — dan biaya itu adalah harga dari kemampuan mundur.

---

## 6. Catatan atas target waktu

Target yang ditetapkan manajemen adalah **seluruh modul selesai akhir September 2026** (D-30).
Migration Strategy ini disusun untuk target tersebut, dengan urutan yang meletakkan pekerjaan
berisiko rendah lebih dulu sehingga bila waktu tidak mencukupi, yang tertinggal adalah bagian
yang paling sedikit dampaknya.

Perbedaan antara target itu dan ukuran pekerjaan yang terukur didokumentasikan di **R-05**
(Risk Analysis) — bukan untuk menolak target, melainkan agar keputusan pemangkasan scope atau
penambahan sumber daya di kemudian hari punya dasar angka yang jelas.

**Tiga hal yang tidak bisa dipercepat oleh penambahan orang**, dan karena itu harus dimulai
sekarang juga:
1. Menunggu source 64 procedure & function dari DBA (**R-01**)
2. Menunggu 6 API pengganti DB Link dibangun tim lain (**R-03**)
3. Waktu belajar tim atas Go, React, dan TypeScript (D-09)
