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
│   │   ├── inboxautoclaim/          MODUL — Inbox Auto Claim (U-3, inbox pertama)
│   │   │   ├── usecase/                 orkestrasi: daftar, rincian, ekspor, unggah
│   │   │   ├── repo/                    sqlstore (TMP_BATCH_AUTO_CLAIM, M_AUTO_CLAIM_PNC), memory
│   │   │   └── http/                    5 rute; satu di antaranya menjawab CSV, bukan JSON
│   │   ├── masterstatus/            MODUL — Master Status Klaim (F-4)
│   │   │   ├── usecase/                 orkestrasi: daftar, ambil, tambah, ubah
│   │   │   ├── repo/                    sqlstore (M_STS_CLAIM), memory + 33 baris contoh
│   │   │   └── http/                    dto, galat, handler, rute
│   │   ├── mastertipesurveyors/     MODUL — Master Tipe Surveyors (F-4), per portal
│   │   │   ├── usecase/                 orkestrasi: daftar, ambil, tambah, ubah
│   │   │   ├── repo/                    sqlstore (M_SURVEYORS), memory + 4 baris contoh
│   │   │   └── http/                    dto, galat, handler, rute
│   │   ├── masterxol/               MODUL — Master XOL (F-4), per portal; BERTINGKAT 4
│   │   │   ├── usecase/                 orkestrasi: daftar, ambil, simpan+ajukan komite,
│   │   │   │                            hapus berkaskade, bekal isian layar
│   │   │   ├── notification/            pengisi seam Notifier — SMTP, tiruan
│   │   │   ├── repo/                    sqlstore (MST_XOL_PNC · _BUSINESS · _LAYER · _REAS),
│   │   │   │                            memory + 4 induk contoh yang meniru keanehan produksi
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
backend/migrations/0001_user_and_session.up.sql          tabel BARU: CPNC_PENGGUNA, CPNC_SESI_AKTIF
backend/migrations/0001_user_and_session.down.sql
backend/migrations/0002_master_claim_status.up.sql       MENGUBAH objek milik sistem lama
backend/migrations/0002_master_claim_status.down.sql
backend/migrations/0003_master_tipe_surveyor.up.sql      indeks unik saja — OPSIONAL
backend/migrations/0003_master_tipe_surveyor.down.sql
backend/migrations/0004_master_surveyor.up.sql           kolom baru — WAJIB bagi modulnya
backend/migrations/0005_master_penyebab_kerugian.up.sql  MENGUBAH objek milik sistem lama
backend/migrations/0005_master_penyebab_kerugian.down.sql
```

Menjalankannya menuntut permintaan perubahan skema tertulis dan persetujuan Work Owner (`D-63`).
**Belum satu pun pernah dijalankan di lingkungan mana pun.**

> **`0002` berbeda sifatnya dari `0001` dan menuntut perhatian lebih.** `0001` hanya menambah dua
> tabel baru; `0002` mengisi kolom pada `POOLDATA.M_STS_CLAIM` dan **mendefinisikan ulang
> `POOLDATA.V_STS_CLAIM`**, yang dibaca 23 rule Pega. Bila definisinya salah, yang rusak bukan layar
> master melainkan laporan TAT dan KPI — dan rusaknya **tanpa galat**, hanya kolom status yang
> kosong. Berkasnya memuat langkah verifikasi yang wajib dijalankan di antara langkah perubahan,
> serta satu berkas DDL yang wajib disimpan DBA lebih dulu agar rollback mungkin.

> **`0005` sesifat dengan `0002`, dan menuntut satu keputusan tambahan sebelum dijalankan.** Ia
> memindahkan keterangan dari `JSON_DATA` ke kolom pada `POOLDATA.M_CAUSE_OF_LOSS` dan
> **mendefinisikan ulang `POOLDATA.V_M_CAUSE_OF_LOSS`**, yang dibaca 19 rule Pega — dua di antaranya
> mengelompokkan laporan dengan `GROUP BY COL_DESC`.
>
> Bedanya dari `0002`: di sana layar yang dipindahkan adalah satu-satunya penulis tabelnya, **di
> sini tidak**. Layar `CauseOfLossInboxSimasOnline` (`MENU_ID 21`) masih menulis tabel yang sama
> dari Pega, dan barisnya akan tampil **tanpa keterangan** — tanpa galat — sesudah langkah 3
> dijalankan. Berkasnya **melarang langkah 3 dijalankan** sebelum Work Owner memilih cara
> menutupnya, dan memuat langkah 0 berisi enam kueri yang wajib dijalankan DBA lebih dulu karena
> enam fakta skemanya masih ditebak (`R-08`).


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
| `GET` | `/api/menu` | wajib | — | peta menu pemanggil dari `POOLDATA.M_MENU_APLIKASI_PNC`, disaring `M_OTORISASI_PNC` |
| `GET` | `/api/master/posisi-klaim` | wajib | — | empat posisi klaim untuk dropdown; daftar milik aplikasi, bukan isi basis data entitas |
| `GET` | `/api/master/status-progres-1` | wajib | **wajib** | daftar master dari `POOLDATA.GCNM_MST_PROGRESS_KLAIM` |
| `POST` | `/api/master/status-progres-1` | wajib | **wajib** | `{nama, kode_posisi}` → `201` + baris tersimpan; ID diterbitkan server |
| `PUT` | `/api/master/status-progres-1/{id}` | wajib | **wajib** | `{nama, kode_posisi}`; ID tidak pernah ikut berubah |
<<<<<<< HEAD
| `GET` | `/api/inbox-auto-claim/tab` | wajib | — | tiga tab beserta tabel sumbernya; `bawaan` menyebut tab yang terbuka lebih dulu |
| `GET` | `/api/inbox-auto-claim` | wajib | **wajib** | daftar batch; saringan `sumber`, `perusahaan`, `halaman`, `ukuran` |
| `GET` | `/api/inbox-auto-claim/ringkasan` | wajib | **wajib** | jumlah batch per perusahaan untuk panel ringkasan; saringan `sumber` |
| `GET` | `/api/inbox-auto-claim/perusahaan` | wajib | **wajib** | daftar Master Auto Claim — **tanpa pemanggil** sejak dropdown dibuang |
| `GET` | `/api/inbox-auto-claim/{kode}/{batch}` | wajib | **wajib** | rincian satu batch; saringan `sumber`, `hasil`, `halaman` |
| `GET` | `/api/inbox-auto-claim/{kode}/{batch}/ekspor` | wajib | **wajib** | berkas **CSV**; `hasil=berhasil` atau `hasil=gagal`, saringan `sumber` |
| `POST` | `/api/inbox-auto-claim/unggah` | wajib | **wajib** | `multipart/form-data`, bagian `berkas`, saringan `sumber` → `201` + batch yang terbit dan baris yang ditolak |
| `GET` | `/api/inbox-auto-claim/format-unggahan` | wajib | — | judul kolom yang diterima berkas unggahan |

> `?sumber=` bernilai `aneka`, `kredit`, atau `travel`. Kosong berarti `aneka`; nilai lain
> **ditolak**, tidak diam-diam dijatuhkan ke bawaannya.

### Inbox Auto Claim

Menggantikan harness Pega `InboxAutoClaim`, yang dijoin ke `POOLDATA.M_AUTO_CLAIM_PNC`.
Perusahaan rekanan mengirim klaim **borongan** sebagai berkas; satu baris di layar adalah
satu pasangan (kode perusahaan × nomor batch).

**Tiga tab, tiga tabel** — persis seperti harness lama:

| Tab | Tabel | Kolom perusahaan |
|---|---|---|
| **Asuransi Kredit** (bawaan) | `POOLDATA.TMP_BATCH_CLAIM_KREDIT` | **`AGENID`** |
| **ANEKA** | `POOLDATA.TMP_BATCH_AUTO_CLAIM` | `INISIALID` |
| **Travel** | `POOLDATA.TMP_BATCH_AUTO_TRAVEL` | `INISIALID` |

Pega menggandakan setiap rule tiga kali untuk ini — utang teknis yang
`03-CURRENT-ARCHITECTURE.md §4.6` sebut sebagai *duplikasi masif per lini bisnis*. Di sini
ketiganya memakai **satu berkas `.sql`** dengan dua titik substitusi (`{{TABEL}}`,
`{{KOLOM}}`) yang diisi sekali saat proses menyala, dari **enum tertutup** — bukan dari
`?sumber=`.

**Penyaring perusahaannya panel ringkasan**, bukan dropdown: donut + tabel
"Nama Perusahaan / Jumlah Batch", dengan baris **All** untuk membatalkan. Dropdown di bilah
judul grid dibuang atas permintaan Work Owner (2026-09-20), mengikuti layar Pega. Tabelnya memuat
**hanya perusahaan yang punya batch di tab itu**, sehingga ketiga tab menghasilkan daftar yang
berbeda dan setiap baris yang tampil pasti menghasilkan isi grid bila diklik.

**Tiga dari tujuh tombol layar lama belum dapat dikerjakan**, dan ketiganya tetap tampil
di layar dalam keadaan nonaktif beserta alasannya:

| Tombol | Keadaan |
|---|---|
| Upload Data Klaim · DETAIL · EXPORT BERHASIL · EXPORT GAGAL | ✅ jalan |
| Proses Klaim | ❌ menunggu `B-2`, `B-3`, `B-5`, `B-10` — ia membuat case klaim utuh |
| Generate DLA | ❌ menunggu `B-9` |
| Cek Premi | ❌ menunggu `S-4` |

**Unggahan bukan penyisipan biasa.** Berkasnya **tidak** memuat kode perusahaan, nomor
produk, maupun mata uang — ketiganya dicari dari nomor polisnya. Akibatnya satu berkas
dapat menghasilkan beberapa batch sekaligus, dan hasil unggahan membedakan **tiga**
keadaan:

| Keadaan | Barisnya | Yang dilakukan pengguna |
|---|---|---|
| lolos | tersimpan, menunggu diproses | — |
| bertanda | **tersimpan** beserta pesan gagalnya, terlihat di grid | memperbaiki datanya |
| ditolak | **tidak tersimpan** — perusahaannya tidak dapat diturunkan dari polis | mengunggah ulang baris itu |

> **Kuerinya sudah tiba (2026-09-19), dan enam dugaan terbukti salah.** Sebelumnya modul
> ini dibangun dari rekonstruksi karena 17 kueri, 3 harness, 1 flow action, dan DDL kedua
> tabel semuanya hilang dari export (`R-16`). Seluruhnya kini ada. Yang paling berat:
> `TGLPROSES` ternyata bagian **PRIMARY KEY**, sehingga penyisipan versi pertama akan
> **ditolak Oracle pada unggahan pertama** — cacat yang tidak dapat ditangkap satu pun uji
> yang berjalan di atas penyimpanan memori. Rinciannya di
> [`keputusan-implementasi.md` §18](docs/keputusan-implementasi.md).
>
> **Gerbang 1 tetap belum dapat dijalankan**, tetapi penghalangnya berpindah: bukan lagi
> kueri yang hilang, melainkan **Pega staging yang dapat ditembak dari luar** (`ADR-0027`) —
> dan kini itu berlaku untuk **ketiga** tab.
>
> **Panel ringkasannya tidak punya baseline Pega.** Komponen yang menggambarnya tidak ada di
> export, jadi ia kemampuan baru — tidak ada yang dapat dibandingkan dengannya. Yang menjaganya
> jujur satuan hitungnya: **jumlah batch**, sama dengan satu baris grid, sehingga angka panel
> dan total grid setelah disaring selalu cocok dan dapat diperiksa pengguna sendiri.
>
> **Dua hal sengaja dibiarkan kosong** dan itu dinyatakan di muka, bukan ditemukan saat
> pengujian: kolom `CURRENCY` pada setiap baris yang diunggah sistem baru (menunggu `B-1`)
> dan kolom `No Ref Bank` pada berkas ekspor (menunggu API pengganti DB Link, `R-03`).
>
> Kedua berkas CSV **bukan** rekonstruksi: judul kolomnya terbaca utuh di
> `REPORT_AUTO_CLAIM_ACT`, termasuk kenyataan bahwa berkas GAGAL punya satu kolom lebih
> sedikit. Dua kolom yang isinya masih dugaan ditandai di `export.go`.

Berkas unggahan berupa CSV dengan judul kolom pada baris pertama — pemisah koma maupun
titik koma sama-sama diterima, dan BOM dari Excel ditangani:

```
inisialid,nopolis,prodke,tglkejadian,tgllapor,col_id,nilaiklaim,currency,note,keyword
MFIN,0100120260001,1,03/01/2026,05/01/2026,12002,12500000.00,IDR,catatan,REF-1
```

`col_id`, `note`, dan `keyword` boleh tidak ada. Tanggal **wajib** `dd/mm/yyyy`: kolomnya
menyimpan teks, dan pemrosesan memotongnya dengan posisi karakter tetap — bentuk lain
menghasilkan tanggal yang salah tanpa satu pun galat.

Unggahan bersifat **semua-atau-tidak sama sekali**: satu baris yang tidak lolos membatalkan
seluruh berkas, dan semua pelanggaran dilaporkan sekaligus beserta nomor barisnya.
=======
| `GET` | `/api/master/tipe-surveyor` | wajib | **wajib** | daftar golongan surveyor dari `POOLDATA.M_SURVEYORS` |
| `GET` | `/api/master/tipe-surveyor/{kode}` | wajib | **wajib** | satu baris, untuk mengisi form ubah |
| `POST` | `/api/master/tipe-surveyor` | wajib | **wajib** | `{deskripsi}` → `201` + baris tersimpan; kode diterbitkan server |
| `PUT` | `/api/master/tipe-surveyor/{kode}` | wajib | **wajib** | `{deskripsi}`; kode tidak pernah ikut berubah |
| `GET` | `/api/master/status-klaim` | wajib | **wajib** | daftar status klaim + `total` |
| `GET` | `/api/master/status-klaim/{kode}` | wajib | **wajib** | satu baris, untuk mengisi form ubah |
| `POST` | `/api/master/status-klaim` | wajib | **wajib** | `{label}` → `201` + baris beserta kode yang dibuat sistem |
| `PUT` | `/api/master/status-klaim/{kode}` | wajib | **wajib** | `{label}` → `200` + baris setelah diubah |
| `GET` | `/api/master/pic-teknik` | wajib | **wajib** | daftar petugas teknik dari `POOLDATA.MST_USER_TEKNIK` |
| `GET` | `/api/master/pic-teknik/{operatorID}` | wajib | **wajib** | satu baris, untuk mengisi form ubah |
| `POST` | `/api/master/pic-teknik` | wajib | **wajib** | mendaftarkan petugas baru; ID Operator diisi pengguna |
| `PUT` | `/api/master/pic-teknik/{operatorID}` | wajib | **wajib** | mengubah petugas; ID Operator tidak ikut berubah |
| `GET` | `/api/master-rekening` | wajib | **wajib** | daftar rekening; saringan `status`, `nomor_rekening`, `nama_pemilik`, `nama_bank`, `komite_saya`, `batas`, `lewati` |
| `GET` | `/api/master-rekening/bank` | wajib | **wajib** | daftar bank untuk dropdown |
| `GET` | `/api/master-rekening/{kodeBank}/{noRek}` | wajib | **wajib** | satu rekening |
| `POST` | `/api/master-rekening` | wajib | **wajib** | mengajukan rekening baru — selalu lahir berstatus menunggu |
| `PUT` | `/api/master-rekening/{kodeBank}/{noRek}` | wajib | **wajib** | mengubah rekening yang **masih menunggu** keputusan |
| `POST` | `/api/master-rekening/{kodeBank}/{noRek}/keputusan` | wajib | **wajib** | keputusan komite: `status` `"1"` setuju / `"2"` tolak |
| `GET` | `/api/master/recovery/form` | wajib | **wajib** | bekal awal layar: nomor batch **perkiraan** + pilihan tahun |
| `GET` | `/api/master/recovery/principal` | wajib | **wajib** | pilihan principal dari `POOLDATA.MST_VIRTUAL_ACCOUNT_PNC` |
| `GET` | `/api/master/recovery/polis/{nomor}` | wajib | **wajib** | identitas lini bisnis, cabang, agen, marketing dari `MST_DET_SALES@ASMD` |
| `GET` | `/api/master/recovery/format-unggahan` | wajib | **wajib** | berkas contoh CSV daftar klaim; `text/csv`, bukan JSON |
| `POST` | `/api/master/recovery/virtual-account` | wajib | **wajib** | terbitkan VA; `201` bila baru, `200` + `dipakai_ulang=true` bila principal sudah punya |
| `POST` | `/api/master/recovery/bukti-bayar` | wajib | **wajib** | unggah bukti bayar (`multipart`, bagian `berkas`) → `id_dokumen` |
| `POST` | `/api/master/recovery/baris-klaim` | wajib | **wajib** | baca CSV daftar klaim (`multipart`); **tidak menyimpan apa pun** |
| `POST` | `/api/master/recovery` | wajib | **wajib** | Transfer Recovery; `sisa`, `nomor_batch`, identitas polis, dan `dicatat_oleh` **ditolak** bila dikirim klien |
>>>>>>> 3dc63dccaff5bb215be5fb885f83ab60b2e5e9ea

Tidak ada `DELETE` pada master status progres, dan itu disengaja: sistem lama tidak punya
satu pun pernyataan `DELETE` terhadap tabel itu, dan tabelnya tidak punya kolom penanda
terhapus yang dapat dipakai `D-66`. Alasan lengkapnya di
[`docs/keputusan-implementasi.md`](docs/keputusan-implementasi.md) §10.4.

**Master Recovery tidak punya `GET` daftar, `PUT`, maupun `DELETE` sama sekali** — dan itu
bukan pekerjaan yang tertinggal. Tidak ada satu pun kueri di export Pega yang MEMBACA
`POOLDATA.MST_RECOVERY_ASM_PENJAMINAN`; layarnya form entri, bukan pengelola data acuan,
dan `INSERTMASTERRECOVERYKLAIM.prc` hanya mengenal INSERT. Keputusan Work Owner 2026-09-20
menetapkan itu ditiru apa adanya. Rinciannya di
[`docs/keputusan-implementasi.md`](docs/keputusan-implementasi.md) §21.

> **Seluruh modul master kini per portal (2026-09-19).** Master Status Klaim dan Master
> Rekening semula dilayani basis data portal **utama** saja; keduanya diselaraskan atas
> keputusan Work Owner, sekaligus mencabut Isolasi Protektif untuk kedua modul itu.
> Sebelum penyelarasan, rekening pembayaran dan status klaim SELURUH badan hukum berada
> di satu tempat. Rinciannya di
> [`docs/keputusan-implementasi.md`](docs/keputusan-implementasi.md) §19.

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
<<<<<<< HEAD
| `/master-rekening` | **Master Rekening** |
| `/inbox-auto-claim` | **Inbox Auto Claim** — layar inbox pertama |
=======
| `/master/rekening` | **Master Rekening** |
| `/master/tipe-surveyor` | **Master Tipe Surveyors** |
| `/master/pic-teknik` | **Master PIC Teknik** |
>>>>>>> 3dc63dccaff5bb215be5fb885f83ab60b2e5e9ea

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
