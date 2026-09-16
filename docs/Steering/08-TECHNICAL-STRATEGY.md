# Technical Strategy, Folder Structure & Coding Standards

Dokumen paling preskriptif dalam Steering ini. Ketegasannya disengaja: D-09 menetapkan tim
adalah developer Pega yang belum terbiasa Go maupun JavaScript modern, dan D-23 memilih React
yang tidak opinionated. Gabungan keduanya punya satu mode kegagalan yang sangat mungkin terjadi:
**setiap layar ditulis dengan gaya berbeda oleh orang berbeda**, lalu tidak ada yang bisa
merawatnya.

Aturan di sini **mengikat**, bukan anjuran. Pelanggaran ditolak di code review.

---

## 1. Tumpukan teknologi

### Backend

| Bagian | Pilihan | Alasan |
|---|---|---|
| Bahasa | **Go 1.22+** | Ditetapkan pemilik project |
| Router HTTP | **`net/http` + `chi`** | `chi` tipis, idiomatik, tanpa konsep baru di luar `net/http` bawaan. Framework besar seperti Gin/Echo memperkenalkan konteks dan middleware sendiri — beban belajar tambahan yang tidak perlu |
| Akses database | **`database/sql` + driver** | SQL ditulis langsung (D-20). Tanpa ORM |
| Driver Oracle | **`godror`** | Paling matang untuk Oracle |
| Driver PostgreSQL | **`pgx`** (mode `database/sql`) | Standar de-facto |
| Migrasi skema | **`golang-migrate`** | Berkas SQL polos, mudah dibaca dan di-review |
| Logging | **`log/slog`** (pustaka standar) | Terstruktur, tanpa dependensi tambahan |
| Konfigurasi | **Variabel lingkungan + berkas YAML** | — |
| Validasi | **Kode eksplisit di modul domain** | Bukan tag struct. Aturan bisnis harus terbaca sebagai kalimat, bukan tersembunyi di anotasi |
| Pengujian | **`testing` bawaan + `testify/require`** | Tanpa framework BDD |
| PDF | **`maroto`** atau **`gopdf`** | D-11 |
| Excel | **`excelize`** | D-11 |
| Penjadwalan | **`robfig/cron`** | Sederhana, cukup untuk kebutuhan S-6 |

> **Prinsip pemilihan pustaka:** utamakan pustaka standar. Setiap dependensi pihak ketiga adalah
> satu hal lagi yang harus dipelajari tim, dipantau keamanannya, dan bisa ditinggalkan
> pemeliharanya.

### Frontend

| Bagian | Pilihan | Alasan |
|---|---|---|
| Framework | **React 18 + TypeScript** | D-23 |
| Build | **Vite** | Cepat, konfigurasi sederhana |
| Routing | **React Router** | Standar de-facto |
| Data server | **TanStack Query** | Menyeragamkan pengambilan data, cache, dan status loading. **Wajib** — inilah yang mencegah 74 layar punya cara berbeda memanggil API |
| Tabel | **TanStack Table** atau **AG Grid** | D-23. Satu pilihan saja, tidak keduanya |
| Form | **React Hook Form + Zod** | Validasi terketik, sinkron dengan kontrak API |
| State global | **Zustand** | Hanya untuk state yang benar-benar global (pengguna, izin). Selebihnya state server ditangani TanStack Query |
| Styling | **Tailwind CSS** | Konsisten tanpa berdebat penamaan kelas CSS |
| Pengujian | **Vitest + Testing Library** | — |

> **Yang dilarang di frontend:** memanggil `fetch` langsung di komponen (harus lewat TanStack
> Query), menyimpan data server di Zustand, dan membuat komponen tabel baru di luar pustaka
> komponen baku (U-2).

---

## 2. Struktur folder — Backend

