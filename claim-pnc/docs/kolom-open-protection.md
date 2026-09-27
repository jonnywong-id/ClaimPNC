# `POOLDATA.T_CLAIM_OPENPROTECTION`

Tabel Open Protection alur klaim — **sudah dibuat**, dan inilah bentuk yang dipakai kedua
modulnya.

| | |
|---|---|
| **Tanggal** | 2026-09-24 |
| **Untuk** | Work Owner dan DBA |
| **Modul yang memakainya** | `input-req-protection` · `inbox-accept-open-protection` |
| **Keadaan** | 16 kolom · primary key · 1 CHECK · 3 index · **sequence** · **master tipe terisi 9 baris** |
| **Sumber** | dibaca langsung dari katalog Oracle (`ALL_TAB_COLUMNS`, `ALL_SEQUENCES`) di `DEV_PEGA83G` |
| **Revisi** | bentuk berubah 2026-09-24 — tiga kolom berganti nama, pencacah menjadi sequence, master tipe ditambahkan |

---

## 1. Bentuk tabel

```sql
CREATE TABLE POOLDATA.T_CLAIM_OPENPROTECTION
(
  OPEN_PROTECTION_ID  VARCHAR2(20 BYTE),
  CREATE_DATE         TIMESTAMP(6),
  CREATED_BY          VARCHAR2(100 BYTE),
  RESOLVED_BY         VARCHAR2(100 BYTE),
  RESOLVED_DATETIME   TIMESTAMP(6),
  POLICY_NO           VARCHAR2(20 BYTE),
  CLAIM_NO            VARCHAR2(10 BYTE),
  ID_CLAIM            VARCHAR2(100 BYTE),
  PROTECTION_TYPE_ID  VARCHAR2(2 BYTE),
  APPROVAL_STATUS     VARCHAR2(10 BYTE),
  NOTES               VARCHAR2(2000 BYTE),
  OLD_DATA            VARCHAR2(400 BYTE),
  NEW_DATA            VARCHAR2(400 BYTE),
  OBJECT_NAME         VARCHAR2(500 BYTE),
  BRANCH_NAME         VARCHAR2(100 BYTE),
  STATUS_ACTIVE       VARCHAR2(10 BYTE) DEFAULT '1'
)
```

Bentuk ini sudah diverifikasi langsung ke katalog, dan kedua adapter Oracle ditulis
menurutnya — bukan menurut usulan. Tabelnya sudah **empat kali** dibentuk ulang selama
penyusunan, dan yang berlaku selalu keadaan terakhirnya.

### Tiga kolom berganti nama pada revisi 2026-09-24

| Sebelum | Sesudah |
|---|---|
| `ID` | `OPEN_PROTECTION_ID` |
| `PROTECTION_TYPE` | `PROTECTION_TYPE_ID` |
| `RESOLVED_DATE_TIME` | `RESOLVED_DATETIME` |

Index dan constraint **ikut berpindah sendiri** — tabelnya di-`ALTER`, bukan dibuat ulang,
sehingga kelimanya tetap `VALID` tanpa perlu dibuat lagi. Diverifikasi ke katalog, bukan
diandaikan.

`PROTECTION_TYPE_ID` sekaligus menyempit dari `VARCHAR2(100)` menjadi `VARCHAR2(2)`. Itu
sejalan dengan isi nyatanya: produksi hanya memakai satu digit `1`–`9`.

### ⚠️ `CLAIM_NO` menyempit menjadi 10, dan itu TERLALU SEMPIT

| Bentuk | Panjang | Muat di 10? |
|---|---:|---|
| `PNC-1865` — warisan Pega, 118 dari 160 baris produksi | 8 | ✅ |
| `PNCN.26.99` — nomor klaim baru (`D-71`), klaim ke-99 | 10 | ✅ batas persis |
| **`PNCN.26.100`** — klaim ke-100 dan seterusnya | **11** | ❌ |
| dua baris warisan di produksi, tak berpola `PNC-` | 11 dan 16 | ❌ |

