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

## 17. Sesi keenam — Penjenjangan Komite dan Master Ambang (2026-09-17)

Tugasnya: **melanjutkan ke modul Flow Komite**, dengan `Flow/Komite_Flow.xml` sebagai
rujukan.

### 17.1 Tiga temuan yang mengoreksi premis tugas

Analisis dijalankan lebih dulu, dan tiga hal muncul yang mengubah bentuk pekerjaannya.

**Pertama — flow-nya sendiri hampir kosong; logikanya ada di tempat lain.**
`Flow/Komite_Flow.xml` hanya memuat **empat shape**: `Start1` → assignment `ASSIGNMENT63`
(router `KomiteRouter`, `pyImplementation=WorkList`) → decision `Decision1` (`KomiteLoop`)
→ `END52` (`Resolved-Completed`), ditambah satu putaran balik. Seluruh aturannya hidup di
rule yang dirujuknya, dan itulah yang dibaca:

| Rule | Isi yang sebenarnya |
|---|---|
| `Activity/KomiteRouter-Act.xml` | `KomiteCount` 1→4 memetakan ke worklist `komitepnc`…`komitepnc4`, **ditimpa** `.KomiteID` bila `.KomiteAproval==0`, dan `.Komite.KomiteID` bila `Primary.TransferType=='3'`. Router juga yang menaikkan `KomiteCount` dan mereset `AcceptStatus` |
| `When/IsKomiteLoop-When.xml` | `.AcceptStatus = "1"` **dan** `.KomiteCount <= .KomiteLoop` |
| `Activity/KomitePost_Adjustment`, `_Reject`, `_LiableKlaim`, `_Survey` | `AcceptStatus` `1`=setuju `2`=tolak; menulis `KomiteAproval`, `StatusClaim`, `StatusClaimCommittee`, `IsKomiteApprove`; saat tolak, perulangan dipaksa berhenti |
| `Database/INSERTDATAKOMITELIST.prc` | Datanya di `POOLDATA.T_CLAIM_KOMITE_LIST` — `KOMITE_ID`, `NO_KLAIM`, `NAMAKOMITE`, `STATUSAPPROVE`, `NOTEKOMITE`, `KOMITEKE`, `TYPEKOMITE`, `NILAIKLAIM`, `SHAREASM` |

**Kedua — `AutoAcceptKomite` bukan "menyetujui semua tiap hari".**
`TKT-B07-003` menyebutnya job harian jam 06:00 tanpa merinci syaratnya. Isinya ternyata
menyetujui komite yang `DateOfComitee`-nya sudah **lebih dari dua hari** menganggur —
`@DateTimeDifference(.Komite.DateOfComitee,@CurrentDateTime(),"D")>2` — lalu memanggil
`KomitePost_Adjustment`, menulis `KomiteAproval="1"` dengan catatan terpatri
`"Auto Accept by PEGA Claim Non MBU"`, menaikkan `KomiteCount`, dan mengirim ulang surel.
**Tanpa batas nilai maupun jenjang.** Itu eskalasi tenggat yang berubah menjadi
persetujuan, dan ia tidak tercatat di dokumen mana pun sebelum ini.

**Ketiga — `B-7` menggantung di udara.** Papan tiket menetapkan ia bergantung pada `B-5`
(nilai terkonversi), `B-6` (penugasan), dan `F-4` (master ambang). Aplikasi hari ini baru
punya `auth`, `portal`, dan `masterstatus`. Membangun layar keputusan komite sekarang
berarti mengarang entitas Klaim dan Adjustment — persis yang dilarang "No Shortcuts".

Ditambah satu kendala lingkungan: **Go dan Node tidak terpasang di mesin ini**, dan
`backend/.env` tidak ada. Dicari di PATH dan di lokasi umum pemasangan: nihil.

### 17.2 Empat pertanyaan konfirmasi dan jawabannya

| # | Pertanyaan | Jawaban Work Owner |
|---|---|---|
| 1 | Cakupan sesi ini, mengingat `B-5` dan `B-6` belum ada | **Master ambang + mesin penjenjangan.** Layar keputusan menyusul |
| 2 | Tabel warisan mana yang boleh ditulis (`P-1`) | **Baca saja dulu.** Pega tetap penulis tunggal |
| 3 | `AutoAcceptKomite` — dibawa? | **Jangan dibawa** |
| 4 | Go dan Node tidak terpasang | **Tulis kode, uji menyusul** |

Jawaban 2 menutup kemungkinan mengelola master dari aplikasi ini, dan itu yang membuat
seluruh kueri modul ini **tidak memuat satu pun pernyataan tulis** — ditegakkan oleh uji,
bukan hanya oleh niat.

### 17.3 Verifikasi mandiri terhadap ketujuh kasus spec

Sebelum menulis kode, ketujuh kasus jumlah penyetuju pada
`docs/ticketing/B-7-Komite-Persetujuan-Klaim/spec.md` dihitung ulang dengan tangan dari
`Database/emailkomite.csv`. **Ketujuhnya cocok**, dan itu yang menjadi dasar mempercayai
aturan kumulatifnya sebelum satu baris kode ditulis.

### 17.4 Yang dibangun

**Lapisan platform**

`internal/platform/uang` — tipe nilai uang berbasis `int64` satuan terkecil. Tanpa
dependensi pihak ketiga, tanpa `float` sama sekali. Ia dibutuhkan karena modul ini
membandingkan Rp 50.000.001 melawan Rp 50.000.000, dan selisih satu rupiah di sana
menentukan satu jenjang persetujuan ikut atau tidak.

`DariNilaiSQL` menangani setiap bentuk yang mungkin datang dari driver — `int64`,
`float64`, `[]byte`, `string`, `nil`. Memindai langsung ke `*string` **tidak aman**:
bila driver menyerahkan `float64`, `database/sql` memformatnya dengan `'g'`, dan
Rp 100.000.000 menjadi `"1e+08"` yang kemudian gagal diurai.

**Lapisan domain** — `internal/komite`

| Berkas | Isi |
|---|---|
| `ambang.go` | tipe `Ambang` (satu baris master), seam `Repo`, normalisasi penulisan |
| `jenjang.go` | `Tentukan` — fungsi murni, lima langkah, tanpa basis data |
| `integritas.go` | `PeriksaIntegritas` — satu-satunya pemakaian `LIMIT_TOP` yang dibenarkan `D-47` |
| `errors.go` | galat validasi yang mengumpulkan seluruh pelanggaran sekaligus |

**Lapisan adapter** — `repo/sqlstore` (Oracle, baca saja) dan `repo/memori` (30 baris
master nyata, diturunkan dari CSV).

**Lapisan transport** — `internal/komite/http`, tiga rute:

    GET /api/master/ambang-komite              tangga ambang + daftar lini + kebijakan pita
    GET /api/master/ambang-komite/integritas   temuan pemeriksaan master
    GET /api/komite/penjenjangan?nilai=&lini=  siapa saja yang harus menyetujui

Ketiganya GET karena tidak satu pun mengubah apa pun — akibat praktisnya, hasil
perhitungan **dapat ditautkan** dan dikirimkan apa adanya kepada Work Owner.

**Frontend** — `src/lib/uang.ts` (format dan urai rupiah), modul `ambang-komite` dengan
dua layar, tiga ikon baru, dan dua entri menu.

### 17.5 Empat keputusan rancangan yang perlu dijelaskan

**Penyaringan dikerjakan di Go, bukan di klausa `WHERE`.** Seluruh 30 baris dibaca, lalu
disaring di domain. Yang diperoleh: aturan penjenjangan hidup di **satu tempat** yang
dapat diuji tanpa basis data, dan perilakunya dijamin sama antara Oracle dan memori.
Menaruhnya di SQL akan memecahnya menjadi dua salinan yang dapat berbeda pendapat —
persis pola yang membuat sistem lama menyebarkan satu aturan ke activity, SQL, dan
procedure sekaligus.

**Urutan dibuat pasti, lebih pasti daripada Pega.** Kueri lama memakai `ORDER BY DEGREE`
saja. Pada master yang berlaku, Non-MBU pita 1 punya **dua baris ber-DEGREE 1** (ID 7 dan
ID 1), sehingga urutan keduanya diserahkan kepada basis data. Di sini seri dipecahkan
dengan ambang terkecil lebih dulu, lalu ID — dan setiap kali itu terjadi, penandanya
`UrutanTidakPasti` menyala sampai ke layar. **Siapa** yang menyetujui tidak berubah;
hanya urutannya saat seri.

**Kolom `EMAIL` dan `CC` tidak dibaca sama sekali.** Bukan dibaca lalu dibuang — tidak
pernah masuk ke dalam `SELECT`. `D-67` menetapkan alamat pribadi pada master lama tidak
dibawa, dan tidak membacanya sejak kueri membuatnya tidak pernah sampai ke peramban.

**Nilai uang dikirim sebagai teks kanonik, bukan angka JSON.** Angka JSON adalah floating
point ganda di peramban; mengirim uang lewatnya berarti menyerahkan ketepatannya kepada
pembulatan biner. Pemisah ribuan diurai **di layar**, bukan di server — artinya berbeda
antar bahasa, sehingga penafsirannya harus terjadi di tempat yang tahu bahasanya.

### 17.6 Dua pernyataan dokumen yang dikoreksi oleh bukti

**`TKT-B07-001` menulis** bahwa klaim PA dan Travel di atas Rp 200.000.000 "tidak punya
penyetuju sama sekali". Di bawah aturan kumulatif itu **tidak benar**: klaim sebesar apa
pun tetap memenuhi seluruh ambang bawah, sehingga justru mendapat **seluruh** penyetuju
pada tangga itu. Yang sebenarnya terjadi adalah tangganya **berhenti membedakan** di
nilai itu. `D-52` sudah menyatakan hal yang sama; tiketnya yang terlalu jauh.

Pemeriksaan integritas melaporkannya dengan kalimat yang tepat, sebagai **peringatan**
bukan cacat, dan uji `TestLimitTopTidakMenyaring` membuktikannya: PA Rp 5 miliar tetap
menghasilkan 4 penyetuju.

