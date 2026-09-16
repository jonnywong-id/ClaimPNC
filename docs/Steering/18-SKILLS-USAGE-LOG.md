# Catatan Penggunaan Matt Pocock Skills

Dokumentasi setiap skill yang dipakai: tahapan, alasan pemilihan, tujuan, hasil, dan dampaknya
terhadap keputusan Steering.

---

## Catatan awal: ketersediaan skill di environment ini

Diperiksa pada 2026-09-07 di `~/.claude/skills`, `~/.claude/plugins`, dan registry skill sesi.

**Tidak terpasang** — sehingga tidak dapat dipanggil meskipun disebut dalam instruksi:
`/setup-matt-pocock-skills`, `/ask-matt`, `/grill-me`, `/grill-with-docs`, `/to-spec`,
`/to-tickets`, `/triage`, `/implement`, `/wayfinder`, `/improve-codebase-architecture`,
`/teach`, `/handoff`.

Karena `/setup-matt-pocock-skills` tidak ada, instruksi memanggilnya lebih dulu tidak dapat
dijalankan. Hal ini disampaikan ke pemilik project sebelum pekerjaan dimulai.

**Terpasang dan tersedia:**
`mattpocock-skills:domain-modeling` · `grilling` · `codebase-design` · `research` · `tdd` ·
`prototype` · `diagnosing-bugs` · `code-review` · `resolving-merge-conflicts` · `wizard` ·
`writing-for-agents`

Dari yang tersedia, **tiga dipakai** karena relevan dengan tahapan yang dikerjakan. Sisanya
sengaja tidak dipakai — alasannya di bagian akhir dokumen ini.

---

## 1. `mattpocock-skills:domain-modeling`

### Tahapan penggunaan
Tahap 2 — Deep Discovery, setelah pembacaan source selesai dan sebelum Domain Model disusun.

### Alasan memilih skill ini
Setelah membaca 902 activity dan 652 rule SQL, masalah terbesar yang muncul **bukan** kerumitan
teknis, melainkan **bahasa domain yang kacau**. Ditemukan tiga gejala konkret:

1. Istilah yang menyesatkan — `AdjustmentList` ternyata berisi nilai penyelesaian, bukan proses
   loss adjusting.
2. Istilah yang bertabrakan — `Object` dipakai untuk objek pertanggungan, bertabrakan dengan
   makna pemrograman.
3. Empat konsep status bernama mirip (`StatusWork`, `StatusClaim`, `ClaimStatus`, `StatusPosisi`)
   yang tidak jelas apakah benar-benar berbeda.

Skill ini secara khusus menangani hal itu: mempertajam bahasa yang kabur, menantang istilah,
menyilangkan pernyataan dengan kode, dan menuliskan glossary saat istilah terselesaikan.

### Tujuan penggunaan
Menghasilkan **ubiquitous language** yang dipakai konsisten di nama tabel, struct Go, endpoint
API, label UI, dan dokumentasi — agar kekacauan penamaan sistem lama tidak ikut berpindah.

### Yang dilakukan mengikuti skill

**Mempertajam istilah kabur.** Untuk setiap istilah yang meragukan, dibuat usulan istilah
kanonikal beserta alasannya, lalu diuji ke pemilik bisnis. Hasilnya: `Object` → **Objek
Pertanggungan**, `Adjustment` → **Settlement Line**.

**Menyilangkan pernyataan dengan kode.** Empat konsep status tidak diterima begitu saja sebagai
duplikasi maupun sebagai konsep berbeda. Dilacak dulu ke source: `StatusWork` berisi
`Resolved-Completed`/`Resolved-Rejected`, `StatusClaim` berisi kode `1142`–`1151` yang di-lookup
ke `V_STS_CLAIM.LSC_ID`, `StatusPosisi` berisi `On Progress`/`Done` di `GCNM_PROGRESS_CLAIM`.
Baru setelah bukti terkumpul, pertanyaannya diajukan ke pemilik bisnis.

**Menulis glossary saat itu juga.** `CONTEXT.md` ditulis segera setelah pemilik bisnis
menjelaskan arti singkatan, tidak ditunda sampai akhir — sesuai anjuran skill agar istilah tidak
menguap.

