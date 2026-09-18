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
