# Claim PNC — Aplikasi Go + React

Implementasi pengganti aplikasi Pega PRPC 8.3 **Claim PNC**. Dokumen migrasi, ADR, dan papan
tiket pekerjaan berada di repository terpisah: `D:\Jonny\Project\Claude.AI\XML Claim PNC\docs`.

**Yang sudah ada di tahap ini: login dari ujung ke ujung, ditambah satu modul master data.**
Modul bisnis pertama — **Master Status Progres 1** — dibangun pada 2026-09-17 beserta
pemilihan portal per permintaan yang dituntutnya.

| | |
|---|---|
| Tiket yang dikerjakan | `TKT-F3-001` seam identitas · `TKT-F3-003` sesi & token · `TKT-U1-002` alur masuk di frontend |
| Tiket yang disentuh sebagian | `TKT-F1-001` struktur & aturan lapisan · `TKT-F1-002` konfigurasi · `TKT-F1-003` logging · `TKT-F2-001` koneksi & seam repository · `TKT-F6-002` portal melekat pada permintaan · `TKT-U1-003` pustaka komponen baku (`U-2`) · `TKT-U1-001` kerangka portal — menu & bingkai layar |
| Tiket yang **belum** dikerjakan | `TKT-F3-002` provider HCC/HCQ · `TKT-F3-004` tabel peran & izin menu · `TKT-F3-005` middleware otorisasi · `TKT-F6-003` kewenangan portal per pengguna |
**Yang sudah ada di tahap ini: login dari ujung ke ujung, dan modul bisnis pertama — Master Rekening.**
**Yang sudah ada di tahap ini: login dari ujung ke ujung, dan satu modul bisnis — Master Status
Klaim.**

**Ditambah 2026-09-18: modul proses klaim yang pertama — Pelaporan Klaim.**

| | |
|---|---|
| Tiket yang dikerjakan | `TKT-F3-001` seam identitas · `TKT-F3-003` sesi & token · `TKT-U1-002` alur masuk di frontend · **`TKT-F4-005` bagian Master Status Klaim** |
| Tiket yang disentuh sebagian | `TKT-F1-001` struktur & aturan lapisan · `TKT-F1-002` konfigurasi · `TKT-F1-003` logging · `TKT-F2-001` koneksi & seam repository · **`TKT-U2-001` komponen tabel baku** · **`TKT-F4-001` pola master data** |
| Tiket yang **belum** dikerjakan | `TKT-F3-002` provider HCC/HCQ · `TKT-F3-004` tabel peran & izin menu · `TKT-F3-005` middleware otorisasi · `TKT-U1-001` kerangka portal · `TKT-U2-005` pemilihan pustaka tabel |

> **Master Status Klaim belum dapat dipakai terhadap Oracle.** Kolom `LSC_NOTE` pada
> `POOLDATA.M_STS_CLAIM` masih kosong di seluruh 32 baris sampai
> [`migrations/0002`](backend/migrations/0002_master_claim_status.up.sql) dijalankan DBA. Terhadap
> penyimpanan memori ia berfungsi penuh dengan 33 baris nyata. Jalankan `./claimpnc.exe -periksa`
> untuk melihat keadaannya.

> **Pelaporan Klaim juga belum dapat dipakai terhadap Oracle.** Tabel
> `POOLDATA.CPNC_LAPORAN_KLAIM` belum ada sampai
> [`migrations/0003`](backend/migrations/0003_pelaporan_klaim.up.sql) dijalankan DBA. Terhadap
> penyimpanan memori ia berfungsi penuh dengan enam laporan contoh yang mencakup kelima tahap.

Keputusan, penyimpangan dari Steering, dan utang teknis yang disadari dicatat di
[`docs/keputusan-implementasi.md`](docs/keputusan-implementasi.md). Jalannya pengerjaan dicatat di
[`docs/catatan-pengembangan.md`](docs/catatan-pengembangan.md).

---

## Peta repository

Backend dan frontend terpisah penuh, mengikuti susunan aplikasi ClaimQ.