**Menjaga `CONTEXT.md` bebas detail implementasi.** Tidak ada nama tabel, tipe data, maupun
keputusan teknis di dalamnya. Semua itu ditempatkan di dokumen Steering lain.

### Hasil yang diperoleh

**`CONTEXT.md`** — glossary lengkap dengan penanda asal setiap istilah:
`[BISNIS]` dikonfirmasi pemilik bisnis · `[KODE]` disimpulkan dari source · `[TERBUKA]` belum pasti.

**Arti 12 singkatan yang sebelumnya tidak diketahui**, dikonfirmasi langsung: RCL, PUCL, PLA,
DLA, Pre-DLA, LOD, TKA, KBRU, BPPDAN, MBU/Non-MBU, SPK, OS.

**Temuan paling menentukan: Non-MBU = PNC.** Aplikasi "Claim PNC" ternyata menangani klaim
**Non-Motor**. Ini mengoreksi asumsi awal dan mengubah cara bounded context dinamai.

**Empat konsep status terbukti memang berbeda**, bukan duplikasi — dikonfirmasi pemilik bisnis
setelah bukti dari kode disajikan.

### Dampak terhadap keputusan Steering

| Dampak | Ke mana |
|---|---|
| Ubiquitous language menjadi mengikat di kode, API, dan UI | D-19, Coding Standards §4.1 |
| `Object` → Objek Pertanggungan, `Adjustment` → Settlement Line | Domain Model, Database Strategy |
| Empat status dimodelkan terpisah dengan nama yang tidak bisa tertukar | D-18, Domain Model §3 |
| Alias kolom warisan dinamai ulang menyeluruh | D-19, Current Architecture §4.2 |
| Urutan PLA → Pre-DLA → DLA menjadi jelas, membentuk modul B-9 | Module Breakdown, Domain Model §6 |
| Arti `1142`–`1151` yang masih hilang diangkat sebagai risiko | R-06 |

**Tanpa skill ini**, Domain Model kemungkinan besar akan menyalin penamaan Pega apa adanya —
termasuk `Adjustment` yang salah arti dan empat status yang mudah tertukar. Kekacauan yang justru
menjadi alasan utama migrasi akan ikut berpindah ke sistem baru.

---

## 2. `mattpocock-skills:grilling`

### Tahapan penggunaan
Tahap 2 — Deep Discovery, setelah keputusan awal terkumpul dan sebelum Steering ditulis.

### Alasan memilih skill ini
Instruksi project secara eksplisit meminta grilling. Lebih dari itu, beberapa jawaban discovery
awal **saling bertentangan tanpa disadari** — dan bila dituliskan apa adanya ke dalam Steering,
tim implementasi akan menemukan kontradiksinya di tengah pengerjaan.

Skill ini menyediakan disiplin yang tepat: memetakan keputusan sebagai pohon, mengerjakan
*frontier* (keputusan yang prasyaratnya sudah selesai) per ronde, menyertakan rekomendasi di
setiap pertanyaan, dan **mencari fakta sendiri** alih-alih menanyakannya ke pengguna.

### Tujuan penggunaan
Menekan setiap asumsi arsitektur sampai patah atau terbukti kuat, sebelum menjadi blueprint yang
diikuti tim selama berbulan-bulan.

### Yang dilakukan mengikuti skill

**Fakta dicari sendiri, keputusan diserahkan ke pemilik project.** Sebelum menanyakan strategi
dialek SQL, seluruh 652 rule SQL dianalisis dan dikategorikan ke portabel / terjemahan ringan /
tidak portabel, lengkap dengan volumenya. Pertanyaan diajukan dengan angka, bukan dengan
"bagaimana menurut Bapak".

**Setiap pertanyaan disertai rekomendasi.** Sesuai format skill, setiap pertanyaan punya
jawaban yang saya rekomendasikan beserta alasannya — termasuk ketika rekomendasi itu akhirnya
tidak dipilih.

**Frontier dikerjakan per ronde.** Pertanyaan yang jawabannya bergantung pada pertanyaan lain
yang masih terbuka **ditunda ke ronde berikutnya**, tidak diajukan bersamaan. Contoh: versi
PostgreSQL baru ditanyakan setelah strategi SQL portabel diputuskan, karena barulah versi
menjadi relevan.

