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

## Tambahan 2026-09-19 — modul Master Pasal Kerugian

Nama modulnya berbahasa Indonesia sesuai `D-81`, karena itulah nama yang disebut Work Owner:

| Lapisan | Nama |
|---|---|
| Folder & paket Go | `internal/masterpasal` |
| Folder frontend | `src/modules/master-pasal-kerugian` |
| Paket transport | `masterpasalhttp` |

**Isi modulnya Inggris.** Penamaannya menuntut kehati-hatian lebih daripada modul lain: kelas Pega
yang menaunginya adalah `ASM-FW-GCNMFW-Int-V_D_CAUSE_OF_LOSS` — kelas **Detail Cause of Loss**,
bukan kelas pasal — sehingga tidak satu pun nama propertinya menyebutkan isinya.

| Properti Pega | Inggris di kode | Label layar | Catatan |
|---|---|---|---|
| `M_COL_ID` | `Number` | No Pasal | bukan ID cause of loss |
| `OLD_M_COL_ID` | `ID` | — | kunci baris, kolom `IDDATA`; bukan ID lama apa pun |
| `DESCRIPTION` | **`Text`** | **ISI PASAL** | isi ketentuan, bukan keterangan |
| `OLD_D_COL_ID` | **`Description`** | **Deskripsi** | keterangan singkat; **terbalik** dari dugaan wajar terhadap baris di atasnya |
| `pyCountry` | `Category` | Kategori | kode kategori; tidak ada urusan dengan negara |
| `LOSS_CODE` | `CategoryLabel` | Kategori | sebutan kategori; **bukan** kode kerugian |
| `BISNISID[].ID` | `Business[].ID` | Bisnis | kode pada `POOLDATA.BUSINESS` |
| `BISNISID[].Note` | `Business[].Name` | Bisnis | nama lini bisnis |

> **`LOSS_CODE` memikul dua arti di dua kueri berbeda.** Di `GetDataCOLByPasalBisnis_Sql` ia sebutan
> kategori dari dokumen JSON; di `BrowseCOLByPasalDataBisnis_Sql` ia `POOLDATA.BUSINESS.NOTE`, yaitu
> nama lini bisnis. Yang dipetakan karena itu selalu **kolomnya**, bukan aliasnya (`D-19`).

| Indonesia | Inggris | Catatan |
|---|---|---|
| Pasal Kerugian | Clause | satu butir ketentuan polis |
| Lini Bisnis | Business | `POOLDATA.BUSINESS`; hanya dibaca |
| Kategori | Category | Jaminan Polis · Pengecualian · Notifikasi |
| Cari lini bisnis | SearchBusiness | |
| Segarkan nama lini bisnis | resolveBusiness | membaca ulang `NOTE` dari master yang berlaku |
| Dokumen JSON | document | tipe internal `repo/sqlstore`; tidak pernah keluar dari sana |

**Nama field JSON tetap Indonesia** (`no_pasal`, `isi_pasal`, `deskripsi`, `kategori`,
`kategori_label`, `bisnis`) — ia kontrak API.

**Nama kueri `.sql`** berawalan menurut tabelnya, bukan menurut nama modul:
`clause_list` · `clause_get` · `clause_list_id_locked` · `clause_insert` · `clause_update` ·
`clause_delete` · `clause_check_table` · `business_search` · `business_get` ·
`business_check_table`.

---

## Tambahan 2026-09-19 — modul Master Bengkel

Nama folder modul: `internal/masterbengkel` (Go) dan `src/modules/master-bengkel`
(frontend), mengikuti `D-81` — nama modul bisnis, bukan terjemahan Inggrisnya.

### Kolom `POOLDATA.BENGKEL_HE` → nama Inggris

Nama Inggrisnya dipilih agar **mencerminkan isi**, bukan menerjemahkan nama kolomnya apa
adanya. Dua kolom di bawah patut diperhatikan khusus, dan keduanya ditandai tebal.

