# Risk Analysis — Migrasi Claim PNC

Risiko yang teridentifikasi dari analisis source dan hasil discovery. Diurutkan berdasarkan
dampak terhadap keberhasilan project.

Skala: **Dampak** dan **Kemungkinan** = Rendah · Sedang · Tinggi

> **Diperbarui v2.0 (2026-09-14).** Jumlah risiko menjadi **19**, bukan 12: `R-13`…`R-15` lahir
> dari penyusunan BRD, dan `R-16`…`R-19` dari verifikasi bukti Fase 2 (`D-38`). Tiga risiko
> berubah status setelah export bertambah pada 2026-09-09 (`D-45`): **`R-02` dan `R-06` tertutup**,
> **`R-01` sebagian tertutup**, **`R-04` turun** dari penghalang menjadi verifikasi. Ringkasan
> perubahan ada di [`21-RIWAYAT-REVISI.md`](21-RIWAYAT-REVISI.md).

---

## R-01 · Source procedure & function database tidak tersedia

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Terjadi — sudah pasti, bukan kemungkinan |
| **Status** | **Sebagian tertutup (2026-09-14)** — 62 dari 64 objek sudah diterima |

**Masalahnya.** D-02 menetapkan seluruh logika stored procedure ditulis ulang di Go. Tetapi
**source-nya tidak ada di export XML** — kita hanya melihat nama dan daftar parameternya dari
pemanggilan di SQL. Isinya belum pernah dilihat siapa pun di tim ini.

Jumlah terverifikasi: **56 stored procedure + 8 function lokal = 64 objek** yang harus diminta ke
DBA. Ditambah 6 function remote lewat DB Link yang ditangani terpisah di R-03.

**Yang sudah berubah (`D-45`).** Folder `Database/` diterima pada 2026-09-09 dan kini memuat
**55 berkas `.prc` + 8 berkas `.fnc`**. **62 dari 64 objek** yang diminta sudah ada. Dua yang
belum: **`SET_ATTACHFILETEMPSALVAGE`** (menghalangi `B-12`) dan **`UPDATEPREMIUMTEMPLATE`**
(menghalangi `B-9`).

**Yang membuat risiko ini belum tertutup penuh.** 62 berkas yang datang **memanggil 12 objek
lain yang tidak ikut dikirim**, terberat **`UPDATE_LOG_KONVERSI` (162 pemanggilan)**, `GETNEWID`
(42), `PKG_COUNTER_PRODUCTION` (25), `PROCESSQUEUEDIRECT` (10). Logikanya tetap harus dibaca
untuk ditulis ulang, meski objeknya kelak ditinggalkan (`D-68`, `ADR-0007`).

> **Daftar lengkapnya ada di [`19-GAP-EXPORT-DETAIL.md`](19-GAP-EXPORT-DETAIL.md)** — beserta
> jumlah parameter, rule pemanggil, modul terdampak, dan query siap pakai untuk menarik source-nya.

Yang terdampak antara lain `PEGA_JSON_KLAIM_PNC`, `INSERT_PLADLA`, `INSERT_SURVEYORLIST`,
`CONVERTJSONPRODUCTION`, `INSERTDATAKOMITELIST`, `INSERT_SALVAGE_DETAILS`,
`PNC_INSERT_EMAIL_ADJUSTER`, `GET_POSISI_PROGRESS_PNC`, `ADD_NEWMASTERVIRTUALACCOUNT`.

**Akibatnya.** Modul B-5 (settlement), B-7 (komite), B-9 (PLA/DLA), B-10 (akseptasi), dan B-12
(salvage) **tidak dapat diselesaikan** sampai isinya diketahui. Ini bukan penundaan kecil —
kelimanya adalah jantung nilai uang klaim.

**Penanganan.**
1. Minta source lengkap ke DBA **pada hari pertama** — ini di jalur kritis.
2. Sambil menunggu, kerjakan modul yang tidak bergantung: fondasi, registrasi, inbox, dokumen.
3. Bila tetap tidak tersedia, satu-satunya jalan adalah membaca perilakunya lewat perbandingan
   masukan dan keluaran di lingkungan staging — jauh lebih lambat dan tidak menjamin kelengkapan
   kasus tepi.

---

## R-02 · Job terjadwal tidak diketahui

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Terjadi |
| **Status** | **TERTUTUP (2026-09-09)** — `D-57` |

**Masalahnya (saat risiko ini ditulis).** Pemilik project mengonfirmasi ada job terjadwal, tetapi
belum tahu detailnya (D-17). Export XML **tidak memuat rule Agent maupun Queue Processor Pega**,
sehingga tidak dapat dianalisis dari source sama sekali.

