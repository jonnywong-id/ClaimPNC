# Permintaan Artefak ke Tim Pega

Berkas ini memuat **permintaan artefak** yang menghalangi pembangunan modul di aplikasi Claim
PNC yang baru. Satu bab per permintaan, dan **setiap bab menyebut tujuannya sendiri di kepala
bab** — sebagian ke Tim Pega, sebagian ke DBA.

Seluruhnya ditulis supaya dapat dibaca tanpa mengetahui apa pun tentang aplikasi penggantinya
— tanpa istilah Go, React, maupun nama modul internal kami.

| Bab | Ditujukan ke | Pokok permintaan |
|---|---|---|
| 1. Master Pasal AI | **Tim Pega** | export rule yang hilang |
| 2. Inbox Investigator — Export Data Investigation | — | **DIBATALKAN** — fiturnya dihapus; bab disimpan sebagai catatan |
| 3. Inbox Receive TKA | **DBA** dan **Tim Pega** | hak akses, kunci tabel, kewenangan menulis, dan satu koreksi dari pihak kami |

| | |
|---|---|
| Dasar | `R-16` — ±242 rule dirujuk tetapi tidak ada di export; tujuh tipe rule tidak pernah diaudit, **Section dan Activity termasuk di antaranya** |
| Bentuk permintaan | `D-39` — **export ulang berbasis Product rule dengan opsi *include dependent rules***, bukan pemilihan manual per rule |
| Cara memeriksa kelengkapan | setiap bab memuat daftar nama rule yang **wajib ada** pada hasil export |

> **Kenapa daftar nama tetap disertakan padahal `D-39` memilih export berbasis Product
> rule.** Export berbasis Product rule adalah cara yang benar, tetapi tidak ada yang dapat
> membuktikan hasilnya lengkap tanpa sesuatu untuk dicocokkan. Daftar nama di setiap bab
> berfungsi sebagai **alat verifikasi**, bukan sebagai permintaan alternatif.

---

## 1. Master Pasal AI — `MENU_ID 36`

| | |
|---|---|
| **Status** | **Terpenuhi sebagian** — section dan activity diterima; **dua Connect-SQL masih diminta** |
| **Pemilik** | Tim Pega |
| **Menghalangi** | **hanya nama tabelnya** — kolom, penyaring, dan paginasinya sudah terbaca |
| **Butir menu** | `Database/m_menu_aplikasi_pnc.csv:31` — `1,36,"Master Pasal AI","DetailMasterPasalAI",1,1,1126` |
| **Otorisasi** | `Database/m_otorisasi_pnc.csv` — grup `IT`, sama seperti `MENU_ID 27` |

### 1.1 Riwayat permintaan

| Tanggal | Peristiwa |
|---|---|
| 2026-09-22 | Diminta: section isi `DetailMasterPasalAI`, karena harness-nya hanya kerangka |
| 2026-09-22 | **Diterima** — `Section/DetailMasterPasalAI_sect.xml` (159 KB, utuh) |
| 2026-09-22 | Permintaan dipersempit: **`Activity GetListPasalAI`** |
| 2026-09-23 | **Diterima** — `Activity/GetListPasalAI_act.xml` (91 KB) |
| 2026-09-23 | Permintaan dipersempit lagi: **dua Connect-SQL** — `GetListDataPasalAI` dan `CountDataPasalAI` |

Pengiriman kedua mengungkap **nama kolom yang sebenarnya**, yang selama ini tersembunyi di
balik properti klipboard bernama warisan:

```
Activity/GetListPasalAI_act.xml:548
"WHERE (WP_PASAL LIKE '%"+TempSearch.Country+"%'
     OR WP_AYAT LIKE '%"+TempSearch.Country+"%'
     OR WP_KEJADIAN LIKE '%"+TempSearch.Country+"%')"
```

| Judul kolom | Properti klipboard | **Kolom basis data** |
|---|---|---|
| No Pasal | `.City` | **`WP_PASAL`** |
| Ayat | `.CityID` | **`WP_AYAT`** |
| Kejadian | `.District` | **`WP_KEJADIAN`** |

Sekaligus terbaca perilaku pencariannya: satu kata kunci dicocokkan ke **ketiga kolom
sekaligus** dengan `LIKE '%…%'`, digabung `OR`.

Pengiriman pertama menjawab dua hal sekaligus, dan keduanya berharga:

1. **Layarnya memang selesai dibuat.** Section-nya dibuat 2023-05-16, disunting 2023-06-12,
   di-commit 2023-07-14 — dua bulan setelah harness-nya. Dugaan kami sebelumnya bahwa layar
   ini mungkin tidak pernah diselesaikan **gugur**.
2. **Bentuk layarnya kini terbaca lengkap** — lihat §1.4.

### 1.2 Yang diminta sekarang

**Dua Connect-SQL (RDB List):**

```
Rule-Connect-SQL   GetListDataPasalAI   (kelas ASM-FW-GCNMFW-Data-ClaimData, ruleset GCNM)
Rule-Connect-SQL   CountDataPasalAI     (kelas ASM-FW-GCNMFW-Data-ClaimData, ruleset GCNM)
```

Kelas dan ruleset-nya terbaca dari activity yang sudah kami terima:

| Bukti | `berkas:baris` |
|---|---|
| `.SQLName := "GetListDataPasalAI"` | `Activity/GetListPasalAI_act.xml:981` |
| `.ClassSQLName := "ASM-FW-GCNMFW-Data-ClaimData"` | `:964` |
| Rujukan rule `ASM-FW-GCNMFW-Data-ClaimData GCNM GetListDataPasalAI` | `pxRuleReferences` |
| Rujukan rule `ASM-FW-GCNMFW-Data-ClaimData GCNM CountDataPasalAI` | `pxRuleReferences` |

### 1.3 Kenapa hanya ini yang tersisa

Ketiga artefak sebelumnya sudah menjawab hampir seluruhnya:

| Sudah diketahui | Dari mana |
|---|---|
| Rupa layar, tombol, paginasi 30 baris | Section |
| **Nama kolom** — `WP_PASAL`, `WP_AYAT`, `WP_KEJADIAN` | Activity `:548` |
| Perilaku pencarian — `LIKE '%…%'` atas ketiga kolom, digabung `OR` | Activity `:548` |
| Paginasi dihitung di server (`FirstRow`/`LastRow`/`PageSize`) | Activity |

Yang **belum** diketahui tinggal satu hal, dan ia hanya tertulis di dalam kedua Connect-SQL
itu: **nama tabelnya**, beserta daftar kolom `SELECT` dan `ORDER BY`-nya.

Pencarian `WP_PASAL`, `WP_AYAT`, dan `WP_KEJADIAN` atas seluruh export mengembalikan **tepat
satu berkas** — activity itu sendiri. Tidak ada satu pun kueri lain di export yang menyentuh
ketiga kolom tersebut, sehingga tabelnya tidak dapat diturunkan dari apa pun yang ada di
tangan kami.

### 1.4 Bentuk layar yang sudah terbaca — untuk dicocokkan Tim Pega

Kami cantumkan supaya Tim Pega dapat memastikan activity yang dikirim memang yang benar.

| Hal | Isi | `berkas:baris` |
|---|---|---|
| Sifat layar | **baca-saja** — `pyEditingMode = readOnly` | `:4405` |
| Tombol | **hanya `Cari` dan `Refresh`** — tanpa Tambah, Simpan, Ubah, Hapus | `:1833`, `:2222` |
| Isian pencarian | satu kotak teks, terikat `TempSearch.Country`, berlabel "Cari" | `:1521`, `:1579` |
| Perilaku `Cari` | `refresh` atas section ini — memicu ulang deferred load | `:1851`, `:1859` |
| Perilaku `Refresh` | mengosongkan `TempSearch.Country`, lalu memanggil `GetListPasalAI(Flags=2)` | `:2248`, `:2279` |
| Sumber grid | `TempDetailData.pxResults`, kelas `ASM-FW-GCNMFW-Data-ClaimData` | `:3163`, `:3144` |
| Jumlah kolom | **3** | `:3202` |
| Paginasi | `ButtonPagingInbox`, mode `Numeric`, **dihitung di server** — grid sendiri ber-`pyPageMode = None` | `:2993`, `:3001`, `:4488` |
| Ukuran halaman | **25 baris** — ditetapkan activity, bukan section | `Activity/GetListPasalAI_act.xml:874` |
| Tab | tidak ada — satu layar rata | `:728` |

**Ketiga kolomnya:**