```
claim-pnc/
├── backend/                     modul Go — satu binary (modular monolith)
│   ├── cmd/claimpnc/                entrypoint tunggal, tipis, hanya merakit
│   ├── internal/
│   │   ├── auth/                    MODUL — identitas, sesi, pengguna + seam-nya
│   │   │   ├── usecase/                 orkestrasi: masuk, periksa, perpanjang, keluar
│   │   │   ├── provider/                pengisi seam Identitas — HCQ, Lokal, Berantai, Tiruan
│   │   │   ├── repo/                    pengisi seam penyimpanan — sqlstore, memory
│   │   │   └── http/                    handler, dto, middleware sesi, rute modul
│   │   ├── masterrekening/          MODUL — rekening tujuan pembayaran klaim
│   │   │   ├── usecase/                 orkestrasi: ajukan, ubah, putuskan (komite)
│   │   │   ├── cashier/                 pengisi seam Kasir — klien HTTP, tiruan
│   │   │   ├── notification/            pengisi seam Notifier — pengirim SMTP, tiruan
│   │   │   ├── repo/                    sqlstore (LST_ACCOUNT, LST_BANK_GROUP), memory
│   │   │   └── http/                    handler, dto, galat, rute modul
│   │   ├── portal/                  MODUL — entitas & basis datanya (ADR-0030)
│   │   │   ├── repo/                    sqlstore (M_PORTAL_PNC), memory
│   │   │   └── http/                    rute daftar portal + middleware portal aktif
│   │   ├── masterstatusprogres/     MODUL — Master Status Progres 1 & 2
│   │   │   ├── usecase/                 orkestrasi: list, create, update
│   │   │   ├── repo/                    sqlstore (GCNM_MST_PROGRESS_KLAIM), memory
│   │   │   └── http/                    handler, dto, pemetaan galat, rute modul
│   │   ├── menu/                    MODUL — peta menu & otorisasi pemakainya
│   │   │   ├── usecase/                 group login → izin group + izin login → pohon
│   │   │   ├── repo/                    sqlstore (M_MENU_APLIKASI_PNC, M_OTORISASI_PNC,
│   │   │   │                            M_LOGIN_GROUP_PNC, M_APLIKASI), memory
│   │   │   └── http/                    GET /api/menu
│   │   ├── masterstatus/            MODUL — Master Status Klaim (F-4)
│   │   │   ├── usecase/                 orkestrasi: daftar, ambil, tambah, ubah
│   │   │   ├── repo/                    sqlstore (M_STS_CLAIM), memory + 33 baris contoh
│   │   │   └── http/                    dto, galat, handler, rute
│   │   ├── pelaporanklaim/          MODUL — Pelaporan Klaim (B-14)
│   │   │   ├── usecase/                 catat, ubah, transfer, tautkan klaim, daftar
│   │   │   ├── repo/                    sqlstore (CPNC_LAPORAN_KLAIM), memory + 6 contoh
│   │   │   └── http/                    dto, galat, handler, rute
│   │   ├── riwayatklaim/            MODUL — View History Claim (menu 76)
│   │   │   ├── usecase/                 buka layar (gerbang proteksi), cari
│   │   │   ├── repo/                    sqlstore — 11 kueri pencarian atas T_CLAIM_PNC
│   │   │   │                            + gerbang: MST_PROTEKSI_DATA_PNC dibaca,
│   │   │   │                            CPNC_PEMAKAIAN_PROTEKSI ditulis; memory + contoh
│   │   │   └── http/                    dto, galat, handler, rute
│   │   └── platform/                config, logging, db, middleware, clock, httpserver
│   ├── migrations/                  DDL untuk dijalankan DBA
│   ├── spa/                         penyematan hasil build antarmuka ke binary
│   └── go.mod
├── frontend/                    SPA React + TypeScript + Vite
│   └── src/
│       ├── app/                     kerangka: router, provider, penjaga rute, sesi,
│       │                            bilah atas, sidebar menu (dibaca dari basis data),
│       │                            app/menu/registry.ts — peta MENU_PROGRAM → rute
│       ├── modules/                 satu folder per modul — nama modul bisnis (D-81)
│       ├── components/              pustaka komponen baku
│       └── api/                     klien HTTP dan tipe kontrak API (client.ts, types.ts)
└── docs/                        keputusan implementasi & catatan pengembangan
```

> **Kenapa `backend/spa/` tidak berada di dalam `frontend/`.** Folder itu **bukan kode React** —
> isinya satu berkas Go dan folder `dist/` hasil `npm run build`. Letaknya harus di dalam modul Go
> karena direktif `go:embed` tidak dapat menjangkau ke luar direktori paketnya: pola yang memuat
> `../` ditolak kompilator sebagai `invalid pattern syntax`. Sementara `ADR-0002` menuntut produksi
> menjalankan **satu binary** tanpa runtime Node.js, sehingga berkas statisnya wajib ikut tersemat.
> Seluruh kode sumber antarmuka tetap berada di `frontend/src`.
> **SPA** = *Single Page Application*, istilah yang dipakai `ADR-0002`.

### Aturan susunan yang mengikat

1. **Module-first di `internal/`.** Satu folder per modul bisnis; di dalamnya barulah lapisan.
   Modul berikutnya (`registrasi/`, `komite/`, …) menempel sebagai folder sebelah, bukan
   disebar ke empat tempat.
2. **Arah ketergantungan hanya ke dalam.** Paket akar modul (`internal/auth`) memuat aturan dan
   **mendeklarasikan seam-nya sendiri**; `usecase/`, `provider/`, `repo/`, dan `http/`
   mengimpornya, tidak pernah sebaliknya. Paket akar modul **dilarang** mengimpor HTTP, SQL,
   driver, maupun bentuk JSON wire.
3. **Modul memasang rutenya sendiri.** `platform/httpserver` tidak tahu apa pun tentang isi
   modul; `cmd/claimpnc` memanggil `authhttp.Pasang(...)` di bawah `/api`. Menambah modul
   berarti menambah satu baris di sana.
4. **Frontend tidak mengimpor antar-modul.** Kebutuhan bersama naik ke `components/` atau `api/`.
   Menu dan pembungkus layar hidup di `app/`, bukan di salah satu modul — menaruhnya di dalam modul
   akan memaksa modul lain mengimpornya.
