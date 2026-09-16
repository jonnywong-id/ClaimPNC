# Requirement Summary — Claim PNC

Ringkasan final seluruh requirement yang telah tervalidasi, dengan penomoran agar dapat dirujuk
dari [`BRD.md`](BRD/BRD.md) dan dari backlog implementasi.

| | |
|---|---|
| **Tanggal** | 2026-09-08 · **disinkronkan 2026-09-14** |
| **Sumber** | [`Steering/STEERING.md`](Steering/STEERING.md) v2.0 · 24 berkas [`Steering/`](Steering/) · **29 ADR** di [`ADR/`](ADR/) · Sesi 3 interview |
| **Status validasi** | **162 requirement** — 137 tervalidasi · **25 belum dapat difinalkan** karena menunggu 10 artefak dari pihak luar |

> **Disinkronkan dengan Steering v2.0 (2026-09-14).** Angka dan rumusan yang dikoreksi ditandai
> di tempatnya masing-masing. Daftar lengkap perubahan ada di
> [`Steering/21-RIWAYAT-REVISI.md`](Steering/21-RIWAYAT-REVISI.md). **Status validasi di atas
> belum dihitung ulang** — ia hasil evaluasi 2026-09-08, dan penghitungan ulangnya adalah
> pekerjaan tersendiri.

**Penanda status per requirement:**

`✅` tervalidasi dan siap diimplementasikan · `⚠️` tervalidasi tetapi **detailnya menunggu artefak pihak luar** · `🆕` tambahan dari Sesi 3, belum ada di Steering

---

## 1. Prinsip yang mengikat seluruh requirement

| ID | Prinsip | Sumber |
|---|---|---|
| P-1 | Satu tabel hanya boleh **ditulis** satu sistem selama masa paralel. **Tidak ada sinkronisasi dua arah** | D-21 |
| P-2 | Batas modul mengikuti batas kepemilikan tabel — urutan migrasi tidak bebas | D-21 |
| P-3 | Klaim yang sedang berjalan **tidak berpindah sistem di tengah jalan**. `PNC-xxxx` diselesaikan di Pega; `PNCN.YY.xxxx` dimulai di Go | D-22, D-71 |
| P-4 | Migrasi skema **selalu backward-compatible**. Hapus kolom dua tahap; ganti nama kolom tidak pernah langsung | D-27 |
| P-5 | **Perilaku dipertahankan lebih dulu, diperbaiki kemudian.** Hasil yang benar adalah hasil yang sama dengan Pega — kecuali empat perbaikan yang diputuskan eksplisit: hardcode → konfigurasi (D-15), blok `// TESTING` dihapus (D-15), zona waktu (F-5), penamaan (D-19) | §7 |
| P-6 | **Tidak ada nilai bisnis yang di-hardcode.** Seluruh ambang, penerima notifikasi, pemetaan peran, dan endpoint adalah master data yang dapat diubah tanpa deploy | D-15 |
| P-7 | Bahasa domain [`CONTEXT.md`](Steering/CONTEXT.md) dipakai konsisten di nama tabel, struct Go, endpoint API, label UI, dan dokumentasi | D-19 |

---

## 2. Functional Requirement

### 2.1 Modul fondasi — wajib selesai lebih dulu

| ID | Requirement | Ukuran | Status |
|---|---|---|---|
| **FR-F1** | **Kerangka Aplikasi** — struktur folder, konfigurasi tiga lapis, logging terstruktur, error handling, health check endpoint, graceful shutdown, penyajian SPA sebagai berkas statis | Kecil | ✅ |
| **FR-F2** | **Akses Data** — koneksi, connection pool (pool terpisah untuk laporan), transaksi, seam Repository, satu set SQL portabel Oracle 19c + PostgreSQL 17+, migrasi skema versioned | Sedang | ✅ |
| **FR-F3** | **Identitas & Akses** — login via HCC/HCQ, penerbitan session/token milik aplikasi, tabel peran & izin menu **yang belum ada**, middleware otorisasi per endpoint | Sedang | ✅ |
| **FR-F4** | **Master Data** — **≥29 kelompok master** + layar pengelolanya. Menghapus **seluruh** hardcode: **66 alamat email, 24 user ID, 8 ambang komite**, 3 hostname penentu perilaku | **Besar** | ⚠️ **D-40** — tujuan penyimpanan rahasia belum ditetapkan |
| **FR-F5** | **Waktu & Zona Waktu** — seam Clock, penyimpanan UTC, konversi WIB di **satu** tempat, aturan hari kalender | Kecil | ✅ |