| # | Judul kolom | Terikat properti | Kontrol | Lebar |
|---|---|---|---|---|
| 1 | **No Pasal** | `.City` | `pxTextInput` | 48 px |
| 2 | **Ayat** | `.CityID` | `pxTextInput` | 47 px |
| 3 | **Kejadian** | `.District` | `pxTextArea` | 283 px |

> **Nama propertinya tidak mencerminkan isinya, dan itu bukan salah baca kami.** `.City`,
> `.CityID`, dan `.District` adalah sisa dari asal-usul section ini: ia Save-As dari
> `BrowsePasalDeatailMaster` (2022), yang sendirinya Save-As dari `BrowseDetailSuveryors`
> (2017). Properti lamanya dipakai ulang untuk menampung isi yang sama sekali berbeda.
>
> Inilah yang membuat `GetListPasalAI` benar-benar diperlukan: **hanya activity itu yang
> mengetahui kolom basis data mana yang sebenarnya mengisi ketiga properti tersebut.**

**Polanya sudah kami pastikan dari layar sejenis yang lengkap di export.**
`Activity/SearchDataMasking-Act.xml` mengisi page klipboard **`TempDetailData` yang sama**,
dan ia memanggil `RDB List/SearchMasking_SQL-SQL.xml` yang isinya:

```sql
SELECT (SELECT BRANCHNAME FROM POOLDATA.BRANCH a WHERE cabang = a.ID) AS "CABANG",
       CABANG    as "ProvinceID",
       USERINPUT as "UserInput",
       ...
  FROM POOLDATA.MST_PROTEKSI_DATA_PNC
 WHERE {ASIS:InputSearch.CARI1}
```

Perhatikan `CABANG as "ProvinceID"` — kolom basis data dialiaskan ke nama properti klipboard
yang dipakai ulang, persis seperti `.City` / `.CityID` / `.District` pada layar kami.

Jadi `GetListPasalAI` hampir pasti berbentuk sama: sebuah Connect-SQL yang mengaliaskan tiga
kolom tabel pasal AI menjadi `"City"`, `"CityID"`, dan `"District"`. **Connect-SQL itulah
yang kami butuhkan**, karena hanya di sana nama tabel dan ketiga kolom aslinya tertulis.

### 1.5 Dua sisa Save-As yang sudah kami pastikan MATI — tidak perlu dikirim

Agar tidak menghabiskan waktu Tim Pega pada jalur yang salah:

| Sisa | Kenapa mati |
|---|---|
| `BrowseVDSurveyors_RD` atas kelas `ASM-FW-GCNMFW-Int-V_D_SURVEYORS` (`:4487`) | Berada di dalam `pyGridProps`, tetapi grid-nya ber-`pySourceType = Property` (`:4476`), sehingga wiring Report Definition itu **tidak dipakai saat berjalan**. Report Definition-nya sendiri sudah ada di export |
| `pyPostGridUpdate` (`:4471`) dan `pyPreGridUpdate` (`:4485`) | Keduanya ditandai **tidak ada** oleh section itu sendiri — `pyGridPostActivityExists = false` (`:4431`), `pyGridPreActivityExists = false` (`:4491`) |

### 1.6 Yang SUDAH ada di tangan kami — tidak perlu dikirim ulang

| Rule | Tipe | Keterangan |
|---|---|---|
| `DetailMasterPasalAI` | Rule-HTML-Harness | kelas `Data-Portal`, ruleset `GCNMFW 01-02-05` |
| `GridDetailMasterPasalAI` | Rule-HTML-Section | tertanam di dalam berkas harness di atas |
| `DetailMasterPasalAI` | Rule-HTML-Section | kelas `@baseclass`, ruleset `GCNMFW 01-02-06` — **diterima 2026-09-22** |
| `ButtonPagingInbox` · `pyGridNoResultsMessage` | Rule-HTML-Section | keduanya sudah ada |
| `BrowseVDSurveyors_RD` · `BrowseVMSurveyors_RD` | Report Definition | sudah ada; lihat §1.5 — tidak dipakai |
| `pyCaption Detail Pasal AI` · `No Pasal` · `Ayat` · `Kejadian` · `Cari` | Field Value | — |

### 1.7 Cara memeriksa hasil export sudah lengkap

Hasil export dinyatakan lengkap bila memuat **kedua** Connect-SQL berikut, dan masing-masing
memuat teks kuerinya (`pyBrowseSQL`):

```
Rule-Connect-SQL   GetListDataPasalAI
Rule-Connect-SQL   CountDataPasalAI
```

Cara tercepat memastikan yang dikirim benar: buka `pyBrowseSQL`-nya dan periksa bahwa ia
menyebut ketiga kolom **`WP_PASAL`**, **`WP_AYAT`**, dan **`WP_KEJADIAN`**, serta satu nama
tabel `FROM`.

Setelah keduanya diterima, **tidak ada lagi artefak yang kami butuhkan untuk layar ini** —
modulnya dapat langsung dibangun.

---

## 2. Inbox Investigator — Export Data Investigation — `MENU_ID 48`

| | |
|---|---|
| **Keadaan** | **DIBATALKAN 2026-09-24** — Work Owner memutuskan fitur export **dihapus**, bukan ditunda |
| **Bab ini disimpan** | sebagai catatan, supaya penelusurannya tidak perlu diulang bila fitur ini kelak dihidupkan |
| **Tidak ada yang perlu dikirimkan** | permintaan di §2.5 dan §2.6 **tidak lagi berlaku** |

> **Bab ini ditulis ulang pada 2026-09-24.** Bentuk sebelumnya meminta DBA meng-*expose*
> sepuluh properti Pega menjadi kolom. **Permintaan itu DICABUT** — ternyata tidak perlu.
> Lihat §2.3.

### 2.1 Apa yang belum dapat kami bangun

Tombol **"Export Data Investigation"** pada layar Inbox Investigator. Ia nyata dan terlihat di
aplikasi lama, dan menghasilkan berkas CSV berisi rincian hasil investigasi.

Layar daftarnya sendiri **sudah selesai** dan tidak menunggu apa pun. Yang tertahan hanya
tombol export ini.

### 2.2 Apa yang sudah ditemukan

Kami mula-mula menyimpulkan datanya terkubur di dalam data internal aplikasi lama. **Itu
keliru.** Dua kueri katalog dari DBA membuktikan ada tabel biasa yang menyimpannya:

**`POOLDATA.INVESTIGATIONREPORT` — 36 kolom, berkunci `CASEID`:**

```
IDX              NUMBER(22)      wajib      NOTE             VARCHAR2(4000)
CASEID           VARCHAR2(50)    wajib      NOTEANALYST      VARCHAR2(4000)
NAMA             VARCHAR2(100)              HEADNOTE         VARCHAR2(4000)
POLICYNO         VARCHAR2(50)              SLA              NUMBER(22)
REGNO            VARCHAR2(50)              TGLACCHEAD       DATE
COMPANY          VARCHAR2(100)             STSINVEST        VARCHAR2(50)
HOSPITAL         VARCHAR2(100)             ISPATIENTREGIST  CHAR(1)
ADDRESS          VARCHAR2(200)             PATIENTDESC      VARCHAR2(4000)
TGLKEJADIAN      DATE                      ISDATECORRECT    CHAR(1)
TOTALBIAYA       NUMBER(22)                DATEDESC         VARCHAR2(4000)
TGLINVEST        DATE            wajib     ISBILLCORRECT    CHAR(1)
TGLKEMBALI       DATE                      BILLDESC         VARCHAR2(4000)
TGLTELP          DATE                      ISBILLPAID       CHAR(1)
PIC              VARCHAR2(100)             BILLPAIDDESC     VARCHAR2(4000)
TELPRS           VARCHAR2(50)              ISADDRESSCORR    CHAR(1)
INVEST           VARCHAR2(100)             ADDRESSDESC      VARCHAR2(4000)
ANALYST          VARCHAR2(100)             PAIDBY           VARCHAR2(100)
                                           STSKWITANSI      CHAR(1)
                                           KWITANSIDESC     VARCHAR2(4000)
```

Strukturnya sendiri sudah menjelaskan isinya: **enam pasang tanya-jawab**, masing-masing satu
penanda `CHAR(1)` dan satu keterangan panjang.

```
ISPATIENTREGIST / PATIENTDESC      pasien terdaftar?
ISDATECORRECT   / DATEDESC         tanggal benar?
ISBILLCORRECT   / BILLDESC         tagihan benar?
ISBILLPAID      / BILLPAIDDESC     tagihan sudah dibayar?
ISADDRESSCORR   / ADDRESSDESC      alamat benar?
STSKWITANSI     / KWITANSIDESC     status kwitansi
```

Ditambah `INVEST`, `ANALYST`, `SLA`, `TGLINVEST`, `TGLACCHEAD`, dan `STSINVEST` — alur
investigasi dari penugasan sampai persetujuan kepala.

