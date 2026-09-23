# Peta Penamaan — Indonesia ke Inggris

Ditetapkan Work Owner 2026-09-18: **penamaan folder, berkas, dan identifier di dalam kode memakai
bahasa Inggris.** Dokumen ini adalah kamus yang dipakai saat penggantian, dan menjadi acuan untuk
kode yang ditulis sesudahnya.

## Yang berubah, dan yang TIDAK

| Hal | Bahasa | Alasan |
|---|---|---|
| Nama folder dan berkas | **Inggris** | Keputusan Work Owner 2026-09-18 |
| Identifier: paket, tipe, fungsi, variabel, field | **Inggris** | idem |
| **Komentar di dalam kode** | **tetap Indonesia** | Keputusan Work Owner — komentar menjelaskan alasan keputusan, dan menerjemahkannya menggeser nuansa istilah domain |
| **Dokumen di `claim-pnc/docs/`** | **tetap Indonesia** | Rekaman sesi yang sudah terjadi; menerjemahkannya berarti menulis ulang catatan |
| **Nama field JSON API** | **tetap Indonesia** | Keputusan Work Owner — kontrak API tidak diusik |
| **Nama tabel dan kolom basis data** | **tetap Indonesia** | idem. Tabel warisan Pega memang tidak boleh disentuh sama sekali |
| **Teks yang dilihat pengguna** | **mengikuti layar Pega**; yang tidak ada di export Pega dikoreksi ke Inggris | Keputusan Work Owner — `D-13` menetapkan tampilan meniru Pega supaya pengguna tidak belajar ulang |

> **Akibat yang disengaja:** satu berkas dapat memuat identifier Inggris, komentar Indonesia, tag
> JSON Indonesia, dan teks layar Indonesia sekaligus. Itu bukan ketidakkonsistenan yang terlewat —
> keempatnya punya pembaca yang berbeda: pengembang, pengembang, klien API, dan staf klaim.

## Istilah domain

| Indonesia | Inggris | Catatan |
|---|---|---|
| Pengguna | User | |
| Sesi | Session | |
| Identitas | Identity | NIK karyawan atau LOGIN_ID non-karyawan |
| Kredensial | Credential | |
| NamaPengguna | Username | |
| KataSandi | Password | |
| SidikKataSandi | PasswordDigest | sidik SHA-256, bukan kata sandinya |
| SidikToken | TokenDigest | |
| Profil | Profile | |
| Portal | Portal | sudah Inggris |
| StatusKlaim | ClaimStatus | |
| Rekening | BankAccount | rekening bank penerima ganti rugi |
| Bank | Bank | sudah Inggris |
| Kasir | Cashier | |
| Komite | Committee | |
| Pengajuan | Submission | |
| Pengaju | Submitter | |
| Keputusan | Decision | |
| BukuRekening | Passbook | bukti fisik rekening |
| Notifikasi | Notification | |
| Warisan | Legacy | tabel milik sistem lama |

## Kata kerja dan operasi

| Indonesia | Inggris | Catatan |
|---|---|---|
| Baru | New | pembentuk: `LayananBaru` → `NewService` |
| Daftar | List | mengembalikan banyak baris |
| Ambil | Get | mengembalikan satu baris |
| Cari | Find | pencarian yang boleh tidak ketemu |
| Sisip | Insert | lapisan repo |
| Perbarui | Update | lapisan repo |
| Tambah | Create | lapisan usecase — dibedakan dari `Insert` supaya lapisannya terbaca |
| Ubah | Update | lapisan usecase |
| Simpan | Save | |
| Hapus | Delete | **tidak dipakai** — master tidak dihapus (`ADR-0012`) |
| Masuk | Login | |
| Keluar | Logout | |
| Perpanjang | Renew | |
| Cabut | Revoke | |
| Periksa | Check | pemeriksaan yang mengembalikan keadaan |
| Verifikasi | Verify | pemeriksaan kredensial |
| Ajukan | Submit | |
| Putuskan | Decide | |
| Muat | Load | |
| Pasang | Mount | pendaftaran rute |
| Buka / Tutup | Open / Close | |
| Samarkan | Mask | |
| Ringkas | Summary | `Ringkas()` → `Summary()` |

## Istilah teknis

| Indonesia | Inggris |
|---|---|
| Galat | Error (`GalatValidasi` → `ValidationError`, `ErrX` tetap `ErrX`) |
| Pelanggaran | Violation |
| Peringatan | Warning |
| Layanan | Service |
| Opsi | Options |
| Bahan | Deps |
| Hasil | Result |
| Konteks (pemanggil) | Caller |
| Pemanggil | Caller |
| Penulis | Writer |
| Penerima / Pengirim | Recipient / Sender |
| Konfigurasi | Config |
| Lingkungan | Environment |
| Basisdata | Database |
| Kumpulan (koneksi) | Pool |
| Jam / Waktu / Sekarang | Clock / Time / Now |
| BerlakuSampai | ExpiresAt |
| DiterbitkanPada | IssuedAt |
| DicabutPada | RevokedAt |
| DibuatPada / DiperbaruiPada | CreatedAt / UpdatedAt |
| MasaBerlaku | Lifetime |
| SisaBerlaku | Remaining |
| Kueri | Query |
| Rute | Routes |
| Tiruan | Fake |
| Memori | Memory |
| Lokal | Local |
| Berantai / MataRantai | Chain / Link |
| Contoh | Sample |
| Lengkap / Kosong | Complete / Empty |
| Bersih | Clean |
| Aktif / Tersedia / Siap | Active / Available / Ready |
| Utama | Primary |
| Jenis | Kind |
| Nilai | Value |
| Jumlah | Count |
| Urutan | Sequence (urutan nomor) · SortOrder (arah pengurutan tabel) |

## Istilah antarmuka

| Indonesia | Inggris |
|---|---|
| Halaman | Page |
| Beranda | Home |
| KerangkaHalaman | PageShell |
| PenjagaSesi | SessionGuard |
| PeringatanSesi | SessionWarning |
| BilahAtas | TopBar |
| Merek | Brand |
| Navigasi | Navigation |
| ChipPengguna | UserChip |
| PemilihPortal | PortalPicker |
| DaftarPortal | PortalList |
| KolomIsian | Field |
| PesanGalat | ErrorMessage |
| TabelData | DataTable |
| Tombol | Button |
| Ikon | Icon |
| Gaya | Styles |
| Nada | Tone |
| Judul / Keterangan | Title / Description |
| Aksi | Actions |
| Anak | Children |
| Baris / Kolom | Row / Column |
| Tampil | Render |
| Lebar | Width |
| SedangMemuat | IsLoading |
| KeadaanKosong / KeadaanMemuat | EmptyState / LoadingState |
| PenandaUrutan | SortMarker |
| Pemutar | Spinner |
| Inisial | Initials |
| Isian | Values |
| Keadaan | State |
| `gunakanX` (hook) | `useX` |

## Nama uji

Nama fungsi uji ikut diterjemahkan karena ia identifier. Kalimatnya dipertahankan sebagai
**kalimat yang menyatakan aturan**, bukan diringkas menjadi nama teknis — itu yang membuat daftar
uji terbaca sebagai dokumentasi aturan yang selalu mutakhir
(`docs/Steering/14-TESTING-STRATEGY.md` §3.2).

Contoh:

| Sebelum | Sesudah |
|---|---|
| `TestKredensialKosongDijawabSamaDenganKredensialSalah` | `TestEmptyCredentialAnsweredSameAsWrongCredential` |
| `TestSebelasKodePertamaMembawaPenomoranLama` | `TestFirstElevenCodesCarryLegacyNumbering` |
| `TestSeamTidakMenyediakanOperasiHapus` | `TestSeamProvidesNoDeleteOperation` |

---

## Peta folder dan berkas — hasil akhir

Ditetapkan saat penggantian dijalankan pada 2026-09-18. Tabel ini adalah **keadaan akhir**, bukan
usulan.

### Backend

| Sebelum | Sesudah |
|---|---|
| `internal/platform/waktu/{jam,sistem}.go` | `internal/platform/clock/{clock,system}.go` |
| `internal/masterstatus/**` | `internal/masterstatus/**` — nama modul dikembalikan (`D-81`) |
| `internal/masterrekening/**` | `internal/masterrekening/**` — nama modul dikembalikan (`D-81`) |
| `internal/masterrekening/kasir/` | `internal/masterrekening/cashier/` |
| `internal/masterrekening/notifikasi/` | `internal/masterrekening/notification/` |
| `*/repo/memori/memori.go` | `*/repo/memory/memory.go` |
| `*/http/{rute,galat}.go` | `*/http/{routes,errors}.go` |
| `auth/{identitas,pengguna,sesi}.go` | `auth/{identity,user,session}.go` |
| `auth/provider/{berantai,lokal,tiruan}.go` | `auth/provider/{chain,local,fake}.go` |
| `auth/repo/sqlstore/warisan.{go,sql}` | `auth/repo/sqlstore/legacy.{go,sql}` |
| `cmd/claimpnc/periksa.go` | `cmd/claimpnc/check.go` |
| `migrations/0001_pengguna_dan_sesi.*` | `migrations/0001_user_and_session.*` |
| `migrations/0002_master_status_klaim.*` | `migrations/0002_master_claim_status.*` |

### Frontend

| Sebelum | Sesudah |
|---|---|
| `src/api/{klien,tipe}.ts` | `src/api/{client,types}.ts` |
| `src/app/KerangkaHalaman.tsx` | `src/app/PageShell.tsx` |
| `src/app/{PenjagaSesi,PeringatanSesi}.tsx` | `src/app/{SessionGuard,SessionWarning}.tsx` |
| `src/app/sesi.ts` | `src/app/session.ts` |
| `src/components/{Ikon,KolomIsian,PesanGalat,TabelData,Tombol}.tsx` | `src/components/{Icon,Field,ErrorMessage,DataTable,Button}.tsx` |
| `src/gaya.css` | `src/styles.css` |
| `src/modules/beranda/HalamanBeranda.tsx` | `src/modules/home/HomePage.tsx` |
| `src/modules/masuk/HalamanMasuk.tsx` | `src/modules/login/LoginPage.tsx` |
| `src/modules/master-rekening/**` | `src/modules/master-rekening/**` — nama modul dikembalikan (`D-81`) |
| `src/modules/master-status-klaim/**` | `src/modules/master-status-klaim/**` — nama modul dikembalikan (`D-81`) |
| `src/modules/portal/PemilihPortal.tsx` | `src/modules/portal/PortalPicker.tsx` |
| `src/uji/setup.ts` | `src/test/setup.ts` |

### Nama query di berkas `.sql`

Penanda `-- name:` adalah **identifier yang dipanggil kode Go**, sehingga ikut berbahasa Inggris.
**Isi SQL-nya tidak disentuh** — nama tabel dan kolom tetap milik basis data.