**Bagaimana ditutup.** Pada 2026-09-09 export bertambah dan tiga folder yang sebelumnya tidak ada
muncul: `Job Scheduler/` (5 berkas), `Agents/` (1), `Service REST/` (4). Daftar lengkap job
terjadwal, terverifikasi dari `pyRuleName` dan `pyActivityName` masing-masing berkas:

| Job | Frekuensi | Jam mulai | Activity target | Target ada? |
|---|---|---|---|---|
| `JOBForKomiteKlaimPNC` | Daily | **06:00:00** | **`AutoAcceptKomite`** | ✅ |
| `JobHitungDeadlineToTemporaryCLose` | **Weekly** | 23:43:00 | `JobTemporaryCloseClaimPNC` | ✅ |
| `JobSendAutoLODKlaimPersonal` | Daily | 20:54:00 | `Act_SendAutoLODKlaimPersonal` | ✅ |
| `PNCMyReportKlaim3` | Daily | 08:00:00 | `ReportAI_Act` | ✅ |
| `ProcessClaimKredit` | Daily | 10:00:00 | `CreateClaimCredit_Table` | ✅ |

Seluruhnya `pyIsEnabled=true`, `pyApplicableTo=Cluster`, `pyNodeTypesText=BackgroundProcessing`.
Ditambah satu agent: `Agents/TATReportAgent-Agents.xml`, `pyEnable=true`,
`pyTriggerInterval=1800` (30 menit), `pyBypassActivityAuthentication=true`.

**Konsekuensi.** Modul `S-6` Penjadwalan naik dari **TERHALANG** menjadi **SEBAGIAN**. Yang
tersisa bukan lagi soal artefak melainkan soal rancangan dan kewenangan — dipindahkan ke
`ADR-0022`, yang masih berstatus `Proposed`:

1. Siapa yang menjalankan job bila aplikasi hidup di dua instans (`D-27`), karena Go tidak punya
   padanan otomatis untuk `pyApplicableTo=Cluster`.
2. **`AutoAcceptKomite` menyetujui komite otomatis setiap hari jam 06:00**, melewati kontrol menu
   `D-59` — apakah dipertahankan.
3. Tiga job ber-`pyDescription = "job jalan 2 menit"` sementara konfigurasinya `Daily`/`Weekly`.

---

## R-03 · Enam API pengganti DB Link belum ada

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Tinggi |
| **Status** | Terbuka |

**Masalahnya.** D-25 mengganti 64 pemakaian DB Link dengan pemanggilan API. Tetapi API-API itu
kemungkinan besar belum ada, dan yang membangunnya adalah **tim lain dengan jadwal sendiri**.

Terdampak: data HRD, jam kerja, pembayaran GL, master sales, status buka proteksi, dan pengguna
lintas sistem.

Angka terverifikasi: **64 pemakaian · 6 DB Link · 28 objek remote · 27 rule**.

> **Inventaris lengkapnya ada di [`20-DETAIL-KOMITE-DBLINK.md`](20-DETAIL-KOMITE-DBLINK.md)** —
> objek remote per DB Link, proses bisnis yang terdampak, 27 rule pemakai, usulan 6 API pengganti,
> serta mana yang boleh dijembatani salinan berkala dan mana yang wajib API sungguhan.

Dua titik paling kritis: **`GetIDCabang`** (dipanggil 13 aktivitas, termasuk jalur registrasi) dan
**`GENERAL.MST_BUKA_PROTEKSI`** — satu-satunya objek remote yang **menulis**, dipanggil ke tiga
sistem berbeda, sehingga penggantinya wajib API idempoten dan tidak bisa disalin berkala.

**Akibatnya.** Ketergantungan pada pihak di luar kendali tim, tepat pada project dengan jadwal
sangat ketat (D-30).

**Penanganan.**
1. Inventarisasi kebutuhan per sistem dan sampaikan ke tim pemilik **sekarang juga**.
2. Untuk data yang jarang berubah — master sales, cabang, agen — pakai **salinan yang
   disegarkan berkala** sebagai jembatan, di balik seam yang sama sehingga penggantian ke API
   nyata nanti tidak menyentuh kode domain.
3. Untuk data yang harus mutakhir, modul yang bergantung padanya tertahan — dan itu harus
   disampaikan ke manajemen sebagai konsekuensi jadwal, bukan diserap diam-diam.

---