Dari 36 tabel bernama survei/investigasi pada katalog, **hanya ini** yang bernama investigasi;
sisanya seluruhnya survei dan foto.

### 2.3 Yang SUDAH TIDAK kami minta lagi

| Permintaan lama | Keadaan |
|---|---|
| Expose sepuluh properti Pega menjadi kolom | **DICABUT** — datanya sudah ada di tabel biasa |
| DDL tabel penampung | **SUDAH DITERIMA** — §2.2 |
| Kueri katalog pencarian kolom | **SUDAH DIJALANKAN** — hasilnya yang menemukan tabel ini |

### 2.4 Sisa penghalangnya satu: pemetaan

Berkas CSV aplikasi lama diisi **sepuluh isian**. Kami dapat menduga sebagian besar padanannya,
tetapi **tidak akan menebak** — salah petakan berarti nomor rekam medis pasien muncul di kolom
yang bukan tempatnya.

| Isian pada berkas CSV lama | Kolom yang kami duga | Keyakinan |
|---|---|---|
| Status pasien terdaftar | `ISPATIENTREGIST` | kuat |
| Konfirmasi model kwitansi | `STSKWITANSI` | kuat |
| Alamat rumah sakit / klinik | `ADDRESS` | kuat |
| Catatan investigasi | `NOTE` | kuat |
| Nomor telepon yang dihubungi | `TELPRS` | kuat |
| Penanda tidak ada pembayaran | `ISBILLPAID` | sedang |
| Penanda asuransi lain | `PAIDBY` | sedang |
| Penanda pasien | bertabrakan dengan `ISPATIENTREGIST` | **lemah** |
| **Nomor rekap medis** | **tidak ada kolomnya** | — |
| **Pilihan Rumah Sakit / non-Rumah Sakit** | **tidak ada kolomnya** | — |

Kemiripan nama bukan bukti. Kami pernah keliru persis karena itu pada layar yang sama: sebuah
kolom bernama "Lama Masuk Inbox" ternyata berisi tanggal survei, bukan lama menunggu.

### 2.5 Tiga pertanyaan — jalur tercepat, ditujukan ke Work Owner

Ketiganya dapat dijawab oleh siapa pun yang memakai formulir investigasi, tanpa perlu membuka
sistem lama:

1. **Dari keenam pasang pertanyaan pada §2.2** — mana yang di layar lama berlabel
   **"Asuransi Lain"**, mana yang **"Pasien"**, dan mana yang **"Tidak Ada Pembayaran"**?
2. **Nomor rekap medis disimpan di mana?** Tidak ada kolomnya di tabel ini. Apakah ia
   ditumpangkan pada salah satu kolom keterangan, atau memang tidak lagi dipakai?
3. **Apa arti pilihan "Select RS"** — Rumah Sakit versus bukan Rumah Sakit? Apakah cukup
   dilihat dari kolom `HOSPITAL` terisi atau kosong?

Begitu ketiganya terjawab, **export dapat langsung dibangun** tanpa permintaan lain.

### 2.6 Bila §2.5 tidak cukup — barulah ke Tim Pega

Yang diminta: **rule yang menulis dan membaca `POOLDATA.INVESTIGATIONREPORT`**, beserta
**Section formulir investigasi**-nya.

Keduanya memperlihatkan pemetaan isian-ke-kolom secara langsung, tanpa perlu diduga.

Kami sudah memastikan keduanya **tidak ada** di export yang kami pegang: nama tabelnya nol
kemunculan di seluruh 2.634 berkas, dan sepuluh nama kolomnya pun nol. Ini sejalan dengan
`R-16` — sekitar 242 rule dirujuk tetapi tidak ikut terkirim.

### 2.7 Yang TIDAK perlu dikirim

**Jangan kirimkan isi barisnya.** Tabel ini memuat data medis nasabah — nama pasien, alamat
rumah sakit, dan nilai tagihan. Struktur tabelnya sudah cukup bagi kami, dan sudah diterima.

### 2.8 Satu hal untuk pemilik bisnis, di luar pemetaan

Export lama membaca daftar klaimnya dengan parameter **`Operator = 0`**, bukan antrean
`InvestigatorPNC` seperti yang tampil di layar.

Artinya **berkas CSV-nya dapat memuat klaim yang tidak ada di daftar layar**. Apakah itu
disengaja tidak dapat kami buktikan dari export, dan sebaiknya dipastikan sebelum export
dibangun — supaya berkas baru tidak diam-diam berbeda isinya dari berkas lama.

---

## 3. Inbox Receive TKA — `MENU_ID 49`

Ditujukan ke **DBA** dan **Tim Pega**. Berbeda dari §1 dan §2: **layarnya sudah berjalan dan
cacah barisnya sudah cocok dengan Pega** — yang diminta di sini adalah izin, kepemilikan, dan
satu jaminan keunikan. Tidak ada satu pun rule yang masih kami tunggu.

### 3.1 Koreksi dari pihak kami — dicatat lebih dulu

Permintaan sebelumnya pada bab ini meminta kunci dan indeks `POOLDATA.T_CLAIM_TKA_H`.
**Permintaan itu ditarik seluruhnya.** Tabel tersebut ternyata tidak dipakai modul ini sama
sekali; kekeliruannya ada pada kami, dan penelusurannya dicatat di
`catatan-pengembangan.md` §46.

Layar ini kini membaca **tabel yang sama dengan Report Definition Pega**, dengan penyaring
yang sama persis — dan cacahnya **tepat sama** dengan jumlah baris pada layar lama.

### 3.2 Hak akses baca — ke DBA

| Objek | Keperluan | Dipakai untuk |
|---|---|---|
| `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` | `SELECT` | seluruh daftar; penyaring `TKA_1`, `TANGGALDOKLENGKAP`, `PYSTATUSWORK` |
| `POOLDATA.T_CLAIM_PNC` | `SELECT` | penyaring kedua tanggal kelengkapan, dan kunci klaimnya |
| `POOLDATA.T_GENERAL` | `SELECT` | kolom **Nama Peserta** — `INSUREDNAME` pada tabel kerja terbukti kosong |

Akses baca ke tabel engine Pega **aman** dan tidak mengganggu apa pun. Yang tidak aman adalah
menulisnya — lihat §3.3.

### 3.3 Kewenangan menulis satu kolom — ke DBA dan Work Owner

Yang diminta: **`UPDATE` pada `POOLDATA.T_CLAIM_PNC.TGLDOKLENGKAP` saja.**

Ini menempuh `D-63`: permintaan tertulis dari tim pengembang, persetujuan Work Owner,
pelaksanaan DBA. Sejalan dengan `P-1`, kepemilikan tulis kolom itu berpindah ke aplikasi baru
selama masa paralel; Pega hanya membacanya.

**Kami TIDAK meminta izin menulis `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`, dan tidak akan
memintanya.** Pega menyimpan nilai sebenarnya di BLOB kasus dan menyalinnya ke kolom;
menulis kolomnya dari luar berarti nilainya tertimpa tanpa satu pun tanda begitu Pega
menyimpan kasus itu lagi — dan pekerjaan yang tampil di layar ini seluruhnya masih berjalan
(`PYSTATUSWORK = 'New'`).

### 3.4 Akibat yang perlu diketahui Tim Pega — selisih tampilan selama masa paralel

Karena kolom Pega tidak ditulis, sebuah klaim yang sudah dilengkapi lewat aplikasi baru
**masih akan tampil sebagai belum lengkap pada layar TKA di Pega** sampai Pega
menyinkronkannya sendiri.

Ini diterima secara sadar dan dibuat **terlihat**, bukan disembunyikan:

| Tempat | Yang dilakukan |
|---|---|
| `claimpnc -periksa` | mencacah selisihnya lewat `CountPegaOnly` |
| kaki layar | menyebutkannya kepada pengguna |
| `TestPegaWorkTableIsNeverWritten` | menggagalkan build bila ada yang menambahkan penulisannya |

Yang kami minta ke Tim Pega hanyalah **konfirmasi bahwa selisih ini dapat diterima** selama
masa paralel, dan — bila ada — keterangan seberapa sering Pega menyegarkan kolom itu dari
BLOB.

### 3.5 Satu jaminan yang masih kami butuhkan

**Apakah `POOLDATA.T_CLAIM_PNC.CLAIMID` dijamin unik?**

Submit mencari barisnya lewat nomor klaim, lalu menolak bila hasilnya lebih dari satu
(`ErrClaimAmbiguous`). Penolakan itu adalah pengaman, bukan perilaku yang diinginkan. Bila
ada constraint uniknya, kami dapat menyederhanakannya; bila tidak ada, pengamannya tetap
dipasang dan kami mencatatnya sebagai utang.

