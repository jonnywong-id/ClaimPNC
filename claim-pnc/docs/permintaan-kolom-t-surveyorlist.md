# Permintaan penambahan kolom — `POOLDATA.T_SURVEYORLIST`

| | |
|---|---|
| **Tanggal** | 2026-09-29, diperbarui 2026-09-30 (kolom tiba, isinya belum) |
| **Diminta oleh** | Tim migrasi Claim PNC |
| **Ditujukan ke** | **Tim Pega** |
| **Menempuh** | `D-63` — permintaan tertulis tim pengembang, persetujuan Work Owner, pelaksanaan DBA |
| **Modul terdampak** | My Work / Inbox Survey — Loss Adjuster (`MENU_ID 50`) |
| **Sifat perubahan** | Penambahan kolom, **dan pengisiannya** |

---

## 1. Ringkas — separuh permintaan ini SUDAH dipenuhi

Per **2026-09-30** tiga dari empat kolom sudah ditambahkan ke `POOLDATA.T_SURVEYORLIST`.
**Seluruhnya masih kosong**, dan karena itu belum satu pun manfaat di bawah terwujud.

| # | Kolom di `T_SURVEYORLIST` | Yang dihidupkan | Ada | Terisi |
|---|---|---|:---:|:---:|
| 1 | `ADJUSTERACCEPT` | tab Outstanding, ALL, dan Invoice | ✅ | ❌ |
| 2 | `PYSTATUSWORK` | tab Close, dan penyaring "berkas survei masih terbuka" | ✅ | ❌ |
| 3 | `REFNO` | kolom "Reference No" dan setengah kotak cari | ✅ | ❌ |
| 4 | `ADJUSTERPIC` | kolom "Appointment No" | ❌ | — |

> **Perhatikan penamaannya.** Di `T_SURVEYORLIST` kolomnya **tanpa akhiran `_1`**. Akhiran itu
> artefak perataan objek kerja di `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`, dan menghilangkannya memang
> lebih bersih — tetapi versi awal dokumen ini memakai nama Pega, sehingga kolom yang **sudah**
> ditambahkan sempat terbaca sebagai belum ada oleh orang yang menambahkannya. Seluruh dokumen
> ini sekarang memakai nama sebenarnya.

**Yang tersisa dari permintaan ini adalah §2 — pengisiannya.** Butir 4 belum tentu diperlukan;
lihat §4.

> **`POOLDATA.T_CLAIM_SURVEY_DATAPEGA` tidak dipakai** (keputusan Work Owner 2026-09-30). Tabel
> cermin itu sempat menjadi sasaran permintaan ini dan dilepas.

---

## 2. Menambah kolom saja TIDAK cukup — dan inilah yang tersisa

Bagian ini ditulis sebagai peringatan pada 2026-09-29; pada 2026-09-30 ia **menjadi kenyataan**.
Ketiga kolom sudah ada, seluruh 17.641 barisnya `NULL`, dan tidak satu pun manfaat di §1 aktif.

Nilainya berubah **sepanjang hidup berkas survei**:

- `ADJUSTERACCEPT` berubah saat adjuster menerima penugasan;
- `PYSTATUSWORK` berubah saat berkas ditutup atau dibatalkan — **dan justru nilai akhirnya yang
  dibutuhkan**.

Yang dibutuhkan ada dua, dan keduanya **di luar `ALTER`**:

1. **Backfill sekali jalan** dari `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` untuk 17.641 baris yang sudah
   ada — kuncinya `T_SURVEYORLIST.CASEID` ke `pzInsKey` objek kerja `Work-SurveyClaim`.
2. **Jalur pemutakhiran** saat status objek kerja berubah, supaya isinya tetap mutakhir.

Tanpa nomor 2, nomor 1 basi dalam hitungan hari.

`INSERT_SURVEYORLIST` hanya dipanggil saat hasil survei disimpan, dan `UPDATE`-nya dijaga
`TNOTE IS NOT NULL` (baris 37). Ia **tidak pernah dipanggil** saat status objek kerja berubah.

Jadi yang dibutuhkan adalah **jalur pemutakhiran saat status objek kerja berubah** —
mekanismenya kami serahkan ke Tim Pega.

### Kenapa kolom yang ADA tetapi KOSONG lebih berbahaya daripada kolom yang tidak ada

Tab Outstanding menyaring `ADJUSTERACCEPT IS NULL`. Kolom yang ada tetapi seluruhnya kosong
akan menampilkan **seluruh 2.448 berkas** sebagai "belum dikonfirmasi adjuster" — layar terisi
wajar, angkanya masuk akal, dan isinya salah. Tab ALL dan Invoice sebaliknya: kosong sama
sekali, yang terbaca sebagai "tidak ada pekerjaan".

Keduanya cacat diam. Tidak ada yang akan melaporkannya sebagai kerusakan.

Karena itu `-periksa` memisahkan keduanya, dan modul ini **tidak akan menghidupkan tabnya**
sebelum keterisiannya terukur.

---

## 3. Kenapa keempatnya, dan dari mana buktinya

