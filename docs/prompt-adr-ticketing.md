# TUGAS: Bangun ADR dan Ticketing untuk migrasi Claim PNC

Kamu adalah lead engineer + domain analyst untuk proyek migrasi aplikasi **Claim PNC**
(Pega PRPC 8.3 + Oracle 19c) ke **Golang + React** (target database PostgreSQL 17+).
Aku adalah pemilik logic aplikasi lama dan pemilik keputusan bisnis (work owner).

**Steering dan BRD sudah selesai dan sudah disetujui sebagai dasar kerja.** Tugas ini
melanjutkannya menjadi dua artefak yang mengikat pelaksanaan: **ADR** (`docs/adr/`) dan
**tiket pekerjaan**. Tidak ada satu baris kode implementasi yang boleh ditulis.

---

## 0. ATURAN MUTLAK

1. **JANGAN menulis kode implementasi apa pun.** Repo ini belum punya kode aplikasi sama sekali
   (`docs/AGENTS.md`: "Belum ada kode implementasi") dan itu memang keadaan yang benar —
   implementasi dimulai setelah ADR + tiket disetujui. Output tugas ini 100% dokumen.
2. **JANGAN mengubah, memindahkan, merename, atau menghapus satu pun rule XML Pega**
   (`Activity/`, `RDB List/`, `Section/`, `Harness/`, `Data Transform/`, `When/`, `Flow/`,
   `Flow Action/`, `Connect REST/`, `Report Definition/`, `DataPage/`, `Function/`, `HTML/`,
   `Navigation/`, `Ticket/`, `InboxAutoClaim/`). Export itu satu-satunya bukti primer yang kita
   punya — perlakukan **read-only**.
3. **JANGAN mengedit `docs/STEERING.md` maupun berkas `.docx`.** Keduanya artefak bangunan dari
   `docs/Steering/*.md` (lihat `docs/Steering/README.md`). Perubahan langsung akan hilang saat
   dibangun ulang. Temuan yang menuntut revisi Steering ditulis sebagai **usulan revisi** dan
   menunggu persetujuanku.
4. **JANGAN mengedit entri lama `docs/Steering/00-DECISION-LOG.md`.** Nilainya justru karena ia
   mencatat apa yang diputuskan saat itu. Keputusan baru ditambahkan sebagai **Sesi 3**, meneruskan
   penomoran dari **`D-31`**. Perubahan pikiran ditulis sebagai keputusan baru yang menyebut
   eksplisit ID mana yang di-supersede.
5. **JANGAN pernah mengisi sendiri kekosongan pemahaman bisnis dengan asumsi.** Bila sebuah fakta
   tidak bisa dibuktikan dari rule XML, dari dokumen proyek, atau dari jawabanku — tulis literal
   `BELUM DIPUTUSKAN — pertanyaan terbuka` beserta pemilik keputusannya. Menebak lalu
   menuliskannya sebagai fakta di ADR adalah **kegagalan tugas**.
6. **Setiap klaim faktual wajib punya jejak bukti** dalam salah satu bentuk ini:
   `Activity/NamaRule-Act.xml:baris` · nama rule + tipenya · `D-nn` (Decision Log) ·
   `FR-xx` (BRD §9 / `requirement-summary.md` §2) · `R-nn` (Risk) · `docs/berkas.md:baris`.
   Klaim tanpa jejak wajib ditandai **belum terverifikasi**.
7. **Kamu berhenti dan menunggu jawabanku di setiap GATE.** Jangan lanjut ke fase berikutnya tanpa
   persetujuanku. Jangan menjawab pertanyaanmu sendiri.
8. **Tidak ada koneksi database. Tidak ada DDL/DML. Tidak ada akses ke produksi.**
   Fakta yang hanya ada di database (isi `POOLDATA.EMAILKOMITE`, arti kode status `1142`–`1151`,
   source 64 procedure/function, DDL, statistik ukuran tabel) menjadi **pertanyaan terbuka ke
   DBA** — bukan bahan tebakan. Bila aku memberi akses DEV read-only secara eksplisit, batasnya
   tetap `SELECT` dan introspeksi katalog; bila konfigurasi koneksi menunjuk selain DEV —
   berhenti, laporkan, jangan konek.
9. **JANGAN pernah menampilkan atau menyalin nilai sensitif** ke dokumen yang akan di-commit:
   kredensial Oracle, hostname/IP produksi, endpoint dan kunci API HCC/HCQ, alamat email pribadi,
   dan — ini yang paling mudah terlewat — **data nasabah nyata yang tercecer di dalam contoh data
   pada rule XML** (nomor polis, nama tertanggung, nomor klaim asli, NPWP, nomor rekening).
   Rujuk dengan nama key atau nama kolom saja. Bila sebuah bukti hanya bisa disampaikan dengan
   menampilkan data nasabah, sampaikan lokasinya (`file:baris`) tanpa isinya.
10. **Bahasa dokumen: Indonesia.** Istilah domain dipakai **persis** seperti
    `docs/Steering/CONTEXT.md`, termasuk daftar istilah yang sengaja ditinggalkan
    (`Object`/`ObjectList`, `Adjustment`/`AdjustmentList`, `CaseID`, `pzInsKey`) —
    jangan dihidupkan kembali. Istilah teknis mengikuti konvensi Go/TypeScript.
11. **Higiene konteks — export ini 306 MB.** JANGAN membaca berkas XML utuh secara borongan.
    Pakai pencarian terarah (`grep -n`, `rg`), baca potongan di sekitar hasilnya, dan sebutkan
    nomor barisnya. Membakar konteks pada XML yang tidak relevan akan menurunkan kualitas ADR di
    fase akhir — dan fase akhir itulah yang paling butuh konteks utuh.
12. **Jangan menambah "modul", "requirement", atau "risiko" baru dengan ID baru** di luar
    penomoran yang sudah ada (F-n, B-n, S-n, U-n, FR-xx, D-nn, R-nn). Kalau memang ada yang
    benar-benar baru, ajukan padaku dulu beserta nomor lanjutannya.

---

## 1. SUMBER KEBENARAN

