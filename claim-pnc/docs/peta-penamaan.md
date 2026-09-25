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

## Tambahan 2026-09-19 — modul Inbox Auto Claim

Nama modulnya **Indonesia** (`inboxautoclaim` di backend, `inbox-auto-claim` di frontend)
mengikuti `D-81`: ia nama modul bisnis yang disebut Work Owner. Isinya **Inggris**.

### Istilah domain baru

| Indonesia | Inggris | Catatan |
|---|---|---|
| Batch | Batch | sudah Inggris; satu unggahan milik satu perusahaan |
| Baris klaim | Line | satu klaim di dalam sebuah batch |
| Perusahaan rekanan | Company | bank dan lembaga pembiayaan pengirim klaim borongan |
| Halaman | PageRequest · BatchPage · LinePage | permintaan halaman dan hasilnya |
| Penyaring | BatchFilter · LineQuery | |
| Hasil pemrosesan | Result | `berhasil` · `gagal` · kosong berarti belum |
| Baris unggahan | UploadRow | satu baris berkas CSV sesudah dibaca |
| Ringkasan unggahan | UploadResult · BatchRef | |
| Bentuk berkas ekspor | ExportSpec | judul kolom + cara memetakan barisnya |
| Berkas unduhan | ExportFile · DownloadedFile | Go dan TypeScript |

### Tambahan 2026-09-20 — tiga tab

| Indonesia | Inggris | Catatan |
|---|---|---|
| Tab / jenis klaim | **Source** | enum tertutup: `aneka` · `kredit` · `travel` |
| Keterangan tab | **SourceInfo** | label, nama tabel, nama kolom perusahaan |
| Ringkasan per perusahaan | **Summary · CompanySummary** | satuannya **jumlah batch** |

Tiga catatan penamaan yang sengaja:

1. **`Source`, bukan `Tab`.** Yang dipilih pengguna memang tampak sebagai tab, tetapi yang
   ditentukannya **sumber data** — tabel mana yang dibaca. Menamainya `Tab` mengikat nama
   domain pada bentuk tampilan, dan bentuk itu dapat berubah tanpa sumbernya berubah.
2. **Nilai enum-nya Indonesia** (`aneka`, `kredit`, `travel`) karena ia **kontrak API** —
   `?sumber=kredit` dipakai peramban dan tertulis di README, sama sifatnya dengan nama field
   JSON.
3. **Label tab tidak diturunkan dari nama rule.** Rule-nya `BrowseClaimSPKAutoClaim`, tetapi
   `pyCaption` harness menyebut **ANEKA**. Label datang dari harness, dan dikirim server —
   bukan diketik di layar.

### Penamaan ulang delapan kolom grid (`D-19`)

Alias Pega-nya **tidak dibawa**; ketiganya contoh utang teknis §4.2:

| Judul kolom layar | Properti Pega | Nama di kode | Kolom tabel |
|---|---|---|---|
| KODE | `.CaseID` | `CompanyCode` | `INISIALID` |
| Nama Perusahaan | `.AlasanTerlambat` | `CompanyName` | `NAMA_PENERIMA` |
| Batch | `.CauseOfLoss` | `BatchNumber` | `BATCH` |
| Jumlah data yang di upload | `.ChronologicalOfIncodent` | `Uploaded` | `COUNT(*)` |
| Jumlah data yang telah diproses | `.City` | `Processed` | turunan `TMP_MESSAGE` |
| Jumlah Berhasil | `.CityID` | `Succeeded` | turunan `TMP_MESSAGE` |
| Jumlah Gagal | `.ClaimID` | `Failed` | turunan `TMP_MESSAGE` |
| User Upload | `.AnaylstRemarks` | `UploadedBy` | `USERINPUT` |

> `.CauseOfLoss` pada grid adalah **nomor batch**, sementara kolom `COL_ID` pada tabel yang
> sama adalah **penyebab kerugian yang sungguhan** — dan itu yang dinamai `CauseOfLoss` di
> kode. Dua hal berbeda dengan satu nama Pega.

### Nama JSON tetap Indonesia

`kode_perusahaan`, `nama_perusahaan`, `batch`, `jumlah_upload`, `jumlah_proses`,
`jumlah_berhasil`, `jumlah_gagal`, `jumlah_belum_proses`, `user_upload`, `paginasi`,
`baris`, `hasil`, `keterangan`.

### Judul kolom berkas CSV — TIDAK diterjemahkan dan TIDAK diperbaiki

`Inisial`, `No Polis`, `No Klaim`, `No Ref Bank`, `No Aksep`, `Currency`, `Nilai Klaim`,
`No Objek`, `Keterangan`.

Beberapa di antaranya menyesatkan ("No Objek" berisi nomor produk), tetapi berkas ini
dibaca **perusahaan rekanan di luar Sinarmas**. Judulnya kontrak keluaran, bukan nama
internal — penamaan ulang berhenti di batas berkas.

### Judul kolom berkas UNGGAHAN — nama kolom basis data

`inisialid`, `nopolis`, `prodke`, `tglkejadian`, `tgllapor`, `col_id`, `nilaiklaim`,
`currency`, `note`, `keyword`.

Flow action Pega-nya hilang dari export, sehingga judul aslinya tidak diketahui. Yang
dipakai nama kolom tabel — satu-satunya nama yang dapat ditelusuri ke buktinya.

### Prop baru pada komponen bersama

| Prop | Komponen | Arti |
|---|---|---|
| `pagination` | `DataTable` | paginasi sisi server; opsional |
| `hideSearch` | `DataTable` | menyembunyikan kotak pencarian peramban; opsional |
| `unduhBerkas` · `simpanBerkas` | `api/client.ts` | mengambil respons non-JSON dan menyimpannya |
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
**Nama tabel baru** tetap Indonesia karena ia milik basis data (`D-80`):
`POOLDATA.CPNC_PEMAKAIAN_PROTEKSI`.

---

## Master Kategori Sparepart (2026-09-21)

Modul `masterkategorisparepart` — nama folder mengikuti nama modul bisnis yang disebut Work
Owner (`D-81`): backend `internal/masterkategorisparepart`, frontend
`src/modules/master-kategori-sparepart`.

Paket transportnya `masterkategorispareparthttp` — nama modul ditambah akhiran `http` tanpa
tanda hubung, karena Go tidak mengizinkannya. Pola yang sama dipakai `masterspareparthttp`
dan `masterpanelhttp`.

### Kolom POOLDATA.GCNM_M_SPAREPART_CATEGORY

Hanya **tiga kolom**, dan itu dipastikan dari kesembilan rule Pega yang menyentuh tabel ini —
bukan dari satu rule browse saja. Label layar dibaca dari `pyLabelFieldValue` pada
`Section/MasterKategoriSparepartHEApproval-Section.xml`.

| Kolom | Label layar Pega | Nama di kode (Inggris) | Nama JSON (Indonesia) |
|---|---|---|---|
| `PART_CATEGORY_ID` | ID Kategori Sparepart | `ID` | `id_kategori_sparepart` |
| `PART_CATEGORY_NAME` | Nama Kategori Sparepart | `Name` | `nama_kategori_sparepart` |
| `APPROVAL` | — | `Status` | `status` |

Satu field JSON yang **tidak punya kolom**: `status_label`. Ia diturunkan dari `status`
supaya layar tidak perlu menyimpan petanya sendiri.

### Alias kolom Pega yang TIDAK dibawa

Seluruh rule browse mengalias kolomnya menjadi nama yang tidak ada hubungannya dengan isinya
— bentuk utang yang `03-CURRENT-ARCHITECTURE.md` §4.2 catat:

| Kolom asli | Alias Pega | Dibawa? |
|---|---|---|
| `PART_CATEGORY_ID` | `"CityID"` | tidak |
| `PART_CATEGORY_NAME` | `"City"` | tidak |

Pada jalur SIMPAN, penyesatannya berbeda lagi dan lebih jauh —
`RDB List/UpdateMasterSparepartCategory_sql2-SQL.xml` memakai tiga property yang namanya
tidak satu pun cocok dengan apa yang dibawanya:

| Property Pega | Yang sebenarnya dibawanya |
|---|---|
| `InputKategori.CITY_ID` | **nama** kategori |
| `InputKategori.LOGIN_APLIKASI` | **status** persetujuan |
| `InputKategori.ACCOUNT_ID` | **kunci** baris |

Ketiganya disebut apa adanya di kode baru.

### Nama tipe: PartCategory, bukan Category

| Lapisan | Nama | Kenapa |
|---|---|---|
| Domain Go | `PartCategory` | `Category` terlalu umum untuk tipe yang menempati paket bernama modul, dan `mastersparepart` sudah memakai nama itu untuk DTO lookup-nya. `PartCategory` mengikuti awalan kolomnya sendiri, `PART_CATEGORY_*` |
| TypeScript | `PartCategory` | `SparepartCategory` sudah dipakai untuk bentuk BERBEDA — DTO dropdown `{kode, nama}` pada layar Master Sparepart. Dua bentuk untuk satu tabel, dan keduanya memang dibutuhkan |
| Komponen React | `PartCategoryPage`, `PartCategoryForm` | memakai nama **tipe domain**, bukan nama modul — sama seperti `AccountPage` pada `master-rekening` |

### Jalur API

| Operasi | Jalur |
|---|---|
| Daftar | `GET /api/master/kategori-sparepart?status=&cari=` |
| Satu baris | `GET /api/master/kategori-sparepart/{id}` |
| Tambah | `POST /api/master/kategori-sparepart` |
| Simpan | `PUT /api/master/kategori-sparepart/{id}` |
| Keputusan borongan | `POST /api/master/kategori-sparepart/keputusan` |

Tidak ada `DELETE`, dan tidak ada `/pilihan` — lihat `keputusan-implementasi.md` §35.6.

### Nama kueri pada berkas .sql

Seluruhnya berawalan `category_`, bukan `partcategory_` maupun `kategori_`: awalan itu hanya
dipakai di dalam satu berkas milik satu modul, dan awalan yang lebih pendek membuat
pernyataannya terbaca tanpa mengulang nama modul di setiap baris.

| Nama kueri | Padanan Pega |
|---|---|
| `category_list` | `BrowseMasterSparepartCategoryClaimHE` |
| `category_list_search` | — (ditambahkan) |
| `category_get` | `BrowseSparepartCategoryClaimHE_sql` |
| `category_find_by_name` | `ValidationSparepartCat` |
| `category_lock_table` | — (ditambahkan, menutup balapan `max+1`) |
| `category_next_id` | bagian `nvl(max(...),0)+1` pada `InsertMasterSparepartCategory_sql` |
| `category_insert` | `InsertMasterSparepartCategory_sql` |
| `category_update` | `UpdateMasterSparepartCategory_sql2` |
| `category_set_status` | **direkonstruksi** — `UpdateSparepartCategoryClaimHE_sql` hilang (`R-16`) |
| `category_count_by_status` | `CountMasterKatSparepartManager` |
| `category_count_all` · `category_check_table` · `category_count_unknown_status` · `category_count_duplicate_name` · `category_count_orphan_sparepart` | — (pemeriksaan, ditambahkan) |
## Tambahan 2026-09-20 — modul Inbox XOL (`inboxxol`)
## Tambahan 2026-09-20 — modul Inbox Komite

### Alias Pega yang TIDAK dibawa

Ini bagian terpenting dari peta modul ini. Lima property pada section lama bernama sesuatu yang
sama sekali tidak mencerminkan isinya — utang teknis `03-CURRENT-ARCHITECTURE.md` §4.2. Nama yang
dipakai di sini diturunkan dari **apa yang benar-benar dihitung SQL-nya**, bukan dari nama
property-nya (`D-19`).

| Property Pega | Caption di layar lama | Isi sebenarnya | Nama di kode |
|---|---|---|---|
| `.IBNR` | Nilai ASM Share | `NILAIKLAIM × SHAREASM / 100` | `ASMShareValue` |
| `.pyScore` | Nilai OR ASM | `NILAIKLAIM × Σ PRSN_*` | `ORValue` |
| `.DraftWordingID` | PIC Klaim | `T_CLAIM_PNC.PICTEKNIK` | `ClaimPIC` |
| `.RejectedCode` | Alasan Reject | `NOTEKOMITE` | `CommitteeNote` |
| `.StatusKlaim` | Tipe Komite | `TYPEKOMITE × PAYMENTTYPE` | `CommitteeKind` |

`.StatusKlaim` patut disebut khusus: ia **bukan** Status Klaim dalam arti `D-18`. Menyalin namanya
akan menambah tafsir kelima pada konsep yang `D-18` sudah susah payah pisahkan menjadi empat.

### Istilah domain baru

| Indonesia (`CONTEXT.md`) | Inggris | Contoh |
|---|---|---|
| Kasus komite | `CommitteeCase` | `CommitteeCase`, `FindCase`, `ListCases` |
| Kotak masuk | `InboxKind` | `InboxOutstanding`, `InboxAccepted`, `InboxRejected` |
| Keputusan | `Decision` | `Decision`, `DecisionKind`, `DecisionCommand` |
| Setuju · Tolak · Kembalikan | `Approve` · `Reject` · `Return` | `DecisionApprove`, `DecisionReject`, `DecisionReturn` |
| Kesimpulan | `Outcome` | `OutcomePending`, `OutcomeApproved`, `OutcomeRejected`, `OutcomeReturned` |
| Penjenjangan (keadaan) | `Progress` | `Progress`, `Evaluate`, `TierCountUnknown` |
| Umur menunggu | `Aging` | `AgingDays` |
| Pemutus | `Actor` | `Actor`, `ActorLogin`, `ActorName` |
| Tipe komite | `CommitteeKind` | `CommitteeKindOf` |
| Warisan (dari Pega) | `Legacy` | `LegacyOutcome`, `LegacyTier` |

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
| `validasi_gagal` | isian tidak lolos pemeriksaan — 422 |
| `kunci_kategori_sparepart_sudah_ada` | nama sudah dipakai baris lain — 409 |
| `status_tidak_dikenal` | status di luar `"0"`, `"1"`, `"2"` — 422 |
| `tidak_ditemukan` | baris tidak ada — 404 |
| `permintaan_cacat` | badan JSON tidak dapat dibaca — 400 |

Namanya `kunci_kategori_sparepart_sudah_ada`, bukan `nama_...`, supaya sebentuk dengan
`kunci_sparepart_sudah_ada` pada Master Sparepart — keduanya menyatakan kunci alami yang
bentrok, dan kebetulan modul ini hanya punya satu.

## Master Grouping Sparepart (2026-09-21)

Modul dengan **pemetaan nama paling menyesatkan** sejauh ini. Layarnya salinan layar Master
Sparepart — `Harness/GroupingSparePart_HE-Harness.xml` menyebut asalnya sendiri lewat
`pzOriginalInstanceKey = RULE-HTML-HARNESS DATA-PORTAL SPAREPART_HE` — sehingga kolom grouping
dipetakan ke properti klipboard milik modul lain.
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

---

## Modul Daftar Objek Dokumen (2026-09-23)

| | |
|---|---|
| Kunci menu | `ListDocumentObject` |
| MENU_ID | 43, di bawah kelompok MASTER |
| Rute layar | `/master/objek-dokumen` |
| Rute API | `/api/master/objek-dokumen` |
| Paket Go | `daftarobjekdokumen` |
| Folder frontend | `daftar-objek-dokumen` |

### Apa yang dimodelkan, dan bedanya dengan tetangganya

Objek Dokumen menyatakan sebuah dokumen **melekat pada APA**. Ia mudah tertukar dengan Tipe Dokumen,
yang menyatakan dokumen itu **JENISNYA apa** — dan keduanya dirujuk bersamaan oleh satu tabel yang
sama:

```
POOLDATA.LST_TYPE_DOC_BUSINESS
  ├─ DOCUMENT_TYPE_ID  ──► LST_DOC_TYPE   (MENU_ID 40, Daftar Tipe Dokumen)
  └─ OBJECT_DOC_ID     ──► LST_DOC_OBJ    (MENU_ID 43, modul ini)
```

Terbaca berdampingan di `Database/PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:26`. Itulah sebab `ID` sebuah
objek dokumen tidak pernah berubah dan barisnya tidak pernah dihapus.

### Pemetaan nama

| Kolom basis data | Field JSON | Nama Go | Label layar |
|---|---|---|---|
| `ID` | `id` | `ID` | "ID" |
| `KET_DOC_OBJ` | `objek_dokumen` | `Description` | **"Daftar Objek Dokumen"** |
| `OLD_ID` | `id_lama` | `OldID` | *(tidak ditampilkan)* |
| `BISNISID` | `bisnis[].id` | `Business.ID` | *(tersembunyi)* |
| `NOTE` | `bisnis[].nama` | `Business.Name` | "ID Bisnis" |

**`objek_dokumen`, bukan `keterangan`.** Field JSON mengikuti label layar supaya namanya
mencerminkan isinya. Kata "Daftar" di depan label itu milik layarnya — ia daftar objek dokumen —
sedangkan nilai satu barisnya adalah satu objek dokumen.

**`Description` di Go, bukan `ObjectDocument`.** Nama internal berbahasa Inggris (`D-80`), dan yang
dinamai adalah perannya di dalam baris: ia keterangan barisnya. Nama modulnya sendiri tetap
Indonesia (`D-81`).

### Objek basis data — mana yang pasti, mana yang dugaan

| Objek | Status | Sumber |
|---|---|---|
| `POOLDATA.V_LST_DOC_OBJ` | **pasti** | `Report Definition/BrowseVLstDocObj_RD-RD.xml` |
| `POOLDATA.M_SITE_DATABASE` | **pasti** | `Database/PEGA_LST_DOC_TYPE.prc:11` |
| `POOLDATA.BUSINESS` | **pasti** | `RDB List/GetLBUID_SQL-SQL.xml` |
| `POOLDATA.LST_DOC_OBJ` | **DUGAAN** | diturunkan dari nama view-nya |
| `POOLDATA.SET_LST_DOC_OBJ` | **DUGAAN** | diturunkan dari `SET_LST_DOC_TYPE` |
| `POOLDATA.LST_DOC_OBJ_BUSINESS` | **DUGAAN** | tabel baru, dirancang di migrasi 0010 |

Ketiga dugaan diisolasi di `repo/sqlstore/daftarobjekdokumen.sql`. Tidak ada nama tabel yang tercecer
di dalam kode Go.

### Bentuk ID — empat digit, bukan tiga

```
ID = M_SITE_DATABASE.ID || lpad(SET_LST_DOC_OBJ.nextval, 4, '0')
```

Mengikuti `Database/PEGA_LST_DOC_TYPE.prc:21`, procedure tabel bersaudara di rumpun `LST_*`.

**Rumpun `M_CAUSE_OF_LOSS` memakai TIGA digit.** Lebar nomor urut harus dibaca dari procedure
rumpunnya masing-masing, tidak pernah disalin dari modul tetangga. Menyalinnya menghasilkan ID
berbentuk salah yang tidak terlihat sampai baris pertama disimpan ke Oracle.

Pemadatan nolnya dikerjakan **di Go**, bukan dengan `LPAD` — `LPAD` termasuk yang dilarang
`09-DATABASE-STRATEGY.md` §4.

### Kode galat

| Kode | Keadaan |
|---|---|
| `validasi_gagal` | isian melanggar batas panjang; 422 |
| `tidak_ditemukan` | baris yang diminta tidak ada; 404 |
| `permintaan_cacat` | badan JSON tidak dapat dibaca, atau memuat field tak dikenal; 400 |

Ketiganya kode UMUM yang sudah dikenali klien, bukan kode baru milik modul ini. Kode baru untuk arti
yang sama hanya menambah cabang di frontend tanpa menambah keterangan apa pun.

### Teks layar — mengikuti Pega apa adanya

| Tempat | Teks | Sumber |
|---|---|---|
| Judul layar | "Daftar Objek Dokumen" | `Section/BrowseDocumentObject-Section.xml:6715` |
| Header kolom | "ID", "Daftar Objek Dokumen" | section yang sama |
| Judul form | "Memperbaharui Data" — **termasuk pada mode tambah** | `:10859` |
| Label isian | "Daftar Objek Dokumen" | `:11801` |
| Judul grid dalam form | "ID Bisnis" | `:14180` |
| Aksi per baris | "Ubah" | `pyButtonLabel` pada section |
| Tombol | "Tambah", "Refresh", "Simpan", "Ubah", "Batal" | kedua section |

