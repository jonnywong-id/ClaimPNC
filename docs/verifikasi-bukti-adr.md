# Verifikasi Bukti untuk ADR dan Acceptance Criteria

Bukti setingkat baris untuk hal-hal yang akan (i) mengikat keputusan arsitektur, atau
(ii) menjadi acceptance criteria yang dapat diuji. **Bukan** pengulangan analisis Steering —
Steering sudah membaca 2.167 rule; berkas ini menyempitkan dan mendalamkan.

| | |
|---|---|
| **Tanggal** | 2026-09-08 |
| **Basis** | Export rule Pega PRPC 8.3 — 2.167 berkas XML, 297 MiB, 15 folder rule |
| **Metode** | Pencarian terarah (`grep -n`/`rg -n`) + pembacaan potongan; 6 sub-agen read-only paralel, setiap temuan konsekuensial diverifikasi ulang oleh orkestrator |
| **Status** | Fase 1 selesai (GATE 1 disetujui) · **Fase 1-lanjutan di §14** |

> ⚠️ **BACA §14 LEBIH DULU.** Bagian §0–§13 dibangun atas snapshot **2.167 rule** tanggal
> 2026-09-08. Pada 2026-09-09 export bertambah menjadi **2.389 rule** dan folder `Database/`
> muncul dengan 63 objek Oracle + 2 master CSV. **§14 mencatat selisihnya dan mengoreksi enam
> kesalahan di §0–§13** — termasuk satu klaim yang dicabut seluruhnya (§3/`K-20`) dan tiga
> kesimpulan arti kode status yang terbukti salah (§4.4). Setiap angka di §0–§13 berlaku untuk
> snapshot 2.167 dan harus dibaca dengan tanggalnya.

## Aturan bukti yang dipakai di seluruh berkas ini

1. Setiap klaim faktual membawa `path/berkas.xml:baris`.
2. Yang tidak dapat dibuktikan ditulis literal **`TIDAK DAPAT DIPASTIKAN DARI EXPORT`**.
   Tidak ada tebakan yang dituliskan sebagai fakta.
3. Setiap "tidak ada" diterjemahkan menjadi salah satu dari tiga kategori, tidak dibiarkan ambigu:
   **(1)** benar-benar tidak dipakai · **(2)** hidup di luar export (database Oracle / sistem lain /
   ruleset yang tidak diekspor) → pertanyaan ke pihak luar · **(3)** belum diperiksa.
4. **Nilai sensitif tidak direproduksi.** Kredensial, kunci API, hostname/IP produksi, dan alamat
   email dirujuk dengan `berkas:baris` + nama elemen saja. Alamat email disamarkan.
5. **Nama berkas bukan inventaris.** Lihat §9.3 — 2 berkas berisi rule yang bukan namanya.

> ⚠️ **Peringatan istilah.** Di seluruh berkas ini, **"Ticket rule"** berarti mekanisme transisi
> lateral Pega (folder `Ticket/`). **"tiket"** berarti tiket pekerjaan. Keduanya tidak pernah
> dipakai bergantian.

---

# 0. Ringkasan temuan yang mengubah premis kerja

Sembilan temuan di bawah bukan detail tambahan — masing-masing mengubah sesuatu yang sudah
disetujui, atau memindahkan pekerjaan dari "dapat diestimasi" ke "belum dapat diestimasi".

| # | Temuan | Mengubah apa | Rujukan |
|---|---|---|---|
| T-1 | **Gap export ±242 rule, bukan 43.** Tujuh tipe rule tidak diaudit `19-GAP` sama sekali — terberat **116 When rule buatan sendiri** | Permintaan ke Tim Pega; kesiapan hampir semua modul | §9.1 |
| T-2 | **HCC/HCQ nol jejak di export.** Hanya 2 teks pesan error | D-07 adalah integrasi *greenfield*; F-3 tidak punya pembanding untuk uji kesetaraan | §1.1 |
| T-3 | **Toleransi spreading 99,99% adalah pencocokan substring**, bukan perbandingan numerik. Nol pembulatan | BRD §11.3 mendeskripsikan maksud, bukan perilaku. Total `199.99` dan `1100.0` lolos | §8.3 |
| T-4 | **Penyesuaian 7 jam diterapkan tidak konsisten di dalam satu kondisi validasi** | Bukti konkret R-12; F-5 akan mengubah hasil validasi di batas periode polis | §8.1 |
| T-5 | **`ClaimStatus` milik ruleset GISFW**, bukan Claim PNC | Salah satu dari "empat konsep status" D-18 berada di bounded context tim lain | §4.2 |
| T-6 | **`StatusPosisi` bukan properti dan hanya punya satu nilai** (`'On Progress'`) | D-18 menghitungnya sebagai konsep status; buktinya tidak mendukung | §4.3 |
| T-7 | **Penjenjangan komite menugaskan ke operator bernama, bukan workbasket.** Seluruh export hanya punya **4** workbasket | Model B-6/B-7; menjelaskan kolom `STS_ABS` di `EMAILKOMITE` | §6.1, §6.3 |
| T-8 | **Nol Dynamic System Setting.** Konfigurasi dinamis nyata = tabel Oracle yang dikunci per **IP aplikasi** | Bertabrakan langsung dengan D-27 (dua instance di belakang load balancer) | §1.3 |
| T-9 | **Kredensial plaintext di dalam export**: 3 password SMTP di 31 lokasi + 1 pasang kredensial OAuth | Cara repo ini disimpan dan dibagikan, bukan hanya isi ADR | §7.6 |
| T-10 | **Otorisasi sistem lama = penyembunyian menu saja.** `pyPrivilegeName` terisi di **1 dari 902** activity | `FR-R1` terbukti dengan angka; F-3 membangun otorisasi dari nol, dan **0 tabel izin menu** ada di DB | §10.3 |
| T-11 | **Spesifikasi tabel baku U-2 salah sasaran**: median kolom **6**, bukan 18–27. Dan **3 fitur grid yang biasanya mahal tidak dipakai sama sekali** (tambah/hapus baris inline, resize) | Menyederhanakan U-2 secara nyata | §10.7 |
| T-12 | **`OFFSET 500000` tidak berdasar** — `OFFSET` nol kemunculan di export. Masalah nyatanya berbeda dan lebih buruk: **3.189 grid terikat page list klipboard**, `pyMaxRecords=500` di 54 dari 56 laporan | NFR-13 harus ditulis ulang; paginasi keyset adalah **perubahan perilaku**, bukan pemeliharaan | §10.7 |
| T-13 | **≥29 kelompok master, bukan 14** | `F-4` "Sedang–Besar" dan `U-6` "Sedang" undersized | §10.4 |
| T-14 | **Perubahan nilai uang klaim tidak punya jejak audit di sistem lama**; dan 2 tabel log terbukti dimutasi (`UPDATE`/`DELETE`) | Menguatkan D-28; memberi dua anti-pola konkret untuk tiket S-5 | §10.8 |

---

# 1. Autentikasi HCC/HCQ dan integrasi keluar

## 1.1 HCC/HCQ tidak ada di export

`HCC` dan `HCQ` muncul **2×** di seluruh 2.167 rule, dan keduanya teks pesan error yang menyuruh
pengguna menghubungi helpdesk:

| `berkas:baris` | Isi |
|---|---|
| `Activity/InputSurveyorBCAF_PostAct-Act.xml:3114` | teks `"…harap hubungi IT HCC untuk minta ditambahkan report To."` |
| `Activity/InputSurveyorBRIF_PostAct-Act.xml:1300` | teks identik |

Tidak ada Connect REST, activity, Data Page, maupun kelas integrasi HCC/HCQ. Autentikasi sistem
lama adalah **Operator ID bawaan Pega** (`Data-Admin-Operator*` dirujuk di 860 tempat;
`DATAPEGA.PR_OPERATORS` dibaca 3× dari `RDB List/`).

**Klasifikasi:** kategori (1) — HCC/HCQ benar-benar tidak dipakai sistem lama.

**Konsekuensi yang mengikat.** D-07 bukan migrasi, melainkan integrasi baru. Karena itu:

- Kontrak HCC/HCQ (nama field request/response, bentuk kode error, timeout, ada/tidaknya endpoint
  refresh atau validasi token) **harus datang dari pemilik API**. Nol bahan di export.
- **Acceptance criteria F-3 tidak boleh berbentuk "hasil identik Pega"** — tidak ada pembanding.
  Ini pengecualian terhadap gerbang 1 BRD §21.2 #3 yang belum tercatat di mana pun.
- Pemilik API HCC/HCQ adalah **pihak luar yang belum ada** di daftar sepuluh tindakan hari pertama
  (`16-RISK-ANALYSIS.md §Ringkasan`).

## 1.2 Dua belas Connect REST — inventaris lengkap

| Rule | `berkas:baris` | Tujuan | Dipanggil dari | Auth | Endpoint |
|---|---|---|---|---|---|
| `ClaimFeedback` | `Connect REST/ClaimFeedback-ConnectREST.xml:10` | Mitra bank (BRISurf) | `Activity/SendFeedbackBRISurf-Act.xml:797` | `pyUseAuthentication=false` (`:14`); token dari clipboard via header `Authorization` (`:103`) | literal (`:339`) — **host sandbox** |
| `GetTokenClaimBRISurf` | `…GetTokenClaimBRISurf-ConnectREST.xml:40` | OAuth mitra bank | `Activity/SendFeedbackBRISurf-Act.xml:774` | OAuth2 `client_credentials`; **`client_id` dan `client_secret` sebagai `pyMapFrom=Constant`** (`:117`, `:127`) | literal (`:269`) — **host sandbox** |
| `UploadDokumenPNC` | `…UploadDokumenPNC-ConnectREST.xml:30` | API Storage dokumen internal | `Activity/InsertDokumenPNC-Act.xml:772` | `false` (`:50`); token dari stored proc | literal (`:181`), path `/api/v1/upload` (`:183`) |
| `InjectDataArchiveDokumentKlaim` | `…:52` | Arsip dokumen internal | `Activity/SendDataArchiveDOcumentByService-Act.xml:1585` | `false` (`:39`) | literal (`:140`) |
| `KonversiAvif` | `…:55` | Konversi gambar internal | `Activity/Convert_Avif-Act.xml:2177` | `false` (`:63`) | literal (`:78`) — **HTTP tanpa TLS** |
| `RetrieveHistoryProductionPaymentData` | `…:31` | Pega Integration WS (kelas `ASM-FW-GISFW-Work`, `:69`) | `Activity/GetHistoryPayments_act-Act.xml:379` | `false` (`:23`) | literal **IP privat + port** (`:21`, `:355`) |
| `GeneratedPolisFileKlaim` | `…:15` | Generator file polis | `Activity/GetFilePolisByServiceKlaim-Act.xml:301` | `false` (`:24`) | `SETTING` (`:207`, `:208`) |
| `InjectDataRekeningToKasir` | `…:27` | Kasir | `Activity/HitDataRekeningToKasir-Act.xml:600`; `Activity/HitupdateDataRekeningToKasir-Act.xml:1845` | **Auth profile** (`:12`), `true` (`:51`) | `SETTING` (`:163`) |
| `UpdateSearchDataRekeningToKasir` | `…:70` | Kasir | `Activity/SearchRekeningKlaim-Act.xml:349` | **Auth profile** (`:23`) | `SETTING` (`:237`) |
| `ServiceToKBRURefKlaim` | `…:21` | KBRU | `Activity/ActKBRUForUpdateStatusKlaimRef_Adjuster-Act.xml:289` | **Auth profile** (`:59`) | `SETTING` (`:115`) |
| `ServiceOutstandingAcceptance` | `…:47` | Akseptasi | `Activity/HitServiceOSAkseptasiClaimNonMBU-Act.xml:1367` | `false` (`:23`) | **runtime** `=TempServiceResource.SearchName` (`:9`, `:180`) |
| `getPremiumPaidOn` | `…:19` | Sistem premi | `Activity/GetStatusPremi-Act.xml:592`; `Activity/SearchPolicy-Act.xml:2883`; `Activity/InsertObjectList-Act.xml:4075` | `false` (`:21`) | **runtime** (`:56`, `:293`) |

Rekap: **6 endpoint literal · 2 runtime dari clipboard · 4 dari `SETTING`**.

Tiga instance **Authentication Profile tidak ada di export** — hanya namanya yang dirujuk
(`ServiceRekKlaimPNC`, `SERVICEKLAIMKBRU`). Jenis auth persisnya
**TIDAK DAPAT DIPASTIKAN DARI EXPORT**. Klasifikasi: kategori (2) → Tim Pega.

## 1.3 Nol Dynamic System Setting — dan lapis konfigurasi nyata bertabrakan dengan D-27

Pencarian `Data-Admin-System-Setting`, `pyGetSystemSetting`, `getDynamicSystemSetting`,
`pxGetSystemSetting`, `@getDSS` di `Activity/`, `Connect REST/`, `Data Transform/`, `DataPage/`,
`Function/`: **0 hit**.

Lapis konfigurasi dinamis yang benar-benar dipakai adalah **tabel Oracle**:

| Bukti | `berkas:baris` |
|---|---|
| Tabel registry endpoint: `pooldata.GCNM_CONNECT_REST`, kolom `SERVICENAME`, `APPLICATIONIP`, `TYPESERVICE` | `RDB List/BrowseServiceName_sql-SQL.xml:64` |
| Dispatcher generik yang membacanya | `Activity/ConnectRestPNC_act-Act.xml:359`, `:975`, `:1013` |
| Hasilnya diisikan ke `TempServiceResource.SearchName` | `Activity/ConnectRestPNC_act-Act.xml:515` |
| Registry mengenal tipe layanan `LOGIN` | `RDB List/BrowseServiceName_sql-SQL.xml:64` (filter `TYPESERVICE = {TempError.LOGIN}`) |
| Pemetaan hostname → entitas: `pooldata.db_link_pega`, filter `appip LIKE '%'||<hostname>||'%'`; fallback `ASM` | `20-DETAIL-KOMITE-DBLINK.md:179`; `Activity/GetLinkAppClaim-Act.xml` step 3 |

**Endpoint dipilih berdasarkan IP instance aplikasi.** D-27 memutuskan minimal **dua instance di
belakang load balancer**, dan D-27 juga mewajibkan aplikasi **stateless**. Pola lama pecah begitu
ada dua IP. Konfigurasi tiga lapis (D-15) **tidak dapat memungut pola ini** — ia harus dirancang
baru. Ini kandidat ADR, bukan detail implementasi.

## 1.4 Pertanyaan yang hanya bisa dijawab work owner

1. **Kontrak API HCC/HCQ** — nama field request/response login, bentuk kode error, timeout,
   endpoint refresh/validasi. Nol bahan di export. Siapa pemiliknya dan sudah diminta belum?
2. **Isi `pooldata.GCNM_CONNECT_REST`** — daftar `SERVICENAME`/`TYPESERVICE`/`APPLICATIONIP` yang
   aktif, khususnya baris ber-`TYPESERVICE = LOGIN`. Datanya tidak ada di export.
3. **Dua integrasi BRI menunjuk host sandbox** (`ClaimFeedback-ConnectREST.xml:339`,
   `GetTokenClaimBRISurf-ConnectREST.xml:269`). Integrasi BRISurf tidak pernah hidup di produksi,
   atau produksi memang memanggil sandbox? Yang kedua adalah cacat berjalan.
4. **Auth profile `ServiceRekKlaimPNC` dan `SERVICEKLAIMKBRU`** — jenis auth dan pemilik
   kredensialnya.
5. **Endpoint runtime** `=TempServiceResource.SearchName` untuk `getPremiumPaidOn` dan
   `ServiceOutstandingAcceptance` — nilai apa yang dipakai di produksi, dan mengapa dua rule ini
   dinamis sementara enam lainnya hardcode.
6. **`RetrieveHistoryProductionPaymentData`** memakai IP privat + port literal
   (`:21`, `:355`). Endpoint ini masih hidup?

---

# 2. Penomoran klaim dan identitas

## 2.1 Sistem lama tidak pernah membentuk nomor klaim sendiri

| Yang dicari | Hasil | Bukti |
|---|---|---|
| `CLAIM_NO_NONPEGA_SEQ` | **0 hit** di 15 folder rule | hanya ada di `docs/` |
| `nextval` / `.NEXTVAL` | **0 hit** | — |
| Prefix `PNCN-` | **0 hit** | — |
| Prefix `PNC-` | **1 hit**, sebagai precondition diskriminator `@contains(.CaseID,"PNC-")` | `Activity/OpenProtection-Act.xml:1933` |

`RDB List/BroswseKlaimByKlaimNo-SQL.xml` membuktikan tabel `T_CLAIM_PNC` punya **dua kolom
identitas terpisah**:

- `CLAIMID` — identitas Pega, dibentuk `'ASM-FW-GCNMFW-WORK '||{InputData.CARI4}` (`:68`)
- `CLAIMNO` — nomor klaim bisnis, **selalu dialias `"EDMNO"`** (`:65`)

Alias `EDMNO` adalah buktinya: **nomor klaim diterbitkan oleh EDM, sistem di luar Pega.** Pega
hanya mengonsumsinya.

D-22 (`00-DECISION-LOG.md:524`) konsisten dengan ini — jawaban verbatim menyebut sequence
*"yang **akan** diberi nama `POOLDATA.CLAIM_NO_NONPEGA_SEQ`"*. Objeknya memang belum ada.

**Klasifikasi:** kategori (2) — sequence harus **dibuat** DBA. Ini dependensi DBA yang belum
tercatat di sepuluh tindakan hari pertama.

Idiom penomoran lokal yang benar-benar ada justru rawan tabrakan:

| Mekanisme | `berkas:baris` | Bentuk |
|---|---|---|
| `MAX(..)+1` | `RDB List/GetNewNoGroup-SQL.xml:35` | `to_number(nvl(max(NO_GROUP_RANGKA),0)+1)` |
| Hash MD5 + timestamp | `RDB List/GenerateImageID-SQL.xml:32-35` | `STANDARD_HASH('ASMPP'||TO_CHAR(SYSTIMESTAMP,…),'MD5')` |
| Stored proc + `COMMIT` di dalam SQL | `RDB List/GenerateTokenPNCDokumen-SQL.xml:44-45` | `GENERAL.GET_TOKEN_STORAGE(…)` lalu `COMMIT;` |

## 2.2 Prefix `ASM-FW-GCNMFW-WORK` adalah predikat join, bukan sekadar format

Ini fakta yang paling mengikat di area ini.

| Ukuran | Angka |
|---|---|
| Pola konkatenasi `'ASM-FW-GCNMFW-WORK ' \|\| <kolom>` di `RDB List/` | **97 kemunculan di 19 berkas** |
| Total kemunculan prefix di `RDB List/` | 723× di 116 berkas |
| Total kemunculan prefix di `Activity/` | **14.149×** |

Contoh berbukti: `RDB List/AmbilDataKlaimDenganNoRekening-SQL.xml:56` (`NO_KLAIM`) ·
`RDB List/BrowseClaimRCV_Aksep-SQL.xml:46` (`A.pnccaseid`) ·
`RDB List/CountSalvage_sql11_ASI-SQL.xml:46,55,63,72,81,90,99,108` (`B.NOKLAIM`) ·
`RDB List/ExportDataCloseKlaimNONMBU-SQL.xml:142,163,181`.

**Identitas Pega bukan kolom tersimpan, melainkan string turunan.** Membuang prefix (D-22) berarti
97 predikat join berhenti mencocokkan klaim `PNCN-`. Selama masa paralel, dibutuhkan kolom
identitas eksplisit atau lapisan kompatibilitas — bukan sekadar ganti format nomor. Ini konsekuensi
D-22 yang belum tercatat.

Nama kolom penampung nomor klaim tidak seragam — ini beban nyata untuk F-2:
`ClaimNo` 100 · `CLAIMNO` 71 · `claimno` 62 · `no_klaim` 8 · `NO_KLAIM` 6 · `NoClaim` 5 ·
`No_Klaim` 4 · `NO_CLAIM` 1 · ditambah `NOKLAIM` dan `pnccaseid`.

## 2.3 Pertanyaan yang hanya bisa dijawab work owner

1. **Siapa yang menerbitkan `CLAIMNO`/`EDMNO` hari ini?** Apakah EDM tetap sumber untuk klaim
   non-Pega, atau `PNCN-` menggantikannya sepenuhnya?
2. **97 predikat join berprefiks** — tabel Oracle itu akan dimigrasi, di-*dual-write*, atau tetap
   diakses sistem lain di luar Claim PNC? Ini menentukan apakah prefix boleh benar-benar dibuang.
3. **Sequence `POOLDATA.CLAIM_NO_NONPEGA_SEQ` sudah diminta ke DBA?** Ia belum ada, dan tidak ada
   di daftar sepuluh tindakan hari pertama.
4. **Urutan terbit nomor versus commit** — `TIDAK DAPAT DIPASTIKAN DARI EXPORT`. Karena nomor
   menjadi *foreign key* logis di belasan varian nama kolom, ia harus terbit **sebelum** insert
   baris anak. Benar dalam praktik?

---

# 3. Penyimpanan dokumen — tiga mekanisme, dan satu yang keempat

> ⚠️ **DIKOREKSI di §14.2.** Klaim "mekanisme keempat" (Google Cloud Storage) **dicabut** —
> `Activity/UploadDocumentToGoogleStorage-Act.xml` ternyata hanya pembungkus yang mendelegasikan
> ke `InsertDokumenPNC`, yaitu mekanisme #2. **D-16 dan `FR-S1` benar: tiga mekanisme.**

D-16 dan `FR-S1` (`docs/BRD.md:483`) menyatakan ada **tiga** mekanisme yang akan disatukan.
Bukti mendukung ketiganya, **dan menemukan yang keempat**.

| # | Mekanisme | Rule + `berkas:baris` | Jenis dokumen | Status |
|---|---|---|---|---|
| 1 | **Lampiran bawaan Pega** (`Data-WorkAttach` di 92 berkas · `Link-Attachment` 75 · `pyAttachStream` 15) | UI `Section/ASMAttachContentScreen-Section.xml`, `Section/SetUploadDoc_Detl-Section.xml`; hapus `Activity/DeleteAttachment-Act.xml`, `Activity/KlaimDeleteAttachmentFromDB-Act.xml`; temp file `Activity/ASMCollectAttachments-Act.xml:1281` (`java.io.File`) | DLA, LOD, SPS, PDF polis, lampiran email | **hidup** |
| 2 | **API Storage internal** (kelas `ASM-FW-GCNMFW-Int-API_Storage`) | `Connect REST/UploadDokumenPNC-ConnectREST.xml:30`, path `:183`; dipanggil `Activity/InsertDokumenPNC-Act.xml:772` (langkah `Connect-REST` `:398`); token `RDB List/GenerateTokenPNCDokumen-SQL.xml:44` via `GENERAL.GET_TOKEN_STORAGE`; URL + kedaluwarsa `RDB List/GetURLAndEXPDate-SQL.xml` | dokumen klaim bertipe, gambar AVIF | **hidup — jalur paling aktif** |
| 3 | **CLOB di Oracle** | `RDB List/SaveAttachmentToDBTemp_Sql-SQL.xml:37-47` (`TATTACHFILE clob` `:44`) → `POOLDATA.SET_ATTACHFILETEMPSALVAGE(…)` + `COMMIT`; pemanggil `Activity/PNCSaveAttachmentToDBTemp-Act.xml:535` ← `Activity/CNMUpdateMasterRekening_act-Act.xml:2185` | lampiran salvage & bukti rekening | **hidup, hanya 1 pemanggil** |
| **4** | **Google Cloud Storage** | `Call UploadDocumentToGoogleStorage` di `Activity/SetStsSalvagePNC_act-Act.xml:2443`, `:2448`, `:2453`; precondition baca `ViewGoogleStorage.Storage==true` di `Activity/ASMSendsEmailAttachments_PDF-Act.xml:5359` | salvage | **rule-nya HILANG dari export** |

Mekanisme 4 penting dan belum tercatat sebagai mekanisme:

- Activity `UploadDocumentToGoogleStorage` **tidak ada di export** (tercatat di
  `19-GAP-EXPORT-DETAIL.md:217` sebagai activity hilang, tapi tidak dikenali sebagai mekanisme
  penyimpanan keempat).
- Hanya **satu** call site, dan hanya di varian PNC — varian
  `Activity/SetStsSalvagePNC_act_ASI-Act.xml` **tidak** memanggilnya (0 hit `GoogleStorage`);
  ia memanggil `SetUploadDocumentToPanelDocumentList` di `:3428`.
- Pola ini konsisten dengan **rollout parsial yang di-gate flag**, bukan warisan mati.
- Bucket, skema penamaan objek, kredensial, dan pembacanya:
  **TIDAK DAPAT DIPASTIKAN DARI EXPORT**. Klasifikasi kategori (2).

**Tidak ada mekanisme yang terbukti mati.** Yang paling dekat adalah mekanisme 3 (satu pemanggil,
bukan nol).

**Yang mengikat arsitektur.** Ketiga jalur hidup punya sifat transaksional berbeda, dan dua di
antaranya melakukan **`COMMIT` di dalam stored procedure**
(`RDB List/GenerateTokenPNCDokumen-SQL.xml:44-45`, `RDB List/SaveAttachmentToDBTemp_Sql-SQL.xml`).
Artinya penyimpanan dokumen **tidak dapat dibatalkan bersama** transaksi klaim. Jalur tunggal yang
baru harus memilih secara sadar antara pola *outbox* atau menerima dokumen yatim — itu keputusan
ADR, bukan detail.

## 3.1 Pertanyaan yang hanya bisa dijawab work owner

1. **`UploadDocumentToGoogleStorage`** — bucket, skema penamaan objek, kredensial, siapa pembaca
   objeknya. Siapa yang menyetel `ViewGoogleStorage.Storage`, dan apakah jalur ini sudah GA atau
   masih pilot untuk salvage saja?
2. **D-16 menyebut tiga mekanisme; buktinya empat.** Apakah GCS memang mekanisme terpisah yang
   harus ikut disatukan, atau bagian dari API Storage internal?
3. **Semantik `COMMIT` di dalam stored proc** (`POOLDATA.SET_ATTACHFILETEMPSALVAGE`,
   `GENERAL.GET_TOKEN_STORAGE`) — ada pemanggil lain yang bergantung padanya, atau boleh diubah
   agar ikut transaksi pemanggil?

---

# 4. Empat konsep status — bukti tidak mendukung keempatnya

D-18 (`00-DECISION-LOG.md:385-409`) memutuskan `StatusWork` · `StatusClaim` · `ClaimStatus` ·
`StatusPosisi` adalah empat konsep berbeda yang **semuanya dipertahankan terpisah**.
Hasil verifikasi: **dua bertahan, satu milik tim lain, satu bukan status.**

## 4.1 `StatusWork` — sebenarnya dua properti, bukan satu

| Properti | Penulis | Penyimpanan | Nilai yang muncul |
|---|---|---|---|
| `.pyStatusWork` (Pega OOTB) | **penulis tunggal** `Activity/UpdateStatus-Act.xml:1401`. Pemanggil: `Activity/pzUpdateAndDeleteAssignments-Act.xml:1171`, `Activity/SuspendFlows-Act.xml:424`. End-shape flow: `Flow/Register_Flow.xml:2594,2984`; `Flow/Komite_Flow.xml:986`; `Flow/InputReceiveDocument.xml:819`; `Flow/CreateProtection_Flow.xml:292,540` | `DATAPEGA.PC_ASM_FW_GCNMFW_WORK.PYSTATUSWORK` (`RDB List/BrowseClaimALL-SQL.xml:92-96`) | `New`, `Open`, `Pending-PolicyOverride`, `Resolved-Completed`, `Resolved-Rejected` |
| `.ClaimData.StatusWork` (kustom) | **23 titik penulisan**, mis. `Activity/InsertCasePNCAuto_Travel-Act.xml:3580,5651`; `Activity/InsertCasePNC_AutoClaim-Act.xml:2905,4980,6521`; `Activity/KomitePost_Reject-Act.xml:3456` | `T_CLAIM_PNC.STATUSWORK` (`RDB List/BroswseKlaimByName-SQL.xml:14-17`) | mirror Pega (`Resolved-Completed`/`Resolved-Rejected`) **dan** domain terpisah `Close Case`/`Final Report`/`Invoice Fee` (`Activity/ExportKPILoginAdjuster-Act.xml:5169,5536`) |

`.pyStatusWork` punya **satu penulis** — mudah dipusatkan menjadi satu state machine.
`.ClaimData.StatusWork` punya **23 penulis** dan **dua domain nilai yang tidak kompatibel**.

**Tidak ada `INSERT`/`UPDATE` ke `T_CLAIM_PNC` di seluruh `RDB List/`** — siapa yang menulis
kolom itu **TIDAK DAPAT DIPASTIKAN DARI EXPORT**. Klasifikasi kategori (2): job, trigger, atau
integrasi di luar Pega.

## 4.2 `ClaimStatus` — milik ruleset GISFW, bukan Claim PNC

Terverifikasi ulang oleh orkestrator. `ClaimStatus` muncul di **tepat 5 berkas**, dan kelimanya
berkelas **`ASM-FW-GISFW-Work`**:

| Berkas | Kelas |
|---|---|
| `Activity/RetrieveClaimData_MBU-Act.xml` | `ASM-FW-GISFW-Work` |
| `Activity/RetrieveClaimData_NonMBU-Act.xml` | `ASM-FW-GISFW-Work` |
| `Activity/CheckPaymentData_MBU-Act.xml` | `ASM-FW-GISFW-Work` |
| `Activity/CheckOutGoData_MBU-Act.xml` | `ASM-FW-GISFW-Work` |
| `Activity/Edm_SetDataDetailGeneral_Act-Act.xml` | `ASM-FW-GISFW-Work` |

Properti yang dirujuk berkelas `ASM-FW-GISFW-Data-Coverage` dan `ASM-FW-GISFW-Data-Vehicle`
(`Activity/RetrieveClaimData_MBU-Act.xml:489`, `:863`). Nilai yang dibandingkan hanya `"1"`
(`:4889`), dan di-set lewat
`@if(Local.ClaimType=="STOLEN",1,@if(Local.ClaimType=="CTLO",1,0))` (`:3384`, `:4632`).

Tidak ada kolom `CLAIMSTATUS` di seluruh `RDB List/` — **tidak ada bukti ia dipersistensi**.

**Ini flag biner per coverage/vehicle di sisi polis, bukan tahap alur klaim.** D-03 dan D-04
menempatkan GISFW **di luar kepemilikan tim ini**. Jadi salah satu dari "empat status yang
dipertahankan" berada di bounded context tim lain.

## 4.3 `StatusPosisi` — bukan properti, dan hanya satu nilai

| Uji | Hasil |
|---|---|
| Ada sebagai `<PropertiesName>` properti work object? | **tidak** — hanya `Param.statusposisi`, **20 kemunculan** |
| Sebaran | `Activity/` 13 berkas · `RDB List/` 2 berkas · `Data Transform/`, `When/`, `Section/`, `Harness/`, `Report Definition/` **nol** |
| Nilai literal yang muncul | **hanya `'On Progress'`** |

Jalur tulisnya: `Activity/PNCInsertProgressClaim-Act.xml:793` → `tempinsertprogress.RWID` ←
`Param.statusposisi`, lalu stored proc `POOLDATA.PROGRESS_CLAIM_PNC(…)` di
`RDB List/InsertStatusProgress-SQL.xml:42-47`. Penyimpanan:
`POOLDATA.GCNM_PROGRESS_POSISI_PNC.STATUSPOSISI`. Dibaca di
`RDB List/GetProgressAllYearDashboarOS-SQL.xml:12` (`WHERE e.statusposisi = 'On Progress'`).

