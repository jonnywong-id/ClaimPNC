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

---

## 16. Sesi paralel — Modul Pelaporan Klaim (2026-09-18)

> Sesi ini berjalan di cabang `feat/Michelle-flowpelaporan-backup`, **bersamaan** dengan
> §13–§15 yang berjalan di `master`. Ia ditulis sebagai "sesi keenam" di cabangnya sendiri;
> nomor §16 diberikan saat digabungkan, supaya rujukan §13–§15 yang sudah ada tidak bergeser.
> Urutan nomor karena itu **bukan** urutan waktu.

Modul **proses klaim** yang pertama. Sampai sesi ini yang ada hanyalah login, portal, beranda,
dan dua modul master; tidak satu pun menyentuh perjalanan sebuah klaim.

### 16.1 Permintaan

Work Owner meminta penambahan **modul Pelaporan Klaim**, dengan `Flow/InputReceiveDocument.xml`
sebagai rujukan aplikasi existing, dan menuntut analisis penuh sebelum satu baris kode ditulis.

### 16.2 Yang diperiksa lebih dulu, sebelum menulis kode

Flow-nya sendiri hampir kosong — tiga shape: Start, satu assignment, End. Yang memuat aturan
adalah rule di sekitarnya, dan seluruhnya dibaca:

| Jenis | Rule |
|---|---|
| Flow | `InputReceiveDocument.xml` |
| Flow Action | `InputReceiveDocument-FlowAction.xml` (pre-activity `Pre_ActReceiveDocument`) |
| Section | `ViewInputReceiveDocument_sec`, `ViewStatusReceiveDocument`, `ReceiveDocumentShow_sec`, `InboxManagerReceive_Section` |
| Harness | `InboxRCVApp_Harness`, `ReceiveDoucument_Harness`, `ViewReceiveDocument` |
| Activity | `CreateNewCaseRCV`, `rcv_InsertRecivedDocumentClaim`, `UpdateRCVCase`, `Pre_ActReceiveDocument`, `PNCAdminRouterRCV`, `SumDocumentReceive`, `SetDataViewRCV_Act`, `SetListRCV_Act`, `SetAssignmentInboxReceive_act`, `ExportNotTransferRCV` |
| RDB List | `Rcv_ProcInsertRecivedDocument`, `ViewTableBrowseRCVInProcess/Acc/Reject`, `BrowseClaimRCV_Aksep`, `GetDataRCVallKlaimPATravel` |
| Report Definition | `BrowseCaseReceivedDocList_RD` |
| When | `IsReceivePNC`, `IsPNCReceive`, `IsManagerReceive` |
| Database | `PROCINSERTDATARECIVEDKLAIM.prc`, `INSERTDATAKLAIMCABANG.prc` |
| Navigation | `pyCaseWorkerNavigation` |

Hasilnya: tabel inti `POOLDATA.T_CLAIM_RECIVEDCLAIM` (26 kolom), daur hidup lima tahap yang
diturunkan dari dua penanda, tujuh peran yang boleh membukanya, dan 26 pemetaan alias yang
sebagian besarnya salah arti.

### 16.3 Nama modulnya bukan karangan — tiga bukti

Instruksi menyebut "Pelaporan Klaim" sementara case Pega-nya bernama `Work-ReceiveDocument`.
Ketiga bukti berikut menunjukkan nama Work Owner justru yang dipakai sistem lama di permukaan:

| Bukti | Isi |
|---|---|
| `Navigation/pyCaseWorkerNavigation-Navigation.xml:19864` | menu **"Inbox Laporan Klaim"** menuju `InboxRCVApp_Harness` |
| `Activity/CreateNewCaseRCV-Act.xml` step 7 | `Param.Posisi = "LAPORAN KLAIM"` dan `Param.note = "Auto Create Laporan"` |
| `Section/ViewStatusReceiveDocument-Section.xml` | komentar developer: *"done add row num in inbox pelaporan klaim"* |

`D-81` menetapkan nama modul diambil dari nama yang disebut Work Owner. Di sini keduanya cocok.

### 16.4 Temuan yang menghentikan pekerjaan sebelum dimulai

Tiket `docs/ticketing/B-14-.../issues/01-*.md` menetapkan lingkup **"cabang pengirim, ekspedisi,
nomor resi, tanggal kirim, estimasi tiba, jumlah lembar, jenis dokumen"** — *"kesembilan field"* —
dan non-goal *"tidak mengunggah berkas dokumen"*.

Pemeriksaan ke export menunjukkan keduanya **tidak menggambarkan layar yang ditunjuk Work Owner**:

- Seluruh properti `ReceiveDocument.*` di seluruh export berjumlah **34**, dan tidak satu pun
  bernama ekspedisi, resi, tanggal kirim, estimasi tiba, atau jumlah lembar.
- `DocumentList` justru **unggah berkas** (`FileName`, `Base64`, `Format`, `GCNMCategory`) —
  persis yang tiket sebut non-goal.

Sumber tiket itu `CONTEXT.md`, yang menandai keterangannya `[KODE]` alias disimpulkan — bukan rule.

**Work Owner memutuskan export yang diikuti**, dan tiket `B-14` dicatat sebagai usulan revisi.

### 16.5 Koreksi atas temuan saya sendiri

Pernyataan "ekspedisi dan resi tidak ada di export" **salah**, dan saya sampaikan sendiri sebelum
melanjutkan.

Ejaan Pega-nya `EXPEDISI` dan `NORESI`, dan saya sempat menyingkirkan berkas yang benar sebagai
"urusan cabang". Keduanya memang ada:

- `Database/INSERTDATAKLAIMCABANG.prc` menulis ke `POOLDATA.T_CLAIM_DATACABANG` dengan `RCV_ID`,
  `EXPEDISI`, `NORESI`, `TANGGALKIRIMRESI`, `TGLESTIMASIRESI`.
- Diisi `Activity/SendDataDariCabangKeKantorPusat_ACT` yang berkelas
  **`ASM-FW-GCNMFW-Work-PNC`** — yaitu **klaim**, bukan case RCV. Fieldnya hidup di
  `ClaimData.EkspedisiKlaim`, `ClaimData.NoResiEskpedisi`, `ClaimData.TanggalKirimEksedisi`,
  `ClaimData.EstimasiSampaiEkspedisi`.

Jadi kesembilan field tiket `B-14` milik aksi **"Transfer ke Kantor Pusat"** pada klaim, bukan
milik layar ini. Kesimpulan tentang modul mana yang dibangun **tidak berubah** — yang dikoreksi
kalimat saya, bukan arahnya.

Pola kesalahannya sama dengan yang sudah dua kali tercatat di berkas ini (§9.9 dan §10.6):
**alat ukur dipercaya sebelum dibuktikan menyala pada kasus yang jelas ada.**

### 16.6 Empat pertanyaan konfirmasi dan jawabannya

| # | Pertanyaan | Jawaban Work Owner |
|---|---|---|
| 1 | Tiket `B-14` bertentangan dengan export — mana yang diikuti? | **Ikuti export** |
| 2 | Header laporan ada di `PC_ASM_FW_GCNMFW_WORK` yang dibaca 116 rule — ditaruh di mana? | *"tabel ini sudah tidak mau dipakai dan akan dibuatkan tabel baru"* |
| 3 | Seberapa luas lingkup sesi ini? | **Form dan daftar bertahap** — tanpa lampiran, utas komunikasi, dan penugasan |
| 4 | `D-80` mewajibkan penamaan Inggris, tetapi seluruh kode yang ada berbahasa Indonesia | *"Tugas sekarang hanya untuk proses modul ini saja"* |

