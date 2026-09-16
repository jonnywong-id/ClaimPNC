# Skill Usage — Migrasi Claim PNC

Catatan setiap skill yang dipakai sepanjang pekerjaan pemahaman dan penyusunan BRD, mengikuti
format yang diminta: **Nama Skill · Alasan digunakan · Kapan digunakan · Masalah yang
diselesaikan · Hasil yang diperoleh.**

| | |
|---|---|
| **Tanggal** | 2026-09-08 |
| **Log rinci sesi sebelumnya** | [`Steering/18-SKILLS-USAGE-LOG.md`](Steering/18-SKILLS-USAGE-LOG.md) |

---

## 0. Prasyarat yang tidak dapat dijalankan

### `/setup-matt-pocock-skills` — **TIDAK ADA di environment ini**

Instruksi INITIALIZATION meminta perintah ini dijalankan lebih dulu. Diperiksa ulang pada
2026-09-08 terhadap `~/.claude/`, `~/.claude/plugins/`, dan registry skill sesi: **perintah ini
tidak terpasang**, sehingga tidak dapat dijalankan. Temuan yang sama sudah dilaporkan pada
2026-09-07 dan tercatat di
[`Steering/18-SKILLS-USAGE-LOG.md`](Steering/18-SKILLS-USAGE-LOG.md).

**Dampaknya nihil**, karena plugin `mattpocock-skills` **sudah terpasang** — 11 skill tersedia
dan dapat dipanggil langsung tanpa langkah setup. Yang tidak terpasang hanyalah perintah
setup-nya dan sekumpulan slash command lain yang juga disebut di instruksi:
`/ask-matt`, `/grill-me`, `/grill-with-docs`, `/to-spec`, `/to-tickets`, `/triage`,
`/implement`, `/wayfinder`, `/improve-codebase-architecture`, `/teach`, `/handoff`.

### `~/.claude/CLAUDE.md` — **TIDAK ADA**

Instruksi meminta seluruh skill dimuat dari berkas ini. Berkas tidak ada. Instruksi tingkat
project yang berlaku ada di [`AGENTS.md`](AGENTS.md), dan sudah dibaca.

---

## Ringkasan penggunaan

| Skill | Tahap | Dipakai? |
|---|---|---|
| `mattpocock-skills:domain-modeling` | Deep Discovery (2026-09-07) | **Ya** |
| `mattpocock-skills:grilling` | Deep Discovery (2026-09-07) | **Ya** |
| `mattpocock-skills:codebase-design` | Arsitektur & Modul (2026-09-07/08) | **Ya** |
| `mattpocock-skills:grilling` — disiplinnya, tanpa pemanggilan ulang | Discovery BRD (2026-09-08) | **Sebagian** — lihat §4 |
| 8 skill lainnya | — | **Tidak** — alasan per skill di §5 |

---

## 1. `mattpocock-skills:domain-modeling`

**Kapan digunakan.** Tahap Deep Discovery, 2026-09-07 — setelah pembacaan 902 activity dan
652 rule SQL selesai, sebelum Domain Model disusun.

**Alasan digunakan.** Masalah terbesar yang muncul setelah membaca source **bukan kerumitan
teknis, melainkan bahasa domain yang kacau.** Tiga gejala konkret ditemukan: istilah
`AdjustmentList` yang ternyata berisi nilai penyelesaian (bukan proses loss adjusting), istilah
`Object` yang bertabrakan dengan makna pemrograman, dan empat konsep status bernama mirip
(`StatusWork`, `StatusClaim`, `ClaimStatus`, `StatusPosisi`) yang tidak jelas apakah benar-benar
berbeda. Skill ini menangani tepat hal itu.

**Masalah yang diselesaikan.** Kekacauan penamaan sistem lama akan ikut berpindah ke sistem baru
bila tidak ditangani lebih dulu — dan kekacauan itu justru salah satu alasan utama migrasi.

Cara skill dipakai: setiap istilah meragukan diuji ke pemilik bisnis dengan usulan kanonikal
beserta alasannya; **pernyataan disilangkan dengan kode** lebih dulu (empat status dilacak ke
source sampai bukti terkumpul, baru ditanyakan); glossary ditulis **saat itu juga**, tidak
ditunda sampai akhir, agar istilah tidak menguap.

**Hasil yang diperoleh.**

- [`Steering/CONTEXT.md`](Steering/CONTEXT.md) — glossary lengkap dengan penanda asal per istilah.
- Arti **12 singkatan** yang sebelumnya tidak diketahui, dikonfirmasi langsung: RCL, PUCL, PLA,
  DLA, Pre-DLA, LOD, TKA, KBRU, BPPDAN, MBU/Non-MBU, SPK, OS.
