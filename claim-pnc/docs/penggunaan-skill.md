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

# Penggunaan Skill — Sesi 2026-09-19 (modul Inbox Auto Claim)

## Ringkasan

| | |
|---|---|
| Permintaan | Menambah modul **Inbox Auto Claim**, dengan `InboxAutoClaim-Harness.xml` sebagai rujukan |
| Skill yang benar-benar dipakai | **`codebase-design`** · **`domain-modeling`** · disiplin **`grilling`** pada diri sendiri |
| Skill yang ditimbang dan tidak dipakai | `tdd`, `prototype`, `research`, `diagnosing-bugs`, `code-review`, `wizard`, `writing-for-agents`, `dataviz` |
| Keluaran | 1 modul backend (8 berkas + 17 kueri), 1 modul frontend (4 berkas), 4 berkas uji, 3 dokumen |

---

## `codebase-design` — dipakai paling berat

**Kapan.** Sebelum satu berkas Go ditulis, saat memutuskan bentuk seam `Repo` dan
pembagian tanggung jawab antara domain, usecase, dan transport.

**Kenapa skill ini yang dipakai.** Modul ini punya satu pertanyaan rancangan yang tidak
dimiliki ketiga modul master sebelumnya: **ekspor CSV milik siapa?** Ia terasa seperti
urusan HTTP (Content-Type, Content-Disposition), tetapi judul kolom dan urutannya adalah
**ketetapan bisnis** yang disalin dari Pega.

**Yang dihasilkan — tiga batas yang ditarik dengan alasan, bukan kebiasaan:**

| Batas | Letaknya | Alasan yang dipakai skill |
|---|---|---|
| Bentuk berkas CSV | **domain** (`export.go`) | Judul kolomnya aturan bisnis. Di transport ia tidak dapat diuji tanpa menyalakan server |
| Penyusunan berkas | **usecase** | Ia orkestrasi: pilih spec, baca baris, rangkai. Bukan aturan, bukan HTTP |
| Header dan nama berkas | **transport** | Satu-satunya bagian yang benar-benar tentang HTTP |

**Uji deletion yang dijalankan pada seam `Repo`.** Pertanyaannya: kalau `ExportLine`
dihapus dan ekspor memakai `ListLine` berulang, apa yang muncul kembali di pemanggil?
Jawabannya: pengulangan halaman, dan itu memang muncul di `sqlstore.ExportLine`. Tetapi ia
muncul **satu kali di adapter**, bukan di usecase maupun di layar — jadi seam-nya tetap
dibenarkan. Penyimpanan memori mengisinya dengan satu baris karena di sana memang tidak
ada halaman.

**Prinsip "satu adapter berarti seam hipotetis".** Seam `Repo` punya dua adapter nyata —
`sqlstore` dan `memory` — dan yang kedua bukan sekadar pelengkap: ia yang membuat 22 uji
usecase berjalan tanpa Oracle.

---

## `domain-modeling` — dipakai menamai ulang delapan kolom

**Kapan.** Saat membaca pemetaan kolom grid ke properti Pega.

**Kenapa.** Layar ini contoh utang teknis §4.2 yang paling telanjang yang ditemui sejauh
ini. Kedelapan kolomnya terikat ke properti yang namanya **tidak ada hubungannya sama
sekali** dengan isinya:

```
Nama Perusahaan  -> .AlasanTerlambat
Batch            -> .CauseOfLoss
Jumlah Gagal     -> .ClaimID
User Upload      -> .AnaylstRemarks   (salah ketik "Anaylst" pun ikut)
```

**Disiplin skill yang dipakai: menyilangkan pernyataan dengan kode.** Nama barunya tidak
dipilih dari selera — tiap nama diuji terhadap apa yang benar-benar diisi kueri. Contoh
konkret yang tertangkap karenanya: `.CauseOfLoss` pada grid adalah **nomor batch**,
sementara `COL_ID` pada tabel yang sama adalah **penyebab kerugian yang sungguhan**. Dua
hal berbeda dengan satu nama Pega, dan menyalinnya apa adanya akan menghasilkan struct
yang punya dua field bernama mirip untuk hal yang tidak berhubungan.

