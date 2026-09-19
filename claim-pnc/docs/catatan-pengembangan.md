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

---

## 9. Sesi keempat — Master Status Klaim (2026-09-17)

Modul bisnis **pertama**. Sampai sesi ini yang ada hanyalah login, pemilih portal, dan beranda
sementara; `README.md` menyatakannya sendiri: *"Modul bisnis belum ada satu pun."*

### 9.1 Koreksi premis instruksi

Instruksi menyebut modul **Login, Home, dan Master Data** sudah selesai dan dilarang disentuh.
Pemeriksaan terhadap kode menemukan yang ketiga **tidak ada sama sekali** — tidak ada folder master
mana pun di `internal/`, tidak ada layar master di `modules/`, dan tidak ada tabel master di
`migrations/`.

Disampaikan lebih dulu sebelum mengerjakan. Akibatnya bukan sekadar soal penamaan: **Master Status
Klaim menjadi master data yang pertama**, sehingga setiap pilihan di sini menjadi pola untuk
sekurang-kurangnya 28 master berikutnya.

### 9.2 Yang dibaca sebelum menulis kode

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

### 9.3 Empat pertanyaan konfirmasi dan jawabannya

| # | Pertanyaan | Jawaban Work Owner |
|---|---|---|
| 1 | Menulis ke mana? | **Go jadi penulis tunggal `M_STS_CLAIM`**, tidak lagi menyimpan JSON; isi JSON dipindahkan ke kolom. Procedure `PEGA_M_STS_CLAIM` boleh ditinggalkan |
| 2 | Seberapa jauh lingkupnya? | Fungsi dan tampilan **seperti Pega**, tetapi **lebih bagus, mobile friendly, dan user friendly** |
| 3 | Jejak audit? | **Samakan dengan sekarang** — sistem lama tidak punya, jadi tidak perlu ditambahkan |
| 4 | Validasi? | **Tolak ID atau nama status ganda, dan tolak yang kosong** |