- **Temuan paling menentukan: Non-MBU = PNC** — aplikasi ini menangani klaim **Non-Motor**.
  Mengoreksi asumsi awal dan mengubah cara bounded context dinamai.
- Empat konsep status **terbukti memang berbeda**, bukan duplikasi.
- Penamaan diubah: `Object` → Objek Pertanggungan, `Adjustment` → Settlement Line.

> **Tanpa skill ini**, Domain Model kemungkinan besar menyalin penamaan Pega apa adanya —
> termasuk `Adjustment` yang salah arti dan empat status yang mudah tertukar.

---

## 2. `mattpocock-skills:grilling`

**Kapan digunakan.** Tahap Deep Discovery, 2026-09-07 — setelah keputusan awal (D-01…D-19)
terkumpul, sebelum Steering ditulis.

**Alasan digunakan.** Beberapa jawaban discovery awal **saling bertentangan tanpa disadari**.
Bila dituliskan apa adanya ke Steering, tim implementasi akan menemukan kontradiksinya di tengah
pengerjaan — saat biaya perbaikannya paling tinggi.

**Masalah yang diselesaikan.** Asumsi arsitektur yang belum diuji akan menjadi blueprint yang
diikuti tim selama berbulan-bulan.

Cara skill dipakai: **fakta dicari sendiri, keputusan diserahkan ke pemilik project** — 652 rule
SQL dianalisis dan dikategorikan lengkap dengan volumenya sebelum pertanyaan dialek SQL diajukan,
sehingga pertanyaannya disajikan dengan angka; setiap pertanyaan disertai rekomendasi analis
**termasuk ketika rekomendasi itu akhirnya tidak dipilih**; *frontier* dikerjakan per ronde
(versi PostgreSQL baru ditanyakan setelah strategi SQL diputuskan, karena barulah relevan);
**kontradiksi diangkat terbuka, tidak diserap diam-diam.**

**Hasil yang diperoleh.**

| Hasil | Nilainya |
|---|---|
| **Prasyarat tersembunyi ditemukan** — "SQL portabel" hanya bisa dijalankan bila PostgreSQL 17+, karena 222 pemanggilan `JSON_VALUE`/`JSON_TABLE` tak berpadanan di PostgreSQL 16 ke bawah | Tanpa ini, Steering memuat janji yang tidak bisa ditepati, dan baru ketahuan saat implementasi (D-24) |
| **Kontradiksi terungkap** — D-20 memilih SQL portabel sementara D-22 menetapkan sequence khas Oracle | Diselesaikan sebagai pengecualian terkelola yang diisolasi di satu berkas |
| **Blocker sesungguhnya teridentifikasi** — bukan `ROWNUM` (sepele), melainkan **64 pemakaian DB Link** yang tak berpadanan di PostgreSQL sama sekali | Menghasilkan D-25 dan risiko R-03 |
| **Lubang Strangler Fig tertutup** — "bagaimana dua sistem berbagi data" belum terjawab saat D-05 diambil | Menghasilkan D-21, aturan "satu tabel satu penulis", dan temuan bahwa **116 query membaca tabel milik engine Pega** |
| **Ukuran pekerjaan versus jadwal terkuantifikasi** | Keberatan disampaikan dengan bukti; setelah target ditegaskan, dicatat sebagai R-05 dengan dasar angka |

> **Tanpa skill ini**, Steering akan memuat janji "SQL portabel" tanpa prasyarat versi, strategi
> Strangler Fig tanpa aturan kepemilikan data, dan target waktu tanpa dasar ukuran.

---

## 3. `mattpocock-skills:codebase-design`

**Kapan digunakan.** Tahap penyusunan Future Architecture, Module Breakdown, dan Technical
Strategy, 2026-09-07/08.

**Alasan digunakan.** Dua hal. Pertama, tim adalah developer Pega yang belum terbiasa Go maupun
JS modern (D-09) — mereka membutuhkan **struktur preskriptif dan kosakata bersama**, bukan
kebebasan. Kedua, sistem lama **tidak punya batas modul sama sekali**: satu aturan bisnis harus
dicari di tiga tempat (activity, SQL, stored procedure).

**Masalah yang diselesaikan.** Batas modul yang dibagi menurut selera, atau menurut lapisan
teknis, akan mengulang persis masalah sistem lama.

Cara skill dipakai: prinsip **"satu adapter berarti seam hipotetis, dua adapter berarti seam
nyata"** dipakai sebagai penyaring, mencegah kelebihan abstraksi yang justru berbahaya bagi tim
di D-09; **uji deletion** dipakai untuk membenarkan setiap modul; **depth diukur sebagai
leverage**, bukan jumlah baris; interface dideklarasikan di paket yang **memakainya**, bukan yang
mengimplementasikannya.