| Sebelum | Sesudah |
|---|---|
| `pengguna_ambil_by_identitas` | `user_get_by_identity` |
| `sesi_{sisip,cabut,perpanjang,periksa_tabel}` | `session_{insert,revoke,extend,check_table}` |
| `login_lokal_{cari_aktif,periksa_tabel}` | `local_login_{find_active,check_table}` |
| `layanan_alamat` | `service_address` |
| `portal_daftar` | `portal_list` |
| `status_klaim_*` | `claim_status_*` |
| `rekening_*` · `bank_daftar` | `account_*` · `bank_list` |

### Perintah npm

| Sebelum | Sesudah |
|---|---|
| `npm run periksa-tipe` | `npm run typecheck` |
| `npm run tandai-dist` | `npm run mark-dist` |

## Nama yang sengaja TIDAK diterjemahkan

| Nama | Alasan |
|---|---|
| `NIK` | singkatan resmi, bukan kata |
| `HCQ`, `SPA`, `DTO`, `SQL`, `SMTP` | singkatan |
| `Kasir` di dalam **komentar** | nama sistem eksternal sebagaimana disebut bisnis; seam Go-nya tetap `Cashier` |
| Nilai kolom `"Ya"`, `"Tidak"`, `"Aktif"` | **isi data**, bukan nama. Mengubahnya mengubah arti baris di basis data |
| Kode galat API (`isian_tidak_sah`, `sesi_kedaluwarsa`, …) | kontrak API |
| Variabel lingkungan (`PENYIMPANAN`, `PORTAL_UTAMA`, `IDENTITAS_ADAPTER`, `POOLDATA_*_PENGGUNA`) | dipakai berkas `.env` dan skrip deployment — menggantinya merusak lingkungan yang berjalan |
| Flag baris perintah (`-periksa`, `-login`) | idem; ia antarmuka operator, bukan nama internal. Fungsi di baliknya tetap `check()` di `cmd/claimpnc/check.go` |

---

## Nama modul — pengecualian yang berlawanan arah (`D-81`)

Ditetapkan Work Owner 2026-09-18, **sesudah** penggantian nama dijalankan: **nama folder modul
memakai nama modul bisnis dalam bahasa Indonesia**, bukan padanan Inggrisnya.

| Lapisan | Bentuk | Contoh |
|---|---|---|
| Folder backend & paket Go | `namamodul` — tanpa tanda hubung | `internal/masterrekening` · `internal/masterstatus` |
| Folder frontend | `nama-modul` — `kebab-case` | `src/modules/master-rekening` · `src/modules/master-status-klaim` |

**Isi modulnya tetap Inggris.** Yang Indonesia hanya nama modulnya:
`internal/masterrekening/repo/sqlstore/account.go` — kiri nama modul, kanan isi modul.

**Nama modul berikutnya disebutkan Work Owner di prompt.** Jangan menerjemahkan dan jangan
mengarang: "Master Rekening" → `master-rekening`, "Input Receive Document" → `input-receive-document`.
Modul kerangka tanpa nama bisnis (`auth`, `portal`, `platform`, `spa`) tidak berubah.

**Komponen di dalam modul memakai nama tipe domain, bukan nama modul** — karena itu
`AccountPage.tsx` di dalam `master-rekening/`, bukan `MasterRekeningPage.tsx`.

---

## Tambahan 2026-09-18 — modul Master Status Progres & prop komponen bersama

### Istilah domain baru

| Indonesia | Inggris | Catatan |
|---|---|---|
| StatusProgres | ProgressStatus | keterangan progres pada satu posisi klaim |
| Posisi (klaim) | Position | Register · Survey · Komite · Akseptasi |
| Induk | Parent | Status Progres 1 yang menaungi baris tingkat 2 |
| **Isian** (masukan domain Go) | **Input** | dibedakan dari `Values` yang dipakai untuk objek nilai form di React — lihat `keputusan-implementasi.md` §15.4 |
| PelanggaranIsian | Violation | sama dengan modul masterstatus |
| PemilihRepo | RepoSelector | memilih repo milik satu portal entitas |

### Kata kerja tambahan

| Indonesia | Inggris | Catatan |
|---|---|---|
| SisipBaru | InsertNew | lapisan repo — menurunkan ID lalu menyisip dalam satu operasi |
| PastikanPortalSiap | EnsurePortalReady | |
| PilihAktif | SelectActive | modul portal |

### Prop komponen bersama — sisa yang dibereskan 2026-09-18

Penggantian nama 2026-09-18 pagi menyisakan nama prop berbahasa Indonesia pada pustaka komponen.
Seluruhnya kini Inggris:

| Sebelum | Sesudah | Komponen |
|---|---|---|
| `judul` | `title` | `DataTable`, `ErrorMessage`, `Column` |
| `keterangan` | `description` | `DataTable`, `ErrorMessage` |
| `nilai` · `tampil` | `value` · `render` | `Column` |
| `lebar` · `tanpaUrut` · `keKanan` | `width` · `noSort` · `alignRight` | `Column` |
| `aksi` | `actions` | `DataTable`, `AccountTable` |
| `petunjuk` | `hint` | `Field` |
| `kotak` | `box` | `ErrorMessage` |
| `arah` · `'naik'`/`'turun'` | `direction` · `'asc'`/`'desc'` | `DataTable` |
| `anak` | `children` | `PageShell`, `SessionGuard`, `App` |

**`aktif` sengaja TIDAK ikut diganti.** Ia nama field JSON API (`Account.aktif`), bukan nama
internal — kecuali satu prop lokal pada `SortMarker` di `DataTable`, yang memang bukan kontrak.

### Nama kueri `.sql` tambahan

| Sebelum | Sesudah |
|---|---|
| `statusprogres_{daftar,ambil,sisip,perbarui,periksa_tabel}` | `progress_status_{list,get,insert,update,check_table}` |
| `statusprogres_daftar_id_terkunci` | `progress_status_list_id_locked` |
| `statusprogres2_*` | `progress_status2_*` |

### Nama modul yang sudah ditetapkan

| Nama modul bisnis (Work Owner) | Folder backend / paket Go | Folder frontend |
|---|---|---|
| Master Rekening | `internal/masterrekening` | `src/modules/master-rekening` |
| Master Status Klaim | `internal/masterstatus` | `src/modules/master-status-klaim` |
| Master Status Progres 1 | `internal/masterstatusprogres` | `src/modules/master-status-progres` |
| Master Tipe Surveyors | `internal/mastertipesurveyors` | `src/modules/master-tipe-surveyors` |

**Komponen di dalamnya memakai nama tipe domain, bukan nama modul** — karena itu
`ProgressStatusPage.tsx` di dalam `master-status-progres/`, bukan `MasterStatusProgresPage.tsx`.

---

## Tambahan 2026-09-19 — modul menu

Modul `internal/menu` adalah **modul kerangka**, bukan layar Master yang diminta dengan nama bisnis.
Namanya karena itu Inggris, sejajar dengan `auth`, `portal`, dan `platform` — `D-81` hanya berlaku
untuk modul yang Work Owner sebut dengan nama bisnisnya.

| Kolom / istilah basis data | Inggris di kode |
|---|---|
| `MENU_ID` · `MENU_DESC` | `ID` · `Description` |
| `MENU_PROGRAM` | `Program` — nama harness, dikirim apa adanya ke layar |
| `MENU_ID_LEADER` | `ParentID` — nil berarti kelompok tingkat atas |
| `MENU_SEQUENCE` | `Sequence` |
| `LOGIN_ID_GROUP` | **`Subject`** — satu kolom yang menampung login MAUPUN group; `Subjects()` menyusun daftarnya |
| `GROUP_ID` | `Group` |
| butir menu beserta anaknya | `Node` |

| Indonesia | Inggris | Catatan |
|---|---|---|
| Otorisasi | Authorization | `AuthorizedIDs` mengembalikan MENU_ID yang diizinkan |
| Kelompok menu | Group | bukan `Category`: sumbernya memang baris menu yang tidak berinduk |
| Susun pohon | BuildTree | |

**Nama field JSON tetap Indonesia** (`id`, `nama`, `program`, `submenu`) — ia kontrak API.

**Nama kueri `.sql`** berawalan `menu_`, mengikuti nama tabelnya dan bukan nama modul:
`menu_list` · `menu_app_exists` · `menu_groups_of_login` · `menu_authorized_ids` · `menu_check_table`.

---

## Tambahan 2026-09-19 — modul Master Tipe Surveyors

Modul ini **dinamai dengan nama bisnisnya** (`D-81`), dan namanya tidak dikarang: ia tertulis di
`Database/m_menu_aplikasi_pnc.csv` sebagai `MENU_ID 14` — **"Master Tipe Surveyors"**.

### Istilah domain

| Kolom basis data | Inggris di kode | Catatan |
|---|---|---|
| `M_SURVEY_ID` | `Code` | `CHAR(4)`; dipatok langsung di tiga kueri Pega, karena itu tidak pernah berubah |
| `DESCRIPTION` | `Description` | nama tipe yang dibaca petugas |
| `OLD_M_SURVEY_ID` | `LegacyCode` | jejak penomoran sistem sebelumnya; kosong pada seluruh baris |

| Indonesia | Inggris | Catatan |
|---|---|---|
| Tipe surveyor | `SurveyorType` | **golongan** petugas survei, bukan orangnya |
| Surveyor | — | daftar ORANGNYA ada di `D_SURVEYORS`, modul tersendiri yang belum dibangun |
| Pemilih repo per portal | `RepoSelector` | sama dengan `masterstatusprogres` |
| Kunci keunikan | `DescriptionKey` | `UPPER(TRIM(...))`, sepadan dengan indeks migrasi `0003` |

**Istilah yang sengaja TIDAK dipakai:** `Name`. Kolomnya bernama `DESCRIPTION` dan nilainya memang
deskripsi golongan ("INTERNAL SURVEYOR"), bukan nama seseorang. Memakai `Name` akan membuatnya
tertukar dengan `D_SURVEYORS.NAME`, yang justru nama orang.

### Nama berkas

| Berkas | Nama |
|---|---|
| Domain | `mastertipesurveyors.go` — mengikuti nama paket, sama seperti `masterstatus.go` |
| Layar | `SurveyorTypePage.tsx` · `SurveyorTypeForm.tsx` — **nama tipe domain, bukan nama modul** |
| Uji layar | `SurveyorTypePage.test.tsx` |

### Nama kueri `.sql`

Berawalan `surveyor_type_`, mengikuti **konsep domainnya** dan bukan nama tabelnya:

`surveyor_type_list` · `surveyor_type_get` · `surveyor_type_site` ·
`surveyor_type_next_sequence` · `surveyor_type_insert` · `surveyor_type_update` ·
`surveyor_type_check_table`

### Nama field JSON — tetap Indonesia

`kode` · `deskripsi` · `kode_lama` · `tipe_surveyor` · `total` · `portal`

Ia kontrak API, bukan nama internal (`D-80`).

### Kode galat — tetap Indonesia

