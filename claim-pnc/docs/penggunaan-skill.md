# Penggunaan Skill — Sesi 2026-09-15

Ketentuan VI.2 instruksi menuntut pencatatan setiap pemakaian skill khusus: namanya, alasannya,
waktunya, keluarannya, dan manfaatnya.

## Ringkasan

**Tidak ada skill yang dipanggil pada sesi ini.** Berkas ini mencatat kenyataan itu beserta
alasannya, bukan daftar kosong yang dibiarkan tanpa penjelasan.

## Skill yang dipertimbangkan dan alasan tidak dipakai

| Skill | Kapan ia akan menolong | Kenapa tidak dipakai di sesi ini |
|---|---|---|
| `mattpocock-skills:codebase-design` | Merancang modul dalam, menempatkan seam, memutuskan batas antarmuka | Kosakatanya — *module*, *interface*, *depth*, *seam*, *adapter* — **sudah terpakai** di `docs/Steering/04-FUTURE-ARCHITECTURE.md`, yang menyebut skill ini sebagai sumber kosakatanya. Seam `Identity` beserta dua adapternya sudah dirancang di sana pada §3.5. Memanggil skill ini berarti merancang ulang sesuatu yang rancangannya sudah disetujui |
| `mattpocock-skills:tdd` | Membangun fitur secara test-first dengan siklus merah-hijau-refactor | Acceptance criteria tiketnya sudah berfungsi sebagai spesifikasi uji yang siap pakai, dan uji ditulis berdampingan dengan kode. Siklus formalnya tidak menambah apa pun di atas itu |
| `mattpocock-skills:domain-modeling` | Menyusun atau menajamkan kosakata domain, menulis `CONTEXT.md`, mencatat ADR | `docs/Steering/CONTEXT.md` dan 29 ADR sudah ada dan berstatus mengikat. Tahap login menyentuh istilah yang sedikit — pengguna, sesi, token — dan tidak satu pun menuntut pemodelan baru |
| `mattpocock-skills:diagnosing-bugs` | Menelusuri bug yang sulit atau regresi performa | Tidak ada bug yang perlu didiagnosis; kendala yang muncul bersifat perkakas dan lingkungan, bukan perilaku salah yang harus dilacak |
| `mattpocock-skills:research` | Meneliti pertanyaan terhadap sumber primer dan menuliskannya | Penghalang di sesi ini bukan pengetahuan yang bisa diteliti, melainkan **artefak yang tidak ada**: kontrak API HCC/HCQ. Tidak ada sumber yang dapat dibaca untuk menggantikannya, dan mengarangnya justru dilarang `docs/AGENTS.md` aturan 5 |
| `mattpocock-skills:code-review` | Meninjau perubahan terhadap standar repo dan spesifikasi asalnya | Belum ada titik pembanding: repo kode baru dibuat pada sesi ini dan tidak berada di bawah version control (`D-33` — VCS disiapkan Tim GitLab) |

## Catatan untuk tahap berikutnya

Dua skill kemungkinan besar berguna ketika modul bisnis mulai dikerjakan:

- **`mattpocock-skills:codebase-design`** saat membangun `B-2` Registrasi Klaim. Modul itu
  menyembunyikan 137 step `InputRegister_act` di balik satu antarmuka, dan itu persis keputusan
  kedalaman modul yang kosakata skill tersebut bantu tajamkan.
- **`mattpocock-skills:code-review`** begitu repo ini masuk version control, untuk meninjau
  perubahan terhadap standar `08-TECHNICAL-STRATEGY.md` dan terhadap acceptance criteria tiketnya
  sekaligus.

---

# Penggunaan Skill — Sesi 2026-09-17 (modul Master Status Progres 1)

## Ringkasan

**Tidak ada skill yang dipanggil pada sesi ini.** Seperti sesi sebelumnya, berkas ini
mencatat kenyataan itu beserta alasan per skill — bukan daftar kosong.

Satu catatan kejujuran yang perlu dibedakan: instruksi menuntut pertanyaan konfirmasi
diajukan sebelum menulis kode, dan tiga pertanyaan memang diajukan (tercatat di
`catatan-pengembangan.md` §9.3). Itu **pemenuhan instruksi**, bukan pemakaian skill
`grilling` — skillnya tidak dipanggil, dan menyebutnya "memakai skill" akan melebihkan
apa yang benar-benar terjadi.

## Skill yang dipertimbangkan dan alasan tidak dipakai

| Skill | Kapan ia akan menolong | Kenapa tidak dipakai di sesi ini |
|---|---|---|
| `mattpocock-skills:codebase-design` | Merancang modul dalam, menempatkan seam, memutuskan batas antarmuka | Kosakatanya sudah terpakai di `docs/Steering/04-FUTURE-ARCHITECTURE.md`, dan bentuk modulnya sudah ditetapkan di sana beserta `README.md` §"Aturan susunan yang mengikat". Modul ini **mengikuti pola yang sudah ada** (`auth` dan `portal`), bukan merancang pola baru: satu paket domain yang mendeklarasikan seam-nya sendiri, `usecase/`, dua adapter `repo/`, dan `http/`. Memanggil skill ini berarti merancang ulang keputusan yang sudah disetujui |
| `mattpocock-skills:domain-modeling` | Menyusun atau menajamkan kosakata domain, menulis `CONTEXT.md` | Layak dipertimbangkan, dan hampir dipakai — modul ini memang memperkenalkan dua istilah yang belum ada di `CONTEXT.md`: **Status Progres 1** dan **Posisi Klaim**. Yang membuatnya tidak perlu: keduanya **tidak ambigu di sumbernya**. Namanya terbaca langsung dari label isian Pega, dan artinya tegas dari satu tabel dengan tiga kolom. Yang menolong di sini bukan penajaman kosakata melainkan **pembacaan bukti** — dan itulah yang dikerjakan (lihat `keputusan-implementasi.md` §10.3). Usulan menambahkan kedua istilah ke `CONTEXT.md` dicatat di bawah |
| `mattpocock-skills:grilling` | Menekan asumsi sampai patah sebelum menjadi blueprint | Yang perlu ditekan pada sesi ini bukan asumsi arsitektur melainkan **arti sebuah kolom basis data**, dan itu dijawab dengan membaca export — bukan dengan bertanya. Tiga hal yang benar-benar menuntut keputusan manusia diajukan langsung sebagai pertanyaan bersama rekomendasi, tanpa perlu ronde bertingkat |
| `mattpocock-skills:tdd` | Membangun fitur secara test-first dengan siklus merah-hijau-refactor | Uji ditulis berdampingan dengan kode dan **benar-benar menemukan dua cacat** (galat portal dijawab 500; `detail` galat tidak tersalurkan ke layar). Siklus formalnya tidak akan menambah apa pun di atas itu, dan acceptance criteria-nya di sini adalah perilaku Pega yang sudah terbaca dari export |
| `mattpocock-skills:diagnosing-bugs` | Menelusuri bug sulit atau regresi performa | Kedua cacat yang muncul ditemukan uji dan sebabnya langsung terbaca dari pesan gagalnya; tidak ada yang perlu ditelusuri |
| `mattpocock-skills:research` | Meneliti pertanyaan terhadap sumber primer | Seluruh sumber primer ada di dalam repo — 2.634 berkas export. Tidak ada pertanyaan pada sesi ini yang jawabannya berada di luar |
| `mattpocock-skills:code-review` | Meninjau perubahan terhadap standar repo dan spesifikasi asalnya | Repo kini sudah di bawah git, sehingga hambatan sesi sebelumnya hilang. Tidak dipakai karena peninjauannya dikerjakan langsung terhadap `08-TECHNICAL-STRATEGY.md` dan `README.md` §"Aturan yang mengikat siapa pun yang menulis kode di sini" sembari menulis — dan hasilnya diverifikasi mesin (`go vet`, `gofmt`, `tsc`, dua suite uji), bukan hanya dibaca |
| `mattpocock-skills:prototype` · `resolving-merge-conflicts` · `wizard` · `writing-for-agents` | — | Tidak ada pertanyaan desain yang menuntut prototipe, tidak ada konflik merge, tidak ada langkah provisioning, dan dokumen sesi ini ditujukan untuk dibaca manusia |

## Usulan yang lahir dari sesi ini

**Dua istilah layak masuk `CONTEXT.md`**, diajukan ke Work Owner — bukan ditambahkan
sendiri, karena `CONTEXT.md` berstatus mengikat:

| Istilah | Usulan definisi | Dasar |
|---|---|---|
| **Status Progres 1** | Keterangan sudah sampai mana pekerjaan sebuah klaim pada satu **Posisi Klaim**. Dikelola sebagai master data; berjenjang dua tingkat, dan tingkat 2 merupakan rincian dari tingkat 1 | `POOLDATA.GCNM_MST_PROGRESS_KLAIM` dan `GCNM_MST_PROGRESS` |
| **Posisi Klaim** | Tahap perjalanan klaim tempat progres dicatat: Register, Survey, Komite, Akseptasi. Berbeda dari **Status Klaim** (33 kode) dan dari **Status Posisi Progres** (`On Progress`/`Done`) | `Activity/ViewStatusProgress_act-Act.xml`; `GCNM_PROGRESS_POSISI_PNC` |

Butir kedua patut diperhatikan khusus: `CONTEXT.md` sudah memuat **empat konsep status**
yang `D-18` tetapkan berbeda, dan **Posisi Klaim adalah konsep kelima** — bukan salah
satu dari keempatnya. Ia bersinggungan paling dekat dengan **Status Posisi Progres**
tetapi bukan hal yang sama: yang satu menyebut *tahap mana*, yang lain menyebut *sudah
selesai atau belum* pada tahap itu. Menyamakan keduanya akan mengulang persis kekacauan
yang `D-18` perbaiki.

## Catatan untuk tahap berikutnya

**`mattpocock-skills:domain-modeling` akan benar-benar berguna pada modul Status Progres
2.** Tabel `GCNM_MST_PROGRESS` memuat `STS_PROGRESS1` **dan** `STS_PROGRESS2` sekaligus,
ditambah kolom `TIPE` yang belum diketahui artinya dan tidak dipakai kueri mana pun yang
terbaca. Duplikasi kolom antartabel dan kolom bermakna ganda adalah persis jenis kekaburan
yang kosakata skill itu bantu tajamkan — dan pengalaman `TYPE_KOMITE` pada `D-70`, satu
kolom yang memikul dua arti berbeda, menunjukkan akibatnya bila dibiarkan.

**`mattpocock-skills:codebase-design` tetap relevan pada `B-2` Registrasi Klaim**, sesuai
catatan sesi pertama: modul itu menyembunyikan 137 step `InputRegister_act` di balik satu
antarmuka, dan itu keputusan kedalaman modul yang sungguh-sungguh.

---

# Penggunaan Skill — Sesi 2026-09-17 (Modul Master Rekening)

## Ringkasan

**Tidak ada skill khusus yang dipanggil pada sesi ini.** Seperti pada sesi sebelumnya,
berkas ini mencatat kenyataan itu beserta alasannya, bukan daftar kosong.

## Skill yang dipertimbangkan dan alasan tidak dipakai

| Skill | Kapan ia akan menolong | Kenapa tidak dipakai di sesi ini |
|---|---|---|
| `mattpocock-skills:codebase-design` | Menempatkan seam dan batas modul baru | Polanya sudah ditetapkan `04-FUTURE-ARCHITECTURE` §3 dan **sudah berwujud kode** di modul `auth` dan `portal`. Modul ini menyalin pola itu apa adanya — empat seam (`Repo`, `BankRepo`, `Kasir`, `Notifier`), masing-masing dua adapter. Merancang ulang justru berisiko menyimpang dari modul yang sudah berjalan |
| `mattpocock-skills:domain-modeling` | Menyusun kosakata domain baru | Kosakatanya tidak dikarang, melainkan **diekstraksi** dari 20 rule Pega. Yang dibutuhkan pembacaan sumber primer, bukan pemodelan |
| `mattpocock-skills:tdd` | Siklus merah-hijau-refactor | Spesifikasinya adalah perilaku sistem lama yang harus ditiru, bukan perilaku baru yang harus ditemukan. Uji ditulis berdampingan dengan kode dan mengunci perilaku yang sudah diketahui — 21 uji, seluruhnya tanpa basis data dan tanpa jaringan |
| `mattpocock-skills:diagnosing-bugs` | Menelusuri bug sulit | Dua kegagalan yang muncul (`rolldown` kehilangan binding native, `jsdom` menuntut Node lebih baru) adalah masalah **lingkungan**, bukan perilaku salah. Keduanya diselesaikan dengan membaca pesan galatnya dan memeriksa versi Node — bukan penelusuran |
| `mattpocock-skills:research` | Meneliti pertanyaan terhadap sumber primer | Sumber primernya ada di repo ini dan dibaca langsung. Yang **tidak** dapat diteliti adalah alamat API Kasir — ia tidak ada di export mana pun, dan mengarangnya dilarang `docs/AGENTS.md` aturan 5. Karena itu ia dicatat sebagai penghalang, bukan ditebak |
| `dataviz` | Membuat grafik atau dashboard | Tidak ada visualisasi data di modul ini |

## Perkakas non-skill yang dipakai, dan manfaatnya

| Perkakas | Dipakai untuk | Manfaat nyata |
|---|---|---|
| `grep` / `sed` atas XML Pega | Membongkar 20 rule: `pyBrowseSQL`, `pyStepsPreCondParamsWhen`, `pyStepsActivityName` | Menemukan aturan bisnis yang **tidak terlihat di harness** — sembilan kolom wajib, syarat approve, kode respons Kasir "1" dan "9", dan dua alias yang berarti dua kolom berbeda |
| `go vet` / `gofmt` / `tsc --noEmit` | Penegakan otomatis (`08-TECHNICAL-STRATEGY` §6) | Menangkap satu cacat tipe nyata (`exactOptionalPropertyTypes` pada prop opsional `TabelRekening`) sebelum sampai ke layar |

## Keputusan teknis yang lahir dari pembacaan sumber, bukan dari skill

1. **Lima tab menjadi satu endpoint dengan saringan**, bukan lima endpoint — karena
   kelima section lama nyaris identik dan justru berbeda di tempat yang tidak disengaja.
2. **Keputusan komite disimpan sebelum Kasir dihubungi** — lihat
   `keputusan-implementasi.md` §10.5.
3. **Identitas komite diambil dari sesi, bukan dari query string** — bila dari klien,
   siapa pun dapat melihat antrean komite mana pun.

## Perubahan cara kerja yang diminta Work Owner di tengah sesi

Work Owner meminta **seluruh asumsi didaftarkan lebih dulu** sebelum kode ditulis, bukan
ditulis lalu ditandai di dokumen. Urutannya berubah menjadi:

> gali export → daftarkan asumsi → minta konfirmasi → baru menulis kode

**Itu langsung terbayar dua kali di sesi yang sama:**

| Yang tertangkap | Bila urutannya tidak diubah |
|---|---|
| `ACCOUNT_TYPE` ternyata `"BIASA"`/`"VA"`, bukan jenis pemilik | Layar akan menulis nilai yang tidak dikenali Pega |
| `STS_AKTIF` ternyata `"Ya"`/`"Tidak"`, bukan `"1"`/`"0"` | **Setiap** rekening terbaca nonaktif, tanpa satu pun galat muncul |

Keduanya ditemukan karena penggalian diperdalam ke activity `Set*` — bukan berhenti di
SQL. **SQL hanya menunjukkan nama kolom, tidak pernah menunjukkan nilai yang sah.**

Satu koreksi juga lahir dari cara kerja ini: saya sempat menyatakan jalur penerima surel
as-is *buntu*, lalu menemukan sendiri bahwa `MST_USER_TEKNIK.OPERATOR_ID` sama dengan
nama login yang sudah kita simpan. Koreksi disampaikan **sebelum** Work Owner mengunci
keputusannya.

Tidak ada skill khusus yang dipanggil untuk semua ini; perkakasnya `grep`/`sed` atas XML
Pega, dan yang menentukan adalah **urutan kerjanya**, bukan alatnya.

---

# Penggunaan Skill — Sesi 2026-09-17 (Master Status Klaim)

## Ringkasan

**Tidak ada skill yang dipanggil pada sesi ini.** Berkas ini mencatat kenyataan itu beserta
alasannya per skill, bukan daftar kosong yang dibiarkan tanpa penjelasan.

Dua skill yang pada sesi sebelumnya ditandai *"kemungkinan besar berguna ketika modul bisnis mulai
dikerjakan"* ditimbang ulang di bawah, karena sesi inilah modul bisnis yang pertama.

## Skill yang ditimbang ulang karena ini modul bisnis pertama

| Skill | Yang dijanjikan catatan sesi lalu | Keputusan sesi ini |
|---|---|---|
| `mattpocock-skills:codebase-design` | Disebut akan berguna saat `B-2` Registrasi Klaim, yang menyembunyikan 137 step di balik satu antarmuka | **Tidak dipakai.** Modul ini bukan `B-2`. Kedalamannya kecil dan jujur: empat operasi di balik satu seam berisi empat metode, dan satu-satunya keputusan batas yang nyata — di mana keunikan label ditegakkan — dijawab oleh fakta basis data, bukan oleh kosakata desain. Memanggil skill untuk modul sedangkal ini akan menghasilkan pembenaran, bukan rancangan |
| `mattpocock-skills:code-review` | Disebut akan berguna begitu repo masuk version control | **Tidak dipakai.** Repo sudah di bawah git sejak 2026-09-16, sehingga syaratnya kini terpenuhi. Yang belum ada adalah **titik pembanding yang bermakna**: seluruh kode sesi ini baru, bukan perubahan atas kode yang sudah ada, sehingga peninjauan "sejak X" akan meninjau berkas kosong lawan berkas jadi. Ia akan berguna pada sesi berikutnya, yang mengubah kode ini |

## Skill lain dan alasan tidak dipakai

