# 0030 — Satu aplikasi, empat portal, satu database per entitas

Status: Accepted
Tanggal keputusan: 2026-09-15 (`D-75`, `D-77`, `D-78`)    Tanggal dokumen: 2026-09-15
Sifat: greenfield — perilaku ini **ada** di sistem lama tetapi tidak pernah tercatat sebagai rancangan
Pemilik keputusan: Work Owner
Jejak bukti: `D-75`, `D-21`, `D-71`, `FR-F6`, `R-20` · `docs/Steering/03-CURRENT-ARCHITECTURE.md:180` · `docs/verifikasi-bukti-adr.md:788-791`
Terkait: ADR-0004 (dibatasi), ADR-0005, ADR-0023, ADR-0027, modul `F-6`

## Konteks

Permintaannya: satu aplikasi dengan **pilihan portal**, tiap portal menampilkan data dari database
entitasnya sendiri.

**Ini bukan fitur baru.** Sistem lama sudah melakukannya — dengan cara yang tidak terlihat oleh
siapa pun. Bukti dari export:

| Bukti | Angka |
|---|---|
| Perbandingan terhadap `pxRequestor.pxReqServer` | **48 perbandingan, 3 hostname** |
| Hostname penentu perilaku | host **dev** (34) · host **entitas Insurtech** (13) · host **entitas Timor-Leste** (1) |
| Akibat pada aturan uang | satu di antaranya **mengubah ambang komite dari Rp 50.000.000 menjadi 3.500** |
| Logika khusus syariah | `Activity/SpreadingSyariah_Act-Act.xml` · `Activity/CheckSyrData_Act-Act.xml` · **32 berkas** menyebut syariah |

Angka **Rp 50.000.000 versus 3.500** bukan salah ketik — itu **mata uang yang berbeda**. Entitas
Timor-Leste tidak memakai Rupiah.

**Perilaku entitas ditentukan dengan membandingkan nama server.** Tidak ada pilihan portal, tidak
ada konfigurasi, dan tidak ada satu pun dokumen yang menyebutnya sebagai rancangan. Yang diputuskan
di sini adalah **membuatnya eksplisit**, bukan menambah kemampuan baru.

### Jawaban Work Owner mengoreksi bukti kami, bukan sebaliknya

Dokumen mencatat **3 hostname**. Work Owner menyebut **minimal 4 portal**, termasuk **Sinarmas
Asuransi Syariah**. Pemeriksaan membenarkan Work Owner:

> **`IsServerSyariah` hilang dari export dan tidak tercatat di `19-GAP`** — dirujuk
> `Data Transform/SetDataEmail-DT.xml:147`, `:411` dan
> `Activity/SpreadingDataProtection-Act.xml:1294`, `:4153`. Demikian pula `IsDevelopmentServer`.
> `docs/verifikasi-bukti-adr.md:791` sudah menyimpulkan: **"Jumlah hostname sebenarnya bisa lebih
> dari 3."**

Angka 3 rendah karena **rule pembedanya hilang**, bukan karena entitasnya hanya tiga. Kata
**"minimal"** dalam jawaban Work Owner karena itu diperlakukan harfiah: daftar portal adalah
**data**, bukan konstanta di kode.

## Opsi yang dipertimbangkan

1. **Satu database per entitas, dipilih lewat portal** — *dipilih*.
2. Satu database bersama dengan kolom penanda entitas di setiap tabel.
3. Satu instans aplikasi terpisah per entitas.
4. Mempertahankan pembedaan berdasarkan hostname.

Opsi 2 ditolak: pemisahan data menjadi bergantung pada **tidak adanya satu pun kueri yang lupa
menyaring** — kelas kesalahan yang tidak dapat diuji habis. Opsi 3 ditolak karena melipatgandakan
biaya operasional dan membuat perbaikan harus dirilis empat kali. Opsi 4 adalah keadaan sekarang,
dan justru itu yang hendak dihilangkan.

## Keputusan

**Satu aplikasi Go, satu basis kode, empat portal, satu database per entitas.**

| Portal | Keterangan |
|---|---|
| Asuransi Sinar Mas | entitas utama |
| Asuransi Simas Insurtech | 13 perbandingan hostname di export |
| Sinarmas Asuransi Syariah | `IsServerSyariah` **hilang** · `SpreadingSyariah_Act` ada |
| Timor-Leste | 1 perbandingan · **mata uang berbeda** |

