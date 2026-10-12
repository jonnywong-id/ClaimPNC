# Pengisian kolom `POOLDATA.T_CLAIMLIST_ADMIN` yang tertinggal

| | |
|---|---|
| **Tanggal** | 2026-10-11 |
| **Disusun oleh** | Tim migrasi Claim PNC |
| **Ditujukan ke** | **DBA** (pelaksana) · **Work Owner** |
| **Menempuh** | `D-63` — permintaan tertulis, persetujuan Work Owner, pelaksanaan DBA |
| **Modul terdampak** | **My Inbox** (`MENU_ID 51`) dan modul lain yang membaca tabel yang sama |
| **Lingkungan angka** | **DEV** (service `DEV_PEGA83G`) — **bukan produksi** |
| **Sifat** | **SEMENTARA** — lihat §5 |
| **Pendahulu** | [`backfill-t-claimlist-admin.md`](backfill-t-claimlist-admin.md) — premisnya sudah gugur, lihat §0 |

---

## 0. Kenapa berkas ini ada, dan apa yang gugur

Berkas pendahulu menyiapkan `MERGE` **sisipan** untuk 1.770 klaim yang belum dipindahkan. Antara
7 dan 11 Oktober keadaan berubah sendiri:

| | 7 Okt | **11 Okt** |
|---|---:|---:|
| Klaim PNC di Pega | 2.647 | 2.648 |
| Baris `T_CLAIMLIST_ADMIN` | 1.049 | **2.890** |
| **Belum dipindahkan** | 1.770 | **0** |

Jadi sisipan itu **tidak akan menyisipkan apa pun lagi**. Yang tersisa adalah `UPDATE`, dan
itu persis cabang yang pendahulunya tahan.

**Ada yang mengisi tabel itu, dan bukan skrip kita** — buktinya kolom mana yang diisinya dan mana
yang tidak. Lihat §5.

---

## 1. Apa yang sebenarnya kosong

Sensus seluruh kolom terhadap **2.747 baris kelas `Work-PNC`** (11 Okt, DEV):

| Kelompok | Kolom | Keterangan |
|---|---:|---|
| **Kosong seluruhnya, sumber senama ADA di Pega** | **11** | sasaran berkas ini |
| Sebagian terisi, sumber senama ada | 26 | **tidak perlu disentuh** — §1.2 |
| Kosong seluruhnya, tanpa sumber senama | 3 | ada di `PC_ASSIGN_WORKLIST` — §4.2 |
| Sebagian terisi, tanpa sumber senama | 17 | turunan atau milik modul lain |

### 1.1 Kesebelas kolom kosong — dan apakah Pega benar punya isinya

| Kolom | Punya nilai di Pega | Dipakai |
|---|---:|---|
| `DOKUMENLENGKAP_1` | **928** | **My Inbox** — tab Complete / Not complete |
| `TANGGALDOKLENGKAP` | 376 | modul lain |
| `REFNO_1` | 370 | modul lain |
| `STATUSKOMUNIKASI_1` | 268 | modul lain |
| `REQUESTSURVEY_1` | 46 | modul lain |
| `ISPENDINGCLOSE` | **33** | **My Inbox** — tab Temporary Close, penyaring unduhan |
| `TKA_1` | **15** | **My Inbox** — tab TKA, dan pengecualian TKA dari dua tab lain |
| `SURVEYORNAMEMARINE_1` | 10 | modul lain |
| `ADJUSTERSTATUS_1` | **0** | — |
| `KURIR` | **0** | — |
| `NOTREGISTNOTE_1` | **0** | — |

**Tiga yang terakhir tidak diisi skrip ini.** Pega sendiri tidak punya satu nilai pun di sana,
sehingga mengisinya adalah pekerjaan nol yang tetap menyentuh 2.649 baris — risiko tanpa imbalan.

### 1.2 Kenapa 26 kolom "sebagian terisi" **tidak** ikut diisi

Terlihat bolong, ternyata tidak. Diuji per kolom: berapa baris yang **kosong di tujuan padahal
Pega punya nilainya**.

| Kolom | Benar-benar hilang |
|---|---:|
| `SURVEYORTYPE_1` | **0** |
| `USERTEKNIS_1` | **0** |
| `BRANCHNAME` | **0** |
| `PNCCASEID` | **30** |

Jadi kolom-kolom itu tampak sebagian hanya karena **Pega sendiri sebagian**. Satu-satunya celah
nyata adalah `PNCCASEID`, ditangani terpisah di §4.1.

Nilainya pun sudah sepakat: `PYSTATUSWORK` **0 berbeda**, `BRANCHNAME` 1, `USERTEKNIS_1` 3.

### 1.3 Dua kolom tanggal bertipe beda — **tidak** disentuh