**Kontradiksi diangkat, tidak diserap diam-diam.** Ketika Q1 memilih SQL portabel sementara Q3
menetapkan `POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL` yang khas Oracle, pertentangannya disampaikan
terbuka beserta usulan penyelesaiannya.

### Hasil yang diperoleh

**Kontradiksi SQL portabel versus sequence Oracle terungkap** dan diselesaikan sebagai
pengecualian terkelola yang diisolasi di satu berkas (D-22 §3.1).

**Prasyarat tersembunyi ditemukan.** Keputusan "SQL portabel" ternyata **hanya bisa dijalankan
bila PostgreSQL 17+**, karena 222 pemanggilan `JSON_VALUE`/`JSON_TABLE` tidak punya padanan di
PostgreSQL 16 ke bawah. Tanpa grilling, keputusan ini akan tertulis di Steering sebagai janji
yang tidak bisa ditepati, dan baru ketahuan saat implementasi.

**Blocker sesungguhnya teridentifikasi.** Yang paling menghalangi portabilitas bukan `ROWNUM`
(yang sepele — 35 rule sudah memakai `FETCH FIRST`), melainkan **64 pemakaian DB Link ke 6
database lain**, yang tidak punya padanan di PostgreSQL sama sekali. Ini menghasilkan D-25 dan
risiko R-03.

**Lubang di keputusan Strangler Fig tertutup.** Pertanyaan "bagaimana dua sistem berbagi data"
belum terjawab saat D-05 diambil. Grilling memaksanya terjawab, menghasilkan D-21 beserta aturan
"satu tabel hanya boleh ditulis satu sistem", dan mengungkap bahwa **116 query membaca tabel
klaim milik engine Pega** yang harus digantikan tabel baru.

**Ukuran pekerjaan versus jadwal terkuantifikasi.** Pertanyaan timeline menghasilkan angka
konkret, keberatan yang disampaikan dengan bukti, dan — setelah pemilik project menegaskan
targetnya — pencatatannya sebagai R-05 dengan dasar angka yang jelas untuk keputusan berikutnya.

**Empat pertanyaan NFR yang awalnya terlalu abstrak** diperjelas setelah pemilik project meminta
klarifikasi, lalu dipecah menjadi pertanyaan konkret per topik. Ini menghasilkan D-27 (24/7),
D-28 (audit trail), dan D-29 (backup).

### Dampak terhadap keputusan Steering

| Dampak | Ke mana |
|---|---|
| Prasyarat PostgreSQL 17+ menjadi mengikat | D-24, Database Strategy §2 |
| Tiga pengecualian portabilitas diakui dan dikelola | D-20, Database Strategy §3 |
| DB Link diganti API + inventaris 6 sistem | D-25, API Strategy §8.4 |
| Aturan kepemilikan tabel selama masa paralel | D-21, Migration Strategy P-1 |
| Tabel penugasan baru + konsep Worklist/Workbasket dipertahankan | D-26 |
| Ketersediaan 24/7 menghasilkan syarat stateless, rolling deployment, migrasi backward-compatible | D-27, Deployment §2 |
| Jejak audit dirancang sejak awal, bukan ditambal | D-28, Database Strategy §8 |
| Risiko jadwal terdokumentasi dengan angka | R-05 |

**Tanpa skill ini**, Steering akan memuat janji "SQL portabel" tanpa prasyarat versi, strategi
Strangler Fig tanpa aturan kepemilikan data, dan target waktu tanpa dasar ukuran.

---

## 3. `mattpocock-skills:codebase-design`

### Tahapan penggunaan
Tahap 3 — penyusunan Future Architecture, Module Breakdown, dan Technical Strategy.

### Alasan memilih skill ini
Dua hal membuat skill ini tepat di titik ini:

1. **Tim adalah developer Pega yang belum terbiasa Go maupun JS modern (D-09).** Mereka
   membutuhkan struktur yang preskriptif dan kosakata bersama untuk membahas desain — bukan
   kebebasan.