`tipe_surveyor_tidak_ditemukan` · `tipe_surveyor_sudah_ada` ·
`kode_tipe_surveyor_sudah_dipakai` · `kode_tidak_dapat_dibentuk` · `validasi_gagal` ·
`permintaan_cacat`

### Jalur — tunggal, walau nama modulnya jamak

| Hal | Bentuk |
|---|---|
| Rute API | `/api/master/tipe-surveyor` |
| Rute layar | `/master/tipe-surveyor` |

Rute master yang sudah ada seluruhnya tunggal (`status-klaim`, `rekening`, `status-progres-1`).
Menyeragamkan **jalur** lebih berguna daripada menyeragamkannya dengan nama modul: jalur adalah
kontrak, nama modul adalah nama internal.

---

## Tambahan 2026-09-19 — modul Master PIC Teknik

Modul ini **sudah ada** sebelum `D-80` dan ditulis seluruhnya dalam bahasa Indonesia. Seluruh
berkasnya dihapus dan ditulis ulang; folder modulnya tetap Indonesia sesuai `D-81`.

### Berkas

| Sebelum | Sesudah |
|---|---|
| `internal/masterpicteknik/picteknik.go` | `internal/masterpicteknik/masterpicteknik.go` |
| `internal/masterpicteknik/usecase/kelola.go` | `internal/masterpicteknik/usecase/manage.go` |
| `internal/masterpicteknik/http/galat.go` | `internal/masterpicteknik/http/errors.go` |
| `internal/masterpicteknik/http/rute.go` | `internal/masterpicteknik/http/routes.go` |
| `internal/masterpicteknik/repo/memori/` | `internal/masterpicteknik/repo/memory/` |
| `internal/masterpicteknik/repo/memori/contoh.go` | `internal/masterpicteknik/repo/memory/sample.go` |
| `internal/masterpicteknik/repo/sqlstore/kueri.go` | `internal/masterpicteknik/repo/sqlstore/query.go` |
| `internal/masterpicteknik/repo/sqlstore/picteknik.go` | `internal/masterpicteknik/repo/sqlstore/technician.go` |
| `internal/masterpicteknik/repo/sqlstore/picteknik.sql` | `internal/masterpicteknik/repo/sqlstore/technician.sql` |
| — (baru) | `internal/masterpicteknik/directory/{hcq,fake}.go` |
| — (baru) | `frontend/src/modules/master-pic-teknik/` |

**Nama folder modul TIDAK diterjemahkan** (`D-81`): `masterpicteknik` di backend,
`master-pic-teknik` di frontend. "PIC Teknik" adalah nama yang dipakai Work Owner dan yang tertulis
di butir menu `MENU_ID 13`.

### Tipe dan fungsi

| Sebelum | Sesudah |
|---|---|
| `PICTeknik` | `Technician` |
| `Bersih()` | `Clean()` |
| `KunciID()` | `IDKey()` |
| `Periksa()` | `Check()` |
| `EmailMasukAkal()` | `EmailPlausible()` |
| `Pelanggaran` | `Violation` |
| `GalatValidasi` · `GalatValidasiBaru()` | `ValidationError` · `NewValidationError()` |
| `DirektoriOperator` | `EmployeeDirectory` |
| `Layanan` · `LayananBaru()` · `Opsi` | `Service` · `NewService()` · `Options` |
| `HandlerBaru()` | `NewHandler()` |
| `Pasang()` | `Mount()` |
| `RepoBaru()` | `NewRepo()` |
| `ambilKueri()` · `muatSeluruhKueri()` · `pecahPerNama()` | `getQuery()` · `loadAllQueries()` · `splitByName()` |
| `DaftarContoh()` | `SampleList()` |
| `TulisGalat()` · `petakanGalat()` | `ErrorWriter` · `mapError()` |
| `PenulisJSON` · `PenulisGalat` | `JSONWriter` · `ErrorWriter` |

### Metode seam dan usecase

| Sebelum | Sesudah |
|---|---|
| `Repo.Daftar` · `Ambil` · `Sisip` · `Perbarui` | `Repo.List` · `Get` · `Insert` · `Update` |
| `Layanan.Daftar` · `Ambil` · `Tambah` · `Ubah` | `Service.List` · `Get` · `Create` · `Update` |
| `DirektoriOperator.NamaOperator` | `EmployeeDirectory.Lookup` |
| — (baru) | `Service.Lookup` · `Service.EnsurePortalReady` |

### Field domain — dan asal kolomnya

Nama field mengikuti `D-19`: **bukan** nama kolomnya, dan **bukan** alias Pega yang menyesatkan.

| Kolom | Alias Pega lama | Field domain |
|---|---|---|
| `OPERATOR_ID` | `MCL_Name` pada `BrowseEmailUserTeknis` | `OperatorID` |
| `MCL_NAME` | `MCL_NAME` | `Name` |
| `TYPE_BUSINESS` | `TYPE_BUSINESS` | `BusinessLine` |
| `TEAM_GROUP` | `TEAM_GROUP` | `Group` |
| `ATASAN` | `ATASAN` | `Supervisor` |
| `COUNTER_QUOTA` | `COUNTER_QUOTA` | `Quota` |
| `COUNTER_QUOTA2` | **`OLD_OPERATOR_ID`** | `ExternalQuota` |
| `TOTAL_JOB` | `TOTAL_JOB` | `Workload` |
| `GROUPPANEL` | **`IBNR`** | `PanelGroup` |
| `STS_AKTIF` | `STS_AKTIF` | `Active` |

Dua alias yang dicetak tebal adalah alasan tabel ini ada: `OLD_OPERATOR_ID` menyebut sebuah
**angka** seolah identitas, dan `IBNR` menyebut **nama grup panel** dengan istilah akuntansi
asuransi yang tidak berhubungan.

### Nama field JSON — tetap Indonesia

Ia **kontrak**, bukan nama internal (`D-80`):

```go
type TechnicianDTO struct {
	OperatorID    string `json:"id_operator"`
	Name          string `json:"nama"`
	BusinessLine  string `json:"lini_bisnis"`
	ExternalQuota int    `json:"kuota_luar"`
	Workload      int    `json:"beban_kerja"`
	PanelGroup    string `json:"grup_panel"`
	Active        bool   `json:"aktif"`
}
```

Demikian pula **jalur rute** (`/api/master/pic-teknik`, `/direktori/{id}`) dan **kode galat**
(`pic_teknik_tidak_ditemukan`, `direktori_pegawai_belum_terdaftar`).

### Nama komponen frontend

Komponen memakai nama **tipe domain**, bukan nama modul — mengikuti preseden `AccountPage.tsx` pada
modul `master-rekening`:

| Berkas | Komponen |
|---|---|
| `master-pic-teknik/TechnicianPage.tsx` | `TechnicianPage` |
| `master-pic-teknik/TechnicianForm.tsx` | `TechnicianForm` |
| `master-pic-teknik/api.ts` | `useTechnicianList`, `useTechnician`, `useLookupEmployee`, `useSaveTechnician` |

### Nama uji

Berbahasa **Inggris** di Go, **Indonesia** di Vitest — mengikuti pola yang sudah berlaku:

```go
func TestGetReachesInactiveTechnician(t *testing.T)
func TestCreateTakesNameFromDirectoryNotFromInput(t *testing.T)
```

```ts
it('mengisi nama dan atasan dari hasil pencarian direktori', ...)
it('menjelaskan bahwa petugas nonaktif tidak ditampilkan', ...)
```

---

## Tambahan 2026-09-20 — modul Master Surveyors

Modul `internal/mastersurveyors` / `src/modules/master-surveyors`, atas tabel
`POOLDATA.D_SURVEYORS`.

**Nama modulnya Indonesia, isinya Inggris** (`D-80`, `D-81`): folder backend
`internal/mastersurveyors` tanpa tanda hubung, folder frontend `src/modules/master-surveyors`
dengan tanda hubung.

### Dua modul yang sangat mudah tertukar

Ini bagian terpenting di sini. Keduanya bernama mirip, keduanya tentang surveyor, dan keduanya
hidup berdampingan:

| | Master **Tipe** Surveyors | Master Surveyors |
|---|---|---|
| Isi | GOLONGAN petugas survei | ORANG dan lembaganya |
| Tabel | `POOLDATA.M_SURVEYORS` | `POOLDATA.D_SURVEYORS` |
| Butir menu | `MENU_ID 14` | `MENU_ID 15` |
| `MENU_PROGRAM` | `SurveyorsInbox` | `DetailSurveyorsInbox` |
| Paket Go | `mastertipesurveyors` | `mastersurveyors` |
| Rute | `/master/tipe-surveyor` | `/master/surveyor` |
| Urutan kode | `M_SURVEYORS_SEQ`, **3 digit** | `D_SURVEYORS_SEQ`, **6 digit** |
| Persetujuan komite | tidak ada | **ada** |

Yang membedakan nama programnya hanyalah awalan `Detail`. Urutan kodenya berbeda **dan jumlah
digitnya berbeda** — tertukar berarti kode yang terbit salah panjang dan bertabrakan dengan deret
milik master lain. Ada uji yang menjaganya (`TestSelectorUrutanSequenceBenar`).

### Kolom → domain → kontrak API

| Kolom `D_SURVEYORS` | Field domain (Inggris) | Field JSON (Indonesia) |
|---|---|---|
| `D_SURVEY_ID` | `ID` | `id` |
| `OLD_D_SURVEY_ID` | `LegacyID` | — (tidak dikirim) |
| `M_SURVEY_ID` | `TypeCode` | `kode_tipe` |
| `M_SURVEYORS.DESCRIPTION` | `TypeDescription` | `nama_tipe` |
| `NAME` | `Name` | `nama` |
| `ADDRESS` | `Address` | `alamat` |
| `KDPOS` | `PostalCode` | `kode_pos` |
| `STATE` | `State` | `provinsi` |
| `TELEPHONE` | `Phone` | `telepon` |
| `FAKSIMILE` | `Fax` | `faksimile` |
| `EMAIL` | `Email` | `email` |
| `OTHER_CONTACT` | `OtherContact` | `kontak_lain` |
| `BRANCH` | `BranchCode` | `kode_cabang` |
| `BRANCHNAME` | `BranchName` | `nama_cabang` |
| `LOGIN_APLIKASI` | `AppLogin` | `login_aplikasi` |
| `DOCID` | `DocumentID` | `id_dokumen` |
| `APPROVAL` | `Status` | `status` |
| `KOMITE` | `Committee` | `komite` |
| `TRFKOMITE` | `CommitteeTransferred` | — (tidak dikirim) |

Lima kolom yang **ditambahkan migrasi 0004** — namanya Indonesia, mengikuti `D-80` yang
menetapkan nama kolom basis data tetap Indonesia:

| Kolom baru | Field domain | Field JSON |
|---|---|---|
| `TGL_APPROVE_KOMITE` | `DecidedAt` | `tanggal_keputusan` |
| `CATATAN_KOMITE` | `Note` | `catatan` |
| `USER_INPUT` | `CreatedBy` | — |
| `TGL_INPUT` | `CreatedAt` | — |
| `USER_UPDATE` | `UpdatedBy` | — |