5. **Semua tabel memakai `components/TabelData`.** Tidak ada `<table>` mentah di folder `modules/`.
   Inilah yang mengubah 268 grid sistem lama menjadi satu implementasi. Pilihan pustaka tabel
   (`TKT-U2-005`) masih terbuka; bila kelak diputuskan, yang diganti adalah isi satu berkas itu.
6. **Nilai desain yang berulang tinggal di `src/styles.css`**, bukan diketik ulang per layar —
   bayangan, lengkung, dan kurva gerak. Warna memakai palet bawaan Tailwind (blue untuk aksen,
   slate untuk dasar), bukan warna karangan.

### Tampilan

**Light Mode saja** (keputusan Work Owner 2026-09-17); `color-scheme: light` ditegaskan supaya
kontrol bawaan peramban tidak ikut membalik mengikuti tema sistem pengguna.

| Hal | Ketetapan |
|---|---|
| Aksen | **Blue** (`blue-600`) — tombol utama, menu aktif, cincin fokus. Kontras teks putih di atasnya **5,1:1** — AA, bukan AAA |
| Merah | **Hanya untuk galat.** Ia tidak dipakai sebagai aksen meski warna korporat, supaya tombol Simpan tidak tertukar dengan pesan galat |
| Keadaan kontrol | `hover` mengangkat · `active` menekan · `focus-visible` memberi cincin. Ketiganya wajib ada di setiap nada tombol |
| Gerak | Dimatikan seluruhnya pada `prefers-reduced-motion: reduce` |
| Pembedaan penting | **Tidak pernah hanya warna.** Isian salah ditandai tepi + ikon + teks; nada pesan dibedakan bentuk ikonnya |
| Font & ikon | Font sistem, ikon SVG di `components/Ikon.tsx`. **Tanpa Google Fonts dan tanpa pustaka ikon** — aplikasi berjalan di jaringan tertutup |

> **SPA tersemat ke binary** lewat `go:embed`. Proses yang sedang berjalan memuat tampilan **lama**
> sampai dibangun ulang: `cd frontend && npm run build`, lalu jalankan ulang binary-nya. Selama
> mengerjakan antarmuka, `npm run dev` di port 5173 jauh lebih cepat.

Aturan 2 **belum ditegakkan perkakas** — lihat utang teknis nomor 1 di
`docs/keputusan-implementasi.md`.

---

## Menjalankan

### Prasyarat

- Go 1.22+ (diuji pada 1.27.1)
- Node.js 20+ (diuji pada 24.20.0) — **hanya untuk membangun**, tidak dipakai di produksi

### Jalan tercepat, tanpa basis data

```bash
cd frontend && npm install && npm run build && cd ..
cd backend && go build -o claimpnc.exe ./cmd/claimpnc

./claimpnc.exe          # tanpa satu pun variabel: development + penyimpanan memori
```

Tidak perlu `.env` untuk ini: di `development` penyimpanannya memori dan provider identitasnya
tiruan. Keduanya **menolak berjalan di produksi**.

`npm run build` menaruh hasilnya di `backend/spa/dist/`, dan `go build` menyematkannya ke dalam
binary — produksi tetap menjalankan satu proses saja (`ADR-0002`).

Buka `http://localhost:8080`. Pengguna contoh ada di
[`backend/internal/auth/provider/fake.go`](backend/internal/auth/provider/fake.go):

| Nama pengguna | Kata sandi | Untuk mencoba |
|---|---|---|
| `adminpnc` | `rahasia123` | masuk berhasil |
| `pictekniks` | `rahasia123` | masuk berhasil, profil berbeda |
| `penggunanonaktif` | `rahasia123` | galat "akun tidak aktif" |
| `profilbolong` | `rahasia123` | profil dari sistem identitas tidak lengkap |

Nama, NIK, dan email di daftar itu **karangan** — bukan pegawai nyata.

### Dengan Oracle dan HCC/HCQ

```bash
export APP_ENV=development
export PENYIMPANAN=oracle
export PORTAL_UTAMA=ASM
export POOLDATA_ASM_HOST=... POOLDATA_ASM_PORT=1521 POOLDATA_ASM_SERVICE=...
export POOLDATA_ASM_PENGGUNA=... POOLDATA_ASM_SANDI=...
export IDENTITAS_ADAPTER=hcq HCQ_LOGIN_USER=... HCQ_LOGIN_PASSWORD=...
cd backend && ./claimpnc.exe
```

Skema dijalankan lebih dulu oleh **DBA**, bukan oleh aplikasi — akun aplikasi tidak punya hak DDL:

```
backend/migrations/0001_user_and_session.up.sql       tabel BARU: CPNC_PENGGUNA, CPNC_SESI_AKTIF
backend/migrations/0001_user_and_session.down.sql
backend/migrations/0002_master_claim_status.up.sql     MENGUBAH objek milik sistem lama
backend/migrations/0002_master_claim_status.down.sql
backend/migrations/0003_pelaporan_klaim.up.sql         tabel BARU: CPNC_LAPORAN_KLAIM + sequence
backend/migrations/0003_pelaporan_klaim.down.sql
backend/migrations/0004_riwayat_klaim_proteksi.up.sql  tabel BARU: CPNC_PEMAKAIAN_PROTEKSI + sequence
backend/migrations/0004_riwayat_klaim_proteksi.down.sql
```