```
claim-pnc/
├── cmd/
│   └── server/                  titik masuk aplikasi — sangat tipis
│
├── internal/                    seluruh kode aplikasi; tidak bisa diimpor project lain
│   │
│   ├── domain/                  ── LAPISAN DOMAIN ──
│   │   │                        aturan bisnis murni
│   │   │                        DILARANG mengimpor: HTTP, SQL, driver, JSON wire
│   │   ├── klaim/
│   │   │   ├── klaim.go             tipe agregat & invarian
│   │   │   ├── registrasi.go        modul Registrasi Klaim
│   │   │   ├── validasi_tanggal.go  aturan tanggal (Business Understanding §3.1)
│   │   │   ├── validasi_duplikat.go aturan duplikasi (§3.2)
│   │   │   ├── status.go            empat konsep status (D-18)
│   │   │   ├── errors.go            kesalahan domain
│   │   │   └── repository.go        SEAM — interface, bukan implementasi
│   │   ├── polis/                   snapshot polis (D-04)
│   │   ├── objek/                   objek pertanggungan & coverage
│   │   ├── spreading/               aturan 100%, Fac Out, Ex-Gratia
│   │   ├── settlement/              estimasi → usulan → akseptasi → bayar
│   │   ├── komite/                  penjenjangan 1–4 level
│   │   ├── penugasan/               worklist & workbasket (D-26)
│   │   ├── survei/
│   │   ├── reasuransi/              PLA · Pre-DLA · DLA
│   │   ├── salvage/
│   │   ├── dokumen/
│   │   ├── otorisasi/               izin menu & aksi (D-07)
│   │   ├── masterdata/
│   │   └── audit/                   jejak audit append-only (D-28)
│   │
│   ├── app/                     ── LAPISAN APLIKASI ──
│   │   │                        orkestrasi lintas modul domain + transaksi
│   │   ├── registrasiklaim/
│   │   ├── prosesakseptasi/
│   │   └── ...
│   │
│   ├── adapter/                 ── LAPISAN ADAPTER ──
│   │   ├── sqlstore/                implementasi Repository (SQL portabel)
│   │   │   ├── klaim.go
│   │   │   ├── klaim.sql            ← query terpisah dari kode Go
│   │   │   ├── penugasan.go
│   │   │   ├── penugasan.sql
│   │   │   └── nomor_klaim.go       ← SATU-SATUNYA sakelar dialek (D-22)
│   │   ├── hccclient/               autentikasi HCC/HCQ (D-07)
│   │   ├── docstore/                storage dokumen internal (D-16)
│   │   ├── brisurf/                 integrasi BRI Surf
│   │   ├── kasir/
│   │   ├── slikojk/
│   │   ├── smtp/                    adapter Notifier
│   │   └── clock/                   adapter Clock (F-5)
│   │
│   ├── transport/               ── LAPISAN TRANSPORT ──
│   │   └── http/
│   │       ├── handler/             satu berkas per sumber daya
│   │       ├── middleware/          auth · logging · recovery · request ID
│   │       ├── dto/                 bentuk request & response — TERPISAH dari tipe domain
│   │       └── router.go
│   │
│   ├── report/                      engine PDF · Excel · CSV (D-11)
│   ├── scheduler/                   job terjadwal (S-6)
│   └── platform/
│       ├── config/
│       ├── logging/
│       ├── database/                koneksi & connection pool
│       └── errs/                    tipe kesalahan bersama
│
├── migrations/                      berkas SQL golang-migrate
├── web/                             hasil build SPA React (disematkan ke binary)
├── docs/
└── Steering/                        dokumen ini
```

### Aturan struktur yang mengikat

1. **`domain/` tidak boleh mengimpor `adapter/`, `transport/`, maupun pustaka database.**
   Ditegakkan otomatis oleh linter (`depguard`), bukan hanya oleh kesepakatan.
2. **Interface dideklarasikan di paket yang memakainya**, bukan di paket yang mengimplementasikannya.
   `domain/klaim/repository.go` mendeklarasikan apa yang dibutuhkan Klaim; `adapter/sqlstore`
   memenuhinya. Inilah yang membuat seam berada di tempat yang benar.