`D-71` menetapkan nomor klaim sistem baru berbentuk **`PNCN.YY.xxxx`** — dua belas huruf.
Jadi begitu klaim ke-100 terbit, ia tidak dapat ditautkan ke proteksi mana pun.

```sql
ALTER TABLE POOLDATA.T_CLAIM_OPENPROTECTION
    MODIFY (CLAIM_NO VARCHAR2(32 BYTE));
```

#### Kenapa 32, padahal 20 cukup

Ditetapkan Work Owner 2026-09-24 setelah keempat pilihan dihitung, bukan diwarisi dari lebar
sebelumnya:

| Yang harus muat | Panjang |
|---|---:|
| `PNC-1865` — warisan, 118 dari 160 baris produksi | 8 |
| `PNCN.26.0001` — nomor klaim baru (`D-71`) | **12** |
| `PNCN.26.12345` — bila sequence klaim tembus lima digit | 13 |
| baris warisan terpanjang di produksi, bukan pola `PNC-` | **16** |

**16 sudah memuat seluruh data yang ada hari ini**, dan **20** akan menyamakannya dengan
`OPEN_PROTECTION_ID` — bentuk nomornya identik (`OPCN.YY.xxxx` versus `PNCN.YY.xxxx`), jadi
ruang tumbuhnya pun akan sama: dua belas digit untuk nomor urut.

Work Owner tetap memilih **32**. Konsekuensinya diterima: nomor klaim mendapat ruang lebih
besar daripada nomor proteksi yang menampungnya. Itu tidak merusak apa pun — hanya ruang
yang tidak akan terpakai.

Sampai itu dijalankan, aplikasi memvalidasi pada lebar **sasaran** (32), bukan lebar hari
ini (10). Memvalidasi pada 10 akan menolak seluruh nomor klaim sistem baru dengan pesan
yang seolah-olah menyalahkan pengguna; dengan lebar sasaran, kegagalannya muncul sebagai
`ORA-12899` yang menyebut nama kolom — dan itu menunjuk tempat yang benar.

---

## 2. Pemetaan kolom

| Kolom | Properti Pega | Kolom layar | Ditulis modul |
|---|---|---|---|
| `OPEN_PROTECTION_ID` | `.pyID` | **No Proteksi** | `input-req-protection` |
| `POLICY_NO` | `.PolicyNo` | **No Polis** | `input-req-protection` |
| `CLAIM_NO` | `.CaseID` | **No Klaim** | `input-req-protection` |
| `ID_CLAIM` | `.PNCCaseID` | hanya bila BERBEDA dari `CLAIM_NO` | `input-req-protection` (diturunkan) |
| `PROTECTION_TYPE_ID` | `.TypeProtection` | **Tipe Proteksi** | `input-req-protection` |
| `CREATE_DATE` | `.InputDate` | **Tanggal Proteksi Dibuat** | `input-req-protection` |
| `CREATED_BY` | `.pxCreateOpName` | **User Create** | `input-req-protection` |
| `NOTES` | `.Keterangan` | **Keterangan** | `input-req-protection` |
| `OLD_DATA` | lihat §3 | detail perubahan (form) | `input-req-protection` |
| `NEW_DATA` | lihat §3 | detail perubahan (form) | `input-req-protection` |
| `OBJECT_NAME` | `.ObjectName` | Object Name (form) | `input-req-protection` |
| `BRANCH_NAME` | `.BranchName` | Branch Name (form) | `input-req-protection` |
| `STATUS_ACTIVE` | — | — (soft delete `D-66`) | `input-req-protection` |
| `APPROVAL_STATUS` | `.AcceptStatus` | penyaring inti | **`inbox-accept-open-protection`** |
| `RESOLVED_BY` | `.AcceptOpName` | — | **`inbox-accept-open-protection`** |
| `RESOLVED_DATETIME` | `.AcceptDate` | — | **`inbox-accept-open-protection`** |