> **`0003` dan `0004` dijalankan di SETIAP portal entitas, bukan hanya di portal utama** —
> berbeda dari `0001`. Keduanya menyimpan data bisnis milik satu badan hukum, dan `ADR-0030`
> menetapkan pemisahannya ada di tingkat koneksi. Akibatnya masing-masing dijalankan empat kali,
> dan gagal di salah satunya membuat portal tersebut tertinggal versi.
>
> **`0004` juga menuntut satu hak baca terhadap tabel milik sistem lama.** Akun aplikasi
> membutuhkan `SELECT` pada `POOLDATA.MST_PROTEKSI_DATA_PNC`; tanpa itu, gerbang proteksi layar
> View History Claim menolak setiap pengguna. Perinciannya ada di langkah 4 berkas migrasinya.

Menjalankannya menuntut permintaan perubahan skema tertulis dan persetujuan Work Owner (`D-63`).
**Keduanya belum pernah dijalankan di lingkungan mana pun.**

> **`0002` berbeda sifatnya dari `0001` dan menuntut perhatian lebih.** `0001` hanya menambah dua
> tabel baru; `0002` mengisi kolom pada `POOLDATA.M_STS_CLAIM` dan **mendefinisikan ulang
> `POOLDATA.V_STS_CLAIM`**, yang dibaca 23 rule Pega. Bila definisinya salah, yang rusak bukan layar
> master melainkan laporan TAT dan KPI — dan rusaknya **tanpa galat**, hanya kolom status yang
> kosong. Berkasnya memuat langkah verifikasi yang wajib dijalankan di antara langkah perubahan,
> serta satu berkas DDL yang wajib disimpan DBA lebih dulu agar rollback mungkin.


### Menguji integrasi nyata sebelum migrasi dijalankan

Dua tabel yang **ditulis** aplikasi — `CPNC_PENGGUNA` dan `CPNC_SESI_AKTIF` — baru ada setelah DBA
menjalankan migrasi `0001`. Tiga tabel yang hanya **dibaca** tidak menunggu itu. Karena keduanya
terpisah, identitas nyata dapat dipakai sekarang juga:

```bash
cd backend
PENYIMPANAN=memori IDENTITAS_ADAPTER=hcq ./claimpnc.exe
```

Kredensial diverifikasi **sungguhan** ke HCC/HCQ lalu `POOLDATA.M_LOGIN_PNC`, daftar portal dibaca
dari `POOLDATA.M_PORTAL_PNC`, dan masuk lewat layar berfungsi penuh. Yang hilang hanya dua hal, dan
aplikasi memperingatkannya saat start: sesi tidak tahan restart, dan tidak dikenali instans lain.
Setelah migrasi selesai, ganti ke `PENYIMPANAN=oracle` — tidak ada perubahan kode.

Untuk memeriksa tanpa menjalankan server sama sekali:

```bash
./claimpnc.exe -periksa                       # koneksi, daftar portal, alamat HCQ, kesiapan tabel

read -s SANDI && echo "$SANDI" | \
  ./claimpnc.exe -periksa -login NAMA@sinarmas.id    # + coba masuk sungguhan
```

Kata sandi dibaca dari **stdin**, bukan dari argumen: argumen tersimpan di riwayat shell dan
terlihat di daftar proses. Mode periksa **tidak menulis apa pun**.

> **Menguji HCQ dengan benar.** Masuk non-karyawan yang berhasil **tidak** membuktikan HCQ hidup —
> bila HCQ mati, rantai menandainya putus lalu tetap lolos lewat `M_LOGIN_PNC`, dan keluarannya
> sama persis. Pakailah akun yang pasti tidak ada di kedua sumber: bila jawabannya "kredensial
> salah" berarti keduanya menjawab; bila "sistem identitas tidak dapat dihubungi" berarti ada yang
> putus.
### Pengembangan frontend

```bash
cd frontend && npm run dev     # http://localhost:5173, /api diteruskan ke :8080
```

### Konfigurasi

Seluruhnya dari berkas `backend/.env` atau variabel lingkungan; lihat [`backend/.env.example`](backend/.env.example).
Nilai di lingkungan proses menang atas isi `.env`. Aplikasi **gagal start**
bila ada yang wajib tetapi kosong, dan menyebut semuanya sekaligus. Kata sandi basis data tidak
pernah ikut tercetak di log.

---

## Pengujian

```bash
cd backend  && go vet ./... && go test ./...
cd frontend && npm run typecheck && npm test
```

Seluruh uji berjalan **tanpa basis data dan tanpa jaringan**: provider identitas tiruan dan
penyimpanan di memori keduanya hidup di dalam proses.

---

## Kontrak API

