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

# Penggunaan Skill — Sesi 2026-09-17 (Penjenjangan Komite)

## Ringkasan

| Skill | Dipakai | Kapan |
|---|---|---|
| `mattpocock-skills:grilling` | **ya**, disiplinnya | sebelum menulis kode, untuk menguji premis tugas |
| `mattpocock-skills:domain-modeling` | **ya** | saat menamai `Ambang`, `Penyetuju`, `Penjenjangan`, dan `Pita` |
| `mattpocock-skills:codebase-design` | **ya** | saat memutuskan letak seam dan batas paket |
| lainnya | tidak | alasannya di bawah |

## `grilling` — menguji premis sebelum mengerjakan

**Kenapa dipakai.** Tugasnya berbunyi "lanjutkan penambahan modul Flow Komite", dan
`Flow/Komite_Flow.xml` disebut sebagai rujukan. Disiplin skill ini — **cari faktanya
sendiri, jangan tanyakan yang dapat dibaca; ajukan pertanyaan hanya untuk hal yang
benar-benar keputusan** — yang membuat tiga hal muncul sebelum satu baris kode ditulis:

1. Flow-nya hanya empat shape; aturannya ada di rule lain, dan rule itu yang harus dibaca.
2. `AutoAcceptKomite` ternyata bersyarat **lebih dari dua hari menganggur**, bukan
   "menyetujui semua tiap hari" seperti yang tertulis di tiket.
3. `B-7` bergantung pada dua modul yang belum ada, sehingga cakupannya adalah keputusan
   Work Owner — bukan sesuatu yang boleh saya putuskan sendiri.

**Yang dihasilkan.** Empat pertanyaan konfirmasi, masing-masing dengan pilihan jawaban dan
satu rekomendasi. Keempatnya dijawab, dan jawabannya mengubah bentuk pekerjaan secara
nyata — terutama jawaban 2 (baca saja), yang membuat seluruh kueri modul ini tidak memuat
satu pun pernyataan tulis.

**Manfaat yang terukur.** Tanpa disiplin ini, kemungkinan terbesarnya adalah membangun
layar keputusan komite di atas entitas Klaim yang dikarang — persis yang dilarang
"No Shortcuts", dan baru ketahuan setelah `B-5` dibangun dan bentuknya ternyata berbeda.

**Satu hal yang ditangkap disiplin ini pada diri saya sendiri.** Ketujuh kasus jumlah
penyetuju pada `spec.md` mula-mula hendak saya percayai apa adanya. Menghitungnya ulang
dengan tangan dari `Database/emailkomite.csv` bukan sekadar memastikan — ia yang
memunculkan temuan bahwa kalimat `TKT-B07-001` tentang "tidak punya penyetuju sama sekali"
**tidak benar** di bawah aturan kumulatif.

## `domain-modeling` — menajamkan istilah sebelum menamai tipe

**Kenapa dipakai.** Modul ini memperkenalkan empat istilah baru sekaligus, dan salah satunya
— `TYPE_KOMITE` — terbukti **memikul dua arti yang berbeda** di kolom yang sama: pita nilai
di Non-MBU, varian jalur di PA. Menamainya tanpa menyadari itu akan memasang kesalahpahaman
ke dalam tipe data.

**Yang dihasilkan:**

| Istilah sistem lama | Nama di sini | Alasan |
|---|---|---|
| satu baris `EMAILKOMITE` | **Ambang** | yang dimodelkan adalah ambang nilai, bukan alamat surel yang kebetulan ikut tersimpan di sana |
| `DEGREE` | **Jenjang** | dan didokumentasikan tegas bahwa ia **penentu urutan, bukan jumlah** — ia boleh berulang dan boleh melompat |
| `KomiteLoop` / `pxResultCount` | **JumlahJenjang** | nama lamanya menggambarkan mekanisme perulangannya, bukan artinya bagi bisnis |
| `TYPE_KOMITE` | **JenisKomite**, dengan dua arti didokumentasikan | dibiarkan mentah dengan sengaja; menamainya "Pita" akan berbohong untuk lini PA |

**Manfaat.** Field `JenisKomite` membawa komentar yang menyebutkan kedua artinya beserta
buktinya dari data (pada PA nilainya berselang-seling 2 · 1 · 1 · 2 menaiki tangga). Itu
yang membuat kesalahan "berlakukan pita ke semua lini" — yang akan membuat PA Rp 5 juta
kehilangan seluruh penyetujunya — sulit dilakukan tanpa sengaja.