> **FR-F5 kecil tetapi tidak boleh dilewati atau ditunda.** Ia menghapus `+7 jam` manual yang
> tersebar di puluhan tempat di sistem lama. Bila dikerjakan belakangan, seluruh aturan tanggal
> harus ditulis ulang.

### 2.2 Modul bisnis inti

| ID | Requirement | Activity lama | Bergantung | Status |
|---|---|---|---|---|
| **FR-B1** | **Polis & Snapshot** — ambil data polis, bekukan sebagai snapshot saat registrasi | 18 | F-1…F-5 | ✅ |
| **FR-B2** | **Registrasi Klaim** — validasi tanggal, duplikasi, kelengkapan; pembentukan objek dan coverage. **Modul terdalam: 137 step** | 22 | B-1 | ✅ |
| **FR-B3** | **Objek & Coverage** — objek pertanggungan per lini bisnis, coverage, penyebab kerugian, rincian item | 44 | B-2 | ✅ |
| **FR-B4** | **Spreading Reasuransi** — aturan total 100%, Fac Out, Ex-Gratia `OR`→`ORS`, share ASM | bagian dari 69 | B-3 | ✅ |
| **FR-B5** | **Estimasi & Settlement** — Estimasi → Usulan → Akseptasi → Dibayar; salvage; risiko sendiri | 29 | B-3, B-4 | ⚠️ **R-01** |
| **FR-B6** | **Penugasan & Inbox** — Worklist (per orang), Workbasket (antrean bersama), routing, penguncian record | 30 | F-3 | ⚠️ **R-04** |
| **FR-B7** | **Komite** — penjenjangan 1–4 level, matriks nilai × jenis bisnis sebagai master data, jejak persetujuan | 34 | B-5, B-6, F-4 | ⚠️ **D-14, R-01** |
| **FR-B8** | **Survey & Adjuster** — penugasan surveyor, hasil survei, foto lapangan (wajib nyaman di tablet/HP) | 30 | B-6 | ✅ |
| **FR-B9** | **PLA / Pre-DLA / DLA** — pemberitahuan bertahap ke koasuransi/reasuransi | 69 gabungan | B-4, B-5 | ⚠️ **R-01** |
| **FR-B10** | **Akseptasi & Pembayaran** — nomor akseptasi, transfer kasir, status pembayaran, LOD ke tertanggung | 31 | B-5, B-7 | ⚠️ **R-01** |
| **FR-B11** | **RCL / PUCL / Compliance** — penolakan, proses ulang, pemeriksaan kepatuhan, investigator, analyst doctor | 6 + jalur alur | B-6 | ✅ |
| **FR-B12** | **Salvage & Recovery** — barang sisa, balai lelang, pemulihan, virtual account | 26 | B-5 | ⚠️ **R-01** |
| **FR-B13** | **Open Protection** — alur buka proteksi dan penautannya ke klaim (`IsUsedPNC`) | 12 | B-2 | ✅ |
| **FR-B14** | **Receive Document** — penerimaan dokumen fisik, ekspedisi, nomor resi, estimasi tiba | 8 | F-3 | ✅ |

### 2.3 Modul pendukung

| ID | Requirement | Ukuran | Status |
|---|---|---|---|
| **FR-S1** | **Dokumen & Lampiran** — unggah/ambil lewat API storage internal; DB hanya simpan metadata. **Tiga mekanisme penyimpanan lama disatukan menjadi satu jalur** | 54 activity | ✅ |
| **FR-S2** | **Laporan & Export** — 56 laporan + engine PDF/Excel/CSV dibuat sendiri di Go. Export besar **asinkron dan streaming** | 78 activity, 56 RD | ✅ |
| **FR-S3** | **Notifikasi** — seam Notifier **berbasis peristiwa domain**, 5 correspondence, penerima dari master data | 14 activity | ⚠️ **R-07** |
| **FR-S4** | **Integrasi Eksternal** — **21 Connect REST** + 6 API pengganti DB Link + **4 Service REST masuk** | 7 activity + **21 REST** + 4 service | ⚠️ **R-03** |
| **FR-S5** | **Jejak Audit** — pencatatan append-only seluruh perubahan bernilai bisnis | Baru | ✅ |
| **FR-S6** | **Penjadwalan (Scheduler)** — job terjadwal | **Belum terukur** | ⚠️ **R-02** |
| **FR-S7** | **Dashboard & Monitoring Bisnis** — dashboard klaim, TAT, KPI, progres | bagian dari 78 | ✅ |
| **FR-S8** 🆕 | **Perkakas Uji Kesetaraan** — menjalankan permintaan yang sama ke Pega dan ke Go atas data historis yang sama, lalu membandingkan hasilnya dan melaporkan selisihnya | Sedang | ✅ |