**Pembagian kolom inilah yang menjaga `P-1`.** Dua modul menyentuh satu tabel, tetapi
menulis kolom yang berbeda pada tahap hidup yang berbeda. Ditegakkan lewat rute yang tidak
didaftarkan dan uji di `query_test.go` masing-masing.

`CREATE_DATE` memikul dua peran sekaligus — tanggal proteksi dibuat dan waktu baris dibuat.
Tabel hanya punya satu kolom waktu pembuatan, dan sistem lama pun tidak membedakan keduanya.

### `ID_CLAIM` = `CLAIM_NO` pada baris baru

Work Owner menegaskan 2026-09-24: **ClaimNo dan ClaimID berisi nilai yang sama**, yaitu
`PNCN.YY.xxxx`.

| | `CLAIM_NO` | `ID_CLAIM` |
|---|---|---|
| Baris **warisan Pega** | `PNC-1865` | `ASM-FW-GCNMFW-WORK PNC-1865` |
| Baris **sistem baru** | `PNCN.26.0007` | `PNCN.26.0007` |

Server **menurunkan** `ID_CLAIM` dari nomor klaim; form tidak menanyakannya kedua kalinya.
Dua isian yang wajib sama tetapi diketik terpisah akan berbeda cepat atau lambat, dan
perbedaannya tidak menghasilkan galat apa pun — hanya proteksi yang menunjuk dua klaim
berbeda.

Kolomnya **tetap ada dan tetap dibaca**: pada baris warisan nilainya memang berbeda, dan
menghapusnya akan membuat baris warisan tampak seolah ClaimID-nya sama dengan nomor
klaimnya. Form menampilkannya hanya ketika berbeda.

Satu akibat yang perlu dinyatakan: pesan sistem lama **"Silakan Tulis dan Cari Ulang No
Klaim"** tidak dibawa. Ia menegur pengguna Pega yang mengetik nomor tanpa menekan tombol
CARI; di sini tidak ada langkah kedua itu, sehingga syaratnya tidak dapat tercapai. Yang
hilang bersamanya: pencarian itu juga **membuktikan klaimnya ada**. Di sini tidak ada yang
membuktikannya — modul klaim belum terpasang, dan itu keterbatasan yang sudah tercatat
sebelum perubahan ini.

---

## 3. `OLD_DATA` / `NEW_DATA`

Sepasang kolom "nilai sebelum" dan "nilai sesudah"; artinya ditentukan `PROTECTION_TYPE`:

| Tipe | `OLD_DATA` | `NEW_DATA` |
|---|---|---|
| `7` Perubahan DOL | Current Date Of Loss | Next Date Of Loss |
| `8` Perubahan Cause Of Loss | Cause Of Loss sebelumnya | Cause Of Loss dipilih |

Bentuknya meniru `POOLDATA.T_OPENPROTECTION.OLDATA`/`NEWDATA`, yang memakai pola sama untuk
tipe proteksinya sendiri — di sana isinya berupa timestamp, nama, atau keterangan rate,
tergantung tipenya.

**Ejaan `OLD_DATA` sengaja tidak meniru `OLDATA`** milik tabel itu, yang kurang satu `D`.
View `POOLDATA.OPENPROTECTION` di atasnya sudah memperbaikinya.

**Tanggal disimpan sebagai teks `YYYY-MM-DD`,** bukan `DATE`. Konsekuensi dari sepasang kolom
yang melayani dua tipe, dan akibatnya disadari: *"tampilkan permintaan yang mengubah DOL ke
bulan September"* tidak dapat dijawab SQL. Ketiga layar tidak menyaring maupun mengurutkan
berdasarkan isi ini — ia hanya ditampilkan.

---

## 4. `APPROVAL_STATUS` wajib `NULL`, bukan teks kosong

