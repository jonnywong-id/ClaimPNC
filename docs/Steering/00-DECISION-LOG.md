# Decision Log — Migrasi Claim PNC

Semua pertanyaan discovery, pilihan yang ditawarkan, jawaban yang dipilih, dan keputusan akhir
yang dihasilkan. Setiap keputusan punya ID (`D-xx`) yang direferensikan dari dokumen Steering lain.

Status: `DECIDED` = final · `OPEN` = masih perlu konfirmasi pihak lain · `ASSUMED` = asumsi kerja, wajib divalidasi

---

## Sesi 1 — 2026-09-07 — Discovery Awal

### D-01 · Strategi Database
**Status:** DECIDED

**Pertanyaan:** Database POOLDATA/GENERAL dipakai bersama sistem lain (ASMD, SIMASNET, SMI, OPJAVA lewat DB Link) dan berisi 86 stored procedure. Bagaimana nasibnya di migrasi ke PostgreSQL?

**Pilihan yang ditawarkan:**
1. Hanya data milik Claim PNC yang pindah ke PostgreSQL
2. Seluruh POOLDATA + GENERAL ikut pindah ke PostgreSQL
3. PostgreSQL hanya untuk data baru, Oracle tetap dibaca (hybrid jangka panjang)
4. Belum diputuskan — perlu dibahas dengan tim DBA/infra

**Jawaban (Other):**
> "Saya ingin seluruhnya. Dan untuk model pembuatannya nanti buat menjadi postgre friendly, seperti procedurenya di tulis langsung di scriptnya, bukan di panggil seperti oracle. Note : untuk sementara memang masih menggunakan oracle, perpindahan ke postgre masih blm tahu kapan namun nantinya pasti bakal pindah ke postgre"

**Keputusan akhir:**
- Target akhir adalah **PostgreSQL untuk seluruh data**, bukan hanya subset Claim PNC.
- **Tidak ada pemanggilan stored procedure** dari aplikasi. Seluruh SQL ditulis langsung di kode Go.
- Desain skema, tipe data, indexing, dan gaya SQL mengacu ke **PostgreSQL sebagai kanonikal**.
- **Runtime sementara tetap Oracle**, tanggal cutover belum ditentukan. Aplikasi wajib bisa berjalan di Oracle tanpa mengorbankan desain PostgreSQL-first.

**Dampak ke Steering:** Database Strategy · Technical Strategy · Folder Structure · Migration Strategy · Risk Analysis

---

### D-02 · Perlakuan Stored Procedure Oracle
**Status:** DECIDED

**Pertanyaan:** Ada 86 stored procedure/package Oracle (INSERT_PLADLA, PEGA_JSON_KLAIM_PNC, CONVERTJSONPRODUCTION, dll) yang berisi logika bisnis nyata. Bagaimana perlakuannya?

**Pilihan yang ditawarkan:**
1. Ditulis ulang jadi logika Go di application layer
2. Diport ke PL/pgSQL apa adanya
3. Hybrid: yang murni data-access ke Go, yang transaksional berat tetap di PL/pgSQL
4. Belum tahu — isi procedure-nya sendiri belum pernah dilihat

**Jawaban:** Ditulis ulang jadi logika Go di application layer

**Keputusan akhir:** Seluruh logika bisnis yang kini berada di dalam stored procedure dinaikkan ke application layer Go. Database menjadi penyimpanan murni.

**Konsekuensi terbuka:** Source procedure **tidak ada** di export XML ini. Diperlukan DDL/source dari DBA sebelum modul yang bergantung padanya bisa diimplementasikan → **R-01** di Risk Analysis.

**Koreksi jumlah (2026-09-08):** angka "86 stored procedure" pada pertanyaan di atas adalah hitungan awal yang terlalu tinggi — pemindaian pertama ikut menangkap nama tabel pada `INSERT INTO tabel (kolom…)` sebagai pemanggilan procedure. Angka terverifikasi: **56 procedure + 8 function lokal = 64 objek** yang harus diminta ke DBA, ditambah 6 function remote lewat DB Link yang masuk lingkup D-25/R-03. Teks pertanyaan sengaja **tidak diubah**, karena Decision Log mencatat apa yang benar-benar ditanyakan saat itu. Daftar lengkapnya di [`19-GAP-EXPORT-DETAIL.md`](19-GAP-EXPORT-DETAIL.md).

**Dampak ke Steering:** Technical Strategy · Module Breakdown · Risk Analysis · Migration Strategy

---

### D-03 · Scope Migrasi (GCNMFW vs GISFW)
**Status:** DECIDED

**Pertanyaan:** Seberapa besar scope migrasi? Ada 2 framework: GCNMFW (Claim PNC, 633 activity) dan GISFW (policy/underwriting, 141 activity) yang saling terikat.

**Pilihan yang ditawarkan:**
1. Hanya GCNMFW (Claim PNC) — GISFW jadi dependensi eksternal
2. GCNMFW + GISFW sekaligus
3. GCNMFW dulu, GISFW menyusul di fase berikutnya
4. Hanya sebagian modul Claim PNC sebagai pilot

**Jawaban (Other):**
> "buat yg ada dulu, semua sudah sy lampirkan di xmlnya. Memang benar untuk GISFW berisi data polis yang dibutuhkan saat claim, namun GISFW sedang di develop tim lain."

**Keputusan akhir:** Scope = seluruh rule yang ada di export XML ini. GISFW **tidak dimiliki** tim ini (sedang dikembangkan tim lain), tetapi struktur data polis yang dipakai saat klaim tetap masuk scope sebagai data yang dikonsumsi. Batas kepemilikan diatur di **D-04**.

**Dampak ke Steering:** Bounded Context · Module Breakdown · Domain Model

---

### D-04 · Batas Bounded Context terhadap Data Polis (GISFW)
**Status:** DECIDED

**Pertanyaan:** GISFW (data polis) dikembangkan tim lain, tapi 141 activity-nya ada di export ini dan dipakai saat klaim. Bagaimana batas kepemilikan datanya?

**Pilihan yang ditawarkan:**
1. Claim PNC menyimpan salinan snapshot polis saat klaim dibuat
2. Claim PNC memanggil API GISFW tim lain secara real-time
3. Kita ikut migrasikan GISFW juga dalam project ini
4. Belum jelas — perlu koordinasi dengan tim GISFW

**Jawaban:** Claim PNC menyimpan salinan snapshot polis saat klaim dibuat

**Keputusan akhir:** Domain Klaim menyimpan **snapshot polis** pada saat klaim diregistrasi — melanjutkan pola `JSON_POLIS`/`JSON_KLAIM` yang sudah ada, tetapi dengan skema eksplisit. Klaim menjadi kebal terhadap perubahan polis setelah registrasi dan tidak bergantung pada ketersediaan sistem tim lain saat runtime.

**Dampak ke Steering:** Domain Model · Bounded Context · Database Strategy · Integration Strategy

---

### D-05 · Strategi Cutover
**Status:** DECIDED

**Pertanyaan:** Aplikasi lama masih aktif dikembangkan (124 activity diubah tahun 2026). Bagaimana strategi cutover-nya?

**Pilihan yang ditawarkan:**
1. Strangler Fig — modul dipindah bertahap, dua sistem jalan paralel
2. Big Bang — bangun penuh, lalu potong sekali waktu
3. Per case type — Register dulu, lalu Komite, OpenProtection, ReceiveDocument
4. Belum diputuskan

**Jawaban:** Strangler Fig — modul dipindah bertahap, dua sistem jalan paralel

**Keputusan akhir:** Migrasi bertahap. Pega dan aplikasi Go berjalan paralel; modul dialihkan satu per satu. Konsekuensi: dibutuhkan strategi berbagi data/status antar kedua sistem selama masa paralel.

**Dampak ke Steering:** Migration Strategy · Deployment Strategy · Risk Analysis · Database Strategy

---

### D-06 · Penanganan Dialek SQL (Oracle sekarang → PostgreSQL nanti)
**Status:** SUPERSEDED oleh **D-20** — pertanyaan ini diangkat ulang dengan bukti terukur pada sesi grilling dan diputuskan di sana

**Pertanyaan:** Go harus jalan di Oracle dulu, lalu pindah ke PostgreSQL. SQL ditulis langsung di kode. Bagaimana menangani perbedaan dialek (NVL vs COALESCE, ROWNUM vs LIMIT, JSON_TABLE vs jsonb, TO_CHAR, sequence)?

**Pilihan yang ditawarkan:**
1. Repository interface, dua implementasi SQL (Oracle & PostgreSQL) berdampingan
2. Tulis SQL portabel (ANSI) sebisa mungkin, dialek khusus hanya di sedikit tempat
3. Pakai query builder / ORM yang menangani dialek otomatis
4. Bangun langsung untuk PostgreSQL saja, Oracle diakses lewat lapisan sementara

**Jawaban (Other):**
> "buat langsung postgre friendly, maksudnya adalah sudah tidak ada call procedure seperti sistem oracle, dan lain sebagainya. Note : untuk sementara masih pakai database oracle."

**Catatan:** Jawaban menegaskan *tidak ada call procedure* dan *PostgreSQL-friendly*, tetapi belum memilih mekanisme teknis untuk menjembatani dua dialek.

**Tindak lanjut:** pertanyaan diajukan ulang pada sesi grilling setelah seluruh 652 rule SQL dianalisis dan dikategorikan, sehingga pilihannya dapat disajikan dengan volume yang terukur. Hasilnya → **D-20**.

**Dampak ke Steering:** Database Strategy · Technical Strategy · Coding Standards · Testing Strategy

---

### D-07 · Authentication & Authorization
**Status:** DECIDED

**Pertanyaan:** Autentikasi saat ini memakai operator Pega (DATAPEGA.PR_OPERATORS) dengan 22 access group. Di sistem baru pakai apa?

**Pilihan yang ditawarkan:**
1. SSO korporat Sinarmas (LDAP / Active Directory / OIDC)
2. User & password sendiri di aplikasi baru (JWT)
3. Migrasi tabel operator Pega apa adanya ke sistem baru
4. Belum diputuskan

**Jawaban (Other):**
> "autentikasi nantinya akan di buat baru. gambaran singkatnya, user akan login memasukkan username dan pwd yg terdaftar di internal API hcc/hcq. Jika sudah berhasil login, hak akses menu akan di buat terpisah lagi nantinya di aplikasi klaim ini, artinya akan ada tabel baru yg menyimpan user bisa akses menu apa saja. Tapi tabelnya belum di buat"

**Pertanyaan lanjutan:** Untuk API HCC/HCQ, informasi apa yang dikembalikan setelah login berhasil?

**Pilihan yang ditawarkan:**
1. Hanya status berhasil/gagal — data user lain diambil dari tabel aplikasi
2. Mengembalikan token yang harus dipakai untuk request berikutnya
3. Mengembalikan profil lengkap user (NIK, nama, cabang, jabatan, email)
4. Saya belum tahu detail responsnya

**Jawaban:** Mengembalikan profil lengkap user (NIK, nama, cabang, jabatan, email)

**Keputusan akhir:**
- **Authentication** didelegasikan ke **API internal HCC/HCQ** (username + password). Respons memuat profil lengkap: NIK, nama, cabang, jabatan, email.
- **Authorization** dimiliki dan dikelola sepenuhnya oleh aplikasi Claim PNC lewat tabel baru yang memetakan user → menu/permission. Tabel belum ada, akan dirancang dalam project ini.
- Aplikasi menerbitkan session/token miliknya sendiri setelah HCC/HCQ memvalidasi kredensial.

**Dampak ke Steering:** Security · Authentication · Authorization · Domain Model · API Strategy

---

### D-08 · Deployment Target
**Status:** DECIDED

**Pertanyaan:** Di mana sistem baru akan dijalankan?

**Pilihan yang ditawarkan:**
1. On-premise VM / bare metal (data center Sinarmas)
2. On-premise Kubernetes / container platform
3. Cloud (AWS / GCP / Azure)
4. Belum diputuskan / masih dibahas tim infra

**Jawaban:** On-premise VM / bare metal (data center Sinarmas)

**Keputusan akhir:** Deployment ke VM/bare metal on-premise di data center Sinarmas. Tanpa asumsi Kubernetes. Strategi deployment, monitoring, dan scaling harus realistis untuk lingkungan VM.

**Dampak ke Steering:** Deployment Strategy · Scalability · Performance Strategy · Monitoring · Infrastructure

---

### D-09 · Latar Belakang Tim Pengembang
**Status:** DECIDED

**Pertanyaan:** Siapa yang akan membangun dan merawat sistem baru ini?

**Pilihan yang ditawarkan:**
1. Tim internal yang sekarang pegang Pega (belum terbiasa Go/JS modern)
2. Developer Golang yang sudah berpengalaman
3. Developer frontend JavaScript/TypeScript berpengalaman (React/Vue)
4. Vendor / konsultan eksternal

**Jawaban:** Tim internal yang sekarang pegang Pega (belum terbiasa Go/JS modern)

**Keputusan akhir:** Tim adalah developer Pega internal yang akan dilatih ulang. Ini menjadi **batasan utama** dalam pemilihan frontend dan gaya kode backend:
- Utamakan learning curve landai dan jumlah konsep yang sedikit, bukan yang paling canggih.
- Steering harus sangat preskriptif: struktur folder, penamaan, dan pola baku yang seragam.
- Hindari abstraksi berat, generic tingkat lanjut, dan "magic" framework.

**Dampak ke Steering:** Frontend Decision · Coding Standards · Folder Structure · Maintainability · Testing Strategy

---

### D-10 · Volume Data
**Status:** DECIDED (angka pasti masih OPEN)

**Pertanyaan:** Berapa volume data dan beban puncaknya?

**Pilihan yang ditawarkan:**
1. Saya tahu perkiraan angkanya — akan saya isi
2. Belum tahu, perlu ditanyakan ke tim DBA
3. Skala kecil — ratusan klaim per bulan, data historis < 1 juta baris
4. Skala besar — ribuan klaim per bulan, data historis puluhan juta baris

**Jawaban:** Skala besar — ribuan klaim per bulan, data historis puluhan juta baris

**Keputusan akhir:** Sistem menangani **ribuan klaim per bulan** dengan **data historis puluhan juta baris**, tetapi hanya **200–300 user aktif harian**. Karakteristiknya: *beban data besar, konkurensi rendah*. Optimasi diarahkan ke volume data (indexing, paginasi keyset, partisi, arsip) — **bukan** ke konkurensi tinggi.

**Dampak ke Steering:** Performance Strategy · Database Strategy · Scalability · Non Functional Requirements

---

### D-11 · Strategi Reporting & Export
**Status:** DECIDED

**Pertanyaan:** Ada 56 Report Definition + banyak export CSV/Excel (pxConvertResultsToCSV dipanggil 95x). Seberapa penting fitur laporan di sistem baru?

**Pilihan yang ditawarkan:**
1. Sangat penting — semua laporan harus ada sejak hari pertama
2. Penting, tapi bisa menyusul setelah proses inti jalan
3. Sebaiknya dipindah ke tools BI terpisah (Metabase, Superset, Power BI)
4. Belum diputuskan

**Jawaban (Other):**
> "buat script sendiri untuk generate pdf / excel / csv, karena kita sudah tidak lagi menggunaan pega."

**Keputusan akhir:** Pembuatan PDF, Excel, dan CSV **diimplementasikan sendiri di dalam aplikasi Go**. Tidak memakai engine reporting Pega maupun tools BI eksternal. Steering memerlukan bab Reporting & Document Generation Strategy tersendiri, termasuk penanganan export bervolume besar agar tidak membebani transaksi.

**Dampak ke Steering:** Module Breakdown · Performance Strategy · Technical Strategy · Non Functional Requirements

---

### D-12 · Penggunaan Mobile / Lapangan
**Status:** DECIDED

**Pertanyaan:** Ada kategori foto survey (FotoDepan, FotoRangka, FotoSelfieKendaraan, FotoLokasiDebitur) dan role PNCSurveyor. Apakah ada user yang memakai aplikasi ini dari lapangan/HP?

**Pilihan yang ditawarkan:**
1. Ya — surveyor/adjuster memakai HP di lapangan untuk upload foto
2. Tidak — semua user memakai desktop di kantor
3. Mayoritas desktop, tapi surveyor kadang pakai tablet/HP
4. Surveyor pakai aplikasi terpisah, bukan aplikasi ini

**Jawaban:** Mayoritas desktop, tapi surveyor kadang pakai tablet/HP

**Keputusan akhir:** Frontend **desktop-first namun responsive**. Layar padat data (inbox, report, grid akseptasi) dioptimalkan untuk desktop. Layar yang dipakai surveyor — terutama upload foto dan input hasil survei — wajib nyaman di tablet/HP dan tahan jaringan lambat.

**Dampak ke Steering:** Frontend Decision · Non Functional Requirements · Performance Strategy

---

### D-13 · Ekspektasi UI/UX
**Status:** DECIDED

**Pertanyaan:** Bagaimana ekspektasi tampilan sistem baru dibanding Pega sekarang (74 layar, 269 section)?

**Pilihan yang ditawarkan:**
1. Tiru alur & tata letak Pega — supaya user tidak perlu belajar ulang
2. Desain ulang total dengan UX yang lebih baik
3. Alur bisnis sama, tampilan dimodernkan
4. Belum dipikirkan — fokus dulu ke fungsionalitas

**Jawaban:** Tiru alur & tata letak Pega — supaya user tidak perlu belajar ulang

**Keputusan akhir:** Alur kerja, urutan langkah, penempatan field, dan tata letak layar **mengikuti Pega yang ada**. Perubahan tampilan dijaga seminimal mungkin agar user tidak perlu pelatihan ulang.

**Konsekuensi yang harus disadari:**
- Aplikasi mewarisi UX Pega yang padat (grid lebar, form panjang, banyak tab).
- Frontend harus kuat menangani **tabel/grid kompleks** dan **form panjang bernested repeat** — ini jadi kriteria penilaian utama pemilihan framework.
- Kesempatan perbaikan UX ditunda ke fase pasca-migrasi.

**Dampak ke Steering:** Frontend Decision · Module Breakdown · Future Enhancement

---

### D-14 · Aturan Penjenjangan Komite
**Status:** DECIDED (matriks detail: OPEN)

**Pertanyaan:** Komite bisa berjenjang sampai 4 level (KomiteCount 1–4 → workbasket komitepnc..komitepnc4). Apa yang menentukan berapa level yang dibutuhkan?

**Pilihan yang ditawarkan:**
1. Berdasarkan nilai klaim — makin besar makin banyak level
2. Berdasarkan jenis bisnis / GroupPanel
3. Kombinasi nilai klaim dan jenis bisnis
4. Ditentukan manual oleh user, bukan otomatis

**Jawaban:** Kombinasi nilai klaim dan jenis bisnis

**Keputusan akhir:** Jenjang komite ditentukan **matriks (nilai klaim × jenis bisnis)**. Matriks ini menjadi **master data yang dapat dikonfigurasi**, bukan kode. Ambang yang ditemukan di kode lama (50.000.000 umum; 30.000.000 untuk PA/Travel; 3.500 untuk server Timor-Leste) diperlakukan sebagai *nilai awal* yang harus divalidasi ulang oleh tim bisnis.

**Terbuka:** Matriks lengkapnya belum ada di tangan kita.

**Temuan lanjutan (2026-09-08):** matriks ini ternyata **sudah berupa master data** di tabel `POOLDATA.EMAILKOMITE`. Jumlah jenjang = jumlah baris hasil query (`KomiteLoop := pxResultCount`), urutan jenjang = kolom `DEGREE`, ambang nilai = kolom `LIMIT_BOTTOM`. Yang belum kita punya hanyalah **isi tabelnya** — cukup satu query ke DBA. Yang tetap perlu dikonfirmasi ke tim bisnis adalah logika pemilihan di `SetEmailKomite`, yang bercampur dengan nama orang tertentu dan percabangan hostname server. Rincian lengkap: [`20-DETAIL-KOMITE-DBLINK.md`](20-DETAIL-KOMITE-DBLINK.md).

**Dampak ke Steering:** Domain Model · Module Breakdown · Configuration · Risk Analysis

---

### D-15 · Nilai Hardcode → Konfigurasi
**Status:** DECIDED

**Pertanyaan:** Di kode ada nilai hardcode: email pimpinan/UW, user ID tertentu (MORASOTARDODOTARIGAN, ELLENSUPRIYATI), ambang komite, dan hostname server. Bahkan ada blok bertanda "TESTING" yang menimpa email UW asli di alur produksi. Bagaimana perlakuannya di sistem baru?

**Pilihan yang ditawarkan:**
1. Semua jadi konfigurasi / master data yang bisa diubah tanpa deploy
2. Konfigurasi untuk yang sering berubah, konstanta kode untuk yang jarang
3. Salin apa adanya dulu supaya perilaku identik, dibersihkan belakangan
4. Perlu ditinjau satu per satu bersama tim bisnis

**Jawaban:** Semua jadi konfigurasi / master data yang bisa diubah tanpa deploy

**Keputusan akhir:** **Tidak ada nilai bisnis yang boleh di-hardcode.** Semua ambang nilai, penerima notifikasi, pemetaan role, alamat endpoint, dan aturan penjenjangan disimpan sebagai konfigurasi atau master data yang dapat diubah tanpa deploy ulang.

**Konsekuensi:** Dibutuhkan **modul Administrasi/Master Data** beserta layar pengelolanya — tambahan scope yang harus masuk perencanaan sejak awal.

**Blok "TESTING" di `InputRegister_act` step 106–109 tidak dibawa ke sistem baru.** Perilaku benar yang dipakai adalah email UW asli per Group Panel.

**Dampak ke Steering:** Configuration · Module Breakdown · Security · Coding Standards · Migration Strategy

---

### D-16 · Penyimpanan Dokumen
**Status:** DECIDED

**Pertanyaan:** Dokumen/lampiran sekarang tersimpan di 3 tempat (BLOB Oracle, tabel Pega, object storage eksternal via API). Di sistem baru bagaimana?

**Pilihan yang ditawarkan:**
1. Object storage (MinIO / S3-compatible) on-premise, DB hanya simpan metadata
2. Tetap pakai API storage internal yang sudah ada (app13/app8 Sinarmas)
3. Simpan sebagai BLOB di database seperti sekarang
4. Belum diputuskan

**Jawaban:** Tetap pakai API storage internal yang sudah ada (app13/app8 Sinarmas)

**Keputusan akhir:** Dokumen disimpan lewat **API storage internal Sinarmas yang sudah berjalan**. Database aplikasi hanya menyimpan **metadata dan referensi** (ID gambar, URL, masa berlaku, kategori). Tiga mekanisme penyimpanan yang ada sekarang **disatukan menjadi satu jalur**.

**Konsekuensi:** Ketersediaan dokumen bergantung pada layanan tim lain. Steering harus mengatur penanganan kegagalan, retry, dan masa berlaku URL.

**Dampak ke Steering:** Document Storage Strategy · Integration · Error Handling · Risk Analysis

---

### D-17 · Proses Terjadwal / Batch
**Status:** OPEN — gap discovery

**Pertanyaan:** Apakah ada proses terjadwal / batch di sistem sekarang (reminder SLA, kirim email otomatis, sinkronisasi ke kasir/GL, generate laporan harian)?

**Pilihan yang ditawarkan:**
1. Ya — ada job terjadwal, tapi belum tahu detail lengkapnya
2. Ada reminder / notifikasi email otomatis berbasis SLA
3. Ada sinkronisasi/pengiriman data otomatis ke sistem lain
4. Tidak ada — semua proses dipicu aksi user

**Jawaban:** Ya — ada job terjadwal, tapi saya belum tahu detail lengkapnya

**Keputusan akhir:** Sistem baru **membutuhkan komponen scheduler**. Namun daftar job-nya belum diketahui.

**Penyebab gap:** Export XML ini **tidak memuat rule Agent maupun Queue Processor Pega**, sehingga job terjadwal tidak dapat dianalisis dari source. → **R-02** di Risk Analysis.

**Tindak lanjut:** Minta export `Rule-Agent-Queue`, `Rule-Async-QueueProcessor`, dan `Rule-Job-Scheduler` dari sistem Pega.

**Dampak ke Steering:** Module Breakdown · Deployment Strategy · Risk Analysis · Future Architecture

---

### D-18 · Empat Konsep Status
**Status:** DECIDED (arti kode 1142–1151: OPEN)

**Pertanyaan:** Ada 4 konsep status yang tumpang tindih: StatusWork (Pega), StatusClaim (kode 1142–1151), ClaimStatus (0/1), dan StatusPosisi (On Progress/Done). Apakah ini memang 4 hal berbeda?

**Pilihan yang ditawarkan:**
1. Ya, 4 hal berbeda — semuanya harus dipertahankan
2. Sebenarnya duplikat — hasil penumpukan bertahun-tahun, boleh disederhanakan
3. StatusClaim yang utama, sisanya turunan/teknis
4. Saya perlu cek dulu ke user bisnis

**Jawaban:** Ya, 4 hal berbeda — semuanya harus dipertahankan

**Keputusan akhir:** Keempat status dimodelkan sebagai **konsep terpisah dengan nama yang tidak bisa tertukar** (lihat `CONTEXT.md` bagian Status):
- Status Proses (alur kerja)
- Status Klaim (status bisnis, kode 1142–1151)
- Flag Klaim (penanda biner)
- Status Posisi Progres (tahapan progres)

**Terbuka:** Label dan arti tiap kode `1142`–`1151` tersimpan di master status `V_STS_CLAIM` yang **tidak ada di export XML**. Harus diambil dari database sebelum state machine klaim difinalkan.

**Dampak ke Steering:** Domain Model · Database Strategy · Risk Analysis

---

### D-19 · Ubiquitous Language (Glossary Domain)
**Status:** DECIDED

**Pertanyaan:** Arti singkatan domain yang dipakai di seluruh kode.

**Jawaban (Other) — dikonfirmasi langsung oleh pemilik bisnis:**
> RCL = Rejected klaim
> PUCL = Proses Ulang klaim
> PLA = Preliminary Loss Advice, untuk memberitahu nilai estimasi klaim kepada koasuransi/reasuransi
> DLA = Definite Loss Advice, untuk memberitahu nilai akseptasi klaim kepada koasuransi/reasuransi
> Pre DLA = untuk memberitahu nilai klaim yang akan di aksep kepada koasuransi/reasuransi sebelum akseptasi dilakukan
> LOD = Letter of Discharge, surat pemberitahuan nilai ganti rugi klaim yang disetujui asuransi kepada tertanggung
> TKA = jenis bisnis untuk asuransi Tenaga Kerja Asing
> KBRU = Kali Besar Raya Utama, salah satu broker asuransi
> BPPDAN = Badan Pengelola Pusat Data Asuransi Nasional, salah satu jenis treaty asuransi
> MBU vs Non MBU = Motor Business Unit vs Non Motor Business Unit, Non MBU = PNC
> SPK = Sinarmas Penjaminan Kredit
> OS = OutStanding

**Keputusan akhir:** Glossary lengkap ditulis di **`CONTEXT.md`** dan menjadi **ubiquitous language wajib** — dipakai konsisten di nama tabel, struct Go, endpoint API, label UI, dan dokumentasi.

**Temuan penting:** **Non-MBU = PNC.** Aplikasi "Claim PNC" menangani klaim **Non-Motor**. Ini mengoreksi asumsi awal dan mempengaruhi penamaan bounded context.

**Penamaan yang sengaja diubah** (lihat `CONTEXT.md`):
- `Object`/`ObjectList` → **Objek Pertanggungan / Insured Item** (menghindari tabrakan makna pemrograman)
- `Adjustment`/`AdjustmentList` → **Settlement Line** (isinya nilai penyelesaian, bukan proses adjusting)
- Alias kolom SQL warisan dinamai ulang menyeluruh

**Dampak ke Steering:** Domain Model · Bounded Context · Coding Standards · Database Strategy · API Strategy