> **FR-S8 adalah tambahan dari Sesi 3 dan belum ada di
> [`Steering/06-MODULE-BREAKDOWN.md`](Steering/06-MODULE-BREAKDOWN.md).** Ia menjadi modul karena
> I-02 menetapkan ukuran keberhasilan = kesetaraan fungsional dengan Pega, dan I-06 menetapkan uji
> kesetaraan otomatis sebagai **gerbang pertama cutover setiap modul**. Tanpa perkakasnya, ukuran
> keberhasilan yang dipilih manajemen tidak dapat dibuktikan. Module Breakdown perlu diperbarui.

### 2.4 Frontend

| ID | Requirement | Ukuran | Status |
|---|---|---|---|
| **FR-U1** | **Kerangka SPA** — routing, layout, autentikasi, penanganan error, state global | Sedang | ✅ |
| **FR-U2** | **Pustaka Komponen** — tabel baku (**median 6 kolom**, paginasi server-side), form baku, unggah berkas, pemilih tanggal | **Sedang** | ✅ |
| **FR-U3** | **Layar Inbox** — **26** harness bernama *Inbox* | Besar | ✅ |
| **FR-U4** | **Layar Transaksi** — registrasi, estimasi, survei, komite, akseptasi | Besar | ✅ |
| **FR-U5** | **Layar Laporan** — 56 laporan + unduhan | Besar | ✅ |
| **FR-U6** | **Layar Master Data** — pengelolaan **≥29 kelompok master** | **Besar** | ✅ |

> **FR-U2 adalah investasi terpenting di frontend.** 268 dari 269 section memakai pola grid yang
> sama. Satu komponen tabel baku yang benar dipakai ratusan kali (*leverage*), dan perbaikan bug
> di satu tempat memperbaiki seluruh layar (*locality*). Melewatkannya berarti 268 implementasi
> tabel yang berbeda-beda — persis kegagalan yang harus dicegah pada tim di D-09.

### 2.5 Alur kerja yang wajib didukung

| ID | Requirement | Status |
|---|---|---|
| **FR-W1** | **Register Flow** — 23 shape, 13 assignment, 7 decision. Percabangan PA / Travel / Non-MBU, lalu Compliance / PUCL / Analyst Doctor / RCL Dokter | ✅ |
| **FR-W2** | **Transisi lateral wajib diizinkan.** Sebelas *ticket* memungkinkan lompatan langsung ke tahap mana pun tanpa melewati urutan. Model status **tidak boleh** memaksakan urutan kaku | ✅ |
| **FR-W3** | **Open Protection Flow** — Input → Akseptasi → diterima/ditolak; penautan ke klaim | ✅ |
| **FR-W4** | **Receive Document Flow** — menghasilkan `RCV_ID` yang ditautkan ke klaim | ✅ |
| **FR-W5** | **Komite Flow** — berjenjang sampai 4 level; tiap putaran menaikkan jenjang dan mereset status persetujuan, lalu meneruskan ke antrean berikutnya | ⚠️ **D-14** |
| **FR-W6** | **Empat konsep status dipertahankan terpisah** dengan nama yang tidak bisa tertukar: Status Proses, Status Klaim, Flag Klaim, Status Posisi Progres | ⚠️ **R-06** |

### 2.6 Peran pengguna yang wajib didukung

22 access group mengendalikan **47 harness target unik** lewat **51 item menu aksi**. Perlu dicatat: **11 dari 47 harness itu tidak ada di export** (`K-33`). Seluruhnya dipertahankan:

PncAdmin · PncManagerAdmin · PncPICTeknik · PNCKomiteTeknik · PNCKomite · CaseManager ·
PncRCLPUCL · PncAnalystDoctor · PncComplience · PncInvestigator · PNCSurveyor · PncPLADLA ·
PncReceive · PncManagerReceive · PncCollection · PncOPCGeneral · PNCServiceCenter · TreatyIn ·
ViewClaimPNC · PNCReportClaimInternal · PNCReportClaimEksternal · Administrators

| ID | Requirement | Status |
|---|---|---|
| **FR-R1** | Otorisasi diperiksa **di setiap endpoint backend** — tidak boleh bergantung pada penyembunyian menu seperti sistem lama | ✅ |
| **FR-R2** | Akses data medis dibatasi peran Analyst Doctor dan RCL Dokter | ✅ |
| **FR-R3** | `PNCReportClaimEksternal` mendapat akses laporan **terbatas** | ✅ |

---

## 3. Non Functional Requirement

### 3.1 Profil beban — dasar seluruh keputusan NFR