| Skill | Kapan ia menolong | Kenapa tidak di sesi ini |
|---|---|---|
| `mattpocock-skills:domain-modeling` | Menyusun atau menajamkan kosakata domain | Istilah modul ini sudah terdefinisi dan mengikat di `docs/Steering/CONTEXT.md` — **Status Klaim** beserta 33 kodenya diputuskan `D-18` dan `ADR-0018`, dan `R-06` ditutup dengan isi masternya. Tidak ada satu istilah pun yang perlu ditajamkan; yang ada justru larangan tegas untuk **tidak** menyimpulkan arti kode sendiri |
| `mattpocock-skills:tdd` | Membangun fitur secara test-first | Jawaban Work Owner sudah berfungsi sebagai spesifikasi uji yang siap pakai, dan uji ditulis berdampingan dengan kode. **Catatan jujur:** siklus merah-hijau formal mungkin menemukan cacat DOM ganda pada `TabelData` lebih awal — pengujian menemukannya, tetapi setelah komponennya selesai ditulis, bukan sebelum |
| `mattpocock-skills:diagnosing-bugs` | Menelusuri bug sulit atau regresi performa | Dua kendala yang muncul — DOM ganda dan proxy perusahaan — keduanya langsung terbaca dari keluaran uji dan jawaban HTTP. Tidak ada yang perlu ditelusuri |
| `mattpocock-skills:research` | Meneliti pertanyaan terhadap sumber primer | Seluruh fakta sesi ini berasal dari **dalam** repository dan dari katalog basis data: XML rule Pega, source procedure, CSV master, `ALL_TAB_COLUMNS`, `ALL_VIEWS`. Tidak satu pun klaim bersumber dari luar |
| `mattpocock-skills:prototype` | Menjawab pertanyaan desain lewat prototipe sekali pakai | Tidak ada pertanyaan desain yang menuntutnya. Yang paling mendekati — pilihan pustaka tabel — justru **sengaja tidak dijawab** di sini; `TKT-U2-005` menuntutnya diambil dengan pengukuran atas grid 39 kolom dan 10.000 baris, dan itu tiket tersendiri |
| `mattpocock-skills:grilling` | Menekan asumsi sampai patah | Tidak diminta. Disiplinnya tetap dipakai pada diri sendiri: setiap angka yang masuk kode diuji ke sumbernya lebih dulu — dan itulah yang menemukan bahwa tiga tebakan saya tentang skema basis data salah |
| `mattpocock-skills:resolving-merge-conflicts` | Menyelesaikan konflik merge | Tidak ada konflik |
| `mattpocock-skills:wizard` | Memandu langkah yang hanya dapat dikerjakan manusia | Ada kandidatnya — menjalankan migrasi `0002` menuntut DBA, persetujuan Work Owner, dan verifikasi bertahap. Tetapi `D-63` menetapkan prosedurnya menempuh **tiga pihak** dengan permintaan tertulis, dan bentuk yang tepat untuk itu adalah berkas migrasi yang dapat dibaca dan disetujui — bukan skrip interaktif yang dijalankan satu orang |
| `mattpocock-skills:writing-for-agents` | Menulis dokumen untuk agen | Dokumen sesi ini ditujukan untuk manusia: Work Owner, DBA, dan pengembang berikutnya |

## Teknik yang dipakai tanpa memanggil skill

Dicatat karena ketentuan VI.2 menuntut pencatatan **teknik**, bukan hanya nama skill.

| Teknik | Dari mana | Di mana terpakai | Manfaat nyata |
|---|---|---|---|
| **Seam dengan dua adapter** | kosakata `codebase-design`, sudah terserap ke `04-FUTURE-ARCHITECTURE.md` §3 | `masterstatus.Repo` diisi `sqlstore` dan `memori` | Seluruh 4 paket modul diuji **tanpa basis data**; layar juga dapat dicoba lengkap dengan 33 baris sebelum migrasi dijalankan |
| **Antarmuka dideklarasikan di paket pemakai** | idem | `masterstatushttp.Layanan` dideklarasikan di `http/`, bukan diimpor dari `usecase/` | Handler diuji tanpa membentuk seluruh layanan beserta penyimpanannya |
| **Uji sebagai pagar keputusan** | praktik repo ini sendiri, bukan skill | `TestFromDualHanyaDiKueriUrutan` · `TestSeamTidakMenyediakanOperasiHapus` · `TestSelisihDenganBasisDataProduksiTercatat` | Ketiganya tidak menguji kebenaran melainkan **mengunci keputusan**, sehingga membatalkannya menuntut percakapan, bukan sekadar menyunting kode |
| **Membaca katalog basis data sebelum menulis migrasi** | tidak dari skill mana pun | Perkakas diagnostik baca-saja sementara | Menemukan **tiga tebakan saya salah**, termasuk satu yang akan membuat migrasi gagal di baris pertama |

## Catatan untuk sesi berikutnya

- **`code-review`** kini benar-benar siap dipakai: repo di bawah git, dan sesi berikutnya akan
  mengubah kode yang sudah ada — bukan membuat dari kosong.
- **`tdd`** layak dicoba untuk komponen bersama berikutnya. Cacat DOM ganda pada `TabelData` adalah
  contoh nyata bahwa uji yang ditulis **sesudah** komponen selesai menemukan masalah lebih lambat
  daripada yang ditulis sebelum.

---

# Penggunaan Skill — Sesi 2026-09-17 (penataan ulang tampilan)

## Ringkasan

**Tidak ada skill yang dipanggil pada sesi ini.** Dua skill bertema visual tersedia dan keduanya
ditimbang lebih dulu; alasan tidak memakainya dicatat di bawah, bukan dibiarkan kosong.

## Skill bertema visual yang ditimbang

| Skill | Untuk apa ia ada | Kenapa tidak dipakai |
|---|---|---|
| `artifact-design` | Panduan desain untuk **Artifact** — halaman HTML mandiri yang diterbitkan ke claude.ai | Yang didesain di sini adalah **aplikasi React di dalam repository proyek**, bukan Artifact. Kontrak yang berlaku bukan kontrak halaman Artifact melainkan `docs/Steering/08-TECHNICAL-STRATEGY.md` §5 — TanStack Query untuk seluruh data, komponen tabel baku, Tailwind, tanpa CSS inline |
| `dataviz` | Panduan sebelum menulis kode grafik, bagan, atau dasbor | **Tidak ada satu pun grafik** di sesi ini. Yang dibangun adalah tabel, form, bilah atas, dan layar masuk. Memanggilnya untuk tabel data akan salah sasaran — tabel bukan visualisasi data |

## Skill lain dan alasan tidak dipakai

| Skill | Kenapa tidak di sesi ini |
|---|---|
| `mattpocock-skills:code-review` | Sesi lalu mencatat skill ini "kini benar-benar siap dipakai". Masih benar, dan tetap tidak dipakai: perubahan sesi ini **hampir seluruhnya kelas CSS**, dan peninjauan terhadap standar repo maupun spesifikasi tiket tidak punya banyak yang dapat dikatakan tentang nilai `hover:bg-indigo-700`. Ia akan lebih berguna pada sesi yang mengubah perilaku |
| `mattpocock-skills:tdd` | Sesi lalu mencatat skill ini layak dicoba untuk komponen bersama. Tidak dipakai karena pekerjaan ini **mengubah tampilan komponen yang perilakunya sudah punya uji** — 33 uji yang ada justru berfungsi sebagai jaring pengaman, dan itu terbukti: 14 di antaranya gagal begitu pemilih portal pindah, dan kegagalan itulah yang menunjukkan panggilan `/api/portal` kini terjadi di setiap layar |
| `mattpocock-skills:codebase-design` | Batas modul tidak berubah. Yang bergeser hanya **letak tiga kontrol** — Keluar, pemilih portal, nama pengguna — dari halaman ke kerangka, dan alasannya terbaca langsung dari cacatnya: layar master tidak punya tombol keluar |
| `mattpocock-skills:domain-modeling` | Tidak ada istilah domain baru. Sesi ini tidak menyentuh arti apa pun |
| `mattpocock-skills:diagnosing-bugs` | Satu kegagalan yang muncul — 14 uji master — langsung terbaca dari jejak galatnya (`data.portal.map`). Tidak ada yang perlu ditelusuri |
| `mattpocock-skills:research` | Seluruh fakta berasal dari dalam repository dan dari CSS hasil build sendiri |
| `mattpocock-skills:prototype` | Tidak ada pertanyaan desain yang menuntut prototipe sekali pakai |
| `run` | Aplikasi dijalankan langsung dengan `go build` lalu `curl` untuk memeriksa sajian aset dan rute dalam. Untuk pemeriksaan sesederhana itu, skill-nya tidak menambah apa pun |

## Teknik yang dipakai tanpa memanggil skill

| Teknik | Di mana terpakai | Manfaat nyata |
|---|---|---|
| **Token desain terpusat** (`@theme`) | `src/gaya.css` | Bayangan, lengkung, dan kurva gerak punya satu sumber. Mengubah kesan seluruh aplikasi berarti mengubah empat nilai, bukan menyisir puluhan berkas |
| **Verifikasi terhadap keluaran, bukan terhadap masukan** | Pemeriksaan CSS hasil build | Kelas Tailwind yang salah ketik **diam saja** — tidak ada galat, hanya gaya yang tidak muncul. Satu-satunya cara tahu ia benar adalah membaca CSS yang benar-benar dihasilkan |
| **Uji sebagai pagar, bukan beban** | 33 uji yang ada | Tidak satu pun dilonggarkan. Yang gagal justru menunjukkan akibat nyata dari pemindahan pemilih portal — akibat yang tidak saya sadari saat menulisnya |

## Kesalahan sendiri yang tercatat sesi ini

**Alat ukur dipercaya sebelum divalidasi — pola yang sama dengan sesi keempat.**

Pemeriksaan pertama terhadap CSS hasil build mencari `min-width:` dan tidak menemukan satu pun
breakpoint, lalu saya menyimpulkan tata letaknya tidak akan responsif sama sekali. Kesimpulan itu
salah pada dua hal sekaligus: Tailwind v4 memancarkan `@media (width>=48rem)` bukan `min-width:`,
dan nama kelasnya ditulis `.md\:table-cell` dengan garis miring terbalik sehingga pencarian teks
polos tidak pernah cocok.

Koreksinya disampaikan dalam alur kerja yang sama, sebelum menjadi kesimpulan yang dilaporkan.
Pelajarannya sudah dicatat sesi lalu dan terulang di sini: **sebelum menyimpulkan sesuatu tidak
ada, buktikan dulu alat pencarinya menyala pada kasus yang jelas ada.**

---

# Penggunaan Skill — Sesi 2026-09-18 (penamaan kode ke bahasa Inggris)

## Ringkasan

**Tidak ada skill yang dipanggil pada sesi ini.** Alasannya dicatat per skill di bawah, bukan
dibiarkan kosong.

Yang menentukan hasil sesi ini bukan skill melainkan **satu perkakas yang ditulis khusus untuk
pekerjaannya** — lihat "Perkakas yang dibuat sendiri".

## Skill yang ditimbang

| Skill | Kenapa masuk akal ditimbang | Kenapa tidak dipakai |
|---|---|---|
| `mattpocock-skills:domain-modeling` | Pekerjaan ini menyentuh **nama istilah domain**, tepat wilayah skill ini | Arti istilahnya **tidak berubah sama sekali** — hanya bahasanya. `CONTEXT.md` tidak disunting satu baris pun, dan padanan Inggrisnya (`SettlementLine`, `InsuredItem`) sudah ditetapkan `D-19` sejak awal. Skill ini menajamkan arti yang kabur; di sini tidak ada yang kabur |
| `mattpocock-skills:codebase-design` | Folder dan paket berganti nama | **Batas modulnya tidak bergeser satu pun.** `internal/masterrekening` menjadi `internal/bankaccount` dengan isi, seam, dan ketergantungan yang persis sama. Ini penggantian label, bukan perancangan ulang |
| `mattpocock-skills:code-review` | Perubahannya menyentuh 120+ berkas | Tinjauan terhadap standar tidak punya banyak yang dapat dikatakan tentang `Daftar` → `List`. Yang benar-benar menjaga sesi ini adalah **kompilator dan suite uji**, dan keduanya dijalankan pada setiap langkah |
| `mattpocock-skills:tdd` | — | Tidak ada perilaku baru. Uji yang ada justru berperan sebagai **spesifikasi yang tidak boleh berubah**, dan itu peran yang berlawanan dengan menulis uji lebih dulu |
| `mattpocock-skills:diagnosing-bugs` | Tujuh uji sempat merah | Seluruhnya terbaca langsung dari pesan uji (`Tidak ada rows yang cocok`). Tidak ada yang perlu ditelusuri |
| `mattpocock-skills:research` | — | Seluruh fakta berasal dari dalam repository |
| `mattpocock-skills:grilling` | Permintaannya memang tidak menyebut batas | **Disiplinnya dipakai tanpa memanggil skill-nya:** tiga pertanyaan diajukan sebelum satu berkas disentuh, masing-masing dengan pilihan jawaban. Jawaban kedua — "API dan basis data tetap Indonesia" — yang mengubah bentuk seluruh pekerjaan |

## Perkakas yang dibuat sendiri, dan kenapa `sed` tidak memadai

| Perkakas | Isi | Kenapa perlu |
|---|---|---|
| Pemindai pemecah kode | Memecah berkas menjadi potongan **kode** dan **bukan-kode** (komentar `//` dan `/* */`, literal string), lalu menerapkan peta nama pada potongan kode saja. Khusus template literal TypeScript, bagian `${…}` dikembalikan menjadi kode | `sed` merusak komentar (`Sesi` → `Session`), literal data (`"Aktif"` → `"Active"` — ini isi kolom basis data), dan teks layar. **Dua di antaranya tidak terdeteksi kompilator** |
| Pemulih komentar dokumentasi | Mengganti kata pertama komentar dokumentasi **hanya bila** baris deklarasi sesudahnya memang mendeklarasikan nama barunya | Komentar Go diawali nama yang didokumentasikannya. Mengganti butanya merusak prosa yang kebetulan diawali kata yang sama |
| Penghitung identifier | Mengeluarkan seluruh identifier di luar komentar dan string, terurut frekuensi | Menjawab "apa yang **masih** berbahasa Indonesia" secara terukur, bukan dengan membaca ulang dan berharap tidak terlewat |

## Teknik yang dipakai tanpa memanggil skill

| Teknik | Manfaat nyata |
|---|---|
| **Urutan tetap per modul**: ganti → build → vet → test → periksa komentar di `git diff` → perbaiki prosa | Langkah kelima yang paling sering menemukan sesuatu, dan ia tidak dapat digantikan kompilator: komentar yang rusak tetap dapat dikompilasi |
| **Pemulihan dari indeks git, bukan penambalan** | Saat ketahuan peta nama ikut mengganti **nama field API**, berkasnya dipulihkan dengan `git checkout -- src` (isi asli, jalur baru), peta diperbaiki, lalu dijalankan ulang. Menambal satu per satu akan meninggalkan sisa yang tidak terlihat |
| **Membuktikan kegagalan lama memang lama** | Tiga uji frontend gagal. Alih-alih menduga, sebuah `git worktree` pada `HEAD` dibuat dan suitenya dijalankan di sana — hasilnya sama persis. Dugaan menjadi bukti dengan satu perintah |

## Kesalahan sendiri yang tercatat sesi ini

**Peta nama diterapkan sebelum batasnya diverifikasi terhadap kontrak.**

Peta putaran pertama memuat `pengguna → user`, `siap → ready`, `catatan → note`, `kode → code`, dan
`batas → limit`. Kelimanya **juga nama field JSON API** — hal yang Work Owner sudah nyatakan tidak
boleh disentuh pada jawaban nomor 2, sebelum pekerjaan dimulai. Akibatnya `LoginResponse.pengguna`
menjadi `LoginResponse.user`, dan kontraknya diam-diam rusak.

Ketahuan dari `tsc`, bukan dari pembacaan ulang. Dipulihkan dengan mengembalikan seluruh `src` dari
indeks git, membuang kelima kunci itu dari peta, lalu menjalankannya ulang.

> Pelajarannya: **batasan yang sudah dinyatakan di muka harus diterjemahkan menjadi penyaring di
> dalam perkakas, bukan disimpan sebagai kehati-hatian di kepala.** Daftar nama field JSON dapat
> ditarik dari tag `json:"…"` di backend dengan satu perintah — dan seharusnya ditarik **sebelum**
> peta disusun, bukan sesudah kerusakannya terlihat.

**Pola lama yang terulang: alat ukur dipercaya sebelum divalidasi.** Pemindai pemecah kode dianggap
menangkap seluruh teks yang tidak boleh disentuh. Ia tidak menangkap dua kelas — **teks JSX** dan
**literal regex** — dan keduanya baru terlihat sebagai uji merah. Kali ini jaringnya sudah
terpasang, jadi akibatnya tertahan; tetapi yang menahannya adalah uji yang ditulis sesi-sesi
sebelumnya, bukan kehati-hatian sesi ini.

## Catatan untuk sesi berikutnya

- Aturan penamaannya kini ada di `CLAUDE.md` (`D-80` dan §4.1), sehingga kode baru mengikutinya
  tanpa perlu ditanyakan lagi.
- Kamus istilahnya ada di [`peta-penamaan.md`](peta-penamaan.md), termasuk **daftar yang sengaja
  tidak diterjemahkan**. Bacalah daftar itu lebih dulu sebelum mengganti nama apa pun.

---

# Penggunaan Skill — Sesi 2026-09-18 (menyelesaikan merge & Master Status Progres 1)

## Ringkasan

**Tidak ada skill yang dipanggil pada sesi ini.** Alasannya dicatat per skill, bukan dibiarkan
kosong.

Yang menentukan hasil sesi ini bukan skill melainkan **satu pemeriksaan yang dijalankan sebelum
pekerjaan dimulai**: `go build ./...` dan `npx tsc --noEmit` atas keadaan awal. Keduanya gagal, dan
kegagalan itulah yang mengungkap bahwa permintaannya bertumpu pada merge yang belum diselesaikan.

## Skill yang ditimbang

| Skill | Kenapa masuk akal ditimbang | Kenapa tidak dipakai |
|---|---|---|
| `mattpocock-skills:resolving-merge-conflicts` | Sesi ini **memang** menyelesaikan konflik merge — satu-satunya sesi sejauh ini yang demikian | Skill itu menangani konflik yang **sedang** terjadi di working tree, dengan `git status` menyebut berkas mana yang bentrok. Di sini konfliknya **sudah ter-commit**: `git status` bersih, dan yang tersisa hanya penanda `<<<<<<<` di dalam berkas. Yang dibutuhkan adalah membaca kedua sisi dari `git show <sha>:<berkas>` lalu memutuskan per blok, bukan alur `git mergetool` |
| `mattpocock-skills:diagnosing-bugs` | Repo tidak dapat di-build, dan sebabnya tidak disebut permintaan | Penyebabnya terbaca dari pesan kompilator dalam satu langkah (`syntax error: unexpected <<`). Yang perlu ditelusuri hanya **sejarahnya**, dan itu satu perintah `git log --graph` |
| `mattpocock-skills:domain-modeling` | Modul yang diterjemahkan penuh dengan istilah domain | Arti istilahnya tidak berubah sedikit pun — hanya bahasanya. `CONTEXT.md` tidak disunting satu baris pun |
| `mattpocock-skills:codebase-design` | Satu modul berpindah paket dan satu kerangka dihapus | Batas modulnya tidak bergeser. Yang dihapus bukan modul melainkan **percobaan kerangka yang sudah ditinggalkan di cabangnya sendiri** — penilaian bukti, bukan perancangan ulang |
| `mattpocock-skills:tdd` | — | Tidak ada perilaku baru. Uji yang ada justru berperan sebagai **spesifikasi yang tidak boleh berubah** |
| `mattpocock-skills:code-review` | 60+ berkas berubah | Yang menjaga sesi ini adalah kompilator, `go vet`, dan 60 uji — dan ketiganya dijalankan pada setiap langkah, bukan sekali di akhir |