Kunci baris pada daftar sendiri sudah aman: `PZINSKEY` dijamin unik oleh Pega.

### 3.6 Yang TIDAK perlu dikirim

| Hal | Alasan |
|---|---|
| Rule `InboxTKA_Harness`, `InboxTKA_Section`, `InboxTKA_RD` | sudah ada di export dan sudah terbaca utuh |
| `SubmitTanggalLengkapTKA` dan `NotificationKelengkapanTKA` | sudah ada; isi surel sudah dipetakan |
| `SendEmailNotification` | **bawaan Pega** (`Pega-IntegrationEngine`), bukan rule buatan sendiri |
| Kunci dan indeks `POOLDATA.T_CLAIM_TKA_H` | ditarik — lihat §3.1 |

---

## 4. Monitoring SLINK OJK — `MENU_ID 78`

| | |
|---|---|
| **Modul** | `internal/monitoringslinkojk` · `src/modules/monitoring-slink-ojk` |
| **Status modul** | **SEBAGIAN** — kedua segmen dapat dipantau dan diunduh; tiga tombol tulis belum dapat dibangun |
| **Ditujukan ke** | Tim Pega (butir 1–3) · Work Owner (butir 4–5) |

### 4.1 Artefak yang diminta ke Tim Pega

#### (1) Connect REST `Rest_SendDataClientBasedDebitur` — MEMBLOKIR tombol "SLIK OJK"

`Activity/InsertDataSlinkOJKIndividu-Act.xml` memanggilnya lewat langkah `Connect-REST`
untuk mengirim data debitur ke sistem SLIK, lalu menulis `id_transaction` kembali ke
`pooldata.t_claim_slink_individu`.

Berkasnya **nol kemunculan** di direktori `Connect REST/` (21 berkas, dihitung langsung).

Yang dibutuhkan: endpoint, metode, bentuk badan permintaan dan jawabannya, kode galat, cara
autentikasinya, dan batas waktunya. Tanpa kontrak itu, yang dapat dibangun hanyalah tebakan
tentang apa yang dikirim ke sistem luar.

> **Status ketiga tombol tulis berubah pada 2026-09-26.** Ketiganya semula dinyatakan tidak
> dapat dibangun; pemeriksaan ulang membuktikan logikanya ADA, hanya tidak di section.
> **"Proses Data Klaim" dan "Upload Data Klaim" kini berjalan penuh.** Yang tersisa dari bab
> ini tinggal butir (1) — kontrak REST — beserta butir (2) yang lingkupnya menyempit.

#### (2) Aksi lokal `UploadDataSlinkOJK` — hanya memengaruhi segmen F06

`Sec_SegmentF06-Section.xml` memanggilnya sebagai aksi lokal. Rule-nya **tidak ada di
export**. Bandingkan dengan padanannya di segmen D01, `PNCUploadDataKlaimSlikOJK`, yang
ADA — sehingga ketiadaan yang satu ini tampak sebagai gap, bukan sebagai rancangan.

**Lingkupnya kini menyempit.** Unggahan segmen D01 sudah berjalan, memakai pemetaan dari
`PNCUploadAutoClaimSlikOJK`. Yang belum diketahui hanyalah apakah unggahan segmen F06
memuat kolom yang BERBEDA — bila sama, rule ini tidak dibutuhkan sama sekali.

#### (3) Daftar pilihan dropdown "Tipe Generate" — Rule-Obj-FieldValue

Isian `.GenerateType` ada di `Sec_SegmentD01_1` sebagai dropdown, tetapi daftar pilihannya
tidak ada di export, dan **tidak satu pun kueri maupun aktivitas membacanya**.

Yang dibutuhkan: daftar pilihannya, DAN jawaban atas pertanyaan yang lebih penting —
**apakah isian itu memang menyaring sesuatu?** Bila ya, aturannya belum pernah terbaca.
Untuk sementara isiannya diterima dan diteruskan, tetapi tidak menyaring apa pun.

### 4.2 Pertanyaan ke Work Owner

#### (4) Berkas ekspor F06 di produksi — apakah memang hampir seluruhnya kosong?

`ExportDataSlinkFOG` menulis **34 nama properti** yang seluruhnya properti **CIF nasabah**
(`ASMClientID`, `pyFullName`, `ASMNIK`, `SpouseName`, …). Halaman yang diekspornya,
`AllDataObjectSlink`, diisi `GetDataSlinkAllFOG` — yang menghasilkan alias yang **berbeda**
(`CUSTOMERTYPE`, `OBJECTGENDER`, `DATEOFBIRTH`, `ASMZIPCODE`, `TELFAXNUMBER`, …).

Hanya tiga nama yang berpadanan: `ASMAddress`, `KodeKantorCabang`, `OperasiData`.

Bacaan yang paling sesuai bukti: **berkas ekspor F06 hari ini hampir seluruhnya kosong**.
Itu **belum dipastikan** dan perlu diuji terhadap satu berkas ekspor produksi yang
sungguhan — bukan disimpulkan dari kode saja.

Bila benar, pertanyaan lanjutannya: apakah berkas itu memang dipakai melapor ke OJK, atau
pelaporan F06 sebenarnya menempuh jalur lain?

#### (5) ~~Dua selisih terencana yang menunggu persetujuan~~ — **TERJAWAB 2026-09-26**

Keduanya sempat diajukan sebagai selisih terencana (`D-54`). **Work Owner memilih
"samakan persis dengan Pega"**, sehingga keduanya **dicabut** dan tidak lagi menjadi
selisih pada uji kesetaraan gerbang 1.

| # | Semula diusulkan | Keputusan |
|---|---|---|
| a | Ekspor F06 dibuat sejajar 38↔38 | **direplikasi misalign** — 38 judul, 34 kolom data |
| b | Nama berkas ekspor dibetulkan | **direplikasi tertukar** — ekspor D01 bernama "Laporan F06 SLIK OJK", dan sebaliknya |

Keberatan sudah disampaikan sebelum jawaban diminta, beserta akibat paling konkretnya:
nilai **Alamat** tercetak di bawah judul **"Jenis Kelamin"**, kode cabang di bawah
**"Perjanjian Pisah Harta"**, operasi data di bawah **"Melanggar BMPK"**. Keputusan
ditegaskan dan dijalankan penuh. Rinciannya di `catatan-pengembangan.md` §57 dan
`keputusan-implementasi.md` §58.

**Butir 4 di atas karena itu menjadi lebih penting, bukan kurang.** Selama bentuk berkas
produksi belum pernah dilihat, tidak ada yang dapat memastikan replikasi ini benar-benar
setara — yang direplikasi adalah **rule sebagaimana tertulis**, bukan keluaran yang sudah
diperiksa.

Dua cacat lain **direplikasi** dan tidak menuntut persetujuan, tetapi dicatat sebagai calon
butir `P-5`:

- Kolom "Tanggal Pembayaran" terisi `tanggalkondisi` — tabelnya tidak menyimpan tanggal
  pembayaran sama sekali.
- Kode kantor cabang `'001'` sebagai konstanta di dalam kueri — melanggar `D-15`; tempatnya
  master Cabang pada `F-4`. Sudah diangkat menjadi parameter repo supaya ketika masternya
  tiba, yang berubah hanya satu tempat.

#### (6) Bagaimana tombol "SLIK OJK" memilih klaimnya — BELUM dapat dipulihkan

Di Pega, `InsertDataSlinkOJKIndividu` bekerja atas **satu klaim yang sedang dibuka**
(`pyWorkPagee.ClaimData.ClaimNo`). Layar pemantauan ini tidak punya klaim yang sedang
dibuka, dan `Sec_SegmentD01_1` **tidak menghubungkan tombol ini ke aktivitas mana pun** —
sehingga cara klaimnya dipilih di layar ini tidak dapat dibaca dari export.

**Yang dipakai sementara:** tombolnya mengirim seluruh klaim yang cocok dengan penyaring —
sasaran yang sama dengan "Proses Data Klaim", sehingga pelapor menekan "Cari Data" untuk
melihat apa yang akan dikirim lebih dulu.

**Yang ditanyakan:** apakah itu memang perilaku yang dimaksud, atau tombolnya semestinya
mengirim satu klaim yang dipilih dari baris grid? Bila yang kedua, layarnya menuntut kolom
aksi per baris yang tidak ada di layar lama (`D-13`), dan itu keputusan tersendiri.

Rutenya sudah menerima kedua bentuk, sehingga jawabannya tidak menuntut perubahan besar.