Dipanggil dari 14 activity, dan **2 pemanggil mengirim nilai kosong**
(`Activity/CreateNewCaseRCV-Act.xml:2455`, `Activity/InputRegister_act-Act.xml:20894`).

Fungsinya lebih dekat ke **penanda "baris progres aktif"** pada jejak audit progres daripada ke
status. Isi `POOLDATA.PROGRESS_CLAIM_PNC`: kategori (2) → DBA.

## 4.4 `StatusClaim` — bertahan, dan R-06 bisa ditutup sebagian besar tanpa DBA

**79 titik penulisan** tersebar di `Activity/` dan `Data Transform/`. Penyimpanan:
`DATAPEGA.PC_ASM_FW_GCNMFW_WORK.STATUSCLAIM_1` (`RDB List/BrowseClaimALL-SQL.xml:89`) dan
`T_CLAIM_PNC.StatusClaim` (`RDB List/BroswseKlaimByName-SQL.xml:16`).

Kode yang benar-benar di-set sebagai nilai — **terverifikasi ulang oleh orkestrator**:

| Kode | Kemunculan sbg nilai | Rule anchor | Konteks |
|---|---|---|---|
| `1142` | 1 | `Activity/KomitePost_Reject-Act.xml:1875` | keputusan Komite = **Reject** |
| `1143` | 4 | `Activity/InsertCasePNC_AutoClaim-Act.xml:2814`; `…_Kredit_PA-Act.xml:3571`; `…_AsuransiKredit-Act.xml:5430`; `Activity/SetDataClaimKredit-Act.xml:1776` | **status awal** saat case dibuat |
| `1144` | 4 | `Activity/InsertCasePNC_AutoClaim-Act.xml:5030`; `…_Kredit_PA-Act.xml:5287`; `…_AsuransiKredit-Act.xml:7720`; `Activity/InsertCasePNCAuto_Travel-Act.xml:5699` | jalur **reject/penutupan** (berdampingan dgn `StatusWork="Resolved-Rejected"`) |
| `1145` | 1 | `Activity/KomitePost_Survey-Act.xml:19878` | posting **hasil survey** Komite |
| `1146` | 2 | `Activity/InputRegister_act-Act.xml:1896`; `Activity/copy_DisplayPolis_act-Act.xml:1187` | **input register** / display polis |
| `1147` | 1 | `Activity/CallActivityInputRegister-Act.xml:5670` | wrapper pemanggil InputRegister |
| **`1148`** | **0** | — | **tidak pernah di-set** |
| `1149` | 2 | `Activity/SetListComiteeClaimAI-Act.xml:15136`; `Activity/SetListComiteeClaimPerObjAdj-Act.xml:27282` | pembentukan **daftar komite** |
| `1150` | 2 | set: `Activity/AutoPrintPDFDraftLOD-Act.xml:12461`; `Activity/DownloadAutoProposeAdjustment-Act.xml:11122`. **dibandingkan**: `RDB List/CountClaimRegistTravPA-SQL.xml:189` (`WHEN A.STATUSCLAIM_1='1150' THEN 1`) | penanda **"sudah teregister"** |
| `1151` | 1 | `Activity/SetStatusInvestigator_Act-Act.xml:1962` | penetapan status **Investigator** |

> ⚠️ Seluruh kemunculan `1142`–`1151` di `Section/InboxKomite_section-Section.xml` dan
> `Harness/InboxKomite_Harness-Harness.xml` adalah `<pyCellId>` (ID sel UI) — **bukan** nilai
> status. Jangan dipakai sebagai bukti.

Tabel master status: `POOLDATA.V_STS_CLAIM`, kolom `LSC_ID` (kode) + `LSC_NOTE` (label).
Lookup tunggal di `RDB List/GetStatusKlaim-SQL.xml:6`; join denormalisasi di **13 rule**
(`RDB List/BroswseKlaimByDOL-SQL.xml:22`, `…ByPolicyNo-SQL.xml:102`, `BrowseClaimALL-SQL.xml:89`,
dst). Kelas Pega untuk view: `ASM-FW-GCNMFW-Int-V_STS_CLAIM`
(`Report Definition/SelectVStsClaim_RD-RD.xml:64`).

**Konsekuensi untuk R-06.** Sembilan dari sepuluh kode punya rule-anchor yang menunjukkan
konteks penetapannya. Yang benar-benar butuh DBA hanyalah **label `LSC_NOTE`** — bukan arti
fungsionalnya. R-06 dapat diturunkan dari "memblokir seluruh modul berstatus" menjadi
"melengkapi label".

## 4.5 Pertanyaan yang hanya bisa dijawab work owner

1. **D-18 perlu direvisi?** `ClaimStatus` milik GISFW (§4.2) dan `StatusPosisi` bukan status
   (§4.3). Apakah keputusan "empat konsep dipertahankan terpisah" tetap berlaku, atau menjadi
   **dua konsep** (`Status Proses`, `Status Klaim`) + satu penanda progres + satu properti milik
   tim GISFW?
2. **Mengapa `1148` tidak pernah di-set?** Pernah ada lalu ditinggalkan, atau hanya di-set oleh
   rule/procedure yang tidak ada di export?
3. **Siapa/apa yang menulis `T_CLAIM_PNC.STATUSWORK`?** Tidak ada satu pun DML ke tabel itu di
   export.
4. **`.ClaimData.StatusWork` dengan nilai `Close Case`/`Final Report`/`Invoice Fee` (adjuster)
   versus `Resolved-Completed` (case)** — properti yang sama pada baris data yang sama, atau dua
   kelas work berbeda yang berbagi nama properti?
5. **`statusposisi`** — pernah ada nilai selain `'On Progress'` di produksi? Satu query:
   `SELECT DISTINCT statusposisi FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC`.

---

# 5. Transisi lateral lewat Ticket rule (FR-W2)

## 5.1 Tujuh belas nama Ticket rule dirujuk; delapan punya rule

`pyTicketShapes` (`pxObjClass=Data-MO-Event-Exception`) di 4 flow merujuk **17 nama unik**;
folder `Ticket/` berisi **8** rule.

| Flow | Nama Ticket rule yang dirujuk |
|---|---|
| `Flow/CreateProtection_Flow.xml` | `TC_PNCInputProtection`, `Akp_PNCInputProtection` |
| `Flow/InputReceiveDocument.xml` | `SendToReceiveDocument` |
| `Flow/Komite_Flow.xml` | `komiteAccept_ticket`, `KomiteAssign_ticket` |
| `Flow/Register_Flow.xml` | `SendtoAnalysator`, `SendToEstimatorPA`, `SendToEstAdmin`, `RCLDokter`, `Status-Resolved`, `setToRegister_ticket`, `SendtoPUCL`, `SendToEstTravel`, `AcceptanceKomite`, `SendToInvestigator`, `SendToPICTravel`, `CompliancePNC` |

**Sembilan dirujuk tanpa rule** — dikonfirmasi tidak ada di `Ticket/` maupun sebagai rule inline
di folder lain:

| Ticket rule | Dirujuk dari | Klasifikasi |
|---|---|---|
| `SendToReceiveDocument` | `Flow/InputReceiveDocument.xml:690` | (2) Tim Pega |
| `komiteAccept_ticket` | `Flow/Komite_Flow.xml:796` | (2) Tim Pega |
| `KomiteAssign_ticket` | `Flow/Komite_Flow.xml:974` | (2) Tim Pega |
| `RCLDokter` | `Flow/Register_Flow.xml:2954` | (2) Tim Pega |
| `Status-Resolved` | `Flow/Register_Flow.xml:3004` | **(1) bawaan Pega** — dipicu activity OOTB `Work-.Resolve` lewat `Obj-Set-Tickets` (`Activity/Resolve-Act.xml:1331`, nama ticket `:1343`) |
| `setToRegister_ticket` | `Flow/Register_Flow.xml:3052` | (2) Tim Pega |
| `SendtoPUCL` | `Flow/Register_Flow.xml:3197` | (2) Tim Pega |
| `SendToInvestigator` | `Flow/Register_Flow.xml:3624` | (2) Tim Pega |
| `CompliancePNC` | `Flow/Register_Flow.xml:3844` | (2) Tim Pega |

> **Asal angka "11 ticket" di FR-W2 (`docs/BRD.md:518`)**: `Flow/Register_Flow.xml` memuat **12**
> shape ticket, dikurangi `Status-Resolved` (bawaan Pega) = **11 custom**. Jadi angka itu benar
> untuk Register Flow saja, bukan untuk seluruh aplikasi. Angka yang benar untuk seluruh aplikasi:
> **17 dirujuk, 8 ada, 9 hilang.** Perlu konfirmasi definisi.

## 5.2 Hanya tiga transisi lateral yang punya pemicu di export

| Ticket rule | Dipicu dari | Kondisi pemicu | Shape tujuan |
|---|---|---|---|
| `SendToEstimatorPA` | `Activity/KomitePost_Adjustment-Act.xml:15083` (step `call SetTicket`, `:14998`) | precondition **`IsPA`** (`:15033`); deskripsi step *"Pindahin ke inputor apabila semua komite sudah aksep"* (`:14993`) | assignment `Estimation`, `Flow/Register_Flow.xml:2485-2513` |
| `SendToEstimatorPA` | `Activity/SetListComiteeClaimPerObjAdj-Act.xml:28380` (`:28357`) | precondition **`IsPA`** (`:28426`); *"Back to estimator"* | idem |
| `SendtoPUCL` | `Activity/KomitePost_Reject-Act.xml:3172` (`:3096`) | **`pyWorkCover.Policy.Quotation.GroupPanel=="002"`** (`:3121`) | assignment `RCL/PUCL`, `Flow/Register_Flow.xml:3182-3197` |

**Tiga belas nama Ticket rule tidak punya pemicu apa pun** di 902 Activity + 29 Flow Action.
Metode pemicu yang dicari dan jumlah temuannya: `Obj-Set-Tickets` **1** (hanya
`Activity/Resolve-Act.xml:1331`, dan Pega menandainya *deprecated* di `:153`) ·
`call SetTicket` **4** · `<Ticket>` sebagai parameter **4** · `<SetTicketNames>` **1** ·
rujukan ticket di `Flow Action/` **0**.

Dan activity `SetTicket` sendiri **tidak ada di export** — hanya dirujuk sebagai dependensi
(`Activity/KomitePost_Adjustment-Act.xml:22389`). Jadi **pemicu maupun definisinya hilang**.

Satu anomali: ticket `AllCoveredResolved` **dipicu** di `Activity/AllCoveredResolved-Act.xml:198`
tetapi **tidak ada shape penangkapnya** di 4 flow.

## 5.3 Tidak ada satu pun pagar otorisasi pada transisi lateral

| Yang dicari | Lingkup | Hasil |
|---|---|---|
| `<pyPrivilegeName>` non-kosong | `Flow Action/` (29), `Flow/` (4), `Harness/` (74), `Ticket/` (8) | **0 hit** — semua kosong; hanya `pyPrivilegeClass` terisi (mis. `Flow Action/SendToRCLDokter-FA.xml:243-244`) |
| `<pyWhenName>` non-kosong | `Flow Action/` | **0 hit** |
| `pyWhenNotPrivilege`, `pyPrivilegeView`, `pyPrivilegeUpdate` | `Flow Action/` | kosong (`SendToRCLDokter-FA.xml:357,650,656`) |

Gating yang ada hanyalah **When rule bisnis** (`IsPA`) dan ekspresi data (`GroupPanel=="002"`) —
bukan kontrol keamanan. Pemeriksaan keamanan yang ada berada di jalur buka-kerja OOTB
(`Activity/OpenAndLockWork-Act.xml:1872`), bukan di jalur ticket.

**Yang mengikat arsitektur.** State machine sistem baru **tidak boleh** dirancang sebagai "tiru
perilaku Pega": peta transisi lateral yang benar-benar berjalan di produksi tidak dapat
direkonstruksi dari export (13 dari 17 tanpa pemicu, `SetTicket` hilang, `Flow Action/` nol
rujukan). Dan ketiadaan pagar otorisasi adalah sesuatu yang harus **diperbaiki secara sadar**,
bukan direplikasi — yang berarti ia perlu masuk daftar perbaikan eksplisit (P-5), atau tiketnya
melanggar prinsip.

## 5.4 FR-W1 terkonfirmasi persis

`Flow/Register_Flow.xml`, hitungan `<pxObjClass>Data-MO-*`:

| Jenis | Jumlah |
|---|---|
| `Data-MO-Event-Start` | 1 |
| `Data-MO-Activity-Assignment` | **13** |
| `Data-MO-Gateway-Decision` | **7** |
| `Data-MO-Event-End` | 2 |
| **Total shape** | **23** |

FR-W1 (*"23 shape, 13 assignment, 7 decision"*) **cocok persis**. Flow lain:
`CreateProtection_Flow` 6 shape · `Komite_Flow` 4 · `InputReceiveDocument` 3.

## 5.5 Pertanyaan yang hanya bisa dijawab work owner

1. **Tiga belas Ticket rule tanpa pemicu** — dipicu dari ruleset lain yang tidak ikut export, dari
   aksi Section/harness, atau memang sudah mati?
2. **Saat ticket dipicu, assignment yang sedang aktif dibatalkan, diselesaikan, atau ditinggalkan
   menggantung?** `TIDAK DAPAT DIPASTIKAN DARI EXPORT` — `SetTicket` hilang, dan tidak ada flag
   `cancel`/`withdraw`/`pyDeleteAssign` di `Flow/`. Ini harus diuji di sistem berjalan.
3. **Siapa (peran) yang boleh memicu setiap lompatan lateral?** Export menunjukkan **nol** pagar
   teknis. Kontrolnya prosedural/manual, atau memang tidak ada?
4. **`CompliancePNC` dipakai serentak sebagai nama Ticket rule dan nama workbasket**
   (`Flow/Register_Flow.xml:3844` vs `:3769`). Disengaja, atau tabrakan nama yang menyebabkan bug?
5. **`AllCoveredResolved`** masih relevan? Dipicu tapi tidak ada shape penangkapnya.
6. **Konfirmasi definisi "11 ticket"** di FR-W2 — Register Flow saja, seperti dugaan §5.1?

---

# 6. Penugasan, router, penguncian, dan inbox

## 6.1 Hanya empat workbasket di seluruh export

Terverifikasi ulang oleh orkestrator — seluruh nilai `<pyWorkBasket>` non-kosong di
`Activity/`, `Data Transform/`, `Flow/`, `Flow Action/`, `Report Definition/`, `Harness/`,
`Section/`:

| Workbasket | `berkas:baris` | Assignment pemakai |
|---|---|---|
| `ProtectionPNC` | `Flow/CreateProtection_Flow.xml:464` (`WorkBasket` `:454`, router `ToWorkbasket` `:497`) | shape CreateProtection |
| `RCLPUCL` | `Flow/Register_Flow.xml:3159` (`:3173`, `:3226`) | `RCL/PUCL` (`:3185`) |
| `InvestigatorPNC` | `Flow/Register_Flow.xml:3568` (`:3565`, `:3592`) | `Investigator` (`:3560`) |
| `CompliancePNC` | `Flow/Register_Flow.xml:3769` (`:3772`, `:3785`) | `Compliance` (`:3747`) |

Sisanya worklist per-operator: 10 `WorkList` di `Flow/Register_Flow.xml`
(`:2178,2322,2395,2499,2631,2898,3031,3257,3406,3647`) + 3 `WorkBasket` = 13 assignment.

**Dua koreksi terhadap `19-GAP-EXPORT-DETAIL.md`:**

| Klaim dokumen | Bukti | Verdict |
|---|---|---|
| `19-GAP:154` — *"workbasket `komitepnc1..4`"* | Nilai persisnya **`komitepnc`** (tanpa angka), `komitepnc2`, `komitepnc3`, `komitepnc4` — `Activity/KomiteRouter-Act.xml:326,477,580,724`. Dan **bukan** workbasket: assignment ber-`KomiteRouter` bertipe `pyImplementation=WorkList` (`Flow/Komite_Flow.xml:897`), dan tidak satu pun `<pyWorkBasket>` bernilai `komitepnc*` | ❌ salah pada nama **dan** jenis |
| `19-GAP:156` — *"workbasket `ReceiveDocument.UserAdmin`"* | Properti operator, bukan workbasket — `Activity/PNCAdminRouterRCV-Act.xml:407`, `:473` | ❌ salah |

## 6.2 Router — empat hilang, empat ada

| Router | Ada? | Bukti | Dirujuk dari |
|---|---|---|---|
| `PNCAdminRouter` | ❌ | — | `Flow/Register_Flow.xml:2552`, `:2674`, `:3069`, `:3332` (**4 shape**) |
| `PNCTeknikRouter` | ❌ | — | `Flow/Register_Flow.xml:2226`, `:3440`, `:3696` (**3 shape**) |
| `RouterRCLDokter` | ❌ | — | `Flow/Register_Flow.xml:2914` (**1 shape**) |
| `ToWorkList` | ❌ | — | `Flow/Register_Flow.xml:2452` (**1 shape**) |
| `KomiteRouter` | ✅ | `Activity/KomiteRouter-Act.xml`, ruleset `GCNMFW` v`01-01-50` | `Flow/Komite_Flow.xml:923` |
| `PNCAdminRouterRCV` | ✅ | `Activity/PNCAdminRouterRCV-Act.xml`, `GCNMFW` v`01-01-86` | `Flow/InputReceiveDocument.xml:712` |
| `ToCurrentOperator` | ✅ | `Activity/ToCurrentOperator-Act.xml`, `Pega-ProcessEngine` | `Flow/Register_Flow.xml:2350`; `Flow/CreateProtection_Flow.xml:370` |
| `ToWorkbasket` | ✅ | `Activity/ToWorkbasket-Act.xml` — logika `:324`, `:400`, `:403` (`param.AssignTo` ← `param.Workbasket`) | `Flow/Register_Flow.xml:3226,3592,3785`; `Flow/CreateProtection_Flow.xml:497` |

Tabel `19-GAP:133-146` (**4 hilang, 4 ada**, dengan jumlah shape per router) **terkonfirmasi
persis**. Klasifikasi `ToWorkList` sebagai bawaan Pega adalah **inferensi kuat** — `ToWorkbasket`
dan `ToCurrentOperator` keduanya `Pega-ProcessEngine` — tetapi statusnya sendiri
`TIDAK DAPAT DIPASTIKAN DARI EXPORT` karena rule-nya tidak ada.

> Catatan konsistensi: `06-MODULE-BREAKDOWN.md:119` dan R-04 menyebut "3 router". `19-GAP:147`
> sudah merekonsiliasinya: **4 hilang, 3 kritis buatan sendiri**. Bukan kontradiksi.

## 6.3 `KomiteRouter` — satu-satunya contoh penjenjangan komite yang ada source-nya

Seluruh langkah adalah `Property-Set` yang mengisi `param.AssignTo`.

| Kondisi | `param.AssignTo` ← | `berkas:baris` (kondisi / nilai) |
|---|---|---|
| `.KomiteCount==1` | `"komitepnc"` | `:382` / `:326` |
| `.KomiteCount==2` | `"komitepnc2"` | `:455` / `:477` |
| `.KomiteCount==3` | `"komitepnc3"` | `:669` / `:580` |
| `.KomiteCount==4` | `"komitepnc4"` | `:823` / `:724` |
| `.KomiteAproval==0` | `.KomiteID` | `:1212` / `:1110` |
| `Primary.TransferType=='3'` | `.Komite.KomiteID` | `:1296` / `:1365` |
| (tanpa kondisi) | `.KomiteCount` ← `.KomiteCount+1`; `.AcceptStatus` ← `""` | `:903-905`, `:947-953` (desc `:861`) |

Loop dikendalikan `When/IsKomiteLoop-When.xml`: kondisi 1 = `.AcceptStatus = "1"` (`:166`),
kondisi 2 = `.KomiteCount <= .KomiteLoop` (`:285`). Dipakai sebagai decision di
`Flow/Komite_Flow.xml:539`. Sumber nilai `.KomiteLoop`:
**TIDAK DAPAT DIPASTIKAN DARI EXPORT** — tetapi `20-DETAIL-KOMITE-DBLINK.md:26-30` menunjukkan
`KomiteLoop := TempRDBSearchEmailKomite.pxResultCount`, yaitu jumlah baris hasil query ke
`POOLDATA.EMAILKOMITE`. Klasifikasi kategori (2) → DBA.

**Tidak ada langkah untuk `KomiteCount > 4`.**

**Yang mengikat arsitektur.** Jenjang komite adalah **penugasan ke operator bernama**, satu titik
kegagalan per jenjang. Ini menjelaskan mengapa `POOLDATA.EMAILKOMITE` punya kolom `STS_ABS`
(*"user sedang tidak aktif"*, `20-DETAIL:52`). Sistem baru harus memutuskan secara sadar:
pertahankan penugasan per-orang, atau naikkan ke antrean berbasis peran. Itu keputusan ADR.

## 6.4 Penguncian — pesimistik OOTB, nol kustomisasi

| Bukti | `berkas:baris` |
|---|---|
| Handle lock = `pzInsKey` work object (atau `pxCoverInsKey` bila lock parent) | `Activity/DetermineLockString-Act.xml:394-395`, `:541-542`; keputusan `:343` |
| Akuisisi: `Obj-Open-By-Handle` dengan `Lock=-1`, *release on commit* | `Activity/WorkLock-Act.xml:623-651` (`ReleaseOnCommit=-1` `:653`); *lock and hold* `:717-765` |
| Refresh + lock | `Activity/WorkLock-Act.xml:811-896`, `:902-988` |
| Cek pemilik lock | `Activity/WorkLock-Act.xml:1295`, `:1303`, `:1316` |
| Alur buka-kerja | `Activity/OpenAndLockWork-Act.xml:1337`, `:1461-1469`, `:1872`, `:2008-2016` |
| Jalur optimistik **ada tapi tidak diaktifkan untuk kelas apa pun** | `Activity/OpenAndLockWork-Act.xml:1644`, `:2098`; `Activity/pzGetLockingMode-Act.xml:336`, `:396` — semua bergantung `.pyCaseUpdateInfo.pyLockingMode=="Optimistic"` |

Dicari dan **tidak ditemukan**: `pyLockInfo` · `Page-Lock` · `AcquireLock` · kolom `pzLock*` di
`RDB List/` · `<LockRecord>true</LockRecord>` (0 dari 902 activity → tidak ada `Obj-Open` kustom
dengan lock). Nilai `pyLockingMode` per case type
**TIDAK DAPAT DIPASTIKAN DARI EXPORT** (rule Case Type tidak ada di export).

**Yang mengikat arsitektur.** Nol kustomisasi berarti sistem baru **bebas memilih model
penguncian sendiri** tanpa perlu meniru semantik lama. Ini melonggarkan satu batasan yang
sebelumnya diasumsikan mengikat (D-26).

## 6.5 Dua puluh enam inbox berbasis peran, bukan ~20

Sumber otoritatif: `Navigation/pyCaseWorkerNavigation-Navigation.xml` — 26 pasangan
`<pyHarnessName>` + `<pyWhen>`. **22 harness ada, 4 hilang.**

| Harness hilang | `berkas:baris` navigasi | When |
|---|---|---|
| `InboxServiceCenter` | `:4770` | `IsServiceCenterPNC` (ada) |
| `InboxCloseClaim_Harness` | `:17035` | `IsGCNMUser` (**hilang**) |
| `InboxRequestSalvage` | `:26221` | `IsGCNMUser` (**hilang**) |
| `InboxOutstanding_Harness` | `:36340` | `IsGCNMUser` (**hilang**) |

**Enam When rule peran dirujuk navigasi tapi hilang dari `When/`**: `IsSurvey` (`:7749`),
`IsKomite` (`:11386`), **`IsGCNMUser`** (11 entri navigasi: `:17035,26221,29344,31861,36574,36697,36744,38524,38743,38977,39718`),
`IsGCNMReport` (`:33664`), `IsPNCBonding` (`:33715`), `IsNotViewClaim` (`:34673`).

`FR-U3` menulis *"~20 inbox berbasis peran"* — angka sebenarnya **26**. Ini menaikkan ukuran U-3.

## 6.6 Pertanyaan yang hanya bisa dijawab work owner

1. **Logika `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter`** — atas dasar apa memilih
   operator (cabang? skill? beban kerja? tabel master PIC)? Ini **8 dari 13** shape assignment di
   flow utama.
2. **Nama urutan pertama komite: `komitepnc` atau `komitepnc1`?** Dan apakah keempatnya operator
   ID atau workbasket? (Bukti menunjukkan operator ID.)
3. **Berapa nilai praktis `.KomiteLoop`** — 2? 3? 4? Ini kedalaman maksimum penjenjangan.
4. **Apa yang terjadi bila `KomiteCount > 4`?** Tidak ada langkah untuk kasus itu.
5. **Empat harness inbox yang hilang** — masih dipakai user, atau menu mati?
6. **`IsGCNMUser` mengendalikan 11 entri navigasi** dan hilang dari export. Access group mana yang
   dievaluasinya?
7. **Penugasan komite per-orang atau per-peran di sistem baru?** Kolom `STS_ABS` menunjukkan
   masalah nyata yang sudah ditambal di sistem lama.

---

# 7. Seluruh hardcode yang harus dihapus (D-15)

Klaim `docs/BRD.md:453`: **10 email, 4 user ID, 3 ambang, 3 hostname**. Hasil verifikasi:
**lingkup klaim itu hanya dua rule**, bukan seluruh export.

## 7.1 Rekonsiliasi

| Klaim | Lingkup dokumen (`InputRegister_act` + `GetKomiteApproval`) | Seluruh export | Verdict |
|---|---|---|---|
| **10 alamat email** | **11** | **66 unik** | ❌ salah hitung **dan** lingkup terlalu sempit |
| **4 user ID** | **4** — semua terbukti | **24 unik** + belasan tertanam di teks SQL | ⚠️ benar untuk lingkupnya |
| **3 ambang komite** | **3** — di `GetKomiteApproval` | **8 ambang komite unik** + ambang uang non-komite | ⚠️ benar untuk lingkupnya |
| **3 hostname** | **3** | **3** (48 perbandingan) | ✅ **angka benar, daftar salah** |

Penyebab selisih email: `Activity/InputRegister_act-Act.xml:16461` memuat **dua** alamat dalam
satu literal; dokumen menghitungnya satu.

**Untuk `FR-F4` yang berbunyi "menghapus SELURUH hardcode", angka yang relevan adalah 66 / 24 / 8
/ 3.** Angka 10/4/3/3 akan membuat estimasi master data terlalu kecil ±6×.

## 7.2 Email — 66 unik, termasuk akun pribadi di jalur produksi

Alamat disamarkan sesuai aturan bukti 4. Kategori berbukti baris:

| Kategori | Jumlah | Contoh `berkas:baris` |
|---|---|---|
| Email pimpinan (7 alamat dalam satu string) | 7 | `Activity/InputRegister_act-Act.xml:16154` (`local.EmailPimpinan`, step desc `:16098`) |
| Underwriting per Group Panel | 4 | `:16297` (UW FIRE, precondition `GroupPanel=="006"` `:16264`) · `:16461` (UW ANEKA, `"003"` `:16427`) · `:16628` (UW MARINE, `"004"` `:16592`) |
| Komite Simasnet | 2 | `Activity/SetEmailKomiteSimasnet-Act.xml:2244`, `:2850` |
| Salvage/XOL | 1 (di 5 rule) | `Activity/SendKomiteXOLByEmailAndPIC-Act.xml:1230`; `Activity/LetterOfAssignment2_Act-Act.xml:969` |
| Identitas `From`/SMTP | 3 | `Data Transform/SetDataEmail-DT.xml:179`, `:329`, `:436`, `:586` + ±40 activity |
| Reasuransi eksternal (broker) | 1 (di 5 rule) | `Activity/LetterOfAssignment_Act-Act.xml:10011` (BCC); `Activity/CheckerSendEmailApproveReject-Act.xml:2299` |
| **Akun pribadi (Gmail) di jalur produksi** | **≥6** | `Activity/AutoEmailDownloadProposeAdjustment-Act.xml:7872`, `:8031` · `Activity/SetStsSalvagePNC_act-Act.xml:694` · `Activity/LetterOfAssignment2_Act-Act.xml:7725` (`<To>`) · `Activity/SendEmailUnprotectPremi-Act.xml:4823` |
| **Email dipakai sebagai Operator ID** (`.UserTeknis == "…@GMAIL.COM"`) | 5 | `Activity/PNCReportKPI_act-Act.xml:7913`, `:9564` · `Activity/GetNextFUdata_act-Act.xml:3522` · `Activity/EksportDataAllKPIPICKlaim-Act.xml:3790` |
| Email di dalam teks pesan error UI | 5 | `Activity/PrintPDFAcceptanceNote-Act.xml:6653` · `Activity/ServiceGetProductionData-Act.xml:4168`, `:4871` |
| Fallback dev berbasis hostname | 1 | `Activity/SendDLAAutoSaatGeneratedDLA-Act.xml:2706` (`@if(pxReqServer=="pegadev…", <email>, Param.temail)`) |
| PAYDI | 6 | `Activity/NotificationPAYDI-Act.xml:2076`, `:2097` |

Konsumen utama: `Activity/InputRegister_act-Act.xml:18940` →
`<To>local.UWEmail + "," + local.EmailPimpinan</To>`.

> Catatan yang mudah tertukar: `Activity/CNMUpdateMasterRekening_act-Act.xml:640` dan `:674`
> memuat string ber-`@` di elemen `<pyCompany>` — itu **nama perusahaan yang malformed**, bukan
> alamat email.

## 7.3 Blok `// TESTING` — ada, dan secara struktural memang menimpa

Pencarian kata `TESTING` (whole-word) di **seluruh 15 folder rule**: hanya **4 kemunculan, semuanya
di satu rule**.

| `berkas:baris` | Step | Properti yang ditimpa | Menimpa step produksi |
|---|---|---|---|
| `Activity/InputRegister_act-Act.xml:16693` | `TESTING - Pimponan` (typo di export) | `local.EmailPimpinan` | `:16154` (7 email pimpinan) |
| `:16830` | `TESTING - UW FIRE` | `local.UWEmail` | `:16297` — **penimpanya Gmail pribadi** |
| `:16998` | `TESTING - UW ANEKA` | `local.UWEmail` | `:16461` |
| `:17141` | `TESTING - UW MARINE` | `local.UWEmail` | `:16628` — dua alamat dipisah **spasi, bukan koma** (kemungkinan bug) |

Struktur: sub-step 1–4 = produksi (`RH_1.pySteps(53).pySteps(1..4)`), sub-step 5–8 = TESTING.

**Verifikasi orkestrator atas keraguan "apakah blok ini aktif":** sub-agen menggantung klaim BRD
karena keempat step TESTING bertanda `pyStepsBlockName=//`. Teori itu **gugur**:

| Uji | Hasil |
|---|---|
| Sebaran `pyStepsBlockName` di `Activity/` | `//` dipakai **952× di 279 dari 902** activity, berdampingan dengan label bermakna: `ERR` (31), `EXT` (29), `END` (21), `NXT` (18), `SKIP` (10), `LP` (8) |
| Tag kosong (`<pyStepsBlockName/>`) | 14.414× — itulah normanya |
| Elemen penanda aktif/nonaktif (`*Enabled`, `*Disabled`, `pyStepStatus`) | **tidak ada sama sekali** di format export ini |

`//` adalah **label blok** untuk target transisi/jump, bukan sakelar mati. Perbandingan langsung
step produksi vs TESTING untuk UW FIRE:

| | Produksi | TESTING |
|---|---|---|
| Deskripsi | `UW FIRE` (`:16239`) | `TESTING - UW FIRE` (`:16838`) |
| Precondition | `pyWorkPage.Policy.Quotation.GroupPanel=="006"` (`:16264`) | **identik** (`:16859`) |
| Properti di-set | `local.UWEmail` (`:16295`) | **sama** (`:16890`) |