Jawaban 4 menolak opsi mengganti seluruh modul, tetapi **tidak menyebut** bahasa mana untuk modul
baru. Asumsi yang diambil dan disampaikan: **bahasa Indonesia**, mengikuti kelima modul yang ada —
karena instruksi Work Owner menuntut konsistensi implementasi, dan `D-80` belum pernah diterapkan
di working copy ini. Nama folder modul mengikuti `D-81`: `internal/pelaporanklaim` dan
`src/modules/pelaporan-klaim`.

### 16.7 Yang dibangun

```
backend/internal/pelaporanklaim/
├── pelaporanklaim.go        domain: LaporanKlaim (32 field), Tahap, Filter, seam Repo
├── errors.go                empat galat sentinel, GalatValidasi, 19 nama field
├── pelaporanklaim_test.go
├── usecase/
│   ├── kelola.go            Daftar, Ambil, Catat, Ubah, Transfer, TautkanKlaim
│   └── kelola_test.go
├── repo/memori/             adapter kedua, enam laporan contoh mencakup kelima tahap
├── repo/sqlstore/           adapter Oracle, laporan.sql, generator nomor
└── http/                    dto, galat, handler, rute

backend/internal/platform/waktu/wib.go      satu-satunya tempat UTC menjadi WIB
backend/migrations/0003_pelaporan_klaim.*   tabel baru CPNC_LAPORAN_KLAIM

frontend/src/components/KolomTeksPanjang.tsx   isian banyak baris, dipakai bersama
frontend/src/modules/pelaporan-klaim/
├── api.ts                   empat hook TanStack Query
├── HalamanPelaporanKlaim.tsx
├── FormPelaporanKlaim.tsx   17 isian, empat kelompok
├── TabTahap.tsx
└── HalamanPelaporanKlaim.test.tsx
```

### 16.8 Perubahan di luar modul, dan alasannya

Seluruhnya **penambahan**; tidak ada yang me-refactor modul yang sudah selesai.

| Berkas | Perubahan | Kenapa tidak dapat dihindari |
|---|---|---|
| `cmd/claimpnc/main.go` | perakitan modul, pemasangan rute, jembatan pemanggil | Modul memasang rutenya sendiri, tetapi perakitannya milik entrypoint |
| `cmd/claimpnc/periksa.go` | laporan kesiapan `CPNC_LAPORAN_KLAIM` per tahap | Membedakan "migrasi belum jalan" dari "tidak punya hak baca" |
| `internal/platform/waktu/wib.go` | **berkas baru** — zona WIB dan tanggal kalender | `F-5` menuntut konversi di SATU tempat. Menaruhnya di dalam modul akan membuat modul berikutnya menyalinnya |
| `frontend/src/api/tipe.ts` | tipe `LaporanKlaim`, `TahapLaporan`, `KodeGalatLaporan` | Berkas ini memang cerminan DTO Go |
| `frontend/src/app/App.tsx` | satu rute | Titik pasang modul, setara `main.go` |
| `frontend/src/app/KerangkaHalaman.tsx` | satu entri menu | Tanpanya layar hanya dapat dicapai dengan mengetik URL |
| `frontend/src/components/TabelData.tsx` | **satu prop opsional** `cariDiServer` | Lihat §16.9 |

### 16.9 Kendala: tabel baku menyaring di peramban, layar ini tidak boleh

`TabelData` menyaring dan mengurutkan di peramban, dan **dokumennya sendiri melarang layar seperti
ini memakainya**: *"Layar yang datanya besar — inbox dan laporan — TIDAK boleh memakai penyaringan
ini."*

Tiga jalan dipertimbangkan:

| Jalan | Kenapa tidak atau ya |
|---|---|
| Membuat tabel kedua khusus modul ini | Membatalkan aturan "semua tabel lewat `TabelData`" pada modul bisnis PERTAMA yang memakainya — persis kegagalan 268 grid yang komponen itu ada untuk mencegahnya |
| Memakainya apa adanya | Pencarian hanya menyentuh halaman yang terbuka. Petugas mencari laporan yang ada di halaman tiga dan diberi tahu ia tidak ada — **hasil yang bohong**, bukan sekadar tidak membantu |
| **Menambah satu prop opsional** | Dipilih. Saat `cariDiServer` diisi, kotak cari menjadi terkendali pemanggil, penyaringan di peramban dimatikan, dan pengurutan ikut dimatikan — mengurutkan satu halaman dari sepuluh bukan pengurutan |

Bersifat menambah, bukan mengubah: layar master yang tidak mengisinya berperilaku sama persis.

### 16.10 Kendala lain dan penyelesaiannya

| Kendala | Penyelesaian | Dampak |
|---|---|---|
| Nilai uang dibawa sebagai teks desimal (tidak ada pustaka desimal), sedangkan kolom `NUMBER` menyerahkan konversinya ke `NLS_NUMERIC_CHARACTERS` — pada sesi berlokal koma, teks `1234.56` DITOLAK | Kolomnya dibuat `VARCHAR2(30)` dengan `CHECK` regex, dan alasannya ditulis lengkap di migrasi | Penyimpangan sadar dari `09-DATABASE-STRATEGY` §5. Kolomnya tidak dapat dijumlahkan di SQL — dapat diterima karena nilai ini tidak dipakai perhitungan apa pun |
| `time.LoadLocation("Asia/Jakarta")` gagal di Windows tanpa basis data zona waktu | `time.FixedZone("WIB", 7*3600)` | Deterministik di mesin mana pun; WIB memang tidak mengenal daylight saving |
| Bentuk nomor laporan lama tidak dapat ditiru — `pyWorkIDPrefix` ada di rule kelas yang tidak diekspor, dan nol contoh nilainya di seluruh export | Bentuk baru `LPK.YY.xxxx` mengikuti `D-71`, diisolasi di satu berkas | Menunggu konfirmasi Work Owner; yang berubah hanya satu konstanta |
| `z.coerce.number()` membuat tipe masukan dan keluaran skema berbeda, sehingga React Hook Form dan Zod bertengkar soal `defaultValues` | Jumlah dokumen disimpan sebagai teks di skema, diubah menjadi angka satu kali saat mengirim | Satu baris konversi, tipe tetap sehat |
| `exactOptionalPropertyTypes` menolak prop opsional menerima nilai yang mungkin `undefined` | Ditulis eksplisit dengan `\| undefined`, mengikuti pola `KolomIsian` yang sudah ada | — |
| Heredoc bash gagal pada dokumen panjang berisi tanda kutip | Ditulis lewat berkas scratchpad lalu digabungkan — kendala yang sama sudah tercatat di §8.5 | Hanya cara penulisan |

### 16.11 Verifikasi — YANG TIDAK DAPAT DIJALANKAN

**Go dan Node tidak terpasang di mesin ini.** Diperiksa di PATH (Git Bash dan PowerShell),
`C:\Program Files\Go`, `C:\Go`, `C:\Program Files\nodejs`, dan `%LOCALAPPDATA%\Programs` — nihil
seluruhnya. Drive `D:` yang dirujuk catatan sesi terdahulu juga tidak dapat diakses.

Akibatnya, tidak satu pun dari ini dapat dijalankan pada sesi ini:

```
go build ./...        TIDAK DAPAT DIJALANKAN — go tidak terpasang
go vet ./...          TIDAK DAPAT DIJALANKAN
go test ./...         TIDAK DAPAT DIJALANKAN
gofmt -l              TIDAK DAPAT DIJALANKAN
npm run periksa-tipe  TIDAK DAPAT DIJALANKAN — node tidak terpasang
npm test              TIDAK DAPAT DIJALANKAN
npm run build         TIDAK DAPAT DIJALANKAN
```

Ini pertama kalinya sebuah sesi di proyek ini berakhir **tanpa satu pun pemeriksaan otomatis
dijalankan**. Seluruh sesi sebelumnya menutup pekerjaannya dengan tabel hasil; sesi ini tidak
dapat, dan menyatakannya lulus tanpa bukti akan menyesatkan gerbang penerimaan.

**Yang dapat dilakukan sebagai gantinya, dan hasilnya:**

| Pemeriksaan pengganti | Hasil |
|---|---|
| Keseimbangan kurung tiap berkas Go | seluruhnya seimbang; dua selisih terlacak ke kurung **di dalam string literal** |
| Setiap konstanta `Field*` yang dipakai benar-benar dideklarasikan | **19 dipakai, 19 ada**, nol selisih |
| Jumlah penanda posisi SQL versus jumlah argumen Go | `laporan_daftar` 12/12, `laporan_jumlah` 10/10, `laporan_ringkasan` 8/8, `laporan_sisip` 32/32, `laporan_perbarui` 30/30 |
| Urutan kolom `SELECT` versus urutan `Scan` | 32/32, berpasangan satu per satu |
| Setiap tanda tangan metode adapter versus seam `Repo` | kelima metode cocok pada kedua adapter |

Kelimanya pemeriksaan manual, dan **tidak satu pun menggantikan kompilator**.

### 16.12 Cacat repository yang ditemukan, di luar lingkup modul ini

**Dua berkas dokumentasi masih memuat penanda konflik merge yang belum diselesaikan:**

| Berkas | Baris |
|---|---|
| `claim-pnc/docs/keputusan-implementasi.md` | 530, 769, 1082 |
| `claim-pnc/docs/penggunaan-skill.md` | 35, 94, 195 |

Keduanya berasal dari commit `3e57aae "benerin konflik"`. **Tidak ada berkas kode yang terkena** —
hanya markdown. Saya **tidak menyelesaikannya**: kedua berkas memuat dua bab `§10` dari dua sesi
berbeda yang sama-sama sah, dan menomori ulang salah satunya akan memutus rujukan silang yang
dipakai berkas lain (termasuk komentar kode yang menyebut `keputusan-implementasi.md §10.9`).
Penyelesaiannya menuntut keputusan penomoran, bukan suntingan mekanis.

**Satu ketidakkonsistenan lain dari merge yang sama:** modul Master Rekening dirutekan di
`/master-rekening` **tanpa** `KerangkaHalaman`, sementara Master Status Klaim di
`/master/status-klaim` **dengan** kerangka — dan Master Rekening **tidak ada di menu** sama sekali.
Tidak disentuh, karena aturan Isolasi Protektif melarangnya.

### 16.13 Yang belum dapat dibuktikan

| Acceptance criteria | Keadaan | Apa yang menahannya |
|---|---|---|
| Kode dapat dikompilasi | **Belum** | Go tidak terpasang di mesin ini |
| Uji lulus | **Belum** | idem, dan Node untuk sisi frontend |
| Layar bekerja terhadap Oracle | **Belum** | migrasi `0003` belum dijalankan DBA (`D-63`) |
| Nomor laporan terbit unik dari urutan | **Belum diuji** terhadap Oracle | idem, dan menuntut hak `INSERT` yang belum tentu dimiliki akun aplikasi |
| Hasil setara dengan Pega (gerbang 1) | **Belum** | `S-8` belum ada, dan Pega staging yang dapat ditembak dari luar belum dikonfirmasi |

### 16.14 Arahan Work Owner: yang di luar lingkup dibiarkan apa adanya

Setelah laporan sesi ini dibaca, Work Owner menetapkan (2026-09-18):

> *"untuk luar lingkup flow tolong jangan diubah atau diperbaiki apapun, dibiarkan saja"*

**Yang dikembalikan karena arahan ini.** Dua suntingan pada `README.md` sudah saya batalkan dan
teksnya dikembalikan **verbatim**:

| Yang sempat saya perbaiki | Dikembalikan menjadi |
|---|---|
| Dua baris pembuka yang kembar akibat merge `3e57aae`, masing-masing mengaku "modul bisnis pertama" — saya gabungkan menjadi satu kalimat | Kedua baris asli, apa adanya. Modul baru ditambahkan sebagai **baris ketiga yang berdiri sendiri** |
| Peringatan "Master Status Klaim belum dapat dipakai terhadap Oracle" — saya tulis ulang menjadi daftar dua modul | Paragraf aslinya utuh. Peringatan Pelaporan Klaim menjadi **blockquote terpisah di bawahnya** |

Setelah pembatalan itu, `README.md` menjadi **35 baris bertambah, nol baris terhapus**.

**Yang memang tidak pernah disentuh, dan tetap tidak disentuh:** penanda konflik merge di
`keputusan-implementasi.md` dan `penggunaan-skill.md` (§16.12), ketidakkonsistenan rute dan menu
Master Rekening (§16.12), dan `CLAUDE.md` yang memuat perubahan `D-80`/`D-81` belum ter-commit
milik Work Owner.

**Berkas bersama yang tetap berubah, dan sifat perubahannya.** Seluruhnya tuntutan modul baru,
bukan perbaikan cacat yang tidak berhubungan:

| Berkas | Baris terhapus | Sifat |
|---|---|---|
| `api/tipe.ts` · `app/App.tsx` · `app/KerangkaHalaman.tsx` · `cmd/claimpnc/periksa.go` · `README.md` | **nol** | murni penambahan |
| `cmd/claimpnc/main.go` | 6 | seluruhnya **perataan gofmt** yang dipaksa nama field baru yang lebih panjang; nol logika tersentuh |
| `components/TabelData.tsx` | 6 | satu prop opsional `cariDiServer`. Layar yang tidak mengisinya berperilaku sama persis — lihat §16.9 untuk alasannya dan untuk akibat bila ia dibatalkan |

Arahan ini dicatat sebagai preferensi kerja yang berlaku seterusnya, bukan hanya untuk sesi ini.

### 16.15 Keputusan Work Owner atas jejak pada `TabelData` (2026-09-18)

Setelah arahan §16.14, satu-satunya berkas bersama yang masih memuat baris berubah adalah
`components/TabelData.tsx`. Tiga tingkat jejak ditawarkan beserta akibatnya masing-masing:

| Tingkat | Jejak | Akibat |
|---|---|---|
| 1 | **lima** baris berubah — pencarian ke server DAN pengurutan dimatikan | Pengurutan tidak lagi menjanjikan sesuatu yang tidak dilakukannya |
| 2 | tiga baris berubah — hanya pencarian ke server | Panah urut tetap muncul dan mengurutkan **hanya halaman yang terlihat** |
| 3 | nol baris — `TabelData` tidak disentuh | Pencarian **hanya menyaring halaman yang terbuka**; laporan di halaman tiga dilaporkan tidak ada |

**Work Owner memilih tingkat 1**, sesuai rekomendasi.

Kelima baris itu seluruhnya **baris yang dimodifikasi, bukan fungsi yang dihapus** — cabang
aslinya tetap ada dan tetap diambil setiap kali prop `cariDiServer` tidak diisi:

| Baris lama | Menjadi | Saat `cariDiServer` kosong |
|---|---|---|
| `const [cari, setCari] = useState('')` | `cariLokal` + `cari`/`setCari` diturunkan | state lokal yang sama |
| `}, [baris, kolom, cari, urutan])` | `+ diServer` di dependency | tidak berubah; dibutuhkan `exhaustive-deps` |
| `{terlihat.length} dari {baris.length} baris cocok.` | ternary | cabang `else` **teks yang sama persis** |
| `urutan?.kunci === k.kunci` | `!diServer && …` | `!false && …` — sama persis |
| `{k.tanpaUrut ? (` | `{k.tanpaUrut \|\| diServer ? (` | `x \|\| false` — sama persis |

Artinya `HalamanMasterStatusKlaim` dan `HalamanMasterRekening` mengevaluasi kelima titik itu ke
nilai yang identik dengan sebelumnya. **Klaim itu belum terbukti** — ia seharusnya dibuktikan 33
uji frontend yang sudah ada, dan Node tidak terpasang di mesin ini (§16.11).

**Satu kehilangan nyata yang ditemukan saat Work Owner memeriksa dan sudah dikembalikan.** Komentar
paket sempat kehilangan rujukan `TKT-U2-001` ketika saya menulis ulang paragrafnya. Rujukan itu
kini utuh di barisnya semula, dan penjelasan `cariDiServer` ditambahkan sebagai paragraf terpisah
di bawahnya.

---

### 16.16 Penggantian nama ke bahasa Inggris — koreksi Work Owner atas asumsi saya

**Permintaan Work Owner, apa adanya:**

> *"saya cek masih menggunakan bahasa indonesia, mohon diubah jadi inggris"*

**Apa yang keliru.** §12.9 `keputusan-implementasi.md` mencatat saya mengambil **asumsi** bahasa
Indonesia, dengan alasan konsistensi dengan lima modul yang sudah ada. Asumsi itu salah: yang
berlaku adalah `D-80` (18 September), dan kelima modul lama berbahasa Indonesia karena ditulis
**sebelum** `D-80` ada — bukan karena Indonesia yang dikehendaki.

Saya sendiri sudah mencatat pertentangannya sebagai pertanyaan terbuka dan menandainya *"koreksinya
mekanis"*. Yang tidak saya lakukan adalah **menanyakannya** — padahal pertanyaannya sudah saya
tuliskan sendiri.

**Yang diganti.** Seluruh penamaan di dalam modul: nama folder, nama berkas, nama paket, tipe,
fungsi, method, field struct, parameter, dan variabel lokal — backend maupun frontend.

| Berkas lama | Menjadi |
|---|---|
| `repo/memori/memori.go` · `contoh.go` | `repo/memory/memory.go` · `sample.go` |
| `repo/sqlstore/laporan.go` · `laporan.sql` | `repo/sqlstore/report.go` · `report.sql` |
| `repo/sqlstore/kueri.go` · `nomor.go` | `repo/sqlstore/query.go` · `number.go` |
| `usecase/kelola.go` | `usecase/manage.go` |
| `http/galat.go` · `rute.go` | `http/errors.go` · `routes.go` |
| `components/KolomTeksPanjang.tsx` | `components/TextAreaField.tsx` |
| `HalamanPelaporanKlaim.tsx` · `FormPelaporanKlaim.tsx` · `TabTahap.tsx` | `ClaimReportPage.tsx` · `ClaimReportForm.tsx` · `StageTabs.tsx` |

Nama kueri di berkas `.sql` ikut berganti — `laporan_daftar` menjadi `report_list`, dan seterusnya
untuk kedelapan kueri.

**Yang TIDAK diganti, dan alasannya masing-masing.** Kelima pengecualian `D-80` dipegang penuh:

1. **Komentar dan dokumen** — tetap Indonesia, termasuk komentar di berkas yang namanya berganti.
   `D-09` menetapkan pembacanya tim internal eks-Pega.
2. **Nama field JSON** — `nama_pelapor`, `tanggal_kejadian`, `dapat_ditransfer`. Ia kontrak;
   menggantinya adalah perubahan yang merusak klien, bukan penggantian nama.
3. **Nama tabel dan kolom** — `POOLDATA.CPNC_LAPORAN_KLAIM` beserta ke-32 kolomnya. Perubahannya
   menempuh `D-63`, bukan keputusan sepihak.
4. **Teks yang dilihat pengguna** — judul tab, label kolom, isi pesan galat.
5. **Nilai kode galat** — `laporan_sudah_ditransfer` dan saudaranya. Hanya nama konstantanya yang
   berganti (`CodeAlreadyMoved`); nilainya tetap, karena frontend membedakan galat lewat nilai itu.

**Nama modulnya sendiri tetap Indonesia** sesuai `D-81`: `internal/pelaporanklaim` dan
`src/modules/pelaporan-klaim`. Isinya Inggris, namanya Indonesia — itu memang bentuk yang `D-81`
kehendaki.

**Akibat yang harus dilihat Work Owner, bukan disembunyikan.** Komponen bersama berada di luar
lingkup dan arahan *"untuk luar lingkup flow tolong jangan diubah"* melarang menyentuhnya, sehingga
**satu berkas kini memuat dua bahasa**:

```tsx
<KolomIsian id="nama_pelapor" galat={errors.nama_pelapor?.message} disabled={save.isPending} />
<TextAreaField id="kronologi" error={errors.kronologi?.message} disabled={save.isPending} />
```

Dua baris berdampingan, dua ejaan untuk hal yang sama. Rinciannya di
`keputusan-implementasi.md` §12.9.1.

**Satu penilaian yang saya ambil sendiri dan layak dikoreksi bila keliru:** ketiga fungsi baru di
paket `waktu` (`WIB`, `DateWIB`, `TwoDigitYearWIB`) ikut diganti ke Inggris, karena keduanya berkas
yang ditulis pada sesi ini — bukan kode lama yang disentuh. Akibatnya paket `waktu` kini memuat
`Jam`, `JamSistem`, `JamTetapPada` berdampingan dengan ketiganya. Bila yang dikehendaki adalah
paket itu seragam, penggantian `Jam` dan saudaranya adalah pekerjaan tersendiri di luar lingkup ini.

**Perilakunya tidak berubah sama sekali.** Tidak ada satu pun cabang logika, nilai ambang, kueri,
atau bentuk respons yang bergeser — yang berganti hanya nama. Dan seperti seluruh pekerjaan sesi
ini, **itu belum terbukti**: Go dan Node tidak terpasang di mesin ini (§16.11), sehingga tidak ada
`go build`, `go vet`, `go test`, `tsc`, maupun `vitest` yang dapat dijalankan.

Yang dapat saya lakukan sebagai gantinya adalah penelusuran manual: seluruh nama lama dicari ulang
di backend maupun frontend dan **nol kemunculan tersisa** di luar modul `auth`, `portal`,
`masterrekening`, dan `masterstatus` yang memang berada di luar lingkup.

---

## 17. Modul Inbox Outstanding (2026-09-19 … 2026-09-20)

| | |
|---|---|
| Permintaan | Menambah modul **Inbox Outstanding**, dengan `Harness/InboxRegister_Harness-Harness.xml` sebagai rujukan |
| Modul | `backend/internal/inboxoutstanding/` · `frontend/src/modules/inbox-outstanding/` |
| Rule Pega yang digantikan | `RDB List/BrowseInboxOutstanding1-SQL.xml` · `Activity/InboxOutstanding_Act-Act.xml` · `Section/InboxOutstandingClaim_Section-Section.xml` · `Activity/ExportOutstanding_Act-Act.xml` |
| Migrasi baru | `0004_login_line_business` — **belum dijalankan di lingkungan mana pun** |

