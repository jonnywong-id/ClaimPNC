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

---

## 5. Inbox Manager — `MENU_ID 58`, `UserInbox_Harness`

Ditulis 2026-09-27, setelah `Section/InboxManager_Sec-Section.xml` dan kedua belas section
turunannya dibaca seluruhnya, termasuk `pyContainerVisibleWhen` dan `pyPageListProperty`.

Modul ini **sudah berjalan** untuk sepuluh dari dua belas bagiannya. Yang diminta di bawah
membuka **dua tombol keputusan** dan **satu panel riwayat** — bukan membuka modulnya.

### 5.1 Yang diminta ke Tim Pega

| # | Artefak | Dirujuk oleh | Akibat bila tidak ada |
|---|---|---|---|
| P-1 | Rule SQL **`SetStatusAksepNoRangka`** | `Activity/UpdateStatusAksepRangka_HE`, `Activity/UpdateStatusRejectRangka_HE` — **kedua activity-nya ADA di export** | Tombol Approve/Reject pada **Approval Nomor Rangka Beda** tidak dapat dibangun. Kolom `NOTIF_RANGKA_HE.STS_AKSEP` sudah diketahui; **nilai** yang harus ditulis tidak |
| P-2 | Rule SQL **`SaveApproveAkseptasiPaymentLeader_Sql`** | `Activity/SaveApprovalAkseptasiPaymentLeader_Act-Act.xml` — **activity-nya ADA di export**, rule yang dipanggilnya tidak | Tombol Approve/Reject pada **Payment Klaim Akseptasi** tidak dapat dibangun |
| P-3 | Activity **`InsertDataAkseptasiToLeader`**, **`InsertLogKasir_act`**, **`TransferCashierDataASM_act`** | `Activity/TransferToKasir_act_Leader-Act.xml` | Melengkapi P-2. **Menyetujui pembayaran tanpa ketiganya akan menandai pembayaran disetujui tanpa pernah sampai ke kasir.** Ketiganya sudah tercatat hilang pada `D-55` |
| P-4 | Rule SQL **`HistoryManagerPaymetApproval_SQL`** | `Activity/GetDataAkseptasiKlaimKasir-Act.xml`; mengisi `HistoryManager.pxResults` pada `Section/Akseptasi_PaymentLeader1` | Panel riwayat di bawah antrean Payment Klaim Akseptasi tidak dibangun |

Keempatnya sudah tercakup permintaan export ulang berbasis **Product rule** (`D-39`, `R-16`).
Yang membedakan permintaan ini dari permintaan sebelumnya: **P-1 dan P-2 punya activity
pemanggil yang ADA di export**, sehingga bukan dugaan bahwa rule-nya pernah ada.

### 5.2 Satu pertanyaan, bukan permintaan

`Section/Sec_PaymentAkseptasiKlaimCase1-Section.xml` dan
`Section/Sec_PaymentAkseptasiKlaimCase-Section.xml` **keduanya** memberi sub-tab **Approval
Progress Klaim** syarat tampil `<pyContainerVisibleWhen>1==2</pyContainerVisibleWhen>` — syarat
yang tidak mungkin benar.

Pertanyaannya: **apakah bagian itu memang sengaja dimatikan, atau ia dimatikan sementara lalu
terlupakan?**

Kami **tidak** menampilkannya, karena Pega tidak menampilkannya. Kolom dan kuerinya tetap
disiapkan; menghidupkannya adalah menghapus dua baris. Bila jawabannya "memang dikehendaki
hidup", kami juga membutuhkan **rule yang menulis `T_APPROVAL_PROGRESS.STS_APPROVE`** — kueri
daftarnya (`ShowApproveProgressKlaim`) ada, tetapi ia hanya menerjemahkan nilai `'0'`, sehingga
nilai untuk disetujui dan ditolak tidak terbaca dari mana pun.

Ditujukan ke **Work Owner**, bukan ke Tim Pega.

### 5.3 Yang diminta ke DBA

| # | Yang diminta | Untuk |
|---|---|---|
| D-1 | **DDL** tujuh tabel persetujuan: `BENGKEL_HE`, `PANEL_HE`, `SPAREPART_HE`, `GCNM_M_SPAREPART_CATEGORY`, `GCNM_M_SPAREPART_TYPE`, `SPAREPART_HE_VIN_KEY`, `MST_PENOLAKAN_KLAIM_2` | Memastikan tipe kolom status. `NULL` dan `'0'` sengaja masih dibedakan karena tanpa DDL tidak ada yang memastikan keduanya berarti sama (`R-08`) |
| D-2 | **Hak `UPDATE`** akun aplikasi atas ketujuh tabel itu | Enam antrean master dan Penolakan Klaim sudah punya tombol keputusan; haknya **belum pernah diuji**. `-periksa` sengaja tidak mengujinya — mengujinya menuntut menulis sungguhan |
| D-3 | **Koneksi Oracle staging** | Menjalankan `PENYIMPANAN=oracle -periksa`. Keenam kueri dashboard dan kesepuluh kueri antrean sejauh ini baru terbukti **sah secara bentuk**, bukan sah terhadap basis data |
| D-4 | Konfirmasi **`T_APPROVAL_PROGRESS.ATASAN` selalu berbentuk `cabang/…`** | Penyaring cabang memotong di `/`; baris tanpa `/` menghasilkan cabang kosong dan tidak akan pernah muncul. Relevan hanya bila §5.2 dijawab "hidup" |
| D-5 | Seberapa sering **`PEGA_DASHBOARDPNC.COL_DESC`** terisi | Kueri Pega menyaring `COL_DESC IS NOT NULL`. Bila kolomnya jarang terisi, pemecahan "per sebab kerugian" akan tampak kosong tanpa sebab yang terbaca pengguna |

### 5.4 Yang TIDAK kami butuhkan — dicatat supaya tidak diminta berulang

| Hal | Kenapa |
|---|---|
| API pengganti DB Link untuk `mst_det_sales@asmd` | Sempat dicatat sebagai penghalang layar ini. **Keliru**: klasifikasi Leader/Member/Fac-In ada di `GetProgressAllYearDashboarOS`, yang membacanya dari `POOLDATA.PEGA_DASHBOARDPNC` dan `POOLDATA.T_CLAIM_PNC` — keduanya lokal. Dan kueri itu melayani layar Dashboard OS yang berdiri sendiri, bukan Inbox Manager |
| `GetYearDashboardOS`, `GetProgressAllYearDashboarOS` | Sudah ada dan sudah terbaca, tetapi tidak diikat `Section/PNCDashboardOS-Section.xml`. Keduanya milik layar Dashboard OS yang berdiri sendiri |
| Rule pemicu tombol navigasi antar bagian | Perilakunya sudah terbaca lengkap dari `FlagManager.AlasanKlaim` 1–13 |

---

## 6. Inbox Manager Admin — pindah sumber ke `POOLDATA.T_CLAIMLIST_ADMIN`

Ditulis 2026-09-27, sesudah Work Owner menetapkan sumber data layar ini berpindah dari
`DATAPEGA.PC_ASM_FW_GCNMFW_WORK` + `DATAPEGA.PC_ASSIGN_WORKLIST` ke satu tabel datar.

Modul ini **sudah berjalan**. Yang diminta di bawah bukan membuka modulnya, melainkan
membuat ketiga tabnya **berbeda satu sama lain** — tanpa itu ketiganya menampilkan baris
yang sama.

### 6.1 Permintaan kolom DICABUT SELURUHNYA — tidak ada yang perlu ditambahkan

Bagian ini semula meminta **tiga kolom** ke DBA. **Ketiganya ternyata sudah ada**, dan
permintaannya dicabut pada hari yang sama (2026-09-27) setelah Work Owner memeriksa tabelnya.

| Kolom | Sempat diminta karena | Kenyataan |
|---|---|---|
| `PXASSIGNEDORGUNIT` | penyaring ketiga tab; §B.1 dokumen kolom tidak menyebutnya | **sudah ada** |
| `PXCREATEOPNAME` | `inboxoutstanding` memakai `PXCREATEOPERATOR`, disimpulkan kolom namanya tak ada | **sudah ada** |
| `PYORIGUSERID` | tercatat di §B.4 sebagai kolom yang harus dibawa | **sudah ada** |

