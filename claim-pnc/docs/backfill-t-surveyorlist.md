# Pengisian kolom `POOLDATA.T_SURVEYORLIST` dari objek kerja Pega

| | |
|---|---|
| **Tanggal** | 2026-10-07 |
| **Disusun oleh** | Tim migrasi Claim PNC |
| **Ditujukan ke** | **DBA** (pelaksana) · **Tim Pega** (jalur pemutakhiran) |
| **Menempuh** | `D-63` — permintaan tertulis, persetujuan Work Owner, pelaksanaan DBA |
| **Modul terdampak** | My Work / Inbox Survey — Loss Adjuster (`MENU_ID 50`) |
| **Prasyarat** | kelima kolom sudah ada — lihat `permintaan-kolom-t-surveyorlist.md` |

---

## 0a. DUA LINGKUNGAN, DAN ANGKANYA BERBEDA 63×

Kelima kolom **sudah ada di `pega_dev83`**, dan **belum ada di produksi** (keadaan 2026-10-07).
Backfill karena itu dijalankan **dev lebih dulu**, produksi menyusul setelah `ALTER`-nya tiba.

| | `pega_dev83` | Produksi |
|---|---:|---:|
| Baris `T_SURVEYORLIST` | **279** | **17.641** |
| Berkas survei | **188** | **2.448** |
| Langkah per berkas | **1,48** | **7,21** |
| Baris yatim | **1** | **4** |
| Kelima kolom ada? | **ya** | **belum** |

**Setiap angka di berkas ini ditandai lingkungannya.** Angka tanpa penanda berasal dari
produksi, karena di sanalah analisis awal dikerjakan.

### Apa yang dev TIDAK dapat buktikan

Penggandaan baris di dev **1,48×**, di produksi **7,21×**. Penyaring langkah terakhir memang
teruji di dev — 279 baris untuk 188 berkas, selisihnya nyata — tetapi **1,48× jauh lebih mudah
lolos dari mata** daripada 7,21×. Layar yang menampilkan 279 alih-alih 188 masih terlihat masuk
akal; 17.641 alih-alih 2.448 tidak.

Dev membuktikan penyaringnya **jalan**. Ia tidak membuktikan ia **benar pada kasus berat** —
berkas terburuk di produksi punya **176 langkah**, dan di dev tidak ada pembandingnya.

### Yang harus diulang saat giliran produksi

Seluruh §2 dijalankan **ulang di produksi** untuk memperoleh angka dasarnya sendiri. Angka dev
TIDAK boleh dipakai sebagai pembanding di sana — keduanya beda populasi, beda rasio, beda jumlah
baris yatim.

---

## 0. Baca ini lebih dulu

**Berkas ini BUKAN migrasi aplikasi.** Ia tidak ditaruh di `backend/migrations/` dan tidak pernah
dijalankan aplikasi Go, karena `POOLDATA.T_SURVEYORLIST` **dimiliki Pega** selama masa paralel
(`P-1`: satu tabel hanya boleh ditulis satu sistem). Modul Go hanya membacanya — dikunci
`TestSeluruhKueriHanyaMembaca`.

Yang dikerjakan di sini adalah **operasi sekali jalan oleh DBA**, ditambah usulan jalur
pemutakhiran yang mekanismenya diserahkan ke Tim Pega.

---

## 1. Pemetaan — lima kolom

Seluruh pasangan di bawah terbaca dari keempat kueri tab layar lama
(`RDB List/Browse*LostAdjuster-SQL.xml`), bukan dari dugaan.

| `T_SURVEYORLIST` | ← | `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` | Menghidupkan |
|---|---|---|---|
| `ADJUSTERACCEPT` | ← | `AdjusterAccept_1` | tab Outstanding, ALL, Invoice |
| `PYSTATUSWORK` | ← | `PYSTATUSWORK` | tab Close, penyaring berkas tutup |
| `REFNO` | ← | `REFNO_1` | kolom Reference No, setengah kotak cari |
| `ADJUSTER_PIC` | ← | `ADJUSTERPIC_1` | kolom PIC Loss Adjuster |
| `RESCHEDULE_LOCATION` | ← | `RescheduleLocation_1` | kolom Location |

**Perhatikan `PYSTATUSWORK` tidak berakhiran `_1`** — ia kolom bawaan Pega, bukan hasil perataan
objek kerja. Empat lainnya berakhiran `_1`.

