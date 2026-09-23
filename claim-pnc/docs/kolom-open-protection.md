# `POOLDATA.T_CLAIM_OPENPROTECTION`

Tabel Open Protection alur klaim — **sudah dibuat**, dan inilah bentuk yang dipakai kedua
modulnya.

| | |
|---|---|
| **Tanggal** | 2026-09-23 |
| **Untuk** | Work Owner dan DBA |
| **Modul yang memakainya** | `input-req-protection` · `inbox-accept-open-protection` |
| **Keadaan** | 16 kolom · primary key · 1 CHECK · 3 index — seluruhnya `VALID` · 1 baris uji |
| **Sumber** | dibaca langsung dari katalog Oracle (`ALL_TAB_COLUMNS`) di `DEV_PEGA83G` |

---

## 1. Bentuk tabel

```sql
CREATE TABLE POOLDATA.T_CLAIM_OPENPROTECTION
(
  ID                  VARCHAR2(100 BYTE),
  CREATE_DATE         TIMESTAMP(6),
  CREATED_BY          VARCHAR2(100 BYTE),
  RESOLVED_BY         VARCHAR2(100 BYTE),
  RESOLVED_DATE_TIME  TIMESTAMP(6),
  POLICY_NO           VARCHAR2(20 BYTE),
  CLAIM_NO            VARCHAR2(32 BYTE),
  ID_CLAIM            VARCHAR2(100 BYTE),
  PROTECTION_TYPE     VARCHAR2(100 BYTE),
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
menurutnya — bukan menurut usulan. Keduanya sempat berbeda tiga kali selama penyusunan, dan
yang berlaku selalu keadaan tabelnya.

---

## 2. Pemetaan kolom

| Kolom | Properti Pega | Kolom layar | Ditulis modul |
|---|---|---|---|
| `ID` | `.pyID` | **No Proteksi** | `input-req-protection` |
| `POLICY_NO` | `.PolicyNo` | **No Polis** | `input-req-protection` |
| `CLAIM_NO` | `.CaseID` | **No Klaim** | `input-req-protection` |
| `ID_CLAIM` | `.PNCCaseID` | tidak ditampilkan | `input-req-protection` |
| `PROTECTION_TYPE` | `.TypeProtection` | **Tipe Proteksi** | `input-req-protection` |
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
| `RESOLVED_DATE_TIME` | `.AcceptDate` | — | **`inbox-accept-open-protection`** |

**Pembagian kolom inilah yang menjaga `P-1`.** Dua modul menyentuh satu tabel, tetapi
menulis kolom yang berbeda pada tahap hidup yang berbeda. Ditegakkan lewat rute yang tidak
didaftarkan dan uji di `query_test.go` masing-masing.

`CREATE_DATE` memikul dua peran sekaligus — tanggal proteksi dibuat dan waktu baris dibuat.
Tabel hanya punya satu kolom waktu pembuatan, dan sistem lama pun tidak membedakan keduanya.

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
| `T_CLAIM_OPENPROTECTION_PK` | `P` · UNIQUE | `ID` | `ENABLED VALIDATED` |
| `T_CLAIM_OPENPROT_CK_STATUS` | `C` | `APPROVAL_STATUS IN ('1','2')` | `ENABLED VALIDATED` |
| `IX_T_CLAIM_OPENPROT_INBOX` | FUNCTION-BASED | `APPROVAL_STATUS ASC`, `CREATE_DATE DESC` | `VALID` |
| `IX_T_CLAIM_OPENPROT_ANTREAN` | NORMAL | `APPROVAL_STATUS`, `PROTECTION_TYPE`, `CLAIM_NO` | `VALID` |
| `IX_T_CLAIM_OPENPROT_GANDA` | NORMAL | `POLICY_NO`, `PROTECTION_TYPE`, `CREATE_DATE` | `VALID` |

### Primary key — pengaman penerbitan nomor, bukan kerapian

Adapter menurunkan nomor urut dengan `MAX+1` di dalam transaksi. Bila dua permintaan tiba
bersamaan, yang kedua gagal `ORA-00001` lalu dicoba ulang. Tanpa constraint ini keduanya
akan tersimpan dengan nomor yang sama **tanpa satu pun galat**.

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

## 6. Penomoran `ID`

Work Owner menetapkan **`OPCN.YY.xxxx`**, sejajar dengan `PNCN.YY.xxxx` pada `D-71`. Nomor
warisan Pega berbentuk `OPC-XXX` dan dibaca apa adanya; keduanya hidup berdampingan.

`ID` bertipe `VARCHAR2(100)` dan **itu memang yang dibutuhkan** — nomor berformat tidak muat
di kolom angka.

### Pencacahnya belum ada, dan bentuknya belum diputuskan

Tidak ada sequence maupun tabel pencacah bernama terkait di `POOLDATA` (diperiksa lewat
`ALL_OBJECTS`, nol hasil). Sampai salah satunya dibuat, adapter menurunkan nomor urut dari
**nomor tertinggi tahun berjalan**, di dalam transaksi yang sama dengan penyisipannya.

Dua pilihan, keduanya sudah ada presedennya:

| Pilihan | Preseden | Sifat |
|---|---|---|
| Sequence Oracle | `CPNC_LAPORAN_KLAIM_SEQ` (`RCVN.YY.xxxx`) | satu deret global; segmen tahun jadi penanda |
| Tabel pencacah per tahun | `CPNC_NOMOR_KLAIM (TAHUN, TERAKHIR)` (`PNCN.YY.xxxx`) | mulai dari 1 tiap tahun; dikunci `FOR UPDATE` |

Nomor klaim memakai yang kedua. **Belum diputuskan** mana yang dipakai di sini —
`TKT-F2-006` mewarisi pertanyaan yang sama.

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
| 2 | Pencacah nomor: sequence, atau tabel per tahun seperti nomor klaim | Work Owner |
| 3 | **Label** `PROTECTION_TYPE` — nilainya kini diketahui (lihat §9), tetapi artinya hanya terbukti untuk `2`, `7`, `8` | Work Owner |
| 4 | Siapa yang menulis tabel ini selama masa paralel — hanya aplikasi baru, atau Pega juga (`P-1`) | Work Owner |
| 5 | Apakah tabel dibuat di **setiap** basis data portal, atau hanya ASM | Work Owner |

> Butir 3 patut diperhatikan khusus. `T_OPENPROTECTION` memakai **teks** sebagai tipe
> proteksi (`SourceOfBusiness`, `BlackList`), sedangkan alur klaim memakai **kode angka**
> (`2`, `7`, `8`). Kolomnya `VARCHAR2(100)` sehingga kedua bentuk muat — tetapi penyaring
> antrean PREMI membandingkannya dengan `'2'` secara persis, dan bentuk yang berbeda akan
> membuat antrean itu selalu kosong tanpa satu pun galat.

---

## 9. Nilai yang benar-benar dipakai di produksi

Dibaca dari **205 baris OPC** di `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`
(`PXOBJCLASS = 'ASM-FW-GCNMFW-Work-OpenProtection'`) pada 2026-09-23. Hanya kode dan
hitungannya yang dibaca; tidak ada data nasabah yang dicetak (`D-69`).

### `TYPEPROTECTION` — sembilan nilai, bukan delapan

| Nilai | Baris | Arti | Dari mana |
|---|---:|---|---|
| `<NULL>` | 46 | belum diisi | — |
| `7` | 41 | **Perubahan DOL** | `Section/InputProtectionSection-Section.xml:2728` |
| `1` | 32 | belum diketahui | — |
| `8` | 29 | **Perubahan Cause Of Loss** | `:3638` |
| **`9`** | **25** | **belum diketahui — dan TIDAK ADA di export sama sekali** | — |
| `2` | 15 | **PREMI** | `InboxOpenProtection2_RD_collection` menyaring `= "2"` |
| `5` | 9 | belum diketahui | — |
| `4` | 3 | belum diketahui | — |
| `3` | 3 | belum diketahui | — |
| `6` | 2 | belum diketahui | — |

> **Nilai `9` adalah temuan baru.** Seluruh penelusuran export hanya menemukan `1`–`8`;
> `9` tidak muncul di satu pun activity, section, atau when rule. Produksi memakainya pada
> 25 baris — kedua terbanyak setelah `7` di antara nilai yang terisi.
>
> Ini menguatkan alasan layar **menampilkan kode apa adanya** untuk tipe yang labelnya tidak
> diketahui: daftar yang ditebak dari export akan melewatkan `9` sepenuhnya, dan 25 baris
> akan tampil dengan label yang salah atau kosong.

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
| `ID` | `PYID` |
| `CREATE_DATE` | `PXCREATEDATETIME` |
| `CREATED_BY` | `PXCREATEOPNAME` |
| `POLICY_NO` | `POLICYNO` |
| `CLAIM_NO` | `CASEID` |
| `ID_CLAIM` | `PNCCASEID` |
| `PROTECTION_TYPE` | `TYPEPROTECTION` |
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