Satu tabel karena itu memuat **dua gaya penamaan kolom** — `NAME`, `ADDRESS`, `BRANCH` yang
Inggris, dan kelima yang baru yang Indonesia. Itu disadari dan **tidak diseragamkan**: mengganti
nama kolom lama akan memutus rule Pega yang masih membacanya.

### Alias Pega yang TIDAK dibawa

| Alias lama | Isi sebenarnya | Di sini |
|---|---|---|
| `BUSINESS_CODE` (pada `EmailKomiteAdjuster_sql`) | **`OPERATOR_ID`** | `Committee` / `komite` |
| `BRANCH_CODE` (idem) | `EMAIL` | tidak dipakai |
| `BRANCH_NAME` (idem) | `DEGREE` | tidak dipakai |

Ketiga alias itu salah arti sekaligus di satu kueri — contoh paling padat dari utang teknis yang
`docs/Steering/03-CURRENT-ARCHITECTURE.md` §4.2 catat.

### Nama yang sengaja dipilih

| Nama | Alasan |
|---|---|
| `Surveyor`, bukan `DetailSurveyor` | "Detail" di nama Pega menandai ia tabel anak, bukan sifat barisnya. Barisnya adalah seorang surveyor |
| `AppLogin`, bukan `Login` | `Login` sendirian tertukar dengan sesi aplikasi. Ini nama pengguna milik surveyor |
| `AccountRegistrar`, bukan `OperatorCreator` | "Operator" adalah istilah Pega. Yang dibuat adalah akun aplikasi |
| `CommitteeTransferred`, bukan `TrfKomite` | Singkatan `TRF` tidak berarti apa pun di luar tabelnya |
| `DecidedAt`, bukan `ApprovedAt` | Ia terisi pada penolakan juga, bukan hanya persetujuan |

---

## Tambahan 2026-09-20 — modul Master Recovery

Nama modul mengikuti `D-81`: folder backend `internal/masterrecovery` (tanpa tanda hubung,
karena Go tidak mengizinkannya), folder frontend `src/modules/master-recovery`. Isinya
berbahasa Inggris sesuai `D-80`.

### Tiga properti Pega yang isinya BERBEDA dari namanya

Ini yang paling menyesatkan di modul ini, dan ketiganya terbukti di
`RDB List/InsertMasterRecoveryKlaimASM-SQL.xml` — bukan disimpulkan:

| Properti Pega | Kolom tujuan | Isi SEBENARNYA | Di sini |
|---|---|---|---|
| `TempRecovery.NoKTPMasking` | `POSISIKASUS` | keterangan **posisi perkara** — bukan NIK tersamar | `CasePosition` / `posisi_kasus` |
| `TempRecovery.TPLAmount` | `SISAKLAIM` | **sisa klaim** — bukan nilai tanggung jawab pihak ketiga | `Remainder` / `sisa` |
| `TempRecovery.Idlogservice` | `NOHPLL` | **nomor catatan log layanan** — bukan nomor telepon, meski kolomnya bernama demikian | `ServiceLogID` / `id_log_layanan` |

Yang ketiga patut diperhatikan khusus: **nama KOLOMNYA pun menyesatkan**, bukan hanya nama
propertinya. Kolom itu tidak diganti nama — ia dibagi dengan Pega selama masa paralel
(`D-21`) — sehingga yang dikoreksi adalah nama di dalam kode, dan kekeliruannya berhenti
di lapisan `repo/sqlstore`.

### Kolom → domain → kontrak API

| Kolom Oracle | Field domain (Inggris) | Field JSON (Indonesia) |
|---|---|---|
| `BATCH` | `Batch` | `nomor_batch` |
| `NAMAPRINCIPAL` | `PrincipalName` | `nama_principal` |
| `CLIENTID` | `ClientID` | `client_id` |
| `NOVA` | `VirtualAccountNumber` | `nomor_virtual_account` |
| `TAHUN` | `Year` | `tahun` |
| `NILAIKLAIM` | `ClaimAmount` | `nilai_klaim` |
| `NILAIRECOVERY` | `PreviousPayment` | `pembayaran_sebelumnya` |
| `PEMBAYARAN` | `Payment` | `pembayaran` |
| `SISAKLAIM` | `Remainder` | `sisa` |
| `KETERANGAN` | `Remark` | `keterangan` |
| `POSISIKASUS` | `CasePosition` | `posisi_kasus` |
| `DOKUMENID` | `DocumentID` | `id_dokumen` |
| `USERNAME` | `InputBy` | `dicatat_oleh` |
| `NOHPLL` | `ServiceLogID` | `id_log_layanan` |
| `NOPOLIS` | `PolicyNo` | `nomor_polis` |
| `LBU_ID` | `BusinessID` | `id_lini_bisnis` |
| `LDC_ID` | `BranchID` | `id_cabang` |
| `LAG_AGEN_ID` | `AgentID` | `id_agen` |
| `LMO_ID` | `MarketingID` | `id_marketing` |
| `JSON_POLIS` | `ClaimLine` | `baris_klaim` |

Dari `MST_VIRTUAL_ACCOUNT_PNC`:

| Kolom Oracle | Field domain | Field JSON |
|---|---|---|
| `CLIENTNAME` | `Principal.Name` | `nama_principal` |
| `VIRTUALACCOUNTNUMBER` | `Principal.VirtualAccountNumber` | `nomor_virtual_account` |
| `EMAILVA` | `Principal.Email` | `email_inputor_va` |

### Nama yang sengaja dipilih

| Nama | Alasan |
|---|---|
| `PreviousPayment`, bukan `Deductible` | Kolomnya bernama `NILAIRECOVERY` dan propertinya `NilaiDeductible`; **labelnya di layar** berbunyi "Nilai Pembayaran Sebelumnya", dan itulah perannya dalam rumus Sisa. Ketiga namanya berbeda — yang dipakai adalah yang menjelaskan isinya |
| `Remainder`, bukan `Balance` | "Balance" menyiratkan saldo yang berjalan. Ini hasil satu pengurangan pada satu batch |
| `ClaimLine`, bukan `ObjectList` | `Object` bertabrakan dengan makna pemrograman, dan `D-19` sudah menetapkannya ditinggalkan |
| `VirtualAccountIssuer`, bukan `VAService` | Seam dinamai menurut **perannya** — yang menerbitkan — bukan menurut singkatan sistemnya |
| `PaymentProof` (frontend), bukan `document` | `document` menutupi objek global peramban; satu berkas sempat memakainya dan harus menulis `window.document` untuk hal yang sepele |
| `byteOrderMark` | Tiga bita tak terlihat yang Excel tulis di awal CSV. Diberi nama karena ia **tidak dapat dibaca** bila ditulis sebagai hurufnya sendiri |

### Nama uji

Nama uji menyebutkan **aturannya**, bukan nama fungsinya, sehingga daftar uji terbaca
sebagai daftar aturan bisnis yang selalu mutakhir — misalnya
`TestSisaMengabaikanPembayaranKetikaAdaPembayaranSebelumnya` dan
`TestSisaCocokDenganKetigaBarisProduksi`.

---

## Tambahan 2026-09-20 — modul Master Dominan Factor

Nama modul mengikuti `D-81`: `masterdominanfactor` (Go, tanpa tanda hubung) dan
`master-dominan-factor` (frontend, `kebab-case`). Ruas URL-nya `master/dominan-factor` —
nama modul yang dipakai Work Owner dan yang tertulis di butir MENU_ID 18, **bukan**
terjemahan "faktor-dominan".

### Dari `POOLDATA.M_DOMINAN_FACTOR`

Tabelnya hanya dua kolom. Yang menarik justru pada kolom kedua: ia punya **empat nama**, dan
keempatnya benar karena peruntukannya berbeda.

| Kolom Oracle | Field domain | Field JSON | Label di layar |
|---|---|---|---|
| `ID` | `DominantFactor.ID` | `id` | **ID** |
| `NAME` | `DominantFactor.Name` | `nama` | **Keterangan** |

| Lapisan | Aturan yang berlaku |
|---|---|
| Field domain Go | penamaan kode berbahasa Inggris (`D-80`) |
| Field JSON | **kontrak** — mencerminkan isi kolomnya, bukan label layarnya (`D-80`) |
| Label di layar | teks yang dilihat pengguna mengikuti layar Pega **apa adanya** (`D-13`) |

Layar Pega melabelinya "Keterangan" (`.KeteranganPerubahan` pada
`Section/DetailDominanFactor_Sec-Section.xml`), sehingga label itulah yang dipakai. Memakai
"Nama" di layar akan melanggar `D-13`; memakai `keterangan` sebagai field JSON akan membuat
kontraknya tidak mencerminkan kolom yang diwakilinya.

### Nama properti Pega yang TIDAK dibawa

| Properti Pega | Isinya | Diganti |
|---|---|---|
| `TempFactor.DistrictID` | **keterangan faktor** — sama sekali bukan ID kecamatan | `Name` |
| `TempFactor.BranchID` | **ID faktor dominan** — sama sekali bukan ID cabang | `ID` |
| `TempFactor.IDMasterTONP` | ID yang sama, pada jalur lain | `ID` |
| `.KeteranganPerubahan` | keterangan faktor; tidak ada "perubahan" apa pun yang diwakilinya | `Name` |
| `Ketarangan` (`pyName`) | salah ketik dari "Keterangan", dipertahankan bertahun-tahun di rule | — |

Ini contoh keempat di repo ini setelah `NoKTPMasking`, `TPLAmount`, dan alias kolom SQL warisan:
**nama properti di sistem lama tidak dapat dipercaya sebagai keterangan isinya.** Di modul ini
dua di antaranya bahkan menyesatkan ke domain yang sama sekali lain — wilayah dan cabang.

### Nama di dalam procedure yang juga tidak dibawa

| Nama di `PEGA_M_DOMINAN_FACTOR.prc` | Kenyataannya |
|---|---|
| `id_site` (`:3`) | menampung hasil **`max(ID)+1`**, bukan kode situs. Jejak salin-tempel dari procedure master lain yang memang memakai kode situs |
| `id_dominan_factor` (`:4`) | dideklarasikan, **tidak pernah dipakai** |

Keduanya dicatat karena menyesatkan ke arah yang mahal: mengira ada kode situs akan membuat ID
diterbitkan berbentuk `1011` alih-alih `11`.

### Nama yang sengaja dipilih