### Kunci sambungnya

```
T_SURVEYORLIST.CASEID  =  PC_ASM_FW_GCNMFW_WORK.PZINSKEY
```

dengan `PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim'`.

| Lingkungan | Baris | Berpasangan | **Yatim** |
|---|---:|---:|---:|
| Produksi (2026-10-03) | 17.641 | 17.637 | **4** |
| `pega_dev83` (2026-10-07) | 279 | 278 | **1** |

Baris yatim tetap `NULL` setelah backfill, dan itu benar: tidak ada sumber untuk diisi.

### Kolom sumbernya memang terisi — diukur di `pega_dev83`

Ini pengukuran pertama yang membuktikannya, dan menentukan apakah backfill berguna sama sekali:

```
PYSTATUSWORK          278 / 278   100%
RESCHEDULELOCATION_1  278 / 278   100%
ADJUSTERACCEPT_1      259 / 278    93%
ADJUSTERPIC_1         151 / 278    54%
REFNO_1               130 / 278    47%
```

`PYSTATUSWORK` 100% yang paling menentukan: tab Close dan penyaring berkas tutup akan berfungsi
penuh, bukan terisi sebagian.

---

## 2. Periksa DULU — sebelum menulis apa pun

Jalankan ketiganya dan simpan hasilnya. Tanpa angka sebelum, tidak ada cara membuktikan
backfill-nya benar.

```sql
-- 2a. Keadaan awal: berapa yang sudah terisi (harusnya nol di semua kolom)
SELECT COUNT(*)                     AS baris,
       COUNT(s.ADJUSTERACCEPT)      AS accept,
       COUNT(s.PYSTATUSWORK)        AS status_work,
       COUNT(s.REFNO)               AS refno,
       COUNT(s.ADJUSTER_PIC)        AS adjuster_pic,
       COUNT(s.RESCHEDULE_LOCATION) AS reschedule_location
  FROM POOLDATA.T_SURVEYORLIST s;

-- 2b. Berapa yang AKAN terisi — pembanding untuk sesudahnya
SELECT COUNT(*)                        AS baris_berpasangan,
       COUNT(w.ADJUSTERACCEPT_1)       AS accept,
       COUNT(w.PYSTATUSWORK)           AS status_work,
       COUNT(w.REFNO_1)                AS refno,
       COUNT(w.ADJUSTERPIC_1)          AS adjuster_pic,
       COUNT(w.RESCHEDULELOCATION_1)   AS reschedule_location
  FROM POOLDATA.T_SURVEYORLIST s
  JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
    ON w.PZINSKEY = s.CASEID
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim';

-- 2c. Baris yatim — tidak akan terisi, dan itu benar
SELECT COUNT(*) AS baris_tanpa_objek_kerja
  FROM POOLDATA.T_SURVEYORLIST s
 WHERE NOT EXISTS (SELECT 1
                     FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
                    WHERE w.PZINSKEY = s.CASEID
                      AND w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim');
```

---

## 3. Backfill — `MERGE`, bukan `UPDATE` bersubquery

```sql
MERGE INTO POOLDATA.T_SURVEYORLIST s
USING (SELECT w.PZINSKEY,
              w.ADJUSTERACCEPT_1,
              w.PYSTATUSWORK,
              w.REFNO_1,
              w.ADJUSTERPIC_1,
              w.RESCHEDULELOCATION_1
         FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
        WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim') w
   ON (s.CASEID = w.PZINSKEY)
 WHEN MATCHED THEN UPDATE SET
       s.ADJUSTERACCEPT      = w.ADJUSTERACCEPT_1,
       s.PYSTATUSWORK        = w.PYSTATUSWORK,
       s.REFNO               = w.REFNO_1,
       s.ADJUSTER_PIC        = w.ADJUSTERPIC_1,
       s.RESCHEDULE_LOCATION = w.RESCHEDULELOCATION_1;

COMMIT;
```

### Kenapa `MERGE`

`UPDATE … SET kolom = (SELECT …)` menjalankan **lima subquery berkorelasi per baris** — 88.000
pembacaan untuk 17.641 baris. `MERGE` membacanya sekali sebagai satu himpunan.

