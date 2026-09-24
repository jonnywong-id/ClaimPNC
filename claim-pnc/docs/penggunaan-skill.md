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


---

# Penggunaan Skill — Sesi 2026-09-19 (modul Master Tipe Surveyors)

## Ringkasan

| | |
|---|---|
| **Permintaan** | Menambahkan modul Master Tipe Surveyors, dengan `Harness/SurveyorsInbox-Harness.xml` sebagai acuan |
| **Skill Claude Code yang dipanggil** | **tidak ada** |
| **Skill Matt Pocock yang dipanggil** | **tidak ada** |
| **Teknik dari skill yang dipakai tanpa memanggilnya** | `grilling` · `domain-modeling` · `codebase-design` |
| **Perkakas yang dibuat sendiri** | dua perkakas diagnostik Go sementara, keduanya **baca-saja**, keduanya sudah dihapus |

## Skill yang ditimbang, dan kenapa tidak dipanggil

| Skill | Pertimbangan |
|---|---|
| `mattpocock-skills:grilling` | **Tekniknya dipakai, skill-nya tidak dipanggil.** Yang dibutuhkan bukan sesi penggalian panjang melainkan tiga pertanyaan yang jawabannya mengubah bentuk pekerjaan. Ketiganya diajukan sekaligus sebelum kode ditulis, lengkap dengan rekomendasi dan konsekuensinya — bentuk yang diajarkan skill itu, tanpa perlu memanggilnya |
| `mattpocock-skills:domain-modeling` | **Tekniknya dipakai.** Istilah domainnya sudah mapan di `CONTEXT.md` dan tiga modul master sebelumnya; yang perlu ditajamkan hanya satu hal — apakah `M_SURVEYORS` berisi ORANG atau GOLONGAN. Jawabannya (golongan) mempersempit lingkup modul secara berarti, dan itu didapat dari membaca `BrowseSurveyorType*-SQL.xml`, bukan dari memanggil skill |
| `mattpocock-skills:codebase-design` | **Tekniknya dipakai** untuk menempatkan seam `RepoSelector` dan memutuskan di mana middleware portal dipasang. Tidak menghasilkan artefak tersendiri |
| `mattpocock-skills:tdd` | **Tidak dipakai.** Modul ini menyalin bentuk modul yang sudah ada dan sudah teruji; menulis uji lebih dulu untuk struktur yang sudah diketahui bentuknya tidak menambah apa pun. Ujinya tetap ditulis, hanya tidak mendahului |
| `run` | **Tidak dipakai.** Menjalankan aplikasi penuh menuntut masuk lewat HCC/HCQ. Yang benar-benar perlu dibuktikan — kuerinya sah terhadap Oracle — dibuktikan langsung lewat perkakas baca-saja, jauh lebih murah dan lebih tepat sasaran |
| `code-review` · `security-review` | **Tidak dipanggil.** Belum ada permintaan review, dan diff-nya belum dikomit |

## Perkakas yang dibuat sendiri, dan kenapa itu yang paling berpengaruh

Dua perkakas Go sementara, keduanya **baca-saja**, keduanya dihapus setelah dipakai. Preseden dan
bentuknya mengikuti `catatan-pengembangan.md` §11.7.

**1. `cmd/diagsurveyor` — membaca katalog basis data.**

Inilah yang paling menentukan hasil sesi ini. Ia membalik rencana migrasi saya: tebakan awal adalah
menyalin migrasi `0002` (pindahkan JSON ke kolom, definisikan ulang view), dan katalog membuktikan
**tidak satu pun langkah itu dibutuhkan** — kolomnya sudah terisi dan view-nya sudah membacanya.

Lima hal yang tidak akan diketahui tanpa membacanya:

- `DESCRIPTION` **sudah terisi** pada seluruh 4 baris, dan view sudah membacanya.
- `DESCRIPTION` **bukan kolom virtual** (`VIRTUAL_COLUMN=NO`) — kalau virtual, ia tidak dapat
  ditulis sama sekali dan seluruh rancangan modul ini gugur.
- **Tidak ada trigger** pada tabel itu — yang memunculkan temuan bahwa layar Pega efektif tidak
  berfungsi untuk menyimpan.
- `M_SURVEY_ID` bertipe `CHAR(4)` dan kunci utamanya bernama `M_SURVEYORS_PK` — dua nilai yang
  dipakai langsung di kode dan diuji.
- Akun aplikasi **sudah** punya `INSERT`/`UPDATE`, sehingga migrasi `0003` tidak perlu meminta hak
  yang sudah ada.

**2. `cmd/diagbaca` — menjalankan jalur baca modul terhadap Oracle sungguhan.**

Menjalankan `CheckTable`, `List`, `Get`, dan `Get` atas baris yang tidak ada. Ia membuktikan yang
tidak dapat dibuktikan uji unit: kuerinya sah, pemetaan kolomnya benar, perapian `CHAR` bekerja, dan
baris yang tidak ada menghasilkan `ErrNotFound` — bukan galat lain.

Jalur **tulis** sengaja tidak dipanggil: ia menulis ke master produksi.

## Teknik yang dipakai tanpa memanggil skill

| Teknik | Bagaimana dipakai |
|---|---|
| **Menelusuri ke sumber, bukan ke dokumen turunan** | Nama modul dipastikan dari `m_menu_aplikasi_pnc.csv` (`MENU_ID 14`), bukan dari dugaan atas nama harness |
| **Membuktikan penulis tunggal, bukan mengandaikannya** | `P-1` menuntut satu penulis per tabel. Dibuktikan dengan memindai SELURUH export untuk `(INSERT\|UPDATE\|DELETE) …M_SURVEYORS` — hasilnya satu berkas |
| **Menguji disiplin, bukan mengingatnya** | Kueri SQL diuji terhadap 9 pola terlarang, `FROM DUAL` dipagari ke satu kueri, urutan kolom `SELECT` diuji sepadan dengan urutan `Scan` |
| **Mengunci angka yang dibaca dari basis data sebagai uji** | Keempat kode, kekosongan `OLD_M_SURVEY_ID`, dan ketiga kode yang dipatok kueri Pega — seluruhnya menjadi uji yang akan gagal bila datanya bergeser diam-diam |
| **Menandai asumsi sebagai asumsi** | Batas 100 karakter ditulis sebagai asumsi di kode, di keputusan implementasi, dan di daftar "belum dapat dibuktikan" — tiga tempat, supaya tidak menguap |

## Kesalahan sendiri yang tercatat sesi ini

**1. Nyaris menyalin migrasi `0002` tanpa membaca katalog.** Bentuk masternya kembar, dan kemiripan
itu membuat saya mengira langkahnya pasti sama. Seandainya dijalankan, hasilnya adalah
`CREATE OR REPLACE VIEW` yang tidak perlu terhadap view yang dibaca rule Pega — perubahan berisiko
atas sesuatu yang sudah benar. Pelajarannya: **kemiripan bentuk bukan kemiripan keadaan.**

**2. Dua kueri uji yang terlalu longgar**, keduanya di penulisan uji dan bukan di produk:
`getByLabelText(/tipe surveyor/i)` ikut menangkap label kotak cari, dan
`getByText(/administrator Claim PNC/i)` ikut menangkap pesan sidebar saat menunya kosong. Keduanya
diperbaiki dengan mempersempit sasaran, bukan dengan melonggarkan pernyataannya.

## Catatan untuk sesi berikutnya

- **Uji frontend sudah dapat dijalankan** di mesin ini — berbeda dari yang tercatat §10.8. Seluruh
  85 uji lulus.
- **ESLint dan Prettier tidak dikonfigurasi** di proyek ini. Prettier menandai berkas yang tidak
  disentuh sama sekali, sehingga menjalankannya dengan `--write` akan memformat ulang seluruh basis
  kode. Gerbang yang berlaku adalah `tsc --noEmit` dan `vitest`.
- **Butir menu tetangga belum dikerjakan:** `MENU_ID 15` "Master Surveyors"
  (`DetailSurveyorsInbox`) — daftar ORANGNYA, tabel `D_SURVEYORS` dengan 19 kolom dan alur
  persetujuan komite (`APPROVAL`, `KOMITE`, `TRFKOMITE`). Jauh lebih besar dari modul ini.

---

# Penggunaan Skill — Sesi 2026-09-19 (modul Master PIC Teknik)

## Ringkasan

| | |
|---|---|
| Permintaan | Menambahkan modul Master PIC Teknik, acuan `Harness/UserTeknisInbox-Harness.xml` |
| Skill Claude/Matt Pocock yang **dipanggil** | **tidak ada** |
| Skill yang **ditimbang lalu tidak dipanggil** | `code-review`, `simplify`, `run`, `artifact-design`, `dataviz` |
| Teknik yang dipakai tanpa memanggil skill | pembacaan berlapis export Pega, pertanyaan berjenjang, penulisan uji sebagai alat pembuktian |

Alasan tidak ada skill yang dipanggil sama dengan sesi-sesi sebelumnya: seluruh fakta yang
dibutuhkan ada **di dalam repositori ini** — export rule Pega, dokumen Steering, dan modul yang
sudah dibangun. Tidak ada satu pun klaim di sesi ini yang bersumber dari luar.

## Skill yang ditimbang, dan kenapa tidak dipanggil

| Skill | Pertimbangan |
|---|---|
| `code-review` | Berguna, tetapi sesi ini **menulis** modulnya, bukan meninjau perubahan orang lain. Pemeriksaan mutu dijalankan langsung: `go vet`, `gofmt`, `tsc --noEmit`, dan 100 uji baru |
| `simplify` | Dipertimbangkan setelah modul selesai. Tidak dipanggil karena bentuknya sudah **ditentukan modul acuan** (`mastertipesurveyors`); menyederhanakannya sepihak justru akan membuat modul kedelapan berbeda dari tujuh yang lain |
| `run` | Menjalankan aplikasi sungguhan **tidak mungkin membuktikan apa pun** di sini: dua bahan utamanya — definisi view dan baris katalog layanan — belum ada di basis data. Yang dapat dibuktikan sudah dibuktikan uji |
| `artifact-design`, `dataviz` | Tidak relevan; yang dibangun adalah layar di dalam aplikasi, bukan artefak atau visualisasi |

## Teknik yang dipakai, dan mana yang paling berpengaruh

### 1. Pembacaan export secara berlapis — bukan sekali jalan

Urutan pembacaannya menentukan hasilnya, dan tiga lapis terakhir yang paling berpengaruh:

| Lapis | Yang terjawab |
|---|---|
| `m_menu_aplikasi_pnc.csv` | memastikan harness yang diminta memang `MENU_ID 13` |
| Harness | judul, tombol, dan **ketiadaan tombol hapus** |
| Report Definition | **`STS_AKTIF = '1'` dipatok**, dan sumbernya sebuah **view** |
| Section | mana isian yang `pyReadOnly` — yang membedakan `TOTAL_JOB` dari `COUNTER_QUOTA2` |
| Activity | alur pencarian nama **dua tingkat** |
| RDB List + `.prc` | nama tabel sesungguhnya, dan cacat `ErrMsg` |

**Yang paling berpengaruh: membaca Section sampai tuntas.** Pemindaian pertama saya berhenti di
baris 21059 dan menyimpulkan `OLD_OPERATOR_ID` tidak ada di form — kesimpulan yang akan menghapus
satu isian yang sebenarnya dapat disunting. Pemindaian kedua menemukannya di baris 22351 dengan
`pyReadOnly=false`.

Pelajarannya: **berhenti membaca di tempat yang terasa cukup adalah cara paling mudah kehilangan
satu isian.**

### 2. Pertanyaan berjenjang — jawaban yang membuka pertanyaan berikutnya

Tiga putaran, dan setiap putaran hanya mungkin setelah yang sebelumnya dijawab:

```
Putaran 1  filter aktif · TOTAL_JOB · sumber nama
              └─ jawaban "tidak pakai PR_OPERATORS" menghapus satu-satunya sumber nama
Putaran 2  sumber pengganti · kolom pencocokan
              └─ jawaban menyebut GCNM_CONNECT_REST, tetapi dengan TYPESERVICE berbeda dari Pega
Putaran 3  nilai TYPESERVICE
              └─ jawabannya memuat "akan return atasan" — petunjuk yang memecahkan dua temuan
```

Menanyakan ketiganya sekaligus di awal tidak mungkin: pertanyaan putaran 2 baru ada **karena**
jawaban putaran 1, dan pertanyaan putaran 3 baru ada **karena** saya membandingkan jawaban putaran 2
dengan apa yang benar-benar dipakai Pega.

**Kalimat sampingan yang ternyata paling berharga.** Jawaban putaran 3 menyebut *"akan return
atasan"* — empat kata yang mudah dilewatkan. Menelusurinya ke `hcq_test.go:52` menemukan blok
`EmpLeader` di contoh respons yang **sudah ada di repositori sejak 2026-09-16**, dan itu memecahkan
isian Atasan sekaligus — masalah yang saya kira masih terbuka.

### 3. Uji ditulis untuk membuktikan, bukan untuk menaikkan angka cakupan

Tiga uji yang paling berguna adalah yang menguji hal yang **tidak terlihat di layar**:

| Uji | Yang dibuktikannya |
|---|---|
| `TestGetReachesInactiveTechnician` | Petugas nonaktif hilang dari daftar tetapi **tidak hilang dari sistem** — inti dari seluruh keputusan filter aktif |
| `TestUpdateCanDeactivateThenReactivate` | Jalan buntunya benar-benar tidak buntu; dijalankan bolak-balik, bukan diperiksa sekali |
| `TestWithoutPortalRejected` | Permintaan tanpa portal **ditolak**, bukan dilayani portal utama (`R-20`) |

Ketiganya lulus. Kalau salah satu gagal, yang salah adalah kodenya — bukan ekspektasinya.

## Kesalahan sendiri yang tercatat sesi ini

Empat, dan ketiga yang pertama ditangkap **perkakas**, bukan pembacaan ulang.

| # | Kesalahan | Penangkapnya | Yang saya pelajari |
|---|---|---|---|
| 1 | Alias impor memuat huruf **Sirilik** yang mirip ASCII (`к`, `і`) | pemindaian non-ASCII | Nama yang "terlihat benar" belum tentu benar; pemindaian byte lebih dipercaya daripada mata |
| 2 | Menduga `Lookup` meneruskan `ErrEmployeeUnknown` apa adanya | uji usecase | **Kodenya yang benar**, ekspektasi saya yang salah. Yang saya perbaiki adalah ujinya — bukan kodenya, supaya uji tidak dipaksa menyetujui dugaan yang keliru |
| 3 | Uji mengetik surel di atas isian yang sudah terisi otomatis | uji layar | Kegagalannya justru membuktikan fitur pengisian otomatis bekerja; saya tambahkan satu uji baru yang menguji itu secara langsung |
| 4 | Pemindaian Section berhenti terlalu awal (§ di atas) | pembacaan ulang | Satu-satunya dari empat yang **tidak** ditangkap perkakas — dan itu yang paling mahal bila lolos |

## Satu hal yang saya tahan, bukan kerjakan

Keputusan "daftar hanya menampilkan yang aktif" membawa jalan buntu yang nyata. Godaan untuk
menambahkan saklar "tampilkan nonaktif" besar — ia perbaikan yang jelas berguna.

**Tidak dilakukan.** Work Owner sudah menegaskan jawabannya setelah konsekuensinya disampaikan, dan
menambahkannya diam-diam berarti mengganti keputusan orang lain dengan keputusan saya. Yang
dilakukan sebagai gantinya: memastikan ambil-satu-baris tetap menjangkau baris nonaktif — yang
ternyata **memang perilaku Pega**, sehingga jalan buntunya tidak lebih dalam daripada aslinya.

Perbaikan yang benar di sini adalah membaca sistem lama lebih teliti, bukan menambah fitur.

## Catatan untuk sesi berikutnya

| Hal | Keterangan |
|---|---|
| Modul acuan | `mastertipesurveyors` tetap yang paling mutakhir: `RepoSelector`, `ActivePortal` di dalam `Mount`, `DisallowUnknownFields`, penulis galat sadar-portal |
| Pola baru dari sesi ini | **seam direktori pegawai** (`masterpicteknik/directory`) — dapat dipakai ulang modul mana pun yang perlu mencari pegawai; katalognya disuntik dari `cmd`, bukan diimpor |
| Yang perlu diingat | Modul lama mungkin masih ada yang belum portal-aware. Periksa `RepoSelector` sebelum menganggap sebuah modul sudah selesai |
| Jangan ulangi | Menganggap satu pemindaian berkas Section sudah cukup |

## Lanjutan sesi yang sama — putaran 2 dan 3 (penutupan pertanyaan terbuka)

Sesi ini berlanjut setelah modulnya selesai: seluruh pertanyaan terbuka diajukan, dijawab, lalu
jawabannya diterapkan. Skill yang dipanggil tetap **tidak ada**; yang berubah hanya perkakasnya.

### Perkakas ketiga yang dibuat sendiri

`cmd/diagportal` — **baca-saja**, dijalankan terhadap SELURUH portal yang terkonfigurasi, lalu
dihapus. Ia lahir dari satu jawaban Work Owner ("migrasi 0003, semua portal") yang menuntut
prasyaratnya diperiksa di setiap entitas.

Ia menjawab hal yang tidak dapat dijawab dengan membaca berkas mana pun:

| Temuan | Akibatnya |
|---|---|
| Hanya **ASM** yang kredensialnya terisi di `.env` | "Semua portal" **tidak dapat saya periksa**; pemeriksaannya berpindah menjadi bagian permintaan ke DBA |
| ASM bersih pada keempat prasyarat | Migrasi dapat dijalankan di sana tanpa kejutan |
| Nol baris melebihi 100 karakter | Batas yang baru ditetapkan tidak menjebak baris yang sudah ada |

Yang ketiga itu pemeriksaan yang **saya tambahkan sendiri** setelah batasnya ditetapkan — bukan
diminta. Alasannya: batas baru selalu punya risiko yang sama, yaitu baris lama yang melanggarnya
tidak hilang tetapi berhenti dapat disunting. Lebih murah diketahui sekarang daripada ditemukan
pengguna.

Keempat kuerinya tidak berakhir di catatan ini melainkan **dipindahkan ke dalam berkas migrasi**
sebagai LANGKAH 0 — tempat yang lebih tepat, karena di sanalah DBA membacanya.

### Teknik yang dipakai tanpa memanggil skill

| Teknik | Bagaimana dipakai |
|---|---|
| **Menolak menebak jawaban yang ambigu** | Jawaban *"sesuai dengan aplikasi PEGA sebelumnya"* punya dua bacaan berlawanan, dan salah satunya membatalkan keputusan Work Owner sendiri di putaran sebelumnya. Kedua bacaan disajikan beserta akibatnya, dan **tidak ada yang diubah sambil menunggu** |
| **Memisahkan milik siapa sebuah kerusakan** | Saat pohon merah karena modul lain, yang saya perbaiki hanya uji di `app/` yang **premisnya** batal. Dua sisanya di dalam modul yang sedang aktif disunting — dan pemiliknya menyelesaikannya sendiri sementara saya bekerja |
| **Memilih contoh uji yang tahan lama** | Uji Sidebar rusak karena memakai modul yang kemudian dibangun. Penggantinya dipilih dari butir yang memang belum dibangun DAN tidak sedang dikerjakan, dan komentarnya menyebut jebakan itu supaya tidak terulang |
| **Memeriksa alat ukur sebelum mempercayai hasilnya** | Dipakai dua kali, dan sekali gagal — lihat di bawah |

### Kesalahan sendiri pada putaran ini

**Skrip smoke saya sendiri memberi hasil palsu.** Percobaan pertama melaporkan nama ganda dijawab
`201`, bukan `409`. Sesaat itu tampak seperti cacat pada modul yang baru saya tulis.

Sebabnya urutan di dalam skrip: langkah "ubah 1003" sudah mengganti `EXPERT` menjadi `TENAGA AHLI`,
sehingga nama yang saya tabrakkan memang sudah bebas. Diuji ulang terhadap nama yang pasti ada,
hasilnya `409`.

Ini pengulangan pelajaran yang sudah tercatat di sesi 2026-09-19 sebelumnya — **alat ukur yang baru
ditulis sendiri pun perlu divalidasi** — dan kali ini yang nyaris tertipu adalah saya terhadap kode
saya sendiri. Yang menyelamatkan bukan kecurigaan, melainkan kebiasaan membaca keluaran sebelum
melaporkannya.

**Satu angka yang sempat saya laporkan salah.** Satu putaran `vitest` menghasilkan "5 berkas, 59
uji" padahal sebelumnya 8 dan 103. Saya nyaris melaporkannya sebagai berkas uji yang hilang;
pemeriksaan langsung ke direktori menunjukkan kedelapan berkasnya ada. Run itu tertangkap saat
proses paralel sedang menulis. Dijalankan ulang: 8 berkas, 103 uji, seluruhnya lulus.

### Catatan untuk sesi berikutnya

- **Bekerja di pohon yang sedang disunting orang lain menuntut verifikasi diulang, bukan
  dipercaya sekali.** Dalam sesi ini `tsc` dan `vitest` berubah hasilnya tiga kali tanpa saya
  menyentuh berkas yang bersangkutan.
- **Lima portal belum dapat dijangkau dari lingkungan pengembangan.** Setiap pekerjaan berikutnya
  yang menyentuh basis data entitas akan menemui batas yang sama, dan pemeriksaannya selalu
  berpindah ke DBA.

## Lanjutan sesi yang sama — penyelarasan lingkup portal dua modul lama

Permintaan "bereskan semua" menghasilkan pekerjaan yang jauh lebih besar dari yang tersirat:
memindahkan Master Status Klaim dan Master Rekening dari portal utama menjadi per portal. Skill
yang dipanggil tetap **tidak ada**.

### Teknik yang dipakai tanpa memanggil skill

| Teknik | Bagaimana dipakai |
|---|---|
| **Menolak menembus instruksi tetap tanpa izin tertulis** | "Bereskan semua" tidak cukup untuk mencabut Isolasi Protektif. Pekerjaannya diajukan lebih dulu beserta ukuran dan risikonya, dan baru dikerjakan setelah Work Owner mencabutnya secara eksplisit |
| **Memisahkan yang tidak dapat dikerjakan dari yang belum dikerjakan** | Dari lima sisa, empat bukan di tangan saya sama sekali. Menyebutkannya apa adanya lebih berguna daripada menjanjikan "beres" atas hal yang bergantung DBA dan Infra |
| **Memilih bentuk dari sifat modulnya, bukan dari keseragaman** | `RepoSelector` untuk satu modul, `ServiceSelector` untuk yang lain. Menyeragamkannya akan memaksa `portalAlias` menjadi parameter yang dapat tertukar pada modul yang memakainya untuk memutuskan pendaftaran ke Kasir |
| **Menulis uji untuk penjagaan, bukan untuk fungsinya** | Yang diuji bukan "handler bekerja" melainkan "handler MENOLAK permintaan tanpa entitas" — karena itulah yang baru, dan itulah yang gagalnya tidak terlihat |
| **Mengubah berbasis pola, lalu memeriksa hasilnya** | Penggantian mekanis dipakai untuk enam handler dan belasan panggilan, tetapi hasilnya selalu diperiksa `go build`, `go vet`, dan uji — bukan dianggap selesai karena skripnya berhasil |