3. **Satu berkas = satu tanggung jawab.** Berkas melebihi ~400 baris adalah tanda modul perlu
   dipecah.
4. **DTO transport tidak sama dengan tipe domain.** Memakai tipe domain langsung sebagai bentuk
   JSON membuat perubahan internal bocor ke klien dan sebaliknya.
5. **SQL berada di berkas `.sql` terpisah**, bukan sebagai string di tengah kode Go. Query bisa
   dibaca, di-review, dan diuji langsung terhadap database.

---

## 3. Struktur folder — Frontend

```
web/src/
├── app/                     kerangka: router, provider, layout, guard
├── shared/
│   ├── api/                 klien HTTP + tipe hasil generate dari kontrak API
│   ├── components/          ── PUSTAKA KOMPONEN BAKU (U-2) ──
│   │   ├── DataTable/           satu-satunya tabel di seluruh aplikasi
│   │   ├── Form/                field, label, pesan kesalahan
│   │   ├── FileUpload/
│   │   ├── DatePicker/
│   │   └── ...
│   ├── hooks/
│   └── lib/                 format tanggal, angka, mata uang — SATU tempat
│
└── features/                satu folder per modul bisnis
    ├── registrasi-klaim/
    │   ├── api/                 hook TanStack Query untuk fitur ini
    │   ├── components/          komponen khusus fitur ini
    │   ├── pages/
    │   └── types.ts
    ├── inbox/
    ├── komite/
    ├── akseptasi/
    ├── survei/
    ├── laporan/
    └── master-data/
```

**Aturan mengikat:**
1. **Fitur tidak boleh mengimpor dari fitur lain.** Kebutuhan bersama naik ke `shared/`.
2. **Semua tabel memakai `shared/components/DataTable`.** Tidak ada `<table>` mentah di
   folder `features/`. Ini yang mengubah 268 grid menjadi satu implementasi.
3. **Semua pemanggilan API lewat hook TanStack Query** di `features/*/api/`. Tidak ada `fetch`
   di dalam komponen.
4. **Semua format tanggal, angka, dan mata uang lewat `shared/lib`.** Bukan format lokal per
   komponen — inilah yang mencegah terulangnya masalah `TO_CHAR` tersebar (utang teknis 4.4).

---

## 4. Coding Standards — Backend

### 4.1 Penamaan

| Hal | Aturan | Contoh |
|---|---|---|
| Paket | Kata benda tunggal, huruf kecil, tanpa garis bawah | `klaim`, `spreading`, `settlement` |
| Berkas | `snake_case.go` | `validasi_tanggal.go` |
| Tipe & fungsi ekspor | `PascalCase` | `Klaim`, `RegistrasiKlaim` |
| Interface | Nama peran, bukan berakhiran `Interface` | `KlaimRepository`, bukan `IKlaimRepository` |
| Istilah domain | **Ikuti `CONTEXT.md` tanpa perkecualian** | `ObjekPertanggungan`, bukan `Object` · `SettlementLine`, bukan `Adjustment` |

**Bahasa penamaan:** istilah domain memakai **bahasa Indonesia** sesuai `CONTEXT.md`, karena
itulah bahasa yang dipakai bisnis dan tim. Istilah teknis memakai **bahasa Inggris** mengikuti
konvensi Go. Contoh: `type Klaim struct` dengan method `Validate()`.

Alasannya: mencampur istilah domain berbahasa Inggris yang salah terjemah (seperti `Adjustment`
yang ternyata berarti nilai penyelesaian) adalah tepat sumber kekacauan yang sedang kita
perbaiki.

### 4.2 Penanganan kesalahan

- Kesalahan selalu dikembalikan, **tidak pernah** `panic` di jalur normal.
- Setiap kesalahan **dibungkus dengan konteks** saat naik ke pemanggil, sehingga jejaknya
  terbaca dari pesan.