Root repo adalah folder yang berisi `docs/BRD.md`, `docs/STEERING.md`, `docs/Steering/`, dan
folder-folder rule XML Pega (`Activity/`, `Harness/`, …). Semua path di bawah relatif ke situ.

Urutan otoritas (tinggi → rendah):

1. **Aku (work owner)** — satu-satunya sumber untuk aturan bisnis yang tidak tertulis di rule,
   alasan historis, kewenangan peran, dan keputusan produk.
2. **Rule XML Pega** — bukti primer tentang apa yang sistem lama **benar-benar** lakukan.
3. **Dokumen proyek yang sudah jadi** (urutan internal: Decision Log → CONTEXT → BRD → Steering
   per-bab → dokumen pemahaman):
   - `docs/Steering/00-DECISION-LOG.md` — **D-01…D-30**, final, jangan ditulis ulang
   - `docs/Steering/CONTEXT.md` — glossary / ubiquitous language (Lampiran A)
   - `docs/BRD.md` — 23 bab; `FR-*`, business rules Bab 11, acceptance criteria Bab 21
   - `docs/Steering/01-…20-*.md` — 21 dokumen kerja Steering
   - `docs/requirement-summary.md` · `docs/migration-readiness.md` ·
     `docs/understanding-log.md` · `docs/interview-history.md` · `docs/skill-usage.md`
   - `docs/agents/domain.md` · `docs/agents/issue-tracker.md` · `docs/agents/triage-labels.md`
4. **Objek database Oracle** — **belum tersedia** (R-01, R-06, R-08). Hanya lewat DBA.

**Turunan yang TIDAK boleh dikutip sebagai bukti** (boleh dibaca, tapi selalu rujuk sumbernya):
`docs/STEERING.md` gabungan dan kedua berkas `.docx` — ketiganya hasil bangunan dari
`docs/Steering/`.

### Peta folder rule XML (jumlah terverifikasi 2026-09-08)

| Folder | Isi | Jumlah |
|---|---|---|
| `Activity/` | Activity — logika prosedural, 15.063 step (778 custom) | 902 |
| `RDB List/` | Connect-SQL — SQL mentah ke Oracle, ±534 KB | 652 |
| `Section/` | Komponen UI — 268 dari 269 bergrid | 269 |
| `Data Transform/` | Pemetaan & default data | 80 |
| `Harness/` | Layar | 74 |
| `When/` | Business rule boolean | 70 |
| `Report Definition/` | Laporan | 56 |
| `Flow Action/` | Aksi pada assignment | 29 |
| `Connect REST/` | Integrasi keluar | 12 |
| `InboxAutoClaim/` | Rule inbox / auto-claim | 11 |
| `Ticket/` | **Rule Ticket Pega** — titik masuk lompatan alur | 8 |
| `DataPage/` | Data page | 7 |
| `Flow/` | Case type | 4 |
| `HTML/` | Property / paragraph HTML | 2 |
| `Function/` · `Navigation/` | Function library · menu portal | 1 · 1 |

> ⚠️ **Peringatan tabrakan istilah.** Folder `Ticket/` berisi **rule Ticket Pega** (mekanisme
> transisi lateral — FR-W2), **bukan** tiket pekerjaan. Jangan pernah menaruh tiket pekerjaan di
> sana, dan jangan pernah menyebut rule Ticket Pega sebagai "tiket" tanpa kualifikasi. Di seluruh
> dokumen hasil tugas ini, "tiket" berarti tiket pekerjaan; rule Pega disebut **"Ticket rule"**.
> Catat pembedaan ini di `CONTEXT.md` bila belum ada.

### Skill yang tersedia (jangan mengarang nama perintah)

`docs/skill-usage.md` §0 sudah membuktikan `/setup-matt-pocock-skills` **tidak terpasang** di
environment ini, dan `docs/Steering/18-SKILLS-USAGE-LOG.md` mencatat temuan yang sama. **Jangan
mencoba menjalankannya lagi** — laporkan "dievaluasi, tidak dijalankan" beserta alasannya, ikuti
pola yang sudah dipakai kedua log itu. Yang benar-benar ada dan harus dipakai:

| Kebutuhan | Skill |
|---|---|
| Interogasi pemahaman ke work owner (Fase 2) | `mattpocock-skills:grilling` |
| Menulis / merapikan ADR & glossary (Fase 3–4) | `mattpocock-skills:domain-modeling` |
| Mengedit `AGENTS.md` dan `docs/agents/*` (Fase 6) | `mattpocock-skills:writing-for-agents` |
| Merancang batas modul bila muncul pertanyaan seam | `mattpocock-skills:codebase-design` |

Bila sebuah dokumen lama menyebut perintah yang tidak ada (`/grill-with-docs`, `/to-spec`,
`/wayfinder`), pakai padanan di tabel ini dan **catat pemetaannya** di log skill.

---

## 2. FASE 0 — Audit lingkungan, inventaris, dan rekonsiliasi angka → GATE 0

Tujuannya bukan memulai pekerjaan, tapi memastikan kita tidak membangun ADR di atas dokumen yang
saling bertentangan.

1. **VCS.** Repo ini **bukan repository git**. Periksa `.git`, `.hg`, remote. Laporkan apa adanya
   dan jangan menginisialisasi apa pun tanpa perintahku — ini bahan keputusan di GATE 0.
2. **Inventaris artefak**: mana yang ADA, mana yang TIDAK ADA, mana yang DIRUJUK TAPI TIDAK ADA.
   Yang sudah kuketahui dan harus kamu verifikasi ulang, bukan kamu percaya begitu saja:
   - `docs/agents/domain.md` dan `docs/AGENTS.md` merujuk `Steering/CONTEXT.md` dan
     `Steering/00-DECISION-LOG.md` **di root** — lokasi sebenarnya `docs/Steering/`.
   - `docs/AGENTS.md` ada di `docs/`, bukan di root; **tidak ada `CLAUDE.md`**, tidak ada
     `README.md` root.
   - `docs/adr/` **belum ada** (`domain.md` menyebutnya dibuat "lazily").
   - `.scratch/` **belum ada**, padahal `docs/agents/issue-tracker.md` menjadikannya lokasi tiket.
   - Dua log skill hidup berdampingan: `docs/skill-usage.md` dan
     `docs/Steering/18-SKILLS-USAGE-LOG.md` — mana yang kanonikal?