Teks yang **bukan** dari Pega, ditulis sebagai tambahan dalam bahasa Indonesia: kalimat pengantar di
bawah judul layar, keterangan *"Belum ada bisnis yang dipilih. Objek dokumen ini tetap dapat
disimpan."* — yang ada karena grid kosong di layar lama tidak menjelaskan apa artinya kosong — dan
pesan saat daftar bisnis gagal dimuat.

Satu isian Pega **tidak dibawa**: `TempDocObj.pyNote` berlabel "Catatan" (`:3097`). Ia kotak
read-only tempat Pega menaruh kalimat yang dikembalikan procedure lewat `ErrMsg`; `D-68` menetapkan
kontrak galat berbasis teks itu tidak dibawa.

---

## Daftar Tipe Dokumen Bisnis — MENU_ID 42 (2026-09-23)

Nama modul: **`daftartipedokumenbisnis`** (backend), **`daftar-tipe-dokumen-bisnis`** (frontend),
mengikuti MENU_DESC "Daftar Tipe Dokumen Bisnis" (`D-81`). Tipe domainnya `DocumentRule`, sehingga
berkas frontend bernama `BusinessDocumentRule*.tsx` — **nama tipe, bukan nama modul** (`D-80`).

### Kolom basis data → nama di kode

`POOLDATA.LST_TYPE_DOC_BUSINESS`:

| Kolom | Go | JSON | Keterangan |
|---|---|---|---|
| `ID` | `ID` | `id` | `kode_situs` + 4 digit |
| `BUSINESSID` | `BusinessID` | `id_bisnis` | **tidak dapat diubah** setelah baris dibuat |
| `DOCUMENT_TYPE_ID` | `DocumentTypeID` | `id_tipe_dokumen` | → `V_LST_DOC_TYPE` |
| `OBJECT_DOC_ID` | `ObjectDocID` | `id_object_dokumen` | → `V_LST_DOC_OBJ`; boleh NULL |
| `DOC_TYPE_DT_ID` | `DetailTypeDocID` | `id_detail_dokumen` | → `V_LST_DET_TYPE_DOC` |
| `DETAIL_DOKUMEN` | `DetailDocument` | `detail_dokumen` | `"-"` menyembunyikan baris |
| `STS_WAJIB` | `Mandatory` | `status_wajib` | **teks** `'1'`/`'0'`, bukan angka |
| `MIN_DOC` | `MinDocument` | `minimum_dokumen` | |
| `EDIT_DATE` | `Editor.At` | — | jejak simpan, tidak dikirim klien |
| `USER_EDIT` | `Editor.Identity` | — | jejak simpan, tidak dikirim klien |
| `FLAGTYPES` | — | — | **tidak ditulis**; menentukan urutan di layar unggah |
| `CREDENTIAL`, `DURATION` | — | — | **tidak ditulis**; dibaca `BrowseRegisterCvg` |

`POOLDATA.COVERAGE_DOC_BUSINESS`:

| Kolom | Go | JSON | Keterangan |
|---|---|---|---|
| `ID` | — | — | rujukan ke baris aturan, bukan kunci sendiri |
| `BUSINESSID` | — | — | diturunkan dari baris aturannya |
| `COVERAGEID` | `Coverage.ID` | anggota `jenis_klaim` | |
| `NOKLAIM` | — | — | **tidak pernah ditulis**; milik jalur klaim, disaring `IS NULL` |

Nama hasil join yang **tidak tersimpan** di baris: `nama_bisnis` (`BUSINESS.NOTE`), `tipe_dokumen`
(`V_LST_DOC_TYPE.TYPE_DOCUMENT`), `object_dokumen` (`V_LST_DOC_OBJ.KET_DOC_OBJ`).

### Sebelas alias menyesatkan yang DIBUANG

`Select_TYPE_DOCUMENT` memaksakan kolomnya masuk kelas Pega generik `T_GENERAL`, sehingga nama
aliasnya tidak ada hubungannya dengan isinya. Pemetaannya, untuk siapa pun yang membandingkan kueri
lama dan penggantinya:

| Alias lama | Isinya sebenarnya |
|---|---|
| `IDPEGA` | `LST_TYPE_DOC_BUSINESS.ID` |
| `BUSINESSCODE` | `BUSINESS.ID` |
| `BUSINESSNAME` | `BUSINESS.NOTE` |
| `ACCUMCODE` | `V_LST_DOC_TYPE.ID` |
| `BRANCHNAME` | `V_LST_DOC_TYPE.TYPE_DOCUMENT` — tahap klaim |
| `BUSINESSTYPE` | `OBJECT_DOC_ID` |
| `CLIENTID` | `V_LST_DET_TYPE_DOC.ID` |
| `EDMNO` | `V_LST_DET_TYPE_DOC.DETAIL_DOCUMENT` |
| `EDMTYPE` | `STS_WAJIB` |
| `FOLLOWEDPOLICY` | `MIN_DOC` |
| `MARKETINGCODE` | `USER_EDIT` |

Ditambah satu di `BrowseLSTDetailDocument_sql`: `b.note AS DOCUMENT_TYPE_ID` — **nama bisnis**
dialias sebagai kode tipe dokumen.

### Kode galat

| Kode | Kapan |
|---|---|
| `tipe_dokumen_bisnis_tidak_ditemukan` | baris yang dituju tidak ada |
| `nama_bisnis_belum_diisi` | tidak satu pun lini bisnis dipilih — **ditiru dari layar lama** |
| `permintaan_cacat` | badan JSON tidak dapat dibaca, atau memuat field tak dikenal |

Hanya **satu** kode isian, dan itu cerminan layar lamanya: `nama_bisnis_belum_diisi` meniru
satu-satunya validasi yang benar-benar ada di sana
(`Activity/InsertDetailTypeDocumentBusiness_act-Act.xml:209`), beserta ejaan kalimatnya.

> **Dikoreksi 2026-09-23.** Daftar ini semula memuat dua kode lagi —
> `baris_dokumen_belum_diisi` dan `jenis_klaim_belum_dipilih`. Keduanya penambahan saya sendiri
> dengan alasan mengganti kegagalan senyap menjadi kegagalan yang terbaca, dan **dicabut** atas
> keputusan Work Owner ("seperti aplikasi Pega saja"): di Pega kedua keadaan itu dilewati
> diam-diam, bukan ditolak. Lihat `catatan-pengembangan.md` §23.12.

### Teks layar — mengikuti Pega apa adanya

| Tempat | Teks | Sumber |
|---|---|---|
| Judul layar | "Detail Tipe Dokumen Bisnis" | `Section/DetTypeDocumenBisnis_Portal-Section.xml` |
| Header kolom grid kedua | "Tipe Dokumen", "Object Dokumen", "Detail Dokumen", "Status Wajib", "Minimum Dokumen" | `Section/InputListDetailTypeDocumentBusiness_sect-Section.xml` |
| Header kolom grid pertama | "ID", "Nama Bisnis", "Aksi" | `Section/BrowseListDetailTypeDocumentBusiness_sect-Section.xml` |
| Judul form ubah | "Update Data" | kedua section |
| Tombol | "Tambah", "Refresh", "Simpan", "Batal", "Detail" | kedua section |
| Pesan validasi | "Nama Bisnis belum di isi" — **termasuk ejaan terpisahnya** | `Activity/InsertDetailTypeDocumentBusiness_act-Act.xml:209` |
| Label jaminan | "Jenis Klaim" | `InputListDetailTypeDocumentBusiness_sect` |

Judul layar **sengaja berbeda** dari `MENU_DESC` di tabel menu yang berbunyi "Daftar Tipe Dokumen
Bisnis". Keduanya ditiru di tempatnya masing-masing: yang di menu dibaca dari basis data, yang di
layar dari section Pega (`D-13`). Ketidakcocokan itu ada di sistem lama, bukan diperkenalkan di
sini.

Ejaan **"Object Dokumen"** dengan "c" dipertahankan pada label yang dilihat pengguna, sementara
jalur API-nya memakai `objek` mengikuti modul yang memiliki tabelnya. Keduanya disengaja.

Teks yang **bukan** dari Pega, ditulis sebagai tambahan: judul form tambah **"Tambah Data"** (Pega
memakai judul yang sama untuk keduanya), peringatan **"Wajib di sini belum berarti wajib di
klaim"**, keterangan **"— disembunyikan"** pada baris ber-`DETAIL_DOKUMEN` `"-"`, kalimat jumlah
baris yang akan lahir pada form tambah, dan kalimat *"Jenis klaim hanya dapat ditambahkan, tidak
dapat dibuang"*. Kelimanya menjelaskan aturan yang **sudah ada** tetapi tidak terlihat di layar
lama.

### Konfigurasi

| Variabel | Isi |
|---|---|
| `BISNIS_DIKECUALIKAN_PILIH_SEMUA` | Kode lini bisnis yang dilewati tombol "Pilih semua", dipisah koma. Bawaannya `10028,10164,10114,10084,10093` — kelima kode lini MBU yang di Pega ditulis langsung di dalam rule (`Activity/SetAllBusiness-Act.xml:984`) |

Nama variabelnya berbahasa Indonesia mengikuti `D-80`: variabel lingkungan sifatnya **kontrak**
dengan berkas `.env` dan skrip deployment, bukan nama internal.

Daftarnya ditaruh di konfigurasi, bukan di master data, karena masternya belum ada — keadaan yang
sama dengan `XOL_PENERIMA_KOMITE`. Ia sudah memenuhi `D-15` ("dapat diubah tanpa deploy"), tetapi
**belum** dapat diubah pengguna bisnis sendiri.

---

## Daftar Detail Tipe Dokumen — MENU_ID 41 (2026-09-23)

Nama modulnya **`daftardetailtipedokumen`** di backend dan **`daftar-detail-tipe-dokumen`** di
frontend, mengikuti MENU_DESC "Daftar Detail Tipe Dokumen" (`D-81`). Tipe domainnya `DetailType`,
sehingga nama modul (Indonesia) dan nama tipe (Inggris) memang berbeda — itu yang dikehendaki.

**Jalur API-nya `/master/detail-tipe-dokumen`, lebih pendek dari nama modulnya.** Sebabnya:
hubungan ketiga master dokumen harus terbaca dari URL-nya.

    /master/tipe-dokumen           MENU_ID 40   induk
    /master/detail-tipe-dokumen    MENU_ID 41   ini
    /master/objek-dokumen          MENU_ID 43   master yang dirujuknya

### Kolom basis data ke field domain

| Kolom | Field JSON API | Field Go | Label layar Pega |
|---|---|---|---|
| `ID` | `id` | `ID` | "ID" |
| `DOC_TYPE_ID` | `id_tipe_dokumen` | `DocumentTypeID` | **"ID Tipe Dokumen"** |
| `TYPE_DOCUMENT` | `nama_tipe_dokumen` | `DocumentTypeName` | (tidak ada — hasil join) |
| `DETAIL_DOCUMENT` | `detail_dokumen` | `Detail` | **"Detail Dokumen"** |
| `STS_INSURED` | `status_tertanggung` | `InsuredStatus` | **"Status Tertanggung"** |
| `DOC_COL_ID` | `id_penyebab_kerugian` | `CauseOfLossID` | (tidak ada — target tersembunyi) |
| `DOC_COL_INFO` | `keterangan_penyebab_kerugian` | `CauseOfLossDescription` | **"Dokumen kolom ID"** |
| `OBJ_DOC` | `id_objek_dokumen` | `ObjectDocumentID` | (tidak ada — target tersembunyi) |
| `OBJ_DOC_DESC` | `keterangan_objek_dokumen` | `ObjectDocumentDescription` | **"Objek Dokumen"** |
| `RISK` | `resiko` | `Risk` | **"Resiko"** |
| `DFT_BISNIS_ID` | `id_bisnis` | `BusinessID` | **"ID Bisnis"** |
| `STS_WAJIB` | `status_wajib` | `Mandatory` | **"Status Wajib"** |
| `MIN_DOC` | `minimum_dokumen` | `MinDocument` | **"Minimum Dokumen"** |

Perhatikan **`DOC_COL_ID` → `CauseOfLossID`**: nama kolomnya menyebut "kolom dokumen", sementara
isinya rujukan ke `M_CAUSE_OF_LOSS.M_COL_ID` — penyebab kerugian. Nama Go mengikuti **isinya**,
bukan nama kolomnya, persis yang `D-19` perintahkan.

Perhatikan pula ke mana label **"Dokumen kolom ID"** melekat: bukan ke `DOC_COL_ID` melainkan ke
`DOC_COL_INFO`. Labelnya menyebut "ID" padahal isiannya **keterangan** — dan kodenya justru tidak
punya label sama sekali karena ia target tersembunyi autocomplete. Label itu ditiru apa adanya
(`D-13`); yang tidak ditiru adalah kekeliruannya. Hal yang sama berlaku pada "Objek Dokumen", yang
melekat ke `OBJ_DOC_DESC`.

Hal yang sama berlaku pada **`DFT_BISNIS_ID`**: awalan `DFT_` tidak berarti apa pun bagi
pembacanya, dan nama Go-nya cukup `BusinessID`.

### Teks yang dilihat pengguna

| Tempat | Teks | Sumber |
|---|---|---|
| Judul layar | "Detail Tipe Dokumen" | `Section/DetTypeDocument-Section.xml` (`pyCaption Detail Tipe Dokumen`) |
| Tombol | "Tambah", "Refresh" | section yang sama |
| Tombol form | "Simpan", "Ubah" | `Section/BrowseListDetailTypeDocument-Section.xml` |
| Header kolom grid | "ID", "Tipe Dokumen", "Detail Dokumen" | section yang sama |
| Label isian | ketujuhnya pada tabel di atas | section yang sama, elemen `pyLabelPreview` |
| Pilihan Status Wajib | "Ya", "Tidak" | nilai yang **tersimpan**, bukan hanya tampilan |

Teks yang **bukan** dari Pega, ditulis sebagai tambahan:

- Kalimat penjelas di bawah judul layar.
- Dua header kolom grid tambahan: **"Objek Dokumen"** dan **"Penyebab Kerugian"** — grid Pega tidak
  memuat keduanya, dan alasannya ada di `keputusan-implementasi.md` §25.8.
- Keterangan di bawah isian rujukan: **"Tipe dokumen: …"**, **"Penyebab kerugian: …"**, **"Objek
  dokumen: …"** — menggantikan peran daftar autocomplete Pega setelah pilihannya ditutup.
- Kalimat **"Kosong dibaca sebagai nol, sama seperti di layar lama"** pada isian Resiko.
- Kalimat **"Belum ada lini bisnis…"** pada grid yang kosong.
- Peringatan **"Sebagian daftar pilihan tidak dapat dimuat"**, beserta nama master yang hilang.

Keenamnya menjelaskan aturan yang **sudah ada** tetapi tidak terlihat di layar lama.

### Nama yang sengaja TIDAK dipakai

| Tidak dipakai | Dipakai | Alasan |
|---|---|---|
| `DetailTypeDoc`, `DetTypeDoc` | `DetailType` | Singkatan Pega tidak dibawa (`D-19`) |
| `DocColID` | `CauseOfLossID` | Nama mengikuti isinya, bukan nama kolomnya |
| `DftBisnisID` | `BusinessID` | Awalan `DFT_` tidak bermakna bagi pembacanya |
| `Wajib`, `StatusWajib` | `Mandatory` | Identifier di dalam kode berbahasa Inggris (`D-80`) |
| `/master/daftar-detail-tipe-dokumen` | `/master/detail-tipe-dokumen` | Hubungan ketiga master terbaca dari URL-nya |

---

## Inbox Compliance — MENU_ID 47 (2026-09-24)

Nama modulnya **Indonesia** (`D-81`), isinya **Inggris** (`D-80`).

| Lapisan | Nama |
|---|---|
| Paket Go | `inboxcompliance` — huruf kecil, tanpa tanda hubung |
| Folder backend | `internal/inboxcompliance/` |
| Folder frontend | `src/modules/inbox-compliance/` |
| Rute layar | `/inbox-compliance` |
| Jalur API | `/api/inbox-compliance`, `/api/inbox-compliance/tab` |

Nama modulnya diambil dari `MENU_DESC` pada `Database/m_menu_aplikasi_pnc.csv` — "Inbox
Compliance" — bukan dari nama harness-nya. Kebetulan keduanya mirip di sini, berbeda dari beberapa
modul lain yang nama programnya tidak menyebut isinya sama sekali.

### Tipe dan fungsi

| Pega | Kode ini | Catatan |
|---|---|---|
| — | `WorkItem` | satu baris antrean |
| — | `Tab`, `Column` | bentuk grid, ditetapkan server |
| — | `Query`, `QueryInput` | permintaan isi satu tab |
| — | `Pagination`, `Page` | paginasi server-side |
| `GETSELISIHJAM` | `AgingHoursBetween` | hitungan jam, potong akhir pekan |
| `GetSelisihJam_sql` | `FormatAging` | pemformatan teks Aging |
| `weekends2` | `weekendDaysWIB` | **tafsir**; source aslinya tidak dikirim (`R-01`) |
| `Param.Operator` | `WorkbasketCompliance` | konstanta bernilai `CompliancePNC` |

### Properti Pega ke field Go ke field JSON

Kolom tab **Compliance**, berurutan seperti di grid:

| Properti Report Definition | Kolom Oracle | Field Go | Field JSON | Judul layar |
|---|---|---|---|---|
| `.pyID` | `A.PYID` | `CaseID` | `nomor_case` | Nomor Case |
| `.pzInsKey` | `A.PZINSKEY` | `Reference` | `referensi` | *(tidak digambar)* |
| `.Policy.PolicyNo` | `A.POLICYNO` | `PolicyNumber` | `no_polis` | No Polis |
| `.Policy.QQName` | `A.QQNAME` | `InsuredName` | `nama_tertanggung` | Nama Tertanggung |
| `.Policy.Quotation.BusinessName` | `A.BUSINESSNAME` | `BusinessName` | `nama_bisnis` | Nama Bisnis |
| `.Policy.Quotation.BranchName` | `A.BRANCHNAME` | `BranchName` | `nama_cabang` | Nama Cabang |
| `.pyOrigUserID` | `A.PYORIGUSERID` | `AdminName` | `nama_admin` | Nama Admin |
| `.ClaimData.TanggalBuatCompliance` | `p.COMPLIANCE_CREATEDATE` | `ComplianceSentDate` | `tanggal_kirim_compliance` | Tanggal Kirim Compliance |
| `.ClaimData.AgingKlaim` | *(dihitung di Go)* | `AgingHours` | `aging`, `aging_jam` | Aging |

Kolom tab **Post Audit**. Sumbernya `POOLDATA.T_CLAIM_COMPLIANCE_H` — tabel datar yang DDL-nya
diterima 2026-09-24, **bukan** tabel DATAPEGA seperti yang dibaca Report Definition lama:

Ketujuh kolomnya, berurutan seperti di layar Pega:

| # | Judul layar | Properti RD | Kolom Oracle | Field Go | Field JSON |
|---|---|---|---|---|---|
| 1 | Nomor Case | `.pyID` | `h.CASEID` | `CaseID` | `nomor_case` |
| 2 | No Klaim | `.pxCoverInsKey` | `h.NO_KLAIM` | `ClaimNumber` | `no_klaim` |
| 3 | Nama Tertanggung | `subr.Nama_Tertanggung` | `h.NAMA_TERTANGGUNG` | `InsuredName` | `nama_tertanggung` |
| 4 | No Polis | `subr.No_Polis` | `h.NO_POLIS` | `PolicyNumber` | `no_polis` |
| 5 | Catatan | `.ComplianceRemarks` | `h.REMARKS` | `ComplianceRemarks` | `catatan_compliance` |
| 6 | Tanggal Kirim Audit Compliance | `.TanggalKirimPostAudit` | `h.TGL_KIRIM_POST_AUDIT` | `PostAuditSentDate` | `tanggal_kirim_post_audit` |
| 7 | OutStanding | `.pyNote` | *(dihitung di Go)* | `Outstanding` | `outstanding` |

**Koreksi 2026-09-24.** Versi pertama tabel ini memetakan `NO_KLAIM` ke "Case ID" dan
menyembunyikan `CASEID` — kebalikan dari yang benar — serta hanya memuat lima kolom dengan judul
dari Report Definition. Tangkapan layar Pega yang berjalan membetulkan keduanya:

```
Nomor Case : CPL-19
No Klaim   : ASM-FW-GCNMFW-WORK PNC-2114
```