`REGISTERDATE_1` dan `REPORTDATE_1` bertipe `DATE` di tujuan tetapi **`VARCHAR2` di Pega**. Itu
jebakan nyata, dan formatnya dibuktikan dengan mencocokkan ke baris yang sudah terisi — bukan
ditebak:

| Kolom | Bentuk di Pega | `TO_DATE(SUBSTR(x,1,8),'YYYYMMDD')` |
|---|---|---|
| `REGISTERDATE_1` | 8 karakter, seluruhnya angka | **1.727 cocok · 0 meleset** |
| `REPORTDATE_1` | 23 karakter | **2.248 cocok · 0 meleset** |

Formatnya terbukti, tetapi **keduanya sudah terisi sesuai Pega** — jadi tidak ada yang perlu
dikerjakan. Rumusnya dicatat supaya tidak perlu diselidiki ulang bila kelak dibutuhkan.

Satu lagi yang perlu diketahui: **tidak ada penggeseran zona waktu.** `PXCREATEDATETIME` tujuan dan
Pega berselisih **0 jam pada 2.784 baris**, dan −7 jam pada **1 baris**. Penyalinan apa adanya
benar, dan `R-12` tidak berlaku di jalur ini.

---

## 2. Skripnya

### 2.1 Periksa SEBELUM

```sql
SELECT 'Work-PNC di tujuan' AS hal, COUNT(*) AS jml
  FROM POOLDATA.T_CLAIMLIST_ADMIN WHERE PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
UNION ALL
SELECT 'punya padanan di Pega (AKAN tersentuh)', COUNT(*)
  FROM POOLDATA.T_CLAIMLIST_ADMIN t
 WHERE t.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND EXISTS (SELECT 1 FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w WHERE w.PZINSKEY = t.PZINSKEY)
UNION ALL
SELECT 'YATIM — tidak akan tersentuh', COUNT(*)
  FROM POOLDATA.T_CLAIMLIST_ADMIN t
 WHERE t.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND NOT EXISTS (SELECT 1 FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w WHERE w.PZINSKEY = t.PZINSKEY);
```

Di DEV 11 Okt: **2.747 · 2.649 · 98**.

Lalu buktikan kedelapan kolom memang masih kosong — inilah yang membuat rollback §2.3 lossless:

```sql
SELECT COUNT(DOKUMENLENGKAP_1)     AS dok,
       COUNT(ISPENDINGCLOSE)       AS pending,
       COUNT(TKA_1)                AS tka,
       COUNT(TANGGALDOKLENGKAP)    AS tgl_dok,
       COUNT(REFNO_1)              AS refno,
       COUNT(STATUSKOMUNIKASI_1)   AS komunikasi,
       COUNT(REQUESTSURVEY_1)      AS req_survey,
       COUNT(SURVEYORNAMEMARINE_1) AS marine
  FROM POOLDATA.T_CLAIMLIST_ADMIN
 WHERE PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC';
```

**Seluruhnya wajib `0`.** Bila ada yang bukan nol, **hentikan** — berarti sudah ada yang
mengisinya, dan rollback §2.3 akan ikut menghapus pekerjaan itu.

### 2.2 Pengisian

```sql
MERGE INTO POOLDATA.T_CLAIMLIST_ADMIN t
USING (
    SELECT w.PZINSKEY,
           w.DOKUMENLENGKAP_1,
           w.ISPENDINGCLOSE,
           w.TKA_1,
           w.TANGGALDOKLENGKAP,
           w.REFNO_1,
           w.STATUSKOMUNIKASI_1,
           w.REQUESTSURVEY_1,
           w.SURVEYORNAMEMARINE_1
      FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
     WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
) s
ON (t.PZINSKEY = s.PZINSKEY)
WHEN MATCHED THEN UPDATE SET
     t.DOKUMENLENGKAP_1     = s.DOKUMENLENGKAP_1,
     t.ISPENDINGCLOSE       = s.ISPENDINGCLOSE,
     t.TKA_1                = s.TKA_1,
     t.TANGGALDOKLENGKAP    = s.TANGGALDOKLENGKAP,
     t.REFNO_1              = s.REFNO_1,
     t.STATUSKOMUNIKASI_1   = s.STATUSKOMUNIKASI_1,
     t.REQUESTSURVEY_1      = s.REQUESTSURVEY_1,
     t.SURVEYORNAMEMARINE_1 = s.SURVEYORNAMEMARINE_1;

COMMIT;
```

Tidak ada `WHEN NOT MATCHED` — skrip ini **tidak menyisipkan apa pun**. Dan 98 baris yatim tidak
tersentuh karena tidak berpasangan.

