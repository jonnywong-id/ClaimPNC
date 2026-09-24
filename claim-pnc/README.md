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
│   │   ├── masterlogin/             MODUL — Master Login (menu 37)
│   │   │   ├── usecase/                 orkestrasi: daftar, ambil, tambah, simpan
│   │   │   ├── repo/                    sqlstore (MST_LOGIN_SURVEYOR), memory + 6 contoh
│   │   │   └── http/                    dto, galat, handler, rute
│   │   │                                TANPA IDSource — kuncinya diturunkan dari NAMA
│   │   ├── masterreas/              MODUL — Master Reas (menu 35)
│   │   │   ├── usecase/                 orkestrasi: daftar. TANPA tambah dan simpan
│   │   │   ├── repo/                    sqlstore (T_REINSURER) — hanya SELECT,
│   │   │   │                            memory + 7 baris contoh
│   │   │   └── http/                    dto, galat, rute — hanya GET
│   │   │                                BACA-SAJA — tabelnya ditulis alur PLA/DLA
│   │   ├── detailpenyebab/          MODUL — Detail Penyebab Kerugian (menu 38)
│   │   │   ├── usecase/                 orkestrasi: daftar, ambil, tambah, simpan, cari
│   │   │   ├── repo/                    sqlstore — TULIS ke D_CAUSE_OF_LOSS,
│   │   │   │                            BACA dari V_D_CAUSE_OF_LOSS (+3 view lain);
│   │   │   │                            memory + 8 baris contoh
│   │   │   └── http/                    dto, galat, handler, rute — TANPA DELETE
│   │   │                                Isi baris hidup di JSONDATA; view membentangkannya
│   │   ├── inboxreceivetka/         MODUL — Inbox Receive TKA (menu 49)
│   │   │   ├── usecase/                 orkestrasi: daftar + isi tanggal + beri tahu
│   │   │   ├── repo/                    sqlstore — BACA dari PC_ASM_FW_GCNMFW_WORK
│   │   │   │                            (+ T_CLAIM_PNC, T_GENERAL); TULIS hanya
│   │   │   │                            T_CLAIM_PNC.TGLDOKLENGKAP — tabel engine
│   │   │   │                            Pega TIDAK PERNAH ditulis maupun dikunci
│   │   │   ├── notification/            seam Notifier — SMTP dan perekam
│   │   │   └── http/                    dto, galat, handler, rute — GET + POST
│   │   │                                Modul INBOX pertama yang MENULIS
│   │   ├── inboxinvestigator/       MODUL — Inbox Investigator (menu 48)
│   │   │   ├── usecase/                 orkestrasi: daftar + hitung lama menunggu
│   │   │   ├── repo/                    sqlstore — PC_ASM_FW_GCNMFW_WORK +
│   │   │   │                            PC_ASSIGN_WORKBASKET + T_CLAIM_OBJECTLIST +
│   │   │   │                            T_SURVEYORLIST, hanya SELECT;
│   │   │   │                            memory + 8 baris contoh
│   │   │   └── http/                    dto, galat, rute — hanya GET
│   │   │                                INBOX pertama; BACA-SAJA — layar kerjanya
│   │   │                                belum ada. 4 tabel di 2 skema.
│   │   │                                Kolom ke-9 bercaption "Lama Masuk Inbox"
│   │   │                                tetapi BERISI tanggal survei — apa adanya
│   │   │                                dari Pega. Export Data Investigation
│   │   │                                TIDAK dibawa (keputusan Work Owner)
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
| `GET` | `/api/master/penolakan-klaim` | wajib | **wajib** | daftar Status Penolakan 2 dari `POOLDATA.MST_PENOLAKAN_KLAIM_2`, beserta keadaan persetujuannya |
| `GET` | `/api/master/penolakan-klaim/status-1` | wajib | **wajib** | pilihan Status Penolakan 1 dari `POOLDATA.MST_PENOLAKAN_KLAIM_1` |
| `POST` | `/api/master/penolakan-klaim` | wajib | **wajib** | `{nama, id_status_1, nama_status_1}` → `201`; tepat satu isian induk yang boleh terisi |
| `PUT` | `/api/master/penolakan-klaim/{id}` | wajib | **wajib** | isian sama; **mengembalikan baris ke antrean persetujuan** |
| `GET` | `/api/master/penolakan-komite` | wajib | **wajib** | daftar dari `POOLDATA.MST_REJECTED_KOMITE` |
| `POST` | `/api/master/penolakan-komite` | wajib | **wajib** | `{catatan}` → `201`; ID diterbitkan server |
| `PUT` | `/api/master/penolakan-komite/{id}` | wajib | **wajib** | `{catatan}`; ID tidak pernah ikut berubah |
| `GET` | `/api/master/auto-claim` | wajib | **wajib** | daftar dari `POOLDATA.M_AUTO_CLAIM_PNC`; saringan `status` (`0`/`1`/`2`, bawaan `1`) dan `komite_saya=true` |
| `GET` | `/api/master/auto-claim/{inisial}` | wajib | **wajib** | satu baris, untuk dimuat ke form |
| `POST` | `/api/master/auto-claim` | wajib | **wajib** | mengajukan sumber bisnis baru — selalu lahir berstatus menunggu |
| `PUT` | `/api/master/auto-claim/{inisial}` | wajib | **wajib** | menyimpan **sekaligus** memutuskan: `status` `"0"` simpan · `"1"` approve · `"2"` reject |
| `GET` | `/api/master/auto-claim/sumber-bisnis` | wajib | **wajib** | lookup `POOLDATA.AGENT`; saringan `cari`, minimal 2 huruf |
| `GET` | `/api/master/auto-claim/client` | wajib | **wajib** | lookup `POOLDATA.CLIENT`; saringan `cari`, minimal 2 huruf |
| `GET` | `/api/master/auto-claim/bank` | wajib | **wajib** | daftar bank dari `GENERAL.LST_BANK_GROUP` |
| `GET` | `/api/master/pasal-kerugian` | wajib | **wajib** | daftar dari `POOLDATA.V_M_DATA_PASAL`; **tanpa** lini bisnis |
| `GET` | `/api/master/pasal-kerugian/kategori` | wajib | — | tiga pilihan Kategori; isinya milik aplikasi, bukan data entitas |
| `GET` | `/api/master/pasal-kerugian/bisnis` | wajib | **wajib** | lookup `POOLDATA.BUSINESS`; saringan `cari`, minimal 2 huruf |
| `GET` | `/api/master/pasal-kerugian/{id}` | wajib | **wajib** | satu pasal **lengkap dengan lini bisnisnya**, untuk dimuat ke form |
| `POST` | `/api/master/pasal-kerugian` | wajib | **wajib** | `{no_pasal, isi_pasal, deskripsi, kategori, bisnis}` → `201` |
| `PUT` | `/api/master/pasal-kerugian/{id}` | wajib | **wajib** | isian sama; `IDDATA` tidak pernah ikut berubah |
| `DELETE` | `/api/master/pasal-kerugian/{id}` | wajib | **wajib** | **menghapus permanen**; lihat peringatan di bawah |
| `GET` | `/api/master/bengkel` | wajib | **wajib** | daftar dari `POOLDATA.BENGKEL_HE`, urut `ID_BENGKEL` **menurun**; saringan `status` (`0`/`1`/`2`, bawaan `1`) dan `cari` |
| `GET` | `/api/master/bengkel/{id}` | wajib | **wajib** | satu bengkel, untuk dimuat ke form |
| `POST` | `/api/master/bengkel` | wajib | **wajib** | 33 isian; `ID_BENGKEL` diterbitkan server → `201` |
| `PUT` | `/api/master/bengkel/{id}` | wajib | **wajib** | isian sama; menyimpan **selalu** mengembalikan baris ke Waiting Approval |
| `POST` | `/api/master/bengkel/keputusan` | wajib | **wajib** | `{id_bengkel: [...], status}` — keputusan **borongan**, paling banyak 200 baris |
| `GET` | `/api/master/bengkel/cabang` | wajib | **wajib** | daftar cabang; `GENERAL.LST_USER_ASURANSI` + `LST_DET_CABANG` |
| `GET` | `/api/master/bengkel/kota` | wajib | **wajib** | lookup `CITY`; saringan `cari`, minimal 2 huruf |
| `GET` | `/api/master/bengkel/bank` | wajib | **wajib** | daftar bank dari `GENERAL.LST_BANK_GROUP` |
| `GET` | `/api/master/panel/pilihan` | wajib | — | pilihan Lokasi dan Sisi; **konstanta**, bukan data entitas |
| `GET` | `/api/master/panel` | wajib | **wajib** | daftar dari `POOLDATA.PANEL_HE` **lengkap dengan lokasinya**; saringan `status` (`0`/`1`/`2`, bawaan `1`) dan `cari` |
| `GET` | `/api/master/panel/{id}` | wajib | **wajib** | satu panel beserta lokasinya, untuk dimuat ke form |
| `POST` | `/api/master/panel` | wajib | **wajib** | 10 isian wajib + daftar lokasi; `ID_PANEL` diterbitkan server → `201` |
| `PUT` | `/api/master/panel/{id}` | wajib | **wajib** | isian sama; daftar lokasi **diganti seluruhnya**; menyimpan selalu mengembalikan baris ke Waiting Approval |
| `POST` | `/api/master/panel/keputusan` | wajib | **wajib** | `{id_panel: [...], status, catatan}` — keputusan **borongan**, paling banyak 200 baris |
| `GET` | `/api/master/sparepart/pilihan` | wajib | **wajib** | Kategori dan Tipe dari `GCNM_M_SPAREPART_CATEGORY` dan `GCNM_M_SPAREPART_TYPE`; **data entitas**, hanya yang sudah disetujui |
| `GET` | `/api/master/sparepart` | wajib | **wajib** | daftar dari `POOLDATA.SPAREPART_HE`, urut `ID` menaik; saringan `status` (`0`/`1`/`2`, bawaan `1`) dan `cari` (nama, **nomor**, **kode**) |
| `GET` | `/api/master/sparepart/{id}` | wajib | **wajib** | satu sparepart, untuk dimuat ke form |
| `POST` | `/api/master/sparepart` | wajib | **wajib** | 20 isian, **4 wajib** (nomor, nama, kode, harga); `ID` diterbitkan server → `201` |
| `PUT` | `/api/master/sparepart/{id}` | wajib | **wajib** | isian sama; menyimpan **selalu** mengembalikan baris ke Waiting Approval |
| `POST` | `/api/master/sparepart/keputusan` | wajib | **wajib** | `{id_sparepart: [...], status}` — keputusan **borongan**, paling banyak 200 baris. **Tanpa catatan**: tabelnya tidak punya kolom penampungnya |
| `GET` | `/api/master/grouping-sparepart/pilihan` | wajib | **wajib** | Panel dari `POOLDATA.PANEL_HE` (hanya yang disetujui) dan Tipe Kendaraan dari `branddetail`; **data entitas** |
| `GET` | `/api/master/grouping-sparepart/sisi` | wajib | **wajib** | `?id_panel=&nama_panel=` — sandi Sisi milik satu panel, dari `POOLDATA.LOKASI_PANEL_HE`. Daftar kosong adalah jawaban yang **sah** |
| `GET` | `/api/master/grouping-sparepart/sparepart` | wajib | **wajib** | `?nomor=` — lima isian turunan dari `POOLDATA.SPAREPART_HE`; `404` bila nomornya tidak ada |
| `GET` | `/api/master/grouping-sparepart` | wajib | **wajib** | daftar dari `SPAREPART_HE_VIN_KEY` **di-JOIN** dengan `SPAREPART_HE_VIN_GROUP`, urut `ID` menaik; saringan `status` (`0`/`1`/`2`, bawaan `1`) dan `cari` (nomor & nama sparepart, nama panel, **no rangka**) |
| `GET` | `/api/master/grouping-sparepart/{id}` | wajib | **wajib** | satu grouping, untuk dimuat ke form |
| `POST` | `/api/master/grouping-sparepart` | wajib | **wajib** | 8 isian, **4 wajib** (nomor sparepart, nama panel, no rangka, sisi). Lima isian turunan **dibaca server**, bukan dikirim klien; `ID` dan nomor grup diterbitkan server → `201` |
| `PUT` | `/api/master/grouping-sparepart/{id}` | wajib | **wajib** | isian sama; menyimpan **selalu** mengembalikan baris ke Waiting Approval |
| `POST` | `/api/master/grouping-sparepart/keputusan` | wajib | **wajib** | `{id_grouping: [...], status}` — keputusan **borongan**, paling banyak 200 baris. **Tanpa catatan**: kedua tabelnya tidak punya kolom penampungnya |
| `GET` | `/api/master/tipe-sparepart/pilihan` | wajib | **wajib** | Kategori dari `POOLDATA.GCNM_M_SPAREPART_CATEGORY` (**hanya yang disetujui**); **data entitas**. Penanda `terpotong` ikut dikirim bila daftarnya mencapai batas |
| `GET` | `/api/master/tipe-sparepart` | wajib | **wajib** | daftar dari `GCNM_M_SPAREPART_TYPE` **LEFT JOIN** tabel kategori, urut `PART_SECTION_ID` menaik; saringan `status` (`0`/`1`/`2`, bawaan `1`) dan `cari` (nama tipe **dan nama kategori**) |
| `GET` | `/api/master/tipe-sparepart/{id}` | wajib | **wajib** | satu tipe, untuk dimuat ke form |
| `POST` | `/api/master/tipe-sparepart` | wajib | **wajib** | 2 isian, **keduanya wajib** (nama, kategori). Keberadaan kategori **diperiksa server**; ID diterbitkan server → `201` |
| `PUT` | `/api/master/tipe-sparepart/{id}` | wajib | **wajib** | isian sama; menyimpan **selalu** mengembalikan baris ke Waiting Approval, dan **boleh memindahkan** tipe ke kategori lain |
| `POST` | `/api/master/tipe-sparepart/keputusan` | wajib | **wajib** | `{id_tipe_sparepart: [...], status}` — keputusan **borongan**, paling banyak 200 baris. **Tanpa catatan**: tabelnya tidak punya kolom penampungnya |
| `GET` | `/api/master/supplier` | wajib | **wajib** | daftar dari `M_SUPPLIER`; saringan `cari` (nama, kota, contact person) |
| `GET` | `/api/master/supplier/{id}` | wajib | **wajib** | satu supplier, untuk dimuat ke form |
| `POST` | `/api/master/supplier` | wajib | **wajib** | 23 isian, 15 wajib; ID diterbitkan server → `201`. Selalu lahir **belum aktif** |
| `PUT` | `/api/master/supplier/{id}` | wajib | **wajib** | isian sama; **nama ditolak bila berbeda** dari yang tersimpan |
| `GET` | `/api/master/supplier/cabang` | wajib | **wajib** | daftar cabang dari `M_BRANCH` |
| `GET` | `/api/master/supplier/kota` | wajib | **wajib** | lookup `CITY`; saringan `cari`, minimal 2 huruf |
| `GET` | `/api/master/supplier/negara` | wajib | **wajib** | daftar negara dari `COUNTRY` |
| `GET` | `/api/master/supplier/bank` | wajib | **wajib** | daftar bank dari `GENERAL.LST_BANK_GROUP` |
| `GET` | `/api/master/supplier/sandi` | wajib | **wajib** | **kelima** daftar dropdown bersandi sekaligus; lihat catatan di bawah |
| `GET` | `/api/master/login` | wajib | **wajib** | daftar dari `POOLDATA.MST_LOGIN_SURVEYOR`, urut `NAMA`; saringan `cari` (nama, login, email). **Tanpa saringan status** — tabelnya tidak punya kolom `APPROVAL` |
| `GET` | `/api/master/login/{login}` | wajib | **wajib** | satu baris, untuk dimuat ke form. Jalurnya memakai **LOGIN**, bukan ID terpisah |
| `POST` | `/api/master/login` | wajib | **wajib** | **4 isian**, 3 wajib (nama, email, telp); `LOGIN`, `STSLOGIN`, dan `LOGINLEADER` **diturunkan server** → `201` |
| `PUT` | `/api/master/login/{login}` | wajib | **wajib** | isian sama; **nama ditolak bila berbeda** dari yang tersimpan. Ketiga kolom turunan tidak pernah ikut berubah |
| `GET` | `/api/master/reas` | wajib | **wajib** | daftar dari `POOLDATA.T_REINSURER`, urut `REINSURERNAME, TYPE, REINSURERID`; saringan `cari` (kode, nama, login, email). **Satu-satunya modul master tanpa endpoint tulis** — lihat catatan di bawah |