## R-04 · Tiga router penugasan tidak ada di export

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Terjadi |
| **Status** | **Turun menjadi verifikasi (2026-09-14)** — algoritmanya sudah dapat direkonstruksi |

**Masalahnya.** `PNCAdminRouter`, `PNCTeknikRouter`, dan `RouterRCLDokter` dirujuk oleh
Register_Flow tetapi **tidak ada di export**. Ketiganya buatan sendiri (ruleset GCNMFW) dan
menentukan **siapa menerima tugas apa** — inti dari modul B-6.

**Yang sudah berubah.** Algoritma pembagian beban **ditemukan di tempat lain** dan tidak lagi
perlu ditebak: `RDB List/BrowsePICRandomTeam-SQL.xml:39-40` memilih petugas dengan
**`ORDER BY counter_quota ASC`** — yang paling sedikit bebannya — lalu `AddTJobCounterPIC_SQL`
menaikkan pencacahnya. Risiko ini karena itu **turun dari penghalang menjadi verifikasi**: yang
dibutuhkan bukan lagi artefaknya, melainkan konfirmasi bahwa rekonstruksi ini memang aturan yang
berlaku (`ADR-0019`).

**Catatan yang mempersempit lingkup `B-6`.** Penjenjangan komite ternyata menugaskan ke
**operator bernama**, bukan ke workbasket, dan seluruh export hanya memuat **4 workbasket**
(`T-7`). Model penugasan `B-7` karena itu tidak seluruhnya mengikuti pola Worklist/Workbasket.

Satu router lain juga tidak ada di export, yaitu `ToWorkList`, tetapi itu **bawaan Pega**
(`Pega-ProcessEngine`) dengan perilaku baku — bukan gap nyata dan tidak perlu diminta.

Yang tersedia hanya `KomiteRouter` dan `PNCAdminRouterRCV`, sehingga polanya bisa ditebak
(pemilihan berdasarkan lini bisnis, cabang, dan beban kerja), tapi aturan persisnya tidak.

> **Daftar lengkap 8 router beserta shape yang memakainya ada di
> [`19-GAP-EXPORT-DETAIL.md`](19-GAP-EXPORT-DETAIL.md).**

**Penanganan.** Minta export ketiganya. Bila tidak tersedia, gali aturannya dari tim operasional
dan buktikan dengan membandingkan hasil penugasan pada data historis.

---

## R-05 · Jadwal tidak sepadan dengan ukuran pekerjaan

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Tinggi |
| **Status** | Diterima manajemen (D-30) |

**Masalahnya.** Target seluruh modul selesai akhir September 2026 (D-30), sekitar 3 minggu sejak
analisis ini. Ukuran pekerjaan yang terukur dari source:

| Yang harus dibangun | Ukuran |
|---|---|
| Layar | 74 harness, 269 section, 268 bergrid |
| Logika bisnis | 15.063 step activity |
| Query | 652 rule SQL, 534 KB |
| Procedure & function di-Go-kan | 64 — source belum ada (R-01) |
| Laporan + engine dokumen | 56 report definition |
| Integrasi | **21 REST** + 6 API baru (R-03) |
| Belum ada sama sekali | Tabel auth/authz, tabel penugasan, modul master data, daftar job (R-02) |

Ditambah tim yang harus belajar **Go, React, dan TypeScript sekaligus** (D-09).

Perkiraan berdasarkan ukuran di atas: **9–15 bulan dengan tim 5–6 orang.**

**Status.** Keberatan sudah disampaikan beserta angkanya. Pemilik project menegaskan target
tetap berlaku dan seluruh modul harus pindah. Keputusan ini dihormati dan Migration Strategy
disusun untuknya.

**Penanganan.**
1. Urutan pengerjaan disusun agar pekerjaan berisiko rendah berjalan lebih dulu — sehingga bila
   waktu tidak mencukupi, yang tertinggal adalah bagian yang paling sedikit dampaknya.
2. Tiga hal yang **tidak bisa dipercepat oleh penambahan orang** harus dimulai hari ini:
   menunggu source procedure & function (R-01), menunggu API tim lain (R-03), dan waktu belajar tim.
3. Bila pemangkasan menjadi perlu, dokumen ini menyediakan dasar angka untuk memutuskan bagian
   mana yang ditunda.

---

## R-06 · Arti kode status 1142–1151 tidak diketahui

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Terjadi |
| **Status** | **TERTUTUP (2026-09-09)** — isi master diterima |

