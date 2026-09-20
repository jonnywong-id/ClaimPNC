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

## 17. Master Status Progres 2 (2026-09-19, sesi kesembilan)

### 17.1 Kenapa tingkat 2 menempel pada paket domain yang sama

`internal/masterstatusprogres` kini memuat dua tingkat, bukan dua paket. Alasannya bukan kedekatan
nama, melainkan ketergantungan yang tidak dapat dipisahkan: **penambahan tingkat 2 tidak dapat
dilakukan tanpa membaca tingkat 1 lebih dulu** — untuk memastikan induknya ada, dan untuk menyalin
namanya ke kolom `STS_PROGRESS1`.

Memisahkannya menjadi dua paket akan membuat paket tingkat 2 mengimpor paket tingkat 1 untuk hal
yang berada di inti operasinya sendiri. Satu paket, dua tingkat, dua seam (`Repo` dan `Repo2`).

Yang **tidak** disatukan adalah layanannya: `Service` dan `Service2` terpisah meski sepaket, karena
keduanya memilih penyimpanan yang berbeda. Satu layanan yang melayani dua tabel akan menerima dua
pemilih repo dan bercabang di setiap method.

### 17.2 Ada rute ubah — perilaku baru; tetap tidak ada rute hapus

> **Keputusan ini dibalik dalam satu hari.** Ditanyakan ulang pada 2026-09-20, Work Owner mula-mula
> memilih **tidak menambahkan** fitur ubah. Beberapa saat kemudian ia bertanya *"Dimana button
> ubah? kenapa tidak muncul?"*, lalu memilih **menambahkannya, mencakup nama dan induk**.
> Riwayat lengkapnya di `catatan-pengembangan.md` §16.11 dan §16.13.
>
> Pembalikan itu bukan inkonsistensi: pertanyaan pertama adalah *"apakah mereplikasi tombol
> Pega"*, pertanyaan kedua *"apakah baris yang salah ketik dapat diperbaiki"*. Jawaban yang benar
> untuk keduanya memang berlawanan.

**Sistem lama tidak punya penyuntingan yang bekerja.** Nol `UPDATE`, nol `DELETE` terhadap
`POOLDATA.GCNM_MST_PROGRESS` di seluruh export. Tombol "Update" di layar lama menulis ke **tabel
lain** (`GCNM_PROGRESS_CLAIM`) dengan dua page klipboard yang tidak pernah diisi; rinciannya di
`catatan-pengembangan.md` §16.4.

**Yang ditambahkan bukan jalur itu.** Jalur Pega, bila kedua page klipboardnya kebetulan terisi
sisa nilai dari layar lain dalam sesi yang sama, akan menimpa catatan progres sebuah klaim dengan
isian layar master. Yang dibuat adalah `UPDATE` yang benar ke tabel master.

| Hal | Ketetapan |
|---|---|
| Yang dapat diubah | **nama** dan **induk** |
| `ID_MST` | **tidak pernah** ikut di-`SET` — ia dirujuk `GCNM_PROGRESS_CLAIM.STATUS_PROGRESS2` pada data klaim berjalan |
| `STS_PROGRESS1` | **ditulis ulang server** dari baris induk yang baru; ia salinan, bukan rujukan |
| `TIPE` | tidak disentuh — tidak pernah ditulis sistem lama maupun sistem ini |
| Letak ID | **di jalur**, bukan di badan — badan yang memuat ID membuka kemungkinan satu permintaan menyebut dua ID |

**Ini selisih pada gerbang 1, dan dinyatakan di muka** (`D-54`): sistem lama tidak mengubah baris
master, sistem baru mengubahnya.

**Rute hapus tetap tidak ada.** Ia tidak pernah ada di sistem lama, tabelnya tidak punya kolom
penanda terhapus yang dapat dipakai `D-66`, dan barisnya dirujuk data klaim yang sudah berjalan.
Satu uji menjaganya: `TestOnlyPutIsRegistered2`.

### 17.3 ID tingkat 2 tanpa awalan nol

`FormatID2` menghasilkan angka polos; `FormatID` tingkat 1 menghasilkan `"0" + nomor`. Ini
perbedaan nyata di sistem lama, terverifikasi dua kali:

```
tingkat 1  Activity/InsertMstStatusProgress1_act  TempInputStatus.CaseID  := "0"+.City
tingkat 2  Activity/InsertMstStatusProgress2_act  TempInputStatus2.CaseID := .District
```

Dikuatkan data yang beredar: `GetDataProgressClaim-SQL.xml` menyaring
`STATUS_PROGRESS2 not in ('2','24','60')` — angka polos.

Memakai `FormatID` tingkat 1 di sini akan menerbitkan ID yang **tidak dapat dicocokkan** dengan
baris `GCNM_PROGRESS_CLAIM.STATUS_PROGRESS2` yang sudah ada — dan tidak ada galat apa pun yang
muncul.

Akibat sampingan yang menguntungkan: tanpa awalan, tingkat 2 luput dari persoalan urutan teks yang
menghinggapi tingkat 1 (`"010"` mendahului `"09"`).

### 17.4 `ErrParentNotFound` dijawab 422 pada isian, bukan 404

Keduanya berarti "tidak ditemukan", dan keduanya dipetakan berbeda dengan sengaja:

| Galat | Status | Alasan |
|---|---|---|
| `ErrNotFound` | 404 | baris yang **diminta** pemanggil tidak ada |
| `ErrParentNotFound` | **422** dengan `kolom: "id_induk"` | **isian** yang dipilih pengguna sudah tidak ada — ia dapat memperbaikinya |

404 akan membuat layar mengatakan barisnya sendiri hilang, padahal yang hilang adalah induk yang
dipilih di dropdown. Nama isiannya `id_induk`, sama persis dengan field JSON yang dikirim layar,
supaya keterangan galat menempel di tempat yang benar tanpa penerjemahan.

Pemetaannya ditaruh **sebelum** `ErrNotFound` di `mapError`, karena urutan `switch` menentukan.

Sebelum sesi ini galat itu **tidak dipetakan sama sekali** — ia jatuh ke penulis galat bersama dan
dijawab **500**. Induk yang sudah dihapus petugas lain karena itu tampil sebagai kerusakan sistem,
bukan sebagai isian yang perlu dipilih ulang. `TestCreate2RejectsMissingParent` menjaga itu tidak
kembali.

### 17.5 Seluruh rute tingkat 2 menuntut portal — termasuk daftar induknya

Berbeda dari tingkat 1, yang menyisakan `/master/posisi-klaim` di luar pemeriksaan portal karena
keempat posisi itu daftar milik aplikasi.

Di tingkat 2 tidak ada satu pun rute yang isinya milik aplikasi: **daftar induk pun dibaca dari
basis data entitas**. Dua entitas punya Status Progres 1 yang berbeda, dan menyajikan daftar satu
entitas kepada entitas lain adalah kebocoran yang justru dicegah `R-20`.

Akibat lanjutannya di frontend: `useProgressStatus2ParentList` **tidak** diberi `staleTime` panjang
seperti `useClaimPositionList`. Isinya dapat berubah kapan saja lewat layar Master Status Progres 1.

### 17.6 Pemilih repo memori bertanya ke pemilih tingkat 1, tidak memeriksa sendiri

`progressStatus2SelectorMemory` menerima `RepoSelector` tingkat 1 sebagai bahan, lalu menanyakan
portalnya ke sana. Portal yang ditolak di tingkat 1 ditolak di tingkat 2 dengan galat yang sama
persis.

Dua pemeriksaan terpisah atas hal yang sama akan berbeda begitu salah satunya disunting — dan yang
dipertaruhkan pada `R-20` bukan pesan galat, melainkan pemisahan data antar badan hukum.

Repo tingkat 1 yang dikembalikannya **dipakai langsung** sebagai induk, bukan disalin menjadi
daftar terpisah. Kalau disalin, kedua adapter akan berbeda pada hal yang justru paling ingin diuji:
induk yang baru ditambahkan lewat layar tingkat 1 tidak akan terlihat di dropdown tingkat 2.
`TestService2SeesNewlyAddedParent` menjaga pernyataan itu tetap benar.

### 17.7 `TIPE` dibaca, ditampilkan, tidak pernah ditulis

Artinya tidak diketahui: di seluruh export ia hanya muncul pada dua `SELECT`, tanpa satu pun
`INSERT`, `UPDATE`, maupun penyaring yang memakainya. DDL tabelnya belum diterima (`R-08`),
sehingga tidak ada pula daftar nilai sahnya.

Tiga pilihan ditimbang, dan yang ketiga diambil:

| Pilihan | Akibat |
|---|---|
| Tidak dibaca sama sekali | nilai yang benar-benar tersimpan tidak terlihat petugas |
| Ditulis dengan nilai tebakan | mengarang; dan `NULL` pun keputusan yang belum diputuskan siapa pun |
| **Dibaca dan dikirim, tidak pernah ditulis** | baris lama tidak kehilangan nilainya hanya karena disentuh layar baru |

> **Dikoreksi setelah header grid Pega dibaca.** Baris ketiga semula berbunyi *"dibaca dan
> **ditampilkan**"*, dan layar ini sempat memuat kolom `Tipe`. Pega **tidak punya kolom itu** —
> `DistrictID` terikat ke page `TempUpdateStatus2`, yaitu modal penyuntingan yang tidak dibawa.
> Kolomnya dicabut; pembacaan dan pengirimannya lewat API **tidak** berubah. Rinciannya di
> `catatan-pengembangan.md` §16.10.

`INSERT`-nya karena itu menyebut **empat kolom saja**, persis seperti sistem lama — basis data
mengisi `TIPE` dengan default kolomnya sendiri. `TestInsert2NeverWritesKind` menjaganya.

Di layar, `TIPE` yang kosong ditandai `—` supaya sel kosong tidak terbaca sebagai kegagalan memuat.

### 17.8 Salinan nama induk dipertahankan, beserta cacatnya

`STS_PROGRESS1` menyimpan **salinan** nama induk pada saat baris disimpan, bukan hasil join.
Denormalisasi sistem lama ini dijalankan as-is atas keputusan Work Owner.

Konsekuensinya disadari dan dicatat supaya tidak dikira rancangan: **mengganti nama sebuah Status
Progres 1 tidak memperbarui salinan di baris-baris tingkat 2 yang sudah ada**, sehingga keduanya
dapat berbeda.

Bahwa perbedaan itu benar-benar terjadi di produksi terbaca dari kueri laporan yang membacanya —
`GetDataOutstandingperCabangExport-SQL.xml` memakai
`SELECT id_progress, MAX(sts_progress1) ... GROUP BY id_progress`, dan `MAX` hanya diperlukan bila
baris ber-`id_progress` sama menyimpan nama yang berlainan.

### 17.9 `MAX(ID_MST)+1` diberi `FOR UPDATE` — dan itu bukan perubahan aturan

Di Pega, `NVL(MAX(B.ID_MST),0)+1` dijalankan sebagai kueri lepas, lalu hasilnya dipakai `INSERT`
beberapa langkah kemudian. Di antara keduanya tidak ada apa pun yang menghalangi penambahan lain
masuk lebih dulu — dua petugas yang menambah bersamaan dapat menerima nomor yang sama.

Pembacaan, penurunan nomor, dan penyisipan karena itu berada di dalam **satu operasi repo** dengan
`FOR UPDATE`. Bentuk nomornya tetap sama; yang ditutup hanyalah lubang balapan pada cara nomor itu
diturunkan.

`NVL` sendiri tidak dibawa — ia diganti pembacaan daftar ID lalu penurunan di Go, sejalan dengan
`D-20`.

### 17.10 Panjang nama 100 adalah asumsi, dan dinyatakan begitu

`MaxNameLength2 = 100` disamakan dengan tingkat 1. Berbeda dari tingkat 1, angka ini **bukan**
ketetapan Work Owner: DDL `POOLDATA.GCNM_MST_PROGRESS` belum ada (`R-08`), dan kedua kolom menyimpan
hal yang sejenis pada tabel yang sekerabat.

Bila basis data ternyata menerima lebih pendek, penolakannya datang dari basis data dan terbaca
sebagai galat teknis, bukan sebagai pesan yang menuntun pengguna. Itu kekurangan yang diterima
sampai DDL-nya tiba — **bukan** alasan menebak angka yang lebih longgar, karena menebak longgar
justru memindahkan kegagalannya ke tempat yang lebih sulit dibaca.

Angka yang sama diulang di `ProgressStatus2Form.tsx`. Bila berubah, kedua tempat harus ikut berubah;
keduanya saling menyebut di doc comment-nya.

### 17.11 Yang belum dikerjakan

| Hal | Alasan |
|---|---|
| Isi `SampleList2()` diganti data sebenarnya | isi `GCNM_MST_PROGRESS` tidak ada di export dan belum diminta ke DBA. Isi contohnya **susunan sendiri**, ditandai jelas, dan tidak boleh dipakai sebagai dasar uji kesetaraan gerbang 1 |
| Panjang kolom yang sebenarnya | menunggu DDL — `R-08` |
| Arti kolom `TIPE` | menunggu DDL dan konfirmasi Work Owner |
| ~~Penyuntingan baris~~ | ✅ **selesai 2026-09-20** atas keputusan Work Owner — nama dan induk; §17.2 |
| Penghapusan baris | tidak ada di sistem lama, tidak ada kolom penanda terhapus, dan barisnya dirujuk data klaim berjalan — §17.2 |
| Kewenangan menulis di produksi | selama Pega masih penulis tabel ini, layar dijalankan modus baca saja (`ADR-0004`, penulis tunggal per tabel) |
| Tabrakan nama tab/tombol di Master Rekening (3 uji merah) | tetap menunggu keputusan Work Owner — §14.4 |

---

## 18. Modul Master Penolakan Klaim (2026-09-19, sesi kesepuluh)

Modul bisnis keempat, mengganti `Harness/PNC_MasterTolakKlaim-Harness.xml` (MENU_ID 25).

### 18.1 Keputusan yang diambil Work Owner pada sesi ini

| # | Perkara | Keputusan |
|---|---|---|
| 1 | Cakupan layar | **Kedua master dibangun**, sesuai Pega: `MST_PENOLAKAN_KLAIM_1`/`_2` untuk Penolakan Klaim, `MST_REJECTED_KOMITE` untuk Penolakan Komite |
| 2 | Cacat `MASTERPENOLAKANKLAIM1` | **Diperbaiki** — induk dipilih dari daftar, boleh tambah baru |
| 3 | Pengubahan mereset status persetujuan | **Dibawa apa adanya** |
| 4 | Kolom persetujuan | **Tampil di grid, baca-saja** |

### 18.2 Satu penyimpangan dari `P-5`, dan alasannya

`P-5` menetapkan perilaku dipertahankan lebih dulu, kecuali untuk perbaikan yang diputuskan
eksplisit. Modul ini punya **tepat satu** perbaikan semacam itu.

**Cacatnya.** `Database/MASTERPENOLAKANKLAIM1.prc` melakukan `INSERT` pada **kedua** cabang
`IF`-nya, tidak pernah `UPDATE`:

```
:8   IF tid_st is null or tid_st='' then
:9       INSERT INTO POOLDATA.MST_PENOLAKAN_KLAIM_1 (ID_ST,NOTE_ST) VALUES (id_mst,tnotest);
:13  elsif tid_st is not null or tid_st!='' then
:14      INSERT INTO POOLDATA.MST_PENOLAKAN_KLAIM_1 (ID_ST,NOTE_ST) VALUES (tid_st,tnotest);
```

`Activity/InsertMasterPenolakanNoteKlaim-Act.xml` memanggilnya pada **setiap** simpan, lalu
memakai ID hasilnya sebagai `ID_ST` baris tingkat 2. Akibatnya:

- setiap penambahan menerbitkan satu baris tingkat 1 baru, walau teksnya sama persis;
- setiap **pengubahan** juga menerbitkannya, dan memindahkan baris tingkat 2 ke induk yang baru —
  meninggalkan induk lama tanpa satu pun yang merujuknya;
- `MST_PENOLAKAN_KLAIM_1` tumbuh sebanyak jumlah penyimpanan, bukan sebanyak jumlah alasan
  penolakan yang sebenarnya ada.

**Yang diputuskan.** Induk **dipilih dari daftar** yang sudah ada; baris tingkat 1 baru lahir
hanya bila pengguna memang meminta yang baru. Jalur "pilih" tidak menulis apa pun ke tabel
tingkat 1.

**Konsekuensi yang diterima, dan dinyatakan di muka supaya tidak ditemukan sebagai kejutan pada
uji kesetaraan gerbang 1:**

| Hal | Sistem lama | Sistem baru |
|---|---|---|
| Menyimpan dengan induk yang sudah ada | `MST_PENOLAKAN_KLAIM_1` bertambah satu baris | tidak bertambah |
| `MST_PENOLAKAN_KLAIM_2.ID_ST` sesudah diubah | selalu menunjuk induk yang **baru dibuat** | menunjuk induk yang **dipilih** |
| Bentuk isian "Status Penolakan 1" | kotak teks bebas | daftar pilihan + opsi tambah baru |

**Baris lama dibiarkan.** Duplikat yang sudah telanjur ada di produksi tidak dibersihkan modul
ini: membersihkannya berarti memutuskan baris mana yang menang, dan baris tingkat 2 yang merujuk
induk yang "kalah" harus dipindahkan. Itu keputusan data yang belum diambil siapa pun. Yang
dikerjakan adalah **menghentikan pertumbuhannya**, dan melaporkan besarnya — mode `-periksa`
menghitung nama tingkat 1 yang kembar dan menyebut angkanya.

### 18.3 Tiga perilaku yang DIREPLIKASI, termasuk yang tampak aneh

| Perilaku | Bukti | Kenapa tidak diperbaiki |
|---|---|---|
| Pengubahan mereset `STATUS` ke `'0'` dan `TANGGALKIRIM` ke waktu sekarang | `MASTERPENOLAKANKLAIM2.prc:14` | Keputusan Work Owner, dan masuk akal secara bisnis: teks yang sudah disetujui tidak boleh berubah diam-diam |
| Ketiga kolom persetujuan **tidak** ikut dibersihkan saat reset | `MASTERPENOLAKANKLAIM2.prc:14` — hanya enam kolom yang di-`SET` | Ia jejak keputusan yang **pernah** ada. Layar menyebutnya demikian alih-alih menyembunyikannya |
| Baris pertama pada `MST_REJECTED_KOMITE` yang kosong bernomor **111** | `INSERTMASTERREJECTEDKOMITE.prc:9` | Tidak ada keterangan apa pun tentang asal angka itu, dan keadaannya hanya terjadi sekali seumur tabel |

### 18.4 Penyaring yang TIDAK dibawa, dan kenapa itu bukan kelalaian

`RDB List/BrowseStatusPenolakanKlaim2-SQL.xml` memuat `{ASIS:MasterCheckerPenolakan.RemakApprove}`
— potongan teks SQL yang disisipkan dari nilai klipboard. Yang mengisinya adalah
`Activity/BrowseStatusPenolakanKlaim_2-Act.xml` langkah 2, dengan nilai `"WHERE STATUS='0'"`.

Langkah itu **berprasyarat `Param.master=="1"`**, dan layar Master Penolakan Klaim **tidak
mengirim parameter itu** — yang mengirimnya adalah layar checker pada Inbox Manager. Dari layar
ini potongan penyaringnya karena itu tetap kosong, dan gridnya menampilkan **seluruh** baris.

Menyalin penyaringnya ke sistem baru akan menyembunyikan baris yang sudah diputuskan: itu
**perubahan perilaku**, bukan replikasi. Saya sempat menuliskannya sebelum membaca
preconditionnya — uji `TestListShowsEveryRowRegardlessOfApproval` yang menjaganya tidak kembali.

Potongan `{ASIS:...}`-nya sendiri tidak dibawa dalam bentuk apa pun. Selain melanggar
§4.3 `08-TECHNICAL-STRATEGY.md` (parameter binding tanpa perkecualian), ia membawa **kebocoran
antarlayar**: nilai yang tertinggal dari layar checker dalam sesi yang sama akan diam-diam
menyaring layar ini.

### 18.5 Keputusan desain

| # | Keputusan | Alasan |
|---|---|---|
| 1 | **Satu paket Go untuk dua master** (`internal/masterpenolakan`) | Satu butir menu, satu layar. Memecahnya berarti dua rakitan dan dua pemilih portal untuk sesuatu yang dilihat pengguna sebagai satu layar |
| 2 | **Dua seam terpisah** di dalamnya — `Repo` dan `RepoKomite` | Tabelnya tidak sekerabat dan tidak punya satu pun kolom yang menghubungkannya. Menyatukan seam-nya akan menyiratkan hubungan yang tidak ada |
| 3 | **Satu seam untuk dua tabel Penolakan Klaim** | Penambahannya satu operasi yang tidak dapat dipecah: induk baru harus lahir bersama anaknya, di dalam transaksi yang sama. Memecahnya menuntut `*sql.Tx` bocor ke luar repo |
| 4 | Nama tipe `RejectionStatus` / `RejectionStatus2` | Mengikuti `ProgressStatus`/`ProgressStatus2` yang bentuknya sama persis, dan mengikuti label layar Pega ("Status Penolakan 1" / "2"). Tidak ada istilah yang dikarang |
| 5 | `TANGGALKIRIM` diisi dari seam `Clock`, bukan `SYSDATE` | Mengikuti preseden `account_insert` pada Master Rekening. Konsekuensinya dicatat — lihat §18.6 |
| 6 | Kolom `USER_INPUT` diisi **login**, bukan NIK | Sistem lama mengisinya `OperatorID.pyUserIdentifier`, sehingga baris lama sudah berisi login. Mengisinya dengan NIK membuat satu kolom memuat dua jenis pengenal yang tidak dapat dibedakan sesudahnya |
| 7 | `ORDER BY ID_ST ASC` ditambah `ID_ND ASC` | Kueri lama tidak menentukan urutan di antara baris ber-`ID_ST` sama, sehingga basis data bebas memulangkannya dalam urutan apa pun. Kunci kedua membuat yang tadinya sembarang menjadi tetap — tidak bertentangan dengan urutan lama |
| 8 | Daftar Status Penolakan 1 diurutkan menurut **nama** | Tidak ada kueri lama yang urutannya harus disamai (`MST_PENOLAKAN_KLAIM_1` tidak muncul di satu pun rule SQL), dan yang dibaca manusia di sana namanya. Pada tabel yang kemungkinan besar penuh nama kembar, ia juga membuat kembarannya berdampingan |
| 9 | Tidak ada `DELETE` pada ketiga tabel | Seluruh export tidak memuat satu pun, layar lama tidak punya tombolnya, dan tidak satu pun tabelnya punya kolom penanda terhapus yang dapat dipakai `D-66` |
| 10 | Aturan wajib-isi dan batas panjang **ditambahkan** | Tidak ada di sistem lama. Mengikuti keputusan yang sama pada Master Status Klaim (2026-09-17) |

### 18.6 Utang teknis yang disadari

| # | Utang | Dampak bila dibiarkan |
|---|---|---|
| 1 | **Panjang maksimum 100 karakter adalah ASUMSI.** DDL ketiga tabel belum ada (`R-08`), dan procedure lama menerima parameternya sebagai `varchar2` tanpa panjang | Bila kolomnya lebih pendek, penolakannya datang dari basis data sebagai galat teknis, bukan pesan yang menuntun pengguna. Angkanya diulang di `RejectionForm.tsx` dan `CommitteeRejectionForm.tsx` — bila berubah, ketiga tempat harus ikut |
| 2 | **`TANGGALKIRIM` UTC versus WIB.** Baris yang ditulis Pega memuat waktu server (WIB); baris yang ditulis aplikasi ini memuat UTC. Keduanya terpaut tujuh jam pada kolom yang sama selama masa paralel | Wujud nyata `R-12` pada tabel ini. Tidak dapat dihindari tanpa melanggar §4.4, dan terbatas pada satu kolom yang hanya ditampilkan — tidak dipakai perhitungan mana pun |
| 3 | **Modal "Master Status Penolakan 1" tidak dibangun.** `Flow Action/Flo_MasterPenolakanKlaim` tidak ada di export (`R-16`) | Fungsinya tercakup opsi "+ Status Penolakan 1 baru…" pada form, tetapi isi modal aslinya tidak diketahui — mungkin ada isian yang terlewat |
| 4 | **Data contoh adalah susunan sendiri.** Isi ketiga tabel tidak ikut dikirim: tidak ada CSV-nya di `Database/` | Tidak boleh dipakai sebagai dasar uji kesetaraan gerbang 1 |
| 5 | **Pemetaan galat masih milik modul.** Sama seperti tiga modul master sebelumnya | Pindah ke tempat bersama begitu `TKT-F1-004` diputuskan |
| 6 | **Nama kunci pelanggaran `kolom`, bukan `field`.** Mengikuti `masterstatusprogres` | Ketidakseragaman yang sudah dikenali; `APIError.violations()` menyatukannya |

### 18.7 Yang sengaja tidak dikerjakan

| Hal | Alasan |
|---|---|
| **Layar checker** (menyetujui / menolak) | Ia `Section/Sec_PenolakanKlaimChecker` pada `UserInbox_Harness`, MENU_ID 58 — modul tersendiri. Keputusan Work Owner: di layar ini keempat kolom persetujuan baca-saja |
| **Membersihkan duplikat tingkat 1 yang sudah ada** | Menuntut keputusan data yang belum diambil siapa pun. Yang dikerjakan adalah menghentikan pertumbuhannya dan melaporkan besarnya |
| **Migrasi skema** | Ketiga tabel adalah tabel **warisan** yang sudah ada; tidak satu pun dibuat aplikasi ini. Yang dibutuhkan hanyalah hak baca-tulis dari DBA |
| **Paginasi tabel** | `DataTable` bersama belum punya; menambahkannya menyentuh `U-2` yang dipakai seluruh layar master |

---

## 19. Modul Master Auto Claim (2026-09-19, sesi kesebelas)

### 19.1 Keputusan yang diambil Work Owner pada sesi ini