**Pertanyaan terbuka "baris `DEGREE=0` maksudnya apa?"** terjawab dari datanya sendiri.
Satu-satunya baris ber-DEGREE 0 yang masih aktif adalah ID 9, dan baris itu ber-`STS_ADJ`
**kosong** sementara `STS_REG`-nya menyala — ia penerima pemberitahuan registrasi, bukan
jenjang. Penyaring `STS_ADJ` sudah mengeluarkannya tanpa perlu aturan khusus tentang
DEGREE. Pengamatan ini **dilaporkan, bukan diputuskan**: yang menetapkan artinya tetap
Work Owner.

### 17.7 Kendala dan penyelesaiannya

| Kendala | Penyelesaian |
|---|---|
| Go dan Node tidak terpasang; tidak ada `.env` | Kode dan uji ditulis lengkap, dan **dinyatakan terus terang belum dijalankan**. Lihat §17.8 |
| Tipe kolom `EMAILKOMITE` tidak diketahui — DDL belum pernah dilihat (`R-08`) | Seluruh kolom dipindai ke `any` lalu ditafsirkan eksplisit, sehingga modul tahan terhadap kolom bertipe angka maupun teks. Penafsirannya diuji tersendiri |
| `OPERATOR_ID` baris ID 4 di CSV diakhiri **baris baru** | `Ambang.Bersih` merapikan seluruh field teks; diuji di `TestSpasiTepiDibuangDariDataMaster` |
| Pengelompokan integritas PA sempat salah | Bila dikelompokkan per `TYPE_KOMITE`, tangga PA terbelah dan tampak berlubang parah — padahal di PA kolom itu membedakan PA reguler dari PA TKI (`D-70`). Pengelompokan dibuat **per lini** untuk lini tanpa pita, **per pita** untuk Non-MBU |
| Atap pita bawah Non-MBU sempat dilaporkan sebagai peringatan | Itu peringatan palsu: berhenti tepat di batas pita justru benar, karena di atasnya klaim berpindah pita. Dikecualikan, dan pengecualiannya diuji |

### 17.8 Verifikasi — apa yang BELUM dijalankan

**Tidak satu pun uji di sesi ini pernah dijalankan.** Go dan Node tidak terpasang di mesin
ini, sehingga `go test`, `go build`, `npm test`, dan `tsc --noEmit` tidak dapat dieksekusi.
Basis data juga tidak dapat dihubungi — `backend/.env` tidak ada.

Ini menyimpang dari kebiasaan sesi-sesi sebelumnya, yang selalu ditutup dengan
"verifikasi yang benar-benar dijalankan". Penyimpangannya disetujui Work Owner sebagai
jawaban pertanyaan 4, dan dicatat di sini alih-alih disamarkan.

**Yang harus dijalankan sebelum modul ini dianggap berjalan:**

    cd backend
    gofmt -w ./...        # WAJIB DULUAN — berkas baru ditulis tanpa perkakas format
    go vet ./...
    go test ./...

    cd ../frontend
    npm run periksa-tipe
    npm test
    npm run build

`gofmt -w` disebut lebih dulu dengan sengaja: berkas Go di sesi ini ditulis tanpa dapat
menjalankan pemformatnya, sehingga perataan spasi pada literal struct dan blok konstanta
hampir pasti belum sesuai. Itu mekanis dan tidak menyentuh arti kode, tetapi `gofmt -l`
di CI akan menandainya bila dilewati.

**Uji yang menunggu dijalankan**, beserta apa yang dibuktikannya:

| Berkas | Yang dibuktikan |
|---|---|
| `komite/jenjang_test.go` | ketujuh kasus spec; pita hanya Non-MBU; `LIMIT_TOP` tidak menyaring; ambang tepat di batas; urutan pasti saat seri |
| `komite/integritas_test.go` | master yang berlaku bersih dari cacat; atap PA dan Travel dilaporkan; pengelompokan PA per lini |
| `komite/repo/sqlstore/kueri_test.go` | tidak ada satu pun kueri yang menulis; SQL portabel; surel tidak dibaca |
| `komite/http/rute_test.go` | ketujuh kasus lewat HTTP; uang sebagai teks; hanya GET yang tersedia |
| `platform/uang/uang_test.go` | perbandingan tepat pada selisih satu rupiah; notasi ilmiah ditolak |
| `lib/uang.test.ts` | format dan urai saling membalik tanpa menggeser nilai |
| `modules/ambang-komite/AmbangKomite.test.tsx` | nilai dikirim kanonik; perhitungan hanya setelah tombol; keraguan urutan tampil |

### 17.9 Yang tidak dibangun, dan kenapa

| Tidak dibangun | Sebab |
|---|---|
| Layar keputusan komite — setuju, tolak, kembalikan | `TKT-B07-002`; bergantung pada `B-5` dan `B-6` yang belum ada. Membangunnya sekarang menuntut mengarang entitas Klaim |
| Pencatatan jejak keputusan komite | idem, ditambah `S-5` yang belum ada |
| `AutoAcceptKomite` | Work Owner memutuskan **tidak dibawa** |
| Pengelolaan master ambang | Work Owner memutuskan **baca saja**; memindahkan kepemilikan menuntut prosedur `D-63` |
| Pemeriksaan peran pada rute | `TKT-F3-005`; keadaannya sama dengan seluruh rute lain hari ini |

### 17.10 Yang perlu dijawab Work Owner

1. **Beban bila `AutoAcceptKomite` dihapus belum dihitung.** `TKT-B07-003` menuntut
   angkanya dilaporkan sebelum rilis. Kuerinya menunggu akses basis data:

       SELECT COUNT(*) AS jumlah_auto_accept
         FROM POOLDATA.T_CLAIM_KOMITE_LIST
        WHERE NOTEKOMITE LIKE 'Auto Accept by PEGA%'
          AND DATEOFCOMMITE_CREATE >= SYSDATE - 90;

2. **Arti `DEGREE=0`** — pengamatan pada §17.6 menunggu ditegaskan atau dikoreksi.
3. **Apakah `dbms_random.value` pada dua kueri Simasnet disengaja.** Bila ya, jalur itu
   tidak dapat diuji secara deterministik, dan aturannya di sini harus ditinjau ulang.
4. **Batas pita Rp 100.000.000** masih berupa angka di dalam kode — satu-satunya nilai
   bisnis yang demikian di modul ini. Ia sudah dapat dipasok dari luar; yang belum ada
   adalah tabel masternya.

---

## 18. Sesi keenam, lanjutan — empat jawaban Work Owner (2026-09-18)

Keempat pertanyaan terbuka pada §17.10 dijawab, dan **dua di antaranya membatalkan kode
yang sudah ditulis sehari sebelumnya**. Bagian ini mencatat apa yang berubah dan atas
dasar bukti apa.

### 18.1 Jawaban dan akibatnya

| # | Pertanyaan | Jawaban | Akibat |
|---|---|---|---|
| 1 | `AutoAcceptKomite` | **Jangan dihapus, tapi dikonversi** | Membatalkan keputusan 2026-09-17 "tidak dibawa". Aturannya dibangun |
| 2 | Arti `DEGREE=0` | **Tidak dipakai** | Penyaringnya ditulis eksplisit |
| 3 | `dbms_random` disengaja? | **Ya** — mengacak operator, dan **penginput dikecualikan** | Membatalkan larangan yang saya pasang. Mode kedua dibangun |
| 4 | Batas pita Rp 100 juta dari mana? | dijawab dengan bukti — lihat §18.5 | Ditemukan **kembaran USD 7.000** yang belum pernah tercatat |

### 18.2 Jawaban 3 mengoreksi kesalahan saya, dan ini yang paling serius

Sehari sebelumnya saya menulis di `kueri_test.go`:

> `{"DBMS_RANDOM", "urutan penyetuju harus pasti, bukan diacak"}`

dengan alasan "pengacakan membuat hasilnya tidak dapat dibandingkan saat uji kesetaraan".
Saya memperlakukan pengacakan sebagai cacat yang tidak perlu dibawa.

**Itu keliru.** Work Owner menjelaskan maksudnya: bila untuk klaim di bawah nilai tertentu
ada tiga orang berwenang — A, B, dan C — dan **A yang menginput**, maka sistem mengacak
penyetujunya menjadi B atau C.

Penelusuran ke sumber membenarkan penjelasan itu sepenuhnya, dan menemukan bahwa aturannya
**lebih tegas** daripada yang digambarkan. `RDB List/EmailKomiteBerjenjangSimasnet_sql-SQL.xml`:

```sql
SELECT * FROM (
  SELECT EMAIL, DEGREE, OPERATOR_ID, CC
    FROM POOLDATA.EMAILKOMITE
   WHERE STS_ADJ = '1' AND STS_AKTIF = '1'
     AND trim(TYPE_BUSINESS) = trim({tempAdj.pyMemo})
     AND LIMIT_BOTTOM <= {tempAdj.ConvertAdjustmentValue}
     {ASIS:tempAdj.UploadLOD}
   ORDER BY degree, dbms_random.value)
{ASIS:tempAdj.IsDLA}
```

dengan kedua potongan dinamisnya diisi `Activity/SetEmailKomiteSimasnet-Act.xml`:

| Properti | Isi |
|---|---|
| `tempAdj.UploadLOD` | `"AND OPERATOR_ID!='" + OperatorID.pyUserIdentifier + "'"` |
| `tempAdj.IsDLA` | `"WHERE rownum = 1"` |

Jadi jalur ini: **mengecualikan operator yang sedang menginput**, mengurutkan menurut
jenjang lalu acak, dan **mengambil TEPAT SATU baris**.

**Ia bukan variasi kecil dari aturan kumulatif — ia aturan yang berbeda sama sekali.**
Kumulatif menghasilkan beberapa penyetuju berurutan; jalur ini menghasilkan satu.

Dan ini **kontrol pemisahan tugas yang nyata**, satu-satunya yang ditemukan sejauh ini.
`D-59` menetapkan tidak ada pemisahan tugas formal di sistem lama; temuan ini tidak
membatalkannya — cakupannya hanya satu entitas — tetapi ia menunjukkan bahwa pernyataan
itu tidak berlaku mutlak.

### 18.3 Siapa yang memilih mode, dan kenapa itu urusan portal

`Activity/SetListComiteeClaimPerObjAdj-Act.xml`:

```
bila TempGetApp.LSC_ID == "SIMASNET"  → Call SetEmailKomiteSimasnet
selain itu                            → Call SetEmailKomite
```