Precondition identik, properti sama, posisi sesudah. **Klaim BRD §11.6 #3 dan D-15 didukung
bukti**: saat Group Panel = `006`, nilai TESTING menimpa email UW produksi.

Yang tetap tidak dapat dipastikan hanyalah apakah Pega Designer menandainya *commented out* lewat
sesuatu yang tidak ikut ter-export. Export **tidak memberi satu pun bukti penonaktifan**.

Metadata pembuat blok: `pxCreateOperator=YENIHULU2`, `pxCreateSystemID=pegadev`,
`pxCreateDateTime=20190226T065106.871 GMT` (`:16687-16691`) — **dibuat 2019 di server dev**.

> Koreksi terhadap dokumen: `docs/STEERING.md:354` menyebut *"step 103 UW FIRE / step 107
> TESTING"*, dan D-15 menyebut *"step 106–109"*. Keduanya **tidak cocok dengan export** — ini
> sub-step 2 dan 6 di dalam **step 53** (`RH_1.pySteps(53).pySteps(2)` di `:16236`;
> `…pySteps(6)` di `:16834`). Kemungkinan nomor berasal dari tampilan editor Pega yang meratakan
> sub-step.

## 7.4 User ID — 4 di lingkup dokumen, 24 di seluruh export

Empat yang disebut dokumen, semuanya terbukti:

| User ID | `berkas:baris` | Untuk apa |
|---|---|---|
| `MORASOTARDODOTARIGAN` | `Activity/GetKomiteApproval-Act.xml:1122`, `:1455` (precondition); `:926`, `:1021`, `:1400` (di-set); `When/IsManagerPNC_CLOSE-When.xml:840,842,948`; `Section/PNCAdminShow_sec-Section.xml:8837` | penentu approver komite PA + hak Manager/Close. Memo rule `Activity/GetKomiteApproval-Act.xml:14`: *"ganti balik PA ke MORASOTARDODOTARIGAN"* |
| `ELLENSUPRIYATI` | `Activity/GetKomiteApproval-Act.xml:639`; `Activity/SetEmailKomite-Act.xml:1514,1790,2082`; `Activity/SetDataKomiteNonMBU_Act-Act.xml:777,904,2108`; `Activity/PNCReportKPI_act-Act.xml:9720` | bypass/penentu limit komite + filter KPI |
| `IRMANOPITAPURBA_1` | `Activity/GetKomiteApproval-Act.xml:873`; `When/IsAdministrators-When.xml:766,769,860`; **di dalam teks SQL**: `RDB List/GetDataKPIAdminPA-SQL.xml:100,118,130,148,167`; `RDB List/BrowseDataKPIAdmin_PA-SQL.xml:37` | hak Administrator + filter laporan admin |
| `RATNAGUSNITASARI` | `When/IsManagerPNC-When.xml:600,603,655`; `When/IsManagerPNC_CLOSE-When.xml:728,732,781` | hak Manager PNC |

Populasi lengkap **24 unik**. Selain empat di atas, yang paling sering:
`INDRAGUNAWAN` (24 kemunculan, `Activity/PNCReportKPI_act-Act.xml:10880`) ·
`YOHANESRAYMONDADIKARTA` (3, `Activity/SetEmailKomite-Act.xml:2276`) ·
`DANIELLISWANDI` (4) · `NOVERHALOMOAN` (4) · `TONY` (3) · `BAMBANGSETIADJIGUNAWAN` (3) ·
`ANDREWHANDOKO` (3) · plus 5 ID berbentuk alamat Gmail huruf besar.

`Param.AssignTo` diisi literal user ID: **dicari, tidak ditemukan**.

> ⚠️ Nama-nama ini sudah muncul apa adanya di dokumen yang disetujui (D-15,
> `20-DETAIL-KOMITE-DBLINK.md:1.5.4`). Apakah boleh tetap tertulis lengkap di ADR dan tiket, atau
> harus disamarkan, **menunggu keputusan work owner** (Fase 2 Lapis 8).

## 7.5 Ambang — 8 ambang komite, bukan 3

| Nilai | `berkas:baris` | Dibandingkan terhadap | Lini bisnis / pemicu |
|---|---|---|---|
| `3500` **atau** `50000000` | `Activity/GetKomiteApproval-Act.xml:335` | di-set ke `tempAdj.ConvertAdjustmentValue` | dipilih oleh **hostname** — entitas Timor-Leste → 3500 |
| `30000000` | `Activity/GetKomiteApproval-Act.xml:488`, kondisi `:467` | idem | `OperatorID.pyPosition=="PA"\|\|"TRAVEL"` |
| `3500` | `Activity/SetEmailKomite-Act.xml:2082`, `:2583` | `<=3500` | entitas USD (`LSC_ID="SMI"`) |
| `7000` | `Activity/SetEmailKomite-Act.xml:2922`, `:3506` | `<=7000` / `>7000` | idem, level berikutnya |
| `3000000` | `Activity/SetEmailKomiteSimasnet-Act.xml:2775` | `<=3000000` | Simasnet |
| `5000000` | `Activity/SetListComiteeClaimPerObjAdj-Act.xml:27787` | `>5000000` | per objek-adjustment |
| `20000000` | `Activity/SetEmailKomiteSimasnet-Act.xml:2017,2194,2491,3872` | `<=`/`>` | Simasnet |
| `50000000` | `Activity/SetEmailKomite-Act.xml:1445,1790,2276`; `Activity/SetEmailKomiteAdjuster-Act.xml:811,968,1126,1363,1517`; `Activity/SetEmailKomiteSalvage-Act.xml:888,966,1145` | `<=`/`>=`/`>` — **tiga operator berbeda** | Indonesia, level 1 |
| `100000000` | `Activity/SetEmailKomite-Act.xml:2796,3185`; `Activity/SetEmailKomiteAdjuster-Act.xml:968,1517,1698`; `Activity/SetEmailKomiteSalvage-Act.xml:888,1443,1573` | `<=`/`>` | Indonesia, level 2 |

Ambang uang non-komite yang juga hardcode: `1000000000`
(`Activity/InputRegister_act-Act.xml:19110` — gerbang Notice of Large Losses; juga
`Activity/NotificationUnpaidPremi-Act.xml:1576`) · `50000000000`
(`Activity/CheckEstimateValue-Act.xml:9617,9831`) · `35000`/`35001`
(`Activity/SetListComiteeClaimPerObjAdj-Act.xml:12354`) · `17500`/`250000000`
(`Activity/SetListComiteeClaimAI-Act.xml:11602,11742`) · `5000000000`
(`RDB List/BrowseClaimStudy-SQL.xml:93`).

Semua ambang ditulis sebagai **integer polos** — tidak ada varian bertitik atau berdesimal
(dicari, nihil). Tidak ada satu pun ambang komite yang berasal dari master data di `DataPage/`
atau `Data Transform/` (dicari, nihil) — **semuanya literal di dalam activity**.

## 7.6 Hostname dan kredensial

**Hostname penentu perilaku: 3 unik, 48 perbandingan, semuanya terhadap
`pxRequestor.pxReqServer`.** Nilai tidak direproduksi sesuai aturan bukti 4.

| Host | Perbandingan | Perilaku yang dipicu | Contoh `berkas:baris` |
|---|---|---|---|
| host **dev** | 34 | melewati/mengganti step; mengganti penerima email ke email dev; menentukan `IsServiceCenterPNC` | `When/IsServiceCenterPNC-When.xml:463,466,530`; `Activity/SendDLAAutoSaatGeneratedDLA-Act.xml:2706`; `Activity/CheckViewPolis_act-Act.xml:942,1137,2060,4861` |
| host **entitas Timor-Leste** | 1 | mengubah ambang komite dari 50.000.000 → 3.500 | `Activity/GetKomiteApproval-Act.xml:335` |
| host **entitas Insurtech** | 13 | mengganti kode entitas (`SMI` vs `ASM`), status investigator, filter view klaim | `When/IsServiceCenterPNC-When.xml:581,583,641`; `Activity/Insert_salvageToGAByService-Act.xml:2822,2826,2838`; `Activity/SetStatusInvestigator_Act-Act.xml:1621` |

> **Koreksi daftar dokumen.** `docs/STEERING.md` menyebut host produksi utama sebagai salah satu
> dari tiga. Itu **tidak akurat** — host itu hanya muncul sebagai **konstanta URL**
> (`Activity/ConnectRestPNC_act-Act.xml:175`, `Activity/GetLinkAppClaim-Act.xml:765`,
> `Activity/HitupdateDataRekeningToKasir-Act.xml:2151`), tidak pernah dibandingkan. Hostname
> ketiga yang benar-benar menentukan perilaku adalah host entitas Insurtech. Angka 3 kebetulan
> tetap benar; daftarnya tidak.

Dicari dan **tidak ditemukan**: `getHostName` · `InetAddress` · `pyHostName` ·
`pxProcess.pxSystemName` sebagai kondisi.

**Dua When rule penentu perilaku berbasis server hilang dari export** dan **tidak tercatat di
`19-GAP`**: `IsServerSyariah` (dirujuk `Data Transform/SetDataEmail-DT.xml:147`, `:411`;
`Activity/SpreadingDataProtection-Act.xml:1294`, `:4153`) dan `IsDevelopmentServer` (dirujuk
`When/IsDevToLive-When.xml:243`). **Jumlah hostname sebenarnya bisa lebih dari 3.**

### Kredensial di dalam export — perlu tindakan di luar lingkup dokumen ini

Nilai tidak direproduksi. Hanya lokasi dan jenis:

| Jenis | Lokasi | Status |
|---|---|---|
| **Password SMTP** | **31 lokasi** di elemen `<Password>` pada step `Email-Send`, **3 nilai unik**. Contoh: `Activity/KomitePost_Adjustment-Act.xml:13867`, `:20895` · `Activity/LetterOfAssignment_Act-Act.xml:10009` · `Activity/SendWorkMailReport_Act-Act.xml:1567` · `Activity/UpdateDetailPLA2-Act.xml:10418`, `:10676`. Ditambah `Data Transform/SetDataEmail-DT.xml:201`, `:459` | **plaintext** |
| **Kredensial OAuth** | `Connect REST/GetTokenClaimBRISurf-ConnectREST.xml:117` (`client_id`, 32 karakter) dan `:127` (`client_secret`, 16 karakter), keduanya `pyMapFrom=Constant` | **plaintext** — terverifikasi ulang oleh orkestrator |
| Password terlindungi (pembanding) | `Activity/ASMSendsEmailAttachments-Act.xml:808`, `:933` — via `@(Pega-IntSvcs:ServicesUtilities).getStringIndirect(.pyPassword)` | terlindungi |
| Host SMTP + port | `Data Transform/SetDataEmail-DT.xml:238,240,261,273` | plaintext |

**Export 297 MiB ini harus diperlakukan sebagai berisi kredensial aktif** sampai dikonfirmasi
sebaliknya. Ini memengaruhi cara repo disimpan dan dibagikan, bukan hanya isi ADR.

## 7.7 Pertanyaan yang hanya bisa dijawab work owner

1. **Definisi "hardcode" yang dipakai `FR-F4`** — sengaja dibatasi ke dua rule, atau dimaksudkan
   total export? Bila total, angkanya harus direvisi ke **66 / 24 / 8 / 3**.
2. **Rotasi 3 password SMTP** (31 lokasi) dan kredensial OAuth. Masih berlaku?
3. **Akun pribadi di jalur produksi** — penerima notifikasi personal (Gmail) masih bekerja di
   perusahaan? Mailbox fungsional mana yang menggantikannya?
4. **Ambang mana yang masih berlaku?** `3500`, `7000`, `17500`, `20000000`, `100000000`,
   `250000000`, `5000000`, `3000000` — dan kurs 14.285 yang dibekukan lewat pasangan angka
   (`20-DETAIL:1.5.1`) sudah tidak mencerminkan kurs sekarang.
5. **Kondisi berbasis nama orang** (`ELLENSUPRIYATI`, `IRMANOPITAPURBA_1`,
   `MORASOTARDODOTARIGAN`, `INDRAGUNAWAN`, `YOHANESRAYMONDADIKARTA`) — dibawa sebagai **peran**,
   atau dibuang? Dan mana yang masih aturan bisnis sah versus sisa tambalan lama?
6. **Isi `IsServerSyariah` dan `IsDevelopmentServer`** — keduanya menentukan perilaku tapi tidak
   diekspor.
7. **Boleh menulis nama user ID dan alamat email lengkap di ADR/tiket, atau harus disamarkan?**
   (Lapis 8)

---

# 8. Business rule BRD Bab 11 — angka yang benar-benar ada di rule

Rule inti: `Activity/InputRegister_act-Act.xml` (1,1 MiB) — seluruh validasi tanggal (step 18–28),
duplikasi (step 32–33), spreading, Fac Offer, SLIK, Deklarasi.

**Konvensi precondition Pega yang dipakai** (disimpulkan dari pola `IsPA`/`IsTravel`):
`pyStepsPreCondParamsWhenTrue/False` = `2` → lanjut, `3` → skip step. Nilai `4`
**TIDAK DAPAT DIPASTIKAN DARI EXPORT** (muncul di `:8720` dan `:11353`).

Definisi lini bisnis — **ini bahan acceptance criteria langsung**:

| When | Definisi | `berkas:baris` |
|---|---|---|
| `IsBonding` | `.Policy.Quotation.BusinessType == "Bonding"` | `When/IsBonding-When.xml:142` |
| `IsTravel` | `GroupPanel == "005"` | `When/IsTravel-When.xml:128` |
| `IsPA` | `GroupPanel == "002"` | `When/IsPA-When.xml:325` |
| `IsTravelPA` | `"005"` OR `"002"` | `When/IsTravelPA-When.xml:251,489` |
| `IsAneka` | `BusinessCode == "10140"` | `When/IsAneka-When.xml:118` |

## 8.1 Aturan tanggal — dan bukti konkret R-12

Fungsi yang dipakai: `@addToDate(<datetime>,"<hari>","","","")` + `@CompareDates(a,b)`.
`@DateTimeDifference`/`@daysBetween` **tidak dipakai** untuk validasi ini.

Properti: `.ClaimData.DateOfLoss`, `.ClaimData.ReportDate`, `.ClaimData.DateReceived`,
`.Policy.StartDateTime`, `.Policy.EndDateTime`.

| Aturan | Nilai | Properti | Lini bisnis | `berkas:baris` | Cocok BRD? |
|---|---|---|---|---|---|
| DOL dalam periode polis | — | DOL, Start, End | **semua, tanpa pengecualian** | `Activity/InputRegister_act-Act.xml:5788` (step 21, msg `errorDOLNotInRange` `:5733`) | ⚠️ lihat §8.1.2 |
| DOL s/d 30 hari setelah polis berakhir | **`"30"`** | idem | `IsBonding` (`:6497`) | `:6480` (step 25) | ⚠️ literal ada, **efek tidak** |
| Tanggal cetak s/d 90 hari setelah berakhir | **`"90"`** | `TempDate.TanggalCetakDLA` | `IsTravelPA` (`:5077`) | `:5092` (step 18) | ✅ nilai; ⚠️ pesan salah sasaran |
| Tanggal Lapor ≤ DOL + 7 hari | **`"7"`** | `.ReportDate` | **`IsPA` dikecualikan** (`:5499`) | `:5516` (step 20, msg `errorReportDate7days`) | ✅ |
| Tanggal Terima ≤ DOL + 90 hari | **`"90"`** | `.DateReceived` | `IsTravel` saja (`:5361`) | `:5396` (step 19, msg `errReceivedDate90`) | ✅ |
| DOL ≤ Tanggal Lapor | — | — | semua | cond `:5954`; **escape kesetaraan** `:5989` (`local.DOL==local.ReportDate`); desc `:5831` *"jika tgl sama tidak masalah"* | ✅ |
| Tanggal Lapor ≤ Tanggal Terima | — | — | semua | `:6107` (step 23) — **tanpa escape kesetaraan** | ⚠️ asimetris |
| DOL ≤ hari ini | — | `TempDate.TanggalCetakDLA` | semua | `:6231` (step 24) | ✅ |
| Tanggal Lapor ≤ hari ini | — | idem | semua | `:6580` (step 26) | ✅ |
| Tanggal Terima ≤ hari ini | — | idem | semua | `:6829` (step 27) | ✅ |

### 8.1.1 Penyesuaian 7 jam diterapkan tidak konsisten **di dalam satu kondisi** (T-4)

Kondisi step 21 (`:5788`), diverifikasi ulang oleh orkestrator:

```
@CompareDates(.ClaimData.DateOfLoss, .Policy.EndDateTime)      ← DOL mentah
|| @CompareDates(.Policy.StartDateTime, TempDate.DateOfLoss)   ← DOL + 7 jam
```

dengan `:4805`:
`TempDate.DateOfLoss = @DateTime.addCalendar(.ClaimData.DateOfLoss, 0,0,0,0,7,0,0)`

**Ujung atas periode polis diuji dengan DOL mentah; ujung bawah dengan DOL yang sudah digeser 7
jam.** Untuk DOL di sekitar tengah malam pada tanggal polis mulai, pergeseran itu memindahkan
tanggalnya dan mengubah hasil validasi.

Pembanding "hari ini" juga digeser: `TempDate.TanggalCetakDLA = @CurrentDateTime()` (`:4919`) lalu
`+7 jam` (`:4936`).

Inilah "7-jam-manual" yang F-5/`FR-F5` ada untuk menghapus, dan **R-12 memprediksinya**. Karena
F-5 termasuk empat perbaikan yang sudah diputuskan eksplisit
(`docs/requirement-summary.md:26`), memperbaikinya sah — **tetapi konsekuensinya harus disetujui
lebih dulu**: uji kesetaraan gerbang 1 akan menunjukkan selisih pada klaim dengan DOL di batas
periode polis.

### 8.1.2 Toleransi 30 hari Bonding tidak dapat berlaku

Step 21 (`:5788`) **tidak punya guard `bukan IsBonding`** — satu-satunya precondition-nya adalah
ekspresi tanggal. Step 25 (toleransi Bonding, `:6480`) adalah **sibling top-level**, bukan
pengganti. Jadi DOL = polis berakhir + 15 hari pada polis Bonding **tetap ditolak step 21**;
step 25 hanya mencegah error **kedua**.

BRD `:610` menulis toleransi ini sebagai aturan yang berlaku. Dua kemungkinan: BRD salah, atau ada
rule penonaktif yang tidak ikut export. **Dapat diselesaikan dengan satu uji** di Pega berjalan.

### 8.1.3 Semantik `@CompareDates` — koreksi orkestrator

Sub-agen menyimpulkan `CompareDates` adalah fungsi custom yang hilang, sehingga 8 acceptance
criteria batas tanggal tidak dapat ditentukan. **Koreksi:** sensus pemanggilan library function
menunjukkan `@DateTime.CompareDates(` dipakai **20×**, di library `DateTime` yang sama dengan
`@DateTime.addCalendar(` (577×), `@DateTime.FormatDateTime(` (108×), dan
`@DateTime.DateTimeDifference(` (227×) — semuanya **bawaan Pega-RULES**.

Jadi semantiknya (`>` versus `>=`) **dapat dijawab dari dokumentasi Pega atau satu uji**, bukan
pertanyaan ke work owner. Delapan AC batas tanggal pindah dari "tidak dapat dipastikan" ke "dapat
dipastikan tanpa work owner".

### 8.1.4 Jalur validasi kedua yang lebih longgar — belum pernah tercatat

`Activity/AdoptJSONInsertJSON-Act.xml:1377`, `:1384`, `:1392` dan
`Activity/CallActivityInputRegister-Act.xml:7114` memvalidasi DOL dengan ekspresi berbeda
(`@CompareDates(.ClaimData.DateOfLoss,.Policy.StartDateTime)&&@CompareDates(.Policy.EndDateTime,.ClaimData.DateOfLoss)`
— **arah argumen terbalik**) dan **tanpa aturan 7/30/90 sama sekali**.

Klaim yang masuk lewat jalur API/JSON melewati gerbang validasi yang jauh lebih longgar daripada
klaim yang masuk lewat form. Tidak ada di BRD maupun Steering.

## 8.2 Aturan duplikasi — kunci berbeda dari yang dijanjikan BRD

| Varian | Kolom `WHERE` persis | `berkas:baris` |
|---|---|---|
| **non-PA** | `a.policyno` · `b.objectid` · fragmen lokasi (dinamis) · **`TRUNC(dateofloss_1) = to_date(...)`** · `a.pystatuswork != 'Resolved-Rejected'` · `upper(a.closeclaimnote_1) NOT LIKE '%CWP%'` · `(exgratia_1 != '1' OR exgratia_1 IS NULL)` · `a.pzinskey != <current>` | `RDB List/BrowseClaimPNCValidation-SQL.xml:30-42`; dipanggil `Activity/InputRegister_act-Act.xml:7993` |
| **PA** | `a.policyno` · `c.objectid` · `EXISTS(… t_claim_objectcoverage b WHERE coverageid='10009' AND b.claimid=a.pzinskey)` · `a.pystatuswork != 'Resolved-Rejected'` · `a.pzinskey != <current>` — **tanpa kolom lokasi, tanpa kolom cause of loss** | `RDB List/BrowseClaimPNCValidationPA-SQL.xml:4-10`; pemicu `Activity/InputRegister_act-Act.xml:8719` |

Selisih terhadap BRD §11.2:

| BRD | Bukti | Selisih |
|---|---|---|
| "Nomor Polis + Objek + Lokasi" | juga memfilter **`TRUNC(dateofloss_1)`** | **DOL adalah bagian kunci** → dua klaim polis+objek+lokasi sama dengan DOL berbeda dianggap **sah**. Jauh lebih permisif dari yang dijanjikan |
| lokasi berlaku semua lini | fragmen lokasi hanya di-set bila **bukan `IsPA`** (`:7802`) dan **bukan `IsTravel`** (`:7825`) | PA & Travel **tidak** diperiksa lokasi |
| PA: "+ Penyebab Kerugian `12002` + Lokasi" | `12002` hanya **precondition step** (`:8719`); SQL PA memfilter `coverageid='10009'` | kunci PA berbeda; kode `10009` tidak ada di BRD |

**Normalisasi dan risiko injeksi.** Fragmen lokasi dibangun sebagai **konkatenasi string SQL** di
`Activity/InputRegister_act-Act.xml:7750`:

```
"AND replace(upper(location_1),' ','') = replace(upper('" + pyWorkPage.ClaimData.Location + "'),' ','')"
```

Jadi lokasi **case-insensitive dan spasi dibuang**; `policyno`, `objectid`, `dateofloss`
**tanpa TRIM/UPPER** (sensitif spasi dan kapital). Dan ini **perangkaian SQL dari nilai
pengguna** — persis yang dilarang `BRD §21.2` kriteria #11.

**Klaim yang dibatalkan/ditolak:** non-PA mengecualikan `Resolved-Rejected`, catatan `CWP`, dan
klaim ex-gratia; status lain (termasuk `Resolved-Completed`) **ikut dihitung**. PA hanya
mengecualikan `Resolved-Rejected` — **CWP dan ex-gratia ikut dihitung** (asimetri).

**Pesan penolakan menyertakan nomor klaim** ✅ — `:8096` (non-PA,
`pyReportContentPage2.pxResults(1).pyID`) dan `:8956` (PA, `TempPA.pxResults(1).SearchName`).

## 8.3 Aturan reasuransi — toleransi 99,99% adalah cacat berjalan (T-3)

**Terverifikasi ulang oleh orkestrator.** Kondisi persis di
`Activity/InputRegister_act-Act.xml:13183`:

```
@contains(local.totalspreading,100.0) || local.totalspreading==100 || @contains(local.totalspreading,99.99)
```

`@contains` adalah **pencocokan substring**, bukan perbandingan angka. Dan
`grep 'round\|Round\|setScale\|FormatNumber'` di seluruh berkas 1,1 MiB = **0**.

| Total spreading | Hasil | Sebabnya |
|---|---|---|
| `199.99` | ✅ **lolos** | memuat substring `99.99` |
| `1100.0` | ✅ **lolos** | memuat substring `100.0` |
| `99.995` | ✅ **lolos** | memuat substring `99.99` |
| `99.98` | ❌ ditolak | benar |
| `100.01` | ❌ ditolak | benar |

BRD `:631` (*"Total spreading wajib 100% (toleransi 99,99%)"*) mendeskripsikan **maksud, bukan
perilaku**. Menulis AC "total 100% ± 0,01" adalah **memperbaiki**, bukan menyetarakan —
dan gerbang 1 akan menampilkan selisih pada data historis. Harus disetujui sebagai perbaikan
eksplisit lebih dulu.

Aturan reasuransi lainnya:

| Aturan | Nilai di rule | `berkas:baris` | Catatan |
|---|---|---|---|
| Akumulasi share | `local.totalspreading + .SharePercentage` | `:12822`, init `0` di `:2330`, `:10860` | `local.totalspreading` dideklarasi `Decimal` (`:26677-26678`) |
| Spreading bertanda hapus dibuang | `.FlagDelete=="1"` → `Page-Remove` | desc `:11297`, `Page-Remove` `:11306`, kondisi `:11356`; juga skip akumulasi `:12881` | ✅ dua lapis |
| Fac Out → Fac Offer wajib | Fac Out = `.TreatyType=="10015"`; `@SizeOfPropertyList(Policy.FacOfferList)==0` | `:11465`, `:11497`; pesan `:11524` | string `"Fac Out"`/`"FacOut"` **tidak ada** — hanya kode |
| Group Panel `003` → Fac Offer wajib Object Name | `GroupPanel=="003"` AND `TreatyType=="10015"` AND `FacOfferList(1).AnekaList(1).ObjectName==""` | `:11669`, `:11692`, `:11718`, `:11738` | ⚠️ **hanya baris (1)(1)** — Object Name kosong pada baris ke-2+ **lolos** |
| Ex-Gratia → `OR` jadi `ORS` | kode **`"10001"` → `"10007"`** | desc `:12916` *"set OR-> ORS jika ex gratia"*, set `:12926`, properti `:12928`, kondisi `:13020`, `:13044` | ⚠️ kode numerik, bukan string |

Pemetaan `10001=OR`, `10007=ORS`, `10015=Fac Out`: **TIDAK DAPAT DIPASTIKAN DARI EXPORT**.
Klasifikasi kategori (2) → DBA / tim bisnis.

## 8.4 Aturan nilai dan notifikasi

| Aturan | Nilai | Operator | `berkas:baris` | Cocok BRD? |
|---|---|---|---|---|
| Konversi ke IDR pakai kurs standar | RDB `CurrencyStandard` | `KursValue * EstimationValue` | RequestType `Activity/SetConvertValueKurs_Estimation-Act.xml:4377`; set kurs `:4594`; SQL `RDB List/CurrencyStandard-SQL.xml`; akumulasi `Activity/CheckEstimateValue-Act.xml:6032` | ✅ |
| Peringatan estimasi > 1 M (di form) | `1000000000` | **`>`** strict | `Activity/CheckEstimateValue-Act.xml:10268`, pesan `:10176` | ✅ tepat Rp 1.000.000.000 **tidak** memicu |
| Notice of Large Losses (email) | `1000000000` | **`>`** strict | kondisi `Activity/InputRegister_act-Act.xml:19110`; komplemen `<1000000000` `:19121`; desc `:18882`; `CorrName=NoticeOfLargeLosses_Email` `:18934` | ✅ |
| Ambang komite umum | `50000000` **atau `3500`** | `@If(hostname…)` | `Activity/GetKomiteApproval-Act.xml:335` | ⚠️ ditentukan hostname |
| Ambang komite PA/Travel | `30000000` | assign | `Activity/GetKomiteApproval-Act.xml:488`, kondisi `:467` | ⚠️ dipicu **jabatan operator** `OperatorID.pyPosition`, bukan `IsPA`/`IsTravel` yang berbasis `GroupPanel`. Klaim PA yang diproses operator berposisi lain memakai ambang Rp 50 juta |
| Premi belum lunas → notifikasi | `paymenttype != '3'`, `!= '4'` | `!=` | `Activity/NotificationUnpaidPremi-Act.xml:1462`, `:1484`, desc `:1537` | ✅ |

**Operator pada ambang Rp 50.000.000 tidak konsisten antar rule** — semuanya terhadap
`tempAdj.ConvertAdjustmentValue`:

| Operator | `berkas:baris` |
|---|---|
| `<=50000000` | `Activity/SetEmailKomite-Act.xml:1445` |
| `>=50000000 && <=100000000` | `Activity/SetEmailKomiteAdjuster-Act.xml:968` |
| `>50000000 && <=100000000 && OperatorID.pyPosition=="NONMBU" && ClaimData.UserTeknis!="INDRAGUNAWAN"` | `Activity/SetEmailKomiteSalvage-Act.xml:888` |

**Tepat Rp 50.000.000 masuk dua jenjang sekaligus di jalur Adjuster.** Ini persis pertanyaan
"Rp 8.800.000 boleh, Rp 8.800.001 memicu jenjang berikutnya" yang diminta brief — dan jawabannya
saat ini **tidak konsisten**.

## 8.5 Aturan kelengkapan

| Aturan | Bukti | `berkas:baris` | Cocok BRD? |
|---|---|---|---|
| Penyebab Kerugian wajib | `.CauseOfLossID==""`; **`IsTravel` dikecualikan** (`:11231`) | `Activity/InputRegister_act-Act.xml:11246`, msg `:11175` teks `:10836` | ✅ |
| Nomor SLIK wajib | `.NoSLIK==""`; gate `TempGetApp.LSC_ID=="PEGAKREDIT"` (`:1342`) **DAN** When `IsAsuransiKredit` (`:1371`) | kondisi `:1388`, pesan `:1297` | ⚠️ **`When/IsAsuransiKredit-When.xml` tidak ada di export** |
| Nilai klaim ≤ TSI | `Local.sumTSIIDR < Local.totalEstimasi` → error (strict `<`, jadi estimasi **=** TSI diizinkan) | `Activity/CheckRequest_fromSurvey-Act.xml:3896`, pesan `:3845`; akumulasi `:2606`, `:3719` | ✅ sesuai BRD |
| **PA boleh melebihi TSI** | precondition `IsPA` (`:10474`), desc *"is pa coment bisa lebih dari limit TSI"* | `Activity/CheckEstimateValue-Act.xml:10394` | ❌ **tidak ada di BRD** |
| Objek tanpa Coverage dibuang | `@SizeOfPropertyList(.ObjectCoverageList)>0` → skip; `Page-Remove` bila 0 | `Activity/InputRegister_act-Act.xml:1586`, `Page-Remove` `:1558`, desc `:1565` | ✅ |
| Hubungan "lain-lain" → keterangan wajib | kode **`7`**: `.InsuredRelationship==7` AND NOT `@PropertyHasValue(.InsuredRelationshipOthers)` | `:7011`, `:7045`, msg `:6950` | ✅ (arti kode `7` **TIDAK DAPAT DIPASTIKAN DARI EXPORT**) |
| Polis Deklarasi tidak dapat diklaim | `Policy.IsDeclaration=="true"` (**string**, bukan boolean); **`IsAneka` dikecualikan** (`:3105`) | kondisi `:3077`, msg `:3007` teks `:2685` | ✅ |
| Validasi sisa TSI | **rule tidak ada** — lihat §8.5.1 | — | ✅ BRD benar |

> **Kode mati yang mendeskripsikan aturan lebih ketat.** `local.errorEstimation` = *"Nilai Estimasi
> Klaim harus **lebih kecil** dari Nilai TSI."* (`Activity/InputRegister_act-Act.xml:2079`,
> dideklarasi `:26370`) **tidak pernah dipakai** sebagai `<Message>` di activity itu. Pesan
> mendeskripsikan `<`, kode aktif memakai `≤`. Mana yang benar secara bisnis?