**Satu istilah yang sengaja TIDAK dinamai ulang.** Judul kolom kedua berkas CSV
dipertahankan apa adanya, termasuk yang menyesatkan ("No Objek" berisi nomor produk).
Alasannya batas kepemilikan: berkas itu dibaca **perusahaan rekanan di luar Sinarmas**,
sehingga judulnya kontrak keluaran, bukan nama internal. Penamaan ulang berhenti di batas
berkas.

---

## Disiplin `grilling` — dipakai pada diri sendiri, bukan pada Work Owner

**Kapan.** Sepanjang pembacaan export, sebelum setiap angka dipakai.

Skill-nya tidak dipanggil sebagai perintah; yang dipakai aturannya: **cari faktanya
sendiri, dan jangan pakai angka yang belum diperiksa ke sumbernya.**

Dua hal yang tertangkap karenanya, dan keduanya akan menjadi kesalahan yang mahal:

| Yang nyaris dipakai | Apa yang sebenarnya benar |
|---|---|
| `ValidationInitial_act` sebagai bukti aturan validasi **unggahan** | Ditelusuri pemanggilnya: ia milik `InsertMstAutoClaim_act` — layar **Master Auto Claim**, memeriksa inisial baru tidak kembar. Sama sekali bukan jalur unggah |
| Pemetaan kolom CSV milik jalur **Asuransi Kredit** disalin ke Auto Claim | `TMP_BATCH_AUTO_CLAIM` **tidak punya** kolom `NOASURANSI` yang dipakai jalur kredit. Salinannya akan menghasilkan kueri yang gagal — atau lebih buruk, kolom yang diisi nilai yang salah |

**Yang juga datang dari disiplin ini: tiga pertanyaan ke Work Owner diajukan dengan angka,
bukan dengan "bagaimana menurut Bapak".** Pertanyaan tentang kueri yang hilang menyebut
keenam namanya dan menyatakan sudah dicari ke 2.634 berkas; pertanyaan lingkup menyebut
tujuh tombol beserta modul yang menahan masing-masing.

---

## Skill yang ditimbang dan alasan tidak dipakai

| Skill | Alasan |
|---|---|
| `tdd` | Uji ditulis **sesudah** perilakunya jelas dari export, bukan sebelum. Pada modul migrasi, "merah dulu" menuntut mengetahui hasil yang benar — dan di sini hasil yang benar justru yang sedang dicari dari XML. Uji tetap ditulis lengkap; yang tidak dipakai urutannya |
| `prototype` | Tidak ada pertanyaan rancangan yang butuh dijawab dengan kode buangan. Ketiga pertanyaan yang tersisa dijawab Work Owner, bukan oleh percobaan |
| `research` | Seluruh fakta ada di dalam repository. Tidak satu pun klaim di sesi ini bersumber dari luar |
| `diagnosing-bugs` | Tidak ada cacat yang sedang dikejar. Cacat sistem lama yang ditemukan (kegagalan senyap kode perusahaan) dicatat sebagai keputusan, bukan didiagnosis |
| `code-review` | Tidak ada perubahan orang lain untuk ditinjau |
| `wizard` | Tidak ada langkah yang hanya dapat dijalankan manusia |
| `writing-for-agents` | Dokumen sesi ini untuk dibaca manusia |
| `dataviz` | Layar ini tabel dan angka ringkas, bukan grafik. Pega pun memuat `pxChart` di section-nya, tetapi jalur Auto Claim tidak memakainya |

---

## Teknik yang dipakai tanpa memanggil skill

**Membedah XML 1,66 MiB dengan skrip, bukan membacanya.** Harness-nya terlalu besar untuk
dibaca manusia maupun dimuat utuh. Yang dipakai: skrip Node yang mengekstrak urutan
kemunculan tag tertentu, lalu mencocokkan judul kolom dengan properti **berdasarkan
urutannya di dokumen**. Itulah yang menghasilkan pemetaan delapan kolom — mustahil didapat
dengan membaca berurutan.

**Silang-periksa ke seluruh export sebelum menyatakan sesuatu hilang.** Setiap nama rule
yang tampak hilang dicari ke **seluruh 2.634 berkas**, bukan hanya ke folder yang terduga.
Itu yang membuktikan keenam kueri benar-benar tidak ada — dan juga yang menemukan bahwa
sepupunya untuk dua lini lain ikut hilang, sehingga polanya menjadi jelas.