## `codebase-design` — letak seam dan batas paket

**Kenapa dipakai.** Ada pertanyaan batas yang nyata: master ambang milik `F-4`, mesin
penjenjangan milik `B-7`. Apakah keduanya dua paket Go?

**Yang dihasilkan.** Satu paket `internal/komite`, dengan alasan yang ditulis di doc
paketnya: paket Go adalah satuan **kohesi**, bukan satuan modul proyek. Memecahnya menjadi
dua akan menghasilkan paket yang hanya memuat satu fungsi murni ditambah pemetaan tipe
bolak-balik. Batas modulnya tetap terlihat — di **rute** (`master/ambang-komite` versus
`komite/penjenjangan`) dan di doc yang menyatakan modul ini **membaca** master, tidak
memilikinya.

**Penerapan prinsip "dua adapter, bukan satu".** Seam `Repo` punya dua pengisi nyata:
`sqlstore` (Oracle) dan `memori` (30 baris master sungguhan). Yang kedua bukan sekadar
untuk pengujian — ia yang membuat kedua layar dapat dicoba lengkap tanpa Oracle, termasuk
ketujuh kasus spec.

**Satu keputusan yang diambil karena prinsip ini.** Seam `Repo` hanya punya **satu metode**,
dan penyaringan dikerjakan di domain. Antarmuka yang dangkal di permukaan tetapi dalam di
isinya; dan yang lebih penting, aturan bisnisnya tidak tersalin ke dalam SQL.

## Skill lain dan alasan tidak dipakai

| Skill | Alasan |
|---|---|
| `tdd` | Uji ditulis berdampingan dengan kode, tetapi **tidak dapat dijalankan** — Go tidak terpasang. Siklus merah-hijau-refaktor mustahil ditempuh, dan mengaku memakainya akan menyesatkan |
| `prototype` | Tidak ada pertanyaan rancangan yang menuntut prototipe. Yang meragukan justru terjawab dengan membaca rule dan menghitung ulang master |
| `diagnosing-bugs` | Tidak ada cacat yang sedang didiagnosis pada kode sendiri. Cacat sistem lama dicatat sebagai temuan, bukan didiagnosis |
| `code-review` | Tidak ada perubahan orang lain untuk ditinjau |
| `research` | Seluruh fakta ada di dalam repositori. Tidak satu pun klaim di sesi ini bersumber dari luar |
| `dataviz` | Tidak ada grafik. Tangga ambang justru paling terbaca sebagai tabel, dan menggambarnya sebagai bagan akan menyembunyikan angka yang justru harus diperiksa |
| `artifact-*` | Keluarannya kode di repositori, bukan halaman yang dibagikan |

## Teknik yang dipakai tanpa memanggil skill

**Menghitung ulang angka dokumen terhadap sumbernya.** Ketujuh kasus spec dihitung ulang
dari CSV; ketiganya cocok, dan proses itu yang memunculkan koreksi atas kalimat tiket.

**Menjadikan aturan sebagai uji, bukan sebagai catatan.** Tiga hal yang paling mudah
dilanggar kemudian — jangan tulis ke `EMAILKOMITE`, jangan pakai `LIMIT_TOP` untuk
menyaring, jangan berlakukan pita ke semua lini — ketiganya dipagari uji yang menyebutkan
akibatnya bila dilanggar. Aturan yang hanya ada di dokumen akan dilanggar oleh kode
berikutnya.

**Melaporkan ketidakpastian alih-alih menyembunyikannya.** Tiga penanda dibawa sampai ke
layar: `UrutanTidakPasti`, `TanpaPenyetuju`, dan `SedangAbsen`. Ketiganya keadaan yang
tidak dapat diputuskan modul ini sendiri, dan menyembunyikannya akan membuat perbedaan
terhadap Pega terbaca sebagai cacat.

## Kesalahan sendiri yang tercatat sesi ini

1. **Pengelompokan integritas mula-mula per `TYPE_KOMITE` untuk semua lini.** Itu membelah
   tangga PA menjadi dua potongan yang tampak berlubang parah. Tertangkap saat menghitung
   temuan yang diharapkan terhadap master nyata — bukan saat menulis kodenya.
2. **Atap pita bawah Non-MBU sempat dilaporkan sebagai peringatan.** Berhenti tepat di batas
   pita justru benar; melaporkannya akan menjadi peringatan palsu yang muncul setiap kali
   layar dibuka, dan peringatan yang selalu muncul akhirnya diabaikan.