Satu-satunya butir yang bila keliru **tidak menghasilkan galat apa pun**.

| Nilai | Arti | Bukti |
|---|---|---|
| `NULL` | belum diakseptasi | ketiga RD menyaring `IS NULL` |
| `'1'` | disetujui | `When/IsAcceptProtection-When.xml` |
| `'2'` | ditolak | cabang Else `Flow/CreateProtection_Flow.xml` |

Baris ber-`APPROVAL_STATUS = ''` tidak cocok dengan `IS NULL`, sehingga **hilang dari seluruh
inbox tanpa gejala**. Karena itu kolomnya sengaja tanpa `DEFAULT`, dan `protection_insert`
tidak menyebutnya sama sekali — nilainya jatuh ke `NULL` dengan sendirinya.

---

## 5. Constraint dan index

**Seluruhnya sudah dibuat.** Diverifikasi 2026-09-23 lewat `ALL_INDEXES`, `ALL_IND_COLUMNS`,
dan `ALL_CONSTRAINTS` — bukan dengan membaca kembali teks DDL-nya.

| Objek | Tipe | Isi | Status |
|---|---|---|---|
| `T_CLAIM_OPENPROTECTION_PK` | `P` · UNIQUE | `OPEN_PROTECTION_ID` | `ENABLED VALIDATED` |
| `T_CLAIM_OPENPROT_CK_STATUS` | `C` | `APPROVAL_STATUS IN ('1','2')` | `ENABLED VALIDATED` |
| `IX_T_CLAIM_OPENPROT_INBOX` | FUNCTION-BASED | `APPROVAL_STATUS ASC`, `CREATE_DATE DESC` | `VALID` |
| `IX_T_CLAIM_OPENPROT_ANTREAN` | NORMAL | `APPROVAL_STATUS`, `PROTECTION_TYPE_ID`, `CLAIM_NO` | `VALID` |
| `IX_T_CLAIM_OPENPROT_GANDA` | NORMAL | `POLICY_NO`, `PROTECTION_TYPE_ID`, `CREATE_DATE` | `VALID` |

### Primary key — kini jaring pengaman, bukan lagi pengaman utama

Sebelum sequence ada, adapter menurunkan nomor dengan `MAX+1` dan bergantung pada constraint
ini untuk menahan dua permintaan bersamaan — yang kedua gagal `ORA-00001` lalu dicoba ulang.

Sejak `CLAIM_PROTECTION_SEQ` dibuat, bentroknya tidak lagi mungkin: setiap pemanggil
`NEXTVAL` menerima nilai berbeda tanpa membaca tabel. Percobaan ulangnya **dihapus**, bukan
disederhanakan — percobaan ulang yang dipertahankan setelah sebabnya hilang justru
menyembunyikan bentrok yang menandakan hal lain, misalnya nomor yang disisipkan tangan.

Constraint-nya tetap berguna sebagai jaring terakhir terhadap penyisipan di luar aplikasi.

### CHECK membolehkan `NULL`, dan itu memang yang dibutuhkan

`IN ('1','2')` tidak menolak `NULL` — sebuah perbandingan dengan `NULL` menghasilkan
`UNKNOWN`, dan CHECK hanya menolak yang tegas `FALSE`. Jadi ketiga keadaan yang sah tetap
lolos: `NULL` belum diakseptasi, `'1'` disetujui, `'2'` ditolak. Yang tertutup adalah nilai
lain — termasuk **teks kosong**, yang §4 catat sebagai jebakannya.

### Kenapa `IX_..._INBOX` tercatat FUNCTION-BASED

`CREATE_DATE DESC` membuat Oracle menyimpan kolom virtual tersembunyi, dan
`ALL_IND_COLUMNS` menampilkannya sebagai `SYS_NC00017$` alih-alih nama kolomnya. Itu
**perilaku normal index menurun**, bukan tanda salah buat — `DESCEND = DESC` pada baris yang
sama menegaskannya.

