# Interview History — Discovery Claim PNC

Catatan seluruh proses interview dengan pemilik project/bisnis. Setiap sesi memuat
**Pertanyaan · Jawaban · Kesimpulan**.

| Sesi | Tanggal | Fokus | Hasil |
|---|---|---|---|
| [Sesi 1](#sesi-1--2026-09-07--discovery-awal) | 2026-09-07 | Discovery awal — arah teknis & scope | D-01 … D-19 |
| [Sesi 2](#sesi-2--2026-09-07--grilling) | 2026-09-07 | Grilling — menekan asumsi arsitektur | D-20 … D-30 |
| [Sesi 3](#sesi-3--2026-09-08--discovery-brd) | 2026-09-08 | Discovery BRD — celah yang tidak ada di Steering | I-01 … I-06 |

> **Catatan kelengkapan.** Transkrip penuh Sesi 1 dan Sesi 2 — termasuk pilihan yang ditawarkan,
> kutipan jawaban verbatim, dan keputusan akhirnya — ada di
> [`Steering/00-DECISION-LOG.md`](Steering/00-DECISION-LOG.md). Berkas ini merangkumnya agar riwayat
> interview terbaca sebagai satu alur, lalu memuat **Sesi 3 secara lengkap** karena sesi itu
> dijalankan pada sesi kerja ini dan belum tercatat di manapun.

---

## Sesi 1 — 2026-09-07 — Discovery Awal

Dijalankan setelah pembacaan 2.167 rule XML selesai. 19 pertanyaan, masing-masing disertai
empat pilihan dan satu rekomendasi analis.

| ID | Pertanyaan | Jawaban | Kesimpulan |
|---|---|---|---|
| D-01 | Nasib database POOLDATA/GENERAL yang dipakai bersama sistem lain? | *(Other)* Seluruhnya pindah ke PostgreSQL, dibuat PostgreSQL-friendly; sementara masih Oracle | Target akhir PostgreSQL untuk **seluruh** data. Desain PostgreSQL-first, runtime sementara Oracle. Tanggal cutover DB belum ditentukan |
| D-02 | Perlakuan stored procedure Oracle? | Ditulis ulang jadi logika Go | Database jadi penyimpanan murni. **Konsekuensi terbuka:** source-nya tidak ada di export → R-01 |
| D-03 | Scope migrasi — GCNMFW saja atau termasuk GISFW? | *(Other)* Yang ada di export saja; GISFW sedang dikembangkan tim lain | Scope = seluruh rule di export. GISFW jadi dependensi eksternal |
| D-04 | Batas kepemilikan data polis? | Claim PNC menyimpan **snapshot** saat klaim dibuat | Klaim kebal perubahan polis setelah registrasi, dan tidak bergantung ketersediaan sistem tim lain saat runtime |
| D-05 | Strategi cutover? | **Strangler Fig** — modul bertahap, dua sistem paralel | Butuh strategi berbagi data selama masa paralel → diangkat lagi di D-21 |
| D-06 | Cara menangani dua dialek SQL? | *(Other)* PostgreSQL-friendly, tanpa call procedure | Jawaban belum memilih mekanisme teknisnya. **SUPERSEDED** — diangkat ulang dengan bukti terukur di D-20 |
| D-07 | Authentication & authorization? | *(Other)* Login ke API internal HCC/HCQ; hak akses menu di tabel baru milik aplikasi. Lanjutan: HCC/HCQ mengembalikan **profil lengkap** (NIK, nama, cabang, jabatan, email) | Autentikasi didelegasikan, otorisasi dimiliki aplikasi. Tabel peran/izin belum ada, dirancang dalam project ini |
| D-08 | Deployment target? | On-premise VM / bare metal | Tanpa asumsi Kubernetes. Strategi deployment dan monitoring harus realistis untuk VM |
| D-09 | Latar belakang tim pengembang? | Tim internal pemegang Pega, belum terbiasa Go/JS modern | **Batasan utama seluruh keputusan teknis.** Utamakan learning curve landai; Steering wajib preskriptif; hindari abstraksi berat |
| D-10 | Volume data dan beban puncak? | Ribuan klaim/bulan, historis puluhan juta baris | Profil **data besar, konkurensi rendah** (200–300 DAU). Optimasi ke volume data, bukan ke request/detik |
| D-11 | Seberapa penting laporan? | *(Other)* Buat script sendiri untuk PDF/Excel/CSV | Engine dokumen dibuat sendiri di Go. Tidak pakai engine Pega maupun tools BI |
| D-12 | Ada pemakaian dari lapangan/HP? | Mayoritas desktop, surveyor kadang tablet/HP | Frontend **desktop-first namun responsive**. Layar surveyor wajib nyaman di HP dan tahan jaringan lambat |
| D-13 | Ekspektasi UI/UX dibanding Pega? | **Tiru alur & tata letak Pega** | User tidak perlu belajar ulang. Konsekuensi: mewarisi UX padat Pega; frontend wajib kuat pada grid kompleks dan form panjang |
| D-14 | Apa yang menentukan jumlah level komite? | **Kombinasi nilai klaim dan jenis bisnis** | Matriks (nilai × jenis bisnis) jadi **master data**, bukan kode. Ambang di kode lama = nilai awal yang harus divalidasi ulang. **Matriks lengkap masih terbuka** |
| D-15 | Perlakuan nilai hardcode (email, user ID, ambang, hostname)? | **Semua jadi konfigurasi / master data** | **Tidak ada nilai bisnis yang boleh di-hardcode.** Konsekuensi: butuh modul Master Data + layarnya. Blok `// TESTING` yang menimpa email UW produksi **tidak dibawa** |
| D-16 | Penyimpanan dokumen? | Tetap pakai API storage internal (app13/app8) | DB hanya simpan metadata. Tiga mekanisme lama disatukan jadi satu jalur. Ketersediaan dokumen bergantung tim lain |
| D-17 | Ada proses terjadwal/batch? | Ya, ada — **tapi belum tahu detailnya** | Sistem baru butuh scheduler. **Daftar job tidak diketahui**; export tidak memuat Rule-Agent → R-02 |
| D-18 | Empat konsep status — benar berbeda atau duplikat? | **Empat hal berbeda, semuanya dipertahankan** | Dimodelkan sebagai konsep terpisah dengan nama yang tidak bisa tertukar. **Arti kode 1142–1151 masih terbuka** → R-06 |
| D-19 | Arti singkatan domain? | *(Other)* Penjelasan 12 singkatan langsung dari pemilik bisnis | `CONTEXT.md` jadi ubiquitous language wajib. **Temuan paling menentukan: Non-MBU = PNC** — aplikasi ini menangani klaim Non-Motor, mengoreksi asumsi awal |

**Kesimpulan Sesi 1.** Arah teknis dan batas scope tertutup. Tetapi tiga jawaban meninggalkan
lubang yang belum terlihat saat itu: D-05 belum menjawab bagaimana dua sistem berbagi data,
D-06 belum memilih mekanisme dialek, dan tidak ada pertanyaan NFR sama sekali. Ketiganya
menjadi alasan Sesi 2.

---

## Sesi 2 — 2026-09-07 — Grilling

Dijalankan dengan skill `mattpocock-skills:grilling`. Prinsip yang dipakai: **fakta dicari
sendiri, keputusan diserahkan pemilik project** — 652 rule SQL dianalisis dan dikategorikan
lebih dulu, sehingga pertanyaan diajukan dengan volume terukur, bukan dengan "bagaimana menurut
Bapak".

| ID | Pertanyaan | Jawaban | Kesimpulan |
|---|---|---|---|
| D-20 | Mekanisme menangani dua dialek SQL? *(disertai tabel volume: `TO_CHAR` 411×, `JSON_VALUE` 195×, DB Link 64×, `TRUNC` 150×, `NVL` 100×, `ROWNUM` 68×)* | **Satu set SQL portabel** — *berbeda dari rekomendasi saya (repository per dialek)* | Satu set SQL untuk Oracle 19c dan PostgreSQL 17+, dengan **tiga pengecualian terkelola**: pemformatan tanggal/angka dikeluarkan ke Go (menghapus 411 `TO_CHAR` sekaligus memperbaiki bug senyap pengurutan tanggal), generator nomor klaim, dan paginasi diseragamkan |
| D-21 | Bagaimana Pega dan Go berbagi data selama paralel? | *(Other)* Satu database; data yang kini di tabel Pega dipindah ke tabel baru — tabelnya belum ada | Satu database bersama. **Aturan mengikat: satu tabel hanya boleh ditulis satu sistem. Tidak ada sinkronisasi dua arah.** Terungkap **116 query membaca tabel klaim milik engine Pega** yang harus digantikan |
| D-22 | Penomoran klaim dan data historis? | *(Other)* Format `PNCN-xxxx` dari sequence `POOLDATA.CLAIM_NO_NONPEGA_SEQ` | Lebih baik dari rekomendasi awal saya: prefix `PNCN` membuat klaim sistem baru **langsung bisa dibedakan** dari warisan Pega (`PNC-xxxx`) tanpa tabel pemetaan — sangat berharga selama masa paralel panjang |
| D-23 | Frontend mana? *(disertai perbandingan 8 alternatif)* | **React + TypeScript + Vite + TanStack Table/AG Grid** — *berbeda dari rekomendasi saya (Vue 3 + PrimeVue)* | SPA murni disajikan sebagai berkas statis oleh binary Go; tanpa runtime Node.js di production. **Mitigasi wajib:** React tidak opinionated dan tim di D-09 belum terbiasa JS modern, sehingga konvensi ditetapkan **mengikat**, bukan anjuran |
| D-24 | Versi PostgreSQL target? | **PostgreSQL 17 atau lebih baru** | **Persyaratan teknis mengikat, bukan preferensi.** PostgreSQL 17 mendukung `JSON_TABLE`/`JSON_VALUE` bersintaks sama dengan Oracle, membuat 222 query JSON portabel apa adanya. Di 16 ke bawah, D-20 **tidak bisa dijalankan** |
| D-25 | Pengganti DB Link? | Ganti dengan pemanggilan API ke sistem pemilik data | Kopling tersembunyi antar database dihapus, diganti kontrak eksplisit. **Risiko: API-nya kemungkinan besar belum ada** → R-03 |
| D-26 | Konsep penugasan apa yang dipertahankan? | **Worklist (per orang) dan Workbasket (antrean bersama)** | Pembagian ini terbukti dipakai nyata di `Register_Flow`, bukan warisan kosong. Mengubahnya akan mengubah cara kerja user, sedangkan D-13 menetapkan alur tetap sama |
| D-27 | Kapan aplikasi harus bisa dipakai? | **24/7** | Konsekuensi mengikat: minimal 2 instance di belakang load balancer, aplikasi **wajib stateless**, migrasi skema **wajib backward-compatible**, butuh health check dan graceful shutdown |
| D-28 | Ada kewajiban jejak audit dan retensi? | **Ya — ada kewajiban** | Jejak audit **dirancang sejak awal, bukan ditambal**. Append-only: siapa, kapan, nilai sebelum, nilai sesudah. **Lama retensi masih terbuka** → ke Compliance |
| D-29 | Berapa data boleh hilang dan berapa lama sistem boleh mati? | Ikuti standar backup data center Sinarmas | RPO/RTO mengikuti kebijakan korporat. **Catatan yang diangkat:** standar backup menjawab *pemulihan bencana*, sedangkan 24/7 menjawab *operasional harian* — keduanya berbeda dan selisihnya harus diperiksa ke tim infra |
| D-30 | Kapan project mulai dan selesai? | Awal September 2026 – akhir September 2026. Lanjutan: *"Memang seluruh aplikasi, jadwal sudah ditetapkan manajemen"*; *"semua modul sudah harus pindah"* | **Keberatan disampaikan beserta angkanya** (74 harness, 269 section, 15.063 step, 652 rule SQL, 64 procedure tanpa source, 56 laporan, tim belajar 3 teknologi sekaligus; perkiraan **9–15 bulan tim 5–6 orang**). Pemilik project menegaskan target tetap. Keputusan dihormati; risiko dicatat sebagai **R-05** dengan dasar angka |

**Kesimpulan Sesi 2.** Grilling menemukan empat hal yang tidak akan muncul tanpa ditekan:

1. **Kontradiksi terungkap.** D-20 memilih SQL portabel sementara D-22 menetapkan sequence khas
   Oracle. Pertentangannya disampaikan terbuka dan diselesaikan sebagai pengecualian terkelola
   yang diisolasi di satu berkas.
2. **Prasyarat tersembunyi ditemukan.** "SQL portabel" ternyata **hanya bisa dijalankan bila
   PostgreSQL 17+**. Tanpa grilling, ini akan tertulis di Steering sebagai janji yang tidak bisa
   ditepati, dan baru ketahuan saat implementasi.
3. **Blocker sesungguhnya teridentifikasi.** Penghalang portabilitas terbesar bukan `ROWNUM`
   (sepele — 35 rule sudah pakai `FETCH FIRST`), melainkan **64 pemakaian DB Link** yang tidak
   punya padanan di PostgreSQL sama sekali.
4. **Empat pertanyaan NFR yang awalnya terlalu abstrak** diperjelas setelah pemilik project
   meminta klarifikasi, lalu dipecah per topik — menghasilkan D-27, D-28, dan D-29.

---

## Sesi 3 — 2026-09-08 — Discovery BRD

Dijalankan pada sesi kerja ini. **Enam pertanyaan, dua ronde.**

**Cara pertanyaan dipilih.** Saya membaca seluruh 23 berkas Steering lebih dulu, lalu menyaring:
setiap hal yang **sudah terjawab** di Steering tidak ditanyakan lagi, dan setiap hal yang
**bisa saya turunkan sendiri** dari Steering (scope, out-of-scope, assumptions, constraints,
process flow, data flow) juga tidak ditanyakan. Yang tersisa hanya hal yang (a) diperlukan BRD,
(b) tidak ada di Steering, dan (c) **hanya pemilik project yang tahu**. Enam pertanyaan inilah
sisanya.

Ronde diurutkan berdasarkan ketergantungan: pertanyaan tentang *acceptance* sengaja ditunda ke
ronde 2, karena bentuk acceptance bergantung pada apa yang lebih dulu ditetapkan sebagai
*ukuran keberhasilan*.

### Ronde 1

#### I-01 · Pendorong bisnis utama migrasi

**Pertanyaan.** Apa pendorong bisnis utama migrasi ini? `Steering/STEERING.md` §1.6 hanya menyebut
modernisasi platform — tetapi tidak menjelaskan kenapa tenggatnya akhir September 2026.

**Pilihan yang ditawarkan** *(boleh pilih lebih dari satu)*:
1. Biaya lisensi Pega
2. Lisensi / dukungan vendor Pega berakhir
3. Kemandirian teknologi & maintainability
4. Arahan strategis manajemen/grup

**Jawaban:** **Kemandirian teknologi & maintainability** — hanya ini, tiga lainnya tidak dipilih.

**Kesimpulan.** Pendorongnya adalah keinginan lepas dari ketergantungan platform berlisensi dan
vendor, agar tim dapat mengembangkan sendiri tanpa keterbatasan model rule Pega.

Yang lebih penting adalah **apa yang tidak dipilih**, dan ini menghasilkan temuan baru:

> **Tidak ada tenggat eksternal.** Tidak ada lisensi yang berakhir, tidak ada dukungan vendor
> yang habis, tidak ada penalti biaya, dan bukan arahan top-down grup. Karena itu target akhir
> September 2026 (D-30) adalah **target internal manajemen** — dan penjadwalan ulang sepenuhnya
> berada di dalam kendali manajemen Sinarmas, tanpa konsekuensi kontraktual.
>
> Ini menghilangkan satu-satunya alasan yang akan membenarkan menerima risiko R-05 apa adanya.
> **Tidak** berarti D-30 diabaikan — D-30 sudah ditegaskan ulang dan tetap menjadi dasar
> Migration Strategy — tetapi wajib disampaikan ke manajemen sebagai bahan keputusan.

Konsekuensi lain: karena penghematan lisensi bukan pendorong, **tidak ada angka penghematan
biaya** yang bisa dipakai sebagai justifikasi di BRD. Justifikasinya bersifat kualitatif
(maintainability, kemandirian) — dan itu harus ditulis apa adanya, bukan dikarang.

→ [`BRD.md`](BRD/BRD.md) §2, §3, §20

#### I-02 · Ukuran keberhasilan

**Pertanyaan.** Bagaimana keberhasilan migrasi diukur oleh manajemen? Steering punya target NFR
teknis, tapi belum ada ukuran keberhasilan bisnis.

**Pilihan yang ditawarkan** *(boleh pilih lebih dari satu)*:
1. Kesetaraan fungsional dengan Pega
2. Penghematan biaya lisensi
3. Perbaikan TAT penyelesaian klaim
4. Belum ditetapkan manajemen

**Jawaban:** **Kesetaraan fungsional dengan Pega** — hanya ini.

**Kesimpulan.** Sukses = seluruh modul berjalan di Go dengan hasil identik Pega, sampai Pega
dapat dimatikan. Ukurannya **biner per modul**: modul pindah dan setara, atau belum.

Tiga konsekuensi:

- **Konsisten penuh dengan prinsip P-5** Migration Strategy ("hasil yang benar adalah hasil yang
  sama dengan Pega"). Ukuran keberhasilan dan prinsip migrasi tidak bertentangan.
- **Tidak perlu baseline TAT.** Karena perbaikan TAT bukan ukuran keberhasilan, tidak ada
  kewajiban mengambil baseline TAT Pega sebelum cutover — satu dependensi yang gugur.
- **Menuntut bukti kesetaraan per modul**, bukan sekadar "modul selesai dibangun". Ini menjadikan
  perkakas uji kesetaraan sebagai **item yang harus dibangun**, bukan aktivitas sampingan.

→ [`BRD.md`](BRD/BRD.md) §21, §22 · [`requirement-summary.md`](requirement-summary.md) FR-S8

#### I-03 · Stakeholder

**Pertanyaan.** Siapa stakeholder project ini? Minimal siapa yang menyetujui BRD dan siapa
pemilik bisnisnya — Steering hanya memuat 22 peran *pengguna aplikasi*, bukan stakeholder
*project*.

**Pilihan yang ditawarkan:**
1. Saya sebutkan sendiri (nama/jabatan)
2. Pakai jabatan generik saja
3. Turunkan dari peran di STEERING.md

**Jawaban:** **Pakai jabatan generik saja.**

**Kesimpulan.** BRD memakai penyebutan peran tanpa nama orang: "Sponsor Project", "Business
Owner Claim PNC", "Tim DBA", "Tim Pega", "Tim Infra", "Tim Compliance", "Tim GISFW".

Konsekuensi yang harus disadari: **kolom persetujuan di BRD §8 harus diisi manual** sebelum
dokumen dipresentasikan, karena tanpa nama tidak ada yang benar-benar terikat pada komitmennya.
Ini dicatat sebagai catatan di BRD, bukan disembunyikan.

→ [`BRD.md`](BRD/BRD.md) §8

#### I-04 · Bahasa dokumen

**Pertanyaan.** BRD ini ditulis dalam bahasa apa?

**Jawaban:** **Bahasa Indonesia** (pilihan yang saya rekomendasikan).

**Kesimpulan.** Konsisten dengan 23 dokumen Steering yang sudah ada dan dengan istilah domain di
`CONTEXT.md`. Menghindari risiko istilah asuransi Indonesia (akseptasi, tertanggung, penjaminan,
koasuransi) bergeser makna saat diterjemahkan.

### Ronde 2

#### I-05 · Status sepuluh tindakan hari pertama

**Pertanyaan.** Sepuluh tindakan hari pertama di Risk Analysis — source 64 procedure ke DBA,
export Rule-Agent, DDL, isi `V_STS_CLAIM`, 3 router + 45 activity, 6 API DB Link, isi
`EMAILKOMITE`, retensi audit, pembekuan Pega, pelatihan tim — sudah sampai mana?

**Pilihan yang ditawarkan:**
1. Belum ada yang diminta
2. Sudah diminta, menunggu jawaban
3. Sebagian sudah diterima
4. Saya belum tahu statusnya

**Jawaban:** **Sudah diminta, menunggu jawaban.**

**Kesimpulan.** Permintaan sudah dikirim ke pihak-pihak terkait, tetapi **belum ada satu pun
artefak yang diterima**.

Ini menentukan verdict kesiapan: **NOT READY — BLOCKED ON EXTERNAL.** Tim sudah melakukan
bagiannya; yang menahan sepenuhnya berada di luar kendali tim pengembang.

Karena permintaan sudah terkirim, tindakan yang tepat **bukan** mengirim ulang, melainkan:
menetapkan **tanggal komitmen** dari masing-masing pihak, dan menetapkan **jalur eskalasi** bila
tanggal itu tidak dipenuhi. Tanpa tanggal komitmen, "sudah diminta" dan "belum diminta" punya
dampak jadwal yang sama persis.

→ [`migration-readiness.md`](migration-readiness.md) §2, §7

#### I-06 · Gerbang penerimaan (acceptance)

**Pertanyaan.** Siapa yang menyatakan sistem baru sudah benar dan boleh menggantikan Pega per
modul? Testing Strategy sudah mengatur uji kesetaraan teknis, tapi belum menetapkan siapa yang
menandatangani.

**Pilihan yang ditawarkan:**
1. UAT oleh user bisnis per modul
2. Uji kesetaraan otomatis + persetujuan business owner
3. Keduanya — otomatis lalu UAT
4. Belum ditetapkan

**Jawaban:** **Keduanya — otomatis lalu UAT.**

**Kesimpulan.** Cutover per modul melewati **dua gerbang berurutan**:

| Gerbang | Isi | Penanggung jawab |
|---|---|---|
| **1 — Kesetaraan otomatis** | Perbandingan hasil Go vs Pega atas data historis yang sama. Selisih apa pun harus dijelaskan sebagai bug atau sebagai perbaikan yang sudah diputuskan (P-5) | Tim pengembang |
| **2 — UAT user bisnis** | User bisnis dari peran yang memakai modul itu menguji dan menyetujui | User bisnis per peran + Business Owner |

Pilihan paling aman untuk data uang klaim — dan sekaligus paling lambat. Dua konsekuensi yang
**belum tercatat di Steering** dan karena itu menjadi tambahan dari sesi ini:

- **Waktu user bisnis harus dialokasikan resmi.** UAT per modul untuk ~20 inbox berbasis peran
  dan 5 layar transaksi utama menuntut ketersediaan user yang juga sedang menjalankan operasional
  harian. Ini **constraint baru** di BRD §19, bukan detail pelaksanaan.
- **Perkakas uji kesetaraan menjadi item yang harus dibangun.** Ia bukan aktivitas sampingan:
  ia butuh kemampuan menjalankan permintaan yang sama ke Pega dan ke Go atas data historis yang
  sama, lalu membandingkan hasilnya.

→ [`BRD.md`](BRD/BRD.md) §19, §21 · [`requirement-summary.md`](requirement-summary.md) FR-S8

---

## Kesimpulan seluruh interview

**Yang tertutup.** Arah teknis (D-01…D-08, D-20…D-27), batas scope (D-03, D-04), batasan tim
(D-09), profil beban (D-10), ekspektasi UI (D-12, D-13), bahasa domain (D-19), pendorong bisnis
(I-01), ukuran keberhasilan (I-02), dan gerbang penerimaan (I-06).

**Yang tetap terbuka setelah tiga sesi — dan alasannya.** Enam hal, dan **tidak satu pun dapat
ditutup dengan interview lebih lanjut kepada pemilik project**, karena jawabannya tidak ada
padanya:

| Terbuka | Ada pada | Ref |
|---|---|---|
| Isi 64 procedure & function database | DBA | R-01 |
| Daftar job terjadwal | Tim Pega | D-17 · R-02 |
| Arti kode status 1142–1151 | DBA (`V_STS_CLAIM`) | D-18 · R-06 |
| Matriks penjenjangan komite | DBA (`POOLDATA.EMAILKOMITE`) + tim bisnis | D-14 |
| Kontrak 6 API pengganti DB Link | Tim pemilik sistem | D-25 · R-03 |
| Aturan 3 router penugasan | Tim Pega | R-04 |
| Lama retensi audit | Compliance | D-28 |
| Angka RPO/RTO | Tim infra | D-29 |

**Karena itu interview dihentikan di sini, bukan karena kehabisan pertanyaan.** Melanjutkan
interview kepada pemilik project akan menghasilkan tebakan, bukan jawaban — dan itu bertentangan
langsung dengan ketentuan "jangan membuat asumsi". Penutupan delapan item di atas dipindahkan
menjadi tindakan berjadwal dengan pihak yang tepat, tercatat di
[`migration-readiness.md`](migration-readiness.md) §7.