**Tidak ada satu kolom pun yang perlu ditambahkan untuk layar ini.** `migrations/0005` sudah
dikembalikan ke keadaan semula.

> **`STATUSCLAIM_1` memang belum ada**, dan pemakaiannyalah yang membuat layar ini gagal
> dengan ORA-00904. Ia tidak lagi dipakai: Status Klaim kini diturunkan dari `PYSTATUSWORK`.
> Barisnya tetap di `migrations/0005` tahap 1 untuk modul lain yang memintanya sejak
> 2026-09-22.

**Kenapa ketiganya sempat diminta.** Daftar kolom `T_CLAIMLIST_ADMIN` yang sudah ada **belum
pernah dibaca dari katalog** — dokumen hanya mencatat jumlahnya. Ketiadaan sebuah kolom
karena itu disimpulkan dari modul lain yang memakai kolom berbeda, dan itu bukan bukti.

Yang tersisa sebagai permintaan nyata di bagian ini: **tidak ada**. Yang dibutuhkan hanyalah
**L-3** di §7.6 — daftar kolom lengkapnya, supaya kekeliruan sejenis tidak terulang.

### 6.2 Dua temuan yang menyentuh modul lain — ditujukan ke **Work Owner**

Keduanya ditemukan saat pemindahan ini dikerjakan. **Tidak satu pun saya ubah**, karena
keduanya berada di luar lingkup yang diminta.

**T-1 · `PXOBJCLASS` wajib disaring — tabel ini memuat lebih dari satu kelas kasus.**

`inboxmanager/repo/sqlstore/inboxmanager.sql:76-79` membuang penyaring `PXOBJCLASS` dengan
alasan *"isinya diandaikan klaim PNC saja"*, dan menandainya sendiri **"ASUMSI INI PERLU
DIKONFIRMASI"**. Konfirmasinya kini ada, dan hasilnya **berlawanan**:

| Bukti | Isi |
|---|---|
| `inboxlaporanklaim.sql:197-200` | menggabung tabel ini dengan syarat `t.pxobjclass = w.pxobjclass` sementara `w` disaring `'ASM-FW-GCNMFW-Work-ReceiveDocument'` — gabungan itu tidak akan pernah menghasilkan baris bila isinya hanya Work-PNC |
| `README.md` | mencatat **142 baris RCV** di dalamnya |

Akibatnya **pencacah dashboard Inbox Manager ikut menghitung baris Receive Document**,
sehingga angka yang dibaca penyelia lebih besar daripada semestinya. Karena Receive Document
dan klaim sama-sama punya `PYID`, tidak ada yang tampak salah.

Perbaikannya satu baris `AND a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'` pada kesepuluh
pencacahnya — menunggu persetujuan lingkup.

**T-2 · Dua konvensi penamaan yang bertentangan untuk tabel yang sama.**

| Konvensi | Dipakai | Bentuk |
|---|---|---|
| **A — nama asli** | `inboxoutstanding`, `inboxlaporanklaim`, `inboxmanageradmin` | `PYID` · `PYSTATUSWORK` · `USERTEKNIS_1` · `PXOBJCLASS` |
| **B — nama baru** | `inboxmanager` | `NOKLAIM` · `STATUSWORK` · `USERTEKNIS` · `BUSINESSCODE` |

Satu tabel hanya punya satu nama untuk satu kolom, jadi keduanya tidak dapat sama-sama
benar. Konvensi A **terbukti** — ketiganya membaca kolom yang memang ada. Konvensi B adalah
**permintaan** kolom yang belum ada.

Yang perlu diputuskan: kolom Konvensi B ditambahkan dengan nama barunya, atau kuerinya
disesuaikan ke nama yang sudah ada. Membiarkan keduanya berarti satu tabel dengan dua nama
untuk `PYID` yang sama — dan modul berikutnya akan menebak.

### 6.3 Yang TIDAK kami butuhkan — dicatat supaya tidak diminta berulang

| Hal | Kenapa |
|---|---|
| `PXREFOBJECTKEY`, `PXREFOBJECTINSNAME` | Kunci join worklist → work object. Karena kedua tabel menyatu, keduanya menjadi `PZINSKEY` dan `PYID` yang sudah ada. Menambahkannya justru **mengembalikan** dua cacat join yang baru saja hilang |
| Riwayat penugasan | Layar ini hanya membutuhkan unit organisasi yang **sedang** memegang, bukan yang pernah. Rata-rata 34 penugasan per klaim tidak muat dalam satu baris, dan layar ini memang tidak memintanya |
| Kolom tampilan lain dari `PC_ASM_FW_GCNMFW_WORK` | Layar Pega menggambar **8 kolom** dari 13 isian Report Definition. Empat yang tidak digambar — `BRANCHNAME`, `DATEOFLOSS_1`, `USERTEKNIS_1`, `PYORIGUSERID` — tidak dibawa, dan tidak akan dibawa hanya karena tabelnya kebetulan memilikinya |

---

## 7. Inbox Manager Admin — isi `M_LOGIN_PNC.LINE_BUSINESS`

Ditulis 2026-09-27, sesudah sumber lini bisnis layar itu dibetulkan.

**Ini bukan permintaan artefak, dan bukan permintaan DDL.** Kolomnya **sudah ada** dan
kodenya **sudah membacanya**. Yang diminta adalah **mengisi datanya**.

### 7.1 Keadaan hari ini

| Hal | Keadaan |
|---|---|
| Kolom `POOLDATA.M_LOGIN_PNC.LINE_BUSINESS` | **ada** — terverifikasi dari `ALL_TAB_COLUMNS` |
| Terisi pada | **1 dari 1 baris** saat diperiksa 2026-09-24 (`migrations/0004_DICABUT.md`) |
| Nilai yang dikenali layar | `NONMBU` · `PA` · `TRAVEL` |
| Akibat bila kosong | petugas **tidak melihat satu tab pun** — sama seperti `pyPosition` kosong di Pega |

Selama kolomnya kosong, layar menampilkan keterangan yang menyebut keadaan itu apa adanya
("belum diisi") beserta ketiga nilai yang membuka antrean. Ia **tidak** menampilkan layar
kosong tanpa sebab, dan **tidak** menghasilkan galat.

### 7.2 Yang diminta

| # | Yang diminta | Kepada |
|---|---|---|
| L-1 | **Isi `LINE_BUSINESS`** bagi setiap petugas admin klaim, dengan `NONMBU`, `PA`, atau `TRAVEL` | Work Owner menetapkan pemetaannya; DBA atau pengelola master pengguna mengisinya |
| L-2 | Tetapkan **siapa yang merawatnya** ketika petugas berpindah lini bisnis | Work Owner |

Padanannya di sistem lama adalah `OperatorID.pyPosition` pada rekaman operator Pega — field
yang diisi administrator, bukan yang datang dari sistem kepegawaian. Isinya karena itu dapat
diambil dari konfigurasi operator Pega yang berlaku sekarang.

### 7.3 Kenapa ini tidak dapat diambil dari HCQ

Sampai 2026-09-27 aplikasi justru mencobanya: nilainya dibaca dari
`EmpResponse.Placement.PositionName`. Itu **jabatan kepegawaian** — "IT SPECIALIST" dan
sejenisnya — dan tidak pernah berisi kode lini bisnis, sehingga **tidak seorang pun** melihat
satu tab pun.

Tidak ada pemetaan dari jabatan ke lini bisnis di basis data maupun di export, dan membuatnya
berarti menebak. Karena itu nilainya disimpan sebagai **data master yang dirawat manusia**,
persis seperti di Pega.

### 7.4 Yang TIDAK diminta — dicatat supaya tidak diminta berulang

