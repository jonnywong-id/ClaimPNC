# Permintaan penambahan kolom — `POOLDATA.T_CLAIM_OPENPROTECTION`

| | |
|---|---|
| **Tanggal** | 2026-10-05 |
| **Diminta oleh** | Tim migrasi Claim PNC |
| **Ditujukan ke** | **Work Owner** — yang menyetujui sekaligus menjalankan `ALTER`-nya (2026-10-05) |
| **Menempuh** | `D-63` — permintaan tertulis tim pengembang, persetujuan Work Owner, pelaksanaan DBA, lalu diuji dengan menjalankan Pega dan Go bersamaan |
| **Modul terdampak** | Inbox Accept Open Protection (`MENU_ID 66`) · Input Req Protection |
| **Sifat perubahan** | Penambahan 2 kolom · **dan pengisiannya** · **dan satu hak akses** |
| **Pelaksanaan** | `backend/migrations/0015_openprotection_sasaran_coverage.up.sql` |

---

## 1. Ringkas

**Dua kolom**, keduanya NULLABLE tanpa DEFAULT, pada `POOLDATA.T_CLAIM_OPENPROTECTION`:

| Kolom baru | Tipe | Menyalin dari |
|---|---|---|
| `OBJECT_ID` | sama dengan kolom asalnya | `T_CLAIM_OBJECTCOVERAGE.OBJECTID` |
| `OBJECT_COVERAGE_ID` | sama dengan kolom asalnya | `T_CLAIM_OBJECTCOVERAGE.OBJECTCOVERAGEID` |

**Tipenya tidak ditulis tangan.** Migrasi membacanya dari katalog dan memakai tipe yang
sama persis; bila kolom asalnya tidak ditemukan, ia berhenti dengan galat alih-alih
menebak. Tim pengembang tidak punya akses katalog, dan menebak `VARCHAR2(20)` di sini tidak
akan menghasilkan galat — Oracle mengonversi tipe diam-diam, dan yang terlihat hanyalah
pencocokan yang gagal pada nilai berspasi.

---

## 2. Apa yang rusak tanpa kedua kolom ini

Akseptasi Open Protection **tipe `8` — Perubahan Cause of Loss — tidak mengubah klaim.**

Petugas melihat permintaannya, melihat Cause of Loss sebelum dan sesudah, menekan Setujui,
dan keputusannya tersimpan. **Penyebab Kerugian pada klaim tetap seperti semula**, tanpa
satu pun gejala. Di Pega, persetujuan yang sama benar-benar mengubahnya.

Seluruh modul `inbox-accept-open-protection` hanya punya dua pernyataan tulis:

| Pernyataan | Sasaran | Untuk |
|---|---|---|
| `acceptance_decide` | `T_CLAIM_OPENPROTECTION` | keputusan disetujui/ditolak |
| `claim_apply_loss_date` | `T_CLAIM_PNC` | tipe `7` — Tanggal Kejadian |

Tipe `7` bisa karena sasarannya **kepala klaim**: satu kolom, satu baris, kunci `CLAIMID`.
Tipe `8` tidak bisa karena sasarannya **tabel anak** `T_CLAIM_OBJECTCOVERAGE` — satu klaim
dapat punya banyak objek, dan tiap objek banyak coverage.

Nilai barunya **sudah ada**: `OLD_DATA` dan `NEW_DATA` berisi Cause of Loss sebelum dan
sesudah. Yang tidak ada adalah **coverage mana** yang harus diubah.

### Kenapa tidak dicocokkan lewat `OBJECT_NAME` saja

Karena pada klaim nyata ia **pasti salah**, bukan sekadar berisiko.

Layar Pega `InputProtectionFlow` untuk `OPC-221` (klaim `PNC-1452`, diperlihatkan Work Owner
2026-10-05) menampilkan panel **Detail Perubahan Cause Of Loss** berisi empat baris:

| Object Name | Coverage Name | Cause of Loss |
|---|---|---|
| JackHugh | **Resiko A** | ILLNESS |
| JackHugh | **Resiko A** | STORM |
| JackHugh | Katastropi | WINDSTORM |
| JackHugh | **Resiko A** | HURRICANE |

`JackHugh / Resiko A` muncul **tiga kali**. Nama objek ditambah nama coverage karena itu
**tidak menunjuk satu baris**; yang membedakan ketiganya hanya `OBJECTCOVERAGEID`.

`OBJECT_NAME` adalah **nama** (`VARCHAR2(500)`), bukan kunci. Mencocokkan dengannya pada
klaim seperti ini akan menulis Penyebab Kerugian ke **baris yang salah dua dari tiga kali**,
tanpa galat.

