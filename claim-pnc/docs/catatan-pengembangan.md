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

## 16. Sesi kesembilan — Master Status Progres 2 dihidupkan (2026-09-19)

Rujukan utama: `Harness/StatusProgress2-Harness.xml`.

### 16.1 Yang ditemukan sebelum satu baris pun ditulis

Pemeriksaan pertama memberi hasil yang tidak diduga: **backend tingkat 2 sudah ada di repo**. Ia
berasal dari sesi sebelumnya, ikut masuk ke `master` lewat commit `4481dda`, lalu dinamai ulang ke
bahasa Inggris oleh rekan. Tetapi ia **kode mati** — lengkap, dan tidak terpasang ke apa pun.

Enam pemeriksaan dijalankan sebelum rencana disusun, dan semuanya nol:

| Pemeriksaan | Hasil |
|---|---|
| `Mount2` dirakit di `main.go` | **0 rujukan** |
| `ErrParentNotFound` dipetakan di `errors.go` | **0 rujukan** |
| Berkas uji tingkat 2 | **0 berkas** |
| Frontend tingkat 2 | **0 berkas** |
| Rute di `App.tsx` | **0 rujukan** |
| `StatusProgress2` di `registry.ts` | **0 rujukan** |

Membuka `/master/status-progres-2` hari itu karena itu tidak menghasilkan apa pun, dan menembak
`/api/master/status-progres-2` menjawab HTML halaman SPA — bukan JSON.

> Pelajaran yang dicatat: **kode yang ada di repo belum tentu kode yang hidup.** Yang membuktikan
> sebuah modul berjalan bukan keberadaan berkasnya, melainkan satu baris di berkas perakitan.

### 16.2 Layar lama dibaca ulang, bukan diandaikan kembar

`StatusProgress2-Harness.xml` (282.747 bita) dan `StatusProgress-Harness.xml` (282.680 bita)
berselisih **67 bita**, dan seluruh selisihnya metadata export. Perbedaan yang benar-benar berarti
hanya dua: `pyLabel` "Master Status Progress 2", dan Report Definition `BrowseStatusProgress2`.

Meski kembar secara struktur, **isinya tidak kembar** — dan tiga perbedaan berikut hanya terlihat
setelah rule-nya dibaca satu per satu:

| Hal | Tingkat 1 | Tingkat 2 |
|---|---|---|
| Tabel | `GCNM_MST_PROGRESS_KLAIM` | **`GCNM_MST_PROGRESS`** (nama lebih pendek) |
| Bentuk ID | `"0" + nomor` → `01`, `02` | **angka polos** → `1`, `2`, `60` |
| Isian kedua | Posisi klaim, daftar milik aplikasi | **induk**, dibaca dari tabel tingkat 1 milik entitas |
| Penyuntingan | ada dan bekerja | **ada tetapi tidak bekerja** — §16.4 |

Bentuk ID diverifikasi **dua kali, dari dua arah**: dari rule
(`Activity/InsertMstStatusProgress2_act` memakai `MAX(ID_MST)+1` apa adanya, sementara tingkat 1
merangkai `"0"+`), dan dari data yang beredar di kueri lain — `GetDataProgressClaim-SQL.xml`
menyaring `STATUS_PROGRESS2 not in ('2','24','60')`, angka polos, bukan `'02'`.

### 16.3 Alias kolom yang tidak boleh dipakai sebagai petunjuk arti

`BrowseStatusProgress2-SQL.xml` mengaliaskan kelima kolomnya ke nama yang tidak mencerminkan isi
sama sekali — utang teknis `03-CURRENT-ARCHITECTURE.md` §4.2 dalam bentuknya yang paling parah di
seluruh modul yang sudah dikerjakan:

```
ID_MST         AS "CaseID"       bukan nomor klaim
STS_PROGRESS1  AS "City"         nama INDUK
STS_PROGRESS2  AS "CityID"       nama BARIS INI SENDIRI
ID_PROGRESS    AS "District"     ID induk
TIPE           AS "DistrictID"   arti tidak diketahui
```

Perhatikan `City` dan `CityID`: keduanya **tidak berpasangan**. Dugaan yang wajar — "CityID adalah
kode dari City" — justru salah. Kelimanya dinamai ulang mengikuti `D-19`, dan yang dipetakan adalah
**kolomnya**, bukan aliasnya.

### 16.4 Tombol "Update" di layar lama tidak mengubah apa pun

Ini temuan yang paling menentukan bentuk modul ini, dan ia hanya muncul karena rantai
rule-nya ditelusuri sampai ujung, bukan berhenti pada namanya:

```
RDB List/UpdateStatusProgress2-SQL.xml      namanya "Update", isinya SELECT satu baris
Activity/UpdateMstStatusProgress2_act       menyiapkan lima Local.* lalu memanggil ↓
RDB List/UpdateStatusProgress2_sql-SQL.xml  UPDATE POOLDATA.GCNM_PROGRESS_CLAIM
                                            SET JSONSTATUS_PROGRESS2 = ...
```

Pernyataan terakhir menyentuh **tabel lain** — `GCNM_PROGRESS_CLAIM` adalah catatan progres milik
satu klaim, bukan master. Ia menyaring dengan `{tempSearchProgress.AnalystDoctorRemaks}` dan
`{tempSearchProgress.Email}`, dua page klipboard yang **tidak diisi** activity itu maupun
section layarnya. Kelima `Local.*` yang disiapkan dengan cermat tidak pernah dipakai satu pun.

Seluruh export diperiksa: **nol `UPDATE` dan nol `DELETE`** terhadap `POOLDATA.GCNM_MST_PROGRESS`.

Keputusan Work Owner untuk modul ini adalah **jalankan as-is**, dan itu yang dikerjakan — dengan
satu pengecualian yang dinyatakan terbuka: **jalur `UPDATE` ke `GCNM_PROGRESS_CLAIM` tidak
direproduksi.** Yang direplikasi adalah **hasil yang teramati** — baris master tidak berubah —
bukan jalur yang menghasilkannya. Menyalin jalurnya berarti membawa pernyataan yang, bila kedua
page itu kebetulan terisi sisa nilai dari layar lain dalam sesi yang sama, menimpa catatan progres
sebuah klaim dengan isian layar master.

### 16.5 Yang dikerjakan sesi ini

**Backend — dua sambungan yang mengubah kode mati menjadi API yang hidup:**

| Berkas | Perubahan |
|---|---|
| `cmd/claimpnc/main.go` | `NewHandler2` dirakit, `Mount2` dipasang, `Service2` dibangun, dua pemilih repo ditambahkan (Oracle dan memori) |
| `internal/masterstatusprogres/http/errors.go` | `ErrParentNotFound` dipetakan ke **422 pada isian `id_induk`** |

Pemilih repo memori tingkat 2 **tidak memeriksa alias portalnya sendiri**, melainkan menanyakannya
ke pemilih tingkat 1 — portal yang ditolak di sana ditolak di sini dengan galat yang sama persis.
Dua pemeriksaan terpisah atas hal yang sama akan berbeda begitu salah satunya disunting, dan yang
dipertaruhkan pada `R-20` adalah pemisahan data antar badan hukum.

Repo tingkat 1 yang dikembalikannya **dipakai langsung** sebagai induk, bukan disalin. Dengan
begitu status progres 1 yang baru ditambahkan lewat layarnya langsung muncul di dropdown tingkat 2
— perilaku yang sama dengan adapter SQL, yang membaca tabel induk di dalam transaksi yang sama.

**Backend — empat berkas uji, 0 → 40 kasus:**

| Berkas | Isi |
|---|---|
| `masterstatusprogres2_test.go` | `Clean`, `Check` (termasuk kasus batas dan pengumpulan seluruh pelanggaran), `FormatID2` tanpa awalan nol, `ErrParentNotFound` terbedakan |
| `usecase/manage2_test.go` | pemisahan antarportal, penolakan portal tak dikenal, induk baru langsung terlihat, ID diturunkan server, nama induk disalin |
| `repo/sqlstore/query2_test.go` | **tabel yang benar** (§16.6), tanpa UPDATE/DELETE, parameter binding, `TRIM(ID_MST)`, `FOR UPDATE`, `TIPE` tidak pernah ditulis |
| `http/routes2_test.go` | seluruh rute di balik sesi, **seluruh rute menuntut portal**, daftar kosong berupa `[]`, induk tak ada → 422 pada `id_induk`, PUT/PATCH/DELETE → 404 |

**Frontend — modul baru, menempel pada folder modul yang sama:**

`api2.ts` · `ProgressStatus2Form.tsx` · `ProgressStatus2Page.tsx` · `ProgressStatus2Page.test.tsx`
(11 kasus), ditambah tipe di `src/api/types.ts`, satu rute di `App.tsx`, dan **satu baris** di
`src/app/menu/registry.ts`.

### 16.6 Uji yang paling penting di modul ini: nama tabel

`TestQueries2TargetTheChildTable` memeriksa setiap kueri berawalan `progress_status2_` menyentuh
`POOLDATA.GCNM_MST_PROGRESS` dan **tidak** menyentuh `GCNM_MST_PROGRESS_KLAIM`.

Tampak sepele, dan justru itu sebabnya ia ada. Nama kedua tabel nyaris sama, **kedua tabel punya
kolom `ID_PROGRESS`**, dan tertukar sekali saja berarti layar tingkat 2 membaca — atau lebih buruk,
menulis — ke tabel induknya. Tidak ada galat basis data apa pun yang akan muncul.

### 16.7 Hasil pemeriksaan

| Pemeriksaan | Hasil |
|---|---|
| `gofmt` · `go build` · `go vet` | bersih |
| `go test ./...` | **24 paket lulus** |
| `tsc --noEmit` | bersih |
| `npm test` | **81 lulus · 3 gagal** (naik dari 68; 13 tambahannya uji Status Progres 2) |

Ketiga kegagalan itu tetap kegagalan lama yang sama di `AccountPage.test.tsx` — modul Master
Rekening yang berada di bawah Isolasi Protektif dan tidak disentuh sesi ini. Jumlahnya sama persis
sebelum dan sesudah seluruh pekerjaan.

**Ditembak ke binary produksi, bukan hanya lewat uji.** Uji membuktikan bentuknya benar; hanya
binary yang benar-benar berjalan — lengkap dengan SPA tersemat — yang membuktikan **rakitannya**
benar. Layar dibangun (`npm run build` → `backend/spa/dist`), disematkan (`go build`), lalu
dijalankan di porta lain supaya server Work Owner tidak terganggu:

```
GET  /api/menu                            MENU_ID 24 "Master Status Progress 2"
                                          program StatusProgress2  →  terkirim
GET  /master/status-progres-2             200 text/html  (SPA melayani rutenya)
bundel assets/index-*.js                  memuat "/master/status-progres-2",
                                          "StatusProgress2", dan judul layarnya

alur yang dilalui pengguna, berurutan:
  1. buka layar    GET  daftar          200
  2. tekan Tambah  GET  induk           200   6 pilihan
  3. Simpan        POST                 201   id=7 · induk 02 · tipe=''
  4. muat ulang    GET  daftar          200   6 → 7 baris
```

Tiga hal yang **hanya terbukti di sini**, tidak di uji mana pun:

| Yang terbukti | Kenapa uji tidak dapat membuktikannya |
|---|---|
| Butir menunya **hidup** | `registry.ts` dan `/api/menu` dirakit dua pihak yang berbeda; kecocokan ejaan `StatusProgress2` baru terbukti saat keduanya bertemu |
| Rutenya **masuk ke bundel** | uji Vitest merender komponen langsung, tidak melewati `App.tsx` maupun proses build |
| `Mount2` benar-benar **terpasang** | uji rute merakit servernya sendiri; hanya `cmd/claimpnc` yang membuktikan perakitan sungguhan |

**4 dari 71 butir menu kini punya layar** — naik dari 3. Ketiga yang lain: Master Status Klaim,
Master Rekening, Master Status Progres 1.

### 16.8 Satu temuan di luar lingkup: jalur `/api` yang tak dikenal dijawab halaman SPA

Terlihat saat menembak aplikasi yang benar-benar berjalan, bukan lewat uji — dan uji tidak dapat
melihatnya karena `httpserver.Router` di dalam uji dirakit **tanpa** berkas SPA.

```
DELETE /api/jalur-karangan                → 200, badan berisi index.html
DELETE /api/master/status-progres-2/1     → 200, badan berisi index.html
DELETE /api/master/status-progres-1/01    → 405  (jalur ini PUNYA rute, metodenya saja beda)
```

**Ini bukan bawaan modul ini dan bukan perilaku baru.** Ia berlaku untuk **setiap** jalur `/api`
yang tidak punya rute — termasuk jalur yang namanya dikarang. Penyebabnya penampung SPA yang
memasang dirinya pada seluruh sisa jalur, tanpa mengecualikan awalan `/api`.

Akibatnya bagi klien: permintaan ke endpoint yang tidak ada **tidak** dijawab `404` dalam bentuk
JSON, melainkan `200` berisi HTML. Klien yang mengurai jawabannya sebagai JSON akan gagal dengan
pesan yang tidak menyebutkan sebab sebenarnya.

Untuk modul ini akibatnya terbatas: `PUT` dan `DELETE` memang **tidak melakukan apa pun** — tidak
ada baris yang berubah, dan `TestNoEditRoute2Exists` membuktikan rutenya memang tidak terdaftar.
Yang keliru hanyalah **bentuk penolakannya**.

> **Sudah tidak berlaku sejak 2026-09-20** untuk `PUT`: rute penyuntingan ditambahkan atas
> keputusan Work Owner (§16.13), sehingga `PUT /api/master/status-progres-2/{id}` kini dilayani.
> `TestNoEditRoute2Exists` diganti `TestOnlyPutIsRegistered2`, yang menjaga `DELETE` tetap tidak
> dilayani — pada router modulnya, chi menjawabnya `405`.
>
> `DELETE` pada jalur itu juga **tidak** lagi tertelan penampung SPA: penampung itu dipasang
> sebagai `r.NotFound(...)` saja (`internal/platform/httpserver`, baris 44), sementara chi
> menjawab metode yang tidak dilayani lewat handler `MethodNotAllowed` yang **tidak** ditimpa.
>
> Paragraf di atas dibiarkan sebagai rekaman keadaan saat itu. Cacat penampung SPA-nya sendiri
> **belum diperbaiki** dan tetap berlaku untuk jalur `/api` yang tidak terdaftar sama sekali.

**Tidak diperbaiki di sesi ini**, dan itu disengaja: perbaikannya menyentuh `internal/platform/httpserver`
yang dipakai seluruh modul, sementara tugas sesi ini dibatasi pada Master Progres 2. Dicatat di sini
supaya tidak ditemukan ulang sebagai kejutan.

### 16.9 Yang belum dikerjakan, dan kenapa

| Hal | Alasan |
|---|---|
| Isi contoh `SampleList2()` | **bukan data produksi.** Isi `GCNM_MST_PROGRESS` tidak ada di export — tidak ada CSV-nya seperti `v_sts_claim.csv`, dan DDL-nya belum diterima (`R-08`). Nama-namanya susunan sendiri, ditandai jelas di doc comment-nya, dan **tidak boleh dipakai sebagai dasar uji kesetaraan gerbang 1** |
| Arti kolom `TIPE` | tidak diketahui. Di seluruh export ia hanya muncul pada dua `SELECT` — tanpa satu pun `INSERT`, `UPDATE`, maupun penyaring. Ia dibaca dan ditampilkan, tidak pernah ditulis |
| Panjang maksimum `STS_PROGRESS2` | **asumsi yang disadari** — 100, disamakan dengan tingkat 1. Berbeda dari tingkat 1, angka ini bukan ketetapan Work Owner; DDL-nya belum ada |
| Penyuntingan dan penghapusan | tidak ada di sistem lama — §16.4 |
| Salinan nama induk yang dapat menyimpang | dijalankan as-is atas keputusan Work Owner. Bahwa penyimpangan itu benar-benar terjadi di produksi terbaca dari `GetDataOutstandingperCabangExport-SQL.xml`, yang memakai `MAX(sts_progress1) ... GROUP BY id_progress` — `MAX` hanya diperlukan bila baris ber-`id_progress` sama menyimpan nama yang berlainan |

### 16.10 Harness dibaca sampai ke header kolom — dan dua kolom saya ternyata keliru

Sesi ini ditutup dengan membaca `Harness/StatusProgress2-Harness.xml` **sampai ke isinya**, bukan
hanya strukturnya. Pembacaan pertama berhenti pada kesimpulan "kembar dengan tingkat 1, selisihnya
hanya metadata export" — benar, tetapi **tidak cukup**: yang menentukan tata letak bukan harness-nya
melainkan section yang dirakitnya.

Header grid terbaca eksplisit di `Section/BrowseStatusProgress2-Section.xml`, dalam bentuk
`<pyValue>&lt;b&gt;…&lt;b&gt;</pyValue>`:

| Header Pega | Terikat ke | Kolom basis data |
|---|---|---|
| `No` | `.CaseID` | `ID_MST` |
| `Status Progress 1` | `.City` | `STS_PROGRESS1` — **nama induk** |
| `Status Progress 2` | `.CityID` | `STS_PROGRESS2` — nama baris ini |
| *(tanpa judul)* | `.pyTemplateInputBox` | tombol **Update** |

**Dua kekeliruan yang ditemukan pada layar yang sudah saya bangun:**

| # | Keliru | Yang benar |
|---|---|---|
| 1 | Nama baris ditaruh **sebelum** nama induk | Pega menaruh **induk lebih dulu** |
| 2 | Kolom **Tipe** ditambahkan sebagai kolom keempat | Pega **tidak punya** kolom itu sama sekali |

Keduanya sudah diperbaiki.

**Kekeliruan 1 berlawanan dengan dugaan yang wajar**, dan itu sebabnya lolos: pada layar master
mana pun, nama barisnya sendiri biasanya mendahului rujukan induknya. Di sini kebalikannya —
dan alias `City`/`CityID` yang menyesatkan membuat urutannya makin sulit dibaca dari kueri saja.
`TestProgressStatus2Page` kini memuat kasus yang mematok urutan `ID · Status Progres 1 ·
Status Progres 2`, sehingga tidak dapat tertukar lagi tanpa uji yang merah.

**Kekeliruan 2 adalah tambahan saya sendiri, bukan warisan Pega.** `DistrictID` (TIPE) terikat ke
page `TempUpdateStatus2` — **modal penyuntingan**, bukan baris grid. Karena modal itu memang tidak
dibawa (§16.4), TIPE tidak punya tempat di layar ini.

Yang dihapus **hanya kolomnya**. TIPE tetap dibaca kueri dan tetap dikirim pada respons API,
sehingga nilainya tidak hilang dari aplikasi dan siap dipakai bila kelak ada layar rincian. Satu
kasus uji menjaga pernyataan itu tetap benar.

> Ini **membalik** alasan yang saya tulis sendiri di §16.9 dan pada doc comment kolomnya —
> *"ditampilkan supaya nilai yang benar-benar tersimpan terlihat petugas"*. Alasan itu masuk akal,
> tetapi ia **alasan saya**, bukan perilaku sistem lama. `D-13` menetapkan tata letak mengikuti
> Pega, dan kolom yang tidak pernah ada di Pega adalah layar yang menuntut pengguna belajar hal
> baru tanpa ia memintanya.

**Tiga hal yang sengaja TIDAK diseragamkan ke Pega**, karena bertabrakan dengan konsistensi antar
layar bersaudara — dan `Master Status Progres 1` berada di bawah Isolasi Protektif sehingga tidak
dapat ikut disesuaikan:

| Hal | Pega | Dipakai di sini | Alasan |
|---|---|---|---|
| Judul kolom pertama | `No` | `ID` | Layar Progres 1 memakai `ID`; mengubah satu layar saja membuat dua layar bersaudara berbeda |
| Ejaan | `Progress` | `Progres` | idem — Progres 1 sudah memakai ejaan Indonesia |
| Judul layar | `Master Status Progress 2` | `Master Status Progres 2` | idem |

Ketiganya **kosmetik** dan tidak mengubah cara pengguna membaca tabel. Yang diperbaiki adalah yang
**substantif** — urutan kolom dan kolom yang tidak seharusnya ada.

**Satu hal yang saya catat sebagai utang, padahal itu kelalaian saya sendiri.**

Paragraf ini semula berbunyi: *"Grid Pega memuat `pyGridPaginator`, sedangkan `DataTable` bersama
**belum punya paginasi**. Menambahkannya berarti menyentuh `U-2` … dicatat sebagai utang."*

**Klaim itu salah.** `DataTable` **sudah** punya paginasi — prop `pageSize`, lengkap dengan
`pageWindow()` dan penjepitan halaman. Yang benar: saya **tidak mengisi propnya**. Bukan komponen
yang kurang, melainkan satu baris yang tidak saya tulis.

Yang membuatnya lebih buruk daripada sekadar terlewat: **layar tingkat 1 sudah memakainya**
(`pageSize={15}`), begitu pula Master Supplier (`pageSize={20}`). Jadi Progres 2 bukan hanya
menyimpang dari Pega — ia menyimpang dari **layar saudaranya sendiri**, persis hal yang dituntut
konsisten oleh prompt Work Owner.

Diperbaiki pada 2026-09-20 setelah Work Owner menanyakannya, dengan `pageSize={15}`. Angkanya
dibaca dari section **milik layar ini** — `Section/BrowseStatusProgress2-Section.xml`,
`pyPageSize = Other` dan `pyPageSizeOther = 15` — bukan disalin dari tingkat 1. Kebetulan sama,
dan kebetulan itu diperiksa. Satu uji baru menjaganya tidak hilang lagi.

> Pelajarannya: **"dicatat sebagai utang" adalah kalimat yang harus dicurigai.** Ia terdengar
> bertanggung jawab, dan justru karena itu ia dapat menutupi kelalaian yang sebenarnya berbiaya
> satu baris. Sebelum menulisnya, periksa dulu apakah kemampuannya memang belum ada.

**Yang diperiksa ulang dan ternyata SUDAH benar:**

| Hal | Pega | Layar |
|---|---|---|
| Tombol | `Tambah` · `Refresh` | sama persis |
| Urutan isian form tambah | `TempInputStatus2.CityID` (induk) → `.District` (nama) | induk lalu nama ✔ |
| Jumlah isian form tambah | 2 | 2 ✔ |
| Judul layar di section | `Master Status Progress 2` | ✔ (ejaan, lihat tabel di atas) |

---

### 16.11 Dua pertanyaan Work Owner: paginasi dan fitur ubah (2026-09-20)

Work Owner menanyakan dua hal sekaligus: *"kenapa tidak ada pagination, fitur ubah seperti pada
Pega?"* Keduanya wajar ditanyakan, dan **jawabannya berbeda sama sekali** — yang satu kelalaian
saya, yang satu perilaku sistem lama yang memang tidak bekerja.

#### Paginasi — kelalaian, sudah diperbaiki

Lihat §16.10. Ringkasnya: `DataTable` **sudah** punya paginasi, propnya tidak saya isi, dan layar
tingkat 1 sudah memakainya. Diperbaiki dengan `pageSize={15}` yang dibaca dari section milik layar
ini sendiri.

#### Fitur ubah — bukti diperluas, lalu diputuskan tetap tidak ada

Kesimpulan kemarin (§16.4) bersandar pada pembacaan rantai rule. Untuk menjawab pertanyaan ini,
buktinya diperluas ke **seluruh export**, dan hasilnya jauh lebih tegas:

| Bukti | Hasil |
|---|---|
| `UPDATE`/`DELETE` terhadap `GCNM_MST_PROGRESS` di seluruh export | **nol** |
| Pembanding: `UPDATE` terhadap `GCNM_MST_PROGRESS_KLAIM` (tingkat 1) | **ada** — `UpdateStatusProgress1_sql` |

Kedua pernyataan `UPDATE` itu berdampingan memperlihatkan asimetrinya:

```sql
-- tingkat 1 — mengubah MASTER, berkunci ID barisnya sendiri
UPDATE POOLDATA.GCNM_MST_PROGRESS_KLAIM
   SET STS_PROGRESS1 = {TempInputStatus.City}, STATUS = {TempInputStatus.CityID}
 WHERE ID_PROGRESS = {TempInputStatus.CaseID}

-- tingkat 2 — mengubah TABEL LAIN, berkunci NOMOR KLAIM
UPDATE POOLDATA.GCNM_PROGRESS_CLAIM C
   SET JSONSTATUS_PROGRESS2 = {TempInputStatusProgress2.City}
 WHERE PNCCASEID = {tempSearchProgress.AnalystDoctorRemaks}
   AND ID_UPDATE = {tempSearchProgress.Email}
```

Dan **rantainya putus di tiga nama page yang berbeda**:

| Page | Dipakai | Diisi di mana |
|---|---|---|
| `TempUpdateStatus2` | form modal mengikat ke sini | 5 berkas — section dan activity layar ini |
| `TempInputStatusProgress2` | SQL membaca `.City` dari sini | **tepat 1 berkas: SQL itu sendiri** — tidak pernah diisi siapa pun |
| `tempSearchProgress` | SQL menyaring dengan dua propertinya | hanya di layar **progres klaim** (`InputProgress-Harness`), bukan layar master ini |

Form menulis ke satu page; SQL membaca dari dua page lain, salah satunya tidak pernah ada isinya.
Menekan "Update" karena itu **tidak mengubah satu baris pun di tabel master**.

#### Tiga pilihan yang ditawarkan, dan yang dipilih

| Pilihan | Jangkauan | Keputusan |
|---|---|---|
| Ubah **nama saja** | `SET STS_PROGRESS2` · `WHERE TRIM(ID_MST)` — induk dan kunci tidak tersentuh | — |
| Ubah **nama + induk** | ditambah `ID_PROGRESS` dan salinan `STS_PROGRESS1` dibaca ulang | — |
| **Tidak ditambahkan** | tetap as-is | ✅ **dipilih Work Owner** |

Pada kedua pilihan pertama, `ID_MST` **tidak pernah** ikut di-`SET` — ia dirujuk
`GCNM_PROGRESS_CLAIM.STATUS_PROGRESS2` pada data klaim yang sudah berjalan. Batas yang sama
dipegang tingkat 1, yang juga tidak pernah meng-`SET` kolom kuncinya.

**Akibat yang diterima secara sadar:** salah ketik nama **tidak dapat diperbaiki** lewat layar
ini. Satu-satunya jalan adalah menambah baris baru, dan baris lama tetap ada karena tombol hapus
pun tidak ada. Ini keadaan yang sama persis dengan sistem lama — dan itulah yang membuat gerbang 1
uji kesetaraan tidak akan menemukan selisih pada modul ini.

> Yang membuat pertanyaan ini berharga: ia **memaksa bukti diperluas dari rantai rule ke seluruh
> export**. Kesimpulannya tidak berubah, tetapi dasarnya berpindah dari "saya menelusuri dan
> menemukan jalurnya buntu" menjadi "pernyataan itu tidak ada di mana pun". Yang kedua dapat
> diperiksa ulang siapa saja dengan satu perintah.

## 17. Sesi kesepuluh — Master Penolakan Klaim (2026-09-19)

Modul bisnis keempat. Pengganti `Harness/PNC_MasterTolakKlaim-Harness.xml`, MENU_ID 25.

### 17.1 Analisis pra-implementasi

Instruksi Work Owner menyebut satu berkas sebagai rujukan: `Harness/PNC_MasterTolakKlaim-Harness.xml`
(292 KB). Berkas itu ternyata hanya **cangkang portal** — ia memuat `MasterRejectNoteKlaim`, yang
sendirinya pembungkus tipis (`pyUsage`: *"thin wrapper around the My Cases display"*). Isi
sebenarnya ada dua lapis lebih dalam.

Jejak yang ditelusuri, berurutan:

```
Harness/PNC_MasterTolakKlaim-Harness.xml      cangkang, pyClassName Data-Portal
  └─ Section/MasterRejectNoteKlaim            pembungkus; pyMemo "master penolakan komite"
       └─ Section/BrowseNoteRejectClaim       700 KB — ISI SEBENARNYA
            ├─ Activity/GetAllDataMasterRejected        grid komite
            ├─ Activity/InsertMasterRejectedKomite      simpan komite
            ├─ Activity/BrowseStatusPenolakanKlaim_2    grid penolakan klaim
            ├─ Activity/InsertMasterPenolakanNoteKlaim  simpan penolakan klaim
            ├─ Activity/UpdateStatusPenolakanKlaim_act  muat satu baris ke modal
            └─ Activity/SetStatusMasterRejectsKlaim     pemilih tab
```

Tiga stored procedure yang mendasarinya **ada source-nya** di `Database/` — berbeda dari
kebanyakan modul lain yang harus menebak dari pemanggilnya:
`MASTERPENOLAKANKLAIM1.prc`, `MASTERPENOLAKANKLAIM2.prc`, `INSERTMASTERREJECTEDKOMITE.prc`.

**Temuan yang mengubah rancangan.** Empat, dan tiga di antaranya nyaris saya baca terbalik:

| # | Temuan | Kalau salah baca |
|---|---|---|
| 1 | Layar ini mengelola **DUA master** yang tabelnya tidak berhubungan, dipilih dua tombol lewat `FlgMasterPenolakan.FlagASO` | Setengah layar tidak terbangun |
| 2 | Grid Penolakan Klaim **tidak tersaring**. `{ASIS:MasterCheckerPenolakan.RemakApprove}` diisi `"WHERE STATUS='0'"` — tetapi langkah pengisinya **berprasyarat `Param.master=="1"`**, dan layar ini tidak mengirimnya | Baris yang sudah diputuskan hilang dari layar |
| 3 | Persetujuan **bukan milik layar ini**. `Sec_PenolakanKlaimChecker` dipakai `UserInbox_Harness` (Inbox Manager, MENU_ID 58) | Dua layar berebut menulis kolom yang sama |
| 4 | `MASTERPENOLAKANKLAIM1.prc` **INSERT pada kedua cabang IF-nya** (`:9` dan `:14`), tidak pernah UPDATE | Replikasi cacat yang menumbuhkan tabel tanpa batas |

Temuan 2 adalah yang paling mudah terlewat: saya sempat menulis penyaring `WHERE STATUS='0'` ke
dalam kueri sebelum membaca `pyStepsPreCondition` activity-nya.

**Gap yang tercatat, bukan ditambal:** `Flow Action/Flo_MasterPenolakanKlaim` — modal tombol
"Master Status Penolakan 1" — **tidak ada di export** (kelas `R-16`). DDL ketiga tabelnya juga
belum ada (`R-08`), dan isi datanya tidak ikut dikirim: tidak ada CSV-nya di `Database/`.

### 17.2 Pertanyaan konfirmasi dan jawabannya

Empat hal yang tidak dapat saya putuskan sendiri karena mengubah perilaku yang menyentuh data.

| # | Pertanyaan | Jawaban Work Owner |
|---|---|---|
| 1 | Dua master ini dibangun keduanya, atau salah satu dulu? | **Keduanya, sesuai Pega** — `MST_PENOLAKAN_KLAIM` untuk Master Penolakan Klaim, `MST_REJECTED_KOMITE` untuk Master Penolakan Komite |
| 2 | Cacat `MASTERPENOLAKANKLAIM1` — setiap simpan menerbitkan baris tingkat 1 baru | **Perbaiki**: pilih dari daftar, boleh tambah baru |
| 3 | `MASTERPENOLAKANKLAIM2` cabang update mereset `STATUS` ke `'0'` — setiap pengubahan mengembalikan baris ke antrean persetujuan | **Bawa apa adanya** |
| 4 | Kolom persetujuan (diisi Inbox Manager) ditampilkan bagaimana? | **Tampil di grid, baca-saja** |

Jawaban 2 adalah satu-satunya **penyimpangan dari `P-5`** pada modul ini, dan ia masuk daftar
perbaikan eksplisit sebagaimana `D-49`. Rinciannya di `keputusan-implementasi.md` §18.

### 17.3 Yang dibangun

**Backend — `internal/masterpenolakan`** (satu paket, dua master):

| Berkas | Isi |
|---|---|
| `masterpenolakan.go` | domain Penolakan Klaim — `RejectionStatus`, `RejectionStatus2`, `Input`, `Submission`, `ApprovalStatus`, seam `Repo` |
| `komite.go` | domain Penolakan Komite — `CommitteeRejection`, `InputKomite`, seam `RepoKomite` |
| `usecase/manage.go` · `usecase/komite.go` | dua layanan; yang pertama memegang `Clock`, yang kedua tidak |
| `repo/sqlstore/masterpenolakan.{go,sql}` · `komite.{go,sql}` | 17 kueri bernama |
| `repo/memory/memory.go` · `komite.go` | adapter kedua + data contoh |
| `http/dto.go` · `dto_komite.go` · `errors.go` · `routes.go` · `routes_komite.go` | satu Handler, dua tab |

**Frontend — `src/modules/master-penolakan-klaim`**: `RejectionPage.tsx` (dua tab),
`RejectionForm.tsx`, `CommitteeRejectionForm.tsx`, `api.ts`, `apiKomite.ts`.

**Tujuh endpoint baru**, seluruhnya di balik sesi **dan** portal:

```
GET  /api/master/penolakan-klaim
GET  /api/master/penolakan-klaim/status-1
POST /api/master/penolakan-klaim
PUT  /api/master/penolakan-klaim/{id}
GET  /api/master/penolakan-komite
POST /api/master/penolakan-komite
PUT  /api/master/penolakan-komite/{id}
```

Ditambah: `cmd/claimpnc` merakitnya, `check.go` melaporkan kesiapan ketiga tabelnya pada mode
`-periksa`, dan `app/menu/registry.ts` memetakan `PNC_MasterTolakKlaim` ke rutenya.

### 17.4 Kendala yang muncul dan penyelesaiannya

| Kendala | Penyelesaian |
|---|---|
| `Section/BrowseNoteRejectClaim` 700 KB, XML Pega yang tidak terbaca sebagai pohon layout | Diurai lewat **offset kemunculan** kata kunci: urutan `pyButtonLabel`, `pyCaption`, dan nama page memberi susunan layar tanpa perlu merekonstruksi pohonnya |
| Uji `TestListReadsRawStatusWithoutFormattingIt` gagal: kueri "mengandung APPROVED" | Kolom `NOTEAPPROVED` dan `TANGGAL_APPROVE` memang memuat kata itu. Asersinya dipersempit ke **literal bertanda kutip** `'APPROVED'` |
| `curl` ke `localhost` dijawab **HTTP 403 dari squid** | Proxy korporat mencegat. Dipakai `--noproxy` dengan pola semua host. Ini juga yang menjelaskan port 8080 "terpakai" — yang memegangnya squid, bukan instans aplikasi |
| `vitest run` seluruh suite gagal dengan *"Timeout waiting for worker to respond"* | Bukan uji yang gagal melainkan penjadwalan worker. Dijalankan per berkas; seluruhnya lulus |

### 17.5 Verifikasi yang benar-benar dijalankan

**Uji otomatis — seluruhnya lulus:**

| Paket | Yang dibuktikan |
|---|---|
| `masterpenolakan` | aturan isian, tepat-di-batas, label status, penurunan nomor |
| `masterpenolakan/repo/memory` | **perbaikan duplikasi induk**, reset status saat ubah, jejak persetujuan tetap, pemisahan antarportal |
| `masterpenolakan/repo/sqlstore` | tabel yang benar, nol DELETE, parameter binding, nol konstruksi khas Oracle, `FOR UPDATE` |
| `masterpenolakan/http` | sesi, portal, 422 seluruh pelanggaran, pelaku dari sesi, `DELETE` tidak dirutekan |
| `RejectionPage.test.tsx` — 18 uji | kolom dan urutannya, dua tab, kedua jalur induk, peringatan pembatalan persetujuan |

```
go vet ./...        bersih  (di luar internal/masterautoclaim — lihat §17.6)
go test ./...       lulus
npm run typecheck   bersih
```

**Uji terhadap aplikasi yang benar-benar berjalan** — binary hasil `go build` dengan SPA tersemat,
`PENYIMPANAN=memori`, port 8099:

| Yang diuji | Hasil |
|---|---|
| Pilih induk yang **sudah ada** | induk **4 → 4** — tidak ada duplikat |
| Minta induk **baru** | induk **4 → 5** — tepat satu |
| `PUT` pada baris ber-status `APPROVED` | status **kembali `MENUNGGU`**; `disetujui_oleh` dan catatannya **tetap terbaca** |
| Nomor komite pada tabel berisi 111–113 | baris baru **114** |
| Tanpa header `X-Portal` | **400 `portal_tidak_disebut`** — tidak jatuh ke portal utama |
| Tanpa sesi | **401** |
| `DELETE` | **405** |
| Isian kosong | **422** dengan **dua** pelanggaran sekaligus |
| Rute SPA `/master/penolakan-klaim` | **200** |

### 17.6 Satu hal di luar kendali sesi ini

Selama sesi berjalan, folder `internal/masterautoclaim/` muncul di working tree (13:18) — **bukan
buatan sesi ini**, dan `cmd/claimpnc/check.go` ikut disunting dari luar (penambahan
`checkAutoClaim`). Paket itu **tidak dapat dikompilasi** pada saat pemeriksaan terakhir:

```
internal/masterautoclaim/masterautoclaim.go:666: undefined: LookupRepo
```

Ia tidak disentuh sama sekali, sejalan dengan Isolasi Protektif. Akibatnya `go build ./...` dan
`go vet ./...` seluruh repo gagal; perintah verifikasi di §17.5 karena itu mengecualikannya. Ini
perlu diselesaikan pemiliknya sebelum repo dapat dibangun utuh.

**Tiga uji `AccountPage.test.tsx` juga gagal**, dan itu **pre-existing**: dibuktikan dengan
mengembalikan ketiga berkas bersama (`api/types.ts`, `app/App.tsx`, `app/menu/registry.ts`) ke
HEAD, menjalankan ulang uji itu — ketiganya tetap gagal — lalu mengembalikan versi sesi ini.
Penyebabnya tab bernama "Approve" pada layar Master Rekening yang ikut tertangkap
`queryByRole('button', { name: 'Approve' })`.

---

## 18. Sesi kesebelas — Master Auto Claim (2026-09-19)

### 18.1 Permintaan

> "lanjutkan untuk penambahan modul Master Auto Claim. cek secara penuh aplikasi existing pada
> dokumen `Harness/AutoKlaim-Harness.xml` jadikan ini sebagai referensi."

### 18.2 Yang ditemukan sebelum satu baris pun ditulis

Harness-nya 291 KiB, dan nama berkasnya tidak memberi tahu apa pun tentang isinya. Yang dibaca
lebih dulu, berurutan:

| Langkah | Hasilnya |
|---|---|
| `pyMemo` pada harness | *"harness untuk master auto claim"* — memastikan ini layar master, bukan inbox |
| Rujukan rule di dalamnya | **empat** section: `BrowseAutoKlaim`, `BrowseAutoKlaimKomite`, `BrowseAutoKlaimApproval`, `BrowseAutoKlaimReject`, dirakit `MasterAutoKlaim` |
| `pyDeferLoadRetrievalActivityParams` tiap section | penyaring tiap tab, **bukan tebakan** |
| `BrowseAutoKlaim-SQL.xml` | tabelnya `POOLDATA.M_AUTO_CLAIM_PNC`, 14 kolom |
| `InsertAutoClaim` + `UpdateAutoClaim` + `UpdateAutoClaim1` | pemetaan kolom ↔ page klipboard |
| Keempat activity-nya | urutan langkah, validasi, dan nilai yang diturunkan sistem |

Keempat tab terbaca persis, dan itu yang menghapus seluruh tebakan tentang arti kolom `APPROVAL`:

```
BrowseAutoKlaim          stsapprove="1"                 -> Master Auto Klaim
BrowseAutoKlaimKomite    stsapprove="0"  komite="ya"    -> Komite Approval
BrowseAutoKlaimApproval  stsapprove="0"                 -> Waiting Approval
BrowseAutoKlaimReject    stsapprove="2"                 -> Reject
```

### 18.3 Apa yang sebenarnya dikelola layar ini — dan namanya menyesatkan

"Master Auto Claim" terdengar seperti master klaim. Ia bukan. Yang menjawabnya bukan nama
melainkan pemakaian hilirnya, `RDB List/GetReceiverClaimAsuransiKredit-SQL.xml`:

```sql
select nama_penerima, alamat_penerima, bank_penerima, no_rekening, email_lapor,
       inisialid, pct_max
  from pooldata.m_auto_claim_pnc
 where inisialid = (select b.sourceofbusiness from t_general b where b.nopolis = ...)
   and claim_allowed = 1
   and APPROVAL = '1'
```

Tiga hal terbaca sekaligus dari satu kueri itu, dan ketiganya menyetir seluruh modul:

1. **`INISIALID` dicocokkan dengan `T_GENERAL.SOURCEOFBUSINESS`** — jadi ia kode **Sumber
   Bisnis**, bukan nomor apa pun yang diketik petugas. Itu sebabnya ia dipilih dari lookup
   `POOLDATA.AGENT`, tidak pernah diketik.
2. **`CLAIM_ALLOWED = 1` adalah penanda**, bukan pencacah.
3. **`APPROVAL = '1'`** — hanya baris yang disetujui komite yang dipakai membayar.

Jadi modul ini adalah **daftar sumber bisnis yang klaimnya boleh dibuat otomatis, beserta ke
mana ganti ruginya dibayarkan**. Ia master bernilai uang, dan itu yang menjelaskan kenapa ia
punya alur persetujuan komite sementara master lain tidak.

### 18.4 Alias kolom yang tidak boleh dipakai sebagai petunjuk arti

Sepuluh kolom dialiaskan ke nama yang tidak mencerminkan isi sama sekali. Dua di antaranya
menyesatkan **secara aktif**, dan keduanya ada di berkas yang sama:

| Alias | Saat DIBACA | Saat DITULIS |
|---|---|---|
| `City` | `NAMA_PENERIMA` (`BrowseAutoKlaim-SQL.xml`) | **`BANK_PENERIMA`** (`InsertAutoClaim-SQL.xml`) |

Dan pasangan yang tampak berpasangan ternyata tidak:

| Alias | Kolom |
|---|---|
| `District` | `NO_REKENING` |
| `DistrictID` | `EMAIL_LAPOR` |

Ditambah `CLAIM_ALLOWED AS "AnalystDoctorRemaks"`, `PIC_LAPOR AS "AlasanTerlambat"`,
`KOMITE AS "ProdKe"`, `CLIENTID AS "FlagReject"`, `CLIENTNAME AS "EmailTertanggung"`.

Yang dipetakan di seluruh modul karena itu adalah **kolomnya**, bukan aliasnya (`D-19`). Peta
lengkapnya ditulis di doc comment `masterautoclaim.go` supaya penelusuran balik ke Pega tetap
mungkin.

### 18.5 Empat keanehan yang dibawa ke Work Owner, bukan diputuskan sendiri

Keempatnya menyentuh aturan yang menentukan ke mana uang dikirim, sehingga tidak satu pun
diputuskan sepihak.

| # | Temuan | Jawaban Work Owner |
|---|---|---|
| 1 | `CLAIM_ALLOWED` ditimpa `"1"` saat **tambah**, tetapi memakai isian pengguna saat **ubah** — sehingga satu baris dapat "mati" karena disunting | **Selalu "1", tidak dapat diubah** |
| 2 | `KOMITE` diambil dari `emailkomite` dengan `type_business='BONDING'` **tertanam**, lalu baris pertama saja, **tanpa ORDER BY** | **Replikasi apa adanya** |
| 3 | Approve/Reject menulis ulang 13 kolom dari isi form, bukan hanya `APPROVAL` | **Kirim ulang seluruh isian seperti Pega** |
| 4 | `UpdateAutoClaim` tidak menulis `NAMA_PENERIMA` | **Nama penerima tidak bisa diupdate** |

### 18.6 Cacat yang ikut terbawa jawaban 3 — dan bagaimana ia ditutup tanpa mengubah bentuknya

Jawaban 3 mempertahankan "kirim ulang seluruh isian". Di Pega isian itu diambil dari **page
form**, dan page form hanya terisi bila barisnya lebih dulu dimuat lewat tombol Update.

Akibatnya: **komite yang menekan Approve langsung dari grid mengirim `CLIENTID` dan `CLIENTNAME`
kosong**, dan `UpdateAutoClaim-SQL.xml` menulis keduanya apa adanya. Menyetujui sebuah baris di
sana dapat **menghapus data client-nya**, tanpa satu pun pesan.

Bentuknya dipertahankan; jalurnya tidak. Dua perubahan kecil yang menutupnya:

1. **Kueri daftar ikut membaca `CLIENTID` dan `CLIENTNAME`** — kueri Pega tidak. Dengan begitu
   layar selalu memegang nilai yang sebenarnya, dan pengiriman ulang tidak dapat menghapus apa
   pun. Menambah kolom pada sebuah `SELECT` tidak mengubah satu baris pun.
2. **`KOMITE` tidak pernah datang dari permintaan** — ia dibaca dari baris tersimpan dan ditulis
   kembali apa adanya. Di Pega ia pun ditulis dari isi form, sehingga menyetujui dapat menghapus
   **penyetujunya sendiri**.

Yang direplikasi adalah hasil yang teramati pada jalur normal, bukan jalur yang merusak.

### 18.7 Yang dikerjakan sesi ini

**Backend** — `internal/masterautoclaim/`, susunan sama dengan modul master lain:

| Berkas | Isi |
|---|---|
| `masterautoclaim.go` | agregat, `ApprovalStatus`, `Input`, aturan isian, seam `Repo` |
| `lookup.go` | `BusinessSource`, `Client`, `Bank`, seam `LookupRepo` |
| `usecase/manage.go` | `List`, `Get`, `Create`, `Save`, ketiga lookup |
| `repo/sqlstore/` | 12 kueri bernama + adapter |
| `repo/memory/` | adapter kedua + data contoh keempat tabel |
| `http/` | dto, pemetaan galat, handler, rute |

**Frontend** — `src/modules/master-auto-claim/`: `AutoClaimPage` (4 tab), `AutoClaimForm`,
`LookupPicker`, `api.ts`.

**Perakitan** — `cmd/claimpnc/main.go` (satu selector per portal) dan `check.go` (mode periksa).

### 18.8 Tiga selisih yang DIRENCANAKAN terhadap Pega

Ketiganya dinyatakan di muka supaya tidak ditemukan sebagai kejutan pada uji kesetaraan gerbang 1.

| Selisih | Sebab |
|---|---|
| `CLAIM_ALLOWED` selalu `"1"` juga pada jalur **ubah** | keputusan Work Owner 18.5 #1 |
| `PCT_MAX` ditolak bila bukan angka 0–100 | kolomnya persentase; Pega menerima teks apa pun, dan akibatnya baru muncul jauh di hilir |
| Daftar memakai `ORDER BY INISIALID` | kueri lama tanpa `ORDER BY`, sehingga urutannya apa pun yang dikembalikan basis data |

Ditambah satu perubahan yang **bukan** selisih hasil: penyaring komite yang di Pega dirangkai
menjadi teks SQL (`"and KOMITE = '" + OperatorID.pyUserIdentifier + "'"`) menjadi kueri
tersendiri dengan parameter terikat. Baris yang dikembalikan sama; celah injeksinya tidak ikut.

### 18.9 Satu pemeriksaan yang dibuat LEBIH kuat daripada aslinya

Pega menolak dengan *"Nama bank jangan diketik manual"* bila `TempInputAutoClaim.Location` —
kode bank dari autocomplete — kosong. Ia memeriksa "ada sesuatu yang dipilih".

Kode bank itu **tidak pernah disimpan**: tabelnya tidak punya kolomnya. Menerimanya dari layar
karena itu berarti mempercayai peramban atas nilai yang tidak dapat dibaca kembali — dan membuat
tombol Approve bergantung padanya.

Yang dipakai sebagai gantinya: server mencocokkan **nama bank yang akan tersimpan** ke
`GENERAL.LST_BANK_GROUP`. Maksudnya sama, tetapi ia tidak dapat ditipu dengan mengirim kode
karangan bersama nama karangan. Galatnya tetap menempel pada isian `nama_bank`, bukan menjadi
500.

### 18.10 Uji yang paling penting di modul ini: urutan kolom

`scanRow` membaca ketiga kueri pembaca dengan **satu** fungsi, berdasarkan **posisi** kolom. Satu
kolom yang bergeser di salah satu kueri akan menaruh nomor rekening ke kolom alamat **tanpa satu
pun galat** — pada modul yang menentukan ke mana uang dikirim.

`TestReaderQueriesShareColumnOrder` membaca ketiga kueri dari berkas `.sql`, mengambil daftar
kolomnya, dan menuntut ketiganya identik.

Ditambah `TestUpdateNeverTouchesReceiverName`, yang menjaga keputusan 18.5 #4 tetap menjadi
keputusan sadar — bukan kelalaian saat menyalin daftar kolom dari `INSERT` di sebelahnya.

### 18.11 Hasil pemeriksaan

| Pemeriksaan | Hasil |
|---|---|
| `go build ./...` | **hijau** |
| `go vet ./...` | **hijau**, nol keluaran |
| `go test ./...` | **hijau**, seluruh paket |
| `internal/masterautoclaim/...` | **hijau** — 4 paket |
| `npm run typecheck` | **hijau** |
| `vitest run src/modules/master-auto-claim` | **hijau**, 14 uji |
| `gofmt` atas berkas modul ini | bersih |

**Tiga uji `AccountPage.test.tsx` gagal, dan itu pre-existing** — sama seperti yang dicatat §17.
Dibuktikan ulang dengan cara yang sama: `git stash` atas keempat berkas sesi ini
(`api/types.ts`, `app/App.tsx`, `app/menu/registry.ts`, `modules/master-auto-claim/`), jalankan
ulang — **ketiganya tetap gagal** — lalu `git stash pop`. Modul `master-rekening` tidak tersentuh
sesi ini (Isolasi Protektif).

### 18.12 Yang belum dikerjakan, dan kenapa