**Menguji terhadap aplikasi yang benar-benar berjalan.** 16 permintaan HTTP ke instans
sementara, termasuk jalur galat. Dua di antaranya tidak akan tertangkap uji unit: bahwa
unggahan bercacat tidak menyisakan satu baris pun, dan bahwa baris hasil unggahan
benar-benar bertanda belum diproses.

---

## Satu hal yang saya siapkan tetapi tidak jadi dipakai

Sempat disiapkan pemeriksaan **`tgllapor >= tglkejadian`** pada unggahan — aturan `I-2`
yang memang berlaku pada registrasi klaim.

Tidak jadi dipakai. Alasannya `P-5`: jalur unggah sistem lama tidak memeriksanya, dan
pelanggarannya ditangani **saat pembuatan case** sebagai kegagalan baris dengan pesannya
sendiri. Menambahkannya di sini akan menolak berkas yang sistem lama terima, lalu muncul
di uji kesetaraan sebagai selisih yang tidak ada di daftar 13 butir `D-49`.

Aturan yang benar di tempat yang salah tetap merupakan perubahan perilaku.

---

## Catatan untuk sesi berikutnya

1. **Modul ini belum dapat lulus gerbang 1.** Keenam kueri Pega yang hilang adalah
   penghalangnya, dan itu ada di tangan Tim Pega.
2. **Dua kolom CSV masih dugaan.** Bila kueri aslinya tiba, keduanya yang pertama
   diperiksa — tujuh kolom lain sudah pasti.
3. **`DataTable` sekarang punya paginasi server.** Layar inbox berikutnya memakainya
   langsung; tidak perlu membangun ulang.
4. **`callAPI` sekarang menerima `FormData` dan ada `unduhBerkas`.** Modul mana pun yang
   butuh unggah atau unduh memakai keduanya, bukan memanggil `fetch` sendiri.

---

# Sesi kesembilan lanjutan — 2026-09-19

Artefak Pega yang diminta sesi sebelumnya tiba seluruhnya, dan pekerjaannya berubah sifat: dari
**merekonstruksi** menjadi **membandingkan rekonstruksi dengan buktinya**. Perbedaan itu
menentukan skill mana yang berguna.

## Skill yang dipakai

### `mattpocock-skills:grilling` — dipakai pada diri sendiri

**Kapan.** Sepanjang pembacaan 20 berkas baru, sebelum satu baris kode pun disunting.

**Kenapa skill ini.** Godaan terbesar ketika artefak akhirnya tiba adalah membaca sekilas,
menemukan yang cocok, lalu menyimpulkan "rekonstruksinya sudah benar". Disiplin grilling
membalik arahnya: **tiap dugaan diuji untuk dipatahkan**, bukan untuk dibenarkan.

**Yang dihasilkan.** Enam dugaan patah, tiga bertahan. Yang paling penting — butir `TGLPROSES` —
tidak ditemukan dengan membaca kueri melainkan dengan membaca **DDL**, bagian yang paling
menggoda untuk dilewati karena "hanya definisi tabel".

**Manfaat nyata bagi proyek.** Tanpa disiplin ini, modulnya akan lulus seluruh uji, lulus
tinjauan, lalu **gagal pada unggahan pertama di Oracle** dengan ORA-01400 — jenis kegagalan yang
baru muncul di tangan pengguna.

### `mattpocock-skills:domain-modeling` — untuk memisahkan tiga keadaan hasil unggahan

**Kapan.** Saat merancang `UploadResult`, setelah membaca `InsertKlaimToTable_Other`.

**Yang dipertajam.** Awalnya saya punya dua keadaan: berhasil dan gagal. Membaca activity-nya
memperlihatkan **tiga** yang berbeda akibatnya:

| Keadaan | Barisnya di tabel | Terlihat di grid | Yang harus dilakukan pengguna |
|---|---|---|---|
| lolos | ada | ya | menunggu pemrosesan |
| bertanda | ada | ya, sebagai gagal | memperbaiki datanya |
| ditolak | **tidak ada** | **tidak** | mengunggah ulang barisnya |