| Hal | Kenapa |
|---|---|
| Kolom baru untuk lini bisnis | `LINE_BUSINESS` sudah ada. Menambah kolom kedua berarti dua kolom untuk satu arti, dan modul berikutnya akan menebak yang mana |
| Migrasi `0004_login_line_business` | Sudah **dicabut** sebelum dijalankan karena namanya salah (`LINEBUSINESS` tanpa garis bawah). Jangan dihidupkan kembali |
| Perubahan pada HCQ | Jabatan kepegawaian memang bukan lini bisnis. Yang berhenti adalah memakainya sebagai lini bisnis, bukan HCQ-nya yang perlu berubah |

---

### Catatan 2026-09-27 — permintaan §5 DITANGGUHKAN

Modul Inbox Manager (`MENU_ID 58`) dihapus atas perintah Work Owner pada hari yang sama
(`keputusan-implementasi.md` §66). Keempat permintaan pada §5.1 dan kelima permintaan pada §5.3
karena itu **tidak perlu dikirim** sampai modul ini dibangun ulang.

Satu butir TETAP berlaku dan bukan milik modul ini: `POOLDATA.T_CLAIMLIST_ADMIN.PXASSIGNEDORGUNIT`
yang diminta modul **Inbox Manager Admin** (`MENU_ID 57`) dan menunggu `migrations/0005` tahap 1
dijalankan DBA.

### 7.5 Cacat yang sama ditemukan di DUA modul lain — ditujukan ke **Work Owner**

Diperiksa sesudah koreksi Inbox Manager Admin selesai, karena cacatnya berpola: `pyPosition`
Pega berisi **kode lini bisnis**, sementara `auth.User.Position` berisi **jabatan
kepegawaian** HCQ. Setiap modul yang membandingkan keduanya akan salah.

Keduanya **TIDAK diubah** — permintaan sesi ini menyebut "hanya ubah bagian ini saja".

| # | Modul | Perbandingannya | Akibat |
|---|---|---|---|
| C-1 | **Archive Dokumen Klaim** | `BranchScopeFor(caller.Position)` pada `criteria.go:239` mencocokkan `NONMBU`/`PA`/`TRAVEL` | Selalu jatuh ke `default:` → `BranchScope{}` → **tidak ada Group Panel yang dikecualikan**. Petugas melihat **LEBIH BANYAK** daripada yang semestinya |
| C-2 | **Daftar Tipe Dokumen Bisnis** | `usecase/manage.go:247` mencocokkan `bulkSelectPosition = "NONMBU"` | Tombol "Tambah semua bisnis NONMBU" **tidak pernah tampil** bagi siapa pun |

**C-1 lebih berat daripada cacat yang baru diperbaiki**, dan arahnya berlawanan. Di Inbox
Manager Admin akibatnya petugas melihat **terlalu sedikit** — layar kosong, dan orang
melapor. Di Archive Dokumen Klaim akibatnya petugas melihat **terlalu banyak**, dan tidak ada
yang melapor karena layarnya tampak bekerja.

Ia **bukan `R-20`**: pemisahan antarentitas tetap dijaga `RepoSelector`, dan yang gagal adalah
pembatasan lini bisnis **di dalam satu badan hukum**.

**Perbaikannya sama dengan yang sudah dikerjakan di sini**: baca dari
`M_LOGIN_PNC.LINE_BUSINESS` lewat seam, bukan dari sesi. Seam, kueri, dan ujinya sudah ada
sebagai contoh di `internal/inboxmanageradmin/`.

Yang diminta: **persetujuan lingkup** untuk memperbaiki keduanya. C-1 mengubah baris yang
terlihat petugas, sehingga ia perlu diketahui lebih dulu — bukan diperbaiki diam-diam.

### 7.6 Satu permintaan yang mencegah seluruh kelas kekeliruan ini

| # | Yang diminta | Kepada |
|---|---|---|
| L-3 | **Daftar lengkap kolom `POOLDATA.T_CLAIMLIST_ADMIN`** beserta tipe dan panjangnya | DBA |

Satu kueri sudah cukup:

```sql
SELECT column_name, data_type, data_length, data_precision, data_scale, nullable
  FROM all_tab_columns
 WHERE owner = 'POOLDATA' AND table_name = 'T_CLAIMLIST_ADMIN'
 ORDER BY column_id;
```

**Kenapa ini penting melebihi kolom mana pun yang diminta di atas.**
`docs/kolom-t-claimlist-admin.md` mencatat **jumlahnya** (40 kolom) tetapi tidak pernah
menyebut **namanya** satu per satu. Akibatnya setiap modul yang memakai tabel ini harus
menebak apakah sebuah kolom ada — dan menebak dari sumber yang tidak membuktikan apa pun,
yaitu modul lain yang kebetulan tidak memakainya.

Dua kekeliruan pada 2026-09-27 lahir persis dari situ: `PXCREATEOPNAME` diminta padahal sudah
ada, dan `PYORIGUSERID` dicatat tidak dibawa tanpa diketahui ia ada. Keduanya ditemukan
Work Owner, bukan oleh pemeriksaan kami.

Kekeliruan seperti itu **berakibat nyata bila lolos**: `ALTER TABLE ... ADD` polos gagal
dengan **ORA-01430** pada kolom pertama yang sudah ada, dan menyisakan tabel setengah jadi.

Dengan daftar itu di repo, setiap pertanyaan "apakah kolom X ada" terjawab dengan membaca —
bukan dengan menyimpulkan.
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
---

## 8. Inbox Manager dibangun ulang (2026-09-28) — status seluruh permintaan

Modul `MENU_ID 58` dibangun ulang pada 2026-09-28. Catatan penangguhan di bawah §7.4 karena
itu **tidak berlaku lagi**, dan status tiap permintaan diperbarui di sini.

### 8.1 `L-3` TERTUTUP — daftar kolom sudah dibaca

Kueri katalog pada §7.6 **sudah dijalankan** terhadap ASM (`DEV_PEGA83G`, 2026-09-28).
Hasilnya lengkap dengan keterisian tiap kolom ada di `kolom-t-claimlist-admin.md` §H.

| Yang dicatat sebelumnya | Kenyataan |
|---|---|
| 40 kolom | **58 kolom** |
| `STATUSCLAIM_1` belum ada | **ADA** — tetapi kosong di seluruh 1.014 baris |
| `DOKUMENLENGKAP_1`, `ISPENDINGCLOSE`, `SURVEYORTYPE_1`, `ADJUSTERPIC_1`, `STATUSKOMUNIKASI_1`, `PXDEADLINETIME`, `PXGOALTIME` "harus ditambahkan" | **SELURUHNYA ADA** — dan seluruhnya kosong |

**Akibat yang menggeser pekerjaan.** Yang menahan tujuh tab layar Inbox bukan lagi DDL
melainkan **proses pengisi tabel**: kolomnya sudah ada, isinya tidak pernah ditulis.
Menjalankan migrasi tambahan tidak akan mengubah apa pun.

### 8.2 Permintaan kolom untuk Inbox Manager — TIDAK ADA

Seluruh kolom yang dibutuhkan keempat kueri dashboard **sudah ada**, termasuk dua yang
berasal dari tabel worklist:

`PXOBJCLASS` · `PYSTATUSWORK` · `USERTEKNIS_1` · `PXCREATEDATETIME` · `GROUPPANEL_1` ·
`BRANCHNAME` · `BUSINESSCODE_1` · `BUSINESSGROUPID` · **`PXFLOWNAME`** · **`PXTASKLABEL`**

Tidak ada satu pun yang perlu diminta ke DBA untuk modul ini.

### 8.3 Dua artefak Pega yang MASIH hilang — ke **Tim Pega**

Keduanya dibutuhkan jalur **Setuju** pada tab Payment Klaim Akseptasi, dan tanpa keduanya
jalur itu ditahan di aplikasi.

| # | Yang diminta | Dipanggil oleh |
|---|---|---|
| P-1 | `RDB List/UpdateChasierIDTablePembayaran` | `TransferToKasir_act_Leader` |
| P-2 | `RDB List/GetDataMSTDetailSales` | `TransferCashierDataASM_act` |

