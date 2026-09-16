# Migration Readiness — Claim PNC

Evaluasi apakah aplikasi Claim PNC sudah siap dimigrasikan ke Golang, mengikuti checklist
Phase 5.

| | |
|---|---|
| **Tanggal evaluasi** | 2026-09-08 · **dimutakhirkan 2026-09-14** |
| **Sumber** | [`Steering/STEERING.md`](Steering/STEERING.md) v2.0 · 24 berkas [`Steering/`](Steering/) · **29 ADR** · [`requirement-summary.md`](requirement-summary.md) |

---

## 1. Verdict

> **Pemutakhiran 2026-09-14.** Sejak evaluasi ini ditulis, **empat penghalang tertutup** dan
> **tujuh penghalang baru muncul**. Verdict keseluruhan **tidak berubah** — masih tertahan pihak
> luar — tetapi **komposisinya berubah**, dan tiga penghalang baru memblokir **gerbang kelulusan**,
> bukan sekadar pengerjaan. Tabel kesiapan per kelompok di bawah sudah disesuaikan; rincian di
> [`Steering/21-RIWAYAT-REVISI.md`](Steering/21-RIWAYAT-REVISI.md) §4.

> ## ⛔ NOT READY — BLOCKED ON EXTERNAL
>
> **Pemahaman dan perencanaan sudah lengkap. Yang menahan bukan analisis, melainkan sepuluh
> artefak yang harus datang dari pihak di luar tim pengembang — dan belum satu pun diterima.**
>
> Tim pengembang **sudah menyelesaikan bagiannya**: Steering selesai, BRD selesai, requirement
> bernomor dan tervalidasi, risiko terdokumentasi, dan permintaan artefak sudah dikirim (I-05).

**Namun ini bukan verdict "berhenti".** Kesiapan tidak seragam per modul:

| Kelompok | Modul | Kesiapan | Boleh mulai? |
|---|---|---|---|
| **Fondasi** | FR-F1, FR-F2, FR-F5 | Lengkap | ✅ **Ya, hari ini** |
| **Identitas** | FR-F3 | **Kontrak HCC/HCQ nol jejak di export** | ⛔ **Tidak** — memblokir login seluruh aplikasi (`ADR-0024`) |
| **Master data** | FR-F4 | Lengkap kecuali **tujuan penyimpanan rahasia** (`D-40`) | ⚠️ Sebagian |
| **Kerangka UI** | FR-U1, FR-U2 | Lengkap — dan **lebih ringan** dari perkiraan (median 6 kolom, bukan 18–27) | ✅ **Ya, hari ini** |
| **Jejak audit** | FR-S5 | **Tanpa baseline Pega** — butuh daftar peristiwa wajib audit dari Compliance | ⚠️ Boleh dibangun; **tidak dapat lulus gerbang** sampai kontraknya ada (`ADR-0026`) |
| **Jalur klaim awal** | FR-B1 … FR-B4, FR-B13, FR-B14, FR-S1 | Lengkap | ✅ **Ya** |
| **Baca-saja** | Inbox, pencarian, detail, laporan | Lengkap | ✅ **Ya** — dan sengaja didahulukan (Tahap 2) |
| **Jantung nilai uang** | FR-B7, FR-B9, FR-B10, FR-B12 | **Lepas dari §21.4** — 62 dari 64 objek diterima | ✅ **Ya**, dengan penghalang spesifik per modul |
| **Settlement** | FR-B5 | **Masih terikat §21.4** — `GCNM_FEE_SCALE`, `m_currencystandard`, `BrowseT_Claim_Adjustment_SQL` | ⛔ **Tidak** |
| **Penugasan** | FR-B6 | **R-04 turun menjadi verifikasi** — algoritma beban terbaca dari `BrowsePICRandomTeam-SQL.xml:39-40` | ✅ **Ya**, dengan verifikasi |
| **Scheduler** | FR-S6 | **R-02 tertutup** — 5 job + 1 agent; sisa keputusan rancangan dua instans | ⚠️ Sebagian (`ADR-0022`) |
| **Perkakas uji** | FR-S8 | **Pega staging belum dikonfirmasi** | ⛔ **Tidak** — memblokir **gerbang 1 seluruh modul** (`ADR-0027`) |
| **Alur lompatan tahap** | FR-B11, FR-B13 | **8 Ticket rule custom hilang**; 14 dari 17 nama tanpa pemicu | ⚠️ Sebagian (`ADR-0021`) |
| **Integrasi DB Link** | FR-S4, INT-10 | **Terhalang R-03** | ⚠️ Sebagian — seam boleh, adapter nyata tidak |

