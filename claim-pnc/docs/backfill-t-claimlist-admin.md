# Pengisian `POOLDATA.T_CLAIMLIST_ADMIN` dari objek kerja Pega

| | |
|---|---|
| **Tanggal** | 2026-10-07 |
| **Disusun oleh** | Tim migrasi Claim PNC |
| **Ditujukan ke** | **DBA** (pelaksana) · **Work Owner** (empat keputusan di §7) |
| **Menempuh** | `D-63` — permintaan tertulis, persetujuan Work Owner, pelaksanaan DBA |
| **Modul terdampak** | **My Inbox** (`Harness/InboxRegister_Harness`, `MENU_ID 51`) |
| **Portal** | **ASM lebih dulu** — lihat §8 |
| **Lingkungan angka** | **DEV** (service `DEV_PEGA83G`) — **bukan produksi**; lihat §8 |
| **Status** | **RANCANGAN. Belum dijalankan.** |

---

## 0. Baca ini lebih dulu

**Berkas ini BUKAN migrasi aplikasi.** Ia tidak ditaruh di `backend/migrations/` dan tidak pernah
dijalankan aplikasi Go, karena `POOLDATA.T_CLAIMLIST_ADMIN` **tidak dimiliki Go** — modul My Inbox
hanya membacanya.

Yang dikerjakan di sini adalah **operasi sekali jalan oleh DBA**, ditambah satu pertanyaan yang
harus dijawab lebih dulu dan belum terjawab sampai hari ini: **siapa yang mengisi tabel ini
sekarang.**

---

## 1. Kenapa ini dibutuhkan — angkanya

Dibaca langsung dari katalog **lingkungan DEV** (service `DEV_PEGA83G`) pada 2026-10-07, bukan dari
dokumen.

> **Seluruh angka pada berkas ini angka DEV.** Angka produksi **belum pernah dibaca** dan hampir
> pasti berbeda — termasuk perbandingan "sepertiga" di bawah. Sebelum dijalankan di produksi,
> kueri §6.1 **wajib dijalankan ulang di sana** dan angkanya diperlakukan sebagai angka baru,
> bukan dicocokkan dengan tabel ini.

| | |
|---|---:|
| Klaim PNC di `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` | **2.647** |
| Baris di `POOLDATA.T_CLAIMLIST_ADMIN` | **1.049** |
| Klaim PNC yang **belum** ada di tabel tujuan | **1.770** |

Tabel tujuan baru memuat sekitar **sepertiga** klaim PNC. Itu sebab langsung layar My Inbox
menampilkan lebih sedikit daripada Pega hari ini.

Satu akibat yang lebih kasatmata daripada jumlah baris: **`DOKUMENLENGKAP_1` kosong di seluruh
baris yang ada**, sehingga tab **Complete documents** dan **Documents not complete** sama-sama
menunjukkan nol — bukan karena tidak ada klaimnya, melainkan karena kolomnya belum pernah diisi.

> Koreksi atas `kolom-t-claimlist-admin.md`: dokumen itu menyebut tabel tujuan **40 kolom, 1.014
> baris**. Yang benar **59 kolom, 1.049 baris**. §D.6 dokumen tersebut memang mengakui daftar
> kolomnya tidak pernah dibaca dari katalog.

---

## 2. Apa yang dipindahkan — 24 kolom, bukan 59

Hanya kolom yang **benar-benar dibaca** modul My Inbox. Kolom milik modul lain sengaja tidak
disentuh: mengisinya berarti menebak apa yang modul lain butuhkan, dan tebakan itu tidak
menghasilkan galat — hanya data yang salah.

| Dipakai | Kolom |
|---|---|
| **Daftar layar** | `PZINSKEY` · `PYID` · `POLICYNO` · `QQNAME` · `BUSINESSNAME` · `SOBNAME` · `BRANCHNAME` · `GROUPPANEL_1` · `PNCCASEID` · `REGISTERDATE_1` · `PXCREATEDATETIME` · `DATEOFLOSS_1` · `REPORTDATE_1` · `PYSTATUSWORK` · `STATUSLOCK_1` · `USERTEKNIS_1` · `PXCREATEOPERATOR` · `PXTASKLABEL` · `PXASSIGNEDOPERATORID` · `DOKUMENLENGKAP_1` |
| **Unduhan saja** | `PXOBJCLASS` · `PXFLOWNAME` · `ISPENDINGCLOSE` · `BUSINESSGROUPID` |
| **Tidak diisi** | `AGING` — lihat §3.3 |