| ID | Requirement | Nilai |
|---|---|---|
| **NFR-01** | Pengguna aktif harian | 200–300 |
| **NFR-02** | Klaim baru | Ribuan per bulan |
| **NFR-03** | Data historis | **Puluhan juta baris** |
| **NFR-04** | Lingkungan | 2 VM on-premise di belakang load balancer |

> **Profilnya: data besar, konkurensi rendah.** Optimasi diarahkan ke **volume data**, bukan ke
> jumlah permintaan per detik. Salah membaca ini akan membuat tim mengoptimalkan hal yang bukan
> hambatan.

### 3.2 Performa — target persentil 95

| ID | Operasi | Target |
|---|---|---|
| **NFR-05** | Buka layar sederhana | < 1 detik |
| **NFR-06** | Inbox dan pencarian | < 3 detik **terhadap data produksi penuh** |
| **NFR-07** | Simpan registrasi klaim | < 3 detik termasuk seluruh validasi |
| **NFR-08** | Perhitungan spreading | < 1 detik |
| **NFR-09** | Laporan interaktif | < 10 detik |
| **NFR-10** | Export besar PDF/Excel/CSV | **Asinkron** — tidak menahan pengguna, diberi tahu saat selesai |
| **NFR-11** | Pemanggilan sistem eksternal | Batas waktu 30 detik; kegagalan **ditangani**, bukan menggantung |

### 3.3 Aturan performa yang mengikat

| ID | Requirement |
|---|---|
| **NFR-12** | Paginasi **selalu** server-side. Tidak pernah mengambil seluruh baris lalu memotongnya |
| **NFR-13** | **Paginasi keyset** untuk inbox dan pencarian. **Premis lama dicabut** (`K-31`): `OFFSET` **nol kemunculan** di seluruh export, sehingga NFR ini **bukan perbaikan atas paginasi yang lambat** melainkan **penambahan kemampuan** — sistem lama memotong hasil di `pyMaxRecords=500`, bukan memaginasinya |
| **NFR-14** | Tidak ada `COUNT(*)` atas tabel besar sebagai bagian permintaan biasa. Frontend memakai pola "muat lebih banyak" |
| **NFR-15** | Tidak ada query di dalam perulangan — muat sekaligus per batch |
| **NFR-16** | `SELECT` menyebutkan kolom. Mengambil kolom `CLOB` yang tidak dipakai sangat mahal |
| **NFR-17** | Export dan laporan besar **streaming** — memori tetap datar berapa pun jumlah barisnya |
| **NFR-18** | Pool koneksi **terpisah** untuk laporan |
| **NFR-19** | Setiap query baru wajib diperiksa rencana eksekusinya terhadap **data sebesar produksi** sebelum merge |

### 3.4 Ketersediaan

| ID | Requirement | Nilai |
|---|---|---|
| **NFR-20** | Ketersediaan | **24/7**, termasuk saat deployment |
| **NFR-21** | Aplikasi **wajib stateless** — tidak ada session di memori satu instance | Mengikat |
| **NFR-22** | Rolling deployment tanpa downtime; minimal 2 instance | Mengikat |
| **NFR-23** | Migrasi skema **wajib backward-compatible** — versi lama dan baru berjalan bersamaan terhadap skema yang sama | Mengikat |
| **NFR-24** | Health check endpoint + graceful shutdown agar request berjalan tidak terputus | Mengikat |
| **NFR-25** | RPO / RTO | ⚠️ Mengikuti standar korporat — **angkanya belum ada** (D-29) |

> **Catatan yang harus diangkat ke tim infra:** standar backup korporat menjawab *pemulihan saat
> bencana*, sedangkan NFR-20 menjawab *operasional harian*. Keduanya berbeda, dan perlu
> dipastikan standar yang ada memang sejalan dengan tuntutan 24/7.

### 3.5 Auditabilitas

| ID | Requirement | Status |
|---|---|---|
| **NFR-26** | Setiap perubahan bernilai bisnis tercatat permanen: **siapa, kapan, nilai sebelum, nilai sesudah** | ✅ |
| **NFR-27** | Cakupan minimal: nilai estimasi, nilai settlement, akseptasi, keputusan komite, penolakan (RCL), proses ulang (PUCL), perubahan status klaim, pembayaran | ✅ |
| **NFR-28** | Data audit **append-only**, ditegakkan lewat **hak akses database** — aplikasi hanya diberi `INSERT` dan `SELECT`, sehingga tidak ada jalur di dalam aplikasi yang bisa menghapus jejaknya sendiri | ✅ |
| **NFR-29** | Lama retensi data audit | ⚠️ **Belum ditentukan** — ke Compliance (D-28) |

### 3.6 Maintainability