### 8.5.1 `ValidasiSisaTSI` — dikonfirmasi tidak ada, tapi jejaknya informatif

| Aspek | Bukti |
|---|---|
| Rule-nya | `find -iname "*SisaTSI*"` → **nihil** |
| Satu-satunya penyebutan | `Activity/SetNilaiResikoSendiri-Act.xml:9678` (`<pyStepsActivityName>Call ValidasiSisaTSI`), `:9682` |
| Posisi step | `rowdata REPEATINGINDEX="54"`, `pyStepPageReference = RH_1.pySteps(54)` (`:9689`) |
| **Deskripsi step** | `:9679` — *"Call Validasi SISA TSI jika Pernah Melakukan KLAIM"* |
| **Precondition pemanggil** | **`IsPA`** (`:9760`) |
| **Parameter yang dikirim** | `ObjectID` = `local.idxOBJ`, `CoverageID` = `local.idxCVG` — keduanya `STRING`, `pyParametersParamInOut=true` (in/out) (`:9724-9727`) |
| Kelas step | `ASM-FW-GCNMFW-Data-Adjustment` (`:9697`) |

Jejak ini cukup untuk merumuskan hipotesis yang dapat dikonfirmasi: **mengurangi TSI dengan
akumulasi klaim terbayar per objek per coverage, hanya untuk lini PA**. Tapi itu hipotesis, bukan
fakta — klasifikasi kategori (2) → Tim Pega + tim bisnis.

## 8.6 Pengumpulan kesalahan validasi — klaim BRD terbukti benar

BRD §11.6 #2 menyatakan seluruh kesalahan validasi dikumpulkan dan dikembalikan bersamaan, dan
menyebutnya **kesetaraan perilaku** (artinya sistem lama sudah begitu). **Terbukti:**

| Bukti | `berkas:baris` |
|---|---|
| `Activity/InputRegister_act-Act.xml` **tidak punya** `Exit-Activity` maupun `Activity-End` | grep count = **0** |
| 11× `Property-Set-Messages` + 21 kemunculan `Page-Set-Messages`, semuanya step terpisah dengan precondition sendiri | `:5050,5228,5440,5644,6031,6217,6380,6572,6735,6908,10956,11115,11814,11957,13145,8209,9067` |
| Precondition gagal → `WhenFalse=3`/`WhenTrue=3` = **skip step**, bukan exit | `:5397,5519,5792,6110,11248,13192` |
| Validator lain juga tanpa early-exit | `Activity/CheckEstimateValue-Act.xml`, `Activity/ValidTotalEstimation-Act.xml`, `Activity/ValidationInitial_act-Act.xml` — masing-masing 0 |

**AC yang dapat langsung dipakai:** sistem baru harus mengembalikan **semua** error tanggal +
duplikasi + spreading + kelengkapan dalam satu respons HTTP `422`, masing-masing terikat ke field
(`Primary.ClaimData.DateOfLoss`, `…ReportDate`, `…DateReceived`,
`…InsuredRelationshipOthers`).

## 8.7 Pertanyaan yang hanya bisa dijawab work owner

1. **Apakah polis Bonding dengan DOL di luar periode memang ditolak hari ini?** (§8.1.2) Bila ya,
   BRD `:610` salah dan harus dicabut.
2. **`@contains(totalspreading, 99.99)`** — toleransi yang dimaksud sebenarnya numerik
   `>= 99.99 && <= 100.00`? Perilaku sekarang meloloskan `199.99` dan `1100.0`. Bug, atau sudah
   diketahui dan diterima?
3. **Berapa desimal share spreading yang valid secara bisnis?** Tidak ada pembulatan di rule;
   jawaban ini wajib untuk menulis toleransi numerik yang benar.
4. **Apakah DOL benar-benar bagian kunci duplikasi?** (§8.2) BRD tidak menyebutnya.
5. **Mengapa lokasi tidak diperiksa untuk PA dan Travel?** Sengaja atau bug?
6. **Ex-gratia dan CWP diabaikan sebagai duplikat pada non-PA tetapi dihitung pada PA** — asimetri
   ini benar?
7. **Ambang komite Rp 50.000.000: `>`, `>=`, atau `<=`?** Tiga rule, tiga operator, nilai sama.
8. **Ambang PA/Travel Rp 30.000.000 dipicu jabatan operator atau lini bisnis klaim?**
9. **Group Panel `003`: Object Name harus diperiksa untuk SEMUA baris Fac Offer**, atau memang
   hanya baris pertama?
10. **Aturan `ValidasiSisaTSI`** — hipotesis §8.5.1 benar?
11. **Mengapa PA dikecualikan dari batas TSI?** Tidak ada di BRD.
12. **`local.errorEstimation` (`<`) versus kode aktif (`≤`)** — mana yang benar?
13. **Jalur API/JSON dibebaskan dari aturan 7/30/90?** (§8.1.4)
14. **Normalisasi lokasi (`upper` + hapus spasi)** aturan bisnis yang harus dipertahankan, atau
    artefak workaround data?
15. **Pemetaan kode → label**: `TreatyType` `10001`/`10007`/`10015` · `CoverageID` `10009` ·
    `CauseOfLossID` `12002` · `GroupPanel` `002`/`003`/`005` · `BusinessCode` `10140` ·
    `InsuredRelationship` `7` · `paymenttype` `3`/`4`.

---

# 9. Audit ketersediaan — gap ±6× lebih besar dari yang terdokumentasi

## 9.1 Selisih terhadap `19-GAP-EXPORT-DETAIL.md`

`19-GAP` mengaudit **tiga** kategori: R-01 (objek database), R-04 (router), R-07 (activity).
**Tujuh tipe rule tidak diaudit sama sekali.**

| Kategori | Hilang | Buatan sendiri | Tercatat `19-GAP`? |
|---|---|---|---|
| Activity | 46 | **40** | ✅ (dengan 5 positif palsu — §9.2) |
| Router | 4 | **3** | ✅ terkonfirmasi persis |
| **When rule** | 141 | **116** | ❌ |
| **Flow Action** | 66 (+28 belum pasti) | **63** | ❌ |
| **Section** | 32 | **24** | ❌ |
| **Data Transform** | 18 (batas bawah) | **16** | ❌ |
| **Ticket rule** | 9 | **8** | ❌ |
| **Report Definition** | 9 | **9** | ❌ |
| **Function / library** | 6 fungsi + 2 library | **semuanya** | ❌ |
| Procedure & function lokal (R-01) | 64 | — | ✅ (kategori 3 — Oracle) |
| Function remote (R-03) | 6 | — | ✅ (kategori 3 — DB Link) |
| **Total perlu diminta Tim Pega** | | **±242** vs **43** tercatat | |

**116 When rule buatan sendiri yang hilang adalah yang paling merusak.** Di Pega, When rule
*adalah* aturan bisnis — klasifikasi produk, syariah/konvensional, per bank, PAYDI, EDM. Contoh
berbukti:

| When | Dirujuk dari | Mengendalikan |
|---|---|---|
| `IsAsuransiKredit` | `Activity/CheckOutGoData_Aneka-Act.xml:6312`; `Activity/InputRegister_act-Act.xml:1371` | gerbang validasi SLIK |
| **`NonMBU`** | `Flow/Register_Flow.xml:6856` (`pyTaskWhen`) | **mencabangkan flow utama** |
| `IsGCNMUser` | `Navigation/pyCaseWorkerNavigation-Navigation.xml` (11 entri) | 11 entri menu portal |
| `IsSyariah` | `Activity/InputDtlCoverage_PreActAneka-Act.xml:51119` | percabangan syariah |
| `IsBCA` | `Activity/CalculateKBGCommission2_Act-Act.xml:4277` | perhitungan komisi per bank |
| `IsPAYDI` | `Activity/CheckPaymentData_Act-Act.xml:1890` | jalur PAYDI |
| `IsKomite`, `IsSurvey`, `IsEdmNCB`, `isCovered` | berbagai | peran & percabangan |

Sebaran elemen perujuk: `pyStepsPreCondParamsWhen` 127 · `pyTaskWhen` 17 ·
`pyStepsTransParamsWhen` 9 · `pyWhen` 8 (ada tumpang tindih).

**Section yang hilang berdampak langsung ke UI.** `Section/ViewPolicyDetail-Section.xml` ada,
tetapi tiga sub-section terbesarnya hilang: `ViewCoverage` (`:14727`), `ViewObject` (`:13902`),
`ViewPayment` (`:18379`) — **layar View Polis tidak dapat direkonstruksi dari export**.

**Function library.** `Function/` berisi **1** berkas (`GetPageJSONString-Function.xml`, ruleset
`GISFW`). Sensus pemanggilan library custom — diverifikasi ulang oleh orkestrator:

| Pemanggilan | Jumlah | Status |
|---|---|---|
| `@GCNM.GetPageJSONString(` | **74×** | ❌ library `GCNM` **tidak ada sama sekali** |
| `@ASM.GetPageJSONString(` | 5× | ✅ satu-satunya yang tercakup |
| `@ASM.NumberAddSeparator(` | 4× | ❌ |
| `@ASM.IndexInPageListReturnZero(` | 3× | ❌ |
| `@ASM.ASMGetMetaData(` | 2× | ❌ |
| **`@ASM.HmacSHA256(`** | 1× | ❌ — dipakai untuk **tanda tangan integrasi BRI** di `Activity/SendFeedbackBRISurf-Act.xml:3423`; tanpa source-nya integrasi itu **tidak dapat direplikasi** |
| `@GCNM.GCNMGetMetaData(` | 1× | ❌ |

Library bawaan Pega-RULES yang tidak perlu diminta: `@Math.*` (893 `divide`), `@String.*`
(623 `equals`), `@DateTime.*` (577 `addCalendar`, 227 `DateTimeDifference`, 108 `FormatDateTime`,
78 `CurrentDateTime`, **20 `CompareDates`**), `@Utilities.*` (64 `SizeOfPropertyList`).

## 9.2 Lima positif palsu di R-07 — terverifikasi ulang oleh orkestrator

Kelimanya **ADA** di export, hanya beda kapitalisasi huruf pertama. Dan **kelimanya rule bawaan
Pega**, jadi `19-GAP` salah dua kali per baris:

| Ditulis `19-GAP` | Berkas sebenarnya | Ruleset |
|---|---|---|
| `CreateWorkPage` | `Activity/createWorkPage-Act.xml` | `Pega-ProcessEngine` |
| `SetPageErrors` | `Activity/setPageErrors-Act.xml` | `Pega-ProCom` |
| `addWork` | `Activity/AddWork-Act.xml` | `Pega-ProcessEngine` |
| `GetEmailSenderInfo` | `Activity/getEmailSenderInfo-Act.xml` | `Pega-IntegrationArchitect` |
| `PerformFlowAction` | `Activity/performFlowAction-Act.xml` | `Pega-ProcessEngine` |

Angka R-07 yang benar: **40 activity buatan sendiri** (bukan 45) + **6 bawaan Pega** (bukan 5 —
`pyConvertToJavaMap` terlewat) = **46**, bukan 50.

## 9.3 Integritas inventaris — nama berkas bukan inventaris

Orkestrator memeriksa **seluruh 902 berkas** `Activity/` dalam satu lintasan, membandingkan nama
berkas terhadap `<pyRuleName>` pertama. **7 ketidakcocokan; 2 nyata:**

| Berkas | `pyRuleName` sebenarnya | Jenis |
|---|---|---|
| `Activity/SendEmailNotification-Act.xml` | **`CompressImage_Act`** (class `Link-Attachment`, ruleset `GISFW`) | ⚠️ **isi berbeda** |
| `Activity/NotificationPAYDI-Act.xml` | **`JobSendEmailNotificationPAYDI`** | ⚠️ **isi berbeda** |
| `DeleteFromTabelMst-Act.xml.xml` · `GCNMSetRecommendationLogin_act-Act.xml.xml` · `InputKodePos1_Act.xml` · `SetCoveragePremiumObjectItem_Act.xml` · `pyCaseWorkerNavigation.xml` | cocok | hanya akhiran berkas tidak baku |

Yang pertama penting: `SendEmailNotification` adalah activity hilang **paling banyak dipanggil
(15×)** dan pemblokir S-3 Notifikasi (R-07). Ada berkas bernama itu, **tapi isinya rule lain**.
Kesimpulan `19-GAP` bahwa ia hilang **tetap benar** — tetapi siapa pun yang memverifikasi dengan
`ls` akan salah menyimpulkan sebaliknya.

**Konsekuensi metode:** seluruh pekerjaan berikutnya harus membangun inventaris dari `<pyRuleName>`,
bukan dari nama berkas. Dengan metode itu: 902 berkas Activity → **899 nama rule unik**.

## 9.4 DB Link — 71 pemakaian, bukan 64

Enam nama link **cocok persis** dengan `20-DETAIL-KOMITE-DBLINK.md:310-322`, dan 64 pemakaian di
`RDB List/` juga cocok. Tetapi ada **selisih cakupan**: `20-DETAIL` hanya menginventaris
`RDB List/`.

| DB Link | Pemakaian di `RDB List/` | Objek remote | Cocok? |
|---|---|---|---|
| `@asmd.sinarmas.co.id` | 55 | 20 objek | ✅ |
| `@simasnet` | 3 | 2 | ✅ |
| `@smi.sinarmas.co.id` | 2 | 2 | ✅ |
| `@opjava.sinarmas.co.id` | 2 | 2 | ✅ |
| `@prod_asm.sinarmas.co.id` | 1 | 1 | ✅ |
| `@prod_tka.sinarmas.co.id` | 1 | 1 | ✅ |
| **Total** | **64** | 28 | ✅ |

**Tujuh pemakaian tambahan dari SQL yang ditulis inline di `Activity/`**, termasuk **2 objek
remote yang belum tercatat**:

| Objek remote | `berkas:baris` | Tercatat `20-DETAIL`? |
|---|---|---|
| `collection.lst_account@asmd` | `Activity/CNMUpdateMasterRekening_act-Act.xml:6297`, `:6327`, `:6339` | ❌ |
| `GL.T_KASIR_FILE@asmd` | `Activity/SendAttachmenttoCashier_act-Act.xml:8461` | ❌ |
| `mst_det_sales@asmd` | `Activity/PNCGetDashboardOSInbox_Act-Act.xml:3428`, `:3439`, `:3447` | objek tercatat, lokasi tidak |

Angka `@ASMD` yang benar: **62 pemakaian, 22 objek**. Total seluruh export: **71 pemakaian**.

**Dua objek yang paling mengikat**, karena logikanya harus **ditulis ulang**, bukan sekadar
di-query lewat API:

- **`DATAMINING.GET_WORKING_HOURS@ASMD` — 17 pemakaian.** Dasar seluruh perhitungan TAT/KPI.
- **`GENERAL.HRD_LBR@ASMD`** — kalender hari libur.

## 9.5 Angka final versus angka dokumen

| Angka | Dokumen | Hasil verifikasi | Cocok? |
|---|---|---|---|
| 64 objek diminta DBA (R-01) | 56 proc + 8 fn lokal | **64** — spot-check 5/5 lolos | ✅ |
| 70 total objek DB | 64 + 6 remote | **70** | ✅ |
| 50 activity hilang (R-07) | 45 + 5 | **46** (40 + 6) | ❌ dokumen lebih tinggi 4 |
| 45 activity buatan sendiri | 45 | **40** | ❌ 5 positif palsu |
| 5 activity bawaan Pega | 5 | **6** | ❌ `pyConvertToJavaMap` terlewat |
| 8 router | 8 | **8** | ✅ |
| 4 router hilang | 4 | **4** | ✅ jumlah shape per router juga cocok |
| 3 router kritis | 3 | **3** | ✅ dengan syarat (klasifikasi `ToWorkList` = inferensi) |
| 64 pemakaian DB Link | 64 | **64** di `RDB List/`, **71** seluruh export | ⚠️ kurang cakupan |
| 6 DB Link | 6 | **6** | ✅ |
| 20 objek `@ASMD` | 20 | **22** | ❌ +2 |
| 2.167 rule XML | 2.167 | **2.167** | ✅ |
| 268 dari 269 section bergrid | 268 | **268** | ✅ |
| FR-W1 23 shape / 13 assignment / 7 decision | — | **cocok persis** | ✅ |
| FR-U3 ~20 inbox berbasis peran | ~20 | **26** | ❌ |
| FR-W2 11 ticket | 11 | **17 dirujuk, 8 ada, 9 hilang** (11 = custom di Register Flow) | ⚠️ definisi |

**Ringkas: R-01 dan R-04 tahan uji. Kelemahan terbesar `19-GAP` bukan angkanya, melainkan ruang
lingkupnya.**

## 9.6 Yang harus diminta ke pihak luar

### Tim Pega (pemilik ruleset `GCNMFW` / `GISFW`)

1. **Export ulang lengkap dengan dependensi** — pakai *Product rule* / application-based export
   dengan *include dependent rules*, bukan seleksi per rule. Export sekarang memuat 2.140 nama rule
   tetapi ±242 rule yang dirujuk tidak ikut.
2. **116 When rule buatan sendiri** — prioritas tertinggi setelah export ulang; ini aturan bisnis
   inti.
3. **9 Ticket rule** (§5.1) + activity `SetTicket`, dan konfirmasi `Status-Resolved` sebagai ticket
   standar Pega class `Work-`.
4. **3 router** (`PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter`) beserta **flow action**
   pada 8 shape assignment terkait.
5. **Library `GCNM` dan `CNM` seluruhnya** + 4 function `ASM` yang hilang.
   **Prioritas: `ASM.HmacSHA256`** (`Activity/SendFeedbackBRISurf-Act.xml:3423`).
6. **63 Flow Action · 24 Section** (khususnya `ViewCoverage`, `ViewObject`, `ViewPayment`) ·
   **9 Report Definition · 16 Data Transform · 40 Activity** (R-07 dikurangi 5 positif palsu).
7. **4 harness inbox** yang hilang (§6.5) + **6 When rule peran** (`IsGCNMUser` dkk).
8. **2 Authentication Profile** (`ServiceRekKlaimPNC`, `SERVICEKLAIMKBRU`).
9. **Konfirmasi cacat export**: `Activity/SendEmailNotification-Act.xml` dan
   `Activity/NotificationPAYDI-Act.xml` berisi rule lain (§9.3). Bila prosesnya bermasalah,
   **seluruh 902 berkas perlu divalidasi ulang**.
10. **Apakah ada rule Declare Expression, Decision Table/Tree, Agent, Job Scheduler, Service
    inbound, Access Group, Role, Privilege, Operator, Work Queue?** Tidak ada direktorinya di export
    sama sekali. Terdeteksi 10 step `Property-Map-DecisionTable` dan 3 `Property-Map-DecisionTree`
    yang menunjuk rule di luar export. (Ini juga memblokir `FR-F3`, yang bertumpu pada 22 access
    group.)

### DBA

1. Source **64 objek** — query siap pakai di `19-GAP:243-313`, terverifikasi valid.
2. **Sequence `POOLDATA.CLAIM_NO_NONPEGA_SEQ`** — belum ada, harus dibuat (D-22). **Belum tercatat
   di sepuluh tindakan hari pertama.**
3. Isi master **`POOLDATA.V_STS_CLAIM`** (`LSC_ID`, `LSC_NOTE`) — R-06.
4. Isi **`POOLDATA.EMAILKOMITE`** — D-14.
5. Isi **`POOLDATA.DB_LINK_PEGA`** — master pemetaan hostname → entitas yang menentukan perilaku
   `GetLinkAppClaim` (§1.3).
6. `SELECT DISTINCT statusposisi FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC` (§4.3).
7. Definisi **6 DB Link** — host, service name, user, hak akses.
8. DDL **2 objek remote tambahan**: `COLLECTION.LST_ACCOUNT@ASMD`, `GL.T_KASIR_FILE@ASMD`.
9. Sinonim publik untuk 2 objek yang dipanggil tanpa skema: `KONVERSIT_VEHICLELIST`,
   `PROCESSNEWEDMOBJECTDATA`.
10. Isi `POOLDATA.PROGRESS_CLAIM_PNC` dan `GENERAL.GET_TOKEN_STORAGE`.
11. DDL lengkap + statistik ukuran tabel — R-08.

### Tim pemilik sistem

| Sistem | Yang diminta |
|---|---|
| **ASM / ASMD** (62 dari 71 pemakaian) | Kontrak API pengganti 22 objek remote. Paling kritis: `DATAMINING.GET_WORKING_HOURS` (17×, dasar TAT/KPI) dan `GENERAL.HRD_LBR` (kalender hari libur) — logikanya harus **ditulis ulang** |
| **Simasnet** · **SMI** | `GENERAL.MST_BUKA_PROTEKSI`, `V_KLAIM` — jalur Open Protection |
| **OPJAVA** | `NEW_GENERAL.M_USER`, `M_USER_JOB` — beban kerja per user; **terkait langsung dengan 3 router yang hilang** |
| **TKA** | `ANEKA.MST_SHARE_PU` — master share Penanggung Utama untuk spreading |
| **PROD_ASM** | `POOLDATA.AGENT` |
| **Pemilik API HCC/HCQ** | Kontrak login — **pihak luar yang belum ada di daftar sepuluh tindakan** (§1.1) |
| **Pemilik Google Cloud Storage** | Bucket, auth, konvensi nama objek; pemiliknya **TIDAK DAPAT DIPASTIKAN DARI EXPORT** (§3) |
| **BRI** | Skema tanda tangan `ASM.HmacSHA256`; dan konfirmasi endpoint sandbox versus produksi (§1.4) |

---

# 10. Cakupan modul gelombang 1 (F-1…F-5, U-1, U-2, S-5)

Delapan modul yang akan ditiketkan penuh. Angka di bagian ini adalah **bahan acceptance criteria
langsung**.

## 10.1 F-1 Kerangka Aplikasi

| Aspek | Temuan | Bukti |
|---|---|---|
| **Konfigurasi** | Sistem lama praktis **tidak punya lapis konfigurasi**. Hanya **1** Dynamic System Setting yang dirujuk (`ServiceFromTable`, 4 rujukan), dan **instansnya tidak ada di export** | `Connect REST/UpdateSearchDataRekeningToKasir-ConnectREST.xml:99`; `InjectDataRekeningToKasir-ConnectREST.xml:153`; `GeneratedPolisFileKlaim-ConnectREST.xml:101`; `ServiceToKBRURefKlaim-ConnectREST.xml:88` |
| | `Data-Admin-DB-Name` (definisi koneksi & pool DB) **tidak ada di export** | — |
| | Konfigurasi lingkungan tersebar di **tiga** tempat: literal di activity, tabel `POOLDATA.DB_LINK_PEGA`, dan Auth Profile yang tidak diekspor | `Activity/GetLinkAppClaim-Act.xml:765` (**dipanggil dari 52 berkas**); `RDB List/GetAppClaim-SQL.xml:73` |
| **Error handling** | **182 dari 902** activity (20,2%) punya penanganan error eksplisit; **720 (79,8%) tidak punya sama sekali** | §10.1 tabel bawah |
| | **0** handler error terpusat (`pyFailureStatus`, `pzStandardErrorHandler`, `pxErrorHandler` — semuanya nihil) | — |
| | 174 blok `catch` di **92** activity; 55 transisi `StepStatusFail` di **25** activity | `Activity/acquireWorkObject-Act.xml:1110-1111`, `:1668` |
| | **0 `ROLLBACK`** dan **0 `EXCEPTION WHEN`** di seluruh export — 48 `COMMIT;` PL/SQL tanpa satu pun handler | — |
| **Logging** | **207** `oLog.error`, 11 `debug`, 4 `info`, **0 `warn`**; hanya 2 step `Log-Message`. Format **teks bebas berkonkatenasi**, tanpa field, tanpa correlation ID | `Activity/ActivationRetrievePage_Act-Act.xml:2151` (`oLog.error("ReloadSection:Invalid JSON Stream…"+e.getMessage())`) |
| | Salah tulis `"Expection"` diduplikasi ke 8+ activity → **pola copy-paste, bukan util bersama** | idem `:2153` |
| **Health check** | **0** — `healthcheck`/`readiness`/`liveness`/`heartbeat`/`actuator`/`/health` nihil di `Activity/`, `Connect REST/`, `Harness/`, `DataPage/` | — |

Contoh kedua pola error handling, keduanya menyentuh data:

| Activity | Step simpan | `PAGE-SET-MESSAGES` | `catch` | `StepStatusFail` | Akibat kegagalan |
|---|---|---|---|---|---|
| `Activity/InputRegister_act-Act.xml` | 1× `OBJ-SAVE` | **10×** | ada | ada | tertangani |
| `Activity/InsertAdjustmentList-Act.xml` | **3× `RDB-SAVE`** (`:9700`, `:10008`, `:10181`) | **0** | **0** | **0** | kegagalan simpan ke-2 meninggalkan tabel ke-1 tertulis — **tanpa pesan, tanpa rollback** |

**Konsekuensi tiket F-1.** Daftar setting **tidak bisa disalin dari sistem lama** — harus disusun
dari daftar hal yang sekarang hardcode. Kandidat yang sudah terbukti: base URL aplikasi (1 titik,
52 pemakai), 6 nama DB Link, 2 nama auth profile, 8 ambang komite (§7.5), 3 batas tanggal 7/30/90.

## 10.2 F-2 Akses Data

### Skema dan tabel

**9 skema unik** — dua menanggung 97% trafik, tujuh hanya dijangkau **lewat DB Link**:

| Skema | Baris SQL | Berkas | Akses |
|---|---|---|---|
| `POOLDATA` | 821 | 395 | langsung |
| `DATAPEGA` | 147 | 102 | langsung |
| `GENERAL` | 15 | 13 | campuran |
| `COLLECTION` · `NEW_GENERAL` · `MBU` · `HRDASM` · `GL` · `ANEKA` | 1–2 masing-masing | 1 masing-masing | **hanya DB Link** |

Contoh: `RDB List/ActivationRetrieveT_ObjectList-SQL.xml:7` · `RDB List/BroswseKlaimperday-SQL.xml:67` ·
`RDB List/GetShareTKA-SQL.xml:48` · `RDB List/BrowseNonMBUUsers-SQL.xml:29`.

| Sumber inventaris tabel | Objek unik |
|---|---|
| Disebut langsung di SQL `RDB List/` | **126** |
| Kelas `ASM-FW-*FW-Int-<TABEL>` | 78 |
| **Union, dedup** | **177** (11 view `V_*`, 8 tabel kerja Pega `PC_*`/`PR_*`) |

> **Klaim "245 tabel" tidak dapat direproduksi** — selisih ±68 objek. Kandidat penyebab: tabel yang
> hanya disentuh 64 procedure yang source-nya tidak ada (R-01), mis.
> `POOLDATA.PEGA_JSON_INSERT_HISTORY_CLAIM_PNC` (`RDB List/InsertHistoryClaimPNC-SQL.xml:69`),
> `POOLDATA.INSERT_SALVAGE` (`RDB List/InsertNewSalvage-SQL.xml:11`).

### Konstruksi Oracle non-portabel — daftar kerja nyata untuk D-20

| Konstruksi | Kemunculan | Berkas | Contoh |
|---|---|---|---|
| `TO_CHAR(` | **411** | 79 | `RDB List/BroswseDLA_SQL-SQL.xml:71` |
| `\|\|` konkatenasi | 191 | 46 | `RDB List/AmbilDataKlaimDenganNoRekening-SQL.xml:56` |
| `TRUNC(` | 150 | 44 | `RDB List/BroswseKlaimByBirthDate-SQL.xml:91` |
| `TO_DATE(` | 137 | 49 | `RDB List/BroswseKlaimByBirthDate-SQL.xml:85` |
| `NVL(` | 100 | 22 | `RDB List/BroswseKlaimByRegisterDate-SQL.xml:39` |
| `TO_NUMBER(` | 83 | 35 | `RDB List/ActivationRetrieveT_ObjectList-SQL.xml:6` |
| `FETCH NEXT/FIRST … ROWS ONLY` | 72 | 35 | `RDB List/BroswseDLA_SQL-SQL.xml:71` |
| `SYSDATE` | 69 | 40 | `RDB List/CallProccedureInsertMitra-SQL.xml:62` |
| `SUBSTR(` | 69 | 28 | `RDB List/BroswseKlaimByPolicyNo-SQL.xml:100` |
| **`ROWNUM`** | **68** | 44 | `RDB List/BrowseCaseNotAssigned-SQL.xml:82` |
| `ADD_MONTHS` | 18 | 2 | `RDB List/BrowseDataPolicyRNWAllFilter_SQL-SQL.xml:100` |
| `DECODE(` | 16 | 7 | `RDB List/BrowseGetLsbsid_SQL-SQL.xml:47` |
| `ROW_NUMBER() OVER(` | 14 | 13 | `RDB List/ActivationRetrieveT_PersonList-SQL.xml:68` |
| `DUAL` | 12 | 11 | `RDB List/CheckDataPaymentPA_TRAVEL-SQL.xml:15` |
| `INSTR(` | 11 | 8 | `RDB List/GetBrowseDataAIPA-SQL.xml:113` |
| `LAST_DAY` · `RPAD`/`LPAD` | 6 · 6 | 2 · 2 | `RDB List/BrowseDataPolicyRNWAllFilter_SQL-SQL.xml:104`, `:100` |
| `LISTAGG` | 4 | 4 | `RDB List/GetDataDominanFactorListOS-SQL.xml:6` |
| `MONTHS_BETWEEN` · `(+)` outer join | 2 · 2 | 1 · 2 | `RDB List/BroswseKlaimByRegisterDate-SQL.xml:49`; `RDB List/GetKomitePAOutstanding-SQL.xml:86` |
| `CONNECT BY` | **1** | 1 | `RDB List/CheckWeekDays_SQL-SQL.xml:77` |
| `MERGE INTO` · `START WITH` · `NVL2(` · `REGEXP_*` · `SEQ.NEXTVAL` | **0** | 0 | tidak ada |

Ditambah **PL/SQL anonim di dalam rule SQL**: `DECLARE` 47 baris · `BEGIN` 65 · `COMMIT;` 48
(mis. `RDB List/CheckSTS_ABS-SQL.xml:33-45`). Blok ini **tidak portabel apa pun dialeknya** dan
harus dipindah ke Go.

Mask tanggal di SQL: `'dd/mm/yyyy'` 379× · `'ddmmyyyy'` 6× · `'dd/mm/yy'` 6× · `'yyyymmdd'` 4× ·
`'yyyy/mm/dd'` 1× · dan **satu salah tulis `'dd/mm/yyy'`** 1×.

> **Koreksi terhadap D-20 / `08-TECHNICAL-STRATEGY.md:231` / `09-DATABASE-STRATEGY.md:83`.**
> Dokumen menyatakan `ROWNUM` diganti `OFFSET … FETCH NEXT` dan bahwa *"pola ini sudah dipakai di
> 35 rule"* sehingga bukan hal baru bagi tim. Angka 35 benar, tetapi **mayoritasnya
> `FETCH NEXT 1 ROW ONLY`** — ambil satu baris teratas, **bukan paginasi**. Dan `OFFSET` **nol
> kemunculan** di seluruh export. Jadi tim **belum pernah menulis paginasi `OFFSET … FETCH`**;
> klaim "bukan hal baru" tidak didukung. Angka `ROWNUM` **68 di 44 berkas** cocok persis dengan
> `09-DATABASE-STRATEGY.md:71` dan dapat dipakai apa adanya.

### Transaksi

| Mekanisme | Jumlah |
|---|---|
| Step `COMMIT` | 103 step di 80 activity |
| Step `OBJ-SAVE` | 109 step di 84 activity |
| Step `RDB-SAVE` | 63 step di 41 activity |
| **Step `ROLLBACK`** | **0** |
| `WriteNow` (bypass commit) | 288 kemunculan di 85 activity — `Activity/AttachToWork-Act.xml:2681` |
| `COMMIT;` di dalam SQL | 48 baris di ±30 berkas |