Keempatnya diajukan sebagai pertanyaan **sebelum** satu baris kode ditulis, karena keempatnya
menyentuh aturan yang menentukan ke mana uang klaim dikirim.

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | `CLAIM_ALLOWED` ditimpa `"1"` saat tambah tetapi memakai isian pengguna saat ubah — bagaimana di sistem baru? | **Selalu "1", tidak dapat diubah** |
| 2 | `KOMITE` diambil dengan `type_business='BONDING'` tertanam, baris pertama, tanpa `ORDER BY` | **Replikasi apa adanya** |
| 3 | Approve/Reject menulis ulang 13 kolom dari isi form | **Kirim ulang seluruh isian seperti Pega** |
| 4 | `UpdateAutoClaim` tidak menulis `NAMA_PENERIMA` | **Nama penerima tidak bisa diupdate** |

### 19.2 Nama modulnya menyesatkan, dan itu bukan alasan menggantinya

"Master Auto Claim" bukan master klaim. Ia daftar **Sumber Bisnis yang klaimnya boleh dibuat
otomatis**, beserta ke mana ganti ruginya dibayarkan.

Yang membuktikannya bukan penamaan melainkan pemakaian hilirnya — `INISIALID` dicocokkan dengan
`T_GENERAL.SOURCEOFBUSINESS` pada `GetReceiverClaimAsuransiKredit-SQL.xml`.

Namanya **tetap** "Master Auto Claim": itu nama pada `MENU_DESC` MENU_ID 26 dan nama yang dipakai
Work Owner, dan `D-81` menetapkan folder modul dinamai menurut nama yang disebut Work Owner.
Yang dikerjakan adalah menuliskan artinya di doc comment paket, bukan mengarang nama yang lebih
tepat tetapi tidak dikenali siapa pun.

### 19.3 Menyimpan dan memutuskan adalah SATU operasi

Bukan penyederhanaan, melainkan bentuk sistem lama: `Activity/UpdateMstAutoClaim_act` melayani
tiga tombol sekaligus, dibedakan hanya oleh parameternya.

```
tab Master / Reject, tombol Update   stsapprove="0"
tab Komite, tombol Approve           stsapprove="1"
tab Komite, tombol Reject            stsapprove="2"
```

Karena itu **tidak ada** endpoint `/keputusan` terpisah seperti pada Master Rekening — satu
`PUT /api/master/auto-claim/{inisial}` dengan `status` di dalam badannya. Keputusan Work Owner
19.1 #3 mempertahankan bentuk itu.

**Akibat yang mengikuti dan memang dikehendaki:** menyunting baris yang sudah disetujui
**mengembalikannya ke antrean persetujuan**. Persetujuan lama tidak berlaku atas isi yang sudah
berubah — dan itu benar untuk master yang menentukan ke mana uang dikirim.

### 19.4 Bentuk "kirim ulang" dipertahankan; jalur yang merusaknya tidak

Di Pega, isian yang dikirim ulang diambil dari **page form** — dan page form hanya terisi bila
barisnya lebih dulu dimuat lewat tombol Update. Komite yang menekan Approve langsung dari grid
mengirim `CLIENTID`, `CLIENTNAME`, dan `KOMITE` **kosong**, dan kueri `UPDATE` menulis ketiganya
apa adanya.

Dua perubahan menutupnya tanpa mengubah bentuk kontraknya:

| Perubahan | Alasan |
|---|---|
| Kueri **daftar** ikut membaca `CLIENTID` dan `CLIENTNAME` | layar selalu memegang nilai sebenarnya, sehingga pengiriman ulang tidak dapat menghapusnya. Menambah kolom pada `SELECT` tidak mengubah satu baris pun |
| `KOMITE` **tidak pernah** datang dari permintaan | dibaca dari baris tersimpan, ditulis kembali apa adanya. Di Pega ia dapat menghapus penyetujunya sendiri |

Yang direplikasi adalah **hasil yang teramati pada jalur normal**, bukan jalur yang
menghasilkannya.

### 19.5 Pemeriksaan bank dipindahkan ke lapisan aplikasi — dan dibuat lebih kuat

Pega menolak dengan *"Nama bank jangan diketik manual"* bila kode bank dari autocomplete kosong.
Kode itu **tidak pernah disimpan**; tabelnya tidak punya kolomnya.

| Pilihan | Akibat |
|---|---|
| Terima kode bank dari layar | mempercayai peramban atas nilai yang tidak dapat dibaca kembali; tombol Approve menjadi bergantung padanya |
| **Cocokkan NAMA bank ke `GENERAL.LST_BANK_GROUP`** | maksud yang sama, tidak dapat ditipu kode karangan |

Karena pencocokannya menuntut pembacaan basis data, ia tidak dapat berada di domain. Ia ada di
`usecase.ensureBankKnown`, dan galatnya dibungkus `masterautoclaim.OneViolation` supaya sampai ke
layar dalam bentuk yang **sama** dengan pelanggaran isian lain — bukan sebagai 500.

### 19.6 Dua pemeriksaan isian, bukan satu

| Method | Dipakai | Menuntut |
|---|---|---|
| `Input.Check` | tambah | seluruh isian, termasuk Sumber Bisnis dan nama penerima |
| `Input.CheckEditable` | simpan & keputusan | hanya isian yang **dapat** diubah |

Keduanya berbagi `checkEditable`, bukan disalin, supaya aturan yang sama tidak pernah berbeda
antara menambah dan menyimpan.

Versi pertama tidak begitu: ia memakai satu `Check` dan handler mengisi `ReceiverName: "-"` agar
lolos. Itu akal-akalan yang **menyembunyikan aturan yang sebenarnya berlaku**, dan diganti sebelum
uji ditulis.

### 19.7 `Store` menyatukan pemilihan, bukan kepentingannya

`Repo` (tabel master) dan `LookupRepo` (empat tabel acuan) tetap **dideklarasikan terpisah**:
keduanya menjawab pertanyaan yang berbeda dan dapat berubah sendiri-sendiri.

Yang disatukan hanyalah **cara memilihnya** — `RepoSelector` mengembalikan `Store` yang memuat
keduanya. Alasannya: keduanya selalu berasal dari koneksi entitas yang sama, sehingga dua pemilih
terpisah hanya akan membuka kemungkinan keduanya menunjuk entitas yang **berbeda**. Itu persis
kelas cacat yang dicegah `R-20`.

### 19.8 `PCT_MAX` disimpan sebagai TEKS

DDL `POOLDATA.M_AUTO_CLAIM_PNC` belum diterima (`R-08`), sehingga tipe kolomnya belum diketahui.
Mengubahnya menjadi angka di dalam aplikasi berarti memutuskan pembulatan dan presisi tanpa dasar
— dan `D-51` melarang nilai uang maupun persentase ditebak.

Yang dikerjakan: baris lama **dibaca apa adanya**, dan yang **baru** diperiksa berbentuk angka
0–100. Koma maupun titik diterima sebagai pemisah desimal — petugas Indonesia mengetik "82,5"
sementara basis data menyimpan "82.5"; menolak salah satunya berarti menolak isian yang benar
hanya karena papan ketiknya.

`strconv.ParseFloat` dipakai, bukan `fmt.Sscanf`: Sscanf berhenti pada karakter pertama yang
tidak cocok dan **tetap melapor sukses**, sehingga `"82abc"` akan lolos sebagai 82.

### 19.9 Identitas pemanggil memakai LOGIN, dan di modul ini itu MENENTUKAN

Modul lain memakai login karena rapi. Di sini ia menentukan: nilainya dibandingkan dengan kolom
`KOMITE`, yang berisi `OPERATOR_ID` dari `POOLDATA.EMAILKOMITE`. Memakai NIK akan membuat tab
Komite Approval **selalu kosong, tanpa satu pun galat** — kegagalan yang tidak terlihat di layar
mana pun.

Nilai yang dibandingkan tidak pernah diterima dari permintaan: layar hanya mengirim
`komite_saya=true`, dan siapa "saya" diambil dari sesi. Menerimanya dari layar berarti siapa pun
dapat melihat antrean persetujuan komite lain dengan mengganti satu nilai.

### 19.10 Selisih yang direncanakan, dan yang bukan

| Selisih | Terlihat di | Sebab |
|---|---|---|
| `CLAIM_ALLOWED` selalu `"1"` juga saat ubah | isi tabel | keputusan Work Owner 19.1 #1 |
| `PCT_MAX` bukan angka 0–100 **ditolak** | penolakan isian | kolomnya persentase; Pega menerima teks apa pun |
| Daftar terurut `INISIALID` | urutan baris di layar | kueri lama tanpa `ORDER BY` |

**Bukan selisih hasil**, tetapi tetap dicatat: penyaring komite berpindah dari perangkaian teks
SQL menjadi kueri terikat, dan daftar membaca dua kolom lebih banyak. Keduanya tidak mengubah
satu baris pun.

### 19.11 Utang teknis yang disadari

| Utang | Keterangan |
|---|---|
| `'BONDING'` tertanam di kueri komite | Work Owner memilih replikasi apa adanya. Persis bentuk hardcode yang `D-15` perintahkan menjadi master. Dijaga terlihat oleh `TestCommitteeQueryKeepsHardcodedBusinessType` — siapa pun yang mengangkatnya kelak menghapus uji itu **dengan sadar** |
| Balapan penambahan hanya **dipersempit** | `FOR UPDATE` tidak dapat mengunci baris yang belum ada. Penutupnya constraint unik pada `INISIALID`; menunggu `R-08` dan `D-63` |
| Batas panjang isian **asumsi** | DDL belum ada. Angkanya diulang di `AutoClaimForm.tsx`, dan duplikasi itu dijaga terlihat oleh `TestLengthLimitsAreStated` |
| Pemetaan galat milik modul sendiri | `TKT-F1-004` masih terhalang; bentuk `{kode, pesan}` tetap sama sehingga klien tidak menghadapi dua bentuk |
| `USERINPUT` menyimpan **pelaku terakhir**, bukan pembuat | tabelnya tidak punya kolom kedua, dan menambah kolom menempuh `D-63` |
| Tidak ada kolom waktu sama sekali | jejak audit (`D-28`, modul `S-5`) belum dapat disandarkan pada tabel ini |

### 19.12 Yang sengaja tidak dikerjakan

| Hal | Alasan |
|---|---|
| **`DELETE`** | Sistem lama tidak punya satu pun terhadap tabel ini, dan `D-66` melarang penghapusan fisik data bernilai bisnis. Baris yang tidak dipakai **ditolak komite**, bukan dibuang. Dijaga `TestNoDeleteStatement` |
| **Isian CLAIM ALLOWED di layar** | selalu `"1"`. Kotak yang isinya selalu diabaikan lebih buruk daripada tidak ada kotak sama sekali |
| **Memindahkan baris ke sumber bisnis lain** | `INISIALID` kunci baris; `UpdateAutoClaim-SQL.xml` pun tidak pernah memindahkannya. Salah pilih diperbaiki dengan menolak lalu menambah baru |
| **Migrasi skema** | `M_AUTO_CLAIM_PNC` dan keempat tabel acuannya **warisan** yang sudah ada; yang dibutuhkan hanyalah hak baca-tulis dari DBA |

## 20. Modul Master Pasal Kerugian (2026-09-19, sesi kedua belas)

Menggantikan `Harness/DetailMasterPasalRejected-Harness.xml` (MENU_ID 27), judul di layar
**"Detail Pasal Kerugian"**, atas tabel `POOLDATA.V_M_DATA_PASAL`.

### 20.1 Keputusan yang MENYUPERSEDE aturan Steering — dan kenapa ia ditulis di depan

`D-66` menetapkan **soft delete menyeluruh**: tidak ada `DELETE` fisik pada data bernilai bisnis di
sistem baru, dan penghapusan dinyatakan lewat penanda.

Layar lama modul ini **punya tombol Hapus**, dan tombol itu menghapus barisnya secara fisik:

```
Section/BrowsePasalDeatailMaster-Section.xml   tombol "Delete"
  -> CNMInsertPasalDataMaster(DeleteFlag="1")
  -> RDB List/DeleteDataPasalDataMaster-SQL.xml
     Delete from POOLDATA.V_M_DATA_PASAL where IDDATA = {InputData.OLD_M_COL_ID}
```

Tabelnya hanya punya tiga kolom dan tidak punya penanda terhapus; menambah kolom menempuh `D-63`.
Tiga jalan keluar diajukan ke Work Owner pada 2026-09-19:

| Pilihan | Akibatnya |
|---|---|
| Penanda di dalam `JSONPASAL` | `D-66` terpenuhi tanpa perubahan skema, tetapi baris yang dihapus **tetap terlihat di Pega** selama masa paralel |
| Minta kolom penanda ke DBA (`D-63`) | paling bersih, tetapi tombolnya **tidak dapat dibangun** sampai kolomnya tiba |
| `DELETE` fisik seperti Pega | setara dengan sistem lama, tetapi **melanggar `D-66`** |

**Jawaban Work Owner: "coba jalankan secara as is"** — pilihan ketiga.

Keputusan itu dihormati dan dijalankan. Yang TIDAK dilakukan adalah menyembunyikannya. Ia dinyatakan
di **lima tempat**, supaya siapa pun yang menyentuh modul ini menemukannya tanpa mencari:

1. doc comment `masterpasal.Repo` — lengkap dengan ketiga pilihan yang ditawarkan;
2. banner `masterpasal.sql` aturan ke-5, dan pada kueri `clause_delete` sendiri;
3. `TestDeleteAppearsInExactlyOneQuery` — pengecualiannya tidak boleh melebar ke kueri lain;
4. README, dengan blok peringatan tersendiri;
5. layar, lewat konfirmasi yang menyebut kata "permanen".

**Konsekuensi yang diterima secara sadar:** baris yang dihapus tidak dapat dipulihkan, dan **tidak
meninggalkan jejak apa pun** — tabelnya juga tidak punya kolom pencatat siapa dan kapan. Bila kelak
ada temuan audit yang menuntut jejak penghapusan, yang berubah adalah skema tabelnya, bukan kode
modul ini.

### 20.2 Tiga keputusan "as is" lainnya

Ketiganya diajukan bersama yang di atas, dan ketiganya dijawab sama.

| # | Yang ditawarkan | Keputusan |
|---|---|---|
| 2 | Kategori: tiga kode `1`/`2`/`3`, atau dua kode plus cabang `else` | **cabang `else`** — kode "Notifikasi" KOSONG |
| 3 | Bisnis: wajib dipilih dari master (seperti Master Auto Claim), atau boleh diketik bebas | **boleh diketik bebas**, seperti `pyAllowFreeFormInput=true` |
| 4 | No Pasal wajib unik, atau boleh kembar | **boleh kembar**; satu-satunya isian wajib tetap No Pasal |

**Kenapa keputusan 2 bukan sekadar soal selera.** Daftar pilihan dropdown hidup di Rule-Obj-Property
`JaminanPengecualianApproval`, dan **tidak ada satu pun direktori Properties di export** (`R-16`).
Yang terbaca hanya ekspresi turunannya:

```
@if(...==1,"Jaminan Polis", @if(...==2,"Pengecualian","Notifikasi"))
```

Mengarang kode `"3"` berarti menebak, dan tebakan itu akan terbukti salah **tanpa satu pun galat**:
baris lama berkode lain akan tampak benar di layar tetapi tersimpan ulang dengan kode yang berbeda.
Yang dipakai karena itu cabang `else` apa adanya.

Ditambah satu pengaman yang tidak diminta tetapi menutup lubang yang sama: bila baris tersimpan
memuat kode di luar ketiganya, layar menambahkan **pilihan bayangan** berlabel
`Notifikasi (kode lama <x>)`. Kode aslinya tetap terpilih dan tetap tersimpan utuh selama pengguna
tidak sengaja menggantinya. Tanpa itu, membuka lalu menyimpan baris semacam itu akan diam-diam
mengganti kodenya.

### 20.3 Yang TIDAK direplikasi, dan itu selisih terencana

Ketiganya dinyatakan di muka supaya tidak ditemukan sebagai kejutan pada uji kesetaraan gerbang 1.

| Selisih | Sebab |
|---|---|
| Nomor `IDDATA` dihitung di Go | `PEGA_D_PASAL_MASTER.prc:11` memakai `max()` **tanpa** `NVL`, sehingga tabel kosong menghasilkan kunci kosong; dan `TO_NUMBER` atas kolom teks gagal ORA-01722 begitu satu baris saja bukan angka |
| `PUT` atas baris yang hilang dijawab `404` | cabang `ELSE` procedure lama justru **menyisipkan baris baru**, sehingga menyunting baris yang sudah dihapus petugas lain menerbitkan baris kedua tanpa satu pun tanda |
| Daftar diurutkan `IDPASAL, IDDATA` | kueri lama tidak punya `ORDER BY` sama sekali, sehingga urutannya dapat berbeda antar pemanggilan |

Ditambah satu pembersihan kecil pada isian Bisnis: butir yang **kode dan namanya sama-sama kosong**
dibuang saat disimpan. Layar lama menerbitkan baris kosong seketika saat ikon tambah ditekan, dan
menyimpannya berarti menyimpan butir yang tidak menunjuk apa pun.

### 20.4 Satu perilaku Pega yang sengaja TIDAK ditiru pada jalur yang merusak

Saat sebuah pasal dibuka untuk disunting, Pega membaca ulang nama lini bisnisnya dari
`POOLDATA.BUSINESS` lewat sub-kueri `(select NOTE from BUSINESS c where c.ID = A.D_COL_ID)`, lalu
menyalinnya ke `.Note`.

Akibatnya pada jalur ketikan bebas: butir itu tidak punya kode, sub-kuerinya tidak menemukan apa
pun, dan `.Note` menjadi **kosong**. Nama yang diketik pengguna **hilang** begitu pasalnya dibuka
kembali — tanpa satu pun pesan.

Di sini penyegaran namanya tetap dilakukan — itu yang membuat nama lini bisnis selalu mutakhir —
tetapi butir yang kodenya tidak ketemu **mempertahankan nama tersimpannya**. Yang direplikasi adalah
hasil yang teramati pada jalur normal, bukan jalur yang merusak; alasan yang sama dipakai modul
Master Auto Claim saat menutup penghapusan data client secara diam-diam (§19.6 pada sesi itu).

### 20.5 Penyimpangan dari dokumen Steering

| Ketetapan | Yang dijalankan | Alasan |
|---|---|---|
| `D-66` soft delete menyeluruh | `DELETE` fisik pada `V_M_DATA_PASAL` | keputusan Work Owner 2026-09-19; rinciannya §20.1 |
| §4.5 batas transaksi di lapisan aplikasi | transaksi dibuka **di dalam** `Repo.Insert` | nomor `IDDATA` diturunkan dari isi tabel itu sendiri; memisahkannya membuka lubang balapan yang justru sedang ditutup. Pengecualian yang sama sudah diambil `masterpenolakan` |
| `10-API-STRATEGY.md` §2 awalan `/api/v1/...` | jalur tanpa awalan versi | kontrak yang ada belum memakainya; memperkenalkannya di satu modul akan membuat dua gaya jalur hidup berdampingan |

### 20.6 Keputusan desain yang diambil sendiri

| Keputusan | Alasan |
|---|---|
| **Domain tidak mengenal JSON sama sekali** | dokumen `JSONPASAL` adalah BENTUK PENYIMPANAN, bukan aturan bisnis. Pembongkarannya berhenti di `repo/sqlstore`, persis seperti nama kolom |
| **`View_DATA_PASAL` tidak dipakai** | view itu tidak ada di export, DDL-nya belum ada (`R-08`), hampir pasti memakai `JSON_TABLE` khas Oracle, dan kuerinya merangkai penyaring dari `{ASIS:...}`. `JSONPASAL` dibaca utuh lalu dibongkar di Go — hasilnya sama, tanpa bergantung pada objek yang tidak dapat dibaca maupun dipindahkan |
| **Daftar tidak memuat lini bisnis** | grid layar lama pun tidak menampilkannya; memuatnya berarti satu pembacaan master lini bisnis untuk setiap pasal demi kolom yang tidak ada |
| **Rute `/kategori` di luar pemeriksaan portal** | isinya milik aplikasi, bukan data entitas. Menuntut portal di sana akan membuat form gagal dimuat justru saat pengguna belum memilih entitas |
| **Nama lini bisnis dibaca satu per satu, bukan dengan `IN`** | jumlahnya kecil — daftar lini bisnis satu pasal, bukan satu tabel — dan klausa `IN` yang panjangnya berubah-ubah berarti satu teks kueri berbeda untuk setiap jumlah baris |
| **`ClauseForm` tanpa zod dan React Hook Form** | aturannya tinggal SATU. Memasang keduanya untuk satu perbandingan dengan teks kosong berarti tiga lapis perantara, sementara isian Bisnis yang berupa daftar justru lebih jernih sebagai state biasa. Server tetap memeriksa ulang seluruhnya |
| **Konfirmasi hapus berupa panel, bukan modal** | modal menuntut perangkap fokus dan penanganan Escape sendiri; yang dibutuhkan hanyalah agar akibatnya terbaca sebelum tombolnya ditekan, dan panel di tempat tidak menutupi baris yang sedang dibicarakan |
| **`DELETE` dibuka di `api/client.ts`** | komentar di sana memang menuntut keputusan sadar sebelum metodenya dibuka; keputusan itu kini ada, dan alasannya ditulis di tempat yang sama |

### 20.7 Yang belum dapat dibuktikan

| Hal | Sebabnya |
|---|---|
| **Pengikatan CLOB lebih dari 4000 karakter** | `go-ora` dapat menolaknya dengan ORA-01461 bila mengikatnya sebagai `VARCHAR2`. Tidak ada Oracle di lingkungan ini; dicatat di banner `masterpasal.sql` sebagai hal yang **wajib** dicoba pada basis data sungguhan dengan satu pasal berisi teks panjang |
| **Bentuk dokumen `JSONPASAL` pada baris lama** | ia dihasilkan `ClipboardPage.getJSON`, dan keluarannya bergantung pada tipe properti klipboard yang tidak ada di export. Yang dikerjakan adalah membuat pembacanya **memaafkan** tipe non-teks, dan melaporkan baris yang tidak dapat diurai beserta `IDDATA`-nya di mode periksa |
| **Kode Kategori yang benar-benar ada di data** | mode `-periksa` melaporkannya bila ditemukan kode di luar ketiganya; sampai itu dijalankan pada basis data sungguhan, dugaan apa pun tetap dugaan |
| **Isi `POOLDATA.BUSINESS`** | tidak ikut dikirim bersama export. Data contoh untuk pengembangan adalah susunan sendiri — **nama lini bisnisnya nyata** (dari `CONTEXT.md`), **kodenya dikarang** — dan tidak boleh dipakai sebagai dasar uji kesetaraan gerbang 1 |

### 20.8 Yang sengaja tidak dikerjakan

