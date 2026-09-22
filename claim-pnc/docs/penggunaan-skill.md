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