**51 dari 118** activity yang punya step simpan **tidak punya `Commit` sama sekali**; **14**
menyimpan ≥2× tanpa commit tunggal. Contoh: `Activity/InsertAdjustmentList-Act.xml` (3×
`RDB-SAVE`, 0 commit) · `Activity/CheckAllData_Act-Act.xml:3024`, `:3162`, `:3638`.

Sistem lama menyandarkan atomisitas pada auto-commit implisit Pega **dan** pada `COMMIT;` di dalam
blok PL/SQL yang commit **di tengah** transaksi pemanggil
(`RDB List/InsertHistoryClaimPNC-SQL.xml:70`). Tidak ada satu pun rollback eksplisit.

### Perangkaian SQL dari nilai pengguna — pelanggaran `BRD §21.2` #11

Terverifikasi ulang oleh orkestrator:

| Pola | Kemunculan | Berkas | Sifat |
|---|---|---|---|
| `{Page.Property}` — **bound parameter** | 1.366 | 447 dari 652 | aman |
| **`{ASIS:Page.Property}` — substitusi literal** | **538** | **233 dari 652** | **injeksi SQL** |
| `{Param.x}` | **0** | 0 | tidak dipakai sama sekali |

**17 baris paling berbahaya** — `{ASIS:}` di dalam `LIKE '%…%'`, semuanya menerima kotak pencarian
pengguna. Terverifikasi persis:

| `berkas:baris` | Konteks |
|---|---|
| `RDB List/BroswseKlaimByName-SQL.xml:17` | `like '%{ASIS:InputData.CARI4}%'` |
| `RDB List/BroswseKlaimByDLANo-SQL.xml:97` | idem |
| `RDB List/BroswseKlaimByObjectName-SQL.xml:102` | idem |
| `RDB List/BroswseKlaimByPLANo-SQL.xml:13` | idem |
| `RDB List/BrowseAPPName_sql-SQL.xml:89` | `like '%{ASIS:TempError.ID_Offer}%'` |
| `RDB List/BrowseServiceName_sql-SQL.xml:64` | `like '%{ASIS:TempError.source}%'` |
| `RDB List/GetAppClaim-SQL.xml:73` | `like '%{ASIS:TempGetApp.LSC_NOTE}%'` |
| `RDB List/GetClientName-SQL.xml:103` | `like '%{ASIS:TempDcol.ATASAN}%'` |

Pola kedua yang **tidak bisa** diubah jadi bind parameter satu-per-satu: `{ASIS:}` menyuntik
**fragmen klausa `WHERE` utuh** (`RDB List/BroswseKlaimByBirthDate-SQL.xml:91`) dan bahkan
**daftar kolom `SELECT`** (`RDB List/ActivationRetrieveT_ObjectList-SQL.xml:5`). Ini menuntut
**query builder** di F-2, bukan sekadar penggantian placeholder.

Tambahan dari §8.2: `Activity/InputRegister_act-Act.xml:7750` merangkai fragmen SQL dari
`ClaimData.Location` **di dalam activity**, bukan di rule SQL.

## 10.3 F-3 Identitas & Akses

### 22 access group — terverifikasi seluruhnya

Semua 22 nama di `docs/requirement-summary.md:118-121` ditemukan sebagai literal
`GCNMFW:<nama>`. **Tidak ada access group ke-23.** Yang terbanyak dirujuk: `Administrators` 137× ·
`CaseManager` 87× · `PncManagerAdmin` 62× · `PNCReportClaimEksternal` 43× · `ViewClaimPNC` 42×.
Contoh: `When/IsAdministrators-When.xml:9` · `When/IsAdminPNC-When.xml:312`, `:553`, `:676`.

> **Temuan tambahan:** tiga nama muncul dalam **dua kapitalisasi** (`ViewClaimPNC`/`VIEWCLAIMPNC`,
> `PncReceive`/`PNCRECEIVE`). Perbandingan access group di sistem lama **tidak konsisten soal
> case** — kandidat bug, dan hal yang harus dinormalkan di F-3.

### 47 menu — angka benar, definisinya berbeda

`Navigation/pyCaseWorkerNavigation-Navigation.xml` (39.822 baris; **satu-satunya** navigation rule):

| Ukuran | Nilai |
|---|---|
| Elemen navigasi (`Embed-Rule-Navigation-Element`) | 107 blok (termasuk salinan `pzSavedElementPage`) |
| Item bertipe `pyType` | **54**: `Action` 51 · `Navigation` 2 · `Separator` 1 |
| Aksi | `showHarness` 96 · `runScript` 6 · `openUrlInWindow` 2 · `logOff` 2 · `load` 1 |
| **Harness target unik** | **47** |
| Hirarki | **2 level** — 51 daun + 2 sub-menu (`:2783`, `:39375`); rule sub-navigasinya **tidak ada di export** |

> **Angka 47 terverifikasi — sebagai 47 harness target unik, bukan 47 baris menu.** Baris menu
> sebenarnya **51 item aksi** (+1 separator, +2 sub-menu). Selisihnya karena beberapa item menuju
> harness yang sama.

**Sebelas dari 47 harness target tidak ada di export** — 7 layar portal proyek + 4 OOTB Pega:
`InboxCloseClaim_Harness`, `InboxOutstanding_Harness`, `InboxRequestSalvage`, `InboxServiceCenter`,
`LostAdjuster_harness`, `PNCViewClaim`, `ReportProduksiPA_harnes` · `pyCMCases7`, `pyCMEvents7`,
`pyTagDashboard`, `pyTeamDashboard`. **Gap baru, belum ada nomor `R-nn`.**

### Otorisasi sistem lama = penyembunyian menu saja — terbukti dengan angka

Terverifikasi ulang oleh orkestrator:

| Ukuran | Nilai | Bukti |
|---|---|---|
| **Activity dengan `pyPrivilegeName` non-kosong** | **1 dari 902** | `Activity/DownloadFile-Act.xml:314` → `zipMoveExport` |
| Activity dengan `<pyPrivilegeName/>` kosong | **900** | `Activity/AllCoveredResolved-Act.xml:123` |
| `pxRequiresPrivilege` · `HasPrivilege` · `Access-Deny` | **0** | — |
| Item menu dengan `pyVisibleForPrivilege` terisi nilai | **0 dari 54** (semua placeholder `ruleRef` dengan nilai kosong) | `Navigation/…:2176` |
| Item menu dengan `pyVisible=ConditionWhn` | **54 dari 54** | `Navigation/…:2164` |
| When rule visibilitas menu | 52 rujukan, **34 nama When** berbeda | `Navigation/…:2717`, `:2806` |
| When rule yang membaca `pyAccessGroup` | **25 dari 70** | `When/IsAdminPNC-When.xml:312` |
| Section / Harness memakai When berbasis peran | 9 dari 269 · 1 dari 74 | — |

> **`FR-R1` dan `BRD §21.2` #8 terbukti, bukan diasumsikan.** Dari 902 activity, **901 tidak punya
> pemeriksaan izin apa pun** — siapa pun yang bisa memanggil URL activity-nya bisa
> menjalankannya. Satu-satunya privilege yang ada (`zipMoveExport`) adalah privilege bawaan Pega
> untuk ekspor ruleset, bukan aturan bisnis klaim.

### Tidak ada tabel izin menu di database

| Yang dicari | Hasil | Bukti |
|---|---|---|
| Tabel pemetaan operator → access group | **hanya alias operator**: `POOLDATA.T_ACCESS_GROUP_PNC` (`OPERATOR_ID`, `OLD_OPERATOR_ID`, `STS_AKTIF`) | `RDB List/GetOperatorID-SQL.xml:73`, `RDB List/GetOperatorIDs-SQL.xml:97` |
| **Tabel izin menu (peran → menu)** | **TIDAK ADA** — nihil objek `*MENU*`/`*ROLE*`/`*PRIV*`/`*PERMISSION*` di 177 tabel | — |
| Tabel peran teknis | `POOLDATA.MST_USER_TEKNIK`/`MST_USER_TEKNIS` — daftar PIC teknik, **bukan** peran otorisasi | `Report Definition/BrowseVMstUserTeknis_RD-RD.xml` |

Pemetaan **22 access group → 51 item menu** hidup **hanya di dalam 34 When rule + XML navigasi**,
bukan data. Ini memperkuat `FR-F3` *"tabel peran & izin menu yang belum ada"*: F-3 harus
**membuat** tabelnya dari nol, dan sumbernya adalah **membaca 34 When rule satu per satu** —
enam di antaranya **hilang dari export** (§6.5).

## 10.4 F-4 Master Data — daftar 14 master jauh kurang

### Peta 14 master ke export

| # | Master | Status | Bukti utama |
|---|---|---|---|
| 1 | Lini Bisnis | ✅ ADA | `POOLDATA.BUSINESS`, `BUSINESSGROUP`, `PANEL_HE`; `Report Definition/BrowseBusiness_RD-RD.xml` |
| 2 | Penyebab Kerugian | ✅ ADA | `M_CAUSE_OF_LOSS`, `V_D_CAUSE_OF_LOSS_BUSINESS`; `Harness/CauseOfLossInbox` |
| 3 | Cabang & Wilayah | ✅ ADA | `BRANCH`, `M_BRANCH`, `CITY`, `PROVINCE`; `Report Definition/BrowseBranch_RD-RD.xml` |
| 4 | Surveyor & Adjuster | ✅ ADA | `T_SURVEYORLIST`, `V_D_SURVEYORS`, `MST_LOGIN_SURVEYOR`; `Harness/MasterLoginSurvey-Harness.xml` |
| 5 | Jenis Treaty | ✅ ADA | `REINSURANCETYPE`, `T_REINSURER`, `MST_XOL_*`; `Harness/DetailMasterXOL-Harness.xml` |
| 6 | Mata Uang & Kurs | ✅ ADA | `CURRENCY`, `CURRENCYSTANDARD`, `KURS_STANDARD` |
| 7 | Bank & Rekening | ✅ ADA | `LST_ACCOUNT`, `LST_BANK_GROUP`, `MST_VIRTUAL_ACCOUNT_PNC`; `Harness/MasterRekening-Harness.xml` + **7 section alur Approval/Approve/Reject** |
| 8 | Jenis Dokumen | ✅ ADA | `LST_TYPE_DOC_BUSINESS`, `V_LST_DET_TYPE_DOC`, `M_DOCTRAVEL` |
| 9 | **Ambang Komite** | ⚠️ **sudah berupa DATA**, bukan hardcode | **`POOLDATA.EMAILKOMITE`** — `DEGREE`, `TYPE_BUSINESS`, `TYPE_KOMITE`, **`LIMIT_BOTTOM`**, `STS_*`; **17 rule SQL** mis. `RDB List/EmailKomiteBerjenjang_sql-SQL.xml:52` |
| 10 | Penerima Notifikasi | ✅ ADA | `EMAILKOMITE.EMAIL` + flag `STS_*`, `LST_EMAIL_PREMI`, `M_KOMUNIKASI_PNC` (46 rujukan) |
| 11 | **Ambang Large Losses** | ❌ **TIDAK ADA** | tidak ada tabel di 177 objek |
| 12 | **Batas Aturan Tanggal** | ❌ **TIDAK ADA — hardcode** | `Activity/InputRegister_act-Act.xml:5516`, `:5531`, `:5563` (7 hari) · `:6480` (30) · `:5092`, `:5102`, `:5115`, `:5396` (90) — **8 titik**, dan **nilainya sama untuk semua lini bisnis** |
| 13 | **Peran & Izin Menu** | ❌ **TIDAK ADA** | §10.3 |
| 14 | Master Status Klaim | ✅ ADA (tabel) | `V_STS_CLAIM` (16 rujukan), `GCNM_MST_PROGRESS`; arti kode tetap R-06 |

> **Koreksi terhadap `05-DOMAIN-MODEL.md:184`.** Dokumen mengklaim master Ambang Komite
> menggantikan hardcode `50000000`, `30000000`, `3500`. Matriks jenjang × jenis bisnis **sudah
> berupa master data** di `EMAILKOMITE.LIMIT_BOTTOM` + `DEGREE`. Yang benar-benar hardcode dan
> harus dipindahkan adalah: **ambang Ex-Gratia `250000000`** (`Activity/SetListComiteeClaimAI-Act.xml`,
> `Activity/SetListComiteeClaimPerObjAdj-Act.xml`) dan **ambang open protection `50000000000`**
> (`Activity/CheckEstimateValue-Act.xml:9617`, `:9831`, `:6447`) — plus 8 ambang di §7.5.

### Master yang ADA di export tapi TIDAK ada di daftar 14

Ini **mengubah ukuran F-4 dan U-6 secara material**. Lima belas kelompok tambahan, yang terbesar:

| Master tambahan | Objek DB | Layar |
|---|---|---|
| **Sparepart / Bengkel HE** | **8 tabel** — `BENGKEL_HE`, `SPAREPART_HE`, `SPAREPART_HE_VIN_KEY`, `GCNM_M_SPAREPART_CATEGORY`, `GCNM_M_SPAREPART_TYPE`, `NOTIF_RANGKA_HE`, `PANEL_HE`, `LOKASI_PANEL_HE` | **5 harness + 20 section**, termasuk alur persetujuan 4 tahap: `Harness/GCNMMasterSparepartType-Harness.xml`, `Harness/BengkelHE-Harness.xml`, `Harness/SparePart_HE-Harness.xml`; `Report Definition/BrowseBengkelHE_RD-RD.xml` |
| Master Pasal / Klausul | `CLAUSE`, `M_CLAUSE`, `V_M_DATA_PASAL` | `Harness/DetailMasterPasalRejected-Harness.xml` |
| Master Penolakan Klaim | `MST_PENOLAKAN_KLAIM_2`, `MST_REJECTED_KOMITE`, `M_PERIHAL_RCLPUCL` | `Harness/PNC_MasterTolakKlaim-Harness.xml` |
| Master Supplier / Mitra | `M_SUPPLIER`, `TENDER_SUPPLIER`, `LST_MITRA` | `Harness/MasterSupplier-Harness.xml` |
| Master Proteksi / Visibility | `MST_PROTEKSI_DATA_PNC`, `MST_BUKA_PROTEKSI` | `Harness/MasterProteksiVisibilityData-Harness.xml` (grid **21 kolom**) |
| Master Recovery · Auto Klaim · Dominan Factor · Template Korespondensi · Blacklist/Whitelist · KPI & Komisi · Virtual Account · Akseptasi · Pengguna Teknik · Objek/Occupation/Paket | `MST_RECOVERY_ASM_PENJAMINAN`, `M_AUTO_CLAIM_PNC`, `M_DOMINAN_FACTOR`, `M_TEMPLATE`, `M_CLIENTBLACKLIST`, `M_KPI_PNC`, `MST_VIRTUAL_ACCOUNT_PNC`, `MST_AKSEPTASI`, `MST_USER_TEKNIK`, `OBJECTITEMTYPE` dll. | berbagai |

Ukuran keseluruhan: **≥29 kelompok master**, **49 section bernama `Master*`/`Mst*`**,
**11 harness bernama `*Master*`**.

> Sub-domain Sparepart/Bengkel HE saja adalah 8 tabel dengan alur persetujuan 4 tahap di 20
> section — **itu ukuran modul tersendiri, bukan "1 dari 14 master"**. Sizing `F-4` "Sedang–Besar"
> dan `U-6` "Sedang" (`06-MODULE-BREAKDOWN.md:20`, `:72`) kemungkinan besar **terlalu rendah**.
>
> Catatan lingkup: `Bengkel`/`Sparepart`/`Supplier` adalah area yang di GATE 0 Anda putuskan
> **diabaikan, tidak masuk scope**. Bukti di sini menunjukkan area itu punya 8 tabel + 20 section
> + 5 harness + alur persetujuan. Keputusan Anda tetap dihormati; angkanya dicatat agar
> konsekuensinya terlihat, dan agar tiket `F-4`/`U-6` tidak diam-diam menyertakannya.

## 10.5 F-5 Waktu & Zona Waktu — "Kecil" tidak akurat

### Sensus penyesuaian jam manual — terverifikasi ulang oleh orkestrator

| Pola | Kemunculan | Berkas |
|---|---|---|
| `addCalendar(x,0,0,0,0,**7**,0,0)` = **+7 jam** | **102** | **34 activity** |
| `addToDate(x,"","7","","")` = +7 jam | 12 | 2 activity |
| `addToDate(...,0,7,0,0)` di ekspresi FormatDateTime | 4 | 1 activity |
| **Union berkas dengan penyesuaian +7 jam apa pun** | — | **36 activity** |
| `addCalendar(...,0,0,0,0,**12**,0,0)` = **+12 jam** | **101** | **4 activity** |
| Konstanta `25200` · `* 3600` · `addHours` · `plusHours` | **0** | 0 |
| **Konversi arah balik (−7 jam)** | **0** | **0** |

Ada activity helper bernama literal **`Set7Hours`** (`Activity/Set7Hours-Act.xml:232` —
`@DateTime.addCalendar(Param.In,0,0,0,0,7,0,0)`; `:235` menerapkannya ke `.ClaimData.DateOfLoss`),
dipanggil dari `Activity/InputRegister_act-Act.xml:13324`. **Tetapi 33 activity lain menulis ulang
aritmetika yang sama inline** — helper-nya ada dan tidak dipakai.

> **Klaim "7-jam-manual tersebar" terbukti, dan lebih berat dari yang diklaim.** **118 titik** di
> **36 activity**, plus **101 titik +12 jam** di 4 activity yang maksudnya
> **TIDAK DAPAT DIPASTIKAN DARI EXPORT**.
>
> **Nol konversi arah balik.** Setiap nilai yang sudah digeser +7 jam lalu disimpan akan digeser
> **lagi** bila lewat jalur yang sama dua kali. Ini risiko korupsi data — dan menjadi acceptance
> criteria F-5 yang wajib: **uji idempotensi**, plus pertanyaan apakah sudah ada data produksi
> yang tergeser ganda (+14 jam).

### Tiga representasi penyimpanan, empat definisi "sekarang"

| Representasi | Bukti |
|---|---|
| `DateTime` GMT Pega, format `YYYYMMDDThhmmss.SSS GMT` | `Activity/AchiveDocument_klaimAdmin-Act.xml:72` (pola muncul 183.689×) |
| `DATE` Oracle via `TO_CHAR`/`TO_DATE` mask lokal | `RDB List/BroswseDLA_SQL-SQL.xml:71`, `RDB List/BroswseKlaimByBirthDate-SQL.xml:91` |
| Teks `yyyyMMdd` dikonversi di aplikasi | `Activity/Set7Hours-Act.xml:235` (`@DateTime.toDateTime(.ClaimData.DateOfLoss,"yyyyMMdd","","")`) |
| String GMT Pega di-parse **di dalam SQL** | `RDB List/InsertNewSalvage-SQL.xml:11` (`TO_TIMESTAMP(…,'YYYYMMDD"T"HH24MISS.FF3 "GMT"')`) |

**Tidak ada satu pun kolom yang membawa offset zona waktu.**

| Definisi "sekarang" | Kemunculan | Zona |
|---|---|---|
| `@CurrentDateTime()` **tanpa argumen zona** | 190 di 82 berkas | zona server JVM |
| `@CurrentDate/@FormatDateTime(…,"Asia/Jakarta")` | 137 di 23 berkas | benar |
| `SYSDATE` di SQL | 69 di 40 rule | zona server DB |
| **`@CurrentDate(mask,"WIB")`** | **25 di 16 berkas** | **`"WIB"` bukan ID zona IANA/Java yang sah** |

Contoh literal `"WIB"`: `Activity/CekStatusOPCKlaimPNCPengkinianData-Act.xml:685` ·
`Activity/CheckViewPolis_act-Act.xml:3164` · `Activity/DLACoins_act-Act.xml:3378`.

> **Empat definisi "sekarang" berdampingan.** Apa yang dilakukan runtime Pega dengan literal
> `"WIB"` **TIDAK DAPAT DIPASTIKAN DARI EXPORT** — tapi keberadaan 25 pemakaiannya berdampingan
> dengan 137 pemakaian `"Asia/Jakarta"` sudah cukup membuktikan **tidak ada satu sumber waktu**.
> Bila kedua jalur menghasilkan tanggal berbeda, ada bug tanggal yang aktif sekarang.

## 10.6 U-1 Kerangka SPA

`pyHarnessPurpose` **tidak ada** di `Harness/` (0 dari 74 berkas) — pengukuran memakai kelas rule.

| Ukuran | Nilai |
|---|---|
| Total harness | **74** |
| Kelas `Data-Portal` (layar portal) | **58** |
| Kelas `@baseclass` | 9 |
| Kelas `ASM-FW-GCNMFW-Work*` (form kasus) | 7 |
| **Rute tingkat atas** (target menu) | **47** unik — 36 ada, **11 hilang** |
| **Rute anak / modal** (harness tanpa entri menu) | **38** |
| `pyIsModalWindow` (kemunculan) | 695 |

Dari 38 harness tanpa entri menu, **mayoritas adalah layar master data** (`MasterRekening`,
`MasterSupplier`, `MasterRecovery`, `BengkelHE`, `SparePart_HE`, `GCNMCatSparepart`,
`MasterProteksiVisibilityData`, `PNC_MasterTolakKlaim`, `DetailMasterXOL`, `StatusClaimInbox`, …)
— **sekali lagi menunjukkan U-6 lebih besar dari "Sedang"**.

`pyHarnessPurpose` tidak ada berarti export **tidak membedakan** layar berdiri sendiri dari
modal/panel. Pemisahan 47 rute atas vs 38 rute anak adalah **inferensi dari keberadaan entri
menu**, bukan metadata — perlu konfirmasi work owner.

## 10.7 U-2 Pustaka Komponen — klaim 18–27 kolom salah

### Jumlah kolom grid

Metode: setiap kolom grid Pega punya satu blok `<pyGridColumnProps>`; runs dipisah bila jarak > 60
baris. Diverifikasi terhadap `Section/PNCInboxAdmin-Section.xml:16351` yang menyatakan
`<pyColumnCount>15` lalu tepat 15 blok di `:16365`–`:16659`.

**375 grid di 197 section:**

| Ukuran | Nilai |
|---|---|
| minimum | **1** |
| kuartil-1 | 4 |
| **median** | **6** |
| kuartil-3 | 9 |
| maksimum | **39** |
| rata-rata | 6,6 |
| **grid 18–27 kolom** | **3 (0,8%)** |
| grid < 18 kolom | **371 (98,9%)** |
| grid > 27 kolom | 1 |

Modus: 3 kolom (62 grid) dan 6 kolom (50 grid). Grid ≥18 kolom hanya **empat**: 21, 21, 25, 39.
Yang terbesar: `Section/Sec_SegmentF06-Section.xml:6509` (**39 kolom**) ·
`Section/PNCStudyClaim-Section.xml:13343` (25) · `Section/Sec_SegmentD01_1-Section.xml:12008` (21) ·
`Section/ReportKPI_Section-Section.xml:40608` (21).

> **Klaim "tabel baku 18–27 kolom" (`06-MODULE-BREAKDOWN.md:68`) SALAH.** Rentang nyata **1–39
> dengan median 6**. API komponen tabel harus dirancang untuk **6–15 kolom sebagai kasus normal**,
> dengan satu kasus ekstrem 39 kolom yang butuh horizontal scroll/pinning.
>
> Batas metode: 197 section punya `pyGridColumnProps`; 71 section bergrid sisanya (dari 268)
> memakai repeat layout tanpa metadata kolom — jumlah kolomnya
> **TIDAK DAPAT DIPASTIKAN DARI EXPORT** dengan metode ini.

### Paginasi — klaim `OFFSET 500000` tidak berdasar

Terverifikasi ulang oleh orkestrator:

| Yang dicari | Hasil |
|---|---|
| `OFFSET` di `RDB List/`, `Report Definition/`, `DataPage/`, `When/`, `Data Transform/`, `Section/` | **0 kemunculan** |
| `500000` | **2 kemunculan**, keduanya bagian dari `5000000000` (nilai rupiah) — `RDB List/BrowseClaimStudy-SQL.xml:93` |

> **Klaim `OFFSET 500000` di `docs/requirement-summary.md:162` TIDAK TERBUKTI.** Konstruksi
> `OFFSET` tidak ada sama sekali di sistem lama, dan angka `500000` pun tidak ada. Itu ilustrasi,
> bukan temuan. Rujukan turunannya di `docs/Steering/09-DATABASE-STRATEGY.md:151` dan
> `docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md:80` mewarisi masalah yang sama, dan **NFR-13
> harus ditulis ulang**.

Masalah performa yang **nyata** — dan lebih buruk:

| Mekanisme | Jumlah | Bukti |
|---|---|---|
| **Grid terikat page list klipboard** (`pySourceType=Property`) | **3.189** | vs `Report Definition` 48× dan `Data Object` 25× |
| **`pyListLoadMode=auto`** (muat semua) | **2.331** | vs `lazyload` **1×** — `Section/ViewInputReceiveDocument_sec-Section.xml:10120` |
| Paginasi database **sejati** (`ROW_NUMBER()` + `Pagination.FirstRow/LastRow`) | **19 dari 652** rule SQL | `RDB List/ActivationRetrieveT_PersonList-SQL.xml:64-75` — dan itu pun lewat `{ASIS:}`, bukan bind |
| `FETCH NEXT n ROWS ONLY` | 35 rule, **mayoritas `FETCH NEXT 1 ROW`** = ambil satu baris teratas | `RDB List/BroswseDLA_SQL-SQL.xml:71` |
| **`pyMaxRecords=500`** di Report Definition | **54 dari 56** | pemotongan diam-diam |
| Report Definition **tanpa batas** (`0`) | 2 | `Report Definition/BrowseCity_RD-RD.xml:154` |
| Report Definition batas **50.000** | 1 | `Report Definition/BrowseBranch_RD-RD.xml:184` |
| `pyPageMode` | `Numeric` 299 · `None` 204 · `Next Previous` 161 · `First X Results` 4 · `Progressive Load` **1** | — |

> **Sistem lama pada dasarnya TIDAK server-side.** `pyMaxRecords=500` pada 54 dari 56 laporan
> adalah **pemotongan hasil, bukan paginasi** — laporan diam-diam kehilangan baris ke-501 ke atas.
> Dua RD tanpa batas dan satu berbatas 50.000 adalah bom performa. Paginasi keyset di NFR-12/13
> adalah **perubahan perilaku**, bukan pemeliharaan — dan pengguna yang selama ini melihat
> laporan terpotong akan melihat angka berbeda. Itu perubahan bisnis.

### Fitur grid — API minimum yang dibuktikan

| Fitur | Elemen | `true` | Section |
|---|---|---|---|
| **Filter dropdown per kolom** | `pyColumnFilteringDropDown` | 12.898 | **268 / 269** |
| **Sortir per kolom** | `pyColumnSorting` | 10.975 | **267 / 269** |
| **Filter teks per kolom** | `pyColumnFiltering` | 5.193 | **265 / 269** |
| Sortir tingkat grid | `pyGridSorting` | 354 | 187 |
| Expand/collapse baris | `pyShowExpandCollapseColumn` | 45 | 22 |
| Resize kolom | `pyColumnResizing` | **1** | 1 |
| Kategorisasi kolom | `pyColumnCategorize` | **0** | 0 |
| **Hapus baris inline** | `pyGridDeleteActivityExists` | **0** (5.748× `false`) | **0** |
| **Tambah baris inline** | `pyGridAppendActivityExists` | **0** | **0** |
| Grid RD-driven | `pyGridRDName` non-kosong | 922 | 77 |
| Ekspor CSV | `pxConvertResultsToCSV` | **67 activity**, 95 step | — |
| Ekspor Excel | `ExportToExcel` | 1 section + 1 activity | — |

> **API minimum komponen tabel baku (dibuktikan):** sortir per kolom · filter teks per kolom ·
> filter dropdown per kolom · paginasi · ekspor CSV.
> **BUKAN bagian dari API:** tambah/hapus baris inline (**0 dari 5.748**), resize kolom (1),
> kategorisasi (0). Ini menghemat pekerjaan U-2 secara nyata — tiga fitur yang biasanya mahal
> ternyata tidak dipakai sama sekali di 269 section.

## 10.8 S-5 Jejak Audit — modul baru, dan buktinya lebih kuat dari klaimnya

### Apa yang sistem lama sudah catat

| Mekanisme | Apa yang dicatat | Bukti |
|---|---|---|
| Step `History-Add` | status/penugasan/lampiran | **22 step di 15 activity, SEMUANYA activity OOTB Pega**: `Activity/Save-Act.xml`, `UpdateStatus-Act.xml`, `AttachToWork-Act.xml`, `DeleteAttachment-Act.xml:4801`, `AddToCover-Act.xml:449`, dst. |
| `POOLDATA.PNC_CHRONOLOGYTAT` | posisi kerja, user, `TIMEIN`, `TIMEOUT`, `AGING`, `NOKLAIM`, `NOTE` — **TAT, bukan nilai** | `RDB List/CallProccedureInsertMitra-SQL.xml:61` |
| `POOLDATA.CLAIM_SERVICE_LOG` | `SERVICEID`, `JSONIN`, `JSONOUT` — payload integrasi | `RDB List/QueryLogServiceClaim-SQL.xml:58` |
| `POOLDATA.DATA_ATTACHFILE_HISTORIKLAIM` | riwayat **dokumen** | `RDB List/InsertDokumentHistoriKlaimPNC-SQL.xml:55`, `:71` |
| `POOLDATA.JSON_KLAIM_LOG` · `T_LOG_GLOBALPROTECTION` · `T_LOG_BIGQUERY` · `T_WHATSAPP_LOG` · `T_LOGINCOAS` | log per domain | `RDB List/INSERTLOGGLOBALPROTECTION-SQL.xml:81` dst. |
| Procedure `POOLDATA.PEGA_JSON_INSERT_HISTORY_CLAIM_PNC` | **isi tidak diketahui** — source tidak ada (R-01) | `RDB List/InsertHistoryClaimPNC-SQL.xml:69` |

**Nol di seluruh 902 activity:** `pxHistory` · `Data-WorkHistory` · `pxAuditNote` · `pyAuditNote` ·
`pyHistoryDetails` · `pxTrackSecurityChanges` · `pyAddHistory` · `pxHistoryClass`.

### Apa yang TIDAK dicatat — perubahan nilai uang klaim

Terbukti **dengan ketidakhadiran**, tabel per tabel:

| Tabel nilai uang | Rujukan SQL | Tabel history pasangannya |
|---|---|---|
| `POOLDATA.T_CLAIM_ADJUSTMENT` (nilai akseptasi) | 44 | **TIDAK ADA** di 177 tabel |
| `POOLDATA.T_CLAIM_KOMITE_LIST` (`NILAIKLAIM`, `SHAREASM`) | 33 | **TIDAK ADA** |
| `POOLDATA.T_CLAIM_ESTIMASI` | 9 | **TIDAK ADA** |
| `POOLDATA.OS_AKSEPTASI_KLAIM` | 8 | **TIDAK ADA** |
| `POOLDATA.PNC_SALVAGE` / `DETAIL_PNC_SALVAGE` | — | **TIDAK ADA** |

Sembilan activity pengubah nilai uang, **tidak satu pun memanggil `History-Add`**:
`InsertAdjustmentList` (3× `RDB-SAVE` di `:9700`, `:10008`, `:10181`) · `InsertAdjustmentListKredit` ·
`InsertAdjustmentList_Kredit_PA` · `InsertAdjustmentList_AutoClaim` · `ProteksiTotalKlaimEstimasi` ·
`OsAkseptasiKlaimKredit` · `InputRegister_act` · `KomitePost_Adjustment` ·
`RDB List/SaveRemarksRecommendation_sql-SQL.xml:56` (`update POOLDATA.T_CLAIM_PNC`).