#### (7) Satu selisih yang BELUM pernah ditanyakan — batas baris ekspor

`MaxRecords` pada `GetTempDataD01` dan `GetAllDataSumbisSlink` **kosong**: Pega tidak
membatasi jumlah baris ekspor sama sekali. Modul ini memasang batas **200.000 baris**
sebagai penjaga terhadap `D-10` (puluhan juta baris), dan berkas yang menyentuhnya diberi
**baris penanda** — bukan dipotong diam-diam seperti `pyMaxRecords=500` pada laporan lain.

Ia hanya menggigit di atas 200.000 baris. Bila Work Owner menghendaki tanpa batas seperti
Pega, yang berubah satu konstanta di `http/export.go`.

### 4.3 Yang TIDAK diminta, dan alasannya

DDL `POOLDATA.T_CLAIM_SLIK_OJK` tidak diminta terpisah — ia bagian `R-08` yang sudah
tercatat. Ketiadaannya **tidak memblokir** modul ini: nilai kolom dibaca sebagai `any` lalu
diformat di satu tempat, sehingga kolom bertipe TEKS maupun TANGGAL sama-sama terlayani.

Yang akan berubah ketika DDL-nya tiba hanyalah fungsi `text` di
`repo/sqlstore/monitoringslinkojk.go` — dan hanya bila ternyata bentuk keluarannya perlu
disesuaikan.

---

## 5. Case Study Claim — `MENU_ID 74`

Modulnya **sudah dibangun dan dapat dijalankan**. Yang di bawah bukan penghalang, melainkan
hal yang perlu diputuskan atau dikonfirmasi supaya modul ini tidak menyimpan asumsi diam.

### 5.1 Yang perlu DIPUTUSKAN Work Owner — satu butir, dan ia menyangkut kewenangan menulis

**Serah-terima kepemilikan tulis atas SATU kolom** (`D-63`):

    UPDATE POOLDATA.T_CLAIM_PNC SET REMARKRECOMENDATION = … WHERE CLAIMNO = …