| Kolom Pega | Nama Inggris | Label layar Pega | Catatan |
|---|---|---|---|
| `ID_BENGKEL` | `ID` | ID BENGKEL | kunci baris; diterbitkan server |
| `NAMA_BENGKEL` | `Name` | NAMA BENGKEL | kunci alami; tidak boleh ganda |
| `ALM_BENGKEL` | `Address` | ALAMAT BENGKEL | |
| `TELP_BENGKEL` | `Phone` | TELP BENGKEL | |
| `NOHP_BENGKEL` | `Mobile` | NO HP BENGKEL | |
| `MAIL` | `Email` | EMAIL | |
| `MAIL_WO` | `WorkOrderEmail` | EMAIL WO | tujuan perintah kerja, terpisah dari surel umum |
| `CABANG_ID` | `BranchID` | — | |
| `NAMA_CABANG` | `BranchName` | NAMA CABANG | |
| `CITY_ID` | `CityID` | — | |
| **`NAMA_KABUPATEN`** | **`CityName`** | **NAMA KOTA** | kolomnya menyebut kabupaten, layarnya menyebut kota, dan sumbernya tabel `CITY`. Label layar yang diikuti (`D-13`) |
| `STATUS_REKANAN` | `PartnerStatus` | STATUS REKANAN | satu-satunya penanda status yang nilainya diketahui |
| `STS_BENGKEL` | `WorkshopStatus` | STATUS BENGKEL | |
| **`ALASAN_STS_BGKL`** | **`StatusReason`** | **ALASAN STATUS BENGKEL** | di `SetApprovalAllMaster` kolom yang sama dipakai membawa **nama tabel**; arti kedua itu tidak dibawa |
| `TGL_STATUS` | `StatusDate` | TANGGAL STATUS | teks, bukan tanggal — DDL belum ada (`R-08`) |
| `LOGIN_APLIKASI` | `Login` | LOGIN APLIKASI | disimpan; akunnya tidak diterbitkan |
| `BANK_ID` | `BankID` | — | |
| `NAMA_BANK` | `BankName` | NAMA BANK | |
| `NO_ACCOUNT` | `AccountNumber` | NO REKENING | |
| `NAMA_ACCOUNT` | `AccountName` | — | ada di tabel, tidak ditampilkan layar Pega |
| **`ACCOUNT_ID`** | **`AccountID`** | — | di `UpdateBengkelHE-SQL` properti klipboard bernama sama dipakai membawa **seluruh dokumen JSON**; arti kedua itu tidak dibawa |
| `NAMA_NPWP` | `TaxName` | NAMA NPWP | |
| `NO_NPWP` | `TaxNumber` | NO NPWP | |
| `ALM_NPWP` | `TaxAddress` | ALAMAT NPWP | |
| `JENIS_PPH` | `IncomeTaxType` | JENIS PPH | |
| `PPN` | `ValueAddedTax` | PPN (%) | teks presisi penuh (`D-51`) |
| `DISC_JASA` | `ServiceDiscount` | DISCOUNT JASA (%) | idem |
| `DISC_SPART` | `PartDiscount` | DISCOUNT SPAREPART (%) | idem |
| `PERSEN_MATERIAL` | `MaterialPercent` | PERSEN MATERIAL (%) | idem |
| `PCT_SELISIH_PL` | `PriceListGapPercent` | — | ada di tabel, tidak ditampilkan layar Pega |
| `SLA` | `SLA` | SLA | satuannya tidak disebut di mana pun |
| `STS_SUPPLY` | `SuppliedByASM` | STATUS DISUPPLY ASM | nilai sah tidak diketahui |
| `SUPPLIER` | `Supplier` | — | |
| `STS_EKLAIM` | `EClaimStatus` | STATUS EKLAIM | nilai sah tidak diketahui |
| `STS_AUTO_AKSEP` | `AutoAcceptStatus` | STATUS AUTO AKSEP | idem |
| `STS_PAYMENT` | `PaymentStatus` | STATUS PAYMENT | idem |
| `STS_AUTOPAYMENT` | `AutoPaymentStatus` | STATUS AUTOPAYMENT | idem |
| `STS_TEKNO` | `TeknoStatus` | STATUS TEKNO | idem |
| `STS_ORDER` | `OrderStatus` | STATUS ORDER | idem |
| `DOKUMENID` | `DocumentID` | — | lampiran; dipertahankan, tidak ditimpa |
| `APPROVAL` | `Status` | — | "0" menunggu · "1" disetujui · "2" ditolak |

### Istilah domain baru

| Indonesia | Inggris | Catatan |
|---|---|---|
| Bengkel | Workshop | tipe agregat modul ini |
| Cabang | Branch | `GENERAL.LST_USER_ASURANSI` + `LST_DET_CABANG`; hanya dibaca |
| Kota | City | tabel `CITY`, nama kota ada di kolom **`NOTE`** |
| Bank | Bank | `GENERAL.LST_BANK_GROUP`; hanya dibaca |
| Rekanan | Partner | `STATUS_REKANAN` bukan nol |
| Non-rekanan | NonPartner | `STATUS_REKANAN` bernilai nol; tidak diberi login |
| Keputusan borongan | Decide | satu status untuk sekumpulan baris |
| Terbitkan ID | NextID / ComposeID | kode situs + nomor urut sepuluh digit |

### Nama field JSON tetap Indonesia

Ia kontrak API (`D-80`): `id_bengkel`, `nama_bengkel`, `alamat_bengkel`, `telp_bengkel`,
`nohp_bengkel`, `email`, `email_wo`, `id_cabang`, `nama_cabang`, `id_kota`, `nama_kota`,
`status_rekanan`, `status_bengkel`, `alasan_status_bengkel`, `tanggal_status`,
`login_aplikasi`, `id_bank`, `nama_bank`, `no_rekening`, `nama_rekening`, `id_rekening`,
`nama_npwp`, `no_npwp`, `alamat_npwp`, `jenis_pph`, `ppn`, `diskon_jasa`,
`diskon_sparepart`, `persen_material`, `pct_selisih_pl`, `sla`, `status_disupply_asm`,
`supplier`, `status_eklaim`, `status_auto_aksep`, `status_payment`, `status_autopayment`,
`status_tekno`, `status_order`, `id_dokumen`, `status`, `status_label`, `rekanan`.

### Nama kueri `.sql`

Berawalan menurut tabelnya, bukan menurut nama modul:
`bengkel_list` · `bengkel_list_search` · `bengkel_get` · `bengkel_find_by_name` ·
`bengkel_find_by_login` · `bengkel_lock_by_name` · `bengkel_lock_by_login` ·
`bengkel_insert` · `bengkel_update` · `bengkel_set_status` · `bengkel_count_pending` ·
`bengkel_count_all` · `bengkel_check_table` · `bengkel_check_json_mirror` ·
`bengkel_count_json_mirror` · `bengkel_site` · `bengkel_next_sequence` ·
`bengkel_branch_list` · `bengkel_city_search` · `bengkel_bank_list`.