---

## Sesi 2 — 2026-09-07 — Grilling (skill `mattpocock-skills:grilling`)

### D-20 · Mekanisme Dialek SQL
**Status:** DECIDED

**Pertanyaan (Grilling Q1):** Bagaimana mekanisme menangani dua dialek SQL (Oracle sekarang, PostgreSQL nanti)?

**Pilihan yang ditawarkan:**
1. Repository interface + SQL per dialek, PostgreSQL kanonikal *(rekomendasi saya)*
2. Satu set SQL portabel yang jalan di kedua database
3. Tulis SQL PostgreSQL saja, Oracle ditangani lapisan penerjemah
4. Pakai query builder yang menangani dialek otomatis

**Jawaban:** Satu set SQL portabel yang jalan di kedua database
*(berbeda dari rekomendasi saya — keputusan pemilik project yang diikuti)*

**Bukti terukur yang dikumpulkan sebelum keputusan** (dari 652 rule SQL, 534 KB):

| Kategori | Volume |
|---|---|
| Sudah portabel | `CASE WHEN` 317× · `\|\|` 191× · `FETCH FIRST` 72× · `SUBSTR` 69× · `ROW_NUMBER()` 14× · CTE 8× |
| Terjemahan mekanis | `TRUNC` 150× · `NVL` 100× · `SYSDATE` 69× · `DECODE` 16× · `DUAL` 12× · `INSTR` 11× · `LISTAGG` 4× · `(+)` 2× |
| Tidak portabel | `JSON_VALUE/QUERY` 195× · `JSON_TABLE` 27× · DB Link 64× · `TO_CHAR` 411× · `SEQ.NEXTVAL` |

**Keputusan akhir:** **Satu set SQL portabel** yang berjalan di Oracle 19c dan PostgreSQL 17+, dengan **tiga pengecualian terkelola**:

1. **Pemformatan tanggal dan angka dikeluarkan dari SQL ke Go.** Menghapus 411 pemakaian `TO_CHAR` sekaligus memperbaiki bug senyap: SQL saat ini mengembalikan tanggal sebagai string `'dd/mm/yyyy'`, sehingga pengurutan dan penyaringan tanggal salah secara diam-diam.
2. **Generator nomor klaim** adalah satu-satunya tempat dengan sakelar dialek eksplisit (lihat D-23).
3. **Paginasi diseragamkan ke `OFFSET … FETCH NEXT … ROWS ONLY`**, menggantikan 68 pemakaian `ROWNUM`. Pola ini sudah dipakai di 35 rule pada codebase yang ada, jadi bukan hal baru bagi tim.

**Prasyarat wajib:** PostgreSQL 17+ (lihat D-24). Tanpa itu keputusan ini tidak bisa dijalankan.

**Dampak ke Steering:** Database Strategy · Coding Standards · Technical Strategy · Testing Strategy

---

### D-21 · Berbagi Data Selama Jalan Paralel
**Status:** DECIDED (desain tabel baru: OPEN)

**Pertanyaan (Grilling Q2):** Selama Pega dan Go jalan paralel, bagaimana keduanya berbagi data klaim?

**Pilihan yang ditawarkan:**
1. Satu database bersama, kepemilikan tabel dibagi per modul *(rekomendasi saya)*
2. Database terpisah dengan sinkronisasi dua arah
3. Go jadi satu-satunya penulis, Pega diubah agar hanya membaca
4. Perlu dibahas dulu dengan tim DBA dan tim Pega

**Jawaban (Other):**
> "memang 1 database namun untuk beberapa data yg saat ini masih disimpan di pega nantinya akan dibuatkan kemudian ditampung di tabel baru. untuk sementara tabelnya belum ada"

**Keputusan akhir:** **Satu database bersama.** Data yang saat ini hidup di tabel milik engine Pega dipindahkan ke **tabel baru milik aplikasi Claim PNC**. Tabel-tabel ini belum ada dan harus dirancang dalam project ini.

**Cakupan yang terukur dari source — tabel Pega yang harus digantikan:**

| Tabel Pega | Dibaca oleh | Isi |
|---|---|---|
| `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` | **116 rule** | Tabel header klaim (pyID, policyno, qqname, statusclaim, statuswork, dateofloss, grouppanel, surveyortype, …) |
| `DATAPEGA.PC_ASSIGN_WORKLIST` | 18 rule | Penugasan per orang |
| `DATAPEGA.PC_ASSIGN_WORKBASKET` | 6 rule | Antrean bersama |
| `DATAPEGA.PR_OPERATORS` | 6 rule | Master user |
| `DATAPEGA.PC_LINK_ATTACHMENT` | 3 rule | Kaitan lampiran |
| `DATAPEGA.PC_DATA_WORKATTACH` | 2 rule | Lampiran |
| `DATAPEGA.PR_SYS_LOCKS` | 1 rule | Penguncian record |

**Aturan kepemilikan selama masa paralel:** setiap tabel hanya boleh ditulis oleh **satu** sistem. Modul yang sudah pindah ke Go memiliki tabelnya, Pega hanya membaca; sebaliknya untuk modul yang belum pindah. **Tidak ada sinkronisasi dua arah.**

**Dampak ke Steering:** Migration Strategy · Database Strategy · Domain Model · Risk Analysis

---

### D-22 · Penomoran Klaim
**Status:** DECIDED

**Pertanyaan (Grilling Q3):** Bagaimana penomoran klaim dan penanganan data historis?

**Pilihan yang ditawarkan:**
1. Pertahankan `PNC-xxxx`, buang prefix `ASM-FW-GCNMFW-WORK` *(rekomendasi saya)*
2. Pertahankan format lama sepenuhnya termasuk prefix
3. Mulai penomoran baru, data lama dipetakan
4. Perlu dicek dulu sistem lain mana saja yang memakai format ini

**Jawaban (Other):**
> "Akan gunakan format PNCN-xxxx tanpa prefix ASM-FW-GCNMFW-WORK dan penomorannya gunakan sequence di oracle yang akan diberi nama POOLDATA.CLAIM_NO_NONPEGA_SEQ dengan syntax 'PNCN-' || TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)"

**Keputusan akhir:** Klaim yang dibuat sistem baru memakai format **`PNCN-xxxx`**, dihasilkan dari sequence **`POOLDATA.CLAIM_NO_NONPEGA_SEQ`**. Prefix `ASM-FW-GCNMFW-WORK` **tidak dipakai lagi**.

**Kenapa ini lebih baik dari rekomendasi awal saya:** prefix `PNCN` membuat klaim buatan sistem baru **langsung bisa dibedakan** dari klaim warisan Pega (`PNC-xxxx`) tanpa perlu tabel pemetaan. Selama masa paralel yang panjang, kemampuan membedakan asal data ini sangat berharga untuk penelusuran dan rekonsiliasi.

**Pengecualian portabilitas yang diterima secara sadar:** sintaks sequence berbeda antara Oracle (`SEQ.NEXTVAL`) dan PostgreSQL (`nextval('seq')`). Ini **satu-satunya** tempat dengan sakelar dialek, diisolasi di satu generator nomor klaim. Memaksakan portabilitas di sini akan mengorbankan jaminan keunikan nomor — pertukaran yang tidak sepadan.

**Klaim warisan `PNC-xxxx` tetap dapat dibaca dan diproses**, hanya tidak lagi dibuat baru.

**Dampak ke Steering:** Domain Model · Database Strategy · Migration Strategy

---

### D-23 · Pilihan Frontend
**Status:** DECIDED

**Pertanyaan (Grilling Q4):** Frontend mana yang dipilih?

**Analisis lengkap:** `01-FRONTEND-ANALYSIS.md` — membandingkan React, Next.js, Vue 3, Nuxt, Angular, Svelte, SolidJS, dan Go+templ+HTMX terhadap kelebihan, kekurangan, performa, maintainability, kecocokan dengan Go, skalabilitas, ekosistem, dan learning curve.

**Pilihan yang ditawarkan:**
1. Vue 3 + TypeScript + Vite + PrimeVue *(rekomendasi saya)*
2. React + TypeScript + Vite + TanStack Table / AG Grid
3. Angular
4. Go + templ + HTMX

**Jawaban:** **React + TypeScript + Vite + TanStack Table / AG Grid**
*(berbeda dari rekomendasi saya — keputusan pemilik project yang diikuti)*

**Keputusan akhir:** Frontend memakai **React + TypeScript + Vite**, dengan **TanStack Table** atau **AG Grid** untuk kebutuhan grid berat (268 dari 269 section bergrid; inbox utama 18–27 kolom). Dibangun sebagai **SPA murni** yang disajikan sebagai berkas statis oleh binary Go — tidak ada runtime Node.js di production.

**Yang gugur dan alasannya:**
- **Next.js dan Nuxt** — keunggulan utamanya SSR/SEO, sementara aplikasi ini ada di balik login tanpa akses publik sehingga manfaatnya nol; yang tersisa hanya biayanya, yaitu runtime Node.js kedua di VM on-premise.
- **Svelte dan SolidJS** — ekosistem DataTable enterprise belum memadai untuk kebutuhan inti kita.
- **Angular** — paling terstruktur, tapi RxJS dan Dependency Injection menghantam langsung batasan terkuat kita (D-09).

**Mitigasi wajib atas risiko yang diketahui:** React tidak opinionated, dan tim di D-09 belum terbiasa JS modern. Risiko nyatanya adalah **kode berbeda gaya di setiap layar**. Karena itu Steering menetapkan konvensi yang mengikat untuk struktur folder, pengambilan data, pengelolaan state, penanganan form, dan pola tabel — bukan sekadar anjuran. Ini bukan formalitas: tanpa itu, kelemahan yang jadi alasan saya tidak merekomendasikan React akan benar-benar terjadi.

**Dampak ke Steering:** Frontend Architecture · Coding Standards · Folder Structure · Testing Strategy · Maintainability

---

### D-24 · Versi PostgreSQL
**Status:** DECIDED

**Pertanyaan (Grilling Q5):** SQL portabel untuk JSON hanya mungkin bila PostgreSQL 17+. Ada 222 pemakaian JSON_VALUE/JSON_TABLE di 29 rule. Versi PostgreSQL apa yang jadi target?

**Pilihan yang ditawarkan:**
1. PostgreSQL 17 atau lebih baru *(rekomendasi saya)*
2. PostgreSQL 15 atau 16
3. Menyesuaikan versi standar Sinarmas
4. Belum ditentukan — tim infra yang memutuskan

**Jawaban:** PostgreSQL 17 atau lebih baru

**Keputusan akhir:** Target database adalah **PostgreSQL 17 atau lebih baru**. Ini **persyaratan teknis mengikat**, bukan preferensi.

**Alasannya:** PostgreSQL 17 mendukung fungsi SQL/JSON standar `JSON_TABLE`, `JSON_VALUE`, dan `JSON_QUERY` dengan sintaks yang sama seperti Oracle. Ini membuat **seluruh 222 query JSON portabel apa adanya**. PostgreSQL 16 ke bawah belum punya `JSON_TABLE`, sehingga 29 rule harus ditulis ulang memakai operator `jsonb` khas PostgreSQL — dan keputusan D-20 (SQL portabel) menjadi tidak bisa dijalankan.

**Dampak ke Steering:** Database Strategy · Infrastructure · Risk Analysis

---

### D-25 · Pengganti DB Link
**Status:** DECIDED (kontrak API per sistem: OPEN)

**Pertanyaan (Grilling Q6):** PostgreSQL tidak punya DB Link. Ada 64 pemakaian ke 6 database lain untuk ambil data HRD, pembayaran GL, master sales, dan polis. Bagaimana penggantinya?

**Pilihan yang ditawarkan:**
1. Ganti dengan pemanggilan API ke sistem pemilik data *(rekomendasi saya)*
2. Replikasi data yang dibutuhkan ke database Claim PNC
3. Pakai Foreign Data Wrapper PostgreSQL
4. Perlu dibahas per sistem

**Jawaban:** Ganti dengan pemanggilan API ke sistem pemilik data

**Keputusan akhir:** Seluruh akses lintas database lewat DB Link **diganti pemanggilan API** ke sistem pemilik data. Kopling tersembunyi antar database dihapus dan diganti kontrak yang eksplisit.

**Inventaris DB Link yang harus digantikan:**

| DB Link | Pemakaian | Data yang diambil |
|---|---|---|
| `@ASMD` | **55×** | `DATAMINING.GET_WORKING_HOURS` (17×), `HRDASM.V_HRD_MST`, `MST_DET_SALES`, `GENERAL.MST_BUKA_PROTEKSI`, `GL.T_ALL_PAYMENT`, `LST_USER_ASURANSI`, `GENERAL.LST_MITRA`, `TREATY_LOSS`, `MBU.T_CADANGAN_KLAIM_KREDIT`, fungsi `GET_NAMA_*` |
| `@SIMASNET` | 3× | `GENERAL.MST_BUKA_PROTEKSI`, `V_KLAIM` |
| `@SMI` | 2× | `GENERAL.MST_BUKA_PROTEKSI`, `V_KLAIM` |
| `@OPJAVA` | 2× | `NEW_GENERAL.M_USER`, `M_USER_JOB` |
| `@PROD_ASM` | 1× | `POOLDATA.AGENT` |
| `@PROD_TKA` | 1× | `ANEKA.MST_SHARE_PU` |

**Risiko yang harus disadari:** API-API ini **kemungkinan besar belum ada**. Setiap satu memerlukan koordinasi dengan tim pemilik sistem. → **R-03** di Risk Analysis.

**Dampak ke Steering:** Integration Strategy · API Strategy · Risk Analysis · Migration Strategy

---

### D-26 · Model Penugasan (Inbox)
**Status:** DECIDED

**Pertanyaan (Grilling Q7):** Sistem baru harus punya model penugasan sendiri menggantikan tabel inbox Pega. Konsep apa yang dipertahankan?

**Pilihan yang ditawarkan:**
1. Pertahankan konsep Worklist (per orang) dan Workbasket (antrean bersama) *(rekomendasi saya)*
2. Sederhanakan jadi satu model penugasan saja
3. Rancang ulang total sesuai kebutuhan bisnis sekarang
4. Perlu dibahas dengan tim bisnis dulu

**Jawaban:** Pertahankan konsep Worklist (per orang) dan Workbasket (antrean bersama)

**Keputusan akhir:** Model penugasan mempertahankan **dua konsep**:
- **Worklist** — tugas ditugaskan ke satu orang tertentu
- **Workbasket** — antrean bersama, diambil siapa pun yang berwenang

**Dasar dari source (Register_Flow):** pembagian ini nyata dipakai, bukan warisan kosong.
- **Workbasket** (`impl=WorkBasket`, router `ToWorkbasket`): `RCL/PUCL`, `Investigator`, `Compliance`
- **Worklist** (`impl=WorkList`): `Input Register`, `View Polis`, `Estimation`, `Input Estimasi`, `Choose Surveyor`, `Send To Analis`, `Send To PIC Teknik`, `RCLDokter`, `Analyst Doctor`

Mengubahnya akan mengubah cara kerja user, sedangkan D-13 menetapkan alur kerja tetap sama.

**Router yang harus dibuat ulang sebagai aturan routing:** `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter`, `KomiteRouter`, `PNCAdminRouterRCV`, `ToCurrentOperator`, `ToWorkList`, `ToWorkbasket`.
Catatan: `PNCAdminRouter`, `PNCTeknikRouter`, dan `RouterRCLDokter` **tidak ada di export XML** — logikanya harus digali ulang. → **R-04**.

**Dampak ke Steering:** Domain Model · Module Breakdown · Database Strategy

---

### D-27 · Ketersediaan (Availability)
**Status:** DECIDED

**Pertanyaan (Grilling Q8a):** Kapan aplikasi harus bisa dipakai?

**Pilihan yang ditawarkan:**
1. Jam kerja kantor saja (Senin–Jumat, 07.00–18.00 WIB)
2. Jam kerja diperpanjang, termasuk Sabtu atau lembur
3. Harus hidup terus 24/7
4. Belum ada ketentuan resmi

**Jawaban:** Harus hidup terus 24/7

**Keputusan akhir:** Aplikasi harus tersedia **24/7**, termasuk saat deployment.

**Konsekuensi arsitektur yang mengikat:**
- Minimal **dua instance aplikasi** di belakang load balancer, di-update bergantian (rolling deployment).
- Aplikasi wajib **stateless** — tidak boleh ada session tersimpan di memori satu instance.
- **Migrasi skema database harus backward-compatible**: selama rolling deployment, versi lama dan baru berjalan bersamaan terhadap skema yang sama. Kolom dihapus dalam dua tahap, tidak pernah sekali jalan.
- Perlu **health check endpoint** dan graceful shutdown agar request yang sedang berjalan tidak terputus.

**Dampak ke Steering:** Deployment Strategy · Scalability · Technical Strategy · Database Strategy · NFR

---

### D-28 · Audit Trail dan Retensi Data
**Status:** DECIDED (lama retensi: OPEN)

**Pertanyaan (Grilling Q8b):** Apakah ada kewajiban dari OJK atau audit internal soal jejak perubahan data dan lama penyimpanan?

**Pilihan yang ditawarkan:**
1. Ya — ada kewajiban jejak audit dan retensi data
2. Tidak ada kewajiban formal, tapi jejak perubahan tetap diinginkan
3. Belum tahu — perlu tanya tim Compliance
4. Tidak perlu jejak audit khusus

**Jawaban:** Ya — ada kewajiban jejak audit dan retensi data

**Keputusan akhir:** Jejak audit adalah **persyaratan wajib yang dirancang sejak awal**, bukan tambahan.

Setiap perubahan yang bernilai bisnis harus tercatat permanen — **siapa, kapan, nilai sebelum, nilai sesudah** — minimal untuk: nilai estimasi klaim, nilai settlement, akseptasi, keputusan komite, penolakan (RCL), proses ulang (PUCL), perubahan status klaim, dan pembayaran.

Data audit bersifat **append-only**: tidak boleh diubah atau dihapus oleh jalur aplikasi mana pun.

**Terbuka:** lama retensi belum ditentukan; harus dikonfirmasi ke tim Compliance sebelum go-live.

**Dampak ke Steering:** Security · Database Strategy · Domain Model · NFR · Performance Strategy

---

### D-29 · Backup dan Pemulihan
**Status:** DECIDED (angka RPO/RTO: mengikuti standar infra)

**Pertanyaan (Grilling Q8c):** Kalau server database rusak, berapa banyak data yang boleh hilang dan berapa lama sistem boleh mati?

**Pilihan yang ditawarkan:**
1. Ikuti standar backup yang sudah berlaku di data center Sinarmas
2. Tidak boleh ada data hilang sama sekali
3. Boleh hilang maksimal beberapa jam, sistem boleh mati sehari
4. Belum tahu — tolong usulkan angka yang wajar

**Jawaban:** Ikuti standar backup yang sudah berlaku di data center Sinarmas

**Keputusan akhir:** Target RPO dan RTO **mengikuti kebijakan backup korporat** yang sudah berlaku untuk database Oracle saat ini. Aplikasi tidak menetapkan targetnya sendiri.

**Catatan penting:** keputusan ini perlu dibaca bersama D-27. Standar backup korporat menjawab *pemulihan saat bencana*, sedangkan tuntutan 24/7 menjawab *operasional harian*. Keduanya berbeda, dan Steering harus memastikan standar korporat yang ada memang sejalan dengan tuntutan 24/7 — bila tidak, selisihnya harus diangkat ke tim infra.

**Dampak ke Steering:** Deployment Strategy · NFR · Risk Analysis

---

### D-30 · Timeline dan Scope
**Status:** DECIDED oleh manajemen — dicatat dengan risiko terbuka

**Pertanyaan (Grilling Q9):** Kapan project ditargetkan mulai dan selesai?

**Jawaban awal (Other):**
> "dimulai awal September 2026 dan ditargetkan selesai akhir September 2026"

**Keberatan yang diajukan beserta buktinya:** disampaikan bahwa target ~3 minggu tidak realistis, dengan ukuran terukur dari source: 74 harness, 269 section, 15.063 step activity, 652 rule SQL, 86 stored procedure yang source-nya belum ada, 56 report, 12 REST + 6 DB Link, ditambah tabel auth/authz dan penugasan yang belum ada sama sekali, serta tim yang harus belajar Go, React, dan TypeScript sekaligus. Perkiraan yang diajukan: **9–15 bulan dengan tim 5–6 orang**.

**Pertanyaan lanjutan:** Target akhir September itu untuk apa tepatnya?

**Pilihan yang ditawarkan:**
1. Dokumen Steering yang selesai, implementasi menyusul
2. Satu modul percontohan yang jalan di produksi
3. Memang seluruh aplikasi, jadwal sudah ditetapkan manajemen
4. Perlu didiskusikan ulang setelah melihat angka

**Jawaban:** Memang seluruh aplikasi, jadwal sudah ditetapkan manajemen

**Pertanyaan lanjutan:** Bila scope harus dipangkas, mana yang paling mendesak?

**Jawaban (Other):** "semua modul sudah harus pindah"

**Keputusan akhir:** Target **seluruh modul selesai akhir September 2026** diterima sebagai keputusan manajemen dan menjadi dasar penyusunan Migration Strategy. Tidak ada pemangkasan scope.

**Risiko jadwal tetap didokumentasikan** beserta angkanya di **R-05**, bukan untuk menolak target, melainkan agar keputusan pemangkasan atau penambahan sumber daya di kemudian hari punya dasar ukuran yang jelas.

**Dampak ke Steering:** Migration Strategy · Risk Analysis · Testing Strategy

---

## Sesi 3 — 2026-09-08 — Penyusunan ADR dan Ticketing

Sesi ini menurunkan Steering dan BRD yang sudah disetujui menjadi dua artefak yang mengikat
pelaksanaan: ADR (`docs/adr/`) dan tiket pekerjaan (`docs/ticketing/`). Tidak ada kode
implementasi yang ditulis.

Dua kelompok keputusan: **D-31…D-35** dari GATE 0 (audit lingkungan dan rekonsiliasi angka), dan
**D-36…D-40** dari Ronde 1 grilling (Lapis 1 — status keputusan terbuka dan risiko).

Atas permintaan pemilik project, pertanyaan grilling disusun memakai kerangka **5W + 1H**
(Apa · Siapa · Kapan · Di mana · Mengapa · Bagaimana) dan selalu menyertakan pilihan jawaban.

> Catatan penomoran: `docs/interview-history.md` sudah memakai label "Sesi 3" untuk sesi
> Discovery BRD 2026-09-08. Di Decision Log ini, Sesi 3 adalah sesi ADR & Ticketing — secara
> kronologis sesi wawancara keempat. Penomoran `D-nn` tetap berurutan dan tidak bertabrakan.

---

### D-31 · Lokasi dan Format Artefak ADR & Tiket
**Status:** DECIDED

**Pertanyaan:** Di mana ADR dan tiket pekerjaan disimpan, dan format ADR mana yang dipakai? `docs/agents/domain.md:21` menyebut `docs/adr/` dibuat *lazily*; `docs/agents/issue-tracker.md:3` menetapkan `.scratch/` sebagai lokasi tiket, tetapi tiket ini akan ditinjau manajemen, harus ter-commit, dan diimpor ke GitLab.

**Pilihan yang ditawarkan:**
1. ADR di `docs/adr/`, tiket di `docs/ticketing/` — ter-commit dan terlihat di GitLab
2. Tiket di `.scratch/<fitur>/issues/` sesuai `issue-tracker.md` sekarang
3. Format ADR mengikuti template brief tugas (dengan field `Jenis` dan `Modul terdampak`)
4. Format ADR mengikuti `Sample/ADR Ticketing.md` apa adanya

**Jawaban:**
> "a. lokasi adr di docs/adr/
> b. lokasi ticketing di docs/ticketing/
> c. tulis apa adanya sesuai dengan format ADR yang ada di folder Sample"

**Keputusan akhir:** ADR di **`docs/adr/`**, tiket pekerjaan di **`docs/ticketing/`**. Format tiap ADR mengikuti `Sample/ADR Ticketing.md:280-297` apa adanya: `Status` · `Tanggal keputusan`/`Tanggal dokumen` · `Sifat` · `Pemilik keputusan` · `Jejak bukti` · `Terkait`, lalu `Konteks` · `Opsi yang dipertimbangkan` · `Keputusan` · `Rationale` · `Konsekuensi` (Positif / Negatif–utang teknis / Risiko yang diterima sadar) · `Pertanyaan terbuka`.

**Dua penyesuaian yang tidak terhindarkan:** (a) `Sample` menulis `Jejak bukti: decision-log Qnn` karena proyek TEKNO memakai skema `Qnn`; di sini dipakai `D-nn`, `FR-xx`, `R-nn` sesuai skema proyek ini. (b) Format `Sample` tidak punya field `Jenis` (baru / turunan `D-nn` / supersede `D-nn`) maupun `Modul terdampak`; keduanya ditaruh sebagai **kolom di tabel index `docs/adr/README.md`** agar tidak hilang.

**Konsekuensi:** `docs/agents/issue-tracker.md` harus disesuaikan dari `.scratch/` ke `docs/ticketing/` (dikerjakan di Fase 6).

**Dampak ke Steering:** tidak ada — ini keputusan tata letak artefak, bukan arsitektur.

---

### D-32 · Log Penggunaan Skill Dibiarkan Utuh
**Status:** DECIDED

**Pertanyaan:** Dua log skill hidup berdampingan — `docs/skill-usage.md` (sesi BRD 2026-09-08) dan `docs/Steering/18-SKILLS-USAGE-LOG.md` (sesi Steering 2026-09-07, sekaligus Lampiran D dokumen gabungan). Mana yang kanonikal untuk sesi ini?

**Pilihan yang ditawarkan:**
1. `docs/skill-usage.md` menjadi kanonikal; `18-SKILLS-USAGE-LOG.md` dibiarkan sebagai catatan sesi Steering
2. Gabungkan keduanya menjadi satu log
3. Biarkan keduanya utuh apa adanya

**Jawaban:** "e. biarkan utuh"

**Keputusan akhir:** Kedua log **dibiarkan utuh**, tidak digabung dan tidak direstrukturisasi. Keduanya sudah saling merujuk (`skill-usage.md:10` menyebut `18-SKILLS-USAGE-LOG.md` sebagai log sesi sebelumnya), sehingga hubungannya berurutan, bukan bertentangan.

**Terbuka:** di mana entri sesi ADR & Ticketing ini dicatat — ditanyakan pada Fase 6.

**Dampak ke Steering:** tidak ada.

---

### D-33 · Tanpa Version Control di Repository Ini
**Status:** DECIDED

**Pertanyaan:** Repo ini bukan repository VCS apa pun — `.git`, `.hg`, `.svn`, dan `.gitignore` semuanya tidak ada. Selama Fase 2–5 akan ditulis banyak dokumen; tanpa VCS tidak ada cara memperlihatkan diff untuk direview di setiap gate. Apakah `git init` dijalankan sekarang?

**Pilihan yang ditawarkan:**
1. `git init` sekarang, dengan `.gitignore` yang mengecualikan 297 MiB export XML
2. Repo git hanya untuk `docs/`, export XML tetap di luar
3. Tidak perlu VCS sekarang

**Jawaban:** "d. tidak perlu karena nantinya akan disiapkan CI/CD dari team gitlab"

**Keputusan akhir:** **Tidak ada inisialisasi VCS** oleh tim ini. Version control dan CI/CD disiapkan **Tim GitLab** pada tahap berikutnya. Repo dibiarkan apa adanya.

**Konsekuensi yang harus disadari:** (a) tidak ada diff yang dapat ditunjukkan antar gate — review dilakukan atas berkas utuh; (b) tidak ada perlindungan terhadap kehilangan pekerjaan; (c) keputusan mengecualikan export XML dari riwayat git berpindah menjadi tanggung jawab Tim GitLab, dan **harus disampaikan pada mereka sebelum commit pertama** — lihat **D-40**, karena export memuat kredensial plaintext.