**Terima kasih atas lima artefak yang sudah dikirim pada hari yang sama** —
`SetStatusAksepNoRangka_SQL`, `SaveApproveAkseptasiPaymentLeader_Sql`,
`InsertDataAkseptasiToLeader`, `InsertLogKasir_act`, dan `TransferCashierDataASM_act`. Dua yang
pertama **langsung dipakai**: antrean Approval Nomor Rangka Beda kini dapat diputuskan penuh,
dan jalur Tolak pada Payment Akseptasi pun.

**Satu hal yang TIDAK akan selesai dengan artefak tambahan.** `SaveApprovalAkseptasiPaymentLeader_Act`
juga menulis kolom pada objek kerja Pega (`.StsPenerimaKlaim`, `.FlagAtasan`, `.RemarkManager`,
`.TransferCashierStatus`) lewat `Obj-Save` + `Commit`. Tabel itu berada di skema `DATAPEGA` dan
dimiliki Pega selama masa paralel (`P-1`), sehingga aplikasi ini tidak boleh menulisinya —
berapa pun artefak yang dikirim. Ditambah integrasi keluar ke sistem Kasir yang merupakan
lingkup modul `S-4`, yang belum dibangun.

Jalur Setuju karena itu **tetap ditahan** sampai ketiganya selesai, dan alasannya ditampilkan
di layar supaya penyelia tidak melaporkannya sebagai kerusakan.

### 8.4 Dua temuan BASIS DATA — ke **DBA**, dan keduanya menyentuh modul yang SUDAH JADI

Ditemukan saat memeriksa kesembilan sumber antrean. Keduanya **bukan** cacat aplikasi, dan
tidak dapat diperbaiki dari sisi aplikasi.

#### D-1 · `POOLDATA.SPAREPART_HE` adalah VIEW berstatus **INVALID**

```
ORA-04063: view "POOLDATA.SPAREPART_HE" has errors
ORA-00904: "A"."JSONDATA": invalid identifier
```

Definisi view-nya membaca `JSON_VALUE(a.JSONDATA, '$.NO_SPART')` dan dua puluh enam ekspresi
sejenis dari `POOLDATA.M_SPAREPART_HE`. Kolom `JSONDATA` **sudah tidak ada** di tabel itu:
isinya kini 18 kolom bernama `MERK`, `TYPE`, `SUBTYPE`, `PRODDATE`, `KATEGORI`, `SUBKATEGORI`,
`BAGIAN`, `PARTNO`, `PARTNAME`, `QTY`, `NOMORPNC`, `SERIALNUMBER`, `REFF`, `TGLINPUT`,
`SOURCE`, `JENISALATBERAT`, `ID_IMAGE`, `OTOMATIS_MAPPING` — 9.869.746 baris.

**Yang perlu diperhatikan:** nama kolom baru itu **tidak sama** dengan nama parameter JSON yang
dibaca view-nya, dan **tidak satu pun tabel `POOLDATA.%SPAREPART%` punya kolom `APPROVAL`**
selain view yang rusak itu sendiri. Artinya kolom persetujuan Master Sparepart HE belum punya
tempat setelah JSON-nya dibongkar.

**Terdampak:**

| Layar | Akibat |
|---|---|
| Inbox Manager — tab Master Sparepart | pencacah dan daftarnya tidak dapat dibaca; layar menyatakannya apa adanya |
| **Modul Master Sparepart** (`MENU_ID` tersendiri, sudah selesai) | membaca DAN menulis view yang sama — seluruh layarnya gagal |

Modul Master Sparepart **tidak disentuh** sesi ini (Isolasi Protektif). Yang dibutuhkan adalah
view-nya diperbaiki; begitu itu terjadi, kedua layar hidup tanpa satu baris kode pun berubah.

#### D-2 · `T_CLAIMLIST_ADMIN.PXASSIGNEDORGUNIT` terisi **1 dari 1.014 baris**

Kolomnya ADA — koreksi 2026-09-27 benar. Yang belum berjalan adalah **pengisinya**.

Modul **Inbox Manager Admin** (`MENU_ID 57`) menyaring ketiga tabnya dengan kolom itu
(`AdminPNC` / `AdminPA` / `AdminTRAVEL`), sehingga ketiganya menampilkan paling banyak satu
baris hari ini. Kuerinya benar dan `-periksa` akan hijau; yang kurang adalah datanya.

Ia keadaan yang **tidak menghasilkan satu pun galat**, dan karena itu hanya terlihat bila
dihitung — persis keadaan `STATUSLOCK_1` dan `REQUESTSURVEY_1` yang sudah tercatat.

### 8.5 Satu keadaan data yang perlu diketahui, bukan diperbaiki

`T_CLAIMLIST_ADMIN.GROUPPANEL_1` **tidak memuat satu pun nilai `005`** (Travel):

| Nilai | Baris |
|---|---:|
| `006` | 657 |
| `003` | 305 |
| `004` | 50 |
| `002` | 2 |

Penyaring lini bisnis **TRAVEL** pada dashboard Outstanding karena itu mengembalikan nol baris
hari ini. Itu keadaan DATA, bukan cacat kueri — dicatat supaya tidak dilaporkan sebagai
kerusakan.

### 8.6 Yang TIDAK kami butuhkan — dicatat supaya tidak diminta berulang

| Hal | Alasan |
|---|---|
| `GetYearDashboardOS` | Ia mengisi daftar pilihan tahun, dan tab Outstanding tidak punya penyaring periode — ketiga kuerinya tidak menyaring tanggal sama sekali |
| `ShowApproveProgressKlaim` beserta sub-tab "Approval Progress Klaim" | Kontainernya ber-`pyContainerVisibleWhen = 1==2` di Pega — sudah mati di sana, dan tidak dibawa |
| Kolom tambahan untuk kesembilan antrean persetujuan | Seluruhnya POOLDATA dan seluruh kolomnya sudah ada |

---

## 9. Inbox RCL/PUCL — `MENU_ID 61`

Bab ini ditujukan ke **Tim Pega**, dan isinya **satu permintaan**.

Sepuluh pertanyaan modul ini diajukan pada 2026-09-30. **Sembilan sudah ditutup** pemilik
bisnis pada hari yang sama — sebagian dengan menjelaskan arti kolomnya, sebagian dengan
memutuskan mengikuti sistem lama apa adanya. Tidak ada satu pun permintaan ke DBA yang
tersisa dari modul ini.

### 9.1 Sembilan isian yang tidak punya kolom

Layar kerja RCL/PUCL memuat tujuh belas isian. **Delapan** dapat diisi; **sembilan** tidak,
dan sebabnya bukan data yang kosong melainkan **tempat penyimpanannya**.

Kesembilannya tersimpan sebagai properti clipboard pada objek kerja — dijelaskan Work Owner
pada 2026-09-24 — dan properti clipboard yang tidak dioptimasi tidak punya kolom tabel,
sehingga tidak dapat dibaca kueri biasa selama objek kerjanya masih dimiliki sistem lama:

| Isian di layar | Properti |
|---|---|
| No Kontrak | `.ClaimData.PUCLStatus.NIK` |
| Business Unit / Seksi | `.ClaimData.PUCLStatus.*` |
| Perihal | `.ClaimData.PUCLStatus.*` |
| Keterangan (tiga isian) | `.ClaimData.PUCLStatus.*` |
| Email Tertanggung | `.ClaimData.PUCLStatus.*` |
| Tanggal Kelengkapan Dokumen | `.ClaimData.PUCLStatus.*` |
| Tanggal terima Dokumen (daftar) | `.ClaimData.PUCLStatus.*` |

**Yang diminta:** kesembilan properti itu **dioptimasi menjadi kolom** (*Optimize for
reporting*), lalu **nama kolom hasilnya** dikirimkan kepada kami.

Nama kolomnya perlu disebutkan, bukan hanya optimasinya dijalankan: nama kolom hasil optimasi
tidak selalu sama dengan nama propertinya, dan menebaknya menghasilkan kueri yang gagal
seluruhnya — bukan satu isian yang kosong.

### 9.2 Kenapa ini tidak dapat digantung

Keberhasilan setiap layar diukur dengan membandingkan hasilnya terhadap sistem lama. Selama
kesembilan isian ini tidak dapat dibaca, layar RCL/PUCL **tidak akan pernah dapat dinyatakan
setara** — bukan karena ada yang salah, melainkan karena sembilan isiannya memang tidak
terbaca.