## Perkakas yang dipakai ulang dari sesi sebelumnya

| Perkakas | Isi | Perubahan sesi ini |
|---|---|---|
| Pemindai pemecah kode | memecah berkas menjadi potongan **kode** dan **bukan-kode**, lalu menerapkan peta nama pada potongan kode saja | ditambah mode **daftar berkas eksplisit**, supaya peta nama bergenerik (`Kode`, `Pesan`, `Daftar`) dapat dipakai pada satu modul tanpa menyentuh modul lain |
| Penghitung identifier | mengeluarkan identifier di luar komentar dan literal | ditulis ulang memakai `Map`, karena versi lama memakai objek biasa dan **pecah** saat identifier bernama `add` muncul di kode |
| Penyelesai konflik | membaca kedua sisi blok `<<<<<<</=======/>>>>>>>` dan memilih satu | **baru** — dua varian: "selalu HEAD" untuk kode, dan "yang terisi; bila keduanya terisi ambil sisi cabang" untuk dokumen |

## Teknik yang dipakai tanpa memanggil skill

| Teknik | Manfaat nyata |
|---|---|
| **Ukur keadaan awal sebelum menyentuh apa pun** | Inilah yang menemukan merge yang belum selesai. Bila langsung mengerjakan permintaan apa adanya, kegagalan build akan tampak seperti akibat pekerjaan sendiri |
| **Peta nama dijalankan bertahap, bukan sekaligus** | Tiga putaran: identifier modul, rujukan antarpaket, lalu sisa variabel lokal. Setelah tiap putaran, penghitung identifier dijalankan ulang untuk melihat **apa yang masih tersisa secara terukur** — bukan dengan membaca ulang dan berharap tidak terlewat |
| **Membuktikan kode mati memang mati** | `Kerangka.tsx` dihapus setelah `git show 4481dda:…/App.tsx` membuktikan cabang yang melahirkannya pun tidak memakainya. Tanpa langkah itu, penghapusannya hanya dugaan |
| **Membedakan nama internal dari nama kontrak sebelum mengganti** | Seluruh tag `json:"…"` di backend ditarik dengan satu `grep` dan dibandingkan dengan peta nama, sebelum peta dijalankan. Itu yang menahan `aktif` — nama field API — ikut terganti |

## Kesalahan sendiri yang tercatat sesi ini

**Laporan "seluruh pemeriksaan bersih" pada sesi lalu menjadi menyesatkan.** Ia benar saat
diverifikasi, dan menjadi salah sesudah merge. Pelajarannya bukan "jangan melapor", melainkan
**laporan verifikasi perlu menyebut commit yang diverifikasi** — tanpa itu, pembaca tidak punya cara
mengetahui kapan laporan itu berhenti berlaku.

**Perbaikan `GalatAPI.field` sesi lalu salah arah.** Saya menyimpulkan `field` tidak ada dari
membaca kelasnya, lalu menggantinya dengan `detail`. Yang seharusnya dibaca adalah **kontrak
backend** — di sana `field` ada dan `detail` tidak. Akibatnya pesan galat per kolom pada satu form
berhenti muncul tanpa satu pun tanda.

> Pola yang sama dengan dua sesi sebelumnya, dalam bentuk lain: **sumber kebenaran dibaca di tempat
> yang salah.** Untuk bentuk data yang menyeberangi jaringan, sumbernya dto di backend — bukan kelas
> di frontend yang kebetulan sedang dibaca.

## Catatan untuk sesi berikutnya

- Kontrak galat validasi **belum seragam antar tiga modul master** (`detail`/`field`, `field`/`kolom`).
  `APIError.violations()` menutupinya di satu tempat; penyeragaman sesungguhnya adalah `TKT-F1-004`.
- Master Status Progres **tingkat 2** sudah lengkap di backend tetapi belum punya layar dan belum
  dipasang di `cmd/claimpnc`.

---

# Penggunaan Skill — Sesi 2026-09-19 (menu kiri dari basis data)

## Ringkasan

**Tidak ada skill yang dipanggil pada sesi ini.** Alasannya dicatat per skill, bukan dibiarkan
kosong.

Yang menentukan hasil sesi ini adalah **satu pencarian yang dijalankan sebelum pekerjaan dimulai**:
mencari ketiga nama tabel ke seluruh export dan ke seluruh dokumen proyek. Keduanya nihil, dan
justru nihil itulah temuannya — ia mengubah pekerjaan dari "meniru perilaku Pega" menjadi
"membangun kemampuan baru terhadap kontrak yang diberikan".

## Skill yang ditimbang

| Skill | Kenapa masuk akal ditimbang | Kenapa tidak dipakai |
|---|---|---|
| `mattpocock-skills:domain-modeling` | Sesi ini memperkenalkan istilah baru ke dalam domain: **menu**, **otorisasi**, **group** | Arti ketiganya sudah ditetapkan DDL dan isi tabelnya, bukan hasil perundingan istilah. Yang perlu dikerjakan adalah membaca kolomnya dengan benar, bukan menajamkan artinya. `CONTEXT.md` tidak disunting — lihat "Catatan untuk sesi berikutnya" |
| `mattpocock-skills:codebase-design` | Modul baru, seam baru, dan satu keputusan batas yang sungguh-sungguh: peta rute di frontend atau di backend | Kosakatanya dipakai tanpa memanggil skill-nya. Keputusan batasnya diambil dengan satu pertanyaan yang lebih tajam daripada "di mana seam-nya": **siapa yang TAHU jawabannya**. Rute antarmuka hanya diketahui frontend, jadi di sanalah petanya |
| `mattpocock-skills:grilling` | Permintaannya menyisakan dua hal yang tidak dapat disimpulkan dari data | Disiplinnya dipakai tanpa memanggil skill-nya: dua pertanyaan diajukan dengan pilihan jawaban, masing-masing disertai pratayang bentuk layarnya. Jawaban kedua **tidak memilih satu pun pilihan** melainkan menuliskan urutan langkahnya sendiri — dan itu jawaban yang lebih baik daripada ketiga pilihan yang saya tawarkan |
| `mattpocock-skills:tdd` | Perilaku baru, bukan penggantian nama | Uji ditulis berbarengan, bukan lebih dulu. Aturan tampilnya dibaca DARI DATA, dan menulis uji sebelum datanya dibaca berarti menguji aturan yang saya karang sendiri |
| `mattpocock-skills:research` | — | Seluruh fakta ada di dalam repository |
| `mattpocock-skills:code-review` | — | Yang menjaga sesi ini adalah kompilator, `go vet`, 24 paket uji Go, dan 71 uji frontend — dijalankan pada setiap langkah |

## Teknik yang dipakai tanpa memanggil skill

| Teknik | Manfaat nyata |
|---|---|
| **Cari dulu, jangan tanya dulu** | Pertanyaan yang hampir saya ajukan — "bagaimana bentuk tabelnya?" — terjawab satu perintah `ls` di folder `Database/`: DDL lengkap dan lima CSV. Menanyakannya akan memindahkan pekerjaan membaca ke Work Owner |
| **Baca aturan dari data, bukan dari dugaan** | Aturan "kelompok tampil bila ada anaknya" bukan pilihan gaya: group `IT` tidak punya satu pun izin atas kelompok, dan aturan yang wajar (kelompok harus punya izin sendiri) akan menghapus seluruh menunya. Dugaan itu akan lolos uji yang saya tulis sendiri, dan gagal di layar |
| **Diff dua daftar untuk menguji dokumen lama** | `menu_program` versus isi direktori `Harness/` menghasilkan dua daftar yang keduanya bermakna: 9 harness hilang (memperjelas `K-33`) dan 8 harness yang tidak muncul di menu — dan kedelapan itu persis harness berkelas `Work`, menguatkan Lampiran G tanpa ada yang meminta |
| **Isi contoh dibangkitkan dari CSV, bukan diketik** | 80 baris menu disalin lewat skrip. Diketik tangan, satu digit `MENU_ID_LEADER` yang salah akan menggantung sebuah butir di kelompok yang keliru — dan tidak ada yang memeriksanya |
| **Tembak aplikasi yang benar-benar berjalan** | Uji membuktikan bentuknya benar; hanya permintaan HTTP sungguhan yang membuktikan RAKITANNYA benar. Instans sementara di porta lain dipakai supaya server Work Owner tidak terganggu |

## Satu hal yang saya siapkan tetapi tidak jadi dipakai

Saya menyiapkan perubahan pada modul Login, karena kunci pencocokan yang ditetapkan Work Owner —
"login yang diketik" — tampak tidak tersimpan di mana pun: `Profile.Login` untuk karyawan diisi
nilai dari HCQ, bukan dari yang diketik.

Sebelum menyentuhnya, contoh respons HCQ di `hcq_test.go` dibaca. Di sana terlihat HCQ
**memantulkan kembali** nilai yang dikirim: permintaan membawa `Login: k.Username`, respons
mengembalikan `Person.Login` dengan nilai yang sama. Tidak ada yang perlu diubah.

> Pelajarannya: **sebelum mengubah modul yang sudah selesai, baca dulu contoh datanya.** Membaca
> kodenya saja menunjukkan apa yang MUNGKIN terjadi; contoh datanya menunjukkan apa yang BENAR-BENAR
> terjadi.

## Catatan untuk sesi berikutnya

- **`CONTEXT.md` belum memuat istilah `Menu`, `Otorisasi`, dan `Group`.** Ketiganya kini punya arti
  tepat di dalam sistem ini dan dipakai di kode, sehingga layak masuk glossary. Tidak dikerjakan di
  sesi ini karena `CONTEXT.md` adalah dokumen Steering yang perubahannya ditulis sebagai keputusan,
  bukan disunting menyusul.
- Menambah layar baru kini menuntut **satu baris** di `frontend/src/app/menu/registry.ts`. Bila
  butirnya tetap tampak "belum tersedia", yang pertama diperiksa adalah ejaan `MENU_PROGRAM`-nya —
  ia dicocokkan persis, termasuk huruf besar-kecilnya.

# Penggunaan Skill — Sesi 2026-09-18 (Modul Pelaporan Klaim)

Ketentuan VI.2 instruksi menuntut pencatatan setiap pemakaian skill khusus: namanya, alasannya,
waktunya, keluarannya, dan manfaatnya.

## Ringkasan

**Tidak ada skill khusus yang dipanggil pada sesi ini.** Seperti pada empat sesi sebelumnya,
berkas ini mencatat kenyataan itu beserta alasannya, bukan daftar kosong yang dibiarkan tanpa
penjelasan.

Satu skill sempat **relevan dan tetap tidak dipakai** — alasannya di bawah, dan ia berbeda dari
alasan sesi-sesi sebelumnya.

## Skill yang dipertimbangkan dan alasan tidak dipakai

| Skill | Kapan ia akan menolong | Kenapa tidak dipakai di sesi ini |
|---|---|---|
| `mattpocock-skills:codebase-design` | Menempatkan seam dan batas modul baru | Polanya sudah ditetapkan `04-FUTURE-ARCHITECTURE` §3 dan **sudah berwujud kode** di empat modul. Modul ini menyalinnya apa adanya: satu seam (`Repo`) dengan dua adapter, lapisan `usecase` yang tidak tahu HTTP maupun SQL, dan `http/` yang mendeklarasikan antarmuka sempitnya sendiri. Merancang ulang justru berisiko menyimpang dari modul yang sudah berjalan |
| `mattpocock-skills:domain-modeling` | Menyusun kosakata domain baru | Kosakatanya **diekstraksi**, bukan dikarang: 26 nama kolom dari satu procedure, 34 nama properti dari seluruh export, dan lima nama tahap dari judul tab. Yang dibutuhkan pembacaan sumber primer, bukan pemodelan. Satu istilah yang memang baru — `Tahap` — diturunkan langsung dari ekspresi `CASE` di kueri lama |
| `mattpocock-skills:tdd` | Siklus merah-hijau-refactor | **Di sinilah alasannya berbeda dari sesi lalu.** Siklus TDD menuntut uji dapat DIJALANKAN untuk melihat merah lalu hijau. Go tidak terpasang di mesin ini, sehingga tidak ada satu pun titik dalam siklus itu yang dapat diamati. Uji tetap ditulis — 40 uji Go dan 16 uji frontend — tetapi menyebutnya TDD akan menyiratkan ia pernah merah lalu hijau, dan itu tidak terjadi |
| `mattpocock-skills:diagnosing-bugs` | Menelusuri bug sulit atau regresi | Tidak ada bug yang dapat didiagnosis: tanpa kompilator dan tanpa peramban, tidak ada perilaku salah yang dapat diamati. Yang muncul adalah cacat **repository** (penanda konflik merge di dua berkas markdown), dan itu ditemukan dengan `grep`, bukan dengan penelusuran |
| `mattpocock-skills:research` | Meneliti pertanyaan terhadap sumber primer | Sumber primernya ada di repo ini dan dibaca langsung — 20 rule Pega, dua procedure, satu navigation. Yang **tidak** dapat diteliti adalah bentuk nomor laporan lama: ia ditentukan `pyWorkIDPrefix` pada rule kelas yang tidak diekspor. Karena itu ia dicatat sebagai keputusan baru beserta alasannya, bukan ditebak |
| `mattpocock-skills:code-review` | Meninjau perubahan terhadap standar repo | Repo sudah di bawah git kali ini, sehingga syaratnya terpenuhi — tetapi peninjauan tanpa kompilator hanya dapat memeriksa gaya dan disiplin, bukan kebenaran. Yang dikerjakan sebagai gantinya adalah lima pemeriksaan manual di `catatan-pengembangan.md` §11.11, dan keterbatasannya dinyatakan apa adanya |
| `dataviz` | Membuat grafik atau dashboard | Lencana angka pada tab bukan visualisasi data — ia enam bilangan bulat. Menariknya ke pustaka grafik akan menambah dependensi untuk sesuatu yang muat di satu `<span>` |
| `anthropic-skills:xlsx` | Membaca atau menulis berkas spreadsheet | Dua CSV di `Database/` dibaca sebagai teks biasa lewat `grep`; tidak ada spreadsheet yang dihasilkan. Export CSV milik `ExportNotTransferRCV` berada di luar lingkup sesi ini |

## Perkakas non-skill yang dipakai, dan manfaatnya

| Perkakas | Dipakai untuk | Manfaat nyata |
|---|---|---|
| `grep` / `sed` atas XML Pega | Membongkar 20 rule: `pyBrowseSQL`, `pyStepsActivityName`, `PropertiesName`/`PropertiesValue`, `pyCaptionPrompt` | Menemukan yang **tidak terlihat di harness**: 26 pemetaan alias, daur hidup lima tahap, tujuh peran, dan judul menu "Inbox Laporan Klaim" yang membuktikan nama modulnya bukan karangan |
| Skrip ekstraksi langkah activity | Mengubah XML berisi 135 KB boilerplate menjadi daftar langkah yang terbaca | `CreateNewCaseRCV` dan `rcv_InsertRecivedDocumentClaim` — dua activity terpenting — dapat dibaca utuh dalam satu tampilan alih-alih ditelusuri baris per baris |
| Hitung kurung per berkas Go | Pengganti sebagian dari kompilator yang tidak ada | Dua selisih yang muncul terlacak ke kurung **di dalam string literal**; nol cacat sintaks nyata |
| `comm` atas daftar konstanta | Memastikan 19 nama field yang dipakai benar-benar dideklarasikan | Nol selisih — pemeriksaan yang biasanya dikerjakan kompilator dalam sepersekian detik |

## Catatan jujur: yang hilang karena perkakasnya tidak ada

Empat sesi sebelumnya menutup pekerjaannya dengan tabel hasil `go vet`, `go test`,
`npm run periksa-tipe`, dan uji asap terhadap binary yang benar-benar berjalan. Sesi ini tidak
dapat.

Yang paling terasa hilang bukan uji otomatisnya, melainkan **kompilator**. Pada sesi keempat,
`tsc --noEmit` menangkap satu cacat tipe nyata sebelum sampai ke layar; pada sesi kelima,
pengujian menemukan cacat satu-DOM-dua-pohon yang **tidak ketahuan dari membaca ulang kode**.
Kedua jaring itu tidak ada di sesi ini.

Pemeriksaan manual yang dikerjakan sebagai gantinya menutup satu kelas kesalahan — nama yang tidak
ada, jumlah argumen yang tidak cocok, urutan kolom yang bergeser — dan **tidak menutup sisanya**.

## Catatan untuk tahap berikutnya

- **`mattpocock-skills:tdd` menjadi relevan** begitu Go terpasang. Modul berikutnya di jalur klaim
  (`B-2` Registrasi) menegakkan delapan aturan tanggal dan dua aturan duplikasi — persis bentuk
  spesifikasi yang siklus merah-hijau melayaninya dengan baik.
- **`mattpocock-skills:codebase-design`** saat `B-2` dikerjakan, karena modul itu menyembunyikan
  137 step `InputRegister_act` di balik satu antarmuka. Itu keputusan kedalaman modul yang
  sesungguhnya, berbeda dari modul ini yang bentuknya sudah ditentukan sistem lama.

---

## Tambahan sesi ini: penggantian nama ke bahasa Inggris (19 September 2026)

### Skill yang dipakai — tidak ada

Penggantian nama menyeluruh bukan pekerjaan desain maupun pemodelan domain: bentuk modulnya tidak
berubah, batas modulnya tidak berubah, dan tidak ada satu pun istilah domain yang artinya
dipertanyakan ulang. Memanggil `codebase-design` atau `domain-modeling` di sini hanya akan
menghasilkan pembenaran atas pekerjaan yang keputusannya sudah diberikan Work Owner.

Yang dipakai adalah **pembacaan `CONTEXT.md` sebagai kamus istilah** — untuk memilih padanan
Inggris yang benar, bukan terjemahan harfiah:

| Istilah domain | Padanan yang dipakai | Yang sengaja DIHINDARI |
|---|---|---|
| Objek Pertanggungan | `InsuredItem` | `Object` — bertabrakan dengan makna pemrograman |
| Settlement Line | `SettlementLine` | `Adjustment` — alias Pega yang salah arti |
| Laporan Klaim | `ClaimReport` | `ReceiveDocument` — nama kelas internal Pega |
| Tahap | `Stage` | `Status` — sudah dipakai empat konsep berbeda (`D-18`) |
| Hasil Klaim | `ClaimOutcome` | `Result` — terlalu umum untuk domain yang punya empat status |

