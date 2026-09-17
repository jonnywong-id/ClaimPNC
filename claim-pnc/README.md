# Claim PNC — Aplikasi Go + React

Implementasi pengganti aplikasi Pega PRPC 8.3 **Claim PNC**. Dokumen migrasi, ADR, dan papan
tiket pekerjaan berada di repository terpisah: `D:\Jonny\Project\Claude.AI\XML Claim PNC\docs`.

**Yang sudah ada di tahap ini: login dari ujung ke ujung, dan satu modul bisnis — Master Status
Klaim.**

| | |
|---|---|
| Tiket yang dikerjakan | `TKT-F3-001` seam identitas · `TKT-F3-003` sesi & token · `TKT-U1-002` alur masuk di frontend · **`TKT-F4-005` bagian Master Status Klaim** |
| Tiket yang disentuh sebagian | `TKT-F1-001` struktur & aturan lapisan · `TKT-F1-002` konfigurasi · `TKT-F1-003` logging · `TKT-F2-001` koneksi & seam repository · **`TKT-U2-001` komponen tabel baku** · **`TKT-F4-001` pola master data** |
| Tiket yang **belum** dikerjakan | `TKT-F3-002` provider HCC/HCQ · `TKT-F3-004` tabel peran & izin menu · `TKT-F3-005` middleware otorisasi · `TKT-U1-001` kerangka portal · `TKT-U2-005` pemilihan pustaka tabel |

> **Master Status Klaim belum dapat dipakai terhadap Oracle.** Kolom `LSC_NOTE` pada
> `POOLDATA.M_STS_CLAIM` masih kosong di seluruh 32 baris sampai
> [`migrations/0002`](backend/migrations/0002_master_status_klaim.up.sql) dijalankan DBA. Terhadap
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
│   │   │   ├── repo/                    pengisi seam penyimpanan — sqlstore, memori
│   │   │   └── http/                    handler, dto, middleware sesi, rute modul
│   │   ├── portal/                  MODUL — entitas & basis datanya (ADR-0030)
│   │   │   ├── repo/                    sqlstore (M_PORTAL_PNC), memori
│   │   │   └── http/                    rute daftar portal
│   │   ├── masterstatus/            MODUL — Master Status Klaim (F-4)
│   │   │   ├── usecase/                 orkestrasi: daftar, ambil, tambah, ubah
│   │   │   ├── repo/                    sqlstore (M_STS_CLAIM), memori + 33 baris contoh
│   │   │   └── http/                    dto, galat, handler, rute
│   │   └── platform/                config, logging, db, middleware, waktu, httpserver
│   ├── migrations/                  DDL untuk dijalankan DBA
│   ├── spa/                         penyematan hasil build antarmuka ke binary
│   └── go.mod
├── frontend/                    SPA React + TypeScript + Vite
│   └── src/
│       ├── app/                     kerangka: router, provider, penjaga rute, sesi
│       ├── modules/                 satu folder per modul — masuk, portal, beranda
│       ├── components/              pustaka komponen baku
│       └── api/                     klien HTTP dan tipe kontrak API
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
6. **Nilai desain yang berulang tinggal di `src/gaya.css`**, bukan diketik ulang per layar —
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
[`backend/internal/auth/provider/tiruan.go`](backend/internal/auth/provider/tiruan.go):

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
backend/migrations/0001_pengguna_dan_sesi.up.sql       tabel BARU: CPNC_PENGGUNA, CPNC_SESI_AKTIF
backend/migrations/0001_pengguna_dan_sesi.down.sql
backend/migrations/0002_master_status_klaim.up.sql     MENGUBAH objek milik sistem lama
backend/migrations/0002_master_status_klaim.down.sql
```

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
cd frontend && npm run periksa-tipe && npm test
```

Seluruh uji berjalan **tanpa basis data dan tanpa jaringan**: provider identitas tiruan dan
penyimpanan di memori keduanya hidup di dalam proses.

---

## Kontrak API

| Metode | Jalur | Sesi | Keterangan |
|---|---|---|---|
| `POST` | `/api/masuk` | — | `{nama_pengguna, kata_sandi}` → token + profil |
| `POST` | `/api/keluar` | opsional | mencabut sesi di server |
| `GET` | `/api/saya` | wajib | identitas pemanggil + batas berlaku sesi |
| `POST` | `/api/sesi/perpanjang` | wajib | menggeser batas berlaku |
| `GET` | `/api/portal` | wajib | daftar entitas dari `POOLDATA.M_PORTAL_PNC` + portal utama |
| `GET` | `/api/master/status-klaim` | wajib | daftar status klaim + `total` |
| `POST` | `/api/master/status-klaim` | wajib | `{label}` → `201` + baris beserta kode yang dibuat sistem |
| `GET` | `/api/master/status-klaim/{kode}` | wajib | satu status klaim |
| `PUT` | `/api/master/status-klaim/{kode}` | wajib | `{label}` → `200` + baris setelah diubah |

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