Layarnya tetap dapat dipakai hari ini; yang tertahan adalah pernyataan selesainya. Karena itu
permintaan ini berada di jalur kritis kelulusan modul, dan kami meminta **tanggal komitmen
tertulis** — bukan sekadar konfirmasi bahwa permintaannya diterima.

Keputusan Work Owner 2026-09-30 memilih jalur ini secara sadar, di antara dua kemungkinan:
mengekspos propertinya, atau mengeluarkan kesembilan isian itu dari lingkup. Yang dipilih
**mengekspos**, sehingga layarnya benar-benar setara dan bukan setara-dengan-pengecualian.

### 9.3 Sementara menunggu

Kesembilan isian tetap **digambar di tempatnya** dengan penanda tersendiri — bukan
dihilangkan, dan bukan pula digambar sebagai sel kosong.

Perbedaannya bukan kosmetik. Sel kosong berarti **petugas belum mengisinya**; penanda ini
berarti **nilainya ada tetapi tidak terbaca dari tabel**. Keduanya menuntut tindakan dari
orang yang berbeda, dan menyamakannya membuat yang satu tersamar sebagai yang lain.

### 9.4 Yang TIDAK perlu dikirim — dicatat supaya tidak diminta berulang

Seluruh baris di bawah ini **sudah ditutup** dan tidak memerlukan artefak, kueri, maupun
jawaban dari siapa pun.

| Hal | Sebabnya ditutup |
|---|---|
| Arti `MSIG_1`, `PUCLAPPROVE_1`, `LAMAKLAIM_1`, `RCL_PUCL_1` | dijelaskan Work Owner 2026-09-30 |
| Arti `STATUSCASE_1` | Work Owner memutuskan mengikuti sistem lama apa adanya |
| Jumlah klaim bersurat yang penandanya kosong | keputusan 2026-09-30: ikuti sistem lama apa adanya; penyaringnya tidak diubah |
| Apakah `MSIG_1` pernah terisi | nilai pembandingnya disalin dari Report Definition, bukan ditebak — tab ini kosong di sini persis bila ia kosong juga di sistem lama |
| Apakah `LAMAKLAIM_1` sama isinya dengan `TANGGALKIRIMPUCL_1` | bila sama pun, kedua kolom tetap digambar — duplikasinya ada di sistem lama |
| Rule Section `SendtoRCLPUCL` | sudah diterima 2026-09-24 |
| Report Definition ketiga antrean | ketiganya ada di export dan sudah terbaca utuh |

### 9.5 Dua baris §9.4 yang kini punya jawaban terukur — tidak mengubah permintaannya

Keduanya ditutup pada 2026-09-30 dengan alasan yang benar: perilakunya mengikuti sistem lama,
sehingga jawabannya tidak dapat mengubah apa pun. Angkanya kemudian **terbaca sendiri** saat
modul ini dijalankan terhadap Oracle pada hari yang sama, dan dicatat di sini supaya tidak
ditanyakan lagi:

| Hal | Angkanya |
|---|---|
| Apakah `MSIG_1` pernah terisi | **Ya, sekali.** `GROUP BY MSIG_1` pada portal ASM: `MSIG` 1 baris, kosong 7.721 baris. Dugaan "tidak pernah terisi" terbantah |
| Apakah `LAMAKLAIM_1` sama dengan `TANGGALKIRIMPUCL_1` | **Nyaris.** Keduanya ditulis pada langkah yang sama dan terpaut milidetik; 61 baris sama persis, 25 berbeda. Setelah digambar sampai satuan detik, keduanya kerap identik |

Keduanya **tidak mengubah satu baris kode pun** — penyaring dan kolomnya memang sudah
mengikuti sistem lama. Yang berubah hanya keterangan di layar, yang sebelumnya menyatakan tab
MSIG "kemungkinan besar kosong" dan kini menyatakan isinya sangat sedikit.

---

## 10. Inbox RCL/PUCL — alamat portal Pega untuk tautan "buka di Pega"

Bab ini **bukan** ditujukan ke Tim Pega. Pemiliknya **Work Owner + Tim Infra**, dan isinya
satu keputusan konfigurasi — bukan artefak.

### 10.1 Apa yang dibutuhkan, dan untuk apa

Layar Inbox RCL/PUCL dan layar kerjanya **membaca saja** sampai sistem lama dimatikan
(keputusan 2026-09-30). Kelima tombol tindakannya digambar tetapi mati, dan petugas
mengerjakan tindakannya di sistem lama memakai kunci klaim yang sudah ditampilkan layar kerja.

Yang diusulkan: kelima tombol itu membuka klaim yang sama **langsung di sistem lama**,
sehingga langkah mencari klaimnya hilang.

### 10.2 Kenapa ia belum dapat dikerjakan

**Tidak ada satu pun alamat portal sistem lama di konfigurasi maupun di basis data.**
Diperiksa langsung pada 2026-09-30:

- Berkas konfigurasi aplikasi tidak memuatnya sama sekali.
- `POOLDATA.GCNM_CONNECT_REST` — tempat aplikasi ini membaca alamat layanan per portal, pola
  yang sama dengan alamat layanan HCQ — memuat **hanya alamat layanan REST**
  (`…/prweb/PRRestService/…`, `…/resources/restws/…`). Tidak satu pun barisnya alamat portal
  yang dapat dibuka orang.

Sebagian baris itu memang memuat nama host sistem lama, tetapi **menurunkan alamat portal dari
alamat layanan REST berarti menebak** — dan host-nya sendiri tidak sepakat: baris untuk
layanan yang sama kadang menunjuk host produksi, kadang host pengembangan.

### 10.3 Yang diminta diputuskan

1. **Apakah tautan "buka di Pega" memang dikehendaki** pada layar yang baca-saja.
2. Bila ya: **alamat portal sistem lama per entitas**, beserta **bentuk tautan yang membuka
   satu klaim** dari kunci `pzInsKey` yang sudah ditampilkan layar.
3. **Di mana alamat itu disimpan.** Usul kami: satu baris baru di `POOLDATA.GCNM_CONNECT_REST`
   per portal — pola yang sudah dipakai alamat layanan HCQ, sehingga perpindahan alamat
   menjadi perubahan data oleh DBA, bukan rilis ulang aplikasi.

**Variabel lingkungan baru sengaja TIDAK dibuat** sambil menunggu. Alamat per entitas adalah
keputusan konfigurasi milik pemilik lingkungan, bukan milik satu modul — dan membuatnya lebih
dulu akan menetapkan tempat penyimpanannya tanpa ada yang memutuskannya.

## 9. Inbox Banding Harga Salvage — `MENU_ID 72`, `InboxRequestSalvage` (2026-09-29)

### 9.1 Tujuh artefak sudah DITERIMA pada hari yang sama

Diminta bertahap dalam tiga putaran, karena setiap berkas yang datang memunculkan rujukan ke
berkas berikutnya. Dicatat supaya urutan itu terbaca bila kelak ada layar sejenis:

| Artefak | Ditemukan dari | Isi yang dibawa |
|---|---|---|
| `Section/InboxReqSalvageASM-Section.xml` | rujukan di harness | DUA grid, kolomnya, kotak Cari |
| `Activity/GCNMCountRequestSalvage_act-Act.xml` | rujukan di harness | tabel ringkas; label dari `Local.LOOP` |
| `Activity/SetReqSalvage_Act-Act.xml` | `pyActivity` di section | pemasok kedua grid, paginasi, penyaring bernama orang |
| `RDB List/CountRequestSalvage_Sql-SQL.xml` | rujukan di activity pencacah | pencacah Request dan total paginasi |
| `RDB List/CountHistoryReqSalvage_Sql-SQL.xml` | idem | pencacah History |
| `RDB List/DataReqSalvage_SQL-SQL.xml` | rujukan di `SetReqSalvage_Act` | kueri grid Request |
| `RDB List/HistoryReqSalvage_SQL-SQL.xml` | idem | kueri grid History |

