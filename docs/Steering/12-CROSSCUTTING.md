# Error Handling, Logging, Configuration & Observability

Hal-hal yang menyentuh seluruh modul. Bila tidak diseragamkan sejak awal, setiap modul akan
melakukannya dengan caranya sendiri — dan pada tim di D-09 dengan 74 layar, itu pasti terjadi.

---

## 1. Error Handling

### 1.1 Tiga jenis kesalahan

| Jenis | Contoh | Penanganan | HTTP |
|---|---|---|---|
| **Kesalahan validasi bisnis** | Total spreading bukan 100% · DOL di luar periode polis · Nomor SLIK kosong | Kumpulkan **semua**, kembalikan bersamaan | `422` |
| **Pelanggaran aturan / konflik** | Akseptasi sebelum komite selesai · penugasan sudah diambil orang lain | Kembalikan satu kesalahan yang jelas | `409` |
| **Kegagalan teknis** | Database mati · sistem eksternal tidak merespons · bug | Catat lengkap, kembalikan pesan umum | `500` |

### 1.2 Aturan

1. **Kesalahan validasi dikumpulkan seluruhnya, tidak berhenti pada yang pertama.**
   `InputRegister_act` sistem lama memeriksa belasan aturan dan menampilkan semuanya sekaligus.
   Pada form registrasi berisi puluhan field, mengembalikan satu kesalahan per percobaan akan
   membuat pengguna menyerah. Ini bukan preferensi — ini kesetaraan perilaku (P-5).

2. **Kesalahan domain adalah tipe, bukan string.** Transport yang memetakannya ke kode HTTP.
   Domain tidak boleh tahu tentang HTTP.

3. **Setiap kesalahan dibungkus dengan konteks saat naik**, sehingga pesan akhirnya menceritakan
   jalurnya — bukan sekadar "record not found" tanpa keterangan record apa.

4. **Kesalahan tidak pernah ditelan.** Tidak ada `_ = err`. Bila memang sengaja diabaikan,
   wajib disertai komentar yang menjelaskan alasannya.

5. **Detail internal tidak pernah bocor ke klien.** Pesan `500` yang dikirim ke pengguna hanya
   memuat id permintaan; detail lengkapnya ada di log. Membocorkan struktur database atau
   jejak tumpukan ke browser adalah celah keamanan.

6. **Pesan untuk pengguna berbahasa Indonesia dan menjelaskan cara memperbaiki**, mengikuti gaya
   sistem lama yang sudah dikenal pengguna:
   > "Tanggal Lapor tidak boleh lebih dari 7 hari setelah Tanggal Kejadian."
   > "Nomor Polis sudah terdaftar dengan nomor klaim PNC-1865."
   > "Total spreading harus 100%."

7. **`panic` hanya untuk kondisi yang tidak mungkin terjadi**, ditangkap oleh middleware
   recovery, dicatat lengkap, dan dikembalikan sebagai `500`. Tidak pernah dipakai sebagai alur
   kendali.

---

## 2. Logging

### 2.1 Bentuk
**Terstruktur dalam JSON** memakai `log/slog`. Bukan teks bebas — log teks bebas tidak bisa
dicari, disaring, maupun diagregasi ketika sedang menelusuri masalah di production.

### 2.2 Tingkat

| Tingkat | Untuk | Contoh |
|---|---|---|
| `ERROR` | Butuh perhatian manusia | Database tidak dapat dihubungi · kegagalan tak terduga |
| `WARN` | Tidak normal tapi tertangani | Sistem eksternal gagal lalu berhasil saat dicoba ulang |
| `INFO` | Peristiwa bisnis penting | Klaim diregistrasi · akseptasi diterbitkan · pengguna login |
| `DEBUG` | Penelusuran mendalam | Mati di production, dinyalakan sementara saat dibutuhkan |

**Kegagalan validasi bisnis bukan `ERROR`.** Pengguna salah mengisi form adalah hal normal.
Mencatatnya sebagai `ERROR` akan menenggelamkan kesalahan sungguhan di antara ribuan baris
kesalahan pengisian.

### 2.3 Field wajib di setiap baris log
Waktu (UTC), tingkat, pesan, **id permintaan**, id pengguna, metode dan jalur HTTP.
Untuk peristiwa bisnis, tambahkan nomor klaim.

**Id permintaan** dibuat di middleware paling luar, dibawa lewat `context.Context`, dan muncul
di setiap baris log dari permintaan itu — juga dikembalikan ke klien pada respons `500`.
Tanpa ini, menelusuri "apa yang terjadi pada permintaan pengguna tadi" hampir mustahil.

### 2.4 Yang dilarang masuk log

Password · token dan kunci API · **NIK** · nomor telepon · alamat · **data medis** · nomor
rekening · isi dokumen.