**Masalahnya (saat risiko ini ditulis).** `StatusClaim` memakai kode `1142`–`1151` yang artinya
tersimpan di view `V_STS_CLAIM.LSC_NOTE`. **Isi view itu tidak ada di export.** Yang berhasil
terbaca dari kode hanya `1146`, yang di-set ketika pengguna menekan "Back" pada tahap registrasi.

**Bagaimana ditutup.** Isi master diterima sebagai `v_sts_claim.csv`. Hasilnya **memperbaiki
premisnya sendiri**: domain `StatusClaim` bukan 10 kode `1142`–`1151`, melainkan **33 kode
`1134`–`1166`**. Sebelas kode pertama (`1134`–`1144`) membawa penomoran lama `01`–`11`; sisanya
hanya bernomor baru. Contoh: `1142` Rejected Claim · `1143` Close Claim for this object ·
`1144` Cancelled Claim · `1147` Register · `1149` Claim Committee · `1163` Paid ·
`1164` Reopen Claim.

Rentang `1142`–`1151` yang dipakai dokumen-dokumen awal ternyata **sebagian** dari domainnya,
bukan keseluruhannya. Definisi resmi ada di [`CONTEXT.md`](CONTEXT.md) dan `ADR-0018`.

**Catatan atas tebakan yang gugur.** Sebelum master diterima, tiga arti kode sempat disimpulkan
dari pemakaiannya di rule dan **ketiganya salah**: `1143` bukan "status awal" melainkan Close
Claim, `1150` bukan penanda terdaftar melainkan LOD Report, dan `1151` bukan Investigator
melainkan Analyst. Ini alasan konkret mengapa arti kode status tidak boleh disimpulkan dari
pemakaian.

---

## R-07 · 50 activity dipanggil tapi tidak ada di export

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Terjadi |
| **Status** | Terbuka |

**Masalahnya.** Dari 554 activity yang dipanggil, 50 tidak ada di export. Sebagian adalah
bawaan Pega dan tidak masalah, tetapi beberapa jelas berisi logika bisnis:

`SendEmailNotification` (dipanggil **15×**), `SetTicket` (4×), `SetKasir_Act` (3×),
`generatePDF`, `generatePD4ML`, `PNCSalvageHistorySemuaKlaim`, `SendFilePendukungLelangKeSimasBit`,
`InsertDataSlinkOJKIndividu`, `UploadDocumentToGoogleStorage`, `ValidasiSisaTSI`,
`GCNMGetLossAdjuster_Act`, `SendUpdateCIF_act`.

> **Daftar lengkap 50 activity — 45 buatan sendiri dan 5 bawaan Pega — beserta activity pemanggil
> dan modul terdampak ada di [`19-GAP-EXPORT-DETAIL.md`](19-GAP-EXPORT-DETAIL.md).**

`ValidasiSisaTSI` patut diperhatikan khusus — namanya menunjukkan aturan validasi sisa TSI yang
tidak muncul di analisis aturan bisnis manapun.

**Penanganan.** Minta export ke-50 activity tersebut. Prioritaskan `SendEmailNotification`,
`ValidasiSisaTSI`, `SetKasir_Act`, dan `SetTicket`.

---

## R-08 · DDL tabel tidak tersedia

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Terjadi |
| **Status** | Terbuka |

**Masalahnya.** 245 tabel teridentifikasi dari SQL, tetapi **tipe kolom, panjang, index,
constraint, dan foreign key tidak diketahui**. Desain skema baru harus menebak.

Contoh nyata dampaknya: kode lama memotong `ObjectName` pada 3.800 karakter "karena melebihi
batas database" — batas sebenarnya tidak diketahui.

**Penanganan.** Minta DDL lengkap schema `POOLDATA`, `DATAPEGA`, dan `GENERAL`. Sekaligus minta
statistik ukuran tabel — itu yang menentukan strategi index dan partisi (Database Strategy §6).

---

## R-09 · Sistem sumber masih aktif berubah

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Tinggi |
| **Status** | Melekat pada Strangler Fig |

**Masalahnya.** 124 activity diubah pada 2026 dan 180 pada 2024. Pega adalah **sasaran bergerak**:
perubahan yang dibuat tim Pega selama migrasi tidak otomatis ikut ke sistem baru.

**Penanganan.**
1. Sepakati **pembekuan perubahan** pada modul yang sedang dimigrasi.
2. Perubahan mendesak yang tetap harus dilakukan di Pega **wajib dicatat** dan diterapkan juga
   di sistem baru.
3. Rekonsiliasi berkala membandingkan perilaku kedua sistem (Testing Strategy §6).