### Kesalahan sendiri pada putaran ini

**1. Saya merusak penomoran subbagian milik modul lain.** Mengganti nomor bagian saya dari §18
menjadi §19 dilakukan dengan penggantian pola teks, dan polanya ikut mengenai `### 18.1`–`### 18.7`
milik Master PIC Teknik di atasnya. Dikembalikan berdasarkan **rentang baris**, bukan pola.

Pelajaran: **penggantian berbasis pola pada dokumen bersama harus dibatasi rentangnya** — pola yang
sama hidup di bagian milik orang lain.

**2. Satu uji lama sempat lulus karena alasan yang salah, dan saya hampir membiarkannya.**
`TestOversizedRequestBodyRejected` tetap hijau setelah middleware portal dipasang — tetapi `400`-nya
datang dari pemeriksaan portal, bukan dari penolakan badan permintaan. Ia lulus tanpa menguji apa
pun yang namanya janjikan.

Yang menangkapnya bukan uji lain, melainkan kebiasaan memeriksa **kenapa** sebuah uji lulus ketika
sesuatu di sekitarnya berubah. Diperbaiki dengan memeriksa kode galatnya, bukan hanya statusnya.

**3. Satu laporan angka yang hampir salah.** Satu putaran `vitest` melaporkan "5 berkas, 59 uji"
padahal sebelumnya 8 dan 103 — tertangkap saat proses paralel sedang menulis berkas. Diperiksa
langsung ke direktori sebelum dilaporkan; kedelapan berkasnya ada.

### Catatan untuk sesi berikutnya

- **Master Rekening kini punya uji HTTP**, yang sebelumnya tidak ada sama sekali. Modul berikutnya
  yang menyentuh pemisahan antarentitas dapat memakainya sebagai contoh.
- **Lima portal masih tidak dapat dijangkau.** Setiap penyelarasan berikutnya akan menemui batas
  yang sama: perilakunya dapat ditulis, tidak dapat dibuktikan.
- **README masih memuat sisa merge** di bagian pembuka — tiga kalimat "Yang sudah ada di tahap ini"
  yang saling bertentangan. Sengaja tidak disentuh: ia ringkasan keadaan proyek, bukan milik satu
  modul.

---

# Penggunaan Skill — Sesi 2026-09-20 (modul Master Surveyors)

## Ringkasan

| | |
|---|---|
| Permintaan | Menambahkan modul Master Surveyors, acuan `Harness/DetailSurveyorsInbox-Harness.xml` |
| Skill Claude/Matt Pocock yang **dipanggil** | **tidak ada** |
| Skill yang **ditimbang lalu tidak dipanggil** | `code-review`, `simplify`, `run`, `security-review`, `artifact-design` |
| Teknik yang dipakai tanpa memanggil skill | pembacaan berlapis export Pega, pertanyaan berjenjang sebelum menulis kode, penulisan uji sebagai alat pembuktian |

Alasan tidak ada skill yang dipanggil sama dengan sesi-sesi sebelumnya: **seluruh fakta yang
dibutuhkan ada di dalam repositori ini** — export rule Pega, dokumen Steering, dan delapan modul
yang sudah dibangun. Tidak ada satu pun klaim di sesi ini yang bersumber dari luar.

## Skill yang ditimbang, dan kenapa tidak dipanggil

| Skill | Pertimbangan |
|---|---|
| `code-review` | Berguna, tetapi sesi ini **menulis** modulnya, bukan meninjau perubahan orang lain. Pemeriksaan mutu dijalankan langsung: `go vet`, `gofmt`, `tsc --noEmit`, dan 30 uji baru |
| `simplify` | Dipertimbangkan setelah modul selesai. Tidak dipanggil karena bentuknya **ditentukan modul acuan** (`masterrekening`); menyederhanakannya sepihak justru akan membuat modul kesembilan berbeda dari delapan yang lain |
| `run` | Menjalankan aplikasi sungguhan **tidak dapat membuktikan bagian yang paling meragukan**: migrasi 0004 belum dijalankan di basis data mana pun, sehingga jalur Oracle-nya tidak dapat disentuh. Mode memori dapat dijalankan, tetapi yang dibuktikannya sudah dibuktikan uji |
| `security-review` | Sempat terasa paling relevan — modul ini menyentuh sandi bawaan dan pembuatan akun. Tidak dipanggil karena temuannya **sudah diketahui dan sudah diputuskan**: sandi sementara wajib-ganti, dan akun tidak dibuat sama sekali di tahap ini. Memanggilnya akan menghasilkan daftar yang sudah ada di `keputusan-implementasi.md` §20.4 |
| `artifact-design` | Tidak relevan: yang dibangun layar di dalam aplikasi, bukan halaman yang dipublikasikan |

## Teknik yang dipakai, dan hasilnya

### 1. Pembacaan berlapis export Pega — sebelum satu baris kode ditulis

Urutannya sengaja dari yang paling mengikat ke yang paling rinci: harness → section yang
dirujuknya → Report Definition → activity → stored procedure → kueri RDB.

**Yang dihasilkan, dan tidak akan terbaca bila urutannya dibalik:**

- **19 kolomnya** dari kedua Report Definition, bukan dari tebakan atas nama tabel.
- **Alur 20 langkah** `CNMInsertDetailSurveyors_act` beserta deskripsi tiap langkahnya — yang
  mengungkap kewajiban login untuk Internal Surveyor dan uji bentrok Operator ID.
- **Kedelapan parameter** `GCNMCreateOperator`, termasuk `pyChangePasswordOnNextLogin` yang
  mengubah penilaian risiko sandi bawaannya.
- **Alias `BUSINESS_CODE` yang sebenarnya `OPERATOR_ID`** — terbaca hanya karena kueri pengisinya
  ikut dibuka.

### 2. Pertanyaan berjenjang, bukan sekaligus

Tiga ronde. Ronde kedua **baru dapat disusun setelah** jawaban ronde pertama diketahui: lingkup
penuh berarti alur komite dibawa, dan barulah ketiadaan rule Approve/Reject menjadi pertanyaan
yang perlu diajukan.

Satu pertanyaan **tidak** diajukan ketiga kalinya. Work Owner sudah dua kali menjawab "seperti
Pega" soal pembuatan akun; menanyakannya lagi akan mengulang keputusan yang sudah diambil.
Sebagai gantinya, bacaan saya **dinyatakan sebagai asumsi** beserta alasannya, dan Work Owner
diberi jalan membalikkannya.

### 3. Uji sebagai alat pembuktian, bukan pelengkap

30 uji baru. Yang paling berguna justru yang **gagal lebih dulu**:

`TestKueriUpdateTidakMenyentuhKolomYangDimilikiSistem` merah karena pola `"KOMITE ="` ikut cocok
dengan `TGL_APPROVE_KOMITE = :16`. **SQL-nya benar; uji saya yang kurang tajam.** Diperbaiki
dengan mengurai nama kolom alih-alih mencocokkan substring.

Ini pelajaran yang sama persis dengan cacat yang tercatat di Steering: toleransi spreading sistem
lama memakai `@contains` dan karenanya meloloskan `199.99`. Pencocokan substring atas hal yang
berstruktur adalah kelas kesalahan yang berulang.

## Tiga koreksi atas diri sendiri pada sesi ini

Dicatat karena pola kesalahannya lebih berguna daripada kesalahannya.

| Koreksi | Bagaimana ketahuan | Pelajaran |
|---|---|---|
| **`KomitePost_Survey` disangka rule Approve/Reject yang dicari** | Dibaca sampai habis: ia membuat child case dan menerbitkan Letter of Assignment — komite penunjukan surveyor pada KLAIM, milik `B-8` | Nama yang cocok bukan bukti. Isi yang menentukan |
| **Sandi `login+123456` dilaporkan sebagai sandi lemah permanen** | `GCNMCreateOperator` menyetel `pyChangePasswordOnNextLogin = "True"` — ia sandi sementara | Melaporkan risiko sebelum membaca mekanismenya sampai habis membuat dasar keputusan Work Owner keliru. Koreksinya disampaikan sebelum ia memutuskan |
| **Templating `{{columns}}` dan `REPLACE` dua argumen di berkas SQL** | Keduanya ditulis lalu langsung ditinjau ulang: templating tidak ada di codebase ini, dan `REPLACE` dua argumen tidak ada di PostgreSQL — melanggar `D-20` | Menyalin pola dari kebiasaan, bukan dari codebase yang ada, memasukkan mekanisme yang tidak dipakai siapa pun |
| **Uji frontend dilaporkan "tidak dapat dijalankan"** | Percobaan pertama `vitest` gagal menyalakan worker. Dijalankan ulang dengan satu worker, 121 uji berjalan normal — dan satu di antaranya merah karena modul ini | Kegagalan perkakas dilaporkan sebagai kegagalan perkakas, lalu **dicoba ulang** — bukan dijadikan alasan menyatakan verifikasi selesai |
| **"Rule Approve/Reject tidak ada di export" — SALAH, dan sempat ditulis di enam tempat** | Work Owner bertanya *"rule Approve/Reject ini dipanggil di mana"*. Penelusuran pemanggilnya menemukan kedua tombol memanggil `CNMInsertDetailSurveyors_act` dengan `approval="1"`/`"2"` | Saya mencari rule **bernama** Approve/Reject, tidak menemukannya, lalu berhenti. **Mencari berdasarkan nama lalu menyimpulkan ketiadaan adalah kekeliruan yang sama dengan `KomitePost_Survey`** — hanya arah sebaliknya. Yang benar: telusuri PEMANGGILNYA, bukan namanya |
| **Email diperlakukan opsional, padahal WAJIB** | Work Owner menyuruh *"coba dicek lg"*. Langkah 4 menyiapkan `local.email := "Email harus diisi"`, langkah 5 memancarkannya dengan prasyarat `EMAIL==""` | Saya menyimpulkan "tidak ada aturan yang mewajibkan" setelah memeriksa **struktur** langkahnya, tanpa membaca **variabel pesannya**. Teks galat adalah bukti aturan yang paling langsung, dan ia yang terakhir saya baca |
| **`Update` tidak mengembalikan surveyor ke antrean komite** | Dari pemeriksaan yang sama: langkah 4 ber-`pyStepsPreCondition: true` menyetel `APPROVAL := Param.approval` **tanpa syarat**, dan tombol Simpan mengirim `"0"` | Saya membaca nilai parameter Approve/Reject, lalu berhenti — tanpa bertanya **apa yang dikirim tombol Simpan**. Akibatnya paling berat di sesi ini: nama login dapat diubah setelah komite menyetujui, melubangi satu-satunya kontrol yang `D-59` sisakan |

Ditambah dua hal kecil yang diperbaiki sendiri sebelum sempat menjadi cacat: `itoa` buatan
sendiri padahal modul lain memakai `strconv`, dan nomor urut penyimpanan memori yang dimulai dari
1 sehingga penambahan pertama akan menimpa contoh data.

## Catatan untuk sesi berikutnya

- **Migrasi 0004 WAJIB dijalankan** sebelum modul ini dipakai di atas Oracle — berbeda dari 0003
  yang opsional. Langkah 0d-nya menjawab pertanyaan yang paling menentukan: apakah
  `V_D_SURVEYORS` membaca kolom atau JSON.
- **`vitest` perlu dijalankan dengan `--no-file-parallelism --maxWorkers=1`** di mesin ini.
  Tanpa itu ia gagal menyalakan worker (`Timeout waiting for worker to respond`) dan melaporkan
  "no tests" — yang mudah disalahartikan sebagai kerusakan. Dengan satu worker, 121 uji lulus.
- **Uji lama dapat menjadi merah karena modul baru, dan itu pernah terjadi dua kali di berkas
  yang sama.** `Sidebar.test.tsx` memakai butir menu "yang belum punya modul" sebagai contoh;
  contohnya usang setiap kali modul itu dibangun. Kini ia memakai harness yang tidak ada di
  export sama sekali. **Periksa berkas itu setiap kali menambah butir menu.**
- **Modul ini PUNYA baseline** dan dapat diuji setara dengan Pega — dikoreksi 2026-09-20 setelah
  pemanggil rule Approve/Reject ditelusuri. Pernyataan sebaliknya di versi sebelumnya dicabut.
- **Email WAJIB** (terjawab 2026-09-20), dan **menyunting mengembalikan surveyor ke antrean
  komite**. Keduanya cacat implementasi yang ditemukan karena Work Owner menyuruh memeriksa
  ulang; keduanya sudah diperbaiki beserta ujinya.
- **Satu hal terbuka:** arti deret tipe `1021`–`1028` pada precondition langkah 8. Ia memancarkan
  pesan *"Nama Belum Terdaftar."*, bukan pesan login — dugaan awal saya tidak terbukti. Ke
  **Tim Pega**.
- **Pola yang berulang tiga kali di sesi ini:** berhenti pada bukti pertama yang cocok. Nama rule
  yang tidak ketemu disimpulkan hilang; struktur langkah yang diperiksa tanpa membaca pesannya;
  parameter Approve/Reject dibaca tanpa menanyakan parameter Simpan. Ketiganya ketahuan hanya
  karena Work Owner bertanya lagi.

---

## Sesi kedua belas — Modul Master Recovery (2026-09-20)

### Skill yang dipakai

| Skill | Kapan | Keluaran | Manfaat nyata |
|---|---|---|---|
| **`mattpocock-skills:grilling`** | Sebelum satu baris kode ditulis | Tiga pertanyaan ke Work Owner beserta bukti yang mendasarinya | **Ini yang paling menentukan pada sesi ini.** Pembacaan export membuktikan layarnya bukan CRUD master, dan tiga pembacaan yang berbeda menghasilkan pekerjaan yang berbeda. Disiplinnya: bawa BUKTI, bukan pertanyaan terbuka — setiap pilihan disertai apa yang ditemukan di export |
| **`mattpocock-skills:codebase-design`** | Saat menentukan batas modul | Dua seam: `Repo` dan `VirtualAccountIssuer` | Aturan "satu adapter berarti seam hipotetis, dua adapter berarti seam nyata" yang menentukan bentuknya. `VirtualAccountIssuer` punya dua pengisi nyata — Pega dan Fake — dan Fake-nya **bukan kenyamanan**: alamat yang terdaftar menerbitkan rekening SUNGGUHAN, sehingga tanpa adapter kedua tidak ada cara menguji jalur ini sama sekali |
| **`mattpocock-skills:domain-modeling`** | Saat menamai field | Tiga nama properti Pega yang **dikoreksi**, bukan disalin | `NoKTPMasking` menyimpan **posisi kasus**, `TPLAmount` menyimpan **sisa**, `Idlogservice` menyimpan **nomor log layanan** — bukan nomor telepon meski kolomnya bernama `NOHPLL`. Membawa namanya berarti membawa kekeliruannya ke sistem baru selama sepuluh tahun berikutnya |
| **`mattpocock-skills:tdd`** | Saat menulis uji layar | Satu cacat nyata ditemukan | Uji yang memeriksa **isi** dropdown Tahun, bukan sekadar keberadaannya, menemukan bahwa isian itu tidak pernah terisi karena `defaultValues` dibaca sebelum data tiba |

### Cara verifikasi yang dipakai, dan kenapa

Bukan skill, tetapi kebiasaan yang paling banyak menyelamatkan pada sesi ini:
**memeriksa ke basis data yang berjalan, bukan ke nama parameter procedure.**

Yang ditemukan karenanya, dan tidak akan ditemukan dengan membaca export saja:

| Temuan | Akibatnya bila tidak diperiksa |
|---|---|
| `INSERTDATE` ber-`DEFAULT sysdate` | Aplikasi akan menulisnya dari jam server aplikasi — dan pada dua instans di belakang penyeimbang beban, keduanya belum tentu sama (`R-12`) |
| `DATA_ATTACHFILE` punya kolom BLOB | Unggahan bukti bayar akan ditunda sampai modul `S-1` ada, padahal jalurnya sudah tersedia |
| `NUMBER` tanpa presisi dan skala | Keputusan "rupiah utuh" akan tampak dibaca dari kolomnya, padahal ia **keputusan** |
| Nol baris bernilai pecahan | Keputusan itu tidak akan punya dasar sama sekali |
| `DATAID` berbentuk YY + 10 angka | Penomoran lampiran akan dikarang, dan bertabrakan dengan yang diterbitkan Pega |

### Kesalahan sendiri pada putaran ini

**1. Satu uji saya lulus karena alasan yang salah.** `TestTidakAdaDaftarUbahMaupunHapus`
memastikan `PUT` dan `DELETE` tidak dilayani, dan ia hijau. Uji asap terhadap aplikasi
yang benar-benar berjalan menunjukkan keduanya dijawab **200 dengan HTML SPA**.

Sebabnya: uji HTTP merakit modulnya saja, tanpa penyaji SPA. Yang menangkapnya bukan uji
lain, melainkan **menjalankan aplikasinya sungguhan**. Diperiksa lebih lanjut pada modul
lain — perilakunya sama di sana — sehingga ia temuan aplikasi, bukan cacat modul ini, dan
diserahkan alih-alih diperbaiki sepihak.

Pelajarannya mengulang §17.11 dan §19.5 dalam bentuk ketiga: **uji yang merakit sebagian
aplikasi hanya membuktikan hal tentang sebagian itu.**

**2. Tiga percobaan terbuang karena alat, bukan karena kode.** Satu berkas sumber Go
ditolak kompilator karena huruf BOM yang tidak terlihat ikut tertulis di tengah baris;
uji asap pertama dijawab proxy Squid, bukan oleh aplikasinya; dan satu bendera vitest yang
tidak didukung versi ini dilaporkan sebagai kegagalan uji.

Ketiganya punya pola yang sama: **kegagalan alat menyamar sebagai kegagalan pekerjaan.**
Yang membedakannya adalah membaca pesannya sampai habis alih-alih mengulang perintahnya.

### Catatan untuk sesi berikutnya

- **`src/lib/money.ts` kini ada.** Ia berkas pemformatan uang bersama yang pertama. Modul
  berikutnya yang menangani nilai uang memakainya kembali — jangan menulis versinya
  sendiri, karena itu persis kegagalan `TO_CHAR` tersebar di 411 tempat yang sedang kita
  tinggalkan.
- **`uploadAPI` dan `downloadAPI` kini ada di `api/client.ts`.** Modul berikutnya yang
  perlu mengunggah atau mengunduh berkas memakainya, bukan memanggil `fetch` sendiri.
- **Dua rule hilang menghalangi modul ini menjadi lengkap** — `VirtualAccountClaimsPNC`
  dan `GetListYear`, keduanya bagian dari `R-16`. Keduanya sudah ditandai di kode pada
  tempat yang tepat, sehingga yang menyambungnya kelak tidak perlu mencarinya.
- **Pekerjaan paralel sedang berjalan di repo yang sama.** Modul `mastersurveyors` muncul
  di pohon kerja selama sesi ini. Tidak ada bentrokan — keduanya menyentuh berkas bersama
  pada baris yang berbeda — tetapi berkas bersama (`App.tsx`, `registry.ts`, `types.ts`,
  `main.go`) sebaiknya dibaca ulang sesaat sebelum disunting.

---

## Sesi ketiga belas — Modul Master Dominan Factor (2026-09-20)

### Skill yang dipakai

| Skill | Kapan | Keluaran | Manfaat nyata |
|---|---|---|---|
| **`mattpocock-skills:grilling`** | Sebelum satu baris kode ditulis | Tiga pertanyaan ke Work Owner, masing-masing disertai bukti `berkas:baris` dan akibat tiap pilihan | **Yang paling menentukan pada sesi ini.** Pertanyaan 2 mengubah bentuk seluruh modul: jawabannya "tiru Pega" menghapus kebutuhan indeks unik, menghapus galat `ErrNameTaken`, dan menghapus **seluruh berkas migrasi**. Bila saya menyamakannya dengan dua master sebelumnya tanpa bertanya, saya akan menulis migrasi `0005` yang tidak diperlukan dan menolak isian yang seharusnya diterima |
| **`mattpocock-skills:codebase-design`** | Saat menentukan letak `SortByID`, `NextID`, `NumericID` | Ketiganya di **lapisan domain** (`ordering.go`), bukan di masing-masing adapter | Pertanyaannya: "apa yang akan rusak bila ini diduplikasi?" Jawabannya konkret — adapter SQL dan adapter memori akan dapat berbeda pendapat tentang urutan yang benar, dan uji yang lulus di memori akan gagal di Oracle. Satu fungsi bersama membuat perbedaan itu mustahil |
| **`mattpocock-skills:domain-modeling`** | Saat menamai field | Kolom `NAME` → field domain `Name` → JSON `nama` → label layar **"Keterangan"** | Empat nama untuk satu hal, dan keempatnya benar karena peruntukannya berbeda (`D-80` untuk kode dan kontrak, `D-13` untuk teks yang dilihat pengguna). Menyeragamkannya akan melanggar salah satu dari keduanya |
| **`mattpocock-skills:tdd`** | Saat menulis uji | Uji ditulis sebagai **pemagar keputusan**, bukan hanya pembukti kode | `TestEmptyNameAccepted` dan `TestCreateAcceptsEmptyAndDuplicateNames` akan gagal bila seseorang menambahkan `.min(1)` demi keseragaman dengan modul tetangga. Komentarnya menyebut keputusan yang harus dibaca lebih dulu — sehingga uji yang gagal menuntun ke jawabannya, bukan ke perdebatan |

### Kebiasaan yang paling menyelamatkan pada sesi ini

**Membaca procedure basis data, bukan menyimpulkan dari layar atau dari modul tetangga.**

Bila saya menyalin pola Master Status Klaim — yang wajar, karena bentuk layarnya nyaris
identik — saya akan menerbitkan ID berbentuk `1011` alih-alih `11`. `PEGA_M_DOMINAN_FACTOR.prc:11`
memakai `max(ID)+1` murni, **tanpa** kode situs dan **tanpa** sequence.

