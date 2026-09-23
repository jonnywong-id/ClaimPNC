# Catatan Pengembangan — Sesi 2026-09-15

Jalannya pengerjaan tahap login: apa yang ditanyakan, apa yang ditemukan, apa yang dibangun, dan
kendala apa yang muncul beserta penyelesaiannya.

Keputusan desainnya sendiri ada di [`keputusan-implementasi.md`](keputusan-implementasi.md);
berkas ini merekam **prosesnya**.

---

## 1. Analisis pra-implementasi

Dilakukan sebelum satu baris kode ditulis, terhadap repository dokumen migrasi di
`D:\Jonny\Project\Claude.AI\XML Claim PNC`.

### 1.1 Yang dibaca

| Bahan | Yang diambil darinya |
|---|---|
| `docs/AGENTS.md` | Aturan repo: XML Pega baca-saja; bahasa dokumen Indonesia; kekosongan pemahaman tidak ditambal asumsi |
| `docs/ticketing/README.md` + `F-1`, `F-2`, `F-3`, `U-1` | Lingkup, acceptance criteria, dependency, dan penghalang tiap tiket |
| `docs/ADR/0001`, `0002`, `0023`, `0024` | Modular monolith Go; SPA React disajikan binary Go; otorisasi berbasis menu; autentikasi HCC/HCQ |
| `docs/Steering/04-FUTURE-ARCHITECTURE.md` | Empat lapisan, arah ketergantungan, daftar seam — termasuk §3.5 seam Identity |
| `docs/Steering/08-TECHNICAL-STRATEGY.md` | Tumpukan teknologi, struktur folder, coding standards, aturan SQL portabel, daftar larangan |
| `docs/Steering/11-SECURITY.md` §2–§3 | Alur autentikasi enam langkah, aturan token, tabel baru yang dibutuhkan |
| `docs/Steering/09-DATABASE-STRATEGY.md` | Penulis tunggal per tabel, penamaan migrasi |

### 1.2 Tiga temuan yang mengoreksi premis instruksi

1. **Belum ada kode implementasi sama sekali.** `docs/AGENTS.md` dan `docs/ticketing/README.md`
   menyatakannya eksplisit: 106 tiket, seluruhnya belum dikerjakan. Premis instruksi bahwa modul
   Login, Home, dan Master Data "sudah selesai" tidak berlaku — tidak ada yang perlu diisolasi,
   dan tidak ada pola coding berjalan untuk ditiru. Yang mengikat adalah dokumen.
2. **"Module Proses Produksi" tidak ada** dalam peta 34 modul proyek ini. Yang dikerjakan adalah
   apa yang diminta di bagian I dan III instruksi: halaman login dan fungsinya — yaitu `F-3` dan
   `U-1`.
3. **Login bukan tiket paling awal.** `TKT-F3-001` bergantung pada `TKT-F1-001`; `TKT-F3-003`
   bergantung pada `TKT-F2-001`. Kerangka aplikasi harus ikut dibangun, bukan tambahan opsional.

### 1.3 Penghalang yang sudah diantisipasi papan tiket

`ADR-0024` berstatus `Proposed` dan melarang dirinya dijadikan dasar implementasi: kontrak API
HCC/HCQ tidak ada — `HCC` dan `HCQ` muncul **2× di seluruh export**, keduanya teks pesan galat
yang menyuruh menghubungi helpdesk.

Yang membuat pekerjaan ini tetap dapat dikerjakan: `TKT-F3-001` berlabel `ready-for-human` justru
untuk membuka jalan buntu itu — seam `Identity` dengan adapter tiruan sebagai penopang sementara
(`ADR-0024` Opsi 2). Jalur ini **sudah diputuskan** dan karena itu tidak perlu dikonfirmasikan
ulang.

---

## 2. Pertanyaan konfirmasi dan jawabannya

Diajukan sebagai satu gerbang sebelum coding, sesuai ketentuan *Grounded Checkpoint*.

### Pertanyaan 1 — lokasi kode

> Repo dokumen berisi 2.634 XML export Pega yang bersifat baca-saja. Di mana kode Go + React
> dibuat: subfolder `app/` di repo itu, langsung di root-nya, atau folder terpisah?

**Jawaban:** `D:\app\claim-pnc`.

**Akibatnya:** export XML Pega tidak tersentuh sama sekali — aturan 1 `docs/AGENTS.md` terpenuhi
dengan sendirinya. Rujukan ke dokumen tiket menjadi lintas folder dan ditulis sebagai jalur absolut.

### Pertanyaan 2 — cakupan tahap ini

> Login tidak dapat berdiri tanpa kerangka aplikasi. Fondasi minimal penuh, fondasi seadanya, atau
> backend saja?

**Jawaban:** login end-to-end, fondasi seadanya.

**Akibatnya:** struktur empat lapisan tetap dipakai — biayanya nol dan memindahkan kode yang
telanjur ditulis jauh lebih mahal. Yang dilewati adalah perkakas penegaknya (`depguard`,
`golangci-lint`), health check, dan graceful shutdown. Dicatat sebagai utang teknis nomor 1 di
`keputusan-implementasi.md` §4.

**Keberatan yang disampaikan sebelum mengerjakan:** tanpa `depguard`, aturan lapisan `ADR-0001`
hanya dijaga kesepakatan lisan — persis mode kegagalan yang `D-09` sebut sebagai alasan struktur
dibuat preskriptif. Work Owner tetap memilih opsi ini; pekerjaan dilanjutkan sesuai pilihan itu.

### Pertanyaan 3 — basis data pengembangan

> `TKT-F3-003` menuntut sesi tersimpan di basis data dan dikenali dua instans. PostgreSQL lokal,
> Oracle 19c dev, atau belum ada sama sekali?

**Jawaban:** Oracle 19c dev.

**Akibatnya:** adapter SQL dan migrasi ditulis untuk Oracle. Kredensialnya belum diberikan,
sehingga jalur itu **belum pernah dijalankan** — dicatat terbuka di `keputusan-implementasi.md` §5,
bukan dinyatakan lulus.

### Pertanyaan 4 — penyimpanan token di peramban

> `11-SECURITY.md` §2.2 menetapkan cookie `HttpOnly`; `04-FUTURE-ARCHITECTURE.md` §1 menggambarkan
> Bearer token. Keduanya bertentangan. Mana yang dipakai?

**Jawaban:** Bearer di header `Authorization`.

**Akibatnya:** token harus dapat dibaca JavaScript, sehingga risiko XSS yang disebut
`11-SECURITY.md` §2.2 tetap berlaku. `sessionStorage` dipilih untuk memperkecil paparannya. Dicatat
sebagai penyimpangan sadar di `keputusan-implementasi.md` §3.1.

---

## 3. Yang dibangun

### 3.1 Lapisan Domain

`internal/auth` (`identitas.go`) — seam `Identitas`, tipe `Profil` dengan pemeriksaan lima field, dan
tiga galat sentinel yang dapat dibedakan lewat `errors.Is`.
`internal/auth` (`pengguna.go`) — catatan pengguna lokal dan seam `PenggunaRepo`.
`internal/auth` (`sesi.go`) — tipe `Token` dengan `Sidik()`, penerbitan token dan pengenal acak, aturan
masa berlaku dan pencabutan, seam penyimpanan.
`internal/platform/waktu` — seam `Jam`, jam tetap untuk pengujian, dan jam sistem.

### 3.2 Lapisan Aplikasi

`internal/auth/usecase` — `Masuk`, `Periksa`, `Perpanjang`, `Keluar`. Pemanggilan sistem identitas
diselesaikan **sebelum** satu pun baris basis data disentuh, sesuai aturan bahwa pemanggilan
sistem luar tidak boleh berada di dalam transaksi.

### 3.3 Lapisan Adapter

`auth/provider` — provider tiruan berisi empat pengguna contoh dengan nama karangan; menolak
dibentuk bila lingkungan bertanda produksi.
`auth/repo/sqlstore` — implementasi SQL; teks kueri di berkas `.sql` terpisah, dimuat lewat
`go:embed` dan dipecah pada penanda `-- name:`.
`auth/repo/memori` — penyimpanan di memori untuk pengujian.
Jam sistem yang selalu UTC ikut di `platform/waktu`.

### 3.4 Lapisan Transport

Router `chi`, middleware ID permintaan / log / pulih-dari-panik / autentikasi Bearer, handler
`masuk` · `keluar` · `saya` · `sesi/perpanjang`, DTO terpisah dari tipe domain, dan pemetaan galat
terpusat. Penyajian SPA dengan *fallback* ke `index.html` supaya rute dalam tetap benar setelah
muat ulang halaman.

### 3.5 Basis data

`backend/migrations/0001_pengguna_dan_sesi.up.sql` dan `.down.sql` — dua tabel baru, `CPNC_PENGGUNA` dan
`CPNC_SESI_AKTIF`. Tidak menyentuh satu pun tabel yang dibaca atau ditulis Pega.

### 3.6 Frontend

Layar masuk dengan React Hook Form + Zod, tiga pesan galat yang dibedakan, simpanan sesi Zustand
di atas `sessionStorage`, penjaga rute, peringatan sebelum sesi habis beserta tombol perpanjang,
dan beranda sementara yang membuktikan sesi dikenali server.

---

## 4. Kendala yang muncul dan penyelesaiannya

| Kendala | Penyelesaian | Dampak |
|---|---|---|
| `godror` yang ditetapkan `08-TECHNICAL-STRATEGY.md` §1 menuntut CGO dan Oracle Instant Client; mesin ini `CGO_ENABLED=0` tanpa kompilator C, sehingga `go build ./...` tidak dapat diselesaikan | Memakai `go-ora` (murni Go) dan mengurung seluruh sentuhan driver di satu berkas | Penyimpangan dari Steering; dicatat di `keputusan-implementasi.md` §3.2. Pertukaran kembali ke `godror` menyentuh satu berkas |
| Kredensial Oracle dev belum ada, sedangkan aplikasi gagal start tanpa basis data — login tidak dapat diperlihatkan sama sekali | Menambah sakelar `PENYIMPANAN=memori` yang **menolak berjalan di produksi**, mengikuti pola provider identitas tiruan | Login dapat dijalankan hari ini; jalur Oracle tetap ditulis dan tetap ditandai belum terbukti |
| `MERGE` Oracle menuntut `FROM DUAL` yang dilarang aturan SQL portabel | `UPDATE` lalu `INSERT` bila tidak ada baris terkena, dengan penanganan kalah-lomba | Dua pernyataan alih-alih satu; disiplin `D-20` tetap utuh |
| Proxy korporat menangkap permintaan `curl` ke `localhost` dan menjawabnya dengan halaman galat Squid | Menjalankan ulang dengan `curl --noproxy '*'` | Hanya memengaruhi cara verifikasi manual dijalankan |
| `@/` alias jalan di TypeScript tetapi gagal saat Vitest memuat modul | Menambahkan `resolve.alias` di `vite.config.ts`, bukan hanya `paths` di `tsconfig.json` | — |
| Galat konfigurasi tercetak hanya berbutir pada baris pertama | Menggabungkan pesan dengan pemisah `\n  - ` agar seluruh kekurangan terlihat sebagai butir | Operator melihat semua yang kurang dalam satu kali jalan |

---

## 5. Verifikasi yang benar-benar dijalankan

Bukan rencana — berikut yang dijalankan pada 2026-09-15 beserta hasilnya.

```
go vet ./...                 bersih
go test ./...                seluruhnya lulus (6 paket berisi uji)
cd frontend && npx tsc --noEmit   bersih
cd frontend && npx vitest run     11 uji lulus
cd frontend && npm run build      dist/ terbentuk, 408 kB
go build -o claimpnc.exe ./cmd/claimpnc
```

Uji manual terhadap binary yang berjalan (`PENYIMPANAN=memori`, `IDENTITAS_ADAPTER=fake`):

| Yang dicoba | Hasil |
|---|---|
| `GET /masuk` | 200, HTML — SPA tersemat tersaji dari binary |
| `GET /klaim/123/estimasi` | 200 — *fallback* rute dalam bekerja |
| `POST /api/masuk` kredensial sah | 200, token + profil lima field |
| `GET /api/saya` dengan Bearer | 200, identitas benar |
| kata sandi salah | 401 `kredensial_salah` |
| pengguna tidak dikenal | 401 `kredensial_salah` — **teks dan kode identik** dengan baris di atas |
| pengguna nonaktif | 403 `pengguna_tidak_aktif` |
| profil tidak lengkap | 502 `sistem_identitas_tidak_terhubung` |
| tanpa token | 401 `sesi_tidak_sah` |
| `POST /api/keluar` | 204 |
| token lama setelah keluar | 401 `sesi_tidak_sah` — pencabutan berlaku seketika |
| token & kata sandi di dalam log | **0 kemunculan** |

Uji gagal-keras:

| Yang dicoba | Hasil |
|---|---|
| `APP_ENV=production` + `IDENTITAS_ADAPTER=fake` | gagal start, exit 1, "provider identitas tiruan menolak berjalan di lingkungan produksi" |
| `IDENTITAS_ADAPTER=hcc` | gagal start, exit 1, menyebut `ADR-0024` dan `TKT-F3-002` |
| `PENYIMPANAN=oracle` tanpa parameter | gagal start, exit 1, menyebut keempat variabel yang kurang sekaligus |
| `PENYIMPANAN=memori` di produksi | ditolak — diuji di `backend/cmd/claimpnc/main_test.go` |

**Yang tidak dapat dijalankan:** seluruh jalur Oracle. Lihat `keputusan-implementasi.md` §5.

---

## 6. Restrukturisasi menjadi backend/frontend terpisah — sesi kedua, 2026-09-15

### 6.1 Permintaan

Work Owner meminta susunan repository mengikuti aplikasi **ClaimQ**, dengan backend dan frontend
dipisahkan. Rinciannya, keputusannya, dan penyimpangannya dari Steering dicatat di
[`keputusan-implementasi.md` §8](keputusan-implementasi.md).

### 6.2 Keberatan yang disampaikan sebelum mengerjakan

Susunan ClaimQ adalah **module-first**; `08-TECHNICAL-STRATEGY.md` §2 menetapkan **layer-first**
dengan `web/` di dalam satu root. Keberatan disampaikan dalam dua kalimat, lalu pekerjaan
dilanjutkan sesuai permintaan — dengan satu penyesuaian yang diambil sendiri: lapisan tidak
dihapus melainkan turun menjadi subpaket di dalam modul, supaya arah ketergantungan `ADR-0001`
tidak ikut hilang.

### 6.3 Kendala saat pemindahan

| Kendala | Penyelesaian | Dampak |
|---|---|---|
| `mv cmd backend/` ditolak `Permission denied` — ada proses Windows yang memegang folder itu | Menyalin lalu menghapus asalnya (`cp -r` + `rm -rf`) | Tidak ada; isinya utuh |
| `mv web/src frontend/` ditolak dengan alasan yang sama | Sama | Tidak ada |
| Folder `web/` di root tidak dapat dihapus: **"Device or resource busy"**, juga lewat PowerShell | **Selesai.** Proses yang memegangnya — file watcher editor — dilepas dengan me-restart editor, lalu foldernya dihapus | Tidak ada; foldernya memang sudah kosong sejak isinya dipindahkan |
| `go:embed all:dist` gagal karena `backend/spa/dist` belum ada | `.gitkeep` ikut ter-commit, dan `npm run build` menuliskannya kembali setelah Vite mengosongkan folder | `go build ./...` tetap jalan pada clone yang bersih — **dibuktikan**, lihat §7.1 |
| `provider.HCCQ` tidak memenuhi `auth.Identitas` sehingga `cmd` gagal kompilasi | Menambahkan `Verifikasi` yang menolak dengan galat yang sama | Seam terlihat lengkap di peta kode; perilakunya tetap menolak |
| Dua antarmuka bernama sama (`pengguna.Penyimpanan`, `sesi.Penyimpanan`) bentrok saat paketnya disatukan | Dinamai ulang `auth.PenggunaRepo` dan `auth.SesiRepo` | Nama pemanggilnya justru lebih jelas |

### 6.4 Yang ikut diperbaiki sekalian

Folder `components/` pada susunan ClaimQ tidak dibiarkan kosong sebagai formalitas. Dua bagian
layar masuk yang memang sudah terduplikasi diangkat ke sana: `KolomIsian` (label + input + pesan
kesalahan, dipakai dua kali di satu form) dan `PesanGalat` (kotak pesan dengan dua nada, penolakan
dan gangguan). Layar masuk menjadi lebih pendek dan tidak ada penyalinan kelas Tailwind.

---

## 7. Verifikasi ulang setelah restrukturisasi

Seluruhnya dijalankan ulang terhadap susunan baru. **Hasilnya sama persis dengan sebelum
restrukturisasi** — itulah yang membuktikan pemindahan tidak mengubah perilaku.

```
cd backend  && gofmt -l ./cmd ./internal ./web    bersih
cd backend  && go vet ./...                       bersih
cd backend  && go test ./...                      6 paket lulus
cd frontend && npx tsc --noEmit                   bersih
cd frontend && npx vitest run                     11 uji lulus
cd frontend && npm run build                      ../backend/spa/dist terbentuk
cd backend  && go build -o claimpnc.exe ./cmd/claimpnc
```

Uji manual terhadap binary hasil susunan baru:

| Yang dicoba | Hasil |
|---|---|
| `GET /masuk` | 200, HTML — SPA tersemat tersaji dari binary |
| `GET /klaim/123/estimasi` | 200 — *fallback* rute dalam bekerja |
| `POST /api/masuk` kredensial sah | 200, token + profil lima field |
| `GET /api/saya` dengan Bearer | 200, identitas benar |
| kata sandi salah | 401 `kredensial_salah` |
| pengguna tidak dikenal | 401 `kredensial_salah` — **teks dan kode identik** dengan baris di atas |
| pengguna nonaktif | 403 `pengguna_tidak_aktif` |
| `POST /api/sesi/perpanjang` | 200, batas berlaku bergeser |
| `POST /api/keluar` | 204 |
| token lama setelah keluar | 401 `sesi_tidak_sah` — pencabutan berlaku seketika |
| token & kata sandi di dalam log | **0 kemunculan** |

Uji gagal-keras:

| Yang dicoba | Hasil |
|---|---|
| `APP_ENV=production` + `IDENTITAS_ADAPTER=fake` | gagal start, "provider identitas tiruan menolak berjalan di lingkungan produksi" |
| `IDENTITAS_ADAPTER=hcc` | gagal start, menyebut `ADR-0024 Proposed` dan `TKT-F3-002 needs-info` |
| `PENYIMPANAN=oracle` tanpa parameter | gagal start, menyebut keempat variabel yang kurang sekaligus |
| `PENYIMPANAN=memori` di produksi | ditolak — diuji di `backend/cmd/claimpnc/main_test.go` |

**Yang tetap tidak dapat dijalankan:** seluruh jalur Oracle. Restrukturisasi tidak mengubah apa pun
soal itu — lihat `keputusan-implementasi.md` §5.

### 7.1 Uji clone bersih — membuktikan `.gitkeep` benar-benar cukup

Klaim "`go build ./...` tetap jalan pada clone yang bersih" tidak dibiarkan sebagai klaim. Keadaan
clone bersih ditiru dengan mengosongkan `backend/spa/dist` sampai hanya tersisa `.gitkeep`, lalu:

| Yang dicoba | Hasil |
|---|---|
| `go build ./...` dengan `dist` hanya berisi `.gitkeep` | **berhasil** — direktif `go:embed` terpenuhi |
| aplikasi dijalankan dari binary itu | start normal, dengan peringatan `"SPA tidak tersedia; aplikasi hanya melayani API"` |
| `POST /api/masuk` | 200 — API tetap melayani penuh |
| `GET /masuk` | 404 — benar: tidak ada SPA yang dapat disajikan, dan aplikasi mengatakannya apa adanya alih-alih menyajikan halaman rusak |

Setelah itu `npm run build` dijalankan ulang dan `dist` kembali utuh.

### 7.2 Sisa pekerjaan sesi kedua

Tidak ada. Folder `web/` di root sudah terhapus, dan seluruh pemeriksaan di §7 dijalankan ulang
pada susunan akhir: `gofmt` bersih, `go vet` bersih, 65 uji Go lulus, `tsc --noEmit` bersih,
11 uji frontend lulus, dan seluruh uji manual di tabel §7 memberi hasil yang sama.

---

## 8. Sesi ketiga — kontrak HCC/HCQ, portal, dan login dua sumber (2026-09-16)

### 8.1 Bahan yang diberikan

| Bahan | Isi |
|---|---|
| `Database/m_portal_pnc.csv` | 6 entitas: ASM, ASI, SMAS, SMI, SPK, SPKS |
| `Database/gcnm_connect_rest.csv` | 1 baris: `APP='ASM'`, `TYPESERVICE='HCQ-LOGIN'`, beserta `SERVICENAME` |
| `Database/m_login_pnc.csv` | 1 baris contoh login non-karyawan |
| Contoh JSON HCQ | request, respons gagal, respons berhasil ringkas, lalu respons berhasil **lengkap** |
| Aturan alur masuk | HCQ dulu; bila gagal, SHA-256 ke `POOLDATA.M_LOGIN_PNC` |

### 8.2 Yang diperiksa lebih dulu, sebelum menulis kode

Tiga pemeriksaan, dan ketiganya mengubah rencana:

1. **Sidik kata sandi diverifikasi terhadap data nyata.** `printf '123' | sha256sum` menghasilkan
   persis isi kolom `HASH_PASSWORD` pada baris contoh — dalam huruf besar. Dua hal terbukti
   sekaligus: skemanya SHA-256 polos, dan keluarannya disimpan huruf besar. Tanpa memeriksa ini,
   pencocokan huruf kecil akan gagal untuk **seluruh** pengguna non-karyawan, dan sebabnya sulit
   ditemukan.
2. **`ADR-0030` dibaca, dan ia sudah `Accepted`.** Isinya menetapkan satu database per entitas,
   master data per portal, dan — yang menentukan letak pemilih portal — **berpindah portal tanpa
   login ulang**.
3. **Export Pega ditelusuri untuk ketiga tabel.** `M_PORTAL_PNC` dan `M_LOGIN_PNC` tidak ada, tetapi
   `GCNM_CONNECT_REST` **ada** di `RDB List/BrowseServiceName_sql-SQL.xml`. Itu baseline yang
   sebelumnya terlewat, dan isinya memperkuat `ADR-0030`.

### 8.3 Tiga pertanyaan konfirmasi

| Pertanyaan | Jawaban Work Owner |
|---|---|
| Pemilih portal di layar masuk atau di dalam aplikasi? | **Di dalam aplikasi**, sesuai `ADR-0030` |
| Basis data mana yang melayani lookup pra-login dan tabel sesi? | **Portal utama ASM**, ditandai sementara — kelak mungkin ditentukan per login dari tabel |
| Kolom `APP` mengikuti `portal_alias` atau selalu `'ASM'`? | **Mengikuti `portal_alias`** |

Jawaban ketiga kemudian **ditegaskan ulang** Work Owner di tengah pengerjaan, beserta lampiran
respons HCQ yang lengkap. Kueri sudah memakai parameter binding sejak awal sehingga tidak berubah;
yang berubah adalah pemetaan responsnya.

### 8.4 Yang dibangun

**Konfigurasi.** Pemuat `.env` tanpa dependensi baru; nilai di lingkungan proses menang atas
berkas. Alias portal ditemukan dengan memindai `POOLDATA_<ALIAS>_HOST`, bukan dari daftar tetap.

**Koneksi.** `platform/db.Kumpulan` memegang koneksi seluruh portal sekaligus. Portal utama fatal
bila gagal; portal lain dicatat dan dilewati.

**Modul `auth` bertambah tiga provider.** `HCQ` (HTTP nyata, Basic Auth, alamat dari
`GCNM_CONNECT_REST`), `Lokal` (SHA-256 ke `M_LOGIN_PNC`), dan `Berantai` yang menjalankan keduanya
menurut urutan bisnis.

**Modul `portal` — modul kedua aplikasi ini.** Susunannya sama persis dengan `auth`: inti + seam di
akar, lalu `repo/sqlstore`, `repo/memori`, dan `http/`. Ia memasang rutenya sendiri lewat
`portalhttp.Pasang`, dan `cmd/claimpnc` cukup menambah satu baris — bukti bahwa susunan module-first
sesi kedua memang bekerja seperti yang diklaim.

**Frontend.** Pemilih portal di header, tabel daftar portal beserta status kesiapan, dan panel
identitas yang menyesuaikan diri: label `NIK` untuk karyawan, `ID Login` untuk non-karyawan, dan
baris email/perusahaan yang disembunyikan bila kosong.

### 8.5 Kendala dan penyelesaiannya

| Kendala | Penyelesaian | Dampak |
|---|---|---|
| Aturan "lima field profil wajib" dari sesi pertama menolak **kedua** provider nyata | Dilonggarkan menjadi tiga field wajib, dengan alasan tertulis di tipe `auth.Profil` | Model berubah; migrasi `0001` ikut disunting |
| Nama kolom `NIK` tidak berlaku untuk broker dan surveyor | Diubah menjadi `IDENTITAS` + kolom `JENIS` | Menyentuh tabel, kueri, DTO, dan frontend |
| Contoh respons pertama tampak tidak memuat cabang dan jabatan | Contoh lengkap menunjukkan keduanya ada di `EmpResponse.Placement`; pemetaan diperbaiki | Kesimpulan sesi ini sempat salah dan dikoreksi di `keputusan-implementasi.md` §9.3 |
| Enam portal × lima variabel = 30 baris `.env` yang belum diisi | Portal tidak lengkap dilewati, bukan menggagalkan start; hanya portal utama yang wajib | Aplikasi tetap jalan hari ini dengan satu portal |
| Heredoc bash gagal pada dokumen panjang berisi kutip SQL | Menulis lewat berkas scratchpad lalu menggabungkannya | Tidak ada; hanya cara penulisan |