---

## 3. Tiga kolom yang tidak punya sumber senama

### 3.1 `BUSINESSGROUPID` — **terbukti** dapat diturunkan

```sql
POOLDATA.BUSINESS.BUSINESSGROUPID  dengan  BUSINESS.ID = BUSINESSCODE_1
```

Diuji terhadap **seluruh 1.049 baris** yang sudah ada: **1.049 cocok, 0 meleset.**

Jalur yang sama dipakai layar lama — `RDB List/ExportDataDetailKlaim-SQL.xml` melakukan
`JOIN pooldata.business f ON c.BUSINESSCODE_1 = f.id`.

Kolom ini menentukan **cakupan unduhan per lini bisnis**. Tanpa ia terisi, unduhan PA/Travel/
Non-MBU/Bonding mengembalikan berkas kosong.

### 3.2 `STATUSPROGRESS1` — dapat diturunkan, **dengan catatan**

`POOLDATA.GET_POSISI_PROGRESS_PNC(PYID,'sts_prg1')` **ada dan `VALID`**. Diuji terhadap 200 baris
yang sudah terisi: **89 cocok, 111 berbeda.**

Bacaan yang paling sesuai bukti: fungsinya mengembalikan posisi **sekarang**, sedangkan kolomnya
**snapshot** yang sudah bergerak sejak diisi.

**My Inbox tidak menampilkan kolom ini**, sehingga perbedaannya tidak terlihat pengguna modul ini.
Modul lain mungkin menampilkannya — dan itu keputusan Work Owner, bukan keputusan skrip.

### 3.3 `AGING` — **tidak dapat direkonstruksi.** Dibiarkan `NULL`

Diuji terhadap 970 baris terisi:

| Rumus yang dicoba | Cocok |
|---|---:|
| `SYSDATE - DATEFORAGING_1` | **0** |
| `SYSDATE - REGISTERDATE_1` | **0** |
| `SYSDATE - PXCREATEDATETIME` | **0** |

Menambahkan `AGING` kembali ke tanggal acuannya pun tidak menghasilkan satu tanggal yang dominan —
jadi ia **bukan** "umur pada hari tertentu" yang dibekukan sekali.

**Membiarkannya `NULL` aman bagi modul ini.** Kolom "Aging" di layar menampilkan tanda hubung, dan
kolom "Total Aging" tetap benar karena dihitung Go dari `REGISTERDATE_1`, bukan dari kolom ini.

---

## 4. Assignment mana yang dipakai — ini pilihan, bukan temuan

`PC_ASSIGN_WORKLIST` dapat memuat lebih dari satu assignment terbuka untuk satu klaim, sedangkan
tabel tujuan hanya punya **satu baris per klaim**. Terukur: **64 klaim PNC** punya lebih dari satu.

Skrip ini memakai **assignment terbaru** menurut `PXCREATEDATETIME`, dengan `PXINSNAME` sebagai
pemutus seri supaya hasilnya sama setiap kali dijalankan.

**Perilaku pengisi yang ada sekarang tidak dapat direkonstruksi.** Dari 53 klaim bercabang yang
sudah ada di tabel tujuan:

| Cocok dengan | Jumlah |
|---|---:|
| assignment **terbaru** | 24 |
| assignment **terlama** | 18 |
| keduanya tidak | 11 |

Tidak ada aturan yang konsisten untuk ditiru. "Terbaru" dipilih karena itulah tugas yang sedang
dikerjakan petugas — dan itu memang isi sebuah inbox.

> Koreksi atas `kolom-t-claimlist-admin.md`: dokumen itu menyebut "34 assignment per klaim".
> Angka itu membagi **seluruh** worklist dengan klaim PNC saja. Yang benar: **263.903 klaim punya
> tepat satu**, dan hanya 64 klaim PNC yang bercabang.

---

## 5. Kenapa `MERGE`, dan kenapa **hanya menyisipkan**

Skrip ini memakai `WHEN NOT MATCHED` **saja** — baris yang sudah ada tidak diubah.

Sebabnya `P-1`/`ADR-0004`: satu tabel ditulis satu sistem. **Siapa yang mengisi
`T_CLAIMLIST_ADMIN` hari ini belum diketahui** — tabel itu **nol kemunculan** di seluruh export
Pega (2.634 XML, 55 `.prc`, 8 `.fnc`). Bila skrip ini ikut memperbarui baris yang sudah ada
sementara pengisi yang tidak dikenal itu juga menulisnya, keduanya saling menimpa — dan
kegagalannya **tidak menghasilkan galat**, hanya data yang berubah sendiri.