Bila diperlukan untuk penelusuran, catat **referensinya** (nomor klaim, id dokumen), bukan
isinya. Log biasanya disimpan lebih longgar daripada database, sehingga data sensitif di dalam
log adalah kebocoran yang mudah terlewat.

### 2.5 Log pemanggilan sistem eksternal
Setiap pemanggilan keluar dicatat: sistem tujuan, operasi, lama, hasil, dan percobaan ke berapa.
Ini yang membuat pertanyaan "apakah lambatnya karena kita atau karena sistem mereka" bisa
dijawab dengan data.

---

## 3. Configuration

### 3.1 Tiga lapis, dari umum ke khusus

```
Nilai baku di kode  →  berkas YAML per lingkungan  →  variabel lingkungan
       (paling umum)                                      (paling menang)
```

- **Rahasia hanya dari variabel lingkungan** — tidak pernah dari berkas YAML yang masuk repository.
- **Aplikasi gagal saat start bila konfigurasi wajib tidak ada.** Gagal keras di awal jauh lebih
  baik daripada gagal diam-diam saat pengguna sedang bekerja.

### 3.2 Konfigurasi teknis (butuh restart)
Alamat database dan ukuran pool · alamat sistem eksternal · batas waktu · port · tingkat log ·
masa berlaku token.

### 3.3 Master data (dapat diubah tanpa restart) — D-15

Ini yang menggantikan seluruh hardcode di sistem lama:

| Master | Menggantikan hardcode | Ukuran terverifikasi |
|---|---|---|
| Ambang komite per lini bisnis | `50000000` · `30000000` · `20000000` · `7000` · `3500` · … | **8 ambang komite unik** + 7 ambang uang non-komite |
| Ambang Large Losses | `1000000000` | 1 |
| Penerima notifikasi per peristiwa dan Group Panel | email UW, pimpinan, komite, broker | **66 alamat unik**, termasuk **≥6 akun Gmail pribadi di jalur produksi** |
| Batas aturan tanggal per lini bisnis | 7 hari · 30 hari · 90 hari | — |
| Peran dan izin menu | 22 access group | 22 peran → 51 item menu |
| Operator ID penentu perilaku | `MORASOTARDODOTARIGAN`, `ELLENSUPRIYATI`, `IRMANOPITAPURBA_1`, `RATNAGUSNITASARI`, … | **24 unik** |
| Master status klaim | — | **33 kode `1134`–`1166`** (`R-06` tertutup) |
| Kurs per tanggal | `GETCURRENCYSTANDARD` yang mengembalikan `1` saat kurs tidak ada | — (`D-48`) |
| Hari libur dan jam kerja | `GET_WORKING_HOURS` dan `HRD_LBR` lewat DB Link | 18 pemakaian di 7 berkas |
| Retensi data | tidak ada | satu kebijakan untuk data klaim **dan** jejak audit (`D-62`) |

> **Angka pada versi sebelumnya terlalu kecil.** `docs/BRD` menyebut 10 email, 4 user ID, dan
> 3 ambang; verifikasi terhadap seluruh export memberi **66 / 24 / 8**. Angka lama benar untuk
> lingkupnya — dua rule saja — dan memakainya untuk `FR-F4` akan membuat estimasi master data
> meleset sekitar **enam kali lipat**.
>
> **Tidak ada akun pribadi yang dibawa** (`D-67`): seluruh penerima notifikasi berasal dari master
> Penerima Notifikasi berupa **mailbox fungsional**. Lima alamat Gmail yang dipakai sebagai
> **Operator ID** di filter laporan KPI adalah masalah **identitas**, bukan notifikasi — ia
> menyentuh `F-3` sekaligus `F-4`.

**Perbedaan yang menentukan:** konfigurasi teknis milik tim infrastruktur dan berubah saat
deployment. Master data milik pengguna bisnis dan berubah saat kebijakan berubah. Menaruh ambang
komite di berkas konfigurasi berarti setiap perubahan kebijakan membutuhkan deployment — itu
persis masalah yang membuatnya di-hardcode sejak awal.

### 3.4 Larangan perilaku berdasarkan hostname

Sistem lama membandingkan `pxRequestor.pxReqServer` terhadap **3 hostname** pada **48 titik**, dan
perbandingan itu **menentukan perilaku bisnis**, bukan sekadar tampilan:

| Host | Perbandingan | Perilaku yang dipicu |
|---|---|---|
| host **dev** | 34 | melewati atau mengganti step; mengganti penerima email; menentukan `IsServiceCenterPNC` |
| host **entitas Timor-Leste** | 1 | **mengubah ambang komite dari Rp 50.000.000 menjadi 3.500** |
| host **entitas Insurtech** | 13 | mengganti kode entitas, status investigator, filter view klaim |