Pilihan `ClaimReport` di atas `ReceiveDocument` bukan selera: `D-19` menetapkan alias internal Pega
tidak dibawa, dan menu portal sistem lama sendiri berbunyi **"Inbox Laporan Klaim"**.

### Yang skill TIDAK tangkap, dan tertangkap Work Owner

Pertentangan antara `D-80` dan konsistensi dengan kode lama sudah saya tulis sendiri sebagai
pertanyaan terbuka di `keputusan-implementasi.md` §12.13, lengkap dengan pemiliknya (Work Owner).
Lalu saya **menjawabnya sendiri dengan asumsi** alih-alih menanyakannya.

Ini pola kesalahan yang sama dengan yang sudah tercatat dua kali di dokumen proyek ini: bukan salah
membaca bukti, melainkan **melanjutkan tanpa menunggu jawaban atas pertanyaan yang sudah tertulis**.
Bedanya kali ini murah — koreksinya mekanis dan tidak menyentuh perilaku.

Tidak ada skill yang akan menangkapnya. Yang menangkapnya adalah Work Owner yang memeriksa hasilnya.

### Catatan untuk tahap berikutnya — tidak berubah

`mattpocock-skills:tdd` dan `codebase-design` tetap menjadi yang relevan saat `B-2` Registrasi
dikerjakan, dengan syarat yang sama: Go terpasang lebih dulu, sehingga siklus merah-hijau benar-benar
dapat dijalankan alih-alih dibayangkan.

---

# Penggunaan Skill — Sesi 2026-09-20 (modul View History Claim)

## Ringkasan

Satu modul dibangun dari nol: `MENU_ID 76` "View History Claim", pengganti harness
`PNCSearchKlaim`. Tidak ada skill Claude maupun Matt Pocock yang **dipanggil** sebagai
perintah; yang dipakai adalah tekniknya, dan itu dicatat di bawah beserta apa yang
dihasilkannya.

## Skill yang ditimbang

| Skill | Ditimbang untuk | Dipakai? |
|---|---|---|
| `mattpocock-skills:grilling` | menyusun pertanyaan sebelum menulis kode | **tekniknya dipakai**, skill tidak dipanggil |
| `mattpocock-skills:domain-modeling` | menamai ulang 16 kolom beralias menyesatkan | **tekniknya dipakai** |
| `mattpocock-skills:codebase-design` | menentukan batas modul dan letak seam | **tekniknya dipakai** |
| `mattpocock-skills:tdd` | uji lebih dulu | tidak — uji ditulis setelah bentuk domainnya jelas, sejalan dengan modul-modul sebelumnya |
| `code-review` | meninjau diff sendiri | tidak — tidak ada diff pihak lain untuk ditinjau |

## Teknik yang dipakai tanpa memanggil skill

### Grilling — enam pertanyaan sebelum satu baris kode

Kerangkanya: **cari faktanya sendiri, serahkan keputusannya**. Seluruh enam pertanyaan
disertai angka dari export dan rekomendasi beserta alasannya, bukan "bagaimana menurut
Bapak".

Yang dihasilkannya, dan tidak akan muncul tanpa itu:

| Pertanyaan | Yang terungkap |
|---|---|
| Gerbang proteksi | Layar ini **tergerbang**, dan itu tidak tercatat di dokumen migrasi mana pun |
| Cacat aturan | Tiga cacat, salah satunya membuat satu tipe pencarian tidak pernah berfungsi |
| No Rekening | Work Owner **bertanya balik** — dan penelusurannya membuktikan nomor rekening yang diketik tidak pernah sampai ke kueri |
| Penulisan gerbang | Benturan `P-1` yang tidak terlihat sampai "bangun penuh" dipilih |

**Satu kesalahan sendiri yang tertangkap oleh disiplin ini.** Pertanyaan tentang gerbang
saya ajukan dengan keterangan yang keliru — menyebut kuota *lihat data* padahal yang
berkurang kuota *pencarian*, dan menyebut log ditulis padahal tidak. Kekeliruannya
ketahuan saat memverifikasi sebelum menulis kode, dan koreksinya disampaikan lengkap
dengan tabel "saya katakan / sebenarnya". Arah keputusannya tidak berubah, tetapi isinya
berubah banyak.

### Domain modeling — menolak mewarisi nama yang salah

Layar ini kasus paling pekat dari alias menyesatkan di seluruh export: **12 dari 16** nama
properti grid tidak menyatakan isinya. `.EDMNO` berarti No Klaim, `.THEINSURED` berarti
Posisi Klaim, `.FLAGEDMBATAL` berarti Catatan Close.

Teknik yang dipakai: **silangkan pernyataan dengan kode** sebelum menamai. Setiap alias
ditelusuri ke kolom basis datanya lewat SQL-nya, lalu ke arti yang dibaca pengguna lewat
label kolom grid-nya. Barulah nama Inggrisnya dipilih dari `CONTEXT.md`.

Hasilnya dicatat di tiga tempat supaya tidak hilang: kepala `riwayatklaim.sql`,
`peta-penamaan.md`, dan komentar pada tiap isian `ClaimHistory`.

Satu temuan yang hanya muncul karena penelusuran ini: **dua properti tanggal yang namanya
tertukar dengan perannya** — properti bernama `SearchDate` adalah isian "Tanggal Lahir",
sementara isian "Tanggal Pencarian" bernama `DateOfSendInputor`. Dari situlah cacat yang
direplikasi itu lahir, dan tanpa menelusuri namanya satu per satu, cacatnya akan terbaca
sebagai kesalahan ketik biasa.

### Codebase design — seam yang dibenarkan, bukan yang mungkin

Prinsip "satu adapter berarti seam hipotetis, dua adapter berarti seam nyata" dipakai
sebagai penyaring. Tiga seam dibuat, seluruhnya punya dua pengisi nyata:

| Seam | Pengisi 1 | Pengisi 2 |
|---|---|---|
| `Repo` | SQL Oracle | memori + 5 klaim contoh |
| `ProtectionRepo` | SQL Oracle (dua tabel) | memori + 3 baris proteksi contoh |
| `Clock` | jam sistem | jam tetap di uji |

`ProtectionRepo` dipisahkan dari `Repo` bukan karena kerapian melainkan karena
**kepemilikan tabelnya berbeda**: yang satu membaca tabel sistem lama, yang lain menulis
tabel aplikasi. Pemisahan itu bentuk nyata `P-1`, dan `query_test.go` menegakkannya.

Uji penghapusan dipakai pada `Criteria.QueryValue`: bila method itu dihapus dan kuerinya
membaca isian langsung, aturan pemilihan nilai — termasuk cacat yang direplikasi — akan
tersebar ke sebelas tempat. Ia jelas membayar dirinya sendiri.

## Kesalahan sendiri yang tercatat sesi ini

| Kesalahan | Bagaimana ketahuan |
|---|---|
| Menyebut kuota "lihat data" padahal "pencarian", dan menyebut log ditulis padahal tidak | Verifikasi ke `InsertLogProteksiDataKlaimMasking` sebelum menulis kode; koreksinya dilaporkan lengkap |
| Menyebut pengguna "tidak dapat mengetik nomor rekening sama sekali" pada tipe 12 | Pembacaan ulang kondisi tampil — isiannya ADA, isinya yang dibuang |
| Memakai `serverPaging` dan `hideSearch` pada `DataTable` | Keduanya tidak ada di `master`; saya mengacu pada versi komponen di branch lain yang tidak ikut ke `master` |
| Memakai `disabled` pada `SelectOption` | Tidak ada di tipenya; diganti penandaan pada label, tanpa mengubah komponen bersama |
| Uji menekan dropdown sebelum isinya tiba | Delapan uji gagal sekaligus; helper `renderOpened()` menunggu pilihannya, bukan labelnya |

Kelimanya punya pola yang sama: **mengandalkan ingatan alih-alih memeriksa keadaan
sekarang.** Empat di antaranya lahir dari satu sebab tunggal — working tree berpindah
branch di tengah sesi, dan saya meneruskan dengan peta dari branch sebelumnya.

## Catatan untuk sesi berikutnya

1. **Periksa branch aktif sebelum mengacu pada pola modul lain.** Repo ini punya banyak
   branch paralel dengan implementasi berbeda untuk menu yang sama.
2. **Modul rincian klaim (`MENU_ID 75`)** akan memakai `Protection.MaskPhone`,
   `MaskEmail`, dan `MaskIDCard` yang sudah dibaca modul ini tetapi belum dipakai. Ketiga
   penanda itu sengaja dibawa supaya modul itu tidak menafsirkannya ulang.
3. **Kuota `LOGSEEN`** milik layar rincian, dan belum ada yang memakainya. Pola
   `Check`/`Grant` di modul ini dapat dipakai ulang apa adanya.

---

<<<<<<< HEAD
# Penggunaan Skill — Sesi 2026-09-19 (Master Status Progres 2)

## Ringkasan

| | |
|---|---|
| **Skill Matt Pocock yang dipanggil** | **tidak ada** |
| **Skill Claude Code yang dipakai** | Read · Grep · Glob · Bash · Write · Edit — perkakas baku, bukan skill terpaket |
| **Lama sesi** | satu sesi kerja |
| **Keluaran** | 2 sambungan backend · 4 berkas uji backend (40 kasus) · 4 berkas frontend (11 kasus) · 3 titik sambung · 3 dokumen |

## Kenapa tidak ada skill yang dipanggil

Alasannya sama seperti sesi-sesi sebelumnya, dan tetap berlaku: **seluruh fakta sesi ini ada di
dalam repository.** Tidak satu pun klaim di dokumen ini bersumber dari luar.

| Skill | Ditimbang, tidak dipakai — alasannya |
|---|---|
| `domain-modeling` | Istilahnya sudah ditetapkan sesi sebelumnya, dan `CONTEXT.md` tidak bertambah. Yang dikerjakan di sini adalah **memakai** bahasa itu, bukan menajamkannya |
| `grilling` | Tidak ada premis yang perlu diuji ke Work Owner. Tiga keputusan yang dibutuhkan — tanpa jalur ubah, nama induk disalin, `TIPE` baca-saja — **sudah dijawab** pada sesi sebelumnya dengan "jalani saja as-is", dan menanyakannya ulang hanya menambah beban tanpa menambah kendali |
| `codebase-design` | Batas modul dan seam-nya sudah ditetapkan sesi sebelumnya (`Repo2`, `RepoSelector2`). Sesi ini memasangnya, bukan merancangnya |
| `tdd` | Kode produksinya sudah ada lebih dulu. Ujinya ditulis **setelahnya** — dan itu dinyatakan apa adanya, bukan disamarkan sebagai TDD |

## Teknik yang dipakai tanpa memanggil skill

| Teknik | Kenapa di sesi ini |
|---|---|
| **Buktikan keadaan awal, jangan nyatakan** | Enam pemeriksaan `grep` dijalankan sebelum rencana disusun, dan hasilnya **membalik dugaan**: backend-nya ternyata sudah ada, tetapi mati. Kalau dugaan awal dipakai apa adanya, sesi ini akan menulis ulang modul yang sudah ada |
| **Telusuri rantai rule sampai ujung, bukan berhenti pada namanya** | `UpdateStatusProgress2-SQL.xml` bernama "Update" dan isinya `SELECT`. Yang benar-benar menulis ada di berkas lain, ke **tabel lain**, dengan parameter yang tidak pernah diisi. Berhenti pada namanya akan menghasilkan tombol Ubah yang tampak bekerja dan tidak mengubah apa pun |
| **Verifikasi dua arah untuk hal yang tidak dapat dilihat gagal** | Bentuk ID tanpa awalan nol dibuktikan dari **rule** dan dari **data yang beredar di kueri lain**. Salah di sini tidak menghasilkan galat apa pun — hanya ID yang tidak cocok dengan baris klaim yang sudah ada |
| **Tulis uji untuk hal yang gagalnya senyap** | `TestQueries2TargetTheChildTable` memeriksa nama tabel. Kedua tabel punya kolom `ID_PROGRESS`, sehingga tertukar tidak menghasilkan galat basis data — hanya data yang salah tempat |
| **Jalankan suite penuh sebelum dan sesudah** | 3 kegagalan lama di `master-rekening` dihitung di kedua titik. Tanpa angka sebelum, "tiga yang gagal itu bukan dari saya" hanyalah klaim |

## Kesalahan sendiri yang tercatat sesi ini

| Kesalahan | Bagaimana ketahuan | Perbaikan |
|---|---|---|
| Nama helper uji `twoPortals` bertabrakan dengan milik uji tingkat 1 di paket yang sama | kompilasi gagal | diganti `twoPortals2` |
| Nama field JSON pelanggaran ditebak `"field"` | uji panik — `interface {} is nil` | dibaca ke `dto.go`: tagnya **`kolom`**. Menebak nama kontrak adalah hal yang tidak perlu ditebak, karena berkasnya ada |
| Tipe `portalhttp.ErrorWriter` dipakai langsung | kompilasi gagal | ditiru pola uji tingkat 1, yang mendeklarasikan tipe fungsi polos dengan sengaja |
| **`gofmt -w` dijalankan pada SELURUH direktori modul**, bukan pada berkas yang saya tulis | `git status` menandai **18 berkas** berubah, termasuk berkas tingkat 1 yang tidak saya sentuh | diperiksa dengan `git diff -w`: seluruh 17 berkas lain **nol perubahan isi** — murni akhir-baris LF versus CRLF. Semuanya diseragamkan kembali ke CRLF, dan selisih akhirnya tinggal **2 berkas backend** yang memang saya sunting |

Tiga yang pertama tertangkap kompilasi atau uji dalam hitungan detik — dan itu memang gunanya.

Yang keempat berbeda sifatnya, dan itu yang membuatnya layak dicatat: **tidak ada perkakas yang
akan mengeluh.** Kode tetap terkompilasi, uji tetap hijau, dan satu-satunya yang menunjukkannya
adalah `git status` — yang sengaja diperiksa di akhir justru untuk melihat apakah lingkupnya
melebar. Tugas sesi ini dibatasi "tanpa perlu ubah di bagian yang lain", dan menandai 16 berkas
tingkat 1 sebagai berubah sudah melanggarnya meski isinya tidak bergeser satu karakter pun.

> Pelajarannya: **perintah yang menyentuh direktori lebih berbahaya daripada perintah yang
> menyentuh berkas.** `gofmt -w berkas1.go berkas2.go`, bukan `gofmt -w direktori/`.

### Kesalahan kelima — dan yang paling lama tidak ketahuan

| Kesalahan | Bagaimana ketahuan | Perbaikan |
|---|---|---|
| **Dua kolom grid keliru**: urutan tertukar, dan satu kolom (`Tipe`) ditambahkan padahal Pega tidak punya | Work Owner meminta harness dibaca **secara penuh**. Pembacaan pertama berhenti pada "kembar dengan tingkat 1, selisihnya metadata export" — benar, tetapi tata letak tidak tinggal di harness melainkan di **section** yang dirakitnya | Header dibaca langsung dari `<pyValue>&lt;b&gt;…&lt;b&gt;</pyValue>`: `No · Status Progress 1 · Status Progress 2`. Urutan diperbaiki, kolom `Tipe` dicabut, dua uji baru memaku keduanya |

Ini berbeda sifatnya dari empat yang lain, dan itu yang membuatnya layak dicatat terpisah.

**Kompilasi tidak mengeluh. Uji tidak merah.** Layarnya tampil rapi, angkanya benar, dan seluruh
uji hijau — karena ujinya saya tulis sendiri terhadap kolom yang saya rancang sendiri. Uji hanya
membuktikan layar sesuai **niat penulisnya**; ia tidak membuktikan niat itu sesuai sistem lama.

Dua hal yang membuatnya lolos:

1. **Urutannya berlawanan dengan dugaan yang wajar.** Pada layar master mana pun, nama barisnya
   sendiri biasanya mendahului rujukan induknya. Di sini kebalikannya — dan alias `City`/`CityID`
   yang menyesatkan membuat urutan itu makin sulit dibaca dari kueri saja.
2. **Kolom `Tipe` punya alasan yang terdengar masuk akal** — *"supaya nilai yang tersimpan terlihat
   petugas"*. Alasannya benar sebagai gagasan; yang salah adalah ia **alasan saya**, bukan perilaku
   sistem lama. `D-13` menetapkan tata letak mengikuti Pega.

> Pelajarannya: **"kembar secara struktur" bukan alasan berhenti membaca.** Yang menentukan tata
> letak bukan harness, melainkan section yang dirakitnya — dan header kolomnya tertulis harfiah
> di sana, tidak perlu ditebak sama sekali.

Satu hal lagi yang ikut terbukti: **permintaan Work Owner untuk memeriksa "secara penuh" bukan
formalitas.** Tanpa permintaan itu, kedua kolom keliru ini akan ikut ke produksi tanpa satu pun
perkakas yang mengeluh.

Yang tidak tertangkap perkakas mana pun adalah kesalahan pembacaan rule, dan untuk itu satu-satunya
penangkal adalah membaca rantainya sampai ujung.

## Catatan untuk sesi berikutnya

- **`SampleList2()` bukan data produksi.** Isinya susunan sendiri karena `GCNM_MST_PROGRESS` tidak
  ada di export. Ia sudah ditandai di doc comment-nya, tetapi tanda itu mudah terlewat ketika
  layarnya tampak berisi dan masuk akal. **Minta isi tabelnya ke DBA** sebelum modul ini dipakai
  untuk uji kesetaraan gerbang 1.
- **Tiga angka di modul ini masih asumsi**, seluruhnya karena `R-08`: panjang `STS_PROGRESS2` (100),
  tipe kolom `ID_MST`, dan arti `TIPE`.
- **Menambah layar baru tetap menuntut satu baris** di `frontend/src/app/menu/registry.ts`. Ejaan
  `MENU_PROGRAM`-nya dicocokkan persis, termasuk huruf besar-kecilnya — untuk modul ini
  `StatusProgress2`, yang sudah ada di `M_MENU_APLIKASI_PNC` dengan `MENU_ID` 24.

## Tambahan 2026-09-20 — paginasi, ID induk, lalu penyuntingan

Tiga umpan balik Work Owner atas layar yang sudah dinyatakan selesai. Tidak ada skill yang
dipanggil; ketiganya adalah koreksi terhadap bukti, bukan keputusan rancangan baru.

### Kesalahan keenam — mencatat "utang" untuk kemampuan yang sudah ada