`LSC_ID` dibaca `GetLinkAppClaim` dari `POOLDATA.DB_LINK_PEGA` dengan **mencocokkan nama
server**. `D-75` mengganti pengenalan berbasis nama server itu dengan **portal**, sehingga
di sistem baru mode penjenjangan **melekat pada portal**, bukan pada lini bisnis.

Itulah sebabnya `Mode` menjadi medan pada `Kebijakan` — satu kebijakan per portal — dan
bukan pada `Ambang` maupun `Lini`.

### 18.4 Yang dibangun untuk jawaban 3

| Hal | Letak |
|---|---|
| `Mode` — `kumulatif` dan `satu-penyetuju` | `komite/jenjang.go` |
| `KebijakanSimasnet()` — satu penyetuju, tanpa pita | idem |
| Pengecualian penginput, tahan terhadap perbedaan penulisan | `KunciOperator` di `komite/ambang.go` |
| Seam `Pengacak`, dengan `PengacakTetap` untuk pengujian | `komite/jenjang.go` |
| Pengacak sungguhan | `internal/platform/acak` |
| `Kandidat` dan `DikecualikanPenginput` dilaporkan sampai ke layar | DTO dan layar |

**Pengacakan pindah dari SQL ke Go, dan itu disengaja.** Yang dilarang `kueri_test.go`
sekarang bukan perilakunya melainkan **tempatnya**: diacak di dalam SQL membuat aturannya
tidak dapat diuji sama sekali. Di balik seam, pengujian memakai pemilih tetap sementara
produksi tetap mengacak — dan aturannya, yaitu siapa saja yang layak, tetap dapat diperiksa.

**Kandidat ikut dikirim ke layar** karena pada mode ini yang dapat diperiksa bukan siapa
yang terpilih — itu acak — melainkan apakah kumpulan yang layak sudah benar. Layar
menyatakannya terus terang: hasil yang berbeda pada nilai yang sama **bukan** cacat.

**Yang belum dapat dibuktikan:** master yang diserahkan **tidak memuat satu baris pun**
ber-`TYPE_BUSINESS` SIMASNET. Itu konsisten dengan `ADR-0030` — satu basis data per
entitas — sehingga baris Simasnet hidup di basis data portal Insurtech. Mode ini karena
itu diuji dengan data buatan yang meniru keadaan yang digambarkan Work Owner, dan **belum
pernah dijalankan terhadap data sungguhan**.

### 18.5 Jawaban 4 — dari mana angka Rp 100.000.000

Pertanyaannya tepat sasaran, dan jawabannya bukan "dari Decision Log".

Angkanya **ada di dalam rule Pega**, sebagai satu ekspresi di
`Activity/SetEmailKomite-Act.xml`:

```
tempAdj.AcceptedNo := @If(tempAdj.ConvertAdjustmentValue > 100000000, 2, 1)
```

dan `tempAdj.AcceptedNo` itulah yang menyaring `trim(TYPE_KOMITE) = trim({tempAdj.AcceptedNo})`
pada `EmailKomiteBerjenjang_sql`. `D-52` kemudian menegaskannya dengan kalimat Work Owner
sendiri — *"karena untuk komite sampai 100 Jt pakai type_komite=1"* — sehingga rule dan
keputusan sepakat.

**Yang belum pernah tercatat, dan ditemukan saat menelusuri ini:** rule yang sama memuat
kembarannya untuk entitas bermata uang dolar, empat baris di bawahnya:

```
tempAdj.AcceptedNo := @If(tempAdj.ConvertAdjustmentValue > 7000, 2, 1)
```

Jadi batas pitanya **Rp 100.000.000 pada entitas rupiah dan USD 7.000 pada entitas SMI**.
Rasio keduanya — 100.000.000 ÷ 7.000 ≈ **14.285** — adalah kurs yang dibekukan ke dalam
kode, bukan kurs yang dibaca dari master mata uang.

Akibatnya bagi kode sesi kemarin: konstanta tunggal `BatasPitaNonMBUBawaan` **benar untuk
portal rupiah dan salah untuk portal SMI**, dengan selisih sekitar 14.000 kali lipat.
Kembarannya kini dicantumkan sebagai `BatasPitaNonMBUSMI` — bukan supaya dipakai apa
adanya, melainkan supaya perbedaannya terlihat. Kurs 14.285 sudah tidak mencerminkan kurs
hari ini, sehingga angkanya wajib dikonfirmasi ulang sebelum portal SMI dilayani.

### 18.6 Jawaban 2 — `DEGREE=0` disaring eksplisit

`JenjangPersetujuan()` kini menuntut `Jenjang > 0` di samping `STS_AKTIF` dan `STS_ADJ`.

Hari ini penyaring itu **tidak mengubah satu hasil pun**: baris ber-DEGREE 0 yang masih
aktif hanya ID 9, dan ia sudah tersaring lebih dulu karena `STS_ADJ`-nya kosong.

Menuliskannya tetap perlu, karena tanpa itu aturannya hanya **berlaku secara kebetulan**.
Satu baris baru ber-DEGREE 0 dengan `STS_ADJ` menyala akan diam-diam ikut menyetujui uang,
dan tidak ada yang akan menyadarinya. Diuji dengan baris yang seluruh syarat lainnya
terpenuhi.

### 18.7 Jawaban 1 — `AutoAcceptKomite` dikonversi, bukan dihapus

Keputusan 2026-09-17 dibatalkan. Yang dibangun sekarang adalah **aturannya**, di
`komite/otomatis.go`:

| Hal | Perlakuan |
|---|---|
| Syarat lama menganggur | **ditiru** — inti aturannya, dan rule lama tidak punya syarat lain |
| Pelaku persetujuan | **diperbaiki** — dicatat sebagai `SISTEM`, bukan dititipkan pada kalimat di kolom catatan |
| Batas nilai dan batas jenjang | **ditambahkan sebagai kemungkinan, mati secara bawaan** |
| Fitur secara keseluruhan | **mati secara bawaan** |

Dua yang terakhir perlu dijelaskan. Batas nilai dan jenjang **mati** supaya kesetaraan
dengan Pega tetap terjaga — sistem lama tidak membatasi keduanya sama sekali. Begitu Work
Owner menjawab "jenjang dan nilai mana yang boleh disetujui otomatis", jawabannya tinggal
diisi, bukan ditulis ulang.

Fiturnya sendiri mati secara bawaan karena job ini **melewati seluruh kontrol otorisasi**:
`D-59` menetapkan izin bersatuan menu, dan job tidak punya pengguna sehingga tidak ada menu
yang dapat diperiksa. Sesuatu yang menyetujui uang tanpa kontrol tidak boleh menyala hanya
karena kelalaian menyetelnya.

**Satu ketidakpastian yang tidak saya tutupi.** Syarat aslinya berbunyi:

```
@DateTimeDifference(.Komite.DateOfComitee, @CurrentDateTime(), "D") > 2
```

`> 2` dapat berarti **lewat 48 jam** (bila fungsinya mengembalikan pecahan) atau **genap
3 hari** (bila ia mengembalikan bilangan bulat terpotong). Export tidak menyelesaikannya,
dan selisihnya satu hari penuh pada aturan yang menyetujui uang tanpa manusia.

Yang dipakai adalah bacaan yang **lebih lambat menyetujui** — genap 3 hari — karena bila
tebakannya keliru, akibatnya klaim menunggu sehari lebih lama, bukan uang yang telanjur
disetujui sendiri oleh sistem. Ia parameter, bukan konstanta, sehingga dapat dikoreksi
tanpa rilis ulang setelah diverifikasi terhadap Pega staging.

**Yang belum dibangun, dan kenapa:** penulisan persetujuannya dan penjadwalnya. Keduanya
menuntut jalur keputusan komite (`TKT-B07-002`, bergantung `B-5` dan `B-6`) dan mekanisme
penjadwal (`S-6`), yang belum ada — dan menulis ke `T_CLAIM_KOMITE_LIST` melanggar
keputusan "baca saja" yang masih berlaku.

Satu hal lagi yang perlu disadari saat menyalakannya kelak: sistem lama **menyetel ulang**
`DateOfComitee` setiap kali sebuah jenjang disetujui otomatis, sehingga hitungan harinya
dimulai lagi dari nol untuk jenjang berikutnya. Klaim yang seluruh jenjangnya disetujui
otomatis karena itu membutuhkan waktu sebanyak jumlah jenjangnya dikali ambangnya — bukan
sekali saja.

### 18.8 Verifikasi

**Masih belum dijalankan.** Go dan Node tetap tidak terpasang di mesin ini. Perintah yang
harus dijalankan sama dengan §17.8, ditambah berkas uji baru:

| Berkas | Yang dibuktikan |
|---|---|
| `komite/mode_test.go` | mode satu-penyetuju memilih tepat satu; penginput dikecualikan dan pencocokannya tahan penulisan; jenjang terendah yang dipilih; pengacak di balik seam; `DEGREE=0` tidak pernah menyetujui; batas pita dapat berbeda antar entitas |
| `komite/otomatis_test.go` | fitur mati secara bawaan; ambang diuji tepat di batas; batas nilai dan jenjang menolak yang melampaui; yang tidak layak dikembalikan beserta alasannya |
| `komite/http/rute_test.go` | mode dikirim pada kedua respons; kandidat dilaporkan; penginput diabaikan pada mode kumulatif |
| `modules/ambang-komite/AmbangKomite.test.tsx` | isian Operator ID hanya muncul pada mode yang memakainya; calon dan pengecualian tampil |

### 18.9 Yang masih menunggu jawaban

1. **Ambang `> 2` itu 48 jam atau 72 jam** — hanya dapat dipastikan dengan menjalankan
   Pega staging.
2. **Nilai dan jenjang mana yang boleh disetujui otomatis.** Keduanya sudah ada tempatnya
   di `KebijakanOtomatis`, tinggal diisi.
3. **Batas pita untuk portal SMI** — USD 7.000 berasal dari kurs beku 14.285 yang sudah
   tidak berlaku.
4. **Master ambang portal Simasnet** belum pernah dilihat, sehingga mode satu-penyetuju
   belum teruji terhadap data sungguhan.
5. **Menyambungkan kebijakan ke portal aktif** — `TKT-F6-002`. Hari ini seluruh portal
   memakai kebijakan rupiah kumulatif.

---

