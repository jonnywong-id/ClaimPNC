# Current Architecture — Sistem Pega yang Berjalan

Peta arsitektur sistem yang ada sekarang, beserta utang teknis yang harus dijawab oleh desain
baru. Seluruh angka diukur langsung dari export rule XML Pega. **Snapshot v1.0 memakai 2.167
rule; snapshot berlaku sejak 2026-09-14 memuat 2.634 berkas XML**, ditambah 55 `.prc` dan 8
`.fnc` di folder `Database/` yang baru diterima.

---

## 1. Gambaran umum

```
┌──────────────────────────────────────────────────────────────────┐
│                      Browser (Portal Pega)                       │
│         74 Harness · 269 Section · Navigasi 47 menu              │
└────────────────────────────┬─────────────────────────────────────┘
                             │ HTTP (server-rendered)
┌────────────────────────────▼─────────────────────────────────────┐
│                  Pega PRPC 8.3 (JBoss / WebLogic)                │
│                                                                  │
│  Case Types:  Work-PNC · Work-Komite · Work-OpenProtection       │
│               Work-ReceiveDocument                               │
│                                                                  │
│  Rule:  902 Activity (15.063 step) · 80 Data Transform           │
│         70 When · 29 Flow Action · 56 Report Definition          │
│         652 Connect-SQL · 21 Connect-REST · 7 Data Page          │
│                                                                  │
│  Framework: GCNMFW (Claim) ── di atas ── GISFW (Policy/UW)       │
└──────┬────────────────────────────────────────────┬──────────────┘
       │ JDBC                                       │ HTTPS
┌──────▼──────────────────────────────┐   ┌─────────▼──────────────┐
│         Oracle Database              │   │  Sistem Eksternal      │
│                                      │   │                        │
│  DATAPEGA  — tabel engine Pega       │   │  BRI Surf              │
│  POOLDATA  — data bisnis inti        │   │  Arsip (app8)          │
│  GENERAL   — storage & token         │   │  Storage (app13)       │
│  GL · COLLECTION · MBU · ANEKA       │   │  AI Image              │
│                                      │   │  History Payment       │
│  245 tabel · 70 proc/function        │   │  SMTP                  │
└──────┬───────────────────────────────┘   └────────────────────────┘
       │ DB Link (64 pemakaian)
┌──────▼───────────────────────────────────────────────────────────┐
│  @ASMD (55×) · @SIMASNET · @SMI · @OPJAVA · @PROD_ASM · @PROD_TKA │
│  HRD · GL Payment · Master Sales · Polis · Jam Kerja · Treaty     │
└──────────────────────────────────────────────────────────────────┘
```

---

## 2. Lapisan aplikasi

### 2.1 Lapisan presentasi
- **74 Harness** = layar utuh (inbox, form, laporan, dashboard).
- **269 Section** = komponen UI yang dirakit ke dalam harness. **268 di antaranya memakai
  repeat/grid.**
- **29 Flow Action** = aksi yang bisa dijalankan pengguna pada satu tahap alur, masing-masing
  memetakan ke satu Section plus pre/post processing.
- Semua dirender di **sisi server** oleh Pega. Tidak ada API terpisah; UI dan logika menyatu.

### 2.2 Lapisan proses
- **4 Flow** mendefinisikan alur kerja per case type.
- **Assignment** menempatkan tugas ke **Worklist** (per orang) atau **Workbasket** (antrean bersama).
- **Router** menentukan penerima tugas: `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter`,
  `KomiteRouter`, `PNCAdminRouterRCV`, `ToCurrentOperator`, `ToWorkList`, `ToWorkbasket`.
- **Ticket** memungkinkan lompatan ke tahap mana pun, di luar urutan alur.

### 2.3 Lapisan logika
- **902 Activity** dengan **15.063 step**, rata-rata 17,9 step per activity.
- Distribusi ruleset: **GCNMFW 633** · **GISFW 141** · Pega OOTB 124 · GKM 4.
- Metode yang paling sering dipakai: `Property-Set` (6.953), `RDB-List` (1.144), `Page-New` (938),
  `Page-Copy` (662), `Java` (253 + 69), `Obj-Save` (109), `Commit` (103), `Connect-REST` (56).