Yang membuatnya mudah tertukar: hasilnya ditampung ke variabel bernama **`id_site`**, dan
`id_dominan_factor varchar2(4)` dideklarasikan lalu tidak pernah dipakai — keduanya jejak
salin-tempel dari procedure master lain. **Nama variabel di sistem lama tidak dapat dipercaya
sebagai keterangan isinya**, dan ini contoh ketiga di repo ini setelah `NoKTPMasking` dan alias
kolom SQL warisan.

### Satu hal yang saya tahan, bukan kerjakan

**Tidak menulis migrasi `0005`.** Dorongan untuk menambahkan indeks unik atas `NAME` kuat —
dua master sebelumnya punya, dan pola berulang terasa seperti kebenaran. Tetapi Work Owner
memutuskan nama ganda diterima, sehingga indeks itu akan **menolak data yang sah** dan membuat
modulnya gagal pada penyimpanan pertama yang namanya kebetulan sama.

Pola yang berulang tiga kali belum tentu aturan. Yang membedakannya adalah bertanya.

### Satu koreksi atas diri sendiri

Saya sempat melaporkan suite frontend gagal — `master-recovery/RecoveryPage.test.tsx`, *"Test
timed out in 5000ms"*. Dijalankan sendirian berkas itu **lulus (16 uji)**, dan putaran suite
penuh berikutnya **lulus seluruhnya (137 uji)**.

Ia flake batas waktu di bawah beban paralel, bukan kegagalan pekerjaan — dan modul Master
Recovery tidak disentuh sama sekali pada sesi ini. Yang benar dilakukan: **menjalankan ulang dan
menjalankan terpisah sebelum menyimpulkan**, bukan langsung mencari sebabnya di perubahan
sendiri.

Ini mengulang pola "kegagalan alat menyamar sebagai kegagalan pekerjaan" dari sesi kedua belas,
dalam bentuk keempat.

### Catatan untuk sesi berikutnya

- **`masterdominanfactor.SortByID` dan `NextID` dapat dipakai ulang.** Master lain yang
  ID-nya dibentuk `max+1` tanpa padding — bila ada — memakai polanya, bukan menulis versinya
  sendiri. Periksa procedure-nya dulu: **tidak semua master memakai skema yang sama.**
- **Isi `M_DOMINAN_FACTOR` masih perlu diminta ke DBA.** Satu kueri:
  `SELECT ID, NAME FROM POOLDATA.M_DOMINAN_FACTOR ORDER BY TO_NUMBER(ID)`. Sampai itu tiba,
  `SampleList` berisi teks karangan yang **sudah ditandai besar-besar** sebagai karangan —
  jangan mengutipnya ke dokumen mana pun.
- **Modul ini tidak menuntut migrasi.** Bila kelak DBA melaporkan lebar kolom `NAME` lebih
  sempit dari 100, yang berubah cukup satu konstanta `MaxNameLength` — bukan berkas migrasi.
- **Tiga master kini punya aturan validasi yang BERBEDA.** Status Klaim dan Tipe Surveyors
  menolak kosong dan ganda; Dominan Factor menerima keduanya. Perbedaan itu disengaja dan
  dipagari uji. Jangan menyeragamkannya tanpa keputusan baru.

## Sesi keempat belas — Modul Master Masking (2026-09-20)

### Ringkasan

| | |
|---|---|
| **Skill yang dipakai** | `mattpocock-skills:grilling` · `mattpocock-skills:domain-modeling` · `mattpocock-skills:codebase-design` |
| **Skill yang ditimbang, tidak dipakai** | `tdd` · `diagnosing-bugs` · `research` · `prototype` |
| **Hasil** | modul lengkap dua sisi, 48 paket Go + 153 uji frontend lulus, jalur baca terbukti terhadap Oracle |

### `grilling` — dipakai paling berat, dan dua kali menyelamatkan pekerjaan

**Alasan memakainya.** Work Owner secara eksplisit melarang menulis kode sebelum analisis
selesai. Itu persis disiplin skill ini: **fakta dicari sendiri, keputusan diserahkan ke
pemilik project** — dan setiap pertanyaan diajukan dengan angka, bukan dengan "bagaimana
menurut Bapak".

**Cara dipakainya.** Delapan pertanyaan dalam dua ronde, seluruhnya disertai rekomendasi
beserta alasannya. Ronde kedua **hanya diajukan setelah** ronde pertama dijawab, karena
jawabannya mengubah pertanyaan berikutnya — itu aturan *frontier* pada skill ini.

**Hasil yang paling menentukan: satu keputusan ditinjau ulang di tengah jalan.**

Pada ronde pertama, Work Owner memilih kolom `PASSWORD` "dibawa apa adanya" — dan saya
menyajikan pilihan itu dengan premis bahwa kolom tersebut berisi kata sandi. Penelusuran
lanjutan membuktikan premis saya salah:

- `Activity/InsermaskingDataKlaimPnc_-Act.xml` menetapkan literal `"saya"`;
- tidak ada isian PASSWORD di layar Pega sama sekali;
- seluruh logika verifikasinya dikomentari di procedure.

Temuan itu **dilaporkan kembali sebelum satu baris kode ditulis**, beserta catatan bahwa
keputusan tadi diambil di atas premis yang keliru. Work Owner menimbang ulang dan tetap
memilih menulis literal yang sama — kali ini dengan mengetahui apa isinya.

Inilah yang skill ini maksud dengan *"kontradiksi diangkat, tidak diserap diam-diam"*.
Menjalankan keputusan pertama tanpa mengoreksi premisnya akan menghasilkan kode yang
benar secara perintah tetapi salah secara pemahaman.

**Hasil kedua: satu pertanyaan yang tidak terpikir diajukan sama sekali.** Menyiapkan
pilihan untuk MODUL/SUB MODUL memaksa saya memeriksa dari mana daftarnya berasal — dan
menemukan rule `MODULKLAIMMASKING` **hilang dari export** (`R-16`). Tanpa itu, saya akan
mengarang daftar pilihan dari satu-satunya nilai yang kebetulan ada di data, lalu
menolak nilai sah yang belum pernah dipakai.

### `domain-modeling` — menajamkan tiga istilah yang menyesatkan

**Alasan memakainya.** Sama seperti pada sesi-sesi sebelumnya, masalah terbesar bukan
kerumitan teknis melainkan **bahasa yang kacau**. Di modul ini tiga gejalanya tajam:

| Istilah lama | Yang dikira | Yang sebenarnya |
|---|---|---|
| `LOGSEARCH` / `LOGSEEN` | penanda boleh/tidak | **KUOTA** — sebarannya 1 sampai 100.000 |
| Tombol `DELETE` | menghapus baris | hanya mengubah `STS_AKTF` |
| `InputData.BranchID` | kode cabang | **`STS_AKTF`** — alias yang menunjuk kolom lain |

**Cara dipakainya.** Skill ini menuntut **menyilangkan pernyataan dengan kode** sebelum
menerima arti sebuah istilah. Label "MAX CARI DATA" di harness menjadi petunjuk pertama
bahwa `LOGSEARCH` bukan flag; tipe `INT` pada parameter procedure menguatkannya; dan
sebaran nilai di basis data menutupnya.

**Hasilnya di kode.** Nama domain dipilih supaya artinya tidak dapat tertukar:
`SearchQuota`/`ViewQuota` (bukan `LogSearch`), `SetActive` (bukan `Delete`), `BranchID`
(bukan `BranchID` versi Pega yang berarti status). Pemetaan lengkapnya masuk
`peta-penamaan.md`.

### `codebase-design` — menempatkan satu seam baru pada tempat yang benar

**Alasan memakainya.** Satu pertanyaan rancangan muncul: daftar cabang dibaca dari
`POOLDATA.BRANCH`, dan modul Master Surveyors juga menyentuh cabang. Haruskah dijadikan
modul bersama?

**Cara dipakainya.** Prinsip *"satu adapter berarti seam hipotetis, dua adapter berarti
seam nyata"* dipakai sebagai penyaring — dan jawabannya **tidak**. Master Surveyors
menyimpan nama cabang sebagai **teks biasa**; modul ini memperlakukan kodenya sebagai
**kunci asing sungguhan**. Keduanya tidak membutuhkan hal yang sama, dan menyatukannya
sekarang akan memaksa dua hal berbeda menjadi satu.

**Hasilnya.** `ListBranches` dan `BranchExists` tinggal di seam `Repo` modul ini. Bila
modul ketiga membutuhkannya, barulah ia naik — dan saat itu bentuknya sudah diketahui dari
dua pemakai nyata, bukan ditebak dari satu.

### Skill yang ditimbang tetapi tidak dipakai

| Skill | Alasan |
|---|---|
| `tdd` | Uji ditulis setelah rancangan mengeras, bukan sebelum. Pada modul yang bentuknya ditentukan tabel yang sudah ada, menulis uji lebih dulu akan menguji tebakan tentang kolom — bukan kolomnya |
| `diagnosing-bugs` | Tidak ada bug yang didiagnosis. Cacat yang ditemukan di sistem lama dicatat sebagai temuan, bukan diperbaiki di sini |
| `research` | Seluruh fakta ada di dalam repository dan di basis data yang dapat dibaca. Tidak satu pun klaim di sesi ini bersumber dari luar |
| `prototype` | Tidak ada pertanyaan desain yang membutuhkannya; semuanya terjawab dengan membaca export dan bertanya |

### Dua kesalahan sendiri pada sesi ini

**1. Dugaan `'1'`/`'0'` yang hampir masuk ke kode.** Blok yang **dikomentari** di
`UPDATE_LOG_PROTEKSI.prc:126-131` memuat `if T_STS_EMAIL='1'`, dan saya nyaris memakainya
sebagai dasar penerjemahan kolom. Basis data membuktikan nilainya `'Ya'`/`'Tidak'`.

Polanya sama dengan kesalahan arti kode status pada sesi-sesi sebelumnya: **kode yang
tidak berjalan bukan bukti tentang data yang berjalan.** Kalau dugaan itu dipakai, setiap
kewenangan akan terbaca sebagai "tidak boleh" — dan seluruh 25 baris produksi kehilangan
maknanya tanpa satu pun galat.

**2. Uji asap pertama dijawab proxy Squid, bukan oleh aplikasinya.** Kesalahan yang persis
sama sudah tercatat di berkas ini dari sesi kedua belas, dan saya mengulanginya.
Perbaikannya satu bendera: `curl --noproxy '*'`. Dicatat ulang di sini karena catatan
sebelumnya jelas belum cukup menonjol untuk mencegahnya terulang.

### Catatan untuk sesi berikutnya

- **Dua probe Oracle sementara ditulis lalu dihapus.** Polanya berguna dan layak diulang:
  satu probe membaca **struktur dan sebaran** (`ALL_TAB_COLUMNS`, `GROUP BY`), satu lagi
  menjalankan **repo yang sesungguhnya** untuk membuktikan kueri `.sql` benar-benar sah.
  Keduanya murni `SELECT`; jalur tulis tidak pernah disentuh terhadap tabel produksi.
- **`config.LoadEnvFile(".env")` harus dipanggil sendiri** oleh program selain
  `cmd/claimpnc` — `config.Load()` tidak membacanya. Satu percobaan terbuang karena ini.
- **ESLint tidak terpasang di project ini** (tidak ada `eslint.config.*` maupun skrip
  `lint`). Gerbang frontend yang benar-benar berjalan adalah `tsc --noEmit` dan `vitest`.
- **Pekerjaan paralel masih berjalan di repo yang sama.** Modul `masterdominanfactor`
  muncul di pohon kerja sebelum sesi ini. Tidak ada bentrokan, tetapi berkas bersama
  (`main.go`, `App.tsx`, `registry.ts`, `types.ts`) tetap dibaca ulang sesaat sebelum
  disunting — dan itu memang menyelamatkan satu suntingan.

---

## Sesi kelima belas — Modul Master Penyebab Kerugian (2026-09-20)

### Ringkasan

| | |
|---|---|
| **Skill yang dipakai** | `mattpocock-skills:grilling` · `mattpocock-skills:domain-modeling` · `mattpocock-skills:codebase-design` |
| **Skill yang ditimbang, tidak dipakai** | `tdd` · `diagnosing-bugs` · `research` · `prototype` · `code-review` |
| **Hasil** | modul lengkap dua sisi, seluruh uji backend lulus, 172 uji frontend lulus, jalur tulis-baca terbukti terhadap aplikasi berjalan |

### `grilling` — dipakai paling berat, dan menemukan satu hal yang mengubah lingkup

**Alasan memakainya.** Work Owner secara eksplisit melarang menulis kode sebelum analisis
selesai. Itu persis disiplin skill ini: **fakta dicari sendiri, keputusan diserahkan ke
pemilik project** — dan setiap pertanyaan diajukan dengan angka, bukan dengan "bagaimana
menurut Bapak".

**Cara dipakainya.** Tiga pertanyaan dalam satu ronde, seluruhnya disertai rekomendasi
beserta alasannya, dan **tidak satu pun diajukan sebelum buktinya terkumpul**. Sebelum
bertanya, yang sudah dihitung: berapa rule membaca view-nya (19), berapa layar menulis
tabelnya (2), dan bagaimana persisnya ID dibentuk.

**Hasil yang paling menentukan: `P-1` ternyata tidak terpenuhi, dan itu hampir terlewat.**

Modul ini tampak sebagai salinan Master Status Klaim — pola dokumen JSON yang sama,
pembentukan ID situs+urutan yang sama, procedure dengan cacat COMMIT yang sama. Godaannya
kuat untuk langsung mengikuti migrasi 0002 apa adanya.

Yang menahan adalah satu pertanyaan yang dituntut skill ini sebelum menyalin keputusan
lama: **"apa yang membuat keadaan ini berbeda?"** Pemeriksaan pemanggil
`RDB List/UpdateMCauseOfLoss-SQL.xml` menemukan **dua** layar, bukan satu:
`CauseOfLossInbox` dan `CauseOfLossInboxSimasOnline`.

Akibatnya bukan detail: memindahkan view ke kolom tanpa menyadari penulis kedua akan
membuat baris tulisan Simas Online tampil **tanpa keterangan** di 19 rule pembaca —
**tanpa satu pun galat**. Layar tampil normal, laporan tetap jalan, hanya kelompoknya
kosong. Kelas cacat yang paling sulit ditemukan.

Temuan itu disampaikan **sebelum** pertanyaan penyimpanan diajukan, sehingga keputusannya
diambil dengan mengetahui harganya. Work Owner tetap memilih pindah ke kolom, dan yang
dikerjakan sebagai gantinya adalah membuat celahnya terlihat: peringatan di kepala
migrasi, larangan menjalankan langkah 3 sebelum keputusan diambil, kueri penghitung, dan
laporan di mode periksa.

**Hasil kedua: satu temuan yang tidak dicari.** Menelusuri layar Simas Online untuk
memastikan tabelnya sama menemukan bahwa ia menitipkan empat field tambahan — `BISNISID`,
`MST_COL_ID`, `DISC`, `KOMISI` — ke dalam **dokumen JSON yang sama**, dan keempatnya tidak
ada di view maupun tabel mana pun di export. Itu bukti langsung tentang bahaya pola
dokumen JSON, dan ia dicatat sebagai bahan yang harus dijawab saat MENU_ID 21 dipindahkan.

### `domain-modeling` — memisahkan dua tingkat yang bernama nyaris sama

**Alasan memakainya.** Gejala yang sama dengan sesi-sesi sebelumnya: masalah terbesar
bukan kerumitan teknis melainkan **bahasa yang mudah tertukar**.

| Yang mudah tertukar | Golongan | Rincian |
|---|---|---|
| Tabel | `M_CAUSE_OF_LOSS` | `D_CAUSE_OF_LOSS` |
| Harness | `CauseOfLossInbox` | `DetailCauseOfLoss` |
| Menu | 20 | 38 |
| Lebar urutan ID | **tiga** digit | **empat** digit |
| Kolom | `COL_DESC` | `DESCRIPTION`, `LOSS_CODE`, `STS_AKTIF` |

Perbedaan satu digit pada baris keempat itu yang paling mudah terlewat, dan ia terbaca
hanya dengan membandingkan kedua procedure berdampingan.

**Cara dipakainya.** Skill ini menuntut **menyilangkan pernyataan dengan kode** sebelum
sebuah istilah diterima. Dua penerapannya:

1. **Kode situs `1` tidak ditebak.** Ia dicocokkan dengan ID penyebab kerugian yang
   dikutip aturan duplikasi klaim PA, `12002` — situs `1` ditambah empat digit pada
   tingkat rincian, konsisten dengan tiga digit pada golongan.
2. **Judul kolom `COL_DESC` ditolak sebagai istilah.** `D-13` menetapkan teks pengguna
   mengikuti Pega, tetapi skill ini menuntut bertanya *apakah ini benar-benar istilah,
   atau nama internal yang bocor*. Jawabannya yang kedua: grid-nya tidak memasang caption
   sama sekali, sehingga judulnya jatuh ke nama properti. Ia ditulis "Keterangan".

### `codebase-design` — menjaga seam tetap di tempat yang sama

**Alasan memakainya.** Modul ini modul master kelima yang dibangun. Nilai terbesarnya
bukan merancang sesuatu yang baru, melainkan **tidak menyimpang** dari yang sudah ada.

**Cara dipakainya.** Prinsip *"satu adapter berarti seam hipotetis, dua adapter berarti
seam nyata"* dipakai sebagai penyaring, dan hasilnya seam yang sama persis dengan modul
sekerabatnya: `Repo` dengan dua pengisi (Oracle dan memori), dipilih per portal lewat
`RepoSelector`.

Satu hal yang **ditambahkan** dan tidak ada di modul lain: `CountPendingJSON`. Ia tidak
melayani satu pun permintaan pengguna — hanya mode periksa. Ia ada karena modul ini punya
pertanyaan yang tidak dimiliki modul lain: *berapa baris yang ditulis penulis kedua*.
Menaruhnya di jalur pengguna akan mencampurkan alat diagnosis dengan fungsi bisnis.

### Skill yang ditimbang dan tidak dipakai

| Skill | Alasan |
|---|---|
| `tdd` | Pola modulnya sudah mapan; uji ditulis bersama kode mengikuti bentuk modul sekerabat, bukan memandu rancangan yang belum diketahui |
| `diagnosing-bugs` | Tidak ada bug yang didiagnosis. Cacat sistem lama dicatat sebagai temuan, bukan didiagnosis |
| `research` | Seluruh fakta ada di dalam repository. Tidak ada satu pun klaim di sesi ini yang bersumber dari luar |
| `prototype` | Tidak ada pertanyaan rancangan yang menuntut prototipe |
| `code-review` | Perubahannya ditulis pada sesi ini; yang menggantikannya adalah suite uji dan uji setara terhadap aplikasi berjalan |

### Catatan untuk sesi berikutnya

- **Jangan menyalin keputusan modul sekerabat tanpa memeriksa pemanggilnya.** Modul ini
  tampak identik dengan Master Status Klaim sampai pemanggil SQL-nya dihitung. Satu kueri
  `grep -rl` menyelamatkan satu kelas cacat yang tidak menimbulkan galat.
- **`npx eslint` tidak dapat dijalankan** — project belum punya `eslint.config.*`, dan
  ESLint 10 tidak lagi membaca `.eslintrc.*`. Gerbang frontend yang benar-benar berjalan
  tetap `tsc --noEmit` dan `vitest`.
- **`curl` masih dijawab proxy** bila `--noproxy '*'` dilupakan. Ini kendala ketiga kali
  berturut-turut.
- **Pohon kerja memuat perubahan sesi lain** (penggantian nama `masterpicteknik` yang
  sedang berjalan). Berkas bersama — `main.go`, `check.go`, `App.tsx`, `registry.ts`,
  `types.ts` — dibaca ulang sesaat sebelum disunting, dan tidak ada yang bentrok.

### Putaran kedua — `domain-modeling` menangkap kesalahan pembacaan sendiri

Work Owner meminta layar diperiksa ulang terhadap Pega dan diselesaikan sesuai layar lama
saja. Putaran kedua ini menemukan **kesalahan pembacaan pada putaran pertama**, dan
skill-nya yang menunjukkan jalannya.

**Apa yang salah.** Putaran pertama menyimpulkan grid Pega tidak memasang caption pada
kolom deskripsi, sehingga judulnya "jatuh ke nama properti `COL_DESC`" — lalu layar baru
menuliskannya **"Keterangan"**, dengan alasan panjang lebar bahwa `COL_DESC` bukan teks
yang dapat dibaca pengguna.

**Kenapa salah.** Pencarian pertama menanyakan `pyCaption`. Captionnya memang ada, hanya
tersimpan dalam bentuk lain: `pyValue` pada sel berpenanda `pyCellHeader=true`. Dua bentuk
berbeda untuk hal yang sama, dan yang kedua terlewat.

**Yang menemukannya** adalah disiplin `domain-modeling` yang menuntut **menyilangkan
pernyataan dengan kode sebelum sebuah istilah diterima**. Alih-alih mencari tag tertentu,
section dibaca **dalam urutan dokumen** — dan "Deskripsi Kerugian" muncul pada baris 2336,
berdampingan dengan `pyLabelFor .COL_DESC`.

**Pelajarannya, dan ia pahit:** putaran pertama membangun *penalaran yang rapi di atas
premis yang tidak diverifikasi*. Panjangnya penjelasan justru membuatnya tampak meyakinkan.
Yang seharusnya memicu kecurigaan adalah pertanyaan sederhana — *"masuk akalkah sebuah
layar produksi menampilkan nama kolom mentah sebagai judul?"* Jawabannya tidak, dan itu
sudah cukup untuk mencari lebih lama sebelum menyimpulkan.

**Lima koreksi lain menyusul dari pembacaan ulang yang sama:** label isian, nama field
JSON, label tombol (`Refresh`, bukan "Muat ulang"), kolom ID lama yang ternyata tidak ada
di grid Pega, dan satu pemberitahuan tambahan yang tidak punya padanan di Pega — dicabut.

**Satu temuan yang membenarkan kode yang sudah ada**, dan tidak akan muncul tanpa membaca
form dalam urutan dokumen: **Pega memuat DUA isian berlabel "Deskripsi Kerugian"**, berbagi
`pyAutomationID` yang sama. Yang kedua mati — `SetCauseOfLossValue_act` tidak pernah
mengisinya, dan view tidak punya kolom untuk membacanya kembali. Modul ini kebetulan sudah
membawa satu isian saja; sesudah temuan ini, ketiadaan yang kedua **dijaga uji** dan
alasannya tercatat.

**Catatan untuk sesi berikutnya:** saat sebuah atribut layar Pega tampak "tidak ada", cari
bentuk penyimpanannya yang lain sebelum menyimpulkan. `pyCaption`, `pyValue` pada sel
header, `pyLabelPreview`, dan `pyLabelFieldValue` sama-sama dapat membawa teks yang dilihat
pengguna.

