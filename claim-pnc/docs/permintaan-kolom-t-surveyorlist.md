# Permintaan penambahan kolom — `POOLDATA.T_SURVEYORLIST`

| | |
|---|---|
| **Tanggal** | 2026-09-29, diperbarui 2026-09-30 |
| **Diminta oleh** | Tim migrasi Claim PNC |
| **Ditujukan ke** | **Tim Pega** |
| **Menempuh** | `D-63` — permintaan tertulis tim pengembang, persetujuan Work Owner, pelaksanaan DBA |
| **Modul terdampak** | My Work / Inbox Survey — Loss Adjuster (`MENU_ID 50`) |
| **Sifat perubahan** | Penambahan kolom, **dan pengisiannya** |

---

## 1. Ringkas — apa yang diminta

Empat kolom ditambahkan ke `POOLDATA.T_SURVEYORLIST`, **dan diisi**.

| # | Kolom | Yang dihidupkan |
|---|---|---|
| 1 | `ADJUSTERACCEPT_1` | tab Outstanding, ALL, dan Invoice |
| 2 | `PYSTATUSWORK` | tab Close, dan penyaring "berkas survei masih terbuka" |
| 3 | `ADJUSTERPIC_1` | kolom "Appointment No" |
| 4 | `REFNO_1` | kolom "Reference No" dan setengah kotak cari |

**Tipe dan panjang mengikuti kolom asalnya** di `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`. Kami tidak
mengusulkan angkanya karena DDL tabel itu belum kami terima (`R-08`) — mohon disalin dari
sumbernya. Seluruhnya **nullable tanpa default**.

> **`POOLDATA.T_CLAIM_SURVEY_DATAPEGA` tidak dipakai** (keputusan Work Owner 2026-09-30). Tabel
> cermin itu sempat menjadi sasaran permintaan ini dan dilepas.

---

## 2. Menambah kolom saja TIDAK cukup

Ini bagian yang paling mudah terlewat, dan tanpa itu **tidak satu pun** dari empat manfaat di
§1 terwujud.

Keempat nilai berubah **sepanjang hidup berkas survei**:

- `ADJUSTERACCEPT_1` berubah saat adjuster menerima penugasan;
- `PYSTATUSWORK` berubah saat berkas ditutup atau dibatalkan — **dan justru nilai akhirnya yang
  dibutuhkan**.

`INSERT_SURVEYORLIST` hanya dipanggil saat hasil survei disimpan, dan `UPDATE`-nya dijaga
`TNOTE IS NOT NULL` (baris 37). Ia **tidak pernah dipanggil** saat status objek kerja berubah.

Jadi yang dibutuhkan adalah **jalur pemutakhiran saat status objek kerja berubah** —
mekanismenya kami serahkan ke Tim Pega.

### Kenapa kolom yang ADA tetapi KOSONG lebih berbahaya daripada kolom yang tidak ada

Tab Outstanding menyaring `ADJUSTERACCEPT_1 IS NULL`. Kolom yang ada tetapi seluruhnya kosong
akan menampilkan **seluruh antrean** sebagai "belum dikonfirmasi adjuster" — layar terisi wajar,
angkanya masuk akal, dan isinya salah.

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

---

## 4. Dua kolom yang SEMPAT diminta dan sudah gugur

Dicatat supaya tidak diminta ulang.

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

```
[catat] Empat kolom belum ada di POOLDATA.T_SURVEYORLIST
        ->  [ok] Keempat kolom SUDAH ADA di POOLDATA.T_SURVEYORLIST

[catat] Keterisian keempat kolom, dari N baris:
          ADJUSTERACCEPT_1 … · ADJUSTERPIC_1 … · REFNO_1 … · PYSTATUSWORK …
[BELUM] ADJUSTERACCEPT_1 ADA tetapi SELURUHNYA kosong
```

Baris `[BELUM]` itu yang menentukan. Selama ia muncul, tabnya tetap dinyatakan belum tersedia.

Bila `-periksa` tidak dapat dijalankan, satu kueri setara:

```sql
SELECT COUNT(*)                  AS baris,
       COUNT(adjusteraccept_1)   AS terisi_accept,
       COUNT(pystatuswork)       AS terisi_status,
       COUNT(adjusterpic_1)      AS terisi_pic,
       COUNT(refno_1)            AS terisi_refno
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