| Nama | Alasan |
|---|---|
| `DominantFactor`, bukan `DominanFactor` | Tipe adalah **kode**, jadi berbahasa Inggris penuh (`D-80`). Yang tetap "Dominan" hanyalah **nama modulnya** — `masterdominanfactor` — karena itu nama yang dipakai Work Owner (`D-81`) |
| `NextID`, bukan `GenerateID` | Ia tidak membangkitkan apa pun; ia membaca nomor tertinggi lalu menambah satu. Namanya menyebutkan itu |
| `NumericID`, bukan `ParseID` | `Parse` menyiratkan galat sebagai hasil. Yang dikembalikan adalah "ini bilangan atau bukan" — dan bukan-bilangan adalah keadaan yang **sah** di sini |
| `SortByID`, bukan `Sort` | Menyebutkan menurut apa diurutkan, karena urutannya **numerik meski ID-nya teks** — hal yang justru perlu disebut |
| `lockExistingIDs`, bukan `getIDs` | Namanya menyebutkan akibat sampingnya. Fungsi ini **mengunci baris**, dan pemanggil yang tidak tahu itu dapat memakainya di luar transaksi |
| `ErrIDTaken`, bukan `ErrDuplicate` | Menyebutkan **apa** yang bentrok. Di modul ini nama ganda justru sah, sehingga "duplicate" akan terbaca salah |

### Nama uji

Nama uji menyebutkan **aturannya**, bukan nama fungsinya — dan di modul ini sebagian menyebutkan
aturan yang **sengaja tidak ada**, supaya ketiadaannya tidak terbaca sebagai kelalaian:
`TestEmptyNameAccepted`, `TestCreateAcceptsEmptyAndDuplicateNames`,
`TestSortByIDIsNumericNotTextual`, `TestNextIDMimicsProcedure`, `TestNoDeleteRoute`.

## Tambahan 2026-09-20 — modul Master Masking

Nama modul mengikuti butir menu `MENU_ID 17` "Master Masking" (`D-81`), **bukan** nama
harness-nya (`MasterProteksiVisibilityData`) yang panjang dan tidak menyebut "masking"
sama sekali. Yang dipakai adalah nama yang disebut Work Owner dan yang tertulis di menu.

| Lapisan | Bentuk |
|---|---|
| Folder & paket Go | `internal/mastermasking` |
| Folder frontend | `src/modules/master-masking` |
| Rute API | `/api/master/masking` |
| Rute layar | `/master/masking` |
| Kunci `MENU_ROUTES` | `MasterProteksiVisibilityData` (harus sama persis dengan `MENU_PROGRAM`) |

### Kolom basis data ke nama domain

Kolom tabel `POOLDATA.MST_PROTEKSI_DATA_PNC` dinamai ulang mengikuti `D-19` dan `D-80`.
Aliasnya di sistem lama menyesatkan — lihat kolom terakhir.

| Kolom | Nama domain (Go) | Field JSON | Label layar | Alias Pega yang menyesatkan |
|---|---|---|---|---|
| `ID_MST` | `ID` | `id` | — | — |
| `CABANG` | `BranchID` | `cabang` | Cabang | `InputData.ReportAddress` |
| — (hasil JOIN) | `BranchName` | `nama_cabang` | Cabang | — |
| `LOGIN` | `Login` | `login` | Nama User | `InputData.UserName` |
| `MODUL` | `Module` | `modul` | Modul | `InputData.City` |
| `SUBMODUL` | `SubModule` | `sub_modul` | Sub Modul | `InputData.CaseID` |
| `LOGSEARCH` | `SearchQuota` | `maks_cari` | Max Cari Data | `InputData.Amount` |
| `LOGSEEN` | `ViewQuota` | `maks_lihat` | Max Lihat Data | `InputData.ClaimAmount` |
| `STS_KTP` | `ViewIDCard` | `lihat_ktp` | View KTP | `InputData.ResponseNote` |
| `STS_EMAIL` | `ViewEmail` | `lihat_email` | View Email | `InputData.ResponseMsg` |
| `STS_NOTELP` | `ViewPhone` | `lihat_notelp` | View NoTelp | `InputData.ResponseNoteDocKasir` |
| `STS_AKTF` | `Active` | `aktif` | Status Aktif | `InputData.BranchID` |
| `USERINPUT` | `InputBy` | `dicatat_oleh` | Inputor | `InputData.RemarkRecommendation` |
| `TANGGALINPUT` | `InputAt` | `dicatat_pada` | — | — |
| `PASSWORD` | — **tidak dimodelkan** | — | — | `InputData.ResponseCode` |

Perhatikan baris `STS_AKTF`: aliasnya di Pega adalah `InputData.BranchID` — nama yang
justru menunjuk **kolom lain** di tabel yang sama. Inilah contoh paling tajam dari utang
teknis §4.2, dan alasan `D-19` menuntut penamaan ulang menyeluruh.

`PASSWORD` sengaja tidak punya nama domain: ia ditulis dari konstanta `legacyPassword` di
`repo/sqlstore` dan tidak pernah dibaca kembali.

### Nilai tersimpan ke tipe domain

| Kolom | Nilai di basis data | Tipe domain |
|---|---|---|
| `STS_KTP` / `STS_EMAIL` / `STS_NOTELP` | `'Ya'` / `'Tidak'` | `bool` |
| `STS_AKTF` | `'AKTIF'` / `'TIDAK AKTIF'` | `bool` |

Penerjemahannya ada di `repo/sqlstore` saja, lewat `isYes`, `isActiveText`, `flag`, dan
`status`. Nilai yang tidak dikenal selalu diterjemahkan ke arah **aman** — tidak boleh
melihat, tidak berlaku.

### Nama tipe Go

| Tipe | Isi |
|---|---|
| `Masking` | satu baris master |
| `Branch` | satu pilihan cabang (baca `POOLDATA.BRANCH`) |
| `Filter` | penyaring daftar |
| `SearchBy` | tipe pencarian: `""`, `"cabang"`, `"login"` |

### Nama kueri SQL

Seluruhnya berawalan `masking_`, kecuali dua yang membaca tabel milik sistem lain:

```
masking_list · masking_get · masking_find_pair · masking_next_id
masking_insert · masking_update · masking_set_active
branch_list · branch_exists
```

---

## Tambahan 2026-09-20 — modul Master Penyebab Kerugian

Modul `masterpenyebabkerugian` — `MENU_ID 20`, harness `CauseOfLossInbox`, tabel
`POOLDATA.M_CAUSE_OF_LOSS`.

Nama foldernya mengikuti `D-81`: **nama modul dalam bahasa Indonesia**
(`internal/masterpenyebabkerugian`, `src/modules/master-penyebab-kerugian`), sementara
**isinya berbahasa Inggris** sesuai `D-80`.

### Kolom basis data ke nama domain

| Kolom | Nama domain (Go) | Field JSON | Label layar |
|---|---|---|---|
| `M_COL_ID` | `ID` | `id` | ID |
| `COL_DESC` | `Description` | `deskripsi` | Deskripsi Kerugian |
| `OLD_M_COL_ID` | `LegacyID` | `id_lama` | *(tidak ditampilkan)* |

Tiga catatan:

1. **`Description` bukan `Desc`.** `Desc` adalah kata kunci SQL dan singkatan yang
   maknanya bergeser (*descending* versus *description*).
2. **Label layar "Deskripsi Kerugian" diambil apa adanya dari Pega** (`D-13`) — `pyValue`
   pada sel ber-`pyCellHeader=true` di `Section/BrowseCauseOfLoss-Section.xml`, dan
   `pyLabelPreview` pada isian formnya.

   Pembacaan pertama sempat menyimpulkan kolom itu tidak punya caption sehingga judulnya
   "jatuh ke nama properti", lalu menuliskannya **"Keterangan"**. Itu **salah**: captionnya
   ada, hanya tersimpan sebagai `pyValue` pada sel header — bukan sebagai `pyCaption` yang
   dicari pembacaan pertama.
3. **Field JSON `deskripsi` mengikuti label itu**, bukan `keterangan` dan bukan `col_desc`.
   Field JSON adalah KONTRAK yang mencerminkan isinya (`D-80`); membiarkannya berbeda dari
   label layar akan mengulang persis utang teknis §4.2 — nama yang tidak mencerminkan isinya.
4. **`id_lama` ada di kontrak tetapi tidak di layar.** Lapisan data mengikuti Report
   Definition `BrowseVMCauseOfLoss_RD` yang memuatnya; lapisan layar mengikuti section-nya,
   yang grid-nya hanya dua kolom.

### Nama properti Pega yang TIDAK dibawa

| Pega | Kenapa tidak dibawa |
|---|---|
| `TempCauseOfLoss` | halaman kerja klipboard; tidak punya padanan di arsitektur baru |
| `"UnknownID"` | penanda baris baru; diganti metode HTTP (`POST` versus `PUT`) |
| `InputData.COL_DESC` | **alias menyesatkan** — isinya bukan deskripsi melainkan SELURUH dokumen JSON hasil `@GCNM.GetPageJSONString()` |
| `OutputData.COL_DESC` | idem, tetapi isinya `ErrMsg` — pesan yang pada jalur BERHASIL pun terisi kalimat |
| `pyNote` | penampung pesan procedure di layar, berlabel "Catatan"; diganti kode galat yang dapat dibaca mesin |
| `TempCauseOfLoss.Description` | **isian mati** — berlabel sama dengan `COL_DESC` dan berbagi `pyAutomationID` yang sama, tetapi tidak pernah diisi `SetCauseOfLossValue_act` dan tidak punya kolom untuk dibaca kembali |

Baris ketiga dan keempat adalah contoh utang teknis §4.2 yang paling tajam di modul ini:
satu nama properti dipakai untuk tiga hal berbeda — nama kolom, dokumen JSON, dan pesan
galat. Baris terakhir adalah contoh lain: DUA isian berlabel sama di satu form, dan hanya
satu yang berfungsi.

### Nama di dalam procedure yang juga tidak dibawa

| Procedure | Padanan di Go |
|---|---|
| `id_site` | `site` di `issueID` |
| `id_mcouseofloss_ins` *(salah ketik dipertahankan Pega)* | `id` |
| `Datapega` | — tidak ada; dokumen JSON tidak lagi ditulis |
| `IDPega` | — tidak ada; ID datang dari jalur URL atau dibuat penyimpanan |
| `ErrMsg` | — tidak ada; diganti `error` Go dan kode galat HTTP |

### Nama yang sengaja dipilih

| Nama | Alasan |
|---|---|
| `CauseOfLoss` | istilah `CONTEXT.md` untuk Penyebab Kerugian; tunggal, karena ia satu baris |
| `ThreeDigits` | menyebut apa yang dilakukannya, bukan `pad` yang tidak menyebut lebarnya. Sengaja berbeda dari rincian yang memakai empat digit |
| `CountPendingJSON` | "pending" menyebut keadaannya — baris yang belum ikut dipindahkan — bukan `countJSON` yang tidak menjelaskan apa pun |
| `PrimaryKeyName` | konstanta, supaya kode dan migrasi tidak dapat berbeda pendapat diam-diam |
| `FieldDescription` | `"keterangan"` — nilainya mengikuti field JSON, karena layar memakainya untuk menandai kolom yang salah |

### Nama kueri SQL

```
cause_of_loss_list · cause_of_loss_get
cause_of_loss_site · cause_of_loss_next_sequence
cause_of_loss_insert · cause_of_loss_update
cause_of_loss_check_table · cause_of_loss_count_pending_json
```