Ketiganya diberi nama yang tidak dapat tertukar — dan nama field-nya sengaja **tidak** memakai
kata "berhasil"/"gagal", karena kata itu sudah dipakai untuk hasil pemrosesan menjadi klaim.
Memakainya dua kali untuk dua hal berbeda persis kesalahan yang `D-19` suruh hindari.

**Manfaat.** Panel hasil unggahan di layar kini membedakan ketiganya. Menggabungkan yang kedua
dan ketiga akan membuat pengguna **mencari baris yang tidak pernah tersimpan**.

### `mattpocock-skills:codebase-design` — untuk bentuk seam `Repo`

**Kapan.** Saat rantai pemeriksaan polis harus ditempatkan.

**Keputusan yang dihasilkan.** `ResolveReceiver` dan `FindPolicyProductSeq` menjadi **dua method
terpisah** di seam `Repo`, bukan satu `ResolvePolicy` yang mengembalikan segalanya. Sebabnya
bukan kerapian: di Pega keduanya **dua kueri ke dua tabel berbeda**, dan kegagalannya
menghasilkan **pesan yang berbeda** serta **akibat yang berbeda** — yang pertama menolak baris,
yang kedua menandainya. Menyatukannya akan menghapus pembedaan itu di dalam adapter, tempat yang
paling sulit diperiksa.

Rantainya sendiri hidup di **lapisan aplikasi** (`usecase.resolve`), bukan di adapter: ia
urutan aturan bisnis, dan adapter hanya menjawab pertanyaan data.

## Skill yang ditimbang dan tidak dipakai

| Skill | Alasan |
|---|---|
| `tdd` | Perubahan sesi ini sebagian besar **membalik uji yang sudah ada** karena premisnya terbukti salah. Menulis uji lebih dulu untuk perilaku yang baru saja terbukti tidak menambah apa pun — buktinya sudah ada di berkas XML-nya |
| `diagnosing-bugs` | Tiga uji `master-rekening` yang gagal memang cacat nyata, tetapi **Isolasi Protektif** melarang saya menyentuh modul itu. Yang saya lakukan hanya membuktikan penyebabnya bukan dari sesi ini |
| `research` | Seluruh fakta ada di dalam repository |

## Teknik tanpa skill

**Membuktikan kegagalan bukan milik saya sebelum melaporkannya.** Tiga uji `master-rekening`
gagal. Alih-alih menduga, saya men-stash ketiga berkas bersama yang saya sentuh lalu menjalankan
ulang uji itu — ketiganya tetap gagal. Baru setelah itu saya menyatakannya pre-existing.

Ini penting justru karena mudah disalahgunakan ke arah sebaliknya: "bukan salah saya" adalah
kalimat yang harus **dibuktikan**, bukan diasumsikan.

**Menguji terhadap server sungguhan dengan berkas yang mencakup setiap keadaan.** Berkas
unggahan uji berisi lima baris yang masing-masing memicu jalur berbeda — nomor polis bertitik,
polis perusahaan lain, polis tanpa JSON, polis tanpa perusahaan, dan tanggal terbalik. Satu
perintah membuktikan kelima jalurnya sekaligus, dan tiga di antaranya **tidak dapat dibuktikan
uji memori** karena menyangkut pencarian polis.

## Satu kesalahan metode yang patut dicatat

Sesi lalu saya menulis tiga uji yang **menegakkan dugaan sebagai aturan**. Ketiganya lulus, dan
ketiganya salah.

Yang menyelamatkan bukan uji yang lebih banyak, melainkan **komentar di dalam uji itu yang
menyebut dugaannya sebagai dugaan**. Orang berikutnya — dalam hal ini saya sendiri — langsung
tahu uji mana yang boleh dicabut begitu buktinya datang.

Kesimpulannya bukan "jangan menulis uji untuk hal yang belum pasti". Uji seperti itu tetap
berguna: ia mengunci perilaku supaya tidak berubah tanpa sengaja. Yang wajib adalah
**menyebutkan dasarnya di dalam uji itu sendiri**.

## Catatan untuk sesi berikutnya

1. **Modul ini sudah setara dengan artefaknya**, tetapi gerbang 1 tetap belum dapat dijalankan —
   penghalangnya kini **Pega staging**, bukan kueri yang hilang.