## Audit ulang Master Masking — `grilling` dipakai pada diri sendiri (2026-09-20)

Work Owner meminta modul dicocokkan ulang ke Pega. Yang dipakai adalah disiplin `grilling`,
tetapi kali ini **sasarannya pekerjaan saya sendiri**, bukan asumsi orang lain.

**Hasilnya: enam cacat.** Yang paling berat dua — pencarian menurut status yang hilang
sama sekali, dan status yang saya keluarkan dari form atas inisiatif sendiri.

**Pola kesalahannya satu, dan layak diingat.** Saya mengumpulkan label layar dengan
`sort -u`, lalu menyimpulkan susunannya. Dua hal rusak sekaligus: **urutan hilang** karena
diurut abjad, dan label yang sama ternyata **muncul di dua tempat berbeda** — grid dan form
— dengan arti yang tidak sama. Kesimpulannya masuk akal dibaca, dan seluruhnya salah.

Yang membetulkannya bukan membaca lebih teliti, melainkan **mengganti alat ukur**: berhenti
memakai daftar label, dan memakai **nomor baris properti yang terikat** sebagai urutan.
Begitu itu dilakukan, susunan grid dan form terbaca langsung tanpa penafsiran.

Ini persis kesalahan yang sama dengan "alat ukur rusak" pada sesi kedua belas (lebar tabel
`.docx` yang angkanya tidak bergerak). Dua kali berturut-turut, sebabnya sama: **alat ukur
dipercaya sebelum divalidasi terhadap satu kasus yang jelas benar.**

**Satu temuan yang hanya muncul karena audit dijalankan:** `Emb_ModulForMaskingData`
hilang dari export (`R-16`). Ia tidak akan pernah terlihat dari membaca tabel atau data —
hanya dari menelusuri `pyInclude` section satu per satu.

**Catatan untuk sesi berikutnya.** Bila sebuah layar Pega dipindahkan, baca
`Section/*.xml`-nya **langsung**, bukan harness-nya: harness hanya membundel salinan section
yang sama, dan nomor barisnya menyesatkan. Urutan kolom diambil dari urutan
`<pyValue>.PROPERTI</pyValue>`, dan `pyInclude` diperiksa satu per satu — di modul ini satu
di antaranya hilang, dan itu mengubah apa yang boleh dijanjikan.

---

## Sesi keenam belas — Modul Master XOL (2026-09-21)

### Ringkasan pemakaian

| Skill | Dipakai? | Kapan | Yang dihasilkan |
|---|---|---|---|
| `mattpocock-skills:grilling` | **ya** | sebelum satu baris kode ditulis | 7 pertanyaan keputusan, dan koreksi atas premis saya sendiri |
| `mattpocock-skills:domain-modeling` | **ya** | saat merancang tipe domain | pemisahan `Amount`/`Share`, penamaan yang menolak alias Pega |
| `mattpocock-skills:codebase-design` | **ya** | saat menetapkan seam | seam `Notifier` dengan dua adapter nyata; `Repo` per portal |
| `mattpocock-skills:tdd` | sebagian | saat menulis uji | dua cacat sendiri tertangkap uji sebelum sampai ke pengguna |
| `mattpocock-skills:code-review` | **ya** | pada akhir, atas diri sendiri | tiga temuan, seluruhnya diperbaiki |

---

### `grilling` — dipakai lebih dulu, dan dipakai pada diri sendiri

**Alasan pemakaian.** Modul ini bertingkat empat dan menyentuh uang, sementara tiga rule
Pega yang menentukan isinya hilang dari export. Menebak salah satu akan mengarang aturan
bisnis, dan `CLAUDE.md` melarangnya tegas.

**Waktu.** Sebelum satu baris kode ditulis, setelah seluruh export dan basis data dibaca.

**Keluaran.** Tujuh pertanyaan dalam dua putaran, seluruhnya dijawab Work Owner.

**Manfaat yang dapat diukur.** Disiplin skill ini menuntut FAKTA dicari sendiri dan hanya
KEPUTUSAN yang ditanyakan. Itu yang membuat pertanyaan nomor 4 dapat diajukan dengan angka
nyata — "Type 2 hanya mengembalikan AVIATION HULL" — bukan sebagai dugaan. Tanpa
menjalankan ketiga penyaring ke basis data lebih dulu, pertanyaannya akan berbunyi "apakah
penyaringnya sudah benar?", yang tidak dapat dijawab siapa pun.

**Dipakai pada diri sendiri.** Dua premis saya sendiri patah di bawah pemeriksaan:

1. Saya menyiapkan pertanyaan tentang "salah ketik `HEAVY EQUPMENT`" sebagai cacat yang
   perlu diperbaiki. Menjalankan kedua ejaan ke ASM membuktikan **hasilnya sama** — nama
   itu tidak ada dalam ejaan mana pun. Pertanyaannya diubah sebelum diajukan.
2. Saya sempat menyimpulkan prasyarat `local.totalshare==100` sebagai bug terbalik.
   Memeriksa kode cabangnya (`true=3` lewati, `false=2` jalankan) membuktikan ia benar.
   Kesimpulan yang salah itu tidak pernah sampai ke Work Owner.

---

### `domain-modeling` — menajamkan bahasa sebelum menulis tipe

**Alasan pemakaian.** Properti Pega di modul ini dipakai untuk hal yang sama sekali berbeda
dari namanya. Membawa namanya berarti membawa kekeliruannya.

**Keluaran.** Empat penamaan ulang yang dicatat beserta buktinya di `masterxol.go`:

```
TempXOL.UserName       → NAMA        nama MASTER, bukan nama pengguna
TempXOL.Amount         → KURSVALUE   KURS, bukan nilai klaim
TempXOL.AreaClaimId    → REMARKPIC   CATATAN PIC, bukan id area klaim
TempXOL.CNPSupportDoc  → TYPEXOL     JENIS XOL, bukan dokumen pendukung
```

**Manfaat.** Skill ini menuntut satuan dinyatakan, bukan diandaikan — dan di modul ini
satuannya memang **berbeda per field**: `ExchangeRate` rupiah, `Limit` dan `Excess` dolar,
`ConvertedLimit` rupiah. Menyatakannya di komentar tipe `Amount` mencegah kelas kesalahan
yang tidak akan tertangkap kompiler: mengalikan dua angka yang keduanya bertipe sama tetapi
bersatuan berbeda.

Pemisahan `Share` dari `Amount` lahir dari sini juga. Keduanya `int64`, tetapi yang satu
persen dan yang lain uang — dan tipe terpisah membuat pertukaran keduanya tidak dapat
lolos diam-diam.

---

### `codebase-design` — seam hanya dibuat bila ada dua adapter nyata

**Alasan pemakaian.** Prinsipnya tegas: *satu adapter berarti seam hipotetis; dua adapter
berarti seam nyata.* Modul ini menggoda untuk membuat seam yang tidak perlu.

**Yang DIBUAT, beserta adapter keduanya:**

| Seam | Adapter 1 | Adapter 2 |
|---|---|---|
| `Repo` | `sqlstore` terhadap Oracle | `memory` untuk uji dan pengembangan |
| `Notifier` | `notification.Sender` lewat SMTP | `notification.Fake` yang merekam |

**Yang TIDAK dibuat.** Tidak ada seam untuk "sumber pilihan grup bisnis" maupun "sumber
daftar tahun", meski keduanya tampak seperti kandidat. Alasannya: masing-masing hanya akan
punya satu adapter, dan keduanya sudah berada di balik `Repo`. Menambah seam di sana akan
menambah lapisan tanpa satu pun hal yang benar-benar bervariasi.

**Manfaat yang dapat diukur.** Adapter kedua `Notifier` bukan sekadar pelengkap uji: ia
yang membuat modul tetap dapat dipakai ketika SMTP belum dikonfigurasi. Menyimpan Master XOL
tetap berhasil, pemberitahuannya tercatat, dan penyimpanan yang sebenarnya sudah selesai
tidak digagalkan oleh surel yang tidak terkirim.

---

### `tdd` — dua cacat sendiri tertangkap sebelum sampai ke pengguna

**Alasan pemakaian.** Uji ditulis menyatakan ATURAN, bukan fungsi, mengikuti
`14-TESTING-STRATEGY.md` §3.2 — sehingga daftar uji terbaca sebagai dokumentasi aturan.

**Keluaran.** 30 uji domain, 21 uji usecase, 24 uji transport, 12 uji layar.

**Manfaat yang dapat diukur — keduanya nyata, bukan hipotetis:**

| Uji yang gagal | Cacat yang ditemukan |
|---|---|
| `ParseAmount("13500.75")` | Fungsi itu **ambigu**: titik dibuang sebagai pemisah ribuan, sehingga pecahan lolos sebagai `1350075`. Fungsinya dibuang |
| `limit_idr` pada detail | Repo memori **tidak merapikan saat membaca**, sehingga berperilaku berbeda dari sqlstore |

Yang kedua paling berharga: tanpa uji itu, penyimpanan memori akan tampak benar selama
pengembangan lalu berbeda dari Oracle — dan selisihnya baru terlihat di staging.

---

### `code-review` — dipakai pada diri sendiri di akhir

**Alasan pemakaian.** `D-46` menetapkan agen me-review pekerjaannya sendiri, dengan
kewajiban menjalankan pemeriksaan yang benar-benar dijalankan — bukan penilaian kualitatif.

**Tiga temuan, seluruhnya diperbaiki:**

1. **Komentar berjanji lebih dari yang dikerjakan kode.** `checkXOL` menyebut pemeriksaan
   baris yatim di doc comment-nya tetapi tidak mengerjakannya. Diperbaiki dengan
   **mengerjakannya**, bukan memangkas komentarnya — dan hasilnya menemukan 5 baris yatim
   nyata di produksi.
2. **Melanggar aturan sendiri.** Kueri `xol_orphan_count` versi pertama memakai `ROWNUM = 1`,
   yang justru dilarang uji disiplin SQL yang ditulis beberapa menit sebelumnya. Diganti
   `UNION ALL` atas tiga agregat.
3. **Kode mati.** `DeleteBusinessWithoutID` beserta kuerinya ditulis, lalu tidak dipanggil
   dari mana pun setelah layar memutuskan mematikan tombolnya. Dibuang.

**Catatan untuk sesi berikutnya.** Temuan 1 dan 2 punya pola yang sama: keduanya muncul
pada bagian yang ditulis **paling akhir**, setelah perhatian berpindah dari modul ke
perkakas pemeriksanya. Bagian yang ditulis terakhir layak ditinjau dengan kecurigaan yang
sama besar dengan bagian intinya.

---

### Skill yang tersedia tetapi tidak dipakai

| Skill | Alasan |
|---|---|
| `prototype` | Tidak ada pertanyaan desain yang butuh prototipe; seluruhnya terjawab dari export dan basis data |
| `diagnosing-bugs` | Cacat yang ditemukan berasal dari SISTEM LAMA dan dicatat, bukan didiagnosis untuk diperbaiki di sana |
| `research` | Seluruh fakta ada di dalam repository dan di basis data ASM; tidak ada klaim yang bersumber dari luar |
| `resolving-merge-conflicts` | Tidak ada konflik merge |
| `writing-for-agents` | Dokumen sesi ini ditujukan untuk dibaca manusia |


---

# Penggunaan Skill — Sesi 2026-09-21 (Master Dokumen Travel)

## Ringkasan

**Tidak ada satu pun skill yang dipanggil pada sesi ini.** Yang dipakai adalah disiplin dan
kosakatanya, sama seperti empat sesi sebelumnya — dicatat di bawah apa adanya, bukan diklaim
sebagai pemanggilan skill.

Satu perkakas dibuat sendiri dan dijalankan: pemindai token Go untuk penggantian nama
`masterpicteknik`.

## Skill yang ditimbang

| Skill | Kenapa masuk akal ditimbang | Kenapa tidak dipakai |
|---|---|---|
| `mattpocock-skills:grilling` | Tiga hal tidak dapat disimpulkan dari data: izin memperbaiki build, basis data mana, dan validasi | Disiplinnya dipakai tanpa memanggil skill-nya: tiga pertanyaan diajukan sekaligus, masing-masing dengan pilihan jawaban, rekomendasi, dan **akibat yang diterima bila memilihnya**. Satu jawaban ("seperti aplikasi Pega") tidak menjawab pertanyaan izin, dan pertanyaannya **diajukan ulang** alih-alih ditebak — karena jawabannya menentukan apakah saya boleh menyentuh modul yang masuk Isolasi Protektif |
| `mattpocock-skills:codebase-design` | Modul baru, seam baru, dan satu keputusan batas: portal utama atau per entitas | Kosakatanya dipakai tanpa memanggil skill-nya. Pertanyaan yang menentukan bukan "di mana seam-nya" melainkan **siapa pemilik datanya** — dan jawabannya mengikuti `D-75`: daftar dokumen Travel milik satu badan hukum, bukan milik empat |
| `mattpocock-skills:domain-modeling` | Istilah baru masuk domain: `TravelDocument`, `DOCID`, `NAMADOKUMEN` | Artinya sudah ditetapkan tabel dan procedure-nya, bukan hasil perundingan istilah. Yang perlu dikerjakan adalah **membedakannya dari master detail** `V_LST_DOC_TRAVEL` yang mudah tertukar — dan itu dikerjakan dengan menulis batasnya di tiga tempat, bukan dengan menajamkan arti |
| `mattpocock-skills:tdd` | Perilaku baru | Uji ditulis berbarengan, bukan lebih dulu. Perilakunya dibaca DARI PROCEDURE yang ada; menulis uji sebelum `DOCTRAVEL_CVG.prc` dibaca berarti menguji aturan yang saya karang sendiri |
| `mattpocock-skills:research` | — | Seluruh fakta ada di dalam repository, termasuk source procedure-nya |
| `mattpocock-skills:code-review` | — | Yang menjaga sesi ini adalah kompilator, `go vet`, 30 paket uji Go, dan 82 uji frontend — dijalankan pada setiap langkah |

## Perkakas yang dibuat sendiri, dan kenapa `sed` tidak memadai

§14.3 sudah menyebut alasannya, tetapi perkakasnya tidak pernah di-commit sehingga dibuat ulang.
Kali ini di atas **`go/scanner` pustaka standar**, bukan pemindai buatan sendiri — dan itu lebih
kuat, karena pemecahan kode/bukan-kode-nya adalah pemecahan yang dipakai kompilator Go sendiri.

| Yang dilindungi | Kenapa penting di sesi ini |
|---|---|
| Komentar (`token.COMMENT`) | `peta-penamaan.md` menetapkan komentar TETAP Indonesia. `sed` akan menerjemahkannya diam-diam |
| Literal string (`token.STRING`) | **Tag JSON adalah literal string.** `sed` yang mengganti `Nama`→`Name` akan mengubah `json:"nama"` menjadi `json:"name"` — dan itu **merusak kontrak API tanpa satu pun galat kompilasi** |
| Nama kueri `.sql` | Ia juga literal, sehingga **tidak** ikut terganti — dan itu justru benar: penanda `-- name:` di berkas `.sql` harus berubah berbarengan dengan pemanggilnya, jadi dikerjakan terpisah dan sengaja |

Dijalankan dua kali: `collect` mengeluarkan seluruh identifier yang benar-benar ada, barulah
petanya disusun dari daftar itu. Menyusunnya dari ingatan akan melewatkan identifier, dan yang
terlewat belum tentu terdeteksi kompilator.

Hasil: **788 identifier di 11 berkas**, nol galat kompilasi, nol uji yang rusak.

## Teknik yang dipakai tanpa memanggil skill

| Teknik | Manfaat nyata |
|---|---|
| **Ukur baseline sebelum menyentuh apa pun** | `go build` dan `npm test` dijalankan SEBELUM pekerjaan dimulai. Itu yang membuat "3 uji gagal" di akhir sesi dapat dinyatakan sebagai kegagalan lama dengan yakin, bukan sebagai kerusakan yang mungkin saya buat |
| **Baca procedure-nya, jangan simpulkan dari pemanggilnya** | `DOCTRAVEL_CVG.prc` mengungkap tiga cacat yang tidak terlihat dari sisi Pega — termasuk parameter keluaran yang **tertukar**, sehingga catatan "Data Sudah Disimpan dengan ID …" selama ini mendarat di kolom judul dokumen dan `pyNote` selalu kosong |
| **Periksa tetangga yang namanya mirip** | `M_DOCTRAVEL` (MENU_ID 22) dan `V_LST_DOC_TRAVEL` (MENU_ID 39) hanya berbeda satu kata di menu. Memeriksa keduanya lebih dulu mencegah modul ini terlanjur memuat kolom `MINUNGGAH` dan `STSWAJIB` yang bukan miliknya |
| **Ikuti modul terdekat, bukan modul terakhir** | Bentuk datanya paling mirip Master Status Klaim (dua kolom, kode dari situs+urutan), tetapi kepemilikan datanya paling mirip Master Status Progres (per entitas). Keduanya diambil dari modul yang berbeda alih-alih menyalin satu modul bulat-bulat |
| **Kunci keputusan dengan uji, bukan dengan komentar** | "Tanpa validasi" mudah hilang saat orang berikutnya menambahkan `min(1)` karena tampak seperti kelalaian. Dua uji bernama `...LikeInPega` membuat penambahan itu gagal lebih dulu, sehingga menjadi keputusan yang disadari |

## Kesalahan sendiri yang tercatat sesi ini

**Satu, dan ia soal alat ukur — lagi.**

Saya melaporkan dua konflik frontend, lalu beberapa menit kemudian `grep` yang sama hanya
menemukan satu. Dugaan pertama saya adalah hasil grep yang basi. Yang benar: **Work Owner sedang
menyunting berkasnya bersamaan dengan saya bekerja**.

Yang saya lakukan sesudahnya, dan inilah bagian yang layak dipertahankan: saya **tidak menimpa
berkasnya**. `git diff` dijalankan lebih dulu untuk melihat resolusi mana yang sudah diambil,
hasilnya diperiksa (ia mengambil sisi yang sama dengan yang saya rekomendasikan), lalu pekerjaan
saya dilanjutkan di atasnya.

> Pelajarannya, sebagai pelengkap catatan sesi-sesi sebelumnya: **repo yang sedang dikerjakan
> orang lain bukan repo yang diam.** Sebelum menulis ke berkas yang pernah dibaca beberapa langkah
> sebelumnya, periksa apakah ia masih seperti yang diingat.

Hal yang sama terjadi pada modul `mastercolsimasonline`, yang muncul di tengah sesi. Ia diperiksa,
tidak bertabrakan, dan rutenya kini berdampingan dengan rute modul ini di `App.tsx`.

## Catatan untuk sesi berikutnya

- **Perkakas penggantian nama masih belum di-commit.** Ia dibuat dua kali sekarang — §14.3 dan
  sesi ini. Bila penggantian nama ketiga dibutuhkan, pertimbangkan menaruhnya di `claim-pnc/tools/`
  alih-alih membuatnya ulang.
- **Merge dengan penggantian nama massal sudah dua kali meninggalkan berkas lama yang bertahan.**
  Pemeriksaan yang murah dan menangkap keduanya: `go build ./...` dan
  `grep -rn "^<<<<<<<" src backend` segera setelah merge, sebelum di-commit.
- **`CONTEXT.md` belum memuat istilah `Dokumen Travel`.** Sama seperti catatan sesi sebelumnya
  tentang `Menu` dan `Otorisasi`: ia kini punya arti tepat di sistem ini, tetapi `CONTEXT.md`
  adalah dokumen Steering yang perubahannya ditulis sebagai keputusan.

---

# Penggunaan Skill — Sesi 2026-09-21 (Master COL Simas Online)

## Ringkasan

| Skill | Dipakai | Kapan |
|---|---|---|
| `mattpocock-skills:grilling` | **ya** | sebelum satu berkas pun ditulis — tiga pertanyaan ke Work Owner |
| `mattpocock-skills:domain-modeling` | **ya** | saat menamai `CauseOfLoss`, `Business`, `MasterCode` |
| `mattpocock-skills:codebase-design` | **ya** | saat memutuskan `BusinessRepo` menjadi seam tersendiri |
| `mattpocock-skills:tdd` | tidak | uji ditulis setelah perilakunya jelas, bukan lebih dulu |
| `mattpocock-skills:research` | tidak | seluruh fakta ada di dalam repo |

## `grilling` — tiga pertanyaan, dan kenapa ketiganya layak ditanyakan

Disiplin skill ini: **cari faktanya sendiri, serahkan keputusannya**. Yang dicari sendiri: bentuk
layar, tabel sebenarnya, cara kode diterbitkan, dan bukti bahwa isian Bisnis berupa PageList.
Yang **tidak** boleh diputuskan sendiri, dan karena itu ditanyakan:

| Pertanyaan | Kenapa bukan keputusan saya |
|---|---|
| Sejauh apa sisa merge boleh dibereskan | Menyentuh modul yang dinyatakan selesai; Isolasi Protektif melarangnya tanpa izin |
| Lingkup isian Bisnis | Membangunnya penuh melipatkan pekerjaan; menyederhanakannya **kehilangan data** |
| Penyimpanan JSON atau kolom | Menyentuh objek basis data produksi dan menempuh `D-63` |

**Hasilnya nyata, bukan formalitas.** Jawaban ketiga mengubah bentuk seluruh modul: kalau saya
memutuskan sendiri, saya akan mereplikasi dokumen JSON supaya "setara Pega" — dan itu menyalin
kembali persis bentuk penyimpanan yang `D-02` dan migrasi `0002` justru tinggalkan.

**Rekomendasi saya disertakan di setiap pertanyaan**, sesuai format skill. Pada pertanyaan kedua
dan ketiga jawabannya **berbeda dari rekomendasi saya** — saya mengusulkan repo memori dulu dengan
SQL bertanda "belum terverifikasi"; Work Owner memilih yang lebih tegas.

## `domain-modeling` — istilah yang ditajamkan

| Nama Pega | Nama di kode | Alasan |
|---|---|---|
| `M_COL_ID` | `Code` | ia kode, bukan "ID" generik — sejajar `ClaimStatus.Code` |
| `COL_DESC` | `Description` | isi kolomnya memang keterangan |
| `MST_COL_ID` | `MasterCode` | "ID Master Kerugian" — kunci padanan di sistem sebelah, bukan kunci baris ini |
| `BISNISID` | `Businesses []Business` | jamak, karena buktinya PageList |
| `NOTE` | `Business.Name` | `Note` menyesatkan: isinya nama bisnis, bukan catatan |

Penajaman terakhir yang paling berguna: kueri lama mengaliaskannya `NOTE as "Note"`, dan kalau
alias itu dibawa, seluruh kode akan menyebut nama bisnis sebagai "catatan" — persis kelas kesalahan
yang `D-19` cegah.