---

## Tambahan 2026-09-20 — modul Master Panel

Modul master **pertama yang punya baris anak**. Penamaannya karena itu memuat satu hal
yang belum pernah ada: nama untuk entitas anak dan nama untuk kumpulannya.

### Folder dan paket

| Lapisan | Bentuk | Nilai |
|---|---|---|
| Folder backend & nama paket Go | `namamodul`, tanpa tanda hubung | `internal/masterpanel` |
| Folder frontend | `nama-modul`, `kebab-case` | `src/modules/master-panel` |

Nama modulnya **Indonesia** karena itulah nama yang dipakai Work Owner dan yang tertulis di
`m_menu_aplikasi_pnc.csv` (`MENU_DESC = "Master Panel"`) — `D-81`. Isinya **Inggris** —
`D-80`.

### Tipe dan fungsi

| Kolom / konsep Pega | Nama di kode (Inggris) | Nama di kontrak API (Indonesia) |
|---|---|---|
| satu baris `PANEL_HE` | `Panel` | — |
| satu baris `LOKASI_PANEL_HE` | `PanelLocation` | — |
| kolom `ID_PANEL` | `Panel.ID` | `id_panel` |
| kolom `NAME` | `Panel.Name` | `nama_panel` |
| kolom `STS_REPAIR` | `Panel.RepairStatus` | `status_repair` |
| kolom `STS_EDIT_QTY` | `Panel.EditQuantityStatus` | `status_edit_quantity` |
| kolom `STS_PREMIUM_REPAIR` | `Panel.PremiumRepairStatus` | `status_premium_repair` |
| kolom `STS_PECAH` | `Panel.ShatterStatus` | `status_pecah` |
| kolom `STS_STICKER` | `Panel.StickerStatus` | `status_sticker` |
| kolom `STS_SISI` (**induk**) | `Panel.SideStatus` | `status_sisi` |
| kolom `STS_RUSAK_PARAH` | `Panel.SevereDamageStatus` | `status_rusak_parah` |
| kolom `STS_AKTIF` | `Panel.ActiveStatus` | `status_aktif` |
| kolom `EXCLUSION_C` | `Panel.ExclusionC` | `exclusion_c` |
| kolom `STS_APPROVAL` | `Panel.ApprovalMark` | `status_approval` |
| kolom `ALASAN_TOLAK` | `Panel.RejectReason` | `alasan_tolak` |
| kolom `DOKUMENID` | `Panel.DocumentID` | `id_dokumen` |
| kolom `APPROVAL` | `Panel.Status` | `status` |
| kolom `LOKASI_PANEL` (**anak**) | `PanelLocation.Name` | `lokasi_panel` |
| kolom `SISI_PANEL` (**anak**) | `PanelLocation.Side` | `sisi_panel` |
| koleksi baris anak | `Panel.Location` | `lokasi` |

### Satu nama Pega yang dipisah menjadi dua

`STS_SISI` dipakai untuk **dua hal yang sama sekali berbeda**:

| Di Pega | Di modul ini |
|---|---|
| kolom `PANEL_HE.STS_SISI`, caption layar "STATUS SISI" | `Panel.SideStatus` · `status_sisi` |
| alias `SISI_PANEL as "STS_SISI"` pada `GetLokasiSisiPanel-SQL.xml` | `PanelLocation.Side` · `sisi_panel` |

Nama alias TIDAK dibawa. Ini perlakuan yang sama dengan `ACCOUNT_ID` dan `ALASAN_STS_BGKL`
di Master Bengkel: nama yang memikul dua arti dipecah menjadi dua nama yang masing-masing
berarti satu hal.

### `ApprovalMark`, bukan `ApprovalStatus`

`STS_APPROVAL` dan `APPROVAL` adalah **dua kolom berbeda** yang berdampingan di report
definition yang sama, dan hanya `APPROVAL` yang dipakai menyaring tab.

Menamai keduanya dengan kata "status" akan membuat keduanya tertukar pada pembacaan
sekilas — persis kelas cacat yang `D-18` cegah pada empat konsep status klaim. Karena itu:

| Kolom | Nama | Alasan |
|---|---|---|
| `APPROVAL` | `Status` bertipe `ApprovalStatus` | ia yang menentukan tab |
| `STS_APPROVAL` | `ApprovalMark` bertipe `string` | artinya tidak diketahui; ia bukan status apa pun yang dikenali modul ini |

### Tipe baru yang lahir di modul ini

| Nama | Isi | Kenapa tipe tersendiri |
|---|---|---|
| `Side` | `"-"` · `"1"` · `"2"` | Satu-satunya daftar nilai modul ini yang **benar-benar terbaca** dari export, dan ia punya `Label()` serta `Known()` — dua hal yang tidak dimiliki `string` |
| `LocationOptions` | `KIRI` · `KANAN` · `DEPAN` · `BELAKANG` · `LAIN-LAIN` | Variabel paket, bukan tipe: ia daftar **saran**, bukan aturan — nama lokasi di luar kelimanya tetap diterima |

### Nama uji