Dan `UPDATE` bersubquery punya jebakan yang lebih berbahaya daripada lambatnya: pada baris yatim
subquery mengembalikan `NULL`, sehingga kolom yang sudah terisi akan **dikosongkan**. `MERGE`
hanya menyentuh baris yang berpasangan.

### Sifatnya

**Idempoten.** Dijalankan dua kali menghasilkan keadaan yang sama — tidak ada baris disisipkan,
tidak ada pencacah dinaikkan. Aman diulang bila terputus.

### Catatan volume

Satu objek kerja memasok **rata-rata 7,21 baris** `T_SURVEYORLIST`, karena tabel itu jejak
perkembangan — satu baris per langkah. Nilai yang sama karena itu tersalin ke seluruh langkah
satu berkas survei. Terberat satu berkas = **176 baris**.

Itu disengaja dan tidak merugikan modul ini: kueri daftar **hanya membaca langkah terakhir**
(`ROW_NUMBER() … WHERE STEP_RANK = 1`). Bila kelak diinginkan hanya langkah terakhir yang diisi,
lihat §6.

---

## 4. Periksa SESUDAH

```sql
-- 4a. Angkanya harus sama dengan hasil 2b
SELECT COUNT(*)                     AS baris,
       COUNT(s.ADJUSTERACCEPT)      AS accept,
       COUNT(s.PYSTATUSWORK)        AS status_work,
       COUNT(s.REFNO)               AS refno,
       COUNT(s.ADJUSTER_PIC)        AS adjuster_pic,
       COUNT(s.RESCHEDULE_LOCATION) AS reschedule_location
  FROM POOLDATA.T_SURVEYORLIST s;

-- 4b. Tidak boleh ada satu pun nilai yang berbeda dari sumbernya
SELECT COUNT(*) AS baris_tidak_sepadan
  FROM POOLDATA.T_SURVEYORLIST s
  JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
    ON w.PZINSKEY = s.CASEID
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim'
   AND (   NVL(s.ADJUSTERACCEPT,      '~') <> NVL(w.ADJUSTERACCEPT_1,      '~')
        OR NVL(s.PYSTATUSWORK,        '~') <> NVL(w.PYSTATUSWORK,          '~')
        OR NVL(s.REFNO,               '~') <> NVL(w.REFNO_1,               '~')
        OR NVL(s.ADJUSTER_PIC,        '~') <> NVL(w.ADJUSTERPIC_1,         '~')
        OR NVL(s.RESCHEDULE_LOCATION, '~') <> NVL(w.RESCHEDULELOCATION_1,  '~'));
```

**Keduanya wajib:** 4a membuktikan jumlahnya benar, 4b membuktikan isinya benar. Jumlah yang
benar dengan isi yang tertukar antar kolom akan lolos 4a dan tertangkap 4b.

Lalu dari sisi aplikasi:

```
claimpnc -periksa
```

Baris `[BELUM]` untuk kelima kolom harus hilang. Selama satu pun masih muncul, modul **tidak
menghidupkan** tab maupun kolom yang bergantung padanya.

---

## 5. Rollback

```sql
UPDATE POOLDATA.T_SURVEYORLIST
   SET ADJUSTERACCEPT      = NULL,
       PYSTATUSWORK        = NULL,
       REFNO               = NULL,
       ADJUSTER_PIC        = NULL,
       RESCHEDULE_LOCATION = NULL;

COMMIT;
```

Mengembalikan kelima kolom ke keadaan sebelum backfill. **Tidak ada data lain yang tersentuh** —
kelimanya kolom baru yang belum pernah dipakai siapa pun.

Setelah rollback, modul kembali ke keadaan sekarang dengan sendirinya: tab tetap tertahan,
kolom tetap bertanda `· pengganti`. Tidak ada perubahan kode yang dibutuhkan.

---

## 6. Yang TIDAK diselesaikan backfill ini

**Backfill saja basi dalam hitungan hari.** Ketiga nilai berubah sepanjang hidup berkas survei:

- `ADJUSTERACCEPT` berubah saat adjuster menerima penugasan;
- `PYSTATUSWORK` berubah saat berkas ditutup atau dibatalkan — **dan justru nilai akhirnya yang
  dibutuhkan**;
- `RESCHEDULE_LOCATION` berubah saat survei dijadwalkan ulang.

`INSERT_SURVEYORLIST` hanya dipanggil saat hasil survei disimpan, dan `UPDATE`-nya dijaga
`TNOTE IS NOT NULL` (`INSERT_SURVEYORLIST.prc:36`). Ia **tidak pernah dipanggil** saat status
objek kerja berubah.