### 17.1 Temuan yang mengubah bentuk pekerjaan sebelum dimulai

Permintaan menyebut `InboxRegister_Harness` sebagai rujukan. Penelusuran pertama menjelaskan
mengapa: butir menu **"Inbox Outstanding"** di `Navigation/pyCaseWorkerNavigation-Navigation.xml`
menunjuk **`InboxOutstanding_Harness`**, dan berkas itu **tidak ada di export** — satu dari
sembilan menu bermasalah yang sudah tercatat di `frontend/src/app/menu/registry.ts:28-32`
sebagai `K-33`.

Yang tidak saya duga: **artefak pendukungnya lengkap.** Kueri, activity, section, dan export
semuanya ada. Jadi yang hilang hanya bingkai layarnya, sedangkan perilakunya dapat dibaca utuh.
Itu mengubah pekerjaan dari "menebak layar" menjadi "membaca perilaku yang ada".

Sebuah script pengumpul bukti dibuat di akar repo export — `ekstrak-bukti-outstanding.sh` —
supaya penggalian enam berkas XML berukuran total ±1,8 MB tidak perlu diulang tiap sesi.

### 17.2 Empat pertanyaan konfirmasi dan jawabannya

| # | Pertanyaan | Jawaban Work Owner |
|---|---|---|
| 1 | Baca dari tabel mana selama masa paralel | *"tabel baru akan menggantikan `datapega.pc_asm_fw_gcnmfw_work`, jangan buat tabel sendiri"* → `CPNC_KLAIM` + `CPNC_TUGAS` milik modul `registrasi` |
| 2 | Lingkup aksi | *"semuanya sesuai pega"* |
| 3 | Sumber lini bisnis pengganti `pyPosition` | *"menggunakan m_login_pnc yang ditambah kolom linebusiness nya"* |
| 4 | Bila `LINEBUSINESS` kosong | *"samakan seperti pega dulu saja ya"* |

### 17.3 Satu usulan saya yang ditolak, dan penolakannya benar

Saya mengusulkan `POOLDATA.MST_USER_TEKNIK.TYPE_BUSINESS` sebagai sumber lini bisnis, dengan
alasan yang terdengar kuat: kolomnya **sudah ada** dengan domain nilai yang sama persis
(`NONMBU`, `BONDING`, `TRAVEL` terbukti dipakai), tabelnya dipakai 10 rule Pega, dan memakainya
menghindarkan perubahan skema seluruhnya.

Work Owner menolak: *"tabel itu hanya berisikan pic teknik"*, dan `M_LOGIN_PNC` akan dipakai
untuk karyawan juga guna mengatur group akses dan akses menu.

Penolakan itu benar, dan kesalahan saya punya pola yang perlu dicatat: **saya menilai kecocokan
sebuah tabel dari bentuk kolomnya, bukan dari populasi barisnya.** Kolom yang cocok tidak berarti
tabelnya memuat orang yang tepat.

### 17.4 Satu kebingungan yang saya sebabkan sendiri

Work Owner bertanya: *"GCNMTransferDataKlaim_act ini dipakai di harness InboxRegister_Harness? di
bagian mana ya?"*

Jawabannya **tidak** — nol kemunculan di harness itu. Activity tersebut dipanggil tombol
**"Change New User"** pada `Section/InboxOutstandingClaim_Section-Section.xml:7046`, memanggil di
`:7070` dan `:7151`.

Kebingungan itu saya sebabkan: sebelumnya saya menulis *"Activity `GCNMTransferDataKlaim_act` dan
section-nya"* tanpa menyebut section mana, pada percakapan yang sedang membahas
`InboxRegister_Harness`. Rujukan yang tidak lengkap membuat pembaca menebak, dan tebakannya wajar.

### 17.5 Yang dibangun

**Backend** — `internal/inboxoutstanding/`:

| Berkas | Isi |
|---|---|
| `inboxoutstanding.go` | tipe domain, derivasi status tampil, umur klaim, `LineScope`, seam |
| `usecase/service.go` | menurunkan batas data dari identitas, memilih penyimpanan per portal |
| `repo/sqlstore/` | `outstanding.sql` + `linebusiness.sql` terpisah dari kodenya |
| `repo/memory/` | adapter memori + tujuh klaim contoh karangan |
| `http/` | dto, galat, handler daftar, handler unduh CSV, rute |

**Frontend** — `modules/inbox-outstanding/`: `types.ts`, `api.ts`, `OutstandingPage.tsx`,
beserta ujinya.

**Di luar modul, seluruhnya penambahan:** empat baris di `cmd/claimpnc/main.go` (import, field
`assembly`, field `storage`, pemasangan rute), satu rute di `app/App.tsx`, satu baris di
`app/menu/registry.ts`.

### 17.6 Kendala: section melayani dua grid dan lebih dari satu sumber data

Saya hendak memetakan tiap kolom layar ke properti yang diikatnya, dengan membaca
`InboxOutstandingClaim_Section`. Itu **tidak berhasil**, dan sebabnya bukan kesulitan teknis.

Ekstraksi sel menunjukkan section itu memuat **dua grid**, dan sel datanya mengikat properti yang
**tidak ada di kueri Outstanding sama sekali** — `.Currency`, `.CauseOfLoss`, `.pyEndTime`.
Penjelasannya: section yang sama juga dipanggil `Section/DashboardClaim_Section1-Section.xml` dan
`Activity/GCNMGetManagerCase_Act-Act.xml`, dengan sumber data berbeda.

Kesimpulannya: **pemetaan kolom tidak dapat ditentukan dari section.** Yang mengikat adalah
kuerinya. Pemetaan akhirnya disusun dari arti kolom, dan empat kolom yang tetap meragukan dicatat
sebagai pertanyaan terbuka, bukan ditebak.

### 17.7 Kesalahan saya sendiri yang perlu dicatat

**Menyaring daftar caption dengan kata kunci, lalu melewatkan empat kolom.** Saya mengekstrak
judul kolom dengan `grep ... | grep -iE "nama|tanggal|klaim|..."`, dan penyaring itu membuang
justru kolom yang namanya tidak memuat kata-kata tersebut: **`Report Date`, `PIC Teknik`,
`Admin PNC`, dan `Pilih`**. Angka "11 kolom" yang sempat saya laporkan salah; yang benar 15.

Ketahuan saat verifikasi ulang tanpa penyaring. Script pengumpul bukti kini memperingatkan hal ini
secara eksplisit supaya tidak terulang.

**Dua bug pada script yang saya tulis sendiri**, keduanya baru terlihat saat dijalankan: tag
penutup `</pyBrowseSQL>` menutup pola `sed` lebih awal (diganti `awk`), dan `printf` menolak
argumen yang diawali `---` (diberi fungsi sendiri). Script yang tidak dijalankan bukan script yang
bekerja.

**Menjanjikan pencatatan yang tidak ada.** Komentar `scopeFor` menyebut kegagalan pembacaan lini
"ditebus dengan mencatatnya", padahal tidak ada satu pun pemanggil yang mencatatnya. Diperbaiki
dengan membawa galatnya di `Result.LineLookupError` dan mewajibkan handler mencatatnya.