| Kesalahan | Bagaimana ketahuan | Perbaikan |
|---|---|---|
| Menulis *"DataTable belum punya paginasi"* di dokumen, lalu melanjutkan | Work Owner bertanya kenapa layarnya tidak berhalaman. `DataTable` **sudah** punya prop `pageSize`, dan dua layar lain sudah memakainya | satu prop: `pageSize={15}`, angkanya dari `pyPageSizeOther` pada section Pega |

> Pelajarannya: **"dicatat sebagai utang" adalah kalimat yang harus dicurigai.** Ia terdengar
> bertanggung jawab, dan justru karena itu ia lolos tanpa diperiksa. Yang benar: sebelum
> menuliskannya, baca komponennya.

### Kesalahan ketujuh — kesalahan kolom yang ketiga pada modul yang sama

ID induk ditempelkan di sebelah nama induk (`LAPORAN AWAL 001`). Pega mengikat kolom itu ke
`.City` **saja**. Ini menyusul dua kesalahan kolom sebelumnya — urutan tertukar dan kolom `Tipe`
yang tidak ada — sehingga polanya tidak lagi dapat disebut kebetulan.

Ketiganya punya akar yang sama: **membaca kueri untuk menentukan tampilan.** Kueri memberitahu
data apa yang tersedia; ia tidak memberitahu data mana yang digambar.

> Aturan yang dipakai sejak sekarang: **satu kolom hanya boleh digambar bila ada `<rowdata>` yang
> mengikatnya ke properti baris.** Kolom yang hanya muncul di klausa `SELECT` bukan kolom layar.

### Teknik: memisahkan "apa yang terlihat" dari "apa yang terjadi"

Dipakai untuk menjawab apakah tombol Ubah perlu direplikasi. `UpdateStatusProgress2_sql`
**terlihat** menyunting, tetapi tiga bukti yang berdiri sendiri membuktikan ia tidak menyentuh
tabel master: tabelnya lain, page sumbernya hanya muncul di berkas SQL itu sendiri, dan page
penyaringnya hanya diisi layar lain.

Yang layak direplikasi adalah **hasil yang teramati**, bukan jalur yang menghasilkannya —
terutama ketika jalur itu, pada keadaan tertentu, akan menimpa catatan progres sebuah klaim.

### Keputusan yang dibalik, dan kenapa itu benar

Work Owner mula-mula memilih tidak menambahkan penyuntingan, lalu beberapa saat kemudian
memintanya. Pertanyaan pertama adalah *"apakah mereplikasi tombol Pega"* — jawabannya tidak.
Pertanyaan kedua *"apakah baris yang salah ketik dapat diperbaiki"* — jawabannya harus bisa,
karena tombol hapus pun tidak ada.

Yang dijaga saat membangunnya: ia dinyatakan sebagai **perilaku baru** di lima tempat — doc
comment seam, form, layar, `catatan-pengembangan.md` §16.13, dan `keputusan-implementasi.md`
§17.2 — sehingga selisihnya pada gerbang 1 ditemukan sebagai hal yang sudah tertulis, bukan
sebagai kejutan (`D-54`).

### Satu uji yang harapannya dibalik, dan itu bukan kelonggaran

`TestNoEditRoute2Exists` memastikan `PUT` dijawab 404. Setelah rutenya ada, chi menjawab **405** —
dan 405 lebih benar: jalurnya ada, metodenyalah yang tidak dilayani. Ujinya diganti menjadi
`TestOnlyPutIsRegistered2` yang menjaga hal yang masih berlaku: **`DELETE` tetap tidak dilayani.**

---

# Penggunaan Skill — Sesi 2026-09-19 (Master Penolakan Klaim)

## Ringkasan

| | |
|---|---|
| Pekerjaan | Modul Master Penolakan Klaim — dua master pada satu layar (MENU_ID 25) |
| Skill yang **dipanggil** | **tidak ada** |
| Teknik skill yang dipakai **tanpa memanggilnya** | `grilling` · `domain-modeling` · `codebase-design` |
| Perkakas lain | `AskUserQuestion` untuk empat pertanyaan konfirmasi |

## Kenapa tidak ada skill yang dipanggil

Tiga skill Matt Pocock terpasang dan relevan pada paruh kedua sesi — `tdd`, `code-review`,
`codebase-design`. Tidak satu pun dipanggil, dan alasannya sama untuk ketiganya: **pekerjaannya
sudah punya cetakan yang lebih ketat daripada yang skill itu tawarkan.**

| Skill | Kenapa tidak dipanggil |
|---|---|
| `codebase-design` | Batas modul sudah ditetapkan `README.md` §"Aturan susunan yang mengikat" dan dicontohkan tiga modul master sebelumnya. Yang dibutuhkan bukan merancang batas baru melainkan **mengikuti batas yang sudah ada** — dan satu-satunya pertanyaan rancangan yang benar-benar baru (satu seam untuk dua tabel, atau dua) dijawab oleh **bukti**: procedure lama memanggil keduanya dalam satu langkah |
| `tdd` | Urutannya memang uji-lebih-dulu untuk perbaikan duplikasi induk — uji `TestChoosingExistingParentDoesNotCreateDuplicateParent` ditulis sebagai pernyataan aturan sebelum adapter memori selesai. Tetapi itu disiplin yang sudah tertulis di `14-TESTING-STRATEGY.md` §3.2, bukan sesuatu yang perlu dimuat ulang |
| `code-review` | Sesi ini menulis modul baru, bukan meninjau perubahan orang lain. Yang ditinjau — `masterautoclaim` milik sesi lain — justru **tidak boleh** disentuh (Isolasi Protektif) |

## Teknik yang dipakai tanpa memanggil skill

### `grilling` — menekan premis sampai patah, sebelum menulis kode

Dipakai pada empat premis. **Tiga di antaranya patah**, dan ketiganya akan menghasilkan kode yang
salah bila diterima apa adanya:

| Premis awal | Ditekan dengan | Hasil |
|---|---|---|
| "Harness ini adalah layarnya" | membaca `pyUsage` dan `pyMemo`-nya | **patah** — ia cangkang; isinya dua lapis lebih dalam di `BrowseNoteRejectClaim` |
| "Grid menyaring `STATUS='0'`, seperti tertulis di activity-nya" | membaca `pyStepsPreCondition` langkah itu | **patah** — berprasyarat `Param.master=="1"`, yang layar ini tidak kirim |
| "Layar ini mengelola satu master" | menghitung kemunculan nama page dan tombol | **patah** — dua master, dua tombol, satu `FlagASO` |
| "Persetujuan diisi di layar ini" | mencari siapa yang memakai `Sec_PenolakanKlaimChecker` | **patah** — milik `UserInbox_Harness`, MENU_ID 58 |

Yang membuat teknik ini bekerja di sini bukan kecurigaan umum melainkan satu kebiasaan: **setiap
kali sebuah activity tampak melakukan sesuatu, precondition-nya dibaca lebih dulu.** Premis kedua
hanya patah karena itu.

### `domain-modeling` — menolak alias yang menyesatkan

`BrowseStatusPenolakanKlaim2-SQL.xml` mengaliaskan kedelapan kolomnya ke nama yang tidak
mencerminkan isi sama sekali — bentuk paling parah dari utang teknis §4.2 yang sudah dikenal:

```
NOTE_ST      AS "City"            nama INDUK, bukan nama kota
NOTE_ND      AS "CityID"          nama BARIS INI, bukan kode kota
ID_ND        AS "District"        kunci baris, bukan kabupaten
USER_INPUT   AS "UserTeknis"      pengaju, bukan PIC Teknik
NOTEAPPROVED AS "NoteKasir"       tidak ada urusan dengan kasir
<derivasi>   AS "AnaylstRemarks"  bukan catatan analis — dan salah eja
```

`City`/`CityID` **tidak berpasangan**, persis seperti pada Master Status Progres 2. Aturan yang
dipegang: **yang dipetakan adalah KOLOMNYA, bukan aliasnya.** Aliasnya dicatat di doc comment
sebagai peringatan, tidak pernah dipakai sebagai petunjuk arti.

### `codebase-design` — seam dibenarkan bukti, bukan simetri

Satu pertanyaan rancangan yang benar-benar baru: dua tabel Penolakan Klaim dilayani satu seam atau
dua? Simetri dengan Master Status Progres menyarankan **dua** (di sana tingkat 1 dan 2 punya seam
masing-masing). Buktinya menyarankan **satu**:

- `Activity/InsertMasterPenolakanNoteKlaim` memanggil `MasterPenolakanKlaim1` lalu langsung
  memakai ID hasilnya untuk `MasterPenolakanKlaim2` — satu langkah, bukan dua;
- induk baru **harus lahir bersama anaknya**, kalau tidak ia baris menggantung — persis cacat yang
  sedang diperbaiki;
- memecahnya menuntut `*sql.Tx` bocor ke luar repo agar keduanya sebungkus transaksi.

Bukti menang. Yang berbeda dari Master Status Progres bukan kelalaian: di sana kedua tingkat punya
**layar sendiri-sendiri**, di sini keduanya satu layar.

## Kesalahan sendiri yang tercatat sesi ini

Tiga, seluruhnya tertangkap sebelum masuk ke kode yang dijalankan:

| # | Kesalahan | Yang menangkapnya |
|---|---|---|
| 1 | Menulis penyaring `WHERE STATUS='0'` ke dalam kueri daftar | Membaca `pyStepsPreCondition` — sesuatu yang belum saya lakukan saat pertama membaca activity-nya |
| 2 | Asersi uji "kueri tidak boleh memuat APPROVED" | Kolom `NOTEAPPROVED` dan `TANGGAL_APPROVE` memang memuat kata itu. Uji gagal, asersinya dipersempit ke literal bertanda kutip |
| 3 | Menyimpulkan port 8080 dipegang instans aplikasi lain | Respons `HTTP 403` dengan header `Server: squid/5.5` — yang memegangnya proxy korporat |

Kesalahan 3 layak dicatat karena kesimpulannya **masuk akal dan salah**: "port terpakai" saat
mengembangkan aplikasi web hampir selalu berarti instans lain. Yang mengoreksinya bukan penalaran
ulang melainkan **membaca respons mentahnya**.

## Satu hal yang tidak saya kerjakan, dan itu disengaja

`internal/masterautoclaim/` muncul di working tree selama sesi berjalan dan **tidak dapat
dikompilasi** (`undefined: LookupRepo`). Ia bukan buatan sesi ini. Memperbaikinya akan melanggar
Isolasi Protektif dan menabrak sesi yang sedang menulisnya — `cmd/claimpnc/check.go` bahkan
disunting dari luar di tengah sesi.

Yang dikerjakan: menjalankan verifikasi dengan **mengecualikan** paket itu, lalu melaporkannya.
Membiarkan `go build ./...` gagal tanpa menyebutkannya akan menyerahkan kejutannya ke orang
berikutnya.

## Catatan untuk sesi berikutnya

Modul berikutnya yang paling erat hubungannya adalah **Inbox Manager** (MENU_ID 58,
`UserInbox_Harness`) — di sanalah persetujuan Penolakan Klaim benar-benar dikerjakan, lewat
`Sec_PenolakanKlaimChecker` dan `RDB List/UpdateStatusPenolakanKlaim2-SQL.xml`. Seluruh bahannya
sudah terbaca sesi ini:

- kolom yang ditulisnya: `STATUS`, `APPROVEBY`, `TANGGAL_APPROVE`, `NOTEAPPROVED`;
- penyaringnya: `WHERE STATUS='0'` — di sanalah `Param.master=="1"` benar-benar dikirim;
- pencacahnya: `RDB List/CountCheckerRejectNotes-SQL.xml`.

Satu hal yang perlu diputuskan sebelum modul itu dibangun: **siapa yang boleh menyetujui.**
`D-59` menetapkan satuan izin adalah menu tanpa pemisahan tugas formal, sehingga orang yang
mengajukan dapat pula menyetujui bila ia punya kedua menu itu.

---

# Sesi kesebelas — Master Auto Claim (2026-09-19)

## Ringkasan

| | |
|---|---|
| Permintaan | tambahkan modul **Master Auto Claim**, dengan `Harness/AutoKlaim-Harness.xml` sebagai acuan |
| Skill Matt Pocock yang dipanggil | **tidak ada** |
| Skill Claude Code yang dipanggil | **tidak ada** |
| Perkakas yang dipakai | `AskUserQuestion` · dua skrip pembaca XML Pega yang dibuat sesi ini |

## Kenapa tidak ada skill yang dipanggil

Alasannya sama dengan beberapa sesi terakhir, dan tetap berlaku:

| Skill | Kenapa tidak |
|---|---|
| `grilling` | Pertanyaannya sudah **terjawab oleh rule-nya sendiri**. Yang tersisa adalah empat keputusan bisnis, dan itu diajukan langsung ke Work Owner lewat `AskUserQuestion` — bukan digali lewat ronde pertanyaan |
| `domain-modeling` | Istilah barunya nol. `Sumber Bisnis`, `Client`, `Bank`, dan `Komite` semuanya sudah ada di `CONTEXT.md` atau di modul yang sudah dibangun |
| `codebase-design` | Batas modulnya sudah ditentukan lima modul sebelumnya. Yang dikerjakan adalah **mengikuti** pola itu, bukan merancang ulang |
| `tdd` · `code-review` · `diagnosing-bugs` | Tidak diminta, dan pola ujinya sudah mapan di repo ini |

## Perkakas yang dibuat sendiri, dan kenapa `grep` tidak memadai

Harness-nya 291 KiB dan seluruh isinya satu pohon XML dengan `rowdata` bersarang. Tiga kali
pembacaan dengan regex menghasilkan jawaban yang **salah**, dan ketiganya baru ketahuan setelah
dibandingkan dengan pembacaan yang benar:

| Percobaan | Kenapa gagal |
|---|---|
| `grep -oE` atas nama atribut | Pega menyimpan nilainya sebagai **elemen**, bukan atribut — nol keluaran |
| Regex atas `<rowdata>` | `rowdata` bersarang sampai empat tingkat; langkah aktivitas dan parameter UI tercampur jadi satu |
| Membaca `pyStepsPreCondition` berurutan | urutan elemen di dalam satu `rowdata` **tidak dijamin**, sehingga prasyarat menempel ke langkah yang salah |

Yang dipakai akhirnya: dua skrip `xml.etree.ElementTree` di scratchpad —
satu untuk langkah aktivitas beserta prasyarat dan aksinya, satu untuk langkah **bersarang**
(badan perulangan `pxResults`). Keduanya menelusuri pohon, bukan teks, sehingga sarangnya tidak
pernah tercampur.

Satu hal yang hanya terbaca lewat pembacaan pohon: **kode aksi prasyarat**. Nilai `2`/`3`/`6`
pada `pyStepsPreCondParamsWhenTrue`/`WhenFalse` berarti *continue* / *skip step* / *exit
activity* — dan tanpa memetakannya, langkah 5 pada `InsertMstAutoClaim_act` terbaca **terbalik**:
prasyaratnya `Param.err==""`, yang sekilas berarti "keluar bila tidak ada galat". Yang benar
kebalikannya, dan itu hanya terlihat dari `WhenFalse = 6`.

## Teknik yang dipakai tanpa memanggil skill

**Membaca pemakaian HILIR untuk mengetahui arti sebuah tabel.** Nama modul dan nama kolom
keduanya menyesatkan. Yang menjawab "apa sebenarnya ini" adalah satu kueri di modul **lain** —
`GetReceiverClaimAsuransiKredit-SQL.xml` — yang mencocokkan `INISIALID` dengan
`T_GENERAL.SOURCEOFBUSINESS`. Tanpa itu, modul ini akan dibangun sebagai "master klaim otomatis"
dan seluruh penamaannya salah.

**Menolak alias sebagai petunjuk arti.** Sepuluh alias tidak mencerminkan isi, dan satu di
antaranya — `City` — berarti **kolom yang berbeda** saat dibaca dan saat ditulis. Yang dipetakan
adalah kolomnya.

**Menanyakan yang menyentuh uang, memutuskan sisanya.** Empat pertanyaan diajukan; belasan
keputusan lain diambil sendiri dan dicatat. Pemisahnya satu: apakah jawabannya mengubah **nilai
yang tersimpan** atau **siapa yang berwenang**.

**Membuktikan kegagalan uji itu pre-existing, bukan mengklaimnya.** Tiga uji `AccountPage`
gagal. Alih-alih menyebutnya "tidak berhubungan", keempat berkas sesi ini di-`git stash`, uji
dijalankan ulang — tetap gagal — lalu `git stash pop`.

## Kesalahan sendiri yang tercatat sesi ini

| # | Kesalahan | Bagaimana ketahuan |
|---|---|---|
| 1 | Menulis `parsePercent` dengan `fmt.Sscanf` dan tiga lapis perbandingan teks untuk menambal kelemahannya | Dibaca ulang sebelum uji ditulis: Sscanf berhenti pada karakter tak cocok dan **tetap melapor sukses**. `strconv.ParseFloat` menyelesaikannya dalam satu baris |
| 2 | Menerima `BankCode` dari layar untuk memeriksa "bank tidak diketik manual" | Kode itu **tidak pernah disimpan** — tabelnya tidak punya kolomnya. Akibatnya tombol Approve akan bergantung pada nilai yang tidak dapat dibaca kembali dari baris tersimpan |
| 3 | Mengisi `ReceiverName: "-"` di handler supaya `Check` lolos pada jalur simpan | Akal-akalan yang menyembunyikan aturan sebenarnya. Diganti `CheckEditable` yang menyatakan aturannya apa adanya |
| 4 | Menulis komentar Go berisi `'.',''` | `gofmt` mengubah dua petik tunggal berurutan menjadi tanda kutip tipografis, sehingga cuplikan SQL-nya tidak lagi sama dengan kuerinya |
| 5 | Menyebut `ESCAPE '\'` di komentar Go tetapi **tidak menuliskannya di kueri** | Tertangkap saat membaca ulang berkas `.sql`. Oracle tidak punya karakter pelolos bawaan pada `LIKE`, jadi komentarnya benar dan kuerinya yang kurang |

Kelima-limanya tertangkap **sebelum** uji ditulis atau sebelum commit — tidak satu pun lolos ke
hasil akhir. Yang menangkapnya bukan perkakas melainkan membaca ulang apa yang baru saja ditulis.

## Catatan untuk sesi berikutnya

**`gofmt -l` tidak berguna di repo ini.** Seluruh berkas memakai CRLF sementara gofmt menuntut
LF, sehingga ia menandai **setiap** berkas. Cara memisahkan temuan nyata dari derau akhiran
baris: bandingkan sisi `+` dan `-` dari `gofmt -d` setelah keduanya dinormalkan. Dua temuan nyata
pada sesi ini terungkap begitu — perataan blok `const` dan kutipan tipografis di atas.