| Hal | Alasan |
|---|---|
| **Membuat No Pasal unik** | ditawarkan dan ditolak (§20.2 #4). Kuncinya `IDDATA`, dan Pega pun tidak memeriksanya |
| **Membatasi panjang isian** | lebar kolom `IDPASAL` tidak diketahui (`R-08`); menolak berdasarkan angka yang dikarang berarti menolak isian yang sebenarnya diterima basis data. Yang ada hanyalah batas badan permintaan di lapisan transport |
| **Mengangkat textarea ke `components/`** | hanya modul ini yang memakainya. Menaikkannya lebih dulu berarti menebak bentuk yang dibutuhkan modul lain sebelum modul itu ada — alasan yang sama dipakai `LookupPicker` pada Master Auto Claim |
| **Migrasi skema** | `V_M_DATA_PASAL` dan `BUSINESS` adalah objek **warisan** yang sudah ada; yang dibutuhkan hanyalah hak baca-tulis dari DBA |

---

## 21. Modul Master Bengkel (2026-09-19, sesi ketiga belas)

Menggantikan `Harness/BengkelHE-Harness.xml` (MENU_ID 28) atas `POOLDATA.BENGKEL_HE`.

**Work Owner menjawab keempat pertanyaan dengan kalimat yang sama:** *"coba jalankan
secara as is"*. Keputusan di bawah adalah penerapan `P-5` — replikasi perilaku lebih dulu
— beserta **satu tempat yang tidak dapat direplikasi**, dan alasannya teknis.

### 21.1 Jalur simpan: kolom bernama ke `BENGKEL_HE`, bukan JSON ke `M_BENGKEL_HE`

Ini keputusan terbesar modul ini, dan ia **menyimpang dari "as is"**. Alasannya bukan
selera.

Sistem lama memakai dua tabel untuk satu master:

```
tulis   Activity/UpdateBengkelHE_act  →  InputBengkel.ACCOUNT_ID := @GCNM.GetPageJSONString()
        RDB List/UpdateBengkelHE-SQL  →  POOLDATA.PEGA_M_BENGKEL_HE(Datapega, IDPega, out)
        Database/PEGA_M_BENGKEL_HE.prc:22 → INSERT INTO POOLDATA.M_BENGKEL_HE(ID, JSONDATA)

baca    Report Definition/BrowseBengkelHE_RD → POOLDATA.BENGKEL_HE, 40 kolom
        RDB List/ValidationMasterBengkel     → pooldata.bengkel_he
        RDB List/CountMasterBengkelManagee   → POOLDATA.BENGKEL_HE
        RDB List/GetIDDokumenBengkel         → POOLDATA.BENGKEL_HE
```

Memanggil procedure-nya dilarang `D-02`. Yang tersisa dua pilihan, dan **hanya satu yang
dapat dikerjakan tanpa menebak**:

| Pilihan | Dapat dikerjakan? |
|---|---|
| Menulis dokumen JSON ke `M_BENGKEL_HE.JSONDATA` dengan SQL biasa | **TIDAK.** Nama kunci JSON-nya diterbitkan `@GCNM.GetPageJSONString()`, dan `Function/GetPageJSONString-Function.xml` **hanya memuat tanda tangannya** — badan fungsinya tidak ikut di export (`R-16`). Setiap kunci yang ditulis akan menjadi tebakan, pada master yang menentukan diskon, pajak, dan rekening tujuan pembayaran |
| Menulis kolom bernama ke `BENGKEL_HE` | **YA.** Keempat puluh kolomnya terbaca lengkap dari `BrowseBengkelHE_RD-RD.xml`, sehingga setiap nilai yang ditulis diketahui benar, dan apa yang ditulis dapat dibaca kembali oleh layar yang sama |

Yang kedua dipakai. **Perlakuannya sama dengan Master Status Klaim**, yang menghadapi
keluarga procedure `PEGA_M_*` yang sama persis — lima belas procedure dengan pola
site-prefix + `JSONDATA` — dan memutuskan hal yang sama: Go menjadi penulis tunggal dan
berhenti menulis `JSONDATA` (`D-02`, `D-68`). Lihat banner
`internal/masterstatus/repo/sqlstore/masterstatus.sql` dan migrasi 0002.

**Penomorannya tetap direplikasi apa adanya**, dan itu yang membuat keputusan ini aman
terhadap Pega: `PEGA_M_BENGKEL_HE.prc:11,19` membaca `M_SITE_DATABASE.ID` lalu menambahkan
`LPAD(BENGKEL_HE_SEQ.NEXTVAL,10,'0')`. Kedua sumber yang sama dipakai di sini, sehingga ID
yang diterbitkan aplikasi ini **melanjutkan deret yang sudah ada** dan tidak pernah
bertabrakan dengan ID yang pernah diterbitkan Pega.

#### Satu hal yang belum dapat dipastikan, dan cara memastikannya

Apakah `POOLDATA.BENGKEL_HE` sebuah **tabel** atau sebuah **view** atas
`M_BENGKEL_HE.JSONDATA`. DDL-nya tidak ada di export (`R-08`), dan keduanya sama-sama
masuk akal — preseden bentuk kedua ada: `V_STS_CLAIM` adalah view atas
`M_STS_CLAIM.JSONDATA`, dan migrasi 0002 yang membongkarnya menempuh `D-63`.

Yang dikerjakan alih-alih menebak: **`claimpnc -periksa` membandingkan jumlah baris
keduanya**, dan melaporkan hasilnya beserta apa artinya. Bila jumlahnya sama, keduanya
kemungkinan satu sumber dan kueri modul ini berjalan apa adanya. Bila berbeda, keduanya
dua tabel terpisah — dan penulisan modul ini tidak akan sampai ke Pega maupun sebaliknya.

**Jalur tulis tidak boleh diaktifkan di produksi sebelum DBA memastikannya.**

### 21.2 Pembuatan akun bengkel TIDAK dibawa

`Activity/ValidationLoginBengkel_act` step 9 memanggil `GCNMCreateOperator` dengan
`accessGroup=GKM:InboxWorkshop`, `unitName="WorkShop"`, `orgName="ASM"`, dan — pada step 2
— `Local.password := "123456"`.

Dua alasan, dan yang kedua cukup sendirian:

1. **Tidak ada targetnya.** Sistem baru tidak punya operator Pega, dan kontrak
   identitasnya sendiri belum ada (`F-3`, `R-14`).
2. **Kata sandinya sama untuk setiap bengkel yang pernah dibuatkan akun.** Mereplikasinya
   berarti menerbitkan akun dengan kata sandi yang sudah diketahui siapa pun yang pernah
   membaca rule itu. Ia bukan "keanehan sistem lama yang direplikasi demi kesetaraan"
   melainkan cacat keamanan aktif.

**Yang tetap dibawa:** `LOGIN_APLIKASI` disimpan, diperiksa keunikannya, dan **diwajibkan
untuk bengkel rekanan**. Syarat terakhir itu bukan karangan — `ValidationLoginBengkel_act`
melompat keluar pada prasyarat `Local.STS_REKANAN=='0'`, sehingga bengkel non-rekanan
memang tidak pernah punya login.

**Yang diperiksa berbeda, dan itu tidak terhindarkan.** Pega mencari login yang sama di
daftar operator Pega (`Data-Admin-Operator-ID` lewat report `GCNMGetListOfOperators`); di
sini keunikannya diperiksa terhadap kolom `LOGIN_APLIKASI` pada tabel bengkel itu sendiri.
Akibatnya: login yang bertabrakan dengan operator Pega yang **bukan** bengkel tidak lagi
tertangkap. Selisih terencana, tertutup begitu `F-3` punya kontrak.

**Layar MENYATAKAN ketiadaannya.** Saat status rekanan dipilih, form menampilkan kalimat
bahwa login hanya *disimpan* dan akunnya belum diterbitkan. Tanpa itu, petugas akan
mengira bengkelnya sudah bisa masuk hanya karena login-nya tersimpan.

### 21.3 Surel pemberitahuan TIDAK dikirim, tetapi dicatat

`Activity/UpdateBengkelHE_act` step 18–19 mengirim surel ke satu alamat yang tertanam di
dalam rule — **alamat pribadi seseorang**, bukan mailbox fungsional. `D-67` melarangnya
dibawa, dan seam Notifier (`S-3`) belum ada.

Yang dikerjakan: peristiwanya **dicatat di log** sebagai `Info`, menyebutkan bengkel mana,
portal mana, oleh siapa, dan **kenapa tidak dikirim**. Yang hilang adalah satu langkah
proses yang nyata — PIC tidak lagi diberi tahu bahwa ada bengkel menunggu persetujuan —
dan ketiadaannya harus terbaca di log alih-alih ditemukan berbulan kemudian oleh petugas
yang bertanya kenapa antreannya menumpuk.

### 21.4 Approve dan Reject ditaruh di layar ini

Di Pega keduanya **tidak** ada di layar bengkel: `Section/ApprovalMasterBengkelHE` dipakai
`InboxManager_Sec`, dan keputusannya dijalankan `Activity/SetApprovalAllMaster` yang
melayani bengkel, panel, dan sparepart sekaligus lewat `Param.TIPE2`.

Rule SQL yang benar-benar menjalankan penetapannya **tidak ada di export** (`R-16`):
`SetApprovalAllMaster` dirujuk empat berkas tetapi tidak punya berkas sendiri. Yang
terbaca hanyalah kolom yang disentuhnya — `ID_BENGKEL`, `APPROVAL`, dan `MAIL`.

Inbox Manager belum dibangun. Menunda keputusannya sampai layar itu ada berarti setiap
bengkel yang ditambah tertahan di Waiting Approval tanpa satu pun cara menyelesaikannya —
dan alur ini tidak dapat dicoba sama sekali.

**Yang dipakai adalah bentuk yang sama persis**: centang beberapa baris, lalu satu tombol
untuk seluruh pilihan. Memindahkannya ke Inbox Manager kelak hanya soal letak tombol,
bukan soal perilaku.

**Satu kolom sengaja TIDAK ikut ditulis.** `SetApprovalAllMaster` menulisi
`InputBengkel.MAIL := .USER_UPDATE` — menimpa kolom surel bengkel dengan identitas petugas
yang menyetujui. Itu kolom berarti ganda yang ketiga, dan membawanya berarti menghapus
alamat surel bengkel setiap kali ia disetujui. `TestDecisionQueryTouchesOnlyApproval`
menjaganya.

### 21.5 Nilai sah tujuh penanda status: saran dari data, bukan tebakan

`STS_SUPPLY`, `STS_EKLAIM`, `STS_AUTO_AKSEP`, `STS_PAYMENT`, `STS_AUTOPAYMENT`,
`STS_TEKNO`, `STS_ORDER` dirender `pxRadioButtons` atau `pxDropdown`, dan daftar
pilihannya ada di rule Field Value yang tidak ikut di export.

Pencarian menyeluruh atas `Activity/`, `When/`, `RDB List/`, dan seluruh section bengkel
untuk setiap perbandingan terhadap ketujuhnya menghasilkan **nol hasil**.

Tiga pilihan dipertimbangkan:

| Pilihan | Ditolak karena |
|---|---|
| Dropdown "Ya/Tidak" dengan nilai `1`/`0` | menebak domain kolom yang menentukan kanal mana yang boleh dipakai bengkel. Satu tebakan yang salah berarti bengkel kehilangan kanal tanpa satu pun tanda |
| Teks bebas tanpa bantuan apa pun | jujur, tetapi petugas harus **hafal** kodenya — dan tidak ada satu pun tempat ia dapat melihatnya |
| **Teks bebas dengan saran dari nilai yang sudah dipakai baris lain** | **dipakai** |

Yang ketiga menjawab pertanyaan "nilai apa yang sah di kolom ini" dengan satu-satunya
sumber yang tersedia: **data itu sendiri**. Layar mengumpulkan nilai berbeda dari baris
yang sedang termuat lalu menawarkannya sebagai `<datalist>`. Tidak ada yang dikarang, dan
petugas tetap dapat mengetik nilai yang belum pernah dipakai.

**Satu-satunya yang dibuat dropdown adalah `STATUS_REKANAN`**, karena nilainya diketahui —
dan diketahuinya pun dari percabangan, bukan dari label: `ValidationLoginBengkel_act`
melompat keluar pada `Local.STS_REKANAN=='0'`. Ia dibuat dropdown karena ia **menentukan
wajib-tidaknya isian lain**, sehingga teks bebas di sana akan membuat aturan validasinya
bergantung pada ketikan.

### 21.6 Tiga isian wajib, tidak lebih

Sistem lama **tidak punya satu pun prasyarat "wajib diisi"** pada layar ini — berbeda dari
Master Auto Claim yang menolak delapan isian kosong sekaligus. Yang ada hanyalah dua
pemeriksaan keunikan.

Yang diwajibkan dibatasi pada tiga isian yang **tanpanya baris itu tidak dapat dipakai
siapa pun**, dan ketiganya dapat dibenarkan dari export:

| Isian | Kenapa wajib |
|---|---|
| `NAMA_BENGKEL` | ia kunci alami — `ValidationMasterBengkel` mencari baris DENGAN nama itu, sehingga baris tanpa nama tidak akan pernah tertangkap pemeriksaan ganda, dan dua baris tanpa nama akan lolos berdampingan |
| `STATUS_REKANAN` | ia yang menentukan apakah bengkel diberi login |
| `LOGIN_APLIKASI` | wajib **hanya** bila bengkelnya rekanan; bengkel rekanan tanpa login akan tersimpan diam-diam tanpa pernah dapat masuk |

Selebihnya boleh kosong, persis seperti hari ini. Mewajibkan lebih banyak akan menolak
penambahan yang sekarang diterima — dan itu selisih perilaku yang tidak diminta siapa pun.
`TestEverythingElseMayBeBlank` menjaganya.

### 21.7 Empat puluh kolom bertipe TEKS, termasuk yang jelas angka dan tanggal

Termasuk `PPN`, `DISC_JASA`, `DISC_SPART`, `PERSEN_MATERIAL`, `SLA`, dan `TGL_STATUS`.

Alasannya satu: **DDL tabelnya belum diterima** (`R-08`). Mengubahnya menjadi angka atau
waktu berarti memutuskan presisi, pembulatan, dan zona waktu tanpa dasar — dan nilai uang
maupun persentase tidak boleh ditebak (`D-51`, `F-5`).

Baris lama dibaca apa adanya; yang baru **diperiksa** berbentuk angka 0–100 lewat
`Input.Check`, tetapi **tidak pernah diubah bentuknya saat disimpan**. Koma dan titik
keduanya diterima sebagai pemisah desimal — petugas Indonesia mengetik "12,5" sementara
basis data menyimpan "12.5" — dan yang tersimpan tetap apa yang diketik.
`TestPercentValueKeptVerbatim` menjaganya.

### 21.8 Dua hal yang Pega hapus diam-diam, dan di sini dipertahankan

Keduanya ditemukan saat membaca urutan langkah, bukan saat mencari cacat:

| Apa | Di Pega | Di sini |
|---|---|---|
| `DOKUMENID` | diisi dari hasil `PNCSaveAttachmentToDB` pada **setiap** penyimpanan, sehingga penyimpanan tanpa lampiran menimpanya dengan kosong — lampiran lenyap hanya karena barisnya disunting | dibaca dari baris yang tersimpan dan ditulis kembali apa adanya |
| `MAIL` | ditimpa `USER_UPDATE` saat menyetujui | tidak ikut ditulis sama sekali |

Keduanya **bukan** selisih hasil yang terlihat pengguna pada jalur normal; yang berbeda
adalah jalur yang menghapus datanya sendiri tidak ikut dibawa.

### 21.9 Pemeriksaan keunikan ada di jalur simpan juga

Di Pega, `ValidateMasterBengkel` dipanggil dari layar tanpa memandang tambah atau ubah,
dan `UpdateBengkelHE_act` memanggil `ValidationLoginBengkel_act` pada **ketiga** tabnya.

Melewatkannya pada jalur simpan akan memperbolehkan dua bengkel bernama sama — cukup
dengan menyunting salah satunya. Karena itu `ensureUnique` dijalankan juga saat menyimpan,
dengan **baris yang sedang disunting dikecualikan**: menyimpan tanpa mengubah namanya
tidak boleh ditolak karena namanya sendiri sudah dipakai oleh dirinya sendiri.

Pada jalur **tambah** pemeriksaannya berada di dalam `Repo.Insert`, di dalam satu
transaksi bersama `FOR UPDATE` — karena di sanalah balapan penambahan dapat terjadi.
Keterbatasannya dinyatakan terang-terangan di kedua tempat: **`FOR UPDATE` tidak dapat
mengunci baris yang belum ada**, sehingga dua penambahan atas nama yang sama masih dapat
lolos keduanya. Penutupnya adalah constraint unik, dan itu menunggu `R-08` dan `D-63`.

### 21.10 Keputusan borongan: satu pernyataan per baris, di dalam satu transaksi

Bukan satu pernyataan dengan daftar kunci yang panjangnya berubah-ubah. Daftar placeholder
yang dibentuk saat berjalan membuat teks SQL tidak lagi tetap, dan itu persis bentuk yang
`08-TECHNICAL-STRATEGY.md` §4.3 larang.

Biayanya beberapa perjalanan tambahan pada operasi yang jarang dan berbaris sedikit; yang
diperoleh adalah teks kueri yang dapat dibaca utuh di berkas `.sql`.

Transaksinya melingkupi **seluruh** baris, sehingga persetujuan borongan tidak pernah
setengah jalan. Sistem lama tidak menjamin itu — `SetApprovalAllMaster` menjalankan satu
RDB-List per baris tanpa transaksi yang melingkupinya.

Batas **200 baris** per permintaan ditambahkan. Ia bukan aturan bisnis melainkan penjaga
sumber daya: transaksi yang menahan ribuan kunci baris menghalangi Pega yang sedang
melayani produksi pada tabel yang sama (`D-21`).

### 21.11 `CityPicker` disalin, tidak dinaikkan ke `components/`

Ia nyaris sama dengan `LookupPicker` milik Master Auto Claim. Aturan susunan frontend
melarang satu fitur mengimpor dari fitur lain, dan kebutuhan bersama naik ke
`components/` — tetapi menaikkannya **sekarang** berarti menyunting `master-auto-claim`
yang sudah dinyatakan selesai, dan Isolasi Protektif melarangnya.

Syarat menaikkannya kelak ditulis di dalam berkasnya supaya tidak perlu ditemukan ulang:
begitu ada modul **ketiga** yang membutuhkan kotak cari–pilih, ketiganya dipindahkan
sekaligus dalam satu perubahan — bukan dua modul menunggu satu sama lain.

### 21.12 Tombol keputusan dinamai berbeda dari tab

Uji frontend menemukan bahwa tab "Approve"/"Reject" dan tombol keputusan "Approve"/"Reject"
tidak dapat dibedakan dari namanya. Caption tab **harus** tetap — ia caption Pega (`D-13`)
— sehingga yang diubah adalah tombolnya: **"Approve terpilih"** dan **"Reject terpilih"**.

Caption tombol keputusan sendiri tidak dapat dibaca dari export (`ApprovalMasterBengkelHE`
hanya menyisakan caption 'PILIH' dan 'BUTTON'), sehingga tidak ada teks Pega yang
dikorbankan.

Ia perbaikan aksesibilitas, bukan selera: pembaca layar mengumumkan dua kontrol yang sama
sekali berbeda dengan kata yang sama persis, dan pengguna perintah suara tidak punya cara
memilih yang mana.

### 21.13 Utang yang dicatat, bukan diselesaikan

| Utang | Kenapa dibiarkan |
|---|---|
| Penyaring `LDI_ID='0076'` tertanam di kueri cabang | persis bentuk hardcode yang `D-15` perintahkan menjadi konfigurasi, tetapi tidak ada satu pun keterangan di export tentang artinya — mengangkatnya berarti menebak nilainya untuk entitas lain. Ia literal, bukan masukan pengguna, sehingga bukan celah injeksi. Dijaga terlihat oleh `TestBranchQueryKeepsHardcodedApplicationCode` |
| Tabel `CITY` disebut **tanpa skema** | persis seperti seluruh kueri lama yang membacanya. Melengkapinya berarti menebak, dan tebakan yang salah membuat lookup Kota kosong tanpa galat yang menjelaskan sebabnya |
| Pemetaan galat milik modul sendiri | kontrak galat yang mengikat seluruh aplikasi adalah `TKT-F1-004` dan masih terhalang; menambah kode ke modul auth berarti menyunting modul yang sudah selesai |
| Jalur `/api/master/bengkel` tanpa `/v1` | `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`; memperkenalkannya di satu modul saja akan membuat dua gaya jalur hidup berdampingan |

---

## 22. Modul Master Panel (2026-09-20, sesi keempat belas)

Menggantikan `Harness/MasterPanel_HE-Harness.xml` (MENU_ID 30) atas `POOLDATA.PANEL_HE`
beserta tabel anaknya `POOLDATA.LOKASI_PANEL_HE`.

**Modul master pertama yang mengelola baris anak.** Seluruh modul master sebelumnya rata.

### 22.1 Dua pertanyaan, dua jawaban "as is"

| Pertanyaan | Jawaban | Yang dijalankan |
|---|---|---|
| Sub-tabel Lokasi/Sisi | *"Jalankan sebagai as is"* | dikelola penuh — baca, tambah, ubah, hapus |
| Kesembilan kolom `STS_*` | *"Jalankan sebagai as is"* | disimpan sebagai teks apa adanya, tanpa menebak enum |

Keduanya dibaca sebagai **`P-5`**: replikasi perilaku lebih dulu, perbaikan kemudian.

### 22.2 Menulis kolom bernama, bukan dokumen JSON

Sistem lama memakai **dua tabel** untuk satu master — pola dua-penyimpanan yang
`03-CURRENT-ARCHITECTURE.md` §3.2 catat sebagai `R-10`:

```
tulis  Activity/CNMUpdatePanelHE_act  →  InputData.NAME := @GCNM.GetPageJSONString()
       RDB List/UpdatePanel_HE-SQL    →  POOLDATA.PEGA_M_PANEL_HE(Datapega, IDPanel, out)
       Database/PEGA_M_PANEL_HE.prc   →  INSERT INTO POOLDATA.M_PANEL_HE(ID, JSONDATA)

baca   BrowseMasterPanel_HE_RD        →  POOLDATA.PANEL_HE, 14 kolom
       ValidationMasterPanel          →  pooldata.panel_he
       GetLokasiSisiPanel             →  pooldata.lokasi_panel_he
```

Menulisnya lewat procedure dilarang `D-02`. Dua pilihan tersisa, dan hanya satu yang dapat
dikerjakan tanpa menebak:

| Pilihan | Dapat dikerjakan? |
|---|---|
| (a) menulis dokumen JSON ke `M_PANEL_HE.JSONDATA` | **TIDAK** — nama kunci JSON-nya diterbitkan `@GCNM.GetPageJSONString()`, dan badan fungsinya tidak ikut di export (`R-16`). Setiap kunci menjadi tebakan |
| (b) menulis kolom bernama ke `PANEL_HE` dan `LOKASI_PANEL_HE` | **YA** untuk induk (14 kolom terbaca lengkap dari RD); **sebagian** untuk anak — lihat §22.4 |

**(b) yang dipakai**, sama dengan Master Bengkel dan Master Status Klaim yang menghadapi
keluarga procedure `PEGA_M_*` yang sama.

**Yang wajib dipastikan DBA sebelum jalur tulis dipakai di produksi:** apakah `PANEL_HE`
sebuah TABEL atau sebuah VIEW atas `M_PANEL_HE.JSONDATA`. DDL-nya tidak ada (`R-08`), dan
keduanya sama-sama masuk akal — preseden bentuk VIEW ada: `V_STS_CLAIM` adalah view atas
`M_STS_CLAIM.JSONDATA`. `claimpnc -periksa` membandingkan jumlah baris keduanya.

### 22.3 Lebar ID enam digit, bukan sepuluh

`PEGA_M_PANEL_HE.prc:21` memakai `lpad(…,6,'0')`, sementara `PEGA_M_BENGKEL_HE.prc:19`
memakai `lpad(…,10,'0')`. Kedua procedure ditulis dengan pola yang sama dan lebarnya tetap
berbeda; **ditiru apa adanya**, karena menyeragamkannya akan menerbitkan ID yang tidak
sebentuk dengan ID yang sudah ada.

Akibat sampingan yang dicatat: pada lebar enam, batas `ComposeID` jauh lebih dekat. Nomor
urut yang melewati enam digit **tidak dipotong** — kuncinya dibiarkan tumbuh, dan itu
terlihat, alih-alih bertabrakan diam-diam seperti yang dilakukan `LPAD` Oracle.

### 22.4 Kolom `NAMA` pada tabel anak — asumsi, dan pemeriksaannya

Tabel anak punya empat kolom; hanya tiga yang artinya pasti.

| Kolom | Sumber | Arti |
|---|---|---|
| `ID_PANEL` | `GetLokasiSisiPanel-SQL` | kunci induk |
| `LOKASI_PANEL` | idem, dialiaskan `"NAME"` | nama lokasi |
| `SISI_PANEL` | idem, dialiaskan `"STS_SISI"` | sandi sisi |
| `NAMA` | `GetDataSisiPanel-SQL` | **hanya muncul sebagai penyaring** |

**Keputusan:** `NAMA` diisi nilai yang **sama** dengan `LOKASI_PANEL`.

Alasannya bukan kemudahan melainkan **menjaga pembaca hilirnya tetap bekerja**:
`GetDataSisiPanel` dipakai modul **Grouping Sparepart HE** untuk mencari sisi sebuah
lokasi, dan ia menyaring `nama = <nama lokasi>`. Membiarkan `NAMA` kosong akan mematikan
modul itu **tanpa satu pun pesan galat** — kelas kegagalan yang paling mahal ditemukan.

**Asumsinya dapat dibantah, dan cara membantahnya sudah terpasang.** `claimpnc -periksa`
menghitung baris yang `NAMA`-nya berbeda dari `LOKASI_PANEL`, dan melaporkan `[WASPADA]`
beserta larangan mengaktifkan jalur tulis bila hasilnya bukan nol.

Ini satu-satunya pemeriksaan di seluruh mode periksa yang menguji **asumsi penulisan**,
bukan ketersediaan tabel.

### 22.5 Hapus-lalu-sisip-ulang — pertentangan dengan `D-66` yang dinyatakan terbuka

Penyimpanan mengganti **seluruh** baris lokasi: dibuang, lalu disisipkan ulang.

| Hal | Isi |
|---|---|
| **Kenapa** | baris anak tidak punya kunci sendiri — tidak ada kolom surrogate yang memungkinkan satu baris dikenali lintas penyimpanan |
| **Preseden Pega** | `PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503` melakukan hal yang sama pada 12 tabel |
| **Yang dilanggar** | `D-66` — soft delete menyeluruh |
| **Status penggantinya** | `ADR-0013` — **belum diputuskan** |
| **Keputusan Work Owner** | *"as is"* (2026-09-19, dan ditegaskan lagi 2026-09-20) |
| **Yang menahan akibatnya** | keduanya di dalam **satu transaksi**; sistem lama tidak menjamin itu |
| **Yang menjaganya tetap terlihat** | `TestDeleteOnlyOnTheChildTable` — `DELETE` terhadap tabel **induk** tetap dilarang |

### 22.6 Kesepuluh isian induk WAJIB

Berbeda dari Master Bengkel, yang tidak punya satu pun isian wajib di layarnya.

Kesepuluhnya bertanda `pyRequired=true` pada `Section/BrowsePanelHEApproval-Section.xml` —
diperiksa satu per satu, bukan disimpulkan dari salah satunya. Termasuk `EXCLUSION_C`,
yang artinya **tidak diketahui sama sekali** tetapi tetap diwajibkan karena begitulah
layar lamanya.

### 22.7 Dua perbedaan yang DIRENCANAKAN terhadap Pega

| # | Perbedaan | Alasan |
|---|---|---|
| 1 | **Sisi di luar tiga sandi yang dikenal ditolak** | `GetSisiPanel-Act.xml` membaca apa pun yang bukan `"-"` dan bukan `"1"` sebagai `"KANAN"`, sehingga nilai rusak tampil sebagai nilai yang sah |
| 2 | **Lokasi kembar pada sisi yang sama ditolak** | sistem lama tidak memeriksanya, dan dua baris kembar membuat `GetDataSisiPanel` — yang membaca satu nilai saja — mengembalikan baris yang mana pun lebih dulu ditemukan |

Baris lama yang sudah memuat keduanya tetap **dibaca apa adanya**; yang ditolak hanya
penyimpanan baru.

### 22.8 Yang TIDAK dibatasi, meski tampak wajar

Nama lokasi **tidak** dibatasi pada kelima pilihan `LocationOptions`. Baris lama dapat
memuat lokasi lain, dan menolaknya berarti baris yang hari ini sah tidak dapat disimpan
ulang. Dropdown layar menawarkan kelimanya; validasi tidak memaksakannya.

### 22.9 `STS_SISI` memikul dua arti — dipisah di sini

| Tempat | Nama di modul ini | Nama di kontrak API |
|---|---|---|
| kolom `PANEL_HE.STS_SISI`, caption "STATUS SISI" | `Panel.SideStatus` | `status_sisi` |
| alias `SISI_PANEL as "STS_SISI"` pada tabel anak | `PanelLocation.Side` | `sisi_panel` |

Bentuk utang yang sama dengan dua yang sudah dicatat di Master Bengkel (`ACCOUNT_ID` dan
`ALASAN_STS_BGKL`).

### 22.10 Alasan penolakan hanya tersimpan pada keputusan TOLAK

`POOLDATA.PANEL_HE` punya kolom `ALASAN_TOLAK`, dan layar lama menaruh isian "Catatan"
(`TempStsClaim.pyNote`) berdampingan dengan tombol keputusan. Keduanya jelas sepasang.

Menuliskannya pada persetujuan akan mengisi kolom bernama "alasan tolak" pada baris yang
justru **disetujui** — satu lagi kolom berarti ganda, yang justru sedang dihindari modul
ini. Layar **menyatakan** aturan itu di bawah isiannya, bukan membiarkannya menjadi
kejutan.

### 22.11 Daftar pilihan disajikan endpoint tanpa portal

`GET /api/master/panel/pilihan` adalah satu-satunya rute modul ini yang **tidak** menuntut
portal. Isinya konstanta yang ditanam di `Activity/SetLokasiSisiPanel-Act.xml`, bukan
bacaan basis data entitas mana pun.

Memasangi pemeriksaan portal padanya akan membuat form tidak dapat menggambar dropdown-nya
sebelum portal dipilih — penolakan yang tidak melindungi apa pun. Perlakuannya sama dengan
daftar Kategori pada Master Pasal Kerugian.

### 22.12 Utang teknis yang ditambahkan sesi ini

| Utang | Rencana penyelesaian |
|---|---|
| Kode galat modul dipetakan di modul sendiri | pindah ke tempat bersama begitu `TKT-F1-004` diputuskan |
| Jalur `/api/master/panel` tanpa awalan `/v1` | penyeragaman seluruh aplikasi, bukan sepihak di satu modul |
| Batas panjang diulang di backend dan `PanelForm.tsx` | dijaga `TestLengthLimitsAreTheOnesTheFormRepeats`; hilang begitu DDL diterima (`R-08`) |
| `sequenceWidth` disalin di `sqlstore` dan `memory` | paket memori tidak boleh bergantung pada paket sqlstore; kesamaannya dijaga uji |
| Asumsi kolom `NAMA` | dibantah atau dikuatkan `claimpnc -periksa`; bila terbantah, `masterpanel.sql` diperbaiki sebelum jalur tulis aktif |

---

## 23. Modul Master Supplier (2026-09-20, sesi kelima belas)

Modul kesebelas, atas tabel **`M_SUPPLIER`**. Ia berbeda dari sepuluh modul sebelumnya
pada satu hal yang menentukan hampir seluruh keputusan di bawah: **tabelnya hanya punya
tiga kolom — `ID`, `OLDID`, dan `JSONDATA`.**

### 23.1 Menulis dokumen JSON, dan kenapa itu justru yang benar di sini

`D-02` menetapkan aplikasi tidak memanggil stored procedure. `PEGA_M_SUPPLIER` karena itu
tidak dipanggil, dan yang tersisa adalah menulis tabelnya langsung — tetapi tabelnya tidak
punya kolom bernama satu pun.

Master Bengkel menghadapi pertanyaan yang **sama persis** dan menjawabnya **berbeda**.
Banner `masterbengkel.sql` menolak pilihan "menulis dokumen JSON dengan SQL biasa" dengan
alasan yang tegas: nama kuncinya diterbitkan `@GCNM.GetPageJSONString()`, dan badan fungsi
itu tidak ada di export — *"setiap kunci yang ditulis akan menjadi tebakan, pada master
yang menentukan diskon, pajak, dan rekening tujuan pembayaran."*

Alasan itu **tidak berlaku di sini**, dan itulah yang membalik keputusannya.
`RDB List/GetDataEditMasterSupller-SQL.xml` membaca setiap kunci satu per satu:

```sql
A.JSONDATA.NAMA AS "NAMA", A.JSONDATA.ALAMAT AS "ALAMAT", …
```

Kedua puluh lima kuncinya terbaca lengkap. Ditambah tiga yang terbaca dari kedua activity
penyimpan — `USERKLAIMID`, `TGL_INSERT`, dan `ID` — seluruh dokumen dapat ditulis **tanpa
satu pun tebakan**.

**Yang menjaganya:** `TestWrittenKeysMatchReadKeys` membandingkan kunci yang ditulis
`documentValue` dengan kunci yang dibaca kueri. Sebuah kunci yang ditulis tetapi tidak
pernah dibaca kembali — kelas cacat yang hanya terlihat sebagai isian yang diam-diam
kosong setelah disimpan — membuat uji itu gagal.

### 23.2 Notasi titik Oracle diganti `JSON_VALUE`

Kueri lamanya memakai `A.JSONDATA.NAMA`, yang **tidak ada padanannya di PostgreSQL**.
Menyalinnya apa adanya berarti seluruh modul ini berhenti bekerja pada hari perpindahan
basis data, dan berhentinya tidak akan terlihat sebelum itu.

`JSON_VALUE(JSONDATA, '$.NAMA')` mengembalikan nilai yang sama persis dan berlaku di Oracle
12c+ maupun PostgreSQL 17+ — justru alasan `D-24` mewajibkan versi 17 (222 pemanggilan
SQL/JSON di seluruh sistem lama).

Dijaga `TestNoOracleDotNotation`, uji yang khas modul ini: modul master lain tidak membaca
kolom JSON sama sekali.

### 23.3 Penggabungan dokumen, bukan penimpaan

`PEGA_M_SUPPLIER.prc:36` mengganti **seluruh** dokumen:

```sql
UPDATE M_SUPPLIER SET JSONDATA = DataPega WHERE ID = IDPega;
```

dan yang dikirim Pega hanyalah kunci yang kebetulan ada di halaman klipboardnya. Kunci apa
pun yang tidak dibaca `GetDataEditMasterSupller` karena itu **lenyap pada setiap
penyimpanan** — termasuk kunci yang ditulis jalur lain, dan termasuk kunci yang belum
diketahui siapa pun.

`mergeDocument` membaca dokumen tersimpan lebih dulu (`FOR UPDATE`), menimpa kunci yang
dikenal, dan membiarkan sisanya. Perlakuannya **sama persis** dengan `DOKUMENID` pada
Master Bengkel: jalur yang menghapus datanya sendiri tidak ikut dibawa.

`FOR UPDATE` bukan hiasan. Tanpa itu, dua penyimpanan atas baris yang sama akan sama-sama
membaca dokumen lama dan yang terakhir menimpa perubahan yang pertama — tanpa satu pun
tanda.

**Dokumen lama yang rusak tidak menghentikan penyimpanan.** Baris yang `JSONDATA`-nya bukan
JSON sah ditulis ulang dari nol alih-alih menolak permintaannya; yang hilang hanyalah kunci
yang memang sudah tidak terbaca siapa pun. Menolak akan mengunci petugas tanpa cara
memperbaikinya.

### 23.4 `PROTEKSI_ID` yang tidak dapat ditiru

Baris permintaan persetujuan menuntut sebuah kunci, dan di Pega kunci itu adalah
`ChildPageProtection.pyID` — ID work object case `ASM-FW-GKM-Work-Protection` yang dibuat
tiga langkah sebelumnya lewat `CreateWorkPage` dan `AddWork`.

Sistem baru tidak punya work object Pega, dan **tidak ada satu pun rule di export yang
memperlihatkan bentuk ID-nya.** Tiga langkah pembuatan case itu tidak dibawa; yang
dibutuhkan darinya hanyalah kuncinya.

`ComposeApprovalID` membentuknya sebagai `SUP.` + ID supplier + waktu UTC berformat
`yyyyMMddHHmmss`. Tiga hal yang diperolehnya:

1. **Asalnya terbaca.** Awalan `SUP.` tidak mungkin diterbitkan Pega — alasan yang sama
   dengan prefiks `PNCN` pada nomor klaim (`D-22`).
2. **Barisnya tertaut ke supplier-nya** tanpa membaca kolom `NO_KLAIM`, yang namanya justru
   menyesatkan — ia diisi ID supplier, bukan nomor klaim.
3. **Penyimpanan berulang dalam detik yang sama ditolak** basis data alih-alih menghasilkan
   dua permintaan kembar, bila kolomnya memang berkunci utama.

Waktunya UTC, bukan WIB, dan itu disengaja: ia kunci teknis. Memakai zona waktu setempat
pada sebuah kunci berarti kuncinya berulang setiap kali zona waktunya bergeser.

**Bentuk ini BELUM disetujui.** Pilihan lain — sequence tersendiri — menuntut objek basis
data baru dan menempuh `D-63`.

### 23.5 `SUPPLIER_HE` diturunkan ke DUA arah, sistem lama hanya satu

`CreateNewMasterSupplier_post` step 7 hanya punya cabang "bila", tanpa cabang "selain itu".
Akibatnya supplier HE yang diubah menjadi bukan-HE **tetap tersimpan sebagai HE**, dan
`GetDataSupplier_pre` mengembalikan `JENIS_STATUS` menjadi `"1"` saat dimuat — perubahannya
hilang tanpa satu pun tanda.

`DeriveHeavyEquipment` selalu menulis kedua arah. **Selisih yang direncanakan**, dan
satu-satunya yang terlihat pengguna.

Arah kebalikannya — `DeriveSupplyType` — dipakai saat **membaca**, dan itu bukan kerapian:
dokumen lama dapat memuat `JENIS_STATUS` yang tidak sejalan dengan `SUPPLIER_HE` justru
karena cacat di atas, dan yang menentukan perilaku sistem hilir adalah `SUPPLIER_HE`.
Membacanya dari sanalah yang membuat layar menampilkan keadaan yang sebenarnya berlaku.

### 23.6 Daftar dropdown dibaca dari DATA, bukan dikarang

Kelima isian bersandi — `STS_REKANAN`, `JENIS_STATUS`, `JENIS_SUPPLIER`,
`STS_AKTIF_PROMLIST`, `STS_AUTOPAYMENT` — dirender `pxDropdown` bersumber `associated`,
artinya daftar pilihannya hidup di rule **Field Value**. Tidak satu pun rule Field Value
ikut di export (`R-16`).

Ada tiga jalan, dan hanya satu yang tidak menebak:

| Jalan | Verdict |
|---|---|
| Mengarang daftarnya | **DITOLAK** — nilai yang salah tersimpan ke master yang menentukan ke rekening siapa uang berpindah, dan salahnya tidak terlihat di layar mana pun |
| Mengubah isiannya menjadi teks bebas | **DITOLAK** — layar lamanya dropdown, dan salah ketik pada kolom bersandi tersimpan diam-diam |
| Menawarkan nilai yang BENAR-BENAR ADA di data | **dipakai** |

Keterbatasannya dinyatakan, bukan disembunyikan: nilai sah yang belum pernah dipakai satu
baris pun tidak akan muncul. Karena itu daftarnya **digabung** dengan tiga sandi yang
artinya terbukti dari percabangan activity:

| Sandi | Buktinya |
|---|---|
| `JENIS_STATUS` `"1"`/`"0"` | `CreateNewMasterSupplier_post` step 7 dan `GetDataSupplier_pre` step 6.3 saling membalik keduanya |
| `STS_AKTIF_PROMLIST` `"1"`/`"0"` | step 6 menetapkan `"0"` pada supplier baru; step 12 hanya meminta persetujuan bila `"1"` atau kosong |
| `STS_AUTOPAYMENT` | isiannya `pxCheckbox`, sehingga hanya punya dua keadaan |

`STS_REKANAN` dan `JENIS_SUPPLIER` **sengaja tidak diberi nilai dasar**. Master Bengkel
punya bukti bahwa `STATUS_REKANAN = "0"` berarti bukan rekanan — tetapi itu kolom tabel
**lain** pada modul lain, dan memindahkan artinya ke sini berarti mengandaikan kedua master
memakai sandi yang sama. Tidak ada satu pun bukti untuk itu.

Penggabungannya ada di **lapisan aplikasi**, bukan di adapter: adapter menjawab "apa yang
ada di data", dan yang memutuskan bagaimana itu dilengkapi adalah aturan. Dijaga
`TestListCodesOnEmptyStoreStillOffersProvenValues` — tanpa penggabungan, basis data yang
masih kosong akan menyajikan lima dropdown tanpa satu pun pilihan, dan supplier pertama
tidak akan pernah dapat ditambahkan.

Label sandi yang hanya ditemukan di data berisi **sandinya sendiri**, bukan tebakan artinya.
Menampilkan tebakan yang tampak meyakinkan lebih buruk daripada menampilkan sandinya apa
adanya.

### 23.7 `LookupPicker` bersama — syaratnya kini TERPENUHI, dan tetap tidak dikerjakan

`master-bengkel/CityPicker.tsx` mencatat syaratnya sendiri:

> *"begitu ada modul KETIGA yang membutuhkan kotak cari–pilih, ketiganya dipindahkan
> sekaligus ke `components/LookupPicker` dalam satu perubahan."*

**Modul ketiga itu adalah modul ini** — setelah `master-auto-claim/LookupPicker` dan
`master-bengkel/CityPicker`. Syaratnya terpenuhi.

Yang menahannya bukan lagi syarat itu melainkan **Isolasi Protektif**: menaikkannya menuntut
menyunting dua modul master yang sudah dinyatakan selesai. Perubahan itu layak dijadwalkan
Work Owner sebagai satu pekerjaan tersendiri, bukan diselipkan ke dalam penambahan modul
ini.

Sampai itu dijadwalkan, modul ini punya `CityPicker`-nya sendiri, dan berkasnya menyatakan
keadaan itu terang-terangan alih-alih mendiamkannya.

### 23.8 Kota dipilih lewat cari–pilih, padahal layar lama memakai dropdown

`Section/CreateMasterSupplier_Sec` memang memasang `pxDropdown` atas `BrowseCity_RD`, dan
report definition itu **tidak membatasi hasilnya sama sekali** (`pyMaxRecords=0`) — seluruh
tabel `CITY` yang berbaris ribuan dimuat ke klipboard, lalu disaring di peramban.

Menirunya berarti mengirim ribuan baris untuk satu pilihan, pada setiap kali form dibuka.
Yang ditiru karena itu adalah **hasilnya** — petugas memilih satu kota dari master kota —
bukan cara memuatnya.

Cabang, Negara, dan Bank tetap dropdown biasa: ketiganya daftar pendek, persis seperti
layar lama.

### 23.9 Yang disimpan adalah NAMA, bukan kode

Dokumen supplier tidak punya `KOTA_ID`, `CABANG_ID`, maupun `BANK_ID` — kueri bacanya hanya
menyebut `KOTA`, `NAMA_CABANG`, `NEGARA`, dan `BANK`. Itu berbeda dari Master Bengkel, yang
menyimpan kode dan nama berpasangan.

Akibat yang harus disadari: bila sebuah cabang berganti nama, baris supplier yang menyebut
nama lamanya **tidak ikut berubah** dan tidak lagi cocok dengan satu pun pilihan di
dropdown. Itu keadaan sistem lama; membetulkannya menuntut kolom yang tabelnya tidak punya
— perubahan skema (`D-63`), bukan keputusan modul ini.

Yang dikerjakan di sini: nilai tersimpan **selalu ditampilkan apa adanya**, supaya
menyunting baris lama tidak diam-diam mengosongkan cabang atau kotanya.

### 23.10 Nama terkunci — ditegakkan server, bukan hanya layar

`pyReadOnlyCondition` pada isian NAMA adalah satu-satunya syarat read-only di seluruh form
itu. Layar Pega menegakkannya dengan mengunci isiannya, **dan hanya itu**.

Server ikut memeriksanya, dan perbandingannya **mengabaikan besar-kecil huruf dan spasi di
ujung**: yang dilarang adalah mengganti namanya, bukan mengirimkannya kembali dengan ejaan
yang sedikit berbeda. Layar mengirim nilai yang dimuatnya apa adanya, tetapi klien lain
belum tentu.

Galatnya dibedakan dari "nama sudah dipakai" (`nama_supplier_terkunci` versus
`nama_supplier_sudah_ada`) karena tindakan perbaikannya berbeda: yang pertama menuntut
membatalkan dan memuat ulang, yang kedua menuntut nama lain.

### 23.11 Pemeriksaan nama ganda DITAMBAHKAN

Berbeda dari Master Bengkel yang punya `ValidationMasterBengkel`, **tidak ada satu pun rule
di export yang memeriksa nama supplier ganda.**

Ia ditambahkan karena layar sendiri memperlakukan nama sebagai kunci alami: sekali
tersimpan, isiannya terkunci dan tidak dapat diperbaiki lagi. Gabungan keduanya berarti dua
supplier bernama sama di sistem lama akan hidup selamanya tanpa satu pun cara membedakannya
dari layar.

**Selisih yang direncanakan:** penambahan yang dulu diterima kini ditolak, dan baris lama
dibaca apa adanya.

Penutupnya yang sebenarnya adalah constraint unik, dan di modul ini ia lebih jauh daripada
di modul lain: **namanya tersimpan di dalam dokumen JSON**, sehingga constraint unik atasnya
menuntut index berbasis fungsi lebih dulu.

### 23.12 Menonaktifkan supplier TIDAK melewati persetujuan

`EditMasterSupplier_post` step 12 memasang prasyarat:

```
@String.equals(STS_AKTIF_PROMLIST,"1") || @String.equals(STS_AKTIF_PROMLIST,"")
```

Artinya **menonaktifkan sebuah supplier tersimpan langsung tanpa persetujuan siapa pun**,
sedangkan mengaktifkannya harus menunggu.

Itu tampak disengaja dan arahnya masuk akal — menutup kerja sama tidak perlu izin,
membukanya perlu — sehingga ditiru apa adanya. Nilai kosong ikut diterima persis seperti
prasyaratnya, meski form mewajibkan isian itu: permintaan yang tidak datang dari form dapat
mengirimkannya kosong, dan Pega memperlakukannya sebagai "perlu persetujuan".

Ia **satu-satunya jalur di modul ini yang mengubah keadaan tanpa melewati antrean mana
pun**, dan tiga hal menjaganya tetap terlihat: log menyertakan penanda
`persetujuan_diminta`, layar menyatakannya di bawah tombol Simpan, dan
`TestSaveSkipsApprovalWhenDeactivating` gagal bila perilakunya berubah.

### 23.13 `TGL_INSERT` tetap teks `dd/MM/yyyy`

Kedua activity penyimpan menulisnya lewat
`@DateTime.FormatDateTime(…,"dd/MM/yyyy","in_ID","Asia/Jakarta")`. Tanggal sebagai teks
tidak dapat diurutkan, tidak dapat disaring sebagai rentang, dan tidak memakai index —
persis utang yang `09-DATABASE-STRATEGY.md` §3.2 catat.

Ia **dipertahankan**, dan hanya di dalam dokumen yang dibagi dengan Pega: menyimpannya
sebagai waktu yang benar akan membuat layar Pega membaca sesuatu yang berbeda dari yang
ditulisnya sendiri (`D-21`).

Baris permintaan persetujuan, yang **tidak dibaca layar mana pun**, memakai waktu sungguhan
— tidak ada yang mengikatnya.

Konversinya diisolasi di `timestamp.go`, satu berkas untuk seluruh modul, sesuai
`08-TECHNICAL-STRATEGY.md` §4.4. `TestFormatJakartaDateCrossesMidnight` menjaga kasus yang
`R-12` catat: pukul 17:30 UTC sudah pukul 00:30 WIB keesokan harinya, dan mengambil tanggal
dari waktu UTC akan menuliskan tanggal **kemarin** pada setiap penyimpanan selepas pukul
lima sore waktu Jakarta.

### 23.14 Baris permintaan membawa `Supplier`, bukan teks JSON

`ApprovalRequest.Snapshot` bertipe `Supplier`, bukan dokumen yang sudah dirangkai.
Alasannya batas lapisan: **bentuk dokumen JSON adalah urusan penyimpanan**, dan lapisan
aplikasi tidak boleh mengetahuinya.

Yang merangkainya adalah adapter yang sama dengan yang merangkai dokumen master, sehingga
keduanya tidak pernah dapat berbeda bentuk.

### 23.15 Utang teknis yang ditambahkan sesi ini

| Utang | Rencana penyelesaian |
|---|---|
| Kode galat modul dipetakan di modul sendiri | pindah ke tempat bersama begitu `TKT-F1-004` diputuskan |
| Jalur `/api/master/supplier` tanpa awalan `/v1` | penyeragaman seluruh aplikasi, bukan sepihak di satu modul |
| Batas panjang diulang di backend dan `SupplierForm.tsx` | dijaga `TestLengthLimitsAreStable`; hilang begitu DDL diterima (`R-08`) |
| `CityPicker` ketiga yang belum dinaikkan ke `components/` | satu pekerjaan tersendiri yang menyentuh tiga modul; lihat §23.7 |
| Permintaan persetujuan di luar transaksi penyimpanan | menuntut transaksi yang dipegang lapisan aplikasi — bentuk seam yang menyentuh seluruh modul master |
| Bentuk `PROTEKSI_ID` buatan sendiri | menunggu keputusan Work Owner; pilihan lain menuntut objek basis data baru (`D-63`) |
| `ViolationDTO` memakai `kolom`, sementara Master Pasal memakai `field` | keduanya sah hari ini karena klien membaca keduanya; penyeragamannya `TKT-F1-004` |

### 23.16 Paginasi: opt-in di komponen bersama, angkanya milik layar

Ditambahkan menyusul pada hari yang sama, atas permintaan *"buatkan pagination sesuai
pega"*.

#### Kenapa ukuran halaman menjadi prop, bukan angka di dalam komponen

Karena di sistem lama pun ia berbeda-beda. Setelannya dibaca dari `pyPageSize` — atau dari
`pyPageSizeOther` bila nilainya `"Other"` — pada `pyGridProps` masing-masing section:

| Layar | Ukuran | Mode |
|---|---|---|
| Master Supplier · Master Bengkel · Master Panel | **20** | Numeric |
| Master Rekening · Status Klaim · Status Progres · Pasal Kerugian · Penolakan Klaim | **15** | Numeric |
| Master Auto Claim | — | tanpa paginator |

Tidak ada satu angka yang benar untuk semuanya. Menaruh salah satunya sebagai bawaan
komponen berarti delapan layar lain memakai angka yang bukan angkanya.

#### Kenapa opt-in, dan bukan menyala secara bawaan

`DataTable` dipakai **sembilan layar master yang sudah dinyatakan selesai**. Menyalakan
paginasi sebagai bawaan mengubah kesembilannya sekaligus — Master Status Klaim yang berisi
33 baris akan langsung terpotong dua halaman, beserta ujinya yang gagal.

Itu melanggar **Isolasi Protektif**, dan pelanggarannya tidak diperlukan: satu prop
opsional menjawab kebutuhan layar ini tanpa menyentuh yang lain.

```
pageSize?: number     tidak diisi  →  tanpa paginasi, persis seperti sebelumnya
                      diisi 20     →  paginasi numerik, 20 baris per halaman
```

Yang menjaganya bukan niat melainkan uji: `tanpa paginasi > menggambar SELURUH baris bila
pageSize tidak diisi` berada **di berkas uji komponen**, sehingga perubahan bawaan
kelak gagal di sana — bukan di uji milik modul yang sudah selesai.

#### Kenapa di komponen bersama, dan bukan di layar Supplier saja

Karena itu alasan `DataTable` ada. `07-TECHNICAL-STRATEGY.md` §3 aturan 2 menetapkan
seluruh tabel memakai satu komponen, dan `06-MODULE-BREAKDOWN.md` menyebut `U-2` sebagai
investasi frontend terpenting justru karena 268 grid sistem lama memakai pola yang sama.

Menulis paginator di dalam `SupplierPage` berarti paginator kesepuluh menyusul saat layar
berikutnya membutuhkannya — persis kegagalan yang `D-09` khawatirkan untuk tim yang sedang
belajar React.

`components/` sendiri **bukan** modul yang dilindungi Isolasi Protektif; yang dilindungi
adalah modul Login, Home, dan Master Data. Perubahan di sini tetap dijaga agar aditif:
satu prop opsional, satu fungsi terekspor, satu komponen dalam berkas yang sama.

#### Paginasi berjalan SESUDAH pencarian dan pengurutan

Urutannya menentukan. Memotong halaman lebih dulu akan membuat pencarian hanya menemukan
baris yang kebetulan ada di halaman yang sedang dibuka — cacat yang tidak terlihat sebagai
galat, hanya sebagai hasil pencarian yang salah.

Urutan yang dipakai juga yang ditiru: grid Pega terikat pada page list klipboard
(`pyPageListProperty = ListMasterSupllier.pxResults`), sehingga paginatornya memotong
daftar yang **sudah** tersaring — bukan meminta halaman berikutnya ke server.

Dua akibat yang ikut diurus, keduanya mengembalikan posisi ke halaman pertama:

| Peristiwa | Alasan |
|---|---|
| kata kunci berubah | halaman ketiga daftar lama hampir pasti tidak ada pada daftar baru |
| urutan berubah | mengurutkan ulang menyusun ulang seluruh daftar; halaman ketiga tidak lagi memuat baris yang sama |

#### Halaman dijepit saat menggambar, bukan disetel lewat efek

Baris dapat berkurang di luar kendali komponen — penyaring dipersempit, atau daftarnya
dimuat ulang setelah sebuah baris berpindah. Halaman yang sudah tidak ada karena itu
menampilkan **halaman terakhir**, bukan tabel kosong tanpa penjelasan.

Dikerjakan dengan `Math.min` saat menggambar. `useEffect` yang menyetel state akan
menghasilkan satu gambar perantara yang kosong lebih dulu, dan itu terlihat sebagai
kedipan.

#### Dua hal yang DITAMBAHKAN terhadap Pega

| Tambahan | Alasan |
|---|---|
| **Ringkasan "Menampilkan 1–20 dari 57 baris"** | Pega hanya menggambar nomor halamannya. Tanpa ringkasan itu, nomor halaman tidak memberi tahu seberapa banyak yang belum dilihat — pada tabel yang baru disaring, justru itulah yang ingin diketahui |
| **Sela `…` pada daftar panjang** | daftar seribu baris menggambar lima puluh tombol nomor yang membungkus beberapa baris, dan justru membuat halaman yang sedang dibuka sulit ditemukan. Aturannya di `pageWindow`, diekspor supaya dapat diuji terpisah dari komponennya |

Pencacah "Total Data :" milik `Section/DataCountMasterSupllier` tetap ada di kepala layar
dan menghitung hal yang **berbeda**: seluruh baris yang dimuat, bukan yang tersaring.
Keduanya sengaja dibiarkan berdampingan.

#### Yang dijaga untuk pembaca layar

| Hal | Cara |
|---|---|
| Berpindah halaman diumumkan | ringkasan barisnya `role="status"` — diumumkan tanpa memindahkan fokus |
| Halaman yang sedang dibuka | `aria-current="page"`, bukan hanya warna |
| Bilah nomornya dapat dilompati | `<nav aria-label="Halaman tabel">` |
| Tombol panah punya nama | `aria-label` eksplisit; ikonnya `aria-hidden` dan tidak pernah menjadi satu-satunya penanda |

`aria-current` bukan pelengkap: warna saja tidak terbaca pembaca layar, dan tidak
terbedakan oleh sekitar satu dari dua belas laki-laki yang mengalami buta warna
merah-hijau.

#### Yang TIDAK dikerjakan

**Paginasi keyset sisi server.** Itu `TKT-U2-001`, dan Steering menyebutnya **perubahan
perilaku, bukan pemeliharaan**: grid lama memang memotong di klipboard, bukan memaginasi
di server. Layar berpuluh juta baris menuntutnya; layar master yang berbaris puluhan
tidak.

Endpoint `GET /api/master/supplier?cari=` sudah menerima penyaring, sehingga perpindahannya
kelak tidak menuntut kontrak baru — hanya menambah `cursor` dan `limit`.

**Menyalakan paginasi pada delapan layar master lain.** Setelan Pega-nya sudah terbaca dan
tercatat di tabel di atas, sehingga menyalakannya cukup satu prop per layar. Ia tidak
dikerjakan di sini karena menyentuh modul yang sudah dinyatakan selesai — keputusan Work
Owner, bukan keputusan sesi ini.

#### Utang yang ditambahkan

| Utang | Rencana penyelesaian |
|---|---|
| Delapan layar master memakai paginasi Pega-nya, tetapi belum menyalakannya | satu prop per layar; menunggu keputusan Work Owner karena menyentuh modul selesai |
| Ukuran halaman ditulis di layar, bukan dibaca dari konfigurasi | sistem lama pun menanamnya di section; mengangkatnya menjadi konfigurasi menuntut keputusan tersendiri (`D-15`) |

---

## 24. Koreksi Master Panel — layar diselaraskan dengan Pega (2026-09-20)

Ditulis sebagai bagian tersendiri, bukan disunting ke dalam §22, karena §22 merekam apa
yang **benar-benar diputuskan saat itu**. Yang berubah ditulis sebagai keputusan baru yang
menyebut butir yang disupersede — perlakuan yang sama dengan `00-DECISION-LOG.md`.

### 24.1 Apa yang memicunya

Work Owner mengirim **tangkapan layar Master Panel HE yang sebenarnya** beserta satu
kalimat: *"Saya ingin tampilannya konsisten dengan pega seperti ini."*

Empat hal pada layar yang dibangun ternyata menyimpang, dan seluruhnya **dapat dibuktikan
dari export** — bukan soal selera:

| Hal | Bukti export | Yang dibangun | Putusan |
|---|---|---|---|
| Judul | `pyCaption Master Panel HE` pada `ListPanelHE` | "Master Panel" | **diperbaiki** |
| Urutan tab | urutan section pada `BrowsePanelHE`: Approve → Reject → Waiting Approval | Approve → Waiting Approval → Reject | **diperbaiki** |
| Kolom grid | 11 kolom berdampingan, `pyLabelFieldValue` pada `BrowsePanelHEApproval` | 9 penanda diringkas jadi satu kolom "Penanda", ditambah kolom "Lokasi" | **diperbaiki** |
| Urutan baris | `pySortType=DESC` atas `ID_PANEL` | `ORDER BY NAME` | **diperbaiki** |

Ditambah tiga tombol yang memang ada di layar lama dan belum dibangun sama sekali:
`Upload Document`, `Upload Data Master Panel`, `Upload Data Lokasi Panel`
(`pyButtonLabel` pada `BrowsePanelHE`).

### 24.2 Kenapa keempatnya menyimpang, dan apa pelajarannya

Keempatnya berasal dari **satu kebiasaan yang sama**: memperbaiki tata letak yang dianggap
sulit dibaca, alih-alih menirunya.

Alasan yang dipakai saat itu tertulis apa adanya di kode:

> *"Grid Pega menampilkan kesebelas kolomnya sekaligus. Itu tidak ditiru seluruhnya:
> sembilan penanda berdampingan menjadi deretan sandi yang tidak dapat dibaca mata."*

Alasan itu **masuk akal sebagai desain, dan salah sebagai migrasi**. `D-13` tidak menuntut
tata letak yang lebih baik; ia menuntut tata letak yang **sama**, supaya pengguna tidak
perlu belajar ulang. Petugas yang setiap hari memindai kolom `STATUS PECAH` pada posisi
yang sama akan kehilangan kebiasaannya begitu kolom itu dilipat ke dalam ringkasan.

**Pelajarannya, dan ia berlaku untuk modul berikutnya:** penyimpangan tata letak menuntut
alasan yang lebih kuat daripada "lebih mudah dibaca". Yang layak menjadi alasan hanyalah
hal yang **tidak dapat dikerjakan** — rule yang hilang, data yang tidak ada, atau perilaku
yang berbahaya. Urutan tab dan susunan kolom bukan salah satunya.

Yang sebelumnya dicatat sebagai *"selisih yang direncanakan, dan satu-satunya yang tampak
di layar daftar"* karena itu **dicabut**. Modul ini sekarang tidak punya satu pun selisih
tata letak terhadap Pega.

### 24.3 Yang berubah

| Berkas | Perubahan |
|---|---|
| `PanelPage.tsx` | judul → "Master Panel HE"; urutan tab → Approve · Reject · Waiting Approval; 11 kolom grid dikembalikan berdampingan; kolom "Lokasi" dan "Penanda" dibuang; tiga tombol unggah ditambahkan dalam keadaan mati; `LocationSummary` dihapus |
| `masterpanel.sql` | `ORDER BY NAME` → `ORDER BY ID_PANEL DESC` pada `panel_list` dan `panel_list_search` |
| `repo/memory/memory.go` | pengurutan memori mengikuti, supaya urutan saat pengembangan sama dengan produksi |
| `query_test.go` | `TestListOrderFollowsPega` — mengunci arah urutan |
| `manage_test.go` | `TestListOrderIsIDDescending` — mengunci hal yang sama di adapter memori |
| `PanelPage.test.tsx` | uji judul, kesebelas kolom, urutan tab, ketiadaan kolom Lokasi, dan ketiga tombol unggah |

### 24.4 Tiga tombol unggah: ditampilkan MATI, bukan dihilangkan

Rule yang menjalankan ketiganya **tidak ikut di export** (`R-16`), sehingga tidak ada
perilaku yang dapat ditiru.

Tiga pilihan ditimbang:

| Pilihan | Akibat |
|---|---|
| dihilangkan | petugas mengira fiturnya hilang, dan tidak ada apa pun di layar yang menjelaskannya |
| ditampilkan hidup | tombol yang tidak melakukan apa-apa — kelas cacat yang paling merusak kepercayaan |
| **ditampilkan mati beserta alasannya** | kemajuan migrasi terbaca langsung dari layar |

Yang ketiga dipakai, mengikuti preseden yang sudah ada: butir menu yang belum punya layar
pun **tetap tampil**, tidak dapat diklik, dan bertanda "belum tersedia" — keputusan Work
Owner 2026-09-18.

### 24.5 Yang TIDAK ikut berubah, dan kenapa

| Hal | Alasan |
|---|---|
| **Nilai penanda tetap ditampilkan sebagai sandi** (`1`, `0`) | menggantinya dengan "Ya"/"Tidak" berarti menebak domain sembilan kolom yang daftar nilainya tidak ada di export (`R-16`) — dan menebak di tempat yang paling terlihat. Pega pun menampilkannya apa adanya |
| **Kolom "Pilih" pada tab Waiting Approval** | ia bukan tambahan: `Section/ApprovalMasterPanelHE` punya kolom `pyCaption Pilih` yang sama. Yang berpindah hanyalah letaknya — dari Inbox Manager yang belum dibangun ke layar ini |
| **Daftar lokasi dikelola di dalam form** | grid Pega memang tidak punya kolom lokasi; lokasi hanya muncul saat sebuah panel dibuka |
| **Tombol "Ubah" berupa tombol, bukan tautan** | Pega menggambarnya sebagai tautan, tetapi `components/Button` adalah komponen baku yang dipakai sembilan layar master lain. Menyimpang di satu layar akan membuat aplikasinya terlihat dirakit dari dua tempat — dan katanya tetap sama |

### 24.6 Hasil pemeriksaan sesudah koreksi

| Perintah | Hasil |
|---|---|
| `go build ./...` · `gofmt -l internal/masterpanel` | hijau / bersih |
| `go test -count=1 ./internal/masterpanel/...` | **hijau**, keempat paket |
| `npm run typecheck` | **hijau** |
| `vitest run src/modules/master-panel` | **hijau**, 18 uji (naik dari 16) |

---

## 25. Koreksi Master Status Progres 1 — nilai Posisi dan paginasi (2026-09-20)

Tiga keputusan, seluruhnya lahir dari umpan balik Work Owner atas layar yang sudah
dibangun. Rinciannya di `catatan-pengembangan.md` §24.

### 25.1 Nilai yang tersimpan di kolom STATUS adalah TEKS, bukan kode angka

**Keputusan.** `GCNM_MST_PROGRESS_KLAIM.STATUS` menyimpan teks posisinya —
`"REGISTER"`, `"KOMITE"`, `"SURVEY"`, `"AKSEPTASI"`, `"OUTSTANDING"`, `"BENGKEL"`,
`"PROCUREMENT"`, `"All"` — bukan kode `"002"`/`"004"`/`"006"`/`"007"` seperti yang
ditulis versi sebelumnya.

**Ini mencabut keputusan §10 pada berkas ini** yang menetapkan pasangan kode→label.
Entri §10 **tidak disunting** — ia merekam apa yang diputuskan saat itu beserta alasannya,
dan mengubahnya akan menghapus jejak bahwa kesimpulannya pernah keliru. Aturan yang sama
dipakai `00-DECISION-LOG.md` terhadap `D-01`…`D-30`.

**Dasarnya.** Sel "Posisi" pada `Section/BrowseStatusProgress-Section.xml` mengikat
`pyValue` **dan** `pyPrompt` ke properti yang sama (`.CaseID`), dan
`Activity/ViewStatusProgress_act-Act.xml` mengisi `.CaseID` dengan literal teks
(`"AKSEPTASI"`). Nilai simpanan dan labelnya karena itu satu dan sama.

**Kenapa ini penting melebihi "daftar yang kurang panjang".** Menyimpan `"002"` ke kolom
yang di seluruh sistem lama berisi `"REGISTER"` **tidak menimbulkan galat apa pun** —
barisnya tersimpan, layarnya tampak benar, dan kegagalannya baru terlihat ketika ada yang
membandingkan nilainya. Ia kelas cacat yang sama dengan `R-20`: salah, tetapi senyap.

### 25.2 Daftar posisi tetap di kode, dan sumbernya tangkapan layar

**Keputusan.** Kedelapan nilai tetap hidup sebagai daftar bernama di lapisan domain
(`position.go`) dan disajikan lewat API — **tidak** disalin ke frontend, **tidak** dibuat
sebagai tabel baru.

**Dasarnya tidak berubah dari §10.3:** tidak ada tabel master posisi di sistem lama yang
dapat dibaca, dan membuatnya menuntut permintaan perubahan skema tertulis, persetujuan
Work Owner, serta pelaksanaan DBA (`D-63`) — modul ini akan terhalang sampai itu selesai.
Mengarang tabel beserta isinya berarti menebak.

**Yang baru: sumber daftarnya.** Rule yang mengisi `TempPosition` untuk harness ini tidak
ada di export — sudah dicari ke `Activity/`, `Section/`, `RDB List/`, dan
`Data Transform/`, dan kedelapan nilainya tidak pernah muncul bersama di satu tempat mana
pun (`R-16`). Karena artefaknya memang tidak ada, **tangkapan layar Pega yang sedang
berjalan adalah sumber yang sahih**, dan itu diterima sebagai dasar — bukan ditebak dan
bukan ditunda.

**Utang teknis yang tetap tercatat.** Daftar ini seharusnya menjadi master data milik
`F-4` sesuai `D-15`, dapat diubah tanpa merilis ulang aplikasi. Selama masih di kode,
menambah posisi berarti mengubah kode.

### 25.3 Bentuk kontrak API sengaja tidak diubah

**Keputusan.** `kode_posisi` dan `nama_posisi` **tetap dua field**, meskipun isinya kini
selalu sama. Begitu pula `kode` dan `nama` pada daftar posisi.

**Yang dipertimbangkan dan tidak diambil:** menyatukannya menjadi satu field `posisi`.
Ia lebih jujur terhadap keadaan hari ini, tetapi harganya perubahan yang merusak klien
demi penamaan yang lebih rapi. Dan pemisahannya tetap berguna: bila daftarnya kelak
pindah menjadi master data `F-4`, label yang dibaca pengguna dapat dibedakan dari nilai
yang tersimpan tanpa mengubah kontrak sama sekali.

**Yang berubah sebagai gantinya:** keterangan pada `dto.go` menyatakan terang-terangan
bahwa keduanya bernilai sama pada layar ini, beserta sebabnya. Dan kolom Posisi di layar
tidak lagi menyandingkan label dengan kodenya — menyandingkan dua nilai yang identik
hanya menulis "REGISTER REGISTER".

### 25.4 Pembakuan ejaan pindah ke Input.Clean

**Keputusan.** `Input.Clean` **tidak lagi meng-huruf-besarkan** nilai posisi. Ia
membakukan ejaannya ke baris daftar yang cocok lewat `FindPosition`, dan membiarkan nilai
yang tidak dikenal apa adanya supaya `Check` yang menolaknya — bukan `Clean`, agar
pesannya sampai ke pengguna.

**Sebabnya satu nilai:** `"All"` ber-ejaan campuran. `strings.ToUpper` akan menyimpan
`"ALL"`, yang tidak sama dengan apa pun yang pernah ditulis sistem lama.

**Akibat sampingan yang diinginkan:** tabel tidak lagi dapat terisi `"All"`, `"ALL"`, dan
`"all"` sekaligus. Pembakuan itu terjadi di **satu tempat**, bukan diulang di setiap
pemanggil.

### 25.5 Paginasi 15 baris dinyalakan

**Keputusan.** Grid Master Status Progres 1 berhalaman **15 baris**, sesuai
`pyPageSizeOther = 15` pada `Section/BrowseStatusProgress-Section.xml`.

Angkanya milik **layar**, bukan milik komponen — `DataTable.pageSize` sudah dirancang
opt-in karena layar lain memakai 20. Yang diperbaiki hanyalah layar ini tidak pernah
menyalakannya.

**Uji baru menjaganya.** Tanpa uji, prop satu baris seperti ini hilang tanpa jejak pada
penyuntingan berikutnya — dan hilangnya tidak menimbulkan galat, hanya mengembalikan
perilaku yang salah.

### 25.6 Aturan kerja yang ditambahkan untuk diri sendiri

**Sebelum memakai sebuah daftar nilai dari export, periksa dulu ikatan `pyValue` sel yang
memakainya** — bukan hanya menelusuri dari mana nilainya berasal.

Kesalahan ini berasal dari membaca deretan angka di sebelah label pada activity yang
benar, lalu menyimpulkan keduanya berpasangan untuk layar ini. Polanya sama dengan `R-06`
pada Steering: tiga arti kode status yang disimpulkan dari pemakaian ternyata seluruhnya
salah. Aturannya sudah ditulis sebagai komentar di `position.go`, bukan hanya di sini.

### 25.7 Kedua pertanyaan §25.1–§25.2 dijawab: "Sesuaikan saja dengan Pega"

Jawaban Work Owner, 2026-09-20. Rincian buktinya di `catatan-pengembangan.md` §24.9.

**Keputusan A — daftar posisi berisi sembilan baris, delapan nilai berbeda.** "All"
direplikasi dua kali persis seperti layar lama. `pyHasNoSelection = false` pada sel
dropdown membuktikan Pega tidak menyisipkan baris kosong, sehingga pengulangan itu ada di
datanya sendiri — bukan baris prompt yang salah terbaca. Pengulangan tidak mengubah apa
yang tersimpan, sehingga biayanya nol dan yang diperoleh kesetaraan tampilan (`D-13`).

**Keputusan B — nilainya "PROCUREMENT", tanpa "/SUPPLIER".** Kekhawatiran §24.8 butir 2
**dicabut dengan bukti**: `'PROCUREMENT/SUPPLIER'` hanya dibandingkan terhadap
`.AlasanTerlambat` dan `.SurveyorAddrress` di tiga berkas, dan tidak pernah terhadap
kolom `STATUS` milik master ini. Tidak ada perubahan yang diperlukan.

**Keputusan C — `SelectField` mengunci option dengan posisi, bukan nilai.** Konsekuensi
langsung dari Keputusan A: daftar kini boleh memuat nilai kembar, dan kunci berbasis nilai
membuat React menemui dua kunci yang sama. Perubahannya **netral terhadap perilaku** —
urutan daftar ditentukan server dan tidak pernah disusun ulang di layar — dan seluruh uji
sembilan layar yang memakai komponen ini tetap hijau.

**Penyimpangan yang sengaja dipertahankan.** `SelectField` tetap menyisipkan baris kosong
`— pilih —`, sedangkan Pega tidak dan langsung memilih baris pertama. Tanpa baris kosong,
penambahan baru akan **menyimpan "All" tanpa pengguna pernah memilihnya** — nilai yang
tidak pernah ia tentukan, tersimpan tanpa satu pun tanda. Penyimpangan ini disebut
terang-terangan di sini supaya ia menjadi pilihan yang tercatat, bukan temuan berikutnya;
mengembalikannya ke perilaku Pega adalah satu prop di layar, bukan perubahan komponen.

---

## 26. Koreksi Master Auto Claim — grid diselaraskan dengan Pega (2026-09-20)

### 26.1 Yang dikoreksi Work Owner

Grid layar Master Auto Claim **menggabungkan kolom yang di Pega terpisah**. Laporannya
menyebut satu contoh — bank dan nomor rekening — dan pemeriksaan menemukan empat.

| Versi pertama | Pega |
|---|---|
| `Bank & rekening` | `BANK PENERIMA` · `NO REKENING` |
| `PIC lapor` (PIC + email) | `EMAIL LAPOR` · `PIC` |
| `Nama penerima` (+ client) | `NAMA PENERIMA`; client tidak ada di grid |
| `Dipakai` | tidak ada |
| — | `ALAMAT PENERIMA` **hilang** dari layar |

Penyebabnya bukan salah baca sumber: komentar di atas kode itu sudah memuat kesembilan header
beserta alias dan kolomnya. Yang terjadi adalah tampilan "dirapikan" saat menulis JSX tanpa
membandingkan hasilnya kembali dengan komentar yang baru saja ditulis.

`D-13` tidak menyisakan ruang untuk itu: alur dan tata letak ditiru supaya pengguna tidak perlu
belajar ulang, dan penggabungan kolom adalah perubahan tata letak.

### 26.2 Kolom mengikuti tab, karena di Pega memang begitu

| Tab | Kolom |
|---|---|
| Master Auto Klaim · Waiting Approval · Reject | 9 |
| **Komite Approval** | **8** — tanpa KOMITE |

Di tab Komite seluruh barisnya memang milik pemanggil, sehingga kolom itu tidak memberi tahu apa
pun. Lebar kolomnya diambil apa adanya dari section: `72 · 177 · 167 · 105 · 139 · 163 · 60 ·
90 · 115 · 93` piksel.

### 26.3 Approve dan Reject berada di FORM, bukan di baris grid

Versi pertama menaruh keduanya di setiap baris tab Komite. Letaknya di Pega berbeda, dan itu
terbaca dari posisinya di dalam berkas section:

| Elemen | Offset |
|---|---|
| isian form | 59.116 |
| tombol `stsapprove` | 152.789 – 165.855 |
| header grid | 232.228 |
| tombol `INISIAL` (Update per baris) | 279.901 |

Alur Pega karena itu: **Update pada baris → isian termuat → komite memutuskan atas isi yang
benar-benar dilihatnya.** Baris grid hanya punya satu tombol, di keempat tab.

Letak itu pula yang menjelaskan cacat pada §19.4: komite yang **tidak** memuat barisnya lebih
dulu mengirim page form kosong, dan `UPDATE` menulis `CLIENTID`/`CLIENTNAME` kosong apa adanya.
Penutupnya tidak berubah — nilai yang dikirim tetap berasal dari baris yang dimuat.

### 26.4 Tombol form berbeda per tab, dan satu tab baca saja

| Tab | Tombol form |
|---|---|
| Master Auto Klaim · Reject | Simpan (`stsapprove="0"`) |
| Komite Approval | Approve · Reject — **tanpa Simpan** |
| **Waiting Approval** | **tidak ada** |

Tab Waiting Approval hanya memanggil `UpdateMstAutoClaim_act1`, yang memuat baris ke form, tanpa
satu pun activity penyimpan. Ia layar **baca saja**: yang memutuskan adalah komite yang ditunjuk,
lewat tabnya sendiri. Isiannya dinonaktifkan supaya keadaan itu terlihat, bukan hanya terasa saat
pengguna mencari tombol yang tidak ada.

### 26.5 Keterangan `CLAIM_ALLOWED` dipindahkan, bukan dibuang

Kolom `Dipakai` dibuat karena keadaan "disetujui tetapi `CLAIM_ALLOWED` bukan 1" tidak terlihat
di layar mana pun — di Pega maupun di sini — padahal ia membuat sebuah baris tidak pernah dipakai
pembuatan klaim otomatis.

Alasannya masih berlaku; tempatnya yang salah. Ia sekarang **catatan di atas tabel**, muncul
hanya bila barisnya ada, menyebut inisialnya. Grid tetap mengikuti Pega kolom per kolom.

| Pilihan | Akibat |
|---|---|
| Kolom tambahan | grid tidak lagi sama dengan Pega — ditolak |
| Dihapus sama sekali | keadaan yang membuat baris "mati" kembali tidak terlihat siapa pun |
| **Catatan di luar grid** | keduanya terpenuhi |

Bila Work Owner lebih suka keterangan itu hilang, menghapusnya satu blok — dan mode `-periksa`
tetap melaporkan angkanya.

### 26.6 Dua perbedaan yang TETAP ada terhadap tangkapan layar

| Hal | Alasan |
|---|---|
| Tanpa ikon saring (▼) per kolom | `DataTable` bersama menyediakan satu kotak pencarian dan pengurutan per kolom. Penyaring per kolom mengubah komponen yang dipakai SELURUH layar master (`U-2`) |
| Header kolom aksi kosong | mengikuti Pega. Sel judul kosong tidak menamai kolomnya bagi pembaca layar; yang menutupinya adalah tombol di dalam tiap sel, yang bernama "Update" |

### 26.7 Uji yang menjaga koreksi ini

Tiga uji baru, dan yang pertama sengaja **menguji urutan sebagai senarai** — kolom yang benar
tetapi berpindah tempat pun gagal:

| Uji | Menjaga |
|---|---|
| `menampilkan sembilan kolom Pega pada urutan yang sama` | tidak ada kolom yang digabung, hilang, atau bertambah |
| `menaruh setiap nilai di selnya sendiri, tidak digabung` | ISI selnya, bukan hanya headernya |
| `menghilangkan kolom Komite pada tab Komite Approval` | perbedaan antartab |
| `baris grid hanya punya tombol Update, termasuk di tab Komite` | Approve/Reject tidak kembali ke baris |

Uji kedua ditambahkan setelah yang pertama ditulis, dan alasannya layak dicatat: **uji header
saja tidak cukup.** Render yang headernya benar tetapi tetap menggabungkan dua nilai ke dalam
satu sel akan lolos — dan itu persis bentuk kesalahan yang dikoreksi di sini.

Ditambah `tab Waiting Approval membuka form baca saja` dan `tidak menyediakan tombol Simpan di
tab Komite`. Seluruhnya **18 uji**, naik dari 14; backend tidak tersentuh.

## 27. Koreksi Master Bengkel — grid, urutan, dan paginasi diselaraskan dengan Pega (2026-09-20)

Work Owner mengirim tangkapan layar Pega yang sedang berjalan dan meminta kolom serta
paginasinya disesuaikan. Lima hal diubah, dan **kelimanya dikonfirmasi ulang ke export**
sebelum disentuh — tangkapan layar dipakai untuk menemukan selisihnya, bukan sebagai
sumber kebenarannya.

Alasannya bukan kehati-hatian berlebihan: gambar dapat terpotong di tepi kanan, dan
kolom ketujuh yang kebetulan tidak terlihat akan hilang tanpa ada yang menyadarinya.
`pyColumnCount = 7` menjawabnya tanpa keraguan.

### Grid enam kolom, bukan kolom rakitan

Yang sebelumnya saya buat adalah kolom **gabungan** — "Nama bengkel" dengan ID di
bawahnya, "Kota & cabang", "Kontak", "Diskon jasa / sparepart". Bentuk itu lazim pada
tabel modern dan memuat lebih banyak keterangan dalam ruang yang sama.

Ia tetap dicabut. `D-13` menuntut tata letak yang ditiru supaya pengguna tidak perlu
belajar ulang, dan kolom gabungan mengubah **apa yang dicari mata** di setiap baris.
Keempat kolom yang saya tampilkan — `NAMA_KABUPATEN`, `STATUS_REKANAN`, `NAMA_CABANG`,
`MAIL` — bahkan tidak muncul sama sekali di area grid Pega; seluruhnya hanya ada di form.

Pelajarannya layak dicatat: **"lebih informatif" bukan alasan yang sah** pada layar yang
sedang dimigrasikan. Yang dibandingkan pengguna adalah layar lama, bukan layar yang ideal.

### Satu keterangan yang tetap dipertahankan, dengan tempat yang berbeda

Kolom "Rekanan" hilang, tetapi penandaan bengkel **rekanan yang login aplikasinya kosong**
dipindahkan ke dalam sel Login Aplikasi.

Ia bukan hiasan. Di Pega, sel kosong pada kolom itu tidak dapat dibedakan antara "memang
tidak diberi login" (non-rekanan, dan itu benar) dan "seharusnya punya tetapi tidak"
(rekanan, dan bengkelnya tidak akan pernah dapat masuk). Perbedaannya tidak terlihat di
layar mana pun di sistem lama.

Keterangan itu muncul **hanya** pada keadaan kedua. Menandai keduanya akan membuat tandanya
tidak berarti apa-apa — dan itu yang dijaga uji
`tidak menandai bengkel non-rekanan yang tanpa login`.

### Urutan bawaan dipindahkan ke server, dan kenapa itu bukan kosmetik

`ORDER BY NAMA_BENGKEL` menjadi `ORDER BY ID_BENGKEL DESC`, meniru `pySortType=DESC`
`pySortOrder=1` pada kolom pertama grid.

Di Pega, pengurutan itu dikerjakan grid **di peramban** atas page list yang sudah dimuat —
kueri lamanya sendiri tidak punya `ORDER BY` sama sekali. Menaruhnya di server bukan
peniruan mekanismenya melainkan peniruan **hasilnya**, dan pada layar berpaginasi keduanya
tidak dapat dipisahkan: urutan menentukan baris mana yang ada di halaman berapa.

Kelas cacatnya patut disebut: urutan yang salah **tidak terlihat sebagai galat**. Halaman
pertama tetap terisi, jumlah barisnya tetap benar, dan angkanya tetap masuk akal — yang
berbeda hanya baris mana yang ada di halaman ketiga. `TestListQueriesOrderByKeyDescending`
yang menjaganya, dan repo memori ikut disesuaikan supaya urutan saat pengembangan sama
dengan di produksi.

Satu keterbatasan dicatat di berkas `.sql`: pengurutannya leksikografis bila kolomnya
bertipe teks, dan itu sama dengan urutan penerbitan **hanya selama lebar kunci tetap**.
Kunci hari ini berlebar tetap; bila nomor urut melampaui sepuluh digit, kunci tumbuh —
lihat `ComposeID`, yang sengaja tidak memotongnya — dan urutannya menyimpang tanpa
menimbulkan galat apa pun.

### Paginasi: tidak ada komponen bersama yang disentuh

`DataTable` sudah punya paginasi opt-in lewat prop `pageSize`, ditambahkan sesi
`masterpasal` yang berjalan bersamaan — dan doc comment-nya bahkan sudah mencatat "Master
Supplier · Master Bengkel → 20 baris". Yang dikerjakan hanyalah menyalakannya dengan satu
prop.

Angkanya tetap disebut di layar, bukan dijadikan bawaan komponen: `pyPageSize` berbeda per
section di sistem lama, dan layar master lain memakai 15.

### Dua tombol yang sengaja TIDAK digambar

"Upload Document" dan "Upload Data Master Bengkel" nyata di layar Pega. Rule di baliknya
tidak ikut di export (`R-16`), sehingga yang dapat digambar hanyalah tombol yang tidak
melakukan apa pun.

Itu lebih buruk daripada tidak menggambarkannya: petugas akan menekannya, tidak terjadi
apa-apa, dan yang dilaporkan adalah "aplikasinya rusak" — bukan "fiturnya belum ada".
Ketiadaannya sudah tercatat sebagai pekerjaan yang belum dikerjakan (§20.12).

---

## 28. Tiga tombol unggah Master Panel dihapus — menyupersede §24.4 (2026-09-20)

### 28.1 Keputusan

Ketiga tombol unggah pada layar Master Panel — **"Upload Document"**, **"Upload Data
Master Panel"**, dan **"Upload Data Lokasi Panel"** — **tidak digambar sama sekali**.

Ini **menyupersede §24.4**, yang memutuskan ketiganya ditampilkan dalam keadaan mati
beserta keterangan "belum tersedia". Keputusan Work Owner, 2026-09-20, sesudah alasan
ketiadaannya ditelusuri sampai ke rule-nya.

### 28.2 Apa yang ditemukan saat ditelusuri

Pertanyaan Work Owner — *"kenapa tombol upload belum tersedia?"* — dijawab dengan
penelusuran, bukan dengan mengulang klaim. Hasilnya:

| Tombol | Local action yang dipanggil | Ada di export? |
|---|---|---|
| Upload Document | `UploadDocument` | **tidak** |
| Upload Data Master Panel | `PNCUploadMasterPanelCSV` | **tidak** |
| Upload Data Lokasi Panel | `PNCUploadLokasiPanelCSV` | **tidak** |

Terbaca dari `<pyLocalAction>` pada `Section/BrowsePanelHE-Section.xml`. Pencarian rule
yang `pyRuleName`-nya persis ketiga nama itu, di seluruh 2.634 berkas: **nol**.

Dua di antaranya — `PNCUploadMasterPanelCSV` dan `PNCUploadLokasiPanelCSV` — **hanya
muncul di satu berkas di seluruh export**, yaitu section yang memanggilnya.

**Ini gap, bukan kategori yang memang tidak diekspor.** Folder `Flow Action/` berisi
**29 berkas**, dan salah satunya justru Flow Action unggah CSV yang serupa —
`PNCUploadDataKlaimSlikOJK-FA.xml`, ruleset GCNMFW yang sama, menunjuk activity
`PNCUploadAutoClaimSlikOJK` dan `pxUploadCSVResults`. Jadi Flow Action memang ikut
diekspor; ketiga milik Master Panel yang tidak ikut (`R-16`).

### 28.3 Yang tidak diketahui karenanya

Flow Action-lah yang menyebutkan form mana yang dibuka dan activity mana yang memprosesnya.
Tanpa ketiganya:

| Tidak diketahui | Akibatnya |
|---|---|
| **Susunan kolom CSV** | menebaknya berarti berkas diterima sistem tetapi kolomnya dipetakan ke tempat yang salah — dan itu tidak muncul sebagai galat |
| Validasi yang berjalan | apakah nama ganda ditolak, apakah sandi `STS_*` diperiksa |
| Activity yang menulis | jalur yang sama dengan Simpan, atau jalur sendiri |
| **Apakah baris hasil unggah masuk antrean persetujuan** | unggah massal yang melewati `APPROVAL="0"` adalah jalan pintas yang **memintas seluruh kontrol persetujuan** |

Butir terakhir yang membuat ketiganya tidak layak ditebak lalu dibangun: menebak salah di
sana berarti membuat pintu masuk data yang tidak diperiksa siapa pun.

### 28.4 Kenapa dihapus, bukan dibiarkan mati

§24.4 menimbang tiga pilihan dan memilih "ditampilkan mati", dengan alasan kemajuan migrasi
terbaca dari layar. Work Owner memilih yang pertama — **dihilangkan**.

Alasannya sah, dan ia mengoreksi timbangan §24.4 yang terlalu berat ke satu sisi: tombol
mati yang tidak pernah hidup selama berbulan-bulan **berhenti menjadi penanda kemajuan dan
berubah menjadi perabot**. Preseden "butir menu tetap tampil bertanda belum tersedia" yang
dipakai §24.4 sebenarnya tidak setara — butir menu itu menunjuk layar yang **sedang**
dibangun satu per satu, sementara ketiga tombol ini menunggu artefak dari pihak lain tanpa
tanggal.

Ketiadaannya tetap terbaca — bukan dari layar, melainkan dari tempat yang memang dibaca
saat orang mencarinya: `README.md`, komentar pada `PanelPage.tsx`, dan bagian ini.

### 28.5 Yang berubah

| Berkas | Perubahan |
|---|---|
| `PanelPage.tsx` | blok tiga tombol dihapus; diganti komentar yang menyebut ketiga local action beserta alasan ketiadaannya |
| `PanelPage.test.tsx` | uji dibalik menjadi **`tidak menggambar satu pun tombol unggah`** — menjaga ketiganya tidak kembali tanpa keputusan baru |
| `README.md` | baris "Tombol" pada tabel tata letak; paragraf ketiga tombol ditulis ulang sebagai "tidak dibawa" |

Jumlah uji layar **tetap 18** — satu uji diganti, bukan dibuang.

### 28.6 Cara membukanya kelak

Salah satu dari dua ini cukup:

1. **Tiga Flow Action di atas** beserta section form dan activity pemrosesnya, diminta ke
   Tim Pega — masuk ke permintaan export ulang berbasis Product rule (`D-39`).
2. **Contoh berkas CSV** untuk Master Panel dan Lokasi Panel dari Work Owner. Dari situ
   susunan kolomnya terbaca; yang tetap perlu diputuskan adalah apakah baris hasil unggah
   masuk **Waiting Approval** atau langsung **Approve**.

---

## 29. Koreksi Master Supplier — grid diselaraskan dengan Pega (2026-09-20)

Menyupersede bagian "Susunan kolom" pada §23.

### Keputusan yang dicabut, dan kenapa

Grid sembilan kolom sempat saya gabungkan menjadi tujuh kolom majemuk, dengan alasan yang
**disalin dari Master Bengkel**: grid empat puluh kolom tidak dapat dibaca pada layar mana
pun.

Alasan itu tidak berlaku di sini. Grid supplier hanya **sembilan** kolom, dan sembilan
kolom memang dapat dibaca. Menyalin alasan dari modul tetangga tanpa memeriksa apakah
premisnya masih berlaku adalah kekeliruannya — bukan penilaian tata letaknya.

Yang tersisa dari penggabungan itu hanyalah biayanya: petugas yang hafal urutan kolom Pega
harus mencarinya kembali, dan `D-13` menuntut justru sebaliknya.

### Kesembilan kolom, terverifikasi ke export

`pyColumnCount = 9` pada `Section/InboxMasterSupplier-Section.xml`, dengan urutan sel:

| Properti | Caption export | Caption produksi |
|---|---|---|
| `.ID` | ID | ID |
| `.NAMA` | NAMA | **Input Nama** |
| `.ALAMAT` | ALAMAT | ALAMAT |
| `.TELEPON` | TELP | TELP |
| `.JENIS_STATUS_NOTE` | JENIS SUPPLIER | JENIS SUPPLIER |
| `.STS_REKANAN_NOTE` | STATUS REKANAN | STATUS REKANAN |
| `.STS_AKTIF` | STATUS AKTIF | STATUS AKTIF |
| `.POSISI` | POSISI | POSISI |
| tombol | OPTION | OPTION |

Tangkapan layar **tidak** dipakai sebagai sumber tunggal: ia dapat terpotong di tepi
kanan, sehingga kolomnya dihitung ulang dari definisi grid lebih dulu. Yang diambil dari
tangkapan layar hanyalah hal yang tidak ada di export — lihat tiga butir di bawah.

### Urutan baris: TIDAK ada yang dapat ditiru

Grid ini tidak punya `pySortType`, `pySortOrder`, maupun `pyInitialSortColumn`, dan
`pyDisplayInitialSort = false`. Urutannya mengikuti kueri daftar, yang justru tidak ada di
export (`R-16`).

`ORDER BY` menurut nama pada `supplier_list` karena itu **dipertahankan**. Ini berbeda dari
Master Bengkel, yang urutannya (`pySortType = DESC` pada kolom pertama) memang terbaca dan
karena itu wajib diikuti. Di sini tidak ada yang wajib diikuti, dan urutan yang ditentukan
lebih baik daripada urutan yang berubah-ubah antar pemanggilan.

### Kolom POSISI digambar meski selalu kosong

Sempat saya hilangkan dengan alasan: menampilkan angka yang tidak diketahui artinya lebih
buruk daripada tidak menampilkannya.

Alasan itu gugur setelah melihat layar aslinya — **di Pega pun kolom itu kosong**. Yang
dihilangkan karena itu bukan angka yang membingungkan, melainkan satu kolom yang memang
ada. Ia dikembalikan, dan selnya menampilkan `—`.

### Kolom bersandi menampilkan LABEL, dengan sandi sebagai cadangan

`JENIS SUPPLIER` dan `STATUS REKANAN` di Pega terikat `.JENIS_STATUS_NOTE` dan
`.STS_REKANAN_NOTE` — **label**, bukan sandi. Keduanya hanya muncul di harness dan section
layar ini; tidak ada satu pun kueri di export yang memuatnya (`R-16`).

Tiga jalan, dan yang ketiga dipakai:

| Jalan | Verdict |
|---|---|
| Menampilkan sandi mentah | **DITOLAK** — layar berbeda dari Pega, dan sandi tidak berarti apa pun bagi petugas |
| Mengarang labelnya | **DITOLAK** — pada kolom yang menentukan status kerja sama |
| Mencari label dari daftar `/sandi`, jatuh ke sandi bila tidak diketahui | **dipakai** |

`labelOf` mengerjakannya. Hari ini `STATUS AKTIF` sudah tampil sebagai "Aktif" seperti
Pega; kedua kolom lain masih menampilkan sandinya — **terlihat, bukan tersamar**. Begitu
daftar Field Value-nya diterima, keduanya ikut menampilkan label tanpa satu baris kode pun
berubah.

### Tiga temuan dari layar produksi yang TIDAK ada di export

**Caption "Input Nama".** Export menuliskannya `NAMA`, dan teks "Input Nama" tidak ada di
section maupun harness Master Supplier. Layar produksi **sudah berubah sejak export
diambil** — bukti langsung untuk `R-09`. Yang diikuti adalah layar yang dilihat pengguna
(`D-13`), dan selisihnya dicatat alih-alih didiamkan.

**Label `JENIS SUPPLIER` berbunyi "ASM".** Ini melemahkan label yang saya berikan pada
`DefaultCodeOption`, yang menamai `JENIS_STATUS = "1"` sebagai *"Heavy Equipment"* —
disimpulkan dari `SUPPLIER_HE := "1"` yang memang terbukti.

Bila labelnya di produksi berbunyi "ASM", domain `JENIS_STATUS` kemungkinan **lebih kaya
daripada dua nilai**, dan `GetDataSupplier_pre` yang memaksanya menjadi `0`/`1` justru
membuang informasi — yang berarti form kami ikut membuangnya.

**Belum diubah**, karena hubungan `JENIS_STATUS = "1"` → `SUPPLIER_HE = "1"` tetap
terbukti dan tidak dibantah apa pun. Diangkat sebagai pertanyaan terbuka; mengubah label
tanpa daftar Field Value hanya akan mengganti satu tebakan dengan tebakan lain.

**ID produksi tujuh digit.** `1005972` **tidak mungkin** dihasilkan
`id_site || lpad(to_char(supplier_seq.nextval), 11, '0')`, yang selalu menghasilkan
sebelas digit atau lebih.

Dan ia benar-benar kolom `ID`, bukan `OLDID`: tombol Edit mengirimkan nilai kolom itu ke
`GetDataSupplier_pre`, yang menyaring `A.ID = {ParamSP.ID}`. Baris yang ada sekarang
karena itu **tidak lahir dari procedure itu**.

Akibatnya diterima secara sadar: ID yang diterbitkan aplikasi baru akan terlihat berbeda
dari setiap baris yang sudah ada — persis sifat yang `D-22` justru pilih pada nomor klaim,
karena asal sebuah baris terbaca dari kuncinya tanpa tabel pemetaan. Yang berubah hanyalah
bahwa ia sekarang **dinyatakan**, bukan ditemukan.

### Pelajaran yang dicatat, bukan hanya perbaikannya

Kekeliruan ini **tidak ditangkap satu pun uji** — yang menangkapnya adalah Work Owner yang
melihat layarnya. Sebabnya: seluruh uji layar memeriksa *isi* baris, tidak satu pun
memeriksa *susunan kolomnya*.

Tiga uji ditambahkan untuk menutupnya, dan yang pertama membandingkan seluruh
`columnheader` dengan daftar harfiah — penggabungan kolom apa pun gagal di sana lebih
dulu.

### Utang yang ditambahkan

| Utang | Rencana penyelesaian |
|---|---|
| Kolom `JENIS SUPPLIER` dan `STATUS REKANAN` masih menampilkan sandi | menunggu daftar Field Value; `labelOf` sudah siap menerimanya tanpa perubahan kode |
| Label `JENIS_STATUS = "1"` mungkin bukan "Heavy Equipment" | menunggu jawaban Work Owner atas temuan "ASM" |
| Kolom `POSISI` selalu kosong | menunggu keputusan tentang antrean `proteksi_klaimmbu` |
| Nama supplier tampak sebagai tautan di produksi | menunggu konfirmasi ia membuka apa; export tidak memasang aksi apa pun |

---

## 30. Paginasi Master Panel — ukuran halaman berbeda ANTARTAB (2026-09-20)

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

## 31. Koreksi Master Auto Claim — caption dan urutan tab (2026-09-20)

### 31.1 Yang dikoreksi Work Owner

Tab pertama layar ini bercaption **"Master Auto Klaim"**. Di Pega, tab dengan nama itu tidak
ada — **"Master Auto Klaim" adalah judul layar**, sebaris dengan tombol Tambah dan Refresh di
atas keempat tab.

Caption dan urutan tab yang sebenarnya ada di `pyTitle` tiap container pada
`Section/MasterAutoKlaim-Section.xml` (`pyHeaderType=TABBED`, `pyTabAlignment=Top`,
`pyStretchTab=false`), dalam urutan dokumen:

| # | `pyTitle` | section | penyaring |
|---|---|---|---|
| 1 | **Approve** | `BrowseAutoKlaim` | `stsapprove="1"` |
| 2 | **Reject** | `BrowseAutoKlaimReject` | `stsapprove="2"` |
| 3 | **Waiting Approval** | `BrowseAutoKlaimApproval` | `stsapprove="0"` |
| 4 | **Komite Approval** | `BrowseAutoKlaimKomite` | `stsapprove="0"`, `komite="ya"` |

Akibat kekeliruan itu: tab **Approve hilang sama sekali**, dan tiga tab lain bergeser. Hanya
satu dari empat yang kebetulan benar tempatnya.

### 31.2 Kenapa penyaringnya benar tetapi captionnya salah

Keduanya datang dari tempat yang berbeda di berkas yang sama:

| Yang dibaca | Memberi |
|---|---|
| `pyDeferLoadRetrievalActivityParams` | **penyaring** tiap tab — dibaca sejak awal, keempatnya benar |
| `pyTitle` container | **caption** tiap tab — tidak pernah dibaca |

Caption disusun dari nama section dan dari judul layar, bukan dari sumbernya. Pelajarannya
dicatat: satu berkas dapat menjawab dua pertanyaan yang berbeda, dan menjawab yang satu tidak
berarti yang lain ikut terjawab.

### 31.3 Judul layar memakai ejaan Pega, menu memakai ejaan basis data

| Tempat | Teks | Sumber |
|---|---|---|
| `<h1>` layar | Master Auto **Klaim** | `Section/MasterAutoKlaim-Section.xml` |
| Butir menu di kolom samping | Master Auto **Claim** | `POOLDATA.M_MENU_APLIKASI_PNC.MENU_DESC` |

Kedua ejaan itu memang berbeda di sistem lama. Keduanya direplikasi dari sumbernya
masing-masing; menyeragamkannya berarti memilih salah satu tanpa dasar, dan `D-80` menetapkan
teks yang dilihat pengguna mengikuti layar Pega apa adanya.

### 31.4 Dua caption yang bertabrakan dengan nama tombol

"Approve" kini caption tab **dan** caption tombol pada form tab Komite; begitu pula "Reject".
Keduanya memang bernama sama di Pega, dan bagi pengguna tidak membingungkan karena letaknya
berbeda.

Bagi uji ia jebakan — dan jebakan yang **sudah pernah menjatuhkan tiga uji Master Rekening**.
Penutupnya: seluruh penekanan tab lewat `clickTab`, yang membatasi pencarian pada `<nav>` bilah
tab. Tidak ada pencarian tab yang global lagi.

### 31.5 Bentuk visual tab TIDAK diubah — dibawa ke Work Owner

Tabnya tetap bergaya garis bawah, bukan tab berbingkai gaya Pega klasik.

Dua instruksi bertabrakan di sini:

| Instruksi | Menuntut |
|---|---|
| "UI mengikuti Pega" | tab berbingkai |
| "Konsisten dengan modul sebelumnya" | garis bawah, seperti seluruh layar master lain |

Seluruh layar master yang sudah ada — Master Rekening, Penolakan Klaim, Supplier, Panel,
Bengkel — memakai gaya yang sama. Mengubah satu modul membuat aplikasinya tidak seragam;
mengubah semuanya menyentuh modul di bawah Isolasi Protektif.

**Belum diputuskan**, dan sengaja tidak diputuskan sendiri.

### 31.6 Uji yang menjaga koreksi ini

`menampilkan keempat tab Pega pada urutan yang sama` membandingkan caption **sebagai senarai**
— caption yang benar tetapi berpindah tempat pun gagal — lalu menegaskan "Master Auto Klaim"
ada sebagai `heading` tetapi **tidak** sebagai tab.

Seluruhnya **19 uji**, naik dari 18; backend tidak tersentuh.

---

## 32. Modul Master Sparepart (2026-09-20, sesi keenam belas)

Master ketiga dari keluarga alat berat, setelah Master Bengkel (§21) dan Master Panel (§22).
Ketiganya berbagi satu activity persetujuan yang sama di Pega — `SetApprovalAllMaster` —
dibedakan hanya oleh nilai yang ditulis ke `InputBengkel.ALASAN_STS_BGKL`, yang untuk modul
ini bernilai `"M_SPAREPART_HE"`.

### 32.1 Empat pertanyaan yang diajukan, dan jawabannya

| # | Pertanyaan | Jawaban Work Owner |
|---|---|---|
| 1 | Grid Pega hanya 5 kolom dari 23 — susunan mana yang dipakai? | **Ikuti Pega, buang sel kosong** |
| 2 | Dropdown Kategori/Tipe: rule Pega tidak sepakat soal penyaring APPROVAL | **seperti aplikasi PEGA** |
| 3 | Daftar pilihan SATUAN tidak ada di export | **seperti aplikasi PEGA** |
| 4 | Dua tombol unggah: hapus (preseden §28) atau gambar mati? | **seperti aplikasi PEGA** |

Ketiga jawaban "seperti aplikasi PEGA" menuntut penelusuran tambahan sebelum dapat
dikerjakan — jawaban itu menunjuk perilaku Pega, dan perilaku itu belum tentu yang tertulis
di rule yang pertama kali ditemukan. Hasilnya:

**Pertanyaan 2 tertutup oleh bukti.** Export memuat TIGA rule yang membaca kedua tabel acuan
dengan penyaring berbeda. Yang benar-benar dipakai layar Master Sparepart adalah
`Activity/BrowseTipeKategoriPart-Act.xml` — dirujuk ketiga section tabnya — yang menjalankan:

    TempStatus.City := "1"
    BrowseMasterSparepartCategoryClaimHE  ->  where APPROVAL = {TempStatus.City}
    BrowseTipeSparepart                   ->  where APPROVAL = '1'

**Keduanya menyaring "sudah disetujui".** Kedua rule yang memakai penyaring "menunggu"
dipanggil dari layar Master Kategori dan Master Tipe (MENU_ID 33 dan 34), bukan dari sini.

**Pertanyaan 3 menuntut kompromi yang dinyatakan.** Pega merender SATUAN sebagai `pxDropdown`,
JENIS_SPART sebagai `pxRadioButtons`, dan STATUS_SPART sebagai `pxDropdown` — tetapi daftar
pilihan ketiganya ada di rule Field Value yang **tidak ikut di export** (`R-16`). "Seperti
Pega" karena itu dapat dipenuhi pada BENTUKNYA, tidak pada ISINYA. Yang dikerjakan:

- bentuknya dropdown dan radio, persis seperti Pega;
- pilihannya diambil dari **nilai yang sudah dipakai baris lain** pada entitas itu;
- selalu ada pilihan **"Lainnya…"** yang membuka isian ketik.

Butir terakhir bukan hiasan: tanpa ia, basis data yang masih kosong menghasilkan dropdown
tanpa satu pun pilihan, dan baris lama yang nilainya belum pernah dipakai baris lain tidak
dapat disimpan ulang. Komponennya `ChoiceField.tsx`, sengaja tinggal di dalam modul dan
bukan di `shared/components/` — ia jawaban atas daftar pilihan yang hilang, bukan pola
antarmuka yang layak dipakai ulang.

**Pertanyaan 4 menyimpang dari §28 dengan sengaja.** Pada Master Panel, Work Owner memilih
menghapus tiga tombol unggahnya. Di sini ia memilih "seperti aplikasi PEGA", sehingga kedua
tombolnya **digambar dalam keadaan mati** beserta keterangan kenapa. Perbedaan perlakuan
antara dua layar yang berdekatan itu disengaja, bukan kelalaian.

### 32.2 Empat hal yang membedakannya dari Master Panel

| | Master Panel | Master Sparepart |
|---|---|---|
| Pencatat pelaku | tidak ada kolomnya | **`USER_UPDATE` ada** — diisi login pemanggil |
| Alasan tolak | `ALASAN_TOLAK` ada | **tidak ada kolomnya sama sekali** |
| Tabel acuan | tidak ada | **dua**, keduanya dibaca dari basis data entitas |
| Lebar nomor urut ID | enam digit | **sepuluh digit** |

Akibat yang paling terlihat: layar ini **tidak menggambar isian Catatan** pada bilah
keputusan. Layar persetujuan Pega memang punya `pyNote`, tetapi di modul ini ia tidak menuju
ke mana pun — `SetValueSparepartHE` justru mengosongkannya dan `UpdateSparepartHE_act`
memakainya sebagai penampung sementara hasil penyimpanan. Menggambar isian yang diam-diam
membuang isinya lebih buruk daripada tidak menggambarnya.

Akibat kedua: `/pilihan` di modul ini **dipasangi pemeriksaan portal**, berbeda dari Master
Panel yang tidak. Di sana daftarnya konstanta yang ditanam di activity Pega; di sini ia
dibaca dari `GCNM_M_SPAREPART_CATEGORY` dan `GCNM_M_SPAREPART_TYPE` milik entitas — dan
menyajikan daftar satu entitas kepada entitas lain adalah kebocoran yang `R-20` cegah.

### 32.3 Dua cacat warisan yang TIDAK dibawa

**Sel `.TELP_BENGKEL` di dalam grid Sparepart.** Satu sel salin-tempel dari grid Master
Bengkel, menunjuk properti yang tidak ada di `SPAREPART_HE`, jadi selalu kosong. Menggambar
kolom yang selalu kosong bukan kesetaraan melainkan peniruan cacat.

**`ErrMsg` yang memikul dua arti.** `Database/PEGA_M_SPAREPART_HE.prc:23` menuliskan pesan
keberhasilan beserta ID yang diterbitkan ke parameter bernama `ErrMsg` — pada jalur SUKSES.
Satu kolom keluaran memikul pesan berhasil dan pesan galat sekaligus, persis pola yang `D-68`
tolak pada `ADD_NEWMASTERVIRTUALACCOUNT`. Kontrak galat berbasis string itu tidak dibawa.

Ditambah alias menyesatkan yang tidak dibawa: ketiga rule validasi sparepart membandingkan
nilainya terhadap halaman `TempInputPanelHE` — halaman milik Master **Panel** — dengan
properti `.CaseID` untuk nama, `.City` untuk nomor, dan `.Country` untuk kode. Kedua tabel
acuan pun mengaliaskan `PART_CATEGORY_ID` menjadi `"CityID"` dan `PART_CATEGORY_NAME`
menjadi `"City"`. Seluruhnya bentuk utang yang `03-CURRENT-ARCHITECTURE.md` §4.2 catat.

### 32.4 Satu perilaku ganjil yang SENGAJA direplikasi

`TGL_UPDATE_HARGA` distempel pada **setiap** penyimpanan yang harganya terisi, bukan hanya
saat harganya berubah. Itu meniru `Activity/UpdateSparepartHE_act` apa adanya: preconditionnya
memeriksa `PropertyHasValue(TempSparepart.HARGA_JUAL)` — yang diperiksa hanyalah harganya
TERISI, bukan harganya BERUBAH.

Akibatnya kolom bernama "tanggal update harga" ikut bergerak ketika yang berubah hanya nama
sparepartnya. Itu ganjil, dan ia **tidak diperbaiki**: memperbaikinya berarti menambah butir
ke daftar perbaikan eksplisit `P-5`, dan daftar itu milik `D-49` — keputusan Work Owner, bukan
tafsiran modul.

### 32.5 Tiga selisih yang DIRENCANAKAN terhadap sistem lama

Ketiganya menolak isian yang di sistem lama diterima. Baris lama yang sudah memuat nilai
seperti itu tetap **dibaca apa adanya**; penolakan hanya terjadi saat barisnya disimpan ulang.

1. **Kedelapan isian angka wajib berupa angka.** Sistem lama menerima apa pun, dan
   `HARGA_JUAL` bertuliskan "seribu" tersimpan apa adanya lalu muncul di laporan.
2. **Harga jual tidak boleh negatif, dan dibatasi Rp 100 miliar.** Penjaring salah ketik, bukan
   aturan bisnis: satu nol berlebih mengubah Rp 1,25 juta menjadi Rp 12,5 juta.
3. **Stok minimal tidak boleh melampaui stok maksimal.** Pasangan yang terbalik membuat setiap
   pemeriksaan stok di modul hilir selalu benar atau selalu salah.

Ditambah satu yang lebih halus: ketiga rule validasi Pega hanya meng-`UPPER` tanpa memangkas,
sehingga baris yang tersimpan dengan spasi di ujung lolos sebagai nilai yang berbeda. Kueri
di sini memakai `UPPER(TRIM(...))`.

### 32.6 Asumsi yang disadari, beserta cara memeriksanya

| Asumsi | Dasarnya | Cara memeriksanya |
|---|---|---|
| `PROD_DATE` bertipe TEKS | Pega merendernya `pxTextInput` TANPA `pyDateTimeFormat` | `claimpnc -periksa` membaca kolomnya |
| `TGL_UPDATE_HARGA` bertipe WAKTU | Pega mengisinya dengan waktu sistem | idem |
| `SPAREPART_HE` dan `M_SPAREPART_HE_BU` satu sumber | keduanya dibaca/ditulis berpasangan | perbandingan jumlah baris di `-periksa` |
| Batas panjang enam isian | tidak ada satu pun `pyMaxLength` di layar lama (`R-08`) | belum dapat diperiksa; menunggu DDL |

Akhiran `_BU` pada nama tabel JSON Pega **tidak disebut artinya di mana pun** dalam export.
Bila `-periksa` melaporkan jumlah baris keduanya berbeda, jalur tulis tidak boleh diaktifkan
di produksi sebelum DBA memastikan mana yang menjadi sumbernya.

### 32.7 Catatan lingkup

`D-34` menyatakan area Bengkel, Sparepart, dan Supplier dikeluarkan dari lingkup migrasi.
Keputusan itu sudah dilangkahi dalam praktik: Master Bengkel (§21), Master Panel (§22), dan
Master Supplier (§23) dari daftar pengecualian yang sama sudah dibangun dan diterima. Modul
ini mengikuti preseden itu.

Pencabutan `D-34` karena itu **tersirat, bukan tertulis**. Bila kelak ada yang membaca
Decision Log tanpa membaca dokumen ini, ia akan menemukan larangan yang sudah tidak berlaku.
Keputusan baru yang menyebut `D-34` sebagai disupersede layak ditulis — dan itu milik Work
Owner, bukan tafsiran modul.

---

## 33. Modul Pelaporan Klaim (2026-09-18)

> Sesi ini berjalan di cabang `feat/Michelle-flowpelaporan-backup`, **bersamaan** dengan
> §14–§16 yang berjalan di `master`. Ia ditulis sebagai §12 di cabangnya sendiri; nomor §33
> diberikan saat digabungkan, supaya rujukan §14–§16 yang sudah ada tidak bergeser.

Modul **proses klaim** yang pertama. Karena itu setiap keputusan di bawah bukan hanya tentang satu
layar: ia menjadi pola untuk tiga belas modul bisnis berikutnya, yang seluruhnya berurusan dengan
klaim dan bukan dengan master.

### 33.1 Keputusan Work Owner pada sesi ini

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Tiket `B-14` bertentangan dengan export — mana yang diikuti? | **Ikuti export.** Tiket dicatat sebagai usulan revisi, tidak disunting diam-diam |
| 2 | Header laporan ada di tabel engine Pega yang dibaca 116 rule | **Tabel baru milik aplikasi.** *"tabel ini sudah tidak mau dipakai dan akan dibuatkan tabel baru"* |
| 3 | Lingkup sesi ini | **Form dan daftar bertahap.** Tanpa lampiran, utas komunikasi, dan penugasan |
| 4 | Bahasa penamaan | *"Tugas sekarang hanya untuk proses modul ini saja"* — lihat §33.9 |

### 33.2 Nama modul: kenapa "Pelaporan Klaim" dan bukan "Receive Document"

`D-81` menetapkan nama modul diambil dari nama yang disebut Work Owner. Yang membuat keputusan ini
mudah adalah bahwa sistem lama pun memakai nama itu di permukaan yang dilihat orang — hanya nama
kelas internalnya yang berbeda:

| Bukti | Isi |
|---|---|
| `Navigation/pyCaseWorkerNavigation-Navigation.xml:19864` | menu **"Inbox Laporan Klaim"** |
| `Activity/CreateNewCaseRCV-Act.xml` step 7 | `Param.Posisi = "LAPORAN KLAIM"` |
| `Section/ViewStatusReceiveDocument-Section.xml` | komentar developer menyebut *"inbox pelaporan klaim"* |

`Work-ReceiveDocument` adalah nama internal, dan `D-19` menetapkan alias internal tidak dibawa.

### 33.3 Tahap DIHITUNG, tidak disimpan

Sistem lama tidak menyimpan status laporan sebagai kolom. Ia menurunkannya dari kombinasi dua
penanda, terbaca dari `RDB List/BrowseClaimRCV_Aksep-SQL.xml`:

```
PNCCASEID null   + STATUSLOCK null     -> belum ditransfer ke ASM
PNCCASEID null   + STATUSLOCK terisi   -> sudah ditransfer, belum diregistrasi
PNCCASEID terisi + STATUSLOCK terisi   -> sudah diregistrasi menjadi klaim
```

Pola itu **dipertahankan**. Menyimpan tahap sebagai kolom tersendiri akan membuat dua sumber
kebenaran yang dapat berselisih — dan selisihnya tidak menimbulkan galat, hanya laporan yang
tertahan di tab yang salah.

Konsekuensinya mengikat di dua tempat sekaligus: ekspresi `CASE` di SQL dan metode `Tahap()` di Go
harus sama persis. Keduanya dipagari uji `TestEkspresiTahapSeragamDiSeluruhKueri`.

**Dua tahap yang belum dapat terjadi.** `SUDAH_AKSEPTASI` dan `DITOLAK` ditentukan modul klaim
(`B-5`, `B-10`) yang belum ada; sistem lama menghitungnya dengan menengok `t_claim_adjustment` dan
`t_claim_pnc`. Modul ini tidak menengok ke sana — ia menyediakan satu field `HasilKlaim` yang
diisi modul klaim saat hasilnya diketahui. Hari ini field itu selalu kosong, dan kedua tab itu
selalu nol. Tabnya **tetap ditampilkan**: tab yang menghilang saat kosong membuat pengguna mengira
tabnya tidak ada.

### 33.4 Kenapa tabel baru, dan apa yang ditinggalkan bersama tabel lama

Data laporan di sistem lama hidup di dua tempat, dan keduanya tidak dapat dipakai:

| Tempat | Kenapa tidak |
|---|---|
| `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` | Tabel milik **engine Pega**, dibaca 116 rule, dan tetap ditulis Pega untuk seluruh case type lain yang belum bermigrasi. `P-1` melarang dua sistem menulis satu tabel |
| `POOLDATA.T_CLAIM_RECIVEDCLAIM` | Penulis tunggalnya memang terbukti — hanya `PROCINSERTDATARECIVEDKLAIM` yang menyentuhnya. Tetapi ia **hanya memuat sebagian**: penanda transfer, estimasi, tipe klaim, dan jumlah dokumen tidak punya kolom di sana; keempatnya hanya hidup di blob case Pega |

**Satu hal yang harus dijawab sebelum modul ini menyala di produksi.** Penelusuran seluruh export
tidak menemukan satu pun rule yang **membaca** `T_CLAIM_RECIVEDCLAIM`. Sejauh yang terlihat, tabel
itu write-only. Bila ada pembaca di luar export — laporan BI, perkakas cabang, kueri berkala — ia
akan berhenti menerima baris baru begitu modul ini menyala, dan berhentinya **tidak menimbulkan
galat apa pun**. Kuerinya untuk DBA ada di kepala migrasi `0003`.

### 33.5 Alias Pega tidak dibawa masuk

Pemetaan properti klipboard ke kolom di sistem lama menyesatkan secara aktif. Enam yang terburuk,
dari `RDB List/Rcv_ProcInsertRecivedDocument-SQL.xml:102-124`:

| Properti | Kolom sebenarnya | Kenapa berbahaya |
|---|---|---|
| `TelpTertanggung` | `USERINPUT` | petugas penginput, bukan telepon tertanggung |
| `TanggalSelesaiRawatInap` | `REGISTDATE` | tanggal registrasi, bukan tanggal pulang rawat |
| `NamaSurveyor` | `NAMAKURIRASM` | nama kurir, bukan surveyor |
| `UserTeknis` | `NAMATERTANGGUNG` | nama tertanggung, bukan petugas teknis |
| `LokasiSurveyor` | `LOKASIKEJADIAN` | lokasi kejadian |
| `ReferenceId` | `TANGGALTERIMADOKUMEN` | sebuah **tanggal**, bukan nomor referensi |

`StatusLock` juga bukan seperti namanya: ia bukan kunci baris melainkan penanda bahwa laporan sudah
dikirim ke ASM pusat.

Pemetaan tiga arah lengkapnya ditulis di kepala `repo/sqlstore/laporan.sql` — satu-satunya tempat
ketiganya dapat dibandingkan berdampingan. Ini melaksanakan `03-CURRENT-ARCHITECTURE` §4.2.

### 33.6 Perilaku yang sengaja TIDAK ditiru, beserta alasannya

| Perilaku lama | Kenapa tidak dibawa |
|---|---|
| Kunci layar `StatusLock` **dapat dilewati** operator berjabatan "PA" atau bercabang kantor pusat (`Pre_ActReceiveDocument` step 3-4) | Pagar kewenangan yang ditulis di dalam kode — persis yang `D-15` larang. Di sini yang mengunci adalah REGISTRASI, tanpa pengecualian |
| `UpdateRCVCase` **menimpa** nomor polis, tertanggung, tanggal kejadian, dan kronologi pada laporan dengan nilai dari klaim | Itu menghapus apa yang benar-benar dilaporkan pelapor — dan dengan begitu menghapus satu-satunya cara mengetahui bahwa pelapor semula menyebut polis yang keliru |
| `TANGGALTERIMADOKUMEN` bertipe **VARCHAR2** | Tanggal sebagai teks membuat pengurutan menjadi pengurutan teks dan penyaringan rentang tidak dapat memakai indeks — cacat yang `09-DATABASE-STRATEGY` §3.2 perintahkan dihapus |
| `ROWNUM` pada ketiga kueri inbox | `OFFSET … FETCH NEXT`, sesuai §3.3 |
| Tiga penyisipan `{ASIS:tempQuery.*}` per kueri inbox | Penyaring sebagai parameter. Ketiga rule inbox menjadi satu kueri |
| Prefix `'ASM-FW-GCNMFW-WORK '` pada nomor klaim | `D-22` dan `D-71`: kunci teknis Pega tidak lagi bocor ke data bisnis |
| `ErrMsg` berbasis string dan `COMMIT` di dalam rule | `D-68`: kepemilikan transaksi pindah ke Go |

### 33.7 Perpindahan tahap menjadi aksi tersendiri

Di sistem lama, pengisian `StatusLock` dan `DateOfSendASM` terjadi sebagai **efek samping
penyimpanan layar** — tanpa langkah tersendiri, dan tanpa apa pun yang mencegah laporan ditransfer
dua kali.

Di sini keduanya menjadi sub-sumber daya: `POST /{nomor}/transfer` dan `POST /{nomor}/klaim`.
`10-API-STRATEGY` §2 menetapkannya, dan alasannya nyata di sini — kedua aksi punya invarian
sendiri, dan pembaruan field generik akan melewatkan keduanya sekaligus.

**Transfer kedua ditolak, bukan dibiarkan lolos.** Menimpa tanggal transfer berarti menghapus kapan
laporan itu benar-benar dikirim. Begitu pula penautan klaim kedua: satu laporan melahirkan satu
klaim, dan menautkannya ke klaim kedua membuat dua klaim mengaku berasal dari laporan yang sama.

### 33.8 Penyimpangan yang disadari

#### 17.8.1 Nilai uang disimpan sebagai teks, bukan `NUMBER(18,2)`

`09-DATABASE-STRATEGY` §5 menetapkan nilai uang bertipe `NUMBER(18,2)`. Kolom `NILAI_ESTIMASI`
dibuat `VARCHAR2(30)`.

**Alasannya bukan kemudahan.** Aplikasi membawa nilai ini sebagai teks desimal, karena pustaka
standar Go tidak punya tipe desimal dan `float64` akan membulatkan diam-diam — hal yang `I-12`
larang. Menuliskan teks itu ke kolom `NUMBER` menyerahkan konversinya kepada
`NLS_NUMERIC_CHARACTERS`: pada sesi yang pemisah desimalnya koma, `1234.56` akan **ditolak**.

Kegagalan itu bergantung lingkungan, dan **di mesin tempat berkas ini ditulis tidak ada basis data
untuk membuktikannya**. Memilih tipe yang kegagalannya tidak dapat saya deteksi adalah pilihan yang
salah. Bentuknya dipagari `CHECK` regex di basis data dan diperiksa lagi di domain.

**Batas yang harus disadari:** kolomnya tidak dapat dijumlahkan atau dibandingkan sebagai angka di
SQL. Itu dapat diterima karena nilai ini tidak dipakai perhitungan apa pun — ambang komite dan
Notice of Large Losses dihitung dari nilai pada **klaim** (`B-5`), bukan dari perkiraan pelapor.

**Pertanyaan terbuka:** apakah pustaka desimal boleh ditambahkan sebagai dependensi? Bila ya,
kolom ini menjadi `NUMBER(18,2)` lewat migrasi tersendiri, dan yang berubah hanya adapter.

#### 17.8.2 Bentuk nomor laporan adalah keputusan baru

Nomor laporan lama ditentukan `pyWorkIDPrefix` pada rule kelas Pega, yang **tidak ada di export**;
pencarian seluruh export tidak menemukan satu pun contoh nilainya. Bentuk lamanya karena itu tidak
diketahui dan tidak dapat ditiru.

Yang dipakai mengikuti satu-satunya keputusan penomoran yang pernah diambil proyek ini, `D-71`:
`LPK.YY.xxxx` bersebelahan dengan `PNCN.YY.xxxx`.

Satu cacat `D-71` **tidak diwarisi**: sintaks nomor klaim memakai `TO_CHAR` tanpa format mask,
sehingga lebar segmen terakhir berubah-ubah dan pengurutan sebagai teks tidak sesuai urutan
penerbitan (`.10` mendahului `.9`). Tabel ini baru dan nomornya belum pernah terbit, jadi lebarnya
dibuat tetap. Dikunci uji `TestPengurutanTeksSesuaiUrutanPenerbitan`.

**Menunggu konfirmasi Work Owner.** Bila bentuk lain yang dikehendaki, yang berubah hanya satu
konstanta.

#### 17.8.3 Satu prop baru pada komponen tabel baku

Lihat `catatan-pengembangan.md` §11.9. Ringkasnya: `TabelData` menyaring di peramban dan dokumennya
sendiri melarang layar seperti ini memakainya. Membuat tabel kedua akan membatalkan aturan "semua
tabel lewat `TabelData`" pada modul bisnis pertama yang memakainya; memakainya apa adanya
menghasilkan pencarian yang **bohong**. Yang dipilih: satu prop opsional, bersifat menambah.

### 33.9 Bahasa penamaan: asumsi yang salah, lalu dikoreksi Work Owner

`CLAUDE.md` memuat `D-80` (18 September) yang mewajibkan seluruh nama di dalam kode berbahasa
Inggris. Seluruh kode di working copy ini — `auth`, `portal`, `masterrekening`, `masterstatus` —
berbahasa Indonesia. Keduanya tidak dapat dipenuhi bersamaan.

Jawaban Work Owner, *"Tugas sekarang hanya untuk proses modul ini saja"*, menolak opsi mengganti
seluruh modul tetapi tidak menyebut bahasa mana untuk modul baru.

**Asumsi yang saya ambil saat itu: bahasa Indonesia**, mengikuti kelima modul yang ada, dengan
alasan instruksi sesi ini menuntut *konsistensi implementasi* secara eksplisit.

**Asumsi itu SALAH.** Work Owner memeriksa hasilnya dan menyatakan: *"saya cek masih menggunakan
bahasa indonesia, mohon diubah jadi inggris"*. Yang berlaku adalah `D-80`, bukan konsistensi dengan
kode yang ditulis sebelum `D-80` ada.

**Yang kemudian dikerjakan:** seluruh penamaan di dalam modul ini diganti ke bahasa Inggris —
folder, berkas, paket, tipe, fungsi, method, field, parameter, dan variabel lokal, di backend
maupun frontend.

**Lima hal TETAP berbahasa Indonesia**, sesuai kelima pengecualian `D-80`:

| Yang tetap Indonesia | Contoh di modul ini |
|---|---|
| Komentar dan dokumen | seluruh komentar Go dan TSX, serta ketiga dokumen di `docs/` |
| Nama field JSON pada API | `nama_pelapor`, `tanggal_kejadian`, `dapat_ditransfer` |
| Nama tabel dan kolom basis data | `POOLDATA.CPNC_LAPORAN_KLAIM`, `NOMOR`, `DITRANSFER` |
| Teks yang dilihat pengguna | judul tab, label kolom, pesan galat |
| Nilai kode galat | `laporan_sudah_ditransfer` — ia kontrak yang sudah dibaca frontend |

**Nama folder mengikuti `D-81` sepenuhnya:** `internal/pelaporanklaim` (tanpa tanda hubung, karena
Go tidak mengizinkannya) dan `src/modules/pelaporan-klaim`. Nama modulnya Indonesia; isinya Inggris.

#### 17.9.1 Akibat yang diterima: berkas campur dua bahasa

Komponen bersama yang ditulis sebelum `D-80` berada **di luar lingkup** pekerjaan ini, dan arahan
Work Owner *"untuk luar lingkup flow tolong jangan diubah atau diperbaiki apapun"* melarang
menyentuhnya. Akibatnya satu baris kode dapat memuat dua bahasa, dan itu memang yang dikehendaki:

```tsx
<KolomIsian id="nama_pelapor" galat={errors.nama_pelapor?.message} disabled={save.isPending} />
<TextAreaField id="kronologi" error={errors.kronologi?.message} disabled={save.isPending} />
```

Yang tetap Indonesia karena berada di luar lingkup: props `TabelData` (`kolom`, `baris`,
`kunciBaris`, `judul`, `aksi`, `cariDiServer`), props `KolomIsian` (`galat`, `petunjuk`), props
`PesanGalat` (`judul`, `keterangan`, `nada`), prop `Tombol` (`nada`), klien API (`panggilAPI`,
`metode`, `badan`, `GalatAPI`), store sesi (`gunakanSesi`), dan seam jam (`waktu.Jam`,
`JamSistem`, `JamTetapPada`, `.Maju`).

Tiga fungsi yang DITULIS pada sesi ini di paket `waktu` ikut diganti ke Inggris — `DateWIB`,
`TwoDigitYearWIB` — karena keduanya berkas baru, bukan kode lama yang disentuh. Paket `waktu`
karenanya kini memuat keduanya berdampingan.

### 33.10 Kontrak API modul ini

| Metode | Jalur | Jawaban |
|---|---|---|
| `GET` | `/api/pelaporan-klaim` | halaman, `jumlah`, `batas`, `lewati`, dan `ringkasan` kelima tahap |
| `POST` | `/api/pelaporan-klaim` | `201` beserta nomor yang dibuat sistem |
| `GET` | `/api/pelaporan-klaim/{nomor}` | satu laporan |
| `PUT` | `/api/pelaporan-klaim/{nomor}` | `200` beserta laporan setelah diubah |
| `POST` | `/api/pelaporan-klaim/{nomor}/transfer` | `200`, atau `409` bila sudah ditransfer |
| `POST` | `/api/pelaporan-klaim/{nomor}/klaim` | `200`, atau `409` bila sudah diregistrasi |

**Tidak ada `DELETE`.** `ADR-0012` menetapkan penghapusan lunak menyeluruh, dan laporan yang sudah
tertaut klaim dirujuk klaimnya lewat `ClaimData.RCV_ID` — rujukan yang dipakai 41 rule Pega.
Ketiadaannya dikunci dua uji, supaya penambahannya kelak menjadi keputusan sadar.

**Ringkasan tahap dikirim bersama halaman, bukan lewat endpoint terpisah.** Alasannya sama dengan
alasan sistem lama memakai satu kueri berisi enam `SUM(CASE WHEN ...)`: angka tiap tab harus
konsisten dengan isi tab yang sedang terbuka. Dua permintaan terpisah dapat tiba di antara dua
perubahan, dan pengguna melihat lencana "3" di atas tabel berisi empat baris.

Kode galat baru:

| Kode | HTTP | Kenapa bukan yang lain |
|---|---|---|
| `validasi_gagal` | `422` | Permintaannya berbentuk benar, isinya yang melanggar aturan bisnis |
| `laporan_sudah_ditransfer` | `409` | Isian penggunanya sah, tetapi bentrok dengan keadaan penyimpanan |
| `laporan_sudah_diregistrasi` | `409` | idem |
| `nomor_laporan_sudah_dipakai` | `409` | Seharusnya mustahil — nomornya dibuat urutan |
| `laporan_klaim_tidak_ditemukan` | `404` | — |

### 33.11 Validasi: hanya satu field yang wajib

Layar Pega tidak mewajibkan apa pun. Itu bukan kelalaian melainkan sifat pekerjaannya: laporan
kerugian datang lewat telepon dan surel dengan kelengkapan yang berbeda-beda, dan petugas harus
dapat mencatatnya **sekarang** lalu melengkapinya kemudian. Menolak laporan yang belum lengkap
berarti laporan itu tidak tercatat sama sekali.

Yang diwajibkan hanya **nama pelapor** — tanpa itu laporannya tidak dapat ditindaklanjuti siapa
pun. Kelengkapan yang sesungguhnya ditegakkan saat **registrasi** (`B-2`), tempat invarian `I-2`
sampai `I-10` berlaku.

Yang ditambahkan di luar itu semuanya berasal dari lebar kolom atau dari bentuk yang harus dapat
dikirim ke basis data: batas panjang, bentuk tanggal `YYYY-MM-DD`, bentuk surel yang longgar, dan
bentuk nilai uang. Seluruh pelanggaran dikembalikan **sekaligus** — pada form berisi 17 isian,
sekali-satu akan menyiksa.

### 33.12 Otorisasi: keadaan yang belum berubah

Rute modul ini terlindungi sesi, tetapi **belum diperiksa perannya**. Sistem lama membatasi layar
ini pada tujuh peran lewat When rule `IsReceivePNC`: `Administrators`, `CaseManager`, `PncAdmin`,
`PncManagerAdmin`, `PncReceive`, `PNCReportClaimInternal`, dan `PNCReportClaimEksternal`.

Yang terakhir patut diperhatikan: **pelapor luar** juga membuka layar ini di sistem lama
(`Activity/CreateNewCaseRCV-Act.xml` step 4 bercabang khusus untuknya).

Keadaan ini sama dengan seluruh rute lain hari ini. Yang berubah: **taruhannya naik**. Layar ini
memuat nama tertanggung, nomor polis, kronologi kejadian, dan alamat surel — bukan master data
yang aman dilihat siapa saja.

### 33.13 Pertanyaan terbuka yang ditinggalkan sesi ini

| Pertanyaan | Pemilik | Menahan |
|---|---|---|
| Adakah pembaca `T_CLAIM_RECIVEDCLAIM` di luar export? | Work Owner + DBA | Menyalakan modul ini di produksi |
| Bentuk nomor laporan `LPK.YY.xxxx` — disetujui? | Work Owner | Tidak menahan; koreksinya satu konstanta |
| Bolehkah pustaka desimal ditambahkan sebagai dependensi? | Work Owner | Tipe kolom `NILAI_ESTIMASI` |
| ~~Bahasa penamaan modul baru — Indonesia atau Inggris?~~ | Work Owner | **TERTUTUP** — dijawab Inggris (`D-80`); lihat §33.9 |
| Persetujuan menjalankan migrasi `0003` | Work Owner + DBA (`D-63`) | Layar bekerja terhadap Oracle |
| Hak `INSERT`/`UPDATE` akun aplikasi atas tabel dan urutan baru | DBA | idem |
| Daftar pilihan `Kurir` dan `Tipe Klaim` — tidak ada di export | Work Owner | Tidak menahan; keduanya teks bebas hari ini |
| Kapan lampiran, utas komunikasi, dan penugasan menyusul | Work Owner | Paritas penuh dengan layar lama |
| Go dan Node terpasang di mesin pengembangan | Work Owner / Tim Infra | **Seluruh verifikasi otomatis** |

---

## 34. Modul View History Claim (2026-09-20, sesi kesepuluh)

### 34.1 Keputusan Work Owner pada sesi ini

| # | Keputusan | Akibatnya |
|---|---|---|
| 1 | **Gerbang proteksi data dibangun penuh** | Layar menolak pengguna yang belum terdaftar, dan satu jatah pencarian terpakai tiap kali layar dibuka |
| 2 | **Ketiga cacat aturan direplikasi apa adanya** | Pencarian Tanggal Lahir tidak pernah membuahkan hasil; kolom Posisi Klaim kosong pada pencarian Nama Objek |
| 3 | **Tipe No Rekening direplikasi apa adanya** | Dua isian tetap digambar; tipenya ditandai belum tersedia karena menembus DB Link (`R-03`) |
| 4 | **Label tipe pencarian memakai turunan dari deskripsi langkah**, kode 10 dilewati | Dua belas pilihan dengan kode `1`–`9`, `11`–`13` |
| 5 | **Penulisan gerbang diarahkan ke tabel baru milik aplikasi** | `P-1` terjaga; migrasi 0004 |
| 6 | Branch `Feat/arlexy-View-History-Claim` | — |

### 34.2 Kenapa cacat direplikasi, dan bagaimana ia dijaga tetap terlihat

`P-5` menetapkan hasil yang benar adalah hasil yang sama dengan Pega, kecuali perbaikan
yang diputuskan eksplisit. Work Owner memutuskan ketiga cacat layar ini **tidak** masuk
daftar perbaikan.

Keputusan yang direplikasi punya satu bahaya khas: ia tidak dapat dibedakan dari
kelalaian. Enam bulan lagi, seseorang yang menemukan pencarian Tanggal Lahir selalu kosong
akan "memperbaikinya" — dan tanpa menyadarinya, ia mengubah perilaku yang sengaja
dipertahankan.

Karena itu ketiganya dipagari di **empat tempat sekaligus**:

| Tempat | Bentuk |
|---|---|
| `searchtype.go` `QueryValue` | komentar menyebut langkah activity-nya dan mengapa prakondisinya tautologi |
| `riwayatklaim_test.go` | `TestPencarianTanggalLahirMemakaiIsianYangSalah` — menguji cacatnya |
| `usecase/search_test.go` | `TestPencarianTanggalLahirSelaluKosong` — data contoh MEMUAT sasarannya, dan hasilnya tetap kosong |
| layar | keterangan di bawah judul tabel menyatakan keterbatasannya kepada pengguna |

Yang keempat penting: tanpa itu, hasil yang selalu kosong akan dilaporkan berulang kali
sebagai kerusakan modul.

### 34.3 Satu hal yang TIDAK ikut direplikasi, dan alasannya

**Perangkaian nilai ke dalam teks SQL.** Kedua belas kueri lama menyisipkan nilai langsung
lewat pola `{InputData.CARI4}` dan `{ASIS:InputData.CARI4}`; yang kedua bahkan menyisipkan
**potongan SQL**, bukan nilai. Gerbang proteksinya lebih jauh lagi —
`GetFileOnPc_link_attachmentGCNM` berisi tepat satu baris penyisipan, sehingga SELURUH
SQL-nya datang dari sebuah properti klipboard.

`08-TECHNICAL-STRATEGY.md` §4.3 melarang keduanya tanpa perkecualian. Keputusan "replikasi
apa adanya" dibaca sebagai berlaku pada **perilaku bisnis**, bukan pada celah injeksi —
dan perbedaan itu dinyatakan di sini supaya ia menjadi keputusan yang tercatat, bukan
kelonggaran yang diambil diam-diam.

Seluruh nilai lewat parameter binding, dan `query_test.go` menjaganya.

### 34.4 Kenapa jatah dihitung, bukan disimpan

Sistem lama mengurangi jatah dengan `UPDATE` terhadap `POOLDATA.MST_PROTEKSI_DATA_PNC`.
Menirunya berarti aplikasi ini dan Pega sama-sama menulis satu tabel selama masa paralel —
tepat yang dilarang `P-1`.

Akibatnya bukan galat, dan itulah yang membuatnya berbahaya: kedua sistem menulis sisa
menurut hitungannya masing-masing, yang menulis belakangan menang, dan jatah seorang
pengguna berubah tanpa satu pun jejak yang menjelaskannya.

Di sini master **hanya dibaca**; pemakaian dicatat di tabel milik aplikasi, dan sisa jatah
dihitung sebagai jatah master dikurangi pemakaian yang tercatat.

**Akibat yang diterima secara sadar:** selama layar Pega dan layar ini sama-sama hidup,
seorang pengguna memperoleh jatah **lebih banyak** daripada yang tertulis di master —
sebanyak pemakaian di salah satu sistem tidak terlihat oleh yang lain. Pilihan lainnya
adalah dua sistem menulis satu tabel, yang merusak lebih dalam dan lebih sulit dilacak. Ia
berakhir dengan sendirinya saat layar Pega dimatikan.

### 34.5 Satu hal yang saya putuskan sendiri, dan dinyatakan supaya dapat dikoreksi

**Pencarian dicatat ke jejak audit meski sistem lama tidak mencatatnya di layar ini.**

Verifikasi membuktikan parameter `flagloging` tidak pernah dikirim dari layar ini,
sehingga sistem lama tidak menulis satu baris log pun saat orang mencari. Replikasi
harfiah berarti tidak mencatat apa pun.

Yang dipilih: **tetap mencatat**, dengan tiga alasan — `D-59` menjadikan jejak audit
satu-satunya kontrol pengimbang karena tidak ada pemisahan tugas; ia tidak melanggar `P-1`
karena ditulis ke tabel milik aplikasi; dan satu tabel yang sama sekaligus menjadi dasar
hitungan sisa jatah.

Baris pencarian ditandai `MEMAKAI_JATAH = 0`, sehingga ia **tidak** mengurangi jatah —
perilaku jatahnya tetap sama persis dengan sistem lama. Yang bertambah hanya jejaknya.

### 34.6 Satu perubahan terhadap kueri lama yang dituntut `D-22`

Pencarian No Klaim di sistem lama merangkai nama kelas internal Pega ke dalam kunci
pencariannya. Klaim terbitan sistem baru berformat `PNCN.YY.xxxx` dan tidak pernah menulis
awalan itu lagi, sehingga kueri lama tidak akan pernah menemukannya.

Yang dicari sekarang adalah `CLAIMNO`. Ia **setara** untuk baris warisan — awalannya tepat
19 karakter, dan kueri lama sendiri memperlakukan `SUBSTR(CLAIMID,20)` sebagai nomor klaim
pada `BroswseKlaimByPolicyNo`. Masukan yang sama menemukan baris yang sama; yang bertambah
hanya baris terbitan sistem baru.

Awalan itu **dipertahankan** pada pencarian No Survey, dan perbedaannya disengaja: yang
dirangkai di sana bukan nomor klaim melainkan `CASEID` milik baris surveyor, dan kolom itu
memang menyimpan kunci berformat Pega.

### 34.7 Penyimpangan dari dokumen Steering

| Hal | Steering | Di sini | Alasan |
|---|---|---|---|
| Paginasi | tidak diatur untuk layar ini | `OFFSET … FETCH NEXT` di server | Kueri lama menarik SELURUH baris yang cocok tanpa `MaxRecords`; terhadap `T_CLAIM_PNC` berpuluh juta baris (`D-10`) itu tidak dapat dibawa apa adanya. `09-DATABASE-STRATEGY.md` §6.3 menyatakannya **perubahan perilaku**, dan ia dicatat begitu |
| Isian wajib | sistem lama tidak memeriksa apa pun | isian yang tampak wajib diisi | Pengaman, bukan aturan bisnis: menekan Cari dengan isian kosong pada tipe Nama Customer menghasilkan pencarian berpola kosong — seluruh isi tabel |
| Urutan hasil | kueri lama tidak mengurutkan | `ORDER BY REFERENCE` | Tanpa urutan yang ditetapkan, paginasi membuat satu baris muncul di dua halaman sekaligus hilang dari halaman lain |
| `hideSearch` pada `DataTable` | komponen baku hanya menyediakan yang dipakai layar master | satu prop opsional ditambahkan | Layar ini sudah punya formulir pencarian sendiri; kotak cari kedua hanya menyaring halaman yang sedang terbuka dan hasilnya menyesatkan. Bawaannya `false`, sehingga tidak satu pun layar lama berubah |

### 34.8 Yang sengaja TIDAK dikerjakan

| Hal | Alasan |
|---|---|
| Tombol "Lihat Detail Klaim" | Layar rincian adalah `MENU_ID 75` "View Claim" — modul tersendiri yang belum dibangun. Kuncinya sudah ikut dikirim di setiap baris (`referensi`), sehingga menyalakannya kelak tidak menuntut perubahan kontrak |
| Masking KTP/telepon/surel | Tidak berlaku di layar ini: grid-nya tidak punya satu pun kolom itu. Ketiga penandanya tetap DIBACA dan dibawa di `Protection`, supaya modul rincian kelak membaca keadaan yang sama alih-alih menafsirkannya ulang |
| Pencarian No Rekening yang berfungsi | Menembus DB Link ke basis data pembayaran yang belum punya API pengganti (`R-03`) |
| Memperbaiki 3 uji `master-rekening` yang gagal | Sudah gagal di `master` sebelum sesi ini, dan berada di luar lingkup tugas. Dibuktikan dengan menjalankannya terhadap `DataTable` versi `master` |

### 34.9 Yang belum dapat dibuktikan

- **Seluruh SQL modul ini belum pernah dijalankan terhadap Oracle.** Tidak ada basis data
  di mesin tempat berkas ini ditulis, dan migrasi 0004 belum dijalankan DBA. Yang terbukti
  hanyalah bentuk kuerinya lewat `query_test.go`.
- **Nama kolom pada `MST_PROTEKSI_DATA_PNC` dibaca dari SQL dinamis di activity, bukan dari
  DDL.** DDL-nya tidak ada di export (`R-08`). Bila nama kolomnya berbeda, gerbang gagal
  dengan galat Oracle yang menyebut kolomnya — bukan gagal diam-diam.
- **Kode tipe pencarian `10` diasumsikan memang tidak ada.** Tidak ada langkah
  `Search Type 10` di activity mana pun, dan tidak ada kueri yang menganggur menunggunya.

### 34.10 Pertanyaan terbuka yang menahan tahap berikutnya

| # | Pertanyaan | Pemilik |
|---|---|---|
| 1 | Apakah Master Proteksi Data sudah berisi baris ber-`MODUL='PNCSearchKlaim'`? Bila nol, layar menolak setiap pengguna | DBA + Work Owner |
| 2 | Apakah layar Master Proteksi Data ikut dimigrasikan? Selama belum, jatah hanya dapat ditambah lewat layar Pega | Work Owner |
| 3 | Label tipe pencarian yang sebenarnya dibaca pengguna — yang dipakai sekarang turunan dari deskripsi langkah, karena definisi propertinya tidak ada di export | Tim Pega |
| 4 | Apakah cacat pencarian Tanggal Lahir kelak diperbaiki? Bila ya, ia menjadi butir baru pada daftar perbaikan eksplisit `P-5` | Work Owner |

## 19. Modul Inbox Admin (2026-09-20, sesi kesebelas)

### 19.1 Keputusan Work Owner pada sesi ini

| # | Keputusan | Akibatnya |
|---|---|---|
| 1 | **Seluruh tab dibangun sekaligus** | Delapan tab, bukan bertahap |
| 2 | **Tiga tab Komunikasi tidak dipakai** — sudah di-remark di Pega | Kueri `BrowseClaimALLKomunikasi` yang cacat sintaksis tidak dibawa sama sekali |
| 3 | **Penyaring cabang MENUNGGU API pengganti** DB Link HRD | Seam ada, adapter belum; sampai API tiba, daftar tidak dibatasi cabang |
| 4 | **Tombol "Lihat Detail Klaim" dibangun**, tombol lain tidak | Delapan tombol lain tidak dibangun; tujuannya diarahkan ke rute yang menyatakan keadaan |
| 5 | **Tab bawaan: All Case Admin** | Layar terbuka pada klaim MILIK petugas, bukan seluruh klaim berjalan |
| 6 | **Paginasi direplikasi apa adanya** | Seluruh baris ditarik, lalu dipotong di aplikasi |
| 7 | Nama modul `inboxadmin` / `inbox-admin` | `D-81` |

### 19.2 Kenapa satu bentuk baris untuk delapan tab

Kedelapan grid sistem lama membaca halaman klipboard yang sama, dan tiap tab hanya
menampilkan sebagian kolomnya. `WorkItem` karena itu satu tipe dengan 31 isian, sebagian
kosong pada tab mana pun — pola yang sama dengan `ClaimHistory` di modul View History Claim.

Alternatifnya — empat bentuk baris menurut halaman klipboard asalnya — akan menghasilkan
empat pemindai, empat DTO, dan empat tabel di layar, padahal yang berbeda hanyalah kolom
mana yang terlihat.

Yang menjaga agar isian kosong tidak terbaca sebagai data hilang: **kolom mana yang digambar
ditetapkan `Tab.Columns`, bukan ditebak dari isi baris**. Menebaknya dari isi akan membuat
kolom menghilang ketika seluruh baris halaman itu kebetulan kosong.

### 19.3 Satu alias, dua arti — dan kenapa itu tidak dibawa

Ini yang membedakan modul ini dari yang mana pun sebelumnya:

| Properti Pega | di tab ALL | di tab Request Survey |
|---|---|---|
| `.RCVID` | `A.SOBNAME` (sumber bisnis) | `T_REQ_SURVEY.SURVEYOR` (nama surveyor) |
| `.Keterangan` | `PXCREATEOPERATOR` (pembuat) | `T_REQ_SURVEY.BRANCH` (cabang survei) |
| `.Kurir` | `B.PXFLOWNAME` (nama flow) | `A.BRANCHNAME` (cabang polis) |
| `.UserAdmin` | nama cabang klaim | `SUBSTR(SURVEYID,20,30)` (no survei) |

Di View History Claim, satu alias salah arti tetapi setidaknya **konsisten**. Di sini ia
berubah arti tergantung tab yang sedang terbuka — sehingga membawanya berarti membangun
sistem baru yang tidak dapat dijelaskan tanpa menyebut tab mana yang dimaksud.

Pemetaan tiga arahnya dicatat di kepala `inboxadmin.sql`, satu-satunya tempat ketiganya
dapat dibandingkan berdampingan.

### 19.4 Paginasi direplikasi apa adanya — dan keberatannya dicatat

Keberatan disampaikan **sebelum** keputusan diambil, beserta angkanya: dengan penyaring
cabang yang belum aktif, tab ALL menarik seluruh klaim yang masih berjalan ke memori
aplikasi, dan tabelnya berisi puluhan juta baris (`D-10`). Work Owner memilih replikasi.
Keputusan itu dihormati dan dijalankan.

Yang ditambahkan sebagai gantinya **tidak mengubah perilaku sama sekali**: satu peringatan
di log begitu satu permintaan melampaui `LargeResultWarning` (5.000 baris), menyebut tab,
jumlah baris, dan sebabnya. Memotong hasil akan menyalahi keputusan; memperingatkan membuat
akibatnya terlihat operator sebelum terlihat sebagai aplikasi yang kehabisan memori.

Satu hal yang **tidak** direplikasi: kueri `GetAllCaseAdmin` dan
`GetRequestDokumenKomunikasi` tidak mengurutkan hasilnya sama sekali. Itu dapat dibiarkan
selama hasilnya tidak dipaginasi; begitu halamannya dipotong, urutan yang tidak ditetapkan
membuat satu baris muncul di dua halaman sekaligus hilang dari halaman lain. `ORDER BY`
ditambahkan pada keduanya.

### 19.5 Tiga tab yang tidak dibangun, dan bagaimana ketiadaannya dijaga terbaca

Work Owner menyatakan ketiganya sudah di-remark di Pega. Pembacaan export mendukungnya dari
dua sisi: ada **empat elemen ber-`pyCondition` `1==2`** di section itu, dan kuerinya memang
**cacat sintaksis** — UNION dengan 15 kolom di satu cabang dan 14 di cabang lain.

Ketiadaannya dipagari di **tiga tempat**, karena pertanyaan "kenapa tab Komunikasi tidak
ada" pasti diajukan lagi:

| Tempat | Bentuk |
|---|---|
| `tab.go` `DisabledTabs` | data, bukan komentar — ikut terkirim ke layar |
| `NewQuery` | tautan lama ber-`tab=4` dijawab dengan ALASANNYA, bukan "tab tidak dikenal" |
| layar | disebut di bawah tabel, bukan digambar sebagai tab mati yang mengundang klik |

Ditambah `query_test.go` yang **gagal** bila suatu saat ada kueri menganggur untuk ketiganya.

### 19.6 Batas yang saya tarik sendiri pada "replikasi apa adanya"

Dua hal sengaja TIDAK direplikasi, dan keduanya dinyatakan di sini supaya menjadi keputusan
yang tercatat, bukan kelonggaran yang diambil diam-diam.

**Pertama, perangkaian nilai ke dalam teks SQL.** Ketujuh kueri lama menyusun klausa
WHERE-nya di activity lalu menyisipkannya lewat tujuh titik `ASIS` — termasuk klausa
paginasinya sendiri. `08-TECHNICAL-STRATEGY.md` §4.3 melarangnya tanpa perkecualian, dan
keputusan "replikasi apa adanya" dibaca sebagai berlaku pada **perilaku bisnis**, bukan pada
celah injeksi. Sama seperti sesi sebelumnya.

**Kedua, perpindahan tab diam-diam saat mencari.** Sistem lama menyetel ulang tab menjadi
`3` atau `7` menurut ada-tidaknya teks "PNC" di kotak cari. Ia bagian dari alur dua mode
milik Pega, dan di layar React ia akan tampil sebagai tab yang berpindah sendiri saat
pengguna mengetik. Tidak dibawa.

### 19.7 Aging dihitung hari kalender, dan itu dinyatakan ke pengguna

Sistem lama menghitungnya lewat `GCNMTimeDifferenceWorkCalender_Act`, yang membaca kalender
libur `GENERAL.HRD_LBR` lewat DB Link. Keputusan Work Owner menunda seluruh sambungan itu
sampai API penggantinya ada (`R-03`), sehingga kalender kerja belum tersedia.

Yang dipilih: **hitung hari kalender, dan nyatakan keterbatasannya di layar.** Mengganti
hari kerja dengan hari kalender secara diam-diam adalah kelas perubahan perilaku yang paling
berbahaya — angkanya masuk akal, hanya lebih besar, dan tidak ada yang menyadarinya sampai
seseorang membandingkannya dengan Pega.

Daftar keterbatasannya dikirim **server**, bukan ditulis tetap di layar, supaya hilang dengan
sendirinya begitu penghalangnya hilang — tanpa menyunting frontend.

### 19.8 Aging dihitung di Go, bukan di SQL

Dua alasan. Hasilnya dapat diuji secara deterministik lewat seam `Clock`, dan kuerinya tetap
portabel tanpa fungsi tanggal khas Oracle.

Selisihnya dihitung terhadap **tanggal**, bukan timestamp: baris yang masuk pukul 23.00 dan
dibaca pukul 01.00 esok harinya berumur satu hari, bukan nol hari
(`08-TECHNICAL-STRATEGY.md` §4.4). Tanggal di masa depan menghasilkan nol, bukan angka
negatif — baris seperti itu ada di data warisan, dan "minus tiga hari" tidak berarti apa pun
bagi petugas yang membaca kolom tenggat.

### 19.9 Kontrak API modul ini

| Metode | Jalur | Isi |
|---|---|---|
| `GET` | `/api/inbox-admin/tab` | `tab[]`, `tab_bawaan`, `lini_bisnis[]`, `tab_dinonaktifkan[]`, `keterbatasan[]`, `portal` |
| `GET` | `/api/inbox-admin` | `tab`, `baris[]`, `paginasi`, `penyaring`, `portal` |

Parameter: `tab`, `bisnis`, `cari`, `halaman`, `ukuran`.

**Bentuk layar datang dari server**, termasuk daftar kolom tiap tab. Alasannya bukan
kerapian: daftar itu adalah hasil pembacaan export Pega, dan tempat pembacaan itu tercatat
adalah backend. Menyalinnya ke layar berarti daftar yang sama hidup di dua tempat, dan yang
satu akan tertinggal saat yang lain diperbaiki.

**`penyaring` dikembalikan** karena tidak selalu sama dengan yang diminta: tab yang tidak
mendukung pencarian mengembalikan kata kunci kosong, sehingga layar dapat membersihkan
kotaknya alih-alih menampilkan kata kunci yang tampak aktif padahal tidak menyaring apa pun.

### 19.10 Otorisasi: keadaan yang belum berubah

Rutenya terlindungi sesi dan pemeriksaan portal. Pemeriksaan peran — "apakah peran pemanggil
memiliki menu ini" — adalah `TKT-F3-005` yang belum ada.

Satu pembedaan sistem lama karena itu belum dibawa: layar ini memperlakukan
`GCNMFW:CaseManager` dan `GCNMFW:PncManagerAdmin` berbeda — keduanya melewati penyaring
cabang dan memperoleh pemilih korwil. Akibatnya untuk sekarang **seluruh pengguna
berperilaku seperti manajer**, yaitu tanpa penyaring cabang. Itu kebetulan sejalan dengan
keputusan menunda penyaring cabang, dan keduanya akan selesai bersama-sama.

### 19.11 Tombol "Lihat Detail Klaim" dan tujuannya

Work Owner memutuskan tombolnya dibangun; layar tujuannya `MENU_ID 75` "View Claim" belum
ada. Tiga pilihan, dua di antaranya buruk:

| Pilihan | Akibat |
|---|---|
| Tombol menuju rute tak terdaftar | Pengguna terlempar ke beranda tanpa penjelasan — tampak rusak |
| Tombol mati | Tidak memenuhi keputusan, dan menyembunyikan bahwa jalurnya sudah lengkap |
| **Tujuan yang menyatakan keadaan** | Yang dipakai |

`src/app/ViewClaimPlaceholder.tsx` karena itu ada, dan ia **tidak memanggil server sama
sekali**. Menampilkan sebagian data klaim di layar sementara akan menciptakan kontrak yang
harus dipelihara, untuk layar yang seluruh bentuknya belum dirancang. Berkas itu **akan
dihapus** saat modul View Claim lahir.

Yang dikirim adalah `referensi` — kunci teknis Pega yang memang dipakai `setDataViewKlaim_Act`
di sistem lama — sehingga menyalakan layar rincian kelak tidak menuntut perubahan kontrak.

### 19.12 Pertanyaan terbuka yang ditinggalkan sesi ini

| # | Pertanyaan | Pemilik |
|---|---|---|
| 1 | Apakah ketiga tab Komunikasi memang gagal di Pega produksi (ORA-01789), dan perlukah dilaporkan ke Tim Pega? | Work Owner + Tim Pega |
| 2 | Kapan API pengganti DB Link HRD tersedia? Sampai itu ada, penyaring cabang dan Aging hari kerja keduanya tertahan | Tim pemilik sistem HRD |
| 3 | Delapan tombol yang belum dibangun — mana yang menyusul, dan dalam urutan apa? | Work Owner |
| 4 | Nilai `STATUSLOD` berarti terbalik antara lini OTO/BFI dan lini lain. Direplikasi apa adanya; perlukah ia masuk daftar perbaikan `P-5`? | Work Owner |