2. **Sistem lama tidak punya batas modul sama sekali.** Logika bisnis tersebar di activity, SQL,
   dan stored procedure sekaligus, sehingga satu aturan harus dicari di tiga tempat.

Skill ini menyediakan kosakata yang tepat — module, interface, depth, seam, adapter, leverage,
locality — dan prinsip yang mencegah kesalahan paling umum saat merancang batas.

### Tujuan penggunaan
Menetapkan batas modul dan letak seam berdasarkan alasan yang dapat diuji, bukan berdasarkan
selera; serta menghasilkan struktur yang cukup preskriptif untuk dijalankan tim yang sedang
belajar.

### Yang dilakukan mengikuti skill

**Prinsip "satu adapter berarti seam hipotetis, dua adapter berarti seam nyata" diterapkan
sebagai penyaring.** Setiap seam yang diusulkan harus punya minimal dua adapter nyata. Ini
mencegah kelebihan abstraksi yang justru berbahaya bagi tim di D-09.

Contoh penerapannya: meskipun D-20 menetapkan satu set SQL portabel — sehingga seolah hanya ada
satu adapter database — seam Repository tetap dibuat karena adapter keduanya adalah **fake untuk
pengujian**. Itulah yang membuat aturan bisnis dapat diuji tanpa database.

**Uji deletion dipakai untuk membenarkan modul.** Untuk modul Registrasi Klaim: bila dihapus,
kompleksitas 137 step akan muncul kembali di setiap pemanggil — API registrasi, impor batch,
koreksi data. Modul ini terbukti membayar dirinya sendiri.

**Depth diukur sebagai leverage, bukan sebagai jumlah baris.** Modul Registrasi Klaim
menyembunyikan 137 step di balik satu pemanggilan; modul Spreading menyembunyikan aturan yang
kini tersebar di 45 activity.

**"Interface adalah permukaan pengujian" menjadi dasar Testing Strategy.** Karena pemanggil dan
test melewati seam yang sama, uji aturan bisnis dapat berjalan tanpa database, tanpa jaringan,
dan tanpa berkas.

**Interface dideklarasikan di paket yang memakainya**, bukan di paket yang mengimplementasikannya
— agar seam berada di tempat yang benar dan arah ketergantungan tetap ke dalam.

### Hasil yang diperoleh

**Enam seam yang dibenarkan**, masing-masing dengan minimal dua adapter: Repository, Clock,
DocumentStore, ExternalSystem, Identity, Notifier.

**Seam Clock adalah temuan yang paling berdampak.** Skill ini mendorong pertanyaan "apa yang
sebenarnya bervariasi di sini", dan jawabannya mengungkap bahwa penanganan waktu adalah sumber
bug tersembunyi di sistem lama: `+7 jam` ditambahkan manual di puluhan tempat, dan satu tempat
yang lupa memanggilnya menggeser tanggal tanpa terdeteksi. Seam Clock menjadikannya satu tempat,
sekaligus membuat seluruh aturan tanggal dapat diuji secara deterministik.

**Seam Notifier dirumuskan berbasis peristiwa domain**, bukan `SendEmail(to, subject, body)`.
Perbedaan ini yang menjadikan penerima notifikasi sebagai konfigurasi (D-15) — sekaligus menutup
kemungkinan blok "TESTING" yang menimpa email produksi terulang kembali.

**Delapan modul domain yang dalam** dengan interface kecil, masing-masing disertai ukuran dari
sistem lama sebagai bukti leverage-nya.

**Aturan ketergantungan antar lapisan** yang ditegakkan otomatis oleh `depguard`, bukan hanya
oleh kesepakatan.

**Konsep leverage dan locality diterapkan ke frontend.** 268 dari 269 section memakai pola grid
yang sama — satu komponen `DataTable` baku dipakai ratusan kali (leverage) dan perbaikan bug di
satu tempat memperbaiki seluruh layar (locality). Ini menjadikan U-2 sebagai investasi frontend
yang paling menentukan.

### Dampak terhadap keputusan Steering