## 19. Empat jawaban penutup, dan satu akibat yang harus diketahui sebelum rilis (2026-09-18)

### 19.1 Jawaban

| # | Pertanyaan | Jawaban | Akibat pada kode |
|---|---|---|---|
| 1 | `> 2` itu 48 atau 72 jam? | **72 jam** | Ambiguitas tertutup; bawaan 3 hari dipastikan benar |
| 2 | Nilai dan jenjang mana yang boleh auto-accept? | **Tidak dibatasi** — selama `KomiteCount <= KomiteLoop` | Dua medan konfigurasi **dihapus**, syarat jenjang ditambahkan |
| 3 | Batas pita portal SMI | *(belum dijawab)* | tetap terbuka |
| 4 | Pengecualian penginput berlaku di entitas lain? | **Ya** | Pengecualian menjadi **berlaku di semua mode** |

### 19.2 Jawaban 2 menutup pertanyaan, dan karena itu dua knob dihapus

Sehari sebelumnya saya membangun `BatasNilai` dan `JenjangMaksimum` sebagai tempat bagi
jawaban yang belum ada, mati secara bawaan. Jawabannya kini datang: **tidak ada batas
apa pun**; syaratnya hanya lama menganggur dan jenjangnya belum habis.

Keduanya **dihapus, bukan dibiarkan mati**. Konfigurasi yang tidak pernah dipakai adalah
jalur yang tidak pernah diuji, dan ia menyiratkan kemampuan yang tidak diminta siapa pun.
Menambahkannya kembali kelak lebih murah daripada memeliharanya sekarang.

Sebagai gantinya, `KomiteTertunda` kini membawa `JumlahJenjang` dan aturannya memeriksa
`MasihDalamJenjang()` — `KomiteCount <= KomiteLoop`, diambil apa adanya dari
`When/IsKomiteLoop-When.xml`. Ia diperiksa di dalam aturannya, bukan diandaikan sudah
disaring pemanggil: aturan yang menyetujui uang tidak boleh bergantung pada asumsi bahwa
masukannya sudah bersih.

### 19.3 Jawaban 4 adalah perubahan perilaku, dan akibatnya sudah dihitung

Pengecualian penginput kini berlaku pada **kedua mode**. Di sistem lama ia hanya ada pada
satu kueri — jalur Simasnet.

**Ini bukan peniruan melainkan perubahan aturan**, dan pada mode kumulatif akibatnya
langsung: jumlah penyetuju **berkurang satu** setiap kali penginputnya kebetulan anggota
komite pada tangga itu.

Yang lebih perlu diketahui: dihitung dari `Database/emailkomite.csv`, **setiap lini punya
jenjang terendah yang diisi SATU orang saja**. Bila orang itu yang mengajukan, klaimnya
berakhir **tanpa penyetuju sama sekali** — bukan berkurang satu, melainkan berhenti.

| Lini | Rentang nilai | Satu-satunya penyetuju |
|---|---|---|
| NONMBU pita 1 | ≤ Rp 50.000.000 | ELLENSUPRIYATI |
| NONMBU pita 2 | Rp 100.000.001 – Rp 500.000.000 | BAMBANGSETIADJIGUNAWAN |
| NONMBUAB | seluruhnya | ELLENSUPRIYATI |
| NONMBUC | seluruhnya | ELLENSUPRIYATI |
| PA | ≤ Rp 10.000.000 | WAHYUKRISTANTI |
| TRAVEL | ≤ Rp 50.000.000 | RATNAGUSNITASARI |
| BONDING | seluruhnya | RIZALGREATLIN |

Ketujuhnya diuji sebagai kasus yang **harus** menghasilkan nol penyetuju
(`TestPengecualianPenginputDapatMenghabiskanSeluruhPenyetuju`) — bukan supaya dianggap
benar, melainkan supaya keadaannya terlihat sebelum ada klaim nyata yang berhenti
karenanya.

**Seberapa mungkin terjadi** bergantung pada apakah anggota komite juga menginput klaim.
Satu petunjuk: ELLENSUPRIYATI muncul di `Activity/SetEmailKomite-Act.xml` sebagai
`UserTeknis` pada logika lama berbasis nama — artinya ia juga PIC Teknik, bukan hanya
anggota komite. Itu menaikkan kemungkinannya, dan pantas dipastikan ke pengguna bisnis.

**Yang dikerjakan supaya tidak gagal diam-diam:**

- `Tersingkir` dibedakan dari `DikecualikanPenginput`. Yang pertama menyatakan siapa yang
  **benar-benar** keluar; yang kedua hanya siapa yang diminta dikecualikan. Penginput yang
  bukan anggota komite tidak mengubah apa pun, dan layar tidak boleh menyiratkan
  sebaliknya.
- Layar membedakan **dua sebab** klaim tanpa penyetuju: master yang tidak menjangkau nilai
  itu, versus seluruh penyetuju tersingkir karena penginputnya. Keduanya menuntut tindakan
  dari orang yang berbeda.
- Isian Operator ID pengaju kini muncul pada semua mode, tidak lagi hanya Simasnet.

### 19.4 Yang masih terbuka

1. **Batas pita portal SMI** — USD 7.000 berasal dari kurs beku 14.285 yang sudah tidak
   berlaku. Belum dijawab.
2. **Master ambang portal Simasnet** belum pernah dilihat; mode satu-penyetuju masih diuji
   dengan data buatan.
3. **Seberapa sering anggota komite menginput klaim sendiri** — menentukan apakah ketujuh
   keadaan di §19.3 adalah risiko nyata atau kemungkinan teoretis. Pertanyaan untuk
   pengguna bisnis, bukan untuk kode.
4. **Menyambungkan kebijakan ke portal aktif** — `TKT-F6-002`.

---

## 20. Penelusuran tiga pertanyaan Work Owner (2026-09-19)

### 20.1 Batas USD 7.000 adalah PENIMPA, bukan cabang yang setara

Pertanyaannya: "ini maksudnya bagaimana, dan dipanggil di mana?"

Keduanya tidak berdampingan sebagai dua cabang. Di `Activity/SetEmailKomite-Act.xml`
urutannya:

| Urutan | Syarat | Yang dikerjakan |
|---|---|---|
| lebih dulu | **tanpa syarat apa pun** | `tempAdj.AcceptedNo := @If(ConvertAdjustmentValue > 100000000, 2, 1)` |
| sesudahnya | `TempGetApp.LSC_ID == "SMI"` | **menimpanya** dengan `@If(ConvertAdjustmentValue > 7000, 2, 1)` |

Diverifikasi langsung: blok yang memuat ekspresi `>100000000` **tidak punya satu pun**
`pyStepsPreCondParamsWhen`; blok yang memuat `>7000` punya tepat satu, yaitu
`TempGetApp.LSC_ID=="SMI"`.

Jadi **Rp 100.000.000 adalah nilai bawaan yang berlaku bagi semua entitas**, dan
**USD 7.000 hanya berlaku bagi SMI**, dengan cara menimpa.

`AcceptedNo` kemudian menjadi penyaring `trim(TYPE_KOMITE) = trim({tempAdj.AcceptedNo})`
pada `EmailKomiteBerjenjang_sql`.

**Rantai pemanggilannya:**

```
ValidationTypePaymentAdj
  └─ SetListComiteeClaimPerObjAdj        (atau SetListComiteeClaimAI)
       └─ SetEmailKomite                 ← kedua ekspresi di atas ada di sini
            └─ RDB EmailKomiteBerjenjang_sql
```

**Bahaya yang melekat pada cara entitas dikenali.** `LSC_ID` tidak datang dari
konfigurasi melainkan dari pencocokan **nama server**:

```sql
-- RDB List/BrowseAPPName_sql-SQL.xml, dijalankan GetLinkAppClaim
SELECT APP AS "DOCUMENTNAME" FROM pooldata.DB_LINK_PEGA
 WHERE APPIP LIKE '%<nama server>%'
```

dan bila pencocokan gagal, `GetLinkAppClaim` jatuh ke nilai bawaan `"ASM"`. Akibatnya:
server SMI yang berganti nama membuat klaim berdenominasi **dolar** diperiksa terhadap
ambang **rupiah** — meleset sekitar 14.000 kali lipat, **tanpa satu pun galat**. Inilah
yang `D-75` hapus dengan mengganti pengenalan berbasis nama server menjadi portal.

**Di aplikasi ini, konstanta `BatasPitaNonMBUSMI` belum dipakai jalur mana pun.** Ia
menunggu kebijakan disambungkan ke portal aktif (`TKT-F6-002`).

### 20.2 Master Simasnet: tabel yang SAMA, disaring TYPE_BUSINESS

Pertanyaannya: "Master Simasnet dipanggil di mana / ada di activity mana?"

**Tidak ada tabel master tersendiri.** Yang dibaca tetap `POOLDATA.EMAILKOMITE`, hanya
dengan penyaring lini yang berbeda:

| Lapisan | Berkas |
|---|---|
| Activity | `Activity/SetEmailKomiteSimasnet-Act.xml` — dua step `RDB-LIST` |
| Kueri | `RDB List/EmailKomiteBerjenjangSimasnet_sql-SQL.xml` |
| Tabel | `POOLDATA.EMAILKOMITE`, `WHERE trim(TYPE_BUSINESS) = trim({tempAdj.pyMemo})` |
| Nilai penyaring | `tempAdj.pyMemo` diisi **`"SIMASNET"`** atau **`"SIMASNETA"`** |
| Pemanggil | `SetListComiteeClaimPerObjAdj`, saat `TempGetApp.LSC_ID == "SIMASNET"` |

**Kenapa barisnya tidak ada di CSV yang kita terima.** `Database/emailkomite.csv` berisi
`TYPE_BUSINESS` NONMBU, NONMBUAB, NONMBUC, PA, TRAVEL, dan BONDING — tidak satu pun
SIMASNET. Itu konsisten dengan `ADR-0030`: satu basis data per entitas, dan CSV tersebut
berasal dari basis data portal **ASM**. Baris Simasnet hidup di
`POOLDATA.EMAILKOMITE` milik instans Simasnet.

**Yang perlu diminta supaya mode satu-penyetuju dapat diuji terhadap data nyata:**