**Ada sesi lain yang menulis di repo ini bersamaan.** Modul `masterpenolakan` bertambah di
working tree **selama** sesi ini berjalan, termasuk empat berkas dokumentasi. Akibatnya nomor bab
bergeser: bab yang semula §17/§18 menjadi §18/§19. Sebelum menyunting dokumen, periksa ulang
nomor bab terakhirnya — jangan mengandalkan yang terbaca di awal sesi.

---

# Sesi kedua belas — Master Pasal Kerugian (2026-09-19)

## Ringkasan

| | |
|---|---|
| **Skill yang dipanggil** | **tidak ada** |
| **Skill yang ditimbang** | `mattpocock-skills:grilling`, `codebase-design`, `domain-modeling`, `tdd` |
| **Alat bantu yang dibuat sendiri** | pembaca XML Pega berbasis `ElementTree` untuk membongkar section dan activity |
| **Keputusan Work Owner yang dicatat** | 4 — seluruhnya "coba jalankan secara as is" |

## Kenapa tidak ada skill yang dipanggil

Sama seperti sebelas sesi sebelumnya, dan alasannya tidak berubah: pekerjaan sesi ini adalah
**membaca export Pega lalu menurunkannya menjadi modul**, dan tidak satu pun skill yang tersedia
mengerjakan salah satu dari keduanya. Yang menentukan hasil di sini adalah ketelitian membaca XML,
bukan teknik yang dapat dipinjam.

Dua yang paling dekat, dan kenapa keduanya tetap tidak dipanggil:

| Skill | Kenapa ditimbang | Kenapa tidak dipanggil |
|---|---|---|
| `grilling` | Ada empat hal yang memang perlu ditanyakan ke Work Owner | Pertanyaannya sudah terbentuk sendiri dari bukti — tiga di antaranya berupa pertentangan langsung dengan `D-66`, `R-16`, dan pola modul sebelumnya. Yang dibutuhkan hanyalah menyajikan pilihannya beserta akibat masing-masing, dan itu satu pemanggilan `AskUserQuestion` |
| `domain-modeling` | Penamaan ulang `D-19` adalah inti modul ini | `CONTEXT.md` sudah memuat kosakata yang dipakai; yang dikerjakan adalah memetakan properti Pega ke istilah yang sudah ada, bukan menyusun istilah baru |

`tdd` tidak dipanggil karena urutannya memang tidak dapat dibalik di sini: bentuk dokumen JSON
harus dibaca dari export lebih dulu, dan uji yang ditulis sebelum bentuknya diketahui hanya akan
menguji tebakan.

## Teknik yang dipakai tanpa memanggil skill

**Membaca XML Pega dengan parser, bukan dengan `grep`.** Section `BrowsePasalDeatailMaster` besarnya
354 KiB dan seluruh isinya satu baris panjang; `grep` mengembalikan potongan yang tidak dapat
ditelusuri ke elemen induknya. Yang dipakai adalah skrip Python `ElementTree` sekali pakai yang
menelusuri `Embed-Display-Table-Cell` **dalam urutan dokumen**, lalu mencetak `pyValue`,
`pyLabelFieldValue`, dan `pyFormat` tiap sel. Itu yang membuat pasangan label ↔ properti terbaca
persis — termasuk yang terbalik dari dugaan wajar (`ISI PASAL` ↔ `.DESCRIPTION`).

**Menolak menebak apa yang dapat dibaca.** Bentuk dokumen `JSONPASAL` mula-mula tampak harus
ditebak. Ia tidak: `Function/GetPageJSONString-Function.xml` memuat `stepPage.getJSON(false)`, dan
satu baris itu menentukan bahwa kuncinya adalah nama properti page apa adanya — yang kemudian
dikonfirmasi silang oleh keempat `json_value` pada kueri lama.

**Membuktikan kegagalan yang pre-existing, bukan mengasumsikannya.** Sesi ini menyentuh berkas
bersama `api/client.ts`, sehingga "tiga uji `AccountPage` memang sudah gagal" tidak cukup
diucapkan. Ia dibuktikan: `git stash` atas kedua berkas bersama, jalankan ulang, ketiganya tetap
gagal, lalu `git stash pop`.

**Menyatakan selisih di muka.** Tiga perbedaan terhadap Pega ditulis sebagai daftar tersendiri di
README, di `keputusan-implementasi.md`, dan di doc comment — bukan ditemukan belakangan sebagai
kejutan pada uji kesetaraan gerbang 1.

## Kesalahan sendiri yang tercatat sesi ini

| # | Kesalahan | Bagaimana ketahuan | Pelajaran |
|---|---|---|---|
| 1 | `ls -R internal \| head -200` dipakai untuk memetakan modul yang ada; hasilnya terpotong dan saya menyimpulkan `masterbengkel` tidak ada | `go build` berhasil atas kode yang mengimpornya | Jangan memakai keluaran yang **sengaja dipotong** sebagai dasar kesimpulan tentang kelengkapan |
| 2 | `fakeRow.Scan` menulis `return assign(...)` di dalam perulangan, sehingga hanya kolom pertama terisi | empat uji gagal sekaligus dengan nilai kosong | `return` di dalam perulangan yang seharusnya mengisi seluruh elemen — selalu periksa ulang bahwa loop benar-benar berjalan sampai habis |
| 3 | Uji menuntut **nol** permintaan jaringan sebelum portal dipilih | uji gagal: daftar Kategori memang sengaja tidak bergantung portal | Ekspektasi uji ikut salah bila ditulis sebelum perilakunya dipikirkan tuntas. Yang diperbaiki ekspektasinya, bukan kodenya |
| 4 | Heredoc `bash` dipakai menambahkan bagian dokumen sepanjang ratusan baris; gagal terurai | pesan `unexpected EOF while looking for matching quote` | Untuk isi panjang, tulis ke berkas lebih dulu lalu `cat >>` — jangan menaruhnya di dalam perintah shell |

Keempatnya ditemukan oleh perkakas — build, uji, dan shell — bukan oleh pembacaan ulang. Itu pola
yang sama dengan sesi-sesi sebelumnya, dan alasan mengapa uji ditulis untuk hal yang **tidak
terlihat di layar**.

## Catatan untuk sesi berikutnya

**Modul Master Bengkel muncul di working tree SELAMA sesi ini berjalan**, dikerjakan orang lain —
`internal/masterbengkel/` sudah ada sejak awal sesi, dan `frontend/src/modules/master-bengkel/`
menyusul di tengah sesi. Keduanya tidak disentuh (Isolasi Protektif), dan suntingan pada
`App.tsx` serta `registry.ts` dilakukan dengan `Edit` atas potongan spesifik sehingga pekerjaan
keduanya hidup berdampingan tanpa saling menimpa.

Akibatnya nomor bab bergeser lagi: bab terakhir `catatan-pengembangan.md` menjadi §19 dan
`keputusan-implementasi.md` menjadi §20. **Periksa ulang nomor bab terakhir sebelum menyunting** —
jangan mengandalkan yang terbaca di awal sesi.

**Satu hal yang menunggu basis data sungguhan:** pengikatan `JSONPASAL` yang lebih panjang dari
4000 karakter. Itu satu-satunya bagian modul ini yang tidak dapat dibuktikan di lingkungan
pengembangan, dan ia ditandai di banner `masterpasal.sql`.

---

# Sesi ketiga belas — Master Bengkel (2026-09-19)

## Ringkasan

| | |
|---|---|
| **Skill yang dipanggil** | **tidak ada** |
| **Skill yang ditimbang** | `mattpocock-skills:grilling`, `codebase-design`, `domain-modeling`, `tdd`, `code-review` |
| **Alat bantu yang dibuat sendiri** | dua pembaca XML Pega sekali pakai: pengekstrak caption dari indeks `pzIndexes`, dan pemindai prasyarat langkah activity |
| **Keputusan Work Owner yang dicatat** | 4 — seluruhnya "coba jalankan secara as is" |

## Kenapa tidak ada skill yang dipanggil

Sama seperti dua belas sesi sebelumnya, dan alasannya tidak berubah: pekerjaan sesi ini adalah
**membaca export Pega lalu menurunkannya menjadi modul**, dan tidak satu pun skill yang tersedia
mengerjakan salah satu dari keduanya. Yang menentukan hasil di sini adalah ketelitian membaca XML,
bukan teknik yang dapat dipinjam.

Lima yang ditimbang, dan kenapa kelimanya tetap tidak dipanggil:

| Skill | Kenapa ditimbang | Kenapa tidak dipanggil |
|---|---|---|
| `grilling` | Empat hal memang perlu ditanyakan ke Work Owner, dan salah satunya (jalur simpan) menentukan seluruh bentuk modul | Pertanyaannya sudah terbentuk sendiri dari bukti: dua tabel untuk satu master, badan fungsi JSON yang hilang, operator ber-kata-sandi tetap, dan rule persetujuan yang tidak ada di export. Yang dibutuhkan hanyalah menyajikan pilihannya beserta akibat masing-masing — satu pemanggilan `AskUserQuestion` |
| `codebase-design` | Modul ini punya **tiga** seam, satu lebih banyak dari modul master mana pun sebelumnya (`Repo`, `LookupRepo`, `IDSource`) | Kosakatanya sudah terpakai di `04-FUTURE-ARCHITECTURE.md`, dan pola tiga-seam-satu-`Store` sudah ada preseden langsungnya di `masterautoclaim`. Yang dikerjakan adalah mengikuti pola yang sudah disetujui, bukan merancang batas baru |
| `domain-modeling` | Empat puluh kolom, dan **tiga** di antaranya berarti ganda | `CONTEXT.md` sudah memuat kosakata yang dipakai. Yang dikerjakan adalah memetakan kolom Pega ke istilah yang mencerminkan isinya, dan kriterianya sudah ditetapkan `D-19` — bukan menyusun istilah baru |
| `tdd` | Aturan validasinya banyak dan berlapis (wajib bersyarat, persentase, keunikan) | Urutannya tidak dapat dibalik: aturan mana yang berlaku baru diketahui setelah prasyarat langkah activity dibaca. Uji yang ditulis sebelum itu hanya akan menguji tebakan. Ujinya tetap ditulis berdampingan dengan kode, 54 di backend dan 16 di frontend |
| `code-review` | Modul ini menyentuh `main.go` yang sedang disunting sesi lain | Yang dibutuhkan bukan tinjauan terhadap standar melainkan memastikan tidak ada pekerjaan pihak lain yang tertimpa, dan itu dijawab `git status` — bukan skill |

## Teknik yang dipakai tanpa memanggil skill

**Menemukan bahwa harness-nya cangkang, bukan layar.** `Harness/BengkelHE-Harness.xml` besarnya
282 KiB, dan pencarian pertama untuk `NAMA_BENGKEL` di dalamnya menghasilkan **nol**. Yang
dilakukan berikutnya bukan menyerah pada harness, melainkan memindai seluruh identifier
ber-substring "bengkel" di dalamnya — yang mengungkap satu-satunya kaitan nyata: section
`MasterBengkelHE`. Dari sana empat lapis ditelusuri sampai ketemu ketiga tab dan formnya.

**Membaca caption dari indeks rule, bukan dari layout.** Empat percobaan pertama mengekstrak label
form gagal: format export menyimpan referensi rule pada `pzIndexes`, bukan sebagai atribut layout,
sehingga tidak ada `pyCaption` yang dapat dibaca berpasangan dengan propertinya. Yang akhirnya
berhasil adalah membaca `pxRuleFamilyName` berawalan `PYCAPTION!` beserta `pyRuleName` bertipe
`Rule-Obj-Property` dari blok `rowdata` yang sama — dan dari sana ke-33 caption terbaca utuh.

**Membuktikan ketiadaan, bukan mengasumsikannya.** Nilai sah tujuh penanda status tidak diketahui.
Alih-alih menulis "tidak ada di export" berdasarkan tidak-menemukannya, dijalankan pemindaian
menyeluruh atas `Activity/`, `When/`, `RDB List/`, dan seluruh section bengkel untuk **setiap
bentuk perbandingan** terhadap ketujuh kolom itu. Hasilnya nol, dan barulah kalimat "nol bukti"
layak ditulis. Pemindaian yang sama justru menemukan satu-satunya nilai yang **memang** dapat
dibaca — `Local.STS_REKANAN=='0'` di dalam `pyStepsPreCondParamsWhen`, tempat yang tidak terbaca
oleh pencarian atas `pyStepsPreCondition`.

**Menolak menebak apa yang tidak dapat dibaca.** Berbeda dari sesi sebelumnya yang berhasil
membaca `stepPage.getJSON(false)` di dalam `GetPageJSONString-Function.xml`, kali ini berkas yang
**sama** hanya memuat tanda tangannya — badan fungsinya tidak ikut. Itulah yang menentukan bahwa
jalur simpan JSON tidak dapat dikerjakan, dan keputusannya berdiri di atas ketiadaan yang
terbukti, bukan di atas preferensi.

**Memperlakukan kegagalan uji sebagai temuan, bukan sebagai gangguan.** Enam uji frontend gagal
dengan *"Found multiple elements with the role button and name Approve"*. Perbaikan yang paling
cepat adalah menyempitkan pencarian di uji. Yang dikerjakan adalah membaca kenapa ambiguitasnya
ada — dan ia ada **di layar**, bukan di uji: dua kontrol berbeda diberi nama yang sama persis.
Yang diperbaiki karena itu adalah UI-nya.

## Kesalahan sendiri yang tercatat sesi ini

| # | Kesalahan | Bagaimana tertangkap |
|---|---|---|
| 1 | Mengira `Section/MasterBengkelHE` memuat form, karena ia satu-satunya section yang dirujuk harness | Pemindaian propertinya menghasilkan nol kolom bengkel; formnya ternyata dua lapis lebih dalam |
| 2 | Mengira `Section/ApprovalMasterBengkelHE` adalah tab keempat layar ini | Pencarian pemakainya menunjukkan ia dipakai `InboxManager_Sec` — layar yang sama sekali berbeda |
| 3 | Menulis `TestInsertColumnCountMatchesArguments` memanggil `sampleWorkshop()` yang belum ada | Kompilasi uji gagal pada jalannya yang pertama |
| 4 | Menamai tombol keputusan "Approve"/"Reject", sama persis dengan caption tab | Enam uji frontend gagal sekaligus; diperbaiki di UI, bukan di uji |
| 5 | Meninggalkan parameter `call` yang tidak terpakai pada satu pemanggilan `defaultReply` | `tsc --noEmit` menandainya `TS6133` |

Kelimanya tertangkap **sebelum** pekerjaan dinyatakan selesai. Yang menangkap tiga di antaranya
adalah perkakas (kompilator, `tsc`, uji); dua sisanya tertangkap karena hasil pembacaan diperiksa
ulang terhadap export alih-alih dipercaya.

## Catatan untuk sesi berikutnya

**Format section Pega punya dua tempat penyimpanan yang berbeda.** Modul-modul sebelumnya
menemukan pasangan label ↔ properti di dalam `Embed-Display-Table-Cell`; section bengkel
menyimpannya di indeks `pzIndexes` sebagai `PYCAPTION!<LABEL>` berdampingan dengan
`Rule-Obj-Property`. Coba keduanya sebelum menyimpulkan sebuah section tidak memuat apa-apa.

**Keluarga procedure `PEGA_M_*` berjumlah lima belas, dan seluruhnya berpola sama:** kode situs
dari `M_SITE_DATABASE` + sequence, lalu `INSERT INTO <tabel>(ID, JSONDATA)`. Dua di antaranya
sudah dikerjakan (`M_STS_CLAIM`, `M_BENGKEL_HE`) dan keduanya diputuskan sama. Modul berikutnya
dari keluarga itu — panel, sparepart, supplier, surveyor, cause of loss — dapat mengikuti pola
yang sama tanpa mengulang analisisnya, **asalkan** pertanyaan "tabel atau view" tetap ditanyakan
ke DBA untuk masing-masing.

**Ada sesi lain yang menulis di repo ini bersamaan.** Modul `masterpasal` bertambah di working
tree **selama** sesi ini berjalan, dan `cmd/claimpnc/main.go` sempat tidak dapat dikompilasi
karena perakitannya setengah jadi. Jangan memperbaiki atau mengembalikan pekerjaan pihak lain —
tunggu, lalu bangun ulang. Nomor bab dokumentasi juga bergeser: periksa ulang nomor terakhirnya
sebelum menyunting.

---

# Sesi keempat belas — Master Panel (2026-09-20)

## Ringkasan

| | |
|---|---|
| **Skill yang dipanggil** | **tidak ada** |
| **Skill yang ditimbang** | `mattpocock-skills:grilling`, `codebase-design`, `domain-modeling`, `tdd`, `code-review` |
| **Alat bantu yang dibuat sendiri** | tiga pembaca XML Pega sekali pakai: pengekstrak elemen ber-SQL dari rule RDB, pembaca `pyRequired` + `pyControlDisplayTitle` berpasangan dengan propertinya, dan pemindai `PropertiesName`/`PropertiesValue` pada activity |
| **Keputusan Work Owner yang dicatat** | 2 — keduanya "Jalankan sebagai as is" |

## Kenapa tidak ada skill yang dipanggil

Sama seperti tiga belas sesi sebelumnya, dan alasannya tidak berubah: pekerjaan sesi ini adalah
**membaca export Pega lalu menurunkannya menjadi modul**, dan tidak satu pun skill yang tersedia
mengerjakan salah satu dari keduanya.

Lima yang ditimbang, dan kenapa kelimanya tetap tidak dipanggil:

| Skill | Kenapa ditimbang | Kenapa tidak dipanggil |
|---|---|---|
| `grilling` | Dua hal memang perlu ditanyakan, dan yang pertama (nasib sub-tabel) menentukan seluruh bentuk modul — termasuk apakah repo-nya perlu transaksi sama sekali | Pertanyaannya sudah terbentuk sendiri dari bukti: tabel anak yang kolom keempatnya hanya muncul sebagai penyaring, dan sembilan dropdown yang daftar pilihannya tidak ikut di export. Yang dibutuhkan hanyalah menyajikan pilihannya beserta akibat masing-masing — satu pemanggilan `AskUserQuestion` |
| `codebase-design` | Modul ini **aggregate pertama** di aplikasi ini: satu induk dengan koleksi anak, disimpan dan diganti sebagai satu kesatuan | Batasnya sudah ditentukan bentuk datanya, bukan oleh pilihan rancangan: baris anak tidak punya kunci sendiri, sehingga ia **tidak dapat** menjadi entitas berdiri sendiri. Yang tersisa adalah menaruh keduanya di balik satu `Repo` — keputusan yang tidak punya alternatif nyata |
| `domain-modeling` | Satu nama (`STS_SISI`) memikul **dua** arti yang sama sekali berbeda, dan salah satunya alias | `CONTEXT.md` dan `D-19` sudah menetapkan kriterianya: pakai nama yang mencerminkan isi. Yang dikerjakan adalah menerapkannya, bukan menyusun kosakata baru |
| `tdd` | Aturan validasi baris anak berlapis: wajib, sandi sah, dan kembar — ketiganya pada daftar yang panjangnya berubah-ubah | Urutannya tidak dapat dibalik: sandi sisi mana yang sah baru diketahui setelah `SetLokasiSisiPanel-Act` dan `GetSisiPanel-Act` dibaca dan **disilangkan**. Uji yang ditulis sebelum itu hanya menguji tebakan. Ujinya tetap ditulis berdampingan dengan kode — 63 di backend, 16 di frontend |
| `code-review` | `main.go`, `App.tsx`, `types.ts`, dan `registry.ts` **sedang disunting sesi lain** saat sesi ini berjalan | Yang dibutuhkan bukan tinjauan terhadap standar melainkan memastikan tidak ada pekerjaan pihak lain yang tertimpa, dan itu dijawab `git status` dan `go build` — bukan skill |

