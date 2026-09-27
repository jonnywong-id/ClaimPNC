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

## 4. Archive Dokumen Klaim — `MENU_ID 77`

Modulnya **sudah dibangun dan berjalan** dengan rekonstruksi di tempat yang artefaknya
hilang. Yang diminta di bawah bukan penghalang pekerjaan, melainkan hal-hal yang membuat
rekonstruksi itu dapat diganti dengan yang sebenarnya — dan satu hal yang memblokir satu
fungsi.

### 4.1 Koreksi dari pihak kami — dicatat lebih dulu

`Section/KodeArchiveDoc-Section.xml` **bukan** berisi rule bernama `KodeArchiveDoc`. Isinya
rule bernama **`SecCariKodeArchiveDoc`** (`pzDocumentKey: RULE-HTML-SECTION DATA-PORTAL
SECCARIKODEARCHIVEDOC`).

Nama berkas dan nama rule berbeda. Ini kasus kedua setelah dua berkas yang sudah tercatat
pada `D-39`, dan ia menguatkan ketetapan yang sama: **inventaris rule dibangun dari
`pyRuleName`, bukan dari nama berkas.**

### 4.2 Yang memblokir — ke DBA

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

### 4.3 DDL `POOLDATA.T_CLAIM_ARCHIVE_FILE` — ke DBA

Tabelnya **tidak punya DDL di export** (`R-08`), dan tiga hal bergantung padanya:

| Yang tidak diketahui | Akibatnya hari ini |
|---|---|
| Panjang kolom teks | batas panjang isian kami pasang longgar sebagai pengaman, bukan sebagai cerminan skema |
| Nilai bawaan `CABANGSTATUS` | kami menulisnya **`'0'` eksplisit**; lihat §4.4 |
| Ada tidaknya constraint unik | tidak diketahui apakah basis data menahan ID ganda; lihat §4.5 |

Yang diminta: `DBMS_METADATA.GET_DDL('TABLE','T_CLAIM_ARCHIVE_FILE','POOLDATA')`.

### 4.4 Prosedur di export lebih tua daripada pemanggilnya — ke Tim Pega

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

### 4.5 Penomoran ID — ke Work Owner, diajukan kembali

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

### 4.6 Saringan lini bisnis yang tampak terbalik — ke Work Owner dan pemilik bisnis

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

### 4.7 Pemilih Kode Filling — ke Tim Pega

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

### 4.8 Yang TIDAK perlu dikirim

- Harness `PNCArchiveDokumen` dan section `SecArchiveDokumen`: sudah dibaca, dan isinya
  hampir seluruhnya boilerplate template.
- Kedua RDB List pencarian klaim: sudah dibaca lengkap.
- `Connect REST/InjectDataArchiveDokumentKlaim`: sudah dibaca; yang kurang hanya baris
  katalognya (§4.2).

### 4.9 Tiga tombol yang wiring-nya tidak dapat ditelusuri — ke Tim Pega dan pemilik bisnis

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

### 4.10 Dua tombol pemilih Kode Filling — ke pemilik bisnis

`SecCariKodeArchiveDoc` punya **Input Kode** dan **Generated Kode** di samping Cari Kode
dan Pilih. Keduanya menyiratkan kode arsip memang DIBUAT dari layar itu — dan itulah dasar
rekonstruksi daftar kode kami (§4.7).

**Pertanyaannya:** apakah ada aturan pembentukan kodenya — misalnya awalan tahun, nomor
urut per boks, atau kode cabang? Bila ada, kami dapat menggantikan daftar rekonstruksi
dengan pembuat kode yang benar, dan tombol "Generated Kode" menjadi dapat dibangun.

Bila tidak ada aturan dan kodenya memang diketik bebas, cukup dinyatakan begitu — isian
Kode Filling kami sudah menerima ketikan langsung, sehingga tidak ada yang perlu berubah.

### 4.11 Satu perilaku yang perlu dikonfirmasi ke pemilik bisnis

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

### 6.1 Yang diminta ke DBA — tiga kolom

Ketiganya sudah masuk [`migrations/0005`](../backend/migrations/0005_claimlist_admin_kolom.up.sql)
**tahap 1** dan tinggal dijalankan. Menempuh `D-63`: permintaan tertulis, persetujuan Work
Owner, pelaksanaan DBA, lalu diuji dengan menjalankan Pega dan Go bersamaan.

| # | Kolom | Asalnya | Akibat bila tidak ada |
|---|---|---|---|
| K-1 | **`PXASSIGNEDORGUNIT`** | `PC_ASSIGN_WORKLIST` | **Ketiga tab menampilkan baris yang persis sama.** Bukan layar kosong yang terlihat rusak — tiga tab yang tampak bekerja sambil menampilkan hal yang salah |

> **Dikoreksi 2026-09-27.** Permintaan ini semula memuat TIGA kolom. Work Owner memeriksa
> kolom tabelnya langsung, dan dua di antaranya dikeluarkan:
>
> | Kolom | Kenyataan |
> |---|---|
> | `PXCREATEOPNAME` | **sudah ada.** Saya menyimpulkannya tidak ada karena `inboxoutstanding` memakai `PXCREATEOPERATOR` — dugaan, bukan bukti |
> | `STATUSCLAIM_1` | **tidak lagi dipakai** modul ini; Status Klaim diturunkan dari `PYSTATUSWORK`. Barisnya tetap di migrasi untuk modul lain |
>
> Kekeliruan itu berakibat nyata bila lolos: `ALTER TABLE ... ADD` polos **gagal dengan
> ORA-01430** pada kolom pertama yang sudah ada, dan menyisakan tabel setengah jadi.

> **Panjang K-1 belum diverifikasi ke katalog.** Ia belum ikut dalam pembacaan
> `ALL_TAB_COLUMNS` 2026-09-22. `VARCHAR2(128 CHAR)` mengikuti kolom sekerabat yang memang
> terbaca saat itu (`PXUPDATEOPNAME`, `PXASSIGNEDUSERNAME`). **Mohon dipastikan sebelum
> dijalankan** — kolom yang terlalu sempit memotong nilai tanpa satu pun galat.

**Satu hal yang tidak selesai dengan menjalankan DDL.** Kolom yang ditambahkan **tidak
terisi dengan sendirinya**; proses pengisi tabel harus ikut diperluas. Tanpa itu hasilnya
sama dengan keadaan `STATUSLOCK_1` dan `REQUESTSURVEY_1` hari ini: kolomnya ada, isinya
tidak pernah ditulis — dan ketiga tab mengembalikan nol baris **tanpa satu pun galat**.

Perintah `PENYIMPANAN=oracle -periksa` sudah dipasangi penjaga yang gagal menyebut ketiga
nama kolom itu, sehingga ketiadaannya ketahuan saat pemeriksaan lingkungan dan bukan saat
petugas membuka layar. Yang **tidak** dapat dibuktikannya: apakah kolomnya benar-benar
terisi.

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