| Uji | Yang dijaganya |
|---|---|
| `TestEveryParentFieldIsRequired` | kesepuluh isian induk wajib, sesuai `pyRequired=true` |
| `TestPanelWithoutLocationIsValid` | panel tanpa lokasi adalah keadaan yang sah |
| `TestUnknownSideRejected` | selisih terencana terhadap `GetSisiPanel-Act` |
| `TestDuplicateLocationRejected` | selisih terencana; sistem lama tidak memeriksanya |
| `TestLocationOutsideTheFiveOptionsAccepted` | yang sengaja **tidak** dibatasi |
| `TestComposeIDDoesNotTruncate` | lebar enam digit, dan kunci dibiarkan tumbuh alih-alih bertabrakan |
| `TestDeleteOnlyOnTheChildTable` | `DELETE` terhadap tabel induk tetap dilarang (`D-66`) |
| `TestLocationInsertHasFourArguments` | kolom `NAMA` ikut ditulis — lihat asumsinya di `keputusan-implementasi.md` §22.4 |
| `TestParentAndChildSearchFiltersMatch` | penyaring induk dan anak sama persis, supaya tidak ada baris yang tampil tanpa lokasinya |
| `TestLengthLimitsAreTheOnesTheFormRepeats` | duplikasi batas panjang antara backend dan `PanelForm.tsx` tetap terlihat |

---

## Tambahan 2026-09-20 — modul Master Supplier

Modul `mastersupplier` (frontend `master-supplier`), atas tabel `M_SUPPLIER`.

Penamaannya mengikuti `D-80` dan `D-81` seperti modul lain: **nama modul** berbahasa
Indonesia, **isi modul** berbahasa Inggris, **kontrak** (JSON API) berbahasa Indonesia.

### Kunci JSON → nama Go

Berbeda dari modul lain yang memetakan **kolom** tabel, modul ini memetakan **kunci di
dalam dokumen JSON**. Kedua puluh lima kunci pertama dibaca dari
`RDB List/GetDataEditMasterSupller-SQL.xml:90-118`.

| Kunci JSON | Nama Go | Nama JSON API | Catatan |
|---|---|---|---|
| `NAMA` | `Name` | `nama` | kunci alami; terkunci setelah tersimpan |
| `ALAMAT` | `Address` | `alamat` | |
| `KOTA` | `City` | `kota` | **nama**, bukan kode — tidak ada `KOTA_ID` |
| `NAMA_CABANG` | `BranchName` | `nama_cabang` | **nama**, bukan kode |
| `KODE_POS` | `PostalCode` | `kode_pos` | |
| `NEGARA` | `Country` | `negara` | **nama**, bukan kode |
| `TELEPON` | `Phone` | `telepon` | |
| `FAX` | `Fax` | `fax` | |
| `EMAIL` | `Email` | `email` | |
| `NPWP` | `TaxNumber` | `npwp` | |
| `CONTACT_PERSON` | `ContactPerson` | `contact_person` | |
| `STS_REKANAN` | `PartnerStatus` | `status_rekanan` | nilai sahnya tidak diketahui |
| `JENIS_STATUS` | `SupplyType` | `status_supply` | label layarnya **"Status Supply"**, bukan "Jenis Status" |
| `SUPPLIER_HE` | `HeavyEquipment` | `supplier_he` | turunan `SupplyType` |
| `TOP` | `TermOfPayment` | `term_of_payment` | satuannya tidak disebut di mana pun |
| `TOD` | `TermOfDelivery` | `term_of_delivery` | idem |
| `KETERANGAN` | `Note` | `keterangan` | ikut ke kolom `ALASAN_REQ` |
| `BANK` | `Bank` | `bank` | **nama**, bukan kode |
| `ACCOUNT_NO` | `AccountNumber` | `no_account` | |
| `ACCOUNT_NAME` | `AccountName` | `account_name` | |
| `BANK_BRANCH` | `BankBranch` | `bank_branch` | |
| `JENIS_SUPPLIER` | `SupplierType` | `jenis_supplier` | nilai sahnya tidak diketahui |
| `STS_AKTIF_PROMLIST` | `ActiveRequested` | `status_aktif` | yang **diisi** pengguna |
| `STS_AKTIF` | `Active` | `status_aktif_berlaku` | yang **berlaku**; bukan isian |
| `STS_AUTOPAYMENT` | `AutoPayment` | `status_autopayment` | |
| `USERKLAIMID` | `UpdatedBy` | `diubah_oleh` | ditulis, tidak pernah dibaca sistem lama |
| `TGL_INSERT` | `UpdatedAt` | `diubah_pada` | teks `dd/MM/yyyy` WIB |
| `ID` | `ID` | `id_supplier` | juga sebuah **kolom** tabel |

Kolom tabel yang bukan kunci dokumen:

| Kolom | Nama Go | Nama JSON API |
|---|---|---|
| `OLDID` | `OldID` | `id_lama` |
| `JSONDATA` | — | — |

### Dua nama yang TIDAK dibawa, dan kenapa