| Metode | Jalur | Sesi | Portal | Keterangan |
|---|---|---|---|---|
| `POST` | `/api/masuk` | — | — | `{nama_pengguna, kata_sandi}` → token + profil |
| `POST` | `/api/keluar` | opsional | — | mencabut sesi di server |
| `GET` | `/api/saya` | wajib | — | identitas pemanggil + batas berlaku sesi |
| `POST` | `/api/sesi/perpanjang` | wajib | — | menggeser batas berlaku |
| `GET` | `/api/portal` | wajib | — | daftar entitas dari `POOLDATA.M_PORTAL_PNC` + portal utama |
| `GET` | `/api/master/posisi-klaim` | wajib | — | empat posisi klaim untuk dropdown; daftar milik aplikasi, bukan isi basis data entitas |
| `GET` | `/api/master/status-progres-1` | wajib | **wajib** | daftar master dari `POOLDATA.GCNM_MST_PROGRESS_KLAIM` |
| `POST` | `/api/master/status-progres-1` | wajib | **wajib** | `{nama, kode_posisi}` → `201` + baris tersimpan; ID diterbitkan server |
| `PUT` | `/api/master/status-progres-1/{id}` | wajib | **wajib** | `{nama, kode_posisi}`; ID tidak pernah ikut berubah |

Tidak ada `DELETE` pada master status progres, dan itu disengaja: sistem lama tidak punya
satu pun pernyataan `DELETE` terhadap tabel itu, dan tabelnya tidak punya kolom penanda
terhapus yang dapat dipakai `D-66`. Alasan lengkapnya di
[`docs/keputusan-implementasi.md`](docs/keputusan-implementasi.md) §10.4.

### Portal entitas pada permintaan modul bisnis

Endpoint bertanda **Portal wajib** menyentuh basis data satu entitas. Ia menuntut header:

```
X-Portal: ASM
```

Permintaan yang tidak menyebutkannya **ditolak**, tidak pernah dilayani portal utama
sebagai cadangan — jatuh ke koneksi default berarti membaca atau menulis data satu badan
hukum di basis data badan hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).

| Kode | HTTP | Artinya |
|---|---|---|
| `portal_tidak_disebut` | 400 | header `X-Portal` tidak ada; pengguna belum memilih entitas |
| `portal_tidak_dikenal` | 400 | alias tidak ada di `POOLDATA.M_PORTAL_PNC` |
| `portal_belum_siap` | 503 | entitasnya ada, tetapi kredensial basis datanya belum diisi |
| `validasi_gagal` | 422 | isian melanggar aturan bisnis; `detail` memuat **seluruh** pelanggaran per isian |
| `tidak_ditemukan` | 404 | baris yang dimaksud tidak ada |

**Pemeriksaannya menjawab "portal ini ada dan koneksinya hidup", bukan "pengguna ini
berwenang atas portal ini".** Kewenangan portal per pengguna adalah `TKT-F6-003` yang
masih terhalang, sehingga setiap pengguna yang sudah masuk dapat memilih portal mana pun
yang koneksinya hidup. Itu bagian `R-20` yang **belum** tertutup.

### Layar yang tersedia

| Jalur di peramban | Layar |
|---|---|
| `/masuk` | masuk |
| `/` | beranda sementara, memuat pemilih portal |
| `/master/status-progres-1` | **Master Status Progres 1** |
| `/master/status-klaim` | **Master Status Klaim** |
| `/master-rekening` | **Master Rekening** |
| `/pelaporan-klaim` | **Pelaporan Klaim** — menu 64 |
| `/riwayat-klaim` | **View History Claim** — menu 76, pencarian riwayat klaim |

Keduanya dapat dicapai lewat **menu utama** di kerangka aplikasi — kolom samping di layar
lebar, deret mendatar di layar sempit (`D-12`: surveyor memakai tablet dan ponsel).
Kerangka juga memuat pemilih portal, tombol keluar, dan peringatan sesi hampir habis,
sehingga ketiganya tersedia di setiap layar.

> **Menu BUKAN kendali akses.** Daftarnya masih tetap, belum disaring izin peran: tabel
> 22 peran dan 51 izin menu adalah `TKT-F3-004`, peta peran → menu hidup di 34 When rule
> yang **lima di antaranya hilang dari export**, dan penugasan operator ke peran **tidak
> ada di basis data**. Yang menjadi kendali adalah pemeriksaan di server pada setiap
> endpoint (`D-59`); menyembunyikan menu hanya kenyamanan tampilan. Navigasinya menyatakan
> keterbatasan itu di layar supaya tidak disalahpahami penguji.