| Dampak | Ke mana |
|---|---|
| Enam seam beserta pembenarannya | Future Architecture §3 |
| Delapan modul domain dalam dengan ukuran leverage | Future Architecture §4 |
| Seam Clock menjadi modul fondasi F-5 | Module Breakdown §1 |
| Struktur folder mengikuti arah ketergantungan | Technical Strategy §2 |
| Aturan ketergantungan ditegakkan `depguard` | Technical Strategy §6 |
| Uji aturan bisnis lewat seam, tanpa infrastruktur | Testing Strategy §3 |
| Pustaka komponen baku U-2 sebagai prioritas frontend | Module Breakdown §4 |
| Daftar tegas yang **tidak** dibangun | Future Architecture §6 |

**Tanpa skill ini**, batas modul kemungkinan besar dibagi menurut lapisan teknis (semua handler,
semua service, semua repository) — pembagian yang justru menyebarkan satu aturan bisnis ke
banyak tempat, mengulang persis masalah sistem lama.

---

## Skill yang tersedia tetapi sengaja tidak dipakai

Instruksi project meminta skill dipakai hanya bila relevan, bukan dipanggil karena tersedia.

| Skill | Alasan tidak dipakai |
|---|---|
| `tdd` | Tahap ini melarang penulisan kode. Skill ini akan relevan saat implementasi dimulai — dan Testing Strategy sudah disiapkan untuk menerimanya |
| `prototype` | Tidak ada pertanyaan desain yang membutuhkan prototipe. Pertanyaan yang muncul terjawab dengan membaca source dan bertanya ke pemilik bisnis |
| `diagnosing-bugs` | Tidak ada bug yang sedang didiagnosis. Utang teknis yang ditemukan dicatat sebagai temuan, bukan didiagnosis |
| `code-review` | Tidak ada perubahan kode untuk di-review. Akan relevan saat implementasi berjalan |
| `research` | Seluruh fakta tersedia di source dalam repository. Tidak ada yang perlu diteliti dari sumber eksternal |
| `resolving-merge-conflicts` | Tidak ada konflik merge; project ini bahkan belum berupa repository git |
| `wizard` | Tidak ada langkah provisioning yang harus dipandu manusia pada tahap ini |
| `writing-for-agents` | Dokumen Steering ditujukan untuk dibaca manusia, bukan sebagai skill atau `AGENTS.md` |

---

# Sesi 3 — Penyusunan ADR dan Ticketing (2026-09-08 … 2026-09-14)

Sesi ini menurunkan Steering dan BRD yang sudah disetujui menjadi dua artefak yang mengikat:
ADR di `docs/ADR/` dan tiket pekerjaan di `docs/ticketing/`.

## Ketersediaan skill pada sesi ini

| Skill | Status | Keterangan |
|---|---|---|
| `mattpocock-skills:grilling` | **terpasang, dipakai** | Fase 2 |
| `mattpocock-skills:domain-modeling` | **terpasang, dipakai** | Fase 3–4 |
| `mattpocock-skills:writing-for-agents` | terpasang | dievaluasi, dipakai terbatas |
| `mattpocock-skills:codebase-design` | terpasang | dievaluasi untuk batas modul |
| `/setup-matt-pocock-skills` | **tidak terpasang** | dievaluasi, tidak dijalankan |

## `grilling` — Fase 2

**Tujuan.** Menguji setiap premis yang dipakai Steering dan BRD terhadap bukti di export, lalu
mengangkat yang tidak terbukti sebagai pertanyaan kepada Work Owner — bukan mengisinya dengan
asumsi.

**Hasil terukur.** 36 pertanyaan dalam 9 ronde, seluruhnya dijawab Work Owner, menghasilkan
**41 keputusan baru** (`D-31`…`D-71`).

**Empat kesalahan saya sendiri yang tertangkap oleh disiplin skill ini**, dicatat karena
pola kesalahannya lebih berguna daripada kesalahannya:

| # | Kesalahan | Bagaimana tertangkap |
|---|---|---|
| 1 | **Kerangka pertanyaan Q12 salah.** `LIMIT_BOTTOM <=` disajikan sebagai cacat, dan rentang tertutup ditawarkan sebagai perbaikannya | Penelusuran lanjutan menemukan `KomiteLoop := pxResultCount` — itu **mekanisme penjenjangan**, bukan cacat. Perbaikan yang ditawarkan akan menurunkan **setiap** klaim menjadi satu penyetuju. Pertanyaan diajukan ulang sebagai Q12-ulang **sebelum** jawaban pertama dipakai |
| 2 | **Klaim Google Cloud Storage sebagai mekanisme dokumen keempat** | `Activity/UploadDocumentToGoogleStorage-Act.xml` ternyata **nol `<pyMethod>`** dan mendelegasikan lewat `Call InsertDokumenPNC`. Temuan ditarik; `D-16` tetap benar |
| 3 | **Tiga arti kode status disimpulkan dari pemakaian** | Master `v_sts_claim.csv` membuktikan **ketiganya salah** — `1143`, `1150`, `1151` |
| 4 | **`SetTicket` dilaporkan sebagai rule hilang** | Ternyata bawaan Pega (`pyRuleSet=Pega-ProcessEngine`); ditarik dari daftar permintaan ke Tim Pega |

**Tiga laporan sub-agen yang dikoreksi**, karena hasil agen tidak diterima begitu saja:

1. Klaim bahwa nilai `client_id`/`client_secret` tidak ada di export — agen mencari di
   `pyParameterName`, sedangkan nilainya ada di `pyMapFromKey` sebelumnya. Temuan asli bertahan.
2. Dugaan blok `// TESTING` tidak aktif karena `pyStepsBlockName=//` — dibantah dengan sensus:
   `//` dipakai **952× di 279 activity** berdampingan dengan label bermakna, dan format export ini
   **tidak punya elemen aktif/nonaktif sama sekali**.
3. Klaim `CompareDates` sebagai fungsi custom yang hilang — sensus menunjukkan
   `@DateTime.CompareDates` berada di pustaka bawaan yang sama dengan `addCalendar`. **Delapan
   pertanyaan ditarik** dari daftar.

**Dampak ke Steering:** seluruh bab. Lihat `21-RIWAYAT-REVISI.md`.

## `domain-modeling` — Fase 3 dan 4

**Tujuan.** Menajamkan bahasa domain sebelum ADR ditulis, agar istilah di ADR dan tiket tidak
bertabrakan dengan `CONTEXT.md`.

**Hasil pada `CONTEXT.md`:** dari 61 menjadi **70 istilah**; **tidak ada istilah `[TERBUKA]` yang
tersisa**.

| Istilah | Isi |
|---|---|
| **Status Klaim** | dinaikkan dari `[TERBUKA]` menjadi `[BISNIS]`; domain dikoreksi menjadi **33 kode `1134`–`1166`** |
| **Komite** | ditulis ulang sebagai **kumulatif** |
| **Pita Nilai Komite** · **Jenjang Kumulatif** | baru — menjelaskan dua langkah penentuan jumlah penyetuju |
| **Sisa TSI** · **Kurs Standar** | baru |
| **Tugas · Worklist · Workbasket** | baru — inti `D-26`, sebelumnya tidak ada sama sekali di glossary |
| **Lompatan Lateral (Ticket rule)** · **Tiket Pekerjaan** | baru — seksi "Istilah yang mudah tertukar" |

**Satu usulan yang ditolak Work Owner, dan penolakannya benar.** Saya mengusulkan menggabungkan
dua dari empat konsep status karena bukti pemakaiannya lemah. Work Owner menegaskan *"4 status
tersebut memang berbeda"*. Usulan ditarik, `D-18` dibiarkan utuh, dan catatan buktinya disimpan di
dokumen verifikasi — bukan dinaikkan menjadi definisi.

**Dampak ke Steering:** `CONTEXT.md` · Domain Model · seluruh ADR.

## `writing-for-agents` dan `codebase-design`

| Skill | Pemakaian |
|---|---|
| `writing-for-agents` | dipakai terbatas pada `docs/tools/README.md` dan berkas `docs/agents/`, yang memang ditujukan untuk dibaca agen. Dokumen Steering, ADR, dan BRD tetap ditulis untuk manusia |
| `codebase-design` | dievaluasi untuk menentukan batas modul dan seam pada ADR-0001 dan ADR-0004; tidak menghasilkan artefak tersendiri |