```sql
SELECT ID, NAME, OPERATOR_ID, TYPE_BUSINESS, TYPE_KOMITE,
       LIMIT_BOTTOM, LIMIT_TOP, DEGREE,
       STS_AKTIF, STS_ADJ, STS_REG, STS_REJECT, STS_ABS
  FROM POOLDATA.EMAILKOMITE
 WHERE TRIM(TYPE_BUSINESS) IN ('SIMASNET', 'SIMASNETA')
 ORDER BY TYPE_KOMITE, LIMIT_BOTTOM, DEGREE, ID;
```

dijalankan pada basis data **portal Simasnet**, bukan portal ASM. Kolom EMAIL dan CC
sengaja tidak diminta (`D-67`).

### 20.3 Kenapa Simasnet mengecualikan pengaju — dan kenapa alasannya tidak berlaku di lini lain

Work Owner menjelaskan: **di Simasnet, komitenya adalah tim klaim itu sendiri.** Yang
mengajukan dan yang menyetujui berasal dari kumpulan orang yang sama.

Itu menjelaskan seluruh bentuk aturannya sekaligus:

- **Pengecualian pengaju bukan tambahan, melainkan INTI-nya.** Tanpa itu, seseorang akan
  menyetujui pekerjaannya sendiri — dan karena kumpulannya sama, itu bukan kemungkinan
  kecil melainkan kejadian sehari-hari.
- **Pengacakan mengikuti dari situ.** Bila tiga orang sama-sama berwenang, mengambil satu
  secara acak menyebar beban alih-alih selalu menjatuhkannya pada orang yang sama.
- **`WHERE rownum = 1` masuk akal.** Komite sesama rekan tidak berjenjang; satu tanda
  tangan dari rekan yang lain sudah cukup.

**Yang perlu disadari:** alasan itu **tidak berlaku di entitas lain**. Di sana komitenya
kelompok senior tersendiri — Dr. Wahyu, Bambang, Daniel, Marten — bukan tim klaim. Tangga
mereka juga tipis: setiap lini punya jenjang terendah yang diisi SATU orang.

Itulah sebabnya keputusan `D`-2026-09-18 (pengecualian berlaku di semua entitas)
menghasilkan tujuh keadaan tanpa penyetuju yang tercatat di §19.3 — bukan karena
aturannya salah diterapkan, melainkan karena di luar Simasnet tidak ada "rekan lain" yang
tersisa.

Pertanyaan itu diajukan ulang ke Work Owner dengan konsekuensinya, karena penjelasan pada
§20.3 adalah keterangan BARU yang belum tersedia saat keputusan sebelumnya diambil.

### 20.4 Keputusan pengecualian pengaju ditegaskan ulang

Pertanyaan §20.3 diajukan kembali kepada Work Owner pada 2026-09-19, lengkap dengan
ketujuh keadaan tanpa penyetuju yang sudah dihitung dan tiga jalan keluar: melengkapi
master pada titik-titik tipisnya, membatasi pengecualian kembali ke Simasnet saja, atau
menaikkan klaim ke jenjang berikutnya bila penyetujunya habis.

**Jawabannya: tetap seperti keputusan 2026-09-18.** Pengecualian berlaku di seluruh
entitas apa adanya; klaim yang diajukan oleh satu-satunya penyetujunya akan berhenti dan
ditangani manual.

Tidak ada perubahan kode — yang terpasang sudah persis demikian. Yang ditambahkan hanya
catatan pada `keluarkanPenginput`, supaya pembaca berikutnya tahu perilaku itu **keputusan
yang diambil setelah akibatnya diketahui**, bukan kelalaian yang menunggu diperbaiki.

### 20.5 Ringkasan pertanyaan terbuka setelah sesi ini

| # | Terbuka | Pemilik |
|---|---|---|
| 1 | Isi master ambang portal **Simasnet** belum pernah dilihat — kueri pengambilannya ada di §20.2 | DBA / Tim Simasnet |
| 2 | Batas pita portal **SMI** — USD 7.000 berasal dari kurs beku 14.285 yang sudah tidak berlaku | Work Owner |
| 3 | Menyambungkan kebijakan ke **portal aktif** (`TKT-F6-002`) — hari ini seluruh portal memakai kebijakan rupiah kumulatif | Lead Engineer |
| 4 | Penulisan persetujuan otomatis dan penjadwalnya — menunggu `B-5`, `B-6`, dan `S-6` | — |

---

## 21. Dua koreksi dari Work Owner (2026-09-19)

### 21.1 Simasnet berjalan di basis data ASM — dugaan saya keliru

Saya menyimpulkan bahwa baris `TYPE_BUSINESS = 'SIMASNET'` tidak ada di
`Database/emailkomite.csv` **karena** ia hidup di basis data portal Insurtech, dan
menyandarkannya pada `ADR-0030` (satu basis data per entitas).

**Work Owner mengoreksi: Simasnet tetap dijalankan di basis data ASM.**

Maka kesimpulannya berubah sepenuhnya. Baris `SIMASNET` dan `SIMASNETA` seharusnya berada
di `POOLDATA.EMAILKOMITE` **yang sama** dengan baris NONMBU, PA, dan Travel — dan
ketiadaannya di berkas yang diserahkan berarti **ekspornya belum lengkap**, bukan datanya
ada di tempat lain.

Yang dibutuhkan karena itu lebih sederhana daripada yang saya tulis sebelumnya: kueri yang
SAMA, di basis data yang SAMA, hanya tanpa penyaring:

```sql
SELECT ID, NAME, OPERATOR_ID, TYPE_BUSINESS, TYPE_KOMITE,
       LIMIT_BOTTOM, LIMIT_TOP, DEGREE,
       STS_AKTIF, STS_ADJ, STS_REG, STS_REJECT, STS_ABS
  FROM POOLDATA.EMAILKOMITE
 ORDER BY TYPE_BUSINESS, TYPE_KOMITE, LIMIT_BOTTOM, DEGREE, ID;
```

Tanpa `WHERE` sama sekali — supaya ketidaklengkapan yang sama tidak terulang. Kolom EMAIL
dan CC sengaja tidak diminta (`D-67`).

Catatan yang menyandarkan diri pada dugaan lama sudah dikoreksi di `komite/mode_test.go`
dan `komite/http/rute_test.go`.

**Satu hal yang perlu ditegaskan sebelum mode satu-penyetuju dipakai:** bila baris
`SIMASNET` memang tidak ada di tabel itu hari ini, maka jalur
`EmailKomiteBerjenjangSimasnet_sql` mengembalikan NOL baris, dan klaim Simasnet tidak
mendapat penyetuju sama sekali. Itu pertanyaan untuk DBA — apakah ekspornya yang disaring,
atau tabelnya yang memang belum berisi.

### 21.2 Berkasnya 29 record, bukan 30 — dan dua "lini" palsu yang sempat saya laporkan

Saat memeriksa ulang, ditemukan bahwa pembacaan per baris terhadap berkas itu **tidak
sahih**: OPERATOR_ID pada baris ID 4 diakhiri BARIS BARU di dalam tanda kutip, sehingga
satu record membentang di dua baris fisik.

Akibatnya, ringkasan "per lini" yang saya hasilkan dengan `awk` sempat memunculkan dua
nilai TYPE_BUSINESS yang tidak pernah ada — `NJOMANSUDARTHA` dan `1` — yang keduanya
sebenarnya pecahan kolom lain dari record yang terbelah.

Dibaca dengan pengurai CSV yang benar:

| Ukuran | Nilai |
|---|---|
| Record logis | **29** (bukan 30) |
| Jenjang persetujuan aktif | **15** |
| TYPE_BUSINESS yang ada | NONMBU, NONMBUAB, NONMBUC, PA, TRAVEL, BONDING — **tidak ada SIMASNET** |

**Yang penting: tangga jenjangnya tidak berubah.** Kelima belas baris aktif hasil
pembacaan ulang **cocok persis** dengan isi `repo/memori/contoh.go`, sehingga ketujuh kasus
pada `spec.md` dan seluruh uji penjenjangan tetap berdiri. Yang keliru hanya ringkasan
sekali pakai yang saya cetak, bukan data yang dipakai kode.

Catatan pada `contoh.go` sudah dikoreksi: "30 baris" menjadi 29 record, beserta sebab
terbelahnya dan alasan hanya 19 dari 29 yang dibawa.

**Pelajaran yang pantas dicatat:** ini kesalahan ketiga di sesi ini yang berpola sama —
alat ukur dipercaya sebelum divalidasi. CSV yang memuat baris baru di dalam kolom tidak
dapat dibaca `awk` per baris, dan yang membongkarnya adalah angka yang terlihat janggal,
bukan pemeriksaan yang direncanakan.

### 21.3 Ekspresi pita memang penentu TYPE_KOMITE — dikonfirmasi

Work Owner menegaskan bahwa `@If(ConvertAdjustmentValue > 100000000, 2, 1)` dan kembaran
USD-nya memang **untuk mengambil type komite**-nya.

Itu sejalan dengan penelusuran §20.1: hasilnya disimpan ke `tempAdj.AcceptedNo`, dan
`AcceptedNo` dipakai apa adanya sebagai penyaring
`trim(TYPE_KOMITE) = trim({tempAdj.AcceptedNo})` pada `EmailKomiteBerjenjang_sql`.

Dengan itu, pemetaan pita di aplikasi ini terkonfirmasi benar: `KebijakanPita.Pilih`
mengembalikan `"1"` atau `"2"`, dan nilai itu dicocokkan ke kolom `TYPE_KOMITE` —
persis seperti yang dikerjakan rule lama.

### 21.4 Ralat: `LSC_ID` hanya pengecekan server, bukan keterangan letak data

Work Owner meralat jawaban pada §21.1: **pencocokan `DB_LINK_PEGA` itu hanya pengecekan
server mana yang menjalankan aplikasi** — ia tidak menyatakan apa pun tentang di basis data
mana barisnya tersimpan.

**Yang menjadi lebih kuat.** Ralat ini justru menegaskan penelusuran §20.1:

```sql
-- RDB List/BrowseAPPName_sql-SQL.xml, dijalankan GetLinkAppClaim
SELECT APP AS "DOCUMENTNAME" FROM pooldata.DB_LINK_PEGA
 WHERE APPIP LIKE '%<nama server>%'
```