Tabelnya milik sistem lama, dan `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem.
Presedennya sudah berjalan — modul Inbox Receive TKA menulis `TGLDOKLENGKAP` pada tabel yang
sama — sehingga yang dibutuhkan bukan keputusan baru melainkan **perluasan satu kolom** pada
serah-terima yang sudah ada.

Prosedurnya menempuh tiga pihak: permintaan tertulis tim pengembang, persetujuan Work Owner,
pelaksanaan DBA.

Work Owner sudah menyetujui pembangunannya pada 2026-09-26; yang belum adalah langkah
formal di atas.

### 5.2 Yang perlu DIKONFIRMASI DBA — dua kolom, keduanya bagian `R-08`

Keduanya **tidak memblokir** modul ini. Ia berjalan tanpa keduanya, dan yang berubah ketika
DDL-nya tiba hanya satu tempat masing-masing.

| Kolom | Yang tidak diketahui | Akibat bila dugaannya salah |
|---|---|---|
| `POOLDATA.T_CLAIM_PNC.REMARKRECOMENDATION` | **Lebar kolomnya** | Batas 2.000 karakter di `errors.go` adalah pengaman yang dipilih sadar, bukan hasil pembacaan DDL. Bila lebar sebenarnya lebih kecil, catatan yang lolos pemeriksaan kami akan ditolak Oracle dengan ORA-12899 — dan pengguna melihatnya sebagai "terjadi kesalahan sistem" setelah mengetik satu halaman penuh |
| `POOLDATA.PEGA_DASHBOARDPNC.THNREGIS` | **Tipenya** — VARCHAR2 atau NUMBER | Ia yang dibandingkan penyaring rentang. Kueri membandingkannya terhadap TEKS, persis seperti kueri lama membandingkannya terhadap keluaran `TO_CHAR`; bila ia NUMBER, Oracle mengubah teksnya dan hasilnya sama. Bila ia bertipe lain sama sekali, penyaringnya gagal — dan `claimpnc -periksa` yang akan menunjukkannya |

Satu kueri katalog menjawab keduanya:

    SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE, DATA_LENGTH, DATA_PRECISION, DATA_SCALE
      FROM ALL_TAB_COLUMNS
     WHERE OWNER = 'POOLDATA'
       AND ( (TABLE_NAME = 'T_CLAIM_PNC'      AND COLUMN_NAME = 'REMARKRECOMENDATION')
          OR (TABLE_NAME = 'PEGA_DASHBOARDPNC' AND COLUMN_NAME = 'THNREGIS') );

### 5.3 Yang perlu DIKONFIRMASI tim bisnis — satu temuan, bukan permintaan artefak

**Cakupan NONMBU dan BONDING pada dropdown Bisnis BERTUMPANG TINDIH.**

    NONMBU   panel 003/004/006/009 MINUS lima kode bisnis
    BONDING  panel 003 DAN sepuluh kode bisnis tertentu

Kesepuluh kode BONDING tidak ada di daftar lima yang dikecualikan NONMBU, sehingga **setiap
klaim Bonding muncul pada kedua pilihan**. Menjumlahkan hasil keempat pilihan karena itu
menghitung sebagian klaim dua kali.

Ini perilaku kueri Pega apa adanya, sudah direplikasi (`P-5`), dan **tidak menuntut
perubahan**. Yang diminta hanyalah konfirmasi: apakah tumpang tindih itu memang dikehendaki,
atau ia cacat lama yang selama ini tidak pernah terlihat karena tidak ada yang
membandingkan keempat pilihan berdampingan.

### 5.4 Yang TIDAK diminta, dan alasannya

**Isi master kode bisnis.** Kedua daftar kode — lima yang dikecualikan NONMBU dan sepuluh
yang membentuk BONDING — di-hardcode di dalam rule Pega, dan `D-15` menetapkannya menjadi
master data (`F-4`). Master itu belum ada, dan keduanya tinggal di berkas `.sql` tempat
pemetaannya ke kueri lama dapat dibaca berdampingan. Memintanya sekarang berarti meminta
sesuatu yang belum punya tempat untuk disimpan.

**Ambang Rp 5.000.000.000.** Sama halnya: ia menunggu master ambang pada `F-4`, dan sudah
diangkat menjadi satu nilai bernama yang terlihat serta dikirim ke layar — sehingga ketika
masternya tiba, yang berubah hanya dari mana nilai itu dibaca.

**Section grid `BrowseClaimStudy`.** Tidak ada, dan tidak dibutuhkan: keduapuluh empat judul
kolom beserta properti pengisinya terbaca lengkap dari
`Section/PNCStudyClaim-Section.xml` — berpasangan satu-satu, 24 lawan 24.

### 5.5 Keputusan Work Owner 2026-09-26 — dua dari tiga butir DITUTUP

Ketiga butir di atas dibawa ke Work Owner. Jawabannya:

> "2 dan 3 biarin aja seperti PEGA, nomor 1 tinggal minta DBA"

| Butir | Status | Yang berlaku sekarang |
|---|---|---|
| §5.1 Serah-terima kepemilikan tulis | **TERBUKA** — satu-satunya yang tersisa | Permintaan tertulis ke DBA; teksnya siap kirim di §5.6 |
| §5.2 DDL dua kolom | **DITUTUP** — tidak dikejar | Modul berjalan tanpa DDL. Lihat akibatnya di bawah |
| §5.3 Tumpang tindih NONMBU/BONDING | **DITUTUP** — direplikasi | Tidak ada perubahan kode: itu memang yang sudah dibangun (`P-5`) |

**§5.3 tidak menuntut satu baris pun berubah.** Modul sejak awal mereplikasi tumpang tindih
itu apa adanya, dan `TestCakupanNONMBUDanBONDINGBertumpangTindih` menjaganya tetap begitu.
Yang berubah hanyalah statusnya: dari "menunggu konfirmasi" menjadi "sudah dikonfirmasi".

**§5.2 menuntut satu penambahan, dan alasannya perlu dibaca.**

Menutup butir ini berarti `MaxRemarkLength = 2000` tetap menjadi **dugaan**. Pega sendiri
tidak membatasi apa pun — `pyMaxLength` **nol kemunculan** di
`Section/PNCStudyClaim-Section.xml` — sehingga "seperti Pega" berarti batas sebenarnya
hanyalah lebar kolom di basis data.

Bila dugaan itu terlalu longgar, Oracle menolaknya dengan **ORA-12899**, dan tanpa
penanganan khusus galat itu sampai ke pengguna sebagai *"terjadi kesalahan pada sistem"* —
setelah ia mengetik satu halaman penuh, tanpa satu pun petunjuk bahwa yang salah hanyalah
panjangnya.

Karena itu ditambahkan:

| Yang ditambahkan | Berkas |
|---|---|
| `ErrRemarkRejectedByColumn` — galat domain tersendiri, dibedakan dari `ErrRemarkTooLong` | `errors.go` |
| `isValueTooLarge` — mengenali ORA-12899 lewat NOMORNYA, bukan teks pesannya | `repo/sqlstore/casestudyclaim.go` |
| Pemetaan ke **422** menunjuk isian `catatan`; nomor galat Oracle hanya masuk log | `http/errors.go` |
| Tiga uji yang mengunci ketiganya | `query_test.go`, `export_test.go` |

Akibatnya: DDL-nya boleh tidak pernah datang, dan pengguna tetap diberi tahu apa yang salah.
Kemunculan galat ini di log sekaligus menjadi **bukti langsung** bahwa 2.000 terlalu longgar
— satu-satunya cara mengetahuinya selama DDL belum ada.

### 5.6 Permintaan ke DBA — siap kirim

> **Perihal:** Permintaan kewenangan tulis satu kolom — modul Case Study Claim (`MENU_ID 74`)
>
> Modul Case Study Claim pada aplikasi Claim PNC yang baru menggantikan harness Pega
> `PNCStudyClaim`. Salah satu fungsinya — tombol **Save** pada kolom *Remark* di setiap baris
> — menyimpan catatan hasil telaah, persis seperti di Pega.
>
> Di Pega, penyimpanan itu dikerjakan `SaveRemarksRecommendation_act` lewat pernyataan:
>
> ```sql
> UPDATE POOLDATA.T_CLAIM_PNC
>    SET REMARKRECOMENDATION = :1
>  WHERE CLAIMNO = :2
> ```
>
> **Yang diminta:** kewenangan `UPDATE` bagi akun aplikasi atas **satu kolom** —
> `POOLDATA.T_CLAIM_PNC.REMARKRECOMENDATION`. Tidak ada kolom lain yang ditulis modul ini,
> dan tidak ada `INSERT` maupun `DELETE`.
>
> **Dasar:** `D-63` menetapkan perubahan kewenangan menempuh permintaan tertulis tim
> pengembang, persetujuan Work Owner, dan pelaksanaan DBA. Persetujuan Work Owner sudah
> diberikan pada 2026-09-26.
>
> **Preseden:** modul Inbox Receive TKA sudah menulis kolom `TGLDOKLENGKAP` pada tabel yang
> sama, dengan pagar yang sama.
>
> **Pagar yang sudah terpasang di aplikasi:**
>
> - Pernyataan tulisnya **satu-satunya** di seluruh modul, dan dijaga uji otomatis yang
>   menolak pernyataan tulis kedua masuk tanpa disadari.
> - Jumlah baris terpengaruh **diperiksa**: lebih dari satu baris menjadi galat, bukan lewat
>   begitu saja — karena keunikan `CLAIMNO` belum dibuktikan DDL mana pun.
> - Setiap penyimpanan **dicatat di log** beserta pelakunya, portalnya, dan nomor klaimnya.
>   Isi catatannya tidak ikut dicatat.
>
> **Sekaligus, bila memungkinkan** — tidak menghalangi, dan boleh diabaikan: lebar kolom
> `REMARKRECOMENDATION`. Aplikasi memakai batas 2.000 karakter sebagai pengaman yang dipilih
> sendiri; bila lebar sebenarnya lebih kecil, kiriman yang melampauinya tetap ditolak dengan
> pesan yang benar, hanya saja penolakannya terjadi di basis data dan bukan di layar.
>
> ```sql
> SELECT DATA_TYPE, DATA_LENGTH
>   FROM ALL_TAB_COLUMNS
>  WHERE OWNER = 'POOLDATA'
>    AND TABLE_NAME = 'T_CLAIM_PNC'
>    AND COLUMN_NAME = 'REMARKRECOMENDATION';
> ```

## 6. Archive Dokumen Klaim — `MENU_ID 77`

Modulnya **sudah dibangun dan berjalan** dengan rekonstruksi di tempat yang artefaknya
hilang. Yang diminta di bawah bukan penghalang pekerjaan, melainkan hal-hal yang membuat
rekonstruksi itu dapat diganti dengan yang sebenarnya — dan satu hal yang memblokir satu
fungsi.

### 6.1 Koreksi dari pihak kami — dicatat lebih dulu

`Section/KodeArchiveDoc-Section.xml` **bukan** berisi rule bernama `KodeArchiveDoc`. Isinya
rule bernama **`SecCariKodeArchiveDoc`** (`pzDocumentKey: RULE-HTML-SECTION DATA-PORTAL
SECCARIKODEARCHIVEDOC`).

Nama berkas dan nama rule berbeda. Ini kasus kedua setelah dua berkas yang sudah tercatat
pada `D-39`, dan ia menguatkan ketetapan yang sama: **inventaris rule dibangun dari
`pyRuleName`, bukan dari nama berkas.**

### 6.2 Yang memblokir — ke DBA

**Satu baris di `POOLDATA.GCNM_CONNECT_REST`.**

Fungsi "Kirim ke Cabang" mengirim berkas arsip ke sistem Arsip lewat REST. Alamatnya
dibaca dari tabel katalog layanan — cara yang sama dengan alamat login HCQ yang sudah ada
di sana — dengan:

| Kolom | Nilai |
|---|---|
| `APP` | alias portal entitas (`ASM`, `SMI`, …) |
| `TYPESERVICE` | **`ARCHIVE-INJECT`** |
| `SERVICENAME` | alamat lengkap endpoint injeksi arsip |

Alamatnya **tidak kami tuliskan di sini** sesuai `D-69`; ia ada di rule
`Connect REST/InjectDataArchiveDokumentKlaim-ConnectREST.xml` pada elemen `pyBaseURL` dan
`pyResourcePath`, dan Tim Infra sudah memegangnya lewat dokumen serah-terima terpisah.

Sampai barisnya ada, pengiriman gagal dengan **503** dan pesan yang menyebut tepat apa yang
kurang. Ketiga fungsi lain modul ini berjalan normal.

**Satu pertanyaan yang menyertainya, ke Tim Infra/Security:** rule lamanya
ber-`pyUseAuthentication=false`. Apakah layanan Arsip memang tanpa autentikasi, atau
autentikasinya ditangani di lapisan lain? Bila ia menuntut kredensial, kami perlu tahu
bentuknya sebelum barisnya dipasang.

### 6.3 DDL `POOLDATA.T_CLAIM_ARCHIVE_FILE` — ke DBA

Tabelnya **tidak punya DDL di export** (`R-08`), dan tiga hal bergantung padanya:

| Yang tidak diketahui | Akibatnya hari ini |
|---|---|
| Panjang kolom teks | batas panjang isian kami pasang longgar sebagai pengaman, bukan sebagai cerminan skema |
| Nilai bawaan `CABANGSTATUS` | kami menulisnya **`'0'` eksplisit**; lihat §6.4 |
| Ada tidaknya constraint unik | tidak diketahui apakah basis data menahan ID ganda; lihat §6.5 |

Yang diminta: `DBMS_METADATA.GET_DDL('TABLE','T_CLAIM_ARCHIVE_FILE','POOLDATA')`.

### 6.4 Prosedur di export lebih tua daripada pemanggilnya — ke Tim Pega

`Database/INSERTDATASFILLINGARCHIVE.prc` menerima **15 parameter + ErrMsg**.
`RDB List/InsertToClaimArchive-SQL.xml` memanggilnya dengan **17 + out**, menambahkan
`TKODECABANG` dan `TGROUPPANEL`.

Salah satu dari keduanya bukan versi yang berjalan di produksi. Yang kami ikuti adalah
**pemanggilnya**, karena `GROUPPANEL` menentukan siapa melihat barisnya di daftar kirim ke
cabang.

**Yang diminta:** source prosedur yang benar-benar terpasang di produksi hari ini —
`DBMS_METADATA.GET_DDL('PROCEDURE','INSERTDATASFILLINGARCHIVE','POOLDATA')`.

**Yang perlu dikonfirmasi bersamaan:** prosedur di export **tidak menulis `CABANGSTATUS`
sama sekali**. Bila bawaan kolomnya bukan `'0'`, maka di sistem lama **setiap berkas yang
baru diarsipkan tidak pernah muncul di daftar pengiriman ke cabang** — dan tidak pernah
sampai ke sistem Arsip. Apakah itu keadaan yang sedang berjalan?

### 6.5 Penomoran ID — ke Work Owner, diajukan kembali

Prosedurnya memberi nomor dengan `select max(ID_ARCHIVE) into counts_id ... +1`. Dua
penyimpanan yang berjalan bersamaan membaca angka yang sama, dan yang kedua **menimpa**
baris yang pertama. Tidak ada galat dan tidak ada jejak; yang terjadi hanyalah satu berkas
arsip hilang.

Work Owner memutuskan 2026-09-25: **replikasi apa adanya** (`P-5` murni). Keputusan itu
dijalankan, dan perilakunya kini sama persis dengan sistem lama.

**Yang kami mintakan sebagai tindak lanjut, bukan sebagai bantahan:**

1. **Ke DBA** — apakah `T_CLAIM_ARCHIVE_FILE` punya constraint unik pada `ID_ARCHIVE`?
   Bila ya, penyimpanan kedua akan **gagal dengan galat** alih-alih menimpa diam-diam, dan
   akibatnya jauh lebih ringan daripada yang kami khawatirkan.
2. **Ke DBA** — berapa baris yang ID-nya pernah tertimpa? Dapat diperiksa dengan
   membandingkan `COUNT(*)` dan `COUNT(DISTINCT ID_ARCHIVE)`.
3. **Ke Work Owner** — bila jawaban (1) "tidak ada constraint" dan (2) menunjukkan baris
   yang hilang, apakah keputusan replikasi ditinjau ulang? Penggantinya satu sequence, dan
   perubahannya menempuh `D-63`.

### 6.6 Saringan lini bisnis yang tampak terbalik — ke Work Owner dan pemilik bisnis

`Activity/GetDataArchiveCabangKlaim-Act.xml` menyaring daftar kirim ke cabang menurut
`OperatorID.pyPosition`:

| Jabatan | Klausa | Artinya |
|---|---|---|
| `NONMBU` | `GROUPPANEL not in ('002','005')` | tidak melihat PA maupun Travel |
| `PA` | `GROUPPANEL not in ('002')` | **tidak melihat Personal Accident** |
| `TRAVEL` | `GROUPPANEL not in ('005')` | **tidak melihat Travel** |
| lainnya | — | melihat seluruh lini |

Pasangan prakondisi dan nilainya sudah kami periksa ulang dengan posisi byte, dan ia benar
— jadi ini bukan salah baca.

Direplikasi apa adanya (`P-5`), diuji, dan diumumkan di layar supaya berkas yang hilang
dari daftar tidak dilaporkan sebagai kerusakan modul.

**Pertanyaannya:** apakah petugas PA memang tidak boleh mengirimkan berkas klaim PA ke
sistem Arsip, dan petugas Travel tidak boleh mengirimkan berkas Travel? Bila jawabannya
"seharusnya sebaliknya", perubahannya **satu baris per lini** di kueri kami — tetapi ia
mengubah siapa mengerjakan apa, sehingga tidak kami ambil sendiri.

### 6.7 Pemilih Kode Filling — ke Tim Pega

Tiga rule dirujuk `SecCariKodeArchiveDoc` dan **tidak ada satu pun di export**:

| Rule | Tipe | Perannya |
|---|---|---|
| `SetKodeandSearchArchiveDoc` | Activity | **mengisi daftar kodenya** |
| `CariKodeArchiveDoc` | (tidak diketahui) | dirujuk dari section yang sama |
| — | tabel master | tidak ada satu pun tabel kode arsip di 2.634 berkas |

Ketiganya masuk lingkup export ulang berbasis Product rule (`D-39`), jadi **tidak perlu
permintaan terpisah** — cukup dipastikan ikut terbawa.

Sementara itu daftarnya kami susun dari kode yang sudah pernah dipakai. Bila masternya
kelak tiba, yang berubah **satu kueri bernama** (`filling_codes`), bukan layarnya.

**Satu pertanyaan ke pemilik bisnis:** tombol "Input Kode" dan "Generated Kode" pada layar
lama — apakah keduanya membuat kode baru, dan apakah ada aturan pembentukannya? Bila ada,
kami dapat menggantikan daftar rekonstruksi dengan pembuat kode yang benar.

### 6.8 Yang TIDAK perlu dikirim

- Harness `PNCArchiveDokumen` dan section `SecArchiveDokumen`: sudah dibaca, dan isinya
  hampir seluruhnya boilerplate template.
- Kedua RDB List pencarian klaim: sudah dibaca lengkap.
- `Connect REST/InjectDataArchiveDokumentKlaim`: sudah dibaca; yang kurang hanya baris
  katalognya (§6.2).

### 6.9 Tiga tombol yang wiring-nya tidak dapat ditelusuri — ke Tim Pega dan pemilik bisnis

Harness `PNCArchiveDokumen` dan section `SecArchiveDokumen` memuat **sebelas** tombol.
Delapan sudah kami kenali dan bangun. Tiga sisanya tidak dapat dipetakan ke aksi mana pun:

| Tombol | Yang kami ketahui |
|---|---|
| **Tambah** | hanya labelnya |
| **Update Box** | hanya labelnya; namanya menyiratkan pengubahan nama boks secara massal |
| **Transfer To Pusat** | hanya labelnya; berbeda dari **Transfer To Archive** yang sudah kami bangun |

**Cara kami mencoba menelusurinya, supaya tidak diulang:**

1. Mengambil `<pyActivity>` terdekat di sekitar posisi labelnya — nihil. Definisi label
   tersimpan di bagian belakang section (posisi byte 1.211.574 ke atas), jauh dari tempat
   aksinya dipasang (52.322–1.165.508).
2. Mendaftar **seluruh** `<pyActivity>` di section itu — hasilnya **tujuh**, dan ketujuhnya
   sudah kami kenali: `FlagForArchiveData`, `SearchDataArchiveFilling`,
   `SaveAttachArchiveToDatabase`, `GetDataArchiveCabangKlaim`, `ShowInsertArchiveKlaim_Act`,
   `SetDataArchiveDokumentCase`, `SENDDATACABANGKEARCHIVE`.

Tidak ada activity kedelapan. Jadi ketiga tombol itu **tidak memanggil activity sama
sekali** — kemungkinannya aksi klien murni, atau Flow Action yang tidak ikut terekspor.

**Yang kami minta, salah satu saja sudah cukup:**

- **Ke pemilik bisnis** — apa yang terjadi saat ketiga tombol itu ditekan? Satu kalimat per
  tombol sudah memadai, atau tangkapan layar Pega-nya.
- **Ke Tim Pega** — bila ketiganya memanggil Flow Action, ia termasuk tipe rule yang belum
  pernah diaudit (`R-16`) dan seharusnya ikut pada export ulang berbasis Product rule
  (`D-39`). Cukup dipastikan ikut terbawa.

Sampai salah satunya tiba, ketiga tombol itu **tidak kami bangun**. Menebak fungsinya pada
layar yang menulis ke sistem Arsip milik tim lain bukan risiko yang sepadan.

### 6.10 Dua tombol pemilih Kode Filling — ke pemilik bisnis

`SecCariKodeArchiveDoc` punya **Input Kode** dan **Generated Kode** di samping Cari Kode
dan Pilih. Keduanya menyiratkan kode arsip memang DIBUAT dari layar itu — dan itulah dasar
rekonstruksi daftar kode kami (§6.7).

**Pertanyaannya:** apakah ada aturan pembentukan kodenya — misalnya awalan tahun, nomor
urut per boks, atau kode cabang? Bila ada, kami dapat menggantikan daftar rekonstruksi
dengan pembuat kode yang benar, dan tombol "Generated Kode" menjadi dapat dibangun.

Bila tidak ada aturan dan kodenya memang diketik bebas, cukup dinyatakan begitu — isian
Kode Filling kami sudah menerima ketikan langsung, sehingga tidak ada yang perlu berubah.

### 6.11 Satu perilaku yang perlu dikonfirmasi ke pemilik bisnis

Menyimpan berkas di Pega **langsung mengirimkannya** ke sistem Arsip
(`SaveAttachArchiveToDatabase` langkah 5), tetapi **tidak menandai** `CABANGSTATUS`.
Akibatnya berkas yang sama dikirim **dua kali**: sekali saat disimpan, sekali lagi dari
layar Dokument Cabang yang masih memuatnya.

Work Owner memutuskan 2026-09-25 perilaku itu direplikasi apa adanya, dan sudah kami
terapkan.

**Yang kami mintakan sebagai tindak lanjut, bukan sebagai bantahan:** apakah sistem Arsip
menerima dokumen yang sama dua kali tanpa menghasilkan dua catatan arsip? Bila ia
membuat duplikat, yang terlihat bukan cacat aplikasi kami melainkan dua berkas arsip untuk
satu klaim — dan itu baru ketahuan saat berkas fisiknya dicari.

Ditujukan ke **tim pemilik sistem Arsip**.

### 6.12 Dua kolom penyaring Inbox RCL — ke DBA dan Tim Pega

Layar Inbox RCL (`MENU_ID 62`, `RCL_Harness`) disaring `Report Definition/InboxRCLDokter_RD`
dengan dua properti yang Pega tandai **`unexposed`**:

| Properti | Peran di layar |
|---|---|
| `.ClaimData.TanggalAnalystSendRCL` | penyaring `IS NOT NULL` **dan** kolom "Tanggal Masuk Inbox" |
| `.ClaimData.NamaDokterRCL` | penyaring `= identitas lama pemanggil` |

Kueri katalog 2026-09-27 (portal ASM) membuktikan **keduanya tidak punya kolom** di
`DATAPEGA.PC_ASM_FW_GCNMFW_WORK` — tidak ada satu pun kolom berunsur `DOKTER`/`DOCTOR`.
Yang namanya berdekatan hanya `ANALYSTTRANSFERDATE_1` (TIMESTAMP, statistik 0 nilai), dan
itu properti lain.

**Yang kami minta — pilih salah satu:**

1. **DBA**: mengekspos kedua properti sebagai kolom (`TANGGALANALYSTSENDRCL_1`,
   `NAMADOKTERRCL_1`, mengikuti konvensi `_1`), lalu mengisi ulang dari blob; **atau**
2. **Tim Pega**: menyatakan kolom mana — bila ada — yang memang memuat kedua nilai itu.
   Khususnya: apakah `ANALYSTTRANSFERDATE_1` diisi pada saat yang sama dengan
   `TanggalAnalystSendRCL`?

Sampai dijawab, layar itu **sengaja gagal dimuat** terhadap Oracle dengan galat yang menyebut
kolomnya. Menghilangkan penyaringnya akan menampilkan seluruh worklist pemanggil sebagai
antrean RCL, tanpa pesan galat. `./claimpnc.exe -periksa` melaporkan keadaannya.

**Pembaruan 2026-09-27.** Work Owner menetapkan modul ini membaca `POOLDATA.T_CLAIMLIST_ADMIN`,
bukan tabel Pega. Permintaannya kini:

1. **DBA** — menjalankan `migrations/0012_claimlist_admin_rcl.up.sql` (tiga kolom).
2. **Pemilik proses pengisi `T_CLAIMLIST_ADMIN`** — mengisi ketiganya, dan memuat klaim
   tahap RCL Dokter (hari ini nol).
3. **Tim Pega** — rule mana yang menulis `NamaDokterRCL` dan `TanggalAnalystSendRCL`
   (dugaan `RouterRCLDokter`, hilang dari export), supaya pengisi tahu kapan nilainya sah.

### 6.13 Inbox Service Center — dua artefak, ke Tim Pega dan DBA

Modul `MENU_ID 46` sudah dibangun dan berjalan dari export susulan 2026-09-28. Dua artefak
tetap kurang, dan keduanya menahan hal yang berbeda.

#### a. Kueri grid `GetDataServiceCenter` — ke **Tim Pega**

`Activity/DataServiceCenter-Act.xml` memanggil RDB rule bernama `GetDataServiceCenter` pada
kelas **`ASM-FW-GCNMFW-Data-ClaimData`**.

Yang ikut terekspor adalah rule **bernama sama pada kelas lain** —
`ASM-FW-GCNMFW-Int-T_GENERAL` (`RDB List/GetDataServiceCenter-SQL.xml`, `pzInsKey`
`RULE-CONNECT-SQL ASM-FW-GCNMFW-INT-T_GENERAL GCNM!GETDATASERVICECENTER`). Isinya kueri lain
sama sekali: ia membaca `pooldata.service_log_nonmbu` lewat `JSON_VALUE`, bukan
`T_KLAIM_PORTAL_REKANAN`.

**Yang diminta:** rule `GetDataServiceCenter` pada kelas `ASM-FW-GCNMFW-Data-ClaimData`.

**Dampaknya bila tidak datang — kecil tetapi nyata.** Kuerinya sudah disusun ulang dari tiga
rule sekelas yang ada (`CountDataServiceCenter`, `ExportDataServiceCenter`,
`GetDataServiceCenter_Update`), dan keenam kolom gridnya ada di kedua terakhir. Yang **tidak**
dapat disusun ulang hanyalah klausa `ORDER BY`-nya: dua bukti yang tersedia berbeda —
`order by A.INPUTDATE desc` (langkah 28 activity) dan `ORDER BY ID ASC`
(`ExportDataServiceCenter`). Dipakai yang pertama. Bila rule aslinya datang dan urutannya
berbeda, yang berubah satu klausa.

Sekalian dikonfirmasi: apakah rule bernama sama pada dua kelas berbeda itu disengaja, atau
sisa Save-As yang salah kelas.

#### b. Source `POOLDATA.PEGA_PORTAL_REKANAN` — ke **DBA**

Stored procedure **ber-90 parameter**, dipanggil `RDB List/CallProcServiceCenter-SQL.xml`. Ia
satu-satunya jalur tulis layar ini: menyimpan rincian klaim, memutuskan komite, dan menulis
ketujuh komponen biaya beserta pasangan "approve"-nya.

Sumbernya **tidak ada** di `Database/` — 74 berkas, tidak satu pun untuknya. Ini bagian dari
`R-01`.

**Dampaknya besar:** `D-02` menetapkan logikanya ditulis ulang di Go, dan itu tidak dapat
dilakukan tanpa membacanya. Sampai ia datang, modul Inbox Service Center **MEMBACA SAJA** —
tombol simpan dan keputusan komite tidak dibangun sama sekali, bukan dibangun lalu
dinonaktifkan.

Parameter `ErrMsg`-nya patut diperhatikan saat ditulis ulang: pola yang sama pada procedure
lain ternyata tidak di-set pada jalur sukses, sehingga `NULL` berarti berhasil (`D-68`).

Kueri penarik source-nya sama dengan §1:

```sql
SELECT owner, name, type, line, text
  FROM all_source
 WHERE owner = 'POOLDATA' AND name = 'PEGA_PORTAL_REKANAN'
 ORDER BY type, line;