Jadi dibutuhkan **jalur pemutakhiran saat objek kerja berubah**. Mekanismenya diserahkan Tim
Pega; tiga yang mungkin, dengan kelemahannya masing-masing:

| Cara | Kelemahan yang harus disadari |
|---|---|
| Menambah pemanggilan di activity yang mengubah status | harus ditemukan **seluruh** tempatnya — sebagian berada di luar export (`R-16`) |
| Trigger pada `PC_ASM_FW_GCNMFW_WORK` | menambah beban tulis pada tabel yang dibaca **116 rule Pega** |
| Menjalankan ulang `MERGE` §3 secara terjadwal | nilainya tertinggal selama selang jadwal; untuk tab Close berarti berkas tutup masih tampil |

Yang **tidak** disarankan: menyisipkan baris baru ke `T_SURVEYORLIST` hanya untuk mencatat
perubahan status. Tabel itu memberi makan KPI adjuster — dibaca `GetReportKPI_Survey`,
`GetReportKPIAdjuster_Progress`, dan `GetReportKPIAdjuster_FinalReport` — sehingga baris tambahan
**mengubah angka kinerja orang**. Menambah kolom aman; menambah baris tidak.

### Varian: hanya langkah terakhir

Bila diputuskan cukup langkah terakhir yang diisi, ganti `USING` pada §3 dengan pasangan ini:

```sql
USING (SELECT t.CASEID, t.INDEX_SURVEY,
              w.ADJUSTERACCEPT_1, w.PYSTATUSWORK, w.REFNO_1,
              w.ADJUSTERPIC_1, w.RESCHEDULELOCATION_1
         FROM (SELECT t.*,
                      ROW_NUMBER() OVER (PARTITION BY t.CASEID
                        ORDER BY LPAD(TRIM(t.INDEX_SURVEY), 10, '0') DESC NULLS LAST,
                                 t.TGLINPUT DESC NULLS LAST) AS STEP_RANK
                 FROM POOLDATA.T_SURVEYORLIST t) t
         JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
           ON w.PZINSKEY = t.CASEID
        WHERE t.STEP_RANK = 1
          AND w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim') w
   ON (s.CASEID = w.CASEID AND s.INDEX_SURVEY = w.INDEX_SURVEY)
```

Urutan `LPAD` dan `NULLS LAST` **harus sama persis** dengan kueri modul
(`inboxsurvey.sql`, CATATAN 3). `LPAD` bukan `TO_NUMBER`: ada baris warisan yang
`INDEX_SURVEY`-nya bukan angka, dan `TO_NUMBER` menjatuhkannya dengan ORA-01722. `NULLS LAST`
disebut tegas karena bawaan Oracle untuk `DESC` adalah `NULLS FIRST` — tanpanya baris
ber-`INDEX_SURVEY` kosong terpilih sebagai "langkah terakhir".

Satu baris yang tidak sepadan di sini berarti layar membaca langkah yang tidak terisi, dan
selnya kosong tanpa galat apa pun.

---

## 7. Yang berubah di layar setelah kolomnya terisi

Tidak ada perubahan kode yang dibutuhkan. Keempat tab dan kedua kolom menyala sendiri.

| Sebelum | Sesudah |
|---|---|
| 4 tab tertahan dengan sebabnya | Outstanding · ALL · Invoice · Close berjalan |
| Reference No bertanda `belum tersedia` | terisi |
| PIC Loss Adjuster bertanda `· pengganti` | terisi dari `ADJUSTER_PIC` |
| Location bertanda `· pengganti` | terisi dari `RESCHEDULE_LOCATION` |
| Berkas yang sudah tutup ikut tampil | tersaring sendiri oleh `PYSTATUSWORK` |

Dua kolom terakhir menuntut satu baris penukaran sumber di `inboxsurvey.sql` — dikerjakan
**setelah** `-periksa` melaporkan keterisiannya, bukan sebelum. Menukarnya lebih awal mengubah
kolom yang hari ini terisi menjadi kosong.

Diukur 2026-09-29: **1.070 dari 2.448** berkas survei sudah berstatus tutup, jadi tab yang
berjalan akan menyusut berarti begitu `PYSTATUSWORK` terisi.