**Akibat yang diterima:** 1.049 baris lama yang `DOKUMENLENGKAP_1`-nya kosong **tetap kosong**. Tab
status dokumen baru benar untuk 1.770 baris yang disisipkan. §6.5 menyediakan pembaruannya secara
terpisah, untuk dijalankan hanya setelah pertanyaan di atas terjawab.

### 5.1 Seberapa besar akibatnya — ditakar, bukan dikira

Kueri Go memperlakukan `NULL` sebagai **belum lengkap**, mengikuti Pega (`= '0' OR IS NULL`). Jadi
setiap baris lama akan jatuh ke tab "Documents not complete". Pertanyaannya: berapa di antaranya
yang di Pega sebenarnya **sudah lengkap**, dan karena itu salah golong?

Diukur di DEV pada 2026-10-07 terhadap 1.049 baris lama:

| Nilai sebenarnya di Pega | Jumlah | Akibat di layar |
|---|---:|---|
| kosong juga di Pega | **444** | **tidak ada selisih** — Pega sendiri menggolongkannya belum lengkap |
| `0` | **403** | **tidak ada selisih** — `0` dan `NULL` sama-sama belum lengkap |
| `1` | **30** | **salah golong** — tampil "belum lengkap", padahal lengkap |
| tidak terhitung | 172 | sebabnya belum ditelusuri — lihat §7 butir 6 |

**Jadi 30 klaim, bukan 1.049.** Itu sebabnya `WHEN MATCHED` ditahan tanpa membuat layar jadi
menyesatkan secara berarti: 30 baris dapat dibereskan belakangan, setelah kepemilikan tabel jelas.

Satu hal yang tersingkap sambil mengukurnya: **444 klaim kosong juga di Pega.** Jadi "status
dokumen tidak terisi" bukan semata akibat belum dipindahkan — tab "Documents not complete" memang
memuat klaim yang statusnya **tidak pernah ditetapkan**, di kedua sistem.

Skrip ini **aman diulang**: menjalankannya dua kali tidak menggandakan baris.

---

## 6. Skripnya

### 6.1 Periksa SEBELUM — jangan langsung menulis

Cocokkan angkanya dengan §1. Bila jauh berbeda, **hentikan dan tanyakan** — premis skrip ini sudah
berubah.

```sql
SELECT (SELECT COUNT(*) FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK
         WHERE PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC')              AS klaim_pnc_di_pega,
       (SELECT COUNT(*) FROM POOLDATA.T_CLAIMLIST_ADMIN)            AS baris_tujuan_sekarang,
       (SELECT COUNT(*) FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
         WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
           AND NOT EXISTS (SELECT 1 FROM POOLDATA.T_CLAIMLIST_ADMIN t
                            WHERE t.PZINSKEY = w.PZINSKEY))         AS akan_disisipkan
  FROM dual;
```

Di **DEV** hasilnya `2647` · `1049` · `1770` (terukur 2026-10-07). Di **produksi**, angka ini belum
pernah dibaca — catat apa pun yang keluar sebagai angka awal lingkungan itu, jangan mencocokkannya
dengan ketiga angka di atas.

### 6.2 Pengisian