Kolom keduanya bukan sekadar demi urutan. B-tree Oracle **tidak menyimpan entri yang seluruh
kolomnya `NULL`**, sehingga index berkolom tunggal atas `APPROVAL_STATUS` tidak akan pernah
terpakai oleh penyaring `APPROVAL_STATUS IS NULL` — penyaring inti ketiga layar. Kolom kedua
yang terisi itulah yang membuat entrinya tetap tersimpan dan `IS NULL` dapat di-index.

> **Belum diukur:** rencana eksekusinya belum diperiksa, dan pada tabel berisi satu baris
> pemeriksaan itu tidak bermakna — optimizer akan memilih pemindaian penuh apa pun index-nya.
> Pengukuran yang sahih menunggu tabelnya terisi data nyata.

### Hak akses

```sql
-- GRANT SELECT, INSERT, UPDATE ON POOLDATA.T_CLAIM_OPENPROTECTION TO <AKUN_APLIKASI>;
```

**Tanpa `DELETE`.** `D-66` menetapkan tidak ada penghapusan fisik pada data bernilai bisnis;
penghapusan dinyatakan lewat `STATUS_ACTIVE`. Menegakkannya lewat hak akses, bukan hanya
lewat kode, adalah yang `09-DATABASE-STRATEGY.md` §8 tuntut.

---

## 6. Penomoran `OPEN_PROTECTION_ID`

Work Owner menetapkan **`OPCN.YY.xxxx`**, sejajar dengan `PNCN.YY.xxxx` pada `D-71`. Nomor
warisan Pega berbentuk `OPC-XXX` dan dibaca apa adanya; keduanya hidup berdampingan.

### ✅ Pencacahnya sudah ada — sequence

Dibuat Work Owner 2026-09-24, diverifikasi lewat `ALL_SEQUENCES`:

```sql
CREATE SEQUENCE POOLDATA.CLAIM_PROTECTION_SEQ
  START WITH 1 MAXVALUE 999999999999999 MINVALUE 1
  NOCYCLE NOCACHE NOORDER;
```

`NEXTVAL` saat diperiksa masih **1** — belum satu pun nomor terbit.

Ini menggantikan `MAX+1` yang dipakai sebelumnya. Perubahannya bukan kerapian: `MAX+1`
membaca isi tabel, sehingga dua permintaan bersamaan dapat membaca nilai yang sama dan
tertahan hanya oleh primary key, dengan percobaan ulang sebagai penambalnya.

### Tiga sifat yang mengikuti, dan diterima

| Sifat | Akibatnya |
|---|---|
| **Global, tidak direset tiap tahun** | nomor urut menembus pergantian tahun (`OPCN.26.0009` → `OPCN.27.0010`). Segmen tahun menjadi **penanda**, bukan penghitung per tahun |
| **`NOCACHE`** | setiap pemanggilan menulis ke kamus data — lebih lambat, tetapi tidak membuang blok nomor saat basis data direstart |
| **Tidak kembali saat rollback** | deret nomor dapat **berlubang**. Itu sifat sequence di Oracle maupun PostgreSQL, dan lubangnya bukan tanda kerusakan |

### Dua hal yang BERBEDA dari sintaks yang dituliskan Work Owner

Sintaksnya berbunyi `'OPCN' || '.' || TO_CHAR(SYSDATE,'RR') || '.' || TO_CHAR(seq.NEXTVAL)`.
Yang dipakai berbeda pada dua hal, keduanya disengaja:

1. **Nomor urut dipadatkan nol sampai empat digit.** Tanpa format mask, lebar segmen
   terakhir berubah-ubah sehingga `.10` mendahului `.9` saat diurutkan sebagai teks — dan
   daftar proteksi memang memakai nomor sebagai pemutus seri. `D-71` butir 2 mencatat cacat
   yang sama untuk nomor klaim.