2. **Tab Asuransi Kredit dan Travel dapat dikerjakan kapan saja.** Kuerinya seluruhnya ada
   (`*_AsuransiKredit`, `*_Travel`), pola sama persis dengan Auto Claim. Ia perubahan lingkup,
   bukan koreksi — perlu persetujuan Work Owner.
3. **`CURRENCY` menunggu `B-1`.** Saat modul snapshot polis ada, satu tempat yang berubah:
   `usecase.resolve`.
4. **Tiga uji `master-rekening` masih merah** dan bukan dari sesi ini. Perlu diputuskan Work
   Owner kapan diperbaiki.
5. **Penyimpanan memori tidak dapat menangkap pelanggaran constraint.** Sebelum modul berikutnya
   menulis ke tabel warisan, baca DDL-nya lebih dulu — bukan kuerinya saja.

---

# Sesi kesepuluh — 2026-09-20

Sesi perbaikan cacat yang dilaporkan langsung dari layar, lalu perluasan menjadi tiga tab. Sifat
pekerjaannya berbeda dari sesi mana pun sebelumnya: **yang dilaporkan bukan "belum dibangun",
melainkan "dibangun tetapi tidak berfungsi"** — dan itu menentukan skill mana yang berguna.

## Skill yang dipakai

### `mattpocock-skills:diagnosing-bugs`

**Kapan dipanggil.** Setelah laporan *"filter perusahaan belum berfungsi saat klik pada donut
maupun baris tabel"*, ketika uji komponen yang ada **seluruhnya hijau** dan backend sudah
terverifikasi terhadap Oracle.

**Kenapa skill ini, bukan langsung membaca kode.** Justru karena semuanya hijau. Keadaan
"uji lulus tetapi pengguna melihat cacat" adalah keadaan paling menjebak: godaannya membaca kode
sampai menemukan sesuatu yang *terlihat* mencurigakan, lalu mengubahnya dan berharap. Skill ini
melarang persis itu — **tanpa satu perintah yang dapat merah, tidak boleh ada hipotesis.**

**Yang dihasilkan, dan ini bagian terpentingnya.** Loop pertama yang saya bangun **HIJAU**,
padahal cacatnya nyata. Ia mengeklik tombol nama — satu-satunya tempat yang memang sudah
berfungsi. Pengguna mengeklik **angkanya**.

Fase 1 skill ini menuntut loop yang *red-capable*: bukan "berjalan tanpa galat", melainkan
**mampu merah pada cacat ini**. Loop pertama gagal memenuhi syarat itu, dan tanpa syaratnya saya
akan menyimpulkan "tidak ada cacat" dari bukti yang tidak menguji apa pun.

Setelah loop diperlebar untuk mengeklik sel angka → **merah** → sebabnya langsung terbaca:
barisnya menyala saat disentuh kursor, tetapi hanya teks namanya yang berupa tombol.

**Fase 5 dijalankan penuh pada ketiga cacat**: uji ditulis lebih dulu, dibuktikan **merah** dengan
memasukkan kembali cacatnya, baru diperbaiki.

| Cacat | Cara membuktikan uji-nya merah |
|---|---|
| ORA-01008 | Menomori ulang bind ke `:1` dua kali → uji gagal |
| Penyaring tidak cocok | Memasang kembali `strings.ToUpper` → uji gagal |
| Berpindah tab membawa penyaring lama | Menghapus `setCompany('')` → uji gagal |

**Fase 6 dijalankan.** Seluruh instrumentasi bertanda `[DEBUG-a4f2]` dan `[DEBUG-b7c1]` dihapus,
dan `grep` prefiksnya dijalankan untuk memastikannya.

### `mattpocock-skills:codebase-design`

**Kapan.** Saat memutuskan bentuk tiga tab, sebelum satu berkas pun ditambahkan.

**Kenapa.** Pega menyelesaikannya dengan **menggandakan setiap rule tiga kali**. Pertanyaannya
bukan "apakah menyalin tiga kali itu jelek" — melainkan **di mana seam-nya**.