| ID | Requirement | Batas |
|---|---|---|
| **NFR-30** | Panjang fungsi | ~80 baris |
| **NFR-31** | Panjang berkas | ~400 baris Go · ~200 baris komponen React |
| **NFR-32** | Pelanggaran aturan ketergantungan antar lapisan | **Nol** — merge diblokir, ditegakkan `depguard` |
| **NFR-33** | Cakupan test aturan bisnis | Seluruh aturan terdokumentasi punya test bernama kalimat bisnis |
| **NFR-34** | Penambahan dependensi pihak ketiga | Butuh alasan tertulis |

### 3.7 Kompatibilitas & antarmuka

| ID | Requirement | Status |
|---|---|---|
| **NFR-35** | Frontend **desktop-first namun responsive** | ✅ |
| **NFR-36** | Layar surveyor — unggah foto dan input hasil survei — wajib nyaman di tablet/HP dan **tahan jaringan lambat** | ✅ |
| **NFR-37** | Alur kerja, urutan langkah, penempatan field, dan tata letak layar **mengikuti Pega** agar user tidak perlu pelatihan ulang | ✅ |
| **NFR-38** | Pesan kesalahan untuk pengguna **berbahasa Indonesia** dan menjelaskan cara memperbaiki | ✅ |

---

## 4. Business Rules

### 4.1 Aturan tanggal

| ID | Aturan | Berlaku untuk |
|---|---|---|
| **BR-01** | Tanggal Kejadian (DOL) harus di dalam periode polis | Semua |
| **BR-02** | DOL boleh sampai **30 hari** setelah polis berakhir | Bonding |
| **BR-03** | Tanggal cetak boleh sampai **90 hari** setelah polis berakhir | Travel & PA |
| **BR-04** | Tanggal Lapor ≤ DOL + **7 hari** | Semua **kecuali PA** |
| **BR-05** | Tanggal Terima Dokumen ≤ DOL + **90 hari** | Travel |

### 4.2 Aturan duplikasi

| ID | Aturan |
|---|---|
| **BR-06** | Klaim **ditolak** bila sudah ada klaim lain dengan **Nomor Polis + Objek + Lokasi** yang sama |
| **BR-07** | Untuk **PA**, pengecekan diperketat: **Nomor Polis + Objek + Penyebab Kerugian `12002` + Lokasi** |
| **BR-08** | Pesan penolakan **menyertakan nomor klaim yang sudah ada** |

### 4.3 Aturan reasuransi

| ID | Aturan |
|---|---|
| **BR-09** | **Total spreading wajib 100%**, **toleransi 4 desimal, `99,9999`–`100,0001`** (`D-51`). Bila tidak, submit **ditolak** |
| **BR-10** | Bila ada spreading **Fac Out** (`TreatyType = 10015`), data Fac Offer **wajib ada** |
| **BR-11** | Untuk **Group Panel `003`**, Fac Offer wajib menyertakan Object Name |
| **BR-12** | Bila klaim ditandai **Ex-Gratia**, treaty `OR` (`10001`) **otomatis berubah** menjadi `ORS` (`10007`) |
| **BR-13** | Spreading bertanda `FlagDelete = "1"` **dibuang sebelum** perhitungan |

### 4.4 Aturan nilai dan notifikasi

| ID | Aturan | Status |
|---|---|---|
| **BR-14** | Nilai estimasi **dikonversi ke IDR memakai kurs standar** sebelum dibandingkan dengan ambang apa pun | ✅ |
| **BR-15** | Estimasi **> Rp 1.000.000.000** memicu **Notice of Large Losses** ke Underwriting sesuai Group Panel dan ke jajaran pimpinan | ✅ |
| **BR-16** | Persetujuan komite dipicu ambang **Rp 50.000.000** (umum) dan **Rp 30.000.000** (PA/Travel) — *nilai awal, wajib divalidasi ulang tim bisnis* | ⚠️ **D-14** |
| **BR-17** | Bila premi belum lunas (`AgingAmount > 1`), notifikasi premi tertunggak dikirim | ✅ |
| **BR-18** | Jumlah jenjang komite = jumlah baris matriks; urutan dari kolom `DEGREE`; ambang dari kolom `LIMIT_BOTTOM` | ⚠️ **isi matriks belum ada** |
| **BR-19** | Di setiap tahap, nilai dikurangi **Salvage** dan **Recovery** bila ada, lalu dibagi ke para penanggung sesuai **Spreading** dan **Koasuransi** | ✅ |

> **Ambang `Rp 3.500` untuk server Timor-Leste tidak dibawa sebagai aturan.** Di sistem lama ia
> ditentukan oleh **hostname server**, yang dilarang P-6. Bila aturan ini memang masih berlaku
> secara bisnis, ia menjadi baris master data per entitas — dan itu **harus dikonfirmasi tim
> bisnis**, tidak boleh disalin apa adanya.