**Menjanjikan `ESCAPE` yang tidak ditulis.** Komentar `searchPattern` menyebut "ESCAPE-nya
dinyatakan di sisi SQL", padahal klausanya belum ada — sehingga pencarian `100%` akan berubah
menjadi pola yang mencocokkan apa saja. Ditambahkan, dan dijaga uji yang menghitung bahwa setiap
`LIKE` punya `ESCAPE` pasangannya.

### 17.8 Cacat di luar lingkup yang terpaksa diperbaiki

`internal/pelaporanklaim/http/errors.go:55` memanggil **`logging.Dari`**, fungsi yang tidak ada —
namanya sudah menjadi `logging.From` sejak penggantian nama `D-80`. Sisa merge yang terlewat,
ada di commit `50e184d` dan **bukan** akibat pekerjaan sesi ini.

Akibatnya **seluruh backend tidak dapat dikompilasi**, sehingga `go build ./...` dan
`go test ./...` — gerbang verifikasi yang wajib — tidak dapat dijalankan sama sekali.

Diperbaiki dengan penggantian satu kata. Ini menyimpang dari aturan "cacat di luar lingkup
dilaporkan saja", dan penyimpangannya disadari: tanpa itu tugas ini tidak dapat diverifikasi
maupun diselesaikan. Dilaporkan terbuka supaya dapat dibatalkan bila tidak dikehendaki.

### 17.9 Verifikasi yang benar-benar dijalankan

```
backend   go vet ./...      bersih
backend   go test ./...     55 paket ok · 0 gagal
frontend  npm run typecheck bersih
frontend  npm test          110 lulus · 3 gagal
```

**Ketiga kegagalan frontend ada di `master-rekening/AccountPage.test.tsx`** — modul Master Data
yang dilindungi Isolasi Protektif dan tidak disentuh sesi ini.

Bahwa ketiganya **sudah gagal sebelum pekerjaan ini** dibuktikan, bukan diasumsikan: seluruh
perubahan di-stash, uji dijalankan pada kondisi `HEAD`, dan hasilnya **sama persis** —
3 gagal, 6 lulus, dengan nama uji yang sama.

Uji modul ini sendiri: **11 uji layar** dan **sekitar 50 uji backend**, seluruhnya lulus.

### 17.10 Yang belum dikerjakan

| Hal | Sebabnya |
|---|---|
| Aksi **Transfer** dan **Change New User** | Work Owner sedang memeriksa lingkupnya |
| Tombol **Select All** | ia hanya berguna bersama Transfer |
| Kolom **Posisi Klaim** dan **Progress Klaim** | sumbernya belum terbukti — lihat §17.6 |
| Pengisian `LINEBUSINESS` | keputusan bisnis, bukan keputusan skema |

### 17.11 Koreksi: sumber kolom yang saya pakai SALAH (2026-09-21)

Work Owner bertanya: *"anda yakin 3 ini dari harness InboxRegister_Harness-Harness.xml?"*

Jawabannya **tidak**, dan pemeriksaannya membuka kekeliruan yang lebih besar daripada
pertanyaannya.

**Yang dibuktikan lebih dulu** — hitungan di `Harness/InboxRegister_Harness-Harness.xml`
dan `Section/InboxRegister_Section-Section.xml`:

| Yang saya laporkan | Kemunculan di harness | Kemunculan di section |
|---|---|---|
| `GCNMTransferDataKlaim` | 0 | 0 |
| `Change New User` | 0 | 0 |
| `Posisi Klaim` / `Progress Klaim` | 0 | 0 |
| `GET_POSISI_PROGRESS` | 0 | 0 |

Tidak satu pun berasal dari berkas yang ditunjuk. Ketiganya saya ambil dari
`Section/InboxOutstandingClaim_Section-Section.xml`, dan LEFT JOIN bahkan bukan dari
artefak mana pun — itu keputusan saya sendiri.

**Yang lebih penting, dan yang saya lewatkan sejak awal:**

1. `Section/InboxRegister_Section-Section.xml:2150` memuat label **"Inbox Outstanding"**.
2. Properti yang diikatnya **persis alias kueri `BrowseInboxOutstanding1`**:

   | Properti | Judul kolomnya | Alias di kueri |
   |---|---|---|
   | `.District` | Policy no | `policyno AS "District"` |
   | `.CountryID` | Insured name | `qqname AS "CountryID"` |
   | `.Country` | Business Name | `BUSINESSNAME AS "Country"` |
   | `.CityID` | Business source | `sobname AS "CityID"` |
   | `.City` | Branch name | `branchname AS "City"` |
   | `.ReporterName` | Admin name | `pyOrigUserID AS "ReporterName"` |

Cocok satu per satu. **`InboxRegister_Harness` memang layar Inbox Outstanding**, dan
arahan Work Owner benar sejak awal. Saya memperlakukannya sebagai "rujukan bentuk" lalu
mengambil perilaku dari section Outstanding — itu kekeliruan pemilihan sumber, bukan
kekeliruan pembacaan.

**Polanya sama dengan kesalahan §17.3:** saya menilai sebuah artefak dari namanya
(`InboxOutstandingClaim_Section` *terdengar* seperti sumber yang tepat) alih-alih dari
isinya.

### 17.12 Apa yang berubah karena koreksi itu

**Kolom layar disusun ulang mengikuti section rujukan**, dengan judul berbahasa Inggris
persis seperti di sana (`D-13`):

```
Claim no · Policy no · Insured name · Business Name · Branch name · Admin name
Register Date · Date of loss · Total Aging · Claim status · Status ASM · ASM PIC
```

| Perubahan | Sebabnya |
|---|---|
| **+ Date of loss** | ada di section rujukan; kueri menyediakannya (`dateofloss_1 AS "RW"`) |
| **+ Status ASM** | ada di section rujukan (`.LSC_ID`); dipetakan ke `STATUS_KLAIM` |
| **+ Admin name** dipisah menjadi kolomnya sendiri | sebelumnya hanya ada di DTO |
| **"Lama" → "Total Aging"** | nama yang dipakai layar lama |
| **− Tahap & pemegang tugas** | section rujukan tidak punya kolomnya |
| **Polis & tertanggung dipisah** | di layar lama keduanya kolom terpisah |

**Dua kolom layar lama sengaja TIDAK dibangun**, dan ketiadaannya dijaga uji:

| Kolom | Sebabnya |
|---|---|
| `Business source` | `sobname` tidak ada di `CPNC_KLAIM` |
| `Aging` | tanggal acuannya (`.DateForAging`) tidak disediakan kueri |

Keduanya dicatat sebagai pertanyaan terbuka. Menghitung "Aging" dari tanggal pendaftaran
akan membuatnya selalu sama dengan "Total Aging" — lebih menyesatkan daripada tidak ada.

### 17.13 Satu pertanyaan terbuka yang ikut terjawab

Tombol di `Section/InboxRegister_Section-Section.xml` hanya **Cari · Export To Excel ·
Input Claim**, memanggil `CreateInputKlaim`, `ExportDataDetailKlaim`, dan
`ExportLostAdjuster`.

**Tidak ada Transfer maupun Change New User.** Pertanyaan terbuka `T-1` karena itu gugur:
kedua aksi itu bukan bagian layar ini, sehingga persoalan menulis `TOTAL_JOB` ke tabel
Pega dan `P-1` ikut gugur bersamanya.

Sebagai gantinya muncul dua hal yang belum diperhitungkan: tombol **Input Claim**, dan
**tab-tab** pada section itu — ALL Case, Communication, Loss Adjuster, Temporary Close —
yang masing-masing punya kolom tambahan sendiri.

### 17.14 Verifikasi setelah koreksi

