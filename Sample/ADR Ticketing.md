# TUGAS: Bangun ADR, Ticketing, dan README aplikasi TEKNO baru

Kamu adalah lead engineer + domain analyst untuk proyek migrasi aplikasi TEKNO
(ASP.NET Web Forms VB.NET + Classic ASP + Oracle) ke Golang + React.
Aku adalah developer/pemilik logic aplikasi lama dan pemilik keputusan bisnis (work owner).

## 0. ATURAN MUTLAK

1. **JANGAN menulis, mengubah, me-refactor, atau menghapus kode aplikasi apa pun**
   (`backend/**/*.go`, `web/src/**`). Tidak ada reverse-engineering ulang, tidak ada
   perbaikan bug, tidak ada perubahan skema. Output tugas ini 100% dokumen + penataan
   struktur repo.
2. **JANGAN menjalankan DDL/DML apa pun** ke database. Query hanya boleh SELECT /
   introspeksi katalog (`ALL_TAB_COLUMNS`, `ALL_SOURCE`, dll), dan hanya ke DB DEV.
   Jika `.env` menunjuk selain DEV — berhenti, laporkan, jangan konek.
3. **JANGAN pernah mengisi sendiri kekosongan pemahaman bisnis dengan asumsi.**
   Jika sebuah fakta tidak bisa dibuktikan dari source code / data / jawabanku,
   tulis literal `BELUM DIPUTUSKAN — pertanyaan terbuka` beserta pemilik keputusannya.
   Menebak lalu menuliskannya sebagai fakta di ADR adalah kegagalan tugas.
4. **Setiap klaim faktual wajib punya jejak bukti**: `path/file:baris`, nama objek DB,
   nomor `Qnn` di `docs/decision-log.md`, atau hasil perintah yang kamu jalankan.
   Klaim tanpa jejak harus ditandai sebagai belum terverifikasi.
5. **Kamu berhenti dan menunggu jawabanku di setiap GATE.** Jangan lanjut ke fase
   berikutnya tanpa persetujuanku. Jangan menjawab pertanyaanmu sendiri.
6. **Jangan pernah menampilkan atau menyalin nilai rahasia** (isi `backend/.env`,
   password, string backdoor legacy, NPWP, host/IP produksi) ke dokumen yang akan
   di-commit. Rujuk dengan nama key saja.
7. Bahasa dokumen: **Indonesia** untuk dokumen bisnis/manajemen (README, ticketing,
   pengantar); ADR dan `CONTEXT.md` boleh Indonesia dengan istilah domain asli
   dipertahankan apa adanya (SPK, OR, TCR, dst — jangan diterjemahkan).

## 1. SUMBER KEBENARAN

Pertama, tentukan root repo: folder yang berisi `CLAUDE.md`, `docs/STEERING.md`,
`backend/go.mod`, dan `web/package.json`. Semua path di bawah relatif ke root itu.
Jika tidak ditemukan, **berhenti dan tanya aku** — jangan menebak lokasi.

Urutan otoritas (tinggi → rendah):

1. **Aku (developer/work owner)** — satu-satunya sumber untuk aturan bisnis yang tidak
   tertulis di kode, alasan historis, dan keputusan produk.
2. **Source code legacy** — `*.aspx`, `*.aspx.vb`, `App_Code/*.vb` (terutama
   `General.vb`, `DataConnection.vb`), `*.asp`, `MasterPage*.master`, `web.config`,
   `Global.asax`, `*.ashx`.
3. **PL/SQL** — `docs/legacy-plsql/` (65 objek, ±19.6k baris) + `ALL_SOURCE` bila perlu.
4. **Data live DB DEV** — read-only, untuk membuktikan apa yang benar-benar dipakai.
5. **Dokumen internal proyek** — `docs/STEERING.md`,
   `docs/Business Requirement Document - TEKNO eBENGKEL.docx`,
   `docs/decision-log.md`, `docs/understanding-findings.md`, `docs/skill-usage-log.md`,
   `docs/*-analysis.md`, `docs/agents/*.md`, `IMPLEMENTATION_NOTE.md`,
   `docs/pengantar-review-adr-ticketing.md`.