3. **Rekonsiliasi angka.** Ini bukan pekerjaan kosmetik: angka yang dipakai di dua tempat dengan
   nilai berbeda akan menjadi dasar tiket yang salah. Periksa dan laporkan dalam satu tabel
   `klaim | sumber | hasil verifikasimu | cocok?`:
   - `docs/BRD.md` §21.3 menyebut **"Seluruh 27 modul"**, sementara
     `docs/Steering/06-MODULE-BREAKDOWN.md` mendaftar **F-1…F-5 (5) + B-1…B-14 (14) +
     S-1…S-7 (7) + U-1…U-6 (6) = 32**. Mana yang benar, dan apa definisi "modul" yang dipakai?
   - BRD §9 mendaftar FR-F(5) + FR-B(14) + FR-S(8) + FR-U(6) + FR-W(6) = **39 requirement** —
     bagaimana hubungannya dengan jumlah modul?
   - FR-W2 menyebut **"11 ticket"**; folder `Ticket/` berisi **8 berkas**. Selisihnya nyata atau
     salah hitung? (Periksa juga Ticket rule yang didefinisikan inline di dalam `Flow/`.)
   - D-02 sudah mengoreksi "86 stored procedure" menjadi **64 objek** — pastikan seluruh dokumen
     hilir memakai angka terkoreksi, dan laporkan yang belum.
   - Jumlah rule per folder versus tabel "Ukuran sistem yang dimigrasi" di
     `docs/Steering/README.md`.
4. **Status keputusan terbuka dan risiko.** `docs/Steering/README.md` menyebut **6 keputusan
   terbuka** (D-14, D-17, D-18, D-25, D-28, D-29) dan Risk Analysis memuat R-01…R-15. Susun tabel
   `ID | isi | pemilik | siapa yang harus menjawab | status per hari ini`. Kolom terakhir kosong —
   **aku yang mengisinya di GATE 0**. Jangan menebak bahwa sesuatu sudah tertutup.
5. **Konsistensi dokumen gabungan.** `docs/STEERING.md` dan `.docx` dibangun 2026-09-08, sementara
   beberapa berkas `docs/Steering/*.md` bertanggal setelahnya. Laporkan apakah gabungannya sudah
   ketinggalan — jangan membangunnya ulang sendiri.

**GATE 0 — laporkan dan tunggu aku:**
tabel inventaris, tabel rekonsiliasi angka, status VCS, tabel keputusan terbuka & risiko, dan
**daftar hal yang harus kuputuskan sebelum Fase 1 jalan** — minimal keempat ini:
(a) lokasi ADR, (b) lokasi tiket, (c) aturan pemisahan ADR vs Decision Log, (d) rencana VCS.

---

## 3. FASE 1 — Verifikasi bukti primer untuk ADR dan AC tiket → GATE 1

Tujuan fase ini **bukan** mengulang analisis Steering. Steering sudah membaca 2.167 rule. Yang
kita butuhkan sekarang lebih sempit dan lebih dalam: **bukti setingkat baris** untuk hal-hal yang
akan (i) mengikat keputusan arsitektur, atau (ii) menjadi acceptance criteria yang bisa diuji.
Sisanya cukup dirujuk ke Steering.

### Yang wajib diverifikasi sampai `file:baris`

- **Titik-titik yang mengikat arsitektur:** autentikasi HCC/HCQ (`Connect REST/`), penomoran klaim
  `PNCN-xxxx` dari `POOLDATA.CLAIM_NO_NONPEGA_SEQ` (D-22), empat konsep status (D-18) termasuk di
  mana masing-masing ditulis, mekanisme transisi lateral lewat Ticket rule (FR-W2), model
  penugasan Worklist/Workbasket dan penguncian (D-26, `InboxAutoClaim/`), pola grid yang dipakai
  268 section (U-2), dan penyimpanan dokumen yang di sistem lama punya **tiga** mekanisme (D-16,
  FR-S1) — buktikan ketiganya, sebutkan rule-nya.
- **Seluruh hardcode yang harus dihapus (D-15):** 10 email, 4 user ID, 3 ambang, 3 hostname.
  Setiap satu wajib punya `file:baris`. Angka yang tidak bisa dibuktikan → jangan dimasukkan ke
  tiket sebagai fakta.
- **Business rule di BRD Bab 11** (tanggal, duplikasi, reasuransi, ambang nilai & notifikasi,
  kelengkapan): temukan implementasinya di `When/`, `Activity/`, atau `RDB List/`. Untuk setiap
  aturan, catat **nilai/ambang yang benar-benar ada di rule** — inilah bahan AC berangka konkret.
- **Modul yang akan ditiketkan penuh** (ditetapkan di Fase 5, tapi verifikasinya di sini):
  cakupan nyata F-1…F-5, U-1, U-2, S-5.
- **Audit ketersediaan.** Untuk setiap procedure, function, activity, dan router yang **dipanggil
  tapi tidak ada di export**, keluarkan tabel `nama | tipe | dipanggil dari (file:baris) | ada di
  export? | tercatat di 19-GAP-EXPORT-DETAIL.md?`. **Bandingkan dengan
  `docs/Steering/19-GAP-EXPORT-DETAIL.md`, jangan menghitung ulang dari nol tanpa
  membandingkan** — selisih apa pun terhadap daftar itu wajib dilaporkan sebagai temuan, bukan
  diam-diam menggantikannya.

### Cara kerja

- Gunakan sub-agen paralel **read-only** per area. Setiap sub-agen wajib:
  (a) menyertakan `file:baris` untuk setiap klaim,
  (b) menulis `TIDAK DAPAT DIPASTIKAN DARI EXPORT` alih-alih menebak,
  (c) melaporkan **apa yang dicarinya dan tidak ditemukan**, bukan hanya apa yang ditemukan.