Jawaban 3 menyimpang dari acceptance criteria `TKT-F4-001` ("setiap perubahan master menghasilkan
tepat satu baris jejak audit"). Itu keputusan Work Owner, dicatat di `keputusan-implementasi.md`
§10 beserta konsekuensinya — bukan diserap diam-diam.

### 9.4 Yang dibangun

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

### 9.5 Kendala dan penyelesaiannya

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

### 9.6 Verifikasi yang benar-benar dijalankan

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

### 9.7 Yang ditemukan dari basis data yang berjalan

Mode periksa dijalankan terhadap Oracle, lalu satu perkakas diagnostik **baca-saja** sementara untuk
membaca katalog. Perkakasnya sudah dihapus; kuerinya dicatat di §9.8 supaya dapat diulang.

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

### 9.8 Kueri diagnostik, supaya dapat diulang tanpa perkakas

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

### 9.9 Yang belum dapat dibuktikan

| Acceptance criteria | Keadaan | Apa yang menahannya |
|---|---|---|
| Layar bekerja terhadap Oracle | **Belum.** `LSC_NOTE` masih kosong pada 32 baris, sehingga daftar akan tampil tanpa label | Migrasi `0002` belum dijalankan DBA |
| Indeks unik menolak label ganda di basis data | **Belum diuji.** Yang terbukti baru pemeriksaan di aplikasi | idem |
| Kode baru terbit dari `M_STS_CLAIM_SEQ` | **Belum diuji** terhadap Oracle | idem, dan menuntut hak `INSERT` yang belum tentu dimiliki akun aplikasi |
| Pega tetap membaca benar setelah view diganti | **Belum diuji** | `D-63` menuntut pengujian dengan menjalankan Pega dan Go bersamaan |

---

## 10. Sesi kelima — penataan ulang tampilan (2026-09-17)

Permintaan Work Owner: tampilan yang menarik secara visual, gaya modern, kontras baik, efek hover
dan active dengan transisi halus, responsif di ponsel dan desktop, bayangan lembut dan sudut
membulat, **Light Mode**.

### 10.1 Dua keputusan yang diminta lebih dulu

| Pertanyaan | Jawaban Work Owner |
|---|---|
| Layar mana yang ikut didesain ulang? | **Seluruh aplikasi** — termasuk Masuk dan Beranda |
| Warna aksennya apa? | **Biru profesional**, abu-abu batu sebagai dasar. Mula-mula indigo, diganti menjadi `blue` atas permintaan susulan — lihat §10.11 |

Pertanyaan pertama diajukan karena aturan proyek sampai sesi lalu **melarang menyentuh modul Login
dan Home**, sedangkan desain ulang visual pasti menyentuh keduanya. Jawabannya mencabut larangan
itu untuk urusan tampilan.

**Merah korporat sengaja ditawarkan dan sengaja tidak dipilih.** Merah adalah bahasa universal untuk
galat; tombol Simpan berwarna merah di sebelah pesan galat berwarna merah sulit dibedakan sekilas.
Bila kelak merek menuntutnya, warna galat harus digeser lebih dulu.

### 10.2 Sistem desain, bukan tempelan per layar

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
putih mencapai 5,1:1, lewat ambang AA. (Angka 8.6:1 yang sempat tertulis di sini salah — lihat §10.11.)

### 10.3 Tiga hal yang dikerjakan karena diminta, dan satu yang tidak diminta

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

### 10.4 Tiga hal yang dipindahkan, dan alasannya bukan estetika

| Yang pindah | Dari | Ke | Kenapa |
|---|---|---|---|
| Tombol **Keluar** | halaman Beranda | bilah atas | Pengguna yang sedang membuka layar master **tidak punya cara keluar** tanpa kembali ke beranda dulu |
| **Pemilih portal** | halaman Beranda | bilah atas | Sama: portal menentukan basis data seluruh layar, bukan hanya beranda |
| **Nama pengguna** | kartu identitas Beranda | bilah atas | Berlaku di seluruh layar; menyisakannya di dua tempat membuat nama yang sama muncul dua kali |

Baris "Nama" pada kartu identitas Beranda **dihapus** sebagai akibatnya. Itu bukan sekadar
kerapian — uji beranda mencari nama pengguna dengan **pencocokan persis**, dan dua elemen berisi
nama yang sama membuat pencarian itu gagal. Pemindahan dan penghapusan harus dilakukan bersamaan.

### 10.5 Kendala: uji layar master ikut rusak, dan itu benar

Setelah pemilih portal pindah ke bilah atas, **14 dari 15 uji layar master gagal**. Sebabnya bukan
tampilan: bilah atas kini memanggil `/api/portal` pada **setiap** layar di balik sesi, sedangkan
peladen tiruan di uji master hanya menjawab daftar status. Jawaban yang salah bentuk membuat
`data.portal.map` melempar.

Yang diperbaiki adalah **fixture-nya**, bukan komponennya: peladen tiruan menjawab `/api/portal`
otomatis, pola yang sama dengan `HalamanMasuk.test.tsx` yang sudah melakukannya sejak awal.

Melunakkan `PemilihPortal` supaya tahan jawaban yang salah bentuk sempat dipertimbangkan dan
**ditolak**: kontrak API menjamin bentuknya, dan komponen yang diam saat menerima bentuk salah
menyembunyikan cacat yang seharusnya terlihat.

### 10.6 Kesalahan saya sendiri yang perlu dicatat

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

### 10.7 Keputusan yang sengaja tidak diambil

| Yang tidak dipakai | Kenapa |
|---|---|
| **Google Fonts** | Aplikasi berjalan di VM on-premise tanpa jaminan akses internet. Huruf yang gagal dimuat mengubah seluruh tata letak. Dipakai tumpukan font sistem |
| **Pustaka ikon** | Delapan bentuk yang seluruhnya beberapa baris `path` tidak sebanding dengan satu dependensi yang harus dipelajari, dipantau keamanannya, dan ikut membesarkan bundel. Ikon digambar langsung sebagai SVG di `components/Ikon.tsx` |
| **Pustaka tabel** (TanStack / AG Grid) | `TKT-U2-005` menuntut keputusannya diambil dengan pengukuran. Belum berubah sejak sesi lalu |
| **Mode gelap** | Work Owner meminta Light Mode saja. `color-scheme: light` ditegaskan supaya kontrol bawaan peramban tidak ikut membalik mengikuti tema sistem |
| **Menu hamburger** | Dengan dua entri, hamburger menambah satu ketukan untuk menyembunyikan sesuatu yang sebenarnya muat. Menu digulir menyamping. Perlu ditinjau ulang bila menunya kelak berasal dari izin peran dan bertambah banyak |

### 10.8 Aksesibilitas — yang dikerjakan supaya kontras tidak berhenti di warna

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

### 10.9 Verifikasi yang benar-benar dijalankan

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

### 10.10 Yang perlu diketahui saat mencoba

SPA **tersemat ke binary** lewat `go:embed` (`ADR-0002`). Proses yang sedang berjalan memuat
tampilan **lama** sampai dibangun ulang:

```bash
cd frontend && npm run build     # hasilnya ke backend/spa/dist
cd ../backend && go run ./cmd/claimpnc
```

Selama pengembangan antarmuka, `npm run dev` di port 5173 jauh lebih cepat — perubahan langsung
terlihat tanpa membangun ulang binary.

### 10.11 Koreksi warna aksen: indigo menjadi biru (2026-09-17, masih hari yang sama)

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

## 11. Sesi keenam — Penjenjangan Komite dan Master Ambang (2026-09-17)

Tugasnya: **melanjutkan ke modul Flow Komite**, dengan `Flow/Komite_Flow.xml` sebagai
rujukan.

### 11.1 Tiga temuan yang mengoreksi premis tugas

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

### 11.2 Empat pertanyaan konfirmasi dan jawabannya

| # | Pertanyaan | Jawaban Work Owner |
|---|---|---|
| 1 | Cakupan sesi ini, mengingat `B-5` dan `B-6` belum ada | **Master ambang + mesin penjenjangan.** Layar keputusan menyusul |
| 2 | Tabel warisan mana yang boleh ditulis (`P-1`) | **Baca saja dulu.** Pega tetap penulis tunggal |
| 3 | `AutoAcceptKomite` — dibawa? | **Jangan dibawa** |
| 4 | Go dan Node tidak terpasang | **Tulis kode, uji menyusul** |

Jawaban 2 menutup kemungkinan mengelola master dari aplikasi ini, dan itu yang membuat
seluruh kueri modul ini **tidak memuat satu pun pernyataan tulis** — ditegakkan oleh uji,
bukan hanya oleh niat.

### 11.3 Verifikasi mandiri terhadap ketujuh kasus spec

Sebelum menulis kode, ketujuh kasus jumlah penyetuju pada
`docs/ticketing/B-7-Komite-Persetujuan-Klaim/spec.md` dihitung ulang dengan tangan dari
`Database/emailkomite.csv`. **Ketujuhnya cocok**, dan itu yang menjadi dasar mempercayai
aturan kumulatifnya sebelum satu baris kode ditulis.

### 11.4 Yang dibangun

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

### 11.5 Empat keputusan rancangan yang perlu dijelaskan

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

### 11.6 Dua pernyataan dokumen yang dikoreksi oleh bukti

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

### 11.7 Kendala dan penyelesaiannya

| Kendala | Penyelesaian |
|---|---|
| Go dan Node tidak terpasang; tidak ada `.env` | Kode dan uji ditulis lengkap, dan **dinyatakan terus terang belum dijalankan**. Lihat §11.8 |
| Tipe kolom `EMAILKOMITE` tidak diketahui — DDL belum pernah dilihat (`R-08`) | Seluruh kolom dipindai ke `any` lalu ditafsirkan eksplisit, sehingga modul tahan terhadap kolom bertipe angka maupun teks. Penafsirannya diuji tersendiri |
| `OPERATOR_ID` baris ID 4 di CSV diakhiri **baris baru** | `Ambang.Bersih` merapikan seluruh field teks; diuji di `TestSpasiTepiDibuangDariDataMaster` |
| Pengelompokan integritas PA sempat salah | Bila dikelompokkan per `TYPE_KOMITE`, tangga PA terbelah dan tampak berlubang parah — padahal di PA kolom itu membedakan PA reguler dari PA TKI (`D-70`). Pengelompokan dibuat **per lini** untuk lini tanpa pita, **per pita** untuk Non-MBU |
| Atap pita bawah Non-MBU sempat dilaporkan sebagai peringatan | Itu peringatan palsu: berhenti tepat di batas pita justru benar, karena di atasnya klaim berpindah pita. Dikecualikan, dan pengecualiannya diuji |

### 11.8 Verifikasi — apa yang BELUM dijalankan

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

### 11.9 Yang tidak dibangun, dan kenapa

| Tidak dibangun | Sebab |
|---|---|
| Layar keputusan komite — setuju, tolak, kembalikan | `TKT-B07-002`; bergantung pada `B-5` dan `B-6` yang belum ada. Membangunnya sekarang menuntut mengarang entitas Klaim |
| Pencatatan jejak keputusan komite | idem, ditambah `S-5` yang belum ada |
| `AutoAcceptKomite` | Work Owner memutuskan **tidak dibawa** |
| Pengelolaan master ambang | Work Owner memutuskan **baca saja**; memindahkan kepemilikan menuntut prosedur `D-63` |
| Pemeriksaan peran pada rute | `TKT-F3-005`; keadaannya sama dengan seluruh rute lain hari ini |

### 11.10 Yang perlu dijawab Work Owner

1. **Beban bila `AutoAcceptKomite` dihapus belum dihitung.** `TKT-B07-003` menuntut
   angkanya dilaporkan sebelum rilis. Kuerinya menunggu akses basis data:

       SELECT COUNT(*) AS jumlah_auto_accept
         FROM POOLDATA.T_CLAIM_KOMITE_LIST
        WHERE NOTEKOMITE LIKE 'Auto Accept by PEGA%'
          AND DATEOFCOMMITE_CREATE >= SYSDATE - 90;

2. **Arti `DEGREE=0`** — pengamatan pada §11.6 menunggu ditegaskan atau dikoreksi.
3. **Apakah `dbms_random.value` pada dua kueri Simasnet disengaja.** Bila ya, jalur itu
   tidak dapat diuji secara deterministik, dan aturannya di sini harus ditinjau ulang.
4. **Batas pita Rp 100.000.000** masih berupa angka di dalam kode — satu-satunya nilai
   bisnis yang demikian di modul ini. Ia sudah dapat dipasok dari luar; yang belum ada
   adalah tabel masternya.

---

## 12. Sesi keenam, lanjutan — empat jawaban Work Owner (2026-09-18)

Keempat pertanyaan terbuka pada §11.10 dijawab, dan **dua di antaranya membatalkan kode
yang sudah ditulis sehari sebelumnya**. Bagian ini mencatat apa yang berubah dan atas
dasar bukti apa.

### 12.1 Jawaban dan akibatnya

| # | Pertanyaan | Jawaban | Akibat |
|---|---|---|---|
| 1 | `AutoAcceptKomite` | **Jangan dihapus, tapi dikonversi** | Membatalkan keputusan 2026-09-17 "tidak dibawa". Aturannya dibangun |
| 2 | Arti `DEGREE=0` | **Tidak dipakai** | Penyaringnya ditulis eksplisit |
| 3 | `dbms_random` disengaja? | **Ya** — mengacak operator, dan **penginput dikecualikan** | Membatalkan larangan yang saya pasang. Mode kedua dibangun |
| 4 | Batas pita Rp 100 juta dari mana? | dijawab dengan bukti — lihat §12.5 | Ditemukan **kembaran USD 7.000** yang belum pernah tercatat |

### 12.2 Jawaban 3 mengoreksi kesalahan saya, dan ini yang paling serius

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

### 12.3 Siapa yang memilih mode, dan kenapa itu urusan portal

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

### 12.4 Yang dibangun untuk jawaban 3

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

### 12.5 Jawaban 4 — dari mana angka Rp 100.000.000

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

### 12.6 Jawaban 2 — `DEGREE=0` disaring eksplisit

`JenjangPersetujuan()` kini menuntut `Jenjang > 0` di samping `STS_AKTIF` dan `STS_ADJ`.

Hari ini penyaring itu **tidak mengubah satu hasil pun**: baris ber-DEGREE 0 yang masih
aktif hanya ID 9, dan ia sudah tersaring lebih dulu karena `STS_ADJ`-nya kosong.

Menuliskannya tetap perlu, karena tanpa itu aturannya hanya **berlaku secara kebetulan**.
Satu baris baru ber-DEGREE 0 dengan `STS_ADJ` menyala akan diam-diam ikut menyetujui uang,
dan tidak ada yang akan menyadarinya. Diuji dengan baris yang seluruh syarat lainnya
terpenuhi.

### 12.7 Jawaban 1 — `AutoAcceptKomite` dikonversi, bukan dihapus

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

### 12.8 Verifikasi

**Masih belum dijalankan.** Go dan Node tetap tidak terpasang di mesin ini. Perintah yang
harus dijalankan sama dengan §11.8, ditambah berkas uji baru:

| Berkas | Yang dibuktikan |
|---|---|
| `komite/mode_test.go` | mode satu-penyetuju memilih tepat satu; penginput dikecualikan dan pencocokannya tahan penulisan; jenjang terendah yang dipilih; pengacak di balik seam; `DEGREE=0` tidak pernah menyetujui; batas pita dapat berbeda antar entitas |
| `komite/otomatis_test.go` | fitur mati secara bawaan; ambang diuji tepat di batas; batas nilai dan jenjang menolak yang melampaui; yang tidak layak dikembalikan beserta alasannya |
| `komite/http/rute_test.go` | mode dikirim pada kedua respons; kandidat dilaporkan; penginput diabaikan pada mode kumulatif |
| `modules/ambang-komite/AmbangKomite.test.tsx` | isian Operator ID hanya muncul pada mode yang memakainya; calon dan pengecualian tampil |

### 12.9 Yang masih menunggu jawaban

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

## 13. Empat jawaban penutup, dan satu akibat yang harus diketahui sebelum rilis (2026-09-18)

### 13.1 Jawaban

| # | Pertanyaan | Jawaban | Akibat pada kode |
|---|---|---|---|
| 1 | `> 2` itu 48 atau 72 jam? | **72 jam** | Ambiguitas tertutup; bawaan 3 hari dipastikan benar |
| 2 | Nilai dan jenjang mana yang boleh auto-accept? | **Tidak dibatasi** — selama `KomiteCount <= KomiteLoop` | Dua medan konfigurasi **dihapus**, syarat jenjang ditambahkan |
| 3 | Batas pita portal SMI | *(belum dijawab)* | tetap terbuka |
| 4 | Pengecualian penginput berlaku di entitas lain? | **Ya** | Pengecualian menjadi **berlaku di semua mode** |

### 13.2 Jawaban 2 menutup pertanyaan, dan karena itu dua knob dihapus

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

### 13.3 Jawaban 4 adalah perubahan perilaku, dan akibatnya sudah dihitung

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

### 13.4 Yang masih terbuka

1. **Batas pita portal SMI** — USD 7.000 berasal dari kurs beku 14.285 yang sudah tidak
   berlaku. Belum dijawab.
2. **Master ambang portal Simasnet** belum pernah dilihat; mode satu-penyetuju masih diuji
   dengan data buatan.
3. **Seberapa sering anggota komite menginput klaim sendiri** — menentukan apakah ketujuh
   keadaan di §13.3 adalah risiko nyata atau kemungkinan teoretis. Pertanyaan untuk
   pengguna bisnis, bukan untuk kode.
4. **Menyambungkan kebijakan ke portal aktif** — `TKT-F6-002`.

---

## 14. Penelusuran tiga pertanyaan Work Owner (2026-09-19)

### 14.1 Batas USD 7.000 adalah PENIMPA, bukan cabang yang setara

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

### 14.2 Master Simasnet: tabel yang SAMA, disaring TYPE_BUSINESS

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

### 14.3 Kenapa Simasnet mengecualikan pengaju — dan kenapa alasannya tidak berlaku di lini lain

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
menghasilkan tujuh keadaan tanpa penyetuju yang tercatat di §13.3 — bukan karena
aturannya salah diterapkan, melainkan karena di luar Simasnet tidak ada "rekan lain" yang
tersisa.

Pertanyaan itu diajukan ulang ke Work Owner dengan konsekuensinya, karena penjelasan pada
§14.3 adalah keterangan BARU yang belum tersedia saat keputusan sebelumnya diambil.

### 14.4 Keputusan pengecualian pengaju ditegaskan ulang

Pertanyaan §14.3 diajukan kembali kepada Work Owner pada 2026-09-19, lengkap dengan
ketujuh keadaan tanpa penyetuju yang sudah dihitung dan tiga jalan keluar: melengkapi
master pada titik-titik tipisnya, membatasi pengecualian kembali ke Simasnet saja, atau
menaikkan klaim ke jenjang berikutnya bila penyetujunya habis.

**Jawabannya: tetap seperti keputusan 2026-09-18.** Pengecualian berlaku di seluruh
entitas apa adanya; klaim yang diajukan oleh satu-satunya penyetujunya akan berhenti dan
ditangani manual.

Tidak ada perubahan kode — yang terpasang sudah persis demikian. Yang ditambahkan hanya
catatan pada `keluarkanPenginput`, supaya pembaca berikutnya tahu perilaku itu **keputusan
yang diambil setelah akibatnya diketahui**, bukan kelalaian yang menunggu diperbaiki.

### 14.5 Ringkasan pertanyaan terbuka setelah sesi ini

| # | Terbuka | Pemilik |
|---|---|---|
| 1 | Isi master ambang portal **Simasnet** belum pernah dilihat — kueri pengambilannya ada di §14.2 | DBA / Tim Simasnet |
| 2 | Batas pita portal **SMI** — USD 7.000 berasal dari kurs beku 14.285 yang sudah tidak berlaku | Work Owner |
| 3 | Menyambungkan kebijakan ke **portal aktif** (`TKT-F6-002`) — hari ini seluruh portal memakai kebijakan rupiah kumulatif | Lead Engineer |
| 4 | Penulisan persetujuan otomatis dan penjadwalnya — menunggu `B-5`, `B-6`, dan `S-6` | — |

---

## 15. Dua koreksi dari Work Owner (2026-09-19)

### 15.1 Simasnet berjalan di basis data ASM — dugaan saya keliru

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

### 15.2 Berkasnya 29 record, bukan 30 — dan dua "lini" palsu yang sempat saya laporkan

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

### 15.3 Ekspresi pita memang penentu TYPE_KOMITE — dikonfirmasi

Work Owner menegaskan bahwa `@If(ConvertAdjustmentValue > 100000000, 2, 1)` dan kembaran
USD-nya memang **untuk mengambil type komite**-nya.

Itu sejalan dengan penelusuran §14.1: hasilnya disimpan ke `tempAdj.AcceptedNo`, dan
`AcceptedNo` dipakai apa adanya sebagai penyaring
`trim(TYPE_KOMITE) = trim({tempAdj.AcceptedNo})` pada `EmailKomiteBerjenjang_sql`.

Dengan itu, pemetaan pita di aplikasi ini terkonfirmasi benar: `KebijakanPita.Pilih`
mengembalikan `"1"` atau `"2"`, dan nilai itu dicocokkan ke kolom `TYPE_KOMITE` —
persis seperti yang dikerjakan rule lama.

### 15.4 Ralat: `LSC_ID` hanya pengecekan server, bukan keterangan letak data

Work Owner meralat jawaban pada §15.1: **pencocokan `DB_LINK_PEGA` itu hanya pengecekan
server mana yang menjalankan aplikasi** — ia tidak menyatakan apa pun tentang di basis data
mana barisnya tersimpan.

**Yang menjadi lebih kuat.** Ralat ini justru menegaskan penelusuran §14.1:

```sql
-- RDB List/BrowseAPPName_sql-SQL.xml, dijalankan GetLinkAppClaim
SELECT APP AS "DOCUMENTNAME" FROM pooldata.DB_LINK_PEGA
 WHERE APPIP LIKE '%<nama server>%'
```

adalah **deteksi lingkungan**, bukan konfigurasi. Dan karena `GetLinkAppClaim` jatuh ke
nilai bawaan `"ASM"` saat pencocokan gagal, server yang berganti nama akan diam-diam
memakai perilaku entitas yang salah — termasuk ambang pita rupiah untuk klaim dolar.
Itulah yang `D-75` hapus dengan mengganti deteksi nama server menjadi portal.

**Yang kembali terbuka.** Kesimpulan §15.1 — "ekspornya belum lengkap, bukan datanya ada di
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

### 15.5 Jawaban final: tiap server punya POOLDATA sendiri

Work Owner menegaskan 2026-09-19: **aplikasi dijalankan di POOLDATA server masing-masing,
sesuai server yang menjalankannya.**

Itu menutup pertanyaan §15.4, dan membalikkan kesimpulan §15.1 kembali ke dugaan semula —
kali ini dengan dasar, bukan dugaan:

| Pernyataan | Status |
|---|---|
| `Database/emailkomite.csv` **lengkap** untuk basis datanya | ya — ia POOLDATA server **ASM** |
| Baris `SIMASNET`/`SIMASNETA` ada di POOLDATA server **Simasnet** | ya |
| Ekspor yang kita terima "terpotong" | **tidak** — dugaan §15.1 salah, dan ditarik |

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

### 15.6 Rancangan portal dikonfirmasi, dan jaraknya ke selesai ternyata satu mata rantai

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

### 15.7 "Pool sesuai database yang ada" — sudah terpasang, bukan pekerjaan baru

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

## 16. Penggantian nama Indonesia → Inggris pada lingkup Komite dan platform (2026-09-19)

> **Aturan yang dipakai bab ini SUDAH TIDAK BERLAKU.** `CLAUDE.md` kemudian dibangun ulang
> dengan `D-80` dan `D-81`, yang menuntut istilah domain ikut berbahasa Inggris. Bab ini
> dibiarkan apa adanya sebagai rekaman; yang berlaku adalah **§17**.

### 16.1 Yang diminta, dan bagaimana lingkupnya dipersempit

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

### 16.2 Aturan yang dipakai, dan dari mana ia datang

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

### 16.3 Yang berubah

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

### 16.4 Yang sengaja TIDAK disentuh

| Hal | Alasan |
|---|---|
| `auth`, `masterstatus`, `portal`, `platform/{config,db,httpserver,logging,waktu}` | Isolasi Protektif; dan seluruhnya dipakai bersama modul yang sudah selesai |
| `components/Ikon.tsx`, `api/klien.ts`, `app/sesi.ts` | dipakai bersama Login, Home, dan Master Data |
| Seluruh komentar | jawaban Work Owner: identifier saja |
| Nama fungsi uji (`TestJenjangKumulatif`, …) | `CLAUDE.md` §13.2 menetapkan nama uji **menyebutkan aturannya** supaya daftar uji terbaca sebagai dokumentasi aturan bisnis; menerjemahkannya justru melawan ketetapan itu |
| Nama field JSON (`jumlah_jenjang`, `berpita_nilai`, …) dan jalur rute (`/api/master/ambang-komite/integritas`) | **kontrak API**, bukan penamaan kode; mengubahnya adalah perubahan perilaku |
| Kode galat (`"validasi_gagal"`, …) | idem — yang diganti hanya nama konstantanya (`KodeValidasiGagal`→`CodeValidationFailed`), nilainya tetap |

### 16.5 Tiga kesalahan yang terjadi saat pengerjaan, dan polanya

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

### 16.6 Keadaan verifikasi — belum ada satu pun yang dijalankan

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

`gofmt -l` yang mengeluarkan nama berkas berarti perapi kolom di §16.5 masih berbeda dari
gofmt, dan hasilnya tinggal ditimpa `gofmt -w`.

### 16.7 Sisa yang belum dikerjakan

Penggantian ini **baru separuh jalan secara sengaja** — "Komite dan platform dulu". Yang
belum: `auth`, `masterstatus`, `portal`, sisa `platform`, dan komponen bersama frontend.

Akibatnya tata letak modul Komite berbeda dari ketiga modul lain untuk sementara:
`repo/memory` di Komite berdampingan dengan `repo/memori` di modul lain, dan
`AmbangKomitePage.tsx` berdampingan dengan `HalamanMasuk.tsx`.

Perbedaan itu disengaja dan akan hilang saat modul berikutnya menyusul. Ia dicatat di sini
supaya tidak terbaca sebagai kelalaian oleh siapa pun yang membuka repo di antara kedua
tahap.

---

## 17. Penggantian nama diselesaikan mengikuti `D-80` dan `D-81` (2026-09-19)

> **§16 mencatat aturan yang sudah tidak berlaku.** Ia tetap dibiarkan apa adanya sebagai
> rekaman apa yang dikerjakan saat itu; yang berlaku sekarang adalah bab ini.

### 17.1 Aturannya berubah di tengah jalan, dan perubahannya mendasar

`CLAUDE.md` dibangun ulang dan kini memuat **`D-80`** dan **`D-81`** — keduanya menggantikan
§7.4.1 yang saya pakai pada §16.

| Hal | Aturan yang saya pakai di §16 | **`D-80`/`D-81` yang berlaku** |
|---|---|---|
| Istilah domain (`Ambang`, `Jenjang`, `Penyetuju`, `Lini`, `Pita`) | tetap Indonesia | **Inggris**, memakai padanan `CONTEXT.md` |
| Parameter dan variabel lokal | dibiarkan Indonesia | **Inggris** |
| Nama folder modul | — | **tetap Indonesia** — satu-satunya pengecualian (`D-81`) |

Akibatnya penggantian pada §16 **baru separuh benar**: paket, berkas, dan identifier teknisnya
sudah tepat, tetapi seluruh istilah domainnya masih Indonesia. Bab ini menyelesaikannya.

Sasaran `D-19` tetap dipegang dan itu penting: yang dipakai adalah **padanan Inggris yang
benar**, bukan alias Pega. `Adjustment` tetap tidak dipakai; `Object` tetap tidak dipakai.

### 17.2 Peta istilah yang dipakai

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

### 17.3 Tiga kesalahan pada pass ini — polanya sama dan sudah berulang

| # | Kesalahan | Bagaimana ketahuan | Akibat |
|---|---|---|---|
| 1 | Sed untuk variabel lokal di berkas uji **mengganti deklarasi tetapi tidak seluruh pemakaiannya** — `temuan :=` menjadi `findings :=` sementara `saring(temuan, …)` di bawahnya tidak ikut | pemindaian sesudahnya menemukan `temuan` yang sudah tidak punya deklarasi | **dibatalkan seluruhnya**; berkas uji dipulihkan dari salinan sebelum sed |
| 2 | Pemeriksa silang "rujukan versus deklarasi" memakai `^\t` di dalam ERE, padahal **POSIX ERE tidak mengenal `\t`** — ia mencocokkan huruf `t` | 18 konstanta yang jelas-jelas ada dilaporkan hilang | laporan palsu; setelah `[[:space:]]` dipakai, deklarasi terkumpul naik dari 55 menjadi **139** dan temuannya menjadi **nol** |
| 3 | `Aktif:` diganti `Active:` secara seragam, padahal pada `AutoPolicy` medannya bernama `Enabled` | pembacaan ulang literal `AutoPolicy{…}` di berkas uji | tujuh tempat diperbaiki |

Kesalahan kedua adalah **yang keempat kalinya** pola yang sama muncul dalam dua sesi terakhir:
alat ukur dipercaya sebelum diuji pada kasus yang jelas benar. Tiga yang sebelumnya tercatat di
§16.5.

**Dua kesalahan itu tertangkap oleh hal yang sama**: menjalankan pemeriksaan yang **dirancang
untuk gagal** — memulihkan dari salinan sebelum menyentuh apa pun, dan membandingkan hasil alat
terhadap kasus yang sudah pasti benar.

### 17.4 Yang TIDAK dikerjakan, dan alasannya

**Variabel lokal di dalam berkas uji Go tetap berbahasa Indonesia** — `hasil`, `temuan`,
`ambang`, `kebijakan`, dan sekitar 170 kemunculan lain.

Ini **melanggar `D-80`**, dan dicatat sebagai utang, bukan disamarkan. Alasannya dua, dan
keduanya berasal dari percobaan yang benar-benar dijalankan:

1. **Sed tidak dapat membedakan kode dari prosa Indonesia.** Kata-kata itu muncul juga di
   komentar **dan di dalam pesan assertion** — `require.Len(t, …, "kedua baris yang seri harus
   disebutkan")`. Pola berjangkar menyelesaikan sebagian, dan bagian yang tersisa justru yang
   memutus konsistensi deklarasi-pemakaian (§17.3 butir 1).
2. **Tidak ada kompiler untuk membuktikan hasilnya.** Menulis ulang 1.830 baris uji tanpa dapat
   menjalankan `go build` menukar cacat kosmetik dengan risiko cacat nyata.

**Nama fungsi uji tetap berbahasa Indonesia dengan sengaja** — dan ini bukan utang. `CLAUDE.md`
§13.2 menetapkan nama uji **menyebutkan aturannya** sehingga daftar uji terbaca sebagai
dokumentasi aturan bisnis; menerjemahkannya justru melawan ketetapan itu. Pesan assertion
mengikuti alasan yang sama.

**Modul lain belum disentuh** — `auth`, `masterstatus`, `portal`, sisa `platform`, dan komponen
bersama frontend masih memakai penamaan Indonesia sepenuhnya (`HandlerBaru`, `Opsi`, `Layanan`,
`TulisRespon`, `TabelData`, `KolomIsian`). Lingkupnya memang "Komite dan platform dulu".

### 17.5 Keadaan verifikasi — tidak berubah dari §16.6

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