**Pelajaran untuk permintaan berikutnya:** harness saja tidak cukup untuk menyatakan apa yang
kurang. Ia hanya menyebut section teratas; rantai sesungguhnya baru terbaca setelah section itu
dibuka, lalu activity-nya, lalu rule SQL-nya. Meminta "harness beserta seluruh rule yang
dirujuknya secara berantai" akan memangkas tiga putaran menjadi satu.

### 9.2 TIGA artefak yang MASIH kurang

| # | Artefak | Yang tertahan karenanya |
|---|---|---|
| 1 | `Section/ButtonApproveRejectedRequest-Section.xml` | Kolom **Action** — tombol Approve dan Reject banding. Kolom mana yang ditulisnya pada `T_CLAIM_CHEKER_SALVAGE` dan nilai apa yang disetelnya tidak diketahui, sehingga tombolnya tidak digambar sama sekali |
| 2 | `Flow Action/DetailHistoryRequestSalvage-FA.xml` | Panel rincian satu baris pada grid **History Cheker** |
| 3 | Rule SQL yang dipanggil nomor 1 | Belum diketahui namanya — ia baru terbaca setelah section-nya ada |

Padanannya yang ADA di export dan dapat dipakai sebagai pembanding bentuk:
`Section/ButtonApproveRejectedChecker-Section.xml`, yang memuat `pyButtonLabel Approve`,
`pyButtonLabel REJECTED`, dan `pyButtonLabel Lihat File`. Ia **bukan** rule yang dicari — ia
milik grid Checker pada layar Inbox Salvage — tetapi bentuknya menunjukkan apa yang diharapkan.

### 9.3 Satu permintaan ke DBA, bukan ke Tim Pega

**DDL `POOLDATA.T_CLAIM_CHEKER_SALVAGE`**, beserta beberapa baris contoh.

Tabel ini **nol kemunculan** di seluruh export selain di dalam satu potongan SQL yang dirangkai
`Activity/SetReqSalvage_Act` — ia bahkan tidak muncul sebagai nama tabel pada rule SQL mana pun
sebelum keempat rule di 9.1 diterima. Akibatnya seluruh nama kolom modul ini disimpulkan dari
teks kueri, dan **satu di antaranya disimpulkan dari nama properti grid**:

| Kolom | Dari mana namanya diketahui |
|---|---|
| `NOKLAIM`, `IDSALVAGE`, `IDDETAILSALVAGE`, `NAMAKOMITE` | disebut kueri |
| `NAMABARANG`, `HARGABARANG`, `HARGAREQUEST`, `ALASANREQUEST` | disebut kueri |
| `TGLREQUEST`, `TGLAPPROVE`, `STATUSAPPROVE` | disebut kueri |
| **`NOTEKOMITE`** | **disimpulkan dari `.NoteKomite`, properti kolom "Note Checker"** |

Yang dibutuhkan dari DDL ada tiga, dan ketiganya berakibat nyata:

1. **Apakah `NOTEKOMITE` benar-benar ada.** Bila tidak, kuerinya gagal seluruhnya — bukan
   mengosongkan satu kolom. `check_table` sudah menyebut kedua belas kolomnya satu per satu
   supaya kekeliruan itu terbaca saat aplikasi start, tetapi DDL menutupnya lebih awal.
2. **Tipe `HARGABARANG` dan `HARGAREQUEST`.** Keduanya nilai uang; bila ternyata `VARCHAR2`,
   perbandingan dan pengurutan apa pun terhadapnya menjadi leksikografis (`I-12`).
3. **Apakah `TGL_REQUEST` (dengan garis bawah) ada.** `CountRequestSalvage_Sql` menutup
   `SELECT COUNT(1)` dengan `ORDER BY TGL_REQUEST DESC`, dan ejaan itu **nol kemunculan** di
   mana pun selain baris tersebut. Klausanya dibuang di sistem baru (selisih terencana nomor
   3); DDL memastikan pembuangan itu memang benar dan bukan menutupi kolom yang nyata.

### 9.4 Yang TIDAK kami butuhkan — dicatat supaya tidak diminta berulang

| Hal | Alasan |
|---|---|
| `Section/TambahData_Salvage` | Sudah dipakai modul Inbox Salvage; tombol "Tambah" di layar ini membuka form yang sama, dan tidak digandakan |
| `pxChart` pada tabel ringkas | Harness menggambar isi yang sama dua kali — tabel dan diagram. Dua baris angka tidak memerlukan grafik |
| `When/IsGCNMUser` | Sudah ada di export, dan isinya `compareTwoValues(1, "=", 2)` — sakelar "jangan tampilkan ini", bukan pemeriksaan peran |

### 9.5 Koreksi atas 9.2 — kedua artefak itu SUDAH diterima, dan keduanya cangkang pula (2026-09-29)

`Section/ButtonApproveRejectedRequest-Section.xml` dan
`Flow Action/DetailHistoryRequestSalvage-FA.xml` diterima pada hari yang sama. Bagian 9.2 di
atas **tidak disunting** — ia merekam keadaan saat permintaan itu diajukan; yang berlaku
sekarang adalah bagian ini.

**Keduanya ternyata cangkang, persis seperti harness-nya.** Masing-masing menunjuk rule
berikutnya yang belum ada, sehingga putaran permintaan bertambah menjadi **empat**.

#### Yang kini DIKETAHUI dari `ButtonApproveRejectedRequest`

Kedua tombol memanggil satu activity, `ApprovalCheckerSalvage`, dengan lima isian yang sama:

| Parameter | Nilai | Kolom sebenarnya |
|---|---|---|
| `statusapprove` | `1` pada dua titik panggil, `0` pada dua titik lain | `STATUSAPPROVE` |
| `noteapprove` | `.NoteKomite` | `NOTEKOMITE` — catatan yang diketik komite |
| `iddetailsalvage` | `.ClientName` | `IDDETAILSALVAGE` |
| `idsalvage` | `.ClaimNo` | `IDSALVAGE` |
| `hargarequest` | `.Email` | `HARGAREQUEST` |

**Ketiga baris terakhir meneguhkan peta kolom modul ini dari sumber yang berbeda** — berguna,
dan dicatat. Ditambah satu hal yang sebelumnya hanya tertulis di `.NoteKomite`: kolom
`NOTEKOMITE` memang **ada dan ditulis**, sehingga dugaan pada 9.3 butir 1 menguat.

Ditemukan pula tombol ketiga, **"Lihat File"**, yang membuka local action
`DokumenBandingSalvage`.

#### Yang MASIH tidak diketahui, dan itu yang menahan

1. **Kolom mana yang ditulis `ApprovalCheckerSalvage`**, dan apakah ia juga menyentuh
   `TGLAPPROVE`, `PNC_SALVAGE.STSTRANSFER`, atau `HISTORY_KOMUNIKASI_SALVAGE`.
2. **Arti `statusapprove = 0`.** Pada `T_CLAIM_KOMITE_LIST` kode yang sama berarti **MENUNGGU**
   (`RDB List/CountAIDiterima_SQL`: `0` = MENUNGGU, `1` = DITERIMA, `2` = DITOLAK), sementara
   di layar ini penolakan mestinya memindahkan barisnya ke History — dan grid History menuntut
   `STATUSAPPROVE IS NOT NULL`. Kedua bacaan itu tidak dapat dipilih tanpa membaca activity-nya,
   dan memilih yang salah berarti banding yang ditolak **tidak pernah hilang dari antrean** atau
   sebaliknya **hilang tanpa pernah diputus**.

#### Empat artefak yang diminta sekarang

| # | Artefak | Yang tertahan karenanya | Pemilik |
|---|---|---|---|
| 1 | `Activity/ApprovalCheckerSalvage-Act.xml` | **Tombol Approve dan Reject.** Ini yang paling menahan — tanpa isinya, keputusan atas nilai uang tidak dapat ditulis | Tim Pega |
| 2 | `Section/DetailHistReqSalvage-Section.xml` | Isi panel rincian baris History | Tim Pega |
| 3 | `Activity/ShowDtlHistoryReqSalvage_Act-Act.xml` | Pemasok panel itu — flow action-nya mengirim nomor klaim (`ClaimID`) | Tim Pega |
| 4 | `DokumenBandingSalvage` (local action / section) | Tombol "Lihat File" pada kolom Action | Tim Pega |