> **Perubahan nilai estimasi, akseptasi, dan pembayaran TIDAK punya jejak audit nilai di sistem
> lama.** `History-Add` hanya dipakai 15 activity bawaan Pega (status & lampiran). Satu-satunya
> kandidat adalah procedure `PEGA_JSON_INSERT_HISTORY_CLAIM_PNC` yang **source-nya tidak ada** —
> jadi apakah ia mencatat nilai lama vs baru **TIDAK DAPAT DIPASTIKAN DARI EXPORT** (blocker R-01).
>
> Ini **menguatkan D-28**: S-5 memang modul baru, dan seluruh pencatatan nilai bisnis dibangun
> dari nol.

### Jejak yang ada BUKAN append-only — dua anti-pola konkret

| Operasi | Tabel log | `berkas:baris` |
|---|---|---|
| **`UPDATE`** | `pooldata.claim_service_log` — `set jsonout = {…} where id = {…}` | `RDB List/UpdateLogServiceClaim-SQL.xml:27` |
| **`DELETE`** | `POOLDATA.JSON_KLAIM_LOG` — `delete … where a.idpega={…} and a.tgl_input=…` | `RDB List/InsertClaimPNC-SQL.xml:77` |
| `UPDATE` | `T_LOGINCOAS` — `SET LOGINAPLIKASI=…, PASSWORD=…` | `RDB List/BrowseOldEmailCoas-SQL.xml:69` |
| `UPDATE` | `pooldata.mst_login_surveyor` | `RDB List/UpdateMasterLoginSurvey-SQL.xml:9` |
| Step `RDB-DELETE` · `OBJ-DELETE` | 12 step di 7 activity · 2 step di 2 activity | — |

Tidak ada satu pun mekanisme pencegah mutasi (trigger, constraint, kolom versi) yang terlihat.

> Yang bisa **dipakai ulang** dari sistem lama hanya *pola* `PNC_CHRONOLOGYTAT` (append per
> perpindahan posisi) dan *daftar* peristiwa yang sudah dicatat. `claim_service_log` dan
> `json_klaim_log` adalah **anti-pola yang harus disebut eksplisit di tiket S-5** sebagai hal yang
> tidak boleh diulang.

## 10.9 Pertanyaan yang hanya bisa dijawab work owner

**F-1**
1. Ada Dynamic System Setting di produksi selain `ServiceFromTable`? Bila tidak, bolehkah daftar
   setting F-1 kami tetapkan dari nol berdasarkan hardcode yang sudah ditemukan?
2. Literal host+path di `Activity/GetLinkAppClaim-Act.xml:765` dipakai **52 berkas** — nilainya
   berbeda antar lingkungan, atau satu nilai?
3. **720 dari 902 activity tanpa penanganan error apa pun.** Kegagalan diam-diam ini pernah
   menimbulkan insiden? Jawabannya menentukan apakah sistem baru boleh **mengubah perilaku**
   (gagal keras + pesan) atau harus meniru kegagalan diam-diam demi P-5.
4. Bagaimana ketersediaan aplikasi dimonitor sekarang, dan siapa penerima alarmnya?

**F-2**
5. Enam skema yang hanya dijangkau DB Link akan diganti API (D-25) atau tetap diakses langsung?
   Menentukan apakah 9 atau 3 skema masuk migration script.
6. Selisih **68 objek** antara 177 (export) dan 245 (dokumen) — angka 245 dari inventaris DBA?
   Mohon daftarnya.
7. `T_CLAIM_PNC`, `T_CLAIM_ADJUSTMENT`, `T_CLAIM_ESTIMASI`, `T_CLAIM_KOMITE_LIST` ditulis dari
   beberapa activity tanpa commit tunggal. **Pernah ada data setengah-tersimpan yang diperbaiki
   manual?** Kasus nyata akan menjadi test case transaksi F-2.
8. **538 pemakaian `{ASIS:}`** — sebagian menyuntik fragmen `WHERE` utuh dan bahkan daftar kolom
   `SELECT`. Apa fungsi bisnisnya (filter lanjutan dari UI)? Kami perlu himpunan filter yang sah
   untuk merancang query builder yang aman.

**F-3**
9. Pemetaan 22 access group → 51 item menu hanya ada di **34 When rule** (6 di antaranya hilang).
   Masih akurat? Ada aturan yang diberlakukan manual di luar rule?
10. `POOLDATA.T_ACCESS_GROUP_PNC` hanya memetakan `OPERATOR_ID` → `OLD_OPERATOR_ID`. Apa fungsinya
    sebenarnya, dan masih dipakai?
11. **901 dari 902 activity tanpa pemeriksaan izin** — pernah ada temuan audit/pentest soal ini?
    Bila ada, temuan itu harus jadi acceptance criteria F-3.
12. **7 harness portal dirujuk menu tapi tidak ada di export.** Masih aktif di produksi? Bila ya
    mohon export tambahan; bila tidak, 7 rute SPA bisa dihapus dari lingkup.

**F-4**
13. Export memuat **≥29 kelompok master**, bukan 14. Mana yang masuk lingkup? (Bengkel/Sparepart/
    Supplier sudah Anda putuskan di luar scope — konfirmasi bahwa 15 kelompok tambahan lainnya
    juga perlu diputuskan satu per satu.)
14. Ambang Komite sudah berupa data (`EMAILKOMITE.LIMIT_BOTTOM` + `DEGREE` + `TYPE_KOMITE`).
    Struktur ini dipertahankan? Dan siapa yang berhak mengubah `LIMIT_BOTTOM` sekarang?
15. **Ambang Large Losses**: tidak ada tabel maupun literal ambangnya. Fiturnya memang belum ada,
    atau logikanya di dalam procedure yang tidak diekspor?
16. Batas tanggal 7/30/90 saat ini **sama untuk semua lini bisnis**. Dokumen menyebut "per lini
    bisnis" — itu kebutuhan **baru**? Bila ya, apa matriksnya?
17. Alur persetujuan master (Approval → Approve/Reject) muncul di Master Rekening, Bengkel HE,
    Sparepart HE, Panel HE. Berlaku untuk **semua** master di sistem baru, atau hanya tertentu?
    Ini keputusan desain besar untuk F-4/U-6.

**F-5**
18. **`addCalendar(...,12,0,0)` — 101 titik di 4 activity.** Apa maksud +12 jam? Tidak bisa
    dimigrasikan tanpa tahu maksudnya.
19. Literal **`"WIB"` di 25 tempat** berdampingan dengan `"Asia/Jakarta"` di 137 tempat. Kedua
    jalur menghasilkan tanggal yang sama di produksi? Bila tidak, **ada bug tanggal aktif
    sekarang**.
20. **Nol konversi −7 jam.** Ada nilai tanggal produksi yang sudah tergeser ganda (+14 jam)?
    Menentukan apakah F-5 butuh skrip perbaikan data, bukan hanya kode baru.
21. Tanggal disimpan dalam 3 representasi. Untuk `DateOfLoss` dan `ReportDate` — mana yang
    otoritatif saat keduanya berbeda?

**U-1 / U-2**
22. Grid **39 kolom** (`Section/Sec_SegmentF06-Section.xml:6509`) dan 25 kolom
    (`Section/PNCStudyClaim-Section.xml:13343`) — semua kolom benar-benar dilihat pengguna, atau
    warisan? Bila bisa dikurangi, komponen tabel baku jadi jauh lebih sederhana.
23. **54 dari 56 laporan memotong hasil di 500 baris.** Pengguna tahu laporannya terpotong? Bila
    tidak, sistem baru yang menampilkan semua baris **mengubah angka yang mereka lihat** — itu
    perubahan bisnis, bukan teknis.
24. Grid lama tidak punya tambah/hapus baris inline (**0 dari 5.748**). Pengguna menginginkannya
    di sistem baru, atau pola "buka form terpisah" dipertahankan?
25. Dari 38 harness tanpa entri menu, mana layar berdiri sendiri (butuh rute SPA) dan mana
    modal/panel? Export tidak membedakannya (`pyHarnessPurpose` tidak ada).

**S-5**
26. Procedure `PEGA_JSON_INSERT_HISTORY_CLAIM_PNC` mencatat **nilai lama vs baru**, atau hanya
    snapshot? Satu-satunya kandidat jejak audit nilai, dan source-nya tidak ada.
27. `CLAIM_SERVICE_LOG` di-`UPDATE` dan `JSON_KLAIM_LOG` di-`DELETE`. Ada kewajiban retensi/
    regulasi (OJK) atas log ini? **Bila ada, mutasi ini sudah melanggar sekarang.**
28. Peristiwa apa saja yang **wajib** masuk jejak audit dari sisi kepatuhan? Sistem lama hanya
    mencatat status, penugasan, dokumen, dan TAT — perubahan nilai tidak tercatat sama sekali.
    Daftar peristiwa wajib harus datang dari pemilik proses, bukan dari export.

---

# 11. Kontradiksi terhadap Steering dan BRD — terkonsolidasi

Setiap baris menyandingkan rujukan dokumen dengan rujukan bukti. **Tidak ada yang saya ubah
sendiri** — seluruhnya diajukan sebagai usulan revisi yang menunggu persetujuan work owner.

| # | Dokumen | Bukti | Sifat |
|---|---|---|---|
| K-1 | `19-GAP` R-07: 45 activity buatan sendiri hilang | **40**; 5 positif palsu, semuanya bawaan Pega | koreksi angka |
| K-2 | `19-GAP` R-07: 5 activity bawaan Pega | **6** (`pyConvertToJavaMap`) | koreksi angka |
| K-3 | `19-GAP:154`: workbasket `komitepnc1..4` | **`komitepnc`** (tanpa angka) `2`/`3`/`4`, dan **worklist**, bukan workbasket | koreksi fakta |
| K-4 | `19-GAP:156`: workbasket `ReceiveDocument.UserAdmin` | properti operator, bukan workbasket | koreksi fakta |
| K-5 | `19-GAP` ruang lingkup: 3 kategori | **7 tipe rule tidak diaudit**; gap ±242 vs 43 | **perluasan lingkup** |
| K-6 | `20-DETAIL:310`: 64 pemakaian DB Link, 20 objek `@ASMD` | **71 pemakaian, 22 objek** | koreksi cakupan |
| K-7 | D-18: empat konsep status dipertahankan | `ClaimStatus` milik **GISFW**; `StatusPosisi` bukan properti & satu nilai | **revisi keputusan** |
| K-8 | `BRD.md:453` / `FR-F4`: 10 email, 4 user ID, 3 ambang | **66 / 24 / 8** untuk seluruh export | koreksi angka |
| K-9 | `STEERING.md`: host produksi utama sbg 1 dari 3 hostname penentu perilaku | host itu **tidak pernah dibandingkan**; yang benar host entitas Insurtech | koreksi daftar |
| K-10 | `STEERING.md:354` / D-15: step 103/107 dan 106–109 | sub-step 2 dan 6 **di dalam step 53** | koreksi rujukan |
| K-11 | `BRD.md:610`: DOL s/d 30 hari setelah polis berakhir (Bonding) | step 21 tanpa guard `bukan IsBonding` → **aturan tidak dapat berlaku** | **cacat berjalan atau BRD salah** |
| K-12 | `BRD.md:631`: total spreading 100%, toleransi 99,99% | **pencocokan substring**; `199.99` dan `1100.0` lolos | **cacat berjalan** |
| K-13 | `BRD.md:621`: kunci duplikasi = Polis + Objek + Lokasi | juga **DOL**; dan lokasi hanya untuk non-PA/non-Travel | koreksi aturan |
| K-14 | `BRD.md:622`: PA = Polis + Objek + `12002` + Lokasi | SQL PA: `policyno` + `objectid` + `coverageid='10009'`, **tanpa lokasi & tanpa cause of loss** | koreksi aturan |
| K-15 | `BRD.md:633`: Group Panel `003` Fac Offer wajib Object Name | hanya baris **(1)(1)** diperiksa | koreksi cakupan |
| K-16 | `BRD.md:640`: ambang PA/Travel Rp 30 juta | dipicu **jabatan operator**, bukan lini bisnis klaim | koreksi aturan |
| K-17 | `BRD.md:657`: nilai klaim ≤ TSI | benar, **tetapi PA dikecualikan** — tidak ada di BRD | aturan hilang dari BRD |
| K-18 | `FR-U3`: ~20 inbox berbasis peran | **26** entri navigasi | koreksi angka |
| K-19 | `FR-W2`: 11 ticket | 11 = custom di Register Flow saja; seluruh aplikasi **17 dirujuk, 8 ada** | koreksi definisi |
| K-20 | D-16 / `FR-S1`: tiga mekanisme penyimpanan dokumen | **empat** — GCS mekanisme keempat, rule-nya hilang | penambahan |
| K-21 | D-07: autentikasi HCC/HCQ | **nol jejak di export** → integrasi baru, bukan migrasi; gerbang 1 tidak berlaku untuk F-3 | **revisi premis** |
| K-22 | D-15: konfigurasi tiga lapis | **nol DSS**; pola lama dikunci per IP aplikasi, bertabrakan dengan D-27 | **revisi premis** |
| K-23 | D-26: model penugasan dipertahankan | penguncian **nol kustomisasi** → sistem baru bebas memilih model | pelonggaran |
| K-24 | `16-RISK-ANALYSIS.md`: R-01…R-12 | BRD §20 + `migration-readiness` punya **R-13, R-14, R-15** | Steering belum menyerap |
| K-25 | `07-MIGRATION-STRATEGY.md`: P-1…P-5 | `requirement-summary.md:27-28` punya **P-6, P-7** | Steering belum menyerap |
| K-26 | `understanding-log.md:69` / `BRD.md:71,1290,1478`: 27 modul | enumerasinya sendiri **32**; `FR-S8` tanpa baris modul → 33 | koreksi angka |
| K-27 | `Steering/README.md:90`: 12 risiko | BRD: **15** | Steering belum menyerap |
| K-28 | `19-GAP` tidak mencatatnya | **jalur validasi API/JSON lebih longgar** — tanpa aturan 7/30/90 | temuan baru |
| K-29 | `BRD §21.2` #11 melarang perangkaian SQL | `Activity/InputRegister_act-Act.xml:7750` merangkai SQL dari `ClaimData.Location`; ditambah **538 `{ASIS:}` di 233 dari 652** rule SQL, 17 di dalam `LIKE '%…%'` | cacat berjalan, **skala jauh lebih besar** |
| K-30 | "245 tabel di 5+ skema" | **9 skema**; **177** objek (126 di SQL + 78 kelas `Int-`, dedup). Selisih ±68 diduga hanya disentuh 64 procedure (R-01) | koreksi angka |
| K-31 | `requirement-summary.md:162` / `09-DATABASE-STRATEGY.md:151` / `15-NFR-…md:80`: **`OFFSET 500000`** memaksa DB membaca setengah juta baris | **`OFFSET` = 0 kemunculan** di seluruh export; `500000` hanya muncul sebagai bagian `5000000000` (`RDB List/BrowseClaimStudy-SQL.xml:93`) | **klaim tidak berdasar** — NFR-13 harus ditulis ulang |
| K-32 | `06-MODULE-BREAKDOWN.md:68`: tabel baku **18–27 kolom** | 375 grid: min 1, **median 6**, p75 9, maks 39. Hanya **3 grid (0,8%)** di rentang 18–27 | **spesifikasi U-2 salah sasaran** |
| K-33 | `requirement-summary.md:118`: **47 menu** portal | **47 harness target unik** ✔ tetapi **51 item menu aksi**; dan **11 dari 47 harness tidak ada di export** (7 layar proyek + 4 OOTB) | angka benar, definisi beda; **gap baru** |
| K-34 | Paginasi server-side tersirat di U-2 | **19 dari 652** rule SQL punya paginasi DB sejati; **3.189** grid terikat page list klipboard; `pyListLoadMode=auto` **2.331×** vs `lazyload` **1×**; **54 dari 56** RD memotong di `pyMaxRecords=500` | sistem lama praktis **client-side**; NFR-12/13 adalah **perubahan perilaku** |
| K-35 | `05-DOMAIN-MODEL.md:184`: Ambang Komite menggantikan hardcode `50000000`/`30000000`/`3500` | Matriks jenjang × bisnis **sudah berupa master data** di `EMAILKOMITE.LIMIT_BOTTOM` + `DEGREE`. Yang hardcode: Ex-Gratia `250000000` dan open protection `50000000000` (`Activity/CheckEstimateValue-Act.xml:9617`) | koreksi fakta |
| K-36 | `06-MODULE-BREAKDOWN.md:20`, `:72` / `05-DOMAIN-MODEL.md:176-189`: **14 master**, F-4 "Sedang–Besar", U-6 "Sedang" | **≥29 kelompok master** dengan tabel + layar; **49 section `Master*`**, **11 harness `*Master*`** | **F-4 dan U-6 undersized** |
| K-37 | `06-MODULE-BREAKDOWN.md:21`: F-5 **"Kecil"** | **118 titik +7 jam di 36 activity** + **101 titik +12 jam** di 4 activity + 4 definisi "sekarang" + 6 mask tanggal + 3 representasi penyimpanan + **0 konversi balik** | "Kecil" hanya benar untuk seam Clock, bukan untuk migrasi 118 titik |
| K-38 | D-20 / `08-TECHNICAL-STRATEGY.md:231` / `09-DATABASE-STRATEGY.md:83`: `ROWNUM` → `OFFSET … FETCH NEXT`, *"pola sudah dipakai di 35 rule"* | 35 rule ✔ tetapi **mayoritas `FETCH NEXT 1 ROW ONLY`** (ambil satu baris teratas, bukan paginasi); `OFFSET` **0×** | angka benar, **interpretasi salah** — klaim "bukan hal baru bagi tim" tidak didukung |
| K-39 | `09-DATABASE-STRATEGY.md:71`, `:83`: **68** pemakaian `ROWNUM` | **68 kemunculan di 44 berkas** | ✅ **cocok persis** — dapat dipakai apa adanya |
| K-40 | `FR-R1` / `BRD §21.2` #8: otorisasi tidak boleh bergantung penyembunyian menu "seperti sistem lama" | **Terbukti kuat**: `pyPrivilegeName` non-kosong **1 dari 902** activity (`Activity/DownloadFile-Act.xml:314` → `zipMoveExport`, privilege OOTB Pega); 900 kosong; **0** tabel izin menu di DB | ✅ klaim didukung dengan angka |

---

# 12. Kandidat risiko baru — menunggu penomoran dari work owner

Aturan 12 melarang saya menambah ID baru sendiri. Empat kandidat berikut diajukan beserta
alasannya:

| Kandidat | Isi | Kenapa layak jadi risiko sendiri |
|---|---|---|
| A | **Export tidak lengkap: ±242 rule dirujuk tapi tidak ada**, terberat 116 When rule buatan sendiri | R-01 hanya mencakup objek database; R-07 hanya activity. Ini memblokir **percabangan bisnis di hampir semua modul**, bukan 5 modul |
| B | **Kredensial aktif berada di dalam export** — 3 password SMTP di 31 lokasi + kredensial OAuth | Risiko keamanan, bukan risiko migrasi. Menuntut rotasi dan pengendalian penyimpanan repo |
| C | **Dua integrasi BRI menunjuk host sandbox** di ruleset produksi | Bila produksi memang memanggil sandbox, ini cacat berjalan yang memengaruhi mitra luar |
| D | **Cacat aturan uang yang direplikasi bila P-5 dipatuhi buta**: toleransi spreading substring (K-12), ambang Rp 50 juta tiga operator (§8.4), penyesuaian 7 jam asimetris (§8.1.1) | P-5 mewajibkan hasil identik Pega. Tanpa daftar pengecualian eksplisit, tiket akan menyalin cacat ini ke sistem baru |

---

# 13. Daftar pertanyaan Fase 2 — dikelompokkan per lapis brief

Diturunkan dari bagian "Pertanyaan yang hanya bisa dijawab work owner" di §1–§9, dibuang yang
sudah terjawab di `00-DECISION-LOG.md` atau `docs/interview-history.md`, dan dibuang yang dapat
saya buktikan sendiri.

| Lapis | Jumlah pertanyaan | Sumber |
|---|---|---|
| 1 — status keputusan terbuka & risiko | tanggal komitmen + jalur eskalasi per pihak (lanjutan I-05, bukan ulangan); status 4 kandidat risiko baru (§12) | §12, `interview-history.md:202` |
| 2 — cakupan tiket, pelaksana, ukuran | — | brief |
| 3 — angka untuk acceptance criteria | §8.7 (15 pertanyaan), §7.7 #4–#5 | §7, §8 |
| 4 — uji kesetaraan & 5 modul tak terukur | pengecualian gerbang 1 untuk F-3 (§1.1); daftar cacat yang boleh/tidak boleh direplikasi (§12 kandidat D) | §1.1, §12 |
| 5 — peran, kewenangan, UAT | §6.6 (7 pertanyaan), §5.5 #3 | §5, §6 |
| 6 — operasional, data, compliance | §1.4 #2–#6, §3.1, §2.3 | §1, §2, §3 |
| 7 — jadwal | — | brief |
| 8 — yang rawan & perlu konfirmasi framing | §7.7 #2–#3, #7; §12 kandidat B & C | §7, §12 |

---

**Akhir Fase 1.** Sepuluh area lengkap, menunggu GATE 1.

## Batas metode yang perlu diketahui

1. **Inventaris rule dibangun dari `<pyRuleName>`, bukan nama berkas** (§9.3) — 2 berkas berisi
   rule yang bukan namanya.
2. **Jumlah kolom grid** diukur dari `pyGridColumnProps` (197 dari 269 section). Untuk 71 section
   bergrid yang memakai repeat layout tanpa metadata kolom, jumlah kolomnya
   **TIDAK DAPAT DIPASTIKAN DARI EXPORT** dengan metode ini.
3. **Daftar tabel** diambil dari token sesudah `FROM|JOIN|INSERT INTO|UPDATE` dan dari nama kelas
   `ASM-FW-*FW-Int-<TABEL>`. Tabel yang **hanya** disentuh 64 procedure tidak terhitung.
4. **Tipe kolom database** (`DATE` vs `TIMESTAMP` vs `VARCHAR`) tidak ada di export — hanya
   disimpulkan dari mask `TO_CHAR`/`TO_DATE`. Butuh DDL (R-08).
5. **Klasifikasi bawaan Pega versus buatan sendiri** untuk rule yang **hilang** bersandar pada
   konvensi nama (`px`/`py`/`pz`), bukan bukti ruleset — karena ruleset rule yang hilang tidak
   terbaca.
6. **Jumlah Data Transform yang hilang (18) adalah batas bawah** — 65 step `Apply-DataTransform`
   punya `<pyParamArray>` kosong, sehingga nama DT yang dipanggil tidak selalu terbaca.

## Yang TIDAK dilakukan, sesuai aturan tugas

- Tidak ada berkas rule XML yang diubah, dipindahkan, direname, atau dihapus.
- Tidak ada koneksi database, tidak ada DDL/DML, tidak ada akses produksi.
- `docs/STEERING.md` dan kedua berkas `.docx` tidak disentuh.
- `docs/Steering/00-DECISION-LOG.md` tidak diubah — entri Sesi 3 ditulis pada Fase 2.
- Tidak ada satu baris kode implementasi yang ditulis.
- Seluruh koreksi terhadap Steering/BRD di §11 adalah **usulan yang menunggu persetujuan work
  owner**, bukan perubahan yang sudah dieksekusi.
- Tidak ada ID baru (`R-nn`, `FR-xx`, modul) yang saya buat sendiri — kandidat diajukan di §12.

---

# 14. Fase 1-lanjutan — snapshot 2.389 rule dan folder `Database/`

Ditambahkan **2026-09-09** atas keputusan work owner (**D-41**, Q11 opsi 1: jalankan ulang audit
ketersediaan dan area yang paling terdampak).

Bagian §0–§13 di atas dibangun atas snapshot **2.167 rule** tanggal 2026-09-08. Sejak itu export
bertambah dan folder `Database/` muncul. Bagian ini mencatat **selisihnya** dan **mengoreksi
kesalahan saya sendiri** — §0–§13 dibiarkan apa adanya sebagai catatan keadaan saat itu, kecuali
di tempat yang dirujuk balik dari sini.

## 14.1 Perubahan snapshot

| Folder | 2026-09-08 | 2026-09-09 | Δ |
|---|---|---|---|
| `Activity/` | 902 | **974** | +72 |
| `RDB List/` | 652 | **776** | +124 |
| `Connect REST/` | 12 | **21** | +9 |
| `HTML/` | 2 | **16** | +14 |
| `When/` | 70 | **72** | +2 |
| `Report Definition/` | 56 | **57** | +1 |
| **Total rule XML** | **2.167** | **2.389** | **+222** |
| Nama rule Activity unik | 899 | **969** | +70 |
| **`Database/`** | — | **55 `.prc` + 8 `.fnc` + 2 `.csv`** | baru |

> Ini **R-09 termanifestasi** (*"sistem sumber masih aktif berubah"*). Setiap angka di §0–§13
> berlaku untuk snapshot 2.167 dan harus dibaca dengan tanggalnya.

## 14.2 Enam koreksi atas laporan saya sendiri

Diverifikasi ulang oleh orkestrator, bukan hanya dilaporkan sub-agen.

### K-20 DICABUT — mekanismenya **tiga**, bukan empat

§3 dan `K-20` menyatakan Google Cloud Storage adalah **mekanisme penyimpanan dokumen keempat**
yang tidak tercatat di D-16. **Salah.**

`Activity/UploadDocumentToGoogleStorage-Act.xml` **tidak punya satu pun elemen `<pyMethod>`
miliknya sendiri** (diverifikasi: sensus `pyMethod` → nol hasil). Ia mendelegasikan lewat
`Call InsertDokumenPNC` (`:1849`, deskripsi step *"Insert to google storage"* `:1850`) — yaitu
**activity yang sama** dengan mekanisme #2 (API Storage internal), yang mengunggah lewat
`Connect-REST` di `Activity/InsertDokumenPNC-Act.xml:4070-4082`.

Nama "GoogleStorage" hanyalah nama pembungkus. **D-16 dan `FR-S1` benar: tiga mekanisme.**
Pertanyaan terbuka §3.1 #2 terjawab: bukan mekanisme terpisah.

Tapi ada koreksi balik: §3 menulis *"hanya satu call site"*. Sebenarnya **tiga** —
`Activity/SetStsSalvagePNC_act-Act.xml:2443`, `:2448`, `:2453`. Kesimpulan "rollout parsial
ber-flag" tetap berlaku karena precondition `ViewGoogleStorage.Storage==true` (`:1875`), dan
**penulis flag itu tetap tidak ditemukan** di seluruh repo (hanya 2 pembacaan).

### `SetTicket` adalah rule bawaan Pega, bukan rule kustom yang hilang

§5.2 menyatakan *"activity `SetTicket` sendiri tidak ada di export — jadi pemicu maupun
definisinya hilang"*, dan `19-GAP` memprioritaskannya sebagai rule yang *"berisi logika bisnis"*.
**Keduanya salah.**

`Activity/SetTicket-Act.xml` — `pyRuleSet=`**`Pega-ProcessEngine`** (diverifikasi),
`pyClassName=@baseclass`, `pyRuleAvailable=Final`, `pyUsage=Called from flows.`

Ia **nol logika bisnis ASM**. Empat step-nya: satu `Java` penanganan embedded page,
`Obj-Set-Tickets` (set), `Obj-Set-Tickets` (remove), `Obj-Save`. Nama ticket masuk lewat
parameter tunggal `Ticket`, di-hardcode oleh pemanggil.

Jadi ia tidak perlu diminta ke Tim Pega — logika bisnisnya ada di **4 pemanggil**, yang seluruhnya
**sudah** ada di export sejak awal.

### R-06: tiga dari sembilan kesimpulan saya salah

§4.4 menyimpulkan arti 9 kode `StatusClaim` dari rule-anchor, dan menyatakan R-06 *"dapat
diturunkan dari memblokir seluruh modul berstatus menjadi melengkapi label"*. Master yang asli
(`Database/v_sts_claim.csv`) membuktikan **saya salah 3 dari 9**:

| Kode | Kesimpulan §4.4 | Label sebenarnya | |
|---|---|---|---|
| `1142` | keputusan Komite Reject | **Rejected Claim** | ✅ |
| `1143` | status awal saat case dibuat | **Close Claim for this object** | ❌ |
| `1144` | jalur reject/penutupan | **Cancelled Claim** | ✅ |
| `1145` | posting hasil survey Komite | **Waiting Survey** | ✅ |
| `1146` | input register / display polis | **View Polis** | ✅ |
| `1147` | wrapper pemanggil InputRegister | **Register** | ✅ |
| `1148` | tidak pernah di-set | **CFS Report** — ada di master, memang tidak di-set | ✅ |
| `1149` | pembentukan daftar komite | **Claim Committee** | ✅ |
| `1150` | penanda "sudah teregister" | **LOD Report** | ❌ |
| `1151` | penetapan status Investigator | **Analyst** (Investigator = `1156`) | ❌ |

**Dan domainnya bukan `1142`–`1151`, melainkan `1134`–`1166` — 33 kode.** Rentang yang dipakai
D-18, R-06, dan seluruh dokumen hilir adalah **subset**.

Konsekuensi metode: menyimpulkan arti kode dari konteks pemakaian menghasilkan **67% akurasi**.
Untuk state machine klaim itu tidak cukup. Keputusan menuntut master dari DBA terbukti benar.

Daftar lengkap 33 kode ada di `Database/v_sts_claim.csv` (kolom `LSC_ID`, `OLD_LSC_ID`,
`LSC_NOTE`). Sebelas kode pertama (`1134`–`1144`) punya `OLD_LSC_ID` (`01`–`11`) — bukti migrasi
penomoran lama; sisanya kosong.

### `IsGCNMUser` ada, dan hanya 5 When rule peran yang hilang

§6.5 dan §9.1 menulis `IsGCNMUser` hilang dan *"mengendalikan 11 entri navigasi"*.
`When/IsGCNMUser-When.xml` **ada** (diverifikasi), dirujuk 13× di navigasi.

Yang benar-benar hilang dari 31 When rule yang dipakai navigasi: **5 rule proyek**, masing-masing
mengendalikan **1** entri menu — `IsGCNMReport`, `IsKomite`, `IsNotViewClaim`, `IsPNCBonding`,
`IsSurvey` — plus 2 bawaan Pega (`pyIsMobilePhone`, `pzIsPegaContainer`).

**24 When rule navigasi lainnya ada** dan dapat dibaca untuk merekonstruksi pemetaan peran → menu.
Dampak ke F-3 jauh lebih kecil dari yang saya laporkan.

### R-01 belum tertutup — 62 berkas datang, 12 dependensi tidak

Saya melaporkan *"R-01 hampir tertutup — 62 dari 64"*. Benar di tingkat berkas, **menyesatkan di
tingkat kegunaan.** 62 procedure yang datang memanggil 12 objek yang **tidak ikut dikirim**:

| Objek absen | Dipanggil |
|---|---|
| **`UPDATE_LOG_KONVERSI`** | **162×** |
| `GETNEWID` | 42× |
| `PKG_COUNTER_PRODUCTION` | 25× |
| `PROCESSQUEUEDIRECT` | 10× |
| `INSERTNEWT_VEHICLELIST` · `CLOBTOBLOB` · `CONVERT_PEGA_DATE` · `INSERTT_COVERAGELISTPERSON` · `JSON_MBU` · `JSON_FIRE` · `CEK_DIGIT_POLIS_2` · `COUNT_OUTGO_PROD` | 1–5× |