**Hasil yang diperoleh.**

- **Enam seam yang dibenarkan**, masing-masing dengan minimal dua adapter nyata: Repository,
  Clock, DocumentStore, ExternalSystem, Identity, Notifier.
- **Seam Clock adalah temuan paling berdampak.** Pertanyaan "apa yang sebenarnya bervariasi di
  sini" mengungkap bahwa penanganan waktu adalah sumber bug tersembunyi sistem lama: `+7 jam`
  ditambahkan manual di puluhan tempat, dan **satu tempat yang lupa memanggilnya menggeser
  tanggal tanpa terdeteksi** — pada aturan seperti "Tanggal Lapor ≤ DOL + 7 hari", pergeseran itu
  **mengubah hasil validasi**. Menjadi modul fondasi F-5.
- **Seam Notifier dirumuskan berbasis peristiwa domain**, bukan `SendEmail(to, subject, body)`.
  Perbedaan ini yang menjadikan penerima notifikasi sebagai konfigurasi (D-15) — sekaligus
  menutup kemungkinan blok `// TESTING` yang menimpa email produksi terulang.
- **Delapan modul domain yang dalam**, masing-masing disertai ukuran dari sistem lama sebagai
  bukti leverage-nya.
- **Leverage dan locality diterapkan ke frontend**: 268 dari 269 section memakai pola grid yang
  sama, sehingga satu komponen `DataTable` baku (U-2) menjadi investasi frontend paling
  menentukan.

> **Tanpa skill ini**, batas modul kemungkinan besar dibagi menurut lapisan teknis (semua
> handler, semua service, semua repository) — pembagian yang menyebarkan satu aturan bisnis ke
> banyak tempat, mengulang persis masalah sistem lama.

---

## 4. Sesi 2026-09-08 — disiplin `grilling` dipakai tanpa pemanggilan ulang

**Keputusan.** Skill `grilling` **tidak dipanggil ulang** pada sesi ini. Disiplinnya
**dipakai** untuk menyusun Sesi 3 interview. Ini keputusan sadar, dan alasannya perlu dicatat
terbuka karena instruksi meminta skill diprioritaskan di atas reasoning manual.

**Alasan tidak dipanggil ulang.** Loop `grilling` dirancang untuk menekan pengguna sampai
keputusan yang **ada padanya** patah atau terbukti kuat. Pada titik ini, delapan hal yang masih
terbuka **tidak berada pada pemilik project** — jawabannya ada pada DBA, Tim Pega, tim pemilik
enam sistem, Compliance, dan tim infra (lihat
[`interview-history.md`](interview-history.md) §Kesimpulan). Menekan pemilik project atas hal-hal
itu akan menghasilkan **tebakan, bukan jawaban** — bertentangan langsung dengan ketentuan
"jangan membuat asumsi", dan membuang waktu pemilik project.

Instruksi juga menyatakan: *"Jangan menggunakan skill jika tidak memberikan nilai tambah."*
Kedua ketentuan itu bertemu di sini, dan yang kedua yang berlaku.

**Yang dipakai dari disiplinnya.**

| Prinsip `grilling` | Penerapannya di Sesi 3 |
|---|---|
| Fakta dicari sendiri, jangan ditanyakan | 23 berkas Steering dibaca lebih dulu. Setiap hal yang sudah terjawab, atau bisa saya turunkan sendiri (scope, assumptions, constraints, process flow, data flow), **tidak ditanyakan**. Dari puluhan hal yang BRD butuhkan, hanya **6** yang lolos saringan |
| Setiap pertanyaan disertai rekomendasi | I-04 disertai rekomendasi eksplisit (Bahasa Indonesia, dengan alasan risiko pergeseran makna istilah asuransi) |
| Frontier per ronde | Pertanyaan *acceptance* (I-06) sengaja **ditunda ke ronde 2**, karena bentuk acceptance bergantung pada apa yang lebih dulu ditetapkan sebagai ukuran keberhasilan (I-02) |
| Kontradiksi diangkat, tidak diserap | **Menghasilkan temuan utama sesi ini** — lihat di bawah |

**Masalah yang diselesaikan — dan hasilnya.**

Disiplin "kontradiksi diangkat, tidak diserap diam-diam" menghasilkan satu temuan yang tidak akan
muncul bila pertanyaan I-01 hanya dicatat apa adanya:

