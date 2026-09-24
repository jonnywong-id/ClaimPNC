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