- **335 step berisi Java mentah** (326 KB) — logika yang menembus abstraksi Pega.

### 2.4 Lapisan data
- **652 Connect-SQL** berisi **534 KB SQL mentah**: 636 browse, 43 save, 16 delete, 2 open.
- **7 Data Page**, seluruhnya `refresh=never` — **praktis tidak ada lapisan caching**.
- **56 Report Definition** untuk inbox dan laporan.

---

## 3. Struktur database

### 3.1 Pembagian schema

| Schema | Peran | Contoh tabel |
|---|---|---|
| `POOLDATA` | Data bisnis inti | `T_CLAIM_PNC` (124×), `GCNM_PROGRESS_CLAIM` (119×), `T_CLAIM_ADJUSTMENT` (92×), `M_KOMUNIKASI_PNC` (46×), `T_CLAIM_KOMITE_LIST` (33×), `JSON_KLAIM`, `JSON_POLIS` |
| `DATAPEGA` | Tabel engine Pega | `PC_ASM_FW_GCNMFW_WORK` (116×), `PC_ASSIGN_WORKLIST` (18×), `PR_OPERATORS`, `PC_LINK_ATTACHMENT` |
| `GENERAL` | Storage & token | `T_STORAGE_IMAGE`, `MST_BUKA_PROTEKSI` |
| `GL` · `COLLECTION` · `MBU` · `ANEKA` · `HRDASM` | Domain lain | diakses lewat DB Link |

**245 tabel berbeda** teridentifikasi dari SQL.

### 3.2 Dua model penyimpanan yang hidup berdampingan

Yang membingungkan: **data klaim yang sama disimpan dua kali** dengan bentuk berbeda.

1. **Tabel relasional `POOLDATA`** — `T_CLAIM_PNC`, `T_CLAIM_OBJECTLIST`,
   `T_CLAIM_OBJECTCOVERAGE`, `T_CLAIM_ADJUSTMENT`, dan seterusnya.
2. **Dokumen JSON** — `JSON_KLAIM.DATA_JSONBLOB` dan `JSON_POLIS.DATA_JSONBLOB`, dibaca dengan
   `JSON_TABLE` / `JSON_VALUE` (222 pemakaian di 29 rule).

Sinkronisasi antara keduanya dilakukan **stored procedure** (`PEGA_JSON_KLAIM_PNC`,
`PEGA_CONVERT_JSONKLAIM_PNC`, `CONVERTJSONPRODUCTION`). Bila prosedur gagal atau tidak dipanggil,
kedua representasi menjadi tidak konsisten — dan tidak ada mekanisme yang mendeteksinya.

### 3.3 Konstruksi Oracle yang dipakai

| Konstruksi | Volume |
|---|---|
| `TO_CHAR` | 411× di 79 rule |
| `CASE WHEN` | 317× |
| `JSON_VALUE` / `JSON_QUERY` | 195× di 15 rule |
| `TRUNC` | 150× |
| `NVL` | 100× |
| `SYSDATE` | 69× |
| `ROWNUM` | 68× di 44 rule |
| DB Link `@` | 64× di 27 rule |
| Blok PL/SQL `BEGIN…END` | 44 rule |
| `JSON_TABLE` | 27× di 14 rule |
| `LISTAGG` · `(+)` · `CONNECT BY` · `KEEP DENSE_RANK` | sedikit tapi ada |

### 3.4 Stored procedure dan function database

Logika bisnis nyata hidup di dalam database. Yang teridentifikasi antara lain:
`PEGA_JSON_KLAIM_PNC`, `INSERT_PLADLA`, `INSERT_SURVEYORLIST`, `INSERT_KPIADJUSTER`,
`INSERT_SALVAGE_DETAILS`, `CONVERTJSONPRODUCTION`, `INSERTDATAAIKLAIMPNC`,
`INSERTDATAKOMITELIST`, `ADD_NEWMASTERVIRTUALACCOUNT`, `PNC_INSERT_EMAIL_ADJUSTER`,
`GENERAL.GET_TOKEN_STORAGE`, `COLLECTION.P_GET_DATA_REFUND`, `GET_POSISI_PROGRESS_PNC`,
`BASE64ENCODE`, dan puluhan lainnya.