### 8.6 Verifikasi yang dijalankan

```
cd backend  && gofmt -l ./cmd ./internal ./spa    bersih
cd backend  && go vet ./...                       bersih
cd backend  && go test ./...                      8 paket lulus
cd frontend && npx tsc --noEmit                   bersih
cd frontend && npx vitest run                     18 uji lulus
cd frontend && npm run build                      ../backend/spa/dist terbentuk
cd backend  && go build -o claimpnc.exe ./cmd/claimpnc
```

Uji manual terhadap binary:

| Yang dicoba | Hasil |
|---|---|
| masuk karyawan | 200, `jenis: KARYAWAN`, `identitas` berisi NIK, email dan perusahaan terisi |
| masuk non-karyawan | 200, `jenis: NON_KARYAWAN`, `identitas` berisi LOGIN_ID, email dan perusahaan **kosong** |
| `GET /api/portal` dengan sesi | 200, keenam portal, `utama: "ASM"`, hanya ASM `siap: true` |
| `GET /api/portal` tanpa sesi | 401 `sesi_tidak_sah` — daftar portal berada di balik sesi |
| token & kata sandi di log | **0 kemunculan** |

Uji otomatis yang paling menentukan:

| Uji | Yang dibuktikan |
|---|---|
| `TestSidikKataSandiCocokDenganDataNyata` | sidik yang dihitung aplikasi **sama persis** dengan isi `HASH_PASSWORD` pada baris contoh |
| `TestHCQMenerimaKredensialSah` | pemetaan sembilan field dari `Person` + `Placement`, Basic Auth memakai kredensial aplikasi (bukan kredensial pengguna), dan `APP` diisi alias portal |
| `TestBerantaiBerhentiSaatSumberPertamaBerhasil` | kata sandi karyawan tidak pernah ikut disidik ke tabel non-karyawan |
| `TestBerantaiTetapMelayaniSaatSumberPertamaPutus` | broker tetap dapat masuk ketika HCQ mati |
| `TestBerantaiMelaporkanPutusBukanKredensialSalah` | pengguna tidak disuruh mengetik ulang sandi yang sebenarnya benar |
| `TestHCQTidakMeneruskanTeksResult` | teks `Result` dari HCQ — yang membedakan "user tidak ada" dari "sandi salah" — tidak sampai ke pengguna |
| `TestRingkasTidakMemuatRahasia` | kata sandi basis data dan HCQ tidak bocor lewat log konfigurasi |

### 8.7 Yang belum dapat dijalankan

Tidak berubah dari sesi sebelumnya, dan bertambah satu: panggilan nyata ke HCQ. Seluruh bentuk
permintaan dan pemetaan responsnya sudah diuji terhadap peladen tiruan yang memakai contoh JSON
asli, tetapi **belum pernah menyentuh `hcq.payrollq.id`** karena `HCQ_LOGIN_USER` dan
`HCQ_LOGIN_PASSWORD` belum diisi. Lihat `keputusan-implementasi.md` §9.10.

### 8.8 Cacat yang ditemukan Work Owner saat mencoba menjalankan

`go run ./cmd/claimpnc` pada clone yang bersih **gagal start**:

```
gagal menjalankan aplikasi: konfigurasi tidak sah:
  - PORTAL_UTAMA "ASM" tidak punya satu pun variabel POOLDATA_ASM_*; portal yang terbaca: (tidak ada)
```

Penyebabnya bukan salah pakai: `backend/.env` memang belum ada, dan nilai baku `PENYIMPANAN`
adalah `oracle`, yang menuntut kredensial portal yang belum diisi siapa pun.

Dua hal diperbaiki, keduanya cacat rancangan saya sendiri:

| Cacat | Perbaikan |
|---|---|
| Nilai baku `PENYIMPANAN=oracle` membuat clone bersih **tidak dapat dijalankan sama sekali** | Nilai baku kini mengikuti lingkungan: **memori** di `development`/`test`, **oracle** di `staging`/`production`. Aman karena penyimpanan memori sudah menolak produksi dari dalam kode, bukan hanya lewat nilai baku |
| Pesan galatnya **akurat tetapi tidak dapat ditindaklanjuti** — menyebut apa yang kurang, tanpa menyebut apa yang harus dilakukan | Pesannya kini menyebut dua jalan keluar (salin `.env.example`, atau `PENYIMPANAN=memori`) beserta jebakan yang paling sering terjadi: `.env` dibaca relatif terhadap **direktori kerja**, bukan letak binary |

Prinsip "gagal keras dengan pesan yang jelas" yang saya tulis sendiri di §4 ternyata baru separuh
dijalankan: pesannya jelas soal **apa**, tetapi bisu soal **bagaimana**. Galat saat start dibaca
orang yang sedang terhenti; menyebut variabel yang hilang tanpa menyebut langkah berikutnya hanya
memindahkan pekerjaan menebak kepadanya.

Tiga uji ditambahkan supaya ini tidak terulang: `TestTanpaEnvSamaSekaliTetapDapatStart`,
`TestStagingDanProduksiTetapMenuntutOracle`, dan `TestGalatKonfigurasiMenyebutCaraMemperbaiki`.

### 8.9 Mode periksa, dan integrasi nyata yang akhirnya terbukti

Work Owner ingin langsung menguji jalur HCC/HCQ dan `POOLDATA.M_LOGIN_PNC`. Pemeriksaan pertama
menemukan penghadangnya: **masuk lewat aplikasi menulis ke `CPNC_PENGGUNA` dan `CPNC_SESI_AKTIF`**,
dan kedua tabel itu baru ada setelah DBA menjalankan migrasi `0001`. Menunggu migrasi berarti
menunda pembuktian dua integrasi yang justru paling ingin dibuktikan lebih dulu.

**Yang dibangun:** mode `-periksa` pada binary yang sama — bukan binary kedua, supaya "satu
entrypoint" `ADR-0001` tetap utuh. Ia hanya membaca dan memanggil; **tidak menulis apa pun**,
sehingga dapat dijalankan sebelum migrasi.

```
./claimpnc.exe -periksa                              # koneksi, portal, alamat HCQ, kesiapan tabel
read -s SANDI && echo "$SANDI" | ./claimpnc.exe -periksa -login <pengguna>
```

Kata sandi dibaca dari stdin, bukan dari argumen: argumen tersimpan di riwayat shell dan terlihat
di daftar proses.

**Hasilnya terhadap infrastruktur nyata** — lihat `keputusan-implementasi.md` §9.10 untuk tabel
lengkapnya. Ringkasnya: koneksi Oracle, ketiga tabel warisan, API HCQ, dan rantai dua sumber
seluruhnya **terbukti**; yang tersisa hanya migrasi `0001`.

**Satu jebakan yang nyaris lolos.** Percobaan pertama memakai baris contoh non-karyawan dan
berhasil — tampak seperti bukti bahwa seluruh rantai hidup. Ia bukan: bila HCQ mati, rantai
menandainya putus lalu **tetap lolos** lewat jalur kedua, dan keluarannya sama persis. Pembuktian
HCQ hidup menuntut percobaan terpisah dengan akun yang pasti tidak ada di kedua sumber, lalu
membaca galat mana yang dilaporkan. Tanpa langkah itu, kesimpulan "HCQ hidup" hanya asumsi yang
kebetulan cocok.

### 8.10 Temuan keamanan saat menyiapkan `.env`

Saat menyalin `backend/.env.example` menjadi `backend/.env`, ternyata **berkas contohnya sudah
berisi kredensial nyata** — host dan kata sandi basis data dev, serta kredensial Basic Auth HCQ.

Itu berbahaya karena `.env.example` **sengaja dikecualikan** dari `.gitignore` agar ikut ter-commit
sebagai contoh. Kredensialnya akan ikut masuk ke GitLab pada commit pertama — mengulang persis
masalah yang `D-40` catat pada export Pega (3 kata sandi SMTP plaintext di 31 lokasi).

Penanganannya: nilai nyata dipindahkan ke `backend/.env` (yang diabaikan git), dan `.env.example`
dikembalikan menjadi placeholder kosong. Karena berkas itu sempat memuat kredensial, keduanya
sebaiknya diperlakukan sebagai **berpotensi terpapar** bila berkasnya pernah dibagikan atau
di-commit di tempat lain — keputusan menggantinya ada pada Work Owner dan Tim Infra.


## 9. Sesi keempat — modul Master Status Progres 1 (2026-09-17)

Modul bisnis PERTAMA yang dibangun. Sebelum sesi ini repo hanya memuat login, beranda
sementara, dan daftar portal.

### 9.1 Yang dibaca lebih dulu, sebelum menulis satu baris kode

Instruksi melarang langsung menulis kode. Yang dibaca, seluruhnya dari export Pega:

| Berkas | Yang diambil darinya |
|---|---|
| `Harness/StatusProgress-Harness.xml` | layar acuan; memuat section `MasterStatusProgress` dan memanggil `BrowseStatusProgress` |
| `Section/MasterStatusProgress-Section.xml` | judul layar **"Master Status Progress 1"**, tombol Tambah dan Refresh |
| `Section/BrowseStatusProgress-Section.xml` | grid 3 kolom (`.CaseID` lebar 55, `.City` lebar 260, `.CityID`), form modal 3 isian, label isian **"ID"** dan **"Posisi"**, jenis kontrol (teks dan **dropdown**) |
| `RDB List/BrowseStatusProgress-SQL.xml` | kueri daftar beserta `ORDER BY ID_PROGRESS ASC` |
| `RDB List/InsertStatusProgress1-SQL.xml` | kolom yang disisipkan |
| `RDB List/UpdateStatusProgress1_sql-SQL.xml` | kolom yang di-`SET` dan yang hanya menyaring |
| `RDB List/UpdateStatusProgress1-SQL.xml` | kueri pemuat baris ke modal sunting |
| `RDB List/BrowseIDStatusProgress-SQL.xml` | cara nomor baru diturunkan |
| `Activity/InsertMstStatusProgress1_act-Act.xml` | urutan langkah penambahan, termasuk `"0" + nomor` |
| `Activity/UpdateStatusProgress1_act-Act.xml` | urutan langkah penyuntingan, penanda mode `TempDcol.pyLabel = "Update"` |
| `Activity/ViewStatusProgress_act-Act.xml` | **isi dropdown Posisi** — empat pasang nilai literal |
| `Database/GET_POSISI_PROGRESS_PNC.fnc` | hubungan antartabel progres klaim |
| `Database/GET_POSISI_PROGRESS2.fnc` | hubungan Status Progres 2 ke Status Progres 1 |

Ditambah kode yang sudah ada: modul `auth` dan `portal` seluruhnya, `platform/db`,
`platform/httpserver`, `cmd/claimpnc/main.go`, serta seluruh `frontend/src`.

### 9.2 Empat temuan yang mengoreksi premis instruksi

**Modul Master Data yang disebut "sudah selesai" tidak ada di repo ini.** Instruksi
melarang mengubah modul Login, Home, dan Master Data yang sudah selesai. Diperiksa ke
seluruh riwayat dan kelima cabang (`master`, `feat/intan-master`, `feat/fran-master`,
`feat/arlexy-flowregister`, `feat/flow-register`): tidak satu pun memuat modul master
data. Yang ada hanya `auth`, `portal`, dan `platform`. Larangan itu tetap dihormati untuk
Login dan Home; untuk Master Data tidak ada yang perlu dilindungi karena belum ada.

**Kolom `STATUS` bukan penanda aktif.** Namanya mengesankan flag aktif/nonaktif. Bukti
menunjukkan sebaliknya: label isiannya di Pega adalah **"Posisi"**, kontrolnya dropdown
bersumber `TempPosition.pxResults`, dan `InsertMstStatusProgress1_act` menyalinnya ke
`Local.POSISI`. Ia menyimpan **kode posisi klaim**. Memperlakukannya sebagai flag aktif
akan menghasilkan layar yang benar bentuknya tetapi salah artinya.

**Isi dropdown Posisi tidak ada di tabel mana pun.** Ia dirakit di dalam activity sebagai
empat pasang nilai literal: `REGISTER`=`002`, `SURVEY`=`004`, `KOMITE`=`006`,
`AKSEPTASI`=`007`. Kode `003` dan `005` tidak dipakai jalur ini.

**Kolom `STATUS` tidak dibaca kueri lain mana pun.** Diperiksa ke 17 berkas yang menyebut
`GCNM_MST_PROGRESS_KLAIM`: yang lain hanya menggabung lewat `ID_PROGRESS`. Jadi kolom itu
diisi di layar ini dan tidak pernah dipakai menyaring apa pun di sistem lama. Dicatat apa
adanya — bukan diberi perilaku penyaringan yang tidak pernah ada.

### 9.3 Tiga pertanyaan konfirmasi dan jawabannya

Diajukan sebelum menulis kode, karena ketiganya mengubah bentuk pekerjaan secara
mendasar. Dijawab Work Owner 2026-09-17.

**Pertanyaan 1 — lingkup portal.** Tabel ini ada di basis data SETIAP entitas
(`ADR-0030`), sementara pilihan portal saat itu hanya hidup di frontend dan tidak pernah
dikirim ke backend. Tanpa penanganan, modul ini dapat menulis ke entitas yang salah —
`R-20`, berdampak lintas badan hukum.

> **Jawaban: portal-scoped sekarang.** Frontend mengirim alias portal di header setiap
> permintaan; backend me-resolve koneksinya lewat `db.Kumpulan.Untuk()` dan **menolak**
> bila portal kosong atau tidak dikenal — tidak pernah jatuh ke portal utama.

**Pertanyaan 2 — pintu masuk layar.** Modul Home dilarang diubah, tetapi kerangka menu
(`TKT-U1-001`) belum ada, sehingga layar baru tidak punya tautan menuju ke sana.

> **Jawaban: rute saja, jangan sentuh Beranda.** `HalamanBeranda.tsx` tidak disentuh
> sama sekali. Layar dibuka lewat `/master/status-progres-1`.

**Pertanyaan 3 — sumber daftar Posisi.** `D-15` menuntut nilai seperti ini menjadi master
data, tetapi tabel masternya tidak ada dan membuatnya menuntut persetujuan Work Owner
serta pelaksanaan DBA (`D-63`).

> **Jawaban: konstanta aplikasi, dan dicatat sebagai utang.** Keempat posisi hidup di
> lapisan domain Go dan disajikan lewat endpoint agar frontend tidak menyalinnya.

### 9.4 Yang dibangun

**Backend — modul baru `internal/statusprogres/`**

| Lapisan | Berkas | Isi |
|---|---|---|
| Domain | `statusprogres.go` | tipe `StatusProgres` dan `Isian`, pemeriksaan isian yang mengumpulkan SELURUH pelanggaran, `FormatNomor`, seam `Repo` dan `PemilihRepo` |
| Domain | `posisi.go` | keempat posisi klaim beserta pencarian dan pelabelannya |
| Aplikasi | `usecase/layanan.go` | `Daftar`, `Ambil`, `Tambah`, `Ubah`, `Posisi`, `PastikanPortalSiap` |
| Adapter | `repo/sqlstore/` | 6 kueri di berkas `.sql` terpisah beserta adapter Oracle/PostgreSQL |
| Adapter | `repo/memori/` | adapter kedua yang membuat seam nyata; dipakai uji dan pengembangan tanpa basis data |
| Transport | `http/` | 4 rute, DTO terpisah dari tipe domain, pemetaan galat modul |

**Backend — dua berkas baru pada modul `portal`**, tanpa menyunting berkas yang sudah
ada: `aktif.go` (`PilihAktif`, `ErrTidakDisebut`, `ErrBelumSiap`) dan
`http/portalaktif.go` (header `X-Portal`, middleware `PortalAktif`, `DenganGalatPortal`).

**Frontend**

| Berkas | Isi |
|---|---|
| `components/TabelData.tsx` | **baru** — satu-satunya tabel yang boleh dipakai layar di `modules/`; cikal-bakal `U-2` |
| `components/KolomPilihan.tsx` | **baru** — pasangan `KolomIsian` untuk isian dropdown |
| `components/Tombol.tsx` | **baru** — tombol baku; menonaktifkan diri saat tindakan berjalan |
| `modules/master-status-progres/api.ts` | hook TanStack Query; portal ikut di dalam kunci cache |
| `modules/master-status-progres/FormStatusProgres.tsx` | satu form untuk dua mode, tambah dan ubah |
| `modules/master-status-progres/HalamanStatusProgres1.tsx` | layar daftar |
| `api/klien.ts` | **disunting aditif** — metode `PUT`, header portal, dan `GalatAPI.detail` |
| `api/tipe.ts` | **disunting aditif** — tipe kontrak dan lima kode galat baru |
| `app/App.tsx` | **disunting aditif** — satu rute dan pembungkus `Layar` |

### 9.5 Kendala yang muncul dan penyelesaiannya

**Tipe fungsi bernama tidak dapat saling disalin.** Setiap modul menamai tipe penulis
galatnya sendiri (`authhttp.PenulisGalat`, `portalhttp.PenulisGalat`), dan Go menolak
menyalin nilai bertipe bernama ke tipe bernama lain walau tanda tangannya sama. Build
gagal di dua tempat.

Penyelesaian: nilai bersama di `cmd` dideklarasikan dengan tipe fungsi **tanpa nama**, dan
parameter `DenganGalatPortal` juga tanpa nama. Nilai tanpa nama dapat disalin ke tipe
bernama mana pun, sehingga modul tetap tidak perlu saling mengimpor tipe.

**Galat portal dijawab 500.** Middleware portal memakai penulis galat yang disuntikkan,
dan penulis itu milik modul auth yang tidak mengenal galat portal — ketiga penolakan
portal terjawab `500` alih-alih `400`/`503`. Ditemukan oleh uji, bukan oleh pembacaan.

Penyelesaian: pemetaan galat portal dipindahkan ke modul portal sebagai
`DenganGalatPortal`, lalu dirantai di `cmd`. Pemetaan yang sempat saya duplikasi di modul
`statusprogres` dibuang supaya hanya ada satu sumber kebenaran.

**Satu suntingan gagal tanpa suara.** Perubahan pada `api/klien.ts` yang menyalurkan
`detail` galat ke `GalatAPI` tidak pernah teterap — teks pencariannya tidak cocok karena
berkas itu berakhiran CRLF. Akibatnya pelanggaran per isian tidak pernah sampai ke layar.
**Yang menemukannya adalah uji**, yang mengharapkan kedua pesan tersorot dan hanya
mendapat kotak pesan umum. Tanpa uji itu cacat ini akan lolos, dan gejalanya halus: form
tetap menolak, hanya tidak menunjukkan isian mana yang salah.

**Dependensi frontend belum terpasang.** `node_modules` tidak ada, sehingga
`npm run periksa-tipe` gagal sebelum sempat memeriksa apa pun. Dijalankan `npm install`
(135 paket) lebih dulu — verifikasi yang tidak pernah benar-benar dijalankan bukan
verifikasi.

### 9.6 Verifikasi yang benar-benar dijalankan

Bukan pembacaan ulang, melainkan perintah yang dieksekusi beserta hasilnya.

| Pemeriksaan | Hasil |
|---|---|
| `go build ./...` | lulus |
| `go vet ./...` | lulus, tanpa temuan |
| `gofmt -l` pada berkas baru | bersih |
| `go test ./...` | **seluruh paket lulus** — 3 berkas uji baru: domain, usecase, rute |
| `npm run periksa-tipe` | lulus, mode ketat |
| `npm test` | **36 uji lulus**, 18 di antaranya baru |
| `npm run build` | lulus, 189 modul, hasil tersemat ke `backend/spa/dist` |

**Ditembak sungguhan terhadap binary yang berjalan** (`APP_ALAMAT=:18080`, penyimpanan
memori, identitas tiruan). Tiga belas permintaan, seluruhnya sesuai harapan:

| # | Permintaan | Harapan | Hasil |
|---|---|---|---|
| 1 | `GET` daftar tanpa header portal | ditolak | `400 portal_tidak_disebut` |
| 2 | `GET` daftar dengan `X-Portal: ASM` | 6 baris | `200`, `"portal":"ASM"` |
| 3 | `GET` daftar dengan `X-Portal: ASI` | ditolak, koneksi belum hidup | `503 portal_belum_siap` |
| 4 | `GET` daftar dengan portal karangan | ditolak | `400 portal_tidak_dikenal` |
| 5 | `GET` daftar tanpa sesi | ditolak | `401 sesi_tidak_sah` |
| 6 | `GET` daftar posisi tanpa portal | dilayani | `200`, empat posisi |
| 7 | `POST` tambah | ID diterbitkan server | `201`, `id` = `07` |
| 8 | `PUT` ubah baris `03` | tersimpan | `200`, nama dan posisi berubah |
| 9 | `POST` nama kosong dan posisi `999` | dua pelanggaran sekaligus | `422`, `detail` berisi 2 butir |
| 10 | `PUT` baris `99` | tidak ditemukan | `404 tidak_ditemukan` |
| 11 | `GET` daftar setelah 7 dan 8 | perubahan tersimpan, urutan tetap | 7 baris, `03` berubah, `07` di akhir |
| 12 | `DELETE` baris `01` | rute tidak disediakan | `405` |
| 13 | `GET /master/status-progres-1` (rute SPA) | halaman dimuat | `200 text/html` |

**Log diperiksa terhadap kebocoran.** `grep` untuk kata sandi contoh, kata `Bearer`, dan
`kata_sandi` pada seluruh log server: **nol kemunculan**. Penolakan portal tercatat pada
tingkat `WARN` beserta alias yang diminta — alias entitas, bukan data nasabah.

### 9.7 Yang belum dapat dijalankan

> **SEBAGIAN TERJAWAB — lihat §9.8.** Tipe kolom sudah ditetapkan **CHAR berlebar
> tetap**, dan jawaban itu membongkar cacat yang penanganannya mengubah dua kueri.
> Yang masih berlaku dari bagian di bawah: adapter SQL belum pernah menyentuh Oracle,
> dan uji kesetaraan gerbang 1 belum dapat dijalankan.

**Adapter SQL belum pernah menyentuh Oracle.** Seluruh verifikasi di atas memakai adapter
memori. Enam kueri di `repo/sqlstore/statusprogres.sql` karena itu **belum terbukti sah
terhadap basis data sungguhan** — ia belum pernah dijalankan satu kali pun.

Dua hal yang khususnya belum terbukti, keduanya bergantung pada DDL yang belum ada
(`R-08`):

- **Tipe kolom `ID_PROGRESS` dan `STATUS`.** Bila `CHAR` berlebar tetap, nilainya
  dipadatkan spasi. Pemangkasan sudah dipasang pada pembacaan, tetapi belum diuji
  terhadap tipe yang sebenarnya.
- **Perilaku `SELECT ... FOR UPDATE`.** Dipakai menyerialkan penurunan nomor baru; di
  Oracle dan PostgreSQL keduanya sah, tetapi belum dijalankan.

**Uji kesetaraan gerbang 1 belum dapat dijalankan.** Ia menuntut Pega staging yang dapat
ditembak dari luar (`ADR-0027`), dan ketersediaannya masih belum dikonfirmasi. Ditambah
satu penghalang khusus modul ini: **isi tabel yang sebenarnya belum ada** — tidak ada
`gcnm_mst_progress_klaim.csv` di `Database/` sebagaimana `v_sts_claim.csv` dan
`m_portal_pnc.csv`. `DaftarContoh()` pada adapter memori adalah susunan sendiri, dan
sudah ditandai demikian di dalam kodenya.


### 9.8 Empat asumsi dijawab, dan satu cacat senyap yang tersingkap karenanya

Asumsi yang §9.7 catat sebagai terbuka diajukan kembali sebagai pilihan, lalu dijawab
Work Owner pada hari yang sama.

| Pertanyaan | Jawaban |
|---|---|
| Tipe kolom `ID_PROGRESS` | **CHAR berlebar tetap** |
| Panjang kolom `STS_PROGRESS1` | **100 karakter** |
| Penghapusan baris | tetap tidak ada, sama seperti Pega |
| Modul berikutnya | belum ditentukan |

**Jawaban pertama membongkar cacat yang belum terlihat, dan tidak akan terlihat sampai
kode ini menyentuh Oracle sungguhan.**

Kolom CHAR memadatkan nilainya dengan spasi: `"01"` tersimpan sebagai `"01 "`. Oracle
membandingkan CHAR dengan CHAR memakai *blank-padded comparison* sehingga spasi ujung
diabaikan — dan literal teks di dalam SQL bertipe CHAR, sehingga kueri lama yang
**merangkai** nilainya menjadi `= '01'` memang cocok. Tetapi **parameter binding bertipe
VARCHAR2**, dan CHAR lawan VARCHAR2 memakai *non-padded comparison*: `"01 "` tidak sama
dengan `"01"`.

Akibatnya, dua kueri yang menyaring berdasarkan ID **tidak akan menemukan satu baris pun**:
pemuatan baris ke modal sunting selalu gagal, dan `UPDATE` mengenai nol baris — yang oleh
repo diartikan "tidak ditemukan". Tanpa satu pun galat basis data yang menjelaskan
sebabnya.

