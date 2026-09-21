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
# Penggunaan Skill — Sesi 2026-09-19 (modul Master Tipe Surveyors)

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