**Tiga istilah ini belum ada di `CONTEXT.md`** — Master COL Simas Online, ID Master Kerugian,
Bisnis. Sama seperti catatan sesi sebelumnya tentang `Dokumen Travel`: `CONTEXT.md` dokumen
Steering, perubahannya ditulis sebagai keputusan, bukan disunting diam-diam.

## `codebase-design` — satu seam dipecah dua

Prinsip yang dipakai: **seam nyata bila ada dua adapter, dan bila yang bervariasi memang berbeda.**

`BusinessRepo` dipisahkan dari `Repo` meski keduanya **selalu** dipilih bersamaan dan selalu di
atas koneksi yang sama. Alasannya bukan kerapian: yang berbeda adalah **kepemilikannya**.
`M_CAUSE_OF_LOSS` ditulis modul ini; `POOLDATA.BUSINESS` milik GISFW dan hanya dibaca. Dengan
memisahkannya, "hanya dibaca" menjadi sifat **tipe** — antarmukanya memang tidak punya operasi
tulis — bukan aturan yang harus diingat.

Uji deletion-nya: kalau `BusinessRepo` digabung ke `Repo`, tidak ada apa pun yang menghalangi
seseorang menambahkan `InsertBusiness` enam bulan lagi, dan tidak ada yang akan menangkapnya.

## Teknik yang dipakai tanpa memanggil skill

**Menelusuri dari harness ke rule yang dirujuknya, bukan membaca harness-nya.**
`CauseOfLossInboxSimasOnline-Harness.xml` 289 KB dan nyaris seluruhnya boilerplate Pega.
Yang berguna: daftar `pyRuleName` di dalamnya, yang menunjuk dua section, dua activity, dan satu
data transform. Caption layar pun terbaca dari situ (`pyCaption Bisnis`,
`pyCaption ID Master Kerugian`) tanpa perlu membuka section-nya.

**Mencari bentuk tabel dari kueri yang memakainya, bukan dari DDL yang tidak ada.**
`RDB List/GetLBUID_SQL-SQL.xml` — satu baris SQL — yang memberi tahu bahwa pemetaan COL ke Bisnis
adalah tabel dua kolom `(D_COL_ID, BISNISID)` di-join ke `BUSINESS (ID, NOTE)`. Tanpa itu, bentuk
tabel `M_CAUSE_OF_LOSS_BUSINESS` akan murni karangan.

**Memakai modul yang sudah ada sebagai preseden, bukan menyusun pola baru.**
Master Status Klaim sudah menempuh perpindahan JSON ke kolom, lengkap dengan migrasi, penerbitan
kode dari situs + urutan, dan perlakuan kolom JSON lamanya. Migrasi `0004` mengikutinya langkah
demi langkah — termasuk peringatan "baca katalog dulu" yang lahir dari kesalahan nyata pada `0002`.

## Kesalahan sendiri yang tercatat sesi ini

| # | Kesalahan | Bagaimana ketahuan |
|---|---|---|
| 1 | Melaporkan baseline uji frontend "hijau" padahal ia berhenti pada 120 detik dengan `Failed to start forks worker` — bukan uji yang benar-benar lulus | Saat suite penuh akhirnya berjalan dan menunjukkan 3 merah, saya membuka berkas keluaran baseline-nya dan melihat sebab yang sebenarnya. **Exit code 0 bukan bukti uji lulus** |
| 2 | Menulis `useFieldArray({ control: undefined, ... } as never)` — `as never` untuk memaksa tipe | Ditangkap saat membacanya kembali sebelum `tsc`. `as never` adalah tanda tipe sedang dilawan, bukan dipakai; `control` memang tersedia dari `useForm` |
| 3 | Menyimpulkan bahwa kegagalan 3 uji `AccountPage` "pre-existing" dari exit code baseline | Diperbaiki dengan pembuktian sungguhan: menjalankan uji terhadap `AccountForm.tsx` versi `origin/master`. Ketiganya tetap merah — **baru** itu bukti |

Ketiganya punya pola yang sama dengan yang dicatat sesi-sesi sebelumnya: **alat ukur dipercaya
sebelum divalidasi.** Yang ketiga paling berbahaya, karena kesimpulannya kebetulan benar — dan
kesimpulan benar yang berdiri di atas bukti yang salah tetap kebiasaan yang buruk.

## Catatan untuk sesi berikutnya

- **Dua sesi berjalan bersamaan di repo yang sama.** `go build ./...` dapat merah karena pekerjaan
  sesi lain. Persempit ke `go build ./internal/...` untuk memverifikasi modul sendiri, dan periksa
  ulang seluruhnya di akhir.
- **Periksa nomor migrasi terakhir tepat sebelum membuat berkasnya**, bukan di awal sesi.
  `0003` sempat kosong saat sesi ini mulai, dan sudah terpakai satu jam kemudian.
- **Bila layar Pega punya grid di dalam form**, periksa `pyPageListProperty` lebih dulu. Satu baris
  XML itu yang membedakan dropdown dari grid banyak baris, dan salah membacanya berarti kehilangan
  data tanpa galat.
- **Berkas Go baru sesi ini ditulis LF**, sehingga `gofmt -l` atasnya kosong. Sesi §17 memilih CRLF
  agar seragam dengan 122 berkas lama. Keduanya sah, tetapi repo kini memuat dua konvensi —
  `.gitattributes` yang menetapkan salah satunya akan menutup ini sekali untuk selamanya.

---

# Penggunaan Skill — Sesi 2026-09-21, bagian kedua (pemeriksaan ulang Master COL)

## Ringkasan

| Skill | Dipakai | Kapan |
|---|---|---|
| `mattpocock-skills:grilling` | **ya, dan inilah intinya** | seluruh bagian sesi ini |
| `mattpocock-skills:domain-modeling` | **ya** | saat arti `MST_COL_ID` berubah |
| `mattpocock-skills:codebase-design` | **ya** | saat memutuskan `ComboField` menjadi komponen bersama |

## `grilling` — urutannya yang membuat perbedaan

Permintaan Work Owner adalah "tanyakan hal yang masih janggal" **dan** "cek lagi seperti aplikasi
Pega". Urutan mengerjakannya menentukan hasilnya, dan disiplin skill ini menetapkan urutan yang
benar: **cari faktanya sendiri lebih dulu, baru ajukan keputusannya.**

Kalau pertanyaannya disusun lebih dulu, ia akan berisi hal-hal yang saya ragukan — dan saya tidak
meragukan `MST_COL_ID`, karena saya sudah yakin ia kode dari sistem sebelah. Pemeriksaan ulang yang
memunculkannya.

**Hasilnya terukur:** empat pertanyaan diajukan, **keempatnya** berujung pada perubahan kode. Tidak
ada satu pun yang sekadar menegaskan yang sudah ada.

| Pertanyaan | Kalau tidak ditanyakan |
|---|---|
| `MST_COL_ID` | kolomnya dibangun sebagai teks bebas — arti datanya salah, dan jenjang induk-anak tidak pernah ada |
| Label & urutan | dua selisih `D-13` yang tidak tercatat |
| Judul form | selisih kecil, tetapi tetap selisih |
| Bisnis bebas ketik | **penyimpanan menolak baris yang Pega terima** — dan migrasi akan membuang baris lama yang tidak punya ID |

Yang terakhir paling berbahaya: cacatnya baru terlihat saat data produksi masuk.

**Rekomendasi saya disertakan di setiap pertanyaan**, sesuai format skill. Pada pertanyaan keempat
jawabannya **berbeda dari rekomendasi saya** — saya mengusulkan menolak bisnis di luar master
sebagai perbaikan; Work Owner memilih kesetaraan dengan Pega. Jawaban itu benar, dan alasannya
sekarang jelas: `P-5` menetapkan perilaku dipertahankan lebih dulu, dan "memperbaiki" isian yang
petugas memang pakai untuk mencatat bisnis yang belum masuk master akan menghalangi pekerjaan
mereka.

## `domain-modeling` — satu istilah yang artinya berbalik

`MasterCode` semula saya beri arti "kunci padanan di sistem Simas Online". Setelah bukti dibaca, ia
ternyata "Code baris lain di master yang sama".

Perbedaannya bukan kosmetik — ia mengubah **apa yang boleh diisi**, **apa yang harus divalidasi**,
dan **bentuk isian di layar**. Nama tipenya kebetulan tetap cocok (`MasterCode` tetap terbaca
benar), tetapi seluruh komentarnya harus ditulis ulang, dan itu dikerjakan.

Satu istilah lain ditajamkan: `Business.Name` semula didokumentasikan sebagai "BUSINESS.NOTE".
Setelah koreksi keempat, ia menjadi "nama bisnis — BUSINESS.NOTE bila dipilih dari master, atau apa
yang diketik petugas bila tidak". Perbedaannya menentukan: yang pertama menyiratkan ia selalu
berasal dari master, dan itu tidak benar.

## `codebase-design` — kapan komponen menjadi milik bersama

`ComboField` ditaruh di `src/components/`, bukan di dalam modul.

Ujinya bukan "apakah ia dipakai lebih dari sekali sekarang" — sekarang ia dipakai sekali. Ujinya
adalah **apakah ia menjawab kebutuhan yang berulang**: isian "bebas ketik dengan saran" adalah
bentuk yang muncul di banyak layar Pega, dan `08-TECHNICAL-STRATEGY.md` §5 menuntut seluruh isian
memakai komponen baku. Menaruhnya di dalam modul berarti layar berikutnya akan menyalinnya.

Yang dijaga: berkasnya **baru**, tidak menyunting komponen yang sudah ada. Menambah ke pustaka
bersama aman terhadap Isolasi Protektif; mengubah yang sudah ada tidak.

## Teknik yang dipakai tanpa memanggil skill

**Membaca bukti dari indeks rule, bukan dari layout.** Daftar `pyRuleName` di ekor berkas section
memuat seluruh caption yang dipakai. Dari situ ketahuan ada **lima** caption padahal saya memetakan
empat — dan caption kelima ("ID Kerugian") yang membongkar kekeliruan label.

**Mencari batas sel lewat `pyCellId`, bukan lewat kedekatan baris.** Kontrol daftar pada
`MST_COL_ID` berjarak ~60 baris dari nilainya. Menebak dari kedekatan akan salah; yang memastikan
adalah `pyCellId 378` yang membungkus keduanya.

**Menjalankan aplikasinya sungguhan untuk membuktikan perilaku, bukan hanya uji.** Enam perilaku
baru ditembak lewat HTTP — termasuk `"  fire / property  "` yang harus kembali sebagai
`FIRE / PROPERTY` dengan ID `006`. Uji unit membuktikan logikanya; permintaan sungguhan membuktikan
rakitannya.

## Kesalahan sendiri yang tercatat sesi ini

| # | Kesalahan | Bagaimana ketahuan |
|---|---|---|
| 1 | **Menyimpulkan arti `MST_COL_ID` dari namanya.** "ID Master Kerugian" terbaca seperti kode dari sistem lain, dan saya berhenti di situ | Pemeriksaan ulang menemukan sumber daftarnya. Ini pengulangan pola yang `R-06` sudah catat: **arti kolom tidak boleh disimpulkan dari namanya maupun dari pemakaiannya** |
| 2 | **Mengurutkan pemetaan bisnis menurut `BISNISID`** sementara domainnya menjanjikan "urutan dipertahankan apa adanya" | Terlihat saat koreksi keempat membuat sebagian `BISNISID` kosong. Cacat itu **sudah ada** sebelumnya — adapter memori mempertahankan urutan, adapter SQL tidak, dan tidak satu pun uji menangkapnya |
| 3 | **Menulis `as never` untuk memaksa tipe** pada `useFieldArray` | Ditangkap saat membaca ulang. `as never` adalah tanda tipe sedang dilawan |

Kesalahan kedua paling layak diingat: **dua adapter yang menjanjikan hal sama tetapi berperilaku
beda tidak akan tertangkap uji yang hanya menjalankan salah satunya.** Uji kueri sekarang
menegakkannya (`TestBusinessOrderComesFromItsOwnColumn`), tetapi itu tambalan — yang benar adalah
satu uji kontrak yang dijalankan terhadap kedua adapter.

## Catatan untuk sesi berikutnya

- **Sebelum menetapkan arti sebuah kolom, cari kontrol yang mengisinya di layar.** Satu blok
  `pyListSource`/`pySourceName` memberi tahu lebih banyak daripada nama kolom dan nama caption
  digabung. Untuk modul ini, satu blok itu yang membedakan "teks bebas" dari "rujukan ke tabel
  sendiri".
- **Periksa `pyAllowFreeFormInput` pada setiap autocomplete.** Ia menentukan apakah nilainya
  terbatas pada master — dan karenanya menentukan apakah kolomnya boleh NULL, apakah kunci asing
  boleh dipasang, dan apa yang menjadi kunci baris.
- **Uji kontrak yang sama untuk adapter memori dan adapter SQL belum ada di repo ini.** Setiap
  modul menulis uji terpisah untuk masing-masing, dan perbedaan perilaku di antara keduanya lolos.
  Ia pekerjaan tersendiri yang layak diusulkan.

---

# Penggunaan Skill — Sesi 2026-09-21, bagian ketiga (menyamakan dengan Pega)

## Ringkasan

| Skill | Dipakai | Kapan |
|---|---|---|
| `mattpocock-skills:grilling` | **ya** | seluruh bagian ini — tetapi kali ini menggrill DIRI SENDIRI |
| `mattpocock-skills:codebase-design` | **ya** | saat kunci baris pemetaan harus dipindahkan |

## `grilling` dipakai terbalik: menguji premis sendiri, bukan premis pengguna

Bagian sebelumnya menghasilkan empat "selisih terencana". Instruksi Work Owner — *"cek lagi pada
Pega, samain ja"* — bukan meminta pertanyaan baru, melainkan meminta **premis saya sendiri diuji**.

Disiplin yang dipakai sama: **jangan terima klaim tanpa bukti, termasuk klaim sendiri.** Setiap
dari empat penyimpangan punya alasan yang saya tulis dengan yakin, dan masing-masing diuji ulang:

| Klaim saya | Cara mengujinya | Hasil |
|---|---|---|
| "Pega membolehkan nama kosong, dan itu cacat" | hitung `pyRequired`, cari Validate rule, baca activity simpan | **benar bahwa Pega membolehkan** — tetapi "cacat" adalah penilaian saya, bukan keputusan |
| "Bisnis kembar tidak bermakna" | cari penanda keunikan di grid | **nol** — Pega memang mengizinkan |
| "Rujukan-diri membentuk lingkaran yang berputar tanpa henti" | cari siapa membaca `MST_COL_ID` | **TERBANTAH** — nol penelusuran di seluruh export dan seluruh procedure |

Yang ketiga yang paling penting: **alasan teknis saya salah, bukan sekadar tidak disetujui.** Saya
membayangkan penelusuran jenjang yang tidak pernah ada, lalu membangun aturan untuk melindunginya.
Satu perintah `grep` membantahnya.

## `codebase-design` — ketika satu keputusan bisnis membatalkan rancangan penyimpanan

Mengizinkan bisnis kembar tampak seperti perubahan kecil: buang satu pemeriksaan. Ternyata ia
membatalkan **kunci baris** tabel pemetaan.

Cara memutuskannya: daftarkan seluruh kandidat kunci, lalu gugurkan dengan bukti — bukan dengan
selera.

| Kandidat | Gugur karena |
|---|---|
| `BISNISID` | boleh NULL (`pyAllowFreeFormInput=true`) |
| `NAMA_BISNIS` | tidak unik (kembar diizinkan) |
| `URUTAN` | — |

Yang tersisa ternyata juga yang paling setia pada sumbernya: `TempCauseOfLoss.BISNISID` di Pega
adalah **PageList**, dan baris PageList memang dikenali lewat nomor urutnya. Kunci yang benar
bukan hasil kompromi — ia yang sejak awal dipakai sistem lama, hanya tidak terlihat sampai dua
kandidat lain gugur.

**Uji deletion pada `SameBusiness`:** setelah kembar diizinkan, fungsi itu kehilangan satu-satunya
alasan keberadaannya. Tetapi ia tidak dihapus — ia **berganti peran** menjadi
`NormalizeBusinessName`, karena pencocokan nama ke master tetap butuh normalisasi yang sama di dua
lapisan. Mengganti namanya penting: nama lama menyiratkan "deteksi kembar" yang sudah tidak
dikerjakannya.

## Teknik yang dipakai tanpa memanggil skill

**Membuktikan ketiadaan, bukan hanya keberadaan.** Tiga dari empat pemeriksaan adalah pembuktian
NEGATIF — bahwa Pega TIDAK memvalidasi. Itu lebih sulit daripada membuktikan sesuatu ada, dan
caranya adalah menghitung seluruh kemunculan lalu menunjukkan angkanya nol:

```
pyRequired          -> 19 kemunculan, SELURUHNYA false
Page-Validate       -> nol
direktori Validate/ -> tidak ada
penanda unique      -> nol di seluruh repeat grid
filter pada RD      -> nol
MST_COL_ID di .prc  -> nol
```

**Menjalankan aplikasinya untuk membuktikan pencabutan, bukan hanya penambahan.** Keempat perilaku
yang dicabut ditembak lewat HTTP dan harus **berhasil** — `201` untuk nama kosong, `201` untuk
bisnis kembar, `200` untuk rujukan-diri. Uji yang membuktikan sesuatu TIDAK lagi ditolak mudah
terlewat kalau hanya mengandalkan uji unit.

## Kesalahan sendiri yang tercatat sesi ini

| # | Kesalahan | Pelajaran |
|---|---|---|
| 1 | **Membangun empat aturan yang tidak ada di Pega dan menyebutnya "perbaikan terencana"** | `P-5` menetapkan perilaku dipertahankan lebih dulu, dan `D-49` sudah memutuskan 13 butir perbaikan — tidak satu pun menyangkut layar ini. **"Ini jelas lebih baik" bukan dasar yang cukup untuk menyimpang**; dasarnya keputusan tertulis |
| 2 | **Menulis alasan teknis untuk rujukan-diri tanpa memeriksanya** | "membentuk lingkaran yang berputar tanpa henti" terdengar meyakinkan dan ternyata salah. Satu `grep` sudah cukup membantahnya, dan saya tidak menjalankannya sebelum menulis |
| 3 | **Kunci tabel dirancang sebelum seluruh aturan bisnisnya pasti** | rancangan bagian kedua (kunci = nama) gugur seluruhnya oleh satu keputusan di bagian ketiga. Bila keempat pertanyaan diajukan sebelum tabel dirancang, satu rancangan sudah cukup |

Ketiganya punya akar yang sama: **bertindak atas penalaran yang belum diuji.** Yang pertama
penalaran tentang apa yang seharusnya; yang kedua tentang apa yang berbahaya; yang ketiga tentang
apa yang sudah pasti.

## Catatan untuk sesi berikutnya

- **Sebelum menulis "perbaikan terencana", periksa `D-49`.** Ke-13 butirnya adalah daftar tertutup.
  Yang di luar itu menuntut persetujuan tertulis (`D-54`) — dan pengalaman sesi ini menunjukkan
  persetujuan itu sering tidak diberikan, sehingga menyimpang lebih dulu berarti mengerjakan dua
  kali.
- **Ajukan SELURUH pertanyaan perilaku sebelum merancang penyimpanan.** Satu jawaban dapat
  membatalkan kunci tabel, dan kunci tabel menyentuh domain, repo, migrasi, dan uji sekaligus.
- **Untuk membuktikan Pega tidak melakukan sesuatu, hitung kemunculannya dan tunjukkan nol.**
  Pernyataan "tidak ada validasi" tanpa angka tidak dapat diperiksa siapa pun, termasuk oleh saya
  sendiri di sesi berikutnya.

---

# Sesi kesebelas — modul Daftar Tipe Dokumen (2026-09-21)

## `grilling` — dipakai untuk menolak jawaban yang menjawab pertanyaan lain

**Kapan.** Sebelum satu baris kode ditulis, setelah pembacaan harness selesai.

**Keluaran.** Lima pertanyaan, empat terjawab langsung. Yang kelima memperlihatkan gunanya skill ini
paling jelas: pertanyaan **"apakah datanya dipisah per portal entitas"** dijawab *"tidak memakai JSON
lagi, menyimpan pada basis langsung sesuai PEGA"* — jawaban yang benar dan penting, tetapi untuk
**pertanyaan yang berbeda**.

Disiplin grilling menahan dorongan untuk menerimanya sebagai jawaban portal. Yang dilakukan: mencatat
jawaban itu sebagai keputusan tersendiri (bentuk penyimpanan), lalu **menanyakan ulang** yang portal
dengan pembuka yang menyatakan apa yang sudah tercatat, supaya Work Owner tidak merasa ditanya dua
kali hal yang sama.

**Manfaat.** Bila jawaban itu diterima apa adanya, modul akan dibangun **tanpa `RepoSelector`** —
satu koneksi untuk empat badan hukum. Itu persis kelas cacat yang `R-20` sebut: layar tampil normal,
angkanya masuk akal, dan yang salah hanya *milik siapa* data itu.

## `research` — dipakai setelah Work Owner menolak memilih

Pertanyaan portal yang kedua dijawab *"tolong dicek lagi seperti aplikasi PEGA"*. Itu bukan penolakan
menjawab, melainkan **pengalihan ke bukti** — dan jawabannya memang ada di sana.

**Cara memeriksanya.** Bukan mencari kata "portal" atau "entitas", yang tidak ada di sistem lama.
Yang dicari adalah **mekanisme penerbitan ID**: `PEGA_LST_DOC_TYPE.prc:12` membaca `M_SITE_DATABASE`
untuk mendapat kode situs. Lalu dibandingkan dengan dua procedure master yang lingkup portalnya sudah
diputuskan — `DOCTRAVEL_CVG.prc:12` dan `PEGA_M_CAUSE_OF_LOSS.prc:11` — dan ketiganya identik.

**Manfaat.** Keputusan lingkup portal berdiri di atas bukti yang dapat ditunjuk barisnya, bukan di
atas kemiripan nama modul. Pola yang layak diulang: **ketika lingkup data dipertanyakan, periksa dari
mana kuncinya diterbitkan** — kunci yang lahir dari kode situs basis data adalah kunci yang berumah
di basis data itu.

## `domain-modeling` — satu nama kolom yang berbohong

`STS_PROSES` terbaca sebagai "status proses", dan model pertama yang terbayang adalah enum
aktif/non-aktif. Skill ini menuntut istilah diuji ke pemakaiannya, bukan ke namanya.