## Teknik yang dipakai tanpa memanggil skill

**Menyilangkan dua rule untuk memastikan satu daftar nilai.** Sandi sisi (`-`, `1`, `2`) tidak
disimpulkan dari satu tempat. `SetLokasiSisiPanel-Act` menyusunnya sebagai daftar pilihan;
`GetSisiPanel-Act` menerjemahkannya kembali menjadi label. Keduanya sepakat, dan kesepakatan itulah
yang membuat daftarnya layak dipakai sebagai **aturan validasi** — berbeda dari kesembilan penanda
`STS_*`, yang tidak punya satu pun sumber sehingga hanya boleh diperlakukan sebagai teks.

**Menelusuri pemanggil untuk mengetahui arti sebuah kolom.** `NAMA` pada tabel anak tidak pernah
dibaca di mana pun — ia hanya muncul sebagai penyaring pada satu kueri. Yang dilakukan bukan
menebak artinya, melainkan mencari **siapa yang memanggil kueri itu**: lima section modul Grouping
Sparepart HE, yang mengirimkan sebuah nama lokasi. Dari situ keputusan penulisannya punya dasar,
dan — yang lebih penting — punya **cara dibantah** lewat `claimpnc -periksa`.

**Memastikan `pyRequired` satu per satu, bukan menyimpulkan dari satu contoh.** Dua isian pertama
yang diperiksa bertanda `pyRequired=true`, dan godaannya adalah menyimpulkan seluruhnya begitu.
Kesepuluhnya diperiksa dengan pemindai yang membaca `pyRequired` berpasangan dengan
`pyLabelFieldValue` — dan kesepuluhnya memang wajib, termasuk `EXCLUSION_C` yang artinya tidak
diketahui sama sekali.

**Membaca SQL dari elemen yang benar.** Empat percobaan pertama mengekstrak SQL dari rule RDB
menghasilkan kosong: `<pySQL>` tidak ada di format ini. Yang berhasil adalah mencari elemen `py*`
mana pun yang isinya memuat kata kunci SQL — yang mengungkap `<pyBrowseSQL>`, dan dari sana keenam
kueri panel terbaca utuh.

## Kesalahan sendiri yang tercatat sesi ini

| # | Kesalahan | Bagaimana ketahuan | Pelajaran |
|---|---|---|---|
| 1 | **Menjalankan `go build` di latar bersamaan dengan menyunting `main.go`** | Build melaporkan `could not import claim-pnc/internal/masterpanel` padahal berkasnya sudah ada — ia membaca keadaan setengah jalan | Build yang berjalan bersamaan dengan penyuntingan tidak membuktikan apa pun. Diulang setelah seluruh berkas tertulis |
| 2 | **Uji `TestDeleteOnlyOnTheChildTable` yang tidak membuktikan apa pun** | `"LOKASI_PANEL_HE"` berakhiran `"PANEL_HE"`, sehingga `NotContains(upperCase, "PANEL_HE\n")` selalu gagal — dan bila kebetulan lolos, ia lolos karena alasan yang salah | Uji yang memakai pencocokan substring atas nama yang saling berakhiran harus menguji **sasaran pernyataannya**, bukan ada-tidaknya nama |
| 3 | **Nama aksesibel tombol hapus tersambung tanpa spasi** | Uji melaporkan nama sebenarnya: `"Hapuslokasi baris 1"` | Algoritma nama aksesibel **memangkas setiap simpul teks lalu menyambungnya tanpa pemisah**. Teks berkelas `sr-only` yang mengandalkan spasi di ujung tidak bekerja; `aria-label` eksplisit tidak punya kelemahan itu. **Ini cacat nyata pada komponen, bukan pada ujinya** — dan diperbaiki di komponennya |
| 4 | **Memanggil `npx eslint`** | Project ini tidak memakai ESLint sama sekali; `package.json` hanya punya `typecheck` dan `test`. Perintahnya justru mengunduh ESLint 10 yang tidak dipakai siapa pun | Periksa `package.json` sebelum menjalankan perkakas yang "biasanya ada" |

## Catatan untuk sesi berikutnya

**Modul `mastersupplier` dikerjakan paralel di working tree yang sama sepanjang sesi ini.** Empat
berkas bersama — `main.go`, `App.tsx`, `api/types.ts`, `app/menu/registry.ts` — disunting dua
pihak. Tidak ada yang tertimpa, dan cara memastikannya: **menambahkan di akhir berkas**, lalu
memeriksa `go build`, `npm run typecheck`, dan `git status` sesudahnya. Satu kegagalan build yang
berasal dari pekerjaan paralel itu **tidak disentuh**, dan ia konvergen sendiri.

Perlakuan yang sama dipakai untuk dokumen: bagian baru **ditambahkan di akhir**, bukan disisipkan
di tengah, supaya penomorannya tidak bertabrakan dengan bagian yang sedang ditulis pihak lain.

---

# Sesi kelima belas — Master Supplier (2026-09-20)

> Dikerjakan **paralel** dengan sesi keempat belas (Master Panel) di working tree yang sama.

## Ringkasan

| | |
|---|---|
| **Skill yang dipanggil** | **tidak ada** |
| **Skill yang ditimbang** | `mattpocock-skills:grilling`, `codebase-design`, `domain-modeling`, `tdd`, `code-review` |
| **Alat bantu yang dibuat sendiri** | empat pembaca XML Pega sekali pakai: pemindai sel layout (`pyValue` + `pyLabelFieldValue` + `pyFormat` + `pyRequired` + `pyEditOptions`), pembaca sumber dropdown (`pyListSource` + `pySourceName`), pemindai langkah activity (`pyStepsActivityName` + `pyStepsPreCondParamsWhen`), dan pembaca pasangan `PropertiesName`/`PropertiesValue` |
| **Keputusan Work Owner yang dicatat** | 3 — ketiganya "Jalankan sebagai as is" |
| **Cacat yang ditemukan uji** | 1 — nama field pelanggaran yang tidak dikenali klien bersama |

## Kenapa tidak ada skill yang dipanggil

Sama seperti empat belas sesi sebelumnya, dan alasannya tidak berubah: pekerjaan sesi ini adalah
**membaca export Pega lalu menurunkannya menjadi modul**, dan tidak satu pun skill yang tersedia
mengerjakan salah satu dari keduanya.

Lima yang ditimbang, dan kenapa kelimanya tetap tidak dipanggil:

| Skill | Kenapa ditimbang | Kenapa tidak dipanggil |
|---|---|---|
| `grilling` | Tiga hal memang perlu ditanyakan, dan yang pertama (bentuk penyimpanan) menentukan seluruh modul — ia bahkan **membalik** keputusan yang diambil Master Bengkel atas pertanyaan yang sama | Ketiga pertanyaannya sudah terbentuk sendiri dari bukti: tabel yang hanya tiga kolom, antrean persetujuan yang sisi pemutusnya tidak ada, dan lima dropdown yang daftarnya tidak ikut di export. Yang dibutuhkan hanyalah menyajikan pilihannya beserta akibat masing-masing — satu pemanggilan `AskUserQuestion` |
| `codebase-design` | Modul ini punya **empat seam**, satu lebih banyak daripada modul master mana pun: `Repo`, `LookupRepo`, `ApprovalRepo`, dan `IDSource` | Batasnya ditentukan **kepemilikan proses**, bukan pilihan rancangan: antrean persetujuan dimiliki proses yang layarnya tidak ada, sehingga ia tidak dapat digabung ke `Repo`. Yang tersisa adalah menyatukan cara memilihnya lewat satu `Store` — keputusan yang tidak punya alternatif nyata bila keempatnya harus berasal dari koneksi entitas yang sama |
| `domain-modeling` | Dua nama memikul arti yang bukan namanya: `NO_KLAIM` yang diisi **ID supplier**, dan kolom grid berlabel "JENIS SUPPLIER" yang isinya `.JENIS_STATUS_NOTE` | `CONTEXT.md` dan `D-19` sudah menetapkan kriterianya: pakai nama yang mencerminkan isi. Yang dikerjakan adalah menerapkannya — `SupplierID`, bukan `NoKlaim` — bukan menyusun kosakata baru |
| `tdd` | Aturan penyimpanannya bercabang: dua jalur yang memperlakukan status aktif secara berbeda, dan salah satunya melewati persetujuan sama sekali | Urutannya tidak dapat dibalik: prasyarat mana yang berlaku baru diketahui setelah `CreateNewMasterSupplier_post` dan `EditMasterSupplier_post` dibaca dan **disilangkan**. Uji yang ditulis sebelum itu hanya menguji tebakan. Ujinya tetap ditulis berdampingan dengan kode — 57 di backend, 22 di frontend |
| `code-review` | Empat berkas bersama **sedang disunting sesi lain** saat sesi ini berjalan | Yang dibutuhkan bukan tinjauan terhadap standar melainkan memastikan tidak ada pekerjaan pihak lain yang tertimpa, dan itu dijawab `git status`, `go build`, dan `tsc` — bukan skill |

## Teknik yang dipakai tanpa memanggil skill

**Membaca kunci JSON dari kueri BACANYA, bukan dari fungsi penulisnya.** Ini teknik yang membalik
keputusan modul ini terhadap Master Bengkel. Fungsi penulis dokumen (`@ASM.GetPageJSONString()`)
tidak ada badannya di export — sama persis dengan keadaan di Master Bengkel, yang karena itu
menolak menulis JSON. Yang berbeda: `GetDataEditMasterSupller-SQL.xml` **membaca kembali setiap
kuncinya satu per satu**, sehingga kedua puluh lima nama kunci terbaca tanpa satu pun tebakan.

Pelajarannya umum: ketika penulis sebuah dokumen tidak terbaca, **pembacanya** dapat memberi tahu
hal yang sama.

**Menyilangkan dua activity untuk memastikan sebuah turunan.** Hubungan `JENIS_STATUS` dan
`SUPPLIER_HE` tidak disimpulkan dari satu tempat. `CreateNewMasterSupplier_post` step 7 menurunkan
yang kedua dari yang pertama; `GetDataSupplier_pre` step 6.3 **membalik arahnya** saat memuat.
Keduanya saling membalik, dan justru itulah yang membuktikan keduanya dua wajah dari satu hal —
sekaligus mengungkap cacat pada yang pertama: ia hanya punya cabang "bila", tanpa cabang "selain
itu".

**Menghitung `pyRequired` dari sel layout, bukan dari daftar tag.** Percobaan pertama memindai
seluruh `<pyRequired>` di berkas section dan menghasilkan 47 nilai `true`/`false` tanpa tahu milik
isian yang mana — angka yang tidak dapat dipakai. Yang berhasil adalah memecah berkasnya pada
`Embed-Display-Table-Cell` lebih dulu, lalu membaca `pyRequired` **berpasangan dengan `pyValue`**
di dalam sel yang sama. Kelima belasnya terbaca dengan nama isiannya.

**Mencari tabel sebuah kelas Pega lewat kueri yang memakainya.** Kelas `ASM-FW-GISFW-Int-BRANCH`
tidak punya pemetaan tabel di export. Yang dilakukan bukan menebak nama tabelnya, melainkan mencari
**kueri lain yang membaca nama cabang dari ID-nya** — `GetBranchName-SQL` dan tujuh rule
`SearchKlaimBy*_RDB`, yang seluruhnya membaca `M_BRANCH` dengan kunci JSON `Name`. Itu sekaligus
mengungkap bahwa penamaan kuncinya **berbeda** dari `M_SUPPLIER`: `Name`, bukan `NAMA`.

**Membandingkan modul tetangga sebelum menyalin angkanya.** Lebar nomor urut ID di Master Bengkel
sepuluh digit; godaannya adalah menyalinnya. `PEGA_M_SUPPLIER.prc:21` menyebut **sebelas**.
Bedanya satu karakter, dan tidak mungkin terlihat tanpa membandingkannya — dijaga
`TestSequenceWidthFollowsProcedure`.

## Kesalahan sendiri yang tercatat sesi ini

| # | Kesalahan | Bagaimana ketahuan | Pelajaran |
|---|---|---|---|
| 1 | **`ViolationDTO` memakai nama field `isian`** | Uji frontend "menyorot setiap isian yang ditolak server" gagal: layar menampilkan kotak galat umum alih-alih menyorot isiannya. Klien bersama hanya membaca `field` atau `kolom` | Nama yang **lebih tepat artinya** bukan nama yang benar bila ada kontrak bersama yang sudah menetapkannya. Periksa pembacanya, bukan hanya penulisnya. **Ini cacat nyata**, dan akibatnya besar: pada form lima belas isian wajib, pengguna tidak diberi tahu yang mana |
| 2 | **Memecah klausa SELECT dengan memisahkan koma** | `TestReaderQueriesShareColumnOrder` melaporkan potongan `"JSON_VALUE(JSONDATA"` dan `"'$.NAMA')"` sebagai dua kolom terpisah | Setiap ekspresi `JSON_VALUE` memuat koma **di dalam tanda kurungnya sendiri**. Penggantinya satu regex beralternatif, dengan cabang JSON_VALUE lebih dulu supaya ia melahap seluruh ekspresinya |
| 3 | **Meninggalkan asersi tautologi** `require.Equal(x, x)` | Terbaca saat membaca ulang berkas uji sebelum menyatakannya selesai | Uji yang selalu lulus lebih buruk daripada tidak ada uji: ia memberi rasa aman tanpa menjaga apa pun. Diganti asersi atas `RequestedBy`, `Reason`, dan bentuk `PROTEKSI_ID` |
| 4 | **Mengandaikan portal tidak dikenal dijawab 404** | Uji melaporkan 400. `portalhttp.mapPortalError` memang memetakan `ErrNotFound` ke 400 dengan kode `portal_tidak_dikenal` | Status HTTP sebuah modul bersama dibaca dari **pemetanya**, bukan diandaikan dari artinya. Uji diperbaiki dan sekaligus diperkuat: kini ia memeriksa kodenya juga |
| 5 | **Heredoc bash untuk menulis dokumen panjang** | Bash melaporkan `unexpected EOF while looking for matching` — dan berkasnya tidak tersentuh sama sekali | Teks panjang berisi tanda kutip, backtick, dan pipa tidak ditulis lewat shell. Penggantinya: tulis ke berkas sementara dengan alat tulis, lalu `cat >>` |

## Catatan untuk sesi berikutnya

**Syarat menaikkan `LookupPicker` ke `components/` kini TERPENUHI.**
`master-bengkel/CityPicker.tsx` mencatat syaratnya sendiri: *"begitu ada modul KETIGA yang
membutuhkan kotak cari–pilih, ketiganya dipindahkan sekaligus."* Modul ketiga itu adalah Master
Supplier.

Yang menahannya sekarang bukan lagi syarat itu melainkan **Isolasi Protektif** — menaikkannya
menuntut menyunting dua modul master yang sudah selesai. Ia layak dijadwalkan Work Owner sebagai
satu pekerjaan tersendiri.

**Tiga kegagalan uji di `master-rekening/AccountPage.test.tsx` masih ada**, dan jumlahnya tidak
bertambah sejak §17.11. Folder `master-rekening/` dan `components/` terbukti **bersih** di
`git status`, jadi keduanya bukan berasal dari sesi ini maupun sesi paralelnya.

**Bekerja paralel berhasil tanpa satu pun pekerjaan tertimpa.** Caranya dicatat di
`catatan-pengembangan.md` §22.11, dan yang paling menentukan: **menambahkan di akhir berkas, dan
membaca ulang tepat sebelum menyunting**. Dua suntingan memang gagal karena berkasnya sudah
berubah — dan keduanya gagal dengan aman, bukan menimpa.

## Tambahan pada sesi yang sama — paginasi sesuai Pega

Diminta setelah modulnya selesai. Tidak ada skill yang dipanggil; yang dipakai satu teknik
pembacaan export dan satu keputusan batas.

**Menyurvei seluruh layar sebelum memilih angka.** Godaannya adalah membaca `pyPageSize` pada
layar Supplier saja, menemukan `20`, lalu menjadikannya bawaan komponen bersama. Yang dilakukan
sebaliknya: memindai **259 berkas** section dan harness yang memuat `pyPageSize`, lalu
menghitung nilai efektifnya per layar. Hasilnya mengubah bentuk implementasinya — angkanya
ternyata **20 pada tiga layar dan 15 pada lima layar**, sehingga ia tidak mungkin menjadi bawaan.

Satu jebakan di dalamnya: `pyPageSize` sering bernilai `"Other"`, dan angkanya sebenarnya ada di
`pyPageSizeOther`. Membaca tag pertama saja akan melaporkan delapan layar "tanpa ukuran".

**Memilih batas perubahan, bukan hanya perilakunya.** Paginasi milik komponen bersama — itu
alasan `DataTable` ada. Tetapi komponen itu dipakai sembilan layar yang sudah dinyatakan selesai,
dan Isolasi Protektif melarang perilakunya berubah. Keduanya dipenuhi sekaligus lewat satu prop
opsional yang bawaannya mempertahankan perilaku lama — dan yang menjaganya bukan niat melainkan
uji yang diletakkan **di berkas uji komponen**, bukan di uji modul yang dilindungi.

| # | Kesalahan | Bagaimana ketahuan | Pelajaran |
|---|---|---|---|
| 6 | **Menambahkan bagian dokumen di akhir berkas tanpa memeriksa judul di atasnya** | Sesi paralel sudah menambahkan bagian `## 23`, sehingga `### 22.14` milik sesi ini mendarat **di dalamnya** dan terbaca seolah bagian dari sana | "Tambahkan di akhir" aman terhadap tabrakan tulis, tetapi **tidak** aman terhadap urutan. Pada berkas yang ditulis dua pihak, periksa judul terakhir lebih dulu — lalu sisipkan pada tempatnya, bukan di ujung |