2. **Tahun datang dari aplikasi, bukan `SYSDATE`.** `SYSDATE` jam server basis data; nomor
   yang terbit di sekitar pergantian tahun akan bergeser terhadap tanggal WIB (`R-12`).

Keduanya dilaporkan, bukan diterapkan diam-diam. Bila Work Owner menghendaki sintaksnya
persis, yang berubah hanya `FormatNumber`.

### Batas kolom versus batas sequence

`OPEN_PROTECTION_ID` `VARCHAR2(20)`; delapan huruf terpakai awalan dan pemisah, tersisa
**dua belas** digit. `MAXVALUE` sequence memuat **lima belas**. Selisih itu baru menggigit
pada nomor urut ke-1.000.000.000.000 — dicatat di sini, tidak dijaga di kode.

---

## 6a. `POOLDATA.M_CLAIM_PROTECTION_TYPE` — master tipe

Diterima 2026-09-24, **sudah terisi penuh**:

```sql
CREATE TABLE POOLDATA.M_CLAIM_PROTECTION_TYPE
( PROTECTION_TYPE_ID    VARCHAR2(2 BYTE),
  PROTECTION_TYPE_NAME  VARCHAR2(100 BYTE) );
```

| ID | Nama | ID | Nama |
|---|---|---|---|
| `1` | General | `6` | Klaim >= 50 M |
| `2` | Premi Belum Lunas | `7` | Perubahan DOL |
| `3` | Asuransi Kredit | `8` | Perubahan COL |
| `4` | Pengkinian Data | **`9`** | **Nama Rekening Tidak Sesuai** |
| `5` | Currency Klaim | | |

**Ia menutup pertanyaan yang sejak awal menggantung.** Label `1`, `3`, `4`, `5`, `6`, dan
`9` tidak ada di satu pun berkas export (`R-16`), sehingga layar menampilkan kodenya apa
adanya. Keputusan itu ternyata menyelamatkan tipe **`9`**, yang dipakai **25 baris
produksi** dan **nol kemunculan di export** — daftar tebakan mana pun akan melewatkannya.

### Tiga kode tetap hidup di dalam kode, dan itu bukan hardcode

`'2'`, `'7'`, dan `'8'` tetap menjadi konstanta karena ia **percabangan**, bukan label:
`'2'` memisahkan antrean PREMI, `'7'` dan `'8'` memunculkan panel detail perubahan.

Master hanya punya dua kolom; tidak ada yang menyatakan "tipe ini masuk antrean premi".
Menurunkan perilaku dari NAMA akan membuat satu suntingan ejaan mengubah antrean akseptasi —
tanpa galat dan tanpa gejala.

### Dibaca dengan LEFT JOIN, bukan INNER

Kode yang tidak terdaftar di master tetap **tampil sebagai kodenya**. `INNER JOIN` akan
menghilangkan barisnya dari inbox tanpa satu pun gejala, justru pada baris yang paling perlu
diperiksa manusia. Dijaga uji di `query_test.go`.

---

## 7. `T_OPENPROTECTION` BUKAN sumber data modul ini

Ditegaskan supaya tidak tertukar kelak. Nama keduanya nyaris sama, dan yang satu berisi
6.353 baris hidup.

| Bukti | Artinya |
|---|---|
| `PROTECTIONTYPE` berisi `SourceOfBusiness` (5.940), `BlackList`, `BackDated`, `UsedVehicle`, `Rate`, `ShortPeriod` | pengecualian **underwriting saat penerbitan polis** — bukan perubahan DOL atau Cause of Loss |
| `IDPEGA` mengandung `PNC`: **0 dari 6.353** | bukan work object Claim PNC |
| `GROUPPANEL` memuat `007` | di luar Group Panel klaim (002–006, 009) |
| `APPROVALSTATUS` hanya `1`/`0`, tanpa NULL | tidak punya keadaan "belum diputuskan" |