Nilai hostname **tidak direproduksi di sini** sesuai aturan penulisan `D-69`; lokasinya dirujuk
dengan `berkas:baris` pada `docs/verifikasi-bukti-adr.md` §7.6.

**Ini dilarang.** Perbedaan antar lingkungan dan antar negara dinyatakan sebagai konfigurasi
eksplisit, bukan disimpulkan dari nama server. Perilaku yang bergantung pada hostname tidak dapat
diuji, tidak terlihat saat membaca kode, dan berubah diam-diam ketika server dipindahkan.

> **Yang menggantikannya belum diputuskan.** Bagaimana entitas — Indonesia, Timor-Leste,
> Insurtech — dikenali di sistem baru adalah pertanyaan terbuka pada `ADR-0025`, dan jawabannya
> menentukan **ambang komite mana yang berlaku** untuk sebuah klaim.

### 3.5 Tidak ada Dynamic System Setting yang bisa ditiru

Sistem lama **tidak memiliki satu pun Dynamic System Setting**. Konfigurasi dinamis yang nyata
berupa **tabel Oracle yang dikunci per IP aplikasi** — pola yang bertabrakan langsung dengan
tuntutan **dua instans di belakang load balancer** (`D-27`).

Artinya tidak ada mekanisme konfigurasi lama yang dapat disalin; lapisan konfigurasi §3.1 adalah
**kemampuan baru**, bukan pemindahan.

### 3.6 Rahasia — masih terbuka

Export memuat **3 password SMTP di 31 lokasi** dan **1 pasang kredensial OAuth**, seluruhnya
plaintext, dengan `UseSSL=false` di seluruh kemunculannya (`R-17`).

Aturan §3.1 — *"rahasia hanya dari variabel lingkungan"* — adalah **arah**, bukan keputusan final.
**Ke mana rahasia dipindahkan, siapa pemiliknya, dan apakah kredensial yang telanjur terekspos
harus dirotasi masih `OPEN`** (`D-40`, `ADR-0025`). Sampai itu dijawab, `F-4` dan `F-5` tidak dapat
ditulis lengkap, dan cara repository ini disimpan ikut menjadi persoalan keamanan hari ini.

---

## 4. Observability

Dijaga sederhana dan sepadan dengan skala 200–300 pengguna di VM on-premise (D-08). Tanpa
platform observability besar.

### 4.1 Health check

| Endpoint | Menjawab | Dipakai oleh |
|---|---|---|
| `/health/live` | Proses masih hidup? | Load balancer |
| `/health/ready` | Siap menerima trafik? (database terhubung, migrasi selesai) | Load balancer saat rolling deployment |

Pembedaan keduanya penting untuk D-27: saat rolling deployment, instance baru harus dinyatakan
*belum siap* sampai benar-benar siap, agar load balancer tidak mengirim trafik ke instance yang
sedang start.

### 4.2 Metrik
Diekspos dalam format Prometheus, dikumpulkan bila sudah ada infrastrukturnya.

| Kelompok | Isi |
|---|---|
| HTTP | Jumlah permintaan, lama, kode status per endpoint |
| Database | Koneksi terpakai, koneksi menunggu, lama query |
| Sistem eksternal | Jumlah pemanggilan, lama, tingkat kegagalan per sistem |
| Bisnis | Klaim diregistrasi, klaim diakseptasi, tugas menunggu per workbasket |
| Runtime | Memori, goroutine, GC |

Metrik bisnis sengaja dimasukkan: "jumlah tugas menunggu di workbasket komite" adalah tanda
peringatan operasional yang jauh lebih berguna daripada penggunaan CPU.

### 4.3 Pemberitahuan yang layak membangunkan orang

| Kondisi | Alasan |
|---|---|
| Tingkat kesalahan `500` melebihi ambang | Ada yang rusak |
| Database tidak dapat dihubungi | Aplikasi tidak berfungsi |
| Health check gagal pada satu instance | Kapasitas berkurang; 24/7 terancam (D-27) |
| Sistem eksternal gagal terus-menerus | Fungsi tertentu lumpuh |
| Antrean notifikasi menumpuk | Pemberitahuan tidak terkirim |

**Tidak** memicu pemberitahuan: kegagalan validasi pengguna, `404`, dan lonjakan trafik yang
wajar. Pemberitahuan yang terlalu sering membuat orang berhenti memperhatikannya — dan saat itu
terjadi, pemberitahuan menjadi tidak berguna.

### 4.4 Penelusuran jejak
Id permintaan yang dibawa lewat `context.Context` dan muncul di seluruh log sudah memadai untuk
skala ini. Distributed tracing tidak diperlukan karena hanya ada satu layanan.