- **Batas filesystem bukan batas dunia.** Sub-agen yang hanya melihat export akan menyimpulkan
  "tidak ada" untuk hal yang sebenarnya hidup di database Oracle atau di sistem lain lewat DB
  Link. Kamu sebagai orkestrator wajib menerjemahkan setiap "tidak ada" menjadi salah satu dari:
  **(1)** benar-benar tidak dipakai, **(2)** ada di luar export → pertanyaan ke DBA / tim pemilik,
  **(3)** belum diperiksa. Jangan pernah membiarkannya ambigu.
- Bila temuanmu bertentangan dengan Steering atau BRD, **angkat eksplisit** dengan kedua rujukan
  berdampingan. Jangan diam-diam menimpa dokumen yang sudah disetujui.

**Output:** `docs/verifikasi-bukti-adr.md` — satu berkas, satu bagian per area, setiap bagian
diakhiri **"Pertanyaan yang hanya bisa dijawab work owner"**. Itulah bahan Fase 2.

**GATE 1 — laporkan dan tunggu aku:** ringkasan maksimal 2 halaman + daftar kontradiksi terhadap
Steering/BRD + daftar lubang pemahaman yang akan ditanyakan + daftar yang harus diminta ke pihak
luar (DBA, tim Pega, tim pemilik sistem).

---

## 4. FASE 2 — Grilling: interogasi ke aku → GATE 2 (fase terpanjang)

Jalankan `mattpocock-skills:grilling`. Yang sudah kuberikan di Sesi 1–2 adalah **keputusan
migrasi** (D-01…D-30). Yang belum pernah diambil dariku pada tingkat yang cukup untuk menulis
acceptance criteria adalah **angka, ambang, kewenangan, dan konsekuensi** — dan itu yang membuat
tiket bisa dieksekusi tanpa menebak.

Pembukaan yang harus kamu pakai:

```
Tujuan: mengangkat detail domain Claim PNC dari work owner sampai tingkat yang cukup untuk
menulis ADR dan tiket yang bisa dipertanggungjawabkan dan bisa dieksekusi — bukan sekadar
memahami sistem.

Bukti yang sudah kubaca: docs/BRD.md, docs/Steering/* (termasuk 00-DECISION-LOG D-01…D-30 dan
CONTEXT.md), docs/verifikasi-bukti-adr.md, dan rule XML Pega yang relevan.

Non-goal: menulis kode, mengubah Steering/BRD, merancang modul di luar cakupan tiket yang
disepakati.

Uji ambiguitas dan asumsi tersembunyi. Mulai dari status keputusan yang masih terbuka, lalu
turun sampai angka yang bisa dijadikan acceptance criteria.
```

### Aturan bertanya (WAJIB)

- **Maksimal 5 pertanyaan per putaran**, lalu BERHENTI dan tunggu jawabanku.
- Setiap pertanyaan wajib membawa: (a) **kenapa ini penting** — ADR atau tiket mana yang
  bergantung padanya, (b) **apa yang sudah kamu ketahui** beserta `file:baris` atau `D-nn`,
  (c) **jawaban rekomendasimu**, supaya aku cukup bilang "ya".
- **Jangan menanyakan hal yang bisa kamu buktikan sendiri.** Buktikan dulu, lalu minta konfirmasi:
  "dari rule aku lihat X di `file:baris` — benar dalam praktik lapangan?"
- **Jangan menanyakan ulang yang sudah dijawab** di `00-DECISION-LOG.md` atau
  `docs/interview-history.md`. Rujuk ID-nya. Kalau jawaban lama sudah usang, ajukan sebagai
  **revisi eksplisit**, bukan pertanyaan baru.
- Kalau jawabanku ambigu, tidak konsisten dengan jawaban lain, atau terlalu umum untuk dijadikan
  acceptance criteria — **tekan lagi**. Jangan sopan lalu lanjut.
- Catat setiap putaran ke `docs/Steering/00-DECISION-LOG.md` sebagai **Sesi 3**, mulai `D-31`,
  dengan format persis seperti entri yang sudah ada: `Status` · `Pertanyaan` ·
  `Pilihan yang ditawarkan` · `Jawaban` (**verbatim**, dalam kutipan) · `Keputusan akhir` ·
  `Dampak ke Steering`. Jangan pernah mengedit entri lama.

### Lapisan pertanyaan yang wajib dilalui

**Lapis 1 — Status enam keputusan terbuka dan lima belas risiko.**
Untuk D-14, D-17, D-18, D-25, D-28, D-29 dan R-01…R-15: sudah diminta ke siapa, kapan, ada janji
tanggal atau tidak, dan **apa rencananya kalau tidak datang**. Ini menentukan berapa ADR yang
berstatus `Proposed` dan berapa tiket yang `needs-info` — jadi tanyakan ini lebih dulu, bukan
belakangan.

**Lapis 2 — Cakupan tiket, pelaksana, dan ukuran tiket.**
Modul mana yang ditiketkan penuh lebih dulu; berapa developer; D-09 mencatat tim yang sedang
belajar Go — jadi apakah tiket ditulis untuk manusia junior, untuk agent AFK, atau keduanya;
siapa reviewer; siapa yang boleh menyatakan sebuah tiket selesai.

**Lapis 3 — Angka yang bisa dijadikan acceptance criteria.**
Tuntut ketajaman setingkat *"Rp8.800.000 boleh, Rp8.800.001 memicu jenjang komite berikutnya"* —
kalau jawabannya tidak sampai ke situ, tiketnya belum bisa `ready`. Cakup:
matriks penjenjangan komite nilai × jenis bisnis (D-14); aturan tanggal BRD §11.1 termasuk apa
yang terjadi tepat di batas (tanggal kejadian sama dengan tanggal polis mulai? lebih awal satu
hari?); aturan duplikasi §11.2 (apa persisnya yang membuat dua klaim disebut duplikat); spreading
reasuransi §11.3 (total harus 100% — toleransi pembulatan berapa desimal, dan Fac Out serta
Ex-Gratia memindahkan apa ke apa); ambang nilai & notifikasi §11.4; kelengkapan dokumen §11.5;
dan **daftar persis "empat perbaikan yang sudah diputuskan eksplisit"** yang disebut BRD §21.1
sebagai satu-satunya selisih yang boleh lolos gerbang kesetaraan.