**Menambah layar ke menu = satu baris** di [`frontend/src/app/menu.ts`](frontend/src/app/menu.ts) —
sejajar dengan backend, tempat modul baru cukup menambah satu `Pasang(...)` di `cmd/claimpnc`.
| Metode | Jalur | Sesi | Keterangan |
|---|---|---|---|
| `POST` | `/api/masuk` | — | `{nama_pengguna, kata_sandi}` → token + profil |
| `POST` | `/api/keluar` | opsional | mencabut sesi di server |
| `GET` | `/api/saya` | wajib | identitas pemanggil + batas berlaku sesi |
| `POST` | `/api/sesi/perpanjang` | wajib | menggeser batas berlaku |
| `GET` | `/api/portal` | wajib | daftar entitas dari `POOLDATA.M_PORTAL_PNC` + portal utama |
| `GET` | `/api/master-rekening` | wajib | daftar rekening; saringan `status`, `nomor_rekening`, `nama_pemilik`, `nama_bank`, `komite_saya`, `batas`, `lewati` |
| `POST` | `/api/master-rekening` | wajib | mengajukan rekening baru — selalu lahir berstatus menunggu |
| `GET` | `/api/master-rekening/{kodeBank}/{noRek}` | wajib | satu rekening |
| `PUT` | `/api/master-rekening/{kodeBank}/{noRek}` | wajib | mengubah rekening yang **masih menunggu** keputusan |
| `POST` | `/api/master-rekening/{kodeBank}/{noRek}/keputusan` | wajib | keputusan komite: `status` `"1"` setuju / `"2"` tolak |
| `GET` | `/api/master/status-klaim` | wajib | daftar status klaim + `total` |
| `POST` | `/api/master/status-klaim` | wajib | `{label}` → `201` + baris beserta kode yang dibuat sistem |
| `PUT` | `/api/master/status-klaim/{kode}` | wajib | `{label}` → `200` + baris setelah diubah |
| `GET` | `/api/pelaporan-klaim` | wajib | daftar laporan; saringan `tahap`, `cari`, `cabang`, `batas`, `lewati` — beserta `ringkasan` kelima tahap |
| `POST` | `/api/pelaporan-klaim` | wajib | mencatat laporan baru → `201` + nomor yang dibuat sistem |
| `GET` | `/api/pelaporan-klaim/{nomor}` | wajib | satu laporan |
| `PUT` | `/api/pelaporan-klaim/{nomor}` | wajib | mengubah laporan yang **belum diregistrasi** |
| `POST` | `/api/pelaporan-klaim/{nomor}/transfer` | wajib | menandai laporan dikirim ke ASM pusat; `409` bila sudah |
| `POST` | `/api/pelaporan-klaim/{nomor}/klaim` | wajib | `{nomor_klaim}` — menautkan laporan ke klaim; `409` bila sudah |
| `POST` | `/api/riwayat-klaim/buka` | wajib | menjalankan gerbang proteksi; memakai satu jatah pencarian |
| `GET` | `/api/riwayat-klaim` | wajib | pencarian riwayat klaim; saringan `tipe`, `nilai`, `tanggal_pencarian`, `tanggal_lahir`, `halaman`, `ukuran` |

### View History Claim

Menggantikan harness Pega `PNCSearchKlaim`, yang di menu portal berjudul **"View History Claim"**
(`MENU_ID 76`). Ia layar **pencarian riwayat klaim**, bukan inbox — barisnya bukan pekerjaan,
tidak hilang setelah dikerjakan, dan tidak punya tenggat (`D-79`).

Kedua rutenya menuntut header `X-Portal`: riwayat klaim satu badan hukum bukan riwayat badan
hukum lain, dan jatah proteksi seorang pengguna di satu entitas bukan jatahnya di entitas lain.

| Metode | Jalur | Keterangan |
|---|---|---|
| `POST` | `/api/riwayat-klaim/buka` | menjalankan gerbang proteksi; **memakai satu jatah pencarian** |
| `GET` | `/api/riwayat-klaim` | pencarian. Parameter: `tipe`, `nilai`, `tanggal_pencarian`, `tanggal_lahir`, `halaman`, `ukuran` |

**`POST /buka` mengubah keadaan, dan itulah sebabnya ia bukan `GET`.** Satu jatah pencarian
terpakai setiap kali layar dibuka — perilaku sistem lama, yang menjalankan gerbangnya pada
langkah berprakondisi `TempSearch.SearchType==""`, yaitu hanya sebelum tipe pencarian dipilih.
Layar karena itu memanggilnya **sekali per kunjungan**, lewat `useQuery` ber-`staleTime: Infinity`
— bukan lewat efek, yang di `StrictMode` berjalan dua kali dan akan menghabiskan dua jatah.

#### Gerbang proteksi data

Layar ini satu-satunya yang tergerbang, dan gerbangnya **bukan sekadar izin melainkan jatah yang
berkurang**. Sumbernya `POOLDATA.MST_PROTEKSI_DATA_PNC` milik sistem lama.

| Kode galat | HTTP | Artinya |
|---|---|---|
| `proteksi_belum_terdaftar` | 403 | pemanggil tidak punya baris proteksi untuk modul `PNCSearchKlaim` |
| `jatah_pencarian_habis` | 409 | terdaftar, tetapi jatah pencariannya habis |
| `profil_pemanggil_tidak_lengkap` | 409 | identitas pemanggil tidak terbaca; gerbang tidak dapat diperiksa |

Keduanya dibedakan karena tindakannya berbeda: yang satu menuntut pendaftaran, yang lain menuntut
penambahan jatah.