Jadi **`CASEID` berisi nomor kasus Work-Compliance** dan **`NO_KLAIM` justru berisi kunci teknis
Pega**. Kolom berjudul "No Klaim" tidak memuat nomor klaim — nama yang menyesatkan, ditiru apa
adanya sesuai `D-13`.

Judul kolomnya diambil dari `pyCaption` pada section, **bukan** dari label Report Definition.
Keduanya berbeda, dan yang dibaca pengguna adalah yang pertama.

Tiga properti Report Definition lama **tidak punya padanan** di tabel ini, dan ketiadaannya nyata —
bukan kelalaian pemetaan: `.pyStatusWork`, `.pxCreateDateTime`, dan `.pyOrigUserID`. Akibatnya ada
di `keputusan-implementasi.md` §27 dan §28.

### Dua nama yang mudah tertukar

| Nama | Artinya di sini | Yang mirip, tetapi BUKAN ini |
|---|---|---|
| `.ComplianceRemarks` | catatan petugas Compliance | `"ComplianceRemark"` (tanpa `s`) pada `RDB List/BrowseClaimStudy-SQL.xml:85` — alias untuk `SUM(TOTAL_CLAIM * CURRENCYVALUE)`, sama sekali bukan catatan |
| `CARI1`, `CARI2` | dua argumen **tanggal** untuk `GETSELISIHJAM` | Di modul lain nama yang sama berarti isi kotak pencarian. Layar ini tidak punya kotak pencarian |

### Nama kode tab

Berbeda dari modul Inbox Admin yang mempertahankan angka Pega (`3`, `7`, `9`, `11`), kode tab di
sini berupa kata: `compliance` dan `post-audit`. Alasannya bukan selera — layar ini **tidak punya
properti pemilih tab** di Pega, sehingga tidak ada angka yang perlu dipertahankan sebagai jalan
telusur balik.

### Jalur tulis — pengiriman ke Post Audit (2026-09-24)

| Konsep | Kode ini | Kontrak API | Kolom Oracle |
|---|---|---|---|
| Permintaan kirim | `PostAuditInput` | `{ referensi, catatan }` | — |
| Baris yang ditulis | `PostAuditEntry` | — | keenam kolom `T_CLAIM_COMPLIANCE_H` |
| Nomor terbit | `BuildCaseID` | `nomor_case` | `CASEID` |
| Penerbit nomor | — | — | `POOLDATA.CPNC_POST_AUDIT_SEQ` |

Bentuk nomornya `CPL-100001` — **meniru bentuk Pega** atas keputusan Work Owner, bukan tiga segmen
bertitik seperti `PNCN.YY.xxxx` (`D-71`) dan `LPK.YY.xxxx` (Pelaporan Klaim). Pemisahan dari
terbitan Pega dilakukan lewat **rentang angka**, bukan lewat bentuk.

Rute: `POST /api/inbox-compliance/post-audit`. Ia satu-satunya rute modul ini yang mengubah data.
| Putuskan | `Decide` | aksi bisnis, bukan `Update` — ia punya invarian dan meninggalkan jejak |
| Catat | `Record` | append-only; sengaja BUKAN `Save`, yang menyiratkan dapat menimpa |
| Ringkas | `Summarize` | jumlah per kotak dalam satu perjalanan |
| Tumpangkan | `withProgress` | menumpangkan keputusan kita di atas kasus warisan |

### Nama kueri `.sql` tambahan

Berawalan menurut **apa yang dilayaninya**, bukan menurut nama tabelnya — karena satu kueri di
sini menyentuh enam tabel sekaligus:

    inbox_list · inbox_count · inbox_summary · inbox_get · inbox_check_table
    decision_list_for_cases · decision_insert · decision_check_table

`decision_insert` adalah **satu-satunya pernyataan tulis di seluruh paket**, dan ia terdaftar
eksplisit di `kueriYangBolehMenulis` pada ujinya.

### Nama tabel dan kolom — tetap Indonesia

`POOLDATA.CPNC_KOMITE_KEPUTUSAN` beserta seluruh kolomnya (`CASE_ID`, `NOMOR_KLAIM`, `JENJANG`,
`KEPUTUSAN`, `CATATAN`, `ACTOR_LOGIN`, `ACTOR_NAMA`, `PADA`) berbahasa Indonesia mengikuti `D-80`:
nama basis data dimiliki bersama Pega selama masa paralel, dan perubahannya menempuh `D-63`.

Dua kolom memakai awalan `ACTOR_` yang berbahasa Inggris. Itu disengaja: ia menghindari kata
"pengguna", yang di tabel ini akan menyesatkan — pemutusnya belum tentu ada di tabel pengguna
aplikasi ini, dan nilainya adalah login warisan.

### Nama field JSON — tetap Indonesia

`nomor_case`, `nomor_klaim`, `aging_komite`, `tipe_komite`, `nilai_asm_share`, `nilai_or_asm`,
`penjenjangan`, `kesimpulan`, `keputusan`, `catatan`, `kotak`, `ringkasan` — seluruhnya kontrak
API, bukan nama internal (`D-80`).

Nilai enumnya pun Indonesia dan sengaja sama dengan yang tersimpan di kolom `KEPUTUSAN`:
`setuju` · `tolak` · `kembalikan`, dan `outstanding` · `diterima` · `ditolak`.

### Nama modul

| Lapisan | Nama |
|---|---|
| Backend, folder dan paket | `internal/mastergroupingsparepart` |
| Frontend, folder | `src/modules/master-grouping-sparepart` |
| Rute layar | `/master/grouping-sparepart` |
| Jalur API | `/api/master/grouping-sparepart` |
| MENU_PROGRAM | `GroupingSparePart_HE` (MENU_ID 32) |

Nama modulnya berbahasa Indonesia mengikuti `D-81`; isinya berbahasa Inggris mengikuti `D-80`.

### Properti Pega → arti sebenarnya → nama di kode

Inilah inti utang penamaan modul ini. Kolom kiri adalah nama properti yang dipakai layar Pega;
tidak satu pun ada hubungannya dengan isinya.

| Properti Pega | Kolom basis data | Label layar | Domain (Go) | Kontrak (JSON) |
|---|---|---|---|---|
| `TempSparepart.ID` | `ID` | ID | `ID` | `id_grouping` |
| `TempSparepart.NO_SPART` | `NO_PART` | Nomor Sparepart | `PartNumber` | `nomor_sparepart` |
| `TempSparepart.NAMA_SPART` | `NAMA_PART` | Nama Sparepart | `PartName` | `nama_sparepart` |
| `TempSparepart.KATEGORI_SPART` | `KATEGORI_SPART` | Kategory Sparepart | `CategoryID` | `kategori_sparepart` |
| `TempSparepart.TIPE_SPART` | `TIPE_SPART` | Type Sparepart | `TypeID` | `tipe_sparepart` |
| `TempSparepart.KODE_SPART` | `KODE_PART` | *(tidak digambar)* | `PartCode` | `kode_sparepart` |
| `TempSparepart.PROD_DATE` | `PROD_DATE` | *(tidak digambar)* | `ProductionDate` | `tanggal_produksi` |
| **`TempSparepart.PANJANG`** | `NAMA_PANEL` | **Nama Panel** | `PanelName` | `nama_panel` |
| **`TempSparepart.MIN_STOCK`** | `ID_PANEL` | *(tersembunyi)* | `PanelID` | `id_panel` |
| **`TempSparepart.LEBAR`** | `SISI_PANEL` | **Sisi** | `PanelSide` | `sisi` |
| **`TempSparepart.TINGGI`** | `B.NO_RANGKA` | **No Rangka** | `ChassisNumber` | `no_rangka` |
| **`TempSparepart.QTY_PESAN`** | `B.TIPE` | **Tipe Kendaraan** | `VehicleType` | `tipe_kendaraan` |
| **`TempSparepart.BERAT`** | `GROUPING_DGN_RANGKA` | **Grouping Dengan No Rangka** | `GroupWithChassis` | `grouping_dengan_no_rangka` |
| **`TempSparepart.pyID`** | `NO_GROUP_RANGKA` | *(tidak digambar)* | `GroupNumber` | `nomor_grup` |
| **`TempSparepart.MAX_STOCK`** | `CATATAN` | **Catatan** | `Note` | `catatan` |
| — | `APPROVAL` | *(tab)* | `Status` | `status` |

Tujuh baris bertanda tebal adalah properti yang namanya **sama sekali tidak mencerminkan
isinya**. Nama panel benar-benar tersimpan di properti bernama `PANJANG`, dan catatan di
properti bernama `MAX_STOCK`.

`pyID` patut diperhatikan sendiri: ia properti **bawaan Pega** yang dipinjam untuk memikul
nomor grup kendaraan.

### Alias kolom SQL yang TIDAK dibawa

`RDB List/GetDataMasterGrouping-SQL.xml` menamai ulang kolomnya agar cocok dengan properti di
atas:

| Ditulis kueri lama | Kolom sebenarnya |
|---|---|
| `A.NAMA_PANEL AS "PANJANG"` | nama panel |
| `A.SISI_PANEL AS "LEBAR"` | sisi panel |
| `B.NO_RANGKA AS "TINGGI"` | nomor rangka |
| `B.TIPE AS "QTY_PESAN"` | tipe kendaraan |
| `A.GROUPING_DGN_RANGKA AS "BERAT"` | nomor rangka yang diikuti |
| `A.CATATAN AS "MAX_STOCK"` | catatan |
| `A.ID_PANEL AS "MIN_STOCK"` | id panel |
| `A.NO_GROUP_RANGKA AS "pyID"` | nomor grup |
| `A.NO_PART AS "NO_SPART"` | nomor sparepart |
| `A.NAMA_PART AS "NAMA_SPART"` | nama sparepart |
| `A.KODE_PART AS "KODE_SPART"` | kode sparepart |

Dua alias lagi di luar kueri itu:

| Ditulis kueri lama | Kolom sebenarnya | Di mana |
|---|---|---|
| `id AS "BANK_ID"` | id tipe kendaraan | `BrowseTypeHE_Sql` |
| `TYPENAME AS "NAMA_BANK"` | nama tipe kendaraan | idem |
| `COUNT(A.ID) AS "City"` | jumlah antrean persetujuan | `CountMasterGrupSparepartManager` |
| `sisi_panel AS "NAME"` | sandi sisi | `GetDataSisiPanel` |

Alias `"NAMA_BANK"` itulah sebabnya properti bernama bank muncul di section Master Grouping
Sparepart, padahal tidak ada satu pun bank di layar ini.

### Judul dan caption yang berbeda antara grid dan form

Keduanya dipertahankan di tempatnya masing-masing (`D-13`), bukan diseragamkan:

| Kolom | Judul di GRID | Label di FORM |
|---|---|---|
| `NO_PART` | No Sparepart | Nomor Sparepart |
| `SISI_PANEL` | Sisi Panel | Sisi |

Ejaan **"Kategory Sparepart"** dengan y juga dipertahankan apa adanya — itulah yang tertulis
di layar lama.

### Dua tabel, dan kolom bernama sama di keduanya

| Tabel | Peran | Kolom |
|---|---|---|
| `POOLDATA.SPAREPART_HE_VIN_KEY` | induk | `ID`, `NO_PART`, `NAMA_PART`, `KODE_PART`, `KATEGORI_SPART`, `TIPE_SPART`, `PROD_DATE`, `ID_PANEL`, `NAMA_PANEL`, `SISI_PANEL`, **`NO_RANGKA`**, `GROUPING_DGN_RANGKA`, `NO_GROUP_RANGKA`, `CATATAN`, `APPROVAL` |
| `POOLDATA.SPAREPART_HE_VIN_GROUP` | pendamping, satu-lawan-satu | `ID`, **`NO_RANGKA`**, `TIPE` |

`NO_RANGKA` ada di **keduanya**, dan sistem lama memakai yang berbeda-beda: daftar menampilkan
milik pendamping, pemeriksaan duplikat menyaring milik induk. Lihat
`keputusan-implementasi.md` §36.4.

### Penyimpanan JSON yang TIDAK ditulis lagi

| Tabel | Perlakuan |
|---|---|
| `POOLDATA.M_SPAREPART_HE_VIN_KEY` | ditulis Pega lewat `PEGA_M_GROUPING_SPAREPART_HE`; **hanya DIBACA** aplikasi ini, untuk menghindari tabrakan ID selama masa paralel |

Perhatikan namanya: tabel baca berawalan `SPAREPART_HE_VIN_KEY`, tabel JSON berawalan
`M_SPAREPART_HE_VIN_KEY`. Keduanya berbeda hanya pada satu huruf di depan — dan yang pertama
adalah **substring** dari yang kedua, sehingga pemeriksaan berbasis substring biasa akan selalu
salah. Uji `TestOnlyOwnedTablesAreWritten` memakai pembanding berbasis kata untuk itu.

### Empat sumber acuan yang hanya DIBACA

| Sumber | Untuk isian | Nama tipe di Go |
|---|---|---|
| `POOLDATA.PANEL_HE` | Nama Panel | `Panel` |
| `POOLDATA.LOKASI_PANEL_HE` | Sisi | `Side` |
| `branddetail` | Tipe Kendaraan | `VehicleType` |
| `POOLDATA.SPAREPART_HE` | lima isian turunan | `PartRef` |

`branddetail` disebut **tanpa skema**, persis seperti kueri aslinya — tidak satu pun rule di
export menyebut skemanya, sehingga melengkapinya akan menjadi tebakan.

### Nama tipe: Grouping, bukan Group

`Group` terlalu umum dan bertabrakan dengan `Group` sebagai komponen tata letak di frontend.
`Grouping` mengikuti nama modul bisnis yang disebut Work Owner.

Nama tipe lainnya:

| Konsep | Nama di Go | Kenapa |
|---|---|---|
| Kunci alami empat kolom | `NaturalKey` | Tipe tersendiri, bukan empat argumen berjajar — keempatnya bertipe string dan urutannya tidak boleh dapat tertukar |
| Pasangan penentu daftar Sisi | `SideKey` | Dua nilai yang selalu dipakai bersama |
| Kelima isian turunan | `PartRef` | Ia **rujukan** ke Master Sparepart, bukan salinan sparepartnya |

### Nama kueri pada berkas .sql

Seluruhnya berawalan `grouping_`, mengikuti pola modul lain:

| Kelompok | Nama |
|---|---|
| Baca | `grouping_list`, `grouping_list_search`, `grouping_get`, `grouping_find_by_key` |
| Tulis | `grouping_insert`, `grouping_update`, `grouping_group_insert`, `grouping_group_update`, `grouping_set_status` |
| Kunci | `grouping_lock_by_key` |
| Penomoran | `grouping_max_id`, `grouping_max_id_mirror`, `grouping_group_numbers`, `grouping_group_by_chassis` |
| Acuan | `grouping_panel_list`, `grouping_side_list`, `grouping_vehicle_type_list`, `grouping_part_find` |
| Periksa | `grouping_check_*`, `grouping_count_*` |

### Kode galat

| Kode | Kapan |
|---|---|
| `grouping_sudah_ada` | keempat kunci alami sudah dipakai baris lain — 409 |
| `sparepart_tidak_ditemukan` | nomor sparepart yang diketik tidak ada di Master Sparepart — 404 |
| `validasi_gagal` | isian tidak lolos pemeriksaan — 422 |
| `status_tidak_dikenal` | status di luar 0/1/2 — 422 |
| `tidak_ditemukan` | baris yang dibuka tidak ada — 404 |
| `permintaan_cacat` | badan JSON tidak dapat dibaca, atau memuat field yang tidak dikenal — 400 |

---

## Master Tipe Sparepart (2026-09-22)

Modul `mastertipesparepart` — nama folder mengikuti nama modul bisnis yang disebut Work Owner
(`D-81`): backend `internal/mastertipesparepart`, frontend
`src/modules/master-tipe-sparepart`.

Paket transportnya `mastertipespareparthttp`.

### Kenapa tipenya bernama `PartType`, bukan `PartSection`

Awalan kolomnya `PART_SECTION_*`, dan `masterkategorisparepart.PartCategory` memang mengikuti
awalan kolomnya sendiri. Mengikuti pola itu di sini menghasilkan `PartSection` — dan itu
**menyesatkan**: tidak ada satu pun layar, menu, maupun caption yang menyebut "section". Yang
dilihat dan diucapkan pengguna adalah "Tipe Sparepart", dan nama tabelnya sendiri
`GCNM_M_SPAREPART_TYPE`.

`Type` sendiri tidak dapat dipakai — kata kunci Go. `PartType` juga sama dengan
`mastersparepart.PartType`, yang lebih dulu menamai hal yang sama pada lookup-nya.

### Kolom POOLDATA.GCNM_M_SPAREPART_TYPE

**Empat kolom**, dipastikan dari kesembilan rule Pega yang menyentuh tabel ini. Label layar
dibaca dari caption pada `Section/MasterTipeSparepartHEApproval-Section.xml`.

| Kolom | Label layar Pega | Nama di kode (Inggris) | Nama JSON (Indonesia) |
|---|---|---|---|
| `PART_SECTION_ID` | ID Tipe Sparepart | `ID` | `id_tipe_sparepart` |
| `PART_SECTION_NAME` | Nama Tipe Sparepart | `Name` | `nama_tipe_sparepart` |
| `PART_CATEGORY_ID` | ID Kategori Sparepart | `CategoryID` | `id_kategori_sparepart` |
| `APPROVAL` | — | `Status` | `status` |

Satu field **bukan kolom tabel ini** dan tidak pernah ditulis:

| Asal | Label layar Pega | Nama di kode | Nama JSON |
|---|---|---|---|
| `GCNM_M_SPAREPART_CATEGORY.PART_CATEGORY_NAME` lewat LEFT JOIN | Kategori Sparepart | `CategoryName` | `nama_kategori_sparepart` |

### Alias Pega yang TIDAK dibawa

Yang paling menyesatkan dari seluruh rumpun sparepart — dua alias terakhir **tertukar**
terhadap pola "…ID" yang dipakai dua alias sebelumnya:

| Alias Pega | Isi sebenarnya |
|---|---|
| `"CityID"` | `PART_SECTION_ID` |
| `"City"` | `PART_SECTION_NAME` |
| `"District"` | `PART_CATEGORY_ID` |
| `"DistrictID"` | `PART_CATEGORY_NAME` — **bukan sebuah ID** |

Properti input pada jalur simpan dipinjam dari kelas **Master Bengkel**
(`ASM-FW-GCNMFW-Int-BENGKEL_HE`), dan keempatnya pun tidak bernama seperti isinya:

| Properti Pega | Membawa |
|---|---|
| `InputKategori.CITY_ID` | nama tipe |
| `InputKategori.DISC_JASA` | ID kategori |
| `InputKategori.NO_ACCOUNT` | status persetujuan |
| `InputKategori.ACCOUNT_ID` | kunci baris |

### Nama kueri

| Kelompok | Nama |
|---|---|
| Daftar | `type_list`, `type_list_search` |
| Satu baris | `type_get`, `type_find_by_name` |
| Tulis | `type_lock_table`, `type_next_id`, `type_insert`, `type_update`, `type_set_status` |
| Acuan | `type_category_list` |
| Periksa | `type_check_table`, `type_count_*` |

### Rute dan kunci menu

| Hal | Nilai |
|---|---|
| MENU_ID | 34 |
| `MENU_PROGRAM` | `GCNMMasterSparepartType` |
| Rute frontend | `/master/tipe-sparepart` |
| Jalur API | `/api/master/tipe-sparepart`, `+/pilihan`, `+/keputusan` |

Perhatikan selisihnya: **nama program memakai "SparepartType", rutenya memakai "tipe"**.
Kunci petanya mengikuti basis data; rutenya mengikuti nama bisnis.

### Kode galat

| Kode | Kapan |
|---|---|
| `kunci_tipe_sparepart_sudah_ada` | nama sudah dipakai tipe lain — termasuk di kategori berbeda, termasuk yang ditolak — 409 |
| `kategori_sparepart_tidak_ditemukan` | kategori yang dipilih tidak ada atau tidak lagi disetujui — 409 |
| `validasi_gagal` | isian tidak lolos pemeriksaan — 422 |
| `status_tidak_dikenal` | status di luar 0/1/2 — 422 |
| `tidak_ditemukan` | baris yang dibuka tidak ada — 404 |
| `permintaan_cacat` | badan JSON tidak dapat dibaca, atau memuat field yang tidak dikenal — 400 |
---