### Master Reas

Menggantikan harness Pega `DataMemberReas` (MENU_ID 35) atas `POOLDATA.T_REINSURER` —
**tujuh kolom, enam dibaca, nol isian**.

Ia **satu-satunya layar master yang BACA-SAJA**, dan itu keputusan berdasar bukti, bukan
pekerjaan yang belum selesai.

#### Kenapa tidak ada endpoint tulis

Satu-satunya penulis tabel ini di sistem lama adalah **alur PLA/DLA**, bukan layar master.
Penelusuran pemanggilnya berhenti di dua berkas, dan keduanya layar detail:

```
Database/UPDATEREAS.prc              prosedur upsert-nya
  ← RDB List/UpdateEmailReas-SQL.xml     satu-satunya Connect-SQL yang memanggilnya
      ← Activity/UpdateDetailPLA2-Act.xml    layar detail PLA
      ← Activity/UpdateDetailDLA2-Act.xml    layar detail DLA
```

Harness-nya sendiri hanya memuat satu grid dan satu tombol **Refresh**. Jadi baris
reasuransi lahir dan berubah sebagai efek samping pengiriman PLA/DLA (`B-9`).

**Keterbatasan buktinya dinyatakan, bukan ditutupi:** section grid `BrowseListMemberReas`
tidak ada di export (`R-16`), sehingga ini rekonstruksi. Penambahan jalur tulis kelak
menyentuh `Repo.Insert`/`Update` dan rutenya; domainnya tidak perlu berubah — dan tiga
kelompok uji sengaja akan gagal lebih dulu supaya penambahan itu menjadi keputusan yang
disadari.