Ia milik domain polis/underwriting (GISFW), bounded context tim lain (`D-03`). Awalan
`CLAIM` pada tabel ini yang membedakan keduanya.

Yang diambil darinya hanyalah **kosakata kolomnya**, supaya nama di tabel ini dikenali DBA.

---

## 8. Pertanyaan yang menunggu jawaban

| # | Pertanyaan | Kepada |
|---|---|---|
| ~~1~~ | ~~Primary key dan index (§5)~~ — ✅ **selesai 2026-09-23**, seluruhnya `VALID` | — |
| ~~2~~ | ~~Pencacah nomor~~ — ✅ **selesai 2026-09-24**, `CLAIM_PROTECTION_SEQ` | — |
| ~~3~~ | ~~Label `PROTECTION_TYPE`~~ — ✅ **selesai 2026-09-24**, master terisi kesembilannya | — |
| **6** | **`CLAIM_NO` dilebarkan menjadi `VARCHAR2(32)`?** Lihat §1 — hari ini 10, sedangkan `PNCN.YY.xxxx` butuh 12 | **Work Owner** |
| 4 | Siapa yang menulis tabel ini selama masa paralel — hanya aplikasi baru, atau Pega juga (`P-1`) | Work Owner |
| 5 | Apakah tabel **dan masternya** dibuat di setiap basis data portal, atau hanya ASM | Work Owner |
| ~~7~~ | ~~Apakah `ID_CLAIM` diisi untuk baris baru~~ — ✅ **terjawab 2026-09-24**: sama dengan `CLAIM_NO` | — |

> **Butir 6 yang paling mendesak.** Ia satu-satunya yang menghalangi penyimpanan berfungsi
> untuk klaim sistem baru. Butir 5 menyusul: masternya baru diperiksa ada di portal `ASM`.
>
> Butir 3 sudah tertutup, dan cara ia tertutup layak dicatat. `T_OPENPROTECTION` memakai
> **teks** sebagai tipe proteksi (`SourceOfBusiness`, `BlackList`), sedangkan alur klaim
> memakai **kode angka**. Kekhawatiran bahwa keduanya tercampur terjawab: master yang
> diterima memakai kode angka `1`–`9`, sejalan dengan isi produksi.

---

## 9. Nilai yang benar-benar dipakai di produksi

Dibaca dari **205 baris OPC** di `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`
(`PXOBJCLASS = 'ASM-FW-GCNMFW-Work-OpenProtection'`) pada 2026-09-23. Hanya kode dan
hitungannya yang dibaca; tidak ada data nasabah yang dicetak (`D-69`).

### `TYPEPROTECTION` — sembilan nilai, bukan delapan

| Nilai | Baris | Arti | Dari mana |
|---|---:|---|---|
| `<NULL>` | 46 | belum diisi | — |
| `7` | 41 | **Perubahan DOL** | `Section/InputProtectionSection-Section.xml:2728` + master |
| `1` | 32 | **General** | master (2026-09-24) |
| `8` | 29 | **Perubahan COL** | `:3638` + master |
| **`9`** | **25** | **Nama Rekening Tidak Sesuai** | **master saja — NOL kemunculan di export** |
| `2` | 15 | **Premi Belum Lunas** | `InboxOpenProtection2_RD_collection` `= "2"` + master |
| `5` | 9 | **Currency Klaim** | master |
| `4` | 3 | **Pengkinian Data** | master |
| `3` | 3 | **Asuransi Kredit** | master |
| `6` | 2 | **Klaim >= 50 M** | master |