Itu kelas cacat yang sama dengan `R-19` — aturan yang mengubah data klaim, salah diam-diam.
Karena itu tipe `8` sengaja dibiarkan tidak diterapkan sampai kolom ini ada, dan alasannya
dicatat di `docs/catatan-pengembangan.md` §38.14 sebagai keputusan, bukan sebagai kelalaian.

### Kenapa Pega bisa, dan kita tidak

Pega tidak memakai tabel ini. `Activity/InsertOpenProtectionCase-Act.xml` mengambil
sasarannya dari *work object* yang sedang terbuka di memori:

```
Local.objectid   := .ObjectID
Local.coverageid := .City          ← alias menyesatkan: isinya INDEKS coverage
TempPNCOPEN.ClaimData.ObjectList(objectid).ObjectCoverageList(coverageid).CauseOfLoss := …
```

Keduanya **subscript** — nomor urut baris di dalam daftar yang sedang ditampilkan, hidup
selama sesi form terbuka, dan tersimpan hanya di dalam blob properti internal Pega. Tidak
pernah diekspos sebagai kolom.

Jadi Pega pun tidak punya "coverage ID". Ia punya posisi baris yang kebetulan masih ada di
memori. Sistem baru relasional: nilai ada bila ditulis sebagai kolom.

---

## 3. Kenapa kunci ini, dan bukan yang lain

Kunci alami satu baris coverage adalah **tiga** kolom:

```
CLAIMID  +  OBJECTID  +  OBJECTCOVERAGEID
```

`CLAIMID` sudah ada di tabel proteksi sebagai `ID_CLAIM`. Yang kurang dua sisanya.

`OBJECTCOVERAGEID` berisi **urutan coverage di dalam objeknya** — terverifikasi 2026-09-24
pada data nyata: nilainya `"1"`, dan pasangan (klaim, nilai itu) **berulang lintas objek**.
Ia karena itu bukan pengenal global dan wajib dipasangkan dengan `OBJECTID`. Dua kolom,
bukan satu.

Dua kandidat lain dipertimbangkan dan ditolak:

| Kandidat | Kenapa gugur |
|---|---|
| `COVERAGEID` | Itu **kode jenis jaminan**, bukan pengenal baris. Ia tidak menunjuk baris mana pun secara unik |
| `URUTAN_OBJEK` + `URUTAN` | Pasangan yang dipakai modul `registrasi`, jadi tampak paling konsisten — tetapi **kedua kolom itu ditambahkan migrasi `0008`**, sehingga hanya terisi pada baris yang ditulis aplikasi ini |

### Butir kedua sudah DIUKUR — Work Owner menjalankan kueri §6 pada 2026-10-05

| Kolom | Terisi | dari 2.630 baris |
|---|---:|---:|
| `URUTAN_OBJEK` | **32** | 1,2% |
| `URUTAN` | **32** | 1,2% |
| `OBJECTID` | **2.630** | **100%** |
| `OBJECTCOVERAGEID` | **2.630** | **100%** |

Angka itu memutuskan pilihannya, dan bukan lagi dugaan: memakai `URUTAN_OBJEK`/`URUTAN`
akan membuat perubahan Cause of Loss **gagal menemukan barisnya pada 2.598 dari 2.630 baris**
— dan gagalnya tidak menghasilkan galat, hanya `UPDATE` yang tidak menyentuh apa pun.

`OBJECTID` dan `OBJECTCOVERAGEID` terisi penuh karena keduanya `NOT NULL` di tabel warisan,
sehingga kunci ini berlaku untuk baris warisan Pega maupun baris baru.

Pasangan `OBJECTID` + `OBJECTCOVERAGEID` dipilih karena ia **kunci yang sama dengan yang
Work Owner tetapkan sendiri** untuk `POOLDATA.T_CLAIM_SPREADING` (`CREATE_TABLE_2.sql`,
2026-09-26): `(CLAIMID, OBJECTID, OBJECTCOVERAGEID, TREATYTYPE)`. Dengan begitu kedua tabel
baru menunjuk baris coverage dengan cara yang sama.

---

## 4. Menambah kolom saja **tidak cukup** — tiga hal menyusul

Ini bagian yang paling mudah terlewat. Kolom yang ada tetapi kosong lebih berbahaya
daripada kolom yang tidak ada: layarnya tampak berfungsi, dan kegagalannya diam.

### 4.1 Form permintaan harus menangkapnya — dan hari ini tidak