**Dampak ke Steering:** Deployment Strategy — bagian pipeline perlu menyebut kepemilikan Tim GitLab.

---

### D-34 · Scope: Area Bengkel/Sparepart/Supplier dan Ruleset GKM Dikecualikan
**Status:** DECIDED

**Pertanyaan:** Audit GATE 0 menemukan **35 rule** yang tidak dimiliki modul mana pun: 31 activity bernama Bengkel/Sparepart (`ChooseCabangBengkelHE`, `UpdateSparepartHE_act`, `ValidasiSparepart`, `ValidationLoginBengkel_act`, dan lainnya) berruleset **GCNMFW** — ruleset Claim PNC sendiri — dan 4 activity master Supplier berruleset **GKM**, ruleset ketiga yang tidak pernah dibahas di D-03 maupun dokumen mana pun. Domain ini tidak ada di `CONTEXT.md`, tidak ada di 14 master, tidak ada di 32 modul, tidak ada di 39 `FR-*`. Padahal D-03 menetapkan bahwa scope adalah seluruh rule yang ada di export XML ini.

**Pilihan yang ditawarkan:**
1. Masuk scope — butuh modul dan `FR-*` baru
2. Keluar scope — butuh entri Decision Log yang membatasi D-03
3. Belum diputuskan — tiket terkait ditandai `needs-info`

**Jawaban:** "f. abaikan saja, tidak masuk dalam scope"

**Keputusan akhir:** Area **Bengkel, Sparepart, dan Supplier dikeluarkan dari scope migrasi**, termasuk seluruh 4 activity berruleset **GKM**. Keputusan ini **membatasi D-03** secara eksplisit: "seluruh rule yang ada di export" tidak mencakup area ini.

**Cakupan yang dikecualikan, terukur dari export:** 35 activity; 8 tabel database (`BENGKEL_HE`, `SPAREPART_HE`, `SPAREPART_HE_VIN_KEY`, `GCNM_M_SPAREPART_CATEGORY`, `GCNM_M_SPAREPART_TYPE`, `NOTIF_RANGKA_HE`, `PANEL_HE`, `LOKASI_PANEL_HE`); 5 harness; 20 section — termasuk alur persetujuan 4 tahap (Approval, lalu Approve atau Reject). Rincian di `docs/verifikasi-bukti-adr.md` §10.4.

**Konsekuensi terbuka:** sebagian activity yang dikecualikan adalah **pemanggil `SendEmailNotification`** yang sudah masuk perhitungan R-07 (`19-GAP-EXPORT-DETAIL.md`). Angka R-07 karena itu memuat pemanggil dari area di luar scope, dan perlu diperiksa saat R-07 direvisi. Juga: **siapa yang memiliki 35 rule ini setelah Pega dimatikan belum ditentukan.**

**Dampak ke Steering:** Module Breakdown · Domain Model · Risk Analysis (R-07) · CONTEXT.md (istilah Bengkel, Sparepart, Supplier sengaja tidak dimasukkan)

---

### D-35 · Angka Modul dan Requirement Mengikuti Bukti, Bukan Dokumen
**Status:** DECIDED

**Pertanyaan:** Rekonsiliasi GATE 0 menemukan angka yang bertentangan antar dokumen. Yang terpenting: `docs/BRD.md:71`, `:1290`, `:1478` dan `docs/understanding-log.md:69` menyebut **27 modul**, padahal enumerasi di kalimat yang sama menghasilkan **32** (F-1 sampai F-5 = 5, B-1 sampai B-14 = 14, S-1 sampai S-7 = 7, U-1 sampai U-6 = 6). `BRD §21.3` kriteria pertama — gerbang cutover tingkat sistem — memakai angka 27 itu. Selain itu `FR-S8` (Perkakas Uji Kesetaraan) **tidak punya baris modul** di `06-MODULE-BREAKDOWN.md`, dan tidak ada modul `S-8` di berkas mana pun. Angka mana yang dipakai untuk menulis ADR dan tiket?

**Pilihan yang ditawarkan:**
1. Pakai angka dokumen (27) supaya konsisten dengan BRD yang sudah disetujui
2. Pakai angka hasil verifikasi bukti, dan ajukan koreksi BRD sebagai usulan revisi
3. Tunda sampai BRD direvisi

**Jawaban:** "g. buat apa adanya sesuai dengan bukti yg kamu temukan, jangan ngasal"

**Keputusan akhir:** **Seluruh angka di ADR dan tiket mengikuti hasil verifikasi bukti**, bukan angka dokumen, dan setiap selisih dicatat sebagai usulan revisi yang menunggu persetujuan pemilik project. Berlaku untuk seluruh 40 kontradiksi di `docs/verifikasi-bukti-adr.md` §11 — di antaranya: **32 modul** (bukan 27); `FR-S8` tanpa modul pemilik; hardcode **66 email, 24 user ID, 8 ambang** (bukan 10, 4, 3); **26** inbox berbasis peran (bukan sekitar 20); **177** objek tabel (bukan 245); grid median **6 kolom** (bukan 18 sampai 27); dan `OFFSET 500000` yang **nol kemunculan** di export.

**Konsekuensi:** ADR dan tiket akan berisi angka yang berbeda dari BRD di beberapa tempat. Tiap perbedaan wajib menyebut kedua rujukan berdampingan agar tidak terbaca sebagai kelalaian.

**Dampak ke Steering:** Module Breakdown · Domain Model · NFR & Performance · Risk Analysis · dan BRD Bab 9, 20, 21, 23 — seluruhnya sebagai **usulan revisi**, bukan perubahan yang sudah dieksekusi.

---

### D-36 · Tanggal Komitmen dan Jalur Eskalasi ke Pihak Luar
**Status:** DECIDED

**Pertanyaan:** `I-05` (`docs/interview-history.md:202`) sudah menetapkan seluruh permintaan artefak **sudah dikirim tetapi belum satu pun diterima**, dan mencatat sendiri bahwa **tanggal komitmen dan jalur eskalasi belum ada** — sehingga "sudah diminta" dan "belum diminta" punya dampak jadwal identik. Bagaimana mekanisme yang mengubah "sudah diminta" menjadi tanggal yang bisa dijadwalkan? Pihaknya kini **enam**, bukan lima: DBA, Tim Pega, Tim pemilik sistem, Compliance, Tim Infra, dan **pemilik API HCC/HCQ** — yang terakhir baru muncul dari Fase 1 dan belum pernah dihubungi.

**Pilihan yang ditawarkan:**
1. Work owner meminta tanggal komitmen tertulis dari keenam pihak minggu ini, dan eskalasi ke Sponsor Project bila lewat
2. Sudah ada tanggal komitmen — akan disebutkan per pihak
3. Tidak ada mekanisme tanggal komitmen; jalan dengan apa yang ada
4. Eskalasi bukan wewenang work owner — harus lewat Sponsor Project sejak awal

**Jawaban:** Opsi 1

**Keputusan akhir:** Pemilik project meminta **tanggal komitmen tertulis** dari keenam pihak luar dalam minggu ini (mulai 2026-09-08), dengan **eskalasi ke Sponsor Project** bila tanggal itu terlewat.

**Konsekuensi untuk ADR dan tiket:** sampai tanggal komitmen diterima, seluruh ADR yang bergantung artefak pihak luar berstatus **`Proposed`** (bukan `Accepted`), dan tiket terkait berstatus **`needs-info`** dengan pemilik yang disebut eksplisit. Status ini diperbarui begitu tanggal komitmen masuk.

**Terbuka:** pemilik API HCC/HCQ belum pernah dihubungi — ini pihak luar ketujuh dalam daftar tindakan hari pertama yang sekarang berisi sepuluh butir.

**Dampak ke Steering:** Risk Analysis, bagian Ringkasan tindakan hari pertama · Migration Strategy Tahap 0

---

### D-37 · Kebijakan Modul yang Artefaknya Tidak Pernah Datang
**Status:** DECIDED

**Pertanyaan:** `BRD §21.4` menyatakan **FR-B5, FR-B7, FR-B9, FR-B10, FR-B12 tidak boleh dinyatakan diterima** sampai R-01 tertutup, dan `BRD:1210` menegaskan itu keputusan manajemen, bukan kompromi diam-diam oleh tim. Asumsi `A-2` (`BRD:1059`) mencatat bila source 64 procedure tidak pernah ada, R-01 berubah dari penundaan menjadi **kebuntuan**. Tiket kelima modul itu ditulis sekarang dengan status terhalang, atau ditunda seluruhnya?

**Pilihan yang ditawarkan:**
1. Tiket ditulis sekarang dengan `Kesiapan: terhalang R-nn`, tidak boleh `ready-for-agent`; ukuran penerimaan pengganti diputuskan manajemen tertulis bila artefak tidak datang sampai tanggal tertentu
2. Tiket kelima modul ditunda seluruhnya — tidak ditulis sampai artefak ada
3. Jalan dengan menyimpulkan perilaku dari perbandingan masukan-keluaran di staging, tanpa jaminan kelengkapan kasus tepi
4. Belum dapat diputuskan — harus dibawa ke manajemen dulu

**Jawaban:** Opsi 1

**Keputusan akhir:** Tiket untuk modul yang artefaknya belum ada **ditulis sekarang** dengan penanda `Kesiapan: terhalang <R-nn>`, dan **tidak boleh** berstatus `ready-for-agent`. Bila artefak tidak datang sampai tanggal yang ditetapkan, **ukuran penerimaan pengganti diputuskan manajemen secara eksplisit dan tertulis** — bukan diputuskan tim pengembang.

**Alasan menulis sekarang alih-alih menunda:** menulis tiketnya tidak berbiaya, dan justru membuat besarnya pekerjaan yang tertahan terlihat oleh manajemen. Menundanya menyembunyikan masalah.

**Dampak ke Steering:** Testing Strategy · Migration Strategy · Risk Analysis (R-01, R-14)

---

### D-38 · Empat Risiko Baru — R-16 sampai R-19
**Status:** DECIDED

**Pertanyaan:** Fase 1 memunculkan empat hal yang tidak tercakup risiko mana pun yang sudah ada. Penomoran risiko sekarang: `docs/Steering/16-RISK-ANALYSIS.md` memuat R-01 sampai R-12, sementara `docs/BRD.md` §20 dan `docs/migration-readiness.md` sudah menambah R-13, R-14, R-15 — yang **belum diserap Steering**. Apakah keempat kandidat diterima sebagai risiko bernomor, dan siapa pemiliknya?

**Pilihan yang ditawarkan:**
1. Terima keempatnya sebagai R-16, R-17, R-18, R-19, meneruskan penomoran BRD
2. Terima sebagian saja
3. Jangan buat risiko baru; masukkan sebagai catatan di bawah R-01 dan R-07
4. Tunda sampai Tim Pega menjawab soal export ulang

**Jawaban:** Opsi 1

**Keputusan akhir:** Empat risiko baru diterima, meneruskan penomoran dari R-15:

| ID | Isi | Pemilik |
|---|---|---|
| **R-16** | **Export Pega tidak lengkap** — sekitar 242 rule dirujuk tetapi tidak ada di export, terberat **116 When rule buatan sendiri**. Tujuh tipe rule tidak diaudit `19-GAP` sama sekali: When, Flow Action, Section, Data Transform, Ticket, Report Definition, Function library | Tim Pega |
| **R-17** | **Kredensial aktif berada di dalam export** — 3 password SMTP di 31 lokasi ditambah kredensial OAuth, semuanya plaintext; ditambah `UseSSL=false` pada seluruh kemunculannya | Tim Infra / Security |
| **R-18** | **Dua integrasi BRI menunjuk host sandbox** di ruleset produksi | Work Owner dan tim integrasi |
| **R-19** | **Cacat aturan uang direplikasi bila P-5 dipatuhi buta** — toleransi spreading berupa pencocokan substring, ambang Rp 50 juta dengan tiga operator berbeda, penyesuaian 7 jam asimetris di dalam satu kondisi validasi | Work Owner |

**Alasan tidak digabung ke risiko yang ada:** R-01 mencakup objek database dan R-07 mencakup activity; **116 When rule adalah kategori ketiga** yang tidak tercakup keduanya, dan dampaknya lebih luas — memblokir percabangan bisnis di hampir semua modul, bukan lima modul. R-17 bukan risiko migrasi sama sekali: ia hidup sekarang bahkan bila migrasi dibatalkan. Menggabungkan keempatnya akan menyembunyikan pemilik yang berbeda.

**Dampak ke Steering:** Risk Analysis — perlu menyerap R-13, R-14, R-15 dari BRD **dan** menambah R-16 sampai R-19; total menjadi **19 risiko**, bukan 12. Diajukan sebagai usulan revisi.

---

### D-39 · Permintaan Export Ulang Pega Berbasis Product Rule
**Status:** DECIDED

**Pertanyaan:** Permintaan yang tercatat sekarang adalah mengirim 40 activity dan 3 router. Fase 1 membuktikan gapnya sekitar 242 rule, dan daftar itu sendiri hanya **batas bawah** — 65 step `Apply-DataTransform` punya `pyParamArray` kosong sehingga nama Data Transform yang dipanggil tidak selalu terbaca. Ditambah dugaan cacat proses export: `Activity/SendEmailNotification-Act.xml` berisi rule `CompressImage_Act` dan `Activity/NotificationPAYDI-Act.xml` berisi `JobSendEmailNotificationPAYDI`. Dan tidak ada direktori sama sekali untuk Access Group, Role, Privilege, Operator, Decision Table, Agent, maupun Job Scheduler — padahal `FR-F3` bertumpu pada 22 access group dan R-02 pada job terjadwal. Bagaimana bentuk permintaan yang benar?

**Pilihan yang ditawarkan:**
1. Ubah permintaan menjadi export ulang berbasis Product rule dengan dependensi, dan minta konfirmasi cacat export
2. Minta keduanya: export ulang **dan** daftar 242 rule sebagai jaring pengaman
3. Tetap minta daftar rule saja
4. Tim Pega tidak akan menuruti; jalan dengan export yang ada dan gali aturan bisnis dari wawancara

**Jawaban:** Opsi 2

**Keputusan akhir:** Permintaan ke Tim Pega berisi **dua bagian sekaligus**: (a) **export ulang lengkap** memakai *Product rule* atau application-based export dengan opsi *include dependent rules*, dan (b) **daftar sekitar 242 rule** yang dirujuk tetapi tidak ada, sebagai **alat verifikasi** bahwa export ulangnya benar-benar lengkap. Sekalian diminta konfirmasi apakah dua berkas yang berisi rule lain menandakan **seluruh 902 berkas perlu divalidasi ulang**.

**Alasan meminta keduanya:** export ulang adalah cara yang benar, tetapi bila Tim Pega hanya dapat memberi sebagian, daftar 242 rule menjadi satu-satunya cara membuktikan apa yang masih kurang.

**Konsekuensi metode:** inventaris rule untuk seluruh pekerjaan berikutnya dibangun dari elemen `pyRuleName`, **bukan dari nama berkas** — karena nama berkas terbukti tidak dapat dipercaya.

**Dampak ke Steering:** Risk Analysis (R-16, R-07, R-04) · `19-GAP-EXPORT-DETAIL.md` perlu bagian Cara Meminta ke Tim Pega yang setara dengan bagian untuk DBA

---

### D-40 · Kredensial di Dalam Export — Menunggu Tim Infra/Security
**Status:** OPEN — menunggu Tim Infra / Security

**Pertanyaan:** Export memuat **3 password SMTP unik di 31 lokasi** dan sepasang kredensial OAuth mitra, semuanya plaintext, di berkas yang direncanakan masuk GitLab. Ditambah `UseSSL` bernilai `false` pada **seluruh 14** kemunculan elemen itu sementara 16 dari 31 lokasi memakai port 587. Bagaimana perlakuannya?

**Pilihan yang ditawarkan:**
1. Kredensial dianggap aktif sampai terbukti sebaliknya: minta rotasi, dan export dikecualikan dari repo git lewat `.gitignore` sebelum commit pertama
2. Kredensial sudah mati — tidak perlu tindakan, boleh disebut di ADR dengan `berkas:baris`
3. Perlu dicek dulu ke Tim Infra/Security sebelum memutuskan
4. Jangan tulis temuan ini di dokumen yang akan di-commit sama sekali

**Jawaban:** Opsi 3, disertai permintaan lampiran detail lokasi temuan password SMTP

**Keputusan akhir:** **BELUM DIPUTUSKAN — pertanyaan terbuka.** Keputusan menunggu Tim Infra/Security. Pemilik keputusan: **Tim Infra / Security**, diteruskan oleh Work Owner.

Daftar lengkap lokasi disusun sebagai berkas serah-terima terpisah, **sengaja tidak disimpan di dalam repository** karena ia memuat peta lokasi kredensial. Nilai kredensial **tidak direproduksi** di berkas mana pun — hanya `berkas:baris`, nama elemen, panjang karakter, dan sidik-ringkas untuk membedakan satu nilai dari yang lain.

**Yang diblokir oleh keputusan ini:**
- ADR proses tentang penanganan secrets dan konfigurasi tiga lapis — dapat ditulis, tetapi berstatus `Proposed`.
- Keputusan apakah `docs/verifikasi-bukti-adr.md` §7.6, yang sudah memuat lokasi tanpa nilai, boleh tetap ada saat repo di-commit.
- Instruksi ke Tim GitLab soal `.gitignore` sebelum commit pertama — lihat **D-33**.

**Enam hal yang perlu dijawab Tim Infra/Security:** apakah 3 password SMTP masih aktif · apakah `UseSSL=false` pada port 587 masih berlaku di produksi · apakah kredensial OAuth BRI itu sandbox atau produksi · bagaimana export 297 MiB boleh disimpan dan apakah boleh masuk GitLab · siapa pemilik Authentication Profile `ServiceRekKlaimPNC` dan `SERVICEKLAIMKBRU` · apakah lokasi temuan boleh dicantumkan di dokumen arsitektur yang di-commit.

**Dampak ke Steering:** Security · Cross-Cutting (Configuration) · Deployment Strategy · Risk Analysis (R-17)

---

### D-41 · Cakupan Tiket Penuh
**Status:** DECIDED

**Pertanyaan (Ronde 2 Q6):** Modul mana yang ditiketkan penuh dengan acceptance criteria berangka, dan mana yang cukup stub backlog? Menulis AC berangka untuk 33 modul saat 19 risiko masih terbuka akan menghasilkan tiket yang mengarang.

**Pilihan yang ditawarkan:**
1. Lima modul ditiketkan penuh (`F-1`, `F-2`, `F-5`, `U-2`, `S-5`); `F-3`, `F-4`, `U-1` penuh untuk bagian yang buktinya lengkap dan `needs-info` untuk bagian terhalang; 24 modul sisanya stub backlog
2. Kedelapan modul gelombang 1 ditiketkan penuh apa adanya
3. Hanya lima modul yang benar-benar tidak terhalang; `F-3`, `F-4`, `U-1` seluruhnya stub backlog
4. Seluruh 33 modul ditiketkan penuh

**Jawaban:** Opsi 1, dengan tiga permintaan lampiran — *"untuk F-3 tolong list access group apa saja yang perlu saya lampirkan"* · *"untuk F-4 sudah dilampirkan isi table EMAILKOMITE pada file emailkomite.csv"* · *"untuk U-3 tolong list harness yang belum ada dan beritahu call darimana"*

**Jawaban lanjutan (bersyarat):** *"tetapi Opsi 1 jika masih ada halangan, tapi lampirkan detail kendalanya. Jika sudah tidak ada halangan, Opsi 4."*

**Keputusan akhir:** Syarat diperiksa dan **halangan masih ada di 32 dari 33 modul**, sehingga **Opsi 1 berlaku**. Matriks kendala per modul disusun sebagai `docs/verifikasi-bukti-adr.md` §15, memisahkan **kendala artefak** (dengan nama artefaknya) dari **kendala keputusan** (dengan pertanyaannya).

Rekapitulasi verdict: **PENUH 1 modul** (`U-2`) · **SEBAGIAN 24 modul** · **TERHALANG 8 modul** (`F-3` identitas, `B-5`, `B-7`, `B-11`, `S-3`, `S-4`, `S-7`, `S-8`, `U-6`).

Tiga lampiran yang diminta sudah diserahkan: daftar 22 access group + 5 When rule peran + penugasan operator→grup (F-3) · analisis 21 kolom `POOLDATA.EMAILKOMITE` beserta tangga jenjang PA/TRAVEL/NONMBU/BONDING (F-4) · daftar 7 harness portal beserta baris rujukan di `Navigation/pyCaseWorkerNavigation-Navigation.xml` (U-1/U-3).

**Temuan yang mengubah premis:** **R-01 bukan lagi penghalang utama.** Dari lima modul yang `BRD §21.4` nyatakan tidak dapat diterima sampai R-01 tertutup, `B-9`/`B-10`/`B-12` naik ke SEBAGIAN karena source procedure-nya datang. `B-5` dan `B-7` tetap terhalang, tetapi oleh **isi dua tabel master** (`GCNM_FEE_SCALE`, `m_currencystandard`) dan oleh **keputusan bisnis yang belum diambil** — bukan oleh R-01. Penghalang utama sekarang **R-16** (±397 artefak, terberat 137 When rule) dan keputusan bisnis.

**Dampak ke Steering:** Module Breakdown · Migration Strategy · Risk Analysis (R-01, R-16)

---

### D-42 · Modul S-8 — Perkakas Uji Kesetaraan Masuk Gelombang 1
**Status:** DECIDED

**Pertanyaan (Ronde 2 Q7):** `FR-S8` punya requirement tetapi **tidak punya baris modul** — tidak ada `S-8` di berkas mana pun, dan `06-MODULE-BREAKDOWN.md` tidak menyebut kata "kesetaraan" sekali pun. Beri nomor modul apa, dan masuk gelombang mana?

**Pilihan yang ditawarkan:**
1. Beri nomor `S-8`, masuk gelombang 1 bersama modul fondasi
2. Beri nomor `S-8`, tetap di gelombang 6 bersama laporan
3. Bukan modul terpisah — jadikan bagian dari `F-2`
4. Belum diputuskan

**Jawaban:** Opsi 1

**Keputusan akhir:** `FR-S8` menjadi modul **`S-8` Perkakas Uji Kesetaraan**, masuk **gelombang 1** bersama modul fondasi. Total modul menjadi **33**, bukan 32.

**Alasan:** ia satu-satunya modul yang **memblokir cutover setiap modul lain**. `BRD §21.1` menjadikan uji kesetaraan otomatis sebagai gerbang pertama setiap modul; menaruhnya di gelombang 6 berarti tidak ada satu modul pun dapat lulus gerbang 1 sampai gelombang 6, yang membatalkan gagasan Strangler Fig (D-05).

**Terbuka:** perkakas ini menuntut kemampuan **menembak Pega dari luar** untuk membandingkan hasil. Siapa yang boleh melakukannya, di lingkungan mana, dan apakah boleh membaca produksi read-only — belum ditanyakan (Lapis 4). Karena itu tiket `S-8` berstatus **TERHALANG** meski modulnya di gelombang 1.

**Pengecualian yang harus dicatat:** dua modul **tidak punya baseline Pega** untuk diuji setara — `F-3` (HCC/HCQ nol jejak di export) dan `S-5` (sistem lama tidak punya jejak audit nilai). Untuk keduanya, `BRD §21.2` kriteria #3 tidak dapat diberlakukan dan harus diganti ukuran lain.

**Dampak ke Steering:** Module Breakdown §3 dan §5 · Testing Strategy · BRD Bab 9 dan 21

---

### D-43 · Pembaca Tiket — Manusia Junior
**Status:** DECIDED

**Pertanyaan (Ronde 2 Q8):** Tiket ditulis untuk siapa? Ini menentukan kedalaman acceptance criteria dan seberapa eksplisit constraint teknis ditulis. `D-09` menetapkan tim adalah developer Pega internal yang dilatih ulang.

**Pilihan yang ditawarkan:**
1. Keduanya — badan tiket untuk manusia junior, bagian verifikasi dan constraint cukup eksplisit untuk agent AFK
2. Manusia junior saja
3. Agent AFK saja
4. Manusia junior sekarang; format agent ditambahkan nanti

**Jawaban:** Opsi 2

**Keputusan akhir:** Tiket ditulis **untuk manusia junior** yang sedang belajar Go, sesuai `D-09`. Konsekuensinya: badan tiket menyertakan **konteks dan alasan**, bukan hanya perintah; istilah domain dijelaskan atau dirujuk ke `CONTEXT.md`; dan constraint teknis ditulis preskriptif sesuai tuntutan `D-09` (struktur folder, penamaan, pola baku yang seragam).

**Yang tidak dilakukan:** tiket **tidak** dioptimalkan untuk agent AFK. Bagian `Rencana verifikasi` tetap memuat perintah nyata yang dapat dijalankan, tetapi karena syarat Definition of Ready menuntut verifikasi dapat dilakukan **tanpa pengetahuan pribadi** — bukan karena menyasar agent.

**Dampak ke Steering:** tidak ada — ini keputusan bentuk artefak.

---

### D-44 · Tim Pelaksana — Ditentukan Setelah Tiket Direview
**Status:** OPEN — menunggu review tiket

**Pertanyaan (Ronde 2 Q9):** Berapa developer dan bagaimana komposisinya? `D-09` menetapkan latar belakang tim tetapi **tidak ada satu pun dokumen yang menyebut jumlahnya**. `06-MODULE-BREAKDOWN.md:103-106` mencatat tiga rantai kritis yang tidak dapat diparalelkan, sehingga jumlah orang menentukan bentuk papan tiket.

**Pilihan yang ditawarkan:**
1. Angka disebutkan — berapa backend, berapa frontend, berapa penuh waktu
2. Belum ditetapkan; tiket ditulis netral terhadap jumlah orang
3. Sama dengan tim Pega yang sekarang merawat aplikasi ini
4. Akan ditambah atau dikurangi tergantung hasil review tiket ini

**Jawaban:** Opsi 4

**Keputusan akhir:** Jumlah dan komposisi tim **ditentukan setelah tiket direview** — papan tiket menjadi bahan keputusan sumber daya, bukan sebaliknya.

**Konsekuensi yang mengikat bentuk tiket:** tiket harus ditulis **netral terhadap jumlah orang**, dan `docs/ticketing/README.md` wajib menampilkan **tiga rantai kritis** (`06-MODULE-BREAKDOWN.md:103-106`) beserta dependency antar tiket secara eksplisit — karena itulah yang memungkinkan pembaca menyimpulkan berapa orang dibutuhkan untuk paralelisasi.

**Risiko yang harus disadari:** `06-MODULE-BREAKDOWN.md:76` menyatakan `U-2` adalah investasi terpenting di frontend. Bila tidak ada personel khusus frontend, `U-2` bersaing waktu dengan `F-2` di rantai kritis ketiga (`U-2 → U-3/U-4/U-5`). Ini harus terlihat di papan tiket sebagai risiko urutan.

**Dampak ke Steering:** Migration Strategy · Risk Analysis (R-05, R-11)

---

### D-45 · Fase 1 Dijalankan Ulang terhadap Snapshot Baru
**Status:** DECIDED

**Pertanyaan (Q11):** `docs/verifikasi-bukti-adr.md` sahih untuk snapshot 2.167 rule. Pada 2026-09-09 export bertambah menjadi 2.389 rule (kemudian terhitung 2.634 berkas setelah `InboxAutoClaim/` diekstrak) dan folder `Database/` muncul dengan 63 objek Oracle + 2 master CSV yang belum pernah dibaca. Fase 1 dijalankan ulang atau tidak?

**Pilihan yang ditawarkan:**
1. Jalankan ulang audit ketersediaan dan area yang paling terdampak saja, lalu perbarui dokumen dan lanjut
2. Jalankan ulang seluruh Fase 1 dari nol terhadap snapshot baru
3. Lanjut dengan dokumen yang ada, tandai setiap angka sebagai "per snapshot 2026-09-08"
4. Tunggu sampai Tim Pega menyatakan pengirimannya lengkap