**Lapis 4 — Uji kesetaraan (FR-S8) dan lima modul yang tidak bisa diukur.**
Data historis mana yang dipakai, di lingkungan mana, siapa yang boleh menembak Pega, apakah boleh
membaca produksi read-only, siapa yang menyetujui setiap selisih. Lalu yang paling berat:
BRD §21.4 menyatakan FR-B5, B7, B9, B10, B12 **tidak dapat dinyatakan diterima** sampai R-01
tertutup. Tanyakan: tiketnya ditulis sekarang dengan status terhalang, atau ditunda seluruhnya,
dan siapa yang menetapkan ukuran penerimaan pengganti kalau R-01 tidak pernah tertutup.

**Lapis 5 — Peran, kewenangan, dan UAT.**
Nama peran resmi yang dipakai bisnis (bukan nama access group Pega), siapa yang benar-benar duduk
di layar mana, siapa yang boleh membatalkan/mengubah/menyetujui, ada segregation of duties nyata
atau tidak, siapa yang menanggung akibat kalau data salah. Lalu per modul: **peran mana yang
menjadi penguji gerbang 2**, siapa yang menyatakan lulus, dan apakah waktunya sudah dialokasikan
resmi (C-9, R-15).

**Lapis 6 — Operasional, data, compliance.**
24/7 dan rolling deployment (D-27) — jam maintenance nyata, siapa on-call; retensi jejak audit
(D-28, masih terbuka) — berapa tahun, siapa auditornya; RPO/RTO (D-29); satu database bersama
selama masa paralel (D-21) — siapa yang berwenang mengubah skema saat Pega masih hidup; data
nasabah di dalam export XML — boleh masuk dokumen atau wajib disamarkan; soft delete vs hard
delete; kepemilikan 6 DB Link yang masih hidup.

**Lapis 7 — Jadwal.**
D-30 dan R-05 mencatat target **seluruh modul akhir September 2026** yang ditetapkan manajemen;
R-13 mencatat tenggat itu tidak punya pemaksa eksternal. Tanggal hari ini sudah 2026-09-08.
Tanyakan: apa yang **sebenarnya** harus jadi akhir September, siapa yang boleh menyatakan scope
dikurangi, apa konsekuensi nyata kalau lewat. **Jangan menulis tiket yang berpura-pura jadwal ini
masuk, dan jangan pula memutuskan sendiri untuk mengubahnya.** Catat jawabanku sebagai
keputusanku.

**Lapis 8 — Yang rawan dan harus dikonfirmasi framing-nya.**
Untuk setiap poin berikut, tanyakan **status sekarang**, **pemilik risikonya**, dan **apakah boleh
masuk ADR dengan detail lengkap atau perlu disamarkan** — jangan putuskan sendiri:
prinsip P-5 "perilaku dipertahankan lebih dulu, diperbaiki kemudian" (artinya cacat legacy
direplikasi secara sadar — mana saja?); user ID dan email yang di-hardcode di rule dan masih
hidup; D-02 memutuskan aplikasi baru tidak memanggil stored procedure **padahal sistem lain masih
memanggilnya**; 6 DB Link yang masih dipakai sistem lain; satu database bersama selama masa
paralel; dan hostname atau kredensial yang tercecer di rule XML.

### Kalau jawabanku lemah

Katakan terang-terangan bahwa jawabannya belum cukup untuk membuat acceptance criteria yang bisa
diuji, tunjukkan **bentuk jawaban yang kamu butuhkan** (contoh angka, contoh kasus batas), dan
tanya ulang. Jangan menambal dengan asumsi lalu lanjut. Sebaliknya, kalau aku memutuskan sesuatu
yang menurutmu berisiko, **bantah aku** dengan bukti dan sebutkan konsekuensinya — lalu tetap
catat keputusanku sebagai keputusanku.

**GATE 2 — tunggu aku:** ringkasan seluruh hasil grilling — keputusan baru (D-31…), keputusan lama
yang direvisi, istilah yang dikunci, angka yang sudah bisa dipakai sebagai AC, pertanyaan terbuka
beserta pemiliknya.

---

## 5. FASE 3 — Lengkapi `CONTEXT.md` (bukan tulis ulang) → GATE 3

`docs/Steering/CONTEXT.md` **sudah ada** dan sudah menjadi Lampiran A dokumen gabungan. Jangan
membuat `CONTEXT.md` baru di root — `docs/agents/domain.md` secara eksplisit menyatakan repo ini
tidak memakai layout default.

Yang dikerjakan, terbatas pada ini:

1. Naikkan penanda `[TERBUKA]` → `[BISNIS]` untuk istilah yang terjawab di Fase 2 (mis. Status
   Klaim kode `1142`–`1151` bila R-06 sudah tertutup). Bila masih terbuka, biarkan terbuka.
2. Tambahkan istilah baru yang **benar-benar muncul** saat menulis ADR dan tiket — termasuk
   pembedaan **Ticket rule (Pega)** versus **tiket pekerjaan**.
3. Jangan menghapus istilah, dan jangan mengubah definisi yang sudah `[BISNIS]` tanpa keputusan
   eksplisit dariku (kalau perlu berubah, itu entri Decision Log baru).
4. Ingat aturan 3: ubah `docs/Steering/CONTEXT.md`, **bukan** `docs/STEERING.md`. Catat kebutuhan
   membangun ulang dokumen gabungan sebagai tindakan terpisah yang menunggu persetujuanku.

**GATE 3:** tunjukkan diff usulan (istilah, penanda lama → baru, alasan + bukti) untuk kureview.

---

## 6. FASE 4 — ADR → GATE 4

Buat `docs/adr/` dengan **satu keputusan per berkas**, penamaan `NNNN-slug-singkat.md` mulai
`0001`, plus `docs/adr/README.md` berisi index tabel (nomor, judul, status, tanggal, pemilik
keputusan, `D-nn` yang diturunkan atau di-supersede, ADR yang di-supersede) dan template kosong.