### 4.5 Aliran nilai uang klaim

| ID | Aturan |
|---|---|
| **BR-20** | Urutan nilai **mengikat**: Estimasi → Usulan (Propose) → Persetujuan Komite → Akseptasi → Transfer Kasir → Pembayaran |
| **BR-21** | Urutan pemberitahuan **mengikat**: **PLA** (estimasi) → **Pre-DLA** (akan diaksep) → **DLA** (sudah diaksep) |
| **BR-22** | **LOD** ditujukan ke **tertanggung** — berbeda sasaran dari PLA/DLA yang ditujukan ke koasuransi/reasuransi |
| **BR-23** | Akseptasi adalah persetujuan atas **nilai** klaim, **berbeda** dari persetujuan bahwa klaim dijamin (liability) |

---

## 5. Validation Rules

| ID | Aturan validasi | Status |
|---|---|---|
| **VR-01** | Urutan wajib: **DOL ≤ Tanggal Lapor ≤ Tanggal Terima Dokumen ≤ hari ini** | ✅ |
| **VR-02** | DOL, Tanggal Lapor, dan Tanggal Terima Dokumen **tidak boleh melebihi tanggal hari ini** | ✅ |
| **VR-03** | **Penyebab Kerugian wajib diisi**, kecuali lini Travel | ✅ |
| **VR-04** | **Nomor SLIK wajib diisi** untuk SPK / Asuransi Kredit — tidak boleh kosong | ✅ |
| **VR-05** | Nilai klaim **tidak boleh melebihi TSI** | ✅ |
| **VR-06** | Objek **tanpa Coverage dibuang otomatis** dari daftar | ✅ |
| **VR-07** | Bila hubungan tertanggung dipilih "lain-lain" (`7`), **keterangannya wajib diisi** | ✅ |
| **VR-08** | **Polis Deklarasi tidak dapat diklaim**, kecuali lini Aneka | ✅ |
| **VR-09** | `ObjectName` dipotong pada **3.800 karakter** karena batas kolom database | ⚠️ **R-08** — batas sebenarnya tidak diketahui |
| **VR-10** | Validasi sisa TSI (`ValidasiSisaTSI`) | ⚠️ **R-07** — activity tidak ada di export; **aturannya tidak diketahui sama sekali** |
| **VR-11** | **Seluruh kesalahan validasi dikumpulkan dan dikembalikan bersamaan** (HTTP `422`), tidak berhenti pada yang pertama. Ini **kesetaraan perilaku** (P-5), bukan preferensi — form registrasi punya puluhan field | ✅ |

---

## 6. Data Requirement

| ID | Requirement | Status |
|---|---|---|
| **DAT-01** | **Satu database bersama** selama masa paralel. Tidak ada migrasi massal data klaim historis | ✅ |
| **DAT-02** | Data yang kini hidup di **tabel milik engine Pega** dipindahkan ke tabel baru milik aplikasi: `PC_ASM_FW_GCNMFW_WORK` (dibaca **116 rule**), `PC_ASSIGN_WORKLIST` (18), `PC_ASSIGN_WORKBASKET` (6), `PR_OPERATORS` (6), `PC_LINK_ATTACHMENT` (3), `PC_DATA_WORKATTACH` (2), `PR_SYS_LOCKS` (1) | ✅ |
| **DAT-03** | Nomor klaim baru: **`PNCN.YY.xxxx`** dari sequence `POOLDATA.CLAIM_NO_NONPEGA_SEQ` — sintaks di `ADR-0009`. Prefix `ASM-FW-GCNMFW-WORK` **tidak dipakai lagi** | ✅ |
| **DAT-04** | Klaim warisan `PNC-xxxx` **tetap dapat dibaca dan diproses**, hanya tidak lagi dibuat baru | ✅ |
| **DAT-05** | Snapshot polis disimpan sebagai JSON; data klaim sebagai tabel relasional. **Satu data hanya disimpan satu kali** — menutup duplikasi relasional/JSON sistem lama | ✅ |
| **DAT-06** | Pemeriksaan konsistensi data relasional versus JSON dijalankan pada data produksi **sebelum** migrasi; sumber kebenaran ditetapkan eksplisit per jenis data | ⚠️ **R-10** |
| **DAT-07** | Satu set SQL portabel untuk Oracle 19c dan **PostgreSQL 17+** (mengikat), dengan tiga pengecualian: pemformatan tanggal/angka dikeluarkan ke Go, generator nomor klaim, paginasi diseragamkan `OFFSET … FETCH NEXT` | ✅ |
| **DAT-08** | **Tidak ada pemanggilan stored procedure** dari aplikasi. Seluruh logika naik ke Go | ⚠️ **R-01** |
| **DAT-09** | Zona waktu: penyimpanan UTC, konversi WIB di satu tempat. **Konvensi penyimpanan data lama wajib diperiksa terhadap contoh data produksi** sebelum migrasi | ⚠️ **R-12** |
| **DAT-10** | DDL lengkap `POOLDATA`, `DATAPEGA`, `GENERAL` + statistik ukuran tabel | ⚠️ **R-08** — belum diterima |
| **DAT-11** | Arti kode Status Klaim — **33 kode `1134`–`1166`** dari master `V_STS_CLAIM` | ✅ **diterima** — `R-06` tertutup |