> **DIPUTUSKAN Work Owner 2026-10-05: ikuti Pega.** Panel pemilih dibangun seperti layar
> `InputProtectionFlow`, dan tidak dicari bentuk lain.

`input-req-protection` menyimpan **`OBJECT_NAME` saja** (`validation.go:159` →
`MaxObjectNameLength = 500`), ditampilkan sebagai satu field teks read-only
(`ProtectionForm.tsx:313` — `"Object Name (dari klaim)"`). Tidak ada pilihan coverage sama
sekali.

**Bentuk yang harus dibangun**, disalin dari layar Pega yang diperlihatkan Work Owner:

| Bagian | Isi |
|---|---|
| Panel **Detail Perubahan Cause Of Loss** | tabel seluruh coverage klaim itu: **Object Name · Coverage Name · Cause of Loss**, satu baris per coverage, dengan tombol **Pilih** di tiap baris |
| **Cause Of Loss Dipilih** | terisi dari baris yang ditekan Pilih — inilah yang menjadi `OLD_DATA` |
| **Next Cause Of Loss** | dropdown master penyebab kerugian — menjadi `NEW_DATA` |
| Yang disimpan diam-diam | `OBJECT_ID` dan `OBJECT_COVERAGE_ID` dari baris yang dipilih |

Panelnya hanya muncul bila Tipe Proteksi = **"Perubahan COL"**, sama seperti Pega.

**Satu kueri baru dibutuhkan backend:** daftar coverage satu klaim beserta kunci barisnya —
`OBJECTID`, `OBJECTCOVERAGEID`, nama objek, nama coverage, dan Cause of Loss yang berlaku.
Kueri yang ada sekarang mengembalikan **satu** baris pertama, dan itu yang harus diganti.

Lebih dari itu, kueri yang mengisi panel Detail Perubahan **mengambil baris pertama**, bukan
yang dipilih pengguna (`inputreqprotection/repo/sqlstore/protection.sql`):

```sql
(SELECT cv.CAUSEOFLOSS
   FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE cv
  WHERE cv.CLAIMID = c.CLAIMID
  ORDER BY cv.OBJECTID, cv.COVERAGEID
  FETCH FIRST 1 ROW ONLY)
```

Jadi "Cause of Loss sebelumnya" yang tampil hari ini adalah milik **coverage pertama**, bukan
milik coverage yang hendak diubah. Di Pega pengguna memilihnya —
`ChoseCauseoflossProtection` bekerja pada `ObjectList(Param.ListID)`, yaitu baris yang
diklik.

**Yang dibutuhkan:** pemilih objek dan coverage di layar permintaan, dan kedua kolom baru
diisi dari pilihan itu. Itu pekerjaan kode, bukan DDL, dan akan diajukan terpisah.

### 4.2 Akun aplikasi butuh hak tulis ke tabel coverage

Penerapan perubahan menuntut `UPDATE POOLDATA.T_CLAIM_OBJECTCOVERAGE`, dan modul Open
Protection sampai hari ini tidak pernah menulisinya:

```sql
GRANT SELECT, UPDATE ON POOLDATA.T_CLAIM_OBJECTCOVERAGE TO <AKUN_APLIKASI>;
```

Ini **keputusan tersendiri**, bukan pelengkap `ALTER`. `P-1` menetapkan satu tabel ditulis
satu sistem; memberi aplikasi hak tulis atas tabel coverage klaim harus disetujui secara
sadar. Karena itu ia dibiarkan sebagai komentar di berkas migrasi, tidak ikut dijalankan.

### 4.3 Perilaku saat barisnya tidak ketemu

> **Alasan yang semula ditulis di sini KELIRU, dan Work Owner yang membantahnya
> (2026-10-05):** *"bagaimana mungkin belum ada jika pada saat input saja sudah input no
> polis dan no klaim?"*
>
> Benar. `usecase/service.go:230` mencari klaimnya **di server** saat permintaan disimpan,
> dan meneruskan `ErrClaimNotFound` apa adanya. Permintaan proteksi karena itu **tidak dapat
> dibuat** untuk klaim yang tidak ada di `T_CLAIM_PNC`. Skenario "klaimnya belum ada saat
> akseptasi" tidak dapat terjadi, dan angka 1.397 dari 2.639 tidak relevan di sini — ia
> menghalangi di tahap INPUT, bukan di tahap akseptasi.

Yang tersisa adalah kemungkinan yang jauh lebih sempit: **baris coverage-nya hilang di
antara permintaan dan persetujuan.** Itu bukan kemungkinan karangan — modul `registrasi`
menandai coverage terhapus lewat `coverage_tandai_sisa` ketika petugas membuang coverage dari
sebuah klaim:

```sql
UPDATE POOLDATA.T_CLAIM_OBJECTCOVERAGE
   SET DIHAPUS_PADA = :1
 WHERE CLAIMID = :2 AND URUTAN_OBJEK = :3 AND URUTAN > :4 AND DIHAPUS_PADA IS NULL
```

Jadi penjaganya tetap diperlukan — tetapi sebagai pengaman kejadian jarang, bukan sebagai
jalur yang sering dilalui. **Aturannya tetap sama dengan tipe `7`:** bila `UPDATE` tidak
menyentuh satu baris pun, seluruh transaksi dibatalkan dan pemanggil menerima `409` berisi
sebabnya.

Arah sebaliknya jauh lebih buruk — persetujuan tersimpan atas perubahan yang tidak pernah
terjadi, dan petugas melihat "disetujui" sementara klaim tidak berubah. Itu persis keadaan
hari ini yang permintaan ini ada untuk menutupnya.

---

## 5. Baris warisan tidak dapat ditambal

Permintaan perubahan COL yang dibuat **Pega** dan disalin ke tabel ini tidak akan pernah
punya sasaran, dan kolom baru tidak mengubahnya.

Sebabnya sudah tercatat di `docs/kolom-open-protection.md`: *"`OLD_DATA` dan `NEW_DATA`
tidak punya kolom asal. Detail perubahan DOL dan Cause of Loss tidak diekspos sebagai kolom
di tabel kerja Pega — ia tinggal di dalam blob properti."* Baris warisan karena itu sudah
kosong pada keduanya.

Kolom baru hanya akan terisi untuk permintaan yang dibuat **aplikasi ini**. Permintaan COL
lama tetap harus diselesaikan di Pega selama masa paralel.

---

## 6. Cara memastikan permintaan ini sudah tiba

**Sebelum dijalankan** — kueri yang menentukan apakah pilihan kunci di §3 benar.

> **Sudah dijalankan Work Owner 2026-10-05.** Hasilnya `2630 · 32 · 32 · 2630 · 2630`,
> dan ia membenarkan pilihan kunci di §3. Kueri ini tetap dicantumkan supaya dapat diulang
> di basis data portal lain, yang isinya tidak harus sama (`ADR-0030`).

```sql
SELECT COUNT(*)                AS BARIS,
       COUNT(URUTAN_OBJEK)     AS URUTAN_OBJEK_TERISI,
       COUNT(URUTAN)           AS URUTAN_TERISI,
       COUNT(OBJECTID)         AS OBJECTID_TERISI,
       COUNT(OBJECTCOVERAGEID) AS OBJCOVID_TERISI
  FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE;
```

**Sesudah dijalankan** — diharapkan dua baris:

```sql
SELECT column_name, data_type, char_length, data_precision, data_scale, nullable
  FROM all_tab_cols
 WHERE owner = 'POOLDATA'
   AND table_name = 'T_CLAIM_OPENPROTECTION'
   AND column_name IN ('OBJECT_ID', 'OBJECT_COVERAGE_ID');
```

Tipe keduanya harus **sama persis** dengan `OBJECTID` dan `OBJECTCOVERAGEID` pada
`T_CLAIM_OBJECTCOVERAGE`. Bila berbeda, migrasi tidak berjalan sebagaimana mestinya dan
harus ditinjau — jangan ditambal dengan `ALTER` manual.

---

## 7. Batas pernyataan ini

Yang **terukur** dan dapat diperiksa ulang:

- 16 kolom `T_CLAIM_OPENPROTECTION` dan ketiadaan identitas objek/coverage di dalamnya —
  `docs/kolom-open-protection.md` §1, dibaca dari katalog `DEV_PEGA83G`
- Dua pernyataan tulis modul akseptasi — dihitung dari
  `inboxacceptopenprotection/repo/sqlstore/acceptance.sql`
- `OBJECTCOVERAGEID` berisi urutan dalam objek dan berulang lintas objek — terverifikasi
  2026-09-24, dicatat di `registrasi/repo/sqlstore/claim.sql`
- Form permintaan hanya menyimpan `OBJECT_NAME` dan mengambil coverage pertama — dibaca
  dari `inputreqprotection`
- **Keterisian keempat kolom kunci** — diukur Work Owner 2026-10-05, lihat §3

Tidak ada lagi butir yang belum terukur pada permintaan ini. Satu dugaan yang tersisa
sebelumnya — keterisian `URUTAN_OBJEK`/`URUTAN` — sudah dijawab angka di §3.

Tidak ada satu pun nilai data nasabah di dokumen ini (`D-69`).