### Aturan pemisahan ADR versus Decision Log — baca sebelum menulis satu ADR pun

`docs/agents/domain.md` §"Decision Log vs ADRs" mengikat: **jangan menyatakan ulang keputusan yang
sudah ada di `D-nn` sebagai ADR baru.** Karena D-01…D-30 sudah menutup sebagian besar keputusan
arsitektur, ADR di proyek ini hanya boleh berupa salah satu dari tiga jenis:

| Jenis | Isi | Kewajiban |
|---|---|---|
| **(i) Baru** | Keputusan implementasi yang belum ada di Decision Log | Jejak bukti biasa |
| **(ii) Turunan** | Menurunkan `D-nn` menjadi bentuk yang **mengikat kode**, dan menambah alternatif atau konsekuensi yang tidak tercatat di sana | Wajib blok `Menurunkan: D-nn` + **ringkasan satu paragraf dengan tautan, bukan salinan** |
| **(iii) Supersede** | Menggantikan `D-nn` karena keadaan berubah | Wajib `Menyupersede: D-nn` + alasan + persetujuanku |

**Sebelum menulis isi ADR mana pun, tunjukkan padaku daftar kandidat ADR beserta klasifikasi
(i)/(ii)/(iii) dan alasannya.** Itu bagian pertama dari GATE 4. Kandidat yang ternyata hanya
menyalin `D-nn` harus dibuang dari daftar, bukan diperhalus.

### Format tiap ADR

```
# NNNN — <judul keputusan, kalimat aktif>

Status: Proposed | Accepted | Superseded by ADR-XXXX | Deprecated
Tanggal keputusan: YYYY-MM-DD    Tanggal dokumen: YYYY-MM-DD
Sifat: original | retrospective
Jenis: baru | turunan D-nn | supersede D-nn
Pemilik keputusan: <peran, mis. Work Owner / Lead Engineer / DBA / Compliance>
Jejak bukti: D-nn | FR-xx | R-nn | path/rule.xml:baris | docs/berkas.md:baris
Terkait: CONTEXT.md#<istilah>, ADR-XXXX, tiket TKT-XX-NNN
Modul terdampak: F-n, B-n, S-n, U-n

## Konteks
## Opsi yang dipertimbangkan
## Keputusan
## Rationale
## Konsekuensi
### Positif
### Negatif / utang teknis
### Risiko yang diterima secara sadar
## Dampak ke modul dan tiket
## Pertanyaan terbuka
```

### Aturan penulisan

- **Satu berkas = satu keputusan yang mahal dibalik.** Uji kelayakan: kalau dibalik 6 bulan lagi,
  ada arsitektur atau kode yang harus dibongkar besar, atau ada risiko bisnis? Kalau tidak —
  bukan ADR; cukup catatan keputusan biasa.
- **Bagian "Konsekuensi negatif" tidak boleh kosong.** ADR yang hanya memuat keuntungan adalah
  dokumen jualan, bukan ADR.
- ADR untuk keputusan yang **sudah terjadi** sebelum dokumennya ditulis wajib ditandai
  `Sifat: retrospective` secara terbuka. Jangan berpura-pura dibuat lebih awal.
- Jangan menulis ADR `Accepted` untuk hal yang belum kuputuskan. Pakai `Proposed` + pemilik +
  **sebutkan apa yang diblokir olehnya**.
- Kalau sebuah ADR bertentangan dengan Steering, tulis eksplisit di ADR
  (`Menyupersede STEERING bab X` / `docs/Steering/NN-*.md §Y`) dan masukkan revisi Steering
  sebagai **pertanyaan terbuka** yang harus kusetujui — jangan diam-diam mengedit Steering.
- Setiap ADR wajib menyebut **modul terdampak** dan, setelah Fase 5, **tiket yang menurunkannya**.
  ADR yang tidak menyentuh satu pun modul kemungkinan bukan ADR.

### Cakupan minimum kandidat (checklist, bukan sumber)

Pecah atau gabungkan sesuai bukti, dan untuk setiap butir tentukan jenisnya (i)/(ii)/(iii):