6. **Kode aplikasi baru** — `backend/internal/**`, `web/src/**`, `cmd/api/main.go`,
   `web/src/menuRoutes.ts` (untuk membuktikan status tiket, bukan untuk diubah).

Yang **TIDAK boleh dipercaya**: folder `analisis/` (artefak sesi lama yang merujuk
dokumen sumber `ANALISIS_PROJECT_TEKNO.md` yang tidak ada di repo). Boleh dibaca
sebagai petunjuk, tidak boleh dikutip sebagai bukti.

**Catatan penting:** `docs/pengantar-review-adr-ticketing.md` membahas `docs/ADR.md`
dan `docs/ticketing.md` — kedua file itu **tidak ada**. Pengantar itu memuat daftar
topik ADR yang dimaksud (ADR-001…ADR-020) dan kosakata status tiket. Pakai sebagai
**input dan checklist cakupan**, tapi turunkan ulang setiap ADR dari bukti primer.
Jangan menyalin klaimnya.

## 2. FASE 0 — Audit lingkungan & inventaris gap  →  GATE 0

1. Evaluasi `/setup-matt-pocock-skills`: cek `.claude/skills/`, `docs/agents/*.md`,
   dan blok `## Agent skills` di `CLAUDE.md`. **Jangan jalankan ulang jika sudah
   lengkap** — laporkan "evaluated, not re-run" beserta alasannya (pola ini sudah
   dipakai di `docs/skill-usage-log.md`, ikuti).
2. Cek VCS: `.hg` vs `.git`, ada remote atau tidak.
3. Inventaris artefak: mana yang ADA, mana yang HILANG, mana yang DIRUJUK TAPI TIDAK ADA.
4. Verifikasi bisa-tidaknya build: `go build ./...` dan `go vet ./...` di `backend/`,
   `npm run build` di `web/` (kalau `node_modules` sudah ada). Ini baseline sebelum
   pemisahan repo, supaya nanti bisa dibuktikan pemisahan tidak merusak apa pun.
5. Cek koneksi DB: environment mana yang ditunjuk `backend/.env` (sebut nama key +
   nama environment saja, JANGAN tampilkan credential).

**GATE 0 — laporkan dan tunggu aku:**
tabel inventaris (ada/hilang/dirujuk-tapi-hilang), status VCS, hasil build,
environment DB, dan daftar hal yang menurutmu perlu aku putuskan lebih dulu.

## 3. FASE 1 — Baca ulang & pahami aplikasi legacy  →  GATE 1

Tujuan fase ini **bukan** menulis ulang analisis yang sudah ada. Tujuannya:
menemukan **apa yang tidak bisa dijawab oleh source code**, supaya pertanyaan di
Fase 2 tajam dan tidak membuang waktuku.

Cakupan wajib:

- **Backbone**: auth/session/RBAC (`Default.aspx.vb`, `MasterPage*.master`,
  `M_MENU`/`M_GRUPMENU`/`M_USERMENU`), `General.vb`, `DataConnection.vb`, `web.config`,
  multi-cabang (`BENGKEL_ID`), penomoran dokumen (`GET_COUNTER`, `M_Counter`),
  kalender kerja (`COUNT_WORKDAYS`, `GETNEXTBUSINESSDAY`, `M_Hari_Libur`).
- **Modul prioritas — PROSES PRODUKSI / REGISTRASI KENDARAAN MASUK**:
  `T_SPK_maintenance.aspx(.vb)`, `T_SPK_MAINTENANCE_OCR.aspx(.vb)`, `T_SPK.aspx(.vb)`,
  `registrasi*.asp`, `T_Masuk_Kend*.aspx(.vb)`, plus PL/SQL terkait (`PC_T_SPK`,
  `SP_T_SPK_*`, `SP_FORMULA_ESTIMASI2020`, `SP_INSERT_T_KEND_MASUK`, `GET_EST_SELESAI2`),
  dan padanan barunya `backend/internal/production/spkreg/` +
  `web/src/pages/production/SpkRegistrationPage.tsx`.