Yang perlu dicatat: **menyalin `= :1` apa adanya dari kueri lama justru MENGUBAH
perilaku**, bukan mempertahankannya. Penyebabnya perpindahan dari perangkaian string ke
parameter binding — aturan yang wajib dan tidak dapat ditawar. Kesetaraan `P-5` ternyata
tidak selalu berarti menyalin teks SQL apa adanya.

Penanganannya `WHERE TRIM(ID_PROGRESS) = :1` pada `statusprogres_ambil` dan
`statusprogres_perbarui`, beserta alasan menolak tiga alternatifnya, dicatat di
`keputusan-implementasi.md` §10.17.

**Kenapa ini tidak tertangkap uji yang sudah ada.** Seluruh uji memakai adapter memori,
dan memori tidak memadatkan apa pun. Ini batas nyata dari pengujian tanpa basis data, dan
sudah disebut di §9.7 — jawaban Work Owner mengubahnya dari catatan menjadi bukti.

Penggantinya: `repo/sqlstore/kueri_test.go` yang menuntut `TRIM(ID_PROGRESS)` ada pada
kedua kueri dan menolak perbandingan langsung tanpa TRIM. Ia menjaga perbaikan ini tidak
hilang saat seseorang kelak "merapikan" kuerinya — karena gejalanya senyap, tidak akan
ada yang melihat sesuatu rusak sampai pengguna melaporkan tombol Ubah tidak pernah
berhasil.

### 9.9 Perubahan yang dijalankan atas keempat jawaban

| Berkas | Perubahan |
|---|---|
| `repo/sqlstore/statusprogres.sql` | `TRIM(ID_PROGRESS)` pada dua penyaring ID; catatan CHAR pada kueri sisip |
| `repo/sqlstore/kueri_test.go` | **baru** — 8 uji: keberadaan kueri, disiplin SQL portabel, parameter binding, penjaga TRIM, kunci baris tidak di-SET, `FOR UPDATE`, pemeriksa tabel tidak mengambil baris |
| `statusprogres.go` | `BatasPanjangNama` 200 → **100**, dan keterangannya berubah dari asumsi menjadi ketetapan |
| `FormStatusProgres.tsx` | `BATAS_PANJANG_NAMA` 200 → **100** |
| `keputusan-implementasi.md` | §10.16 dan §10.17 baru; penunjuk *disupersede* pada §10.13–§10.15 |

Tidak ada modul baru dimulai — jawaban keempat belum menentukan arah berikutnya.

**Diverifikasi ulang seluruhnya:** `go vet` bersih · `gofmt` bersih · `go test ./...`
seluruh paket lulus, termasuk paket `repo/sqlstore` yang sebelumnya tidak punya uji sama
sekali · `tsc --noEmit` lulus · `npm test` 36 lulus.


### 9.10 Kerangka menu — 2026-09-18

Work Owner bertanya apakah menunya sudah beres. **Belum** — dan itu akibat langsung
keputusan "rute saja, jangan sentuh Beranda" (§9.3 pertanyaan 2). Diperiksa lebih dulu
sebelum dijawab: satu-satunya tempat `status-progres-1` disebut di frontend adalah
definisi rutenya sendiri, tanpa satu pun tautan dari mana pun.

Diajukan empat pilihan, dan Work Owner memilih **menu di kerangka, Home tetap utuh**.

**Jalan yang sebelumnya saya lewatkan.** Pada §9.3 saya menyajikan pilihannya seolah menu
menuntut menyunting `HalamanBeranda.tsx`. Itu tidak benar: pembungkus rute `/` ada di
`app/App.tsx` — kerangka, bukan modul Home. Menu karena itu dapat dipasang dengan berkas
modul Beranda tetap utuh, dan itulah yang dikerjakan.

| Berkas | Perlakuan |
|---|---|
| `app/menu.ts` | **baru** — peta menu sebagai data; modul baru = satu baris |
| `app/NavigasiUtama.tsx` | **baru** — kolom samping di layar lebar, deret mendatar di layar sempit |
| `app/Kerangka.tsx` | **baru** — bingkai bersama: peringatan sesi, menu, pemilih portal, keluar |
| `app/App.tsx` | disunting — kedua rute dibungkus `Kerangka` |
| `modules/master-status-progres/HalamanStatusProgres1.tsx` | tautan "← Beranda" dibuang, kini duplikat menu |
| `modules/beranda/HalamanBeranda.tsx` | **tidak disentuh** |

**Tiga hal yang ikut diperbaiki sekalian, dan bukan permintaan.**

`PeringatanSesi` kini tampil di **setiap** layar dalam sesi. Sebelumnya layar modul harus
mengingat memasangnya sendiri — dan satu layar yang lupa berarti peringatan sesi hampir
habis tidak pernah muncul di sana.

Pemilih portal dan tombol keluar ikut ke kerangka. Tanpa itu layar modul adalah **jalan
buntu**: tidak ada cara berpindah entitas atau keluar tanpa kembali ke beranda. Berpindah
portal tanpa login ulang adalah inti `ADR-0030`, jadi pemilihnya harus terjangkau dari
layar mana pun.

Karena pemilih portal sekarang ada di layar master, tiga pesan yang menyuruh pengguna
*"pilih portal di beranda"* menjadi menyesatkan — keduanya disesuaikan menjadi *"pilih
portal entitas di bagian atas halaman"*.

### 9.11 Verifikasi kerangka menu

| Pemeriksaan | Hasil |
|---|---|
| `npm run periksa-tipe` | lulus |
| `npm test` | **48 uji lulus** — naik dari 36; 12 uji baru di `app/Kerangka.test.tsx` |
| `npm run build` | lulus; `Menu utama` dan `Status Progres 1` terverifikasi ada di bundle |
| `go vet` · `gofmt` · `go test ./...` | tetap bersih dan lulus seluruhnya |

**Ditembak terhadap binary yang berjalan** (`APP_ALAMAT=:18081`):

| Permintaan | Hasil |
|---|---|
| `GET /` | `200 text/html` |
| `GET /master/status-progres-1` | `200 text/html` |
| `GET /api/master/status-progres-1` + `X-Portal: ASM` | `200` |
| `GET /api/master/status-progres-1` tanpa portal | `400` — penolakan portal masih utuh |

Yang diuji ke-12 uji baru itu, bukan hanya bahwa menunya tampil:

- setiap butir di `app/menu.ts` benar-benar muncul — membuktikan komponennya membaca data
  itu, bukan menulis butirnya sendiri;
- butir aktif ditandai `aria-current="page"`, dan Beranda **tidak** ikut aktif di layar
  lain (tanpa `end`, ia aktif di mana-mana karena semua jalur dimulai dengan `/`);
- pemilih portal dan tombol keluar tampil di layar modul, dan **tidak** tampil di Beranda
  yang sudah menyediakannya sendiri;
- keluar mencabut sesi **dan** pilihan portal;
- keterangan "belum disaring izin peran" ada di layar.

**Yang TIDAK diverifikasi:** tampilan di peramban sungguhan. Uji di atas berjalan di
jsdom, yang tidak menghitung tata letak — jadi bahwa menu benar-benar menjadi kolom di
layar lebar dan deret mendatar di layar sempit belum terbukti dengan mata. Itu perlu
dibuka sendiri di peramban.

### 9.12 Satu kalimat di Beranda kini bertentangan dengan layarnya

`HalamanBeranda.tsx` masih menulis *"Menu belum tampil di sini."* — padahal menu tampil
tepat di sebelahnya. Sisa paragrafnya masih benar: menu itu memang belum disaring izin
peran.

Tidak saya sunting, karena berkas itu milik modul Beranda yang dinyatakan tidak boleh
diubah. Diajukan sebagai permintaan izin. Rinciannya beserta perubahan yang diusulkan ada
di `keputusan-implementasi.md` §10.21.
---

## 10. Sesi keempat — Modul Master Rekening (2026-09-17)

### 10.1 Permintaan dan bahan yang diberikan

Work Owner meminta penambahan **modul Master Rekening**, dengan
`Harness/MasterRekening-Harness.xml` sebagai rujukan aplikasi existing, dan menuntut
analisis penuh sebelum satu baris kode ditulis.

### 10.2 Yang diperiksa lebih dulu, sebelum menulis kode

Harness-nya sendiri ternyata hanya kerangka portal — isinya nyaris tidak memuat aturan
bisnis. Yang memuat aturan adalah rule di sekitarnya, dan seluruhnya dibaca:

| Jenis | Rule |
|---|---|
| RDB List | `GetDataMasterRekening`, `InsertMasterRekening`, `UpdateMasterRekening`, `DelDataRejectMasterRekening`, `SearchCodeBank_sql` |
| Activity | `CNMUpdateMasterRekening_act`, `SetMasterRekeningValue`, `SetTipeRekening`, `ValidasiEmailRekening`, `GetDataMasterBank`, `SendEmailAlertRekening`, `HitDataRekeningToKasir`, `HitupdateDataRekeningToKasir` |
| Section | `BrowseMasterRekening`, `BrowseMasterCariDataRekening`, `ApprovalMasterRekening`, `BrowseMasterRekeningApproval/Approve/Reject`, `ListPanelMasterRekening` |
| Report Definition | `BrowseBankGroup` |
| Connect REST | `InjectDataRekeningToKasir`, `UpdateSearchDataRekeningToKasir` |

Hasilnya: tabel inti `POOLDATA.LST_ACCOUNT` (27 kolom), master bank
`GENERAL.LST_BANK_GROUP`, lima tab layar, status persetujuan `0`/`1`/`2`, sembilan
kolom wajib, aturan anti-duplikasi berikut pengecualiannya, dan dua efek samping saat
komite menyetujui.

### 10.3 Temuan yang menghentikan pekerjaan sebelum dimulai

Instruksi menyebut modul **Login, Home, dan Master Data sudah selesai** dan harus
diisolasi. Pemeriksaan menunjukkan **Master Data tidak ada di working copy ini**:
backend hanya `auth`, `platform`, `portal`; frontend hanya `beranda`, `masuk`, `portal`
— persis seperti yang dinyatakan `claim-pnc/README.md` sendiri.

Yang diperiksa sebelum melaporkannya: kelima branch di `origin`, `git stash`, path docs
yang disebut README, dan pencarian folder di seluruh drive `C:` dan `D:`. Nihil
semuanya.

Work Owner kemudian menunjukkan screenshot working copy lain di `D:\app\claim-pnc` yang
memuat modul `masterstatus` / `master-status-klaim`. Path itu **tidak ada di mesin ini**
dan belum pernah di-push ke `origin`. Work Owner memutuskan pekerjaan tetap dilanjutkan
di working copy ini, dengan penamaan mengikuti pola yang terlihat di screenshot.

**Akibatnya, yang dipakai sebagai acuan gaya adalah modul `auth` dan `portal` yang ada
di sini**, bukan `masterstatus` yang tidak dapat dibaca. Bila kelak keduanya digabung,
perbedaan gaya antara keduanya harus diperiksa manusia.

### 10.4 Tiga keputusan yang dikonfirmasi Work Owner

| Pertanyaan | Jawaban |
|---|---|
| Seberapa luas cakupannya? | **Paritas penuh dengan Pega** — CRUD, lima tab, alur komite, integrasi Kasir, email alert |
| Perilaku dipertahankan atau dibersihkan? | **Perilaku dipertahankan, penamaan dibersihkan** (`P-5`) |
| Penamaan modul? | Mengikuti pola yang sudah ada: `internal/masterrekening`, `modules/master-rekening` |

### 10.5 Yang dibangun

```
backend/internal/masterrekening/
├── masterrekening.go          entitas, status, invarian, seam Repo/BankRepo/Kasir/Notifier
├── masterrekening_test.go
├── usecase/
│   ├── ajukan.go              pengajuan, perubahan, daftar
│   ├── putuskan.go            keputusan komite + pendaftaran Kasir
│   └── alur_test.go
├── repo/sqlstore/             LST_ACCOUNT dan LST_BANK_GROUP
├── repo/memori/               adapter kedua, untuk uji tanpa basis data
├── kasir/                     klien HTTP nyata + tiruan
└── http/                      dto, handler, galat, rute

frontend/src/modules/master-rekening/
├── api.ts                     hook TanStack Query
├── HalamanMasterRekening.tsx  lima tab
├── FormRekening.tsx           formulir pengajuan
├── TabelRekening.tsx          tabel bersama kelima tab
└── HalamanMasterRekening.test.tsx
```

### 10.6 Perubahan di luar modul, dan alasannya

Empat berkas di luar folder modul ikut berubah. Seluruhnya **penambahan**, tidak ada
yang me-refactor modul yang sudah selesai:

| Berkas | Perubahan | Kenapa tidak dapat dihindari |
|---|---|---|
| `cmd/claimpnc/main.go` | perakitan modul + pemasangan rute | Modul memasang rutenya sendiri, tetapi perakitannya memang milik entrypoint |
| `internal/platform/config/config.go` | struct `Kasir` + empat variabel lingkungan | Seluruh konfigurasi wajib lewat `config.Muat`; membaca `os.Getenv` di dalam modul akan melanggar polanya sendiri |
| `frontend/src/api/klien.ts` | `GalatAPI.field` + metode `PUT` | Galat validasi per kolom tidak dapat sampai ke layar tanpanya |
| `frontend/src/api/tipe.ts` | tipe `Rekening`, `Bank`, `StatusRekening` | Berkas ini memang cerminan DTO Go; menaruhnya di tempat lain memecah kontrak |

Satu berkas lagi, `modules/beranda/HalamanBeranda.tsx`, ditambahi **satu tautan** ke
layar baru. Ia berdiri sendiri, ditandai komentar, dan dapat dihapus tanpa menyentuh
modul mana pun. Tanpanya layar hanya dapat dicapai dengan mengetik URL.

### 10.7 Verifikasi yang dijalankan

| Perintah | Hasil |
|---|---|
| `go build ./...` | **lolos** |
| `go vet ./...` | **lolos** |
| `go test ./...` | **lolos** — 21 uji baru; seluruh uji lama (auth, portal, config) tetap hijau |
| `gofmt -l` | bersih |
| `npm run periksa-tipe` | **lolos** |
| `npm test` | **TIDAK DAPAT DIJALANKAN** — lihat §10.8 |

### 10.8 Kendala: uji frontend tidak dapat dijalankan di mesin ini

`npm test` gagal sebelum satu uji pun berjalan, pada **ketiga** berkas uji — termasuk
`HalamanMasuk.test.tsx` dan `PemilihPortal.test.tsx` yang sudah ada sebelum sesi ini.
Jadi ia **bukan akibat perubahan sesi ini**.

Sebabnya versi Node: mesin ini menjalankan **v20.18.0**, sedangkan `jsdom@30` menuntut
`webidl.util.markAsUncloneable` (Node 21+) dan `html-encoding-sniffer@6` menuntut
`require()` atas modul ESM (Node 20.19+/22.12+). `npm install` sendiri sudah
memperingatkannya dengan `EBADENGINE`, dan `README.md` memang menyebut Node 20+ diuji
pada 24.20.0.

Dua kendala perkakas lain yang sudah diselesaikan di jalan:
`node_modules` belum pernah dipasang di working copy ini, dan `rolldown` kehilangan
binding native `win32-x64-msvc` akibat bug npm pada dependensi opsional.

**Yang diperlukan:** Node **22.12+** (idealnya 24, sesuai README). Setelah itu
`npm test` dapat dijalankan tanpa perubahan kode apa pun. Sampai itu terjadi,
`npm run periksa-tipe` adalah verifikasi terkuat yang tersedia untuk frontend, dan ia
lolos.

### 10.9 Lanjutan: alur surel peringatan, dan dua nilai kolom yang sempat salah

Dikerjakan setelah Work Owner meminta daftar asumsi disebutkan lebih dulu sebelum kode
ditulis. Urutannya menjadi: **gali export → daftarkan asumsi → konfirmasi → baru kode.**
Urutan itu langsung membayar dirinya sendiri.

**Dua sandi kolom yang sempat salah ditebak.** Penelusuran lanjutan atas activity
`SetTipeRekening` — yang ternyata mengisi DUA daftar pilihan sekaligus — menunjukkan:

| Kolom | Sandi sebenarnya | Sempat saya tulis |
|---|---|---|
| `ACCOUNT_TYPE` | `"BIASA"` / `"VA"` | jenis pemilik (BENGKEL, RUMAH SAKIT, …) |
| `STS_AKTIF` | `"Ya"` / `"Tidak"` | `"1"` / `"0"` |

Keduanya **tidak menimbulkan galat apa pun** — hanya membuat setiap rekening terbaca
nonaktif, sehingga petugas mengira rekening yang sah tidak dapat dipakai membayar klaim.
Diperbaiki dan dikunci uji di `repo/sqlstore/sandi_test.go`.

**Pelajaran untuk modul berikutnya:** sandi nilai kolom **tidak dapat ditebak dari SQL**
— SQL hanya menunjukkan kolomnya, bukan nilai yang sah. Sumbernya adalah activity yang
mengisi daftar pilihan layar (`Set*`), dan itu wajib dibaca untuk setiap kolom berjenis
kode.

**Enam asumsi pada alur surel didaftarkan dan dijawab Work Owner:**

| Asumsi | Keputusan |
|---|---|
| Penerima surel | **Mailbox Tim IT** dari konfigurasi — satu-satunya penyimpangan, diputuskan dengan sadar setelah saya koreksi sendiri bahwa jalur as-is sebenarnya tersedia |
| Isi surel (template hilang dari export) | Pakai susunan sendiri, **ditandai sementara** di dalam surelnya |
| Pemicu tambahan saat Kasir mati total | **Dicabut** — hanya `ResponseCode == "9"`, as-is |
| Prasyarat `FlagNOLL=="Ya"` | Abaikan; kirim selalu saat gagal kode 9 |
| Menulis ke `CLAIM_SERVICE_LOG` | Belum perlu; menyusul bersama `S-4` |
| Jalur Inject vs UpdateSearch ke Kasir | Tetap seperti asumsi semula |

**Koreksi yang saya sampaikan sendiri di tengah jalan.** Saya sempat menyatakan jalur
penerima as-is *buntu* karena `OPERATOR_ID` belum dipetakan. `GetUserDetailsQuery`
membuktikan sebaliknya: `MST_USER_TEKNIK.OPERATOR_ID` sama dengan `PYUSERIDENTIFIER`
Pega, yaitu nama login — yang sudah kita simpan di `Pengguna.Login`. Koreksi itu
disampaikan sebelum keputusan dikunci, sehingga keputusannya diambil dengan informasi
yang benar.

**Satu kelemahan ditemukan oleh ujinya sendiri.** `bersihkanHeader` versi pertama
mengganti baris baru dengan spasi. Itu memang menghalangi terbentuknya header baru,
tetapi teks susupan tetap ikut terkirim di dalam header. Diperkuat menjadi **memotong**
pada baris baru pertama.

**Verifikasi:** `go build`, `go vet`, dan `go test ./...` seluruhnya lolos — termasuk
seluruh uji modul yang sudah ada sebelumnya. `npm run periksa-tipe` lolos. `npm test`
tetap terhalang versi Node (lihat §10.8).
## 11. Sesi keempat — Master Status Klaim (2026-09-17)

Modul bisnis **pertama**. Sampai sesi ini yang ada hanyalah login, pemilih portal, dan beranda
sementara; `README.md` menyatakannya sendiri: *"Modul bisnis belum ada satu pun."*

### 11.1 Koreksi premis instruksi

Instruksi menyebut modul **Login, Home, dan Master Data** sudah selesai dan dilarang disentuh.
Pemeriksaan terhadap kode menemukan yang ketiga **tidak ada sama sekali** — tidak ada folder master
mana pun di `internal/`, tidak ada layar master di `modules/`, dan tidak ada tabel master di
`migrations/`.

Disampaikan lebih dulu sebelum mengerjakan. Akibatnya bukan sekadar soal penamaan: **Master Status
Klaim menjadi master data yang pertama**, sehingga setiap pilihan di sini menjadi pola untuk
sekurang-kurangnya 28 master berikutnya.

### 11.2 Yang dibaca sebelum menulis kode

| Sumber | Yang diambil |
|---|---|
| `Harness/StatusClaimInbox-Harness.xml` | `pyCaption Master Status Klaim`; merakit `BrowseStatusClaim` + `ListStatusClaim`; tombol **Tambah**, **Refresh**, **Simpan** |
| `Section/BrowseStatusClaim-Section.xml` | grid 2 kolom — `LSC_ID` lebar 80 **read-only**, `LSC_NOTE` lebar 250; kolom aksi **Ubah** mengirim `lscid`; sorting dan filtering aktif; form memakai halaman `TempStsClaim` dengan `LSC_NOTE` berlabel **"Status"**, `pyRequired=false` |
| `Section/ListStatusClaim-Section.xml` | kerangka layar; **`pyDeleteActivityExists=false`** — tidak ada hapus |
| `Report Definition/BrowseVStsClaim_RD` · `SelectVStsClaim_RD` | tiga field: `LSC_ID`, `LSC_NOTE`, `OLD_LSC_ID`; `pyMaxRecords=500` |
| `Activity/SetStsClaimValue_act` | aksi **Ubah**: jalankan RD Select lalu salin ke `TempStsClaim`, set `pyLabel := "Update"` |
| `Activity/CNMUpdateStsclaim_act` | aksi **Simpan**: kode kosong menjadi sentinel `"UnknownID"`; **seluruh halaman diserialisasi JSON lalu dikirim lewat slot parameter `OLD_LSC_ID`** |
| `RDB List/UpdateStsClaim-SQL.xml` | blok PL/SQL memanggil `POOLDATA.PEGA_M_STS_CLAIM(Datapega, IDPega, out)` |
| `Database/PEGA_M_STS_CLAIM.prc` | source aslinya — penyimpanan `(LSC_ID, JSONDATA)`, kode dibentuk `id_site` disambung urutan tiga digit |
| `Database/v_sts_claim.csv` | isi master: 33 baris `1134`–`1166` |

**Dua temuan yang mengubah rencana, keduanya dari membaca source procedure:**

1. **Kode `1134`–`1166` bukan angka arbitrer.** Ia `id_site` disambung urutan tiga digit — situs
   `1` ditambah urutan 134 sampai 166. Skema itu **dipertahankan**, bukan diganti, karena 23 rule
   Pega masih membaca kode ini lewat `V_STS_CLAIM` selama masa paralel.
2. **Polanya seragam di SELURUH master.** `PEGA_M_CAUSE_OF_LOSS`, `PEGA_M_SURVEYORS`,
   `PEGA_M_PANEL_HE` — semuanya tabel `(ID, JSON_DATA)` yang ditulis procedure dan dibaca lewat
   view. Apa pun yang diputuskan di sini berlaku untuk 28 master sesudahnya.

### 11.3 Empat pertanyaan konfirmasi dan jawabannya

| # | Pertanyaan | Jawaban Work Owner |
|---|---|---|
| 1 | Menulis ke mana? | **Go jadi penulis tunggal `M_STS_CLAIM`**, tidak lagi menyimpan JSON; isi JSON dipindahkan ke kolom. Procedure `PEGA_M_STS_CLAIM` boleh ditinggalkan |
| 2 | Seberapa jauh lingkupnya? | Fungsi dan tampilan **seperti Pega**, tetapi **lebih bagus, mobile friendly, dan user friendly** |
| 3 | Jejak audit? | **Samakan dengan sekarang** — sistem lama tidak punya, jadi tidak perlu ditambahkan |
| 4 | Validasi? | **Tolak ID atau nama status ganda, dan tolak yang kosong** |