Ditambah yang belum berubah: **DDL `POOLDATA.T_CLAIM_CHEKER_SALVAGE`** ke DBA (lihat 9.3).

#### Pelajaran yang mengubah cara meminta

Empat putaran untuk satu layar, dan tiap putaran hanya membuka satu lapis:

```
Harness -> Section -> Activity -> rule SQL
                   -> Section tombol -> Activity penulis
                   -> Flow Action     -> Section + Activity pemasoknya
```

Permintaan yang tepat bukan menyebut berkasnya satu per satu, melainkan:
**"harness ini beserta SELURUH rule yang dirujuknya secara berantai, sampai tidak ada lagi
rujukan yang menggantung"** — yakni export berbasis Product rule dengan
*include dependent rules*, persis yang `D-39` tetapkan.

### 9.6 Putaran kelima (2026-09-29, malam) — tiga artefak diterima, enam rule terdalam masih kurang

`ApprovalCheckerSalvage`, `DetailHistReqSalvage`, dan `ShowDtlHistoryReqSalvage_Act` diterima.
Bagian 9.5 tidak disunting; yang berlaku adalah bagian ini.

#### Pertanyaan terbesar pada 9.5 TERJAWAB

`ApprovalCheckerSalvage` langkah 11 menyusun harganya begini:

```
SalvagePrice = @if(statusapprove == "1", hargarequest, hargasalvage)
```

Artinya **`1` = Approve** (harga tandingan balai lelang diterima) dan **`0` = Reject** (harga
semula dipertahankan). Dugaan `0` = MENUNGGU — yang berlaku pada `T_CLAIM_KOMITE_LIST` — GUGUR.

Ikut terbaca: `SalvageStatus := "2"`, `ApprovalStatus := statusapprove`,
`ApprovalUser := <komite>`, `ApprovalNote := noteapprove`, dan satu parameter keenam yang
tidak dikirim tombolnya — `hargasalvage`.

Panel rincian History pun kini terpetakan lengkap — tujuh kolom, dikunci nomor klaim:

| Judul | Properti Pega |
|---|---|
| Tgl Approve | `.DateOfLoss` |
| Detail Object | `.Notes` |
| Nama Barang | `.CityID` |
| Harga Barang | `.ContractNo` |
| Harga Request | `.BranchID` |
| Jawaban Checker | `.AgentID` |
| Nama Checker | `.ClaimID` |

#### Enam rule yang MASIH kurang — seluruhnya rule terdalam

| # | Artefak | Yang tertahan |
|---|---|---|
| 1 | `RDB List/UpdateDataReqSalvage-SQL.xml` | bagian tulis Approve/Reject |
| 2 | `RDB List/UpdateDokReqSalvage-SQL.xml` | jalur Reject (`TempInsert.City == "0"`) |
| 3 | `RDB List/UpdateHargaSalvage-SQL.xml` | harga pada detail salvage |
| 4 | `Connect REST/SendData_SalvageSimasBid-REST.xml` | **mengirim keputusan KEMBALI ke balai lelang** |
| 5 | `RDB List/DetailHistReqSalvage_SQL-SQL.xml` | isi panel rincian History |
| 6 | `DokumenBandingSalvage` | tombol "Lihat File" |

#### Satu pertanyaan yang BELUM terjawab dan berakibat langsung

**Siapa yang menyetel `TGLAPPROVE`.** Satu-satunya `UPDATE` yang terbaca langsung — dirangkai
sebagai teks pada langkah 5 — hanya menyetel `STATUSAPPROVE`, dan justru menyaring
`TGLAPPROVE IS NULL`:

```
UPDATE POOLDATA.T_CLAIM_CHEKER_SALVAGE
   SET STATUSAPPROVE = '<status>'
 WHERE NAMAKOMITE = 'DANIELLISWANDI'        <- nama orang, tertanam di dalam teks SQL
   AND IDDETAILSALVAGE = '<id>'
   AND TGLAPPROVE IS NULL
```

Padahal grid Request menyaring `TGLAPPROVE IS NULL` dan grid History menuntutnya TERISI. Bila
tidak ada rule lain yang mengisinya, banding yang sudah diputus **tidak berpindah ke History
dan tidak hilang dari Request**. Dugaan terkuat: `UpdateDataReqSalvage` yang mengisinya — dan
itu salah satu dari enam yang diminta.

Dicatat pula: `NAMAKOMITE='DANIELLISWANDI'` tertanam di dalam teks SQL yang dirangkai — nama
orang kelima yang menentukan perilaku di layar ini, dan satu lagi titik `{Asis}` yang tidak
dibawa.

#### Permintaan yang seharusnya diajukan sejak awal

Ini putaran **kelima**. Sepuluh artefak sudah diterima, enam masih kurang, dan setiap putaran
hanya membuka satu lapis. Permintaan yang tepat bukan menyebut berkas satu per satu melainkan:

> **Export berbasis Product rule untuk harness `InboxRequestSalvage`, dengan opsi
> *include dependent rules* dinyalakan** — persis yang `D-39` tetapkan untuk seluruh gap
> export.

### 9.7 Putaran ketujuh — yang tersisa setelah jalur tulis selesai

Tanggal: 2026-09-30. Seluruh artefak **rule Pega** yang menahan tombol Approve dan Reject sudah
diterima, dan tombolnya dibangun. Bagian ini mencatat apa yang **masih** kurang, kepada siapa,
dan apa yang tertahan karenanya — supaya permintaan berikutnya tidak salah sasaran.

#### Putaran keenam, yang tidak sempat dicatat tersendiri

Nomor bagian melompat dari putaran kelima ke ketujuh, dan itu bukan kekeliruan penomoran:
putaran **keenam** diterima dan langsung dikerjakan pada hari yang sama, sehingga catatannya
masuk ke `catatan-pengembangan.md` §84.3 tanpa sempat menjadi bagian tersendiri di sini.
Dicatat di sini supaya daftar penerimaannya tetap utuh — kelima rule yang tiba pada putaran itu:

| Rule | Apa yang akhirnya terbaca |
|---|---|
| `UpdateDataReqSalvage` | nama kolom catatan komite yang sebenarnya — **`NOTEAPPROVE`**, bukan `NOTEKOMITE`; dan bahwa `TGLAPPROVE` diisi `sysdate` di pernyataan yang sama |
| `UpdateDokReqSalvage` | penandaan dokumen saat ditolak, beserta penyaringnya yang tampak keliru (§107.2) |
| `UpdateHargaSalvage` | penerapan harga ke `DETAIL_PNC_SALVAGE.HARGAITEM` |
| `DetailHistReqSalvage_SQL` | ketujuh kolom panel rincian beserta terjemahan kode putusannya |
| `SendData_SalvageSimasBid` | bahwa layanan pengirimannya menunjuk host **dev** tanpa autentikasi (§107.1) |

Yang pertama layak digarisbawahi: nama kolom `NOTEKOMITE` sempat **disimpulkan** dari nama
properti Pega `.NoteKomite`, dan kueri dengan nama itu akan gagal seluruhnya dengan
`ORA-00904`. Ia bukan cacat tampilan melainkan layar yang tidak dapat dibuka sama sekali —
bukti bahwa nama kolom tidak boleh disimpulkan dari nama properti.

#### Yang masih kurang, dan kepada siapa

| # | Yang diminta | Kepada | Yang tertahan |
|---|---|---|---|
| 1 | `Activity/LihatDokRequestSalvage` **dan** `Section/DokBandingHargaSalvage` | **Tim Pega** | tombol "Lihat File" pada kolom Action |
| 2 | **DDL `POOLDATA.T_CLAIM_CHEKER_SALVAGE`** — tipe, panjang, index, constraint | **DBA** | lihat di bawah |
| 3 | **Alamat produksi** layanan balai lelang beserta cara autentikasinya | **Work Owner + tim integrasi** | pengiriman keputusan ke balai lelang |
| 4 | Rule yang MENULIS `HISTORY_KOMUNIKASI_SALVAGE.MESSAGE_PNC` | **Tim Pega** | balasan ke balai lelang |