## Master Reas (2026-09-22)

Modul `DataMemberReas` (MENU_ID 35) atas `POOLDATA.T_REINSURER`. **Baca-saja** — tidak ada
nama untuk jalur tulis karena jalurnya memang tidak ada.

### Nama modul dan paket (`D-80`, `D-81`)

| Lapisan | Nama | Catatan |
|---|---|---|
| Folder backend | `internal/masterreas` | nama modul bisnis, tanpa tanda hubung (Go tidak mengizinkannya) |
| Paket domain | `masterreas` | |
| Paket transport | `masterreashttp` | akhiran `http` tanpa tanda hubung |
| Folder frontend | `src/modules/master-reas` | `kebab-case` |
| Komponen layar | `ReasMemberPage` | memakai nama **tipe domain**, bukan nama modul |

### Kolom — aliasnya DIBUANG, namanya mengikuti isi

Inilah contoh telak utang `03-CURRENT-ARCHITECTURE.md` §4.2 pada modul ini. Rule lamanya
mengalias setiap kolom menjadi properti klipboard yang namanya tidak ada hubungannya dengan
isinya:

| Kolom basis data | Alias Pega | Field Go | Field JSON | Judul kolom layar |
|---|---|---|---|---|
| `REINSURERID` | `CityID` | `ReinsurerID` | `kode_reas` | Kode Reas |
| `REINSURERNAME` | `District` | `ReinsurerName` | `nama_reas` | Nama Reas |
| `LOGIN` | `DistrictID` | `Login` | `login` | Login |
| `EMAIL` | **`City`** | `Email` | `email` | Email |
| `COUNTRY` | `Country` | `Country` | `negara` | Negara |
| `TYPE` | — | `Type` | `tipe` | Tipe |
| `COUNTRYID` | — | **tidak ada** | **tidak dikirim** | **tidak ditampilkan** |

Alamat surel dialiaskan menjadi `City`, dan nama perusahaan reasuransi menjadi `District`.
Membaca rule lama berarti menelusuri kelimanya sampai ke pemanggilnya untuk tahu isian mana
yang mana.

**`COUNTRYID` tidak punya nama di mana pun**, dan itu disengaja: ia ditulis
`Database/UPDATEREAS.prc` lalu tidak dibaca satu pun rule di seluruh export.

### Satu nama yang sengaja TIDAK dibuat

| Yang tidak dinamai | Kenapa |
|---|---|
| Arti tiap nilai `TYPE` | Yang terbukti hanyalah ia dicocokkan dengan karakter pertama nomor dokumen PLA/DLA (`substr(a.NODLA,0,1) = TYPE`). Apa arti tiap nilainya **tidak diketahui** (`R-16`), sehingga nama seperti `DocumentType` atau `JenisDokumen` akan menjadi tebakan yang tersamar sebagai fakta |

Yang **dinamai** hanyalah nilai yang artinya terbukti dari dua tempat sekaligus:

| Nama | Nilai | Bukti |
|---|---|---|
| `masterreas.FallbackType` | `"1"` | `BrowseEmailReas` (`or type = '1'`) dan `UPDATEREAS.prc` (`AND TYPE = '1'` lalu `SET TYPE = tTYPE`) |
| field `cadangan` (JSON) | turunan | dihitung server, supaya layar tidak perlu tahu `'1'` punya arti khusus |

### Nama kueri

Seluruhnya berawalan `reas_`, dan **seluruhnya membaca**.

| Kelompok | Nama |
|---|---|
| Daftar | `reas_list`, `reas_list_search` |
| Periksa | `reas_check_table`, `reas_count_all`, `reas_count_duplicate_key`, `reas_count_shared_login`, `reas_count_empty_login`, `reas_count_missing_email`, `reas_count_without_fallback` |

Tidak ada kelompok **Tulis** dan tidak ada kelompok **Acuan** — yang pertama karena modulnya
tidak menulis, yang kedua karena tabel acuannya (`COUNTRY`) tidak dibaca satu pun rule di
export.

### Rute dan kunci menu

| Hal | Nilai |
|---|---|
| MENU_ID | 35 |
| `MENU_PROGRAM` | `DataMemberReas` |
| Rute frontend | `/master/reas` |
| Jalur API | `/api/master/reas` — **hanya `GET`** |

Perhatikan selisihnya: **nama program memakai "DataMember", judul menunya "Master Reas", dan
caption section lamanya "Data Member"**. Kunci petanya mengikuti basis data, rutenya
mengikuti nama menu, dan caption lamanya dipakai sebagai keterangan di bawah judul layar.

### Kode galat

| Kode | Kapan |
|---|---|
| `permintaan_cacat` | kata pencarian melampaui 100 karakter — 400 |

Hanya satu, dan itu bukan kelalaian: modul yang tidak menulis tidak punya keadaan yang dapat
ditolak atas dasar aturan bisnis. Galat portal dipetakan `portalhttp.WithPortalError`, dan
galat teknis diserahkan ke penulis galat bersama.

## Detail Penyebab Kerugian (2026-09-23)

Modul `internal/detailpenyebab` (backend) dan `src/modules/detail-penyebab-kerugian`
(frontend), atas `POOLDATA.D_CAUSE_OF_LOSS`. Nama modulnya berbahasa Indonesia sesuai
`D-81`; isinya berbahasa Inggris sesuai `D-80`.

### Istilah

| Kolom basis data | Identifier Go | Field JSON API | Label layar | Catatan |
|---|---|---|---|---|
| `D_COL_ID` | `ID` | `id` | ID | diterbitkan server; `pyReadOnly` di Pega |
| `OLD_D_COL_ID` | `LegacyID` | `id_lama` | — | tidak digambar; **bukan** penampung lain — lihat di bawah |
| `M_COL_ID` | `MasterID` | `id_master` | ID Master Kerugian | kunci induk |
| `COL_DESC` | `MasterLabel` | `nama_master` | — | milik induk; tidak tersimpan di baris ini |
| `DESCRIPTION` | `Description` | `deskripsi_kerugian` | Deskripsi Kerugian | pengurut daftar |
| `LOSS_CODE` | `LossCode` | `kode_kehilangan` | Kode Kehilangan | |
| `STS_AKTIF` | `Active` | `status_aktif` | Status Aktif | `"1"` / `"0"` |
| — | `ActiveLabel()` | `label_status_aktif` | — | diturunkan server |
| `BISNISID[]` | `Business[]` | `bisnis` | Bisnis | di dalam JSON; dibaca dari view |
| `BUSINESS.ID` | `Business.ID` | `bisnis[].id` | — | |
| `BUSINESS.NOTE` | `Business.Name` | `bisnis[].nama` | — | |

### Tiga nama Pega yang TIDAK dibawa, dan alasannya

| Nama Pega | Kenapa tidak dipakai |
|---|---|
| `OLD_D_COL_ID` | Ia bukan "ID lama dari sesuatu" melainkan **ID baris ini pada sistem sebelum Pega**. Lebih buruk lagi: properti bernama sama dipakai sebagai **pengangkut dokumen JSON** pada jalur simpan — dua arti, dua page berbeda |
| `LOSS_CODE` | Ia **bukan** kode penyebab kerugian. Pada Master Pasal Kerugian, nama yang sama memuat sebutan kategori; pada kueri lain ia nama lini bisnis. Satu nama, tiga isi |
| `M_COL_ID` / `D_COL_ID` | Tidak menyebutkan isinya sama sekali; huruf "COL" adalah singkatan *cause of loss* yang tidak terbaca siapa pun di luar konteksnya |

### Rute dan kunci menu

| Hal | Nilai |
|---|---|
| MENU_ID | 38 |
| `MENU_PROGRAM` | `DetailCauseOfLoss` |
| Rute frontend | `/master/detail-penyebab-kerugian` |
| Jalur API | `/api/master/detail-penyebab` — `GET`, `POST`, `PUT` |
| Jalur pilihan | `/pilihan` (tanpa portal) · `/pilihan/master` · `/pilihan/bisnis` |

**Tidak ada `DELETE`** — layar lamanya tidak punya tombolnya, dan `D-66` melarang
penghapusan fisik. Baris yang tidak lagi dipakai ditandai lewat Status Aktif.

Perhatikan selisih namanya: **rute frontend memakai "penyebab-kerugian" lengkap** mengikuti
`MENU_DESC`, sedangkan **jalur API memendekkannya menjadi `detail-penyebab`** dan **paket Go
menjadi `detailpenyebab`**. Kunci petanya mengikuti basis data (`DetailCauseOfLoss`).

### Kode galat

| Kode | Kapan | Status |
|---|---|---|
| `validasi_gagal` | Status Aktif di luar kedua pilihannya | 422 |
| `kunci_detail_penyebab_sudah_ada` | nomor dari sequence sudah dipakai baris lain | 409 |
| `tidak_ditemukan` | baris tidak ada | 404 |
| `permintaan_cacat` | badan tidak terbaca, field tak dikenal, atau kata cari terlalu panjang | 400 |

`kunci_detail_penyebab_sudah_ada` sebentuk dengan `kunci_login_surveyor_sudah_ada` dan
`kunci_sparepart_sudah_ada` — ketiganya menyatakan kunci yang bentrok. Bedanya: yang ini
**bukan kesalahan pengguna**, karena nomornya diterbitkan sequence dan bukan diketik.

---

## Inbox Investigator (2026-09-23)

`MENU_ID 48` → harness `InboxInvestigator_Harness`. Modul INBOX pertama.

### Nama modul dan paket (`D-80`, `D-81`)

| Lapisan | Nama | Bentuk |
|---|---|---|
| Folder backend | `internal/inboxinvestigator` | huruf kecil, tanpa tanda hubung |
| Paket Go | `inboxinvestigator` | idem |
| Paket transport | `inboxinvestigatorhttp` | nama modul + `http` |
| Folder frontend | `src/modules/inbox-investigator` | `kebab-case` |
| Komponen layar | `InvestigatorInboxPage` | **Inggris**, mengikuti tipe domainnya |

Nama modulnya **"Inbox Investigator"** — nama yang disebut Work Owner dan yang tertulis di
menu (`MENU_DESC` pada `m_menu_aplikasi_pnc.csv`). Isi modulnya berbahasa Inggris (`D-80`).

Perhatikan komponen layarnya: `InvestigatorInboxPage`, bukan `InboxInvestigatorPage`. Ia
mengikuti tata bahasa Inggris — inbox milik investigator — sama seperti `AccountPage` mengikuti
tipe `Account`, bukan nama modul `masterrekening`.

### Istilah — Tugas, bukan Klaim

Pembedaan paling menentukan di modul ini.

| `CONTEXT.md` | Di kode | Kenapa |
|---|---|---|
| **Tugas** | `Task` | yang didaftar layar ini adalah PEKERJAAN pada satu tahap klaim |
| Workbasket | `Workbasket` | antrean bersama yang belum bertuan (`D-26`) |
| Klaim | — | **tidak dipakai** sebagai nama entitas di modul ini |

Menamainya `Claim` akan membuat penyaring workbasket terbaca sebagai penyaring tambahan,
padahal ia **definisi** isi layarnya.

### Kolom — caption grid Pega → nama Go → nama JSON

Nama JSON mengikuti **caption grid layar lama** (`D-13`), bukan nama kolom basis data dan
bukan nama properti Pega.

| Caption grid | Properti Pega | Kolom fisik | Nama Go | JSON |
|---|---|---|---|---|
| — (tidak tampil) | `.pzInsKey` | `PZINSKEY` | `Reference` | `referensi` |
| Nomor Case | `.pyID` | `PYID` | `CaseNumber` | `nomor_case` |
| No Polis | `.Policy.PolicyNo` | `POLICYNO` | `PolicyNumber` | `nomor_polis` |
| Nama Tertanggung | `.Policy.QQName` | `QQNAME` | `InsuredName` | `nama_tertanggung` |
| Nama Peserta | `.ClaimData.ObjectList(1).ObjectName` | `T_CLAIM_OBJECTLIST.OBJECTNAME` | `ParticipantName` | `nama_peserta` |
| Nama Bisnis | `.Policy.Quotation.BusinessName` | `BUSINESSNAME` | `BusinessName` | `nama_bisnis` |
| Nama Cabang | `.Policy.Quotation.BranchName` | `BRANCHNAME` | `BranchName` | `nama_cabang` |
| Nama Admin | `.pyOrigUserID` | `PYORIGUSERID` | `AdminName` | `nama_admin` |
| Tanggal Pendaftaran | `.pxCreateDateTime` | `PXCREATEDATETIME` | `RegisteredAt` | `tanggal_pendaftaran` |
| **Lama Masuk Inbox** | `.ClaimData.SurveyResults(1).SurveyDate` | `T_SURVEYORLIST.SURVEYDATE` | `SurveyDate` | `tanggal_survey` |

**SEMBILAN kolom, bukan sepuluh.** Tidak ada kolom "Tanggal Survey" terpisah, dan `.pyNote`
bukan kolom — ia hanya muncul di dalam definisi kontrol tautan pada kolom pertama.

**Kolom kesembilan adalah satu-satunya tempat nama field TIDAK mengikuti caption**, dan itu
pengecualian yang disengaja: captionnya menyebut durasi ("Lama Masuk Inbox"), isinya tanggal.
Menamainya `lama_masuk_inbox` akan membuat kontraknya berbohong tentang tipe datanya sendiri.
Yang tetap mengikuti Pega adalah **judul kolom di layar** (`D-13`).

Ketidakcocokan judul-isi itu ADA di sistem lama: section ini Save-As dari inbox Compliance,
tempat kolom bernama sama memang berisi lama menunggu. Selnya diikat ulang ke tanggal survei
dan captionnya tidak ikut diganti.

### Empat nama Pega yang TIDAK dibawa, dan alasannya

| Nama Pega | Kenapa tidak dibawa |
|---|---|
| **"Nama Bisinis"** | salah ketik pada caption **penyaring** section; caption kolomnya sendiri sudah benar "Nama Bisnis" |
| **`QQNAME as "CABANG"`** | alias pada `ReminderPUCL-SQL.xml` — isinya nama tertanggung, bukan cabang. Utang `03-CURRENT-ARCHITECTURE.md` §4.2 |
| **`.pyNote`** | properti generik Pega yang dipakai menampung teks lama menunggu. Di sini nilainya dihitung, bukan dititipkan ke properti serbaguna |
| **`Param.Operator`** | nama parameter RD yang isinya nama **workbasket**, bukan operator. Di sini: `Workbasket` |

### Istilah baru di modul ini

| Go | Arti | Asal |
|---|---|---|
| `Workbasket` | antrean bersama yang isinya ditampilkan layar ini | `pyReportDefParams` section |
| `ResolvedWorkStatus` | nilai `pyStatusWork` yang menandai pekerjaan tuntas | penyaring RD |
| `MaxRows` | batas baris yang dikirim satu permintaan | `pyMaxRecords = 500` |
| `Page.Truncated` | penanda hasil terpotong | **baru** — sistem lama memotong dalam diam |

### Nama kueri

| Nama | Isi |
|---|---|
| `investigator_inbox_list` | antrean, tanpa penyaring kata kunci |
| `investigator_inbox_search` | idem, ditambah penyaring 7 kolom |
| `investigator_inbox_check_table` | pembuktian keempat tabel dapat dibaca |
| `investigator_inbox_count_waiting` | cacah seluruh pekerjaan menunggu, tanpa dipotong |
| `investigator_inbox_count_without_survey` | cacah yang lama menunggunya akan kosong |

Seluruh alias kolomnya **sama dan seurut** pada ketiga kueri yang barisnya dipindai —
dijaga uji `TestScannedQueriesShareTheSameColumnOrder`.

### Rute dan kunci menu

| Hal | Nilai |
|---|---|
| Kunci `MENU_ROUTES` | `InboxInvestigator_Harness` |
| Rute frontend | `/inbox/investigator` |
| Endpoint | `GET /api/inbox/investigator` |

**Awalan `/inbox/` diperkenalkan modul ini.** INBOX adalah kelompok menu tersendiri di sistem
lama (`MENU_ID 2`, induk dari 30 butir), sehingga modul inbox berikutnya punya tempat yang
sudah jelas — seperti `/master/...` yang sudah berlaku untuk 20 butir.

### Kode galat

Satu saja: `permintaan_cacat` (`CodeMalformedRequest`), untuk kata kunci yang terlalu panjang.

Modul ini **tidak punya galat domain** — ia hanya membaca, sehingga tidak ada keadaan yang
dapat ditolak atas dasar aturan bisnis. Antrean yang **kosong bukan galat**; ia justru keadaan
yang diharapkan pada inbox yang sudah dikerjakan.

Galat portal dipetakan `portalhttp.WithPortalError`, sama seperti seluruh modul lain.

### Export Data Investigation — nama yang TIDAK dibawa (2026-09-24)

Export-nya belum dibangun (lihat `keputusan-implementasi.md` §45), tetapi pemetaan namanya
dicatat sekarang supaya tidak perlu ditelusuri ulang saat artefaknya tiba.

`Activity/ExportDataInvestigator-Act.xml` mengalias **sepuluh properti sumber** menjadi
**sebelas kolom CSV** yang namanya **sama sekali tidak menyatakan isinya**:

| Properti sumber (`SurveyList(1).*`) | Kolom CSV di Pega | Isi sebenarnya |
|---|---|---|
| `AlamatRSKlinik` | `ClaimNo` | alamat rumah sakit / klinik |
| `NoRekapMedis` | `Location` | nomor rekap medis |
| `PasienTerdaftar` | `NewTelpTertanggung` | status pasien terdaftar |
| `SelectRS` | `ProdKe` | `"1"` → "Rumah Sakit" |
| `KonfirmasiModelKwitansi` | `IsCFS_PNC` | konfirmasi model kuitansi |
| `NoTelpDiHubungi` | `NewEmail` | nomor telepon yang dihubungi |
| `Remaks` | `NoKTP` | catatan investigasi |
| `CheckBoxAsuransiLain` | `ComplianceRemark` | penanda asuransi lain |
| `CheckBoxPasien` | `Country` **dan** `NoteKasir` | penanda pasien |
| `CheckBoxTidakadapembayaran` | `Email` | penanda tidak ada pembayaran |

**Tidak satu pun alias itu dibawa.** Alamat rumah sakit yang bernama `ClaimNo` dan catatan
investigasi yang bernama `NoKTP` adalah bentuk utang `03-CURRENT-ARCHITECTURE.md` §4.2 dalam
wujud paling berbahaya — ia sampai ke berkas yang dibuka orang di luar aplikasi.

Saat export dibangun kelak, nama kolomnya mengikuti **isi**, dan pemetaan di atas dipakai
untuk mencocokkannya dengan berkas lama.

#### Nama komponen layar

| Hal | Nama | Catatan |
|---|---|---|
| Komponen panel | `ExportPanel` | Inggris (`D-80`), di dalam modul `inbox-investigator` |
| Judul panel di layar | "Export Data Investigation" | caption Pega apa adanya (`D-13`) |
| Label isian | "Pilih Investigation", "Dari", "Sampai" | caption Pega apa adanya |
| Label tombol | "Export Data" | `pyButtonLabel` Pega apa adanya |

## Inbox Receive TKA (2026-09-24)

`MENU_ID 49` → harness `InboxTKA_Harness`. Modul INBOX kedua, dan yang pertama menulis.

### Nama modul dan paket (`D-80`, `D-81`)

| Lapisan | Nama | Bentuk |
|---|---|---|
| Folder backend | `internal/inboxreceivetka` | huruf kecil, tanpa tanda hubung |
| Paket Go | `inboxreceivetka` | idem |
| Paket transport | `inboxreceivetkahttp` | nama modul + `http` |
| Paket notifikasi | `notification` | subpaket, bukan nama modul |
| Folder frontend | `src/modules/inbox-receive-tka` | `kebab-case` |
| Komponen layar | `ReceiveTKAInboxPage` | **Inggris**, mengikuti tata bahasanya |

Nama modulnya **"Inbox Receive TKA"** — `MENU_DESC` pada `m_menu_aplikasi_pnc.csv`, dan
caption berformat `Heading 1` pada `InboxTKA_Section` berbunyi sama. Isi modulnya berbahasa
Inggris (`D-80`).

Komponen layarnya `ReceiveTKAInboxPage`, bukan `InboxReceiveTKAPage` — mengikuti tata bahasa
Inggris (inbox untuk penerimaan TKA), sama seperti `InvestigatorInboxPage` pada modul
sebelumnya.