---

## R-10 · Data ganda antara tabel relasional dan JSON

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Sedang |
| **Status** | Terbuka |

**Masalahnya.** Data klaim yang sama disimpan dua kali: tabel relasional `POOLDATA` dan dokumen
JSON (`JSON_KLAIM`, `JSON_POLIS`). Sinkronisasi dilakukan stored procedure. **Tidak ada mekanisme
yang memeriksa apakah keduanya konsisten.**

**Akibatnya.** Bila keduanya berbeda pada data historis, sistem baru harus memutuskan mana yang
benar — dan keputusan itu memengaruhi hasil laporan dan perhitungan.

**Penanganan.**
1. Jalankan pemeriksaan konsistensi pada data produksi sebelum migrasi.
2. Tetapkan sumber kebenaran secara eksplisit per jenis data.
3. Di sistem baru, satu data hanya disimpan satu kali — snapshot polis sebagai JSON (D-04),
   data klaim sebagai tabel relasional.

---

## R-11 · Tim mempelajari tiga teknologi sekaligus

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Tinggi |
| **Status** | Melekat pada D-09 dan D-23 |

**Masalahnya.** Tim adalah developer Pega yang harus menguasai Go, React, dan TypeScript
bersamaan. D-23 memilih React yang tidak opinionated, sehingga risiko **kode berbeda gaya di
setiap layar** menjadi nyata — inilah alasan React bukan rekomendasi awal saya.

**Penanganan.**
1. Coding Standards yang **mengikat**, bukan anjuran (dokumen 08).
2. Penegakan otomatis lewat lint, `depguard`, dan pemeriksaan pola SQL — bukan hanya code review.
3. Pustaka komponen baku (U-2) dibangun lebih dulu, sehingga layar-layar berikutnya merakit,
   bukan mencipta.
4. Code review wajib pada masa awal, terutama untuk layar pertama tiap jenis.
5. Modul baca dikerjakan lebih dulu (Migration Strategy Tahap 2) — tim belajar pada pekerjaan
   yang kesalahannya tidak merusak data.

---

## R-12 · Zona waktu bergeser saat migrasi data

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Sedang |
| **Status** | Terbuka |

**Masalahnya.** Sistem lama menyimpan waktu dalam GMT dan menambahkan 7 jam secara manual di
banyak tempat. Sistem baru menyimpan UTC dan mengonversi di satu tempat (F-5).

Bila ada data lama yang **sudah** tersimpan dengan 7 jam tertambah — karena satu jalur kode
menyimpannya setelah konversi — data itu akan bergeser lagi saat dibaca sistem baru.

**Akibatnya.** Tanggal kejadian dan tanggal lapor bergeser satu hari pada kasus di sekitar
tengah malam, dan itu **mengubah hasil validasi** aturan seperti "Tanggal Lapor ≤ DOL + 7 hari".

**Penanganan.**
1. Periksa contoh data produksi untuk memastikan konvensi penyimpanan yang sebenarnya.
2. Uji khusus untuk kasus di sekitar tengah malam WIB (Testing Strategy §3.1).
3. Bandingkan tanggal yang ditampilkan Pega dan sistem baru pada klaim yang sama.

---

## R-13 · Tenggat tidak punya pemaksa eksternal

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Terjadi |
| **Status** | Terbuka · pemilik **manajemen** |

**Masalahnya.** Pendorong migrasi adalah kemandirian teknologi dan maintainability — **bukan**
lisensi yang berakhir, **bukan** biaya lisensi, dan **bukan** arahan standardisasi grup. Target
akhir September 2026 karena itu adalah **target internal**, tanpa tanggal yang dipaksakan pihak
luar dan **tanpa penalti kontraktual**.

**Kenapa ini risiko, bukan kabar baik.** Menerima `R-05` apa adanya akan wajar bila ada tenggat
eksternal yang tidak dapat digeser. Di sini tidak ada, sehingga risiko yang diterima **tidak
dibayar oleh manfaat apa pun** — yang tersisa hanya biayanya: mutu yang dikompromikan pada sistem
yang menangani uang klaim.

**Status keputusan.** `D-61` menetapkan **jadwal seluruh modul tidak berubah**, dan selisih antara
jadwal dan kesiapan menjadi **tanggung jawab manajemen**. Risiko ini karena itu tetap terbuka
secara sadar, bukan karena belum dibahas.

---

## R-14 · Ukuran keberhasilan tidak dapat dibuktikan untuk lima modul

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Terjadi |
| **Status** | **Membaik** seiring `R-01` — 62 dari 64 objek sudah diterima |