**Jawaban:** Opsi 1

**Keputusan akhir:** Audit ketersediaan dan area paling terdampak dijalankan ulang. Hasilnya `docs/verifikasi-bukti-adr.md` §14 dan §15. Bagian §0–§13 **dibiarkan apa adanya** sebagai catatan keadaan snapshot 2.167, dengan penanda koreksi di tempat yang dirujuk balik.

**Hasil yang paling berdampak:** **R-02 tertutup** (5 job + 1 agent ditemukan di folder `Job Scheduler/` dan `Agents/` yang sebelumnya tidak ada, seluruh activity target-nya ada) · **R-06 tertutup** (`v_sts_claim.csv`, 33 kode `1134`–`1166`, bukan 10 kode `1142`–`1151`) · **D-14 terjawab** (`emailkomite.csv`, 21 kolom) · **R-01 62 dari 64 berkas** tetapi 12 dependensinya belum ikut · **R-16 memburuk** dari 116 menjadi 137 When rule hilang.

**Enam kesalahan saya sendiri terkoreksi** dan tercatat di §14.2 dan §15.1, di antaranya satu klaim yang dicabut seluruhnya (mekanisme penyimpanan dokumen tetap **tiga**, bukan empat) dan tiga kesimpulan arti kode status yang terbukti salah dari sembilan.

**Alasan tidak memilih opsi 4:** `R-09` mencatat sistem sumber masih aktif berubah — export bertambah dua kali selama sesi ini. "Lengkap" mungkin tidak pernah tiba.

**Dampak ke Steering:** Risk Analysis (R-01, R-02, R-06, R-16) · Gap Export Detail

---

### D-46 · Kewenangan Review dan Pernyataan Selesai Tiket
**Status:** DECIDED oleh work owner — dicatat dengan konsekuensi terbuka

**Pertanyaan (Ronde 2 Q10):** Siapa reviewer, dan siapa yang berwenang menyatakan tiket selesai? Untuk gerbang 2 UAT, `I-06` sudah menetapkan user bisnis per peran + Business Owner. Tetapi modul fondasi (`F-1`, `F-2`, `F-5`, `S-5`) **tidak punya peran pemakai** — tidak ada user bisnis yang membuka layar "Akses Data". Format tiket mewajibkan field `Peran penguji gerbang 2`, yang akan kosong tanpa jawaban.

**Pilihan yang ditawarkan:**
1. Lead engineer me-review teknis; Work Owner menyatakan tiket selesai. Untuk modul fondasi, gerbang 2 diganti "diterima Lead engineer + Work Owner"
2. Reviewer dan yang menyatakan selesai adalah orang yang sama
3. Work Owner me-review dan menyatakan selesai untuk semua tiket
4. Belum ada mekanisme review; ditetapkan saat implementasi mulai

**Jawaban:** Menolak keempat pilihan.
> *"Saya tidak ingin opsi 1 sampai 4, yang saya mau : tiketnya langsung kamu review sendiri dan nyatakan selesai, namun kamu harus ketat berarti kamu harus testing hasilnya dari awal sampai akhir, sampai statusnya hijau/aman."*

**Keberatan yang diajukan beserta buktinya.** Disampaikan tiga konsekuensi, masing-masing dengan rujukan:

1. **Tidak ada kode untuk diuji.** `docs/AGENTS.md:9` dan aturan tugas ini menetapkan repo belum punya kode implementasi dan penulisan kode dilarang. "Testing hasilnya sampai hijau" tidak punya objek uji berupa sistem berjalan; yang dapat diuji adalah dokumennya.
2. **`BRD §21.2` kriteria #4** mewajibkan gerbang 2 berupa UAT oleh user bisnis dari peran yang memakai modul, dan `I-06` — jawaban work owner sendiri — sudah menetapkan user bisnis per peran + Business Owner sebagai penanggung jawabnya.
3. **`BRD §21.2` kriteria #9 dan #12** menuntut jejak audit dan kepemilikan tunggal; keduanya kehilangan makna bila pelaksana menyatakan pekerjaannya sendiri selesai.

Ditambah satu bukti dari sesi ini: **sepanjang 2026-09-08 sampai 2026-09-10, sepuluh kesimpulan saya terbukti salah** — tiga arti kode status, satu klaim mekanisme penyimpanan dokumen, satu keberadaan When rule, satu status R-01, satu arah parameter, satu pemetaan job-activity, dan dua positif palsu metode. Yang mengoreksi seluruhnya adalah master data dan source yang datang kemudian, bukan pembacaan ulang yang lebih teliti.

**Penegasan ulang work owner:** pertanyaan diajukan ulang dalam bentuk terpisah (`selesai-ditulis` versus `selesai-dikerjakan`) dan work owner **menegaskan jawaban yang sama**. Keputusan dihormati.

**Keputusan akhir:** **Lead engineer (agen) menjadi reviewer sekaligus yang menyatakan tiket selesai.** Kewajiban yang mengikat sebagai gantinya: setiap tiket dan ADR wajib melewati **suite verifikasi yang benar-benar dijalankan**, dengan hasil dilaporkan **hijau atau merah per butir beserta angkanya** — bukan penilaian kualitatif. Tidak ada artefak dinyatakan selesai selama ada satu butir merah.

Butir yang diuji, seluruhnya dapat dijalankan tanpa kode implementasi:

| # | Butir | Cara uji |
|---|---|---|
| 1 | Setiap `berkas:baris` yang dikutip benar-benar ada dan isinya sesuai | diverifikasi ulang satu per satu terhadap export terbaru |
| 2 | Traceability dua arah: setiap `FR-xx` dalam cakupan punya ≥1 tiket; setiap tiket punya ≥1 `FR-xx` **dan** rule Pega yang digantikan | dihitung, dilaporkan sebagai angka |
| 3 | Setiap tiket ber-`ready-*` lolos seluruh butir Definition of Ready | daftar periksa per tiket |
| 4 | Setiap istilah domain di tiket sudah ada di `CONTEXT.md` | selisih istilah |
| 5 | Nol nilai sensitif di dokumen yang akan di-commit | pemindaian pola kredensial, email, hostname, data nasabah |
| 6 | Setiap `D-nn`, `R-nn`, `FR-xx` yang dirujuk benar-benar ada dan statusnya mutakhir | selisih terhadap Decision Log dan Risk Analysis |
| 7 | Setiap tiket TERHALANG menyebut penghalangnya dengan nama artefak atau `berkas:baris` | pemindaian |
| 8 | Nol tiket mengklaim status penyelesaian pekerjaan | pemindaian |