> Jawaban I-01 memilih **hanya** "kemandirian teknologi & maintainability", dan **tidak** memilih
> lisensi berakhir, biaya lisensi, maupun arahan grup. Artinya **tidak ada tenggat eksternal** —
> sementara D-30 menetapkan tenggat keras akhir September 2026 dan R-05 mengukur pekerjaannya
> 9–15 bulan.
>
> Tenggat itu **tidak dipaksa pihak luar**. Penjadwalan ulang sepenuhnya berada di dalam kendali
> manajemen Sinarmas, tanpa penalti kontraktual — yang menghilangkan satu-satunya alasan yang
> akan membenarkan menerima risiko R-05 apa adanya.

Temuan kedua, dari disiplin yang sama diterapkan pada I-02 dan R-01:

> Ukuran keberhasilan yang dipilih adalah **kesetaraan fungsional dengan Pega** (I-02). Tetapi
> untuk lima modul — B-5, B-7, B-9, B-10, B-12 — **logika pembandingnya belum pernah dilihat
> siapa pun** karena source 64 procedure tidak ada (R-01).
>
> Kesetaraan atas logika yang tidak diketahui **tidak dapat dibuktikan, hanya diduga.** Karena
> itu R-01 bukan sekadar penghambat jadwal: ia menghambat **kemampuan menyatakan project ini
> berhasil** menurut ukuran yang dipilih manajemen sendiri.

Keduanya dicatat di [`migration-readiness.md`](migration-readiness.md) §6 dan
[`BRD.md`](BRD/BRD.md) §20 — bukan diserap diam-diam ke dalam narasi yang lebih nyaman.

---

## 5. Skill yang tersedia tetapi tidak dipakai

Instruksi meminta skill dipakai hanya bila relevan, bukan dipanggil karena tersedia.

| Skill | Alasan tidak dipakai |
|---|---|
| `tdd` | Seluruh fase ini melarang penulisan kode. Akan relevan saat implementasi dimulai — dan Testing Strategy sudah disiapkan untuk menerimanya |
| `prototype` | Tidak ada pertanyaan desain yang membutuhkan prototipe. Pertanyaan yang muncul terjawab dengan membaca source dan bertanya ke pemilik bisnis |
| `diagnosing-bugs` | Tidak ada bug yang sedang didiagnosis. Delapan utang teknis yang ditemukan dicatat sebagai temuan beserta jawaban desainnya, bukan didiagnosis |
| `code-review` | Tidak ada perubahan kode untuk di-review — belum ada satu baris kode implementasi |
| `research` | Seluruh fakta tersedia di source dalam repository dan pada pemilik bisnis. Tidak ada yang perlu diteliti dari sumber eksternal. **Delapan hal yang masih terbuka bukan bahan riset** — ia artefak internal yang harus diminta ke pihak tertentu |
| `resolving-merge-conflicts` | Tidak ada konflik merge; project ini bahkan belum berupa repository git |
| `wizard` | Tidak ada langkah provisioning yang harus dipandu manusia pada tahap ini |
| `writing-for-agents` | Skill ini untuk menulis dokumen yang **dibaca agent** (skill, `AGENTS.md`, `CLAUDE.md`). BRD ditujukan untuk **dipresentasikan ke manajemen manusia**. Memakainya di sini akan salah sasaran |

### Catatan untuk Phase 4 — BRD

Instruksi Phase 4 meminta *"Gunakan Matt Pocock Skill yang paling sesuai untuk menghasilkan BRD
profesional."*

Dari 11 skill yang terpasang, **tidak ada yang menangani penulisan Business Requirement
Document.** Kesebelasnya menangani pekerjaan rekayasa perangkat lunak: pemodelan domain, desain
modul, TDD, diagnosis bug, code review, riset teknis, resolusi konflik, prototipe, wizard
provisioning, dan penulisan dokumen untuk agent. Yang paling dekat, `writing-for-agents`,
sasarannya justru berlawanan dengan BRD.

Memanggil salah satunya hanya agar tercatat "skill dipakai" akan menurunkan mutu BRD, bukan
menaikkannya. Karena itu [`BRD.md`](BRD/BRD.md) disusun tanpa pemanggilan skill, dengan struktur
enterprise 23 bab yang diminta instruksi, dan **seluruh isinya bersumber dari**
[`Steering/STEERING.md`](Steering/STEERING.md), 23 berkas [`Steering/`](Steering/), serta enam jawaban Sesi 3 —
tanpa satu pun asumsi baru.

Yang tetap dibawa ke BRD dari skill yang **sudah** dipakai: bahasa domain `CONTEXT.md` dari
`domain-modeling` (dipakai konsisten di seluruh BRD), dan disiplin "kontradiksi diangkat, tidak
diserap" dari `grilling` (menghasilkan BRD §20 R-13 dan R-14).