```

#### ✅ Status §6.13(b) — DITERIMA 2026-09-28

`Database/PEGA_PORTAL_REKANAN.prc` sudah dikirim Work Owner: 156 baris, 89 parameter masuk +
`ErrMsg` keluar. **Permintaan ini ditutup.**

Dengan itu seluruh artefak modul Inbox Service Center lengkap — pemindaian ulang terhadap
sembilan activity dan sepuluh section modul ini tidak menemukan satu pun activity, section,
rule SQL, atau report definition buatan sendiri yang masih hilang.

Isinya dicatat di catatan pengembangan §66, termasuk empat temuan yang mengubah pemahaman:
tabelnya melayani dua lini (`SC` dan `BROKER`), ada tiga jalur tulis, `COMMIT`-nya tidak
konsisten antar cabang, dan `ErrMsg`-nya justru berperilaku benar.

**§6.13(a) masih terbuka** — rule grid `GetDataServiceCenter` pada kelas
`ASM-FW-GCNMFW-Data-ClaimData`. Dampaknya tetap kecil: hanya memastikan klausa `ORDER BY`.

#### ✅ Status §6.13(a) — DITUTUP 2026-09-28, tidak perlu diminta

Work Owner menegaskan: jalur JSON **sudah tidak dipakai lagi — sekarang langsung ke kolom**.

Itu menjelaskan mengapa rule `GetDataServiceCenter` yang terekspor (kelas
`ASM-FW-GCNMFW-Int-T_GENERAL`) membaca `pooldata.service_log_nonmbu` lewat `JSON_VALUE`: ia
jalur LAMA, bukan kueri grid yang berlaku. Kueri yang berlaku membaca kolom
`POOLDATA.T_KLAIM_PORTAL_REKANAN` secara langsung — persis seperti rekonstruksi di
`inboxservicecenter.sql`, yang memang disusun dari `CountDataServiceCenter`,
`ExportDataServiceCenter`, dan `GetDataServiceCenter_Update` (ketiganya membaca kolom, bukan
JSON).

**Permintaan ini ditutup.** Dengan begitu seluruh §6.13 selesai, dan modul Inbox Service
Center tidak lagi menunggu artefak apa pun.

Satu hal yang TETAP dugaan dan tidak tertutup oleh penegasan ini: klausa `ORDER BY`. Dua
bukti yang ada tetap berbeda — `order by A.INPUTDATE desc` (langkah 28 activity) versus
`ORDER BY ID ASC` (`ExportDataServiceCenter`). Dipakai yang pertama; bila keliru, satu
suntingan.