**Konsekuensi yang diterima secara sadar, dicatat sebagai keputusan work owner:**
- Tidak ada mata manusia di antara tiket yang ditulis agen dan pekerjaan yang dijalankan atasnya.
- Gerbang 2 (`BRD §21.2` #4) tidak diberlakukan pada tahap penulisan tiket; bila ia tetap diberlakukan pada tahap implementasi, itu keputusan terpisah yang belum diambil.
- Field `Peran penguji gerbang 2` pada setiap tiket diisi sesuai peran bisnis yang teridentifikasi di Lapis 5, dan menjadi **rencana** untuk tahap implementasi — bukan pernyataan bahwa gerbang itu sudah dilewati.

**Terbuka:** siapa yang menyatakan **pekerjaan implementasi** selesai — belum diputuskan, dan tidak dapat diputuskan sekarang karena kodenya belum ada.

**Dampak ke Steering:** Testing Strategy · BRD Bab 21 (usulan revisi: pengecualian gerbang untuk tahap penulisan artefak)

---

### D-47 · Model Penjenjangan Komite — Kumulatif, `LIMIT_TOP` sebagai Validasi Master
**Status:** DECIDED

**Pertanyaan (Ronde 3 Q12):** Klaim Rp 75.000.000 NONMBU masuk jenjang komite mana? Query `RDB List/EmailKomiteBerjenjangSimasnet_sql-SQL.xml:98` hanya memfilter `LIMIT_BOTTOM <= nilai` — kolom `LIMIT_TOP` yang ada di master tidak dipakai — dan rentang NONMBU di `POOLDATA.EMAILKOMITE` saat itu tumpang tindih, sehingga Rp 75.000.000 memenuhi dua baris dan Rp 50.000.000 memenuhi empat.

**Pilihan yang ditawarkan (versi pertama):**
1. Rentang tertutup `LIMIT_BOTTOM <= nilai <= LIMIT_TOP`, master NONMBU dirapikan, tie-breaker dibuang
2. Hanya batas bawah, dengan aturan "DEGREE tertinggi yang memenuhi menang"
3. Kumulatif — semua jenjang yang memenuhi harus menyetujui
4. `dbms_random.value` memang disengaja — pertahankan, dan AC `B-7` tidak boleh deterministik

**Jawaban pertama:** Opsi 1, disertai keterangan *"sudah diperbaiki isinya pada emailkomite.csv"*

**Kerangka pertanyaan dikoreksi sebelum jawaban dipakai.** Setelah jawaban diterima, penelusuran lanjutan membuktikan **kerangka pertanyaannya salah**: `LIMIT_BOTTOM <=` bukan cacat, melainkan **mekanisme penjenjangan kumulatif**.

| Bukti | Isi |
|---|---|
| `Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459` | `childPageKomite.KomiteLoop := TempRDBSearchEmailKomite.pxResultCount` |
| `When/IsKomiteLoop-When.xml` | `.AcceptStatus = "1"` **dan** `.KomiteCount <= .KomiteLoop` |

Jumlah jenjang persetujuan **sama dengan jumlah baris yang dikembalikan query**, dan flow memutar `KomiteCount` dari 1 sampai `KomiteLoop`. Jadi `LIMIT_BOTTOM <= nilai` mengembalikan seluruh jenjang sampai tingkat nilai itu — klaim kecil sedikit approver, klaim besar banyak approver. `LIMIT_TOP` tidak dibutuhkan untuk semantik kumulatif.

**Menerapkan Opsi 1 (rentang tertutup) akan mengembalikan tepat satu baris → `KomiteLoop = 1` → hanya satu jenjang persetujuan berapa pun nilai klaim** — menghapus penjenjangan yang menjadi inti D-14 dan `BRD §11.4`.

**Koreksi kedua atas laporan agen:** `dbms_random.value` **tidak** berlaku umum. Dari **17 query** yang membaca `EMAILKOMITE`: 8 memfilter `LIMIT_BOTTOM`, **0 memfilter `LIMIT_TOP`**, 4 memfilter `TYPE_KOMITE`, dan **hanya 2** memakai `dbms_random` — keduanya jalur Simasnet (`EmailKomiteBerjenjangSimasnet_sql`, `GetEmailKomiteSimasnet_Reject`). Jalur utama (`EmailKomiteBerjenjang_sql`, PA, Travel, Adjuster, Salvage) memakai `ORDER BY DEGREE` saja — **deterministik**. `EmailKomiteBerjenjangBonding_sql` tidak memfilter `LIMIT` sama sekali.

**Pilihan yang ditawarkan (versi kedua):**
1. Pertahankan model kumulatif; `LIMIT_TOP` menjadi dokumentasi, tidak dipakai query
2. Kumulatif, **dan** `LIMIT_TOP` ikut divalidasi sebagai pemeriksaan integritas master — menolak master yang rentangnya bolong atau tumpang tindih — tetapi tidak dipakai memilih baris
3. Tetap rentang tertutup — satu jenjang per klaim, penjenjangan dihapus
4. Lain

**Jawaban:** Opsi 2

**Keputusan akhir:** **Model kumulatif dipertahankan.** Jumlah jenjang persetujuan = jumlah baris `POOLDATA.EMAILKOMITE` yang memenuhi `LIMIT_BOTTOM <= nilai klaim` (ditambah filter `STS_ADJ`, `STS_AKTIF`, `TYPE_BUSINESS`, dan `TYPE_KOMITE` pada query yang memakainya), diurutkan `DEGREE`. Perilaku identik dengan Pega, sehingga aman untuk gerbang 1.

**`LIMIT_TOP` dipakai sebagai validasi integritas master data**, bukan untuk memilih baris: sistem baru menolak master yang rentangnya tumpang tindih atau berlubang antar `DEGREE` dalam satu `TYPE_BUSINESS` + `TYPE_KOMITE`.

**Isi master sudah diperbaiki work owner** (`Database/emailkomite.csv`, 2026-09-11). Tangga kumulatif yang dihasilkan, terverifikasi:

| Lini | Batas bawah per jenjang | Jenjang maksimum |
|---|---|---|
| **PA** | `0` · `10.000.001` · `50.000.001` · `100.000.001` | **4** |
| **TRAVEL** | `0` · `50.000.001` · `100.000.001` | **3** |
| **NONMBU** `TYPE_KOMITE=2` | `0` · `100.000.001` · `500.000.001` · `1.000.000.001` | **4** |
| **NONMBU** `TYPE_KOMITE=1` | `50.000.001` | 1 |
| **BONDING** | — query tidak memfilter `LIMIT` | 1 |

Acceptance criteria yang dapat langsung dipakai: **PA Rp 10.000.000 → 1 jenjang; Rp 10.000.001 → 2 jenjang; Rp 50.000.001 → 3 jenjang; Rp 100.000.001 → 4 jenjang.** Ambiguitas Rp 75.000.000 hilang karena perbaikan data, bukan karena perubahan aturan.

**Terbuka — belum dijawab, dibawa ke ronde berikutnya:**
- `TYPE_KOMITE=2` NONMBU tidak punya baris ber-batas-bawah antara `0` dan `100.000.001`, sehingga Rp 50 juta–Rp 100 juta hanya 1 jenjang di jalur itu, sementara `TYPE_KOMITE=1` punya baris `50.000.001`. Benar begitu?
- Di atas **Rp 200.000.000** tidak ada baris untuk PA dan TRAVEL.
- `TYPE_KOMITE` dipilih di `SetEmailKomite` lewat **syarat bernama orang** — dibawa sebagai peran atau dibuang?

**Dampak ke Steering:** Domain Model (master Ambang Komite) · Module Breakdown B-7 · BRD Bab 11.4 · `20-DETAIL-KOMITE-DBLINK.md` §1.2 dan §1.4 (struktur 21 kolom, bukan 14)

---

### D-48 · Basis Kurs dan Perilaku Saat Kurs Tidak Ditemukan
**Status:** DECIDED

**Pertanyaan (Ronde 3 Q13):** `Database/GETCURRENCYSTANDARD.fnc` menerima parameter tanggal `i_tgl_kurs` (`:3`) tetapi **tidak pernah memakainya** — yang dipakai `TRUNC(sysdate)` (`:14`). Pemanggilnya mengirim **tanggal kerugian** (`RDB List/GetDataTrytyInwardFromUploadData-SQL.xml:34`, `:38`; `Activity/BrowseDataDetailKlaimXOLAndReas-Act.xml:3289`, `:3320`, `:3331`). Dan bila kurs tidak ditemukan, fungsi mengembalikan **`1`** (`:20-22`) — valuta asing diperlakukan 1:1 terhadap rupiah, sehingga ambang komite Rp 50 juta/Rp 30 juta dan Notice of Large Losses Rp 1 miliar **tidak terpicu**.

**Pilihan yang ditawarkan — (a) basis kurs:**
1. Kurs pada tanggal kerugian
2. Kurs hari eksekusi
3. Kurs pada tanggal registrasi klaim

**Pilihan yang ditawarkan — (b) bila kurs tidak ditemukan:**
1. Tolak transaksi dengan galat eksplisit
2. Pakai kurs terakhir yang tersedia berapa pun tanggalnya
3. Pertahankan `RETURN 1`

**Jawaban:** (a) Opsi 1 · (b) Opsi 1

**Keputusan akhir:** Konversi ke IDR memakai **kurs yang berlaku pada tanggal kerugian**, bukan kurs hari eksekusi. Bila kurs untuk mata uang dan tanggal itu tidak ditemukan, transaksi **ditolak dengan galat eksplisit** — tidak ada nilai default.

**Konsekuensi yang harus disetujui sebelum gerbang 1 dijalankan:** memperbaiki basis kurs membuat uji kesetaraan **menampilkan selisih pada seluruh data historis valuta asing**, karena laporan lama memakai kurs hari eksekusi. Selisih itu wajib dinyatakan lebih dulu sebagai perbaikan yang direncanakan, bukan ditemukan sebagai kejutan saat pengujian.

**Konsekuensi kedua:** menolak transaksi saat kurs tidak ada akan **menampakkan kegagalan yang selama ini senyap**. Klaim valuta asing yang dulu lolos tanpa komite kini akan ditolak sampai kursnya dilengkapi. Ini perbaikan yang diinginkan, tetapi berdampak operasional dan perlu disiapkan.

**Terbuka:** isi tabel `m_currencystandard` belum ada di repo — masih diminta ke DBA. Tanpa itu `B-5` tetap tidak dapat diuji.

**Dampak ke Steering:** Domain Model (master Mata Uang & Kurs) · BRD Bab 11.4 (status ✅ pada baris konversi kurs perlu diturunkan) · Risk Analysis (R-19)

---

### D-49 · Kebijakan atas Sepuluh Cacat Aturan Uang
**Status:** DECIDED

**Pertanyaan (Ronde 3 Q14):** `P-5` mewajibkan hasil identik Pega **kecuali perbaikan yang diputuskan eksplisit**. Daftar itu berisi empat butir (`requirement-summary.md:26`). Verifikasi bukti menemukan **sepuluh cacat aturan uang**. Mana yang direplikasi, mana yang diperbaiki?

**Pilihan yang ditawarkan:**
1. Setujui seluruh usulan; butir 7 ditanyakan terpisah
2. Setujui sebagian — sebutkan nomor mana yang direplikasi apa adanya
3. Replikasi seluruh sepuluh; perbaikan ditunda sebagai Future Enhancement
4. Bawa ke manajemen dulu karena menyentuh angka historis

**Jawaban:** Opsi 1, dan untuk butir 7: *"sudah benar"*

**Keputusan akhir:** Sembilan cacat **diperbaiki**; satu **direplikasi** karena perilakunya memang benar.

| # | Cacat | Bukti | Keputusan |
|---|---|---|---|
| 1 | Toleransi spreading berupa pencocokan substring; `199.99` dan `1100.0` lolos | `Activity/InputRegister_act-Act.xml:13183` | **PERBAIKI** |
| 2 | Ambang Rp 50 juta memakai tiga operator berbeda di tiga rule | `SetEmailKomite:1445` · `SetEmailKomiteAdjuster:968` · `SetEmailKomiteSalvage:888` | **PERBAIKI** — satu operator |
| 3 | Penyesuaian 7 jam asimetris di dalam satu kondisi validasi | `InputRegister_act:5788` + `:4805` | **PERBAIKI** |
| 4 | Kurs mengabaikan tanggal kerugian | `GETCURRENCYSTANDARD.fnc:3` vs `:14` | **PERBAIKI** — lihat D-48 |
| 5 | Kurs tidak ditemukan → `RETURN 1` | `GETCURRENCYSTANDARD.fnc:22` | **PERBAIKI** — lihat D-48 |
| 6 | `LIMIT_TOP` diabaikan + tie-breaker acak | `EmailKomiteBerjenjangSimasnet_sql:98-99` | **PERBAIKI sebagian** — lihat D-47: model kumulatif dipertahankan, `LIMIT_TOP` jadi validasi master; tie-breaker acak hanya ada di 2 query Simasnet |
| **7** | **Tanggal PLA/DLA dari pengguna dibuang, diganti `SYSDATE`** | `INSERT_PLADLA.prc:67`, `:136`, `:177` | **REPLIKASI — perilaku sekarang sudah benar.** Tanggal PLA/DLA adalah tanggal sistem menerbitkan dokumen, bukan tanggal yang dipilih pengguna |
| 8 | `NilaiSalvage` hanya ditambahkan bila baris terakhir kebetulan bertipe salvage | `ValidasiSisaTSI:817` vs `:1489` | **PERBAIKI** — salvage selalu ditambahkan |
| 9 | `INSERT_SALVAGE` menulis `IDSALVAGE = NULL` saat update | `INSERT_SALVAGE.prc:47` | **PERBAIKI** + perlu hitungan DBA atas baris rusak |
| 10 | `GETSELISIHJAM` gagal → `RETURN 0` jam, tak terbedakan dari nol | `GETSELISIHJAM.fnc:22` | **PERBAIKI** |

**Konsekuensi yang mengikat gerbang 1:** daftar "perbaikan yang diputuskan eksplisit" (`P-5`) bertambah dari **4 menjadi 13** butir. Setiap selisih yang muncul pada uji kesetaraan wajib dapat dipetakan ke salah satu dari 13 butir itu, atau dinyatakan sebagai bug.

Butir 7 menjadi contoh penting untuk tiket `B-9`: parameter `TTGLPLADLA` yang diterima lalu dibuang **bukan cacat** — sistem baru boleh menghapus parameter itu seluruhnya, dan AC-nya menyatakan tanggal dokumen diisi waktu sistem.

**Dampak ke Steering:** Migration Strategy `P-5` · Testing Strategy (gerbang 1) · `requirement-summary.md` §1 · BRD Bab 11 dan 21 · Risk Analysis (R-19)

---

### D-50 · Basis Perhitungan TAT — Dua Fungsi Dipertahankan untuk Keperluan Berbeda
**Status:** DECIDED

**Pertanyaan (Ronde 3 Q15):** `Database/GETSELISIHJAM.fnc` menghitung selisih dengan memotong akhir pekan saja — **nol hari libur, nol jam kerja** — sementara `DATAMINING.GET_WORKING_HOURS@ASMD` adalah objek remote lewat DB Link. Mana basis TAT yang benar?

**Pilihan yang ditawarkan:**
1. `GET_WORKING_HOURS@ASMD` menjadi satu-satunya basis; `GETSELISIHJAM` dipensiunkan
2. `GETSELISIHJAM` menjadi satu-satunya basis
3. Keduanya dipertahankan untuk keperluan berbeda
4. Belum tahu — perlu dicek ke pemilik KPI

**Jawaban:** Opsi 3, disertai permintaan daftar pemakaian kedua fungsi untuk cross-check

**Daftar pemakaian yang diserahkan** (pencarian case-insensitive; pemakaiannya huruf kecil):

| Fungsi | Pemakaian | Berkas |
|---|---|---|
| **`GET_WORKING_HOURS`** | **18** di **7 berkas** | `RDB List/GetDataKPIAdmin-SQL.xml` (4) · `GetDataKPIAdminPA-SQL.xml` (4) · `GetDataKPIAdminPA_khususPA-SQL.xml` (4) · `BrowseDataKPIAdmin_PA-SQL.xml` (2) · `ExportDataKomitesKlaimNONMBU-SQL.xml` (2) · `BrowseDataKPIAdmin-SQL.xml` (1) · **`Database/INSERT_PNCCHRONOLOGYTAT.prc` (1)** |
| **`GETSELISIHJAM`** | **1 jalur** | `RDB List/GetSelisihJam_sql-SQL.xml` — hasilnya **dibagi 24**, jadi satuannya **hari**; satu-satunya pemanggil `Activity/GetInboxRegisterCompliance-Act.xml` |
| **`HRD_LBR`** | **1** | `RDB List/CheckHoliday_SQL-SQL.xml` |

**Keputusan akhir:** Kedua fungsi dipertahankan dengan batas pemakaian yang sudah terbukti:

- **`GET_WORKING_HOURS`** → seluruh **KPI dan laporan admin**, ditambah **penulisan kronologi TAT**. Ini jalur yang angkanya dilaporkan ke manajemen.
- **`GETSELISIHJAM`** → **satu layar saja**, inbox Compliance, dengan keluaran **hari** (dibagi 24), bukan jam kerja.
- **`HRD_LBR`** → pemeriksaan hari libur, dipakai terpisah.

**Temuan yang mendukung keputusan ini:** framing awal agen ("dua basis yang tidak akan pernah menghasilkan angka sama") benar secara aritmetika tetapi **menyesatkan** — keduanya tidak pernah dipakai mengukur hal yang sama. Batas pemakaiannya tidak tumpang tindih.

**Konsekuensi:** `GET_WORKING_HOURS` dan `HRD_LBR` adalah objek **remote lewat DB Link**, sehingga keduanya masuk lingkup **D-25/R-03** dan logikanya harus **ditulis ulang di Go**, bukan sekadar dipanggil lewat API — karena perhitungan jam kerja dan kalender libur adalah aturan bisnis, bukan pengambilan data.

**Tetap diperbaiki apa pun pilihannya:** `GETSELISIHJAM.fnc:22` mengembalikan `0` saat error, tak terbedakan dari nol hari — butir 10 pada D-49.

**Dampak ke Steering:** Module Breakdown S-7 dan B-8 · Integration Strategy (D-25) · Risk Analysis (R-03, R-19)

---

### D-51 · Pembulatan Uang dan Toleransi Total Spreading
**Status:** DECIDED

**Pertanyaan (Ronde 3 Q16):** Sistem lama **tidak punya kebijakan pembulatan di mana pun** — nol `ROUND`/`TRUNC`/`CEIL`/`FLOOR`/`setScale` pada nilai uang, baik di lapisan Pega (`InputRegister_act`, 1,1 MiB) maupun di 12 procedure Oracle yang dibaca. `Database/GET_INTERPOLASIPNC.fnc` melakukan dua pembagian (`:25`, `:37`) tanpa pembulatan, dan `ehasil` dideklarasi `number` polos tanpa presisi. Berapa desimal untuk nilai uang, berapa untuk share spreading, dan kapan dibulatkan?

**Pilihan yang ditawarkan — (a) nilai uang klaim:**
1. Bulat ke rupiah · 2. Dua desimal · 3. Potong ke bawah · 4. Simpan penuh, bulatkan hanya saat tampil

**Pilihan yang ditawarkan — (b) share spreading:**
1. Dua desimal, toleransi `>= 99,99 && <= 100,01` · 2. Empat desimal · 3. Bilangan bulat persen

**Pilihan yang ditawarkan — (c) kapan dibulatkan:**
1. Setiap langkah · 2. Hanya pada nilai akhir sebelum disimpan · 3. Hanya saat ditampilkan

**Jawaban:**
> *"Kalau yang dimaksud untuk hal yang sama terkait total spreading 100% maka (a) Opsi 4 (simpan penuh, bulatkan hanya saat tampil) dengan validasi SUM(share) harus ada toleransi 99,9999 atau 100,0001 dengan pembulatan 4 desimal"*

**Keputusan akhir:**

**Nilai uang** disimpan **presisi penuh**; pembulatan dilakukan **hanya saat ditampilkan**, tidak saat menyimpan dan tidak per langkah perhitungan. Perbandingan terhadap ambang memakai nilai presisi penuh.

**Total spreading reasuransi** divalidasi dengan **pembulatan 4 desimal** dan **toleransi `99,9999` sampai `100,0001`**:

```
ROUND(SUM(share), 4) BETWEEN 99.9999 AND 100.0001
```

Ini menggantikan pencocokan substring `@contains(local.totalspreading, 99.99)` di
`Activity/InputRegister_act-Act.xml:13183` — butir 1 pada D-49.

**Acceptance criteria yang dapat langsung dipakai untuk `B-4`:**

| Total share | Hasil | Alasan |
|---|---|---|
| `100.0000` | **diterima** | tepat |
| `99.9999` | **diterima** | batas bawah toleransi |
| `100.0001` | **diterima** | batas atas toleransi |
| `99.9998` | **ditolak** | di luar toleransi |
| `100.0002` | **ditolak** | di luar toleransi |
| `99.99` | **ditolak** | perilaku lama menerimanya — **selisih yang direncanakan**, butir 1 D-49 |
| `199.99` | **ditolak** | perilaku lama menerimanya — selisih yang direncanakan |
| `1100.0` | **ditolak** | perilaku lama menerimanya — selisih yang direncanakan |

Tiga baris terakhir adalah **selisih yang akan muncul pada gerbang 1** dan sudah disetujui lebih dulu lewat D-49 butir 1.

**Konsekuensi untuk fee adjuster:** `GET_INTERPOLASIPNC` menghasilkan pecahan rupiah tak terbatas dari dua pembagian. Dengan keputusan ini, hasilnya **disimpan penuh** dan dibulatkan hanya saat ditampilkan — sehingga akumulasi lintas adjustment tidak kehilangan presisi.

**Terbuka:** apakah `LIMIT_BOTTOM` dan ambang uang lain di master bertipe numerik atau teks — DDL belum ada (R-08). Bila bertipe `VARCHAR2`, perbandingan menjadi leksikografis dan seluruh AC di atas tidak berlaku.

**Dampak ke Steering:** Domain Model · Database Strategy (tipe kolom nilai uang) · BRD Bab 11.3 · Testing Strategy (gerbang 1)

---

### D-52 · `TYPE_KOMITE` adalah Pemilih Pita Nilai — Melengkapi D-47
**Status:** DECIDED

**Pertanyaan (Ronde 3 Q17):** Setelah `Database/emailkomite.csv` dikoreksi kedua kalinya (2026-09-11 19:06, satu karakter: baris `ID 7` `TYPE_KOMITE` dari `2` menjadi `1`), gabungan kedua sub-tangga menjadi bersambung tanpa celah. Tetapi query `RDB List/EmailKomiteBerjenjang_sql-SQL.xml` memfilter `trim(TYPE_KOMITE) = trim({tempAdj.AcceptedNo})`, sehingga satu klaim hanya melihat satu sub-tangga. Akibatnya `TYPE_KOMITE=2` tidak lagi punya baris di bawah Rp 100.000.001. Bagaimana `TYPE_KOMITE` dipilih, dan apakah pemisahan ini disengaja?

**Pilihan yang ditawarkan:**
1. Gabungkan menjadi satu tangga — `TYPE_KOMITE` tidak lagi memfilter; kelima baris jadi satu deret kumulatif 1–5 jenjang
2. Pertahankan dua tangga, `TYPE_KOMITE` ditentukan aturan bisnis berbasis data (lini bisnis, cabang, jenis klaim)
3. Pertahankan dua tangga dengan pemilihan berbasis peran pengganti ketiga nama orang
4. `TYPE_KOMITE=2` memang seharusnya tidak menangani klaim di bawah Rp 100.000.001 — celah itu disengaja

**Jawaban:** Opsi 4 — *"karena untuk komite sampai 100 Jt pakai type_komite=1"*

**Keputusan akhir:** **`TYPE_KOMITE` adalah pemilih pita nilai klaim**, bukan penanda orang dan bukan penanda organisasi:

| Nilai klaim (setelah konversi IDR per D-48) | `TYPE_KOMITE` |
|---|---|
| Rp 0 sampai **Rp 100.000.000** | **`1`** |
| **di atas Rp 100.000.000** | **`2`** |

Celah pada `TYPE_KOMITE=2` di bawah Rp 100.000.001 **disengaja** dan bukan cacat — pita itu ditangani `TYPE_KOMITE=1`.

**Konsekuensi yang menutup satu butir `D-15`:** karena `TYPE_KOMITE` kini diturunkan dari **nilai klaim**, syarat bernama orang di `Activity/SetEmailKomite-Act.xml` step 10, 12, dan 14 (`ELLENSUPRIYATI` → `1`, `INDRAGUNAWAN` → `2`, `YOHANESRAYMONDADIKARTA` → `NONMBUAB`/`0`) **tidak dibawa ke sistem baru**. Begitu pula pola "nilai pembanding sengaja dinaikkan melewati ambang" yang menyertainya (`20-DETAIL-KOMITE-DBLINK.md:110-120`). Keduanya digantikan satu aturan berbasis nilai.

Ini menjadikan **tiga dari 24 user ID hardcode** hilang dari jalur komite — bagian dari `FR-F4`.

**Acceptance criteria final untuk B-7 — jumlah jenjang persetujuan NONMBU**

Model kumulatif per D-47: jumlah jenjang = jumlah baris `EMAILKOMITE` yang memenuhi
`STS_ADJ='1' AND STS_AKTIF='1' AND TYPE_BUSINESS='NONMBU' AND TYPE_KOMITE=<pita> AND LIMIT_BOTTOM <= nilai`, diurutkan `DEGREE`.

| Nilai klaim | `TYPE_KOMITE` | Baris yang cocok | **Jenjang** |
|---|---|---|---|
| Rp 50.000.000 | 1 | ID 7 | **1** |
| Rp 50.000.001 | 1 | ID 7, ID 1 | **2** |
| Rp 100.000.000 | 1 | ID 7, ID 1 | **2** |
| Rp 100.000.001 | 2 | ID 2 | **1** |
| Rp 500.000.001 | 2 | ID 2, ID 3 | **2** |
| Rp 1.000.000.001 | 2 | ID 2, ID 3, ID 4 | **3** |

**Diskontinuitas yang diterima secara sadar:** menyeberangi Rp 100.000.000 **menurunkan** jumlah approver dari 2 menjadi 1, karena komite berganti badan — dari komite pita rendah (2 anggota) ke komite pita tinggi (3 anggota) yang mulai dari jenjang pertamanya sendiri. Ini konsekuensi langsung dari memperlakukan keduanya sebagai **dua komite terpisah**, bukan satu tangga bersambung. Dicatat agar tidak dibaca sebagai cacat saat uji kesetaraan.

**Catatan penomoran `DEGREE`:** nilai `DEGREE` di master (1,1,2,3,4) **tidak sama dengan** jumlah jenjang. `DEGREE` hanya menentukan urutan (`ORDER BY DEGREE`); jumlah approver ditentukan `pxResultCount`. Pernyataan "berjenjang sampai 4 level" di D-14 dan `BRD §11.4` merujuk nilai `DEGREE`, bukan jumlah approver — jumlah approver maksimum yang benar-benar dapat terjadi adalah **2** (pita rendah) dan **3** (pita tinggi).

**Masih terbuka dari D-47:** di atas **Rp 200.000.000** tidak ada baris untuk **PA** dan **TRAVEL** (maksimum `LIMIT_TOP` keduanya Rp 200.000.000). Klaim PA atau Travel di atas nilai itu tetap mendapat 4 dan 3 jenjang karena model kumulatif, tetapi tidak ada baris yang secara eksplisit mencakupnya — validasi `LIMIT_TOP` per D-47 Opsi 2 akan menandainya sebagai lubang master.

**Dampak ke Steering:** Domain Model (master Ambang Komite) · Module Breakdown B-7 · BRD Bab 11.4 · `20-DETAIL-KOMITE-DBLINK.md` §1.4 dan §1.5 (logika bernama orang dicabut) · `FR-F4` (3 user ID hardcode hilang)

---

### D-53 · Data dan Lingkungan Uji Kesetaraan
**Status:** DECIDED

**Pertanyaan (Ronde 4 Q18):** `BRD §21.1` gerbang 1 menuntut perbandingan atas "data historis yang sama". Data apa, di lingkungan mana, dijalankan siapa? Tanpa ini `S-8` tidak punya acceptance criteria, dan ia gelombang 1 yang memblokir cutover setiap modul.

**Pilihan yang ditawarkan:**
1. Salinan data produksi di staging; Pega staging dan Go staging; dijalankan tim pengembang
2. Produksi read-only untuk Pega + Go staging
3. Data uji buatan yang mencakup kasus tepi yang sudah teridentifikasi
4. Belum dapat diputuskan — perlu tim infra

**Jawaban:** Opsi 1

**Keputusan akhir:** Uji kesetaraan dijalankan atas **salinan data produksi di lingkungan staging**, dengan **Pega staging** dan **Go staging** sebagai kedua sisi pembanding, dan dijalankan oleh **tim pengembang**.

**Alasan yang mengikat:** aturan kerja project melarang akses produksi, dan `D-21` menetapkan satu database bersama selama masa paralel — menembak produksi membawa risiko menulis tanpa sengaja. Data uji buatan (opsi 3) tidak memadai: cacat yang ditemukan pada Fase 1 — toleransi spreading berupa pencocokan substring, kurs `RETURN 1`, `IDSALVAGE = NULL` — justru muncul dari data nyata yang tidak akan terpikir dibuat.

**Konsekuensi yang harus disiapkan:**
- Dibutuhkan **Pega staging yang dapat ditembak dari luar** — belum ada konfirmasi bahwa lingkungan itu tersedia. Ini dependensi ke Tim Pega dan Tim Infra.
- Salinan data produksi membawa **data nasabah nyata** ke staging. Perlakuan penyamaran, hak akses, dan retensi salinan itu masuk lingkup `D-40` dan Compliance.
- `R-12` (zona waktu bergeser saat migrasi data) berlaku pada proses penyalinan itu sendiri, bukan hanya pada migrasi akhir.

**Dampak ke Steering:** Testing Strategy · Deployment Strategy (lingkungan) · Risk Analysis (R-12) · Module Breakdown S-8

---

### D-54 · Kewenangan Menyetujui Selisih pada Gerbang 1
**Status:** DECIDED

**Pertanyaan (Ronde 4 Q19):** Daftar perbaikan eksplisit `P-5` kini berisi **13 butir** (D-49). Setiap selisih yang muncul pada uji kesetaraan wajib dipetakan ke salah satunya atau dinyatakan bug. Siapa yang memutuskan?

**Pilihan yang ditawarkan:**
1. Selisih yang cocok dengan 13 butir `P-5` lolos otomatis; selisih di luar itu wajib persetujuan Work Owner tertulis
2. Setiap selisih tanpa kecuali butuh persetujuan Work Owner
3. Lead engineer memutuskan; Work Owner diberi tahu
4. Belum ditetapkan

**Jawaban:** Opsi 1

**Keputusan akhir:** Selisih yang **cocok dengan salah satu dari 13 butir `P-5`** lolos otomatis dan cukup dicatat. Selisih **di luar 13 butir itu** wajib **persetujuan Work Owner secara tertulis** sebelum modul dinyatakan lulus gerbang 1.

**Alasan:** ketiga belas butir sudah diputuskan eksplisit pada D-49, sehingga meminta persetujuan ulang per modul menambah beban tanpa menambah kendali. Yang benar-benar menuntut perhatian adalah selisih **yang tidak terduga** — dan itulah yang dipagari.

**Konsekuensi untuk perkakas `S-8`:** ia wajib **mengklasifikasikan** setiap selisih, bukan hanya melaporkannya. Keluarannya minimal: selisih yang terpetakan ke butir `P-5` mana, dan selisih yang tidak terpetakan. Yang kedua menjadi antrean persetujuan Work Owner.

**Dampak ke Steering:** Testing Strategy · Migration Strategy `P-5` · Module Breakdown S-8

---

### D-55 · Pencabutan `BRD §21.4` untuk Empat Modul
**Status:** DECIDED — usulan revisi BRD, menunggu persetujuan formal

**Pertanyaan (Ronde 4 Q20):** `BRD §21.4` menyatakan `FR-B5`, `B7`, `B9`, `B10`, `B12` tidak dapat dinyatakan diterima sampai R-01 tertutup. Setelah 62 dari 64 source procedure datang, status kelimanya berubah dan penghalangnya kini berbeda per modul. Apakah §21.4 masih berlaku untuk kelimanya?

**Pilihan yang ditawarkan:**
1. Cabut §21.4 untuk `B-7`, `B-9`, `B-10`, `B-12`; pertahankan hanya untuk `B-5`
2. Pertahankan §21.4 apa adanya sampai seluruh artefak lengkap
3. Cabut seluruhnya — R-01 sudah tidak relevan
4. Bawa ke manajemen karena §21.4 sudah disetujui

**Jawaban:** Opsi 1

**Keputusan akhir:** `BRD §21.4` **dicabut untuk `B-7`, `B-9`, `B-10`, dan `B-12`**, dan **dipertahankan hanya untuk `B-5`**.

Status dan sisa penghalang per modul:

| Modul | Status | Sisa penghalang |
|---|---|---|
| **B-7** Komite | lepas dari §21.4 | hanya sisa: PA dan TRAVEL di atas Rp 200.000.000 tidak punya baris master |
| **B-9** PLA/DLA | lepas dari §21.4 | status `VALID`/`INVALID` `GET_GROUPBUSINESS_XOL` di produksi; keputusan menulis ulang procedure ber-9-commit |
| **B-10** Akseptasi | lepas dari §21.4 | 3 activity hilang: `InsertDataAkseptasiToLeader`, `InsertLogKasir_act`, `TransferCashierDataASM_act` |
| **B-12** Salvage | lepas dari §21.4 | `SET_ATTACHFILETEMPSALVAGE`; hitungan DBA atas baris ber-`IDSALVAGE` NULL |
| **B-5** Settlement | **tetap terikat §21.4** | isi `POOLDATA.GCNM_FEE_SCALE` (17 pita) · isi `m_currencystandard` · `BrowseT_Claim_Adjustment_SQL` |

**Alasan:** §21.4 ditulis ketika R-01 memblokir kelimanya secara seragam. Penghalangnya sekarang berbeda per modul dan sebagian sudah lepas; menahan empat modul atas dasar yang tidak lagi berlaku menunda pekerjaan yang sebenarnya dapat berjalan.

**Ini usulan revisi BRD, bukan perubahan yang dieksekusi agen.** `docs/BRD.md` tidak disentuh; revisinya masuk daftar usulan Fase 6.

**Dampak ke Steering:** BRD Bab 21.4 · Risk Analysis (R-01, R-14) · Module Breakdown

---

### D-56 · Ukuran Penerimaan untuk Modul Tanpa Baseline Pega
**Status:** DECIDED

**Pertanyaan (Ronde 4 Q21):** Dua modul **tidak punya pembanding Pega** sehingga gerbang 1 tidak dapat diberlakukan — `F-3` karena HCC/HCQ nol jejak di export (`docs/verifikasi-bukti-adr.md` §1.1), dan `S-5` karena sistem lama tidak punya jejak audit nilai sama sekali (§14.13). Apa penggantinya?

**Pilihan yang ditawarkan:**
1. Gerbang 1 diganti "lolos uji fungsional terhadap kontrak yang disepakati"
2. Kedua modul hanya melewati gerbang 2 (UAT), tanpa gerbang 1
3. Tunda keduanya sampai ada baseline
4. Belum ditetapkan

**Jawaban:** Opsi 1

**Keputusan akhir:** Untuk `F-3` dan `S-5`, gerbang 1 diganti **uji fungsional terhadap kontrak yang disepakati**:

| Modul | Kontrak yang menjadi acuan | Pemilik kontrak |
|---|---|---|
| **F-3** Identitas & Akses | kontrak API **HCC/HCQ** — field request/response login, kode error, timeout, endpoint refresh/validasi | pemilik API HCC/HCQ |
| **S-5** Jejak Audit | **daftar peristiwa wajib audit** beserta field yang harus tercatat | Compliance |

**Alasan menolak opsi 2:** menghapus gerbang 1 justru pada dua modul yang paling sensitif — `F-3` adalah otorisasi, `S-5` adalah jejak audit — menyisakan hanya UAT sebagai pemeriksaan, dan UAT tidak memeriksa hal yang tidak terlihat di layar.

**Ketergantungan baru yang diciptakan keputusan ini:** kedua kontrak **belum ada**. Sampai keduanya diterima:
- `F-3` tidak dapat lulus gerbang apa pun, dan tiketnya tetap `needs-info`.
- `S-5` dapat ditulis dan dikerjakan, tetapi tidak dapat dinyatakan lulus.

Keduanya masuk daftar tagihan pihak luar pada `D-36`, dan pemilik API HCC/HCQ adalah pihak yang **belum pernah dihubungi**.

**Dampak ke Steering:** Testing Strategy · BRD Bab 21.2 (pengecualian kriteria #3 untuk dua modul) · Risk Analysis (R-16)

---

### D-57 · Daftar Job Terjadwal Ditemukan — Menutup D-17 dan R-02
**Status:** DECIDED — menutup **D-17** yang berstatus `OPEN — gap discovery`

**Pertanyaan asal (D-17, Sesi 1):** Apa saja proses terjadwal / batch yang berjalan di sistem lama? Saat itu rule Agent dan Queue Processor tidak ada di export, sehingga daftar job tidak diketahui (**R-02**).

**Yang berubah:** pada 2026-09-09 export bertambah dan **tiga folder yang sebelumnya tidak ada** muncul — `Job Scheduler/` (5 berkas), `Agents/` (1), `Service REST/` (4).

**Keputusan akhir:** **D-17 ditutup dan R-02 tertutup pada tingkat artefak.** Daftar lengkap job terjadwal, terverifikasi dari `pyRuleName` dan `pyActivityName` masing-masing berkas:

| Job | Frekuensi | Jam mulai | Activity target | Target ada? |
|---|---|---|---|---|
| `JOBForKomiteKlaimPNC` | Daily, interval 1 | **06:00:00** | **`AutoAcceptKomite`** | ✅ |
| `JobHitungDeadlineToTemporaryCLose` | **Weekly** | **23:43:00** | `JobTemporaryCloseClaimPNC` | ✅ |
| `JobSendAutoLODKlaimPersonal` | Daily, interval 1 | **20:54:00** | `Act_SendAutoLODKlaimPersonal` | ✅ |
| `PNCMyReportKlaim3` | Daily, interval 1 | **08:00:00** | `ReportAI_Act` | ✅ |
| `ProcessClaimKredit` | Daily, interval 1 | **10:00:00** | `CreateClaimCredit_Table` | ✅ |

Seluruhnya `pyIsEnabled=true`, `pyApplicableTo=Cluster`, `pyNodeTypesText=BackgroundProcessing`.

**Agent:** `Agents/TATReportAgent-Agents.xml` — `pyEnable=true`, **`pyTriggerInterval=1800`** (30 menit), **`pyBypassActivityAuthentication=true`**. Merujuk `CreateCasePNCAgent_ActButton`, `AlertAgentKasirBlmTransferKlaim`, `TransferAllCaseNotAssigned` — **ketiganya ada**.

**Konsekuensi: `S-6` Penjadwalan naik dari TERHALANG menjadi SEBAGIAN.**

**Tiga hal yang tetap perlu jawaban Work Owner:**
1. **`JOBForKomiteKlaimPNC` menjalankan `AutoAcceptKomite` setiap hari jam 06:00** — job yang **menyetujui komite secara otomatis**. Perilaku otorisasi ini tidak tercatat di dokumen mana pun dan belum pernah dibahas. Benar demikian?
2. Tiga job ber-`pyDescription = "job jalan 2 menit"` sementara konfigurasinya `Daily`/`Weekly` — selisih 720× sampai 5.040×. Mana yang benar?
3. `pyBypassActivityAuthentication=true` pada agent — diterima untuk sistem baru?

**Penemuan lain dari folder baru:** `Service REST/` memuat **empat layanan masuk** — `KomiteAcceptAdjustment`, `KomiteAcceptAdjustmentPA`, `RecivedDataandAttachmentLelangASMSimasbid`, `RequestCreateClaimCredit2`. Dua di antaranya **menerima persetujuan komite dari sistem lain**. Seluruh analisis integrasi sebelumnya hanya melihat Connect REST **keluar**; permukaan masuk ini **menambah lingkup `S-4`** dan belum pernah masuk hitungan `FR-S4` maupun `D-25`.

**Dampak ke Steering:** Risk Analysis (R-02 ditutup) · Module Breakdown S-6 dan S-4 · `19-GAP-EXPORT-DETAIL.md` · BRD Bab 15 dan 21.3 kriteria #3

---

### D-58 · Peran Bisnis = 22 Access Group, Dinamai Ulang
**Status:** DECIDED

**Pertanyaan (Ronde 5 Q22):** Tiket wajib mengisi field `Peran penguji gerbang 2`, dan `F-3` harus membangun tabel peran dari nol. Yang tersedia hanya 22 nama access group Pega, dan `I-03` sudah menetapkan BRD memakai jabatan generik — bukan nama peran operasional. Berapa peran bisnis yang sebenarnya?

**Pilihan yang ditawarkan:**
1. Peran bisnis = 22 access group apa adanya, hanya dinamai ulang agar terbaca manusia
2. Lebih sedikit dari 22 — beberapa access group adalah varian teknis dari peran yang sama
3. Lebih banyak dari 22 — ada peran bisnis yang berbagi satu access group
4. Work owner menyebutkan daftarnya sendiri

**Jawaban:** Opsi 1

**Keputusan akhir:** **Peran bisnis di sistem baru berjumlah 22, satu-untuk-satu dengan access group Pega**, hanya dinamai ulang agar terbaca manusia. Tidak ada penggabungan dan tidak ada pemecahan.

Daftar 22 access group, seluruhnya terverifikasi ada di export sebagai literal `GCNMFW:<nama>`:

`Administrators` · `CaseManager` · `PncAdmin` · `PncManagerAdmin` · `PncPICTeknik` · `PNCKomiteTeknik` · `PNCKomite` · `PncRCLPUCL` · `PncAnalystDoctor` · `PncComplience` · `PncInvestigator` · `PNCSurveyor` · `PncPLADLA` · `PncReceive` · `PncManagerReceive` · `PncCollection` · `PncOPCGeneral` · `PNCServiceCenter` · `TreatyIn` · `ViewClaimPNC` · `PNCReportClaimInternal` · `PNCReportClaimEksternal`

**Konsekuensi yang harus ditangani di `F-3`:**
- **Tiga nama muncul dalam dua kapitalisasi** — `ViewClaimPNC`/`VIEWCLAIMPNC` dan `PncReceive`/`PNCRECEIVE`. Perbandingan access group di rule lama tidak konsisten soal huruf besar-kecil. Sistem baru wajib menormalkannya menjadi satu identitas per peran.
- Pemetaan **peran → 51 item menu** hidup hanya di dalam **34 When rule + `Navigation/pyCaseWorkerNavigation-Navigation.xml`**, bukan sebagai data. Sumber untuk mengisi tabel izin adalah membaca ke-34 When rule itu satu per satu — **5 di antaranya hilang dari export** (`IsGCNMReport`, `IsKomite`, `IsNotViewClaim`, `IsPNCBonding`, `IsSurvey`).
- **Penugasan operator ke peran tidak ada di database.** `POOLDATA.T_ACCESS_GROUP_PNC` hanya memetakan `OPERATOR_ID` → `OLD_OPERATOR_ID`. Tanpa artefak ini, `F-3` dapat membangun tabelnya tetapi tidak dapat mengisinya.

**Dampak ke Steering:** Domain Model (master Peran & Izin Menu) · Module Breakdown F-3 dan U-6 · Security

---

### D-59 · Satuan Izin adalah Menu, Bukan Tindakan — Tanpa Pemisahan Tugas
**Status:** DECIDED oleh work owner — dicatat dengan konsekuensi terbuka

**Pertanyaan (Ronde 5 Q23):** Siapa yang boleh **membatalkan klaim**, **mengubah nilai estimasi/akseptasi setelah komite menyetujui**, dan **menyetujui komite**? Export membuktikan otorisasi sistem lama adalah penyembunyian menu semata: `pyPrivilegeName` terisi hanya pada **1 dari 902** activity, dan yang satu itu privilege bawaan Pega untuk ekspor ruleset, bukan aturan bisnis (`docs/verifikasi-bukti-adr.md` §10.3).

**Pilihan yang ditawarkan:**
1. Ada pemisahan tugas nyata — pelaksana tidak boleh menyetujui; peran disebutkan per tindakan
2. Tidak ada pemisahan formal; kontrolnya prosedural
3. Sama dengan sistem lama — siapa pun yang punya akses menu bisa melakukannya
4. Belum pernah ditetapkan

**Jawaban:** Opsi 3

**Keputusan akhir:** **Satuan izin adalah menu, bukan tindakan individual.** Seorang pengguna yang memiliki akses ke sebuah menu berwenang atas seluruh tindakan yang dijangkau menu itu — termasuk membatalkan klaim, mengubah nilai setelah persetujuan komite, dan menyetujui komite. **Tidak ada pemisahan tugas formal.**

**Ini tetap memenuhi `BRD §21.2` kriteria #8 dan `FR-R1`,** dengan bacaan berikut yang disampaikan ke work owner dan tidak dikoreksi: yang diperiksa di setiap endpoint adalah **"apakah peran pemanggil memiliki menu yang memberi akses ke endpoint ini"**. Perbedaan dengan sistem lama bukan pada satuan izinnya, melainkan pada **tempat penegakannya** — dulu hanya disembunyikan di antarmuka, sekarang ditegakkan di server pada setiap endpoint.

**Konsekuensi yang diterima secara sadar:**

1. **Orang yang sama dapat membuat, menyetujui, dan membayarkan satu klaim** bila perannya memiliki ketiga menu itu. Tidak ada kontrol teknis yang mencegahnya.
2. **Jejak audit menjadi satu-satunya kontrol pengimbang.** Karena tidak ada pencegahan, yang tersisa hanyalah pencatatan. Ini **menaikkan** kepentingan `S-5` dari modul pendukung menjadi kontrol utama — dan `S-5` adalah kemampuan baru 100% tanpa baseline (D-56).
3. **`AutoAcceptKomite` melewati kontrol apa pun.** Job terjadwal harian jam 06:00 (D-57) menyetujui komite tanpa pengguna sama sekali, sehingga bahkan kontrol berbasis menu tidak berlaku padanya.
4. Bila kemudian ada temuan audit atau pentest yang menuntut pemisahan tugas, perubahannya menyentuh model izin di `F-3` — bukan penyesuaian kecil.

**Yang tetap berlaku:** `BRD §21.2` kriteria #9 (jejak audit untuk setiap perubahan bernilai bisnis) menjadi **wajib tanpa pengecualian** pada seluruh tiket modul bisnis, karena ia satu-satunya kontrol yang tersisa.

**Dampak ke Steering:** Security (Authorization) · Module Breakdown F-3 dan S-5 · BRD Bab 17 dan 21.2 · Risk Analysis

---

### D-60 · Peran Penguji Gerbang 2 per Modul
**Status:** DECIDED

**Pertanyaan (Ronde 5 Q24):** Setiap tiket wajib mengisi field `Peran penguji gerbang 2`. Untuk modul bisnis, peran penguji dapat diturunkan dari inbox yang dipakai. Untuk **modul fondasi** tidak bisa — tidak ada user bisnis yang membuka layar "Akses Data".

**Pilihan yang ditawarkan:**
1. Modul bisnis → peran pemakai inbox terkait; modul fondasi → tidak melewati gerbang 2, cukup gerbang 1 ditambah persetujuan Work Owner
2. Modul fondasi tetap melewati gerbang 2 dengan penguji yang ditunjuk Work Owner
3. Seluruh modul diuji peran yang sama — satu tim UAT terpusat
4. Belum ditetapkan

**Jawaban:** Opsi 1

**Keputusan akhir:**

| Kelompok modul | Gerbang 2 | Penguji |
|---|---|---|
| **Modul bisnis** (`B-1`…`B-14`, `S-1`…`S-4`, `S-6`, `S-7`, `U-1`, `U-3`…`U-6`) | **berlaku** | peran bisnis pemakai inbox/layar modul itu, diturunkan dari `Navigation/pyCaseWorkerNavigation-Navigation.xml` dan When rule peran |
| **Modul fondasi** (`F-1`, `F-2`, `F-3`, `F-4`, `F-5`, `S-5`, `S-8`, `U-2`) | **tidak berlaku** | gerbang 1 ditambah **persetujuan Work Owner** |

**Alasan:** memaksakan gerbang 2 pada modul yang tidak punya peran pemakai akan membuat tiket fondasi macet di gerbang yang tidak dapat dilewati siapa pun.

**Catatan penting terhadap `D-46`:** keputusan ini mengatur **tahap implementasi**, bukan tahap penulisan tiket. Pada tahap penulisan, `D-46` menetapkan agen yang me-review dan menyatakan tiket selesai. Field `Peran penguji gerbang 2` pada setiap tiket karena itu diisi sebagai **rencana untuk tahap implementasi**, bukan pernyataan bahwa gerbang itu sudah dilewati.

**Pengecualian yang mengikuti dari D-56:** `F-3` dan `S-5` tidak punya baseline Pega, sehingga gerbang 1 mereka diganti uji fungsional terhadap kontrak. Untuk kedua modul itu, keseluruhan kelulusan bertumpu pada kontrak yang **belum ada**.

**Terbuka — belum dijawab:** `C-9` dan `R-15` mencatat bahwa **waktu user bisnis untuk UAT belum dialokasikan resmi**. Pertanyaan itu diajukan bersama Q24 dan belum terjawab. Tanpa alokasi resmi, gerbang 2 bergantung pada ketersediaan user yang sedang menjalankan operasional harian — dan itu risiko jadwal yang sudah tercatat tetapi belum ditangani.

**Dampak ke Steering:** Testing Strategy · BRD Bab 19 (C-9) dan 21 · Risk Analysis (R-15)

---

### D-61 · Jadwal Tetap Seluruh Modul — Selisih Menjadi Tanggung Jawab Manajemen
**Status:** DECIDED oleh work owner — dicatat dengan risiko terbuka

**Pertanyaan (Ronde 6 Q25):** `D-30` menetapkan seluruh modul selesai akhir September 2026. Hari ini **14 September 2026**, tersisa **16 hari kalender**, dan pemeriksaan `find` untuk `*.go`, `go.mod`, `package.json` menghasilkan **nihil** — belum ada satu baris kode implementasi. Apa yang sebenarnya harus jadi akhir September?

**Yang berubah sejak `D-30` ditetapkan dan belum pernah dibawa kembali:** jumlah modul menjadi **33** (bukan 27, per `D-35` dan `D-42`) · **±397 artefak** masih diminta ke Tim Pega (`R-16`) · tiga pihak luar belum menjawab (`D-36`) · dan `D-44` menetapkan jumlah tim baru ditentukan **setelah** tiket direview.

**Pilihan yang ditawarkan:**
1. Target akhir September berlaku untuk artefak perencanaan — ADR, tiket, Steering yang direvisi — bukan untuk kode
2. Target tetap seluruh modul; tiket ditulis apa adanya dan selisihnya menjadi tanggung jawab manajemen
3. Scope dipangkas — work owner menyebutkan modul mana yang wajib akhir September
4. Dibawa ke Sponsor Project dengan papan tiket sebagai bahan

**Jawaban:** Opsi 2

**Keputusan akhir:** **Target `D-30` tidak berubah** — seluruh modul akhir September 2026. Tiket ditulis **apa adanya**, mencerminkan cakupan dan kendala yang sebenarnya, dan **selisih antara cakupan itu dengan tenggat menjadi tanggung jawab manajemen**.

**Konsekuensi yang mengikat bentuk artefak:**
- Tiket **tidak ditulis** dengan estimasi atau urutan yang berpura-pura tenggat ini terpenuhi. Cakupan ditulis sesuai bukti.
- `docs/ticketing/README.md` **wajib memuat satu bagian jujur tentang jadwal**: jumlah tiket, jumlah modul yang belum tersentuh, dan sisa waktu ke target, berdampingan. Angka tidak disajikan tanpa konteks.
- Papan tiket menjadi bahan keputusan manajemen — konsisten dengan `D-44` yang menjadikan hasil review tiket sebagai dasar penentuan jumlah tim.

**Risiko yang tetap terbuka dan tidak berubah oleh keputusan ini:** `R-05` (jadwal tidak sepadan dengan ukuran pekerjaan, diterima manajemen) dan `R-13` (tenggat tidak punya pemaksa eksternal — `I-01` menetapkan tidak ada lisensi berakhir, tidak ada penalti kontraktual, sehingga penjadwalan ulang sepenuhnya di dalam kendali manajemen Sinarmas).

**Dampak ke Steering:** Migration Strategy · Risk Analysis (R-05, R-13) · `docs/ticketing/README.md`

---

### D-62 · Retensi Jejak Audit Mengikuti Retensi Data Klaim
**Status:** DECIDED — angka masih perlu diambil; menutup arah `D-28`

**Pertanyaan (Ronde 6 Q26):** `D-28` menetapkan jejak audit append-only tetapi **lama retensinya belum ditentukan**. `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang karena tidak ada pemisahan tugas. Dan `BRD §21.3` kriteria #8 menjadikan retensi sesuai ketentuan Compliance sebagai syarat mematikan Pega.

**Pilihan yang ditawarkan:**
1. Ikuti ketentuan OJK untuk dokumen klaim asuransi
2. Sama dengan retensi data klaim yang berlaku sekarang
3. Belum tahu — harus ke Compliance, tiket `S-5` ditandai `needs-info`
4. Angka tertentu yang disebutkan work owner

**Jawaban:** Opsi 2

**Keputusan akhir:** **Retensi jejak audit mengikuti retensi data klaim yang berlaku sekarang.** Tidak ada kebijakan retensi terpisah untuk data audit — satu kebijakan berlaku untuk keduanya.

**Yang ini selesaikan:** `S-5` tidak perlu menunggu keputusan kebijakan baru dari Compliance. Kebijakannya sudah ada; yang dibutuhkan hanyalah **angkanya**.

**Yang masih perlu diambil:** angka retensi data klaim yang berlaku sekarang belum ada di repo maupun di dokumen proyek mana pun. Ia diambil dari kebijakan yang sudah berjalan — bukan diputuskan ulang. Sampai angkanya masuk, `S-5` dibangun dengan **retensi sebagai parameter konfigurasi** (konsisten `D-15`: tidak ada nilai bisnis yang di-hardcode), sehingga modulnya tidak terhalang.

**Dampak ke Steering:** Security · NFR (auditabilitas) · Database Strategy · BRD Bab 21.3 kriteria #8 · Module Breakdown S-5

---

### D-63 · Kewenangan Perubahan Skema Selama Masa Paralel
**Status:** DECIDED

**Pertanyaan (Ronde 6 Q27):** `D-21` menetapkan satu database bersama dan `P-1` satu penulis per tabel; `P-4` mewajibkan migrasi skema backward-compatible. Tetapi **siapa yang boleh menjalankan perubahan skema** belum ditetapkan — padahal `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` dibaca **116 rule** Pega yang sedang melayani produksi.

**Pilihan yang ditawarkan:**
1. DBA menjalankan, atas permintaan tertulis tim pengembang, dengan persetujuan work owner — dan setiap perubahan diuji menjalankan Pega dan Go bersamaan lebih dulu
2. Tim pengembang punya akses langsung ke skema aplikasi sendiri; tabel bersama lewat DBA
3. Hanya DBA, tanpa keterlibatan tim pengembang
4. Belum ditetapkan

**Jawaban:** Opsi 1

**Keputusan akhir:** Perubahan skema dijalankan **DBA**, atas **permintaan tertulis** tim pengembang, dengan **persetujuan Work Owner**. Setiap perubahan **wajib diuji lebih dulu dengan menjalankan Pega dan Go bersamaan** terhadap skema hasil perubahan.

**Alasan:** `P-4` mewajibkan verifikasi backward-compatible dengan menjalankan versi lama dan baru bersamaan — itu tidak dapat dilakukan DBA sendirian maupun tim pengembang sendirian. Dan karena ada tabel yang dibaca 116 rule Pega, satu `ALTER` yang keliru menghentikan sistem yang sedang melayani produksi.

**Konsekuensi untuk acceptance criteria:** `BRD §21.2` kriteria #13 (migrasi skema backward-compatible, diverifikasi dengan menjalankan versi lama dan baru bersamaan) menjadi **dapat diuji**, karena prosedurnya kini punya pemilik dan langkah. Setiap tiket yang menyentuh skema wajib memuat bagian rollback yang tidak kosong, sesuai `P-4`.

**Konsekuensi operasional:** setiap perubahan skema menempuh tiga pihak. Ini memperlambat iterasi, dan konsekuensi itu diterima demi melindungi produksi yang masih dilayani Pega.

**Dampak ke Steering:** Database Strategy (migrasi skema) · Deployment Strategy · Migration Strategy `P-4` · BRD Bab 21.2 kriteria #13

---

### D-64 · Data Nasabah Disalin Apa Adanya ke Staging
**Status:** DECIDED oleh work owner — dicatat dengan konsekuensi terbuka

**Pertanyaan (Ronde 6 Q28a):** `D-53` menetapkan uji kesetaraan dijalankan atas salinan data produksi di staging. Apakah data nasabah disamarkan sebelum disalin?

**Pilihan yang ditawarkan:**
1. Disamarkan sebelum disalin — nomor polis, nama tertanggung, NPWP, rekening
2. Disalin apa adanya, dengan hak akses staging diperketat
3. Perlu ke Compliance dulu

**Jawaban:** Opsi 2

**Keputusan akhir:** Data produksi disalin ke staging **apa adanya, tanpa penyamaran**, dengan **hak akses lingkungan staging diperketat** sebagai kontrol penggantinya.

**Konsekuensi yang diterima secara sadar:**
1. **Staging menjadi lingkungan yang memuat data nasabah nyata** — nomor polis, nama tertanggung, NPWP, nomor rekening, data medis pada lini PA dan Travel. Perlindungannya sepenuhnya bergantung pada hak akses, bukan pada sifat datanya.
2. Klasifikasi staging **naik setara produksi** untuk keperluan keamanan: siapa yang punya akses, berapa lama salinan disimpan, dan kapan dimusnahkan menjadi pertanyaan yang harus dijawab — dan **belum dijawab**.
3. Akses medis yang `FR-R2` batasi pada peran Analyst Doctor dan RCL Dokter berlaku juga di staging, bukan hanya produksi.
4. Keputusan ini **berdampingan dengan `D-40`** yang masih terbuka: export rule sudah memuat kredensial plaintext, dan kini salinan data nasabah menyusul ke lingkungan yang sama-sama belum ditetapkan pengamanannya.

**Yang tidak berubah:** aturan kerja project tetap melarang **menampilkan atau menyalin data nasabah ke dokumen yang akan di-commit**. Keputusan ini mengatur isi lingkungan staging, bukan isi dokumen. ADR dan tiket tetap merujuk dengan nama kolom dan `berkas:baris`, tanpa nilai.

**Terbuka:** siapa yang menyetujui akses staging, berapa lama salinan disimpan, dan prosedur pemusnahannya. Ditujukan ke pihak yang sama dengan `D-40`.

**Dampak ke Steering:** Security (data sensitif) · Testing Strategy · Deployment Strategy (lingkungan) · Risk Analysis (R-17)

---

### D-65 · Kebijakan Penghapusan Data Mengikuti Sistem Lama — dengan Pengecualian Jejak Audit
**Status:** DECIDED oleh work owner — **dengan satu pertentangan terhadap `D-28` yang perlu diselesaikan**

**Pertanyaan (Ronde 6 Q28b):** Bagaimana penghapusan data di sistem baru? Sistem lama melakukan `DELETE` dan `UPDATE` nyata pada tabel log.

**Pilihan yang ditawarkan:**
1. Soft delete di mana pun — tidak ada `DELETE` fisik pada data bernilai bisnis
2. Hard delete untuk data tertentu — work owner menyebutkan mana
3. Sama dengan sistem lama

**Jawaban:** Opsi 3

**Keputusan akhir untuk data bisnis:** kebijakan penghapusan **mengikuti sistem lama**. Pola yang terbukti di export dan menjadi acuan: `RDB-DELETE` pada 12 step di 7 activity, `OBJ-DELETE` pada 2 step di 2 activity, dan `DELETE` pada rule SQL — termasuk `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503` yang menghapus lalu menyisipkan ulang seluruh pohon klaim sebagai pola idempoten.

**Pertentangan yang harus diselesaikan.** `D-28` (Sesi 2) menetapkan:

> *"Data audit bersifat append-only: tidak boleh diubah atau dihapus oleh jalur aplikasi mana pun."*

Sementara sistem lama **melanggar itu pada tabel lognya sendiri**, terbukti:

| Operasi | Tabel | `berkas:baris` |
|---|---|---|
| **`UPDATE`** | `pooldata.claim_service_log` — `set jsonout = … where id = …` | `RDB List/UpdateLogServiceClaim-SQL.xml:27` |
| **`DELETE`** | `POOLDATA.JSON_KLAIM_LOG` — `delete … where a.idpega=… and a.tgl_input=…` | `RDB List/InsertClaimPNC-SQL.xml:77` |
| `UPDATE` | `T_LOGINCOAS` | `RDB List/BrowseOldEmailCoas-SQL.xml:69` |
| `UPDATE` | `pooldata.mst_login_surveyor` | `RDB List/UpdateMasterLoginSurvey-SQL.xml:9` |

Menerapkan "sama dengan sistem lama" secara harfiah pada tabel-tabel itu akan **mereplikasi pelanggaran `D-28`** — dan `D-59` baru saja menjadikan jejak audit **satu-satunya kontrol pengimbang** karena tidak ada pemisahan tugas.

**Resolusi yang diusulkan, menunggu persetujuan work owner:**

| Kelas data | Kebijakan | Dasar |
|---|---|---|
| **Data bisnis** (klaim, objek, coverage, estimasi, adjustment, master) | **mengikuti sistem lama** — hard delete di tempat sistem lama melakukannya | D-65 |
| **Jejak audit `S-5`** (modul baru) | **append-only mutlak** — tidak ada `UPDATE`, tidak ada `DELETE` | D-28, dikuatkan D-59 |
| **Tabel log warisan** (`claim_service_log`, `json_klaim_log`, `T_LOGINCOAS`, `mst_login_surveyor`) | **bukan jejak audit** — diperlakukan sebagai data bisnis, boleh mengikuti sistem lama | konsekuensi pemisahan di atas |

Dengan pemisahan ini, `D-65` dan `D-28` tidak bertentangan: yang "mengikuti sistem lama" adalah data bisnis dan tabel log warisan; yang append-only adalah jejak audit baru yang dibangun `S-5`.

**Bila resolusi ini tidak disetujui**, salah satu dari dua hal harus dipilih secara eksplisit: `D-28` dicabut untuk tabel log warisan, atau `D-65` dikecualikan untuk tabel log warisan. Tiket `S-5` dan `B-1` tidak dapat ditulis sebelum ini jelas.

**Dampak ke Steering:** Database Strategy · Security · Domain Model · Module Breakdown S-5 dan B-1 · BRD Bab 17.5 dan 21.2 kriteria #9

---

### D-66 · Soft Delete Menyeluruh — Menyupersede D-65
**Status:** DECIDED — **menyupersede `D-65`**

**Latar:** `D-65` mencatat jawaban Ronde 6 Q28(b) Opsi 3 ("sama dengan sistem lama") dan mengangkat bahwa jawaban itu bertentangan dengan `D-28` (jejak audit append-only), lalu mengusulkan pemisahan tiga kelas data sebagai resolusi. Work owner **merevisi jawabannya menjadi Opsi 1** sebelum resolusi itu disetujui.

**Pertanyaan (Ronde 6 Q28b):** Bagaimana penghapusan data di sistem baru?

**Pilihan yang ditawarkan:**
1. Soft delete di mana pun — tidak ada `DELETE` fisik pada data bernilai bisnis
2. Hard delete untuk data tertentu — work owner menyebutkan mana
3. Sama dengan sistem lama

**Jawaban pertama:** Opsi 3
**Jawaban revisi:** Opsi 1

**Keputusan akhir:** **Soft delete berlaku menyeluruh.** Tidak ada `DELETE` fisik pada data bernilai bisnis di sistem baru. Penghapusan dinyatakan lewat penanda (mis. kolom flag beserta waktu dan pelaku), bukan lewat pembuangan baris.

**Pertentangan dengan `D-28` hilang.** Karena tidak ada penghapusan fisik pada data bernilai bisnis, jejak audit append-only (`D-28`) dan kebijakan penghapusan (`D-66`) berdiri di atas prinsip yang sama. Pemisahan tiga kelas data yang diusulkan `D-65` **tidak diperlukan dan tidak dipakai**.

**Yang ini perbaiki dari sistem lama.** Sistem lama melakukan penghapusan fisik di banyak tempat, termasuk pada tabel lognya sendiri:

| Operasi | Objek | `berkas:baris` |
|---|---|---|
| `UPDATE` | `pooldata.claim_service_log` | `RDB List/UpdateLogServiceClaim-SQL.xml:27` |
| `DELETE` | `POOLDATA.JSON_KLAIM_LOG` | `RDB List/InsertClaimPNC-SQL.xml:77` |
| `UPDATE` | `T_LOGINCOAS` | `RDB List/BrowseOldEmailCoas-SQL.xml:69` |
| `UPDATE` | `pooldata.mst_login_surveyor` | `RDB List/UpdateMasterLoginSurvey-SQL.xml:9` |
| `RDB-DELETE` · `OBJ-DELETE` | berbagai | 12 step di 7 activity · 2 step di 2 activity |

Seluruhnya tidak dibawa ke sistem baru.

**Konsekuensi desain yang harus ditangani di `B-1`.** Sistem lama memakai pola **hapus-lalu-sisip-ulang** sebagai mekanisme idempotensi: `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503` menghapus **12 tabel** milik satu klaim lalu menyisipkan ulang seluruh pohonnya. Pola itu **bergantung pada penghapusan fisik** dan tidak dapat dipertahankan apa adanya.

Penggantinya harus diputuskan saat menulis tiket `B-1` — kandidatnya *upsert* berbasis kunci alami, atau versioning dengan penanda baris aktif. Catatan pendukung: dua tabel pada blok itu (`T_DLALIST`, `T_PLALIST`) **delete-nya sudah dikomentari** di sumber (`:497`, `:498`) sementara insert-nya tetap aktif (`:1296`, `:1325`) — sehingga konversi ulang pada kedua tabel itu **sudah berpotensi menduplikasi baris hari ini**. Pola penggantinya sekaligus menutup cacat yang sudah ada.

**Konsekuensi untuk uji kesetaraan.** Soft delete **tidak mengubah hasil yang dilihat pengguna** — yang terhapus tetap tidak muncul. Tetapi ia **mengubah isi tabel**. Perkakas `S-8` karena itu membandingkan **hasil kueri sesuai aturan bisnis**, bukan jumlah baris mentah; perbandingan berbasis `COUNT(*)` pada tabel akan selalu berbeda dan bukan indikasi cacat.

**Konsekuensi untuk `P-5`.** Soft delete **tidak ditambahkan** ke daftar 13 perbaikan eksplisit `D-49`, karena ia tidak mengubah hasil yang terlihat — ia mengubah cara penyimpanan. Bila pada uji kesetaraan muncul selisih yang berasal dari perbedaan ini, ia diperlakukan sebagai **selisih metode pembandingan**, bukan selisih hasil, dan ditangani di sisi perkakas `S-8`.

**Dampak ke Steering:** Database Strategy · Domain Model · Security · Module Breakdown B-1, S-5, S-8 · BRD Bab 17.5 dan 21.2 kriteria #9

---

### D-67 · Tidak Ada Akun Pribadi sebagai Penerima Notifikasi
**Status:** DECIDED

**Pertanyaan (Ronde 7 Q29):** Export memuat sedikitnya enam alamat Gmail pribadi sebagai penerima notifikasi di jalur produksi, ditambah lima alamat Gmail huruf besar yang dipakai sebagai **Operator ID** di filter laporan KPI.

**Pilihan yang ditawarkan:**
1. Ganti seluruhnya dengan mailbox fungsional dari master Penerima Notifikasi
2. Sebagian masih sah — work owner menyebutkan mana
3. Orangnya sudah tidak bekerja di sini; hapus saja
4. Perlu dicek dulu

**Jawaban:** Opsi 1

**Keputusan akhir:** **Tidak ada satu pun alamat pribadi yang dibawa ke sistem baru.** Seluruh penerima notifikasi berasal dari master **Penerima Notifikasi** (`F-4`), berupa mailbox fungsional. Pemetaan konkret alamat lama ke mailbox pengganti ditetapkan saat master diisi.

Lokasi yang harus dibersihkan, seluruhnya terverifikasi:

| Jenis | `berkas:baris` |
|---|---|
| Gmail pribadi sebagai penerima | `Activity/AutoEmailDownloadProposeAdjustment-Act.xml:7872`, `:8031` · `Activity/SetStsSalvagePNC_act-Act.xml:694` · `Activity/LetterOfAssignment2_Act-Act.xml:7725` (elemen `<To>`) · `Activity/SendEmailUnprotectPremi-Act.xml:4823` |
| Gmail huruf besar sebagai **Operator ID** di filter KPI | `Activity/PNCReportKPI_act-Act.xml:7913`, `:9564` · `Activity/GetNextFUdata_act-Act.xml:3522` · `Activity/EksportDataAllKPIPICKlaim-Act.xml:3790` |

**Catatan yang memperluas cakupan `F-4`:** lima alamat Gmail yang dipakai sebagai Operator ID **bukan** masalah notifikasi melainkan masalah **identitas** — mereka muncul di klausa `pic in (…)` pada SQL laporan. Pembersihannya menyentuh `F-3` (identitas) sekaligus `F-4` (master), bukan hanya master penerima notifikasi.

**Dampak ke Steering:** Domain Model (master Penerima Notifikasi) · Module Breakdown F-3 dan F-4 · Security · `FR-F4`

---

### D-68 · Claim PNC Satu-satunya Pemanggil Stored Procedure — B-4 dan B-9 Dapat Dibuat Atomik
**Status:** DECIDED

**Pertanyaan (Ronde 7 Q30):** `D-02` memutuskan aplikasi baru tidak memanggil stored procedure. Tetapi **10 dari 12 procedure** yang dibaca melakukan `COMMIT` sendiri — `Database/INSERT_PLADLA.prc` sembilan kali (`:69`, `:74`, `:79`, `:138`, `:143`, `:148`, `:179`, `:184`, `:189`) dengan hanya satu `ROLLBACK` di handler terluar (`:198`), dan `Database/UPDATEREAS.prc` empat kali. Selama procedure itu dipanggil apa adanya, `B-4` dan `B-9` tidak dapat dibuat atomik. Menulis ulangnya di Go menyelesaikan itu — tetapi hanya bila Claim PNC satu-satunya pemanggilnya.

**Pilihan yang ditawarkan:**
1. Claim PNC satu-satunya pemanggil — procedure boleh ditinggalkan setelah logikanya naik ke Go
2. Ada sistem lain yang memanggilnya — procedure harus tetap hidup
3. Belum tahu — perlu ke DBA untuk memeriksa dependensi
4. Semua procedure tetap hidup apa pun jawabannya

**Jawaban:** Opsi 1

**Keputusan akhir:** **Claim PNC adalah satu-satunya pemanggil** stored procedure yang dibahas. Setelah logikanya dinaikkan ke Go sesuai `D-02`, procedure tersebut **boleh ditinggalkan** — tidak perlu dipelihara demi sistem lain.

**Yang ini lepaskan:**
- **`B-4` Spreading Reasuransi** dan **`B-9` PLA/DLA** dapat dibuat **atomik**. Penerbitan PLA/DLA yang di sistem lama menempuh sembilan `COMMIT` dan berpotensi meninggalkan state setengah jalan (karena `ROLLBACK`-nya terjadi setelah commit) kini dapat dibungkus satu transaksi.
- Kepemilikan transaksi berpindah sepenuhnya ke lapisan Go, konsisten dengan `D-02` ("database menjadi penyimpanan murni").
- Kontrak galat berbasis string `ErrMsg` — yang pada enam procedure **tidak di-set pada jalur sukses** sehingga `NULL` berarti sukses, dan pada `ADD_NEWMASTERVIRTUALACCOUNT.prc:18` bahkan **membawa nomor virtual account** sekaligus pesan galat — tidak ikut dibawa.

**Yang tetap harus diminta ke DBA meski keputusan ini diambil:** **12 dependensi** yang dipanggil oleh 62 procedure yang sudah diterima, terberat `UPDATE_LOG_KONVERSI` (**162 pemanggilan**), `GETNEWID` (42), `PKG_COUNTER_PRODUCTION` (25), `PROCESSQUEUEDIRECT` (10). Logikanya tetap perlu dibaca untuk ditulis ulang, meski objeknya nanti ditinggalkan.

**Catatan verifikasi:** keputusan ini dapat dikonfirmasi DBA dengan satu kueri katalog (`ALL_DEPENDENCIES`) bila kelak diperlukan bukti tertulis sebelum procedure benar-benar dinonaktifkan.

**Dampak ke Steering:** Technical Strategy (transaksi) · Database Strategy · Module Breakdown B-4 dan B-9 · `D-02`

---

### D-69 · Perlakuan Nama Orang dan Alamat Email di ADR dan Tiket
**Status:** DECIDED

**Pertanyaan (Ronde 7 Q31):** Apakah nama Operator ID dan alamat email boleh tertulis lengkap di ADR dan tiket yang akan di-commit? Nama seperti `MORASOTARDODOTARIGAN`, `ELLENSUPRIYATI`, dan `INDRAGUNAWAN` **sudah tertulis apa adanya** di dokumen yang disetujui (`D-15`, `20-DETAIL-KOMITE-DBLINK.md` §1.5.4); alamat email tidak pernah.

**Pilihan yang ditawarkan:**
1. Nama Operator ID boleh ditulis; alamat email selalu disamarkan; data nasabah tidak pernah
2. Keduanya disamarkan mulai sekarang, termasuk nama Operator ID
3. Keduanya boleh ditulis lengkap
4. Perlu ke Compliance

**Jawaban:** Opsi 1

**Keputusan akhir:** Aturan penulisan untuk seluruh dokumen yang akan di-commit:

| Jenis nilai | Perlakuan |
|---|---|
| **Nama Operator ID** | **boleh ditulis lengkap** — preseden sudah ada di Steering yang disetujui, dan diperlukan agar tiket dapat menunjuk hardcode mana yang harus dihapus |
| **Alamat email** | **selalu disamarkan** — bagian sebelum `@` diganti, domain boleh tampak |
| **Data nasabah** — nomor polis, nama tertanggung, nomor klaim asli, NPWP, nomor rekening | **tidak pernah ditulis**; dirujuk dengan `berkas:baris` atau nama kolom saja |
| **Kredensial, kunci API, hostname/IP produksi** | **tidak pernah ditulis**; dirujuk dengan `berkas:baris` + nama elemen (lihat `D-40`) |

**Alasan:** nama Operator ID diperlukan untuk ketelusuran — tanpanya tiket `F-4` tidak dapat menunjuk hardcode mana yang dihapus. Alamat email tidak menambah ketelusuran apa pun, sehingga tidak ada alasan menuliskannya.

**Yang tidak berubah:** `D-64` menetapkan data nasabah nyata disalin ke staging. Keputusan ini mengatur **isi dokumen**, bukan isi lingkungan — keduanya berdiri sendiri.

**Berlaku surut:** aturan ini sudah dijalankan sepanjang Fase 1 dan Fase 2 pada `docs/verifikasi-bukti-adr.md` dan seluruh entri Decision Log Sesi 3.

**Dampak ke Steering:** tidak ada — ini aturan penulisan artefak, bukan arsitektur.

---

### D-70 · `TYPE_KOMITE` adalah Pita Nilai **hanya** di Non-MBU — Membatasi D-52
**Status:** DECIDED — **membatasi cakupan `D-52`** (tidak menyupersede; isi `D-52` tetap benar untuk Non-MBU)

**Pertanyaan (Ronde 8 Q32):** Saat menajamkan definisi "kumulatif" di `CONTEXT.md`, work owner menegaskan bahwa akumulasi jenjang didahului filter `TYPE_KOMITE` (`'1'` atau `'2'`) untuk menetapkan jenjang awal. Verifikasi ke master membuktikan aturan itu **tidak berlaku seragam di semua lini**.

**Bukti yang memicu pertanyaan:**

| Bukti | Isi |
|---|---|
| 17 SQL rule membaca `EMAILKOMITE`; **4** memfilter `TYPE_KOMITE` | `EmailKomiteBerjenjang_sql-SQL.xml:52` · `EmailKomiteBerjenjangPATKI_sql-SQL.xml:82` · `EmailKomiteAdjuster_sql-SQL.xml:101` · `EmailKomiteSalvage_sql-SQL.xml:96` |
| Jalur utama **PA** dan **Travel** **tidak** memfilternya | `EmailKomiteBerjenjangPA_sql-SQL.xml:5` · `EmailKomiteBerjenjangTravel_sql-SQL.xml:103` |
| `LIMIT_TOP` tetap **0 kemunculan** di seluruh `RDB List/` | menegaskan kembali `D-47` |

Isi master `Database/emailkomite.csv` (baris aktif, `STS_AKTIF='1'` dan `STS_ADJ='1'`):

| Lini | Ambang bawah per jenjang | `DEGREE` | `TYPE_KOMITE` | `berkas:baris` |
|---|---|---|---|---|
| **NONMBU** | `0` · `50.000.001` | 1 · 1 | **1** · **1** | `emailkomite.csv:7`, `:5` |
| **NONMBU** | `100.000.001` · `500.000.001` · `1.000.000.001` | 2 · 3 · 4 | **2** · **2** · **2** | `emailkomite.csv:2`, `:3`, `:4` |
| **PA** | `0` · `10.000.001` · `50.000.001` · `100.000.001` | 1 · 2 · 3 · 4 | **2** · **1** · **1** · **2** | `emailkomite.csv:24-27` |
| **TRAVEL** | `0` · `50.000.001` · `100.000.001` | 1 · 2 · 3 | **1** · **1** · **1** | `emailkomite.csv:21-23` |

Pada **Non-MBU** nilai `TYPE_KOMITE` memang memisahkan pita ≤ Rp 100 Jt dari pita di atasnya. Pada **PA** nilainya berselang-seling **2 · 1 · 1 · 2** menaiki tangga — ia membedakan **jalur PA reguler dari jalur PA TKI** (`EmailKomiteBerjenjangPATKI_sql-SQL.xml:82` mematok `type_komite='2'`), bukan pita nilai. Pada **Travel** seluruh jenjang aktif bernilai `1`, termasuk jenjang di atas Rp 100 Jt.

**Akibat bila filter pita diberlakukan seragam** (dihitung dari master di atas):

| Kasus | Perilaku sistem lama | Bila pita disaring dulu |
|---|---|---|
| PA Rp 5.000.000 | 1 penyetuju | **0 penyetuju** — klaim mandek |
| PA Rp 75.000.000 | 3 penyetuju | 2 penyetuju |
| PA Rp 150.000.000 | 4 penyetuju | 2 penyetuju |
| Travel Rp 150.000.000 | 3 penyetuju | **0 penyetuju** — klaim mandek |
| Non-MBU, seluruh nilai | — | tidak berubah |

**Pilihan yang ditawarkan:**
1. Filter pita berlaku **per lini sesuai kueri hari ini**; `D-52` dinyatakan berlaku khusus Non-MBU
2. Filter pita berlaku seragam, master PA dan Travel diperbaiki sebelum migrasi
3. `TYPE_KOMITE` dipecah menjadi dua kolom di sistem baru: pita nilai dan varian jalur
4. Perlu dikonfirmasi dulu ke pengguna bisnis PA dan Travel

**Jawaban:** Opsi 1 — *"filter type_komite khusus Non MBU"*

**Keputusan akhir:** **`TYPE_KOMITE` diperlakukan sebagai pita nilai hanya pada lini Non-MBU.**

- **Non-MBU** — pita ditetapkan **lebih dulu** dari nilai klaim (≤ Rp 100.000.000 → pita `1`; di atasnya → pita `2`, sesuai `D-52`), lalu jenjang diakumulasi **di dalam pita itu saja**. Akumulasi tidak menyeberang antarpita.
- **Lini lain** (PA, Travel, Simasnet, Bonding, NONMBUAB, NONMBUC) — **tidak ada** langkah pendahuluan. Akumulasi `LIMIT_BOTTOM <= nilai klaim` berjalan atas seluruh jenjang lini tersebut, persis seperti kuerinya hari ini.
- **PA TKI** tetap jalur tersendiri yang membatasi diri pada `TYPE_KOMITE = '2'`, dan itu **bukan** pita nilai melainkan penanda varian jalur.

**Cakupan `D-52` dibatasi, bukan dibatalkan.** Isi `D-52` — pita ≤ Rp 100 Jt memakai `TYPE_KOMITE=1` — tetap benar dan tetap berlaku, tetapi **hanya untuk Non-MBU**. Di luar Non-MBU, `TYPE_KOMITE` tidak boleh dibaca sebagai pita nilai.

**Mengapa ini yang benar terhadap `P-5`:** perilaku sistem lama dipertahankan persis di setiap lini, sehingga uji kesetaraan gerbang 1 pada `B-7` dapat dijalankan tanpa pengecualian. Menyeragamkan filter justru akan menghasilkan klaim tanpa penyetuju sama sekali pada dua kasus di atas — perubahan perilaku, bukan perbaikan.

**Konsekuensi yang harus ditangani saat menulis tiket:**

1. **`B-7` (Komite) wajib menguji per lini, bukan satu model tunggal.** Kasus uji minimum: PA Rp 5 Jt (1 penyetuju), PA Rp 75 Jt (3), PA Rp 150 Jt (4), Travel Rp 150 Jt (3), Non-MBU Rp 80 Jt (2), Non-MBU Rp 750 Jt (2), Non-MBU Rp 2 M (3).
2. **Satu kolom memikul dua arti berbeda.** `TYPE_KOMITE` berarti pita nilai di Non-MBU dan varian jalur di PA. Opsi 3 (memecahnya menjadi dua kolom) **tidak diambil**; karena itu model data baru wajib mendokumentasikan arti gandanya, dan validasi master `LIMIT_TOP` dari `D-47` hanya dapat dijalankan **per lini** — bukan lintas lini.
3. **Belum terverifikasi — dibawa ke tiket `B-7` dan `B-12`:** tiga kueri yang memfilter `TYPE_KOMITE` secara dinamis — `EmailKomiteBerjenjang_sql` (dipanggil `Activity/SetEmailKomite-Act.xml`), `EmailKomiteAdjuster_sql` (`Activity/SetEmailKomiteAdjuster-Act.xml`), `EmailKomiteSalvage_sql` (`Activity/SetEmailKomiteSalvage-Act.xml`) — menerima nilai `TYPE_BUSINESS` dan `TYPE_KOMITE` dari pemanggilnya lewat `tempAdj.pyMemo` dan `tempAdj.AcceptedNo`. **Lini apa saja yang benar-benar melewati ketiga kueri itu belum ditelusuri sampai ke sumber nilainya.** Bila ternyata ada lini selain Non-MBU yang melewatinya, perilaku lini tersebut mengikuti kuerinya apa adanya — bukan aturan pita. Penelusuran ini menjadi bagian Definition of Ready tiket `B-7` dan `B-12`.

**Dampak ke Steering:** Domain Model (Pita Nilai Komite, Jenjang Kumulatif) · Module Breakdown B-7 dan B-12 · Testing Strategy · `D-52` · `D-47`

---

### D-71 · Format Nomor Klaim Menjadi `PNCN.YY.xxxx` — Menyupersede Bagian Format pada D-22
**Status:** DECIDED — **menyupersede `D-22` pada bagian format nomor**; sisa isi `D-22` tetap berlaku

**Latar:** `D-22` menetapkan format `PNCN-xxxx` dengan sintaks `'PNCN-' || TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)`. Format itu tidak memuat unsur tahun, sehingga pertanyaan "berapa digit `xxxx`, dan apakah memuat unsur tahun" tercatat sebagai pertanyaan terbuka pada `ADR-0009`.

**Jawaban work owner:**
> "koreksi dulu penomoran klaim dari format PNCN-xxxx menjadi PNCN.YY.xxxx menggunakan sequence di oracle yang akan diberi nama POOLDATA.CLAIM_NO_NONPEGA_SEQ dengan syntax `'PNCN' || '.' || TO_CHAR(SYSDATE,'RR') || '.' || TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)`"

**Keputusan akhir:** Nomor klaim yang diterbitkan sistem baru berformat **`PNCN.YY.xxxx`** — tiga segmen dipisahkan **titik**:

| Segmen | Isi | Sumber |
|---|---|---|
| `PNCN` | penanda tetap asal sistem baru | literal |
| `YY` | dua digit tahun | `TO_CHAR(SYSDATE,'RR')` |
| `xxxx` | nomor urut | `TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)` |

Sintaks Oracle yang ditetapkan, apa adanya:

```sql
'PNCN' || '.' || TO_CHAR(SYSDATE,'RR') || '.' || TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)
```

**Yang tetap berlaku dari `D-22`** — tidak disupersede:
- Prefix `ASM-FW-GCNMFW-WORK` **tidak dipakai lagi**.
- Nomor klaim lama **dibiarkan apa adanya**, tidak dinomori ulang; kedua format hidup berdampingan permanen.
- Prefix `PNCN` membuat asal sebuah klaim terbaca langsung dari nomornya tanpa tabel pemetaan.

**Yang disupersede:** bentuk `PNCN-xxxx` (tanda hubung, tanpa segmen tahun) beserta sintaks lamanya.

**Fakta yang diverifikasi ulang saat keputusan ini diambil:** `POOLDATA.CLAIM_NO_NONPEGA_SEQ` **nol kemunculan di seluruh export** — sequence itu memang **belum ada** dan akan dibuat, sesuai kalimat *"yang akan diberi nama"* pada `D-22` maupun `D-71`. Catatan pada `ADR-0009` yang menyebut "pemakaiannya oleh sistem lain belum diverifikasi" karena itu **dikoreksi**: bukan belum diverifikasi, melainkan objeknya belum ada.

**Tiga hal yang mengikuti dari sintaks ini dan perlu diputuskan sebelum `B-2` ditulis lengkap — tidak saya isi sendiri:**

1. **Apakah sequence direset setiap awal tahun?** Sintaksnya memakai satu sequence global. Bila **tidak** direset, nomor urut terus bertambah melewati pergantian tahun — `PNCN.26.8125` diikuti `PNCN.27.8126` — sehingga segmen tahun bersifat penanda, bukan penghitung per tahun. Bila **direset**, nomor urut mulai dari 1 tiap tahun dan keunikan lintas tahun dijamin segmen `YY`. Keduanya sah; pilihannya mengubah cara nomor dibaca orang.
2. **`TO_CHAR(...NEXTVAL)` tanpa format mask tidak memberi angka nol di depan**, sehingga lebar segmen terakhir berubah-ubah: `PNCN.26.9` lalu `PNCN.26.10` lalu `PNCN.26.1000`. Akibatnya **pengurutan sebagai teks tidak sesuai urutan penerbitan** (`.10` mendahului `.9`). Bila lebar tetap dikehendaki, mask-nya perlu ditetapkan (mis. `TO_CHAR(seq.NEXTVAL,'FM0000')`). Sintaks yang ditetapkan work owner dipakai apa adanya sampai ada keputusan lain.
3. **`SYSDATE` adalah tanggal server basis data**, bukan tanggal kejadian maupun tanggal registrasi. Klaim yang diterbitkan di sekitar pergantian tahun akan mengambil tahun dari jam server — bertaut dengan `R-12` (pergeseran zona waktu) dan dengan temuan penyesuaian 7 jam pada `D-49` butir 3.

**Catatan teknis netral:** pada `TO_CHAR`, format `'RR'` menghasilkan dua digit tahun yang **identik** dengan `'YY'`; perbedaan perilaku `RR` hanya berlaku saat menafsirkan masukan (`TO_DATE`), bukan saat mengeluarkan teks. Sintaks work owner dipakai apa adanya.

**Konsekuensi untuk `ADR-0005`:** generator nomor klaim tetap menjadi **satu-satunya tempat dengan sakelar dialek eksplisit**. Padanan PostgreSQL-nya menyentuh dua hal sekaligus — `nextval('pooldata.claim_no_nonpega_seq')` dan `to_char(current_date,'YY')` — sehingga sakelar itu kini membungkus dua perbedaan dialek, bukan satu.

**Dokumen yang memuat format lama dan perlu direvisi — diajukan sebagai usulan, belum dikerjakan:**

| Dokumen | Lokasi |
|---|---|
| `docs/BRD.md` | `:246`, `:296`, `:311`, `:322`, `:770`, `:883`, `:1436` |
| `docs/requirement-summary.md` | `:24` (`P-3`), `:296` (`DAT-03`) |
| `docs/prompt-adr-ticketing.md` | `:193`, `:448` |

`docs/interview-history.md:65` **tidak diusulkan berubah** — berkas itu catatan riwayat wawancara, dan isinya memang merekam jawaban yang berlaku saat itu.

**Yang sudah dikerjakan atas keputusan ini:** `docs/adr/0009-penomoran-klaim-pncn.md` dan `docs/adr/README.md` dikoreksi, lalu `docs/ADR.md` dan `docs/ADR.docx` dibangun ulang.

**Dampak ke Steering:** Domain Model · Database Strategy · Module Breakdown B-2 · `D-22` · `ADR-0005`, `ADR-0009`

---

### D-72 · Steering dan BRD Diperbarui Langsung — Larangan Menyunting Dicabut
**Status:** DECIDED — **mencabut batasan kerja awal**, bukan menyupersede keputusan arsitektur mana pun

**Latar:** aturan kerja proyek sejak awal berbunyi *"JANGAN mengedit `docs/STEERING.md` maupun berkas `.docx`"*; temuan yang menuntut revisi Steering ditulis sebagai **usulan** yang menunggu persetujuan. Sepanjang Sesi 3 terkumpul **41 keputusan** (`D-31`…`D-71`), **37 di antaranya berdampak ke Steering**, ditambah 14 temuan Fase 1 (`T-1`…`T-14`) dan 40 kontradiksi (`K-1`…`K-40`) yang membuat sejumlah pernyataan di Steering dan BRD tidak lagi benar.

**Pertanyaan (Ronde 9 Q34–Q36):** bentuk pembaruan · perlakuan `STEERING.md` terhadap 20 bab sumber · perlakuan kedua berkas `.docx`.

**Jawaban work owner:**
> "Saya pindahkan file Steering.md beserta format .docx dan file BRD.md beserta format .docx ke folder masing2. Update juga semua perubahan ke dokumen Steering dan BRD ini."
> "Q34 Opsi 1 / Q35 Opsi 1 / Q36 Opsi 3"

**Keputusan akhir:**

| Hal | Keputusan |
|---|---|
| **Kewenangan menyunting** | `docs/Steering/STEERING.md`, `docs/BRD/BRD.md`, dan kedua berkas `.docx` **boleh diperbarui langsung**. Larangan lama dicabut. |
| **Bentuk pembaruan (Q34 Opsi 1)** | Sunting di tempat menyeluruh, **ditambah satu bab baru "Riwayat Revisi"** di depan yang meringkas apa yang berubah dan atas dasar `D-nn` mana |
| **Sumber tunggal (Q35 Opsi 1)** | Bab sumber di `docs/Steering/` menjadi satu-satunya tempat menyunting; `STEERING.md` **dibangun ulang otomatis** oleh `docs/tools/build-steering.js` |
| **Berkas `.docx` (Q36 Opsi 3)** | Dibangun ulang dengan pipeline yang sama seperti `ADR.docx`; berkas `.docx` lama **disimpan sebagai arsip** dengan nama bertanda versi lama |

**Struktur berkas baru, ditetapkan work owner:** `docs/ADR/` (ADR + `ADR.md` + `ADR.docx`) · `docs/BRD/` (`BRD.md` + `.docx`) · `docs/Steering/` (20 bab sumber + `CONTEXT.md` + `00-DECISION-LOG.md` + `STEERING.md` + `.docx`).

**Koreksi atas premis yang saya pakai menyusun Q35 — disampaikan sebelum pekerjaan dimulai.** Saya menyebut `STEERING.md` sebagai "dokumen gabungan dari 20 bab". Pemeriksaan membuktikan hubungannya **tidak seragam**:

| Bagian `STEERING.md` | Hubungan dengan berkas sumber | Bukti |
|---|---|---|
| **Lampiran A, C, D, E, F** | **salinan verbatim** dari `CONTEXT.md`, `01-FRONTEND-ANALYSIS.md`, `18-SKILLS-USAGE-LOG.md`, `19-GAP-EXPORT-DETAIL.md`, `20-DETAIL-KOMITE-DBLINK.md` | teks identik kata per kata |
| **Lampiran B** | **salinan verbatim `00-DECISION-LOG.md` yang berhenti di `D-30`** — 752 baris, 30 entri, sementara sumbernya kini 2.070 baris dan 71 entri | `STEERING.md:2615-3366` vs `00-DECISION-LOG.md:1-752` |
| **Bab 1–26** | **tulisan ulang yang diringkas**, bukan salinan | Bab 21 memuat `R-01`…`R-12` yang sama dengan `16-RISK-ANALYSIS.md` tetapi dalam **prosa 149 baris**, sedangkan sumbernya **tabel 347 baris** |

**Konsekuensi yang mengikuti dari Q35 Opsi 1, dan diterima:** karena bab 1–26 akan dibangun dari berkas sumber, **kata-kata hasil peringkasan pada bab 1–26 tidak dipertahankan** — yang berlaku adalah teks bab sumber. Akibatnya `STEERING.md` menjadi **lebih panjang dan lebih rinci**, dan susunannya mengikuti berkas sumber. Sebagai gantinya:

1. Tidak ada lagi dua versi pernyataan yang sama yang bisa berbeda.
2. **Lampiran B ikut terbarui dengan sendirinya** dari `D-30` menjadi `D-01`…`D-72` — salah satu keusangan terbesar di dokumen itu tertutup tanpa pekerjaan terpisah.
3. Setiap revisi berikutnya cukup menyunting satu tempat.

**Bab baru yang dibuat:** `docs/Steering/21-RIWAYAT-REVISI.md` — berisi ringkasan perubahan v1.0 → v2.0 beserta dasar `D-nn`-nya. Nomor `21` adalah **nomor berkas**, bukan ID modul/risiko/requirement, sehingga tidak melanggar aturan penomoran proyek.

**Yang tetap tidak berubah:** rule XML Pega tetap **baca-saja**. Larangan yang dicabut hanya menyangkut dokumen proyek.

**Dampak ke Steering:** seluruh bab · `README.md` · `docs/tools/`

---

### D-73 · Seluruh Modul Ditiketkan dan Dinamai Menurut Fungsi Bisnis
**Status:** DECIDED — **membatasi `D-41`**, tidak mencabutnya

**Pertanyaan:** Work Owner menyatakan kode modul `F-1`, `F-2`, dan seterusnya **tidak dapat dipahami** saat membaca papan tiket, dan meminta modul dinamai seperti yang dikenalnya sehari-hari — *"Saya ingin seperti contoh: Input Receive Document, Input Register, dll"* — serta meminta **tiket lengkap untuk masing-masing modul**, dengan kode tetap di depan agar rujukan silang ke ADR tetap dapat dicari.

**Keputusan akhir — dua bagian.**

**1. Penamaan.** Setiap folder modul dinamai `<KODE>-<Nama-Fungsi-Bisnis>`, misalnya `B-14-Input-Receive-Document`, `B-2-Input-Register`, `S-7-Dashboard-TAT-dan-KPI`. Nama bisnisnya **tidak dikarang**: diambil dari tahapan `Flow/Register_Flow.xml` dan dari nama harness di export, sehingga yang terbaca Work Owner adalah istilah yang memang dipakai sistem berjalan. Kode tetap di depan supaya rujukan dari ADR, Steering, dan BRD tidak putus.

**2. Cakupan.** **Seluruh 33 modul kini punya tiket** — 102 tiket. Ini melampaui `D-41` Opsi 1 (8 modul penuh + 25 stub), dan karena itu dicatat sebagai keputusan tersendiri alih-alih dijalankan diam-diam.

**Yang TIDAK berubah — dan inilah bagian terpenting entri ini.** `D-41` menolak Opsi 4 dengan alasan yang **masih berlaku sepenuhnya**: menulis acceptance criteria berangka untuk modul yang penghalangnya belum hilang akan menghasilkan kriteria yang mengarang. Aturan itu **tetap ditegakkan**. Yang berubah hanya **bentuk** tiketnya, bukan izin untuk mengarang:

- Modul terhalang mendapat **struktur tiket lengkap** — hasil dan nilai bisnis, lingkup, non-goal, constraint, rollout/rollback, bukti ruang lingkup.
- Bagian acceptance criteria-nya **tidak diisi dengan angka yang dikarang**. Sebagai gantinya, setiap tiket terhalang memuat bagian **"Yang kurang dan siapa yang bisa melengkapinya"** yang menyebut artefak atau keputusan yang hilang **beserta pemiliknya** dan **alasan ia menahan**.
- Satu tiket **menolak menuliskan acceptance criteria sama sekali**: `TKT-S6-003` (persetujuan komite otomatis jam 06:00), karena pertanyaan pokoknya — apakah perilaku itu memang dikehendaki — belum dijawab Work Owner. Menuliskan daftar bercentang di sana akan membuat tiket itu tampak siap dikerjakan padahal lingkupnya belum ada.

**Hasil terukur:** 102 tiket · **30** `ready-for-human` · **72** `needs-info` (**71%**). Kesiapan modul tidak berubah dari `D-41`: **1 PENUH · 23 SEBAGIAN · 9 TERHALANG** (`F-3` terhalang hanya pada bagian identitas). Bertambahnya jumlah tiket **tidak menaikkan kesiapan satu modul pun** — ia hanya membuat lingkup yang selama ini tersembunyi di balik stub menjadi terlihat.

**Temuan yang muncul justru karena penulisan ini dilakukan.** Empat hal berikut tidak akan terlihat bila 25 modul dibiarkan sebagai stub, dan ketiga yang pertama **memintas kendali yang sedang dibangun modul lain**:

1. **`JOBForKomiteKlaimPNC` menjalankan `AutoAcceptKomite` tiap hari 06:00** — menyetujui komite **tanpa pengguna**, di luar jangkauan kontrol berbasis menu (`D-59`).
2. **Dua Service REST masuk menerima persetujuan komite dari sistem lain** — permukaan yang belum pernah diaudit.
3. **18 dari 21 Connect REST ber-`pyUseAuthentication=false`**, satu di antaranya memakai `http://` tanpa TLS untuk data premi.
4. **`GETSELISIHJAM.fnc` dan `GET_WORKING_HOURS@ASMD` adalah dua basis TAT yang saling eksklusif**, ditambah `RETURN 0` yang menyamarkan kegagalan sebagai nol jam.