```sql
MERGE INTO POOLDATA.T_CLAIMLIST_ADMIN t
USING (
    SELECT w.PZINSKEY,
           w.PYID,
           w.PXOBJCLASS,
           w.POLICYNO,
           w.QQNAME,
           w.BUSINESSNAME,
           w.SOBNAME,
           w.BRANCHNAME,
           w.GROUPPANEL_1,
           w.PNCCASEID,
           w.BUSINESSCODE_1,
           w.REGISTERDATE_1,
           w.PXCREATEDATETIME,
           w.DATEOFLOSS_1,
           w.REPORTDATE_1,
           w.DATEFORAGING_1,
           w.PYSTATUSWORK,
           w.STATUSLOCK_1,
           w.USERTEKNIS_1,
           w.PXCREATEOPERATOR,
           w.DOKUMENLENGKAP_1,
           w.ISPENDINGCLOSE,

           -- Diturunkan; lihat 3.1 dan 3.2.
           b.BUSINESSGROUPID,
           POOLDATA.GET_POSISI_PROGRESS_PNC(w.PYID, 'sts_prg1') AS STATUSPROGRESS1,

           -- Dari assignment TERBARU. Subkueri berperingkat, bukan join langsung:
           -- join langsung MENGGANDAKAN 64 klaim yang punya lebih dari satu assignment.
           a.PXASSIGNEDOPERATORID,
           a.PXTASKLABEL,
           a.PXFLOWNAME
      FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w

      -- LEFT JOIN, bukan INNER.
      --
      -- Kueri Pega lama memakai INNER JOIN ke worklist, dan itu membuang klaim yang tidak
      -- punya assignment terbuka — klaim yang di Pega tetap ada tetapi hilang dari layar.
      -- Di sini klaim tanpa assignment tetap ikut, dengan ketiga kolom assignment NULL.
      LEFT JOIN (
            SELECT PXREFOBJECTKEY,
                   PXASSIGNEDOPERATORID,
                   PXTASKLABEL,
                   PXFLOWNAME
              FROM (SELECT x.PXREFOBJECTKEY,
                           x.PXASSIGNEDOPERATORID,
                           x.PXTASKLABEL,
                           x.PXFLOWNAME,
                           ROW_NUMBER() OVER (
                               PARTITION BY x.PXREFOBJECTKEY
                               ORDER BY x.PXCREATEDATETIME DESC, x.PXINSNAME DESC
                           ) AS urutan
                      FROM DATAPEGA.PC_ASSIGN_WORKLIST x)
             WHERE urutan = 1
      ) a ON a.PXREFOBJECTKEY = w.PZINSKEY

      LEFT JOIN POOLDATA.BUSINESS b ON b.ID = w.BUSINESSCODE_1

      -- WAJIB. Tanpa ini, case type lain ikut terbawa ke antrean klaim — dan karena
      -- semuanya punya PYID, tidak ada yang tampak salah.
     WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
) s
ON (t.PZINSKEY = s.PZINSKEY)
WHEN NOT MATCHED THEN
  INSERT (PZINSKEY, PYID, PXOBJCLASS, POLICYNO, QQNAME, BUSINESSNAME,
          SOBNAME, BRANCHNAME, GROUPPANEL_1, PNCCASEID, BUSINESSCODE_1,
          BUSINESSGROUPID, REGISTERDATE_1, PXCREATEDATETIME, DATEOFLOSS_1,
          REPORTDATE_1, DATEFORAGING_1, PYSTATUSWORK, STATUSLOCK_1,
          STATUSPROGRESS1, USERTEKNIS_1, PXCREATEOPERATOR, DOKUMENLENGKAP_1,
          ISPENDINGCLOSE, PXASSIGNEDOPERATORID, PXTASKLABEL, PXFLOWNAME)
  VALUES (s.PZINSKEY, s.PYID, s.PXOBJCLASS, s.POLICYNO, s.QQNAME, s.BUSINESSNAME,
          s.SOBNAME, s.BRANCHNAME, s.GROUPPANEL_1, s.PNCCASEID, s.BUSINESSCODE_1,
          s.BUSINESSGROUPID, s.REGISTERDATE_1, s.PXCREATEDATETIME, s.DATEOFLOSS_1,
          s.REPORTDATE_1, s.DATEFORAGING_1, s.PYSTATUSWORK, s.STATUSLOCK_1,
          s.STATUSPROGRESS1, s.USERTEKNIS_1, s.PXCREATEOPERATOR, s.DOKUMENLENGKAP_1,
          s.ISPENDINGCLOSE, s.PXASSIGNEDOPERATORID, s.PXTASKLABEL, s.PXFLOWNAME);

COMMIT;
```

`AGING` sengaja tidak ada di daftar — rumusnya belum diketahui (§3.3).

### 6.3 Periksa SESUDAH — empat kueri, masing-masing membuktikan hal berbeda

Jumlah yang benar dengan isi yang tertukar antar kolom akan lolos 1 dan 2, lalu tertangkap 3 dan 4.