Keempat kolom milik objek kerja `ASM-FW-GCNMFW-Work-SurveyClaim`. Ketiga rule yang tersisa
membacanya dari objek yang sama, tanpa join:

| Rule | Bukti |
|---|---|
| `RDB List/CountOSLostAdjuster-SQL.xml` | `FROM PC_ASM_FW_GCNMFW_WORK A WHERE a.pxobjclass = 'ASM-FW-GCNMFW-Work-SurveyClaim'` |
| `RDB List/BrowseLossAdjuster-SQL.xml` | `A.REFNO_1`, `A.AdjusterPIC_1`, `A.PYSTATUSWORK` dari alias `A` |
| `RDB List/BrowseInternalSurveyor-SQL.xml` | idem |

Ketujuh tab layar lama seluruhnya terbentuk darinya:

```sql
tab Outstanding : PYSTATUSWORK NOT IN ('Resolved-Completed','Resolved-Rejected')
                  AND AdjusterAccept_1 IS NULL
tab ALL         : PYSTATUSWORK NOT IN (…) AND AdjusterAccept_1 = '1'
tab Invoice     : PYSTATUSWORK NOT IN (…) AND AdjusterAccept_1 = '1'
                  AND AdjusterStatus_1 = 'Invoice Fee'
tab Close       : PYSTATUSWORK = 'Resolved-Completed'
3 tab komunikasi: PYSTATUSWORK NOT IN (…) + status pesan di M_KOMUNIKASI_PNC
```

Perhatikan lawan tab Outstanding adalah **`IS NULL`**, bukan `<> '1'`. Keduanya berbeda pada
baris bernilai `'0'`, dan perilaku Pega dibawa apa adanya (`P-5`).

> Nama ber-`_1` pada blok di atas dikutip **apa adanya dari rule Pega**, dan hanya berlaku di
> `PC_ASM_FW_GCNMFW_WORK`. Padanannya di `T_SURVEYORLIST` ada di tabel §1 — tanpa akhiran itu.

---

## 4. Kolom yang SEMPAT diminta dan sudah gugur

Dicatat supaya tidak diminta ulang.

### `ADJUSTERPIC` — kemungkinan besar tidak diperlukan

Judul layar **"Appointment No"** adalah kolom paling kiri grid My Work, dan ia mengikat
`.City`, yang `BrowseLossAdjuster-SQL.xml` isi dari `AdjusterPIC_1`.

`ExportDataDetailKlaim-SQL.xml` membuktikan kolom itu berisi **nama adjuster**, bukan nomor
penugasan:

```sql
CASE WHEN C.SURVEYORTYPE_1 IN ('2','3','4') THEN C.Adjusterpic_1 ELSE '' END
CASE WHEN C.SURVEYORTYPE_1 = '1'            THEN c.SURVEYORNAME_1 …
```

Keduanya konsep yang sama, dipisah menurut jenis surveyor — eksternal versus internal. Dan
`Database/INSERT_SURVEYORLIST.prc` hanya punya **satu** kolom nama, `SURVEYOR_NAME`, diisi
`TempSurvey.SurveyorName` tanpa memandang jenisnya.

Bila itu benar, kolom "Appointment No" menggandakan "PIC Loss Adjuster" di sebelahnya, dan
`SURVEYOR_NAME` sudah membawanya. **Satu kueri memastikannya sebelum butir 4 dicoret:**

```sql
SELECT surveytype, COUNT(*) AS baris, COUNT(surveyor_name) AS terisi
  FROM POOLDATA.T_SURVEYORLIST GROUP BY surveytype ORDER BY surveytype;
```

Bila `terisi` penuh di semua `surveytype` — termasuk `1` dan `2` — satu kolom itu cukup.

**Keputusan Work Owner yang menyertainya:** kolom "Appointment No" di layar baru dihapus, atau
dipertahankan menampilkan nama adjuster seperti Pega? Menghapusnya menyimpang dari `D-13`;
mempertahankannya berarti dua kolom bersebelahan berisi nama yang sama.

### `ADJUSTERSTATUS_1` — `STS_SURVEY` sudah membawanya

Sebaran `POOLDATA.T_SURVEYORLIST.STS_SURVEY` di produksi memuat ketiga nilai yang dipakai Pega
sebagai penyaring, dengan jumlah yang nyata:

```
Final Report  1.466      Invoice Fee  1.069      Close Case  316      Reject Case  11
```

berdampingan dengan seluruh tahapan hidup survei — 22 nilai berbeda. Kolom "Status ASM"
karena itu **sudah terisi** tanpa menunggu siapa pun.

### `SURVEYORNAME_1` — `SURVEYOR_NAME` sudah membawanya

Kolom layar "PIC Loss Adjuster" mengikat `.AnaylstRemarks`, dan
`BrowseOSLossAdjusterPIC-SQL.xml` mengisinya dari `SURVEYORNAME_1`. Jadi keduanya satu kolom,
dan padanannya `T_SURVEYORLIST.SURVEYOR_NAME` sudah dibaca.

Yang menyesatkan: kolom itu dialiaskan **tiga nama berbeda** di tiga rule — `AnaylstRemarks`,
`CountryID`, dan `ComplianceRemark` — sehingga terbaca seolah tiga kolom berbeda.