---

# Sesi 2026-09-20 (lanjutan) — koreksi Master Status Progres 1

Umpan balik Work Owner atas layar yang sudah dibangun: paginasi hilang, dan dropdown
Posisi tidak selengkap Pega. Rinciannya di `catatan-pengembangan.md` §24 dan
`keputusan-implementasi.md` §25.

## Kenapa tidak ada skill yang dipanggil

Diperiksa lebih dulu terhadap daftar skill yang terpasang, bukan dilewati begitu saja:

| Skill | Kenapa tidak dipakai |
|---|---|
| `grilling` | Pekerjaannya **bukan mengambil keputusan baru**, melainkan memeriksa dua keluhan konkret terhadap export. Yang dibutuhkan bukti, bukan pertanyaan |
| `domain-modeling` | Istilahnya sudah ada di `CONTEXT.md` dan tidak bertambah. Yang berubah **nilai**, bukan konsep |
| `codebase-design` | Batas modul dan seam tidak tersentuh. Perbaikannya di dalam satu paket yang sudah ada |
| `tdd` | Uji memang ditulis, tetapi mengikuti koreksi yang sudah terbukti dari export — bukan memandu rancangannya |
| `diagnosing-bugs` | Kedua cacat **sudah ditunjuk Work Owner beserta bukti layarnya**. Tidak ada yang perlu dipersempit |

Memanggil skill di sini akan menambah langkah tanpa menambah apa pun pada hasilnya.

## Teknik yang dipakai tanpa memanggil skill

**Memeriksa ikatan kendali, bukan hanya asal nilainya.** Inilah teknik yang *tidak* saya
pakai saat membangun modulnya, dan ketiadaannya persis yang menyebabkan cacatnya. Yang
benar: sebelum memakai sebuah daftar nilai, baca dulu `pyValue` dan `pyPrompt` sel yang
memakainya. Keduanya menunjuk `.CaseID` yang sama — dan itu langsung menjawab pertanyaan
"kode atau teks" tanpa perlu menebak.

**Membedakan "tidak ditemukan" dari "tidak ada".** Kedelapan nilai dicari ke seluruh
`Activity/`, `Section/`, `RDB List/`, dan `Data Transform/` sebelum menyimpulkan
artefaknya hilang. Tanpa pencarian yang dituntaskan, "saya tidak menemukannya" mudah
tersamar sebagai "ia tidak ada" — dan pada `R-16` keduanya berakibat sangat berbeda: yang
pertama kelalaian saya, yang kedua kekurangan export yang harus ditagih ke Tim Pega.

**Menerima tangkapan layar sebagai bukti, dengan batasnya disebut.** Karena artefaknya
memang tidak ada, layar yang sedang berjalan adalah sumber terbaik yang tersedia. Yang
dijaga: batas kesahihannya ditulis di kode, dan **dua hal yang tangkapan layar tidak dapat
jawab** — apakah "All" boleh tersimpan, dan "PROCUREMENT" versus "PROCUREMENT/SUPPLIER" —
diangkat sebagai pertanyaan, bukan diisi sendiri.

**Menelusuri akibat perubahan nilai ke seluruh jalurnya.** Mengganti kode menjadi teks
tampak seperti mengganti isi satu senarai. Yang ikut terseret ternyata `Input.Clean`:
`strings.ToUpper` benar untuk kode angka, dan diam-diam salah untuk `"All"`. Cacat itu
tidak akan terlihat dari uji mana pun yang ada — ia baru muncul sebagai baris ber-`"ALL"`
di basis data.

## Kesalahan sendiri yang tercatat sesi ini

| # | Kesalahan | Bagaimana ketahuan | Pelajaran |
|---|---|---|---|
| 7 | **Menyimpulkan pasangan kode→label dari dua properti berdampingan** pada activity yang benar, tanpa memeriksa sel yang memakainya | Work Owner membandingkan layar baru dengan Pega yang berjalan | Nilai yang berdekatan di satu berkas belum tentu berpasangan. **Ikatan kendali yang memakainya** yang menentukan — bukan kedekatan letak |
| 8 | **Tidak menyalakan paginasi** pada layar yang komponennya sudah mendukungnya | Idem | Kemampuan yang opt-in **tidak menyala sendiri**. Saat sebuah kemampuan bersama ditambahkan, layar yang sudah ada harus ditinjau ulang satu per satu — dan yang menjaganya uji, bukan ingatan |

Keduanya punya sifat yang sama, dan itu yang paling perlu diingat: **tidak satu pun
menimbulkan galat**. Layarnya tampil rapi, barisnya tersimpan, dan seluruh uji hijau.
Yang menemukannya manusia yang membandingkannya dengan sistem lama.

## Catatan untuk sesi berikutnya

Dua pertanyaan menunggu jawaban Work Owner sebelum modul ini dapat dinyatakan setara
dengan Pega — keduanya di `catatan-pengembangan.md` §24.8. Selama belum dijawab,
`position.go` memuat nilai yang **mungkin** benar, dan komentarnya menyatakan itu
terang-terangan.

## Koreksi Work Owner — kolom grid tidak sesuai Pega

Work Owner mengirim tangkapan layar Pega yang sedang berjalan dengan satu pertanyaan:
*"kenapa kolomnya tidak sesuai pega?"*

**Menghitung ulang ke export sebelum menyalin dari gambar.** Godaannya adalah membaca kolom
dari tangkapan layar lalu menuliskannya. Tangkapan layar dapat terpotong di tepi kanan, sehingga
yang dilakukan lebih dulu adalah memastikan jumlah kolomnya dari `pyColumnCount` dan urutannya
dari urutan sel di dalam definisi grid. Gambar hanya dipakai untuk hal yang **tidak ada** di
export — dan justru di situ ia paling berharga: tiga temuan yang tidak mungkin terbaca dari
export sama sekali.

**Memeriksa urutan bawaan, bukan mengandaikannya.** Sesi paralel menemukan
`pySortType = DESC` pada grid Master Bengkel, sehingga urutannya wajib diikuti. Godaannya adalah
menerapkan temuan itu ke sini juga. Yang dilakukan: memeriksanya — dan grid supplier ternyata
**tidak punya setelan urutan sama sekali**, sehingga `ORDER BY` yang sudah ada justru
dipertahankan. Temuan modul tetangga diperiksa, bukan diberlakukan.

| # | Kesalahan | Bagaimana ketahuan | Pelajaran |
|---|---|---|---|
| 7 | **Menggabungkan 9 kolom grid menjadi 7 kolom majemuk** | Work Owner melihat layarnya. **Tidak satu pun uji menangkapnya** — seluruh uji layar memeriksa isi baris, tidak satu pun memeriksa susunan kolom | Alasan dari modul tetangga **tidak boleh disalin tanpa memeriksa premisnya**. "Grid 40 kolom tidak terbaca" benar untuk Master Bengkel; grid ini sembilan kolom. Uji pembanding daftar header ditambahkan supaya penggabungan berikutnya gagal di sana lebih dulu |
| 8 | **Menamai sandi `JENIS_STATUS = "1"` sebagai "Heavy Equipment"** | Layar produksi menampilkan label "ASM" untuk kolom yang sama | Saya sendiri menetapkan aturan "sandi yang artinya tidak diketahui ditampilkan apa adanya, bukan tebakan" — lalu melanggarnya untuk satu sandi karena namanya *terdengar* jelas. Hubungan `JENIS_STATUS→SUPPLIER_HE` memang terbukti; **labelnya tidak** |

---

## Sesi Master Sparepart (2026-09-20)

### Skill yang dipanggil

**Tidak satu pun.** Alasannya sama dengan sesi-sesi modul master sebelumnya dan tidak
berubah: seluruh fakta yang dibutuhkan ada di dalam repository ini — 2.634 berkas export
Pega, `docs/Steering/`, dan sembilan modul yang sudah selesai. Tidak ada satu pun klaim di
sesi ini yang bersumber dari luar, sehingga `research` tidak relevan; dan tidak ada bug yang
sedang didiagnosis, konflik merge, maupun prototipe yang perlu dibuat.

### Teknik dari skill yang dipakai tanpa memanggilnya

**`grilling` — cari faktanya sendiri, serahkan keputusannya.**

Empat pertanyaan yang diajukan ke Work Owner seluruhnya disertai bukti terukur, bukan
"bagaimana menurut Bapak": jumlah kolom grid (5 dari 23, dengan satu sel sisa salin-tempel),
ketiga rule acuan yang penyaringnya tidak sepakat, satu-satunya jejak nilai SATUAN di seluruh
export, dan preseden §28 untuk tombol unggah.

Disiplin yang sama dipakai **setelah** jawaban diterima. Tiga dari empat jawaban berbunyi
"seperti aplikasi PEGA" — jawaban yang tampak menutup pertanyaan padahal memindahkannya
menjadi "apa yang sebenarnya dilakukan Pega". Penelusuran lanjutan itu yang menemukan bahwa
dropdown Kategori dan Tipe pada layar ini menyaring **yang sudah disetujui**, bukan yang
menunggu seperti dua rule lain yang lebih mudah ditemukan.

Manfaatnya konkret: tanpa langkah itu, layar akan menawarkan kategori yang belum disetujui,
dan kesalahannya tidak akan terlihat sampai ada kategori yang ditolak.

**`domain-modeling` — tolak istilah yang memikul dua arti.**

Empat nama warisan ditolak dan diberi nama sendiri di modul ini:

| Nama di Pega | Masalahnya | Nama di sini |
|---|---|---|
| `TempInputPanelHE.CaseID` / `.City` / `.Country` | halaman milik Master **Panel**, dipakai validasi Sparepart; ketiga propertinya tidak ada hubungannya dengan isinya | `Name`, `Number`, `Code` |
| `PART_CATEGORY_ID as "CityID"` | dialiaskan menjadi nama yang tidak ada hubungannya dengan kategori | `Category.ID` |
| `InputBengkel.ALASAN_STS_BGKL` | properti bernama "alasan status bengkel" memikul **jenis master** | tidak dibawa |
| `ErrMsg` pada `PEGA_M_SPAREPART_HE.prc` | satu keluaran memikul pesan berhasil DAN pesan galat | tidak dibawa |

**`codebase-design` — satu adapter berarti seam hipotetis.**

`LookupRepo` dideklarasikan terpisah dari `Repo` meski keduanya selalu dipilih bersama lewat
`RepoSelector`. Alasannya bukan kerapian: keduanya menjawab pertanyaan yang berbeda — "apa
pilihan yang tersedia" versus "apa isi master ini" — dan kedua tabel acuannya dimiliki sistem
lain, sehingga keduanya dapat berubah sendiri-sendiri. Yang disatukan hanyalah cara
memilihnya.

Prinsip yang sama menahan `ChoiceField.tsx` tetap di dalam modul, bukan naik ke
`shared/components/`: ia jawaban atas daftar pilihan yang hilang di SATU modul, bukan pola
antarmuka yang layak dipakai ulang. Menaikkannya akan mengundang layar lain memakainya di
tempat yang daftar pilihannya sebenarnya diketahui.

### Kesalahan sendiri yang tercatat sesi ini

**`git stash` dipakai untuk membaca keadaan.** Untuk membuktikan bahwa tiga kegagalan uji
`master-rekening` bukan akibat sesi ini, saya menjalankan `git stash push` dengan pathspec.
Perintahnya melaporkan galat pathspec — tetapi stash-nya **tetap terbentuk**, dan ia menelan
seluruh berkas frontend modul ini termasuk direktori yang belum terlacak. Dikembalikan
dengan `git stash pop`, seluruh berkas utuh, uji kembali hijau.

Yang salah bukan perintahnya melainkan pilihannya: pertanyaan "apakah berkas ini berubah?"
sudah terjawab `git diff --stat`, yang tidak menyentuh working tree sama sekali. Alat yang
mengubah keadaan tidak boleh dipakai untuk membaca keadaan — dan galat yang dilaporkannya
tidak berarti tidak ada yang terjadi.

**Label `<span>` pada ChoiceField.** Versi pertamanya memakai `<span>` sebagai label untuk
kedua varian, sehingga isiannya tidak tertaut ke labelnya. Tertangkap uji, tetapi akibat
sebenarnya lebih jauh daripada uji yang gagal: label yang tidak tertaut juga tidak dibacakan
pembaca layar.

### Catatan untuk sesi berikutnya

Modul Master Kategori Sparepart (MENU_ID 33) dan Master Tipe Sparepart (MENU_ID 34) adalah
lanjutan langsung modul ini — keduanya mengelola justru dua tabel acuan yang di sini hanya
dibaca. Section keduanya sudah terbaca dan labelnya sudah diketahui:

- Kategori: `ID Kategori Sparepart`, `Nama Kategori Sparepart`
- Tipe: `ID Tipe Sparepart`, `Nama Tipe Sparepart`, `Kategori Sparepart`

Keduanya memakai pola tiga tab dan activity persetujuan yang sama.
=======
# Penggunaan Skill — Sesi 2026-09-20 (modul Inbox Admin)

## Ringkasan

| Hal | Isi |
|---|---|
| Modul | Inbox Admin (`MENU_ID 63`, harness `PNCInboxAdmin`) |
| Skill resmi yang dipanggil | **tidak ada** |
| Teknik yang dipakai | grilling, domain modeling, codebase design — diterapkan tanpa memanggil skill |

## Skill yang ditimbang

Daftar skill yang tersedia pada sesi ini diperiksa lebih dulu. Tak satu pun cocok dengan
pekerjaan ini: yang tersedia adalah skill dokumen (`docx`, `pdf`, `pptx`, `xlsx`), skill
artefak halaman, dan skill perkakas Claude Code. Tidak ada skill pemodelan domain maupun
perancangan kode di antaranya.

Keputusan: **tidak memanggil satu pun**, dan menerapkan tekniknya secara manual. Memanggil
skill dokumen untuk menulis kode Go akan menambah langkah tanpa menambah apa pun.

## Teknik yang dipakai tanpa memanggil skill

### Grilling — delapan pertanyaan sebelum satu baris kode

Dua putaran. Putaran pertama empat pertanyaan; jawabannya memunculkan satu jawaban yang
menjawab hal lain, sehingga putaran kedua mengajukannya ulang beserta tiga pertanyaan yang
baru terbuka oleh jawaban putaran pertama.

**Manfaat yang terukur.** Empat dari delapan jawaban **mengubah lingkup pekerjaan**:

| Jawaban | Yang berubah |
|---|---|
| Tab Komunikasi tidak dipakai | Tiga tab dan satu kueri cacat tidak dibangun sama sekali |
| Tunggu API pengganti | Penyaring cabang tidak dibangun; seam-nya tetap ada |
| Hanya "Lihat Detail Klaim" | Delapan tombol tidak dibangun |
| Default All Case Admin | Tab bawaan berubah dari dugaan saya (ALL) |

Tanpa pertanyaan kedua dan ketiga, saya akan membangun tiga tab yang gagal dengan galat
Oracle dan satu penyaring yang menembus batas yang sengaja belum dilewati.

**Satu pertanyaan yang gagal, dan itu berguna.** Pertanyaan tentang tombol aksi dijawab
dengan keterangan tab bawaan. Alih-alih menebak maksudnya, pertanyaannya diajukan ulang —
dan jawaban "salah sasaran" itu ternyata memberi keterangan yang tidak saya tanyakan dan
tidak akan saya temukan sendiri.

### Domain modeling — menolak mewarisi nama yang berubah arti

Layar ini kasus terberat sejauh ini: **lima alias Pega berarti hal yang berbeda tergantung
tab mana yang terbuka.** Teknik yang dipakai sama dengan sesi View History Claim — telusuri
tiap alias ke kolom sebenarnya, lalu namai menurut ARTI bagi pengguna.

Bedanya: karena satu alias punya dua arti, pemetaannya tidak dapat ditulis sebagai satu
tabel. Ia ditulis sebagai **tiga tabel menurut kelompok tab**, di kepala `inboxadmin.sql`.

Yang dihasilkan: `WorkItem` dengan 31 isian bernama menurut yang dibaca pengguna —
`sumber_bisnis` bukan `rcvid`, `cabang_survei` bukan `keterangan`.

### Codebase design — seam yang dibenarkan, bukan yang mungkin

Dua seam saja: `Repo` dan `Clock`. Keduanya punya dua pengisi nyata (SQL dan memori; jam
sistem dan jam tetap), memenuhi aturan "satu adapter berarti seam hipotetis".

`Repo` sengaja hanya punya **satu operasi, dan tidak ada yang menulis**. Seluruh tabel yang
dibacanya milik Pega; operasi tulis yang tidak tersedia di seam tidak dapat dipakai kode
yang ditulis kemudian tanpa keputusan sadar (`P-1`).

`Repo.List` juga sengaja **tidak menerima `Pagination`**. Menaruhnya di sana akan
menyembunyikan bahwa seluruh baris memang ditarik — dan justru itulah keputusan yang perlu
tetap terlihat.

## Kesalahan sendiri yang tercatat sesi ini

| Kesalahan | Bagaimana ketahuan |
|---|---|
| Menyusun peta label-tab ke kode dari urutan markup XML | Dua arah pembacaan menghasilkan dua peta berbeda, dan keduanya bertentangan dengan semantik kueri. Ditarik sebelum dipakai |
| Regex alias di uji ikut menangkap `CAST(NULL AS DATE)` | Uji gagal pada kueri yang sebenarnya benar |
| Uji "tanpa perangkaian SQL" ikut menolak pola `LIKE` yang sah | Uji gagal pada kueri yang sebenarnya benar |
| Menonaktifkan kotak cari saat permintaan berjalan | Uji layar gagal — dan sebabnya bukan uji yang keliru melainkan cacat nyata yang menelan ketikan |

Dua yang pertama punya pola sama dengan sesi lalu: **menyimpulkan dari bentuk, bukan dari
arti.** Dua yang terakhir sebaliknya — keduanya ditemukan uji, dan yang terakhir menemukan
cacat yang tidak akan terlihat sampai ada yang mengetik cepat di layar sungguhan.

## Catatan untuk sesi berikutnya

1. **Urutan tag di dalam satu `rowdata` XML Pega ACAK.** Jangan menyimpulkan pasangan
   apa pun dari urutan kemunculannya; pakai semantik rule-nya.
2. **Modul View Claim (`MENU_ID 75`)** akan menghapus `src/app/ViewClaimPlaceholder.tsx`.
   Kuncinya sudah dikirim di setiap baris sebagai `referensi`.
3. **`gofmt -l` tidak berguna di repo ini** — berkasnya CRLF, gofmt menginginkan LF, dan ia
   menandai seluruh repo. Pakai `go vet` dan `go test`.
>>>>>>> Feat-arlexy-Inbox-admin