Bila `B-7` dibangun dengan cermat sementara ketiga jalur pertama dibiarkan, kecermatan itu tidak ada artinya.

**Koreksi angka yang ditemukan saat penulisan:**

| Angka lama | Terverifikasi | Sumber |
|---|---|---|
| `SendEmailNotification` **17 pemanggil** (`verifikasi-bukti-adr.md:2783`) | **15**, dengan kelima belas pemanggilnya disebut namanya | `19-GAP-EXPORT-DETAIL.md:196` |
| Rotasi SMTP **44 lokasi** (`verifikasi-bukti-adr.md:2783`) | **31 lokasi** | `D-40` · `16-RISK-ANALYSIS.md:484` |
| **12 Connect REST keluar** (`06-MODULE-BREAKDOWN.md:63`, `16-RISK-ANALYSIS.md:193`, `BRD §FR-S4`) | **21 berkas** di `Connect REST/` — 12 yang terhitung sejak awal ditambah **9 yang baru ditemukan** | direktori `Connect REST/` (dihitung langsung) |

Ketiganya semula **diusulkan sebagai revisi**. **Work Owner menyetujui penerapannya pada 2026-09-14**, dan ketiganya sudah diterapkan ke `06-MODULE-BREAKDOWN.md`, `16-RISK-ANALYSIS.md`, `BRD.md`, `verifikasi-bukti-adr.md`, `migration-readiness.md`, dan `requirement-summary.md`. Rinciannya — termasuk **dua tempat yang sengaja tidak disunting karena berupa rekaman** — dicatat di `21-RIWAYAT-REVISI.md` §7.