Kedelapan kolom **bertipe sama di kedua sisi** (`VARCHAR2`→`VARCHAR2`, `TIMESTAMP(6)`→`TIMESTAMP(6)`),
jadi tidak ada konversi — tidak ada tempat untuk salah format.

### 2.3 Periksa SESUDAH

```sql
-- 1. Tab status dokumen akhirnya terbagi.  Harapan: 589 bernilai '0', 339 bernilai '1'
SELECT NVL(DOKUMENLENGKAP_1,'(kosong)') AS nilai, COUNT(*) AS jml
  FROM POOLDATA.T_CLAIMLIST_ADMIN
 WHERE PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
 GROUP BY NVL(DOKUMENLENGKAP_1,'(kosong)') ORDER BY 2 DESC;

-- 2. Tab Temporary Close punya populasi.  Harapan: 10 baris Resolved-Completed + true
SELECT PYSTATUSWORK, ISPENDINGCLOSE, COUNT(*) AS jml
  FROM POOLDATA.T_CLAIMLIST_ADMIN
 WHERE PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC' AND ISPENDINGCLOSE IS NOT NULL
 GROUP BY PYSTATUSWORK, ISPENDINGCLOSE ORDER BY 3 DESC;

-- 3. TKA.  Harapan: 14 bernilai '1'
SELECT NVL(TKA_1,'(kosong)') AS nilai, COUNT(*) AS jml
  FROM POOLDATA.T_CLAIMLIST_ADMIN
 WHERE PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
 GROUP BY NVL(TKA_1,'(kosong)') ORDER BY 2 DESC;

-- 4. Tidak ada nilai yang menyimpang dari Pega.  Harapan: 0
SELECT COUNT(*) AS tidak_cocok
  FROM POOLDATA.T_CLAIMLIST_ADMIN t
  JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK w ON w.PZINSKEY = t.PZINSKEY
 WHERE t.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND (NVL(t.DOKUMENLENGKAP_1,'~') <> NVL(w.DOKUMENLENGKAP_1,'~')
        OR NVL(t.ISPENDINGCLOSE,'~') <> NVL(w.ISPENDINGCLOSE,'~')
        OR NVL(t.TKA_1,'~')          <> NVL(w.TKA_1,'~'));
```

Kueri 4 memakai `NVL(...,'~')` dengan sengaja: tanpa itu, dua `NULL` dianggap **tidak** sama oleh
Oracle, dan seluruh baris kosong akan terbaca sebagai menyimpang.

### 2.4 Rollback

Lossless, dan **hanya karena** §2.1 membuktikan kedelapan kolom benar-benar kosong sebelumnya.
Tidak ada nilai lama yang hilang karena memang tidak ada nilai lama.

```sql
UPDATE POOLDATA.T_CLAIMLIST_ADMIN
   SET DOKUMENLENGKAP_1 = NULL, ISPENDINGCLOSE = NULL, TKA_1 = NULL,
       TANGGALDOKLENGKAP = NULL, REFNO_1 = NULL, STATUSKOMUNIKASI_1 = NULL,
       REQUESTSURVEY_1 = NULL, SURVEYORNAMEMARINE_1 = NULL
 WHERE PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC';
COMMIT;
```

> **Rollback ini punya masa berlaku.** Bila pengisi tabel yang tidak dikenal itu kemudian mulai
> mengisi salah satu kolom di atas, menjalankan rollback akan **menghapus pekerjaannya juga**.
> Jalankan §2.1 ulang sebelum rollback, dan batalkan bila angkanya sudah tidak nol.

---

## 3. Yang berubah di layar setelah ini dijalankan

| Tab | Sebelum | Sesudah |
|---|---|---|
| **Complete documents** | selalu `0` | terbagi sesuai Pega |
| **Documents not complete** | memuat **seluruh** klaim | hanya yang benar belum lengkap |
| **TKA** | mati | dapat dihidupkan — 14 klaim |
| **Temporary Close** | mati | dapat dihidupkan — populasinya kecil, 10 klaim |

Dua baris pertama perlu digarisbawahi: angka pada tab "Documents not complete" **akan turun**, dan
itu **bukan data yang hilang** — melainkan klaim yang selama ini salah digolongkan karena kolomnya
kosong, dan kueri memperlakukan kosong sebagai belum lengkap (mengikuti Pega).

---

## 4. Tambahan opsional

### 4.1 `PNCCASEID` — 30 baris yang benar-benar hilang