| Nama lama | Nama baru | Sebabnya |
|---|---|---|
| `NO_KLAIM` pada `proteksi_klaimmbu` | `SupplierID` | Kolom itu diisi **ID supplier**, bukan nomor klaim. Kolom berarti ganda persis seperti yang `03-CURRENT-ARCHITECTURE.md` §4.2 catat; kolomnya tetap ditulis apa adanya, tetapi di dalam kode ia bernama sesuai isinya (`D-19`) |
| Kolom grid berlabel "JENIS SUPPLIER" | dipecah dua | Di Pega kolom itu menampilkan `.JENIS_STATUS_NOTE` — label **Status Supply**, bukan Jenis Supplier. Layar baru menampilkan keduanya, masing-masing dengan namanya sendiri |

### Nama yang sengaja BERBEDA dari modul tetangga

| Modul lain | Modul ini | Kenapa |
|---|---|---|
| `masterbengkel.Bank{Code, Name}` disimpan berpasangan | `mastersupplier.Bank{Code, Name}`, hanya `Name` yang disimpan | Dokumen supplier tidak punya kunci `BANK_ID`. Kodenya hanya pembeda di daftar pilihan |
| `masterbengkel.City` disimpan dengan ID-nya | `mastersupplier.City`, hanya nama yang disimpan | idem — tidak ada `KOTA_ID` |
| `sequenceWidth = 10` (bengkel) | `SequenceWidth = 11` | `PEGA_M_SUPPLIER.prc:21` memakai sebelas digit. Bedanya satu karakter, dan tidak mungkin terlihat tanpa membandingkannya |

### Tipe baru yang lahir di modul ini

| Nama | Isi | Kenapa tipe tersendiri |
|---|---|---|
| `ApprovalRequest` | satu baris `pooldata.proteksi_klaimmbu` | Ia milik **proses lain** — antrean yang layar pemutusnya tidak ada di export. Menggabungkannya ke `Supplier` akan menyamarkan batas kepemilikan itu |
| `CodeOption` · `CodeSet` | pilihan kelima dropdown bersandi | Ia menjawab pertanyaan yang berbeda dari master mana pun: bukan "apa isi baris ini" melainkan "sandi apa yang pernah dipakai". Labelnya sengaja dapat berisi sandinya sendiri, sebagai tanda bahwa artinya **tidak diketahui** |
| `Country` | satu baris `COUNTRY` | Lookup keempat yang tidak dimiliki modul master lain |

### Nama uji

| Uji | Yang dijaganya |
|---|---|
| `TestRequiredFieldsRejectEmpty` | kelima belas isian wajib, sesuai `pyRequired=true` — **dan jumlahnya**, lewat `require.Len(blank, 15)` |
| `TestHeavyEquipmentAlwaysWritten` | selisih terencana: menonaktifkan HE benar-benar mengubah nilainya |
| `TestFormatJakartaDateCrossesMidnight` | kasus `R-12`: 17:30 UTC sudah keesokan harinya di WIB |
| `TestComposeApprovalIDUsesUTC` | kunci teknis tidak memakai zona waktu setempat |
| `TestComposeIDDoesNotTruncate` | lebar sebelas digit, dan kunci dibiarkan tumbuh alih-alih bertabrakan |
| `TestSequenceWidthFollowsProcedure` | lebarnya **sebelas**, bukan sepuluh seperti modul tetangga |
| `TestNoOracleDotNotation` | notasi titik Oracle atas JSONDATA tidak menyelinap masuk |
| `TestWrittenKeysMatchReadKeys` | setiap kunci yang ditulis benar-benar dibaca kembali |
| `TestCodeGroupNamesMatch` | nama kelompok sandi di SQL sama dengan konstanta di Go |
| `TestOnlyOwnedTablesAreWritten` | hanya dua tabel yang boleh ditulis; enam acuan hanya dibaca |
| `TestSaveSkipsApprovalWhenDeactivating` | menonaktifkan berlaku seketika — perilaku sistem lama yang paling mudah hilang |
| `TestDefaultCodeOptionOnlyHoldsProvenValues` | sandi yang artinya terbukti tidak bertambah tanpa bukti baru |
| `TestLengthLimitsAreStable` | duplikasi batas panjang antara backend dan `SupplierForm.tsx` tetap terlihat |

### Nama yang lahir dari paginasi (komponen bersama)

Ditambahkan ke `components/DataTable.tsx`, bukan ke modul ini — lihat
`keputusan-implementasi.md` §23.16 untuk alasannya.

| Setelan Pega | Nama di kode | Tempat | Catatan |
|---|---|---|---|
| `pyPageSize` (atau `pyPageSizeOther`) | `pageSize` | prop `DataTable` | opsional; tidak diisi berarti tanpa paginasi |
| `pyPageMode = Numeric` | — | — | bentuknya diwujudkan `Paginator`, bukan disimpan sebagai nilai |
| `pyGridPaginator` | `Paginator` | komponen dalam `DataTable.tsx` | bilah nomor halaman di kaki tabel |
| — | `pageWindow` | fungsi terekspor | memilih nomor mana yang digambar; **diekspor** supaya dapat diuji terpisah dari komponennya |

Nama-nama di dalam komponen mengikuti `D-80` — berbahasa Inggris — kecuali satu:
`'sela'`, penanda `…` pada daftar nomor halaman. Ia **nilai data**, bukan nama simbol, dan
padanan Inggrisnya (`gap`, `ellipsis`) tidak lebih jelas bagi pembaca yang membaca
komentarnya dalam bahasa Indonesia.

Teks yang dilihat pengguna tetap Indonesia sesuai `D-13`: "Menampilkan 1–20 dari 57
baris.", "Halaman sebelumnya", "Halaman berikutnya", "Halaman tabel".