- **Peta hulu-hilir**: dari mana data Registrasi datang dan ke mana mengalir
  (estimasi → SPK → progress/QC → sparepart/bahan → OR/tagihan → closing/jurnal).
  Cukup sampai batas antarmuka, tidak perlu mendalam per modul.

Cara kerja:

- Gunakan sub-agen paralel (read-only) per area, masing-masing diwajibkan:
  (a) sertakan `file:baris` untuk setiap klaim,
  (b) tulis `NOT DETERMINABLE FROM SOURCE` alih-alih menebak,
  (c) audit ketersediaan setiap stored procedure yang dipanggil dalam tabel
      `nama | source ada? | path`.
- Ingat pelajaran yang sudah tercatat di `docs/skill-usage-log.md` Session 11:
  sub-agen yang dibatasi ke filesystem akan melaporkan batas filesystem sebagai batas
  dunia. Kamu sebagai orkestrator wajib melengkapi dengan `ALL_SOURCE` dan introspeksi
  data live (read-only) untuk hal yang tidak ada di source: struktur kolom sebenarnya,
  skala data, dan isi tabel referensi.
- Bandingkan temuanmu dengan `docs/*-analysis.md` yang sudah ada. Jika bertentangan,
  **surface eksplisit** — jangan diam-diam menimpa.

Output: `docs/legacy-understanding/registrasi-kendaraan-masuk.md` dan
`docs/legacy-understanding/backbone.md`. Setiap file diakhiri bagian
**"Pertanyaan yang hanya bisa dijawab oleh work owner"** — inilah bahan Fase 2.

**GATE 1 — laporkan dan tunggu aku:**
ringkasan maksimal 2 halaman + daftar kontradiksi terhadap dokumen lama + daftar
lubang pemahaman yang akan ditanyakan.

## 4. FASE 2 — `/grill-with-docs`: interogasi pemahaman ke aku  →  GATE 2 (fase terpanjang)

Ini bagian yang dilewatkan di sesi-sesi sebelumnya dan penyebab utama ADR belum bisa
dipertanggungjawabkan. `docs/decision-log.md` sekarang isinya dominan "AI menawarkan
opsi → user memilih" (keputusan **migrasi**). Yang belum pernah diambil dariku adalah
**ground truth domain**: kenapa prosesnya begitu, siapa yang benar-benar melakukan apa,
dan aturan mana yang tidak pernah tertulis di kode.

Jalankan `/grill-with-docs` dengan pembukaan seperti ini:

```
/grill-with-docs

Tujuan: mengangkat ground truth domain TEKNO dari work owner sampai tingkat yang
cukup untuk menulis ADR dan tiket yang bisa dipertanggungjawabkan — bukan sekadar
memahami kode.

Bukti yang sudah kubaca: docs/STEERING.md, BRD, docs/decision-log.md,
docs/legacy-understanding/*, docs/legacy-plsql/, source legacy modul Registrasi
Kendaraan Masuk, dan implementasi baru di backend/internal/production/spkreg/.

Non-goal: menulis kode, mengubah implementasi, merancang modul yang belum digarap.

Uji ambiguitas dan asumsi tersembunyi. Mulai dari yang paling general, turun sampai
yang paling teknis. Catat istilah domain di CONTEXT.md dan keputusan permanen di ADR.
```

### Aturan bertanya (WAJIB)

- **Maksimal 5 pertanyaan per putaran**, lalu BERHENTI dan tunggu jawabanku.
- Setiap pertanyaan wajib membawa: (a) **kenapa ini penting** — ADR/tiket mana yang
  bergantung padanya, (b) **apa yang sudah kamu ketahui dari kode** beserta
  `file:baris`, (c) **jawaban rekomendasimu** supaya aku bisa cukup bilang "ya".