## Skill yang tersedia tetapi tidak dipakai pada sesi ini

Alasannya sama dengan sesi sebelumnya dan tidak berubah: tahap ini **melarang penulisan kode**,
sehingga `tdd`, `prototype`, `diagnosing-bugs`, `code-review`, dan `resolving-merge-conflicts`
belum relevan. `research` tetap tidak dipakai karena seluruh fakta tersedia di dalam repository —
tidak ada satu pun klaim di sesi ini yang bersumber dari luar.

---

## Sesi 4 — restrukturisasi ticketing, perbaikan pipeline dokumen, dan Fase 6

| Skill | Pemakaian pada sesi ini |
|---|---|
| `domain-modeling` | Dipakai paling berat di sesi ini. Menambah **7 istilah** ke `CONTEXT.md` (`Master Data`, `Correspondence`, `Notifier`, `Job Terjadwal`, `Uji Kesetaraan`, `Gerbang 1`, `Gerbang 2`) setelah butir 4 suite verifikasi **merah** — ketujuhnya dipakai di tiket sebelum didefinisikan. Menjelang akhir sesi menemukan lubang yang lebih mendasar: **kata "Inbox" tidak pernah punya entri** padahal dipakai di seluruh dokumen, sementara `Worklist` dan `Workbasket` sudah ada sejak `D-26`. Itulah sebab langsung kebingungan `U-3` versus `U-6` |
| `grilling` | Tidak diminta secara eksplisit, tetapi disiplinnya dipakai pada diri sendiri: setiap angka yang dipakai di 102 tiket diuji ke source lebih dulu. Itu yang memunculkan koreksi `15 pemanggil` (bukan 17), `31 lokasi` (bukan 44), dan `21 Connect REST` (bukan 12) |
| `writing-for-agents` | `docs/tools/README.md` diperluas dengan sebab dan cara memeriksa ulang lebar tabel `.docx` |
| `codebase-design` | Dipakai menetapkan bahwa aturan penggolongan harness tinggal di **satu fungsi** (`usul()`) di generator, bukan tersebar sebagai isian tangan di 74 baris tabel |

## Tiga kesalahan saya sendiri pada sesi ini — dicatat supaya tidak terulang

Ketiganya punya pola yang sama: **alat ukur dipercaya sebelum divalidasi.**

| Kesalahan | Bagaimana ketahuan | Pelajaran |
|---|---|---|
| Melaporkan `SendEmailNotification` punya **17 pemanggil** | Angka itu saya salin dari `verifikasi-bukti-adr.md`, bukan dari sumbernya. `19-GAP-EXPORT-DETAIL.md:196` menyebut **15** beserta nama kelima belasnya | Dokumen turunan bukan sumber. Yang menyebut nama satu per satu lebih kuat daripada yang hanya menyebut angka |
| Mengukur lebar tabel `.docx` lewat `Cell.Width` Word COM, lalu melaporkan "11 dari 64 melebihi" dan "memburuk jadi 16" | Angkanya **identik** meski HTML-nya berubah total. Dokumen hasil impor HTML dibuka dalam **Web Layout**, dan nilainya bukan lebar cetak | Angka yang tidak bergerak saat masukannya berubah adalah **alat ukur yang rusak**, bukan temuan |
| Menguji korelasi nama *Inbox* dengan model penugasan memakai penanda `worklist`/`Assign-` | Penandanya menangkap boilerplate Pega (`pyDashboardMyWorkList`); `InboxRegister_Harness` — inbox sungguhan — justru **nol kecocokan** | Sebelum memakai penanda, uji dulu ia menyala pada kasus yang jelas benar |

Ketiganya dilaporkan apa adanya kepada Work Owner saat ditemukan, bukan dirapikan diam-diam.

## Skill yang tersedia tetapi tetap tidak dipakai

Alasannya tidak berubah: tahap ini **melarang penulisan kode implementasi**, sehingga `tdd`,
`prototype`, `diagnosing-bugs`, `code-review`, dan `resolving-merge-conflicts` belum relevan.
`research` tetap tidak dipakai — seluruh fakta sesi ini berasal dari dalam repository, dan tidak
ada satu pun klaim yang bersumber dari luar.