| Uji | Yang dijaganya |
|---|---|
| `tanpa paginasi > menggambar SELURUH baris bila pageSize tidak diisi` | bawaan komponen tidak berubah — inilah yang melindungi sembilan layar master yang sudah selesai |
| `mencari di SELURUH baris, bukan hanya halaman yang tampil` | urutan saring → urutkan → potong, sama seperti page list klipboard Pega |
| `kembali ke halaman pertama saat kata kunci berubah` | posisi halaman tidak bertahan melewati daftar yang berbeda |
| `pageWindow > tidak memakai sela untuk menyembunyikan satu nomor saja` | `…` tidak pernah memakan ruang yang sama dengan nomor yang disembunyikannya |
| `memaginasi 20 baris per halaman, sesuai pyPageSize layar lama` | angka 20 milik LAYAR Supplier, bukan bawaan komponen |

---

## Master Sparepart (2026-09-20)

Modul `mastersparepart` — nama folder mengikuti nama modul bisnis yang disebut Work Owner
(`D-81`): backend `internal/mastersparepart`, frontend `src/modules/master-sparepart`.

### Kolom POOLDATA.SPAREPART_HE

Kedua puluh empat kolomnya, beserta nama di kode dan nama pada kontrak API. Label layar
dibaca dari `pyLabelFieldValue` pada `Section/BrowseMasterSparepartHEApproval-Section.xml`.

| Kolom | Label layar Pega | Nama di kode (Inggris) | Nama JSON (Indonesia) |
|---|---|---|---|
| `ID` | ID Sparepart | `ID` | `id_sparepart` |
| `NAMA_SPART` | Nama Sparepart | `Name` | `nama_sparepart` |
| `NO_SPART` | Nomor Sparepart | `Number` | `nomor_sparepart` |
| `KODE_SPART` | Kode Sparepart | `Code` | `kode_sparepart` |
| `HARGA_JUAL` | Harga Jual (Rp) | `SellingPrice` | `harga_jual` |
| `KATEGORI_SPART` | Kategori Sparepart | `CategoryID` | `kategori_sparepart` |
| `TIPE_SPART` | Tipe Sparepart | `TypeID` | `tipe_sparepart` |
| `BERAT` | Berat Sparepart (gram) | `Weight` | `berat` |
| `PANJANG` | Panjang Sparepart (cm) | `Length` | `panjang` |
| `LEBAR` | Lebar Sparepart (cm) | `Width` | `lebar` |
| `TINGGI` | Tinggi Sparepart (cm) | `Height` | `tinggi` |
| `MIN_STOCK` | Stock Minimal | `MinStock` | `stock_minimal` |
| `MAX_STOCK` | Stock Maximal | `MaxStock` | `stock_maximal` |
| `QTY_PESAN` | Kuantitas Pesanan | `OrderQuantity` | `kuantitas_pesanan` |
| `PROD_DATE` | Tanggal Produksi | `ProductionDate` | `tanggal_produksi` |
| `SUBSTITUSI_SPART` | Part Substitusi | `Substitute` | `part_substitusi` |
| `JENIS_SPART` | Jenis Sparepart | `Kind` | `jenis_sparepart` |
| `SATUAN` | Satuan | `Unit` | `satuan` |
| `STS_AKTIF` | Status Aktif | `ActiveStatus` | `status_aktif` |
| `STS_PART` | Status Sparepart | `PartStatus` | `status_sparepart` |
| `USER_UPDATE` | User Update | `UpdatedBy` | `user_update` |
| `TGL_UPDATE_HARGA` | Tanggal Update | `PriceUpdatedAt` | `tanggal_update_harga` |
| `DOKUMENID` | — | `DocumentID` | `id_dokumen` |
| `APPROVAL` | — | `Status` | `status` |

`CategoryID` dan `TypeID` **berakhiran ID dengan sengaja**: kolomnya menyimpan
`PART_CATEGORY_ID` dan `PART_SECTION_ID`, bukan namanya. Namanya dikirim terpisah sebagai
`nama_kategori_sparepart` dan `nama_tipe_sparepart`, dihitung server dari daftar acuan.

`Kind`, bukan `Type`, untuk `JENIS_SPART` — `type` adalah kata kunci Go. Alasan yang sama
membuat tipe acuannya bernama `PartType`, bukan `Type`.

### Kolom tabel acuan

| Kolom | Nama di kode | Nama JSON | Catatan |
|---|---|---|---|
| `PART_CATEGORY_ID` | `Category.ID` | `kode` | dialiaskan `"CityID"` di Pega — TIDAK dibawa |
| `PART_CATEGORY_NAME` | `Category.Name` | `nama` | dialiaskan `"City"` — TIDAK dibawa |
| `PART_SECTION_ID` | `PartType.ID` | `kode` | |
| `PART_SECTION_NAME` | `PartType.Name` | `nama` | |
| `PART_CATEGORY_ID` (pada tabel tipe) | `PartType.CategoryID` | `kode_kategori` | dialiaskan `"District"` — TIDAK dibawa |

### Nama yang TIDAK dibawa