- Kesalahan domain adalah **tipe tersendiri**, bukan string. Transport memetakannya ke kode HTTP.
- Kesalahan validasi bisnis **dikumpulkan seluruhnya**, tidak berhenti pada yang pertama.
  Ini meniru perilaku Pega yang menampilkan semua pesan sekaligus — mengubahnya jadi
  satu-per-satu akan sangat menyiksa pengguna pada form registrasi yang panjang.
- Kesalahan **tidak pernah** ditelan diam-diam. Tidak ada `_ = err`.

### 4.3 Aturan SQL

Ini yang membuat D-20 bisa dijalankan.

| Aturan | Alasan |
|---|---|
| **Selalu parameter binding**, tidak pernah merangkai SQL dari string | Menutup celah SQL injection yang ada di pola `{ASIS:...}` warisan (utang teknis 4.5) |
| `COALESCE`, bukan `NVL` | Portabilitas |
| `CURRENT_TIMESTAMP`, bukan `SYSDATE` | Portabilitas |
| `CASE WHEN`, bukan `DECODE` | Portabilitas |
| `OFFSET … FETCH NEXT … ROWS ONLY`, bukan `ROWNUM` | Portabilitas; pola ini sudah dipakai di 35 rule lama |
| `POSITION`, bukan `INSTR` | Portabilitas |
| `STRING_AGG`, bukan `LISTAGG` | Portabilitas |
| `LEFT JOIN`, bukan `(+)` | Portabilitas |
| Hilangkan `FROM DUAL` | Portabilitas |
| **Tanpa `TO_CHAR` untuk pemformatan tampilan** | Format tanggal dan angka dilakukan di Go |
| **Tanpa pemanggilan stored procedure** | D-02 |
| `JSON_VALUE` / `JSON_TABLE` boleh dipakai | Portabel bila PostgreSQL 17+ (D-24) |
| Satu-satunya sakelar dialek: generator nomor klaim | D-22 |

**Kolom yang dipilih harus disebutkan namanya.** `SELECT *` dilarang — kolom baru di database
tidak boleh diam-diam mengubah perilaku aplikasi.

### 4.4 Waktu

- Disimpan **UTC** di database.
- Ditampilkan **WIB (Asia/Jakarta)**.
- Konversi hanya di modul Clock (F-5).
- **Tidak ada penambahan 7 jam manual di mana pun.** Ini pelanggaran yang otomatis ditolak
  review.
- Aturan berbasis hari kalender dihitung terhadap tanggal WIB.

### 4.5 Transaksi

- Transaksi dimulai dan diakhiri di **lapisan aplikasi** (`internal/app/`), bukan di dalam
  repository maupun handler.
- Satu permintaan pengguna = satu transaksi, kecuali ada alasan yang didokumentasikan.
- Pemanggilan sistem eksternal **tidak boleh berada di dalam transaksi database** — kegagalan
  jaringan tidak boleh menahan kunci baris.

**Kepemilikan transaksi berpindah sepenuhnya ke Go** (`D-68`). Ini bukan penegasan gaya kode
melainkan perbaikan nyata: **10 dari 12 procedure** yang dibaca melakukan `COMMIT` sendiri —
`Database/INSERT_PLADLA.prc` **sembilan kali** (`:69`, `:74`, `:79`, `:138`, `:143`, `:148`,
`:179`, `:184`, `:189`), dengan satu-satunya `ROLLBACK` di handler terluar (`:198`) yang
**terjadi setelah commit** sehingga tidak memulihkan apa pun.