#### Tujuh kolom, satu ditulis tetapi tidak pernah dibaca

| Kolom | Alias klipboard Pega | Dikirim | Catatan |
|---|---|---|---|
| `REINSURERID` | `CityID` | ya | penghubung ke `T_PLALIST.REINSCODE` dan `IDREAS` |
| `REINSURERNAME` | `District` | ya | bagian kunci alami |
| `LOGIN` | `DistrictID` | ya | **menentukan klaim yang dilihat mitra** |
| `EMAIL` | `City` | ya | tujuan pemberitahuan PLA/DLA |
| `COUNTRY` | `Country` | ya | ikut tercetak di dokumen PLA/DLA |
| `TYPE` | — | ya | karakter pertama nomor dokumen PLA/DLA |
| `COUNTRYID` | — | **tidak** | ditulis `UPDATEREAS`, **nol pembaca di seluruh export** |

Aliasnya contoh telak utang `03-CURRENT-ARCHITECTURE.md` §4.2: alamat surel menjadi `City`,
nama perusahaan reasuransi menjadi `District`.

#### Kunci alaminya TIGA kolom

`UPDATEREAS` memeriksa keberadaan baris dengan `REINSURERID + REINSURERNAME + TYPE`
sekaligus. Akibatnya **satu perusahaan dapat muncul beberapa kali** di daftar — satu baris
per jenis dokumen, dengan surel yang berbeda-beda.

Baris ber-`TYPE = '1'` adalah **cadangan**: `BrowseEmailReas` memakainya ketika tidak ada
baris yang cocok dengan jenis dokumen yang sedang dikirim. Layar menandainya, dan
`-periksa` mencacah perusahaan yang tidak punya baris cadangan.

#### Enam pemeriksaan pada `claimpnc -periksa`

Dua di antaranya khas tabel ini: `LOGIN` yang dipakai lebih dari satu kode reasuransi —
keadaan yang sistem lama sendiri akui mungkin terjadi (`order by reinsurerid desc fetch
next 1 row only`) dan yang berarti seseorang berpotensi melihat klaim mitra lain (`R-20`) —
dan perusahaan tanpa baris cadangan.

Hak yang perlu diminta ke DBA untuk tabel ini adalah **BACA saja**.

### Master Login

Menggantikan harness Pega `MasterLoginSurvey` (MENU_ID 37) atas
`POOLDATA.MST_LOGIN_SURVEYOR` — **tujuh kolom, lima isian, tanpa tab**.

Ia **master paling sederhana di aplikasi ini**, dan itu bukan kebetulan melainkan akibat
bentuk tabelnya. Yang **tidak** ada di sana menentukan bentuk layarnya:

| Yang tidak ada | Akibatnya |
|---|---|
| kolom `APPROVAL` | tanpa tab, tanpa alur persetujuan, tanpa keputusan borongan |
| kolom pelaku dan waktu | **siapa mengubah apa tidak tersimpan di mana pun** |
| penanda aktif | **tidak ada cara menyatakan sebuah login sudah tidak berlaku** |

#### Kuncinya DITURUNKAN, bukan diterbitkan — satu-satunya master yang begitu

`Activity/SetLoginSurveyor_act` membentuk `LOGIN` dari `NAMA` dengan membuang **spasi,
titik, koma, dan tanda hubung**, tanpa mengubah huruf besar-kecil. Tidak ada sequence dan
tidak ada `MAX(...)+1`.

Satu perbedaan yang disengaja: **penurunannya pindah ke server.** Di Pega ia dihitung di layar
lalu dikirim kembali sebagai isian biasa, sehingga permintaan yang tidak datang dari layar
dapat mengirim `LOGIN` apa pun — dan `LOGIN` adalah kunci baris, penyaring `WHERE` setiap
pernyataan simpan. Layar tetap memperlihatkan hasilnya saat pengguna mengetik; ia
menghitungnya untuk **ditampilkan**, bukan untuk dikirim.

Akibat lanjutannya: **Nama terkunci setelah baris tersimpan** (`pyDisabledWhen` pada kontrol
Nama di Pega), dan **server ikut menolaknya** — penguncian di antarmuka adalah kenyamanan
tampilan, dan permintaan yang tidak datang dari layar tidak tersentuh olehnya.

> **Akun aplikasi TIDAK diterbitkan, dan layar menyatakannya.** Di Pega, menyimpan baris baru
> ikut memanggil `GCNMCreateOperator` dengan **kata sandi yang sama untuk setiap orang**, dan
> pesan suksesnya menyebut kata sandi itu terang-terangan.
>
> Tiga hal menghalanginya: tidak ada operator Pega untuk diterbitkan, kontrak identitas `F-3`
> belum ada (`R-14`), dan mengumumkan kata sandi bagi akun yang tidak diterbitkan adalah
> **keterangan yang salah** — petugas akan menyampaikannya kepada surveyor yang kemudian
> tidak dapat masuk. Perlakuan yang sama dipakai Master Bengkel.

