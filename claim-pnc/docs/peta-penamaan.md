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

## Tambahan 2026-09-20 — modul View History Claim (`riwayatklaim`)

Modul ini **kasus paling pekat** dari alias menyesatkan di seluruh export, dan karena itu
pemetaannya dicatat utuh di sini — bukan hanya di berkas `.sql`-nya.

### Nama modul

`riwayatklaim` di backend, `riwayat-klaim` di frontend. Ia mengikuti `D-81`: nama modulnya
berbahasa Indonesia karena itulah nama yang dipakai Work Owner, sementara isinya berbahasa
Inggris. Judul yang dibaca pengguna tetap **"View History Claim"**, mengikuti judul layar
Pega (`D-13`).

### Properti grid Pega → arti sebenarnya → nama di kode

Dua belas dari enam belas kolom bernama sesuatu yang sama sekali tidak menyatakan isinya.

| Properti grid Pega | Kolom basis data | Arti bagi pengguna | Nama di kode |
|---|---|---|---|
| `.IDPEGA` | `CLAIMID` | kunci teknis Pega | `Reference` |
| `.EDMNO` ⚠ | `CLAIMNO` | No Klaim | `Number` |
| `.NOPOLIS` | `NOPOLIS` | No Polis | `PolicyNumber` |
| `.QQNAME` | `QQNAME` | Nama Tertanggung | `InsuredName` |
| `.STARTDATE` ⚠ | `DATEOFLOSS` | Tgl Kejadian | `LossDate` |
| `.BUSINESSNAME` | `BUSINESSNAME` | Bisnis | `BusinessName` |
| `.BRANCHNAME` | `BRANCHNAME` | Cabang | `BranchName` |
| `.STATUSBUSINESS` ⚠ | `STATUSWORK` | Status | `WorkStatus` |
| `.THEINSURED` ⚠ | `V_STS_CLAIM.LSC_NOTE` | Posisi Klaim | `ClaimPosition` |
| `.ENDDATE` ⚠ | `CLOSECLAIMDATE` | Tanggal Close | `CloseDate` |
| `.FLAGEDMBATAL` ⚠ | `CLOSECLAIMNOTE` | Catatan Close | `CloseNote` |
| `.SOBNAME` ⚠ | `PICTEKNIK` | PIC Teknis | `TechnicalPIC` |
| `.OLDPOLICYNO` ⚠ | `DETAIL_PNC_SALVAGE.NOAKSEPTASI` | No Akseptasi | `AcceptanceNumber` |
| `.WARRANTYNO` ⚠ | `DETAIL_PNC_SALVAGE.IDBALAILELANG` | No Balai Lelang | `AuctionHouseID` |
| `.SOBLEADER1` ⚠ | `T_PERSON.FULLNAME` | Nama Objek | `InsuredItemName` |
| `.EDMDATE` ⚠ | `T_PERSON.ASMDATEOFBIRTH` | Tanggal Lahir | `BirthDate` |

⚠ menandai nama yang menyesatkan secara aktif. `.THEINSURED` yang berarti **Posisi Klaim**
dan `.FLAGEDMBATAL` yang berarti **Catatan Close** adalah dua yang paling jauh.

`InsuredItemName` memakai istilah `CONTEXT.md`: objek pertanggungan, bukan "Object" yang
bertabrakan dengan makna pemrograman.

### Isian formulir — nama properti tertukar satu sama lain

Inilah sumber salah satu cacat yang direplikasi: **dua properti tanggal yang namanya
justru tertukar dengan perannya.**

| Label di layar | Properti Pega | Nama di kode | Dipakai kueri? |
|---|---|---|---|
| Nama Pencarian | `TempSearch.SearchName` | `Text` | ya, untuk tipe teks |
| Tanggal Pencarian | `TempSearch.DateOfSendInputor` ⚠ | `SearchDate` | ya, untuk SELURUH tipe tanggal |
| Tanggal Lahir | `TempSearch.SearchDate` ⚠ | `BirthDate` | **tidak pernah** |