**Rekomendasi ringkas:** mulai kelompok yang bertanda ✅ sekarang, sambil **menetapkan tanggal
komitmen** untuk sepuluh artefak yang ditunggu. Jangan menunggu semuanya lengkap — dan jangan
memulai kelompok ⛔ dengan menebak.

---

## 2. Checklist Phase 5

| # | Item checklist | Status | Keterangan |
|---|---|---|---|
| 1 | Business sudah dipahami | ✅ **TERPENUHI** | Proses, peran, aturan, aliran nilai uang terpetakan dari export rule XML (**2.634 berkas**) dan dikonfirmasi pemilik bisnis. Pendorong bisnis dikonfirmasi (I-01) |
| 2 | Requirement lengkap | ⚠️ **SEBAGIAN** | 162 requirement bernomor; 137 tervalidasi, **25 tidak dapat difinalkan** karena menunggu 10 artefak pihak luar |
| 3 | BRD selesai | ✅ **TERPENUHI** | [`BRD.md`](BRD/BRD.md) — 23 bab, siap dipresentasikan ke manajemen |
| 4 | Risiko terdokumentasi | ✅ **TERPENUHI** | **19 risiko** di [`Steering/16-RISK-ANALYSIS.md`](Steering/16-RISK-ANALYSIS.md) — `R-13`…`R-15` diserap dari BRD, `R-16`…`R-19` dari verifikasi bukti (`D-38`) |
| 5 | Dependency diketahui | ✅ **TERPENUHI** | 3 rantai kritis internal + 6 pihak eksternal teridentifikasi |
| 6 | Database dipahami | ⚠️ **SEBAGIAN** | 245 tabel dan 5 schema terpetakan. **DDL — tipe kolom, panjang, index, constraint — tidak diketahui** (R-08) |
| 7 | Integration dipahami | ⚠️ **SEBAGIAN** | **21 Connect REST** — 12 terpetakan lengkap, **9 baru ditemukan** dan belum dianalisis setara (`D-73`). **6 API pengganti DB Link belum ada kontraknya** (R-03). **4 Service REST masuk** belum pernah diaudit |
| 8 | Edge case diketahui | ⚠️ **SEBAGIAN** | 10 kasus tepi terdokumentasi. **Kasus tepi di dalam 64 procedure yang belum terbaca tidak diketahui sama sekali** (R-01) |
| 9 | Tidak ada requirement yang ambigu | ⛔ **TIDAK TERPENUHI** | 25 requirement masih ⚠️, berpangkal pada 9 ambiguitas. Lihat §4 |

**Empat dari sembilan item terpenuhi penuh. Empat sebagian. Satu tidak terpenuhi.**

Instruksi Phase 5 menyatakan: *"Jika checklist belum terpenuhi, kembali ke phase sebelumnya."*
Itu **sudah dilakukan** — Phase 1 dan Phase 3 dijalankan ulang pada sesi ini, dan hasilnya
adalah temuan bahwa **sembilan item terbuka itu tidak dapat ditutup dengan kembali ke phase
manapun**, karena informasinya tidak ada di export XML dan tidak ada pada pemilik project.
Penutupannya adalah **tindakan berjadwal dengan pihak ketiga**, bukan pekerjaan analisis. Lihat §7.

---

## 3. Requirement Completeness

| Kategori | Jumlah | Tervalidasi | Menunggu artefak |
|---|---|---|---|
| Functional Requirement | 42 | 31 | 11 |
| Non Functional Requirement | 38 | 36 | 2 |
| Business Rules | 23 | 21 | 2 |
| Validation Rules | 11 | 9 | 2 |
| Data Requirement | 11 | 6 | 5 |
| Integration Requirement | 15 | 12 | 3 |
| Security Requirement | 15 | 15 | 0 |
| Error Handling Requirement | 7 | 7 | 0 |
| **Total** | **162** | **137** | **25, atas 10 artefak** |