### Istilah — Tugas, dan satu yang baru

| `CONTEXT.md` | Di kode | Kenapa |
|---|---|---|
| **Tugas** | `Task` | yang didaftar layar ini PEKERJAAN, bukan klaim |
| — | `Completion` | satuan kerja tombol Submit: satu baris, satu tanggal |
| — | `Notice` | peristiwa domain "dokumen sudah lengkap" |

`Completion`, bukan `Update`: yang terjadi bukan penyuntingan field melainkan **penyelesaian
pekerjaan** — barisnya keluar dari inbox dan pemberitahuan dilepaskan.

`Notice` menyatakan APA YANG TERJADI, bukan siapa yang harus diberi tahu — itulah yang
membuat penerimanya menjadi urusan konfigurasi (`04-FUTURE-ARCHITECTURE.md` §3.6).

**Tanpa `Workbasket`.** Berbeda dari Inbox Investigator, layar ini tidak punya antrean
penugasan sama sekali; Report Definition-nya mendeklarasikan halaman workbasket tetapi tidak
pernah merujuknya.

### Kolom — caption grid Pega → kolom fisik → nama Go → nama JSON

Nama JSON mengikuti **caption grid layar lama** (`D-13`), dengan satu pengecualian.

> **Dikoreksi 2026-09-24.** Tabel di bawah semula memetakan seluruh kolom ke
> `POOLDATA.T_CLAIM_TKA_H`. Tabel itu **tidak dipakai sama sekali** — lihat
> `catatan-pengembangan.md` §46 dan `keputusan-implementasi.md` §48.

| Caption grid | Kolom fisik | Nama Go | JSON |
|---|---|---|---|
| — (tidak tampil) | `PC_ASM_FW_GCNMFW_WORK.PZINSKEY` | `Reference` | `referensi` |
| — (tidak tampil) | `T_CLAIM_PNC.CLAIMID` | `ClaimKey` | `klaim_tersedia` |
| Nomor Klaim | `PC_ASM_FW_GCNMFW_WORK.PYID` | `ClaimNumber` | `nomor_klaim` |
| No Polis | `PC_ASM_FW_GCNMFW_WORK.POLICYNO` | `PolicyNumber` | `nomor_polis` |
| Nama Tertanggung | `PC_ASM_FW_GCNMFW_WORK.QQNAME` | `InsuredName` | `nama_tertanggung` |
| Nama Peserta | `POOLDATA.T_GENERAL.THEINSURED` | `ParticipantName` | `nama_peserta` |
| **Date Of Loss** | `PC_ASM_FW_GCNMFW_WORK.DATEOFLOSS_1` | `DateOfLoss` | **`tanggal_kejadian`** |
| Aging | `PC_ASM_FW_GCNMFW_WORK.REGISTERDATE_1` | `RegisteredOn` | `tanggal_registrasi` |
| Tanggal Dokumen Lengkap | `T_CLAIM_PNC.TGLDOKLENGKAP` | — (isian) | `tanggal_dokumen_lengkap` |

**Pengecualian pada kolom Date Of Loss.** Captionnya berbahasa Inggris — sementara
`D-80` menetapkan nama field JSON berbahasa Indonesia dan `CONTEXT.md` sudah menetapkan
padanan resminya: **Tanggal Kejadian**. Yang mengikuti Pega adalah **judul kolom di layar**;
yang mengikuti glossary adalah nama kontraknya.

**Kolom "Aging" mengirim TANGGAL, bukan kalimat.** Pega menampilkan `.ClaimData.RegisterDate`
sebagai waktu relatif ("2 years 6 months ago"). Server mengirim `tanggal_registrasi` apa
adanya dan **layar yang menyusun kalimatnya**, karena "berapa lama menunggu" bergantung pada
kapan ia dibaca. Kolom fisiknya `VARCHAR2(32)` berisi `yyyymmdd`; penguraiannya di adapter,
bukan di SQL (`D-20` melarang `TO_DATE`). Bentuk yang tidak dikenali menjadi **null**, bukan
tanggal karangan.

**Dua kolom yang TIDAK di-expose, dan penggantinya.** `.ClaimData.ClaimNo` diganti `PYID`
(nilainya sama dengan yang tampil di layar Pega), dan `.Policy.TheInsured` diganti
`T_GENERAL.THEINSURED` lewat `NOPOLIS` + `PRODKE` — karena `INSUREDNAME` pada tabel kerja
terbukti kosong. Properti yang sekadar ditampilkan tidak menuntut kolom; Pega membacanya dari
BLOB kasus, dan kita tidak bisa.

**`klaim_tersedia` menggantikan `referensi` kosong** sebagai penanda baris yatim. `referensi`
kini selalu terisi karena sumbernya tabel yang menggerakkan kueri, bukan hasil gabungan.

### Caption yang pemasangannya sempat terbalik

| Sumber | `.Policy.QQName` | `.Policy.TheInsured` |
|---|---|---|
| `InboxTKA_Section` | Nama Tertanggung | Nama Peserta |
| `InboxTKA_RD` (`pyFieldLabel`) | Nama Peserta | Nama Tertanggung |
| `NotificationKelengkapanTKA` | **Nama Tertanggung** | **Nama Peserta** |

Yang dipakai adalah Section, dikuatkan badan surel. Label Report Definition tidak dibawa.

### Nama properti Pega yang TIDAK dibawa

`SubmitTanggalLengkapTKA` menampung nilai surel pada properti yang namanya menyesatkan —
bentuk yang sama dengan alias pada `03-CURRENT-ARCHITECTURE.md` §4.2:

| Properti Pega | Isi sebenarnya | Nama di sistem baru |
|---|---|---|
| `TempHTML.District` | nama tertanggung | `Notice.InsuredName` |
| `TempHTML.DistrictID` | nama peserta | `Notice.ParticipantName` |
| `TempHTML.Country` | Date Of Loss | `Notice.DateOfLoss` |
| `TempHTML.CountryID` | tanggal kelengkapan dokumen | `Notice.CompletedAt` |

### Galat — nama Go → kode JSON

| Nama Go | Kode JSON | HTTP |
|---|---|---|
| `ErrClaimNumberRequired` | `permintaan_cacat` | 400 |
| `ErrDateRequired` | `tanggal_dokumen_wajib_diisi` | 422 |
| `ErrTaskNotFound` | `pekerjaan_tidak_ditemukan` | 409 |
| `ErrClaimMissing` | `klaim_tidak_ditemukan` | 409 |
| `ErrClaimAmbiguous` | `nomor_klaim_ganda` | 409 |

### Rute dan variabel lingkungan

| Hal | Nilai |
|---|---|
| Rute layar | `/inbox/receive-tka` |
| Endpoint daftar | `GET /api/inbox/receive-tka` |
| Endpoint Submit | `POST /api/inbox/receive-tka/kelengkapan-dokumen` |
| Kunci peta menu | `InboxTKA_Harness` — persis seperti `MENU_PROGRAM` |
| Variabel lingkungan baru | `SMTP_PENERIMA_TKA` |

Nama variabel lingkungan tetap **bahasa Indonesia** (`D-80`): ia dipakai berkas `.env` dan
skrip deployment, sehingga sifatnya kontrak — sama seperti `SMTP_PENERIMA_PERINGATAN` yang
sudah ada.

### Nama kueri SQL

Berawalan `tka_` supaya tidak bentrok dengan modul lain di dalam satu berkas `.sql` yang
dimuat lewat penanda `-- name:`.

| Kelompok | Nama |
|---|---|
| Baca | `tka_inbox_list`, `tka_inbox_search`, `tka_inbox_find_one` |
| Tulis | `tka_claim_set_document_date` — **satu kueri, satu kolom** |
| Periksa | `tka_inbox_check_table`, `tka_inbox_check_claim_column` |
| Cacah | `tka_inbox_count_waiting`, `..._pega_only`, `..._orphan_claim`, `..._missing_participant` |
| Contoh | `tka_inbox_sample_registered_on` |

**Tanpa kueri penguncian.** `SELECT ... FOR UPDATE` dicabut: menahan kunci pada tabel yang
sedang dilayani Pega dapat menghentikan alur kerja yang berjalan di atas kasus yang sama.
Penggantinya penjaga `AND TGLDOKLENGKAP IS NULL` pada UPDATE ditambah pemeriksaan jumlah
baris terpengaruh. `TestNoQueryLocksThePegaWorkTable` dan `TestPegaWorkTableIsNeverWritten`
menjaga keduanya.

#### Tabel penampungnya ketemu (2026-09-24)

`POOLDATA.INVESTIGATIONREPORT` — 36 kolom, berkunci `CASEID`. Strukturnya **enam pasang
tanya-jawab**, masing-masing satu penanda `CHAR(1)` dan satu keterangan `VARCHAR2(4000)`:

```
ISPATIENTREGIST / PATIENTDESC      ISBILLPAID     / BILLPAIDDESC
ISDATECORRECT   / DATEDESC         ISADDRESSCORR  / ADDRESSDESC
ISBILLCORRECT   / BILLDESC         STSKWITANSI    / KWITANSIDESC
```

**Pemetaan isian CSV ke kolom BELUM PASTI** dan sengaja tidak ditebak. Yang sudah kuat:

| Isian CSV lama | Kolom | Keyakinan |
|---|---|---|
| Status pasien terdaftar | `ISPATIENTREGIST` | kuat |
| Konfirmasi model kwitansi | `STSKWITANSI` | kuat |
| Alamat rumah sakit / klinik | `ADDRESS` | kuat |
| Catatan investigasi | `NOTE` | kuat |
| Nomor telepon yang dihubungi | `TELPRS` | kuat |
| Penanda tidak ada pembayaran | `ISBILLPAID` | sedang |
| Penanda asuransi lain | `PAIDBY` | sedang |
| Penanda pasien | bertabrakan `ISPATIENTREGIST` | lemah |
| **Nomor rekap medis** | **tidak ada** | — |
| **Pilihan RS / non-RS** | **tidak ada** | — |

Pemetaan final ditetapkan setelah tiga pertanyaan pada `permintaan-artefak-pega.md` §2.5
terjawab — **bukan** dari kemiripan nama. Nama kolom di sistem ini sudah terbukti menyesatkan
(`03-CURRENT-ARCHITECTURE.md` §4.2), dan pada layar ini sudah sekali menjebak: kolom
bercaption "Lama Masuk Inbox" ternyata berisi tanggal survei.
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

---

## Tambahan 2026-09-22 — modul Inbox Claim Treaty Prop (`inboxclaimtreatyprop`)

Menu `MENU_ID 54`, pengganti harness `InboxClaimTreaty_Harness`.

### Kenapa modul ini kasus terburuk sejauh ini

Di modul sebelumnya, nama properti Pega MENYESATKAN — `.ASMFull` berarti Tanggal Kejadian,
`.RCVID` berarti sumber bisnis. Di sini nama propertinya **tidak menyatakan apa pun**:
seluruh kolom bernama `CARI` ditambah nomor urut.

Dan nomornya tidak stabil. Tanggal Kejadian bernomor `CARI10` di dua kueri dan `CARI13` di
kueri ketiga — perbedaan yang menjadi cacat nyata, karena grid-nya terikat ke `CARI10`
saja.

### Pemetaan tiga arah

| Alias Pega | Asal sebenarnya | Nama di kode | Field JSON | Judul kolom |
|---|---|---|---|---|
| `CARI1` | `a.PXREFOBJECTKEY` | `WorkKey` | — (tidak dikirim) | — |
| `CARI2` | `a.PXREFOBJECTINSNAME` | `ClaimID` | `claim_id` | Claim ID |
| `CARI3` | `a.PXASSIGNEDOPERATORID` | `AssignedOperator` | — (tidak dikirim) | — |
| `CARI4` | `a.PZINSKEY` | `Reference` | `referensi` | — (tidak digambar) |
| `CARI5` | `b.NOPOLIS` | `PolicyNumber` | `no_polis` | Policy No |
| `CARI6` | `$.QuotationData.BusinessName` | `BusinessName` | `nama_bisnis` | Business Name |
| `CARI7` | `$.QuotationData.SobName` | `BusinessSource` | `sumber_bisnis` | Source Of Business |
| `CARI8` | `$.QuotationData.CedingCoName` | `CedingCompany` | `ceding_co` | Ceding Co Name |
| `CARI9` | `$.InsuredName` | `InsuredName` | `nama_tertanggung` | Insured Name |
| `CARI10` (!) | `$.DateOfLoss` | `LossDate` | `tanggal_kejadian` | Date Of Loss |
| `CARI13` (!) | `$.DateOfLoss` | `LossDate` | `tanggal_kejadian` | Date Of Loss |
| `CARI14` | `$.IsSubjectivity` | `Subjectivity` | `subjectivity` | Subjectivity |
| `CARI15` | `$.IDMaster` | `MasterID` | `id_master` | ID Master |
| `CARI43` | checkbox layar | `Query.SeeAll` | `lihat_semua` | See All Claim |

Tanda (!) menandai satu nilai yang punya DUA nomor alias. Di sistem baru keduanya menjadi
satu kolom `LOSS_DATE` — perbaikan `P-5` yang disetujui Work Owner 2026-09-21.

### Properti yang BUKAN kolom data

| Properti Pega | Artinya sebenarnya | Di sistem baru |
|---|---|---|
| `InputData.CARI13` | pemanggil adalah anggota komite (`EMAILKOMITE.OPERATOR_ID`) | tidak dibawa — ketiga kontainer menjadi tab yang dapat dipilih |
| `SearchWorkbasket.CARI1` | login pemanggil, dibandingkan ke `TreatyinPNCTeknik` | `Caller.Login` dan konstanta `TechnicalWorkbasket` |
| `SearchWorkbasket.CARI43` | checkbox "See All Claim" | `Query.SeeAll` |

Perhatikan `CARI1`: ia berarti **dua hal berbeda** tergantung halamannya. Pada
`DataPNC.pxResults` ia kunci objek kerja; pada `SearchWorkbasket` ia login pemanggil. Nama
yang sama, arti yang sama sekali berbeda — persis pola `03-CURRENT-ARCHITECTURE.md` §4.2.

### Judul yang dipertahankan apa adanya (`D-13`)

| Di Pega | Dibawa? | Alasan |
|---|---|---|
| "Work List Treatyin **Propotional**" | ya, salah ejanya ikut | Membetulkannya menjadi "Proportional" membuat tab tidak dikenali pengguna yang mencarinya |
| "Work Teknik Treatyin " (spasi di ujung) | ya, TANPA spasinya | Spasi itu artefak pengetikan, bukan teks yang dibaca; membawanya hanya menggagalkan pembandingan judul |
| "Business Name" vs "Class Of Business" | keduanya | Isi yang sama berjudul berbeda antar grid; perbedaannya dijaga lewat `Tab.Columns` |

### Nama modul dan berkas (`D-80`, `D-81`)

| Lapisan | Bentuk |
|---|---|
| Paket Go | `inboxclaimtreatyprop` — huruf kecil, tanpa tanda hubung |
| Folder frontend | `inbox-claim-treaty-prop` — `kebab-case` |
| Rute antarmuka | `/inbox-claim-treaty-prop` |
| Jalur API | `/api/inbox-claim-treaty-prop` |
| Komponen layar | `ClaimTreatyPropPage` — nama TIPE, bukan nama modul |