---

## 7. Integration Requirement

| ID | Sistem | Arah | Keperluan | Status |
|---|---|---|---|---|
| **INT-01** | **HCC / HCQ** | Masuk | Autentikasi username+password; respons memuat profil lengkap (NIK, nama, cabang, jabatan, email) | ✅ |
| **INT-02** | **Storage Dokumen** (`app13/api`) | Dua arah | Unggah dan ambil dokumen klaim; DB hanya simpan metadata dan referensi | ✅ |
| **INT-03** | **Arsip Dokumen** (`app8/asm-archive`) | Keluar | Injeksi data arsip dokumen klaim | ✅ |
| **INT-04** | **BRI Surf** (`partner.api.bri.co.id`) | Keluar | OAuth token + kirim umpan balik klaim | ✅ |
| **INT-05** | **Konversi Gambar** (`aiimage`) | Keluar | Konversi format AVIF | ✅ |
| **INT-06** | **History Payment** (WebLogic internal) | Masuk | Riwayat pembayaran produksi | ✅ |
| **INT-07** | **Kasir** | Keluar | Data rekening dan permintaan pembayaran | ✅ |
| **INT-08** | **SLIK OJK** | Keluar | Pelaporan regulator untuk lini SPK | ⚠️ **R-07** |
| **INT-09** | **Email SMTP** | Keluar | 5 correspondence: notifikasi register, large losses, VA, laporan klaim, error produksi | ⚠️ **R-07** |
| **INT-10** | **6 API pengganti DB Link** | Masuk | HRD, jam kerja, GL payment, master sales, buka proteksi, pengguna lintas sistem | ⚠️ **R-03** — **belum ada** |

**Aturan integrasi yang mengikat:**

| ID | Requirement |
|---|---|
| **INT-11** | Setiap pemanggilan keluar punya **batas waktu 30 detik**, penanganan kegagalan, dan retry di mana relevan |
| **INT-12** | Log pemanggilan sistem eksternal dicatat; **data sensitif tidak masuk log** |
| **INT-13** | Pengganti `GENERAL.MST_BUKA_PROTEKSI` **wajib API idempoten** — ia satu-satunya objek remote yang **menulis**, ke tiga sistem berbeda, sehingga **tidak boleh** dijembatani salinan berkala |
| **INT-14** | Untuk data yang jarang berubah (master sales, cabang, agen), **salinan yang disegarkan berkala** boleh dipakai sebagai jembatan — **di balik seam yang sama**, sehingga penggantian ke API nyata nanti tidak menyentuh kode domain |
| **INT-15** | Pengganti `GetIDCabang` prioritas tertinggi — dipanggil **13 activity**, termasuk jalur registrasi |

---

## 8. Security Requirement