> **Master tidak pernah ditulis aplikasi ini.** Sistem lama mengurangi jatah dengan `UPDATE`
> terhadap tabelnya sendiri; menirunya berarti dua sistem menulis satu tabel selama masa paralel,
> tepat yang dilarang `P-1`. Di sini master hanya **dibaca**, pemakaian dicatat ke
> `POOLDATA.CPNC_PEMAKAIAN_PROTEKSI` milik aplikasi ([`migrations/0004`](backend/migrations/0004_riwayat_klaim_proteksi.up.sql)),
> dan sisa jatah **dihitung** — jatah master dikurangi pemakaian yang tercatat.
>
> Akibatnya selama kedua layar sama-sama hidup, seorang pengguna memperoleh jatah lebih banyak
> daripada yang tertulis di master. Diterima secara sadar; ia berakhir saat layar Pega dimatikan.

#### Dua belas tipe pencarian

Kodenya dipertahankan apa adanya, **termasuk lompatan dari 9 ke 11** — tidak ada tipe 10 di
sistem lama.

| Kode | Tipe | Isian | Catatan |
|---|---|---|---|
| 1 | No Polis | teks | |
| 2 | Nama Customer | teks | dicocokkan sebagian |
| 3 | Nama Objek | teks | dicocokkan sebagian; **kolom Posisi Klaim kosong** |
| 4 | No PLA | teks | dicocokkan sebagian |
| 5 | No DLA | teks | dicocokkan sebagian |
| 6 | Tgl Kejadian | tanggal | |
| 7 | No Klaim | teks | melayani `PNC-xxxx` **dan** `PNCN.YY.xxxx` |
| 8 | No Akseptasi | teks | |
| 9 | Tanggal Lahir | tanggal | membawa kolom Nama Objek + Tanggal Lahir; **hasilnya selalu kosong** |
| 11 | No Survey | teks | |
| 12 | No Rekening | teks + tanggal | **belum tersedia** — menembus DB Link (`R-03`) |
| 13 | ID Balai Lelang | teks | membawa kolom No Akseptasi + No Balai Lelang |

> **Tiga cacat sistem lama DIREPLIKASI, bukan diperbaiki** (keputusan Work Owner 2026-09-20):
> pencarian Tanggal Lahir membaca isian yang tersembunyi sehingga hasilnya selalu kosong,
> prakondisi pemilih nilainya adalah tautologi, dan pencarian Nama Objek tidak membawa kolom
> Posisi Klaim. Ketiganya dipagari uji dan dinyatakan di layar supaya tidak dilaporkan berulang
> kali sebagai kerusakan modul. Alasan lengkapnya di
> [`docs/keputusan-implementasi.md`](docs/keputusan-implementasi.md) §18.2.
>
> **Yang TIDAK ikut direplikasi: perangkaian nilai ke dalam teks SQL.** Kedua belas kueri lama
> menyisipkan nilai — dan sebagian menyisipkan potongan SQL — langsung ke teks kuerinya.
> `08-TECHNICAL-STRATEGY.md` §4.3 melarangnya tanpa perkecualian, dan keputusan "replikasi apa
> adanya" dibaca berlaku pada perilaku bisnis, bukan pada celah injeksi.

**Paginasi sisi server adalah perubahan perilaku yang disadari.** Tak satu pun kueri lama
menyetel batas baris, sehingga seluruh hasil ditarik sekaligus — terhadap `T_CLAIM_PNC` yang
berisi puluhan juta baris (`D-10`), itu tidak dapat dibawa apa adanya
(`09-DATABASE-STRATEGY.md` §6.3).

### Pelaporan Klaim

Menggantikan harness Pega `InboxRCVApp_Harness`, yang di menu portal berjudul **"Inbox Laporan
Klaim"**, beserta flow action `InputReceiveDocument`.

**Tahap dihitung, tidak disimpan.** Lima tahap — belum ditransfer, belum diregistrasi, sudah
diregistrasi, sudah diakseptasi, ditolak — diturunkan dari tiga kolom, persis seperti sistem lama
menurunkannya dari kombinasi `PNCCASEID` dan `STATUSLOCK`. Dua tahap terakhir diisi modul klaim
(`B-5`, `B-10`) yang belum ada, sehingga hari ini keduanya selalu nol.

**Perpindahan tahap adalah aksi tersendiri**, bukan efek samping penyimpanan seperti di Pega, dan
tidak dapat terjadi dua kali. **Tidak ada `DELETE`** — laporan yang sudah tertaut klaim dirujuk
klaimnya lewat `ClaimData.RCV_ID`.

Hanya **nama pelapor** yang wajib diisi. Itu bukan kelonggaran: laporan kerugian datang dengan
kelengkapan yang berbeda-beda dan harus dapat dicatat sekarang lalu dilengkapi kemudian —
kelengkapan yang sesungguhnya ditegakkan saat registrasi (`B-2`).

### Master Status Klaim

Menggantikan harness Pega `StatusClaimInbox`. **Kode tidak pernah dikirim klien:** pada penambahan
ia dibuat penyimpanan mengikuti skema warisan (kode situs disambung nomor urut tiga digit), pada
pengubahan ia diambil dari jalur URL. **Tidak ada `DELETE`** — layar Pega pun tidak punya, dan
`ADR-0012` melarang master dihapus permanen karena klaim lama merujuknya.