> **Dua kolom ditulis tetapi tidak pernah digambar layar Pega.** `STSLOGIN` (selalu
> `"Member"`) dan `LOGINLEADER` diturunkan sistem dan **tidak dapat dikirim klien** —
> mengirimnya ditolak sebagai permintaan cacat. Keduanya tetap **ditampilkan baca-saja** di
> form: nilainya menentukan peran dan tim seseorang, dan menyembunyikan hal yang tersimpan
> tidak membuatnya tidak tersimpan.
>
> `LOGINLEADER` diturunkan dari **leader milik pengguna yang menyimpan**, bukan dari login
> pengguna itu sendiri — sehingga surveyor yang menambahkan rekannya memberi rekan itu leader
> yang sama dengan dirinya.

> **Cakupan daftarnya adalah REKONSTRUKSI.** Rule yang mengisi grid Pega
> (`LoginMemberSurvey.pxResults`) **tidak ada di export** (`R-16`); yang ada hanyalah pemuat
> satu baris untuk tombol Ubah. Yang dipakai adalah bentuk kueri yang benar-benar ada, tanpa
> penyaring — sehingga daftarnya memuat seluruh baris entitas itu.
>
> Bacaan lain yang mungkin: daftarnya disaring per tim, sehingga seorang leader hanya melihat
> anggotanya. Perbedaan keduanya menentukan **siapa yang boleh menyunting login milik tim
> lain**, dan itu pertanyaan terbuka untuk Work Owner.

> **Pemeriksaan login ganda juga REKONSTRUKSI.** Pega memeriksanya terhadap **tabel operator
> Pega** (`Data-Admin-Operator-ID` lewat report definition `GCNMGetListOfOperators`), yang
> tidak ada di sistem baru. Penggantinya keunikan `LOGIN` pada tabel modul ini sendiri — kunci
> alaminya — dengan pesan Pega apa adanya: *"Login sudah terdaftar dengan nama yang sama"*.

#### Grid — lima kolom, satu-lawan-satu dengan Pega

`Nama · Login · Email · Telp · Alamat`, dibaca dari `pyLabelFieldValue` pada
`Section/BrowseLoginSurveyor-Section.xml`. `Status Login` dan `Login Leader` **tidak ada di
grid** karena Pega pun tidak punya; keduanya hanya muncul saat sebuah baris dibuka.

Paginasi **15 baris**, dari `pyPageSizeOther` pada grid itu. Satu selisih yang disadari:
gridnya memakai `pyPageMode = "Next Previous"` sedangkan `DataTable` menomori halamannya —
modenya diseragamkan dengan seluruh layar master lain agar tidak ada dua gaya paginasi hidup
berdampingan.

### Master Supplier

Menggantikan harness Pega `MasterSupplier` (MENU_ID 29) atas tabel `M_SUPPLIER`, form **24 isian,
15 di antaranya wajib**, tanpa tab.

Ia master **komersial**: tenggat bayar, tenggat kirim, dan rekening tujuan pembayaran di dalamnya
menentukan ke mana uang berpindah.

Ia juga **layar master pertama yang seluruh isinya tinggal di satu kolom JSON.** `M_SUPPLIER` hanya
punya `ID`, `OLDID`, dan `JSONDATA` — tidak ada kembaran berkolom bernama seperti
`POOLDATA.BENGKEL_HE`.

#### Kenapa modul ini MENULIS dokumen JSON, padahal Master Bengkel menolaknya

Master Bengkel menghadapi pertanyaan yang sama dan menjawabnya berbeda, dan alasannya bukan selera:
di sana nama kunci JSON-nya **tidak diketahui**, karena badan `@GCNM.GetPageJSONString()` tidak ikut
di export.

Di sini keadaannya berbeda. `RDB List/GetDataEditMasterSupller-SQL.xml` **membaca kembali setiap
kuncinya satu per satu**, sehingga kedua puluh lima nama kunci terbaca lengkap — **tanpa satu pun
tebakan**. Dijaga `TestWrittenKeysMatchReadKeys`.

Notasi titik Oracle (`A.JSONDATA.NAMA`) tidak disalin: ia tidak ada padanannya di PostgreSQL.
Penggantinya `JSON_VALUE`, yang berlaku di kedua basis data — justru alasan `D-24` mewajibkan
PostgreSQL 17. Dijaga `TestNoOracleDotNotation`.

#### Dua jalur penyimpanan yang berperilaku BERBEDA

| Jalur | Status aktif | Persetujuan |
|---|---|---|
| **Tambah** | selalu `"0"`, berapa pun yang dipilih di form | **selalu** diminta |
| **Ubah** | menyusul pilihan di form | diminta **hanya** bila supplier-nya diaktifkan |

Artinya **menonaktifkan supplier berlaku seketika, tanpa persetujuan siapa pun.** Itu jalur
satu-satunya di modul ini yang mengubah keadaan tanpa melewati antrean mana pun, dan itu perilaku
sistem lama yang ditiru apa adanya (`EditMasterSupplier_post` step 12). Layar menyebutkannya
terang-terangan di bawah tombol Simpan.

> **Nama supplier tidak dapat diubah setelah tersimpan.** Aturannya dibaca dari
> `pyReadOnlyCondition` pada isian NAMA — satu-satunya isian di form itu yang punya syarat
> read-only. Layar mengunci isiannya, **dan server ikut menolaknya**: penguncian di antarmuka
> adalah kenyamanan tampilan, dan permintaan yang tidak datang dari layar itu tidak tersentuh
> olehnya.

> **Daftar nilai kelima dropdown bersandi TIDAK ada di export** (`R-16`): kelimanya dirender
> `pxDropdown` dengan `pyListSource=associated`, dan rule Field Value-nya tidak ikut dikirim.
>
> Yang ditawarkan `/api/master/supplier/sandi` adalah **gabungan** sandi yang artinya terbukti dari
> percabangan activity — `JENIS_STATUS`, `STS_AKTIF_PROMLIST`, `STS_AUTOPAYMENT` — dengan sandi yang
> **benar-benar dipakai** baris yang ada. Tidak ada satu pun yang dikarang, dan sandi yang hanya
> ditemukan di data ditampilkan apa adanya tanpa tebakan artinya.

> **Satu hal yang WAJIB dipastikan DBA sebelum jalur tulis dipakai di produksi:** lebar dan tipe
> kolom `JSONDATA`. Seluruh 28 nilai masuk ke satu kolom, sehingga kolom yang terlalu sempit menolak
> **baris utuh** — bukan isian yang kepanjangan. `claimpnc -periksa` melaporkannya, sekaligus
> memeriksa berapa baris yang dokumen JSON-nya benar-benar terbaca: `JSON_VALUE` menjawab NULL
> alih-alih gagal, sehingga kunci yang dinamai lain membuat layar menampilkan baris kosong **tanpa
> satu pun galat**.

#### Grid — sembilan kolom, satu-lawan-satu dengan Pega

`ID · Input Nama · ALAMAT · TELP · JENIS SUPPLIER · STATUS REKANAN · STATUS AKTIF ·
POSISI · OPTION`, terverifikasi dari `pyColumnCount = 9` pada
`Section/InboxMasterSupplier-Section.xml`. Tombolnya bercaption **Edit**, bukan "Ubah".

Grid ini **tidak punya urutan bawaan** (`pySortType` dan `pyInitialSortColumn` kosong,
`pyDisplayInitialSort = false`), sehingga urutan menurut nama pada `supplier_list` adalah
pilihan yang ditentukan — bukan tiruan.

