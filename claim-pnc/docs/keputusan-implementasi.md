# Keputusan Implementasi — Tahap Login

Tanggal: 2026-09-15
Lingkup: `TKT-F3-001`, `TKT-F3-003`, `TKT-U1-002`, sebagian `TKT-F1-001`/`002`/`003` dan `TKT-F2-001`

Berkas ini mencatat apa yang diputuskan saat menulis kode, apa yang **menyimpang** dari dokumen
Steering, dan apa yang **sengaja ditinggalkan** — supaya tidak ada yang perlu menebak belakangan
mengapa sesuatu ditulis begitu.

Nomor `D-nn`, `ADR-nnnn`, dan `TKT-*` merujuk ke repository dokumen migrasi
(`D:\Jonny\Project\Claude.AI\XML Claim PNC\docs`).

---

## 1. Keputusan yang diambil Work Owner pada sesi ini

Empat pertanyaan diajukan sebelum baris kode pertama ditulis. Jawabannya mengikat berkas ini.

| # | Pertanyaan | Jawaban | Akibatnya pada kode |
|---|---|---|---|
| 1 | Di mana kode dibuat? | `D:\app\claim-pnc` | Export XML Pega tidak tersentuh sama sekali; rujukan ke dokumen tiket menjadi lintas folder |
| 2 | Seberapa jauh cakupan tahap ini? | **Login end-to-end, fondasi seadanya** | Struktur berlapis tetap dipakai; `depguard`, `golangci-lint`, health check, dan graceful shutdown **dilewati** — lihat §4 |
| 3 | Basis data untuk pengembangan? | **Oracle 19c dev** | Adapter SQL dan migrasi ditulis untuk Oracle; kredensialnya belum diberikan sehingga jalur itu **belum pernah dijalankan** — lihat §5 |
| 4 | Penyimpanan token di peramban? | **Bearer di header `Authorization`** | Menyimpang dari `11-SECURITY.md` §2.2 yang menganjurkan cookie `HttpOnly` — lihat §3.1 |

---

## 2. Keputusan desain

### 2.1 Token opaque, bukan JWT

`11-SECURITY.md` §2.2 membolehkan keduanya: "JWT bertanda tangan, atau token opaque + penyimpanan
session". Yang dipilih **token opaque**: 32 byte acak dari `crypto/rand`, disandikan base64url.

Alasannya satu dan menentukan: **pencabutan harus berlaku seketika** (`TKT-F3-003`). JWT yang
memverifikasi dirinya sendiri tetap sah sampai kedaluwarsa, sehingga pencabutan menuntut daftar
cabutan di server — yang artinya tetap membaca basis data setiap permintaan, persis seperti token
opaque, tetapi dengan tambahan kerumitan penandatanganan dan rotasi kunci.

### 2.2 Yang disimpan adalah sidik token, bukan tokennya

Tabel `CPNC_SESI_AKTIF` menyimpan `SIDIK_TOKEN` = SHA-256 token dalam heksadesimal. Token
mentahnya tidak pernah menyentuh basis data.

Akibatnya: bocornya isi tabel sesi — lewat backup, export, atau kueri DBA — **tidak** dengan
sendirinya memberi orang lain sesi yang dapat dipakai.

### 2.3 Pengenal sesi acak dan berdiri sendiri

`CPNC_SESI_AKTIF.ID` tidak diturunkan dari token maupun sidiknya. Pengenal sesi akan muncul di
jejak audit dan layar administrasi; tidak satu pun dari keduanya boleh menjadi petunjuk menuju
token yang masih hidup.

### 2.4 Izin tidak ikut di dalam sesi

Sesi hanya memuat NIK, pengenal, dan batas berlaku. Status aktif dan izin dibaca dari basis data
pada setiap permintaan.

Alasannya ada di `11-SECURITY.md` §2.2: izin dapat berubah kapan saja lewat layar master data, dan
izin yang tertanam di sesi baru berlaku setelah sesi berakhir. Konsekuensinya sudah terbukti di
uji: pengguna yang dinonaktifkan setelah masuk kehilangan akses pada permintaan **berikutnya**,
bukan satu jam kemudian.

### 2.5 Tiga jenis galat autentikasi dibedakan; keberadaan akun tidak

Pemanggil membedakan ketiganya lewat `errors.Is` di Go dan lewat field `kode` di JSON — tidak
pernah dengan mencocokkan teks pesan.

| Galat | HTTP | Tindak lanjut pengguna |
|---|---|---|
| kredensial salah | 401 | ketik ulang |
| pengguna tidak aktif | 403 | hubungi administrator; mengetik ulang tidak menolong |
| sistem identitas tidak dapat dihubungi | 503 | tunggu; mencoba berulang membanjiri sistem yang sedang bermasalah |

Yang **tidak** dibedakan: pengguna yang tidak ada versus kata sandi yang salah. Keduanya memakai
kode, status, dan teks yang sama persis — diuji di tiga tempat (adapter, HTTP, layar).

### 2.6 Profil tidak lengkap ditolak, bukan diteruskan

Sistem identitas yang menjawab dengan salah satu dari lima field kosong (NIK, nama, cabang,
jabatan, email) dianggap gagal. Tanpa aturan ini, pengguna dapat berhasil masuk lalu **tidak
dikenali oleh data klaimnya sendiri** — kegagalan yang jauh lebih mahal bila baru ketahuan
setelah ia mengisi satu form registrasi penuh.

### 2.7 Pencabutan sesi adalah penandaan, bukan penghapusan

`ADR-0012` menetapkan soft delete menyeluruh. `TKT-F3-003` menulis "keluar menghapus sesi dari
database". Keduanya didamaikan begini: kolom `DICABUT_PADA` diisi, barisnya tetap ada, dan token
lama ditolak sejak permintaan berikutnya. Yang dituntut tiket — token lama tidak lagi berlaku —
terpenuhi; jejaknya tetap dapat ditelusuri `S-5`.

### 2.8 Penamaan tabel `CPNC_`

Dua tabel baru: `CPNC_PENGGUNA` dan `CPNC_SESI_AKTIF`. Awalan `CPNC_` dipakai supaya tabel milik
aplikasi baru tidak pernah tertukar dengan tabel warisan berawalan `T_` di skema `POOLDATA`, dan
supaya aturan penulis tunggal per tabel (`ADR-0004`) terbaca dari namanya saja.

`OPERATOR_ID` disediakan tetapi **dibiarkan kosong dan nullable**: cara mencocokkan identitas
HCC/HCQ dengan `OPERATOR_ID` yang dipakai seluruh data klaim belum ditetapkan (`ADR-0024`,
pertanyaan terbuka nomor 4). Menebaknya akan menghasilkan pemetaan yang salah di seluruh data
klaim.

### 2.9 Upsert pengguna ditulis UPDATE-lalu-INSERT, bukan MERGE

`MERGE` pada Oracle menuntut `FROM DUAL`, dan `FROM DUAL` tidak ada di PostgreSQL. Disiplin SQL
portabel (`D-20`) lebih berharga daripada satu pernyataan yang lebih ringkas. Perlombaan dua
permintaan untuk NIK yang sama-sama baru ditangani dengan mencoba `UPDATE` sekali lagi setelah
`INSERT` kalah pada kunci utama.

---

## 3. Penyimpangan dari dokumen Steering

Ketiganya disengaja dan dicatat terbuka. Tidak ada yang lain.

### 3.1 Token di header `Authorization`, bukan cookie `HttpOnly`

**Dokumen:** `11-SECURITY.md` §2.2 menetapkan `Cookie HttpOnly + Secure + SameSite=Strict`, dengan
alasan tertulis bahwa token yang dapat dibaca JavaScript membuat satu kerentanan XSS langsung
berarti pencurian sesi. `04-FUTURE-ARCHITECTURE.md` §1 menggambarkan hal berbeda: "HTTPS · JSON ·
Bearer token". Kedua dokumen bertentangan.

**Yang dipakai:** Bearer di header, atas keputusan Work Owner 2026-09-15.

**Konsekuensi yang diterima:** token harus dapat dibaca JavaScript. Ia disimpan di
`sessionStorage` — bukan `localStorage` — supaya hilang saat tab ditutup dan tidak dibagi antar
tab. **Risiko XSS yang disebut `11-SECURITY.md` §2.2 tetap berlaku dan tidak dimitigasi oleh
pilihan ini.** Bila kelak ada temuan pentest soal ini, perubahannya menyentuh cara frontend
menyimpan dan mengirim token, bukan cara server memverifikasinya — sisi server memakai token
opaque yang sama apa pun wadahnya.

### 3.2 Driver Oracle `go-ora`, bukan `godror`

**Dokumen:** `08-TECHNICAL-STRATEGY.md` §1 menetapkan `godror`, "paling matang untuk Oracle".

**Kendalanya:** `godror` menuntut CGO dan Oracle Instant Client. Mesin pengembangan ini
`CGO_ENABLED=0` dan tidak punya kompilator C (`gcc: command not found`), sehingga `go build ./...`
tidak dapat diselesaikan sama sekali dengan `godror`.

**Yang dipakai:** `github.com/sijms/go-ora/v2`, driver Oracle murni Go.

**Dampak pertukarannya kecil:** keduanya berbicara `database/sql`, dan seluruh sentuhan driver
terkurung di satu berkas — [`backend/internal/platform/db/oracle.go`](../backend/internal/platform/db/oracle.go).
Bila Instant Client tersedia di mesin build nanti, mengembalikannya ke `godror` menyentuh berkas
itu saja.

**Yang belum terverifikasi:** `go-ora` belum pernah benar-benar menghubungi Oracle di proyek ini —
lihat §5.

### 3.3 Sakelar `PENYIMPANAN=memori`

Tidak disebut dokumen mana pun; ditambahkan supaya login dapat dijalankan dan diperlihatkan
sebelum kredensial Oracle dev tersedia.

Ia mengikuti pola yang sama dengan provider identitas tiruan: **penolakan terhadap produksi ada di
dalam kode, bukan pada nilai konfigurasi.** Alasannya bukan kerapian — sesi di memori satu instans
tidak akan dikenali instans kedua di belakang load balancer, dan itu pelanggaran langsung terhadap
tuntutan stateless `D-27`. Penolakan ini diuji di
[`backend/cmd/claimpnc/main_test.go`](../backend/cmd/claimpnc/main_test.go).

---

## 4. Yang sengaja tidak dikerjakan

Konsekuensi langsung dari pilihan "fondasi seadanya" (§1 nomor 2). Semuanya adalah lingkup tiket
yang sudah ada, bukan penemuan baru.

| Yang dilewati | Tiket | Akibat bila dibiarkan |
|---|---|---|
| `depguard` + `golangci-lint` | `TKT-F1-001` | **Aturan lapisan hanya dijaga kesepakatan.** Satu `import "database/sql"` di `domain/` akan lolos tanpa ada yang menyadarinya — persis mode kegagalan yang dikhawatirkan `D-09` |
| Health check + graceful shutdown | `TKT-F1-005` | Dua instans tidak dapat di-update bergantian tanpa memutus permintaan yang sedang berjalan (`D-27`) |
| Pool laporan terpisah | `TKT-F2-001` | Belum menggigit: belum ada laporan |
| Kepemilikan transaksi di lapisan aplikasi | `TKT-F2-003` | Pembaruan catatan pengguna dan penyimpanan sesi belum satu transaksi. Dampaknya terbatas dan idempoten — dicatat di komentar `segarkanPengguna` |
| Kontrak galat API yang mengikat | `TKT-F1-004` | Bentuk `{kode, pesan}` yang dipakai sekarang **sementara**; tiketnya terhalang keputusan Work Owner soal kegagalan senyap 720 activity |
| `CONTRIBUTING.md` aturan penamaan | `TKT-F1-001` | Diringkas di `README.md` bagian terakhir sebagai penambal sementara |
| Kerangka portal, navigasi, peta rute | `TKT-U1-001`, `TKT-U1-004` | Beranda sekarang hanya membuktikan sesi dikenali; menu belum ada |

---

## 5. Yang belum dapat dibuktikan

Ditulis terbuka karena `TKT-F3-003` menuntutnya dan menyatakannya lulus tanpa bukti akan menyesatkan
gerbang penerimaan.

| Acceptance criteria | Keadaan | Apa yang menahannya |
|---|---|---|
| "Sesi tersimpan di basis data; **dua instans aplikasi mengenali sesi yang sama**" | **Belum terbukti terhadap Oracle.** Yang terbukti: dua layanan yang berbagi satu penyimpanan saling mengenali sesi, termasuk pencabutannya (`TestSesiDikenaliInstansLain`) | Kredensial Oracle dev belum ada. Adapter SQL dan migrasinya sudah ditulis tetapi **belum pernah dijalankan** |
| "Mencabut sesi membuat permintaan berikutnya ditolak seketika" | Terbukti terhadap penyimpanan memori; jalur SQL-nya belum dijalankan | sama |
| Kueri SQL benar terhadap Oracle | **Belum diuji sama sekali.** Yang diuji baru disiplinnya: tanpa `SELECT *`, `NVL`, `SYSDATE`, `ROWNUM`, `TO_CHAR`, `FROM DUAL`, dan seluruhnya memakai parameter binding | sama |
| Migrasi `up` lalu `down` mengembalikan skema semula | Belum dijalankan | `D-63` menuntut permintaan tertulis + persetujuan Work Owner + pelaksanaan DBA |

**Yang dibutuhkan untuk menutup keempatnya:** host, service name, dan akun Oracle dev; ditambah
persetujuan menjalankan `backend/migrations/0001`.

---

## 6. Dependency yang ditambahkan

Seluruhnya sesuai `08-TECHNICAL-STRATEGY.md` §1 kecuali yang ditandai.

**Backend:** `go-chi/chi/v5` (router) · `sijms/go-ora/v2` (driver Oracle — **menyimpang**, §3.2) ·
`stretchr/testify` (uji). Selebihnya pustaka standar: `log/slog`, `database/sql`, `crypto/rand`,
`crypto/sha256`, `embed`.

**Frontend:** `react` 18+, `react-dom`, `react-router-dom`, `@tanstack/react-query`,
`react-hook-form`, `zod`, `@hookform/resolvers`, `zustand`, `tailwindcss`. Perkakas: `vite`,
`typescript`, `vitest`, `@testing-library/react`, `jsdom`, `@types/node`.

**Belum dipakai** karena belum ada layarnya: TanStack Table / AG Grid (`TKT-U2-005`, pilihannya
sendiri masih terbuka di `ADR-0002`).

---

## 7. Pertanyaan terbuka yang menahan tahap berikutnya

Tidak satu pun dapat dijawab dari kode atau dokumen yang ada. Seluruhnya sudah tercatat di ADR;
diulang di sini karena masing-masing menahan pekerjaan yang sudah di depan mata.

| Pertanyaan | Pemilik | Menahan |
|---|---|---|
| Apakah API HCC/HCQ sudah ada, dan bagaimana bentuk kontraknya? | Tim HCC/HCQ | `TKT-F3-002` — dan login sungguhan seluruh aplikasi |
| Apa yang terjadi bila HCC/HCQ tidak dapat dihubungi? Ada jalur cadangan? | Work Owner | Perilaku 503 di layar masuk; menyentuh tuntutan 24/7 `D-27` |
| Bagaimana identitas HCC/HCQ dicocokkan dengan `OPERATOR_ID`? | Work Owner + Tim HCC/HCQ | Pengisian `CPNC_PENGGUNA.OPERATOR_ID`; tanpa ini pengguna yang berhasil masuk tetap tidak dikenali data klaimnya |
| Berapa masa berlaku sesi yang final? | Work Owner + Security | Nilai 60m dipakai sementara dan dapat diubah lewat konfigurasi tanpa menyentuh kode |
| Dari mana daftar operator per peran diperoleh? | Work Owner + DBA | `TKT-F3-004` — tabel peran dapat dibangun tetapi tidak dapat diisi |
| Kredensial Oracle dev | DBA | Seluruh §5 |

---

## 8. Restrukturisasi menjadi backend/frontend terpisah — 2026-09-15 (sesi kedua)

Keputusan Work Owner: susunan repository mengikuti aplikasi **ClaimQ**, dengan backend dan
frontend terpisah penuh.

### 8.1 Yang berubah

| Sebelum | Sesudah |
|---|---|
| `cmd/`, `internal/`, `migrations/`, `web/`, `go.mod` di root | seluruhnya di bawah `backend/` |
| `web/` memuat kode Go **dan** sumber React | `backend/spa/` hanya penyematan; sumber React pindah ke `frontend/` |
| `cmd/server/` | `cmd/claimpnc/` — satu entrypoint bernama aplikasinya |
| `internal/{domain,app,adapter,transport}/` — **layer-first** | `internal/auth/` + `internal/platform/` — **module-first** |
| `internal/platform/database/` | `internal/platform/db/` |
| `internal/domain/waktu` + `internal/adapter/jam` | `internal/platform/waktu` — seam dan pengisinya bersebelahan |
| `src/{app,shared,features}/` | `src/{app,modules,components,api}/` |
| `vite build` → `web/dist` | `vite build` → `../backend/spa/dist` |

Penamaan ikut disesuaikan agar terbaca sebagai satu modul: `pengguna.Penyimpanan` dan
`sesi.Penyimpanan` — dua antarmuka bernama sama di dua paket — menjadi `auth.PenggunaRepo` dan
`auth.SesiRepo`; `pengguna.ErrTidakDitemukan` menjadi `auth.ErrPenggunaTidakDitemukan`;
`masuk.Layanan` menjadi `usecase.Layanan`.

### 8.2 Penyimpangan dari Steering — keempat

`docs/Steering/08-TECHNICAL-STRATEGY.md` §2 menetapkan susunan **layer-first** di bawah satu root:
`claim-pnc/cmd/server`, `internal/domain/…`, `internal/app/…`, `internal/adapter/…`,
`internal/transport/…`, `web/`. Susunan ClaimQ adalah **module-first**, dan memisahkan backend
dari frontend.

**Yang dipakai:** susunan ClaimQ, atas keputusan Work Owner 2026-09-15.

**Yang dipertahankan dari Steering, dan alasannya:** lapisan tidak dihapus, melainkan **turun satu
tingkat menjadi subpaket di dalam modul**. `internal/auth` memuat aturan dan mendeklarasikan
seam-nya; `usecase/`, `provider/`, `repo/`, dan `http/` mengimpornya dan tidak pernah sebaliknya.
Arah ketergantungan `ADR-0001` karena itu tetap utuh dan tetap dapat ditegakkan `depguard` kelak —
yang berubah hanya daftar paketnya, bukan aturannya.

**Yang justru membaik:** `D-09` menyebut alasan struktur dibuat preskriptif adalah tim eks-Pega
yang butuh pola seragam. Module-first membuat satu modul dapat dibaca tanpa melompat ke empat
folder, dan membuat batas modul `ADR-0001` terlihat dari daftar folder — bukan hanya dari
kesepakatan. Konsekuensi `ADR-0001` "kelak satu modul dapat dipisah tanpa membongkar seluruhnya"
juga menjadi lebih harfiah.

**Yang memburuk:** susunan di `08-TECHNICAL-STRATEGY.md` §2 kini **tidak lagi menggambarkan kode
yang ada**. Dokumen itu mengikat modul-modul berikutnya, sehingga selisih ini harus diselesaikan —
diperbarui atau dikembalikan — sebelum modul bisnis pertama ditulis, bukan sesudahnya. Pemilik
keputusan: Work Owner.

### 8.3 Tiga hal kecil yang ikut diputuskan

**Nama paket `authhttp` di folder `http/`.** Foldernya `http` supaya seragam dengan susunan ClaimQ;
nama paketnya dibedakan supaya tidak menutupi `net/http` yang dipakai hampir di setiap berkas di
dalamnya. Paket yang bernama `http` dan sekaligus mengimpor `net/http` memang sah di Go, tetapi
membacanya menuntut pengetahuan yang tidak perlu dibebankan ke tim yang sedang belajar Go (`D-09`).

**`backend/spa/dist/.gitkeep` ikut ter-commit.** Direktif `go:embed all:dist` menuntut foldernya
ada saat kompilasi. Tanpa berkas penanda, `go build ./...` pada clone yang bersih gagal sebelum
siapa pun sempat menjalankan `npm run build`. Skrip `npm run build` menuliskannya kembali setelah
Vite mengosongkan folder.

**Folder penyematan dinamai `backend/spa/`, bukan `backend/web/`.** Nama lamanya menyesatkan: ia
terbaca seperti "folder aplikasi web", padahal isinya satu berkas Go dan satu folder hasil build —
**tanpa satu baris pun kode React**. Kebingungan itu benar-benar terjadi dan ditanyakan Work Owner,
yang kemudian memilih sendiri nama `spa`.

Yang **tidak** dapat diubah adalah letaknya. Folder itu wajib berada di dalam modul Go karena
direktif `go:embed` tidak dapat menjangkau ke luar direktori paketnya — pola ber-`../` ditolak
kompilator sebagai `invalid pattern syntax`, dan itu diuji langsung, bukan diasumsikan. Sementara
`ADR-0002` menuntut produksi menjalankan satu binary tanpa runtime Node.js, sehingga berkas
statisnya wajib ikut tersemat. Memindahkannya ke `frontend/` berarti membatalkan `ADR-0002`.

Karena "SPA" adalah singkatan yang tidak semua orang kenal — dan dalam bahasa Indonesia terbaca
sebagai tempat pijat — komentar paketnya mengejanya lengkap pada kalimat pertama, dan `README.md`
memuat catatan khusus yang menjelaskan kenapa folder itu ada di backend.

### 8.4 Bukti bahwa restrukturisasi tidak mengubah perilaku

Seluruh uji dan seluruh verifikasi manual diulang setelah pemindahan, dan hasilnya sama persis
dengan sebelum restrukturisasi — lihat `catatan-pengembangan.md` §7.

---

## 9. Kontrak HCC/HCQ tiba, dan portal multi-entitas — 2026-09-16 (sesi ketiga)

Sesi ini mengubah status penghalang terbesar proyek. Bahan yang diberikan Work Owner: kontrak API
HCC/HCQ, isi `POOLDATA.M_PORTAL_PNC`, `POOLDATA.M_LOGIN_PNC`, dan `POOLDATA.GCNM_CONNECT_REST`.

### 9.1 `ADR-0024` kini dapat diputuskan

`ADR-0024` berstatus **`Proposed`** dengan alasan tertulis: "kontrak yang menjadi tumpuan keputusan
itu **tidak dapat diverifikasi sama sekali** dari bahan yang ada". Empat pertanyaan terbukanya kini
terjawab dua penuh, satu sebagian:

| Pertanyaan terbuka `ADR-0024` | Keadaan setelah 2026-09-16 |
|---|---|
| 1. Apakah API HCC/HCQ ada, atau harus dibangun? | **Terjawab** — ada, alamatnya tersimpan di `POOLDATA.GCNM_CONNECT_REST` |
| 2. Bagaimana bentuk kontraknya? | **Terjawab** — Basic Auth, request `{Login, Password}`, respons ber-`pyErrorCode`; "200" berarti sah |
| 3. Apa yang terjadi bila HCC/HCQ tidak dapat dihubungi? | **Sebagian** — non-karyawan tetap dapat masuk lewat jalur kedua; untuk karyawan belum ada jawaban |
| 4. Bagaimana pengguna dicocokkan dengan `OPERATOR_ID`? | **Masih terbuka** — kolomnya disediakan, dibiarkan kosong |
| 5. Berapa lama sesi berlaku? | **Masih terbuka** — 60m dipakai sementara |

**Rekomendasi:** `ADR-0024` layak dinaikkan dari `Proposed` menjadi `Accepted` dengan mencatat
kontraknya. Itu keputusan Work Owner; tidak dilakukan sepihak dari sisi kode.

### 9.2 Ada baseline Pega yang sebelumnya terlewat

`GCNM_CONNECT_REST` ternyata **ada** di export: `RDB List/BrowseServiceName_sql-SQL.xml`. Rule itu
menyaring dengan `APPLICATIONIP like '%{ASIS:TempError.source}%'` dan `TYPESERVICE`.

Dua hal terbaca dari satu baris itu. Pertama, sistem lama memilih endpoint dengan **membandingkan
nama server** — persis perilaku tersembunyi yang `ADR-0030` putuskan untuk dibuat eksplisit. Kedua,
ia memakai pola `{ASIS:...}` yang merangkai nilai ke dalam teks SQL, yaitu celah injeksi yang sudah
tercatat sebagai utang teknis.

Penggantinya menyaring dengan `APP = :1` lewat parameter binding, sesuai koreksi Work Owner
2026-09-16 bahwa nilainya mengikuti `portal_alias`. Satu perubahan menutup keduanya sekaligus.

### 9.3 Kontrak nyata mengoreksi `D-07` — dan koreksi itu sendiri dikoreksi

`D-07` dan `11-SECURITY.md` §2.1 menyatakan HCC/HCQ mengembalikan "profil lengkap: NIK, nama,
cabang, jabatan, email".

Contoh respons **pertama** hanya memuat blok `Person` — NIK, Name, Login, pyEmail1, pyCompany —
tanpa cabang dan jabatan. Atas dasar itu aturan "lima field wajib" dilonggarkan menjadi tiga. Contoh
**lengkap** yang diberikan kemudian menunjukkan blok `EmpResponse.Placement` yang memuat
`BranchName`, `BranchCode`, dan `PositionName`. Jadi cabang dan jabatan memang ada — bukan di
`Person`, melainkan di `Placement`, dan keduanya kini dipetakan.

Yang tetap berlaku: **tiga field wajib**, bukan lima. Alasannya bukan lagi HCQ melainkan sumber
kedua — `POOLDATA.M_LOGIN_PNC` hanya memuat `login_id` dan `login_name`. Memaksa lima field wajib
akan menolak **seluruh** broker dan surveyor independen.

| Yang wajib | Alasan |
|---|---|
| `Identitas` | kunci alami; tanpa ini pengguna tidak dapat dicocokkan dengan data klaimnya |
| `Nama` | ditampilkan di aplikasi |
| `Jenis` | menentukan arti `Identitas` — NIK atau LOGIN_ID |

### 9.4 `NIK` menjadi `Identitas`

Broker dan surveyor independen **tidak punya NIK**. Kolom dan field yang semula bernama `NIK`
diubah menjadi `Identitas`, dengan kolom `JENIS` yang menyatakan artinya: `KARYAWAN` berarti NIK,
`NON_KARYAWAN` berarti `LOGIN_ID`.

Migrasi `0001` **disunting**, bukan ditambah `0002`. Ia belum pernah dijalankan di lingkungan mana
pun — menambal tabel yang belum ada dengan `ALTER` hanya menambah langkah tanpa menambah keamanan.
Begitu `0001` dijalankan sekali, aturan backward-compatible `P-4` berlaku penuh dan penyuntingan
seperti ini tidak boleh lagi.

### 9.5 Yang diambil dari respons HCQ, dan yang sengaja tidak

Respons lengkapnya memuat lebih dari empat puluh field, termasuk `EmpLeader` (data atasan), grade,
tanggal bergabung, dan susunan organisasi. Atas arahan Work Owner ("ambil data yang perlu saja"),
yang dipetakan hanya sembilan — masing-masing dengan alasan yang tertulis di
`backend/internal/auth/provider/hcq.go`.

Menyalin seluruhnya bukan sekadar berlebihan: setiap field yang ikut masuk menjadi data pegawai
yang tersimpan dan harus dijaga, padahal tidak ada aturan bisnis yang membutuhkannya. Field yang
tidak didaftarkan diabaikan `encoding/json` tanpa galat, sehingga penambahan field di sisi HCQ tidak
merusak apa pun.

`Placement.BranchCode` ikut diambil meski belum dipakai hari ini: `11-SECURITY.md` §3.2 menetapkan
batas data ditegakkan per **cabang**, dan itu menuntut kodenya, bukan hanya namanya.

### 9.6 Urutan dua sumber identitas adalah aturan bisnis

Ditetapkan Work Owner: HCQ dulu; bila gagal, `POOLDATA.M_LOGIN_PNC`. Urutannya tidak boleh
dibalik — mendahulukan tabel lokal berarti kata sandi karyawan ikut disidik dan dicocokkan ke tabel
yang bukan tempatnya.

Satu keputusan yang **tidak** disebut aturan dan diambil sendiri: **galat mana yang dilaporkan bila
kedua jalur gagal.** Bila ada jalur yang tidak dapat dihubungi, yang dilaporkan adalah "sistem
identitas tidak dapat dihubungi", bukan "kata sandi salah". Alasannya: dalam keadaan itu kita memang
tidak tahu apakah kredensialnya benar, dan menyuruh pengguna mengetik ulang sesuatu yang sudah benar
hanya membuang waktunya. Bila seluruh jalur hidup dan semuanya menolak, barulah dilaporkan sebagai
kredensial salah.

### 9.7 Portal: daftar dari tabel, koneksi per entitas

`POOLDATA.M_PORTAL_PNC` memuat enam entitas. Daftar itu **tidak ditulis di kode**: alias portal
ditemukan dengan memindai variabel lingkungan berpola `POOLDATA_<ALIAS>_HOST`, dan nama yang
ditampilkan dibaca dari kolom `PORTAL_NAME`. Menambah entitas berarti menambah satu baris tabel dan
lima baris `.env` — tanpa menyentuh kode, persis yang `ADR-0030` tuntut.

| Keputusan | Isi |
|---|---|
| Letak pemilih portal | **Di dalam aplikasi**, bukan di layar masuk — `ADR-0030` menetapkan berpindah portal tanpa login ulang |
| Portal yang belum siap | **Ditampilkan dan ditandai**, bukan disembunyikan — pengguna tahu entitas itu direncanakan |
| Portal yang gagal dibuka | **Dicatat, tidak menghentikan aplikasi** — satu entitas yang belum siap tidak boleh menghalangi yang sudah siap |
| Portal utama | **Wajib lengkap**; basis datanya melayani daftar portal, alamat HCQ, login non-karyawan, dan tabel sesi |

Nilai `PORTAL_UTAMA` dibaca dari konfigurasi dan ditandai **sementara** atas catatan Work Owner:
kelak portal utama mungkin ditentukan per login dari tabel. Karena itu ia tidak ditulis di kode.

### 9.8 Penyimpangan kelima: skema sidik kata sandi yang lemah

`POOLDATA.M_LOGIN_PNC.HASH_PASSWORD` memakai **SHA-256 polos**, heksadesimal huruf besar, tanpa
garam dan tanpa peregangan. Data contoh membuktikan betapa lemahnya: sidik pada baris contoh adalah
SHA-256 dari `"123"`, yang dapat dibalik dari tabel pelangi dalam hitungan detik.

Ia tetap dipakai apa adanya. Nilainya **sudah tersimpan** di kolom itu dan dipakai sistem yang
sedang berjalan; menggantinya dengan bcrypt atau Argon2 menuntut seluruh pengguna non-karyawan
menyetel ulang kata sandinya. Itu keputusan Work Owner, bukan keputusan yang boleh diambil diam-diam
saat memindahkan aplikasi.

**Pertanyaan terbuka:** apakah skema sidik kata sandi non-karyawan akan ditingkatkan, dan bila ya,
bagaimana masa peralihannya? Pemilik: Work Owner + Security.

### 9.9 `Placement.IsActive` direkam tetapi tidak menggerbang

Respons HCQ memuat `EmpResponse.Placement.IsActive`. Aturan yang ditetapkan Work Owner hanya
menyebut `pyErrorCode = "200"` sebagai syarat masuk, jadi `IsActive` **tidak** dipakai menolak —
menambah syarat sendiri berarti mengarang aturan kewenangan.

Nilainya tetap direkam ke `auth.Profil.AktifDiSumber` supaya terlihat dan siap dipakai begitu
diputuskan.

**Pertanyaan terbuka:** haruskah pegawai dengan `IsActive = false` ditolak masuk? Hari ini ia
diterima selama HCQ menjawab "200", dan yang menggerbang hanya kolom `AKTIF` pada catatan pengguna
lokal yang dikelola administrator. Pemilik: Work Owner.

### 9.10 Yang akhirnya terbukti — dan satu yang masih menghadang

**Diperbarui 2026-09-16 sore.** Kredensial Oracle dev dan HCQ diisi Work Owner, lalu mode periksa
(`./claimpnc.exe -periksa`) dijalankan terhadap infrastruktur nyata. Ini pertama kalinya satu pun
jalur Oracle benar-benar berjalan di proyek ini.

| Integrasi | Keadaan | Bukti |
|---|---|---|
| Koneksi Oracle portal utama | **TERBUKTI** | pool terbuka untuk ASM |
| `POOLDATA.M_PORTAL_PNC` | **TERBUKTI** | 6 portal terbaca, cocok dengan export CSV |
| `POOLDATA.GCNM_CONNECT_REST` | **TERBUKTI** | alamat HCQ terbaca untuk `APP='ASM'` |
| API HCC/HCQ terjangkau dan menjawab | **TERBUKTI** | akun karangan dijawab `pyErrorCode` bukan "200"; bila endpoint mati, rantai akan melaporkan `ErrSistemTidakTerhubung`, bukan kredensial salah |
| `POOLDATA.M_LOGIN_PNC` + sidik SHA-256 | **TERBUKTI** | baris contoh diterima; profil `NON_KARYAWAN` terbentuk dengan nama yang benar |
| Rantai dua sumber berpindah jalur | **TERBUKTI** | login non-karyawan ditolak HCQ lalu diterima tabel lokal, dalam satu panggilan |
| Masuk lewat aplikasi | **MASIH TERHALANG** | `CPNC_PENGGUNA` dan `CPNC_SESI_AKTIF` menjawab `ORA-00942` — migrasi `0001` belum dijalankan DBA |
| Sesi dikenali dua instans | **MASIH TERHALANG** | menunggu migrasi yang sama |

Satu jebakan yang ditemukan justru saat memverifikasi: **login non-karyawan yang berhasil TIDAK
membuktikan HCQ hidup.** Bila HCQ mati, rantai menandainya putus lalu tetap lolos lewat jalur
kedua, dan hasilnya tampak sama persis. Pembedanya harus diuji terpisah dengan akun yang pasti
tidak ada di kedua sumber — barulah terlihat apakah yang dilaporkan "kredensial salah" (kedua
sumber menjawab) atau "sistem tidak dapat dihubungi" (ada yang putus).

**Yang tersisa untuk membuka penghalang terakhir:** DBA menjalankan
`backend/migrations/0001_pengguna_dan_sesi.up.sql` di basis data portal utama, setelah permintaan
perubahan skema tertulis dan persetujuan Work Owner (`D-63`).

### 9.11 Koneksi Oracle dipisahkan dari penyimpanan sesi — koreksi rancangan

Work Owner bertanya: untuk apa migrasi `0001`, dan apakah tidak bisa langsung memakai API HCQ dan
`POOLDATA.M_LOGIN_PNC` saja. Pertanyaan itu membongkar cacat rancangan saya.

**Yang salah:** perakitan di `cmd/claimpnc` menolak kombinasi `PENYIMPANAN=memori` +
`IDENTITAS_ADAPTER=hcq` dengan alasan "hcq menuntut PENYIMPANAN=oracle". Itu menyatukan dua hal
yang sebenarnya terpisah:

| Kebutuhan | Sumbernya | Apakah butuh migrasi 0001 |
|---|---|---|
| Alamat layanan HCQ (`GCNM_CONNECT_REST`) | **dibaca** dari Oracle | tidak |
| Daftar login non-karyawan (`M_LOGIN_PNC`) | **dibaca** dari Oracle | tidak |
| Daftar portal (`M_PORTAL_PNC`) | **dibaca** dari Oracle | tidak |
| Catatan pengguna lokal (`CPNC_PENGGUNA`) | **ditulis** aplikasi | ya |
| Sesi aktif (`CPNC_SESI_AKTIF`) | **ditulis** aplikasi | ya |

Empat baris pertama hanya butuh **koneksi**; dua terakhir butuh **tabel hasil migrasi**.
Menyatukannya memaksa migrasi selesai sebelum integrasi HCC/HCQ dapat dicoba lewat layar — padahal
keduanya tidak saling bergantung sama sekali.

**Perbaikannya:** koneksi dibuka bila `PENYIMPANAN=oracle` **atau** `IDENTITAS_ADAPTER=hcq`.
Kombinasi `memori` + `hcq` kini sah, dan aplikasi mencatat peringatan saat start bahwa sesinya
tidak tahan restart dan tidak dikenali instans lain.

### 9.12 Untuk apa sebenarnya kedua tabel itu

Jawaban jujurnya berbeda untuk masing-masing.

**`CPNC_SESI_AKTIF` tidak dapat dihindari.** Aplikasi menerbitkan sesinya sendiri (`D-07`), dan
`TKT-F3-003` menuntut tiga hal yang seluruhnya menuntut penyimpanan bersama: pencabutan berlaku
**seketika**, sesi dikenali **dua instans** di belakang load balancer (`D-27`), dan keluar mencabut
sesi di server. Alternatifnya hanya dua, dan keduanya gugur: JWT yang memverifikasi dirinya sendiri
tidak dapat dicabut seketika, dan sesi di memori tidak dikenali instans kedua. Satu-satunya cara
lain adalah memanggil HCQ pada **setiap** permintaan — yang berarti menyimpan kata sandi pengguna.

**`CPNC_PENGGUNA` sebenarnya dapat ditunda untuk login semata.** Ia dibutuhkan oleh apa yang
datang sesudahnya, bukan oleh masuk itu sendiri:

| Yang membutuhkannya | Kenapa |
|---|---|
| `TKT-F3-004` tabel 22 peran dan 51 izin menu | peran menempel pada pengguna; tanpa baris pengguna tidak ada tempat menautkannya |
| `S-5` jejak audit (`ADR-0026`) | setiap perubahan bernilai bisnis merujuk pelakunya — dan `ADR-0023` menjadikan jejak audit **satu-satunya** kontrol pengimbang karena tidak ada pemisahan tugas |
| Penonaktifan oleh administrator | kolom `AKTIF` dimiliki administrator Claim PNC; HCC/HCQ tidak dapat disunting dari sini |
| Bekerja saat HCC/HCQ mati | pemeriksaan sesi membaca catatan lokal, bukan memanggil HCQ ulang — inilah inti keputusan `D-07` |

Jadi ia tetap dibangun, tetapi ketiadaannya **tidak menghalangi pengujian integrasi** — dan itulah
yang diperbaiki di §9.11.


## 10. Modul Master Status Progres 1 — 2026-09-17 (sesi keempat)

Modul bisnis pertama. Acuannya `Harness/StatusProgress-Harness.xml` atas tabel
`POOLDATA.GCNM_MST_PROGRESS_KLAIM`.

### 10.1 Keputusan yang diambil Work Owner pada sesi ini

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Lingkup portal modul yang menyentuh basis data entitas | **Portal-scoped sekarang**: alias portal dikirim di header, koneksi di-resolve per permintaan, permintaan tanpa portal **ditolak** |
| 2 | Pintu masuk layar, sementara modul Home dilarang diubah | **Rute saja**; `HalamanBeranda.tsx` tidak disentuh |
| 3 | Sumber daftar Posisi yang di Pega di-hardcode | **Konstanta di lapisan domain**, disajikan lewat API, dicatat sebagai utang menuju master data `F-4` |

### 10.2 Penamaan ulang tiga kolom (D-19)

Kueri lama mengaliaskan ketiga kolomnya ke nama yang tidak mencerminkan isi — persis
utang teknis §4.2 `03-CURRENT-ARCHITECTURE.md`. Alias itu tidak dibawa:

| Kolom | Alias Pega | Nama baru | Kenapa aliasnya menyesatkan |
|---|---|---|---|
| `ID_PROGRESS` | `"CaseID"` | `ID` | bukan nomor klaim sama sekali |
| `STS_PROGRESS1` | `"City"` | `Nama` | bukan nama kota |
| `STATUS` | `"CityID"` | `KodePosisi` | bukan kode kota, dan bukan penanda aktif |

### 10.3 `STATUS` adalah kode posisi klaim, bukan flag aktif

Kesimpulan ini menentukan seluruh bentuk layar, jadi buktinya dicatat utuh:

| Bukti | Isi |
|---|---|
| `Section/BrowseStatusProgress-Section.xml` | sel isian `TempInputStatus.CityID` ber-`pyLabelFieldValue` = **"Posisi"**, `pyControlDisplayTitle` = **Dropdown**, sumber `TempPosition.pxResults` |
| `Activity/InsertMstStatusProgress1_act-Act.xml` | `Local.POSISI := TempInputStatus.CityID` |
| `Activity/ViewStatusProgress_act-Act.xml` | step "INPUT POSISI KLAIM untuk dropdown" berisi `REGISTER`/`002`, `SURVEY`/`004`, `AKSEPTASI`/`007`, `KOMITE`/`006` |
| 17 berkas yang menyebut `GCNM_MST_PROGRESS_KLAIM` | tidak satu pun menyaring `STATUS`; semuanya menggabung lewat `ID_PROGRESS` |

Butir terakhir dicatat apa adanya: kolom ini **diisi layar ini dan tidak pernah dipakai
menyaring apa pun** di sistem lama. Tidak diberi perilaku penyaringan yang tidak pernah
ada — itu akan menjadi fitur karangan, bukan migrasi.

### 10.4 Tidak ada penghapusan, dan itu disengaja

Diperiksa ke seluruh export: **tidak ada satu pun pernyataan `DELETE`** terhadap
`GCNM_MST_PROGRESS_KLAIM`. Berkas `Data Transform/CNMShowInsertValueMstStsProgress_delet-DT.xml`
yang namanya menjanjikan penghapusan ternyata **tidak memuat langkah apa pun**.

Tiga alasan tidak menambahkannya:

1. Menambah tombol hapus berarti mengarang perilaku yang tidak pernah ada di sistem lama.
2. `D-66` menuntut soft delete, dan tabelnya **tidak punya kolom penanda terhapus**.
   Menambah kolom menuntut permintaan skema tertulis, persetujuan Work Owner, dan
   pelaksanaan DBA (`D-63`).
3. Barisnya dirujuk `GCNM_PROGRESS_CLAIM.STATUS_PROGRESS1` dan
   `GCNM_MST_PROGRESS.ID_PROGRESS` pada data klaim yang sudah berjalan — penghapusan
   fisik akan memutus rujukan itu.

### 10.5 Bentuk nomor direplikasi, cacatnya dicatat

`Activity/InsertMstStatusProgress1_act-Act.xml` menyusun ID baru sebagai `"0" + nomor`
atas hasil `SELECT NVL(MAX(ID_PROGRESS),0)+1`. Bentuk itu **direplikasi**, karena
`ID_PROGRESS` adalah kunci yang dirujuk dua tabel lain dan baris lamanya sudah memakainya
— menggantinya akan memutus baris baru dari data historis. `P-5` berlaku, dan bentuk ini
tidak ada di daftar 13 perbaikan eksplisit `D-49`.

Dua cacat ikut terbawa. Keduanya dicatat sebagai uji supaya tidak disangka rancangan:

**Lebarnya tidak dipadatkan**, sehingga urutan teks tidak sama dengan urutan penerbitan —
`"010"` mendahului `"09"`. Persoalan yang sama sudah dikenali pada nomor klaim
`PNCN.YY.xxxx` (`D-71` butir 2). Karena itu daftar diurutkan **di basis data** mengikuti
kueri lama, bukan diurutkan ulang sebagai teks di frontend.

### 10.6 Nomor berikutnya dihitung di Go, bukan dengan MAX di SQL

Ini satu-satunya tempat perilaku sengaja **dibuat berbeda** dari kueri lama, dan
alasannya bukan selera.

Bila `ID_PROGRESS` bertipe `VARCHAR2`, `MAX(ID_PROGRESS)` adalah maksimum
**leksikografis**. Begitu tabel memuat `"010"`, maksimumnya tetap `"09"` karena `'9'`
lebih besar dari `'1'` pada karakter kedua — nomor berikutnya kembali `10`, dan ID
`"010"` diterbitkan dua kali. Tipe kolomnya sendiri belum diketahui (`R-08`), sehingga
apakah cacat itu sudah aktif hari ini tidak dapat dipastikan.

Menghitungnya di Go menghindari pertanyaan itu: setiap ID ditafsirkan sebagai angka,
diambil yang terbesar, ditambah satu. **Untuk rentang yang kedua cara sepakat — satu
sampai sembilan baris — hasilnya sama persis dengan Pega**, sehingga uji kesetaraan tidak
melihat selisih. Sekaligus memenuhi aturan Steering bahwa pemformatan angka dilakukan di
Go, bukan di SQL.

ID yang tidak dapat ditafsirkan sebagai angka diabaikan saat mencari yang terbesar tetapi
tetap dihitung sebagai terpakai — baris lama dapat memuat apa saja, dan menabraknya lebih
buruk daripada melewatinya.

### 10.7 Penyimpangan keenam: batas transaksi berada di dalam repo

`08-TECHNICAL-STRATEGY.md` §4.5 menetapkan transaksi dimulai dan diakhiri di **lapisan
aplikasi**. Pada `SisipBaru`, transaksinya berada di dalam **repo**.

Alasannya: nomor baru diturunkan dari isi tabel itu sendiri. Memecahnya menjadi "ambil
nomor" di aplikasi lalu "sisip" di repo melebarkan jarak antara membaca dan menulis —
dan jarak itulah yang membuat dua penambahan bersamaan bertabrakan. Kueri pengambil ID
karena itu memakai `SELECT ... FOR UPDATE`, sehingga penambahan kedua menunggu yang
pertama selesai lalu membaca ulang termasuk baris yang baru masuk.

Ia tetap portabel: `FOR UPDATE` didukung Oracle maupun PostgreSQL, jadi `D-20` tidak
dilanggar. Biayanya dapat diterima — tabel ini master berbaris sedikit dan nyaris tidak
pernah ditulis.

**Yang masih tersisa:** tanpa constraint unik pada `ID_PROGRESS` — dan keberadaannya
belum diketahui (`R-08`) — penyerialan ini bersandar pada penguncian, bukan pada jaminan
basis data. Constraint unik masuk daftar permintaan DDL ke DBA.

### 10.8 Empat posisi klaim masih di kode — utang yang dicatat terang

`D-15` melarang nilai bisnis di-hardcode. Keempat posisi tetap berada di
`internal/statusprogres/posisi.go`, dan itu **pelanggaran yang disadari**, bukan kelalaian.

Yang membuatnya dapat diterima sekarang:

| Hal | Keadaan |
|---|---|
| Tidak ada tabel master posisi di sistem lama | tidak ada yang dapat dibaca; di Pega pun ia dirakit di dalam activity |
| Membuat tabelnya | menuntut permintaan skema tertulis, persetujuan Work Owner, pelaksanaan DBA (`D-63`) — modul akan terhalang |
| Mengarang tabel beserta isinya | menebak, dan menebak dilarang |

Yang sudah dibereskan sekarang hanyalah **tempatnya**: ia ada di satu tempat dan
disajikan lewat `GET /api/master/posisi-klaim`, sehingga frontend tidak menyimpan
salinannya. Di sistem lama keempat pasang nilai itu ditulis ulang di setiap activity yang
membutuhkannya.

Pemindahannya menjadi master data `F-4` menyentuh satu berkas dan satu endpoint.

### 10.9 Portal melekat pada permintaan, bukan pada keadaan server

`R-20` menyebut dua jalur kegagalan. Keduanya ditutup, dan keduanya ditutup dengan cara
yang sama — **menolak**:

| Jalur kegagalan `R-20` | Penanganan |
|---|---|
| Jatuh ke koneksi default ketika portal tidak diketahui | `portal.PilihAktif` mengembalikan `ErrTidakDisebut`; handler menolak dengan `400`. Tidak ada jalur mana pun yang memanggil `Kumpulan.Utama()` dari modul bisnis |
| Portal aktif disimpan sebagai keadaan global di server | portal terpilih diletakkan di `context.Context` permintaan. Dua permintaan bersamaan dari pengguna yang sama tidak dapat saling menimpa |

Ditambah satu pembedaan yang tidak diminta `R-20` tetapi berguna: **"tidak ada di daftar"
dibedakan dari "koneksinya belum hidup"** (`400 portal_tidak_dikenal` versus
`503 portal_belum_siap`). Keduanya menuntut tindak lanjut berbeda — yang pertama
kemungkinan cacat antarmuka, yang kedua pekerjaan tim infrastruktur.

**Yang TIDAK dijawab modul ini, dan harus disebut terang:** pemeriksaannya menjawab
"portal ini ada dan koneksinya hidup", **bukan** "pengguna ini berwenang atas portal
ini". Kewenangan portal per pengguna adalah `TKT-F6-003` dan masih terhalang — `D-78`
menyisakan pertanyaan terbuka di mana data kewenangan itu disimpan, karena ia tidak dapat
ikut tinggal di basis data masing-masing entitas.

Selama itu belum terjawab, **setiap pengguna yang sudah masuk dapat memilih portal mana
pun yang koneksinya hidup**. Itu `R-20` yang masih terbuka, bukan yang sudah ditutup.

### 10.10 Galat validasi dikembalikan seluruhnya, dan disorot per isian

`11-CROSSCUTTING.md` §1.2 aturan 1 menuntut kesalahan validasi dikumpulkan seluruhnya.
Diterapkan pada ketiga lapisan:

- Domain mengembalikan `GalatValidasi` yang memuat seluruh `Pelanggaran`.
- Transport memetakannya ke `422` dengan `detail` berisi `{kolom, pesan}` per pelanggaran.
- Layar menyorotnya pada isiannya masing-masing lewat `setError`, bukan meringkasnya
  menjadi satu kotak pesan.

`422` dibedakan dari `400` sesuai `10-API-STRATEGY.md` §5: `400` berarti ada cacat di
frontend, `422` berarti pengguna perlu memperbaiki isiannya.

### 10.11 Pemetaan galat modul, sementara TKT-F1-004 masih terhalang

Kontrak galat yang mengikat seluruh aplikasi adalah `TKT-F1-004`, dan ia belum diputuskan.
Yang ada hanyalah pemetaan milik modul auth.

Menambah kode galat ke modul auth berarti menyunting modul yang sudah dinyatakan selesai.
Karena itu galat dipetakan berlapis, dan setiap lapis dimiliki modul yang memang memiliki
galatnya:

```
galat domain statusprogres  -> statusprogreshttp.petakanGalat
galat portal                -> portalhttp.DenganGalatPortal
sisanya                     -> authhttp.TulisGalat  (500, pesan umum, rincian ke log)
```

Bentuk badannya tetap `{kode, pesan}` pada ketiganya, sehingga klien tidak menghadapi
dua bentuk galat yang berbeda. **Utang:** begitu `TKT-F1-004` diputuskan, ketiga pemetaan
ini menjadi satu.

Kekeliruan yang sempat terjadi dan cara ia terbongkar dicatat di
`catatan-pengembangan.md` §9.5 — awalnya galat portal dipetakan **dua kali**, di modul
portal dan di modul statusprogres. Duplikatnya dibuang.

### 10.12 Jalur API tanpa awalan `/v1`

`10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`. Kontrak yang sudah berjalan tidak
memakainya (`/api/masuk`, `/api/portal`), dan memperkenalkannya di modul ini saja akan
membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai utang, bukan
diselesaikan sepihak di satu modul.

### 10.13 Batas panjang nama adalah asumsi, bukan fakta

> **DISUPERSEDE oleh §10.16.** Panjang kolom sudah ditetapkan Work Owner 2026-09-17:
> **100 karakter**. Bagian di bawah merekam keadaan sebelum jawaban itu — angkanya (200)
> tidak lagi berlaku, alasan menduplikasinya di dua tempat masih berlaku.

`BatasPanjangNama = 200` **bukan** panjang kolom `STS_PROGRESS1` yang sebenarnya — DDL-nya
tidak ada di export (`R-08`). Angkanya dipilih longgar dengan sengaja: penjaga yang
terlalu ketat menolak data yang sebenarnya sah, sedangkan yang terlalu longgar paling
buruk berujung galat dari basis data — dan galat itu terlihat, bukan diam-diam memotong
data.

Nilainya diulang di frontend (`FormStatusProgres.tsx`) supaya pengguna tahu sebelum
mengirim. Server tetap yang berwenang; pemeriksaan di layar hanya kenyamanan. Begitu DDL
diterima, **kedua tempat** harus disesuaikan — dan itu utang yang disadari dari
menduplikasi sebuah angka.

### 10.14 Yang belum dapat dibuktikan

> **SEBAGIAN TERJAWAB — lihat §10.16 dan §10.17.** Tipe kolom `ID_PROGRESS` sudah
> ditetapkan **CHAR berlebar tetap**, dan penanganannya mengubah dua kueri. Yang masih
> belum terbukti dari tabel di bawah: keenam kueri terhadap Oracle sungguhan, **lebar**
> kolomnya, constraint unik, dan kesetaraan gerbang 1.

| Hal | Sebab |
|---|---|
| Keenam kueri SQL sah terhadap Oracle | belum pernah dijalankan; seluruh verifikasi memakai adapter memori |
| Tipe dan panjang kolom | DDL belum ada (`R-08`) |
| Ada tidaknya constraint unik pada `ID_PROGRESS` | idem — penyerialan nomor bersandar pada penguncian, bukan jaminan basis data |
| Kesetaraan hasil dengan Pega (gerbang 1) | menuntut Pega staging (`ADR-0027`) **dan** isi tabel yang sebenarnya, yang belum ada |
| Perilaku pada tipe kolom `CHAR` berlebar tetap | pemangkasan sudah dipasang tetapi belum diuji terhadap tipe sebenarnya |

### 10.15 Yang perlu diminta

> **Dua baris pertama menyempit — lihat §10.16.** Tipe kolom dan panjang
> `STS_PROGRESS1` sudah dijawab Work Owner; dari DDL yang tersisa dibutuhkan **lebar**
> kolom `ID_PROGRESS` dan ada-tidaknya constraint unik.

| Yang diminta | Kepada | Untuk |
|---|---|---|
| DDL `POOLDATA.GCNM_MST_PROGRESS_KLAIM` | DBA | menetapkan `BatasPanjangNama`, memastikan tipe kolom, memeriksa constraint unik |
| Isi tabelnya (CSV, seperti `v_sts_claim.csv`) | DBA | menggantikan `DaftarContoh()` dan memungkinkan uji kesetaraan |
| Constraint unik pada `ID_PROGRESS` bila belum ada | DBA + Work Owner | menjadikan keunikan nomor jaminan basis data, bukan hanya hasil penguncian |
| Penegasan bahwa layar ini tidak punya penghapusan | Work Owner | menutup pertanyaan apakah ketiadaan `DELETE` di Pega memang disengaja |
| Kewenangan portal per pengguna | Work Owner (`TKT-F6-003`, `D-78`) | menutup sisa `R-20` |


### 10.16 Empat asumsi dijawab Work Owner — 2026-09-17 (lanjutan)

Empat pertanyaan yang §10.13 dan §10.14 catat sebagai terbuka diajukan sebagai pilihan
dan dijawab pada hari yang sama. Dua di antaranya **mengubah kode**, satu menegaskan yang
sudah ada, satu masih ditahan.

| Asumsi | Jawaban | Akibat |
|---|---|---|
| Tipe kolom `ID_PROGRESS` | **CHAR berlebar tetap** | **Kode berubah** — lihat §10.17 |
| Panjang kolom `STS_PROGRESS1` | **100 karakter** | `BatasPanjangNama` 200 → 100, di backend dan frontend |
| Penghapusan baris | **tetap tidak ada**, sama seperti Pega | tidak berubah; ketiadaan tombol hapus kini ketetapan, bukan tafsiran |
| Modul berikutnya | belum ditentukan | tidak ada modul baru dimulai |

`BatasPanjangNama` **bukan lagi asumsi**. §10.13 yang menyebutnya "asumsi, bukan fakta"
berlaku untuk keadaan sebelum jawaban ini; yang berlaku sekarang adalah entri ini.
Angkanya tetap diulang di dua tempat — backend dan frontend — dan itu tetap utang yang
disadari; yang hilang hanyalah ketidakpastian nilainya.

### 10.17 CHAR berlebar tetap: parameter binding mengubah perilaku, bukan mempertahankannya

Jawaban "CHAR berlebar tetap" membongkar cacat yang sebelumnya tidak terlihat, dan
sebabnya justru aturan yang wajib dipatuhi.

**Bagaimana cacatnya bekerja.** Kolom CHAR memadatkan nilainya dengan spasi tanpa memberi
tanda apa pun: `"01"` tersimpan sebagai `"01 "`. Oracle membandingkan dua nilai CHAR
dengan **blank-padded comparison** — spasi di ujung diabaikan. Literal teks di dalam SQL
bertipe CHAR, sehingga kueri lama yang **merangkai** nilainya menjadi `= '01'` memang
cocok dengan `"01 "`.

Tetapi **parameter binding bertipe VARCHAR2**, dan CHAR lawan VARCHAR2 memakai
**non-padded comparison**. `"01 "` tidak sama dengan `"01"`, dan barisnya tidak ketemu.

| Kueri | Gejala bila dibiarkan `= :1` |
|---|---|
| `statusprogres_ambil` | pemuatan baris ke modal sunting selalu gagal |
| `statusprogres_perbarui` | `UPDATE` mengenai **nol baris**, dan nol baris diartikan repo sebagai "tidak ditemukan" |

Gejalanya menyesatkan justru karena **tidak ada galat basis data sama sekali**.
Penyuntingan hanya melaporkan "tidak ditemukan" untuk setiap baris yang sebenarnya ada.

**Yang penting untuk dipahami:** menyalin `= :1` apa adanya dari kueri lama akan
**mengubah** perilaku, bukan mempertahankannya. Penyebabnya perpindahan dari perangkaian
string ke parameter binding — yang diwajibkan `08-TECHNICAL-STRATEGY.md` §4.3 dan tidak
dapat ditawar, karena perangkaian nilai adalah celah injeksi yang justru sedang ditutup.

Ini contoh konkret bahwa **kesetaraan perilaku `P-5` tidak selalu berarti menyalin teks
SQL apa adanya**. Kadang teks yang sama menghasilkan perilaku yang berbeda begitu cara
nilainya masuk berubah.

**Penanganannya:** `WHERE TRIM(ID_PROGRESS) = :1` pada kedua kueri.

| Pilihan | Kenapa tidak dipakai |
|---|---|
| `CAST(:1 AS CHAR(n))` | lebar kolom `n` belum diketahui, dan salah menebaknya mengulang cacat yang sama |
| Memadatkan nilai bind di Go | menuntut `n` yang sama; juga menyebarkan pengetahuan tentang tipe kolom ke lapisan yang tidak seharusnya tahu |
| Kembali merangkai literal | melanggar §4.3; tidak dipertimbangkan |

`TRIM` benar untuk lebar berapa pun dan portabel — Oracle maupun PostgreSQL sama-sama
mendukungnya, jadi `D-20` tidak dilanggar. Pada PostgreSQL, tipe `char(n)` bahkan sudah
mengabaikan spasi ujung saat membandingkan, sehingga `TRIM` di sana tidak berakibat apa
pun selain menjadikan maksudnya terbaca.

**Biaya yang diterima:** index atas `ID_PROGRESS` tidak terpakai. Dapat diterima di sini —
tabel ini master berbaris sedikit dan dibaca per baris hanya saat menyunting. Pola ini
**tidak boleh ditiru begitu saja** pada tabel bervolume besar; di sana `n` harus diketahui
dan nilai bind yang dipadatkan.

**Uji penjaganya:** `repo/sqlstore/kueri_test.go` menuntut `TRIM(ID_PROGRESS)` ada pada
kedua kueri dan menolak `WHERE ID_PROGRESS =` tanpa TRIM. Ia ada karena gejala cacatnya
senyap: seseorang yang kelak "merapikan" kueri ini tidak akan melihat apa pun rusak
sampai pengguna melaporkan bahwa tombol Ubah tidak pernah berhasil.

**Yang sudah benar sejak awal dan tidak perlu diubah:** pembacaan sudah memangkas ketiga
kolom lewat `pindaiBaris`, sehingga domain tidak pernah melihat nilai berpadat spasi;
`CariPosisi` juga sudah memangkas kode posisi. Keduanya memang dipasang untuk kemungkinan
ini, dan jawaban Work Owner mengubahnya dari kehati-hatian menjadi keharusan.

**Yang masih belum terbukti:** lebar kolomnya. Ia tidak dibutuhkan penanganan di atas,
tetapi tetap diminta bersama DDL — nilai yang lebih panjang dari lebar kolom akan ditolak
basis data, dan itu memang yang diinginkan.


### 10.18 Kerangka menu — Home tetap utuh, dan cara itu mungkin

Ditanyakan Work Owner 2026-09-18: *"jadi menunya apakah sudah beres?"* Jawabannya saat
itu **belum** — dan itu memang akibat langsung §10.1 pertanyaan 2 (*"rute saja, jangan
sentuh Beranda"*). Layar master hanya dapat dicapai dengan mengetik alamatnya.

Work Owner memilih **menu di kerangka, Home tetap utuh**.

**Jalan yang sebelumnya terlewat.** Pada §10.1 saya menyajikan pilihan seolah menu
menuntut menyunting `HalamanBeranda.tsx`. Itu tidak benar: pembungkus rute `/` berada di
`app/App.tsx`, yang **kerangka, bukan modul Home**. Menu karena itu dapat dipasang tanpa
menyentuh satu byte pun berkas modul Beranda — dan itulah yang dikerjakan.

| Berkas | Perlakuan |
|---|---|
| `app/menu.ts` | **baru** — peta menu sebagai data |
| `app/NavigasiUtama.tsx` | **baru** — penampil menu, responsif, penanda aktif |
| `app/Kerangka.tsx` | **baru** — bingkai: peringatan sesi, menu, pemilih portal, tombol keluar |
| `app/App.tsx` | disunting — kedua rute dibungkus `Kerangka`; `Layar` yang sementara dibuang |
| `modules/beranda/HalamanBeranda.tsx` | **tidak disentuh** |

**Menu adalah data, bukan JSX.** Menambah modul berarti menambah satu baris di
`app/menu.ts` — sejajar dengan backend, tempat modul baru cukup menambah satu pemanggilan
`Pasang(...)` di `cmd/claimpnc`. Kalau menunya ditulis sebagai JSX, setiap modul baru
menuntut menyunting tata letak, dan pada 74 layar itu berubah menjadi tata letak yang
berbeda-beda.

**Dua bentuk menurut lebar layar.** Kolom samping di layar lebar, deret mendatar yang
dapat digulir di layar sempit. Bukan satu bentuk yang dipaksakan: kolom samping pada lebar
ponsel memakan hampir separuh layar, dan `D-12` menetapkan surveyor memakai tablet dan
ponsel di lapangan. Judul kelompok disembunyikan pada layar sempit karena di dalam deret
mendatar ia memutus alurnya; butirnya tetap terlihat seluruhnya.

**Penanda aktif memakai `aria-current`, bukan hanya warna.** Pengguna pembaca layar perlu
tahu ia sedang di mana, dan warna tidak menyampaikan itu. Butir Beranda memakai `end`
supaya ia tidak ikut aktif pada setiap jalur — tanpa itu ia aktif di mana-mana, karena
semua jalur dimulai dengan `/`.

### 10.19 Menu BUKAN kendali akses, dan itu dinyatakan di layar

Daftar menu masih **tetap**, belum disaring izin peran. Penghalangnya berlapis:

| Penghalang | Keadaan |
|---|---|
| Tabel 22 peran dan 51 izin menu | `TKT-F3-004`, belum dikerjakan |
| Peta peran → menu | hidup di 34 When rule; **lima hilang dari export** |
| Penugasan operator ke peran | **tidak ada di basis data** — `POOLDATA.T_ACCESS_GROUP_PNC` hanya memetakan `OPERATOR_ID` → `OLD_OPERATOR_ID` |

Butir ketiga yang paling mendasar: tabel izinnya dapat dibangun tetapi **belum dapat
diisi**.

Karena itu navigasinya memuat satu baris keterangan: *"Daftar menu masih tetap, belum
disaring izin peran. Kewenangan tetap diperiksa di server pada setiap permintaan."*

**Kenapa keterangan itu ada di layar, bukan hanya di dokumen.** Penguji bisnis yang
melihat menu lengkap dapat mengira izin sudah ditegakkan di antarmuka — dan itu persis
cacat sistem lama yang tidak boleh diulang: `pyPrivilegeName` terisi pada **1 dari 902**
activity, sehingga otorisasi di sana hanyalah penyembunyian menu. `D-59` menetapkan yang
menjadi kendali adalah pemeriksaan di server pada setiap endpoint; menyembunyikan menu
hanya kenyamanan tampilan.

Begitu `TKT-F3-004` tersedia, yang berubah adalah penyaringan `menuUtama` terhadap izin
pengguna — bentuk datanya tidak perlu berubah.

### 10.20 Pemilih portal dan tombol keluar pindah ke kerangka — dengan satu penyesuaian

Keduanya diletakkan di kerangka, bukan hanya di Beranda. Tanpa itu layar modul menjadi
**jalan buntu**: pengguna tidak dapat berpindah entitas maupun keluar tanpa kembali ke
beranda lebih dulu. Berpindah portal tanpa login ulang adalah inti `ADR-0030`, jadi
pemilihnya harus terjangkau dari layar mana pun.

**Penyesuaian yang disadari:** Beranda dibangun sebelum kerangka ini ada dan memuat
keduanya di header-nya sendiri. Menampilkannya dua kali membuat tidak jelas pemilih portal
mana yang berlaku, sehingga kerangka menerima penanda `aksiDiHalaman` dan Beranda
melewatinya.

Keduanya **seharusnya tinggal di kerangka saja**. Memindahkannya menuntut menyunting
`modules/beranda/HalamanBeranda.tsx`, dan modul Beranda dinyatakan tidak boleh diubah.
Dicatat sebagai utang teknis, bukan dikerjakan sepihak — lihat §10.21.

**Akibat baiknya yang ikut terbawa:** `PeringatanSesi` kini tampil di **setiap** layar
dalam sesi, bukan hanya di beranda. Sebelumnya layar modul harus mengingat memasangnya
sendiri, dan satu layar yang lupa berarti peringatan sesi hampir habis tidak pernah muncul
di sana — pada sistem yang formnya panjang, itu berarti pekerjaan hilang tanpa peringatan.

### 10.21 Satu kalimat di Beranda kini bertentangan dengan layar

`modules/beranda/HalamanBeranda.tsx` memuat paragraf:

> *"Menu belum tampil di sini. Daftar menu mengikuti izin peran, dan tabel 22 peran
> beserta 51 izin menu adalah TKT-F3-004 — masih menunggu daftar penugasan operator per
> peran dari DBA dan Work Owner."*

Kalimat **pertama** sekarang salah: menu tampil, tepat di sebelahnya. Sisanya masih benar —
menu itu memang belum disaring izin peran.

**Tidak saya sunting.** Berkas itu milik modul Beranda yang dinyatakan tidak boleh diubah,
dan memperbaikinya sepihak berarti mengabaikan batasan yang Anda tetapkan demi kerapian
satu paragraf. Diajukan sebagai permintaan izin, bukan diambil sendiri.

Bila diizinkan, perubahannya satu paragraf: kalimat pertama dibuang, sisanya
dipertahankan — dan pemilih portal serta tombol keluar di header Beranda ikut dipindahkan
ke kerangka sehingga penanda `aksiDiHalaman` pada §10.20 tidak dibutuhkan lagi.
## 11. Modul Master Rekening (2026-09-17)

### 11.1 Master ini punya alur persetujuan — dan itu menjawab pertanyaan `TKT-F4-001`

`TKT-F4-001` mencatat pertanyaan terbuka: *"apakah perubahan master butuh alur
persetujuan, dan berlaku untuk master yang mana?"*

Untuk Master Rekening pertanyaan itu **tidak perlu ditunggu jawabannya** — sistem yang
berjalan hari ini sudah menjawabnya. `POOLDATA.LST_ACCOUNT` memuat `APPROVAL`,
`KOMITE_APPROVAL`, dan `TANGGALAPPROVEKOMITE` sejak awal, dan
`CNMUpdateMasterRekening_act` menegakkan alurnya.

Alasannya juga jelas: rekening menentukan **ke mana uang klaim dikirim**. Satu baris
yang keliru berarti pembayaran mendarat di rekening yang salah, dan tidak ada langkah
sesudahnya yang dapat menangkapnya.

**Yang tetap menjadi keputusan Work Owner** adalah apakah pola ini berlaku untuk master
lain. Modul ini tidak memutuskannya, dan tidak memaksakan bentuknya ke master mana pun.

### 11.2 Alias Pega tidak dibawa masuk

Alias kolom sistem lama menyesatkan secara aktif. Yang paling berbahaya: **satu alias
dipakai untuk dua kolom berbeda pada rule yang berbeda**.

| Alias | Artinya saat SELECT | Artinya saat UPDATE |
|---|---|---|
| `pyID` | `USER_INPUT` | `APPROVAL` |
| `KOMISI` | `FLAGUPDATE` | `TANGGALAPPROVEKOMITE` |

Ditambah `CaseID`→`STS_AKTIF`, `CoverID`→`ACCOUNT_TYPE`, `pyCountry`→`KOMITE_APPROVAL`,
dan `NoHpUserAccount`→`EMAILINPUT` (surel, bukan nomor HP).

Seluruhnya diganti nama domain berbahasa Indonesia. Pemetaan tiga arah
alias→kolom→domain ditulis lengkap di kepala
`repo/sqlstore/rekening.sql` — satu-satunya tempat ketiganya dapat dibandingkan.
Ini melaksanakan `03-CURRENT-ARCHITECTURE` §4.2.

### 11.3 Perilaku yang sengaja DIPERTAHANKAN walau cacat

Work Owner memilih paritas lebih dulu (`P-5`). Tiga hal berikut **tidak** diperbaiki,
dan dicatat di sini supaya tidak terbaca sebagai kelalaian:

| Perilaku lama | Kenapa cacat | Kenapa tetap dipertahankan |
|---|---|---|
| Nomor rekening yang **ditolak** komite **dihapus** lalu disisip ulang saat diajukan lagi | Menghilangkan jejak penolakan sebelumnya — persis pola yang `ADR-0013` perintahkan diganti | Menggantinya dengan versi baru mengubah perilaku, sehingga uji kesetaraan tidak lagi 1:1 |
| Portal yang didaftarkan ke Kasir **di-hardcode** `ASM` dan `SIMASNET` | `ADR-0025` menuntutnya menjadi master/konfigurasi | Master portal untuk keperluan ini belum ada |
| Tipe rekening di-hardcode di `SetTipeRekening` | Idem | Master tipe rekening belum ada |

Keduanya yang terakhir **tidak disebar di dalam percabangan**: masing-masing menjadi
satu konstanta bernama (`portalYangDidaftarkanKeKasir`, `TIPE_REKENING`), sehingga saat
masternya tersedia yang perlu diubah hanya satu tempat.

### 11.4 Yang DIPERBAIKI, karena memperbaikinya tidak mengubah perilaku

| Cacat lama | Perbaikan |
|---|---|
| `DELETE FROM LST_ACCOUNT where {ASIS:TempDataBank.City}` — klausa WHERE dirangkai dari properti klipboard | Kunci dan syarat `APPROVAL='2'` ditulis **di dalam** kueri; pemanggil tidak dapat menggesernya |
| `UPDATE … where account_no = {TempBank.pyEmailAddress}` — kunci satu kolom, lewat properti bernama alamat surel | Kunci menjadi pasangan `ACCOUNT_NO` + `BANKID`; nomor rekening yang sama dapat ada di dua bank |
| `TGL_INPUT = sysdate` | Waktu dari seam `platform/waktu`, disimpan UTC, dapat diuji deterministik |
| `SUBSTR(response_kasir, INSTR(…))` di dalam SQL | Pindah ke `masterrekening.PangkasResponsKasir`; `INSTR` tidak portabel ke PostgreSQL |
| 68 pemakaian `ROWNUM` | `OFFSET … FETCH NEXT` (`09-DATABASE-STRATEGY` §3.3) |

### 11.5 Keputusan komite disimpan SEBELUM Kasir dihubungi

Urutannya disengaja dan berbeda dari cara membacanya sepintas.

Keputusan komite adalah **fakta bisnis yang sudah terjadi** begitu orangnya menekan
tombol. Bila ia baru disimpan setelah Kasir menjawab, satu kegagalan jaringan akan
membuang keputusan yang sudah benar-benar diambil, dan komite harus memutuskan ulang
tanpa tahu kenapa.

Karena itu: keputusan disimpan dulu, lalu pendaftaran ke Kasir dijalankan sebagai
akibatnya. Kegagalannya **tidak** membatalkan keputusan; ia dicatat di `STS_SERVICE` dan
`RESPONSE_KASIR`, diberitahukan ke PIC, dan **ditampilkan di layar** — bukan
disembunyikan. Rekening yang disetujui tetapi gagal didaftarkan akan menahan pembayaran,
dan satu-satunya orang yang dapat menindaklanjutinya adalah petugas yang melihat layar.

### 11.6 Asumsi yang disadari dan menunggu konfirmasi

`CNMUpdateMasterRekening_act` memanggil **kedua** Connect-REST Kasir dengan prasyarat
yang **sama persis** (`komite="ya" && APPROVAL="1"` dan portal ASM/SIMASNET), tanpa
syarat pembeda di antara keduanya. Export tidak menunjukkan mana yang dipakai kapan.

**Asumsi yang diambil:** rekening yang menggantikan rekening lama
(`OLDACCOUNT_NO` terisi) dikirim lewat `UpdateSearchDataRekeningToKasir`; selebihnya
lewat `InjectDataRekeningToKasir`. Dasarnya nama servicenya sendiri.

**Menunggu konfirmasi Work Owner.** Bila salah, yang berubah hanya satu percabangan di
`usecase/putuskan.go`.

### 11.7 Penghalang yang masih ada

| Penghalang | Pemilik | Akibatnya sekarang |
|---|---|---|
| **Alamat dan kredensial API Kasir** tidak ada di export — ia di konfigurasi instans Pega | **Tim Infra** | Seam Kasir terisi tiruan; rekening tetap dapat diputuskan komite, pendaftaran ke Kasir dilewati. Aplikasi **memperingatkannya di log saat start** |
| **Bentuk badan permintaan Kasir** disusun dari properti yang disalin activity ke `MyServicePage`, belum pernah diuji terhadap sistem nyata | **Tim Infra** | Bila Kasir menuntut bentuk lain, yang berubah hanya `kasir/kasir.go` |
| **Otorisasi menu** (`TKT-F3-005`) belum ada | **Work Owner / DBA** | Setiap pengguna yang dapat masuk dapat membuka layar ini. Rute sudah berada di balik sesi; yang belum ada adalah pemeriksaan kewenangan |
| **Jejak audit** (`S-5`) belum ada | — | Perubahan master rekening belum tercatat siapa-kapan-dari apa-menjadi apa, padahal `TKT-F4-001` mensyaratkannya. `UPDATEBY` dan `TANGGALAPPROVEKOMITE` hanya menyimpan keadaan terakhir, bukan riwayat |
| **Kepemilikan tulis `LST_ACCOUNT`** | **Work Owner** | `P-1` menuntut satu tabel ditulis satu sistem. Layar Pega-nya wajib dimatikan pada saat modul ini dinyalakan, bukan sesudahnya |
| **Node 22.12+** di mesin pengembangan | **Work Owner / Tim Infra** | Uji frontend tidak dapat dijalankan; lihat `catatan-pengembangan.md` §9.8 |

### 11.8 Penyimpangan dari Steering yang disadari

**`TKT-U6-001` menuntut penghapusan lunak**, dan modul ini **tidak** menyediakan
penghapusan sama sekali — bukan lunak, bukan keras. `LST_ACCOUNT` tidak punya kolom
penanda hapus, dan menambahkannya menuntut DDL yang menyentuh tabel milik bersama
selama masa paralel (`P-1`, `ADR-0004`).

Penggantinya yang sudah ada: kolom `STS_AKTIF`. Rekening yang tidak dipakai lagi
**dinonaktifkan**, tetap terbaca, dan tetap dapat dirujuk klaim lama — yang secara
perilaku adalah apa yang dituntut penghapusan lunak. Layar menandainya secara terpisah
supaya rekening disetujui-tetapi-nonaktif tidak disalahbaca sebagai siap pakai.

Menambah kolom penanda hapus yang sesungguhnya menunggu DDL dan keputusan Work Owner.

### 11.9 Koreksi: dua sandi kolom yang sempat salah ditebak

Ditemukan saat penelusuran lanjutan atas `SetTipeRekening`, **setelah** implementasi
pertama selesai. Activity itu mengisi **dua** daftar pilihan sekaligus, dan keduanya
sempat saya salah baca:

| Kolom | Alias Pega | Sandi sebenarnya | Sempat saya tulis |
|---|---|---|---|
| `ACCOUNT_TYPE` | `CoverID` | **`"BIASA"` / `"VA"`** (Virtual Account) | `TERTANGGUNG` / `BENGKEL` / `RUMAH SAKIT` / `PIHAK KETIGA` |
| `STS_AKTIF` | `CaseID` | **`"Ya"` / `"Tidak"`** | `"1"` / `"0"` |

Sumbernya:

```
TempTipeBank.pxResults(<APPEND>).Telephone     = "BIASA" · "VA"      → TempBank.CoverID
TempTipeBank.pxResults(<APPEND>).NomorKontrak  = "Ya" · "Tidak"      → TempBank.CaseID
```

**Kenapa ini berbahaya dan tidak berisik.** Salah sandi `STS_AKTIF` tidak membuat apa
pun gagal — tidak ada galat, tidak ada baris yang ditolak. Ia hanya membuat **setiap
rekening terbaca sebagai nonaktif**, sehingga `dapat_dipakai` selalu `false` dan petugas
mengira seluruh rekening yang sah tidak dapat dipakai membayar klaim. Kegagalan diam
seperti itu baru ketahuan di produksi.

Salah nilai `ACCOUNT_TYPE` sama diamnya: layar akan menulis `"BENGKEL"` ke kolom yang
hanya dikenali Pega sebagai `"BIASA"` atau `"VA"`.

**Perbaikannya** — `sandiAktif` menulis `"Ya"`/`"Tidak"`; `bacaAktif` menerima
`Ya`/`1`/`Y`/`Aktif` tanpa peduli besar-kecil huruf, supaya baris lama yang ejaannya
berbeda tetap terbaca aktif; daftar tipe di formulir menjadi `BIASA` dan `VA`.

**Dan dikunci uji** — `repo/sqlstore/sandi_test.go` menguji ketiganya, termasuk pulang
pergi tulis-lalu-baca. Uji itu ada justru karena kesalahannya tidak menimbulkan galat:
yang tidak berisik harus diuji, bukan diandalkan pada pembacaan ulang.

**Pelajaran yang berlaku untuk modul berikutnya.** Sandi nilai kolom **tidak dapat
ditebak dari nama kolom maupun dari SQL** — SQL hanya menunjukkan kolomnya, bukan nilai
yang sah. Sumbernya adalah activity yang mengisi daftar pilihan layar (`Set*Value`,
`Set*`), dan itu wajib dibaca untuk setiap kolom berjenis kode.

### 11.10 Peringatan surel — menggantikan `SendEmailAlertRekening`

**Yang ditiru apa adanya:**

| Perilaku lama | Di sini |
|---|---|
| Surel dipicu **hanya** bila `ResponseCode == "9"` | Sama persis. Kode `"1"` tetap gagal **tanpa** surel |
| Dikirim setelah rekening disetujui komite lalu didaftarkan ke Kasir | Sama |

Satu pemicu yang sempat saya tambahkan — **Kasir tidak dapat dihubungi sama sekali** —
sudah **dicabut**. Work Owner menetapkan alur bisnis dipertahankan apa adanya dan tidak
ditambah apa pun di luar yang sudah ada (2026-09-17).

*Akibat yang disadari:* Kasir yang mati total **tidak** memicu surel. Kegagalannya tetap
terlihat di kolom Kasir pada layar dan di log aplikasi, tetapi tidak ada yang memberi
tahu secara aktif. Ini perilaku sistem lama, dan dicatat supaya tidak terbaca sebagai
kelalaian.

#### Satu-satunya penyimpangan di modul ini: penerimanya

Rule lama mencari alamat penerima dengan
`select email from pooldata.mst_user_teknik where operator_id = {TempIns.pyID}`, diisi
`OperatorID.pxInsName` — **operator yang sedang masuk**. Di sini penerimanya adalah
**mailbox Tim IT** dari konfigurasi (`SMTP_PENERIMA_PERINGATAN`).

**Riwayat keputusannya dicatat jujur, termasuk kesalahan saya.** Mula-mula saya
menyimpulkan jalur lamanya buntu karena pemetaan identitas HCC/HCQ ke `OPERATOR_ID`
belum ditetapkan (`ADR-0024` pertanyaan terbuka no. 4). **Kesimpulan itu terlalu cepat.**
`GetUserDetailsQuery` menunjukkan:

```sql
(select email from POOLDATA.MST_USER_TEKNIK where operator_id = "PYUSERIDENTIFIER")
  from DATAPEGA.pr_operators where "PYUSERIDENTIFIER" = {GetUserKlaim.source}
```

`MST_USER_TEKNIK.OPERATOR_ID` ternyata sama dengan `PYUSERIDENTIFIER` Pega — yaitu
**nama login**, yang sudah disimpan aplikasi baru di `Pengguna.Login`. Jadi jalur as-is
kemungkinan besar dapat dipakai; yang kurang hanya satu query verifikasi kecocokannya.

Temuan itu disampaikan ke Work Owner, dan Work Owner **tetap memilih Tim IT**
(2026-09-17) — kini dengan mengetahui bahwa jalur as-is tersedia. Alasannya: peringatan
kegagalan integrasi ditujukan ke pihak yang dapat **memperbaikinya**, bukan ke orang
yang kebetulan menekan tombol approve.

Komite yang memutuskan tetap disebut **di dalam badan surel** sebagai keterangan, supaya
Tim IT tahu kepada siapa harus bertanya.

> **Bila kelak hendak dikembalikan ke as-is**, yang dibutuhkan hanya satu seam baru
> (`EmailOperator(operatorID) string`) berisi kueri di atas, lalu mengganti sumber
> `Peringatan.Kepada`. Pemicu, isi, dan pengirimannya tidak berubah.

#### Yang tidak dapat direproduksi

Rule HTML `EmailAlertRekeningToPIC` **tidak ada di dalam export** — hanya pemanggilannya
yang terlihat. Badan suratnya disusun ulang, dan surelnya sendiri menyatakan bahwa
susunannya **sementara**. Menunggu wording resmi dari tim bisnis.

#### Dua langkah rule lama yang tidak ditiru

| Langkah lama | Kenapa tidak ditiru |
|---|---|
| Membaca ulang `POOLDATA.CLAIM_SERVICE_LOG` untuk mengambil `ResponseCode` | Kode responsnya sudah di tangan sebagai nilai balik panggilan Kasir. Lagipula SQL-nya, `rownum=1 order by insertdate desc`, memotong baris **sebelum** mengurutkan sehingga tidak menjamin baris terbaru |
| Prasyarat `TempClaimAttach.FlagNOLL=="Ya"` pada langkah kirim, berketerangan *"skip jika error karena sudah ada di kasir dan belum ada di lst account"* | Maksud bisnisnya tidak dapat dipastikan dari export. Work Owner menetapkan: **kirim selalu saat gagal kode 9**. Dicatat sebagai lubang paritas yang diketahui |

#### Yang belum dibangun dan memang belum diperlukan

Sistem lama menulis setiap panggilan service ke `POOLDATA.CLAIM_SERVICE_LOG`. Modul ini
tidak. Respons Kasir sudah tersimpan di `STS_SERVICE` dan `RESPONSE_KASIR` pada barisnya
sendiri dan tampil di layar; log terpusat menyusul bersama modul `S-4` Integrasi Sistem
Luar (keputusan Work Owner 2026-09-17).

#### Keamanan pengiriman

Tiga hal ditegakkan di kode, bukan diserahkan ke konfigurasi:

- **Kredensial SMTP hanya dikirim setelah STARTTLS berhasil.** Server yang tidak
  menawarkan STARTTLS membuat pengiriman ditolak, bukan dilanjutkan tanpa enkripsi.
- **Nilai header dipotong pada baris baru pertama.** Satu `\r\n` di dalam alamat cukup
  untuk menyisipkan header tambahan — termasuk penerima tambahan. Ujinya menemukan
  kelemahan nyata pada versi pertama, yang hanya mengganti baris baru dengan spasi
  sehingga teks susupan tetap ikut terkirim di dalam header.
- **Seluruh nilai dari data di-escape** sebelum masuk badan HTML. Nama pemilik rekening
  dan pesan dari Kasir adalah teks yang dimasukkan pihak lain.
## 12. Master Status Klaim — modul bisnis pertama (2026-09-17)

Sampai sesi ini belum ada satu pun modul bisnis. Karena itu setiap keputusan di bawah bukan hanya
tentang satu layar: ia menjadi pola untuk sekurang-kurangnya **28 master berikutnya**.

### 12.1 Keputusan Work Owner pada sesi ini

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Menulis ke mana | **Go jadi penulis tunggal `POOLDATA.M_STS_CLAIM`**, tidak lagi menyimpan JSON; isi JSON dipindahkan ke kolom. Procedure `PEGA_M_STS_CLAIM` boleh ditinggalkan |
| 2 | Lingkup | Fungsi dan tampilan **seperti Pega**, tetapi lebih bagus, mobile friendly, dan user friendly |
| 3 | Jejak audit | **Samakan dengan sekarang** — sistem lama tidak punya, jadi tidak ditambahkan |
| 4 | Validasi | **Tolak ID atau nama status ganda, dan tolak yang kosong** |

### 12.2 Kepemilikan tabel: kenapa `P-1` tidak dilanggar

`P-1` berbunyi *satu tabel hanya boleh ditulis satu sistem* selama masa paralel — bukan bahwa tabel
lama tidak boleh ditulis sama sekali.

Layar Master Status Klaim adalah **satu-satunya penulis** `M_STS_CLAIM` di sistem lama: hanya
`RDB List/UpdateStsClaim-SQL.xml` yang memanggil `PEGA_M_STS_CLAIM`, dan hanya layar itu yang
memanggil rule tersebut. Memindahkan layarnya karena itu memindahkan kepemilikan tabelnya **secara
utuh**. Pega berubah menjadi pembaca saja lewat `V_STS_CLAIM`, yang dibaca 23 rule-nya.

**Alternatif yang ditolak: tabel baru milik Go.** Ia lebih bersih secara skema, tetapi Pega tetap
membaca `V_STS_CLAIM` — sehingga akan ada **dua master yang menyimpang** begitu ada perubahan,
kecuali dibangun sinkronisasi dua arah yang justru dilarang `P-1`. Satu sumber kebenaran menang atas
skema yang lebih rapi.

### 12.3 Skema kode warisan dipertahankan apa adanya

Kode dibentuk `id_site` disambung nomor urut tiga digit, persis seperti `PEGA_M_STS_CLAIM.prc`.
Dengan situs `1` dan urutan 134 sampai 166, hasilnya tepat `1134` sampai `1166`.

Tidak diganti dengan skema yang lebih baik, dan itu keputusan sadar: **23 rule Pega membaca kode ini
selama masa paralel**, dan klaim lama menyimpannya. Skema baru berarti dua sistem penomoran hidup
berdampingan tanpa alasan.

Yang **diperbaiki** hanya pembentukannya: perangkaian dan pemformatan angka dikerjakan di Go, bukan
lewat `LPAD` dan `TO_CHAR` di SQL — keduanya dilarang `09-DATABASE-STRATEGY.md` §4 karena mengikat
kueri pada dialek Oracle.

**Batas yang diwarisi, dan tidak ditutupi.** `LSC_ID` bertipe `CHAR(4)`. Saat urutan mencapai 1000,
kodenya menjadi lima karakter dan penyisipan **ditolak** dengan `ORA-12899`. Memotongnya menjadi
tiga digit akan menghasilkan **kode ganda** — jauh lebih buruk daripada penyisipan yang gagal dengan
pesan jelas. Perilakunya dibiarkan, batasnya dicatat: urutan berada di **193** pada 2026-09-17,
menyisakan sekitar **806 penambahan**.

### 12.4 `FROM DUAL` — pengecualian dialek yang kedua, dan dipagari

`09-DATABASE-STRATEGY.md` §4 melarang `FROM DUAL` karena tidak ada padanannya di PostgreSQL.
Pengambilan `NEXTVAL` menuntutnya.

Pengecualiannya diperlakukan sama dengan generator nomor klaim pada `ADR-0005`: **satu kueri
bernama, diisolasi**, dan dipagari uji `TestFromDualHanyaDiKueriUrutan` yang **gagal bila ada kueri
kedua** memakainya. Disiplin yang hanya ditulis di dokumen akan dilanggar pada bulan ketiga; yang
dipagari uji tidak.

### 12.5 Jejak audit tidak dibangun — penyimpangan yang disadari

`TKT-F4-001` menuntut *"setiap perubahan master menghasilkan tepat satu baris jejak audit dengan
nilai sebelum dan sesudah"*. **Tidak dibangun**, atas keputusan Work Owner: sistem lama tidak
mencatat apa pun, dan yang diminta adalah menyamakannya.

**Konsekuensinya dicatat terbuka, bukan disembunyikan:**

1. Perubahan nama status **tidak dapat ditelusuri** — siapa mengubahnya, kapan, dari apa menjadi
   apa. Nama status yang diubah langsung terbaca 23 rule Pega, termasuk laporan TAT dan KPI yang
   dibaca manajemen.
2. `D-59` menghapus pemisahan tugas dan menjadikan jejak audit **satu-satunya kontrol pengimbang**
   yang tersisa. Di modul ini kontrol itu belum ada.
3. Menambahkannya kelak menyentuh **seluruh** master, bukan hanya yang ini — itulah sebabnya
   `TKT-F4-001` menempatkannya di kerangka, bukan di masing-masing master.

Bila `S-5` dibangun kemudian, tempat menyisipkannya sudah jelas: lapisan `usecase`, di dalam
transaksi yang sama dengan penyimpanan.

### 12.6 Tiga aturan validasi, dan yang sengaja tidak ada

Layar Pega **tidak memvalidasi apa pun**: `pyRequired=false`, tanpa batas panjang, tanpa pemeriksaan
keunikan. Work Owner memutuskan tiga aturan baru.

| Aturan | Di mana ditegakkan | Kenapa di situ |
|---|---|---|
| Label wajib diisi | Zod di layar **dan** domain Go | Layar menjawab tanpa perjalanan jaringan; domain menegakkan, karena pemanggilan langsung ke API tidak melewati layar |
| Label tidak boleh ganda | Pemeriksaan di `usecase` **dan** indeks unik basis data | Pemeriksaan memberi pesan yang jelas; indeks yang menjamin. Dua permintaan bersamaan dapat sama-sama lolos pemeriksaan — hanya indeks yang tidak dapat ditembus |
| Kode tidak boleh ganda | Kunci utama `M_STS_CLAIM_PK` | Kode dibuat urutan, jadi bentrok seharusnya mustahil. Penerjemahannya ada supaya kemustahilan itu **terlihat** bila terjadi, bukan menimpa baris lain |

**Batas panjang 100 karakter** bukan permintaan Work Owner melainkan akibat kolomnya:
`LSC_NOTE VARCHAR2(100)`. Tanpa batas di aplikasi, label yang kepanjangan akan ditolak basis data
sebagai galat `500` alih-alih pesan yang dapat diperbaiki pengguna.

Perbandingan keunikan mengabaikan besar-kecil huruf dan spasi tepi — `Paid` dan `PAID  ` adalah
status yang sama bagi pengguna. Ekspresinya sama persis di kedua tempat: `masterstatus.KunciLabel`
di Go dan `UPPER(TRIM(LSC_NOTE))` di indeks. Bila keduanya berbeda, aplikasi akan menerima label
yang kemudian ditolak basis data.

**Yang sengaja tidak ada: Hapus.** Layar Pega tidak punya tombol hapus, procedure-nya hanya mengenal
INSERT dan UPDATE, dan `ADR-0012` melarang master dihapus permanen karena klaim lama merujuknya.
Ketiadaan itu **dikunci tiga uji** — seam tanpa metode `Hapus`, rute `DELETE` menjawab `405`, dan
tidak ada kueri yang memuat `DELETE` — supaya penambahannya menjadi keputusan sadar, bukan
kelalaian yang lolos review.

### 12.7 Kenapa tanpa pustaka tabel

`ADR-0002` sengaja meninggalkan pilihan TanStack Table versus AG Grid **terbuka**, dan `TKT-U2-005`
menuntut keputusannya diambil dengan **angka** — karena koreksi ukuran (median grid ternyata 6
kolom, bukan 18 sampai 27) melemahkan alasan memilih pustaka kelas berat.

Menarik salah satunya sekarang berarti mendahului keputusan itu. `TabelData` karena itu dibuat
sesempit mungkin: cari, urut, tiga keadaan tampilan. Bila pustaka kelak dipilih, yang diganti adalah
**isi satu berkas** — bukan setiap layar yang memakainya.

**Pencarian dan pengurutan dikerjakan di peramban**, dan itu keputusan berbasis angka, bukan
kemalasan: masternya 33 baris dan bertambah beberapa baris per tahun. Menyaring 33 baris di server
berarti satu perjalanan jaringan untuk setiap huruf yang diketik, tanpa satu pun manfaat. Layar yang
datanya besar — inbox dan laporan — **tidak boleh** mengikuti pola ini; keduanya menuntut paginasi
keyset dari server (`D-10`), dan itu lingkup `TKT-U2-001`.

### 12.8 Satu DOM untuk meja dan kartu

Versi pertama `TabelData` menggambar dua pohon: `<table>` untuk layar lebar, daftar kartu untuk
layar sempit, masing-masing disembunyikan bergantian dengan kelas Tailwind.

Itu salah, dan pengujian yang membuktikannya: kelas Tailwind hanya menyembunyikan lewat CSS,
sehingga **kedua pohon tetap ada di DOM**. Setiap isi sel muncul dua kali, pembaca layar membacanya
dua kali, dan 14 dari 15 uji gagal karena setiap pencarian menemukan dua elemen untuk satu nilai.

Yang dipakai sekarang: **satu `<table>`** yang elemennya diubah menjadi blok lewat CSS pada layar
sempit. Nama kolom digambar ulang di dalam sel sebagai label kecil yang hilang pada layar lebar, dan
label itu `aria-hidden` karena `<th scope="col">` sudah menjelaskan selnya.

### 12.9 Kontrak API modul ini

| Metode | Jalur | Jawaban |
|---|---|---|
| `GET` | `/api/master/status-klaim` | daftar + `total` |
| `POST` | `/api/master/status-klaim` | `201` + baris beserta kode yang dibuat sistem |
| `GET` | `/api/master/status-klaim/{kode}` | satu baris |
| `PUT` | `/api/master/status-klaim/{kode}` | `200` + baris setelah diubah |

**Kode tidak pernah datang dari klien.** Pada penambahan ia dibuat penyimpanan; pada pengubahan ia
di jalur URL. Sentinel `"UnknownID"` yang dipakai Pega tidak dibawa sama sekali.

**`PUT`, bukan `PATCH`:** seluruh isi yang boleh diubah — satu field — dikirim setiap kali, sehingga
permintaannya menggantikan dan **idempoten**. Diuji: dua permintaan identik menghasilkan jawaban
yang sama persis.

Kode galat baru, dan kenapa statusnya berbeda-beda:

| Kode | HTTP | Kenapa bukan yang lain |
|---|---|---|
| `validasi_gagal` | `422` | Permintaannya berbentuk benar, isinya yang melanggar aturan bisnis. `400` berarti bug frontend; `422` berarti kesalahan pengguna yang harus ditandai di kolomnya |
| `label_status_sudah_dipakai` | `409` | Isian penggunanya sah, tetapi bentrok dengan keadaan penyimpanan — mungkin karena orang lain baru saja memakai nama itu |
| `kode_status_sudah_dipakai` | `409` | idem, dan seharusnya mustahil |
| `status_klaim_tidak_ditemukan` | `404` | — |

Galat validasi membawa `detail` berisi **seluruh** pelanggaran beserta nama field-nya. Mengembalikan
satu per satu akan membuat pengguna menekan Simpan berkali-kali untuk menemukan kesalahan
berikutnya — dan sistem lama menampilkan semuanya sekaligus.

### 12.10 Otorisasi: keadaan yang belum berubah

Rute modul ini **terlindungi sesi**, tetapi **belum diperiksa perannya**. `D-59` menetapkan satuan
izin adalah menu, dan penegakan "apakah peran pemanggil memiliki menu Master Data" adalah
`TKT-F3-005` — yang bergantung pada tabel peran `TKT-F3-004`, yang dapat dibangun tetapi **belum
dapat diisi** karena penugasan operator ke peran tidak ada di basis data maupun di export.

Keadaan ini sama dengan seluruh rute lain hari ini. Yang berubah: sekarang ada rute yang **menulis
master**, sehingga taruhannya naik. Daftar menu di `app/KerangkaHalaman.tsx` juga masih tetap —
setiap pengguna yang masuk melihat menu yang sama.

### 12.11 Yang berubah di luar modul baru

Isolasi modul Login, Home, dan Portal dipatuhi. Lima berkas bersama ikut berubah, seluruhnya
penambahan:

| Berkas | Perubahan | Kenapa tidak dapat dihindari |
|---|---|---|
| `api/klien.ts` | `PUT` ditambahkan ke daftar metode; `GalatAPI` membawa `detail` | Pengubahan master menuntut `PUT`; `detail` yang membuat galat dapat ditandai per kolom. `DELETE` **sengaja tidak** ditambahkan |
| `api/tipe.ts` | tipe `StatusKlaim` dan empat kode galat baru | Cerminan DTO Go; pemeriksaan tipe adalah jaring pengaman antara layar dan API |
| `app/App.tsx` | satu rute + pembungkus `KerangkaHalaman` | Titik pasang modul, setara `main.go` di backend |
| `cmd/claimpnc/main.go` | perakitan modul dan pemasangan rute | idem |
| `cmd/claimpnc/periksa.go` | laporan kesiapan `M_STS_CLAIM` | Membedakan "migrasi belum jalan" dari "tidak punya hak baca" |

`HalamanBeranda.tsx` **tidak disentuh** — menu dipasang di `app/`, bukan di dalam modul beranda,
karena modul tidak boleh saling mengimpor.

### 12.12 Pertanyaan terbuka yang ditinggalkan sesi ini

| Pertanyaan | Pemilik | Menahan |
|---|---|---|
| **Kenapa basis data memuat 32 baris sementara CSV memuat 33?** Kode `1165` "Rejected Chasier" tidak ada di `POOLDATA.M_STS_CLAIM` | Work Owner + DBA | Tidak menahan pembangunan; menahan pernyataan "master memuat tepat 33 kode" |
| Persetujuan menjalankan migrasi `0002` | Work Owner + DBA (`D-63`) | Layar bekerja terhadap Oracle |
| Hak `INSERT`/`UPDATE` akun aplikasi atas `M_STS_CLAIM`, dan hak baca atas `M_SITE_DATABASE` serta urutan | DBA | Penambahan dan pengubahan terhadap Oracle |
| Apakah `LSC_ID` diperlebar sebelum urutan mencapai 1000 | Work Owner + DBA | Tidak mendesak — sekitar 806 penambahan lagi |
| Kapan `JSONDATA` boleh dibuang | Work Owner | Tidak menahan apa pun; sebaiknya setelah masa pengamatan |

---

## 13. Sistem desain antarmuka (2026-09-17, sesi kelima)

### 13.1 Keputusan Work Owner

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Layar mana yang didesain ulang | **Seluruh aplikasi**, termasuk Masuk dan Beranda |
| 2 | Warna aksen | **Blue** (`blue-600`), slate sebagai dasar. Mula-mula indigo; diganti atas permintaan susulan hari yang sama karena indigo condong ke ungu |
| 3 | Mode tampilan | **Light Mode saja** |

Jawaban 1 **mencabut aturan isolasi untuk urusan tampilan**. Modul Login dan Home boleh disentuh
kelas Tailwind-nya; alur, validasi, penanganan galat, dan logikanya tetap tidak boleh diubah — dan
33 uji yang lulus tanpa dilonggarkan adalah buktinya.

### 13.2 Kenapa bukan merah korporat

Merah Sinar Mas ditawarkan sebagai pilihan dan tidak dipilih. Alasannya bukan selera:

Merah adalah bahasa universal untuk galat dan bahaya. Bila ia menjadi warna tombol utama, tombol
**Simpan** berwarna merah akan berdiri di sebelah **pesan galat** berwarna merah, dan keduanya sulit
dibedakan sekilas — persis pada saat pengguna paling perlu membedakannya.

Bila merek kelak menuntutnya, warna galat harus digeser lebih dulu (misalnya ke rose tua), bukan
sesudahnya.

### 13.3 Token, bukan kelas yang diulang

Seluruh nilai desain hidup di `src/gaya.css` sebagai token Tailwind v4 (`@theme`). Yang ditaruh di
sana hanya yang **berulang di banyak layar**; sisanya tetap kelas Tailwind biasa.

| Token | Kenapa ia layak menjadi token |
|---|---|
| Tiga tangga bayangan + satu bayangan aksen | Dipakai di kartu, tabel, form, bilah atas, dan tombol. Nilainya **dua lapis** — satu rapat untuk tepi, satu lebar untuk ketinggian; satu lapis terlihat "ditempel" |
| Dua tangga lengkung (`kartu`, `kontrol`) | Membatasi pilihan. Tanpa batas, satu layar bisa memuat tiga radius berbeda tanpa ada yang menyadarinya |
| Satu lengkung gerak (`--ease-halus`) | Seluruh transisi memakai kurva yang sama, sehingga aplikasi terasa satu benda |

**Warnanya memakai palet bawaan Tailwind (blue, slate), bukan warna karangan.** Nilainya sudah ada
di pustaka sehingga tidak dapat salah ketik, dan kontrasnya sudah teruji.

### 13.4 Tiga keadaan yang wajib terlihat pada setiap kontrol

Bukan hanya pada tombol utama:

| Keadaan | Yang terjadi | Kenapa |
|---|---|---|
| `hover` | warna menua, bayangan melebar, naik 1px | Memberi tahu bahwa benda itu dapat ditekan |
| `active` | turun kembali, menyusut 98% | Umpan balik antara menekan dan hasilnya muncul. Tanpanya tombol terasa mati pada jaringan lambat |
| `focus-visible` | cincin 4px beropasitas rendah | Satu-satunya cara pengguna papan ketik tahu ia ada di mana |

**`focus-visible`, bukan `focus`.** Memakai `focus` membuat cincin ikut muncul setiap kali tombol
diklik tetikus, yang terlihat seperti cacat tampilan — dan berujung pada orang menghapus cincinnya
sama sekali, termasuk bagi pengguna papan ketik yang benar-benar membutuhkannya.

### 13.5 Gerak dimatikan bila pengguna memintanya

`prefers-reduced-motion: reduce` mematikan seluruh transisi dan animasi. Ini **tidak diminta**
Work Owner, dan tetap dikerjakan.

Alasannya: permintaannya adalah "transisi yang halus", dan bagi sebagian orang transisi menimbulkan
pusing atau mual. Sistem operasinya sudah menyatakan itu. Mengabaikannya berarti membuat aplikasi
tidak dapat dipakai bagi mereka — yang membatalkan maksud permintaannya sendiri.

Transisi **dimatikan**, bukan dipercepat: nol lebih aman daripada nyaris nol.

### 13.6 Kontras tidak berhenti di warna

Warna saja tidak terbaca oleh sekitar satu dari dua belas laki-laki yang mengalami buta warna
merah-hijau. Setiap pembedaan yang menentukan tindakan karena itu ditandai **lebih dari satu cara**:

| Pembedaan | Cara menandainya |
|---|---|
| Isian salah | warna tepi **+** ikon **+** teks pesan **+** `aria-invalid` |
| Penolakan versus gangguan | warna **+ BENTUK ikon** (lingkaran versus segitiga) |
| Portal siap versus belum | warna **+** titik **+** teks |
| Kolom sedang diurutkan | warna **+** arah panah |

### 13.7 Tiga hal yang pindah ke bilah atas

Tombol **Keluar**, **pemilih portal**, dan **nama pengguna** pindah dari halaman Beranda ke bilah
atas aplikasi.

Alasannya bukan estetika melainkan cacat nyata: ketiganya berlaku untuk seluruh layar di balik
sesi, dan selama ia hidup di dalam Beranda, **pengguna yang sedang membuka layar master tidak punya
cara keluar** tanpa kembali ke beranda lebih dulu.

Akibat yang harus dikerjakan bersamaan: baris "Nama" pada kartu identitas Beranda **dihapus**.
Menyisakannya membuat nama yang sama muncul dua kali di satu layar — dan membuat uji beranda yang
mencarinya dengan pencocokan persis gagal menemukannya.

### 13.8 Yang sengaja tidak dipakai

| Tidak dipakai | Alasan |
|---|---|
| **Google Fonts** | Aplikasi berjalan di VM on-premise tanpa jaminan akses internet (`D-08`). Huruf yang gagal dimuat mengubah seluruh tata letak. Dipakai tumpukan font sistem |
| **Pustaka ikon** | Delapan bentuk sederhana tidak sebanding dengan satu dependensi yang harus dipelajari tim (`D-09`), dipantau keamanannya, dan ikut membesarkan bundel |
| **Pustaka tabel** | `TKT-U2-005` menuntut keputusannya diambil dengan pengukuran, bukan kesan. Belum berubah |
| **Mode gelap** | Light Mode saja, keputusan Work Owner. `color-scheme: light` ditegaskan supaya kontrol bawaan peramban tidak ikut membalik mengikuti tema sistem |
| **Menu hamburger** | Dengan dua entri, ia menambah satu ketukan untuk menyembunyikan sesuatu yang muat. Perlu ditinjau ulang bila menu kelak berasal dari izin peran (`TKT-F3-004`) dan bertambah banyak |

### 13.9 Yang berubah, dan yang tidak

**Berubah:** seluruh berkas antarmuka — kelas Tailwind, susunan elemen, dan penempatan tiga kontrol
di §12.7.

**TIDAK berubah:**

- Satu pun endpoint, DTO, atau kontrak API.
- Satu pun aturan bisnis, validasi, atau penanganan galat.
- Satu baris pun kode backend.
- Satu pun uji dilonggarkan. Yang disunting hanya **fixture** uji layar master, supaya peladen
  tiruannya menjawab `/api/portal` — panggilan yang memang baru muncul karena pemilih portal pindah
  ke bilah atas.

---

## 14. Penamaan kode berbahasa Inggris (2026-09-18, sesi keenam)

### 14.1 Batas yang ditetapkan — dan kenapa batasnya yang penting

`D-80` menetapkan seluruh nama di dalam kode memakai bahasa Inggris. Yang membuat keputusan itu
dapat dijalankan tanpa merusak apa pun adalah **lima pengecualiannya**, bukan aturannya:

| Tetap Indonesia | Sifatnya | Bila dilanggar |
|---|---|---|
| Nama field JSON API | kontrak | klien rusak; ini perubahan yang merusak, bukan penggantian nama |
| Nama tabel & kolom | dimiliki bersama Pega (`D-21`) | menempuh `D-63`; satu `ALTER` keliru menghentikan produksi |
| Komentar & dokumen | penjelasan untuk tim (`D-09`) | penjelasan yang tidak dibaca |
| Teks layar yang ada di XML Pega | `D-13` | pengguna harus belajar ulang |
| Variabel lingkungan & flag baris perintah | antarmuka operator | berkas `.env` dan skrip deployment yang sudah berjalan rusak |

Empat yang pertama dijawab Work Owner di muka. **Yang kelima muncul saat pekerjaan berjalan** dan
diputuskan di tempat — dicatat di sini, dan dinaikkan ke `D-80` serta §4.1, supaya ia terbaca
sebagai keputusan dan bukan sebagai sesuatu yang terlewat.

### 14.2 Keputusan penamaan yang tidak sepele

| Hal | Pilihan | Alasan |
|---|---|---|
| `masterrekening` | **`bankaccount`** | "master" adalah jenis data, bukan isinya. Yang dimodelkan adalah rekening bank |
| `masterstatus` | **`claimstatus`** | idem; `CONTEXT.md` menyebutnya Status Klaim |
| `Kasir` | **`Cashier`** untuk seam Go, **`Kasir`** di dalam prosa komentar | Go-nya identifier; prosanya menyebut sistem eksternal sebagaimana bisnis menyebutnya |
| `Rute` | **`AppRoute`**, bukan `Route` | `Route` bertabrakan dengan `react-router` — satu-satunya tabrakan nama pihak ketiga yang ditemukan |
| `Pencarian` (tipe) vs `cari` (state) | **`SearchBox`** dan **`query`** | keduanya "search" bila diterjemahkan lurus; membedakannya menjaga keduanya tetap terbaca di satu berkas |
| `KodeGalat.isianTidakSah` | **`ErrorCode.invalidInput`** dengan nilai tetap `'isian_tidak_sah'` | kunci adalah kode; nilainya kontrak |
| `StatusRekening.menunggu` | **tidak diganti** | `menunggu`, `disetujui`, `ditolak` juga muncul sebagai teks layar; mengganti identifiernya berisiko merusak teks, dan imbalannya kecil |
| `-periksa` (flag) | **tidak diganti**; fungsinya `check()` di `check.go` | lihat §14.1 baris kelima |

### 14.3 Kenapa penggantian dikerjakan pemindai, bukan `sed`

`sed` dengan batas kata merusak tiga hal yang tidak boleh disentuh: komentar, literal string, dan
teks JSX. Dua di antaranya **tidak terdeteksi kompilator** — kode tetap dibangun, dan yang berubah
hanya arti kalimat yang dibaca manusia atau nilai data yang dibandingkan.

Yang dipakai adalah pemindai yang memecah berkas menjadi potongan kode / bukan-kode lebih dulu.
Ia tetap tidak sempurna: **teks JSX dan literal regex** bagi pemindai adalah kode. Keduanya
ditangkap suite uji frontend, bukan oleh pembacaan ulang.

> Pelajaran yang sama terulang dari sesi sebelumnya: **alat ukur dipercaya sebelum divalidasi.**
> Bedanya kali ini jaringnya sudah terpasang — 42 uji frontend yang memeriksa teks layar apa adanya.

### 14.4 Temuan di luar lingkup — tabrakan nama tombol di layar Master Rekening

Tiga uji `HalamanMasterRekening` gagal, **dan sudah gagal sebelum sesi ini** (dibuktikan dengan
menjalankan suite pada `git worktree` di `HEAD`: hasilnya sama persis).

Sebabnya nyata dan bukan soal uji:

```
Tab layar   : Cari Data Rekening · Komite Approval · Waiting Approval · Approve · Reject
Tombol aksi : Approve · Reject
```

Tab dan tombol aksi memakai **nama yang sama persis**. Akibatnya:

1. `getByRole('button', { name: 'Approve' })` menemukan **tab**, bukan tombol aksi.
2. Uji yang mengklik `getAllByRole(...)[0]` berpindah tab alih-alih menyetujui rekening — sehingga
   permintaan `POST /keputusan` tidak pernah terkirim.

**Ini bukan sekadar cacat uji.** Pengguna papan ketik dan pembaca layar menghadapi hal yang sama:
dua kendali berbeda dengan nama yang tidak dapat dibedakan pada satu layar.

Tidak diperbaiki pada sesi ini — perbaikannya menyentuh label layar, dan `D-13` menetapkan label
mengikuti Pega. Yang diperlukan adalah keputusan Work Owner: memberi tab `aria-label` yang
membedakannya (mis. "Tab Approve"), atau mengubah label tombol aksinya. Diangkat sebagai pertanyaan
terbuka.

### 14.5 Cacat tipe lama yang terpaksa diperbaiki

`GalatAPI.field` tidak pernah ada; yang ada `detail: PelanggaranField[]`. Dua berkas memanggilnya,
dan keduanya menghalangi `tsc` setelah penggantian nama. Diperbaiki menjadi pembacaan `detail`.

Konsekuensi yang perlu disadari: **pesan galat per kolom pada form Master Rekening sebelumnya tidak
pernah tampil** — `Object.entries(undefined)` melempar, dan efeknya tertelan. Sesudah perbaikan ini,
kolom yang ditolak server ditandai di tempatnya sebagaimana dirancang.

### 14.6 Yang belum dikerjakan

| Hal | Alasan |
|---|---|
| Tabrakan nama tab/tombol di Master Rekening | menunggu keputusan Work Owner — §14.4 |
| `StatusRekening.{menunggu,disetujui,ditolak}` masih Indonesia | §14.2 |
| Variabel lingkungan masih Indonesia | disengaja — §14.1 |

### 14.7 Koreksi `D-81` — nama modul justru dikembalikan ke bahasa Indonesia

Satu jam setelah `D-80` dijalankan, Work Owner meminta nama modul memakai **nama bisnisnya**:
`master-rekening` dan `master-status-klaim`, bukan `bank-account` dan `claim-status`.

**Ini bukan pembatalan `D-80`, melainkan penerapannya yang lebih tepat.** `D-80` sudah menetapkan
lima hal tetap berbahasa Indonesia, dan seluruhnya punya satu ciri yang sama: **dipakai orang di
luar kode** — kontrak API, kolom basis data, teks layar, variabel lingkungan. Nama modul masuk
kategori yang sama dan **terlewat** saat `D-80` disusun: ia dipakai Work Owner saat memesan
pekerjaan, dan tertulis di `docs/ticketing/`.

| Hal | Bahasa | Alasan |
|---|---|---|
| **Nama folder & paket modul** | **Indonesia** | Dipakai Work Owner dan tiket — `masterrekening`, `master-rekening` |
| Isi modul — tipe, fungsi, field, variabel | **Inggris** | `D-80` tidak berubah — `Account`, `Check()`, `Number` |

**Batas yang dipegang saat menjalankannya.** Godaan terbesar adalah ikut menerjemahkan isi modul
kembali ke bahasa Indonesia. Itu **tidak dilakukan**: `masterrekening.Account` tetap `Account`, dan
`BankAccountPage` menjadi **`AccountPage`** — mengikuti nama tipe domain, bukan nama modul. Bila
komponen ikut memakai nama modul, hasilnya `MasterRekeningPage` di dalam `master-rekening/`, yang
mengulang nama tanpa menambah keterangan apa pun.

**Aturannya ditulis supaya tidak perlu ditanyakan lagi:** nama modul **disebutkan Work Owner di
prompt**, tidak diterjemahkan dan tidak dikarang. Tercatat di `D-81` dan `08-TECHNICAL-STRATEGY.md`
§4.1, sehingga `CLAUDE.md` membawanya ke setiap sesi berikutnya.

**Yang perlu disadari saat membaca kode:** satu jalur berkas kini memuat dua bahasa —
`internal/masterrekening/repo/sqlstore/account.go`. Itu disengaja, dan pemisahannya tegas: segmen
pertama nama modul, sisanya isi modul.

---

## 15. Merge yang belum selesai, dan Master Status Progres ke standar baru (2026-09-18, sesi ketujuh)

### 15.1 Koreksi atas laporan saya sendiri di sesi sebelumnya

Dua hal yang saya laporkan pada §14 ternyata tidak benar, dan keduanya diperbaiki di sesi ini.

**Pertama — "seluruh pemeriksaan bersih".** Itu benar untuk keadaan repo saat verifikasi
dijalankan. Sesudahnya, cabang `Push Master Status Progress 1` digabungkan dan **commit merge-nya
disimpan dengan konflik belum diselesaikan**. Sejak saat itu repo tidak dapat di-build sama sekali,
dan laporan "bersih" menjadi menyesatkan bila dibaca sebagai keadaan repo hari ini.

**Kedua — perbaikan `GalatAPI.field` pada §14.5 salah arah.** Saya mengganti pembacaan
`error.field` menjadi `error.detail`, dengan alasan `field` tidak ada di kelasnya. Yang benar
sebaliknya: **backend modul Master Rekening memang mengirim `field`**, dan yang hilang adalah
propertinya di kelas `APIError` — properti itu ikut terhapus saat merge.

Bentuk yang sebenarnya dikirim ketiga modul master, dan ketiganya berbeda:

| Modul | Bentuk | Bukti |
|---|---|---|
| `masterstatus` | `detail: [{ field, pesan }]` | `internal/masterstatus/http/dto.go:55` |
| `masterstatusprogres` | `detail: [{ kolom, pesan }]` | `internal/masterstatusprogres/http/dto.go:82` |
| `masterrekening` | `field: { kolom: pesan }` | `internal/masterrekening/http/dto.go:151` |

Akibat kekeliruan saya: pesan galat per kolom pada form Master Rekening membaca senarai yang
**selalu kosong**, sehingga isian yang ditolak server tidak pernah disorot. Layarnya tidak error —
ia hanya diam, dan itu kelas kegagalan yang paling sulit terlihat.

**Perbaikannya:** `APIError` kembali memegang `detail` DAN `field`, ditambah satu method
`violations()` yang menyatukan keduanya menjadi peta `kolom → pesan`. Layar memanggil `violations()`
dan tidak lagi perlu tahu modul mana memakai bentuk yang mana. Ketika `TKT-F1-004` menyeragamkan
kontraknya kelak, yang berubah hanya `api/client.ts`.

### 15.2 Konflik merge: mana yang diambil, dan atas dasar apa

| Berkas | Keputusan |
|---|---|
| `cmd/claimpnc/main.go` (9 blok) | ambil sisi HEAD (berbahasa Inggris), lalu **pasang ulang** rakitan Status Progres dalam bahasa Inggris |
| `internal/masterrekening/masterrekening.go` | ambil nama Inggris HEAD, ambil **nomor bab `§11.10`** dari sisi cabang — HEAD menunjuk `§10.10` yang sudah bergeser |
| `internal/auth/provider/tiruan.go` | **dihapus** — kembar lama `fake.go` yang hidup lagi |
| `README.md` | gabungkan: baris portal versi Inggris + blok modul baru, ditulis ulang sebagai `masterstatusprogres` |
| `docs/*.md` (27 blok) | ambil sisi yang terisi; bila keduanya terisi, ambil penomoran bab dari sisi cabang |

**Kenapa penomoran bab dokumen mengikuti sisi cabang.** Cabang itu menyisipkan bab baru
(Master Status Progres 1) sebagai `§10`, menggeser Master Rekening ke `§11` dan seterusnya. Doc
comment di `masterrekening.go` sudah menunjuk `§11.10`, sehingga penomoran cabanglah yang konsisten
dengan kode. `catatan-pengembangan.md` ikut dirapikan karena merge meninggalkan **dua bab `## 9.`**.

### 15.3 Kenapa modulnya `masterstatusprogres`, bukan `progressstatus`

`D-81` menetapkan nama folder modul memakai **nama modul bisnis** yang disebut Work Owner. Work
Owner menyebutnya "Master Status Progres 1", sehingga paketnya `masterstatusprogres` — sejajar
dengan `masterrekening` dan `masterstatus`.

Isinya tetap Inggris, dan tipe domainnya memakai nama **tipe**, bukan nama modul: `ProgressStatus`,
bukan `MasterStatusProgres`. Itu sebabnya nama kueri `.sql` berawalan `progress_status_`, mengikuti
`claim_status_` pada modul Master Status Klaim — kuerinya membaca satu tabel, bukan satu modul.

### 15.4 `Isian` menjadi `Input`, bukan `Values`

[`peta-penamaan.md`](peta-penamaan.md) memetakan `Isian` → `Values` untuk konteks **antarmuka**
(objek nilai form di React). Di lapisan domain Go, `Isian` adalah tipe masukan yang belum
diperiksa — `Values` di sana akan terbaca seperti kumpulan nilai apa saja. Dipakai `Input`, dan
petanya ditambah satu baris supaya perbedaannya tercatat, bukan terlihat sebagai ketidakkonsistenan.

### 15.5 Kode mati dihapus, bukan diperbaiki

`src/app/Kerangka.tsx`, `Kerangka.test.tsx`, `NavigasiUtama.tsx`, dan `menu.ts` dihapus.

Dasarnya bukan penilaian gaya: `App.tsx` **pada cabang yang melahirkannya** memakai
`KerangkaHalaman`, bukan `Kerangka`. Keempat berkas itu percobaan kerangka yang ditinggalkan di
cabang yang sama, dan perannya sudah diambil `PageShell.tsx` — bilah atas, menu, identitas
pengguna, dan tombol keluar ada di sana.

Memperbaikinya berarti memelihara dua kerangka yang bersaing, dan `D-09` menyebut persis itu sebagai
mode kegagalan yang harus dicegah: satu hal dikerjakan dengan dua cara berbeda.

### 15.6 Entri menu ditambahkan, dan itu keputusan yang perlu disebut

`PageShell` memuat daftar menu **tetap** — `D-59` menetapkan satuan izin adalah menu, tetapi tabel
peran dan izinnya (`TKT-F3-004`) belum dapat diisi karena penugasan operator ke peran tidak ada di
basis data. Menambah satu baris di sana berarti **setiap pengguna yang dapat masuk melihat menu
Master Status Progres 1**, sama seperti kedua menu yang sudah ada.

Itu keadaan yang sama dengan seluruh aplikasi hari ini, bukan pelonggaran baru. Kendalinya tetap di
server: rutenya berada di balik middleware sesi DAN middleware portal aktif.

### 15.7 Yang belum dikerjakan

| Hal | Alasan |
|---|---|
| Tabrakan nama tab/tombol di Master Rekening (3 uji merah) | menunggu keputusan Work Owner — §14.4. Perbaikan yang **tidak** menyentuh teks layar tersedia: beri `aria-label` pembeda pada tab-nya |
| Uji untuk `PageShell` | `Kerangka.test.tsx` dihapus bersama komponennya; uji penggantinya belum ditulis |
| Master Status Progres **tingkat 2** belum punya layar | backend-nya lengkap (`Repo2`, `Service2`, `Mount2`), rutenya belum dipasang di `main.go` dan layarnya belum ada |
| `AccountStatus.{menunggu,disetujui,ditolak}` masih Indonesia | §14.2 — ketiganya juga teks layar |

---

## 16. Menu aplikasi dibaca dari basis data (2026-09-19, sesi kedelapan)

### 16.1 Keputusan Work Owner pada sesi ini

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Menu tanpa otorisasi dan menu yang belum ada modulnya | **Tidak berizin disembunyikan; belum ada modul tampil nonaktif** bertanda "belum tersedia" |
| 2 | Nilai mana yang dicocokkan ke `M_OTORISASI_PNC` | **Dari login yang diketik**, cari GROUP_ID-nya di `M_LOGIN_GROUP_PNC`; lalu cari izin untuk group-group itu DAN untuk login itu sendiri |

### 16.2 Temuan yang menyentuh keputusan lama: `D-58`, `D-59`, dan `TKT-F3-004`

`D-58` menetapkan peran bisnis berjumlah **22, satu-untuk-satu dengan access group Pega**, dan
mencatat penghalangnya:

> **Penugasan operator ke peran tidak ada di database.** `POOLDATA.T_ACCESS_GROUP_PNC` hanya
> memetakan `OPERATOR_ID` ke `OLD_OPERATOR_ID`. Tanpa artefak ini, `F-3` dapat membangun tabelnya
> tetapi **tidak dapat mengisinya**.

Artefak itu **kini ada**, tetapi bukan dalam bentuk yang diperkirakan:

| Yang diperkirakan `D-58`/`D-59` | Yang benar-benar diterima |
|---|---|
| operator ke **22 peran** lalu ke 51 item menu | login ke **group** lalu ke butir menu |
| peta peran-ke-menu hidup di **34 When rule** | izin hidup sebagai **baris tabel** `M_OTORISASI_PNC` |
| 5 When rule hilang dari export | tidak relevan — modelnya tidak memakai When rule |

**Modelnya berbeda, bukan hanya sumbernya.** Tidak ada konsep "peran" di sini: subjek otorisasi
adalah **login** atau **group**, dan keduanya tinggal di kolom yang sama (`LOGIN_ID_GROUP`).

Yang TIDAK berubah: `D-59` tetap berlaku — **penyembunyian menu bukan kendali akses.** Yang
menggerbang tetap pemeriksaan di server pada setiap endpoint modulnya masing-masing. Modul ini
tidak menambah maupun mengurangi kewenangan siapa pun; ia hanya berhenti menawarkan pintu yang
pasti tertutup.

**Pertanyaan terbuka untuk Work Owner:** apakah `M_OTORISASI_PNC` **menggantikan** rencana 22 peran
pada `D-58`, atau keduanya akan hidup berdampingan? Jawabannya menentukan apakah `TKT-F3-004` masih
perlu dikerjakan dalam bentuknya yang sekarang. Saya tidak memutuskannya sendiri: itu mencabut
sebuah keputusan Decision Log, dan pencabutan ditulis sebagai keputusan baru, bukan dikerjakan
diam-diam.

### 16.3 Tidak ada baseline Pega — gerbang 1 tidak berlaku

Ketiga tabelnya **tidak dipakai satu pun rule di export**, dicari ke seluruh 2.634 berkas XML.
DDL-nya pun datang sebagai skrip pembuatan (`Database/CREATE_MENU.sql`), bukan sebagai bagian dari
skema lama.

Artinya modul ini tidak dapat diuji kesetaraannya dengan Pega — polanya sama dengan `F-3` dan `S-5`
pada `D-56`, dan penggantinya sama pula: **uji fungsional terhadap kontrak**. Kontraknya di sini
sudah ada seluruhnya (DDL, isi contoh, dan aturan dari Work Owner), sehingga modul ini **tidak
terhalang** seperti keduanya.

### 16.4 Aturan tampil dibaca dari data, bukan dipilih

| Aturan | Bukti yang memaksanya |
|---|---|
| Kelompok tampil bila ada **anaknya** yang tampil | group `IT` diberi izin atas MENU_ID 11 sampai 81 dan **tidak satu pun** atas 1 sampai 4. Menuntut baris izin untuk kelompoknya akan menghapus seluruh menunya |
| Kelompok yang punya izin tetapi anaknya kosong **disembunyikan** | login `JONNY` diberi izin atas MENU_ID 4 (REPORT). Judul tanpa isi hanya menambah barang di layar |
| Izin group dan izin login **digabung** | keduanya memberi butir yang berbeda: IT memberi MASTER, INBOX, VIEW; JONNY memberi REPORT |
| Daun tanpa `MENU_PROGRAM` **tetap tampil** | MENU_ID 83 "Report Adjuster" ada di master tanpa tujuan. Menyembunyikannya mengubur kekosongan data yang justru perlu dilihat |

### 16.5 Kunci pencocokan: login yang DIKETIK, dan kenapa itu sudah tersedia

Work Owner menetapkan kuncinya **login yang diketik** — bukan NIK, dan bukan nilai yang dikembalikan
HCQ. Sempat tampak menuntut perubahan pada modul Login, yang `D-80` sudah sentuh sekali.

Ternyata tidak. `Profile.Login` didokumentasikan sebagai *"yang diketik pengguna di layar masuk"*,
dan implementasinya `firstNonEmpty(orang.Login, response.Login, k.Username)`. Contoh respons HCQ di
`hcq_test.go` menunjukkan **HCQ memantulkan kembali apa yang dikirim**: permintaannya membawa
`Login: k.Username`, dan responsnya mengembalikan `Person.Login` dengan nilai yang sama persis.
Jadi `User.Login` yang sudah tersimpan memang login yang diketik, untuk kedua populasi pengguna.

**Konsekuensi yang harus disebut terang.** Isi contoh `m_login_group_pnc.csv` dan
`m_otorisasi_pnc.csv` hanya memuat login **non-karyawan** `JONNY`. Karyawan yang masuk lewat HCC/HCQ
akan melihat **menu kosong** sampai barisnya ditambahkan dengan login HCQ-nya — misalnya alamat
surel, bila itu yang mereka ketik. Itu keadaan data, bukan cacat kode, dan layarnya menyebutkannya
apa adanya: *"Belum ada menu yang diberikan untuk pengguna ini."*

### 16.6 Dibaca dari basis data portal UTAMA, bukan per entitas

Keempat tabelnya **tidak punya kolom entitas**, dan letaknya sekerabat dengan `M_LOGIN_PNC` serta
`M_PORTAL_PNC` yang sudah dibaca dari portal utama. Peta menu dan kewenangan pemakainya adalah data
lingkup **identitas**, bukan data bisnis milik satu badan hukum.

Akibat yang disengaja: **menu seseorang sama di keempat portal.** Berpindah portal mengubah data
yang dibaca layar, bukan daftar layar yang boleh ia buka. Rutenya karena itu **tidak** dipasangi
middleware portal — menuntut portal di sini akan membuat menunya gagal justru saat pengguna belum
memilih entitas.

### 16.7 Peta rute di frontend, bukan di backend

Pembagian tugasnya tegas:

```
backend   butir menu mana yang boleh DILIHAT pemanggil   (M_OTORISASI_PNC)
frontend  butir menu mana yang sudah punya LAYAR          (app/menu/registry.ts)
```

Yang dipetakan adalah **rute antarmuka**, dan backend tidak menyimpannya. Menaruh peta itu di server
berarti ia harus tahu bentuk URL React, dan setiap perubahan rute menjadi perubahan di dua tempat.

**Menambah modul sama dengan menambah satu baris** di `registry.ts`. Hari ini isinya tiga:
`StatusClaimInbox`, `MasterRekening`, `StatusProgress` — dari 75 butir yang punya program.

### 16.8 Beranda tidak diambil dari tabel menu

Ia bukan pengganti harness Pega mana pun, melainkan layar milik aplikasi baru ini. Menambahkannya ke
`M_MENU_APLIKASI_PNC` berarti mengarang baris master. Tautannya karena itu tetap di `Sidebar`, di
atas kelompok-kelompok yang datang dari basis data.

### 16.9 Isi contoh adapter memori disalin, bukan disusun

Berbeda dari adapter memori modul lain — yang isinya **susunan sendiri** karena tabel aslinya tidak
ada di export — ketiga daftar di sini **disalin apa adanya** dari CSV yang diterima. Menu yang
terlihat saat pengembangan karena itu sama persis dengan menu produksi, termasuk keanehannya.

Satu tambahan yang TIDAK disalin, dan dipisahkan supaya jelas: `NewDevRepo()` memberi login provider
tiruan (`adminpnc` dan kawan-kawan) keanggotaan group `IT`. Tanpa itu, masuk saat pengembangan
menghasilkan menu kosong — bukan karena ada yang rusak, melainkan karena login itu memang tidak ada
di `m_login_group_pnc.csv`. Tambahan ini hanya hidup di adapter memori; jalur Oracle membaca tabel
yang sebenarnya.

### 16.10 Daftar parameter IN disusun di Go — dan itu bukan perangkaian SQL

Banyaknya subjek berbeda tiap pengguna, sehingga daftar `IN` tidak dapat ditulis tetap di berkas
`.sql`. Penanda `SUBJECTS` di dalam kueri diganti daftar `:2, :3, …` oleh `expandSubjects`.

Yang disisipkan hanyalah **penanda parameter**, tidak pernah nilainya — seluruh nilai tetap dikirim
terpisah. Celah `{ASIS:...}` warisan (`03-CURRENT-ARCHITECTURE.md` §4.5) tetap tertutup, dan
`TestExpandSubjectsInsertsPlaceholdersNeverValues` yang menjaga pernyataan itu tetap benar bila
fungsinya kelak disunting.

### 16.11 Yang belum dikerjakan

| Hal | Alasan |
|---|---|
| Layar pengelolaan menu dan otorisasi | Modul ini hanya MEMBACA. Menambah atau mengubah baris `M_MENU_APLIKASI_PNC` dan `M_OTORISASI_PNC` belum punya layar — dan belum diminta |
| Baris otorisasi untuk pengguna karyawan | Data, bukan kode — lihat §16.5 |
| Apakah `M_OTORISASI_PNC` menggantikan 22 peran `D-58` | menunggu keputusan Work Owner — §16.2 |
| Tabrakan nama tab/tombol di Master Rekening (3 uji merah) | tetap menunggu keputusan Work Owner — §14.4 |

## 17. Modul Inbox Laporan Klaim (2026-09-19, sesi kesembilan)

## 17. Modul Inbox Auto Claim (2026-09-19, sesi kesembilan)

Modul **INBOX** pertama yang dibangun. Sebelumnya seluruh modul bisnis adalah layar
master; ini yang pertama menampilkan **pekerjaan yang menunggu diproses** — dan `D-79`
menetapkan itulah yang membedakan Inbox dari layar daftar biasa.

### 17.1 Keputusan yang diambil Work Owner pada sesi ini

| # | Pertanyaan | Keputusan |
|---|---|---|
| 1 | Enam kueri penggerak layar ini tidak ada di export. Rekonstruksi, atau tunggu Tim Pega? | **Rekonstruksi + tetap minta ke Tim Pega** |
| 2 | Tujuh tombol; empat mesinnya belum ada. Sejauh mana lingkupnya? | **Baca + Export + Upload.** Sisanya tampil nonaktif beserta alasannya |
| 3 | Paginasi server-side seperti Pega, atau client-side seperti layar master? | **Server-side, 15 baris per halaman** |

### 17.2 Kueri yang direkonstruksi, dan apa artinya bagi gerbang 1

Keenam kueri di bawah **tidak ada di export** dan disusun ulang dari tabel serta dari
activity pemanggilnya:

| Kueri Pega yang hilang | Penggantinya |
|---|---|
| `BrowseClaimSPKAutoClaim` | `auto_claim_batch_list` |
| `BrowseClaimSPK1_AutoClaim` | menyatu ke kueri yang sama |
| `BrowseClaimSPK_COUNT_AutoClaim` | `auto_claim_batch_count` |
| `BrowseAutoClaim_COUNT` | `auto_claim_line_count` |
| `BrowseClaimSPK_detail_AutoClaim` | `auto_claim_line_list` |
| `BrowseReportClaimSPK_AutoClaim` | `auto_claim_line_list_succeeded` / `_failed` |

**Konsekuensi yang harus dipegang:** modul ini **tidak dapat dinyatakan lulus gerbang 1**
sampai keenam kueri aslinya tiba, karena tidak ada yang dapat dibandingkan (`D-42`). Yang
dapat dinyatakan sekarang hanyalah bahwa hasilnya konsisten dengan tabel dan dengan
ambang yang tertulis harfiah di activity.

**Yang TIDAK direkonstruksi, dan itu penting:** judul kolom kedua berkas CSV terbaca utuh
di `REPORT_AUTO_CLAIM_ACT`, jadi keduanya **salinan**, bukan dugaan. Termasuk kenyataan
bahwa berkas GAGAL punya **satu kolom lebih sedikit** ("No Ref Bank" tidak ada di sana) —
perbedaan yang dipertahankan apa adanya.

### 17.3 Dua kolom berkas ekspor yang isinya masih DUGAAN

Tujuh dari sembilan kolom dapat ditelusuri ke kolom tabel tanpa ragu. Dua tidak:

| Judul kolom | Diisi dari | Keyakinan |
|---|---|---|
| `No Ref Bank` | `KEYWORD` | **DUGAAN** |
| `No Objek` | `PRODKE` | **DUGAAN** |

Dugaannya diturunkan dari jalur Asuransi Kredit yang berbagi activity yang sama. Di sana
kedua kolom itu diisi properti `WARRANTYNO` dan `CLIENTID`, dan `InsertTempAutoClaim`
memperlihatkan `WARRANTYNO` disimpan ke `PRODKE` serta `CLIENTID` ke `NOASURANSI` pada
tabel kredit.

Tetapi **`TMP_BATCH_AUTO_CLAIM` tidak punya kolom `NOASURANSI`** — sudah diperiksa ke
seluruh export — sehingga pemetaan kredit tidak dapat disalin utuh.

Keduanya ditandai di `export.go` supaya orang yang menjalankan gerbang 1 tahu kolom mana
yang perlu dibandingkan lebih dulu, dan tidak menghabiskan waktu pada tujuh yang sudah
pasti.

### 17.4 Satu penolakan yang TIDAK ada di sistem lama

**Baris unggahan yang kode perusahaannya tidak ada di `M_AUTO_CLAIM_PNC` ditolak.**

Sistem lama tidak memeriksanya saat unggah. Tetapi baris seperti itu **tidak akan pernah
berhasil**: `GetReceiverClaimAsuransiKredit` mencari penerima klaim di master yang sama,
dan tanpa baris master pemrosesan tidak menemukan penerima.

Jadi pilihannya bukan "menolak" versus "menerima", melainkan:

- menolak saat unggah dengan pesan yang dapat diperbaiki, atau
- menerima lalu gagal diam-diam berhari-hari kemudian.

Yang dipilih yang pertama. Ia **penyimpangan yang disengaja** dan harus disebut saat uji
kesetaraan supaya tidak terbaca sebagai cacat.

**Yang tidak ikut diperketat, dan itu juga disengaja:** `col_id` kosong TETAP diterima,
karena `CreateCasePNC_AutoClaim` menanganinya sebagai kegagalan BARIS ("Penyebab kerugian
tidak ditemukan"), bukan penolakan berkas. Menolaknya akan mengubah perilaku yang sudah
ada — pelanggaran `P-5`.

### 17.5 Bentuk tanggal diperiksa, walau nilainya disimpan sebagai teks

`TMP_BATCH_AUTO_CLAIM` menyimpan tanggal sebagai **teks** `dd/mm/yyyy`. Itu terbaca dari
`CreateCasePNC_AutoClaim` yang menyusun ulang timestamp-nya dengan pemotongan karakter
berposisi tetap:

```
Local.dol = @substring(.DateOfLoss,6,10)+@substring(.DateOfLoss,3,5)+@substring(.DateOfLoss,0,2)+"T000000.000 GMT"
```

Dua akibat yang mengikat rancangan:

1. **Nilainya tidak boleh ditafsirkan lalu ditulis ulang.** Membacanya sebagai tanggal
   dan menyimpannya kembali akan mengubah isinya, dan pada tabel yang masih ditulis Pega
   itu melanggar `P-5`. Ia juga mengaktifkan kembali seluruh kelas cacat zona waktu yang
   `F-5` justru tutup.
2. **Bentuknya tetap diperiksa saat unggah.** Teks berbentuk lain tidak menghasilkan
   galat — ia menghasilkan TANGGAL LAIN, diam-diam. `"2026-09-19"` akan terbaca sebagai
   tanggal `09-20` tahun `26-0`, dan tidak ada apa pun yang memberitahukannya.

Pemeriksaan bentuk itu **bukan aturan bisnis baru**; ia menutup satu jalur kegagalan
senyap pada aturan yang sudah ada. Kalender penuh (29 Februari) sengaja TIDAK diperiksa —
itu menuntut penafsiran zona waktu, dan modul ini tidak boleh menafsirkan tanggal sama
sekali.

### 17.6 Nilai uang tetap teks dari ujung ke ujung

`NILAIKLAIM` dibaca, disimpan, diekspor, dan dikirim ke layar sebagai **teks**, bukan
angka. `I-12` menuntut nilai uang disimpan presisi penuh dan dibulatkan hanya saat
ditampilkan; modul ini tidak menghitung apa pun atas nilai itu — ia hanya memindahkannya —
sehingga mengubahnya menjadi `float64` hanya menambah kesempatan kehilangan presisi.

Pemisah ribuan ditambahkan **di layar saja** (`formatMoney` pada `BatchDetail.tsx`), dan
bagian desimalnya dibawa apa adanya, bukan dipaksa dua angka.

### 17.7 LEFT JOIN, bukan INNER — dan uji yang menjaganya

Batch yang kode perusahaannya tidak ada di master **tetap tampil**, dengan nama kosong dan
tanda "Tidak terdaftar di Master Auto Claim".

Dengan INNER JOIN barisnya hilang dari layar tanpa satu pun tanda — padahal baris seperti
itu justru yang **tidak akan pernah berhasil diproses**. Menyembunyikannya adalah
kegagalan senyap yang paling mahal.

Tiga hal menjaganya tidak berubah kelak: baris contoh `ZZZZ` di penyimpanan memori, uji
`TestMasterJoinStaysLeft` pada `query_test.go`, dan uji layar yang mencari teks
penandanya.

### 17.8 Nomor batch diterbitkan server, dan dihitung di Go

Sistem lama tidak menunjukkan dari mana nomor batch berasal — flow action unggahnya
hilang. Yang dipilih: **diturunkan dari isi tabel** per perusahaan, di dalam transaksi
yang sama dengan penyisipannya.

Nomornya dihitung **di Go**, bukan dengan `MAX` di SQL, dengan alasan yang sama seperti
`masterstatusprogres.nextID`: DDL tabelnya tidak ada di export (`R-08`), sehingga tipe
kolom `BATCH` tidak dapat dipastikan. Bila ia `VARCHAR2`, `MAX(BATCH)` adalah maksimum
**leksikografis** — begitu tabel memuat `"10"`, maksimumnya tetap `"9"`, dan nomor
berikutnya kembali `10`.

Satu berkas dapat memuat beberapa perusahaan, dan **setiap perusahaan mendapat nomornya
sendiri**. Itu mengikuti kunci pengelompokan `GroupingAutoClaim2` yang
`(inisialid, batch)`, bukan `batch` saja.

### 17.9 Unggahan bersifat semua-atau-tidak sama sekali

Seluruh berkas diperiksa lebih dulu; bila ada satu baris yang tidak lolos, **tidak satu
baris pun disimpan**, dan seluruh pelanggaran dilaporkan sekaligus beserta nomor barisnya.

Sistem lama menjalankan `RDB-Save` per baris dan tidak menjamin ini. Perubahannya disengaja
dan sejalan dengan `D-68`, yang sudah memindahkan kepemilikan transaksi ke Go justru untuk
kelas masalah yang sama pada `B-4` dan `B-9`.

Alasannya praktis: unggahan yang tersimpan setengah adalah keadaan yang **tidak dapat
diperbaiki pengguna** — ia tidak punya cara mengetahui baris mana yang sudah masuk.

### 17.10 Empat kolom yang sengaja dibiarkan NULL saat menyisipkan

`IDPEGA`, `PROGRESS`, `NOAKSEPTASI`, dan `TMP_MESSAGE` **tidak ikut diisi**. Keempatnya
adalah syarat yang membuat batch baru terambil pemrosesan:

```
GroupingAutoClaim2 : idpega is null AND (progress is null OR progress='0')
GroupingAutoClaim  : noakseptasi IS NULL AND progress='1'
```

Mengisi salah satunya dengan teks kosong alih-alih membiarkannya NULL akan membuat batch
yang baru diunggah **tidak pernah diproses, tanpa satu pun galat**. Uji
`TestInsertLeavesProcessingMarkersUntouched` menjaganya.

### 17.11 `'Sukses Klaim'` boleh berada di dalam kode

`D-15` melarang nilai bisnis di-hardcode. Nilai ini dikecualikan dengan sadar, karena ia
**bukan kebijakan melainkan isi protokol tabel warisan**: yang menuliskannya adalah
`POOLDATA.INSERT_AUTOCLAIM`, dan empat kueri sistem lama menuliskannya apa adanya.

Menjadikannya konfigurasi justru berbahaya — mengubahnya tidak akan mengubah apa yang
ditulis procedure, sehingga layar dan data berselisih diam-diam.

Ia tinggal di **satu konstanta** supaya keempat tempat yang memakainya tidak dapat berbeda
satu sama lain — persis cacat "satu ambang, tiga operator" pada `D-49` #2.

### 17.12 Penyaring memakai KODE, bukan NAMA

Penyaring lama merangkai nama perusahaan langsung ke teks SQL lalu menyisipkannya lewat
`{ASIS:...}`:

```
TemporaryInboxKasirAutoClaim.CaseID = "and B.NAMA_PENERIMA='"+TempSimpan1.CauseOfLoss+"'"
```

Itu celah SQL injection yang `08-TECHNICAL-STRATEGY.md` §4.3 tutup tanpa perkecualian.
Menyaring pada **kode** sekaligus memperbaiki cacat kedua: nama perusahaan tidak dijamin
unik, sehingga dua perusahaan bernama sama akan tercampur.

Yang dilihat pengguna tetap namanya; kodenya hanya nilai di balik pilihan. Uji
`TestCompanyFilterUsesCodeNotName` dan satu uji layar menjaganya tidak kembali ke nama.

### 17.13 Tiga tombol ditampilkan nonaktif, bukan disembunyikan

| Tombol | Menunggu |
|---|---|
| Proses Klaim | `B-2` Input Register · `B-3` Objek & Coverage · `B-5` Estimasi · `B-10` Akseptasi |
| Generate DLA | `B-9` PLA/Pre-DLA/DLA |
| Cek Premi | `S-4` Integrasi Sistem Luar |

Perlakuannya sama dengan butir menu yang belum punya layar (keputusan Work Owner
2026-09-18). Menyembunyikannya akan membuat layar ini **tampak selesai** padahal separuh
alurnya belum ada — kesalahpahaman yang paling mahal di antara semua pilihan.

`Proses Klaim` khususnya tidak boleh dikerjakan setengah: ia memanggil
`CreateCasePNC_AutoClaim` yang membuat **case klaim utuh** — objek, coverage, spreading,
adjustment, akseptasi, sampai penutupan case. Mengarang sebagiannya berarti menulis nomor
klaim palsu ke tabel produksi.

### 17.14 Paginasi ditambahkan ke DataTable secara aditif

`components/DataTable.tsx` mendapat dua prop **opsional**: `pagination` dan `hideSearch`.
Tanpa keduanya, perilakunya sama persis seperti sebelumnya — ketiga layar master tidak
berubah sedikit pun.

`hideSearch` ada karena kotak pencarian di peramban **menyesatkan** pada tabel berpaginasi
server: ia hanya menyaring halaman yang sedang tampil, sementara pengguna mengira ia
mencari ke seluruh data. Baris yang dicarinya ada di halaman lain dan tidak akan pernah
muncul.

### 17.15 Utang teknis yang disadari

| # | Utang | Kenapa diterima sekarang |
|---|---|---|
| 1 | **Gerbang 1 tidak dapat dijalankan** sampai keenam kueri Pega tiba | Menunggu berarti tidak ada layar sama sekali; keputusan Work Owner butir 1 |
| 2 | **Dua kolom CSV masih dugaan** (`No Ref Bank`, `No Objek`) | Ditandai di kode; yang dapat memastikannya hanya kueri asli |
| 3 | **Bentuk berkas unggahan dikarang** dari nama kolom tabel | Flow action aslinya hilang; judul kolomnya hidup di satu tempat dan dilayani server, jadi menggantinya kelak menyentuh satu berkas |
| 4 | **OFFSET di atas urutan yang tidak unik** dapat melewatkan atau menggandakan baris saat berpindah halaman | Kunci primer tabelnya tidak diketahui (`R-08`). Penutupnya DDL, bukan kueri yang lebih pintar |
| 5 | **Penyisipan dapat ditolak kolom NOT NULL yang belum diketahui** | DDL belum ada. Mode `-periksa` memperingatkannya, dan satu unggahan kecil di staging akan membuktikannya |
| 6 | **Urutan nomor batch berbeda antara penyimpanan memori dan SQL** bila kolomnya ternyata teks | Memori mengurutkan sebagai angka, SQL apa adanya. Fake yang meniru cacat akan membuat uji lulus untuk urutan yang salah |
| 7 | Modul ini memetakan galatnya sendiri, bentuk keempat setelah tiga modul lain | Kontrak galat bersama adalah `TKT-F1-004` yang masih terhalang. Kunci pelanggarannya dibuat **sama** dengan `masterstatusprogres` (`kolom`) supaya tidak menambah bentuk baru |
| 8 | Jalur tanpa `/v1` | Sama seperti modul lain; penyeragamannya bukan keputusan satu modul |

### 17.16 Yang perlu diminta ke pihak lain

| Yang diminta | Kepada | Menutup |
|---|---|---|
| Export ulang enam kueri `BrowseClaimSPK*_AutoClaim` beserta sepupu Kredit dan Travel-nya | **Tim Pega** | gerbang 1 modul ini |
| Harness `Detail_AUTOCLAIM_Harness` | **Tim Pega** | susunan kolom layar rincian |
| Flow action di balik tombol "Upload Data Klaim" | **Tim Pega** | bentuk berkas unggahan yang sebenarnya |
| DDL `TMP_BATCH_AUTO_CLAIM` dan `M_AUTO_CLAIM_PNC` | **DBA** | utang 4, 5, dan 6 |
| Hak baca kedua tabel + hak tulis pada yang pertama | **DBA** | menjalankan modul ini terhadap Oracle |

---

## 18. Inbox Auto Claim disetarakan dengan artefak yang akhirnya tiba (2026-09-19, sesi kesembilan lanjutan)

Bab 17 ditulis di atas **rekonstruksi**: tujuh belas kueri, tiga harness rincian, satu flow
action, dan DDL kedua tabel semuanya hilang dari export, dan modulnya dibangun dari nama kolom
yang terbaca di tempat lain.

Pada 2026-09-19 Work Owner mengirimkan **seluruhnya**. Bab ini mencatat apa yang berubah karena
dugaan bertemu bukti — termasuk **enam hal yang saya bangun keliru**.

### 18.1 Apa yang diterima

| Berkas | Isi |
|---|---|
| `Database/CREATE_TABLE_1.sql` | DDL `TMP_BATCH_AUTO_CLAIM` (19 kolom, PK, 7 index) dan `M_AUTO_CLAIM_PNC` |
| `InboxAutoClaim/BrowseClaimSPK*` (15 berkas) | daftar batch, rincian, dan hitungan untuk ketiga tab |
| `InboxAutoClaim/BrowseReportClaimSPK_*` (3) | kueri di balik kedua tombol ekspor |
| `InboxAutoClaim/BrowseCompanyClaimCredit` | sumber penyaring Nama Perusahaan |
| `InboxAutoClaim/Detail_*_Harness` (3) | harness layar rincian |
| `InboxAutoClaim/PNCUploadClaimCSV-FlowAction.xml` + `InsertKlaimToTable*` (4) | rantai unggahan yang sebenarnya |

### 18.2 Enam hal yang saya bangun keliru

Diurutkan menurut akibatnya, bukan menurut ukurannya.

| # | Yang saya bangun | Yang sebenarnya | Akibat bila dibiarkan |
|---|---|---|---|
| 1 | Penyisipan **tidak mengisi `TGLPROSES`** | Ia bagian **PRIMARY KEY** `(NOPOLIS, TGLPROSES, TGLKEJADIAN)` | **Setiap unggahan ditolak ORA-01400.** Tidak akan pernah ketahuan lewat penyimpanan memori — hanya saat menembak Oracle |
| 2 | Berkas unggahan meminta **`inisialid`, `prodke`, `currency`** | Ketiganya **hasil pencarian polis**, bukan isian | Meminta pengunggah mengisi nilai yang tidak ia ketahui, dan membiarkannya salah tanpa satu pun pemeriksaan |
| 3 | Baris yang gagal validasi **ditolak seluruh berkasnya** | Barisnya **DISISIPKAN** beserta pesannya di `IDPEGA`, `NOAKSEPTASI`, dan `TMP_MESSAGE` | Kehilangan jejak baris bermasalah; perilaku yang sama sekali berbeda dari Pega |
| 4 | Kolom `CURRENCY` ditampilkan **apa adanya** | Ia **id**; layar menampilkan hasil lookup ke `POOLDATA.CURRENCY` | Pengguna melihat "1", bukan "IDR" |
| 5 | Ekspor: `No Objek` dari `PRODKE`, `No Ref Bank` dari `KEYWORD` | `No Objek` dari **`COL_ID`**; `No Ref Bank` lewat **DB Link** | Dua dari sembilan kolom berisi data yang salah, dan berkasnya dibaca perusahaan di luar Sinarmas |
| 6 | Daftar batch diurutkan **menaik** | `ORDER BY BATCH DESC` | Batch yang baru diunggah berada di halaman terakhir |

Butir 1 yang paling patut dicatat sebagai pelajaran. Ia **tidak dapat ditangkap satu pun uji
yang saya tulis**, karena penyimpanan memori tidak punya constraint. Uji yang berjalan di atas
tiruan hanya membuktikan kode sesuai dengan tiruannya.

### 18.3 Tiga dugaan yang ternyata benar

Dicatat karena keseimbangannya penting: yang keliru enam, yang benar juga ada.

| Dugaan | Bukti |
|---|---|
| `TMP_MESSAGE = 'Sukses Klaim'` menandai baris berhasil | keempat kueri hitung memakainya persis |
| Baris belum diproses **bukan** baris gagal — `IS NOT NULL` wajib | `AND TMP_MESSAGE!='Sukses Klaim' and TMP_MESSAGE is not null` |
| Keempat hitungan adalah subquery berkorelasi atas tabel yang sama | benar, termasuk bentuk `COUNT(NOPOLIS)`-nya |

### 18.4 Empat selisih yang SENGAJA dipertahankan terhadap kueri Pega

Keempatnya dinyatakan di muka sebagai selisih yang diketahui, bukan ditemukan saat gerbang 1.

| # | Pega | Sistem baru | Alasan |
|---|---|---|---|
| 1 | **INNER JOIN** ke master (`FROM a, b WHERE a.INISIALID = b.INISIALID`) | **LEFT JOIN** | Batch yang kodenya tidak ada di master HILANG dari layar tanpa satu pun tanda — padahal baris itu justru yang tidak akan pernah berhasil diproses |
| 2 | `{ASIS:...CaseID}` — potongan SQL disisipkan mentah | satu kueri per variasi penyaring | Celah injeksi yang diwarisi pola `{ASIS:}` (538 kemunculan di export) |
| 3 | `ROWNUM` berlapis | `OFFSET … FETCH NEXT` | `D-20`. **Ini perubahan perilaku**: pada pola tertentu ROWNUM dihitung sebelum ORDER BY |
| 4 | `substr(IDPEGA,20,30)` bila memuat `PNC` | dipangkas bila **berawalan prefix Pega** | Nomor `PNCN.YY.xxxx` (`D-71`) memuat "PNC" dan lebih pendek dari 20 karakter — Oracle akan mengembalikan **teks kosong**. Untuk setiap nilai yang benar-benar ada hari ini, kedua aturan memberi hasil identik |

### 18.5 Rantai unggahan yang sebenarnya

`Activity/InsertKlaimToTable_Other-Act.xml` bukan penyisipan biasa; ia **rantai pemeriksaan
polis** 32 langkah:

```
nomor polis (titik dibuang)
      |
      v
GetReceiverClaimAsuransiKredit    t_general.sourceofbusiness
   tidak ketemu --------------->  "Sumber Bisnis Tidak ditemukan"  -> baris DITOLAK
      |
      v
BrowsePolisAso                    json_polis -> prodke + currency
   tidak ketemu --------------->  "No Polis tidak di temukan"      -> baris DISISIPKAN bertanda
      |
      v
tanggal lapor >= tanggal kejadian
   dilanggar -----------------> "Tanggal lapor harus setelah..."   -> baris DISISIPKAN bertanda
      |
      v
DOL dalam periode polis           BELUM DAPAT DIPERIKSA - menuntut snapshot polis (B-1)
premi lunas (CekPremiAutoKlaim)   rule-nya tidak ada di export
      |
      v
disisipkan, menunggu diproses
```

Pembedaan **ditolak** versus **disisipkan bertanda** bukan kerapian: baris bertanda masih ada di
tabel dan terlihat di grid, baris ditolak tidak ada di mana pun. Hanya satu sebab yang menolak —
tanpa kode perusahaan, barisnya tidak punya tempat di grid mana pun.

### 18.6 Keputusan Work Owner: unggahan tetap dibuka

Saya mengangkat satu risiko dan mengusulkan menutup unggahan sampai dua hal jelas:

1. Baris yang saya sisipkan **memenuhi syarat pengambilan `GroupingAutoClaim2`**
   (`idpega is null AND (progress is null OR progress='0')`), sehingga job Pega yang masih hidup
   akan memprosesnya menjadi klaim — termasuk baris yang **belum melewati dua pemeriksaan yang
   tidak dapat saya jalankan**.
2. `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel, dan tabel ini
   masih ditulis Pega.

**Work Owner memutuskan: "Unggahan tetap perlu, jangan ditutup."** Keputusan dihormati, dan
modulnya dibuat sesetia mungkin terhadap artefak yang ada. Kedua pemeriksaan yang tidak dapat
dijalankan **dicatat di kode beserta konstanta pesannya**, supaya penambahannya kelak memakai
teks yang sama persis dan baris lama tetap dikenali.

### 18.7 CURRENCY dikirim NULL saat unggah

Pega mengambilnya dari **snapshot polis** (`TempPNC2.Policy.Currency`, hasil parsing JSON polis)
— itu pekerjaan `B-1`, yang belum ada.

Yang dipilih: **mengirim NULL**, bukan menebak. Menebaknya berarti mengisi kolom id dengan nilai
yang tidak pernah cocok dengan `POOLDATA.CURRENCY`, dan akibatnya tidak terlihat sebagai galat —
kolom mata uang hanya kosong di layar dan di berkas ekspor, persis seperti bila NULL. Bedanya,
nilai yang salah **tidak dapat dibedakan dari nilai yang benar** saat gerbang 1 dijalankan.

### 18.8 Perubahan yang dikerjakan

| Berkas | Perubahan |
|---|---|
| `internal/inboxautoclaim/inboxautoclaim.go` | `Batch.ProcessedDate`; `Line` ditambah `CurrencyCode`, `ProcessedDate`, `ObjectName`, `FlagNoPayout`, `BankRefNo`; `Repo` ditambah `ResolveReceiver`, `FindPolicyProductSeq`; `InsertUpload` menerima `UploadLine` |
| `upload.go` | ditulis ulang — kolom berkas yang sebenarnya, enam konstanta pesan harfiah, `CheckUploadShape` + `CheckDateOrder` menggantikan `CheckUpload` |
| `export.go` | pemetaan kolom dikoreksi; `StripClaimPrefix` ditambahkan |
| `repo/sqlstore/inboxautoclaim.sql` | 20 kueri, tiap kueri menyebut berkas sumbernya dan tiap penyimpangan |
| `repo/sqlstore/inboxautoclaim.go` | `TRIM` dibuang (kolomnya `VARCHAR2`, bukan `CHAR`); ekspor punya kueri sendiri; `TGLPROSES` diisi |
| `repo/memory/*` | `PolicyRow`; `groupBatch` dua fase dengan `TGLPROSES` sebagai kunci |
| `usecase/manage.go` | `resolve()` — rantai pemeriksaan polis per baris |
| `http/dto.go`, `http/routes.go` | tiga keadaan hasil unggahan; kolom baru |
| `frontend/src/api/types.ts` | cerminan ketiganya |
| `frontend/.../AutoClaimInboxPage.tsx` | kolom **Tgl Proses**; panel hasil membedakan lolos / bertanda / ditolak |
| `frontend/.../BatchDetail.tsx` | kolom **Tgl Proses**; kunci baris mengikuti PK |
| `frontend/.../UploadForm.tsx` | keterangan bahwa tiga kolom tidak perlu diisi |

### 18.9 Utang teknis — diperbarui

Utang 2, 3, dan 5 pada §17.15 **tertutup**. Yang tersisa dan yang baru:

| # | Utang | Kenapa diterima sekarang |
|---|---|---|
| 1 | Gerbang 1 tetap belum dijalankan | Kuerinya sudah ada, tetapi **Pega staging yang dapat ditembak dari luar** belum dikonfirmasi (`ADR-0027`) |
| 2 | **`CURRENCY` kosong pada setiap baris yang diunggah sistem baru** | Menunggu `B-1`. Terlihat di layar dan di berkas ekspor |
| 3 | **`No Ref Bank` selalu kosong** di berkas ekspor BERHASIL | DB Link diganti API (`D-25`), API-nya belum ada (`R-03`) |
| 4 | **Dua pemeriksaan tidak dijalankan** — DOL dalam periode polis, premi lunas | Keduanya menuntut artefak yang belum ada; konstanta pesannya sudah disiapkan |
| 5 | `OBJECTNAME` dan `FLAGTIDAKBAYAR` dibawa tanpa diketahui artinya | Baru terbaca dari DDL; tidak muncul di satu pun kueri |
| 6 | Paginasi `OFFSET` menggantikan `ROWNUM` | Selisih perilaku yang dinyatakan di muka (§18.4 butir 3) |
| 7 | Modul ini memetakan galatnya sendiri | `TKT-F1-004` masih terhalang |

### 18.10 Yang perlu diminta ke pihak lain — diperbarui

Lima permintaan pada §17.16 **seluruhnya terpenuhi**. Yang tersisa:

| Yang diminta | Kepada | Menutup |
|---|---|---|
| Hak baca `TMP_BATCH_AUTO_CLAIM`, `M_AUTO_CLAIM_PNC`, **`POOLDATA.CURRENCY`** + hak tulis pada yang pertama | **DBA** | menjalankan modul ini terhadap Oracle |
| **Pega staging yang dapat ditembak dari luar** | **Tim Pega + Infra** | gerbang 1 modul ini, dan seluruh modul lain |
| Rule **`CekPremiAutoKlaim`** | **Tim Pega** | pemeriksaan premi lunas saat unggah |
| Rule **`GetMaxBatchAutoClaim`** | **Tim Pega** | memastikan cara penomoran batch sama persis |
| **API pengganti `gl.t_claim_asuransi_credit@asmd`** | **Tim pemilik GL** | kolom `No Ref Bank` pada berkas ekspor |
| Konfirmasi **`GroupingAutoClaim2` masih hidup di produksi** | **Tim Pega** | risiko §18.6 butir 1 |

---

## 19. Inbox Auto Claim: tiga tab, panel ringkasan, dan dropdown yang dibuang (2026-09-20, sesi kesepuluh)

### 19.1 Apa yang berubah dan atas permintaan siapa

Tiga permintaan Work Owner, berurutan dalam satu sesi:

| # | Permintaan | Hasil |
|---|---|---|
| 1 | *"apakah tampilan bisa dibuat seperti contoh terlampir?"* (tangkapan layar Pega: donut + tabel Nama Perusahaan/Count) | Panel ringkasan §19.4 |
| 2 | *"kenapa query yg dipakai berbeda dengan pega?"* + *"tab lain tidak muncul, hanya muncul data Aneka"* | Tiga tab §19.2 |
| 3 | *"kalau sudah muncul tab dan data sudah bisa ter-filter, tidak perlu dropdown perusahaan lagi (sama seperti pega)"* | Dropdown dibuang §19.5 |

Permintaan 2 berawal dari kueri contoh yang dikirim Work Owner. Kueri itu membaca
`POOLDATA.TMP_BATCH_CLAIM_KREDIT`, bukan `TMP_BATCH_AUTO_CLAIM` — jadi ia **bukan koreksi atas
kueri yang saya tulis**, melainkan kueri **tab yang lain**. Work Owner menegaskannya sendiri:
*"iya sample yg sy berikan tadi untuk Asuransi Kredit"*.

Memperlakukannya sebagai koreksi akan membuat tab ANEKA membaca tabel Asuransi Kredit — salah
tanpa satu pun tanda di layar, karena tabelnya tetap terisi dan angkanya tetap masuk akal.

### 19.2 Tiga tab, SATU cetakan kueri

`InboxAutoClaim/InboxAutoClaim-Harness.xml` memuat tiga `pyCaption`: **INBOX AUTO CLAIM**,
**ANEKA**, **Asuransi Kredit**, **Travel**. Masing-masing membaca tabel sendiri:

| Tab | Tabel | Kolom perusahaan |
|---|---|---|
| ANEKA | `POOLDATA.TMP_BATCH_AUTO_CLAIM` | `INISIALID` |
| Asuransi Kredit | `POOLDATA.TMP_BATCH_CLAIM_KREDIT` | **`AGENID`** |
| Travel | `POOLDATA.TMP_BATCH_AUTO_TRAVEL` | `INISIALID` |

Pega menyelesaikannya dengan **menggandakan setiap rule tiga kali** —
`BrowseClaimSPKAutoClaim`, `BrowseClaimSPKClaimKredit`, `BrowseClaimSPKTravel`, dan seterusnya
untuk COUNT, detail, dan laporannya. Itu persis utang teknis yang
`03-CURRENT-ARCHITECTURE.md §4.6` sebut sebagai **duplikasi masif per lini bisnis**: satu
perubahan aturan harus diterapkan di tiga tempat, dan sering hanya diterapkan di dua.

**Yang dibangun: satu berkas `.sql`, tiga hasil.** Ke-23 kueri memakai `{{TABEL}}` (27×) dan
`{{KOLOM}}` (52×), lalu disubstitusi **sekali saat proses menyala** untuk setiap tab:

```go
var resolved = resolveAllQueries()                  // kunci: "<sumber>::<nama kueri>"
func getQueryFor(source Source, name string) string
```

Tiga hal yang membuat ini aman, dan ketiganya disengaja:

1. **Substitusinya bukan dari masukan pengguna.** `Source` enum tertutup; `ParseSource` menolak
   nilai lain dengan `ErrUnknownSource`. Nama tabel tidak dapat diparameterkan dalam SQL, jadi
   satu-satunya cara aman adalah daftar tertutup — bukan perangkaian teks dari `?sumber=`.
2. **Substitusinya di muka, bukan per permintaan.** `resolveAllQueries` **panik** bila masih ada
   `{{` tersisa. Kegagalannya terjadi saat start, bukan pada permintaan pengguna pertama yang
   kebetulan membuka tab itu.
3. **Rujukan ke MASTER sengaja tidak ikut disubstitusi.** `B.INISIALID`, `M.INISIALID`, dan
   `R.INISIALID` menunjuk `M_AUTO_CLAIM_PNC`, yang kolomnya **selalu** `INISIALID` — juga pada
   tab Asuransi Kredit yang tabel batch-nya memakai `AGENID`. Mengganti semuanya membabi buta
   akan menghasilkan `M.AGENID`, kolom yang tidak ada.

Butir 3 dijaga uji `TestEverySourceResolvesToItsOwnTable`, yang untuk tab kredit menuntut
`A.AGENID` **ada** dan `A.INISIALID` **tidak ada**.

### 19.3 Nama tabel dikirim server bersama tabnya

`GET /api/inbox-auto-claim/tab` mengembalikan `{kode, label, tabel}` per tab, dan rutenya
**tidak** di balik pemeriksaan portal — daftar tab sama untuk setiap entitas dan tidak satu
baris data entitas pun dibacanya. Menuntut portal di sana membuat layar gagal menggambar tabnya
justru sebelum pengguna memilih entitas.

Dua hal ikut dari server, bukan diketik di layar:

- **Label.** Rule-nya bernama `*_AutoClaim` sementara yang dilihat pengguna **ANEKA**.
  Menurunkan label dari nama rule menghasilkan tab yang salah nama.
- **Nama tabel** pada keterangan "Sumber: …" di bawah judul grid. Sebelumnya teks tetap
  `POOLDATA.TMP_BATCH_AUTO_CLAIM` — keterangan yang menjadi **salah pada dua dari tiga tab**,
  dan salah dengan cara yang menyesatkan: petugas yang menelusuri selisih angka akan menanyakan
  tabel yang bukan sumbernya kepada DBA.

### 19.4 Panel ringkasan: donut + tabel

Bentuknya mengikuti tangkapan layar yang dikirim Work Owner. Komponen Pega yang menggambarnya
**tidak ada di export** dan tidak punya rule yang dapat dibaca, sehingga panel ini **kemampuan
baru** — tidak ada yang dapat dibandingkan dengannya pada gerbang 1.

| Keputusan | Isi |
|---|---|
| Satuan hitung | **jumlah BATCH**, sama dengan satu baris grid — sehingga angka panel dan total paginasi grid setelah disaring selalu cocok, dan pengguna dapat memeriksanya sendiri |
| Pustaka grafik | **Recharts 3.10.1** — dependensi baru |
| Sumber angka | `FULL OUTER JOIN` ke `M_AUTO_CLAIM_PNC`, bukan `LEFT JOIN` |
| Yang digambar donut | hanya yang berjumlah **> 0** |
| Yang dimuat tabel | **seluruh** perusahaan master, termasuk yang nol |

`FULL OUTER JOIN` bukan kerapian. Dengan `LEFT JOIN` dari tabel batch, panel hanya memuat
perusahaan yang **pernah** mengirim — dan Work Owner melaporkannya langsung: *"hanya muncul 2
perusahaan seharusnya datanya lebih dari 2"*. Sejak dropdown dibuang (§19.5) akibatnya lebih
berat daripada tampilan: perusahaan yang belum pernah mengirim batch tidak akan punya **cara
dipilih sama sekali**.

Arah join-nya dijaga `TestSummaryJoinSpansBothSides`, yang menuntut `FULL OUTER JOIN` beserta
`COALESCE(R.JUMLAH_BATCH, 0)` — menggantikan `TestSummaryJoinStaysLeft` yang justru mengunci
perilaku yang keliru.

### 19.5 Dropdown perusahaan dibuang

Atas permintaan Work Owner, dan sejalan dengan layar Pega yang memang tidak punya dropdown.
Penyaringnya sekarang **hanya** panel ringkasan: irisan donut, baris tabel, dan nama di legenda;
baris **All** membatalkannya.

| Yang dibuang | Berkas |
|---|---|
| Komponen `CompanyFilter` | `AutoClaimInboxPage.tsx` |
| Hook `useAutoClaimCompanyList` beserta rutenya | `api.ts` |
| Invalidasi `inbox-auto-claim-perusahaan` setelah unggah | `api.ts` |

**Endpoint `GET /api/inbox-auto-claim/perusahaan` sengaja TIDAK dihapus.** Ia satu-satunya
pembacaan Master Auto Claim sebagai daftar, dan menghapus permukaan API yang sudah diuji bukan
bagian dari permintaan ini. Bahwa ia kini tanpa pemanggil dicatat sebagai utang, bukan
disembunyikan.

Konsekuensi yang diterima sadar: pada portal dengan puluhan perusahaan, menemukan satu nama
berarti menelusuri tabel ringkasan, bukan mengetik di daftar pilihan. Itulah sebab tabelnya
wajib memuat seluruh master (§19.4).

### 19.6 Kueri ditahan sampai tabnya diketahui

`source` kosong berarti **daftar tab belum tiba**, bukan "semua tab". Ketiga kueri per-entitas
karena itu memakai `enabled: … && source !== ''`.

Tanpa itu, layar menembak server dua kali untuk satu pemuatan — sekali dengan `sumber=` kosong,
sekali lagi begitu tabnya diketahui — dan di antara keduanya panel ringkasan sempat tergambar
lalu **diganti kerangka pemuatan**. Kedipan itulah yang membuat enam uji komponen gagal dengan
`element could not be found in the document` pada elemen yang baru saja ditemukan: elemennya
memang sudah terlepas dari DOM.

### 19.7 Data contoh memori mengisi KETIGA tab

`NewSampleRepo` kini menyemai `SampleKreditLines()` dan `SampleTravelLines()`, dengan isi yang
**sengaja berbeda** dari tab ANEKA.

Alasannya bukan kelengkapan: dengan isi yang sama, layar yang lupa mengirim `?sumber=` akan
tampak benar di ketiga tab. Dan tab yang kosong di lingkungan pengembangan tidak dapat dibedakan
dari tab yang gagal memuat — keduanya tampak sama di layar.

### 19.8 Utang teknis — diperbarui

Ketujuh utang §18.9 **masih berlaku**. Yang bertambah:

| # | Utang | Kenapa diterima sekarang |
|---|---|---|
| 8 | `GET /inbox-auto-claim/perusahaan` tanpa pemanggil | §19.5 |
| 9 | Panel ringkasan **tanpa baseline Pega** | Komponennya tidak ada di export; gerbang 1 tidak berlaku padanya |
| 10 | Recharts menaikkan berkas JS terpaket di atas 500 kB | Belum dipecah; pemecahan kode belum pernah diputuskan untuk aplikasi ini |
| 11 | Nama tabel Oracle terbaca klien lewat `/tab` | Sebelumnya pun tertulis di dalam berkas JS terpaket; yang berubah hanya tempatnya, dan kini ia satu sumber alih-alih tiga salinan |

---

## 20. Urutan tab dan penyaring yang benar-benar menyaring (2026-09-20, sesi kesepuluh lanjutan)

### 20.1 Yang dilaporkan, dan apa sebenarnya sebabnya

Empat laporan berurutan dari Work Owner, yang ternyata **satu sebab**:

| Laporan | Sebab |
|---|---|
| "tab Asuransi Kredit dipindahkan ke depan, Aneka kedua" | permintaan urutan — bukan cacat |
| "filter untuk ketiga tab seharusnya hasilnya berbeda, belum berfungsi" | ringkasan dihitung **dari master**, sehingga ketiga tab memuat daftar perusahaan yang sama persis |
| "semua tab belum berfungsi filternya" | idem |
| "table detail di bawah tidak berubah saat klik pindah perusahaan" | idem — perusahaan yang diklik berjumlah **0 batch**, jadi gridnya kosong pada keduanya |

Penyaringnya **tidak pernah rusak**. Dibuktikan sebelum satu baris pun diubah: `-periksa`
terhadap Oracle menyaring per tab dan cocok (10 dari 11, 31 dari 100, 12 dari 13), dan permintaan
HTTP sungguhan ke instans terpisah mengembalikan baris yang berbeda untuk tiap perusahaan.

Yang rusak adalah **isi panelnya**: ia penuh baris yang bila diklik memang tidak menghasilkan apa
pun.

### 20.2 Ringkasan dihitung dari tabel batch, bukan dari master

Arah join berbalik dua kali, dan keduanya berasal dari laporan Work Owner:

| Kapan | Arah | Akibat |
|---|---|---|
| awal | dari tabel batch | panel memuat 2 perusahaan; dilaporkan "seharusnya lebih dari 2" |
| §19.4 | `FULL OUTER JOIN` ke master | seluruh ~19 perusahaan tampil, **sama di ketiga tab** |
| **sekarang** | `LEFT JOIN` dari tabel batch | tiap tab memuat perusahaannya sendiri |

Laporan pertama pun sebenarnya bukan tentang master: yang dicari Work Owner data **Asuransi
Kredit**, yang saat itu belum punya tabnya sendiri. Tab ANEKA memang hanya punya 2 perusahaan
berbatch — angkanya benar, tabnya yang salah.

Hasil terverifikasi terhadap Oracle: **Asuransi Kredit 9 perusahaan · ANEKA 2 · Travel 2.**

**`LEFT JOIN`, bukan `INNER` seperti Pega.** Kueri Pega (`WHERE C.AGENID = D.INISIALID`) membuang
batch yang kodenya tidak ada di master, padahal barisnya tetap tampil di grid — angka panel dan
isi grid jadi tidak cocok, dan justru baris itulah yang tidak akan pernah berhasil diproses.

**Akibat yang diterima sadar:** perusahaan terdaftar yang belum pernah mengirim batch tidak dapat
dipilih. Memilihnya pun menghasilkan grid kosong, jadi yang hilang hanya cara menyatakan "rekanan
ini belum mengirim apa pun" — dan itu pertanyaan master data, bukan pertanyaan inbox.

### 20.3 Urutan tab, dan satu jebakan yang nyaris ikut terkirim

`AllSource()` menjadi **kredit → aneka → travel**, dan `DefaultSource` mengikuti yang pertama.
Ia **sengaja berbeda** dari urutan harness Pega, yang menyebut ANEKA lebih dulu.

Mengubah `DefaultSource` menyingkap dua tempat yang diam-diam bergantung padanya:

| Tempat | Yang terjadi |
|---|---|
| `memory.NewRepo` menyemai baris ke `DefaultSource` | seluruh data contoh ANEKA **berpindah ke tab Asuransi Kredit**; tab ANEKA menjadi kosong tanpa satu baris kode modulnya berubah |
| `kueriAneka()` pada uji sqlstore memakai `DefaultSource` | namanya tetap "Aneka" sementara yang diperiksanya tabel Asuransi Kredit |

Keduanya diperbaiki dengan menyebut tabnya **terang-terangan** (`SourceAneka`). Tab bawaan adalah
keputusan **tampilan**; ia tidak boleh menentukan tabel mana yang dibaca atau diuji.

Hal yang sama diterapkan pada uji usecase dan HTTP: 39 rujukan `DefaultSource` diganti
`SourceAneka`, dan URL ujinya kini menyebut `?sumber=aneka`.

### 20.4 Uji yang tidak dapat menangkap cacatnya

Uji penyaring yang ada hanya memeriksa **URL permintaan**, dan stub `fetch`-nya mengembalikan
daftar yang sama untuk permintaan apa pun. Cacat "permintaannya terkirim tetapi tabelnya tidak
berubah" — persis kalimat Work Owner — tidak mungkin ditangkapnya.

Yang ditambahkan:

| Uji | Yang dijaga |
|---|---|
| stub `fetch` kini **benar-benar menyaring** | tanpa itu, seluruh uji penyaring hanya menguji perakitan URL |
| `ISI tabel batch benar-benar berubah` | jumlah **baris yang tampil**, bukan URL — dan berpindah antar dua perusahaan, bukan sekadar menyalakan penyaring |
| `setiap baris ringkasan menghasilkan batch` | tidak ada lagi baris berjumlah nol yang kliknya sia-sia |
| `TestRingkasanTiapTabBerbeda` | ketiga tab menghasilkan daftar perusahaan yang berbeda |
| `TestSummaryCountsFromBatchTableNotMaster` | arah join tidak berbalik lagi tanpa disadari |

Ketiganya dibuktikan **merah lebih dulu**: menghapus `parameter.set('perusahaan', …)` membuat uji
isi tabel gagal, dan menyemai master kembali membuat uji ringkasan gagal.

### 20.5 Utang teknis — diperbarui

Utang §19.8 tetap berlaku, kecuali butir 9 yang berubah bunyinya. Yang bertambah:

| # | Utang | Kenapa diterima sekarang |
|---|---|---|
| Adakah pembaca `T_CLAIM_RECIVEDCLAIM` di luar export? | Work Owner + DBA | Menyalakan modul ini di produksi |
| Bentuk nomor laporan `LPK.YY.xxxx` — disetujui? | Work Owner | Tidak menahan; koreksinya satu konstanta |
| Bolehkah pustaka desimal ditambahkan sebagai dependensi? | Work Owner | Tipe kolom `NILAI_ESTIMASI` |
| ~~Bahasa penamaan modul baru — Indonesia atau Inggris?~~ | Work Owner | **TERTUTUP** — dijawab Inggris (`D-80`); lihat §17.9 |
| Persetujuan menjalankan migrasi `0003` | Work Owner + DBA (`D-63`) | Layar bekerja terhadap Oracle |
| Hak `INSERT`/`UPDATE` akun aplikasi atas tabel dan urutan baru | DBA | idem |
| Daftar pilihan `Kurir` dan `Tipe Klaim` — tidak ada di export | Work Owner | Tidak menahan; keduanya teks bebas hari ini |
| Kapan lampiran, utas komunikasi, dan penugasan menyusul | Work Owner | Paritas penuh dengan layar lama |
| Go dan Node terpasang di mesin pengembangan | Work Owner / Tim Infra | **Seluruh verifikasi otomatis** |

---

## 18. Modul Inbox Outstanding (2026-09-19 … 2026-09-20)

### 18.1 Keputusan Work Owner pada sesi ini

| # | Keputusan | Kutipan |
|---|---|---|
| 1 | Membaca `CPNC_KLAIM` + `CPNC_TUGAS`; **tidak membuat tabel sendiri** | *"tabel baru akan menggantikan datapega.pc_asm_fw_gcnmfw_work, jangan buat tabel sendiri"* |
| 2 | Seluruh aksi layar lama masuk lingkup | *"semuanya sesuai pega"* |
| 3 | Lini bisnis dari **`M_LOGIN_PNC.LINEBUSINESS`**, bukan `MST_USER_TEKNIK` | *"tabel itu hanya berisikan pic teknik … m_login_pnc akan dipakai untuk karyawan juga"* |
| 4 | `LINEBUSINESS` kosong → **tiru Pega**, tampilkan seluruh lini | *"samakan seperti pega dulu saja ya"* |

### 18.2 "Outstanding" didefinisikan STATUS, bukan keberadaan tugas

Definisinya mengikat dan terbaca dari satu baris —
`RDB List/BrowseInboxOutstanding1-SQL.xml:125`:

```sql
AND pystatuswork NOT IN ('Resolved-Completed', 'Resolved-Rejected')
```

Layar ini karena itu **bukan Inbox** menurut `D-79`: isinya bukan pekerjaan pemanggil melainkan
seluruh klaim yang belum tuntas. Barisnya tidak hilang setelah dikerjakan, dan tidak ada tombol
"Ambil". Ia layar **pemantauan**.

### 18.3 INNER JOIN diganti LEFT JOIN — perubahan perilaku yang disengaja

Kueri lama menggabungkan klaim dengan assignment memakai inner join:

```sql
FROM datapega.pc_asm_fw_gcnmfw_work a, datapega.pc_assign_worklist b
WHERE a.pzInsKey = b.pxrefobjectkey
```

Bentuk itu punya dua akibat yang tampaknya tidak disadari penulisnya:

1. **Klaim tanpa assignment terbuka tidak muncul sama sekali** — meski statusnya masih berjalan,
   yaitu masih outstanding menurut definisi kueri itu sendiri.
2. **Klaim dengan dua assignment terbuka muncul dua kali.** Alur Register bercabang ke Compliance,
   PUCL, Analyst Doctor, dan RCL Dokter, sehingga ini bukan kasus langka.

Di sini dipakai LEFT JOIN beserta subquery yang memilih satu tugas, sehingga **satu klaim selalu
satu baris** dan klaim tanpa tugas terbuka tetap terlihat.

Alasannya: yang mendefinisikan outstanding adalah status klaim, bukan keberadaan assignment. Klaim
berjalan yang tidak dipegang siapa pun justru yang paling perlu terlihat — ia pekerjaan yang
terhenti.

> **INI PERUBAHAN PERILAKU** dan akan memunculkan selisih pada uji kesetaraan `S-8`. Dinyatakan di
> muka, bukan ditemukan sebagai kejutan, dan **menunggu penegasan Work Owner**.

### 18.4 `{ASIS:}` diganti cabang di Go, bukan diterjemahkan apa adanya

Batas data sistem lama disisipkan ke kueri sebagai potongan teks WHERE
(`BrowseInboxOutstanding1-SQL.xml:129`), yang dipilih activity menurut `OperatorID.pyPosition`:

| Jabatan | Potongan yang dipasang |
|---|---|
| `NONMBU` | `AND GROUPPANEL_1 in ('003','004','006') AND c.businessgroupid NOT IN (…)` |
| `BONDING` | `AND c.businessgroupid NOT IN (…)` |
| `PA` | `AND GROUPPANEL_1='002'` |
| `TRAVEL` | `AND GROUPPANEL_1='005'` |

Di sini cabangnya ditentukan di Go dan **nilainya lewat parameter binding**. Penanda `/*SCOPE*/`
hanya diganti daftar **penanda** `:9, :10, …` — bukan nilainya. Sebuah uji menjaga pernyataan itu
tetap benar dengan memeriksa nilai lini tidak pernah muncul di teks SQL.

### 18.5 Aturan BONDING tidak dapat diterapkan, dan itu dinyatakan

Potongan BONDING hanya mengecualikan `c.businessgroupid`, kolom yang datang dari
`pooldata.businessgroup` lewat join. **Kolom itu tidak ada di `CPNC_KLAIM`.**

Akibatnya BONDING berperilaku sama dengan tanpa batas, dan pengecualian business group juga hilang
dari NONMBU. Ini penyimpangan yang disadari — membawanya berarti menarik join ke tabel Pega ke
dalam modul baru, dan itu keputusan tersendiri yang belum diambil.

### 18.6 Scope kosong gagal TERTUTUP

Tiga keadaan batas data, dan yang ketiga yang paling penting:

| Keadaan | Perilaku |
|---|---|
| `Unrestricted` | seluruh lini terlihat |
| ada `GroupPanels` | hanya lini itu |
| **keduanya kosong** | **tidak meloloskan apa pun** (`AND 1 = 0`) |

Keadaan ketiga hampir pasti cacat pemrograman. Meloloskan semuanya akan mengubah cacat itu menjadi
kebocoran data antar lini yang **tidak menghasilkan galat apa pun** — hanya baris yang seharusnya
tersembunyi. Diuji khusus di adapter memori maupun sqlstore.

### 18.7 Risiko yang diterima sadar: `LINEBUSINESS` kosong berarti melihat semuanya

Work Owner menetapkan perilaku Pega ditiru apa adanya. Konsekuensinya perlu dinyatakan terang:

**Petugas yang datanya belum dilengkapi admin melihat klaim seluruh lini bisnis, termasuk lini yang
bukan haknya, dan tidak ada galat yang muncul.** Karena kolomnya baru ada setelah migrasi `0004`
dijalankan, keadaan itu berlaku untuk **seluruh pengguna** pada mulanya.

Yang dilakukan sebagai penyeimbang — tanpa mengubah perilaku:

1. Layar **menyatakan** keadaannya: "Penyaringan lini bisnis belum berlaku", beserta saran
   menghubungi administrator.
2. Respons membawa `batas_lini`, sehingga keadaan itu terbaca klien mana pun.
3. Kegagalan **membaca** lini dibedakan dari **tidak punya** lini lewat `Result.LineLookupError`,
   dan handler wajib mencatatnya. Tanpa pembedaan itu, kegagalan basis data tidak dapat dibedakan
   dari pengguna yang datanya memang belum diisi.

### 18.8 Kegagalan membaca lini tidak mematikan layar

Kolom `LINEBUSINESS` belum ada, sehingga pembacaannya gagal dengan `ORA-00904` di setiap lingkungan
hari ini. Menghentikan permintaan berarti layar mati total sampai perubahan skema selesai.

Yang dipilih: galat diperlakukan sama dengan "lini tidak diketahui" → jatuh ke tanpa batas, persis
perilaku Pega. **Konsekuensinya diterima:** galat basis data yang sesungguhnya menghasilkan batas
data yang sama dengan pengguna tanpa lini. Yang membedakannya hanyalah catatan di log — dan karena
itu mencatatnya menjadi kewajiban, bukan kenyamanan.

### 18.9 "Export Excel" ternyata CSV

Tombolnya berbunyi "Export Excel", tetapi `Activity/ExportOutstanding_Act-Act.xml` memanggil
**`pxConvertResultsToCSV`** (tiga kemunculan) — tidak ada konversi XLSX di mana pun.

Dipakai `encoding/csv` dari pustaka standar. Itu **setara dengan sistem lama**, bukan
penyederhanaan, dan tidak menambah satu pun dependensi — sejalan dengan Steering §1 yang menyuruh
mengutamakan pustaka standar.

Unduhan ditulis **sambil dibaca**, 500 baris sekali jalan, sehingga memori tetap datar. Batas
10.000 baris dipilih sadar: sistem lama membatasi 500 lewat `pyMaxRecords` pada 54 dari 56
laporannya, sehingga **tidak ada data historis yang sahih** untuk menentukan angkanya (`ADR-0011`).

### 18.10 Per portal, karena klaim adalah data entitas

`CPNC_KLAIM` ada di basis data **setiap** entitas (`ADR-0030`), sehingga modul ini memakai
`RepoSelector` seperti master status progres — bukan repo tunggal. Seluruh rutenya menuntut header
`X-Portal`; permintaan tanpanya **ditolak**, tidak pernah dilayani portal utama sebagai cadangan
(`R-20`, `TKT-F6-002`). Diuji untuk **kedua** rute, termasuk unduhan.

Sebaliknya `M_LOGIN_PNC` dibaca dari portal **utama**: ia master pengguna, dan satu login berlaku
di keempat entitas (`D-78`). Membacanya per portal akan membuat lini bisnis seseorang berubah
mengikuti portal yang sedang dibukanya.

### 18.11 Empat kolom yang sumbernya belum terbukti

Pemetaan kolom **tidak dapat ditentukan dari section**: `InboxOutstandingClaim_Section` memuat dua
grid dan juga dipakai `DashboardClaim_Section1` serta `GCNMGetManagerCase_Act` dengan sumber data
berbeda — terbukti dari sel yang mengikat `.Currency`, `.CauseOfLoss`, dan `.pyEndTime`, yang tidak
ada di kueri Outstanding.

| Kolom | Dipetakan ke | Dasarnya |
|---|---|---|
| `Report Date` | `TANGGAL_LAPOR` | `CONTEXT.md`: "Report Date — Tanggal Lapor" |
| `Admin PNC` | `DIBUAT_OLEH` | petugas pencatat; `pyOrigUserID` pada kueri lama |
| `Posisi Klaim` | `STATUS_POSISI_PROGRES` | **belum terbukti setara** |
| `Progress Klaim` | — | **tidak ada padanan**; belum ditampilkan |

Dua yang pertama dipakai dengan dasar yang dapat dipertanggungjawabkan; dua yang terakhir menunggu
konfirmasi. Tidak ada yang ditebak diam-diam.

### 18.12 Tidak ada rute tulis, dan ketiadaannya disengaja

Klaim dimiliki modul `registrasi`, dan `P-1` menetapkan satu tabel ditulis satu sistem. Modul ini
**hanya membaca** — seam-nya pun tidak punya method tulis.

Aksi **Transfer** dan **Change New User** karena itu belum ada. Keduanya menyentuh penugasan,
`UserTeknis`, penghitung beban PIC pada `MST_USER_TEKNIK` yang **masih ditulis Pega**, dan
pengiriman surel yang modulnya (`S-3`) belum ada. Lingkupnya menunggu keputusan Work Owner.

### 18.13 Kontrak API modul ini

| Metode | Jalur | Sesi | Portal | Keterangan |
|---|---|---|---|---|
| `GET` | `/api/inbox-outstanding` | wajib | **wajib** | daftar klaim berjalan; saringan `cari`, `tahap`, `cabang`, `batas`, `lewati` — beserta `total` dan `batas_lini` |
| `GET` | `/api/inbox-outstanding/unduh` | wajib | **wajib** | CSV seluruh hasil yang cocok; mengabaikan paginasi, tunduk pada batas data yang sama |

`batas` di atas 100 **ditolak**, bukan dipangkas diam-diam: klien yang meminta seribu baris lalu
menerima seratus tanpa diberi tahu akan menampilkan daftar yang ia kira lengkap.

### 18.14 Pertanyaan terbuka yang ditinggalkan sesi ini

| Yang belum diputuskan | Pemilik | Yang tertahan |
|---|---|---|
| Lingkup **Transfer** dan **Change New User** | Work Owner | dua aksi, dan tombol Select All |
| Bolehkah menulis `COUNTER_QUOTA` pada `MST_USER_TEKNIK` | Work Owner | Transfer; `P-1` melarangnya hari ini |
| LEFT JOIN menggantikan INNER JOIN — disetujui? | Work Owner | selisih terencana pada gerbang 1 |
| Sumber **Posisi Klaim** dan **Progress Klaim** | Work Owner / Tim Pega | dua kolom |
| Daftar nilai `LINEBUSINESS` yang sah | Work Owner | tidak menahan; nilai tak dikenal jatuh ke tanpa batas |
| Persetujuan menjalankan migrasi `0004` | Work Owner + DBA (`D-63`) | batas data per lini benar-benar berlaku |
| Siapa mengisi `LINEBUSINESS` per pengguna | Work Owner | idem; datanya tidak ada di export maupun basis data |

### 18.15 Koreksi sumber kolom (2026-09-21) — menggantikan §18.11

§18.11 menyatakan pemetaan kolom tidak dapat ditentukan karena
`InboxOutstandingClaim_Section` melayani dua grid dan beberapa sumber data. Pernyataan itu
benar **tentang section yang salah**.

Section yang mengikat modul ini adalah **`Section/InboxRegister_Section-Section.xml`** —
yang dimuat `Harness/InboxRegister_Harness-Harness.xml`, harness yang Work Owner tunjuk
sejak awal, dan yang di dalamnya sendiri berjudul "Inbox Outstanding" (`:2150`).

Pengikatnya terbukti: properti sel di section itu **persis alias kueri
`BrowseInboxOutstanding1`** — `.District` untuk "Policy no", `.CountryID` untuk "Insured
name", `.City` untuk "Branch name", `.ReporterName` untuk "Admin name".

**Pemetaan yang berlaku sekarang:**

| Kolom layar | Properti section | Kolom `CPNC_KLAIM` |
|---|---|---|
| Claim no | `.ClaimNo` | `NOMOR` |
| Policy no | `.District` | `POLIS_NOMOR` |
| Insured name | `.CountryID` | `POLIS_TERTANGGUNG` |
| Business Name | `.Country` | `POLIS_JENIS_BISNIS` |
| Branch name | `.City` | `POLIS_KODE_CABANG` |
| Admin name | `.ReporterName` | `DIBUAT_OLEH` |
| Register Date | `.pxCreateDateTime` | `DIBUAT_PADA` |
| Date of loss | `.DateOfLoss` | `TANGGAL_KEJADIAN` |
| Claim status | `.StatusClaim` | `STATUS_PROSES` → label tampil |
| Status ASM | `.LSC_ID` | `STATUS_KLAIM` — **kode, bukan label** |
| ASM PIC | — | `USER_TEKNIS` |
| Total Aging | — | dihitung dari `DIBUAT_PADA` |

Dua kolom sebelumnya diragukan — "Posisi Klaim" dan "Progress Klaim" — **bukan kolom layar
ini sama sekali**. Keduanya milik section Outstanding yang dipakai dashboard.

### 18.16 Tiga keterbatasan yang dicatat, bukan ditutupi

| Hal | Keadaannya |
|---|---|
| **Business source** | `sobname` tidak ada di `CPNC_KLAIM`; kolomnya tidak dibangun |
| **Aging** | tanggal acuannya (`.DateForAging`) tidak disediakan kueri; hanya "Total Aging" yang ada |
| **Status ASM** | yang tampil KODE status, bukan labelnya — label hidup di master status, dan menariknya ke sini berarti menyentuh modul lain |

Ketiadaan dua yang pertama **dijaga uji**, supaya tidak diam-diam diisi nilai yang
dikarang di kemudian hari.

### 18.17 `T-1` gugur: Transfer bukan bagian layar ini

Tombol pada `Section/InboxRegister_Section-Section.xml` hanya **Cari · Export To Excel ·
Input Claim** (`CreateInputKlaim`, `ExportDataDetailKlaim`, `ExportLostAdjuster`). Nol
kemunculan `GCNMTransferDataKlaim` maupun "Change New User".

Akibatnya tiga pertanyaan terbuka §18.14 ikut gugur:

- lingkup Transfer / Change New User — **bukan bagian layar ini**
- menulis penghitung beban ke tabel Pega — **tidak terjadi**
- `P-1` atas `mst_user_teknis.TOTAL_JOB` — **tidak terjadi**

Sekaligus mengoreksi laporan sebelumnya: penghitung yang disentuh Transfer adalah
`mst_user_teknis.TOTAL_JOB`, **bukan** `MST_USER_TEKNIK.COUNTER_QUOTA`. Keduanya tabel
berbeda yang namanya beda satu huruf; `COUNTER_QUOTA` milik routing pembagian beban
(`B-6`), dan tidak pernah disentuh layar ini.

### 18.18 Pertanyaan terbuka setelah koreksi

| Yang belum diputuskan | Pemilik | Yang tertahan |
|---|---|---|
| **Tab-tab** pada section rujukan — ALL Case, Communication, Loss Adjuster, Temporary Close | Work Owner | masing-masing punya kolom tambahan; yang dibangun baru daftar intinya |
| Tombol **Input Claim** | Work Owner | ia membuka alur registrasi, milik modul lain |
| Sumber **Business source** dan **Aging** | Work Owner / DBA | dua kolom |
| Label **Status ASM** | Work Owner | hari ini yang tampil kodenya |
| LEFT JOIN menggantikan INNER JOIN | Work Owner | **ditunda atas arahan Work Owner**; kueri dibiarkan apa adanya |
| Persetujuan migrasi `0004` | Work Owner + DBA (`D-63`) | batas data per lini benar-benar berlaku |

### 18.19 Koreksi kedua: sumber datanya `POOLDATA.T_CLAIMLIST_ADMIN` (2026-09-21)

Menggantikan §18.15 dan seluruh bagian §18 yang menyebut `CPNC_KLAIM`.

**Yang menggantikan `datapega.pc_asm_fw_gcnmfw_work` adalah `POOLDATA.T_CLAIMLIST_ADMIN`**,
ditetapkan Work Owner. `CPNC_KLAIM` yang sempat saya pakai adalah tabel rancangan modul
`registrasi` yang belum pernah dibuat dan modulnya belum dipasang — layar akan selalu kosong
tanpa satu pun galat.

**Empat keterbatasan yang sebelumnya dicatat gugur seluruhnya:**

| Dicatat sebelumnya | Keadaan sebenarnya |
|---|---|
| `Business source` tidak ada | kolom `SOBNAME` |
| `Aging` tidak dapat dihitung | kolom `AGING`, sudah berupa angka |
| aturan BONDING tidak dapat diterapkan | kolom `BUSINESSGROUPID` ada |
| INNER JOIN versus LEFT JOIN | tabel datar — **tidak ada join sama sekali** |

**Batas data kini lengkap.** NONMBU menyaring Group Panel **dan** mengecualikan kelompok
bisnis; BONDING seluruhnya berupa pengecualian, tanpa menyaring lini. Sisi SQL memeriksa
`BUSINESSGROUPID IS NULL` lebih dulu — tanpa itu, klaim yang kelompoknya belum terisi hilang
dari layar oleh aritmetika tiga-nilai, bukan karena dikecualikan.

**`AGING` dibaca apa adanya, tidak dihitung ulang.** Tabel juga punya `DATEFORAGING_1` yang
tampaknya menjadi acuannya, tetapi artinya belum dipastikan; menghitung ulang berarti menebak
dari tanggal mana. Nilai yang belum terisi dibedakan dari nol — pada layar tampil sebagai
tanda hubung, pada CSV sebagai sel kosong.

**Dua pemetaan yang BELUM PASTI, dicatat bukan dinyatakan selesai:**

| Kolom | Dipakai | Keraguannya |
|---|---|---|
| `Status ASM` | `STATUSLOCK_1` | section mengikatnya ke `.LSC_ID`; kueri lain mengisi `.StatusLock` dari `v_sts_claim.lsc_note` |
| `Register Date` | `REGISTERDATE_1` | tabel punya `PXCREATEDATETIME` juga; yang kedua dipakai sebagai cadangan saja |

**Satu penyaring sengaja tidak ditambahkan:** `STS_AKTIF`. Kolomnya ada dan tabel datar
biasanya memakainya untuk menandai baris aktif, tetapi kueri lama tidak menyebutnya —
menambahkannya berarti mengubah perilaku tanpa dasar.

### 18.20 Pertanyaan terbuka setelah koreksi kedua

| Yang belum diputuskan | Pemilik | Yang tertahan |
|---|---|---|
| Arti `DATEFORAGING_1`, dan apakah `AGING` memang dihitung darinya | Work Owner / DBA | tidak menahan; `AGING` dibaca apa adanya |
| Pemetaan `Status ASM` — `STATUSLOCK_1` atau label dari `v_sts_claim` | Work Owner | satu kolom |
| Perlukah menyaring `STS_AKTIF` | Work Owner | baris non-aktif ikut tampil bila ada |
| **Tab-tab** section rujukan — ALL Case, Communication, Loss Adjuster, Temporary Close | Work Owner | masing-masing punya kolom tambahan |
| Tombol **Input Claim** | Work Owner | membuka alur registrasi, milik modul lain |
| Persetujuan migrasi `0004` (`LINEBUSINESS`) | Work Owner + DBA (`D-63`) | batas data per lini benar-benar berlaku |

Tiga pertanyaan §18.18 **gugur**: lingkup Transfer, penulisan penghitung beban, dan
INNER/LEFT JOIN — ketiganya tidak berlaku pada layar dan tabel yang benar.

### 18.21 Nilai `PYSTATUSWORK` diambil dari data, bukan dari tabel yang salah pakai (2026-09-22)

**Keputusan.** Derivasi kolom "Claim status" memakai nilai yang benar-benar tersimpan:

| `PYSTATUSWORK` | Kolom "Claim status" |
|---|---|
| `New` | On Progress |
| `Resolved-Completed` | Close |
| `Resolved-Rejected` | Reject |
| **lainnya** | **ditampilkan apa adanya** |

Pemetaannya sudah tertulis benar di dokumentasi tipe `DisplayStatus` sejak awal — bersumber
`Activity/InboxOutstanding_Act-Act.xml` dan dikuatkan `RDB List/BrowseClaimALL-SQL.xml`.
Yang keliru kodenya, yang masih membandingkan nilai `CPNC_KLAIM.STATUS_PROSES`.

**Baris terakhir adalah perubahan perilaku yang disengaja.** Sebelumnya setiap nilai tak
dikenal jatuh ke `default` dan keluar sebagai "On Progress" — termasuk klaim yang sudah
ditutup. Menampilkannya apa adanya membuat nilai asing terlihat, bukan menyamar.

Ini **bukan** selisih terhadap Pega: Pega pun hanya memetakan tiga nilai itu, dan `CASE WHEN`
tanpa `ELSE` mengembalikan `NULL` untuk sisanya. Perbedaannya hanya pada apa yang tampil saat
hal yang mestinya mustahil terjadi.

### 18.22 "Status ASM" ditampilkan apa adanya, tanpa menyatakan kode atau label

**Keputusan.** Isi `STATUSLOCK_1` dirender sebagai teks biasa. Layar **tidak** menyatakan
apakah ia kode atau label.

**Sebelumnya** ia dirender `font-mono` dengan tooltip "Kode status klaim; labelnya ada di
master status" — pernyataan yang tidak berdasar. Buktinya justru menunjuk arah sebaliknya:

| Bukti | Arahnya |
|---|---|
| Kueri lama mengambil label lewat `v_sts_claim` dari kolom `STATUSCLAIM_1` | kode disimpan terpisah |
| `T_CLAIMLIST_ADMIN` **tidak punya** `STATUSCLAIM_1` | kodenya tidak dibawa ke tabel baru |
| `STATUSLOCK_1` selebar `VARCHAR2(100)` | kode butuh 4 karakter; label butuh ±30 |

**Belum dikonfirmasi** karena baris produksi yang diserahkan tidak menyertakan kolom itu.

**Kenapa tidak menebak saja.** Menyatakan "ini kode" lalu menampilkan kalimat utuh membuat
pengguna menyangka layarnya rusak. Menampilkan apa adanya benar pada kedua kemungkinan.

### 18.23 Data contoh menyerap bentuk baris produksi, bukan isinya

**Keputusan.** Satu baris contoh dibentuk dari baris produksi yang diserahkan Work Owner,
dengan **nomor polis, nama tertanggung, nomor klaim, dan nama sumber bisnis diganti karangan**
(`D-69`), sementara bentuknya dipertahankan utuh.

`D-64` yang mengizinkan data produksi apa adanya berlaku untuk **isi lingkungan staging** —
bukan untuk berkas yang di-commit. Keduanya berdiri sendiri.

**Yang dipertahankan, dan tidak akan terpikir dikarang:** tanggal kejadian sesudah tanggal
pendaftaran · `AGING` 618 hari yang tidak sejalan dengan umur sejak pendaftaran · "Status ASM"
kosong · kelompok bisnis yang lolos dua lini sekaligus · nomor klaim berformat lama ·
`PZINSKEY` berprefix kelas Pega.

### 18.24 Pertanyaan terbuka — diperbarui

| Yang belum diputuskan | Pemilik | Yang tertahan |
|---|---|---|
| Isi `STATUSLOCK_1`: kode atau label | Work Owner / DBA | tampilan satu kolom |
| Arti `DATEFORAGING_1` | Work Owner / DBA | tidak menahan — `AGING` dibaca apa adanya |
| Perlukah menyaring `STS_AKTIF` | Work Owner | baris non-aktif ikut tampil bila ada |
| Tab-tab section rujukan — ALL Case, Communication, Loss Adjuster, Temporary Close | Work Owner | masing-masing punya kolom tambahan |
| Tombol **Input Claim** | Work Owner | membuka alur registrasi, milik modul lain |
| Persetujuan migrasi `0004` (`LINEBUSINESS`) | Work Owner + DBA (`D-63`) | batas data per lini benar-benar berlaku |

**Satu pertanyaan §18.20 gugur sebagian:** pemetaan "Status ASM" bukan lagi pilihan antara
`STATUSLOCK_1` dan kolom lain — tidak ada kolom lain. Yang tersisa hanya: apa isinya.

### 18.25 Tabrakan tabel klaim dengan modul `registrasi` — belum diputuskan (2026-09-22)

Ditemukan saat menjawab pertanyaan Work Owner tentang `CPNC_KLAIM`. **Bukan cacat hari ini**,
tetapi akan menggigit saat `B-2` dipasang — dan diam-diam, tanpa galat.

| Modul | Tabel klaim | Perannya | Terpasang? |
|---|---|---|---|
| `registrasi` (`B-2`) | `CPNC_KLAIM` | **menulis** | **tidak** — nol kemunculan di `main.go`, tidak diimpor paket mana pun |
| `inboxoutstanding` | `POOLDATA.T_CLAIMLIST_ADMIN` | membaca | ya |
| Pega | `POOLDATA.T_CLAIMLIST_ADMIN` | menulis | ya |

**Akibatnya bila `registrasi` dipasang apa adanya:** klaim yang didaftarkan lewat Go masuk
`CPNC_KLAIM`, sedangkan layar ini membaca `T_CLAIMLIST_ADMIN`. Klaim itu **tidak akan pernah
muncul di Inbox Outstanding**, dan tidak ada galat yang menandainya — layarnya hanya tidak
menampilkannya.

Ini persis kelas kegagalan yang membuat modul ini salah pilih tabel (§17.15): bukan gagal,
melainkan kosong.

**Pertanyaannya milik `P-1`** — satu penulis per tabel selama masa paralel. Tiga kemungkinan,
belum satu pun dipilih:

1. `B-2` menulis ke `T_CLAIMLIST_ADMIN` juga — tetapi Pega penulisnya, sehingga melanggar `P-1`
2. Inbox Outstanding kelak membaca dua sumber — melanggar "satu layar satu permintaan"
3. `CPNC_KLAIM` ditinggalkan, dan `B-2` menulis ke tabel yang sama dengan Pega setelah
   kepemilikannya berpindah (`ADR-0004`)

**Pemilik: Work Owner.** Harus dijawab **sebelum** `registrasi` dipasang ke `main.go`, bukan
sesudah — sesudahnya berarti klaim nyata sudah tersebar di dua tabel.

> Tidak ada berkas `registrasi` maupun migrasinya yang disentuh sesi ini. Temuan ini
> dilaporkan, bukan diperbaiki.

### 18.26 Sebelas tab layar rujukan — hanya satu yang dapat bersumber dari `T_CLAIMLIST_ADMIN` (2026-09-22)

Work Owner menyebut sebelas pilihan yang ada di layar Pega dan meminta sumber datanya
diarahkan langsung ke `T_CLAIMLIST_ADMIN`. Pemeriksaan ke export **dan ke isi tabelnya**
menunjukkan sebagian besar tidak dapat.

**Pemetaan tab, dari `Activity/SetClaimPNC-Act.xml`.** Tab dikendalikan satu properti
`TempVisibility.Email` bernilai 0–10 — sebelas nilai, sejalan dengan sebelas butir yang
disebut Work Owner. Labelnya terbaca dari `pyStepsDescription` tiap cabang:

| `Email` | `pyStepsDescription` | Label layar |
|---|---|---|
| 0 | dokumen LENGKAP | Complete documents |
| 1 | dokumen BELUM LENGKAP | Documents not complete |
| 2 | os loss adjuster | Loss Adjuster |
| 3 | count temporary close · ALL DATA | (nilai bawaan) |
| 4 | internal surveyor | Internal Surveyor |
| 5 · 6 · 7 | Komunikasi · "yang tanya belum di jawab" | Not Answered · Replied… |
| 8 | temporary close | Temporary Close |
| 9 | Deadline To Temporary Close | Deadline To Temporary Close |
| 10 | dokumen cabang | — |

**Apakah `T_CLAIMLIST_ADMIN` dapat menjawabnya** — diperiksa atas 1.014 baris:

| Tab | Kolom yang dibutuhkan | Keadaan |
|---|---|---|
| **ALL Case** | — cukup melepas `PYSTATUSWORK NOT IN (…)` | ✅ **dapat** |
| Complete / not complete documents | penanda kelengkapan dokumen | ❌ tidak ada kolomnya |
| Loss Adjuster · Internal Surveyor | `REQUESTSURVEY_1` | ❌ **100% NULL** pada 1.014 baris |
| Temporary Close · Deadline | penanda temporary close | ❌ tidak ada |
| Not Answered · Replied (3 tab) | riwayat komunikasi | ❌ tabel lain |

Tiga sub-section yang dirujuk tab — `InboxPICTeknik`, `_1`, `_2` — **hilang dari export**
(`R-16`), sehingga penyaringnya pun tidak dapat dibaca dari sana.

**Kesimpulan:** mengarahkan sumbernya ke `T_CLAIMLIST_ADMIN` hanya menyelesaikan **ALL
Case**. Sepuluh sisanya menuntut tabel lain, dan dua di antaranya (`Not Replied From
Receiver`, `Replied From Receiver`) **nol kemunculan di seluruh export**.

### 18.27 `STATUSLOCK_1` kosong di SELURUH tabel — kolom "Status ASM" akan selalu hampa

Ditemukan saat memeriksa kolom kandidat, dan ini lebih berdampak daripada persoalan tab:

```
TOTAL_BARIS 1014 · STATUSLOCK_TERISI 0
```

Kolom yang dipakai modul ini untuk "Status ASM" **tidak pernah diisi**. Bukan kosong pada
baris uji Work Owner saja — kosong pada **seluruh 1.014 baris**.

Ini menutup pertanyaan terbuka §18.22 dengan jawaban yang tidak diduga: bukan "kode atau
label", melainkan **tidak keduanya**. Dugaan bahwa tabel datar sudah menyelesaikan pencarian
`v_sts_claim` di muka **gugur** — ia hanya menyediakan kolomnya, tanpa mengisinya.

**Akibatnya:** kolom "Status ASM" akan selalu tampil sebagai tanda hubung. Tampilannya tidak
rusak — keputusan §18.22 menampilkannya apa adanya justru menyelamatkan keadaan ini — tetapi
kolomnya tidak membawa informasi apa pun.

**Tiga kemungkinan, belum diputuskan (pemilik: Work Owner + DBA):**

1. Kolomnya memang belum diisi proses pengisi tabel, dan akan terisi kelak
2. Statusnya harus diambil dari master (`M_STS_CLAIM`, 33 kode) lewat kolom lain
3. Layar tidak memerlukan kolom itu, dan ia dihapus

Sampai dijawab, kolomnya **dipertahankan** — menghapusnya adalah perubahan yang lebih sulit
dibalik daripada membiarkan satu kolom kosong.

Catatan pendukung: `NOTREGISTNOTE_1` dan `REQUESTSURVEY_1` juga **0% terisi**, sedangkan
`ENDDATE` 646, `STATUSPROGRESS2` 826, `REINSURER` 266, `KURIR` 112 dari 1.014.
| 12 | Perusahaan terdaftar yang belum punya batch **tidak dapat dipilih** di panel | §20.2 — memilihnya pun menghasilkan grid kosong |
| 13 | `-periksa` menguji penyaring dengan kode **mentah**, sementara HTTP memangkasnya | ditutup sebagian: pemeriksaan spasi-di-ujung ditambahkan, dan pada Oracle hari ini tidak ada yang berspasi |

---

## 21. Paginasi yang tidak menentukan urutan (2026-09-20)

### 21.1 Cacat yang ditemukan karena Work Owner bertanya

Pertanyaan *"perhatikan juga paging-nya, apakah sudah benar?"* menemukan cacat yang tidak
dilaporkan siapa pun dan tidak akan muncul sebagai galat.

Kedua kueri daftar batch mengurutkan **hanya** `ORDER BY A.BATCH DESC`. `BATCH` berulang antar
perusahaan dan antar tanggal — pada tab Asuransi Kredit, 100 baris grid hanya berisi 62 pasangan
(perusahaan, batch) yang berbeda.

`OFFSET … FETCH NEXT` memotong hasil **menurut urutan**. Bila urutannya tidak menentukan satu
susunan tunggal, basis data boleh menyusun baris yang seri dengan cara berbeda pada tiap
eksekusi — dan halaman 2 lalu dapat mengulang baris halaman 1 sementara baris lain **tidak pernah
muncul sama sekali**.

**Pega tidak menghadapinya** karena gridnya memuat seluruh page list sekaligus; ia tidak pernah
memotong hasil. Ini konsekuensi langsung dari paginasi sisi server, dan `15-NFR` memang sudah
mencatat paginasi sebagai **perubahan perilaku**, bukan pemeliharaan.

### 21.2 Perbaikannya, dan kenapa keempat kolomnya

```sql
ORDER BY A.BATCH DESC, A.{{KOLOM}}, CAST(A.TGLPROSES AS DATE) DESC, A.USERINPUT
```

Keempatnya bersama-sama adalah **kunci `GROUP BY`** kueri itu, sehingga tidak ada dua baris hasil
yang dapat seri — urutannya total. Ini penambahan terhadap kueri Pega, dan wajib disebut saat uji
kesetaraan: **urutan baris dalam satu nomor batch dapat berbeda dari Pega.** Isinya tidak.

`BATCH` bertipe `NUMBER` menurut `Database/CREATE_TABLE_1.sql`, jadi urutannya memang numerik —
tidak ada persoalan `'10' < '9'` yang biasa muncul bila nomor batch bertipe teks.

### 21.3 Yang TIDAK terbukti, dan dikatakan apa adanya

Pemeriksaan terhadap Oracle **tidak** menemukan halaman yang tumpang tindih:

```
tab Asuransi Kredit halaman 1 dan 2 tidak beririsan (15 + 15 dari 100)
```

Oracle kebetulan menyusun baris yang seri secara konsisten untuk rencana eksekusi yang sama. Itu
**bukan jaminan** — rencana dapat berubah karena statistik, paralelisme, atau data baru. Cacatnya
nyata, kemunculannya belum. Diperbaiki karena kebenarannya tidak boleh bergantung pada kebetulan.

### 21.4 Tiga pemeriksaan baru pada `-periksa`

Ketiganya lahir dari laporan yang tidak dapat dijawab oleh jumlah baris saja, dan semuanya
melaporkan **angka**, tidak pernah nama perusahaan:

| Pemeriksaan | Menjawab |
|---|---|
| kode perusahaan berspasi di ujung | "penyaring tidak berfungsi" yang disebabkan kolom `CHAR` — HTTP memangkas spasinya, repo tidak |
| ketiga tab dibandingkan isinya | "tab ANEKA menampilkan data Asuransi Kredit" |
| halaman 1 vs halaman 2 | "apakah paging-nya benar" |

Yang kedua menjawab laporan 2026-09-20 dengan bukti: **Asuransi Kredit vs ANEKA beririsan 0
baris.** Ketiga tab membaca tabel yang berbeda, dan itu kini diperiksa setiap kali `-periksa`
dijalankan — bukan disimpulkan dari membaca kode.

---

## 22. Baris kembar: `CAST(timestamp AS DATE)` tidak memotong apa pun di Oracle (2026-09-20)

### 22.1 Tangkapan layar yang menyelesaikan penelusuran

Work Owner mengirim tangkapan layar tab Asuransi Kredit: **tiga baris DIRECT MO yang sama
persis** — kode, nama, batch, tanggal, jumlah, dan pengunggah seluruhnya identik.

Itu yang memecahkan penelusuran dua putaran sebelumnya. Sampai saat itu bukti menunjukkan
penyaring bekerja benar di repo maupun HTTP, dan laporan "filter tidak berfungsi" tidak dapat
dipertemukan dengan buktinya.

### 22.2 Sebabnya: satu kalimat yang saya tulis sendiri dan salah

Kueri Pega mengelompokkan per hari dengan `to_date(to_char(a.TGLPROSES,'dd/mm/yyyy'),'dd/mm/yyyy')`.
Padanannya saya tulis `CAST(A.TGLPROSES AS DATE)` beserta keterangan *"portabel dan berarti
sama (D-20)"*.

**Keterangan itu salah.** Tipe `DATE` Oracle **membawa jam**, jadi cast dari `TIMESTAMP`
tidak memotong apa pun di sana; di PostgreSQL cast yang sama memotong ke hari. Satu kueri, dua
perilaku — persis yang `D-20` larang.

Akibatnya satu hari kalender terpecah menjadi **satu kelompok per detik yang berbeda**, dan Go
memformat tanggalnya hanya sampai hari — sehingga barisnya tampak kembar persis tanpa apa pun di
layar yang menjelaskannya.

| Tab | Baris grid sebelum | Sesudah | Kembar |
|---|---:|---:|---:|
| Asuransi Kredit | 100 | **62** | 38 |
| ANEKA | 11 | **9** | 2 |
| Travel | 13 | 13 | 0 |

### 22.3 Perbaikannya

`EXTRACT(YEAR/MONTH/DAY FROM A.TGLPROSES)` untuk pengelompokan — ANSI, berarti sama di kedua
basis data, dan tidak bergantung pada apakah tipe `DATE` setempat membawa jam. Tanggal yang
ditampilkan diambil `MIN(A.TGLPROSES)`: seluruh anggota kelompok berada pada hari yang sama, jadi
nilai mana pun memberi tampilan yang sama.

Diterapkan pada **lima** kueri yang harus sepakat satuannya — dua daftar batch, dua penghitung,
dan ringkasan per perusahaan. Bila salah satu tertinggal, "Total Data" dan jumlah baris yang
tampil akan berselisih lagi.

### 22.4 Dua cacat yang ikut tertutup

| Cacat | Bagaimana terkait |
|---|---|
| **Paginasi meleset** | Penghitung TIDAK menjoin master dan tidak terkena pemecahan kelompok, sehingga "Total Data" menyebut 100 sementara baris yang tampil lebih banyak. Halaman terakhir menjadi pendek |
| **Master berbaris ganda** | `M_AUTO_CLAIM_PNC` memuat lebih dari satu baris per `INISIALID`. `MAX(B.NAMA_PENERIMA)` ditambahkan dan namanya dikeluarkan dari `GROUP BY` — ia bukan penyebab baris kembar yang dilaporkan, tetapi penggandanya nyata dan akan muncul begitu master bertambah |

### 22.5 Kenapa tidak ada uji yang menangkapnya

Karena tidak ada yang menguraikan SQL. Penyimpanan memori menyimpan tanggal sebagai teks
`dd/mm/yyyy`, jadi pengelompokannya memang per hari — **fake-nya benar, yang nyata yang salah**.
Ini kelas cacat yang sama dengan `ORA-01008` dan `TGLPROSES` sebagai bagian PRIMARY KEY: hanya
Oracle sungguhan yang dapat menunjukkannya.

Penjaganya sekarang dua lapis:

| Penjaga | Menangkap |
|---|---|
| `TestDayGroupingDoesNotRelyOnDateCast` | `CAST(... AS DATE)` kembali dipakai untuk pengelompokan hari |
| `-periksa`: baris kembar per perusahaan | pengelompokan yang pecah, dari sebab apa pun — termasuk yang belum terpikirkan |

Yang kedua lebih berharga: ia memeriksa **akibatnya di data**, bukan bentuk kuerinya.
## 17. Modul Master Tipe Surveyors (2026-09-19, sesi kesembilan)

Modul master data keempat, dan yang **pertama** menyimpang dari pola Master Status Klaim pada hal
yang mendasar: lingkup portalnya.

### 17.1 Per portal, bukan portal utama — dan itu perbedaan yang disengaja

**Keputusan Work Owner 2026-09-19.** Modul ini membaca dan menulis basis data **entitas yang sedang
dipilih pengguna**, bukan basis data portal utama.

Itu membuatnya berbeda dari dua modul master yang sudah ada:

| Modul | Lingkup | Alasan |
|---|---|---|
| Master Status Klaim | portal utama | keputusan saat modul bisnis pertama dibangun; tidak ditinjau ulang |
| Master Rekening | portal utama | idem |
| Master Status Progres 1 | **per portal** | `GCNM_MST_PROGRESS_KLAIM` ada di basis data setiap entitas |
| **Master Tipe Surveyors** | **per portal** | `M_SURVEYORS` ada di basis data setiap entitas |

Dasarnya `D-75` butir 4 — **master data per portal** — dan `ADR-0030` Opsi 1, yang menempatkan
pemisahan antarentitas di tingkat **koneksi**, bukan di tingkat penyaringan baris. Tidak ada satu
pun kueri modul ini yang menyaring berdasarkan entitas, dan memang tidak boleh ada.

**Konsekuensi yang diterima:** dua modul master yang sudah ada kini berbeda lingkup dari dua yang
lebih baru. Itu **tidak diperbaiki sesi ini** — menyentuh Master Status Klaim dan Master Rekening
melanggar Isolasi Protektif, dan perubahannya bukan perapian melainkan perubahan perilaku: layar
yang selama ini menampilkan data ASM akan mulai menampilkan data entitas yang dipilih. Ia keputusan
tersendiri yang perlu diambil Work Owner, bukan efek samping modul baru.

**Yang dipakai untuk menegakkannya** sama persis dengan Master Status Progres 1:

- `mastertipesurveyors.RepoSelector` — fungsi yang memilih repo satu portal, dan **menghasilkan
  galat** bila portalnya tidak dikenal atau koneksinya belum hidup. Tidak pernah jatuh ke portal
  utama sebagai cadangan (`R-20`).
- Middleware `portalhttp.ActivePortal` dipasang **di dalam `Mount`**, bukan di `cmd`, karena
  SELURUH rute modul ini menyentuh basis data entitas — berbeda dari Master Status Progres yang
  punya satu rute (`posisi-klaim`) yang tidak.

### 17.2 Tidak ada migrasi pemindahan data — dan itu bukan kelalaian

Modul ini **tidak** menyalin langkah migrasi `0002`. Alasannya dibaca dari katalog basis data, bukan
disimpulkan dari pola master:

| | Master Status Klaim | Master Tipe Surveyors |
|---|---|---|
| Kolom label | `LSC_NOTE` ada tetapi **kosong** pada 32 baris | `DESCRIPTION` ada dan **sudah terisi** pada 4 baris |
| View | membaca `JSON_VALUE(JSONDATA, '$.LSC_NOTE')` | **sudah membaca kolom** `DESCRIPTION` |
| Menuntut `CREATE OR REPLACE VIEW`? | **ya** — menyentuh 23 rule Pega | **tidak** |
| Menuntut pemindahan isi JSON ke kolom? | **ya** | **tidak** |

Akibatnya **modul ini bekerja penuh walau migrasi `0003` belum dijalankan DBA** — berbeda dari
Master Status Klaim, yang daftarnya tampil tanpa label sampai migrasi `0002` jalan.

Yang hilang tanpa migrasi `0003` hanya satu: penegakan keunikan nama **di basis data**. Aplikasi
tetap menolak nama ganda; yang tidak terjaga adalah dua permintaan yang tiba bersamaan.

### 17.3 Yang ditulis saat menyimpan: `DESCRIPTION` saja

**Keputusan Work Owner 2026-09-19:** *"save ke kolom DESCRIPTION saja karena kolom json_data sudah
tidak mau dipakai"*.

Konsekuensi yang disadari dan dicatat, bukan disembunyikan:

1. **Baris baru meninggalkan `JSON_DATA` bernilai NULL.** Itu sah: constraint `VALID_JSON_DATA1`
   berbunyi `JSON_DATA IS JSON (STRICT)`, dan pemeriksaan semacam itu menghasilkan UNKNOWN — bukan
   FALSE — untuk NULL, sehingga barisnya diterima.
2. **Baris lama yang diubah akan memuat JSON yang usang.** Tidak ada rule Pega yang membacanya;
   satu-satunya yang menyentuhnya adalah `PEGA_M_SURVEYORS`, yang sejak sekarang tidak dipanggil
   siapa pun.
3. Kapan `JSON_DATA` boleh dibuang adalah keputusan tersendiri, sebaiknya setelah masa pengamatan.

Diuji, bukan sekadar dijanjikan: `TestWriteQueriesDoNotTouchJSONColumn` gagal bila ada kueri tulis
yang menyebut kolom itu.

### 17.4 Kenapa modulnya `mastertipesurveyors`

`D-81` menetapkan nama folder modul mengikuti **nama modul bisnis** yang disebut Work Owner. Nama
itu tidak dikarang — ia ada di `Database/m_menu_aplikasi_pnc.csv` sebagai `MENU_ID 14`:
**"Master Tipe Surveyors"**.

| Lapisan | Bentuk |
|---|---|
| Folder backend & paket Go | `mastertipesurveyors` — huruf kecil, tanpa tanda hubung |
| Folder frontend | `master-tipe-surveyors` — `kebab-case` |
| Rute API | `/api/master/tipe-surveyor` — **tunggal** |
| Rute layar | `/master/tipe-surveyor` — **tunggal** |

Rutenya tunggal sementara nama modulnya jamak, dan itu disengaja: rute master yang sudah ada
seluruhnya tunggal (`status-klaim`, `rekening`, `status-progres-1`). Menyeragamkan jalurnya lebih
berguna daripada menyeragamkannya dengan nama modul, karena jalur adalah **kontrak** sedangkan nama
modul adalah nama internal.

Isi modulnya tetap berbahasa Inggris (`D-80`): tipe `SurveyorType`, field `Code`, `Description`,
`LegacyCode`. Nama field JSON tetap Indonesia karena ia kontrak: `kode`, `deskripsi`, `kode_lama`.

### 17.5 Tidak ada Hapus, dan buktinya tiga lapis

Sama seperti master lain, tetapi di sini bukti penolakannya lebih kuat dan layak disebut:

1. **Layar Pega tidak punya tombol hapus** — `Section/GridSurveyors-Section.xml` hanya memuat
   **Tambah** dan **Refresh**.
2. **`PEGA_M_SURVEYORS.prc` hanya mengenal INSERT dan UPDATE.** Pemindaian seluruh export untuk
   `(INSERT INTO|UPDATE|DELETE FROM) …M_SURVEYORS` menemukan **satu berkas saja**, dan tidak ada
   `DELETE` di dalamnya.
3. **Barisnya dirujuk data yang sedang berjalan** — `V_D_SURVEYORS` pada 2026-09-19 memuat 19
   surveyor bertipe `1001`, 18 bertipe `1002`, dan 6 bertipe `1004`. Ditambah tiga kueri Pega yang
   mematok kodenya langsung.

Penegakannya di tiga tempat: seam tidak menyediakan operasinya, rute tidak mendaftarkan metodenya,
dan uji `TestNoQueryDeletes` menolak kueri yang memuat `DELETE`, `TRUNCATE`, atau `DROP`.

### 17.6 Batas panjang 100 karakter — keputusan, bukan bacaan

**Ditetapkan Work Owner 2026-09-19.** Ia satu-satunya angka di modul ini yang **tidak** dapat
dibaca dari mana pun, sehingga ia memang harus diputuskan — dan sampai diputuskan, ia ditandai
sebagai asumsi di kode maupun di sini.

Yang diketahui pasti: kolomnya `VARCHAR2(4000)`, dan layar Pega tidak membatasi apa pun
(`pyMaxLength` kosong). Keduanya tidak memberi angka yang berguna — nama tipe sepanjang 4000
karakter akan merusak setiap grid yang menampilkannya.

Seratus dipilih karena ia angka yang sama dengan `masterstatus.MaxLabelLength` dan
`masterstatusprogres.MaxNameLength`, sehingga batas yang dilihat pengguna seragam antarlayar master.
Nilai terpanjang yang benar-benar ada hari ini 17 karakter.

Bila diubah, dua tempat wajib ikut berubah — domain Go dan skema Zod di form — dan komentarnya
sudah menyebut itu di keduanya.

**Satu akibat yang menuntut pemeriksaan sebelum modul dipakai:** batas ini baru, sehingga baris
lama yang namanya melebihi 100 karakter tidak akan hilang tetapi **tidak dapat disunting** lewat
layar baru. Pemeriksaannya ditambahkan sebagai langkah `0b` migrasi `0003`. Pada ASM hasilnya nol
baris; portal lain belum dapat diperiksa.

### 17.7 Besar-kecil huruf tidak diseragamkan

Keempat nilai yang ada seluruhnya huruf besar, dan godaan untuk menambahkan `toUpperCase` pada
penyimpanan nyata. **Tidak dilakukan.**

`Activity/ValidasiMasterSurveyor-Act.xml` memang memanggil `@toUpperCase`, tetapi atas
`TempDetailSurveyors.NAME` — yaitu nama **surveyor** di `D_SURVEYORS`, bukan deskripsi **tipe** di
`M_SURVEYORS`. Tidak ada satu pun rule yang menyeragamkan yang kedua.

Menambahkannya berarti mengarang perilaku yang tidak pernah ada, dan `P-5` menetapkan perilaku
dipertahankan lebih dulu. Yang diseragamkan hanyalah **perbandingan** saat menguji keunikan
(`DescriptionKey`), dan ekspresinya sepadan dengan indeks `UPPER(TRIM(DESCRIPTION))` pada migrasi
`0003` — keduanya harus sama, atau aplikasi akan menerima nama yang kemudian ditolak basis data.

### 17.8 Utang yang diwarisi, bukan dibuat

Ketiganya sudah ada sebelum modul ini dan tetap ada sesudahnya:

| Utang | Keadaan |
|---|---|
| **Kewenangan peran belum diperiksa** | Rutenya di balik sesi dan portal, tetapi "apakah pemanggil punya menu Master Data" (`D-59`) belum ditegakkan — `TKT-F3-005`, yang bergantung pada tabel peran `TKT-F3-004` yang dapat dibangun tetapi belum dapat diisi |
| **Kontrak galat belum seragam** | Modul ini memakai `detail: [{field, pesan}]` mengikuti `masterstatus`, sementara `masterstatusprogres` memakai `kolom`. Keduanya sudah ditampung `APIError.violations()` di frontend. Penyeragamannya `TKT-F1-004` |
| **Jalur API tanpa `/v1`** | `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`; memperkenalkannya di satu modul akan membuat dua gaya jalur hidup berdampingan |

### 17.9 Temuan yang perlu jawaban, bukan perbaikan

**Layar Master Tipe Surveyors di Pega efektif tidak berfungsi untuk menyimpan.** Procedure-nya
menulis `JSON_DATA`, view-nya membaca `DESCRIPTION`, dan tidak ada trigger yang menghubungkan
keduanya (`ALL_TRIGGERS` nol baris).

Bukan cacat yang dibuat modul ini, dan bukan pula sesuatu yang perlu diperbaiki di sisi Pega —
begitu Go menjadi penulis tunggal, yang ditulis adalah kolom yang memang dibaca. Tetapi ia
menjelaskan satu hal yang berguna: bila Work Owner mengira layar Pega itu masih dipakai untuk
menambah tipe, penambahan terakhir dari sana kemungkinan besar muncul dengan deskripsi kosong.

Rinciannya di `catatan-pengembangan.md` §17.4.

---

## 18. Modul Master PIC Teknik (2026-09-19, sesi kesepuluh)

Layar `Harness/UserTeknisInbox` atas `POOLDATA.MST_USER_TEKNIK`, butir menu `MENU_ID 13`.

### 18.1 Modul yang sudah ada dibawa ke standar yang berlaku, bukan ditambal

`internal/masterpicteknik` sudah ada sebelum sesi ini, tetapi ditulis **sebelum `D-80`** dan
sebelum pola portal-aware: berkasnya `galat.go`/`rute.go`/`kueri.go`, foldernya `repo/memori/`,
identifiernya `PICTeknik`/`Layanan`/`Daftar`, dan usecase-nya memegang **satu `Repo`**.

Tiga pilihan ditimbang:

| Pilihan | Akibat |
|---|---|
| Biarkan, tambahkan wiring saja | Modul bisnis kedelapan memakai gaya yang sudah dicabut `D-80`, dan **tidak portal-aware** — melanggar `ADR-0030` |
| Tambal sebagian | Setengah berkas Inggris setengah Indonesia; lebih membingungkan daripada keduanya |
| **Tulis ulang seluruhnya mengikuti `mastertipesurveyors`** | Dipilih |

Yang menentukan bukan kerapian melainkan `RepoSelector`: tanpa itu, satu instans `Repo` melayani
seluruh entitas, dan daftar petugas satu badan hukum dapat terbaca dari badan hukum lain (`R-20`).
Itu cacat yang tidak terlihat sebagai galat — layarnya tampak normal, isinya masuk akal, dan yang
salah hanya milik siapa data itu.

**Isolasi Protektif tetap dipatuhi**: modul Login, Home, dan Master Data yang sudah selesai tidak
disentuh satu baris pun.

### 18.2 Daftar hanya menampilkan petugas aktif — dan akibatnya diterima sadar

Report Definition `BrowseVMstUserTeknis_RD` memasang dua penyaring bergabung `A AND B`, dan yang
kedua dipatok: `STS_AKTIF = '1'`.

**Keputusan Work Owner: dipertahankan persis.** Saya menyampaikan konsekuensinya lebih dulu — di
Pega, petugas yang dinonaktifkan hilang dari layar dan tidak dapat ditemukan lagi — dan jawabannya
ditegaskan.

Yang saya lakukan sesudahnya bukan menawar keputusan itu melainkan **memeriksa sejauh mana jalan
buntunya**. `GetMasterPICTeknis` ternyata **tidak menyaring status aktif sama sekali**, sehingga:

| Jalur | Menyaring aktif? |
|---|---|
| Daftar (`technician_list`, dari view) | **ya** — sama dengan Pega |
| Ambil satu baris (`technician_get`, dari tabel) | **tidak** — sama dengan Pega |

Artinya petugas nonaktif hilang dari daftar tetapi **tetap terjangkau lewat ID dan dapat diaktifkan
kembali**. Jalan buntunya tidak lebih dalam daripada di sistem lama, dan itu dicapai dengan meniru
Pega lebih teliti — bukan dengan menyimpang darinya.

Layar menyatakannya terang-terangan (*"Petugas nonaktif tidak ditampilkan, sama seperti di sistem
lama"*), dan form memperingatkan **sebelum** menyimpan bahwa menonaktifkan akan menghilangkan baris
dari daftar. Tanpa itu, penonaktifan akan dilaporkan pengguna sebagai kehilangan data.

### 18.3 Daftar dibaca dari VIEW, penulisan selalu ke TABEL

| Objek | Peran |
|---|---|
| `POOLDATA.V_MST_USER_TEKNIS` | dibaca untuk daftar — memuat `TOTAL_JOB` yang tidak ada di tabel |
| `POOLDATA.MST_USER_TEKNIK` | dibaca untuk satu baris, dan **satu-satunya yang ditulis** |

Ejaannya berbeda dan itu bukan salah ketik: view berakhiran **TEKNIS**, tabel berakhiran **TEKNIK**.
Keduanya terbaca apa adanya dari export.

Pembagian ini meniru sistem lama persis — RD berkelas `...-Int-V_MST_USER_TEKNIS`, sementara
`GetMasterPICTeknis` dan procedure penulisnya menyebut `MST_USER_TEKNIK`.

**Satu hal BELUM terverifikasi dan dicatat terbuka.** Definisi view tidak ada di export; nama
kolomnya **diduga** `OLD_OPERATOR_ID` — diturunkan dari nama properti kelas Pega, yang memetakan
properti ke kolom bernama sama. Bila dugaan itu keliru, yang berubah **hanya kueri
`technician_list`**; tidak ada satu baris kode Go yang ikut berubah. `claimpnc -periksa` melaporkan
kegagalannya beserta kueri `all_views` yang perlu dijalankan DBA.

### 18.4 `PR_OPERATORS` ditinggalkan; direktori pegawai menjadi seam tersendiri

Sistem lama mencari nama **dua tingkat**: tabel operator milik Pega, lalu — bila kosong — layanan
REST. **Keputusan Work Owner: tinggalkan tabel operator Pega seluruhnya**, pakai layanan saja.

Itu lurus dengan arah migrasi: `DATAPEGA.PR_OPERATORS` milik engine Pega, dan bergantung padanya
berarti modul ini ikut mati ketika Pega dimatikan.

Alamat layanannya dibaca **per entitas**:

```sql
SELECT servicename FROM POOLDATA.GCNM_CONNECT_REST
 WHERE app = <portal_alias> AND typeservice = 'HCQ-LOGIN';
```

ID yang diketik dicocokkan ke **LOGIN**, dan `TYPESERVICE` yang dipakai **`'HCQ-LOGIN'`** — keduanya
ditetapkan Work Owner. Saya melaporkan bahwa Pega memakai `'GetEmployee'` untuk pencarian ini;
keputusannya menyatukan keduanya ke satu baris layanan.

**Barisnya sudah ada, dan itu terverifikasi** — bukan diandaikan. `Database/gcnm_connect_rest.csv`
memuat `SERVICEID 79` dengan `APP='ASM'` dan `TYPESERVICE='HCQ-LOGIN'`, dan baris yang sama itulah
yang **sudah dipakai modul Login hari ini** lewat kueri `service_address` pada
`internal/auth/repo/sqlstore/legacy.sql`. Tidak ada permintaan baru ke DBA untuk portal utama.

Nilainya tetap dapat dikonfigurasi lewat `Options.ServiceKind`, sehingga pemindahan ke layanan
tersendiri kelak menjadi perubahan konfigurasi, bukan rilis ulang.

### 18.5 Satu pencarian mengisi DUA isian

Respons HCQ memuat blok `EmpLeader` — terbukti dari contoh respons yang diberikan Work Owner
2026-09-16 dan tersimpan di `internal/auth/provider/hcq_test.go:52`. Satu pencarian karena itu
menjawab **Nama** sekaligus **Atasan**.

Itu memecahkan dua hal sekaligus: pengganti isian Atasan yang di Pega berupa autocomplete atas
daftar operator (yang kini ditinggalkan), dan pengisian nama yang memang tidak pernah diketik.

**Usulan direktori tidak menimpa pilihan pengguna.** Atasan hanya diisi bila pengguna
mengosongkannya, karena atasan menurut struktur organisasi belum tentu atasan penanganan klaim —
dan layar Pega pun menandai isian itu `pyReadOnly=false`. Aturan yang sama diterapkan **di server**,
bukan hanya di layar, supaya pemanggilan API langsung berperilaku sama.

Nama, sebaliknya, **selalu** ditimpa hasil pencarian. Menerimanya dari klien akan membuat master
memuat nama yang tidak cocok dengan identitasnya, dan setiap layar penugasan akan menyebut orang
yang berbeda dari yang sesungguhnya bertugas.

### 18.6 Adapter direktori berdiri sendiri, modul Login tidak disentuh

`auth/provider.HCQ.Verify` **tidak** dipakai ulang. Keduanya menembak endpoint yang sama tetapi
menanyakan hal berbeda:

| | Pertanyaannya |
|---|---|
| `auth/provider.HCQ.Verify` | benarkah sandi ini milik orang ini? |
| `masterpicteknik/directory.HCQ.Lookup` | siapa orang dengan identitas ini, dan siapa atasannya? |

`Verify` menolak dengan `ErrWrongCredential` begitu `pyErrorCode` bukan `"200"`, dan membuang
seluruh blok `EmpResponse` — justru bagian yang dibutuhkan pencarian. Memakai ulangnya menuntut
menyunting modul yang sudah dinyatakan selesai.

Yang **dipakai ulang** adalah pembaca katalognya: `sqlstore.Legacy` disuntikkan dari `cmd` lewat
antarmuka sempit `ServiceCatalog` yang dideklarasikan modul ini sendiri. Kedua modul tetap tidak
saling mengimpor — yang tahu keduanya hanyalah berkas perakitan.

Galat `provider.ErrServiceNotRegistered` diteruskan sebagai **nilai**, bukan diimpor tipenya. Itu
yang membuat modul ini dapat membedakan katalog yang belum diisi dari jaringan yang putus, tanpa
bergantung pada modul auth.

### 18.7 Sandi pengisi, dan kenapa `pyErrorCode` tidak dipakai

Permintaan ke endpoint itu menuntut sepasang Login dan Password. Untuk pencarian nama tidak ada
sandi yang dapat diberikan — yang dicari justru orang lain, bukan pemanggil.
`SetMstUserTeknisMstUser_act` langkah 6 mengisinya `"1"`, dan nilai itu dipakai sebagai bawaan.

Konsekuensinya mengikat: **`pyErrorCode` tidak dapat dipakai sebagai penentu**, karena sandi pengisi
memang tidak pernah benar. Yang menentukan adalah ada-tidaknya nama di dalam blok `EmpResponse` —
bila ada, pegawainya dikenal; bila tidak, ia dianggap tidak terdaftar.

**Yang belum dapat dibuktikan:** apakah HCQ benar-benar mengisi `EmpResponse` ketika sandinya salah.
Itu hanya terjawab dengan memanggilnya sungguhan — alamatnya sudah terbaca dari katalog, tetapi
panggilan sungguhan belum pernah dijalankan dari lingkungan ini. Bila ternyata
tidak, yang berubah adalah `Options.ServiceKind` — diarahkan ke layanan pencarian tersendiri — tanpa
menyentuh domain maupun usecase.

### 18.8 Tiga nilai yang tidak boleh datang dari klien

`nama`, `grup_panel`, dan `beban_kerja` **ditolak** bila ada di badan permintaan, bukan diabaikan
diam-diam — `decoder.DisallowUnknownFields()`.

| Nilai | Kenapa |
|---|---|
| `nama` | dimiliki direktori; dicari server setiap kali menyimpan |
| `grup_panel` | procedure lama tidak pernah menulisnya, pada cabang INSERT maupun UPDATE |
| `beban_kerja` | milik view, dihitung, dan tidak ada di tabel yang ditulis modul ini |

Mengabaikannya diam-diam akan membuat klien yang mengirimnya mengira nilainya tersimpan.

### 18.9 Penyimpangan dari sistem lama yang disengaja

| Hal | Pega | Di sini | Alasan |
|---|---|---|---|
| Perbandingan `OPERATOR_ID` | procedure memakai `=` tanpa `UPPER`; pencarian nama memakai `UPPER` | **`UPPER` di semua jalur** | Ketidakkonsistenannya membuat "BUDI" dan "budi" dapat menjadi dua baris untuk orang yang sama |
| Email kosong | diterima | **ditolak** | Petugas tanpa surel tidak menerima satu pun pemberitahuan penugasan, dan ketiadaannya baru ketahuan jauh dari layar ini |
| WHERE dirangkai `{ASIS:InputCOL.OPERATOR_ID}` | ya | **parameter binding** | Celah injeksi (utang teknis 4.5) |
| Penulisan lewat `PEGA_MST_USER_TEKNIS` | ya | **SQL langsung dari Go** | `D-02`; ditambah `ErrMsg` yang pada jalur BERHASIL berisi *"Data Sudah Disimpan dengan ID : ..."* sehingga berhasil dan gagal tidak dapat dibedakan tanpa membaca teks (`D-68`) |
| Pencarian nama | berjalan diam-diam | **punya tombol, hasilnya terlihat sebelum Simpan** | Di Pega kegagalannya baru muncul sebagai penolakan setelah seluruh form diisi |

Satu hal yang **tidak** diperbaiki: generator `id_site || lpad(MST_USER_TEKNIS_SEQ.nextval,6,'0')`
yang dihitung procedure lalu tidak pernah dipakai. Ia kode mati, dan tidak dibawa — bukan
diperbaiki, karena tidak ada yang rusak.

### 18.10 Keberadaan diperiksa di dalam transaksi, bukan diserahkan ke batasan unik

DDL `MST_USER_TEKNIK` tidak ada di export (`R-08`), sehingga **keberadaan batasan unik pada
`OPERATOR_ID` tidak dapat dipastikan**. Mengandalkannya saja berarti bertaruh: bila ternyata tidak
ada, dua baris untuk satu petugas tersimpan diam-diam dan penugasan klaim menjadi tidak dapat
ditebak.

`Insert` karena itu memeriksa keberadaan **di dalam satu transaksi** dengan penyisipannya. Deteksi
`ORA-00001` tetap dipasang sebagai jaring pengaman bila batasannya ternyata ada dan dua permintaan
tiba bersamaan — dicocokkan lewat **kode**, bukan nama constraint yang tidak diketahui.

### 18.11 Yang belum dapat dibuktikan

| Hal | Sebab |
|---|---|
| Nama kolom `V_MST_USER_TEKNIS` | definisi view tidak ada di export; dugaannya dari nama properti kelas Pega |
| Perilaku HCQ saat sandi pengisi salah | hanya terjawab dengan memanggil layanan sungguhan; alamatnya sudah terbaca, tetapi panggilannya belum pernah dijalankan dari lingkungan ini |
| Keberadaan batasan unik `OPERATOR_ID` | DDL tabel tidak ada (`R-08`) |
| Isi nyata `MST_USER_TEKNIK` | tidak ada CSV maupun dump di export; contoh di `repo/memory` **dikarang** dan memakai domain `example.invalid` |

Keempatnya dilaporkan `claimpnc -periksa` atau dinyatakan di komentar berkasnya, bukan disimpan
sebagai pengetahuan yang hanya ada di kepala.

---

## 19. Penyelarasan lingkup portal: Master Status Klaim & Master Rekening (2026-09-19)

**Keputusan Work Owner 2026-09-19**, sekaligus **mencabut Isolasi Protektif** untuk kedua modul
itu — keduanya termasuk "Master Data yang sudah selesai" yang instruksi tetap larang disentuh.

### 19.1 Apa yang salah sebelumnya

Empat modul master dibangun berurutan, dan dua yang pertama memakai lingkup yang keliru:

| Modul | Sebelum | Sesudah |
|---|---|---|
| Master Status Klaim | portal **utama** saja | **per portal** |
| Master Rekening | portal **utama** saja | **per portal** |
| Master Status Progres 1 | per portal | tidak berubah |
| Master Tipe Surveyors | per portal | tidak berubah |
| Master PIC Teknik | per portal | tidak berubah |

Dasarnya `D-75` butir 4 — **master data per portal** — dan `ADR-0030` Opsi 1, yang menempatkan
pemisahan antarentitas di tingkat **koneksi**. Kedua modul pertama menyimpang darinya, dan
penyimpangan itu tidak pernah diputuskan; ia hanya belum ditinjau ketika modul bisnis pertama
dibangun.

**Akibat nyatanya bukan soal kerapian.** `POOLDATA.LST_ACCOUNT` dan `POOLDATA.M_STS_CLAIM` ada di
basis data setiap entitas. Melayani keduanya dari portal utama berarti:

- Rekening pembayaran klaim SELURUH badan hukum terbaca dan tertulis di satu basis data —
  lengkap dengan nomor rekening, NIK, dan surel pihak ketiga.
- Status klaim satu entitas dipakai layar entitas lain.

Keduanya kegagalan yang **tidak terlihat sebagai galat**: layar tampil normal, angkanya masuk akal,
dan yang salah hanya milik siapa data itu. Persis `R-20`.

### 19.2 Bentuk yang dipakai, dan kenapa keduanya berbeda

Dua modul, dua bentuk — dan perbedaannya bukan selera.

**Master Status Klaim → `RepoSelector`**, sama dengan Master Status Progres dan Master Tipe
Surveyors. Layanannya hanya memegang penyimpanan, sehingga alias portal cukup menjadi parameter
tiap method: `List(ctx, portalAlias)`, `Create(ctx, portalAlias, label)`, dan seterusnya.

**Master Rekening → `ServiceSelector`**, satu LAYANAN per portal. Bentuknya berbeda karena
layanannya memegang lebih dari penyimpanan: seam Kasir, seam Notifier, jam, dan — yang menentukan —
`portalAlias` yang dipakai `portalsRegisteredWithCashier` untuk memutuskan **apakah rekening yang
disetujui didaftarkan ke Kasir** (`usecase/decide.go:131`).

Menjadikan alias sebagai parameter method di sana berarti setiap method harus merakit ulang
keputusan itu. Satu layanan per portal membuat `portalAlias` tetap menjadi **sifat layanan**, bukan
parameter yang dapat tertukar.

### 19.3 Satu cacat yang ikut terperbaiki tanpa diminta

Sebelum penyelarasan, `PortalAlias` yang dipakai Master Rekening **selalu portal utama**
(`cfg.PrimaryPortal`), apa pun entitas yang sedang dilihat pengguna.

Artinya keputusan "daftarkan rekening ini ke Kasir atau tidak" dijawab dengan **entitas yang salah**
bagi setiap pengguna yang sedang melihat entitas selain portal utama — mendaftarkan yang seharusnya
tidak, atau melewatkan yang seharusnya didaftarkan. Sekarang ia dijawab dengan entitas yang
benar-benar dipilih.

Cacat ini tidak pernah dilaporkan siapa pun, dan tidak akan muncul sebagai galat. Ia ditemukan
karena `portalAlias` harus dibaca ulang untuk memindahkan modulnya.

### 19.4 Yang TIDAK berubah, dan itu disengaja

| Hal | Keterangan |
|---|---|
| Seam Kasir, Notifier, dan jam | **dipakai bersama** seluruh entitas — ketiganya konfigurasi tingkat aplikasi, bukan data entitas |
| Aturan bisnis kedua modul | tidak satu baris pun disentuh; yang berpindah hanya **dari basis data mana** ia bekerja |
| Bentuk kontrak API | jalur dan badan permintaan tetap sama. Yang bertambah: header `X-Portal` wajib, dan respons menyebut `portal` yang menjawab |
| Keunikan nomor rekening | tetap **per entitas**, bukan lintas entitas — nomor yang sama di dua badan hukum bukan duplikat, dan itu diuji |

### 19.5 Akibat yang harus disadari sebelum dipakai

1. **Layar kedua modul sekarang MENOLAK dilayani tanpa portal.** Pengguna yang belum memilih
   entitas melihat penjelasan, bukan daftar. Di layar itu praktis tidak terasa: pemilih portal di
   bilah atas memilih portal utama dari server secara otomatis.
2. **Isi yang tampil dapat berubah bagi pengguna yang memilih entitas selain ASM.** Itu memang
   maksudnya — tetapi bagi pengguna yang terbiasa selalu melihat data ASM, perubahannya nyata.
3. **Tidak dapat diverifikasi untuk lima entitas.** Kredensial ASI, SMAS, SMI, SPK, dan SPKS belum
   terisi di lingkungan pengembangan, sehingga yang terbukti hanya perilaku terhadap ASM dan
   terhadap adapter memori. Ini **penghalang yang sama** dengan migrasi `0003`.
4. **Layanan per portal dibuat saat pertama diminta lalu dipakai kembali.** Ia tidak pernah
   dilepaskan selama proses hidup; dengan enam entitas, itu enam layanan — bukan beban yang layak
   dioptimalkan.

### 19.6 Yang diuji, dan yang sengaja diuji lebih keras

Uji HTTP baru ditulis untuk **kedua** modul, dan yang diuji bukan "handler-nya bekerja" melainkan
**penjagaannya menolak**:

| Perkara | Master Status Klaim | Master Rekening |
|---|---|---|
| Tanpa header portal → `400` pada setiap rute | ✅ 4 rute | ✅ 6 rute |
| Portal tidak dikenal → `400`, belum siap → `503` | ✅ | ✅ |
| Tiap entitas menjawab dengan isinya sendiri | ✅ | ✅ |
| Menulis di satu entitas tidak menyentuh entitas lain | ✅ | ✅ |
| Nomor/kunci yang sama di dua entitas bukan duplikat | — | ✅ |

Master Rekening **belum punya uji HTTP sama sekali** sebelum ini. Menambahkannya bersamaan dengan
perubahan rutenya disengaja: perubahan yang menyentuh pemisahan antarbadan hukum tidak layak
diserahkan tanpa uji yang membuktikan pemisahannya.

### 19.7 Satu uji yang sempat lulus karena alasan yang salah

`TestOversizedRequestBodyRejected` pada Master Status Klaim memeriksa badan permintaan raksasa
ditolak `400`. Setelah middleware portal dipasang, ia **tetap lulus** — tetapi `400`-nya datang dari
pemeriksaan portal, bukan dari penolakan badan permintaan.

Diperbaiki dengan mengirim header portal DAN memeriksa **kode galatnya**, bukan hanya statusnya.
Dua sebab yang menghasilkan status sama harus dapat dibedakan uji, atau uji itu berhenti menguji
apa yang namanya janjikan.

---

## 20. Modul Master Surveyors (2026-09-20, sesi kesebelas)

Modul `MENU_ID 15` (`DetailSurveyorsInbox`) atas tabel `POOLDATA.D_SURVEYORS` — daftar ORANG
yang melakukan survei, anak dari Master Tipe Surveyors yang berisi golongannya.

### 20.1 Keputusan Work Owner

| # | Perkara | Keputusan | Tanggal |
|---|---|---|---|
| 1 | Lingkup modul | **Alur persetujuan komite penuh**, seperti Master Rekening | 2026-09-19 |
| 2 | Verifikasi struktur tabel | **Tanpa kueri katalog** — dirancang dari export Pega saja | 2026-09-19 |
| 2b | `V_D_SURVEYORS` membaca kolom atau JSON? | **Membaca KOLOM**, tidak memakai `JSON_DATA` lagi → migrasi 0004 langkah 3 dicabut | 2026-09-20 |
| 3 | `BRANCH` / `LOGIN_APLIKASI` / `DOCID` | Mengikuti perilaku Pega apa adanya | 2026-09-19 |
| 4 | Aturan Approve/Reject yang disangka hilang | **Ternyata ADA** — satu activity, parameter `approval` (§20.2). Penurunan dari `masterrekening` tidak jadi dipakai | dikoreksi 2026-09-20 |
| 5 | Pembuatan akun surveyor | Perilaku Pega dibawa; tabel operator Pega **tidak** ditulis | 2026-09-19 |

Keputusan 5 adalah bacaan saya atas jawaban "seperti PEGA saja", dan ia **dinyatakan sebagai
asumsi kepada Work Owner sebelum kode ditulis**, bukan diambil diam-diam. Alasannya di §20.4.

### 20.2 Keputusan komite: satu activity, tiga nilai parameter

**Bagian ini ditulis ulang pada 2026-09-20.** Versi sebelumnya menyatakan rule Approve/Reject
**tidak ada di export** dan bahwa bentuknya **diturunkan** dari `masterrekening`. Itu keliru.

Yang memperbaikinya adalah pertanyaan Work Owner: *"rule Approve/Reject ini dipanggil di mana"*.
Saya mencari rule **bernama** Approve/Reject, tidak menemukannya, lalu menyimpulkan ia hilang —
tanpa menelusuri **pemanggilnya**. Kesalahannya bukan pada pembacaan, melainkan pada berhenti
terlalu cepat.

**Yang sebenarnya berlaku.** Tidak ada rule tersendiri karena memang tidak perlu ada. Ketiga
tombol memanggil activity yang **sama**, dibedakan satu parameter — dibaca dari
`Section/BrowseDetailSuveryorsKomite-Section.xml` dan `-Approve`/`-Reject`:

| Tombol | Activity | Parameter |
|---|---|---|
| **Approve** | `CNMInsertDetailSurveyors_act` | `approval = "1"` |
| **Reject** | `CNMInsertDetailSurveyors_act` | `approval = "2"` |
| **Simpan** | `CNMInsertDetailSurveyors_act` | `approval = "0"` |
| **Ubah** | `SetDetailSurveryorsValue_act` | `dsurveyid = .D_SURVEY_ID` |
| **View Document** | flow action `ViewDocumentMasterRekening` | — |

dan di dalam activity, pada langkah 4: `TempDetailSurveyors.APPROVAL := Param.approval`.

Kedua tombol keputusan **hanya ada di tab "Komite Approval"**. Tab Waiting, Approve, dan Reject
tidak memilikinya.

**Akibat terpentingnya: modul ini PUNYA baseline**, sehingga dapat diuji setara dengan Pega.
Pernyataan "tidak dapat lulus gerbang 1" **dicabut**.

**Satu perbedaan yang tetap disengaja, dan dinyatakan.** Sistem lama memakai satu jalan masuk
untuk menyimpan dan memutuskan. Modul ini memisahkannya:

| | Pega | Di sini |
|---|---|---|
| Menyimpan isian | `CNMInsertDetailSurveyors_act(approval="0")` | `POST /api/master/surveyor` · `PUT /{id}` |
| Keputusan komite | `CNMInsertDetailSurveyors_act(approval="1"\|"2")` | `POST /{id}/keputusan` |

Alasannya bukan kerapian: satu jalan masuk berarti badan permintaan yang sama dapat membawa
isian **dan** status persetujuan, sehingga siapa pun yang boleh menyunting surveyor dapat
menyetujuinya sendiri dalam satu permintaan. `D-59` menetapkan tidak ada pemisahan tugas formal,
jadi bentuk kontraknya adalah satu-satunya penjagaan yang tersisa.

Keadaan akhir yang tersimpan **tetap sama**: `APPROVAL` berisi `"0"`, `"1"`, atau `"2"`.

### 20.2b Saringan tab: terbukti, bukan ditafsirkan

`BrowseVDSurveyors_RD` berparameter, dan tiap section mengisinya berbeda (`pyRDParams`):

| Tab | `Approve` | `Komite` |
|---|---|---|
| Waiting Approval | `"0"` | (kosong) |
| Komite Approval | `"0"` | `OperatorID.pyUserIdentifier` |
| Approve | `"1"` | (kosong) |
| Reject | `"2"` | (kosong) |

Baris kedua membuktikan **dua hal sekaligus**: antrean komite menyaring dengan identitas operator
yang sedang masuk, dan kolom `KOMITE` memang berisi **Operator ID** — menguatkan temuan soal
alias `BUSINESS_CODE` di §20.6. Saringan di `surveyor_list` sudah sesuai dan tidak perlu diubah.

`PNCIsInternalSurveyors` juga terbaca: isinya `TempDetailSurveyors.M_SURVEY_ID = "1001"`, sama
persis dengan konstanta `InternalTypeCode`.

### 20.2c Dua CACAT implementasi yang ditemukan dari penelusuran lanjutan

Work Owner menanggapi §20.2 dengan: *"masalahnya apa, kalau Approve parameter approval='1' dan
kalau reject approval='2' … coba dicek lg"*.

Mekanismenya memang sesederhana itu. Yang keliru bukan Pega-nya — **implementasi saya**, dan
pemeriksaan ulang menemukan dua cacat.

#### Cacat 1 — email diperlakukan opsional, padahal WAJIB

Langkah 4 `CNMInsertDetailSurveyors_act` menyiapkan teks galatnya sebagai variabel lokal, dan
langkah 5 memancarkannya:

```
langkah 4   local.email := "Email harus diisi"
langkah 5   precondition: TempDetailSurveyors.EMAIL==""   → pesan local.email
```

Saya sebelumnya menyimpulkan email opsional karena "tidak ada aturan yang mewajibkannya di
export". Aturannya ada; saya belum membaca variabel pesannya.

**Diperbaiki** di domain (`Check()`) dan di skema Zod form, dengan pesan yang sama persis.
Label kolomnya menjadi **Email (wajib)**.

Dua teks galat lain ikut terbaca dari langkah yang sama, dan keduanya menguatkan aturan yang
sudah dibawa: `local.err = "Nama Belum Terdaftar."` dan
`local.errSameOperator = "User ID sudah terdaftar. Silahkan pilih User ID yang lain."`

#### Cacat 2 — menyunting TIDAK mengembalikan surveyor ke antrean komite

Ini yang lebih berat, dan ia justru muncul dari pertanyaan tentang parameter itu.

Langkah 4 berjalan **tanpa syarat**:

```
pyStepsPreCondition: true          (tidak ada precondition when)
TempDetailSurveyors.APPROVAL := Param.approval
```

Tombol **Simpan** pada tab "Approve" dan "Reject" mengirim `approval = "0"`. Artinya menyunting
surveyor yang **sudah disetujui** — atau sudah ditolak — **mengembalikannya ke posisi menunggu**.

Penguatnya: `SetDetailSurveryorsValue_act`, yang mengisi formulir Ubah, **tidak menyalin
`APPROVAL` sama sekali**. Nilainya memang hanya datang dari parameter, tidak pernah dari baris
yang sedang disunting.

Implementasi saya sebelumnya **mempertahankan** status pada `Update`. Akibatnya nama login,
alamat, dan email surveyor dapat diubah **setelah** komite menyetujuinya, dan komite tidak
pernah melihat perubahannya. Itu bukan sekadar beda dari Pega — itu melubangi satu-satunya
kontrol yang ada, karena `D-59` menetapkan tidak ada pemisahan tugas formal.

**Diperbaiki:** `Update` selalu menyetel status ke menunggu. Jejak keputusan sebelumnya
(`TGL_APPROVE_KOMITE`, `CATATAN_KOMITE`) ikut dibuang — baris berstatus menunggu yang memuat
tanggal keputusan adalah keadaan yang tidak dapat dibaca siapa pun. Kedua kolom itu tambahan
modul ini, sehingga pembuangannya tidak punya padanan di Pega dan dicatat sebagai keputusan
tersendiri.

Komite yang ditunjuk **tetap** — ia ditetapkan sekali saat pengajuan pertama, sejalan dengan
kueri `surveyor_update` yang memang tidak menyentuh kolom `KOMITE`.

#### Satu hal yang MASIH terbuka

Precondition langkah 8 menyebut `M_SURVEY_ID` = `1021`, `1022`, `1023`, `1025`, `1026`, `1027`,
`1028` — tujuh kode tipe di luar empat baris yang diverifikasi modul induk.

Ia memancarkan `local.err` (*"Nama Belum Terdaftar."*), **bukan** pesan login — sehingga dugaan
awal bahwa deret itu memperluas kewajiban login **tidak terbukti**. Apa sebenarnya artinya belum
jelas, dan keempat baris master di portal ASM tidak memuat satu pun darinya. Dibawa ke **Tim
Pega**, tidak ditebak.

### 20.3 `V_D_SURVEYORS` membaca KOLOM — ditegaskan Work Owner 2026-09-20

Pertanyaan terbesar modul ini **sudah terjawab**, dan jawabannya menyederhanakan banyak hal.

Latarnya: modul induk menemukan jebakan pada tabel tetangga — `PEGA_M_SURVEYORS` menulis
**hanya** ke `JSON_DATA` sementara `V_M_SURVEYORS` membaca kolom, tanpa trigger yang menyalin.
Layar Pega tampak berhasil menyimpan, tetapi hasilnya tidak pernah terlihat di view.
`PEGA_D_SURVEYORS.prc` memakai **pola yang sama persis**, sehingga pertanyaannya wajar diajukan.

**Jawaban Work Owner:** `V_D_SURVEYORS` membaca **kolom**, tidak memakai `JSON_DATA` lagi.

| Yang berubah karenanya | Isi |
|---|---|
| **Migrasi 0004 langkah 3 DICABUT** | Definisi ulang view tidak diperlukan. Ia tidak dibiarkan sebagai pernyataan berkomentar "untuk jaga-jaga" — mendefinisikan ulang view yang dibaca rule Pega produksi adalah tindakan berisiko, dan menyediakannya pada berkas yang dijalankan DBA adalah undangan yang tidak perlu ada |
| **Langkah 0d turun derajat** | Dari pertanyaan penentu menjadi **laporan keadaan** |
| **Tulisan Go langsung terlihat Pega** | Tanpa satu pun perubahan pada view, sama seperti `M_SURVEYORS` |

**Satu pertanyaan tersisa untuk Tim Pega, dan ia BUKAN penghalang modul ini.**

Bila view membaca kolom sementara `PEGA_D_SURVEYORS` menulis JSON, maka baris yang ditulis
**layar Pega** meninggalkan kolomnya kosong dan tampak hilang di view — cacat yang sama dengan
yang tercatat pada `M_SURVEYORS`, dan ia **sudah berjalan hari ini**.

Bahwa 43 baris yang ada **terbaca** lewat view berarti kolomnya memang terisi, sehingga barisnya
pasti diisi lewat jalur lain: migrasi data, penyisipan langsung, atau trigger. Langkah 0d
memeriksanya sebagai laporan. Modul ini tidak memperbaikinya dan tidak perlu — sejak Go menjadi
penulis tunggal, yang ditulis adalah kolom yang memang dibaca.

**Yang MASIH belum diverifikasi** dan tetap menjadi asumsi: panjang kolom `NAME`, nama constraint
kunci utama, dan tipe data kolom-kolomnya. Ketiganya diminta pada migrasi 0004 langkah 0c.

### 20.4 Akun aplikasi surveyor: perilaku dibawa, tabel Pega tidak ditulis

`CNMInsertDetailSurveyors_act` langkah 18 memanggil `GCNMCreateOperator`, yang membuat instans
`Data-Admin-Operator-ID` — baris pada tabel operator milik **engine Pega**.

Dua hal melarang modul master menulisnya:

- **`P-1` (`D-21`)**: satu tabel hanya boleh ditulis satu sistem selama masa paralel. Tabel
  operator dimiliki Pega sampai `F-3` memindahkannya. Pelanggarannya **tidak menimbulkan pesan
  galat apa pun** — ia baru terlihat sebagai data rusak.
- **`F-3` terhalang**: kontrak HCC/HCQ nol jejak di export (`R-14`, `ADR-0024`).

**Yang diambil:** modul menyatakan kehendaknya lewat seam `AccountRegistrar` dengan **kedelapan
parameter Pega apa adanya**, dan pengisi seam di tahap ini **mencatat** permintaannya tanpa
membuat akun.

Aturan Pega yang **tetap dibawa utuh**: login wajib untuk Internal Surveyor, keunikannya diuji,
sandi sementara `LOGIN_APLIKASI + "123456"`, penanda wajib-ganti-sandi, access group
`GCNMFW:PNCSurveyor`, dan unit `Internal`/`Eksternal`.

**Yang hilang, dan dinyatakan di layar:** surveyor internal yang ditambahkan lewat layar baru
**belum dapat masuk** sampai `F-3` siap. Ini disebut pada panel peringatan di halamannya — bukan
hanya di dokumen ini.

**Koreksi atas kekhawatiran saya sendiri.** Saya melaporkan sandi `login+123456` kepada Work
Owner sebagai masalah keamanan. Setelah `GCNMCreateOperator` dibaca sampai habis, ia menyetel
`pyChangePasswordOnNextLogin = "True"` — sandi itu **sementara dan wajib diganti saat login
pertama**. Risikonya jauh lebih kecil dari yang saya sampaikan. Koreksinya disampaikan **sebelum**
Work Owner mengambil keputusan, sehingga dasar keputusannya benar.

### 20.5 Lima kolom yang DITAMBAHKAN ke tabel lama

`TGL_APPROVE_KOMITE` · `CATATAN_KOMITE` · `USER_INPUT` · `TGL_INPUT` · `USER_UPDATE`.

Kedua Report Definition yang ada tidak memuat satu pun dari kelimanya, jadi ini **penambahan
kemampuan**, bukan pemindahan perilaku.

**Alasannya bukan kerapian.** `D-59` menetapkan satuan izin adalah menu dan **tidak ada pemisahan
tugas formal** — satu orang dapat mengajukan surveyor lalu menyetujuinya sendiri bila perannya
memiliki menu itu. Jejak audit karena itu menjadi **satu-satunya kontrol pengimbang yang
tersisa**, dan keputusan komite tanpa pencatat dan tanpa waktu tidak dapat ditelusuri siapa pun.

Seluruhnya NULLABLE, mengikuti `P-4`: selama rolling deployment versi lama dan baru berjalan
bersamaan terhadap skema yang sama.

**Migrasi 0004 WAJIB** — berbeda dari 0003 yang opsional. Kelima kolom disentuh pada setiap
pembacaan dan penyimpanan; tanpanya layar gagal dengan `ORA-00904` pada permintaan pertama.

### 20.6 Aturan nama ganda lebih ketat dari modul induk

`ValidasiMasterSurveyor` membandingkan dengan `@toUpperCase(@replaceAll(.NAME," ",""))` — huruf
**dan seluruh spasi** diabaikan. Akibatnya `"BUDI SANTOSO"` dan `"budisantoso"` adalah orang yang
sama bagi sistem.

Modul induk memakai `UPPER(TRIM(...))` saja. **Menyalinnya akan melonggarkan aturan yang sistem
lama tegakkan**, dan membuat nama ganda lolos.

Ekspresinya sama persis di tiga tempat, dan ada uji yang menjaganya: `mastersurveyors.NameKey` di
Go, `UPPER(REPLACE(d.NAME, ' ', ''))` di SQL, dan indeks unik `UX_D_SURVEYORS_NAME` pada migrasi
0004. `REPLACE` ditulis tiga argumen, bukan dua — bentuk dua argumen sah di Oracle tetapi tidak
ada di PostgreSQL (`D-20`).

### 20.7 Empat field yang diperlakukan wajib

Tipe surveyor, nama, **email**, dan login aplikasi untuk surveyor internal.

**Email sempat terlewat**, dan itu dibetulkan 2026-09-20 — lihat §20.2d.

**Kenapa hanya tiga.** Formulir masukan layar lama **tidak ada di export** — tidak ada section
maupun flow action yang memuat `TempDetailSurveyors.ADDRESS` dan kawan-kawannya. Artinya field
mana yang bertanda wajib di layar Pega **tidak dapat dibaca**, dan itu bagian dari `R-16`.

Keempatnya wajib karena punya **pesan galat sendiri** atau **aturan yang bergantung padanya** — bukan karena diduga wajib.
Sisanya dibiarkan opsional: menambahkan kewajiban yang tidak terbaca di mana pun akan menolak
data yang hari ini sah — perubahan perilaku yang menyamar sebagai kerapian.

### 20.8 Dua hal yang dipatok, dan kenapa itu benar

| Nilai | Alasan |
|---|---|
| `InternalTypeCode = "1001"` | Memang dipatok di sistem lama, di dua tempat yang menentukan perilaku: `@If(M_SURVEY_ID=="1001","Internal","Eksternal")` dan ketiga kueri `BrowseSurveyorType*` yang mematok `1002`/`1003`/`1004` |
| `TRFKOMITE = "1"` saat keputusan diambil | **DUGAAN yang dinyatakan.** Kolomnya hanya pernah DISALIN di export, tidak pernah dibandingkan dengan apa pun — nilai yang bermakna baginya tidak dapat diketahui. "1" dipilih karena itu yang dipakai kolom penanda biner lain di basis data yang sama |

### 20.9 Pertanyaan yang masih terbuka

| Pertanyaan | Pemilik |
|---|---|
| ~~Rule Approve/Reject Detail Surveyors — apakah masih ada di sistem Pega?~~ | ✅ **TERJAWAB 2026-09-20** — tidak ada rule tersendiri; satu activity, parameter `approval` |
| ~~Apakah email WAJIB?~~ | ✅ **TERJAWAB 2026-09-20** — wajib, pesannya `"Email harus diisi"` |
| Apa arti deret tipe `1021`–`1028` pada precondition langkah 8? | **Tim Pega** |
| ~~Apakah `V_D_SURVEYORS` membaca kolom atau JSON?~~ | ✅ **TERJAWAB 2026-09-20** — membaca kolom |
| Dari mana kolom 43 baris yang ada terisi, bila procedure hanya menulis JSON? | **Tim Pega / DBA** (langkah 0d, laporan) |
| Panjang kolom `D_SURVEYORS.NAME` yang sebenarnya | **DBA** (langkah 0c) |
| Nilai `TRFKOMITE` yang bermakna | **Tim Pega / Work Owner** |
| Kueri `EMAILKOMITE` mana yang dipakai jalur surveyor | **Tim Pega** |
| Kapan `F-3` siap, sehingga akun surveyor dapat benar-benar diterbitkan | **Work Owner** |
| Master cabang, agar `BRANCH` menjadi daftar pilihan dan bukan isian teks | **Work Owner** |
| Unggah lampiran (`DOCID`), menunggu modul `S-1 Dokumen` | **Work Owner** |

---

## 21. Modul Master Recovery (2026-09-20, sesi kedua belas)

Sumber: `Harness/MasterRecovery-Harness.xml`, kedua section-nya, lima activity, tiga rule
SQL, tiga stored procedure, dan **katalog Oracle portal ASM yang berjalan**.

### 21.1 Bentuk modul: entri saja

**Keputusan Work Owner 2026-09-20** atas tiga pertanyaan yang diajukan sebelum mulai:
kerjakan seperti yang ada di Pega · tiru apa adanya, entri saja · simpan saja, tanpa Ubah
maupun Hapus.

Dasarnya diperiksa, bukan diandaikan: **nol** kueri di seluruh export yang MEMBACA
`POOLDATA.MST_RECOVERY_ASM_PENJAMINAN`, dan `INSERTMASTERRECOVERYKLAIM.prc` hanya mengenal
INSERT.

**Akibatnya pada kode:** rute `GET /`, `PUT /{n}`, dan `DELETE /{n}` **tidak didaftarkan
sama sekali** — bukan didaftarkan lalu menolak. Rute yang tidak ada tidak dapat dipanggil
kode yang ditulis kemudian tanpa keputusan sadar.

### 21.2 Nilai uang: `int64` rupiah utuh

| Yang diperiksa | Hasil |
|---|---|
| Tipe kolom | `NUMBER` **tanpa presisi, tanpa skala** — menerima pecahan |
| Baris bernilai pecahan | **nol**, diperiksa dengan `NILAIKLAIM <> TRUNC(NILAIKLAIM) OR …` |
| `NLS_NUMERIC_CHARACTERS` | `".,"` |

Float dilarang untuk nilai uang tanpa perkecualian (`09-DATABASE-STRATEGY.md` §5), dan
kolomnya tidak memberi batas apa pun. Karena itu angkanya **ditetapkan**, bukan dibaca.

**Konsekuensi yang diterima sadar:** isian bernilai pecahan **ditolak**, sementara layar
Pega akan menerimanya. Menerima pecahan lalu memotongnya diam-diam jauh lebih berbahaya
daripada menolaknya dengan pesan. Bila sen kelak dibutuhkan, yang berubah adalah tipe
`Amount` menjadi titik-tetap dua desimal — kolomnya tidak perlu disentuh.

### 21.3 Aturan Sisa DIPERTAHANKAN apa adanya

Dua cabang, dan cabang kedua **mengabaikan pembayaran batch berjalan**:

| Prasyarat | Rumus |
|---|---|
| Pembayaran Sebelumnya = 0 | `Sisa = Nilai Klaim − Pembayaran` |
| Pembayaran Sebelumnya > 0 | `Sisa = Nilai Klaim − Pembayaran Sebelumnya` |

Ketiga baris produksi cocok tanpa perkecualian. **Tidak diperbaiki**: `P-5` menetapkan
perilaku dipertahankan lebih dulu, dan aturan ini tidak ada di daftar 13 perbaikan
eksplisit `D-49`. Bila ia memang cacat, perbaikannya menempuh keputusan tertulis.

Dikunci uji di tiga tempat — domain Go, layar, dan uji asap.

### 21.4 Empat cacat sistem lama yang TIDAK dibawa

| Cacat | Bukti | Penggantinya |
|---|---|---|
| Nomor batch dibaca saat layar dibuka, lalu **INSERT dilewati diam-diam** bila sudah dipakai | `INSERTMASTERRECOVERYKLAIM.prc` — `IF count_master<=0 THEN insert` tanpa cabang lain | Nomor diterbitkan **di dalam transaksi** dengan `LOCK TABLE … IN EXCLUSIVE MODE`; bentrok yang tersisa menjadi `409` yang terlihat |
| `ErrMsg` membawa **nomor VA** sekaligus pesan galat | `ADD_NEWMASTERVIRTUALACCOUNT.prc:18` | Nomor dan keadaan "dipakai ulang" dipisah; `dipakai_ulang` adalah penanda yang dapat dibaca program (`D-68`) |
| `COMMIT` di dalam procedure, `ROLLBACK` sesudahnya | `SET_ATTACHMENT_64BIT.prc` | Ketiga langkah penyimpanan lampiran dalam **satu** transaksi Go |
| Klausa WHERE dirangkai dari nama principal ketikan pengguna | `GetDataVAbyValidasiVA-SQL.xml` — `{ASIS:GeneratedVARecovery.NoteKomite}` | Parameter binding tanpa perkecualian |

### 21.5 Bukti bayar disimpan ke kolom BLOB yang sudah ada

`D-16` menetapkan dokumen disimpan lewat API storage internal, dan modul yang
membangunnya (`S-1`) belum ada. Menunggunya berarti layar ini tidak dapat menerima bukti
bayar sama sekali.

Pemeriksaan katalog menemukan `POOLDATA.DATA_ATTACHFILE` **berkolom BLOB `ATTACHFILE`**,
dan procedure lama pun menyisipkan ke tabel yang sama. Karena itu menyimpannya di sana
**memakai jalur yang memang sudah ada**, bukan jalan pintas.

`IMAGEID` sengaja dikosongkan — ia penunjuk ke penyimpanan luar, dan mengisinya dengan
nilai karangan akan membuat pembaca mana pun mengira berkasnya ada di sana. `IDPEGA` juga
dikosongkan: batch recovery bukan kasus Pega dan tidak punya kunci semacam itu.

**Penanda `DATAID` mengikuti bentuk yang terbukti**: dua digit tahun + sepuluh angka urut,
diverifikasi dari `DATAID` terbesar yang ada dan dari `ATTACHFILE_SEQ`. Urutan yang sama
dipakai, sehingga penomorannya melanjutkan deret Pega dan tidak pernah bertabrakan.

### 21.6 Pencarian polis: DB Link dipakai, dan utangnya dicatat

`MST_DET_SALES@ASMD` adalah salah satu dari 64 pemakaian DB Link yang `D-25` tetapkan
diganti API — "API Master Sales", yang **belum ada** (`R-03`).

Selama masa paralel link-nya masih hidup dan masih dibaca Pega, sehingga memakainya
**meneruskan** ketergantungan yang sudah ada alih-alih menambah yang baru. Ia diisolasi
pada **satu kueri bernama**, sehingga penggantinya kelak menyentuh satu tempat.

**Kegagalannya tidak membatalkan penyimpanan.** Sistem lama pun meneruskan dengan keempat
kolom kosong — `Insert_mst_recoveryKlaimASM` tidak punya satu pun prasyarat yang
menghentikan alur. Yang dicatat adalah uang yang sudah diterima; menolak menyimpan karena
basis data pihak lain sedang mati akan membuang seluruh isian petugas.

Keadaannya **dikembalikan ke layar** lewat `identitas_polis_terisi`, dikirim terpisah dan
bukan disimpulkan dari kolom kosong — kosong dapat berarti nomor polisnya memang tidak
diisi, dan itu hal yang berbeda.

### 21.7 Dua hal yang DISIMPULKAN

| Hal | Yang diketahui pasti | Yang disimpulkan |
|---|---|---|
| Kontrak penerbit VA | alamat (`GCNM_CONNECT_REST`, `TYPESERVICE=GENERATEDVA`), metode POST, dan properti yang diisi/dibaca activity | pemetaan nama field JSON. Rule `VirtualAccountClaimsPNC` **tidak ada di export** (`R-16`) |
| Daftar pilihan Tahun | kolom `TAHUN VARCHAR2(20)`; ketiga baris berisi `"2018"` | daftar nilainya. `GetListYear` **tidak ada di export** (`R-16`) |

Keduanya ditandai di kode pada tempat yang tepat. Pembacaan respons VA sengaja dibuat
**toleran** terhadap beberapa ejaan kunci: satu perbedaan ejaan tidak boleh membuat VA
yang sudah benar-benar terbit gagal tercatat.

Untuk Tahun, yang ditegakkan adalah **bentuknya** — empat angka dalam rentang wajar —
bukan daftar nilainya. Menebak isi daftarnya berarti mengarang aturan bisnis; menerima apa
saja berarti membiarkan `"20188"` tersimpan.

### 21.8 Penerbit VA tiruan bukan kenyamanan

Alamat yang terdaftar menunjuk layanan Pega yang **menerbitkan rekening virtual
sungguhan**. Menembaknya dari lingkungan pengembangan meninggalkan rekening nyata yang
tidak diminta siapa pun, pada sistem yang dipakai orang lain.

Pilihannya mengikuti **ada atau tidaknya koneksi basis data**, bukan `IDENTITAS_ADAPTER` —
karena alamatnya dibaca dari Oracle. Perbedaannya **diumumkan di log saat start**, bukan
dibiarkan senyap: layar yang tampak bekerja padahal nomornya karangan adalah kegagalan
yang tidak terlihat siapa pun sampai dana pertama dikirim ke nomor itu.

### 21.9 Yang ditambahkan ke berkas bersama, dan kenapa aditif

| Berkas | Tambahan | Alasan |
|---|---|---|
| `src/lib/money.ts` | **baru** | `08-TECHNICAL-STRATEGY.md` §3 aturan 4 menetapkan pemformatan angka dan mata uang melewati satu berkas bersama. Master Recovery modul pertama yang menangani uang, jadi berkasnya lahir bersamanya |
| `api/client.ts` | `uploadAPI`, `downloadAPI` | Aturan "tidak ada `fetch` di dalam komponen" akan runtuh pada modul pertama yang mengunggah berkas. **Aditif** — tidak ada perilaku lama yang diubah |
| `api/types.ts` | tipe + 10 kode galat | Cerminan dto Go |
| `App.tsx`, `registry.ts` | satu rute, satu baris peta | Menambah modul = menambah satu baris |
| `main.go`, `check.go` | perakitan dan pemeriksaan | idem |

Tidak ada berkas modul Login, Home, maupun Master Data yang sudah selesai yang disunting.

### 21.10 Pemeriksaan peran masih belum ada — dan di modul ini akibatnya lebih berat

Rutenya terlindungi sesi dan portal, tetapi **belum diperiksa perannya** (`TKT-F3-005`,
terhalang `TKT-F3-004`). Keadaan ini sama dengan seluruh rute lain hari ini.

Ia disebut khusus karena di modul ini akibatnya berbeda derajat: rutenya **menerbitkan
rekening virtual** dan **mencatat nilai uang**, sehingga tanpa pemeriksaan peran keduanya
terbuka bagi setiap pengguna yang berhasil masuk.

---

## 22. Modul Master Dominan Factor (2026-09-20, sesi ketiga belas)

Menggantikan `Harness/DetailDominanFactor-Harness.xml` atas tabel
`POOLDATA.M_DOMINAN_FACTOR`.

### 22.1 Modul ini SENGAJA berbeda dari dua master sebelumnya

Ini keputusan terpenting di sesi ini, dan yang paling mudah "diperbaiki" menjadi salah oleh
siapa pun yang menyamakannya dengan modul tetangganya.

| Hal | Master Status Klaim & Tipe Surveyors | **Master Dominan Factor** |
|---|---|---|
| Nama kosong | **ditolak** | **diterima** |
| Nama ganda | **ditolak** (indeks unik) | **diterima** (tanpa indeks) |
| Pembentuk kode | `id_site \|\| lpad(urutan,3,'0')` | **`max(ID)+1`**, tanpa padding |
| Sumber nomor | `M_SITE_DATABASE` + sequence | tabel itu sendiri |
| Migrasi basis data | wajib / opsional | **tidak ada sama sekali** |

Ketiga perbedaan pertama adalah **keputusan Work Owner 2026-09-20**, diambil setelah akibatnya
dinyatakan di muka. Dua yang terakhir mengikuti dari sana.

Yang mengikat di kode: uji `TestEmptyNameAccepted` dan
`TestCreateAcceptsEmptyAndDuplicateNames` akan **gagal** bila seseorang menambahkan `.min(1)`
pada skema Zod atau pemeriksaan keunikan di usecase. Bila uji itu gagal, yang perlu dibaca lebih
dulu adalah keputusan ini — bukan ujinya.

### 22.2 `max+1` ditiru, bukan diganti — tetapi cacat balapannya ditutup

`Database/PEGA_M_DOMINAN_FACTOR.prc:11`:

```sql
select nvl(max(to_number(ID)),0)+1 into id_site from M_DOMINAN_FACTOR;
```

Menggantinya dengan sequence akan mengubah **bentuk ID yang diterbitkan**, dan
`T_CLAIM_DOMINANFACTOR` menyimpan nilainya apa adanya — deret lama dan baru tidak akan sinambung.
Jadi perhitungannya ditiru persis, termasuk hasil `"1"` saat tabel kosong dan ketiadaan nol di
depan.

Yang **tidak** ditiru adalah ketiadaan kuncinya. Prosedur lama membaca nilai tertinggi tanpa
kunci apa pun, sehingga dua penyimpanan bersamaan menerbitkan nomor yang sama. Di sini ketiga
langkahnya — kunci, hitung, sisip — berada dalam **satu transaksi Go** dengan
`SELECT ID ... FOR UPDATE`, sejalan `D-68` yang memindahkan kepemilikan transaksi ke Go.

**Batasnya dinyatakan, bukan disembunyikan:** bila tabel kosong tidak ada baris yang dapat
dikunci, sehingga dua penyisipan pertama yang benar-benar bersamaan masih dapat kembar. Keadaan
itu hanya mungkin sekali seumur hidup tabel, dan penjagaan sisanya ada di pemeriksaan ID
terhadap daftar yang sudah terkunci.

### 22.3 Nama constraint tidak diketahui — disiasati, bukan ditebak

Master Status Klaim menerjemahkan galat bentrok dengan mencocokkan **nama constraint**
(`M_STS_CLAIM_PK`), yang dibaca dari `ALL_CONSTRAINTS` pada 2026-09-17. Untuk
`M_DOMINAN_FACTOR`, nama itu **tidak diketahui** — DDL-nya tidak ada di export (`R-08`).

Menebaknya dari pola penamaan adalah persis kesalahan yang pernah terjadi di Master Status
Klaim, ketika `PK_M_STS_CLAIM` ditulis lebih dulu dan ternyata terbalik. Jadi pendekatannya
diganti: karena seluruh ID **sudah dibaca dan dikunci** di dalam transaksi yang sama,
pemeriksaan bentrok dilakukan di Go atas daftar itu. Ia tidak bergantung pada nama objek basis
data mana pun.

### 22.4 Pengurutan dikerjakan di Go, dan itu perbaikan yang disengaja

Kueri Pega **tidak punya `ORDER BY` sama sekali**, sehingga urutan barisnya ditentukan basis data
dan dapat berubah sewaktu-waktu. Menambahkan `ORDER BY ID` akan mengurutkan sebagai **teks** —
dan karena ID tidak bernol di depan, `10` akan mendahului `9`.

Mengurutkannya secara numerik di SQL menuntut `TO_NUMBER(ID)`, yang terikat dialek Oracle dan
dilarang `09-DATABASE-STRATEGY.md` §4. Jadi urutannya dikerjakan di
`masterdominanfactor.SortByID` — **satu fungsi di lapisan domain**, dipakai adapter SQL maupun
adapter memori, supaya keduanya tidak dapat berbeda pendapat tentang urutan yang benar.

Ini mengubah susunan di layar, **tidak mengubah satu pun nilai**.

Baris ber-ID bukan bilangan ditempatkan di **belakang**. Menaruhnya di depan akan menyembunyikan
baris normal di bawah keanehan warisan.

### 22.5 `to_number` yang gagal total diganti pelewatan yang tercatat

Procedure lama memakai `to_number(ID)` saat membentuk nomor berikutnya. Satu baris warisan
ber-ID bukan bilangan akan membuatnya gagal dengan ORA-01722 dan **membatalkan seluruh
penambahan**, tanpa pesan yang menyebut baris mana penyebabnya.

`NumericID` melewati baris seperti itu. Perbedaan yang disengaja, dan arahnya menguntungkan:
satu baris cacat tidak lagi membuat penambahan faktor baru mustahil.

Ia tidak dibiarkan senyap: mode periksa melaporkannya sebagai `[WASPADA]`, karena **Pega masih
dapat menulis tabel ini** selama masa paralel dan di sana keadaan itu tetap fatal.

### 22.6 Tidak ada Hapus, dan ketiadaannya diuji

Tiga bukti, bukan satu: layar Pega tidak punya tombolnya, procedure-nya tidak punya cabangnya,
dan menghapus satu baris akan membuat setiap klaim yang menyimpan ID itu di
`T_CLAIM_DOMINANFACTOR` kehilangan artinya — sehingga laporan Outstanding per Cabang menampilkan
faktor yang hilang tanpa penjelasan (`ADR-0012`).

Ketiadaannya **diuji di tiga lapisan**, bukan sekadar dicatat di komentar:
`TestNoQueryDeletes` (SQL), `TestNoDeleteRoute` (HTTP, 405), dan
`tidak menyediakan tombol hapus` (layar).

### 22.7 Nama field JSON `nama`, label layar "Keterangan" — dan keduanya benar

| Lapisan | Dipakai | Alasan |
|---|---|---|
| Kolom basis data | `NAME` | milik sistem lama |
| Field domain Go | `Name` | penamaan kode berbahasa Inggris (`D-80`) |
| Field JSON | `nama` | **kontrak** — mencerminkan isinya, bukan label layarnya (`D-80`) |
| Label di layar | **"Keterangan"** | teks yang dilihat pengguna mengikuti layar Pega apa adanya (`D-13`) |

Perbedaan itu dicatat di komentar dto supaya tidak terbaca sebagai ketidakkonsistenan. Ia
mengikuti dua keputusan yang memang berbeda peruntukan.

Nama modul mengikuti `D-81` — `masterdominanfactor` di backend, `master-dominan-factor` di
frontend, ruas URL `master/dominan-factor`: nama modul yang dipakai Work Owner dan yang tertulis
di butir MENU_ID 18, bukan terjemahan "faktor-dominan".

### 22.8 Akibat pada laporan disebut di layar, bukan hanya di kode

`GetDataOutstandingperCabangExport` merangkai nama faktor dengan `LISTAGG(m.name, ', ')`.
Mengubah keterangan di layar master **langsung mengubah isi laporan yang dibaca manajemen**.

Itu dinyatakan di paragraf pembuka layar, bukan hanya di komentar kode — pengguna yang
mengubahnya perlu tahu jangkauan akibatnya sebelum menekan Simpan, bukan sesudah.

### 22.9 Peringatan menggantikan pemeriksaan yang sengaja tidak ada

Karena keterangan ganda diterima, **pengguna tidak akan pernah ditolak sistem**. Jadi
satu-satunya cara ia tahu adalah diberi tahu: form tambah memuat peringatan bahwa keterangan
yang sama boleh terdaftar lebih dari satu kali, dan menyarankan memeriksa daftar lebih dulu.

Alasan yang sama membuat form digambar sebagai panel **di atas tabel**, bukan dialog melayang —
daftarnya harus tetap terlihat justru saat pengguna sedang mengetik.

Baris ber-keterangan kosong ditandai `(tanpa keterangan)` di tabel, bukan dibiarkan sebagai sel
kosong yang terlihat seperti tabel rusak. Keduanya konsekuensi langsung dari keputusan 22.1.

### 22.10 Contoh data ditandai sebagai karangan

Isi `M_DOMINAN_FACTOR` yang sebenarnya **belum pernah diterima** — tidak ada artefak setara
`Database/v_sts_claim.csv` untuk tabel ini.

`SampleList` karena itu diberi peringatan besar di kepalanya bahwa isinya dikarang, lengkap
dengan kueri yang perlu diminta ke DBA. Pilihan membiarkannya kosong ditolak: daftar kosong
membuat pengurutan numerik, tombol Ubah, dan keadaan tabel berisi tidak dapat dicoba sama sekali
tanpa Oracle. Yang dikarang hanyalah **teksnya**; perilaku yang diujinya nyata.

Sepuluh baris dipilih dengan sengaja supaya ID mencapai dua digit — itulah yang membuktikan
pengurutan numerik benar-benar bekerja, dan bukan kebetulan karena semua ID masih satu digit.

### 22.11 Yang ditambahkan ke berkas bersama, dan kenapa aditif

| Berkas | Tambahan | Alasan |
|---|---|---|
| `api/types.ts` | 3 tipe + 2 kode galat | Cerminan dto Go |
| `App.tsx`, `registry.ts` | satu rute, satu baris peta | Menambah modul = menambah satu baris |
| `main.go`, `check.go` | perakitan dan pemeriksaan | idem |

Tidak ada berkas modul Login, Home, maupun Master Data yang sudah selesai yang disunting.
Tidak ada berkas migrasi yang ditambahkan — lihat 22.1.

### 22.12 Pemeriksaan peran masih belum ada

Rutenya terlindungi sesi dan portal, tetapi **belum diperiksa perannya** (`TKT-F3-005`, terhalang
`TKT-F3-004`). Keadaan ini sama dengan seluruh rute lain hari ini.

Di modul ini akibatnya lebih ringan daripada Master Recovery — tidak ada uang maupun rekening
yang tersentuh — tetapi tidak nol: keterangan yang diubah ikut terbaca laporan Outstanding per
Cabang.

## 23. Modul Master Masking (2026-09-20, sesi keempat belas)

Butir menu `MENU_ID 17`, harness `MasterProteksiVisibilityData`, tabel
`POOLDATA.MST_PROTEKSI_DATA_PNC`.

Modul ini berbeda derajat dari master lain yang sudah dibangun: isinya bukan data acuan,
melainkan **kewenangan melihat data pribadi nasabah**. Setiap keputusan di bawah diambil
dengan menimbang satu hal yang sama — apa akibatnya bila ia salah, dan apakah salahnya
akan terlihat.

### 23.1 Kunci alaminya CABANG + LOGIN, bukan ID_MST

**Keputusan.** Keunikan ditegakkan atas pasangan cabang+pengguna, bukan atas ID.

**Dasarnya.** `Database/UPDATE_LOG_PROTEKSI.prc` menolak penyisipan bila pasangannya sudah
ada (`:29-31`, pesan "LOGIN … SUDAH ADA") dan mencocokkan pasangan yang sama saat update
(`:74-76`). Basis data membenarkannya: 25 baris menghasilkan 25 pasangan unik.

**Akibat bila diabaikan.** Dua baris untuk orang yang sama di satu cabang berarti dua
kewenangan yang keduanya berlaku — dan yang satu dapat luput saat dicabut.

**Yang harus diketahui.** Tabel ini **tidak punya indeks unik**; ALL_INDEXES hanya memuat
satu indeks NONUNIQUE. Artinya pemeriksaan di usecase adalah **satu-satunya** yang menolak
pasangan ganda, dan dua permintaan yang tiba benar-benar bersamaan dapat sama-sama lolos.
Sistem lama pun demikian — penegakannya sepenuhnya bersandar pada procedure. Indeks unik
diusulkan ke DBA; sampai ada, keadaan ini dicatat terbuka, bukan disembunyikan.

### 23.2 Status aktif dipisahkan dari form

**Keputusan.** Status punya jalur tersendiri (`PUT /{id}/status`), dan `SaveRequest`
**tidak menerima** field `aktif`. Server menolak badan permintaan yang memuatnya.

**Dasarnya.** Ini bukan kerapian REST melainkan pencegahan satu kelas kesalahan. Bila
status ikut di dalam form, menyimpan perubahan isian pada baris nonaktif akan
**mengembalikan kewenangan membuka data pribadi** — tanpa pesan galat, tanpa tanda di
layar, dan tanpa ada yang bermaksud demikian. Kegagalan seperti itu dapat berjalan lama
tanpa ada yang menyadarinya.

**Penegakannya berlapis tiga:** DTO tidak punya fieldnya · usecase menimpa `Active` dengan
nilai tersimpan · `masking_update` tidak menyentuh kolom `STS_AKTF`. Tiga uji menjaganya
(domain, usecase, layar), ditambah satu perkara uji asap.

### 23.3 Tombol DELETE tidak pernah menghapus, dan namanya diubah

**Keputusan.** Tidak ada rute `DELETE`. Penonaktifan memakai `PUT /{id}/status`, dan
tombolnya di layar disebut **Nonaktifkan / Aktifkan**, bukan Hapus.

**Dasarnya.** `RDB List/DeleteMstProteksi_SQL-SQL.xml` hanya menjalankan
`UPDATE … SET STS_AKTF`, dan `Activity/DeleteMasking-Act.xml` mengisinya `'TIDAK AKTIF'`.
Tidak ada satu pun pernyataan `DELETE` terhadap tabel ini di seluruh export. `D-66`
menetapkan hal yang sama untuk sistem baru.

**Kenapa namanya ikut diubah.** Nama "Hapus" untuk tindakan yang tidak menghapus membuat
petugas mengira datanya hilang, lalu menambahkannya lagi dari awal — dan barisnya menjadi
dua. Ini satu-satunya tempat modul ini menyimpang dari `D-13` (tampilan meniru Pega), dan
penyimpangannya disengaja karena nama lamanya salah menggambarkan apa yang terjadi.

**Akibat yang diterima.** Sepuluh dari 25 baris produksi berstatus tidak aktif, dan
masing-masing adalah catatan bahwa seseorang PERNAH diberi kewenangan. Daftar karena itu
menampilkan baris nonaktif **secara baku** — menyembunyikannya akan membuat 40% isinya
lenyap dari layar tanpa penjelasan, dan tidak ada cara mengaktifkannya kembali.

### 23.4 Kolom PASSWORD ditulis literal, dan tidak pernah dibaca

**Keputusan Work Owner (2026-09-20).** Kolomnya diisi persis seperti sistem lama.

**Yang ditemukan sebelum keputusan diambil.** Work Owner semula memilih "dibawa apa adanya"
ketika kami sama-sama mengira kolom itu berisi kata sandi. Penelusuran membuktikan
sebaliknya, dan temuannya dilaporkan kembali sebelum kode ditulis:
`Activity/InsermaskingDataKlaimPnc_-Act.xml` menetapkan
`InputData.ResponseCode := "saya"` — literal tetap; tidak ada isian PASSWORD di layar Pega;
dan seluruh logika verifikasinya dikomentari di `UPDATE_LOG_PROTEKSI.prc:114-122`. Basis
data membenarkannya: 25 dari 25 baris berisi nilai sepanjang tepat 4 karakter.

Setelah diberi tahu, Work Owner **tetap memilih menulis literal yang sama**, supaya baris
lama dan baru seragam. Keputusan itu dihormati dan dijalankan.

**Dua pagar yang dipasang.** Nilainya menjadi konstanta bernama `legacyPassword` yang
membawa penjelasan lengkap — supaya ia tidak pernah lagi terbaca sebagai rahasia, dan
supaya menghapusnya kelak cukup menyentuh satu tempat. Dan kolomnya **tidak pernah dibaca,
tidak pernah masuk DTO, tidak pernah dikirim ke peramban**. Satu uji menjaganya
(`TestPasswordNeverLeavesServer`), diperkuat uji asap yang memeriksa nol kemunculan kata
"password" dan "sandi" di respons daftar.

### 23.5 ID tetap dibentuk MAX+1

**Keputusan Work Owner.** Meniru `Database/UPDATE_LOG_PROTEKSI.prc:33-34`, bukan diganti
sequence.

**Dasarnya.** Penomoran tetap bersambung dengan yang dibuat Pega selama masa paralel
(`P-5`). Sequence akan memutus kesinambungan itu dan menuntut migrasi DBA.

**Bahaya yang disadari dan diterima.** Dua penyimpanan bersamaan dapat membaca nilai yang
sama, dan tidak ada indeks unik yang menolaknya. Penanganannya: nomor dibaca **di dalam
transaksi yang sama** dengan penyisipannya, dan bentrokan dicoba ulang sampai tiga kali.
Itu mempersempit peluangnya, **tidak menutupnya** — dan itu dinyatakan apa adanya di kode
maupun di sini.

### 23.6 MODUL dan SUB MODUL diketik bebas

**Keputusan Work Owner.** "Seperti aplikasi Pega saja".

**Dasarnya.** Daftar pilihannya di Pega dipasok rule `MODULKLAIMMASKING`, dan rule itu
**hilang dari export** (`R-16`) — daftarnya tidak dapat direproduksi tanpa mengarangnya.

**Kenapa tidak dibatasi ke nilai yang ada di data.** Satu-satunya nilai MODUL di produksi
adalah `PNCSearchKlaim`. Membatasi isian ke nilai itu akan menolak nilai sah yang belum
pernah dipakai — dan menolak data yang benar jauh lebih merugikan daripada menerima salah
ketik yang dapat disunting kembali.

**Yang dilakukan sebagai gantinya.** Form mengisi MODUL dengan `PNCSearchKlaim` sebagai
**saran**, bukan batasan. Bentuk daftar SUB MODUL — dipisah koma dengan koma di ujung —
dibawa apa adanya, karena Pega masih membaca tabel yang sama selama masa paralel (`D-21`).

### 23.7 Baris produksi yang cacat ditampilkan apa adanya

**Keputusan Work Owner.** Tidak diperbaiki otomatis, tidak disembunyikan.

**Dasarnya.** Satu baris memuat `Penerima Pembayaran KlaimRegistrasi,Dokumen,` — kurang
koma dan kurang dua huruf. Merapikannya saat ditampilkan akan membuat layar terlihat benar
sementara datanya tetap salah, dan tidak ada yang tahu ia perlu diperbaiki. Perbaikannya
menempuh jalur DBA.

Hal yang sama berlaku untuk cabang yang tidak dikenal: `LEFT JOIN`, bukan `INNER`, dan
barisnya ditandai di layar. Tidak ada yang menggantung hari ini, tetapi menyembunyikannya
bila kelak terjadi berarti sebuah kewenangan hidup tanpa pernah terbaca benar di layar
mana pun.

### 23.8 Daftar cabang dibaca di modul ini, bukan di modul bersama

**Keputusan.** `GET /api/master/masking/cabang` membaca `POOLDATA.BRANCH` — hanya membaca,
tidak pernah menulis.

**Kenapa tidak dijadikan modul bersama.** Modul inilah satu-satunya yang memperlakukan kode
cabang sebagai **kunci asing sungguhan**; Master Surveyors menyimpan nama cabang sebagai
teks biasa. Kebutuhannya berbeda, dan menyatukannya sekarang akan memaksa dua hal yang
tidak sama menjadi satu. Bila modul ketiga membutuhkannya, barulah ia naik.

**Kenapa disaring, bukan dimuat seluruhnya.** Tabelnya 803 baris. Mengirim semuanya ke
peramban setiap kali form dibuka membebani jaringan untuk daftar yang hampir seluruhnya
tidak akan dilihat — layar lama pun memakai autocomplete.

### 23.9 Transaksi dimiliki Go, procedure tidak dipanggil

**Keputusan.** `POOLDATA.Update_Log_Proteksi` tidak dipanggil sama sekali; pernyataan
SQL-nya ditulis langsung (`D-02`), dan transaksinya dimiliki Go (`D-68`).

**Dasarnya, dan ia contoh yang paling jelas dari `D-68`.** Procedure itu melakukan `COMMIT`
sendiri di **tiga cabang** (`:57`, `:99`, `:161`) lalu menaruh `ROLLBACK` sesudahnya,
sehingga kegagalan di tengah meninggalkan data setengah jalan. Kontrak galatnya pun berbasis
teks: parameter keluaran `MSG` berisi `'1'` bila berhasil dan kalimat berbahasa Indonesia
bila gagal — pemanggil tidak dapat membedakan keduanya tanpa membaca isi teksnya.

**Satu perbedaan yang disengaja.** `TANGGALINPUT` diisi dari aplikasi, bukan `SYSDATE`.
Waktu datang dari satu tempat (seam Clock, `F-5`), bukan dari jam basis data yang tidak
dapat dikendalikan maupun diuji — dan itu yang menutup `R-12`.

### 23.10 Pencarian memakai parameter binding

**Keputusan.** Seluruh nilai lewat bind; tidak ada perangkaian ke teks SQL.

**Dasarnya.** `Activity/SearchDataMasking-Act.xml` merangkai kata kunci langsung ke
pernyataannya lewat `{ASIS:InputSearch.CARI1}`, dan `CekmaskingDataPerLoginUserKlaim`
merangkai `OperatorID.pyUserIdentifier`. Keduanya persis celah yang
`docs/Steering/07` §4.3 tutup tanpa perkecualian.

**Satu keputusan teknis yang perlu dicatat.** `masking_list` mengikat kata kunci **dua
kali** sebagai `:2` dan `:3` alih-alih mengulang satu penanda, supaya kueri tidak
bergantung pada perilaku driver terhadap penanda berulang. Ada uji yang menjaga keputusan
itu tidak batal saat kueri disunting kelak (`TestBindMarkersAppearOnce`).

**Tipe pencarian yang tidak dikenal DITOLAK**, tidak diam-diam diperlakukan sebagai "tanpa
penyaring". Mengabaikannya akan menampilkan seluruh baris kepada pengguna yang mengira ia
sedang menyaring — pada layar ini, itu berarti membuka seluruh peta kewenangan tanpa
diminta.

### 23.11 Nilai kolom diterjemahkan dengan arah aman

**Keputusan.** `STS_KTP`/`STS_EMAIL`/`STS_NOTELP` bernilai `'Ya'`/`'Tidak'`; hanya `'Ya'`
yang berarti boleh. `STS_AKTF` bernilai `'AKTIF'`/`'TIDAK AKTIF'`; hanya `'AKTIF'` yang
berarti berlaku.

**Dasarnya.** Dibaca langsung dari 25 baris produksi. Dugaan `'1'`/`'0'` yang sempat muncul
dari blok yang dikomentari di procedure **terbukti salah** — dan itu alasan konkret
mengapa nilai kolom tidak boleh disimpulkan dari kode yang tidak berjalan.

**Arah penerjemahannya disengaja dan tidak boleh dibalik.** Nilai yang tidak dikenal —
kosong, NULL, atau yang belum pernah terlihat — diperlakukan sebagai **tidak boleh**.
Akibatnya data nasabah tetap tersamar. Aman salah ke arah menyamarkan, tidak pernah ke
arah membuka.

### 23.12 Yang diserahkan, dan apa akibatnya bila tidak ditindaklanjuti

| Hal | Pemilik | Akibat bila dibiarkan |
|---|---|---|
| Rule `MODULKLAIMMASKING` hilang (`R-16`) | Tim Pega | MODUL dan SUB MODUL tetap teks bebas; salah ketik membuat masking diam-diam tidak berlaku |
| Indeks unik `CABANG`+`LOGIN` | DBA | Dua permintaan bersamaan dapat membuat kewenangan ganda yang satu di antaranya luput saat dicabut |
| Satu baris sub modul cacat | DBA | Baris itu tetap tampil apa adanya |
| Pemeriksaan peran `TKT-F3-005` | Work Owner | **Siapa pun yang dapat masuk dapat memberi dirinya sendiri kewenangan membuka data pribadi nasabah** |
| Kredensial lima portal | Tim Infra | Modul hanya dapat dipakai di portal ASM |

Butir keempat adalah yang paling berat di seluruh modul ini, dan ia **bukan** akibat
rancangan modul — ia keadaan yang sama dengan seluruh rute lain yang sudah ada hari ini.
Yang membedakannya adalah taruhannya.

---

## 24. Modul Master Penyebab Kerugian (2026-09-20, sesi kelima belas)

`MENU_ID 20` · harness `CauseOfLossInbox` · tabel `POOLDATA.M_CAUSE_OF_LOSS`.

### 24.1 Lingkup dibatasi pada TINGKAT GOLONGAN

**Keputusan.** Modul ini hanya mengelola `M_CAUSE_OF_LOSS`. Rinciannya
(`D_CAUSE_OF_LOSS`, harness `DetailCauseOfLoss`, `MENU_ID 38`) menjadi modul tersendiri
yang belum dibangun.

**Dasarnya.** Work Owner menjawab *"seperti aplikasi PEGA saja"*, dan di Pega keduanya
memang terpisah: butir menu berbeda, harness berbeda, Report Definition berbeda
(`BrowseVMCauseOfLoss_RD` versus `SelectVDCauseOfLoss_RD`), procedure berbeda.

**Yang ini tutup.** Godaan menumpangkan rincian sebagai sub-sumber daya
(`/penyebab-kerugian/{id}/rincian`). Itu akan menyatukan dua layar yang di sistem lama
terpisah, dan menyulitkan penegakan izin per menu (`D-59`) karena keduanya butir menu
yang berbeda. Ketiadaannya **diuji**, bukan sekadar dicatat.

### 24.2 Isi pindah dari dokumen JSON ke kolom — dan `P-1` belum terpenuhi utuh

**Keputusan.** Go membaca dan menulis kolom `M_COL_ID`, `OLD_M_COL_ID`, dan `COL_DESC`.
`JSON_DATA` **tidak pernah ditulis lagi**. Perpindahan isinya dikerjakan migrasi 0005,
mengikuti pola migrasi 0002 pada `M_STS_CLAIM`.

**Dasarnya.** Work Owner menjawab *"menggunakan tabel yang di baca pada PEGA, tidak pakai
json lagi"*, sejalan dengan `D-02` (procedure tidak dipanggil) dan `D-68` (kontrak galat
`ErrMsg` tidak dibawa — pada jalur BERHASIL ia berisi kalimat, bukan kekosongan).

**Perbedaan penting dari migrasi 0002, dan konsekuensinya diterima sadar.** Pada
`M_STS_CLAIM`, layar yang dipindahkan adalah satu-satunya penulis tabelnya. **Di sini
tidak:**

| Layar | Menu | Status |
|---|---|---|
| `CauseOfLossInbox` | 20 | dipindahkan sesi ini |
| `CauseOfLossInboxSimasOnline` | 21 | **masih hidup di Pega** |

Keduanya menulis lewat `RDB List/UpdateMCauseOfLoss-SQL.xml`. Selama layar kedua dipakai,
baris yang ditulisnya hanya mengisi `JSON_DATA` dan akan tampil **tanpa keterangan** di 19
rule pembaca — tanpa satu pun galat.

Keberatan ini disampaikan sebelum pekerjaan dimulai dan arahnya tetap dipilih. Yang
dikerjakan sebagai gantinya adalah membuat celahnya **terlihat**, bukan menambalnya
diam-diam:

- Migrasi 0005 menaruh peringatan di kepala berkas dan **melarang langkah 3 dijalankan**
  sebelum Work Owner memilih satu dari tiga cara menutup celahnya.
- Kueri `cause_of_loss_count_pending_json` menghitung barisnya.
- Mode periksa melaporkan angkanya setiap kali dijalankan, beserta dua sebab yang mungkin.

**Yang ini tutup.** Menulis `JSON_DATA` sekaligus kolom "supaya aman". Itu akan membuat
dua salinan yang dapat berbeda, dan mengulang persis utang teknis §3.2 yang menjadi alasan
migrasi ini ada.

### 24.3 Tanpa validasi — kosong dan ganda diterima

**Keputusan.** Keterangan kosong dan keterangan ganda **diterima**. Tidak ada indeks unik
yang dibuat.

**Dasarnya.** Work Owner menjawab *"sesuai PEGA saja"*, dan Pega memang tidak memvalidasi
apa pun di layar ini: `Section/BrowseCauseOfLoss-Section.xml` tidak memasang `pyRequired`
maupun `pyMaxLength`, dan `Database/PEGA_M_CAUSE_OF_LOSS.prc` tidak memeriksa apa pun
sebelum menyisipkan. `P-5` menetapkan perilaku dipertahankan lebih dulu.

**Ini berbeda dari Master Status Klaim dan Master Tipe Surveyors, yang justru menolak
keduanya** (keputusan Work Owner 2026-09-17 dan 2026-09-19). Kedua keputusan diambil untuk
layar yang berbeda dan keduanya berlaku. Perbedaannya dicatat di kode supaya tidak
"diseragamkan" tanpa keputusan baru.

**Satu hal yang tetap ada, dan ia bukan aturan bisnis.** Batas 100 karakter. Ia penjaga
terhadap penolakan basis data: tanpa batas, isian yang melebihi lebar kolom sampai ke
pengguna sebagai galat 500 beserta nomor galat Oracle. Lebar `COL_DESC` yang sebenarnya
belum diketahui (`R-08`); bila kelak lebih sempit, yang berubah cukup satu konstanta di
tiap sisi.

**Akibat yang diterima sadar, dan disampaikan di layar.** Karena ganda diterima, pengguna
tidak akan pernah ditolak sistem — jadi satu-satunya cara ia tahu adalah **diberi tahu**.
Form penambahan memuat peringatan yang menggantikan pemeriksaan yang sengaja tidak ada.

### 24.4 ID dibentuk seperti procedure lama, batasnya dibiarkan terlihat

**Keputusan.** `M_COL_ID` dibentuk kode situs ditambah tiga digit ber-nol di depan, dengan
kode situs dari `M_SITE_DATABASE WHERE CURRENT_SITE = '1'` dan urutan dari
`M_CAUSE_SEQ.NEXTVAL` — persis `Database/PEGA_M_CAUSE_OF_LOSS.prc`.

**Dasarnya.** ID yang diterbitkan aplikasi ini harus MELANJUTKAN deret yang sudah ada.
Deret baru akan bertabrakan dengan ID yang pernah diterbitkan Pega.

**Batas yang dibiarkan apa adanya.** Bilangan di atas 999 dikembalikan apa adanya, sama
seperti `LPAD` Oracle, sehingga ID ke-1000 menjadi lima karakter. Bila kolomnya memang
`varchar2(4)` seperti yang dideklarasikan procedure, penyisipannya akan **ditolak basis
data**. Ia **tidak dipotong** menjadi tiga digit: itu akan menghasilkan ID GANDA, yang
jauh lebih buruk daripada penyisipan yang gagal dengan pesan jelas.

**Perangkaian dan pemformatannya dikerjakan di Go**, bukan `LPAD`/`TO_CHAR` di SQL —
keduanya mengikat kueri pada dialek Oracle (`D-20`). `FROM DUAL` yang tidak terhindarkan
diisolasi di satu kueri, dan isolasi itu **diuji**.

### 24.5 Pengurutan teks, bukan numerik — kebalikan dari Master Dominan Factor

**Keputusan.** Daftar diurutkan menurut ID secara teks, di Go.

**Dasarnya.** Bentuk ID-nya tetap — kode situs ditambah tiga digit ber-nol di depan —
sehingga pengurutan teks dan numerik menghasilkan urutan yang sama. Ini kebalikan dari
Master Dominan Factor, yang ID-nya `max+1` tanpa nol di depan sehingga `10` mendahului `9`
bila diurutkan sebagai teks.

Pengurutannya tetap di Go, bukan `ORDER BY`, karena pengurutan kolom CHAR berpadding
berperilaku berbeda antar-basis-data.

### 24.6 ID lama ditampilkan meski Pega tidak menampilkannya

**Keputusan.** Kolom `OLD_M_COL_ID` dikirim API dan ditampilkan di tabel serta di form
pengubahan — sebagai keterangan, bukan isian.

**Dasarnya.** Ia satu-satunya yang menjelaskan data historis yang masih memakai penomoran
lama. Keputusan yang sama sudah diambil di Master Status Klaim untuk `OLD_LSC_ID`.
Grid Pega tidak menampilkannya, tetapi Report Definition-nya membawanya — jadi ia bukan
data yang sengaja disembunyikan, hanya tidak ditampilkan.

Ia **tidak pernah dapat diisi dari luar**: tidak diterima di badan permintaan, dan tidak
diisi kueri penyisipan. Ketiadaannya di jalur tulis **diuji**.

### 24.7 Seluruh teks layar diambil apa adanya dari Pega

**Keputusan.** Setiap teks yang dilihat pengguna disalin dari export, bukan dikarang:

| Yang tampil | Sumbernya |
|---|---|
| Judul **Master Penyebab Kerugian** | `pyValue` pada `Harness/CauseOfLossInbox-Harness.xml` |
| Kolom **ID** | `pyValue` pada sel ber-`pyCellHeader=true`, `pyLabelFor .M_COL_ID` |
| Kolom **Deskripsi Kerugian** | idem, `pyLabelFor .COL_DESC` |
| Isian **Deskripsi Kerugian** | `pyLabelPreview` pada form |
| Tombol **Tambah** dan **Refresh** | `pyButtonLabel` pada harness |
| Aksi **Ubah** | `pxLink` berlabel `Ubah` pada section |
| Judul form **Memperbaharui Data** | `pyTitle` pada section |

**Ini mengoreksi keputusan yang sempat diambil sebelumnya.** Pembacaan pertama menyimpulkan
grid Pega tidak memasang caption pada kolom deskripsi, sehingga judulnya "jatuh ke nama
properti `COL_DESC`" dan layar baru menuliskannya **"Keterangan"**.

Kesimpulan itu **salah**: captionnya ada, hanya tersimpan sebagai `pyValue` pada sel
berpenanda `pyCellHeader=true` — bukan sebagai `pyCaption` yang dicari pembacaan pertama.
Membaca section dalam urutan dokumen memunculkannya. `D-13` karena itu berlaku penuh di
sini, tanpa pengecualian yang sempat dibuat.

**Field JSON ikut berubah menjadi `deskripsi`**, bukan `keterangan`. Membiarkannya berbeda
dari label layar akan mengulang persis utang teknis §4.2 — nama yang tidak mencerminkan
isinya.

### 24.8 Kolom ID lama dikirim API, tetapi tidak ditampilkan

**Keputusan.** `OLD_M_COL_ID` tetap ada di kontrak API, tetapi **tidak tampil di layar**.

**Dasarnya.** Dua artefak Pega yang berbeda diikuti di tempatnya masing-masing:

| Lapisan | Mengikuti | Memuat `OLD_M_COL_ID`? |
|---|---|---|
| Data | `Report Definition/BrowseVMCauseOfLoss_RD` | **ya** |
| Layar | `Section/BrowseCauseOfLoss` | **tidak** — grid hanya dua kolom |

**Ini juga mengoreksi keputusan sebelumnya**, yang menampilkan kolom itu dengan alasan "ia
menjelaskan data historis" — mengikuti preseden Master Status Klaim. Alasannya tetap benar,
tetapi Work Owner meminta layar mengikuti Pega saja, dan grid Pega tidak memuatnya.

Ia tidak dibuang dari kontrak karena lapisan datanya memang memuatnya, dan modul Rincian
Penyebab Kerugian (`MENU_ID 38`) kelak tidak perlu mengubah kontrak ini untuk menautkan
data historisnya. Ketiadaannya di layar **dijaga uji**, supaya menampilkannya kembali
menjadi keputusan — bukan perbaikan yang menyelinap.

### 24.9 Satu isian Pega yang sengaja tidak dibawa

**Keputusan.** Form hanya memuat SATU isian yang dapat diketik.

**Dasarnya.** Form Pega sebenarnya memuat **dua** isian berlabel "Deskripsi Kerugian":

| Terikat | `pyAutomationID` | Diisi saat mengubah? | Dapat dibaca kembali? |
|---|---|---|---|
| `TempCauseOfLoss.COL_DESC` | `201703231439450790383511` | ya | ya, lewat view |
| `TempCauseOfLoss.Description` | **sama persis** | **tidak** | **tidak** |

`pyAutomationID` yang sama menandakan salin-tempel. Yang kedua **mati**:
`Activity/SetCauseOfLossValue_act-Act.xml` mengisi form dengan `M_COL_ID`, `COL_DESC`, dan
`pyNote` saja, dan `V_M_CAUSE_OF_LOSS` tidak punya kolom untuk membacanya kembali — apa pun
yang diketik di sana hanya menumpang di dokumen JSON lewat `@GCNM.GetPageJSONString()` lalu
hilang dari pandangan.

Membawanya berarti menyalin isian yang sejak semula tidak berfungsi. Ketiadaannya dijaga
uji.

### 24.10 Yang berbeda dari Pega, dan alasannya masing-masing

Sesudah koreksi di atas, yang tersisa hanyalah hal yang menyangkut **mekanisme**, bukan isi
layar. Ketiganya dipertahankan dengan alasan yang dapat diperiksa:

| Hal | Pega | Di sini | Alasan |
|---|---|---|---|
| Pencarian | filter dropdown per kolom | satu kotak cari | komponen `DataTable` yang sama dipakai seluruh modul master; Pega pun menyaring, hanya bentuknya berbeda |
| Pesan hasil simpan | isian **Catatan** berisi `ErrMsg` | galat berkode; berhasil menutup form | kontrak galat `ErrMsg` tidak dibawa (`D-68`) — pada jalur BERHASIL pun ia berisi kalimat, sehingga pengguna harus membaca teks untuk tahu hasilnya |
| Peringatan "deskripsi ganda diterima" | tidak ada | ada, saat menambah | menggantikan pemeriksaan yang sengaja tidak ada; bentuknya sama dengan Master Dominan Factor |

**Satu pemberitahuan DICABUT.** Form pengubahan sempat memuat catatan bahwa deskripsi
dipakai mengelompokkan laporan. Akibat itu nyata — `BrowseCaseClaimPerCauseOfLoss-SQL.xml`
dan `GetDataXOLPerBusiness-SQL.xml` keduanya `GROUP BY COL_DESC` — tetapi layar Pega tidak
memuatnya, dan Work Owner meminta layar mengikuti Pega saja. Akibatnya tetap tercatat di
dokumen ini dan di komentar kode; ia hanya tidak lagi ditampilkan.

### 24.11 Label tombol diseragamkan ke Pega di SELURUH modul master

**Keputusan Work Owner 2026-09-20.** Setelah perbedaan diangkat, Work Owner menetapkan
seluruh modul master mengikuti Pega: label tombolnya **"Refresh"**, bukan "Muat ulang".

**Dasarnya, dan ia tidak menyisakan keraguan.** Ke-10 harness master di export diperiksa
satu per satu, dan **seluruhnya** memasang `pyButtonLabel` yang sama:

```
StatusClaimInbox · MasterRekening · StatusProgress · SurveyorsInbox
DetailSurveyorsInbox · UserTeknisInbox · MasterRecovery · DetailDominanFactor
CauseOfLossInbox                                    → Refresh
MasterProteksiVisibilityData                        → REFRESH
```

Tidak ada satu pun yang berbunyi "Muat ulang". `D-13` menetapkan teks yang dilihat pengguna
mengikuti layar lama, sehingga "Muat ulang" adalah penyimpangan yang tidak pernah
diputuskan — ia menyelinap, bukan dipilih. Docstring `DominantFactorPage` bahkan sudah
menyebut "Refresh" di tabel "yang ditiru", tetapi tombolnya tidak mengikuti.

**Yang disentuh, dan hanya itu.** Perubahannya satu kata per tempat, tanpa mengubah
perilaku apa pun:

| Jenis | Diubah? |
|---|---|
| Label tombol `{isFetching ? 'Memuat…' : 'Muat ulang'}` | **ya** — 7 modul |
| Kalimat yang MENYEBUT tombolnya ("…lalu tekan Muat ulang") | **ya** — agar tidak menunjuk tombol yang tidak ada |
| Komentar dan docstring yang menyebut tombolnya | **ya** — agar tidak menyesatkan pembaca berikutnya |
| Prosa umum ("Muat ulang daftarnya.") | **tidak** — ia kalimat Indonesia biasa, bukan rujukan tombol |
| "Muat ulang halaman" di `AccountForm` | **tidak** — ia memang memuat ulang HALAMAN peramban |

**Kenapa ini tidak melanggar Isolasi Protektif.** Aturan itu melarang **saya** mengubah
atau me-refactor modul yang sudah selesai atas inisiatif sendiri. Ini bukan itu: Work Owner
yang memintanya, perubahannya satu kata, tidak ada logika yang bergeser, dan tidak ada uji
modul terdahulu yang mengunci label itu — diperiksa lebih dulu, dan hasilnya nol.

**Satu modul TIDAK ikut diubah: `master-xol`.** Ia sedang ditulis sesi lain di pohon kerja
yang sama — berkasnya berubah beberapa kali selama sesi ini berlangsung, dan satu berkas
baru (`XOLForm.tsx`) muncul di tengah jalan. Menyuntingnya akan bertabrakan dengan pekerjaan
yang belum selesai. Ia masih berbunyi "Muat ulang" dan **perlu diseragamkan oleh sesi yang
memilikinya**.

**Yang belum ada penjaganya.** Hanya modul ini yang punya uji atas label tombolnya. Modul
lain tidak, sehingga "Muat ulang" dapat kembali menyelinap tanpa ada yang menahan.
Menambahkan uji setara ke sembilan modul lain adalah pekerjaan tersendiri, dan tidak
dikerjakan di sini.

### 24.12 Yang tetap terbuka setelah modul ini selesai

1. **`P-1` belum terpenuhi utuh** — layar Simas Online (MENU_ID 21) masih menulis tabel
   yang sama dari Pega. Cara menutupnya menunggu keputusan Work Owner; ketiga pilihannya
   tertulis di kepala migrasi 0005.
2. **Migrasi 0005 belum pernah dijalankan**, dan enam fakta skema di dalamnya masih
   ditebak. Langkah 0 berisi kueri yang harus dijalankan DBA lebih dulu.
3. **Isi master yang sebenarnya belum diterima.** Daftar contoh deskripsinya dikarang;
   hanya bentuk ID-nya yang diturunkan dari bukti.
4. **Empat field Simas Online belum terpetakan** — `BISNISID`, `MST_COL_ID`, `DISC`,
   `KOMISI` hidup di dalam dokumen JSON dan tidak ada di kolom mana pun. Ia menjadi bahan
   yang harus dijawab saat MENU_ID 21 dipindahkan.
5. **`master-xol` belum ikut diseragamkan** — ia sedang ditulis sesi lain dan masih
   berbunyi "Muat ulang" (§24.11). Perlu dikerjakan sesi yang memilikinya.
6. **Sembilan modul master belum punya uji atas label tombolnya**, sehingga "Muat ulang"
   dapat menyelinap kembali tanpa ada yang menahan (§24.11).
7. **Pemeriksaan peran belum ada** (`TKT-F3-005`), sama dengan seluruh rute lain hari ini.

### 23.13 Koreksi setelah audit ulang terhadap Pega (2026-09-20)

Enam keputusan di §23 dikoreksi setelah `MasterProteksi_Sec` dibaca langsung. Rinciannya di
`catatan-pengembangan.md` §23.12; yang berikut adalah keputusannya, bukan ceritanya.

| Keputusan semula | Keputusan sekarang | Dasar |
|---|---|---|
| Grid memuat MODUL & SUB MODUL | Keduanya hanya di **form**; grid memakai kolom **LIHAT MODUL** | susunan kolom `:15671`–`:17415` |
| KTP/Email/NoTelp satu kolom | **Tiga kolom terpisah** | `.STS_KTP` `:16158` · `.STS_EMAIL` `:16340` · `.STS_NOTELP` `:16516` |
| MAX CARI lalu MAX LIHAT | **MAX LIHAT lebih dulu** | `.LOGSEEN` `:16680` mendahului `.LOGSEARCH` `:16868` |
| Dua tipe pencarian | **Empat** tipe, termasuk status aktif | `SearchData.Type` 1–4 |
| Status **di luar** form, ada tombol Aktifkan | Status **di dalam** form; aksi hanya pada baris aktif | isian `:22449` · syarat `.STS_AKTF=='AKTIF'` pada kedua tombol |

**Yang paling penting dicatat sebagai pelajaran.** Pemisahan status dari form adalah
perlindungan yang saya tambahkan sendiri — masuk akal, tetapi **tidak tercatat di 13 butir
`P-5`**, dan karena itu melanggar prinsipnya. Pega sudah punya perlindungan yang sepadan di
tempat lain: tombol Ubah tidak muncul pada baris nonaktif. Keduanya kini mengikuti Pega.

**Konsekuensi yang menunggu keputusan Work Owner.** Baris yang sudah dinonaktifkan kini
**tidak dapat diaktifkan kembali dari layar ini** — sama persis dengan sistem lama, dan
sepuluh baris produksi berada dalam keadaan itu. Bila reaktivasi dikehendaki, ia
**penambahan perilaku** yang perlu diputuskan dan dicatat sebagai butir `P-5` baru.

**Temuan `R-16` baru:** `Emb_ModulForMaskingData` — section yang memuat isian MODUL dan SUB
MODUL pada form — **tidak ada di export**. Ini menguatkan keputusan sebelumnya bahwa
keduanya diketik bebas. Panel "TEMPLATE AKSES" **tidak dibangun**: kedua section yang
memasoknya mengikat properti dari domain yang berbeda sama sekali, sehingga tidak ada bukti
tentang apa yang sebenarnya ia kerjakan — dan menebaknya akan melanggar larangan dummy
logic.

---

## 25. Modul Master XOL (2026-09-21, sesi keenam belas)

Butir menu `MENU_ID 19`, harness `DetailMasterXOL`, empat tabel `POOLDATA.MST_XOL_*`.

### 25.1 Lingkupnya "sesuai Pega" — dan itu termasuk mengajukan ke komite

Work Owner memilih lingkup **sesuai aplikasi Pega**. Yang perlu dicatat: di layar lama,
menekan **Simpan** tidak berhenti pada menyimpan. `Activity/InsertUpdateMasterXOL-Act.xml`
menutup rangkaiannya dengan dua pemanggilan berturut-turut, keduanya **tanpa prasyarat**:

```
pySteps(10)  Call UpdateStatusMasterKomitexol  → PIC, STSKOMITE='0', REMARKPIC
pySteps(11)  Call SendDataMasterXOLToKomites   → pemberitahuan, FlagKomites=1
```

Akibatnya: menyimpan induk yang **sudah disetujui komite** mengembalikannya ke keadaan
menunggu. Itu masuk akal — strukturnya berubah, sehingga persetujuan atas bentuk sebelumnya
tidak lagi berlaku — dan ia ditiru apa adanya.

Layar menyatakannya terang-terangan di kepala form, supaya pengguna tidak terkejut
menemukan induk yang tadinya disetujui kembali berstatus menunggu.

**Yang TIDAK termasuk:** persetujuan komite itu sendiri. Ia butir menu tersendiri
(`MENU_ID 53`, harness `Inbox_XOL_Harness`) dan modul tersendiri.

### 25.2 Total share 100% adalah PERINGATAN, bukan penolakan

Ini keputusan yang paling mudah salah dibaca, sehingga buktinya dicatat utuh.

Layar lama memeriksa total share per lapisan — tetapi **tidak memblokir penyimpanan**.
Urutan langkahnya di `InsertUpdateMasterXOL`:

```
pySteps(2)     ulangi tiap lapisan
  pySteps(2.1) local.totalshare := 0
  pySteps(2.2) ulangi tiap reas → local.totalshare += .IndividualRiskPercentage
  pySteps(2.3) SETEL PESAN bila total bukan 100
pySteps(3)     SIMPAN INDUK        ← tanpa prasyarat apa pun
pySteps(7)     SIMPAN LAPISAN & REAS
pySteps(9)     tampilkan pesan bila ada
```

Langkah 3 dan 7 tidak punya prasyarat. Penyimpanan berjalan lebih dulu, pesannya muncul
sesudahnya. **Produksi membuktikannya**: lapisan `10004` milik induk `10002` tersimpan
dengan nol reasuradur.

Satu jebakan pembacaan dicatat supaya tidak diperbaiki keliru oleh orang berikutnya:
prasyarat langkah 2.3 berbunyi `local.totalshare==100` dengan kode cabang **`true=3`
(lewati)** dan **`false=2` (jalankan)**. Dibaca sekilas ia tampak terbalik — pesannya
justru muncul ketika total BUKAN 100.

Keputusan Work Owner: perilaku itu ditiru. Akibatnya pada kontrak API: jawaban simpan
membawa senarai **`peringatan`** yang terpisah dari galat, dan statusnya tetap `201`/`200`.

**Toleransi empat desimal `D-51` sengaja TIDAK dipakai.** Ia ditetapkan untuk aturan
spreading di modul `B-4`; memakainya di sini akan memperkenalkan aturan yang tidak pernah
ada di layar ini. Perbandingannya bilangan bulat, persis seperti `local.totalshare==100`.

### 25.3 Hapus BERKASKADE — satu-satunya selisih terencana yang diminta

`Activity/DeleteFromTabelMst-Act.xml.xml` menghapus **satu tabel per pemanggilan**.
Akibatnya terlihat di produksi pada 2026-09-20: induk `10003` sudah terhapus, tetapi
**2 lapisan dan 3 baris bisnis miliknya masih ada** dan tidak dapat dicapai layar mana pun.

Work Owner memilih **hapus fisik, tetapi berkaskade**. Menghapus satu induk membuang grup
bisnis, lapisan, dan reas di bawahnya dalam **satu transaksi**.

Urutannya mengikat: reas lebih dulu, lalu lapisan, lalu bisnis, baru induknya. Membalik dua
yang pertama menghilangkan satu-satunya cara mengetahui `IDLAYER` mana yang milik induk itu.

**Soft delete (`D-66`) tidak dipakai**, dan itu penyimpangan yang disadari. Alasannya:
`D-66` menuntut kolom penanda baru pada empat tabel lewat jalur DBA (`D-63`), yang akan
menahan modul ini sampai migrasi selesai — dan Work Owner memilih jalur yang tidak
tertahan. Dicatat sebagai utang: bila kebijakan soft delete kelak ditegakkan menyeluruh,
modul ini termasuk yang harus menyesuaikan.

### 25.4 Cacat produksi yang DITIRU, bukan diperbaiki

Penyaring grup bisnis mengikuti Type XOL, dan ketiga polanya disalin apa adanya dari
`ShowDetailGroupBisnisXol_Act`:

| Type | Pola | Hasil nyata di ASM |
|---|---|---|
| 1 | `%PROPERTY%` `%MOTOR%` `%ENGINEERING%` | 6 grup, termasuk HEAVY EQUIPMENT |
| 2 | `%PA%` `%GA%` | **hanya AVIATION HULL** |
| 3 | `%MARINE%` `%HEAVY EQUPMENT%` | MARINE CARGO, MARINE HULL |

**Type 2 cacat.** Ia dimaksudkan menjaring PA dan GA, tetapi keduanya bernaung di bawah
grup treaty "GENERAL ACCIDENT" yang tidak memuat potongan itu. Yang justru terjaring adalah
AVIATION HULL, karena induknya "AVIATION & AEROSPACE" memuat `PA` di dalam kata AERO**SPA**CE.

**Salah ketik `HEAVY EQUPMENT` tidak berakibat apa-apa.** Dijalankan langsung ke ASM, ejaan
yang benar maupun yang salah mengembalikan hasil yang sama: nama itu tidak ada di
`PROPORTIONALARRG.TREATYGROUPNAME` dalam ejaan mana pun.

Work Owner memilih keduanya ditiru (`P-5`). Sebuah uji dibuat khusus untuk membuktikan
cacat Type 2 memang direproduksi — bukan diam-diam diperbaiki.

### 25.5 Dua cacat yang JUSTRU diperbaiki, dan kenapa

Keduanya bertipe sama: parameter diterima procedure lalu **dibuang** pada cabang update.

| Cacat | Bukti | Akibatnya di sistem lama |
|---|---|---|
| `TYPEXOL` tidak ikut diubah | `INSERT_UPDATE_MST_XOL.prc:39` menyetel NAMA, TAHUN, KURSVALUE saja | Mengubah Type XOL pada induk yang sudah ada **tidak pernah tersimpan** |
| Nama reas tidak ikut diubah | `:115` menyetel `PERCENTSHARE` saja | Membetulkan ejaan nama reasuradur **tidak pernah tersimpan** |

Keduanya diperbaiki karena isian yang hilang tanpa pesan adalah kelas cacat yang paling
sulit dipercaya pengguna: layar menerima, menyatakan tersimpan, lalu menampilkan nilai lama.

**Selisihnya dicatat terbuka**: layar baru akan mulai menyimpan perubahan yang dulu hilang
diam-diam. Keduanya di luar 13 butir `P-5`, sehingga bila uji kesetaraan menemukannya, ia
sudah tercatat di sini sebagai perbaikan yang disengaja.

### 25.6 Tiga rule Pega hilang, dan apa yang dipakai sebagai gantinya

| Yang hilang | Perannya | Pengganti | Dasarnya |
|---|---|---|---|
| `GetTypeofxolclaim` | daftar pilihan Type XOL | label diturunkan dari isi penyaring bisnis | satu-satunya bukti tersisa: cabang `CNPSupportDoc == "1"/"2"/"3"` |
| `InputPanelReas` | picker Reasuransi | ID dan nama **diketik bebas** | section yang tersisa (`InputDetailPanelReasGenerated`) menampilkan keduanya sebagai isian teks |
| activity pengisi `TempYear` | daftar pilihan Tahun | `POOLDATA.M_TREATYYEAR` | satu-satunya master tahun treaty; memuat kelima tahun yang dipakai master XOL |

Untuk Reas, menebak masternya sengaja dihindari: dari 18 nama yang dipakai, hanya **5** ada
di `T_REINSURER` dan **8** di `TREATYREINSURER`; "SWISS RE" tidak ada di keduanya. Memilih
salah satunya akan membuat 13 nama yang sudah dipakai menjadi tidak dapat dipilih lagi.

Ketiganya dicatat sebagai `R-16` dan menunggu Tim Pega.

### 25.7 Penyimpangan yang disadari

**`ParseAmount` tidak ada di modul ini**, berbeda dari `masterrecovery` dan modul lain yang
punya pembaca angka dari teks. Versi pertama berkas ini punya, lalu dibuang setelah uji
membuktikannya **ambigu**: karena titik dibuang sebagai pemisah ribuan, `"13.500"` dan
`"13500.75"` menjadi angka yang sama — sehingga isian pecahan yang seharusnya ditolak
justru diterima sebagai `1.350.075`.

Ia dibuang, bukan ditambal, karena memang tidak dibutuhkan: seluruh angka di modul ini tiba
sebagai **angka** di dalam JSON. **Catatan untuk modul lain:** pola yang sama ada di
`masterrecovery.ParseAmount`; ia tidak disentuh karena Isolasi Protektif, tetapi
kelemahannya sama dan layak ditinjau bila modul itu dibuka lagi.

**Kolom `"LIMIT"` selalu dikutip** di seluruh kueri. Di Oracle ia bukan kata cadangan
sehingga dapat ditulis polos — dan justru itu yang membuat kelalaiannya tidak akan ketahuan
sampai cutover ke PostgreSQL, tempat `LIMIT` **adalah** kata cadangan. Sebuah uji menjaganya.

**Penerima notifikasi komite dari konfigurasi**, bukan dari master. `D-67` menetapkan
tempatnya adalah master **Penerima Notifikasi** (`F-4`), dan master itu belum dibangun.
`XOL_PENERIMA_KOMITE` adalah tempat sementara — tetap dapat diubah tanpa menyentuh kode,
tetapi belum dapat diubah pengguna bisnis sendiri.

**Metode `DELETE` ditambahkan ke `callAPI`.** Komentar di `api/client.ts` menyatakan ia
sengaja tidak ada dan menuntut "keputusan sadar" sebelum ditambahkan. Keputusan itu diambil
Work Owner pada 2026-09-20; alasannya dicatat di komentar yang sama.

### 25.8 Yang tidak dibangun, dan kenapa

**Baris grup bisnis yang `IDBUSINESS`-nya kosong tidak dapat dihapus.** Kueri hapus lama
mencocokkan `IDBUSINESS = :id`, dan pencocokan itu tidak pernah benar untuk NULL — sehingga
dua baris produksi milik induk 10009 **sudah tidak dapat dihapus dari layar Pega hari ini**.
Batasannya dipertahankan; layar mematikan tombolnya beserta keterangannya, bukan membiarkan
pengguna menekan tanpa akibat.

Sebuah kueri yang dapat menghapusnya sempat ditulis lalu **dibuang**: ia tidak dipanggil
dari mana pun, dan kueri hapus yang menganggur adalah kode mati yang berbahaya.

**Panel persetujuan komite tidak dibangun** — ia butir menu tersendiri (`MENU_ID 53`).
Kolom `KOMITE` dan `REMARKKOMITE` karena itu hanya **dibaca**, tidak pernah ditulis modul
ini.

### 25.9 Utang yang dicatat terbuka

| Utang | Pemilik |
|---|---|
| Label Type XOL menunggu `GetTypeofxolclaim` | Tim Pega |
| Sumber daftar Reasuransi menunggu `InputPanelReas` | Tim Pega |
| Sumber dropdown Tahun belum dikonfirmasi | Tim Pega / Work Owner |
| 5 baris yatim di produksi menunggu pembersihan | DBA (`D-63`) |
| Indeks unik pada `MST_XOL_BUSINESS` dan `MST_XOL_REAS` | DBA |
| Penerima notifikasi pindah ke master Penerima Notifikasi | `F-4` |
| Soft delete `D-66` bila kelak ditegakkan menyeluruh | Work Owner |
| Pemeriksaan peran per menu | `TKT-F3-005` |


---

## 17. Modul Master Dokumen Travel, dan merge kedua yang belum selesai (2026-09-21, sesi kesembilan)

### 17.1 Keputusan Work Owner pada sesi ini

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Repo tidak dapat dibangun — boleh diperbaiki? | **Boleh, dan rapikan `masterpicteknik` sekalian** |
| 2 | `POOLDATA.M_DOCTRAVEL` di basis data yang mana | **"Seperti aplikasi Pega"** → per entitas |
| 3 | Validasi isian | **Tiru Pega apa adanya — tanpa validasi** |

Jawaban kedua tidak menyebut salah satu pilihan yang saya tawarkan, dan saya membacanya sebagai
**per entitas** dengan alasan yang disampaikan lebih dulu sebelum mengerjakan: di Pega setiap
entitas menjalankan instansnya sendiri di atas POOLDATA-nya sendiri, dan
`Database/DOCTRAVEL_CVG.prc:12` justru membaca `M_SITE_DATABASE WHERE CURRENT_SITE = '1'` — kode
situs **tempat procedure itu berjalan**. Tidak ada satu pun kolom entitas di tabelnya, karena
memang tidak dibutuhkan: pemisahannya ada di tingkat basis data.

### 17.2 Repo tidak dapat dibangun saat sesi dimulai, dan itu ditemukan sebelum satu baris pun ditulis

Branch `feat/fran-master` pada commit `2322f15` ("Merge origin/master into feat/fran-master")
meninggalkan empat kerusakan yang ter-commit:

| Tempat | Kerusakan |
|---|---|
| `internal/masterrekening/http/galat.go` | tertinggal berdampingan dengan `errors.go` hasil rename → definisi ganda, paket gagal kompilasi |
| `internal/masterpicteknik/http/galat.go` | idem, ditambah `logging.Dari` yang sudah berganti nama menjadi `logging.From` |
| `frontend/src/app/App.tsx:91` | penanda konflik `<<<<<<< HEAD` ter-commit |
| `frontend/src/modules/master-rekening/AccountForm.tsx:106` | idem |

Sebabnya sama untuk keempatnya: merge itu **mengganti nama** berkas secara besar-besaran
(`galat.go`→`errors.go`, `rute.go`→`routes.go`, `memori/`→`memory/`), dan pada dua paket
pendeteksian rename-nya gagal sehingga berkas lama ikut bertahan.

Ini **pengulangan** dari §15.1: sesi 2026-09-18 pun dibuka dengan merge yang belum tuntas. Pola
yang sama, dua kali berturut-turut, pada merge yang sama-sama membawa penggantian nama massal.

**Yang dikerjakan atas izin Work Owner:**

1. `masterrekening/http/galat.go` dihapus. `errors.go` dipertahankan, tetapi ia **kehilangan satu
   fungsi** saat merge — `pelanggaranDari`, yang mengubah `ValidationError.Field` (peta) menjadi
   `Detail []PelanggaranDTO` (senarai terurut). Tanpa fungsi itu `errors.go` merujuk field
   `ErrorResponse.Field` yang sudah tidak ada. Fungsinya dipulihkan sebagai `violationsOf`,
   dengan pengurutan yang sama supaya jawaban atas permintaan yang sama selalu identik.
2. `masterpicteknik` dialihkan seluruhnya ke penamaan Inggris (`D-80`) — 12 berkas, 788
   identifier, 6 nama kueri `.sql`, dan 7 berkas/folder yang berganti nama.

### 17.3 Penggantian nama dikerjakan pemindai token Go, bukan `sed`

§14.3 sudah menetapkan alasannya: `sed` dengan batas kata merusak komentar dan literal string,
dan keduanya **tidak terdeteksi kompilator**. Perkakas sesi itu tidak ada di repo, jadi dibuat
ulang — kali ini di atas `go/scanner` pustaka standar, bukan pemindai buatan sendiri.

Keuntungannya nyata dan bukan soal kerapian:

- Hanya token bertipe `token.IDENT` yang diganti. Komentar (`token.COMMENT`) dan literal string
  (`token.STRING`) tidak pernah tersentuh — sehingga komentar berbahasa Indonesia tetap utuh
  (`peta-penamaan.md`), **dan nama field JSON pada tag struct tidak berubah sama sekali**. Yang
  terakhir itu penting: tag JSON adalah kontrak API, dan `sed` akan mengubahnya diam-diam.
- Nama kueri `.sql` juga literal, sehingga ia **tidak ikut** terganti dan harus dikerjakan
  terpisah — dan itu justru benar, karena penanda `-- name:` di berkas `.sql` harus berubah
  berbarengan dengan pemanggilnya di Go.

Perkakasnya dijalankan dua kali: sekali `collect` untuk mengumpulkan seluruh 300-an identifier
yang benar-benar ada, lalu `apply` dengan peta yang disusun dari daftar itu. Menyusun petanya
dari daftar yang nyata, bukan dari ingatan, yang mencegah identifier terlewat.

### 17.4 `PICTeknik` sengaja tidak diterjemahkan

`D-80` menetapkan identifier berbahasa Inggris, tetapi `PICTeknik` dibiarkan apa adanya.

Alasannya sama dengan alasan `D-81` mengembalikan nama modul ke bahasa Indonesia: ia **nama peran
bisnis** yang tertulis di `CONTEXT.md` ("PIC Teknik / User Teknis") dan yang dipakai Work Owner
saat menyebut modulnya. `TechnicalPIC` bukan istilah yang dikenal siapa pun di proyek ini.

### 17.5 Master Dokumen Travel: dua kolom, dan itu memang seluruhnya

Yang dibaca dari export sebelum menulis kode:

| Artefak | Yang diambil darinya |
|---|---|
| `Harness/BrowseMasterDocumentTravel_Harness-Harness.xml` | kelas `ASM-FW-GCNMFW-Int-M_DOCTRAVEL`, judul layar, tombol Tambah/Ubah/Simpan/Refresh |
| `Report Definition/BrowseMstDocTravel_RD-RD.xml` | dua kolom `DOCID` + `NAMADOKUMEN`, `ORDER BY DOCID ASC`, `pyMaxRecords=500` |
| `Section/BrowseMasterDocumentTravel-Section.xml` | grid dua kolom, tanpa satu pun `pyRequired=true` maupun `pyMaxLength` |
| `Activity/SetMstDocTravelValue_act-Act.xml` | memuat satu baris ke `TempMstDocTravel`, menandainya `"Update"` |
| `Activity/CNMInsertMstDocTravel_act-Act.xml` | sentinel `"UnknownID"`, urutan langkah simpan |
| `RDB List/UpdateMstDocTravel-SQL.xml` | pemanggilan `POOLDATA.DOCTRAVEL_CVG` dan pemetaan parameternya |
| `Database/DOCTRAVEL_CVG.prc` | **isi procedure-nya** — bentuk DOCID, pilihan INSERT/UPDATE, dan tiga cacatnya |

Berbeda dari `R-01` pada modul-modul sebelumnya, source procedure-nya **ada**. Tidak ada yang
perlu ditebak tentang aturan simpannya.

### 17.6 Tiga cacat procedure yang tidak dibawa

`Database/DOCTRAVEL_CVG.prc` adalah contoh buku untuk `D-68`:

| # | Cacat | Bukti |
|---|---|---|
| 1 | **Kontrak galatnya tidak dapat dipakai.** Parameter `ErrMsg` berisi kalimat **"Data Sudah Disimpan dengan ID : …"** pada jalur BERHASIL | `:24` dan `:38` |
| 2 | **Parameter keluarannya tertukar di sisi Pega.** `{OutputData.DOCID out}` dipetakan ke `format`, yang **tidak pernah diisi** pada jalur berhasil — ia hanya di-set `null` di handler terluar | `UpdateMstDocTravel-SQL.xml` vs `.prc:51` |
| 3 | **COMMIT sendiri, ROLLBACK setelahnya.** `:25` dan `:39` commit di dalam cabangnya; satu-satunya ROLLBACK (`:53`) berjalan SESUDAH commit itu dan tidak memulihkan apa pun | `.prc` |

Cacat kedua punya akibat yang terlihat pengguna dan tampaknya tidak pernah disadari: langkah
terakhir `CNMInsertMstDocTravel_act` menyetel `TempMstDocTravel.pyNote := OutputData.DOCID`,
sehingga **catatan itu selalu kosong** — sementara kalimat "Data Sudah Disimpan dengan ID …"
mendarat di `OutputData.NAMADOKUMEN`, yaitu kolom judul dokumen.

Ketiganya tidak dibawa. Yang dibawa hanya **aturannya**: bentuk DOCID, pilihan INSERT versus
UPDATE, dan kolom mana yang disentuh.

### 17.7 Tanpa validasi — dan kenapa itu berbeda dari Master Status Klaim

Pada 2026-09-17 Work Owner memutuskan Master Status Klaim **diperketat**: judul wajib diisi dan
tidak boleh ganda (§12.6). Pada 2026-09-21, untuk layar yang bentuknya nyaris identik, ia
memutuskan **sebaliknya**.

Perbedaan itu disengaja, bukan tidak konsisten, dan saya tidak menyamakannya sendiri. Akibatnya
mengalir ke seluruh lapisan:

| Lapisan | Akibat |
|---|---|
| Domain | tidak ada fungsi `Check`, tidak ada `ValidationError`, tidak ada `Violation` |
| Domain | hanya **satu** galat: `ErrNotFound` |
| Transport | tidak ada kode `validasi_gagal`, dan `ErrorResponse` **tidak punya** field `detail` |
| Basis data | **tidak ada indeks unik** di migrasi 0003 |
| Frontend | skema Zod ada tetapi **tanpa satu pun aturan**; isian tanpa `maxLength` |

Yang tetap dikerjakan, dan ia **bukan** validasi: `Input.Clean()` memangkas spasi tepi. Alasannya
teknis, bukan aturan bisnis — pembacaan memangkas padding kolom CHAR, sehingga tanpa memangkas
saat menulis, apa yang disimpan dan apa yang dibaca kembali dapat berbeda tanpa terlihat di layar.

**Dua uji mengunci keputusan ini** supaya penambahan aturan kelak menjadi keputusan yang disadari,
bukan kelalaian yang menyelinap: `TestEmptyTitleIsAcceptedLikeInPega` dan
`TestDuplicateTitleIsAcceptedLikeInPega`, masing-masing di sisi Go dan di sisi layar.

### 17.8 Per entitas, bukan portal utama — dan itu menyimpang dari Master Status Klaim

Master Status Klaim memakai koneksi portal **utama**; modul ini memakai **pemilih repo per
portal**, mengikuti Master Status Progres 1.

Keduanya master data, dan perbedaannya patut disebut terang-terangan: `D-75` butir 4 menetapkan
master data **per portal**, dan Master Status Klaim adalah pengecualiannya — bukan sebaliknya.
Daftar jenis dokumen Travel milik Asuransi Sinar Mas bukan milik Asuransi Simas Insurtech.

Jaminan yang mengikuti, dan diuji: portal yang tidak dikenal atau belum hidup **menghasilkan
galat**, tidak pernah dialihkan ke koneksi utama sebagai cadangan (`R-20`). Tiga uji menjaganya —
permintaan tanpa portal ditolak, data ASI tidak terlihat lewat ASM, dan menulis lewat ASI tidak
menambah baris di ASM.

### 17.9 Batas modul: ini master INDUK, bukan detailnya

Ada dua menu bernama mirip, dan hanya satu yang dikerjakan:

| MENU_ID | Nama | Harness | Tabel | Status |
|---|---|---|---|---|
| **22** | Master Dokumen Travel | `BrowseMasterDocumentTravel_Harness` | `M_DOCTRAVEL` | **dikerjakan** |
| 39 | Daftar Detail Dokumen Travel | `ListDocumentTravel` | `V_LST_DOC_TRAVEL` | belum |

Yang kedua merujuk `DOCID` milik yang pertama dan menambahkan `MINUNGGAH` dan `STSWAJIB` — aturan
wajib-tidaknya sebuah dokumen dan jumlah unggahan minimumnya.

Akibat yang mengikat modul ini: **DOCID tidak pernah berubah dan tidak pernah dihapus**. Batas ini
ditulis di kepala paket dan di peta rute menu, karena keduanya tempat orang berikutnya paling
mungkin tertukar.

### 17.10 Migrasi 0003 tidak mengubah skema, dan dijalankan di SETIAP entitas

Berbeda dari 0002, berkas ini tidak memuat satu pun `ALTER`, `CREATE`, atau `CREATE OR REPLACE
VIEW`. Isinya hanya GRANT dan kueri pemeriksaan, karena kedua kolom yang dibutuhkan sudah ada dan
keunikan judul justru diputuskan **tidak** ditegakkan.

Yang paling mudah terlewat, dan karena itu ditulis sebagai judul tersendiri di dalam berkasnya:
ia dijalankan di **setiap** basis data entitas. Entitas yang terlewat akan menjawab galat hak
akses saat pengguna membuka layarnya — bukan saat aplikasi start.

Tiga hal yang diminta ke DBA lewat berkas itu, dan ketiganya belum diketahui: bentuk kolom
`M_DOCTRAVEL` (`R-08`), keberadaan dan posisi `DOCTRAVEL_SEQ`, dan isi `M_SITE_DATABASE`.

### 17.11 Batas penomoran, direplikasi bukan ditambal

DOCID dibentuk `kode_situs || lpad(urutan, 5, '0')`, dan penampungnya di dalam procedure
dideklarasikan `varchar(8)` (`.prc:5`). Saat urutan mencapai 100000 nomornya menjadi enam digit
dan LPAD **memanjangkan**, tidak memotong.

Tidak dipotong menjadi lima digit di sisi Go, dan itu keputusan: memotongnya menghasilkan **DOCID
ganda** — dua jenis dokumen berbagi satu kunci yang dirujuk `V_LST_DOC_TRAVEL` dan dokumen klaim
yang sudah terunggah. Penyisipan yang gagal dengan ORA-12899 jauh lebih baik daripada itu.

Perlakuannya sama dengan `ThreeDigits` pada Master Status Klaim (§12), dengan satu perbedaan:
jaraknya ke batas **belum dapat dihitung**, karena `LAST_NUMBER` urutan dan lebar kolom DOCID
keduanya belum diketahui.

### 17.12 Yang belum dapat dibuktikan

| Hal | Sebab |
|---|---|
| Adapter SQL belum pernah menyentuh Oracle | tidak ada koneksi basis data di mesin ini; yang teruji adapter memori dan rakitan HTTP-nya |
| Lebar kolom `DOCID` dan `NAMADOKUMEN` | DDL tidak ada di export (`R-08`); migrasi 0003 langkah persiapan 1 yang menjawabnya |
| Keberadaan `POOLDATA.DOCTRAVEL_SEQ` di setiap entitas | idem, langkah persiapan 2 |
| Isi sebenarnya `M_DOCTRAVEL` | tidak ada CSV-nya di `Database/`, dan tidak ada satu pun judul dokumen tertulis di rule Pega mana pun. Isi contoh adapter memori adalah **susunan sendiri** dan ditandai demikian |
| Uji kesetaraan gerbang 1 | menuntut Pega staging yang dapat ditembak dari luar (`ADR-0027`), yang belum dikonfirmasi |

### 17.13 Utang yang tidak saya sentuh

| Hal | Kenapa dibiarkan |
|---|---|
| **3 uji `AccountPage` gagal** | Sudah gagal sebelum sesi ini — tabrakan nama tab dan tombol "Approve"/"Reject" yang §14.4 catat menunggu keputusan Work Owner |
| **`registry.ts` menunjuk `/master-rekening`** | Rute itu kini pengalihan ke `/master/rekening`; ia berfungsi, hanya menempuh satu lompatan tambahan. Merapikannya menyentuh modul yang sudah selesai |
| **Master Rekening tidak memakai `Protected`** | Ia dirender di dalam `<div>` polos, sehingga layarnya tidak punya bilah atas dan menu. Bentuk itu datang dari sisi `origin/master` saat konflik diselesaikan; mengubahnya keputusan tersendiri |
| **122 berkas Go tidak lolos `gofmt -l`** | Seluruhnya CRLF, dan itu berlaku repo-wide sejak sebelum sesi ini. Berkas baru sesi ini dibuat CRLF juga, supaya tidak menambah selisih baru |

---

## 18. Modul Master COL Simas Online (2026-09-21, sesi kesepuluh)

### 18.1 Keputusan Work Owner pada sesi ini

| # | Pertanyaan | Keputusan |
|---|---|---|
| 1 | Sisa merge `2322f15` membuat repo tidak build; sejauh apa boleh dibereskan? | **Bereskan seminimal mungkin** — hanya yang mematikan build, tanpa menyentuh logika modul yang sudah selesai |
| 2 | Isian "Bisnis" terbukti PageList (banyak baris) milik GISFW; lingkupnya? | **"seperti aplikasi PEGA"** — dibangun penuh sebagai grid banyak baris |
| 3 | `PEGA_M_CAUSE_OF_LOSS` harus ditulis ulang (`D-02`), tetapi DDL view-nya tidak ada (`R-08`) | **"metode penyimpanan sudah tidak pakai JSON lagi, langsung simpan ke data di tempat sesuai PEGA"** |

Keputusan ketiga adalah yang paling mengikat. Ia menempatkan modul ini di jalur yang **sama
persis** dengan Master Status Klaim: isi dokumen JSON dipindahkan ke kolom, view didefinisikan
ulang supaya membaca kolom, dan procedure ditinggalkan. Preseden itu ada di migrasi `0002`, dan
diikuti apa adanya — termasuk keputusan membiarkan kolom JSON lamanya tetap ada.

### 18.2 Nama modul

Mengikuti `D-81` — nama folder modul memakai nama modul bisnis yang Work Owner sebut, isinya
berbahasa Inggris:

| Lapisan | Nama |
|---|---|
| Paket Go dan folder backend | `internal/mastercolsimasonline` |
| Folder frontend | `src/modules/master-col-simas-online` |
| Tipe domain | `CauseOfLoss`, `Business`, `Input` |
| Komponen | `CauseOfLossPage.tsx`, `CauseOfLossForm.tsx` — nama **tipe domain**, bukan nama modul |

### 18.3 Bentuk penyimpanan yang dipilih

Sistem lama: `POOLDATA.M_CAUSE_OF_LOSS (M_COL_ID, JSON_DATA)` — satu CLOB berisi seluruh baris,
dibongkar kembali lewat `V_M_CAUSE_OF_LOSS (M_COL_ID, OLD_M_COL_ID, COL_DESC)`. Kolom
`OLD_M_COL_ID` **menyajikan dokumen JSON-nya**, dan layar Simas Online mem-parse kolom itu untuk
memperoleh `MST_COL_ID` dan senarai `BISNISID`
(`Activity/SetDataCauseofflossOnline-Act.xml`).

Sistem baru, ditetapkan migrasi `0004`:

| Objek | Isi |
|---|---|
| `M_CAUSE_OF_LOSS` | ditambah kolom `COL_DESC VARCHAR2(100)` dan `MST_COL_ID VARCHAR2(20)` |
| `M_CAUSE_OF_LOSS_BUSINESS` | **tabel baru** `(M_COL_ID, BISNISID, STS_AKTIF)`, PK gabungan |
| `V_M_CAUSE_OF_LOSS` | didefinisikan ulang supaya `COL_DESC` datang dari kolom |
| `V_M_CAUSE_OF_LOSS_BUSINESS` | didefinisikan ulang supaya membaca tabel, bukan JSON |

Nama kolom tabel pemetaan bukan karangan: `RDB List/GetLBUID_SQL-SQL.xml` menunjukkan bentuk yang
setara sudah ada di tingkat detail — `V_D_CAUSE_OF_LOSS_BUSINESS (D_COL_ID, BISNISID)` di-join ke
`BUSINESS (ID, NOTE)`.

### 18.4 Enam keputusan desain, beserta alasannya

**(a) Seam dipecah dua: `Repo` dan `BusinessRepo`.**
`BusinessRepo` **tidak punya satu pun operasi tulis**, dan itu disengaja. `POOLDATA.BUSINESS`
milik GISFW (`D-03`); batas kepemilikan itu ditegakkan oleh bentuk antarmuka, bukan oleh ingatan
orang yang menulis kode berikutnya. Uji `TestBusinessTableIsNeverWritten` menegakkannya sekali
lagi di tingkat kueri.

**(b) Daftar TIDAK membawa pemetaan bisnis.**
Grid Pega hanya menampilkan ID dan Description. Menarik pemetaan seluruh baris berarti satu kueri
yang hasilnya tidak pernah dilihat siapa pun. Konsekuensinya mengikat frontend: baris yang dibuka
untuk disunting **wajib** dimuat ulang lewat `GET /{id}`. Bila itu dilupakan, form memakai
`bisnis: []` dari daftar, tampak seolah seluruh bisnisnya sudah dihapus, dan menekan Simpan
benar-benar menghapusnya. Dijaga uji frontend dan dijelaskan di komentar `CauseOfLossPage.tsx`.

**(c) Pemetaan bisnis diganti dengan penanda, bukan hapus-lalu-sisip-ulang.**
`D-66` melarang penghapusan fisik data bernilai bisnis dan **secara khusus mencabut** pola
hapus-lalu-sisip-ulang (`PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503`). Menyimpan pilihan grid dengan
`DELETE` lalu `INSERT` adalah pola yang sama dalam ukuran kecil.

Yang dipakai: kolom `STS_AKTIF`; seluruh baris ditandai `'0'`, lalu yang dipilih dihidupkan
kembali (`UPDATE` dulu, `INSERT` hanya bila mengenai nol baris). Upsert bentuk ini dipilih karena
portabel — `MERGE` sintaksnya berbeda jauh antara Oracle dan PostgreSQL, dan
`INSERT ... ON CONFLICT` hanya ada di PostgreSQL (`D-20`).

Penanda dipilih, bukan versioning, karena bentuknya **sudah dipakai domain ini**:
`V_D_CAUSE_OF_LOSS` memuat kolom `STS_AKTIF`.

Konsekuensi yang mengikat: setiap kueri pembaca wajib menyaring `STS_AKTIF = '1'`
(`09-DATABASE-STRATEGY.md` §8.1). Dijaga `TestBusinessReaderFiltersInactiveRows`.

**(d) Keberadaan bisnis diperiksa aplikasi, bukan diserahkan ke kunci asing.**
Kunci asing ke `POOLDATA.BUSINESS` **sengaja tidak dibuat**: memasang constraint terhadap tabel
milik tim lain berarti perubahan mereka dapat menggagalkan penyimpanan di sini tanpa mereka tahu.
Pemeriksaannya di `usecase.checkInput`, yang juga menghasilkan pesan yang **menyebut bisnis mana**
yang salah — sesuatu yang tidak dapat dilakukan galat kunci asing.

Master dibaca **sekali** lalu dicocokkan di memori, bukan satu kueri per baris grid: satu kueri per
baris adalah N+1 yang dilarang `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 butir 4.

**(e) Nama bisnis tidak pernah diterima dari klien.**
`SaveRequest.bisnis` berisi ID saja. Nama dimiliki `POOLDATA.BUSINESS` dan selalu dibaca ulang
server; menerimanya dari luar berarti mempercayai klien atas data milik tabel lain.

**(f) `/master/bisnis` didaftarkan modul ini, bukan diberi awalan nama modul.**
Mengikuti `/master/posisi-klaim` pada modul Master Status Progres — daftar acuan dinamai menurut
isinya. Konsekuensinya dicatat di `routes.go`: bila kelak ada modul Master Bisnis tersendiri, rute
ini pindah, dan chi akan panik saat start bila keduanya mendaftarkannya bersamaan.

Berbeda dari `/master/posisi-klaim`, rute ini **menuntut portal**: `POOLDATA.BUSINESS` hidup di
basis data setiap entitas, sedangkan daftar posisi klaim memang milik aplikasi.

### 18.5 Penyimpangan yang disengaja dari sistem lama

| # | Perilaku Pega | Yang dibangun | Alasan |
|---|---|---|---|
| 1 | `COL_DESC` `pyRequired=false` — boleh kosong | **wajib diisi** | Baris tanpa nama muncul sebagai pilihan **kosong** di setiap dropdown penyebab kerugian. Perlakuannya sama dengan label Master Status Klaim yang sudah disetujui Work Owner |
| 2 | Bisnis kembar dapat tersimpan | **ditolak** | Tidak punya arti bisnis, dan pemetaannya berkunci `(M_COL_ID, BISNISID)` — membiarkannya lolos berarti galat kunci ganda mentah yang tidak dapat dibaca pengguna |
| 3 | Join implisit membuang pemetaan yang bisnisnya tidak ada | **LEFT JOIN**, baris tetap tampil dengan nama kosong | Baris yang hilang tanpa tanda membuat pengguna menyimpan ulang tanpa sadar telah kehilangan satu pemetaan |
| 4 | Sembilan `COMMIT` sendiri di procedure, `ROLLBACK` sesudah commit | satu transaksi | `D-68`. Akibatnya pada kegagalan **berbeda secara sengaja** dan sudah dinyatakan di muka (`14-TESTING-STRATEGY.md` §6.4 butir 2) |

Butir 1 sampai 3 **belum ada di daftar 13 butir `P-5`** (`D-49`). Ketiganya harus disetujui Work
Owner sebelum gerbang 1, sesuai `D-54`: selisih di luar 13 butir itu menuntut persetujuan tertulis.

### 18.6 Yang sengaja TIDAK dikerjakan

| Hal | Alasan |
|---|---|
| Tombol/rute hapus | `D-66`, dan sistem lama pun tidak punya satu pun `DELETE` terhadap tabel ini. Baris dirujuk `D_CAUSE_OF_LOSS.M_COL_ID` pada data berjalan. Dijaga `TestThereIsNoDeleteRoute` |
| Layar **Master COL biasa** (`CauseOfLossInbox`, MENU_ID 20) | Di luar permintaan. Tabelnya sama, tetapi layarnya tanpa ID Master Kerugian dan tanpa pemetaan bisnis. Butir menunya tetap "belum tersedia" |
| Master **Bisnis** | Milik GISFW (`D-03`). Modul ini hanya membacanya |
| Memperbaiki salah ketik `"ONLNE"` pada MENU_DESC | Ada di basis data, bukan di kode. Memperbaikinya dari kode membuat layar berbeda dari isi tabel; perbaikannya menempuh `D-63` |
| Memperbaiki 122 berkas Go ber-CRLF | Repo-wide, sejak sebelum sesi ini. Berkas baru sesi ini dibuat **LF** sehingga `gofmt -l` atasnya kosong — berbeda dari sesi §17 yang memilih CRLF, dan dicatat supaya perbedaannya tidak mengejutkan |
| Memperbaiki 3 uji `AccountPage` yang merah | Isolasi Protektif. Sudah merah sebelum sesi ini — dibuktikan dengan menjalankan uji terhadap `AccountForm.tsx` versi `origin/master` |

### 18.7 Utang teknis yang sengaja diciptakan

| Utang | Sebab | Kapan lunas |
|---|---|---|
| `MaxDescriptionLength` (100) dan `MaxMasterCodeLength` (20) **penjaga, bukan lebar kolom** | DDL `M_CAUSE_OF_LOSS` belum ada (`R-08`); layar Pega tidak membatasi apa pun | DBA mengirim DDL |
| Kedua angka itu **diduplikasi** di `CauseOfLossForm.tsx` | Pengguna perlu tahu sebelum mengirim; server tetap yang berwenang | tidak akan — dijaga terlihat oleh `TestLengthLimitsAreMirroredInTheFrontend` |
| Kunci JSON pada migrasi (`$.COL_DESC`, `$.MST_COL_ID`, `$.BISNISID[*].ID`) **diturunkan dari nama property Pega** | DDL view belum dibaca dari katalog | langkah 0 migrasi `0004` |
| Kode galat modul ini menyalin nilai modul Master Status Progres | Kontrak galat bersama (TKT-F1-004) masih terhalang | TKT-F1-004 diputuskan |
| Rute tanpa awalan `/v1` | Kontrak yang ada belum memakainya; memperkenalkannya di satu modul membuat dua gaya hidup berdampingan | penyeragaman menyeluruh |

### 18.8 Pertanyaan terbuka yang menahan modul ini

| Pertanyaan | Pemilik | Menahan apa |
|---|---|---|
| Bentuk sebenarnya `V_M_CAUSE_OF_LOSS` dan dokumen JSON-nya | DBA | langkah 1 dan 3 migrasi `0004` |
| Lebar `M_COL_ID` dan posisi `M_CAUSE_SEQ` | DBA | kapan penomoran mentok |
| `GRANT SELECT` atas `POOLDATA.BUSINESS` | DBA | isian Bisnis kosong tanpa itu |
| Persetujuan tiga penyimpangan §18.5 | Work Owner | gerbang 1 |
| Persetujuan titik cutover langkah 5 migrasi | Work Owner | sesudahnya layar Simas Online di Pega berhenti mutakhir |
| `CONTEXT.md` belum memuat istilah **Master COL Simas Online**, **ID Master Kerugian**, dan **Bisnis** sebagai entri | Work Owner | perubahan `CONTEXT.md` ditulis sebagai keputusan, bukan disunting diam-diam |

### 17.14 Pemeriksaan ulang terhadap layar Pega, dan empat teks yang dikoreksi (2026-09-21, lanjutan)

Work Owner meminta implementasinya diperiksa ulang terhadap export Pega. Pemeriksaan itu
menemukan **lima selisih**; tiga saya buat sadar, **dua tidak saya sadari sebelumnya**.

Yang lebih penting: pemeriksaan ini juga **membuktikan alurnya identik**, dan itu sebelumnya
hanya saya duga. `Section/BrowseMasterDocumentTravel-Section.xml:2465-2900` menunjukkan tombol
Tambah menjalankan Data Transform `CNMShowInsertMstDocTravel_dt`, yang isinya dua langkah:

```
REMOVE  TempMstDocTravel
SET     TempMstDocTravel.pyLabel = "Update"
```

`"Update"` adalah syarat tampil wadah form (`pyContainerVisibleWhen` di `:10371`). Jadi **Tambah
dan Ubah memang membuka satu form yang sama**, dan bedanya hanya isi halaman temp-nya — persis
bentuk yang sudah saya bangun. Tanpa membaca Data Transform itu, saya hanya dapat menduganya.

#### Keputusan Work Owner atas kelima selisih

| # | Hal | Pega | Semula saya tulis | **Diputuskan** |
|---|---|---|---|---|
| 1 | Label isian DOCID di form | **"ID Kerugian"** | "ID" | **ikuti Pega** |
| 2 | Header kolom kedua grid | **"Judul Dokumen Travel"** | "Judul Dokumen" | **ikuti Pega** |
| 3 | Judul form | **"Memperbaharui Data"**, sama untuk kedua mode | "Tambah/Ubah Dokumen Travel" | **ikuti Pega** |
| 4 | Wadah form | modal | panel inline | **tetap panel inline** |
| 5 | Grid | paginasi 50, tanpa pencarian | semua baris + kotak cari | **ikuti Pega** |

#### "ID Kerugian" adalah label yang KELIRU, dan tetap ditiru

Isian itu terikat ke `TempMstDocTravel.DOCID` (`:11051`), tetapi berlabel "ID Kerugian" —
"Kerugian" berarti *loss*, dan itu milik master Penyebab Kerugian. Ia terbawa karena rule ini
hasil klon: `pxMoveOriginalKey` pada `RDB List/UpdateMstDocTravel-SQL.xml` masih menunjuk
`ASM-FW-CNMFW-INT-V_M_SURVEYORS`.

Work Owner tetap memilih menirunya, dan alasannya berdiri: `D-13` menetapkan teks layar mengikuti
Pega supaya petugas tidak belajar ulang, dan **isian ini read-only** sehingga label yang keliru
tidak dapat menyesatkan pengetikan siapa pun. Memperbaikinya adalah keputusan tersendiri.

Ini menjadi contoh yang berguna untuk modul berikutnya: `D-13` menang atas kerapian penamaan
ketika teks yang keliru itu **tidak mengubah apa yang dilakukan pengguna**. Bandingkan dengan
`D-19`, yang mengganti nama *di dalam kode* — di sana tidak ada pengguna yang perlu mengenalinya.

#### Dua teks yang berbeda untuk satu kolom, dan itu memang begitu di Pega

Kolom `NAMADOKUMEN` punya **dua label berbeda** di layar lama:

| Tempat | Teks | Bukti |
|---|---|---|
| Header kolom grid | "Judul Dokumen **Travel**" | `Embed-Display-Table-Cell` indeks 2, `:7019` |
| Label isian di form | "Judul Dokumen" | `pyLabelFieldValue` `:11335` |

Keduanya ditiru apa adanya. Menyeragamkannya akan lebih rapi, dan justru karena itu perlu
ditulis: orang berikutnya yang melihat dua teks berbeda untuk satu kolom akan mengira salah
satunya salah ketik saya.

#### Judul "Memperbaharui Data" pada layar TAMBAH

Konsekuensi yang diterima secara sadar: menekan **Tambah** membuka form berjudul "Memperbaharui
Data", yang menyatakan hal yang tidak sedang terjadi. Itu perilaku Pega apa adanya — satu wadah,
satu judul, dua mode.

#### Yang TIDAK diikuti: bentuk modal

Satu-satunya selisih yang dipertahankan. Alasannya dua, dan keduanya diterima Work Owner:

1. **Keseragaman menang atas kesetiaan di sini.** Master Status Klaim dan Master Status Progres
   sudah memakai panel inline. Membuat layar ketiga berperilaku lain membuat petugas menghadapi
   dua pola di dalam satu kelompok menu yang sama.
2. **Panel membiarkan daftar tetap terlihat**, dan di layar ini itu berguna justru karena **judul
   ganda DIIZINKAN** — pengguna perlu melihat apa yang sudah ada sebelum mengetik, dan modal
   menutupi jawabannya.

#### `DataTable` bersama ditambah dua kemampuan opsional

Keputusan kelima menuntut dua hal yang belum dimiliki pustaka komponen: mematikan kotak cari, dan
paginasi. Keduanya ditambahkan sebagai **prop opsional dengan bawaan yang mempertahankan perilaku
sekarang** — `searchable = true` dan `pageSize` tidak diisi:

| Prop | Bawaan | Akibat pada layar yang sudah ada |
|---|---|---|
| `searchable` | `true` | **tidak ada** — kotak cari tetap tampil |
| `pageSize` | tidak diisi | **tidak ada** — seluruh baris tetap ditampilkan |

Ini yang membuat perubahan pada komponen bersama **tidak menyentuh** Master Status Klaim maupun
Master Status Progres, sekalipun berkasnya disunting. Dibuktikan dengan menjalankan seluruh suite:
101 lulus, dan ketiga yang gagal adalah ketiga kegagalan lama `AccountPage`.

Paginatornya **tidak digambar bila hanya ada satu halaman**. Pada master berisi puluhan baris,
itulah keadaan yang paling sering terjadi — dan tombol yang tidak pernah dapat ditekan bukan
petunjuk, ia gangguan. Nomor halaman juga tidak digambar satu per satu: dengan 50 baris per
halaman, master yang bertambah beberapa baris per tahun tidak akan pernah punya cukup halaman
untuk membuatnya lebih berguna daripada "Sebelumnya / Berikutnya".

#### Lima uji baru mengunci kelimanya

Ditambahkan ke `TravelDocumentPage.test.tsx`, di bawah `describe('kesetiaan pada layar Pega')`.
Alasannya disebut di dalam berkasnya: tanpa uji, keempat teks itu **tampak seperti kelalaian**
bagi orang berikutnya — label yang keliru, judul yang menyatakan hal yang salah, dan kotak cari
yang hilang semuanya "terlihat seperti bug" dan mengundang diperbaiki diam-diam.

| Uji | Yang dikunci |
|---|---|
| `memakai label "ID Kerugian" seperti layar lama, walau labelnya keliru` | selisih 1 |
| `memakai judul form yang SAMA untuk tambah dan ubah` | selisih 3, sekaligus membuktikan form tambah terbuka kosong |
| `tidak menampilkan kotak pencarian` | selisih 5a |
| `memaginasi 50 baris per halaman seperti pyPageSize` | selisih 5b, dengan 120 baris |
| `tidak menampilkan tombol halaman bila hanya ada satu halaman` | perilaku paginator |

Header kolom "Judul Dokumen Travel" (selisih 2) sudah terkunci uji yang ada sejak awal.

#### Satu pelajaran metode

Kelima selisih ditemukan dengan membaca **satu berkas yang semula saya lewati**:
`Section/BrowseMasterDocumentTravel-Section.xml`. Pada putaran pertama saya membacanya hanya untuk
mencari nama activity dan ada-tidaknya validasi, lalu berpindah ke Report Definition untuk
mendapatkan kolomnya.

Report Definition memberi **nama kolom basis data**; Section memberi **teks yang dilihat
pengguna**. Keduanya tidak sama, dan mengambil teks layar dari Report Definition adalah kekeliruan
yang tidak akan pernah terdeteksi kompilator maupun uji yang saya tulis sendiri.

> Untuk modul berikutnya: **teks layar hanya boleh diambil dari Section dan Harness.** Report
> Definition menjawab pertanyaan "kolom apa", bukan "tertulis apa".

---

## 19. Koreksi Master COL Simas Online setelah pemeriksaan ulang ke Pega (2026-09-21)

### 19.1 Empat keputusan Work Owner

Diambil setelah pemeriksaan ulang terhadap export menemukan empat hal yang meleset pada
implementasi pertama.

| # | Pertanyaan | Keputusan | Akibat |
|---|---|---|---|
| 1 | `MST_COL_ID` teks bebas atau daftar? | **Dropdown cause of loss lain** | mengubah arti kolom dan model validasi |
| 2 | Label dan urutan isian | **Samakan persis dengan Pega** | teks dan susunan form |
| 3 | Judul form mode tambah | **Samakan dengan Pega** — satu judul untuk dua modus | teks |
| 4 | Bisnis di luar master | **Terima apa adanya** | mengubah bentuk tabel pemetaan |

### 19.2 `MST_COL_ID` adalah rujukan-diri

Dugaan pertama — bahwa isinya kode padanan di sistem Simas Online — **terbantah oleh export**.
`Section/Online_BrowseCauseOfLoss-Section.xml:4594-4604`: isiannya berupa daftar bersumber
`BrowseVMCauseOfLoss_RD`, `pyAppliesTo = ASM-FW-GCNMFW-Int-V_M_CAUSE_OF_LOSS`,
`pyValue = .M_COL_ID`, `pyPrompt = .COL_DESC`.

Artinya penyebab kerugian ini **bernaung di bawah penyebab kerugian lain** di master yang sama.

Dua aturan baru yang mengikuti, keduanya di usecase karena menuntut membaca penyimpanan:

1. **Induk yang tidak ada ditolak sebagai pelanggaran isian**, bukan diserahkan ke kunci asing.
   Kunci asing menjadi 500 di layar dan tidak menyebut kode mana yang salah.
2. **Rujukan-diri ditolak.** Baris yang menjadi induk bagi dirinya sendiri membentuk lingkaran
   yang membuat setiap penelusuran jenjang berputar tanpa henti. Layar pun mengeluarkan baris yang
   sedang disunting dari pilihannya, sehingga penolakan itu tidak pernah perlu terjadi.

Keduanya **tidak ada di Pega**, dan karena itu masuk daftar selisih terencana yang menunggu
persetujuan Work Owner sebelum gerbang 1 (`D-54`).

### 19.3 Nama bisnis boleh diketik bebas — dan itu mengubah bentuk tabel

`Section/Online_BrowseCauseOfLoss-Section.xml:6089` menetapkan `pyAllowFreeFormInput=true` pada
isian Bisnis. Work Owner menetapkan perilaku itu dipertahankan.

**Konsekuensi yang menentukan: `BISNISID` boleh NULL**, sehingga ia tidak dapat menjadi kunci.

| Keputusan | Alasan |
|---|---|
| Kunci baris pemetaan adalah **NAMA_BISNIS** | ia selalu terisi, dan ia yang benar-benar terikat di layar Pega (`pyValue = .Note`) |
| Permintaan API membawa **nama**, bukan ID | nama yang diketik bebas memang tidak punya ID; server yang menyelesaikannya ke ID |
| Nama yang cocok disimpan dalam **ejaan master** | supaya satu bisnis tidak tampil dalam dua ejaan; "fire / property" menjadi "FIRE / PROPERTY" |
| Kolom **URUTAN** ditambahkan | tanpa itu pengurutan satu-satunya adalah BISNISID, yang menempatkan baris tanpa ID di satu ujung |
| Keunikan lewat **indeks unik atas `UPPER(TRIM(NAMA_BISNIS))`** | kunci utama tidak dapat dibuat atas ekspresi |
| Join ke `POOLDATA.BUSINESS` **dibuang** dari pembaca | nama sudah tersimpan; join akan gagal pada baris yang memang tidak punya ID |
| Kunci asing ke `POOLDATA.BUSINESS` **tidak dibuat** | ia akan menolak tepat baris yang diputuskan harus diterima |
| Kegagalan membaca master **tidak menggagalkan penyimpanan** | master hanya melengkapi ID; menolak penyimpanan yang sah lebih buruk |

Satu ekspresi normalisasi dipakai di **tiga tempat** dan harus sama persis:
`mastercolsimasonline.SameBusiness` di Go, `UPPER(TRIM(NAMA_BISNIS))` pada kueri upsert, dan
indeks unik migrasi `0004`. Bila ketiganya berbeda pendapat, baris kembar lolos ke basis data.

### 19.4 Komponen bersama baru: `ComboField`

`<input list>` + `<datalist>` — menawarkan saran tanpa memaksanya.

`SelectField` tidak dapat menirunya: dropdown menutup nilai di luar daftar, dan itu mengubah
perilaku layar. `<datalist>` adalah padanan HTML baku yang paling dekat, tanpa pustaka tambahan
dan tanpa menulis sendiri penanganan papan ketik yang sudah disediakan peramban.

Ditaruh di `src/components/`, bukan di dalam modul, karena isian "bebas ketik dengan saran" akan
muncul lagi pada layar lain dan `08-TECHNICAL-STRATEGY.md` §5 menuntut seluruh isian memakai
komponen baku. Berkasnya **baru** — tidak menyunting komponen yang sudah ada, sehingga Isolasi
Protektif tidak tersentuh.

### 19.5 Daftar selisih terencana menjadi EMPAT

Seluruhnya di luar 13 butir `P-5` (`D-49`), sehingga seluruhnya menuntut persetujuan tertulis Work
Owner sebelum gerbang 1 (`D-54`):

| # | Perilaku Pega | Yang dibangun |
|---|---|---|
| 1 | `COL_DESC` boleh kosong | **wajib diisi** |
| 2 | Nama bisnis kembar dapat tersimpan | **ditolak** |
| 3 | Induk yang tidak ada dapat tersimpan | **ditolak** |
| 4 | Baris dapat menjadi induk bagi dirinya sendiri | **ditolak** |

Ditambah dua perubahan yang mengubah keluaran tanpa mengubah aturan, dan karena itu ditangani di
sisi perkakas pembanding, bukan sebagai selisih: **soft delete** pada pemetaan bisnis (`D-66`) dan
**satu transaksi** menggantikan sembilan `COMMIT` (`D-68`).

### 19.6 Yang TETAP menjadi utang

| Utang | Kapan lunas |
|---|---|
| `MaxBusinessNameLength` = 100 masih penjaga; lebar `BUSINESS.NOTE` belum diketahui | DBA mengirim DDL |
| Ketiga batas panjang diduplikasi di `CauseOfLossForm.tsx` | tidak akan — dijaga terlihat oleh `TestLengthLimitsAreMirroredInTheFrontend` |
| Kunci JSON pada migrasi (`$.ID`, `$.Note`) diturunkan dari nama property Pega | langkah 0 migrasi `0004` |
| `ComboField` belum punya berkas ujinya sendiri | diuji tidak langsung lewat `CauseOfLossPage.test.tsx`; uji komponennya pekerjaan tersendiri |

---

## 20. Keempat penyimpangan Master COL dicabut — kesetaraan penuh dengan Pega (2026-09-21)

### 20.1 Keputusan Work Owner

> "untuk yang butuh persetujuan silahkan cek lagi pada PEGA samain ja"

Keempat hal yang bagian sebelumnya catat sebagai "selisih terencana menunggu persetujuan"
**dicabut**. Layar disamakan dengan Pega.

Keputusan ini menegakkan `P-5` dalam bentuknya yang paling langsung: perilaku dipertahankan lebih
dulu, dan perbaikan hanya yang sudah diputuskan eksplisit di `D-49` — yang tidak memuat satu pun
butir tentang layar ini.

### 20.2 Bukti yang mendasari, bukan asumsi

Tiap butir diperiksa ke export sebelum diubah:

| Klaim | Bukti |
|---|---|
| Pega tidak memvalidasi apa pun di layar ini | `pyRequired=false` pada **seluruh 19 isian**; activity simpan hanya `Page-New`/`Property-Set`/`RDB-List`/`Page-Remove`; **tidak ada direktori `Validate/`** di export |
| Bisnis kembar diizinkan | **nol penanda** unique/duplicate di seluruh repeat grid |
| Rujukan-diri diizinkan | dropdown bersumber `BrowseVMCauseOfLoss_RD` **tanpa penyaring** |
| Rujukan-diri tidak berbahaya | `MST_COL_ID` dibaca **hanya 2 tempat**, keduanya layar ini; **nol** di seluruh `Database/*.prc` — tidak ada yang menelusuri jenjangnya |

### 20.3 Yang dicabut

| # | Aturan yang dibuang | Ganti |
|---|---|---|
| 1 | Nama Cause of loss wajib diisi | boleh kosong |
| 2 | Bisnis kembar ditolak | diterima, kedua baris tersimpan |
| 3 | Judul berbeda untuk mode tambah | satu judul untuk dua modus |
| 4 | Rujukan-diri ditolak; baris sendiri dikeluarkan dari dropdown | diterima; baris sendiri ikut ditawarkan |

Yang **tetap** hanyalah batas panjang ketiga isian teks — dan itu bukan aturan bisnis melainkan
penjaga terhadap lebar kolom: tanpa itu, nilai kepanjangan ditolak Oracle dengan ORA-12899 yang
muncul sebagai galat 500 dan tidak dapat dibaca pengguna.

### 20.4 Satu pemeriksaan yang TIDAK dicabut

**Keberadaan induk tetap diperiksa server** — dan itu bukan penyimpangan.

Di Pega isiannya `pxDropdown` yang hanya menawarkan kode yang ada, sehingga kode yang tidak ada
tidak pernah dapat tersimpan lewat layar itu. Aturannya sama; yang berbeda hanya tempat
penegakannya, dan `D-59` menuntut setiap endpoint memeriksa sendiri karena API dapat ditembak
tanpa melewati layar.

Presedennya sudah disetujui: `masterstatusprogres.Input.Check` menolak kode posisi yang tidak
dikenal dengan alasan yang sama persis.

### 20.5 Akibat rancangan: kunci baris pemetaan berpindah ke POSISI

Mengizinkan bisnis kembar membatalkan rancangan sebelumnya, karena nama tidak lagi dapat menjadi
kunci.

| Kandidat kunci | Status | Sebab |
|---|---|---|
| `BISNISID` | gugur | boleh NULL — nama yang diketik bebas tidak punya ID |
| `NAMA_BISNIS` | gugur | tidak unik — bisnis kembar diizinkan |
| **`URUTAN`** | **dipakai** | posisi baris di grid; `TempCauseOfLoss.BISNISID` di Pega memang PageList |

Perubahannya:

| Hal | Sebelum | Sesudah |
|---|---|---|
| Kunci tabel | indeks unik atas `UPPER(TRIM(NAMA_BISNIS))` | **PRIMARY KEY `(M_COL_ID, URUTAN)`** |
| Kueri upsert | `WHERE ... UPPER(TRIM(NAMA_BISNIS)) = ...` | `WHERE ... URUTAN = ...` |
| Helper domain | `SameBusiness` — deteksi kembar | **`NormalizeBusinessName`** — pencocokan ke master |

Konsekuensi yang diterima sadar: mengubah baris pada satu posisi **menimpa** nilai lamanya, bukan
menyimpannya sebagai riwayat. Itu perlakuan yang sama dengan setiap kolom lain — `D-66` menyangkut
penghapusan, bukan versioning setiap suntingan. Penghapusan baris tetap lewat penanda `STS_AKTIF`.

### 20.6 Daftar selisih terencana menjadi NOL

Sebelumnya empat, dan seluruhnya dicabut. Yang tersisa hanyalah dua perubahan yang mengubah
keluaran **tanpa mengubah aturan**, dan keduanya sudah punya dasar keputusan sendiri:

| Perubahan | Dasar | Perlakuan pada uji kesetaraan |
|---|---|---|
| Soft delete pada pemetaan bisnis | `D-66` | dibandingkan lewat hasil kueri, bukan `COUNT(*)` |
| Satu transaksi menggantikan sembilan `COMMIT` | `D-68` | keadaan akhir saat gagal memang berbeda |

Gerbang 1 modul ini karena itu **tidak lagi menunggu persetujuan selisih** (`D-54`).

### 20.7 Utang yang tersisa

| Utang | Kapan lunas |
|---|---|
| Tiga batas panjang masih penjaga, bukan lebar kolom yang diketahui | DBA mengirim DDL `M_CAUSE_OF_LOSS` dan `BUSINESS` |
| Ketiganya diduplikasi di `CauseOfLossForm.tsx` | tidak akan — dijaga `TestLengthLimitsAreMirroredInTheFrontend` |
| Kunci JSON pada migrasi diturunkan dari nama property Pega | langkah 0 migrasi `0004` |
| `ComboField` belum punya berkas ujinya sendiri | diuji tidak langsung lewat `CauseOfLossPage.test.tsx` |
| Uji kontrak bersama untuk adapter memori dan SQL belum ada | pekerjaan tersendiri; lihat catatan sesi |

---

## 21. Daftar Tipe Dokumen — keputusan yang membentuknya (2026-09-21)

### 21.1 Keputusan Work Owner

| # | Perkara | Keputusan |
|---|---|---|
| 1 | `STS_PROSES` | **Teks bebas, ditiru apa adanya** — bukan penanda aktif/non-aktif |
| 2 | Validasi isian | **Tidak ada**, seperti Master Dokumen Travel |
| 3 | Bentuk penyimpanan | **Tanpa JSON** — nilai ditulis langsung ke kolom, di tempat yang sama seperti Pega |
| 4 | Bentuk grid | **Tanpa kotak cari**, 50 baris per halaman |
| 5 | Lingkup portal | *diperiksa ke Pega atas permintaan Work Owner* — **per entitas** |

### 21.2 Kenapa `STS_PROSES` tidak dijadikan sakelar aktif

Ini keputusan yang paling mudah diambil keliru, dan alasannya bukan selera:

| Bukti | Artinya |
|---|---|
| `Section/BrowseListDocumentType-Section.xml:13455` — `pyEditOptions=Auto`, tanpa daftar pilihan | Di Pega ia memang isian teks |
| `Activity/SearchDataArchiveFilling-Act.xml` membacanya sebagai `"NoteKasir"` | Ada pembaca lain yang memperlakukannya sebagai catatan |

Menjadikannya dropdown berarti menolak isian yang selama ini sah, dan menyempitkan kolom yang
dibaca layar Arsip Dokumen. `P-5` berlaku — perilaku ini tidak ada di daftar 13 perbaikan eksplisit
`D-49`. Di layar ia dirender **teks biasa**, bukan lencana berwarna: lencana akan menyiratkan domain
tertutup yang tidak ada, dan membuat nilai yang tidak dikenali tampak seperti data rusak.

### 21.3 Kenapa modul ini portal-aware, dan bagaimana itu dibuktikan

Work Owner tidak menjawab langsung, melainkan meminta diperiksa ke Pega. Hasilnya:

`Database/PEGA_LST_DOC_TYPE.prc:12` membentuk ID dengan membaca `M_SITE_DATABASE` — **persis
mekanisme** `DOCTRAVEL_CVG.prc:12` (Master Dokumen Travel) dan `PEGA_M_CAUSE_OF_LOSS.prc:11`
(COL Simas Online), keduanya sudah diputuskan per entitas. Kode situs melekat pada basis data tempat
procedure berjalan, sehingga **tiap entitas menerbitkan awalan ID-nya sendiri** — dan itu hanya
masuk akal bila tabelnya memang terpisah per entitas.

Konsekuensinya sama dengan ketiga modul portal-aware yang sudah ada: `RepoSelector` per portal,
middleware `ActivePortal` atas seluruh rute, portal yang tidak dikenal **ditolak** dan tidak pernah
dialihkan ke koneksi utama (`R-20`), dan migrasi 0005 dijalankan di **basis data setiap entitas**.

### 21.4 Jejak simpan: yang baru di modul ini

Tiga modul master sebelumnya tidak menulis `USER_EDIT`/`TGL_EDIT`; modul ini menulis keduanya karena
`CNMInsertListDocumentType_act` di Pega pun menulisnya. Dua keputusan rancangan mengikutinya:

| Keputusan | Alasan |
|---|---|
| Identitas datang lewat **jembatan `Caller` dari cmd**, bukan impor modul auth | Modul tidak saling mengimpor lapisan transport-nya; bentuknya sama dengan Master Rekening |
| Waktu datang lewat **seam `Clock` di usecase**, bukan `time.Now()` di repo dan bukan jam basis data di SQL | `F-5` menetapkan satu seam; tanpa itu penyimpanan tidak dapat diuji deterministik, dan waktunya akan mengikuti zona waktu server basis data (`R-12`) |

**Identitas yang kosong tidak menghentikan penyimpanan.** Ia hanya mengisi kolom jejak, bukan
menentukan apa yang tersimpan — menolak penyimpanan karenanya akan membuat petugas kehilangan isian
yang sudah diketik demi kolom yang tidak dilihat siapa pun di layar.

Keduanya **tidak dapat dikirim klien**: `SaveRequest` hanya menerima dua field dan menolak field
tak dikenal, sehingga badan permintaan yang menyertakan `id` atau `user_edit` dijawab `400`. Tanpa
itu, `USER_EDIT` — kolom yang justru dipakai menelusuri siapa mengubah apa — dapat diaku-aku.

### 21.5 Cacat sistem lama yang TIDAK dibawa

| Cacat | Bukti | Perlakuan |
|---|---|---|
| Kontrak galat berbasis teks — `ErrMsg` berisi kalimat sukses | `PEGA_LST_DOC_TYPE.prc:22`, `:35` | Tidak dibawa (`D-68`) |
| Kegagalan `RETURN` diam-diam; pemanggil menerima bentuk yang sama seperti sukses | `:15-18`, `:25-29`, `:38-42` | Menjadi galat yang benar-benar galat |
| `ROLLBACK` atas transaksi milik pemanggil | `:17`, `:27`, `:40` | Transaksi dimiliki Go (`D-68`) |
| `replace(DataPega,'UnknownID',id)` menimpa teks "UnknownID" yang kebetulan diketik petugas **di isian mana pun** | `:23` | Hilang bersama JSON-nya |

Yang terakhir layak diperhatikan: ia cacat yang hanya mungkin ada karena penyimpanannya JSON, dan
ikut hilang tanpa perlu keputusan tersendiri.

### 21.6 Selisih terencana terhadap Pega: NOL

Tidak ada satu pun perilaku yang sengaja dibuat berbeda. Yang berbeda hanyalah hal yang tidak
terlihat pengguna atau memang bukan perilaku data:

| Berbeda | Sifatnya |
|---|---|
| Layar sempit menjadi kartu | tampilan (`D-12`) |
| Entitas yang dilihat disebut terang-terangan | keamanan (`R-20`) |
| "Status Proses" diberi keterangan singkat | penjelasan, bukan aturan |
| Transaksi atomik dan kontrak galat yang benar | tidak terlihat pengguna (`D-68`) |

Judul form **"Update Data"** dipertahankan pada mode TAMBAH sekalipun, dan itu memang menyatakan hal
yang tidak sedang terjadi — sama seperti Master Dokumen Travel yang layar tambahnya berjudul
"Memperbaharui Data". `D-13` menang: petugas yang hafal layar lama membaca teks yang sama persis.

Begitu pula dua label yang **berbeda satu sama lain** di layar lama dan keduanya ditiru: header grid
berbunyi **"Tipe Dokumen"**, label isian di form berbunyi **"Jenis Dokumen"**.

### 21.7 Yang tetap menjadi utang

| Utang | Pemilik |
|---|---|
| DDL `V_LST_DOC_TYPE` — menentukan apakah `OLD_ID` kolom atau nilai JSON | DBA |
| Lebar kolom `ID`; batas nomor urut empat digit belum dapat diperiksa (`R-08`) | DBA |
| Isi tabel yang sebenarnya; contoh di `sample.go` bukan data produksi | DBA |
| Pemeriksaan peran — "apakah pemanggil memiliki menu Master Data" (`D-59`) | `TKT-F3-005`, terhalang `TKT-F3-004` |
| Awalan `/v1` pada jalur API | `TKT-F1-004`, berlaku seluruh aplikasi |
| Kontrak galat bersama | `TKT-F1-004` |

Dua master **turunan** layar ini belum dibangun, dan keduanya merujuk `ID` yang diterbitkan modul
ini: "Daftar Detail Tipe Dokumen" (`V_LST_DET_TYPE_DOC`) dan "Detail Tipe Dokumen per Bisnis"
(`LST_TYPE_DOC_BUSINESS`). Itulah sebab `ID` tidak pernah boleh berubah dan tidak pernah dihapus.

---

## 22. Daftar Detail Dokumen Travel — keputusan yang membentuknya (2026-09-22)

Modul kelima dari rumpun master, dan **turunan langsung** dari modul §17. Batas keduanya sudah
ditulis di §17.9 sebagai "belum" — sesi ini yang mengerjakannya.

| MENU_ID | Nama | Harness | Tabel | Status |
|---|---|---|---|---|
| 22 | Master Dokumen Travel | `BrowseMasterDocumentTravel_Harness` | `M_DOCTRAVEL` | selesai §17 |
| **39** | **Daftar Detail Dokumen Travel** | `ListDocumentTravel` | `V_LST_DOC_TRAVEL` | **selesai sesi ini** |

### 22.1 Keputusan Work Owner yang membentuk modul ini

| # | Perkara | Keputusan | Akibatnya |
|---|---|---|---|
| 1 | Jalur simpan hilang dari export | **Bangun utuh, simpan langsung ke basis data tanpa JSON** | bentuk INSERT/UPDATE disusun di modul ini, nama objeknya ditanyakan ke DBA |
| 2 | Lingkup sub-grid Plan/Jaminan | **Ikut dibangun, "seperti aplikasi PEGA saja"** | modul memuat dua tabel, bukan satu |
| 3 | Validasi isian | **Tidak ada**, sama seperti §17 | hanya pemangkasan spasi dan penjaga bentuk angka |

Jawaban 2 disertai perintah **"coba cek lagi"**, dan pengecekan ulang itu mengubah rancangan —
lihat §22.3.

### 22.2 Apa yang membedakan modul ini dari empat modul master sebelumnya

**Jalur tulisnya tidak dapat ditiru, hanya disusun.** Ini pertama kalinya. Pada keempat modul
sebelumnya selalu ada procedure atau activity yang memperlihatkan bentuk penyimpanannya —
`DOCTRAVEL_CVG.prc`, `PEGA_M_CAUSE_OF_LOSS.prc`, `PEGA_LST_DOC_TYPE.prc`. Di sini tidak ada satu
pun: tombol Simpan menunjuk `CNMInsertDocumentTravel_act` dan tombol Ubah menunjuk
`CNMSetDetailTravelDocument_act`, dan **keduanya hilang dari export** (`R-16`).

Yang dapat dibaca tetap banyak — APA yang disimpan terbaca lengkap dari form dan kedua view. Yang
tidak dapat dibaca hanyalah KE MANA. Perlakuannya:

1. Seluruh nama objek tulis **diisolasi di satu berkas** —
   `repo/sqlstore/daftardetaildokumentravel.sql`. Tidak ada nama tabel yang tercecer di dalam kode
   Go, sehingga jawaban DBA yang berbeda hanya mengubah satu berkas.
2. Ketiga nama yang merupakan **dugaan** diberi tanda sebagai dugaan, di kepala berkas itu, dengan
   menyebut dari mana masing-masing diturunkan.
3. Migrasi `0006` ditulis **sebagai daftar pertanyaan**, bukan sekadar pemberian hak. Bagian 1-nya
   enam kueri katalog yang jawabannya menentukan apakah modul dapat menyimpan sama sekali.

| Objek | Sumber namanya | Status |
|---|---|---|
| `V_LST_DOC_TRAVEL` | `BrowseLstDocTravel_RD-RD.xml` | **terbaca** |
| `V_LST_DOC_TRAVEL_COVERAGE` | `BrowseDocTravel-Act.xml` | **terbaca** |
| `M_DOCTRAVEL` | `DOCTRAVEL_CVG.prc`, sudah dipakai modul §17 | **terbaca** |
| `M_PLANTRAVEL` | pembenaran peringatan di `SetspreadingtoCoverage-Act.xml` | **terbaca** |
| `LST_DOC_TRAVEL` | diturunkan dari nama view-nya | **DUGAAN** |
| `LST_DOC_TRAVEL_COVERAGE` | diturunkan dari nama view-nya | **DUGAAN** |
| `LST_DOC_TRAVEL_SEQ` | diturunkan dari pola `DOCTRAVEL_SEQ` | **DUGAAN** |

### 22.3 Plan dan Jaminan adalah SATU tabel, bukan dua

Pembacaan pertama menyimpulkan keduanya berasal dari dua sumber berbeda, karena kelas Pega-nya
memang dua: `ASM-FW-GISFW-Int-PLANTRAVEL` dan `ASM-FW-GISFW-Int-COVERAGETRAVEL`. Pengecekan ulang
yang diminta Work Owner membatalkan kesimpulan itu.

`Activity/SetspreadingtoCoverage-Act.xml` menyimpan pembenaran peringatan Pega yang berbunyi apa
adanya — *"ngambil data coverage bukan dari coverage travel tapi dari m_plantravel"* — dan rule yang
membacanya bernama `GetDataMasterCoverageTravel_m_plantravel`. Kedua kelas itu **dua sudut pandang
atas satu tabel**.

Keputusan: keduanya diisi **satu seam** (`PlanRepo` dengan `ListPlans` dan `ListCoverages`), bukan
dua. Dua seam akan menyiratkan dua sumber yang sebenarnya satu, dan orang berikutnya akan mencari
tabel kedua yang tidak pernah ada.

### 22.4 Tiga seam, bukan satu — dan kenapa dua di antaranya baca-saja

| Seam | Tabel | Operasi |
|---|---|---|
| `Repo` | `V_LST_DOC_TRAVEL` + `V_LST_DOC_TRAVEL_COVERAGE` | baca **dan** tulis |
| `DocumentRepo` | `M_DOCTRAVEL` | **hanya `List`** |
| `PlanRepo` | `M_PLANTRAVEL` | **hanya `ListPlans` dan `ListCoverages`** |

Ketiadaan operasi tulis pada dua yang terakhir adalah **penegakan `P-1` dan `D-03` lewat bentuk
antarmuka**, bukan lewat ingatan orang yang menulis kode berikutnya. `M_DOCTRAVEL` dimiliki modul
§17; `M_PLANTRAVEL` dimiliki GISFW. Tidak ada tempat di seam itu untuk menambahkan tulis tanpa
sengaja.

Pola yang sama sudah dipakai `BusinessRepo` pada §18, dan alasannya sama.

### 22.5 Daftar pembatasan plan diganti seluruhnya saat menyimpan

Grid di form mengirim susunan akhir yang dikehendaki petugas. Tidak ada satu pun penanda di sana
yang menyatakan baris mana yang baru, mana yang berubah, dan mana yang dibuang — sehingga
penggantian menyeluruh adalah satu-satunya tafsiran yang tidak menebak.

Teknisnya: `DELETE` seluruh baris coverage milik satu detail, lalu sisip ulang, **di dalam satu
transaksi** bersama perubahan baris induknya.

**`DELETE` di sini tidak melanggar `D-66`.** `D-66` melarang penghapusan fisik data bernilai bisnis.
Baris coverage bukan itu: satu-satunya isinya adalah dua rujukan — plan mana dan jaminan mana — dan
membuangnya tidak menghilangkan keterangan apa pun yang tidak dapat disusun ulang dari susunan
barunya. Baris **induknya** tetap tidak dapat dihapus lewat jalur mana pun.

Hak `DELETE` pada migrasi 0006 karena itu diberikan **hanya** pada tabel coverage, tidak pada
induknya.

### 22.6 Daftar tidak membawa pembatasan plan — dan itu mengikat layar

Kueri daftar tidak membacanya; grid Pega pun lima kolom dan tidak satu pun menyebut plan. Akibat
yang mengikat: layar **wajib memuat ulang** baris saat dibuka untuk disunting.

Memakai baris dari daftar akan membuat form tampak seolah seluruh pembatasannya sudah dihapus, dan
menyimpannya **benar-benar menghapusnya** — karena §22.5 mengganti seluruh daftar. Kewajiban itu
dijaga dua pengujian: satu di backend (`TestDaftarTidakMembawaJaminannya`) dan satu di frontend
(`memuat ULANG baris yang dibuka…`).

Ini cacat yang sama persis dengan yang dicegah §18 pada pemetaan bisnis, dan disebut ulang di sini
karena bentuknya mudah terulang pada master turunan berikutnya.

### 22.7 Nama boleh diketik bebas; kodenya dicarikan dari namanya

`SearchCoverageTravel_RD` ber-`pyAllowFreeFormInput=true`: nama di luar master TETAP boleh diketik
dan tersimpan, tanpa kode. Karena itu:

- isiannya `ComboField`, bukan `SelectField` — dropdown akan menutup kemungkinan itu, dan itu
  **perubahan perilaku**, bukan perbaikan;
- layar mengirim **nama dan kode** sekaligus, bukan kode saja;
- kegagalan memuat daftar pilihan **tidak menghalangi penyimpanan**. Yang hilang hanya kenyamanan
  memilih, dan form itu sendiri yang mengatakannya.

Penyaringan jaminan menurut plan — yang di Pega dikerjakan server lewat parameter `plan` —
dikerjakan **layar**, atas daftar yang sudah di tangan. Menyaring di server berarti satu permintaan
per baris grid setiap kali plannya berganti.

Satu perilaku yang perlu disebut: plan yang diketik bebas tidak cocok dengan satu pun baris master,
dan pada keadaan itu **seluruh** jaminan ditawarkan — bukan daftar kosong. Daftar kosong akan
terbaca sebagai "tidak ada jaminan untuk plan ini", padahal yang benar adalah "plan ini tidak
dikenali master".

### 22.8 Minimal Unggah adalah isian TEKS, bukan isian angka

Isiannya di Pega `pxTextInput`. Saya sempat memakai `type="number"` dan itu keliru dua kali:
peramban menolak ketikan tak-valid sebelum sampai ke kode sehingga penjaganya tidak pernah berjalan,
dan — lebih buruk — peramban **mengosongkan nilainya diam-diam** saat ketikan sementara tidak valid,
sehingga `-2` tersimpan sebagai kosong tanpa satu pun tanda bagi pengguna.

Diperbaiki menjadi `type="text"` dengan `inputMode="numeric"`. Penjaga bentuk angka di skema form
adalah **satu-satunya pemeriksaan isian** di layar ini, dan ia bukan aturan bisnis melainkan bentuk
kolomnya: teks yang bukan angka akan ditolak Oracle sebagai galat yang tidak dapat dibaca pengguna.

### 22.9 `STSWAJIB` adalah angka, dan itu tidak bocor ke kontrak API

`Activity/BrowseDocTravel-Act.xml` membuktikannya lewat precondition langkahnya: `.STSWAJIB==1`
memberi `"Ya"`, `.STSWAJIB==0` memberi `"Tidak"`. Teks itu **tampilan**, bukan yang tersimpan.

Kontrak API mengirimnya sebagai **boolean**. Angkanya bentuk penyimpanan, dan membocorkannya ke
kontrak berarti setiap klien harus mengetahui arti 1 dan 0 — padahal Pega pun sudah menerjemahkannya
sebelum menampilkan. Nilai `'1'`/`'0'` tetap dipakai sebagai nilai dropdown di form, supaya yang
tampil di layar dan yang tersimpan di tabel dapat dibandingkan langsung saat menelusuri masalah.

### 22.10 Penamaan modul

Mengikuti `D-81`: nama modul adalah nama yang disebut Work Owner — **"Daftar Detail Dokumen
Travel"**, sama dengan MENU_DESC pada MENU_ID 39.

| Lapisan | Bentuk |
|---|---|
| Paket Go dan folder backend | `internal/daftardetaildokumentravel` |
| Folder frontend | `src/modules/daftar-detail-dokumen-travel` |
| Rute layar | `/master/daftar-detail-dokumen-travel` |
| Jalur API | `/api/master/daftar-detail-dokumen-travel` |

Jalurnya panjang, dan itu dipilih sadar. Jalur yang lebih pendek akan membuat nama modul, nama
folder, dan jalur API tidak lagi saling menunjuk — dan pada aplikasi yang akan memuat
sekurang-kurangnya 29 master, ketiganya harus dapat ditebak dari satu sama lain. Isi modulnya tetap
berbahasa Inggris sesuai `D-80`.

### 22.11 Yang sengaja TIDAK dikerjakan

| Tidak dikerjakan | Alasan |
|---|---|
| Tombol hapus | Layar Pega tidak punya, dan baris ini menentukan kelengkapan dokumen klaim yang sedang berjalan (`D-66`, `ADR-0012`) |
| Validasi isian | Keputusan Work Owner; layar Pega tidak memuat satu pun `pyRequired` maupun Validate rule |
| Memeriksa DOCID ada di master | Bagian dari "tanpa validasi" — Pega pun tidak memeriksanya sebelum menyimpan |
| Menyalin `pyMaxRecords=500` | Pemotongan senyap, bukan aturan bisnis (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2) |
| Memperbaiki `Field` agar memakai `role="alert"` | Dipakai seluruh modul yang sudah selesai; Isolasi Protektif |
| Memperbaiki 3 uji `master-rekening` yang gagal | Kegagalan pre-existing dari pekerjaan orang lain yang belum di-commit |

### 22.12 Yang tetap menjadi utang

| Utang | Pemilik |
|---|---|
| Definisi kedua view — menentukan **nama tabel dasar** yang ditulis aplikasi | DBA, migrasi 0006 Bagian 1 |
| Nama dan bentuk kolom `LST_DOC_TRAVEL` dan `LST_DOC_TRAVEL_COVERAGE` | DBA |
| Nama urutan penerbit ID, beserta contoh ID yang benar-benar terpakai | DBA |
| Nama kolom `M_PLANTRAVEL` — **tidak memblokir**, hanya daftar pilihannya yang hilang | DBA / GISFW |
| Isi tabel yang sebenarnya; contoh di `sample.go` bukan data produksi | DBA |
| Enam rule yang hilang dari export (`R-16`) | Tim Pega |
| Pemeriksaan peran — "apakah pemanggil memiliki menu Master Data" (`D-59`) | `TKT-F3-005`, terhalang `TKT-F3-004` |
| Awalan `/v1` pada jalur API | `TKT-F1-004`, berlaku seluruh aplikasi |
| Kontrak galat bersama; modul ini memakai kode `detail_dokumen_travel_tidak_ditemukan` sendiri | `TKT-F1-004` |
| `Field` tanpa `role="alert"` sementara `ComboField` memakainya | pustaka komponen bersama |

Satu catatan yang berlaku ke depan: **`MINUNGGAH` hari ini tidak terbaca jalur registrasi klaim.**
`TravelDocument_act` menuliskan `MIN_DOC := 1` — angka tetap, bukan nilai dari baris ini. Modul ini
menyimpannya dengan benar dan **tidak mengubah pembacanya**; bila kelak diputuskan angka itu harus
dipakai, perubahannya ada di jalur registrasi, bukan di sini.

---

## 23. Daftar Objek Dokumen — keputusan yang membentuknya (2026-09-23)

Modul dari rumpun master yang merupakan **saudara langsung** modul §21. Keduanya dirujuk
BERSAMAAN oleh satu tabel yang sama, dan itu terbaca apa adanya di
`Database/PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:26`:

| MENU_ID | Nama | Harness | Tabel | Menjawab |
|---|---|---|---|---|
| 40 | Daftar Tipe Dokumen | `ListDocumentTypeInbox` | `LST_DOC_TYPE` | dokumennya JENISNYA apa |
| **43** | **Daftar Objek Dokumen** | `ListDocumentObject` | **`LST_DOC_OBJ`** | dokumennya **MELEKAT PADA APA** |

Keduanya menjadi `DOCUMENT_TYPE_ID` dan `OBJECT_DOC_ID` pada `POOLDATA.LST_TYPE_DOC_BUSINESS`.
Itulah sebab `ID` sebuah objek dokumen **tidak pernah boleh berubah** dan barisnya tidak pernah
dihapus.

### 23.1 Keputusan Work Owner yang membentuk modul ini

| # | Perkara | Keputusan | Akibatnya |
|---|---|---|---|
| 1 | Lingkup grid "ID Bisnis" di dalam form | **Ikut dibangun penuh** | modul memuat dua tabel dan seam BusinessRepo baca-saja, bukan satu tabel |
| 2 | Jalur simpan, karena activity-nya hilang | **Tulis ke kolom `KET_DOC_OBJ`**, bukan ke `JSON_DATA` | mengikuti `D-02`/`D-68` seperti lima modul master sebelumnya |

### 23.2 Apa yang terbaca, dan apa yang tidak

Keadaan yang sama dengan §22: **APA** yang disimpan terbaca lengkap, **KE MANA** tidak.

Jalur simpan layar lama menunjuk `CNMInsertLstDocObj_act` (tombol Simpan,
`Section/BrowseDocumentObject-Section.xml:18337`) dan `SetsLstDocObjValue_act` (klik baris,
`:7405`). **Keduanya hilang dari export** (`R-16`), dan tidak ada `PEGA_LST_DOC_OBJ.prc` di
`Database/`.

| Objek | Sumber namanya | Status |
|---|---|---|
| `V_LST_DOC_OBJ` | `Report Definition/BrowseVLstDocObj_RD-RD.xml` — kelas `ASM-FW-GCNMFW-Int-V_LST_DOC_OBJ` dengan kolom `ID`, `KET_DOC_OBJ`, `OLD_ID` | **terbaca** |
| `M_SITE_DATABASE` | `Database/PEGA_LST_DOC_TYPE.prc:11` | **terbaca** |
| `BUSINESS` | `RDB List/GetLBUID_SQL-SQL.xml` | **terbaca** |
| `LST_DOC_OBJ` | diturunkan dari nama view-nya, mengikuti pasangan `V_LST_DOC_TYPE` ke `LST_DOC_TYPE` | **DUGAAN** |
| `SET_LST_DOC_OBJ` | diturunkan dari pola `SET_LST_DOC_TYPE` pada procedure yang sama | **DUGAAN** |
| `LST_DOC_OBJ_BUSINESS` | **tabel baru**, dirancang di migrasi 0010 | **DUGAAN** |

Perlakuannya sama dengan §22, ditambah satu hal yang tidak ada di sana:

1. Seluruh nama objek **diisolasi di satu berkas** — `repo/sqlstore/daftarobjekdokumen.sql`.
2. Ketiga dugaan diberi tanda sebagai dugaan, di kepala berkas itu, dengan menyebut dari mana
   masing-masing diturunkan.
3. Migrasi `0010` ditulis **sebagai daftar pertanyaan**; bagian 0-nya sembilan kueri katalog.
4. **Mode `-periksa` menguji KETIGA objek satu per satu**, bukan sekaligus. Itu yang membuktikan
   dugaan nama benar atau salah — dan menyebut MANA yang salah — sebelum pengguna pertama menekan
   Simpan. Pada §22 pemeriksaannya belum sedetail ini.

Dua pertanyaan di migrasi 0010 dirancang untuk **membatalkan** sebagian pekerjaannya sendiri bila
jawabannya mengejutkan: kueri 0.7 mencari tabel mana pun yang sudah memetakan objek dokumen ke
bisnis. Bila hasilnya menemukan tabel yang tidak terduga, langkah 3 — pembuatan tabel baru —
**tidak perlu dijalankan sama sekali**. Layar lama jelas menyimpan pemetaan itu, sehingga tabelnya
kemungkinan memang sudah ada dengan nama yang tidak dapat ditebak.

### 23.3 Tabel pemetaan dirancang dengan dua kolom yang §18 tidak punya

Ini satu-satunya tempat modul ini sengaja **berbeda** dari preseden terdekatnya, dan alasannya
bukan selera:

| | §18 Master COL Simas Online | modul ini |
|---|---|---|
| Tabel pemetaan | **sudah ada**, warisan | **belum ada**, dirancang di sini |
| Kolom urutan | tidak ada | **`URUTAN`** |
| Penanda aktif | tidak ada | **`STS_AKTIF`** |
| Akibatnya | urutan pengguna hilang; nama kembar menyatu; **pencabutan tidak dapat disimpan sama sekali** | urutan terjaga; nama kembar tetap dua baris; pencabutan tersimpan sebagai penandaan |

Ketiadaan `STS_AKTIF` pada §18 tercatat di sana sebagai utang teknis yang menunggu satu kolom dari
DBA. Mewarisinya ke tabel yang bahkan belum dibuat berarti menciptakan utang yang sama secara
sukarela.

Konsekuensinya pada kode: fungsi penyimpanannya bernama **`replaceBusinesses`** — dan di sini nama
itu benar. Pada §18 fungsi yang setara sengaja **bukan** bernama replace, justru karena ia tidak
menggantikan.

### 23.4 `/master/bisnis` dipakai bersama, tidak diduplikasi

Rute daftar bisnis sudah dimiliki §18. Modul ini **tidak mendaftarkannya ulang** — chi akan panik
saat start bila dua modul mendaftarkan jalur yang sama, dan panik itu justru yang membuat
kekeliruan ini mustahil lolos diam-diam.

Yang dikerjakan sebagai gantinya, di ketiga lapisan:

| Lapisan | Perlakuan |
|---|---|
| Backend seam | `BusinessRepo` **milik modul ini sendiri**, bukan tipe milik §18 — modul tidak saling mengimpor |
| Backend rute | tidak didaftarkan; uji `TestRuteBisnisTidakDidaftarkanModulIni` yang menjaganya |
| Frontend | hook sendiri di `api.ts` menembak rute yang sama, dengan **kunci cache yang sama persis** (`['master-bisnis', portal, token]`) sehingga kedua layar berbagi satu entri dan tidak mungkin menampilkan master yang berbeda |

Pemisahan tipe seam-nya mengikuti preseden `travelChoiceSelector` pada §22, yang membaca
`M_DOCTRAVEL` milik modul §17 tanpa mengimpor tipenya.

### 23.5 Yang sengaja TIDAK dibawa dari layar lama

| Isian Pega | Perlakuan | Alasan |
|---|---|---|
| `TempDocObj.pyNote` berlabel **"Catatan"** (`:3097`, `:3160`) | **tidak dibawa** | kotak read-only tempat Pega menaruh kalimat yang dikembalikan procedure lewat `ErrMsg`. `D-68` menetapkan kontrak galat berbasis teks itu tidak dibawa; penggantinya pesan galat ber-`kode` |
| Kolom `OLD_ID` di grid | **dikirim API, tidak ditampilkan** | lapisan data mengikuti Report Definition yang memuatnya; lapisan layar mengikuti section yang tidak. Perlakuan yang sama dengan §24.8 pada Master Penyebab Kerugian |
| Tombol hapus | **tidak ada** | seluruh grid ber-`pyGridDeleteActivityExists=false`, `D-66` melarang penghapusan fisik, dan barisnya dirujuk `LST_TYPE_DOC_BUSINESS.OBJECT_DOC_ID` |

### 23.6 Nama field JSON `objek_dokumen`, bukan `keterangan`

Field JSON mengikuti **label layar**, bukan nama kolom — aturan yang ditetapkan §24.7 setelah label
yang dikarang terbukti menyesatkan. Labelnya di Pega adalah **"Daftar Objek Dokumen"**, baik pada
judul kolom grid (`:6715`) maupun pada isian formnya (`:11801`).

Kata "Daftar" di depan label itu milik **layarnya** — ia daftar objek dokumen — sedangkan nilai satu
barisnya adalah satu objek dokumen. Karena itu fieldnya `objek_dokumen`, bukan
`daftar_objek_dokumen`.

### 23.7 Satu kesalahan sendiri yang ditangkap uji

Data contoh di `repo/memory` mula-mula memakai ID empat karakter (`1001`), yaitu pola rumpun
`M_CAUSE_OF_LOSS` yang nomor urutnya **tiga digit**. Rumpun `LST_*` memakai **empat**
(`PEGA_LST_DOC_TYPE.prc:21`), sehingga ID-nya lima karakter.

`TestIDDiterbitkanPenyimpanan` menyebutkan itu pada percobaan pertama. Yang salah data contohnya,
bukan kodenya — dan uji yang menegaskan bentuk ID secara eksplisit itulah yang menangkapnya. Lebar
nomor urut **harus dibaca dari procedure rumpunnya masing-masing**, tidak pernah disalin dari modul
tetangga.

### 23.8 Yang masih ditunggu, dan dari siapa

| Yang dibutuhkan | Dari | Memblokir? |
|---|---|---|
| Definisi `V_LST_DOC_OBJ` — menentukan **nama tabel dasarnya** | DBA, migrasi 0010 kueri 0.1 | **ya** |
| Apakah `LST_DOC_OBJ` dan `SET_LST_DOC_OBJ` memang bernama itu | DBA, kueri 0.2 dan 0.4 | **ya** |
| Apakah tabel pemetaan objek ke bisnis **sudah ada** dengan nama lain | DBA, kueri 0.7 | **ya** — bila ada, langkah 3 batal |
| Lebar kolom `ID`, `BUSINESS.ID`, `BUSINESS.NOTE` | DBA, kueri 0.3 dan langkah 3 | tidak — batas panjang sementara dipasang 100 |
| `GRANT SELECT` atas `POOLDATA.BUSINESS` | DBA, kueri 0.9 | tidak — isian Bisnis hanya kehilangan sarannya |
| Isi tabel yang sebenarnya; contoh di `memory.go` **bukan** data produksi | DBA | tidak |
| `CNMInsertLstDocObj_act` dan `SetsLstDocObjValue_act` | Tim Pega (`R-16`) | tidak — bentuk simpannya sudah dapat disusun dari form |
| Pemeriksaan peran — "apakah pemanggil memiliki menu Master Data" (`D-59`) | `TKT-F3-005`, terhalang `TKT-F3-004` | tidak |
| Awalan `/v1` pada jalur API | `TKT-F1-004`, berlaku seluruh aplikasi | tidak |

---

## 24. Daftar Tipe Dokumen Bisnis — keputusan yang membentuknya (2026-09-23)

### 24.1 Keputusan yang diambil TANPA jawaban Work Owner

Lima pertanyaan diajukan sebelum kode ditulis dan **tidak dijawab** — Work Owner menjawab
"lanjutkan" dua kali. Kelimanya karena itu diputuskan sendiri **mengikuti preseden yang sudah ada
di repositori ini**, dicatat di sini sebagai asumsi yang dapat dicabut, bukan sebagai kesepakatan.

| # | Perkara | Keputusan | Dasarnya |
|---|---|---|---|
| 1 | Lingkup daftar jaminan | **Dibangun** bersama tabel induknya | Ia yang menentukan wajib-tidaknya dokumen di klaim; tanpanya modul menghasilkan perilaku klaim yang salah |
| 2 | `FLAGTYPES`, `CREDENTIAL`, `DURATION` | **Tidak ditulis** | Procedure lama pun tidak mengisinya (`P-5`); perlakuan sama dengan `OLD_ID` pada modul MENU_ID 40 |
| 3 | Nilai `"-"` pada Detail Dokumen | **Ditiru apa adanya** | `P-5`; mengubahnya menjadi sakelar berarti menebak maksud setiap "-" yang sudah ada |
| 4 | `COVERAGE_DOC_BUSINESS_NONGENERAL` | **Di luar lingkup** | Tidak ditulis satu pun jalur layar ini |
| 5 | Validasi | **Hanya satu**, ditiru dari layar lama | `Activity/InsertDetailTypeDocumentBusiness_act-Act.xml:209` |

Keputusan 1 adalah yang paling mahal untuk dicabut: membatalkannya berarti membuang satu tabel,
satu rute, dan satu bagian layar. Keempat sisanya adalah pilihan "jangan lakukan apa-apa", dan
mencabutnya hanya menambah pekerjaan, tidak membuang pekerjaan yang sudah ada.

### 24.2 Kenapa daftar jaminan ikut dibangun, dan bukan ditunda

Ini keputusan yang paling mudah diambil keliru, dan alasannya bukan kelengkapan fitur.

`STS_WAJIB` **tidak menentukan apa pun sendirian**. Keenam kueri unggah dokumen menghitung jaminan
baris itu lebih dulu, dan nol jaminan berarti "Tidak" — apa pun isi `STS_WAJIB`
(`RDB List/BrowseRegisterCvg-SQL.xml`). Membangun modulnya tanpa jaminan berarti menyerahkan
layar yang **tampak** dapat mewajibkan dokumen tetapi tidak pernah benar-benar mewajibkannya, dan
kekeliruan itu tidak terlihat di layar mana pun sampai sebuah klaim diregistrasi tanpa dokumen
yang seharusnya diminta.

Konsekuensi yang mengikat: karena tidak ada jalur hapus jaminan di sistem lama, layar ini pun
tidak punya — sehingga jaminan yang salah ditambahkan **tidak dapat dibatalkan dari aplikasi**.
Itu dinyatakan terang-terangan di layar, bukan disembunyikan.

### 24.3 Kenapa jaminan DITAMBAHKAN, sementara modul sebelumnya MENGGANTI

Dua modul master turunan dibangun berturut-turut, dan keduanya memperlakukan daftar anaknya
**secara berlawanan**. Itu bukan ketidakkonsistenan:

| Modul | Daftar anak | Perlakuan | Bukti |
|---|---|---|---|
| Daftar Detail Dokumen Travel | plan dan jaminan | **diganti seluruhnya** | grid di form mengirim susunan akhir; tidak ada penanda baris baru/dibuang |
| Daftar Tipe Dokumen Bisnis | jenis klaim | **ditambahkan saja** | procedure hanya menyisipkan bila belum ada (`:55`); **nol DELETE** di seluruh export |

Yang membedakan adalah **bentuk penyimpanannya di sistem lama**, bukan selera. Di modul ini,
penggantian menyeluruh menuntut penghapusan — dan penghapusan jaminan mengubah dokumen yang tadinya
wajib menjadi tidak wajib **pada klaim yang sedang berjalan**.

### 24.4 Kenapa isian rujukan menjadi dropdown, padahal Pega memakai autocomplete bebas

Ketiganya `pyAllowFreeFormInput=true` di Pega, dan modul Daftar Detail Dokumen Travel
mempertahankan kebebasan itu dengan `ComboField`. Di sini tidak, dan sebabnya bentuk datanya:

| Modul | Yang tersimpan | Akibat isian bebas |
|---|---|---|
| Daftar Detail Dokumen Travel | nama **dan** kode | nama tetap tersimpan, hanya tanpa kode — bermakna |
| Daftar Tipe Dokumen Bisnis | **kode saja** | tidak ada yang tersimpan; barisnya menjadi rujukan kosong |

Di Pega, autocomplete menyalin kode ke properti tersembunyi saat sebuah pilihan dipilih; teks yang
diketik bebas sekadar tidak menghasilkan kode. Meniru kebebasan itu di sini berarti menyediakan
isian yang hasilnya **hilang tanpa pesan**. Dropdown menyatakan batasnya terang-terangan.

Nama dokumen — `DETAIL_DOKUMEN` — tetap isian bebas, karena ia memang teks yang tersimpan apa
adanya.

### 24.5 Dua cacat sistem lama yang tidak dapat diputuskan tim pengembang

Berbeda dari cacat yang `D-49` sudah putuskan, kedua ini **mengubah apa yang dibandingkan pada
gerbang 1**, dan keputusannya milik Work Owner:

1. **Jalur "Tambah" tidak menulis apa pun** sebagaimana diekspor — dua precondition yang
   ditambahkan pada 2022-06-03 saling meniadakan. Modul ini membangun perilaku yang
   **dimaksudkan**, bukan yang berjalan, sehingga uji kesetaraan pada jalur ini **akan**
   menunjukkan selisih.
2. **Jalur "Copy" membuka layar kosong** — `Tipe=="COPY"` huruf besar diperiksa terhadap `"copy"`
   yang dikirim tombolnya.

Keduanya dilaporkan apa adanya. Tidak satu pun "diperbaiki diam-diam", dan tidak satu pun
direplikasi sebagai cacat — Copy sekadar **tidak dibangun**, karena bentuk yang benar tidak dapat
dibaca dari rule yang tidak pernah berjalan.

### 24.6 Cacat sistem lama yang TIDAK dibawa

| Cacat | Bukti | Perlakuan |
|---|---|---|
| Kontrak galat berbasis teks — `ErrMsg` berisi kalimat sukses, dan dikembalikan lewat parameter bernama `STS_WAJIB` | `PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:27`, `:42` | Tidak dibawa (`D-68`) |
| Kegagalan `RETURN` diam-diam; pemanggil menerima bentuk yang sama seperti sukses | `:17-19`, `:31-33`, `:45-47` | Menjadi galat yang benar-benar galat |
| `ROLLBACK` atas transaksi milik pemanggil | `:18`, `:32`, `:46` | Transaksi dimiliki Go (`D-68`) |
| Sentinel `'UnknownID'` sebagai penanda "baris baru" | `:11`, `:36` | Hilang bersama pemisahan INSERT dari UPDATE |
| `COMMIT` per baris, sehingga kegagalan di tengah meninggalkan sebagian bisnis terisi | `UpdateDetTypeDocBusiness_SQL` | Satu transaksi untuk seluruh perkalian |
| Perangkaian SQL `{ASIS:}` pada penyaring dan daftar jaminan | `Select_TYPE_DOCUMENT`, enam kueri unggah | Parameter binding tanpa perkecualian |
| Baris yang masternya hilang menghilang dari layar tanpa pesan | `Select_TYPE_DOCUMENT` merangkai empat tabel dengan koma | `LEFT JOIN` |

Yang terakhir layak diperhatikan: ia cacat yang **menyembunyikan aturan yang tetap berlaku**.
Baris yang hilang dari layar master tidak dapat diperbaiki petugas, sementara keenam kueri unggah
membaca tabelnya langsung dan tetap memakainya.

### 24.7 Selisih terencana terhadap Pega

Berbeda dari tiga modul sebelumnya yang selisihnya **nol**, modul ini punya empat — dan ketiganya
yang pertama adalah akibat langsung dari cacat sistem lama, bukan pilihan:

| Selisih | Sebab |
|---|---|
| Jalur Tambah **menyimpan** | di Pega ia tidak menulis apa pun (§24.5) |

> **Dua baris lain dicabut 2026-09-23** setelah Work Owner menjawab "3 hal itu ikutin PEGA aja":
>
> - **Tombol Copy kini ada.** Ia menyalin seluruh aturan sebuah bisnis ke form Tambah tanpa
>   ID-nya — bentuk yang dibaca dari cabang salin `UpdateDetailTypeDocumentBusiness_act`, yang
>   memang tidak pernah mengisi `.ID`.
> - **"Pilih semua" kini mengecualikan kelima kode**, dan tombolnya hanya tampil bagi
>   `pyPosition='NONMBU'`. Kelima kodenya tetap **tidak** ditanam di kode: ia konfigurasi
>   `BISNIS_DIKECUALIKAN_PILIH_SEMUA` yang bawaannya sudah sama dengan Pega, sehingga `D-15`
>   tetap dipatuhi tanpa mengubah perilaku. Gerbang `pyPosition` ditiru sebagai penyembunyian
>   tampilan, bukan kewenangan — di Pega pun ia `pyVisible`, dan menjadikannya izin akan
>   menghidupkan kembali model kewenangan yang `D-59` gantikan.
>
> **Jalur Tambah tetap menyimpan.** Cacatnya sejenis dengan cacat Copy — sebuah syarat yang
> ditambahkan belakangan dan mematikan jalur yang jelas dimaksudkan bekerja. Memperbaiki Copy
> sambil mereplikasi Tambah berarti memperlakukan dua cacat identik dengan dua ukuran berbeda;
> dan menirunya berdua menghasilkan layar master yang tidak dapat membuat satu baris pun.
> Rinciannya di `catatan-pengembangan.md` §23.13, dan keputusan ini mudah dicabut.

> **Dikoreksi 2026-09-23.** Tabel ini semula memuat empat baris. Yang keempat — "baris yang
> masternya hilang tetap tampil" — **dicabut** atas keputusan Work Owner ("seperti aplikasi Pega
> saja"): bentuk join dikembalikan mengikuti kueri lama, sehingga baris yatim kembali tersembunyi
> persis seperti di Pega. Persoalannya tidak hilang, ia berpindah menjadi pertanyaan ke Work Owner
> di migrasi `0010` Bagian 1.
>
> Dua penyimpangan lain ikut dicabut pada kesempatan yang sama dan karena itu **tidak pernah**
> masuk tabel ini: isian rujukan dikembalikan menjadi autocomplete ketik-cari, dan kedua validasi
> tambahan (baris dokumen kosong, jaminan kosong) dihapus sehingga keduanya dilewati diam-diam
> seperti di Pega. Rinciannya di `catatan-pengembangan.md` §23.12.

Yang **tidak** terhitung sebagai selisih karena tidak mengubah data: layar sempit menjadi kartu
(`D-12`), entitas disebut terang-terangan (`R-20`), peringatan "wajib di sini belum berarti wajib
di klaim", dan transaksi atomik (`D-68`).

Judul form **"Update Data"** dan judul layar **"Detail Tipe Dokumen Bisnis"** dipertahankan apa
adanya, sekalipun yang kedua berbeda dari `MENU_DESC` di tabel menu yang berbunyi "Daftar Tipe
Dokumen Bisnis". Keduanya ditiru di tempatnya masing-masing (`D-13`).

### 24.8 Yang tetap menjadi utang

| Utang | Pemilik |
|---|---|
| Bentuk kedua tabel — lebar `ID`, tipe `MIN_DOC` dan `STS_WAJIB` | DBA (migrasi 0010 Bagian 1) |
| Apakah `ID` benar-benar unik | DBA — **memblokir**; penyuntingan bertumpu padanya |
| Posisi `LST_TYPE_DOC_BUSINESS_SEQ` | DBA |
| Siapa mengisi `FLAGTYPES`, `CREDENTIAL`, `DURATION` | Work Owner |
| Arti nilai `"-"` pada `DETAIL_DOKUMEN` | Work Owner |
| Keputusan atas dua cacat §24.5 | Work Owner |
| `DetailCoverageDoc` — bentuk layar penambahan jaminan | Tim Pega (`R-16`) |
| Pemeriksaan peran — "apakah pemanggil memiliki menu Master Data" (`D-59`) | `TKT-F3-005`, terhalang `TKT-F3-004` |
| Awalan `/v1` pada jalur API, dan kontrak galat bersama | `TKT-F1-004`, berlaku seluruh aplikasi |

Satu master **turunan** yang masih belum dibangun dan dirujuk layar ini: **MENU_ID 41 "Daftar
Detail Tipe Dokumen"** (`V_LST_DET_TYPE_DOC`). Modul ini sudah membacanya sebagai daftar pilihan,
sehingga seam-nya tinggal dipakai ulang saat modulnya dibangun.

---

## 25. Daftar Detail Tipe Dokumen — keputusan yang membentuknya (2026-09-23)

Modul `ListDetTypeDocument`, MENU_ID 41. Jalannya pengerjaan ada di
[`catatan-pengembangan.md`](catatan-pengembangan.md) §24; berkas ini mencatat **keputusannya**.

### 25.1 Tiga pertanyaan yang diajukan sebelum satu baris kode ditulis

| # | Pertanyaan | Jawaban Work Owner | Akibatnya pada kode |
|---|---|---|---|
| 1 | Tabelnya hanya `(ID, JSON_DATA)` dan kedua view membongkar JSON itu. Bagaimana modul Go menyimpannya? | **Tidak memakai JSON lagi — tabelnya sudah punya kolom selain ID dan JSON_DATA; baca dan tulis kolom itu** | **Sebelas kolom** ditulis (lihat §25.1a), `JSON_DATA` tidak disentuh. Nama kolomnya diturunkan dari nama kolom view dan diisolasi di satu berkas `.sql`; migrasi 0009 menanyakannya ke DBA |
| 2 | Nilai apa yang disimpan `STS_WAJIB` untuk baris baru — kolomnya memuat empat nilai berbeda di produksi | **`"Ya"` / `"Tidak"`** | Penulisan tegas, pembacaan longgar. Lihat §25.3 |
| 3 | Dari mana daftar pilihan **Objek Dokumen** diambil — masternya belum punya modul | **Endpoint baca-saja di modul ini** | `ReferenceRepo`, seam BACA-SAJA tanpa satu pun operasi tulis. Ternyata modul masternya **sudah dibangun sesi lain**, sehingga yang dibaca adalah tabel yang sama |

### 25.1a Koreksi: dua keterangan TERSIMPAN, bukan hasil join

Rancangan pertama menganggap ketiga keterangan pada view induk hasil join. Pemeriksaan ulang atas
permintaan Work Owner membuktikan **hanya satu** yang begitu.

Yang memutuskan: kolom view dibandingkan dengan isi halaman `TempDTDoc`, yakni halaman yang
diserialisasi menjadi dokumen JSON. Apa yang ada di sana pasti tersimpan; apa yang tidak ada
mustahil tersimpan.

| Kolom view | Ada di `TempDTDoc`? | Kesimpulan | Ditulis aplikasi? |
|---|---|---|---|
| `TYPE_DOCUMENT` | tidak | JOIN ke `V_LST_DOC_TYPE` | tidak |
| `DOC_COL_INFO` | **ya** | **TERSIMPAN** | **ya** |
| `OBJ_DOC_DESC` | **ya** | **TERSIMPAN** | **ya** |
| `NOTE` bisnis (view anak) | ya, tetapi view anak tidak memaparkannya | JOIN ke `BUSINESS` | tidak |

Baris terakhir yang membuat kesimpulannya tidak dapat diambil dari satu sisi saja: **"ada di
halaman" syarat perlu, bukan syarat cukup.** Yang menentukan tetap apa yang dipaparkan view-nya.

Akibat kekeliruan itu bila tidak dikoreksi: kedua isian yang dilihat petugas berisi **keterangan**
dengan `pyAllowFreeFormInput=true`, sehingga keterangan di luar master boleh diketik. Menjoinnya
berarti isian itu tidak punya tempat disimpan — dan apa yang baru diketik hilang saat dimuat ulang,
tanpa satu pun pesan.

### 25.2 Kolom yang ditulis adalah DUGAAN, dan itu dinyatakan terang-terangan

Procedure lama hanya menyebut dua kolom:

    INSERT INTO POOLDATA.LST_DET_TYPE_DOC(ID, JSON_DATA)

Work Owner menyatakan kolomnya sudah ada, tetapi **namanya tidak ada di satu artefak pun** — DDL
tabelnya tidak ikut di export (`R-08`). Yang dipakai karena itu nama kolom **view**-nya, bukti
terkuat yang tersedia.

Tiga hal menjaga dugaan itu tidak berubah menjadi kegagalan senyap:

1. **Seluruh nama objek dan kolom hanya muncul di satu berkas** —
   `repo/sqlstore/daftardetailtipedokumen.sql`. Bila jawaban DBA berbeda, yang disunting hanya
   berkas itu; tidak ada nama tabel yang tercecer di dalam kode Go.
2. **Pemeriksaan BACA dan pemeriksaan TULIS dipisah.** `CheckTable` menguji kedua view,
   `CheckWriteTable` menguji kedua tabel dasar. Bila yang kedua gagal sementara yang pertama
   berhasil, artinya layarnya dapat menampilkan data tetapi **tidak dapat menyimpan** — dan
   perbedaan itu terbaca di mode `-periksa`, sebelum pengguna pertama menekan Simpan.
3. **Migrasi 0009 ditulis sebagai daftar pertanyaan**, bukan DDL yang tinggal dijalankan. Enam
   pertanyaan bernomor, masing-masing dengan kueri katalog yang menjawabnya.

### 25.3 `STS_WAJIB` — penulisan tegas, pembacaan longgar

Bukti dari export, bukan dugaan:

| Pembaca | Nilai yang dikenalinya |
|---|---|
| `Activity/SetTypePDFAdjustment-Act.xml` | **hanya `"Ya"`** |
| `Activity/ValidationUploadDocument_act-Act.xml` | `"Ya"`, `"Tidak"`, `"1"`, `"0"` |
| `Activity/ValidationUploadRegister-Act.xml` | keempatnya |

Bahwa dua pembaca menerima keempat nilai membuktikan data produksinya memang bercampur. Bahwa satu
pembaca hanya mengenali `"Ya"` membuktikan menulis `"1"` akan **merusaknya tanpa galat**.

Keputusannya karena itu dua arah:

- **Menulis** selalu `"Ya"` atau `"Tidak"` — `MandatoryText`.
- **Membaca** menerima `"ya"`, `"1"`, `"y"`, `"true"` sebagai wajib, dan sisanya termasuk NULL
  sebagai tidak wajib — `MandatoryFrom`. Menolak nilai yang tidak dikenal akan menggagalkan
  seluruh daftar karena satu baris warisan.

Kontrak API mengirimnya sebagai **boolean**: teksnya bentuk penyimpanan, dan membocorkannya ke
kontrak berarti setiap klien harus mengetahui keempat nilai itu.

### 25.4 Baris bisnis DIHAPUS saat diganti, bukan ditandai tidak aktif

Ini menyimpang dari modul Daftar Objek Dokumen, yang untuk grid sebentuk memilih penandaan lunak
(`STS_AKTIF = '0'`). Penyimpangannya disengaja, dan sebabnya ada di pembacanya:

`RDB List/GetLbuDetType-SQL.xml` membaca `V_LST_DET_TYPE_DOC_BISNIS` **tanpa satu pun syarat**
selain ID induknya, dan view itu dibaca Pega hari ini juga. Baris yang ditandai tidak aktif karena
itu akan **tetap diberlakukan sebagai aturan dokumen pada klaim** — persis kebalikan dari yang
dimaksud petugas saat membuangnya dari grid, dan tidak terlihat sebagai galat di layar mana pun.

Penandaan lunak di sini lebih berbahaya daripada penghapusan.

Bedanya dengan Daftar Objek Dokumen: tabel di sana **baru**, dirancang di migrasi 0010 lengkap
dengan kolom penandanya, dan tidak ada satu pun pembaca Pega yang melewatinya. Perlakuan yang
dipakai di sini sama dengan baris coverage pada Daftar Detail Dokumen Travel.

`D-66` tidak dilanggar: yang dihapus adalah baris penghubung yang seluruh isinya dapat disusun
ulang dari susunan barunya, dan jejak perubahannya menjadi tanggung jawab `S-5`.

### 25.5 Keempat master rujukan berada di SATU seam, bukan empat

Berbeda dari Daftar Tipe Dokumen Bisnis yang memakai empat selector untuk empat master yang sama
persis. Alasannya: keempatnya dibutuhkan **bersamaan** oleh satu form dan **gagal dengan cara yang
sama** — daftar pilihannya kosong sementara isiannya tetap dapat diketik sendiri dan penyimpanannya
tetap berjalan.

Memisahkannya akan menghasilkan empat selector yang selalu dipilih bersamaan dan empat jalur galat
yang ditangani dengan cara yang persis sama.

Akibatnya di transport: satu rute `/master/detail-tipe-dokumen/pilihan` yang mengembalikan keempat
daftar sekaligus, ditambah senarai `tidak_tersedia` yang menyebut master mana yang gagal dibaca.
Senarai itu ada supaya layar dapat membedakan **"masternya kosong"** dari **"masternya tidak dapat
dibaca"** — dua keadaan yang tampak sama persis di layar tetapi menuntut kalimat yang berbeda.

Rutenya berada di bawah sub-rute modul ini, bukan sejajar seperti pada dua modul sebelumnya, karena
yang dikembalikan bukan salah satu master melainkan **gabungan** keempatnya dalam bentuk yang hanya
berarti bagi form ini. Ia tidak menyiratkan kepemilikan tabel mana pun.

### 25.6 Dua isian bekerja atas KETERANGAN, dua bekerja atas KODE

Pembedaannya dibaca dari section, bukan dari labelnya — dan label di sini justru menyesatkan.

| Isian | Terikat ke | Isinya | Kode disimpan di |
|---|---|---|---|
| ID Tipe Dokumen | `TempDTDoc.DOC_TYPE_ID` | **kode** | isian itu sendiri |
| Dokumen kolom ID | `TempDTDoc.DOC_COL_INFO` | **keterangan** | `DOC_COL_ID` (target tersembunyi) |
| Objek Dokumen | `TempDTDoc.OBJ_DOC_DESC` | **keterangan** | `OBJ_DOC` (target tersembunyi) |
| ID Bisnis (grid) | `.Note` | **nama** | `.ID` (target tersembunyi) |

Label **"Dokumen kolom ID"** menyebut "ID" padahal isiannya keterangan. Labelnya ditiru apa adanya
(`D-13`); yang tidak ditiru adalah kekeliruannya.

**Kenapa dua di antaranya tetap bekerja atas kode di aplikasi ini**, meski di Pega berisi nama:

- **ID Tipe Dokumen** memang berisi kode di Pega juga.
- **ID Bisnis** berisi nama di Pega, tetapi view anaknya **tidak menyimpan namanya** —
  `GetLbuDetType-SQL.xml` menjoin `POOLDATA.BUSINESS`. Nama di luar master karena itu tidak punya
  tempat disimpan, dan membiarkannya diketik bebas akan menghasilkan baris yang hilang saat dimuat
  ulang. Isiannya dibuat bekerja atas kode — satu-satunya bentuk yang tidak kehilangan data.

Kedua isian tengah TIDAK punya persoalan itu: `DOC_COL_INFO` dan `OBJ_DOC_DESC` adalah kolomnya
sendiri, sehingga keterangan bebas tersimpan utuh — persis seperti di Pega.

Keempatnya tetap `ComboField`, bukan dropdown, karena keempatnya autocomplete
ber-`pyAllowFreeFormInput=true` di Pega. Pasangannya ditampilkan sebagai `hint`: pada isian berisi
kode ditampilkan namanya, pada isian berisi keterangan ditampilkan kodenya.

**Satu penyimpangan kecil yang disengaja.** Pega hanya mengisi kode saat sebuah pilihan benar-benar
DIPILIH dari daftar, sehingga keterangan yang diketik ulang persis sama meninggalkan kode LAMA yang
tidak lagi cocok. Di sini kodenya selalu dicocokkan ulang terhadap keterangannya. Perbedaannya tidak
terlihat pengguna, dan yang dihasilkan tidak pernah menyimpang antara kode dan keterangan.

### 25.7 Yang diperiksa, dan yang sengaja TIDAK diperiksa

Layar lama tidak memuat satu pun `pyRequired` bernilai true, tidak satu pun Rule-Obj-Validate, dan
procedure penyimpannya menyisipkan tanpa memeriksa apa pun. `P-5` karena itu berlaku: isian kosong
diterima, rujukan kosong diterima, dan kode yang tidak ada di master pun diterima.

Yang tetap diperiksa, dan keduanya **bentuk kolom** bukan aturan bisnis:

| Yang diperiksa | Sebabnya |
|---|---|
| Panjang keenam isian teks | Lebar kolom sebenarnya belum diketahui (`R-08`); angkanya pilihan, dan migrasi 0009 Q5 menanyakannya. Tanpa penjaga ini, isian yang jelas keliru ditolak Oracle dengan ORA-12899 yang tidak dapat dibaca petugas |
| `Resiko` berbentuk angka | `SetTypePDFAdjustment` membacanya dengan `@toDecimal(.RISK)`. Teks yang bukan angka terbaca **nol** di sana tanpa satu pun tanda |
| `Minimum Dokumen` berbentuk angka bulat | idem, bentuk kolom |

`Resiko` **kosong tetap diterima**: `toDecimal("")` di Pega menghasilkan nol, sehingga baris tanpa
resiko berperilaku sama dengan baris beresiko nol. Menolaknya akan mengubah perilaku layar yang
tidak sedang dimigrasikan.

Yang sengaja TIDAK diperiksa: keberadaan keempat kode rujukan di masternya. Layar lama pun tidak
memeriksanya, dan menolaknya akan menghalangi petugas memperbaiki baris warisan yang rujukannya
memang sudah hilang.

### 25.8 Dua kolom tambahan di grid yang tidak ada di Pega

Grid Pega menampilkan tiga kolom: ID, Tipe Dokumen, dan Detail Dokumen. Di sini ditambahkan
**Objek Dokumen** dan **Penyebab Kerugian**.

Sebabnya bukan hiasan: tanpa keduanya, satu-satunya cara mengetahui rincian dokumen ini melekat
pada objek apa adalah membuka barisnya satu per satu. Keduanya sudah ikut terbaca kueri daftar
karena view induknya memang memaparkannya, sehingga tidak ada kueri tambahan.

Kolom Tipe Dokumen menampilkan **nama beserta kodenya**, dan keduanya dapat dicari dari satu kotak
cari — petugas yang hafal kodenya dan petugas yang hafal namanya sama-sama dilayani.

### 25.9 Yang masih terbuka

Kelima pertanyaan skema yang sempat tercatat di sini **sudah terjawab sendiri** lewat katalog
Oracle pada 2026-09-23 (`catatan-pengembangan.md` §24.10). Yang tersisa bukan pertanyaan
melainkan permintaan perubahan skema:

| Yang tersisa | Kepada |
|---|---|
| **Buat `POOLDATA.LST_DET_TYPE_DOC_BISNIS`**, pindahkan isi JSON, definisikan ulang view anaknya — **memblokir jalur tulis** | DBA (0009 LANGKAH 1) |
| Hak tulis atas tabel anak itu | DBA (0009 LANGKAH 2) |
| Nasib `JSON_DATA` baris lama sesudahnya | Work Owner (0009 LANGKAH 3) |
| Kapan layar Pega MENU_ID 41 dinonaktifkan — `P-1` menuntutnya bersamaan dengan LANGKAH 1c | Work Owner |
| Pemeriksaan peran — "apakah pemanggil memiliki menu Master Data" (`D-59`) | `TKT-F3-005`, terhalang `TKT-F3-004` |
| Awalan `/v1` pada jalur API, dan kontrak galat bersama | `TKT-F1-004`, berlaku seluruh aplikasi |

Dengan modul ini, rumpun tiga master dokumen — MENU_ID 40, 41, dan 42 — **lengkap seluruhnya**.

Catatan "belum dibangun" yang menunjuk modul ini masih tersisa di **dua tempat**, dan dibiarkan
dengan sengaja: `daftartipedokumen.go` adalah modul selesai yang dilindungi Isolasi Protektif, dan
`daftartipedokumenbisnis` beserta blok rakitannya di `main.go` adalah pekerjaan sesi lain yang
belum di-commit. Keduanya dicatat di `catatan-pengembangan.md` §24 supaya pembaruannya dapat
dikerjakan bersamaan saat kedua berkas itu memang sedang disentuh.

### 25.10 Tanpa JSON sama sekali — keputusan Work Owner 2026-09-23

> "pakai database saja udah ga pakai json lagi ya"

Keputusan ini menutup pertanyaan terbuka §25.9 tentang nasib `JSON_DATA`, dan mengubah satu hal
yang belum tersesuaikan: **jalur baca daftar lini bisnis**.

#### Yang berlaku sekarang

| Hal | Ketetapan |
|---|---|
| Baris induk | dibaca dari `V_LST_DET_TYPE_DOC` (yang memang SELECT kolom), ditulis ke 11 kolom `LST_DET_TYPE_DOC` |
| Daftar lini bisnis | dibaca **dan** ditulis dari `LST_DET_TYPE_DOC_BISNIS` — tabel, bukan view |
| `JSON_DATA` | tidak dibaca, tidak ditulis, **tidak dikosongkan** |

#### Kenapa daftar bisnis dibaca dari TABEL, bukan dari view anaknya

Karena view anaknya hari ini adalah `JSON_TABLE` atas `JSON_DATA`. Membacanya berarti tetap
membaca JSON — persis yang diputuskan berhenti.

Sesudah migrasi 0009 LANGKAH 1c view itu memang akan membaca tabel yang sama. Tetapi view
dimiliki pihak lain sementara modul ini **menulis** tabelnya sendiri, sehingga membaca dari tabel
dasar membuat apa yang ditulis dan apa yang dibaca kembali PASTI sama. Alasan dan preseden yang
sama dipakai modul Master Penyebab Kerugian.

**Akibat yang diterima:** sampai DBA menjalankan LANGKAH 1, membuka satu baris dan menyimpan
gagal dengan `ORA-00942`. Daftarnya tetap dapat dimuat — 159 baris terbaca pada 2026-09-23.
Gagal terang-terangan dipilih daripada menampilkan daftar bisnis KOSONG untuk baris yang
sebenarnya masih punya isi di JSON; yang kedua akan terbaca petugas sebagai "aturannya memang
belum ada", lalu tersimpan sebagai benar-benar kosong.

#### `JSON_DATA` dibiarkan, bukan dikosongkan

"Tidak pakai JSON lagi" dibaca sebagai **berhenti memakainya**, bukan **menghapusnya**:

- `D-66` melarang penghapusan fisik data bernilai bisnis.
- LANGKAH 1b **menyalin** isinya ke tabel anak, bukan memindahkan, sehingga bila salinannya
  ternyata tidak lengkap sumbernya masih utuh.
- Menghapus kolomnya adalah perubahan tak terbalikkan pada tabel yang dibaca puluhan rule Pega,
  dan tidak ada yang menuntutnya.

Risiko yang diterima: seseorang kelak membaca `JSON_DATA` dan mengira itu keadaan sekarang.
Penangkalnya catatan di kepala berkas `.sql` dan di migrasi — bukan penghapusan.

#### Satu hal yang TETAP wajib meski modul ini tidak lagi memakainya

**View anaknya tetap harus didefinisikan ulang** (LANGKAH 1c) — bukan untuk aplikasi Go,
melainkan untuk **sembilan rule Pega** yang masih membacanya, di antaranya
`Section/ViewUploadDocument-Section.xml` dan `Activity/InsertDocumentPendukungPA_-Act.xml`.
Tanpa itu, baris yang disimpan aplikasi ini tidak akan pernah terlihat Pega.

#### Yang masih menunggu Work Owner

Kapan layar Pega MENU_ID 41 dinonaktifkan. `UpdateDetTypeDoc-SQL` masih menulis `JSON_DATA`,
bukan kolom — sehingga selama layar itu hidup, baris yang disimpannya tampil KOSONG di view
induk dan lini bisnisnya tidak pernah masuk ke tabel anak. `P-1` menuntut keduanya terjadi
bersamaan dengan LANGKAH 1, bukan berurutan.

---

## 26. Inbox Compliance — keputusan yang membentuknya (2026-09-24)

Menu `MENU_ID 47`, harness `inboxCompliance_Harness`. Modul antrean kerja kedua setelah Inbox
Admin, dan perbedaannya dari modul itu bukan kebetulan — beberapa di antaranya disengaja dan
dicatat di sini supaya tidak dibaca sebagai ketidakkonsistenan.

### 26.1 Keputusan Work Owner pada sesi ini

| # | Pertanyaan | Jawaban | Akibatnya pada kode |
|---|---|---|---|
| 1 | Sumber data tab Post Audit | **`POOLDATA.T_CLAIM_COMPLIANCE_H`**, bukan tabel DATAPEGA | Menggugurkan rencana membaca `PC_ASM_FW_GCNMFW_WORK` dengan `PXOBJCLASS = 'ASM-FW-GCNMFW-Work-Compliance'` |
| 2 | Section pembungkus yang hilang | **Akan ditambahkan** | Berkasnya tiba dan dibaca; ternyata pembungkus tab tanpa field |
| 3 | Perhitungan Aging | **Tulis ulang di Go, potong akhir pekan** | `aging.go` — `D-02` dan `D-50` |
| 4 | Daftar kolom `T_CLAIM_COMPLIANCE_H` | **Sudah ditambahkan** — tetapi TIDAK ditemukan | Tab Post Audit dibangun sebagai tab yang menyatakan penghalangnya |

### 26.2 Lima perbedaan sengaja dari modul Inbox Admin

Keduanya sama-sama layar antrean kerja, dan konsistensi antarmodul diminta secara eksplisit.
Kelima perbedaan berikut tetap diambil, masing-masing dengan alasannya.

| Hal | Inbox Admin | Inbox Compliance | Alasan |
|---|---|---|---|
| **Paginasi** | seluruh baris ditarik, dipotong di aplikasi | dipotong di **basis data** | Pemotongan di aplikasi pada Inbox Admin adalah keputusan Work Owner 2026-09-20 yang berlaku **khusus layar itu**. Tidak ada keputusan serupa di sini, sehingga yang berlaku adalah standar koding: paginasi selalu server-side |
| **Kode tab** | angka (`3`, `7`, `9`, `11`) | kata (`compliance`, `post-audit`) | Inbox Admin mempertahankan angka karena angka itu muncul di prakondisi 34 langkah activity — jalan telusur balik ke export. Layar ini **tidak punya properti pemilih tab sama sekali**; tabnya dipilih peramban. Tidak ada angka yang perlu dipertahankan |
| **Jembatan Caller** | ada — tiga tab menyaring menurut login | **tidak ada** | Antreannya WORKBASKET, yakni antrean bersama. Identitas tidak ikut menentukan baris mana yang tampil, sehingga jembatan dari modul auth hanya akan menjadi ketergantungan yang tidak dipakai |
| **Penyaring** | kotak cari + dropdown lini bisnis | **tidak ada sama sekali** | `InputCompliance_Section` tidak memuat satu pun field masukan. Menambahkannya berarti mengarang kemampuan yang tidak pernah ada |
| **Format tanggal DTO** | `value.UTC()` | `value.In(clock.ZoneWIB)` | Yang dikirim adalah TANGGAL tanpa jam, dan tanggal hanya berarti setelah zonanya ditetapkan. Klaim yang masuk pukul 06.00 WIB tanggal 2 adalah pukul 23.00 UTC tanggal 1 — memformatnya UTC menampilkan tanggal kemarin |

Perbedaan terakhir membuat kedua modul menampilkan tanggal dengan cara yang berbeda. Yang benar
adalah yang di sini (`F-5`); perbedaannya dicatat sebagai temuan atas modul Inbox Admin, **tidak**
diperbaiki di sini — Isolasi Protektif melarang menyentuh modul yang sudah selesai.

### 26.3 Kolom Aging — apa yang direplikasi dan apa yang diperbaiki

**Direplikasi apa adanya** (`P-5`):

- Pemotongan hanya hari Sabtu dan Minggu — **bukan** hari libur nasional, **bukan** jam kerja.
- Bentuk teksnya: `"N hours ago"` di bawah 24 jam, `"N days M hours ago"` di atasnya.
- Jam **dipotong**, bukan dibulatkan: 23,9 jam tampil `"23 hours ago"`.
- Bentuk jamak yang salah tidak diperbaiki: satu jam tetap `"1 hours ago"` (`D-13`).

**Diperbaiki, dan sudah diputuskan sebelumnya:**

`Database/GETSELISIHJAM.fnc:22` mengembalikan `0` pada setiap kegagalan, sehingga kegagalan tidak
dapat dibedakan dari klaim yang baru masuk antrean. `D-49` butir 10 memutuskan cacat itu
diperbaiki, dan ia tercatat sebagai butir ke-13 daftar perbaikan eksplisit `P-5`. Di sini: Aging
yang tidak dapat dihitung bernilai `null` dan tampil sebagai tanda pisah — **bukan** `0`.

**Tidak direplikasi, dan ini penyesuaian yang perlu disebut:**

Activity lama menambahkan tujuh jam secara manual ke `TanggalBuatCompliance` sebelum
membandingkannya dengan `SYSDATE`. Penambahan itu adalah utang teknis yang `F-5` hapus. Yang
diambil hanyalah AKIBAT yang benar darinya — bahwa batas harinya tengah malam WIB — lewat
`clock.DateWIB`. Bila batas hari UTC yang dipakai, jumlah hari akhir pekan dapat meleset satu,
yakni 24 jam pada kolom Aging.

### 26.4 Nama workbasket sebagai konstanta, bukan konfigurasi

`WorkbasketCompliance = "CompliancePNC"`, dibaca dari `Flow/Register_Flow.xml:3769`.

`D-15` melarang nilai bisnis di-hardcode. Yang dilarangnya adalah nilai yang berubah menurut
**kebijakan** — ambang uang, penerima notifikasi, pemetaan peran. Nama workbasket bukan salah
satunya: ia identitas antrean yang menjadi bagian dari bentuk alur kerja, dan mengubahnya berarti
mengubah flow-nya. Preseden yang sama sudah ada di kueri Inbox Admin (`'RCLPUCL'`).

Satu hal tetap dicatat sebagai utang: di Pega nilainya sampai ke Report Definition lewat
`Param.Operator`, yakni PARAMETER — sehingga secara mekanisme ia memang dapat berbeda per
pemanggil. Section yang menyetelnya tidak memuat pilihan bagi pengguna, jadi satu nilai tetap
adalah pembacaan yang benar hari ini.

Berbeda dari Inbox Admin, nilainya **tidak ditulis di dalam teks SQL** melainkan dikirim sebagai
bind. Ada uji yang menjaganya (`TestWorkbasketIsBoundNotInlined`).

### 26.5 Sumber kolom Tanggal Kirim Compliance

Report Definition membacanya dari halaman kerja sebagai `.ClaimData.TanggalBuatCompliance`. Kode
ini membacanya dari **tabel datar** `POOLDATA.T_CLAIM_PNC.COMPLIANCE_CREATEDATE`.

Alasannya satu: kolom itu **terbukti ada** — ia ditulis
`Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:513` dari kunci JSON `TanggalBuatCompliance` (`:419`).
Nama kolomnya di tabel kerja Pega hanya dapat **ditebak** (`TANGGALBUATCOMPLIANCE_1`, mengikuti
pola `STATUSCLAIM_1`), dan DDL-nya belum ada (`R-08`). Menebak berarti kueri yang baru gagal saat
menyentuh Oracle nyata.

Join-nya **LEFT**, bukan INNER: tabel datar diisi prosedur konversi yang berjalan terpisah,
sehingga klaim yang baru masuk antrean dapat belum punya barisnya. INNER JOIN akan
**menghilangkan pekerjaan dari antrean** hanya karena konversinya tertinggal — kelas cacat yang
jauh lebih berbahaya daripada kolom tanggal yang kosong.

### 26.6 Kolom yang keberadaannya masih disimpulkan

Dua kolom dipakai kueri daftar tanpa bukti DDL:

| Kolom | Dasar |
|---|---|
| `PC_ASM_FW_GCNMFW_WORK.PYORIGUSERID` | Report Definition memberi label "Nama Admin" tepat pada `.pyOrigUserID`; ia properti kerja bawaan Pega |
| `T_CLAIM_PNC.COMPLIANCE_CREATEDATE` | Terbukti dari `INSERT` di prosedur konversi — ini yang **paling kuat** dari keduanya |

Penangkalnya bukan komentar, melainkan **perintah `-periksa`**: entri Inbox Compliance di
`check.go` menjalankan kueri daftar yang sesungguhnya, bukan sekadar menyentuh tabelnya. Bila salah
satu kolom bernama lain, yang menemukannya adalah perintah itu saat aplikasi start — bukan petugas
Compliance saat membuka layar.

### 26.7 Batas 500 baris tidak dibawa

Kedua Report Definition memasang `pyMaxRecords=500`, sehingga antrean yang lebih panjang
**terpotong tanpa pemberitahuan** di Pega. Sistem baru memaginasinya, sehingga seluruh barisnya
terjangkau.

Ini penambahan kemampuan yang sudah diperkirakan `ADR-0011`, dan **wajib dinyatakan di muka sebagai
selisih terencana pada gerbang 1** — bukan ditemukan sebagai kejutan saat uji kesetaraan.

### 26.8 Tab Post Audit — dibangun sebagai tab yang menyatakan penghalangnya

Tab itu **tidak disembunyikan**, dan itu keputusan. Menyembunyikannya membuat pengguna yang
mencarinya menduga modulnya belum selesai — sementara yang sebenarnya kurang adalah satu artefak
yang pemiliknya jelas.

| Lapisan | Perlakuan |
|---|---|
| Domain | `Tab.Available = false` + `Tab.Blocker` yang menyebut artefaknya |
| Query | `NewQuery` menolak dengan `TabNotReadyError`, **bukan** `ValidationError` |
| HTTP | **503**, bukan 404 dan bukan 422 |
| Layar | Tab tetap digambar, bertanda, dan `aria-disabled`; isinya panel penjelas, bukan tabel kosong |

Pilihan 503 disengaja: tabnya ADA dan permintaannya benar — yang belum ada adalah artefak dari
pihak lain, dan keadaan itu akan berubah tanpa pengguna melakukan apa pun. 404 akan membuat
pengguna mengira tabnya tidak pernah ada; 422 akan membuatnya mencari isian yang salah pada layar
yang tidak punya isian.

Bentuk kolomnya **sudah disusun** dari label Report Definition, sehingga begitu daftar kolom tiba
yang perlu ditambah hanya satu kueri dan satu pemindai.

### 26.9 Kolom Aging tidak dapat diurutkan

`DataTable` mengurutkan berdasarkan teks yang dikembalikan `value`. Isi kolom ini berbentuk
`"2 days 3 hours ago"` — mengurutkannya sebagai teks menaruh `"10 hours ago"` sebelum
`"2 days ago"`, urutan yang tampak masuk akal sampai seseorang mengandalkannya untuk mencari
pekerjaan yang paling lama menunggu.

Angka mentahnya sudah dikirim sebagai `aging_jam`; yang belum ada adalah kemampuan `DataTable`
mengurutkan dengan nilai selain teks yang ditampilkan. Itu lingkup `TKT-U2-001` dan **tidak**
diselesaikan sepihak di satu layar. Sementara itu barisnya sudah datang terurut dari server.

### 26.10 Yang sengaja TIDAK dibangun

| Hal | Alasan |
|---|---|
| Kotak cari dan penyaring | Tidak ada di sistem lama; `InputCompliance_Section` nol field masukan |
| Aksi mengubah data | Layar ini hanya membaca antrean. Pemeriksaan kepatuhan, penerusan ke Post Audit, dan penutupan klaim terjadi di layar lain |
| Pemeriksaan peran | `TKT-F3-005`, bergantung pada tabel peran yang **dapat dibangun tetapi belum dapat diisi** (`11-SECURITY.md` §3.1). Di modul ini akibatnya lebih berat daripada di modul master: antrean ini di Pega hanya dapat dibuka `GCNMFW:PncComplience`, dan barisnya memuat nomor polis serta nama tertanggung |
| Migrasi basis data | Modul hanya membaca tabel warisan; tidak ada objek baru |
| Lencana jumlah per tab | Sistem lama tidak punya, dan menghadirkannya berarti menjalankan kueri tab yang belum punya kueri |

---

## 27. Inbox Compliance tab Post Audit — keputusan yang membentuknya (2026-09-24)

Melanjutkan §26. DDL `POOLDATA.T_CLAIM_COMPLIANCE_H` diterima pada hari yang sama, sehingga
penghalang yang dicatat di §26.8 gugur dan tab-nya dihidupkan.

```
CASEID                VARCHAR2(100)
NO_KLAIM              VARCHAR2(1000)
NAMA_TERTANGGUNG      VARCHAR2(4000)
NO_POLIS              VARCHAR2(4000)
REMARKS               VARCHAR2(4000)
TGL_KIRIM_POST_AUDIT  DATE
```

Enam kolom, **tanpa primary key, tanpa constraint unik, tanpa satu pun `NOT NULL`**. Ketiga
ketiadaan itu masing-masing punya akibat, dan ketiganya ditangani — bukan diabaikan.

### 27.1 Tiga properti Report Definition yang tidak punya padanan

| Properti | Dipakai RD untuk | Akibat di sini |
|---|---|---|
| `.pyStatusWork` | penyaring `= "New"` | **Penyaringnya tidak dapat dibawa** — lihat §27.2 |
| `.pxCreateDateTime` | kunci urut kedua | Urutan memakai `TGL_KIRIM_POST_AUDIT` |
| `.pyOrigUserID` | kolom "Originating User ID" | Tidak ditampilkan; RD pun tidak menampilkannya di grid |

### 27.2 Penyaring status tidak dapat direplikasi — dan itu terlihat pengguna

`InboxCompliance_RD` menyaring `.pyStatusWork = "New"`. Tabel barunya **tidak punya kolom status
sama sekali**, sehingga penyaring itu tidak dapat ditulis.

Pembacaan yang dipakai: tabel ini memang sudah berisi apa yang hendak ditampilkan — barisnya lahir
ketika Post Audit dikirim, dan akhiran `_H` beserta kolom `TGL_KIRIM_POST_AUDIT` menguatkannya.
Menambahkan penyaring buatan di atasnya berarti mengarang aturan yang tidak ada sumbernya.

**Yang tidak dilakukan:** menebak kolom status yang mungkin ada, atau menyaring dengan aturan
karangan seperti "hanya 90 hari terakhir".

**Yang dilakukan:** keterbatasannya ditulis sebagai kalimat yang **tampil di layar**, bukan hanya
di dokumen —

> "Tab Post Audit menampilkan seluruh baris POOLDATA.T_CLAIM_COMPLIANCE_H. Sistem lama
> menyaringnya ke pemeriksaan yang berstatus baru; tabel itu tidak menyimpan status, sehingga
> penyaringnya tidak dapat dibawa. Bila daftar ini terasa lebih panjang daripada di Pega, itu
> sebabnya."

Alasannya: ini bukan keterbatasan perkakas melainkan **pertanyaan terbuka yang akibatnya terlihat
pengguna**. Petugas yang membandingkannya dengan Pega akan melihat selisih jumlah baris, dan tanpa
kalimat itu ia akan melaporkannya sebagai kerusakan.

**Pertanyaan untuk Work Owner:** apakah tabel ini memang hanya berisi yang belum ditindaklanjuti,
atau seluruh riwayat? Bila yang kedua, dibutuhkan penyaring — dan sumbernya harus ditetapkan.

### 27.3 Urutan diberi dua pemutus, bukan satu

```sql
ORDER BY h.TGL_KIRIM_POST_AUDIT DESC NULLS LAST,
         h.NO_KLAIM DESC NULLS LAST,
         h.CASEID DESC NULLS LAST
```

Tabelnya tidak punya kunci unik, sehingga baris kembar mungkin ada. Urutan yang tidak tetap pada
`OFFSET … FETCH` berakibat nyata: **satu baris muncul di dua halaman sekaligus sementara baris lain
hilang sama sekali** — dan hilangnya tidak menimbulkan galat apa pun.

`NULLS LAST` dipakai karena seluruh kolom boleh NULL. Ia sah di Oracle maupun PostgreSQL, sehingga
tidak melanggar `D-20`.

### 27.4 Dua tabel berbeda berarti dua daftar alias, bukan satu yang dipakai bersama

Modul Inbox Admin memakai SATU daftar 29 alias untuk ketujuh kuerinya, dengan `CAST(NULL AS …)` di
kolom yang tidak berlaku. Pola itu **tidak diikuti** di sini.

Alasannya: di sana ketujuh kueri membaca satu keluarga tabel yang sama, sedangkan di sini kedua tab
membaca **tabel yang benar-benar berbeda** — warisan Pega versus tabel datar baru. Memaksakan satu
daftar bersama hanya akan menambah enam `CAST(NULL AS …)` pada dua kueri tanpa satu pun manfaat,
dan menyamarkan bahwa keduanya memang tidak sekerabat.

Gantinya: `complianceColumns` dan `postAuditColumns`, masing-masing dengan pemindainya sendiri, dan
`plans` yang mengikat kueri-argumen-pemindai dalam satu nilai supaya ketiganya tidak dapat berubah
sendiri-sendiri. Ada uji yang menjaga setiap tab punya rencana dan setiap rencana dimiliki tab
(`TestEveryTabHasPlan`).

### 27.5 Pemeriksaan tabel dipecah dua, dan pesannya pun dipecah

`CheckTable` sekarang menjalankan dua kueri. Bukan kerapian:

| Kelompok | Kegagalannya hampir selalu berarti |
|---|---|
| `PC_ASM_FW_GCNMFW_WORK`, `PC_ASSIGN_WORKBASKET`, `T_CLAIM_PNC` | **grant belum diberikan** — ketiganya tabel warisan yang sudah ada |
| `T_CLAIM_COMPLIANCE_H` | **tabelnya belum ada di entitas itu** — ia tabel baru |

Menyatukan keduanya membuat penelusurannya menempuh satu putaran lebih panjang. Ini juga yang
menjadikan `-periksa` berguna pada aplikasi empat portal: tabel baru itu mungkin sudah dibuat di
satu entitas dan belum di entitas lain.

### 27.6 Mekanisme "tab belum siap" TIDAK dibuang

Penghalang Post Audit hilang, tetapi `TabNotReadyError`, `Tab.Available`, `Tab.Blocker`, jawaban
503, dan panel penjelas di layar **tetap ada** — beserta ujinya, yang kini memakai tab buatan.

Alasannya: modul ini masih akan menumbuhkan tab baru, dan yang membedakan "tab tidak dikenal" dari
"tab menunggu artefak pihak lain" adalah mekanisme itu. Membuangnya sekarang berarti menulisnya
lagi dari nol saat dibutuhkan — dan sementara itu pengguna akan menerima "tab tidak dikenal" untuk
tab yang sebenarnya ada.

### 27.7 Yang TIDAK perlu diubah, dan itu buktinya rancangannya benar

**Frontend: nol perubahan pada kode layar.** `InboxCompliancePage.tsx`, `ComplianceTabs.tsx`,
`api.ts`, dan `types.ts` tidak disentuh sama sekali. Tab-nya hidup semata karena
`Tab.Available` dibalik di backend dan kolomnya datang dari metadata.

Itu tepat seperti yang dijanjikan §26.8, dan ia membuktikan keputusan "bentuk layar datang dari
server" di §26 memang membayar dirinya. Yang berubah di frontend hanya **berkas ujinya** — dan itu
pun karena uji lama menyatakan Post Audit terhalang, pernyataan yang sudah tidak benar.

### 27.8 Yang masih terbuka

| # | Pertanyaan | Pemilik |
|---|---|---|
| 1 | Apakah `T_CLAIM_COMPLIANCE_H` berisi seluruh riwayat atau hanya yang belum ditindaklanjuti? Menentukan perlu-tidaknya penyaring | **Work Owner** |
| 2 | Benarkah `NO_KLAIM` yang ditampilkan dan `CASEID` kunci teknisnya? Pembacaannya kuat, tetapi tidak ada satu pun rule di export yang membaca tabel ini sebagai pembanding | **Work Owner / DBA** |
| 3 | Siapa yang MENGISI tabel ini? Tidak ada satu pun penulisnya di export — bukan rule Pega, bukan prosedur. Selama pengisinya tidak diketahui, `P-1` (satu penulis per tabel) belum dapat dinyatakan terpenuhi | **DBA / Tim Pega** |

Butir 3 yang paling perlu dijawab sebelum masa paralel: modul ini hanya MEMBACA, sehingga tidak
melanggar `P-1` — tetapi siapa penulisnya tetap harus diketahui supaya kepemilikan tabelnya jelas.

---

## 28. Inbox Compliance tab Post Audit — koreksi dari layar Pega (2026-09-24)

Melanjutkan §27. Tangkapan layar tab Post Audit di Pega yang berjalan membetulkan tiga hal yang
diputuskan di §27, dan menambah satu keputusan baru.

### 28.1 Tiga koreksi

| # | Diputuskan §27 | Terbukti salah | Yang benar |
|---|---|---|---|
| 1 | `NO_KLAIM` ditampilkan sebagai "Case ID", `CASEID` disembunyikan | Pega menampilkan `CPL-19` di "Nomor Case" dan `ASM-FW-GCNMFW-WORK PNC-2114` di "No Klaim" | `CASEID` → Nomor Case, `NO_KLAIM` → No Klaim, **keduanya ditampilkan** |
| 2 | Lima kolom, judul dari Report Definition | Section punya tujuh `pyCaption` | Tujuh kolom, judul dari section |
| 3 | Tanggal tanpa jam | Pega menampilkan `22/04/25 13:46` | Tanggal berikut jam, satu kolom saja |

Ketiganya punya sebab yang sama: **label Report Definition dipakai sebagai pengganti label
section**, karena pencarian `pyLabelFieldValue` di section mengembalikan nol hasil. Elemennya
memang kosong di section itu; judulnya tersimpan sebagai `pyCaption`.

Aturan yang ditarik darinya, dan ia berlaku untuk modul berikutnya: **pencarian berbasis nama
elemen yang mengembalikan nol hasil adalah sinyal elemennya salah, bukan sinyal datanya tidak
ada.** Pencarian teks biasa dipakai sebagai pemeriksa silang.

### 28.2 Nama kolom yang menyesatkan TIDAK diperbaiki

Kolom berjudul "No Klaim" berisi kunci teknis Pega, bukan nomor klaim. Itu persis kelas cacat yang
`03-CURRENT-ARCHITECTURE.md` §4.2 sebut sebagai utang teknis — nama yang tidak mencerminkan isi.

Ia tetap **ditiru apa adanya**, karena `D-13` menetapkan tampilan mengikuti Pega supaya pengguna
tidak perlu belajar ulang. Yang tidak ditiru adalah penamaan di dalam kode: field-nya bernama
`ClaimNumber` dengan komentar yang menyatakan isinya bukan nomor klaim, dan ada uji yang mematok
kolom itu memang berisi kunci teknis — supaya tidak ada yang "memperbaikinya" tanpa keputusan.

Bila kelak diputuskan judulnya diperbaiki, tempatnya satu: `tab.go`.

### 28.3 OutStanding adalah waktu kalender, dan itu dibuktikan dari angkanya

Kolom ketujuh berbunyi `1 year 5 months ago`. Ia BUKAN kolom Aging dengan nama lain, dan
pembedaannya tidak perlu dikira-kira:

| Cara hitung | 22 Apr 2025 → 24 Sep 2026 | Bunyinya |
|---|---|---|
| Kalender | 520 hari = 1 tahun 5 bulan 2 hari | "1 year 5 months ago" — **cocok dengan Pega** |
| Potong akhir pekan (`GETSELISIHJAM`) | ±371 hari | "1 year 0 months ago" — tidak cocok |

Karena itu `FormatElapsed` dibuat terpisah dari `FormatAging`, dan `WithAging` diganti namanya
menjadi `WithElapsed` karena kini mengisi dua kolom dengan dasar yang berbeda.

Tahun dan bulan dihitung secara **kalender**, bukan dengan membagi selisih detik: membaginya
memakai "bulan" sepanjang 30 hari yang tidak ada di kalender mana pun, dan melesetnya bertambah
seiring rentangnya memanjang.

**Yang direkonstruksi.** Hanya cabang tahun yang teramati. Rule penyusunnya tidak ada di export,
dan section menaruh hasilnya di `.pyNote` — properti tampungan, persis seperti
`.ClaimData.AgingKlaim` pada tab sebelah. Cabang bulan, hari, jam, dan menit adalah rekonstruksi
dari bentuk baku pemformat waktu relatif Pega, dan statusnya ditulis terbuka di komentar fungsinya
serta di nama kasus ujinya.

### 28.4 Dua pemformat tanggal dalam satu modul

`toDateString` (`YYYY-MM-DD`) dan `toDateTimeString` (`YYYY-MM-DD HH:MM`). Yang kedua dipakai satu
kolom saja.

Ini ketidakseragaman yang **disengaja**, bukan yang terlewat: layar Pega memang menampilkan jam di
kolom itu dan tidak di kolom lain. Menyeragamkannya berarti menghapus informasi yang selama ini
dibaca pengguna, atau menambah jam pada enam kolom yang tidak pernah punya.

Layar menempelkan jamnya setelah memformat tanggalnya lewat fungsi bersama `formatDate`, sehingga
bentuk tanggalnya tetap seragam dengan seluruh layar lain. Ada uji yang mematok hasilnya
(`18 September 2026 13:46`), supaya penyeragaman pemformatan kelak tidak membuang jamnya tanpa ada
yang menyadarinya.

### 28.5 Kenapa layar baru kosong — dan kenapa itu bukan cacat

Dugaan yang diajukan Work Owner — "Pega membaca JSON, yang baru langsung ke tabel" — diperiksa dan
**tidak terbukti**: nol kemunculan `JSON_VALUE`, `JSON_TABLE`, `JSON_QUERY`, `JSON_KLAIM`, maupun
`DATA_JSONBLOB` pada Report Definition dan section Post Audit. Pega membaca tabel kerja Pega, bukan
JSON.

Sebab sebenarnya: `POOLDATA.T_CLAIM_COMPLIANCE_H` **belum berisi**. DDL-nya diawali
`DROP TABLE … CASCADE CONSTRAINTS` lalu `CREATE TABLE`, dan tidak ada satu pun penulis di export.

Konsekuensi yang perlu diputuskan, bukan diasumsikan: **selama tidak ada yang mengisi tabel itu,
tab Post Audit akan selalu kosong.** Modul ini hanya membaca — menambahkan penulisnya berarti
menulis ke tabel yang pemiliknya belum ditetapkan, dan itu menyentuh `P-1`.

### 28.6 Yang masih terbuka

| # | Pertanyaan | Pemilik | Berubah dari §27 |
|---|---|---|---|
| 1 | Siapa yang MENGISI `T_CLAIM_COMPLIANCE_H` | **DBA / Tim Pega** | **naik menjadi mendesak** — tabelnya kosong |
| 2 | Seluruh riwayat atau hanya yang belum ditindaklanjuti | **Work Owner** | tetap |
| 3 | Bentuk OutStanding untuk rentang pendek — "3 days ago"? "2 months ago"? | **Work Owner** | **baru** |
| 4 | `CASEID` versus `NO_KLAIM` | — | ✅ **tertutup** oleh tangkapan layar |

Butir 3 hanya dapat dijawab dengan melihat baris Post Audit yang lebih baru di Pega. Satu-satunya
baris berisi hari ini berumur 1,5 tahun, sehingga cabang lainnya tidak dapat diamati.

---

## 29. Inbox Compliance tab Post Audit — disesuaikan ke aplikasi Pega (2026-09-24)

Melanjutkan §28. Tiga jawaban Work Owner, dan dua di antaranya berbunyi "sesuaikan sama aplikasi
PEGA" / "coba di cek lagi" — diperlakukan sebagai perintah menyisir ulang, bukan persetujuan.

### 29.1 OutStanding memakai `DateTime-Frame`, format bawaan Pega

`Section/InputPostAuditDtl_Section-Section.xml:4036` memasang `pyDateTimeFormat = DateTime-Frame`
pada mode kontrol KEDUA properti `.TanggalKirimPostAudit`. Itu menjelaskan tiga hal sekaligus:

1. **Kenapa `.TanggalKirimPostAudit` muncul dua kali** di section — satu kolom tanggal, satu kolom
   relatif, properti yang sama.
2. **Kenapa pencarian "months ago" nol hasil** di seluruh export termasuk `HTML/` dan `Function/` —
   kodenya ada di platform Pega, yang tidak ikut dalam export rule aplikasi.
3. **Kenapa penulisan ulangnya sepadan** — format yang sama dipakai belasan section Inbox lain
   (`DashboardClaim_Section1`, `InboxAnalystDoctor_Section`, `InboxManagerAdmin_Section`,
   `InboxOutstandingClaim_Section`, …), sehingga modul-modul itu akan memakainya kembali.

Dugaan §28 bahwa hasilnya tersimpan di `.pyNote` **gugur**: `SetAssignmentCompliance` memang
menulis `.pyNote`, tetapi isinya `"ASSIGN-WORKBASKET " + param.inskey + "!Compliance_Flow"` — kunci
assignment untuk tautan, bukan teks waktu.

### 29.2 Urutan baris: TEKS menurun, bukan tanggal

Layar Pega menampilkan `CPL-3, CPL-2, CPL-19, CPL-17, CPL-16, CPL-15, CPL-1`. Ketujuhnya cocok
persis dengan pengurutan teks menurun, dan tidak cocok dengan pengurutan angka.

Kunci urut dibetulkan menjadi `CASEID DESC` sebagai kunci PERTAMA, menggantikan
`TGL_KIRIM_POST_AUDIT DESC`.

**Kenapa ini layak dicatat sebagai keputusan, bukan sekadar perbaikan.** Mengurutkan `CPL-nn`
sebagai teks terlihat seperti cacat: `CPL-19` jatuh di bawah `CPL-3`. Pembaca berikutnya akan
tergoda "merapikannya" menjadi urutan angka — dan itu justru yang menyimpang dari Pega. Karena itu
alasannya ditulis di kueri, dan ada uji yang mematok ketujuh nomor persis dalam urutan layar Pega.

Pilihan ini mengikuti `P-5`: perilaku dipertahankan lebih dulu. Bila urutan angka lebih dikehendaki,
ia perbaikan yang harus diputuskan tersendiri.

### 29.3 Bentuk tunggal dan jamak, disimpulkan dari satu contoh

`"1 year 5 months ago"` memuat keduanya dalam satu kalimat. Dari situ:

- Setiap satuan memakai bentuknya sendiri — `1 year`, `5 months`, `1 day`, `3 days`.
- Sisa nol bulan tidak ditulis: `"1 year ago"`, bukan `"1 year 0 months ago"`.

Butir kedua adalah penyimpulan, bukan pengamatan — tidak ada baris tepat berulang tahun di layar.
Ia dipilih karena tidak ada pemformat waktu relatif yang menulis "0 months", dan akibatnya terbatas
pada satu hari dalam setahun per baris.

### 29.4 Hak akses: tidak ada yang diubah, dan itu hasil pemeriksaan

Jawaban Work Owner: belum mengurus hak akses, pastikan JONNY bisa, jangan hardcode.

| Yang diperiksa | Hasil |
|---|---|
| Identitas di-hardcode | **nol** di kode non-uji |
| Gerbang peran | **tidak ada** — setiap pengguna yang sudah masuk dapat membukanya |
| JONNY melihat menunya | **ya**, terbukti dari tangkapan layar |
| Isi antrean bergantung identitas | **tidak** — antreannya workbasket, isinya sama bagi semua |

Keputusannya karena itu: **tidak ada perubahan**. Menambahkan sesuatu agar "JONNY bisa" justru akan
memperkenalkan ketergantungan pada identitas yang sekarang tidak ada.

Satu hal ditegaskan supaya tidak salah dibaca kelak: `WorkbasketCompliance = "CompliancePNC"` BUKAN
identitas pengguna. Ia nama antrean yang dibaca dari `Flow/Register_Flow.xml:3769`, dan ia tetap
konstanta dengan alasan yang sudah dicatat di §26.4.

### 29.5 Yang masih terbuka

| # | Pertanyaan | Pemilik | Status |
|---|---|---|---|
| 1 | Siapa yang MENGISI `T_CLAIM_COMPLIANCE_H` | DBA / Tim Pega | **terbuka, mendesak** — tanpa pengisi, tab ini permanen kosong |
| 2 | Bunyi OutStanding untuk rentang pendek | Work Owner | **terbuka** — hanya cabang tahun yang teramati |
| 3 | Penyaring status | — | ✅ tertutup: isinya mengikuti Pega, tabelnya purpose-built |
| 4 | `CASEID` versus `NO_KLAIM` | — | ✅ tertutup oleh tangkapan layar |

Butir 1 tidak dapat diselesaikan dari modul ini: menambahkan penulisnya berarti menulis ke tabel
yang pemiliknya belum ditetapkan, dan itu menyentuh `P-1`.

---

## 30. Inbox Compliance — jalur tulis ke Post Audit (2026-09-24)

Melanjutkan §29. Dua jawaban Work Owner: tabelnya diisi lewat aplikasi, alurnya "seperti aplikasi
PEGA saja", nomornya "CPL-100001 — meniru bentuk Pega".

### 30.1 Alurnya dibaca dari kolom tabelnya, karena flow-nya hilang

`Compliance_Flow` TIDAK ADA di export — dirujuk `SetAssignmentCompliance-Act.xml:423`, tetapi
`Flow/` hanya memuat empat flow. Jadi "seperti aplikasi Pega" tidak dapat dibaca dari alurnya.

Yang dipakai sebagai gantinya adalah bukti yang ADA: keenam kolom `T_CLAIM_COMPLIANCE_H` cocok satu
per satu dengan baris tab Compliance ditambah satu catatan dan satu tanggal. Bentuk itu hanya masuk
akal bila barisnya lahir DARI sisi Compliance — dan itulah yang dibangun.

**Ini penyimpulan, bukan pembacaan.** Ia ditulis terbuka di kepala `postaudit.go` supaya pembaca
berikutnya tahu mana yang terbaca dan mana yang disimpulkan.

### 30.2 Penomoran: meniru bentuk Pega, dipisahkan lewat rentang

Keputusan Work Owner: `CPL-100001`, bukan tiga segmen bertitik seperti `PNCN.YY.xxxx` (`D-71`)
maupun `LPK.YY.xxxx` yang dipakai modul Pelaporan Klaim.

Pemisahan dari terbitan Pega karena itu lewat **rentang**, bukan bentuk. Tiga konsekuensi diterima
dan dicatat di `repo/sqlstore/number.go`:

1. Asal sebuah nomor tidak terbaca dari bentuknya — berbeda dari nomor klaim, yang `D-22` sengaja
   buat dapat dibedakan tanpa tabel pemetaan.
2. Rentangnya harus dijaga; bila Pega mencapai 100.001 keduanya bertabrakan, dan tabelnya tidak
   punya constraint unik yang menolaknya.
3. Tanpa segmen tahun. Tahun pengirimannya tetap terbaca dari `TGL_KIRIM_POST_AUDIT`.

Nomornya **tanpa nol di depan**, berbeda dari modul Pelaporan Klaim. Alasannya kebalikan dari
alasan di sana: kolom ini sudah berisi nomor Pega yang lebarnya tidak seragam, dan urutannya TEKS —
menambahkan nol tidak menyeragamkan apa pun terhadap baris lama, sedangkan lebar enam digit sudah
menjaga urutannya sendiri.

### 30.3 Empat hal yang sengaja TIDAK dikarang

| Yang tidak ditambahkan | Alasan |
|---|---|
| Penyaring kelayakan selain "klaimnya ada di antrean" | Tidak ada sumbernya; `Compliance_Flow` hilang |
| Perubahan status pada klaimnya | idem — dan mengubah status berarti MENULIS tabel Pega, melanggar `P-1` |
| Pemberitahuan | idem |
| Kewajiban mengisi Catatan | Kolomnya nullable; tidak ada bukti ia wajib |

Yang ditambahkan hanyalah satu pemeriksaan yang punya dasar: klaimnya harus SEDANG menunggu di
antrean Compliance. Itu meniru Pega, yang menuntut assignment-nya ada sebelum flow dijalankan, dan
predikat kuerinya sama persis dengan kueri daftar — sehingga tidak ada klaim yang tampil di layar
tetapi ditolak saat dikirim, maupun sebaliknya.

### 30.4 Transaksi di repo, bukan di usecase — dan kenapa

`08-TECHNICAL-STRATEGY.md` §4.5 menetapkan transaksi dimulai di lapisan aplikasi. `CreatePostAudit`
menyimpang: ia membungkus pengambilan nomor dan penyisipan baris dalam satu transaksi di dalam
repo.

Alasannya keduanya adalah SATU operasi penyimpanan yang tidak punya arti terpisah — nomor tanpa
baris bukan apa-apa. Memecahnya ke usecase berarti memindahkan detail penyimpanan ke lapisan yang
tidak boleh mengetahuinya.

Yang tetap dinyatakan: rollback TIDAK mengembalikan nomor yang sudah diambil; sequence memang tidak
dapat dibatalkan. Yang dijamin hanyalah bahwa baris yang tersimpan selalu punya nomor, tidak pernah
sebaliknya.

### 30.5 Dua kekurangan yang diterima secara sadar

**Tanpa kunci idempotensi.** Melanggar `10-API-STRATEGY.md` §7, dan akibatnya nyata: dua kali tekan
menghasilkan dua baris. Penahannya hanya tombol yang nonaktif selama permintaan berjalan — tidak
berlaku bagi permintaan langsung.

Penyaring "sudah pernah dikirim" tidak ditambahkan karena akan mengarang aturan: di Pega satu klaim
tampaknya dapat punya lebih dari satu pemeriksaan.

**Pengirimnya tidak tersimpan.** Tabelnya tidak punya kolom untuk itu; identitasnya hanya masuk
log. Log bukan jejak audit — ia berputar, dan retensinya bukan retensi audit. Ini bertentangan
dengan `D-28`, yang menuntut pelaku tercatat permanen, dan lebih berat lagi karena `D-59`
menjadikan jejak audit satu-satunya kontrol pengimbang.

**Usulan:** tambahkan kolom pengirim pada `T_CLAIM_COMPLIANCE_H`. Selama belum ada, jalur ini
sebaiknya tidak dipakai di produksi tanpa `S-5`.

### 30.6 Hak akses — tidak ada gerbang peran, dan di jalur ini akibatnya paling berat

Rute tulis ini dapat dipanggil setiap pengguna yang sudah masuk. Ia menerbitkan baris yang dibaca
pemeriksa Post Audit, dan tidak ada yang dapat menghapusnya dari layar.

Itu `TKT-F3-005`, yang bergantung pada tabel peran yang dapat dibangun tetapi belum dapat diisi
(`11-SECURITY.md` §3.1). Dicatat di `http/routes.go` sebagai utang yang disadari.

### 30.7 Yang masih terbuka

| # | Pertanyaan | Pemilik |
|---|---|---|
| 1 | Kolom pengirim pada `T_CLAIM_COMPLIANCE_H` | **Work Owner + DBA** |
| 2 | Apakah satu klaim boleh dikirim lebih dari sekali | **Work Owner** — menentukan perlu-tidaknya penyaring dan kunci idempotensi |
| 3 | Bunyi OutStanding untuk rentang pendek | **Work Owner** |
| 4 | Menjalankan `migrations/0011` di keempat entitas beserta grant-nya | **DBA** (`D-63`) |
| 1 | Apakah Master Proteksi Data sudah berisi baris ber-`MODUL='PNCSearchKlaim'`? Bila nol, layar menolak setiap pengguna | DBA + Work Owner |
| 2 | Apakah layar Master Proteksi Data ikut dimigrasikan? Selama belum, jatah hanya dapat ditambah lewat layar Pega | Work Owner |
| 3 | Label tipe pencarian yang sebenarnya dibaca pengguna — yang dipakai sekarang turunan dari deskripsi langkah, karena definisi propertinya tidak ada di export | Tim Pega |
| 4 | Apakah cacat pencarian Tanggal Lahir kelak diperbaiki? Bila ya, ia menjadi butir baru pada daftar perbaikan eksplisit `P-5` | Work Owner |

---

## 35. Modul Master Kategori Sparepart (2026-09-21)

Pengganti `Harness/GCNMCatSparepart-Harness.xml` atas `POOLDATA.GCNM_M_SPAREPART_CATEGORY`
(MENU_ID 33).

### 35.1 Pertanyaan yang diajukan sebelum kode ditulis, dan jawabannya

Tiga hal tidak dapat diputuskan dari bukti saja. Ketiganya diajukan ke Work Owner beserta
pilihan dan rekomendasinya; ketiganya dijawab sesuai rekomendasi.

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Balapan `max(PART_CATEGORY_ID)+1` — tiru apa adanya, kunci, atau minta sequence ke DBA? | **Tiru `max+1`, tetapi dikunci** |
| 2 | Tombol Approve/Reject di dalam layar ini, atau tunggu Inbox Manager? | **Di dalam layar ini** |
| 3 | Nama yang pernah ditolak tetap memblokir — tiru, atau kecualikan baris Reject? | **Tiru apa adanya** |

### 35.2 Keputusan 1 — penomoran memakai kunci tabel, bukan sequence baru

**Yang diputuskan.** Bentuk ID tetap **angka berurut** `MAX(PART_CATEGORY_ID)+1`, persis
seperti `RDB List/InsertMasterSparepartCategory_sql-SQL.xml`. Balapannya ditutup dengan
`LOCK TABLE ... IN EXCLUSIVE MODE` di dalam transaksi penambahan.

**Kenapa bukan sequence.** Meminta sequence baru menempuh `D-63` — permintaan tertulis,
persetujuan Work Owner, pelaksanaan DBA — dan modulnya tertahan sampai itu selesai. Bentuk
ID-nya juga akan berubah, padahal nilainya disalin apa adanya ke
`SPAREPART_HE.KATEGORI_SPART` yang sudah berisi angka-angka lama.

**Kenapa `LOCK TABLE`, bukan `FOR UPDATE`.** Ini perbedaan teknis yang menentukan, dan ia
berbeda dari modul master lain yang memakai `FOR UPDATE`:

> `FOR UPDATE` mengunci baris yang SUDAH ADA. Yang bertabrakan pada `max+1` adalah baris yang
> sedang **dibuat oleh keduanya** — dan baris yang belum ada tidak dapat dikunci.

`LOCK TABLE <nama> IN EXCLUSIVE MODE` berbentuk sama persis di Oracle 19c dan PostgreSQL 17+,
sehingga ia tidak melanggar `D-20`. Modus EXCLUSIVE dipilih karena **pembacaan tidak
terhalang** di kedua basis data.

**Keterbatasan yang disadari.** Penguncian menutup balapan antar-penulis yang melewati basis
data yang sama — termasuk kedua instans aplikasi (`D-27`) dan termasuk Pega selama masa
paralel, karena kunci tabel ditegakkan basis data. Yang **tidak** ditutupnya adalah baris
kembar yang sudah terlanjur ada. Penutupnya constraint unik, dan itu menunggu `R-08` dan
`D-63`.

**Utang teknis:** satu-satunya pernyataan non-DML di seluruh modul. Bila kelak sequence
tersedia, `category_lock_table` dan `category_next_id` dihapus bersamaan dan tidak ada tempat
lain yang perlu berubah.

### 35.3 Keputusan 2 — Approve/Reject di dalam layar

Sama dengan Master Bengkel, Panel, dan Sparepart, dan dengan alasan yang sama: di Pega
keputusan itu ada di `Section/ApprovalMasterKategoriSparepartHE` yang dipakai Inbox Manager,
dan Inbox Manager belum dibangun.

**Yang membuat penundaannya lebih mahal di sini:** kategori yang tertahan di Waiting Approval
**tidak dapat dipakai sparepart mana pun** — layar Master Sparepart menyaring
`APPROVAL = '1'`. Menunda keputusan di sini berarti memblokir layar lain, bukan hanya layar
ini.

Bentuknya ditiru persis: centang beberapa baris, satu tombol untuk seluruh pilihan.

**Yang TIDAK ditiru:** tombol **DETAILS** pada layar persetujuan Pega. Di layar ini setiap
baris sudah punya tombol Ubah yang membuka seluruh isinya — dan isinya hanya satu isian.
Tombol kedua yang membuka hal yang sama menambah pilihan tanpa menambah kemampuan.

### 35.4 Keputusan 3 — nama yang ditolak tetap memblokir, tetapi pesannya menjelaskan

`RDB List/ValidationSparepartCat-SQL.xml` tidak menyaring `APPROVAL` sama sekali. Perilakunya
ditiru apa adanya (`P-5`); selisih nol pada uji kesetaraan.

**Yang ditambahkan bukan perilaku melainkan keterangan.** Tanpa itu, penolakannya tidak dapat
dijelaskan dari layar — barisnya tidak terlihat di tab Approve maupun Waiting Approval.
Keterangannya masuk di `detail[].pesan`, sehingga hasil akhirnya sama persis dengan Pega dan
yang berbeda hanyalah pengguna tahu harus mencari ke mana.

**Future Enhancement yang dicatat:** mengecualikan baris berstatus Reject dari pemeriksaan
keunikan. Itu selisih perilaku yang menuntut butir baru pada daftar perbaikan eksplisit
`P-5` (`D-49`) dan persetujuan Work Owner tertulis — bukan diputuskan modul.

### 35.5 Penyimpangan yang diambil sendiri, dan alasannya

| # | Penyimpangan | Alasan |
|---|---|---|
| 1 | **Nama wajib diisi** — sistem lama menyimpan apa pun termasuk kosong | Kategori tanpa nama muncul sebagai baris kosong pada dropdown Master Sparepart: tidak dapat dibedakan dari "belum memilih", dan tidak dapat dipilih ulang setelah salah pilih. Baris lama yang sudah kosong tetap DIBACA apa adanya; penolakan hanya saat disimpan ulang |
| 2 | **Batas panjang nama 100 karakter** | ASUMSI, bukan dari DDL (`R-08`). Tanpa batas, penolakan datang sebagai ORA-12899 yang tidak menuntun ke mana pun. Seratus dipilih agar sama dengan Master Sparepart, karena nilai kolom ini muncul sebagai label di layar itu |
| 3 | **Pencarian pada nama** | Sistem lama memuat seluruh baris ke klipboard lalu menyaring di peramban (`pyPageSize=50`, tanpa kotak pencarian). Yang dipakai adalah pencarian bawaan `DataTable`, sama seperti layar master lain |
| 4 | **`TRIM` pada perbandingan `APPROVAL` dan kunci** | Rule lama membandingkan langsung. Bila kolomnya `CHAR`, Oracle memadatkan pembandingnya tetapi PostgreSQL tidak — baris yang sama akan hilang setelah pindah basis data (`D-24`). `TRIM` tidak mengubah hasil di Oracle dan menyelamatkannya di PostgreSQL |
| 5 | **Pemberitahuan di form tentang akibat menyimpan** | Akibatnya keluar dari layar ini — lihat §35.7. Satu-satunya cara pengguna mengetahuinya adalah diberi tahu |

### 35.6 Yang sengaja TIDAK dibangun

| Tidak dibangun | Alasan |
|---|---|
| **DELETE** | Kesembilan rule Pega yang menyentuh tabel ini tidak memuat satu pun pernyataan hapus, dan `D-66` melarang penghapusan fisik. Kategori yang tidak dipakai DITOLAK, bukan dibuang — dengan begitu sparepart lama yang menunjuknya tetap dapat menampilkan namanya |
| **Endpoint `/pilihan`** | Modul ini tidak punya tabel acuan: ia SENDIRI yang menjadi acuan bagi Master Sparepart dan Master Tipe Sparepart |
| **Isian Catatan pada penolakan** | Tabelnya tidak punya kolom penampungnya. Menggambar isian yang diam-diam membuang isinya lebih buruk daripada tidak menggambarnya |
| **Surel pemberitahuan** | `D-67` melarang akun pribadi dibawa apa adanya, dan `Activity/UpdateKategoriSparepart_act2` memakai satu alamat pribadi yang di-hardcode. Seam Notifier (`S-3`) belum ada; peristiwanya dicatat di log, alamatnya tidak ditulis di mana pun (`D-69`) |
| **Migrasi skema** | Tabelnya sudah ada di Oracle dan sudah dibaca modul `mastersparepart`. Tidak ada yang perlu dibuat |

### 35.7 Kopling ke Master Sparepart yang harus disadari

Ini bukan keputusan melainkan **akibat** yang ditemukan, dan ia layak tercatat karena
melintasi dua modul:

Menyimpan kategori SELALU menetapkan `APPROVAL := "0"`
(`Activity/UpdateKategoriSparepart_act2`). Selama menunggu, kategori itu **hilang dari
dropdown Kategori pada layar Master Sparepart**, yang menyaring `APPROVAL = '1'`.

Sparepart yang sudah menunjuk kategori itu tetap menyimpan ID-nya, tetapi namanya tidak lagi
dapat ditampilkan — dan sparepart BARU tidak dapat digolongkan ke sana sampai kategorinya
disetujui ulang.

Perilaku itu milik sistem lama apa adanya (`P-5`), bukan pilihan modul ini. Yang dikerjakan
hanyalah membuatnya terlihat: pemberitahuan di form, dan pemeriksaan
`category_count_orphan_sparepart` pada `claimpnc -periksa`.

### 35.8 Rekonstruksi yang dinyatakan, bukan disamarkan

Rule SQL yang menjalankan keputusan persetujuan **tidak ada di export** (`R-16`):
`Activity/UpdateKategoriSparepart_act` memanggil `UpdateSparepartCategoryClaimHE_sql` yang
tidak ada di antara 2.634 berkas. Kategori juga tidak ikut `SetApprovalAllMaster`, yang hanya
melayani `M_BENGKEL_HE`, `M_PANEL_HE`, dan `M_SPAREPART_HE`.

Bentuk `category_set_status` karena itu direkonstruksi dari `UpdateMasterSparepartCategory_sql2`
yang memang terbaca. Rekonstruksinya dinyatakan di **tiga tempat** — `Repo.SetStatus`, berkas
`.sql`, dan `usecase.Service.Decide` — supaya ia dapat diuji ulang begitu rule aslinya tiba.

**Yang harus diminta ke Tim Pega:** rule `UpdateSparepartCategoryClaimHE_sql`.

### 35.9 Utang teknis yang ditambahkan sesi ini

| # | Utang | Penutupnya |
|---|---|---|
| 1 | `MAX+1` berkunci tabel, bukan sequence | Sequence baru lewat `D-63`, bila kelak diputuskan |
| 2 | Tidak ada constraint unik pada `PART_CATEGORY_NAME` | DDL (`R-08`) + `D-63`. Sampai itu ada, `claimpnc -periksa` melaporkan nama kembar |
| 3 | Batas panjang nama diulang di dua tempat (Go dan `PartCategoryForm.tsx`) | `TestMaxNameLengthMatchesFrontendForm` menjaganya tetap terlihat |
| 4 | Pemetaan galat modul sendiri, bukan kontrak bersama | `TKT-F1-004` |
| 5 | Jalur tanpa awalan `/v1` | Penyeragaman seluruh aplikasi, bukan satu modul |
| 6 | Contoh memori modul ini dan `mastersparepart` TERPISAH | Keterbatasan modus memori, bukan cacat. Terhadap Oracle keduanya membaca tabel yang sama |

Utang ke-6 patut dijelaskan: saat aplikasi berjalan **tanpa Oracle**, kategori yang
ditambahkan di layar ini tidak muncul di dropdown layar Master Sparepart, karena kedua modul
punya penyimpanan memori sendiri-sendiri. Menyatukannya menuntut satu modul mengimpor
penyimpanan modul lain — tautan yang tidak ada di produksi, dan yang membuat modul selesai
harus disunting setiap kali tetangganya berubah. Dibiarkan dan dicatat, bukan ditambal.

### 35.10 Contoh memori memakai kunci angka, berbeda dari contoh Master Sparepart

`mastersparepart/repo/memory/sample.go` memakai kunci kategori `"KAT01"`.."KAT03". Bentuk itu
**tidak dapat benar**: `nvl(max(PART_CATEGORY_ID),0)+1` mustahil bekerja atas kunci seperti
itu.

Contoh modul ini karena itu memakai `"1"`.."6". Contoh pada modul Master Sparepart **tidak
diubah** menyesuaikannya — modul itu sudah selesai dan berada di bawah Isolasi Protektif, dan
bentuk kunci contohnya tidak memengaruhi apa pun di produksi. Selisihnya dicatat di sini
supaya tidak terbaca sebagai kelalaian.

### 35.11 Temuan di luar lingkup — dilaporkan, tidak diperbaiki

| Temuan | Status |
|---|---|
| `mastergroupingsparepart` (MENU_ID 32) dikerjakan **sesi paralel** selagi sesi ini berjalan — backend saat sesi ini dimulai, frontend menyusul menjelang selesai | Di luar lingkup (Fokus Penuh). Berkas bersama diperiksa ulang di akhir sesi: kedua modul berdampingan bersih di `App.tsx`, `menu/registry.ts`, dan `api/types.ts` |
| Tiga uji `master-rekening/AccountPage.test.tsx` gagal | **Pre-existing** — dipastikan dengan menjalankan uji itu setelah seluruh perubahan sesi ini di-stash. Modul di bawah Isolasi Protektif |
| Satu selisih `gofmt` pada komentar `checkGrouping...` di `cmd/claimpnc/check.go` | Milik modul grouping, bukan sesi ini |

## 36. Modul Master Grouping Sparepart (2026-09-21)

Pengganti `Harness/GroupingSparePart_HE-Harness.xml` atas `POOLDATA.SPAREPART_HE_VIN_KEY`
beserta pendampingnya `SPAREPART_HE_VIN_GROUP` (MENU_ID 32).

### 36.1 Pertanyaan yang diajukan sebelum kode ditulis, dan jawabannya

Empat hal tidak dapat diputuskan dari bukti saja. Keempatnya diajukan ke Work Owner beserta
pilihan dan rekomendasinya, dan keempatnya dijawab dengan satu prinsip yang sama:
**"sesuai Pega dan konsisten"**.

| # | Pertanyaan | Penerapan prinsip itu |
|---|---|---|
| 1 | Kolom `NAMA` pada `LOKASI_PANEL_HE` — nama panel atau nama lokasi? | Kueri Pega ditiru apa adanya; jawabannya dicari dari data, bukan ditebak |
| 2 | `SPAREPART_HE_VIN_GROUP` — tulis keduanya, atau induk saja? | Keduanya: Pega membaca keduanya, dan form mengisi kolom milik keduanya |
| 3 | Persetujuan — simpan ulang seluruh baris seperti Pega, atau borongan? | Borongan; hasil yang TERLIHAT identik, dan bentuknya konsisten dengan tiga master HE lain |
| 4 | Isian acuan — dropdown, atau ketik bebas? | Persis kontrol Pega **per isian**, tidak diseragamkan |

### 36.2 Keputusan 1 — kueri Sisi ditiru apa adanya, dan pertanyaannya dijawab `-periksa`

**Persoalannya.** Kolom `NAMA` pada `POOLDATA.LOKASI_PANEL_HE` TIDAK pernah muncul sebagai
kolom yang dibaca di seluruh export — hanya sebagai penyaring pada satu kueri
`RDB List/GetDataSisiPanel-SQL.xml`:

    select sisi_panel AS "NAME" from pooldata.lokasi_panel_he
     where id_panel = {TempSparepart.MIN_STOCK} and nama = {TempSparepart.PANJANG}

Dua pembacaan sama-sama masuk akal, dan keduanya tidak dapat benar bersamaan:

| | Isi kolom `NAMA` | Siapa yang menuntutnya |
|---|---|---|
| (a) | nama **LOKASI** | modul **Master Panel**, yang jalur tulisnya mengisi `NAMA := LOKASI_PANEL` |
| (b) | nama **PANEL** | modul **ini** — nilainya berasal dari autocomplete atas `BrowseMasterPanel_HE_RD`, report definition atas `PANEL_HE` yang menampilkan `.NAME` |

Bukti untuk (b) langsung: `pySourceName = BrowseMasterPanel_HE_RD`, kelasnya
`ASM-FW-GCNMFW-Int-PANEL_HE`, dan label isiannya memang **"Nama Panel"**.

**Yang diputuskan.** Kueri ditiru apa adanya — `ID_PANEL` DAN `NAMA` — dan pertanyaannya
dijawab **secara empiris**: `claimpnc -periksa` menghitung berapa baris `NAMA`-nya cocok
dengan `PANEL_HE.NAME` versus dengan `LOKASI_PANEL`-nya sendiri, lalu menyandingkan keduanya
beserta apa yang harus dilakukan pada setiap kemungkinan.

**Modul Master Panel TIDAK disunting dari sini.** Ia sudah dinyatakan selesai, dan
mengubahnya atas dasar tafsiran satu modul lain justru kebalikan dari apa yang dibutuhkan.
Yang dikerjakan adalah menyediakan angkanya supaya keputusannya diambil atas bukti.

**Konsekuensi yang harus disadari.** Bila (b) yang benar, jalur tulis Master Panel akan
**mematikan daftar Sisi di layar ini tanpa satu pun pesan galat** — kelas kegagalan yang
paling mahal ditemukan, karena layarnya tampak berfungsi dan hanya dropdown-nya yang kosong.
Sisi adalah isian **wajib**, sehingga akibatnya bukan ketidaknyamanan melainkan layar yang
tidak dapat dipakai sama sekali.

### 36.3 Keputusan 2 — dua tabel ditulis dalam satu transaksi

**Yang diputuskan.** Penyimpanan menulis `SPAREPART_HE_VIN_KEY` dan
`SPAREPART_HE_VIN_GROUP` dalam SATU transaksi. Pembacaannya tetap `INNER JOIN`, persis
seperti `GetDataMasterGrouping`.

**Kenapa bukan induk saja.** "No Rangka" dan "Tipe Kendaraan" adalah kolom milik tabel
pendamping, dan keduanya isian yang diketik pengguna. Menulis induk saja akan membuat kedua
isian itu menjadi hiasan yang tidak tersimpan.

**Kenapa `INNER JOIN` dipertahankan.** Baris induk tanpa pendamping TIDAK muncul di layar
lama. Menggantinya `LEFT JOIN` akan memunculkan baris yang selama ini tidak terlihat siapa
pun — perubahan perilaku yang tidak diminta. Jumlahnya dilaporkan `-periksa` lewat
`grouping_count_without_group`.

**Kenapa pendampingnya di-UPDATE, bukan dihapus-lalu-disisipkan-ulang.** Berbeda dari baris
anak Master Panel yang tidak punya kunci sendiri dan jumlahnya berubah-ubah, pendamping di
sini berhubungan **satu-lawan-satu** dengan induknya. Ia dapat di-UPDATE di tempat, sehingga
**tidak ada satu pun DELETE di seluruh modul ini** (`D-66`). Uji `TestNoDeleteAnywhere` yang
menjaganya.

### 36.4 `NO_RANGKA` ada di KEDUA tabel, dan sistem lama memakai keduanya

Ini temuan yang paling mudah terlewat, dan ia menyentuh keunikan data:

| Rule | Kolom yang disentuh |
|---|---|
| `RDB List/GetDataMasterGrouping-SQL.xml` | memilih **`B.NO_RANGKA`** untuk ditampilkan |
| `Activity/UpdateGroupingSparepartHE_act` langkah 3 | menyaring **`A.NO_RANGKA`** untuk memeriksa duplikat |
| `RDB List/GetNoGroup-SQL.xml` | menyaring **`A.NO_RANGKA`** untuk mencari nomor grup |

Artinya yang DITAMPILKAN dan yang DIPERIKSA keunikannya bukan kolom yang sama. Selama
keduanya berisi nilai yang sama, perbedaannya tidak terlihat; begitu berbeda, sebuah baris
dapat lolos pemeriksaan duplikat sambil tampil sebagai duplikat di layar.

**Yang dikerjakan.** Penyimpanan menulis nomor rangka ke **keduanya** dengan nilai yang sama,
sehingga ketiga rule selalu sepakat dan selisih baru tidak dapat lahir lewat modul ini. Baris
lama yang sudah terlanjur berbeda TIDAK diperbaiki — jumlahnya dilaporkan
`grouping_count_chassis_mismatch` supaya besarnya terlihat lebih dulu.

### 36.5 Cacat nomor grup yang TIDAK direplikasi, dan kenapa

`RDB List/GetNewNoGroup-SQL.xml`:

    select to_number(nvl(max(NO_GROUP_RANGKA),0)+1) AS "ID" from POOLDATA.sparepart_he_vin_key

`MAX` atas kolom **teks** adalah maksimum leksikografis, sedangkan nomor grup ditulis
berawalan nol oleh `UpdateGroupingSparepartHE_act` langkah 7 (awalan `000` ditambah nomornya).
Akibatnya `0009` lebih besar daripada `00010` — sehingga **setelah grup kesembilan, kueri itu
selalu mengembalikan 10**, dan setiap grup baru menerima nomor yang sama.

**Tidak direplikasi.** Meniru cacat yang menghasilkan kunci ganda bukan kesetaraan perilaku
melainkan kerusakan data. Nomor terbesarnya dihitung **secara angka di Go**; nilai yang tidak
dapat diurai dilewati dan jumlahnya dilaporkan `-periksa`.

**Bentuk berawalan `000` TETAP dipertahankan**, termasuk lebarnya yang bertambah mulai nomor
10. Baris yang sudah ada memakainya, dan menerbitkan bentuk lain akan membuat dua bentuk hidup
berdampingan untuk hal yang sama.

### 36.6 Satu departure yang menuntut persetujuan `D-54`

`RDB List/GetNoGroup-SQL.xml` mengembalikan `TO_NUMBER(NO_GROUP_RANGKA)`. Akibatnya baris yang
**bergabung** ke sebuah grup menyimpan `1` sementara baris yang **membuka** grup itu menyimpan
`0001` — dua bentuk teks berbeda untuk grup yang sama, pada kolom yang sama.

**Yang dikerjakan.** `TO_NUMBER` tidak dibawa: nomor grup disimpan APA ADANYA seperti yang
tersimpan pada grup yang diikuti, sehingga baris yang bergabung membawa nomor yang SAMA PERSIS
dengan grupnya.

**Statusnya.** Ia **tidak ada** pada tiga belas butir `P-5` (`D-49`), sehingga selisih yang
muncul pada uji kesetaraan gerbang 1 menuntut **persetujuan Work Owner tertulis** (`D-54`).
Dicatat di sini supaya ia diputuskan, bukan ditemukan.

Yang meringankannya: nomor grup TIDAK digambar di layar lama mana pun, sehingga perubahan ini
tidak mengubah satu pun isian yang dilihat pengguna.

### 36.7 Keputusan 3 — persetujuan borongan, hanya kolom APPROVAL

**Di sistem lama, menyetujui berarti MENYIMPAN ULANG seluruh baris.**
`Section/ApprovalPNCMasterGroupingSparepartHE` memanggil `UpdateGroupingSparepartHE_act` —
activity penyimpanan yang sama — dengan `Param.Approval` yang berbeda.

**Yang dikerjakan.** Endpoint `/keputusan` borongan yang hanya menyentuh kolom `APPROVAL`,
bentuk yang sama dengan Master Bengkel, Master Panel, dan Master Sparepart.

**Hasil yang dilihat pengguna sama persis.** Yang berbeda adalah dua hal yang keduanya cacat
pada bentuk lama:

1. **Persetujuan dapat GAGAL karena validasi.** Baris yang sudah terlanjur tersimpan dengan
   isian yang kini dianggap tidak sah tidak akan pernah dapat disetujui — dan pesan yang
   muncul bicara tentang isian, padahal yang sedang dilakukan adalah menyetujui.

2. **Menyetujui dapat MENIMPA isi baris** dengan apa pun yang sedang ada di form persetujuan,
   termasuk nilai yang sudah basi.

Konsekuensi yang sama juga ada di jalur simpan: `Param.Approval` tidak lagi dapat dikirim
klien, sehingga satu permintaan simpan tidak dapat lagi menyetujui dirinya sendiri.

### 36.8 Keputusan 4 — kontrol isian ditiru per isian, tidak diseragamkan

| Isian | Kontrol Pega | Yang dibangun |
|---|---|---|
| Nomor Sparepart | `pxTextInput` + `SetDataSparepart` saat blur | ketik bebas + pencarian saat blur |
| Nama/Kategory/Type Sparepart | terisi otomatis, baca-saja | teks baca-saja, bukan kotak yang dimatikan |
| Nama Panel | `pxAutoComplete` atas `BrowseMasterPanel_HE_RD`, APPROVAL 1 | dropdown dari daftar yang sama |
| Sisi | `pxDropdown` dari `TempDataSisiPanel.pxResults` | dropdown, dibaca setelah panel dipilih |
| Tipe Kendaraan | `pxAutoComplete` atas `D_TypeHEList` | dropdown dari sumber yang sama |
| No Rangka · Grouping Dengan No Rangka · Catatan | `pxTextInput` | isian teks |

Nomor Sparepart sengaja **tidak** dijadikan dropdown meski itu lebih nyaman: Pega
membiarkannya diketik, dan daftar sparepart jauh lebih panjang daripada daftar panel.

### 36.9 Selisih terencana terhadap sistem lama

Seluruhnya menolak isian yang di sistem lama diterima; baris lama tetap **dibaca apa adanya**,
dan penolakan hanya terjadi saat barisnya disimpan ulang.

| # | Selisih | Kenapa |
|---|---|---|
| 1 | Keempat anggota kunci alami **wajib diisi** | Kunci yang salah satu anggotanya kosong tidak dapat membedakan dua baris. Sistem lama memeriksa duplikat atas kunci yang seluruhnya kosong, sehingga baris kedua yang kosong SELALU ditolak dengan pesan yang tidak menuntun |
| 2 | Kunci alami diperiksa **juga pada jalur simpan** | `UpdateGroupingSparepartHE_act` langkah 3 bersyarat ID sama dengan `UnknownID` — hanya benar pada penambahan. Akibatnya dua baris berkunci sama dapat lahir cukup dengan menyunting salah satunya |
| 3 | Pemeriksaan kunci **mengabaikan besar-kecil huruf dan spasi tepi** | Kueri lama membandingkannya apa adanya |
| 4 | **Menggabungkan baris dengan nomor rangkanya sendiri ditolak** | Sistem lama tidak memeriksanya, dan hasilnya pesan galat yang membingungkan |
| 5 | Sandi sisi di luar tiga yang dikenal **dikembalikan apa adanya** | `GetSisiPanel` memaksanya menjadi KANAN, sehingga nilai rusak tampil sebagai nilai yang sah |
| 6 | Nomor grup dihitung **secara angka** | Lihat §36.5 |
| 7 | Nomor grup grup yang diikuti disimpan **apa adanya** | Lihat §36.6 — satu-satunya yang menuntut `D-54` |

### 36.10 Yang TIDAK dibawa dari sistem lama

| Yang ditinggalkan | Dasarnya |
|---|---|
| Surel pemberitahuan ke akun pribadi ter-hardcode | `D-15`, `D-67`; `SendEmailNotification` pun tidak ada di export (`R-07`) |
| Perangkaian penyaring SQL dari nilai pengguna | `08-TECHNICAL-STRATEGY.md` §4.3 — seluruh nilai menempuh parameter binding |
| Kontrak galat berbasis string `ErrMsg` yang memikul pesan SUKSES | `D-68`, pola yang sama ditolak pada `ADD_NEWMASTERVIRTUALACCOUNT` |
| Penulisan dokumen JSON ke `M_SPAREPART_HE_VIN_KEY` | `D-02`, `D-68`; nama kunci JSON-nya tidak dapat diketahui (`R-16`) |
| Alias kolom yang menyesatkan — `PANJANG`, `LEBAR`, `TINGGI`, `MAX_STOCK`, `BANK_ID`, `City` | `D-19`, `03-CURRENT-ARCHITECTURE.md` §4.2 |

### 36.11 Utang yang disadari

| Utang | Kenapa dibiarkan |
|---|---|
| DDL kedua tabel belum ada | `R-08`. Panjang isian, tipe kolom, dan constraint unik seluruhnya asumsi |
| Balapan `MAX(ID)+1` hanya **dipersempit** | Penutupnya sequence atau constraint unik; keduanya menempuh `D-63` |
| Kode galat dipetakan modul ini sendiri | `TKT-F1-004` masih terhalang; menambah ke pemetaan modul auth berarti menyunting modul yang sudah selesai |
| Jalur tanpa `/v1` | Kontrak yang ada belum memakainya; memperkenalkannya di satu modul akan membuat dua gaya hidup berdampingan |
| `eslint` tidak dijalankan | Repo ini tidak punya `eslint.config.*`; gerbang lint frontend memang belum terpasang |

---

## 37. Modul Master Tipe Sparepart (2026-09-22)

Pengganti `Harness/GCNMMasterSparepartType-Harness.xml` atas
`POOLDATA.GCNM_M_SPAREPART_TYPE` (MENU_ID 34). Layar keempat dan terakhir di rumpun
sparepart.

### 37.1 Dua pertanyaan yang diajukan sebelum kode ditulis

| # | Pertanyaan | Jawaban Work Owner |
|---|---|---|
| 1 | Nama tipe unik **global** atau **per kategori**? `ValidationSparepartType` tidak menyaring `PART_CATEGORY_ID` maupun `APPROVAL` | **Tiru apa adanya — unik global** |
| 2 | Kategori berstatus apa yang boleh dipilih di dropdown? | **"Sesuai PEGA"** |

Jawaban kedua tidak dapat langsung dieksekusi: rule-nya tidak ada. Lihat §37.3.

### 37.2 Keputusan 1 — keunikan nama ditiru apa adanya, termasuk cakupannya yang aneh

`RDB List/ValidationSparepartType-SQL.xml`:

```sql
select PART_SECTION_NAME from POOLDATA.gcnm_m_sparepart_type
 where upper(PART_SECTION_NAME) = {TempValidateTypeSparepart.CaseID}
```

Tidak ada penyaring `PART_CATEGORY_ID`, dan tidak ada penyaring `APPROVAL`. Dua akibatnya:

1. **"KACA DEPAN" tidak dapat ada sekaligus di kategori BODY dan KABIN.** Keunikan berlaku
   di seluruh tabel, bukan di dalam satu kategori.
2. **Nama yang pernah DITOLAK memblokir selamanya**, dan barisnya tidak terlihat dari tab
   mana pun selain tab Reject.

Keduanya ditiru apa adanya (`P-5`, keputusan Work Owner 2026-09-22), konsisten dengan
keputusan yang sama pada Master Kategori Sparepart sehari sebelumnya.

**Yang ditambahkan bukan aturannya melainkan KETERANGANNYA.** Pesan galat 409-nya menyebut
kedua kemungkinan itu secara eksplisit — "termasuk tipe di kategori yang berbeda, dan
termasuk tipe yang sudah ditolak" — karena tanpa itu penolakannya tidak dapat dijelaskan dari
layar. Pengguna melihat nama yang tampak bebas, ditolak tanpa sebab yang terlihat.

Perbaikannya — keunikan per kategori — dicatat sebagai **Future Enhancement**, bukan
dikerjakan sambil jalan.

### 37.3 Keputusan 2 — "sesuai Pega" atas rule yang tidak ada

**Persoalannya.** Dropdown `---PILIH KATEGORI---` terikat pada page
`TempSparepartTypeClaimHE2` lewat `pyListSource=pageList`. Page itu **tidak dimuat satu pun
rule di export** (`R-16`):

| Activity layar ini | Page yang dimuatnya |
|---|---|
| `SetMasterTipeSparepart_act` | `TempInputTipeSparepart`, `TempSparepartTypeClaimHE` |
| `SetMasterTipeSparepartReject_act` | ditambah `TempReject`, `TempKategori` |
| `GCNMBrowseMasterSparepartType_act` | `TempSparepartTypeClaimHE` |

Nol yang menyentuh `...HE2`. Pencarian seluruh repositori atas nama page itu hanya menemukan
harness dan keempat section-nya.

**Yang diputuskan.** Jawabannya **diturunkan dari pola Pega sendiri**, bukan dipilih:

| Rule | Penyaring | Perannya |
|---|---|---|
| `BrowseTipeSparepart` | `APPROVAL = '1'` | pemilih |
| `GCNMBrowseMasterSparepartCategory_act2` | `approval = 1` | pemilih |
| `GCNMBrowseMasterSparepartCategory_act` | `approval = 0` | antrean persetujuan |

Setiap dropdown yang menawarkan master sebagai PILIHAN menyaring `'1'`; `'0'` hanya dipakai
layar antrean. Dropdown ini pemilih → **`'1'`**.

**Rekonstruksinya dinyatakan, bukan disamarkan.** Ia ditulis di tiga tempat: doc comment
`LookupRepo.ListCategories`, komentar `type_category_list` pada berkas `.sql`, dan konstanta
`approvedLookup` — ketiganya menyebut rantai buktinya dan menyatakan ia dapat diuji ulang
begitu rule aslinya tiba.

### 37.4 Keputusan 3 — inner join Pega diganti LEFT JOIN

**Ini satu-satunya selisih perilaku pada jalur baca, dan ia diambil sadar.**

`BrowseMasterSparepartTypeClaimHE_sql` memakai inner join gaya koma. Akibatnya tipe yang
menunjuk kategori yang tidak ada **hilang dari layar lama** — tidak dapat dilihat, tidak
dapat disunting, tidak dapat diperbaiki. Barisnya tetap ada di basis data, dan tetap terbaca
dropdown Tipe pada layar Master Sparepart yang tidak ber-JOIN.

| Pilihan | Akibatnya |
|---|---|
| Tiru inner join | kesetaraan sempurna; baris rusak **terkunci selamanya** |
| **LEFT JOIN** *(dipilih)* | baris tetap terlihat dengan kolom Kategori bertanda "—"; **dapat diperbaiki** lewat Ubah |

Dipilih yang kedua karena **baris yang tidak dapat dilihat tidak dapat diperbaiki**,
sementara modul ini justru menuntut kategori diisi saat menyimpan.

**Konsekuensinya dikelola, bukan diabaikan:**

1. `claimpnc -periksa` melaporkan jumlahnya lewat `type_count_orphan_category`, dengan
   kalimat yang menyebutnya **selisih yang disengaja** dan menyatakan itulah baris yang akan
   tampak berlebih pada uji kesetaraan gerbang 1.
2. Uji `TestJoinToCategoryIsLeftJoin` menolak siapa pun yang "merapikannya" kembali menjadi
   inner join demi menyamai Pega.
3. Form mempertahankan kategori tidak sah sebagai pilihan berlabel *"(tidak lagi
   disetujui)"*, supaya pengguna yang hanya ingin mengubah nama tidak ikut memindahkan
   kategorinya tanpa sadar.

### 37.5 Keputusan 4 — keberadaan kategori diperiksa saat menyimpan

Sistem lama tidak memeriksanya: layar hanya menawarkan dari daftarnya sendiri. Cukup selama
satu-satunya jalan masuk adalah layar itu — dan tidak cukup ketika ada API.

Pemeriksaannya ada di **lapisan aplikasi**, bukan di dalam transaksi penyisipan, dan itu
disengaja. Alasannya ditulis di `type_lock_table`: menariknya ke dalam transaksi akan
memperluas cakupan kunci ke tabel milik modul lain **tanpa menutup keadaan apa pun** —
kategori tetap dapat ditolak semenit setelah tipenya tersimpan. Penguncian hanya akan
mempersempit jendelanya dari selamanya menjadi selamanya-dikurangi-sedetik.

Galatnya dibedakan: **409 `kategori_sparepart_tidak_ditemukan`**, bukan 422. Penyebabnya
bukan salah ketik — pengguna memilihnya dari dropdown — melainkan dunia yang berubah di luar
formnya. Perbaikannya "muat ulang pilihan", bukan "betulkan isian", dan pesannya mengatakan
persis itu.

### 37.6 Keputusan 5 — satu tabel ditulis, satu tabel dibaca

`P-1` ditegakkan secara mekanis, bukan hanya disepakati:

| Tabel | Perlakuan | Penulisnya |
|---|---|---|
| `POOLDATA.GCNM_M_SPAREPART_TYPE` | **ditulis** modul ini | modul ini |
| `POOLDATA.GCNM_M_SPAREPART_CATEGORY` | **dibaca saja** | `masterkategorisparepart` |
| `POOLDATA.SPAREPART_HE` | dibaca saja, hanya oleh pencacah `-periksa` | `mastersparepart` |

Uji `TestOnlyOwnedTableIsWritten` memeriksa setiap kueri yang diawali `INSERT`, `UPDATE`,
atau `LOCK`: ia wajib menyentuh tabel tipe, dan wajib TIDAK menyentuh kedua tabel lainnya.

Tipe `Category` dideklarasikan sendiri di `lookup.go` alih-alih mengimpor
`masterkategorisparepart`. Harganya disadari — tiga tipe berbeda atas satu tabel di tiga
modul — dan dibayar karena mengimpor akan mengikat dua modul master yang seharusnya dapat
berpindah sendiri-sendiri.

### 37.7 Penamaan: kenapa `PartType`, bukan `PartSection`

Awalan kolomnya `PART_SECTION_*`, dan `masterkategorisparepart.PartCategory` memang mengikuti
awalan kolomnya sendiri. Mengikuti pola itu di sini akan menghasilkan `PartSection` — dan itu
menyesatkan: **tidak ada satu pun layar, menu, maupun caption yang menyebut "section"**. Yang
dilihat dan diucapkan pengguna adalah "Tipe Sparepart".

`PartType` mengikuti **nama tabelnya** (`GCNM_M_SPAREPART_TYPE`) dan nama bisnisnya. Ia juga
sama dengan `mastersparepart.PartType` yang sudah lebih dulu menamai hal yang sama pada
lookup-nya. `Type` sendiri tidak dapat dipakai — kata kunci Go.

Nama folder mengikuti `D-81`: `internal/mastertipesparepart` dan
`src/modules/master-tipe-sparepart`, dari nama modul yang disebut Work Owner. Isinya
berbahasa Inggris sesuai `D-80`.

### 37.8 Yang TIDAK dibawa dari sistem lama

| Yang ditinggalkan | Dasarnya |
|---|---|
| Alias `"CityID"`, `"City"`, `"District"`, `"DistrictID"` — dua terakhir **tertukar** terhadap pola "…ID" | `D-19`, `03-CURRENT-ARCHITECTURE.md` §4.2 |
| Properti input pinjaman dari kelas Master Bengkel: `CITY_ID` (nama), `DISC_JASA` (kategori), `NO_ACCOUNT` (status), `ACCOUNT_ID` (kunci) | idem |
| Surel ke PIC lewat `SendEmailNotification` | `D-15`, `D-67`; activity-nya sendiri tidak ada di export (`R-07`) |
| Inner join yang membuang baris yatim | lihat §37.4 |
| Balapan `max(...)+1` tanpa penguncian | ditutup penguncian tabel, mengikuti Master Kategori Sparepart |

### 37.9 Utang yang disadari

| Utang | Kenapa dibiarkan |
|---|---|
| Penyaring `APPROVAL='1'` pada dropdown adalah **rekonstruksi** | Rule pemuatnya tidak ada (`R-16`). Dinyatakan di tiga tempat supaya dapat diuji ulang saat rule-nya tiba |
| DDL tabel belum ada | `R-08`. `MaxNameLength = 100` asumsi; tipe kolom disimpulkan dari bekerjanya `max(...)+1` |
| Tidak ada constraint unik pada nama | `R-08`, `D-63`. Baris kembar yang sudah ada dilaporkan `-periksa` |
| Balapan ID hanya **dipersempit** | Penutupnya sequence atau constraint unik; keduanya menempuh `D-63` |
| Selisih LEFT JOIN akan muncul di gerbang 1 | Disengaja; dilaporkan `-periksa` lebih dulu agar dapat dijelaskan sebelum pengujian |
| Kode galat dipetakan modul ini sendiri | `TKT-F1-004` masih terhalang |
| Jalur tanpa `/v1` | Kontrak yang ada belum memakainya |
| Modus memori: dropdown dilayani salinan acuan sendiri | Menyatukannya menuntut satu modul mengimpor penyimpanan modul lain — tautan yang tidak ada di produksi |
| `eslint` tidak dijalankan | Repo ini masih tidak punya `eslint.config.*` |

---

## 38. Modul Master Login (2026-09-22)

Layar `MasterLoginSurvey` (MENU_ID 37) atas `POOLDATA.MST_LOGIN_SURVEYOR`. Tujuh kolom, lima
isian, **tanpa persetujuan**, **tanpa penghapusan**, **tanpa tabel acuan**.

Work Owner menjawab keempat pertanyaan pembuka dengan **"Sesuaikan dengan PEGA"**. Bab ini
mencatat apa artinya itu untuk tiap keputusan — termasuk dua yang tidak dapat dituruti
secara harfiah, dan mengapa.

### 38.1 Keputusan 1 — LOGIN diturunkan di SERVER, bukan diterima dari klien

Di Pega, LOGIN dihitung di layar pada setiap perubahan isian Nama
(`Section/BrowseLoginSurveyor` memasang aksi `refresh` bereven `change` yang menjalankan
`SetLoginSurveyor_act`), lalu dikirim kembali sebagai isian biasa.

Di sini badan permintaan **tidak memuatnya sama sekali**; server menurunkannya sendiri.

Alasannya bukan selera: LOGIN adalah **kunci baris** — setiap pernyataan simpan menyaring
`where login = ...`. Kunci baris tidak boleh bergantung pada kejujuran klien. Permintaan yang
tidak datang dari layar dapat mengirim LOGIN apa pun, dan yang dapat dilakukannya bukan
sekadar menyimpan nilai aneh: ia dapat **menimpa baris orang lain**.

Layar tetap memperlihatkan hasilnya saat pengguna mengetik — meniru perilaku lamanya — tetapi
ia menghitungnya untuk **ditampilkan**, bukan untuk dikirim. Karena kedua sisi menghitung hal
yang sama, keduanya diuji dengan **kasus yang sama persis**:

| Sisi | Uji |
|---|---|
| Backend | `TestDeriveLoginMatchesPegaExpression`, 13 kasus |
| Frontend | `deriveLogin(%j)`, 11 kasus yang sama |

Ekspresinya ditiru apa adanya, termasuk yang **tidak** dilakukannya: tidak ada
`@toUpperCase`, sehingga huruf besar-kecil dipertahankan.

### 38.2 Keputusan 2 — Nama terkunci, dan penguncian itu ditegakkan DI SERVER

`pyDisabledWhen = TempLoginSurvey.pyLabel='Update'` pada kontrol Nama. Layar baru
menguncinya, **dan server menolaknya** lewat `ErrNameLocked` → `409`.

Penguncian di antarmuka adalah kenyamanan tampilan; permintaan yang tidak datang dari layar
itu tidak tersentuh olehnya. Preseden yang sama sudah dipakai isian NAMA pada Master
Supplier.

Akibatnya pada keutuhan data, bukan kerapian: Nama yang berubah menghasilkan LOGIN yang
berubah, sedangkan pernyataan simpannya masih menyaring `where login = <login lama>` —
barisnya tidak akan pernah ditemukan, dan penyimpanan "berhasil" tanpa mengubah apa pun.

Perbandingannya **peka huruf besar-kecil**, berbeda dari pemeriksaan keunikan. NAMA adalah
teks yang dibaca manusia dan ditampilkan apa adanya; LOGIN adalah kunci.

### 38.3 Keputusan 3 — pemeriksaan login ganda DIREKONSTRUKSI

Pega memeriksanya terhadap **tabel operator Pega**:

```
Param.pyReportName  := "GCNMGetListOfOperators"
Param.pyReportClass := "Data-Admin-Operator-ID"
Param.UserId        := local.login
```

Tabel itu **tidak ada di sistem baru**, dan tidak akan pernah ada: `ADR-0002` menolak membawa
engine Pega, dan kontrak identitas `F-3` belum ditetapkan (`R-14`).

Penggantinya: keunikan `LOGIN` pada `POOLDATA.MST_LOGIN_SURVEYOR` sendiri. Itu bukan pilihan
sembarang — `LOGIN` memang kunci alaminya, dan dua baris berlogin sama membuat satu
penyimpanan mengubah keduanya sekaligus.

Pesannya diambil dari `local.msg` apa adanya: *"Login sudah terdaftar dengan nama yang sama"*.
Yang **ditambahkan** hanyalah keterangan pada `detail`, karena penolakannya tidak dapat
dijelaskan tanpa itu: yang bentrok bukan Nama melainkan LOGIN yang **diturunkan** darinya,
sehingga "Budi Hartono" dan "Budi.Hartono" menghasilkan login yang sama persis.

Rekonstruksi ini dinyatakan di **empat tempat** — `ErrLoginTaken`, `login_find_by_key`,
`CodeLoginTaken`, dan `SurveyorLoginErrorCode.loginTaken` — supaya ia dapat diuji ulang
begitu kontrak `F-3` tiba, bukan tersamar sebagai fakta.

Perbandingannya **tidak peka huruf besar-kecil** (`UPPER(TRIM(LOGIN))`), sedangkan
pengambilan satu baris peka. Pembedaan itu disengaja dan dijelaskan di banner berkas `.sql`.

### 38.4 Keputusan 4 — `GCNMCreateOperator` dan kata sandi bersama TIDAK dibawa

Ini satu-satunya tempat jawaban "sesuaikan dengan Pega" **tidak dapat dituruti secara
harfiah**, dan itu dinyatakan di muka kepada Work Owner.

`SetLoginSurveyor_act` menetapkan satu kata sandi tetap lalu memanggil `GCNMCreateOperator`;
pesan suksesnya menyebut kata sandi itu terang-terangan. Nilainya **tidak direproduksi** di
kode, di dokumen ini, maupun di log (`D-69`).

| Penghalang | Dasarnya |
|---|---|
| Tidak ada tempat menerbitkan akunnya | tidak ada operator Pega; `F-3` belum ada (`R-14`, `ADR-0024`) |
| Kata sandinya tidak boleh ditulis di artefak yang di-commit | `D-69` |
| Nilai bisnis tidak boleh di-hardcode | `D-15` |
| Mengumumkan kata sandi bagi akun yang tidak diterbitkan adalah **keterangan yang salah** | petugas akan menyampaikannya, dan surveyor tidak akan dapat masuk |

Baris master tetap tersimpan — itulah yang dikerjakan layar ini. Ketiadaan akunnya
**dinyatakan di layar** pada dua tempat (peringatan pada form penambahan, dan kaki halaman)
dan dicatat di log lewat `noteAccountNotIssued`, bukan tersamar.

Perlakuan yang sama dipakai **Master Bengkel**, yang menghadapi `GCNMCreateOperator` yang
sama persis dan memutuskan hal yang sama.

### 38.5 Keputusan 5 — `STSLOGIN` dan `LOGINLEADER` tetap diturunkan, tetapi DITAMPILKAN

Keduanya ditulis sistem lama dan **tidak muncul sekali pun** di
`Section/BrowseLoginSurveyor-Section.xml` — dibuktikan dengan pencarian
`TempLoginSurvey.ObjectName` dan `TempLoginSurvey.NamaPasien` di seluruh section: nol
kemunculan.

| Hal | Ketetapan |
|---|---|
| Cara diisi | `STSLOGIN` selalu `"Member"` pada penambahan; `LOGINLEADER` diturunkan dari leader milik pengguna yang menyimpan |
| Pada penyuntingan | **dipertahankan apa adanya** dari baris tersimpan, tidak ditimpa |
| Dapat dikirim klien? | **tidak** — `DisallowUnknownFields` menolaknya sebagai permintaan cacat |
| Dapat dilihat? | **ya**, sebagai keterangan baca-saja pada form |

Ditampilkannya adalah satu-satunya penyimpangan dari Pega di sini, dan alasannya: nilainya
menentukan **peran dan tim** seseorang, dan menyembunyikan hal yang tersimpan tidak membuatnya
tidak tersimpan. Baris ber-`LOGINLEADER` kosong tidak dapat dibedakan dari yang bertim bila
tidak ditampilkan.

`TestSavePreservesUnknownLoginStatus` menjaga nilai di luar `"Member"` tidak ditimpa: tidak
satu pun rule menuliskannya (`R-16`), tetapi baris lama dapat memuatnya — dan menimpanya
berarti diam-diam mengubah peran seseorang pada baris yang petugas hanya ingin perbarui nomor
teleponnya.

### 38.6 Keputusan 6 — LOGINLEADER diturunkan dari PENYIMPAN

`GetLoginLeaderSurveyor` mencari baris milik **pengguna yang menekan Simpan**, lalu mengambil
kolom `LOGINLEADER`-nya — bukan login pengguna itu sendiri.

Akibatnya: surveyor yang menambahkan rekannya memberi rekan itu **leader yang sama dengan
dirinya**, bukan menjadi leader rekannya. Dijaga `TestCreateInheritsLeaderOfTheSaver`, yang
pesan gagalnya menyebut kemungkinan terbaliknya secara eksplisit.

Pengguna yang tidak punya baris di tabel ini menghasilkan `LOGINLEADER` kosong tanpa satu pun
galat. Ditiru apa adanya (`P-5`) — menolaknya akan menghalangi petugas admin menambahkan
surveyor sama sekali. Yang **ditambahkan** hanyalah `noteLeaderMissing` bertingkat `Warn`.

Konsekuensi arsitektur: `Caller` di modul ini **bukan sekadar pengisi log** seperti pada
Master Kategori dan Master Tipe Sparepart. Identitas pemanggil yang hilang membuat setiap
baris baru lahir tanpa tautan tim, sehingga handler menolak keras bila konteksnya kosong.

### 38.7 Keputusan 7 — cakupan daftar adalah REKONSTRUKSI, dan itu dinyatakan

Grid layar lama terikat page list klipboard `LoginMemberSurvey.pxResults`, dan **rule yang
mengisinya tidak ada di antara 2.634 berkas export** (`R-16`). Yang ada hanyalah
`SetLoginSurveyorValue_act`, yang menembak `pyReportContentPage` dan menyaring satu login
untuk tombol Ubah.

Yang dipakai: bentuk kueri yang **benar-benar ada**, tanpa penyaring — sehingga daftarnya
memuat seluruh baris entitas itu.

**Bacaan lain yang mungkin, dan tidak dapat dibantah maupun dibuktikan**: daftarnya disaring
`LOGINLEADER` = leader milik pengguna yang membukanya, sehingga seorang leader hanya melihat
anggotanya. Tiga hal menunjuk ke arah itu — nama page list-nya (`LoginMemberSurvey`),
`STSLOGIN` yang selalu `"Member"`, dan keberadaan `GetLoginLeaderSurveyor`.

Perbedaan keduanya menentukan **siapa yang boleh menyunting login milik tim lain**. Itu
pertanyaan terbuka untuk Work Owner, dan ia dinyatakan di lima tempat: `masterlogin.Filter`,
`login_list`, `Handler.List`, `SurveyorLoginPage`, dan `login_count_orphan_leader` pada
`-periksa`. `TestListReturnsEveryRow` mengikatnya supaya perubahannya kelak menjadi keputusan
yang terlihat, bukan pergeseran yang tidak disadari.

### 38.8 Keputusan 8 — kewajiban isian DITEGAKKAN di server

`pyRequired = true` pada Nama, Email, dan Telp; `false` pada Alamat. Di Pega ketiganya hanya
ditandai di layar — `CNMInsertMstLoginSurveyor_act` sendiri hanya menolak LOGIN yang kosong.

Penegakan di server **ditambahkan**. Alasannya bukan kerapian: surel pada baris ini adalah
alamat yang dipakai memberi tahu surveyor tentang penugasannya, dan baris tanpa surel gagal
diam-diam — tidak ada galat, hanya pemberitahuan yang tidak pernah sampai.

Baris lama yang sudah kosong tetap **dibaca apa adanya**; penolakan hanya terjadi saat
barisnya disimpan ulang. Jumlahnya dilaporkan `-periksa` lewat `login_count_missing_contact`,
supaya diketahui sebelum petugas menemukan bahwa baris yang selama ini tersimpan tidak lagi
dapat disimpan.

**Bentuk surel TIDAK diperiksa.** Tidak ada satu pun rule di export yang memeriksanya, dan
menambahkannya berarti menolak alamat yang selama ini diterima (`P-5`).
`TestCheckDoesNotValidateEmailShape` menjaga ketiadaan itu tetap disengaja.

### 38.9 Keputusan 9 — satu tabel, satu penulis, dan tanpa DELETE

| Tabel | Perlakuan | Penulisnya |
|---|---|---|
| `POOLDATA.MST_LOGIN_SURVEYOR` | **ditulis** modul ini | modul ini |

Tidak ada tabel lain yang disentuh — modul master paling sederhana di aplikasi ini.

**Tanpa DELETE**: tidak satu pun rule di export menghapus baris tabel ini, dan `D-66`
melarang penghapusan fisik data bernilai bisnis. `TestNoDeleteRoute` menjaganya.

Yang harus disadari, dan dinyatakan di kaki layar: tabelnya **juga tidak punya penanda
aktif**, sehingga tidak ada cara menyatakan sebuah login sudah tidak berlaku — bukan lewat
penghapusan, dan bukan lewat penonaktifan. Itu keterbatasan tabelnya, dan ia dinyatakan
alih-alih ditutupi dengan tombol yang mengarang kolom baru.

### 38.10 Penamaan: kenapa `SurveyorLogin`, bukan `Login`

`Login` sudah dipakai untuk hal yang sama sekali berbeda di aplikasi ini: masuknya pengguna
ke sistem. Menyamakan keduanya akan membuat dua hal yang tidak berhubungan terlihat
berhubungan — dan pada modul ini kekeliruan itu paling mahal, karena menyimpan baris di sini
**tidak** menerbitkan akun siapa pun (§38.4).

Nama folder mengikuti `D-81`: `internal/masterlogin` dan `src/modules/master-login`, dari
nama modul yang disebut Work Owner ("Master Login"). Isinya berbahasa Inggris sesuai `D-80`.

Nama field JSON mengikuti **label di layar** — `nama`, `login`, `email`, `telp`, `alamat` —
bukan nama kolom. Dua yang terakhir memakai nama kolomnya karena keduanya tidak punya label
di layar mana pun.

### 38.11 Yang TIDAK dibawa dari sistem lama

| Yang ditinggalkan | Dasarnya |
|---|---|
| `GCNMCreateOperator` beserta kata sandi bersamanya | §38.4 |
| Pesan sukses yang menyebut kata sandi | `D-69`, dan ia keterangan yang salah |
| Penyaring `{ASIS:InputLogin.IDIndex}` yang dirangkai dari teks tanpa pelolosan | `03-CURRENT-ARCHITECTURE.md` §4.5 |
| Alias `"SurveyName"`, `"SurveyorID"`, `"Ekst"`, `"BodyLetterTo"`, `"ObjectName"`, `"NamaPasien"` | `D-19`, `03-CURRENT-ARCHITECTURE.md` §4.2 |
| Penurunan LOGIN di sisi klien | §38.1 |
| Pemeriksaan ganda terhadap tabel operator Pega | §38.3 |

### 38.12 Utang yang disadari

| Utang | Kenapa dibiarkan |
|---|---|
| Cakupan daftar adalah **rekonstruksi** | Rule pemuatnya tidak ada (`R-16`). Dinyatakan di lima tempat dan diikat uji |
| Pemeriksaan login ganda adalah **rekonstruksi** | Sasaran aslinya tidak ada di sistem baru. Dinyatakan di empat tempat |
| DDL tabel belum ada | `R-08`. Keempat batas panjang (`100/100/50/250`) **asumsi** |
| Tidak ada constraint unik pada `LOGIN` | `R-08`, `D-63`. Balapan **dipersempit** kunci tabel; baris kembar yang sudah ada dilaporkan `-periksa` |
| Tidak ada cara menonaktifkan sebuah login | Tabelnya tidak punya kolomnya. Dinyatakan di kaki layar |
| Siapa mengubah apa tidak tersimpan | Tabelnya tidak punya kolom pelaku maupun waktu. Hanya masuk log — bukan pengganti `S-5` |
| Paginasi **menomori halaman**, Pega memakai Next/Previous | `pyPageSizeOther=15` ditiru; modenya diseragamkan dengan seluruh layar master lain agar tidak ada dua gaya paginasi |
| `MAX_NAME_LENGTH` diulang di dua tempat | Dijaga `TestMaxNameLengthMatchesFrontendForm` |
| Kode galat dipetakan modul ini sendiri | `TKT-F1-004` masih terhalang |
| Jalur tanpa `/v1` | Kontrak yang ada belum memakainya |
| `eslint` tidak dijalankan | Repo ini masih tidak punya `eslint.config.*` — dan tidak punya skrip `lint` |

---

## 39. Modul Master Reas (2026-09-22)

Layar `DataMemberReas` (MENU_ID 35) atas `POOLDATA.T_REINSURER`. Tujuh kolom, **enam**
dibaca, **nol** isian — **satu-satunya layar master yang BACA-SAJA**.

Work Owner menjawab keempat pertanyaan pembuka dengan **"Ikuti PEGA"**. Bab ini mencatat apa
artinya itu untuk tiap keputusan — termasuk satu penambahan yang disadari, dan satu yang
tidak dapat dipastikan karena artefaknya hilang.

### 39.1 Keputusan 1 — modul ini TIDAK MENULIS

Keputusan terbesar di modul ini, dan ia lahir dari satu penelusuran: **siapa yang memanggil
`UPDATEREAS`?**

```
Database/UPDATEREAS.prc              prosedur upsert-nya
  ← RDB List/UpdateEmailReas-SQL.xml     satu-satunya Connect-SQL yang memanggilnya
      ← Activity/UpdateDetailPLA2-Act.xml    layar detail PLA
      ← Activity/UpdateDetailDLA2-Act.xml    layar detail DLA
```

Pemindaian seluruh `Activity/`, `Section/`, `Flow Action/`, `Data Transform/`, `DataPage/`,
dan `Report Definition/` untuk `UpdateEmailReas` menghasilkan **tepat dua berkas**, dan
keduanya adalah layar detail PLA/DLA.

Ditambah harness-nya sendiri yang hanya memuat grid dan satu tombol **Refresh**
(`pyButtonLabel Refresh` pada `ListMemberReas`), kesimpulannya: baris reasuransi lahir dan
berubah sebagai **efek samping alur PLA/DLA** (`B-9`), bukan lewat pemeliharaan master.

**Yang ini cegah**, dan sebabnya bukan kehati-hatian umum: `LOGIN` pada tabel ini menentukan
**klaim mana yang dilihat seorang mitra reasuransi** — lima kueri inbox menyaringnya. Layar
tulis yang tidak pernah diminta siapa pun, pada kolom yang menentukan visibilitas data lintas
badan hukum, adalah risiko tanpa imbalan.

**Keterbatasan buktinya dinyatakan.** Section `BrowseListMemberReas` tidak ada di export
(`R-16`), sehingga ini **rekonstruksi**, bukan hal yang terbukti mustahil. Penambahan jalur
tulis kelak menyentuh `Repo.Insert`/`Update` dan rutenya; domainnya tidak perlu berubah.

### 39.2 Keputusan 2 — `COUNTRYID` tidak dibaca dan tidak dikirim

`UPDATEREAS.prc` menyisipkan tujuh kolom, salah satunya `COUNTRYID`, yang diterjemahkan dari
`COUNTRY`:

```sql
SELECT ID INTO NEGARA_ID FROM COUNTRY WHERE COUNTRY = tCOUNTRY;
INSERT INTO POOLDATA.T_REINSURER
  (REINSURERID, REINSURERNAME, LOGIN, EMAIl, COUNTRY, COUNTRYID, TYPE) VALUES (...);
```

Lalu **tidak dibaca satu pun rule di seluruh export** — tidak oleh SELECT mana pun, tidak
oleh laporan, dan tidak oleh dokumen PLA/DLA. Yang dibaca dokumen adalah `COUNTRY`, bukan
`COUNTRYID` (`GetDataPreDLA`, `BrowseAllDataXOL_PLA`).

Ia karena itu tidak di-SELECT dan tidak masuk DTO. Membawa kolom yang tidak ada pembacanya
berarti mengarang kegunaan yang tidak dapat ditunjukkan — dan pada layar, menampilkannya
akan memancing pertanyaan yang tidak ada jawabannya.

Dijaga uji: `TestCountryIDNotSelected` (sqlstore) dan `TestListMengirimEnamKolom...` (http).

### 39.3 Keputusan 3 — kunci baris TIGA kolom, bukan kode reas

`UPDATEREAS` memeriksa keberadaan baris dengan ketiganya sekaligus:

```sql
SELECT COUNT(1) INTO REAS FROM T_REINSURER
 WHERE REINSURERID = tREINSID AND REINSURERNAME = tREINSNAME;
SELECT COUNT(1) INTO JUMLAH_TIPE FROM T_REINSURER
 WHERE REINSURERID = tREINSID AND REINSURERNAME = tREINSNAME AND TYPE = tTYPE;
```

Ini **berbeda dari Master Login**, yang kuncinya satu kolom — dan perbedaannya menentukan
bentuk layar: satu perusahaan reasuransi muncul beberapa kali di daftar.

Akibat langsungnya di frontend: `rowKey` menggabungkan **kode + nama + tipe**. Memakai kode
reas sendirian akan membuat tiga baris milik satu perusahaan berbagi kunci yang sama, dan
React menganggap ketiganya satu baris — hanya salah satunya tampil.

Pemisahnya `\u001f` (unit separator), bukan tanda hubung: nama perusahaan reasuransi memang
memuat tanda baca, dan `RE-001 + Andalas` tidak boleh menghasilkan kunci yang sama dengan
`RE + 001-Andalas`. Dijaga `TestNaturalKeyTidakBertabrakanLewatPemisah`.

**Keunikannya tidak dijamin basis data.** Tidak ada DDL-nya (`R-08`), dan bukti mengarah
sebaliknya — lihat §39.6. Karena itu `NaturalKey` dipakai sebagai kunci baris di layar, bukan
sebagai jaminan.

### 39.4 Keputusan 4 — kolom `TYPE` ditampilkan, artinya TIDAK diterjemahkan

**PENAMBAHAN terhadap SELECT lama, dan dinyatakan begitu.** `BrowseEmailReas` memakai `TYPE`
hanya sebagai penyaring, tidak pernah meng-SELECT-nya.

Alasan menambahkannya: tanpa kolom itu, satu perusahaan dengan tiga jenis dokumen muncul
sebagai **tiga baris yang terlihat kembar**, dengan surel berbeda-beda dan tanpa satu pun
keterangan mengapa. Daftar seperti itu tidak dapat dipercaya pembacanya.

Alasan **tidak** menerjemahkannya: yang terbukti hanyalah bahwa ia dicocokkan dengan karakter
pertama nomor dokumen (`substr(a.NODLA,0,1) = TYPE`). Tidak ada master, tidak ada daftar
nilai sah, dan tidak ada satu pun rule yang menerjemahkannya menjadi label (`R-16`).

Satu-satunya hal yang **dihitung** adalah penanda `cadangan` (`TYPE = '1'`), dan itu pun
karena artinya terbukti dari dua tempat sekaligus:

| Bukti | Isi |
|---|---|
| `BrowseEmailReas` | `... and (type = {TempReasPLA.NoPLA} or type = '1')` — dipakai bila jenisnya tidak ada |
| `UPDATEREAS.prc` | `... AND TYPE = '1'` lalu `UPDATE ... SET TYPE = tTYPE` — baris `'1'` dinaikkan menjadi tipe yang diminta |

Penanda itu **dihitung server**, bukan di layar: menaruh pengetahuan bahwa `'1'` punya arti
khusus di dua tempat berarti dua tempat harus ikut berubah bila artinya berubah.

### 39.5 Keputusan 5 — penyaring `cari` ada di server, tetapi layar memakai `DataTable`

Endpoint menerima `?cari=`, menyaring **empat** kolom (kode, nama, login, email) lewat
`UPPER(TRIM(...)) LIKE ... ESCAPE '\'`. Layar tidak memakainya; ia memakai pencarian bawaan
`DataTable`, sama seperti seluruh layar master lain.

Alasannya sama dengan Master Login: pencarian kedua di kepala halaman hanya membingungkan,
dan perpindahan ke penyaringan sisi server dilakukan bersama paginasi sisi server
(`TKT-U2-001`), bukan sendirian.

`COUNTRY` dan `TYPE` sengaja **tidak** ikut dicari — yang pertama penggolongan, yang kedua
satu karakter sehingga akan mencocokkan hampir setiap baris. Dijaga
`TestListMenyaringEmpatKolom`, yang secara eksplisit memastikan mencari "Singapura"
menghasilkan **nol** baris.

Kata kunci dibatasi `MaxKeywordLength = 100` dan yang melampauinya ditolak **400**, bukan
dijawab daftar kosong: daftar kosong terbaca sebagai "tidak ada datanya", dan pengguna tidak
punya cara membedakan keduanya (`10-API-STRATEGY.md` §4).

### 39.6 Keputusan 6 — enam pemeriksaan `-periksa`, dua di antaranya baru

Dua pemeriksaan tidak ada padanannya di modul mana pun, dan keduanya lahir dari membaca
kueri lamanya — bukan dari daftar periksa generik.

**`reas_count_shared_login`** — satu `LOGIN` dipakai beberapa `REINSURERID`.

Sistem lama **mengakui** keadaan ini mungkin terjadi, dan menyelesaikannya dengan memilih
sembarang satu:

```sql
-- GetPNCList_PLA1
(select reinsurerid from pooldata.t_reinsurer
  where login = {OperatorID.pyUserIdentifier}
  order by reinsurerid desc fetch next 1 row only)
```

`order by ... desc fetch next 1 row only` adalah pengakuan bahwa hasilnya dapat lebih dari
satu. Karena kolom itu menentukan klaim mana yang dilihat seorang mitra, setiap kejadiannya
berarti seseorang **berpotensi melihat klaim milik mitra lain** — dan layarnya tampil normal
tanpa satu pun tanda (`R-20`).

**`reas_count_without_fallback`** — perusahaan tanpa baris `TYPE '1'`.

Keadaan ini **dapat tercipta sistem lama sendiri**: `UPDATEREAS` menaikkan baris `'1'`
menjadi tipe yang diminta alih-alih menyisipkan baris baru, sehingga cadangannya habis
terpakai. Sesudah itu, jenis dokumen lain tidak menemukan surel tujuannya sama sekali.

Ditambah empat yang lazim: jumlah baris, kunci alami kembar, `LOGIN` kosong, `EMAIL` kosong.

**Hak yang diminta ke DBA adalah BACA saja**, dan itu disebut eksplisit di keluaran
`-periksa`. Meminta hak tulis untuk tabel yang tidak ditulis berarti meminta kewenangan yang
menganggur — pada tabel yang menentukan ke mana pemberitahuan klaim dikirim, itu risiko tanpa
imbalan.

### 39.7 Keputusan 7 — keadaan yang gagal dalam diam DINYATAKAN di layar

Tiga keadaan pada tabel ini tidak menimbulkan galat apa pun di sistem lama, dan karenanya
tidak pernah terlihat siapa pun:

| Keadaan | Akibatnya | Perlakuan di layar |
|---|---|---|
| `LOGIN` kosong | mitra tidak akan pernah melihat klaimnya sendiri | sel bertulis **"belum ada"**, bukan kosong |
| `EMAIL` kosong | dokumen PLA/DLA terbit, tercatat terkirim, tidak sampai | sel bertulis **"belum ada"**, bukan kosong |
| `LOGIN` dipakai dua kode reas | mitra berpotensi melihat klaim mitra lain | dinyatakan di kaki halaman, dan dicacah `-periksa` |

Sel kosong tidak menunjukkan satu pun dari ketiganya. `COUNTRY` kosong diperlakukan berbeda —
ia ditampilkan sebagai `—`, karena baris yang lahir dari `GetListDataLoginReas` memang
disisipkan tanpa kolom itu dan itu keadaan yang wajar, bukan yang perlu ditindaklanjuti.

### 39.8 Keputusan 8 — tanpa `Caller`, tanpa `Clock`, tanpa `Logger` di usecase

Modul master lain menerima ketiganya. Modul ini tidak, dan ketiadaannya adalah akibat
langsung dari modulnya yang hanya membaca:

| Yang tidak diterima | Kenapa |
|---|---|
| `Caller` | tidak ada baris yang diturunkan dari identitas pemanggil, dan tidak ada perubahan yang perlu dicatat pelakunya |
| `Clock` | `T_REINSURER` tidak punya satu pun kolom waktu |
| `Logger` (di usecase) | tidak ada peristiwa yang perlu dicatat; transport tetap mencatat kegagalan |

Menerimanya "untuk jaga-jaga" berarti menerima bahan yang tidak pernah dipakai, dan itu
menyesatkan pembaca berikutnya — ia akan mencari jalur tulis yang memang tidak ada.

Galat portal **tidak** dicatat sebagai `Error` di transport: permintaan tanpa header portal
adalah kesalahan klien yang terjadi setiap kali seseorang membuka aplikasi sebelum memilih
portal, dan mencatatnya akan menenggelamkan kegagalan sungguhan. `ErrNotReady` dikecualikan
dari pengecualian itu — ia 503, dan penyebabnya memang pekerjaan administrator.

### 39.9 Yang tidak diputuskan, dan menunggu pihak lain

| Hal | Milik siapa | Kenapa tertahan |
|---|---|---|
| Daftar negara yang sah | DBA | tabel `COUNTRY` **nol pembaca** di export; isinya tidak pernah dilihat (`R-16`, `R-08`) |
| Arti tiap nilai `TYPE` | Work Owner | tidak ada master maupun rule yang menerjemahkannya |
| Apakah layar lamanya punya tombol simpan | Tim Pega | section `BrowseListMemberReas` hilang dari export (`R-16`) |
| Kewenangan menu — MENU_ID 35 hanya untuk grup `IT` | `TKT-F3-005` | tabel peran dapat dibangun tetapi belum dapat diisi |

---

## 40. Modul Master Pasal AI — ditunda (2026-09-22)

Layar `DetailMasterPasalAI` (MENU_ID 36). Bab ini mencatat **keputusan untuk tidak
membangunnya sekarang**, beserta tiga jalan keluar yang dipertimbangkan dan ditolak.

### 40.1 Keputusan — modul TIDAK dibangun; section isinya diminta ke Tim Pega

Harness rujukannya hanya kerangka: ia menggambar judul "Detail Pasal AI" lalu menyerahkan
seluruh layar ke `Section DetailMasterPasalAI`, dan **section itu tidak ada di export**
(`Harness/DetailMasterPasalAI-harness.xml:2622`).

Yang hilang bersamanya adalah **seluruh hal yang menentukan modul**: tabel yang dibaca dan
ditulis, kolom grid, tombol, dan aturan validasi. Sapuan menyeluruh atas Connect-SQL,
Activity, Report Definition, Data Page, When, Data Transform, Flow, Flow Action, Connect
REST, Service REST, dan `Database/` **tidak menemukan satu pun** objek yang mempertemukan
"pasal" dengan "AI" — termasuk nol kemunculan untuk `M_DATA_PASAL_AI`, `PASAL_AI`,
`AI_PASAL`, dan `MST_PASAL`.

Work Owner memilih **meminta section-nya ke Tim Pega** (2026-09-22). Permintaannya disusun
sebagai `docs/permintaan-artefak-pega.md`, mengikuti bentuk yang ditetapkan `D-39` — export
ulang berbasis **Product rule** dengan opsi *include dependent rules*, bukan pemilihan
manual per rule, karena nama rule yang dibutuhkan justru belum diketahui.

### 40.2 Tiga jalan keluar yang ditolak, dan kenapa

| Jalan keluar | Kenapa ditolak |
|---|---|
| **Bangun di atas tabel baru yang dirancang sendiri**, kembaran struktural Master Pasal Kerugian | Menuntut migrasi DDL yang membuat tabel bisnis di **empat basis data entitas**, menempuh `D-63` (permintaan tertulis → persetujuan Work Owner → pelaksanaan DBA), demi tabel yang belum tentu boleh ada. Dan modulnya **tidak akan punya baseline Pega**, sehingga tidak dapat lulus gerbang 1 (`D-42`) |
| **Pakai ulang `POOLDATA.V_M_DATA_PASAL`** milik Master Pasal Kerugian | Tabel itu **tidak punya penanda AI** — tidak ada kolom, kategori, maupun penyaring yang membedakannya. Hasilnya dua butir menu yang menampilkan data yang sama persis, dan petugas tidak punya cara mengetahui mana yang benar |
| **Bangun layar kosong yang meniru harness apa adanya** | Harness-nya memang hanya judul. Layar yang hanya memuat judul lebih buruk daripada butir menu bertanda "belum tersedia": ia tampak sudah jadi |

Ketiganya punya cacat yang sama: **mengarang tabel dan kolom**, yang justru dilarang selama
proses bisnis aslinya masih dapat dipelajari — dan di sini ia masih dapat, cukup dengan satu
berkas dari Tim Pega.

### 40.3 Keadaan yang benar bagi butir menu ini hari ini

`Master Pasal AI` tetap tampil di menu, **tidak dapat diklik, bertanda "belum tersedia"** —
sama seperti 62 butir menu lain yang belum punya layar (keputusan Work Owner 2026-09-18).
Tidak ada baris yang ditambahkan ke `MENU_ROUTES`.

Dengan begitu kemajuan migrasi tetap terbaca langsung dari layar, dan tidak ada layar
setengah jadi yang menyesatkan.

### 40.4 Satu koreksi fakta di dalam kode

`backend/internal/menu/repo/memory/sample.go` menyatakan **sembilan** `MENU_PROGRAM`
menunjuk harness yang tidak ada di export. Dua di antaranya — `DataMemberReas` dan
`DetailMasterPasalAI` — **harness-nya sudah masuk** pada 2026-09-22. Angkanya dikoreksi
menjadi **tujuh**.

`frontend/src/app/menu/registry.ts` memuat pernyataan yang sama dan **sudah diperbaiki sesi
paralel** yang mengerjakan Master Reas; berkas itu **tidak disentuh** di sesi ini.

Yang **tidak** disunting: `docs/catatan-pengembangan.md` §15.4, yang memuat daftar sembilan
itu. Ia **rekaman temuan bertanggal**, bukan pernyataan yang berlaku — perlakuan yang sama
dengan entri lama pada Decision Log (`CLAUDE.md` §7).

### 40.5 Yang masih tertahan

| Hal | Milik siapa | Kenapa tertahan |
|---|---|---|
| **Section `DetailMasterPasalAI`** beserta rule yang dirujuknya | Tim Pega | hilang dari export (`R-16`); tanpa ini tabel dan kolomnya tidak dapat diketahui siapa pun |
| Apakah layarnya **pernah selesai dibuat** di Pega | Tim Pega | dibuat 2023-05-04, disunting sekali 2023-05-05, lalu tidak pernah disentuh; pembungkusnya tanpa tombol; label `.Detail Pasal Kerugian` masih tertinggal dari asalnya |
| Kewenangan menu — `MENU_ID 36` hanya untuk grup `IT` | `TKT-F3-005` | tabel peran dapat dibangun tetapi belum dapat diisi |

### 40.6 Section diterima di tengah sesi — keputusan §40.1 TETAP, alasannya berubah

`Section/DetailMasterPasalAI_sect.xml` diterima 2026-09-22, beberapa jam setelah permintaan
disusun. Ia menjawab pertanyaan §40.5 baris kedua: **layarnya memang selesai dibuat** —
dibuat 2023-05-16, di-commit 2023-07-14, dua bulan sesudah harness-nya.

Keputusan untuk tidak membangun modulnya **tidak berubah**, tetapi alasannya bergeser dan
menyempit:

| | Sebelum section masuk | Sesudah |
|---|---|---|
| Yang tidak diketahui | tabel · kolom · tombol · aturan · bahkan apakah layarnya ada | **hanya tabel dan tiga nama kolom** |
| Yang diminta | seluruh section isi beserta dependensinya | **satu activity: `GetListPasalAI`** |
| Bentuk layar | tidak diketahui sama sekali | **terbaca lengkap** — lihat `catatan-pengembangan.md` §38.8 |

### 40.7 Satu dugaan saya yang terbantah, dan itu membenarkan keputusan §40.2

Saya menduga Master Pasal AI adalah **kembaran CRUD** Master Pasal Kerugian — dugaan yang
wajar, karena section-nya memang Save-As dari `BrowsePasalDeatailMaster`.

**Ia bukan.** Ia layar **baca-saja**: `pyEditingMode = readOnly`
(`Section/DetailMasterPasalAI_sect.xml:4405`), tanpa Tambah, Simpan, Ubah, maupun Hapus.
Pengembangnya mengklon section CRUD itu lalu **memangkasnya** menjadi layar pencarian.

Inilah pembenaran paling konkret bagi §40.2: seandainya jalan keluar "bangun sebagai
kembaran Pasal Kerugian" diambil, hasilnya adalah modul CRUD lengkap dengan tabel karangan,
migrasi DDL di empat basis data entitas, dan jalur tulis yang **tidak pernah ada di sistem
lama** — seluruhnya salah, dan seluruhnya baru ketahuan setelah terlanjur dijalankan DBA.

### 40.8 Kenapa bentuk layar yang sudah lengkap pun belum cukup untuk membangun

Tiga kolomnya terikat properti `.City`, `.CityID`, dan `.District` — **nama yang tidak
mencerminkan isinya**, sisa Save-As berlapis dari `BrowseDetailSuveryors` (2017).

Pola pengisiannya sudah pasti, dibuktikan dari layar sejenis yang lengkap di export:
`Activity/SearchDataMasking-Act.xml` mengisi `TempDetailData` yang **sama** lewat
`RDB List/SearchMasking_SQL-SQL.xml`, yang mengaliaskan kolom tabel ke properti yang dipakai
ulang (`CABANG as "ProvinceID"` atas `POOLDATA.MST_PROTEKSI_DATA_PNC`).

Jadi `GetListPasalAI` hampir pasti mengaliaskan tiga kolom menjadi `"City"`, `"CityID"`, dan
`"District"`. **Dan justru itulah yang membuat menebak mustahil**: nama propertinya tidak
memberi satu pun petunjuk tentang kolom aslinya, sehingga tidak ada cara menurunkan tabelnya
dari apa pun yang sudah ada di tangan.

Seandainya modul tetap dibangun sekarang, satu-satunya adapter yang dapat ditulis adalah
penyimpanan memori. Layarnya akan hidup di pengembangan dan **kosong di produksi** — bentuk
kegagalan yang paling mahal, karena ia tampak selesai.

### 40.9 Dua sisa Save-As yang dinyatakan MATI

Dicatat supaya permintaan berikutnya tidak menempuh jalur buntu:

| Sisa | Kenapa mati |
|---|---|
| `BrowseVDSurveyors_RD` atas `ASM-FW-GCNMFW-Int-V_D_SURVEYORS` (`:4487`) | ada di dalam `pyGridProps`, tetapi grid-nya ber-`pySourceType = Property` (`:4476`) — wiring Report Definition itu tidak dipakai saat berjalan |
| `pyPreGridUpdate` (`:4485`) · `pyPostGridUpdate` (`:4471`) | ditandai tidak ada oleh section itu sendiri — `pyGridPreActivityExists = false` (`:4491`), `pyGridPostActivityExists = false` (`:4431`) |

### 30.8 Koreksi setelah dicoba di lingkungan nyata (2026-09-23)

Layar ini gagal memuat daftarnya di lingkungan Work Owner. Penelusurannya mengoreksi **dua
hal yang saya tulis sendiri di §30**, dan menemukan satu penghalang yang ada di basis data.

**Koreksi pertama: `POOLDATA.SPAREPART_HE` adalah VIEW, bukan tabel.** Seluruh §30 dan
banner `mastersparepart.sql` menyebutnya tabel. Oracle menyebutnya view secara eksplisit
lewat `ORA-04063: view "POOLDATA.SPAREPART_HE" has errors`. Dugaan pada banner itu —
"keduanya SATU sumber (view atas JSON)" — ternyata benar, tetapi ditulis sebagai kemungkinan
padahal sudah dapat dipastikan.

**Koreksi kedua: `PENYIMPANAN=memori` TIDAK berarti master memakai penyimpanan memori.**
`needsOracle()` pada `cmd/claimpnc/main.go` bernilai benar bila `PENYIMPANAN=oracle`
**atau** `IDENTITAS_ADAPTER=hcq`, dan cabang itulah yang memilih adapter SQL untuk SELURUH
master. Dengan `IDENTITAS_ADAPTER=hcq`, `PENYIMPANAN=memori` hanya mengatur tabel pengguna
dan sesi.

Saya sempat menguji modul ini pada instans terpisah dengan `IDENTITAS_ADAPTER=fake` dan
menyimpulkan "endpoint-nya bekerja". Kesimpulan itu **tidak berlaku** untuk lingkungan Work
Owner: pengujian itu menempuh cabang memori, sedangkan lingkungan nyata menempuh cabang SQL.
Dua jalur yang berbeda, dan hanya satu yang diuji.

**Penghalangnya, terverifikasi `claimpnc -periksa`:**

| Objek | Keadaan |
|---|---|
| `POOLDATA.SPAREPART_HE` | **ORA-04063** — ada, definisinya RUSAK |
| `POOLDATA.M_SPAREPART_HE_BU` | **ORA-00942** — tidak ada, atau tidak diberikan ke akun aplikasi |
| `POOLDATA.GCNM_M_SPAREPART_CATEGORY` | terbaca, 9 baris |
| `POOLDATA.GCNM_M_SPAREPART_TYPE` | terbaca, 11 baris |
| `POOLDATA.SPAREPART_HE_VIN_KEY` | terbaca, 12 baris |

Bukan kueri modul ini yang salah: **tiga modul berbeda** menabrak `ORA-04063` yang sama pada
posisi berbeda — `sparepart_check_table` (472), `grouping_count_orphan_part` (121), dan
`category_count_orphan_sparepart` (35). Tetangga se-keluarga terbaca normal.

Keduanya patut diduga satu sebab: view yang merujuk objek yang hilang tidak dapat
dikompilasi. Kepastiannya menuntut `all_errors` dan `all_dependencies`, dan itu milik DBA.

**Akibat yang belum dipastikan:** Pega membaca objek yang SAMA, sehingga layar Master
Sparepart di Pega semestinya ikut gagal hari ini. Bila ternyata Pega normal, dugaan
"objeknya hilang" gugur dan yang tersisa adalah persoalan hak akses akun aplikasi — dua
jalan yang berbeda, dan tim Pega yang dapat memisahkannya dalam satu menit.

**Yang diperbaiki di kode.** `checkSparepart` tidak lagi berhenti pada kegagalan pertama: ia
melanjutkan ke `checkSparepartStore` yang membaca `M_SPAREPART_HE_BU` sendirian. Tanpa itu,
pemeriksa hanya melaporkan "tidak dapat dibaca" dan menyarankan meminta hak akses —
saran yang menyesatkan untuk ORA-04063, dan yang akan menghabiskan satu putaran percakapan
dengan DBA untuk hal yang bukan penyebabnya. Keduanya kini dibedakan beserta kueri katalog
yang menjawabnya.

**Yang TIDAK diubah:** kueri modul ini, karena tidak ada bukti ia salah. Ia menyebut kolom
yang dibaca dari `BrowseSparepartHE_RD` dan belum pernah benar-benar dijalankan terhadap
objek yang sehat. Begitu view-nya diperbaiki, barulah asumsi tipe `PROD_DATE` dan
`TGL_UPDATE_HARGA` pada §30.6 dapat diuji — dan itu pemeriksaan yang masih tertunda, bukan
yang sudah lulus.

**Satu keterbatasan yang terlihat karenanya.** Layar hanya menampilkan pesan umum "Daftar
sparepart tidak dapat dimuat" untuk kode galat apa pun yang tidak dikenalinya, sehingga
masalah rute, sesi, dan basis data terlihat sama persis bagi pengguna. Penyebab sebenarnya
hanya ada di log backend. Ia berlaku di seluruh layar master, bukan hanya di sini, dan
perbaikannya menyentuh kode bersama milik modul yang sudah selesai — dicatat sebagai utang,
tidak dikerjakan sepihak.

### 40.10 Activity diterima — keputusan tetap, penghalang tinggal satu

`Activity/GetListPasalAI_act.xml` diterima 2026-09-23. Penghalangnya menyempit sekali lagi:

| | Sebelum activity masuk | Sesudah |
|---|---|---|
| Yang tidak diketahui | tabel · nama kolom · perilaku pencarian · paginasi | **hanya nama tabel** |
| Yang diminta | satu activity | **dua Connect-SQL** — `GetListDataPasalAI`, `CountDataPasalAI` |

Modul tetap **belum dibangun**, dan alasannya kini tinggal satu kalimat: **repositori tidak
dapat ditulis tanpa nama tabel.** Seluruh sisanya — kolom, penyaring, paginasi — sudah
terbaca.

### 40.11 Tiga keputusan desain yang sudah dapat ditetapkan sekarang

Ketiganya tidak menunggu Connect-SQL, karena buktinya sudah lengkap.

**1. Klausa `WHERE` menjadi parameter terikat, bukan teks yang dirangkai.**

Sistem lama merangkainya sebagai teks di dalam activity lalu menyerahkannya ke Connect-SQL
lewat `TempQuery.AlasanKlaim` (`Activity/GetListPasalAI_act.xml:546-548`) — pola `{ASIS:…}`
yang `08-TECHNICAL-STRATEGY.md` §4.3 larang mutlak, dan yang `R-29`/utang teknis 4.5 sebut
sebagai celah SQL injection.

Di modul baru: **satu parameter terikat**, dengan kata kunci diloloskan dari `%` dan `_`,
persis seperti `likePattern` pada Master Login. Ini **selisih terencana** — perilakunya sama
bagi pengguna, jalurnya tidak.

**2. Ukuran halaman 25, dan paginasinya di server.**

Bukan 30. Grid-nya ber-`pyPageMode = None`, sehingga paginasinya memang sudah dikerjakan
server sejak di Pega — lihat `catatan-pengembangan.md` §38.13.

Ini kebetulan yang menguntungkan: seluruh layar master lain di aplikasi ini memaginasi di
peramban, dan `TKT-U2-001` menyebut perpindahan ke paginasi sisi server sebagai **perubahan
perilaku**. Di layar ini ia justru **pemeliharaan** — sistem lamanya sudah begitu.

**3. Modul ini BACA-SAJA, mengikuti bentuk Master Reas.**

Satu method `List(ctx, Filter)` pada seam Repo, satu rute `GET`, tanpa `Get`/`Insert`/
`Update`/`Delete` — karena layar lamanya memang tidak punya jalurnya.

### 40.12 Satu dugaan yang TIDAK saya jadikan dasar

Awalan `WP` pada ketiga kolom hampir pasti **Wording Polis**, sejalan dengan
`PENGGUNAAN_WORDING_POLIS` dan `WORDING_POLIS` pada `POOLDATA.T_CLAIM_DATA_RESULTS_AI`.

Dugaan itu **tidak dipakai untuk menebak nama tabelnya.** Tidak ada satu pun rule di export
yang menyatakan hubungan itu, dan menurunkan nama tabel dari kemiripan awalan adalah persis
jenis tebakan yang §40.2 tolak. Ia dicatat sebagai petunjuk bagi pembaca berikutnya, bukan
sebagai dasar desain.

---

## 41. Modul Master Pasal AI — keputusan implementasi (2026-09-23)

Melanjutkan §40. Modul dibangun **kecuali adapter SQL**; bab ini mencatat keputusan yang
diambil saat membangunnya.

### 41.1 Keputusan 1 — dibangun sekarang, tanpa adapter SQL

Yang belum diketahui tinggal **nama tabel**. Seluruh sisanya — kolom, penyaring, paginasi,
cacah — terbaca dari `Activity/GetListPasalAI_act.xml`.

Yang menentukan: **nama tabel hanya menyentuh satu adapter.** Domain, lapisan aplikasi,
transport, dan seluruh antarmuka digerakkan oleh LAYAR, bukan oleh tabel — dan layarnya sudah
terbaca utuh. Risiko bahwa kueri yang menyusul mengubah bentuk domain karena itu kecil:
gridnya tiga kolom baca-saja, tanpa aksi klik baris dan tanpa layar detail, sehingga kolom
tambahan pada tabel pun tidak akan dibutuhkan siapa pun.

Yang **tidak** dilakukan: menulis `.sql` dengan nama tabel karangan atau penampung. Berkas
`.sql` di aplikasi ini nyata dan diuji; satu yang tidak dapat dijalankan adalah ranjau.

### 41.2 Keputusan 2 — `ErrPortalNotReady` dijawab 503, bukan 500

Pembedaannya bukan kerapian:

| | Artinya |
|---|---|
| `500` | ada yang rusak, dan kami belum tahu apa |
| `503` | belum dilayani, dan sebabnya diketahui persis |

Di sini sebabnya diketahui sampai ke nama rule-nya. Menjawab 500 akan menyembunyikan hal yang
justru sudah jelas, dan membuat petugas melaporkan "sistem error" atas sesuatu yang sedang
ditunggu dari pihak lain.

Pesannya menyebutkan apa yang ditunggu tetapi **tidak menyebut nama rule Pega**: pembacanya
petugas klaim, bukan tim pengembang.

### 41.3 Keputusan 3 — `Paginator` diekspor dari `DataTable`

Modul ini yang pertama memaginasi di sisi server, dan prop `pageSize` tidak dapat dipakai —
prop itu memaginasi baris yang sudah di tangan, sedangkan di sini yang di tangan hanya satu
halaman.

Yang dilakukan: menambahkan kata `export` pada fungsi yang sudah ada. **Nol perubahan
perilaku**, dibuktikan dengan menjalankan seluruh suite frontend.

Alternatif yang ditolak: menyalin markup paginator ke dalam modul. Dua paginator yang terlihat
sama tetapi hidup di dua berkas akan berbeda begitu salah satunya disunting — dan prompt
proyek ini menuntut keterpakaian kembali komponen.

### 41.4 Keputusan 4 — pengurutan kolom DIMATIKAN

Ketiga kolom ber-`noSort`. Dua alasan yang menguatkan satu sama lain:

1. Grid Pega-nya ber-`pySortType = NONE` pada ketiganya, tanpa `pyInitialSortColumn`.
2. Barisnya hanya satu halaman dari server. Mengurutkannya di peramban akan mengurutkan
   **halaman**, bukan daftar — dan pengguna tidak punya cara mengetahui bedanya.

### 41.5 Keputusan 5 — mencari dengan menekan tombol, bukan sambil mengetik

Kotak "Cari" di Pega tidak mencari saat diketik: action set pada even `change`-nya **kosong**.
Yang menjalankan pencarian adalah tombol **Cari**.

Ditiru apa adanya (`D-13`), dan di sini kebetulan juga pilihan yang benar: setiap ketukan
berarti satu permintaan ke basis data, dan mencari sambil mengetik menembaknya sekali per
huruf. Dijaga uji `TIDAK mencari sambil diketik`.

Tombol **Refresh** mengosongkan kotak cari **lebih dulu**, lalu memuat ulang — urutan yang
sama dengan rangkaian aksi Pega (`setValue` → `postValue` → `refresh`).

### 41.6 Tiga selisih terencana

| # | Selisih | Alasan |
|---|---|---|
| 1 | Klausa `WHERE` menjadi parameter terikat | pola `{ASIS:…}` dilarang §4.3; celah SQL injection |
| 2 | Kata kunci diloloskan dari `%` dan `_` | tanpa itu, `%` mencocokkan seluruh baris diam-diam |
| 3 | Kata kunci dibatasi 200 karakter, **ditolak** bila lebih | memotongnya mengembalikan hasil yang tidak diminta siapa pun |

Ketiganya menyentuh jalur, bukan hasil.

### 41.7 Ukuran halaman milik DOMAIN, bukan layar

Pada seluruh modul master lain, ukuran halaman adalah **prop layar** — angkanya berbeda-beda
per section di sistem lama, dan ia hanya menentukan berapa baris digambar.

Di sini ia menentukan **jendela yang dibaca dari basis data**, sehingga ia milik domain
(`masterpasalai.PageSize`). Ia tetap ikut dikirim ke klien di dalam `paginasi.ukuran_halaman`
supaya layar tidak menanamkan angkanya sendiri — dua tempat yang memuat angka yang sama akan
berpisah begitu salah satunya disunting.

Dijaga uji `TestPageSizeFollowsActivityNotSection`, yang ada khusus supaya angka **30** dari
`pyGridProps` tidak pernah masuk kembali dari membaca section saja.

### 41.8 Yang masih tertahan

| Hal | Milik siapa | Kenapa tertahan |
|---|---|---|
| Kueri `GetListDataPasalAI` dan `CountDataPasalAI` | Tim Pega | belum ada di export (`R-16`); tanpa keduanya nama tabelnya tidak diketahui |
| Urutan baris (`ORDER BY`) | Tim Pega | tidak diketahui; layarnya tidak memberi petunjuk, dan adapter memori sengaja **tidak mengarang** pengurutan |
| Arti kolom `WP_KEJADIAN` | Work Owner | tidak ada master maupun rule yang menjelaskannya; layar menampilkannya apa adanya |
| Pemasangan ke menu dan rute | — | menunggu adapter SQL; lihat `catatan-pengembangan.md` §39.6 |
| Kewenangan menu — MENU_ID 36 hanya untuk grup `IT` | `TKT-F3-005` | tabel peran dapat dibangun tetapi belum dapat diisi |

## 42. Detail Penyebab Kerugian (2026-09-23)

Modul atas `POOLDATA.D_CAUSE_OF_LOSS`, MENU_ID 38. Yang dicatat di sini hanyalah keputusan
yang **menyimpang** dari layar lama atau yang **tidak dapat dibaca** dari export; sisanya
mengikuti Pega apa adanya dan dijelaskan di tempatnya pada kode.

### 42.1 Keputusan 1 — nilai `STS_AKTIF` diambil dari pemakaiannya, bukan dari daftar pilihannya

Dropdown Status Aktif ber-`pyListSource = associated`
(`Section/BrowseDetailCauseOfLoss-Section.xml:2123`), artinya pilihannya datang dari
Rule-Obj-Property `STS_AKTIF` — dan **tidak ada satu pun direktori Properties di export**
(`R-16`).

**Yang dipakai:** `"1"` aktif, `"0"` tidak aktif.

**Dasarnya:** seluruh perbandingan `STS_AKTIF` di export menyaring pada angka satu, dan
tidak satu pun membandingkannya dengan nilai lain. Modul Master Supplier di aplikasi ini
sudah memodelkannya demikian (`mastersupplier.ActiveYes`/`ActiveNo`), dan menyimpang
darinya akan membuat dua layar menuliskan arti yang berbeda ke dalam kolom yang sama.

**Risiko yang diterima:** bila daftar pilihan aslinya ternyata memuat nilai ketiga, baris
lama bernilai itu akan terbaca "Tidak Aktif". Ia TIDAK gagal dibaca — `ActiveLabel`
memaafkan nilai yang tidak dikenal, persis seperti penyaring lama yang hanya melewatkan
yang bernilai satu.

### 42.2 Keputusan 2 — daftar TIDAK menyaring baris tidak aktif

Report Definition pengisi grid, `BrowseVDCauseOfLoss_RD`, **hilang dari export** (`R-16`),
sehingga tidak ada yang dapat memastikan apakah grid lamanya menyaring sesuatu.

**Yang dipilih:** tidak menyaring, mengikuti
`RDB List/QueryGetAllDataCauseOfLoss-SQL.xml` yang membaca view yang sama tanpa satu pun
penyaring.

**Alasannya:** baris tidak aktif yang disembunyikan akan tampak hilang bagi petugas, dan
tidak ada tombol mana pun di layar ini untuk memunculkannya kembali. Salah ke arah
menampilkan dapat dikoreksi dengan penyaring; salah ke arah menyembunyikan tidak dapat
dikoreksi dari layar sama sekali.

### 42.3 Keputusan 3 — dokumen JSON dibaca dulu, lalu ditimpa SEBAGIAN

Menyimpan berarti menulis ulang seluruh `JSONDATA`. Dokumen itu dapat memuat kunci yang
**tidak dibentangkan view mana pun** — `TempDcol` terbukti juga menampung `pyNote` dan
`pyLabel` (`Activity/CNMInsertDetailCauseOfLoss_act-Act.xml:741,790`), dan keduanya ikut
terserialkan oleh `GetPageJSONString()`.

Menyusun dokumen baru hanya dari keenam kolom yang dikenal akan **membuang kunci itu
diam-diam pada setiap penyimpanan**. Karena itu `Repo.Update` membaca dokumen aslinya lebih
dulu (`detail_document`), menimpa kunci yang memang disunting, dan membiarkan sisanya.

Dokumen yang **rusak** diperlakukan sebagai dokumen baru, bukan galat: baris yang JSONDATA
-nya tidak dapat diurai tetap harus dapat diperbaiki lewat layar, dan menolaknya justru
mengunci satu-satunya jalan memperbaikinya.

### 42.4 Keputusan 4 — `replace(...,'UnknownID',...)` TIDAK dibawa

`Database/PEGA_D_CAUSE_OF_LOSS.prc:22` menyisipkan dengan
`replace(DataPega,'UnknownID',id_dcol_ins)` — karena di sistem lama ID belum terbit ketika
dokumen disusun.

Di sini ID diterbitkan **sebelum** dokumen disusun, sehingga penggantian itu tidak
diperlukan. Dan ia lebih dari sekadar tidak berguna: `replace` bekerja atas **seluruh
dokumen**, bukan hanya kunci ID — sebuah deskripsi kerugian yang kebetulan memuat kata
`UnknownID` akan ikut tertimpa nomor baris.

**Ini selisih terencana yang memperbaiki cacat**, dan ia tidak terlihat di layar.

### 42.5 Keputusan 5 — baris baru diawali Status Aktif

Layar lama tidak menentukan nilai awal apa pun; isiannya lahir kosong.

**Yang dipilih:** baris baru diawali `"1"`.

**Alasannya:** baris yang baru dibuat memang dimaksudkan berlaku, dan membiarkannya kosong
membuat setiap baris baru lahir bertanda "Belum diisi" — keadaan yang di sistem lama hanya
dimiliki baris warisan.

Baris LAMA yang kosong **tidak diubah**: dropdown-nya menyediakan pilihan kosong
(`emptyText`), sehingga baris semacam itu dapat disimpan ulang tanpa dipaksa memilih.
Memaksanya berarti mengubah data yang tidak diminta siapa pun untuk diubah.

### 42.6 Keputusan 6 — `OLD_D_COL_ID` dijaga server, bukan hanya oleh layar

`OLD_D_COL_ID` tidak digambar di form mana pun, tetapi **dimuat dan disimpan ulang** oleh
sistem lama (`CNMSetDetailCauseOfLoss_act-Act.xml:1297-1299`).

Klien yang tidak mengirimnya karena itu tidak boleh menghapusnya: `usecase.Service.Save`
mengambil nilainya dari baris tersimpan bila permintaan mengosongkannya. Klien yang
**mengirimnya** tetap dihormati, supaya koreksi atas ID warisan tetap mungkin.

Layar pun mengirimkannya kembali — penjagaan ganda, karena tautan ke sistem sebelum Pega
tidak dapat dipulihkan bila hilang.

### 42.7 Keputusan 7 — satu-satunya aturan isian yang diberlakukan

`CNMInsertDetailCauseOfLoss_act` **tidak memeriksa apa pun** — nol `Property-Set-Messages`,
nol precondition, nol isian wajib. Perilakunya ditiru (`P-5`), mengikuti preseden Master
Pasal Kerugian yang usul aturannya ditolak Work Owner pada 2026-09-19.

Satu-satunya yang diberlakukan: **Status Aktif harus salah satu pilihannya, atau kosong.**

Ia TIDAK menolak isian pengguna — dropdown-nya hanya punya dua pilihan, sehingga nilai di
luar keduanya tidak dapat dikirim dari layar ini. Yang ditolak adalah permintaan yang datang
**dari luar layar**, dan membiarkannya lewat berarti menulis nilai yang tidak dapat
ditampilkan kembali oleh dropdown yang sama.

### 42.8 Keputusan 8 — tanpa jalur hapus, dan itu BUKAN penyimpangan

Berbeda dari Master Pasal Kerugian — yang tombol hapusnya nyata dan memaksa penghapusan
fisik yang menyupersede `D-66` — layar ini **memang tidak punya tombolnya**:

| Bukti | Isi |
|---|---|
| Tombol yang ada | Simpan, Ubah, Cari Data, Clear Pencarian |
| `pyDeleteSQL` | **kosong** (`UpdateDCauseOfLoss-SQL.xml:6`) |
| Activity penghapus | **nol** di seluruh export |

Modul ini karena itu **tidak bertentangan dengan `D-66`**. Baris yang tidak lagi dipakai
dinyatakan lewat Status Aktif — itulah gunanya kolom itu.

### 42.9 Keputusan 9 — kolom "Master Kerugian" DITAMBAHKAN ke grid

Grid Pega hanya menampilkan empat kolom; sebutan induknya tidak ada di sana.

**Ditambahkan**, karena tanpa itu sebuah detail tidak dapat dibedakan dari detail lain yang
deskripsinya mirip — dan di Pega petugas harus membuka barisnya satu per satu untuk
mengetahuinya. Datanya pun sudah tersedia: pencarian lama mengambilnya lewat sub-kueri
(`BrowseCOLByBisnis_Sql-SQL.xml:64`), sehingga tidak ada pembacaan tambahan yang
diperkenalkan.

Baris **yatim** — induknya tidak ada — ditandai terang-terangan alih-alih menampilkan sel
kosong. Tidak ada foreign key yang diketahui (`R-08`), sehingga baris semacam itu mungkin
ada.

### 42.10 Yang TIDAK dapat diverifikasi, dan harus diuji terhadap Oracle

**Pemetaan kunci JSON ke kolom view adalah rekonstruksi.** Definisi
`POOLDATA.V_D_CAUSE_OF_LOSS` tidak ada di export (`R-08`).

Dasarnya kuat — nama kolom view sama persis dengan nama properti page `TempDcol` yang
diserialkan, dan pola itu sudah terbukti pada Master Pasal Kerugian — tetapi ia **tetap
dugaan**.

**Cara membuktikannya, dan ia WAJIB dilakukan sebelum modul dinyatakan lulus:** simpan satu
baris lewat layar ini terhadap basis data sungguhan, lalu pastikan keenam kolom view-nya
terisi.

Bila salah satu kosong, kuncinya berbeda dari yang diduga — dan **itu tidak akan
menghasilkan galat apa pun**, hanya kolom kosong yang tampak seperti data yang memang belum
diisi. Mode periksa (`checkCauseOfLossDetail`) menangkap gejalanya: bila barisnya ada
tetapi `DESCRIPTION` seluruhnya kosong, ia berhenti dan menyuruh meminta definisi view ke
DBA.

### 42.11 Utang teknis yang disadari

| Hal | Keadaan |
|---|---|
| Pemetaan kunci JSON | rekonstruksi; harus diuji terhadap Oracle (§42.10) |
| Cakupan daftar | rekonstruksi; `BrowseVDCauseOfLoss_RD` hilang (`R-16`) |
| Daftar pilihan Status Aktif | diturunkan dari pemakaian; Properties tidak ada di export |
| Jejak audit | **tidak ada** — tabelnya tidak punya kolom pelaku maupun waktu; menambahnya menempuh `D-63` |
| Induk belum punya modul | Master Penyebab Kerugian (MENU_ID 20) belum dibangun; induk baru tidak dapat dibuat dari layar ini |
| Pemetaan galat | modul memetakan galatnya sendiri sampai `TKT-F1-004` diputuskan |
| `CLOB` lewat 4000 karakter | belum dapat diuji tanpa Oracle; ORA-01461 mungkin |

### 41.9 Kedua Connect-SQL diterima — modul selesai, §41.1 tertutup

Tabelnya **`POOLDATA.MST_PASAL_AI`**. Keputusan §41.1 — membangun tanpa adapter SQL —
terbukti benar: **hanya satu paket yang perlu ditambahkan**, dan tidak satu pun baris di
`usecase/` maupun `http/routes.go` yang berubah karenanya.

Yang berubah di luar `repo/sqlstore` hanya dua, dan keduanya karena kuerinya mengungkap hal
baru — bukan karena rancangannya keliru:

| Perubahan | Sebab |
|---|---|
| `Clause.ID` ditambahkan | kueri memilih `WP_ID` dan mengurutkan dengannya |
| Adapter memori diurutkan `ORDER BY WP_ID` | sebelumnya ia sengaja **tidak mengarang** urutan |

### 41.10 Keputusan 6 — `ORDER BY` TIDAK dikarang saat belum diketahui

Saat adapter SQL belum dapat ditulis, adapter memori sengaja mengembalikan urutan penyisipan
dan **menyatakan di doc comment-nya** bahwa urutan sebenarnya belum diketahui.

Godaannya: mengurutkan menurut No Pasal, karena itu kolom pertama dan tampak paling wajar.
Bila diambil, ia akan **terbukti salah** — kuerinya mengurutkan menurut `WP_ID`, kolom yang
bahkan tidak digambar layar — dan selisihnya hanya muncul setelah datanya banyak: dua halaman
berturut-turut memuat baris yang sama sementara baris lain tidak pernah tampil.

Ini pasangan dari §41.1: **yang tidak diketahui dinyatakan, bukan diisi dengan yang tampak
masuk akal.**

### 41.11 Keputusan 7 — kode mati DIBUANG, bukan disimpan

Lima hal dibuang begitu penghalangnya hilang: `ErrPortalNotReady`, pemetaan 503-nya,
`mapError`, cabang pemilih Oracle yang mengembalikan galat, dan ujinya.

Menyimpannya "untuk jaga-jaga" berarti setiap orang yang membuka berkas itu harus membaca dan
memahami jalur yang tidak mungkin dijalani.

`http/errors.go` karena itu menyusut menjadi satu kode galat — dan itu **bukan kekurangan**:
modul baca-saja tidak punya isian yang dapat cacat maupun baris yang dapat bentrok.

### 41.12 Tiga selisih terencana — kini terbukti terhadap kuerinya

§41.6 menyebutkan ketiganya sebelum kuerinya terlihat. Setelah terlihat, ketiganya bertahan:

| # | Selisih | Yang terbukti dari kuerinya |
|---|---|---|
| 1 | `WHERE` menjadi parameter terikat | kueri lama benar-benar memakai `{ASIS:TempQuery.AlasanKlaim}` |
| 2 | Kata kunci diloloskan dari `%` dan `_` | kueri lama menempelkannya ke dalam `LIKE` tanpa pelolosan apa pun |
| 3 | Kata kunci dibatasi 200 karakter | kueri lama tidak membatasi sama sekali |

Ditambah satu yang **baru** setelah kuerinya terbaca:

| # | Selisih | Alasan |
|---|---|---|
| 4 | `UPPER` di kedua sisi perbandingan | kueri lama membandingkan apa adanya, dan Oracle peka huruf pada `LIKE` — mencari "Kebakaran" tidak menemukan "KEBAKARAN". Ini **memperluas** hasil, dan itu perbaikan yang disengaja |

Ditambah satu penggantian yang dituntut portabilitas, bukan perbaikan:

| # | Selisih | Alasan |
|---|---|---|
| 5 | `ROWNUM` + tiga tingkat subquery → `OFFSET … FETCH NEXT` | `ROWNUM` tidak ada di PostgreSQL (`D-20`). Hasilnya identik: `FirstRow = offset + 1` |

### 41.13 Yang masih terbuka

| Hal | Milik siapa | Kenapa penting |
|---|---|---|
| **Siapa yang MENGISI `POOLDATA.MST_PASAL_AI`** | Work Owner | Tidak ada satu pun rule di export yang menulisinya. Layar ini hanya membaca — dan bila tidak ada yang mengisinya, ia akan selalu kosong |
| Arti `WP_KEJADIAN`, dan apakah `WP` = "Wording Polis" | Work Owner | Dugaan yang masuk akal, **tidak dipakai menurunkan apa pun** |
| Tipe dan lebar keempat kolom (`R-08`) | DBA | Menentukan apakah `ORDER BY WP_ID` mengurutkan angka atau teks — pada teks, "10" mendahului "9" |
| Hak `SELECT` akun aplikasi | DBA | Tanpa itu layar menjawab galat teknis di produksi |

### 42.12 Koreksi — tombol Tambah (ditanyakan Work Owner 2026-09-23)

**Pertanyaannya:** apakah tombol Tambah memang tidak bisa dipakai.

**Jawabannya: bisa, tetapi saya menguncinya terlalu ketat.** Tombolnya bergerbang
`portal === null || editingID !== null`, sehingga pada sesi baru — sebelum portal dipilih —
ia kelabu. Layar ini **satu-satunya dari 18 modul** yang begitu; 17 tetangganya hanya
menggerbangi dengan "form sedang terbuka".

Akibatnya tidak dapat dijelaskan kepada pengguna: layar yang sama bentuknya berperilaku
berbeda tanpa alasan yang terlihat.

**Diperbaiki menjadi `disabled={editingID !== null}`.** Portal tetap ditegakkan, tetapi di
tempat yang benar — server menolak permintaan tanpa portal (`TKT-F6-002`), dan penolakannya
kini diterjemahkan `saveMessage` menjadi pesan yang menyebut apa yang harus dilakukan
**serta menegaskan isian yang sudah diketik tidak hilang**.

Tiga uji mengunci perilaku barunya: tombol tetap hidup tanpa portal, terkunci hanya selagi
form terbuka, dan pesan portal muncul saat penyimpanan ditolak.

#### Temuan yang ikut terangkat: Pega TIDAK punya tombol Tambah

Diperiksa ulang atas `Section/BrowseDetailCauseOfLoss-Section.xml`. Seluruh label tombol
kustomnya hanya **empat**: `Simpan`, `Ubah`, `Cari Data`, `Clear Pencarian`.

Sebabnya terbaca dari dua tempat:

| Bukti | Isi |
|---|---|
| `pyTitle = "Memperbaharui Data"` dengan **`pyIsBodyVisibilityOption = ALWAYS`** dan `pySelfClear = true` (`:1165-1167`) | panel formnya **selalu tampil**, tidak pernah tersembunyi |
| `@if(TempDcol.D_COL_ID!="", TempDcol.D_COL_ID, "UnknownID")` (`CNMInsertDetailCauseOfLoss_act-Act.xml:236`) | `D_COL_ID` kosong berarti **baris baru** |

Jadi di Pega, **form kosong ITULAH jalur tambah** — pengguna mengetik lalu menekan Simpan.
`Ubah` hanya mengisi form yang sudah ada di layar. Tidak ada tombol Tambah karena tidak
dibutuhkan.

**Yang dibangun di sini berbeda bentuknya, dan itu disengaja:** form disembunyikan sampai
`Tambah` atau `Ubah` ditekan. Alasannya konsistensi — ketujuh belas layar master lain sudah
memakai pola itu, dan memperkenalkan pola kedua di satu layar akan membuat pengguna belajar
dua cara untuk pekerjaan yang sama.

Ongkosnya **satu klik tambahan** dibanding Pega, dan `D-13` menetapkan tata letak ditiru —
sehingga ini **selisih yang perlu diketahui Work Owner**, bukan diputuskan sepihak.
Mengembalikannya menjadi form yang selalu tampil adalah perubahan kecil dan terbatas pada
`DetailPage.tsx`; menunggu keputusan.

---

## 43. Modul Inbox Investigator (2026-09-23)

`MENU_ID 48` → harness `InboxInvestigator_Harness`. **Modul INBOX pertama.**

### 43.1 Keputusan 1 — layar ini INBOX, dan itu menentukan seluruh bentuknya

`D-79` mendefinisikan Inbox lewat empat ciri, dan layar ini memenuhi keempatnya:

| Ciri | Terpenuhi karena |
|---|---|
| Barisnya **pekerjaan**, bukan data acuan | penyaringnya penugasan (`pxAssignedOperatorID`), bukan atribut klaim |
| Baris **hilang** setelah selesai | `pyStatusWork != "Resolved-Completed"` |
| "Hanya milik saya" adalah **kewenangan** | isinya satu workbasket, bukan pilihan pengguna |
| Barisnya punya **tenggat** | kolom "Lama Masuk Inbox" |

**Akibatnya pada rancangan:** entitas intinya dinamai `Task`, bukan `Claim`. Yang didaftar
adalah Tugas — satuan pekerjaan pada satu tahap klaim (`CONTEXT.md`, `D-26`). Klaim yang sama
dapat muncul di beberapa inbox pada waktu berbeda, dan yang membedakannya penugasannya.

Ini juga alasan rutenya `/inbox/investigator`, bukan `/inbox-investigator`: INBOX adalah
kelompok menu tersendiri di sistem lama (`MENU_ID 2`, induk 30 butir), sehingga modul inbox
berikutnya punya tempat yang sudah jelas — seperti `/master/...` yang sudah berlaku.

### 43.2 Keputusan 2 — nama workbasket KONSTANTA, bukan parameter permintaan

`Workbasket = "InvestigatorPNC"`, dibaca dari `pyReportDefParams` pada section.

Menjadikannya parameter tampak lebih luwes dan **ditolak**: ia akan menyediakan cara membaca
antrean peran lain — Compliance, RCL/PUCL, Komite — lewat endpoint Investigator. Itu bukan
kesetaraan perilaku melainkan kewenangan baru yang tidak pernah ada.

**Ia bertentangan dengan `D-15`** (tidak ada nilai bisnis di-hardcode), dan pertentangan itu
diterima sadar: ia bukan nilai yang berubah menurut kebijakan melainkan **identitas layar
ini** — mengubahnya berarti layar ini menjadi layar lain. Bila kelak antrean investigator
dipecah per lini bisnis, ia naik menjadi master data.

Meski konstanta, ia tetap **dikirim sebagai parameter SQL**. Larangan merangkai nilai ke teks
SQL tidak mengenal pengecualian "nilainya toh dari kode sendiri"; uji
`TestWorkbasketNameNeverAppearsInSQLText` yang menjaganya.

### 43.3 Keputusan 3 — titik awal "Lama Masuk Inbox" adalah Tanggal Survey

**Keputusan Work Owner**, dan ia memperbaiki pertanyaan saya sendiri.

Saya menawarkan tiga kandidat; Work Owner menunjuk yang keempat —
`.ClaimData.SurveyResults(1).SurveyDate` — yang memang ada di Report Definition gridnya
**dan** digambar sebagai kolom.

| Kandidat | Nasib |
|---|---|
| `INVESTIGATOR_TF_DATE` | ditolak — dipakai kueri **export**, tidak ada di RD grid |
| `TanggalBuatCompliance` | ditolak — ada di RD, tetapi mengukur lama di inbox **Compliance** |
| kosongkan dulu | ditolak |
| **`SurveyResults(1).SurveyDate`** | **dipakai** |

Dikunci uji `TestTitikAwalLamaMenungguAdalahTanggalSurvei`, sehingga perubahan kelak menjadi
keputusan sadar alih-alih pergeseran diam-diam.

### 43.4 Keputusan 4 — `null` lama menunggu BERBEDA dari `0`

Perbaikan `D-49` butir 13, dibawa sampai ke layar.

`Database/GETSELISIHJAM.fnc` menutup dirinya dengan `EXCEPTION WHEN OTHERS THEN RETURN 0`,
sehingga kegagalan apa pun — termasuk tanggal kosong — mengembalikan nol jam, yang tidak
dapat dibedakan dari pekerjaan yang baru saja masuk.

Di sini pembedaannya dijaga di **empat lapis**:

| Lapis | Bentuk |
|---|---|
| Domain | `Task.WaitingHours` mengembalikan `*float64`; nil bila tanggalnya kosong |
| Usecase | `Listed.WaitingHours` bertipe `map[string]*float64` |
| API | `lama_menunggu_jam: number \| null` |
| Layar | tanda hubung, dan **tidak pernah** "0 jam" |

### 43.5 Keputusan 5 — hasil perhitungan dijepit di nol

Rumus aslinya dapat negatif, dan bukan pada kasus tepi yang jauh: pekerjaan yang masuk Sabtu
pukul 10.00 dan dibaca Sabtu pukul 18.00 menghasilkan 8 − 24 = **−16 jam**.

Dijepit di nol. Ia **penyimpangan yang disengaja** terhadap rumus lama, dan boleh karena
kolom ini **tidak pernah terisi** di layar Investigator — tidak ada baseline yang dilanggar
(lihat catatan-pengembangan §41.6).

### 43.6 Keputusan 6 — pemotongan 500 baris DINYATAKAN

`pyMaxRecords = 500` dipertahankan angkanya. Yang **berubah adalah sifatnya**.

| | Sistem lama | Di sini |
|---|---|---|
| Memotong? | ya | ya |
| Memberi tahu? | **tidak** | **ya** — `terpotong: true` + spanduk di layar |

Caranya: kueri meminta `MaxRows + 1` baris, lalu baris terakhir dibuang bila jumlahnya
melebihi. Keberadaan baris ke-501 itulah buktinya. `COUNT(*)` terpisah ditolak — dua
penembakan basis data untuk satu layar, dan kedua angkanya dapat berasal dari saat berbeda
sehingga daftar dan keterangannya saling bertentangan.

### 43.7 Keputusan 7 — urutan `ObjectList(1)` dan `SurveyResults(1)` DITETAPKAN

Kedua subquery diberi `ORDER BY` — `OBJECTID` dan `INDEX_SURVEY`.

Sistem lama tidak punya padanannya: Pega membaca elemen page list, yang urutannya sudah
ditentukan saat halaman dimuat. Di SQL, subquery tanpa `ORDER BY` dapat mengembalikan objek
berbeda pada dua pemuatan daftar yang sama — kolom "Nama Peserta" berubah isinya tanpa ada
yang berubah di data.

`INDEX_SURVEY` bukan tebakan: `Database/INSERT_SURVEYORLIST.prc` menerimanya sebagai
parameter `TSRVINDEX`, yaitu **indeks page list** yang disimpan apa adanya.

### 43.8 Keputusan 8 — Clock diterima meski modul hanya membaca

Berbeda dari Master Reas, yang **menolak** Clock karena tidak ada peristiwa untuk distempel.

Di sini jamnya **bagian dari jawaban**: "Lama Masuk Inbox" dihitung terhadap sekarang. Ia
dibaca **sekali** untuk seluruh daftar — membacanya per baris membuat dua baris pada daftar
yang sama diukur terhadap saat berbeda, dan urutan yang dihasilkan tidak dapat
dipertanggungjawabkan. `ObservedAt` ikut dikirim supaya layar dapat menyebut angkanya berlaku
kapan.

### 43.9 Keputusan 9 — lama menunggu dikirim sebagai ANGKA

Bukan teks yang sudah diformat seperti `GetSelisihJam_sql` merangkainya di dalam SQL.

Dua alasan: pemformatan untuk layar bukan urusan lapisan data
(`09-DATABASE-STRATEGY.md` §3.2, dan alasan yang sama berlaku bagi API), dan angka dapat
diurutkan serta dibandingkan — `"2 days 5 hours ago"` tidak.

### 43.10 Keputusan 10 — teks lama menunggu berbahasa Indonesia

**Penyimpangan dari `D-13`, dan yang paling perlu ditinjau Work Owner.**

`RDB List/GetSelisihJam_sql-SQL.xml` merangkainya dalam bahasa Inggris —
`'... days ... hours ago'`. Tetapi teks itu dipakai inbox **Compliance**; pada layar
Investigator kolomnya tidak pernah terisi.

Jadi tidak ada teks lama pada layar INI yang harus disamai, dan kalimat Inggris di bawah judul
kolom berbahasa Indonesia akan terbaca sebagai sisa yang belum diterjemahkan. Yang dipakai
`3 hari 5 jam`. Bila Work Owner menghendaki bentuk aslinya, yang berubah hanya fungsi
`formatWaiting`.

### 43.11 Keputusan 11 — penandaan "sudah lama menunggu" DITAMBAHKAN

Baris yang menunggu lebih dari **5 hari kerja** ditebalkan dan diberi warna kuning tua.

Alasannya bukan hiasan: pada antrean bersama berpaginasi 50 baris, angka polos menuntut
pembacanya membandingkan sendiri lima puluh angka untuk menemukan mana yang tertinggal — dan
itulah justru pertanyaan yang dibawa orang saat membuka inbox.

**Ambangnya DIKARANG**; tidak ada satu pun tenggat investigasi di export. Karena itu:
- satu ambang saja, bukan beberapa tingkat yang mengaku tahu lebih banyak daripada yang
  dapat dibuktikan;
- judul kolomnya menyebutkan ambangnya di kaki halaman, sehingga penandaannya **tidak
  bergantung pada warna semata** — warna sendirian tidak terbaca pembaca layar dan tidak
  terbedakan oleh sekitar satu dari dua belas laki-laki yang mengalami buta warna
  merah-hijau;
- tercatat sebagai **perlu dikonfirmasi**.

### 43.12 Keputusan 12 — Nama Admin kosong ditulis "proses terjadwal"

`pyOrigUserID` kosong berarti kasusnya dibuat job terjadwal, bukan orang (`D-57`).

Dibiarkan sebagai sel kosong, ia terbaca seperti data yang hilang. Ditulis apa adanya, ia
menjelaskan dirinya sendiri.

### 43.13 Keputusan 13 — tanpa CallerReader, dan itu akan berubah

Handler tidak membaca identitas pemanggil. Bukan kelalaian: yang ditampilkan adalah isi
**workbasket**, antrean bersama yang belum bertuan (`D-26`), sehingga tidak ada baris yang
diturunkan dari identitas pemanggil dan tidak ada perubahan yang perlu dicatat pelakunya.

Itu **berubah begitu layar kerjanya dibangun**: mengambil pekerjaan dari antrean menuntut
identitas pengambilnya. Modul itulah yang akan membutuhkannya, bukan modul ini.

### 43.14 Tiga hal yang TIDAK dibawa, dan dasarnya

| Yang tidak dibawa | Dasar |
|---|---|
| Dropdown "Pilih Investigation" | `pyCondition = 1==2` — permanen tersembunyi di sistem lama |
| Grid kedua | kolom, RD, dan parameter **identik** dengan grid pertama — sisa Save-As |
| Tombol Export Data Investigation | menarik `NoKTP`, `AlamatRSKlinik`, `NoRekapMedis` — `FR-R2`; di luar lingkup |

Ketiganya dicatat di banner paket dan di doc comment layar, bukan dihilangkan diam-diam.

### 43.15 Yang TIDAK dapat diverifikasi, dan harus diuji terhadap Oracle

1. **Nama kolom fisik** `POLICYNO`, `QQNAME`, `BUSINESSNAME`, `BRANCHNAME`, `PYORIGUSERID`
   pada `PC_ASM_FW_GCNMFW_WORK` — dibaca dari `ReminderPUCL-SQL.xml`, bukan dari DDL
   (`R-08`).
2. **Bacaan `weekends2`** — inklusif di kedua ujung; source-nya belum ada (`R-01`).
3. **Hak baca lintas skema** DATAPEGA + POOLDATA dalam satu kueri.
4. **Kinerja dua subquery berkorelasi** terhadap antrean besar; tidak diketahui apakah
   `T_CLAIM_OBJECTLIST.CLAIMID` dan `T_SURVEYORLIST.PNCCASEID` ter-index.

Keempatnya dijawab `claimpnc -periksa`, yang menjalankan kuerinya sungguhan.

### 43.16 Utang teknis yang disadari

| Utang | Keterangan |
|---|---|
| Ambang 5 hari kerja dikarang | menunggu konfirmasi |
| Bentuk teks lama menunggu | menyimpang dari `D-13`; menunggu konfirmasi |
| Nama workbasket di kode, bukan master data | menyimpang dari `D-15`; disadari, lihat §43.2 |
| Tanpa pemeriksaan peran | `TKT-F3-005`; sistem lama membatasi lewat `When/IsInvestigator` |
| Jalur tanpa `/v1` | mengikuti seluruh modul lain; penyeragaman ditunda |
| Baris belum dapat dibuka | layar kerja Investigator belum ada |

---

## 44. Inbox Investigator — keputusan yang DICABUT dan yang menggantikannya (2026-09-23)

Pemeriksaan ulang ke XML Pega atas permintaan Work Owner membatalkan empat keputusan §43.
Rinciannya di `catatan-pengembangan.md` §42.

### 44.1 DICABUT — §43.3 "titik awal Lama Masuk Inbox adalah Tanggal Survey"

Premisnya salah. Kolom itu **bukan durasi yang dihitung**; selnya terikat langsung pada
`.ClaimData.SurveyResults(1).SurveyDate`, terbukti dari pemasangan caption-ke-sel satu lawan
satu pada section.

**Penggantinya:** kolom kesembilan **menampilkan tanggal survei apa adanya**, di bawah caption
"Lama Masuk Inbox" yang dibawa apa adanya dari Pega (`D-13`, `P-5`).

### 44.2 DICABUT — §43.4 dan §43.5 (`null` ≠ `0`, dan penjepitan di nol)

Keduanya menjawab persoalan yang tidak ada. Tidak ada durasi yang dihitung, sehingga tidak ada
nol yang perlu dibedakan dari "tidak diketahui", dan tidak ada hasil negatif yang perlu
dijepit.

Yang tersisa dan tetap berlaku: tanggal survei yang kosong tampil sebagai **tanda hubung**,
dan barisnya **tetap ada** di antrean.

### 44.3 DICABUT — §43.8 "Clock diterima meski modul hanya membaca"

Alasannya hilang bersama durasinya. Seam `Clock` dibuang dari domain, `Options`, dan
perakitan di `cmd`.

Ini justru memulihkan prinsip yang §43.8 sendiri kutip: **seam dibuat hanya bila ada yang
benar-benar bervariasi di baliknya.** Tidak ada satu pun nilai modul ini yang bergantung pada
jam dinding.

### 44.4 DICABUT — §43.9 dan §43.10 (angka vs teks, dan bahasa teksnya)

Keduanya membahas bentuk penyajian durasi yang ternyata tidak ada. Kontrak kini mengirim
`tanggal_survey` bertipe tanggal ISO, dan layar memformatnya ke WIB seperti kolom tanggal
lain.

**Utang teknis "bentuk teks lama menunggu menyimpang dari `D-13`" karena itu HAPUS** — bukan
diselesaikan, melainkan tidak pernah ada.

### 44.5 DICABUT — §43.11 "penandaan sudah lama menunggu"

Ambang 5 hari kerja yang saya karang bersandar pada durasi yang tidak ada. Penandaannya
dibuang seluruhnya.

**Ini menghapus satu-satunya nilai yang saya karang di modul ini.**

### 44.6 DIKOREKSI — §43.14 "tiga hal yang tidak dibawa"

| Klaim §43.14 | Keadaan sebenarnya |
|---|---|
| Dropdown "Pilih Investigation" tersembunyi `1==2` | **SALAH.** `1==2` melekat pada `.pyTemplateInputBox` (placeholder desain) di posisi 107291/257681. Dropdown-nya di 58121 dan ber-`pyVisible = ALWAYS` |
| Grid kedua duplikat | **BENAR** — dan yang kedua bahkan mengeja "Nama Bisinis" |
| Export di luar lingkup | **SALAH.** Lingkup "Layar inbox saja" yang dipilih Work Owner memuat tombol Export pada teks opsinya |

### 44.7 Keputusan BARU — tiga kendali kepala layar ditunda karena BUNTU BUKTI

Layar lama punya "Pilih Investigation", "Dari", "Sampai", dan "Export Data Investigation" —
seluruhnya terlihat, dan seluruhnya melayani CSV hasil investigasi.

Ketiganya **tidak dapat dibangun sekarang**, dan sebabnya bukan lingkup:

| Bahan | Keadaan |
|---|---|
| ±20 kolom keluaran (`AlamatRSKlinik`, `NoRekapMedis`, `PasienTerdaftar`, …) | **nol kemunculan** di seluruh `RDB List/` dan `Database/` — tabel penyimpannya tidak diketahui (`R-16`, `R-08`) |
| Penyaring `IsInvestigated` | **nol kemunculan** di SQL mana pun |
| Rentang tanggal `INVESTIGATOR_TF_DATE` | **diketahui** — `POOLDATA.T_CLAIM_PNC` |

Satu dari tiga bahan tidak cukup untuk menulis kuerinya. Dicatat sebagai **permintaan artefak
ke Tim Pega/DBA**, bukan sebagai keputusan lingkup.

Catatan untuk pembangunnya kelak: export memanggil Report Definition yang sama dengan
`Param.Operator = 0`, **bukan** `"InvestigatorPNC"` — populasinya berbeda dari grid.

### 44.8 Keputusan BARU — kotak cari `DataTable` dipertahankan sebagai PENAMBAHAN

Grid Pega **tidak punya penyaring maupun kotak cari sama sekali** — `pyGridFilter`,
`pyEnableFiltering`, `pyFilterCriteria` seluruhnya nol kemunculan. Premis pertanyaan ketiga
saya kepada Work Owner karena itu salah.

Kotak cari bawaan `DataTable` tetap dipakai, dengan dua alasan yang dinyatakan terbuka:

1. Ia konvensi **seluruh layar** aplikasi ini; menghapusnya di satu layar membuat layar itu
   terasa dirakit dari aplikasi berbeda.
2. Antrean dapat mencapai 500 baris terpaginasi 50 — tanpa pencarian, menemukan satu nomor
   case menuntut membuka sepuluh halaman.

**Ia PENAMBAHAN, bukan peniruan.** Bila Work Owner menghendaki kesetaraan penuh, yang dihapus
cukup satu prop `DataTable`.

### 44.9 Yang TETAP berlaku dari §43

Tujuh keputusan tidak tersentuh koreksi ini: §43.1 (layar ini Inbox, entitasnya `Task`),
§43.2 (workbasket konstanta), §43.6 (pemotongan 500 dinyatakan), §43.7 (urutan
`ObjectList(1)`/`SurveyResults(1)` ditetapkan), §43.12 (Nama Admin kosong → "proses
terjadwal"), §43.13 (tanpa CallerReader), dan §43.15 (empat hal yang harus diuji terhadap
Oracle).

### 44.10 Utang teknis — keadaan setelah koreksi

| Utang | Keadaan |
|---|---|
| ~~Ambang 5 hari kerja dikarang~~ | **HAPUS** — penandaannya dibuang |
| ~~Bentuk teks lama menunggu~~ | **HAPUS** — tidak pernah ada durasinya |
| Kotak cari tidak ada di Pega | **BARU** — penambahan sadar, §44.8 |
| Export + Pilih Investigation + rentang tanggal | **BARU** — buntu bukti, §44.7 |
| Nama workbasket di kode, bukan master data | tetap |
| Tanpa pemeriksaan peran (`TKT-F3-005`) | tetap |
| Jalur tanpa `/v1` | tetap |
| Baris belum dapat dibuka | tetap |

---

## 45. Export Data Investigation — keputusan setelah penelusuran tuntas (2026-09-24)

Bukti lengkapnya di `catatan-pengembangan.md` §43.

### 45.1 Keputusan 1 — export TIDAK dibangun, dan itu tuntutan `CLAUDE.md`

`CLAUDE.md` melarang dummy logic **bila proses bisnis aslinya dapat dipelajari**. Sesi ini
membuktikan ia **tidak dapat dipelajari**: kesepuluh properti sumbernya nol kemunculan di 652
rule SQL dan 63 berkas `Database/`.

Membangunnya berarti mengarang kolom, dan CSV yang kolomnya berbeda dari aplikasi lama tanpa
seorang pun tahu adalah persis yang `P-5` larang.

**Yang membedakan keputusan ini dari "malas":** permintaan penggantinya sudah ditulis lengkap
di `docs/permintaan-artefak-pega.md` §2, beserta kueri katalog siap jalan. Menunda dengan
permintaan yang jelas bukan menunda.

### 45.2 Keputusan 2 — kendalinya DIGAMBAR, tetapi MATI

Tiga pilihan dipertimbangkan:

| Pilihan | Akibat |
|---|---|
| Tidak digambar sama sekali | petugas melaporkan tombol "hilang"; kemajuan migrasi tidak terbaca dari layar |
| Digambar dan berfungsi sebagian | CSV berjalan dengan kolom kosong — **lebih menyesatkan** daripada tombol mati |
| **Digambar, mati, bersebab tertulis** | **dipilih** |

Yang ketiga bukan gagasan baru: ia **konvensi proyek ini** untuk butir menu yang belum jadi
(keputusan Work Owner 2026-09-18) — tetap tampil, tidak dapat ditekan, bertanda "belum
tersedia", supaya kemajuan migrasi terbaca langsung dari layar. Alasannya berlaku sama persis
untuk kendali di dalam layar.

### 45.3 Keputusan 3 — satu `fieldset disabled`, bukan `disabled` per isian

Keempat kendali dimatikan oleh **satu** `fieldset disabled` yang membungkusnya.

Alasannya bukan kerapian: isian yang ditambahkan kemudian **ikut mati dengan sendirinya**.
Mematikan satu per satu berarti setiap penambahan menuntut seseorang mengingat memasang
`disabled` — dan yang terlupa akan menjadi isian hidup di panel yang seharusnya mati.

### 45.4 Keputusan 4 — dropdown SENGAJA tanpa pilihan

`SelectField` diberi `options={[]}` dan hanya menyisakan `==Pilih==`.

Nilai sah `SurveyList(1).IsInvestigated` **tidak diketahui** — propertinya nol kemunculan di
SQL mana pun. Mengarang daftarnya berarti menjanjikan penyaring yang tidak pernah ada, dan
pembaca berikutnya akan memperlakukan karangan itu sebagai fakta.

Dijaga uji `tidak mengarang pilihan pada dropdown Pilih Investigation`.

### 45.5 Keputusan 5 — rentang tanggal ikut dimatikan meski BUKAN buntu

`INVESTIGATOR_TF_DATE` pada `POOLDATA.T_CLAIM_PNC` sudah terbaca; secara teknis isian "Dari"
dan "Sampai" dapat difungsikan hari ini.

**Tetap dimatikan.** Satu dari tiga bahan yang menyala hanya menghasilkan export yang berjalan
tetapi kolomnya kosong. Kendali yang berfungsi setengah menuntut pengguna menebak bagian mana
yang bekerja — dan itu lebih buruk daripada satu panel yang jelas mati seluruhnya.

### 45.6 Koreksi angka: 10 properti, bukan ±20

§41 dan §42 menyebut "±20 kolom". Angka itu lahir karena saya mencacah properti **tujuan**
(kolom CSV), bukan properti **sumber**. Yang benar: **11 kolom CSV dari 10 properti unik** —
`CheckBoxPasien` dipetakan ke dua kolom sekaligus.

### 45.7 Temuan yang mengubah bentuk permintaan

`SetStatusInvestigator_Act` memakai **`Obj-Save` + `Commit`**, bukan `RDB-Save`. Kesepuluh
properti karena itu **tersimpan di BLOB objek kerja Pega** (`pzPVStream` pada
`DATAPEGA.PC_ASM_FW_GCNMFW_WORK`), bukan hilang.

Permintaan ke DBA berubah dari *"tolong cari datanya"* menjadi **"tolong expose sepuluh
properti ini menjadi kolom"** — satu tindakan yang jelas, dan datanya sudah ada di sana.

### 45.8 Yang perlu dikonfirmasi pemilik bisnis sebelum export dibangun

`ExportDataInvestigator` memanggil Report Definition yang sama dengan **`Param.Operator = 0`**,
bukan `"InvestigatorPNC"` seperti gridnya. **Populasi export berbeda dari populasi layar** —
CSV-nya dapat memuat klaim yang tidak ada di antrean investigator.

Apakah itu disengaja tidak dapat dibuktikan dari export.

### 45.9 Utang teknis — keadaan setelah sesi ini

| Utang | Keadaan |
|---|---|
| Export + Pilih Investigation | **terhalang artefak** — permintaan sudah diajukan, §2 permintaan-artefak |
| Populasi export vs populasi grid berbeda | **BARU** — menunggu konfirmasi pemilik bisnis |
| Kotak cari tidak ada di Pega | tetap — penambahan sadar (§44.8) |
| Nama workbasket di kode, bukan master data | tetap |
| Tanpa pemeriksaan peran (`TKT-F3-005`) | tetap |
| Jalur tanpa `/v1` | tetap |
| Baris belum dapat dibuka | tetap |

## 46. Modul Inbox Receive TKA (2026-09-24)

`MENU_ID 49` → harness `InboxTKA_Harness`. Work Owner mendelegasikan empat keputusan dengan
arahan **"Rekomendasi dan sesuaikan ke PEGA"**; keempatnya beserta empat keputusan lain
dicatat di sini.

### 46.1 Sumber data — `POOLDATA.T_CLAIM_TKA_H`

**Keputusan Work Owner.** Penyaring utama layar lama, `.ClaimData.TKA = "1"`, **tidak dapat
ditulis sebagai SQL**: propertinya nol kolom fisik di seluruh export, dan aturan yang
menyalakannya (`isPA_PNC` DAN `IsTKI` pada `Activity/CheckViewPolis_act-Act.xml`) bertumpu
pada When rule `IsTKI` yang hilang (`R-16`).

Keanggotaan `T_CLAIM_TKA_H` menggantikannya. Tabel itu **nol kemunculan di seluruh export** —
ia pengetahuan sisi basis data, bukan sisi rule.

### 46.2 Gabungan dua lompatan, dan gabungannya LEFT

```sql
FROM POOLDATA.T_CLAIM_TKA_H t
LEFT JOIN POOLDATA.T_CLAIM_PNC c            ON c.CLAIMNO = TRIM(t.NO_KLAIM)
LEFT JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK w  ON w.PZINSKEY = c.CLAIMID
                                           AND w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
```

Satu gabungan menyelesaikan empat hal: penyaring status Pega dapat diterapkan, `pzInsKey`
kembali tersedia, sasaran tulis Submit terjangkau, dan urutan daftar memakai
`c.REGISTERDATE` — **kolom yang sama persis dengan yang Report Definition pakai**.

**LEFT, bukan INNER.** Baris yang klaimnya tidak ditemukan TETAP TAMPIL, dengan `referensi`
kosong; penolakannya terjadi saat Submit. Gabungan INNER akan membuangnya diam-diam, dan
pekerjaan yang hilang tanpa jejak lebih mahal daripada pekerjaan yang tampil berlebih.

Penyaring statusnya karena itu ditulis `(w.PYSTATUSWORK IS NULL OR w.PYSTATUSWORK <> :1)` —
NULL berarti "tidak diketahui", dan yang tidak diketahui tidak disembunyikan.

### 46.3 Fungsi hanya di sisi tabel kecil

Gabungannya `c.CLAIMNO = TRIM(t.NO_KLAIM)` — `TRIM` **tidak pernah** membungkus kolom
`T_CLAIM_PNC`.

Sebabnya kinerja, dan selisihnya besar: `T_CLAIM_PNC` memuat data historis puluhan juta baris
(`D-10`), dan membungkus kolomnya dengan fungsi membuat index apa pun atas `CLAIMNO` tidak
dapat dipakai. `T_CLAIM_TKA_H` tidak punya index sama sekali, sehingga `TRIM` di sana tidak
mengorbankan apa pun.

Perbandingannya **peka huruf besar-kecil**, dan itu aman: kedua sisi diisi sistem, bukan
diketik orang — nomor klaim diterbitkan generator (`PNC-xxxx` atau `PNCN.YY.xxxx`, `D-71`).

### 46.4 Submit menulis DUA tabel dalam SATU transaksi

**Didelegasikan; keputusan saya.**

| Tabel | Kolom | Kenapa |
|---|---|---|
| `POOLDATA.T_CLAIM_PNC` | `TGLDOKLENGKAP` | agar tanggalnya sampai ke klaim, seperti Pega |
| `POOLDATA.T_CLAIM_TKA_H` | `TGL_DOC_LENGKAP` | agar barisnya HILANG dari inbox seketika |

Sistem lama menulis case Pega, dan nilainya mengalir ke `T_CLAIM_PNC.TGLDOKLENGKAP` lewat
`Database/PEGA_CONVERT_JSONKLAIM_PNC.prc` yang memetakan kunci JSON `TanggalDokLengkap` ke
kolom itu.

Menulis salah satu saja mematahkan separuh perilakunya: hanya `T_CLAIM_PNC` berarti barisnya
tidak hilang dan pengguna menekan Submit berulang; hanya `T_CLAIM_TKA_H` berarti barisnya
hilang tetapi tanggalnya tidak pernah sampai ke klaim.

**Urutan penguncian selalu sama** — tabel inbox lebih dulu, klaim menyusul — supaya dua
permintaan atas klaim yang sama tidak saling menunggu selamanya.

### 46.5 Setiap UPDATE diperiksa jumlah baris terpengaruhnya

Tidak ada satu pun constraint keunikan pada kedua tabel (`R-08`), dan UPDATE yang menyentuh
nol atau dua baris sama-sama "berhasil" menurut basis data.

| Baris tersentuh | Perlakuan |
|---|---|
| 1 | sebagaimana mestinya |
| 0 | transaksi dibatalkan, `ErrTaskNotFound` |
| >1 | transaksi dibatalkan, `ErrClaimAmbiguous` |

Andaian keunikan diubah menjadi kegagalan yang **bersuara**. Memilih salah satu baris berarti
memperbarui klaim yang belum tentu benar pada tabel nilai klaim — kerusakan yang tidak akan
terlihat sampai ada yang merekonsiliasi angkanya.

### 46.6 Pemberitahuan DI LUAR transaksi — menyimpang dari Pega dengan sengaja

`SubmitTanggalLengkapTKA` memanggil `SendEmailNotification` pada langkah ke-8 dan baru
`Commit` pada langkah ke-9 — surel dikirim saat transaksi masih terbuka.

**Tidak dibawa.** `10-API-STRATEGY.md` §8.2 melarang pemanggilan sistem eksternal berada di
dalam transaksi basis data. Pada layar ini akibatnya nyata: server surel yang menggantung
akan menahan kunci pada `T_CLAIM_PNC`, tabel yang dibaca 116 rule Pega yang sedang melayani
produksi.

**Akibat yang harus dinyatakan:** surel yang gagal **tidak** membatalkan penyimpanan. Di
sistem lama, kegagalan sebelum `Commit` membuang seluruh pekerjaan pengguna. Pilihan ini
berpihak pada pekerjaan pengguna — kehilangan tanggal yang sudah diketik karena server surel
mati adalah kerugian yang lebih besar daripada satu surel yang harus dikirim ulang.

Layar membedakan **tiga** keadaan: tersimpan + terkirim · tersimpan + belum dipasang ·
tersimpan + gagal kirim.

### 46.7 Penerima surel dari konfigurasi — percabangan per orang DICABUT

**Keputusan Work Owner.**

`SubmitTanggalLengkapTKA` memilih penerimanya dengan bercabang pada tiga Operator ID yang
tertanam di dalam rule (`NOVERHALOMOAN`, `ANDREWHANDOKO`, `INTANHENNYSETIYAWATI`), dan salah
satu cabangnya menunjuk **akun Gmail pribadi di jalur produksi**.

Dicabut (`D-15`, `D-67`), sebagaimana pola "nama orang menjadi syarat" sudah dicabut pada
`D-52` untuk penjenjangan komite. Penerima datang dari `SMTP_PENERIMA_TKA`.

Ikut tidak dibawa: **kata sandi SMTP plaintext** dan `UseSSL=false` (`R-17`). Nilainya tidak
direproduksi di berkas mana pun yang di-commit (`D-69`). Adapter baru memakai STARTTLS bila
server menawarkannya, dan **menolak mengirim kredensial melalui sambungan terbuka**.

Yang **dibawa apa adanya**: subjek, alamat pengirim, keenam baris badan surel beserta
labelnya, dan urutannya (`D-13`).

> `SMTP_PENERIMA_TKA` dipisahkan dari `SMTP_PENERIMA_PERINGATAN` karena pembacanya berbeda.
> Yang kedua adalah Tim IT yang menerima kabar integrasi gagal; yang pertama adalah pihak
> bisnis yang menerima kabar dokumen sudah lengkap — peristiwa yang sepenuhnya normal.
> Kelak keduanya pindah ke master Penerima Notifikasi (`F-4`).

### 46.8 `AGING` tidak ditafsirkan, hanya dicoba dibaca

**Didelegasikan; keputusan saya.**

Kolomnya `VARCHAR2(4000)` sementara sel yang diisinya di Pega terikat sebuah TANGGAL. Keduanya
tidak dapat benar sekaligus, dan tabelnya tidak muncul satu kali pun di export.

Yang dilakukan: dicoba dibaca sebagai jumlah hari (`Task.AgingDays`), dan bila gagal, teksnya
ditampilkan apa adanya. Kontraknya mengirim **dua** field — `aging` mentah dan `aging_hari`
yang boleh null.

**Null berbeda dari nol.** Nol berarti "baru masuk hari ini"; null berarti "tidak diketahui".
Menampilkan "0 hari" untuk keduanya menyatakan sesuatu yang tidak diketahui sebagai fakta.

`claimpnc -periksa` mengambil 20 nilai `AGING` yang berbeda supaya bentuk isinya dapat
diketahui dari data nyata. Bila ternyata seragam, kehati-hatian ini dapat dicabut.

Angka negatif ikut ditolak sebagai "bukan jumlah hari": lama menunggu tidak dapat kurang dari
nol.

### 46.9 Tanggal dikirim sebagai TANGGAL, bukan stempel waktu

Kontraknya memakai `YYYY-MM-DD`, bukan ISO 8601 lengkap seperti modul lain.

Kedua kolomnya bertipe `DATE` dan keduanya memang tanggal kalender. Mengirimkannya sebagai
stempel waktu berarti mengarang bagian jam, dan jam karangan itu punya akibat nyata: peramban
yang menerima `2026-09-14T00:00:00Z` lalu menampilkannya dalam WIB akan menuliskan
**15 September** — persis kelas cacat yang `R-12` catat.

Frontend pun **tidak memakai `new Date()`** untuk menampilkannya; teksnya disusun ulang tanpa
menyentuh zona waktu. Ada uji khusus yang menjaganya.

Pemotongan bagian jam dilakukan di Go (`Completion.Clean`), bukan dengan `TRUNC` di SQL yang
`D-20` larang.

### 46.10 Submit PER BARIS, bukan satu tombol untuk seluruh tabel

Mengikuti sistem lama. Grid pada `InboxTKA_Section` menggambar tombol Submit yang mengirim
parameter milik BARISNYA — `Inskey`, `ClaimNo`, `NoPolis`, `QQName`, `Insured`, `DOL`,
`Tanggal` — dan activity-nya menerima ketujuhnya dalam bentuk tunggal.

Mengirim banyak baris sekaligus akan menuntut keputusan yang belum pernah diambil siapa pun:
apakah kegagalan pada baris ketiga membatalkan dua yang pertama.

### 46.11 Rute POST, dan ia tidak idempoten

`POST /api/inbox/receive-tka/kelengkapan-dokumen`, bukan `PATCH` pada barisnya. Aksi bisnis
dimodelkan sebagai PERISTIWA (`10-API-STRATEGY.md` §2): ia memindahkan pekerjaan keluar dari
inbox, menulis dua tabel, dan melepaskan pemberitahuan.

Permintaan kedua atas baris yang sama ditolak 409. `10-API-STRATEGY.md` §7 mewajibkan kunci
idempotensi pada aksi yang menimbulkan akibat di luar sistem; yang menggantikannya di sini
adalah penyaring `TGL_DOC_LENGKAP IS NULL` yang ikut dikunci di dalam transaksi — dua
permintaan bersamaan hanya membuat satu berhasil, sehingga **surel ganda tidak dapat
terjadi**.

**Tidak ada PUT maupun DELETE.** Tanggal yang sudah diisi tidak dapat diubah, dan itu bukan
kekurangan: layar lamanya pun tidak bisa. Begitu terisi, barisnya lenyap dan tidak ada
kendali di harness itu yang dapat memanggilnya kembali.

### 46.12 Lima galat domain, dan kenapa dibedakan

| Galat | HTTP | Tindakan pengguna |
|---|---|---|
| `ErrDateRequired` | 422 | isi tanggalnya — pesan **"Silahkan isi tanggal terlebih dahulu"** apa adanya dari `Local.ErrMessages`, termasuk ejaan "Silahkan" (`D-13`) |
| `ErrTaskNotFound` | 409 | segarkan daftar |
| `ErrClaimMissing` | 409 | **menyegarkan TIDAK menolong** — barisnya akan muncul lagi; laporkan ke administrator |
| `ErrClaimAmbiguous` | 409 | laporkan ke administrator; tidak ada yang diubah |
| `errMalformedDate` | 400 | cacat frontend, bukan kesalahan pengguna |

Ketiga yang tengah dibedakan karena TINDAKANNYA berbeda, bukan demi kerapian.

### 46.13 Utang yang disadari

1. **Tanpa jejak audit pelaku.** Modul ini mengubah `TGLDOKLENGKAP` pada data klaim, dan siapa
   pelakunya seharusnya tercatat. `S-5` belum ada dan daftar peristiwa wajib auditnya masih
   ditunggu Compliance (`ADR-0026`). Sampai itu tiba, pelakunya hanya tercatat di log
   aplikasi. Ini tidak memadai: `D-59` menetapkan jejak audit adalah satu-satunya kontrol
   pengimbang karena tidak ada pemisahan tugas.

2. **Kepemilikan tulis `T_CLAIM_PNC` belum diserahterimakan.** `P-1` menetapkan satu tabel
   hanya boleh ditulis satu sistem, dan tabel itu hari ini ditulis Pega. Menyalakan jalur ini
   menempuh `D-63`. Bahaya yang menyertainya: bila Pega mengonversi ulang klaim yang sama
   dari BLOB-nya — BLOB yang tidak memuat tanggal ini — nilainya tertimpa. Selama masa
   paralel, layar lama untuk klaim TKA sebaiknya tidak dipakai bersamaan.

3. **Tanpa pemeriksaan peran.** `TKT-F3-005` belum ada, dan untuk layar ini sistem lama tidak
   memberi petunjuk apa pun: tidak ada When rule yang menjaga `MENU_ID 49`.

4. **Jalur tanpa `/v1`.** Mengikuti kontrak yang ada; penyeragamannya bukan urusan satu modul.

5. **Penerima surel di variabel lingkungan, bukan master data.** `F-4` belum ada.

---

## 46. Export Data Investigation — §45 dikoreksi setelah tabelnya ketemu (2026-09-24)

Bukti lengkapnya di `catatan-pengembangan.md` §44.

### 46.1 DICABUT — §45.1 "proses bisnisnya tidak dapat dipelajari"

Premis itu bersandar pada satu kesimpulan yang **gugur**: bahwa kesepuluh properti hanya hidup
di BLOB objek kerja Pega.

Kueri katalog dari DBA menemukan **`POOLDATA.INVESTIGATIONREPORT`** — 36 kolom, berkunci
`CASEID`, berisi enam pasang penanda `IS*` beserta keterangannya. **Datanya ada di tabel biasa
yang dapat dibaca SQL.**

### 46.2 Yang TETAP berlaku dari §45, dan kenapa

Keputusan **tidak membangun export** tetap berdiri — tetapi alasannya berganti:

| | §45 | §46 |
|---|---|---|
| Alasan | datanya tidak dapat dibaca | **pemetaan isian-ke-kolom belum pasti** |
| Yang diminta | expose 10 properti | **tiga pertanyaan ke Work Owner** |
| Jarak ke selesai | jauh — menunggu perubahan basis data | **dekat — satu percakapan** |

`CLAUDE.md` melarang dummy logic bila prosesnya dapat dipelajari. Prosesnya kini **sebagian**
dapat dipelajari: lima dari sepuluh isian punya padanan kolom yang kuat. Lima sisanya tidak —
dua bahkan tanpa kolom kandidat sama sekali (`NoRekapMedis`, `SelectRS`).

Membangun dengan setengah pemetaan yang ditebak adalah dummy logic yang sama, hanya lebih
meyakinkan penampakannya.

### 46.3 Keputusan BARU — pemetaan TIDAK ditebak dari nama kolom

Godaan terbesar sesi ini: `ISPATIENTREGIST`, `STSKWITANSI`, `ADDRESS`, `NOTE`, `TELPRS`
seluruhnya "jelas" padanannya. Tinggal empat lagi yang dikira-kira, dan export selesai hari
ini.

**Ditolak.** Dua alasan, dan keduanya berbobot:

1. **Kekeliruan serupa sudah terjadi di layar yang sama.** Kolom bercaption "Lama Masuk Inbox"
   ternyata berisi tanggal survei (§42.1). Saat itu pun namanya "jelas".
2. **Taruhannya data medis.** Salah petakan berarti nomor rekam medis pasien muncul di kolom
   yang bukan tempatnya — pada berkas CSV yang dibuka orang di luar aplikasi (`FR-R2`).

### 46.4 Keputusan BARU — bertanya ke Work Owner, bukan ke Tim Pega

Jalur lama menunggu rule yang hilang (`R-16`) — waktu tunggunya di luar kendali.

Jalur yang dipilih: **tiga pertanyaan tentang formulirnya**, yang dapat dijawab siapa pun yang
memakainya tanpa membuka sistem lama:

1. Dari keenam pasang pertanyaan, mana yang berlabel "Asuransi Lain", "Pasien", dan
   "Tidak Ada Pembayaran"?
2. Nomor rekap medis disimpan di mana? Tidak ada kolomnya.
3. Apa arti pilihan "Select RS"?

Sumbernya lebih dapat dipercaya daripada rekonstruksi dari nama kolom: yang menjawab adalah
orang yang memakai formulirnya.

### 46.5 Keputusan BARU — teks di layar dibersihkan dari istilah dapur

Teks lama berbunyi *"…tersimpan di dalam data internal aplikasi lama dan belum tersedia
sebagai kolom yang dapat dibaca. Permintaannya sudah diajukan ke DBA."*

Dua cacat sekaligus: **kini salah** (datanya ada di tabel biasa), dan **terlalu teknis** —
petugas investigasi tidak perlu tahu soal kolom dan DBA.

Penggantinya: *"Rincian hasil investigasi belum dapat dibaca aplikasi baru. Sedang disiapkan
bersama tim data."*

Penjelasan teknisnya pindah ke tempat yang memang dibaca pengembang: doc comment `ExportPanel`
dan `permintaan-artefak-pega.md` §2. Dijaga uji yang menolak kata "DBA", "kolom", "BLOB", dan
"SQL" muncul di layar.

### 46.6 Utang teknis — keadaan setelah sesi ini

| Utang | Keadaan |
|---|---|
| Export + Pilih Investigation | **menyempit** — tinggal pemetaan; tiga pertanyaan §2.5 |
| ~~Expose 10 properti ke kolom~~ | **HAPUS** — tidak pernah diperlukan |
| Populasi export vs populasi grid berbeda (`Operator = 0`) | tetap — menunggu pemilik bisnis |
| Kotak cari tidak ada di Pega | tetap — penambahan sadar (§44.8) |
| Nama workbasket di kode, bukan master data | tetap |
| Tanpa pemeriksaan peran (`TKT-F3-005`) | tetap |
| Baris belum dapat dibuka | tetap |

---

## 47. Export Data Investigation — DIHAPUS (2026-09-24)

### 48.1 Keputusan Work Owner

Fitur export **dihapus dari layar**, bukan ditunda. Instruksinya tegas dan diambil setelah
tiga putaran pertukaran bukti yang tidak berujung pada fitur yang jalan.

Ini **mencabut §45.2 dan §46.2**, yang keduanya memutuskan "tidak dibangun, tetapi digambar
mati sambil menunggu jawaban". Menunggu itu yang dihentikan.

### 48.2 Keputusan 1 — panel dihapus seluruhnya, bukan disembunyikan

Alternatif yang ditolak: menyembunyikan panel di balik flag, atau membiarkannya mati tanpa
penjelasan.

Keduanya meninggalkan kode yang tidak dipakai siapa pun tetapi tetap harus dibaca, diuji, dan
dipelihara. Keputusan "dihapus" ditulis di komentar dan di dokumen; kodenya tidak perlu ikut
tinggal untuk mengingatkan.

### 48.3 Keputusan 2 — uji keberadaan diganti uji KETIADAAN

Ketiga uji `ExportPanel` diganti satu uji yang memastikan **tidak ada** kendali export di
layar.

Alasannya bukan kelengkapan: keputusan menghapus dapat batal diam-diam bila kelak seseorang
menghidupkannya kembali tanpa pemetaan kolomnya jelas. Uji ini yang gagal lebih dulu.

### 48.4 Keputusan 3 — analisisnya disimpan, bukan ikut dihapus

`permintaan-artefak-pega.md` §2 ditandai **DIBATALKAN**, isinya utuh.

Yang disimpan: DDL `POOLDATA.INVESTIGATIONREPORT` (36 kolom), pemetaan 13 kolom CSV ke 12
properti, dan tiga hal yang buntu. Menghapusnya berarti tiga putaran pertukaran bukti dengan
Work Owner harus diulang dari nol bila fitur ini dihidupkan kelak.

### 48.5 Yang TIDAK berubah

Backend tidak tersentuh sama sekali — export tidak pernah punya rute, kueri, tipe, maupun uji.
Grid sembilan kolomnya juga tidak berubah.

### 48.6 Utang teknis — keadaan setelah penghapusan

| Utang | Keadaan |
|---|---|
| ~~Export + Pilih Investigation~~ | **HAPUS** — fiturnya dihapus, bukan ditunda |
| ~~Expose 10 properti ke kolom~~ | **HAPUS** — tidak pernah diperlukan |
| ~~Populasi export vs grid (`Operator = 0`)~~ | **HAPUS** — tidak ada export untuk dibedakan |
| Kotak cari tidak ada di Pega | tetap — penambahan sadar (§44.8) |
| Nama workbasket di kode, bukan master data | tetap |
| Tanpa pemeriksaan peran (`TKT-F3-005`) | tetap |
| Baris belum dapat dibuka | tetap |

## 48. Inbox Receive TKA — keputusan yang DICABUT dan yang menggantikannya (2026-09-24)

Keputusan §46 diambil atas premis bahwa penanda TKA tidak dapat dibaca dari basis data.
Premis itu **salah**, dan seluruh keputusan yang berdiri di atasnya ikut dicabut.

### 48.1 Yang dicabut

| Keputusan §46 | Nasib |
|---|---|
| §46.1 sumber data `POOLDATA.T_CLAIM_TKA_H` | **DICABUT** — tabel itu tidak dipakai sama sekali |
| §46.2 gabungan dua lompatan dari tabel turunan | **DICABUT** — tidak ada lagi yang perlu dijembatani |
| §46.3 fungsi hanya di sisi tabel kecil | **DICABUT** — tidak ada lagi gabungan berbasis nomor klaim |
| §46.4 Submit menulis DUA tabel | **DICABUT** — hanya satu kolom; lihat 47.4 |
| §46.5 pemeriksaan jumlah baris pada dua UPDATE | **disempitkan** ke satu UPDATE |
| §46.8 `AGING` tidak ditafsirkan | **DICABUT** — kolomnya tidak dipakai; lihat 47.5 |
| §46.10 Submit per baris | **tetap berlaku** |
| §46.6 pemberitahuan di luar transaksi | **tetap berlaku** |
| §46.7 penerima dari konfigurasi | **tetap berlaku** |
| §46.9 tanggal dikirim sebagai tanggal | **tetap berlaku** |
| §46.11 POST, tidak idempoten | **tetap berlaku**, pengamannya berubah; lihat 47.6 |
| §46.12 lima galat domain | **tetap berlaku** |

### 48.2 Sumber data — tabel yang SAMA dengan Report Definition Pega

```sql
FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
WHERE w.PXOBJCLASS        = 'ASM-FW-GCNMFW-Work-PNC'
  AND w.TKA_1             = '1'                  -- penyaring A
  AND w.TANGGALDOKLENGKAP IS NULL                -- penyaring B, sisi Pega
  AND c.TGLDOKLENGKAP     IS NULL                -- penyaring B, sisi aplikasi ini
  AND w.PYSTATUSWORK     <> 'Resolved-Completed' -- penyaring C
ORDER BY w.REGISTERDATE_1, w.PYID
```

Terverifikasi ke katalog dan ke data: cacahnya **tepat 2**, sama dengan layar Pega.

**Kenapa premis lama salah** — dua hal, dan keduanya berlaku umum di luar modul ini:

1. Export memuat **rule**, bukan skema. Kolom yang hanya dipakai Report Definition tidak
   pernah muncul di sana.
2. Report Definition menjalankan SQL terhadap **kolom**. Sebuah properti hanya dapat menjadi
   penyaring bila ia di-expose — sehingga RD yang berjalan sudah membuktikan kolomnya ada.

### 48.3 Dua kolom yang tidak di-expose, dan penggantinya

Properti yang sekadar DITAMPILKAN tidak menuntut kolom; Pega membacanya dari BLOB. Kita
tidak bisa.

| Properti | Pengganti | Dasar |
|---|---|---|
| `.ClaimData.ClaimNo` | `PYID` | layar Pega menampilkan `PNC-1546`/`PNC-1729` pada kolom itu, yaitu nilai `PYID`-nya |
| `.Policy.TheInsured` | `POOLDATA.T_GENERAL.THEINSURED` lewat `NOPOLIS` + `PRODKE` | `INSUREDNAME` pada tabel kerja terbukti KOSONG; `T_GENERAL` berisi nama yang sama dengan layar Pega |

Gabungan ke `T_GENERAL` dijembatani `T_CLAIM_PNC`, karena tabel kerja Pega tidak menyimpan
`PRODKE`.

### 48.4 Submit menulis SATU kolom, dan tabel engine Pega tidak disentuh

`POOLDATA.T_CLAIM_PNC.TGLDOKLENGKAP` saja.

**Kenapa bukan juga kolom Pega**, meski itu yang membuat barisnya hilang di sana: Pega
menyimpan nilai sebenarnya di BLOB kasus dan menyalinnya ke kolom. Menulis kolomnya dari
luar berarti nilainya tertimpa **tanpa satu pun tanda** begitu Pega menyimpan kasus itu
lagi — dan pekerjaan yang tampil di layar ini seluruhnya masih berjalan.

Yang menggantikannya: penyaring daftar memeriksa **kedua** kolom tanggal, sehingga barisnya
tetap hilang seketika dari layar ini.

**Harga yang diterima, dan ia TERLIHAT:** selama masa paralel, layar TKA di Pega masih
menampilkan klaim itu sebagai belum lengkap sampai Pega menyinkronkan. Ketidakcocokan yang
terlihat jauh lebih murah daripada data yang hilang tanpa jejak.

Tiga hal dipasang supaya harga itu tidak menjadi kejutan:

| Tempat | Yang dilakukan |
|---|---|
| `claimpnc -periksa` | `CountPegaOnly` mencacah selisihnya |
| kaki layar | menyebutkannya kepada pengguna |
| `query_test.go` | `TestPegaWorkTableIsNeverWritten` menggagalkan build bila ada yang menambahkannya |

Penulisan ini tetap menuntut serah-terima kepemilikan (`P-1`, `D-63`).

### 48.5 Kolom "Aging" — tanggal dikirim, kalimatnya disusun layar

Pega menampilkan `.ClaimData.RegisterDate` sebagai waktu relatif. Terverifikasi:
`REGISTERDATE_1 = 20240319` ditampilkan sebagai **"2 years 6 months ago"**.

Server mengirim **tanggalnya** (`tanggal_registrasi`); layar yang menyusun kalimatnya.
Alasannya: "berapa lama menunggu" bergantung pada KAPAN ia dibaca, dan menghitungnya di
server berarti nilainya membeku pada saat permintaan — sekaligus memaksa modul ini memiliki
seam Clock hanya demi satu label tampilan.

Kolomnya `VARCHAR2(32)` berisi `yyyymmdd`; penguraiannya di adapter, bukan di SQL (`D-20`
melarang `TO_DATE`). Bentuk yang tidak dikenali menjadi **nil**, bukan tanggal karangan, dan
layar menuliskannya sebagai tanda hubung. `claimpnc -periksa` memperlihatkan contoh nilainya
supaya keseragaman formatnya dapat dipastikan dari data nyata.

**Kalimatnya bahasa Indonesia**, sementara Pega menuliskannya dalam bahasa Inggris. Itu
bukan penyimpangan dari `D-13`: teks itu tidak pernah ditulis siapa pun di rule mana pun —
ia bawaan kontrol Pega. Yang ditiru adalah bentuk keterangannya.

### 48.6 Pengaman pengisian ganda berpindah ke penjaga pada UPDATE

§46 memakai `SELECT ... FOR UPDATE` pada dua tabel. Itu dicabut: menguncinya berarti menahan
baris pada tabel yang sedang dilayani Pega, dan kunci yang ditahan permintaan kita dapat
menghentikan alur kerja yang berjalan di atas kasus yang sama.

Penggantinya `WHERE ... AND TGLDOKLENGKAP IS NULL` pada UPDATE, ditambah pemeriksaan jumlah
baris terpengaruh. Dua permintaan bersamaan hanya membuat satu menyentuh satu baris; yang
kedua menyentuh nol dan ditolak — sehingga **surel ganda tidak dapat terjadi** tanpa kunci
idempotensi terpisah.

`TestNoQueryLocksThePegaWorkTable` menjaganya.

### 48.7 Kontrak yang berubah

| Field | Sebelum | Sesudah |
|---|---|---|
| `aging`, `aging_hari` | teks + jumlah hari | **dihapus** |
| `tanggal_registrasi` | — | **baru** — tanggal ISO, boleh null |
| `klaim_tersedia` | — | **baru** — false berarti klaimnya tidak ada di tabel bisnis |
| `referensi` | dapat kosong | **selalu terisi**, dan menjadi kunci baris |

`klaim_tersedia` menggantikan peran `referensi` yang kosong sebagai penanda baris yatim.
Ia diperlukan karena `referensi` kini selalu ada: sumbernya tabel yang menggerakkan kueri,
bukan hasil gabungan.

### 48.8 Utang yang masih berlaku

Ketiga utang §46.13 tetap berlaku — tanpa jejak audit pelaku (`S-5`, `ADR-0026`), kepemilikan
tulis `T_CLAIM_PNC` belum diserahterimakan (`P-1`, `D-63`), dan tanpa pemeriksaan peran
(`TKT-F3-005`).

Satu utang §46 **hilang**: ketiadaan kunci pada tabel turunan tidak lagi relevan, karena
tabel itu tidak dipakai. Kunci barisnya kini `PZINSKEY`, yang dijamin unik oleh Pega sendiri.

Satu utang **baru** menggantikannya: selisih tampilan antara layar ini dan layar Pega selama
masa paralel, yang diukur `CountPegaOnly` dan hanya hilang bila kepemilikan tulis
diserahterimakan sepenuhnya.

### 30.9 Layar diselaraskan dengan grid Pega, dan unggah ditarik (2026-09-24)

Dua permintaan Work Owner, keduanya mengoreksi keputusan pada §30.1.

**1. Kolom tetap terlihat meski tidak ada baris.** Grid Pega menggambar kepala kolomnya
beserta `pyGridNoResultsMessage` di bawahnya saat hasilnya nol — bukan menggantinya dengan
gambar kotak kosong, dan bukan pula dengan kotak peringatan.

Layar ini sebelumnya mengganti seluruh tabelnya dengan `ErrorMessage` begitu pemuatan gagal,
dan `DataTable` sendiri mengganti seluruh tabelnya dengan `EmptyState` begitu barisnya nol.
Keduanya membuat kolom hilang dari layar dalam keadaan yang di Pega justru menampilkannya.

Yang dikerjakan:

- `DataTable` mendapat prop **`showHeaderWhenEmpty`**, bawaannya `false`. Bila menyala,
  tabelnya tetap digambar dan pesan kosongnya tinggal di dalam `<tbody>`.
- `SparepartPage` menggambar tabelnya di **setiap** keadaan — termuat, kosong, gagal, bahkan
  sebelum portal dipilih — dan tidak lagi memakai `ErrorMessage` untuk jalur daftar.

Prop-nya **opt-in dengan sengaja**, sama alasannya dengan `pageSize`: menjadikannya bawaan
akan mengubah tampilan sepuluh layar master yang sudah selesai sekaligus, dan tidak satu pun
memintanya. Isolasi Protektif tetap terjaga — perilaku layar lain tidak berubah satu piksel
pun, dan `DataTable.test.tsx` beserta 27 berkas uji lain tetap lulus.

**2. Kedua tombol unggah ditarik.** §30.1 memutuskan menggambarnya dalam keadaan mati atas
jawaban "seperti aplikasi PEGA". Work Owner kini memilih menariknya sama sekali, sehingga
perlakuannya kembali sama dengan Master Panel (§28). `UploadDocument` dan
`PNCUploadMasterSparepartCSV` tetap tidak ada di export (`R-16`); yang berubah hanya
keputusan menampilkannya atau tidak.

**Satu hal yang TIDAK ikut dihapus, dan alasannya.** Permintaannya berbunyi "datanya memang
kosong, hapus pesan galatnya". Pada saat ini datanya **bukan** kosong melainkan **tidak
terbaca**: `POOLDATA.SPAREPART_HE` masih menjawab `ORA-04063` (§30.8), dan pemeriksaan ulang
pada 2026-09-24 masih menunjukkan keadaan yang sama.

Karena itu yang dihapus adalah **kotaknya**, bukan keterangannya. Grid tetap membedakan dua
keadaan lewat kalimat di dalamnya:

| Keadaan | Kalimat di dalam grid |
|---|---|
| Berhasil, nol baris | **"Data tidak ada"** — kata Pega apa adanya |
| Gagal dibaca | "Daftarnya belum dapat dimuat dari basis data. Tekan Refresh untuk mencoba lagi." |
| Portal belum dipilih | "Pilih portal entitas di bagian atas halaman…" |

Menyamakan ketiganya akan membuat view yang rusak terbaca sebagai "memang belum ada
datanya" — dan menghapus satu-satunya petunjuk di layar bahwa ada yang perlu diperbaiki DBA.
Begitu view-nya sehat dan tabelnya memang kosong, kalimat pertama yang muncul, dan layarnya
tampil persis seperti yang diminta.

Uji `tetap menggambar kolomnya saat pemuatan gagal, dengan kalimat yang berbeda` yang
menjaga pembedaan itu.

**Koreksi kalimat kosong (2026-09-24, kemudian pada hari yang sama).** Kalimat pertama
semula karangan sendiri — "Belum ada sparepart pada tab Approve." Work Owner mengoreksinya:
layar Pega berbunyi **"data tidak ada"**.

Teks itu memang TIDAK dapat dibaca dari export. Grid Pega mengambilnya dari field value
`GridNoResultsOnLoad` (dan `GridNoResultsOnFiltering` untuk hasil pencarian yang kosong),
sementara export tidak memuat satu pun direktori `Field Value/` — `R-16` lagi, dan kali ini
pada teks yang dibaca pengguna setiap hari.

Karena itu sumbernya adalah Work Owner yang membaca layar sungguhan, dan itu dicatat di doc
comment `emptyMessageFor` supaya tidak terbaca sebagai karangan pada pembacaan berikutnya.
Nama tab ikut dibuang: pesan Pega sama di ketiga tab.

Pesan untuk hasil pencarian yang kosong — "Tidak ada baris yang cocok dengan …" — sengaja
TIDAK disamakan. Pega pun memakai field value yang berbeda untuk keadaan itu.

**Koreksi kedua, hari yang sama.** Pembedaan kalimat antara "kosong" dan "gagal dibaca"
**dicabut** atas keputusan Work Owner. Seluruh keadaan nol baris kini memakai satu kalimat
yang sama, **"Data tidak ada"**, persis seperti grid Pega yang memang hanya punya satu pesan.

Akibatnya disampaikan tiga kali sebelum dikerjakan, dan diterima: **layar tidak lagi dapat
dipakai membedakan tabel yang memang kosong dari tabel yang gagal dibaca.** Pada saat
keputusan ini diambil, view `POOLDATA.SPAREPART_HE` sedang rusak (`ORA-04063`) dan layar
menampilkannya sebagai data kosong.

Yang menggantikan pembedaan itu ada di dua tempat yang tidak dilihat pengguna — log backend
dan `claimpnc -periksa` — dan keduanya disebut di doc comment `emptyMessageFor` supaya
penggantinya ikut terbaca oleh siapa pun yang membaca kodenya.

Satu pengecualian dipertahankan: sebelum portal dipilih, kuerinya belum pernah dijalankan
sama sekali, sehingga yang dibutuhkan pengguna adalah petunjuk tindakan — bukan keterangan
tentang data.

Uji `memakai kalimat yang sama saat pemuatan gagal` mengunci keputusan ini, supaya
perubahannya kelak disengaja dan bukan tergelincir.
## 26. Modul Inbox XOL (2026-09-20, sesi ketujuh belas)

Menu `MENU_ID 53`, pengganti harness `Inbox_XOL_Harness`.

### 26.1 Keputusan Work Owner pada sesi ini

| # | Keputusan | Akibatnya pada kode |
|---|---|---|
| 1 | Keempat tampilan dibangun, **tanpa aksi tulis** | Enam rute baca; tiga rute tulis ada tetapi menolak |
| 2 | **Belum menulis** — kepemilikan tabel tetap di Pega | Tidak ada `INSERT`/`UPDATE`/`DELETE` di mana pun; diuji |
| 3 | Section `InboxClaimXOL` **disediakan** Work Owner | Seluruh susunan grid dibaca dari bukti, bukan direkonstruksi |
| 4 | "Print Perhitungan" **diganti unduhan CSV** lebih dulu | `GET /pla-dla/unduh`, labelnya bukan "Print Perhitungan" |

### 26.2 Keputusan desain

**Modul ini membaca saja, dan larangan menulisnya DIUJI — bukan sekadar dituliskan.**

`P-1` menetapkan satu tabel hanya boleh ditulis satu sistem. Empat tabel yang ditulis
sistem lama tetap dimiliki Pega selama masa paralel:

	POOLDATA.XOL_TABLE_ALL_KLAIM   ditulis "INSERT DOL DAN COL"
	POOLDATA.T_PLA_XOL             ditulis penerbitan dan persetujuan PLA
	POOLDATA.T_DLA_XOL             ditulis penerbitan dan persetujuan DLA
	POOLDATA.MST_XOL_PNC           ditulis pengajuan master ke komite

Aturan yang hanya ditulis di komentar akan dilanggar oleh kode berikutnya.
`TestTidakAdaPernyataanYangMenulis` karena itu memindai seluruh kueri terhadap `INSERT`,
`UPDATE`, `DELETE`, `MERGE`, `TRUNCATE`, dan `COMMIT`, lalu menuntut setiap kueri dimulai
dengan `SELECT`.

**Tombol yang belum tersedia DIGAMBAR, dan menjawab dengan alasan.**

Ketiga aksi tulis tetap punya rutenya, dan rutenya menjawab `409` beserta penjelasan
bahwa kewenangan menulis masih ada di Pega. Bukan `404`, dan bukan pula tombol yang
disembunyikan:

- Tombol yang hilang dilaporkan pengguna sebagai fitur yang rusak.
- `404` terbaca seperti salah alamat, sehingga penyebab sesungguhnya tidak sampai ke
  pengguna maupun ke penelusur masalah.

**Pembagian kurs hidup di usecase, bukan di SQL.**

Grid berjudul "OS Value (USD)" sedangkan tabelnya menyimpan rupiah. Sistem lama
membaginya di lapisan aktivitas (`Activity/GetClaimXOL-Act.xml`). Tempatnya dipertahankan
di lapisan setara, dan itu bukan sekadar kesetiaan: pembagian di SQL akan mengulang kurs
yang sama di empat kueri berbeda, dan satu yang lupa tidak menghasilkan galat apa pun —
hanya angka yang salah.

**Treaty inward TIDAK ikut dibagi kurs, dan pembedaannya diuji khusus.**

Setiap baris treaty inward punya mata uangnya sendiri yang tidak terbawa ke hasil,
sehingga konversinya wajib terjadi sebelum penjumlahan. Membaginya lagi dengan kurs
perjanjian akan mengecilkan nilainya sebesar kurs untuk kedua kalinya — kekeliruan yang
tidak menghasilkan galat apa pun.

**Nama tabel dipilih dengan MEMILIH KUERI, bukan dirangkai.**

Sistem lama memilih `T_PLA_XOL` atau `T_DLA_XOL` dengan menyusun teks SQL di properti
klipboard lalu menyisipkannya mentah. Di sini keduanya menjadi dua kueri terpisah yang
dipilih lewat map bertipe `AdviceType`. Harga dari pilihan itu adalah kembaran yang dapat
berpisah diam-diam, dan `TestKeduaKueriPemberitahuanKembarPersis` membayarnya: yang boleh
berbeda HANYA nama tabelnya.

**Daftar group business menempuh placeholder, bukan nilai.**

Penanda di berkas `.sql` digantikan deretan `:3, :4, :5` sebelum kueri dikirim. Yang
dirangkai adalah tanda tanya, bukan isi; seluruh kode tetap dikirim sebagai argumen
terpisah. Senarai kosong DITOLAK — klausa `IN ()` tidak sah di Oracle, dan di sebagian
dialek lain ia mengembalikan seluruh baris, yang berarti menampilkan klaim group business
yang tidak ditanggung perjanjian itu.

**Pemilih perjanjian menjadi dropdown, bukan grid di dalam modal.**

Di sistem lama ia grid: pengguna menekan "INSERT DOL DAN COL", memilih satu baris, lalu
menekan "Pilih". Bentuk itu ada karena modal tersebut sekaligus tempat MENAMBAH data DOL
dan COL — dan penambahan itulah yang belum dipindahkan. Yang tersisa hanyalah pemilihan
satu nilai dari daftar pendek. Ketiga kolomnya tetap terbaca di dalam labelnya.

**Dua tombol lama menjadi SATU panel PLA/DLA.**

"Generated DLA PLA XOL" dan "Cari Data DLA PLA XOL" sama-sama memanggil
`BrowseDataXOLPLADLAGenerated` dengan tiga parameter yang sama. Yang membedakannya hanya
dari mana nilainya diambil. Menggambar dua tombol yang membuka dua panel berisi tabel yang
sama akan mengulang kekeliruan asalnya, bukan meniru perilaku yang berbeda.

### 26.3 Penyimpangan dari sistem lama, dan alasannya

| # | Perilaku lama | Di sini | Dasar |
|---|---|---|---|
| 1 | Kurs memakai `TRUNC(sysdate)`, parameter tanggalnya diabaikan | Memakai **tanggal kejadian** | `D-49` butir 4, `D-48` |
| 2 | Kurs tidak ditemukan → `RETURN 1`, valuta asing jadi 1:1 terhadap rupiah | Baris **ditandai**, nilainya tidak dikarang | `D-49` butir 5 — diterapkan lebih sempit, lihat §19.4 |
| 3 | Alamat surel reasuradur ditimpa alamat tetap bila operatornya bernama tertentu | **Tidak dibawa** | `D-15`, `D-67` |
| 4 | `PERCENT \|\| ' %'` dirangkai di SQL | Angka; tanda persen ditambahkan saat ditampilkan | `08-TECHNICAL-STRATEGY.md` §4.3 |
| 5 | Nomor dan revisi dirakit `CASE` di SQL | Dibawa terpisah, dirakit di Go | idem |
| 6 | Nilai disisipkan mentah ke teks SQL | Parameter binding tanpa perkecualian | idem |
| 7 | Dua function basis data dipanggil dari kueri | Ditulis ulang di tempatnya | `D-02` |
| 8 | Kueri akumulasi tanpa `ORDER BY` | Urutan ditetapkan | grid yang barisnya berpindah tanpa sebab terbaca sebagai kerusakan |

**Penyimpangan 2 diterapkan LEBIH SEMPIT daripada bunyi `D-48`.** `D-48` menolak
TRANSAKSI yang kursnya tidak ada; layar ini tidak bertransaksi, ia meringkas. Menolak
seluruh layar karena satu baris historis kehilangan kurs akan menutup data lain tanpa
sebab. Yang dikerjakan: barisnya ditandai, nilainya tidak dikarang, dan layar menyatakan
kursnya tidak tersedia. **Penyempitan ini keputusan saya, bukan keputusan Work Owner**, dan
dicatat di sini supaya dapat dikoreksi.

### 26.4 Cacat yang DIREPLIKASI, bukan diperbaiki

`P-5` menetapkan perilaku dipertahankan lebih dulu, dan hanya 13 butir `D-49` yang boleh
diperbaiki. Ketiga cacat berikut **tidak termasuk** di dalamnya:

| Cacat | Di mana | Akibatnya |
|---|---|---|
| `MAX(TO_CHAR(TGLINSERT,'dd/mm/yyyy'))` mengambil teks terbesar, bukan tanggal terbaru | antrean persetujuan | `31/01/2024` dianggap lebih besar daripada `01/12/2024` |
| Kolom berjudul "Date Of Loss" berisi TAHUN perjanjian | grid Approval XOL | Judulnya dipertahankan (`D-13`); keterangannya dinyatakan di bawah tabel |
| "Total Klaim" menghitung nomor klaim untuk bisnis sendiri, tetapi nama perusahaan untuk treaty inward | grid rincian | Dua hitungan berbeda di kolom yang sama |

Ketiganya dicatat di komentar tipe yang bersangkutan supaya tidak terbaca sebagai cacat
baru saat uji kesetaraan dijalankan.

### 26.5 Yang sengaja tidak dikerjakan

| Hal | Alasan |
|---|---|
| Aksi tulis apa pun | Keputusan Work Owner; `P-1` |
| Dokumen PLA/DLA resmi (PDF) | Diganti unduhan CSV lebih dulu. Berkas yang tampak resmi padahal bukan berakibat ke luar perusahaan — PLA dan DLA dikirim kepada reasuradur |
| Pemeriksaan peran per tab | `TKT-F3-004` belum ada; penugasan operator ke peran tidak ada di basis data maupun di export |
| Modul Master XOL (`MENU_ID 19`, `DetailMasterXOL`) | Menu tersendiri, layar tersendiri |
| Kolom `min(LIMIT)` dan `min(EXCESS)` dari `MST_XOL_LAYER` | Tidak satu pun dari keenam grid menampilkannya |
| Paginasi | Jumlah baris di layar ini dibatasi jumlah tahun perjanjian dan jumlah reasuradur per layer — puluhan, bukan puluhan juta |

### 26.6 Yang belum dapat dibuktikan

1. **Tidak satu pun dari kedua belas tabel XOL pernah dilihat isinya.** DDL-nya belum ada
   (`R-08`), dan tidak ada basis data di mesin tempat berkas ini ditulis. Yang terbukti
   hanyalah bentuk kuerinya.
2. **Kolom `M_CURRENCYSTANDARD` disimpulkan dari tanda tangan function**, bukan dari DDL:
   `ID`, `CurrencyDate`, `CurrencyValue`. Bila nama sebenarnya berbeda, kueri treaty
   inward gagal — dan gagalnya baru terlihat saat layar dibuka.
3. **`GetDataMasterXOL` kelas `Data-Adjustment` direkonstruksi**, bukan disalin. Ia tidak
   ada di export (`R-16`).
4. **Mata uang dasar `10001` di-hardcode**, dengan jalan keluar lewat
   `NewRepoWithBaseCurrency`. Tempatnya adalah master `F-4` yang belum ada
   (`TKT-F4-004`).

### 26.7 Pertanyaan terbuka yang menahan tahap berikutnya

| # | Pertanyaan | Kepada |
|---|---|---|
| 1 | Kapan kewenangan menulis tabel XOL berpindah ke Go, dan apakah Pega berhenti menulis pada saat yang sama? | Work Owner + Tim Pega |
| 2 | Apakah kolom `M_CURRENCYSTANDARD` memang `ID`/`CurrencyDate`/`CurrencyValue`? | DBA |
| 3 | Apakah `GetDataMasterXOL` kelas `Data-Adjustment` dapat dikirim, untuk memeriksa rekonstruksinya? | Tim Pega |
| 4 | Sub-section `PrintPLADLA_XOL` bervisibilitas `1==2` — apakah kolom aksi itu memang sudah mati, atau dimatikan sementara? | Work Owner |
| 5 | Kode mata uang dasar perhitungan treaty inward — apakah `10001` berlaku sama di keempat entitas? | Work Owner |
| 6 | Bentuk dokumen PLA/DLA resmi, untuk menggantikan unduhan CSV | Work Owner |

---

## 27. Modul Inbox Claim Treaty Prop (2026-09-21 … 2026-09-22, sesi kedelapan belas)

Menu `MENU_ID 54`, pengganti harness `InboxClaimTreaty_Harness`. Modul `U-3`.

### 27.1 Tiga keputusan Work Owner, diambil sebelum kode ditulis

Ketiganya diajukan bersama buktinya masing-masing dan dijawab 2026-09-21.

| # | Pertanyaan | Keputusan | Akibatnya pada kode |
|---|---|---|---|
| 1 | Tab "Komite Treaty ASM" membaca properti klipboard Pega yang nol kemunculan di seluruh SQL export dan tidak punya DDL (`R-08`) | **Bangun 3 tab; Komite ditandai terhalang** | `Tab.Blocked`, `BlockedReason`, `BlockedOwner` menjadi DATA yang dikirim ke layar |
| 2 | Kolom "Date Of Loss" pada grid antrean teknik selalu kosong karena kueri mengisi `CARI13` sementara sel terikat `CARI10` | **Diperbaiki**, dicatat sebagai selisih terencana `P-5` | Ketiga kueri mengaliaskannya ke `LOSS_DATE` yang sama; butir pertama `PlannedDifferences` |
| 3 | Tombol "Create Claim Treaty Prop" memanggil activity yang MENULIS objek kerja Pega | **Digambar, menolak dengan alasan** | Rute `POST` ada dan menjawab 501 `belum_tersedia` |

### 27.2 `CARI13` diterjemahkan sebagai pemeriksaan peran, bukan sakelar layar

**Yang ditemukan.** `Section/InboxClaimTreaty_Section-Section.xml` menjaga ketiga
kontainernya dengan `InputData.CARI13`. Nilai itu tidak disetel activity pemuat grid mana
pun; ia disetel `Activity/GetDataInboxTreaty_act-Act.xml` langkah 3, sesudah langkah 2
menjalankan report atas kelas `ASM-FW-GCNMFW-Int-EMAILKOMITE`, dengan prakondisi
`@equalsIgnoreCase(.OPERATOR_ID, Inputdata.CARI10)`.

**Artinya** mode `tampil` berarti pemanggil adalah anggota komite menurut
`POOLDATA.EMAILKOMITE.OPERATOR_ID`.

**Keputusan.** Ketiga kontainer menjadi TAB yang dapat dipilih pengguna, bukan mode yang
dipilih keadaan. Alasannya: pemeriksaan peran belum ada (`TKT-F3-004`), dan tabel peran
dapat dibangun tetapi belum dapat diisi — penugasan operator ke peran tidak ada di basis
data maupun di export (`11-SECURITY.md` §3.1).

**Konsekuensi yang diterima secara sadar.** Setiap pengguna yang dapat masuk melihat
ketiga tab, termasuk antrean teknik yang di sistem lama hanya terlihat oleh pemegang akun
`TreatyinPNCTeknik`. Taruhannya terbatas selama modul ini membaca saja: antrean yang
terlihat bukan antrean yang dapat diambil. Begitu pengambilan pekerjaan dipindahkan ke
sini, pemeriksaan peran menjadi prasyarat — dicatat di kepala `http/routes.go`.

**Alternatif yang ditolak.** Menebak keanggotaan komite dengan membaca `EMAILKOMITE`
sendiri. Itu akan membuat modul ini punya model peran sendiri yang berbeda dari
`TKT-F3-004`, dan dua model peran yang hidup berdampingan lebih buruk daripada satu yang
belum ada.

### 27.3 Lima grid disatukan menjadi tiga tab

**Yang ditemukan.** Kelima grid section dipasok hanya EMPAT sumber, dan satu sumber
memasok dua grid:

| Kontainer | Kueri | Mode |
|---|---|---|
| Work List Treatyin Propotional | `GetClaimTreaty_SQL` / `…AllAdmin_SQL` | bukan komite |
| Treaty Klaim | `GetClaimTreatyTeknik_SQL` | bukan komite |
| Work Teknik Treatyin | `GetClaimTreatyTeknik_SQL` | komite |
| Komite Treaty ASM | `WorkListKomite2` + `InboxKomiteTreaty_RD` | komite |

**Keputusan.** "Treaty Klaim" dan "Work Teknik Treatyin" disatukan menjadi satu tab
berjudul "Work Teknik Treatyin".

**Alasannya.** Keduanya dipasok kueri yang sama persis dengan kolom yang sama persis.
Membawa keduanya berarti dua tab yang isinya DIJAMIN identik — dan pengguna yang melihat
keduanya akan mencari perbedaan yang tidak ada.

### 27.4 Tab terhalang digambar sebagai DATA, bukan disembunyikan

**Keputusan.** `Tab.Blocked`, `BlockedReason`, dan `BlockedOwner` dikirim server ke layar,
dan layar menggambar penjelasan alih-alih tabel.

**Kenapa data, bukan teks tetap di frontend.** Supaya penghalangnya hilang dengan
sendirinya begitu DDL yang dibutuhkan tiba — tanpa menyunting frontend.

**Kenapa `BlockedOwner` ada.** Penghalang tanpa alamat tidak pernah hilang; itu pelajaran
yang sudah tercatat di `D-36`. Isinya menyebut DBA beserta artefak yang dibutuhkan.

**Kenapa tab terhalang TIDAK menggambar tabel kosong.** Tabel kosong terbaca sebagai
"tidak ada pekerjaan", padahal yang benar adalah "belum dapat dibaca". Keduanya menuntut
tindakan yang berbeda dari pengguna.

**Penolakannya terjadi di domain, bukan di penyimpanan.** `NewQuery` menolak tab terhalang
dengan `ValidationError` berisi `BlockedReason`. Kalau dibiarkan sampai ke repo, ia akan
gagal sebagai galat internal 500 — jawaban yang tidak menyebut sebabnya kepada siapa pun.

### 27.5 Paginasi dikerjakan basis data, berbeda dari Inbox Admin

**Keputusan.** `OFFSET … FETCH NEXT … ROWS ONLY` di dalam kueri, dengan
`COUNT(*) OVER ()` membawa jumlah seluruh baris.

**Kenapa berbeda dari Inbox Admin.** Inbox Admin memotong halaman di aplikasi karena Work
Owner memutuskan paginasinya direplikasi apa adanya — sistem lama memang menarik seluruh
baris lalu menghitung totalnya dari yang sudah ditarik. Di sini tidak ada perilaku seperti
itu untuk direplikasi: ketiga kueri treaty tidak punya paginasi sama sekali, dan
`pyMaxRecords` pada rule-nya kosong.

Jadi pilihannya bukan "replikasi atau perbaiki", melainkan "tentukan". Yang ditentukan
adalah bentuk yang portabel dan tidak menarik seluruh antrean ke memori aplikasi
(`09-DATABASE-STRATEGY.md` §3.3).

**Kenapa `COUNT(*) OVER ()`, bukan kueri penghitung terpisah.** Satu perjalanan, bukan
dua. Kueri kedua akan membaca ulang seluruh gabungan ke `JSON_KLAIM`, dan gabungan itulah
bagian yang mahal.

**Utang yang disadari.** Halaman yang seluruhnya di luar rentang mengembalikan `Total`
nol, karena totalnya dibawa baris — dan tidak ada baris. Keadaan itu hanya tercapai lewat
parameter yang diketik sendiri; layar tidak pernah memintanya. Dicatat di tempatnya di
`repo/sqlstore/inboxclaimtreatyprop.go`.

### 27.6 `ORDER BY` ditambahkan pada kueri yang tidak punya

`GetClaimTreatyTeknik_SQL` dan `GetClaimTreatyAllAdmin_SQL` tidak mengurutkan hasilnya.
Itu dapat dibiarkan selama seluruh baris ditarik sekaligus; begitu halamannya dipotong,
urutan yang tidak ditetapkan membuat satu baris muncul di dua halaman sekaligus hilang
dari halaman lain.

Urutannya mengikuti kueri yang MEMANG punya — `ORDER BY A.PXCREATEDATETIME DESC` pada
`GetClaimTreaty_SQL` — ditambah `PXREFOBJECTINSNAME` sebagai pemutus seri supaya
deterministik. Dinyatakan ke pengguna sebagai butir ketiga `PlannedDifferences`.

### 27.7 Tanggal Kejadian dibawa sebagai TEKS

**Keputusan.** `WorkItem.LossDate` bertipe `string`, bukan `time.Time`, dan DTO-nya pun
teks biasa — bukan `*string` berformat `YYYY-MM-DD` seperti Inbox Admin.

**Alasannya.** Nilainya dibaca `JSON_VALUE` dari `DATA_JSONBLOB`, yang selalu mengembalikan
teks. Bentuk teks di dalam blob itu tidak dapat diperiksa: tidak ada DDL, dan isinya belum
pernah dilihat (`R-08`). Mengubahnya menjadi tanggal berarti menebak formatnya untuk
seluruh baris historis.

**Akibat di layar.** Sel diformat menjadi `14 Agustus 2026` HANYA bila bentuknya
`YYYY-MM-DD`; selain itu ditampilkan apa adanya. Menampilkan apa adanya lebih jujur
daripada mengubah nilai yang tidak dikenali menjadi teks yang salah.

### 27.8 Nama akun antrean teknik menjadi konstanta domain, dikirim sebagai bind

`TreatyinPNCTeknik` literal di dalam `GetClaimTreatyTeknik_SQL`. Di sini ia konstanta
`inboxclaimtreatyprop.TechnicalWorkbasket` dan dikirim sebagai parameter binding.

**Kenapa bukan ditulis di dalam SQL.** Supaya SQL dan penyimpanan memori membaca nilai
yang sama, sehingga keduanya tidak dapat berselisih tanpa ketahuan — dan uji yang lulus di
atas memori menyatakan sesuatu yang benar tentang yang berjalan di Oracle.

**Kenapa ia tidak melanggar `D-15`.** Yang `D-15` larang adalah nilai BISNIS yang berubah
— ambang uang, penerima notifikasi, pemetaan peran. Ini kunci antrean yang menentukan tab
mana yang dibaca, bukan nilai yang diubah pengguna bisnis. Ia tetap dikumpulkan di satu
tempat, dan `-periksa` memperingatkan bila antreannya kosong supaya perubahan namanya di
produksi terlihat sebelum pengguna melaporkannya.

### 27.9 Tidak ada kotak cari, dan itu disengaja

Tak satu pun dari ketiga kueri menyaring menurut kata kunci. Penyaringnya hanya
`PXREFOBJECTKEY LIKE '%CLMP%'` ditambah pemilik antreannya.

Menambah kotak cari berarti menambah kemampuan yang tidak pernah ada — dan pada layar yang
sedang diuji kesetaraannya, kemampuan tambahan adalah selisih yang harus
dipertanggungjawabkan (`D-54`). `Query` karena itu tidak punya isian kata kunci sama
sekali, bukan punya tetapi diabaikan.

### 27.10 Perbaikan keadaan repo — dikerjakan, dan alasannya

Lima titik di `cmd/claimpnc/` tidak dapat dikompilasi sejak commit `b764434` dan `1b66136`
(rinciannya di `catatan-pengembangan.md` §27.2).

**Kenapa dikerjakan meski di luar lingkup.** Tanpa build yang berjalan, modul ini tidak
dapat dirakit maupun diverifikasi — dan menyerahkan modul yang tidak pernah dikompilasi
bersama aplikasinya berarti menyerahkan sesuatu yang belum terbukti apa pun.

**Kenapa tidak lebih dari itu.** Keempat perbaikan seluruhnya PEMULIHAN: tiga di antaranya
mengembalikan potongan yang hilang, diambil dari riwayat git bukan dikarang, dan satu
menyelesaikan penanda konflik yang kedua sisinya aditif. Tidak satu pun mengubah perilaku.

**Yang TIDAK dikerjakan, dan itu keputusan sadar:**

| Terlihat | Kenapa tidak disentuh |
|---|---|
| 130 galat typecheck di 36 berkas frontend | Seluruhnya akibat isi `src/api/types.ts` yang hilang pada merge `b764434`. Mayoritas ada di modul Master Data, yang dilindungi Isolasi Protektif |
| 45 uji frontend gagal di 8 berkas | Sama sebabnya. Memperbaikinya adalah pekerjaan tersendiri yang menyentuh modul selesai |
| Modul `inboxadmin` ada tetapi tidak dirakit di `main.go` | Perakitannya keputusan tersendiri, bukan pemulihan. Dicatat di `catatan-pengembangan.md` §27.8 |

### 27.11 `npm install` dijalankan

`recharts` terdaftar di `package.json` tetapi tidak ada di `node_modules`, sehingga 19 dari
29 berkas uji gagal DIMUAT — dan berkas yang gagal dimuat tidak menjalankan satu pun
ujinya.

Memasangnya bukan menambah dependensi; ia memasang yang sudah diputuskan sebelumnya.
Akibatnya angka uji frontend naik dari 161 menjadi 407, dan 45 kegagalan yang selama ini
tersembunyi menjadi terlihat. Kegagalan itu **bukan kegagalan baru** — dan mengembalikan
keadaan tersembunyi hanya demi angka yang terlihat lebih baik akan menyesatkan.

## 35. Modul Inbox Progress Claim (2026-09-21, sesi kedua belas)

Menu `MENU_ID 65`, pengganti harness `ProgressClaim_Harness`.

### 35.1 Lingkup — empat dari lima region, tanpa satu pun jalur tulis

**Keputusan Work Owner 2026-09-21.** Layar lama menumpuk lima bagian; yang dibawa empat:

| Region | Dibawa | Alasan |
|---|---|---|
| Outstanding | ya | grid utama, baca-saja |
| Next Follow Up | ya | grid yang sama + saringan jatuh tempo |
| Progress Klaim per PIC | ya | rekap lima pencacah, baca-saja |
| Evaluasi Progress Klaim | ya | **kosong di Pega** — dibawa sebagai bagian berketerangan |
| Approval Progress Klaim | **tidak** | satu-satunya bagian yang MENULIS |

Tombol "Input Progress Claim" ikut tidak dibawa, dengan alasan yang sama.

**Kenapa ini bukan sekadar pemangkasan lingkup.** Kedua bagian yang ditinggalkan menulis ke
`POOLDATA.GCNM_PROGRESS_CLAIM` lewat `PROGRESS_CLAIM_PNC.prc` — procedure 16 parameter yang
melakukan `COMMIT` sendiri. Selama masa berjalan paralel, tabel itu masih ditulis Pega, dan
`P-1` menetapkan satu tabel hanya boleh ditulis satu sistem. Membawanya berarti dua sistem
saling menimpa, dan akibatnya bukan galat melainkan data yang rusak diam-diam.

Akibat yang sengaja dipertahankan: seam `Repo` modul ini **tidak punya satu pun operasi
tulis**, sehingga kode yang ditulis kemudian tidak dapat menulis tanpa keputusan sadar.

### 35.2 Judul kolom memakai alias Pega, meski aliasnya menyesatkan

**Keputusan Work Owner 2026-09-21**, dari tiga pilihan yang diajukan.

Judul kolom layar ini **tidak ada di export**: setiap sel grid ber-`pyHeaderTitle` kosong,
judulnya datang dari label properti Pega, dan folder `Property` tidak ikut dikirim.

Yang dipilih: **alias apa adanya**. Konsekuensinya diterima secara sadar — sebagian judul
menyatakan hal yang bukan isinya:

| Judul di layar | Isi sebenarnya |
|---|---|
| `CaseID` | Nomor Klaim |
| `ClaimNo` | **Nomor Polis** — tertukar dengan yang di atas |
| `District` | Nama Tertanggung |
| `Country` | Catatan LGB |
| `City` / `CityID` / `CountryID` | Posisi / Status Progres 1 / Status Progres 2 |
| `AnalystTransferDate` | Next Follow Up |
| `KomiteApproveDate` | Follow Up terawal — tidak berhubungan dengan komite |
| `NOKLAIM` / `NOAKSEP` / `REINSURER` / `STSKLAIM` / `NOPOLIS` | lima pencacah pada rekap per PIC |

**Yang ditambahkan supaya keputusan itu tidak merugikan pengguna:** setiap kolom membawa
`keterangan` yang menyatakan isinya sebenarnya, dikirim server dan digambar layar sebagai
tooltip kolom. Tanpa itu, layar baru akan sama tidak terbacanya dengan layar lama, dan
satu-satunya tempat artinya tercatat adalah kode backend.

**Nama di dalam kode TIDAK ikut memakai alias** (`D-19`, `D-80`): `ClaimNumber`,
`PolicyNumber`, `InsuredName`. Kontrak JSON juga tidak: `no_klaim`, `no_polis`,
`nama_tertanggung`. Yang memakai alias hanyalah judul yang dibaca orang.

### 35.3 Dua kontrol direplikasi sebagai kontrol mati

**Keputusan Work Owner 2026-09-21.** Kotak "Tanggal Kejadian" dan dropdown lini bisnis pada
kedua region klaim **digambar tetapi tidak menyaring apa pun**, persis seperti di Pega.

Bukti bahwa keduanya memang mati:

- `TempRefresh.DateOfLoss` **nol kemunculan** di seluruh direktori `RDB List/`. Langkah
  terakhir `GetDataProgressClaim` hanya menuliskannya kembali ke dirinya sendiri supaya
  isiannya bertahan setelah layar disegarkan.
- Dropdown lini bisnis pada grid mengisi `tempgetpic.CaseID`, dan properti itu tidak dibaca
  satu pun kueri. Yang benar-benar dibaca `DataProgressClaim` hanyalah
  `tempgetpic.MCL_NAME` dan `TempCabang.District`.

**Yang tidak dilakukan, dan alasannya.** Membuat keduanya benar-benar bekerja adalah
PERUBAHAN PERILAKU: ia memunculkan selisih pada uji kesetaraan gerbang 1 dan menuntut
persetujuan tertulis sebagai butir `P-5` baru (`D-54`). Itu keputusan tersendiri.

**Penyesuaian yang diambil di dalam batas keputusan itu:** kontrolnya tidak digambar sebagai
kontrol mati yang mengundang klik. Yang digambar adalah **alasannya**, dikirim server sebagai
`kontrol_mati`. Pengguna yang mencari penyaring Tanggal Kejadian memperoleh jawaban alih-alih
menduga modulnya belum selesai — dan tidak ada yang melaporkan penyaring yang "tidak bekerja"
berulang kali.

### 35.4 Lini bisnis mengikuti definisi Export, bukan definisi grid

**Keputusan Work Owner 2026-09-21.** Sistem lama memuat DUA definisi yang tidak sama:

| | jalur grid Outstanding | jalur Export & per PIC |
|---|---|---|
| NONMBU | `groupbisnisid NOT IN ('06','09','11','16')` | `GROUP_PANEL IN ('003','004','006') AND groupbisnisid NOT IN ('09','11','16','25')` |
| BONDING | `groupbisnisid IN ('11','16')` | `groupbisnisid IN ('09','11','16','25')` |
| PA | `groupbisnisid IN ('06')` | `GROUP_PANEL IN ('002')` |
| TRAVEL | **tidak ada** | `GROUP_PANEL IN ('005')` |

Yang dipakai versi kanan. Ia satu-satunya yang benar-benar dieksekusi — versi kiri menulis
ke properti yang tidak dibaca kueri mana pun, sehingga tidak pernah berpengaruh sejak awal.

**Tanpa pilihan "semua", dan itu bukan kelalaian.** Nilai yang sama dicocokkan ke
`MST_USER_TEKNIK.TYPE_BUSINESS`; tanpa lini bisnis, tidak ada satu pun petugas yang cocok
dan hasilnya selalu kosong. Karena itu lini bisnis **wajib** pada rekap per PIC, dan
layarnya menyatakan itu.

### 35.5 Lini bisnis dipilih pengguna, karena sumber aslinya tidak ada di sistem baru

Di sistem lama lini bisnis pada rekap per PIC **tidak dipilih pengguna**. Prakondisi keempat
cabangnya berbunyi `OperatorID.pyPosition == "NONMBU"` dan seterusnya — ia dibaca dari
jabatan pada catatan operator Pega, yang rupanya diisi nama lini bisnis.

Nilai itu tidak tersedia di sistem baru: HCC/HCQ mengembalikan jabatan sebenarnya
(`Placement.PositionName`), bukan lini bisnis.

| Pilihan | Akibat |
|---|---|
| Bandingkan `Position` apa adanya | region **selalu kosong** bagi setiap pengguna |
| **Dipilih pengguna lewat dropdown** ← diambil | region dapat dipakai; penyaringnya persis sama, yang berbeda hanya asal nilainya |
| Tunda region ini | membuang satu dari tiga region yang sudah disetujui masuk lingkup |

Keterbatasan itu **dinyatakan di layar** sebagai salah satu butir `keterbatasan`, bukan
disembunyikan. Ia hilang dengan sendirinya begitu pemetaan pengguna ke lini bisnis menjadi
master data (`F-4`).

### 35.6 `GET_POSISI_PROGRESS_PNC` ditulis ulang sebagai satu kueri per halaman

`D-02` menetapkan tidak ada pemanggilan stored function dari aplikasi. Fungsi ini dipanggil
**empat kali untuk setiap baris grid** — sekali per kategori (`POSISI`, `sts_prg1`,
`sts_prg2`, `nextfu`) — dan setiap panggilan mengulang kursor yang sama persis. Pada satu
halaman 15 baris itu 60 pemanggilan.

Penggantinya kueri `positions`: **satu kueri untuk seluruh baris satu halaman**, dan
penggabungan antarposisi terjadi di Go.

**Kenapa penggabungannya tidak di SQL.** `LISTAGG` dilarang repo (`09-DATABASE-STRATEGY.md`
§4), sementara penggantinya `STRING_AGG` **tidak ada di Oracle 19c** yang berjalan hari ini.
Memindahkannya ke Go menyelesaikan keduanya sekaligus, dan sebagai akibatnya jumlah posisi
sebuah klaim menjadi angka yang dapat dihitung — bukan koma yang harus dihitung pengguna.

**Satu cacat yang hilang karenanya.** Fungsi lama memformat tenggat dengan
`to_char(..., 'DD-MM-YYYY hh:mm:ss')`. Pada Oracle, `mm` di dalam bagian jam berarti
**bulan**, bukan menit — menitnya seharusnya `mi`. Jam yang selama ini tampil karena itu
berbunyi `jam:BULAN:detik`. Di sistem baru tenggatnya tanggal sungguhan dan jamnya tidak
dikirim sama sekali. Ini **selisih terencana** pada uji kesetaraan, bukan cacat.

### 35.7 Paginasi benar-benar di basis data — berbeda dari Inbox Admin

Inbox Admin menarik seluruh baris lalu memotongnya di aplikasi, karena begitulah Pega
melakukannya di layar itu. **Layar ini sebaliknya**: `DataProgressClaim` memotong dengan
`ROW_NUMBER` antara `FirstRow` dan `LastRow`, dan `GcnmCountProgressClaim_SQL` menghitung
totalnya terpisah.

Perbedaannya bukan pilihan gaya — keduanya mengikuti layarnya masing-masing.

| Hal | Ketetapan |
|---|---|
| Ukuran halaman | **15**, dari `.PageSize` pada activity. Bukan 25 seperti Inbox Admin |
| Sintaks | `OFFSET … FETCH NEXT`, bukan `ROW_NUMBER` (`09-DATABASE-STRATEGY.md` §3.3) |
| Pencacah | kueri terpisah, penyaringnya **wajib sama persis** — dijaga uji |
| Urutan | `TGL_PROSES ASC`, **ditambah nomor klaim sebagai pemutus seri** |

Pemutus seri itu tambahan, dan ia menutup cacat nyata: tanpanya, baris ber-`tgl_proses` sama
dapat berpindah halaman antarpermintaan sehingga satu baris muncul dua kali sementara baris
lain tidak pernah muncul.

### 35.8 Satu endpoint melayani dua bentuk baris

Region klaim dan rekap per PIC punya bentuk baris yang berbeda: yang satu klaim, yang lain
petugas beserta pencacahnya.

| Pilihan | Alasan ditolak |
|---|---|
| Satu struktur gabungan | setiap baris klaim mengirim enam isian pencacah bernilai nol |
| Dua endpoint terpisah | layar harus memilih endpoint sebelum ia tahu region mana yang diminta — padahal yang menentukannya adalah metadata dari server |
| **`ListResponse.Rows` bertipe `any`** ← diambil | bentuknya ditentukan `bagian.bentuk`, dan alasannya ditulis di tempatnya |

Standar koding melarang `any` **tanpa alasan tertulis**; alasannya ada di `dto.go`. Domain
sendiri tetap punya dua tipe terpisah — `ClaimRow` dan `PICSummary` — sehingga yang longgar
hanyalah kontrak wire, bukan modelnya.

### 35.9 Daftar bind disusun dari jumlah baris, bukan dari isinya

Kueri `positions` mencari posisi untuk seluruh nomor klaim pada satu halaman, dan panjang
daftarnya berubah-ubah.

| Pilihan | Alasan ditolak |
|---|---|
| Satu kueri per baris | kueri di dalam perulangan, dilarang `15-NFR` §3.2 |
| Bind sebanyak `MaxPageSize` lalu sisanya NULL | setiap kueri membawa 100 bind meski halamannya 15 baris |
| Mengulang kueri halaman sebagai subkueri | kueri rumit yang sama hidup di dua tempat dan dapat berselisih |
| **Menyusun penanda bind dari jumlah baris** ← diambil | yang disusun `:7, :8, :9`, bukan nilainya |

Larangan yang berlaku adalah **merangkai NILAI** ke dalam teks SQL. Yang disusun di sini
panjangnya ditentukan jumlah baris, bukan isi baris, dan seluruh nomor klaim tetap dikirim
sebagai argumen bind. `query_test.go` menguji tepat itu: keluaran `inList` hanya boleh
terdiri atas penanda bind dan pemisahnya.

### 35.10 Bagian bertumpuk, bukan bertab — dan dimuat saat dibuka

`Section/ProgressClaim_Section-Section.xml` menumpuk kelima bagiannya sebagai kontainer
berjudul, bukan sebagai bilah tab. Inbox Admin memang bertab; layar ini tidak. `D-13`
menetapkan tampilan meniru Pega, jadi keduanya dibedakan.

Tiap kontainer di Pega memasang `pyDeferLoadRetrievalActivity` sendiri, sehingga kuerinya
berjalan saat bagiannya ditampilkan. Perilaku itu dibawa: **bagian memuat datanya sendiri
saat dibuka**. Alasannya bukan kesetiaan semata — rekap per PIC menghitung lima subkueri
agregat, dan membayarnya untuk bagian yang tidak dilihat adalah pemborosan yang terasa.

Bagian Evaluasi **tidak meminta apa pun**, bahkan saat dibuka: jawabannya sudah pasti
kosong.

### 35.11 Kolom yang digambar dua kali dipertahankan

Region Outstanding mengikat `.DateForAging` di **dua sel grid terpisah**, keduanya terikat
`tglklaim`. Dua kolom bersebelahan karena itu menampilkan tanggal yang sama dengan judul
yang sama pula.

Dipertahankan atas keputusan "replikasi apa adanya": menghapus kolom kedua mengubah jumlah
kolom yang dibandingkan pada uji kesetaraan gerbang 1.

**Dugaan yang TIDAK diterapkan:** kueri mengembalikan `tgl_proses` yang tidak terikat ke sel
mana pun, dan sangat mungkin sel kedua seharusnya menggambar kolom itu. Itu dugaan, bukan
bukti — ia dicatat di `view.go` dan diajukan, bukan diam-diam diperbaiki.

Akibat teknisnya: identitas kolom (`Key`) **dipisah** dari nama isiannya (`Field`), karena
dua kolom yang menggambar isian yang sama akan bertabrakan bila identitasnya diambil dari
isiannya.

### 35.12 Penyaring cabang belum dibangun

Kueri lama menyaring cabang lewat `GetIDCabang`, yang menembus DB Link:

    from branch a, hrdasm.v_hrd_mst@asmd.sinarmas.co.id c,
         lst_user_asuransi@asmd.sinarmas.co.id b

DB Link `@ASMD` belum punya API pengganti (`R-03`, `D-25`). Penyaring cabang karena itu
**tidak dibangun**, sama seperti di Inbox Admin. Akibatnya untuk sekarang seluruh pengguna
melihat klaim seluruh cabang — persis seperti yang di sistem lama hanya berlaku bagi
pengguna kantor pusat.

Uji `TestBranchFilterIsAbsentEverywhere` menjaga agar ia tidak masuk diam-diam lewat DB
Link, yang akan menembus batas yang sengaja belum dilewati.

### 35.13 Konflik merge diselesaikan secara aditif

`cmd/claimpnc/main.go` dan `check.go` ter-commit dengan enam penanda konflik yang belum
diselesaikan, sehingga paket `cmd` tidak dapat dikompilasi sejak sebelum sesi ini.

Keenamnya **murni aditif** — satu sisi menambah modul master, sisi lain menambah
`inboxadmin`. Penyelesaiannya mempertahankan **kedua sisi**, tanpa menghapus satu baris pun
milik siapa pun. Tidak ada perilaku yang diubah; yang berubah hanya berkas yang kini dapat
dikompilasi.

---

## 36. Modul Inbox Laporan Klaim (2026-09-19, sesi kesembilan)

Menggantikan harness Pega `InboxRCVApp_Harness` — menu `MENU_ID 64`, judul layar lama
**"Inbox Reporting Claim"**.

### 36.1 Keputusan Work Owner pada sesi ini
| 1 | Lingkup modul | **Paritas penuh dengan Pega, tetapi penulisan TIDAK lagi masuk ke `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`** |
| 2 | Paginasi | **Sisi server**, seperti aplikasi Pega yang berjalan sekarang |
| 3 | Penyimpanan | **sqlstore + memory**, mengikuti seluruh modul sebelumnya |

Keputusan 1 adalah yang menentukan seluruh bentuk modul. Ia bukan sekadar preferensi: tabel itu
dibaca **116 rule Pega** yang masih melayani produksi, dan menulis ke sana selama masa paralel
melanggar penulis tunggal per tabel (`ADR-0004`, `P-1`).

### 36.2 Dua tabel, satu daftar

| Tabel | Peran | Siapa penulisnya |
|---|---|---|
| `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` | berkas laporan warisan | **Pega** — aplikasi ini hanya membaca |
| `POOLDATA.CPNC_LAPORAN_KLAIM` | berkas laporan baru | **aplikasi ini** — migrasi `0003` |

Daftar yang dilihat petugas adalah **gabungan keduanya**, dan asal setiap baris dibawa apa adanya di
kolom `asal` (`pega` / `claimpnc`). Kolom itu **tidak ada di layar lama** — di sana tabelnya memang
satu. Ia ada di sini karena pertanyaan pertama pada setiap selisih rekonsiliasi masa paralel adalah
"baris ini ditulis siapa", dan menjawabnya tidak boleh menuntut membuka basis data.

Nomor berkas baru berbentuk **`RCVN.YY.xxxx`**, meniru `PNCN.YY.xxxx` (`D-71`) beserta alasannya:
asal sebuah baris harus terbaca dari nomornya tanpa tabel pemetaan.

**Dua hal sengaja berbeda dari `D-71`**, dan keduanya perbaikan yang mungkin karena belum ada satu
pun nomor berbentuk ini yang terbit:

1. Nomor urut **dipadatkan nol** sampai empat digit. `D-71` butir 2 mencatat bahwa tanpa pemadatan,
   `".10"` mendahului `".9"` saat diurutkan sebagai teks. Cacat itu tidak dibawa.
2. Tahunnya diambil dari **seam Clock**, bukan dari `SYSDATE` basis data — menutup catatan ketiga
   `D-71` yang bertaut dengan `R-12`.

### 36.3 Sembilan tab, dan kenapa angkanya sembilan

`SetListRCV_Act` memilih kueri lewat rantai `@if` atas `param.Note`. Enam kueri, **sembilan tab**:
ketiga tab komunikasi memakai `ViewRejectKomunikasiUser` dengan penyaring percakapan yang berbeda.

| Tab (teks layar Pega) | `param.Note` | Kueri lama |
|---|---|---|
| Outstanding Data | 1 | `ViewTableBrowseClaimRegister` |
| Unregistered data | 9 | `ViewTableBrowseClaimNotRegister` |
| Data hasn't been transferred | 2 | `ViewTableBrowseRCVInProcess` |
| Data has been accepted | 3 | `ViewTableBrowseRCVAcc` |
| Data rejected | 4 | `ViewTableBrowseRCVReject` |
| Not answered communication | 5 | `ViewRejectKomunikasiUser` |
| Not replied from ASM | 7 | idem |
| Replied from ASM | 8 | idem |
| All data | lainnya | `ViewAllCase` |

**Angka warisan tetap disimpan** (`Category.LegacyCode`) meski tidak dipakai jalur normal: perkakas
uji kesetaraan `S-8` menembak kueri Pega yang sama dengan tab yang sedang dibandingkan, dan tanpa
pemetaan ini pasangannya harus ditebak.

**Kode tab yang tidak dikenal DITOLAK.** Di sistem lama, rantai `@if` menjadikan setiap nilai tak
dikenal jatuh ke `ViewAllCase` — sehingga salah ketik menghasilkan daftar yang tampak wajar tetapi
bukan yang diminta. Itu bukan aturan bisnis melainkan akibat bentuk rantai `@if`, dan tidak dibawa.

### 36.4 Tiga cacat sistem lama yang dibiarkan terlihat, bukan ditutup

| Cacat | Perlakuan |
|---|---|
| **Tab "Data rejected" tidak punya lencana** — kueri pencacah menyaring `PYSTATUSWORK NOT IN (Resolved-Completed, Resolved-Rejected)`, sehingga berkas ditolak justru yang dikecualikan | Lencananya **tidak digambar**. `Summary.CountOf` mengembalikan angka DAN apakah ia dihitung — nol berarti "tidak ada berkas", ketiadaan lencana berarti "tidak dihitung". Menghitungnya dengan cara lain akan menaruh dua angka berdampingan yang dihitung berbeda |
| **Kueri grid dan kueri pencacah tidak sepakat** untuk "Replied from ASM": grid `sender != saya`, pencacah `sender = saya` | Grid mengikuti kueri **grid** (itulah yang dilihat pengguna), lencana mengikuti **pencacah** (itulah angka yang selama ini tampil). Selisihnya dicatat di `MessageFilter` supaya tidak dikira cacat baru saat uji kesetaraan |
| **Tab akseptasi tidak menyaring `PYSTATUSWORK`** — satu-satunya tab yang tidak | Dipertahankan apa adanya, dan disebut namanya di `categoryArguments` |

### 36.5 Lima alias kolom yang tidak dibawa (`D-19`)

Kesembilan kueri lama mengaliaskan kolomnya ke nama properti Pega yang sudah ada — utang teknis §4.2
`03-CURRENT-ARCHITECTURE.md`. Lima di antaranya berbahaya karena menyebut hal yang sama sekali lain:

| Kolom | Alias Pega | Nama di sini |
|---|---|---|
| `BUSINESSNAME` | `Kurir` | `BusinessName` — nama bisnis, **bukan** kurir |
| `br.branchname` | `UserAdmin` | `BranchName` — nama cabang, **bukan** nama pengguna |
| `pxcreateoperator` | `KodeCabang` | `CreatedBy` — operator, **bukan** kode cabang |
| `kodecabang_1` | `StatusKomunikasi` | `BranchCode` — kode cabang, **bukan** status |
| `KETERANGAN_1` | `SIM` | `Reason` — keterangan, **bukan** nomor SIM |

Pemetaan baliknya ditulis di kepala berkas `.sql`; itu satu-satunya tempat ketiganya — kolom, alias
lama, dan nama domain — dapat dibandingkan berdampingan.

### 36.6 Satu aturan yang ditambahkan lalu dicabut

Versi pertama menolak pembuatan berkas ketika cabang pemanggil tidak terbaca. **Dicabut.**

`CreateNewCaseRCV` langkah 19 mengisi cabang dari hasil `GetIDCabang` dan tidak memeriksa hasilnya
sama sekali. Menolaknya adalah aturan **baru**, dan `P-5` menetapkan perilaku dipertahankan lebih
dulu kecuali untuk 13 butir yang `D-49` sebut satu per satu — penolakan ini tidak ada di antaranya.

**Akibat yang diterima secara sadar:** berkas yang lahir tanpa cabang tetap terlihat pembuatnya
(penyaring cabang tidak berlaku bagi pemanggil yang cabangnya kosong) tetapi tidak terlihat petugas
cabang mana pun. Itu perilaku sistem lama, dan memperbaikinya adalah keputusan Work Owner.

### 36.7 Penyimpangan yang disadari

| # | Penyimpangan | Alasan |
|---|---|---|
| 1 | **`ROWNUM` diganti `OFFSET ... FETCH NEXT`** | `D-20` menuntut satu set SQL yang berjalan di Oracle 19c dan PostgreSQL 17+. Pola penggantinya sudah dipakai 35 rule lain di sistem lama |
| 2 | **`FROM DUAL` dipakai satu kali** — pengambilan sequence | Pengecualian yang sama yang `D-22` dan `D-71` akui untuk nomor klaim. Namanya disebut di `query_test.go`, dan kueri LAIN yang memakainya akan gagal uji |
| 3 | **Teks SQL disambung Go** — fragmen `WITH` + badan kueri | Yang disambung dua teks dari berkas `.sql` sendiri, keduanya konstanta saat kompilasi. Yang dilarang adalah merangkai NILAI, dan seluruh nilai tetap lewat parameter binding |
| 4 | **Nomor bind bergaya Oracle `:n`** | Utang yang sudah ada sebelum modul ini dan berlaku untuk seluruh berkas `.sql` di aplikasi ini; PostgreSQL memakai `$n` |
| 5 | **Transaksi dibuka di lapisan repo** saat menyisip | Nomor yang sudah diambil dari sequence tidak dapat dikembalikan. Membungkusnya tidak menutup lubang pada deret nomor — sequence Oracle memang tidak ikut di-rollback — tetapi memastikan tidak ada berkas separuh jadi tersimpan. Pengecualian yang sama sudah diambil modul Master Status Progres |
| 6 | **Ekspor CSV ditampung Blob di peramban** | Peladen mengalirkan berkasnya potong demi potong, peramban menampungnya utuh. Manfaat pengaliran tinggal di sisi peladen. Alternatifnya menaruh token di alamat, dan itu ditolak |

### 36.8 Batas ekspor: 50.000 baris

Sistem lama **tidak punya batas** pada layar ini, dan justru itu alasannya ada. `pyMaxRecords=500`
terpasang pada 54 dari 56 laporan Pega, sehingga kebutuhan ekspor bervolume besar **belum pernah
benar-benar dilayani** — berapa baris yang wajib dilayani satu ekspor adalah pertanyaan terbuka
`ADR-0011` yang belum dijawab.

Angkanya karena itu **bukan aturan bisnis melainkan penjaga**: ia mencegah satu permintaan menarik
puluhan juta baris (`D-10`) sebelum jawabannya ada. Ia dipasang jauh di atas 500 supaya tidak
diam-diam mengulang pemotongan lama, dan berkas yang menyentuhnya **diberi tanda di baris
terakhir** — bukan dipotong tanpa satu pun pemberitahuan, yang persis cacat sistem lama.

### 36.9 `DataTable` diberi mode server — bersifat MENAMBAH

Prop `serverPaging` dan `hideSearch` keduanya opsional. Tanpa keduanya, perilaku komponen **tidak
berubah sama sekali**, sehingga tiga layar master yang sudah selesai tidak tersentuh — dan itu
dibuktikan, bukan diandaikan: ketiga kegagalan uji yang tersisa di `AccountPage.test.tsx` terbukti
tetap gagal ketika perubahan `DataTable.tsx` di-`git stash`.

Kehadiran `serverPaging` mematikan penyaringan DAN pengurutan internal sekaligus. Ketiganya harus
berubah bersamaan karena tabel hanya memegang satu halaman: menyaringnya menghasilkan "3 dari 10
baris cocok" padahal yang cocok di seluruh tabel ada 84 — bukan sekadar kurang berguna, melainkan
menyesatkan.

Ini **tidak mendahului `TKT-U2-005`** (pemilihan pustaka tabel): yang bertambah adalah permukaan
komponen, bukan pustakanya.

> **Koreksi saat merge ke `master` (2026-09-22).** Prop `serverPaging` beserta `PagingBar` yang
> dijelaskan di atas **tidak jadi masuk**. `master` sudah lebih dulu menumbuhkan `DataTable`
> dengan API paginasi servernya sendiri — `pagination: ServerPagination` beserta `PageBar` — dan
> enam layar di sana sudah memakainya. Dua API paginasi pada satu komponen akan membatalkan alasan
> komponen itu ada, sehingga `ClaimReportInboxPage` disesuaikan memakai `pagination`.
>
> Yang berubah hanya pemetaan namanya: `pageSize` → `size`, `totalPages` → `totalPage`.
> `hideSearch` tetap, karena `master` sudah memilikinya dengan arti yang sama.
>
> **Satu perbedaan perilaku yang ikut terbawa dan disadari:** pada `master`, yang mematikan
> penyaringan dan pengurutan internal adalah `serverSearch`, **bukan** `pagination`. Layar ini
> memakai `hideSearch` sehingga penyaringan tidak berjalan, tetapi **pengurutan kolom masih
> bekerja pada halaman yang sedang terbuka saja**. Itu persis keberatan yang ditulis di atas, dan
> ia berlaku sama pada seluruh inbox `master` yang sudah ada — bukan khusus layar ini. Menutupnya
> adalah pekerjaan tersendiri di `DataTable`, bukan bagian dari merge ini.

### 36.10 Yang belum dapat dibuktikan

| Hal | Sebabnya |
|---|---|
| **Kueri terhadap Oracle yang sebenarnya** | Migrasi `0003` belum dijalankan di lingkungan mana pun, dan tabel warisannya hanya ada di basis data produksi. Yang terbukti hari ini: bentuk kueri lolos pemeriksaan pola, dan seluruh perilakunya lolos uji terhadap penyimpanan memori |
| **Nama kolom nama pelapor pada tabel warisan** | Tidak satu pun dari sembilan kueri lama menyentuhnya (`R-08`). Kolomnya dibaca `NULL` untuk baris warisan |
| **Nama kanwil** | `POOLDATA.BRANCH` hanya menyimpan kodenya di `BASTERRITORY`; tabel namanya tidak ikut dikirim bersama export. Kodenya dipakai sebagai nama, dan itu dicatat di repo — bukan ditambal nama karangan |

### 36.11 Pertanyaan terbuka untuk Work Owner

| # | Pertanyaan | Kenapa ia penting |
|---|---|---|
| 1 | **Apakah memilih Kanwil MENGGANTIKAN batas cabang petugas, atau menyempit di dalamnya?** | `SetListRCV_Act` menyusun kedua penyaring, dan rule When yang memilih di antaranya **hilang dari export** (bagian dari 137 When rule, `R-16`). Yang diberlakukan sekarang: kanwil menggantikan batas cabang — **andaian**, ditandai sebagai andaian di `usecase.buildFilter`. Bila salah, petugas melihat berkas cabang yang bukan haknya |
| 2 | **Apa nama kelompok bisnis `10008`, `10010`, `10015`, `10023`?** | Ia dipakai sebagai kelompok tersendiri yang dikecualikan dari Non-MBU, tetapi **tidak ada satu pun rule yang menyebut namanya** dan `POOLDATA.BUSINESSGROUP` tidak ikut dikirim. Labelnya kini "Kelompok bisnis khusus" — menebaknya menjadi "Kredit" atau "Bonding" akan membuat petugas mengira sedang melihat lini yang bukan |
| 3 | **Apakah berkas tanpa cabang boleh dibuat?** | Lihat §17.6. Perilaku Pega dipertahankan; berkasnya tidak terlihat petugas cabang mana pun |
| 4 | **Berapa baris maksimum yang wajib dilayani satu ekspor?** | `ADR-0011`. Sampai dijawab, batasnya 50.000 dan berkas yang menyentuhnya diberi tanda |
| 5 | **Apakah selisih lencana "Replied from ASM" diperbaiki atau direplikasi?** | Lihat §17.4 baris kedua. Ia cacat yang sudah ada; memperbaikinya mengubah angka yang selama ini dibaca petugas |

### 36.12 Utang teknis yang bertambah

1. **Kontrak galat masih per modul.** Modul ini memetakan galatnya sendiri, sama seperti
   `masterstatus` dan `masterstatusprogres`, karena `TKT-F1-004` belum diputuskan. Bentuk `detail`
   yang dipakainya mengikuti `masterstatusprogres` (`{kolom, pesan}`) — bentuk ketiga **tidak**
   ditambahkan.
2. **Tipe TypeScript ditulis tangan.** `09-API-STRATEGY.md` §6 menetapkan tipe dihasilkan dari
   kontrak OpenAPI yang belum ada. Utang yang sama dimiliki seluruh modul sebelumnya.
3. **Jalur tanpa `/v1`.** Mengikuti kontrak yang sudah ada; memperkenalkannya di satu modul akan
   membuat dua gaya jalur hidup berdampingan.
4. **Tidak ada jejak audit** atas pembuatan berkas. `S-5` belum ada; kolom `DIBUAT_OLEH` dan
   `DIBUAT_PADA` pada tabel baru adalah yang terdekat dengannya hari ini.

---

## 37. Modul Inbox Claim Treaty Non Prop (2026-09-22, sesi kesembilan belas)

Menu `MENU_ID 55`, pengganti harness `InboxClaimNonProp_Harness`. Layar saudara dari
`MENU_ID 54` yang dibangun sesi sebelumnya.

### 37.1 Pertanyaan konfirmasi dan jawabannya

Ketiganya diajukan **sebelum satu baris kode ditulis**, seluruhnya disertai bukti.

| # | Pertanyaan | Jawaban Work Owner |
|---|---|---|
| 1 | Tombol ekspor (`GenerateClaimNonPropCSV`) dibangun sekarang atau ditunda? Report Definition-nya hilang dari export, tetapi susunan kolomnya terbaca dari activity-nya | **Bangun sekarang dari activity** |
| 2 | "See TBA Claim" di Pega hanya berpengaruh bila "See All Claim" ikut dicentang (dua prakondisi ber-AND). Replikasi atau perbaiki? | **Perbaiki — TBA berdiri sendiri** |
| 3 | Tab Komite dipasok kueri yang hilang dari export. Bagaimana? | **Terhalang, sama seperti modul Prop** |

### 37.2 Kenapa modul ini BUKAN salinan modul Prop

Keduanya mirip di layar dan namanya hanya berbeda dua kata. Di bawah permukaan keduanya
berbeda pada hal-hal yang menentukan kuerinya:

| | Prop (`MENU_ID 54`) | Non Prop (paket ini) |
|---|---|---|
| Penanda objek kerja | `PXREFOBJECTKEY LIKE '%CLMP%'` | `PXREFOBJECTINSNAME LIKE 'CLMNP-%'` |
| Jumlah tabel | 2 | **3** |
| Asal kolom bisnis | `JSON_VALUE(DATA_JSONBLOB, …)` | **kolom `PC_ASM_FW_GCNMFW_WORK`** |
| Kolom JSON yang dibaca | `DATA_JSONBLOB` | **`DATA_JSON`** |
| Varian tab pertama | 2 | **4** (3 di Pega + 1 baru) |
| Kolom tambahan | — | Status, Aging, Create/Last Update Operator |

Menyatukan keduanya akan memaksa satu kueri melayani dua bentuk data yang tidak sama —
persis cacat yang sedang ditinggalkan (utang teknis 4.6).

### 37.3 Dua kolom JSON berbeda pada satu tabel

`POOLDATA.JSON_KLAIM` punya **dua** kolom JSON, dan kedua layar treaty membaca yang
berbeda:

```
DATA_JSONBLOB   dibaca layar Prop      (GetClaimTreaty_SQL:104-109)
DATA_JSON       dibaca layar ini       (GetKlaimNonPropAdmin_SQL:43-44)
```

Keduanya **tidak disamakan**. Apakah isinya sama tidak dapat diperiksa: DDL-nya belum
tersedia (`R-08`). Menukar salah satunya ke yang lain adalah perubahan yang **tidak
menghasilkan galat apa pun bila salah** — ia hanya menampilkan tanggal dan ID master milik
dokumen yang berbeda.

Yang diubah hanya **sintaksnya**, bukan kolomnya: notasi titik Oracle
(`c.data_json.DateOfLoss`) menjadi `JSON_VALUE(c.DATA_JSON, '$.DateOfLoss')`, supaya kueri
tetap satu set untuk Oracle dan PostgreSQL (`D-20`, `D-24`).

### 37.4 Dua kolom "master id" dipertahankan terpisah

Layar lama menampilkan **keduanya berdampingan**, dan keduanya dari tempat berbeda:

| Kolom grid | Asal | Alias Pega |
|---|---|---|
| "MasterID" | `b.MASTERID` — kolom objek kerja | `CARI19` |
| "ID Master" | `c.DATA_JSON` jalur `$.IDMaster` | `CARI23` |

Menyatukannya berarti memutuskan salah satu yang benar — keputusan yang tidak dapat diambil
tanpa melihat isi kedua sumbernya. Bila kelak terbukti selalu sama, menyatukannya sepele;
bila ternyata berbeda, menyatukannya sekarang menyembunyikan ketidakcocokan data yang
justru perlu ketahuan.

Satu baris data contoh sengaja memuat nilai yang **berbeda** pada keduanya, supaya
penyatuan diam-diam di kemudian hari langsung ketahuan.

### 37.5 "See TBA Claim" dibuat berdiri sendiri — selisih terencana `P-5`

**Yang terbukti di export.** `Activity/GetDataTreatyinNonProp_Act-Act.xml` langkah 4
dijaga **dua** prakondisi ber-AND:

```
Inputdata.CARI10 == ""    <- "See All Claim" tercentang
Inputdata.CARI11 == ""    <- "See TBA Claim" tercentang
```

Akibatnya mencentang "See TBA Claim" sendirian **tidak menjalankan kueri apa pun yang
berbeda** — layar tetap menampilkan hasil langkah 2.

**Keputusan Work Owner 2026-09-22:** keduanya menjadi penyaring yang saling bebas. Ia
melahirkan kueri kelima yang tidak ada di sistem lama — `list_admin_tba` — dan dinyatakan
ke pengguna lewat `PlannedDifferences`, bukan hanya dicatat di komentar.

### 37.6 Kolom "Status" tidak menyatakan status klaim

Ia **literal** di dalam kueri, bukan kolom: `Estimation` untuk seluruh baris tab Admin,
`Acceptation` untuk seluruh baris tab Teknik. Tidak ada satu pun kolom status yang dibaca.

Artinya kolom bernama "Status" itu menyatakan **antrean mana baris ini berasal**. Tak satu
pun dari kedua teks itu termasuk dalam 33 kode status klaim yang sebenarnya (`R-06`).

Perilakunya **dipertahankan apa adanya** (`P-5`); yang ditambahkan hanyalah keterangan ke
pengguna, karena kolom bernama "Status" yang tidak menyatakan status adalah hal yang wajar
disalahpahami.

### 37.7 "Aging" adalah hari kalender, bukan TAT

`TRUNC(SYSDATE) - TRUNC(b.PXCREATEDATETIME)` menghasilkan selisih hari kalender apa adanya:
akhir pekan dan hari libur ikut terhitung. Ia **bukan** TAT — perhitungan TAT memotong jam
kerja lewat `GET_WORKING_HOURS` dan `HRD_LBR` (`D-50`), dan layar ini tidak menyentuh
keduanya.

**Bentuknya diportabelkan tanpa mengubah hasilnya.** `SYSDATE` dan `TRUNC(tanggal)`
keduanya ada di daftar padanan wajib `09-DATABASE-STRATEGY.md` §4, dan uji disiplin SQL
milik repo ini melarang keduanya. Yang dipakai:

```sql
CAST(CURRENT_TIMESTAMP AS DATE) - CAST(b.PXCREATEDATETIME AS DATE)
```

Pemangkasan kedua sisi dipertahankan — tanpanya selisih dihitung dari JAM, sehingga
pekerjaan yang dibuat kemarin sore terhitung nol hari sampai lewat 24 jam.

### 37.8 Dua hal di layar lama yang TIDAK dibawa

| Yang tidak dibawa | Bukti | Alasan |
|---|---|---|
| **Nama orang menentukan kewenangan komite** | `GetWorkCNP_Act` langkah 4 membandingkan `OperatorID.pyUserIdentifier` dengan satu Operator ID yang ditanam di rule | Salah satu dari 24 Operator ID hardcode `D-15`. Namanya **tidak disalin** ke repo (`D-69`) |
| **Fragmen SQL dirangkai dari string** | `GetWorkCNP_Act` menyusun klausa `WHERE` dengan penggabungan teks | Pola `{ASIS:…}`, celah injeksi (utang teknis 4.5). Larangan `08-TECHNICAL-STRATEGY.md` §4.3 tidak dikecualikan keputusan mana pun |

### 37.9 Ekspor: namanya CSV, keluarannya bukan

`GenerateClaimNonPropCSV` berakhir dengan `call MSOGenerateExcelFile` atas halaman
`TempData` — berkas yang benar-benar diunduh pengguna adalah **berkas Excel**, meski nama
rule-nya menyebut CSV.

Yang dibangun di sini menghasilkan **CSV sungguhan**: ia dibuka Excel tanpa perantara,
tidak menuntut pustaka pihak ketiga, dan dapat dialirkan potong demi potong — yang ketiga
tidak mungkin dilakukan penghasil Excel.

**Tujuh kolomnya dan urutannya** diambil dari langkah `Property-Set` activity itu:
`.CARI11` → `.CARI15` → `.CARI14` → `.CARI16` → `.CARI17` → `.CARI22` → `.CARI12`. Yang
berubah hanya **baris judulnya**: `MSOGenerateExcelFile` memakai nama properti sebagai
judul, sehingga berkas lama berjudul kolom `CARI1`…`CARI7`.

**Lima kolom yang ada di grid sengaja TIDAK ikut** — nomor polis, kedua master id, status,
aging, dan operator pengubah. Kelimanya tidak pernah ada di berkas ekspor sistem lama, dan
menambahkannya adalah kemampuan baru, bukan pemindahan.

### 37.10 Tab Komite terhalang — pemiliknya Tim Pega, bukan DBA

Berbeda dari tab komite modul Prop, yang menunggu DDL dari DBA. Di sini yang hilang adalah
**rule-nya sendiri**: `KmtGetInboxListCNP_SQL` DIPANGGIL `GetWorkCNP_Act` tetapi tidak ada
di export, sehingga tidak diketahui tabel mana yang dibacanya maupun kolom apa yang
dikembalikannya.

Menyusunnya sendiri dari pola kedua kueri lain berarti **menebak aturan yang menentukan
persetujuan nilai uang**. Tab-nya karena itu digambar dengan alasan dan pemiliknya, dan
ditolak di **domain** (`NewQuery`) — bukan di penyimpanan, yang akan menjadikannya galat
500 tanpa sebab yang terbaca.

### 37.11 Yang dibangun

| Lapisan | Berkas |
|---|---|
| Domain | `inboxclaimtreatynonprop.go`, `tab.go`, `query.go`, `errors.go` |
| Usecase | `usecase/list.go` |
| Repo | `repo/sqlstore/` (5 kueri daftar + 2 pemeriksa), `repo/memory/` |
| Transport | `http/` — dto, errors, handler, **export**, routes |
| Uji | 12 uji aturan modul + 18 uji kueri + 17 uji layar |
| Perakitan | `cmd/claimpnc/main.go` (8 titik), `check.go` (pemeriksa tersendiri) |
| Frontend | `modules/inbox-claim-treaty-non-prop/`, rute `App.tsx`, peta `registry.ts` |

**Tidak ada migrasi basis data.** Seluruh tabel yang dibaca milik sistem lama dan sudah
ada; modul ini **tidak menulis satu pun** (`P-1`).

### 37.12 Yang diverifikasi

| Pemeriksaan | Hasil |
|---|---|
| `go build ./...` | lulus |
| `go vet` modul baru | bersih |
| `gofmt -l` modul baru | bersih |
| `go test ./...` | **111 paket lulus, 0 gagal** |
| `npx tsc --noEmit` modul baru | **bersih** (120 galat lain pra-ada, dibuktikan) |
| `npx vitest run` modul baru | **17 uji lulus** |

---

## 38. Modul Inbox Manager Receive / PUCL (2026-09-22 … 2026-09-23, sesi kedua puluh)

Menu `MENU_ID 56` "Inbox Manager Receive / PUCL", pengganti harness
`ReceiveDoucument_Harness`. Prosesnya ada di
[`catatan-pengembangan.md`](catatan-pengembangan.md) §36; berkas ini merekam
**keputusannya**.

### 38.1 Pertanyaan konfirmasi dan jawabannya

Tiga diajukan sebelum satu baris kode ditulis. Ketiganya mengubah lingkup secara material,
dan tidak satu pun dapat dijawab dari export.

| # | Pertanyaan | Jawaban Work Owner |
|---|---|---|
| 1 | `.ReceiveDocument.TypeOfClaim` — pembeda kedua grid Receive — ditandai `unexposed` di Report Definition-nya, sehingga SQL tidak dapat menyaringnya. Bagaimana? | **Pakai Group Panel sebagai pengganti** |
| 2 | Filter `pxAssignedOrgUnit = Param.OrgUnit` ada di RD, tetapi parameternya tidak pernah diisi satu pun activity. Apa yang berlaku? | **Ikuti export apa adanya — tanpa saring** |
| 3 | Lingkup tulis modul ini | **Baca saja + ekspor** |

### 38.2 Jenis Klaim diturunkan dari Group Panel

**Buktinya, dan kenapa tidak ada jalan lain.**
`Report Definition/ManagementRecieveView-RD.xml` menandai propertinya sendiri sebagai
`pzPropertyType` bernilai `unexposed`, ditambah dua peringatan Pega — "Not optimized for
reporting" dan "Not optimized for filtering" — yang **keduanya menyebut properti itu
satu-satunya**.

Penelusuran seluruh export menguatkannya: `TYPEOFCLAIM_1`, `SENDER_1`, dan
`NUMBEROFDOCUMENT_1` **nol kemunculan**, sementara 15 kueri lain pada kelas yang sama
memakai `STATUSLOCK_1`, `KODECABANG_1`, `DATEFORAGING_1`, `BUSINESSCODE_1`, `DATEOFLOSS_1`,
`BOOKNO_1`, dan `GROUPPANEL_1`.

**Penggantinya.** `GROUPPANEL_1` punya kolom, memang dibaca kueri lain pada kelas yang sama
(`RDB List/GetDataRCVallKlaimPATravel-SQL.xml:10`), dan memang membedakan Personal Accident
dari lini lain — `002` adalah PA (`CONTEXT.md`, Business Understanding §1).

**Akibat yang diterima secara sadar:** berkas yang Group Panel-nya kosong tidak muncul di
tab mana pun. Pembanding ketidaksamaan tidak menangkap NULL di Oracle maupun PostgreSQL.
Itu perilaku yang **sama** dengan layar lama, tempat berkas tanpa `TypeOfClaim` tidak cocok
dengan grid mana pun — dan diuji tegas lewat
`TestDocumentWithoutGroupPanelAppearsInNoReceiveTab` supaya keadaan itu menjadi keputusan
yang tercatat, bukan kebetulan yang kelak "diperbaiki".

Penerjemahannya tinggal di **domain** (`ClaimTypeOf`), bukan di SQL: penyimpanan SQL dan
penyimpanan memori wajib menghasilkan teks yang sama persis, kalau tidak uji yang lulus di
atas memori tidak menyatakan apa pun tentang Oracle.

### 38.3 Tab RCL/PUCL menyaring antrean bersama — penyaring yang TIDAK ada di RD-nya

Ini keputusan yang paling jauh dari "salin apa adanya", dan alasannya perlu dibaca utuh.

`Report Definition/InboxRCLPUCL_RD-RD.xml`:

- `pyJoinInfo`-nya **kosong** — tidak ada gabungan sama sekali;
- satu-satunya filter: `pyStatusWork` tidak sama dengan `Resolved-Completed`;
- punya parameter bernama `assign` yang **dideklarasikan tetapi tidak dirujuk satu filter
  pun**.

Ditiru apa adanya, tab itu menampilkan **seluruh klaim PNC yang belum selesai**. Pada basis
data berisi puluhan juta baris (`D-10`), hasilnya bukan sekadar keliru melainkan tidak dapat
dipakai.

Dua kueri Pega pada domain yang **sama** menyaringnya dengan cara yang sama persis:

| Rule | Penyaring |
|---|---|
| `RDB List/CountKlaimPUCL-SQL.xml` | `PXASSIGNEDOPERATORID` = akun antrean RCLPUCL |
| `RDB List/ReminderPUCL-SQL.xml` | `PXASSIGNEDOPERATORID` = akun antrean RCLPUCL |

**Keputusan:** penyaring itu dipakai. Arahnya **menyempitkan** — lebih sedikit baris, bukan
lebih banyak — sehingga kekeliruan yang mungkin tersisa tidak dapat membocorkan baris yang
seharusnya tersembunyi. Nama parameter `assign` yang menganggur itu sendiri adalah petunjuk
bahwa penyaringnya memang pernah ada dan hilang.

Ia dinyatakan ke pengguna lewat `PlannedDifferences`, **bukan** disembunyikan sebagai detail
kueri, dan `-periksa` menyebutnya eksplisit bila tab itu kosong.

**Tiga penyaring yang TIDAK ikut dibawa** dari `ReminderPUCL-SQL.xml`:
`TANGGALCETAKDOKUMENPUCL_1 IS NOT NULL`, `PUCLAPPROVE_1` tidak sama dengan 1, dan `MSIG_1`
kosong. Ketiganya milik **job pengingat** — klaim yang suratnya sudah dicetak tetapi belum
disetujui — bukan milik antrean inbox. Membawanya akan menyembunyikan klaim yang suratnya
belum dicetak, padahal justru itu yang menunggu tindakan. Dijaga
`TestReminderOnlyFiltersAreNotCarried`.

### 38.4 Dua grid bertumpuk menjadi tiga tab

`Section/InboxManagerReceive_Section-Section.xml` punya **dua** judul tab (`Receive` dan
`RCL/PUCL`) tetapi **tiga** grid: tab "Receive" memuat `ManagementRecieveView` **dua kali**,
satu dijalankan dengan parameter `Position1` bernilai `PA` dan satu dengan `NONMBU`.
Keduanya ber-`pyVisible` `ALWAYS`, dan **keduanya tanpa judul apa pun** — penelusuran
seluruh section tidak menemukan satu pun label di antara keduanya.

**Keputusan: tiga tab, dan kedua tab Receive diberi judul.** Dua alasan:

1. **Paginasi.** Masing-masing grid punya halamannya sendiri (`pyPageSize` 50, penomoran
   Numeric). Dua tabel berhalaman yang bertumpuk menghasilkan dua penomoran yang mudah
   tertukar.
2. **Kejujuran.** Dua tabel berkolom identik tanpa judul adalah cacat tampilan yang tidak
   perlu dibawa — dan di layar ini akibatnya nyata, karena yang membedakan keduanya adalah
   lini bisnis klaimnya.

Polanya bukan hal baru: modul Inbox Claim Treaty Non Prop menempuh hal yang sama pada tiga
kontainer yang di Pega dipilih oleh keadaan pemanggil (§37).

Judul yang **ada** di Pega dipertahankan apa adanya, termasuk garis miring pada "RCL/PUCL".
Judul kedua tab Receive memakai ejaan yang sama dengan nilai parameter RD-nya — "PA" dan
"NONMBU" — bukan ejaan yang lebih rapi.

### 38.5 Dua kolom dibaca dari tabel cermin yang tidak pernah dibaca sistem lama

"Nama Pengirim" dan "Tanggal Terima Dokumen" tidak punya kolom pada objek kerja. Yang ada
adalah `POOLDATA.T_CLAIM_RECIVEDCLAIM`, diisi `Database/PROCINSERTDATARECIVEDKLAIM.prc`
dengan kunci `CLAIMID` yang berisi `pzInsKey` apa adanya.

**Bahwa `NAMAPELAPOR` memang "nama pengirim"** terbaca dari label layar input:
`Section/ViewInputReceiveDocument_sec-Section.xml` memberi `.ReceiveDocument.Sender` judul
**"Nama Pengirim / Pelapor Dokumen"**. Ia BUKAN `NAMAKURIRASM`, yang berjudul "Nama Kurir
ASM". Tanpa label itu keduanya sama-sama masuk akal, dan pilihan yang salah tidak
menghasilkan galat apa pun.

**Risiko yang disadari:** tabel itu **tidak pernah dibaca** sistem lama — satu-satunya
penyentuhnya adalah procedure yang menulisinya. Kelengkapan isinya belum terverifikasi,
sehingga gabungannya `LEFT JOIN`: baris tanpa pasangan tetap muncul dengan kedua kolom
kosong, bukan hilang. `-periksa` menyebutnya bila seluruh baris begitu.

`TANGGALTERIMADOKUMEN` dibawa sebagai **teks**, dan itu bukan pilihan: parameter procedure
yang mengisinya bertipe `varchar2` sementara parameter tanggal lain pada procedure yang sama
bertipe `DATE`.

### 38.6 Satu kolom yang SELALU kosong, dan tetap digambar

"Jumlah Lembar Dokumen" (`.ReceiveDocument.NumberOfDocument`) tidak punya kolom basis data
mana pun, dan `T_CLAIM_RECIVEDCLAIM` tidak menyimpannya — procedure yang mengisinya menerima
26 parameter dan tidak satu pun berisi jumlah lembar.

**Kolomnya tetap digambar.** Menghilangkannya membuat layar tampak setara dengan Pega
padahal ada isian yang belum terbawa — persis yang tidak boleh terjadi pada uji kesetaraan
gerbang 1. Selnya digambar sebagai tanda pisah, yang menyatakan "tidak ada isinya" alih-alih
"gagal dimuat".

### 38.7 Filter unit organisasi tidak dibawa

`newAssignPage.pxAssignedOrgUnit = Param.OrgUnit` ada di RD, tetapi section mengirim
`OrgUnit` **kosong** dan tidak ada satu pun activity di seluruh export yang mengisinya.
Penyaringnya karena itu **tidak pernah berlaku** — ia tidak dihilangkan, melainkan memang
tidak ada.

**Akibatnya pada bentuk modul:** ini satu-satunya layar inbox yang **tidak satu pun tabnya
menyaring menurut pemanggil**. Modul inbox lain setidaknya punya satu tab "milik saya".
Konsekuensinya dua, dan keduanya ditangani:

1. Sifat itu **dinyatakan di layar**, bukan hanya di kode — tanpa keterangan, petugas yang
   terbiasa dengan inbox lain akan mengira daftarnya keliru karena memuat pekerjaan orang
   lain.
2. **Setiap** pembukaan dicatat, bukan hanya yang mencurigakan. Modul lain mencatat saat
   penyaring kepemilikan dilepas; di sini penyaring itu memang tidak pernah ada. Sampai
   `TKT-F3-004` selesai, jejak itulah satu-satunya kontrol pengimbang (`D-59`).

Identitas pemanggil karena itu tetap **wajib** meski tidak dipakai menyaring: permintaan
tanpa identitas ditolak, karena pembukaan layar ini harus tercatat atas nama seseorang.

### 38.8 Ekspor adalah kemampuan BARU

Layar lama **tidak punya** tombol ekspor: tidak ada activity ekspor yang dirujuk harness
maupun section-nya, dan tidak ada rule `Generate*CSV` maupun `MSOGenerateExcelFile` di
antara keduanya.

Penambahannya diputuskan Work Owner, dan dinyatakan ke pengguna sebagai selisih terencana —
bukan disajikan seolah fitur yang dipindahkan.

Susunan kolom berkasnya **mengikuti tab yang sedang terbuka**, karena ketiga tab punya kolom
yang berbeda. Judul dan barisnya dibangun dari **senarai kolom yang sama**, bukan dari dua
daftar yang kebetulan sejalan — penambahan kolom di `tab.go` karena itu tidak dapat
menggeser isi berkas tanpa menggeser judulnya sekaligus.

### 38.9 `INNER JOIN` dipertahankan, meski ia dapat menggandakan baris

Report Definition-nya memakai gabungan dalam ke tabel penugasan, sehingga objek kerja yang
punya **dua** penugasan terbuka muncul dua kali. Itu perilaku sistem lama apa adanya
(`P-5`).

Menggantinya dengan `EXISTS` akan mengubah jumlah baris yang terlihat pengguna **tanpa satu
pun keputusan yang mendasarinya**. Ia dicatat di kepala berkas `.sql`, bukan diperbaiki
diam-diam.

### 38.10 Yang dibangun

| Lapisan | Berkas |
|---|---|
| Domain | `inboxmanagerreceivepucl.go`, `tab.go`, `query.go`, `errors.go` |
| Usecase | `usecase/list.go` |
| Repo | `repo/sqlstore/` (3 kueri daftar + 2 pemeriksa), `repo/memory/` |
| Transport | `http/` — dto, errors, handler, export, routes |
| Uji | 16 uji aturan modul + 19 uji kueri + 12 uji layar |
| Perakitan | `cmd/claimpnc/main.go` (8 titik), `check.go` (pemeriksa tersendiri) |
| Frontend | `modules/inbox-manager-receive-pucl/`, rute `App.tsx`, peta `registry.ts` |

**Tidak ada migrasi basis data.** Seluruh tabel yang dibaca milik sistem lama dan sudah ada;
modul ini **tidak menulis satu pun** (`P-1`).

### 38.11 Yang diverifikasi

| Pemeriksaan | Hasil |
|---|---|
| `go build ./...` | lulus |
| `go vet` modul baru + `cmd/...` | bersih |
| `gofmt -l` modul baru | bersih |
| `go test ./...` | seluruh paket lulus, 0 gagal |
| `npx tsc --noEmit` modul baru | **bersih** (120 galat lain pra-ada) |
| `npx vitest run` modul baru | **12 uji lulus** |
| Baseline kegagalan uji frontend lain | **dibuktikan pra-ada** — lihat `catatan-pengembangan.md` §36.7 |
## 18. Form Input Receive Document (2026-09-22, sesi kesepuluh)

Melengkapi §17. Tombol "Buat Baru" yang sebelumnya menerbitkan berkas kosong tanpa tempat
mengisinya kini membuka form yang mengisinya — persis seperti alur
`Flow/InputReceiveDocument.xml`.

### 18.1 Apa yang salah sebelumnya

Bukan bug pada permintaannya: `POST` menjawab `201` dan berkasnya tersimpan. Yang salah adalah
**lingkup yang saya tetapkan sendiri** pada sesi sebelumnya.

Saya membaca `CreateNewCaseRCV` dan menyimpulkan benar bahwa ia membuat berkas KOSONG, lalu
berhenti — tanpa menelusuri ke mana berkas itu pergi sesudahnya. Jawabannya ada di alur yang
memanggilnya: berkas lahir kosong JUSTRU supaya assignment "Receive Document" yang mengisinya.

> **Pelajarannya, dicatat supaya tidak terulang:** membaca activity yang MEMBUAT sesuatu tidak
> cukup. Yang menentukan artinya adalah alur yang menerimanya.

### 18.2 Sumber yang dipakai, dan satu yang tidak ada

| Artefak | Keadaan |
|---|---|
| `Flow/InputReceiveDocument.xml` | ada — Start → assignment `Receive Document` (WorkList, `PNCAdminRouterRCV`) → End |
| `Flow Action/InputReceiveDocument-FlowAction.xml` | ada — merender section `InputReceiveDocument` |
| **Section `InputReceiveDocument`** | **TIDAK ADA di export** (`R-16`) |
| `Section/ViewInputReceiveDocument_sec` | ada, 1,24 MB — **dipakai sebagai pengganti sumber** |
| `Database/PROCINSERTDATARECIVEDKLAIM.prc` | ada — menyilangkan isian form dengan kolom yang benar-benar disimpan |

Section formnya sendiri hilang. Yang dipakai adalah **varian tampilnya**, yang memuat label dan
properti terikat yang sama persis. Itu bukti terbaik yang tersedia, dan keterbatasannya ditulis di
kepala `detail.go` alih-alih ditutupi.

### 18.3 Lingkup form ditentukan oleh bentuk keterikatannya

Properti pada section membelah dirinya sendiri, dan pembelahan itu yang dipakai — bukan penilaian
saya tentang mana yang penting:

| Kelompok | Terikat sebagai | Keputusan |
|---|---|---|
| 17 isian `.ReceiveDocument.*` | field tunggal | **dibawa** |
| Blok pelapor + alamat | `.ReportHE.*`; alamatnya page list | **tidak** — `ReportHE` adalah area **Heavy Equipment**, dan `D-34` mengeluarkan Bengkel/Sparepart/Supplier beserta area HE dari lingkup migrasi |
| Grid dokumen | page list `.ReceiveDocument.DocumentList` | **tidak** — menuntut `S-1`/`D-16`; angka totalnya tetap dibawa |
| Riwayat komunikasi & progres | page list `tempHistoryKomunikasi`, `tempViewProgress` | **tidak** — menampilkan data milik modul lain, bukan isian form |

### 18.4 Pemeriksaan isian: penjaga penyimpanan, bukan aturan bisnis

Flow action `InputReceiveDocument` **tidak punya satu pun** validate rule maupun isian bertanda
wajib — diperiksa langsung ke berkasnya. Pega menyimpan apa pun yang diketik.

Yang diperiksa di sini karena itu hanya dua hal:

| Pemeriksaan | Kenapa ia bukan aturan bisnis |
|---|---|
| **Panjang teks** terhadap lebar kolom migrasi 0004 | Tanpa ia, Oracle menolak dengan `ORA-12899` yang tidak menyebut isian mana. Dihitung dalam RUNE, bukan byte — satu huruf beraksen memakan dua byte, dan nama orang Indonesia memuatnya |
| **Angka tidak negatif**, jumlah dokumen ≤ 9999 | Penjaga salah ketik. Angka enam digit pada "Total Jumlah Dokumen" hampir pasti bukan jumlah dokumen |

**Aturan tanggal sengaja TIDAK ada.** DOL di dalam periode polis, Tanggal Lapor ≤ DOL + 7 hari, dan
seterusnya adalah milik `B-2` (`02-BUSINESS-UNDERSTANDING.md` §3.1). Berkas laporan justru sering
masuk sebelum tanggalnya dipastikan; menambahkannya di sini akan menolak berkas yang di Pega
diterima — perubahan perilaku yang tidak ada di 13 butir `D-49`.

### 18.5 Berkas Pega dibuka baca saja

`Repo.Update` menolak baris ber-`Origin` warisan dengan `ErrReadOnlyOrigin`, dan penolakannya ada
di **kedua** pengisi seam — SQL maupun memori. Seam yang kedua pengisinya berperilaku berbeda
adalah seam yang menyembunyikan cacat.

Penolakannya terjadi **sebelum** satu pun isian diperiksa: memberi pengguna daftar isian yang harus
diperbaiki pada form yang memang tidak dapat disimpan sama sekali adalah menyesatkan.

**Kewenangannya dikirim server** sebagai `dapat_disunting`, bukan disimpulkan layar dari kolom
`asal`. Aturan siapa yang boleh menulis milik server; menyalinnya ke layar berarti satu aturan hidup
di dua tempat, dan yang di layar akan tertinggal saat yang di server berubah.

### 18.6 Kode galat tersendiri untuk penolakan itu

Versi pertama menjawab `validasi_gagal` dengan `409`. **Dikoreksi menjadi `laporan_hanya_baca`.**

Ia bukan kegagalan validasi: tidak ada satu pun isian yang dapat diperbaiki pengguna. Klien
membedakan jenis galat lewat `kode`, dan memakai kode validasi akan membuat layar menunggu `detail`
yang tidak pernah datang — lalu menampilkan form yang tampak dapat diperbaiki padahal tidak.

`409`, bukan `403`: yang menolak bukan kewenangan pengguna melainkan **keadaan berkasnya**.
Pengguna yang sama dapat menyimpan berkas lain tanpa masalah.

### 18.7 Migrasi 0004 berdiri sendiri, bukan suntingan pada 0003

0003 sudah diserahkan sebagai permintaan perubahan skema (`D-63`) dan mungkin sudah dijalankan DBA
di salah satu portal. Menyuntingnya berarti dua orang memegang berkas bernomor sama dengan isi
berbeda — dan yang menjalankan versi lama tidak punya cara mengetahuinya.

0004 karena itu bersifat **menambah** dan aman dijalankan apa pun keadaannya, selama 0003 sudah
lebih dulu. Seluruh kolomnya NULLABLE, sehingga versi aplikasi yang lama tetap berjalan terhadap
skema ini (`P-4`).

### 18.8 Kueri detail memilih lebih banyak kolom daripada kueri daftar

Sepuluh kolom lebih banyak, dan dua di antaranya berlebar 4.000 karakter. Menariknya pada setiap
halaman daftar berarti memindahkan ratusan kilobita yang tidak pernah digambar grid, pada tabel
berpuluh juta baris (`D-10`).

Biayanya: dua pembaca baris, bukan satu. Yang menjaganya adalah uji yang menuntut **urutan kolom
daftar tetap menjadi AWALAN kolom detail** — `Scan` membaca secara posisi, dan menyisipkan kolom
baru di tengah alih-alih di ujung akan menggeser salah satunya tanpa menghasilkan galat.

### 18.9 Utang teknis yang bertambah

1. **Batas panjang isian hidup di dua tempat** — `detail.go` dan `types.ts`. Server tetap yang
   berwenang; yang di frontend hanya kenyamanan. Bila salah satu berubah, keduanya harus ikut.
2. **Tipe `Money` diulang**, tidak diimpor dari modul `registrasi`. Paket domain satu modul tidak
   boleh bergantung pada paket domain modul lain; tipe uang bersama adalah `TKT-U2-004` yang belum
   ada.
3. **`UpdatedBy`/`UpdatedAt` bukan jejak audit.** Jejak audit adalah `S-5` yang mencatat nilai
   sebelum dan sesudah (`D-28`). Yang ini hanya menjawab "siapa terakhir menyentuh berkas ini".

### 18.10 Pertanyaan terbuka yang bertambah

| # | Pertanyaan | Kenapa ia penting |
|---|---|---|
| 6 | **Apakah berkas yang sudah diserahkan masih boleh disunting?** Form sekarang menerima penyimpanan pada berkas berposisi apa pun selama ia milik aplikasi ini. Di Pega, assignment-nya hilang setelah berkas berpindah tahap — sehingga formnya tidak lagi dapat dibuka. Perilaku itu **tidak** direplikasi karena mekanisme assignment-nya tidak dibawa, dan pembatasannya menuntut keputusan Work Owner |
| 7 | **Siapa yang boleh mengisi form ini?** Sekarang setiap pengguna yang dapat membuka layarnya. Kewenangan per peran adalah `TKT-F3-005` yang belum ada; keadaannya sama dengan seluruh layar lain hari ini |

---

## 19. Penerjemahan cabang klaim (2026-09-22, sesi kesebelas)

Melengkapi §17 dan §18. Daftar Inbox Laporan Klaim tampil **kosong tanpa satu pun galat**, dan
sebabnya adalah satu nilai yang tampak benar namun berasal dari sistem penomoran yang berbeda.

### 19.1 Keputusan: penerjemahan login menjadi cabang adalah seam, bukan fungsi pembantu

**Pilihan yang ditimbang**

| # | Pilihan | Kenapa ditolak / diambil |
|---|---|---|
| 1 | Menambah `JOIN` ke kueri daftar | Ditolak. Kueri daftar sudah membaca dua tabel lewat `UNION ALL`; menambah tiga tabel lagi — dua di antaranya lintas DB link — membuat setiap halaman daftar bergantung pada ketersediaan basis data lain |
| 2 | Fungsi pembantu di dalam `sqlstore` | Ditolak. `usecase` tetap tidak dapat diuji tanpa basis data, padahal aturan "cabang tidak terbaca berarti tidak disaring" adalah aturan **usecase**, bukan aturan SQL |
| 3 | **Seam `BranchResolver` yang dideklarasikan domain** | **Diambil** |

**Alasannya.** Menerjemahkan login menjadi kode cabang adalah **pengambilan data dari sistem
lain** — persis bentuk yang `04-FUTURE-ARCHITECTURE.md` §3.4 tetapkan sebagai seam. Ia juga
memenuhi syarat "dua adapter nyata, bukan satu": adapter SQL untuk produksi, adapter memori untuk
pengujian dan pengembangan lokal.

Dan ada alasan kedua yang lebih mendesak: `D-25` menetapkan seluruh DB link diganti pemanggilan
API, dan `R-03` mencatat API itu belum ada. Seam ini **adalah tempat penggantian itu kelak
terjadi** — satu adapter ditukar, `usecase` dan `domain` tidak tersentuh.

### 19.2 Keputusan: `Caller` tidak lagi membawa kode cabang sama sekali

`Caller.BranchCode` dihapus, bukan diperbaiki isinya.

**Kenapa dihapus dan bukan diisi dengan nilai yang benar.** Field bernama `BranchCode` pada profil
pemanggil akan diisi lagi oleh pembaca berikutnya dari sumber terdekat yang bernama sama — yaitu
`auth.User.BranchCode` dari HCQ. Selama namanya ada di sana, kesalahan yang sama dapat terjadi
lagi tanpa seorang pun berniat salah.

Ini penerapan `D-19` pada tempat yang tidak terduga: **nama yang tidak mencerminkan isi** adalah
utang teknis yang sama dengan alias kolom Pega, hanya saja kali ini di kode baru.

### 19.3 Keputusan: tiga keluaran penerjemahan dibedakan, bukan dua

```go
Resolve(ctx, login) (code string, resolved bool, err error)
```

| Keluaran | Arti | Perlakuan |
|---|---|---|
| `code`, `true`, `nil` | cabang diketahui | daftar disaring |
| `""`, `false`, `nil` | petugas tidak terdaftar di HRD | **bukan galat**; daftar tidak disaring |
| `""`, `false`, `err` | sumbernya tidak dapat dibaca | galat dicatat; daftar tidak disaring |

**Kenapa baris kedua bukan galat.** Login yang tidak terdaftar di HRD adalah keadaan data yang
wajar — pengguna non-karyawan sudah ada di sistem ini (`M_LOGIN_PNC`). Menjadikannya galat akan
menutup layar bagi orang yang berhak membukanya.

**Kenapa baris ketiga tetap galat meski akibatnya sama.** Akibatnya memang sama di layar, tetapi
perbaikannya berbeda jauh: yang satu urusan data HRD, yang lain hak baca atau DB link mati. Pada
sesi ini saya baru saja memperbaiki cacat yang lahir dari dua hal berbeda yang diperlakukan sama —
mengulanginya di tempat lain akan menjadi lelucon yang mahal.

### 19.4 Keputusan: cabang yang tidak terbaca TIDAK mengosongkan daftar

Ini **penyimpangan yang disengaja** dari perilaku sistem lama, dan karena `P-5` menuntut setiap
selisih dapat dipetakan, alasannya ditulis lengkap.

**Perilaku lama.** `GetIDCabang` tidak mengembalikan baris; nilai kosong disisipkan ke teks SQL
lewat `{ASIS:...}`; kueri menjadi `branch where ID = ''`; hasilnya nol baris.

**Kenapa itu bukan aturan bisnis.** Kekosongan itu **akibat perangkaian string**, bukan keputusan
siapa pun. Tidak ada satu pun rule, when, maupun dokumen yang menyatakan "petugas tanpa cabang
tidak boleh melihat apa pun". `03-CURRENT-ARCHITECTURE.md` §4.5 mencatat pola `{ASIS:}` sebagai
utang teknis yang justru harus dihapus — 538 kemunculan.

**Kenapa ini tidak melanggar `P-5`.** Yang `P-5` pertahankan adalah **hasil aturan bisnis**.
Menyalin akibat sampingan dari cacat teknis yang sedang kita hapus bukanlah kesetaraan; itu
membawa cacatnya sekalian. Selisih ini juga **bukan** butir baru pada daftar 13 perbaikan `D-49`,
karena ia tidak mengubah hasil aturan mana pun — ia mengubah apa yang terjadi ketika sebuah
prasyarat tidak terpenuhi.

**Harga yang dibayar, dan bagaimana ia ditagih.** Petugas yang cabangnya tidak terbaca melihat
berkas seluruh cabang. Itu pelonggaran batas data, dan karena itu ia **dinyatakan**, bukan
didiamkan:

| Tempat | Bentuk |
|---|---|
| Respons API | `batas_cabang` dan `cabang_terbaca` |
| Layar | pemberitahuan berlatar amber, `role="status"` |
| `-periksa` | `checkClaimReportBranch` melaporkannya ke operator |

Alternatifnya — meniru Pega dan menampilkan daftar kosong — menghasilkan layar yang **tampak
rusak** tanpa memberi tahu apa yang harus dibetulkan, dan itulah persis cacat yang sedang
diperbaiki sesi ini.

### 19.5 Keputusan: dua field respons, bukan satu

`batas_cabang` kosong punya **dua sebab yang berbeda artinya**:

| Sebab | `batas_cabang` | `cabang_terbaca` |
|---|---|---|
| Pengguna memilih kanwil sendiri | `""` | `true` |
| Cabang petugas tidak terbaca | `""` | `false` |

Yang pertama pilihan pengguna dan tidak perlu dikomentari; yang kedua keadaan yang harus
diberitahukan. Satu field tidak dapat membedakan keduanya, dan layar yang menebaknya akan
memberitahu pada saat yang salah.

### 19.6 Keputusan: `Create` menurunkan cabang dengan jalan yang sama dengan `List`

Terlihat sepele, dan justru karena itu ditulis sebagai keputusan.

Bila `Create` mengisi `BranchCode` berkas dari sumber yang berbeda dengan yang dipakai `List`
menyaring, berkas yang baru dibuat **langsung hilang dari daftar pembuatnya** — cacat yang
bentuknya persis sama dengan yang sedang diperbaiki, hanya lebih sulit dilihat karena hanya
menimpa satu baris. `TestNewReportLandsInTheBranchTheListFiltersBy` menguncinya.

### 19.7 Utang teknis yang ditambahkan dengan sadar

1. **Modul ini memakai DB link.** `branch_of_login` menembus `@asmd.sinarmas.co.id` ke
   `HRDASM.V_HRD_MST` dan `LST_USER_ASURANSI` — tepat yang `D-25` tetapkan untuk diganti API
   (`R-03`). Dibiarkan demikian karena API-nya belum ada, dan ditaruh di balik seam supaya
   penggantiannya kelak menyentuh satu adapter saja.
2. **Satu kueri tambahan per permintaan daftar.** Penerjemahan cabang berjalan pada setiap
   pemanggilan `List`. Belum di-cache; bila terbukti mahal, tempat cache-nya adalah adapter, dan
   `14-NFR` §3.3 sudah menetapkan cache in-process untuk data yang jarang berubah.
3. **Padding `CHAR` dipangkas di adapter.** `POOLDATA.BRANCH.ID` kemungkinan bertipe `CHAR`
   sehingga nilainya dikembalikan dengan spasi di belakang; adapter memangkasnya. DDL-nya belum
   pernah dilihat (`R-08`), jadi ini penjagaan, bukan pengetahuan.

### 19.8 Pertanyaan terbuka yang bertambah

| # | Pertanyaan | Kenapa ia penting |
|---|---|---|
| 8 | **Apakah petugas yang cabangnya tidak terbaca memang boleh melihat seluruh cabang?** Keputusan §19.4 diambil karena alternatifnya lebih buruk, bukan karena ada yang menyatakannya benar. Bila jawabannya "tidak", yang berubah bukan kodenya melainkan kebijakannya — dan layar harus menolak, bukan mengosongkan diam-diam |
| 9 | **Siapa pemilik data pemetaan login menjadi cabang di sistem baru?** Hari ini ia dibaca lintas DB link dari HRD. Setelah `D-25` dijalankan, ia menjadi API milik tim lain — dan ketersediaannya menjadi prasyarat batas data layar ini |

---

## 20. Cabang sebagai batas data yang mengikat (2026-09-22, sesi kesebelas — koreksi)

**Menyupersede §19.4.** Bagian itu tidak disunting: ia rekaman keputusan yang benar-benar berlaku
beberapa jam sebelumnya beserta alasannya, dan menghapusnya menghilangkan jejak bahwa pilihan itu
pernah diambil. Yang berlaku adalah bagian ini.

### 20.1 Keputusan Work Owner

> "petugas yang cabangnya tidak terbaca tidak boleh melihat seluruh cabang"

Dengan itu **cabang bukan kenyamanan penyaring, melainkan batas data**. Konsekuensi logisnya
langsung: batas data yang tidak dapat ditentukan berarti permintaannya **tidak dapat dilayani** —
bukan berarti batasnya gugur.

### 20.2 Keputusan: menolak, bukan mengembalikan daftar kosong

Sistem lama menghasilkan daftar kosong dalam keadaan ini (`branch where ID=''`). Meniru itu
**ditolak**, dan alasannya adalah cacat yang sedang diperbaiki sesi ini:

| Bentuk penolakan | Yang terjadi di kursi petugas |
|---|---|
| Daftar kosong | tidak terbedakan dari "tidak ada pekerjaan hari ini" — dan justru ketidakterbedaan itu yang membuat penyaring cabang yang salah bertahan tanpa ada yang melaporkannya |
| **Penolakan yang menyebut sebab** | akses tertutup sama rapatnya, **dan** petugas tahu apa yang harus dibetulkan |

Keduanya sama-sama memenuhi keputusan Work Owner. Yang kedua dipilih karena ia tidak menukar satu
kegagalan senyap dengan kegagalan senyap yang lain.

### 20.3 Keputusan: dua sebab, dua jawaban HTTP

Pembedaan tiga keluaran `BranchResolver.Resolve` (§19.3) terbayar di sini — ia menjadi dua jawaban
yang berbeda, bukan satu:

| Keadaan domain | HTTP | Kode | Dibereskan di |
|---|---|---|---|
| `ErrBranchUnknown` — login belum terdaftar di HRD | **403** | `cabang_tidak_dikenali` | data pegawai; menimpa **satu** orang |
| `ErrBranchUnreadable` — `POOLDATA.BRANCH` atau DB link mati | **503** | `sumber_cabang_tidak_terbaca` | infrastruktur; menimpa **seluruh** petugas |

**Kenapa 403 dan bukan 404 atau 422.** Pemanggilnya sudah masuk, alamatnya benar, dan tidak ada
satu pun isian yang dapat diperbaiki. Yang tidak dapat ditetapkan adalah batas datanya —
`10-API-STRATEGY.md` §5: *"sudah login tapi tidak berwenang"*.

**Kenapa 503 dan bukan 403.** Yang gagal bukan kewenangan pemanggil melainkan sumber datanya, dan
ia menimpa semua orang sekaligus. `503` juga menyatakan keadaannya **sementara**, sehingga mencoba
lagi memang masuk akal. Sebab aslinya dibungkus dengan `%w` supaya log menyebut DB link atau tabel
mana yang gagal, sementara peramban hanya menerima pesan umum.

### 20.4 Keputusan: `cabang_terbaca` dihapus dari kontrak

Field itu ditambahkan beberapa jam sebelumnya untuk menyatakan "batasnya sedang tidak berlaku".
Sejak keadaan itu ditolak, ia **tidak dapat lagi bernilai `false`** — dan field yang hanya pernah
bernilai satu macam adalah kebohongan yang menunggu giliran: pembaca berikutnya akan menulis cabang
penanganan untuk keadaan yang tidak pernah terjadi.

`batas_cabang` **tetap ada**, dan kosongnya kini punya satu arti saja: pengguna memilih kanwil.
Alasannya bertahan: petugas yang tidak tahu daftarnya sedang disaring akan menyimpulkan tidak ada
pekerjaan, padahal yang benar adalah tidak ada pekerjaan **di cabangnya**.

### 20.5 Keputusan: penegakan di satu tempat, menutup tiga permukaan

`requireBranch` dipanggil `List` dan `Create`; **Ekspor menempuh `List`**. Batas yang ditegakkan
pada daftar tetapi tidak pada ekspor bukan batas sama sekali — ia hanya menyulitkan orang yang
patuh. Satu uji mengunci bahwa ekspor memang menempuh jalur yang sama.

### 20.6 Perluasan yang diambil sendiri: pembuatan berkas ikut ditolak

**Keputusan Work Owner berbunyi tentang "melihat", bukan "membuat".** Perluasannya ditulis terbuka
di sini, di komentar kode, dan di uji — supaya dapat dikoreksi.

Alasannya: sejak cabang menjadi batas yang mengikat, berkas yang lahir tanpa cabang **tidak akan
pernah terlihat siapa pun** — pembuatnya tidak dapat membuka daftarnya, dan petugas cabang mana pun
tersaring darinya. Membolehkannya berarti menerbitkan baris yang dijamin tidak dapat dikerjakan,
dan itu lebih buruk daripada menolak dengan sebab yang jelas.

Ini juga mencabut alasan yang ditulis §19 dan sebelumnya: dulu pembuatan dibiarkan karena `P-5` dan
karena `CreateNewCaseRCV` tidak memeriksa apa pun. Yang berubah bukan pembacaan atas Pega,
melainkan **akibat** dari perilaku itu di sistem yang batas cabangnya kini mengikat.

### 20.7 Akibat pada data contoh — dan apa yang tersingkap karenanya

Sebelas uji memakai "cabang tidak terbaca" sebagai cara melihat seluruh berkas contoh. Jalan itu
kini ditutup, dan penggantinya adalah pemilihan **kanwil** — satu-satunya cara sah melihat lebih
dari satu cabang di layar sungguhan.

Itu menyingkap sesuatu yang selama ini tersembunyi di balik pandangan tanpa batas: **data contoh
menaruh saksi aturan kelompok bisnis khusus di kanwil 02 dan 03**, sehingga aturan itu tidak dapat
dicoba dari kursi mana pun. Satu berkas (`RCV-0006`) dipindahkan ke cabang 1002.

Ini bukan menyesuaikan data agar uji lulus. `SampleList` sejak awal berjanji "setiap penyaring
punya baris yang cocok maupun yang tidak"; sejak tidak ada lagi pandangan tanpa batas, janji itu
hanya bermakna **di dalam lingkup yang dapat dilihat** — dan baru sekarang ia benar-benar diuji.

### 20.8 Pertanyaan terbuka yang menjadi mendesak

| # | Pertanyaan | Kenapa ia kini mendesak |
|---|---|---|
| 10 | **Bolehkah petugas memilih kanwil mana pun?** Dropdown Kanwil menggantikan batas cabang, sehingga petugas Jakarta dapat melihat Surabaya. Bila cabang adalah batas data yang mengikat, kanwil yang bebas dipilih adalah **pintu yang sama, hanya lebih lebar**. Aturan yang menentukannya di sistem lama ada di antara 137 When rule yang hilang (`R-16`) — jadi ini keputusan, bukan temuan |
| 11 | **Bolehkah sebuah berkas dibuka lewat nomornya tanpa memeriksa cabang?** `Get` dan `Save` tidak menyaring cabang; yang menjaganya hanyalah nomor berkas tidak diketahui dari luar daftar. Itu bukan penjagaan. Keadaannya **mendahului** sesi ini dan menimpa seluruh petugas, bukan hanya yang cabangnya tidak terbaca |
| 9 (diperkuat) | **Siapa pemilik pemetaan login-ke-cabang setelah `D-25`?** Kini bukan lagi soal kenyamanan: bila sumbernya mati, **seluruh layar tertutup** bagi semua orang — bukan sekadar melebar |

---

## 39. Gerbang tipe dipisahkan dari perintah build (2026-09-23)

Prosesnya ada di [`catatan-pengembangan.md`](catatan-pengembangan.md) §37; berkas ini
merekam **keputusannya**.

### 39.1 Keadaan yang memaksanya

`npm run build` menjalankan `tsc --noEmit` lebih dulu dan gagal bila ada satu galat tipe di
mana pun di repo. Repo ini punya **120 galat**, seluruhnya di tujuh modul Master Data milik
tim lain.

Akibatnya bukan "peringatan yang diabaikan" melainkan **bundel yang tidak pernah terbentuk**:
`backend/spa/dist` beku sejak 22 Sep 14:41, dan setiap layar yang dibangun sesudahnya —
Inbox Outstanding, Inbox Manager Receive / PUCL — tidak pernah sampai ke `:8080`.

Yang membuatnya sulit terlihat: **tidak ada satu pun pesan galat di peramban**. Aplikasinya
berjalan normal, hanya menyajikan versi lama.

### 39.2 Keputusan

```
"build":         "vite build && npm run mark-dist",
"build:checked": "npm run typecheck && npm run build",
```

### 39.3 Kenapa bukan salah satu dari ketiga pilihan lain

| Pilihan | Kenapa ditolak |
|---|---|
| Perbaiki ke-120 galatnya | Seluruhnya di modul **Master Data**, yang Isolasi Protektif larang diubah. 100 di antaranya menuntut **mengarang ±90 bentuk tipe** milik modul orang lain — dilarang "No Shortcuts" |
| `tsc --noEmit \|\| true` di dalam `build` | Gerbangnya jadi hijau palsu. Galat tetap ada, tetapi tidak seorang pun melihatnya lagi |
| Longgarkan `tsconfig` | Menyembunyikan cacat nyata — 18 di antaranya parameter tanpa tipe, yang justru kelas cacat yang `08-TECHNICAL-STRATEGY.md` §5 larang |

### 39.4 Apa yang hilang, dan apa yang tidak

**Tidak hilang:** pemeriksaan tipe itu sendiri. Ia sudah punya perintah sendiri sejak awal
(`npm run typecheck`), dan kini juga dijalankan `build:checked`.

**Hilang:** paksaan bahwa setiap bundel lokal bertipe bersih. Itu memang yang dilepas dengan
sengaja — memaksanya hari ini berarti tidak ada seorang pun di tim yang dapat membangun
apa pun.

**Belum ada CI di repo ini** — `.github/`, `.gitlab-ci.yml`, `Jenkinsfile`, dan
`azure-pipelines.yml` seluruhnya tidak ada. Jadi gerbang yang dilepas itu hari ini tidak
menjaga apa pun selain memblokir pekerjaan sendiri. Begitu CI dipasang Tim GitLab (`D-33`),
yang dipanggil adalah `build:checked`, bukan `build`.

### 39.5 Utang yang dinyatakan, bukan disembunyikan

Ketujuh modul Master Data di `catatan-pengembangan.md` §37.2 tidak akan lulus
`build:checked` sampai tipenya dilengkapi. Selama itu, gerbang tipe **tidak dapat**
dikembalikan ke jalur build utama tanpa memblokir seluruh tim lagi.

Ini utang dengan pemilik yang jelas per modul, bukan utang tanpa alamat — dan itulah syarat
yang `D-36` tetapkan supaya sebuah penghalang punya peluang hilang.
---

## 18. Penjenjangan Komite dan Master Ambang (2026-09-17, sesi keenam)

### 18.1 Empat keputusan Work Owner pada sesi ini

| # | Pertanyaan | Keputusan |
|---|---|---|
| 1 | `B-7` bergantung pada `B-5` dan `B-6` yang belum ada. Apa cakupannya? | **Master ambang + mesin penjenjangan.** Layar keputusan komite menyusul setelah prasyaratnya ada |
| 2 | Tabel warisan mana yang boleh ditulis? | **Baca saja.** `POOLDATA.EMAILKOMITE` dan `T_CLAIM_KOMITE_LIST` tetap dimiliki Pega |
| 3 | `AutoAcceptKomite` dibawa? | **Tidak dibawa** |
| 4 | Go dan Node tidak terpasang di mesin ini | **Tulis kode, uji menyusul** |

### 18.2 Kepemilikan tabel: kenapa modul ini berbeda dari Master Status Klaim

Modul sebelumnya **menulis** ke tabel warisan, dan itu sah karena kepemilikannya benar-benar
berpindah: layar Master Status Klaim adalah satu-satunya penulis `M_STS_CLAIM` di sistem
lama, sehingga memindahkan layarnya memindahkan tabelnya secara utuh.

Modul ini **tidak**. `POOLDATA.EMAILKOMITE` masih ditulis Pega dan dibaca **17 kueri** di
sana — `EmailKomiteBerjenjang_sql` beserta varian PA, Travel, Bonding, Simasnet, Adjuster,
dan Salvage. Tidak ada satu layar pun yang dapat dipindahkan untuk memindahkan
kepemilikannya, karena tabel itu tidak punya layar pengelola di sistem lama sama sekali.

`P-1` karena itu ditegakkan dengan cara yang paling keras yang tersedia: **berkas `.sql`
modul ini tidak memuat satu pun `INSERT`, `UPDATE`, `DELETE`, `MERGE`, `TRUNCATE`, `DROP`,
atau `ALTER`**, dan `TestTidakAdaKueriYangMenulis` memindainya. Aturan yang hanya ada di
dokumen akan dilanggar oleh kode berikutnya; aturan yang dijaga uji tidak.

Akibatnya di layar: tidak ada tombol Tambah maupun Ubah, dan **ketiadaannya dijelaskan di
layar itu sendiri**. Pengguna yang terbiasa dengan layar master lain akan mencarinya, dan
tanpa keterangan ia akan menyimpulkan layarnya belum selesai.

### 18.3 Kenapa penyaringan dikerjakan di Go, bukan di `WHERE`

Seluruh 30 baris master dibaca tanpa klausa penyaring, lalu disaring di lapisan domain.

Biayanya nol — isinya 30 baris. Yang diperoleh: aturan penjenjangan (kumulatif, pemilihan
pita, urutan) hidup di **satu tempat** sebagai fungsi murni yang dapat diuji tanpa basis
data, dan perilakunya dijamin sama persis antara Oracle dan penyimpanan di memori.

Menaruh penyaringan di SQL akan memecah aturan itu menjadi dua salinan yang dapat berbeda
pendapat — persis pola yang membuat sistem lama menyebarkan satu aturan bisnis ke activity,
SQL, dan stored procedure sekaligus, sehingga satu perubahan harus dicari di tiga tempat
dan sering hanya ditemukan di dua.

### 18.4 Tipe nilai uang: kenapa dibangun sendiri

`I-12` melarang `float` untuk nilai uang, dan modul ini adalah yang pertama benar-benar
memerlukannya: ia membandingkan Rp 50.000.001 melawan Rp 50.000.000, dan selisih satu
rupiah menentukan satu jenjang persetujuan ikut atau tidak.

`internal/platform/uang` menyimpannya sebagai `int64` satuan terkecil. Tidak ada dependensi
pihak ketiga — untuk kebutuhan yang seluruhnya penjumlahan dan perbandingan pada dua
desimal, bilangan bulat sudah cukup.

**Satu jebakan yang ditangani eksplisit:** memindai NUMBER Oracle ke `*string` tampak aman
tetapi tidak. Bila driver menyerahkan `float64`, `database/sql` memformatnya dengan
`strconv.FormatFloat(v, 'g', -1, 64)`, dan `'g'` menghasilkan notasi ilmiah untuk angka
besar — Rp 100.000.000 menjadi `"1e+08"`. Karena itu setiap bentuk driver ditangani sendiri,
dan `float64` berdesimal **ditolak** alih-alih dibulatkan diam-diam.

### 18.5 Urutan penyetuju dibuat pasti — perbedaan yang disengaja

Kueri sistem lama mengurutkan dengan `ORDER BY DEGREE` saja. Pada master yang berlaku,
Non-MBU pita 1 memiliki **dua baris ber-DEGREE 1** (ID 7 dan ID 1), sehingga urutan keduanya
diserahkan kepada basis data dan dapat berubah antar eksekusi.

Modul ini memecahkan seri secara pasti: ambang bawah lebih kecil lebih dulu — yang secara
bisnis memang masuk akal, karena jenjang berambang lebih rendah menyetujui lebih awal —
lalu ID sebagai pemecah terakhir.

Ini **tidak mengubah siapa** yang menyetujui, hanya urutannya saat seri. Dan setiap kali
terjadi, penanda `UrutanTidakPasti` menyala sampai ke layar, supaya perbedaan urutan
terhadap Pega pada kasus seri **tidak terbaca sebagai cacat** saat uji kesetaraan `S-8`
dijalankan kelak.

### 18.6 `LIMIT_TOP` dipakai — untuk satu hal saja

`D-47` menetapkan `LIMIT_TOP` bukan penyaring pemilih baris; memakainya untuk memilih akan
mengembalikan tepat satu baris dan menghapus penjenjangan seluruhnya. Perannya adalah
**validasi integritas master**.

Di sistem lama kolom itu tersimpan tetapi **tidak pernah dipakai satu kueri pun**.
`PeriksaIntegritas` memberinya pekerjaan: menemukan rentang yang tumpang tindih, berlubang,
atau terbalik; melaporkan jenjang ganda; dan menyebutkan sampai nilai berapa tangga tiap
lini masih membedakan jenjang.

**Pengelompokannya berbeda antar lini, dan itu wajib.** Untuk Non-MBU, per pita. Untuk lini
lain, per lini saja — bila tangga PA dikelompokkan per `TYPE_KOMITE`, ia terbelah menjadi
dua potongan yang tampak berlubang parah, padahal di PA kolom itu membedakan PA reguler
dari PA TKI (`D-70`).

### 18.7 Kontrak API modul ini

    GET /api/master/ambang-komite              200  tangga + daftar lini + kebijakan pita
    GET /api/master/ambang-komite/integritas   200  temuan, termasuk saat ada cacat
    GET /api/komite/penjenjangan?nilai=&lini=  200  penyetuju berurutan
                                               400  nilai bukan angka kanonik
                                               404  lini tidak ada di master
                                               422  isian melanggar aturan

Tiga hal yang disengaja:

**Seluruhnya GET.** Tidak satu pun mengubah apa pun. Akibat praktisnya: hasil perhitungan
dapat ditautkan, sehingga seseorang yang menemukan angka meragukan dapat mengirimkan
tautannya apa adanya kepada Work Owner.

**Integritas menjawab 200 walau ada cacat.** Cacat pada master adalah **temuan yang
dilaporkan endpoint ini**, bukan kegagalan permintaan. Menjawabnya dengan galat akan
membuat layar menampilkan halaman gagal justru pada saat ia paling perlu menampilkan isinya.

**"Lini tidak ada" (404) dibedakan dari "tidak ada jenjang yang cocok" (200 + penanda).**
Yang pertama salah ketik atau lini baru yang belum diisi; yang kedua keadaan data yang harus
dilihat Work Owner. Menjawab keduanya sama akan menyembunyikan yang kedua.

**Nilai uang dikirim sebagai teks desimal kanonik**, bukan angka JSON — angka JSON adalah
floating point ganda di peramban. Pemisah ribuan diurai **di layar**, karena artinya berbeda
antar bahasa dan penafsirannya harus terjadi di tempat yang tahu bahasanya.

### 18.8 Alamat surel tidak dibaca sama sekali

Master ambang memuat kolom `EMAIL` dan `CC`. Keduanya **tidak masuk ke dalam `SELECT`** —
bukan dibaca lalu dibuang di lapisan berikutnya.

Dua alasan yang saling menguatkan: modul ini menghitung **siapa yang menyetujui**, bukan ke
mana pemberitahuan dikirim (itu `S-3`); dan `D-67` menetapkan alamat pribadi pada master
lama — sekurang-kurangnya enam akun Gmail di jalur produksi — tidak dibawa ke sistem baru
sama sekali.

Tidak membacanya sejak kueri membuat alamat itu tidak pernah sampai ke peramban, alih-alih
mengandalkan setiap lapisan sesudahnya ingat membuangnya. Dijaga dua uji: satu memindai
kuerinya, satu memindai badan respons.

### 18.9 Batas pita: satu-satunya nilai bisnis yang masih di dalam kode

`BatasPitaNonMBUBawaan = 100_000_000` ada sebagai konstanta, dan itu menyimpang dari `D-15`.

Penyimpangannya dibatasi: konstanta itu **tidak dibaca mesin penjenjangan**. Ia hanya nilai
bawaan yang membentuk `Kebijakan`, dan `Kebijakan` **dipasok dari luar** lewat
`usecase.Opsi`. Memindahkannya menjadi master `F-4` kelak tidak menyentuh satu baris pun
aturan di `jenjang.go`. Sifat itu diuji di `TestKebijakanPitaDapatDiganti`.

Kenapa belum menjadi master: tidak ada tabel yang memuatnya. Di sistem lama pita dipilih
dengan **membandingkan nama orang** — `Activity/SetEmailKomite-Act.xml` step 10, 12, dan 14
mencocokkan `UserTeknis` dengan tiga nama tertentu lalu memaksa nilai pembandingnya melewati
ambang. `D-52` mencabut cara itu; yang menggantikannya adalah pita yang diturunkan dari
nilai klaim, dan angkanya belum punya rumah.

Ia juga ditampilkan di layar, bukan disembunyikan: angka yang menentukan uang tidak boleh
hanya hidup di dalam kode tanpa pernah terlihat siapa pun.

### 18.10 Jejak audit tetap tidak dibangun

Sama dengan modul sebelumnya, dan alasannya kini lebih kuat: modul ini **tidak menulis apa
pun**, sehingga tidak ada perubahan bernilai bisnis yang perlu dicatat.

Yang akan membutuhkannya adalah `TKT-B07-002` — pencatatan keputusan komite — dan di sana
jejak audit bukan pelengkap melainkan **satu-satunya kontrol pengimbang**, karena `D-59`
menghapus pemisahan tugas.

### 18.11 Pertanyaan terbuka yang ditinggalkan sesi ini

1. **Beban bila `AutoAcceptKomite` dihapus** belum dihitung; `TKT-B07-003` menuntut angkanya
   sebelum rilis. Kuerinya ada di catatan pengembangan §17.10.
2. **Arti baris `DEGREE=0`** — datanya menunjukkan ia penerima pemberitahuan registrasi
   (`STS_ADJ` kosong, `STS_REG` menyala), dan penyaring yang ada sudah mengeluarkannya.
   Pengamatan itu dilaporkan, belum ditegaskan Work Owner.
3. **`dbms_random.value` pada dua kueri Simasnet** — tidak dibawa, dan larangannya dijaga
   uji. Bila ternyata disengaja, aturan urutan di modul ini harus ditinjau ulang.
4. **Perilaku `STS_ABS`** — kolomnya dibaca dan dibawa sampai ke layar, tetapi **tidak
   menyaring siapa pun**. Tidak ada rule di export yang memperlihatkan apa yang terjadi bila
   ia menyala, dan menebaknya berarti mengarang aturan yang menentukan siapa menyetujui uang.
5. **Pemeriksaan peran** pada rute modul ini belum ada — `TKT-F3-005`, sama dengan seluruh
   rute lain hari ini. Yang perlu disadari khusus modul ini: isi layarnya memperlihatkan
   siapa yang berwenang menyetujui uang.

---

## 19. Empat jawaban Work Owner yang mengoreksi sesi sebelumnya (2026-09-18)

### 19.1 Dua keputusan kemarin dibatalkan

| Keputusan 2026-09-17 | Keputusan 2026-09-18 |
|---|---|
| `AutoAcceptKomite` **tidak dibawa** | **dikonversi**, bukan dihapus |
| Pengacakan penyetuju **tidak dibawa** | **dibawa** — ia disengaja, dan membawa kontrol yang berharga |

Keduanya dicatat sebagai pembatalan, bukan disunting menjadi seolah-olah tidak pernah ada.

### 19.2 Dua mode penjenjangan, ditentukan per portal

Temuan terbesar sesi ini: penjenjangan komite **tidak punya satu aturan**, melainkan dua,
dan yang berlaku ditentukan **entitas** — bukan lini bisnis.

| Mode | Berlaku pada | Perilaku |
|---|---|---|
| `kumulatif` | seluruh entitas selain Simasnet | setiap jenjang yang ambang bawahnya terlampaui ikut menyetujui |
| `satu-penyetuju` | entitas **Simasnet** | dipilih **tepat satu**, diacak di antara jenjang terendah, **penginput dikecualikan** |

Pemilihnya di sistem lama (`Activity/SetListComiteeClaimPerObjAdj-Act.xml`):

```
bila TempGetApp.LSC_ID == "SIMASNET"  → SetEmailKomiteSimasnet
selain itu                            → SetEmailKomite
```

dan `LSC_ID` dibaca dari `POOLDATA.DB_LINK_PEGA` dengan **mencocokkan nama server** —
persis pola yang `D-75` ganti dengan portal. Karena itu `Mode` menjadi medan pada
`Kebijakan`, satu per portal, bukan pada `Ambang` maupun `Lini`.

### 19.3 Pengecualian penginput: kontrol pemisahan tugas yang nyata

Aturan Simasnet mengeluarkan **operator yang sedang menginput** dari daftar calon
penyetuju. Sumbernya `Activity/SetEmailKomiteSimasnet-Act.xml`, yang menyusun potongan SQL
`"AND OPERATOR_ID!='" + OperatorID.pyUserIdentifier + "'"`.

Ini penting melampaui modul komite: `D-59` menetapkan **tidak ada pemisahan tugas formal**
di sistem lama. Temuan ini tidak membatalkannya — cakupannya satu entitas — tetapi ia
membuktikan pernyataan itu tidak berlaku mutlak, dan bahwa mekanismenya pernah ada.

Perbandingan identitasnya dinormalkan lewat `KunciOperator` (huruf besar, spasi tepi
dibuang). Itu bukan kelonggaran: bila pencocokan gagal, penginput tetap menjadi calon dan
**dapat terpilih menyetujui pengajuannya sendiri** — dan kegagalannya tidak terlihat,
karena hasilnya tetap berupa nama yang masuk akal.

### 19.4 Pengacakan pindah dari SQL ke Go

Kueri lama mengacak di dalam SQL (`ORDER BY degree, dbms_random.value`). Di sini
pengacakannya pindah ke Go, di balik seam `komite.Pengacak`.

Yang dilarang `kueri_test.go` karena itu bukan perilakunya melainkan **tempatnya**.
Alasannya: diacak di dalam SQL membuat aturannya tidak dapat diuji sama sekali — hasil
yang berbeda tiap kali dijalankan tidak dapat dibandingkan dengan apa pun. Di balik seam,
pengujian memakai pemilih tetap sementara produksi memakai `platform/acak`.

Bawaannya **tetap**, bukan acak. Sesuatu yang diam-diam menjadi acak jauh lebih berbahaya
daripada sesuatu yang diam-diam menjadi tetap: yang pertama baru ketahuan saat dua orang
membandingkan hasil dan menemukannya berbeda.

**Akibatnya bagi gerbang 1:** pada mode ini, yang dapat dibandingkan dengan Pega bukan
SIAPA yang terpilih melainkan **apakah kumpulan calonnya sama**. Karena itu `Kandidat`
dikirim sampai ke layar, dan layarnya menyatakan terus terang bahwa hasil berbeda pada
nilai yang sama bukan cacat.

### 19.5 Batas pita berbeda per entitas — dan satu angka yang nyaris salah dipakai

Pertanyaan Work Owner "ini diambil dari mana" menghasilkan temuan yang tidak dicari.

Batas Rp 100.000.000 **ada di dalam rule**, bukan disimpulkan:

```
tempAdj.AcceptedNo := @If(tempAdj.ConvertAdjustmentValue > 100000000, 2, 1)
```

Tetapi empat baris di bawahnya, rule yang sama memuat kembarannya untuk entitas dolar:

```
tempAdj.AcceptedNo := @If(tempAdj.ConvertAdjustmentValue > 7000, 2, 1)
```

Jadi konstanta tunggal yang ditulis sesi sebelumnya **benar untuk portal rupiah dan salah
untuk portal SMI**, dengan selisih sekitar 14.000 kali lipat. Rasio 100.000.000 ÷ 7.000 ≈
14.285 adalah **kurs yang dibekukan ke dalam kode**, bukan kurs dari master mata uang.

`BatasPitaNonMBUSMI` dicantumkan bukan supaya dipakai apa adanya, melainkan supaya
perbedaannya terlihat. Angkanya wajib dikonfirmasi ulang sebelum portal SMI dilayani.

### 19.6 `DEGREE=0` disaring eksplisit meski hari ini tidak mengubah apa pun

Work Owner menegaskan baris ber-`DEGREE` nol tidak dipakai. Penyaringnya ditulis meski
tidak mengubah satu hasil pun pada master yang berlaku — baris seperti itu sudah tersaring
lebih dulu oleh `STS_ADJ`.

Alasannya: tanpa penyaring itu, aturannya hanya **berlaku secara kebetulan**. Satu baris
baru ber-DEGREE 0 dengan `STS_ADJ` menyala akan diam-diam ikut menyetujui uang.

### 19.7 `AutoAcceptKomite`: yang ditiru dan yang diperbaiki

| Hal | Perlakuan | Alasan |
|---|---|---|
| Syarat lama menganggur | **ditiru** | inti aturannya; rule lama tidak punya syarat lain |
| Pelaku `SISTEM` | **diperbaiki** | sistem lama hanya menitipkan jejaknya pada kalimat di kolom catatan, sehingga satu-satunya cara mengetahui sebuah persetujuan itu otomatis adalah mencocokkan teks |
| Batas nilai dan jenjang | **ditambahkan, mati secara bawaan** | menjaga kesetaraan dengan Pega, sekaligus menyediakan tempat bagi jawaban Work Owner |
| Fitur secara keseluruhan | **mati secara bawaan** | job ini melewati seluruh kontrol otorisasi (`D-59`); tidak boleh menyala karena kelalaian menyetel |

**Ambangnya parameter, bukan konstanta**, karena `> 2` pada rule lama dapat berarti 48 jam
atau 72 jam dan export tidak menyelesaikannya. Yang dipakai adalah bacaan yang lebih lambat
menyetujui — bila keliru, akibatnya klaim menunggu sehari lebih lama, bukan uang yang
telanjur disetujui sendiri oleh sistem.

Penulisan persetujuan dan penjadwalnya **belum dibangun**: keduanya menuntut jalur
keputusan komite dan mekanisme penjadwal yang belum ada, dan menulis ke
`T_CLAIM_KOMITE_LIST` melanggar keputusan "baca saja" yang masih berlaku.

### 19.8 Pertanyaan terbuka yang bertambah

1. `> 2` pada `AutoAcceptKomite` — 48 jam atau 72 jam. Hanya Pega staging yang dapat
   memastikan.
2. Nilai dan jenjang mana yang boleh disetujui otomatis.
3. Batas pita portal SMI — USD 7.000 berasal dari kurs beku yang sudah tidak berlaku.
4. Master ambang portal Simasnet belum pernah dilihat; mode satu-penyetuju belum teruji
   terhadap data sungguhan.
5. Apakah pengecualian penginput seharusnya berlaku di entitas lain juga — hari ini ia
   hanya ada pada satu jalur, dan meluaskannya adalah **perubahan aturan**, bukan
   perbaikan.

---

## 20. Empat jawaban penutup, dan satu akibat yang harus diketahui sebelum rilis (2026-09-18)

### 20.1 Jawaban

| # | Pertanyaan | Jawaban | Akibat pada kode |
|---|---|---|---|
| 1 | `> 2` itu 48 atau 72 jam? | **72 jam** | Ambiguitas tertutup; bawaan 3 hari dipastikan benar |
| 2 | Nilai dan jenjang mana yang boleh auto-accept? | **Tidak dibatasi** — selama `KomiteCount <= KomiteLoop` | Dua medan konfigurasi **dihapus**, syarat jenjang ditambahkan |
| 3 | Batas pita portal SMI | *(belum dijawab)* | tetap terbuka |
| 4 | Pengecualian penginput berlaku di entitas lain? | **Ya** | Pengecualian menjadi **berlaku di semua mode** |

### 20.2 Jawaban 2 menutup pertanyaan, dan karena itu dua knob dihapus

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

### 20.3 Jawaban 4 adalah perubahan perilaku, dan akibatnya sudah dihitung

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

### 20.4 Yang masih terbuka

1. **Batas pita portal SMI** — USD 7.000 berasal dari kurs beku 14.285 yang sudah tidak
   berlaku. Belum dijawab.
2. **Master ambang portal Simasnet** belum pernah dilihat; mode satu-penyetuju masih diuji
   dengan data buatan.
3. **Seberapa sering anggota komite menginput klaim sendiri** — menentukan apakah ketujuh
   keadaan di §20.3 adalah risiko nyata atau kemungkinan teoretis. Pertanyaan untuk
   pengguna bisnis, bukan untuk kode.
4. **Menyambungkan kebijakan ke portal aktif** — `TKT-F6-002`.

---

## 21. Penamaan modul Komite mengikuti `D-80` dan `D-81` (2026-09-19)

### Keputusan

Seluruh nama di dalam kode modul Komite — folder, berkas, tipe, fungsi, method, field,
parameter, dan variabel lokal — memakai **bahasa Inggris**, kecuali **nama folder modulnya**
(`internal/komite`, `src/modules/ambang-komite`) yang tetap berbahasa Indonesia.

### Kenapa demikian, bukan sebaliknya

Ini bukan pilihan yang saya ambil sendiri: `CLAUDE.md` §7.4.1 sudah menetapkannya lewat `D-80`
dan `D-81`. Yang saya kerjakan adalah menerapkannya, setelah pass sebelumnya (§22 catatan
pengembangan) memakai aturan lama yang menyuruh istilah domain tetap Indonesia.

Dua alasan `D-80` yang langsung terasa di modul ini:

1. **Pustaka standar Go dan React seluruhnya berbahasa Inggris.** Sebelum penggantian, satu
   baris dapat berpindah bahasa dua kali — `sortDeterministic(cocok)` berdampingan dengan
   `strings.TrimSpace`. Sesudahnya tidak.
2. **Bahasa Indonesia tidak mengenal infleksi.** `Jenjang` dipakai untuk tiga hal berbeda di
   modul ini — nomor DEGREE, jumlah penyetuju, dan tangga itu sendiri — dan ketiganya sempat
   tertukar. Padanan Inggrisnya memisahkan mereka sendiri: `Tier`, `TierCount`, dan tangga yang
   diwakili `[]Threshold`.

### Yang menahan penerapannya sampai tuntas

**Variabel lokal di berkas uji tetap Indonesia.** Alasannya bukan kelalaian melainkan dua hal
yang diuji dan gagal: kata-kata itu hidup berdampingan dengan prosa Indonesia di komentar **dan
di pesan assertion**, sehingga penggantian otomatis memutus konsistensi deklarasi-pemakaian; dan
tanpa kompiler, menulis ulang 1.830 baris uji menukar cacat kosmetik dengan risiko cacat nyata.

Rinciannya beserta buktinya ada di `catatan-pengembangan.md` §23.4.

### Yang TIDAK terpengaruh, dan itu disengaja

Kontrak yang menyeberang keluar dari kode tidak ikut berubah: **nama field JSON**, **nilai kode
galat**, **jalur rute**, **nama kolom basis data**, dan **teks yang dilihat pengguna**. Mengubah
salah satunya bukan penggantian nama melainkan perubahan yang merusak klien.

Akibatnya satu baris dapat memuat dua bahasa, dan itu memang yang dikehendaki `D-80`:

```go
LowerBound money.Money `json:"batas_bawah"`
```

---

## 22. Inbox Komite (2026-09-20, sesi kesembilan)

### 22.1 Tiga keputusan Work Owner pada sesi ini

| # | Pertanyaan | Jawaban | Akibat pada kode |
|---|---|---|---|
| 1 | Inbox dibaca dari mana? | **Tabel warisan Pega, baca-saja** | `InboxRepo` membaca empat tabel warisan; nol pernyataan tulis |
| 2 | Seberapa jauh lingkupnya? | **Inbox + pencatatan keputusan** | `DecisionRepo` + migrasi `0004` |
| 3 | Kunci pencocokan pemilik inbox | **`User.Login` huruf besar** | `InboxCaller.Login`, dinormalkan `OperatorKey` |

### 22.2 Kepemilikan tabel: dua sumber, satu layar

Jawaban 1 dan 2 tidak dapat dipenuhi bersamaan tanpa membelah kepemilikannya. `P-1` menetapkan
satu tabel hanya boleh ditulis satu sistem, dan `POOLDATA.T_CLAIM_KOMITE_LIST` — tempat sistem
lama menyimpan `STATUSAPPROVE`, `NOTEKOMITE`, `NAMAKOMITE`, dan `KOMITEKE` — masih ditulis Pega
lewat `INSERTDATAKOMITELIST` serta dibaca puluhan kueri di sana.

`TKT-B07-002` sudah menetapkan jalan keluarnya pada bagian migrasi skema: *"Menambah tabel jejak
komite. Backward-compatible."*

| Seam | Tabel | Hak |
|---|---|---|
| `komite.InboxRepo` | `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`, `PC_ASSIGN_WORKLIST`, `POOLDATA.T_CLAIM_KOMITE_LIST`, `T_CLAIM_DATA_RESULTS_AI`, `T_CLAIM_PNC`, `PEGA_DASHBOARDPNC` | **SELECT saja** |
| `komite.DecisionRepo` | `POOLDATA.CPNC_KOMITE_KEPUTUSAN` (migrasi `0004`) | SELECT + INSERT |

Dua seam, bukan satu, meski adapter memori mengisi keduanya dengan objek yang sama. Pembelahannya
mengikuti kepemilikan tabel — dan itu yang membuat aturan "yang satu tidak boleh menulis" berhenti
bergantung pada kehati-hatian orang yang menyuntingnya berikutnya. Dijaga uji
`TestTidakAdaKueriYangMenulis` dan `TestKueriTulisHanyaMenyentuhTabelMilikSendiri`.

### 22.3 Tabel keputusan APPEND-ONLY, dan penegakannya di hak akses

Tidak ada UPDATE dan tidak ada DELETE — bukan hanya di kode, melainkan di **hak akses basis
data**: migrasi `0004` menginstruksikan DBA memberi `SELECT, INSERT` saja.

Alasannya lebih kuat di modul ini daripada di mana pun: `D-59` menghapus pemisahan tugas — satu
orang dapat membuat, menyetujui, dan membayarkan satu klaim bila perannya memiliki ketiga menu
itu. Tidak ada kontrol teknis yang mencegahnya, sehingga jejak inilah **satu-satunya kontrol
pengimbang yang tersisa** (`ADR-0012`, `ADR-0023`, `09-DATABASE-STRATEGY.md` §8).

Aturan yang hanya ada di dalam kode dapat dilanggar oleh kode berikutnya; aturan yang ada di hak
akses tidak.

### 22.4 Satu orang memutuskan sekali — dijaga dua lapis

| Lapis | Di mana | Kenapa perlu keduanya |
|---|---|---|
| Aturan | `Progress.DecidedBy` di usecase | memberi pesan yang dapat dibaca, bukan galat 500 |
| Constraint | `CPNC_KOMITE_KEPUTUSAN_UK (CASE_ID, ACTOR_LOGIN)` | pemeriksaan dan penyimpanan **bukan satu operasi atomik** — dua permintaan bersamaan dapat lolos keduanya |

Bentrok constraint diterjemahkan menjadi `komite.ErrDecisionClosed` lewat nama constraint-nya
(`sqlstore.UniqueKeyName`), bukan lewat nomor galat driver: nama itu milik kita dan tidak berubah
saat driver atau basis datanya berganti.

### 22.5 Kenapa "sekali per orang", bukan "sekali per jenjang"

Jumlah jenjang sebuah kasus **belum dapat dihitung** — lihat `catatan-pengembangan.md` §24.4.
Tanpa angka itu, "seluruh jenjang selesai" tidak dapat disimpulkan.

Yang masih dapat dijamin: seorang anggota komite memutuskan sekali. Itulah yang mengeluarkan kasus
dari kotak Outstanding **miliknya**, persis seperti menyelesaikan assignment mengeluarkannya dari
worklist di sistem lama — sementara jenjang lain tetap menunggu pemiliknya masing-masing.

Penugasan komite memang per ORANG, bukan per antrean bersama (`T-7`), sehingga semantik itu bukan
penyederhanaan melainkan bentuk yang benar.

### 22.6 Kontrak API modul ini

    GET  /api/komite/inbox?kotak=&cari=&dari=&sampai=&lewati=&batas=
                                                   200  satu halaman + ringkasan tiga kotak
                                                   422  tanggal cacat
    GET  /api/komite/inbox/{nomor}                 200  satu kasus + penjenjangan
                                                   404  tidak ada ATAU bukan milik pemanggil
    POST /api/komite/inbox/{nomor}/keputusan       200  kasus sesudah keputusan
                                                   400  badan tidak dapat dibaca
                                                   404  tidak ada ATAU bukan milik pemanggil
                                                   422  keputusan/catatan melanggar aturan
                                                   409  sudah diputuskan

Enam hal yang disengaja:

**404 yang sama untuk "tidak ada" dan "bukan milik Anda".** Membedakannya akan mengubah endpoint
ini menjadi alat untuk menebak nomor case: `404` berarti tidak ada, `403` berarti ada tetapi milik
orang lain — dan yang kedua membocorkan keberadaan pekerjaan beserta nilainya kepada siapa pun
yang punya sesi. Perbedaannya **tetap ada di dalam domain** (`ErrCaseNotFound` versus
`ErrNotAssigned`), sehingga log dapat menyebut sebab yang sebenarnya.

**409, bukan 422, untuk keputusan kedua.** Permintaannya sah dan isinya benar; yang berubah adalah
KEADAAN kasusnya. Layar menanganinya berbeda: ia memuat ulang, bukan menandai isian yang salah.

**Badan permintaan menolak field yang tidak dikenali.** Klien yang mengirim `jenjang` atau `pada`
harus tahu keduanya tidak dipakai — mengabaikannya diam-diam akan membuatnya mengira ia menentukan
sesuatu yang sebenarnya ditetapkan server.

**Tanggal cacat ditolak, paginasi cacat tidak.** `dari=20-09-2026` mengubah APA yang ditampilkan,
sehingga mengabaikannya akan menampilkan seluruh riwayat kepada seseorang yang mengira ia melihat
satu minggu. `lewati=abc` hanya mengubah dari mana halamannya dimulai, sehingga jatuh ke halaman
pertama adalah pemulihan yang benar.

**Respons memantulkan `kotak` dan `operator`.** Kotak yang tidak dikenali JATUH ke outstanding
alih-alih ditolak, dan tanpa pantulan itu layar tidak tahu permintaannya diperlakukan berbeda.
Operator dipantulkan supaya inbox kosong dapat dibedakan sebabnya — tidak ada pekerjaan, versus
identitas sesi tidak cocok dengan satu pun `OPERATOR_ID` warisan (`ADR-0024`).

**Nilai uang sebagai teks desimal kanonik**, bukan angka JSON — sama dengan modul ini sebelumnya
(§18.7). Angka JSON adalah floating point ganda di peramban, dan `I-12` menetapkan nilai uang
disimpan presisi penuh.

### 22.7 Aturan kotak hidup di DUA tempat, dan itu diakui

Definisi kanoniknya `komite.CommitteeCase.InBox`; klausa `CASE WHEN` pada `inbox.sql` menirunya.
Itu melanggar "satu aturan satu tempat", dan pelanggarannya disengaja.

Berbeda dari master ambang — 30 baris, dibaca utuh, disaring di Go (§18.3) — kasus komite tumbuh
bersama jumlah klaim. Menyaringnya di Go menuntut seluruh antrean komite dibaca ke memori sebelum
satu halaman ditampilkan.

Yang dikerjakan supaya kedua salinan tidak berbeda pendapat diam-diam: fungsi Go-lah yang
**mendefinisikan** aturannya, kuerinya menyebutnya dalam komentar, dan adapter memori memakainya
langsung — sehingga setiap uji yang berjalan tanpa basis data menguji definisi itu, bukan salinan
SQL-nya. Ditambah `TestPenyaringDaftarDanPenghitungSama`, yang membandingkan klausa penyaring
`inbox_list` dengan `inbox_count` kata per kata: bila keduanya berbeda, layar akan menampilkan
jumlah halaman yang tidak pernah ada isinya.

### 22.8 Operator kosong berarti NOL BARIS, dan dijawab tanpa menyentuh basis data

`InboxRepo.ListCases` mengembalikan halaman kosong sebelum kueri dikirim bila penyaring pemiliknya
kosong. Kueri yang penyaring pemiliknya kosong **tidak boleh pernah dikirim** — supaya satu salah
ketik pada klausa WHERE tidak dapat membuatnya mengembalikan seluruh antrean komite perusahaan.

Kegagalan yang aman pada sebuah inbox adalah menampilkan **terlalu sedikit**, bukan terlalu banyak.

### 22.9 Pertanyaan terbuka yang ditinggalkan sesi ini

1. **`PRSN_PSPLNSOR` dijumlahkan dua kali** pada rumus Nilai OR ASM. Diduga salin-tempel yang lupa
   diganti, ditiru apa adanya sesuai `P-5`, dan **belum dikonfirmasi**. Bila benar cacat, ia
   melebihkan angka yang dibaca komite saat menyetujui uang. Pemilik: **Work Owner** — perbaikannya
   menuntut penambahan pada 13 butir `D-49`.
2. **Zona waktu pada penyaring tanggal.** Batas rentang disusun sebagai tengah malam WIB, sementara
   `PXCREATEDATETIME` disimpan Pega dalam GMT lalu digeser tujuh jam secara manual di 118 titik
   (`R-12`). Bila sebagian data tersimpan sudah tergeser dan sebagian belum, batasnya meleset tujuh
   jam pada sebagian baris. Tidak dapat dipastikan dari export; menuntut pembandingan terhadap data
   nyata di staging (`D-53`).
3. **Keputusan tidak menutup case di Pega.** Selama masa paralel, orang yang sama dapat memutuskan
   kasus itu lagi di sana, dan alur Pega tidak berlanjut karena persetujuan kita. Keputusan mana
   yang sah adalah pertanyaan untuk **Work Owner**.
4. **`T_CLAIM_DATA_RESULTS_AI` berapa baris per kasus?** Kueri diagregasi `MAX` supaya tidak
   menggandakan baris inbox; bila tabelnya memang satu baris per kasus, agregasinya tidak mengubah
   apa pun. Pemilik: **DBA**.
5. **Koneksi portal.** Kueri masih memakai koneksi UTAMA, sama dengan master ambang (§18 dan
   `main.go`). Di sini akibatnya lebih berat: yang salah portal adalah **daftar pekerjaan** badan
   hukum lain. `TKT-F6-002`.
6. **Retensi jejak keputusan.** `D-62` menetapkan ia mengikuti retensi data klaim; **angkanya belum
   diserahkan**, sehingga tidak ada penghapusan berkala yang boleh dijadwalkan.

---

## 40. Penggabungan cabang Inbox Komite ke `master` (2026-09-23)

### Keputusan: menyatukan, bukan memilih sisi

Kesebelas berkas berkonflik diselesaikan dengan **menyatukan kedua sisi**. Tidak satu pun
diselesaikan dengan `--ours` maupun `--theirs`.

**Alasannya bukan kehati-hatian umum, melainkan sifat konfliknya.** Seluruh sebelas konflik
adalah *dua cabang menambah hal berbeda pada berkas terpusat yang sama* — daftar rute, peta
menu, kumpulan ikon, kumpulan tipe, rangkaian modul di `main.go`. Pada bentuk konflik seperti
ini, memilih satu sisi berarti **membuang modul yang sudah jadi**, bukan memilih versi yang
lebih baru.

### `lib/money.ts` — dua API dipertahankan, dan itu disengaja

Konflik add/add: kedua cabang membuat berkas bernama sama dengan isi berbeda.

| Sisi | Ekspor |
|---|---|
| `master` | `formatRupiah`, `parseRupiah` |
| cabang Komite | `formatMoney`, `parseMoney`, `remainder` |

Yang **tidak** dilakukan: menyatukannya menjadi satu API. Itu akan menyentuh seluruh modul
pemakai di kedua sisi sekaligus, pada saat penggabungan — persis ketika kesalahan paling sulit
ditelusuri karena bercampur dengan konflik lain.

Yang dilakukan: **kelima fungsi dipertahankan**, dan kepala berkas menjelaskan bahwa keduanya
mewakili dua hal berbeda. Penyatuannya, bila memang dikehendaki, adalah pekerjaan tersendiri
dengan gerbangnya sendiri.

### Penjaga pada uji contoh menu, bukan sekadar mengganti contohnya

Uji `Sidebar.test.tsx` gugur karena contoh butir "belum tersedia" yang dipakainya keburu
dibangun cabang Komite. Ini **kejadian ketiga** dengan sebab yang sama.

Mengganti contohnya saja akan mengulang siklusnya untuk keempat kali. Karena itu contohnya
diangkat menjadi konstanta `BELUM_DIBANGUN`, dan ditambahkan satu uji yang tugasnya **hanya
memeriksa apakah contohnya masih sahih**, dengan pesan gagal yang menyebut konstanta mana yang
harus diganti dan di mana daftar kandidatnya.

**Yang ini akui secara terbuka:** tidak ada kriteria pemilihan contoh yang kebal. Alasan yang
dipakai sebelumnya — *harness-nya tidak ada di export* — terbukti tidak cukup, karena layar
Outstanding dibangun dari kueri, activity, dan section tanpa harness-nya sama sekali. Jadi yang
dipasang bukan contoh yang lebih baik, melainkan **kegagalan yang menjelaskan dirinya sendiri**.

### 120 galat tipe dibiarkan, dengan bukti

Ke-120 galat `tsc` seluruhnya milik `master-sparepart`, `master-supplier`, dan
`master-status-progres` — modul yang tercakup **Isolasi Protektif**.

Sebelum dibiarkan, dibuktikan dulu bahwa merge tidak menyebabkannya: ketiga kelompok tipe itu
bernilai **nol kemunculan di ketiga sisi** — `master` sebelum merge, cabang Komite, dan hasil
resolusi. Jadi tipe-tipe itu memang tidak pernah ada, dan resolusi `api/types.ts` tidak
menghapus apa pun.

Tanpa pemeriksaan tiga sisi itu, "jumlahnya sama, berarti aman" hanyalah dugaan — jumlah yang
sama dapat saja berasal dari galat lama yang hilang dan galat baru yang muncul sama banyak.

### Penomoran bab dokumen TIDAK dirapikan

Penomoran bab di `catatan-pengembangan.md` dan `keputusan-implementasi.md` ganda di banyak
tempat (§17 empat kali, §18 tiga kali), akibat banyak cabang paralel yang masing-masing
menempel di akhir berkas.

Menomori ulang akan menyentuh ratusan baris dan memutus setiap rujukan silang yang menyebut
nomor bab — pekerjaan berisiko yang tidak diminta dan tidak berhubungan dengan keluhan yang
sedang ditangani. Bab baru karena itu memakai nomor yang belum terpakai, dan kondisinya dicatat
di sini supaya terlihat sebagai **pilihan sadar**, bukan kelalaian.

### Butir menu Komite tidak ditambah-tambahi

Master menu hanya memuat **satu** butir Komite: MENU_ID 52 "Inbox Komite". Layar Penjenjangan
Komite dan Ambang Komite punya rute tetapi tidak punya butir menu.

Menambahkan butir menu untuk keduanya akan **mengarang data master** yang tidak ada di Pega —
melanggar larangan dummy logic. Keduanya tetap dicapai lewat layar Inbox Komite atau URL
langsung, persis seperti sistem lama.

---

## 41. Modul Inbox Close Claim (2026-09-23, sesi kedua puluh satu)

Menu `MENU_ID 59` "Inbox Close Claim", pengganti harness `InboxCloseClaim_Harness`.
Prosesnya ada di [`catatan-pengembangan.md`](catatan-pengembangan.md) §39; berkas ini
merekam **keputusannya**.

### 41.1 Pertanyaan konfirmasi dan jawabannya

Delapan diajukan dalam dua ronde, seluruhnya sebelum satu baris kode ditulis.

**Ronde pertama** — lingkup dan sumber data:

| # | Pertanyaan | Jawaban Work Owner |
|---|---|---|
| 1 | Tabel mana yang dibaca | **`DATAPEGA.PC_ASM_FW_GCNMFW_WORK`**, sama dengan kueri lamanya |
| 2 | Lingkup tulis | **ReOpen dan Copy Klaim keduanya dibangun** |
| 3 | Kolom "Lama Waktu Klaim" | **Hitung umur dalam hari** |
| 4 | Kueri hitung yang kehilangan penyaring | **Samakan dengan kueri daftar** |

**Ronde kedua** — setelah dilaporkan bahwa aturan ReOpen/Copy tidak ada di export:

| # | Pertanyaan | Jawaban Work Owner |
|---|---|---|
| 5 | Sumber aturannya | **Work Owner menyebutkannya sekarang** |
| 6 | Kewenangan tulis terhadap `P-1` | **Go menulis lewat tabel permintaan miliknya sendiri** |
| 7 | Efek ReOpen | status kerja + status klaim + pencacah |
| 8 | Lingkup Copy Klaim | polis + objek + coverage, **tanpa nilai** |

### 41.2 Harness rujukan tidak ada — dan apa yang dipakai sebagai gantinya

`InboxCloseClaim_Harness` dirujuk butir menu tetapi **nol berkas** di antara 74 harness yang
diekspor. Ia salah satu dari sembilan yang sudah tercatat di
`frontend/src/app/menu/registry.ts:29` — `K-33` dan `R-16`.

**Keputusan: dibangun dari delapan artefak yang memang ada**, mengikuti preseden
`InboxOutstanding_Harness`. Yang menentukan modul ini tetap dapat dibangun dengan setia
adalah `Activity/GCNMGetManagerReopenCase_Act-Act.xml`: ia menyebut **keenam penyaringnya
satu per satu** beserta potongan SQL yang disisipkannya, sehingga panel penyaring yang
hilang (`FilterDashboardClaimclose`) tidak perlu ditebak sama sekali.

**Konsekuensi untuk gerbang 1:** daftar, penyaring, paginasi, dan ekspor dapat diuji setara
dengan Pega — kedua kuerinya ada. Kedua aksi tulisnya **tidak dapat**, dan itu dibahas di
§41.5.

### 41.3 Sumber data: tabel Pega, bukan tabel datar

`POOLDATA.T_CLAIMLIST_ADMIN` ditetapkan Work Owner 2026-09-21 sebagai pengganti
`DATAPEGA.PC_ASM_FW_GCNMFW_WORK`, dan modul Inbox Outstanding sudah memakainya. Layar ini
**tidak**.

Dua alasan, keduanya terukur:

1. **`STATUSCLAIM_1` belum ada di sana.** Ia baru ditambahkan `migrations/0005` tahap 1, yang
   belum dijalankan DBA. Kolom itulah yang dibandingkan penyaring **Status Bayar** terhadap
   `'1163'` — tanpa kolomnya, satu dari enam penyaring tidak dapat dibangun sama sekali.
2. **Isinya 1.014 dari 7.703 klaim (13%).** Untuk layar yang isinya justru klaim LAMA, itu
   berarti sebagian besar barisnya hilang — dan hilangnya tidak terlihat sebagai galat,
   hanya sebagai daftar yang lebih pendek.

Pola tiga tabel yang dipakai — objek kerja + `BUSINESS` + `BUSINESSGROUP` — sama dengan
`inboxadmin`, `inboxlaporanklaim`, `inboxmanagerreceivepucl`, dan `komite`.

**`INNER JOIN` ke `BUSINESSGROUP` dipertahankan meski tak satu pun kolomnya dibawa.** Ia
MENYARING: klaim yang lini bisnisnya tidak punya baris di `BUSINESS` tidak muncul.
Membuangnya akan menambah baris tanpa satu pun pesan galat.

### 41.4 Penyaring lini bisnis TIDAK memakai ulang modul lain

Ini keputusan yang paling mudah dilanggar oleh orang berikutnya, karena ketiganya bernama
sama dan tampak dapat dipakai ulang.

| Modul | BONDING | NONMBU |
|---|---|---|
| **Inbox Close Claim** | `businessgroupid` **IN** | `('003','004','006')` |
| Inbox Admin | `businessgroupid` **IN** | `('003','004','006','009')` |
| Inbox Outstanding | `businessgroupid` **NOT IN** | `('003','004','006')` |

Ketiganya disalin dari activity-nya masing-masing dan memang berbeda di Pega.

**Keputusan: `inboxcloseclaim.BusinessLine` berdiri sendiri**, tidak memakai ulang
`inboxoutstanding.ScopeFor` maupun `inboxadmin.BusinessLine`. Kesamaan namanya kebetulan;
isinya tidak sama, dan menyeragamkannya akan menampilkan **kebalikan** dari lini yang diminta
pengguna tanpa satu pun galat.

Dijaga `TestBondingFiltersWithIN` dan `TestNonMBUDoesNotInclude009` — keduanya menyebut modul
pembanding di dalam komentarnya, supaya penyeragaman yang tampak rapi tertangkap saat
ditulis.

### 41.5 ReOpen dan Copy Klaim mencatat PERMINTAAN, bukan mengubah klaim

**Aturannya bukan hasil pembacaan export.** Penelusuran seluruh export menghasilkan nol pada
setiap jalur: ketiga rule modalnya tidak ada, tidak ada rule yang menulis status `1164`,
tidak ada yang menyentuh `PYREOPENCOUNT`/`PYREOPENTIMESTAMP`, tidak ada activity Copy Klaim,
dan `SaveReOpenAct` yang memang ada ternyata satu langkah `Property-Set` yang tidak menulis
apa pun.

Reopen di sistem lama adalah **mekanisme platform Pega** — `PYREOPENTIMESTAMP` terisi pada 47
klaim, dan `UpdateStatus` OOTB menyimpan catatan *"A resolved work object should be reopened
first"*.

**Keputusan Work Owner 2026-09-23**, dicatat sebagai keputusannya dan bukan sebagai temuan:

| Aksi | Efek yang dikehendaki |
|---|---|
| **ReOpen** | `PYSTATUSWORK` ke `'New'` · `STATUSCLAIM_1` ke `'1164'` (Reopen Claim) · `PYREOPENCOUNT` bertambah satu · `PYREOPENTIMESTAMP` diisi |
| **Copy Klaim** | salin polis, objek pertanggungan, dan coverage; nilai estimasi/usulan/akseptasi/pembayaran **tidak** disalin. Klaim baru bernomor sendiri (`D-71`), mulai dari tahap registrasi |

**Kewenangan tulisnya menempuh tabel permintaan, bukan klaimnya.** `P-1` dan `ADR-0004`
menetapkan satu tabel ditulis satu sistem, dan `PC_ASM_FW_GCNMFW_WORK` hari ini ditulis Pega.
Work Owner memilih: aplikasi ini menulis `POOLDATA.CPNC_PERMINTAAN_KLAIM` — tabelnya sendiri
— dan eksekusinya tetap di Pega.

Dengan begitu `P-1` utuh, jejak audit `S-5` terpenuhi, dan tidak ada dua sistem yang saling
menimpa — kegagalan yang bila terjadi TIDAK menghasilkan galat apa pun, hanya data yang
berubah sendiri.

**Konsekuensi yang HARUS diketahui sebelum modul ini menyala di produksi:**

1. **Klaim tidak berubah saat tombolnya ditekan.** Barisnya tetap tampil. Layar karena itu
   WAJIB menandai baris yang permintaannya sudah terkirim — dan penanda itu **menggantikan**
   tombolnya, bukan berdampingan dengannya.
2. **Siapa yang menjalankan permintaan, dan seberapa sering, BELUM ditetapkan.** Sampai itu
   ada, barisnya menumpuk dalam keadaan `menunggu` dan tidak ada yang terjadi.
3. **Tidak ada pemberitahuan otomatis** kepada pemohon saat permintaannya dijalankan.

Ketiganya dicatat di kepala `migrations/0006` supaya terbaca DBA sekaligus.

### 41.6 Niat ikut disimpan pada setiap baris permintaan

`EFEK_STATUS_KERJA`, `EFEK_STATUS_KLAIM`, dan `LINGKUP_SALIN` disimpan pada barisnya meski
aturannya konstan di dalam kode.

**Alasannya:** supaya permintaan yang dijalankan bulan depan dijalankan menurut aturan yang
berlaku **saat ia diajukan**. Bila Work Owner kelak mengubah aturannya, baris lama tetap
menyimpan niat aslinya, dan yang menjalankannya tidak perlu menebak aturan mana yang berlaku.
Menyimpan aturan hanya di dalam kode membuat jejaknya hilang begitu kodenya berubah.

Ini penting justru karena aturannya **bukan** dari export: ia keputusan yang dapat berubah,
tidak seperti perilaku yang disalin.

### 41.7 Satu permintaan menunggu per (jenis, klaim)

Ditegakkan **indeks unik berbasis fungsi**, bukan constraint unik biasa:

```sql
CREATE UNIQUE INDEX POOLDATA.CPNC_PERMINTAAN_KLAIM_UK_PENDING
    ON POOLDATA.CPNC_PERMINTAAN_KLAIM (
        CASE WHEN STATUS = 'menunggu' THEN JENIS   END,
        CASE WHEN STATUS = 'menunggu' THEN CASE_ID END
    );
```

Yang harus unik bukan `(JENIS, CASE_ID)` — sebuah klaim boleh dibuka kembali hari ini dan
diminta lagi tahun depan — melainkan `(JENIS, CASE_ID)` **di antara yang masih menunggu**.
Oracle tidak mengindeks baris yang seluruh kunci indeksnya NULL, sehingga baris yang sudah
dijalankan keluar dari indeks dengan sendirinya.

**Kenapa ini bukan kerapian.** Lapisan usecase sudah memeriksanya lebih dulu, tetapi
pemeriksaan dan penyimpanannya BUKAN satu operasi atomik: dua tab yang terbuka, keduanya
ditekan, dapat lolos keduanya. Pada `salin`, dua baris yang lolos berarti **dua klaim baru
dari satu tombol**.

Bentroknya diterjemahkan menjadi `409` beserta pesan yang dapat dibaca, lewat konstanta
`sqlstore.UniqueKeyName` yang namanya sama dengan nama constraint di migrasi.

### 41.8 Tiga selisih terencana, dinyatakan DI LAYAR

`D-54` menetapkan selisih di luar 13 butir `P-5` menuntut persetujuan Work Owner tertulis.
Ketiganya sudah disetujui 2026-09-23, dan dikirim ke layar lewat `selisih_terencana` —
**bukan disembunyikan sebagai detail teknis**.

| # | Selisih | Sebabnya |
|---|---|---|
| 1 | Kolom "Lama Waktu Klaim" berisi **umur dalam hari** | Di Pega ia terikat `.pxCreateDateTime` berformat `pxDateTime` — tanggal pendaftaran, untuk kedua kalinya |
| 2 | Jumlah total **mengikuti penyaring Status Bayar** | `GCNMCountCloseClaim` kehilangan `{ASIS:TempFilter.DistrictID}`, sehingga totalnya di Pega tidak cocok dengan barisnya |
| 3 | ReOpen dan Copy Klaim mencatat **permintaan** | `P-1` — lihat §41.5 |

Menyatakannya di layar itulah yang membuat keputusan itu terlihat oleh orang yang **memakai**
layarnya, bukan hanya oleh yang membaca dokumen.

### 41.9 Titik akhir "Lama Waktu Klaim" — dan dua kolom yang ditambahkan

Work Owner menetapkan "hitung umur dalam hari" tanpa menyebut titik akhirnya. Yang dipakai:
**sampai klaim ditutup**, bukan sampai hari ini — klaim yang tutup tiga tahun lalu bukan klaim
berumur seribu hari.

Urutan sumbernya:

| # | Kolom | Terisi pada |
|---|---|---|
| 1 | `CLOSECLAIMDATE_1` | 514 klaim — tanggal penutupan menurut bisnis |
| 2 | `PYRESOLVEDTIMESTAMP` | 3.142 klaim — waktu objek kerjanya diselesaikan Pega |
| 3 | waktu sekarang | bila keduanya kosong |

**Kedua kolom pertama TIDAK ada di kueri lama** dan sengaja ditambahkan; keduanya memang ada
di tabel. Baris yang jatuh ke cadangan ketiga ditandai berbeda di layar (`900 hari*`), karena
"dihitung sampai hari ini" adalah arti yang berbeda dari lamanya klaim berjalan.

### 41.10 Penyaring "BELUM LUNAS" membuang baris tanpa kode status — ditiru apa adanya

`A.STATUSCLAIM_1 <> '1163'`. Di Oracle, `NULL <> '1163'` bernilai UNKNOWN, dan baris
ber-UNKNOWN dibuang WHERE. Jadi klaim yang kode statusnya kosong **hilang** saat penyaring
"BELUM LUNAS" dipakai — meski secara bisnis ia jelas belum lunas.

**Keputusan: ditiru apa adanya** — dan sejak 2026-09-23 alasannya lebih kuat daripada `P-5`.

Ditanyakan tegas kepada Work Owner apakah klaim tanpa kode status seharusnya ikut terbaca
sebagai belum lunas. **Jawabannya: tidak.**

Artinya perilaku Pega di sini **benar**, bukan sekadar perilaku yang wajib ditiru. Klaim yang
kode statusnya kosong bukan klaim "belum lunas"; ia klaim yang keadaan bayarnya **belum
diketahui**, dan menempatkannya di bawah penyaring "Belum Lunas" akan menyatakan sesuatu yang
tidak diketahui sebagai fakta.

Ia ditiru **tegas**, bukan kebetulan: `unpaidMatches` di penyimpanan memori memeriksa kode
kosong secara eksplisit, kepala `inboxcloseclaim.sql` menyebutnya beserta larangan
"memperbaikinya", dan `TestUnpaidFilterDropsRowsWithoutStatusCode` menjaganya. Menambahkan
`IS NULL OR` akan memasukkan baris yang justru tidak boleh ada di sana — dan penambahannya
tidak menghasilkan galat apa pun.

**Pertanyaan terbuka yang sebelumnya tercatat di sini: TERTUTUP** (Work Owner, 2026-09-23).

### 41.11 Penomoran bind mengikuti pola yang TERBUKTI menyentuh Oracle

Dua modul melakukannya berbeda, dan keduanya tidak dapat benar bersamaan:

| Modul | Pola | Sudah menyentuh Oracle? |
|---|---|---|
| `inboxoutstanding` | satu nomor per KEMUNCULAN | **ya** — terpasang di `cmd/claimpnc` |
| `inboxadmin` | nomor diulang, nilai dikirim sekali | **tidak** — modulnya belum dirakit |

**Keputusan: mengikuti `inboxoutstanding`**, ditambah satu aturan: nomornya ditulis **menaik
sesuai urutan kemunculan** di dalam teks, sehingga penafsiran driver mana pun — menurut nomor
atau menurut urutan kemunculan — menghasilkan pengikatan yang sama.

Akibatnya nilai lini bisnis dikirim lima kali, status transfer tiga kali, status bayar lima
kali. Kesesuaian jumlahnya dijaga `TestFilterArgsMatchesBindCount` dan
`TestBindMarkersAreUniqueAndAscending`.

### 41.12 Kunci klaim di BADAN permintaan, bukan di jalur URL

Kunci warisan berbentuk `ASM-FW-GCNMFW-WORK PNC-9001` — **mengandung spasi**, dan nama kelas
internal Pega di dalamnya.

**Keputusan: satu endpoint `POST /api/inbox-close-claim/permintaan`** yang melayani kedua
aksi, dengan `jenis` dan `klaim_id` di badan permintaan.

Dua alasan: penyandian jalur URL yang keliru di satu sisi menghasilkan "klaim tidak
ditemukan" alih-alih galat yang jelas; dan `alasan` yang boleh sepanjang 1.500 karakter
memang tempatnya di badan permintaan.

Memisahkannya menjadi dua handler ditolak: keduanya menempuh pemeriksaan, penyimpanan, dan
penolakan yang sama persis, dan menggandakannya membuka celah keduanya menyimpang.

### 41.13 Daftar tetap tampil meski tabel permintaan belum ada

Migrasi `0006` belum dijalankan DBA di lingkungan mana pun, sehingga pembacaan permintaan
akan **gagal di setiap portal hari ini**.

**Keputusan: kegagalan itu tidak menghentikan layar.** Daftarnya tetap tampil, penanda
permintaan tidak muncul, dan `permintaan_terbaca: false` dikirim ke layar yang menyatakannya
kepada pengguna.

Mengembalikannya sebagai galat berarti layar ini **mati total** sampai perubahan skema
selesai — padahal daftarnya sendiri sudah dapat dipakai dan tidak menyentuh tabel itu sama
sekali.

Kegagalannya tetap dicatat di log lewat `logPendingLookupFailure`, yang WAJIB dipanggil pada
setiap jalur yang memakai hasilnya. Dijaga `TestListSurvivesMissingRequestTable`.

### 41.14 Yang TIDAK dibangun, dan alasannya

| Tidak dibangun | Alasan |
|---|---|
| Tautan `.CaseIDView` ke layar detail | Membuka harness `ViewTempDetailClaim`, milik modul lain yang belum ada |
| Kotak centang "Pilih" | Diganti tombol per baris — pilihan dan tindakan tidak lagi terpisah, sehingga tidak mungkin menekan ReOpen tanpa menyadari baris mana yang dipilih |
| Pembatalan permintaan | Aplikasi tidak punya hak `UPDATE` atas tabelnya; yang membatalkan adalah pelaksana |
| Pemeriksaan peran per menu | `TKT-F3-005`, masih terhalang `TKT-F3-004` |

### 41.15 Kewenangan: penjaganya `IsGCNMUser`, dan rule itu SELALU SALAH

Ditanyakan siapa yang boleh melakukan ReOpen dan Copy Klaim. Jawaban Work Owner 2026-09-23:
**`IsGCNMUser`**.

Rule itu ADA di export, dan isinya satu kondisi tunggal (`When/IsGCNMUser-When.xml`):

```
pyConditionValue1 = @(Pega-RULES:ExpressionEvaluators).compareTwoValues(1, "=", 2)
pyLogic           = A
```

Yaitu **`1 = 2`** — kondisi yang tidak pernah benar. Di sistem lama ia dipakai **13 kali** di
navigasi untuk MENYEMBUNYIKAN butir menu ("Report Adjuster", "My Work", "Calendar",
"Lost Adjuster", "Inbox Banding Harga Salvage"); ia sakelar "jangan tampilkan ini", bukan
pemeriksaan peran.

**Akibatnya dinyatakan lebih dulu kepada Work Owner:** memakainya sebagai penjaga membuat
kedua tombol tidak dapat dipakai SIAPA PUN. Tiga alternatif ditawarkan — dibaca sebagai
istilah bisnis "pengguna aplikasi GCNM", dipakai daftar empat access group yang menjaga butir
menunya, atau Work Owner menyebutkan perannya sendiri.

**Work Owner menegaskan pilihannya: rule ditiru apa adanya (`P-5`).** Keputusan itu dihormati
dan diterapkan penuh.

**Satu hal yang perlu dicatat supaya tidak salah dibaca kelak:** rule ini **tidak** menjaga
kedua tombol di `Section/InboxManagerReopen1_Sec-Section.xml`. Di sana tombolnya tidak punya
penjaga visibilitas sama sekali. Menjadikannya penjaga di sini karena itu **keputusan Work
Owner**, bukan peniruan perilaku layar lama.

#### Bagaimana ia diterapkan

| Lapisan | Perilaku |
|---|---|
| Domain | `EvaluateIsGCNMUser()` menuliskan kedua angkanya apa adanya; `CanRequestAction()` memakainya |
| Usecase | menolak dengan `ErrRequestNotAllowed` **sebelum** klaimnya dicari |
| Transport | `403` beserta alasan yang datang dari domain, dan `boleh_mengajukan: false` pada daftar |
| Layar | kedua tombol **dimatikan beserta alasannya**, ditambah pemberitahuan di kepala layar |

**Kenapa diperiksa sebelum klaimnya dicari.** Memeriksanya belakangan berarti pemanggil yang
tidak berwenang tetap dapat menanyakan keberadaan sebuah klaim lewat perbedaan galat yang ia
terima — "tidak ditemukan" untuk kunci yang salah, "tidak berwenang" untuk kunci yang benar.
Perbedaan itu cukup untuk menebak nomor klaim satu per satu. Dijaga
`TestGerbangDiperiksaSebelumKlaimDicari`.

**Kenapa tombolnya dimatikan, bukan disembunyikan.** Kolom "Tindakan" yang kosong tanpa sebab
yang terbaca akan dilaporkan sebagai fitur yang hilang. Tombol mati yang menjelaskan dirinya
sendiri lebih jujur daripada ruang kosong.

#### Gerbangnya seam, dan itu keputusan tersendiri

`usecase.Options.CanRequest` dapat diisi, dan bawaannya adalah aturan yang sesungguhnya.

**Alasannya bukan melonggarkan aturan.** Tanpa seam ini, seluruh jalur pengajuan menjadi
tidak dapat diuji: gerbangnya menolak sebelum apa pun berjalan, sehingga pemeriksaan klaim,
pencatatan niat, penolakan permintaan ganda, dan pemetaan galatnya tidak pernah tersentuh
satu uji pun.

Akibatnya bukan sekadar cakupan uji yang turun. Hari ketika Work Owner membuka gerbangnya,
yang menyala adalah kode yang **tidak pernah sekali pun dijalankan** — dan itu justru saat
cacatnya paling mahal, karena kedua aksi menyentuh klaim yang sudah tutup.

Di `cmd/claimpnc` seam itu **tidak diisi**, sehingga yang berlaku di aplikasi sungguhan tetap
aturan yang sesungguhnya. Dijaga tiga uji: `TestIsGCNMUserSelaluSalah` (domain),
`TestRequestDitolakSaatGerbangTertutup` (usecase), dan
`TestGerbangTertutupMenolakPengajuanDengan403` (transport).

#### Cara menyalakannya kelak

Ganti isi `EvaluateIsGCNMUser` dengan aturan yang sesungguhnya, lalu hapus
`TestIsGCNMUserSelaluSalah` yang sengaja mengunci keadaan hari ini. Tidak ada tempat lain
yang perlu disentuh.

### 41.16 Pelaksana permintaan BELUM ADA — dan layar menyatakannya

Ditanyakan siapa yang MENJALANKAN permintaan yang sudah tercatat, mengubah klaimnya di sisi
Pega. Jawaban Work Owner 2026-09-23: **belum ada pelaksananya**, dan itu diterima untuk
sekarang.

**Keputusan: dinyatakan tegas di layar**, lewat `pelaksana_belum_ada`. Pengguna yang
mengajukan lalu menunggu perubahan yang tidak akan datang akan melaporkannya sebagai
kegagalan sistem — dan yang ditelusuri orang berikutnya adalah cacat yang tidak ada.

Keterangan itu muncul **hanya bila pengajuannya terbuka**. Menampilkan keduanya sekaligus
akan membuat pengguna mengira tombolnya mati karena pelaksananya belum ada, padahal sebabnya
kewenangan. Dijaga `TestDaftarMenyatakanPengajuanTertutup` dan
`TestDaftarMenyatakanPelaksanaBelumAda`.

**Akibat yang berlaku hari ini:** karena pengajuannya sendiri tertutup (§41.15), tidak akan
ada baris permintaan yang tercatat sama sekali. Kedua keterangan ini baru saling bergantian
ketika gerbang §41.15 dibuka.

### 41.17 Kewenangan menu: yang masih belum ada

Di Pega, butir menu ini dijaga `When/IsManagerPNC_CLOSE-When.xml` — empat access group
(`Administrators`, `CaseManager`, `PncManagerAdmin`, `PNCKomiteTeknik`) **ditambah tiga
Operator ID perorangan** yang namanya tertanam di dalam rule. Ketiga nama itu tidak dibawa
(`D-15`); penggantinya `M_OTORISASI_PNC`, yang sudah menentukan siapa **melihat** butirnya.

**Yang belum ada: pemeriksaan di endpoint.** Sampai `TKT-F3-005` dikerjakan, siapa pun yang
punya sesi dapat memanggil `POST /permintaan`.

Taruhannya di sini lebih besar daripada di layar yang hanya membaca, dan perlu dinyatakan
terbuka: kedua aksinya menyentuh klaim yang **sudah tutup** — sebagian di antaranya sudah
dibayar. `D-59` sudah menghapus pemisahan tugas, sehingga jejak permintaan inilah satu-satunya
kontrol pengimbang yang tersisa.

**RISIKO YANG DITERIMA SADAR**, dicatat supaya tidak ditemukan sebagai kejutan saat pentest.

---

## 42. Modul Inbox Analyst Doctor (2026-09-23, sesi kedua puluh dua)

Keputusan implementasi modul `inboxanalystdoctor` — antrean penilaian medis, `MENU_ID 60`.
Jalannya pengerjaan ada di `catatan-pengembangan.md` §41.

### 42.1 Nama modul: `inboxanalystdoctor` / `inbox-analyst-doctor`

`D-81` menetapkan nama modul mengikuti nama yang dipakai Work Owner. Butir menunya berbunyi
**"Inbox Analyst Doctor"** (`m_menu_aplikasi_pnc.csv`, `MENU_ID 60`), dan judul di dalam
harness-nya sama persis.

Nama itu berbahasa Inggris **di sumbernya sendiri**, sehingga tidak ada yang perlu
diterjemahkan — berbeda dari `masterrekening` dan `masterstatus` yang nama Indonesianya
memang nama bisnisnya. Isinya tetap berbahasa Inggris (`D-80`).

### 42.2 Penyaring dan urutan disalin dari Report Definition, bukan dari kueri lain

Tiga penyaring (`A AND B AND C`), gabungan dalam ke `Assign-Worklist`, dan urutan
`PXCREATEDATETIME DESC, PZINSKEY DESC` seluruhnya dari
`Report Definition/InboxAnalystDoctor_RD-RD.xml`.

**Tidak satu pun dipinjam dari modul inbox lain**, meski beberapa terlihat mirip. Dua
perbedaan yang paling mudah diseragamkan tanpa sengaja:

| Hal | Modul ini | Modul lain |
|---|---|---|
| Status yang dikecualikan | `Resolved-Completed` **saja** | `inboxoutstanding` mengecualikan **keduanya** |
| Tabel penugasan | `PC_ASSIGN_WORKLIST` (per orang) | tab RCL/PUCL memakai `PC_ASSIGN_WORKBASKET` |

Keduanya dikunci uji, karena keduanya gagal **tanpa galat** — yang berubah hanya isi layar.

### 42.3 Nama kolom untuk dua properti tak terekspos: konvensi `_1`, menunggu DBA

**Keputusan:** memakai `ISCOMPLIANCETRANSFER_1` dan `ANALYSTDOCTORREMAKS_1`, mengikuti
konvensi yang terbukti berlaku pada properti `ClaimData` lain di tabel yang sama.

**Dasarnya:** jawaban Work Owner 2026-09-23 — *"nilainya langsung di-set 2"* — menutup
pertanyaan apakah nilainya perlu dicarikan sumber pengganti. Ia nilai tersimpan; yang tersisa
hanyalah namanya.

**Yang TIDAK dipilih, dan kenapa:**

| Pilihan | Alasan tidak diambil |
|---|---|
| `POOLDATA.PEGA_DASHBOARDPNC.SURPLUS1` | Ia tabel **cermin laporan** yang disegarkan berkala (`LastRefresh_act`). Inbox yang membaca cermin akan menampilkan keadaan basi: tugas yang sudah ditangani masih muncul, yang baru masuk belum muncul |
| Menghilangkan penyaingnya supaya kuerinya jalan | Akan menampilkan **seluruh** tugas worklist pemanggil sebagai tugas medis, tanpa satu pun galat |
| Menunda modul sampai DBA menjawab | Menahan seluruh pekerjaan — domain, layar, uji — yang sama sekali tidak bergantung pada jawaban itu |

**Konsekuensi yang diterima sadar:** jalur Oracle **belum terbukti**. Bila kolomnya tidak ada,
kuerinya gagal dengan ORA-00904 yang menyebut namanya. Kegagalan yang menyebut sebabnya dipilih
di atas layar yang terlihat benar.

`-periksa` memeriksanya dalam **dua langkah terpisah** — tabel dan kolom — karena keduanya
gagal karena sebab yang berbeda, dan galat yang menyebut sebab yang salah mengirim orang yang
memperbaikinya ke arah yang keliru.

### 42.4 Kolom yang belum terbawa tetap ada di kontrak API

`komentar_pic_teknis` tetap dikirim meski selalu kosong terhadap Oracle. Menghilangkannya dari
jawaban akan membuat layar berhenti menggambarnya — dan isian yang belum terbawa menjadi tidak
terlihat oleh siapa pun.

Preseden: `inboxmanagerreceivepucl` memakai `CAST(NULL …)` pada "Jumlah Lembar Dokumen" dengan
alasan yang sama.

### 42.5 "Lama Waktu Klaim" dihitung di Go, bukan di SQL dan bukan dari `LAMAKLAIM_1`

Tiga pilihan, dan dua ditolak:

| Pilihan | Alasan |
|---|---|
| Baca `LAMAKLAIM_1` | Kolomnya ada (86 nilai berbeda) tetapi **tidak dibaca layar ini**, dan artinya — dihitung sampai kapan, diperbarui kapan — tidak terbaca dari export mana pun |
| Hitung di SQL | "Hari" yang dimaksud pengguna adalah hari **WIB** sementara kolomnya UTC. Menaruh konversi zona waktu di dalam SQL adalah cara tercepat menyebarkannya — persis cacat `Set7Hours` sistem lama, 118 titik di 36 activity |
| **Hitung di Go lewat seam Clock** | Terpusat di satu tempat (`F-5`), dan dapat diuji tanpa bergantung jam mesin |

Dihitung terhadap **pergantian tanggal**, bukan selisih jam dibagi 24: tugas yang masuk pukul
23.00 dan dilihat pukul 01.00 keesokan harinya sudah berumur satu hari bagi pengguna.

### 42.6 Identitas pemanggil diperiksa SEBELUM portal dan penyimpanan

Urutannya mengikat, dan diuji. Dua alasan:

1. Memilih repo untuk permintaan yang pasti ditolak adalah perjalanan yang terbuang.
2. Yang lebih penting: urutan ini membuat kegagalan sesi terbaca **sebagai kegagalan sesi** —
   bukan sebagai antrean kosong.

Galatnya dipetakan **409**, bukan 401 dan bukan 200 berisi daftar kosong. 401 akan melempar
pengguna ke halaman masuk lalu mengembalikannya ke galat yang sama; 200 berisi daftar kosong
akan membuatnya mengira ia tidak punya pekerjaan.

### 42.7 Perbandingan operator memakai `UPPER` di kedua sisi

Diputuskan meski mengorbankan pemakaian indeks biasa. Dasarnya `11-SECURITY.md` §3.1: nama
identitas di sistem lama terbukti muncul dalam dua kapitalisasi.

Kegagalan yang dihindari bersifat **senyap** — antrean kosong bagi pengguna yang login-nya
tersimpan berbeda huruf. Kegagalan yang diterima bersifat **terukur** — kueri lebih lambat,
yang dapat dilihat dan diperbaiki dengan function-based index.

Penyimpanan memori memakai `strings.EqualFold` supaya keduanya sepakat; tanpa itu, uji yang
lulus di memori tidak menyatakan apa pun tentang Oracle.

### 42.8 Modul ini TIDAK menulis apa pun, dan ketiadaannya disengaja

Menyelesaikan tugas Analyst Doctor berarti menjalankan Flow Action `SendAnalystDoctor`, yang
menulis objek kerja **dan** memindahkan penugasannya. Keduanya milik Pega selama masa paralel
(`P-1`).

Membawanya ke sini menuntut modul alur kerja tersendiri — bukan satu method di antrean. Seam
`Repo` karena itu hanya punya `List`, dan ketiadaan method tulis dinyatakan di komentarnya.

### 42.9 Data contoh tidak memakai Operator ID sungguhan

`DRRATNA` tertanam di `Flow/Register_Flow.xml` sebagai penerima seluruh tugas tahap ini —
salah satu dari 24 hardcode yang `D-15` hapus.

`D-69` mengizinkannya ditulis di **dokumen** agar tiket dapat menunjuk hardcode mana yang
dibuang; izin itu tidak berlaku untuk **data yang dijalankan**. Data contoh memakai
`ADMINKLAIM` dan `ADMINLAIN`, dan satu uji menjaga `DRRATNA` tidak masuk.

### 42.10 Utang teknis yang disadari

| # | Utang | Pemilik |
|---|---|---|
| 1 | **Dua nama kolom belum dikonfirmasi DBA** — `ISCOMPLIANCETRANSFER_1`, `ANALYSTDOCTORREMAKS_1`. Sampai dijawab, jalur Oracle belum terbukti | DBA |
| 2 | **Penerima tugas tahap ini masih satu operator yang tertanam di alur Pega.** Selama belum pindah ke master data, hanya satu orang yang melihat isi antrean | Work Owner + `F-4` |
| 3 | **Pemeriksaan kewenangan menu belum ada** (`TKT-F3-005`). Yang meredamnya penyaring identitas — peredam, bukan kendali. Barisnya menyangkut data medis yang `FR-R2` batasi | `TKT-F3-005` |
| 4 | **`registrasi.Claim.ComplianceTransfer` bertipe `bool`** padahal domainnya `"1"` dan `"2"`. Cabang `IsCompliance` akan bernilai benar untuk klaim yang menuju Analyst Doctor. Di luar lingkup sesi ini — lihat `catatan-pengembangan.md` §41.13 | keputusan tersendiri |
| 5 | **Penanda bind gaya Oracle `:n`** belum portabel ke PostgreSQL. Utang yang sudah ada sebelum modul ini dan berlaku untuk seluruh berkas `.sql` | `09-DATABASE-STRATEGY.md` §10 |

## 43. Modul Inbox RCL/PUCL (2026-09-24, sesi kedua puluh tiga)

Butir menu `MENU_ID 61`, pengganti `Harness/RCLPUCL_Harness`. Layar ini adalah antrean klaim
yang **ditolak (RCL)** atau **diproses ulang (PUCL)**, dipartisi menurut perjalanan surat PUCL.

### 43.1 Nama modul

`inboxrclpucl` (backend) dan `inbox-rcl-pucl` (frontend), mengikuti `D-81`: nama modul diambil
dari nama yang dipakai Work Owner. Butir menunya sendiri berbunyi **"Inbox RCL/PUCL"**, dan
harness-nya memuat judul yang sama (`pyCaption Inbox RCL/PUCL`). Isi modulnya berbahasa
Inggris sesuai `D-80`; yang berbahasa Indonesia hanya nama modul, nama field JSON, dan teks
yang dilihat pengguna.

### 43.2 Keputusan: tab "Klaim MSIG" dibangun apa adanya meski kemungkinan selalu kosong

**Masalahnya.** Penyaring tab ketiga adalah `MSIG_1 = 'MSIG'`. Kolom itu **tidak muncul** di
inventaris kolom terisi yang dibaca langsung dari katalog Oracle pada 2026-09-22
(`docs/kolom-t-claimlist-admin.md` §B.3), sementara keempat kolom PUCL lain semuanya ada di
sana. Artinya kolomnya ada tetapi tampaknya belum pernah diisi.

**Tiga pilihan diajukan:** bangun apa adanya dan tandai sebagai temuan · tunggu konfirmasi DBA
lebih dulu · tab MSIG tidak dibawa.

**Jawaban Work Owner: bangun apa adanya, tandai sebagai temuan.**

**Yang mengikuti dari jawaban itu, dan satu pembedaan yang disengaja.** Tabnya **tidak**
ditandai `Blocked`, meski layar ini sudah punya mekanisme tab terhalang yang dipakai modul
lain. Alasannya: tab terhalang DITOLAK di `NewQuery` sebelum menyentuh penyimpanan, sehingga
menandainya begitu akan membuat baris yang **mungkin memang ada** tidak pernah ditampilkan.
Kuerinya dapat dijalankan, dan kosongnya adalah **jawaban** — bukan ketidakmampuan menjawab.

Yang dipakai sebagai gantinya tiga lapis, seluruhnya berupa DATA dari server:

- `Tab.Notice` — keterangan di atas grid, hanya pada tab itu;
- `PlannedDifferences` — butir pertama, di bawah tabel;
- `-periksa` — mencetak kueri `SELECT COUNT(*) … WHERE MSIG_1 IS NOT NULL` yang DBA perlu
  jalankan.

Ketiganya hilang dengan sendirinya begitu keterangannya diubah di satu tempat.

### 43.3 Keputusan: `PUCLAPPROVE_1 <> '1'` direplikasi meski menyembunyikan baris

**Masalahnya.** `NULL <> '1'` bernilai **UNKNOWN** di Oracle maupun PostgreSQL — bukan TRUE.
Klaim yang penanda persetujuannya belum pernah diisi karena itu tidak muncul di tab Kelengkapan
Dokumen maupun Klaim MSIG, meski suratnya sudah dicetak dan ia jelas belum disetujui. Kolomnya
hanya punya **dua nilai berbeda** di produksi, sehingga jumlah baris terdampak bisa besar.

**Keputusan: direplikasi apa adanya (`P-5`).** Memperbaikinya menjadi
`(… IS NULL OR … <> :x)` akan **menambah** baris yang di Pega tidak pernah terlihat — itu
perubahan perilaku pada layar yang sedang diuji kesetaraannya, bukan perbaikan yang sudah
diputuskan siapa pun.

**Yang dikerjakan supaya keputusan ini tidak berubah diam-diam:**

- penyimpanan memori meniru semantik UNKNOWN secara eksplisit (`matchesApproval` menolak nilai
  kosong lebih dulu, alih-alih menulis `!=` yang akan meloloskannya);
- satu uji khusus menjaganya (`TestAnEmptyApprovalFlagAlsoHidesTheRow`), beserta satu baris
  contoh yang memang tertolak karenanya;
- dinyatakan ke pengguna lewat `PlannedDifferences`;
- `-periksa` mencetak kueri sebaran nilainya saat tabnya kosong.

**Pembanding diikat sebagai TEKS**, mengikuti `CountKlaimPUCL-SQL.xml` (`<> '1'`).
`ReminderPUCL-SQL.xml` menulis `<> 1` tanpa kutip pada kolom yang sama — dua rule Pega yang
tidak sepakat tentang tipe kolomnya sendiri. Yang dipilih bentuk bertanda kutip karena DDL-nya
tidak tersedia (`R-08`) dan seluruh kolom ber-akhiran `_1` lain dibaca sebagai teks.

### 43.4 Keputusan: rentang tanggal hanya menyetir ekspor, bukan grid

**Masalahnya.** Tab "Cetak Surat" punya dua isian tanggal ("FROM RCL/PUCL", "TO RCL/PUCL")
di atas grid, tetapi Report Definition grid-nya **tidak menyaring tanggal sama sekali**.
Keduanya memasok `GetDataPUCLRCLForDailyReport-SQL.xml` — kueri berbeda di balik tombol ekspor.

**Tiga pilihan diajukan:** replikasi apa adanya · tanggal menyaring grid DAN ekspor · tanggal
menyaring grid, ekspor mengikuti grid.

**Jawaban Work Owner: replikasi apa adanya (`P-5`).**

**Akibat yang harus ditangani di antarmuka.** Pengguna akan mengisi tanggal, melihat tabel
tidak berubah, lalu melaporkannya sebagai kerusakan. Karena itu:

- peringatan tegas di bawah kedua isian — *"Kedua tanggal ini hanya dipakai tombol unduh"* —
  ditambah penjelasan bahwa isi berkasnya pun berbeda;
- rentang tanggal **tidak ikut dikirim** pada permintaan daftar, supaya lalu lintas jaringan
  tidak menyiratkan sebaliknya (diuji);
- butir tersendiri di `PlannedDifferences`.

Kedua tanggal ditaruh di `ReportRequest`, **bukan** di `Query`. Menaruhnya di `Query` akan
membuat pembaca kode mengira grid-nya ikut tersaring.

### 43.5 Keputusan: laporan harian adalah tipe tersendiri, bukan `WorkItem`

`DailyReportRow` dipisah dari `WorkItem` karena isinya memang berbeda, dan menyamakannya akan
menyembunyikan perbedaan yang justru harus terlihat:

| | Grid | Laporan harian |
|---|---|---|
| Penyaring status kerja | ya | **tidak** |
| Penyaring tanggal cetak / persetujuan | ya | **tidak** |
| Rentang tanggal | tidak | **ya** |
| Cabang Personal Accident tanpa antrean | — | **ya** (`UNION`) |
| `STATUSCLAIM_1` (Status Klaim, 33 kode) | tidak digambar | **digambar** |
| `LAMAKLAIM_1` | digambar | **tidak ada** |

`UNION` dipertahankan, bukan `UNION ALL`: klaim PA yang berada di antrean RCL/PUCL memenuhi
kedua cabang, dan `UNION ALL` akan memunculkannya dua kali di berkas.

**`STATUSCLAIM_1` versus `STATUSKLAIM_1`** hanya berbeda satu huruf dan artinya berbeda jauh.
Keduanya dipisah sebagai isian tersendiri di penyimpanan memori supaya tertukarnya dapat
tertangkap uji (`TestDailyReportCarriesClaimStatusNotExpiryStatus`).

### 43.6 Keputusan: satu rute ekspor, percabangan di sifat tab

Ketiga tab memakai `GET /api/inbox-rcl-pucl/ekspor` yang sama; yang menentukan isi berkasnya
adalah `Tab.HasDateRangeReport`, **bukan** kode tab yang ditulis tetap di lapisan transport.

Alasannya dua: di Pega pun tombolnya satu dan sama (hanya activity di baliknya yang berbeda),
dan menaruh nomor tab di transport akan membuat penambahan tab kelak menuntut suntingan di dua
tempat.

Nama berkasnya dibedakan — `rcl-pucl-kelengkapan-dokumen.csv`, `rcl-pucl-klaim-msig.csv`,
`laporan-harian-rcl-pucl-<dari>-sd-<sampai>.csv`. Di layar ini alasannya lebih kuat daripada
biasa: ketiga tab punya kolom **identik**, sehingga berkas yang tertimpa di folder unduhan
tidak dapat dikenali dari isinya sama sekali.

### 43.7 Keputusan: kode jalur diterjemahkan di Go, bukan di SQL

`RCL_PUCL_1` dikembalikan **mentah** sebagai `TRACK_CODE`; penerjemahannya menjadi "RCL"/"PUCL"
dikerjakan `inboxrclpucl.TrackOf`.

Ini **berbeda** dari modul Inbox Manager Receive / PUCL, yang menuliskan `CASE` penerjemah di
dalam SQL-nya. Yang dipakai di sini adalah pola `ClaimTypeOf` pada modul itu, dan alasannya
sama: penyimpanan SQL dan penyimpanan memori wajib menghasilkan teks yang sama persis, dan dua
penerjemah di dua tempat dapat menyimpang tanpa ketahuan.

Hasilnya identik dengan `CASE` tanpa `ELSE` di sistem lama: kode di luar `1` dan `2`
menghasilkan teks kosong.

**Satu selisih terencana:** kueri laporan lama menulis angka mentah ke dalam berkas Excel. Di
sini ia diterjemahkan, supaya berkas dan layar menyebut hal yang sama dengan kata yang sama.

### 43.8 Keputusan: urutan baris TIDAK diubah, meski kuncinya tidak terlihat

Urutan mengikuti `PXCREATEDATETIME DESC, PYID DESC` — terbaca dari `ORDER BY 5 DESC, 7 DESC`
pada SQL hasil generate Pega.

Kolom itu **tidak ditampilkan** di layar; yang tampil sebagai "Tanggal Masuk Inbox" adalah
`TANGGALKIRIMPUCL_1`, dan keduanya dapat terpaut berbulan-bulan. Tabel karena itu dapat terbaca
tidak urut oleh penggunanya.

Sempat dipertimbangkan mengganti kunci urut ke kolom yang terlihat. **Tidak diambil**:
mengganti kunci urut mengubah baris mana yang ada di halaman pertama, dan itu selisih yang
belum diputuskan siapa pun (`P-5`). Ia dinyatakan lewat `PlannedDifferences`, bukan diperbaiki
sepihak.

### 43.9 Keputusan: `TRUNC` pada kolom tanggal diganti rentang setengah terbuka

Kueri lama menulis `trunc(TANGGALKIRIMPUCL_1) >= … AND trunc(…) <= …`. `TRUNC` pada kolom
dilarang (`09-DATABASE-STRATEGY.md` §4), mematikan index, dan tidak portabel ke PostgreSQL.

Penggantinya `>= awal AND < akhir + INTERVAL '1' DAY`, yang **memilih baris yang sama persis**
— termasuk baris yang punya komponen jam — dan tetap dapat memakai index. Menuliskannya sebagai
`<= akhir` akan membuang seluruh baris yang jamnya bukan tengah malam, dan kesalahan itu hanya
terlihat pada data nyata. Satu uji menjaganya (`TestDailyReportIncludesTheWholeLastDay`).

### 43.10 Keputusan: aksi tulis digambar tetapi ditolak beralasan

Layar lama punya dua tindakan yang menulis: **mencetak surat PUCL/RCL** — yang mengisi
`TANGGALCETAKDOKUMENPUCL_1` sehingga klaimnya **berpindah tab** — dan **Reminder PUCL**.

**Jawaban Work Owner: tombol digambar, aksi ditolak beralasan.** Mengikuti preseden
`RejectWrite` pada modul Inbox Manager Receive / PUCL.

Jawabannya **501**, bukan 403 maupun 404: 403 akan menyatakan pengguna tidak berwenang padahal
ia berwenang; 404 akan membuat tombolnya terbaca sebagai kerusakan. 501 menyatakan yang
sebenarnya — alamatnya ada, permintaannya sah, kemampuannya yang belum dibangun.

Di antarmuka, keduanya dinyatakan lewat panel **"Yang masih dikerjakan lewat Pega"** di bawah
tabel. Tanpa itu, layar yang kehilangan tombolnya akan dilaporkan sebagai kerusakan, dan
penggunanya tidak akan tahu ia masih harus mengerjakannya di Pega.

### 43.11 Keputusan: layar ini TIDAK disatukan dengan Inbox Manager Receive / PUCL

Keduanya membaca **workbasket yang sama** (`RCLPUCL`) pada tabel yang sama. Yang membedakan
adalah seberapa halus antrean itu dipartisi:

| | Cakupan |
|---|---|
| Inbox Manager Receive / PUCL (`MENU_ID 56`) | satu tab RCL/PUCL tanpa penyaring halus — **superset** layar ini, untuk penyelia |
| Inbox RCL/PUCL (`MENU_ID 61`) | tiga tab menurut perjalanan surat PUCL, untuk petugas yang mengerjakannya |

**Jawaban Work Owner: hanya `MENU_ID 61` yang dikerjakan; keduanya tetap terpisah.** Pega pun
punya dua menu dan dua harness untuk keduanya, ditujukan pada peran yang berbeda. Menunjuk
keduanya ke satu rute akan menghilangkan partisi yang justru menjadi inti layar ini.

`MENU_ID 62` "Inbox RCL" (`RCL_Harness`) **belum dipetakan** — ia harness tersendiri dan belum
dianalisis sama sekali.

### 43.12 Perangkap penamaan yang didokumentasikan, bukan diseragamkan

Dua judul kolom yang sama menunjuk kolom basis data yang **berbeda** di kedua layar, dan
keduanya bersilangan:

| Judul | Inbox RCL/PUCL | Inbox Manager Receive / PUCL |
|---|---|---|
| Status RCL/PUCL | `RCL_PUCL_1` | `STATUSKLAIM_1` |
| Status Kadaluarsa | `STATUSKLAIM_1` | `STATUSCASE_1` |

Keduanya diverifikasi dari sel grid section masing-masing, bukan disimpulkan. Masing-masing
layar membawa pemetaannya sendiri (`D-13`, `P-5`); menyamakannya akan menampilkan kolom yang
salah **tanpa satu pun galat**, karena keempat nilainya sama-sama teks yang masuk akal.

Perangkapnya ditulis di tiga tempat yang akan dibaca orang yang hendak menyamakannya: komentar
pada `WorkItem.Track` dan `WorkItem.ExpiryStatus`, kepala berkas `.sql`, dan uji alias kueri.

### 43.13 Yang TIDAK dibangun, dan alasannya

| Tidak dibangun | Alasan |
|---|---|
| Kotak cari | Ketiga Report Definition tidak menyaring kata kunci sama sekali. Menambahkannya adalah kemampuan baru pada layar yang sedang diuji kesetaraannya |
| Penyaring apa pun pada grid | Seluruh penyaringnya TETAP di Pega; tidak satu pun dapat diubah pengguna |
| Migrasi basis data | Seluruh tabelnya milik Pega (`P-1`); modul ini hanya membaca |
| Pemeriksaan peran | `When/IsRCLPUCL` membatasi menu pada `PncRCLPUCL` + `Administrators`, bukan `ViewClaimPNC`. Penegakannya `TKT-F3-004`, belum ada — dan di layar ini tidak ada peredam apa pun, karena antreannya bersama |

## 44. Inbox RCL/PUCL — aksi klik Nomor Case (2026-09-24)

Melengkapi §43 setelah dua koreksi Work Owner atas penelusuran aksi klik baris.

### 44.1 Keputusan: tautan pindah ke sel "Nomor Case", kolom tombol dibuang

Bukti: ketiga section RCL/PUCL menggambar sel Nomor Case sebagai `pyUIElement = link`
ber-`pyLabel = .pyID`, menjalankan `SetAssignmentInboxPUCL_act` dengan `inskey = .pzInsKey`.
Tidak ada kolom tombol di mana pun.

Kolom tombol "Lihat Detail" pada versi pertama modul ini adalah **kemampuan yang dikarang**,
dan dibuang. `D-13` menetapkan tampilan mengikuti layar lama.

### 44.2 Keputusan: TIDAK berpindah halaman, dan tidak diarahkan ke View Claim

Dua kemungkinan tujuan diperiksa, dan keduanya ditolak dengan alasan:

| Tujuan | Kenapa ditolak |
|---|---|
| Layar kerja `SendtoRCLPUCL` | **Section-nya tidak ada di export** (`R-16`). Isinya belum pernah terbaca; membangunnya berarti menebak |
| Layar rincian `ViewTempDetailClaim` | Memang ada, tetapi dibuka `PNCInboxAdmin`, `PNCSearchKlaim`, `InboxManagerReopen1_Sec`, `InputProgressClaim`, dan `DashboardClaimShow_Sec` — **tidak satu pun dari RCL/PUCL**. Mengarahkan ke sana menambah kemampuan yang tidak pernah ada di layar ini |

Yang dibangun: **panel di halaman yang sama**, menggambar kesembilan isian baris yang sudah
diambil, ditambah `pzInsKey` dan keterangan yang menyebut nama artefak yang hilang.

Tidak ada permintaan tambahan ke server, dan tidak ada isian yang dikarang — konsekuensi
langsung dari larangan "No Shortcuts" pada proses bisnis yang **tidak dapat** dipelajari.

### 44.3 Konsekuensi yang diterima

1. Petugas **tidak dapat mengerjakan** klaim dari layar ini — hanya memantau dan mengunduh.
   Itu bukan pilihan rancangan melainkan akibat `P-1` ditambah rule yang hilang.
2. Panelnya **bukan** pengganti layar kerja, dan tidak boleh tumbuh menjadi begitu. Begitu
   section `SendtoRCLPUCL` diterima, yang dibangun adalah layar aslinya — bukan perluasan
   panel ini.
3. `ViewTempDetailClaim` tetap layak menjadi modul tersendiri untuk keempat layar yang
   memang membukanya. Pemetaannya sudah selesai: 8 tab, 106 properti terikat, hampir tanpa
   section bersarang.

## 45. Layar kerja RCL/PUCL (2026-09-24)

Menggantikan §44.2 setelah Work Owner menambahkan rule `Section SendtoRCLPUCL` yang hilang.

### 45.1 Keputusan: layar kerjanya dibangun BACA SAJA

Di Pega ia layar tulis — bagian pertama menyusun lampiran surat RCL/PUCL (tindakan "Cetak
Surat"), bagian kedua mencatat penerimaan dokumen (memo penulisnya: *"add button save"*).
Keduanya menulis objek kerja, yang selama masa paralel dimiliki Pega (`P-1`).

Yang dibawa hanya pembacaannya. Layar menyatakan itu sendiri lewat `tindakan_masih_di_pega`,
sebagai DATA dari server — bukan teks tetap — supaya ia berubah di satu tempat begitu
kepemilikan tabelnya berpindah.

### 45.2 Keputusan: `UP` DIPERBAIKI menjadi nilai pertanggungan

> **DICABUT 2026-09-24 oleh §47.1.** Work Owner meralat: UP memang `ObjectName`. Isi di bawah
> dibiarkan utuh sebagai rekaman keputusan yang pernah diambil, bukan sebagai aturan yang
> berlaku.

`SetDataLampiranSuratRCLPUCL_Act` menetapkan `.UP` **dan** `.NamaPeserta` dari ekspresi yang
sama persis (`ObjectList(1).ObjectName`). Kolom "UP" di Pega karena itu berisi nama objek,
bukan nilai pertanggungan — cacat kelas `R-19`, karena ia menyangkut uang.

Ia **tidak diperbaiki sepihak**. Ia diangkat sebagai pertanyaan terbuka, dan dijawab
**Work Owner pada 2026-09-24: "Iya nilai pertanggungan"**.

`P-5` menetapkan perilaku dibawa **kecuali** perbaikannya diputuskan eksplisit, dan inilah
keputusan itu — mekanisme yang sama persis dengan ketiga belas butir `D-49`, hanya lebih muda.

**Sumber penggantinya tidak dikarang:**

```
.UP <- pyWorkPage.ClaimData.ObjectList(1).ObjectCoverageList(1).SumTSI
```

`SumTSI` adalah properti kelas `ASM-FW-GCNMFW-Data-ObjectCoverage`, kolomnya `SUMTSI` pada
`POOLDATA.T_CLAIM_OBJECTCOVERAGE`, dan `GetDataSlinkAllFOG-SQL.xml` membacanya dari tabel itu.
Navigasinya **sama persis** dengan `JumlahTagihan` di sebelahnya, hanya berhenti satu tingkat
lebih dangkal — perbaikan ini karena itu tidak memperkenalkan jalur baru, ia memotong jalur
yang sudah terbukti dipakai.

**Satu jalan pintas yang ditolak:** mengambil UP dari `T_CLAIM_ADJUSTMENT`, tabel yang
subkuerinya sudah ditulis untuk jumlah tagihan. Itu lebih sedikit kode, dan akan menampilkan
**nilai usulan sebagai nilai pertanggungan** — dua angka yang sama-sama masuk akal, tanpa satu
pun galat bila tertukar.

**Uji lama dibalik, bukan dihapus.** Yang dulu menuntut kedua isian SAMA kini menuntut
keduanya BERBEDA. Uji yang dihapus tidak menahan siapa pun mengembalikan salin-tempel lama
atas nama "menyetarakan dengan Pega"; uji yang dibalik menahannya.

Selisihnya terhadap Pega dinyatakan di muka lewat selisih terencana — pada uji kesetaraan,
isian ini akan berbeda pada **setiap** klaim yang punya objek.

### 45.2b Keputusan: syarat `when` pada layar kerja BELUM DIBERLAKUKAN

`pyMemo` pada `SendtoRCLPUCL` mencatat `visibility when .ClaimData.PUCLStatus.RCL_PUCL != 3`
— ada satu nilai kode jalur yang menyembunyikan seluruh layar ini di Pega.

**Work Owner, 2026-09-24: "tidak usah pakai when dulu".** Layar kerja karena itu terbuka untuk
kode jalur apa pun.

Yang dikerjakan justru **tidak menambah kode**: tidak ada penjaga, tidak ada penolakan, tidak
ada cabang. Konstanta `TrackHidden` tetap ada — tidak dipakai kode mana pun — karena ia
memegang satu-satunya rekaman bahwa syarat itu ada, dan syaratnya baru dapat ditegakkan
setelah arti kode `3` diketahui.

Akibatnya dinyatakan apa adanya lewat selisih terencana: bila nilai itu memang menandai klaim
yang seharusnya tidak dikerjakan lewat layar ini, klaimnya **tetap dapat dibuka di sini
padahal di Pega tidak**.

### 45.3 Keputusan: "pertama" ditetapkan tegas dengan `ORDER BY`

Pega memakai indeks `(1)` pada page list klipboard, dan urutannya tidak terbaca dari export
mana pun. Di sini: `OBJECTID` terkecil untuk objek, lalu `COVERAGEID` dan `ADJUSTMENTID`
terkecil untuk adjustment-nya.

Ini **selisih terencana**. Membiarkannya tidak ditetapkan akan membuat isian surat
berubah-ubah antar pemanggilan pada klaim yang punya lebih dari satu objek — cacat yang tidak
menghasilkan satu pun galat.

Adjustment-nya terikat pada **objek pertama**, bukan sekadar adjustment pertama klaim, karena
jalur Pega-nya `ObjectList(1).ObjectCoverageList(1).AdjustmentList(1)`.

### 45.4 Keputusan: penyaringnya HANYA kunci dan kelas objek kerja

Tanpa antrean, tanpa status kerja — persis seperti Pega, tempat tautannya mengirim `inskey`
tanpa satu pun penyaring antrean.

Alasannya bukan kemudahan: klaim yang sudah berpindah antrean sejak daftarnya dimuat tetap
harus dapat dibuka, dan menolaknya akan membuat layar gagal justru pada keadaan yang paling
sering terjadi.

Kelas objek kerja tetap disaring karena ia menutup kekeliruan yang **tidak menghasilkan
galat**: kunci milik kelas lain mengembalikan baris berkolom PUCL kosong, terbaca persis
seperti klaim yang belum diisi.

### 45.5 Keputusan: "tidak ditemukan" menyebut PORTAL

`ErrClaimNotFound` dijawab **404** dengan pesan yang menyebut entitas yang sedang dipilih.
Penyebab paling mungkin bukan klaim yang hilang melainkan kunci yang benar dibuka pada portal
yang salah — keadaan yang tidak menghasilkan satu pun tanda lain (`R-20`).

### 45.6 Keputusan: delapan isian tanpa kolom tetap DISEBUTKAN

NIK · Business Unit/Seksi · Perihal · Keterangan 1–3 · Tanggal Terima Dokumen PUCL ·
Email LOD · daftar penerimaan dokumen.

Kedelapannya dikirim sebagai `isian_belum_terpetakan` — DATA dari server, bukan teks tetap di
layar — supaya daftarnya menyusut di satu tempat begitu kolomnya ditemukan.

Mereka **tidak** dikirim sebagai isian kosong: isian kosong membuat layar mengira datanya
memang belum diisi, padahal kolomnya yang belum ditemukan. Itu dua hal yang berbeda, dan
hanya yang kedua yang menuntut permintaan ke DBA.

## 46. Layar kerja RCL/PUCL menjadi rute tersendiri (2026-09-24)

### 46.1 Keputusan: rute `/inbox-rcl-pucl/klaim/:referensi`, bukan panel

Di Pega, mengklik Nomor Case menjalankan Open Assignment dan klaimnya terbuka pada tahap alur
kerjanya **untuk dikerjakan**. Panel di bawah tabel menyiratkan pratinjau baris; itu bukan yang
terjadi.

Rute ini TIDAK dipakai modul lain. Enam inbox lain menuju `/view-claim/:referensi` — layar
"View Claim" yang belum dibangun. RCL/PUCL berbeda karena layar tujuannya sudah diketahui:
ketiga rule section-nya diterima 2026-09-24.

**Konsekuensi yang wajib ditangani bersamaan:** berpindah rute melepas komponen antrean, dan
tab serta nomor halaman yang hanya hidup di `useState` akan hilang. Keduanya karena itu pindah
ke **alamat**. Tanpa itu, petugas yang membuka satu klaim dari halaman ketiga tab kedua mendarat
kembali di tab pertama halaman pertama — pada antrean yang dikerjakan berpuluh baris sehari, itu
bukan ketidaknyamanan kecil.

Rentang tanggal **tidak** ikut ke alamat: ia hanya dipakai tombol ekspor dan tidak menyaring
tabel. Menaruhnya di alamat akan menyiratkan ia bagian dari apa yang sedang dilihat.

### 46.2 Keputusan: judul isian mengikuti SECTION, bukan kolom grid

Judul grid dan judul section berbeda untuk isian yang sama, dan keduanya sama-sama masuk akal
dibaca — sehingga tertukarnya tidak menghasilkan satu pun galat:

| Isian | Judul grid | **Judul section** |
|---|---|---|
| `KOMENTARANALISATOR_1` | Deskripsi Analyst | **Catatan dari Analyst** |
| `KOMENTARPUCL_1` | Komentar PUCL | **Catatan untuk Analyst** |
| `RCL_PUCL_1` | Status RCL/PUCL | **Status RCL / PUCL / MSIG** |

Yang dibawa adalah judul section (`D-13`). Dua yang pertama berpasangan — yang satu catatan
Analyst untuk PUCL, yang satu balasan PUCL untuk Analyst — sehingga menyamakannya menukar arah
percakapannya, bukan sekadar mengganti kata.

Satu label sengaja dibawa meski tidak sejalan dengan propertinya: **"No Kontrak"** menempel pada
`.ClaimData.PUCLStatus.NIK`. Yang dibawa label-nya, bukan nama propertinya.

### 46.3 Keputusan: layar kerja TIDAK membawa isian yang bukan milik section

"Tanggal Cetak Surat" dan "Tanggal Kirim RCL/PUCL" dihapus dari layar kerja dan dari kueri
`detail`. Keduanya kolom **grid**; section tidak memuat satu pun dari keduanya.

Keduanya ada di tabel yang sama dan sudah diambil ketiga kueri daftar, sehingga membawanya
nyaris tanpa biaya — dan itulah sebabnya ia perlu ditahan uji
(`TestDetailCarriesOnlyWhatTheSectionDraws`). Aturannya: layar kerja menggambar apa yang
digambar section-nya, bukan apa yang kebetulan mudah diambil.

`detailColumns` karena itu turun dari 12 menjadi **10 alias**.

### 46.4 Keputusan: isian tanpa kolom digambar DI TEMPATNYA, dengan penanda tersendiri

Sembilan dari tujuh belas isian tidak punya kolom yang dapat ditemukan di export. Tiga
perlakuan mungkin, dan dua buruk:

| Perlakuan | Akibat |
|---|---|
| Dihilangkan | Layar tampak setara padahal tidak |
| Digambar sebagai sel kosong | Kolom yang belum ditemukan tidak dapat dibedakan dari data yang memang belum diisi |
| **Digambar dengan penanda `belum terpetakan`** | Dipakai |

Bedanya bukan kosmetik: isian kosong urusan **petugas**, isian belum terpetakan urusan **DBA**.
Tanda pisah `—` sudah dipakai untuk yang pertama, sehingga yang kedua wajib berbeda.

Grid "Tanggal terima Dokumen" digambar sebagai tabel berkerangka dua kolom, bukan diganti satu
isian tanggal — menggantinya menyembunyikan bahwa layar lama menerima BANYAK tanggal.

### 46.5 Keputusan: ketiga tombol digambar, seluruhnya mati, alasannya di sebelahnya

"Cetak", "Unggah Dokumen", dan "Kirim ke PIC Teknik" seluruhnya MENULIS, dan tabelnya masih
dimiliki Pega selama masa paralel (`P-1`).

Tombol yang dihilangkan menyembunyikan bahwa tindakannya ada; tombol yang hidup akan menulis ke
tabel yang bukan miliknya. Alasannya ditulis **di sebelah tombol**, bukan di balik pesan yang
baru muncul setelah ditekan — tombol mati tanpa keterangan terbaca sebagai kerusakan, dan
petugas akan menekannya berulang kali.

## 47. Pencabutan perbaikan `UP`, dan sifat isian clipboard (2026-09-24)

### 47.1 Keputusan: perbaikan `UP` DICABUT — ia direplikasi apa adanya

§45.2 memutuskan UP diperbaiki menjadi `SumTSI`. **Work Owner meralatnya pada hari yang
sama:** kolom UP memang berisi `ObjectName`, sama dengan Nama Peserta.

Seluruh jejak perbaikan itu dibongkar — subkueri, alias, isian penyimpanan memori, kedua uji,
butir selisih terencana, keterangan layar, dan petunjuk `-periksa`. `P-5` berlaku apa adanya.

**Dua uji dipasang sebagai gantinya, bukan satu.** Yang di memori menuntut UP dan Nama Peserta
**sama**; yang di kueri menuntut `T_CLAIM_OBJECTCOVERAGE` **tidak disentuh sama sekali**.
Isian ini sudah dua kali terbaca sebagai cacat; dua lapis penahan sepadan dengan itu.

**Catatan cara kerja, dan ini yang sebenarnya perlu diperbaiki.** Cacatnya saya ajukan sebagai
pertanyaan — itu benar. Tetapi pertanyaannya membawa kesimpulan: dokumen sudah menyebutnya
*"sumber yang SALAH"* dan kelas `R-19` sebelum ada yang mengonfirmasi, dan sumber penggantinya
sudah saya siapkan. Pertanyaan yang membawa kesimpulan mengundang jawaban yang mengiyakan.

Yang seharusnya ditanyakan: *"kolom bernama UP berisi nama objek — apakah itu memang yang
dikehendaki?"*

### 47.2 Keputusan: kesembilan isian dinyatakan sebagai properti CLIPBOARD, bukan kolom hilang

Work Owner menjelaskan kesembilannya diambil dari clipboard Pega
(`.ClaimData.PUCLStatus.NIK` dan seterusnya). Properti clipboard yang tidak dioptimasi tidak
punya kolom sendiri.

Itu mengubah tiga hal sekaligus:

| Hal | Sebelum | Sesudah |
|---|---|---|
| Sebab | "kolom belum ditemukan" | "tidak ada kolomnya" |
| Pemilik pertanyaan | DBA | **Tim Pega + Work Owner** — apakah propertinya akan diekspos |
| Penanda di layar | `belum terpetakan`, amber | `di clipboard Pega`, abu-abu miring |

Warnanya ikut berubah dengan sengaja. Amber menyiratkan pekerjaan yang tertunda; ini keadaan
yang dinyatakan. Yang tidak berubah adalah pemisahannya dari sel kosong — dan alasannya kini
lebih tajam: sel kosong berarti **belum diisi**, "di clipboard Pega" berarti **ada nilainya,
tetapi tidak terbaca dari tabel**.

## 48. Layar digambar selalu, dan `/api` tidak lagi jatuh ke SPA (2026-09-24)

### 48.1 Keputusan: `/api` yang tidak terdaftar dijawab JSON 404

Sebelumnya `r.NotFound(spa(...))` berlaku juga di dalam `/api`, sehingga alamat API yang salah
dijawab `index.html` dengan status **200**. Bagi peramban itu jawaban yang berhasil; bagi klien
API ia badan yang tidak dapat diurai — dan `callAPI` mengubahnya menjadi `null`, bukan galat.

Gejalanya bukan pesan yang salah melainkan **layar kosong tanpa petunjuk**, dan itu berlaku
untuk setiap modul, bukan hanya RCL/PUCL.

`NotFound` dan `MethodNotAllowed` dipisah: alamat yang salah dan metode yang salah menuntut
perbaikan yang berbeda.

**Ini kode bersama.** Dikerjakan karena ia sebab langsung laporan Work Owner, perbaikannya
terkurung di satu tempat, dan seluruh 138 paket uji tetap lulus — tidak ada modul yang
bergantung pada jawaban HTML untuk alamat API. Yang tidak ikut berubah, dan diuji supaya tetap
begitu: rute halaman milik router peramban.

### 48.2 Keputusan: layar kerja digambar SELALU, keadaannya dinyatakan di atasnya

Permintaan Work Owner: *"kalau datanya tidak bisa diambil, buat kosong aja dulu"*.

Pilihan yang diambil bukan sekadar menurutinya. Versi sebelumnya **mengganti** seluruh badan
halaman dengan pesan memuat atau kotak galat, sehingga satu keadaan yang tidak terduga —
`data` bernilai `null` — menghasilkan halaman yang tidak menggambar apa pun.

Sekarang: kerangka selalu tergambar, keterangan keadaan berada **di atasnya**. Empat keadaan,
dan keempatnya punya tampilan — termasuk yang dulu tidak ada:

| Keadaan | Tampilan |
|---|---|
| memuat | teks + kerangka |
| galat | kotak galat + kerangka |
| **kosong** (`data` `null`) | "Isi klaim tidak terbaca" + **alamat yang harus diperiksa** + kerangka |
| siap | kerangka terisi |

Alasannya bukan selera: bentuk layar ini tidak bergantung pada data — ia tetap kedua bagian
dengan isian yang sama, dan sembilan di antaranya memang tidak pernah terisi (§47.2).

## 49. Archive Dokumen Klaim (2026-09-25)

Modul `MENU_ID 77`, pengganti harness `PNCArchiveDokumen`. Rute layar
`/archive-dokumen-klaim`; paket backend `internal/archivedokumenklaim`, folder frontend
`src/modules/archive-dokumen-klaim` (`D-81`).

### 49.1 Pertanyaan konfirmasi dan jawabannya

Tiga hal ditanyakan sebelum satu baris kode ditulis, karena jawabannya mengubah pekerjaan
secara berarti — bukan memilih di antara default yang setara.

| Pertanyaan | Jawaban Work Owner |
|---|---|
| Lingkup: satu bagian, dua, atau ketiganya? | **"seperti aplikasi PEGA"** — ketiganya |
| Pemilih Kode Filling, yang masternya hilang dari export | **"samakan dengan PEGA"** |
| Penomoran `ID_ARCHIVE` yang memakai `max(ID)+1` | **"Replikasi apa adanya (P-5 murni)"** |

Jawaban ketiga **menolak usul saya** untuk memperbaikinya sebagai selisih terencana.
Keputusan dihormati dan dijalankan; yang saya kerjakan sebagai gantinya adalah membuat
akibatnya terbaca di tiga tempat, bukan tersembunyi di satu.

### 49.2 Jawaban "samakan dengan PEGA" pada hal yang tidak ada di Pega

Pemilih Kode Filling digambar oleh `SecCariKodeArchiveDoc`; yang mengisinya activity
`SetKodeandSearchArchiveDoc`, dan **activity itu tidak ada di export** (`R-16`). Tidak ada
tabel master kode arsip di seluruh 2.634 berkas.

Jadi "samakan dengan PEGA" hanya dapat dipenuhi pada **bentuknya**. Keputusan:

- Bentuknya ditiru: tombol "Pilih Kode", dialog berisi pencarian dan daftar, tombol Pilih.
- Sumber datanya **direkonstruksi** dari kode filling yang sudah pernah dipakai baris
  arsip.
- Kolom "Desc Archive" **tidak digambar** — isinya berasal dari master yang tidak ada.
  Penggantinya jumlah pemakaian.
- Isian Kode Filling **tetap dapat diketik langsung**, karena layar lama pun menyediakan
  "Input Kode" dan "Generated Kode".
- Layar **menyatakan** asal daftarnya. Tanpa itu, pengguna akan menyimpulkan kode barunya
  ditolak.

Alasan rekonstruksi ini dipilih alih-alih menunggu artefaknya: kedua tombol pembuat kode
membuktikan kode memang dibuat **dari layar itu**, bukan dipilih dari master tetap — jadi
kumpulan kode yang sudah dipakai adalah pendekatan terdekat yang datanya benar-benar ada.

### 49.3 Penomoran ID direplikasi, dan cacatnya disebut terang

`Database/INSERTDATASFILLINGARCHIVE.prc` memberi nomor dengan `max(ID_ARCHIVE)+1`. Dua
penyimpanan bersamaan membaca angka yang sama dan yang kedua menimpa baris yang pertama —
tanpa galat, tanpa jejak.

Keputusan Work Owner: **replikasi**. Yang dikerjakan:

| Tempat | Isi |
|---|---|
| kueri `next_archive_id` | cacatnya ditulis lengkap di komentar, beserta siapa yang memutuskannya |
| komentar `Repo.Save` | menyatakan transaksinya **tidak** menutup celah itu |
| `permintaan-artefak-pega.md` | pertanyaannya diajukan kembali, terbuka |

Kedua cabang prosedurnya disatukan menjadi `NVL(MAX(ID_ARCHIVE),0)+1`. Ia menghasilkan
angka yang **sama persis** pada kedua cabang, termasuk saat tabel kosong — jadi
penyederhanaan ini bukan perubahan perilaku.

### 49.4 Tiga kolom yang ditulis meski `.prc` tidak menulisnya

Prosedur di export menerima **15 parameter**; pemanggilnya mengirim **17**. Prosedur itu
revisi yang lebih tua, dan yang diikuti adalah pemanggilnya.

| Kolom | Keputusan | Alasan |
|---|---|---|
| `GROUPPANEL` | **ditulis** | ia menentukan siapa melihat barisnya di daftar kirim ke cabang |
| `KODECABANG` | **ditulis** dari kode cabang pemanggil | pembacaan paling lurus atas `TempCabang.KodeCabang`; ditandai sebagai **simpulan** |
| `CABANGSTATUS` | **ditulis `'0'` eksplisit** | prosedurnya tidak menulisnya sama sekali, dan DDL-nya tidak ada (`R-08`) — bila bawaannya bukan `'0'`, berkas baru tidak akan pernah muncul di daftar pengiriman |

Yang ketiga adalah **perbedaan yang disengaja**, dan ia memperbaiki alur yang putus, bukan
mengubah aturan bisnis.

### 49.5 Perbedaan lain terhadap sistem lama, dan alasannya

| # | Perbedaan | Sifat |
|---|---|---|
| 1 | Parameter binding menggantikan `{ASIS:...}` | menutup celah injeksi; tidak dikecualikan oleh "replikasi apa adanya" |
| 2 | Paginasi + `ORDER BY ARCHIVE_ID DESC` | **perubahan perilaku yang disadari** (`09-DATABASE-STRATEGY.md` §6.3) |
| 3 | Kata kunci dibesarkan hurufnya di kedua sisi | **memperbaiki**: nomor klaim huruf kecil dulu tidak pernah cocok |
| 4 | Nama dokumen lewat `LEFT JOIN`, bukan 2 kueri per baris | hasil identik; menghapus kueri di dalam perulangan |
| 5 | `GROUPPANEL IS NULL` **diloloskan** saringan lini | `not in (...)` Oracle tidak meloloskan NULL — berkas tanpa lini akan hilang dari daftar tanpa tanda |
| 6 | `TGLKIRIMDOK` **diisi** saat pengiriman | sistem lama menampilkan kolomnya tetapi tidak pernah mengisinya |
| 7 | Dua `UPDATE` pengiriman disatukan menjadi satu | menutup keadaan "jawaban tersimpan tetapi status belum" |
| 8 | Pemeriksaan "sudah pernah dikirim" | sistem lama tidak punya; menekan tombol dua kali mengirim dua kali |
| 9 | Validasi formulir | sistem lama tidak memeriksa satu isian pun |
| 10 | `ErrMsg` berbasis teks tidak dibawa | `D-68` |

Butir 5, 6, 8, dan 9 adalah **penambahan**, bukan penyalinan. Keempatnya disebut di sini
supaya tidak terbaca sebagai perbaikan diam-diam.

Yang **tidak** diubah meski menggoda: pencocokan kata kunci tetap `=`, bukan `LIKE`.
Mengubahnya memunculkan baris yang dulu tidak pernah muncul, dan pada tabel arsip itu
tidak dapat ditarik kembali.

### 49.6 Saringan lini bisnis direplikasi meski tampak terbalik

`OperatorID.pyPosition` menentukan lini yang **disembunyikan**: `PA` menyembunyikan Group
Panel `002` (Personal Accident), `TRAVEL` menyembunyikan `005` (Travel).

Dipasangkan ulang dengan posisi byte sebelum dipercaya, dan pasangannya benar. Direplikasi
(`P-5` murni), **diuji** supaya tetap terlihat sebagai keputusan, dan **diumumkan di
layar** supaya berkas yang hilang dari daftar tidak dilaporkan berulang kali sebagai
kerusakan modul.

Satu hal **diperbaiki**: perbandingan jabatannya tidak lagi peka besar-kecil huruf. Sistem
lama memakai `==` apa adanya, sehingga jabatan yang tersimpan huruf kecil jatuh ke cabang
"tanpa saringan" dan petugasnya melihat seluruh lini. Alasannya sama dengan penormalan
kapitalisasi peran pada `D-58`.

### 49.7 Alamat layanan Arsip adalah DATA, bukan konfigurasi

Alamatnya dibaca dari `POOLDATA.GCNM_CONNECT_REST` lewat seam `ServiceCatalog` yang sudah
dipakai provider HCC/HCQ dan direktori pegawai — dengan `TYPESERVICE` baru
**`ARCHIVE-INJECT`**.

Tiga akibat: perpindahan endpoint menjadi pekerjaan DBA, tiap portal boleh punya alamat
Arsip sendiri, dan **hostname produksi tidak masuk repository** (`D-69`).

**Barisnya belum ada.** Sampai ia masuk, pengiriman gagal dengan **503** dan pesan yang
menyebut tepat apa yang kurang. Di mode tanpa basis data, perekam dipakai dan jawabannya
**menyatakan terang** bahwa berkasnya tidak dikirim.

### 49.8 Bentuk API

| Rute | Metode | Isi |
|---|---|---|
| `/api/arsip-dokumen/buka` | GET | isi dropdown + cakupan lini pemanggil |
| `/api/arsip-dokumen` | GET | grid ARCHIVE FILE KLAIM, berhalaman |
| `/api/arsip-dokumen` | POST | simpan — `id` nol berarti baris baru |
| `/api/arsip-dokumen/klaim` | GET | calon klaim |
| `/api/arsip-dokumen/kode-filling` | GET | isi pemilih Kode Filling |
| `/api/arsip-dokumen/kirim-cabang` | GET | berkas yang belum dikirim |
| `/api/arsip-dokumen/{id}/kirim-cabang` | POST | kirim satu berkas |

`buka` memakai **GET**, berbeda dari View History Claim yang POST: membuka layar ini tidak
mengubah apa pun dan tidak memakai jatah, sehingga mengulangnya aman.

Simpan memakai **satu** endpoint untuk sisip dan ubah. Layar lama punya satu tombol "Save
To Archive" yang melayani keduanya; memisahkannya memaksa layar menebak lebih dulu, dan
tebakan yang salah menyisipkan baris ganda alih-alih mengubah yang ada.

### 49.9 Ketiga bagian layar menjadi tab, bukan tiga rute

Bagian yang terbuka adalah keadaan **di dalam** layar. Memberi masing-masing alamat
sendiri akan menjanjikan tautan-dalam yang isinya bergantung pada pencarian yang belum
dijalankan.

### 49.10 Yang TIDAK dikerjakan

- **Tombol hapus.** Layar lama tidak punya, dan `D-66` menetapkan soft delete menyeluruh —
  keduanya menuntut keputusan tersendiri.
- **Kunci idempotensi** pada pengiriman (`10-API-STRATEGY.md` §7): kontrak sistem Arsip
  tidak menyediakan tempat membawanya. Yang menahan pengiriman ganda adalah pemeriksaan
  status di server.
- **Pemeriksaan peran** (`TKT-F3-005`): belum ada di modul mana pun.

## 50. Archive Dokumen Klaim — tiga keputusan setelah selisih UI ditemukan (2026-09-25)

Work Owner menanyakan perbedaan UI terhadap Pega. Pemeriksaan ulang memunculkan empat
selisih yang belum tercatat, dan tiga di antaranya diputuskan.

### 50.1 Pertanyaan konfirmasi dan jawabannya

| Pertanyaan | Jawaban Work Owner |
|---|---|
| Simpan di Pega ikut mengirim ke layanan Arsip, tanpa menandai status — sehingga berkasnya terkirim dua kali. Bagaimana? | **"Samakan dengan Pega — Simpan ikut mengirim"** |
| Empat tombol Pega belum dibangun. Mana yang dikejar? | **"Export To Excel"** |
| Tombol Ubah adalah tambahan saya; Pega tidak punya jalur sunting. Dipertahankan? | **"Hapus — samakan dengan Pega"** |

Ketiganya menolak rekomendasi saya. Semuanya dijalankan apa adanya.

### 50.2 Simpan ikut mengirim, dan pengiriman gandanya direplikasi

Sumbernya `Activity/SaveAttachArchiveToDatabase-Act.xml` langkah 5, yang memanggil
`SendDataArchiveDOcumentByService` tepat setelah prosedur penyisipannya selesai.

**Yang menentukan bentuk implementasinya** adalah apa yang TIDAK dilakukan jalur itu:
`UpdateDataArchiveKlaimSetelahService` hanya menyentuh `KODESERVICE`, `NOTESERVICE`, dan
`HITARCHIVE` — bukan `CABANGSTATUS`. Hanya jalur Dokument Cabang yang menandainya.

Karena itu seam `Repo` punya **dua** operasi penyimpanan jawaban, dan keduanya wajib tetap
berbeda:

| Operasi | Menandai `CABANGSTATUS`? | Dipakai jalur |
|---|---|---|
| `StoreReceipt` | **tidak** | Simpan |
| `MarkSent` | ya, `'1'` | Kirim ke Cabang |

Menyatukannya menjadi satu operasi akan menghapus pengiriman kedua — perubahan perilaku,
bukan pembersihan. `TestKeduaPenyimpananJawabanBerbedaHanyaPadaCabangStatus` menahannya.

**Kegagalan mengirim TIDAK menggagalkan penyimpanan.** Barisnya sudah tersimpan — di
sistem lama pun prosedurnya `COMMIT` sebelum langkah pengiriman dijalankan. Mengembalikan
galat akan membuat layar melaporkan "gagal menyimpan" atas berkas yang sebenarnya ADA, dan
pengguna akan menyimpannya lagi: baris ganda yang tidak dapat dihapus dari layar ini.

Yang dikerjakan sebagai gantinya: `SaveResponse` membawa `terkirim`, `kode_layanan`,
`catatan_layanan`, dan `galat_kirim`; pesannya menyebutkan keadaannya; dan berkasnya tetap
berada di daftar Dokument Cabang sehingga pengirimannya dapat diulang dari sana.

### 50.3 Export To Excel dibangun sebagai CSV

`pxConvertResultsToCSV` pada langkah 17 activity pencarian. Labelnya menyebut Excel karena
berkas CSV memang dibuka dengan Excel — label dipertahankan (`D-13`), mekanismenya
mengikuti aslinya.

Membuat `.xlsx` sungguhan menambah satu dependensi DAN mengubah bentuk keluaran terhadap
sistem lama; keduanya keputusan tersendiri yang tidak diambil di sini.

| Hal | Ketetapan | Alasan |
|---|---|---|
| Bentuk | CSV | mengikuti `pxConvertResultsToCSV` |
| Penyaring | sama persis dengan daftar yang tampil | ekspor yang mengabaikan penyaring menghasilkan berkas yang tidak dapat dicocokkan dengan apa pun di layar |
| Aliran | potong demi potong, 100 baris | memori tetap datar berapa pun jumlah barisnya |
| Batas | 50.000 baris, **dengan baris penanda** | `ADR-0011` belum menjawab berapa yang wajib dilayani; pemotongan senyap adalah cacat sistem lama |
| Bentuk tanggal | `YYYY-MM-DD` | berkas ini diurutkan di Excel; `dd/mm/yyyy` terurut sebagai teks yang salah |
| Galat | dijawab **sebelum** satu byte pun ditulis | setelah header terkirim, galat tidak dapat lagi dijawab sebagai JSON |

### 50.4 Tombol Ubah dicabut; gridnya baca-saja

Penanda `flags` pada prosedur simpan hanya pernah disetel `"insert"` di seluruh export.
Cabang `update`-nya ada di prosedur tetapi **tidak pernah dipanggil dari layar ini**.

Yang dicabut hanyalah **jalur layarnya**. Backend tetap menerima `id` bukan nol pada
endpoint simpan, karena prosedurnya memang punya cabang itu dan tombol "Update Box" yang
ditunda kemungkinan memakainya. Keadaan itu ditulis terang di `types.ts` — endpoint yang
tidak punya pemanggil adalah hal yang harus disebut, bukan ditinggalkan untuk ditemukan.

### 50.5 Empat selisih yang TETAP terbuka

| Selisih | Status |
|---|---|
| `Tambah`, `Update Box`, `Transfer To Pusat` | **wiring tidak dapat ditelusuri** dari export; ketujuh activity yang dipanggil section sudah dikenali seluruhnya, dan tidak ada yang kedelapan |
| `Input Kode`, `Generated Kode` pada pemilih kode | menunggu aturan pembentukan kode dari pemilik bisnis |
| `Refresh` | tidak dibangun sebagai tombol — tabel memuat ulang sendiri setelah menyimpan dan mengirim |
| Kolom "Desc Archive" pada pemilih kode | masternya tidak ada; diganti jumlah pemakaian |

Ketiga yang pertama tercatat di `permintaan-artefak-pega.md` §4.

### 50.6 Catatan atas cara memverifikasi

`tsc --noEmit | head -30` **menyembunyikan kegagalan**: exit code yang terbaca milik
`head`. Dua galat tipe lolos karenanya pada sesi sebelumnya.

Sejak keputusan ini: `tsc` dijalankan tanpa pipe, dan exit code-nya dicetak eksplisit.
Aturan yang sama berlaku untuk setiap alat verifikasi — **exit code pipeline bukan exit
code alatnya.**