Properti bernama `SearchDate` adalah isian **Tanggal Lahir**, dan properti bernama
`DateOfSendInputor` adalah isian **Tanggal Pencarian**. Nama di kode mengikuti **label yang
dibaca pengguna**, bukan nama propertinya — kalau tidak, kode ini akan mewarisi persis
kekeliruan yang membuat cacatnya lahir.

### Gerbang proteksi data — alias yang tidak dapat ditebak

Kueri lama membaca master proteksi dengan alias yang tak satu pun menyatakan isinya. Arti
keenamnya hanya terbaca dari komentar langkah di activity-nya.

| Alias di kueri lama | Kolom basis data | Arti | Nama di kode |
|---|---|---|---|
| `City` ⚠ | `LOGSEEN` | jatah **lihat data** (layar rincian) | `ViewQuota` |
| `CityID` ⚠ | `LOGSEARCH` | jatah **pencarian** (layar ini) | `SearchQuota` |
| `Country` ⚠ | `STS_NOTELP` | masking nomor telepon | `MaskPhone` |
| `CountryID` ⚠ | `STS_EMAIL` | masking surel | `MaskEmail` |
| `Province` ⚠ | `STS_KTP` | masking nomor KTP | `MaskIDCard` |
| `ProvinceID` ⚠ | `SUBMODUL` | daftar submodul | `SubModules` |

| Indonesia | Inggris | Catatan |
|---|---|---|
| Tipe pencarian | `SearchType` | |
| Kriteria pencarian | `Criteria` | dibentuk hanya lewat `NewCriteria` |
| Gerbang / keadaan izin | `Access` | `Check` memeriksa, `Grant` memakai satu jatah |
| Jatah | `Quota` | `QuotaTotal` · `QuotaUsed` · `QuotaRemaining` |
| Pemakaian jatah | `Usage` | satu baris jejak; `ConsumesQuota` membedakan buka layar dari pencarian |
| Baris proteksi | `Protection` | isi `MST_PROTEKSI_DATA_PNC` |

**Nama field JSON tetap Indonesia** — `nomor_klaim`, `posisi_klaim`, `pic_teknis`,
`tipe_pencarian`, `proteksi`, `jatah_sisa`. Ia kontrak API.

**Nama kueri `.sql`** berawalan `search_` untuk kesebelas pencarian dan `protection_`
untuk gerbangnya: `search_policy_number` · `search_claim_number` · `search_birth_date` ·
`protection_find` · `protection_count_usage` · `protection_record_usage` ·
`protection_check_table`.

**Nama tabel baru** tetap Indonesia karena ia milik basis data (`D-80`):
`POOLDATA.CPNC_PEMAKAIAN_PROTEKSI`.

## Tambahan 2026-09-20 — modul Inbox XOL (`inboxxol`)

### Nama modul

| Lapisan | Nama |
|---|---|
| Backend, folder dan paket Go | `internal/inboxxol` |
| Frontend, folder modul | `src/modules/inbox-xol` |
| Rute antarmuka | `/inbox-xol` |
| Awalan rute API | `/api/inbox-xol` |

Mengikuti `D-81`: nama modulnya diambil dari nama yang dipakai Work Owner dan tertulis di
menu — "Inbox XOL". Isinya tetap berbahasa Inggris (`D-80`).

### Properti Pega → arti sebenarnya → nama di kode

Alias di layar ini menyesatkan lebih parah daripada modul mana pun sebelumnya: namanya
bukan singkatan tidak lazim, melainkan **berarti hal lain**.

**Grid "DATA XOL BASED ON DOL AND COL"** — dari `GetDataXOL_Calulation`:

| Properti Pega | Kolom sumber | Artinya | Nama di kode |
|---|---|---|---|
| `.ASMFull` | `DOL` | Tanggal Kejadian | `ClaimSummary.LossDate` |
| `.AcceptedNo` | `CAUSEOFLOSS` | Penyebab Kerugian | `ClaimSummary.CauseOfLoss` |
| `.Currency` | `SUM(OSVALUE)` | Nilai Outstanding | `ClaimSummary.OutstandingValue` |
| `.CurrencyID` | `SUM(AKSEPVALUE)` | Nilai Akseptasi | `ClaimSummary.AcceptedValue` |
| `.BranchOfBank` | — (dari master) | Nama Group Business | `ClaimSummary.BusinessGroup` |