Tiga aturan yang **tidak ada** di sistem lama, diputuskan Work Owner 2026-09-17: nama status wajib
diisi, paling panjang 100 karakter, dan tidak boleh sama dengan status lain (mengabaikan besar-kecil
huruf dan spasi tepi).

> **Batas yang diwarisi.** `LSC_ID` bertipe `CHAR(4)` dan `M_STS_CLAIM_SEQ` berada di 193 per
> 2026-09-17. Saat urutan mencapai 1000 kodenya menjadi lima karakter dan penyisipan ditolak
> `ORA-12899` — sekitar **806 penambahan** lagi. Memotongnya menjadi tiga digit akan menghasilkan
> kode ganda, jadi perilakunya dibiarkan dan batasnya dicatat di sini.

### Alur masuk — dua sumber identitas
Ditetapkan Work Owner 2026-09-16. Urutannya **tidak boleh dibalik**.

1. Kredensial dikirim ke **API HCC/HCQ**. Alamatnya dibaca dari basis data, bukan dari konfigurasi:
   `SELECT servicename FROM POOLDATA.GCNM_CONNECT_REST WHERE app = <portal_alias> AND typeservice = 'HCQ-LOGIN'`.
   Autentikasinya Basic Auth dengan kredensial **aplikasi** (`HCQ_LOGIN_USER` / `HCQ_LOGIN_PASSWORD`).
   Bila `Response.pyErrorCode` = `"200"`, pengguna masuk sebagai **karyawan**; namanya dari
   `EmpResponse.Person.Name`, cabang dan jabatan dari `EmpResponse.Placement`.
2. Bila langkah 1 gagal, kata sandi disidik **SHA-256** (heksadesimal huruf besar) lalu dicocokkan:
   `SELECT login_id, login_name FROM POOLDATA.M_LOGIN_PNC WHERE login_id = ? AND active_status = '1' AND UPPER(hash_password) = UPPER(?)`.
   Bila ditemukan, pengguna masuk sebagai **non-karyawan** (broker / surveyor independen).

Karena non-karyawan tidak punya NIK, kunci alaminya disebut **Identitas**: NIK untuk karyawan,
`LOGIN_ID` untuk non-karyawan, dengan field `jenis` yang menyatakan artinya.

Token dikirim sebagai `Authorization: Bearer <token>`. Galat berbentuk `{kode, pesan}`; **klien
membedakan jenis galat lewat `kode`**, tidak pernah dengan mencocokkan teks pesan.

| Kode | HTTP | Artinya |
|---|---|---|
| `kredensial_salah` | 401 | nama pengguna atau kata sandi salah — **tidak** membedakan apakah akunnya ada |
| `pengguna_tidak_aktif` | 403 | akun ada tetapi dinonaktifkan |
| `sistem_identitas_tidak_terhubung` | 503 / 502 | sistem identitas bermasalah atau menjawab profil tidak lengkap |
| `sesi_kedaluwarsa` | 401 | sesi habis di tengah pekerjaan |
| `sesi_tidak_sah` | 401 | token tidak dikenal atau sudah dicabut |
| `permintaan_cacat` | 400 | badan permintaan tidak dapat dibaca |
| `galat_internal` | 500 | selebihnya |
| `validasi_gagal` | 422 | isian melanggar aturan bisnis; badan memuat `detail` berisi **seluruh** pelanggaran beserta nama field-nya |
| `label_status_sudah_dipakai` | 409 | nama status sudah dipakai baris lain — konflik keadaan, bukan isian cacat |
| `kode_status_sudah_dipakai` | 409 | kode yang dibuat sistem bentrok; seharusnya mustahil |
| `status_klaim_tidak_ditemukan` | 404 | kode yang diminta tidak ada di master |

Bentuk galat ini **sementara**: kontrak galat yang mengikat seluruh aplikasi adalah `TKT-F1-004`,
yang masih terhalang keputusan Work Owner.

---

## Aturan yang mengikat siapa pun yang menulis kode di sini

Diringkas dari `docs/Steering/08-TECHNICAL-STRATEGY.md`; yang di sana tetap yang berlaku, kecuali
susunan folder yang mengikuti §"Peta repository" di atas.

1. **Istilah domain berbahasa Indonesia**, mengikuti `docs/Steering/CONTEXT.md`. Istilah teknis
   berbahasa Inggris mengikuti konvensi Go.
2. **Interface dideklarasikan di paket yang memakainya**, bukan di paket yang mengimplementasikannya.
3. **Teks SQL di berkas `.sql` terpisah**, tidak pernah string di tengah kode Go.
4. **Parameter binding tanpa perkecualian.** Tidak pernah merangkai nilai ke dalam teks SQL.
5. **`SELECT *` dilarang.** Kolom disebut namanya.
6. **Waktu disimpan UTC**, ditampilkan WIB. **Tidak ada penambahan 7 jam manual di mana pun.**
7. **Kredensial dan token tidak pernah masuk log**, termasuk sebagiannya dan termasuk pada jalur galat.
8. **DTO transport terpisah dari tipe modul.**
9. Frontend: seluruh panggilan API lewat hook TanStack Query; tidak ada `fetch` di dalam komponen.