```sql
-- 1. Tidak ada klaim PNC yang tertinggal.  Harapan: 0
SELECT COUNT(*) AS masih_tertinggal
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
 WHERE w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND NOT EXISTS (SELECT 1 FROM POOLDATA.T_CLAIMLIST_ADMIN t
                    WHERE t.PZINSKEY = w.PZINSKEY);

-- 2. Tidak ada klaim yang tergandakan.  Harapan: 0
SELECT COUNT(*) AS pzinskey_ganda
  FROM (SELECT PZINSKEY FROM POOLDATA.T_CLAIMLIST_ADMIN
         GROUP BY PZINSKEY HAVING COUNT(*) > 1);

-- 3. Tab status dokumen jadi punya isi.
--    Inilah yang membedakan "baris bertambah" dari "layarnya ikut benar".
SELECT NVL(DOKUMENLENGKAP_1, '(kosong)') AS dokumen_lengkap, COUNT(*) AS jml
  FROM POOLDATA.T_CLAIMLIST_ADMIN
 WHERE PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
 GROUP BY NVL(DOKUMENLENGKAP_1, '(kosong)')
 ORDER BY 2 DESC;

-- 4. Cakupan unduhan per lini bisnis hidup.  Harapan: 0
SELECT COUNT(*) AS tanpa_businessgroupid
  FROM POOLDATA.T_CLAIMLIST_ADMIN
 WHERE PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND BUSINESSGROUPID IS NULL;
```

Pada kueri 3, `(kosong)` yang tersisa adalah 1.049 baris lama — itu **diharapkan**, bukan
kegagalan. Lihat §5.

### 6.4 Rollback

`P-4` mewajibkan setiap perubahan yang menyentuh data punya rollback yang tidak kosong. Karena
skrip ini **hanya menyisipkan**, rollback-nya membuang persis yang disisipkan — bukan menyentuh
1.049 baris yang sudah ada sebelumnya.

**Jalankan sebelum pengisian** untuk menyimpan daftarnya:

```sql
CREATE TABLE POOLDATA.TMP_BACKFILL_MYINBOX_20261007 AS
SELECT PZINSKEY FROM POOLDATA.T_CLAIMLIST_ADMIN;
```

**Rollback:**

```sql
DELETE FROM POOLDATA.T_CLAIMLIST_ADMIN t
 WHERE NOT EXISTS (SELECT 1 FROM POOLDATA.TMP_BACKFILL_MYINBOX_20261007 b
                    WHERE b.PZINSKEY = t.PZINSKEY);
COMMIT;
```

### 6.5 Memperbarui baris yang sudah ada — **jangan dijalankan dulu**

Blok berikut menyembuhkan 1.049 baris lama yang kolomnya kosong. **Jangan dijalankan sebelum §7
butir 1 terjawab** — selama pengisi yang ada belum diketahui, ini menjadikan tabel itu bertuan dua.

Bila Work Owner memutuskan Go menjadi satu-satunya penulis, tambahkan cabang ini **sebelum**
`WHEN NOT MATCHED` pada §6.2:

```sql
WHEN MATCHED THEN UPDATE SET
     t.PYSTATUSWORK         = s.PYSTATUSWORK,
     t.PXTASKLABEL          = s.PXTASKLABEL,
     t.PXFLOWNAME           = s.PXFLOWNAME,
     t.PXASSIGNEDOPERATORID = s.PXASSIGNEDOPERATORID,
     t.USERTEKNIS_1         = s.USERTEKNIS_1,
     t.DOKUMENLENGKAP_1     = s.DOKUMENLENGKAP_1,
     t.ISPENDINGCLOSE       = s.ISPENDINGCLOSE,
     t.STATUSLOCK_1         = s.STATUSLOCK_1,
     t.BUSINESSGROUPID      = s.BUSINESSGROUPID
```

Kesembilan kolom itu yang **berubah** selama klaim berjalan. Kolom identitas dan tanggal
registrasi sengaja tidak ikut: bila keduanya berbeda antara Pega dan tabel tujuan, itu **temuan
yang harus diperiksa** — bukan yang ditimpa diam-diam.

---

## 7. Yang belum terjawab — mohon dijawab sebelum §6.2 dijalankan

| # | Pertanyaan | Pemilik |
|---|---|---|
| 1 | **Siapa yang mengisi `T_CLAIMLIST_ADMIN` sekarang, dan dengan apa?** Tabel ini nol kemunculan di seluruh export Pega. Selama pengisinya tidak diketahui, setiap penulisan berisiko bertabrakan dengannya | **DBA + Tim Pega** |
| 2 | **Dari mana `AGING` diisi?** Tidak dapat direkonstruksi; dibiarkan `NULL` | **DBA** |
| 3 | Apakah **assignment terbaru** memang pilihan yang benar untuk 64 klaim bercabang? | **Work Owner** |
| 4 | `STATUSPROGRESS1` diisi dari fungsinya, atau dibiarkan kosong? Hasil fungsi berbeda dari 111 dari 200 baris terisi | **Work Owner** |
| 5 | **Setelah pengisian, apakah klaim baru ikut dipindahkan berkala?** Skrip ini sekali jalan — tanpa penjadwalan, My Inbox tertinggal lagi seiring Pega terus membuat klaim baru | **Tim Pega + Work Owner** |
| 6 | **172 baris tidak terhitung** di §5.1 — belum ditelusuri sebabnya | **DBA** |