**Grid "PILIH MASTER XOL"** — dari `GetDataMasterXOL` kelas `Data-Adjustment`, yang
**tidak ada di export** dan direkonstruksi:

| Properti Pega | Artinya | Nama di kode |
|---|---|---|
| `.CurrencyName` | Tahun XOL | `MasterXOL.Year` |
| `.AcceptedNo` | Kurs perjanjian | `MasterXOL.ExchangeRate` |
| `.Currency` | Nama group business | `MasterXOL.BusinessGroups[].Name` |
| `.CurrencyID` | Kode group business | `MasterXOL.BusinessGroups[].ID` |
| `.Notes` | Kode master XOL | `MasterXOL.ID` |

**Grid rincian `Sec_Detail_claim_XOL`** — dari `GetDataMasterXOL` kelas `Data-ClaimData`
dan `GetDataXOLPerBusiness`:

| Properti Pega | Kolom sumber | Artinya | Nama di kode |
|---|---|---|---|
| `.BranchID` | `MST_XOL_PNC.ID` | Kode Master XOL | `MasterXOL.ID` |
| `.UserName` | `MST_XOL_PNC.NAMA` | Nama Master XOL | `MasterXOL.Name` |
| `.UserAdmin` | `MST_XOL_PNC.TAHUN` | Tahun XOL | `MasterXOL.Year` |
| `.Amount` | `MST_XOL_PNC.KURSVALUE` | Kurs | `MasterXOL.ExchangeRate` |
| `.FlagASO` | `STSKOMITE` | Status Komite | `MasterXOL.CommitteeStatus` |
| `.NoteKasir` | `REMARKKOMITE` | Catatan Komite | `MasterXOL.CommitteeNote` |
| `.CABANG` | `TYPEXOL` | Tipe Master | `MasterXOL.Type` |
| `.CloseClaimNote` | `REMARKPIC` | Catatan PIC | `MasterXOL.PICNote` |
| `.Country` | `businessgroup.NOTE` | Nama Group Business | `BusinessBreakdown.BusinessGroup` |
| `.IsDLA` | `COUNT(DISTINCT claimno)` | Jumlah Klaim | `BusinessBreakdown.ClaimCount` |
| `.DLAShare` | `SUM(os_value)` | Nilai Outstanding | `BusinessBreakdown.OutstandingValue` |
| `.KlaimAmount` | `SUM(aksep_value)` | Nilai Akseptasi | `BusinessBreakdown.AcceptedValue` |
| `.IsKirim` | `businessgroupid` | Kode Group Business | `BusinessBreakdown.BusinessGroupID` |

**Grid PLA/DLA** — dari `BrowseAllDataXOL_PLA`:

| Properti Pega | Kolom sumber | Artinya | Nama di kode |
|---|---|---|---|
| `.ResponseCode` / `.CaseID` / `.ref_no` | `NO_PLADLAXOL` | Nomor PLA/DLA | `Advice.Number` |
| `.CoverInsKey` | `NAMAREAS` | Nama Reasuradur | `Advice.ReinsurerName` |
| `.CABANG` | `NAMALAYER` | Nama Layer | `Advice.LayerName` |
| `.ClaimFrom` | `TAHUN` | Tahun XOL | `Advice.Year` |
| `.PNCSearch` | `KURS` | Kurs | `Advice.ExchangeRate` |
| `.ERROR` | `PERCENT` | Share Percent | `Advice.SharePercent` |
| `.NOTE` | `EMAIL` | Alamat Surel | `Advice.Email` |
| `.HASIL5` | `REMARKREAS` | Catatan Reasuradur | `Advice.Remark` |
| `.HASIL2` | `REMARKAPPROVE` | Catatan Persetujuan | `Advice.ApprovalNote` |
| `.AlasanQuotationStock` | `REMARKPIC` | Catatan PIC | `Advice.PICNote` |
| `.Status` | `LIMIT_XOL` | Batas Layer | `Advice.Limit` |
| `.SISI` | `STATUSAPPROVE` | Status Persetujuan | `Advice.ApprovalStatus` |
| `.CARI6` | `IDLAYER` | Kode Layer | `Advice.LayerID` |
| `.pyBPNotes` | `IDMASTER` | Kode Master XOL | `Advice.MasterID` |
| `.LastNoteBy` | `USERINPUT` | Penerbit | `Advice.InputBy` |
| `.pyCaseID` | `CAUSEOFLOSS` | Penyebab Kerugian | `Advice.CauseOfLoss` |
| `.source` | `REVISI` | Nomor Revisi | `Advice.Revision` |
| `.pyCountry` | `T_REINSURER.COUNTRY` | Negara Reasuradur | `Advice.Country` |
| `.pyEmailApprovalAllowed` | `MST_USER_TEKNIK.EMAIL` | Surel Penerbit | `Advice.InputByEmail` |