```sql
-- Simpan dulu daftarnya, supaya rollback-nya tepat sasaran.
CREATE TABLE POOLDATA.TMP_PNCCASEID_20261011 AS
SELECT t.PZINSKEY
  FROM POOLDATA.T_CLAIMLIST_ADMIN t
  JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK w ON w.PZINSKEY = t.PZINSKEY
 WHERE t.PNCCASEID IS NULL AND w.PNCCASEID IS NOT NULL;

MERGE INTO POOLDATA.T_CLAIMLIST_ADMIN t
USING (SELECT PZINSKEY, PNCCASEID
         FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK
        WHERE PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC' AND PNCCASEID IS NOT NULL) s
ON (t.PZINSKEY = s.PZINSKEY)
WHEN MATCHED THEN UPDATE SET t.PNCCASEID = s.PNCCASEID
 WHERE t.PNCCASEID IS NULL;
COMMIT;
```

Penyaring `WHERE t.PNCCASEID IS NULL` pada cabang `UPDATE` **disengaja**: ia membuat skrip hanya
mengisi yang kosong, tidak menimpa 1.751 baris yang sudah benar.

Rollback:

```sql
UPDATE POOLDATA.T_CLAIMLIST_ADMIN SET PNCCASEID = NULL
 WHERE PZINSKEY IN (SELECT PZINSKEY FROM POOLDATA.TMP_PNCCASEID_20261011);
COMMIT;
```

### 4.2 Tiga kolom penugasan — sumbernya `PC_ASSIGN_WORKLIST`

`PXDEADLINETIME`, `PXGOALTIME`, dan `PYASSIGNMENTSTATUS` kosong seluruhnya dan **tidak ada**
padanannya di `PC_ASM_FW_GCNMFW_WORK` — tetapi **ketiganya ada di `PC_ASSIGN_WORKLIST`**, sekelompok
dengan `PXASSIGNEDOPERATORID`, `PXTASKLABEL`, dan `PXFLOWNAME` yang memang sudah terisi dari sana.

**Tidak disertakan** di §2.2 karena ia menuntut keputusan yang sama dengan berkas pendahulu §4:
**assignment mana** yang dipakai bila satu klaim punya lebih dari satu. My Inbox tidak membaca
ketiganya, jadi pengisiannya bukan penghalang apa pun hari ini.

---

## 5. Kenapa ini disebut SEMENTARA

**Ada pengisi lain yang aktif, dan ia tidak mengisi kedua kolom terpenting.** Terbukti dari kolom
mana yang terisi pada 2.890 baris:

| Kolom | Terisi |
|---|---:|
| `BUSINESSGROUPID` | 2.474 |
| `PXASSIGNEDOPERATORID` | 2.222 |
| `AGING` | 827 — **tidak bertambah sejak 7 Okt** |
| `STATUSPROGRESS1` | 804 |
| `SURVEYORTYPE_1` | 674 |
| **`DOKUMENLENGKAP_1`** | **0** |
| **`ISPENDINGCLOSE`** | **0** |

Pengisi itu menyalin banyak kolom tetapi **tidak pernah** menyentuh dua ini — padahal Pega punya
928 dan 33 nilai di sana. Akibatnya:

1. Pengisian ini **dapat ditimpa** kapan saja oleh pengisi tersebut.
2. **Klaim baru akan tetap datang tanpa kedua kolom itu**, sehingga tabnya akan meleset lagi
   secara bertahap.

Jadi skrip ini menyembuhkan keadaan hari ini, **bukan menyelesaikan sebabnya**. Yang menyelesaikan
sebabnya adalah menjawab pertanyaan berikut.

---

## 6. Yang tetap tidak terjawab

| # | Pertanyaan | Pemilik |
|---|---|---|
| 1 | **Siapa yang mengisi `T_CLAIMLIST_ADMIN`, dan kenapa ia melewatkan `DOKUMENLENGKAP_1` dan `ISPENDINGCLOSE`?** Kini terbukti ada dan aktif — 1.841 baris masuk antara 7 dan 11 Oktober | **DBA + Tim Pega** |
| 2 | Setelah dikenali, apakah pengisi itu diperbaiki, atau pengisian ini dijadwalkan berulang? | **Work Owner** |
| 3 | **`AGING`** — masih 827 terisi, rumusnya masih tidak diketahui | **DBA** |
| 4 | **98 baris yatim** — ada di tabel tujuan tanpa padanan di Pega. Dari mana asalnya? | **DBA** |

Butir 4 kemungkinan besar menjawab butir 1: baris yang tidak berasal dari objek kerja Pega
menunjukkan pengisinya membaca sumber lain.

---

## 7. Jejak bukti

Seluruh angka dibaca langsung dari katalog Oracle **lingkungan DEV portal ASM** (service
`DEV_PEGA83G`) pada 2026-10-11, memakai perkakas sementara yang **sudah dihapus**.

Tidak ada satu pun nilai data nasabah yang disalin ke berkas ini (`D-69`) — yang tertulis hanyalah
jumlah, nama kolom, dan nama objek database. Format tanggal pada §1.3 diuji dengan membandingkan
**hasil konversi**, bukan dengan menampilkan nilainya.