Kosakata skill ini menjawabnya: yang bervariasi antar tab **hanya dua nilai** (nama tabel dan
nama kolom perusahaan), sementara seluruh bentuk kueri, paginasi, pengelompokan, dan pemetaan
kolomnya sama. Seam-nya karena itu bukan "tiga repo", melainkan **satu cetakan dengan dua titik
substitusi** — dan `Source` menjadi enum tertutup, bukan string bebas, supaya substitusinya tidak
pernah berasal dari masukan pengguna.

Prinsip **"satu adapter berarti seam hipotetis"** juga dipakai ke arah sebaliknya: saya
**tidak** membuat antarmuka `SourceStrategy` beserta tiga implementasinya. Tiga nilai dalam satu
peta sudah cukup, dan abstraksi tambahan hanya menambah tempat yang harus dibaca.

### `mattpocock-skills:domain-modeling`

**Kapan.** Saat menamai tab.

**Yang dihasilkan.** Label tab **tidak** diturunkan dari nama rule. Rule-nya bernama
`BrowseClaimSPKAutoClaim`, tetapi `pyCaption` harness menyebut **ANEKA**. Menurunkan nama dari
rule menghasilkan tab yang salah nama di mata pengguna — persis kelas kesalahan yang `D-19`
larang: membawa alias internal Pega ke permukaan yang dilihat manusia.

Karena itu pula label dan nama tabel **dikirim server**, bukan diketik di layar: keduanya
pengetahuan tentang sistem lama, dan itu milik backend.

## Skill yang ditimbang dan tidak dipakai

| Skill | Kenapa tidak |
|---|---|
| `tdd` | Dipakai **semangatnya** pada ketiga perbaikan cacat, tetapi jalurnya `diagnosing-bugs` fase 5 — bukan fitur baru yang dibangun dari uji kosong |
| `grilling` | Tidak ada rencana yang perlu ditekan; yang ada laporan cacat dengan gejala yang jelas |
| `code-review` | Perubahannya saya tulis sendiri dalam satu sesi dan sudah dijaga lint, `tsc`, dan uji |
| `dataviz` | Ditimbang untuk donut. Bentuk dan palet sudah ditetapkan **contoh dari Work Owner** dan sistem desain proyek; yang saya pastikan hanyalah aturannya yang memang mengikat di sini — pembedaan tidak pernah hanya warna (irisan terpilih ditandai **garis tepi**), dan grafiknya `aria-hidden` dengan **tabel** sebagai sumber resmi |

## Satu kesalahan metode yang patut dicatat

Saya melaporkan *"[ok] ringkasan per perusahaan berjalan: 2 perusahaan"* sebagai **berhasil**.
Angkanya benar — dua perusahaan memang punya batch — tetapi yang diharapkan Work Owner adalah
seluruh perusahaan master.

Pelajarannya: **"kuerinya mengembalikan sesuatu" bukan "kuerinya mengembalikan yang benar".**
Verifikasi yang hanya memeriksa ketiadaan galat akan meloloskan jawaban yang salah dengan tenang.
Sejak itu `-periksa` tidak lagi hanya menyebut jumlah, melainkan **membandingkan** angka ringkasan
dengan total grid setelah disaring — pemeriksaan yang dapat gagal.

## Catatan untuk sesi berikutnya

1. **Gerbang 1 tetap belum dapat dijalankan.** Penghalangnya masih **Pega staging**, tidak
   berubah dari sesi sebelumnya — dan kini berlaku untuk **ketiga** tab.
2. **Panel ringkasan tidak punya baseline Pega.** Komponennya tidak ada di export; ia kemampuan
   baru dan tidak dapat diuji kesetaraannya.
3. **Endpoint `/inbox-auto-claim/perusahaan` kini tanpa pemanggil.** Perlu diputuskan Work Owner:
   dipertahankan sebagai permukaan API, atau dihapus.
4. **Berkas JS terpaket melewati 500 kB** sejak Recharts masuk. Pemecahan kode belum pernah
   diputuskan untuk aplikasi ini.
5. **Tiga uji `master-rekening` masih merah**, tetap bukan dari sesi ini.
6. **Jangan pernah menimpa `PENYIMPANAN` saat menyalakan server.** `backend/.env` menyetel
   `memori`; menimpanya dengan `oracle` membuat login gagal karena `CPNC_SESI_AKTIF` tidak ada di
   skema itu. Timpaan itu hanya untuk `-periksa`, yang tidak mendengarkan porta.