Ditambah `JSON_POLIS_PA`, `JSON_POLIS_TRAVEL`, `JSON_POLIS_MARINE_CARGO`, `json_aneka`,
`weekends2`, `POOLDATA.datediff`, `POOLDATA.base64decode`, `new_uuid`, `select_sequence`,
`jsonObjectWriter`, `NumberAddSeparator`, dan definisi 4 queue `DBMS_AQ`.

Dan **dua tabel master yang menentukan angka uang tetap tidak ada**:
`POOLDATA.GCNM_FEE_SCALE` (17 pita fee adjuster) dan `m_currencystandard` (daftar kurs).

**Status R-01 yang benar: 62/64 berkas, rantai dependensi belum lengkap, dua tabel master belum
ada.** Yang belum datang dari 64: `SET_ATTACHFILETEMPSALVAGE`, `UPDATEPREMIUMTEMPLATE`. Satu bonus
di luar daftar: `TEMP_SET_ATTACHMENT_64BIT`.

### `ValidasiSisaTSI` — parameter bukan in/out

§8.5.1 menulis parameter `ObjectID`/`CoverageID` bersifat `pyParametersParamInOut=true` (in/out).
`Activity/ValidasiSisaTSI-Act.xml:23` mendeklarasikan keduanya **`INOUT="IN"`**, dan nol penulisan
`Param.*` di seluruh activity. Atribut yang saya lihat di sisi pemanggil adalah metadata kolom
form, bukan arah parameter. Kontraknya **validate-only** — satu-satunya efek keluar adalah pesan
validasi Pega.

## 14.3 Yang sudah datang, dan yang masih hilang

| Artefak | Status | Catatan |
|---|---|---|
| `ValidasiSisaTSI` | ✅ **datang** | aturan penuh terbaca — §14.4 |
| `SetTicket` | ✅ datang | bawaan Pega (§14.2) |
| `UploadDocumentToGoogleStorage` | ✅ datang | pembungkus, bukan mekanisme baru (§14.2) |
| `TransferToKasir_act_Leader` · `SetKasir_Act` · `generatePD4ML` · `getRandomTeam_act` · `PNCSalvageHistorySemuaKlaim` | ✅ datang | R-07 menyusut |
| **`SendEmailNotification`** | ❌ **masih hilang** | dipanggil 15×, pemblokir **S-3** |
| **`PNCAdminRouter` · `PNCTeknikRouter` · `RouterRCLDokter`** | ❌ **masih hilang** | **R-04 tidak bergerak** — tapi lihat §14.5 |
| **`BrowseT_Claim_Adjustment_SQL`** | ❌ **hilang** | query dasar `ValidasiSisaTSI` — lubang terpenting yang tersisa |
| 5 When rule peran | ❌ masih hilang | §14.2 |
| 7 harness portal | ❌ masih hilang | daftar di §14.6 |
| Library `GCNM` dan `CNM` | ❌ masih hilang | `@GCNM.GetPageJSONString` **74×**; `Function/` masih 1 berkas |
| **HCC/HCQ** | ❌ masih nol jejak | 9 Connect REST baru pun bukan HCC/HCQ |
| Access Group / Role / Privilege / Operator | ❌ masih tidak ada foldernya | pemblokir **F-3** |
| Perbaikan 2 berkas salah-isi | ❌ belum | `SendEmailNotification-Act.xml` **masih** berisi `CompressImage_Act` |

## 14.4 `ValidasiSisaTSI` — aturan penuh, dan satu bug yang harus diputuskan

`BRD §11.5` menyebut rule ini *"petunjuk bahwa masih ada aturan bisnis yang belum kita ketahui
keberadaannya"*. Sekarang terbaca — `Activity/ValidasiSisaTSI-Act.xml`, 2.275 baris, 7 step.

**Aturannya:**

> Untuk objek+coverage tertentu, ambil `SumTSI` dari snapshot polis
> (`ObjectList(ObjectID).ObjectCoverageList(CoverageID).SumTSI`, `:297`). Kurangi dengan jumlah
> `Value` semua baris **outstanding akseptasi** yang `PaymentType != "3"` (`:936`, `:1293`), lalu
> **tambahkan** jumlah `Value` semua baris yang `PaymentType == "3"` (salvage) (`:1073`, `:1492`).
> Hasilnya sisa TSI. Bila `ProposeAdjustmentValue` **lebih besar** (`>`, bukan `>=`) dari sisa TSI
> (`:1640`), tolak dengan pesan pada field `.ProposeAdjustmentValue`.

Teks pesannya, bahan AC langsung (`:1597`):
`"Nilai Adjutment tidak boleh lebih besar dari sisa TSI, Sisa SumTSI = " + @NumberAddSeparator(local.TotalAkseptasiKlaim)`
— salah tulis `Adjutment` ada di sumber.

**Dua hal yang mengubah pemahaman:**

1. **Basisnya "outstanding akseptasi", bukan "klaim terbayar".** Kelas hasil SQL
   `ASM-FW-GCNMFW-Data-osAkseptasi` (`:558`), deskripsi step *"Search Os_Akseptasi_klaim"*
   (`:390`). Klaim yang sudah diakseptasi **tapi belum dibayar tetap mengurangi** sisa TSI.

2. **Salvage MEMULIHKAN kapasitas TSI** (`:1492`). Aturan yang tidak tercatat di dokumen mana pun.

**Pembatasan PA bukan sifat rule ini.** Activity berkelas `ASM-FW-GCNMFW-Data-Adjustment` dan
**tidak memuat satu pun cek lini bisnis**. Pembatasan `IsPA` ada di pemanggil
(`Activity/SetNilaiResikoSendiri-Act.xml:9760`). Siapa pun yang memanggilnya tanpa `IsPA`
mendapat validasi yang sama.

**Bug yang harus diputuskan sebelum ditulis ulang.** `local.PaymentType` di-set **di dalam loop**
dari `.User` (`:817-820`), tapi step 4 (`:1254`) dan step 5 (`:1489`) berada **di luar** loop dan
bercabang pada nilai itu. Setelah loop, nilainya adalah `PaymentType` **baris terakhir**.
Akibatnya: **`NilaiSalvage` hanya ditambahkan bila baris terakhir kebetulan bertipe salvage.**
Bila salvage ada di tengah daftar, salvage diabaikan.

Ini bug urutan-dependen, bukan aturan bisnis. Niat kodenya jelas dari `:1492`. Keputusan yang
dibutuhkan: **perbaiki (selalu tambahkan salvage) atau replikasi bug** — dan itu keputusan
bisnis, bukan teknis, karena memperbaikinya membesarkan sisa TSI dan meloloskan adjustment yang
dulu ditolak.

Tiga local variable juga dinamai salah: `local.ClaimID` diisi `.ObjectID` (`:751`),
`local.Nopolis` diisi `.CoverageID` (`:798`), `local.PaymentType` diisi `.User` (`:817`). Dua yang
pertama tidak pernah dibaca lagi — kode mati.

**Lubang yang tersisa:** `BrowseT_Claim_Adjustment_SQL` (`:405`) **tidak ada di export**. Tanpa
query itu, tiga hal tidak dapat ditentukan: apakah ia memfilter per `ObjectID`+`CoverageID`
(activity **tidak** memfilter lagi — loop `:936` menjumlah semua baris tanpa syarat); apakah klaim
reject/void ikut terhitung; dan apa parameter masuknya.

## 14.5 R-04 — algoritma penugasan terbaca meski router-nya masih hilang

Router `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter` **tetap tidak ada**. Tetapi
algoritmanya terbaca dari tempat lain.

`Database/PEGA_MST_USER_TEKNIS.prc:36` mengungkap struktur tabel PIC:
`OPERATOR_ID` · **`COUNTER_QUOTA`** · **`COUNTER_QUOTA2`** · `TYPE_BUSINESS` · `EMAIL` ·
`STS_AKTIF` · **`TEAM_GROUP`** · **`ATASAN`** · `MCL_NAME`.

Dan algoritmanya utuh di SQL — `RDB List/BrowsePICRandomTeam-SQL.xml:39-40`:

```sql
SELECT OPERATOR_ID AS "SearchName", counter_quota
  FROM POOLDATA.MST_USER_TEKNIK
 WHERE sts_aktif = '1' AND type_business = 'NONMBU'
   AND operator_ID != 'ELLENSUPRIYATI'
       {ASIS:TempNoPolis.InvoiceNo}
 ORDER BY counter_quota ASC
```

Penghitungnya dinaikkan setelah penugasan —
`RDB List/AddTJobCounterPIC_SQL-SQL.xml:75` (`counter_quota+1`), `:86` (`counter_quota2+1`);
varian kedua di `RDB List/BrowsePICRandomTeam2-SQL.xml:32` mengurutkan `counter_quota2`.

**Algoritmanya: pilih PIC aktif untuk lini bisnis itu dengan beban terendah, lalu naikkan
bebannya.** Least-loaded, bukan acak.

Tiga hal yang ikut terungkap:
- **Nama rule menyesatkan** — `BrowsePICRandomTeam` sama sekali tidak acak.
- **`operator_ID != 'ELLENSUPRIYATI'` di-hardcode di dalam SQL** — satu orang dikecualikan dari
  penugasan otomatis. User ID hardcode ke-25, lokasinya di teks SQL.
- **`{ASIS:TempNoPolis.InvoiceNo}` menyuntik fragmen `WHERE`** — properti bernama "InvoiceNo"
  membawa potongan SQL.

**R-04 turun dari pemblokir menjadi verifikasi.** B-6 dapat ditiketkan dengan AC berangka, dengan
catatan bahwa yang direkonstruksi adalah algoritmanya; ketiga router tetap perlu diminta untuk
mengonfirmasi bahwa mereka memang memakai jalur ini.

## 14.6 D-14 terjawab — `emailkomite.csv`, dan model penjenjangannya bukan yang diasumsikan

**Tabelnya 21 kolom, bukan 14** seperti disimpulkan `20-DETAIL:33-54` dari query. Enam kolom tidak
pernah terlihat dari SQL: `NAME`, **`LIMIT_TOP`**, `STS_SURVEY`, `STS_SALVAGE`, `STS_ADJUSTER`,
**`LIMIT_TOP_EXGRATIA`**.

**`LIMIT_TOP` mengubah modelnya** — setiap jenjang punya **rentang bawah–atas**, bukan hanya
ambang bawah. Penjenjangan adalah pemetaan rentang.

30 baris, **17 aktif** (`STS_AKTIF="1"`).

**PA — 4 jenjang, rentang rapi.** Inilah ketajaman yang diminta untuk AC:

| DEGREE | Rentang | TYPE_KOMITE |
|---|---|---|
| 1 | Rp 0 – **10.000.000** | 2 |
| 2 | Rp **10.000.001** – 50.000.000 | 1 |
| 3 | Rp **50.000.001** – 100.000.000 | 1 |
| 4 | Rp **100.000.001** – 200.000.000 | 2 |

→ **Rp 10.000.000 → jenjang 1; Rp 10.000.001 → jenjang 2.**

**TRAVEL — 3 jenjang, juga rapi:** `0–50.000.000` · `50.000.001–100.000.000` ·
`100.000.001–200.000.000`. Ex-Gratia punya rentang sendiri.

**NONMBU — rentangnya TUMPANG TINDIH:** D1 `0–100.000.000` (komite 1) · D1 `0–50.000.000`
(komite 2) · D2 `50.000.000–500.000.000` · D3 `500.000.000–1.000.000.000` ·
D4 `1.000.000.000–100.000.000.000`.

**BONDING — satu baris aktif, rentang `0–0`.** Konsisten dengan
`EmailKomiteBerjenjangBonding_sql` yang memfilter tanpa ambang.

### Dan query-nya mengabaikan `LIMIT_TOP`

`RDB List/EmailKomiteBerjenjangSimasnet_sql-SQL.xml:97-100` — diverifikasi:

```sql
SELECT * FROM (
  SELECT EMAIL, DEGREE, OPERATOR_ID, CC FROM POOLDATA.EMAILKOMITE
   WHERE STS_ADJ='1' AND STS_AKTIF='1'
     AND trim(TYPE_BUSINESS) = trim({tempAdj.pyMemo})
     AND LIMIT_BOTTOM <= {tempAdj.ConvertAdjustmentValue}
         {ASIS:tempAdj.UploadLOD}
   ORDER BY degree, dbms_random.value)
  {ASIS:tempAdj.IsDLA}
```

Tiga temuan:
1. **`LIMIT_TOP` tidak ada di predikat.** Setiap baris dengan `LIMIT_BOTTOM ≤ nilai` ikut terpilih.
2. **Pemenangnya ditentukan `dbms_random.value`** — siapa yang menerima permintaan komite
   **tidak deterministik antar eksekusi**. Ini menentukan apakah B-7 dapat lolos gerbang 1 sama
   sekali: uji kesetaraan atas fungsi acak tidak dapat membandingkan hasil.
3. **Dua `{ASIS:}` dengan nama properti menyesatkan** — `tempAdj.UploadLOD` membawa fragmen
   `WHERE`, `tempAdj.IsDLA` membawa pembungkus query luar.

Ini memperbesar §8.4: saya lapor Rp 50.000.000 masuk **dua** jenjang; di jalur `EMAILKOMITE` ia
memenuhi **empat baris** dengan tiga kombinasi `DEGREE`/`TYPE_KOMITE`.

Dan `Database/INSERTDATAKOMITELIST.prc` — yang saya duga memutus ambiguitas ini — **tidak
memutuskan apa pun**: nol pembacaan `EMAILKOMITE`, nol referensi `LIMIT_BOTTOM`/`DEGREE`, dan
`tNILAIKLAIM` hanya **disimpan**, tidak pernah dibandingkan. Ia penulis baris murni.

### Cacat data di master

`OPERATOR_ID` baris ID 4 memuat **baris-baru di dalam nilainya**; baris ID 6 ber-`NAME`
"YOHANES RAYMOND ADIKARTA" tapi `OPERATOR_ID`-nya `ELLENSUPRIYATI`; dua baris `OPERATOR_ID`
kosong; beberapa `EMAIL` memuat banyak alamat dalam satu kolom, satu dengan spasi di depan;
`STS_ABS` = `"0"` untuk **seluruh 30 baris**. Dua sentinel "tak terbatas" berbeda:
`100.000.000.000` dan `9.999.999.999`.

**Tidak ada satu pun baris `SMI` / Timor-Leste / USD.** `TYPE_BUSINESS` hanya `NONMBU`,
`NONMBUAB`, `NONMBUC`, `TRAVEL`, `PA`, `BONDING`. Jadi ambang `3.500`/`7.000` di `SetEmailKomite`
**tidak punya baris pasangan** — jalur USD itu mati, atau memakai tabel lain.

**Di atas Rp 200.000.000 tidak ada baris untuk PA dan TRAVEL.**

## 14.7 Cacat aturan uang yang baru terungkap dari lapisan database

### Fungsi kurs mengabaikan tanggal dan mengembalikan 1

`Database/GETCURRENCYSTANDARD.fnc` — 23 baris, dibaca utuh dan diverifikasi:

| Baris | Isi |
|---|---|
| `:3` | parameter `i_tgl_kurs` dideklarasi |
| `:9-16` | badan fungsi — **`i_tgl_kurs` tidak muncul sama sekali** |
| `:14` | `AND TRUNC(CurrencyDate) <= TRUNC(sysdate)` — **selalu hari ini** |
| `:15-16` | `ORDER BY CurrencyDate DESC` + `ROWNUM < 2` |
| `:20-22` | `EXCEPTION WHEN NO_DATA_FOUND THEN RETURN 1` |
| `:5` | `RETURN NUMBER result_cache` — cacheable padahal bergantung `sysdate` |
| `:11` | `REPLACE(CurrencyValue,',','.')` — **kurs disimpan sebagai teks** |

Pemanggilnya mengirim **tanggal kerugian**:
`POOLDATA.GETCURRENCYSTANDARD(A.CURRENCYID, to_date(A.DATEOFLOSS,'dd/mm/yyyy'))` —
`RDB List/GetDataTrytyInwardFromUploadData-SQL.xml:34`, `:38`;
`Activity/BrowseDataDetailKlaimXOLAndReas-Act.xml:3289`, `:3320`, `:3331`.

**Dua konsekuensi:**

**(a)** Laporan yang sama menghasilkan angka rupiah berbeda pada hari berbeda.

**(b) `RETURN 1` adalah kegagalan senyap terburuk yang ditemukan.** Mata uang tanpa baris kurs
bertanggal ≤ hari ini diperlakukan **1:1 terhadap rupiah**. Nilai itu masuk perbandingan ambang:

| Ambang | Akibat |
|---|---|
| Notice of Large Losses Rp 1.000.000.000 (`>` strict) | **tidak terpicu** |
| Komite Rp 50.000.000 / Rp 30.000.000 | **tidak terpicu** |

→ **klaim valuta asing bernilai besar dapat lolos tanpa persetujuan komite dan tanpa notifikasi
pimpinan.** Ini lubang otorisasi, bukan utang teknis.

`BRD.md:641` menyatakan aturan ini **✅ siap diimplementasikan**. Urutannya benar, kursnya tidak.
Status itu perlu diturunkan.

### Fee adjuster: tiga rezim, empat nilai hardcode, nol pembulatan

`Database/GET_INTERPOLASIPNC.fnc` menginterpolasi **fee adjuster** dari
`POOLDATA.GCNM_FEE_SCALE (LOSS_AMOUNT, INDEX_FEE, FEE)`:

| Rezim | Kondisi | Rumus | Baris |
|---|---|---|---|
| A | ada `LOSS_AMOUNT >= nilai` **dan** `max_index > 1` | `less_fee + ((v − less_amount) × (Δfee ÷ Δamount))` | `:25` |
| B | pita pertama (`max_index = 1`) | **Rp 1.650.000 konstan** | `:30` |
| C | di atas puncak skala | `FEE(idx 17) + (v − LOSS_AMOUNT(idx 17)) × 2%` | `:36-37` |

**Nol pembulatan** — tidak ada `ROUND`/`TRUNC`/`CEIL`/`FLOOR`; dua pembagian menghasilkan pecahan
rupiah tak terbatas. **Isi tabel `GCNM_FEE_SCALE` tidak ada di repo**, jadi B-5 tetap tidak dapat
diuji meski rumusnya terbaca.

Empat nilai bisnis hardcode melanggar `BRD §11.6` aturan 1: `1650000` (`:30`), `INDEX_FEE=17`
(`:36`), `2/100` (`:37`), plus benih ID `111`/`10001` di procedure lain.

### Tiga validasi yang ternyata mati di produksi

| Validasi | Bukti |
|---|---|
| **Total installment 100%** | `Database/CONVERTJSONPRODUCTION.prc:1516` — `if TotalInstallmentPercentage-100 > 0.1 and TotalInstallmentPercentage-100 < -0.1` — **kondisi mustahil**. Diverifikasi. Polis dengan total ≠ 100% lolos selama ini |
| **Validasi KTP/HP/Email** | `MBU.F_VALIDASI_KLAIM_PENGKINIAN` menerima jenis dokumen di argumen pertama, tapi `RDB List/Validasiklaimpengkiniandata_sql_gcnm-SQL.xml:33` mengirim **alamat email** di posisi itu → selalu `ELSE → RETURN 0`, hasilnya dibuang ke `DBMS_OUTPUT` |
| **`INSERTPOLISTOJSON`** | `Database/INSERTPOLISTOJSON.prc:20-21` — baris pertama badan procedure adalah `o_message := 'GUNAKAN YANG ADA DI GLADMIN.ASMD'; return;`. 90 baris sisanya dead code. Tapi `RDB List/InsertPolicyToJSON-SQL.xml:53` **masih memanggilnya**. Diverifikasi |

Aturan validasi KTP/HP/Email-nya sendiri **bagus dan siap jadi AC**: KTP panjang 14–16 hanya
digit, tolak 7 digit berulang dan urutan naik; HP panjang 10–14 prefiks `08`/`628`; email wajib
`@`, tanpa spasi, **dan tolak domain internal**. Aturannya ada, pemanggilannya salah.

### `INSERT_SALVAGE` menghancurkan kunci baris saat update

`Database/INSERT_SALVAGE.prc` — variabel `idsalvage` hanya di-assign di cabang INSERT (`:21`).
Pada cabang UPDATE, `:47` menulis `IDSALVAGE = idsalvage` yang **masih NULL**, ke baris yang
dicari `WHERE IDSALVAGE = tID` (`:50`), lalu `COMMIT` (`:53`).

**Setiap update salvage menghapus kunci barisnya sendiri.** Perlu satu query ke DBA sebelum
migrasi data: `SELECT COUNT(*) FROM POOLDATA.PNC_SALVAGE WHERE IDSALVAGE IS NULL`.

## 14.8 `StatusPosisi` bukan status — ia konstanta

`Database/PROGRESS_CLAIM_PNC.prc` tidak menulis satu pun literal ke `STATUSPOSISI` — ia meneruskan
parameter `pstsposisi` (`:16-17`, `:39`, `:66`). Enumerasi **seluruh** nilai di 21 call site Pega:
**semuanya `"On Progress"`**, nol pengecualian.

**Penanda selesai yang sebenarnya adalah kolom `PROGRESSDATEDONE`** (timestamp), di-set `SYSDATE`
pada cabang UPDATE (`:40`) dan UPDATE_1 (`:67`) sementara `STATUSPOSISI` tetap `'On Progress'`.

Akibat yang mengikat: `Database/GET_POSISI_PROGRESS_PNC.fnc:15` memfilter
`q.statusposisi = 'On Progress'` — filter yang **cocok dengan semua baris**, termasuk yang sudah
selesai. Fungsi "posisi aktif" itu sebenarnya mengembalikan seluruh riwayat.

**Sistem baru harus memakai `PROGRESSDATEDONE IS NULL` sebagai predikat "aktif", bukan status.**
Ini menguatkan §4.3 dan menambah alasan merevisi D-18.

Juga: `PROGRESS_CLAIM_PNC` punya default **`sysdate + 7`** untuk tanggal follow-up bila kosong
(`:7-8`) — satu-satunya literal angka bisnis di procedure itu, dan bahan AC untuk S-7.
Dua dari empat cabangnya (`'UPDATE'`, `ELSE`) **kode mati** dari sisi Pega.

## 14.9 Dua basis perhitungan TAT yang tidak akan pernah sama

`Database/GETSELISIHJAM.fnc` **tidak memanggil** `DATAMINING.GET_WORKING_HOURS@ASMD` maupun
`GENERAL.HRD_LBR@ASMD`. Algoritmanya (`:9-18`):

1. `weekends2(TRUNC(tglawal), TRUNC(tglakhir))` — jumlah hari akhir pekan
2. `POOLDATA.datediff('SS', tglawal, tglakhir)` — selisih detik kalender penuh
3. kurangi **`86400 × jumlah_akhir_pekan`** bila `cnt > 0`
4. `ROUND(detik ÷ 3600, 2)` — jam, 2 desimal

**Hanya akhir pekan. Tidak ada hari libur, tidak ada jam kerja.** SLA yang dimulai Jumat 16:00 dan
berakhir Senin 09:00 dihitung **17 jam**, bukan ~2 jam kerja.

Sementara `DATAMINING.GET_WORKING_HOURS@ASMD` dipakai **17×** dan `GENERAL.HRD_LBR@ASMD` adalah
kalender hari libur. **Dua basis eksklusif — angka KPI/TAT lama tidak konsisten antar jalur.**

Ditambah `EXCEPTION WHEN OTHERS THEN RETURN 0` (`:19-22`) — kegagalan tak terbedakan dari nol jam,
menyamarkan error di seluruh perhitungan TAT. Dan `weekends2` serta `POOLDATA.datediff` **tidak ada
di export**, jadi definisi "akhir pekan" tidak dapat diverifikasi.

## 14.10 Token penyimpanan dokumen: MD5 tanpa secret, tanpa masa berlaku

`Database/GENERAL.GET_TOKEN_STORAGE.prc:11`:
`standard_hash('ASMAPP'||SYSTIMESTAMP,'MD5')`

**MD5** atas konkatenasi literal `'ASMAPP'` dengan `SYSTIMESTAMP`. Tidak ada secret, tidak ada
salt rahasia, tidak ada nonce acak; `VAPPNAME`/`VUSERINPUT` **tidak masuk ke hash** — hanya
disimpan sebagai kolom (`:13`). **Token sepenuhnya ditentukan waktu server.**

Tabel `GENERAL.GCP_IMAGE` menerima `APPNAME`, `KODEAKSES`, `USERINPUT`, `INPUTDATE` — **tidak ada
kolom kedaluwarsa, tidak ada TTL, tidak ada penanda "sudah dipakai", tidak ada pencabutan.**

Digabung dengan `pyUseAuthentication=false` pada konektor unggah
(`Connect REST/UploadDokumenPNC-ConnectREST.xml:50`) dan **nol retry** (hanya
`pyResponseTimeout=30000`), ini pertanyaan untuk Keamanan Informasi, bukan keputusan teknis.

### `COMMIT` di empat lapis independen pada satu operasi unggah

| Lapis | `berkas:baris` |
|---|---|
| Procedure token | `Database/GENERAL.GET_TOKEN_STORAGE.prc:14` |
| SQL rule pemanggil token | `RDB List/GenerateTokenPNCDokumen-SQL.xml:45` |
| SQL rule insert metadata | `RDB List/InsertDataPNCStorage-SQL.xml:57` |
| Procedure attachment | `Database/SET_ATTACHMENT_64BIT.prc:50` · `TEMP_SET_ATTACHMENT_64BIT.prc:50` |

`GET_TOKEN_STORAGE` juga **tidak punya `ROLLBACK`** di blok exception-nya (`:18-21`).

**Penyimpanan dokumen tidak dapat dibatalkan bersama transaksi klaim pada satu titik pun di
rantai.** Pola *outbox* atau menerima dokumen yatim adalah keputusan ADR yang tidak dapat
dihindari — menguatkan §3.

Tambahan: mekanisme CLOB punya **dua varian tabel** — `pooldata.data_attachfile` (metadata saja)
dan `pooldata.temp_data_attachfile` (metadata + blob, lewat `TEMP_SET_ATTACHMENT_64BIT`).
Dan `SET_ATTACHMENT_64BIT` **menghapus dokumen pada nilai `tCOMMAND` apa pun selain `'INSERT'`**
(`:8`, `:38`) — termasuk string kosong atau salah ketik.

## 14.11 R-10 terbukti — data yang sama hidup empat kali

`R-10` selama ini risiko dugaan. Sekarang ada buktinya:

1. **Dalam satu baris.** `Database/INSERTT_PERSONLIST.prc:141-173` mengekstrak 30 field skalar
   dari JSON, lalu `:351` **menyimpan JSON aslinya juga** ke kolom `OBJECTDATA`.
2. **Lintas tabel.** `Database/CONVERTJSONPRODUCTION.prc:496` membaca
   `JSON_POLIS.DATA_JSONBLOB`, menormalkannya ke `T_POOL_POLICY`/`T_POOL_OUTGO`/`T_POOL_COINS`/
   `T_POOL_INSTALLMENT`/`COVERAGEYEARLY`, lalu **hanya menandai** sumbernya `STS_KONVERSI=1`
   (`:617`, `:4035`) tanpa menghapus.
3. **JSON dirakit ulang lalu ditulis balik ke kolom.** `:4004-4030` merakit JSON **dari** tabel
   relasional lalu `UPDATE T_POOL_POLICY SET responsemsg = …` — salinan ketiga di tabel yang sama.
4. **Salinan keempat untuk backup.** `Database/INSERTOBJECTBACKUP.prc:51-58` menyalin array yang
   sama ke `t_object.OBJECTDATA`.

**Nol pemeriksa konsistensi** di 10 berkas. Satu-satunya "rekonsiliasi" adalah flag
(`STS_KONVERSI`, `IDPROD`, `ISSUCESS`) yang menyatakan *proses selesai*, bukan *data cocok*.

## 14.12 Sepuluh dari dua belas procedure commit sendiri

| Berkas | `COMMIT` |
|---|---|
| **`INSERT_PLADLA.prc`** | **9** (`:69`,`:74`,`:79`,`:138`,`:143`,`:148`,`:179`,`:184`,`:189`) — `ROLLBACK` hanya 1, di handler terluar |
| `UPDATEREAS.prc` | 4 |
| `INSERT_UPDATE_MST_XOL.prc` | 2 dari 6 jalur tulis — **tidak konsisten** |
| `INSERTDATAKOMITELIST` · `INSERT_KPIADJUSTER` · `INSERTMASTERREJECTEDKOMITE` | 2 masing-masing |
| `ADD_NEWMASTERVIRTUALACCOUNT` · `INSERT_SALVAGE` | 1–2 |
| `PEGA_JSON_INSERT_HISTORY_CLAIM_PNC` · `PEGA_JSON_OS_AKSEP_KLAIM` | 0 |

**Selama procedure ini dipanggil apa adanya, Go tidak dapat membungkus beberapa panggilan dalam
satu unit atomik.** Untuk B-9 (`INSERT_PLADLA`, 9 commit) dan B-4 (`UPDATEREAS`, 4 commit) itu
berarti **penerbitan PLA/DLA dan pengelolaan reasuradur tidak dapat dijadikan atomik tanpa
menulis ulang procedure-nya** — dan `INSERT_PLADLA` dapat meninggalkan state setengah jalan
karena `ROLLBACK`-nya terjadi setelah commit.

Pertanyaan yang mengikutinya bukan teknis: **siapa lagi yang memanggil procedure ini selain
Claim PNC?** Menulis ulang procedure yang dipakai sistem lain adalah keputusan lintas tim.

## 14.13 S-5 — sistem lama tidak punya jejak audit nilai

`Database/PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc` — 23 baris. Seluruh isi tulisnya satu baris
(`:7`):

```sql
INSERT INTO LIST_HISTORY_CLAIM_PNC (CASEID, CREATEDATETIME, STATUSNOTE, USERUPDATE)
VALUES (CaseID, CURRENT_TIMESTAMP, StatusNote, UserUpdate);
```

| D-28 mewajibkan | Ada? |
|---|---|
| siapa | ✅ `USERUPDATE` |
| kapan | ✅ `CREATEDATETIME` |
| **nilai sebelum** | ❌ tidak ada kolom, tidak ada parameter |
| **nilai sesudah** | ❌ tidak ada |
| objek/field yang berubah | ❌ tidak ada |

`StatusNote` satu `VARCHAR2` bebas — catatan naratif, bukan pasangan nilai.

**Konsekuensi untuk P-5 dan gerbang 1:** S-5 adalah kemampuan **baru 100%**, bukan migrasi. Tidak
ada baseline Pega untuk diuji setara — jadi `BRD §21.2` kriteria #3 **tidak berlaku untuk S-5**,
sama seperti F-3 yang tidak punya pembanding HCC/HCQ. Dan kolom nilai-sebelum untuk data historis
**tidak akan pernah dapat direkonstruksi**.

Dua cacat kecil: `ErrMsg := 'Data sudah di simpan'` (`:8`) di-set padahal **tidak ada `COMMIT`** di
procedure ini; dan `ROLLBACK` (`:12`, `:19`) **membatalkan seluruh transaksi pemanggil**.
Pemanggilnya juga **tidak ditemukan** di export — kapan jejak audit lama ditulis
**TIDAK DAPAT DIPASTIKAN DARI EXPORT**.

## 14.14 Kandidat R-19 bertambah — dari tiga menjadi sepuluh

`R-19` (cacat aturan uang yang direplikasi bila P-5 dipatuhi buta):