**Kelengkapan: 84,6%.** Angka ini jujur tetapi menyesatkan bila dibaca sendiri, karena
**25 requirement yang tertahan bukan 25 hal kecil yang tersebar** — ia terkonsentrasi pada
**jantung nilai uang klaim**. Kelengkapan diukur per kelompok modul:

| Kelompok | Kelengkapan |
|---|---|
| Fondasi, kerangka UI, security, error handling | **100%** |
| Jalur klaim awal (registrasi, objek, coverage, spreading) | **100%** |
| Baca-saja (inbox, pencarian, laporan) | **~95%** (tertahan R-06 untuk penyaringan status) |
| **Settlement, komite, PLA/DLA, akseptasi, salvage** | **~40%** — struktur diketahui, **logika tidak** |
| **Scheduler** | **0%** — bahkan ukurannya belum diketahui |

---

## 4. Remaining Ambiguity

Sembilan ambiguitas. Kolom terakhir menyatakan apakah ia dapat ditutup dengan analisis lebih
lanjut — dan jawabannya **tidak, untuk semuanya**.

| # | Ambiguitas | Dampak bila dibiarkan | Dapat ditutup dengan analisis? |
|---|---|---|---|
| 1 | **Isi 64 procedure & function** (R-01) | 5 modul jantung nilai uang tidak dapat diselesaikan. Kasus tepi tidak diketahui | ❌ Source tidak ada di export. **Belum pernah dilihat siapa pun di tim ini** |
| 2 | **Daftar job terjadwal** (R-02) | **Kegagalan yang tidak terlihat** — proses otomatis yang berjalan hari ini tidak akan berjalan, tanpa ada yang menyadarinya sampai sesuatu tidak terkirim | ❌ Export tidak memuat rule Agent maupun Queue Processor |
| ~~3~~ | ~~**Arti kode status**~~ (R-06) | — | ✅ **DITERIMA** — **33 kode `1134`–`1166`**, bukan 10 kode `1142`–`1151` |
| 4 | **Matriks penjenjangan komite** (D-14) | Jumlah jenjang dan ambangnya tidak diketahui — modul komite berhenti di struktur | ❌ Isi `POOLDATA.EMAILKOMITE` tidak ada di export |
| 5 | **Aturan 3 router penugasan** (R-04) | Siapa menerima tugas apa tidak diketahui — inti modul FR-B6 | ❌ Tidak ada di export. Polanya bisa **ditebak** dari 2 router yang ada, tetapi aturan persisnya tidak |
| 6 | **Kontrak 6 API pengganti DB Link** (R-03) | 64 pemakaian DB Link tidak punya pengganti | ❌ API-nya belum ada; dibangun tim lain |
| 7 | **DDL tabel** (R-08) | Desain skema baru **harus menebak** tipe, panjang, index, constraint | ❌ Tidak ada di export |
| 8 | **Lama retensi audit** (D-28) | Strategi arsip dan partisi tidak dapat difinalkan | ❌ Ketentuan Compliance |
| 9 | **Angka RPO / RTO** (D-29) | Tidak diketahui apakah standar korporat sejalan dengan tuntutan 24/7 | ❌ Kebijakan tim infra |

**Satu ambiguitas patut diperhatikan khusus:** `ValidasiSisaTSI` (VR-10). Namanya menunjukkan
adanya **aturan validasi sisa TSI yang tidak muncul di analisis aturan bisnis manapun** — bukan
sekadar detail yang hilang, melainkan **petunjuk bahwa masih ada aturan bisnis yang belum kita
ketahui keberadaannya.**

---

## 5. Technical Risk