> **Source procedure ini tidak ada di export XML.** Isinya belum pernah dilihat. → **R-01**

---

## 4. Utang teknis yang harus dijawab desain baru

### 4.1 Kunci teknis Pega bocor ke data bisnis

```sql
CLAIMID = 'ASM-FW-GCNMFW-WORK ' || {no_klaim}
```

Nama kelas internal Pega tertanam sebagai bagian dari primary key di tabel bisnis. Akibatnya
data bisnis tidak bisa dibaca tanpa mengetahui konvensi internal Pega, dan sistem apa pun yang
menggantikan Pega tetap terikat pada nama itu.
→ Dijawab oleh **D-22** dan **D-71** (format `PNCN.YY.xxxx` tanpa prefix Pega).

### 4.2 Alias kolom yang menyesatkan

Contoh nyata dari `BroswseKlaimByRegisterDate-SQL`:

```sql
a.LOCATION      AS "RISKLOCATION"
b.NOPOLIS       AS "NoKTP"
a.picteknik     AS "UserAdmin"
a.ttlos         AS "CaseID"
a.BUSINESSNAME  AS "NOPOLIS"      -- nama bisnis dialiaskan jadi "nomor polis"
a.leader_member AS "BUSINESSTYPE"
```

Penyebabnya: developer memaksa nama kolom agar cocok dengan property clipboard Pega yang sudah
ada, alih-alih membuat property baru. Akibatnya **nama tidak lagi mencerminkan isi**, dan
membaca query berarti menebak.
→ Dijawab oleh **D-19** (penamaan ulang menyeluruh mengikuti `CONTEXT.md`).

### 4.3 Nilai bisnis di-hardcode di dalam logika

Ditemukan di `InputRegister_act` dan `GetKomiteApproval`:

Angka di bawah adalah hasil verifikasi terhadap **seluruh export**, bukan terhadap dua rule saja
seperti pada v1.0. Nilai sensitif **tidak direproduksi** sesuai aturan penulisan `D-69`; lokasinya
dirujuk dengan `berkas:baris` pada `docs/verifikasi-bukti-adr.md` §7.

| Jenis | Jumlah terverifikasi | Catatan |
|---|---|---|
| **Alamat email** | **66 unik** | termasuk **≥6 akun Gmail pribadi di jalur produksi** dan **5 alamat yang dipakai sebagai Operator ID** di filter laporan KPI |
| **Operator ID** | **24 unik** | antara lain `MORASOTARDODOTARIGAN`, `ELLENSUPRIYATI`, `IRMANOPITAPURBA_1`, `RATNAGUSNITASARI`; sebagian tertanam **di dalam teks SQL** |
| **Ambang komite** | **8 unik** | antara lain `50000000`, `30000000`, `20000000`, `7000`, `3500`; ditambah 7 ambang uang non-komite |
| **Hostname penentu perilaku** | **3 unik, 48 perbandingan** | host **dev** (34), host **entitas Timor-Leste** (1), host **entitas Insurtech** (13) |

> **Koreksi daftar hostname.** Versi v1.0 menyebut host produksi utama sebagai salah satu dari
> tiga. Itu **tidak akurat** — host itu hanya muncul sebagai **konstanta URL**, tidak pernah
> dibandingkan. Hostname ketiga yang benar-benar menentukan perilaku adalah host **entitas
> Insurtech**. Angka 3 kebetulan tetap benar; daftarnya tidak.

**Yang paling serius:** `Activity/InputRegister_act-Act.xml` memuat **empat step** bertanda
`// TESTING` (`:16693`, `:16830`, `:16998`, `:17141`) yang **menimpa email Underwriting dan
pimpinan dengan alamat penguji** — salah satunya alamat Gmail pribadi — lalu tetap berada di jalur
produksi.

Teori bahwa blok itu nonaktif karena bertanda `pyStepsBlockName=//` sudah diuji dan **gugur**:
`//` dipakai **952× di 279 activity** berdampingan dengan label bermakna (`ERR`, `EXT`, `END`,
`SKIP`), dan format export ini **tidak memiliki elemen aktif/nonaktif sama sekali**. Perbandingan
langsung membuktikan step TESTING punya **precondition identik** dengan step produksinya dan
berada **sesudahnya**.