#### Kenapa DDL tabel checker kini lebih mendesak daripada sebelumnya

Selama modul ini hanya membaca, tipe kolom hanya memengaruhi tampilan. Sejak ia **menulis**,
tiga kolom menjadi penentu benar-tidaknya sebuah keputusan tersimpan:

| Kolom | Yang belum diketahui | Akibat bila tebakannya salah |
|---|---|---|
| `STATUSAPPROVE` | teks atau angka | Ditulis `'1'` ke kolom `NUMBER` dapat terbaca kembali sebagai `"1.0"`. Perapian sudah dipasang di `normalizeDecisionCode`, tetapi ia menambal **pembacaan**, bukan penulisan |
| `TGLAPPROVE` | `DATE` atau `TIMESTAMP` | `CURRENT_TIMESTAMP` ke kolom `DATE` memangkas jamnya. Pengurutan riwayat per hari menjadi tidak menentu |
| `NOTEAPPROVE` | panjang maksimum | Catatan yang melebihi batas ditolak Oracle saat disimpan — pengguna kehilangan putusan yang baru ia tulis, tanpa peringatan sebelumnya |

Yang terakhir paling layak diminta lebih dulu: ia satu-satunya yang akibatnya **terlihat
pengguna sebagai kegagalan**, dan satu-satunya yang dapat dicegah dengan batas panjang di
layar.

Kueri untuk DBA:

```sql
SELECT column_name, data_type, data_length, data_precision, data_scale, nullable
  FROM all_tab_columns
 WHERE owner = 'POOLDATA' AND table_name = 'T_CLAIM_CHEKER_SALVAGE'
 ORDER BY column_id;
```

#### Satu permintaan yang TIDAK diajukan, dan alasannya

Sempat terpikir meminta isi `POOLDATA.SALAVAGEDOCUMENT` untuk membuktikan bahwa penyaring
`UpdateDokReqSalvage` memang tidak pernah mencocokkan apa pun. Itu **tidak diminta**: Work
Owner sudah memutuskan menirunya apa adanya (§107.2), sehingga jawabannya — cocok atau tidak —
tidak mengubah satu baris kode pun. Permintaan yang jawabannya tidak mengubah apa pun hanya
membebani pihak lain.

Ia layak diminta kembali **bila** perbaikan penyaringnya kelak dipertimbangkan.

#### Catatan atas bentuk permintaan

Putaran keenam menutup seluruh rule yang menahan, dan itu karena permintaannya akhirnya
diajukan sebagai **satu daftar berdasarkan rujukan di dalam activity**, bukan berkas per
berkas. Empat butir di atas berasal dari **tiga pihak berbeda**, sehingga tidak dapat digabung
menjadi satu permintaan — tetapi masing-masing sudah disertai alasan dan akibatnya, supaya
penerimanya dapat menilai urgensinya sendiri.

Yang tetap berlaku sebagai bentuk permintaan yang benar kepada Tim Pega:

> **Export berbasis Product rule untuk harness `InboxRequestSalvage`, dengan opsi
> *include dependent rules* dinyalakan** (`D-39`).

#### Koreksi 2026-09-30 — `DokumenBandingSalvage` sudah diterima, dan ia cangkang

Butir 1 pada tabel di atas **berubah sasarannya**, dan itu perlu dicatat karena inilah kekeliruan
yang sudah tiga kali memboroskan satu putaran di layar ini.

`Flow Action/DokumenBandingSalvage-FA.xml` diterima. Membacanya membuktikan ia **tidak memuat
apa pun yang dapat dibangun**:

| Bagian | Isi | Status |
|---|---|---|
| `<pyPreProcessingActivity>` | `LihatDokRequestSalvage` — yang memuat daftar dokumennya | **tidak ada di export** |
| `pxRuleReferences` ber-`Rule-HTML-Section` | `DokBandingHargaSalvage` — yang menggambarnya | **tidak ada di export** |

Parameter pra-aktivitasnya terbaca, dan ia memakai pemetaan alias yang sama dengan tombol
Approve/Reject: `IDsalvage ← .ClaimNo`, `iddetailsalvage ← .ClientName`.

**Pola yang sudah berulang, dan layak dijadikan aturan permintaan.** Tiga artefak layar ini
diterima "lengkap" lalu ternyata cangkang: harness-nya, section tombolnya, dan kini flow
action-nya. Ketiganya terbaca utuh tanpa satu pun galat — yang hilang justru bagian yang bekerja.

Sebelum sebuah berkas dinyatakan menutup permintaan, dua elemen ini wajib dibaca lebih dulu:

```
<pyPreProcessingActivity>                      nama activity pemasoknya
pxRuleReferences  ber-Rule-HTML-Section        nama section yang menggambarnya
```

Itu pula alasan `D-39` menetapkan permintaan yang benar adalah **export berbasis Product rule
dengan *include dependent rules***, bukan daftar berkas — daftar berkas hanya dapat menyebut
lapis yang sudah diketahui.

### 9.8 Putaran kesembilan (2026-09-30) — tidak ada lagi artefak Pega yang menahan

`LihatDokRequestSalvage` dan `DokBandingHargaSalvage` diterima, dan keduanya menutup lapis
terakhir. Tombol "Lihat File" dibangun.

#### Butir 1 pada tabel §9.7 DITUTUP

Satu rule tetap tidak ada — `GetAttachmentReqSalvage` — dan ia **tidak diminta**, karena
jawabannya sudah pasti tanpa berkasnya:

| Yang diketahui | Dari mana |
|---|---|
| kelasnya `ASM-FW-GCNMFW-Int-DATA_ATTACHFILE` | `pxRuleReferences` pada activity |
| kuncinya `pyID`, diisi `IDDOC` | langkah `TempInputParamAttach.pyID := .ClaimID` |
| tiga kolom yang dibaca hasilnya | langkah Property-Set sesudahnya |
| kueri yang sama sudah berjalan | `internal/inboxpladla/repo/sqlstore/detail.sql` |

Meminta berkas yang isinya sudah tertentukan hanya membebani pihak lain tanpa mengubah satu
baris kode.

#### Yang MASIH kurang, dan kepada siapa

| # | Yang diminta | Kepada | Yang tertahan |
|---|---|---|---|
| 1 | **DDL `POOLDATA.T_CLAIM_CHEKER_SALVAGE`** — terutama panjang `NOTEAPPROVE` | **DBA** | catatan komite yang melebihi batas ditolak saat disimpan |
| 2 | **Alamat produksi** layanan balai lelang + autentikasinya | **Work Owner + tim integrasi** | pengiriman keputusan ke balai lelang |
| 3 | Rule yang MENULIS `HISTORY_KOMUNIKASI_SALVAGE.MESSAGE_PNC` | **Tim Pega** | balasan ke balai lelang |

#### Satu pertanyaan untuk DBA, bukan untuk Tim Pega

Kolom `POOLDATA.SALAVAGEDOCUMENT.NOKLAIM` **berisi id DETAIL salvage**, bukan nomor klaim —
disimpulkan dari dua rule yang membandingkannya begitu, salah satunya kueri baca yang hasilnya
terlihat pengguna. Konfirmasinya satu kueri:

```sql
SELECT DISTINCT NOKLAIM
  FROM POOLDATA.SALAVAGEDOCUMENT
 WHERE TIPEDOCSALVAGE = 'Request Banding Harga Salvage'
   AND ROWNUM <= 20;
```

Bila nilainya berbentuk `<nomor klaim>/<n>`, kesimpulan di atas benar dan nama kolomnya yang
menyesatkan. Bila berbentuk nomor klaim polos, maka kedua rule itu memang tidak pernah cocok —
dan dialog "Lihat File" di sistem lama selalu kosong tanpa ada yang melaporkannya.

Jawabannya **tidak mengubah kode**: kedua kueri meniru perbandingan yang sama apa adanya
(`P-5`). Ia hanya menentukan mana dari kedua kalimat di `PlannedDifferences()` yang berlaku.