> **Kolom "Arti" di atas semula enam baris berbunyi "belum diketahui".** Master yang
> diterima 2026-09-24 mengisi keenamnya sekaligus — lihat §6a.
>
> **Nilai `9` layak dicatat khusus.** Seluruh penelusuran export hanya menemukan `1`–`8`;
> `9` tidak muncul di satu pun activity, section, maupun when rule, tetapi produksi
> memakainya pada 25 baris — terbanyak kedua di antara nilai yang terisi.
>
> Ia membuktikan keputusan **menampilkan kode apa adanya** memang benar: daftar yang ditebak
> dari export akan melewatkan `9` sepenuhnya, dan 25 baris itu akan tampil kosong atau
> berlabel salah. Aturan yang sama kini menjaga tipe kesepuluh yang mungkin muncul kelak —
> lihat LEFT JOIN di §6a.

### `PYSTATUSWORK` — pemetaan ke `APPROVAL_STATUS`

| `PYSTATUSWORK` | Baris | `APPROVAL_STATUS` |
|---|---:|---|
| `New` | 134 | `NULL` — belum diakseptasi |
| `Resolved-Completed` | 69 | `'1'` — disetujui |
| `Resolved-Rejected` | 2 | `'2'` — ditolak |

Pemetaan ini yang dipakai saat menyalin data warisan, dan ia sejalan dengan
`When/IsAcceptProtection-When.xml` yang menguji `.AcceptStatus = "1"`.

### Kesiapan penyalinan

| Ukuran | Jumlah |
|---|---:|
| Baris OPC seluruhnya | 205 |
| Ber-`PYID` awalan `OPC-` | **205 dari 205** |
| Lengkap (`CASEID` + `POLICYNO` + `TYPEPROTECTION`) | 156 |
| Di antaranya ber-`CLOSECLAIMNOTE_1` | 133 |
| `ISUSEDPNC = '1'` | 13 |

### Kolom asal saat menyalin dari Pega

| Kolom tujuan | Kolom asal di `PC_ASM_FW_GCNMFW_WORK` |
|---|---|
| `OPEN_PROTECTION_ID` | `PYID` |
| `CREATE_DATE` | `PXCREATEDATETIME` |
| `CREATED_BY` | `PXCREATEOPNAME` |
| `POLICY_NO` | `POLICYNO` |
| `CLAIM_NO` | `CASEID` |
| `ID_CLAIM` | `PNCCASEID` |
| `PROTECTION_TYPE_ID` | `TYPEPROTECTION` |
| `NOTES` | `CLOSECLAIMNOTE_1` |
| `BRANCH_NAME` | `BRANCHNAME` |
| `APPROVAL_STATUS` | diturunkan dari `PYSTATUSWORK` (tabel di atas) |

**`OLD_DATA` dan `NEW_DATA` tidak punya kolom asal.** Detail perubahan DOL dan Cause of
Loss tidak diekspos sebagai kolom di tabel kerja Pega — ia tinggal di dalam blob properti.
Baris warisan yang disalin karena itu kosong pada keduanya, dan form akseptasi
menampilkannya sebagai tanda hubung. Yang terisi hanyalah baris yang dibuat aplikasi ini.

> **`KETERANGAN_1` TIDAK dipakai**, meski namanya paling cocok: ia **nol terisi** pada
> seluruh 205 baris OPC. Yang terisi `CLOSECLAIMNOTE_1` (139 baris) — kolom yang namanya
> menyebut penutupan klaim, dipakai ulang untuk keterangan proteksi. Itu contoh lain dari
> alias menyesatkan yang `03-CURRENT-ARCHITECTURE.md` §4.2 catat.

### Baris uji yang sudah disalin

Satu baris disalin pada 2026-09-23 sebagai bukti jalur baca berfungsi: **`OPC-216`**,
`PROTECTION_TYPE = 9`, `APPROVAL_STATUS` NULL, tertaut klaim.

Terbaca di **kedua layar**: muncul di Input Req Protection (dengan tautan MATI karena sudah
tertaut klaim) dan di antrean akseptasi NON PREMI. Penyalinannya memakai `INSERT … SELECT`
sehingga nomor polis dan nomor klaimnya berpindah **di dalam basis data** dan tidak pernah
dibaca ke memori aplikasi.