Dua kolom menampilkan **label**, bukan sandi: `JENIS SUPPLIER` dan `STATUS REKANAN`
terikat `.JENIS_STATUS_NOTE` dan `.STS_REKANAN_NOTE`. Sumber labelnya berasal dari kueri
daftar yang **hilang dari export** (`R-16`), sehingga layar mencarinya dari daftar
`/sandi` dan jatuh ke sandinya sendiri bila artinya belum diketahui — tidak pernah
tebakan.

`POSISI` digambar meski selalu kosong: ia milik `pooldata.proteksi_klaimmbu`, dan pada
layar Pega yang sesungguhnya pun kolom itu tampak kosong.

#### Paginasi — 20 baris per halaman, meniru `pyGridPaginator`

Setelannya dibaca dari `pyGridProps` pada `Section/InboxMasterSupplier-Section.xml`:
`pyPageMode = Numeric` (nomor halaman, bukan "muat lebih banyak") dan `pyPageSize = 20`.
Paginatornya rata kanan di kaki tabel, sama seperti gaya sel aslinya.

Paginasinya **di peramban**, bukan di server — grid Pega pun terikat pada page list
klipboard (`pyPageListProperty`) dan memotong daftar yang **sudah** tersaring. Paginasi
keyset sisi server adalah `TKT-U2-001`, yang Steering sebut sebagai perubahan perilaku,
bukan pemeliharaan.

Ukuran halamannya **prop per layar**, bukan bawaan `DataTable`, karena di sistem lama pun
ia berbeda-beda: Supplier, Bengkel, dan Panel memakai **20**; Rekening, Status Klaim,
Status Progres, Pasal Kerugian, dan Penolakan Klaim memakai **15** (`pyPageSizeOther`).
Kedelapan layar lain **belum menyalakannya** — bawaan `DataTable` tetap tanpa paginasi
supaya modul yang sudah selesai tidak berubah perilakunya. Menyalakannya kelak cukup satu
prop per layar.

> **Sisi pemutus persetujuan TIDAK dibangun.** Setiap penyimpanan menyisipkan baris ke
> `pooldata.proteksi_klaimmbu`, persis seperti sistem lama — tetapi tidak ada satu pun rule di
> export yang membacanya, menyetujuinya, menolaknya, atau memajukan `POSISI`-nya (`R-16`).
> Membangunnya berarti mengarang aturan yang menentukan supplier mana yang boleh dipakai.

### Master Panel HE

Menggantikan harness Pega `MasterPanel_HE` (MENU_ID 30) atas `POOLDATA.PANEL_HE`, **15 kolom**,
**ditambah tabel anak `POOLDATA.LOKASI_PANEL_HE`**.

Ia **layar master pertama yang mengelola baris anak**: setiap panel punya daftar lokasi — KIRI,
KANAN, DEPAN, BELAKANG, LAIN-LAIN — beserta sisinya. Seluruh layar master sebelumnya rata.

**Tata letaknya ditiru apa adanya dari Pega**, tanpa satu pun selisih:

| Hal | Isi | Asal |
|---|---|---|
| Judul | "Master Panel HE" | `pyCaption` pada `Section/ListPanelHE` |
| Tab | **Approve · Reject · Waiting Approval** — Reject di **tengah** | urutan section pada `Section/BrowsePanelHE` |
| Kolom grid | **11 kolom berdampingan**, nilainya sebagai sandi apa adanya | `pyLabelFieldValue` pada `BrowsePanelHEApproval` |
| Urutan baris | `ID_PANEL` **menurun** | `pySortType=DESC` pada report definition |
| Paginasi | nomor halaman — **15 baris** tab Approve, **50 baris** tab Reject & Waiting | `pyPageSize` / `pyPageSizeOther` + `pyPageMode=Numeric` per section |
| Tombol | Tambah · Refresh | `pyButtonLabel` pada `ListPanelHE` |

Ukuran halaman **berbeda antartab**, dan itu ditiru apa adanya. Perbedaannya hampir pasti tidak
disengaja — ketiga tab membaca report definition yang sama — tetapi "menurut kami lebih rapi"
bukan alasan yang cukup untuk menyimpang dari `D-13`. Penyeragamannya menunggu keputusan Work
Owner. Daftar lokasi di dalam form **tidak** dipaginasi, mengikuti `pyPageMode="None"` pada grid
lokasi Pega.

Kesembilan kolom `STATUS …` **tidak** diringkas dan **tidak** diterjemahkan menjadi
"Ya"/"Tidak": daftar nilainya tidak ada di export (`R-16`), dan menebaknya di tempat yang paling
terlihat adalah tebakan yang paling mahal. Kolom lokasi **tidak ada di grid** karena Pega pun
tidak punya — daftar lokasi hanya muncul saat sebuah panel dibuka.

**Tiga tombol yang ada di layar Pega TIDAK dibawa**: "Upload Document", "Upload Data Master
Panel", dan "Upload Data Lokasi Panel". Ketiganya memanggil local action `UploadDocument`,
`PNCUploadMasterPanelCSV`, dan `PNCUploadLokasiPanelCSV` — **tidak satu pun ada di export**
(`R-16`), sehingga susunan kolom CSV-nya, validasinya, dan apakah baris hasil unggah masuk
antrean persetujuan seluruhnya tidak diketahui. Yang terakhir menentukan: unggah massal yang
melewati `APPROVAL="0"` adalah jalan pintas yang memintas seluruh kontrol persetujuan.
Perlakuannya sama dengan dua tombol unggah di Master Bengkel.

**Kesepuluh isian induknya WAJIB** — kesepuluhnya bertanda `pyRequired=true` di layar Pega,
termasuk `EXCLUSION_C` yang artinya tidak disebut di mana pun dalam export.

**Daftar nilai kesembilan penanda `STS_*` TIDAK ada di export** (`R-16`): kesembilannya dirender
dropdown atau radio dengan `pyListSource=associated`, dan rule Field Value-nya tidak ikut
dikirim. Keduanya karena itu disimpan sebagai teks apa adanya, dan layar menawarkan nilai yang
sudah dipakai panel lain sebagai saran — bukan daftar yang dikarang.

> **Dua hal yang WAJIB dipastikan DBA sebelum jalur tulis dipakai di produksi**, dan keduanya
> dilaporkan `claimpnc -periksa`:
>
> 1. Apakah `PANEL_HE` sebuah **tabel** atau sebuah **view** atas `M_PANEL_HE.JSONDATA`.
> 2. Apakah kolom `LOKASI_PANEL_HE.NAMA` memuat hal yang **sama** dengan `LOKASI_PANEL`. Modul
>    ini menulis keduanya dengan nilai yang sama — asumsi yang disimpulkan dari dua kueri yang
>    saling melengkapi, bukan dari DDL yang belum ada (`R-08`). Bila terbantah, `masterpanel.sql`
>    harus diperbaiki lebih dulu; penulisan yang salah akan merusak kolom yang dibaca modul
>    Grouping Sparepart.

> **Daftar lokasi DIGANTI seluruhnya saat menyimpan** — dibuang lalu disisipkan ulang, di dalam
> satu transaksi. Ini bertentangan dengan `D-66` (soft delete menyeluruh), dan pertentangannya
> dinyatakan terbuka: baris anak tidak punya kunci sendiri, penggantinya belum diputuskan
> (`ADR-0013`), dan Work Owner memilih **"jalankan as is"**. `DELETE` terhadap tabel **induk**
> tetap dilarang, dijaga `TestDeleteOnlyOnTheChildTable`.

### Master Bengkel