Dua pembacaan mematahkannya: kontrol di form **tanpa daftar pilihan**, dan layar Arsip Dokumen
membacanya sebagai **`"NoteKasir"`**. Ia catatan bebas.

**Manfaat.** Tiga akibat yang seluruhnya mengikuti dari satu koreksi istilah: tipenya `string` biasa
dan bukan union, kolomnya dirender **teks polos** dan bukan lencana berwarna, dan migrasinya membuat
kolom `VARCHAR2` lebar dan bukan `CHAR(1)`. Salah satu saja dari ketiganya akan menolak data yang
selama ini sah.

## `codebase-design` — kapan modul baru boleh menuntut jembatan baru

Modul ini yang pertama menulis jejak simpan (`USER_EDIT`, `TGL_EDIT`), sehingga ia butuh dua hal yang
tiga modul master sebelumnya tidak butuh: identitas pemanggil dan jam.

Pertanyaan yang skill ini paksakan: **di mana keduanya tinggal supaya seam-nya tidak bocor.**
Jawabannya diambil dari pola yang sudah ada, bukan dikarang — jembatan `Caller` ditiru dari Master
Rekening dan dipasang di `cmd`, seam `Clock` ditiru dari modul auth dan dipasang di usecase.

**Manfaat.** Modul auth dan modul ini tetap tidak saling mengimpor, dan waktu simpan dapat diuji.
Ditambah satu uji yang menjaga batasnya: `TestSaveTimeComesFromTheApplicationNotTheDatabase` gagal
bila ada yang kelak mengganti parameter waktu dengan `CURRENT_TIMESTAMP` "supaya lebih ringkas".

## Teknik yang dipakai tanpa memanggil skill

**Membaca urutan grid dari nomor, bukan dari kedekatan baris.** Di
`BrowseLstDocType_RD-RD.xml`, `pySortType=ASC` berada di antara `.TGL_EDIT` dan `.ID`. Yang
menentukan pemiliknya adalah `pySortOrder=1` yang menempel pada `.ID` — bukan baris yang paling
dekat. Menebak dari kedekatan akan menghasilkan `ORDER BY TGL_EDIT`, dan grid tampil dengan urutan
yang berbeda dari layar lama tanpa ada yang menyadarinya.

**Membaca lebar nomor urut dari procedure-nya sendiri, bukan dari modul tetangga.** Tiga master
memakai pola yang sama dengan lebar berbeda: tiga digit (Status Klaim), empat (modul ini), lima
(Dokumen Travel). Ada uji khusus yang menguncinya.

## Kesalahan sendiri yang tercatat sesi ini

**`git stash` dipakai sebagai alat baca di repositori yang working tree-nya memuat pekerjaan orang
lain yang belum di-commit.**

Tiga uji `master-rekening` gagal. Untuk membuktikan itu bukan akibat suntingan saya, saya men-stash
tiga berkas bersama yang saya ubah. Stash itu **ikut membatalkan perbaikan konflik merge rekan yang
belum tersimpan di riwayat** — dan justru dari situ terlihat bahwa `HEAD` sendiri masih memuat
`<<<<<<< HEAD`.

Dipulihkan `git stash pop` tanpa konflik dan diperiksa utuh. Tetapi buktinya sebenarnya sudah
tersedia **tanpa menyentuh apa pun**: `git diff` memperlihatkan penanda konflik itu, dan satu
perintah baca sudah cukup.

Pola yang harus diingat: **sebelum memakai perintah yang mengubah working tree untuk "sekadar
memeriksa", tanyakan apakah `git diff` atau `git show` sudah menjawabnya.** Hampir selalu sudah.

## Catatan untuk sesi berikutnya

- Dua master **turunan** layar ini belum dibangun dan keduanya merujuk `ID` yang diterbitkan modul
  ini: `ListDetTypeDocument` dan `DetTypeDocumenBisnis`. Keduanya tetap tampil "belum tersedia".
- Migrasi 0005 langkah 0 memuat **lima pertanyaan ke DBA**, dan yang pertama menentukan isi langkah
  3: apakah `OLD_ID` kolom sungguhan atau nilai dari JSON.
- Branch ini sedang berada di tengah merge yang belum selesai. Sebelum menambah modul berikutnya,
  keadaan `HEAD` layak dibereskan lebih dulu — uji yang gagal di `master-rekening` kemungkinan besar
  ikut selesai bersamanya.

---

# Sesi kedua belas — modul Daftar Detail Dokumen Travel (2026-09-22)

Catatan kejujuran lebih dulu: **tidak satu pun skill dipanggil lewat perkakas Skill pada sesi ini.**
Yang dicatat di bawah adalah **teknik** dari skill Matt Pocock yang diterapkan tanpa memanggilnya —
sama seperti sesi kesepuluh dan kesebelas. Penyebutannya tetap ditulis karena yang berguna bagi
pembaca berikutnya adalah tekniknya dan hasilnya, bukan apakah sebuah perkakas dijalankan.

## `grilling` — dipakai untuk menolak membangun di atas lubang

Waktu: sebelum satu baris kode ditulis.

Pembacaan `Harness/ListDocumentTravel-Harness.xml` sampai ke rule-rule turunannya menemukan bahwa
**jalur tulis layar ini tidak ada di export**: tombol Simpan menunjuk `CNMInsertDocumentTravel_act`,
tombol Ubah menunjuk `CNMSetDetailTravelDocument_act`, dan keduanya hilang (`R-16`). Tidak ada pula
procedure penggantinya.

Teknik `grilling` yang dipakai: **jangan tawarkan pilihan sebelum faktanya dicari sendiri.** Tiga
pertanyaan yang diajukan ke Work Owner masing-masing sudah disertai apa yang sudah terbaca, apa yang
tidak, dan akibat tiap pilihan — bukan "bagaimana menurut Bapak".

Keluaran: tiga keputusan (`keputusan-implementasi.md` §22.1), dan yang terpenting **jalur tulis
dibangun dengan nama objek yang ditandai sebagai dugaan**, bukan dibangun seolah diketahui.

Manfaat yang terukur: migrasi `0006` menjadi daftar pertanyaan ke DBA dengan enam kueri katalog,
bukan pemberian hak atas objek yang mungkin tidak ada. Tanpa teknik ini, modul akan tampak selesai
dan baru gagal pada penyimpanan pertama di lingkungan sungguhan.

## `research` — dipakai karena Work Owner menyuruh mengecek ulang

Waktu: setelah jawaban pertanyaan lingkup, sebelum rancangan seam ditetapkan.

Work Owner menjawab *"seperti aplikasi PEGA saja **coba cek lagi**"*. Pengecekan ulang itu bukan
formalitas — ia **mengubah rancangan**.

Yang dicari: dari mana sebenarnya daftar Plan dan Jaminan berasal, mengingat kedua Report Definition
yang dirujuk layar (`BrowsePlanTravelMaster_RD`, `SearchCoverageTravel_RD`) hilang dari export.

Yang ditemukan: `Activity/SetspreadingtoCoverage-Act.xml` menyimpan **pembenaran peringatan Pega**
yang berbunyi apa adanya — *"ngambil data coverage bukan dari coverage travel tapi dari
m_plantravel"* — ditambah nama rule `GetDataMasterCoverageTravel_m_plantravel`.

Manfaat: nama tabelnya terbaca, dan kedua kelas Pega yang tampak berbeda terbukti **dua sudut
pandang atas satu tabel**. Akibatnya keduanya diisi **satu seam**, bukan dua. Dua seam akan
menyiratkan dua sumber yang sebenarnya satu, dan orang berikutnya akan mencari tabel kedua yang
tidak pernah ada.

Pelajaran yang layak diingat: **komentar dan pembenaran peringatan di dalam rule Pega adalah sumber
yang sah.** Ia sering memuat hal yang tidak dapat disimpulkan dari struktur rule-nya sendiri.

## `domain-modeling` — satu kolom yang memikul dua peran

Waktu: saat menyusun `Detail` dan `Coverage`.

Yang ditajamkan: `DOCUMENTNAME` dan `STSWAJIB` **ada di kedua tabel**, induk maupun coverage. Godaan
pertama adalah menormalkannya — simpan sekali di induk, baca dari sana.

Pemeriksaan ke `Activity/BrowseDocTravel-Act.xml` membatalkan godaan itu: ia membaca kedua kolom itu
**dari baris coverage**, bukan dari induknya, dan jalur registrasi klaim di
`Activity/TravelDocument_act-Act.xml` bergantung padanya. Mengosongkannya akan membuat dokumen wajib
berhenti terbaca pada klaim yang dibatasi per jaminan — **cacat yang tidak terlihat di layar master
ini sama sekali**.

Keputusan: keduanya tetap diisi di kedua tabel, dan alasannya ditulis di berkas `.sql` tepat di atas
pernyataan yang mengisinya.

Yang ditajamkan kedua: arti `STSWAJIB`. Namanya menyiratkan teks; ia **angka**, dan buktinya
precondition langkah `.STSWAJIB==1` / `.STSWAJIB==0`. Kontrak API tetap mengirimnya sebagai boolean —
angkanya bentuk penyimpanan, dan tempat penerjemahannya dipusatkan di satu berkas.

## `codebase-design` — kapan satu seam menjadi tiga

Waktu: saat menetapkan batas modul.

Pertanyaannya: modul ini menyentuh **empat tabel**, dua miliknya dan dua milik pihak lain. Satu seam
atau beberapa?

Prinsip yang dipakai: **seam dibuat di tempat yang benar-benar bervariasi, dan batas kepemilikan
ditegakkan oleh bentuk antarmuka, bukan oleh ingatan.** Hasilnya tiga seam:

| Seam | Tabel | Operasi | Kenapa terpisah |
|---|---|---|---|
| `Repo` | dua tabel milik modul | baca + tulis | — |
| `DocumentRepo` | `M_DOCTRAVEL` | hanya `List` | dimiliki modul lain (`P-1`) |
| `PlanRepo` | `M_PLANTRAVEL` | hanya baca | dimiliki GISFW (`D-03`) |

Ketiadaan operasi tulis pada dua yang terakhir **bukan kelalaian melainkan penegakan**: tidak ada
tempat di antarmuka itu untuk menambahkan tulis tanpa sengaja. Pola yang sama sudah dipakai
`BusinessRepo` pada modul Master COL Simas Online.

Keputusan kedua dari prinsip yang sama: **seluruh nama objek tulis yang belum terverifikasi
diisolasi di satu berkas `.sql`.** Jawaban DBA yang berbeda hanya mengubah satu berkas, dan tidak ada
nama tabel yang tercecer di dalam kode Go.

## Teknik yang dipakai tanpa memanggil skill

- **Membaca pola dari modul yang sudah ada sebelum menulis yang baru.** Lima modul master yang sudah
  jalan dibaca utuh — domain, usecase, repo, http, frontend — sebelum satu berkas dibuat. Itu yang
  membuat modul ini tidak memperkenalkan satu pun pola baru kecuali yang memang dituntut bentuk
  datanya.
- **Menulis pengujian yang menjaga KEPUTUSAN, bukan sekadar menjalankan kode.** Contoh:
  `TestDaftarTidakMembawaJaminannya` ada supaya perbedaan antara daftar dan pengambilan satu baris
  tidak "dirapikan" seseorang kelak — perapian itu akan menghapus pembatasan plan pengguna tanpa
  satu pun galat.
- **Memakai pengujian yang gagal sebagai pertanyaan, bukan sebagai gangguan.** Satu uji yang gagal
  menuntun ke penemuan bahwa isian Minimal Unggah di Pega adalah `pxTextInput`, dan bahwa pilihan
  `type="number"` saya menyembunyikan isian tak-valid alih-alih menolaknya.

## Kesalahan sendiri yang tercatat sesi ini

| # | Kesalahan | Bagaimana ketahuan | Perbaikan |
|---|---|---|---|
| 1 | Memakai `type="number"` untuk Minimal Unggah | uji validasi gagal — nilainya dikosongkan peramban, bukan ditolak | diperiksa ke Pega: `pxTextInput`. Diganti `type="text"` + `inputMode="numeric"` |
| 2 | Memakai `z.coerce.number()` di skema form | `tsc --noEmit` menolak resolver-nya (Zod 4 membuat masukannya `unknown`) | disimpan sebagai teks, diubah menjadi angka saat menyusun badan permintaan |
| 3 | Uji mencari `role="alert"` yang tidak ada pada komponen `Field` | uji gagal | uji diperbaiki mencari teksnya; ketidakseragaman komponennya dicatat, **tidak** diperbaiki sepihak |
| 4 | Menomori migrasi `0005` yang sudah dipakai modul sesi sebelumnya | mendaftar isi `migrations/` sebelum membuat berkas | seluruh rujukan di empat berkas dikoreksi menjadi `0006` |
| 5 | Semula menyimpulkan Plan dan Jaminan berasal dari dua tabel | pengecekan ulang yang diminta Work Owner | keduanya diisi satu seam |

Kesalahan 1 dan 5 punya pola yang sama dan layak diingat: **keduanya lolos pembacaan pertama dan
tertangkap oleh pemeriksaan kedua** — yang satu oleh pengujian, yang lain oleh perintah Work Owner.
Pembacaan pertama atas rule Pega tidak pernah cukup untuk hal yang menentukan bentuk isian maupun
bentuk data.

## Catatan untuk sesi berikutnya

- **Jalur simpan modul ini belum terbukti terhadap Oracle.** Tiga nama objek masih dugaan, dan
  migrasi `0006` Bagian 1 yang menjawabnya. Mode `claimpnc -periksa` hanya membuktikan jalur BACA;
  itu ditulis eksplisit di laporannya supaya tidak dibaca sebagai "siap".
- **Uji `master-rekening` yang gagal masih yang sama** dengan yang dicatat sesi kesebelas — 3 gagal,
  akibat merge yang belum selesai, bukan akibat modul ini. Ia tetap belum dibereskan.
- Pola "master induk + master turunan" kini sudah muncul **dua kali**: Dokumen Travel (22 → 39) dan
  Tipe Dokumen (40 → dua turunan yang belum dibangun). Modul turunan berikutnya dapat mengikuti
  bentuk modul ini apa adanya, termasuk pemisahan daftar-vs-satu-baris dan penggantian menyeluruh
  daftar anaknya.

---

## Sesi keempat belas — modul Daftar Tipe Dokumen Bisnis (2026-09-23)

### Skill yang dipakai

| Skill | Kapan | Untuk apa | Keluarannya |
|---|---|---|---|
| `mattpocock-skills:codebase-design` | sebelum berkas pertama dibuat | menentukan **berapa seam** dan di mana batasnya | lima seam, empat di antaranya baca-saja |
| `mattpocock-skills:domain-modeling` | saat menamai isi tabel | memisahkan **apa yang disimpan** dari **alias yang menyesatkan** | sebelas alias dibuang, nama domain dipakai |
| `mattpocock-skills:grilling` | sepanjang pembacaan Pega | menguji premis terhadap bukti, bukan terhadap komentar | empat temuan §23.4, tiga di antaranya cacat |

### `codebase-design` — kenapa lima seam, bukan satu

Modul ini menyentuh **enam tabel**: dua miliknya dan empat milik pihak lain. Prinsip yang dipakai
sama dengan sesi kedua belas — **batas kepemilikan ditegakkan oleh bentuk antarmuka, bukan oleh
ingatan**:

| Seam | Objek | Operasi | Pemiliknya |
|---|---|---|---|
| `Repo` | `LST_TYPE_DOC_BUSINESS`, `COVERAGE_DOC_BUSINESS` | baca + tulis | modul ini |
| `BusinessRepo` | `BUSINESS` | hanya `List` | GISFW (`D-03`) |
| `DocumentTypeRepo` | `V_LST_DOC_TYPE` | hanya `List` | MENU_ID 40 (`P-1`) |
| `DetailTypeDocRepo` | `V_LST_DET_TYPE_DOC` | hanya `List` | MENU_ID 41, belum dibangun |
| `ObjectDocRepo` | `V_LST_DOC_OBJ` | hanya `List` | Daftar Objek Dokumen (`P-1`) |

Ketiadaan operasi tulis pada keempat yang terakhir **bukan kelalaian melainkan penegakan**: tidak
ada tempat di antarmuka itu untuk menambahkan tulis tanpa sengaja.

Keputusan kedua dari prinsip yang sama: **satu tipe `Reference` untuk tiga seam**, bukan tiga tipe
kembar — ketiganya benar-benar berbentuk sama. Yang membedakan hanya satu field, `ParentID`, dan ia
ada karena satu daftar memang bergantung pada pilihan daftar lain.

Keputusan ketiga: **seluruh nama objek diisolasi di satu berkas `.sql`**, sehingga jawaban DBA yang
berbeda hanya mengubah satu berkas.

### `domain-modeling` — sebelas alias yang dibuang

`Select_TYPE_DOCUMENT` mengalias-namakan kolomnya ke nama yang **tidak ada hubungannya dengan
isinya** — akibat memaksakannya masuk kelas Pega generik `T_GENERAL`:

```
c.type_document AS BRANCHNAME      nama tahap dokumen, dialias "nama cabang"
a.object_doc_id AS BUSINESSTYPE    objek dokumen, dialias "jenis bisnis"
e.detail_document AS EDMNO         nama dokumen, dialias "nomor EDM"
a.sts_wajib AS EDMTYPE             penanda wajib, dialias "jenis EDM"
a.min_doc AS FOLLOWEDPOLICY        jumlah minimum, dialias "polis yang diikuti"
```

Ditambah satu lagi di `BrowseLSTDetailDocument_sql`: `b.note AS DOCUMENT_TYPE_ID` — **nama bisnis
dialias sebagai kode tipe dokumen**.

Seluruhnya dinamai ulang mengikuti isinya (`D-19`), dan pemetaannya ditulis lengkap di kepala
kueri penggantinya supaya siapa pun yang membandingkan keduanya tidak perlu menelusurinya lagi.

Satu penajaman istilah yang mengubah rancangan: `TYPE_DOCUMENT` ternyata bukan "jenis dokumen"
melainkan **tahap klaim**. Itu terbaca bukan dari namanya melainkan dari keenam kueri yang
mencarinya dengan teks persis (`'REGISTER'`, `'PAYMENT'`, …). Tanpa penajaman itu, isian di layar
akan diberi label yang salah dan penyempitan daftar Detail Dokumen tidak akan pernah terpikir.

### `grilling` — dipakai pada bukti, bukan pada orang

Disiplinnya dipakai pada diri sendiri: **setiap premis diuji ke source lebih dulu**, dan yang
paling penting — **komentar tidak dipercaya, precondition dibaca**.

Itu yang memunculkan keempat temuan §23.4. Komentar langkah berbunyi "kalo tambah > bisa banyak
bisnis & banyak dokumen"; precondition-nya mengatakan langkah itu tidak pernah berjalan. Keduanya
ditulis orang yang sama, pada hari yang sama.

Tiga premis saya sendiri yang **gugur** saat diuji:

| Premis awal | Yang terbukti |
|---|---|
| "Ada satu tabel jaminan" | **Dua** — `COVERAGE_DOC_BUSINESS` dan `COVERAGE_DOC_BUSINESS_NONGENERAL` |
| "Prosedur menulis seluruh kolom tabelnya" | **Tiga kolom** dibaca konsumen tetapi tidak pernah ditulis siapa pun |
| "Daftar Detail Dokumen menampilkan seluruh master" | Ia **disaring** parameter `idDocument` |

Ketiganya lolos pembacaan pertama dan tertangkap pembacaan kedua — pola yang sama dengan yang
dicatat sesi kedua belas, dan sekarang muncul untuk kedua kalinya.

### Teknik yang dipakai tanpa memanggil skill

- **Membaca KONSUMEN sebelum menulis PRODUSEN.** Keenam kueri unggah dokumen dibaca sebelum satu
  baris kode ditulis, dan justru dari sanalah aturan terpenting modul ini terbaca — bahwa
  `STS_WAJIB` disaring lagi oleh daftar jaminan. Membaca layar masternya saja tidak akan pernah
  memperlihatkannya.
- **Menulis pengujian yang menjaga KEPUTUSAN.** `TestDaftarTidakMembawaJaminanTetapiPengambilanSatuBarisMembawanya`
  ada supaya perbedaan antara daftar dan pengambilan satu baris tidak "dirapikan" seseorang kelak —
  perapian itu akan membuat layar menampilkan setiap baris seolah tanpa jaminan.

  Dua belas uji di `repo/sqlstore/query_test.go` dipakai dengan cara yang sama, dan tiga di
  antaranya menjaga hal yang **tidak dapat ditangkap kompilator**:
  `TestNoQueryDeletesAnything` (jaminan tidak pernah dihapus — lebih ketat daripada padanannya di
  modul Travel, yang memang memakai `DELETE`), `TestReferenceTablesAreNeverWritten` (batas `P-1`
  ditegakkan pada teks SQL, bukan hanya pada bentuk seam), dan
  `TestRuleListKeepsRowsWhoseMastersAreGone` (`LEFT JOIN`, supaya baris yatim tidak kembali
  tersembunyi seperti di kueri lama).

- **Menghapus kerapuhan begitu terlihat, bukan mencatatnya sebagai utang.** Jalur penambahan
  jaminan semula membaca `BUSINESSID` dengan memindai sebelas kolom berposisi dari `rule_get`.
  Itu benar hari itu, tetapi satu kolom yang tertukar kelak **tidak menimbulkan galat apa pun** —
  ia hanya menulis `BUSINESSID` yang salah, dan jaminan yang tidak cocok dengan induknya berhenti
  membuat dokumennya wajib tanpa satu pun tanda. Diganti kueri satu kolom tersendiri.
- **Memakai kegagalan alat sebagai tanda, bukan gangguan.** Ekstraktor XML yang mengembalikan nol
  kecocokan tanpa galat ternyata gagal pada CRLF; kalau angkanya diterima apa adanya, kesimpulannya
  adalah "berkas itu tidak memuat SQL".
- **Menjalankan gerbang lebih awal dan sering.** `go build` dijalankan setelah setiap lapisan
  selesai, bukan sekali di akhir — sehingga kesalahan impor `time` tertangkap pada lapisan domain,
  bukan setelah sepuluh berkas ditulis di atasnya.

### Kesalahan sendiri yang tercatat sesi ini