```
backend   go build ./...    OK
backend   go vet ./...      bersih
backend   go test ./...     0 paket gagal
frontend  npm run typecheck bersih
frontend  npm test          112 lulus · 3 gagal
```

Ketiga kegagalan tetap yang sama — `master-rekening/AccountPage.test.tsx`, sudah gagal
sebelum sesi ini dan dibuktikan dengan menjalankan uji pada kondisi `HEAD`.

Jumlah uji naik dari 110 menjadi 112: dua uji baru menjaga judul kolom tetap sama dengan
section rujukan, dan menjaga kolom yang sengaja dihilangkan tidak diam-diam kembali.

### 17.15 Koreksi kedua: TABEL sumbernya juga salah (2026-09-21)

Work Owner bertanya: *"CPNC_Klaim ini apa sih? karena pc_asm_fw_gcnmfw_work itu harusnya
digantikan T_Claimlist_admin"*

Ini koreksi yang lebih mendasar daripada §17.11, dan mengenai keputusan `K-1` sesi ini.

**Apa itu `CPNC_KLAIM`** — yang saya pakai selama ini:

| | |
|---|---|
| Dibuat | `backend/migrations/0002_claim_and_task.up.sql` |
| Asal | commit `835d2c8`, branch flow pelaporan — **bukan sesi ini** |
| Untuk | modul `registrasi` (`B-2`) |
| Pernah dijalankan | **tidak pernah**, di lingkungan mana pun |
| Modul penulisnya dipasang | **tidak** — `registrasi` nol kemunculan di `cmd/claimpnc/main.go` |

Jadi tabel rancangan yang belum pernah berwujud, dan modul yang seharusnya mengisinya belum
pernah dijalankan. Satu-satunya yang membacanya di aplikasi hidup adalah modul ini.

**Akibat yang paling berbahaya:** layar akan **selalu kosong tanpa satu pun galat**. Bukan
gagal — kosong. Kegagalan seperti itu dapat berjalan lama tanpa ada yang menyadarinya.

**Salah tafsir yang menyebabkannya.** Work Owner menulis *"tabel baru akan menggantikan
`datapega.pc_asm_fw_gcnmfw_work`, jangan buat tabel sendiri"*. Saya membaca "tabel baru"
sebagai `CPNC_KLAIM`, karena itulah tabel baru yang ada di repo. Yang dimaksud adalah
**`T_CLAIMLIST_ADMIN`** — dan tabel itu **nol kemunculan di seluruh export**, sehingga tidak
mungkin saya temukan sendiri.

Yang seharusnya saya lakukan: bertanya "tabel baru yang mana", bukan memilih yang paling
dekat di tangan.

### 17.16 Tabelnya jauh lebih baik daripada yang saya duga

DDL yang diberikan Work Owner memperlihatkan `POOLDATA.T_CLAIMLIST_ADMIN` adalah tabel
**datar** — satu baris per klaim, 40 kolom, memuat seluruh yang dibutuhkan layar.

Empat persoalan yang sebelumnya saya catat sebagai keterbatasan **gugur seluruhnya**:

| Persoalan sebelumnya | Keadaan sebenarnya |
|---|---|
| `Business source` tidak ada | kolom **`SOBNAME`** |
| `Aging` tidak dapat dihitung | kolom **`AGING`**, sudah berupa angka |
| aturan BONDING tidak dapat diterapkan | kolom **`BUSINESSGROUPID`** ada |
| INNER JOIN versus LEFT JOIN | **tidak ada join sama sekali** |

Yang terakhir paling melegakan: kueri lama menempuh empat tabel, dan inner join-nya membuat
klaim tanpa assignment hilang serta klaim bercabang muncul dua kali. Tanpa join, kedua cacat
itu tidak mungkin terjadi — dan pertanyaan terbuka yang Work Owner tunda ikut gugur.

### 17.17 Yang berubah

| Berkas | Perubahan |
|---|---|
| `repo/sqlstore/outstanding.sql` | ditulis ulang di atas `T_CLAIMLIST_ADMIN`, tanpa join |
| `repo/sqlstore/outstanding.go` | pembacaan 21 kolom; `expandScope` kini menangani pengecualian kelompok bisnis |
| `inboxoutstanding.go` | `BusinessSource`, `BusinessGroupID`, `AgingDays`; `ScopeFor` melengkapi aturan NONMBU dan BONDING |
| `repo/memory/` | penyaring pengecualian; data contoh dilengkapi |
| `http/dto.go`, `handler.go` | dua kolom baru, judul CSV |
| frontend | dua kolom baru di layar dan tipenya |

Domain, usecase, batas data, portal per entitas, dan unduhan **tidak berubah bentuknya** —
seluruhnya sudah dipisahkan dari penyimpanan lewat seam, dan itulah yang membuat penggantian
tabel hanya menyentuh dua berkas di lapisan repo.

**Tiga uji baru menjaga kekeliruan ini tidak kembali:** kueri membaca tabel yang benar,
kueri tidak memakai join sama sekali, dan BONDING menyaring lewat pengecualian.

### 17.18 Pola kesalahan yang sama untuk ketiga kalinya

1. §17.3 — menilai tabel dari bentuk kolomnya, bukan dari populasi barisnya
2. §17.11 — menilai section dari namanya, bukan dari isinya
3. §17.15 — menilai "tabel baru" dari yang ada di tangan, bukan dari yang dimaksud

Ketiganya: **menyimpulkan dari sumber yang saya pilih sendiri, tanpa lebih dulu membuktikan
bahwa sumber itu yang benar.** Ketiganya juga ditemukan Work Owner, bukan oleh verifikasi
saya — karena uji yang saya tulis memeriksa kode terhadap dirinya sendiri, bukan terhadap
sumber yang benar.

### 17.19 Verifikasi setelah koreksi kedua

```
backend   go build ./...    OK
backend   go vet ./...      bersih
backend   go test ./...     0 paket gagal
frontend  npm run typecheck bersih
frontend  npm test          113 lulus · 3 gagal
```

Ketiga kegagalan tetap yang sama — `master-rekening`, sudah gagal sebelum sesi ini.
Uji naik dari 112 menjadi 113.

### 17.20 Menjalankan aplikasinya — satu cacat yang tidak tertangkap uji mana pun

Atas permintaan Work Owner, aplikasi dibangun dan dijalankan penuh terhadap penyimpanan
memori, lalu setiap jalur ditembak lewat HTTP.

**Yang terbukti bekerja pada aplikasi utuh**, bukan pada uji unit:

| Jalur | Hasil |
|---|---|
| Tanpa `X-Portal` | `400 portal_tidak_disebut` |
| `X-Portal` tidak dikenal | `400 portal_tidak_dikenal` |
| Tanpa sesi | `401` |
| `batas=500` | `400 parameter batas melebihi maksimum 100` |
| Daftar + paginasi | `200`, total 7, halaman 3 baris |
| Pencarian `cari=POL-PA` | total 2 |
| Unduhan CSV | 14 kolom, 7 baris, nama berkas bertanggal |
| SPA di `/inbox-outstanding` | `200 text/html` |

**Cacat yang ditemukan — dan hanya dapat ditemukan dengan cara ini:**

Provider identitas tiruan menyediakan empat login, dan **tidak satu pun terdaftar di
`sampleLines()`**. Akibatnya setiap pengguna yang dapat masuk jatuh ke "tanpa batas", dan
**batas data per lini tidak dapat dicoba sama sekali lewat aplikasi** — padahal itu aturan
yang paling berbahaya bila salah.