| Nama di Pega | Kenapa tidak dibawa |
|---|---|
| `TempInputPanelHE.CaseID` / `.City` / `.Country` | halaman milik Master **Panel**, dipakai validasi Sparepart; ketiga propertinya tidak mencerminkan isinya |
| `InputBengkel.ID_BENGKEL` (dipakai untuk ID sparepart) | nama kolom master lain dipakai memikul kunci master ini |
| `InputBengkel.ALASAN_STS_BGKL` | properti bernama "alasan status bengkel" memikul **jenis master** |
| `ErrMsg` (`PEGA_M_SPAREPART_HE.prc`) | satu keluaran memikul pesan berhasil DAN pesan galat |
| `.TELP_BENGKEL` (sel grid) | menunjuk properti yang tidak ada di `SPAREPART_HE`; selalu kosong |

### Konstanta dan uji yang menjaganya

| Konstanta | Nilai | Asalnya | Uji yang menjaganya |
|---|---|---|---|
| `sequenceWidth` | 10 | `PEGA_M_SPAREPART_HE.prc:21` | `TestSequenceWidthFollowsTheProcedure` |
| `approvedLookup` | status disetujui | `BrowseTipeKategoriPart-Act.xml` | `TestLookupFilterIsApproved` |
| `MaxNameLength` dkk | 6 angka | asumsi (`R-08`) | `TestLengthLimitsAreTheOnesTheFormRepeats` |
| `PAGE_SIZE` | 30 | `pyPageSize` ketiga section tab | — (nilai layar, bukan komponen) |
| `MaxPrice` | 100 miliar | penjaring salah ketik, bukan aturan bisnis | `TestCheckPrice` |

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

---

## Tambahan 2026-09-20 — modul Inbox Admin (`inboxadmin`)

Modul ini **kasus alias paling berat di seluruh export**, dan berbeda jenisnya dari
View History Claim: di sana satu alias salah arti tetapi konsisten, di sini **satu alias
berarti hal yang berbeda tergantung tab mana yang terbuka**. Sebabnya kedelapan grid berbagi
satu halaman klipboard yang sama, dan tiap kueri mengisi ulang properti yang sama dengan
kolom yang berbeda.

### Nama modul

`inboxadmin` di backend, `inbox-admin` di frontend. Ia mengikuti `D-81`: nama modulnya
berbahasa Indonesia karena itulah nama yang dipakai Work Owner, sementara isinya berbahasa
Inggris. Judul yang dibaca pengguna tetap **"Inbox Admin"**, mengikuti judul menu Pega
(`D-13`).

### Kode tab — nilainya dipertahankan, namanya tidak

Properti pemilih tab di Pega bernama `TempView.CityID` — nama yang tidak menyatakan isinya
sama sekali. Namanya **tidak dibawa**; nilainya **dipertahankan**.

Alasan mempertahankan nilai: kode `3`, `7`, `9`, `11` muncul di prakondisi 34 langkah
activity dan di kondisi tampil delapan kontainer grid. Menomori ulang tabnya berarti setiap
penelusuran balik ke export Pega harus menempuh satu tabel terjemahan.

| Kode | Tab | Kueri lama |
|---|---|---|
| `3` | ALL | `BrowseClaimALL` |
| `7` | Unregistered RCV | `BrowseClaimNotRegistAll` |
| `8` | Unregistered RCV Online | `BrowseClaimNotRegistAll` + saringan kurir |
| `9` | Request Survey | `BrowseRequestSurvey` |
| `10` | Request Dokumen | `GetRequestDokumenKomunikasi` |
| `11` | All Case Admin | `GetAllCaseAdmin` |
| `12` | Branch Claim | `GetKlaimCabang` |
| `13` | Status RCL/PUCL | `GetReminderPUCL` |

Kode `4`, `5`, `6` — tab Komunikasi — **tidak dibangun** (keputusan Work Owner 2026-09-20).

### Properti grid Pega → arti sebenarnya → nama di kode

**Kelompok tab ALL / Unregistered RCV / Branch Claim:**

| Properti grid Pega | Kolom basis data | Arti bagi pengguna | Nama di kode |
|---|---|---|---|
| `.PNCCaseID` | `A.PYID` | Case ID | `CaseID` |
| `.TypeOfClaim` ⚠ | `A.PZINSKEY` | kunci teknis Pega | `Reference` |
| `.PolicyNo` | `POLICYNO` | Policy no | `PolicyNumber` |
| `.QQName` | `QQNAME` | Insured name | `InsuredName` |
| `.JenisDokumen` ⚠ | `BUSINESSNAME` | Business name | `BusinessName` |
| `.RCVID` ⚠ | `A.SOBNAME` | Business source | `BusinessSource` |
| `.NumberOfDocument` ⚠ | `BRANCHNAME` | Branch Name | `BranchName` |
| `.UserAdmin` ⚠ | `BRANCH.BRANCHNAME` | Branch Claim | `ClaimBranch` |
| `.Keterangan` ⚠ | `PXCREATEOPERATOR` | Creator | `Creator` |
| `.TglKejadian` | `DATEOFLOSS_1` | Date of loss | `LossDate` |
| `.ReceivedDate` ⚠ | `REPORTDATE_1` / `REGISTERDATE_1` | Report Date | `ReportDate` |
| `.pxCreateDateTime` | `PXCREATEDATETIME` | Input Date | `InputDate` |
| `.DateForAging` | `T_CLAIM_JOB_PERSONALACCIDENT.INSERTDATE` | dasar Aging LOD | `LODDate` |
| `.TelpPengirim` ⚠ | `NOTREGISTNOTE_1` | Note | `Note` |
| `.PosisiProgressID` ⚠ | `CASE PYSTATUSWORK` | Claim Position | `ClaimPosition` |
| `.StatusLock` ⚠ | `V_STS_CLAIM.LSC_NOTE` | Claim Status | `ClaimStatus` |
| `.StatusWorkCase` ⚠ | `CASE` atas `STATUSLOD` | LOD Status | `LODStatus` |
| `.Kurir` ⚠ | `B.PXFLOWNAME` | nama flow — **tidak ditampilkan** | tidak dibawa |
| `.StatusKomunikasi` ⚠ | `KODECABANG_1` | kode cabang — **tidak ditampilkan** | tidak dibawa |