- **Jangan tanyakan hal yang bisa kamu buktikan sendiri** dari source atau data live.
  Buktikan dulu, lalu minta konfirmasi ("dari kode aku lihat X di `file:baris` —
  benar dalam praktik lapangan?").
- Jangan tanyakan hal yang sudah terjawab di `docs/decision-log.md`. Rujuk `Qnn`-nya.
  Kalau jawaban lama sudah usang, tanyakan sebagai revisi eksplisit, bukan pertanyaan baru.
- Kalau jawabanku ambigu, tidak konsisten dengan jawaban lain, atau terlalu umum
  untuk dijadikan acceptance criteria — **tekan lagi**. Jangan sopan lalu lanjut.
  Runbook menyebut ini secara eksplisit: ketidakjelasan bisnis diselesaikan dengan
  decision record, bukan ditutup dengan instruksi coding yang lebih panjang.
- Catat setiap putaran ke `docs/decision-log.md` sebagai entri baru (lanjutkan
  penomoran `Qnn` yang sudah ada), memuat: pertanyaan, opsi yang ditawarkan,
  **jawabanku verbatim**, dan keputusan turunannya. Jangan pernah mengedit entri lama.

### Lapisan pertanyaan yang wajib dilalui

Naikkan kedalaman bertahap. Ini kerangka minimum — kembangkan sesuai temuan Fase 1.

**Lapis 1 — Identitas bisnis (yang tidak ada di kode sama sekali)**
TEKNO ini sebenarnya bisnis apa (bengkel body repair? termasuk salvage/lelang/jual
mobil?), siapa pemiliknya dan hubungannya dengan Sinar Mas, siapa pelanggan yang
membayar (asuransi vs pemilik kendaraan vs internal), model pendapatannya,
berapa cabang dan bedanya, volume harian nyata, jam operasional, dan apa yang
dianggap "gagal" oleh bisnis (bukan oleh sistem).

**Lapis 2 — Aktor & wewenang**
Daftar peran nyata di lapangan (SA, Foreman, Mekanik/MB, Bartender, QC, Gudang
Bahan, Gudang Sparepart, Kasir, Finance, Kepala Bengkel, IT, Surveyor, Supir),
siapa yang benar-benar duduk di depan layar Registrasi, siapa yang boleh
membatalkan/mengubah/menyetujui, apakah ada segregation of duties yang nyata,
dan siapa yang menanggung akibat kalau data salah.

**Lapis 3 — Siklus hidup & terminologi**
Definisi tepat dan yang benar-benar dipakai orang untuk: SPK, OR, TCR, TKU, Bursa,
ASM vs NonASM, MB, PPB, PO, Salvage, RJ (Rawat Jalan), Progress/Pos, Redo,
Klaim, Pelunasan, Closing, Tutup Buku. Untuk setiap istilah: sinonim yang harus
dihindari, dan siapa lawan bicaranya (internal vs asuransi vs customer).
Lalu: state machine kendaraan dari masuk gerbang sampai keluar — state mana yang
tidak boleh dilewati, mana yang boleh mundur, siapa yang boleh memundurkan.

**Lapis 4 — Aturan bisnis yang tidak terlihat di source**
Untuk setiap hal berikut, tanyakan aturan sebenarnya dan **kasus batasnya dengan
angka konkret** (runbook menuntut acceptance criteria yang bisa diuji, contohnya
"Rp8.800.000 boleh, Rp8.800.001 memicu re-approval" — tuntut ketajaman setara):
- estimasi tanggal selesai: hari kerja vs kalender, libur, shift, dan apa yang
  terjadi kalau estimasi terlewat;
- penomoran SPK: format, per cabang atau global, boleh ada gap atau tidak,
  apa yang terjadi kalau dua orang simpan bersamaan;
- kewajiban field kendaraan (Q18 → revisi di grill Session 1: wajib input, tidak
  dicek keberadaan di `M_KENDARAAN`) — masih berlaku? kenapa?;
- harga/diskon/PPN/pengali warna: siapa yang berwenang, batas toleransi perubahan,
  kapan butuh approval;
- Panel Di Cat / Jenis Warna dan ketergantungannya ke `M_PRICELIST_BENGKEL`;
- upah mekanik/borongan dan konsekuensi `PACKAGE_UPAH_MB_MASTER_BARU` yang punya
  8 COMMIT internal (simpan tidak atomik) — ini menyentuh gaji orang, tanyakan
  sikap bisnisnya, bukan hanya sikap teknisnya;
- kasus di mana legacy jelas keliru tapi arsipnya sudah dilaporkan (contoh yang
  sudah tercatat: aritmetika eFaktur `HARGA - RP_DISC * QTY` dan tarif PPN
  hardcoded `'12'`) — replikasi salah, atau koreksi? siapa pemilik keputusannya?

**Lapis 5 — Integrasi, compliance, operasional**
Kontrak nyata ClaimQ / PEGA-HCC / Journal Service / WhatsApp gateway / VA
(mana yang masih hidup, siapa pemiliknya, apa SLA-nya, apa yang terjadi kalau down),
retensi & kewajiban audit di lingkungan asuransi (berapa tahun, siapa auditor,
apakah master data benar-benar boleh tanpa audit trail — Q30 memutuskan
"ditangguhkan", masih dipertahankan?), soft delete vs hard delete, PII/data
pelanggan, jam maintenance, siapa yang jaga kalau sistem down jam 2 pagi.

**Lapis 6 — Migrasi, cutover, rollback**
Cutover per cabang (hasil grill Session 1) masih berlaku? cabang mana dulu?
siapa yang menyatakan "boleh potong"? kriteria rollback? data historis dibawa
semua atau sebagian? paralel-run berapa lama? siapa yang UAT dan apa kriteria
lulusnya? Dan: keputusan Mercurial → Git/GitLab (implikasi Q14) sudah final?

**Lapis 7 — Yang rawan dan harus dikonfirmasi framing-nya**
Baca `docs/pengantar-review-adr-ticketing.md` bagian B dan D. Untuk setiap poin
rawan di sana (insiden tulis ke DB produksi 8 Agu 2026, backdoor login legacy yang
masih hidup, angka pajak yang sengaja meniru legacy, dua pengecualian aturan
"tanpa stored procedure", audit trail master data ditangguhkan, STEERING §5/§10
belum direvisi, hash password lama tidak ter-upgrade, session 120 menit,
nilai sensitif di `.env`), tanyakan padaku: **status sekarang** (masih berlaku?
sudah ditutup?), **pemilik risikonya**, dan **apakah boleh masuk ADR dengan
detail lengkap atau perlu disamarkan**. Jangan putuskan sendiri.

### Kalau aku menjawab lemah

Kalau jawabanku tidak cukup untuk membuat acceptance criteria yang bisa diuji,
katakan itu terang-terangan, tunjukkan bentuk jawaban yang kamu butuhkan, dan
tanya ulang. Jangan menambal dengan asumsi lalu lanjut. Sebaliknya, kalau aku
memutuskan sesuatu yang menurutmu berisiko, **bantah aku** dengan bukti dan
sebutkan konsekuensinya — lalu tetap catat keputusanku sebagai keputusanku.

**GATE 2 — tunggu aku:** ringkasan seluruh hasil grilling — keputusan baru,
keputusan lama yang direvisi, istilah yang dikunci, pertanyaan terbuka + pemiliknya.
Baru setelah aku setuju, lanjut Fase 3.

## 5. FASE 3 — `CONTEXT.md` (glosarium domain)  →  GATE 3

Tulis `CONTEXT.md` di root, Isi: bahasa domain bersama —
setiap istilah dengan definisi, sinonim yang harus dihindari, relasi ke istilah lain,
dan aktor yang memakainya. Sumber: Fase 2, bukan tebakan dari nama kolom.

Ini prasyarat ADR: keputusan tidak bisa dipertanggungjawabkan kalau istilahnya
belum dikunci. Kalau sebuah istilah masih diperdebatkan, tulis sebagai
pertanyaan terbuka, jangan dipaksa punya definisi.

Perbarui juga `docs/agents/domain.md` bila layout dokumentasi domain berubah.

**GATE 3:** tunjukkan `CONTEXT.md` untuk aku review sebelum lanjut.

## 6. FASE 4 — ADR  →  GATE 4

Buat `docs/adr/` dengan satu keputusan per file, penamaan
`NNNN-slug-singkat.md` (mulai `0001`), plus `docs/adr/README.md` berisi index tabel
(nomor, judul, status, tanggal, pemilik keputusan, ADR yang di-supersede) dan
template kosong.

Format tiap ADR (Nygard, plus kolom akuntabilitas yang proyek ini butuhkan):

```
# NNNN — <judul keputusan, kalimat aktif>

Status: Proposed | Accepted | Superseded by ADR-XXXX | Deprecated
Tanggal keputusan: YYYY-MM-DD    Tanggal dokumen: YYYY-MM-DD
Sifat: original | retrospective
Pemilik keputusan: <nama peran, mis. Work Owner / Lead Engineer / Tim Pajak>
Jejak bukti: decision-log Qnn, Qnn | path/file:baris | nama objek DB
Terkait: CONTEXT.md#<istilah>, ADR-XXXX, tiket TKT-XXX

## Konteks
## Opsi yang dipertimbangkan
## Keputusan
## Rationale
## Konsekuensi
### Positif
### Negatif / utang teknis
### Risiko yang diterima secara sadar
## Pertanyaan terbuka
```

Aturan:

- **Satu file = satu keputusan yang mahal dibalik.** Uji kelayakan: kalau dibalik
  6 bulan lagi, ada arsitektur/kode yang harus dibongkar besar atau ada risiko bisnis?
  Kalau tidak — bukan ADR, taruh di `docs/decision-log.md` saja.
- ADR untuk keputusan yang sudah terjadi sebelum dokumennya ditulis **wajib** ditandai
  `Sifat: retrospective` secara terbuka. Jangan berpura-pura dibuat lebih awal.
- **Bagian "Konsekuensi negatif" tidak boleh kosong.** ADR yang hanya memuat
  keuntungan adalah dokumen jualan, bukan ADR.
- Jangan menulis ADR untuk hal yang belum aku putuskan. Status `Proposed` +
  pertanyaan terbuka + pemilik, dan sebutkan apa yang diblokir olehnya.
- Kalau sebuah ADR bertentangan dengan `docs/STEERING.md`, **katakan eksplisit**
  di ADR (`Menyupersede STEERING §X`) dan masukkan revisi STEERING sebagai
  pertanyaan terbuka yang aku harus setujui — jangan diam-diam mengedit STEERING.
- Cakupan minimum yang harus tercakup (silakan pecah/gabung sesuai bukti, dan
  bandingkan dengan daftar di `docs/pengantar-review-adr-ticketing.md` sebagai
  checklist, bukan sebagai sumber): pilihan bahasa/stack backend, arsitektur
  modular monolith, database engine Oracle + standar SQL portable-PostgreSQL,
  topologi single-DB + `BENGKEL_ID`, pilihan frontend & component library,
  autentikasi/session/hash password, RBAC berbasis menu legacy, batas modul,
  strategi migrasi & cutover per cabang, pengganti Crystal Reports, penomoran
  dokumen, penanganan stored procedure (termasuk dua pengecualian dan implikasi
  non-atomik), audit trail master data, replikasi cacat legacy yang disengaja,
  secrets/config, VCS Mercurial → Git/GitLab, observability & deployment,
  serta ADR proses (guard environment DB) untuk insiden 8 Agu 2026 —
  dengan detail sesuai keputusanku di Lapis 7.

**GATE 4:** tunjukkan index ADR + 3 ADR paling berisiko secara utuh untuk aku review
sebelum kamu menulis sisanya.

## 7. FASE 5 — Ticketing  →  GATE 5

Cakupan tiket **lengkap** hanya untuk **PROSES PRODUKSI — Registrasi Kendaraan Masuk**.
Modul lain: cukup stub backlog (judul + status + 1 paragraf + dependency), tanpa
acceptance criteria detail.

Sebelum menulis, **tanyakan padaku dua hal**:

1. Lokasi tiket: `docs/ticketing/` (ikut struktur yang aku bayangkan, ter-commit dan
   terlihat di GitLab) atau `.scratch/<feature>/issues/` (sesuai
   `docs/agents/issue-tracker.md` sekarang, tapi namanya berkesan buangan).
   Rekomendasimu, lalu setelah aku pilih — **perbarui `docs/agents/issue-tracker.md`
   agar konsisten** dan catat sebagai keputusan.
2. Klarifikasi status: `backend/internal/` sudah berisi ~15 grup modul
   (penerimaan, ar, ap, pembelian, laporan, dst), padahal yang aku sebut selesai
   100% dan sudah UAT hanya Registrasi Kendaraan Masuk. Tanyakan status jujur
   masing-masing sebelum memberi label.

Format tiap tiket — gabungkan disiplin runbook dengan kesiapan impor ke GitLab:

```
---
title: "TKT-PP-001 — <judul>"
labels: [modul::proses-produksi, tipe::migrasi, status::selesai, prioritas::tinggi]
milestone: "Modul 1 — Proses Produksi"
epic: "Migrasi TEKNO"
---

# TKT-PP-001 — <judul>

Status: needs-triage | needs-info | ready-for-agent | ready-for-human | wontfix
Progres: Selesai | Selesai* | Sebagian | Ditangguhkan | Belum | Tidak diport
Layar legacy: <MENU_ID / MENU_PROGRAM di M_MENU> — <file.aspx>
Padanan baru: backend/internal/... + web/src/pages/...
Requirement: BRD FR-xx    Keputusan: Qnn    ADR: ADR-xxxx

## Hasil yang diharapkan (dan nilai bisnisnya)
## Ruang lingkup
## Non-goal
## Acceptance criteria
- [ ] <harus bisa diverifikasi tanpa pengetahuan pribadi, pakai angka konkret>
## Dependency / Blocked by
## Constraint keamanan, data, operasional
## Migration / rollout / rollback  (bila menyentuh data atau produksi)
## Cara verifikasi
<perintah nyata: go test ./internal/production/spkreg/..., dst>
## Bukti status
<file:baris, terdaftar di cmd/api/main.go, rute di web/src/menuRoutes.ts, hasil test>
## Comments
```

Aturan:

- Tiket = **vertical slice yang demoable**, bisa diambil independen. Satu layar/submenu
  legacy atau satu pekerjaan teknis.
- Status **wajib diturunkan dari bukti di repo**, bukan dari ingatan atau klaim.
  Pakai kriteria yang sudah dipakai proyek ini: `Selesai` hanya jika paket Go terdaftar
  di `cmd/api/main.go`, halaman React ada, **dan** rutenya terpetakan di `menuRoutes.ts`.
  `Selesai*` untuk yang jalur tulisnya belum pernah dieksekusi — dan sebutkan
  alasannya (instruksi keselamatan, bukan kelalaian).
- Uji "ready-for-agent" (Definition of Ready runbook): hasil & nilai eksplisit,
  AC bisa diverifikasi teknis, scope & non-goal jelas, istilah domain sudah ada di
  `CONTEXT.md`, dependency & blocker diketahui, constraint tersedia, verifikasi bisa
  dilakukan tanpa pengetahuan pribadi, semua keputusan HITL sudah diambil.
  Tiket yang tidak lolos → beri `needs-info` + sebutkan apa yang kurang. **Jangan
  menambal requirement lemah dengan AC yang mengarang.**
- Buat `docs/ticketing/README.md`: cara pakai, kosakata status, tabel ringkasan
  progres (selesai-dari-yang-di-scope **dan** selesai-dari-total-menu, berdampingan —
  supaya angka seperti "2 dari 24" tidak berdiri tanpa konteks), tabel dependency/blocker,
  dan matriks traceability BRD FR → tiket → ADR → kode.

**GATE 5:** tunjukkan README ticketing + seluruh tiket Registrasi Kendaraan Masuk
+ daftar stub backlog untuk aku review.


## 9. FASE 7 — README dan penutup  →  GATE 7

`README.md` root, ditujukan untuk engineer baru **dan** pembaca non-teknis, memuat:

1. Apa itu TEKNO (dari `CONTEXT.md`, bukan dari tebakan), siapa penggunanya,
   dan apa yang sedang dimigrasikan dari apa ke apa.
2. Status proyek yang jujur: modul apa yang selesai & sudah UAT, apa yang berjalan,
   apa yang belum — dengan link ke `docs/ticketing/README.md`.
3. Arsitektur ringkas + diagram teks, dengan link ke ADR untuk setiap keputusan besar
   (jangan mengulang isi ADR di README).
4. Cara menjalankan lokal: prasyarat versi, setup `.env` dari `.env.example`,
   perintah backend, perintah frontend, perintah test. Harus benar — **verifikasi
   dengan menjalankannya**, jangan menulis perintah yang belum pernah kamu jalankan.
5. Struktur direktori + tujuan tiap folder.
6. Cara kerja tim: alur ADR, alur tiket, alur MR, link `CONTRIBUTING.md` dan runbook.
7. Klasifikasi dokumen (Internal — Rahasia Perusahaan) dan peringatan bahwa `.env`
   serta dokumen `docs/legacy/` tidak untuk distribusi keluar.
8. Peta dokumen: satu tabel "kalau mau X, baca Y".

Terakhir, perbarui `docs/skill-usage-log.md` dengan entri sesi ini: skill apa yang
dijalankan, alasan pemilihannya, apa yang **dievaluasi tapi tidak dijalankan** beserta
alasannya, dan apa hasil nyatanya. Ikuti gaya entri yang sudah ada.

**GATE 7 — laporan akhir**, format singkat:

```
Hasil:
Perubahan:
Verifikasi:        ← perintah yang dijalankan + hasil nyatanya
Risiko atau asumsi:
Pertanyaan terbuka + pemiliknya:
```

## 10. DEFINITION OF DONE

- Setiap ADR punya jejak bukti, konsekuensi negatif yang terisi, dan pemilik keputusan.
- Tidak ada satu pun pernyataan bisnis di ADR/tiket yang berasal dari asumsimu —
  semuanya dari source code (dengan `file:baris`), data live, atau jawabanku (dengan `Qnn`).
- Setiap tiket Registrasi Kendaraan Masuk lolos Definition of Ready, atau ditandai
  `needs-info` dengan sebutan jelas apa yang kurang.
- Status tiket bisa dibuktikan dari repo, bukan dari klaim.
- Repo baru bisa di-build dan di-test dari lokasi barunya, terbukti dengan perintah
  yang benar-benar dijalankan.
- Tidak ada secret, credential, host produksi, atau artefak build yang ikut ke repo baru.
- Tidak ada satu baris kode aplikasi yang berubah.
- `docs/decision-log.md` dan `docs/skill-usage-log.md` bertambah, tidak ada entri lama
  yang diedit.

## 11. HIGIENE KONTEKS

Ikuti runbook §10. Pertahankan konteks di dalam rangkaian
Fase 1 → 2 → 3 → 4 → 5 (setiap fase memakai keputusan fase sebelumnya).—
pekerjaan itu tidak butuh riwayat grilling, dan konteks yang sudah besar justru
menurunkan kualitas. Kesimpulan permanen disimpan di `CONTEXT.md`, `docs/adr/`,
`docs/ticketing/`, dan `docs/decision-log.md` — bukan di riwayat chat.

MULAI DARI FASE 0. Jangan lewati gate.