**Dua hal yang masih perlu jawaban Work Owner, dan sengaja tidak saya putuskan sendiri:**

1. **Mana dari 26 harness bernama *Inbox* yang benar-benar inbox peran?** `06-MODULE-BREAKDOWN.md:85` menyebut "~20"; direktori `Harness/` memuat 26, sebagian tampaknya daftar pilihan (`ListDocumentTypeInbox`, `CauseOfLossInbox`, `StatusClaimInbox`). Jumlah layar yang dijanjikan `U-3` bergantung padanya.
2. **Pembagian 48 harness non-Inbox ke `U-4`, `U-5`, dan `U-6` per berkas.** Tanpa itu, jumlah layar ketiga modul frontend adalah perkiraan, bukan komitmen.

**`BACKLOG.md` dicabut.** Kedua puluh lima modul stub di dalamnya kini menjadi modul bertiket, sehingga berkas itu tidak lagi punya isi yang tidak ada di tempat lain. Isinya tidak hilang — ia terserap ke `spec.md` masing-masing modul.

**Dampak ke Steering:** `06-MODULE-BREAKDOWN.md` (angka Connect REST) · `16-RISK-ANALYSIS.md` (R-18) · BRD `§FR-S4` · `docs/verifikasi-bukti-adr.md` §15 · `docs/ticketing/README.md`

---

### D-74 · Fase 6 — Status 40 Usulan Revisi, dan Dua Pernyataan Lama yang Menjadi Usang
**Status:** DECIDED — **melengkapi `D-39`**, dan mengoreksi pernyataan di `D-55`

**Pertanyaan:** `D-39` menetapkan setiap selisih antara dokumen dan bukti dicatat sebagai usulan revisi, dan daftarnya disusun sebagai 40 kontradiksi `K-1`…`K-40` di `verifikasi-bukti-adr.md` §11. Daftar itu **tidak punya penanda status sama sekali**, sehingga tidak ada cara mengetahui mana yang sudah diterapkan tanpa memeriksa satu per satu. Berapa yang sebenarnya masih menggantung?

**Keputusan akhir:** sapuan dijalankan atas keempat puluhnya, hasilnya dicatat di lampiran baru **`23-STATUS-USULAN-REVISI.md`**, dan lampiran itu menjadi tempat status dilacak — **bukan** dengan menyunting §11, yang merupakan rekaman temuan.

| Status | Jumlah |
|---|---:|
| Sudah diterapkan | **11** |
| **Masih menggantung** | **11** |
| Tidak perlu tindakan (bukti mendukung dokumen) | **2** |
| Perlu keputusan, bukan koreksi teks | **11** |
| Belum tuntas diverifikasi — **tidak dinyatakan bersih** | **5** |

**Dua temuan yang lebih berat daripada koreksi angka:**

1. **`K-31` — `NFR-13` berdiri di atas premis yang tidak ada.** Requirement itu dibenarkan oleh `OFFSET 500000`, sementara `OFFSET` **nol kemunculan** di seluruh export. Ia perlu **ditulis ulang**, bukan diperbaiki angkanya.
2. **`K-16` dan `K-17` menyangkut aturan yang menentukan uang** — ambang PA/Travel yang ternyata dipicu **jabatan operator** (bukan lini bisnis), dan batas nilai klaim ≤ TSI yang **mengecualikan PA** tanpa tercatat di BRD. Keduanya ada di kelompok **belum tuntas diverifikasi**, dan layak diprioritaskan di atas koreksi angka mana pun.