| # | Kesalahan | Bagaimana ketahuan | Perbaikan |
|---|---|---|---|
| 1 | Memakai `time.Time` tanpa mengimpor `time` | `go build` pada lapisan domain | impor ditambahkan |
| 2 | Menulis satu fungsi pemilih memori untuk tiga tipe antarmuka berbeda | `go build` menolaknya | dipecah menjadi tiga fungsi, dan alasannya ditulis di komentar |
| 3 | Memakai `exact` pada `ByRoleOptions` | `tsc --noEmit` | **yang keliru bukan pengujiannya melainkan layarnya** — dua tombol berbunyi "Tambah" sambil melakukan hal yang sangat berbeda; tombol jaminan diberi `aria-label` tersendiri |
| 4 | Menamai rute `/master/object-dokumen-pilihan` | pemeriksaan ke modul tetangga sebelum wiring | diseragamkan menjadi `objek`, mengikuti modul yang memiliki tabelnya |
| 5 | Ekstraktor XML berbasis regex dipercaya meski hasilnya nol | `grep` biasa menemukan tag yang sama | seluruh pembacaan XML berpindah ke `tr -d` + `awk` |
| 6 | Menamai kedua master rujukan "belum dibangun" | dua sesi lain membangunnya bersamaan di tengah pengerjaan | enam komentar dikoreksi; **seam-nya tidak berubah** — ia memang sudah menyatakan tabelnya milik modul lain |
| 7 | Uji disiplin SQL mencari potongan teks `"ID ="` untuk memastikan kunci baris tidak diubah | uji GAGAL — `"ID ="` juga cocok dengan `DOCUMENT_TYPE_ID =`, `OBJECT_DOC_ID =`, dan `DOC_TYPE_DT_ID =` | nama kolom yang ditulis **dikumpulkan** dari klausa SET lalu dibandingkan persis, bukan dicari sebagai substring |

Kesalahan 7 layak dicatat meski kecil: uji itu **gagal pada kueri yang benar**. Kalau saya
melonggarkannya alih-alih memperbaikinya, penjaga yang tersisa akan berbunyi benar sambil tidak
menjaga apa pun — dan kolom `ID` yang kelak ikut masuk klausa SET akan lolos tanpa suara.

### Kesalahan pada babak kedua (pemeriksaan ulang "seperti Pega")

| # | Kesalahan | Bagaimana ketahuan | Perbaikan |
|---|---|---|---|
| 8 | Isian autocomplete mengambil nilainya dari **kode hasil pencarian** | dua uji gagal — isiannya **terhapus sendiri setiap huruf diketik**, karena ketikan yang belum cocok menghasilkan kode kosong | teks yang diketik disimpan berdampingan dengan kodenya |
| 9 | `initialRules?: T[]` ditolak `exactOptionalPropertyTypes` | `tsc` | ditulis `?: T[] \| undefined` |
| 10 | `context` tidak diimpor di berkas uji rute | `go test` gagal build | impor ditambahkan |
| 11 | **Modul tidak didaftarkan ke mode `-periksa`** meski `CheckTable`-nya sudah ditulis | Work Owner bertanya "beres 100%?", dan pertanyaan itu memicu pemeriksaan daftar modul di `check.go` — seluruh modul bersaudara ada di sana, modul ini tidak | `checkBusinessDocumentRule` ditambahkan |

**Kesalahan 11 adalah yang paling mudah lolos sampai produksi.** Ia tidak menggagalkan satu pun
gerbang: build lulus, vet lulus, seluruh uji lulus, dan layarnya berfungsi penuh terhadap adapter
memori. Yang hilang hanya kemampuan **membuktikan modul ini terhadap Oracle sebelum dipakai** —
persis hal yang paling dibutuhkan, karena jalur simpannya memang belum pernah diuji di sana.

Yang menemukannya bukan perkakas melainkan **satu pertanyaan**: "beres 100%?". Menjawabnya dengan
memeriksa alih-alih mengiyakan adalah satu-satunya sebab ia ketahuan sekarang dan bukan saat
seseorang menjalankan `-periksa` lalu bingung mengapa modulnya tidak disebut.

**Kesalahan 8 adalah yang paling berharga sesi ini.** Ia cacat yang **mustahil terlihat tanpa
menjalankan layarnya** — kodenya terbaca masuk akal, tipe-nya benar, dan `tsc` diam. Yang
menemukannya adalah dua uji yang saya tulis untuk menjaga hal lain, dan keduanya gagal dengan pesan
yang langsung menunjuk sebabnya.

Ia juga menegaskan pola yang sudah muncul dua kali sebelumnya di sesi ini: **pembacaan pertama atas
rule Pega tidak pernah cukup untuk hal yang menentukan bentuk isian.** Mengganti dropdown menjadi
autocomplete tampak seperti penukaran komponen; ternyata ia mengubah dari mana nilai isian berasal.

### Satu pemeriksaan yang menolak dikerjakan dari laporan orang lain

Saat Work Owner memutuskan "ikutin Pega aja" untuk ketiga hal, butir ketiga — "jalur Tambah tidak
menulis apa pun" — sampai saat itu **hanya berasal dari laporan sub-agen**. Konsekuensinya besar:
menirunya berarti tombol Simpan yang tidak menyimpan.

Ia dibaca ulang langsung dari export sebelum satu baris pun diubah, dan itu menuntut menemukan
dulu **aturan pasangan** antara ekspresi precondition dan langkah pemiliknya — pada berkas ini
ekspresi muncul *sesudah* `pyStepsDescription` milik langkahnya. Setelah itu strukturnya terbaca
dan klaimnya **terkonfirmasi**.

Prinsipnya: **laporan sub-agen adalah petunjuk, bukan bukti** — terutama ketika ia menjadi dasar
keputusan yang mematikan sebuah fitur.

Kesalahan 3 layak diingat: **pengujian yang sulit ditulis sedang melaporkan sesuatu tentang
kodenya**, bukan tentang pengujiannya. Kesulitan membedakan dua tombol di dalam uji adalah
kesulitan yang sama persis yang akan dialami pengguna pembaca layar.

### Catatan untuk sesi berikutnya

- **Jalur simpan modul ini belum terbukti terhadap Oracle.** Berbeda dari sesi kedua belas,
  sebabnya bukan nama objek yang masih dugaan — seluruhnya terbaca dari procedure — melainkan
  **bentuk kolomnya** yang belum diperiksa, dan pertanyaan apakah `ID` benar-benar unik.
- **Dua cacat layar lama menunggu keputusan Work Owner**, dan keduanya menentukan apa yang
  dibandingkan pada gerbang 1. Ini pertama kalinya sebuah modul diserahkan dengan selisih terencana
  yang **bukan** berasal dari daftar 13 butir `D-49`.
- Pola "master induk + master turunan" kini muncul **tiga kali**: Dokumen Travel (22 → 39), Tipe
  Dokumen (40 → 41 dan 42). Yang tersisa dari keluarga itu adalah **MENU_ID 41**, dan modul ini
  sudah membaca tabelnya sebagai daftar pilihan — seam-nya tinggal dipakai ulang.
- **Lima pertanyaan rancangan diputuskan sendiri** karena tidak dijawab. Seluruhnya tercatat di
  `keputusan-implementasi.md` §24.1 beserta dasarnya, dan seluruhnya dapat dicabut.

---

# Sesi modul Daftar Objek Dokumen (2026-09-23)

| | |
|---|---|
| **Permintaan** | Menambahkan modul Daftar Objek Dokumen, acuan `Harness/ListDocumentObject-Harness.xml` |
| **Batasan yang diberikan** | Dilarang menulis kode sebelum memahami struktur, arsitektur, pola, dan alur bisnis Pega |

## `mattpocock-skills:grilling` — dipakai pada diri sendiri, lalu pada Work Owner

**Kapan.** Sebelum satu baris kode ditulis, setelah pembacaan export selesai.

**Yang dikerjakan.** Disiplinnya dipakai dua arah.

Ke **diri sendiri**: setiap nama objek basis data yang hendak dipakai ditanya "dari mana ini
dibaca". Hasilnya memisahkan tiga nama yang **terbaca** dari tiga nama yang **dugaan** — pemisahan
yang tidak akan muncul bila pertanyaannya hanya "apa nama tabelnya".

Ke **Work Owner**: dua pertanyaan diajukan, keduanya disertai rekomendasi beserta akibatnya, dan
keduanya adalah pertanyaan yang **mengubah bentuk pekerjaan** — bukan pertanyaan yang jawabannya
dapat ditebak dengan aman.

| Pertanyaan | Kenapa ia layak ditanyakan |
|---|---|
| Grid "ID Bisnis" ikut dibangun? | Jawabannya menentukan modul memuat satu tabel atau dua, dan perlu tidaknya seam baca-saja ke tabel milik GISFW |
| Jalur simpan: kolom atau `JSON_DATA`? | Jawabannya menentukan bentuk migrasi, dan apakah layar Pega lama masih dapat membaca baris yang ditulis aplikasi baru |

**Manfaatnya.** Pertanyaan pertama menghemat pekerjaan yang salah arah: membangunnya tanpa bertanya
berisiko lingkup yang tidak diinginkan, sedangkan melewatkannya berisiko layar yang kehilangan satu
grid yang jelas ada dan terisi di Pega.

**Yang TIDAK ditanyakan, dan alasannya.** Tiga hal sengaja diputuskan sendiri karena presedennya
sudah tegas dan tercatat: `OLD_ID` dikirim tetapi tidak ditampilkan (§24.8), tidak ada tombol hapus
(`D-66`), dan nama field JSON mengikuti label layar (§24.7). Menanyakan hal yang sudah diputuskan
menambah beban tanpa menambah kendali.

## `mattpocock-skills:codebase-design` — memilih preseden yang tepat, bukan yang terdekat

**Kapan.** Saat menentukan bentuk modul.

**Yang dikerjakan.** Enam modul master sudah ada, dan godaannya menyalin yang terakhir dikerjakan.
Yang dipakai justru **dua preseden berbeda untuk dua hal berbeda**:

| Diambil dari | Apa yang diambil | Kenapa yang itu |
|---|---|---|
| §18 Master COL Simas Online | bentuk modul: master + grid bisnis + seam `BusinessRepo` baca-saja | satu-satunya yang bentuknya sama persis |
| §21 Daftar Tipe Dokumen | bentuk penerbitan ID: **empat digit** | saudara sekandung di rumpun tabel `LST_*` |

Kosakata skill ini yang membuat pemisahan itu terlihat: yang diambil dari §18 adalah **bentuk
seam-nya**, sedangkan yang diambil dari §21 adalah **perilaku di dalam satu adapter**. Keduanya
lapisan yang berbeda, sehingga tidak ada alasan keduanya harus berasal dari modul yang sama.

**Manfaatnya terukur.** Menyalin bentuk ID dari §18 akan menghasilkan ID tiga digit — salah, dan
salah dengan cara yang tidak menimbulkan galat sampai baris pertama disimpan ke Oracle.

## `mattpocock-skills:codebase-design` — satu tempat yang sengaja menyimpang dari presedennya

**Yang dikerjakan.** Prinsip "seam yang bentuknya menegakkan aturan, bukan disiplin orangnya"
dipakai untuk memeriksa apakah kekurangan preseden layak diwarisi.

Tabel pemetaan bisnis pada §18 tidak punya kolom urutan maupun penanda aktif, dan akibatnya tercatat
di sana sebagai utang teknis: **pencabutan pilihan tidak dapat disimpan sama sekali**. Tabel modul
ini belum ada, sehingga kedua kolom itu dapat dirancang sejak awal.

**Keputusannya:** menyimpang. Mewarisi kekurangan yang sudah diketahui ke tabel yang bahkan belum
dibuat berarti menciptakan utang yang sama secara sukarela.

**Yang menjaga penyimpangan itu tetap terbaca:** nama fungsinya. Di modul ini `replaceBusinesses`;
di §18 fungsi yang setara sengaja BUKAN bernama replace, justru karena ia tidak menggantikan.

## `mattpocock-skills:domain-modeling` — satu label yang bermakna dua

**Kapan.** Saat menentukan nama field JSON untuk kolom `KET_DOC_OBJ`.

**Yang dikerjakan.** Labelnya di Pega adalah "Daftar Objek Dokumen" — sama persis dengan judul
layarnya. Menyalinnya apa adanya menghasilkan field `daftar_objek_dokumen`, yang menyatakan barisnya
adalah sebuah daftar.

Ketajaman yang diperlukan: kata "Daftar" di depan label itu milik **layarnya**, bukan milik
**nilainya**. Layar itu daftar objek dokumen; satu barisnya adalah satu objek dokumen.

**Hasilnya:** `objek_dokumen`. Aturan §24.7 — field JSON mengikuti label layar — tetap dipatuhi,
tetapi dipatuhi terhadap apa yang label itu maksud, bukan terhadap hurufnya.

## Teknik yang dipakai tanpa memanggil skill

| Teknik | Hasilnya di sesi ini |
|---|---|
| **Telusuri sampai ke rule turunannya, jangan berhenti di harness** | Harness-nya sendiri 283 KB dan nyaris seluruhnya boilerplate. Yang menjawab pertanyaan justru Report Definition (ketiga kolom) dan Section browse (grid bisnis, dua activity yang hilang) |
| **Cari tetangga yang namanya mirip lebih dulu** | Menemukan bahwa Tipe Dokumen (MENU_ID 40) dan Objek Dokumen (MENU_ID 43) dirujuk BERSAMAAN oleh satu tabel. Tanpa itu, modul ini berisiko dibangun seolah pengganti modul §21 |
| **Baca procedure tetangga sekandung untuk bentuk yang tidak terbaca** | `PEGA_LST_DOC_TYPE.prc` memberi bentuk penerbitan ID, nama urutan, dan pola nama tabel dasar — ketiganya tidak ada di rule modul ini |
| **Tulis migrasi sebagai daftar pertanyaan, bukan DDL** | Dua kueri di bagian 0 dirancang untuk **membatalkan sebagian migrasinya sendiri** bila jawabannya mengejutkan |
| **Uji yang menegaskan bentuk, bukan sekadar keberadaan** | `TestIDDiterbitkanPenyimpanan` menegaskan ID-nya `10005`, bukan sekadar "ada ID". Itulah yang menangkap kesalahan data contoh |

## Kesalahan sendiri yang tercatat sesi ini

| Kesalahan | Bagaimana ketahuan | Pelajaran |
|---|---|---|
| Data contoh memakai ID empat karakter (`1001`) — pola rumpun `M_CAUSE_OF_LOSS` yang tiga digit | `TestIDDiterbitkanPenyimpanan` gagal: `expected "1005", actual "10005"` | Lebar nomor urut **per rumpun tabel**, dibaca dari procedure-nya masing-masing. Kode saya benar; data contohnya yang salah |
| Uji frontend memeriksa tombol "Refresh" sebelum daftar selesai dimuat | Uji gagal; tombolnya memang berbunyi "Memuat…" saat itu | Uji yang memeriksa keadaan sesaat menguji hal yang salah. Yang diperbaiki ujinya, bukan layarnya |
| Import baru ditaruh di urutan yang salah pada `check.go` | `gofmt -l` | Format diperiksa sebagai perkakas, bukan diandalkan pada ketelitian |

Ketiganya dilaporkan apa adanya, bukan dirapikan diam-diam.

## Catatan untuk sesi berikutnya

- **Dua sesi berjalan bersamaan di repo ini.** Modul `daftartipedokumenbisnis` (MENU_ID 42)
  ditambahkan sesi lain saat modul ini dikerjakan. Satu kali `go test ./cmd/...` gagal karena
  `main.go` sedang berada di keadaan setengah tersunting — bukan kegagalan yang nyata. Bila
  kegagalan build muncul pada simbol yang bukan milik modul yang sedang dikerjakan, **periksa apakah
  simbolnya sudah ada** sebelum menyimpulkan apa pun.
- **Rute `/master/bisnis` kini dipakai dua modul.** Yang mendaftarkannya tetap satu — §18. Modul
  berikutnya yang membutuhkan daftar bisnis mengikuti pola yang sama: seam sendiri, rute dipakai
  bersama, kunci cache frontend yang sama persis.
- **`daftartipedokumen`, `daftarobjekdokumen`, dan `daftartipedokumenbisnis`** kini bertiga di rumpun
  yang sama dan namanya mudah tertukar. Komentar silang sudah dipasang di `registry.ts`, dto, dan
  paket domain masing-masing.

---

# Sesi kelima belas — modul Daftar Detail Tipe Dokumen (2026-09-23)

**Tidak ada skill yang dipanggil lewat perkakas Skill pada sesi ini.** Yang dicatat di bawah adalah
**teknik** dari skill-skill itu yang benar-benar dipakai, beserta keluarannya — bukan daftar skill
yang tersedia.

## `mattpocock-skills:grilling` — tiga pertanyaan yang diajukan sebelum kode ditulis

**Kapan.** Sesudah analisis export selesai, sebelum satu baris kode ditulis.

**Kenapa.** Tiga hal ditemukan yang jawabannya mengubah **bentuk pekerjaan**, bukan sekadar
detailnya — dan ketiganya tidak dapat dijawab dari export:

1. Tabelnya hanya `(ID, JSON_DATA)`; kedua view membongkar JSON itu, dan DDL-nya tidak ada.
2. `STS_WAJIB` dibandingkan dengan empat nilai berbeda oleh tiga pembaca yang tidak sepakat.
3. Master `V_LST_DOC_OBJ` yang dirujuknya belum punya modul.

Disiplin yang dipakai: setiap pertanyaan disertai **pilihan jawaban beserta konsekuensinya**, bukan
"bagaimana menurut Bapak". Pilihan pertama pada setiap pertanyaan adalah yang direkomendasikan.

**Keluaran.** Ketiganya dijawab, dan **jawaban pertama membalik rencana**: dugaan awal adalah
tabelnya masih berbentuk JSON dan harus direplikasi. Jawaban Work Owner menyatakan kolomnya sudah
ada. Tanpa bertanya, modul ini akan ditulis untuk menyusun dokumen JSON yang bentuknya hanya dapat
diduga — pekerjaan yang lebih besar dan lebih rapuh, untuk hasil yang salah.

**Manfaat yang terukur.** Satu pertanyaan mencegah penulisan serializer JSON yang tidak akan pernah
dipakai.

## `mattpocock-skills:grilling` — satu bukti yang mengubah jawaban menjadi lebih kuat

Pertanyaan kedua diajukan dengan dua pilihan seimbang: `"Ya"/"Tidak"` versus `"1"/"0"`. Work Owner
memilih yang pertama.

Pemeriksaan sesudahnya menemukan bukti yang **membenarkan pilihan itu**, dan bukti itu belum ada di
tangan saat pertanyaannya disusun: `Activity/SetTypePDFAdjustment-Act.xml` membandingkan
`.STS_WAJIB=="Ya"` dan **tidak mengenali `"1"`**. Menulis angka akan membuatnya berhenti mengenali
dokumen wajib — tanpa satu pun galat.

Dicatat di sini karena polanya berguna: pertanyaan yang disusun sebelum seluruh bukti terkumpul
tetap layak diajukan, **asalkan buktinya dicari sesudahnya** dan jawabannya diperiksa ulang
terhadap bukti itu. Bila buktinya ternyata berlawanan, yang benar adalah kembali bertanya — seperti
yang ditempuh `D-47` pada penjenjangan komite.

## `mattpocock-skills:codebase-design` — kapan seam DIPISAH dan kapan DISATUKAN

**Kapan.** Saat memutuskan bentuk seam untuk keempat master rujukan.

**Persoalannya.** Preseden terdekat, modul Daftar Tipe Dokumen Bisnis, memakai **empat** selector
untuk **empat master yang sama persis**. Menyalinnya adalah jalan yang paling mudah dibenarkan.

**Yang dipakai dari skill.** Prinsip *satu adapter berarti seam hipotetis, dua adapter berarti seam
nyata* dibalik menjadi pertanyaan: **apa yang benar-benar bervariasi di sini?** Jawabannya: tidak
ada. Keempat master hidup di basis data yang sama, dibaca bersamaan oleh satu form, dan gagal
dengan cara yang persis sama — daftar pilihannya kosong sementara penyimpanannya tetap berjalan.

**Keputusannya.** SATU seam `ReferenceRepo` dengan empat metode baca, bukan empat seam. Empat
selector yang selalu dipilih bersamaan dan empat jalur galat yang ditangani identik bukan seam —
itu satu seam yang ditulis empat kali.

**Manfaat.** `assembly` bertambah satu field, bukan empat. Perakitan di `cmd` bertambah dua baris,
bukan lima. Dan penanganan galatnya tertulis satu kali.

## `mattpocock-skills:codebase-design` — satu tempat yang sengaja BERBEDA dari presedennya

Modul Daftar Objek Dokumen, yang dibangun sesi lain untuk grid yang bentuknya sama persis, memilih
**penandaan lunak** (`STS_AKTIF = '0'`) saat mengganti isi grid. Modul ini memilih **DELETE**.

Yang membedakan bukan selera melainkan **pembacanya**: `GetLbuDetType-SQL.xml` membaca view anak
tanpa satu pun syarat selain ID induknya. Baris yang ditandai tidak aktif karena itu **tetap
diberlakukan sebagai aturan dokumen pada klaim** — kebalikan dari yang dimaksud petugas.

Di Daftar Objek Dokumen tabelnya baru, dirancang lengkap dengan kolom penandanya, dan tidak ada
pembaca Pega yang melewatinya.

Dicatat karena inilah bentuk kekeliruan yang paling mudah terjadi saat modul sejenis dibangun
berturut-turut: **menyalin preseden terdekat tanpa memeriksa apakah alasannya masih berlaku.**

## `mattpocock-skills:domain-modeling` — satu kolom, dua arah pembacaan

Kolom `STS_WAJIB` memaksa satu keputusan pemodelan yang tidak biasa: **penulisannya tegas,
pembacaannya longgar.**

Model yang paling mudah — satu tipe enum dengan dua nilai — akan menggagalkan seluruh daftar karena
satu baris warisan berisi `"1"`. Model yang paling longgar — teks apa adanya — akan membocorkan
keempat nilai itu ke kontrak API dan ke setiap klien.

Yang dipakai: **boolean di domain dan di kontrak**, dengan dua fungsi jembatan di batasnya —
`MandatoryText` untuk menulis, `MandatoryFrom` untuk membaca. Keduanya di lapisan domain, bukan di
adapter, supaya kedua adapter (Oracle dan memori) tidak mungkin berbeda tafsiran.

## Teknik yang dipakai tanpa memanggil skill

- **Membaca jalur tulis dari procedure, bukan dari activity.** Activity hanya memperlihatkan
  parameter yang dikirim; procedure memperlihatkan kolom yang benar-benar diisi. Pada modul ini
  keduanya berbeda jauh — activity mengirim satu dokumen JSON, procedure menyisipkannya ke satu
  kolom.
- **Menghitung pembaca sebelum memutuskan apa pun tentang view.** 34 rule membaca view induk
  modul ini; angka itu yang membuat migrasi 0009 ditulis sebagai daftar pertanyaan, bukan DDL.