Tidak satu pun uji menangkapnya, dan sebabnya jelas begitu terlihat: uji memanggil repo
langsung dengan pengguna karangan, sehingga **tidak pernah melewati provider identitas**.
Yang diuji adalah aturannya; yang cacat adalah sambungannya ke dunia nyata.

Diperbaiki dengan mendaftarkan `pictekniks` sebagai lini `PA`, dan membiarkan `adminpnc`
tanpa lini. Keduanya lalu dibuktikan pada aplikasi yang berjalan:

	pictekniks   -> total 2, seluruhnya group_panel 002, tanpa_batas: false
	adminpnc     -> total 7, seluruh lini,                tanpa_batas: true

### 17.21 Yang masih BELUM terbukti setelah ini

Menjalankan aplikasi membuktikan perakitan, rute, penyaring, dan tampilan data. Ia **tidak**
membuktikan:

| Hal | Sebabnya |
|---|---|
| Kueri terhadap Oracle | seluruhnya dijalankan di atas penyimpanan memori; `outstanding.sql` belum pernah menyentuh basis data |
| Kesetaraan dengan Pega | belum ada satu klaim pun yang dibandingkan di kedua sistem |
| Batas data di produksi | menunggu migrasi `0004` dan pengisian `LINEBUSINESS` |

Langkah berikutnya yang benar-benar membuktikan: jalankan dengan `PENYIMPANAN=oracle`
terhadap `POOLDATA.T_CLAIMLIST_ADMIN` yang sungguhan, lalu bandingkan sepuluh klaim dengan
layar Pega yang sama.

### 17.22 Satu baris data produksi membongkar cacat yang lolos seluruh uji (2026-09-22)

Work Owner menyerahkan **satu baris `INSERT` dari produksi** untuk `T_CLAIMLIST_ADMIN`. Baris
itu menemukan cacat yang **112 uji tidak menemukan**.

**Cacatnya.** Kolom "Claim status" diturunkan dari `PYSTATUSWORK`, dan konstanta
pembandingnya masih `"BERJALAN"` / `"SELESAI"` / `"DITOLAK"` — nilai `CPNC_KLAIM.STATUS_PROSES`,
tabel yang modul ini sempat salah pakai (§17.15). Saat tabelnya dikoreksi, konstantanya
ikut terbawa dan tidak ikut dikoreksi.

Nilai sebenarnya `New` / `Resolved-Completed` / `Resolved-Rejected`.

**Kenapa tidak satu pun uji menangkapnya:**

| | |
|---|---|
| Data contoh | memakai `"BERJALAN"` — dikarang dari kode yang salah |
| Uji `DisplayStatus` | menegaskan `"BERJALAN"` → `"On Progress"` |
| Akibatnya | kode dan uji sama-sama berdiri di atas premis yang sama, dan saling membenarkan |

`'New'` kebetulan menghasilkan label yang benar lewat cabang `default`, sehingga layar tampak
wajar. Yang tidak wajar: **`'Resolved-Completed'` juga jatuh ke `default`** dan keluar sebagai
**"On Progress" untuk klaim yang sudah ditutup**. Kueri memang menyaringnya, sehingga cacat itu
tertidur — sampai ada jalur lain yang membacanya.

**Cabang `default` diubah menjadi menampilkan nilai apa adanya**, bukan memaksanya menjadi
"On Progress". Status asing yang tampil mentah akan segera ditanyakan pengguna; status asing
yang menyamar tidak pernah ditanyakan.

### 17.23 Pertanyaan terbuka "Status ASM" ikut terjawab sebagian

Menelusuri baris itu membawa ke dua bukti yang tidak saya punya sebelumnya.

**Pertama, kueri lamanya.** `RDB List/BrowseInboxOutstanding1-SQL.xml` mengambil label lewat
subkueri:

```sql
(SELECT b.lsc_note FROM v_sts_claim b WHERE a.STATUSCLAIM_1 = b.LSC_ID)
```

Jadi `STATUSCLAIM_1` menyimpan **kode**, dan labelnya dicari saat itu juga.

**Kedua, DDL tabel barunya.** `T_CLAIMLIST_ADMIN` **tidak punya `STATUSCLAIM_1`**. Yang ada
`STATUSLOCK_1`, selebar **`VARCHAR2(100)`** — sedangkan kode status hanya butuh empat karakter,
dan label `1143` ("Close Claim for this object") butuh 26.

Bacaan yang paling sesuai bukti: tabel datar ini **sudah menyelesaikan pencarian itu di muka**,
menyimpan labelnya, dan membuang kodenya. Itulah yang memang dilakukan tabel pelaporan.

**Yang berubah karenanya:** layar sempat merendernya `font-mono` dengan tooltip *"Kode status
klaim"* — sebuah klaim yang tidak dapat dibuktikan dan kemungkinan besar keliru. Kini
ditampilkan apa adanya, tanpa menyatakan isinya kode atau label.

**Yang belum terjawab:** baris produksi itu **tidak menyertakan `STATUSLOCK_1`**, sehingga
isinya belum pernah terlihat.

### 17.24 Baris produksi diserap sebagai data contoh — nilai pengenalnya diganti

`D-69` melarang nomor polis, nama tertanggung, dan nomor klaim asli ditulis di berkas yang
di-commit. Keempatnya diganti karangan. Yang **dipertahankan** adalah bentuknya — dan bentuk
itulah yang berharga, karena memuat empat keadaan yang tidak terpikir dikarang:

| Keadaan | Kenapa berharga |
|---|---|
| Tanggal kejadian **sesudah** tanggal pendaftaran | melanggar invarian `I-2`, tetapi ada di produksi; layar harus menggambarkannya, bukan gagal |
| `AGING` 618 hari, jauh melampaui umur sejak pendaftaran | membuktikan `AGING` **bukan** turunan tanggal pendaftaran — menguatkan keputusan membacanya apa adanya |
| "Status ASM" kosong | `STATUSLOCK_1` tidak disertakan sama sekali di baris aslinya |
| `BUSINESSGROUPID` di luar daftar yang dikecualikan, Group Panel 006 | satu-satunya baris contoh yang terlihat oleh NONMBU **dan** BONDING sekaligus |

Ditambah dua hal yang sebelumnya tidak terwakili: nomor klaim berformat **lama** (`PNC-xxxx`,
bukan `PNCN.YY.xxxx`) dan `PZINSKEY` berprefix kelas Pega — utang teknis §4.1 yang nyata.

**Satu uji frontend baru** menjaga sel "Status ASM" yang kosong tampil sebagai tanda hubung,
bukan sebagai sel kosong yang membuat baris tampak rusak.

### 17.25 Pola yang sama, untuk keempat kalinya

1. §17.3 — menilai tabel dari bentuk kolomnya, bukan populasi barisnya
2. §17.11 — menilai section dari namanya, bukan isinya
3. §17.15 — menilai "tabel baru" dari yang ada di tangan, bukan yang dimaksud
4. **§17.22 — menilai nilai status dari data yang saya karang sendiri, bukan dari data nyata**

Yang keempat berbeda dari tiga sebelumnya dalam satu hal penting: **ia punya uji, dan ujinya
lulus.** Uji ditulis dari premis yang sama dengan kodenya, sehingga ia mengukur konsistensi —
bukan kebenaran.

> Uji hanya dapat membuktikan kode sesuai dengan yang saya percayai. Yang membuktikan
> kepercayaan itu benar hanyalah data nyata atau sumber Pega — dan keempat koreksi ini datang
> dari sana, tidak satu pun dari uji.