| Hal | Alasan |
|---|---|
| **Paginasi tabel** | `DataTable` bersama belum punya; menambahkannya menyentuh `U-2` yang dipakai seluruh layar master |
| **Mengangkat `'BONDING'` menjadi konfigurasi** | Work Owner memilih replikasi apa adanya (18.5 #2). Dicatat sebagai utang `D-15`, dan dijaga terlihat oleh `TestCommitteeQueryKeepsHardcodedBusinessType` |
| **Menutup balapan penambahan sepenuhnya** | menuntut constraint unik pada `INISIALID` — DDL-nya belum ada (`R-08`), dan perubahan skema menempuh `D-63` |
| **Migrasi skema** | `M_AUTO_CLAIM_PNC` dan keempat tabel acuannya adalah tabel **warisan** yang sudah ada; yang dibutuhkan hanyalah hak baca-tulis dari DBA |
| **Membersihkan baris ber-`CLAIM_ALLOWED` bukan "1"** | menuntut keputusan data yang belum diambil siapa pun. Yang dikerjakan adalah **melaporkan** jumlahnya di mode periksa dan menandainya di layar |

## 19. Sesi kedua belas — Master Pasal Kerugian (2026-09-19)

### 19.1 Permintaan

> "lanjutkan untuk penambahan modul Master Pasal Kerugian. cek secara penuh aplikasi existing pada
> dokumen `Harness/DetailMasterPasalRejected-Harness.xml` jadikan ini sebagai referensi."

### 19.2 Yang ditemukan sebelum satu baris pun ditulis

Harness-nya 283 KiB, dan **nama berkasnya menyesatkan**: `DetailMasterPasalRejected` terbaca
seperti master penolakan, padahal judul di layarnya "Detail Pasal Kerugian" dan yang dikelolanya
butir ketentuan polis. Yang dibaca lebih dulu, berurutan:

| Langkah | Hasilnya |
|---|---|
| Rujukan rule di dalam harness | judul `pyCaption Detail Pasal Kerugian`, section `GridDetailMasterPasalRejected`, tombol `Tambah` dan `Refresh` |
| `GridDetailMasterPasalRejected-Section.xml` | pembungkus; isinya menyertakan `BrowsePasalDeatailMaster` |
| `BrowsePasalDeatailMaster-Section.xml` | grid 5 kolom, form 5 isian, tombol `Simpan` · `Ubah` · `Delete` |
| `pyPageListProperty` tiap repeat layout | grid daftar → `BrowseCOL.pxResults`; isian Bisnis → `TempPasalCol.BISNISID`, kelas `ASM-FW-GISFW-Int-BUSINESS` |
| `pyActivity` tiap tombol | pemetaan tombol ↔ activity, **bukan tebakan** |
| Kedua activity-nya | urutan langkah, satu-satunya validasi, dan nilai yang diturunkan sistem |
| `PEGA_D_PASAL_MASTER.prc` | sisip ATAU perbarui menurut ada-tidaknya `IDDATA` |
| `GetPageJSONString-Function.xml` | **satu baris Java** yang menentukan bentuk seluruh dokumen JSON |

Pemetaan tombolnya terbaca persis:

```
Ubah (per baris)  PNCGetListPasalDataCOL_Act(idstatusp=.OLD_M_COL_ID, jaminanid=.pyCountry)
Simpan            CNMInsertPasalDataMaster()
Delete            CNMInsertPasalDataMaster(DeleteFlag="1")
```

### 19.3 Temuan yang menyetir seluruh modul: tabelnya hanya tiga kolom

`POOLDATA.V_M_DATA_PASAL` punya `IDDATA`, `IDPASAL`, dan `JSONPASAL`. Seluruh isi selain No Pasal
hidup di dalam dokumen JSON pada kolom ketiga.

Bentuk dokumennya **tidak ditebak**. `Activity/CNMInsertPasalDataMaster-Act.xml` langkah 4 mengisi
`InputData.OLD_D_COL_ID := @GCNM.GetPageJSONString()` dengan step page `TempPasalCol`, dan fungsi
itu isinya satu baris:

```java
String retValue = stepPage.getJSON(false);
```

`false` berarti tanpa metadata Pega — jadi kuncinya adalah nama properti page apa adanya. Empat di
antaranya terbukti langsung dari `GetDataCOLByPasalBisnis_Sql-SQL.xml` yang membacanya dengan
`json_value`: `$.DESCRIPTION`, `$.OLD_D_COL_ID`, `$.pyCountry`, `$.LOSS_CODE`.

### 19.4 Alias kolom yang tidak boleh dipakai sebagai petunjuk arti

Kelas Pega-nya `ASM-FW-GCNMFW-Int-V_D_CAUSE_OF_LOSS` — kelas **Detail Cause of Loss**, bukan kelas
pasal. Propertinya ikut terbawa dari sana, dan tidak satu pun namanya menyebutkan isinya.

| Properti | Arti sebenarnya | Label di layar |
|---|---|---|
| `M_COL_ID` | nomor pasal | No Pasal |
| `DESCRIPTION` | isi ketentuan | **ISI PASAL** |
| `OLD_D_COL_ID` | keterangan singkat | **Deskripsi** |
| `OLD_M_COL_ID` | kunci baris (`IDDATA`) | — |
| `pyCountry` | kode kategori | Kategori |
| `LOSS_CODE` | sebutan kategori | Kategori |

Dua yang paling mudah tertukar adalah `DESCRIPTION` dan `OLD_D_COL_ID` — **dugaan yang wajar justru
terbalik**, dan yang membuktikannya urutan kolom grid: `No Pasal | ISI PASAL | Deskripsi | Kategori`
berpasangan dengan `.M_COL_ID | .DESCRIPTION | .OLD_D_COL_ID | .LOSS_CODE`.

Dan `LOSS_CODE` memikul **dua arti** di dua kueri berbeda: di `GetDataCOLByPasalBisnis_Sql` ia
sebutan kategori dari JSON, sedangkan di `BrowseCOLByPasalDataBisnis_Sql` ia
`POOLDATA.BUSINESS.NOTE` — nama lini bisnis. Satu nama, dua isi yang tidak berhubungan sama sekali.

### 19.5 Empat pertanyaan yang dibawa ke Work Owner, bukan diputuskan sendiri

Keempatnya dijawab sama: **"coba jalankan secara as is"**.

| # | Temuan | Jawaban |
|---|---|---|
| 1 | Tombol Delete menghapus **fisik**, sementara `D-66` menetapkan soft delete menyeluruh; tabelnya tidak punya kolom penanda terhapus | **`DELETE` fisik seperti Pega** |
| 2 | Daftar pilihan Kategori **tidak ada di export** (`R-16`); yang terbaca hanya `1`, `2`, dan cabang `else` | **as is** — "Notifikasi" diperlakukan sebagai cabang `else`, kodenya kosong |
| 3 | Autocomplete Bisnis ber-`pyAllowFreeFormInput=true`: nama di luar master tetap tersimpan tanpa kode | **as is** — ketikan bebas diterima |
| 4 | Satu-satunya validasi adalah No Pasal wajib; nomor boleh kembar | **as is** — keunikan tidak diberlakukan |

Jawaban #1 **menyupersede aturan Steering yang sudah diputuskan**, dan itu tidak diselipkan
diam-diam: ia ditulis terang-terangan di `masterpasal.Repo`, di banner `masterpasal.sql`, di README
dengan blok peringatan tersendiri, dan di `keputusan-implementasi.md` §20.1.

### 19.6 Dua cacat procedure lama yang TIDAK direplikasi — dan kenapa

Keduanya menyangkut nomor yang **diterbitkan sistem**, bukan nilai yang diketik pengguna, sehingga
tidak ada yang berubah di layar.

| Cacat | Bukti | Akibat bila direplikasi |
|---|---|---|
| `max()` **tanpa** `NVL` | `PEGA_D_PASAL_MASTER.prc:11` | pada tabel kosong, `NULL+1` tetap `NULL` → baris pertama lahir **tanpa kunci** |
| `TO_NUMBER` atas kolom teks | idem | satu baris ber-`IDDATA` bukan angka menggagalkan **seluruh** penambahan dengan ORA-01722 |

Pola yang sama sudah dipakai `masterpenolakan.NextSequence`, dan alasannya sama persis.

Ditambah satu perilaku yang tidak dibawa: `PEGA_D_PASAL_MASTER.prc:8` **menyisipkan baris baru**
pada cabang "IDDATA tidak ketemu". Artinya `PUT` atas baris yang sudah dihapus petugas lain akan
diam-diam menerbitkan baris kedua. Di sini ia dijawab `404`.

### 19.7 Satu view yang tidak dapat dibawa, dan penggantinya

Pega membaca daftar lini bisnis sebuah pasal dari `POOLDATA.View_DATA_PASAL`. View itu **tidak ada
di export**, DDL-nya belum diterima (`R-08`), dan kuerinya merangkai penyaring dari
`{ASIS:TempSearchBisnis.DESCRIPTION}` — pola yang memang dilarang §4.3.

Penggantinya: `JSONPASAL` dibaca **utuh** lalu dibongkar di Go. Hasilnya sama — daftar lini bisnis
yang sama, dari dokumen yang sama — tanpa bergantung pada objek basis data yang tidak dapat dibaca
maupun dipindahkan ke PostgreSQL.

### 19.8 Yang dikerjakan sesi ini

**Backend** — `internal/masterpasal/`, susunan sama dengan modul master lain:

| Berkas | Isi |
|---|---|
| `masterpasal.go` | agregat `Clause`, `Business`, `Category`, `Input`, aturan isian, seam `Repo` + `LookupRepo` + `Store` |
| `usecase/manage.go` | `List`, `Get`, `Create`, `Update`, `Delete`, `SearchBusiness` |
| `repo/sqlstore/` | 10 kueri bernama + adapter + pembongkar dokumen JSON |
| `repo/memory/` | adapter kedua + data contoh, termasuk master lini bisnis |
| `http/` | dto, pemetaan galat, handler, rute |

**Frontend** — `src/modules/master-pasal-kerugian/`: `ClausePage`, `ClauseForm`, `BusinessPicker`,
`api.ts`.

**Perakitan** — `cmd/claimpnc/main.go` (satu selector per portal) dan `check.go` (mode periksa).

**Satu perubahan pada berkas bersama**, dan ia menjawab komentar yang memang menunggu:
`frontend/src/api/client.ts` membuka metode `DELETE`. Komentar di sana berbunyi *"metode yang tidak
tersedia di sini tidak dapat dipakai kode yang ditulis kemudian tanpa keputusan sadar"* —
keputusan itu kini ada, dan alasannya ditulis di tempat yang sama.

### 19.9 Uji yang paling penting di modul ini: nama kunci dokumen

Seluruh isi pasal selain No Pasal hidup di dalam satu kolom CLOB. **Satu nama kunci yang salah
ketik berarti isian itu hilang tanpa satu pun galat**, dan baru terlihat saat pengguna membuka
kembali pasalnya.

`TestDocumentKeysMatchPega` menyusun dokumen lalu menuntut ketujuh kuncinya ada — keempat yang
terbukti dari `json_value`, ditambah `M_COL_ID`, `OLD_M_COL_ID`, dan `BISNISID`.
`TestDocumentRoundTrip` membuktikan dokumen yang disusun dapat dibaca kembali utuh, termasuk butir
lini bisnis tanpa kode.

Ditambah `TestDocumentToleratesNonTextValues`: `pyCountry` dibandingkan sebagai **angka** di Pega,
sehingga baris lama dapat memuat angka polos pada kunci itu. Tanpa penerima yang memaafkan, satu
baris semacam itu membuat seluruh daftar gagal dibaca.

### 19.10 Kendala teknis yang muncul, dan penyelesaiannya

| Kendala | Penyelesaian |
|---|---|
| Dokumen JSON ditulis Pega, tipenya tidak dijamin teks | tipe `jsonText` yang menerima teks, angka, boolean, dan `null` — dan **menolak** selebihnya |
| `JSONPASAL` NULL versus rusak | NULL dibaca sebagai pasal tanpa isi; rusak menjadi **galat yang menyebut `IDDATA`-nya**, supaya barisnya dapat langsung dicari |
| Uji `scanRow` menuntut baris palsu | `fakeRow` yang **mewajibkan** penampungnya `*sql.NullString`; bila kelak ada yang membaca ke `string`, uji ini yang menangkapnya |
| Uji pertama saya menuntut **nol** permintaan sebelum portal dipilih | salah: daftar Kategori memang tidak bergantung portal. Ekspektasinya diperbaiki, bukan kodenya |

### 19.11 Hasil pemeriksaan

| Pemeriksaan | Hasil |
|---|---|
| `go build ./...` | **hijau** |
| `go vet ./...` | **hijau**, nol keluaran |
| `go test ./...` | **hijau**, seluruh paket |
| `internal/masterpasal/...` | **hijau** — 3 paket beruji |
| `npm run typecheck` | **hijau** |
| `vitest run src/modules/master-pasal-kerugian` | **hijau**, 14 uji |
| `gofmt` atas berkas modul ini | bersih |

**Tiga uji `AccountPage.test.tsx` gagal, dan itu pre-existing** — sama seperti yang dicatat §17 dan
§18. Dibuktikan ulang dengan cara yang sama, dan kali ini lebih ketat karena sesi ini menyentuh
berkas bersama `api/client.ts`: `git stash` atas `api/client.ts` dan `api/types.ts`, jalankan ulang
`vitest run src/modules/master-rekening` — **ketiganya tetap gagal (3 gagal, 6 lulus)** — lalu
`git stash pop`. Modul `master-rekening` tidak tersentuh sesi ini (Isolasi Protektif).

### 19.12 Yang belum dikerjakan, dan kenapa

| Hal | Alasan |
|---|---|
| **Pengikatan CLOB lebih dari 4000 karakter** | driver `go-ora` dapat menolaknya dengan ORA-01461 bila mengikatnya sebagai `VARCHAR2`. Tidak ada Oracle di lingkungan ini untuk mencobanya; dicatat di banner `masterpasal.sql` sebagai hal yang **wajib** dicoba pada basis data sungguhan sebelum modul dinyatakan lulus |
| **Kode Kategori selain "1", "2", dan kosong** | daftar pilihan aslinya tidak ada di export (`R-16`). Yang dikerjakan adalah **melaporkannya** di mode periksa bila benar-benar ada di data, bukan menebak kodenya |
| **Paginasi tabel** | `DataTable` bersama belum punya; menambahkannya menyentuh `U-2` yang dipakai seluruh layar master |
| **Jejak siapa dan kapan** | tabelnya tidak punya kolomnya. Menambah kolom menempuh `D-63`; sampai itu, perubahan dan penghapusan tidak meninggalkan jejak di basis data |
| **Migrasi skema** | `V_M_DATA_PASAL` dan `BUSINESS` adalah objek **warisan** yang sudah ada; yang dibutuhkan hanyalah hak baca-tulis dari DBA |

---

## 20. Sesi ketiga belas — Master Bengkel (2026-09-19)

Modul kesembilan. Menggantikan `Harness/BengkelHE-Harness.xml` (MENU_ID 28) atas
`POOLDATA.BENGKEL_HE` — **40 kolom**, form **33 isian**, tiga tab.

### 20.1 Pertanyaan yang diajukan dan jawabannya

Empat pertanyaan diajukan sebelum satu baris kode pun ditulis, masing-masing dengan
rekomendasi dan konsekuensinya.

| Pertanyaan | Jawaban Work Owner |
|---|---|
| Jalur simpan: JSON ke `M_BENGKEL_HE` lewat procedure, kolom nyata ke `BENGKEL_HE`, atau baca saja? | *"coba jalankan secara as is"* |
| Pembuatan operator Pega untuk bengkel rekanan | *"coba jalankan secara as is"* |
| Letak tombol Approve/Reject (di Pega ada di Inbox Manager) | *"coba jalankan secara as is"* |
| Cakupan form: 33 isian atau sebagian | *"coba jalankan secara as is"* |

Keempatnya dibaca sebagai **`P-5` — replikasi perilaku lebih dulu**. Tiga di antaranya
dapat dijalankan apa adanya; satu **tidak dapat**, dan alasannya teknis, bukan selera.
Rinciannya di `keputusan-implementasi.md` §21.

### 20.2 Yang dibaca dari export sebelum menulis kode

Harness-nya ternyata hanya cangkang; isinya berlapis empat:

```
Harness/BengkelHE                    282 KB
└─ Section/MasterBengkelHE           judul "MASTER BENGKEL HE"
   └─ Section/BrowseMasterHE         3 tab + tombol "Upload Data Master Bengkel"
      ├─ BrowseMasterHEApprove       tab APPROVE — grid + form 33 isian
      ├─ BrowseMasterHEApproval      tab WAITING APPROVAL
      └─ BrowseMasterHEReject        tab REJECT
```

Ditambah `Section/ApprovalMasterBengkelHE` yang **bukan** bagian layar ini — ia dipakai
`InboxManager_Sec`, tempat manajer menyetujui secara borongan.

Enam belas artefak dibaca seluruhnya: 1 harness, 5 section, 6 activity, 5 rule SQL,
1 report definition, 1 procedure, 2 data transform.

### 20.3 Delapan temuan yang menyetir rancangan

| # | Temuan | Bukti |
|---|---|---|
| 1 | Tulis JSON ke `M_BENGKEL_HE`, baca kolom dari `BENGKEL_HE` — **dua tabel untuk satu master** | `PEGA_M_BENGKEL_HE.prc:22` vs `BrowseBengkelHE_RD-RD.xml` |
| 2 | ID = kode situs + `LPAD(BENGKEL_HE_SEQ.NEXTVAL,10,'0')` | `PEGA_M_BENGKEL_HE.prc:11,19` |
| 3 | Baris baru **dan** baris yang disunting selalu `APPROVAL='0'` | `UpdateBengkelHE_act` step 7 |
| 4 | Nama bengkel tidak boleh ganda | `ValidationMasterBengkel-SQL.xml` |
| 5 | Approve bengkel rekanan **membuat operator Pega** dengan kata sandi yang sama untuk setiap bengkel | `ValidationLoginBengkel_act` step 2 dan 9 |
| 6 | Bengkel **non-rekanan** dilewati seluruh urusan login | prasyarat `Local.STS_REKANAN=='0'` |
| 7 | Penerima surel adalah **satu alamat pribadi** yang tertanam di rule | `UpdateBengkelHE_act`, properti `local.send` |
| 8 | Persetujuan borongan dipakai bersama bengkel, panel, dan sparepart; rule SQL-nya **hilang dari export** | `SetApprovalAllMaster-Act.xml`, `Param.TIPE2` |

### 20.4 Dua kolom yang memikul dua arti

Keduanya persis bentuk utang yang `03-CURRENT-ARCHITECTURE.md` §4.2 catat, dan keduanya
**tidak dibawa**:

| Kolom | Arti pertama | Arti kedua |
|---|---|---|
| `ACCOUNT_ID` | kolom rekening pada `BENGKEL_HE` | **pembawa seluruh dokumen JSON** pada `UpdateBengkelHE-SQL.xml` |
| `ALASAN_STS_BGKL` | alasan status bengkel | **pembawa nama tabel** pada `SetApprovalAllMaster` |

Di modul ini keduanya hanya berarti satu hal: kolomnya sendiri.

### 20.5 Nilai sah tujuh penanda status: nol bukti

`STS_SUPPLY`, `STS_EKLAIM`, `STS_AUTO_AKSEP`, `STS_PAYMENT`, `STS_AUTOPAYMENT`,
`STS_TEKNO`, `STS_ORDER` dirender `pxRadioButtons` atau `pxDropdown`, dan rule Field
Value-nya tidak ikut di export (`R-16`).

Pencarian menyeluruh dijalankan atas `Activity/`, `When/`, `RDB List/`, dan seluruh
section bengkel untuk setiap perbandingan terhadap ketujuhnya: **nol hasil**. Tidak ada
satu pun tempat di export yang menyebutkan nilai sahnya.

Yang dipakai sebagai gantinya: layar menawarkan **nilai yang sudah dipakai baris lain**
sebagai daftar saran. Sarannya berasal dari data, bukan dari tebakan.

Satu-satunya yang nilainya **diketahui** adalah `STATUS_REKANAN` bernilai nol berarti
non-rekanan, dan itu pun terbaca dari percabangan, bukan dari label.

### 20.6 Berkas yang dibuat

**Backend — 12 berkas, paket `masterbengkel`** (`D-81`: nama folder sama dengan nama
modul bisnis):

```
internal/masterbengkel/
├── masterbengkel.go              domain: Workshop 40 kolom, ApprovalStatus, Input, ComposeID
├── masterbengkel_test.go         18 uji domain
├── lookup.go                     seam Branch, City, Bank
├── usecase/manage.go             orkestrasi + ensureUnique
├── usecase/manage_test.go        21 uji aplikasi
├── repo/memory/{memory,sample}.go
├── repo/sqlstore/masterbengkel.{go,sql}
├── repo/sqlstore/query_test.go   15 uji kueri
└── http/{dto,errors,routes,routes_test}.go
```

**Frontend — 5 berkas, `src/modules/master-bengkel/`:**

```
api.ts                7 hook TanStack Query
WorkshopPage.tsx      3 tab + keputusan borongan + DataTable
WorkshopForm.tsx      33 isian dalam 6 kelompok berjudul
CityPicker.tsx        kotak cari-pilih untuk Kota
WorkshopPage.test.tsx 16 uji
```

**Wiring:** `cmd/claimpnc/main.go` (handler, mount, assembly, dua selector),
`cmd/claimpnc/check.go` (`checkBengkel` beserta empat pemeriksa pendukung),
`api/types.ts`, `app/App.tsx`, `app/menu/registry.ts`, `README.md`.

### 20.7 Mode periksa menjawab pertanyaan yang tidak dapat dijawab export

`claimpnc -periksa` kini melaporkan lima hal untuk modul ini, dan yang **ketiga** adalah
alasan utama fungsi itu ada:

1. jumlah baris per status — apakah ketiga tab akan terisi;
2. ketersediaan penomoran (kode situs dan sequence), dengan benar-benar mengambil satu
   nomor — sequence yang dapat dibaca tetapi tidak dapat diambil nomornya adalah keadaan
   yang tidak terlihat dari pemeriksaan yang lebih lembut;
3. **perbandingan jumlah baris `BENGKEL_HE` dengan `M_BENGKEL_HE`** — cara termurah
   mengetahui apakah keduanya satu sumber (view) atau dua sumber yang disinkronkan, tanpa
   DDL;
4. ketiga tabel acuan, termasuk peringatan bila daftar cabang kosong karena penyaring
   kode aplikasi yang tertanam di kuerinya;
5. jumlah bengkel **rekanan tanpa login aplikasi** — keadaan yang di Pega tidak terlihat
   di layar mana pun.

### 20.8 Kendala yang ditemui, dan bagaimana diselesaikan

**Tabrakan nama tombol.** Uji frontend gagal enam kali dengan *"Found multiple elements
with the role button and name Approve"*. Penyebabnya bukan cacat uji: caption tab —
"Approve" dan "Reject" — adalah caption Pega yang memang ditiru (`D-13`), dan tombol
keputusan diberi nama yang sama. Dua kontrol yang sama sekali berbeda karena itu tidak
dapat dibedakan dari namanya, dan pembaca layar mengumumkan keduanya dengan kata yang
sama persis.

Diperbaiki di **UI**, bukan di uji: tombolnya menjadi **"Approve terpilih"** dan **"Reject
terpilih"**. Kata Pega tetap dipakai, ambiguitasnya hilang, dan kata "terpilih" sekaligus
menyebutkan bahwa ia mengenai seluruh baris yang dicentang.

**Modul lain masuk di tengah sesi.** `cmd/claimpnc/main.go` sempat tidak dapat
dikompilasi karena modul `masterpasal` dari sesi lain sedang setengah dirakit di berkas
yang sama. Tidak ada yang dikembalikan atau diperbaiki dari pihak sana; pekerjaan
dilanjutkan, dan build hijau kembali setelah perakitan mereka selesai.

**Pembacaan section Pega.** Empat percobaan pertama mengekstrak label form gagal: format
export menyimpan referensi rule pada indeks `pzIndexes`, bukan sebagai atribut layout.
Yang akhirnya berhasil adalah membaca `pxRuleFamilyName` berawalan `PYCAPTION!` — dari
sana ke-33 caption terbaca utuh beserta nama propertinya.

### 20.9 Selisih terencana terhadap Pega

| Selisih | Alasan |
|---|---|
| Kelima persentase ditolak bila bukan angka 0–100 | kolomnya persentase; Pega menerima teks apa pun, dan akibatnya baru muncul jauh di hilir |
| Daftar memakai `ORDER BY NAMA_BENGKEL` | kueri lama tanpa `ORDER BY`, sehingga urutannya apa pun yang dikembalikan basis data |
| `pyMaxRecords=500` tidak direplikasi | ia memotong daftar tanpa satu pun tanda di layar; penggantinya penyaring kata kunci |
| Nama bengkel dan login diperiksa juga pada jalur **simpan** | tanpa itu, dua bengkel bernama sama dapat lahir cukup dengan menyunting salah satunya |
| `DOKUMENID` **dipertahankan** saat menyimpan | Pega menimpanya dengan kosong pada penyimpanan tanpa lampiran — lampiran lenyap hanya karena barisnya disunting |
| Kolom `MAIL` **tidak** ikut ditulis saat menyetujui | Pega menimpanya dengan identitas petugas, menghapus surel bengkel |
| Keputusan borongan dibungkus **satu transaksi** | Pega menjalankan satu RDB-List per baris tanpa transaksi yang melingkupinya |

### 20.10 Uji yang paling penting di modul ini: urutan kolom

`scanRow` membaca **lima** kueri pembaca dengan **satu** fungsi, berdasarkan **posisi**
kolom — dan kolomnya **empat puluh satu**. Satu kolom yang bergeser akan menaruh nomor
NPWP ke kolom SLA tanpa satu pun galat, dan pergeseran itu tidak mungkin terlihat dengan
membaca.

`TestReaderQueriesShareColumnOrder` membaca kelima kueri dari berkas `.sql`, mengambil
daftar kolomnya, dan menuntut kelimanya identik. Ditambah
`TestInsertColumnCountMatchesArguments` dan `TestUpdateArgumentCountMatchesQuery`, yang
menjaga daftar kolom di `.sql` tetap sejalan dengan urutan argumen di `.go` — dua berkas
terpisah yang masing-masing memuat empat puluh satu nama.

### 20.11 Hasil pemeriksaan

| Pemeriksaan | Hasil |
|---|---|
| `go build ./...` | **hijau** |
| `go vet ./...` | **hijau**, nol keluaran |
| `go test ./...` | **hijau**, seluruh paket |
| `internal/masterbengkel/...` | **hijau** — 4 paket, 54 uji |
| `gofmt -l ./internal/masterbengkel` | bersih |
| `npm run typecheck` | **hijau** |
| `vitest run src/modules/master-bengkel` | **hijau**, 16 uji |
| `vitest run` (seluruhnya) | 143 lulus, **3 gagal** |

**Ketiga kegagalan itu pre-existing** dan seluruhnya di
`master-rekening/AccountPage.test.tsx` — modul yang **tidak tersentuh sesi ini**
(dibuktikan `git status`: tidak ada berkas `master-rekening` yang berubah). Ketiganya
sudah tercatat pada §17.11 dan §18.11 dengan jumlah yang sama persis.

`gofmt -l` juga menandai `cmd/claimpnc/check.go` dan `main_test.go`, dan itu pun
pre-existing: keduanya tersimpan CRLF di working tree Windows sementara repositori
menyimpannya LF. Tidak ada yang diubah untuk itu — memperbaikinya akan menghasilkan diff
seluruh berkas yang menutupi perubahan yang sebenarnya.

### 20.12 Yang belum dikerjakan, dan kenapa

| Hal | Alasan |
|---|---|
| **Jalur tulis di produksi** | menunggu DBA memastikan `BENGKEL_HE` tabel atau view. `claimpnc -periksa` melaporkan keduanya; sebelum itu pasti, modul dijalankan baca saja |
| **Pembuatan akun bengkel** | menunggu kontrak identitas `F-3` (`R-14`). Layar **menyatakan** ketiadaannya, tidak menyembunyikannya |
| **Pemberitahuan ke PIC** | seam Notifier (`S-3`) belum ada, dan penerima di rule lama berupa alamat pribadi yang `D-67` larang dibawa. Peristiwanya dicatat di log |
| **Unggah lampiran** | `PNCSaveAttachmentToDB` di luar lingkup modul ini; `DOKUMENID` baris lama dipertahankan apa adanya |
| **Tombol "Upload Data Master Bengkel"** | ada di layar Pega; rule-nya tidak ikut di export |
| **Penyaring `cari` dipakai layar** | endpoint-nya siap; layar memakai pencarian bawaan `DataTable` seperti seluruh layar master lain. Perpindahannya dilakukan bersama paginasi sisi server (`TKT-U2-001`) |
| **Constraint unik `NAMA_BENGKEL` dan `LOGIN_APLIKASI`** | menutup balapan penambahan sepenuhnya menuntutnya; DDL-nya belum ada (`R-08`), dan perubahan skema menempuh `D-63` |
| **Mengangkat kode aplikasi cabang menjadi konfigurasi** | tidak ada satu pun keterangan di export tentang artinya; mengangkatnya berarti menebak nilainya untuk entitas lain. Dijaga terlihat oleh `TestBranchQueryKeepsHardcodedApplicationCode` |

---

## 21. Sesi keempat belas — Master Panel (2026-09-20)

Modul kesepuluh. Menggantikan `Harness/MasterPanel_HE-Harness.xml` (MENU_ID 30) atas
`POOLDATA.PANEL_HE` — **15 kolom**, form **10 isian wajib**, tiga tab — **ditambah tabel
anak `POOLDATA.LOKASI_PANEL_HE`**.

Ia **layar master pertama yang mengelola baris anak.** Sembilan modul sebelumnya rata:
satu baris layar sama dengan satu baris tabel.

### 21.1 Pertanyaan yang diajukan dan jawabannya

Dua pertanyaan diajukan sebelum satu baris kode pun ditulis, masing-masing dengan
rekomendasi dan konsekuensinya.

| Pertanyaan | Jawaban Work Owner |
|---|---|
| Sub-tabel Lokasi/Sisi: baca saja, kelola penuh, atau tidak dibawa? | *"Jalankan sebagai as is"* |
| Kesembilan kolom `STS_*` yang daftar nilainya tidak ada di export | *"Jalankan sebagai as is"* |

Keduanya dibaca sebagai **`P-5` — replikasi perilaku lebih dulu**:

- Sub-tabel **dikelola penuh**, karena di Pega pun ia ikut tersimpan: seluruh halaman
  klipboard diserialisasi `@GCNM.GetPageJSONString()` dan dikirim ke procedure.
- Kesembilan penanda disimpan **sebagai teks apa adanya**, tanpa menebak enum — perlakuan
  yang sama dengan tujuh penanda `STS_*` di Master Bengkel.

### 21.2 Yang dibaca dari export sebelum menulis kode

Harness-nya ternyata hanya cangkang; isinya berlapis empat:

```
Harness/MasterPanel_HE               289 KB
└─ Section/ListPanelHE               judul "Master Panel HE" + tombol tambah
   └─ Section/BrowsePanelHE          3 tab + isian "Catatan"
      ├─ BrowsePanelHEApproval       APPROVAL="0" — grid + form 10 isian
      ├─ BrowsePanelHEApprove        APPROVAL="1"
      └─ BrowsePanelHEReject         APPROVAL="2"
```

Ditambah `Section/ApprovalMasterPanelHE` yang **bukan** bagian layar ini — ia dipakai
Inbox Manager, tempat manajer menyetujui borongan lewat `Activity/SetApprovalAllMaster`
dengan `Param.TIPE2 = "M_PANEL_HE"`.

Artefak lain yang dibaca:

| Berkas | Yang diambil darinya |
|---|---|
| `Report Definition/BrowseMasterPanel_HE_RD-RD.xml` | 14 kolom, kelas `ASM-FW-GCNMFW-Int-PANEL_HE`, filter `ID_PANEL` + `APPROVAL`, `pyMaxRecords=500` |
| `RDB List/ValidationMasterPanel-SQL.xml` | tolak nama ganda, `upper(trim(name))` |
| `RDB List/UpdatePanel_HE-SQL.xml` | simpan lewat `PEGA_M_PANEL_HE` |
| `RDB List/GetLokasiSisiPanel-SQL.xml` | baris anak: `LOKASI_PANEL`, `SISI_PANEL` |
| `RDB List/GetDataSisiPanel-SQL.xml` | kolom `NAMA` — satu-satunya jejaknya |
| `RDB List/GetIDDokumenPanel-SQL.xml` | kolom `DOKUMENID`, yang tidak ada di RD |
| `Database/PEGA_M_PANEL_HE.prc` | ID = kode situs + `lpad(PANEL_HE_SEQ.nextval, 6, '0')` |
| `Activity/CNMUpdatePanelHE_act-Act.xml` | urutan simpan, `APPROVAL := "0"`, surel hardcode |
| `Activity/SetLokasiSisiPanel-Act.xml` | kelima pilihan Lokasi, ketiga sandi Sisi |
| `Activity/SetPanelHEValue-Act.xml` | pemuatan baris ke form, termasuk `ALASAN_TOLAK` |
| `Database/m_menu_aplikasi_pnc.csv` | MENU_ID 30, `MENU_PROGRAM = MasterPanel_HE` |

### 21.3 Lebar ID ternyata BERBEDA dari Master Bengkel

Kedua procedure ditulis dengan pola yang sama, dan lebarnya tetap tidak sama:

| Procedure | Sintaks | Lebar |
|---|---|---|
| `PEGA_M_BENGKEL_HE.prc:19` | `lpad(to_Char(BENGKEL_HE_SEQ.nextval),10,'0')` | **10** |
| `PEGA_M_PANEL_HE.prc:21` | `lpad(to_Char(PANEL_HE_SEQ.nextval),6,'0')` | **6** |

Menyeragamkannya akan menerbitkan ID yang tidak sebentuk dengan ID yang sudah ada.
Perbedaannya ditiru apa adanya dan dijaga `TestComposeID`.

### 21.4 Satu nama yang memikul dua arti

`STS_SISI` adalah kolom pada `PANEL_HE` dengan caption layar **"STATUS SISI"**. Nama yang
sama dipakai sebagai **alias kolom lain**: `GetLokasiSisiPanel-SQL.xml` menulis
`SISI_PANEL as "STS_SISI"` — kolom tabel **anak**, artinya sama sekali berbeda.

Di modul ini keduanya punya nama sendiri: `Panel.SideStatus` untuk kolom induk,
`PanelLocation.Side` untuk kolom anak; `status_sisi` dan `sisi_panel` pada kontrak API.
Ini bentuk utang yang sama dengan dua yang sudah dicatat di Master Bengkel.

### 21.5 Kolom `NAMA` — asumsi yang disadari, dan cara memeriksanya

Tabel anak punya empat kolom, dan hanya tiga yang artinya pasti. `NAMA` **hanya muncul
sebagai penyaring**, tidak pernah sebagai kolom yang dibaca:

```sql
select sisi_panel from pooldata.lokasi_panel_he
 where id_panel = {...} and nama = {...}
```

Pemanggilnya `Activity/GetSisiPanel-Act.xml`, yang hanya dipakai lima section modul
**Grouping Sparepart HE** — modul lain, di luar lingkup migrasi. Nilai yang dikirimkannya
adalah sebuah nama lokasi.

**Yang ditulis:** `NAMA` diisi nilai yang **sama** dengan `LOKASI_PANEL`. Itu satu-satunya
pembacaan yang konsisten dengan kedua kueri, dan satu-satunya yang menjaga modul Grouping
Sparepart tetap menemukan barisnya. Membiarkannya `NULL` akan **mematikan** modul itu
tanpa satu pun pesan galat.

**Asumsinya dapat diperiksa, dan pemeriksaannya sudah terpasang.** `claimpnc -periksa`
menghitung baris yang `NAMA`-nya berbeda dari `LOKASI_PANEL`:

- **nol** → asumsi benar, jalur tulis aman;
- **selain nol** → keduanya dua hal berbeda, dan `masterpanel.sql` harus diperbaiki
  **sebelum** jalur tulis diaktifkan di produksi.

Ini satu-satunya pemeriksaan di seluruh mode periksa yang menguji **asumsi penulisan**,
bukan sekadar ketersediaan tabel.

### 21.6 Hapus-lalu-sisip-ulang pada tabel anak — pertentangan dengan `D-66`

Penyimpanan mengganti **seluruh** baris lokasi sebuah panel: dibuang, lalu disisipkan ulang
dari daftar yang dikirim layar. Alasannya baris anak **tidak punya kunci sendiri** — tidak
ada kolom surrogate yang memungkinkan satu baris dikenali lintas penyimpanan.

`D-66` menetapkan **soft delete menyeluruh**, dan `ADR-0013` mencatat bahwa pengganti pola
hapus-lalu-sisip-ulang **belum diputuskan**. Pertentangannya dinyatakan terbuka di banner
`masterpanel.sql` dan dijaga `TestDeleteOnlyOnTheChildTable` — perlakuan yang sama dengan
Master Pasal Kerugian, layar pertama yang menghapus permanen.

Yang menahan akibatnya: keduanya berjalan di dalam **satu transaksi**, sehingga panel tidak
pernah berada dalam keadaan "lokasi lama sudah dibuang, yang baru belum masuk". Sistem lama
tidak menjamin itu.

### 21.7 Perubahan pada berkas bersama

| Berkas | Yang ditambahkan |
|---|---|
| `backend/cmd/claimpnc/main.go` | impor, `masterPanel` pada assembly, `panelSelector`, `panelSelectorMemory`, handler, `Mount` |
| `backend/cmd/claimpnc/check.go` | `checkPanel` beserta `checkPanelNumbering`, `checkPanelMirror`, `checkPanelLocation` |
| `frontend/src/api/types.ts` | `Panel`, `PanelLocation`, `PanelStatus`, `PanelSide`, `PanelInput`, `PanelErrorCode`, empat bentuk respons |
| `frontend/src/app/App.tsx` | rute `/master/panel` |
| `frontend/src/app/menu/registry.ts` | `MasterPanel_HE: '/master/panel'` |

### 21.8 Kendala teknis dan penyelesaiannya

| Kendala | Penyelesaian |
|---|---|
| **Build pertama gagal mengimpor `internal/masterpanel`** meski berkasnya sudah ada | `go build` berjalan di latar bersamaan dengan penyuntingan `main.go`, sehingga membaca keadaan setengah jalan. Diulang setelah seluruh berkas tertulis — hijau |
| **`cmd/claimpnc` gagal build dengan impor `mastersupplier` tidak terpakai** | Modul `mastersupplier` sedang dikerjakan **paralel di working tree yang sama** dan belum lengkap. Bukan berasal dari modul ini, dan **tidak disentuh**; ia sudah konvergen sendiri pada pemeriksaan berikutnya |
| **`TestDeleteOnlyOnTheChildTable` gagal pada ujinya sendiri** | Pemeriksaannya keliru: `"LOKASI_PANEL_HE"` berakhiran `"PANEL_HE"`, sehingga pencocokan substring selalu cocok dan tidak membuktikan apa pun. Diganti memeriksa **sasaran** `DELETE FROM POOLDATA.PANEL_HE` |
| **Nama aksesibel tombol hapus terbaca "Hapuslokasi baris 1"** | Cacat nyata pada komponen, bukan pada ujinya: algoritma nama aksesibel **memangkas setiap simpul teks lalu menyambungnya tanpa pemisah**, sehingga teks berkelas `sr-only` yang diawali spasi kehilangan spasinya. Diganti `aria-label` eksplisit |

### 21.9 Hasil pemeriksaan

| Perintah | Hasil |
|---|---|
| `go build ./...` | **hijau** |
| `go vet ./internal/masterpanel/...` | **hijau** |
| `go test ./...` | **hijau**, nol kegagalan |
| `gofmt -l internal/masterpanel` | **bersih** |
| `npm run typecheck` | **hijau** |
| `vitest run src/modules/master-panel` | **hijau**, 16 uji |
| `vitest run` (seluruhnya) | 159 lulus, **3 gagal** |

**Ketiga kegagalan itu pre-existing** dan seluruhnya di
`master-rekening/AccountPage.test.tsx` — modul yang **tidak tersentuh sesi ini**
(dibuktikan `git status`). Ketiganya sudah tercatat pada §17.11, §18.11, dan §20.11 dengan
jumlah yang sama persis.

`gofmt -l` juga menandai `cmd/claimpnc/check.go` dan `main_test.go`, dan itu pun
pre-existing: keduanya tersimpan CRLF di working tree Windows sementara repositori
menyimpannya LF. Dibuktikan langsung — `gofmt -l` terhadap versi `HEAD` berkas yang sama
**tidak menandainya**. Penyuntingan sesi ini mempertahankan CRLF-nya, jadi tidak ada yang
bertambah.

### 21.10 Yang belum dikerjakan, dan kenapa

| Hal | Alasan |
|---|---|
| **Jalur tulis di produksi** | menunggu DBA memastikan (a) `PANEL_HE` tabel atau view, dan (b) apakah `NAMA` sama dengan `LOKASI_PANEL`. `claimpnc -periksa` melaporkan keduanya; sebelum itu pasti, modul dijalankan baca saja |
| **Nilai sah kesembilan penanda `STS_*`** | rule Field Value-nya tidak ikut di export (`R-16`). Layar menawarkan nilai yang sudah dipakai baris lain sebagai saran, dan **menyatakan** bahwa daftarnya belum diterima |
| **Arti `STS_APPROVAL`** | nol keterangan di export: tanpa caption, tanpa penyaring, tanpa perbandingan di rule mana pun. Dibaca dan ditulis kembali apa adanya |
| **Arti `EXCLUSION_C`** | idem. Tetap diwajibkan karena bertanda `pyRequired=true` di layar Pega |
| **Pengganti hapus-lalu-sisip-ulang** | `ADR-0013` belum diputuskan; Work Owner memilih "as is" |
| **Pemberitahuan ke PIC** | seam Notifier (`S-3`) belum ada, dan penerima di rule lama berupa alamat pribadi yang `D-67` larang dibawa. Peristiwanya dicatat di log |
| **Unggah lampiran** | `PNCSaveAttachmentToDB` dan `SetUploadDocumentToPanelDocumentList` di luar lingkup modul ini; `DOKUMENID` baris lama dipertahankan apa adanya |
| **Constraint unik `NAME`** | menutup balapan penambahan sepenuhnya menuntutnya; DDL-nya belum ada (`R-08`), dan perubahan skema menempuh `D-63` |
| **Penyaring `cari` dipakai layar** | endpoint-nya siap; layar memakai pencarian bawaan `DataTable` seperti seluruh layar master lain. Perpindahannya dilakukan bersama paginasi sisi server (`TKT-U2-001`) |

---

## 22. Sesi kelima belas — Master Supplier (2026-09-20)

Modul kesebelas. Menggantikan `Harness/MasterSupplier-Harness.xml` (MENU_ID 29) atas
tabel **`M_SUPPLIER`** — form **24 isian, 15 di antaranya wajib**, tanpa tab.

> **Dikerjakan PARALEL dengan sesi keempat belas (Master Panel)** di working tree yang
> sama. Empat berkas bersama disunting dua pihak: `cmd/claimpnc/main.go`,
> `src/app/App.tsx`, `src/api/types.ts`, dan `src/app/menu/registry.ts`. Cara menghindari
> tabrakan dicatat di §22.11.

Ia **layar master pertama yang seluruh isinya tinggal di satu kolom JSON.** Sepuluh modul
sebelumnya membaca dan menulis kolom bernama.

### 22.1 Pertanyaan yang diajukan dan jawabannya

Tiga pertanyaan diajukan sebelum satu baris kode pun ditulis, masing-masing dengan
rekomendasi, pilihan lain, dan konsekuensinya.

| Pertanyaan | Jawaban Work Owner |
|---|---|
| `M_SUPPLIER` hanya punya `ID`, `OLDID`, `JSONDATA` — bagaimana Go menyimpannya? | *"Jalankan sebagai as is"* |
| Baris persetujuan `pooldata.proteksi_klaimmbu` — direplikasi atau tidak? | *"Jalankan sebagai as is"* |
| Daftar nilai lima dropdown tidak ada di export — bagaimana sementara ini? | *"Jalankan sebagai as is"* |

Ketiganya dibaca sebagai **`P-5` — replikasi perilaku lebih dulu**, dan terjemahannya
dinyatakan ke Work Owner sebelum pekerjaan dimulai:

| Hal | Yang dikerjakan |
|---|---|
| Penyimpanan | Bentuknya **tetap `M_SUPPLIER.JSONDATA`** dengan 25 kunci yang terbaca, dan ID berformat `situs + 11 digit`. Go menyusun dokumennya sendiri — `D-02` (tanpa memanggil procedure) **tidak dicabut sepihak** |
| Persetujuan | Baris `proteksi_klaimmbu` disisipkan **persis** seperti sekarang, termasuk syarat pada jalur Edit |
| Dropdown | **Tetap dropdown**, bukan diubah jadi teks. Pilihannya dari nilai yang terbukti di activity ditambah nilai yang benar-benar ada di data |

### 22.2 Yang dibaca dari export sebelum menulis kode

Harness-nya cangkang; isinya berlapis tiga:

```
Harness/MasterSupplier               341 KB · tombol New Supplier · Edit · Refresh
├─ Section/InboxMasterSupplier       grid 8 kolom, sumber ListMasterSupllier.pxResults
│  └─ Section/DataCountMasterSupllier  "Total Data :"
└─ Flow Action/CreateMasterSupplier  → Section/CreateMasterSupplier_Sec (24 isian)
   Flow Action/EditMasterSupplier    → section yang SAMA
```

Kedua Flow Action memakai **satu section yang sama**; yang berbeda hanya activity pre dan
post-nya.

| Berkas | Yang diambil darinya |
|---|---|
| `RDB List/GetDataEditMasterSupller-SQL.xml` | **25 kunci JSON**, terbaca satu per satu lewat notasi titik Oracle |
| `RDB List/KonversiMasterSupplier_SQL-SQL.xml` | pemanggilan `POOLDATA.PEGA_M_SUPPLIER` |
| `RDB List/InsertProteksiKlaimMBU_SQL-SQL.xml` | 10 kolom baris permintaan persetujuan |
| `Database/PEGA_M_SUPPLIER.prc` | bentuk ID: kode situs ditambah `lpad(supplier_seq.nextval, 11, '0')` |
| `Activity/CreateNewMasterSupplier_post-Act.xml` | `STS_AKTIF := "0"` · `SUPPLIER_HE` · `USERKLAIMID` · `TGL_INSERT` |
| `Activity/EditMasterSupplier_post-Act.xml` | `STS_AKTIF := STS_AKTIF_PROMLIST` · syarat persetujuan |
| `Activity/GetDataSupplier_pre-Act.xml` | penurunan `JENIS_STATUS` dari `SUPPLIER_HE` |
| `Section/CreateMasterSupplier_Sec-Section.xml` | 24 caption, 15 `pyRequired=true`, dan **satu `pyReadOnlyCondition`** |
| `Report Definition/BrowseCity_RD` · `BrowseBranchForGKM_RD` · `BrowseCountry_RD` · `BrowseBankGroup` | keempat lookup |

### 22.3 Aturan bisnis yang ditemukan, bukan dikarang

| Aturan | Buktinya |
|---|---|
| **Nama terkunci setelah tersimpan** | `pyReadOnlyCondition` pada isian NAMA — satu-satunya isian di form itu yang punya syarat read-only |
| **ID disembunyikan saat menambah** | `pyVisible: NOTBLANK`, dan `Read-only` |
| **Supplier baru lahir tidak aktif** | `CreateNewMasterSupplier_post` step 6 menetapkan `STS_AKTIF := "0"` tanpa syarat apa pun |
| **Saat disunting, status aktif menyusul pilihan** | `EditMasterSupplier_post` step 7: `STS_AKTIF := STS_AKTIF_PROMLIST` |
| **`SUPPLIER_HE` turunan, bukan isian** | step 7 menurunkannya dari `JENIS_STATUS`; `GetDataSupplier_pre` step 6.3 membalik arahnya |
| **Menonaktifkan berlaku SEKETIKA** | `EditMasterSupplier_post` step 12 hanya meminta persetujuan bila `STS_AKTIF_PROMLIST` bernilai `"1"` atau kosong |
| **Jejak pelaku tersimpan** | `USERKLAIMID := OperatorID.pyUserIdentifier` — berbeda dari `BENGKEL_HE` dan `PANEL_HE` yang tidak punya kolomnya |

Baris keenam pada tabel itu adalah **jalur satu-satunya di modul ini yang mengubah keadaan
tanpa melewati antrean mana pun**, dan layar menyebutkannya terang-terangan.

### 22.4 Lubang export yang harus dinyatakan (`R-16`)

| Yang hilang | Akibatnya |
|---|---|
| **Kueri daftar supplier** | Grid membaca `ListMasterSupllier.pxResults`; **tidak ada satu pun rule yang mengisinya**. Kolomnya terbaca, sumbernya tidak |
| **Seluruh sisi pemutus persetujuan** | Hanya `INSERT` ke `proteksi_klaimmbu` yang ada. Tidak ada yang membaca, menyetujui, menolak, atau memajukan `POSISI` |
| **Daftar nilai 5 dropdown** | `pxDropdown` bersumber `associated`; rule Field Value tidak ikut di export |
| `JENIS_STATUS_NOTE`, `STS_REKANAN_NOTE` | kolom label di grid, asalnya tidak diketahui |
| `@ASM.GetPageJSONString()` | badan fungsinya tidak ada — **tetapi kuncinya terbaca dari kueri bacanya**, sehingga ia tidak memblokir |

### 22.5 Kenapa modul ini MENULIS dokumen JSON, padahal Master Bengkel menolaknya

Master Bengkel menghadapi pertanyaan yang sama dan menjawabnya **berbeda**. Alasannya
bukan selera:

| | Master Bengkel | Master Supplier |
|---|---|---|
| Tabel berkolom bernama | `POOLDATA.BENGKEL_HE`, 40 kolom | **tidak ada** |
| Tabel JSON | `POOLDATA.M_BENGKEL_HE` | `M_SUPPLIER` |
| Nama kunci JSON | **tidak diketahui** — `GetPageJSONString` hanya tanda tangannya | **terbaca lengkap**, 25 kunci di kueri bacanya |
| Yang dipilih | tulis kolom, berhenti menulis JSON | **tulis JSON**, karena tidak ada pilihan lain |

Pilihan (a) pada banner `masterbengkel.sql` — "menulis dokumen JSON dengan SQL biasa" —
ditolak di sana karena **setiap kunci akan menjadi tebakan**. Di sini tidak ada satu pun
yang ditebak.

Notasi titik Oracle **tidak disalin**: ia tidak ada padanannya di PostgreSQL. Penggantinya
`JSON_VALUE`, yang berlaku di Oracle 12c+ maupun PostgreSQL 17+ — justru alasan `D-24`
mewajibkan versi 17. Dijaga uji `TestNoOracleDotNotation`.

### 22.6 Satu cacat sistem lama yang TIDAK dibawa

`CreateNewMasterSupplier_post` step 7 hanya punya cabang **"bila"**, tanpa cabang "selain
itu":

```
WHEN @equals(MasterSupplier.JENIS_STATUS,"1")  ->  SUPPLIER_HE := "1"
```

Ketika `JENIS_STATUS` **bukan** `"1"`, `SUPPLIER_HE` tidak pernah ditulis dan tetap
bernilai apa pun isinya sebelumnya. Akibatnya: **supplier HE yang diubah menjadi bukan-HE
tetap tersimpan sebagai HE**, dan saat dimuat kembali `GetDataSupplier_pre` mengembalikan
`JENIS_STATUS` menjadi `"1"` — perubahannya hilang tanpa satu pun tanda di layar.

Di sini kedua arah selalu ditulis (`DeriveHeavyEquipment`). **Selisih yang direncanakan**,
dan satu-satunya yang terlihat pengguna. Dijaga `TestSaveTurnsOffHeavyEquipment`.

### 22.7 Satu cacat lain yang ditemukan dan dikoreksi diam-diam

`Database/PEGA_M_SUPPLIER.prc:24` mengganti teks `UnknownID` di dalam dokumen dengan ID
yang baru terbit. **Tetapi tidak ada satu pun langkah di `CreateNewMasterSupplier_post`
yang pernah menaruh nilai itu ke halamannya.** Akibatnya kunci `ID` **di dalam dokumen
tersimpan kosong** sementara kolom `ID` terisi benar.

Sistem baru menulis ID yang sebenarnya ke keduanya. Tidak terlihat pengguna — layar
membaca kolomnya — tetapi salinan dokumen pada baris permintaan persetujuan kini dapat
menyebut supplier-nya sendiri.

### 22.8 Perubahan basis data, API, dan komponen

**Basis data.** Tidak ada migrasi. Dua tabel DITULIS (`M_SUPPLIER`,
`POOLDATA.PROTEKSI_KLAIMMBU`), enam objek hanya DIBACA (`M_BRANCH`, `CITY`, `COUNTRY`,
`GENERAL.LST_BANK_GROUP`, `POOLDATA.M_SITE_DATABASE`, `SUPPLIER_SEQ`). Dijaga uji
`TestOnlyOwnedTablesAreWritten`.

**API — sembilan rute baru**, seluruhnya di balik sesi DAN portal:

```
GET  /api/master/supplier            daftar (?cari=…)
POST /api/master/supplier            tambah
GET  /api/master/supplier/{id}       muat ke form
PUT  /api/master/supplier/{id}       simpan
GET  /api/master/supplier/cabang     lookup
GET  /api/master/supplier/kota       lookup (?cari=…, minimal 2 huruf)
GET  /api/master/supplier/negara     lookup
GET  /api/master/supplier/bank       lookup
GET  /api/master/supplier/sandi      kelima daftar dropdown sekaligus
```

Tidak ada `DELETE` — sistem lama tidak punya tombolnya, dan `D-66` melarangnya. Tidak ada
endpoint keputusan persetujuan — sisi pemutusnya tidak ada di export sama sekali.

**Frontend — empat berkas baru**, tanpa dependency baru:
`master-supplier/{api.ts, CityPicker.tsx, SupplierForm.tsx, SupplierPage.tsx}`.

**Mode periksa.** `claimpnc -periksa` bertambah `checkSupplier`, yang melaporkan lima hal —
termasuk satu yang khas modul ini, lihat §22.9.

### 22.9 Kendala teknis dan penyelesaiannya

| Kendala | Penyelesaian | Dampak |
|---|---|---|
| **`JSON_VALUE` menjawab NULL, bukan gagal**, bila kolomnya bukan JSON sah atau kuncinya dinamai lain | `checkSupplier` membandingkan jumlah seluruh baris dengan jumlah baris yang kunci `NAMA`-nya terbaca | Tanpa itu, layar menampilkan sederet baris berisi kolom kosong **tanpa satu pun galat**. Kelas kegagalan yang tidak dimiliki modul lain |
| **`PROTEKSI_ID` adalah ID work object Pega** yang tidak punya padanan | `ComposeApprovalID` membentuk `SUP.` + id supplier + waktu UTC | Asalnya terbaca, tertaut ke supplier-nya, dan penyimpanan berulang dalam detik yang sama ditolak basis data. **Bentuknya belum disetujui** — dicatat sebagai pertanyaan terbuka |
| **`UPDATE` mengganti SELURUH dokumen**, sehingga kunci yang tidak dikenal lenyap tiap simpan | `mergeDocument` membaca dokumen tersimpan `FOR UPDATE`, menimpa kunci yang dikenal, membiarkan sisanya | Kunci yang belum diketahui siapa pun tetap bertahan. Perlakuan yang sama dengan `DOKUMENID` di Master Bengkel |
| **Kolom `COUNTRY` dan tabel `COUNTRY` bernama sama** | alias `c` dipasang eksplisit | Tanpa alias, `ORDER BY COUNTRY` dapat terbaca sebagai nama tabel di sebagian basis data |
| **Kunci JSON `M_BRANCH` memakai penamaan berbeda** — `Name`, bukan `NAMA` | dibuktikan dari `GetBranchName-SQL` dan tujuh rule `SearchKlaimBy*_RDB` | Jalur JSON case-sensitive; salah satu huruf membuat dropdown Cabang kosong **tanpa galat** |
| **`TGL_INSERT` disimpan sebagai teks `dd/MM/yyyy`** | dipertahankan, diisolasi di `timestamp.go` | Utang `F-5` yang disadari. Dokumennya dibaca bersama Pega, sehingga bentuknya terikat |

### 22.10 Cacat nyata yang ditemukan uji, bukan oleh pembacaan

`ViolationDTO` mula-mula memakai nama field `isian` — nama yang lebih tepat artinya. Klien
bersama `api/client.ts` membaca nama isian dari **`field` atau `kolom` saja**, sehingga
`APIError.violations()` mengembalikan peta kosong.

Akibatnya nyata: pada form berisi **lima belas isian wajib**, pengguna diberi tahu "ada
isian yang belum benar" **tanpa satu pun petunjuk yang mana**. Uji frontend
"menyorot setiap isian yang ditolak server" yang menangkapnya.

Diperbaiki menjadi `kolom`, dan dijaga dari kedua sisi: backend
`TestCreateRejectsIncompleteInput` kini memeriksa nama fieldnya, dan uji frontend memakai
bentuk yang sama.

### 22.11 Bekerja paralel di working tree yang sama

Sesi keempat belas (Master Panel) berjalan bersamaan. Cara menghindari tabrakan pada empat
berkas bersama:

1. **Menambahkan, bukan menyisipkan.** Tipe frontend ditambahkan di akhir `types.ts`;
   butir menu di akhir `MENU_ROUTES`; bagian dokumen di akhir berkas.
2. **Membaca ulang tepat sebelum menyunting.** Dua suntingan gagal karena berkasnya sudah
   berubah — `registry.ts` dan `App.tsx` — dan keduanya diulang setelah dibaca ulang.
3. **Tidak menyentuh pekerjaan pihak lain.** Satu kegagalan build yang berasal dari
   `masterpanel` yang sedang setengah dirakit **dibiarkan**; ia konvergen sendiri dalam
   hitungan detik.

### 22.12 Hasil uji

| Suite | Hasil |
|---|---|
| `go build ./...` | lulus |
| `go vet ./internal/mastersupplier/... ./cmd/...` | bersih |
| `gofmt -l internal/mastersupplier` | bersih |
| `go test ./...` | **seluruhnya lulus** |
| `npx tsc --noEmit` | bersih |
| `npx vitest run src/modules/master-supplier` | **22 lulus** |
| `npx vitest run` (seluruh frontend) | 181 lulus, **3 gagal** |

**Ketiga kegagalan itu pre-existing** dan seluruhnya di
`master-rekening/AccountPage.test.tsx` — modul yang **tidak tersentuh sesi ini**,
dibuktikan `git status` yang menunjukkan folder `master-rekening/` dan `components/`
bersih. Ketiganya sudah tercatat pada §17.11, §18.11, §20.11, dan §21 dengan jumlah yang
sama persis.

### 22.13 Yang belum dikerjakan, dan kenapa

| Hal | Alasan |
|---|---|
| **Jalur tulis di produksi** | menunggu DBA memastikan lebar dan tipe kolom `JSONDATA`. Seluruh 28 nilai masuk ke satu kolom, sehingga kolom yang terlalu sempit menolak **baris utuh** — bukan isian yang kepanjangan |
| **Layar pemutus persetujuan** | sisi pemutus `proteksi_klaimmbu` **tidak ada di export sama sekali**. Membangunnya berarti mengarang aturan yang menentukan supplier mana yang boleh dipakai |
| **Bentuk `PROTEKSI_ID`** | awalan `SUP.` dipilih agar asalnya terbaca; apakah ia diterima, atau permintaan dari aplikasi baru harus memakai sequence tersendiri, **belum diputuskan** |
| **Daftar nilai 5 dropdown** | rule Field Value tidak ikut di export. Layar menawarkan gabungan nilai yang terbukti dan yang ada di data, dan **menyatakan** bahwa daftarnya belum diterima |
| **`OLDID` tidak pernah ditulis** | tidak ada satu pun rule di export yang mengisinya. Dibaca dan ditampilkan; menulisinya berarti menebak apa gunanya |
| **Permintaan persetujuan belum satu transaksi dengan penyimpanan** | menyatukannya menuntut transaksi yang dipegang lapisan aplikasi — perubahan bentuk seam yang menyentuh seluruh modul master. Sistem lama bahkan lebih longgar: procedure-nya `COMMIT` dua kali sebelum baris permintaan disisipkan |
| **Constraint unik nama** | namanya tersimpan **di dalam dokumen JSON**, sehingga constraint unik menuntut index berbasis fungsi lebih dulu. Keduanya menempuh `R-08` dan `D-63` |
| **`LookupPicker` bersama** | syarat menaikkannya — adanya modul ketiga — **kini terpenuhi**. Lihat `keputusan-implementasi.md` §23.7 |

### 22.14 Paginasi sesuai Pega — ditambahkan menyusul pada hari yang sama

Diminta setelah modulnya selesai: *"buatkan pagination sesuai pega"*.

#### Setelan aslinya dibaca dari export, bukan dipilih sendiri

`Section/InboxMasterSupplier-Section.xml` menyisipkan `pyGridPaginator` di kaki grid,
dengan setelan berikut pada `pyGridProps`-nya:

| Setelan | Nilai | Artinya |
|---|---|---|
| `pyPageMode` | `Numeric` | **nomor halaman**, bukan tombol "muat lebih banyak" |
| `pyPageSize` | `20` | dua puluh baris per halaman |
| `pyPaginationButtonsFormat` | `Standard` | tombol baku Pega |
| `pyPageListProperty` | `ListMasterSupllier.pxResults` | sumbernya **page list klipboard** |
| gaya sel paginator | `dataLabelRead gridActionAlignRight` | **rata kanan** |

#### Ukuran halaman TIDAK seragam antarlayar

Ini yang menentukan bentuk implementasinya. Sembilan layar master yang sudah dibangun
memakai dua angka yang berbeda — terbaca dari `pyPageSize`, atau dari `pyPageSizeOther`
bila nilainya `"Other"`:

| Layar | Ukuran halaman | Mode |
|---|---|---|
| **Master Supplier** | **20** | Numeric |
| Master Bengkel | 20 | Numeric |
| Master Panel | 20 | Numeric |
| Master Rekening | 15 | Numeric |
| Master Status Klaim | 15 | Numeric |
| Master Status Progres | 15 | Numeric |
| Master Pasal Kerugian | 15 | Numeric |
| Master Penolakan Klaim | 15 | Numeric |
| Master Auto Claim | — | tidak punya paginator |

Master Supplier satu-satunya yang menuliskan angkanya langsung di `pyPageSize`; delapan
sisanya memakai `"Other"` dengan angkanya di `pyPageSizeOther`.

**Tidak ada satu angka yang benar untuk semuanya.** Karena itu ukuran halaman menjadi
**prop per layar**, bukan angka tetap di dalam komponen.

#### Bentuk yang dipilih: opt-in, supaya modul selesai tidak berubah

`DataTable` adalah komponen bersama yang dipakai **sembilan layar master yang sudah
dinyatakan selesai**. Menyalakan paginasi sebagai perilaku bawaan akan mengubah
kesembilannya sekaligus — dan Master Status Klaim yang berisi 33 baris akan langsung
terpotong menjadi dua halaman, beserta ujinya yang gagal.

Itu melanggar **Isolasi Protektif**. Yang dipakai sebagai gantinya:

```
pageSize?: number     tidak diisi  →  tanpa paginasi, persis seperti sebelumnya
                      diisi 20     →  paginasi numerik 20 baris per halaman
```

Hanya `SupplierPage` yang mengisinya. Kesembilan layar lain tidak disentuh satu baris pun,
dan **ujinya seluruhnya tetap lulus** — itu buktinya, bukan janjinya.

Uji `tanpa paginasi > menggambar SELURUH baris bila pageSize tidak diisi` sengaja ditulis
untuk menjaga bawaan itu: bila kelak seseorang mengubahnya menjadi berpaginasi, uji **di
komponen** yang gagal — bukan uji milik modul yang sudah selesai.

#### Paginasi berjalan SESUDAH pencarian dan pengurutan

Urutannya menentukan, dan salah urutan menghasilkan cacat yang sulit dikenali: memotong
halaman lebih dulu akan membuat pencarian hanya menemukan baris yang kebetulan ada di
halaman yang sedang dibuka.

Urutan yang dipakai — saring, urutkan, baru potong — juga yang ditiru dari Pega: gridnya
terikat pada page list klipboard, sehingga paginatornya memotong daftar yang **sudah**
tersaring, bukan meminta halaman berikutnya ke server.

Dua akibat yang ikut diurus:

- **Kata kunci berubah → kembali ke halaman pertama.** Halaman ketiga daftar lama hampir
  pasti tidak ada pada daftar baru.
- **Urutan berubah → kembali ke halaman pertama.** Mengurutkan ulang menyusun ulang
  seluruh daftar; halaman ketiga tidak lagi memuat baris yang sama.

#### Halaman dijepit saat menggambar, bukan lewat efek

Baris dapat berkurang di luar kendali komponen — penyaring dipersempit, atau daftarnya
dimuat ulang setelah sebuah baris berpindah. Halaman yang sudah tidak ada karena itu
menampilkan **halaman terakhir**, bukan tabel kosong tanpa penjelasan.

Dikerjakan dengan `Math.min` saat menggambar, bukan dengan `useEffect` yang menyetel
state: efek menghasilkan satu gambar perantara yang kosong lebih dulu, dan itu terlihat
sebagai kedipan.

#### Yang DITAMBAHKAN terhadap Pega

**Ringkasan baris** — "Menampilkan 1–20 dari 57 baris". Pega hanya menggambar nomor
halamannya. Tanpa ringkasan itu, nomor halaman tidak memberi tahu apa pun tentang seberapa
banyak yang belum dilihat — pada tabel yang baru saja disaring, justru itulah yang ingin
diketahui.

Pencacah "Total Data :" milik `Section/DataCountMasterSupllier` **tetap ada** di kepala
layar dan menghitung hal yang berbeda: seluruh baris yang dimuat, bukan yang tersaring.

**Sela `…` pada daftar panjang.** Nomor yang digambar dibatasi: halaman pertama, terakhir,
dan halaman sekarang beserta tetangganya. Tanpa itu, daftar seribu baris menggambar lima
puluh tombol nomor yang membungkus beberapa baris — dan justru membuat halaman yang sedang
dibuka sulit ditemukan. Aturannya di `pageWindow`, diuji terpisah.

#### Yang TIDAK dikerjakan

**Paginasi keyset sisi server.** Itu `TKT-U2-001`, dan Steering menyebutnya **perubahan
perilaku, bukan pemeliharaan** — `09-DATABASE-STRATEGY.md` §6.3 dan `15-NFR` mencatat
bahwa grid lama memang memotong di klipboard, bukan memaginasi di server. Layar berpuluh
juta baris menuntutnya; layar master yang berbaris puluhan tidak.

Endpoint `GET /api/master/supplier?cari=` sudah menerima penyaring, sehingga perpindahan ke
sisi server kelak tidak menuntut kontrak baru — hanya menambah `cursor` dan `limit`.

**Menyalakan paginasi pada delapan layar master lain.** Setelan Pega-nya sudah terbaca dan
tercatat di tabel di atas, sehingga menyalakannya cukup satu prop per layar. Ia tidak
dikerjakan di sini karena menyentuh modul yang sudah dinyatakan selesai — keputusan Work
Owner, bukan keputusan sesi ini.

#### Hasil uji setelah penambahan

| Suite | Hasil |
|---|---|
| `npx tsc --noEmit` | bersih |
| `npx vitest run src/components` | **19 lulus** — berkas uji baru `DataTable.test.tsx` |
| `npx vitest run src/modules/master-supplier` | **25 lulus** (22 + 3 uji paginasi) |
| `npx vitest run` (seluruh frontend) | 205 lulus, **3 gagal** |

Ketiga kegagalannya **tetap yang itu-itu juga** — `master-rekening/AccountPage.test.tsx`,
pre-existing, jumlahnya tidak berubah. Kesembilan layar master lain yang memakai
`DataTable` **seluruhnya tetap lulus**, dan itulah bukti bahwa bawaan opt-in tidak
mengubah satu pun di antaranya.

---

## 23. Koreksi Master Panel — layar diselaraskan dengan Pega (2026-09-20)

Putaran umpan balik setelah §21. Ditulis terpisah supaya §21 tetap merekam apa yang
dikerjakan saat itu, bukan disunting menjadi seolah tidak pernah menyimpang.

### 23.1 Umpan balik yang diterima

Work Owner mengirim **tangkapan layar Master Panel HE yang sebenarnya** beserta satu
kalimat: *"Saya ingin tampilannya konsisten dengan pega seperti ini."*

Tangkapan layarnya memperlihatkan empat hal yang berbeda dari layar yang dibangun, dan
keempatnya diperiksa ulang ke export sebelum satu baris pun diubah — bukan diterima
begitu saja, bukan pula dibantah.

### 23.2 Yang diperiksa, dan hasilnya

| Yang diperiksa | Perintah | Hasil |
|---|---|---|
| Judul layar | `pyCaption` pada `Section/ListPanelHE` | **"Master Panel HE"** — yang dibangun "Master Panel" |
| Urutan tab | urutan kemunculan section pada `Section/BrowsePanelHE` | offset 102256 Approve · 137949 Reject · 183023 Approval — **Reject di tengah** |
| Caption tab | `pyCaption` pada section yang sama | Approve · Reject · Waiting Approval |
| Tombol layar | `pyButtonLabel` pada `ListPanelHE` dan `BrowsePanelHE` | Tambah · Refresh · **Upload Document · Upload Data Master Panel · Upload Data Lokasi Panel** |
| Urutan baris | `pySortType` pada report definition | **DESC** atas `ID_PANEL` |

Kelimanya cocok dengan tangkapan layar. Tidak ada satu pun yang menuntut tafsiran.

### 23.3 Akar penyimpangannya — satu kebiasaan, empat akibat

Keempatnya berasal dari hal yang sama: **memperbaiki tata letak yang dianggap sulit
dibaca, alih-alih menirunya.** Alasannya bahkan tertulis apa adanya di kode yang dibangun
§21:

> *"Grid Pega menampilkan kesebelas kolomnya sekaligus. Itu tidak ditiru seluruhnya:
> sembilan penanda berdampingan menjadi deretan sandi yang tidak dapat dibaca mata."*

Alasan itu masuk akal sebagai desain dan **salah sebagai migrasi**. `D-13` menuntut tata
letak yang **sama**, bukan yang lebih baik. Rinciannya di `keputusan-implementasi.md` §24.2.

### 23.4 Yang dikerjakan

| Berkas | Perubahan |
|---|---|
| `PanelPage.tsx` | judul; urutan tab; **11 kolom grid dikembalikan berdampingan**; kolom "Lokasi" dan "Penanda" dibuang; 3 tombol unggah ditambahkan dalam keadaan mati; `LocationSummary` dihapus |
| `masterpanel.sql` | `ORDER BY NAME` → `ORDER BY ID_PANEL DESC`, dua kueri |
| `repo/memory/memory.go` | pengurutan memori mengikuti |
| `query_test.go` · `manage_test.go` | dua uji baru mengunci arah urutan di kedua adapter |
| `PanelPage.test.tsx` | judul, kesebelas kolom, urutan tab, ketiadaan kolom Lokasi, ketiga tombol unggah |

### 23.5 Kendala teknis

| Kendala | Penyelesaian |
|---|---|
| `go test` melaporkan **`(cached)`** padahal `.sql` dan `memory.go` baru diubah | Dijalankan ulang dengan `-count=1`. Hasil ber-cache tidak membuktikan perubahan sudah teruji — keempat paket hijau setelah dipaksa |
| `README.md` berubah di tengah pengerjaan | Sesi Master Supplier menyuntingnya paralel. **Tidak disentuh**; perubahan sendiri tetap ditambahkan di bagiannya |

### 23.6 Hasil pemeriksaan

| Perintah | Hasil |
|---|---|
| `go build ./...` | **hijau** |
| `go test -count=1 ./internal/masterpanel/...` | **hijau**, keempat paket |
| `gofmt -l internal/masterpanel` | **bersih** |
| `npm run typecheck` | **hijau** |
| `vitest run src/modules/master-panel` | **hijau**, **18 uji** — naik dari 16 |

### 23.7 Yang tetap terbuka

Tidak bertambah dan tidak berkurang dari §21.10. Koreksi ini menyentuh **tata letak saja**;
jalur tulis tetap menunggu dua kepastian DBA yang sama — apakah `PANEL_HE` tabel atau view,
dan apakah `LOKASI_PANEL_HE.NAMA` memuat hal yang sama dengan `LOKASI_PANEL`.

Satu hal yang **bertambah**, dan ia kecil: ketiga tombol unggah kini terlihat di layar
dalam keadaan mati. Rule-nya tetap tidak ada di export, jadi yang berubah bukan
kemampuannya melainkan **keterbacaan ketiadaannya**.

---

## 24. Koreksi Master Status Progres 1 — paginasi dan isi dropdown Posisi (2026-09-20)

Putaran umpan balik atas modul yang dibangun §9 dan dibawa ke standar baru §14. Ditulis
terpisah supaya kedua bab itu tetap merekam apa yang dikerjakan saat itu — termasuk
kesimpulan yang ternyata keliru — bukan disunting menjadi seolah tidak pernah salah.

### 24.1 Umpan balik yang diterima

Work Owner mengirim tangkapan layar dialog **"Memperbaharui Data"** milik layar Pega yang
sedang berjalan, beserta dua kalimat:

> *"kenapa tidak ada pagination seperti pada pega? dan untuk dropdown posisi tidak
> selengkap di pega"*

Keduanya diperiksa ke export lebih dulu sebelum satu baris pun diubah. **Keduanya benar**,
dan yang kedua menyingkap kesalahan pembacaan saya sendiri — bukan sekadar daftar yang
kurang panjang.

### 24.2 Keluhan pertama — paginasi

| Yang diperiksa | Hasil |
|---|---|
| `Section/BrowseStatusProgress-Section.xml` | memuat `pyGridPaginator` |
| `pyPageSize` | `Other` |
| `pyPageSizeOther` | **15** |
| `pyPaginationButtonsFormat` | `Standard` (nomor halaman, bukan "muat lebih banyak") |

Grid layar lama **berhalaman 15 baris**. Layar yang dibangun menggambar seluruh baris
sekaligus.

Komponen `DataTable` sendiri **sudah mendukung paginasi** — ia ditambahkan saat modul
Master Supplier dikerjakan (§22), dengan `pageSize` sebagai prop opt-in karena ukuran
halaman berbeda-beda per layar. Yang kurang hanyalah **layar ini tidak pernah
menyalakannya**. Perbaikannya satu prop: `pageSize={15}`.

### 24.3 Keluhan kedua — dan kesalahan saya yang ia singkap

Tangkapan layarnya memperlihatkan dropdown Posisi berisi **delapan** pilihan:

	All · REGISTER · KOMITE · SURVEY · AKSEPTASI · OUTSTANDING · BENGKEL · PROCUREMENT

("All" muncul dua kali — paling atas dalam keadaan terpilih, dan paling bawah.)

Yang dibangun hanya memuat **empat**, dan memuatnya sebagai **pasangan kode→label**:

	"002" REGISTER · "004" SURVEY · "006" KOMITE · "007" AKSEPTASI

Jadi kesalahannya **dua**, bukan satu:

| # | Yang saya tulis | Yang sebenarnya |
|---|---|---|
| 1 | empat pilihan | **delapan** |
| 2 | yang tersimpan di kolom `STATUS` adalah **kode angka** | yang tersimpan adalah **teksnya** |

Butir 2 lebih berat daripada butir 1: seluruh baris yang ditambah lewat layar baru akan
menulis `"002"` ke kolom yang di seluruh sistem lama berisi `"REGISTER"`. Tidak ada galat
yang muncul, dan barisnya tetap tersimpan — ia hanya tidak akan pernah cocok dengan apa
pun yang membacanya.

**Bukti untuk butir 2**, dari export:

Sel "Posisi" pada `Section/BrowseStatusProgress-Section.xml` mengikat dropdown-nya begini:

	pySourceName = TempPosition.pxResults   (kelas Code-Pega-List)
	pyValue      = .CaseID
	pyPrompt     = .CaseID

`pyValue` (yang disimpan) dan `pyPrompt` (yang ditampilkan) menunjuk **properti yang
sama**. Dan `.CaseID` berisi teks, bukan kode — `Activity/ViewStatusProgress_act-Act.xml`
mengisinya dengan literal:

	TempPositionClaim.pxResults(<APPEND>).CaseID := "AKSEPTASI"

### 24.4 Dari mana empat kode angka itu tadinya saya ambil

Dari activity yang **sama**, tetapi dari **pasangan properti yang berbeda** — deretan
angka di sebelahnya, yang mengisi dropdown layar lain. Saya membacanya sebagai pasangan
kode→label untuk layar ini **tanpa memeriksa ikatan `pyValue` sel-nya lebih dulu**.

Itu kesalahan pembacaan, bukan perbedaan data. Polanya sama dengan yang sudah tercatat di
`16-RISK-ANALYSIS.md` `R-06`: **arti sebuah nilai tidak boleh disimpulkan dari
pemakaiannya di tempat lain** — di sana tiga arti kode status yang disimpulkan dari
pemakaian ternyata seluruhnya salah.

Pelajaran yang diambil, dan sudah ditulis sebagai komentar di `position.go` supaya tidak
hilang: **sebelum memakai sebuah daftar nilai, periksa dulu ikatan `pyValue` sel yang
memakainya** — bukan hanya menelusuri dari mana nilainya berasal.

### 24.5 Kenapa daftarnya berasal dari tangkapan layar, bukan dari export

Rule yang mengisi `TempPosition` untuk harness ini **tidak ada di export**. Dicari ke
seluruh `Activity/`, `Section/`, `RDB List/`, dan `Data Transform/`; kedelapan nilainya
tidak pernah muncul bersama di satu tempat mana pun.

Ini persis kelas kekurangan yang dicatat `R-16` — ±242 rule dirujuk tetapi tidak ikut
diekspor. Karena artefaknya memang tidak ada, sumber yang sahih adalah **layar Pega yang
sedang berjalan**, dan itulah yang diberikan Work Owner.

Yang dipegang: daftarnya disalin apa adanya beserta urutannya, tidak diurutkan ulang dan
tidak "dirapikan". Ini migrasi, dan `D-13` menetapkan tata letaknya ditiru.

### 24.6 Yang berubah di kode

| Berkas | Perubahan |
|---|---|
| `internal/masterstatusprogres/position.go` | daftar menjadi **8 nilai teks**; `Code` dan `Name` bernilai sama, dan alasannya ditulis lengkap |
| `internal/masterstatusprogres/masterstatusprogres.go` | `Input.Clean` **tidak lagi meng-huruf-besarkan** posisi; ia **membakukan ejaan** lewat `FindPosition` |
| `internal/masterstatusprogres/http/dto.go` | keterangan `kode_posisi` dan `nama_posisi` disesuaikan; **bentuk kontraknya tidak berubah** |
| `internal/masterstatusprogres/repo/memory/memory.go` | baris contoh memakai nilai teks |
| uji backend (3 berkas) | nilai contoh, panjang daftar, dan urutannya |
| `modules/master-status-progres/ProgressStatusPage.tsx` | `pageSize={15}`; kolom Posisi tidak lagi menyandingkan label dengan kodenya |
| `modules/master-status-progres/ProgressStatusPage.test.tsx` | daftar contoh menjadi 8; **uji baru** yang menjaga paginasi tidak hilang lagi |

**Kenapa `Input.Clean` ikut berubah.** Ia memakai `strings.ToUpper`, dan itu benar selama
nilai simpanannya kode angka. Sejak nilainya menjadi teks, salah satu di antaranya
ber-ejaan campuran: **"All"**. Meng-huruf-besarkan akan menyimpan "ALL" — nilai yang tidak
sama dengan apa pun yang pernah ditulis sistem lama. Penggantinya membakukan ejaan ke
baris daftar yang cocok, sehingga "register" maupun "REGISTER" sama-sama tersimpan sebagai
"REGISTER".

**Kenapa bentuk kontrak API sengaja TIDAK diubah.** Nama `kode_posisi` kini memuat teks,
bukan kode, dan menggantinya menjadi `posisi` sempat dipertimbangkan. Tidak diambil:
`Code` dan `Name` yang bernilai sama bukan kebetulan yang akan hilang — ia bentuk
dropdown-nya di sistem lama — dan memisahkan keduanya tetap berguna bila kelak daftarnya
pindah menjadi master data `F-4` dengan label yang dibedakan dari nilai simpanannya.
Mengubah nama field berarti merusak klien demi penamaan yang lebih rapi, dan itu tidak
sepadan.

### 24.7 Hasil pemeriksaan

| Perintah | Hasil |
|---|---|
| `go build ./...` | **hijau** |
| `go vet ./...` | **hijau** |
| `go test ./...` | **hijau**, seluruh paket |
| `gofmt -l internal/masterstatusprogres` | hanya berkas ber-CRLF bawaan repo; berkas yang disunting tidak menambah daftarnya |
| `tsc --noEmit` | **hijau** |
| `vitest run src/modules/master-status-progres` | **hijau**, **32 uji** — naik dari 31 |
| `npm run build` | **hijau** |

**Satu berkas uji GAGAL di luar lingkup ini**, dan sengaja tidak disentuh:
`src/modules/master-rekening/AccountPage.test.tsx` — 3 uji merah, seluruhnya tentang
tombol Approve/Reject komite. Modul Master Rekening berada di bawah **Isolasi Protektif**,
dan tidak satu pun berkas yang saya ubah diimpor olehnya. Kegagalannya dilaporkan apa
adanya, bukan diperbaiki sepihak.

### 24.8 Dua hal yang masih perlu dipastikan Work Owner

Keduanya sudah ditulis sebagai komentar di `position.go`, dan **tidak satu pun saya tebak
sendiri**:

1. **Apakah "All" memang nilai yang boleh tersimpan** di kolom `STATUS`? Ia muncul dua
   kali di dropdown — paling atas (terpilih) dan paling bawah — dan itu bentuk yang biasa
   dipakai untuk **pilihan penyaring**, bukan nilai simpanan. Di sini ia ditulis sekali,
   dan tetap dapat dipilih.
2. **"PROCUREMENT" atau "PROCUREMENT/SUPPLIER"?** Layar menulis yang pertama, tetapi
   `Section/ApprovalProgressKlaim-Section.xml` membandingkan nilai sejenis dengan
   "PROCUREMENT/SUPPLIER". Bila yang tersimpan sebenarnya berakhiran "/SUPPLIER",
   perbandingan di layar persetujuan itu **tidak akan pernah cocok** untuk baris yang
   ditambah lewat layar ini — dan seperti butir 2 pada §24.3, kegagalannya senyap.

### 24.9 Kedua pertanyaan §24.8 dijawab — "Sesuaikan saja dengan Pega"

Jawaban Work Owner, 2026-09-20. Keduanya diperiksa ulang ke export sebelum diterapkan,
bukan diterima begitu saja.

**Butir 2 — "PROCUREMENT", tanpa "/SUPPLIER". Kekhawatirannya tidak berdasar.**

Teks `PROCUREMENT` dicari ke seluruh export. Ia hanya muncul di **empat berkas**, dan
setiap perbandingan dengan `'PROCUREMENT/SUPPLIER'` ternyata menyangkut **properti lain**:

| Berkas | Properti yang dibandingkan |
|---|---|
| `Section/ApprovalProgressKlaim-Section.xml` | `.AlasanTerlambat` · `.SurveyorAddrress` |
| `Section/DetailKlaimCabang_Sect-Section.xml` | `.AlasanTerlambat` · `.SurveyorAddrress` |
| `Harness/View_DetailKlaimCabang_Harness-Harness.xml` | `.AlasanTerlambat` · `.SurveyorAddrress` |
| `Activity/DownloadPrinftPerhitunganPLADLAXOLSummary-Act.xml` | bukan perbandingan — nama rule `DATAPROCUREMENT` |

Tidak ada satu pun tempat yang membandingkan **posisi klaim** dengan `/SUPPLIER`. Nilai
yang dipakai layar ini tetap `"PROCUREMENT"`, dan tidak ada yang perlu diubah.

**Butir 1 — "All" memang muncul dua kali, dan pengulangannya direplikasi.**

Dugaan bahwa baris "All" pertama adalah baris kosong bawaan Pega yang berkapsi "All"
**gugur**: sel dropdown-nya memuat

	pyHasNoSelection = false

yang berarti Pega **tidak menyisipkan baris kosong apa pun**. Jadi dropdown-nya memang
berisi **sembilan baris** dengan **delapan nilai berbeda** — "All" benar-benar terulang di
datanya sendiri.

Karena Work Owner meminta disesuaikan dengan Pega, pengulangannya **direplikasi**.
Pengulangan itu tidak mengubah apa yang tersimpan — kedua baris bernilai sama — sehingga
biayanya nol dan yang diperoleh adalah layar yang berisi baris persis sama dengan layar
lama (`D-13`). Menghapus baris kedua adalah perubahan satu baris di `position.go` bila
kelak dikehendaki.

**Satu berkas bersama ikut berubah karenanya.** `components/SelectField.tsx` mengunci
setiap `<option>` dengan **nilainya**, dan nilai kembar membuat React menemui dua kunci
yang sama dalam satu daftar — memicu peringatan sekaligus penggambaran ulang yang tidak
dapat diandalkan. Kuncinya diganti menjadi berbasis **posisi**. Perubahan ini
**tidak mengubah perilaku** satu layar pun: urutan daftar ditentukan server dan tidak
pernah disusun ulang di layar. Seluruh uji sembilan layar yang memakainya tetap hijau.

**Satu penyimpangan dari Pega yang sengaja DIPERTAHANKAN**, dan disebut terang-terangan
supaya bukan temuan berikutnya: `SelectField` menyisipkan baris kosong `— pilih —` di
puncak daftar, sedangkan Pega tidak (`pyHasNoSelection = false`) dan langsung memilih
baris pertama. Baris kosong itu dipertahankan karena tanpa ia, penambahan baris baru akan
**menyimpan "All" tanpa pengguna pernah memilihnya**. Bila Work Owner menghendaki
perilaku Pega apa adanya, yang berubah satu prop di layar — bukan komponennya.

**Hasil pemeriksaan ulang**

| Perintah | Hasil |
|---|---|
| `go build ./...` · `go vet ./...` | **hijau** |
| `go test -count=1 ./internal/masterstatusprogres/...` | **hijau**, keempat paket |
| `tsc --noEmit` | **hijau** |
| `vitest run src/modules/master-status-progres src/components` | **hijau**, 52 uji |
| `vitest run` (seluruhnya) | 209 hijau · **3 merah** — ketiganya `master-rekening/AccountPage.test.tsx`, sama persis seperti sebelum perubahan ini |
| `npm run build` | **hijau** |

### 16.12 ID induk dicabut dari grid — kesalahan yang sama untuk ketiga kalinya

Work Owner mengirim tangkapan layar: kolom **Status Progres 1** menggambar
`LAPORAN AWAL  001` — nama induk, lalu ID induk sebagai teks abu-abu kecil di sebelahnya.
Pertanyaannya: *"kenapa terlihat seperti ini?"*

**Jawabannya: itu tambahan saya, bukan perilaku sistem lama.** Kueri dan pengikatan gridnya
berbunyi tegas:

```sql
-- BrowseStatusProgress2-SQL.xml
SELECT ID_MST        AS "CaseID",
       STS_PROGRESS1 AS "City",      -- kolom "Status Progress 1"
       STS_PROGRESS2 AS "CityID",
       ID_PROGRESS   AS "District",  -- ikut di-SELECT ...
       TIPE          AS "DistrictID"
  FROM POOLDATA.GCNM_MST_PROGRESS
```

Properti yang benar-benar terikat ke **baris grid** hanya tiga — `.CaseID`, `.City`, `.CityID`.
`District` (ID induk) dan `DistrictID` (TIPE) ikut di-`SELECT`, **tetapi keduanya dipakai modal
penyuntingan**, bukan digambar di grid. Karena modal itu tidak dibawa, keduanya tidak punya
tempat di layar ini.

ID-nya **tetap ikut ke `value`**, bukan ke `render`. `value` adalah yang dipakai pencarian dan
pengurutan, sehingga petugas yang hafal kode induk tetap dapat menemukan barisnya — tanpa satu
karakter pun bertambah di layar. Satu uji baru membuktikan pencarian `02` masih menemukan
barisnya meski `02` tidak tampak di mana pun.

#### Polanya sudah tidak dapat disebut kebetulan

Ini **ketiga kalinya** pada modul yang sama, dengan bentuk yang persis sama:

| # | Yang saya tambahkan | Yang Pega punya |
|---|---|---|
| 1 | Nama baris ditaruh sebelum nama induk | induk lebih dulu |
| 2 | Kolom `Tipe` | tidak ada — `DistrictID` milik modal |
| 3 | ID induk menempel di nama induk | `.City` saja — nama induk |

Ketiganya lolos dari kompilasi, dari `tsc`, dan dari seluruh uji — karena ujinya saya tulis
sendiri terhadap kolom yang saya rancang sendiri. Ketiganya juga punya alasan yang terdengar
masuk akal saat ditulis. Dan ketiganya ditemukan **hanya karena Work Owner melihat layarnya**.

> Yang menyatukan ketiganya: saya membaca **kueri** untuk menentukan tampilan. Kueri
> memberitahu data apa yang tersedia; ia **tidak** memberitahu data mana yang digambar.
> Yang memberitahu itu adalah **pengikatan `rowdata` pada section** — dan di sana ketiga
> kekeliruan ini terbaca harfiah, tanpa perlu ditebak sama sekali.
>
> Aturan yang dipakai sejak sekarang: **satu kolom hanya boleh digambar bila ada
> `<rowdata>` yang mengikatnya ke properti baris.** Kolom yang hanya muncul di klausa
> `SELECT` bukan kolom layar.

#### Satu pengamatan dari data nyata, dan itu menyangkut layar TINGKAT 1

Tangkapan layar itu memperlihatkan data yang **bukan dari contoh di repo** — contohnya memakai
`01` / `DOKUMEN DITERIMA`, layarnya `001` / `LAPORAN AWAL`. Artinya ia dijalankan di atas data
yang sebenarnya, dan `ID_PROGRESS` di sana berlebar **tiga karakter**.

Sementara itu `masterstatusprogres.FormatID` — milik **tingkat 1** — berbunyi:

```go
func FormatID(sequence int) string { return "0" + strconv.Itoa(sequence) }
```

sehingga baris baru akan menerima `01`, bukan `001`.

**Ini pengamatan, bukan kesimpulan.** Yang dapat saya pastikan hanya: kedua bentuk itu berbeda
lebarnya. Apakah itu berakibat — apakah `ID_PROGRESS` produksi memang selalu tiga karakter,
dan apakah baris baru ber-`01` akan menyulitkan — **tidak dapat saya jawab tanpa melihat isi
tabelnya**, dan isi itu belum ada di repo (`R-08`: DDL pun belum diterima).

Layar tingkat 1 berada di bawah **Isolasi Protektif**, sehingga tidak saya sentuh. Dicatat di
sini supaya dapat diperiksa Work Owner terhadap data produksi.

---

### 16.13 Penyuntingan ditambahkan — keputusan yang dibalik dalam satu sesi

Beberapa saat setelah §16.12 selesai, Work Owner bertanya: *"Dimana button ubah? kenapa tidak
muncul?"*

Pertanyaan itu ditujukan pada keputusan yang **baru saja ia setujui sendiri** — pada pertanyaan
sebelumnya ia memilih *"Jangan ditambahkan"*. Pembalikan ini bukan inkonsistensi; ia terjadi
karena yang ditanyakan pertama adalah *"apakah mereplikasi tombol Pega"*, sedangkan yang
ditanyakan kedua adalah *"apakah baris yang salah ketik dapat diperbaiki"*. Dua pertanyaan
berbeda, dan jawaban yang benar untuk keduanya memang berlawanan.

#### Apa yang sebenarnya dilakukan tombol Pega

Bukti dari export, tidak berubah sejak §16.11:

```sql
-- UpdateStatusProgress2_sql-SQL.xml
UPDATE POOLDATA.GCNM_PROGRESS_CLAIM C
   SET JSONSTATUS_PROGRESS2 = {TempInputStatusProgress2.City}
 WHERE PNCCASEID = {tempSearchProgress.AnalystDoctorRemaks}
   AND ID_UPDATE = {tempSearchProgress.Email}
```

Tiga hal sekaligus membuat pernyataan ini tidak pernah menyentuh tabel master:

| Hal | Bukti |
|---|---|
| Tabelnya **lain** | `GCNM_PROGRESS_CLAIM` — catatan progres satu klaim, bukan `GCNM_MST_PROGRESS` |
| Page sumbernya **tidak pernah diisi** | `TempInputStatusProgress2` muncul di **tepat satu berkas** di seluruh export: berkas SQL ini sendiri |
| Page penyaringnya **milik layar lain** | `tempSearchProgress` hanya diisi di `InputProgress-Harness`, sementara form ini mengikat `TempUpdateStatus2` |

Dan pencarian menyeluruh: **nol** `UPDATE` maupun `DELETE` terhadap `POOLDATA.GCNM_MST_PROGRESS`
di seluruh export.

#### Kenapa "tidak ada penyuntingan" tetap keputusan yang buruk

Replikasi hasilnya benar, tetapi akibatnya tidak dapat diterima: **tidak ada tombol hapus**
juga. Satu huruf yang salah ketik saat menambah baris akan menetap selamanya, dan baris itu
muncul di dropdown setiap layar yang memakainya.

Menyalin jalur Pega bukan pilihan — bila kedua page klipboard itu kebetulan terisi sisa nilai
dari layar lain dalam sesi yang sama, pernyataan itu akan **menimpa catatan progres sebuah
klaim** dengan isian layar master. Yang ditambahkan karena itu bukan jalur lamanya, melainkan
`UPDATE` yang benar ke tabel master.

#### Dua batas yang dijaga

| Batas | Alasan |
|---|---|
| `ID_MST` **tidak pernah** ikut di-`SET` | ia dirujuk `GCNM_PROGRESS_CLAIM.STATUS_PROGRESS2` pada data klaim yang sudah berjalan; mengubahnya memutus rujukan tanpa galat apa pun |
| `STS_PROGRESS1` **ditulis ulang server** | ia SALINAN nama induk, bukan rujukan. Bila induk berpindah, salinan lama akan menyesatkan. Nama induk dibaca dari barisnya di dalam transaksi yang sama, persis seperti pada penambahan |

Permintaan membawa **ID di jalur, bukan di badan** (`PUT /api/master/status-progres-2/{id}`).
Menerimanya dari badan berarti satu permintaan dapat menyebut dua ID yang berbeda, dan salah
satunya pasti diabaikan diam-diam.

#### Ini SELISIH pada gerbang 1, dan dinyatakan di muka

Perilaku baru pada modul yang sedang diuji kesetaraannya wajib dinyatakan lebih dulu, bukan
ditemukan sebagai kejutan saat pengujian (`D-54`). Yang akan terlihat berbeda:

- sistem lama: menekan "Update" **tidak mengubah** baris master
- sistem baru: baris master **berubah**, dan nama induknya ikut disalin ulang bila induk berpindah

Dicatat pula di doc comment `masterstatusprogres.Repo2.Update`, pada form, dan pada layarnya —
sehingga siapa pun yang membacanya dari sisi mana pun menemukan alasannya tanpa perlu mencari.

#### Yang berubah di kode

| Berkas | Perubahan |
|---|---|
| `masterstatusprogres2.go` | `Repo2.Update` ditambahkan ke antarmuka |
| `repo/sqlstore/masterstatusprogres2.sql` | kueri `progress_status2_update` — `SET` tiga kolom, `WHERE TRIM(ID_MST)` |
| `repo/sqlstore/masterstatusprogres2.go` | `Update` dalam satu transaksi: baca induk → `UPDATE` → `RowsAffected()==0` berarti baris hilang |
| `repo/memory/memory2.go` | padanannya; induk dibaca **sebelum** mutex diambil agar tidak mengunci diri sendiri |
| `usecase/manage2.go` | `Service2.Update` — validasi, pilih repo per portal, teruskan |
| `http/routes2.go` | `PUT /master/status-progres-2/{id}` |
| `http/errors.go` | `ErrParentNotFound` → 422 dengan pelanggaran pada isian `id_induk` |
| `api2.ts` | `useUpdateProgressStatus2` |
| `ProgressStatus2Form.tsx` | satu form, **dua mode**; ID tampil sebagai keterangan saat menyunting |
| `ProgressStatus2Page.tsx` | kolom `Aksi`, `FormState` bertiga keadaan, `save` bercabang |

#### Satu uji yang harapannya harus dibalik

`TestNoEditRoute2Exists` dulu memastikan `PUT` dijawab **404**. Setelah rutenya ada, chi
menjawab **405 Method Not Allowed** untuk metode lain pada jalur yang sama — dan 405 justru
lebih benar: jalurnya memang ada, metodenyalah yang tidak dilayani. Ujinya diganti menjadi
`TestOnlyPutIsRegistered2`, yang sekarang menjaga hal yang lebih berguna: **`DELETE` tetap tidak
dilayani.**

---

### 16.14 Judul kolom pertama: "ID" → "No" — kesalahan kolom keempat

Work Owner: *"kolom No seperti pega tidak ada"*.

Benar. Header grid Pega berbunyi **`No`**, terikat ke `.CaseID` (`ID_MST`):

```
<pyValue>&lt;b&gt;No&lt;b&gt;</pyValue>   ->  .CaseID
```

Layar ini menuliskannya **"ID"**. `D-13` menetapkan teks yang dilihat pengguna mengikuti layar
lama, sehingga judulnya diganti menjadi `No` — di grid maupun di keterangan pada form, supaya
satu layar tidak menyebut hal yang sama dengan dua nama.

**Isinya tidak berubah.** Ia tetap `ID_MST`, bukan nomor urut baris. Menomori ulang per halaman
akan menyesatkan: yang dirujuk `GCNM_PROGRESS_CLAIM.STATUS_PROGRESS2` adalah `ID_MST`, dan itulah
nomor yang dicari petugas.

**Layar tingkat 1 juga memakai "ID"**, dan **tidak disentuh** — ia di bawah Isolasi Protektif.
Apakah header Pega-nya pun berbunyi `No` belum diperiksa; dicatat di sini supaya dapat ditinjau
bersama Work Owner.

Ini kesalahan kolom **keempat** pada modul yang sama, dan yang pertama menyangkut **teks judul**,
bukan kolom mana yang digambar. Aturan §16.12 karena itu diperluas: header kolom disalin dari
`<pyValue>` section **apa adanya**, bukan diterjemahkan sendiri menjadi istilah yang terasa lebih
tepat.

---

## 19. Koreksi Work Owner — grid Master Auto Claim tidak mengikuti Pega (2026-09-20)

### 19.1 Yang dilaporkan

> "Pada Modul Master auto claim kenapa kolom rekening dan bank di satukan sedangkan pada pega
> tidak, dan tolong ui nya mengikuti pega ini" — disertai tangkapan layar grid Pega.

Laporannya benar, dan lebih luas dari satu kolom.

### 19.2 Kesalahan saya, dan kenapa ia lolos

Saya **menulis urutan kolom Pega dengan benar di komentar kode**, lalu tidak mengikutinya. Tiga
kolom digabung dan satu ditambahkan sendiri:

| Yang saya buat | Pega |
|---|---|
| `Bank & rekening` — dua nilai satu sel | `BANK PENERIMA` · `NO REKENING` — **dua kolom** |
| `PIC lapor` — PIC + email satu sel | `EMAIL LAPOR` · `PIC` — **dua kolom**, dan urutannya email dulu |
| `Nama penerima` — nama + client satu sel | `NAMA PENERIMA` saja; client **tidak ada di grid** |
| `Dipakai` — kolom karangan | tidak ada |
| `Alamat penerima` | **hilang** dari layar saya |

Penyebabnya bukan salah baca: komentar di atas kode itu memuat kesembilan header beserta alias
dan kolomnya, lengkap. Yang terjadi adalah saya "merapikan" tampilan sambil menulis JSX, dan
tidak membandingkan hasilnya kembali dengan komentar yang baru saja saya tulis sendiri.

**`D-13` tidak menyisakan ruang untuk itu:** alur dan tata letak ditiru supaya pengguna tidak
perlu belajar ulang. Penggabungan kolom adalah perubahan tata letak, sekecil apa pun niatnya.

### 19.3 Dua temuan lanjutan saat memperbaikinya

Membaca ulang keempat section untuk membetulkan kolom memunculkan dua hal yang **juga** salah
di versi pertama, dan keduanya lebih dari kosmetik.

**Pertama: keempat tab tidak berkolom sama.**

```
BrowseAutoKlaim          9 kolom  (… PIC, KOMITE)
BrowseAutoKlaimKomite    8 kolom  (… PIC)          <- TANPA KOMITE
BrowseAutoKlaimApproval  9 kolom
BrowseAutoKlaimReject    9 kolom
```

Masuk akal: di tab Komite seluruh barisnya memang milik pemanggil, sehingga kolom KOMITE tidak
memberi tahu apa pun.

**Kedua: Approve dan Reject TIDAK berada di baris grid.** Saya menaruhnya di sana. Di Pega
keduanya ada di **form**. Dibuktikan dari letaknya di dalam berkas section:

| Elemen | Offset |
|---|---|
| isian form (nama penerima) | 59.116 |
| tombol `stsapprove` (Approve · Reject) | 152.789 – 165.855 |
| header grid (`PIC`) | 232.228 |
| tombol `INISIAL` (Update, per baris) | 279.901 |

Keduanya jauh **sebelum** grid dimulai. Alur Pega karena itu: tekan **Update** pada baris →
isian termuat ke form → komite memutuskan atas isi yang benar-benar dilihatnya.

Dan justru letak itu yang menjelaskan cacat yang sudah saya catat di §18.6: komite yang
**tidak** memuat barisnya lebih dulu mengirim page form kosong, lalu `UPDATE` menulis
`CLIENTID`/`CLIENTNAME` kosong apa adanya.

**Ketiga: tombol form berbeda per tab.** Dari sensus parameter tiap section:

| Tab | Tombol baris | Tombol form |
|---|---|---|
| Master Auto Klaim | Update | Simpan (`stsapprove="0"`) |
| Komite Approval | Update | **Approve · Reject** — tanpa Simpan |
| Waiting Approval | Update | **tidak ada sama sekali** — baca saja |
| Reject | Update | Simpan |

Tab Waiting Approval hanya memanggil `UpdateMstAutoClaim_act1` (memuat baris), tanpa satu pun
activity penyimpan. Ia memang layar baca — yang memutuskan adalah komite yang ditunjuk.

### 19.4 Yang dikerjakan

| Perubahan | Berkas |
|---|---|
| Sembilan kolom terpisah + lebar piksel Pega, kolom aksi tanpa judul | `AutoClaimPage.tsx` |
| Kolom KOMITE dihilangkan di tab Komite | idem |
| Kolom `Dipakai` dicabut | idem |
| Approve/Reject pindah dari baris grid ke form | `AutoClaimPage.tsx` · `AutoClaimForm.tsx` |
| Tombol form mengikuti tab; Waiting Approval menjadi baca saja | `AutoClaimForm.tsx` |
| Catatan `CLAIM_ALLOWED` pindah ke **luar** grid | `AutoClaimPage.tsx` |

Lebar kolomnya diambil apa adanya dari section: `72 · 177 · 167 · 105 · 139 · 163 · 60 · 90 ·
115 · 93` piksel.

### 19.5 Keterangan yang tidak ada di Pega — dipindah, bukan dibuang

Kolom `Dipakai` saya buat karena keadaan "disetujui tetapi `CLAIM_ALLOWED` bukan 1" **tidak
terlihat di layar mana pun**, di Pega maupun di sini, padahal ia membuat sebuah baris tidak
pernah dipakai pembuatan klaim otomatis.

Alasannya masih berlaku; tempatnya yang salah. Sekarang ia **catatan di atas tabel** yang hanya
muncul bila barisnya memang ada, dan menyebut inisialnya. Grid tetap mengikuti Pega kolom per
kolom.

Kalau Work Owner lebih suka keterangan itu hilang sama sekali, menghapusnya satu blok — dan
mode `-periksa` tetap melaporkan angkanya.

### 19.6 Dua hal yang TETAP berbeda dari tangkapan layar, dan itu disengaja

| Hal | Alasan |
|---|---|
| **Tidak ada ikon saring (▼) per kolom** | `DataTable` bersama menyediakan satu kotak pencarian dan pengurutan per kolom. Menambah penyaring per kolom mengubah komponen yang dipakai SELURUH layar master (`U-2`), dan itu di luar lingkup modul ini |
| **Header kolom aksi kosong** | mengikuti Pega. Sel judul kosong tidak memberi nama pada kolomnya bagi pembaca layar; yang menutupinya adalah tombol di dalam tiap sel, yang bernama "Update" |

### 19.7 Hasil pemeriksaan

| Pemeriksaan | Hasil |
|---|---|
| `npm run typecheck` | **hijau** |
| `vitest run src/modules/master-auto-claim` | **hijau**, 18 uji (naik dari 14) |
| Backend | tidak tersentuh — perubahan ini seluruhnya di lapisan tampilan |

Empat uji baru menjaga koreksi ini tidak kembali: urutan kesembilan kolom diuji **sebagai
senarai** (kolom yang benar tetapi berpindah tempat pun gagal), isi **selnya** diuji satu per
satu, kolom KOMITE diuji hilang di tab Komite, dan tombol Approve diuji **tidak ada** di dalam
`<table>`.

Uji sel itu ditambahkan belakangan, dan alasannya penting: **uji header saja tidak cukup.**
Sebuah render yang headernya benar tetapi tetap menggabungkan dua nilai ke dalam satu sel akan
lolos — dan itu persis bentuk kesalahan yang dikoreksi Work Owner. Yang menangkapnya hanya
pembacaan sel per sel.

## 25. Koreksi Master Bengkel — kolom, urutan, dan paginasi diselaraskan dengan Pega (2026-09-20)

Work Owner mengirim **tangkapan layar Pega yang sedang berjalan** dan meminta kolom serta
paginasinya disesuaikan. Tangkapan layar itu mengungkap lima selisih yang tidak terbaca
dari export saja — dan kelimanya kemudian **dikonfirmasi ulang ke export**, bukan disalin
dari gambar.

| # | Sebelum | Sesudah | Bukti di export |
|---|---|---|---|
| 1 | Judul "Master Bengkel" | **"Master Bengkel HE"** | caption `Section/MasterBengkelHE-Section.xml` |
| 2 | Tab: Approve · Waiting Approval · Reject | **Approve · Reject · Waiting Approval** | layar Pega; urutan dokumen XML tidak menentukan urutan visual |
| 3 | 6 kolom rakitan (Nama+ID, Kota&cabang, Kontak, Rekanan, Diskon, Aksi) | **6 kolom Pega + Ubah** | `pyColumnCount = 7`; urutan properti di dalam definisi grid |
| 4 | Urut `NAMA_BENGKEL` menaik | **`ID_BENGKEL` menurun** | `pySortType=DESC`, `pySortOrder=1` pada kolom pertama |
| 5 | Tanpa paginasi | **20 baris per halaman** | `pyPageSize="Other"` + `pyPageSizeOther=20` |

### Kolom grid, terverifikasi dari definisi grid — bukan dari gambar

Tangkapan layar dapat terpotong di tepi kanan, sehingga kolomnya dihitung ulang dari
export sebelum diubah. `Section/BrowseMasterHEApprove-Section.xml` menyetel
`pyColumnCount = 7`, dan keenam kolom datanya muncul pada urutan ini di dalam definisi
gridnya:

```
ID_BENGKEL       ID Bengkel
NAMA_BENGKEL     Nama Bengkel
ALM_BENGKEL      Alamat Bengkel
TELP_BENGKEL     Telp Bengkel
NOHP_BENGKEL     No HP Bengkel
LOGIN_APLIKASI   Login Aplikasi
(aksi)           Ubah
```

Keempat kolom yang sempat saya tampilkan — `NAMA_KABUPATEN`, `STATUS_REKANAN`,
`NAMA_CABANG`, `MAIL` — **tidak muncul sama sekali** di area grid; seluruhnya hanya ada di
form. Kolom rakitan yang "lebih informatif" itu karena itu dicabut: ia membuat layar yang
berbeda dari yang dipakai petugas hari ini (`D-13`).

### Satu keterangan yang tetap dipertahankan, dengan tempat yang berbeda

Kolom "Rekanan" dicabut, tetapi penandaan **bengkel rekanan yang login aplikasinya
kosong** tetap ada — sekarang di dalam sel Login Aplikasi itu sendiri.

Alasannya: di Pega, sel kosong pada kolom itu tidak dapat dibedakan antara "memang tidak
diberi login" (non-rekanan) dan "seharusnya punya tetapi tidak" (rekanan). Yang kedua
berarti bengkelnya tidak akan pernah dapat masuk, dan itu tidak terlihat di layar mana
pun. Keterangan "rekanan tanpa login" hanya muncul pada keadaan kedua — tanpa menambah
satu kolom pun.

### Paginasi: komponennya sudah ada

`DataTable` ternyata **sudah** punya paginasi opt-in lewat prop `pageSize`, ditambahkan
sesi `masterpasal` yang berjalan bersamaan — dan doc comment-nya bahkan sudah mencatat
"Master Supplier · Master Bengkel → 20 baris". Yang dikerjakan di sini hanyalah
menyalakannya. Tidak ada satu baris pun komponen bersama yang disentuh.

### Urutan bawaan dipindahkan ke server

`ORDER BY NAMA_BENGKEL` menjadi **`ORDER BY ID_BENGKEL DESC`** pada `bengkel_list` dan
`bengkel_list_search`, meniru `pySortType=DESC` pada kolom pertama grid.

Itu bukan sekadar kosmetik pada layar berpaginasi: **urutan menentukan baris mana yang ada
di halaman berapa**, dan urutan yang salah tidak terlihat sebagai galat — halaman pertama
tetap terisi dan angkanya tetap masuk akal. `TestListQueriesOrderByKeyDescending` yang
menjaganya, dan repo memori ikut disesuaikan supaya urutan saat pengembangan sama dengan
di produksi.

Satu keterbatasan dicatat di berkas `.sql`: pengurutannya leksikografis, dan itu sama
dengan urutan penerbitan **hanya selama lebar kunci tetap**. Kunci hari ini berlebar tetap;
bila nomor urut kelak melampaui sepuluh digit, kunci tumbuh dan urutannya tidak lagi sama —
tanpa menimbulkan galat apa pun.

### Yang TIDAK ditambahkan, meski ada di tangkapan layar

Dua tombol di bawah judul — **"Upload Document"** dan **"Upload Data Master Bengkel"**.
Keduanya nyata di layar Pega, tetapi rule di baliknya tidak ikut di export (`R-16`).
Menggambar tombol yang tidak melakukan apa pun lebih buruk daripada tidak
menggambarkannya: petugas akan menekannya dan mengira ada yang rusak.

### Hasil pemeriksaan sesudah penyesuaian

| Pemeriksaan | Hasil |
|---|---|
| `go build ./...` · `go vet ./...` | **hijau** |
| `go test ./internal/masterbengkel/...` | **hijau** — 55 uji |
| `gofmt -l ./internal/masterbengkel` | bersih |
| `npm run typecheck` | **hijau** |
| `vitest run src/modules/master-bengkel` | **hijau**, 20 uji (bertambah 4) |
| `vitest run` (seluruhnya) | 222 lulus, 3 gagal — **pre-existing** di `master-rekening` |

Uji yang bertambah: urutan tab, susunan keenam kolom, dua uji paginasi (pemotongan 20
baris dan perpindahan halaman), serta penegasan bahwa bengkel **non**-rekanan tanpa login
TIDAK ikut ditandai.

Satu kegagalan transien tercatat: `go build` sempat menolak dengan *"pattern all:dist:
contains no embeddable files"* karena sesi lain sedang menulis `backend/spa/dist` pada saat
yang sama. Tidak ada yang diperbaiki untuk itu — percobaan berikutnya hijau.

---

## 26. Tiga tombol unggah Master Panel dihapus (2026-09-20)

Putaran umpan balik ketiga pada Master Panel, dan yang paling singkat.

### Pertanyaan yang memicunya

*"kenapa tombol upload belum tersedia?"*

Pertanyaannya wajar, dan jawaban saya sebelumnya — "rule-nya tidak ikut di export" —
adalah **klaim tanpa bukti**: saya menuliskannya di komentar kode dan di dokumen tanpa
pernah menelusuri apa sebenarnya yang dipanggil ketiga tombol itu.

### Yang ditelusuri, dan hasilnya

| Langkah | Hasil |
|---|---|
| Cari `<pyLocalAction>` pada `Section/BrowsePanelHE-Section.xml` | ketiganya memanggil local action: `UploadDocument`, `PNCUploadMasterPanelCSV`, `PNCUploadLokasiPanelCSV` |
| Cari rule ber-`pyRuleName` persis ketiga nama itu di 2.634 berkas | **nol** |
| Cari nama-nama itu di berkas lain | dua di antaranya **hanya muncul di section yang memanggilnya**, tidak di tempat lain mana pun |
| Periksa apakah Flow Action memang diekspor | `Flow Action/` berisi **29 berkas**, termasuk `PNCUploadDataKlaimSlikOJK-FA.xml` yang justru unggah CSV serupa |

Langkah terakhir yang menentukan: karena Flow Action **memang ikut diekspor**, ketiadaan
ketiganya adalah **gap nyata** (`R-16`) — bukan kategori yang tidak pernah dikirim.

### Keputusan

Work Owner memilih **menghapus ketiganya**, membalik pilihan sebelumnya (ditampilkan mati).
Rinciannya beserta pertimbangannya di `keputusan-implementasi.md` §28.

### Yang dikerjakan

| Berkas | Perubahan |
|---|---|
| `PanelPage.tsx` | blok tiga tombol dihapus; diganti komentar yang menyebut ketiga local action dan alasan ketiadaannya |
| `PanelPage.test.tsx` | uji dibalik: `tidak menggambar satu pun tombol unggah` |
| `README.md` | baris "Tombol" dan paragraf ketiga tombol ditulis ulang |

### Pelajaran yang dicatat

**Klaim "tidak ada di export" harus disertai cara memeriksanya, bukan disebut begitu saja.**
Saya menulisnya di tiga tempat — komentar kode, `README`, dan dokumen — dan baru
menelusurinya setelah ditanya. Penelusurannya sendiri hanya butuh empat perintah, dan
hasilnya jauh lebih berguna daripada klaimnya: ia menyebutkan **nama artefak yang harus
diminta** ke Tim Pega, bukan sekadar menyatakan ada yang kurang.

### Hasil pemeriksaan

| Perintah | Hasil |
|---|---|
| `npm run typecheck` | **hijau** |
| `vitest run src/modules/master-panel` | **hijau**, **18 uji** — jumlahnya tetap, satu uji diganti bukan dibuang |

Backend **tidak tersentuh** sama sekali: ketiga tombol itu tidak pernah punya endpoint.

---

## 27. Koreksi Master Supplier — kolom diselaraskan dengan Pega (2026-09-20)

Work Owner mengirim **tangkapan layar Pega yang sedang berjalan** dengan satu pertanyaan:
*"kenapa kolomnya tidak sesuai pega?"*

Pertanyaannya benar, dan sebabnya **kekeliruan saya sendiri**: grid sembilan kolom saya
gabungkan menjadi tujuh kolom majemuk. Alasan yang saya pakai disalin dari Master Bengkel
— *"grid empat puluh kolom tidak dapat dibaca pada layar mana pun"* — dan alasan itu tidak
berlaku di sini. Grid supplier hanya **sembilan** kolom, dan sembilan kolom memang dapat
dibaca.

**Tidak ada satu pun uji yang menangkapnya.** Yang menangkapnya adalah Work Owner yang
melihat layarnya. Uji penjaganya baru ditambahkan sesudah ini — lihat di bawah.

### Selisih yang diperbaiki

| # | Sebelum | Sesudah | Bukti |
|---|---|---|---|
| 1 | 7 kolom rakitan (Nama+ID · Kota&cabang · Kontak · Jenis&supply · TOP/TOD · Status · Aksi) | **9 kolom Pega, urutan sama** | `pyColumnCount = 9`; urutan sel di definisi grid |
| 2 | Kolom `POSISI` dihilangkan | **digambar, meski selalu kosong** | ia ada di grid Pega, dan di layar aslinya pun kosong |
| 3 | Caption tombol "Ubah" | **"Edit"** | tangkapan layar; `D-13` |
| 4 | `JENIS SUPPLIER` & `STATUS REKANAN` menampilkan sandi | **label**, dengan sandi sebagai cadangan | Pega mengikat `.JENIS_STATUS_NOTE` dan `.STS_REKANAN_NOTE` |
| 5 | `STATUS AKTIF` berupa lencana bertitik | **teks "Aktif" apa adanya** | tangkapan layar menampilkan teks polos |

### Kolom grid, terverifikasi ulang ke export — bukan disalin dari gambar

Tangkapan layar dapat terpotong di tepi kanan, sehingga kolomnya dihitung ulang dari
export lebih dulu. `Section/InboxMasterSupplier-Section.xml` menyetel `pyColumnCount = 9`,
dan kesembilan selnya muncul pada urutan ini:

```
.ID                  ID
.NAMA                NAMA            → layar produksi: "Input Nama"
.ALAMAT              ALAMAT
.TELEPON             TELP
.JENIS_STATUS_NOTE   JENIS SUPPLIER  pxDropdown — menampilkan LABEL
.STS_REKANAN_NOTE    STATUS REKANAN  pxDropdown — menampilkan LABEL
.STS_AKTIF           STATUS AKTIF
.POSISI              POSISI
(tombol)             OPTION
```

Kolom yang sempat saya tampilkan — kota, cabang, contact person, TOP, TOD, dan bank —
**tidak satu pun ada di grid Pega**; seluruhnya hanya ada di form.

### Grid ini TIDAK punya urutan bawaan

Berbeda dari Master Bengkel yang menyetel `pySortType = DESC` pada kolom pertamanya,
grid supplier tidak punya `pySortType`, `pySortOrder`, maupun `pyInitialSortColumn`, dan
`pyDisplayInitialSort = false`. Urutannya sepenuhnya mengikuti kueri daftar — yang justru
**tidak ada di export** (`R-16`).

`ORDER BY` menurut nama pada `supplier_list` karena itu **dipertahankan**: tidak ada
urutan Pega yang dapat ditiru, dan urutan yang ditentukan lebih baik daripada urutan yang
berubah-ubah antar pemanggilan. Ini berbeda dari kasus Bengkel, tempat urutan aslinya
memang terbaca dan karena itu wajib diikuti.

### Tiga temuan dari tangkapan layar yang TIDAK terbaca dari export

**1. Caption "Input Nama" tidak ada di export.** Export menuliskannya `NAMA`; pencarian
teks "Input Nama" di seluruh section dan harness Master Supplier tidak menemukannya. Layar
produksi karena itu **sudah berubah sejak export diambil** — bukti langsung untuk `R-09`
(sistem sumber masih aktif berubah). Yang diikuti adalah layar yang dilihat pengguna
(`D-13`).

**2. Label `JENIS SUPPLIER` berbunyi "ASM".** Kolom itu terikat `.JENIS_STATUS_NOTE`, yaitu
label dari `JENIS_STATUS`. Sandinya sendiri tidak terlihat di layar.

Ini **melemahkan** label yang saya berikan pada `DefaultCodeOption`: saya menamai
`JENIS_STATUS = "1"` sebagai *"Heavy Equipment"*, disimpulkan dari `SUPPLIER_HE := "1"`
yang memang terbukti. Tetapi bila labelnya di produksi berbunyi "ASM", domain
`JENIS_STATUS` kemungkinan **lebih kaya daripada dua nilai**, dan form yang memaksanya
menjadi `0`/`1` lewat `GetDataSupplier_pre` justru membuang informasi.

Belum diubah, karena hubungan `JENIS_STATUS = "1"` → `SUPPLIER_HE = "1"` tetap terbukti.
Diangkat sebagai pertanyaan terbuka.

**3. ID pada layar berbunyi `1005972` — tujuh digit.** Procedure menerbitkannya sebagai
`id_site || lpad(to_char(supplier_seq.nextval), 11, '0')`, yang **tidak mungkin**
menghasilkan tujuh karakter.

Dan ia benar-benar kolom `ID`, bukan `OLDID`: tombol Edit mengirimkan nilai kolom itu ke
`GetDataSupplier_pre`, yang menyaring `A.ID = {ParamSP.ID}`. Artinya **baris supplier yang
ada sekarang tidak mengikuti format procedure itu** — ia lahir lewat jalur lain.

Akibatnya: ID yang diterbitkan aplikasi baru akan **terlihat berbeda** dari setiap baris
yang sudah ada. Itu bukan cacat — asal barisnya terbaca dari kuncinya, persis alasan
`D-22` memilih prefiks `PNCN` — tetapi ia harus dinyatakan, bukan ditemukan.

### Label kolom bersandi: dicari, bukan ditebak

`JENIS_STATUS_NOTE` dan `STS_REKANAN_NOTE` hanya muncul di harness dan section layar ini,
dan **tidak ada satu pun kueri di export yang memuatnya** — keduanya berasal dari
`ListMasterSupllier` yang memang hilang (`R-16`).

Yang dikerjakan: label dicari dari daftar sandi yang sudah dikirim server lewat `/sandi`
(`labelOf`). Bila artinya diketahui, labelnya yang tampil; bila tidak, **sandinya sendiri**
yang tampil. Begitu daftar Field Value-nya diterima, kedua kolom ikut menampilkan label
tanpa satu baris kode pun berubah.

Hari ini berarti: `STATUS AKTIF` tampil sebagai "Aktif"/"Tidak aktif" seperti Pega, dan
kedua kolom lain masih menampilkan sandinya — terlihat, bukan tersamar.

### Satu keterangan yang tetap dipertahankan, di dalam kolom yang sama

Keterangan **"menunggu persetujuan"** tetap ada, sekarang di dalam sel `STATUS AKTIF`.

Alasannya tidak berubah: supplier yang baru ditambahkan sebagai aktif tampil sebagai tidak
aktif, karena `CreateNewMasterSupplier_post` step 6 menetapkan `STS_AKTIF := "0"` tanpa
syarat. Tanpa keterangan itu, petugas tidak punya cara mengetahui sebabnya. Ia **tidak**
menambah kolom — susunannya tetap sembilan.

### Uji yang ditambahkan supaya ini tidak terulang

| Uji | Yang dijaganya |
|---|---|
| `menggambar kesembilan kolom Pega pada urutan yang sama` | membandingkan seluruh `columnheader` dengan daftar harfiah — penggabungan kolom gagal di sini lebih dulu |
| `menampilkan label sandi, dan sandinya sendiri bila artinya tidak diketahui` | kolom bersandi tidak kembali menampilkan sandi mentah |
| `tetap menggambar kolom POSISI meski selalu kosong` | kolom kosong tidak "dirapikan" dengan menghapusnya |

Ketiganya menguji hal yang sebelumnya **tidak diuji sama sekali** — dan itulah sebabnya
kekeliruannya lolos sampai ke layar.

### Hasil uji

| Suite | Hasil |
|---|---|
| `npx tsc --noEmit` | bersih |
| `npx vitest run src/modules/master-supplier` | **27 lulus** |
| `npx vitest run` (seluruh frontend) | 225 lulus, **3 gagal** |

Ketiga kegagalannya tetap `master-rekening/AccountPage.test.tsx` — pre-existing, jumlahnya
tidak berubah sejak §17.11.

### Yang tersisa sebagai pertanyaan

1. **Apakah `JENIS_STATUS` benar-benar biner?** Label "ASM" di layar produksi menunjukkan
   kemungkinan sebaliknya. Bila ya, `GetDataSupplier_pre` yang memaksanya menjadi `0`/`1`
   membuang nilai — dan form kami ikut membuangnya.
2. **Nama supplier di layar tampak sebagai tautan** (biru). Export tidak memasang aksi apa
   pun pada sel itu. Dibiarkan sebagai teks; bila di produksi ia membuka sesuatu, perlu
   diketahui membuka apa.
3. **Label `STATUS REKANAN`** — layar menampilkan "Non Rekanan". Master Bengkel punya bukti
   `STATUS_REKANAN = "0"` berarti bukan rekanan, tetapi itu kolom tabel **lain**. Masih
   belum saya pakai sebagai dasar.

---

## 28. Paginasi Master Panel — ukuran halaman berbeda ANTARTAB (2026-09-20)

### Permintaan

*"dan tolong buatkan pagination sesuai pega"*

### Yang dibaca dari export sebelum menyalakannya

Ukuran halaman tidak ditebak. Ia dibaca dari `pyPageSize` pada section masing-masing —
atau `pyPageSizeOther` bila nilainya `"Other"` — beserta `pyPageMode` yang menentukan
bentuk paginatornya:

| Grid | Section | `pyPageSize` | `pyPageMode` | Hasil |
|---|---|---|---|---|
| **Grid utama** | `BrowsePanelHEApprove` | `Other` → **15** | `Numeric` | 15 baris, nomor halaman |
| **Grid utama** | `BrowsePanelHEReject` | **50** | `Numeric` | 50 baris, nomor halaman |
| **Grid utama** | `BrowsePanelHEApproval` | **50** | `Numeric` | 50 baris, nomor halaman |
| Sub-grid Lokasi | ketiganya | 20 | **`None`** | **tidak dipaginasi** |

Cara memastikan mana `pyPageSize` milik grid mana: keduanya berdampingan di dalam satu
section, dan yang membedakannya adalah `pyPageListProperty` di sekitarnya —
`TempStsClaim.LOKASI` untuk sub-grid, `pgRepPgSubSection…pxResults` untuk grid utama.

**Angka 15 dikuatkan bukti kedua yang berdiri sendiri:** tangkapan layar Pega yang dikirim
Work Owner memperlihatkan tepat lima belas baris pada tab Approve — dari `1000106` turun
sampai `1000092` — beserta paginator `1 2 3 4 5 6 7 8 >`.

### Keputusan: perbedaan antartab DITIRU

Satu layar, satu report definition, tiga tab — dan ukuran halamannya berbeda. Tidak ada
alasan bisnis apa pun yang membuat daftar yang **disetujui** layak dipotong lebih pendek
daripada daftar yang **ditolak**. Ia hampir pasti akibat ketiga section dikonfigurasi
sendiri-sendiri, bukan keputusan siapa pun.

Ia tetap ditiru, dan alasannya adalah pelajaran §24: **penyimpangan tata letak menuntut
alasan yang lebih kuat daripada "lebih rapi"**. Yang layak menjadi alasan hanyalah hal yang
tidak dapat dikerjakan — dan menyamakan ketiganya bukan salah satunya.

Penyeragamannya **diangkat sebagai pertanyaan ke Work Owner**, bukan diputuskan sendiri.

### Sub-grid lokasi sengaja TIDAK dipaginasi

`pyPageSize=20` tersimpan pada grid lokasi, tetapi `pyPageMode="None"` mematikan
paginatornya — seluruh baris digambar sekaligus. Masuk akal untuk daftar yang kelima
pilihan lokasinya saja; yang menahan pertumbuhannya adalah `masterpanel.MaxLocationRows`
(50), bukan paginator.

### Yang berubah

| Berkas | Perubahan |
|---|---|
| `PanelPage.tsx` | `pageSize` ditambahkan ke setiap butir `TABS` (15 · 50 · 50), diteruskan ke `DataTable` lewat `active.pageSize` |
| `LocationEditor.tsx` | catatan bahwa daftar lokasi memang tidak dipaginasi, beserta buktinya — supaya tidak ditambahkan kemudian |
| `components/DataTable.tsx` | daftar ukuran halaman per layar dilengkapi Master Panel; ditambah catatan bahwa ia layar pertama yang berbeda **antartab**, sehingga angkanya milik tab dan bukan milik layar |
| `PanelPage.test.tsx` | empat uji baru: potongan 15 baris tab Approve, 50 baris tab Reject, perpindahan halaman, dan paginator yang tidak digambar bila hanya satu halaman |
| `README.md` | baris "Paginasi" pada tabel tata letak beserta catatan perbedaan antartab |

Tidak ada perubahan backend. Paginasi ini **sisi klien**, memotong daftar yang sudah
dimuat — sama dengan seluruh layar master lain, dan sama dengan Pega yang memaginasi page
list klipboard. Paginasi keyset sisi server tetap `TKT-U2-001`.

### Kendala yang ditemui

| Kendala | Penyelesaian |
|---|---|
| Uji tab Reject gagal dengan `Unable to find role="table"` | Tab awal (Approve) sengaja kosong pada uji itu, dan `DataTable` menggambar pesan kosong alih-alih tabel. `await findByRole('table')` di awal dihapus — bilah tab tidak bergantung pada data, sehingga dapat langsung diklik |

### Hasil pemeriksaan

| Perintah | Hasil |
|---|---|
| `npm run typecheck` | **hijau** |
| `vitest run src/modules/master-panel` | **hijau**, **22 uji** — naik dari 18 |
| `vitest run` (seluruhnya) | 229 lulus, 3 gagal — ketiganya pre-existing di `master-rekening`, modul yang tidak tersentuh |

### Yang perlu diputuskan Work Owner

Apakah ketiga tab diseragamkan menjadi satu ukuran halaman. Sekarang mengikuti Pega apa
adanya: 15 di Approve, 50 di dua tab lain. Menyamakannya adalah satu baris perubahan, dan
akibatnya hanya pada tampilan — tidak menyentuh data maupun kontrak API.

---

## 29. Koreksi Master Auto Claim — caption dan urutan tab (2026-09-20)

### 29.1 Yang dilaporkan

> "Ubah filter Master Auto Klaim ini menjadi seperti di PGE" — disertai tangkapan layar tab
> pertama layar ini.

Yang ditunjuk adalah tab bercaption **"Master Auto Klaim"**. Di Pega, tab dengan nama itu
**tidak ada**.

### 29.2 Apa yang sebenarnya salah

`Section/MasterAutoKlaim-Section.xml` adalah layout group ber-`pyHeaderType=TABBED`,
`pyTabAlignment=Top`, `pyStretchTab=false`. Caption tiap tab ada di `pyTitle` containernya,
dan urutan dokumen inilah urutan yang dilihat pengguna:

| offset | `pyTitle` | section di bawahnya | penyaring |
|---|---|---|---|
| 84.789 | **Approve** | `BrowseAutoKlaim` | `stsapprove="1"` |
| 122.544 | **Reject** | `BrowseAutoKlaimReject` | `stsapprove="2"` |
| 164.383 | **Waiting Approval** | `BrowseAutoKlaimApproval` | `stsapprove="0"` |
| 202.564 | **Komite Approval** | `BrowseAutoKlaimKomite` | `stsapprove="0"`, `komite="ya"` |

**"Master Auto Klaim" adalah JUDUL LAYAR**, sebaris dengan tombol Tambah dan Refresh di atas
keempat tab. Saya menjadikannya caption tab pertama, sehingga:

- tab **Approve hilang sama sekali**, dan
- ketiga tab lainnya ikut bergeser urutannya.

| | Punya saya | Pega |
|---|---|---|
| 1 | Master Auto Klaim | **Approve** |
| 2 | Komite Approval | **Reject** |
| 3 | Waiting Approval | Waiting Approval |
| 4 | Reject | **Komite Approval** |

Hanya satu dari empat yang kebetulan benar tempatnya.

### 29.3 Kenapa ini lolos dua kali

Pada §18 saya membaca `pyDeferLoadRetrievalActivityParams` tiap section — itulah yang memberi
**penyaringnya**, dan keempatnya benar sejak awal. Yang tidak saya baca adalah `pyTitle`
containernya, yang memberi **captionnya**. Saya menyusun caption dari nama section dan dari judul
layar, bukan dari sumbernya.

Pada §19, saat membetulkan kolom, saya membaca ulang keempat section — tetapi mencari header
grid, bukan judul tab. Sumber yang sama dibuka dua kali tanpa pertanyaan itu pernah diajukan.

### 29.4 Judul layar: "Klaim" dengan K

Judul `<h1>` diubah dari "Master Auto Claim" menjadi **"Master Auto Klaim"**, mengikuti teks
pada section (`D-80`: teks yang dilihat pengguna mengikuti layar Pega apa adanya).

Butir **menu** di kolom samping tetap "Master Auto **Claim**" dengan C, karena ia datang dari
`POOLDATA.M_MENU_APLIKASI_PNC.MENU_DESC`. Kedua ejaan itu memang berbeda di sistem lama;
keduanya direplikasi dari sumbernya masing-masing, bukan diseragamkan sepihak.

### 29.5 Satu nama yang kini muncul dua kali

"Approve" sekarang menjadi caption **tab pertama** sekaligus caption **tombol** pada form tab
Komite. Begitu pula "Reject". Keduanya memang bernama sama di Pega.

Bagi pengguna itu tidak membingungkan — letaknya berbeda. Bagi uji, ia jebakan: pencarian global
`getByRole('button', { name: 'Approve' })` akan menemukan keduanya. **Kekeliruan yang sama sudah
pernah menjatuhkan tiga uji Master Rekening** (§17).

Penutupnya: seluruh penekanan tab kini lewat `clickTab`, yang membatasi pencarian pada
`<nav>` bilah tab. Tidak ada satu pun pencarian tab yang global lagi.

### 29.6 Yang TIDAK diubah, dan kenapa

**Bentuk visual tabnya** — garis bawah biru, bukan tab berbingkai gaya Pega klasik.

Alasannya bukan kemalasan melainkan tabrakan dua instruksi: "UI mengikuti Pega" versus "konsisten
dengan modul sebelumnya". Seluruh layar master lain — Master Rekening, Penolakan Klaim, Supplier,
Panel, Bengkel — memakai gaya garis bawah yang sama. Mengubah satu modul saja membuat aplikasinya
tidak seragam, dan mengubah semuanya menyentuh modul yang berada di bawah Isolasi Protektif.

Diangkat ke Work Owner sebagai pertanyaan, bukan diputuskan sendiri.

### 29.7 Hasil pemeriksaan

| Pemeriksaan | Hasil |
|---|---|
| `npm run typecheck` | **hijau** |
| `vitest run src/modules/master-auto-claim` | **hijau**, 19 uji (naik dari 18) |
| Backend | tidak tersentuh |

Uji baru `menampilkan keempat tab Pega pada urutan yang sama` membandingkan caption **sebagai
senarai**, dan menegaskan "Master Auto Klaim" ada sebagai `heading` tetapi **tidak** ada sebagai
tab.

---

## 30. Modul Master Sparepart (2026-09-20)

Layar kesepuluh, pengganti `Harness/SparePart_HE-Harness.xml` atas `POOLDATA.SPAREPART_HE`
(MENU_ID 31).

### 30.1 Harness-nya hampir kosong — aturannya ada di sembilan berkas lain

Pembacaan pertama nyaris menyesatkan. `Harness/SparePart_HE-Harness.xml` berukuran 283 KB,
tetapi isinya cangkang portal Pega: yang benar-benar miliknya hanya dua rujukan section dan
dua tombol, **Tambah** dan **Refresh**. Tidak satu pun nama kolom, aturan, atau label isian
ada di dalamnya.

Yang harus dibaca, dan urutan menemukannya:

| Berkas | Yang diberikannya |
|---|---|
| `Section/BrowseMasterSparepartHE-Section.xml` | urutan ketiga tab, dan dua tombol unggah |
| `Section/BrowseMasterSparepartHEApproval-Section.xml` | 20 label isian, 4 penanda wajib, bentuk tiap kontrol |
| `Report Definition/BrowseSparepartHE_RD-RD.xml` | 23 nama kolom `SPAREPART_HE` |
| `RDB List/ValidationMasterSparepart{No,Name,Code}-SQL.xml` | tiga kunci alami |
| `RDB List/GetIDDokumenSparepart-SQL.xml` | kolom ke-24, `DOKUMENID` |
| `Database/PEGA_M_SPAREPART_HE.prc` | bentuk ID, lebar 10 digit, nama sequence |
| `Activity/UpdateSparepartHE_act-Act.xml` | urutan simpan, `APPROVAL := "0"`, stempel pelaku dan waktu |
| `Activity/BrowseTipeKategoriPart-Act.xml` | isi kedua dropdown acuan, beserta penyaringnya |
| `Activity/SetApprovalAllMaster-Act.xml` | keputusan borongan, dan jenis master `"M_SPAREPART_HE"` |

Pelajarannya: **ukuran berkas harness tidak menunjukkan berapa banyak yang ada di dalamnya.**
Pada Pega, harness yang besar biasanya besar karena boilerplate portalnya, bukan karena
isinya.

### 30.2 Tiga jawaban "seperti aplikasi PEGA" yang menuntut penelusuran lanjutan

Work Owner menjawab tiga dari empat pertanyaan dengan "seperti aplikasi PEGA". Jawaban itu
tampak menutup pertanyaan, padahal justru **memindahkannya**: dari "mana yang Anda pilih"
menjadi "apa yang sebenarnya dilakukan Pega".

Yang paling menentukan adalah pertanyaan tentang dropdown Kategori dan Tipe. Export memuat
tiga rule yang membacanya dengan penyaring berbeda, dan pilihan yang salah akan menentukan
**kategori mana yang boleh ditautkan ke sebuah sparepart**:

| Rule | Penyaing | Dipanggil dari |
|---|---|---|
| `BrowseSparepartCategoryClaimHE` | menunggu | layar Master Kategori |
| `BrowseSparepartTipeClaimHE2` | menunggu | layar Master Tipe |
| `BrowseMasterSparepartCategoryClaimHE` + `BrowseTipeSparepart` | **sudah disetujui** | **layar Master Sparepart** |

Yang membuktikannya adalah pencarian pemanggil, bukan pembacaan rule: hanya
`Activity/BrowseTipeKategoriPart-Act.xml` yang dirujuk **ketiga** section tab Master
Sparepart, dan activity itulah yang menjalankan dua rule terakhir.

Kalau penelusuran ini dilewati, layar akan menawarkan kategori yang belum disetujui — dan
kesalahannya tidak akan terlihat sampai ada kategori yang ditolak.

### 30.3 Dua isian yang bentuknya dapat ditiru, isinya tidak

`SATUAN`, `JENIS_SPART`, `STS_AKTIF`, dan `STS_PART` dirender dropdown atau radio di Pega,
tetapi daftar pilihannya ada di rule Field Value yang **tidak ikut di export** (`R-16`).

Satu-satunya jejak nilainya di seluruh export adalah `RDB List/GetDataSparepart_SQL-SQL.xml`,
yang menyebut `PCS` dan `SET` — tetapi atas tabel **`tender_supplier`**, bukan
`SPAREPART_HE`. Memakainya sebagai domain berarti memindahkan nilai dari tabel lain ke tabel
ini tanpa satu pun bukti keduanya sama.

Yang dikerjakan: bentuknya ditiru, isinya diambil dari nilai yang sudah dipakai baris lain,
dan selalu ada jalan keluar berupa **"Lainnya…"**. Komponennya `ChoiceField.tsx`.

Satu cacat yang tertangkap saat menguji: versi pertamanya memakai `<span>` sebagai label,
sehingga `getByLabelText('Satuan')` gagal. Itu bukan sekadar uji yang rewel — label yang
tidak tertaut juga tidak dibacakan pembaca layar. Diperbaiki menjadi `<label htmlFor>` untuk
varian dropdown; varian radio tetap memakai `<span>` di dalam grup ber-`aria-label`, karena
sekumpulan radio memang tidak punya satu elemen untuk ditunjuk.

### 30.4 Dua kolom tanggal yang tipenya belum diketahui

`PROD_DATE` dan `TGL_UPDATE_HARGA` tidak punya DDL (`R-08`), dan export tidak memberi satu
pun petunjuk — tidak ada `TO_CHAR`, `TRUNC`, maupun `TO_DATE` yang dikenakan pada keduanya
di seluruh export.

Yang dipakai, beserta dasarnya:

- `PROD_DATE` → **teks**. Pega merendernya `pxTextInput` TANPA `pyDateTimeFormat`; ia bukan
  kontrol tanggal, dan `SetDataSparepart` menyalinnya apa adanya.
- `TGL_UPDATE_HARGA` → **waktu**. Pega mengisinya dengan waktu sistem, dan properti DateTime
  Pega dipetakan ke kolom DATE.

Bila salah satunya keliru, penyimpanan gagal dengan galat konversi pada percobaan pertama —
gagal keras dan terlihat, bukan diam-diam menulis nilai yang salah. Pemeriksaannya sudah
terpasang: kueri `sparepart_check_table` menyebut **kedua** kolom itu, sehingga
`claimpnc -periksa` akan menolak lebih dulu bila salah satunya tidak dapat dibaca.

### 30.5 Perubahan yang dilakukan

| Lapisan | Berkas |
|---|---|
| Domain | `internal/mastersparepart/mastersparepart.go`, `lookup.go` |
| Aplikasi | `internal/mastersparepart/usecase/manage.go` |
| Adapter | `repo/sqlstore/mastersparepart.{go,sql}`, `repo/memory/{memory,sample}.go` |
| Transport | `internal/mastersparepart/http/{dto,errors,routes}.go` |
| Perakitan | `cmd/claimpnc/main.go`, `cmd/claimpnc/check.go` |
| Frontend | `modules/master-sparepart/{api.ts,SparepartPage.tsx,SparepartForm.tsx,ChoiceField.tsx}` |
| Kontrak | `api/types.ts` (tambahan di akhir berkas) |
| Rute & menu | `app/App.tsx`, `app/menu/registry.ts` |

Uji: 4 berkas, seluruhnya hijau — domain, usecase, kueri SQL, transport HTTP, dan layar.

### 30.6 Yang TIDAK disentuh

Modul Login, Home, dan sembilan layar master yang sudah selesai tidak disentuh sama sekali.
Tiga berkas bersama — `components/DataTable.tsx`, `components/SelectField.tsx`, dan
`api/client.ts` — sudah membawa perubahan orang lain saat sesi ini dimulai, dan tidak satu
pun ikut disunting di sini.

Tiga uji pada `master-rekening/AccountPage.test.tsx` gagal di akhir sesi. Ketiganya **bukan
akibat pekerjaan ini**: berkas `master-rekening` tidak berbeda satu byte pun dari HEAD, dan
tidak satu pun mengimpor berkas yang disentuh sesi ini. Kegagalannya berasal dari perubahan
`DataTable.tsx` dan `SelectField.tsx` yang sudah ada di working tree sebelum sesi dimulai.

### 30.7 Satu kesalahan sendiri yang tercatat

Untuk membuktikan bahwa ketiga kegagalan `master-rekening` itu bukan dari sesi ini, saya
menjalankan `git stash push` dengan pathspec. Perintahnya melaporkan galat pathspec, **tetapi
stash-nya tetap terbentuk** — dan ia menelan seluruh berkas frontend modul ini, termasuk
direktori yang belum terlacak. Dikembalikan dengan `git stash pop`, dan seluruh berkasnya
utuh.

Pelajarannya: `git stash` bukan alat pemeriksaan. Pertanyaan yang sedang dijawab — "apakah
berkas ini berubah?" — sudah terjawab oleh `git diff --stat`, yang tidak menyentuh working
tree sama sekali. Alat yang mengubah keadaan tidak boleh dipakai untuk membaca keadaan.

---

## 31. Sesi paralel — Modul Pelaporan Klaim (2026-09-18)

> Sesi ini berjalan di cabang `feat/Michelle-flowpelaporan-backup`, **bersamaan** dengan
> §13–§15 yang berjalan di `master`. Ia ditulis sebagai "sesi keenam" di cabangnya sendiri;
> nomor §31 diberikan saat digabungkan, supaya rujukan §13–§15 yang sudah ada tidak bergeser.
> Urutan nomor karena itu **bukan** urutan waktu.

Modul **proses klaim** yang pertama. Sampai sesi ini yang ada hanyalah login, portal, beranda,
dan dua modul master; tidak satu pun menyentuh perjalanan sebuah klaim.

### 31.1 Permintaan

Work Owner meminta penambahan **modul Pelaporan Klaim**, dengan `Flow/InputReceiveDocument.xml`
sebagai rujukan aplikasi existing, dan menuntut analisis penuh sebelum satu baris kode ditulis.

### 31.2 Yang diperiksa lebih dulu, sebelum menulis kode

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

### 31.3 Nama modulnya bukan karangan — tiga bukti

Instruksi menyebut "Pelaporan Klaim" sementara case Pega-nya bernama `Work-ReceiveDocument`.
Ketiga bukti berikut menunjukkan nama Work Owner justru yang dipakai sistem lama di permukaan:

| Bukti | Isi |
|---|---|
| `Navigation/pyCaseWorkerNavigation-Navigation.xml:19864` | menu **"Inbox Laporan Klaim"** menuju `InboxRCVApp_Harness` |
| `Activity/CreateNewCaseRCV-Act.xml` step 7 | `Param.Posisi = "LAPORAN KLAIM"` dan `Param.note = "Auto Create Laporan"` |
| `Section/ViewStatusReceiveDocument-Section.xml` | komentar developer: *"done add row num in inbox pelaporan klaim"* |

`D-81` menetapkan nama modul diambil dari nama yang disebut Work Owner. Di sini keduanya cocok.

### 31.4 Temuan yang menghentikan pekerjaan sebelum dimulai

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

### 31.5 Koreksi atas temuan saya sendiri

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

### 31.6 Empat pertanyaan konfirmasi dan jawabannya

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

### 31.7 Yang dibangun

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

### 31.8 Perubahan di luar modul, dan alasannya

Seluruhnya **penambahan**; tidak ada yang me-refactor modul yang sudah selesai.

| Berkas | Perubahan | Kenapa tidak dapat dihindari |
|---|---|---|
| `cmd/claimpnc/main.go` | perakitan modul, pemasangan rute, jembatan pemanggil | Modul memasang rutenya sendiri, tetapi perakitannya milik entrypoint |
| `cmd/claimpnc/periksa.go` | laporan kesiapan `CPNC_LAPORAN_KLAIM` per tahap | Membedakan "migrasi belum jalan" dari "tidak punya hak baca" |
| `internal/platform/waktu/wib.go` | **berkas baru** — zona WIB dan tanggal kalender | `F-5` menuntut konversi di SATU tempat. Menaruhnya di dalam modul akan membuat modul berikutnya menyalinnya |
| `frontend/src/api/tipe.ts` | tipe `LaporanKlaim`, `TahapLaporan`, `KodeGalatLaporan` | Berkas ini memang cerminan DTO Go |
| `frontend/src/app/App.tsx` | satu rute | Titik pasang modul, setara `main.go` |
| `frontend/src/app/KerangkaHalaman.tsx` | satu entri menu | Tanpanya layar hanya dapat dicapai dengan mengetik URL |
| `frontend/src/components/TabelData.tsx` | **satu prop opsional** `cariDiServer` | Lihat §31.9 |

### 31.9 Kendala: tabel baku menyaring di peramban, layar ini tidak boleh

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

### 31.10 Kendala lain dan penyelesaiannya

| Kendala | Penyelesaian | Dampak |
|---|---|---|
| Nilai uang dibawa sebagai teks desimal (tidak ada pustaka desimal), sedangkan kolom `NUMBER` menyerahkan konversinya ke `NLS_NUMERIC_CHARACTERS` — pada sesi berlokal koma, teks `1234.56` DITOLAK | Kolomnya dibuat `VARCHAR2(30)` dengan `CHECK` regex, dan alasannya ditulis lengkap di migrasi | Penyimpangan sadar dari `09-DATABASE-STRATEGY` §5. Kolomnya tidak dapat dijumlahkan di SQL — dapat diterima karena nilai ini tidak dipakai perhitungan apa pun |
| `time.LoadLocation("Asia/Jakarta")` gagal di Windows tanpa basis data zona waktu | `time.FixedZone("WIB", 7*3600)` | Deterministik di mesin mana pun; WIB memang tidak mengenal daylight saving |
| Bentuk nomor laporan lama tidak dapat ditiru — `pyWorkIDPrefix` ada di rule kelas yang tidak diekspor, dan nol contoh nilainya di seluruh export | Bentuk baru `LPK.YY.xxxx` mengikuti `D-71`, diisolasi di satu berkas | Menunggu konfirmasi Work Owner; yang berubah hanya satu konstanta |
| `z.coerce.number()` membuat tipe masukan dan keluaran skema berbeda, sehingga React Hook Form dan Zod bertengkar soal `defaultValues` | Jumlah dokumen disimpan sebagai teks di skema, diubah menjadi angka satu kali saat mengirim | Satu baris konversi, tipe tetap sehat |
| `exactOptionalPropertyTypes` menolak prop opsional menerima nilai yang mungkin `undefined` | Ditulis eksplisit dengan `\| undefined`, mengikuti pola `KolomIsian` yang sudah ada | — |
| Heredoc bash gagal pada dokumen panjang berisi tanda kutip | Ditulis lewat berkas scratchpad lalu digabungkan — kendala yang sama sudah tercatat di §8.5 | Hanya cara penulisan |

### 31.11 Verifikasi — YANG TIDAK DAPAT DIJALANKAN

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

### 31.12 Cacat repository yang ditemukan, di luar lingkup modul ini

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

### 31.13 Yang belum dapat dibuktikan

| Acceptance criteria | Keadaan | Apa yang menahannya |
|---|---|---|
| Kode dapat dikompilasi | **Belum** | Go tidak terpasang di mesin ini |
| Uji lulus | **Belum** | idem, dan Node untuk sisi frontend |
| Layar bekerja terhadap Oracle | **Belum** | migrasi `0003` belum dijalankan DBA (`D-63`) |
| Nomor laporan terbit unik dari urutan | **Belum diuji** terhadap Oracle | idem, dan menuntut hak `INSERT` yang belum tentu dimiliki akun aplikasi |
| Hasil setara dengan Pega (gerbang 1) | **Belum** | `S-8` belum ada, dan Pega staging yang dapat ditembak dari luar belum dikonfirmasi |

### 31.14 Arahan Work Owner: yang di luar lingkup dibiarkan apa adanya

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
`keputusan-implementasi.md` dan `penggunaan-skill.md` (§31.12), ketidakkonsistenan rute dan menu
Master Rekening (§31.12), dan `CLAUDE.md` yang memuat perubahan `D-80`/`D-81` belum ter-commit
milik Work Owner.

**Berkas bersama yang tetap berubah, dan sifat perubahannya.** Seluruhnya tuntutan modul baru,
bukan perbaikan cacat yang tidak berhubungan:

| Berkas | Baris terhapus | Sifat |
|---|---|---|
| `api/tipe.ts` · `app/App.tsx` · `app/KerangkaHalaman.tsx` · `cmd/claimpnc/periksa.go` · `README.md` | **nol** | murni penambahan |
| `cmd/claimpnc/main.go` | 6 | seluruhnya **perataan gofmt** yang dipaksa nama field baru yang lebih panjang; nol logika tersentuh |
| `components/TabelData.tsx` | 6 | satu prop opsional `cariDiServer`. Layar yang tidak mengisinya berperilaku sama persis — lihat §31.9 untuk alasannya dan untuk akibat bila ia dibatalkan |

Arahan ini dicatat sebagai preferensi kerja yang berlaku seterusnya, bukan hanya untuk sesi ini.

### 31.15 Keputusan Work Owner atas jejak pada `TabelData` (2026-09-18)

Setelah arahan §31.14, satu-satunya berkas bersama yang masih memuat baris berubah adalah
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
uji frontend yang sudah ada, dan Node tidak terpasang di mesin ini (§31.11).

**Satu kehilangan nyata yang ditemukan saat Work Owner memeriksa dan sudah dikembalikan.** Komentar
paket sempat kehilangan rujukan `TKT-U2-001` ketika saya menulis ulang paragrafnya. Rujukan itu
kini utuh di barisnya semula, dan penjelasan `cariDiServer` ditambahkan sebagai paragraf terpisah
di bawahnya.

---

### 31.16 Penggantian nama ke bahasa Inggris — koreksi Work Owner atas asumsi saya

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
ini, **itu belum terbukti**: Go dan Node tidak terpasang di mesin ini (§31.11), sehingga tidak ada
`go build`, `go vet`, `go test`, `tsc`, maupun `vitest` yang dapat dijalankan.

Yang dapat saya lakukan sebagai gantinya adalah penelusuran manual: seluruh nama lama dicari ulang
di backend maupun frontend dan **nol kemunculan tersisa** di luar modul `auth`, `portal`,
`masterrekening`, dan `masterstatus` yang memang berada di luar lingkup.

---

## 32. Sesi kesepuluh — modul View History Claim (2026-09-20)

Menu `MENU_ID 76` "View History Claim", pengganti harness `PNCSearchKlaim`. Modul proses
klaim kedua setelah Pelaporan Klaim.

### 32.1 Analisis pra-implementasi

Dikerjakan sebelum satu baris kode ditulis, atas permintaan Work Owner. Yang dibaca:

| Berkas | Yang diambil |
|---|---|
| `Harness/PNCSearchKlaim-Harness.xml` | judul layar "VIEW HISTORY CLAIM" |
| `Section/PNCSearchKlaim-Section.xml` | 3 isian, tombol Cari, 2 grid × 13 kolom, kondisi tampil |
| `Activity/PNCSearchHistoryKlaim_Act-Act.xml` | pemilih kueri per tipe, penyiapan parameter |
| `Activity/InsertLogProteksiDataKlaimMasking-Act.xml` | gerbang proteksi data + kuota |
| `Activity/CekmaskingDataPerLoginUserKlaim-Act.xml` | pembacaan master proteksi |
| `Database/UPDATE_LOG_PROTEKSI.prc` | isi log proteksi |
| 12 berkas `RDB List/` | SQL sesungguhnya tiap tipe pencarian |

**Tiga temuan yang mengubah rancangan**, dan tak satu pun terbaca dari dokumen mana pun:

1. **Dua belas dari enam belas kolom grid bernama menyesatkan.** `.EDMNO` berarti No
   Klaim, `.THEINSURED` berarti Posisi Klaim, `.SOBNAME` berarti PIC Teknis. Pemetaan
   lengkapnya kini di `peta-penamaan.md` dan di kepala `riwayatklaim.sql`.
2. **Layar tergerbang proteksi data.** Pengguna wajib terdaftar di
   `POOLDATA.MST_PROTEKSI_DATA_PNC`, dan satu jatah pencarian terpakai setiap kali layar
   dibuka. Ini tidak tercatat di dokumen migrasi mana pun.
3. **Tiga cacat aturan.** Pencarian Tanggal Lahir membaca isian yang tersembunyi,
   prakondisi langkah 6 adalah tautologi, dan pencarian Nama Objek kehilangan kolom
   Posisi Klaim.

### 32.2 Pertanyaan konfirmasi dan jawabannya

Enam pertanyaan diajukan sebelum kode ditulis; seluruhnya dijawab Work Owner 2026-09-20.

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Gerbang proteksi dibawa sejauh apa | **Bangun penuh** |
| 2 | Ketiga cacat aturan | **Replikasi apa adanya** |
| 3 | Tipe pencarian No Rekening | ditanyakan balik: "script ini dipanggil di mana" → lalu **replikasi apa adanya** |
| 4 | Label tipe pencarian & kode 10 | **pakai turunan dari deskripsi langkah**, kode 10 dilewati |
| 5 | Ke mana penulisan gerbang diarahkan (`P-1`) | **tabel baru milik aplikasi** |
| 6 | Branch dasar | **`Feat/arlexy-View-History-Claim`** |

**Pertanyaan 3 menghasilkan temuan tambahan.** Penelusuran menemukan
`AmbilDataKlaimDenganNoRekening` hanya dipanggil satu tempat di seluruh export — layar ini
sendiri. Tetapi penelusuran yang sama memperlihatkan pencarian itu **tidak pernah memakai
nomor rekening yang diketik**: rekeningnya tertanam tetap di dalam SQL, dan isian pengguna
disisipkan sebagai potongan SQL mentah di posisi yang membuat kuerinya cacat sintaksis.

**Satu koreksi atas deskripsi saya sendiri.** Pertanyaan 1 diajukan dengan keterangan yang
keliru: saya menyebut "kuota lihat data berkurang" dan "tiap pencarian dicatat ke
`LOG_DATA_PROTEKSI_KLAIM`". Verifikasi berikutnya membuktikan yang berkurang adalah kuota
**pencarian** (`LOGSEARCH`), gerbangnya berjalan **sekali saat layar dibuka** bukan tiap
pencarian, log **tidak** ditulis di layar ini, dan masking tidak berlaku karena grid-nya
tidak punya kolom KTP/telepon/surel. Koreksinya disampaikan sebelum kode ditulis; arah
keputusannya tidak berubah.

### 32.3 Yang dibangun

**Backend — `internal/riwayatklaim/`**

| Berkas | Isi |
|---|---|
| `riwayatklaim.go` | `ClaimHistory` 16 isian, `Pagination`, `Page`, `Caller`, seam `Repo`/`Clock` |
| `searchtype.go` | 12 tipe pencarian, `Criteria`, `QueryValue` — tempat cacat direplikasi |
| `proteksi.go` | `Protection`, `Access`, `Usage`, `Check`, `Grant`, seam `ProtectionRepo` |
| `errors.go` | tiga galat domain + `ValidationError` |
| `usecase/search.go` | `Open` (memakai jatah) dan `Search` (tidak) |
| `repo/sqlstore/` | 11 kueri pencarian + 4 kueri gerbang, pembungkus paginasi |
| `repo/memory/` | penyimpanan memori + 5 klaim contoh + 3 baris proteksi contoh |
| `http/` | dto, galat, handler, rute |

**Migrasi** `0004_riwayat_klaim_proteksi.{up,down}.sql` — tabel
`POOLDATA.CPNC_PEMAKAIAN_PROTEKSI`, satu sequence, dua indeks. **Belum pernah dijalankan.**

**Frontend — `src/modules/riwayat-klaim/`** — `types.ts`, `api.ts`, `SearchPanel.tsx`,
`ClaimHistoryPage.tsx`, beserta ujinya.

**Rute API baru:**

| Metode | Jalur | Keterangan |
|---|---|---|
| `POST` | `/api/riwayat-klaim/buka` | menjalankan gerbang, **memakai satu jatah** |
| `GET` | `/api/riwayat-klaim` | pencarian; tidak memakai jatah |

### 32.4 Kendala yang muncul dan penyelesaiannya

| Kendala | Penyelesaian |
|---|---|
| **Working tree berpindah branch di tengah sesi.** Sesi dimulai di `feat/arlexy-inbox-pelaporan-klaim` dengan modul `inboxlaporanklaim` yang saya pakai sebagai acuan pola; di tengah analisis tree berpindah ke `master`, tempat modul itu tidak ada dan digantikan `pelaporanklaim` | Diperiksa lebih dulu apakah pekerjaan hilang — tidak: ia aman di commit `924baf9`. Pola diacu ulang dari `pelaporanklaim` yang ada di `master`, dan branch dasar ditanyakan ke Work Owner |
| **`DataTable` di `master` tidak punya `serverPaging` maupun `hideSearch`** | Paginasi digambar sendiri di layar, mengikuti pola `ClaimReportPage`. `hideSearch` ditambahkan sebagai prop opsional — aditif, bawaan `false`, sehingga tidak satu pun layar lama berubah |
| **`SelectOption` tidak mengenal `disabled`** | Tidak diubah. Tipe yang belum tersedia ditandai pada labelnya, dan yang dinonaktifkan adalah tombol Cari |
| **Membuka layar memakai jatah, sementara `StrictMode` menjalankan efek dua kali** | Pembukaan memakai `useQuery` ber-`staleTime: Infinity`, bukan `useMutation` di dalam `useEffect`. Diuji: tepat satu permintaan `POST` |
| **Uji menekan dropdown sebelum isinya tiba** | Helper `renderOpened()` menunggu salah satu pilihan muncul, bukan menunggu labelnya — label sudah ada sejak penggambaran pertama |

### 32.5 Verifikasi yang benar-benar dijalankan

```
cd backend  && go build ./... && go vet ./... && go test ./...
cd frontend && npm run typecheck && npm test
```

| Pemeriksaan | Hasil |
|---|---|
| `go build ./...` | bersih |
| `go vet ./...` | bersih |
| `go test ./...` | seluruh paket lulus, termasuk 3 paket baru |
| `npm run typecheck` | bersih |
| `npm test` | **112 lulus, 3 gagal** |

**Ketiga kegagalan itu sudah ada di `master` sebelum sesi ini**, seluruhnya di
`master-rekening/AccountPage.test.tsx`. Dibuktikan dengan menjalankan berkas uji itu
terhadap `DataTable` versi `master` — hasilnya sama persis, 3 gagal. Bukan akibat
perubahan sesi ini, dan **tidak diperbaiki** karena berada di luar lingkup tugas.

Uji modul ini: **13 uji layar + 20 uji backend**, seluruhnya lulus.

**Yang TIDAK dapat diverifikasi:** seluruh SQL modul ini belum pernah dijalankan terhadap
Oracle. Tidak ada basis data di mesin tempat berkas ini ditulis, dan migrasi 0004 belum
dijalankan DBA. Yang terbukti hanyalah bentuk kuerinya — lewat `query_test.go` yang
memeriksa keseragaman alias, jumlah parameter, disiplin SQL portabel, dan larangan menulis
ke tabel milik sistem lama.

## 18. Sesi kesebelas — modul Inbox Admin (2026-09-20)

Menu `MENU_ID 63` "Inbox Admin", pengganti harness `PNCInboxAdmin`. Modul proses klaim
ketiga setelah Pelaporan Klaim dan View History Claim.

### 18.1 Analisis pra-implementasi

Dikerjakan sebelum satu baris kode ditulis, atas permintaan Work Owner. Yang dibaca:

| Berkas | Yang diambil |
|---|---|
| `Harness/PNCInboxAdmin-Harness.xml` (1,9 MB) | judul layar; tiga section yang di-*include* |
| `Section/PNCInboxAdmin-Section.xml` (1,9 MB) | 11 tab, 8 kontainer grid, 9 tombol, kondisi tampil |
| `Section/PNCAdminShow_sec-Section.xml` | sel Case ID; di-*include* 6× |
| `Section/ButtonPagingInbox-Section.xml` | First/Previous/Next/Last + "Total Data :" |
| `Activity/SetTempClaimRegistandNotRegist-Act.xml` (396 KB, 38 langkah) | pemilih kueri per tab; penyusun seluruh filter |
| `Activity/{First,Previous,Next,Last}PageGrid_Act` | mekanisme paginasi |
| 9 berkas `RDB List/` | SQL sesungguhnya tiap tab |

**Tujuh temuan yang mengubah rancangan**, dan tak satu pun terbaca dari dokumen mana pun:

1. **Tab dipilih lewat properti bernama `TempView.CityID`** — bukan kode kota, melainkan
   kode tab. Nilainya `3`, `4`–`6`, dan `7`–`13`.
2. **Alias berubah ARTI dari tab ke tab.** `.RCVID` berarti sumber bisnis di tab ALL dan
   nama surveyor di tab Request Survey; `.Keterangan`, `.Kurir`, `.UserAdmin`, dan
   `.StatusKomunikasi` sama. Ini lebih pekat daripada View History Claim, tempat satu alias
   salah arti tetapi setidaknya konsisten.
3. **`BrowseClaimALLKomunikasi` cacat sintaksis.** UNION dengan 15 kolom di cabang pertama
   dan 14 di cabang kedua — Oracle menolaknya dengan ORA-01789.
4. **Muat pertama menarik SELURUH baris.** Klausa paginasi baru terpasang setelah tombol
   halaman ditekan; totalnya dihitung dari baris yang sudah terlanjur ditarik.
5. **Tujuh titik ASIS** menyisipkan potongan SQL, termasuk klausa paginasinya sendiri.
6. **Penyaring cabang menembus DB Link** ke `HRDASM.V_HRD_MST@ASMD` (`R-03`).
7. **Kotak cari hanya menyentuh dua kolom**, dan mengetik teks bermuatan "PNC" memindahkan
   tab secara diam-diam.

### 18.2 Koreksi atas analisis saya sendiri

Saya menyusun pasangan label-tab ke kodenya dari urutan markup section, lalu **menariknya
kembali sebelum dipakai**: urutan tag di dalam satu blok XML ini acak, dan pasangan yang
sama menghasilkan dua peta berbeda tergantung arah pembacaan. Yang dipakai sebagai gantinya
adalah **semantik kueri di activity**, yang tidak bergantung pada urutan markup.

Akibat langsungnya dinyatakan apa adanya: saya **tidak dapat membuktikan sendiri** tab mana
yang ber-`pyCondition` bernilai `1==2`. Yang dapat dibuktikan hanyalah bahwa **empat
elemen** memang bertanda begitu. Pernyataan Work Owner yang menjadi acuan.

### 18.3 Pertanyaan konfirmasi dan jawabannya

Delapan pertanyaan diajukan dalam dua putaran sebelum kode ditulis; seluruhnya dijawab Work
Owner 2026-09-20.

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Sebelas tab sekaligus atau bertahap | **sekaligus** |
| 2 | Tab Komunikasi yang UNION-nya cacat | **tidak dipakai — sudah di-remark di Pega** |
| 3 | Penyaring cabang yang menembus DB Link | **tunggu API pengganti** |
| 4 | Tombol aksi mana yang dibangun | dijawab dengan hal lain, lalu ditanyakan ulang |
| 5 | Selama menunggu API, layar berbuat apa | **penyaring ditandai belum tersedia, data tidak disaring cabang** |
| 6 | Tombol aksi (ulang) | **Lihat Detail Klaim** |
| 7 | Nama modul | **`inboxadmin` / `inbox-admin`** |
| 8 | Paginasi | **replikasi apa adanya** |

**Jawaban 1 dan 2 digabung** menjadi delapan tab: sebelas dikurangi tiga Komunikasi.

**Jawaban atas pertanyaan 4 ternyata menjawab hal lain** — "Layar secara default
menampilkan data All Case Admin". Itu keterangan berharga yang tidak saya tanyakan, dan
menjadi `DefaultTab`; pertanyaan tombolnya diajukan ulang secara terpisah.

### 18.4 Yang dibangun

**Backend — `internal/inboxadmin/`**

| Berkas | Isi |
|---|---|
| `inboxadmin.go` | `WorkItem` 31 isian, `Pagination`, `Page`, `Slice`, `Caller`, seam `Repo`/`Clock` |
| `tab.go` | 8 tab beserta kolomnya, `DisabledTabs`, nama field |
| `query.go` | `BusinessLine`, `QueryInput`, `Query`, `NewQuery` |
| `errors.go` | `ErrCallerUnknown` dan `ValidationError` |
| `usecase/list.go` | `Metadata` dan `List` |
| `repo/sqlstore/` | 7 kueri tab + 1 pemeriksa tabel; pemindai 29 kolom |
| `repo/memory/` | penyimpanan memori + 14 baris contoh mencakup kedelapan tab |
| `http/` | dto, galat, handler, rute |

**Tanpa migrasi.** Seluruh tabel yang dibaca modul ini sudah ada dan milik sistem lama —
ini modul pertama yang tidak menuntut satu pun migrasi.

**Frontend — `src/modules/inbox-admin/`** — `types.ts`, `api.ts`, `InboxTabs.tsx`,
`InboxAdminPage.tsx`, beserta ujinya. Ditambah `src/app/ViewClaimPlaceholder.tsx`.

**Rute API baru:**

| Metode | Jalur | Keterangan |
|---|---|---|
| `GET` | `/api/inbox-admin/tab` | bentuk layar: tab, kolom, dropdown, keterbatasan |
| `GET` | `/api/inbox-admin` | isi satu tab |

### 18.5 Kendala yang muncul dan penyelesaiannya

| Kendala | Penyelesaian |
|---|---|
| **Urutan tag XML acak**, sehingga pasangan label-tab ke kode tidak dapat dibuktikan dari markup | Peta diambil dari semantik kueri di activity; keterbatasannya dinyatakan, bukan ditutupi |
| **Regex alias di uji ikut menangkap `CAST(NULL AS DATE)`** sebagai alias bernama "DATE" | Alias hanya dihitung bila berada di ujung baris kolom |
| **Uji "tanpa perangkaian SQL" ikut menolak pola LIKE yang sah** | Uji diperketat per baris: perangkaian hanya sah pada baris yang memuat `LIKE` |
| **Kotak cari menonaktifkan diri saat memuat**, sehingga ketikan hilang | `disabled` dicabut, dan ketikan diberi jeda 350 ms sebelum dikirim — lihat §18.6 |
| **`gofmt -l` menandai seluruh repo** | Berkasnya CRLF sementara gofmt menginginkan LF; ia bukan sinyal yang berguna di repo ini. Yang dipakai `go vet` dan `go test` |

### 18.6 Cacat yang ditemukan uji sendiri, dan itu bukan cacat uji

Uji "mengirim kata kunci ke server" gagal karena permintaannya tidak pernah terkirim.
Sebabnya bukan uji yang keliru: **kotak cari menonaktifkan dirinya selama permintaan
berjalan**, dan karena setiap ketikan memicu permintaan, sebagian besar huruf yang diketik
cepat akan tertelan.

Dua hal diperbaiki sekaligus: `disabled` dicabut dari kotak cari, dan ketikan diberi **jeda
350 ms**. Jeda itu bukan kosmetik — dengan paginasi yang direplikasi apa adanya, satu
permintaan menarik seluruh baris yang cocok, sehingga satu permintaan per huruf berarti
delapan kali penarikan penuh untuk satu kata.

### 18.7 Verifikasi yang benar-benar dijalankan

    cd backend  && go build ./... && go vet ./... && go test ./...
    cd frontend && npm run typecheck && npm test

| Pemeriksaan | Hasil |
|---|---|
| `go build ./...` | bersih |
| `go vet ./...` | bersih |
| `go test ./...` | seluruh paket lulus, termasuk 3 paket baru |
| `npm run typecheck` | bersih |
| `npm test` | **125 lulus, 3 gagal** |

**Ketiga kegagalan itu sudah ada sebelum sesi ini**, seluruhnya di
`master-rekening/AccountPage.test.tsx`. Angkanya cocok persis dengan baseline sesi
sebelumnya — 112 lulus ditambah 13 uji baru sama dengan 125 — dan `git status` membuktikan
berkas `master-rekening` tidak disentuh sama sekali. **Tidak diperbaiki** karena berada di
luar lingkup tugas.

Uji modul ini: **13 uji layar + 45 uji backend**, seluruhnya lulus.

**Yang TIDAK dapat diverifikasi:** seluruh SQL modul ini belum pernah dijalankan terhadap
Oracle. Tidak ada basis data di mesin tempat berkas ini ditulis. Yang terbukti hanyalah
bentuk kuerinya — lewat `query_test.go` yang memeriksa keseragaman 29 alias, kesesuaian
jumlah bind dengan argumen yang disiapkan, disiplin SQL portabel, larangan menulis, larangan
memaginasi di SQL, dan larangan menembus DB Link.

---

## 33. Sesi kedua belas — modul Inbox Progress Claim (2026-09-21)

Menu `MENU_ID 65` "Inbox Progress Claim", pengganti harness `ProgressClaim_Harness`.
Pemantauan progres klaim yang masih berjalan: sudah sampai posisi mana sebuah klaim, apa
status progresnya, dan kapan ia harus ditindaklanjuti berikutnya.

### 33.1 Yang dibaca sebelum satu baris kode ditulis

| Berkas | Yang diambil darinya |
|---|---|
| `Harness/ProgressClaim_Harness-Harness.xml` | pembungkus layar; menunjuk satu section |
| `Section/ProgressClaim_Section-Section.xml` | **lima** region bertumpuk, kolom grid, tombol |
| `Activity/GetDataProgressClaim-Act.xml` | region Outstanding — 14 langkah penyaring |
| `Activity/GetNextFUdata_act-Act.xml` | region Next Follow Up |
| `Activity/GetProgressPerPIC-Act.xml` | region Progress Klaim per PIC |
| `Activity/StatusProgress_act11-Act.xml` | aksi baris: membuka klaim |
| `RDB List/DataProgressClaim-SQL.xml` | kueri grid, berpaginasi `ROW_NUMBER` |
| `RDB List/GcnmCountProgressClaim_SQL-SQL.xml` | pencacah total baris |
| `RDB List/GetProgressPIC-SQL.xml` | rekap lima pencacah per PIC |
| `RDB List/GetIDCabang-SQL.xml` | penyaring cabang — **menembus DB Link `@ASMD`** |
| `Database/GET_POSISI_PROGRESS_PNC.fnc` | posisi dan status progres per klaim |

### 33.2 Pertanyaan konfirmasi dan jawabannya

Empat pertanyaan diajukan sebelum implementasi dimulai, karena keempatnya mengubah bentuk
pekerjaan secara material.

| # | Pertanyaan | Jawaban Work Owner |
|---|---|---|
| 1 | Dari lima region, mana yang masuk lingkup | **Outstanding, Next Follow Up, Progress Klaim per PIC, Evaluasi**. Approval dan Input Progress Claim **di luar lingkup** |
| 2 | Judul kolom tidak ada di export — bagaimana menetapkannya | **Pakai alias Pega apa adanya** |
| 3 | Dua kontrol terbukti tidak menyaring apa pun | **Direplikasi apa adanya — tampil tetapi mati** |
| 4 | Definisi lini bisnis grid dan Export bertentangan | **Ikuti versi Export — 4 lini** |

Jawaban 1 menutup satu-satunya jalur tulis layar ini, sehingga modulnya **tidak punya
operasi tulis sama sekali** — sama seperti Inbox Admin, dan aman terhadap `P-1`.

### 33.3 Tujuh temuan dari pembacaan export

1. **Alias kolom diacak, lebih parah dari Inbox Admin.** Nomor klaim dan nomor polis
   **tertukar**: `CaseID` berisi `noklaim`, `ClaimNo` berisi `nopolis`. Ditambah `District`
   yang berisi nama tertanggung dan `KomiteApproveDate` yang tidak berhubungan dengan
   komite sama sekali.
2. **`GET_POSISI_PROGRESS_PNC` dipanggil empat kali per baris**, masing-masing mengulang
   kursor yang sama. Pada satu halaman 15 baris itu 60 pemanggilan. Source-nya **ada** di
   `Database/`, sehingga `D-02` dapat dijalankan penuh.
3. **Layar ini benar-benar memaginasi di basis data**, berbeda dari Inbox Admin:
   `ROW_NUMBER` antara `FirstRow`–`LastRow`, ukuran halaman **15**, dengan kueri `COUNT`
   terpisah.
4. **Penyaring cabang menembus DB Link `@ASMD`** — penghalang `R-03` yang sama persis
   dengan Inbox Admin.
5. **Dua kontrol di layar tidak menyaring apa pun.** `TempRefresh.DateOfLoss` nol
   kemunculan di seluruh `RDB List/`; dropdown lini bisnis pada grid menulis ke
   `tempgetpic.CaseID` yang tidak dibaca kueri mana pun.
6. **Definisi lini bisnis grid dan Export bertentangan**, dan grid tidak punya TRAVEL.
7. **Judul kolom tidak dapat dibaca dari export** — seluruh sel ber-`pyHeaderTitle`
   kosong, dan folder `Property` tidak ikut dikirim.

Ditambah dua hal yang terbawa ke daftar selisih terencana: `addCalendar(...,7,0,0)` yang
`F-5` larang, dan seluruh penyaring yang dirangkai sebagai teks SQL.

### 33.4 Dua temuan yang muncul saat merancang, bukan saat membaca

**Region "Evaluasi Progress Klaim" ternyata cangkang kosong.** Ia punya judul dan kerangka
tabel satu baris, tetapi **nol properti terikat dan nol activity pengisi** —
`Refreshpage_act` hanya penyegar generik. Tidak ada yang dapat dimigrasikan. Ia tetap
digambar sebagai bagian dengan keterangan apa adanya, karena menghilangkannya akan membuat
orang mengira modulnya belum selesai.

**`DateForAging` digambar DUA KALI** sebagai dua kolom terpisah pada region Outstanding,
keduanya terikat `tglklaim`. Sementara `tgl_proses` yang dikembalikan kueri **tidak terikat
ke satu sel pun**. Dugaan: sel kedua seharusnya menggambar `tgl_proses` dan salah diikat.
Itu dugaan, bukan bukti — ia dicatat dan diajukan, bukan diam-diam diperbaiki.

### 33.5 Temuan yang mengubah rancangan region per PIC

Penyaring lini bisnis pada rekap per PIC ternyata **bukan dropdown**. Prakondisi keempat
cabangnya berbunyi `OperatorID.pyPosition == "NONMBU"` dan seterusnya — ia dibaca dari
**jabatan pada catatan operator Pega**, yang rupanya diisi nama lini bisnis.

Nilai itu **tidak tersedia di sistem baru**: HCC/HCQ mengembalikan jabatan sebenarnya
(`Placement.PositionName`), bukan lini bisnis. Sampai pemetaan pengguna ke lini bisnis
menjadi master data (`F-4`), lini bisnis **dipilih pengguna lewat dropdown** dan ditandai
wajib. Keterbatasan itu dinyatakan di layar.

Temuan kedua di region yang sama: langkah "Progress Claim per User" yang menyusun
`and a.pic = '<pengguna yang login>'` **tidak punya prakondisi sama sekali**, sehingga
selalu berjalan. Judulnya menyebut "per PIC", tetapi isinya selalu satu petugas — yang
sedang membuka layar.

### 33.6 Konflik merge yang sudah ada sebelum sesi ini

`cmd/claimpnc/main.go` dan `check.go` **ter-commit dengan penanda konflik merge yang belum
diselesaikan** (`<<<<<<< HEAD` … `>>>>>>> Feat-arlexy-Inbox-admin`). Akibatnya paket `cmd`
**tidak dapat dikompilasi** sejak sebelum sesi ini; `git status` membuktikan kedua berkas
tidak disentuh siapa pun di working tree.

Keenam konfliknya **murni aditif** — satu sisi menambah modul master, sisi lain menambah
`inboxadmin`. Penyelesaiannya mempertahankan **kedua sisi**, dan setelah itu backend
terbangun untuk pertama kalinya di branch ini.

### 33.7 Yang dibangun

**Backend — `internal/inboxprogressclaim/`**

| Berkas | Isi |
|---|---|
| `inboxprogressclaim.go` | `ClaimRow`, `Position`, `PICSummary`, paginasi, seam `Repo`/`Clock`/`RepoSelector` |
| `view.go` | keempat region, kolomnya, dan daftar kontrol mati |
| `query.go` | lini bisnis, validasi permintaan, penurunan `ClaimQuery`/`PICQuery` |
| `errors.go` | galat domain dan kumpulan pelanggaran |
| `usecase/list.go` | `Metadata()` dan `List()` yang bercabang menurut bentuk baris |
| `repo/sqlstore/` | enam kueri + pemuat + pemindai |
| `repo/memory/` | penyimpanan contoh yang meniru penyaring dan urutannya |
| `http/` | DTO, galat, handler, rute |

**Frontend — `src/modules/inbox-progress-claim/`** — `types.ts`, `api.ts`,
`ProgressSection.tsx`, `InboxProgressClaimPage.tsx`, beserta ujinya.

**Rute API baru:**

| Metode | Jalur | Keterangan |
|---|---|---|
| `GET` | `/api/inbox-progress-claim/bagian` | bentuk layar: region, kolom, dropdown, keterbatasan |
| `GET` | `/api/inbox-progress-claim` | isi satu region |

Tanpa satu pun migrasi basis data: seluruh tabel yang dibaca sudah ada dan milik sistem
lama.

### 33.8 Kendala yang muncul dan penyelesaiannya

| Kendala | Penyelesaian |
|---|---|
| **Kueri posisi butuh daftar nomor klaim satu halaman**, panjangnya berubah-ubah | Penanda bind disusun dari JUMLAH baris lewat `inList`; tidak ada satu pun nilai yang menyentuh teks SQL, dan ujinya membuktikan keluarannya hanya penanda |
| **`LISTAGG` dilarang repo, `STRING_AGG` tidak ada di Oracle 19c** | Penggabungan antarposisi dipindah ke Go seluruhnya; tidak ada agregasi teks di SQL sama sekali |
| **Uji "tanpa tulis" menolak `ID_UPDATE`** karena memuat kata `UPDATE` | Pencocokan diganti regex berbatas kata |
| **Dua bentuk baris dalam satu endpoint** | `ListResponse.Rows` bertipe `any` dengan alasan tertulis; alternatif dua endpoint ditolak karena layar belum tahu region mana yang diminta sebelum membaca metadata |
| **`exactOptionalPropertyTypes` menolak `render` bernilai `undefined`** | `renderFor` diubah selalu mengembalikan fungsi |
| **Judul kolom muncul dua kali di DOM** — `DataTable` menggambar tampilan meja dan kartu | Uji membaca `columnheader`, bukan teks |

### 33.9 Cacat yang ditemukan uji sendiri

Uji "tidak meminta apa pun ke server untuk bagian Evaluasi" **gagal**, dan itu bukan uji
yang keliru: bagian kosong tetap menembak server saat dibuka. Backend memang menjawabnya
tanpa menyentuh basis data, tetapi perjalanan jaringannya sia-sia — jawabannya sudah pasti
kosong. `enabled` hook diberi syarat tambahan `!empty`.

### 33.10 Verifikasi yang benar-benar dijalankan

    cd backend  && go build ./... && go vet ./... && go test ./...
    cd frontend && npm run typecheck && npm test

| Pemeriksaan | Hasil |
|---|---|
| `go build ./...` | bersih — **untuk pertama kalinya di branch ini** |
| `go vet ./...` | bersih |
| `go test ./...` | seluruh paket lulus, termasuk 4 paket baru |
| `npm run typecheck` | bersih |
| `npm test` | **332 lulus, 3 gagal** |

Ketiga kegagalan itu **sudah ada sebelum sesi ini**, seluruhnya di
`master-rekening/AccountPage.test.tsx`. Angkanya cocok persis dengan baseline yang diukur
di awal sesi — 311 lulus ditambah 21 uji baru sama dengan 332 — dan `git status`
membuktikan berkas `master-rekening` tidak disentuh sama sekali. **Tidak diperbaiki**
karena Isolasi Protektif melarang menyentuh modul Master Data yang sudah selesai.

Uji modul ini: **21 uji layar + 63 uji backend**, seluruhnya lulus.

**Yang TIDAK dapat diverifikasi:** seluruh SQL modul ini belum pernah dijalankan terhadap
Oracle. Tidak ada basis data di mesin tempat berkas ini ditulis. Yang terbukti hanyalah
bentuk kuerinya — lewat `query_test.go` yang memeriksa keseragaman alias, kesesuaian jumlah
bind dengan argumen yang disiapkan, kesamaan penyaring antara kueri daftar dan pencacahnya,
disiplin SQL portabel, larangan menulis, larangan memanggil rutin basis data, larangan
menembus DB Link, dan bahwa hanya kueri daftar yang memaginasi.