adalah **deteksi lingkungan**, bukan konfigurasi. Dan karena `GetLinkAppClaim` jatuh ke
nilai bawaan `"ASM"` saat pencocokan gagal, server yang berganti nama akan diam-diam
memakai perilaku entitas yang salah — termasuk ambang pita rupiah untuk klaim dolar.
Itulah yang `D-75` hapus dengan mengganti deteksi nama server menjadi portal.

**Yang kembali terbuka.** Kesimpulan §21.1 — "ekspornya belum lengkap, bukan datanya ada di
tempat lain" — **ditarik**. Yang tersisa sebagai fakta hanyalah:

| Status | Pernyataan |
|---|---|
| **Pasti** | `Database/emailkomite.csv` tidak memuat satu baris pun `SIMASNET`/`SIMASNETA` |
| **Pasti** | `EmailKomiteBerjenjangSimasnet_sql` membaca `POOLDATA.EMAILKOMITE` dengan `TYPE_BUSINESS` = `"SIMASNET"` atau `"SIMASNETA"` |
| **Belum pasti** | di basis data mana baris itu berada — atau apakah ia ada sama sekali |

**Permintaan yang tidak bergantung pada jawaban itu.** Alih-alih menebak databasenya,
kuerinya dijalankan di **setiap basis data yang melayani portal**, tanpa `WHERE`:

```sql
-- Jalankan di SETIAP basis data portal, lalu sebutkan basis data mana yang menjawab apa.
SELECT ID, NAME, OPERATOR_ID, TYPE_BUSINESS, TYPE_KOMITE,
       LIMIT_BOTTOM, LIMIT_TOP, DEGREE,
       STS_AKTIF, STS_ADJ, STS_REG, STS_REJECT, STS_ABS
  FROM POOLDATA.EMAILKOMITE
 ORDER BY TYPE_BUSINESS, TYPE_KOMITE, LIMIT_BOTTOM, DEGREE, ID;
```

Tanpa `WHERE` supaya ketidaklengkapan yang sama tidak terulang, dan dijalankan di
tiap basis data supaya pertanyaan "di mana barisnya" terjawab oleh hasilnya sendiri alih-alih
oleh dugaan. Kolom EMAIL dan CC sengaja tidak diminta (`D-67`).

**Satu kemungkinan yang tidak boleh diabaikan:** bila ternyata tidak ada basis data mana pun
yang memuat baris `SIMASNET`, maka jalur Simasnet mengembalikan **nol penyetuju** hari ini —
dan itu keadaan produksi yang berjalan sekarang, bukan soal migrasi.

Catatan pada `komite/mode_test.go` dan `komite/http/rute_test.go` sudah disesuaikan supaya
tidak lagi menyatakan sesuatu yang belum pasti.

### 21.5 Jawaban final: tiap server punya POOLDATA sendiri

Work Owner menegaskan 2026-09-19: **aplikasi dijalankan di POOLDATA server masing-masing,
sesuai server yang menjalankannya.**

Itu menutup pertanyaan §21.4, dan membalikkan kesimpulan §21.1 kembali ke dugaan semula —
kali ini dengan dasar, bukan dugaan:

| Pernyataan | Status |
|---|---|
| `Database/emailkomite.csv` **lengkap** untuk basis datanya | ya — ia POOLDATA server **ASM** |
| Baris `SIMASNET`/`SIMASNETA` ada di POOLDATA server **Simasnet** | ya |
| Ekspor yang kita terima "terpotong" | **tidak** — dugaan §21.1 salah, dan ditarik |

Ini juga menjelaskan sesuatu yang tadinya janggal: kenapa kueri Simasnet tetap menyaring
`TYPE_BUSINESS` padahal basis datanya sudah khusus entitas itu. Jawabannya, di dalam basis
data Simasnet pun masih ada dua varian yang dibedakan — `SIMASNET` dan `SIMASNETA`.

#### Akibat yang paling penting: satu cacat yang menunggu di wiring

`cmd/claimpnc/main.go` memasang repo master ambang pada **koneksi utama**:

```go
simpan.komite = komitesql.RepoBaru(utama)
```

Selama isi `EMAILKOMITE` dianggap sama di semua basis data, itu tidak apa-apa. Setelah
jawaban ini, ia menjadi **cacat yang menunggu portal kedua dilayani**: portal mana pun yang
dipilih pengguna akan membaca tangga ambang milik portal UTAMA.

Yang membuatnya berbahaya adalah bentuk kegagalannya — **bukan galat, melainkan angka yang
salah tanpa satu pun tanda**. Layar menampilkan jenjang persetujuan entitas lain, seluruh
namanya masuk akal, dan tidak ada yang terlihat keliru. Itu kelas kegagalan yang sama
dengan `R-20`.

Hari ini belum merugikan karena hanya portal utama yang dilayani. Perbaikannya adalah
`TKT-F6-002` — koneksi diambil dari portal **aktif**, yang menuntut portal melekat pada
permintaan alih-alih pada keadaan global. Catatannya dipasang di tiga tempat supaya tidak
hilang: `cmd/claimpnc/main.go`, `komite/repo/sqlstore/ambang.sql`, dan tiket `TKT-F6-002`.

#### Yang diminta supaya mode satu-penyetuju dapat diuji

Kueri yang sama, dijalankan pada **basis data server Simasnet**:

```sql
SELECT ID, NAME, OPERATOR_ID, TYPE_BUSINESS, TYPE_KOMITE,
       LIMIT_BOTTOM, LIMIT_TOP, DEGREE,
       STS_AKTIF, STS_ADJ, STS_REG, STS_REJECT, STS_ABS
  FROM POOLDATA.EMAILKOMITE
 ORDER BY TYPE_BUSINESS, TYPE_KOMITE, LIMIT_BOTTOM, DEGREE, ID;
```

Tanpa `WHERE`, dan disebutkan berasal dari server mana. Kolom EMAIL dan CC sengaja tidak
diminta (`D-67`).

Bila kelak portal lain ikut dilayani, ekspor yang sama dibutuhkan dari tiap servernya —
karena tangga ambangnya bisa berbeda, dan sekarang kita tahu ia memang berbeda.

### 21.6 Rancangan portal dikonfirmasi, dan jaraknya ke selesai ternyata satu mata rantai

Work Owner menegaskan 2026-09-19: **setiap portal terkoneksi ke database masing-masing**, dan
**pool dijalankan sesuai portal**. Keduanya mengonfirmasi `D-75` dan ruang lingkup
`TKT-F6-002` apa adanya.

Karena itu kode yang ada diperiksa, untuk mengetahui berapa jauh jaraknya ke keadaan yang
diminta. Hasilnya: **lebih dekat daripada dugaan**.

| Bagian | Status | Letak |
|---|---|---|
| Pool per portal | **sudah ada** | `internal/platform/db` — `Untuk(alias)`, `Tersedia()`, `Utama()` |
| Portal terpilih, bertahan melewati muat ulang | **sudah ada** | `frontend/src/app/portal.ts` |
| Daftar portal + kesiapannya | **sudah ada** | `internal/portal`, `GET /api/portal` |
| Portal ikut pada tiap permintaan | **belum** | `api/klien.ts` tidak pernah mengirimnya |
| Server memilih pool dari portal aktif | **belum** | seluruh repo memakai `Utama()` |

Yang hilang tepat satu: portal tidak pernah menyeberang dari peramban ke server.

#### Kenapa mata rantai itu TIDAK saya pasang sekarang

Dua sebab, dan keduanya menahan.

**Pertama, di situ persis letak `R-20`.** Memilih pool dari alias yang dikirim klien tanpa
memeriksa kewenangan pengguna atas portal itu membuat siapa pun dapat membaca data badan
hukum lain dengan mengganti satu header. Itu **lebih buruk daripada keadaan sekarang** —
hari ini semua orang membaca portal utama, yang salah tetapi tidak dapat dieksploitasi.

Pemeriksaan kewenangannya sendiri belum dapat ditulis: `D-78` menetapkan login sama untuk
keempat entitas, sehingga data "siapa berhak atas portal mana" bersifat lintas portal dan
tidak dapat tinggal di dalam database tiap portal — memeriksa hak atas portal B menuntut
membaca database B **sebelum** penggunanya terbukti berhak. Itu pertanyaan terbuka, bukan
detail implementasi.

**Kedua, ia menyentuh modul yang dipagari.** Mengirim portal pada tiap permintaan menyentuh
`api/klien.ts` dan middleware sesi — keduanya dipakai Login dan Master Data, yang instruksi
kerja tetapkan **tidak diubah**.

Urutan yang benar karena itu: `D-78` dijawab → `TKT-F6-003` (kewenangan dinilai ulang di
server pada setiap perpindahan) → baru pemilihan pool di `TKT-F6-002`.

Seluruh temuan ini dipasang di tiketnya, lengkap dengan peringatan bagi yang mengerjakannya,
supaya tidak ada yang memasang mata rantai itu sebagai "perbaikan cepat".

#### Satu jawaban yang dikosongkan

Butir pertama — **berapa portal, dan apakah keempat database sudah tersedia** — tidak dijawab
pada pertukaran ini. Ia tetap terbuka atas nama Work Owner dan Tim Infra, dan menentukan
jumlah pool serta apa yang dapat diuji.

### 21.7 "Pool sesuai database yang ada" — sudah terpasang, bukan pekerjaan baru

Work Owner menjawab butir yang tadinya kosong: **pool mengikuti database yang ada**.

Jawabannya bukan angka melainkan **aturan**, dan itu menutup pertanyaan "berapa portal"
secara lebih baik daripada menyebut angka — daftar portal menjadi data, bukan konstanta
(`D-15`).

Kode diperiksa sebelum diklaim, dan perilakunya **sudah persis demikian**:

| Perilaku | Bukti |
|---|---|
| Daftar portal dipindai dari variabel `POOLDATA_<ALIAS>_HOST` | tidak ada satu pun daftar portal di dalam kode |
| Portal yang konfigurasinya tidak lengkap dilewati | `parameterPortal` — `if !b.Lengkap() { continue }` |
| Portal yang gagal dibuka dilewati dan dicatat, **tidak menjatuhkan aplikasi** | `KumpulanBaru` — `catat(p.Alias, err); continue` |
| **Kecuali portal utama** — gagalnya fatal | `KumpulanBaru` — `if p.Alias == utama { return nil, … }` |
| Portal yang belum siap **terlihat pengguna**, bukan hanya di log | `Tersedia()` → `aliasSiap` → `GET /api/portal` → `Portal.siap`; layar menandainya, tidak menyembunyikannya |