| ID | Risiko | Dampak | Kemungkinan | Status |
|---|---|---|---|---|
| **R-01** | Source 64 procedure & function tidak tersedia | Tinggi | **Terjadi** | Terbuka — sudah diminta |
| **R-02** | Job terjadwal tidak diketahui | Tinggi | **Terjadi** | Terbuka — sudah diminta |
| **R-03** | 6 API pengganti DB Link belum ada | Tinggi | Tinggi | Terbuka — sudah disampaikan |
| **R-04** | 3 router penugasan tidak ada di export | Sedang | **Terjadi** | Terbuka |
| ~~**R-06**~~ | ~~Arti kode status tidak diketahui~~ | — | — | ✅ **TERTUTUP** |
| **R-07** | 50 activity dipanggil tapi tidak ada di export | Sedang | **Terjadi** | Terbuka |
| **R-08** | DDL tabel tidak tersedia | Sedang | **Terjadi** | Terbuka |
| **R-09** | Sistem sumber masih aktif berubah — **124 activity diubah pada 2026** | Sedang | Tinggi | Melekat pada Strangler Fig |
| **R-10** | Data ganda relasional versus JSON, tanpa mekanisme pemeriksa konsistensi | Sedang | Sedang | Terbuka |
| **R-11** | Tim mempelajari **Go, React, dan TypeScript sekaligus** | Tinggi | Tinggi | Melekat pada D-09 dan D-23 |
| **R-12** | Zona waktu bergeser saat migrasi data — mengubah **hasil validasi** aturan tanggal | Sedang | Sedang | Terbuka |

**Risiko teknis paling berbahaya adalah R-02**, bukan R-01. R-01 **terlihat**: modul yang
terhalang tidak akan selesai, dan itu akan disadari. R-02 **tidak terlihat**: sistem baru akan
tampak berjalan normal sementara ada proses otomatis yang diam-diam tidak lagi berjalan —
kandidat paling mungkin adalah pengingat SLA, eskalasi tugas menggantung, penutupan klaim yang
lewat tenggat, pengiriman batch ke kasir, dan pelaporan SLIK OJK.

---

## 6. Business Risk

| ID | Risiko | Dampak | Kemungkinan | Status |
|---|---|---|---|---|
| **R-05** | **Jadwal tidak sepadan dengan ukuran pekerjaan.** Target akhir September 2026 versus perkiraan terukur **9–15 bulan dengan tim 5–6 orang** | Tinggi | Tinggi | Diterima manajemen (D-30) |
| **R-13** 🆕 | **Tenggat tidak punya pemaksa eksternal** | Sedang | — | **Baru — perlu diangkat ke manajemen** |
| **R-14** 🆕 | **Ukuran keberhasilan tidak dapat dibuktikan untuk 5 modul** | Tinggi | **Terjadi** | **Baru — perlu diangkat ke manajemen** |
| **R-15** 🆕 | **Waktu user bisnis untuk UAT belum dialokasikan** | Sedang | Tinggi | **Baru** |

### R-13 · Tenggat tidak punya pemaksa eksternal

I-01 mengonfirmasi pendorong migrasi adalah **kemandirian teknologi dan maintainability** — dan
**bukan** lisensi yang berakhir, **bukan** biaya lisensi, **bukan** arahan grup.

Artinya target akhir September 2026 (D-30) adalah **target internal manajemen**, tanpa tanggal
yang dipaksakan pihak luar dan tanpa penalti kontraktual. Penjadwalan ulang sepenuhnya berada di
dalam kendali manajemen Sinarmas.

**Kenapa ini risiko dan bukan kabar baik.** Menerima risiko R-05 apa adanya akan wajar bila ada
tenggat eksternal yang tidak bisa digeser — mengejar tanggal dengan mutu yang dikompromikan
kadang memang pilihan yang benar. Tetapi di sini **tidak ada tenggat seperti itu**, sehingga
risiko yang diterima **tidak dibayar oleh manfaat apa pun**. Yang tersisa hanya biayanya: mutu
yang dikompromikan pada sistem yang menangani uang klaim.

**Penanganan.** Sampaikan ke manajemen bahwa penjadwalan ulang tidak berbiaya eksternal, disertai
angka R-05 dan urutan pengerjaan berbasis dependensi sebagai dasar keputusan. **Bukan** untuk
menolak D-30 — yang sudah ditegaskan ulang dan tetap menjadi dasar Migration Strategy — melainkan
agar keputusan mempertahankannya diambil dengan mengetahui bahwa alternatifnya tersedia tanpa
penalti.

