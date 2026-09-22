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