---

## 5. Kenapa `STS_SURVEY` tidak dapat menggantikan `PYSTATUSWORK`

Diuji langsung, dan **gagal**. Dari **1.070** berkas survei yang sudah `Resolved-*` di Pega,
hanya **308 (28,8%)** berakhir di `Close Case`/`Reject Case`:

```
Resolved-Completed ->  Invoice Fee 497 · Close Case 301 · Final Report 95 · On Progress 45 · …
Resolved-Rejected  ->  On Progress 42 · (kosong) 22 · Reject Case  **0 dari 70**
```

`Invoice Fee` adalah langkah terakhir pekerjaan adjuster — ia menagih, lalu berkasnya ditutup
petugas ASM. Adjuster tidak pernah mencatat "Close Case" sendiri.

Memakai `STS_SURVEY = 'Close Case'` sebagai pengganti tab Close akan menampilkan **307 dari
1.000** berkas — kehilangan 69%. Itu bukan selisih terencana melainkan tab yang rusak.

Arah sebaliknya aman: hanya **6 berkas** ber-`Close Case` yang belum tutup di Pega.

---

## 6. Catatan tentang bentuk tabel, bukan keberatan

`T_SURVEYORLIST` adalah **jejak perkembangan** — satu baris per perubahan status. Diukur di
produksi 2026-09-29:

```
2.448 berkas survei  ->  17.641 baris     rata-rata 7,21 langkah per berkas
69,1% berkas punya >1 baris; TERBERAT satu berkas = 176 baris
```

Dua akibat yang perlu Tim Pega sadari saat menulis jalur pemutakhirannya:

1. **Nilai per berkas akan terduplikasi** rata-rata 7×, dan satu perubahan status harus
   memutakhirkan sampai 176 baris serentak — atau cukup baris terakhirnya saja, bila itu yang
   dipilih. Modul ini **hanya membaca langkah terakhir**, sehingga keduanya bekerja.
2. **Tabel ini memberi makan KPI adjuster** — dibaca `GetReportKPI_Survey`,
   `GetReportKPIAdjuster_Progress`, dan `GetReportKPIAdjuster_FinalReport`, tiga dari sembilan
   angka KPI. **Menambah kolom aman**; yang perlu dihindari adalah **menyisipkan baris baru**
   hanya untuk mencatat perubahan status, karena itu mengubah nilai kinerja orang.

---

## 7. Cara memastikan permintaan ini sudah tiba

```
claimpnc -periksa
```

Dua tahap, **terpisah dengan sengaja** — kolom ditambahkan DBA lewat `ALTER`, isinya ditulis
procedure:

Keluarannya per 2026-09-30:

```
[catat] Keempat kolom yang ditunggu di POOLDATA.T_SURVEYORLIST:
          ADJUSTERACCEPT ADA · ADJUSTERPIC belum · REFNO ADA · PYSTATUSWORK ADA

[catat] Keterisian, dari 17.641 baris: ADJUSTERACCEPT 0 · REFNO 0 · PYSTATUSWORK 0
[BELUM] PYSTATUSWORK ADA tetapi SELURUHNYA kosong
[BELUM] ADJUSTERACCEPT ADA tetapi SELURUHNYA kosong
[BELUM] REFNO ADA tetapi SELURUHNYA kosong
```

Baris `[BELUM]` itu yang menentukan, **bukan baris pertama**. Selama ia muncul, tabnya tetap
dinyatakan belum tersedia — kolom yang ada tetapi kosong tidak lebih siap daripada kolom yang
tidak ada.

Bila `-periksa` tidak dapat dijalankan, satu kueri setara:

```sql
SELECT COUNT(*)                AS baris,
       COUNT(adjusteraccept)   AS terisi_accept,
       COUNT(pystatuswork)     AS terisi_status,
       COUNT(refno)            AS terisi_refno
  FROM POOLDATA.T_SURVEYORLIST;
```

Pembandingnya: **17.641** baris, **2.448** berkas survei.

---

## 8. Yang masih terbuka, dan bukan bagian permintaan ini

**`LOSSTYPE` untuk kolom "Cause Of Loss"** — belum dikonfirmasi DBA. Kueri Pega yang mengisinya
hilang dari export (`R-16`), dan `LOSSTYPE` adalah satu-satunya kolom pada tabel penggerak yang
menyatakan jenis kerugian.

---

## 9. Batas pernyataan ini

Dua hal yang **tidak** kami klaim pasti:

1. **`R-01`** — 2 dari 64 objek database belum diterima, ditambah **12 dependensi** yang
   dipanggil 62 procedure yang sudah ada. Penulis di luar export mungkin ada.

2. **`R-16`** — ±242 rule hilang dari export, 137 di antaranya When rule. Kesimpulan bahwa
   `STS_SURVEY` membawa domain `ADJUSTERSTATUS_1` dibangun dari **sebaran nilainya di
   produksi**, bukan dari rule penulisnya. Kesamaannya **belum diuji baris per baris**
   terhadap `PC_ASM_FW_GCNMFW_WORK.ADJUSTERSTATUS_1`.