Awalan `cause_of_loss_`, bukan `penyebab_kerugian_`: nama kueri adalah nama di dalam kode
(`D-80`). Yang berbahasa Indonesia hanyalah nama modulnya (`D-81`).

---

## Tambahan 2026-09-21 — modul Master XOL

Nama modul mengikuti `D-81`: `masterxol` (Go, tanpa tanda hubung) dan `master-xol`
(frontend, `kebab-case`). Ruas URL-nya `master/xol`.

"XOL" dipertahankan huruf besar seluruhnya di teks yang dilihat pengguna — ia singkatan
**Excess of Loss** dan memang ditulis begitu di layar lama serta di `CONTEXT.md`. Di dalam
nama Go ia mengikuti aturan singkatan: `XOLPage`, `masterxolhttp`, `useXOLList`.

### Kolom basis data ke nama domain

| Tabel | Kolom | Nama domain | Catatan |
|---|---|---|---|
| `MST_XOL_PNC` | `ID` | `Master.ID` | kunci utama; diterbitkan `max+1` |
| | `NAMA` | `Master.Name` | |
| | `TAHUN` | `Master.Year` | tahun treaty, bukan tanggal |
| | `KURSVALUE` | `Master.ExchangeRate` | RUPIAH per satu dolar |
| | `TYPEXOL` | `Master.Type` | kode `1`/`2`/`3`, boleh kosong |
| | `PIC` | `Master.PIC` | diisi server dari sesi |
| | `STSKOMITE` | `Master.CommitteeStatus` | `''` / `'0'` / `'1'` |
| | `KOMITE` | `Master.Committee` | hanya dibaca modul ini |
| | `REMARKPIC` | `Master.RemarkPIC` | |
| | `REMARKKOMITE` | `Master.RemarkCommittee` | hanya dibaca modul ini |
| | `EMAILKOMITE` | — | **tidak dibawa**; penerima notifikasi dari konfigurasi (`D-67`) |
| `MST_XOL_BUSINESS` | `IDBUSINESS` | `Business.ID` | boleh kosong — "TREATY INWARD" |
| | `GROUPBUSINESS` | `Business.Name` | |
| `MST_XOL_LAYER` | `IDLAYER` | `Layer.ID` | kunci utama; nomornya GLOBAL lintas induk |
| | `NAMA` | `Layer.Name` | |
| | `"LIMIT"` | `Layer.Limit` | DOLAR; **wajib dikutip** — kata cadangan PostgreSQL |
| | `EXCESS` | `Layer.Excess` | DOLAR |
| | `CONVERT_LIMIT` | `Layer.ConvertedLimit` | RUPIAH; dihitung server |
| `MST_XOL_REAS` | `IDREAS` | `Reinsurer.ID` | |
| | `NAMA` | `Reinsurer.Name` | dilabeli "Reasuransi" di layar |
| | `PERCENTSHARE` | `Reinsurer.Share` | persen utuh |

### Nama properti Pega yang TIDAK dibawa

Keempatnya dipakai untuk hal yang sama sekali berbeda dari namanya. Membawa namanya berarti
membawa kekeliruannya (`D-19`), dan buktinya ada di `RDB List/SetMasterXOL-SQL.xml`.

| Properti Pega | Kolom sebenarnya | Kenapa namanya menyesatkan |
|---|---|---|
| `TempXOL.UserName` | `NAMA` | ia nama MASTER XOL, bukan nama pengguna |
| `TempXOL.Amount` | `KURSVALUE` | ia KURS, bukan nilai klaim |
| `TempXOL.AreaClaimId` | `REMARKPIC` | ia CATATAN PIC, bukan id area klaim |
| `TempXOL.CNPSupportDoc` | `TYPEXOL` | ia JENIS XOL, bukan dokumen pendukung |
| `TempXOL.ObjectList` | daftar **layer** | ia lapisan treaty, bukan objek pertanggungan |
| `TempXOL.ComiteeList` | daftar **grup bisnis** | tidak ada hubungannya dengan komite |
| `.ObjectCoverageList` | daftar **reas** | bukan coverage objek |
| `.ClaimAmount` / `.TSIObject` | `LIMIT` / `EXCESS` | bukan nilai klaim dan bukan TSI |

Dua yang terakhir paling berbahaya: `ComiteeList` yang justru berisi grup bisnis, dan
`ObjectCoverageList` yang berisi reasuradur. Keduanya akan terbaca benar oleh siapa pun yang
mengenal domain klaim — dan justru karena itu salah.

### Nama tipe Go

| Nama | Alasan |
|---|---|
| `Amount` | satuan BERBEDA per field; dinyatakan di komentar tipe, bukan di namanya |
| `Share` | terpisah dari `Amount` meski keduanya `int64` — yang satu persen, yang lain uang |
| `Type` | kode Type XOL; `TypeLabel()` yang menerjemahkannya |
| `CommitteeStatus` | tipe tersendiri, bukan `string`, supaya `'0'` dan `'1'` tidak tertukar |
| `Reinsurer`, bukan `Reas` | "Reas" singkatan yang hanya dikenal di dalam rumah; tipe adalah kode (`D-80`) |
| `Business`, bukan `BusinessGroup` | kolomnya `GROUPBUSINESS`, tetapi yang disimpan satu grup — bukan daftar grup |

### Nama kueri SQL

Seluruhnya berawalan `xol_`, lalu kata kerja, lalu sasarannya:

```
xol_list                      xol_get                    xol_business_list
xol_layer_list                xol_reas_list              xol_lock_master_ids
xol_lock_layer_ids            xol_insert_master          xol_update_master
xol_business_count            xol_insert_business        xol_insert_layer
xol_update_layer              xol_reas_count             xol_insert_reas
xol_update_reas               xol_delete_master          xol_delete_business_of_master
xol_delete_reas_of_master     xol_delete_layer_of_master xol_delete_business
xol_delete_layer              xol_delete_reas_of_layer   xol_delete_reas
xol_layer_owner               xol_submit_committee       xol_year_list
xol_business_group_list       xol_orphan_count           xol_check_table
```

Akhiran `_of_master` dan `_of_layer` membedakan **hapus berkaskade** dari hapus satu baris.
Perbedaan itu penting dibaca sekilas: yang satu membuang seluruh anak, yang lain satu baris.

### Nama field JSON

Berbahasa Indonesia karena ia **kontrak**, bukan nama internal (`D-80`):
`id`, `nama`, `tahun`, `kurs`, `tipe`, `tipe_label`, `remark_pic`, `pic`, `status_komite`,
`komite`, `remark_komite`, `bisnis`, `layer`, `reas`, `limit`, `excess`, `limit_idr`,
`share`, `peringatan`.

`limit_idr`, bukan `convert_limit`: nama kolomnya menyatakan CARA, nama kontraknya
menyatakan ISI — dan yang dibaca klien adalah isinya.


---

## Tambahan 2026-09-21 — modul Master Dokumen Travel & penyelarasan Master PIC Teknik

### Nama modul yang ditetapkan sesi ini

| Nama modul bisnis (Work Owner) | Folder backend / paket Go | Folder frontend |
|---|---|---|
| Master Dokumen Travel | `internal/masterdokumentravel` | `src/modules/master-dokumen-travel` |

Komponennya memakai **nama tipe domain**, bukan nama modul: `TravelDocumentPage.tsx` dan
`TravelDocumentForm.tsx` di dalam `master-dokumen-travel/` — mengikuti `D-81`.

### Istilah domain baru

| Indonesia | Inggris | Catatan |
|---|---|---|
| Dokumen Travel | TravelDocument | satu **jenis** dokumen yang dapat diminta pada klaim lini Travel, kolom `M_DOCTRAVEL` |
| Judul Dokumen | Name (field) · `judul` (JSON) | kolom `NAMADOKUMEN`; label layar Pega-nya "Judul Dokumen" |
| Kode Situs | Site | kolom `M_SITE_DATABASE.ID`, bagian pertama setiap DOCID |
| Nomor Urut | Sequence | `DOCTRAVEL_SEQ.NEXTVAL`, bagian kedua DOCID |

`DOCID` **tidak diterjemahkan** di dalam komentar dan nama kueri: ia nama kolom, dan nama kolom
tetap milik basis data. Field Go-nya bernama `ID`, field JSON-nya `id`.

### Nama kueri `.sql`

| Nama | Isi |
|---|---|
| `travel_document_list` · `travel_document_get` | baca |
| `travel_document_site` · `travel_document_next_sequence` | bahan penerbitan DOCID |
| `travel_document_insert` · `travel_document_update` | tulis |
| `travel_document_check_table` | mode periksa |

### Nama yang sengaja TIDAK diterjemahkan — tambahan

| Nama | Alasan |
|---|---|
| **`PICTeknik`** | nama peran bisnis di `CONTEXT.md` ("PIC Teknik / User Teknis"), dan nama yang dipakai Work Owner saat menyebut modulnya. `TechnicalPIC` bukan istilah yang dikenal siapa pun di proyek ini — perlakuannya sejajar dengan `D-81` untuk nama modul |

### Master PIC Teknik diselaraskan ke standar `D-80`

Modul `internal/masterpicteknik` masuk lewat merge `2322f15` dengan penamaan pra-`D-80`, dan
menyebabkan dua paket gagal kompilasi. Diselaraskan sesi ini atas izin Work Owner.

| Sebelum | Sesudah |
|---|---|
| `http/galat.go` · `http/rute.go` | `http/errors.go` · `http/routes.go` |
| `repo/memori/{memori,contoh}.go` | `repo/memory/{memory,sample}.go` |
| `repo/sqlstore/kueri.go` | `repo/sqlstore/query.go` |
| `usecase/kelola.go` | `usecase/manage.go` |

Identifier mengikuti kamus yang sudah ada di dokumen ini. Yang perlu dicatat karena belum ada
entrinya:

| Indonesia | Inggris |
|---|---|
| Atasan | Supervisor |
| Grup · GrupPanel | Group · GroupPanel |
| Kuota · KuotaLuar · KuotaMaksimum | Quota · ExternalQuota · MaxQuota |
| LiniBisnis | BusinessLine |
| IDOperator · NamaOperator | OperatorID · OperatorName |
| Direktori · DirektoriOperator | Directory · OperatorDirectory |
| Daftarkan | Add |
| SandiAktif · bacaAktif | ActiveCode · readActive |
| pemindai · pindaiSatuBaris | rowScanner · scanRow |
| petakanGalat · muatSeluruhKueri · pecahPerNama | mapError · loadAllQueries · splitByName |
| PanjangXMaksimum | MaxXLength |

| Nama kueri sebelum | Sesudah |
|---|---|
| `pic_teknik_{daftar,ambil,sisip,perbarui,periksa_tabel}` | `pic_teknik_{list,get,insert,update,check_table}` |
| `operator_nama` | `operator_name` |

**Nama field JSON dan isi komentar tidak berubah sama sekali** — keduanya dilindungi oleh
pemindai token yang dipakai, karena bagi Go keduanya bukan identifier.

---