**Pernyataan lama yang menjadi usang — dicatat, bukan disunting.** `D-55` berbunyi *"`docs/BRD.md` tidak disentuh; revisinya masuk daftar usulan Fase 6"*. Pernyataan itu benar saat ditulis, tetapi **`D-72` kemudian mencabut larangan menyunting BRD**, dan pencabutan `§21.4` kini benar-benar ada di `BRD.md:1373`. Entri `D-55` **tidak diubah** — aturan proyek melarang menyunting entri lama, dan mengubahnya akan menghapus jejak bahwa keadaannya pernah berbeda. Yang berlaku sekarang adalah entri ini.

**Yang belum diputuskan dan menunggu Work Owner:**

1. **Kesebelas butir "masih menggantung"** — boleh diterapkan seperti tiga koreksi pada `D-73`, atau dipilih sebagian.
2. **Kesebelas butir "perlu keputusan"** — masing-masing menuntut jawaban, bukan suntingan. Empat di antaranya menyangkut keputusan lama yang premisnya berubah: `D-15` (`K-22`), `D-18` (`K-7`), `D-20` (`K-38`), `D-26` (`K-23`).
3. **Definisi istilah "Inbox"** untuk `CONTEXT.md` — diusulkan pada sesi ini karena ternyata kata itu dipakai di seluruh dokumen **tanpa pernah didefinisikan**, sementara `Worklist` dan `Workbasket` sudah punya entri sejak `D-26`. Ini sebab langsung kebingungan penggolongan `U-3` versus `U-6`.

**Dampak ke Steering:** lampiran baru `23-STATUS-USULAN-REVISI.md` · `18-SKILLS-USAGE-LOG.md` · `21-RIWAYAT-REVISI.md`

---

### D-75 · Portal Multi-Entitas — Satu Aplikasi, Satu Database per Entitas
**Status:** DECIDED — **membatasi `D-21`/`ADR-0004`**, dan menambah pertanyaan terbuka pada `D-71`

**Pertanyaan:** Work Owner meminta aplikasi punya **pilihan portal**, sehingga tiap portal menampilkan data dari sumber yang berbeda. Berapa portal, satu database per entitas atau satu database bersama, dan bagaimana laporan, master data, perpindahan portal, serta jejak auditnya?

**Yang ditemukan sebelum pertanyaan dijawab — perilaku ini sudah ada, tetapi tersembunyi:** export memuat **48 perbandingan** terhadap `pxRequestor.pxReqServer` pada **3 hostname** — host dev (34), host entitas Insurtech (13), host entitas Timor-Leste (1). Salah satunya **mengubah ambang komite dari Rp 50.000.000 menjadi 3.500**; selisih itu bukan salah ketik melainkan **mata uang yang berbeda**. Jadi sistem lama sudah melayani beberapa entitas, dengan cara membandingkan nama server — tidak terlihat pengguna, dan tidak pernah tercatat sebagai rancangan di dokumen mana pun.

**Jawaban Work Owner:**

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Berapa portal | **Minimal 4**: Asuransi Sinar Mas · Asuransi Simas Insurtech · Sinarmas Asuransi Syariah · Timor-Leste |
| 2 | Bentuk penyimpanan | **Satu database per entitas** |
| 3 | Laporan | **Per portal** |
| 4 | Master data | **Per portal** |
| 5 | Perpindahan portal | **Tanpa login ulang** |
| 6 | Jejak audit | **Per portal** |

**Keputusan akhir:** **satu aplikasi Go, satu basis kode, empat portal, satu database per entitas**, dengan kelima ketetapan di atas. Dirinci di **`ADR-0030`**. Modul baru **`F-6` Portal & Multi-Sumber Data**, requirement **`FR-F6`**, risiko baru **`R-20`**. Total modul menjadi **34**, bukan 33.

**Jawaban Work Owner mengoreksi bukti kami, bukan sebaliknya.** Dokumen mencatat 3 hostname; Work Owner menyebut 4 portal termasuk **Syariah**. Pemeriksaan membenarkan Work Owner: **`IsServerSyariah` hilang dari export dan tidak tercatat di `19-GAP`** (dirujuk `Data Transform/SetDataEmail-DT.xml:147`, `:411` dan `Activity/SpreadingDataProtection-Act.xml:1294`, `:4153`), sebagaimana `IsDevelopmentServer`. `docs/verifikasi-bukti-adr.md:791` bahkan sudah menyimpulkan **"jumlah hostname sebenarnya bisa lebih dari 3"**. Angka 3 rendah karena **rule pembedanya hilang**. Karena itu kata **"minimal"** diperlakukan harfiah: **daftar portal adalah data, bukan konstanta di kode.**

**Alasan menolak satu database bersama dengan kolom penanda entitas:** pemisahan data akan bergantung pada **tidak adanya satu pun kueri yang lupa menyaring** — kelas kesalahan yang tidak dapat diuji habis. Dengan database terpisah, pemisahan menjadi **sifat struktural**: portal A tidak dapat membaca data portal B karena koneksinya memang berbeda.

**Konsekuensi yang harus disadari, bukan ditemukan belakangan:**

1. **`R-20` — berpindah portal tanpa login ulang berarti satu identitas menjangkau empat database.** Bila kewenangan tidak dinilai ulang saat perpindahan, pengguna dapat melihat data entitas yang bukan haknya. Itu **kebocoran data antar badan hukum**, bukan cacat tampilan.
2. **Empat database harus tetap sekerabat skemanya.** Migrasi skema (`TKT-F2-004`) berjalan empat kali; gagal di salah satunya membuat portal itu tertinggal versi.
3. **Lingkup `F-4` dan `U-6` berlipat** — master ≥29 kelompok kini per portal. Demikian pula beban `S-2`, `S-5`, `S-8`, pencadangan, dan rotasi kredensial.
4. **Laporan konsolidasi lintas portal menjadi tidak mungkin** tanpa keputusan baru. Bila manajemen memintanya kelak, itu keputusan tersendiri.
5. **Portal Syariah belum punya baseline Pega yang utuh.** `SpreadingSyariah_Act` ada, tetapi rule yang menentukan **kapan** ia berlaku hilang. Sampai Tim Pega mengirimkannya, gerbang 1 untuk portal itu **diganti ukuran lain** — sama seperti `F-3` dan `S-5` pada `D-42`, dan **tidak boleh dianggap lulus**.

**Lima pertanyaan yang sengaja dibiarkan terbuka:** jumlah portal yang sebenarnya · perlukah penanda portal di dalam nomor klaim `PNCN.YY.xxxx` karena sequence-nya per database (`D-71`) · sejauh apa aturan bisnis syariah berbeda · satu penyedia identitas untuk keempat entitas atau satu per entitas · mata uang dan pembulatan per portal.

**Dampak ke Steering:** `06-MODULE-BREAKDOWN.md` (modul `F-6`, total 34) · `16-RISK-ANALYSIS.md` (`R-20`) · `09-DATABASE-STRATEGY.md` · `11-SECURITY.md` · `ADR-0004` dibatasi · BRD `FR-F6`

---

### D-76 · Nomor Klaim Tidak Memuat Penanda Portal
**Status:** DECIDED — **menutup pertanyaan terbuka `D-71`** yang dibuka oleh `D-75`

**Pertanyaan:** `D-71` menetapkan nomor klaim `PNCN.YY.xxxx` yang diterbitkan dari sequence database. `D-75` kemudian menetapkan **satu database per entitas**, sehingga sequence-nya pun terpisah per portal. Akibatnya **dua portal dapat menerbitkan nomor klaim yang sama**. Perlukah penanda portal di dalam nomornya?

**Jawaban:** **Tidak perlu penanda portal.**

**Keputusan akhir:** format nomor klaim **tetap `PNCN.YY.xxxx`** persis seperti `D-71`, tanpa penambahan segmen portal. Nomor klaim **unik di dalam satu portal**, dan **tidak dijamin unik antar portal**.

**Konsekuensi yang dicatat supaya menjadi pilihan sadar, bukan kejutan:**

1. **Nomor klaim bukan lagi identitas tunggal di seluruh perusahaan.** Menyebut sebuah nomor klaim tanpa menyebut portalnya menjadi **ambigu**. Setiap kali nomor klaim dipakai di luar konteks satu portal — percakapan telepon, tiket dukungan, surat — portalnya harus ikut disebut.
2. **Dokumen yang keluar ke pihak luar membawa nomor itu.** LOD, PLA, dan DLA dicetak dengan nomor klaim. Nasabah, reasuradur, atau koasuradur yang menerima dokumen dari dua entitas Sinarmas dapat menerima **dua dokumen bernomor sama untuk klaim yang berbeda**. Ini akibat yang terlihat di luar sistem, bukan detail internal.
3. **Keputusan ini sejalan dengan `D-75` butir 3 dan 6**: laporan **per portal** dan jejak audit **per portal**. Selama tidak ada satu pun keluaran yang menggabungkan portal, ambiguitasnya tidak pernah muncul di dalam sistem.
4. **Bila kelak diputuskan ada laporan konsolidasi lintas portal**, keputusan ini **harus ditinjau ulang lebih dulu** — menggabungkan dua portal yang bernomor sama menghasilkan baris yang tidak dapat dibedakan. Menambah penanda portal **setelah** klaim terbit menuntut migrasi data, dan itu jauh lebih mahal daripada memutuskannya sekarang.

**Dampak pada tiket:** `TKT-F2-006` (generator nomor klaim) — salah satu pertanyaan terbukanya tertutup; sequence tetap per database tanpa penanda portal. `TKT-F6-002` — butir "penomoran klaim" pada bagian *Yang kurang* tertutup.

**Dampak ke Steering:** `09-DATABASE-STRATEGY.md` · `ADR-0009` · `ADR-0030` · BRD `FR-F2`

---

### D-77 · Portal — Perilaku Saat Perpindahan Gagal, dan Jumlah Portal
**Status:** DECIDED — melengkapi `D-75`

**Pertanyaan:** `D-75` menetapkan perpindahan portal **tanpa login ulang**, dan `R-20` menandai bahwa satu identitas kini menjangkau empat database milik empat badan hukum. Dua hal belum ditetapkan: apa yang terjadi bila perpindahan **gagal**, dan berapa portal yang harus dilayani.

**Jawaban Work Owner:**

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Bila perpindahan portal gagal | **Tampilkan pesan, dan tetap di portal sebelumnya** |
| 2 | Jumlah portal | **Saat ini 4, dan bisa bertambah** |

**Keputusan akhir:**

1. **Kegagalan perpindahan portal tidak boleh meninggalkan pengguna dalam keadaan menggantung.** Sesi tetap berada di **portal sebelumnya**, dengan pesan yang menjelaskan kegagalannya. Pengguna **tidak** dikeluarkan dari sistem, dan **tidak** dibiarkan di layar tanpa portal aktif.
2. **Empat portal sekarang, dan daftarnya harus dapat bertambah tanpa perubahan kode** — menguatkan `TKT-F6-001` yang sudah memperlakukan daftar portal sebagai **data**.

**Kenapa butir 1 penting untuk `R-20`, bukan sekadar kenyamanan.** Ada tiga kemungkinan perilaku saat perpindahan gagal, dan dua di antaranya berbahaya:

| Perilaku | Akibat |
|---|---|
| Tetap di portal sebelumnya *(dipilih)* | Aman — portal aktif tidak pernah kosong maupun salah |
| Pindah ke portal tujuan meski pemeriksaan gagal | **Kebocoran data antar badan hukum** — persis `R-20` |
| Tidak berada di portal mana pun | Permintaan berikutnya **tidak punya portal aktif**; bila kode jatuh ke koneksi default, akibatnya sama dengan baris di atas |

Keputusan ini karena itu menutup satu jalur kegagalan `R-20`, dan menjadi **acceptance criteria** di `TKT-F6-003`: perpindahan yang ditolak wajib **mempertahankan portal sebelumnya**, bukan mengosongkannya.

**Yang masih terbuka dan tidak tertutup oleh entri ini:** apakah penyedia identitas satu untuk keempat entitas atau satu per entitas · di mana kewenangan portal disimpan · apakah satu pengguna boleh berhak di lebih dari satu portal · sejauh apa aturan bisnis syariah berbeda · mata uang dan pembulatan per portal.

**Dampak ke Steering:** `ADR-0030` · `16-RISK-ANALYSIS.md` (`R-20`) · `TKT-F6-001`, `TKT-F6-003`

---

### D-78 · Satu Identitas Login untuk Keempat Portal
**Status:** DECIDED — melengkapi `D-75` dan `D-77`, menutup satu pertanyaan terbuka `ADR-0030`

**Pertanyaan:** `D-75` menetapkan perpindahan portal tanpa login ulang, dan `D-77` menetapkan mekanismenya berupa **dropdown** tanpa kembali ke halaman login. Yang belum jelas: apakah satu pengguna memakai **user ID yang sama** di keempat entitas, atau tiap entitas punya daftar penggunanya sendiri? Ini menentukan apakah perpindahan tanpa login ulang **mungkin secara teknis**.

**Jawaban:** **Login sama untuk semua entitas.**

**Keputusan akhir:** **satu identitas berlaku di keempat portal.** Pengguna masuk sekali, lalu berpindah portal lewat dropdown tanpa autentikasi ulang. **Portal adalah dimensi cakupan data, bukan batas identitas.**

**Akibat yang menyederhanakan:**

1. Perpindahan portal tidak perlu memetakan empat identitas ke satu sesi — pekerjaan yang jauh lebih besar bila jawabannya sebaliknya.
2. **Autentikasi** (`F-3`, HCC/HCQ) tetap **satu jalur**, tidak berlipat empat. Penghalang `R-14` juga tidak berlipat.
3. Jejak audit dapat menyebut pengguna dengan pengenal yang sama di keempat portal, sehingga penelusuran lintas entitas atas satu orang **tetap mungkin** meski auditnya tersimpan terpisah (`D-75` butir 6).

**Akibat yang memusatkan risiko — dicatat supaya menjadi pilihan sadar:**

**Satu kredensial yang bocor kini menjangkau data empat badan hukum.** Bila tiap entitas punya login sendiri, kebocoran satu akun terbatas pada satu entitas. Dengan keputusan ini, batas itu hilang — yang tersisa sebagai pembatas hanyalah **kewenangan portal per pengguna**. Konsekuensinya, kewenangan portal berhenti menjadi soal kenyamanan dan menjadi **satu-satunya kendali yang memisahkan empat badan hukum**. Ini memperberat `R-20`, bukan meringankannya.

**Satu hal yang menjadi terang justru karena jawaban ini — dan belum diputuskan.** Bila login sama, maka data **"pengguna ini berhak atas portal mana"** bersifat **lintas portal secara alamiah**. Ia tidak dapat disimpan "per portal" seperti master data lain (`D-75` butir 4), karena untuk memeriksa apakah seseorang berhak atas portal B, sistem harus membaca database B **sebelum** orang itu terbukti berhak membacanya — masalah ayam-telur.

Karena itu **kewenangan portal harus tersimpan di satu tempat bersama identitas**, bukan di dalam masing-masing database entitas. Ini **pengecualian yang disengaja** terhadap `D-75` butir 4, dan perlu ditegaskan Work Owner: apakah pengecualian ini diterima, dan di mana tepatnya data itu tinggal.

**Dampak ke Steering:** `ADR-0030` (satu pertanyaan terbuka tertutup) · `16-RISK-ANALYSIS.md` (`R-20` diperberat) · `11-SECURITY.md` · `TKT-F6-003` · `TKT-F3-002`

---

### D-79 · Definisi "Inbox" — Daftar Pekerjaan Milik Pengguna
**Status:** DECIDED — menutup pertanyaan terbuka `U-3` versus `U-6` pada `D-73`

**Pertanyaan:** Kata **"Inbox"** dipakai di seluruh dokumen proyek — README ticketing, Module Breakdown, dan nama modul `U-3` — tetapi **tidak pernah punya entri di `CONTEXT.md`**, sementara `Worklist` dan `Workbasket` sudah terdefinisi sejak `D-26`. Akibatnya penggolongan layar berputar-putar: apakah layar master yang menampilkan data dan dapat dicari boleh disebut Inbox?

**Jawaban:** **Inbox adalah daftar pekerjaan user.**

**Keputusan akhir:** **Inbox** = layar yang menampilkan **daftar pekerjaan milik seorang pengguna** — yakni **Tugas** dari **Worklist** atau **Workbasket** yang menunggu dikerjakan. Definisinya disandarkan pada istilah yang sudah disepakati di `D-26`, bukan pada sumbu baru.

**Empat ciri pembeda terhadap layar daftar biasa:**

| Ciri | Inbox | Layar master |
|---|---|---|
| Isi baris | **pekerjaan** | data acuan |
| Setelah ditindaklanjuti | baris **hilang** | baris **tetap ada** |
| "Hanya milik saya" | **aturan kewenangan** | tidak bermakna |
| Tenggat | ada | tidak ada |

**Layar yang menampilkan data acuan bukan Inbox** — sekalipun dapat dicari, dan sekalipun namanya di sistem lama mengandung kata "Inbox".

**Kenapa ini bukan sekadar soal istilah.** Penggolongan menentukan hal yang nyata: modul mana yang memilikinya (`U-3` atau `U-6`), bergantung pada `B-6` (penugasan) atau `F-4` (master), gelombang berapa, siapa penguji gerbang 2, dan bentuk kendali aksesnya — pada Inbox, penyaringan "hanya pekerjaan saya" **adalah kendali akses**; pada master, kendalinya "siapa boleh mengubah".

**Akibat langsung pada `U-3`.** Sebelumnya `06-MODULE-BREAKDOWN.md` menyebut "~20 inbox berbasis peran", sementara direktori memuat **26 harness bernama *Inbox***. Dengan definisi ini, **tujuh** di antaranya diusulkan pindah ke `U-6` karena bersidik jari layar master — sidik jarinya nyaris identik dengan `MasterRekening` (267–273 KB · 15 rujukan Report Definition), bukan dengan `InboxRegister_Harness` (1.330 KB · 144). Aturan penggolongan di `docs/tools/build-inventaris-harness.js` fungsi `usul()` **sudah menerapkan definisi ini**, sehingga usulannya kini berdiri di atas definisi yang disepakati, bukan sekadar heuristik ukuran berkas.

**Yang tetap terbuka:** **8 usulan bertanda `DUGAAN`** dan **5 bertanda `JANGGAL`** pada inventaris harness masih menunggu koreksi Work Owner. Dua yang paling patut ditinjau: `inboxCompliance_Harness` (145 KB, **nol** Report Definition — antrean kerja tanpa sumber data tidak masuk akal) dan `inboxAnalystDoctor_Harness` (348 KB, menyangkut **data medis** `FR-R2`, sehingga salah golong berakibat pada kendali akses).

**Dampak ke Steering:** `CONTEXT.md` (entri baru) · `06-MODULE-BREAKDOWN.md` `U-3` · `22-INVENTARIS-HARNESS.md` · `docs/ticketing/U-3-*` dan `U-6-*` · `docs/tools/build-inventaris-harness.js`

---

### D-80 · Penamaan di Dalam Kode Menjadi Bahasa Inggris — Membatasi `D-19`
**Status:** DECIDED — **membatasi `D-19`** pada bagian bahasa penamaan; sisa isi `D-19` tetap berlaku

**Pertanyaan:** `D-19` menetapkan istilah domain dipakai sebagai **ubiquitous language** di nama
tabel, struct Go, endpoint API, label UI, dan dokumentasi — dan `08-TECHNICAL-STRATEGY.md` §4.1
menurunkannya menjadi *"istilah domain memakai bahasa Indonesia, istilah teknis memakai bahasa
Inggris"*. Setelah modul Login, Home, Master Rekening, dan Master Status Klaim berjalan, aturan itu
menghasilkan kode yang berpindah bahasa dua kali dalam satu baris. Apakah penamaannya dialihkan
seluruhnya ke bahasa Inggris?

**Jawaban Work Owner (2026-09-18):**
> "ubah struktur folder code Claim PNC, dari bahasa indonesia menjadi bahasa inggris untuk penamaan
> folder, file dan code didalamnya. dan tambahkan keterangan pada CLAUDE.MD supaya selanjutnya sudah
> otomatis menggunakan bahasa inggris."

Tiga pertanyaan lanjutan diajukan sebelum satu berkas pun disentuh, dan seluruhnya dijawab:

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Teks yang dilihat pengguna ikut atau tidak | **Disamakan seperti referensi dari file XML**, kecuali tambahan yang tidak ada di XML — itu dikoreksi menjadi bahasa Inggris |
| 2 | Nama field JSON API dan nama kolom basis data | **Keduanya tetap Indonesia** |
| 3 | Komentar dan dokumen | **Komentar termasuk isi dokumen di `claim-pnc/docs` tetap bahasa Indonesia** |

**Keputusan akhir:** **seluruh nama di dalam kode memakai bahasa Inggris** — folder, berkas, paket,
tipe, fungsi, method, field struct, parameter, dan variabel lokal, di backend Go maupun frontend
TypeScript. Lima hal **tetap berbahasa Indonesia**: komentar dan dokumen, nama field JSON API,
nama tabel/kolom basis data, teks yang dilihat pengguna, serta **nama variabel lingkungan dan flag
baris perintah** — yang terakhir karena ia dipakai berkas `.env` dan skrip deployment, sehingga
menggantinya merusak lingkungan yang sudah berjalan, bukan sekadar mengganti nama.

**Apa yang dibatasi dari `D-19`, dan apa yang tidak.** Yang dibatasi **hanya bahasanya**. Sasaran
`D-19` — tidak membawa alias Pega yang salah arti — **tetap dipegang penuh**: yang dipakai adalah
padanan Inggris yang benar dari `CONTEXT.md`, bukan alias lama. `Adjustment` tetap **tidak** dipakai;
penggantinya `SettlementLine`. `Object` tetap tidak dipakai; penggantinya `InsuredItem`. `CONTEXT.md`
tetap satu-satunya sumber arti istilah, dan entrinya tetap berbahasa Indonesia.

**Alasan yang diterima:**

1. **Pustaka standar Go dan React seluruhnya berbahasa Inggris.** Penamaan campur membuat satu
   baris berpindah bahasa dua kali — `kumpulan.Tersedia()` berdampingan dengan `strings.TrimSpace`.
2. **Bahasa Indonesia tidak mengenal infleksi.** `Daftar`, `Didaftarkan`, dan `Pendaftaran` sering
   tertukar di tempat yang tidak disengaja; `List`, `Register`, dan `Registration` tidak.
3. **Rekrutmen dan rujukan.** Pencarian galat, contoh kode, dan dokumentasi pustaka seluruhnya
   berbahasa Inggris.

**Konsekuensi yang diterima secara sadar:**

1. **Satu baris kode dapat memuat dua bahasa** — `Number string \`json:"nomor_rekening"\`` — dan
   itu memang yang dikehendaki: yang kiri nama internal, yang kanan kontrak.
2. **Riwayat `git blame` pada modul yang sudah selesai terputus** oleh perubahan menyeluruh ini.
   Isolasi Protektif yang biasanya melarang menyentuh modul selesai **dicabut khusus untuk
   penggantian nama ini**, atas permintaan Work Owner; perilakunya tidak diubah sama sekali.
3. **Dokumen `claim-pnc/docs/catatan-pengembangan.md` dan `keputusan-implementasi.md` memuat jalur
   berkas lama.** Keduanya **rekaman**, bukan pernyataan yang berlaku, sehingga tidak disunting —
   sejalan dengan perlakuan yang sama pada `00-DECISION-LOG.md`.

**Dampak ke Steering:** `08-TECHNICAL-STRATEGY.md` §4.1 (ditulis ulang) · `CLAUDE.md` dan
`STEERING.md` dibangun ulang · `D-19` dibatasi

---

### D-81 · Nama Folder Modul Memakai Nama Modul Bisnis — Melengkapi `D-80`
**Status:** DECIDED — **melengkapi `D-80`** dengan satu pengecualian yang berlawanan arah

**Pertanyaan:** `D-80` menetapkan seluruh nama di dalam kode memakai bahasa Inggris, dan
penggantian namanya menghasilkan folder modul `internal/bankaccount`, `internal/claimstatus`,
`src/modules/bank-account`, dan `src/modules/claim-status`. Work Owner tidak mengenali nama-nama
itu sebagai modul yang dimintanya.

**Jawaban Work Owner (2026-09-18):**
> "Untuk modul bank-account dan claim-status diubah menjadi master-rekening dan master-status-klaim
> dan untuk prompt selanjutkan akan diberitahu nama modulnya, contoh modul Master Rekening maka
> nama modulnya pada script menjadi master-rekening"

Satu pertanyaan lanjutan diajukan sebelum berkas disentuh: apakah aturan ini berlaku di backend
juga, atau hanya di frontend — karena `bank-account` dan `claim-status` yang disebut Work Owner
adalah nama folder **frontend**, sementara backend memakai `bankaccount` dan `claimstatus`.
**Jawaban: backend dan frontend.**

**Keputusan akhir:** **nama folder modul mengikuti nama modul bisnis yang disebut Work Owner**,
dalam bahasa Indonesia.

| Lapisan | Bentuk | Contoh |
|---|---|---|
| Folder backend dan nama paket Go | `namamodul` — huruf kecil, **tanpa tanda hubung** | `internal/masterrekening` · `internal/masterstatus` |
| Folder frontend | `nama-modul` — `kebab-case` | `src/modules/master-rekening` · `src/modules/master-status-klaim` |

**Isi modulnya tetap berbahasa Inggris** sesuai `D-80`. Yang berbahasa Indonesia hanyalah **nama
modulnya** — tipe, fungsi, field, dan variabel di dalamnya tidak berubah:

```go
package masterrekening

type Account struct {
	Number string `json:"nomor_rekening"`
}
```

**Kenapa pengecualian ini justru memperkuat `D-80`, bukan melemahkannya.** `D-80` menetapkan yang
tetap berbahasa Indonesia adalah hal-hal yang **dipakai orang di luar kode** — kontrak API, kolom
basis data, teks layar, variabel lingkungan. Nama modul masuk kategori yang sama: ia dipakai Work
Owner saat meminta pekerjaan, tertulis di tiket (`docs/ticketing/`), dan dipakai saat membicarakan
lingkup. Nama yang tidak dikenali orang yang memesannya bukan nama yang berguna.

**Nama modul tidak dikarang.** Ia diambil dari nama yang disebut Work Owner, dan untuk modul
berikutnya **akan disebutkan di prompt**. Modul kerangka yang memang tidak punya nama bisnis —
`auth`, `portal`, `platform`, `spa` — tidak berubah.

**Yang berubah pada sesi ini:**

| Sebelum | Sesudah |
|---|---|
| `internal/bankaccount/**` · `bankaccount.go` | `internal/masterrekening/**` · `masterrekening.go` |
| `internal/claimstatus/**` · `claimstatus.go` | `internal/masterstatus/**` · `masterstatus.go` |
| paket `bankaccounthttp` · `claimstatushttp` | paket `masterrekeninghttp` · `masterstatushttp` |
| awalan galat `"bankaccount: …"` · `"claimstatus: …"` | `"masterrekening: …"` · `"masterstatus: …"` |
| `src/modules/bank-account/**` | `src/modules/master-rekening/**` |
| `src/modules/claim-status/**` | `src/modules/master-status-klaim/**` |
| `BankAccountPage.tsx` | `AccountPage.tsx` — komponen memakai nama **tipe domain**, bukan nama modul |

**Konsekuensi yang diterima secara sadar:** satu jalur berkas dapat memuat dua bahasa —
`internal/masterrekening/repo/sqlstore/account.go`. Yang kiri nama modul, yang kanan isi modul,
dan keduanya memang punya pembaca yang berbeda.

**Dampak ke Steering:** `08-TECHNICAL-STRATEGY.md` §4.1 (tabel Penamaan + bagian baru) ·
`CLAUDE.md` dan `STEERING.md` dibangun ulang · `claim-pnc/docs/peta-penamaan.md`