→ Dijawab oleh **D-15** (semua jadi konfigurasi; blok TESTING tidak dibawa) dan **`ADR-0025`**,
yang masih `Proposed` karena tujuan penyimpanan rahasia belum ditetapkan (`D-40`).

### 4.4 Zona waktu ditangani manual

Tanggal disimpan dalam GMT, lalu **+7 jam ditambahkan secara manual** di setiap tempat yang
membutuhkannya:

```
@DateTime.addCalendar(.ClaimData.DateOfLoss, 0,0,0,0, 7, 0,0)
```

Ada activity khusus bernama `Set7Hours` untuk keperluan ini. Bila satu tempat lupa memanggilnya,
tanggal bergeser 7 jam tanpa ada yang menyadari — dan pada aturan seperti "Tanggal Lapor ≤ DOL
+ 7 hari", pergeseran ini mengubah hasil validasi.

### 4.5 SQL dirangkai dari string

Pola `{ASIS:...}` menyisipkan nilai **langsung ke dalam teks SQL tanpa parameter binding**:

```sql
WHERE claimid = ... {ASIS:InputData.CARI4}
AND b.pxassignedoperatorid IN {ASIS:TempOperator.CityID}
```

Bahkan potongan klausa SQL disimpan sebagai nilai property:

```
.StatusClaim := "and a.STSTRANSFER ='3'"
.StatusClaim := "and trunc(a.TGLAKSEPTASI)>=to_date('...','dd/mm/yyyy') ..."
```

Ini **risiko SQL injection sekaligus penghalang portabilitas**. → Dijawab di Coding Standards:
seluruh query memakai parameter binding, tanpa perkecualian.

### 4.6 Duplikasi masif per lini bisnis

Pola `Browse*`, `Insert*`, `Grouping*`, `CreateCasePNC*`, `GetObjectFromTable*` digandakan untuk
tiap lini bisnis: `_AsuransiKredit`, `_AutoClaim`, `_Travel`, `_Kredit_PA`. Satu perubahan aturan
harus diterapkan di empat tempat — dan sering hanya diterapkan di sebagian.

### 4.7 Nama rule mengandung salah ketik yang dipertahankan

`Broswse*` (11 rule, seharusnya `Browse`), `Complience` (seharusnya `Compliance`),
`Proccedure`, `SALAVAGEDOCUMENT`, `CATRGORY`. Ini menyulitkan pencarian dan menandakan tidak
adanya proses review penamaan.

### 4.8 Sistem masih aktif berubah

Distribusi tahun perubahan terakhir activity:

| Tahun | 2016 | 2017 | 2018 | 2019 | 2020 | 2021 | 2022 | 2023 | 2024 | 2025 | 2026 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| Jumlah | 2 | 35 | 127 | 43 | 108 | 58 | 58 | 97 | **180** | 70 | **124** |

**124 activity berubah pada 2026** — sistem sumber adalah sasaran bergerak. Strategi Strangler
Fig (D-05) harus memperhitungkan bahwa Pega akan terus berubah selama migrasi berjalan.

---

## 5. Yang tidak ada di export dan harus dilengkapi

| Yang hilang | Dampak | Risiko |
|---|---|---|
| Source 64 procedure & function database | Logika bisnis tidak diketahui | **R-01** |
| ~~Rule Agent / Queue Processor~~ | ✅ **diterima 2026-09-09** — 5 job + 1 agent (`D-57`) | ~~R-02~~ tertutup |
| ~~Isi master `V_STS_CLAIM`~~ | ✅ **diterima** — **33 kode `1134`–`1166`**, bukan 10 kode | ~~R-06~~ tertutup |
| `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter` | Aturan penugasan tidak diketahui | **R-04** |
| 50 activity yang dipanggil tapi tidak diekspor | Termasuk `SendEmailNotification` (dipanggil 15×), `generatePDF`, `SetTicket` | **R-07** |
| DDL tabel (tipe kolom, index, constraint) | Desain skema harus menebak | **R-08** |