Menggantikan harness Pega `BengkelHE` (MENU_ID 28) atas tabel `POOLDATA.BENGKEL_HE`, **40 kolom**.

Ia master **komersial**, bukan sekadar daftar alamat bengkel: diskon jasa, diskon sparepart,
persen material, PPN, dan jenis PPh di dalamnya menentukan hasil hitungan uang pada klaim yang
memakai bengkel itu.

**Tiga tab**, caption **dan urutannya** mengikuti layar Pega — APPROVE · REJECT · WAITING
APPROVAL — dan ketiganya membaca report definition yang sama, berbeda hanya pada nilai
`APPROVAL`-nya.

**Gridnya enam kolom + Ubah**, bukan empat puluh. `Section/BrowseMasterHEApprove-Section.xml`
menyetel `pyColumnCount = 7`, dan urutan kolomnya:

| Kolom grid | Kolom tabel | Catatan |
|---|---|---|
| ID Bengkel | `ID_BENGKEL` | urutan bawaan **menurun** (`pySortType=DESC`, `pySortOrder=1`) |
| Nama Bengkel | `NAMA_BENGKEL` | |
| Alamat Bengkel | `ALM_BENGKEL` | |
| Telp Bengkel | `TELP_BENGKEL` | |
| No HP Bengkel | `NOHP_BENGKEL` | |
| Login Aplikasi | `LOGIN_APLIKASI` | bengkel **rekanan** yang kosong ditandai "rekanan tanpa login" |
| Ubah | — | membuka form 33 isian |

Ketiga puluh empat kolom selebihnya ada di tabel tetapi **tidak** di grid; seluruhnya hanya
muncul di form — sama seperti di Pega.

**Paginasi 20 baris per halaman**, angkanya dibaca dari `pyPageSizeOther` pada section yang sama
(`pyPageSize` bernilai `"Other"`). Layar master lain memakai 15, sehingga angka itu milik layar —
bukan bawaan `DataTable`.

Dua tombol yang ada di layar Pega **tidak** dibawa: "Upload Document" dan "Upload Data Master
Bengkel". Rule di baliknya tidak ikut di export (`R-16`), dan tombol yang tidak melakukan apa pun
lebih buruk daripada tombol yang tidak ada.

#### Satu hal yang WAJIB dipastikan DBA sebelum modul ini menulis di produksi

Sistem lama memakai **dua tabel untuk satu master**, dan itu pola dua-penyimpanan yang `R-10`
catat:

```
tulis   UpdateBengkelHE_act  →  @GCNM.GetPageJSONString()
        PEGA_M_BENGKEL_HE    →  INSERT INTO POOLDATA.M_BENGKEL_HE(ID, JSONDATA)
baca    BrowseBengkelHE_RD   →  POOLDATA.BENGKEL_HE, 40 kolom
```

Modul ini **menulis kolom bernama ke `BENGKEL_HE`** — bukan dokumen JSON ke `M_BENGKEL_HE`.
Alasannya bukan selera: badan fungsi `@GCNM.GetPageJSONString()` **tidak ikut di export** (`R-16`),
sehingga setiap nama kunci JSON yang ditulis akan menjadi tebakan — pada master yang menentukan
diskon, pajak, dan rekening tujuan pembayaran. Keempat puluh kolom `BENGKEL_HE` sebaliknya terbaca
lengkap. Perlakuannya sama dengan Master Status Klaim, yang menghadapi keluarga procedure
`PEGA_M_*` yang sama dan memutuskan hal yang sama (`D-02`, `D-68`).

Yang belum dapat dipastikan dari export adalah **apakah `BENGKEL_HE` sebuah tabel atau sebuah view
atas `M_BENGKEL_HE.JSONDATA`** — DDL-nya tidak ada (`R-08`), dan preseden bentuk kedua ada
(`V_STS_CLAIM`). `claimpnc -periksa` melaporkan keduanya beserta perbandingan jumlah barisnya;
jangan aktifkan jalur tulis di produksi sebelum DBA memastikannya.

#### Dua hal yang TIDAK dibawa, dan keduanya disengaja

| Perilaku Pega | Kenapa tidak dibawa |
|---|---|
| Menyetujui bengkel rekanan **membuat operator Pega** (`GCNMCreateOperator`, access group `GKM:InboxWorkshop`) dengan **kata sandi yang sama untuk setiap bengkel** | Sistem baru tidak punya operator Pega, dan kontrak identitasnya belum ada (`F-3`, `R-14`). Mereplikasinya berarti menerbitkan akun dengan kata sandi yang sudah diketahui siapa pun yang pernah membaca rule itu. `LOGIN_APLIKASI` tetap disimpan dan diperiksa keunikannya, dan layar **menyatakan** bahwa akunnya belum diterbitkan |
| Penyimpanan mengirim surel ke **satu alamat pribadi** yang tertanam di rule | `D-67` melarangnya, dan seam Notifier (`S-3`) belum ada. Peristiwanya dicatat di log supaya ketiadaannya terlihat, bukan tersamar |

#### Approve dan Reject ada di layar ini, padahal di Pega tidak

Di Pega keduanya ada di **Inbox Manager** (`Section/ApprovalMasterBengkelHE`), dijalankan
`Activity/SetApprovalAllMaster` yang melayani bengkel, panel, dan sparepart sekaligus — dan rule
SQL-nya sendiri **hilang dari export** (`R-16`).

Inbox Manager belum dibangun. Menunda keputusannya berarti setiap bengkel yang ditambah tertahan
di Waiting Approval tanpa satu pun cara menyelesaikannya. Yang dipakai adalah **bentuk yang sama
persis**: centang beberapa baris, satu tombol untuk seluruh pilihan. Memindahkannya ke Inbox
Manager kelak hanya soal letak.

#### Nilai sah tujuh penanda status tidak diketahui

`STS_SUPPLY`, `STS_EKLAIM`, `STS_AUTO_AKSEP`, `STS_PAYMENT`, `STS_AUTOPAYMENT`, `STS_TEKNO`, dan
`STS_ORDER` dirender radio button atau dropdown di Pega, dan daftar pilihannya ada di rule Field
Value yang **tidak ikut di export**. Pencarian atas seluruh export menghasilkan **nol**
perbandingan terhadap ketujuhnya.

Ketujuhnya karena itu **tidak** dibuat dropdown — mengarang dua pilihan "Ya/Tidak" berarti menebak
domain kolom yang menentukan kanal mana yang boleh dipakai bengkel. Yang ditawarkan layar adalah
**nilai yang sudah dipakai baris lain pada entitas itu**: saran yang berasal dari data, bukan dari
tebakan.

Satu-satunya yang nilainya **diketahui** adalah `STATUS_REKANAN`, dan itu pun terbaca dari
percabangan alih-alih dari label: `ValidationLoginBengkel_act` melompat keluar pada
`Local.STS_REKANAN=='0'`, dan langkah yang dilewatinya adalah pembuatan login. Karena itu ia satu-
satunya yang dibuat dropdown — dan ia yang menentukan wajib-tidaknya isian Login aplikasi.

### Master Pasal Kerugian

Menggantikan harness Pega `DetailMasterPasalRejected` (MENU_ID 27) — judulnya di layar
**"Detail Pasal Kerugian"**. Nama harness-nya menyebut *Rejected*, tetapi yang dikelolanya bukan
penolakan melainkan **butir ketentuan polis** yang dirujuk saat klaim dinilai: jaminan,
pengecualian, atau notifikasi.

**Bentuk penyimpanannya tidak biasa, dan itu menyetir seluruh modul.** Tabelnya hanya punya
**tiga kolom** — `IDDATA`, `IDPASAL`, `JSONPASAL` — dan seluruh isi selain No Pasal hidup di dalam
satu dokumen JSON pada kolom ketiga. Dokumen itu dihasilkan `stepPage.getJSON(false)`
(`Function/GetPageJSONString-Function.xml`), sehingga kuncinya adalah nama properti page klipboard
apa adanya.