| ID | Requirement | Status |
|---|---|---|
| **SEC-01** | **Parameter binding tanpa perkecualian.** Menutup celah `{ASIS:...}` warisan yang menyisipkan nilai pengguna langsung ke teks SQL | ✅ |
| **SEC-02** | Nama kolom sort dan filter **hanya dari daftar yang diizinkan** | ✅ |
| **SEC-03** | Menyimpan potongan klausa SQL sebagai nilai property **dilarang sepenuhnya** | ✅ |
| **SEC-04** | `dangerouslySetInnerHTML` **dilarang** kecuali disetujui tertulis | ✅ |
| **SEC-05** | Cookie `SameSite=Strict` + token CSRF pada permintaan yang mengubah data | ✅ |
| **SEC-06** | Unggahan berkas: validasi jenis dan ukuran; **nama berkas dihasilkan sistem**, tidak pernah dari pengguna | ✅ |
| **SEC-07** | Tidak ada akses berkas berdasarkan jalur dari pengguna | ✅ |
| **SEC-08** | **Masking data sensitif dipertahankan** — konsep `FlagMaskingKTP`, `TelpMasking`, `NoKTPMasking` sudah ada di sistem lama | ✅ |
| **SEC-09** | Data sensitif (NIK, telepon, alamat, data medis, nomor rekening) **tidak pernah** ditulis ke log | ✅ |
| **SEC-10** | Sambungan ke database dan sistem eksternal **wajib terenkripsi** | ✅ |
| **SEC-11** | Kredensial **tidak pernah** di dalam kode maupun repository; dari variabel lingkungan atau pengelola rahasia | ✅ |
| **SEC-12** | **User ID di-hardcode sebagai penentu perilaku dihapus** (`MORASOTARDODOTARIGAN`, `ELLENSUPRIYATI`, `IRMANOPITAPURBA_1`, `RATNAGUSNITASARI`) — diganti peran dan izin dari master data | ✅ |
| **SEC-13** | **Hostname server tidak boleh menentukan perilaku bisnis** — diganti konfigurasi per lingkungan | ✅ |
| **SEC-14** | **Blok `// TESTING` yang menimpa email produksi dilarang** dan ditolak di code review | ✅ |
| **SEC-15** | Detail internal (struktur database, jejak tumpukan) **tidak pernah bocor ke klien**. Pesan `500` hanya memuat id permintaan | ✅ |

---

## 9. Error Handling Requirement

| ID | Requirement |
|---|---|
| **ERR-01** | **Kesalahan validasi bisnis** → HTTP `422`, **seluruhnya dikumpulkan** dan dikembalikan bersamaan |
| **ERR-02** | **Pelanggaran aturan / konflik** → HTTP `409`, satu kesalahan yang jelas |
| **ERR-03** | **Kegagalan teknis** → HTTP `500`, dicatat lengkap, pesan umum ke pengguna |
| **ERR-04** | Kesalahan domain adalah **tipe, bukan string**. Transport yang memetakannya ke kode HTTP; domain tidak boleh tahu tentang HTTP |
| **ERR-05** | Setiap kesalahan **dibungkus dengan konteks saat naik**, sehingga pesan akhirnya menceritakan jalurnya |
| **ERR-06** | Kesalahan **tidak pernah ditelan**. Tidak ada `_ = err`. Bila sengaja diabaikan, wajib disertai komentar yang menjelaskan alasannya |
| **ERR-07** | `panic` hanya untuk kondisi yang tidak mungkin terjadi; ditangkap middleware recovery. **Tidak pernah sebagai alur kendali** |

---

## 10. Requirement yang belum dapat difinalkan

**25 requirement** di atas bertanda ⚠️ dan **tidak dapat difinalkan dengan analisis lebih lanjut
atas export XML ini**. Semuanya bergantung pada **10 artefak** dari pihak luar — satu artefak
kerap memblokir beberapa requirement sekaligus.

| Requirement | Menunggu | Dari | Risiko |
|---|---|---|---|
| FR-B5, FR-B7, FR-B9, FR-B10, FR-B12, DAT-08 | Source **64 procedure & function** | DBA | **R-01** |
| FR-S6 | Daftar **job terjadwal** (Rule-Agent / Queue Processor) | Tim Pega | **R-02** |
| FR-S4, INT-10 | Kontrak **6 API pengganti DB Link** | Tim pemilik sistem | **R-03** |
| FR-B6 | Aturan **3 router penugasan** | Tim Pega | **R-04** |
| ~~FR-W6, DAT-11~~ | ~~Arti kode status~~ — ✅ **diterima**, 33 kode `1134`–`1166` | ~~DBA~~ | ~~R-06~~ tertutup |
| FR-S3, INT-08, INT-09, VR-10 | **40 activity** hilang — terutama `SendEmailNotification` (15×) dan `ValidasiSisaTSI` | Tim Pega | **R-07** |
| VR-09, DAT-10 | **DDL** + statistik ukuran tabel | DBA | **R-08** |
| BR-16, BR-18, FR-B7, FR-W5 | Isi **`POOLDATA.EMAILKOMITE`** + konfirmasi aturan `SetEmailKomite` | DBA + tim bisnis | **D-14** |
| NFR-29 | Lama **retensi audit** | Compliance | **D-28** |
| NFR-25 | Angka **RPO / RTO** | Tim infra | **D-29** |

Status permintaan: **sudah dikirim, menunggu jawaban** (I-05).

**Dampak paling berat: lima modul yang merupakan jantung nilai uang klaim** — FR-B5 (settlement),
FR-B7 (komite), FR-B9 (PLA/DLA), FR-B10 (akseptasi & pembayaran), FR-B12 (salvage) — **tidak
dapat diselesaikan** sampai R-01 tertutup.

Rincian: [`migration-readiness.md`](migration-readiness.md).