Pengecualian portal utama pantas disebut: tanpa basis data utama, sesi dan pencarian
pra-login tidak dapat berjalan sama sekali, sehingga menyala setengah jalan justru
menyesatkan. Portal lain yang belum siap tidak punya sifat itu.

**Tidak ada kode yang diubah.** Yang diperbarui hanya `TKT-F6-002`: butir "berapa portal"
ditutup, keadaan kode dicatat supaya tidak dibangun ulang, dan barisnya dipindahkan ke
daftar "sudah tertutup".

Dengan itu, penghalang `TKT-F6-002` menyusut menjadi **satu**: `D-78` — di mana kewenangan
portal disimpan. Batas koneksi per portal tetap terbuka, tetapi mekanismenya sudah ada
(`POOLDATA_<ALIAS>_MAKS_KONEKSI`); yang belum ada hanya angkanya, dan itu menahan
penyetelan sebelum produksi, bukan pengerjaannya.

---

## 22. Penggantian nama Indonesia → Inggris pada lingkup Komite dan platform (2026-09-19)

> **Aturan yang dipakai bab ini SUDAH TIDAK BERLAKU.** `CLAUDE.md` kemudian dibangun ulang
> dengan `D-80` dan `D-81`, yang menuntut istilah domain ikut berbahasa Inggris. Bab ini
> dibiarkan apa adanya sebagai rekaman; yang berlaku adalah **§23**.

### 22.1 Yang diminta, dan bagaimana lingkupnya dipersempit

Work Owner meminta penamaan kode Claim PNC diubah dari bahasa Indonesia menjadi bahasa
Inggris — **folder, berkas, dan identifier di dalamnya** — dengan pengecualian mengikuti
catatan pada `CLAUDE.md`.

Tiga hal ditanyakan lebih dulu karena jawabannya mengubah pekerjaan secara mendasar, dan
inilah jawabannya:

| Pertanyaan | Jawaban Work Owner |
|---|---|
| Aturan penamaan mana yang dipakai | **Sesuai dokumentasi di `CLAUDE.md`** |
| Berapa luas sekali jalan | **Komite dan platform dulu** |
| Apakah komentar ikut diterjemahkan | **Tidak — identifier saja** |

Jawaban kedua dan ketiga itu yang membuat pekerjaan ini dapat dikerjakan tanpa merusak:
menyentuh seluruh modul sekaligus akan mengubah `auth`, `masterstatus`, dan `portal` yang
`CLAUDE.md` nyatakan **selesai dan dilarang di-refactor** (Isolasi Protektif), dan
menerjemahkan komentar akan menghilangkan penjelasan bukti Pega yang justru menjadi isi
terpenting modul ini.

### 22.2 Aturan yang dipakai, dan dari mana ia datang

`CLAUDE.md` §7.4.1 sudah memuat aturannya, dan aturan itu dipakai apa adanya:

> istilah domain memakai **bahasa Indonesia** sesuai `CONTEXT.md`, karena itulah bahasa
> yang dipakai bisnis dan tim. Istilah teknis memakai **bahasa Inggris** mengikuti
> konvensi Go.

Alasannya disebut tegas di sana: mencampurkan istilah domain berbahasa Inggris yang salah
terjemah — `Adjustment` yang ternyata berarti nilai penyelesaian — adalah **tepat sumber
kekacauan yang sedang diperbaiki** migrasi ini.

Maka pembagiannya:

| Golongan | Perlakuan | Contoh |
|---|---|---|
| Istilah teknis | **Inggris** | `Layanan`→`Service` · `Opsi`→`Options` · `Tentukan`→`Determine` · `Temuan`→`Finding` · `Pengacak`→`Randomizer` |
| Istilah domain `CONTEXT.md` | **tetap Indonesia** | `Ambang` · `Jenjang` · `Penyetuju` · `Penjenjangan` · `Pita` · `Lini` · `Kebijakan` · `Komite` · `Kandidat` · `Penginput` |
| Gabungan yang inti katanya domain | **tetap Indonesia** | `JenjangPersetujuan` · `KebijakanBawaan` · `MasihDalamJenjang` · `TanpaPenyetuju` |

### 22.3 Yang berubah

**Paket platform** — keduanya hanya dipakai modul Komite dan `main.go`, sehingga
penggantiannya tidak menyentuh modul yang dilindungi:

| Sebelum | Sesudah |
|---|---|
| `internal/platform/uang` → `uang.Uang`, `DariRupiah`, `Urai`, `Nol`, `DariNilaiSQL`, `SatuanTerkecil`, `Desimal` | `internal/platform/money` → `money.Money`, `FromRupiah`, `Parse`, `Zero`, `FromSQLValue`, `MinorUnits`, `Decimals` |
| `internal/platform/acak` → `acak.Sistem{}.Pilih(banyak)` | `internal/platform/random` → `random.System{}.Pick(count)` |

**Berkas dan folder modul Komite:**

| Sebelum | Sesudah |
|---|---|
| `integritas.go` · `otomatis.go` | `integrity.go` · `auto.go` |
| `http/rute.go` · `http/galat.go` | `http/routes.go` · `http/errors.go` |
| `usecase/kelola.go` | `usecase/service.go` |
| `repo/sqlstore/kueri.go` | `repo/sqlstore/query.go` |
| `repo/memori/` → `memori.go`, `contoh.go` | `repo/memory/` → `memory.go`, `sample.go` |

Nama folder `komite/`, berkas `ambang.go`, `jenjang.go`, dan `ambang.sql`
**dipertahankan** — seluruhnya istilah domain, dan `ambang-komite` adalah nama modul yang
`CLAUDE.md` kecualikan.

**Frontend**, terbatas pada berkas milik Komite:

| Sebelum | Sesudah |
|---|---|
| `src/lib/uang.ts` · `uraiRupiah` | `src/lib/money.ts` · `parseRupiah` |
| `HalamanAmbangKomite.tsx` · `HalamanPenjenjanganKomite.tsx` | `AmbangKomitePage.tsx` · `PenjenjanganKomitePage.tsx` |
| `gunakanDaftarAmbangKomite` · `gunakanIntegritasAmbangKomite` · `gunakanPenjenjangan` | `useAmbangKomiteList` · `useAmbangKomiteIntegrity` · `usePenjenjangan` |
| `ResponsDaftarAmbangKomite` · `ResponsIntegritasAmbang` · `ResponsPenjenjangan` · `TemuanAmbang` | `AmbangKomiteListResponse` · `AmbangIntegrityResponse` · `PenjenjanganResponse` · `AmbangFinding` |

`formatRupiah` **tidak diubah**: Rupiah adalah nama mata uang, bukan istilah teknis.

Penggantian awalan `gunakan*` menjadi `use*` membawa keuntungan yang tidak diminta tetapi
nyata: aturan `react-hooks` hanya mengenali hook dari awalan `use`, sehingga hook yang
bernama `gunakan*` **tidak pernah diperiksa** aturan itu sama sekali.

### 22.4 Yang sengaja TIDAK disentuh

| Hal | Alasan |
|---|---|
| `auth`, `masterstatus`, `portal`, `platform/{config,db,httpserver,logging,waktu}` | Isolasi Protektif; dan seluruhnya dipakai bersama modul yang sudah selesai |
| `components/Ikon.tsx`, `api/klien.ts`, `app/sesi.ts` | dipakai bersama Login, Home, dan Master Data |
| Seluruh komentar | jawaban Work Owner: identifier saja |
| Nama fungsi uji (`TestJenjangKumulatif`, …) | `CLAUDE.md` §19.2 menetapkan nama uji **menyebutkan aturannya** supaya daftar uji terbaca sebagai dokumentasi aturan bisnis; menerjemahkannya justru melawan ketetapan itu |
| Nama field JSON (`jumlah_jenjang`, `berpita_nilai`, …) dan jalur rute (`/api/master/ambang-komite/integritas`) | **kontrak API**, bukan penamaan kode; mengubahnya adalah perubahan perilaku |
| Kode galat (`"validasi_gagal"`, …) | idem — yang diganti hanya nama konstantanya (`KodeValidasiGagal`→`CodeValidationFailed`), nilainya tetap |

### 22.5 Tiga kesalahan yang terjadi saat pengerjaan, dan polanya

Penggantian dikerjakan dengan `sed` dan sebuah perapi kolom, dan **keduanya salah pada
percobaan pertama**. Ketiganya dicatat karena polanya sama dan sudah berulang di proyek
ini: **perkakas dipercaya sebelum diuji pada kasus yang jelas.**

| # | Kesalahan | Bagaimana ketahuan |
|---|---|---|
| 1 | `s/\buang\./money./g` ikut mengenai kata Indonesia **"uang."** di dalam komentar — tujuh tempat menjadi "menyetujui money.", "master mata money." | pemindaian `money\.` yang tidak diikuti huruf |
| 2 | Perapi kolom meratakan **daftar parameter fungsi multi-baris**, padahal gofmt tidak | menjalankannya pada berkas yang sudah gofmt-bersih |
| 3 | Perapi kolom meratakan **pernyataan penugasan biasa** (`negatif = true`) dan menarik **elemen yang nilainya bersambung ke baris berikutnya** ke dalam kelompok, padahal gofmt memutus kelompok di situ | idem |

Yang menangkap ketiganya bukan pembacaan ulang yang lebih teliti, melainkan **satu uji
yang dirancang untuk gagal**: perapi dijalankan atas **seluruh 85 berkas** backend,
termasuk empat paket yang sudah gofmt-bersih dan tidak disentuh sama sekali. Setiap berkas
di luar lingkup yang ikut berubah adalah bukti perapinya salah.

Hasil akhirnya: **nol berkas di luar lingkup tersentuh**. Itu bukan jaminan hasilnya sama
dengan gofmt, tetapi ia bukti terkuat yang tersedia tanpa Go terpasang — dan jauh lebih
kuat daripada memeriksanya dengan mata.

Satu kesalahan lagi tertangkap dengan sendirinya: `random.System` sudah bernama `Pick`,
sementara antarmuka `komite.Randomizer` masih menuntut `Pilih`. Keduanya berhenti cocok,
dan itu **putus saat kompilasi** — bukan cacat senyap.