| | Cacat | Bukti |
|---|---|---|
| 1 | Toleransi spreading = pencocokan substring | `Activity/InputRegister_act-Act.xml:13183` |
| 2 | Ambang Rp 50 juta, tiga operator berbeda | `SetEmailKomite:1445` · `…Adjuster:968` · `…Salvage:888` |
| 3 | Penyesuaian 7 jam asimetris dalam satu kondisi | `InputRegister_act:5788` + `:4805` |
| **4** | **Kurs mengabaikan tanggal kerugian** | `Database/GETCURRENCYSTANDARD.fnc:3` vs `:14` |
| **5** | **Kurs tak ditemukan → `RETURN 1`** | `Database/GETCURRENCYSTANDARD.fnc:22` |
| **6** | **`LIMIT_TOP` diabaikan + tie-breaker `dbms_random.value`** | `RDB List/EmailKomiteBerjenjangSimasnet_sql-SQL.xml:98-99` |
| **7** | **Tanggal PLA/DLA dari pengguna dibuang, diganti `SYSDATE`** | `Database/INSERT_PLADLA.prc:67`, `:136`, `:177` |
| **8** | **`NilaiSalvage` hanya ditambahkan bila baris terakhir kebetulan salvage** | `Activity/ValidasiSisaTSI-Act.xml:817` vs `:1489` |
| **9** | **`INSERT_SALVAGE` menulis `IDSALVAGE = NULL` saat update** | `Database/INSERT_SALVAGE.prc:47` |
| **10** | **`GETSELISIHJAM` gagal → `RETURN 0` jam** | `Database/GETSELISIHJAM.fnc:22` |

## 14.15 Pertanyaan baru untuk work owner — dari lapisan database

**Menyentuh uang, prioritas tertinggi:**

1. **Kurs mana yang benar: tanggal kerugian atau hari eksekusi?** Salah satu dari keduanya cacat.
   Bila kurs DOL yang benar, **seluruh angka rupiah pada laporan treaty inward historis salah** —
   dan memperbaikinya membuat gerbang kesetaraan menampilkan selisih pada seluruh data historis.
2. **Apa yang harus terjadi bila kurs tidak ditemukan?** Sekarang `RETURN 1`. Tolak transaksi,
   pakai kurs terakhir, atau gagalkan dengan galat?
3. **Berapa desimal pembulatan fee interpolasi, dan kapan dibulatkan?** Nol pembulatan sekarang.
4. **Apakah Rp 1.650.000 dan tarif marjinal 2% masih berlaku?** Keduanya hardcode.
5. **Mohon isi `GCNM_FEE_SCALE` (17 pita) dan `m_currencystandard`** — blocker sisa B-5.

**Menyentuh otorisasi:**

6. **Klaim Rp 75.000.000 NONMBU masuk jenjang mana?** Source tidak memuat jawabannya. Tiga opsi:
   rentang tertutup `LIMIT_BOTTOM ≤ v ≤ LIMIT_TOP` dengan master dirapikan; hanya batas bawah
   dengan aturan "degree tertinggi menang"; atau kumulatif — semua jenjang yang memenuhi harus
   menyetujui.
7. **Apakah pemilihan anggota komite memang boleh acak?** `dbms_random.value` mungkin disengaja
   (pemerataan beban) atau kecelakaan. Bila disengaja, **AC B-7 tidak boleh deterministik**.
8. **Baris `EMAILKOMITE` dengan `DEGREE=0, LIMIT_BOTTOM=0, LIMIT_TOP=0` — apa maksudnya?**
9. **Di atas Rp 200.000.000 tidak ada jenjang untuk PA dan TRAVEL** — sengaja atau lubang?

**Menyentuh VR-10 / B-5:**

10. **Apakah niat bisnisnya selalu menambahkan `NilaiSalvage` ke sisa TSI?** Bila ya, perilaku lama
    salah dan sistem baru harus memperbaikinya — bukan kesetaraan perilaku.
11. **Mengapa salvage memulihkan kapasitas TSI**, dan nilai mana yang dipakai: `NILAIPENAWARAN`,
    `HARGATERJUAL`, atau `NILAIAKSEPTASI`?
12. **Apakah `>` benar, atau seharusnya `>=`** — adjustment tepat sama dengan sisa TSI diizinkan?
13. **`ValidasiSisaTSI` hanya untuk PA?** Pembatasan hanya ada di pemanggil. Lini lain seharusnya
    juga divalidasi tapi belum, atau memang tidak berlaku?

**Menyentuh TAT / KPI:**

14. **Basis TAT yang benar: `GETSELISIHJAM` (akhir pekan saja) atau `GET_WORKING_HOURS@ASMD`?**
    KPI mana yang dilaporkan ke manajemen?
15. **Apakah `RETURN 0` saat error pernah menyamarkan kegagalan di laporan KPI?**

**Menyentuh dokumen / keamanan:**

16. **Apakah token MD5 tanpa secret dan tanpa TTL diterima Keamanan Informasi untuk sistem baru?**
17. **Berapa TTL token dan URL sebenarnya, dan siapa yang membersihkan `GCP_IMAGE`?**
18. **`SET_ATTACHMENT_64BIT` menghapus pada `tCOMMAND` apa pun selain `'INSERT'`** — pernah terjadi
    penghapusan dokumen tak sengaja?
19. **`temp_data_attachfile` masih dipakai atau sisa eksperimen?** DBA mengirimnya tanpa diminta.

**Menyentuh integritas data — perlu query DBA sebelum migrasi:**

20. `SELECT COUNT(*) FROM POOLDATA.PNC_SALVAGE WHERE IDSALVAGE IS NULL`
21. `SELECT DISTINCT STATUSPOSISI FROM POOLDATA.GCNM_PROGRESS_POSISI_PNC`
22. Apakah polis dengan total persentase installment ≠ 100 memang lolos ke produksi?

**Menyentuh transaksi:**

23. **Bolehkah 10 procedure yang commit sendiri ditulis ulang?** Dan **siapa lagi yang
    memanggilnya selain Claim PNC?** Ini keputusan lintas tim.

## 14.16 Yang harus diminta — daftar sisa

**Tim Pega:** `SendEmailNotification` · 3 router (`PNCAdminRouter`, `PNCTeknikRouter`,
`RouterRCLDokter`) · **`BrowseT_Claim_Adjustment_SQL`** · 5 When rule peran · 7 harness portal ·
library `GCNM` dan `CNM` + `NumberAddSeparator` · Access Group / Role / Privilege / Operator ·
perbaikan 2 berkas salah-isi · dan konfirmasi apakah `SendEmailNotification-Act.xml` yang berisi
rule lain menandakan seluruh 974 berkas perlu divalidasi ulang.

> Catat: `SetTicket` **dicabut** dari daftar permintaan — ia bawaan Pega (§14.2).

**DBA:** `SET_ATTACHFILETEMPSALVAGE` · `UPDATEPREMIUMTEMPLATE` · **12 dependensi** (§14.2,
terberat `UPDATE_LOG_KONVERSI` 162×) · **isi `GCNM_FEE_SCALE` dan `m_currencystandard`** ·
sequence `CLAIM_NO_NONPEGA_SEQ` (belum ada) · DDL untuk 9 tabel baru yang muncul dari source ·
tiga query integritas di §14.15 · dan status `VALID`/`INVALID` `GET_GROUPBUSINESS_XOL` (ia
menugaskan ke field indeks kursor FOR-loop, yang Oracle perlakukan read-only — berkas yang dibaca
mungkin bukan yang berjalan).

**Tim pemilik sistem:** `DATAMINING.GET_WORKING_HOURS@ASMD` (17×, dasar TAT/KPI) ·
`GENERAL.HRD_LBR@ASMD` (kalender libur) · pengganti `INSERTPOLISTOJSON` di `GLADMIN.ASMD`.

**Pemilik API HCC/HCQ:** kontrak login — masih nol jejak.

**Keamanan Informasi:** §14.10 dan D-40.

---

# 15. Matriks kendala per modul — dasar penentuan cakupan tiket

Disusun **2026-09-10** atas permintaan work owner: *"Q6 tetap Opsi 1 jika masih ada halangan, tapi
lampirkan detail kendalanya. Jika sudah tidak ada halangan, Opsi 4."*

**Verdict: halangan MASIH ADA di 31 dari 33 modul → Opsi 1 berlaku.** Bukti per modul di bawah.

Metode inventaris: nama rule diambil dari elemen **`pyRuleName` PERTAMA** per berkas,
dibandingkan **case-insensitive**, lintas 19 folder rule termasuk `InboxAutoClaim/`. Total nama
rule sebenarnya: **2.368**. Metode ini penting — memakai `grep` biasa atas `<pyRuleName>` di
seluruh berkas menghasilkan positif palsu, karena setiap berkas rule memuat indeks dependensi yang
juga berisi elemen `pyRuleName`. Saya sendiri tertipu olehnya dua kali (§15.1).

## 15.1 Empat koreksi tambahan atas laporan saya

### R-02 tertutup penuh untuk daftar job — pemetaan saya sebelumnya TERTUKAR

Saya laporkan job `JobTemporaryCloseClaimPNC` menjalankan activity
`JobHitungDeadlineToTemporaryCLose` yang **hilang**, dan menyebutnya perangkap nama berkas
kesembilan. **Keduanya salah — namanya tertukar.** Yang benar:

| Job (nama rule) | Frekuensi | Jam mulai | Activity target | Target ada? |
|---|---|---|---|---|
| `JOBForKomiteKlaimPNC` | Daily, interval 1 | **06:00:00** | **`AutoAcceptKomite`** | ✅ `Activity/AutoAcceptKomite-Act.xml` |
| `JobHitungDeadlineToTemporaryCLose` | **Weekly** | **23:43:00** | `JobTemporaryCloseClaimPNC` | ✅ |
| `JobSendAutoLODKlaimPersonal` | Daily, interval 1 | **20:54:00** | `Act_SendAutoLODKlaimPersonal` | ✅ |
| `PNCMyReportKlaim3` | Daily, interval 1 | **08:00:00** | `ReportAI_Act` | ✅ |
| `ProcessClaimKredit` | Daily, interval 1 | **10:00:00** | `CreateClaimCredit_Table` | ✅ |

Seluruhnya `pyIsEnabled=true`, `pyApplicableTo=Cluster`, `pyNodeTypesText=BackgroundProcessing`.
**Kelima activity target ADA.** Nama berkas juga cocok dengan nama rule — **tidak ada perangkap
nama kesembilan.**

`Agents/TATReportAgent-Agents.xml`: `pyEnable=true`, **`pyTriggerInterval=1800`** (30 menit),
**`pyBypassActivityAuthentication=true`**. Tiga rule yang dirujuknya —
`CreateCasePNCAgent_ActButton`, `AlertAgentKasirBlmTransferKlaim`, `TransferAllCaseNotAssigned` —
**ketiganya ADA**.

> **R-02 pada tingkat artefak: TERTUTUP.** 5 job + 1 agent diketahui, berjadwal, aktif, dan
> seluruh target-nya ada. Yang tersisa adalah pertanyaan bisnis, bukan artefak — lihat S-6.

### `JOBForKomiteKlaimPNC` menyetujui komite otomatis setiap hari jam 06:00

`pyActivityName=AutoAcceptKomite`, `pyBaseClass=ASM-FW-GCNMFW-Work-Komite`, aktif, harian.
Sebuah job terjadwal yang **menyetujui komite secara otomatis** — perilaku otorisasi yang tidak
tercatat di dokumen mana pun.

Berdampingan dengan `dbms_random.value` sebagai penentu anggota komite (§14.6), model persetujuan
B-7 jauh lebih longgar daripada yang digambarkan `BRD §11.4` dan D-14.

### Deskripsi tiga job bertentangan dengan konfigurasinya

`JOBForKomiteKlaimPNC`, `JobHitungDeadlineToTemporaryCLose`, dan `JobSendAutoLODKlaimPersonal`
semuanya ber-`pyDescription = "job jalan 2 menit"`, sementara `pyRecurringFrequency` = `Daily`
atau `Weekly`. Selisihnya 720× sampai 5.040×. Perlu konfirmasi mana yang benar.

### `Service REST/` — permukaan integrasi MASUK yang belum pernah diaudit

Empat layanan **inbound**: `KomiteAcceptAdjustment` · `KomiteAcceptAdjustmentPA` ·
`RecivedDataandAttachmentLelangASMSimasbid` · `RequestCreateClaimCredit2`.

Dua di antaranya **menerima persetujuan komite dari sistem lain**. Seluruh analisis integrasi
sebelumnya (§1.2, `FR-S4`, D-25) hanya melihat Connect REST **keluar**. Permukaan masuk belum
pernah masuk hitungan — ini penambahan lingkup `S-4`, bukan detail.

## 15.2 Artefak kunci yang dikonfirmasi masih hilang

Diverifikasi dengan metode `pyRuleName` pertama, case-insensitive, lintas 19 folder:

| Artefak | Tipe | Memblokir |
|---|---|---|
| `SendEmailNotification` | Activity, **15 pemanggil** (dikoreksi dari 17 — `D-73`) | **S-3** |
| `BrowseT_Claim_Adjustment_SQL` | Connect SQL | **B-5 / VR-10** |
| `PNCAdminRouter` · `PNCTeknikRouter` · `RouterRCLDokter` | Router | **B-6** |
| `SendToReceiveDocument` · `SendtoPUCL` · `KomiteAssign_ticket` · `RCLDokter` | Ticket rule | B-14 · B-11 · B-7 |
| `IsAsuransiKredit` | When | **B-2** (gerbang SLIK) |
| `NonMBU` · `NotPA` · `ElseRCLMSIG` | When, dirujuk `pyTaskWhen` di `Flow/` | **B-2 / B-11** — percabangan flow utama |
| `IsSurvey` · `IsKomite` · `IsGCNMReport` · `IsNotViewClaim` · `IsPNCBonding` | When | U-3 · B-7 · B-8 · S-2 |
| `InsertDataAkseptasiToLeader` · `InsertLogKasir_act` · `TransferCashierDataASM_act` | Activity | **B-10** |
| `CompliancePNC` · `InvestigatorPNC` · `RCLPUCL` | `Data-Admin-WorkBasket` | B-6 · B-11 |
| 7 harness portal | Harness | U-1 · U-3 |
| Library `GCNM` (99 panggilan) · `CNM` (2) · 4 fungsi `ASM` · `CSUtils` | Function | lintas modul |
| Access Group · Role · Privilege · Operator | — | **F-3** |
| `SET_ATTACHFILETEMPSALVAGE` · `UPDATEPREMIUMTEMPLATE` | Oracle | S-1 · B-1 |
| 12 dependensi procedure (`UPDATE_LOG_KONVERSI` **162×**) | Oracle | **B-1** |
| Isi `GCNM_FEE_SCALE` (17 pita) · isi `m_currencystandard` | data | **B-5** |
| Kontrak HCC/HCQ | — | **F-3** |

Total artefak yang masih harus diminta: **±397**.

## 15.3 Matriks kendala per modul

Kolom **Verdict**: `PENUH` = tiket lengkap dengan AC berangka · `SEBAGIAN` = tiket penuh untuk
bagian yang buktinya lengkap, `needs-info` untuk bagian terhalang · `TERHALANG` = tidak dapat
`ready-*` sampai penghalangnya lepas.

### Modul fondasi

| Modul | Kendala artefak | Kendala keputusan | Verdict |
|---|---|---|---|
| **F-1** Kerangka Aplikasi | instance DSS `ServiceFromTable` (1 setting, hanya rujukan) · `Data-Admin-DB-Name` tidak ada | daftar setting boleh disusun dari nol? (§10.9 #1) · **720 dari 902 activity tanpa penanganan error — replikasi kegagalan senyap atau gagal keras?** (§10.9 #3) · cara monitoring sekarang (§10.9 #4) | **SEBAGIAN** |
| **F-2** Akses Data | **64 Connect SQL** · DDL seluruh tabel (R-08) · 12 dependensi procedure · **`pyParamArray` kosong → 1.646 step RDB tidak terbaca, inventaris SQL adalah batas bawah** | 6 skema DB Link → API atau langsung? · selisih 68 objek tabel · **538 `{ASIS:}` — himpunan filter sah yang mana?** | **SEBAGIAN** |
| **F-3** Identitas & Akses | **HCC/HCQ nol jejak — tidak ada kontrak** · **tidak ada folder Access Group/Role/Privilege/Operator sama sekali** · 5 When rule peran · penugasan operator→grup tidak ada di DB | pemetaan 22 access group → 51 menu masih akurat? · `T_ACCESS_GROUP_PNC` fungsinya apa? · pernah ada temuan audit soal 901/902 tanpa privilege? | **TERHALANG** untuk identitas; **PENUH** untuk middleware otorisasi per endpoint |
| **F-4** Master Data | isi `GCNM_FEE_SCALE` · isi `m_currencystandard` · DDL 9 tabel baru | **mana dari ≥29 kelompok master yang masuk lingkup** (baru Bengkel/Sparepart/Supplier yang diputuskan) · struktur `EMAILKOMITE` dipertahankan? · ambang Large Losses ada fiturnya? · batas tanggal per lini bisnis = kebutuhan baru? · alur persetujuan master berlaku untuk semua? | **SEBAGIAN** |
| **F-5** Waktu & Zona Waktu | — | **`addCalendar(...,12,0,0)` 101 titik di 4 activity — apa maksudnya?** · `"WIB"` (25×) vs `Asia/Jakarta` (137×) hasilnya sama? · ada data produksi tergeser ganda +14 jam? · representasi otoritatif untuk `DateOfLoss`/`ReportDate`? | **SEBAGIAN** — lingkup jelas (118 titik, 36 activity), keputusan memengaruhi AC |

### Modul bisnis inti

| Modul | Kendala artefak | Kendala keputusan | Verdict |
|---|---|---|---|
| **B-1** Polis & Snapshot | **12 dependensi procedure** (`UPDATE_LOG_KONVERSI` 162×, `GETNEWID` 42×, `PKG_COUNTER_PRODUCTION` 25×, `PROCESSQUEUEDIRECT` 10×) · 4 definisi queue `DBMS_AQ` · `JSON_MBU`/`JSON_FIRE`/`JSON_POLIS_PA`/`JSON_POLIS_TRAVEL`/`JSON_POLIS_MARINE_CARGO`/`json_aneka` · pengganti `INSERTPOLISTOJSON` di `GLADMIN.ASMD` | snapshot minimal 16 field polis — disengaja atau harus diperluas? · `JSON_POLIS`/`JSON_KLAIM` sumber kebenaran atau staging? · ambang >50 objek → jalur asinkron masih relevan? · 2 IDPEGA hardcode sebagai pengecualian premi | **SEBAGIAN** |
| **B-2** Registrasi Klaim | **`IsAsuransiKredit`** (gerbang SLIK) · **`NonMBU`, `NotPA`, `ElseRCLMSIG`** — percabangan flow utama | **toleransi 30 hari Bonding mati atau BRD salah?** · DOL bagian kunci duplikasi? · lokasi tidak diperiksa untuk PA/Travel — sengaja? · **jalur API/JSON tanpa aturan 7/30/90 — sengaja?** · validasi KTP/HP/Email dipanggil salah argumen — pernah aktif? | **SEBAGIAN** |
| **B-3** Objek & Coverage | banyak When klasifikasi produk Aneka hilang (`IsARBusiness` 24× · `IsAnekaPerYear` 16× · `IsMBBusiness` 16× · `IsMoneyInsurance` 15× · `IsElectronicEquipment` 10× dst.) | pemetaan GroupPanel `003` berkonflik: `LocationList` vs `t_anekalist` | **SEBAGIAN** |
| **B-4** Spreading Reasuransi | — (aturan spreading ada di lapisan Pega, terverifikasi §8.3) | **toleransi 99,99% adalah pencocokan substring — perbaiki atau replikasi?** · berapa desimal share yang sah? · Group Panel `003` cek semua baris Fac Offer atau hanya baris pertama? · pemetaan kode `10001`/`10007`/`10015` · **`UPDATEREAS` 4 commit — boleh ditulis ulang?** | **SEBAGIAN** |
| **B-5** Estimasi & Settlement | **isi `GCNM_FEE_SCALE` (17 pita)** · **isi `m_currencystandard`** · **`BrowseT_Claim_Adjustment_SQL`** | **kurs: tanggal kerugian atau hari eksekusi?** · **kurs tak ditemukan → `RETURN 1`, apa yang benar?** · berapa desimal pembulatan fee? · Rp 1.650.000 dan 2% masih berlaku? · salvage memulihkan TSI — niat bisnisnya? · `>` atau `>=` pada sisa TSI? | **TERHALANG** |
| **B-6** Penugasan & Inbox | **3 router** · 3 `Data-Admin-WorkBasket` · 4 harness inbox | logika 3 router: konfirmasi bahwa mereka memakai jalur `counter_quota` (§14.5) · `operator_ID != 'ELLENSUPRIYATI'` hardcode di SQL — dibawa sebagai peran? | **SEBAGIAN** — algoritma terbaca, router perlu konfirmasi |
| **B-7** Komite | `KomiteAssign_ticket` · `IsKomite` | **klaim Rp 75.000.000 NONMBU masuk jenjang mana?** (`LIMIT_TOP` diabaikan + tie-breaker acak) · **`dbms_random.value` disengaja?** — bila ya, AC tidak boleh deterministik · **`AutoAcceptKomite` harian jam 06:00 — perilaku yang benar?** · baris `DEGREE=0` maksudnya? · di atas Rp 200 juta tidak ada jenjang · `KOMITEKE` immutable setelah insert | **TERHALANG** |
| **B-8** Survey & Adjuster | `IsSurvey` | **rumus skor KPI adjuster tidak ada di mana pun** — `INSERT_KPIADJUSTER` hanya menyimpan 10 komponen tanpa agregasi · kunci upsert `CASEID` saja — adjuster kedua menimpa yang pertama, benar? | **SEBAGIAN** |
| **B-9** PLA / Pre-DLA / DLA | status `VALID`/`INVALID` `GET_GROUPBUSINESS_XOL` di produksi | **`TGLPLA` dari pengguna atau `SYSDATE`?** (parameter dibuang) · **catatan: tiga perilaku berbeda untuk PLA/DLA/PREDLA — mana yang benar?** · PLA tanpa nilai sah? · PLA/DLA tanpa email reasuradur sah? · kunci duplikat PLA 5 kolom vs DLA 6 kolom — sengaja? · `T_PREDLALIST` masih dipakai? · **`INSERT_PLADLA` 9 commit — boleh ditulis ulang?** | **SEBAGIAN** |
| **B-10** Akseptasi & Pembayaran | **`InsertDataAkseptasiToLeader`** · **`InsertLogKasir_act`** · **`TransferCashierDataASM_act`** | header `Authorization` hardcode di 3 Connect REST kasir · `PEGA_JSON_OS_AKSEP_KLAIM` memetakan `STS_PLA` → kolom `STS_DLA` — benar? · `ErrMsg` membawa nomor VA sekaligus pesan galat | **SEBAGIAN** |
| **B-11** RCL / PUCL / Compliance | `SendtoPUCL` · `RCLDokter` · `RouterRCLDokter` · `ElseRCLMSIG` · `IsNotViewClaim` · workbasket `RCLPUCL` dan `CompliancePNC` | siapa yang boleh memicu transisi lateral (nol pagar izin) | **TERHALANG** |
| **B-12** Salvage & Recovery | `SET_ATTACHFILETEMPSALVAGE` | **`INSERT_SALVAGE` menulis `IDSALVAGE = NULL` saat update** — perlu `SELECT COUNT(*) … WHERE IDSALVAGE IS NULL` dari DBA · `tterjual='2'` → `STSTRANSFER='5'`, satu item terjual mengubah status seluruh salvage induk — benar? · `SISAKLAIM` tidak dihitung, diterima apa adanya | **SEBAGIAN** |
| **B-13** Open Protection | — (`IsReqProtection` dan `IsOpenProtectionPNC` **ADA**; Ticket `TC_PNCInputProtection` dan `Akp_PNCInputProtection` **ADA**) | `CreateProtection_Flow` hanya 6 shape — cakupan sekecil itu benar? | **SEBAGIAN** — paling sedikit kendalanya di antara modul bisnis |
| **B-14** Receive Document | `SendToReceiveDocument` (Ticket) | — | **SEBAGIAN** |

### Modul pendukung

| Modul | Kendala artefak | Kendala keputusan | Verdict |
|---|---|---|---|
| **S-1** Dokumen & Lampiran | `SET_ATTACHFILETEMPSALVAGE` · `POOLDATA.base64decode` | **token MD5 tanpa secret dan tanpa TTL — diterima Keamanan Informasi?** · TTL token dan URL berapa? · siapa membersihkan `GCP_IMAGE`? · **`COMMIT` di 4 lapis → pola outbox atau dokumen yatim?** · `SET_ATTACHMENT_64BIT` menghapus pada `tCOMMAND` apa pun selain `'INSERT'` · `temp_data_attachfile` masih dipakai? · penulis `ViewGoogleStorage.Storage` tidak ditemukan | **SEBAGIAN** |
| **S-2** Laporan & Export | **12 Report Definition** (`BrowseVPanel_HE_RD` 82×) · 10 template HTML · 3 Correspondence | **54 dari 56 laporan memotong hasil di 500 baris — pengguna tahu?** Menampilkan semua baris mengubah angka yang mereka lihat | **SEBAGIAN** |
| **S-3** Notifikasi | **`SendEmailNotification` — 15 pemanggil** · 10 template HTML · 3 Correspondence | penerima personal (Gmail) di jalur produksi — mailbox pengganti? · rotasi 3 password SMTP (31 lokasi) | **TERHALANG** |
| **S-4** Integrasi Eksternal | 6 API pengganti DB Link **belum ada** (D-25/R-03) · instance 2 Authentication Profile · nilai DSS `ServiceFromTable` | **4 Service REST masuk baru ditemukan — masuk lingkup?** · kesembilan Connect REST baru `pyUseAuthentication=false` · `GetPremiumPaid_SPK` `http://` ke IP:port tanpa TLS untuk data premi · endpoint BRI menunjuk sandbox · **total Connect REST keluar 21, bukan 12** (`D-73`) | **TERHALANG** |
| **S-5** Jejak Audit | — (terbukti tidak ada baseline) | **peristiwa apa yang wajib masuk audit dari sisi kepatuhan?** · lama retensi (D-28 terbuka) · data historis tanpa kolom before/after — diterima? | **SEBAGIAN** — lingkup jelas, daftar peristiwa `needs-info` |
| **S-6** Penjadwalan | — **R-02 tertutup**: 5 job + 1 agent, berjadwal, aktif, seluruh target ada | **deskripsi "job jalan 2 menit" vs konfigurasi Daily/Weekly — mana yang benar?** · **`AutoAcceptKomite` harian jam 06:00 — perilaku yang benar?** · `pyBypassActivityAuthentication=true` pada agent — diterima? | **SEBAGIAN** — naik dari TERHALANG |
| **S-7** Dashboard & Monitoring | `weekends2` · `POOLDATA.datediff` | **basis TAT: `GETSELISIHJAM` (akhir pekan saja) atau `GET_WORKING_HOURS@ASMD` (17×)?** · `RETURN 0` saat error pernah menyamarkan kegagalan KPI? · `STATUSPOSISI` konstanta — `PROGRESSDATEDONE` jadi predikat aktif | **TERHALANG** |
| **S-8** Perkakas Uji Kesetaraan | — (modul baru, `D-42`) | **seluruh Lapis 4 belum ditanyakan**: data historis mana, lingkungan mana, siapa boleh menembak Pega, boleh baca produksi read-only?, siapa menyetujui setiap selisih · **dan pengecualian: F-3 dan S-5 tidak punya baseline** | **TERHALANG** |

### Frontend

| Modul | Kendala artefak | Kendala keputusan | Verdict |
|---|---|---|---|
| **U-1** Kerangka SPA | **7 harness portal** (`InboxServiceCenter` · `InboxCloseClaim_Harness` · `InboxRequestSalvage` · `PNCViewClaim` · `ReportProduksiPA_harnes` · `InboxOutstanding_Harness` · `LostAdjuster_harness`) | 7 layar itu masih aktif di produksi? Bila tidak, 7 rute SPA dihapus dari lingkup · dari 38 harness tanpa entri menu, mana rute berdiri sendiri dan mana modal? (`pyHarnessPurpose` tidak ada) | **SEBAGIAN** |
| **U-2** Pustaka Komponen | — | grid 39 kolom dan 25 kolom: semua kolom benar-benar dilihat pengguna? · tambah/hapus baris inline diinginkan di sistem baru? | **PENUH** |
| **U-3** Layar Inbox | 4 harness inbox · `IsSurvey` · `IsKomite` | 4 harness itu masih dipakai? | **SEBAGIAN** |
| **U-4** Layar Transaksi | **18 Section** (`ObjectCoverage` 54× · `ObjectItemList` 32×) · **66 Flow Action** (`GCNMViewAttachment` 38× · `PNCUploadClaimCSV` 28×) | — | **SEBAGIAN** |
| **U-5** Layar Laporan | 12 Report Definition · `IsGCNMReport` | pemotongan 500 baris (lihat S-2) | **SEBAGIAN** |
| **U-6** Layar Master Data | DDL 9 tabel baru | **lingkup ≥29 kelompok master belum diputuskan** · alur persetujuan master berlaku untuk semua? | **TERHALANG** |

## 15.4 Rekapitulasi

| Verdict | Jumlah | Modul |
|---|---|---|
| **PENUH** — tiket lengkap, AC berangka, tanpa `needs-info` | **1** | `U-2` |
| **SEBAGIAN** — tiket penuh untuk bagian tak terhalang, `needs-info` untuk sisanya | **24** | `F-1` `F-2` `F-4` `F-5` `B-1` `B-2` `B-3` `B-4` `B-6` `B-8` `B-9` `B-10` `B-12` `B-13` `B-14` `S-1` `S-2` `S-5` `S-6` `U-1` `U-3` `U-4` `U-5` + `F-3` (bagian middleware) |
| **TERHALANG** — tidak boleh `ready-*` | **8** | `F-3` (identitas) `B-5` `B-7` `B-11` `S-3` `S-4` `S-7` `S-8` `U-6` |

**Hanya satu modul dari 33 yang benar-benar tanpa halangan.** Opsi 4 (seluruh modul ditiketkan
penuh) akan menghasilkan 32 modul berisi acceptance criteria yang mengarang.

Lima modul yang `BRD §21.4` sebut tidak dapat dinyatakan diterima sampai R-01 tertutup —
`FR-B5`, `B7`, `B9`, `B10`, `B12` — **statusnya sekarang terbelah**: `B-9`, `B-10`, `B-12` naik
menjadi SEBAGIAN karena source procedure-nya datang; `B-5` dan `B-7` tetap TERHALANG, tetapi
**bukan lagi oleh R-01** — melainkan oleh isi dua tabel master (`GCNM_FEE_SCALE`,
`m_currencystandard`) dan oleh keputusan bisnis yang belum diambil (jenjang Rp 75 juta, kurs).

Itu perubahan yang perlu dicatat: **R-01 bukan lagi penghalang utama.** Penghalang utama sekarang
**R-16** (±397 artefak, terberat 137 When rule) dan **keputusan bisnis yang belum diambil**.

## 15.5 Bentuk tiket yang mengikuti dari verdict ini

Sesuai `D-37` (kebijakan modul terhalang) dan Q6 opsi 1:

- **`U-2`** → tiket penuh, `ready-for-human`, AC berangka dari §10.7.
- **24 modul SEBAGIAN** → dipecah per bagian. Bagian yang buktinya lengkap mendapat AC berangka
  dan boleh `ready-for-human`; bagian terhalang menjadi tiket terpisah ber-`needs-info` dengan
  **sebutan persis apa yang kurang dan siapa yang dapat melengkapinya**.
- **8 modul TERHALANG** → tiket ditulis sekarang dengan `Kesiapan: terhalang <R-nn>`, **tidak
  boleh** `ready-for-agent` maupun `ready-for-human`, sesuai `D-37`.

Setiap tiket ber-`needs-info` wajib menyebut penghalangnya dari §15.2 dan §15.3 dengan
`berkas:baris` atau nama artefak, bukan keterangan umum.