- **Menguji urutan rute.** `/pilihan` dan `/{id}` berada di sub-rute yang sama. chi memang
  mendahulukan jalur tetap, tetapi bergantung pada perilaku pustaka tanpa uji berarti perubahan
  urutan kelak lolos diam-diam.

## Kesalahan sendiri yang tercatat sesi ini

- **Satu `grep` dipercaya terlalu cepat.** Pemeriksaan awal menyimpulkan modul
  `daftartipedokumenbisnis` "belum dirakit di `main.go`". Pemeriksaan belakangan menemukannya di 27
  tempat — berkasnya berubah di tengah pengerjaan karena sesi lain menyuntingnya. Yang salah bukan
  perintahnya melainkan **menyimpulkan keadaan dari satu pembacaan pada repo yang sedang berubah.**
  Sesudah itu, setiap suntingan ke `main.go` didahului pembacaan ulang.
- **Pertanyaan ketiga ke Work Owner sudah usang saat diajukan.** Ia menawarkan tiga cara
  mendapatkan daftar Objek Dokumen, sementara modul masternya sudah dibangun sesi lain dan ada di
  direktori kerja sebagai berkas yang belum di-commit. `git status` seharusnya dibaca **sebelum**
  pertanyaan disusun, bukan sesudahnya.

## Catatan untuk sesi berikutnya

- **Rumpun tiga master dokumen lengkap.** MENU_ID 40, 41, dan 42 semuanya sudah ada. Komentar
  silang di `registry.ts`, `daftartipedokumen.go`, dan `daftartipedokumenbisnis.sql` sudah
  diperbarui dari "belum dibangun".
- **Baca `git status` sebelum menyusun pertanyaan.** Dua modul yang belum di-commit mengubah dua
  dari tiga pertanyaan sesi ini.
- **Pola `UseReferences` pada adapter memori** layak dipakai ulang oleh modul mana pun yang
  keterangannya datang dari join view. Tanpa itu, pengembangan tanpa Oracle memperlihatkan layar
  yang tampak rusak padahal datanya benar.

## Kesalahan sendiri — properti kelas Pega bukan kolom view (2026-09-23)

Modul ini sempat gagal memuat dengan `ORA-00904: "TYPE_DOCUMENT": invalid identifier`.

**Sebabnya satu kekeliruan penalaran.** Daftar kolom view saya susun dari
`Report Definition/BrowseVLstDetTypeDoc_RD-RD.xml`, yang menyebut 13 properti kelas
`ASM-FW-GCNMFW-Int-V_LST_DET_TYPE_DOC` — termasuk `TYPE_DOCUMENT`. Saya memperlakukan daftar itu
sebagai daftar kolom view.

**Itu tidak benar.** Kelas Pega memetakan properti yang dapat diisi dari mana saja — termasuk
dari join yang dikerjakan rule lain, atau tidak diisi sama sekali. Katalog Oracle membuktikan
view-nya hanya punya 12 kolom, dan `TYPE_DOCUMENT` bukan salah satunya.

**Yang seharusnya dilakukan sejak awal**, dan kini menjadi kebiasaan: daftar kolom dibaca dari
`ALL_TAB_COLUMNS`/`ALL_VIEWS`, bukan disimpulkan dari Report Definition. Export Pega cukup untuk
mengetahui *apa artinya* sebuah kolom; ia tidak cukup untuk mengetahui *apakah kolom itu ada*.

Petunjuk yang sebenarnya sudah ada di tangan dan saya lewatkan: modul MENU_ID 42 — yang dibangun
sesi lain dan membaca view yang sama — **menjoin `V_LST_DOC_TYPE` sendiri** untuk mendapatkan
`TYPE_DOCUMENT`. Kalau view-nya sudah memuat kolom itu, join tersebut tidak akan ada. Preseden
yang berlawanan dengan dugaan sendiri layak diperiksa, bukan dilewati.

## Teknik yang terbukti berguna — urutan penelusuran sebelum menuduh basis data

Saat layarnya gagal, godaan terbesar adalah langsung menyimpulkan "ada objek yang belum diminta
ke DBA". Urutan yang dipakai justru membalik itu, dan tiga langkah pertamanya murah:

1. **Apakah rutenya ada?** `curl` → 401, bukan 404. Menyingkirkan dugaan binary basi dalam satu
   perintah.
2. **Penyimpanan apa yang sebenarnya dipakai?** `.env` menyebut `memori`, tetapi `needsOracle()`
   bernilai true karena adapter identitasnya `hcq`. Membaca nama variabel saja akan menyesatkan.
3. **Mode periksa milik aplikasi sendiri** — melaporkan objek dan galat Oracle yang persis.
4. Baru setelah itu, **katalog dibaca langsung** untuk mengetahui bentuk sebenarnya.

Langkah 3 dan 4 yang mengubah "mungkin ada yang kurang" menjadi "satu tabel bernama X belum ada,
dan ini DDL-nya". Perkakas sementara untuk langkah 4 dibuat, dijalankan sekali, lalu **dibuang** —
ia tidak ikut ter-commit.

---

## Sesi Inbox Compliance (2026-09-24)

Ketentuan dokumentasi menuntut pencatatan setiap pemakaian skill: namanya, alasannya, waktunya,
keluarannya, dan manfaatnya terhadap keputusan teknis.

### Skill yang dipanggil

**Tidak ada skill yang dipanggil lewat perkakas Skill pada sesi ini.** Yang dipakai adalah
**kosakata dan disiplin** dua skill Matt Pocock yang sudah menjadi bagian dokumen Steering, dan
pemakaiannya dicatat di bawah karena ia benar-benar mengubah keputusan.

### `codebase-design` — dipakai sebagai disiplin, bukan dipanggil

**Kapan:** saat memutuskan bentuk seam `Repo` modul ini.

**Apa yang diputuskan karenanya.** Seam `Repo` modul Inbox Admin berbunyi
`List(ctx, query) ([]WorkItem, error)` — mengembalikan SELURUH baris. Godaan terbesarnya adalah
menyalin bentuk itu demi keseragaman. Prinsip *"interface adalah permukaan pengujian"* dan
*"interface bicara dalam bahasa domain, bukan bahasa penyimpanan"* mengubahnya: bila paginasi
terjadi di basis data, maka `Pagination` adalah bagian dari PERTANYAAN yang diajukan pemanggil,
bukan detail yang disembunyikan. Bentuknya menjadi
`List(ctx, query, page) (Page, error)`.

**Manfaat nyata.** Menyalin bentuk lama akan membuat seam berbohong — pemanggil mengira ia menerima
seluruh baris, padahal repo memotongnya. Kebohongan itu baru terlihat saat seseorang menghitung
`len(items)` dan mengira itu totalnya.

Prinsip *"satu adapter berarti seam hipotetis, dua adapter berarti seam nyata"* juga dipakai
memeriksa setiap seam: `Repo` punya dua (sqlstore, memory), `Clock` punya dua (sistem, tetap).
Tidak ada seam yang dibuat tanpa dua pengisi.

### `domain-modeling` — dipakai menajamkan satu istilah yang nyaris salah

**Kapan:** saat membaca `Activity/GetInboxRegisterCompliance-Act.xml`.

**Apa yang ditemukan karenanya.** Disiplin *"silangkan pernyataan dengan kode, jangan terima nama
apa adanya"* membatalkan satu rencana yang sudah setengah jalan: parameter bernama `CARI1` dan
`CARI2` hendak diperlakukan sebagai kotak cari — karena di modul lain nama itu memang berarti itu.
Pembacaan langkah activity membuktikan keduanya argumen TANGGAL untuk `GETSELISIHJAM`.

**Manfaat nyata.** Tanpa itu, layar ini akan punya kotak cari yang tidak pernah ada di sistem lama,
dan backend akan punya parameter `cari` yang tidak menyaring apa pun. Keduanya akan lolos review —
karena tampak wajar — dan baru ketahuan saat uji kesetaraan gerbang 1.

Disiplin yang sama menangkap satu alias menyesatkan lagi: `"ComplianceRemark"` pada
`RDB List/BrowseClaimStudy-SQL.xml:85` **bukan** catatan Compliance melainkan
`SUM(TOTAL_CLAIM * CURRENCYVALUE)`. Ia berbeda dari `.ComplianceRemarks` milik layar ini hanya oleh
satu huruf, dan keduanya tidak berhubungan sama sekali. Catatannya masuk ke komentar `WorkItem`.

### Skill yang dipertimbangkan dan alasan tidak dipakai

| Skill | Kenapa tidak dipakai di sesi ini |
|---|---|
| `mattpocock-skills:tdd` | Uji ditulis berdampingan dengan kode dan acceptance criteria-nya sudah berupa bukti terbaca dari export. Siklus merah-hijau formalnya tidak menambah apa pun — meski perlu dicatat bahwa dua uji memang GAGAL lebih dulu dan menangkap kesalahan nyata, sehingga manfaat siklusnya tetap diperoleh |
| `mattpocock-skills:research` | Penghalang di sesi ini bukan pengetahuan yang dapat diteliti melainkan **artefak yang tidak ada**: daftar kolom `POOLDATA.T_CLAIM_COMPLIANCE_H`. Tidak ada sumber yang dapat dibaca untuk menggantikannya, dan mengarangnya dilarang |
| `mattpocock-skills:diagnosing-bugs` | Tidak ada bug yang perlu didiagnosis. Dua uji yang gagal gagal karena ekspektasinya salah, bukan karena kodenya |
| `mattpocock-skills:code-review` | Tidak ada perubahan pihak lain untuk ditinjau; seluruh kode sesi ini ditulis pada sesi ini |
| `mattpocock-skills:grilling` | Disiplinnya dipakai pada diri sendiri — setiap angka dan setiap nama kolom diuji ke sumbernya sebelum ditulis — tetapi tidak ada rangkaian pertanyaan berjenjang kepada Work Owner yang menuntut pemanggilan formalnya. Empat pertanyaan diajukan dan seluruhnya terjawab dalam satu putaran |

### Teknik yang terbukti berguna

**Memecah XML satu baris sebelum mencarinya.** Export Pega menulis XML tanpa baris baru, sehingga
pencarian biasa mengembalikan "baris 1" untuk segalanya. Memecahnya pada tanda tutup tag lebih dulu
membuat nomor baris menjadi bermakna — dan seluruh rujukan `berkas:baris` di dokumentasi modul ini
lahir dari situ.

**Menghitung sidik jari sebelum menyimpulkan.** Dugaan awal "harness 145 KB dengan RD 0 berarti
layar tanpa data" terbantah hanya setelah Report Definition dicari di dalam SECTION, bukan di
harness. Pelajarannya sama dengan yang sudah tercatat pada sesi sebelumnya: **ukuran berkas adalah
bukti korelasional, bukan pernyataan isi.**

**Menolak menebak nama kolom, lalu memindahkan risikonya ke perintah `-periksa`.** Dua kolom yang
keberadaannya disimpulkan tidak dibiarkan sebagai komentar peringatan; kueri daftarnya dijalankan
oleh mode periksa saat start. Ini mengubah "mungkin salah" menjadi "akan ketahuan sebelum pengguna
membukanya".

### Kesalahan sendiri yang tercatat sesi ini

Tiga, seluruhnya ditangkap uji atau pembacaan ulang — rinciannya di
[`catatan-pengembangan.md`](catatan-pengembangan.md) §25.6. Polanya satu dan sama dengan sesi
sebelumnya: **klaim yang terdengar masuk akal tidak diuji sebelum ditulis.** Yang berubah, kali ini
ketiganya tertangkap sebelum sampai ke Work Owner.

---

## Sesi Inbox Compliance, lanjutan — tab Post Audit (2026-09-24)

Tidak ada skill yang dipanggil. Dua disiplin yang dipakai dan benar-benar mengubah keputusan:

### `domain-modeling` — menyilangkan nama kolom dengan preseden, bukan menebaknya

**Kapan:** saat memetakan `CASEID` dan `NO_KLAIM` ke kolom layar.

Keduanya sama-sama masuk akal sebagai "Case ID". Alih-alih memilih yang namanya paling mirip,
preseden dicari lebih dulu di tabel sekerabat: `POOLDATA.T_CLAIM_PNC` memisahkan `CLAIMID` (kunci
teknis, 66 pemakaian di prosedur konversi) dari `CLAIMNO` (nomor yang dibaca orang, 8 pemakaian).
Lebar kolomnya menguatkan — `CASEID` VARCHAR2(100) sepanjang kunci Pega, `NO_KLAIM` bukan.

**Manfaat nyata.** Memilih `CASEID` sebagai kolom pertama akan menampilkan
`ASM-FW-GCNMFW-WORK PNC-…` kepada petugas — persis kebocoran kunci teknis yang `D-22` hapus.
Kesalahannya akan lolos review karena tampak wajar.

### `codebase-design` — menolak menyeragamkan yang memang tidak sekerabat

**Kapan:** saat memutuskan apakah kedua tab berbagi satu daftar alias.

Modul Inbox Admin memakai satu daftar 29 alias untuk ketujuh kuerinya. Menyalin polanya ke sini
terasa seperti konsistensi. Prinsip *"seam bicara dalam bahasa domain"* membalikkannya: kedua tab
membaca **tabel yang benar-benar berbeda**, bukan satu tabel dengan penyaring berbeda. Menyatukan
daftarnya hanya akan menambah enam `CAST(NULL AS …)` dan menyamarkan bahwa keduanya tidak
sekerabat.

Gantinya `plans` — satu nilai yang mengikat kueri, argumen, dan pemindai, sehingga ketiganya tidak
dapat berubah sendiri-sendiri. Ditambah `TestEveryTabHasPlan` yang menjaga setiap tab punya rencana
**dan** setiap rencana dimiliki tab; yang kedua menangkap kode mati.

### Teknik yang terbukti berguna — biarkan uji lama gagal dulu, jangan disunting duluan

Setelah `Tab.Available` dibalik, lima uji langsung merah. Godaannya menyunting kelimanya lebih dulu
lalu membalik penandanya. Urutan yang dipakai justru sebaliknya: **balik dulu, lihat apa yang
jatuh**.

Daftar yang jatuh itulah peta tempat yang perlu disesuaikan — dan ia menangkap satu hal yang tidak
terpikir: `repo/sqlstore/query_test.go` **gagal dikompilasi**, bukan sekadar gagal, karena
`resultColumns` dipecah dua. Menyuntingnya duluan akan melewatkan bahwa uji itu memang dirancang
untuk menolak perubahan bentuk alias.

### Kesalahan sendiri yang TIDAK terjadi, dan kenapa

Dicatat karena penangkalnya layak diulang: dua kali tergoda menebak. Pertama, menebak kolom status
yang "mungkin ada" supaya penyaring `pyStatusWork = "New"` dapat direplikasi. Kedua, menebak bahwa
`CASEID` adalah nomor klaim karena namanya mengandung "CASE".

Keduanya dihindari oleh aturan yang sama: **bila tidak ada di DDL atau di export, ia tidak ditulis
sebagai fakta.** Yang pertama menjadi kalimat keterbatasan yang tampil di layar; yang kedua
diputuskan lewat preseden lalu dicatat sebagai pertanyaan terbuka.

---

## Sesi Inbox Compliance, koreksi — tab Post Audit (2026-09-24)

Tidak ada skill yang dipanggil. Satu teknik dan satu kesalahan yang layak dicatat.

### Teknik: layar yang berjalan mengalahkan penalaran atas artefak

Tiga keputusan yang saya ambil dari pembacaan artefak dibetulkan oleh satu tangkapan layar Pega:
arah `CASEID`/`NO_KLAIM`, jumlah kolom, dan ada-tidaknya jam pada kolom tanggal.

Ketiganya diputuskan dengan penalaran yang sah — preseden tabel sekerabat, label Report Definition,
kebiasaan kolom tanggal di modul lain. Ketiganya tetap salah. Yang membetulkannya bukan penalaran
yang lebih teliti, melainkan **melihat sistemnya berjalan**.

Konsekuensi untuk modul berikutnya: bila layar Pega dapat dibuka, ia diminta LEBIH DULU untuk
layar yang kolomnya tidak dapat dipastikan dari export — bukan dipakai belakangan sebagai
pemeriksa.

### Kesalahan: pencarian nol hasil dibaca sebagai "datanya tidak ada"

Judul kolom tab Post Audit dicari lewat elemen `pyLabelFieldValue`. Hasilnya nol, dan saya
menyimpulkan section itu tidak memuat judul — lalu jatuh ke label Report Definition sebagai
gantinya. Judulnya ternyata ada, tersimpan sebagai `pyCaption`, dan langsung terlihat begitu
dicari sebagai teks biasa.

Aturan yang ditarik: **pencarian berbasis nama elemen yang mengembalikan nol hasil adalah sinyal
elemennya salah, bukan sinyal datanya tidak ada.** Pemeriksa silangnya murah — cari teks yang
diharapkan muncul, bukan nama wadahnya.

Ini kesalahan yang sepola dengan dua sesi sebelumnya (`Cell.Width` yang tidak bergerak, penanda
`worklist` yang menangkap boilerplate): **alat ukur dipercaya sebelum divalidasi pada kasus yang
jelas benar.**

### Yang berhasil dihindari

Dua kali tergoda menebak, dan keduanya ditahan aturan yang sama — bila tidak ada di DDL atau di
export, ia tidak ditulis sebagai fakta:

- Kolom status pengganti `pyStatusWork = "New"`, yang tidak ada di tabelnya. Menjadi kalimat
  keterbatasan yang tampil di layar.
- Tangga satuan penuh kolom OutStanding. Hanya cabang tahun yang teramati; sisanya ditulis sebagai
  rekonstruksi, dan statusnya disebut di komentar fungsinya maupun di nama kasus ujinya.

---

## Sesi Inbox Compliance, koreksi kedua (2026-09-24)

Tidak ada skill yang dipanggil. Satu teknik yang terbukti, dan satu pola kesalahan yang kini
berulang tiga kali.

### Teknik: perlakukan "coba cek lagi" sebagai perintah menyisir ulang

Dua dari tiga jawaban Work Owner berbunyi "sesuaikan sama aplikasi PEGA aja" dan "coba di cek
lagi". Keduanya dapat dibaca sebagai persetujuan atas yang sudah ada. Yang dipilih adalah bacaan
sebaliknya: menyisir ulang export.

Hasilnya dua temuan yang tidak akan muncul dari membaca ulang kode sendiri:

- `pyDateTimeFormat = DateTime-Frame` — pemformat OutStanding ternyata bawaan platform Pega, bukan
  rule yang hilang. Itu sekaligus menjelaskan kenapa pencarian teksnya selalu nol hasil.
- Urutan baris di layar Pega adalah urutan TEKS, bukan tanggal. Diperiksa dengan mengurutkan
  ketujuh nomor case dari tangkapan layar dan membandingkannya — cocok persis.

Yang kedua tidak dicari sama sekali. Ia muncul karena tangkapan layar dibaca sebagai DATA yang
dapat diuji, bukan sebagai ilustrasi.

### Pola kesalahan yang kini berulang tiga kali

| Sesi | Bentuknya |
|---|---|
| Dua sesi lalu | `Cell.Width` dipercaya walau angkanya tidak bergerak saat masukannya berubah |
| Sesi lalu | penanda `worklist` dipercaya walau tidak menyala pada kasus yang jelas benar |
| Sesi ini | `TGL_KIRIM_POST_AUDIT DESC` dipilih karena "paling masuk akal bagi antrean", bukan karena terbukti |

Ketiganya satu pola: **memilih yang masuk akal alih-alih yang terbukti.** Yang berbeda kali ini,
penangkalnya sudah tersedia dan sempat tidak dipakai — tangkapan layar Pega ada di tangan sebelum
kueri ditulis.

Aturan yang ditarik untuk modul berikutnya: bila layar sistem lama dapat dilihat, **urutan baris
dan bentuk teks dibaca dari sana lebih dulu**, bukan disimpulkan dari Report Definition. RD
menyatakan apa yang diminta; layar menyatakan apa yang keluar.

### Yang berhasil dihindari

Menambahkan sesuatu supaya "JONNY bisa". Jawaban Work Owner mudah dibaca sebagai permintaan
membuka akses, padahal pemeriksaan menunjukkan modul ini memang belum punya gerbang peran sama
sekali dan JONNY sudah melihat menunya. Menambahkan apa pun justru akan memperkenalkan
ketergantungan pada identitas yang sekarang tidak ada — kebalikan dari "jangan hardcode".

---

## Sesi Inbox Compliance, jalur tulis (2026-09-24)

Tidak ada skill yang dipanggil. Satu teknik dan satu batas yang layak dicatat.

### Teknik: ketika alurnya hilang, bentuk tabelnya yang bercerita

`Compliance_Flow` tidak ada di export, sehingga "seperti aplikasi Pega" tidak dapat dibaca dari
alurnya. Yang dipakai sebagai gantinya: **keenam kolom tabelnya dibandingkan satu per satu dengan
kolom yang sudah tampil di tab Compliance.** Kelima di antaranya cocok, dan yang tersisa hanya
catatan dan tanggal kirim.

Bentuk itu hanya masuk akal bila barisnya lahir dari sisi Compliance. Jadi skema tabel dapat
menyatakan alur meski flow-nya hilang — asalkan yang dibaca adalah kecocokannya dengan layar yang
ada, bukan nama kolomnya.

### Batas yang diakui, bukan disamarkan

Prinsip kerja proyek ini melarang dummy logic **jika proses aslinya dapat dipelajari**. Di sini ia
tidak dapat. Yang dilakukan bukan berpura-pura sudah mempelajarinya, melainkan:

- menulis di kepala berkasnya apa yang terbaca dan apa yang disimpulkan;
- memilih perilaku yang paling sedikit mengarang pada setiap persimpangan — catatan boleh kosong,
  status klaim tidak diubah, tidak ada notifikasi;
- mencatat keempatnya sebagai keputusan, bukan sebagai detail pelaksanaan.

### Yang berhasil ditahan

Tiga godaan menambahkan aturan yang terdengar masuk akal:

| Godaan | Kenapa ditahan |
|---|---|
| Mewajibkan Catatan diisi | Kolomnya nullable; tidak ada bukti ia wajib |
| Menolak klaim yang sudah pernah dikirim | Di Pega satu klaim tampaknya boleh punya lebih dari satu pemeriksaan |
| Mengubah status klaim setelah dikirim | Itu MENULIS tabel Pega — melanggar `P-1`, bukan sekadar mengarang |

Yang ketiga menunjukkan hal yang berguna: sebagian aturan yang tergoda ditambahkan ternyata bukan
hanya tidak berdasar, melainkan juga terlarang. Memeriksa "tabel siapa yang akan tersentuh" adalah
penyaring yang lebih cepat daripada memeriksa "ada buktinya atau tidak".