**Grid Approval XOL dan DATA MASTER XOL** — dari `GetDataXOLForKomiteApprove` dan
`GetDataMasterXOLForKomiteApprove`:

| Properti Pega | Artinya | Nama di kode |
|---|---|---|
| `.City` | Tahun XOL (grid Approval) / Kode Master (grid Master) | `ApprovalItem.Year` / `MasterXOL.ID` |
| `.CityID` | Penyebab Kerugian / Nama Master | `ApprovalItem.CauseOfLoss` / `MasterXOL.Name` |
| `.Type` | Tipe PLA/DLA / Operator Pengaju | `ApprovalItem.Type` / `MasterXOL.PIC` |
| `.NoteKasir` | Tanggal Insert / Kode Group Business | `ApprovalItem.LastInsertedAt` |
| `.Country` | Tahun XOL | `MasterXOL.Year` |
| `.CountryID` | Kurs | `MasterXOL.ExchangeRate` |

> Perhatikan `.City`, `.CityID`, `.Type`, dan `.NoteKasir`: keempatnya dipakai untuk **dua
> arti berbeda** pada dua grid di layar yang sama. Itu sebabnya nama properti tidak dapat
> dipakai sebagai acuan apa pun.

### Nama kueri `.sql`

`master_list` · `master_pending_committee` · `master_business_list` · `claim_summary` ·
`breakdown_business` · `breakdown_treaty_inward` · `advice_list_pla` · `advice_list_dla` ·
`approval_advice_queue` · `cause_of_loss_list`

### Nama yang sengaja TIDAK diterjemahkan

Judul kolom di layar dibiarkan seperti di Pega (`D-13`): "Date Of Loss", "Cause Of Loss",
"Group Business", "OS Value", "Accepted Value", "NO PLA / DLA", "Nama Insurance", "Nama
Layer", "Share Percent", "ID XOL", "Nama XOL", "Tahun XOL", "Kurs Value", "TIPE",
"Tanggal Insert", "Total Klaim", "Type Master", "ID Master".

Termasuk **judul yang menyesatkan**: kolom "Date Of Loss" pada grid Approval XOL berisi
tahun perjanjian, bukan tanggal kejadian. Judulnya dipertahankan; keterangannya dinyatakan
di bawah tabel, dan namanya di kode dibetulkan menjadi `Year`.

### Istilah domain baru

| Indonesia / Pega | Inggris di kode | Keterangan |
|---|---|---|
| Perjanjian XOL | `MasterXOL` | satu tahun, satu kurs, sekumpulan group business |
| Pemberitahuan PLA/DLA | `Advice`, `AdviceType` | `AdvicePLA`, `AdviceDLA` |
| Akumulasi klaim | `ClaimSummary` | per Tanggal Kejadian × Penyebab Kerugian |
| Rincian per group business | `BusinessBreakdown` | dua sumber: `SourceOwnBusiness`, `SourceTreatyInward` |
| Antrean persetujuan | `ApprovalItem`, `ApprovalQueue` | satu baris = sekumpulan pemberitahuan |
| Penyebab Kerugian | `CauseOfLoss` | yang tersimpan DESKRIPSI-nya, bukan kodenya |
| Kurs tidak tersedia | `RateMissing` | pengganti `RETURN 1` pada function kurs lama |