**Masalahnya.** Keberhasilan diukur sebagai kesetaraan fungsional dengan Pega, dan gerbang pertama
cutover adalah uji kesetaraan otomatis. Keduanya menuntut satu hal yang sama: **mengetahui apa
hasil yang benar.** Untuk `B-5`, `B-7`, `B-9`, `B-10`, dan `B-12`, hasil yang benar ditentukan
oleh procedure yang logikanya belum pernah dibaca (`R-01`).

> **Kesetaraan atas logika yang tidak diketahui tidak dapat dibuktikan, hanya diduga.**

**Yang sudah berubah.** Setelah `Database/` diterima, `D-55` mencabut `BRD §21.4` untuk **`B-7`,
`B-9`, `B-10`, dan `B-12`**; hanya **`B-5`** yang masih terikat, dengan tiga penghalang tersisa —
isi `POOLDATA.GCNM_FEE_SCALE` (17 pita), isi `m_currencystandard`, dan
`BrowseT_Claim_Adjustment_SQL`.

**Dua modul yang persoalannya berbeda.** `F-3` dan `S-5` **tidak punya baseline Pega sama sekali**
— HCC/HCQ nol jejak di export (`T-2`), dan sistem lama tidak punya jejak audit atas nilai
(`T-14`). Untuk keduanya, gerbang 1 diganti **uji fungsional terhadap kontrak** (`D-56`,
`ADR-0028`), dan kedua kontrak itu **belum ada**.

---

## R-15 · Waktu user bisnis untuk UAT belum dialokasikan

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | Terjadi |
| **Status** | Terbuka · pemilik **manajemen** |

**Masalahnya.** Gerbang 2 setiap modul adalah UAT pengguna bisnis, untuk sekitar 20 inbox berbasis
peran dan 5 layar transaksi utama. Pengguna yang sama sedang menjalankan **operasional klaim
harian**, dan alokasi waktunya **belum tercatat di mana pun** serta **tidak punya ruang pada
jadwal `C-2`**.

**Penanganan.** Alokasikan resmi per modul dan perlakukan sebagai **constraint project** (`C-9`),
bukan detail pelaksanaan yang diasumsikan tersedia. `D-60` menetapkan modul fondasi tidak melewati
gerbang 2 — itu mengurangi beban, tetapi tidak menghapusnya untuk modul bisnis.

---

## R-16 · Export Pega tidak lengkap

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Terjadi |
| **Status** | Terbuka · pemilik **Tim Pega** · **memburuk** |

**Masalahnya.** Sekitar **242 rule dirujuk tetapi tidak ada di export** — jauh di atas angka yang
diaudit sebelumnya. Tujuh tipe rule **tidak diaudit `19-GAP` sama sekali**: When, Flow Action,
Section, Data Transform, Ticket, Report Definition, dan Function library.

**Yang memburuk.** Jumlah **When rule buatan sendiri yang hilang naik dari 116 menjadi 137**
setelah penghitungan ulang terhadap snapshot baru (`D-45`). When rule adalah **kategori ketiga**
di luar `R-01` (objek database) dan `R-07` (activity), dan dampaknya lebih luas: ia memblokir
**percabangan bisnis di hampir semua modul**, bukan lima modul.

**Contoh konkret yang sudah menggigit.** Lima When rule yang mengendalikan item menu hilang —
`IsGCNMReport`, `IsKomite`, `IsNotViewClaim`, `IsPNCBonding`, `IsSurvey` — sehingga peta peran →
menu pada `F-3` tidak dapat disusun lengkap (`ADR-0023`). Delapan Ticket rule custom juga hilang
(`ADR-0021`).

**Penanganan.** Permintaan export ulang berbasis **Product rule**, bukan pemilihan manual per tipe
(`D-39`).

---

## R-17 · Kredensial aktif berada di dalam export

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | **Sudah terjadi** |
| **Status** | Terbuka · pemilik **Tim Infra / Security** |

**Masalahnya.** Export memuat **3 password SMTP di 31 lokasi** ditambah **satu pasang kredensial
OAuth**, seluruhnya plaintext, dan `UseSSL=false` pada seluruh kemunculannya.

**Kenapa ini berbeda dari risiko lain di dokumen ini.** Ia **bukan risiko migrasi**: ia hidup
sekarang, bahkan bila migrasi dibatalkan hari ini. Yang dipertaruhkan juga bukan jadwal, melainkan
**cara repository ini disimpan dan dibagikan**.