## Tambahan 2026-09-21 — modul Master COL Simas Online

### Nama modul (`D-81`)

| Nama modul bisnis (Work Owner) | Folder backend / paket Go | Folder frontend |
|---|---|---|
| Master COL SIMAS ONLINE | `internal/mastercolsimasonline` | `src/modules/master-col-simas-online` |

**Komponen memakai nama tipe domain, bukan nama modul** — karena itu `CauseOfLossPage.tsx` dan
`CauseOfLossForm.tsx`, bukan `MasterColSimasOnlinePage.tsx`.

### Istilah domain baru

COL adalah singkatan **Cause of Loss** — penyebab kerugian. Ia sudah ada di `CONTEXT.md` sebagai
istilah bisnis; yang di bawah adalah padanan Inggris yang dipakai di kode.

| Indonesia / Pega | Inggris | Catatan |
|---|---|---|
| `M_COL_ID` | `Code` | kode baris, sejajar `ClaimStatus.Code` — bukan "ID" generik |
| `COL_DESC` | `Description` | di layar berlabel "Nama Cause of loss" |
| `MST_COL_ID` | `MasterCode` | "ID Master Kerugian" — kunci padanan di sistem **Simas Online**, bukan kunci baris ini |
| `BISNISID` | `Businesses []Business` | **jamak**: `pyPageListProperty` membuktikan satu COL dipakai banyak bisnis |
| `BUSINESS.ID` | `Business.ID` | |
| `BUSINESS.NOTE` | `Business.Name` | **bukan** `Note`. Kueri lama mengaliaskannya `NOTE as "Note"`, dan membawa alias itu akan membuat seluruh kode menyebut nama bisnis sebagai "catatan" — persis kelas kesalahan yang `D-19` cegah |
| `STS_AKTIF` | `active` (hanya di SQL) | penanda soft delete pada tabel pemetaan; tidak muncul sebagai field domain |
| Penyebab Kerugian | CauseOfLoss | |
| Bisnis / Lini Bisnis | Business | |

### Kata kerja tambahan

| Indonesia | Inggris | Catatan |
|---|---|---|
| GantiSeluruhPemetaan | replaceBusinesses | menandai semua tidak aktif lalu menghidupkan yang dipilih |
| TerbitkanKode | issueCode | situs disambung nomor urut tiga digit |
| KosongJadiNull | nullIfEmpty | isian opsional yang kosong disimpan NULL, bukan teks kosong |
| KembarPertama | firstDuplicate | |
| IDBisnisTidakDikenal | unknownBusinessIDs | |
| DalamTransaksi | inTransaction | |
| DenganNamaBisnis | withBusinessNames | hanya di adapter memori |

### Nama kueri `.sql`

Berawalan `cause_of_loss_` mengikuti isi tabelnya, kecuali dua kueri master bisnis yang berawalan
`business_` mengikuti nama tabelnya sendiri.

| Kueri | Isi |
|---|---|
| `cause_of_loss_{list,get,insert,update}` | tabel `M_CAUSE_OF_LOSS` |
| `cause_of_loss_{site,next_sequence}` | bahan pembentuk kode |
| `cause_of_loss_business_{list,deactivate_all,activate,insert}` | tabel pemetaan `M_CAUSE_OF_LOSS_BUSINESS` |
| `business_list` | `POOLDATA.BUSINESS` — **baca-saja**, milik GISFW |
| `cause_of_loss_check_table` · `cause_of_loss_business_check_table` · `business_check_table` | mode periksa |

### Nama field JSON — tetap Indonesia

Ia kontrak API, bukan nama internal:

| Field | Memetakan ke |
|---|---|
| `id` | `Code` |
| `nama` | `Description` |
| `id_master_kerugian` | `MasterCode` |
| `bisnis` | `Businesses` (respons: objek; permintaan: **senarai ID saja**) |

### Teks layar — mengikuti Pega apa adanya

`D-13`: tampilan meniru Pega supaya pengguna tidak belajar ulang. Keempat teks berikut disalin
dari caption rule, termasuk gaya penulisannya:

| Teks | Asal |
|---|---|
| "Master Cause Of Loss Simas Online" | `Section/Online_GridCauseOfLoss-Section.xml` |
| "Memperbaharui Data Simas Online" | `Section/Online_BrowseCauseOfLoss-Section.xml` |
| "Nama Cause of loss" · "ID Master Kerugian" · "Bisnis" · "ID" | section yang sama |
| "Tambah" · "Refresh" · "Simpan" · "Ubah" | kedua section |

Satu teks **tidak** ada di Pega dan karena itu disusun sendiri: judul form mode tambah,
**"Tambah Data Simas Online"**. Layar lama memakai judul yang sama untuk kedua modusnya karena
modusnya ditandai `pyLabel`, bukan oleh judul.

### Koreksi 2026-09-21 — setelah pemeriksaan ulang ke export Pega

Empat nama dan satu komponen berubah artinya. Yang lama dicatat supaya koreksinya terbaca, bukan
dihapus diam-diam.

| Nama | Arti yang KELIRU | Arti yang benar |
|---|---|---|
| `CauseOfLoss.MasterCode` (`MST_COL_ID`) | kunci padanan di sistem Simas Online | **Code baris LAIN di master yang sama** — penyebab kerugian induk |
| `Business.Name` | selalu `BUSINESS.NOTE` dari master | nama bisnis: dari master bila cocok, **atau apa yang diketik petugas** bila tidak |
| `Business.ID` | selalu terisi | **boleh kosong** pada nama yang diketik bebas |
| `Input.BusinessIDs` | senarai ID | **`Input.BusinessNames`** — senarai NAMA; ID diselesaikan di server |

### Istilah dan tipe baru

| Indonesia / Pega | Inggris | Catatan |
|---|---|---|
| Isi yang benar-benar disimpan | `SaveData` | terpisah dari `Input`; di antara keduanya ada penyelesaian nama menjadi ID |
| Induk (cause of loss) | `MasterCode` / parent | |
| Nama bisnis | `NAMA_BISNIS` (kolom) | identitas baris pemetaan, karena `BISNISID` boleh NULL |
| Urutan baris grid | `URUTAN` (kolom) | susunan yang disusun pengguna |

| Kata kerja | Inggris | Catatan |
|---|---|---|
| PeriksaInduk | `checkParent` | menolak induk yang tidak ada dan rujukan-diri |
| SelesaikanBisnis | `resolveBusinesses` | nama menjadi pasangan nama+ID |
| MasterBisnis | `businessMaster` | peta bernama; kosong bila master tidak terbaca |
| SamaBisnis | `SameBusiness` | **diekspor** — dipakai domain, adapter, dan indeks unik basis data |
| KeSaveData | `toSaveData` | |

### Nama kueri `.sql` — yang berubah

| Kueri | Perubahan |
|---|---|
| `cause_of_loss_business_list` | tidak lagi join `POOLDATA.BUSINESS`; membaca `NAMA_BISNIS` dan `URUTAN` |
| `cause_of_loss_business_activate` | dicocokkan menurut `UPPER(TRIM(NAMA_BISNIS))`, bukan `BISNISID` |
| `cause_of_loss_business_insert` | ikut menyisipkan `NAMA_BISNIS` dan `URUTAN` |

### Komponen antarmuka baru

| Berkas | Peran |
|---|---|
| `src/components/ComboField.tsx` | isian teks dengan saran (`<input list>` + `<datalist>`) — menawarkan daftar **tanpa memaksanya**, meniru autocomplete Pega ber-`pyAllowFreeFormInput=true` |

### Teks layar — dikoreksi agar persis Pega

| Sebelum | Sesudah | Asal |
|---|---|---|
| "ID" | **"ID Kerugian"** | `Online_BrowseCauseOfLoss-Section.xml:4301` |
| "Tambah Data Simas Online" | **"Memperbaharui Data Simas Online"** untuk kedua modus | `:4103` — satu caption literal, modusnya ditandai `pyLabel` |

Urutan isian juga dikoreksi menjadi: **ID Kerugian → ID Master Kerugian → Nama Cause of loss →
Bisnis** (`:4301`, `:4489`, `:4813`, `:5874`).

### Koreksi kedua 2026-09-21 — layar disamakan penuh dengan Pega

Keempat penyimpangan dicabut. Satu nama berganti karena perannya berubah.

| Nama lama | Nama baru | Sebab |
|---|---|---|
| `SameBusiness(a, b) bool` | **`NormalizeBusinessName(name) string`** | Ia tidak lagi mendeteksi kembar — kembar diizinkan seperti di Pega. Perannya kini mencocokkan nama yang diketik ke master bisnis, dan bentuk fungsinya ikut berubah dari pembanding menjadi penormal |

### Kunci tabel pemetaan bisnis — berpindah

| Sebelum | Sesudah | Sebab |
|---|---|---|
| indeks unik `(M_COL_ID, UPPER(TRIM(NAMA_BISNIS)))` | **PRIMARY KEY `(M_COL_ID, URUTAN)`** | nama tidak unik (kembar diizinkan) dan `BISNISID` boleh NULL; yang tersisa adalah posisi baris |

Urutan kolom pada tabel pun disesuaikan supaya kuncinya terbaca lebih dulu:
`M_COL_ID, URUTAN, BISNISID, NAMA_BISNIS, STS_AKTIF`.

### Aturan isian yang dibuang

Seluruhnya karena Pega tidak memilikinya:

| Aturan | Bukti ketiadaannya di Pega |
|---|---|
| Nama Cause of loss wajib diisi | `pyRequired=false` pada 19 isian; nol Validate rule |
| Bisnis kembar ditolak | nol penanda unique di repeat grid |
| Rujukan-diri ditolak | dropdown bersumber RD **tanpa penyaring** |

Yang **tetap** hanyalah `MaxDescriptionLength`, `MaxMasterCodeLength`, dan
`MaxBusinessNameLength` — ketiganya penjaga terhadap lebar kolom, bukan aturan bisnis.

### Teks layar — koreksi terakhir

| Sebelum | Sesudah | Asal |
|---|---|---|
| "Tambah Data Simas Online" (mode tambah) | **"Memperbaharui Data Simas Online"** untuk KEDUA modus | `Online_BrowseCauseOfLoss-Section.xml:4103` — satu caption literal; modusnya ditandai `pyLabel`, bukan judul |

---

## Modul Daftar Tipe Dokumen (2026-09-21)

Nama modul diambil dari nama menu yang dipakai Work Owner — "Daftar Tipe Dokumen" (`MENU_ID 40`),
sesuai `D-81`: folder modul dinamai menurut nama modul bisnis, isinya berbahasa Inggris (`D-80`).

| Lapisan | Nama |
|---|---|
| Folder & paket backend | `internal/daftartipedokumen` (tanpa tanda hubung — Go tidak mengizinkannya) |
| Paket transport | `daftartipedokumenhttp` |
| Folder frontend | `src/modules/daftar-tipe-dokumen` (`kebab-case`) |

### Istilah dan tipe