| Layar | Kunci dokumen | Catatan |
|---|---|---|
| No Pasal | *(kolom `IDPASAL`)* | **tidak dijamin unik**; kuncinya `IDDATA` |
| Isi Pasal | `$.DESCRIPTION` | di grid berlabel "ISI PASAL" |
| Deskripsi | `$.OLD_D_COL_ID` | pasangannya dengan Isi Pasal memang **terbalik** dari dugaan wajar |
| Kategori | `$.pyCountry` + `$.LOSS_CODE` | sebutannya diturunkan server dari kodenya |
| Bisnis | `$.BISNISID[]` | daftar lini bisnis; kosong pada hasil daftar |

> ### ⚠ Tombol Hapus MENGHAPUS PERMANEN, dan itu menyupersede `D-66`
>
> `D-66` menetapkan **soft delete menyeluruh** — tidak ada `DELETE` fisik pada data bernilai
> bisnis. Tabel ini tidak punya kolom penanda terhapus, dan menambahnya menempuh `D-63`
> (permintaan tertulis → persetujuan Work Owner → pelaksanaan DBA). Tiga jalan keluar diajukan
> ke Work Owner pada 2026-09-19 — penanda di dalam JSON, meminta kolom baru, atau menunda
> tombolnya — dan jawabannya **"jalankan as is"**, yaitu `DELETE` fisik seperti Pega
> (`RDB List/DeleteDataPasalDataMaster-SQL.xml`).
>
> Konsekuensinya: **barisnya tidak dapat dipulihkan, dan tidak meninggalkan jejak apa pun** —
> tabelnya juga tidak punya kolom pencatat siapa dan kapan. Layar menambahkan **konfirmasi**
> sebelum menghapus, yang di Pega tidak ada; itu bukan perubahan aturan bisnis, hanya penghalang
> terhadap satu klik yang tidak disengaja.

**Satu-satunya isian wajib adalah No Pasal**, dan itu memang seluruh pemeriksaan layar lama
(`Activity/CNMInsertPasalDataMaster-Act.xml`: *"Silahkan ISI No Pasal Terlebih Dahulu"*). Isi Pasal,
Deskripsi, Kategori, dan Bisnis boleh kosong; No Pasal boleh kembar; dan nama lini bisnis boleh
**diketik bebas** di luar master — keempatnya keputusan Work Owner 2026-09-19, *"jalankan as is"*.
Butir tanpa kode **ditandai di layar**, karena di Pega ia tampak sama persis dengan yang dipilih
dari master.

> **Tiga selisih yang direncanakan terhadap Pega**, dinyatakan di muka supaya tidak ditemukan
> sebagai kejutan pada uji kesetaraan: daftar diurutkan `IDPASAL, IDDATA` sementara kueri lama
> tidak punya `ORDER BY` sama sekali · nomor `IDDATA` dihitung di Go, sehingga tabel kosong
> menghasilkan `1` alih-alih kunci kosong (`PEGA_D_PASAL_MASTER.prc:11` memakai `max()` **tanpa**
> `NVL`) · `PUT` atas baris yang sudah tidak ada dijawab `404`, bukan diam-diam menyisipkan baris
> baru seperti cabang `ELSE` procedure lama.

> **Batas yang diwarisi.** Daftar pilihan Kategori **tidak ada di export** (`R-16`) — yang terbaca
> hanya ekspresi `1 → "Jaminan Polis"`, `2 → "Pengecualian"`, `else → "Notifikasi"`. Kode
> "Notifikasi" karena itu diperlakukan **kosong**, bukan `"3"` yang akan menjadi tebakan. Baris lama
> berkode lain tetap terbaca "Notifikasi" dan **kodenya tidak ditimpa** selama pengguna tidak
> menyentuh dropdown-nya. Mode `-periksa` melaporkan bila kode semacam itu benar-benar ada di data.

### Master Auto Claim

Menggantikan harness Pega `AutoKlaim` beserta keempat section-nya. Ia **bukan** master klaim:
isinya daftar **Sumber Bisnis yang klaimnya boleh dibuat otomatis**, beserta ke mana ganti
ruginya dibayarkan. `INISIALID` dicocokkan dengan `T_GENERAL.SOURCEOFBUSINESS` milik polis saat
klaim otomatis dibuat (`RDB List/GetReceiverClaimAsuransiKredit-SQL.xml`).

**Menyimpan dan memutuskan adalah satu operasi**, dan itu bentuk sistem lama:
`Activity/UpdateMstAutoClaim_act` melayani tombol Update, Approve, dan Reject sekaligus,
dibedakan hanya oleh parameter `stsapprove`. Akibatnya **menyunting baris yang sudah disetujui
mengembalikannya ke antrean persetujuan** — persetujuan lama tidak berlaku atas isi yang sudah
berubah.

Tiga nilai tidak pernah diterima dari klien: `APPROVAL` (baris baru selalu `"0"`),
`CLAIM_ALLOWED` (selalu `"1"`, keputusan Work Owner 2026-09-19), dan `KOMITE` (dibaca dari
`POOLDATA.EMAILKOMITE`). **`NAMA_PENERIMA` tidak dapat diubah** setelah baris dibuat — kueri
`UPDATE` sistem lama pun tidak menyebut kolomnya.

> **Tiga selisih yang direncanakan terhadap Pega**, dinyatakan di muka supaya tidak ditemukan
> sebagai kejutan pada uji kesetaraan: `CLAIM_ALLOWED` tetap `"1"` juga pada jalur ubah (sistem
> lama menulis apa pun yang diketik, sehingga satu baris dapat "mati" tanpa pesan) · `PCT_MAX`
> yang bukan angka `0`–`100` ditolak · daftar diurutkan `INISIALID`, sementara kueri lama tidak
> punya `ORDER BY` sama sekali.

Tidak ada `DELETE` pada master status progres, dan itu disengaja: sistem lama tidak punya
satu pun pernyataan `DELETE` terhadap tabel itu, dan tabelnya tidak punya kolom penanda
terhapus yang dapat dipakai `D-66`. Alasan lengkapnya di
[`docs/keputusan-implementasi.md`](docs/keputusan-implementasi.md) §10.4.

### Master Penolakan Klaim

Menggantikan harness Pega `PNC_MasterTolakKlaim` (MENU_ID 25). **Satu layar, dua master yang
tabelnya tidak berhubungan** — persis seperti layar lama, yang berpindah isi lewat dua tombol:

| Tab | Tabel | Isi |
|---|---|---|
| **Penolakan Klaim** | `MST_PENOLAKAN_KLAIM_1` + `_2` | alasan penolakan berjenjang dua tingkat, beserta keadaan persetujuannya |
| **Penolakan Komite** | `MST_REJECTED_KOMITE` | catatan penolakan komite; dua kolom saja |

**Satu cacat sistem lama diperbaiki, atas keputusan Work Owner 2026-09-19.**
`Database/MASTERPENOLAKANKLAIM1.prc` melakukan `INSERT` pada **kedua** cabang `IF`-nya
(`:9` dan `:14`), tidak pernah `UPDATE` — sehingga setiap simpan, termasuk setiap pengubahan,
menerbitkan satu Status Penolakan 1 baru dan meninggalkan yang lama menggantung. Di sini induk
**dipilih dari daftar**, dan baris baru lahir hanya bila pengguna memang memintanya. Ini satu-satunya
penyimpangan dari `P-5` pada modul ini; rinciannya di
[`docs/keputusan-implementasi.md`](docs/keputusan-implementasi.md) §18.2.