**Penanganan.** `D-40` masih **OPEN** — ke mana rahasia dipindahkan, siapa pemiliknya, dan apakah
kredensial yang telanjur terekspos harus dirotasi (`ADR-0025`). Lokasi lengkapnya sudah diserahkan
ke Tim Infra/Security lewat dokumen terpisah **di luar repository**; tidak ada satu nilai pun
tertulis di dokumen yang di-commit (`D-69`).

---

## R-18 · Dua integrasi BRI menunjuk host sandbox

| | |
|---|---|
| **Dampak** | Sedang |
| **Kemungkinan** | **Sudah terjadi** |
| **Status** | Terbuka · pemilik **Work Owner + tim integrasi** |

**Masalahnya.** Dua Connect REST ke BRI menunjuk **host sandbox** di dalam ruleset produksi.
Entah integrasi itu memang tidak aktif, atau ia aktif dan mengarah ke lingkungan yang salah —
keduanya harus dijawab sebelum integrasi dibawa ke sistem baru.

**Penanganan.** Konfirmasi status kedua integrasi ke tim integrasi, dan tetapkan endpoint per
lingkungan sebagai konfigurasi (`ADR-0025`), bukan konstanta di dalam rule.

---

## R-19 · Cacat aturan uang direplikasi bila `P-5` dipatuhi buta

| | |
|---|---|
| **Dampak** | Tinggi |
| **Kemungkinan** | Terjadi bila tidak ditangani |
| **Status** | **Ditangani** — `D-49` · pemilik **Work Owner** |

**Masalahnya.** `P-5` menetapkan kesetaraan perilaku lebih dulu, termasuk meniru keanehan sistem
lama. Dipatuhi buta, prinsip itu akan **membawa serta cacat pada aturan yang menghitung uang** —
antara lain toleransi spreading berupa pencocokan substring (`199.99` dan `1100.0` lolos), ambang
Rp 50 juta dengan **tiga operator berbeda** di tiga rule, penyesuaian 7 jam yang asimetris di
dalam satu kondisi validasi, dan kurs yang mengembalikan `1` saat tidak ditemukan.

**Bagaimana ditangani.** `D-49` memutuskan sepuluh cacat satu per satu: **sembilan diperbaiki, satu
direplikasi** karena perilakunya memang benar (tanggal PLA/DLA diambil dari waktu sistem, bukan
dari pilihan pengguna). Daftar perbaikan eksplisit `P-5` bertambah dari **4 menjadi 13 butir**
(`ADR-0017`).

**Yang tersisa.** Data historis tetap mengandung akibat cacat itu — memperbaiki aturannya tidak
memperbaiki baris yang telanjur salah, khususnya `IDSALVAGE = NULL` yang menuntut hitungan DBA.

---

## R-20 · Kebocoran data antar badan hukum saat berpindah portal

| | |
|---|---|
| **Dampak** | **Sangat tinggi** — lintas badan hukum |
| **Kemungkinan** | Terjadi bila tidak ditangani |
| **Status** | **Terbuka** — `D-75` · `ADR-0030` · pemilik **Work Owner + Keamanan Informasi** |

**Masalahnya.** `D-75` menetapkan empat portal dengan **satu database per entitas**, dan
perpindahan portal **tanpa login ulang**. `D-78` kemudian menetapkan **login yang sama untuk
keempat entitas**, sehingga **satu kredensial bocor menjangkau empat badan hukum** — batas yang
tadinya ada bila tiap entitas punya login sendiri kini **hilang**, dan yang tersisa sebagai
pembatas hanyalah **kewenangan portal per pengguna**. Artinya **satu identitas menjangkau empat database milik
empat badan hukum**. Bila kewenangan tidak dinilai ulang pada saat perpindahan, seorang pengguna
dapat melihat data entitas yang bukan haknya.

**Kenapa ini lebih berbahaya daripada cacat akses biasa.** Kegagalannya **tidak terlihat sebagai
galat**. Layar tampil normal, angkanya masuk akal, dan tidak ada pesan kesalahan — yang salah hanya
*milik siapa* data itu. Cacat seperti ini dapat berjalan lama tanpa ada yang menyadarinya.

Dua jalur kegagalan yang sudah terlihat di rancangan:

1. **Jatuh ke koneksi default** ketika portal tidak diketahui, alih-alih menolak permintaan.
2. **Portal aktif disimpan sebagai keadaan global** di server — dua permintaan bersamaan dari
   pengguna yang sama akan saling menimpa, dan data kedua entitas tertukar.