### 22.6 Keadaan verifikasi — belum ada satu pun yang dijalankan

Ini harus dinyatakan tegas, karena tanpanya mudah terbaca seolah pekerjaannya sudah
terbukti:

> **Tidak satu baris pun kode dalam sesi ini pernah dikompilasi atau diuji.** Go maupun
> Node.js tidak terpasang di mesin ini — `command -v go` dan `gofmt` sama-sama kosong.

Yang **sudah** dikerjakan sebagai penggantinya hanyalah pemeriksaan tekstual: pemindaian
identifier lama yang tersisa (nol), pemeriksaan rujukan lintas paket, pemeriksaan
kecocokan nama metode antar-seam, dan uji perapi terhadap berkas gofmt-bersih.

Yang **wajib dijalankan** sebelum pekerjaan ini boleh disebut selesai:

```
cd backend  && gofmt -l ./... && go vet ./... && go test ./...
cd frontend && npm run periksa-tipe && npm test && npm run build
```

`gofmt -l` yang mengeluarkan nama berkas berarti perapi kolom di §22.5 masih berbeda dari
gofmt, dan hasilnya tinggal ditimpa `gofmt -w`.

### 22.7 Sisa yang belum dikerjakan

Penggantian ini **baru separuh jalan secara sengaja** — "Komite dan platform dulu". Yang
belum: `auth`, `masterstatus`, `portal`, sisa `platform`, dan komponen bersama frontend.

Akibatnya tata letak modul Komite berbeda dari ketiga modul lain untuk sementara:
`repo/memory` di Komite berdampingan dengan `repo/memori` di modul lain, dan
`AmbangKomitePage.tsx` berdampingan dengan `HalamanMasuk.tsx`.

Perbedaan itu disengaja dan akan hilang saat modul berikutnya menyusul. Ia dicatat di sini
supaya tidak terbaca sebagai kelalaian oleh siapa pun yang membuka repo di antara kedua
tahap.

---

## 23. Penggantian nama diselesaikan mengikuti `D-80` dan `D-81` (2026-09-19)

> **§22 mencatat aturan yang sudah tidak berlaku.** Ia tetap dibiarkan apa adanya sebagai
> rekaman apa yang dikerjakan saat itu; yang berlaku sekarang adalah bab ini.

### 23.1 Aturannya berubah di tengah jalan, dan perubahannya mendasar

`CLAUDE.md` dibangun ulang dan kini memuat **`D-80`** dan **`D-81`** — keduanya menggantikan
§7.4.1 yang saya pakai pada §22.

| Hal | Aturan yang saya pakai di §22 | **`D-80`/`D-81` yang berlaku** |
|---|---|---|
| Istilah domain (`Ambang`, `Jenjang`, `Penyetuju`, `Lini`, `Pita`) | tetap Indonesia | **Inggris**, memakai padanan `CONTEXT.md` |
| Parameter dan variabel lokal | dibiarkan Indonesia | **Inggris** |
| Nama folder modul | — | **tetap Indonesia** — satu-satunya pengecualian (`D-81`) |

Akibatnya penggantian pada §22 **baru separuh benar**: paket, berkas, dan identifier teknisnya
sudah tepat, tetapi seluruh istilah domainnya masih Indonesia. Bab ini menyelesaikannya.

Sasaran `D-19` tetap dipegang dan itu penting: yang dipakai adalah **padanan Inggris yang
benar**, bukan alias Pega. `Adjustment` tetap tidak dipakai; `Object` tetap tidak dipakai.

### 23.2 Peta istilah yang dipakai

| Domain (`CONTEXT.md`) | Inggris | Contoh |
|---|---|---|
| Ambang | `Threshold` | `komite.Threshold`, `ListThresholds` |
| Jenjang | `Tier` | `Tier`, `TierCount`, `IsApprovalTier` |
| Penjenjangan | `Tiering` | `Tiering`, `TieringWith`, `TieringResponse` |
| Penyetuju | `Approver` | `Approver`, `Approvers`, `NoApprovers` |
| Pita | `Band` | `BandPolicy`, `BandLower`, `UsesBand` |
| Lini bisnis | `BusinessLine` | `BusinessLine`, `ListBusinessLines` |
| Kebijakan | `Policy` | `Policy`, `DefaultPolicy`, `SimasnetPolicy` |
| Penginput | `Applicant` | `Applicant`, `ExcludedApplicant`, `excludeApplicant` |
| Kandidat · Tersingkir | `Candidate` · `Excluded` | `Candidates`, `Excluded` |
| Batas bawah · atas | `LowerBound` · `UpperBound` | — |
| Komite tertunda | `PendingCommittee` | `CurrentTier`, `WaitingSince` |

**Berkas yang ikut berganti nama:** `ambang.go` → `threshold.go` · `jenjang.go` → `tiering.go`
(beserta ujinya) · `repo/sqlstore/ambang.{go,sql}` → `threshold.{go,sql}` ·
`AmbangKomitePage.tsx` → `ThresholdPage.tsx` · `PenjenjanganKomitePage.tsx` → `TieringPage.tsx` ·
`AmbangKomite.test.tsx` → `pages.test.tsx`.

**Yang TETAP:** folder `internal/komite/` dan `src/modules/ambang-komite/` — keduanya nama modul
(`D-81`) · seluruh komentar · seluruh nama field JSON (`nama`, `lini`, `jenjang`,
`berpita_nilai`, …) · seluruh nilai kode galat (`"validasi_gagal"`, …) · seluruh jalur rute ·
seluruh teks yang dilihat pengguna · nama prop komponen bersama (`kolom`, `baris`, `judul`, …),
karena komponen itu milik modul lain yang belum diganti namanya.

### 23.3 Tiga kesalahan pada pass ini — polanya sama dan sudah berulang

| # | Kesalahan | Bagaimana ketahuan | Akibat |
|---|---|---|---|
| 1 | Sed untuk variabel lokal di berkas uji **mengganti deklarasi tetapi tidak seluruh pemakaiannya** — `temuan :=` menjadi `findings :=` sementara `saring(temuan, …)` di bawahnya tidak ikut | pemindaian sesudahnya menemukan `temuan` yang sudah tidak punya deklarasi | **dibatalkan seluruhnya**; berkas uji dipulihkan dari salinan sebelum sed |
| 2 | Pemeriksa silang "rujukan versus deklarasi" memakai `^\t` di dalam ERE, padahal **POSIX ERE tidak mengenal `\t`** — ia mencocokkan huruf `t` | 18 konstanta yang jelas-jelas ada dilaporkan hilang | laporan palsu; setelah `[[:space:]]` dipakai, deklarasi terkumpul naik dari 55 menjadi **139** dan temuannya menjadi **nol** |
| 3 | `Aktif:` diganti `Active:` secara seragam, padahal pada `AutoPolicy` medannya bernama `Enabled` | pembacaan ulang literal `AutoPolicy{…}` di berkas uji | tujuh tempat diperbaiki |

Kesalahan kedua adalah **yang keempat kalinya** pola yang sama muncul dalam dua sesi terakhir:
alat ukur dipercaya sebelum diuji pada kasus yang jelas benar. Tiga yang sebelumnya tercatat di
§22.5.

**Dua kesalahan itu tertangkap oleh hal yang sama**: menjalankan pemeriksaan yang **dirancang
untuk gagal** — memulihkan dari salinan sebelum menyentuh apa pun, dan membandingkan hasil alat
terhadap kasus yang sudah pasti benar.

### 23.4 Yang TIDAK dikerjakan, dan alasannya

**Variabel lokal di dalam berkas uji Go tetap berbahasa Indonesia** — `hasil`, `temuan`,
`ambang`, `kebijakan`, dan sekitar 170 kemunculan lain.

Ini **melanggar `D-80`**, dan dicatat sebagai utang, bukan disamarkan. Alasannya dua, dan
keduanya berasal dari percobaan yang benar-benar dijalankan:

1. **Sed tidak dapat membedakan kode dari prosa Indonesia.** Kata-kata itu muncul juga di
   komentar **dan di dalam pesan assertion** — `require.Len(t, …, "kedua baris yang seri harus
   disebutkan")`. Pola berjangkar menyelesaikan sebagian, dan bagian yang tersisa justru yang
   memutus konsistensi deklarasi-pemakaian (§23.3 butir 1).
2. **Tidak ada kompiler untuk membuktikan hasilnya.** Menulis ulang 1.830 baris uji tanpa dapat
   menjalankan `go build` menukar cacat kosmetik dengan risiko cacat nyata.

**Nama fungsi uji tetap berbahasa Indonesia dengan sengaja** — dan ini bukan utang. `CLAUDE.md`
§19.2 menetapkan nama uji **menyebutkan aturannya** sehingga daftar uji terbaca sebagai
dokumentasi aturan bisnis; menerjemahkannya justru melawan ketetapan itu. Pesan assertion
mengikuti alasan yang sama.

**Modul lain belum disentuh** — `auth`, `masterstatus`, `portal`, sisa `platform`, dan komponen
bersama frontend masih memakai penamaan Indonesia sepenuhnya (`HandlerBaru`, `Opsi`, `Layanan`,
`TulisRespon`, `TabelData`, `KolomIsian`). Lingkupnya memang "Komite dan platform dulu".

### 23.5 Keadaan verifikasi — tidak berubah dari §22.6

> **Tidak satu baris pun pernah dikompilasi atau diuji.** Go maupun Node.js tidak terpasang.

Yang dikerjakan sebagai penggantinya pada pass ini:

| Pemeriksaan | Hasil |
|---|---|
| Rujukan berkualifikasi di berkas uji versus deklarasi paket | **nol** yang tidak ditemukan, dari 139 deklarasi |
| Ekspor frontend versus pemakaiannya di modul komite | **nol** yang tidak ditemukan |
| Pemindaian identifier lama di seluruh backend dan frontend | **nol** sisa |
| Prosa komentar yang rusak oleh sed | **nol** |
| Perapi kolom terhadap 85 berkas backend | hanya berkas dalam lingkup yang berubah |

Yang **wajib** dijalankan sebelum pekerjaan ini disebut selesai tidak berubah:

```
cd backend  && gofmt -l ./... && go vet ./... && go test ./...
cd frontend && npm run periksa-tipe && npm test && npm run build
```