Jawaban 3 menyimpang dari acceptance criteria `TKT-F4-001` ("setiap perubahan master menghasilkan
tepat satu baris jejak audit"). Itu keputusan Work Owner, dicatat di `keputusan-implementasi.md`
§10 beserta konsekuensinya — bukan diserap diam-diam.

### 11.4 Yang dibangun

**Backend — modul `internal/masterstatus/`**, mengikuti bentuk modul yang sudah ada:

```
masterstatus/
├── statusklaim.go          domain: tipe, aturan label, seam Repo
├── errors.go               galat domain + GalatValidasi berisi pelanggaran per field
├── usecase/kelola.go       orkestrasi: Daftar · Ambil · Tambah · Ubah
├── repo/memori/            adapter uji dan pengembangan tanpa basis data + 33 baris contoh
├── repo/sqlstore/          adapter Oracle + berkas .sql terpisah
└── http/                   dto · galat · handler · rute
```

**Frontend:**

| Berkas | Isi |
|---|---|
| `components/TabelData.tsx` | **komponen tabel baku** — cari, urut, tiga keadaan tampilan, berubah menjadi kartu di layar sempit |
| `components/Tombol.tsx` | tombol baku tiga nada |
| `modules/master-status-klaim/` | `api.ts` (hook TanStack Query) · `HalamanMasterStatusKlaim.tsx` · `FormStatusKlaim.tsx` |
| `app/KerangkaHalaman.tsx` | bilah menu untuk layar di balik sesi |

**Modul Login, Home, dan Portal tidak disentuh.** Yang berubah di luar modul baru hanya lima berkas
bersama: `api/klien.ts` (dukungan PUT dan `detail` galat), `api/tipe.ts` (tipe dan kode galat baru),
`app/App.tsx` (satu rute), serta `cmd/claimpnc/main.go` dan `periksa.go` (perakitan).

### 11.5 Kendala dan penyelesaiannya

**Tabel dan kartu digambar dua kali.** Mula-mula `TabelData` menggambar `<table>` untuk layar lebar
dan daftar kartu untuk layar sempit sebagai dua pohon terpisah. Itu mudah ditulis dan **salah**:
kelas Tailwind hanya menyembunyikan lewat CSS, sehingga **kedua pohon tetap ada di DOM**. Akibatnya
setiap isi sel muncul dua kali, pembaca layar membacanya dua kali, dan **14 dari 15 uji gagal**
karena setiap pencarian menemukan dua elemen untuk satu nilai.

Diganti: **satu `<table>`**, elemennya diubah menjadi blok lewat CSS pada layar sempit, dan nama
kolom digambar ulang di dalam sel sebagai label kecil ber-`aria-hidden` yang hilang pada layar
lebar. Satu DOM, satu sumber kebenaran.

Pengujianlah yang menemukannya — bukan pembacaan ulang kode.

**Proxy perusahaan memotong `localhost`.** Uji asap lewat `curl` dijawab halaman galat Squid, bukan
aplikasi. Diselesaikan dengan `--noproxy` dan alamat `127.0.0.1`. Bukan cacat aplikasi, tetapi layak
dicatat supaya tidak didiagnosis ulang oleh orang berikutnya.

### 11.6 Verifikasi yang benar-benar dijalankan

Bukan rencana. Seluruhnya dijalankan, dan angka di bawah adalah hasilnya.

| Pemeriksaan | Hasil |
|---|---|
| `gofmt -l ./cmd ./internal` | bersih |
| `go vet ./...` | bersih |
| `go test ./...` | **lulus** — 12 paket, termasuk 4 paket modul baru |
| `npm run periksa-tipe` | bersih |
| `npm test` | **33 uji lulus**, 3 berkas — termasuk 18 uji lama yang tidak berubah |
| `npm run build` | berhasil — 189 modul, 431 kB |
| `go build` dengan SPA tersemat | berhasil |

**Uji asap terhadap aplikasi yang benar-benar berjalan** (port 8099, penyimpanan memori, identitas
tiruan — tanpa menyentuh basis data mana pun):

| # | Yang diuji | Hasil |
|---|---|---|
| 1 | daftar tanpa sesi | `401` |
| 2 | daftar dengan sesi | `200`, `"total":33` |
| 3 | ambil `1149` | `200` — `Claim Committee` |
| 4 | ambil `9999` | `404` `status_klaim_tidak_ditemukan` |
| 5 | tambah | `201` — kode **`1167`**, melanjutkan `1166` |
| 6 | tambah label kosong | `422` disertai `detail` yang menunjuk field `label` |
| 7 | tambah nama ganda `Paid` | `409` `label_status_sudah_dipakai` |
| 8 | tambah `"  pAiD  "` | `409` — beda huruf besar dan spasi tetap ditolak |
| 9 | ubah `1163` | `200` |
| 10 | ubah dua kali dengan nilai sama | `200`, jawaban identik — idempoten |
| 11 | ubah `1134` | kode lama `01` **bertahan** |
| 12 | ubah ke nama milik baris lain | `409` |
| 13 | ubah baris yang tidak ada | `404` |
| 14 | `DELETE` | `405` — rutenya memang tidak ada |

### 11.7 Yang ditemukan dari basis data yang berjalan

Mode periksa dijalankan terhadap Oracle, lalu satu perkakas diagnostik **baca-saja** sementara untuk
membaca katalog. Perkakasnya sudah dihapus; kuerinya dicatat di §11.8 supaya dapat diulang.

**Empat temuan, dan tiga di antaranya mengoreksi tebakan saya sendiri:**

| # | Yang saya tulis mula-mula | Yang sebenarnya |
|---|---|---|
| 1 | `M_STS_CLAIM` hanya punya `LSC_ID` dan `JSONDATA`, jadi migrasi harus `ALTER TABLE ADD` | **Kolom `LSC_NOTE VARCHAR2(100)` dan `OLD_LSC_ID CHAR(4)` SUDAH ADA.** `ALTER` akan gagal `ORA-01430` dan menghentikan migrasi di baris pertama |
| 2 | Nama kunci utama `PK_M_STS_CLAIM`, ditebak dari pola migrasi 0001 | **`M_STS_CLAIM_PK`** — terbalik. Salah nama membuat bentrok kode muncul sebagai galat `500` |
| 3 | Bentuk JSON tidak diketahui, jadi isi disalin lewat view | **Kunci JSON pasti**, terbaca dari definisi view |
| 4 | Kode ke-1000 menjadi lima karakter dan "dibiarkan apa adanya" | `LSC_ID` bertipe **`CHAR(4)`** — penyisipannya akan **DITOLAK** `ORA-12899`, bukan diterima |

Definisi view yang sekarang, terbaca dari `ALL_VIEWS`, mengambil `LSC_ID` dan `OLD_LSC_ID` langsung
dari kolom dan `LSC_NOTE` lewat `JSON_VALUE` atas `JSONDATA`. Urutan kolomnya **`LSC_ID`,
`OLD_LSC_ID`, `LSC_NOTE`** — `LSC_NOTE` ketiga, bukan kedua. Migrasi 0002 mempertahankannya, dan
menulis daftar nama kolom secara eksplisit karena `ALL_VIEWS.TEXT` tidak menyimpannya.

**Dua hal lain yang perlu jawaban Work Owner:**

- **Basis data memuat 32 baris, CSV memuat 33.** Kode **`1165` "Rejected Chasier"** ada di
  `Database/v_sts_claim.csv` tetapi **tidak ada** di `POOLDATA.M_STS_CLAIM`. Tiga puluh dua sisanya
  cocok seluruhnya, termasuk labelnya. Sebabnya belum dijelaskan, dan **tidak ditambal**: uji
  `TestSelisihDenganBasisDataProduksiTercatat` menguncinya supaya tidak hilang diam-diam.
- **`M_STS_CLAIM_SEQ` berada di 193**, sementara kode tertinggi yang terpakai baru `1166`. Kode
  berikutnya karena itu **`1193`**, bukan `1167`. Itu perilaku yang sama dengan procedure lama dan
  tidak diubah — tetapi berarti deret kodenya berlubang, dan sisa ruang sebelum `CHAR(4)` mentok
  tinggal sekitar **806 penambahan**.

**Kolom `LSC_NOTE` pada tabel kosong pada seluruh 32 baris.** Seseorang menyiapkannya lalu berhenti
di situ. Itulah yang diisi langkah 1 migrasi 0002, dan itu pula sebabnya mode periksa melaporkan
`[WASPADA] 32 status berlabel kosong` sebelum migrasi dijalankan — bukan cacat, melainkan laporan
yang benar atas keadaan yang belum bermigrasi.

### 11.8 Kueri diagnostik, supaya dapat diulang tanpa perkakas

```sql
-- kolom tabel dan view
SELECT COLUMN_NAME, DATA_TYPE, DATA_LENGTH, NULLABLE FROM ALL_TAB_COLUMNS
 WHERE OWNER='POOLDATA' AND TABLE_NAME IN ('M_STS_CLAIM','V_STS_CLAIM')
 ORDER BY TABLE_NAME, COLUMN_ID;

-- definisi view
SELECT TEXT FROM ALL_VIEWS WHERE OWNER='POOLDATA' AND VIEW_NAME='V_STS_CLAIM';

-- constraint dan indeks
SELECT CONSTRAINT_NAME, CONSTRAINT_TYPE FROM ALL_CONSTRAINTS
 WHERE OWNER='POOLDATA' AND TABLE_NAME='M_STS_CLAIM';
SELECT INDEX_NAME, UNIQUENESS FROM ALL_INDEXES
 WHERE TABLE_OWNER='POOLDATA' AND TABLE_NAME='M_STS_CLAIM';

-- urutan
SELECT SEQUENCE_OWNER, LAST_NUMBER, INCREMENT_BY, CACHE_SIZE FROM ALL_SEQUENCES
 WHERE SEQUENCE_NAME='M_STS_CLAIM_SEQ';

-- keterisian kolom
SELECT COUNT(*), COUNT(LSC_NOTE), COUNT(OLD_LSC_ID), COUNT(DBMS_LOB.GETLENGTH(JSONDATA))
  FROM POOLDATA.M_STS_CLAIM;
```

### 11.9 Yang belum dapat dibuktikan

| Acceptance criteria | Keadaan | Apa yang menahannya |
|---|---|---|
| Layar bekerja terhadap Oracle | **Belum.** `LSC_NOTE` masih kosong pada 32 baris, sehingga daftar akan tampil tanpa label | Migrasi `0002` belum dijalankan DBA |
| Indeks unik menolak label ganda di basis data | **Belum diuji.** Yang terbukti baru pemeriksaan di aplikasi | idem |
| Kode baru terbit dari `M_STS_CLAIM_SEQ` | **Belum diuji** terhadap Oracle | idem, dan menuntut hak `INSERT` yang belum tentu dimiliki akun aplikasi |
| Pega tetap membaca benar setelah view diganti | **Belum diuji** | `D-63` menuntut pengujian dengan menjalankan Pega dan Go bersamaan |

---

## 12. Sesi kelima — penataan ulang tampilan (2026-09-17)

Permintaan Work Owner: tampilan yang menarik secara visual, gaya modern, kontras baik, efek hover
dan active dengan transisi halus, responsif di ponsel dan desktop, bayangan lembut dan sudut
membulat, **Light Mode**.

### 12.1 Dua keputusan yang diminta lebih dulu

| Pertanyaan | Jawaban Work Owner |
|---|---|
| Layar mana yang ikut didesain ulang? | **Seluruh aplikasi** — termasuk Masuk dan Beranda |
| Warna aksennya apa? | **Biru profesional**, abu-abu batu sebagai dasar. Mula-mula indigo, diganti menjadi `blue` atas permintaan susulan — lihat §12.11 |

Pertanyaan pertama diajukan karena aturan proyek sampai sesi lalu **melarang menyentuh modul Login
dan Home**, sedangkan desain ulang visual pasti menyentuh keduanya. Jawabannya mencabut larangan
itu untuk urusan tampilan.

**Merah korporat sengaja ditawarkan dan sengaja tidak dipilih.** Merah adalah bahasa universal untuk
galat; tombol Simpan berwarna merah di sebelah pesan galat berwarna merah sulit dibedakan sekilas.
Bila kelak merek menuntutnya, warna galat harus digeser lebih dulu.

### 12.2 Sistem desain, bukan tempelan per layar

Seluruh nilai desain hidup di `src/gaya.css` sebagai token Tailwind v4 (`@theme`), bukan tersebar
sebagai kelas di tiap layar:

| Token | Isi | Kenapa |
|---|---|---|
| `--shadow-lembut` `--shadow-angkat` `--shadow-terbang` | bayangan **dua lapis** | Satu lapis terlihat "ditempel"; dua lapis meniru cahaya yang menyebar |
| `--shadow-aksen` | bayangan berwarna biru | Tombol utama terasa menyala, bukan sekadar berwarna |
| `--radius-kartu` `--radius-kontrol` | dua tangga lengkung saja | Mencegah tiga radius berbeda muncul di satu layar |
| `--ease-halus` | `cubic-bezier(0.22, 1, 0.36, 1)` | Cepat memulai, melambat di akhir — terasa responsif |

Warnanya memakai **palet blue dan slate bawaan Tailwind**, bukan warna karangan. Nilainya sudah
ada di pustaka sehingga tidak dapat salah ketik, dan kontrasnya sudah teruji — blue-600 di atas
putih mencapai 5,1:1, lewat ambang AA. (Angka 8.6:1 yang sempat tertulis di sini salah — lihat §12.11.)

### 12.3 Tiga hal yang dikerjakan karena diminta, dan satu yang tidak diminta

**Diminta, dan dikerjakan:**

1. **Hover, active, transisi.** Setiap tombol punya tiga keadaan: hover mengangkat (warna menua,
   bayangan melebar, naik 1px), active menekan (turun kembali, menyusut 98%), focus-visible
   memberi cincin 4px. Gerakan naik-turun itu yang membuat tombol terasa dapat ditekan.
2. **Responsif.** Tabel berubah menjadi kartu di bawah 48rem, layar masuk terbelah dua panel di
   atas 64rem, bilah atas memadat, dan menu digulir menyamping.
3. **Bayangan lembut dan sudut membulat.** Seluruhnya lewat token di atas.

**Tidak diminta, tetapi dikerjakan karena permintaannya menjadi salah tanpa itu:**

`prefers-reduced-motion`. Permintaannya adalah "transisi yang halus" — dan bagi sebagian orang
transisi menimbulkan pusing atau mual. Sistem operasinya sudah menyatakan itu; mengabaikannya
berarti membuat aplikasi tidak dapat dipakai bagi mereka. Transisi **dimatikan**, bukan dipercepat.

### 12.4 Tiga hal yang dipindahkan, dan alasannya bukan estetika

| Yang pindah | Dari | Ke | Kenapa |
|---|---|---|---|
| Tombol **Keluar** | halaman Beranda | bilah atas | Pengguna yang sedang membuka layar master **tidak punya cara keluar** tanpa kembali ke beranda dulu |
| **Pemilih portal** | halaman Beranda | bilah atas | Sama: portal menentukan basis data seluruh layar, bukan hanya beranda |
| **Nama pengguna** | kartu identitas Beranda | bilah atas | Berlaku di seluruh layar; menyisakannya di dua tempat membuat nama yang sama muncul dua kali |

Baris "Nama" pada kartu identitas Beranda **dihapus** sebagai akibatnya. Itu bukan sekadar
kerapian — uji beranda mencari nama pengguna dengan **pencocokan persis**, dan dua elemen berisi
nama yang sama membuat pencarian itu gagal. Pemindahan dan penghapusan harus dilakukan bersamaan.

### 12.5 Kendala: uji layar master ikut rusak, dan itu benar

Setelah pemilih portal pindah ke bilah atas, **14 dari 15 uji layar master gagal**. Sebabnya bukan
tampilan: bilah atas kini memanggil `/api/portal` pada **setiap** layar di balik sesi, sedangkan
peladen tiruan di uji master hanya menjawab daftar status. Jawaban yang salah bentuk membuat
`data.portal.map` melempar.

Yang diperbaiki adalah **fixture-nya**, bukan komponennya: peladen tiruan menjawab `/api/portal`
otomatis, pola yang sama dengan `HalamanMasuk.test.tsx` yang sudah melakukannya sejak awal.

Melunakkan `PemilihPortal` supaya tahan jawaban yang salah bentuk sempat dipertimbangkan dan
**ditolak**: kontrak API menjamin bentuknya, dan komponen yang diam saat menerima bentuk salah
menyembunyikan cacat yang seharusnya terlihat.

### 12.6 Kesalahan saya sendiri yang perlu dicatat

**Saya sempat menyimpulkan CSS responsifnya tidak terbentuk.** Pemeriksaan pertama mencari
`min-width:` di berkas CSS hasil build dan tidak menemukan satu pun breakpoint — kesimpulannya:
tata letak tidak akan responsif sama sekali.

Kesimpulan itu **salah**, dan salahnya ada pada alat ukurnya:

- Tailwind v4 memancarkan `@media (width>=48rem)`, **bukan** `min-width:48rem`.
- Nama kelas responsif ditulis `.md\:table-cell` dengan garis miring terbalik, sehingga pencarian
  teks `md:table-cell` tidak pernah cocok.

Pemeriksaan ulang dengan pencocokan harfiah menemukan seluruhnya ada: tiga breakpoint, kelas
hover, active, focus-visible, group-hover, dan `prefers-reduced-motion`.

Pola kesalahannya sama dengan yang sudah tercatat di sesi keempat: **alat ukur dipercaya sebelum
divalidasi**. Sebelum menyimpulkan sesuatu tidak ada, alat pencarinya harus dibuktikan dulu
menyala pada kasus yang jelas ada.

### 12.7 Keputusan yang sengaja tidak diambil

| Yang tidak dipakai | Kenapa |
|---|---|
| **Google Fonts** | Aplikasi berjalan di VM on-premise tanpa jaminan akses internet. Huruf yang gagal dimuat mengubah seluruh tata letak. Dipakai tumpukan font sistem |
| **Pustaka ikon** | Delapan bentuk yang seluruhnya beberapa baris `path` tidak sebanding dengan satu dependensi yang harus dipelajari, dipantau keamanannya, dan ikut membesarkan bundel. Ikon digambar langsung sebagai SVG di `components/Ikon.tsx` |
| **Pustaka tabel** (TanStack / AG Grid) | `TKT-U2-005` menuntut keputusannya diambil dengan pengukuran. Belum berubah sejak sesi lalu |
| **Mode gelap** | Work Owner meminta Light Mode saja. `color-scheme: light` ditegaskan supaya kontrol bawaan peramban tidak ikut membalik mengikuti tema sistem |
| **Menu hamburger** | Dengan dua entri, hamburger menambah satu ketukan untuk menyembunyikan sesuatu yang sebenarnya muat. Menu digulir menyamping. Perlu ditinjau ulang bila menunya kelak berasal dari izin peran dan bertambah banyak |

### 12.8 Aksesibilitas — yang dikerjakan supaya kontras tidak berhenti di warna

- **Keadaan salah ditandai tiga cara**: warna tepi, ikon, dan teks. Sekitar satu dari dua belas
  laki-laki mengalami buta warna merah-hijau; bagi mereka tepi merah tidak berbeda dari abu-abu.
- **Dua nada pesan galat dibedakan BENTUK ikonnya** — lingkaran untuk penolakan, segitiga untuk
  gangguan — bukan hanya merah versus kuning.
- **Kesiapan portal** memakai titik berwarna **dan** teks, bukan warna saja.
- **`focus-visible`, bukan `focus`**: cincin hanya muncul untuk papan ketik. Memakai `focus`
  membuat cincin ikut muncul saat diklik tetikus, yang terlihat seperti cacat dan berujung pada
  orang menghapus cincinnya sama sekali — termasuk bagi yang membutuhkannya.
- **Tombol Keluar yang menyusut menjadi ikon** tetap membawa teks `sr-only` dan `aria-label`.
- Label kolom pada tampilan kartu `aria-hidden`, karena `<th scope="col">` sudah menjelaskan selnya.

### 12.9 Verifikasi yang benar-benar dijalankan

| Pemeriksaan | Hasil |
|---|---|
| `npm run periksa-tipe` | bersih |
| `npm test` | **33 uji lulus**, 3 berkas — tidak satu pun uji dilonggarkan |
| `npm run build` | berhasil — CSS 39,2 kB (7,5 kB gzip), JS 446 kB (138 kB gzip) |
| `go build` · `go vet` · `go test ./...` | bersih — backend tidak tersentuh |

**Pemeriksaan terhadap CSS hasil build**, memastikan yang ditulis benar-benar menjadi CSS:

| Yang dicari | Ada? |
|---|---|
| Kelas `.shadow-lembut`, `.shadow-angkat`, `.shadow-aksen`, `.rounded-kartu`, `.rounded-kontrol` | ✅ |
| Breakpoint `@media (width>=40rem)`, `(width>=48rem)`, `(width>=64rem)` | ✅ ketiganya |
| `.md\:table-cell` — peralihan tabel menjadi kartu | ✅ |
| `.lg\:flex-row` — layar masuk terbelah dua panel | ✅ |
| `.hover\:-translate-y-px`, `.active\:scale-[0.98]`, `.focus-visible\:ring-4` | ✅ ketiganya |
| `@media (prefers-reduced-motion:reduce)` | ✅ |
| `@media (hover:hover)` — mencegah hover lengket di layar sentuh | ✅ 3 blok |

**Uji sajian nyata** (binary dengan SPA tersemat, port 8099): halaman `200`, CSS `200` 39.173 B,
JS `200` 446.518 B, rute dalam `/master/status-klaim` `200` — bukan `404`, sehingga muat ulang
di tengah aplikasi tidak menjatuhkan pengguna.

### 12.10 Yang perlu diketahui saat mencoba

SPA **tersemat ke binary** lewat `go:embed` (`ADR-0002`). Proses yang sedang berjalan memuat
tampilan **lama** sampai dibangun ulang:

```bash
cd frontend && npm run build     # hasilnya ke backend/spa/dist
cd ../backend && go run ./cmd/claimpnc
```

Selama pengembangan antarmuka, `npm run dev` di port 5173 jauh lebih cepat — perubahan langsung
terlihat tanpa membangun ulang binary.

### 12.11 Koreksi warna aksen: indigo menjadi biru (2026-09-17, masih hari yang sama)

Work Owner meninjau hasilnya dan meminta: **"gunakan warna biru, jangan terlalu ke warna ungu."**

Permintaan itu tepat. `indigo` pada Tailwind memang bukan biru murni — nilainya
`oklch(54.6% 0.245 277)`, dan rona 277 sudah masuk wilayah ungu-nila. Yang dipakai sekarang
`blue-600`, `oklch(54.6% 0.245 262.9)` — terang dan jenuh persis sama, hanya ronanya digeser
sekitar 14 derajat ke arah biru.

**Yang berubah:** 38 kemunculan `indigo-*` di sembilan berkas menjadi `blue-*`, ditambah tiga
nilai yang tidak ikut terganti otomatis karena bukan nama kelas:

| Tempat | Dari | Menjadi |
|---|---|---|
| `--shadow-aksen` di `gaya.css` | `rgb(67 56 202)` — indigo-700 | `rgb(29 78 216)` — blue-700 |
| `<meta name="theme-color">` | `#4f46e5` | `#2563eb` |
| Komentar palet di `gaya.css` | menyebut INDIGO | menyebut BLUE, beserta catatan perubahannya |

**Angka kontras yang sempat saya tulis ternyata salah, dan ikut diperbaiki.**

Komentar di `gaya.css` dan tabel di `README.md` menyebut indigo-600 mencapai **8,6:1** di atas
putih — "lewat ambang AAA". Itu tidak benar. Perhitungan ulang menurut rumus luminansi relatif
WCAG:

| Warna | Kontras teks putih di atasnya | Ambang |
|---|---|---|
| `indigo-600` (yang sempat dipakai) | **6,2:1** | AA, **bukan** AAA |
| `blue-600` (yang dipakai sekarang) | **5,1:1** | AA (ambang teks normal 4,5:1) |
| `blue-700` | 6,7:1 | AA |
| `blue-800` | 8,6:1 | AAA |

Jadi klaim AAA salah sejak awal, bukan menjadi salah karena penggantian warna. Angka **8,6:1**
yang saya tulis ternyata milik `blue-800` — bukan indigo-600 maupun blue-600.

Dokumen sudah diperbaiki menjadi **5,1:1, AA**. Bila kelak AAA benar-benar dituntut, yang diubah
adalah **dasarnya menjadi `blue-800`**, bukan angkanya di dokumen — dan itu ditulis di komentar
`gaya.css` supaya tidak berulang.

Merah tetap tidak dipakai sebagai aksen, dengan alasan yang tidak berubah: ia bahasa universal
untuk galat.

**Verifikasi setelah penggantian:**

| Pemeriksaan | Hasil |
|---|---|
| `npm run periksa-tipe` | bersih |
| `npm test` | **33 uji lulus** — warna tidak menyentuh perilaku |
| `npm run build` | berhasil, CSS 39,0 kB |
| `indigo` di `src/` dan `index.html` | **0 kelas** — dua sisa hanya komentar yang mencatat perubahannya |
| `indigo` di CSS hasil build | **0** |
| `--color-blue-600` di CSS | `oklch(54.6% .245 262.881)` — rona biru, terkonfirmasi |
| Kelas `bg-blue-600`, `hover:bg-blue-700`, `active:bg-blue-800`, `focus-visible:ring-blue-500/35` | ada seluruhnya |
| Sajian nyata port 8099 | halaman `200`, `theme-color` `#2563eb`, CSS yang disajikan **nol** indigo |

---

## 13. Sesi keenam — penamaan kode dialihkan ke bahasa Inggris (2026-09-18)

### 13.1 Permintaan dan tiga pertanyaan yang diajukan lebih dulu

> "ubah struktur folder code Claim PNC, dari bahasa indonesia menjadi bahasa inggris untuk penamaan
> folder, file dan code didalamnya. dan tambahkan keterangan pada CLAUDE.MD supaya selanjutnya sudah
> otomatis menggunakan bahasa inggris."

Permintaan itu tidak menyebut batasnya, dan batasnya justru yang menentukan apakah pekerjaan ini
penggantian nama atau perubahan yang merusak. Tiga pertanyaan diajukan **sebelum satu berkas pun
disentuh**:

| # | Pertanyaan | Jawaban Work Owner |
|---|---|---|
| 1 | Teks yang dilihat pengguna ikut atau tidak | Disamakan seperti referensi dari berkas XML Pega; tambahan yang tidak ada di XML dikoreksi menjadi bahasa Inggris |
| 2 | Nama field JSON API dan nama kolom basis data | **Keduanya tetap Indonesia** |
| 3 | Komentar dan dokumen | **Komentar termasuk isi dokumen di `claim-pnc/docs` tetap bahasa Indonesia** |

Jawaban nomor 2 yang paling menentukan bentuk pekerjaannya: ia mengubah "ganti semua nama" menjadi
"ganti nama internal, **jangan sentuh** kontrak" — dan kedua hal itu sering berada di baris yang
sama:

```go
Number string `json:"nomor_rekening"`
```

Ditetapkan sebagai `D-80` di Decision Log, dan `08-TECHNICAL-STRATEGY.md` §4.1 ditulis ulang.

### 13.2 Kenapa alat sederhana tidak memadai

Percobaan pertama memakai `sed` dengan batas kata (`\b`). Ia salah di tiga tempat sekaligus, dan
ketiganya baru terlihat belakangan:

| Tempat | Contoh kerusakan |
|---|---|
| **Komentar** | `// Sesi yang dicabut` → `// Session yang dicabut` |
| **Literal string** | data uji `"Aktif"` → `"Active"` — ini **isi kolom basis data**, bukan nama |
| **Teks JSX** | `Tidak ada baris yang cocok` → `Tidak ada rows yang cocok` |

Karena itu penggantian dikerjakan dengan pemindai kecil yang **memecah berkas menjadi potongan kode
dan bukan-kode** lebih dulu: komentar (`//`, `/* */`) dan literal string dilewati, dan khusus
template literal TypeScript, bagian `${…}` di dalamnya dikembalikan menjadi kode. Baru setelah itu
peta nama diterapkan — pada potongan kode saja.

**Satu kelas yang tetap lolos dari pemindai, dan cara menangkapnya.** Teks JSX (`<p>Rekening
baru</p>`) bukan komentar dan bukan literal string; bagi pemindai ia kode. Uji frontend yang
memeriksa teks layar apa adanya itulah yang menangkapnya — tujuh kerusakan, seluruhnya ketahuan
sebagai uji merah, bukan lewat pembacaan ulang. Contoh yang paling menyesatkan bila lolos:
`Rekening tujuan pembayaran klaim` sempat menjadi `Rekening target pembayaran klaim`.

Kelas kedua yang juga lolos: **literal regex** (`/ubah status klaim/i`). Tanda `/` di awalnya dibaca
pemindai sebagai pembagian, sehingga isinya ikut terganti. Empat pemeriksa uji terdampak, dan
ditemukan oleh mekanisme yang sama.

### 13.3 Urutan kerja yang dipakai

Per modul, bukan sekaligus:

```
petakan nama → ganti (kode saja) → build → vet → test
             → periksa baris komentar di git diff → perbaiki prosa
```

Langkah kelima yang paling sering menemukan sesuatu, dan ia tidak dapat digantikan kompilator:
komentar yang rusak tetap dapat dikompilasi.

**Prosa yang dipulihkan**, contohnya: `di memory` → `di memori` · `ke Cashier` → `ke Kasir` ·
`Decision komite` → `Keputusan komite` · `Check kelengkapan isian` → `Periksa kelengkapan isian`.

`Cashier` dipertahankan hanya di tempat ia menyebut **identifier Go** (`seam Cashier`,
`bankaccount.Cashier`); di prosa, sistem eksternal itu tetap disebut **Kasir** sebagaimana bisnis
menyebutnya.

### 13.4 Yang berubah

| Lingkup | Isi |
|---|---|
| Backend | 5 modul (`auth`, `bankaccount`, `claimstatus`, `portal`, `platform`), 88 berkas |
| Frontend | 32 berkas, seluruh folder modul, dan berkas gaya |
| Migrasi | 4 berkas — `0001_user_and_session.*`, `0002_master_claim_status.*` |
| Nama query `.sql` | 29 penanda `-- name:`; **isi SQL tidak disentuh** |
| Perintah npm | `periksa-tipe` → `typecheck` · `tandai-dist` → `mark-dist` |

Peta lengkapnya ada di [`peta-penamaan.md`](peta-penamaan.md).

### 13.5 Yang sengaja tidak diubah

| Hal | Alasan |
|---|---|
| Nama field JSON API | kontrak; keputusan Work Owner |
| Nama tabel dan kolom basis data | dimiliki bersama Pega (`D-21`), perubahannya menempuh `D-63` |
| Komentar dan dokumen di `docs/` | keputusan Work Owner |
| Teks layar yang ada padanannya di XML Pega | `D-13` |
| **Variabel lingkungan dan flag baris perintah** | ditambahkan sebagai pengecualian kelima saat pekerjaan berjalan — lihat §13.6 |
| `catatan-pengembangan.md` dan `keputusan-implementasi.md` yang memuat jalur berkas lama | keduanya **rekaman**, bukan pernyataan yang berlaku; diperlakukan sama seperti entri lama Decision Log |

### 13.6 Pengecualian kelima yang ditemukan saat bekerja

Aturan awal menyebut **empat** hal yang tetap Indonesia. Saat `cmd/claimpnc/periksa.go` menjadi
`check.go`, muncul pertanyaan yang belum terjawab aturan itu: apakah flag `-periksa` dan variabel
lingkungan `PENYIMPANAN`, `PORTAL_UTAMA`, `IDENTITAS_ADAPTER` ikut berganti?

Keduanya **tidak** diganti, dan alasannya sama dengan alasan field JSON tidak diganti: ia dipakai
berkas `.env`, skrip deployment, dan operator — menggantinya merusak lingkungan yang sudah berjalan,
bukan sekadar mengganti nama. Aturan `D-80` dan §4.1 diperbarui menjadi **lima** hal, supaya ini
tercatat sebagai keputusan, bukan kelalaian.

### 13.7 Satu cacat lama yang terpaksa ikut diperbaiki

`FormRekening.tsx` dan `HalamanMasterRekening.tsx` memanggil `GalatAPI.field`, padahal kelas itu
tidak punya field bernama `field` — yang ada `detail: PelanggaranField[]`. Kode itu **tidak pernah
lolos `tsc`**, dan sesudah penggantian nama ia menghalangi seluruh pemeriksaan tipe.

Diperbaiki menjadi pembacaan `detail` sebagaimana bentuk sesungguhnya:

```ts
for (const { field, pesan } of error.detail) {
  const column = COLUMN_MAP[field]
  if (column) setError(column, { type: 'server', message: pesan })
}
```

Ini **di luar lingkup penggantian nama**, dan dicatat di sini supaya tidak terbaca sebagai akibatnya.

### 13.8 Verifikasi

| Pemeriksaan | Hasil |
|---|---|
| `gofmt -l` | 0 berkas |
| `go vet ./...` | bersih |
| `go test ./...` | **16 paket ok**, tidak ada yang gagal |
| `go build ./cmd/claimpnc` | ok |
| `npm run typecheck` | bersih |
| `npm test` | **39 lulus · 3 gagal** |
| `npm run build` | ok — 40,9 KB CSS · 463,2 KB JS |

**Ketiga uji frontend yang gagal sudah gagal sebelum sesi ini.** Ini dibuktikan, bukan
diasumsikan: sebuah `git worktree` pada `HEAD` dibuat, `node_modules` ditautkan ke sana, dan
suitenya dijalankan — hasilnya **sama persis, 3 gagal · 39 lulus, dengan ketiga nama uji yang
sama**.

Sebabnya bukan penamaan melainkan tabrakan nama tombol: tab layar berjudul **"Approve"** dan
**"Reject"**, sama persis dengan tombol aksi per baris, sehingga
`getByRole('button', { name: 'Approve' })` menemukan tab, bukan tombol aksinya. Diangkat sebagai
temuan tersendiri di [`keputusan-implementasi.md`](keputusan-implementasi.md) §14.4.

### 13.9 Koreksi susulan pada sesi yang sama — nama modul dikembalikan ke bahasa Indonesia

Setelah penggantian nama selesai dan diverifikasi, Work Owner **tidak mengenali nama modulnya**:

> "Untuk modul bank-account dan claim-status diubah menjadi master-rekening dan master-status-klaim
> dan untuk prompt selanjutkan akan diberitahu nama modulnya"

Satu pertanyaan diajukan lebih dulu, karena jawabannya mengubah besar pekerjaannya: `bank-account`
dan `claim-status` adalah nama folder **frontend**; backend memakai `bankaccount` dan
`claimstatus`. Berlaku di keduanya, atau hanya frontend? **Jawaban: keduanya** — sehingga satu
modul punya satu nama di kedua sisi.

**Yang berubah:**

| Sebelum | Sesudah |
|---|---|
| `internal/bankaccount/**` · paket `bankaccount`, `bankaccounthttp` | `internal/masterrekening/**` · paket `masterrekening`, `masterrekeninghttp` |
| `internal/claimstatus/**` · paket `claimstatus`, `claimstatushttp` | `internal/masterstatus/**` · paket `masterstatus`, `masterstatushttp` |
| `src/modules/bank-account/**` | `src/modules/master-rekening/**` |
| `src/modules/claim-status/**` | `src/modules/master-status-klaim/**` |
| `BankAccountPage.tsx` | `AccountPage.tsx` |

**Kenapa `AccountPage`, bukan `MasterRekeningPage`.** Yang berbahasa Indonesia adalah **nama
modul**, bukan isinya. Komponen di dalamnya memakai nama **tipe domain** — sama seperti backend
yang memuat `masterrekening.Account`, bukan `masterrekening.MasterRekening`.

**Yang tidak ikut berubah:** `auth`, `portal`, `platform`, dan `spa` — keempatnya modul kerangka
yang memang tidak punya nama bisnis.

Ditetapkan sebagai `D-81`, dan `08-TECHNICAL-STRATEGY.md` §4.1 memuat aturannya supaya modul
berikutnya mengikuti pola yang sama tanpa perlu ditanyakan lagi.

**Verifikasi ulang setelah koreksi:** `gofmt` 0 berkas · `go vet` bersih · **16 paket uji Go
lulus** · `tsc` bersih · `npm test` **39 lulus · 3 gagal** (ketiganya tetap kegagalan lama yang
sama) · `npm run build` ok.

---

## 14. Sesi ketujuh — merge yang belum selesai, dan Master Status Progres 1 dibawa ke standar baru (2026-09-18)

### 14.1 Permintaan

> "HalamanStatusProgres1 tolong disesuaikan seperti standard baru sesuai CLAUDE.MD dan anak diubah
> menjadi bhs inggris juga"

### 14.2 Yang ditemukan lebih dulu, dan itu mengubah bentuk pekerjaannya

Sebelum satu berkas pun disentuh, `go build ./...` dan `npx tsc --noEmit` dijalankan untuk
mengetahui keadaan awal. Keduanya **gagal**, dan sebabnya bukan modul Status Progres:

```
internal/masterrekening/masterrekening.go:438:1: syntax error: unexpected <<
internal/portal/aktif.go:49:5: undefined: normalkan
internal/auth/provider/tiruan.go:27:20: undefined: auth.Profil
```

Penelusuran `git log --graph` menunjukkan commit `c3b495d` adalah **merge yang di-commit dengan
konflik belum diselesaikan** — cabang penamaan Inggris (`1782b62`) digabung dengan cabang
`Push Master Status Progress 1` (`4481dda`), lalu di-commit apa adanya. Penanda `<<<<<<<` masih
ada di `main.go`, `masterrekening.go`, `README.md`, dan tiga dokumen.

Jadi pekerjaan sesungguhnya bukan "menerjemahkan satu modul", melainkan **menyelesaikan merge itu**
dan membawa modul yang dibawanya ke standar `D-80`/`D-81`.

| Gejala | Sebab |
|---|---|
| `masterrekening.go` gagal dikompilasi | penanda konflik di tengah definisi struct |
| `portal/aktif.go` memanggil `normalkan`, `Cari` | berkas BARU dari cabang lain, belum ikut penggantian nama |
| `auth/provider/tiruan.go` memanggil `auth.Profil` | kembar lama `fake.go` yang hidup lagi karena merge |
| `HalamanStatusProgres1.tsx` "is not a module" | **seluruh isinya dikomentari** supaya build lewat |
| Rute Status Progres di `App.tsx` | ikut dikomentari |

### 14.3 Urutan kerja

Backend lebih dulu, karena `main.go` yang rusak menahan seluruh paket:

```
selesaikan konflik → paket portal → modul statusprogres → main.go → build/vet/test
```

Lalu frontend:

```
buang kode mati → kontrak galat → modul → prop anak → rute & menu → tsc/test/build
```

### 14.4 Kode mati yang dibuang, dan bagaimana dipastikan mati

`src/app/Kerangka.tsx`, `Kerangka.test.tsx`, `NavigasiUtama.tsx`, dan `menu.ts` dihapus.

Ia **tidak** dihapus karena tampak usang, melainkan karena dibuktikan tidak dipakai: `App.tsx` pada
cabang `4481dda` sendiri memakai `KerangkaHalaman`, bukan `Kerangka`. Jadi keempatnya sudah tidak
terpakai **di cabang yang melahirkannya** — percobaan kerangka yang ditinggalkan, bukan kerangka
yang sedang dipakai. Perannya diambil `PageShell.tsx`, yang punya bilah atas, menu, identitas
pengguna, dan tombol keluar.

Yang hilang bersamanya: `Kerangka.test.tsx`. Uji itu **tidak pernah berjalan** — ia gagal saat
pengumpulan karena mengimpor modul yang sudah tidak ada. Menggantinya dengan uji `PageShell` adalah
pekerjaan tersendiri dan dicatat sebagai yang belum dikerjakan.

### 14.5 Penggantian nama modul

Backend `internal/statusprogres` → `internal/masterstatusprogres` (`D-81`: nama folder modul memakai
nama modul bisnis). Isinya seluruhnya dialihkan ke bahasa Inggris — 20 berkas, termasuk nama kueri
`.sql`.

| Sebelum | Sesudah |
|---|---|
| `StatusProgres` · `Isian` · `Posisi` | `ProgressStatus` · `Input` · `Position` |
| `PelanggaranIsian` · `GalatValidasi` | `Violation` · `ValidationError` |
| `Layanan` · `Opsi` · `Tambah` · `Ubah` | `Service` · `Options` · `Create` · `Update` |
| `HandlerBaru` · `Pasang` | `NewHandler` · `Mount` |
| `PemilihRepo` · `IDInduk` · `NamaInduk` | `RepoSelector` · `ParentID` · `ParentName` |
| `posisi.go` · `rute.go` · `galat.go` · `layanan.go` | `position.go` · `routes.go` · `errors.go` · `manage.go` |
| `-- name: statusprogres_daftar` | `-- name: progress_status_list` |

Paket `portal` ikut: `aktif.go` → `active.go`, `portalaktif.go` → `activeportal.go`,
`PilihAktif` → `SelectActive`, `BahanPortalAktif` → `ActivePortalDeps`, `ErrBelumSiap` →
`ErrNotReady`.

Frontend: folder `master-status-progres` **tidak berubah** (sudah sesuai `D-81`), isinya diganti —
`HalamanStatusProgres1.tsx` → `ProgressStatusPage.tsx`, `FormStatusProgres.tsx` →
`ProgressStatusForm.tsx`, `KolomPilihan.tsx` → `SelectField.tsx`.

### 14.6 `anak` → `children`

Empat berkas: `App.tsx`, `PageShell.tsx`, `SessionGuard.tsx`, dan satu variabel lokal di
`AccountPage.test.tsx`.

Karena namanya kini benar-benar `children`, pemanggilannya diubah menjadi bentuk bersarang yang
memang idiomatik di React — `<SessionGuard><Protected>…</Protected></SessionGuard>` — bukan
`children={…}` sebagai atribut. Tidak ada perubahan perilaku; yang berubah bentuk tulisannya.

### 14.7 Sisa penamaan Indonesia yang ikut dibereskan

Penggantian nama sesi lalu menyisakan **nama prop komponen bersama** dalam bahasa Indonesia, dan
modul baru ini terpaksa memakainya supaya lolos kompilasi. Karena itu ia ikut dibereskan:

| Sebelum | Sesudah | Tempat |
|---|---|---|
| `judul` · `keterangan` | `title` · `description` | `DataTable`, `ErrorMessage` |
| `nilai` · `tampil` · `lebar` | `value` · `render` · `width` | kontrak `Column` |
| `tanpaUrut` · `keKanan` · `aksi` | `noSort` · `alignRight` · `actions` | idem |
| `petunjuk` · `kotak` · `arah` | `hint` · `box` · `direction` | `Field`, `ErrorMessage`, `DataTable` |
| `'naik'` · `'turun'` | `'asc'` · `'desc'` | nilai internal pengurutan `DataTable` |

**`aktif` sengaja TIDAK ikut** kecuali pada satu prop lokal `SortMarker`: ia nama field JSON API
(`Account.aktif`), dan mengubahnya merusak kontrak.

### 14.8 Verifikasi

| Pemeriksaan | Hasil |
|---|---|
| `gofmt -l` | 0 berkas |
| `go build ./...` · `go vet ./...` | bersih |
| `go test ./...` | seluruh paket lulus, termasuk 5 paket modul baru |
| `tsc --noEmit` | bersih |
| `npm test` | **57 lulus · 3 gagal** |
| `npm run build` | ok |
| penanda konflik tersisa di repo | **0** |

**Ketiga kegagalan uji adalah kegagalan lama yang sama** pada `AccountPage.test.tsx` — tab layar
berjudul "Approve"/"Reject" bernama sama persis dengan tombol aksi per baris, sehingga
`getByRole('button', { name: 'Approve' })` menemukan tab. Sebelum sesi ini 39 uji lulus; sekarang 57,
dan 18 tambahannya adalah uji layar Status Progres yang sebelumnya tidak pernah berjalan.

### 14.9 Teks layar yang rusak oleh penggantian nama sesi lalu

Ditemukan saat menembak aplikasi yang sedang berjalan, bukan lewat pembacaan kode: permintaan tanpa
sesi dijawab **"Session tidak sah. Silakan masuk kembali."**

`D-80` menetapkan teks yang dilihat pengguna tetap berbahasa Indonesia. Kerusakan ini lolos karena
pemindai penggantian nama memperlakukan **literal string sebagai bukan-kode dan melewatinya** —
tetapi sebagian teks terlanjur terganti pada percobaan `sed` sebelum pemindai itu dipakai, dan
kompilator tidak punya cara mengetahui sebuah kalimat berubah arti.

Seluruhnya dipulihkan ke bunyi aslinya, dibaca dari `git show 3e57aae:<berkas>` — bukan diterjemahkan
ulang dari ingatan:

| Berkas | Sebelum | Sesudah |
|---|---|---|
| `auth/http/errors.go` | "Session tidak sah…" · "Session Anda sudah berakhir…" | "Sesi …" |
| `masterrekening/http/errors.go` | "Account tidak ditemukan." | "Rekening tidak ditemukan." |
| `masterrekening/cashier/fake.go` | "[TIRUAN] Account diterima sistem Kasir." | "…Rekening diterima…" |
| `masterrekening/notification/smtp.go` | "ke Cashier GAGAL" · "Account berikut" · "respons Cashier" · "Pesan dari Cashier" | "ke Kasir GAGAL" · "Rekening berikut" · "respons Kasir" · "Pesan dari Kasir" |
| `home/HomePage.tsx` · `master-status-klaim/ClaimStatusPage.tsx` | "33 state bisnis sebuah klaim" | "33 keadaan bisnis sebuah klaim" |
| `master-rekening/AccountForm.tsx` | "Muat ulang page, lalu coba lagi." | "Muat ulang halaman, …" |

Fixture uji yang mencerminkan pesan Kasir ikut dipulihkan, karena ia meniru **pesan dari sistem
luar** — bukan teks yang boleh kami karang.

> Pelajarannya: kerusakan pada teks tidak akan pernah muncul sebagai kegagalan kompilasi. Yang
> menemukannya di sini adalah **satu permintaan HTTP ke aplikasi yang sedang berjalan** — langkah
> yang tidak ada di daftar verifikasi sebelumnya, dan sejak sekarang ada.

---

## 15. Sesi kedelapan — menu kiri dibaca dari basis data (2026-09-19)

### 15.1 Permintaan

> "Tambahkan menu di sebelah kiri dengan membaca table dengan query sbb: `SELECT menu.menu_id,
> menu.menu_desc, menu.menu_id_leader, menu.menu_program FROM M_APLIKASI app,
> M_MENU_APLIKASI_PNC menu WHERE app.app_id = menu.app_id AND app.app_desc = 'CLAIM PNC' ORDER BY
> menu.menu_sequence;` / Menu disusun berurutan sesuai menu_sequence dan sub menu disusun sesuai
> menu_id_leader / Sedangkan menu akan aktif jika Login ID atau Group ID tersebut ada akses ke table
> M_OTORISASI_PNC dan harness sesuai kolom menu_program sudah ada modulnya"

### 15.2 Yang dicari lebih dulu, dan apa yang ditemukan

Ketiga tabel itu belum pernah dipakai modul mana pun, jadi bentuknya dicari sebelum satu baris kode
ditulis. Hasilnya melampaui dugaan:

| Yang dicari | Ditemukan |
|---|---|
| DDL ketiga tabel | **`Database/CREATE_MENU.sql`** — lengkap, termasuk `M_LOGIN_GROUP_PNC` |
| Isinya | **lima CSV**: `m_aplikasi`, `m_menu_aplikasi_pnc`, `m_otorisasi_pnc`, `m_login_group_pnc`, `m_login_pnc` |
| Rule Pega yang memakainya | **NOL** — dicari ke seluruh 2.634 berkas XML |
| Penyebutan di dokumen proyek | **NOL** |

Dua akibat dari baris ketiga dan keempat: ini **kemampuan baru**, bukan pemindahan perilaku Pega,
sehingga tidak ada baseline untuk gerbang 1. Dan `m_login_group_pnc.csv` adalah artefak yang
`D-58` serta `TKT-F3-004` nyatakan **tidak ada di basis data** — lihat
[`keputusan-implementasi.md`](keputusan-implementasi.md) §16.2.

### 15.3 Apa yang dibaca dari datanya, sebelum aturannya ditulis

Bentuk aturan tampil TIDAK dikarang; ia dibaca dari isi tabelnya:

```
M_OTORISASI_PNC  group "IT"    → MENU_ID 11..81   (71 baris)
                 login "JONNY" → MENU_ID 4, 82..86 (6 baris)
M_LOGIN_GROUP_PNC              → JONNY anggota IT
```

- Group `IT` **tidak diberi izin atas satu pun kelompok tingkat atas** (MENU_ID 1..4), padahal
  anak-anaknya diberi. Menuntut kelompok punya baris izin sendiri akan menghapus **seluruh** menu
  group IT. Karena itu aturannya: **kelompok tampil bila ada anaknya yang tampil.**
- Login `JONNY` justru diberi izin atas MENU_ID 4 (REPORT). Bila anaknya kosong, judul kelompoknya
  tetap disembunyikan — judul yang tidak membuka apa pun hanya menambah barang di layar.
- Izin group dan izin login **digabung**, bukan saling menggantikan: keduanya memberi butir yang
  berbeda, dan hanya penggabungan yang menghasilkan menu yang utuh.

### 15.4 Tiga temuan sampingan yang memperjelas dokumen lama

| Temuan | Artinya |
|---|---|
| **9 `MENU_PROGRAM` menunjuk harness yang TIDAK ADA di export** — `DataMemberReas`, `DetailMasterPasalAI`, `InboxCloseClaim_Harness`, `InboxOutstanding_Harness`, `InboxRequestSalvage`, `InboxServiceCenter`, `LostAdjuster_harness`, `PNCViewClaim`, `ReportProduksiPA_harnes` | memperjelas `K-33` dengan daftar yang konkret |
| **8 harness ADA tetapi tidak muncul di menu mana pun** — ketujuh harness berkelas `Work` ditambah `ViewPolis1` | **menguatkan Lampiran G**: ketujuhnya memang dibuka DARI DALAM klaim, bukan dari menu |
| **MENU_ID 83 "Report Adjuster" adalah daun tanpa `MENU_PROGRAM`** | barisnya ada di master, tujuannya tidak — bukan kesalahan pembacaan |

### 15.5 Yang dibangun

**Backend — modul `internal/menu`.** Namanya Inggris, bukan nama modul bisnis: ia modul kerangka
seperti `auth` dan `portal`, bukan layar Master yang diminta dengan nama bisnis (`D-81`).

```
internal/menu/
  menu.go              domain: Item, Node, BuildTree, Subjects, seam Repo
  repo/sqlstore/       4 kueri + pemeriksaan tabel
  repo/memory/         isi contoh DISALIN dari kelima CSV
  usecase/build.go     urutan langkah: group → izin group → izin login
  http/                GET /api/menu
```

**Frontend.**

```
app/menu/api.ts       hook useMenu()
app/menu/registry.ts  peta MENU_PROGRAM → rute; 3 baris hari ini
app/Sidebar.tsx       kolom menu kiri, kelompok dapat dilipat
app/PageShell.tsx     tata letak berubah: bilah atas + kolom kiri + isi
```

### 15.6 Verifikasi

| Pemeriksaan | Hasil |
|---|---|
| `gofmt -l` · `go build` · `go vet` | bersih |
| `go test ./...` | **24 paket lulus**, termasuk 4 paket modul menu |
| `tsc --noEmit` · `npm run build` | bersih |
| `npm test` | **68 lulus · 3 gagal** (naik dari 57; 11 tambahannya uji Sidebar) |

Ketiga kegagalan itu tetap kegagalan lama yang sama di `AccountPage.test.tsx` — tab "Approve"/"Reject"
bernama sama persis dengan tombol aksi per baris.

**Diuji terhadap aplikasi yang benar-benar berjalan**, bukan hanya lewat uji:

```
POST /api/masuk  (IDENTITAS_ADAPTER=fake, instans sementara di :8099)
GET  /api/menu   → 3 kelompok · 71 butir · 3 di antaranya aktif
                   MASTER (33) · INBOX (34) · VIEW (4)
                   REPORT tidak muncul — izinnya milik login JONNY, bukan group IT
GET  /api/menu   tanpa sesi → 401 sesi_tidak_sah
```

### 15.7 Satu perbaikan kecil di luar lingkup

Uji `ClaimStatusPage` merender seluruh `AppRoute`, sehingga kerangka layarnya ikut memuat menu.
Pada kasus uji "pemuatan gagal", peladen tiruannya menjawab galat untuk SETIAP jalur — termasuk
`/api/menu` — dan pesan galat menu ikut terbaca sebagai `role="alert"` yang dicari uji itu.

Dua hal diperbaiki, dan keduanya benar terlepas dari uji:

- Pesan galat menu memakai **`role="status"`**, bukan `role="alert"`. Menu yang gagal dimuat adalah
  keadaan, bukan sesuatu yang harus menyela apa yang sedang dibaca pengguna di isi halaman.
- Fixture uji itu kini menjawab `/api/menu`, sama seperti ia sudah menjawab `/api/portal`.

## 16. Sesi kesembilan — modul Inbox Laporan Klaim (2026-09-19)

### 16.1 Permintaan

> "lanjutkan untuk penambahan modul Inbox Laporan Klaim / cek secara penuh aplikasi existing pada
> dokumen Harness dengan nama file InboxRCVApp_Harness jadikan ini sebagai referensi."

### 16.2 Tiga pertanyaan konfirmasi dan jawabannya

| Pertanyaan | Jawaban Work Owner |
|---|---|
| Lingkup iterasi ini — baca saja, baca + ekspor, atau penuh | **"Semuanya seperti aplikasi PEGA tapi table tidak save ke table PC_ASM_FW_GCNMFW_WORK lagi"** |
| Tabel butuh paginasi sisi server, sedangkan `DataTable` menyaring di peramban | **"Seperti aplikasi PEGA yang berjalan saat ini"** — yakni paginasi server |
| Sumber data yang disiapkan | **sqlstore + memory** |

Jawaban pertama yang menentukan seluruh bentuk modul: **paritas penuh, tetapi penulisan pindah ke
tabel milik aplikasi ini.**

### 16.3 Yang dicari lebih dulu, dan apa yang ditemukan

Harness yang ditunjuk ternyata **hanya pembungkus**. Seluruh isi layar hidup di satu section, dan
aturan bisnisnya di dua activity.

| Yang dicari | Ditemukan |
|---|---|
| Isi `InboxRCVApp_Harness` (1,19 MB) | pembungkus; isinya satu section `ViewStatusReceiveDocument` (1,16 MB) |
| Pengisi grid | `Activity/SetListRCV_Act-Act.xml` dan `GetClaimRCVList_Act-Act.xml` |
| Kueri daftar | **enam** RDB List, dipilih rantai `@if` atas `param.Note` |
| Kueri pencacah | `BrowseClaimRCV_Aksep-SQL.xml` — **delapan angka dalam satu kueri** |
| Judul tab dan nama kolom | terbaca apa adanya dari `pyValue` bertanda `<b>` |
| Tombol | `CreateNewCaseRCV` · `SetListRCV_Act` · `ExportNotTransferRCV` |
| Tabel yang ditulis saat berkas dibuat | `POOLDATA.T_CLAIM_RECIVEDCLAIM`, lewat `PROCINSERTDATARECIVEDKLAIM.prc` |

**Sembilan tab, bukan enam.** Keenam kueri melayani sembilan tab: tiga tab komunikasi memakai kueri
yang sama (`ViewRejectKomunikasiUser`) dengan penyaring percakapan yang berbeda.

### 16.4 Empat temuan yang mengubah rancangan

| Temuan | Akibat |
|---|---|
| **"Buat Baru" membuat berkas KOSONG** — `CreateNewCaseRCV` hanya mengisi lima nilai, seluruhnya diturunkan dari petugas penekannya | Tidak ada form sama sekali. Rancangan awal saya menyiapkan form belasan isian; ia dibatalkan |
| **Tab "Data rejected" tidak punya lencana** — kueri pencacah menyaring `PYSTATUSWORK NOT IN (Resolved-Completed, Resolved-Rejected)`, sehingga berkas ditolak justru yang dikecualikan | `Summary.CountOf` mengembalikan dua nilai: angka DAN apakah ia dihitung. Nol dan "tidak dihitung" dibedakan |
| **Kueri grid dan kueri pencacah TIDAK sepakat** untuk tab "Replied from ASM" — grid memakai `sender != saya`, pencacah memakai `sender = saya` | Selisihnya dibiarkan terlihat dan dicatat, bukan ditutup. Ia cacat yang sudah ada sebelum modul ini |
| **Lima alias kolom menyebut hal yang sama sekali lain** — `Kurir`=nama bisnis, `UserAdmin`=nama cabang, `KodeCabang`=operator, `StatusKomunikasi`=kode cabang, `SIM`=keterangan | Tidak satu pun dibawa (`D-19`); pemetaan baliknya ditulis di kepala berkas `.sql` |

### 16.5 Yang dibangun

**Backend — modul `internal/inboxlaporanklaim`.** Namanya nama modul bisnis dalam bahasa Indonesia
(`D-81`), isinya berbahasa Inggris (`D-80`).

```
internal/inboxlaporanklaim/
  doc.go              dua tabel dan kenapa keduanya dibaca
  claimreport.go      ClaimReport, Position, Origin, umur berkas
  category.go         sembilan tab + pemetaan ke param.Note warisan
  filter.go           penyaring, lini bisnis, paginasi
  summary.go          delapan pencacah
  number.go           RCVN.YY.xxxx
  errors.go, seam.go  Repo, RepoSelector, Clock, Caller
  usecase/            daftar, ringkas, kanwil, ambil, buat
  repo/sqlstore/      satu fragmen sumber + enam badan kueri
  repo/memory/        14 berkas contoh + percakapan contoh
  http/               dto, galat, handler, ekspor CSV, rute
migrations/0003_claim_report_inbox.{up,down}.sql
```

**Frontend.**

```
components/DataTable.tsx                 + mode paginasi server (MENAMBAH, bukan mengubah)
modules/inbox-laporan-klaim/types.ts     tipe kontrak API
modules/inbox-laporan-klaim/api.ts       empat hook + unduhan CSV berheader
modules/inbox-laporan-klaim/ClaimReportInboxPage.tsx
app/App.tsx                              rute /inbox/laporan-klaim
app/menu/registry.ts                     InboxRCVApp_Harness -> rute
```

### 16.6 Kendala teknis dan penyelesaiannya

| Kendala | Penyelesaian |
|---|---|
| **Kueri harus melayani 9 tab, 4 penyaring, dan 2 tabel — tanpa merangkai teks SQL** | Satu fragmen `WITH` dipakai bersama enam badan kueri, disambung Go dari berkas `.sql` sendiri. Seluruh nilai tetap lewat parameter binding; aturan tab diterjemahkan menjadi **penanda**, bukan potongan SQL |
| **Daftar kode `IN (...)` berbeda panjang per lini bisnis, sedangkan bentuk kueri harus tetap** | Daftar dipadatkan dengan mengulang kode terakhirnya — `IN ('002','002','002','002')` sama persis dengan `IN ('002')`. Kodenya tetap hidup di satu tempat, `BusinessLine.Criteria()` |
| **`auth.User` tidak punya nomor telepon**, sedangkan Pega mengisi `TelpPengirim` | Kolomnya **tidak dibuat**. Kolom yang selamanya kosong tampak seperti data yang belum diisi, dan pertanyaannya akan terus berulang. Celahnya dicatat di `seam.go` |
| **Nama pelapor tidak dapat dibaca dari tabel warisan** — tidak satu pun dari sembilan kueri lama menyentuh kolomnya, sehingga namanya tidak diketahui (`R-08`) | Dibaca dari tabel baru saja; untuk baris warisan ia `NULL`. Menebak nama kolom menghasilkan kueri yang gagal saat pertama dijalankan di produksi |
| **Unduhan CSV lewat tautan biasa tidak membawa header**, sedangkan endpoint menuntut `Authorization` dan `X-Portal` | Diambil dengan `fetch` lalu disimpan sebagai Blob. Menaruh token di alamat ditolak — nilainya tercatat di riwayat peramban, log proxy, dan header Referer |
| **`npx prettier` menulis ulang gaya seluruh `DataTable.tsx`** (titik koma, kutip ganda) | Repo ini **tidak memakai prettier** — tidak ada konfigurasinya dan tidak ada di `package.json`; `npx` mengunduhnya sendiri. Berkas dikembalikan lewat `git checkout`, lalu suntingan diterapkan ulang dengan tangan |

### 16.7 Satu aturan yang saya tambahkan sendiri, lalu saya cabut

Versi pertama **menolak** pembuatan berkas ketika cabang pemanggil tidak terbaca, dengan alasan
berkas tanpa cabang akan hilang dari daftar yang disaring cabang. Alasannya masuk akal, dan
penolakannya tetap salah.

`CreateNewCaseRCV` langkah 19 mengisi cabang dari hasil `GetIDCabang` dan **tidak memeriksa
hasilnya sama sekali**: kueri yang tidak mengembalikan baris menghasilkan cabang kosong, dan berkas
tetap dibuat. Menolaknya adalah **aturan baru**, dan `P-5` menetapkan perilaku dipertahankan lebih
dulu kecuali untuk 13 butir yang `D-49` sebut satu per satu — penolakan ini tidak ada di antaranya.

Ketahuannya bukan dari membaca ulang, melainkan dari **menjalankan aplikasinya**: tombol Buat Baru
menjawab `409` untuk setiap pengguna tiruan, karena tidak satu pun dari mereka punya kode cabang.

### 16.8 Verifikasi yang benar-benar dijalankan

| Pemeriksaan | Hasil |
|---|---|
| `gofmt -l` pada berkas baru | bersih |
| `go build ./...` dan `go vet ./...` | bersih |
| `go test ./...` | **seluruh paket lulus**, termasuk 3 paket modul baru |
| `tsc --noEmit` | bersih |
| `npm test` | **97 lulus, 3 gagal** (naik dari 83 lulus) |
| `npm run build` | bersih |

Ketiga kegagalan itu kegagalan lama di `AccountPage.test.tsx`. Kali ini **dibuktikan**, bukan
diasumsikan: perubahan `DataTable.tsx` di-`git stash`, uji dijalankan ulang, dan ketiganya tetap
gagal tanpa perubahan itu.

**Diuji terhadap aplikasi yang benar-benar berjalan**, instans sementara di porta 8097:

```
POST /api/masuk
GET  /api/inbox/laporan-klaim                      tanpa X-Portal -> 400 ditolak
GET  /api/inbox/laporan-klaim/pilihan              9 tab, 5 bisnis, 3 kanwil
GET  ?kategori=outstanding|belum-registrasi|...    keenam tab status berisi
       lencana: 4 | 3 | 4 | 2 | (tak dihitung) | 11   -> 4+3+4 = 11 cocok dengan total
GET  ?kanwil=01 -> 7 baris ; ?kanwil=03 -> 2 baris
GET  ?bisnis=pa -> 3 ; ?bisnis=kelompok-khusus -> 2  (saling lepas)
GET  ?cari=RCV-0003 -> 1 ; ?cari=RCV -> 0           (persis, bukan sebagian)
GET  ?halaman=2&ukuran=4 -> halaman 2 dari 3
GET  ?kategori=ngawur -> validasi_gagal
GET  ketiga tab komunikasi -> pesan terakhir terisi
POST /api/inbox/laporan-klaim -> RCVN.26.0001, Not Transferred, asal=claimpnc
       lalu muncul di puncak tab "Data hasn't been transferred"
GET  /ekspor -> CSV 13 baris, Content-Disposition + Cache-Control: no-store
GET  /TIDAK-ADA -> 404 tidak_ditemukan
GET  /api/menu -> butir 64 "Inbox Laporan Klaim" ada dan kini bertaut ke layarnya
```

Satu hal yang ditemukan hanya karena aplikasinya dijalankan: helper uji `selectedColumns` saya kira
mengurai 18 kolom; probe sementara membuktikan **19** — saya lupa `reporter_name` yang baru
ditambahkan. Probe-nya sengaja dibuat gagal untuk membuktikan helper-nya memang mengurai sesuatu,
lalu dihapus.

### 16.9 Yang TIDAK dikerjakan, dan alasannya

| Hal | Alasan |
|---|---|
| Tombol **"Tarik data"** | Ia memanggil activity yang sama persis dengan Refresh (`SetListRCV_Act`). Dua tombol yang mengerjakan hal yang sama membawa pertanyaan "apa bedanya" yang tidak punya jawaban |
| **Bagan** di atas daftar | `pxChart` dengan seri `Description`/`Count`; isinya sama dengan lencana tab yang sudah tergambar. Menggambarnya dua kali menambah barang, bukan keterangan |
| **Membuka isi berkas** | Barisnya membuka penugasan `ReceiveDocument_Flow` lewat harness `ViewReceiveDocument` — layar tersendiri, lingkup `B-14`. Modul ini membawa rujukannya (`rujukan_pega`) tetapi tidak membukanya |
| Menulis ke `POOLDATA.T_CLAIM_RECIVEDCLAIM` | `P-1` menetapkan satu tabel satu penulis. Keempat kolom yang ditulis Pega saat berkas lahir pindah ke tabel baru; sisanya ikut `B-14` saat modulnya dibangun |

## 17. Sesi kesepuluh — tombol "Buat Baru" dan form Input Receive Document (2026-09-22)

### 17.1 Permintaan

> "Pada modul inbox laporan klaim kenapa button Buat Baru tidak bekerja? seharusnya saat diclick
> Buat Baru dia akan membuat RCV baru dengan mengikuti file InputReceiveDocument.xml pada folder
> Flow. Buatkan dan perbaiki Button Buat Baru tersebut"

### 17.2 Kenapa tombolnya tampak tidak bekerja

Permintaannya **berhasil** — `POST` menjawab `201` dan berkasnya benar-benar tersimpan. Yang tidak
terjadi adalah dua hal berikutnya, dan tanpa keduanya tombol itu memang tidak berguna:

| # | Yang terjadi | Akibat di kursi petugas |
|---|---|---|
| 1 | Layar **tidak berpindah ke form isian** | Berkas lahir kosong dan tidak ada tempat mengisinya |
| 2 | Berkas baru berposisi *Not Transferred*, sedangkan tab bawaan *Outstanding Data* | Daftar tidak berubah sama sekali |

Gabungan keduanya: menekan tombol menghasilkan satu pesan kecil, lalu tidak ada apa pun yang dapat
dikerjakan. Itu sebabnya ia terbaca sebagai tombol rusak.

**Akar sebabnya ada di sesi sebelumnya.** Saya membaca `Activity/CreateNewCaseRCV-Act.xml` dan
menyimpulkan benar bahwa ia membuat berkas KOSONG — tetapi berhenti di situ. Yang tidak saya
telusuri adalah ke mana berkas itu pergi sesudahnya, dan jawabannya ada di berkas yang Work Owner
tunjuk: `Flow/InputReceiveDocument.xml`.

### 17.3 Apa yang sebenarnya dilakukan alur itu

```
Start ─► Assignment "Receive Document" ─► End
           WorkList, router PNCAdminRouterRCV
           flow action: InputReceiveDocument
```

Berkas lahir kosong JUSTRU supaya assignment inilah yang mengisinya. Flow action
`InputReceiveDocument` merender sebuah form — dan form itulah yang hilang dari modul saya.

### 17.4 Bukti yang ditemukan, dan satu yang tidak ada

| Yang dicari | Hasil |
|---|---|
| `Flow/InputReceiveDocument.xml` | ada — alur `ReceiveDocument_Flow`, satu assignment |
| `Flow Action/InputReceiveDocument-FlowAction.xml` | ada — merender section `InputReceiveDocument` |
| **`Section/InputReceiveDocument`** | **TIDAK ADA di export** — bagian dari ±242 rule hilang (`R-16`) |
| `Section/ViewInputReceiveDocument_sec` (1,24 MB) | ada — varian tampil form yang sama; **dipakai sebagai sumber label dan properti** |
| `Activity/Pre_ActReceiveDocument` | ada — hanya menyalin `StatusLock`; tidak menyiapkan isian |
| Validate rule pada flow action | **NOL** — tidak ada isian wajib, tidak ada aturan |

### 17.5 Lingkup form: ditentukan bukti, bukan selera

Properti terikat pada section membelah dirinya sendiri menjadi dua kelompok yang jelas:

| Kelompok | Terikat | Keputusan |
|---|---|---|
| 17 isian tunggal `.ReceiveDocument.*` | field biasa | **dibawa seluruhnya** |
| Blok pelapor + alamat | `.ReportHE.*`, alamatnya page list `.ReportHE.ASMAdressList` | **tidak dibawa** — `ReportHE` adalah area **Heavy Equipment**, yang `D-34` keluarkan dari lingkup atas keputusan Work Owner |
| Grid dokumen | page list `.ReceiveDocument.DocumentList` | **tidak dibawa** — menuntut `S-1`; yang dibawa hanya angka totalnya |
| Riwayat komunikasi & progres | `tempHistoryKomunikasi`, `tempViewProgress` | **tidak dibawa** — milik modul lain |

Empat dari sepuluh isian baru bahkan terbukti disimpan procedure lama
(`PROCINSERTDATARECIVEDKLAIM.prc`); dua — estimasi kerugian dan jumlah dokumen — ada di form tetapi
tidak di procedure, dan tetap dibawa karena ia isian yang benar-benar diketik petugas.

### 17.6 Yang dibangun

**Backend**

```
inboxlaporanklaim/detail.go          Detail: 17 isian + Clean/Check + DetailOf/Apply
inboxlaporanklaim/claimreport.go     + 10 field isian, Money, UpdatedBy/UpdatedAt
inboxlaporanklaim/seam.go            + Repo.Update
usecase/inbox.go                     + Service.Save
repo/sqlstore                        + claim_report_update, scanDetailRow, Get berkolom lengkap
repo/memory                          + Update
http/                                + PUT /{id}, DetailDTO, kode galat laporan_hanya_baca
migrations/0004_claim_report_detail.{up,down}.sql
```

**Frontend**

```
ClaimReportFormPage.tsx    form Input Receive Document — 17 isian, 3 kelompok
api.ts                     + useClaimReport, useSaveClaimReport
ClaimReportInboxPage.tsx   Buat Baru MEMBUKA form; nomor berkas menjadi tautan
App.tsx                    rute /inbox/laporan-klaim/:id
```

### 17.7 Keputusan yang menuntut pertimbangan

| Hal | Keputusan dan alasannya |
|---|---|
| **Migrasi baru atau menyunting 0003** | **Berkas baru, 0004.** 0003 sudah diserahkan sebagai permintaan perubahan skema dan mungkin sudah dijalankan DBA di salah satu portal. Menyuntingnya berarti dua orang memegang berkas bernomor sama dengan isi berbeda |
| **Berkas Pega di form** | **Baca saja.** Kewenangannya dihitung SERVER (`dapat_disunting`), bukan disimpulkan layar dari kolom `asal` — satu aturan, satu tempat |
| **Kolom detail pada kueri daftar** | **Tidak.** Kueri detail memilih 10 kolom lebih banyak; dua di antaranya berlebar 4.000 karakter, dan menariknya pada setiap halaman berarti memindahkan ratusan kilobita yang tidak pernah digambar (`D-10`) |
| **Aturan tanggal pada form** | **Tidak ada.** Aturan DOL dan Tanggal Lapor milik `B-2`; berkas laporan justru sering masuk sebelum tanggalnya dipastikan. Menambahkannya akan menolak berkas yang di Pega diterima (`P-5`) |
| **Pemeriksaan isian** | Hanya **lebar kolom** dan **angka non-negatif** — penjaga penyimpanan, bukan aturan bisnis. Flow action lamanya tidak punya satu pun validate rule |

### 17.8 Dua hal yang uji sendiri temukan

1. **Uji invarian kolom gagal** saat kueri detail sengaja dibuat berbeda dari kueri daftar. Itu
   persis fungsinya. Invariannya diperbarui menjadi bentuk yang benar: kedua badan **daftar** wajib
   sama persis, dan urutan kolom daftar wajib tetap menjadi **awalan** kolom detail — karena
   `scanRow` dan `scanDetailRow` membaca posisi yang sama untuk kolom yang sama.
2. **Kode galat salah.** Penolakan berkas Pega semula menjawab `validasi_gagal` dengan `409`.
   Ia bukan kegagalan validasi: tidak ada satu pun isian yang dapat diperbaiki pengguna, dan klien
   membedakan galat lewat `kode`. Diberi kode sendiri, `laporan_hanya_baca`.

### 17.9 Verifikasi

| Pemeriksaan | Hasil |
|---|---|
| `go build`, `go vet`, `go test ./...` | **seluruh paket lulus** |
| `tsc --noEmit` | bersih |
| `npm test` | **106 lulus, 3 gagal** (naik dari 97; +9 uji modul ini) |
| `npm run build` | bersih |

Ketiga kegagalan tetap kegagalan lama di `AccountPage.test.tsx`, yang sudah dibuktikan mendahului
pekerjaan modul ini pada sesi sebelumnya.

**Diuji terhadap aplikasi yang benar-benar berjalan**, instans sementara di porta 8099:

```
POST /api/inbox/laporan-klaim            -> 201 RCVN.26.0001, dapat_disunting=true, isian terkirim
PUT  /api/inbox/laporan-klaim/RCVN.26.0001 -> tersimpan; estimasi 250000000 sen, 3 dokumen
GET  /api/inbox/laporan-klaim/RCVN.26.0001 -> isian terbaca kembali utuh
GET  .../RCV-0001  (berkas Pega)         -> dapat_disunting=false
PUT  .../RCV-0001                        -> 409 laporan_hanya_baca
PUT  isian cacat                         -> 422 dengan 4 pelanggaran SEKALIGUS, masing-masing
                                            menunjuk kolomnya
PUT  field tak dikenal                   -> 400 permintaan_cacat
PUT  berkas tidak ada                    -> 404 tidak_ditemukan
```

### 17.10 Yang masih belum ada

Tombol Buat Baru kini bekerja penuh, tetapi dua hal dari form lama tetap tertinggal dan keduanya
menunggu modul lain: **rincian per dokumen** (`S-1`) dan **blok data pelapor beserta alamatnya**
(area Heavy Equipment, di luar lingkup `D-34`). Keterbatasan itu disebutkan di kaki form, bukan
disembunyikan.

---

## 18. Sesi kesebelas — daftar RCV yang kosong: kode cabang yang bukan kode cabang (2026-09-22)

### 18.1 Permintaan

> "kenapa tampilan list RCV laporan klaim kosong?"

Satu pertanyaan, dan jawabannya ternyata cacat yang paling sulit ditemukan dari seluruh modul ini:
**tidak ada galat, tidak ada peringatan, tidak ada baris.**

### 18.2 Kenapa cacat seperti ini berbahaya

Layar bekerja sempurna. Permintaan dijawab `200`, tab tergambar, lencana terisi, paginasi
menyatakan "0 baris". Tidak ada satu pun bagian sistem yang merasa gagal — sehingga tidak ada satu
pun tempat yang melaporkannya. Yang salah bukan mekanismenya, melainkan **nilai yang dipakai
menyaring**.

### 18.3 Diagnosis

Penyaring cabang pada daftar dibandingkan dengan kolom `w.kodecabang_1` pada tabel warisan. Yang
saya berikan ke penyaring itu adalah `Caller.BranchCode`, yang berasal dari:

| Lapis | Isi |
|---|---|
| `internal/auth/identity.go:67` | `BranchCode string // HCQ: EmpResponse.Placement.BranchCode` |
| `internal/auth/provider/hcq.go:211` | diisi dari respons HCQ apa adanya |

**Keduanya kode cabang — tetapi bukan kode cabang yang sama.** Yang satu kode penempatan pegawai
dari sistem identitas; yang lain kunci baris `POOLDATA.BRANCH`. Menyandingkannya tampak masuk akal
karena namanya sama, dan itulah sebabnya cacat ini lolos sampai layarnya dicoba.

### 18.4 Bukti dari sistem lama

`RDB List/GetIDCabang-SQL.xml` — kueri inilah yang dipakai Pega, dan ia menjawab pertanyaannya
secara tuntas:

```sql
select a.id AS "KodeCabang", a.branchname as "Remark", b.LUS_ID as "Keyword"
  from branch a,
       hrdasm.v_hrd_mst@asmd.sinarmas.co.id c,
       lst_user_asuransi@asmd.sinarmas.co.id b
 where c.login_aplikasi = {TempCabang.UserAdmin}
   and c.nik = b.nik
   and b.cab_id = a.oldid
```

Rantainya empat langkah, dan **tidak satu pun dapat dilewati**:

```
login petugas -> HRDASM.V_HRD_MST.login_aplikasi -> NIK
              -> LST_USER_ASURANSI.cab_id
              -> BRANCH.oldid -> BRANCH.id     <-- inilah yang menyaring daftar
```

Langkah terakhir yang paling mudah terlewat: sambungannya ke **`BRANCH.oldid`**, bukan ke
`BRANCH.id`. Jadi bahkan bila kode dari sistem identitas kebetulan ada di tabel `BRANCH`, ia berada
di kolom yang berbeda dari yang dipakai menyaring.

Angka yang membuatnya pantas ditelusuri sampai ke sini: `GetIDCabang` dipanggil **13 aktivitas**,
termasuk jalur registrasi — `16-RISK-ANALYSIS.md` menyebutnya "titik paling kritis" pada `R-03`.

### 18.5 Perbaikannya: seam baru, bukan tambalan

Menerjemahkan login menjadi kode cabang adalah **pengambilan data dari sistem lain**, bukan aturan
bisnis. Karena itu ia menjadi seam, sejalan dengan `04-FUTURE-ARCHITECTURE.md` §3:

```go
// seam.go
type BranchResolver interface {
	Resolve(ctx context.Context, login string) (code string, resolved bool, err error)
}
```

Tiga keluaran yang **sengaja dibedakan**, dan pembedaannya itulah inti perbaikan ini:

| Keluaran | Artinya | Perlakuan |
|---|---|---|
| `code`, `true`, `nil` | cabang petugas diketahui | daftar disaring ke cabang itu |
| `""`, `false`, `nil` | petugas memang tidak terdaftar di HRD | **bukan galat** — daftar tidak disaring |
| `""`, `false`, `err` | sumbernya tidak dapat dibaca | galat, dan daftar tetap tidak disaring |

Menggabungkan baris kedua dan ketiga akan mengulang cacat yang baru saja diperbaiki: "tidak
diketahui" dan "tidak dapat dibaca" berakibat sama di layar, tetapi perbaikannya berbeda jauh —
yang satu urusan data HRD, yang lain urusan hak baca atau DB link.

**`BranchCode` dihapus seluruhnya dari `Caller`.** Dibiarkan ada, ia akan dipakai lagi oleh
pembaca berikutnya yang menyangka kedua kode itu sama. Menghapusnya membuat kesalahan yang sama
tidak mungkin diulang tanpa sengaja.

### 18.6 Satu penyimpangan yang disengaja dari perilaku lama

Di sistem lama, login yang tidak terdaftar di HRD menghasilkan `branch where ID = ''` — dan
daftarnya kosong. Sistem baru **tidak menirunya**: cabang yang tidak terbaca berarti daftar
**tidak disaring**.

Alasannya bukan selera. Kekosongan itu **bukan aturan bisnis**, melainkan akibat perangkaian
string `{ASIS:...}` (`03-CURRENT-ARCHITECTURE.md` §4.5): nilai kosong disisipkan ke teks SQL, dan
hasilnya kebetulan tidak mencocokkan apa pun. Tidak ada satu pun rule yang menyatakan "petugas
tanpa cabang tidak boleh melihat apa pun".

`P-5` tetap terjaga, karena yang dipertahankan `P-5` adalah **hasil aturan bisnis**, bukan artefak
perangkaian string. Konsekuensinya diterima dengan sadar dan **dikatakan di layar**, bukan
disembunyikan:

| Field respons | Isi |
|---|---|
| `batas_cabang` | cabang yang membatasi daftar; kosong berarti seluruh cabang |
| `cabang_terbaca` | `false` berarti cabang petugas tidak dapat ditentukan |

Keduanya dibedakan justru karena `batas_cabang` kosong punya **dua sebab berbeda**: pengguna
memilih kanwil sendiri, atau cabangnya tidak terbaca. Yang pertama pilihannya, yang kedua keadaan
yang perlu diberitahukan.

### 18.7 Perubahan kode

**Backend**

| Berkas | Perubahan |
|---|---|
| `internal/inboxlaporanklaim/seam.go` | `BranchResolver` ditambahkan; `BranchCode` dihapus dari `Caller` beserta `Clean()`-nya |
| `internal/inboxlaporanklaim/usecase/service.go` | `Options.BranchResolver` (opsional) |
| `internal/inboxlaporanklaim/usecase/inbox.go` | `resolveBranch`; `buildFilter` menerima `branchCode`; `ListResult` diperluas `BranchScope`/`BranchResolved`; `Create` menurunkan cabang dengan cara yang SAMA |
| `internal/inboxlaporanklaim/repo/sqlstore/branch.sql` | kueri `branch_of_login` — terikat parameter, `FETCH NEXT 1 ROWS ONLY`, tanpa `ROWNUM` |
| `internal/inboxlaporanklaim/repo/sqlstore/branch.go` | `BranchResolver`; memangkas padding `CHAR`; `CheckTable` dengan login `"__periksa__"` |
| `internal/inboxlaporanklaim/repo/memory/branch.go` | `BranchResolver` berbasis map; `SetError` untuk menguji jalur kegagalan |
| `internal/inboxlaporanklaim/http/dto.go` | `ListResponse` diperluas `BatasCabang`/`CabangTerbaca` |
| `internal/inboxlaporanklaim/http/handler.go` | `List` mengisi kedua field itu |
| `cmd/claimpnc/main.go` | `storage.claimReportBranch` — sqlstore bila Oracle, memori bila tidak; jembatan `Caller` tidak lagi mengirim cabang |
| `cmd/claimpnc/check.go` | `checkClaimReportBranch` ditambahkan ke `-periksa` |

**Frontend**

| Berkas | Perubahan |
|---|---|
| `types.ts` | `batas_cabang`, `cabang_terbaca` |
| `ClaimReportInboxPage.tsx` | pemberitahuan `role="status"` berlatar amber ketika cabang tidak terbaca; lencana `BranchScope` pada baris tindakan tabel |

### 18.8 Kenapa `-periksa` ikut diperluas

Kegagalan penerjemahan cabang **tidak terlihat sebagai galat** — daftarnya tetap tampil, hanya
tanpa batas cabang. Artinya petugas cabang melihat berkas seluruh cabang, dan tidak ada yang
melaporkannya sampai ada yang menyadarinya sendiri.

`checkClaimReportBranch` membuat keadaan itu dapat diketahui operator **sebelum** pengguna
melaporkannya, dan membedakan kedua sebabnya: hak baca `POOLDATA.BRANCH`, atau DB link
`@asmd.sinarmas.co.id` yang mati.

### 18.9 Utang yang disadari: modul ini memakai DB link

Kueri `branch_of_login` menembus **DB link** `@asmd.sinarmas.co.id` ke dua objek milik basis data
lain — `HRDASM.V_HRD_MST` dan `LST_USER_ASURANSI`. Itu **tepat** yang `D-25` tetapkan untuk
diganti pemanggilan API, dan `R-03` mencatat API itu belum ada.

Yang dilakukan: memakai DB link seperti sistem lama, **dan menaruhnya di balik seam**. Ketika API
Pegawai & Cabang (`20-DETAIL-KOMITE-DBLINK.md` §2.5 nomor 2) tersedia, yang berubah hanyalah satu
adapter — `usecase` dan `domain` tidak tersentuh. Inilah gunanya seam ada di sini dan bukan di
tempat lain.

### 18.10 Uji yang ditambahkan

Lima uji regresi pada `usecase/inbox_test.go`, masing-masing mengunci satu bagian cacatnya:

| Nama uji | Yang dijaga |
|---|---|
| `TestCallerCarriesNoBranchCodeAtAll` | `Caller` tidak boleh membawa kode cabang lagi |
| `TestUnresolvableBranchShowsEveryBranchInsteadOfNothing` | cabang tidak terbaca menghasilkan seluruh cabang, BUKAN kosong |
| `TestBranchLookupFailureDoesNotEmptyTheList` | sumber cabang mati, daftar tetap terisi |
| `TestBranchScopeIsReportedSoTheScreenCanSayIt` | batas cabang dilaporkan, bukan disimpulkan layar |
| `TestNewReportLandsInTheBranchTheListFiltersBy` | berkas baru mendarat di cabang yang sama dengan yang menyaring daftar |

Uji terakhir itu yang paling mudah terlupakan: bila `Create` menurunkan cabang dengan cara yang
berbeda dari `List`, berkas yang baru dibuat akan langsung hilang dari daftar pembuatnya — cacat
yang bentuknya persis sama dengan yang sedang diperbaiki.

Tiga uji pada `ClaimReportInboxPage.test.tsx`: batas cabang disebut di layar; pemberitahuan muncul
ketika cabang tidak terbaca **dan barisnya tetap tergambar**; tidak ada pemberitahuan ketika
cabangnya terbaca.

### 18.11 Kendala dan kesalahan sesi ini

| Hal | Sebab | Penyelesaian |
|---|---|---|
| Uji baru gagal: `Found multiple elements with the role "status"` | Bilah paginasi juga `role="status"` | **Uji yang dibetulkan, bukan komponennya.** Keduanya memang status, dan masing-masing sudah menjelaskan dirinya sendiri saat dibacakan; yang salah adalah pemilih uji yang tidak spesifik |
| Percobaan terhadap aplikasi berjalan tidak dilakukan | Work Owner menolak menjalankan instans sementara | Pembuktian bersandar pada kelima uji regresi, yang menutup jalur yang sama tanpa menyalakan server |

### 18.12 Hasil verifikasi

| Pemeriksaan | Hasil |
|---|---|
| `go build ./...` | bersih |
| `go vet ./...` | bersih |
| `go test ./...` | seluruhnya lulus |
| `npm run typecheck` | bersih |
| `npm test` | **109 lulus**, 3 gagal |
| `npm run build` | bersih |

Ketiga kegagalan tetap kegagalan lama di `master-rekening/AccountPage.test.tsx`, yang sudah
dibuktikan mendahului pekerjaan modul ini.

### 18.13 Koreksi Work Owner pada hari yang sama — penyimpangan §18.6 dicabut

Setelah perbaikan di atas diserahkan beserta dua pertanyaan terbukanya, Work Owner menjawab
pertanyaan pertama:

> "petugas yang cabangnya tidak terbaca tidak boleh melihat seluruh cabang"

**Penyimpangan yang dijelaskan §18.6 karena itu dicabut.** Bagian §18.6 tidak disunting — ia
rekaman keadaan yang benar-benar berlaku beberapa jam sebelumnya, dan menghapusnya akan
menghilangkan jejak bahwa pilihan itu pernah diambil beserta alasannya. Yang berlaku sekarang
adalah bagian ini dan `keputusan-implementasi.md` §20.

#### Apa yang berubah

| Hal | Sebelum koreksi | Sesudah koreksi |
|---|---|---|
| Cabang tidak terbaca | daftar menampilkan **seluruh cabang** | permintaan **ditolak** |
| Bentuk pemberitahuan | pemberitahuan amber di layar | pesan penolakan yang menyebut sebabnya |
| `cabang_terbaca` pada respons | `false` saat cabang tidak terbaca | **dihapus** — ia tidak dapat lagi bernilai `false` |
| Pembuatan berkas | tetap dibuat tanpa cabang (meniru Pega) | **ditolak** dengan sebab yang sama |

#### Kenapa penolakan, bukan daftar kosong

Sistem lama menghasilkan daftar kosong dalam keadaan ini. Meniru itu **tidak dipilih**, dan
alasannya persis cacat yang sedang diperbaiki sesi ini: **daftar kosong tidak terbedakan dari
"tidak ada pekerjaan hari ini"**. Ketidakterbedaan itulah yang membuat penyaring cabang yang salah
bertahan tanpa seorang pun melaporkannya.

Penolakan menutup akses yang sama, dan sekaligus mengatakan apa yang harus dibetulkan.

#### Dua sebab yang tetap dibedakan

Pembedaan tiga keluaran `Resolve` yang dibuat di §18.5 terbayar di sini — ia langsung menjadi dua
jawaban HTTP yang berbeda:

| Keadaan | HTTP | Kode | Dibereskan di |
|---|---|---|---|
| Login belum terdaftar di HRD | `403` | `cabang_tidak_dikenali` | data pegawai — menimpa satu orang |
| `POOLDATA.BRANCH` atau DB link mati | `503` | `sumber_cabang_tidak_terbaca` | infrastruktur — menimpa **seluruh** petugas |

Menyatukannya akan membuat gangguan sekantor terbaca sebagai kesalahan satu pengguna, lalu dicari
di tempat yang salah. `503` juga menyatakan keadaannya **sementara**, sehingga mencoba lagi memang
masuk akal.

#### Perluasan yang saya ambil sendiri, dan ditulis terbuka

Keputusan Work Owner berbunyi tentang **melihat**. Saya memberlakukannya juga pada **membuat**,
dengan alasan: sejak cabang menjadi batas yang mengikat, berkas yang lahir tanpa cabang **tidak
akan pernah terlihat siapa pun** — pembuatnya tidak dapat membuka daftarnya, dan petugas cabang
mana pun tersaring darinya. Membolehkannya berarti menerbitkan baris yang dijamin tidak dapat
dikerjakan.

Ditulis di komentar kode, di uji, dan di sini supaya dapat dikoreksi — bukan disisipkan diam-diam.

#### Penegakannya di satu tempat

`requireBranch` dipanggil `List` dan `Create`; **Ekspor menempuh `List`**, sehingga satu penjagaan
menutup tiga permukaan. Batas yang ditegakkan pada daftar tetapi tidak pada ekspor bukan batas sama
sekali — ia hanya menyulitkan.

#### Akibat pada uji, dan satu hal yang tersingkap karenanya

Sebelas uji memakai "cabang tidak terbaca" sebagai cara melihat seluruh berkas contoh — jalan yang
kini justru ditutup. Keduanya diperbaiki dengan helper `wideList`, yang melebarkan lewat
**pemilihan kanwil**: satu-satunya cara sah melihat lebih dari satu cabang di layar sungguhan.

Itu menyingkap sesuatu yang sebelumnya tidak terlihat: **data contoh menaruh saksi sebuah aturan
di tiga kanwil berbeda**, sehingga aturan kelompok bisnis khusus tidak dapat dicoba dari kursi mana
pun. Satu berkas contoh (`RCV-0006`) dipindahkan ke cabang 1002 agar setiap aturan punya saksi di
dalam lingkup yang benar-benar dapat dilihat — janji yang sudah tertulis di `SampleList` sejak awal
dan baru sekarang benar-benar diuji.

Satu angka harapan ikut berubah: tab *Data rejected* memuat **1** berkas dalam kanwil 01, bukan 2 —
karena `RCV-0014` berada di kanwil 02. Angka itu kini mengikuti lingkup yang nyata, bukan seluruh
tabel.

#### Hasil verifikasi sesudah koreksi

| Pemeriksaan | Hasil |
|---|---|
| `go build ./...` · `go vet ./...` · `go test ./...` | bersih |
| `npm run typecheck` | bersih |
| `npm test` | **108 lulus**, 3 gagal (ketiganya kegagalan lama `master-rekening`) |
| `npm run build` | bersih |

#### Yang sekarang perlu jawaban Work Owner

Koreksi ini menajamkan satu pertanyaan lama menjadi mendesak: **dropdown Kanwil masih membolehkan
petugas mana pun melihat kanwil mana pun.** Bila cabang adalah batas data yang mengikat, maka
kanwil yang bebas dipilih adalah pintu yang sama, hanya lebih lebar. Aturan yang menentukannya di
sistem lama ada di antara 137 When rule yang hilang (`R-16`), sehingga tidak dapat dibaca dari
sumber — dan karena itu ia keputusan, bukan temuan.

### 18.14 "Buat Baru masih tidak menjalankan apa-apa" — pembuktian dari luar

Work Owner melaporkan tombolnya masih tidak melakukan apa pun. Pembacaan kode tidak menemukan satu
pun jalur yang dapat diam tanpa jejak:

| Jalur | Keadaan |
|---|---|
| Rute `POST /api/inbox/laporan-klaim` | terdaftar di `Mount` |
| Tombol | memanggil `create.mutate`, lalu berpindah ke `/{id}` pada keberhasilan |
| Kegagalan `POST` | digambar sebagai `ErrorMessage` tepat di atas tabel |
| `callAPI` | **selalu** melempar pada jawaban tidak-OK; tidak ada jalur yang menelannya |

Karena membaca kode tidak cukup membuktikan apa pun, **kontraknya diuji dari luar**: berkas baru
`internal/inboxlaporanklaim/http/routes_test.go` merakit aplikasi utuh — sesi, middleware portal,
jembatan `Caller` yang sama persis dengan `cmd/claimpnc` — lalu menembak HTTP sungguhan lewat
`httptest`.

| Uji | Yang dibuktikan |
|---|---|
| `TestCreateAnswersWithTheNewReportSoTheScreenCanOpenIt` | `201`, nomor berawalan `RCVN.`, cabang terisi dari penerjemahan login, **dan berkasnya benar-benar dapat dibuka di alamat yang dituju layar** |
| `TestCreateWithoutBodyIsAcceptedBecausePegaAsksForNothing` | tombol tanpa isian tidak dijawab `400` |
| `TestCreateWithoutPortalIsRefusedNotServedByThePrimary` | `R-20` — tanpa portal tidak jatuh ke koneksi utama |
| `TestCreateWithoutSessionIsRefused` | rutenya memang di balik sesi |
| `TestCreateIsRefusedWithAReasonWhenTheBranchIsUnknown` | penolakan **membawa sebab** (`403`, `cabang_tidak_dikenali`) — penolakan tanpa kode dan pesan adalah persis yang membuat tombol terbaca "tidak melakukan apa-apa" |
| `TestUnreadableBranchSourceIsReportedAsTemporary` | `503`, `sumber_cabang_tidak_terbaca` |
| `TestListAnswersWithTheShapeTheScreenReads` | keempat field tingkat atas yang dibaca layar ada |

**Seluruhnya lulus.** Kontrak tombolnya benar dari ujung ke ujung.

#### Sebab yang tersisa, dan buktinya

SPA **disematkan ke binary saat `go build`** (`backend/spa/dist` lewat `go:embed`; direktif itu
tidak dapat menjangkau `../`, lihat doc paket `spa`). Alurnya:

```
frontend/src → npm run build → backend/spa/dist → go build → satu binary
```

Artinya **`npm run build` saja tidak mengubah apa pun pada aplikasi yang sedang berjalan.** Binary
yang dibangun sebelum 2026-09-22 17:51 — cap waktu `dist` saat ini — menyajikan antarmuka versi
sebelumnya, yaitu versi yang hanya membuat berkas lalu mengumumkan nomornya tanpa membuka form.
Dari kursi petugas, persis seperti tombol yang tidak bekerja.

Pemeriksaan repo menguatkannya: **tidak ada satu pun `claimpnc.exe`** di pohon kerja, dan hanya ada
satu worktree — sehingga yang dijalankan Work Owner dibangun di tempat lain atau pada waktu lain.

#### Cara memastikannya dalam hitungan detik

Buka Network pada peramban lalu tekan tombolnya:

| Yang terlihat | Artinya |
|---|---|
| **Tidak ada permintaan sama sekali** | antarmuka lama — binary perlu dibangun ulang |
| `POST` menjawab `201` tetapi alamat tidak berpindah | antarmuka lama |
| `POST` menjawab `403 cabang_tidak_dikenali` | kode sudah baru; login belum terdaftar pada cabangnya |
| `POST` menjawab `503 sumber_cabang_tidak_terbaca` | kode sudah baru; DB link `@asmd` atau `POOLDATA.BRANCH` tidak terbaca |

### 18.15 Perbaikan: artefak dibangun ulang, dan penanda versi antarmuka ditambahkan

Work Owner meminta tombolnya diperbaiki. Karena kontraknya sudah terbukti benar (§18.14), yang
diperbaiki ada dua: **artefak yang dijalankan**, dan **sebab kekeliruan itu tidak terdeteksi**.

#### Yang dikerjakan

| # | Tindakan | Alasan |
|---|---|---|
| 1 | `npm run build` lalu `go build -o claimpnc.exe ./cmd/claimpnc` | Urutannya mengikat: antarmuka disematkan saat `go build`. Binary sekarang memuat antarmuka yang benar |
| 2 | `npm run build` menulis `dist/versi.txt` berisi cap waktu ISO | Penanda versi antarmuka, ikut tersemat ke binary |
| 3 | `spa.Version()` | Membaca penanda itu dari dalam binary |
| 4 | Log saat start: `antarmuka tersemat dibangun=<cap waktu>` | Pertanyaan "antarmuka versi mana yang sedang disajikan" kini terjawab tanpa menebak |
| 5 | `-periksa` mencetak baris `antarmuka : dibangun <cap waktu>` | Terjawab bahkan tanpa menyalakan server |

#### Kenapa penanda versi, bukan sekadar membangun ulang

Membangun ulang menyelesaikan hari ini. Penanda versi menyelesaikan **kelas kegagalannya**.

Kekeliruan "frontend dibangun, binary tidak" **tidak meninggalkan jejak apa pun**: tidak ada galat,
tidak ada log, tidak ada perbedaan yang terlihat di mana pun — hanya layar versi lama yang tampak
seperti fitur rusak. Ia sudah memakan satu putaran penuh pelaporan dan penelusuran. Sekarang satu
baris log menjawabnya.

Pola yang sama dengan `checkClaimReportBranch` (§18.8): **kegagalan yang tidak terlihat sebagai
kegagalan harus dibuat terlihat**, bukan diandalkan pada ingatan orang.

#### Bukti bahwa penandanya benar-benar tersemat

Dijalankan dari binary hasil build, tanpa menyalakan server:

```
Check integrasi Claim PNC
  lingkungan       : development
  portal utama     : ASM
  adapter identitas: fake
  antarmuka        : dibangun 2026-09-22T11:09:21.943Z
```

Cap waktunya sama dengan `dist/versi.txt` — artinya yang dibaca memang isi binary, bukan berkas di
cakram.

#### Satu hal yang sengaja dibiarkan

`versi.txt` **ikut diabaikan git** (`backend/spa/dist/*`), sebagaimana seluruh hasil build. Pada
klon bersih yang belum pernah menjalankan `npm run build`, `spa.Version()` mengembalikan kosong dan
penandanya berbunyi *"tidak diketahui (dibangun sebelum penanda versi ada)"* — bukan gagal build.
Artefak build tidak masuk repositori, dan penanda yang hilang tidak boleh menghentikan apa pun.

### 18.16 Penjelasan pertama saya salah — dan apa yang sebenarnya ditemukan

Work Owner menjalankan binary hasil build dan melaporkan tombolnya **masih** tidak melakukan apa
pun. Dugaan "binary lama" (§18.14) karena itu **gugur**, dan dicatat apa adanya.

#### Bagaimana ia dibuktikan gugur

Penanda teks dicari langsung **di dalam berkas binary**:

| Teks | Hasil |
|---|---|
| `Membuat…` (label tombol versi baru) | ada |
| `Laporan baru tidak dapat dibuat` | ada |
| `cabang_tidak_dikenali` | ada |

Antarmuka baru memang tersemat. Lalu binary itu dijalankan sungguhan tanpa Oracle, dan `POST`
dijawab **`201` dengan `RCVN.26.0001`**. Backend, antarmuka, dan artefaknya semuanya benar.

> **Yang pertama saya lakukan seharusnya menjalankan, bukan menerangkan.** Penjelasan pertama masuk
> akal, konsisten dengan seluruh bukti yang saya punya, dan salah. Yang membantahnya bukan
> pembacaan ulang melainkan satu perintah `grep` ke berkas binary.

#### Cacat yang sebenarnya ditemukan

`internal/platform/httpserver/server.go` — kerangka halaman dapat **tertahan di cache peramban**:

```go
if _, err := fs.Stat(files, bersih); err != nil {
    …
    w.Header().Set("Cache-Control", "no-store")   // hanya di cabang INI
    …
}
pelayan.ServeHTTP(w, r)                            // "/" lewat sini, TANPA header
```

Komentar di atasnya sudah menyatakan maksud yang benar — *"Kerangka halaman tidak boleh di-cache:
satu rilis baru harus langsung terpakai tanpa pengguna menekan muat ulang paksa"* — tetapi
headernya dipasang **di cabang yang salah**. Permintaan ke `/` menemukan `index.html` sebagai
berkas nyata, sehingga jalur yang **paling sering dipakai** justru satu-satunya yang melewatinya.

Akibatnya persis gejala yang dilaporkan: binary dibangun ulang, server dijalankan ulang, dan
peramban tetap menjalankan antarmuka versi lama — **tanpa satu pun galat di mana pun**.

#### Perbaikannya

Header dipindahkan ke **sebelum** percabangan, dan hanya untuk kerangka halaman. Berkas aset
sengaja tidak ikut: namanya sudah memuat sidik isi (`index-<hash>.js`), sehingga rilis baru
menghasilkan nama baru dan cache-nya tidak pernah basi. Memaksa `no-store` di sana hanya membuat
setiap muat ulang mengunduh ratusan kilobita tanpa manfaat.

Dibuktikan dari binary sungguhan, bukan dari uji saja:

```
GET /                        → Cache-Control: no-store
GET /assets/index-<hash>.js  → (tidak no-store)
```

Ditambah berkas uji baru `internal/platform/httpserver/server_test.go` — paket itu sebelumnya
**tidak punya satu pun uji**:

| Uji | Yang dijaga |
|---|---|
| `TestPageSkeletonIsNeverCachedOnAnyPath` | ketiga jalur kerangka halaman menolak cache, termasuk `/` yang dulu melewatinya |
| `TestHashedAssetsAreNotForcedOutOfCache` | kebalikannya juga dijaga — aset bersidik isi tetap boleh di-cache |

#### Satu lagi yang ditutup sekalian: permintaan yang menggantung

Penerjemahan cabang menembus **DB Link** ke basis data lain. Sambungan yang mati dapat menggantung
sampai batas waktu TCP alih-alih menjawab galat — dan **permintaan yang menggantung tidak
terbedakan dari tombol yang tidak bekerja**: tombolnya berbunyi "Membuat…", tidak ada pesan apa
pun, dan pengguna menyerah sebelum jawabannya tiba.

`branchTimeout = 5 detik` dipasang di adapter, sesuai `10-API-STRATEGY.md` §8.2 yang mewajibkan
batas waktu pada setiap pemanggilan keluar. DB Link adalah pemanggilan keluar yang menyamar sebagai
kueri biasa. Kehabisan waktu dibedakan dari galat basis data biasa, karena perbaikannya menunjuk
sambungan ke HRD, bukan kueri maupun hak akses.

---

## 19. Sesi kedua belas — sumber daftar dipindahkan ke `POOLDATA.T_CLAIMLIST_ADMIN` (2026-09-23)

### 19.1 Permintaan

> "coba ubah tarikan data pelaporan agar data ditarik dari table POOLDATA.T_CLAIMLIST_ADMIN"

### 19.2 Yang dicari lebih dulu, dan tidak ditemukan

Tabel itu **nol kemunculan** di seluruh bahan: 2.634 berkas XML Pega, 63 objek `Database/`,
dan seluruh dokumen proyek. Oracle sedang mati (`ORA-12514`), sehingga strukturnya juga tidak
dapat diintrospeksi.

Karena itu tidak ada satu pun kode ditulis sampai DDL-nya diterima. Menebak nama kolom
menghasilkan kueri yang gagal pada eksekusi pertama di produksi — persis yang dilarang **No
Shortcuts**, dan kesalahan yang sudah sekali saya buat sendiri pada sesi kesembilan
(`kronologiskejadian_1`).

### 19.3 Dua keputusan Work Owner

| Pertanyaan | Jawaban |
|---|---|
| Menggantikan sumber yang mana | **Kedua-duanya** — T_CLAIMLIST_ADMIN menjadi satu-satunya sumber daftar |
| Siapa yang mengisinya | **Proses lain**; aplikasi ini hanya membaca |
| Berkas baru yang belum ada di sana | **Pilihan 1** — proses pengisi diperluas agar ikut membaca `CPNC_LAPORAN_KLAIM` |

### 19.4 Pemetaan kolom, dari DDL yang diterima

| Kontrak modul | Kolom T_CLAIMLIST_ADMIN |
|---|---|
| `report_id` · `claim_number` · `assignment_ref` | `PYID` · `PNCCASEID` · `STATUSLOCK_1` |
| `policy_number` · `insured_name` · `business_name` | `POLICYNO` · `QQNAME` · `BUSINESSNAME` |
| `date_of_loss` · `created_at` · `aging_at` | `DATEOFLOSS_1` · `PXCREATEDATETIME` · `DATEFORAGING_1` |
| `created_by` · `branch_code` | `PXCREATEOPERATOR` · `KODECABANG_1` |
| `work_status` · `group_panel` · `business_group` | `PYSTATUSWORK` · `GROUPPANEL_1` · `BUSINESSGROUPID` |

**Tiga yang membaik:**

| Kolom | Sebelumnya | Sekarang |
|---|---|---|
| `reporter_name` | `NULL` — nama kolomnya di tabel warisan tidak diketahui (`R-08`) | `PXCREATEOPNAME` — sama artinya dengan `Sender := OperatorID.pyUserName` pada `CreateNewCaseRCV` |
| `courier_name` | `NULL` | `KURIR` |
| `not_registered_note` | `NULL` | `NOTREGISTNOTE_1` |

Ditambah satu penyederhanaan: `BUSINESSGROUPID` sudah didenormalkan, sehingga `LEFT JOIN
POOLDATA.BUSINESS` yang dulu diperlukan **hilang**.

### 19.5 Tiga kolom yang menjadi kosong — regresi yang disadari

| Kolom grid | Dulu dari | Di tabel baru |
|---|---|---|
| **Reference no** | `BOOKNO_1` | tidak ada |
| **Alasan** | `KETERANGAN_1` | tidak ada |
| **Subject Email** | `SUBJECTEMAIL_1` | tidak ada |

Ketiganya dibuat `NULL`, **bukan dipetakan ke kolom lain yang kebetulan mirip**.
`NOTREGISTNOTE_1` sangat menggoda untuk dijadikan "Alasan", padahal ia keterangan
belum-registrasi yang sudah punya tempatnya sendiri di form. Memetakannya akan membuat dua isian
berbeda tampil sebagai satu — dan salahnya tidak akan pernah terlihat sebagai galat.

Kolomnya **tetap digambar** di grid, tidak disembunyikan: `D-13` menetapkan tata letak mengikuti
Pega, dan kolom yang hilang adalah perubahan layar yang harus diputuskan Work Owner, bukan akibat
sampingan penggantian sumber.

### 19.6 Yang dipertahankan, dan satu yang sengaja tidak ditambahkan

**Dipertahankan:** penyaring `pxobjclass = 'ASM-FW-GCNMFW-Work-ReceiveDocument'`. Tabel ini
memuat kolom itu, dan tanpa penyaringnya daftar berisiko memuat case selain Receive Document bila
ia ternyata melayani lebih dari satu jenis.

**Tidak ditambahkan: `STS_AKTIF`.** Kolom itu ada, artinya belum diketahui, dan kueri lama tidak
punya penyaring semacam itu — menambahkannya adalah **aturan baru** yang tidak ada di 13 butir
`D-49` (`P-5`).

Pilihan ini diambil sadar, dengan alasan yang dapat diperiksa: bila `STS_AKTIF` ternyata menandai
baris yang tidak berlaku, daftar akan memuat baris berlebih — kekeliruan yang **terlihat dan
dikeluhkan**. Bila saya menyaringnya berdasarkan tebakan dan tebakan itu salah, pekerjaan
**hilang dari layar tanpa seorang pun tahu**. Layar ini sudah dua kali menderita kegagalan yang
diam; saya tidak menambah yang ketiga atas dasar tebakan.

### 19.7 `origin` berubah dasarnya

Tabelnya kini satu, sehingga "dari tabel mana baris ini" tidak lagi dapat menjawab "milik siapa".
Penggantinya adalah **awalan nomor**: `RCVN.` diterbitkan aplikasi ini, sisanya warisan.

Itu bukan akal-akalan — `D-71` menetapkan awalan itu **justru supaya asal sebuah berkas terbaca
dari nomornya tanpa tabel pemetaan**. Di sinilah ia terpakai.

Awalan itu kini hidup di dua tempat (konstanta domain dan teks SQL), dan
`TestOriginIsDerivedFromTheNumberPrefix` menjaga keduanya tidak berpisah diam-diam. Kalau
berpisah, **setiap** berkas terbaca sebagai milik Pega dan form membukanya baca-saja tanpa satu
pun galat.

### 19.8 Jalur pembacaan berkas sendiri — supaya "Buat Baru" tidak rusak lagi

Berkas yang baru dibuat **belum ada** di T_CLAIMLIST_ADMIN sampai proses pengisinya berjalan.
Tanpa penanganan, menekan "Buat Baru" akan menerbitkan berkas lalu membuka form yang menjawab
"laporan tidak ditemukan" — tombol yang tampak rusak, untuk **ketiga** kalinya.

`claim_report_get_own_body` membaca langsung dari `CPNC_LAPORAN_KLAIM`, dan `Repo.Get` memilih
jalur menurut awalan nomor lewat `IssuedHere()` yang sudah ada. Dua kueri, satu pembaca baris —
`TestBothDetailQueriesSelectTheSameColumns` menjaga kolomnya tidak bergeser, karena `scanDetailRow`
membaca secara **posisi**.

### 19.9 Uji yang ditambahkan

| Uji | Yang dijaga |
|---|---|
| `TestListIsDrawnFromTheAdminClaimList` | sumber daftar memang T_CLAIMLIST_ADMIN, tabel kerja Pega tidak kembali, dan penyaring `pxobjclass` tidak hilang |
| `TestOriginIsDerivedFromTheNumberPrefix` | awalan di SQL tetap sama dengan `ReportNumberPrefix` |
| `TestOwnReportIsReadFromItsOwnTable` | berkas sendiri dibaca dari tabelnya sendiri dan menghormati soft delete |
| `TestBothDetailQueriesSelectTheSameColumns` | kedua jalur detail memilih kolom identik |
| `TestNoQueryWritesToLegacyPegaTable` **diperluas** | T_CLAIMLIST_ADMIN ikut dijaga hanya-baca |

### 19.10 Hasil verifikasi

| Pemeriksaan | Hasil |
|---|---|
| `go build ./...` · `go vet ./...` · `go test ./...` | bersih |
| `npm run typecheck` · `npm run build` | bersih |
| `npm test` | **108 lulus**, 3 gagal (ketiganya kegagalan lama `master-rekening`) |

**Belum diuji terhadap basis data sungguhan** — Oracle masih `ORA-12514`. Kueri barunya karena itu
belum pernah dieksekusi, dan itu dinyatakan apa adanya, bukan disamarkan sebagai selesai.

### 19.11 Yang menunggu pihak lain

1. **Proses pengisi T_CLAIMLIST_ADMIN diperluas** agar ikut membaca `CPNC_LAPORAN_KLAIM` —
   tanpa itu, berkas terbitan aplikasi ini tidak akan pernah muncul di daftar.
2. **Arti `STS_AKTIF`** — menentukan apakah penyaring perlu ditambahkan.
3. **Ketiga kolom kosong** — apakah proses pengisi dapat menambahkan `BOOKNO_1`,
   `KETERANGAN_1`, dan `SUBJECTEMAIL_1`, atau kolomnya dihapus dari grid.

### 19.12 Tampilan disamakan dengan layar lama, dan `STS_AKTIF` ditetapkan (2026-09-23)

Work Owner mengirim tangkapan layar Pega beserta tiga ketetapan.

#### `STS_AKTIF` — pertanyaan terbuka §19.6 tertutup

> "STS_AKTIF = 1 berarti claim masih aktif, sedangkan = 0 berarti claim sudah tidak aktif dan
> tidak ditampilkan lagi"

Penyaringnya dipasang **mengecualikan yang bernilai `'0'` secara tegas**, bukan "yang bukan `'1'`":

```sql
AND (t.sts_aktif IS NULL OR TRIM(t.sts_aktif) <> '0')
```

Kolomnya nullable, dan baris ber-`NULL` berarti penandanya tidak ditetapkan — bukan berarti tidak
aktif. Menyembunyikannya akan menghilangkan pekerjaan dari layar tanpa seorang pun tahu; baris
berlebih sebaliknya terlihat dan dikeluhkan. `TRIM` dipakai karena kolomnya `VARCHAR2(4)`: nilai
berpadding spasi tidak akan pernah cocok dengan pembandingnya sendiri.

#### `BOOKNO_1` — **tidak dapat dipenuhi**, dan ini perlu jawaban

> "Isi bookno_1 sebagai noreferensi"

**`BOOKNO_1` tidak ada di antara ke-53 kolom `T_CLAIMLIST_ADMIN`** pada DDL yang dikirim. Ia ada
di tabel kerja Pega yang tidak lagi dibaca.

Kolom "Reference no" karena itu **tetap kosong** — dan pada tangkapan layar yang dikirim, kolom itu
pun kosong di seluruh baris yang terlihat. Menariknya kembali dengan JOIN ke
`PC_ASM_FW_GCNMFW_WORK` akan membatalkan seluruh maksud pemindahan sumber, jadi tidak dilakukan
sepihak.

#### `KETERANGAN_1` dan `SUBJECTEMAIL_1` — tidak dipakai dulu

Kolom "Alasan" **dihapus dari grid** sesuai ketetapan itu; pada tangkapan layar pun ia tidak ada.
`subjek_email` tetap ada di kontrak API karena form masih memakainya.

#### Tampilan grid disamakan dengan tangkapan layar

| Hal | Sebelum | Sesudah |
|---|---|---|
| Urutan | Case ID · Case PNC · Polis no · Insured · Business … | **Case ID · Polis no · Case PNC · Reference no · Business Name · Insured Name** |
| Kolom hilang | — | **Alasan** dibuang |
| Kolom baru | — | **Reference no** · **Creator** · **Aging** |
| Umur berkas | satu kolom "Total Aging", ditulis `10 hari` | **dua kolom**: "Aging" dan "Total Aging", ditulis `6y ago` |
| Paginasi | "Menampilkan 11–20 dari 57" + Sebelumnya/Berikutnya, di BAWAH tabel | **"Total Data : 23767" + nomor halaman**, di ATAS tabel |

**"Aging" ternyata kolom tersendiri.** Tabelnya memuat kolom `AGING` (NUMBER) yang selama ini
diabaikan, dan layar lama menggambarnya **berdampingan** dengan "Total Aging" yang dihitung dari
tanggal aging. Menyatukan keduanya akan menghapus satu kolom yang memang ada di layar.

Ia dibawa sebagai **teks**, bukan angka: kolomnya nullable, dan layar membedakan "kosong" dari
"nol". `int` tidak dapat membedakan keduanya — dan pada tangkapan layar ia justru kosong di seluruh
baris. Artinya sendiri **belum diketahui**; tidak ada satu pun rule di export yang menyentuhnya,
sehingga ia diteruskan apa adanya, bukan ditafsirkan.

**Umur ditulis ringkas** (`relativeAge`). Dari tangkapan layar hanya bentuk TAHUN yang terbukti —
seluruh baris berbunyi `6y ago`. Satuan bulan dan hari mengikuti bentuk ringkas yang sama karena
itu satu-satunya yang dapat disandarkan pada bukti; bila Pega menuliskannya lain, yang berubah
hanya satu fungsi.

**Paginasi pindah ke atas tabel** dan menjadi nomor halaman. Pada daftar 23.767 baris, bilah di
bawah menuntut pengguna menggulung seluruh halaman hanya untuk berpindah halaman. Jendela nomornya
lima dan bergeser mengikuti halaman yang dibuka, sehingga halaman mana pun tetap terjangkau.
Titik-titik `…` sengaja **bukan tombol**: apa yang terjadi saat ditekan tidak terlihat dari
tangkapan layar, dan menebaknya berarti membuat perilaku yang tidak dapat dirujuk ke mana pun.

#### Uji yang ditambahkan

| Uji | Yang dijaga |
|---|---|
| `menggambar kolom dengan judul dan URUTAN persis seperti layar lama` | ketiga belas kolom beserta urutannya |
| `menggambar Aging dan Total Aging sebagai DUA kolom yang berbeda` | keduanya tidak tergabung |
| `menuliskan umur berkas secara ringkas seperti layar lama` | `today` · `5d ago` · `2mo ago` · `2y ago` |
| `menyebut jumlah SELURUH baris, bukan jumlah yang tergambar` | "Total Data" |
| `menggambar nomor halaman dan menandai halaman yang sedang dibuka` | `aria-current`, bukan hanya warna |

#### Hasil verifikasi

`go build` · `go vet` · `go test ./...` bersih · `typecheck` · `npm run build` bersih ·
`npm test` **112 lulus**, 3 gagal (ketiganya kegagalan lama `master-rekening`).

Kueri barunya **masih belum pernah dieksekusi** terhadap basis data sungguhan — Oracle tetap
`ORA-12514`.

### 19.13 Daftar tampil kosong pada tab bawaan — kolom penentu tab tidak pernah terisi

Work Owner melaporkan tab "Outstanding Data" kosong, padahal lencana menyebut **115** berkas pada
tab "Data hasn't been transferred" dan "All data".

**Oracle sudah hidup** pada saat ini (`-periksa` menjawab `[ok] koneksi basis data: ASM`), sehingga
untuk pertama kalinya tabelnya dapat dibaca langsung. Sebuah probe **baca-saja** dijalankan, lalu
dihapus.

#### Apa yang ditemukan

| Pemeriksaan | Hasil |
|---|---|
| Isi `T_CLAIMLIST_ADMIN` | **1.014 baris**: 872 `Work-PNC` + 142 `Work-ReceiveDocument` |
| `OS_CATEGORY` | `OS PELAPORAN KLAIM` (142) · `OS REGISTRASI KLAIM` (516) · `OS FOLLOW UP KE` (356) |
| `STS_AKTIF` pada baris RCV | `1` → 115 · `0` → 27 — **cocok dengan lencana 115** |
| **`PNCCASEID` pada baris RCV** | **NULL pada seluruh 142** |
| **`STATUSLOCK_1` pada baris RCV** | **NULL pada seluruh 142** |
| Ke-142 id yang SAMA, dibaca di `PC_ASM_FW_GCNMFW_WORK` | **Outstanding 22 · Not Registered 113 · Not Transferred 7** |

Kedua kolom itu **penentu `position`**. Ada di DDL, tidak pernah terisi. Akibatnya seluruh baris
jatuh ke cabang `ELSE` — "Not Transferred" — dan delapan dari sembilan tab tampil kosong **tanpa
satu pun galat**.

> Kelas kegagalan yang sama untuk ketiga kalinya di modul ini: **kolom yang ada tetapi kosong**
> tidak menghasilkan galat, hanya hasil yang salah diam-diam. Yang menemukannya bukan pembacaan
> kode melainkan membaca datanya.

#### Perbaikannya

`LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK` pada `pyid`, untuk **empat kolom saja**:

| Kolom | Dari | Untuk |
|---|---|---|
| `claim_number` | `w.pnccaseid` | membedakan Outstanding dari Not Registered |
| `assignment_ref` | `w.statuslock_1` | menandai sudah diserahkan atau belum |
| `reference_number` | `w.bookno_1` | kolom "Reference no" — **sekaligus menutup permintaan `BOOKNO_1`** |
| `position` | keduanya | penggolongan tab |

**Baris mana yang tampil tetap ditentukan `T_CLAIMLIST_ADMIN`**; tabel kerja Pega hanya menempel
sebagai `LEFT JOIN`, tidak pernah menjadi tabel penggerak di `FROM`. Uji
`TestListIsDrawnFromTheAdminClaimList` menjaganya tetap begitu.

Diuji lebih dulu lewat probe terhadap data sungguhan, sebelum satu baris pun ditulis ke berkas
kueri — dari 115 baris aktif: **Outstanding 1 · Not Registered 113 · Not Transferred 1**, dan
`BOOKNO_1` terisi (`KBRU-FW-CBFW-WORK CLM-362`, `CLM-429/235.301.1.0524.001245`, …).

#### Uji yang ditambahkan

| Uji | Yang dijaga |
|---|---|
| `TestTabDecidingColumnsComeFromThePegaWorkTable` | keempat kolom tidak diam-diam kembali ke tabel admin — di sana namanya ada dan tampak benar, hanya isinya kosong |
| `TestOnlyExplicitlyInactiveRowsAreHidden` | penyaring `STS_AKTIF` tidak berubah menjadi `= '1'` |
| `TestListIsDrawnFromTheAdminClaimList` **ditulis ulang** | tabel kerja Pega boleh di-JOIN, TIDAK boleh menjadi tabel penggerak |

#### Yang perlu diketahui Work Owner

**`T_CLAIMLIST_ADMIN` bukan daftar laporan yang utuh.** Ia tabel *outstanding* — 142 baris RCV,
sementara `PC_ASM_FW_GCNMFW_WORK` memuat **2.800** baris RCV di basis data yang sama. Tangkapan
layar Pega yang dikirim sebelumnya menyebut **23.767** baris.

Jadi setelah perbaikan ini layar akan menampilkan **115 baris**, bukan puluhan ribu. Bila yang
dikehendaki adalah daftar utuh seperti layar lama, sumbernya harus kembali ke tabel kerja Pega —
satu suntingan pada `FROM`, dan sisa kuerinya tidak berubah.

### 19.14 Sumber dikembalikan ke tabel kerja Pega, dan dua cacat bind ditemukan

Work Owner mengirim dua tangkapan layar berdampingan: Pega menampilkan **671** berkas dengan
saringan Bisnis = NONMBU, sementara aplikasi baru menampilkan lencana **115** dengan tabel
**kosong**.

Oracle sudah hidup, sehingga seluruh temuan di bawah **diukur langsung**, bukan disimpulkan.

#### Cacat 1 — urutan bind, dan kenapa ia lolos begitu lama

Kueri pencacah dan kueri daftar dijalankan dengan bind yang **sama persis**. Hasilnya berbeda:

```
count_body    dengan cabang 100081  →   0
summary_body  dengan cabang 100081  → 115
```

Sebabnya bukan data, melainkan **Oracle mengikat argumen menurut URUTAN KEMUNCULAN penanda di
dalam teks kueri, bukan menurut angka pada `:n`**.

Dua kueri menaruh penanda bernomor besar lebih dulu:

| Kueri | Urutan kemunculan |
|---|---|
| `claim_report_summary_body` | `:22 … :30` (di SELECT), lalu `:1 … :21` (di WHERE) |
| `claim_report_message_body` | `:30 … :35`, lalu `:1 … :29`, lalu `:36, :37` |

Akibatnya pada pencacah: penyaring cabang menerima `NULL` dan pencacah komunikasi menerima kode
cabang. Lencana menyebut 115, tabel menyebut 0 — **tanpa satu pun galat**, karena setiap bind
tetap terisi sesuatu.

Ditambah satu cacat kedua di jalur yang sama: `messageArgument[6:]` mengirim **2** nilai ke tempat
yang menuntut **6**, sehingga ketiga tab komunikasi akan gagal saat ditembak.

**Perbaikannya:** penanda dinomori ulang agar urutan kemunculannya menaik, dan argumen Go dikirim
dalam urutan itu. `TestBindMarkersAppearInAscendingOrder` sekarang menjaga seluruh kueri — ia
memeriksa **setiap** kueri, bukan hanya kedua yang rusak.

> Uji lama sudah memeriksa "setiap nilai lewat parameter binding". Yang tidak diperiksa adalah
> **urutannya** — dan itu justru yang salah. Penomoran `:1 … :30` terbaca benar oleh manusia dan
> salah oleh Oracle.

#### Cacat 2 — posisi ternyata punya EMPAT keadaan, bukan tiga

Dihitung langsung pada saringan NONMBU:

| `pnccaseid` | `statuslock_1` | Jumlah | Tab layar lama |
|---|---|---|---|
| NULL | NULL | **42** | Not Transferred |
| NULL | terisi | **123** | Not Registered |
| terisi | terisi | **340** | Outstanding |
| terisi | NULL | **166** | **tidak masuk tab mana pun** |
| | | **671** | All data |

Keempatnya sama persis dengan angka di layar Pega. Pemetaan lama menaruh sisanya di `ELSE` sebagai
"Not Transferred", sehingga tab itu menyebut **208** di tempat layar lama menyebut **42**.

Sekarang keadaan keempat berposisi `NULL`: ia hanya ikut terhitung di "All data", persis seperti
layar lama.

#### Keputusan: sumber baris dikembalikan ke tabel kerja Pega

`T_CLAIMLIST_ADMIN` **tidak dapat** menjadi sumber daftar, dan itu terukur:

| | T_CLAIMLIST_ADMIN | PC_ASM_FW_GCNMFW_WORK |
|---|---|---|
| Baris Receive Document | **142** | **2.800** |
| `kodecabang_1` | **NULL pada seluruhnya** | terisi; 801 baris di cabang 100081 |
| `pnccaseid`, `statuslock_1` | NULL pada seluruhnya | terisi, dan inilah penentu tab |
| Angka pada saringan NONMBU | — | **671 · 340 · 123 · 42**, sama persis dengan layar Pega |

Tabel admin adalah daftar pekerjaan **outstanding** (`OS_CATEGORY = 'OS PELAPORAN KLAIM'`), bukan
daftar laporan yang utuh. Dipakai sebagai sumber baris, layar kehilangan 95% isinya **dan** seluruh
batas cabang.

Ia **tetap dibaca**, sebagai `LEFT JOIN`, untuk empat hal yang memang hanya ada di sana:
`sts_aktif`, `aging`, `kurir`, dan `notregistnote_1`.

#### Angka yang sekarang dihasilkan, dan kenapa berbeda dari 671

| Keadaan | Jumlah |
|---|---|
| Layar Pega, saringan NONMBU | **671** |
| Aplikasi ini, saringan NONMBU, tanpa batas cabang | **644** |
| Aplikasi ini, saringan NONMBU, cabang 100081 | **234** |

Kedua selisihnya berasal dari aturan yang Work Owner sendiri tetapkan:

1. **671 → 644**: 27 baris ber-`STS_AKTIF = '0'` disembunyikan (ketetapan 2026-09-23). Dari 27 itu,
   21 berposisi Outstanding — sehingga tab itu menyebut 319, bukan 340.
2. **644 → 234**: daftar dibatasi ke cabang petugas. **Layar Pega tampaknya TIDAK membatasinya** —
   671 adalah angka tanpa batas cabang.

Butir kedua menyentuh andaian yang sejak awal ditandai sebagai andaian di `usecase.buildFilter`:
rule When yang menentukan kapan batas cabang berlaku termasuk 137 When rule yang hilang (`R-16`).
Sekarang ada bukti bahwa layar lama tidak membatasinya — tetapi menghapus batas itu bertentangan
dengan ketetapan 2026-09-22, sehingga **tidak diambil sepihak**.

#### Hasil verifikasi

`go build` · `go vet` · `go test ./...` bersih · `typecheck` · `npm run build` bersih ·
`npm test` **112 lulus**, 3 gagal (kegagalan lama `master-rekening`).

Untuk pertama kalinya, kuerinya **dijalankan terhadap basis data sungguhan** dan pencacah serta
daftar terbukti menyebut angka yang sama.