Ditetapkan Work Owner pada `D-75`:

| Aspek | Keputusan |
|---|---|
| Database | **satu per entitas** |
| Laporan | **per portal** — tidak ada laporan lintas portal |
| Master data | **per portal** |
| Berpindah portal | **tanpa login ulang** |
| Jejak audit | **per portal** |

## Konsekuensi

### Positif

- **Pemisahan data menjadi sifat struktural, bukan disiplin kueri.** Portal A tidak dapat membaca
  data portal B karena koneksinya memang berbeda — bukan karena penyaringnya benar.
- Perilaku entitas berhenti bergantung pada nama server. Menambah portal menjadi **penambahan
  data**, bukan penambahan `if` di 48 tempat.
- `P-1` (satu tabel ditulis satu sistem) menjadi **lebih mudah ditegakkan**, karena berlaku di
  dalam satu database.
- Ambang komite yang berbeda per entitas berhenti menjadi keanehan — ia menjadi master per portal.

### Negatif dan utang teknis

- **Empat database harus tetap sekerabat skemanya.** Satu basis kode tidak dapat melayani empat
  skema yang menyimpang. Migrasi skema (`TKT-F2-004`) kini harus berjalan **empat kali**, dan
  kegagalan di salah satunya membuat portal itu tertinggal versi.
- **Lingkup `F-4` dan `U-6` berlipat.** Master data ≥29 kelompok kini **per portal**.
- **Empat kali beban operasional**: pencadangan, pemantauan, rotasi kredensial, dan `S-8`.
- **Laporan konsolidasi menjadi tidak mungkin** tanpa keputusan baru. Bila manajemen memintanya
  kelak, itu keputusan tersendiri — bukan penyesuaian kecil.
- **Penyelidikan lintas entitas menuntut empat kueri terpisah**, karena jejak audit per portal.

### Yang paling berbahaya — `R-20`

**Berpindah portal tanpa login ulang berarti satu identitas menjangkau empat database.** Bila
kewenangan tidak **dinilai ulang pada saat perpindahan**, seorang pengguna dapat melihat data
entitas yang bukan haknya. Ini bukan bug tampilan; ini kebocoran data antar badan hukum.

Karena itu: perpindahan portal **wajib memeriksa kewenangan di server** (`ADR-0023`), dan portal
yang tidak berhak **tidak boleh muncul di daftar maupun dapat dipanggil langsung**.

### Akibat pada `S-8` yang harus dinyatakan di muka

`SpreadingSyariah_Act` ada di export, tetapi **`IsServerSyariah` tidak**. Artinya **kapan** logika
syariah berlaku tidak diketahui. Sampai rule itu diterima Tim Pega, portal Syariah **tidak punya
baseline Pega yang utuh** untuk diuji setara — keadaan yang sama dengan `F-3` dan `S-5` pada
`D-42`, dan harus diperlakukan sama: gerbang 1 diganti ukuran lain, bukan dianggap lulus.

## Pertanyaan terbuka

1. **Berapa portal sebenarnya?** Work Owner menyebut *minimal* 4. Jumlah pastinya **BELUM
   DIPUTUSKAN — pertanyaan terbuka** (pemilik: **Work Owner**).
2. **Nomor klaim `PNCN.YY.xxxx` (`D-71`) memakai sequence per database.** Dua portal dapat
   menerbitkan nomor yang sama. Perlukah penanda portal di dalam nomornya? (**Work Owner**)
3. **Aturan bisnis syariah berbeda sejauh apa?** `SpreadingSyariah_Act` membuktikan spreading-nya
   berbeda; apakah estimasi, komite, dan penyelesaian juga? (**Work Owner + Tim Pega**)
4. ~~Satu penyedia identitas untuk keempat entitas?~~ — **TERTUTUP `D-78`: login sama untuk semua
   entitas.** Menyisakan pertanyaan baru: **di mana kewenangan portal disimpan**, karena data itu
   lintas portal secara alamiah dan tidak dapat tinggal di masing-masing database (**Work Owner +
   Keamanan Informasi**)
5. **Mata uang per portal** — Timor-Leste bukan Rupiah. Kurs, pembulatan, dan format angka
   mengikuti portal? (**Work Owner**)