**Bagaimana ditangani.** `TKT-F6-003` menuntut kewenangan **dinilai ulang di server pada setiap
perpindahan** (`ADR-0023`), portal yang bukan hak **tidak muncul dan tidak dapat dipanggil
langsung**, dan portal aktif **melekat pada permintaan**, bukan pada keadaan global. `TKT-F6-002`
menuntut permintaan tanpa portal **ditolak**, bukan jatuh ke default.

**Yang tersisa.** Tiga hal belum diputuskan dan menahan penanganan: apakah penyedia identitas satu
untuk keempat entitas atau satu per entitas · di mana kewenangan portal disimpan (bila di database
tiap portal, memeriksa hak atas portal B menuntut membaca database B **sebelum** pengguna berhak
membacanya) · dan apakah satu pengguna boleh berhak di lebih dari satu portal. Ditambah `D-59` yang
menetapkan kontrol berbasis **menu** tanpa pemisahan tugas formal — kendali portal **tidak boleh**
ikut bersandar pada penyembunyian menu.

---

## Ringkasan tindakan yang harus dimulai hari pertama

Diperbarui 2026-09-14. Tanda ✅ menandai tindakan yang **sudah selesai**.

| # | Tindakan | Menutup risiko | Kepada | Status |
|---|---|---|---|---|
| 1 | Minta source 64 procedure & function (daftar: `19-GAP-EXPORT-DETAIL.md`) | R-01 | DBA | ✅ **62 dari 64 diterima** — sisa `SET_ATTACHFILETEMPSALVAGE`, `UPDATEPREMIUMTEMPLATE` |
| 1b | Minta **12 dependensi** yang dipanggil 62 procedure itu (terberat `UPDATE_LOG_KONVERSI` 162×) | R-01 | DBA | **Terbuka** |
| 2 | Minta export Rule-Agent / Queue Processor | R-02 | Tim Pega | ✅ **selesai** — 5 job + 1 agent (`D-57`) |
| 3 | Minta DDL + statistik ukuran tabel | R-08 | DBA | Terbuka |
| 4 | Ambil isi master `V_STS_CLAIM` | R-06 | DBA | ✅ **selesai** — 33 kode `1134`–`1166` |
| 5 | Minta export 3 router + activity yang hilang | R-04, R-07 | Tim Pega | Terbuka — `R-04` turun menjadi verifikasi |
| 5b | **Permintaan export ulang berbasis Product rule** — ±242 rule hilang, 137 di antaranya When rule | R-16 | Tim Pega | **Terbuka** (`D-39`) |
| 6 | Sampaikan kebutuhan 6 API pengganti DB Link | R-03 | Tim pemilik sistem | Terbuka |
| 7 | Ambil isi tabel `POOLDATA.EMAILKOMITE` (matriks komite) | D-14 | DBA | ✅ **selesai** — 21 kolom, 30 baris |
| 7b | Konfirmasi aturan `SetEmailKomite` yang berbasis nama orang | D-14 | Tim bisnis | ✅ **selesai** — dicabut, diganti pita nilai (`D-52`, `D-70`) |
| 8 | Konfirmasi retensi audit | D-28 | Compliance | ✅ **terjawab** — mengikuti retensi data klaim (`D-62`); **angkanya** masih diminta |
| 9 | Sepakati pembekuan perubahan Pega | R-09 | Tim Pega + manajemen | Terbuka |
| 10 | Mulai pelatihan Go, React, TypeScript | R-11 | Tim pengembang | Terbuka |
| 11 | **Tetapkan tujuan penyimpanan rahasia dan keputusan rotasi** | **R-17** | Tim Infra / Security | **Terbuka** (`D-40`) |
| 12 | Konfirmasi status dua integrasi BRI bersandbox | **R-18** | Tim integrasi | **Terbuka** |
| 13 | Sediakan **kontrak API HCC/HCQ** | R-14 | Tim HCC/HCQ | **Terbuka** (`ADR-0024`) |
| 14 | Sediakan **daftar peristiwa wajib audit** | R-14 | Compliance | **Terbuka** (`ADR-0026`) |
| 15 | Konfirmasi ketersediaan **Pega staging yang dapat ditembak dari luar** | R-14 | Tim Pega + Infra | **Terbuka** — penghalang utama `S-8` (`ADR-0027`) |

**Sebagian besar tindakan ini bergantung pada pihak di luar tim pengembang.** Karena itu
seluruhnya harus dimulai hari pertama — waktu tunggunya tidak bisa dipercepat dengan menambah
developer.