arsitektur modular monolith + batas bounded context · struktur folder dan **aturan ketergantungan
antar lapisan yang ditegakkan otomatis** (BRD §21.2 #7) · mekanisme satu set SQL portabel dua
dialek di dalam kode (D-20) · aplikasi tidak memanggil stored procedure dan bagaimana logika 64
procedure yang belum ada diperlakukan (D-02, R-01) · kepemilikan tabel per modul dan satu penulis
per tabel (P-1, D-21) · snapshot data polis (D-04) · penomoran klaim `PNCN-xxxx` dari sequence
(D-22) · identitas dan session milik aplikasi di atas HCC/HCQ (D-07) · model otorisasi baru
(tabel peran & izin) sebagai pengganti 22 access group · empat konsep status dipertahankan
terpisah (D-18, R-06) · model penugasan Worklist/Workbasket dan penguncian (D-26) · transisi
lateral lewat Ticket rule dan bagaimana state machine mengizinkannya tanpa menjadi anarki
(FR-W2) · komponen tabel baku sebagai keputusan arsitektur frontend (U-2) · SPA murni dan meniru
tata letak Pega (D-13, D-23) · engine laporan sendiri + export besar asinkron dan streaming
(D-11, FR-S2) · penyatuan tiga mekanisme penyimpanan dokumen menjadi satu jalur (D-16, FR-S1) ·
notifikasi berbasis peristiwa domain (S-3, R-07) · jejak audit append-only dan retensinya
(D-28, S-5) · penyimpanan UTC dengan satu titik konversi WIB (F-5, FR-F5) · konfigurasi tiga
lapis dan penanganan secrets (D-15) · 6 API pengganti DB Link dan jembatan sementaranya
(D-25, R-03) · perkakas uji kesetaraan sebagai modul sekaligus sebagai gerbang cutover (FR-S8) ·
rolling deployment 24/7 (D-27) · **ADR proses**: guard lingkungan database dan larangan akses
produksi · **ADR proses**: replikasi cacat legacy yang disengaja (P-5) beserta daftar perbaikan
yang dikecualikan · VCS dan tata letak repository (repo ini belum git).

**GATE 4:** (1) daftar kandidat + klasifikasi; lalu setelah kusetujui — (2) `docs/adr/README.md` +
**tiga ADR paling berisiko secara utuh** untuk kureview sebelum kamu menulis sisanya.

---

## 7. FASE 5 — Ticketing → GATE 5

### Tanyakan padaku tiga hal sebelum menulis satu tiket pun

1. **Lokasi tiket.** `docs/ticketing/` atau `.scratch/<fitur>/issues/` (yang sekarang tertulis di
   `docs/agents/issue-tracker.md`). Sampaikan rekomendasimu — pertimbangkan bahwa tiket ini akan
   ditinjau manajemen, harus ter-commit, dan akan diimpor ke GitLab, sementara `.scratch`
   berkesan artefak buangan dan repo ini bahkan belum git. Setelah aku memilih, **perbarui
   `docs/agents/issue-tracker.md` agar konsisten** dan catat sebagai entri Decision Log baru.
2. **Cakupan tiket penuh.** Rekomendasi yang kuharap kamu ajukan beserta alasannya: gelombang 1
   (**F-1…F-5**) ditambah **U-1, U-2, S-5** — karena itulah yang benar-benar bisa dimulai besok
   dan **tidak terhalang** R-01/R-02/R-03/R-04/R-06/R-07 (lihat `06-MODULE-BREAKDOWN.md` §6).
   Sisanya cukup **stub backlog**: judul + modul + satu paragraf + dependency + apa yang
   menghalanginya, tanpa acceptance criteria detail. Tunggu keputusanku.
3. **Pembaca tiket.** Manusia junior yang sedang belajar Go (D-09), agent AFK, atau keduanya —
   ini menentukan kedalaman acceptance criteria dan seberapa eksplisit constraint teknis ditulis.

### Perbedaan mendasar yang tidak boleh kamu lupakan

**Di repo ini belum ada kode implementasi.** Karena itu:

- **Status penyelesaian tiket TIDAK BOLEH diturunkan dari kode** — tidak ada kode untuk
  diperiksa. Seluruh tiket berstatus belum dikerjakan. **Jangan pernah menulis "Selesai"**, dan
  jangan meminjam kriteria dari proyek lain yang memverifikasi status dari isi repo.
- Yang **wajib** dibuktikan dari bukti primer adalah **ruang lingkupnya** — rule Pega mana yang
  digantikan, aturan mana yang harus dipertahankan, angka mana yang mengikat. Itulah "bukti" di
  dalam tiket.
- Bagian "Rencana verifikasi" berisi perintah yang **nanti** dijalankan, dan wajib ditandai
  sebagai rencana. Jangan menulisnya seolah sudah dijalankan.

### Format tiap tiket

```
---
title: "TKT-F1-001 — <judul>"
labels: [modul::F-1, tipe::migrasi, status::needs-triage, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F1-001 — <judul>

Status: needs-triage | needs-info | ready-for-agent | ready-for-human | wontfix
Kesiapan: siap | terhalang <R-nn / D-nn>
Modul: F-1 · Gelombang: 1 · Bergantung pada: <TKT-…, atau —>
Requirement: FR-Fx    Keputusan: D-nn    ADR: NNNN    Risiko: R-nn
Rule Pega yang digantikan: <nama rule + tipe + path/file.xml>
Peran penguji gerbang 2: <peran bisnis dari Fase 2 Lapis 5>

## Hasil yang diharapkan (dan nilai bisnisnya)
## Ruang lingkup
## Non-goal
## Acceptance criteria
- [ ] <bisa diverifikasi tanpa pengetahuan pribadi, pakai angka konkret>
## Dependency / Blocked by
## Constraint keamanan, data, operasional
## Migrasi skema / rollout / rollback   (bila menyentuh data atau produksi)
## Rencana verifikasi
<perintah nyata yang akan dijalankan nanti — ditandai sebagai rencana>
## Bukti ruang lingkup
<rule XML + file:baris · D-nn · FR-xx · BRD §x.y>
## Comments
```

### Aturan penulisan tiket

- **Tiket = irisan vertikal yang bisa didemokan** dan bisa diambil independen. Satu layar atau
  satu alur bisnis, atau satu pekerjaan teknis fondasi yang jelas batasnya.
- **Jangan membuat satu tiket per activity Pega.** 778 activity custom bukan 778 tiket. Activity
  dan rule yang tercakup disebut **di dalam** bagian Ruang lingkup.
- **Status memakai lima label kanonik** dari `docs/agents/triage-labels.md` apa adanya
  (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`) — jangan
  menambah kosakata baru.
- **Definition of Ready.** Sebuah tiket boleh `ready-*` hanya bila: hasil dan nilai bisnisnya
  eksplisit · AC bisa diverifikasi teknis dan berangka · scope dan non-goal jelas · seluruh
  istilah domainnya sudah ada di `CONTEXT.md` · dependency dan blocker diketahui · constraint
  tersedia · verifikasi bisa dilakukan tanpa pengetahuan pribadi · seluruh keputusan yang
  dibutuhkannya sudah kuambil. Yang tidak lolos → `needs-info` + **sebutkan persis apa yang
  kurang dan siapa yang bisa melengkapinya**. **Jangan menambal requirement lemah dengan AC yang
  mengarang.**
- **Dua gerbang wajib muncul di AC** setiap tiket modul bisnis (BRD §21.1): gerbang 1 uji
  kesetaraan otomatis, gerbang 2 UAT peran pemakai. Sertakan pula kriteria §21.2 yang berlaku pada
  tiket itu — khususnya #7 (nol pelanggaran aturan lapisan), #8 (otorisasi diperiksa di setiap
  endpoint), #9 (jejak audit), #10 (tanpa hardcode), #11 (tanpa perangkaian SQL), #12 (kepemilikan
  tabel tunggal), #13 (migrasi skema backward-compatible).
- Untuk **FR-B5, B7, B9, B10, B12**: BRD §21.4 menyatakan gerbang 1 belum dapat dinyatakan sampai
  R-01 tertutup. Tiketnya wajib `Kesiapan: terhalang R-01` dan **tidak boleh** `ready-for-agent`.
- **Traceability dua arah wajib**: dari `FR-xx` turun ke tiket, dan dari tiket kembali ke rule
  Pega yang digantikan. Tiket tanpa salah satu arah dianggap belum selesai ditulis.
- Untuk setiap tiket yang menyentuh data atau skema, bagian rollback tidak boleh kosong — P-4
  mewajibkan migrasi skema selalu backward-compatible dan diverifikasi dengan menjalankan versi
  lama dan baru bersamaan.

### `<lokasi tiket>/README.md`

Wajib memuat: cara pakai · kosakata status (lima label + kosakata Kesiapan) · **tabel ringkasan
tiket per modul dan per gelombang** · tabel tiket yang terhalang beserta penghalang dan
pemiliknya · tabel dependency antar tiket mengikuti urutan gelombang `06-MODULE-BREAKDOWN.md` §5
termasuk **tiga rantai kritis** yang tidak bisa diparalelkan · **matriks traceability
FR → tiket → ADR → rule Pega** · dan satu bagian jujur tentang jadwal: jumlah tiket dan modul yang
belum tersentuh, berdampingan dengan sisa waktu ke target akhir September 2026 (D-30, R-05, R-13).
Angka seperti "8 dari 32" tidak boleh berdiri tanpa konteks.

**GATE 5:** tunjukkan README ticketing + **seluruh tiket cakupan penuh** + daftar stub backlog
untuk kureview.

---

## 8. FASE 6 — Konsistensi dokumen dan penutup → GATE 6

Setelah ADR dan tiket kusetujui, rapikan dokumen navigasi supaya agen dan orang berikutnya tidak
tersesat. Pakai `mattpocock-skills:writing-for-agents`.

1. **`docs/agents/domain.md`** — perbaiki path `Steering/` → `docs/Steering/`, catat bahwa
   `docs/adr/` sekarang ada beserta isinya, dan tuliskan aturan pemisahan ADR vs Decision Log
   hasil keputusan Fase 4.
2. **`docs/agents/issue-tracker.md`** — sesuaikan dengan lokasi tiket yang kupilih.
3. **`docs/AGENTS.md`** — perbarui status proyek dan tambahkan penunjuk ke `docs/adr/` dan lokasi
   tiket. **Tanyakan dulu** apakah `AGENTS.md` perlu dipindah atau disalin ke root (agen lain
   mencarinya di root) — jangan memindahkannya sendiri.
4. **Log skill** — tanyakan mana yang kanonikal (`docs/skill-usage.md` atau
   `docs/Steering/18-SKILLS-USAGE-LOG.md`), lalu tambahkan entri sesi ini dengan format yang sudah
   dipakai di sana: skill apa yang dijalankan, alasan pemilihannya, apa yang **dievaluasi tapi
   tidak dijalankan** beserta alasannya, dan hasil nyatanya.
5. **Daftar usulan revisi Steering/BRD** yang muncul sepanjang tugas ini — sebagai daftar untuk
   kusetujui, **bukan** untuk kamu eksekusi. Ingat `docs/STEERING.md` dan `.docx` adalah artefak
   bangunan.

**GATE 6 — laporan akhir**, format singkat:

```
Hasil:
Perubahan:              ← berkas yang dibuat/diubah
Verifikasi:             ← perintah yang dijalankan + hasil nyatanya
Risiko atau asumsi:
Pertanyaan terbuka + pemiliknya:
Yang menunggu persetujuanku:
```

---

## 9. DEFINITION OF DONE

- Setiap ADR punya jejak bukti, pemilik keputusan, konsekuensi negatif yang terisi, dan
  klasifikasi jenis (baru / turunan `D-nn` / supersede `D-nn`) yang sudah kusetujui.
- Tidak ada satu pun ADR yang hanya menyalin ulang isi `D-01…D-30`.
- Tidak ada satu pun pernyataan bisnis di ADR atau tiket yang berasal dari asumsimu — semuanya
  dari rule XML (dengan `file:baris`), dari dokumen proyek (dengan `D-nn`/`FR-xx`/`R-nn`), atau
  dari jawabanku (dengan `D-31`+).
- Setiap tiket cakupan penuh lolos Definition of Ready, atau ditandai `needs-info` dengan sebutan
  jelas apa yang kurang dan siapa yang bisa melengkapinya.
- Tidak ada tiket yang mengklaim status penyelesaian — di repo ini belum ada kode.
- Setiap `FR-xx` dalam cakupan penuh punya minimal satu tiket; setiap tiket punya minimal satu
  `FR-xx` dan satu rule Pega yang digantikan.
- Enam keputusan terbuka dan lima belas risiko punya status mutakhir, pemilik, dan tercermin di
  status ADR serta kesiapan tiket.
- `00-DECISION-LOG.md` bertambah (Sesi 3, `D-31`+) dan **tidak ada entri lama yang diedit**.
- Tidak ada satu baris kode aplikasi yang ditulis, tidak ada rule XML yang berubah, dan
  `docs/STEERING.md` beserta kedua `.docx` tidak disentuh.
- Tidak ada secret, kredensial, hostname produksi, atau data nasabah yang masuk ke dokumen.

---

## 10. HIGIENE KONTEKS

Pertahankan konteks di dalam rangkaian **Fase 1 → 2 → 3 → 4 → 5**: setiap fase memakai keputusan
fase sebelumnya, dan ADR di Fase 4 adalah tempat konteks paling dibutuhkan. Karena itu jangan
membakar konteks pada pembacaan XML borongan di Fase 1 — pakai pencarian terarah dan sub-agen.
Fase 0 dan Fase 6 berdiri sendiri dan tidak butuh riwayat grilling.

Kesimpulan permanen disimpan di `docs/adr/`, lokasi tiket, `docs/Steering/CONTEXT.md`, dan
`docs/Steering/00-DECISION-LOG.md` — **bukan di riwayat chat.**

**MULAI DARI FASE 0. Jangan lewati gate.**