| Yang ini lepaskan | Isi |
|---|---|
| **`B-4` Spreading dan `B-9` PLA/DLA dapat dibuat atomik** | penerbitan yang dulu menempuh sembilan `COMMIT` dan bisa berhenti setengah jalan kini dibungkus satu transaksi |
| **Kontrak galat berbasis string `ErrMsg` tidak dibawa** | pada enam procedure `ErrMsg` **tidak di-set pada jalur sukses** sehingga `NULL` berarti berhasil; pada `ADD_NEWMASTERVIRTUALACCOUNT.prc:18` kolom yang sama membawa **nomor virtual account sekaligus pesan galat** |
| **Procedure boleh ditinggalkan** | Claim PNC adalah satu-satunya pemanggilnya (`D-68`) |

> **Konsekuensi untuk uji kesetaraan:** perubahan dari sembilan commit menjadi satu transaksi
> **mengubah perilaku saat gagal** — sistem lama meninggalkan sebagian data, sistem baru tidak
> meninggalkan apa pun. Kasus uji harus dirancang menyadari ini, atau ia akan melaporkan selisih
> palsu.
>
> **Yang belum diverifikasi:** klaim "Claim PNC satu-satunya pemanggil" **belum dibuktikan dengan
> kueri katalog**. Satu kueri `ALL_DEPENDENCIES` cukup, dan harus dijalankan **sebelum** procedure
> benar-benar dinonaktifkan — bukan sebelum logikanya ditulis ulang.

### 4.6 Yang dilarang

| Dilarang | Alasan |
|---|---|
| Variabel global yang bisa diubah | Menghancurkan kemampuan uji dan aman-konkuren |
| Nilai bisnis di-hardcode | D-15 |
| Komentar `// TESTING` di jalur produksi | Utang teknis 4.3 tidak boleh terulang |
| Kredensial di dalam kode | Keamanan |
| Fungsi melebihi ~80 baris | Tanda ada modul yang belum dipisahkan |
| `interface{}` / `any` tanpa alasan tertulis | Menghilangkan manfaat tipe |
| Membaca konteks pengguna dari variabel global | Harus lewat `context.Context` |

---

## 5. Coding Standards — Frontend

| Aturan | Alasan |
|---|---|
| **TypeScript mode ketat**, `any` dilarang tanpa alasan tertulis | 74 layar dikerjakan tim yang sedang belajar — tipe adalah jaring pengaman |
| Tipe API **dihasilkan dari kontrak backend**, tidak ditulis tangan | Backend dan frontend tidak boleh berbeda persepsi |
| Komponen adalah fungsi, tanpa class component | Satu cara saja |
| Seluruh data server lewat **TanStack Query** | Menyeragamkan loading, error, cache, dan refetch di 74 layar |
| Seluruh tabel lewat **`shared/components/DataTable`** | 268 grid menjadi satu implementasi |
| Seluruh form lewat **React Hook Form + Zod** | Validasi seragam dan terketik |
| Paginasi **selalu server-side** | D-10: puluhan juta baris |
| Berkas komponen melebihi ~200 baris harus dipecah | Keterbacaan |
| Tanpa CSS inline; memakai Tailwind | Konsistensi |

---

## 6. Penegakan otomatis

Aturan yang hanya ada di dokumen akan dilanggar. Yang berikut **wajib berjalan di CI dan
memblokir merge**:

| Alat | Menegakkan |
|---|---|
| `gofmt` + `goimports` | Format Go |
| `golangci-lint` | Kualitas kode Go |
| `depguard` | **Aturan ketergantungan antar lapisan** (§2 aturan 1) |
| `go vet` | Kesalahan umum |
| `go test ./...` | Seluruh test lulus |
| ESLint + `@typescript-eslint` | Kualitas kode frontend |
| `tsc --noEmit` | Tidak ada kesalahan tipe |
| Prettier | Format frontend |
| Pemeriksaan pola SQL terlarang | `SELECT *`, `NVL`, `ROWNUM`, `SYSDATE`, `TO_CHAR`, perangkaian string SQL |

> Pemeriksaan pola SQL terlarang tampak sepele, tapi inilah yang benar-benar menjaga D-20 tetap
> berlaku setelah bulan ketiga — ketika tekanan jadwal membuat orang menempuh jalan pintas.