### R-14 · Ukuran keberhasilan tidak dapat dibuktikan untuk 5 modul

I-02 menetapkan ukuran keberhasilan: **kesetaraan fungsional dengan Pega**. I-06 menetapkan
gerbang pertama cutover: **uji kesetaraan otomatis**.

Keduanya menuntut satu hal yang sama: **mengetahui apa yang seharusnya menjadi hasil yang benar.**
Untuk FR-B5, FR-B7, FR-B9, FR-B10, dan FR-B12, hasil yang benar ditentukan oleh **64 procedure
yang logikanya belum pernah dilihat siapa pun** (R-01).

> **Kesetaraan atas logika yang tidak diketahui tidak dapat dibuktikan, hanya diduga.**

Ini menaikkan derajat R-01. R-01 bukan sekadar penghambat jadwal — ia menghambat **kemampuan
menyatakan project ini berhasil menurut ukuran yang dipilih manajemen sendiri.** Bila source
tidak pernah datang, satu-satunya jalan adalah menyimpulkan perilaku dari perbandingan masukan
dan keluaran di staging: jauh lebih lambat, dan **tidak menjamin kelengkapan kasus tepi** —
tepat pada lima modul yang mengurus nilai uang klaim.

**Penanganan.** Angkat R-01 ke manajemen bukan sebagai permintaan teknis kepada DBA, melainkan
sebagai **prasyarat ukuran keberhasilan**. Bila ia tidak dapat dipenuhi, ukuran keberhasilan untuk
kelima modul itu harus diubah secara sadar — bukan diam-diam diturunkan menjadi "tampaknya jalan".

### R-15 · Waktu user bisnis untuk UAT belum dialokasikan

I-06 menetapkan **UAT user bisnis sebagai gerbang kedua** setiap cutover modul. UAT itu menuntut
ketersediaan user dari peran yang memakai modul tersebut — untuk ~20 inbox berbasis peran dan
5 layar transaksi utama — sementara user yang sama sedang menjalankan **operasional klaim harian**.

Alokasi waktu ini belum tercatat di manapun, dan pada jadwal D-30 ia tidak punya ruang sama
sekali.

**Penanganan.** Alokasikan waktu user bisnis secara resmi per modul, dan masukkan sebagai
constraint project — bukan sebagai detail pelaksanaan yang diasumsikan tersedia.

---

## 7. Missing Information

Sepuluh artefak. Status: **sudah diminta, menunggu jawaban** (I-05).

| # | Artefak yang ditunggu | Dari | Memblokir | Cara memenuhi |
|---|---|---|---|---|
| 1 | Source **64 procedure & function** — daftar lengkap di [`Steering/19-GAP-EXPORT-DETAIL.md`](Steering/19-GAP-EXPORT-DETAIL.md) | DBA | FR-B5, B7, B9, B10, B12 · **R-14** | Query siap pakai sudah disediakan di berkas itu |
| 2 | Export **Rule-Agent / Queue Processor / Job-Scheduler** | Tim Pega | FR-S6 | Bila tidak tersedia: wawancarai tim operasional + periksa log Pega untuk proses tanpa pemicu pengguna |
| 3 | **DDL lengkap** `POOLDATA`, `DATAPEGA`, `GENERAL` + statistik ukuran tabel | DBA | Desain skema, strategi index & partisi | Satu permintaan |
| 4 | Isi master **`V_STS_CLAIM`** | DBA | FR-W6, seluruh layar berstatus | **Satu query.** Permintaan paling mudah dipenuhi di seluruh daftar ini |
| 5 | Isi **`POOLDATA.EMAILKOMITE`** | DBA | FR-B7, FR-W5, BR-16, BR-18 | Satu query — rincian di [`Steering/20-DETAIL-KOMITE-DBLINK.md`](Steering/20-DETAIL-KOMITE-DBLINK.md) |
| 6 | Konfirmasi aturan **`SetEmailKomite`** yang bercampur nama orang dan percabangan hostname | Tim bisnis | FR-B7 | Wawancara tim bisnis |
| 7 | Export **3 router + 45 activity** yang hilang | Tim Pega | FR-B6, FR-S3, VR-10 | Prioritaskan `SendEmailNotification`, `ValidasiSisaTSI`, `SetKasir_Act`, `SetTicket` |
| 8 | **Kontrak 6 API** pengganti DB Link | Tim pemilik sistem | FR-S4, INT-10 | Inventaris lengkap sudah disiapkan di [`Steering/20-DETAIL-KOMITE-DBLINK.md`](Steering/20-DETAIL-KOMITE-DBLINK.md) |
| 9 | **Lama retensi** data audit | Compliance | NFR-29 | Satu pertanyaan kebijakan |
| 10 | Angka **RPO / RTO** korporat | Tim infra | NFR-25 | Satu pertanyaan kebijakan |