| Kolom Pega | Tipe/field Go | Field JSON API | Alasan |
|---|---|---|---|
| `ID` | `DocumentType.ID` | `id` | — |
| `TYPE_DOCUMENT` | `DocumentType.Type` | `tipe_dokumen` | mengikuti header grid Pega |
| `STS_PROSES` | `DocumentType.ProcessStatus` | `status_proses` | mengikuti label layar Pega |
| `USER_EDIT` | `Editor.Identity` | — | jejak simpan; tidak pernah dikirim ke klien |
| `TGL_EDIT` | `Editor.At` | — | idem |
| `OLD_ID` | *tidak dimodelkan* | — | jejak sejarah; tidak tampil di layar Pega mana pun |

`Editor` sengaja tipe tersendiri, bukan dua field tambahan pada `DocumentType`: ia **bukan isi
baris** melainkan **keterangan tentang penyimpanannya**, dan menggabungkannya akan membuat keduanya
tampak sama-sama dapat diisi pengguna.

### Nama kueri `.sql`

`document_type_list` · `document_type_get` · `document_type_site` · `document_type_next_sequence` ·
`document_type_insert` · `document_type_update` · `document_type_check_table`

Awalannya `document_type_`, bukan `daftar_tipe_dokumen_` — nama kueri menyebut **sumber dayanya**,
bukan nama layarnya, sejalan dengan `travel_document_` dan `cause_of_loss_` pada modul tetangga.

### Nama komponen frontend

| Berkas | Isi |
|---|---|
| `DocumentTypePage.tsx` | layar daftar |
| `DocumentTypeForm.tsx` | form tambah dan ubah |
| `api.ts` | `useDocumentTypeList`, `useCreateDocumentType`, `useUpdateDocumentType` |

Komponen dinamai menurut **tipe domainnya** (`DocumentType`), bukan menurut nama modul — sama seperti
`AccountPage.tsx` pada modul `master-rekening`.

### Jalur

| Hal | Nilai |
|---|---|
| API | `/api/master/tipe-dokumen` dan `/api/master/tipe-dokumen/{id}` |
| Layar | `/master/tipe-dokumen` |
| Kunci menu | `ListDocumentTypeInbox` |
| Kode galat | `tipe_dokumen_tidak_ditemukan`, `permintaan_cacat` |

Jalurnya `tipe-dokumen`, bukan `daftar-tipe-dokumen`: "Daftar" pada nama menu menyatakan **bentuk
layar**, dan bentuk itu sudah dinyatakan metode HTTP-nya.

### Teks layar — mengikuti Pega apa adanya

| Tempat | Teks | Sumber |
|---|---|---|
| Judul layar | "Daftar Tipe Dokumen" | `MENU_DESC` pada `m_menu_aplikasi_pnc.csv:35` |
| Header kolom | "ID", **"Tipe Dokumen"**, "Status Proses" | `Section/BrowseListDocumentType-Section.xml` |
| Label isian form | **"Jenis Dokumen"** | `Section/ListDocumentType-Section.xml` |
| Judul form | "Update Data" — **termasuk pada mode tambah** | `BrowseListDocumentType-Section.xml:11784` |
| Tombol | "Tambah", "Refresh", "Simpan", "Batal" | kedua section |
| Aksi per baris | "Update Data" | `BrowseListDocumentType-Section.xml` |

Dua label yang **sengaja berbeda satu sama lain** — "Tipe Dokumen" di grid, "Jenis Dokumen" di form —
memang berbeda di layar lama, dan keduanya ditiru (`D-13`).

Satu teks yang **bukan** dari Pega, dan ditulis dalam bahasa Indonesia sebagai tambahan: keterangan
di bawah isian Status Proses, *"Catatan bebas, bukan status aktif/non-aktif. Boleh dikosongkan."* Ia
ada karena label "Status Proses" menyesatkan tanpa penjelasan — lihat `keputusan-implementasi.md`
§21.2.

---

## Modul Daftar Detail Dokumen Travel (2026-09-22)

Nama modul mengikuti `D-81` — nama yang disebut Work Owner, sama dengan `MENU_DESC` pada MENU_ID 39.
Isi modulnya berbahasa Inggris sesuai `D-80`.

### Nama modul di setiap lapisan

| Lapisan | Bentuk |
|---|---|
| Paket Go dan folder backend | `internal/daftardetaildokumentravel` |
| Nama paket transport | `daftardetaildokumentravelhttp` |
| Folder frontend | `src/modules/daftar-detail-dokumen-travel` |
| Rute layar | `/master/daftar-detail-dokumen-travel` |
| Jalur API | `/api/master/daftar-detail-dokumen-travel` |
| Kunci menu | `ListDocumentTravel` |

Berbeda dari modul Daftar Tipe Dokumen, kata **"Daftar" TIDAK dibuang** dari jalurnya. Sebabnya di
sini ia bukan menyatakan bentuk layar melainkan **bagian dari nama masternya**, dan membuangnya akan
menghasilkan `/master/detail-dokumen-travel` yang hanya berbeda satu kata dari
`/master/dokumen-travel` milik master induknya — dua jalur yang tertukar saat dibaca sekilas, pada
dua layar yang datanya memang bertaut.

### Kolom basis data → nama Go → field JSON

**`V_LST_DOC_TRAVEL`** — baris aturan dokumen:

| Kolom | Go | JSON | Label layar (Pega) |
|---|---|---|---|
| `ID` | `Detail.ID` | `id` | ID |
| `DOCID` | `Detail.DocumentID` | `id_dokumen` | ID Dokumen |
| `DOCUMENTNAME` | `Detail.DocumentName` | `nama_dokumen` | Nama Dokumen |
| `STSWAJIB` | `Detail.Mandatory` (**bool**) | `status_wajib` (**bool**) | Status Wajib |
| `MINUNGGAH` | `Detail.MinUpload` (`int`) | `minimal_unggah` (`number`) | Minimal Unggah |

**`V_LST_DOC_TRAVEL_COVERAGE`** — pembatasan per plan dan jaminan:

| Kolom | Go | JSON | Label layar (Pega) |
|---|---|---|---|
| `ID` | `Coverage.ID` | `id` | — |
| `TRAVELDOCID` | *(tidak dibawa keluar repo)* | — | — |
| `PLANID` | `Coverage.PlanID` | `id_plan` | — |
| `PLANNAME` | `Coverage.PlanName` | `nama_plan` | Nama Plan |
| `COVERAGEID` | `Coverage.CoverageID` | `id_jaminan` | — |
| `COVERAGENAME` | `Coverage.CoverageName` | `nama_jaminan` | Nama Jaminan |

`TRAVELDOCID` tidak pernah keluar dari lapisan penyimpanan: ia kunci penghubung, dan pemanggil sudah
memegang induknya. Membawanya keluar berarti menawarkan dua sumber untuk satu nilai yang sama.

`STSWAJIB` **berubah bentuk** di batas repo: angka `1`/`0` di basis data — dibuktikan precondition
`.STSWAJIB==1` pada `Activity/BrowseDocTravel-Act.xml` — menjadi `bool` sejak lapisan domain ke atas.
Penerjemahannya ada di `repo/sqlstore` (`mandatoryCode`, `scanDetail`), satu-satunya tempat angka itu
disebut.

### Daftar pilihan — tabel milik pihak lain

| Tabel | Pemilik | Go | Jalur API | Field JSON |
|---|---|---|---|---|
| `M_DOCTRAVEL` | modul Master Dokumen Travel (`P-1`) | `Document{ID, Name}` | `/api/master/dokumen-travel-pilihan` | `dokumen[].id`, `.nama` |
| `M_PLANTRAVEL` | GISFW (`D-03`) | `Plan{ID, Name}` | `/api/master/plan-travel` | `plan[].id`, `.nama` |
| `M_PLANTRAVEL` | GISFW (`D-03`) | `CoverageOption{ID, Name, PlanID}` | jalur yang sama | `jaminan[].id`, `.nama`, `.id_plan` |

`Document` sengaja BUKAN `masterdokumentravel.TravelDocument`: modul tidak saling mengimpor tipenya.
Yang dibagi adalah tabelnya, bukan kodenya.

Jalur `dokumen-travel-pilihan` sengaja berbeda dari `/api/master/dokumen-travel` milik modul induk,
meski keduanya membaca tabel yang sama — yang satu daftar yang dapat disunting, yang lain daftar
pilihan.

### Nama berkas

| Backend | Frontend |
|---|---|
| `daftardetaildokumentravel.go` | `TravelDocumentDetailPage.tsx` |
| `usecase/manage.go` | `TravelDocumentDetailForm.tsx` |
| `repo/memory/memory.go`, `sample.go` | `api.ts` |
| `repo/sqlstore/daftardetaildokumentravel.{go,sql}`, `query.go` | `TravelDocumentDetailPage.test.tsx` |
| `http/{dto,errors,handler,routes}.go` | |

Nama berkas frontend memakai nama **tipe domain** (`TravelDocumentDetail`), bukan nama modul —
sejalan dengan `AccountPage.tsx` pada `master-rekening`.

### Kode galat

| Kode | Kapan |
|---|---|
| `detail_dokumen_travel_tidak_ditemukan` | baris yang dituju tidak ada |
| `permintaan_cacat` | badan JSON tidak dapat dibaca, atau memuat field tak dikenal |

Kode pertama ditambahkan ke `ErrorCode` di `api/types.ts` supaya layar dapat mengenalinya. Modul yang
memakai kode "tidak ditemukan" miliknya sendiri alih-alih `tidak_ditemukan` yang umum adalah keadaan
yang ada hari ini, bukan rancangan — penyeragamannya masuk `TKT-F1-004`.

### Teks layar — mengikuti Pega apa adanya

| Tempat | Teks | Sumber |
|---|---|---|
| Judul layar | "Detail Dokumen Travel" | `Section/LSTDocumentTravel-Section.xml` |
| Header kolom | "ID", "ID Dokumen", "Nama Dokumen", "Status Wajib", "Minimal Unggah" | `Section/BrowseDocumentTravel-Section.xml` |
| Nilai Status Wajib | "Ya" / "Tidak" | `Activity/BrowseDocTravel-Act.xml` |
| Judul form | "Detail Dokumen Travel" — **termasuk pada mode tambah** | section yang sama |
| Tombol | "Tambah", "Refresh", "Simpan", "Ubah", "Batal" | kedua section |
| Label grid dalam form | "Nama Plan", "Nama Jaminan" | `BrowseDocumentTravel-Section.xml` |

Teks yang **bukan** dari Pega, ditulis sebagai tambahan dalam bahasa Indonesia: judul kelompok
**"Plan dan Jaminan"** pada grid berulang (Pega tidak memberinya judul), keterangan di bawah isian
ID Dokumen yang menyebutkan nama dokumen menurut master, dan kalimat *"Dibiarkan kosong berarti
aturan dokumen ini berlaku untuk seluruh plan dan jaminan"* — yang terakhir ada karena grid kosong
di layar lama tidak menjelaskan apa artinya kosong.