Nama modulnya berbahasa Indonesia mengikuti nama yang disebut Work Owner ("Inbox Claim
Treaty Prop"); isinya berbahasa Inggris. Satu jalur berkas karena itu memuat dua bahasa —
`internal/inboxclaimtreatyprop/repo/sqlstore/inboxclaimtreatyprop.go` — dan itu memang
yang dikehendaki `D-81`.

## Tambahan 2026-09-21 — modul Inbox Progress Claim (`inboxprogressclaim`)

Modul ini punya ciri yang **tidak ada di modul mana pun sebelumnya**: dua alias tidak
sekadar salah arti, melainkan **tertukar satu sama lain**. `CaseID` berisi nomor klaim
sementara `ClaimNo` berisi nomor polis — persis terbalik dari yang dijanjikan namanya.

### Nama modul

`inboxprogressclaim` di backend, `inbox-progress-claim` di frontend. Ia mengikuti `D-81`:
nama modulnya diambil dari nama yang dipakai Work Owner dan tertulis di menu
(`MENU_ID 65` "Inbox Progress Claim"), sementara isinya berbahasa Inggris.

### Kode bagian — kata, bukan angka

Berbeda dari Inbox Admin, di sini tidak ada nomor warisan yang perlu dipertahankan. Layar
lama **tidak punya pemilih bagian sama sekali** — kelima bagiannya ditumpuk dan
masing-masing memuat datanya sendiri.

| Kode | Bagian | Activity lama | Kueri lama |
|---|---|---|---|
| `outstanding` | Outstanding | `GetDataProgressClaim` | `DataProgressClaim` + `GcnmCountProgressClaim_SQL` |
| `next-fu` | Next Follow Up | `GetNextFUdata_act` | sama + saringan jatuh tempo |
| `per-pic` | Progress Klaim per PIC | `GetProgressPerPIC` | `GetProgressPIC` |
| `evaluasi` | Evaluasi Progress Klaim | `Refreshpage_act` | **tidak ada** |

Bagian "Approval Progress Klaim" **tidak dibangun** (keputusan Work Owner 2026-09-21): ia
satu-satunya bagian yang menulis.

### Tiga lapis nama, dan ketiganya sengaja berbeda

Layar ini satu-satunya tempat ketiga lapis itu **tidak sama**, dan itu keputusan sadar:

| Lapis | Contoh | Aturannya |
|---|---|---|
| Judul yang dibaca pengguna | `District` | alias Pega apa adanya (Work Owner 2026-09-21, `D-13`) |
| Nama field JSON | `nama_tertanggung` | Indonesia, menyebut isinya (`D-80` pengecualian kontrak) |
| Nama di dalam kode | `InsuredName` | Inggris, menyebut isinya (`D-19`, `D-80`) |

Arti sebenarnya ikut dikirim server sebagai `kolom[].keterangan`, dan layar menggambarnya
sebagai tooltip kolom. Tanpa itu, keputusan memakai alias apa adanya akan membuat layar
baru sama tidak terbacanya dengan layar lama.

### Alias grid Pega → arti sebenarnya → nama di kode

**Bagian Outstanding dan Next Follow Up:**

| Alias Pega | Kolom basis data | Arti bagi pengguna | Nama di kode | Field JSON |
|---|---|---|---|---|
| `CaseID` ⚠ | `a.NOKLAIM` | Nomor Klaim | `ClaimNumber` | `no_klaim` |
| `ClaimNo` ⚠⚠ | `a.NOPOLIS` | **Nomor Polis** | `PolicyNumber` | `no_polis` |
| `NoKTP` ⚠ | `a.NOPOLIS` | nomor polis, KEMBAR | tidak dibawa | — |
| `District` ⚠ | `T_CLAIM_PNC.QQNAME` | Nama Tertanggung | `InsuredName` | `nama_tertanggung` |
| `DateForAging` ⚠ | `a.TGLKLAIM` | Tanggal Registrasi | `RegisterDate` | `tanggal_registrasi` |
| `DateOfLoss` | `a.DATEOFLOSS` | Tanggal Kejadian | `LossDate` | `tanggal_kejadian` |
| `Country` ⚠ | `a.LGB_NOTE` | Catatan LGB | `LGBNote` | `catatan_lgb` |
| `UserTeknis` ⚠ | `a.PIC` | PIC Klaim | `TechnicalPIC` | `pic_klaim` |
| `City` ⚠ | `GET_POSISI_PROGRESS_PNC(…,'POSISI')` | Posisi berjalan | `Position.Name` | `posisi` |
| `CityID` ⚠ | `GET_POSISI_PROGRESS_PNC(…,'sts_prg1')` | Status Progres 1 | `Position.Status1` | `status_progres_1` |
| `CountryID` ⚠ | `GET_POSISI_PROGRESS_PNC(…,'sts_prg2')` | Status Progres 2 | `Position.Status2` | `status_progres_2` |
| `AnalystTransferDate` ⚠ | `GET_POSISI_PROGRESS_PNC(…,'nextfu')` | Next Follow Up | `Position.NextFollowUp` | `next_follow_up` |
| `KomiteApproveDate` ⚠ | `MIN(GCNM_PROGRESS_CLAIM.NEXT_FOLLOWUP)` | Follow Up terawal | `EarliestFollowUp` | `follow_up_terawal` |
| `TanggalAnalystSendRCL` ⚠ | `a.TGL_PROSES` | Tanggal Proses | `ProcessDate` | `tanggal_proses` |
| `ProdKe` | `a.PROD_KE` | Prod ke- | `ProdKe` | `prod_ke` |

⚠⚠ menandai alias yang **tertukar** dengan alias lain di grid yang sama.

`KomiteApproveDate` layak diperhatikan khusus: namanya menyebut persetujuan komite, padahal
ia tidak berhubungan dengan komite sama sekali — isinya tenggat tindak lanjut paling awal
pada klaim itu.

**Bagian Progress Klaim per PIC** — empat dari enam aliasnya menyebut atribut klaim padahal
seluruhnya hasil `COUNT`:

| Alias Pega | Arti bagi pengguna | Nama di kode | Field JSON |
|---|---|---|---|
| `PIC` | Nama petugas | `PIC` | `pic` |
| `NOKLAIM` ⚠ | Jumlah klaim yang ditangani | `ClaimCount` | `jumlah_klaim` |
| `NOAKSEP` ⚠ | Jumlah pembaruan progres, di luar `AUTO%` | `UpdateCount` | `jumlah_pembaruan` |
| `REINSURER` ⚠ | Tindak lanjut jatuh tempo hari ini | `DueTodayCount` | `jatuh_tempo_hari_ini` |
| `STSKLAIM` ⚠ | Tindak lanjut tepat waktu | `OnTimeCount` | `tepat_waktu` |
| `NOPOLIS` ⚠ | Tindak lanjut terlambat | `LateCount` | `terlambat` |

### Nama properti penyaring

| Properti Pega | Isinya | Nama di kode | Parameter query |
|---|---|---|---|
| `TempRefresh.ClaimNo` ⚠ | kata kunci pencarian | `Keyword` | `cari` |
| `TempRefresh.DateOfLoss` | kotak tanggal yang **tidak menyaring apa pun** | tidak dibawa | — |
| `TempRefresh.Remark` ⚠ | tanggal registrasi AWAL | `From` | `dari` |
| `TempRefresh.City` ⚠ | tanggal registrasi AKHIR | `To` | `sampai` |
| `tempgetpic.MCL_NAME` ⚠ | potongan SQL penyaring | tidak dibawa — diganti bind | — |
| `tempgetpic.CaseID` ⚠ | potongan SQL lini bisnis yang **tidak pernah dibaca** | tidak dibawa | — |
| `TempCabang.District` ⚠ | potongan SQL penyaring cabang | belum dibawa (`R-03`) | — |
| `TempBisnis.GROUP_PANEL` | potongan SQL lini bisnis yang benar-benar dipakai | `Business` | `bisnis` |
| `TempBisnis.NOAKSEP` ⚠ | potongan SQL **rentang tanggal**, bukan nomor akseptasi | `From`/`To` | `dari`/`sampai` |
| `OperatorID.pyPosition` ⚠ | lini bisnis petugas, bukan jabatannya | belum ada padanannya | — |

`TempRefresh.Remark` dan `TempRefresh.City` layak dicatat: keduanya adalah **sepasang batas
tanggal**, dan tidak ada satu pun pada namanya yang menyatakan itu.

### Nama tabel yang dibaca

Tidak ada yang dinamai ulang — nama tabel dan kolom milik basis data, pengecualian `D-80`,
dan perubahannya menempuh `D-63`.

| Tabel | Dipakai untuk |
|---|---|
| `POOLDATA.PEGA_DASHBOARDPNC` | tabel ringkasan klaim; sumber utama ketiga bagian |
| `POOLDATA.T_CLAIM_PNC` | nama tertanggung |
| `POOLDATA.GCNM_PROGRESS_CLAIM` | riwayat progres dan tenggat tindak lanjut |
| `POOLDATA.GCNM_PROGRESS_POSISI_PNC` | posisi yang sedang berjalan |
| `POOLDATA.GCNM_MST_PROGRESS_KLAIM` | master Status Progres 1 |
| `POOLDATA.GCNM_MST_PROGRESS` | master Status Progres 2 |
| `POOLDATA.MST_USER_TEKNIK` | daftar petugas per lini bisnis |

Keempat tabel progres itu sudah dimodelkan modul `masterstatusprogres`; modul ini hanya
membacanya.

---

## Tambahan 2026-09-19 — modul Inbox Laporan Klaim

Nama modulnya **nama bisnis dalam bahasa Indonesia** (`D-81`), isinya **berbahasa Inggris**
(`D-80`) — sama seperti ketiga modul master sebelumnya.

| Nama modul bisnis (Work Owner) | Folder backend / paket Go | Folder frontend |
|---|---|---|
| Inbox Laporan Klaim | `internal/inboxlaporanklaim` | `src/modules/inbox-laporan-klaim` |

**Komponennya memakai nama tipe domain, bukan nama modul** — karena itu
`ClaimReportInboxPage.tsx`, bukan `InboxLaporanKlaimPage.tsx`.
| LaporanKlaim | ClaimReport | berkas laporan kerugian yang masuk, sebelum menjadi klaim bernomor |
| Posisi (berkas) | Position | Outstanding · Not Registered · Not Transferred — **teks layar Pega, tidak diterjemahkan** |
| Asal (baris) | Origin | `pega` atau `claimpnc`; menyebut sistem yang menerbitkan baris |
| Kategori / Tab | Category | sembilan tab layar |
| Pencacah | Summary | delapan angka lencana dalam satu kueri |
| LiniBisnis | BusinessLine | dropdown "Bisnis" |
| Kanwil | Region | dropdown "Pilih Kanwil"; sumbernya `BRANCH.BASTERRITORY` |
| Pemanggil | Caller | identitas petugas yang mengirim permintaan |
| Umur berkas | AgingDays | kolom "Total Aging" |
| Pesan terakhir | LastMessage | kolom "Last message" pada ketiga tab komunikasi |
| Diserahkan | Transferred | menggantikan `statuslock_1` yang TIDAK NULL |
| RujukanPenugasan | AssignmentRef | isi `statuslock_1` apa adanya, untuk membuka berkasnya di Pega |
| Halaman | Pagination · Page | `Pagination` yang diminta, `Page` yang dikembalikan |

### Alias Pega yang TIDAK dibawa

Kelimanya menyebut hal yang sama sekali lain dari isinya (`D-19`):

| Kolom | Alias Pega lama | Nama di sini |
|---|---|---|
| `BUSINESSNAME` | `Kurir` | `BusinessName` |
| `br.branchname` | `UserAdmin` | `BranchName` |
| `pxcreateoperator` | `KodeCabang` | `CreatedBy` |
| `kodecabang_1` | `StatusKomunikasi` | `BranchCode` |
| `KETERANGAN_1` | `SIM` | `Reason` |
| `BookNo_1` | `Sender` | `ReferenceNumber` |
| `k.message` | `EmailPengirim` | `LastMessage` |

### Nama kolom basis data — tetap Indonesia

Tabel baru `POOLDATA.CPNC_LAPORAN_KLAIM` memakai nama kolom berbahasa Indonesia, mengikuti
`CPNC_PENGGUNA` dan `CPNC_SESI_AKTIF`: `NO_LAPORAN`, `NO_KLAIM`, `NAMA_PELAPOR`, `KODE_CABANG`,
`STS_DISERAHKAN`, `TGL_AGING`, `DIBUAT_OLEH`, `DIHAPUS_PADA`, dan seterusnya.

### Nama kueri `.sql`

Berawalan `claim_report_`, mengikuti nama domainnya:

`claim_report_source` · `claim_report_list_body` · `claim_report_count_body` ·
`claim_report_message_body` · `claim_report_message_count_body` · `claim_report_summary_body` ·
`claim_report_get_body` · `claim_report_region_list` · `claim_report_next_sequence` ·
`claim_report_insert` · `claim_report_check_table` · `claim_report_check_legacy_table`

Akhiran `_body` menandai fragmen yang **bukan kueri utuh** — ia disambung `claim_report_source`
lebih dulu. Lihat `sourced()` di `repo/sqlstore/query.go`.

### Nama field JSON — tetap Indonesia

Ia kontrak, bukan nama internal: `id`, `nomor_klaim`, `tertanggung`, `nama_bisnis`,
`tanggal_kejadian`, `umur_hari`, `nama_cabang`, `pesan_akhir`, `posisi`, `asal`, `rujukan_pega`,
`kategori`, `halaman`, `total_halaman`.

### Prop komponen bersama yang bertambah

| Prop | Komponen | Keterangan |
|---|---|---|
| `serverPaging` | `DataTable` | mematikan saring & urut internal, menggambar kaki halaman |
| `hideSearch` | `DataTable` | menyembunyikan kotak cari bawaan |
| `ServerPaging` | `DataTable` | tipe baru yang diekspor |

---

## Modul Inbox Manager Receive / PUCL (`MENU_ID 56`)

### Nama folder — Indonesia, mengikuti nama menu (`D-81`)

| Lapisan | Nama |
|---|---|
| Backend, folder + paket Go | `internal/inboxmanagerreceivepucl` |
| Frontend, folder | `src/modules/inbox-manager-receive-pucl` |
| Rute antarmuka | `/inbox-manager-receive-pucl` |
| Jalur API | `/api/inbox-manager-receive-pucl` |

Nama menunya "Inbox Manager Receive / PUCL". Garis miring dan spasinya dibuang pada nama
folder — Go tidak mengizinkan tanda hubung pada nama paket, dan garis miring bukan karakter
yang sah pada nama berkas.

**Nama harness-nya TIDAK dipakai.** `ReceiveDoucument_Harness` menyimpan salah ketik
(`Doucument`) dan hanya menyebut separuh isi layarnya — tab RCL/PUCL tidak tersirat sama
sekali di sana. Yang dipakai adalah nama butir menu, sesuai `D-81`.

Salah ketiknya tetap **dipertahankan apa adanya** di satu tempat: kunci peta
`MENU_ROUTES`, yang harus sama persis dengan `MENU_PROGRAM` di
`POOLDATA.M_MENU_APLIKASI_PNC`. Membetulkannya di sana akan membuat butir menunya tampak
belum tersedia selamanya.

### Nama tipe dan isian — Inggris (`D-80`)

| Properti Pega | Isian Go | Field JSON |
|---|---|---|
| `.pzInsKey` | `Reference` | `referensi` |
| `.pyID` | `CaseID` | `no_case` |
| `.ReceiveDocument.PolicyNo` / `.Policy.PolicyNo` | `PolicyNumber` | `no_polis` |
| `.ReceiveDocument.PNCCaseID` | `ClaimNumber` | `no_klaim_pnc` |
| `.ReceiveDocument.QQName` / `.Policy.QQName` | `InsuredName` | `nama_tertanggung` |
| `.ReceiveDocument.DateOfLoss` | `LossDate` | `tanggal_kejadian` |
| `.ReceiveDocument.TypeOfClaim` | `ClaimType` | `jenis_klaim` |
| `.ReceiveDocument.Sender` | `SenderName` | `nama_pengirim` |
| `.ReceiveDocument.ReceivedDate` | `DocumentReceivedDate` | `tanggal_terima_dokumen` |
| `.ReceiveDocument.NumberOfDocument` | `DocumentSheetCount` | `jumlah_lembar_dokumen` |
| `.pxCreateDateTime` | `InboxEntryAt` | `tanggal_masuk_inbox` |
| `.ClaimData.PUCLStatus.KomentarAnalisator` | `AnalystNote` | `deskripsi_analyst` |
| `.ClaimData.PUCLStatus.RCL_PUCL` | `Track` | `rcl_pucl` |
| `.ClaimData.PUCLStatus.StatusKlaim` | `TrackStatus` | `status_rcl_pucl` |
| `.ClaimData.PUCLStatus.TanggalCetakDokumenPUCL` | `LetterPrintedAt` | `tanggal_cetak_surat` |
| `.ClaimData.PUCLStatus.LamaKlaim` | `ClaimAge` | `lama_klaim` |
| `.ClaimData.PUCLStatus.StatusCase` | `ExpiryStatus` | `status_kadaluarsa` |

### Alias Pega yang sengaja TIDAK dibawa (`D-19`)

Lima kolom di modul ini dialiaskan dengan nama yang **tidak menyatakan isinya**, dan dua di
antaranya dialiaskan **berbeda di dua rule yang berbeda**:

| Kolom | Alias di `GetReminderPUCL` | Alias di `ReminderPUCL` |
|---|---|---|
| `QQNAME` | `NewTelpTertanggung` | `CABANG` |
| `KOMENTARANALISATOR_1` | `NoteKomite` | `LOGSEARCH` |
| `LAMAKLAIM_1` | `LOGSEEN` | `MODUL` |
| `STATUSKLAIM_1` | `StsAcceptance` | `STS_EMAIL` |
| `STATUSCASE_1` | — | `ClaimData.PUCLStatus.Stat25L` |

Baris pertama yang paling jelas: satu kolom berisi **nama tertanggung** dialiaskan "CABANG"
di satu rule dan "NewTelpTertanggung" di rule lain. Itu utang teknis §4.2 apa adanya.

Yang terakhir adalah nama yang **terpotong batas panjang alias Oracle** — `Stat25L` bukan
singkatan, melainkan sisa pemotongan.

### Dua nama kolom yang hanya berbeda satu huruf

| Kolom | Artinya |
|---|---|
| `STATUSKLAIM_1` | status jalur RCL/PUCL — **yang digambar layar ini** |
| `STATUSCLAIM_1` | Status Klaim ber-33 kode `1134`–`1166` (`R-06`) — tidak digambar |

Keduanya ada pada tabel yang sama dan keduanya muncul di Report Definition yang sama.
Menukarnya **tidak menghasilkan galat apa pun**.

### Nama kueri `.sql`

Berawalan nama tabnya, bukan nama modulnya — ketiganya membaca kombinasi tabel yang berbeda:

`list_receive_pa` · `list_receive_non_mbu` · `list_rclpucl` · `check_receive` ·
`check_rclpucl`

### Nama tipe frontend

| Tipe | Keterangan |
|---|---|
| `WorkItem` | satu baris, melayani ketiga tab |
| `WorkItemField` | `keyof WorkItem`, dipakai memilih sel |
| `TabColumn`, `Tab` | bentuk grid yang datang dari server |
| `PageInfo` | keterangan halaman |
| `MetadataResponse`, `ListResponse` | jawaban kedua endpoint |

Komponennya `ManagerReceivePUCLPage` dan `ReceivePUCLTabs` — memakai nama **tipe domain**,
bukan nama modul, mengikuti `AccountPage` pada `master-rekening` (`D-81`).
## Tambahan 2026-09-22 — form Input Receive Document

### Istilah domain baru

| Indonesia | Inggris | Catatan |
|---|---|---|
| Isian (form) | Detail | isi berkas laporan yang dikumpulkan form; dibedakan dari `ClaimReport` yang memuat kepala berkasnya |
| TanggalTerimaDokumen | ReceivedDate | |
| Pelapor | Reporter | `ReporterName`, `ReporterEmail`, `ReporterPhone` |
| Kurir | Courier | `CourierName` |
| EstimasiKerugian | EstimateValue | bertipe `Money`; **bukan** nilai klaim |
| LokasiKejadian | LossLocation | |
| Kronologis | Chronology | |
| RincianKerusakan | DamageDetail | |
| KeteranganBelumRegistrasi | NotRegisteredNote | |
| JumlahDokumen | DocumentCount | |
| Uang | Money | bilangan bulat SEN (`ADR-0016`); diulang dari modul `registrasi`, tidak diimpor |

### Kata kerja tambahan

| Indonesia | Inggris | Catatan |
|---|---|---|
| Simpan (form) | Save | lapisan usecase — dibedakan dari `Update` di lapisan repo |
| Perbarui | Update | lapisan repo |
| Terapkan | Apply | menuliskan isian form ke atas berkas yang sudah ada |
| Bersihkan | Clean | memangkas spasi sebelum diperiksa |
| Periksa | Check | menjalankan seluruh pemeriksaan sekaligus |

### Nama kolom basis data — tetap Indonesia

Migrasi 0004 menambah sepuluh kolom pada `POOLDATA.CPNC_LAPORAN_KLAIM`:
`TGL_TERIMA_DOKUMEN`, `EMAIL_PELAPOR`, `TLP_PELAPOR`, `NAMA_KURIR`, `NILAI_ESTIMASI`,
`LOKASI_KEJADIAN`, `KRONOLOGIS`, `RINCIAN_KERUSAKAN`, `KET_BELUM_REGISTRASI`, `JUMLAH_DOKUMEN`.

### Nama kueri `.sql` tambahan

`claim_report_update` — satu-satunya pernyataan pengubah pada modul ini.

### Nama field JSON — tetap Indonesia

`tanggal_terima_dokumen` · `nama_pelapor` · `email_pelapor` · `telepon_pelapor` · `nama_kurir` ·
`estimasi_kerugian` · `lokasi_kejadian` · `kronologis` · `rincian_kerusakan` ·
`keterangan_belum_registrasi` · `jumlah_dokumen` · `isian` · `dapat_disunting`

### Kode galat tambahan

| Kode | Artinya |
|---|---|
| `laporan_hanya_baca` | berkas ada, tetapi penulisnya masih Pega selama masa paralel |

### Nama komponen

`ClaimReportFormPage.tsx` — nama **tipe domain**, bukan nama modul, mengikuti aturan yang sama
dengan `ClaimReportInboxPage.tsx` dan `AccountPage.tsx`.

---

## Tambahan 2026-09-22 — penerjemahan cabang klaim

Sesi kesebelas menambah satu seam dan **menghapus** satu field. Yang dihapus justru lebih penting
dicatat daripada yang ditambah.

### Nama yang dihapus, dan kenapa

| Nama lama | Keadaan | Alasan |
|---|---|---|
| `inboxlaporanklaim.Caller.BranchCode` | **dihapus** | Namanya tidak mencerminkan isinya. Ia diisi dari `auth.User.BranchCode` — kode penempatan pegawai dari HCQ — sementara yang dibutuhkan layar ini adalah kunci baris `POOLDATA.BRANCH`. Dua sistem penomoran berbeda dengan nama yang sama |

Ini `D-19` yang berlaku pada kode baru, bukan pada warisan Pega: **nama yang tidak mencerminkan isi
adalah utang teknis**, dan memperbaiki isinya sambil membiarkan namanya berarti orang berikutnya
akan mengisinya salah lagi.

### Nama baru

| Nama | Lapisan | Alasan |
|---|---|---|
| `BranchResolver` | seam (`seam.go`) | Peran, bukan mekanisme. Ia menjawab "cabang klaim petugas ini apa", bukan "jalankan kueri HRD" |
| `Resolve(ctx, login)` | seam | Kata kerja yang menyatakan penerjemahan, bukan pencarian. `Find`/`Get` menyiratkan kegagalan adalah ketiadaan baris; di sini "tidak terdaftar" dan "tidak terbaca" adalah dua hal berbeda |
| `resolved bool` | seam | Menjadikan "terdaftar atau tidak" **bukan galat**, sehingga pemanggil tidak perlu membaca isi `error` untuk mengetahuinya |
| `branch_of_login` | `branch.sql` | Dinamai menurut **pertanyaan yang dijawabnya**, sejalan dengan kueri lain modul ini (`claim_report_list_body`, `claim_report_summary_body`) |
| `BranchScope` / `BranchResolved` | `usecase.ListResult` | Bukan `BranchCode`. `Scope` menyatakan ia **batas daftar**, bukan atribut pemanggil — perbedaan yang justru menjadi sebab cacat sesi ini |
| `checkClaimReportBranch` | `cmd/claimpnc/check.go` | Mengikuti pola `checkClaimReport`, `checkClaimStatus`, `checkLoginTable` |

### Nama kontrak API — tetap Indonesia (`D-80`)

| Field | Arti |
|---|---|
| `batas_cabang` | cabang yang membatasi daftar; kosong berarti seluruh cabang |
| `cabang_terbaca` | `false` berarti cabang petugas tidak dapat ditentukan |

**`batas_cabang`, bukan `kode_cabang`.** Baris hasil sudah punya `kode_cabang` yang artinya "cabang
milik berkas ini". Memakai nama yang sama untuk "cabang yang menyaring daftar" akan mengulang
persis jenis kekeliruan yang sedang diperbaiki — dua hal berbeda dengan satu nama.

### Padanan istilah yang kini punya arti tepat

| Istilah | Arti dalam sistem ini | Jangan tertukar dengan |
|---|---|---|
| **kode cabang klaim** | kunci baris `POOLDATA.BRANCH.ID`, diturunkan dari login lewat HRD dan `LST_USER_ASURANSI` | **kode penempatan pegawai** dari HCQ (`EmpResponse.Placement.BranchCode`) — bukan nilai yang sama, dan bukan tabel yang sama |
| **batas cabang** | cabang yang menyaring daftar seorang petugas | **cabang berkas** — cabang tempat sebuah berkas dicatat |

Keduanya diusulkan masuk `CONTEXT.md`, dan tidak disunting sendiri: `CONTEXT.md` adalah dokumen
Steering yang perubahannya ditulis sebagai keputusan (`D-72`), bukan ditambahkan menyusul.

### Koreksi pada sesi yang sama — cabang menjadi batas kewenangan

| Nama | Keadaan | Alasan |
|---|---|---|
| `cabang_terbaca` | **dihapus dari kontrak** | Sejak cabang yang tidak terbaca ditolak, ia tidak dapat lagi bernilai `false`. Field yang hanya pernah bernilai satu macam mengundang cabang penanganan untuk keadaan yang tidak pernah terjadi |
| `ErrBranchUnknown` | baru | Cabang petugas tidak dapat ditentukan — urusan **data pegawai** |
| `ErrBranchUnreadable` | baru | Sumber cabang tidak dapat dibaca — urusan **infrastruktur**. Dipisahkan meski akibatnya sama, karena perbaikannya berbeda dan yang satu menimpa seluruh petugas |
| `requireBranch` | baru (`usecase`) | `require`, bukan `resolve`: namanya menyatakan bahwa kegagalannya MENGHENTIKAN permintaan. Nama lamanya, `resolveBranch`, menyiratkan kegagalan boleh dilanjutkan — dan itu memang perilakunya sebelum dikoreksi |
| `wideList` | baru (uji) | Pandangan sah terluas — satu kanwil. Menggantikan pemakaian "cabang tidak terbaca" sebagai pintu belakang |

**Kode galat kontrak API — tetap Indonesia (`D-80`)**

| Kode | HTTP | Arti |
|---|---|---|
| `cabang_tidak_dikenali` | 403 | login petugas belum terdaftar pada cabangnya |
| `sumber_cabang_tidak_terbaca` | 503 | sumber data cabang sedang tidak dapat dibaca |
| Backend | tidak ada folder baru — ia bagian `internal/komite/` (`D-81`) |
| Frontend | `src/modules/inbox-komite/` — nama modul yang disebut Work Owner |

Berkas frontend berbahasa Inggris sesuai `D-80`: `InboxKomitePage.tsx`, `InboxTabs.tsx`,
`DecisionPanel.tsx`. "Inbox Komite" pada nama folder adalah **nama modulnya**, dan itu satu-satunya
yang berbahasa Indonesia.

---

## Modul Inbox Close Claim (`MENU_ID 59`)

### Nama folder — Indonesia, mengikuti nama menu (`D-81`)

| Lapisan | Nama |
|---|---|
| Backend, folder + paket Go | `internal/inboxcloseclaim` |
| Frontend, folder | `src/modules/inbox-close-claim` |
| Rute antarmuka | `/inbox-close-claim` |
| Jalur API | `/api/inbox-close-claim` |

Nama menunya sendiri berbahasa Inggris — "Inbox Close Claim" — sehingga nama folder dan nama
modul kebetulan sama bentuknya. Yang mengikat tetap `D-81`: nama folder mengikuti **nama
butir menu**, bukan nama harness-nya.

Harness-nya sendiri, `InboxCloseClaim_Harness`, **tidak ada di export**. Nama yang dipakai
karena itu diambil dari `Database/m_menu_aplikasi_pnc.csv:59` dan dari judul yang tertulis di
dalam `Section/InboxManagerReopen1_Sec-Section.xml`.

> Perhatikan: nama SECTION-nya menyebut "ManagerReopen", bukan "CloseClaim". Modul ini
> sengaja TIDAK dinamai menurut section itu — yang dibaca Work Owner dan yang tertulis di
> menu adalah "Inbox Close Claim".

### Alias Pega yang sengaja TIDAK dibawa (`D-19`)

Layar ini memuat alias paling menyesatkan di antara modul mana pun sejauh ini: **tiga dari
sebelas kolomnya** dialias dengan nama yang artinya berlawanan dengan isinya.

| Alias kueri lama | Isinya sebenarnya | Nama di sini |
|---|---|---|
| `ReinsurerName` | PIC Teknik (`USERTEKNIS_1`) | `TechnicalPIC` |
| `MOName` | Admin PNC (`PXCREATEOPNAME`) | `AdminPNC` |
| `CoverNo` | **tanggal kejadian** (`DATEOFLOSS_1`) | `LossDate` |
| `CustomerName` | nama tertanggung (`QQNAME`) | `InsuredName` |
| `CaseID` | kunci teknis (`PZINSKEY`) | `ClaimID` |
| `CaseIDView` | nomor klaim (`PYID`) | `ClaimNumber` |
| `Country` | pencacah hasil pada kueri hitung | — |

Enam properti `TempFilter.*` pada activity lama juga tidak dibawa namanya, dan ketujuhnya
sama menyesatkan: `TempFilter.CaseID` berisi penyaring **No Polis**, `TempFilter.City` berisi
penyaring **No Klaim**, `TempFilter.CityID` berisi penyaring **PIC**, `TempFilter.District`
berisi penyaring **transfer kasir**, dan `TempFilter.DistrictID` berisi penyaring **status
bayar**.

### Nama tipe dan isian — Inggris (`D-80`)

| Pega | Di sini |
|---|---|
| baris klaim tutup | `ClosedClaim` |
| permintaan atas klaim | `ClaimRequest` |
| jenis permintaan | `RequestKind` — `RequestReopen`, `RequestCopy` |
| keadaan permintaan | `RequestStatus` — `RequestPending`, `RequestExecuted`, `RequestCanceled` |
| penyaring lini bisnis | `BusinessLine` |
| penyaring transfer kasir | `TransferStatus` |
| penyaring status bayar | `PaymentStatus` |

### Nama kolom basis data — tetap Indonesia

Tabel baru `POOLDATA.CPNC_PERMINTAAN_KLAIM` memakai nama kolom Indonesia, mengikuti
pengecualian `D-80` yang sama dengan `CPNC_KOMITE_KEPUTUSAN`:

`ID` · `JENIS` · `CASE_ID` · `NOMOR_KLAIM` · `ALASAN` · `STATUS` · `EFEK_STATUS_KERJA` ·
`EFEK_STATUS_KLAIM` · `LINGKUP_SALIN` · `ACTOR_LOGIN` · `ACTOR_NAMA` · `PADA`

Nilai kolom `JENIS` dan `STATUS` juga berbahasa Indonesia (`reopen`/`salin`,
`menunggu`/`dijalankan`/`dibatalkan`) karena ia sama dengan nilai pada kontrak API.

### Nama kueri `.sql`

`close_claim_list` · `close_claim_count` · `close_claim_exists` · `request_insert` ·
`request_pending_for` · `request_check_table`

### Nama field JSON — tetap Indonesia

`klaim_id` · `nomor_klaim` · `nomor_polis` · `nama_tertanggung` · `nama_bisnis` ·
`sumber_bisnis` · `nama_cabang` · `pic_teknik` · `admin_pnc` · `tanggal_pendaftaran` ·
`tanggal_kejadian` · `tanggal_tutup` · `lama_hari` · `status_tampil` · `status_klaim_kode` ·
`status_klaim_label` · `sudah_transfer` · `permintaan_tertunda` · `permintaan_terbaca` ·
`selisih_terencana`

### Nama tipe frontend

`KlaimTutup` · `PermintaanTertunda` · `DaftarResponse` · `PenyaringResponse` ·
`PenyaringKlaimTutup` · `JenisPermintaan` · `PermintaanResponse`

Komponennya: `CloseClaimPage` · `RequestDialog` · `PanelPenyaring` · `BarisTindakan` ·
`LamaKlaim` · `SelisihTerencana`

---

## Modul Archive Dokumen Klaim (`MENU_ID 77`, `archivedokumenklaim`)

Layar lama memakai properti klipboard yang sudah ada alih-alih membuat properti baru.
Akibatnya **sebelas dari lima belas** nama properti tidak menyatakan isinya — utang teknis
`03-CURRENT-ARCHITECTURE.md` §4.2 dalam bentuk paling pekat setelah View History Claim.

Sumbernya `RDB List/SearchDataArchiveFillingCase-SQL.xml` (alias kolomnya) dan caption
harness `PNCArchiveDokumen` (arti bagi pengguna).

### Grid ARCHIVE FILE KLAIM — `POOLDATA.T_CLAIM_ARCHIVE_FILE`

| Arti bagi pengguna | Properti klipboard Pega | Kolom sebenarnya | Nama di kode | Field JSON |
|---|---|---|---|---|
| kunci baris | `.IDMasterTONP` (!) | `ID_ARCHIVE` | `ID` | `id` |
| No Klaim | `.CaseID` | `NOKLAIM` | `ClaimNumber` | `nomor_klaim` |
| No Polis | `.PolicyNo` | `NOPOLIS` | `PolicyNumber` | `nomor_polis` |
| Nama Tertanggung | `.NIK` (!) | `TERTANGGUNG` | `InsuredName` | `nama_tertanggung` |
| Tgl Kejadian | `.DateOfLoss` | `DOL` | `LossDate` | `tanggal_kejadian` |
| PIC Teknis | `.UserTeknis` | `PICTEKNIK` | `TechnicalPIC` | `pic_teknis` |
| Tgl Terima Dokumen | `.TanggalCetakDLA` (!) | `TGLTERIMADOK` | `DocumentReceivedDate` | `tanggal_terima_dokumen` |
| TGL INPUT | `.TanggalAnalystSendRCL` (!) | `TGLINPUT` | `InputDate` | `tanggal_input` |
| Jumlah Lembar | `.AgingAmount` (!) | `JUMLAHLEMBAR` | `SheetCount` | `jumlah_lembar` |
| Tipe Dokumen | `.RWID` (!) | `TIPEDOK` | `DocumentTypeCode` | `kode_tipe_dokumen` |
| Jenis Dokumen | `.TelpTertanggung` (!) | `JENISDOK` | `DocumentKindCode` | `kode_jenis_dokumen` |
| Nama BOX | `.CABANG` (!) | `NAMABOX` | `BoxName` | `nama_box` |
| Kode Filling | `.KodeCabang` (!) | `KODEFILLING` | `FillingCode` | `kode_filling` |
| User Input | `.UserName` | `USERINPUT` | `InputUser` | `user_input` |
| Tanggal Kirim Dok | `.TanggalAI` (!) | `TGLKIRIMDOK` | `SentDate` | `tanggal_kirim_dokumen` |

Tanda (!) menandai nama yang sama sekali tidak menyatakan isinya.

**Enam yang mudah tertukar, dan perlu diucapkan terpisah:**

- `.NIK` **bukan** NIK. Isinya nama tertanggung.
- `.CABANG` **bukan** cabang. Isinya nama boks arsip.
- `.KodeCabang` **bukan** kode cabang. Isinya kode filling.
- `.TelpTertanggung` **bukan** nomor telepon. Isinya kode jenis dokumen.
- `.TanggalCetakDLA` **bukan** tanggal cetak DLA. Isinya tanggal terima dokumen.
- `.AgingAmount` **bukan** nilai uang. Isinya jumlah lembar kertas.

### Enam kolom yang tidak pernah muncul di grid, tetapi menentukan perilaku

| Kolom | Nama di kode | Perannya |
|---|---|---|
| `GROUPPANEL` | `GroupPanel` | menentukan **siapa melihat barisnya** di daftar kirim ke cabang |
| `CABANGSTATUS` | `BranchStatus` | `'0'` belum dikirim, `'1'` sudah |
| `KODECABANG` | `BranchCode` (pada `Draft`) | tidak dibaca satu pun rule di export |
| `KODESERVICE` | `ServiceCode` | jawaban `ResponseCode` sistem Arsip |
| `NOTESERVICE` | `ServiceNote` | jawaban `ResponseMessage` sistem Arsip |
| `HITARCHIVE` | `Receipt.Request` | **badan permintaan** yang dikirimkan, bukan jawaban |

Yang terakhir itu mudah salah dibaca dari namanya: "hit archive" terdengar seperti hasil,
tetapi isinya jejak apa yang dikirim.

### Grid Input Data Archive — `POOLDATA.T_CLAIM_PNC`

Kueri lamanya memakai lapisan alias kedua yang berbeda lagi dari grid di atas.

| Arti bagi pengguna | Properti klipboard | Kolom sebenarnya | Nama di kode |
|---|---|---|---|
| No Klaim | `.Notes` (!) | `CLAIMNO` | `Number` |
| No Polis | `.BranchOfBank` (!) | `NOPOLIS` | `PolicyNumber` |
| Nama Tertanggung | `.NameOfBank` (!) | `QQNAME` | `InsuredName` |
| Tgl Kejadian | `.DateTransferPajak` (!) | `DATEOFLOSS` | `LossDate` |
| Bisnis | `.Receiver` (!) | `BUSINESSNAME` | `BusinessName` |
| Cabang | `.BatasLapor` (!) | `BRANCHNAME` | `BranchName` |
| Status | `.StatusClaim` (!) | `STATUSWORK` | `WorkStatus` |
| Posisi Klaim | `.BranchOfBank2` (!) | `V_STS_CLAIM.LSC_NOTE` | `ClaimPosition` |
| Tanggal Close | `.AcceptedDate` (!) | `CLOSECLAIMDATE` | `CloseDate` |
| Catatan Close | `.NoteAkseptasi` (!) | `CLOSECLAIMNOTE` | `CloseNote` |
| PIC Teknis | `.CAUSE_OF_LOSS` (!) | `PICTEKNIK` | `TechnicalPIC` |
| Group Panel | `.Initial` (!) | `GROUPPANEL` | `GroupPanel` |

**Dua belas dari dua belas menyesatkan.** `.CAUSE_OF_LOSS` untuk PIC Teknis dan
`.StatusClaim` untuk status alur kerja adalah dua yang paling berbahaya: keduanya
**bertabrakan dengan konsep lain yang benar-benar ada** di domain ini.

### Istilah yang dipakai di layar, dan yang tidak

| Yang dipakai | Yang TIDAK dipakai | Alasan |
|---|---|---|
| **Berkas arsip** | "dokumen" | "dokumen" sudah dipakai modul Daftar Tipe Dokumen untuk hal lain |
| **Nama BOX** | "kotak", "boks" | teks yang dibaca pengguna di layar lama (`D-13`) |
| **Kode Filling** | "kode arsip", "kode berkas" | idem |
| **Kirim ke Cabang** | "kirim ke arsip" | nama yang dipakai activity lama, meski tujuannya sistem Arsip |

Yang terakhir patut disadari: bagian itu bernama "Kirim ke Cabang" tetapi tidak mengirim
apa pun **ke cabang** — ia menembak satu layanan REST bernama injeksi arsip. Namanya
dipertahankan karena itu yang dikenal pengguna, dan perbedaannya dicatat di sini alih-alih
diperbaiki diam-diam.
## Tambahan 2026-09-23 — modul Input Req Protection (`inputreqprotection`) dan Inbox Accept Open Protection (`inboxacceptopenprotection`)

### Nama modul: dua butir menu, dan namanya bersilang

`D-81` menetapkan nama folder modul mengikuti nama yang dipakai Work Owner. Di sini nama itu
**bersilang** di Pega, sehingga aturannya tidak dapat diterapkan mentah pada keduanya:

| Butir menu (`pyCaptionPrompt`) | Harness | Judul di dalam harness | Nama modul |
|---|---|---|---|
| Input Req Protection | `InputReqProtection_Harness` | "Inbox Open Protection" | `inputreqprotection` |
| Inbox Open Protection | `InputProtection_Harness` | **"Inbox Accept Open Protection"** | `inboxacceptopenprotection` |

Modul pertama memakai nama **butir menunya**; modul kedua memakai **judul harness-nya**,
karena nama butir menunya sudah dipakai judul layar modul pertama. Memakai nama butir menu
untuk keduanya akan menghasilkan dua modul bernama sama.

| Lapisan | Modul 1 | Modul 2 |
|---|---|---|
| Paket Go | `internal/inputreqprotection` | `internal/inboxacceptopenprotection` |
| Folder frontend | `src/modules/input-req-protection` | `src/modules/inbox-accept-open-protection` |
| Rute API | `/api/input-req-protection` | `/api/inbox-accept-open-protection` |
| Rute layar | `/input-req-protection` | `/inbox-accept-open-protection` |

### Istilah: "Proteksi" berarti DUA hal berbeda di repositori ini

Ini jebakan nama yang paling mudah menjatuhkan orang berikutnya.

| Istilah | Artinya | Tabelnya | Modul |
|---|---|---|---|
| **Open Protection** (Buka Proteksi) | permintaan pembukaan proteksi atas sebuah polis — konsep asuransi | `T_CLAIM_OPENPROTECTION` (baru) | `inputreqprotection`, `inboxacceptopenprotection` |
| **Proteksi Data** | jatah pencarian data nasabah — kewenangan masking | `MST_PROTEKSI_DATA_PNC` | `mastermasking`, `riwayatklaim` |

Keduanya **tidak berhubungan sama sekali**. Tabel `MST_PROTEKSI_DATA_PNC` berkolom `LOGIN`,
`PASSWORD`, `STS_KTP`, `LOGSEARCH` — ia master kewenangan, bukan proteksi polis.

### Penamaan properti Pega ke nama Inggris

`D-80` menetapkan nama di dalam kode berbahasa Inggris; nama kolom basis data dan nama field
JSON tetap Indonesia.

| Properti Pega | Nama Go | Kolom (usulan) | Field JSON | Kolom layar |
|---|---|---|---|---|
| `.pyID` | `Number` | `NO_PROTEKSI` | `nomor_proteksi` | No Proteksi |
| `.PolicyNo` | `PolicyNumber` | `NO_POLIS` | `nomor_polis` | No Polis |
| `.CaseID` | `ClaimNumber` | `NO_KLAIM` | `nomor_klaim` | No Klaim |
| `.PNCCaseID` | `ClaimReference` | `KLAIM_REF` | `referensi_klaim` | — |
| `.TypeProtection` | `Type` | `TIPE_PROTEKSI` | `tipe_proteksi` | Tipe Proteksi |
| `.InputDate` | `InputDate` | `TANGGAL_INPUT` | `tanggal_proteksi` | Tanggal Proteksi Dibuat |
| `.Keterangan` | `Note` | `KETERANGAN` | `keterangan` | Keterangan |
| `.pxCreateOpName` | `CreatedBy` | `DIBUAT_OLEH` | `user_create` | User Create |
| `.AcceptStatus` | `AcceptStatus` | `STATUS_AKSEPTASI` | `status_akseptasi` | — |
| `.AcceptDate` | `AcceptedAt` | `TANGGAL_AKSEPTASI` | `tanggal_akseptasi` | — |
| `.AcceptOpName` | `AcceptedBy` | `DIAKSEP_OLEH` | `diaksep_oleh` | — |
| `.IsUsedPNC` | `UsedByClaim` | `DIPAKAI_KLAIM` | — | — |
| `.ClaimDataProtect.BeforeDateOfLoss` | `ChangeDetail.LossDateBefore` | `DOL_SEBELUM` | `dol_sebelum` | Current Date Of Loss |
| `.ClaimDataProtect.DateOfLoss` | `ChangeDetail.LossDateAfter` | `DOL_BARU` | `dol_baru` | Next Date Of Loss |
| `.ClaimDataProtect.CauseOfLossID` | `ChangeDetail.CauseOfLossID` | `COL_ID` | `penyebab_kerugian` | Cause Of Loss Sebelumnya |
| `.ClaimDataProtect.IDMasterTONP` | `ChangeDetail.CauseOfLossMasterID` | `COL_MASTER_ID` | `penyebab_kerugian_master` | Cause Of Loss Dipilih |

**`.CaseID` dan `.PNCCaseID` mudah tertukar dan artinya berbeda:** yang pertama nomor klaim
yang **diketik** pengguna, yang kedua klaim yang benar-benar **ditemukan**. Sistem lama
menolak penyimpanan saat yang kedua kosong, dengan pesan "Silakan Tulis dan Cari Ulang No
Klaim" — mengetik saja tidak cukup.

### Nomor proteksi

| Bentuk | Asal | Contoh |
|---|---|---|
| `OPC-XXX` | warisan Pega, dibaca apa adanya | `OPC-201` |
| `OPCN.YY.xxxx` | terbitan aplikasi ini (keputusan Work Owner 2026-09-23) | `OPCN.26.0001` |

Sejajar dengan `PNC-xxxx` versus `PNCN.YY.xxxx` pada nomor klaim (`D-22`, `D-71`).

---

## Tambahan 2026-09-24 — modul Inbox Komunikasi Cabang (`inboxkomunikasicabang`)

Nama modul mengikuti `D-81`: folder backend `internal/inboxkomunikasicabang` (tanpa tanda
hubung, karena Go tidak mengizinkannya), folder frontend
`src/modules/inbox-komunikasi-cabang`. Namanya **tidak dikarang** — ia tertulis di
`Database/m_menu_aplikasi_pnc.csv` baris 65 sebagai `MENU_ID 70`, **"Inbox Komunikasi
Cabang"**. Isinya berbahasa Inggris sesuai `D-80`.

### Alias Pega yang TIDAK dibawa — dan kenapa hampir seluruhnya menyesatkan

Kueri layar ini mengalias hampir setiap kolom dengan nama yang menyebut hal lain. Tidak satu
pun dibawa (`D-19`):

| Alias Pega | Kolom sebenarnya | Isi sebenarnya | Nama di sini |
|---|---|---|---|
| `CloseClaimDate` | `CREATEDDATE` | tanggal pesan — tidak ada klaim yang ditutup | `CreatedAt` |
| `Email` | `MESSAGE` | **isi pesan**, bukan alamat surel | `Message` |
| `UserTeknis` | `SENDER` | Operator ID pengirim | `SenderOperator` |
| `pzInsKey` | `CASEID` | **penanda kanal** (`CABANG`), bukan kunci objek kerja Pega | — (tidak dipilih) |
| `ClaimNo` | `KOMUNIKASIID` | **nomor percakapan**, bukan nomor klaim | `ID` |
| `CloseClaimNote` | `REPLYMESSAGE` | isi balasan | `Reply` |
| `StatusClaim` | `KOMUNIKASISTATUS` | status percakapan — **bukan** Status Klaim ber-33 kode (`R-06`) | `Status` |
| `AnalystTransferDate` | `CREATEDATEREPLY` | tanggal balasan | `RepliedAt` |
| `UserAdmin` | `REPLYFROMNAME` | nama penjawab | `ReplierName` |
| `UserName` | `COMMUNICATE_FROM` | kode asal | `SenderOrigin` |
| `UserTeknisEmail` | `COMMUNICATE_TO` | kode tujuan | `RecipientOrigin` |

Tiga baris pertama patut disebut khusus: tabel ini **tidak memuat nomor klaim sama sekali**,
`Email` bukan alamat, dan `pzInsKey` bukan kunci Pega.

### Alias lampiran — tujuh nama, tujuh kali salah

Dari `RDB List/GetDocumentKomunikasi1-SQL.xml`, contoh utang teknis §4.2 paling padat yang
ditemukan sejauh ini:

| Alias Pega | Kolom | Isi | Nama di sini |
|---|---|---|---|
| `BranchCode` | `A.KOMUNIKASI_ID` | nomor percakapan | — (penyaring) |
| `BranchName` | `A.CATEGORYID` | kode kategori dokumen | — (gabungan) |
| `ClaimID` | `A.TYPEID` | kode jenis dokumen | — (gabungan) |
| `ObjectName` | `C.DETAIL_DOCUMENT` | nama rincian dokumen | `DetailName` |
| `ObjectLocation` | `D.TYPE_DOCUMENT` | nama jenis dokumen | `TypeName` |
| `Comment` | `A.NOTE` | catatan | `Note` |
| `Date` | `A.UPLOADDATE` | tanggal unggah | `UploadedAt` |
| `IdCompliance` | `A.DOCUMENTID` | id dokumen di penyimpanan | `DocumentID` |

### Istilah domain baru

| Indonesia | Inggris | Catatan |
|---|---|---|
| Percakapan | `Conversation` | satu baris grid: satu pesan beserta balasannya |
| Utas | `ThreadMessage` · `ConversationDetail` | isi layar Detail Komunikasi |
| Lampiran | `Attachment` | metadata dokumen; berkasnya di penyimpanan lain (`D-16`) |
| Kanal percakapan | `CaseOpen` · `CaseClosed` | nilai `CASEID`: `CABANG` dan `CABANG SELESAI` |
| Asal / Tujuan | `SenderOrigin` · `RecipientOrigin` | sudah diterjemahkan; kode mentahnya tidak dibawa ke layar |
| Batas cabang | `BranchFilter` | kode, penanda kantor pusat, dan penanda **terbaca** |
| Penerjemah cabang | `BranchResolver` | seam ke HRD lewat DB Link (`D-25`, `R-03`) |
| Pencacah | `Summary` | dua angka di atas grid, pengganti diagram lingkaran |

**Istilah yang sengaja TIDAK dipakai:** `SenderName`. Kolomnya ada (`SENDERNAME`) tetapi
**tidak pernah sampai ke layar** — alias `UserName` yang sama dipakai dua kali dalam satu
`SELECT`, dan yang menang adalah `COMMUNICATE_FROM`. Membawanya berarti menyimpan nama yang
harus dijelaskan mengapa diabaikan.

### Dua konstanta yang MUDAH tertukar, dan akibatnya fatal

| Konstanta | Nilai | Kolom |
|---|---|---|
| `HeadOfficeBranch` | `100081` | `POOLDATA.BRANCH.ID` |
| `HeadOfficeCode` | `1` | `COMMUNICATE_FROM` / `COMMUNICATE_TO` |

Keduanya berarti "kantor pusat" tetapi hidup di kolom yang berbeda. Menukarnya menghasilkan
**daftar kosong tanpa satu pun galat**. `ResolveBranch` menerjemahkan yang pertama menjadi yang
kedua, dan satu uji menjaganya
(`TestHeadOfficeIsFilteredByChannelCodeNotByBranchCode`).

### Nama field JSON — tetap Indonesia

Ia **kontrak**, bukan nama internal (`D-80`):

`komunikasi` · `tanggal` · `pengirim` · `asal` · `operator_pengirim` · `pesan` ·
`jawaban_terakhir` · `penjawab` · `tujuan` · `status_register` · `tanggal_jawaban` ·
`ringkasan` · `belum_dijawab` · `sudah_dijawab` · `batas_cabang` · `kantor_pusat` ·
`terbaca` · `lampiran` · `jenis_dokumen` · `rincian_dokumen` · `sudah_diunggah` ·
`tindakan_masih_di_pega`

`komunikasi`, bukan `no_klaim` — tabel ini tidak memuat nomor klaim sama sekali.

### Kode galat — tetap Indonesia

`validasi_gagal` · `profil_pemanggil_tidak_lengkap` · `sumber_cabang_tidak_terbaca` ·
`belum_tersedia` · `komunikasi_tidak_ditemukan` · `galat_internal`

`sumber_cabang_tidak_terbaca` dipisahkan dari `profil_pemanggil_tidak_lengkap` dengan sengaja:
yang satu menimpa **semua orang** dan sementara (503), yang satu menimpa **satu orang** (409).

### Nama kueri `.sql`

Berawalan menurut perannya, bukan menurut nama tabelnya:

`list_not_answered` · `list_answered` · `count_answered` · `count_not_answered` ·
`detail_thread` · `detail_attachments` · `check_table` · `check_attachment_table` ·
`branch_of_login`

`branch_of_login` **sengaja sama namanya** dengan milik modul Inbox Laporan Klaim, dan
berkasnya **disalin, bukan diimpor**: seam dideklarasikan di paket yang memakainya
(`08-TECHNICAL-STRATEGY.md` §2 aturan 2), sehingga kedua modul dapat berpindah ke API
pengganti DB Link pada waktu yang berbeda.

### Nama komponen frontend

Komponen memakai nama **konsep layarnya**, bukan nama modul:

| Berkas | Komponen |
|---|---|
| `KomunikasiCabangPage.tsx` | `KomunikasiCabangPage` |
| `KomunikasiCabangTabs.tsx` | `KomunikasiCabangTabs` |
| `ConversationDetail.tsx` | `ConversationDetail` |

Yang ketiga memakai nama **tipe domain**, mengikuti preseden `AccountPage.tsx` pada modul
`master-rekening` — ia menggambar sebuah `ConversationDetailResponse`, bukan sebuah layar
bernama modul.

### Nama uji

Berbahasa **Inggris** di Go, **Indonesia** di Vitest, mengikuti pola yang sudah berlaku, dan
menyebutkan **aturannya** alih-alih nama fungsinya:

```go
func TestHeadOfficeIsFilteredByChannelCodeNotByBranchCode(t *testing.T)
func TestUnreadableBranchIsServedAsHeadOfficeButMarkedUnresolved(t *testing.T)
func TestSenderOriginKeepsBranchCodeButRecipientOriginDoesNot(t *testing.T)
func TestARepliedConversationWithoutARecordedReplierAppearsButIsCountedNowhere(t *testing.T)
```

```ts
it('memperingatkan bila kode cabang pemanggil tidak terbaca', ...)
it('menggambar TIGA kolom pada tab "Belum Dijawab", tanpa kolom balasan', ...)
it('menyatakan bahwa membalas belum tersedia alih-alih menggambar kotak isian', ...)
```

### Tambahan aksi tulis (2026-09-24)

Nama yang lahir bersama tombol "Balas" dan "Selesai Komunikasi".

| Pega | Kode baru | Sebab |
|---|---|---|
| `Param.Pesan` | `ReplyInput.Message` | — |
| `Param.IDKomunikasi` | `ReplyInput.ID` | — |
| `tempReply.ADDRESS` | `ReplyCommand.Message` | nama Pega menyebut "ADDRESS" untuk isi pesan; tidak dibawa (`D-19`) |
| `tempReply.M_SURVEY_ID` | `ReplyCommand.ID` | menyebut "SURVEY" untuk nomor percakapan; tidak dibawa (`D-19`) |
| `tempReply.NAME` / `Param.kodecabang` | `inboxkomunikasicabang.CaseOpen` | **perangkap penamaan**: isinya `CASEID`, bukan kode cabang |
| `temp.AnalystTransferDate` | `ReplyCommand.RepliedAt` | menyebut "Analyst" untuk waktu balasan; tidak dibawa |
| `OperatorID.pyUserIdentifier` | `Caller.Login` | — |
| `OperatorID.pyUserName` | `Caller.Name` | — |
| `REPLYFROM` (kolom) | `Conversation.ReplyFrom` | kolom basis data tetap Indonesia (`D-80`) |
| `REPLYFROMNAME` (kolom) | `Conversation.ReplierName` | — |
| `KOMUNIKASISTATUS = '1'` | `StatusAnswered` | artinya baru diketahui 2026-09-24 |
| `CASEID = 'CABANG SELESAI'` | `CaseClosed` | — |
| `EndKomunikasiCabang` (activity) | `Service.Finish` · `Repo.Finish` | — |
| `PNCReplyMessageCabang` (activity) | `Service.Reply` · `Repo.Reply` | — |

Nama field JSON tetap berbahasa Indonesia — ia kontrak (`D-80`):

| Field JSON | Isinya |
|---|---|
| `pesan` (badan permintaan balas) | isi balasan |
| `komunikasi` (jawaban aksi) | nomor percakapan yang dikenai tindakan |
| `pesan` (jawaban aksi) | kalimat siap baca tentang apa yang terjadi |
| `balas_tersedia` | **menggantikan** `tindakan_masih_di_pega`, artinya KEBALIKAN |

Alamat rutenya:

```
POST /api/inbox-komunikasi-cabang/komunikasi/{komunikasi}/balas
POST /api/inbox-komunikasi-cabang/komunikasi/{komunikasi}/selesai
```

Nama uji yang lahir bersamanya:

```go
func TestAReplyMovesTheConversationFromOneTabToTheOther(t *testing.T)
func TestAReplyToAClosedConversationIsRefused(t *testing.T)
func TestFinishingATwiceIsRefusedInsteadOfSilentlySucceeding(t *testing.T)
func TestReplyMeasuresLengthInCharactersNotBytes(t *testing.T)
func TestEveryWriteStatementRefusesAClosedConversation(t *testing.T)
func TestWriteStatementsCarryNoTableAlias(t *testing.T)
```

```ts
it('menggambar kotak "Masukkan Balasan" yang benar-benar dapat dipakai', ...)
it('meminta penegasan lebih dulu, tidak langsung menutup percakapan', ...)
it('menahan kalimat yang sudah diketik bila balasannya ditolak', ...)
```

> Uji `it('menyatakan bahwa membalas belum tersedia alih-alih menggambar kotak isian', ...)`
> yang tercantum di atasnya **dihapus** pada 2026-09-24: pernyataannya tidak lagi benar.

### Tambahan "Kirim Pesan" (2026-09-24, lanjutan)

Nama yang lahir bersama form pembuatan percakapan baru. Tujuh di antaranya menggantikan nama
Pega yang **seluruhnya** menyebut hal lain.

| Pega | Kode baru | Sebab |
|---|---|---|
| `TempInputKomunikasi.CaseID` | `NewMessageInput.Destination` | "CaseID" untuk sebuah pilihan PUSAT/CABANG |
| `TempInputKomunikasi.CityID` | (isian pemilih cabang di layar) | "CityID" untuk sebuah cabang |
| **`TempInputKomunikasi.City`** | **`NewMessageInput.Message`** | "City" untuk **isi pesan** |
| `TempInputKomunikasi.DistrictID` | `NewMessageInput.BranchCode` | "DistrictID" untuk kode cabang |
| `TempInputKomunikasi.District` | `BranchOption.Email` | "District" untuk alamat surel |
| `TempInputKomunikasi.pyLabel` | (tidak dibawa) | penanda keadaan layar; di React `useState` |
| `TempInputKomunikasi2.pxResults[].City` | `Destinations()` | daftar dua pilihan dropdown |
| `TempResultSurveyor.pxResults` | `[]BranchOption` | halaman sumber autocomplete |
| `.BRANCHNAME` | `BranchOption.Name` | — |
| `.BRANCH` | `BranchOption.Code` | — |
| `.EMAIL` | `BranchOption.Email` | — |

Tujuh properti `TempEmail.*` pada activity, dan tidak satu pun berarti seperti namanya:

| Pega | Kolom yang diisinya | Kode baru |
|---|---|---|
| `TempEmail.IDSurvey` | `CASEID` | `CaseOpen` |
| `TempEmail.BodyLetterAttn` | `SENDER` | `Caller.Login` |
| **`TempEmail.BodyLetterEmail`** | **`MESSAGE`** | `NewMessageCommand.Message` |
| `TempEmail.BodyLetterOP` | `SENDERNAME` | `Caller.Name` |
| **`TempEmail.BranchToTransfer`** | **`KOMUNIKASISTATUS`** | `StatusNotAnswered` |
| `TempEmail.InsuredPIC` | `COMMUNICATE_TO` | `NewMessageCommand.RecipientCode()` |
| `TempEmail.IDSurveyCase` | `COMMUNICATE_FROM` | `BranchFilter.Code` |

Rule dan artefak:

| Pega | Kode baru |
|---|---|
| `PNCSendMessageKomunikasiCabang` (activity) | `Service.SendMessage` · `Repo.SendMessage` |
| `CNMShowInsertKomunikasi_dt` (data transform) | `useState` pada layar — tidak ada padanan di peladen |
| `IsInsertKomunikasi` / `IsUpdateKomunikasi` (when) | idem |
| `InsertMessageCABANG_PNC` (SQL) | `message_insert` |
| `GetIDMaxKom_cabang` (SQL) | `message_max_id` |
| `ReplyKomunikasiCabang` (SQL, jalur kirim) | `message_history_insert` |
| *(kueri pemasok cabang — tidak ada di export)* | `branch_options` — **penyaringnya ditebak** |

Nama field JSON tetap Indonesia (`D-80`):

| Field JSON | Isinya |
|---|---|
| `tujuan` (permintaan) | `"PUSAT"` atau `"CABANG"` |
| `cabang` (permintaan) | kode cabang tujuan |
| `pesan` (permintaan) | isi pesannya |
| `tujuan` (jawaban `/cabang`) | kedua pilihan dropdown |
| `cabang` (jawaban `/cabang`) | daftar cabang — **tanpa alamat surel** |

Alamat rutenya:

```
POST /api/inbox-komunikasi-cabang/pesan     membuat percakapan baru (201)
GET  /api/inbox-komunikasi-cabang/cabang    daftar cabang + kedua tujuan
```

Uji yang lahir bersamanya:

```go
func TestAMessageToTheBranchNeedsABranchChosen(t *testing.T)
func TestABranchLeftOverFromASwitchedDestinationIsIgnoredNotRefused(t *testing.T)
func TestTheBranchListIsNotFilteredByTheCallerBranch(t *testing.T)
func TestANewMessageCanBeRepliedToImmediately(t *testing.T)
func TestTheNewMessageInsertLeavesTheCreatedDateToTheDatabase(t *testing.T)
func TestEveryLoadedQueryIsListedInAllQueryNames(t *testing.T)
func TestTheBranchPickerNeverLeaksBranchEmailAddresses(t *testing.T)
func TestTheRemovedActionRouteIsGone(t *testing.T)
```

```ts
it('menampilkan form saat "Tambah" ditekan, bukan menolak dengan alasan', ...)
it('tidak menyentuh peladen hanya untuk membuka formnya', ...)
it('menyembunyikan pemilih cabang selama tujuannya PUSAT', ...)
it('mengosongkan cabang yang tertinggal saat tujuannya kembali ke PUSAT', ...)
it('TIDAK lagi menggambar panel "Yang masih dikerjakan lewat Pega"', ...)
```

> **DICABUT pada 2026-09-24:** `RejectWrite` · rute `/tindakan` · `ErrWriteNotAvailable` ·
> `CodeWriteNotAvailable` (`"belum_tersedia"`) · `WriteActionsNotice` ·
> `useKomunikasiCabangAction`. Keempat tindakan tulis layar lama kini bekerja.

### Koreksi utas layar detail (2026-09-24, lanjutan kedua)

Layar detail membaca **`POOLDATA.M_KOMUNIKASI_CABANG`**, bukan `M_KOMUNIKASI_PNC`. Pemetaannya:

| Pega | Kolom | Kode baru |
|---|---|---|
| `.CloseClaimDate` | `CREATEDDATE` *(nama DITEBAK)* | `ThreadMessage.CreatedAt` |
| `.UserName` | `SENDER` | `ThreadMessage.SenderOperator` |
| `.Email` | `MESSAGE` | `ThreadMessage.Message` |
| `TEMPHISTORYCABANGDETAIL.pxResults` | — | `[]ThreadMessage` |
| `GetInboxKomunikasiCabang_detail` *(hilang)* | — | `detail_thread` |
| *(tidak ada padanan)* | — | `detail_header` — pemeriksa keberadaan |
| `DETAILKOMUNIKASICABANG_ACT` | — | `Repo.Detail` |

Isian JSON tiap ucapan menyusut menjadi **tiga**, persis kolom section-nya:

| Field JSON | Isinya |
|---|---|
| `tanggal` | tanggal ucapannya |
| `pengirim` | Operator ID, **apa adanya** — tidak dirakit seperti di grid |
| `pesan` | isi ucapannya |

> **DICABUT:** `jawaban` · `penjawab` · `tanggal_jawaban` · `asal` · `operator_pengirim` pada
> ucapan. Balasan BUKAN isian pada sebuah ucapan — ia ucapan tersendiri.

Uji yang lahir bersamanya:

```go
func TestTheThreadGrowsWithEveryUtteranceNotJustTheLatestOne(t *testing.T)
func TestAConversationWithoutAnyHistoryIsStillFound(t *testing.T)
func TestTheConversationOriginComesFromTheHeaderNotTheThread(t *testing.T)
func TestTheThreadIsReadFromTheHistoryTableNotTheConversationTable(t *testing.T)
func TestEachUtteranceCarriesExactlyTheThreeFieldsTheSectionDraws(t *testing.T)
```

```ts
it('menggambar setiap ucapan sebagai barisnya sendiri, termasuk balasannya', ...)
it('menyatakan keadaan percakapan yang belum punya satu pun ucapan', ...)
```