Butir 5 yang paling mudah terlewat, dan sama persis dengan yang terjadi pada `T_SURVEYORLIST`:
pengisian sekali jalan **basi dalam hitungan hari** bila tidak ada yang memutakhirkannya.

Butir 6 kecil angkanya tetapi dapat menjawab butir 1. Dari 1.049 baris lama, hanya 877 yang
terhitung di §5.1; sisa 172 bisa berarti bukan kelas Work-PNC, kolomnya ternyata sudah terisi, atau
— yang paling menarik — **tidak punya padanan sama sekali di `PC_ASM_FW_GCNMFW_WORK`**. Yang
terakhir berarti ada baris yang **tidak berasal dari objek kerja Pega**, dan itu petunjuk pertama
yang nyata tentang siapa pengisinya. Kueri pemilahnya:

```sql
SELECT sebab, COUNT(*) AS jml
  FROM (
    SELECT CASE
             WHEN NVL(t.PXOBJCLASS, '-') <> 'ASM-FW-GCNMFW-Work-PNC'
                  THEN 'bukan kelas Work-PNC'
             WHEN t.DOKUMENLENGKAP_1 IS NOT NULL
                  THEN 'DOKUMENLENGKAP_1 sudah terisi'
             WHEN NOT EXISTS (SELECT 1 FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
                               WHERE w.PZINSKEY = t.PZINSKEY)
                  THEN 'tidak ada padanannya di Pega'
             ELSE 'terhitung di kueri sebelumnya'
           END AS sebab
      FROM POOLDATA.T_CLAIMLIST_ADMIN t)
 GROUP BY sebab
 ORDER BY 2 DESC;
```

Hasilnya **tidak mengubah skrip pemindahan** — `MERGE`-nya hanya menyisipkan, jadi baris apa pun
yang sudah ada tetap tidak disentuh. Yang berubah hanyalah seberapa kuat alasan mengejar butir 1.

---

## 8. Lingkungan dan portal

Dua sumbu yang berbeda, dan keduanya menuntut angka dihitung ulang.

### 8.1 Lingkungan — DEV lebih dulu

Seluruh angka pada berkas ini dibaca dari **lingkungan DEV**, service `DEV_PEGA83G`. **Angka
produksi belum pernah dibaca.**

Urutannya:

| Urutan | Lingkungan | Yang dikerjakan |
|---|---|---|
| 1 | **DEV** | jalankan §6.1 → §6.2 → §6.3. Inilah tempat `D-63` dipenuhi: **Pega dan Go dijalankan bersamaan** terhadap hasilnya, dan layar My Inbox diperiksa pengguna |
| 2 | **Produksi** | jalankan §6.1 **ulang di sana**, catat angkanya sebagai angka awal baru, baru §6.2 |

**Yang paling mudah salah di langkah 2:** memakai `2647 · 1049 · 1770` sebagai patokan. Ketiganya
angka DEV. Satu-satunya yang dibawa dari DEV ke produksi adalah **skripnya**, bukan angkanya.

Dan karena rollback §6.4 bersandar pada tabel cadangan, tabel itu **dibuat terpisah di setiap
lingkungan** — nama yang sama di DEV tidak melindungi produksi.

### 8.2 Portal

Dijalankan di **database portal ASM lebih dulu** (`ADR-0030`: satu database per entitas). Skripnya
sama untuk portal lain, tetapi setiap portal punya database sendiri — jadi §6.1 dan tabel cadangan
§6.4 diulang untuk masing-masing.

---

## 9. Jejak bukti

Seluruh angka pada berkas ini dibaca langsung dari katalog Oracle **lingkungan DEV portal ASM**
(service `DEV_PEGA83G`) pada 2026-10-07, memakai perkakas sementara yang **sudah dihapus** setelah
pembacaan selesai.

Tidak ada satu pun nilai data nasabah yang disalin ke berkas ini (`D-69`) — yang tertulis hanyalah
jumlah, nama kolom, dan nama objek database. Nama service DEV ditulis karena ia **bukan** hostname
maupun alamat produksi; ia penanda lingkungan yang justru wajib terbaca supaya angkanya tidak
tertukar dengan angka produksi.