**Tiga perilaku lain direplikasi apa adanya**, termasuk yang tampak aneh:

| Perilaku | Bukti |
|---|---|
| Mengubah teks **mengembalikan baris ke antrean persetujuan** — status kembali `MENUNGGU` | `MASTERPENOLAKANKLAIM2.prc:14` |
| Jejak penyetuju sebelumnya **tidak dibersihkan** saat itu terjadi | idem — hanya enam kolom yang di-`SET` |
| Baris pertama pada `MST_REJECTED_KOMITE` yang kosong bernomor **111** | `INSERTMASTERREJECTEDKOMITE.prc:9` |

> **Persetujuan tidak dikerjakan di layar ini.** Kolom Tanggal Approve, Approve By, Status Aproval,
> dan Note Approval hanya **ditampilkan**. Yang mengisinya adalah layar **Inbox Manager**
> (MENU_ID 58, `UserInbox_Harness` lewat `Sec_PenolakanKlaimChecker`) — modul tersendiri yang
> belum dibangun.

**Tidak ada `DELETE`** pada satu tab pun: seluruh export tidak memuat satu pun pernyataan `DELETE`
terhadap ketiga tabelnya, layar lama tidak punya tombolnya, dan tidak satu pun tabelnya punya kolom
penanda terhapus yang dapat dipakai `D-66`.

> **Batas yang diwarisi.** DDL ketiga tabel belum ada (`R-08`), sehingga panjang maksimum 100
> karakter adalah **asumsi** — bukan panjang kolom yang diterima DBA. Isi tabelnya pun tidak ikut
> dikirim bersama export, sehingga data contoh untuk pengembangan adalah susunan sendiri dan tidak
> boleh dipakai sebagai dasar uji kesetaraan gerbang 1.

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
| `/master/status-progres-2` | **Master Status Progres 2** |
| `/master/status-klaim` | **Master Status Klaim** |
| `/master-rekening` | **Master Rekening** |
| `/master/penolakan-klaim` | **Master Penolakan Klaim** — dua tab: Penolakan Klaim dan Penolakan Komite |
| `/master/auto-claim` | **Master Auto Claim** — empat tab: Master Auto Klaim, Komite Approval, Waiting Approval, Reject |
| `/master/pasal-kerugian` | **Master Pasal Kerugian** — satu-satunya layar yang **menghapus permanen** |
| `/master/bengkel` | **Master Bengkel HE** — tiga tab: Approve, Reject, Waiting Approval; grid 6 kolom, 20 baris per halaman; keputusan **borongan** dengan centang |
| `/master/panel` | **Master Panel HE** — tiga tab (Approve · Reject · Waiting Approval); satu-satunya layar master yang mengelola **baris anak** (daftar lokasi per panel) |
| `/master/sparepart` | **Master Sparepart HE** — tiga tab (Approve · Reject · Waiting Approval); grid **5 kolom**, 30 baris per halaman; satu-satunya layar master yang mencatat **pelaku dan waktu** di tabelnya sendiri |
| `/master/grouping-sparepart` | **Master Grouping Sparepart HE** — menu 32; tiga tab (Approve · Reject · Waiting Approval); grid **6 kolom + nomor grup**, 15 baris per halaman. Satu-satunya layar master yang memakai **dua tabel**, yang kunci alaminya **empat kolom bersama-sama**, dan yang lima isiannya **diturunkan** dari master lain |
| `/master/kategori-sparepart` | **Master Kategori Sparepart** — menu 33; tiga tab (Approve · Reject · Waiting Approval); grid **2 kolom**, 50 baris per halaman; master **terkecil** — tabelnya hanya tiga kolom, dan ia MENULIS tabel yang layar Master Sparepart hanya baca |
| `/master/tipe-sparepart` | **Master Tipe Sparepart** — menu 34; tiga tab (Approve · Reject · Waiting Approval); grid **4 kolom**, 50 baris per halaman. Master pertama di rumpun sparepart yang menyimpan **kunci asing** — setiap tipe berinduk pada satu kategori. Satu-satunya layar yang **sengaja menampilkan lebih banyak baris daripada Pega**: inner join Pega diganti LEFT JOIN supaya baris tanpa kategori tetap dapat diperbaiki |
| `/pelaporan-klaim` | **Pelaporan Klaim** — menu 64 |
| `/inbox/investigator` | **Inbox Investigator** — menu 48; antrean bersama workbasket `InvestigatorPNC`. **Layar INBOX pertama**, dan yang pertama di bawah awalan `/inbox/`. Grid **9 kolom**, 50 baris per halaman (`pyPageSize` grid lamanya). Baca-saja: mengambil pekerjaan dan mencatat hasil investigasi ada di layar kerja yang belum dibangun. **Export Data Investigation TIDAK dibawa** — dihapus atas keputusan Work Owner 2026-09-24; pemetaan 3 dari 13 kolom CSV-nya tidak dapat ditelusuri (lihat `docs/permintaan-artefak-pega.md` §2) |
| `/inbox/receive-tka` | **Inbox Receive TKA** — menu 49; klaim TKA yang tanggal penerimaan dokumen aslinya belum diisi. **Layar INBOX kedua, dan yang PERTAMA menulis**: pengguna mengisi tanggal di dalam tabel lalu menekan **Submit per baris**, dan barisnya hilang dari daftar. Grid **7 kolom + kolom aksi**, 50 baris per halaman. Dibaca dari **tabel yang sama dengan Report Definition Pega** — `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` dengan `TKA_1 = '1'`, tanggal kelengkapan masih kosong, dan status kerja belum selesai; nama peserta diambil dari `T_GENERAL` karena kolomnya pada tabel kerja kosong. Submit menulis **satu kolom saja** (`T_CLAIM_PNC.TGLDOKLENGKAP`) lalu melepaskan pemberitahuan **di luar** transaksi — surel yang gagal tidak membatalkan penyimpanan. **Tabel engine Pega tidak pernah ditulis**: nilainya akan tertimpa tanpa jejak saat Pega menyimpan kasus itu lagi, sehingga selama masa paralel layar Pega masih menampilkan klaim itu sebagai belum lengkap — selisih yang dicacah `claimpnc -periksa` dan disebutkan di kaki layar. Baris yang klaimnya tidak ditemukan **tetap tampil** dengan isian dimatikan (lihat `docs/permintaan-artefak-pega.md` §3) |
| `/riwayat-klaim` | **View History Claim** — menu 76, pencarian riwayat klaim |
| `/pelaporan-klaim` | **Pelaporan Klaim** — menu 64 |
| `/riwayat-klaim` | **View History Claim** — menu 76, pencarian riwayat klaim |

Seluruhnya dapat dicapai lewat **menu utama** di kerangka aplikasi — kolom samping di layar
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
| `GET` | `/api/inbox/investigator` | wajib | antrean Inbox Investigator; saringan `cari`; terpotong pada 500 baris dan **menyatakannya** lewat `terpotong` |
| `GET` | `/api/inbox/receive-tka` | wajib | daftar Inbox Receive TKA; saringan `cari`; terpotong pada 500 baris dan **menyatakannya** lewat `terpotong` |
| `POST` | `/api/inbox/receive-tka/kelengkapan-dokumen` | wajib | `{nomor_klaim, tanggal_dokumen_lengkap}` — mengisi **satu kolom**, `T_CLAIM_PNC.TGLDOKLENGKAP`; `422` bila tanggal kosong, `409` bila barisnya sudah diisi orang lain / klaimnya tidak ditemukan / nomor klaimnya ganda. **Tidak idempoten**: permintaan kedua ditolak, sehingga surel ganda tidak dapat terjadi |
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