**Dua tindakan tambahan yang tidak berupa artefak:**

| # | Tindakan | Kepada |
|---|---|---|
| 11 | Sepakati **pembekuan perubahan Pega** pada modul yang sedang dimigrasi; perubahan mendesak yang tetap dilakukan **wajib dicatat** dan diterapkan juga di sistem baru (R-09) | Tim Pega + manajemen |
| 12 | Mulai **pelatihan Go, React, TypeScript** — waktu belajar tidak dapat dipercepat dengan menambah orang (R-11) | Tim pengembang |

> **Sepuluh dari dua belas tindakan bergantung pada pihak di luar tim pengembang.** Waktu
> tunggunya **tidak dapat dipercepat dengan menambah developer** — dan karena itu tidak boleh
> diperlakukan sebagai pekerjaan yang bisa dikejar nanti.

### Tindakan yang tepat sekarang, karena permintaan sudah terkirim

Karena permintaan **sudah dikirim** (I-05), mengirim ulang tidak menambah apa pun. Yang
menentukan sekarang adalah dua hal berikut — dan tanpa keduanya, "sudah diminta" punya dampak
jadwal yang **sama persis** dengan "belum diminta":

1. **Tanggal komitmen per artefak.** Setiap pihak memberi tanggal, bukan "sedang diproses".
   Tanpa tanggal, ketergantungan ini tidak dapat dimasukkan ke rencana apa pun.
2. **Jalur eskalasi yang disepakati.** Ke siapa, dan pada hari keberapa, bila tanggal komitmen
   tidak dipenuhi. Ditetapkan **sekarang**, bukan saat tanggalnya sudah lewat.