**Kelompok tab Request Survey — properti yang SAMA, arti yang BERBEDA:**

| Properti grid Pega | Kolom basis data | Arti bagi pengguna | Nama di kode |
|---|---|---|---|
| `.SubjectEmail` ⚠ | `T_REQ_SURVEY.CLAIMID` | kunci teknis Pega | `Reference` |
| `.StatusKomunikasi` ⚠ | `T_REQ_SURVEY.INPUTDATE` | Tanggal Request | `RequestDate` |
| `.Kurir` ⚠ | `A.BRANCHNAME` | Cabang Polis | `PolicyBranch` |
| `.Keterangan` ⚠ | `T_REQ_SURVEY.BRANCH` | Cabang Survey | `SurveyBranch` |
| `.Resource` ⚠ | `USERTEKNIS_1` | PIC Klaim | `TechnicalPIC` |
| `.RCVID` ⚠ | `T_REQ_SURVEY.SURVEYOR` | Surveyor | `Surveyor` |
| `.UserAdmin` ⚠ | `SUBSTR(SURVEYID, 20, 30)` | No Survey | `SurveyNumber` |

**Kelompok tab Status RCL/PUCL:**

| Properti grid Pega | Kolom basis data | Arti bagi pengguna | Nama di kode |
|---|---|---|---|
| `.ClaimID` | `A.PYID` | Case ID | `CaseID` |
| `.ClaimNo` ⚠ | `A.PZINSKEY` | kunci teknis Pega | `Reference` |
| `.NewTelpTertanggung` ⚠ | `A.QQNAME` | Nama Tertanggung | `InsuredName` |
| `.pxCreateDateTime` ⚠ | `TANGGALKIRIMPUCL_1` | Tanggal Masuk Inbox | `InboxDate` |
| `.NoteKomite` ⚠ | `KOMENTARANALISATOR_1` | Deskripsi Analyst | `AnalystNote` |
| `.Status` | `CASE RCL_PUCL_1` | Status RCL/PUCL | `RCLPUCLStatus` |
| `.TanggalCetakDLA` ⚠ | `TANGGALCETAKDOKUMENPUCL_1` | Tanggal Cetak Surat | `LetterPrintDate` |
| `.LOGSEEN` ⚠ | `LAMAKLAIM_1` | Lama Klaim | `ClaimAge` |
| `.StsAcceptance` ⚠ | `STATUSKLAIM_1` | Status Kadaluarsa | `ExpiryStatus` |

⚠ menandai nama yang tidak menyatakan isinya. Perhatikan `.LOGSEEN`: di modul View History
Claim ia berarti **jatah lihat data proteksi**, di sini ia berarti **lama klaim**.

### Istilah domain baru

| Indonesia | Inggris | Catatan |
|---|---|---|
| Tab / antrean | `Tab` | satu antrean kerja pada layar ini |
| Kode tab | `Tab.Code` | nilai `TempView.CityID` sistem lama |
| Baris pekerjaan | `WorkItem` | satu baris antrean |
| Lini bisnis (penyaring) | `BusinessLine` | `ALL` · `NONMBU` · `BONDING` · `PA` · `TRAVEL` |
| Umur / tenggat | `Aging` | `ReportAgingDays` · `TotalAgingDays` · `LODAgingDays` · `RequestAgingDays` |
| Tab yang tidak dibangun | `DisabledTab` | ketiga tab Komunikasi |
| Keterbatasan | `Limitations` | hal yang belum berjalan penuh, dikirim ke layar |

### Kata kerja tambahan

| Indonesia | Inggris | Catatan |
|---|---|---|
| Potong satu halaman | `Slice` | memotong halaman dari seluruh baris yang sudah di tangan |
| Isi kolom Aging | `WithAging` | mengembalikan salinan, bukan mengubah di tempat |
| Keterangan layar | `Metadata` | daftar tab dan dropdown; tidak menyentuh basis data |

**Nama field JSON tetap Indonesia** — `case_id`, `no_polis`, `sumber_bisnis`,
`cabang_klaim`, `aging_total`, `tab_bawaan`, `tab_dinonaktifkan`, `keterbatasan`. Ia kontrak
API.

**Nama kueri `.sql`** berawalan `list_` untuk ketujuh tab, ditambah satu pemeriksa:
`list_all` · `list_unregistered` · `list_request_survey` · `list_request_document` ·
`list_all_case_admin` · `list_branch_claim` · `list_rcl_pucl` · `check_table`.

**Tidak ada tabel baru** dan **tidak ada migrasi**: seluruh tabel yang dibaca modul ini sudah
ada dan milik sistem lama.