3. **`DariRupiah` sempat panik pada data dari basis data.** Panik pantas untuk konstanta di
   dalam kode, tidak untuk data dari luar — data yang aneh tidak boleh menjatuhkan proses
   yang sedang melayani pengguna lain. Dipisah menjadi `dariRupiahAman`.

## Catatan untuk sesi berikutnya

Modul berikutnya yang wajar adalah **`TKT-B07-002`** — layar keputusan komite — dan ia
**belum dapat dikerjakan**: `B-5` dan `B-6` belum ada, dan dua Ticket rule (`KomiteAssign_ticket`,
`komiteAccept_ticket`) beserta When rule `IsKomite` masih hilang dari export (`R-16`).

Yang **dapat** dikerjakan lebih dulu tanpa menunggu siapa pun: `B-5` nilai terkonversi,
karena `D-48` sudah menetapkan aturannya lengkap — kurs pada tanggal kejadian, dan klaim
**ditolak** bila kursnya tidak ada. Tipe `uang` yang dibangun sesi ini sudah menjadi
fondasinya.

---

## Sesi penggantian nama modul Komite (2026-09-19)

Tidak satu pun skill dipanggil pada sesi ini, dan itu keputusan sadar: pekerjaannya adalah
penggantian nama mengikuti aturan yang sudah tertulis (`D-80`, `D-81`), bukan penggalian fakta
maupun perancangan.

| Skill | Dipakai? | Alasan |
|---|---|---|
| `domain-modeling` | **tidak dipanggil, disiplinnya dipakai** | Padanan Inggris tiap istilah diambil dari `CONTEXT.md`, bukan diterjemahkan sendiri — itu sebabnya `Adjustment` dan `Object` tetap tidak dipakai, dan yang dipakai `SettlementLine` serta `InsuredItem`. Untuk modul ini: `Ambang`→`Threshold`, `Jenjang`→`Tier`, `Penyetuju`→`Approver`, `Pita`→`Band` |
| `grilling` | **tidak dipanggil, disiplinnya dipakai pada diri sendiri** | Setiap penggantian otomatis diuji terhadap kasus yang jelas benar sebelum hasilnya dipercaya. Itu yang menangkap tiga kesalahan di §17.3 |
| `codebase-design` | dievaluasi | Batas modul tidak berubah sama sekali pada sesi ini; hanya namanya |
| `tdd`, `prototype`, `diagnosing-bugs`, `code-review`, `research`, `resolving-merge-conflicts` | tidak relevan | Tidak ada perilaku yang berubah, tidak ada fakta baru yang digali, dan tidak ada yang dapat dijalankan — Go maupun Node.js tidak terpasang |

### Teknik yang dipakai tanpa memanggil skill, dan yang membuatnya berguna

**Uji yang dirancang untuk gagal.** Perapi kolom dijalankan atas **seluruh 85 berkas** backend,
termasuk empat paket yang sudah gofmt-bersih dan tidak disentuh sama sekali. Setiap berkas di
luar lingkup yang ikut berubah adalah bukti perapinya salah — dan itulah yang menangkap tiga
cacat berturut-turut pada perapi itu.

**Salinan sebelum menyentuh.** Sebelum penggantian variabel lokal di berkas uji, seluruh paket
disalin lebih dulu. Ketika penggantian itu terbukti memutus konsistensi deklarasi-pemakaian, ia
dapat dibatalkan seluruhnya alih-alih ditambal sepotong-sepotong.

**Pemeriksa silang rujukan versus deklarasi.** Seluruh `komite.X` di berkas uji dibandingkan
terhadap deklarasi paket. Ia menemukan dua rujukan usang yang tidak tertangkap pemindaian nama
lama — dan, pada percobaan pertama, **melaporkan 18 temuan palsu karena alatnya sendiri salah**
(§17.3 butir 2).

### Kesalahan sendiri yang tercatat sesi ini

Tiga, seluruhnya di `catatan-pengembangan.md` §17.3. Yang kedua — `\t` dipakai di dalam ERE,
padahal POSIX tidak mengenalnya — adalah **kali keempat** pola yang sama muncul dalam dua sesi:
alat ukur dipercaya sebelum diuji pada kasus yang jelas benar.

Yang berubah dari sesi sebelumnya bukan kekerapan kesalahannya, melainkan **kapan ia
ketahuan**: ketiganya tertangkap oleh pemeriksaan yang dijalankan sendiri sebelum hasilnya
dilaporkan, bukan oleh Work Owner sesudahnya.