Empat artefak (#4, #5, #3, #1 dari DBA) adalah **query dan export**, bukan pekerjaan
pengembangan. Dua di antaranya — #4 dan #5 — masing-masing **satu query**, dan keduanya
memblokir modul komite dan seluruh layar berstatus. Ini yang harus dieskalasi lebih dulu:
biaya pemenuhannya paling kecil, dampaknya paling besar.

---

## 8. Recommendation

### 8.1 Boleh dimulai sekarang

1. **Gelombang 1 — Fondasi:** FR-F1 → FR-F2 → FR-F5 → FR-F3 → FR-F4. Tidak ada yang terhalang.
2. **Gelombang 2 — Kerangka UI:** FR-U1 → FR-U2, dan FR-S5 (jejak audit). **FR-U2 adalah
   investasi paling menentukan di frontend** — 268 section memakai pola grid yang sama.
3. **Tahap 2 Migration Strategy — modul baca-saja lebih dulu.** Inbox, pencarian, detail, dan
   laporan. Tiga alasan sekaligus: tidak melanggar P-1 karena tidak menulis apa pun; menjadi
   pembuktian arsitektur yang nyata; dan **tim belajar Go dan React pada pekerjaan yang
   kesalahannya tidak merusak data** (menangani R-11).
4. **FR-S8 — perkakas uji kesetaraan.** Dibangun **bersamaan dengan** modul baca-saja, bukan
   sesudahnya. Ia gerbang pertama setiap cutover (I-06) dan satu-satunya cara membuktikan ukuran
   keberhasilan (I-02). Bila dibangun belakangan, cutover modul pertama akan tertahan olehnya.
5. **Pelatihan Go, React, TypeScript** — dimulai hari ini, paralel dengan semua di atas.

### 8.2 Jangan dimulai sebelum artefaknya datang

**FR-B5, FR-B7, FR-B9, FR-B10, FR-B12** — jantung nilai uang klaim. Memulainya dengan menebak isi
64 procedure akan menghasilkan kode yang **kelihatan selesai tetapi tidak dapat dibuktikan
setara** — dan pada modul yang mengurus uang klaim, itu lebih buruk daripada belum dikerjakan,
karena kesalahannya baru terlihat setelah uang berpindah.

**FR-S6 (scheduler)** — ukurannya bahkan belum diketahui. Membangunnya sekarang berarti membangun
wadah tanpa tahu apa isinya.

### 8.3 Yang harus diangkat ke manajemen

Tiga hal, dan ketiganya keputusan manajemen — bukan keputusan teknis:

| # | Yang diangkat | Kenapa manajemen |
|---|---|---|
| 1 | **R-01 sebagai prasyarat ukuran keberhasilan, bukan permintaan teknis** (R-14) | Bila source tidak pernah datang, ukuran keberhasilan untuk 5 modul harus diubah secara sadar. Itu keputusan manajemen, bukan kompromi yang diambil diam-diam oleh tim |
| 2 | **Tenggat tidak punya pemaksa eksternal** (R-13) | Penjadwalan ulang tersedia tanpa penalti kontraktual. Manajemen berhak mengetahui itu sebelum menegaskan D-30 kembali |
| 3 | **Alokasi waktu user bisnis untuk UAT** (R-15) | Menuntut waktu orang yang sedang menjalankan operasional harian. Hanya manajemen yang dapat mengalokasikannya |

### 8.4 Ambang untuk menyatakan READY

Verdict berubah menjadi **READY** ketika, dan hanya ketika:

- [ ] Artefak #1 (source 64 procedure) **diterima dan dibaca** — bukan sekadar dijanjikan
- [ ] Artefak #2 (daftar job terjadwal) diterima, **atau** daftarnya direkonstruksi dari wawancara tim operasional dan disetujui business owner sebagai lengkap
- [ ] Artefak #3, #4, #5 (DDL, `V_STS_CLAIM`, `EMAILKOMITE`) diterima
- [ ] Artefak #7 (3 router + 45 activity) diterima, **atau** aturan routing digali dari tim operasional **dan dibuktikan** dengan membandingkan hasil penugasan pada data historis
- [ ] Artefak #8 (kontrak 6 API) disepakati beserta **tanggal ketersediaannya**
- [ ] Artefak #9, #10 (retensi, RPO/RTO) dikonfirmasi
- [ ] Pembekuan perubahan Pega disepakati tertulis
- [ ] Tiga hal di §8.3 sudah diputuskan manajemen
- [ ] Ukuran keberhasilan per modul dinyatakan **dapat dibuktikan** (R-14 tertutup)

Sembilan baris. **Tujuh di antaranya tidak berada di tangan tim pengembang.**

Sampai baris-baris itu tercentang, pekerjaan yang boleh berjalan adalah §8.1 — dan itu bukan
pekerjaan kecil: fondasi, kerangka UI, seluruh modul baca-saja, jalur klaim awal, dan perkakas
uji kesetaraan. Cukup untuk berbulan-bulan, dan seluruhnya berada sepenuhnya di dalam kendali
tim.

---

## 9. Kesimpulan

**Pemahaman dan perencanaan: selesai.** Tidak ada lagi nilai yang bisa diambil dari membaca
export XML ini. Analisis sudah mencapai batasnya.

**Kesiapan implementasi: terhalang, dan bukan oleh tim pengembang.** Sepuluh artefak ditunggu;
permintaan sudah terkirim; yang belum ada adalah **tanggal komitmen** dan **jalur eskalasi**.

**Yang paling menentukan sekarang bukan menulis kode, melainkan mengejar sepuluh artefak itu
sampai punya tanggal.** Empat di antaranya cuma query ke database — dan dua dari empat itu
masing-masing satu query yang memblokir seluruh modul komite dan setiap layar berstatus.

**Boleh mulai hari ini:** fondasi, kerangka UI, modul baca-saja, jalur klaim awal, perkakas uji
kesetaraan, dan pelatihan tim.

**Jangan mulai dengan menebak:** lima modul jantung nilai uang klaim, dan scheduler.
