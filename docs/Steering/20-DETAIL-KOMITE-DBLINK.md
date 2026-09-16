# Rincian D-14 (Penjenjangan Komite) dan R-03 (DB Link)

Dokumen ini menjawab dua pertanyaan yang belum terjawab rinci di Steering:

1. **D-14** — aturan penjenjangan komite saat ini berada di mana saja?
2. **R-03** — DB Link dipakai pada proses apa saja, dan menyentuh objek apa saja?

Seluruh isi diekstraksi langsung dari 652 rule Connect-SQL dan 902 Activity di export XML.

> **Diperbarui v2.0 (2026-09-14) — Bagian 1 terjawab.** Isi `POOLDATA.EMAILKOMITE` sudah diterima
> (**21 kolom, 30 baris**), dan mekanisme penjenjangannya kini dipahami penuh. **Dua hal di bawah
> yang dibaca sebagai masalah ternyata bukan masalah**, dan satu di antaranya nyaris "diperbaiki"
> menjadi salah. Ringkasannya di §1.8 yang baru; isi §1.1–§1.7 dibiarkan sebagai catatan
> penelusuran saat itu.

---

# Bagian 1 — D-14: Penjenjangan Komite

## 1.1 Temuan utama: matriksnya sudah berupa master data

Pada Decision Log, D-14 dicatat sebagai *"matriks lengkapnya belum ada, harus dikumpulkan dari
tim bisnis"*. Setelah ditelusuri ke source, **anggapan itu tidak sepenuhnya benar**.

Matriks penjenjangan komite **sudah ada sebagai data**, tersimpan di tabel
`POOLDATA.EMAILKOMITE`. Yang belum kita punya hanyalah **isi tabelnya** — dan itu satu query.

Buktinya ada di `SetListComiteeClaimPerObjAdj` step 98:

```
childPageKomite.KomiteLoop  := TempRDBSearchEmailKomite.pxResultCount
childPageKomite.KomiteCount := 1
```

**Jumlah jenjang komite = jumlah baris yang dikembalikan query ke `EMAILKOMITE`.**
Bukan angka tetap, bukan aturan di dalam kode. Urutan jenjangnya ditentukan kolom `DEGREE`
(`ORDER BY DEGREE` di semua query).

## 1.2 Struktur tabel `POOLDATA.EMAILKOMITE`

Kolom berikut disimpulkan dari 11 query berbeda yang membacanya. Tipe dan panjangnya belum
diketahui (butuh DDL — R-08).

| Kolom | Perannya dalam penjenjangan |
|---|---|
| `DEGREE` | **Urutan jenjang.** Semua query `ORDER BY DEGREE`. Inilah yang menentukan siapa komite ke-1, ke-2, dst. |
| `TYPE_BUSINESS` | **Lini bisnis.** Nilai yang ditemukan: `NONMBU`, `NONMBUAB`, `NONMBUC`, `BONDING`, `TRAVEL`, `PA` |
| `TYPE_KOMITE` | **Kelompok komite.** Nilai `0`–`3`. Ditentukan oleh logika di aktivitas (lihat §1.4) |
| `LIMIT_BOTTOM` | **Ambang nilai bawah.** Filter `LIMIT_BOTTOM <= nilai klaim` — komite dengan ambang di atas nilai klaim tidak ikut terpilih |
| `LIMIT_BOTTOM_EXGRATIA` | Ambang khusus untuk klaim **Ex-Gratia** pada lini Travel |
| `OPERATOR_ID` | User komite — menjadi tujuan penugasan (`Param.AssignTo`) |
| `EMAIL` · `CC` | Alamat notifikasi komite |
| `STS_AKTIF` | Aktif/tidak. Semua query menyaring `= 1` |
| `STS_ADJ` | Berlaku untuk komite **adjustment** (penetapan nilai) |
| `STS_REG` | Berlaku untuk notifikasi saat **registrasi** |
| `STS_REJECT` | Berlaku untuk komite **penolakan** |
| `STS_EXGRATIA` | Berlaku untuk klaim **Ex-Gratia** |
| `STS_ABS` | Penanda user sedang tidak aktif (absen) — dicek terpisah |
| `ID` | Kunci baris; dipakai langsung pada kasus PA PHK (`WHERE ID = 2`) |

## 1.3 Sebelas query yang membaca tabel ini

Masing-masing melayani skenario berbeda. Perbedaannya ada pada kombinasi filter status dan
ambang nilai — inilah bentuk nyata "matriks" yang dimaksud D-14.

| Query (RDB rule) | Filter yang membedakan | Dipakai untuk |
|---|---|---|
| `EmailKomiteBerjenjang_sql` | `STS_ADJ=1` + `TYPE_BUSINESS` + `TYPE_KOMITE` + `LIMIT_BOTTOM` | Komite adjustment umum (Non-MBU) |
| `EmailKomiteBerjenjangBonding_sql` | `STS_ADJ=1` + `TYPE_BUSINESS` saja — **tanpa ambang nilai** | Bonding |
| `EmailKomiteBerjenjangTravel_sql` | `TYPE_BUSINESS=TRAVEL` + `LIMIT_BOTTOM` | Travel |
| `EmailKomiteBerjenjangTravelExGratia_sql` | `STS_EXGRATIA` + `LIMIT_BOTTOM_EXGRATIA` | Travel Ex-Gratia |
| `EmailKomiteBerjenjangPA_sql` | `STS_ADJ=1` + `TYPE_BUSINESS` + `LIMIT_BOTTOM` | Personal Accident |
| `EmailKomiteBerjenjangPATKI_sql` | ditambah `TYPE_KOMITE=2` | PA untuk Tenaga Kerja Asing |
| `EmailKomiteBerjenjangPAPHK_sql` | **`WHERE ID = 2`** — baris tetap, tanpa filter lain | PA kasus PHK |
| `EmailKomiteBerjenjangSimasnet_sql` | ditambah pengacakan `dbms_random.value` | Simasnet |
| `EmailKomiteAdjuster_sql` | `STS_AKTIF` + `TYPE_BUSINESS` + `TYPE_KOMITE` + `LIMIT_BOTTOM` | Komite fee adjuster |
| `EmailKomiteSalvage_sql` | sama, konteks salvage | Komite salvage |
| `GetEmailKomiteSimasnet_Reject` | `STS_REJECT=1` | Komite penolakan |
| `getEmailKomite_Register` | `STS_REG=1` + `TYPE_BUSINESS` | Notifikasi saat registrasi (bukan penjenjangan) |

> Perhatikan `EmailKomiteBerjenjangPAPHK_sql` yang memakai `WHERE ID = 2` — satu baris tetap
> yang di-hardcode di dalam query. Bila baris itu berpindah ID, alur PA PHK diam-diam salah.

## 1.4 Di mana logika pemilihannya berada

Sebelum query dijalankan, tiga parameter harus ditetapkan lebih dulu. Di sinilah aturan bisnis
yang sebenarnya berada — dan di sinilah masalahnya.

| Parameter | Dipetakan ke kolom | Ditetapkan di |
|---|---|---|
| `tempAdj.ConvertAdjustmentValue` | `LIMIT_BOTTOM` | Nilai settlement × kurs — `SetListComiteeClaimAI` / `SetListComiteeClaimPerObjAdj` step 61–63 |
| `tempAdj.pyMemo` | `TYPE_BUSINESS` | `SetEmailKomite` step 8–23 |
| `tempAdj.AcceptedNo` | `TYPE_KOMITE` | `SetEmailKomite` step 8–22 |

Nilai yang dibandingkan dengan `LIMIT_BOTTOM` berbeda menurut jenis pembayaran:

| Payment type | Nilai yang dipakai |
|---|---|
| `1` Final · `2` Interim · `5` Adjustment · Ex-Gratia | `AdjustmentValue` × kurs |
| `3` Salvage | `SalvageValue` × kurs |
| lainnya | `AdjusterFeeValue` × kurs |

**Empat aktivitas menetapkan parameter ini, masing-masing dengan aturan sendiri:**

| Aktivitas | Konteks |
|---|---|
| `SetEmailKomite` | Jalur utama — 32 step, menentukan `TYPE_BUSINESS` dan `TYPE_KOMITE` untuk Non-MBU, Bonding, Travel, PA |
| `SetEmailKomiteAdjuster` | Komite fee adjuster |
| `SetEmailKomiteSalvage` | Komite salvage |
| `SetEmailKomiteSimasnet` | Jalur Simasnet |

## 1.5 Masalah pada logika pemilihan

Aturan di keempat aktivitas itu **bercampur dengan nama orang**. Contoh nyata dari
`SetEmailKomite`:

```
step 10  jika nilai <= 50.000.000 DAN UserTeknis = "ELLENSUPRIYATI"
         maka TYPE_KOMITE := "1" dan nilai pembanding dipaksa jadi 50.000.001
         // komentar asli: "kalo ellen < 50jt, komite start dari INDRA"

step 14  jika nilai <= 100.000.000 DAN UserTeknis = "INDRAGUNAWAN..."
         maka TYPE_KOMITE := "2" dan nilai pembanding dipaksa jadi 100.000.001

step 12  jika nilai <= 50.000.000 DAN UserTeknis = "YOHANES..."
         maka TYPE_BUSINESS := "NONMBUAB", TYPE_KOMITE := "0"
```

Polanya: **nilai pembanding sengaja dinaikkan melewati ambang** agar jenjang komite tertentu
ikut atau tidak ikut terpilih, dengan syarat siapa PIC Teknis-nya.

Ditambah percabangan berdasarkan hostname server:

```
GetKomiteApproval step 1:
  ConvertAdjustmentValue := jika server = <hostname entitas Timor-Leste>
                           maka 3.500  selain itu  50.000.000
GetKomiteApproval step 2:
  jika jabatan user = "PA" atau "TRAVEL" maka 30.000.000
```

Angka `3.500` versus `50.000.000` adalah **mata uang berbeda** — entitas USD memakai 3.500.

## 1.5.1 Telusur lengkap ambang `3.500`

Angka `3.500` hanya muncul di **dua aktivitas, tiga tempat**. Tidak ada satu pun di SQL, Section,
Harness, When, Data Transform, Flow Action, maupun Report Definition — seluruh kemunculan di sana
terbukti hanya timestamp, `pyAutomationID`, dan checksum.

| # | Lokasi | Bentuk | Pemicu |
|---|---|---|---|
| 1 | `GetKomiteApproval` step 1 | `ConvertAdjustmentValue := @If(pxRequestor.pxReqServer == <hostname entitas Timor-Leste>, 3500, 50000000)` | **Hostname server** |
| 2 | `SetEmailKomite` step 11 | syarat `ConvertAdjustmentValue <= 3500` → set `TYPE_KOMITE = 1`, nilai dipaksa `3501` | `LSC_ID = "SMI"` **dan** `UserTeknis = "ELLENSUPRIYATI"` |
| 3 | `SetEmailKomite` step 13 | syarat `ConvertAdjustmentValue <= 3500` → set `TYPE_KOMITE = 0` | `LSC_ID = "SMI"` |

Pasangan USD untuk ambang `100.000.000` adalah `7.000`, di tempat yang sama:

| # | Lokasi | Bentuk | Pemicu |
|---|---|---|---|
| 4 | `SetEmailKomite` step 15 | syarat `<= 7000` → set `TYPE_KOMITE = 2`, nilai dipaksa `7001` | `LSC_ID = "SMI"` **dan** `UserTeknis = "INDRAGUNAWAN"` |
| 5 | `SetEmailKomite` step 18 | `TYPE_KOMITE := @If(ConvertAdjustmentValue > 7000, 2, 1)` | `LSC_ID = "SMI"` |

Rasio kedua pasangan konsisten: `50.000.000 ÷ 3.500 ≈ 100.000.000 ÷ 7.000 ≈ 14.285`. Ini kurs
IDR/USD yang **dibekukan ke dalam kode** — bukan diambil dari master kurs yang sudah dipakai di
tempat lain untuk mengonversi nilai klaim.

> Catatan: `CompressImage_Act` juga memuat angka `7000`, tetapi itu ambang ukuran berkas untuk
> kompresi gambar — **tidak berhubungan** dengan komite.

## 1.5.2 Dua mekanisme berbeda untuk hal yang sama

Ini temuan yang paling perlu diperhatikan. Kedua lokasi di atas menentukan "entitas mana ini"
dengan cara yang **berbeda dan dikelola terpisah**:

| | `GetKomiteApproval` | `SetEmailKomite` |
|---|---|---|
| Cara mengenali entitas | Membandingkan **hostname** dengan teks literal | Membaca `TempGetApp.LSC_ID` |
| Sumber nilainya | Ditulis di dalam kode | Hasil query tabel `POOLDATA.DB_LINK_PEGA` |
| Dipakai di | 1 tempat | 22 tempat |

`LSC_ID` berasal dari `GetLinkAppClaim`, yang menjalankan:

```sql
SELECT app AS "LSC_ID", NPP AS "LSC_NOTE"
  FROM pooldata.db_link_pega
 WHERE appip LIKE '%' || <hostname server> || '%'
```

Artinya **pemetaan hostname → entitas sudah berupa master data** di tabel `DB_LINK_PEGA`, dengan
nilai yang ditemukan: `ASM`, `SIMASNET`, `SMI`, `PEGAKREDIT`. Bila lookup gagal, `GetLinkAppClaim`
step 3 jatuh ke nilai baku `ASM`.

**Akibat dari ketidakkonsistenan ini:** bila server entitas USD dipindah atau namanya diubah,
`SetEmailKomite` tetap benar karena membaca tabel, sedangkan `GetKomiteApproval` diam-diam jatuh
ke `50.000.000` — ambang **rupiah** dipakai untuk klaim **dolar**. Selisihnya sekitar 14.000 kali
lipat, dan tidak ada pesan kesalahan yang muncul.

## 1.5.3 Kapan masing-masing ambang benar-benar dipakai

Ambang di `GetKomiteApproval` **tidak dipakai di jalur klaim normal**. Rantai pemanggilannya:

```
ValidationTypePaymentAdj
  ├─ s43 → SetListComiteeClaimPerObjAdj      ← JALUR KLAIM NORMAL
  │         s61–63  ConvertAdjustmentValue := nilai settlement × kurs
  │         s86–89  → SetEmailKomiteSimasnet / SetEmailKomite /
  │                    SetEmailKomiteSalvage / SetEmailKomiteAdjuster
  │         s98     KomiteLoop := jumlah baris hasil query
  └─ s45 → SetListComiteeClaimAI             ← JALUR AI, pola sama

GetKomiteApproval                            ← JALUR TERPISAH
  dipanggil hanya oleh: CNMInsertDetailSurveyors_act s7
                        CNMUpdateMasterRekening_act s31
  s1  ConvertAdjustmentValue := 3500 / 50.000.000   ← ambang tetap
  s2  := 30.000.000  bila jabatan user = PA atau TRAVEL
  s11 → SetEmailKomite
```

Jadi:

- **Jalur klaim normal** — `ConvertAdjustmentValue` berisi **nilai settlement** hasil konversi
  kurs. Angka 3.500 hanya muncul sebagai **batas pembanding** di syarat step 11 dan 13.
- **Jalur `GetKomiteApproval`** — dipakai dua aktivitas master data (input detail surveyor dan
  pembaruan master rekening). Di sini `ConvertAdjustmentValue` berisi **ambang tetap**, bukan
  nilai klaim.

**Satu properti dipakai untuk dua makna berbeda.** Kadang berarti "nilai klaim", kadang berarti
"ambang". Yang menentukan maknanya adalah aktivitas mana yang berjalan terakhir — dan itu tidak
terlihat saat membaca salah satu aktivitas saja.

## 1.5.4 Yang harus dilakukan di sistem baru

| Masalah | Perlakuan |
|---|---|
| Ambang `3.500` / `7.000` / `50jt` / `100jt` / `30jt` di kode | Master data per entitas dan per mata uang (D-15) |
| Kurs 14.285 dibekukan lewat pasangan angka | Pakai master kurs yang sudah ada, jangan bekukan pasangan ambang |
| Pengenalan entitas lewat hostname literal | Selalu lewat pemetaan entitas — jangan pernah membandingkan hostname di kode (§14.4) |
| `ConvertAdjustmentValue` berarti dua hal | Pisahkan: `NilaiSettlement` dan `AmbangKomite` sebagai dua konsep berbeda |
| Nama orang jadi syarat (`ELLENSUPRIYATI`, `INDRAGUNAWAN`, `YOHANESRAYMONDADIKARTA`) | Peran dan izin dari master data, bukan identitas orang |

**Yang perlu dikonfirmasi ke tim bisnis:** apakah entitas USD memang hanya satu (`SMI`), dan
apakah ambang 3.500/7.000 masih berlaku — mengingat kurs 14.285 sudah tidak mencerminkan kurs
saat ini.

> Seluruh perilaku ini masuk lingkup **D-15**: nama orang, ambang nilai, dan hostname menjadi
> master data yang dapat diubah tanpa deploy. Yang perlu dipastikan ke tim bisnis bukan
> "berapa jenjangnya", melainkan **mana aturan yang masih berlaku dan mana yang sisa tambalan lama**.

## 1.6 Mekanisme perputaran jenjang

```
SetListComiteeClaim*   KomiteLoop  := jumlah baris hasil query EMAILKOMITE
                       KomiteCount := 1
        │
        ▼
KomiteRouter           KomiteCount = 1 → workbasket "komitepnc"
                       KomiteCount = 2 → workbasket "komitepnc2"
                       KomiteCount = 3 → workbasket "komitepnc3"
                       KomiteCount = 4 → workbasket "komitepnc4"
                       KomiteCount := KomiteCount + 1 ; AcceptStatus := ""
        │
        ▼
Komite mengambil keputusan  AcceptStatus = 1 (setuju) / 2 (tolak)
        │
        ▼
IsKomiteLoop (when)    AcceptStatus = 1  DAN  KomiteCount <= KomiteLoop
                       ├─ benar → ulang ke KomiteRouter (jenjang berikutnya)
                       └─ salah → selesai
```

Aturan tambahan yang ditemukan:

- **Menolak = langsung selesai.** `KomitePost_Adjustment` step 65 dan `KomitePost_LiableKlaim`
  step 22–23: bila `AcceptStatus = 2`, `KomiteCount` dipaksa sama dengan `KomiteLoop` sehingga
  perulangan berhenti. Penolakan satu jenjang membatalkan seluruh sisa jenjang.
- **Persetujuan penuh baru diakui di jenjang terakhir.** `IsKomiteApprove` di-set `1` hanya
  ketika `KomiteCount == KomiteLoop` **dan** `AcceptStatus = 1`.
- **`KomitePost_Reject` step 2** menetapkan `KomiteLoop := -1` — penanda alur penolakan.
- **Batas 4 jenjang berasal dari router**, bukan dari data. Bila `EMAILKOMITE` mengembalikan
  lebih dari 4 baris, `KomiteRouter` tidak punya cabang untuk jenjang ke-5.

## 1.7 Yang perlu diminta untuk melengkapi D-14

| # | Yang diminta | Kepada | Kenapa |
|---|---|---|---|
| 1 | Isi tabel `POOLDATA.EMAILKOMITE` (seluruh baris, seluruh kolom) | DBA | Inilah matriks penjenjangan yang sebenarnya |
| 2 | DDL tabel `EMAILKOMITE` | DBA | Tipe kolom, panjang, constraint (bagian dari R-08) |
| 3 | Konfirmasi aturan berbasis nama orang di `SetEmailKomite` step 8–23 | Tim bisnis | Mana yang masih berlaku, mana sisa tambalan lama |
| 4 | Konfirmasi ambang 50jt / 30jt / 100jt / 3.500 | Tim bisnis | Apakah masih berlaku, dan apakah 3.500 memang USD Timor-Leste |
| 5 | Apakah jenjang bisa lebih dari 4 | Tim bisnis | Batas 4 saat ini berasal dari router, bukan dari data |

**Query untuk permintaan nomor 1:**

```sql
SELECT ID, DEGREE, TYPE_BUSINESS, TYPE_KOMITE,
       LIMIT_BOTTOM, LIMIT_BOTTOM_EXGRATIA,
       OPERATOR_ID, EMAIL, CC,
       STS_AKTIF, STS_ADJ, STS_REG, STS_REJECT, STS_EXGRATIA, STS_ABS
  FROM POOLDATA.EMAILKOMITE
 ORDER BY TYPE_BUSINESS, TYPE_KOMITE, LIMIT_BOTTOM, DEGREE;
```

> **Dampak ke status D-14:** dari *"matriks belum ada"* menjadi *"matriks ada sebagai master
> data, tinggal diambil isinya"*. Ini menurunkan bobot risikonya secara berarti — dan sekaligus
> membuktikan bahwa keputusan D-15 (nilai bisnis jadi master data) memang arah yang benar,
> karena sistem lama pun sudah setengah jalan ke sana.

---

## 1.8 Jawaban final setelah master diterima (2026-09-14)

Isi `POOLDATA.EMAILKOMITE` diterima: **21 kolom, 30 baris**. Seluruh pertanyaan §1.7 terjawab, dan
**dua pembacaan di §1.5 terbukti keliru**.

### 1.8.1 `LIMIT_BOTTOM <=` bukan cacat — itu mekanisme penjenjangannya

| Bukti | Isi |
|---|---|
| `Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459` | `childPageKomite.KomiteLoop := TempRDBSearchEmailKomite.pxResultCount` |
| `When/IsKomiteLoop-When.xml` | `.AcceptStatus = "1"` **dan** `.KomiteCount <= .KomiteLoop` |
| Sebaran kolom di 17 kueri `EMAILKOMITE` | `LIMIT_BOTTOM` difilter di **11**; `LIMIT_TOP` di **0** |

**Jumlah jenjang persetujuan = jumlah baris yang dikembalikan kueri**, dan flow memutar
`KomiteCount` dari 1 sampai `KomiteLoop`. Karena penyaringnya hanya batas bawah, seluruh jenjang
sampai tingkat nilai klaim ikut menyetujui: **klaim kecil sedikit penyetuju, klaim besar banyak
penyetuju**.

> **Perbaikan yang nyaris diterapkan dan akan merusak.** Mengganti penyaring menjadi rentang
> tertutup `LIMIT_BOTTOM <= nilai <= LIMIT_TOP` akan mengembalikan **tepat satu baris** →
> `KomiteLoop = 1` → **satu jenjang persetujuan berapa pun nilai klaim**. Itu menghapus
> penjenjangan yang menjadi inti `D-14` dan `BRD §11.4`. Kerangka pertanyaannya dikoreksi sebelum
> jawaban diterapkan (`D-47`).

**Keputusan:** model kumulatif **dipertahankan**; `LIMIT_TOP` dipakai sebagai **validasi
integritas master** — menolak master yang rentangnya tumpang tindih atau berlubang antar `DEGREE`
dalam satu `TYPE_BUSINESS` + `TYPE_KOMITE`.

### 1.8.2 `TYPE_KOMITE` adalah pita nilai **hanya** di Non-MBU

Baris aktif (`STS_AKTIF='1'`, `STS_ADJ='1'`) pada master yang diterima:

| Lini | Ambang bawah per jenjang | `DEGREE` | `TYPE_KOMITE` |
|---|---|---|---|
| **NONMBU** | `0` · `50.000.001` | 1 · 1 | `1` · `1` |
| **NONMBU** | `100.000.001` · `500.000.001` · `1.000.000.001` | 2 · 3 · 4 | `2` · `2` · `2` |
| **PA** | `0` · `10.000.001` · `50.000.001` · `100.000.001` | 1 · 2 · 3 · 4 | **`2` · `1` · `1` · `2`** |
| **TRAVEL** | `0` · `50.000.001` · `100.000.001` | 1 · 2 · 3 | `1` · `1` · `1` |

Pada **Non-MBU**, `TYPE_KOMITE` memang memisahkan pita ≤ Rp 100.000.000 dari pita di atasnya.
Pada **PA** nilainya berselang-seling **2 · 1 · 1 · 2** menaiki tangga — di sana ia membedakan
**jalur PA reguler dari PA TKI** (`EmailKomiteBerjenjangPATKI_sql-SQL.xml:82` mematok
`type_komite='2'`), bukan pita nilai. Pada **Travel**, seluruh jenjang aktif bernilai `1`, termasuk
jenjang di atas Rp 100.000.000.

**Akibat bila filter pita diberlakukan seragam ke semua lini** — dihitung dari master di atas:

| Kasus | Perilaku sistem lama | Bila pita disaring seragam |
|---|---|---|
| PA Rp 5.000.000 | 1 penyetuju | **0 penyetuju** — klaim mandek |
| PA Rp 75.000.000 | 3 penyetuju | 2 penyetuju |
| PA Rp 150.000.000 | 4 penyetuju | 2 penyetuju |
| Travel Rp 150.000.000 | 3 penyetuju | **0 penyetuju** — klaim mandek |
| Non-MBU, seluruh nilai | — | tidak berubah |

**Keputusan (`D-70`):** filter pita berlaku **khusus Non-MBU**; lini lain mengikuti kuerinya apa
adanya. Ini membatasi cakupan `D-52`, tidak membatalkannya.

### 1.8.3 Sisa pertanyaan §1.7

| # | Pertanyaan | Status |
|---|---|---|
| 1 | Isi tabel | ✅ diterima — 21 kolom, 30 baris |
| 2 | DDL tabel | Terbuka — bagian `R-08` |
| 3 | Aturan berbasis nama orang di `SetEmailKomite` | ✅ **dicabut** — diganti pita nilai yang diturunkan dari nilai klaim (`D-52`) |
| 4 | Konfirmasi ambang | ✅ **8 ambang komite unik** terverifikasi; ambang `3.500` dipicu **hostname entitas Timor-Leste** |
| 5 | Apakah jenjang bisa lebih dari 4 | ✅ **ya secara mekanisme** — jumlah jenjang mengikuti jumlah baris master, bukan batas tetap. Maksimum saat ini 4 karena isi master, bukan karena aturan |

**Yang belum terjawab dan dibawa ke tiket `B-7`/`B-12`:** tiga kueri yang memfilter `TYPE_KOMITE`
secara dinamis — `EmailKomiteBerjenjang_sql`, `EmailKomiteAdjuster_sql`, `EmailKomiteSalvage_sql` —
menerima nilainya dari pemanggil lewat `tempAdj.pyMemo` dan `tempAdj.AcceptedNo`. **Lini apa saja
yang benar-benar melewati ketiga kueri itu belum ditelusuri sampai ke sumber nilainya.**

Ditambah satu kekosongan data: **PA dan Travel di atas Rp 200.000.000 tidak punya baris master**.

---

# Bagian 2 — R-03: Inventaris DB Link

## 2.1 Ringkasan

| Ukuran | Jumlah |
|---|---|
| Total pemakaian DB Link | **64** |
| DB Link berbeda | **6** |
| Objek remote unik | **28** |
| Rule Connect-SQL yang memakainya | **27** |

| DB Link | Pemakaian | Objek | Rule | Sistem sumber |
|---|---|---|---|---|
| `@ASMD` | 55 | 20 | 24 | Database inti ASM (produksi) |
| `@SIMASNET` | 3 | 2 | 1 | Simasnet |
| `@SMI` | 2 | 2 | 1 | Sinarmas MSIG |
| `@OPJAVA` | 2 | 2 | 1 | Sistem OPJAVA |
| `@PROD_ASM` | 1 | 1 | 1 | Produksi ASM |
| `@PROD_TKA` | 1 | 1 | 1 | Sistem TKA |

> `@ASMD` menyumbang **86% dari seluruh pemakaian**. Bila hanya satu integrasi yang bisa
> diprioritaskan, itu adalah ASMD.

## 2.2 Objek remote — apa dan untuk apa

| DB Link | Objek remote | Jenis | Rule | Untuk apa |
|---|---|---|---|---|
| `@ASMD` | `DATAMINING.GET_WORKING_HOURS` | function | 6 | Menghitung selisih **jam kerja** antara dua waktu — dipakai untuk TAT dan KPI, agar akhir pekan dan hari libur tidak ikut terhitung |
| `@ASMD` | `HRDASM.V_HRD_MST` | tabel/view | 5 | Master pegawai HRD — mencari cabang, atasan, dan unit kerja seorang user |
| `@ASMD` | `LST_USER_ASURANSI` | tabel/view | 3 | Pemetaan user aplikasi ke kode cabang asuransi |
| `@ASMD` | `MST_DET_SALES` | tabel/view | 3 | Detail data sales/agen per polis — dipakai saat recovery dan penelusuran sumber bisnis |
| `@ASMD` | `GENERAL.LST_MITRA` | tabel/view | 2 | Master mitra (bengkel, investigator, surveyor eksternal) beserta akun login-nya |
| `@ASMD` | `GENERAL.MST_BUKA_PROTEKSI` | function | 2 | Procedure pembukaan proteksi di sistem sumber — dipanggil saat Open Protection dibuat |
| `@ASMD` | `GET_NAMA_AGEN` | function | 2 | Function: kode agen → nama agen |
| `@ASMD` | `GET_NAMA_BISNIS` | function | 2 | Function: kode bisnis → nama lini bisnis |
| `@ASMD` | `GET_NAMA_CABANG` | function | 2 | Function: kode cabang → nama cabang |
| `@ASMD` | `GET_NAMA_CLIENT` | function | 2 | Function: ID client → nama tertanggung |
| `@ASMD` | `GET_NAMA_MO` | function | 2 | Function: kode MO → nama Marketing Officer |
| `@ASMD` | `MST_SALES` | tabel/view | 2 | Master sales — dipakai bersama GET_NAMA_AGEN pada layar polis |
| `@ASMD` | `COLLECTION.MST_DET_SALES` | tabel/view | 1 | Sama seperti di atas, diakses lewat skema COLLECTION untuk laporan tanggal registrasi |
| `@ASMD` | `COLLECTION.TEMP_PREMI_WOM` | tabel/view | 1 | Data premi WOM Finance — dipakai menghitung total premi |
| `@ASMD` | `GENERAL.HRD_LBR` | tabel/view | 1 | Kalender **hari libur** perusahaan — dasar perhitungan hari kerja |
| `@ASMD` | `GL.T_ALL_PAYMENT` | tabel/view | 1 | Riwayat pembayaran di General Ledger — dicari berdasarkan nomor rekening |
| `@ASMD` | `LST_DET_CABANG` | tabel/view | 1 | Detail cabang |
| `@ASMD` | `MBU.T_CADANGAN_KLAIM_KREDIT_04203` | tabel/view | 1 | Cadangan klaim kredit Mandala Finance — dipakai menghitung total premi |
| `@ASMD` | `TREATY_LOSS` | tabel/view | 1 | Data kerugian treaty — dipakai laporan outstanding per cabang |
| `@ASMD` | `V_KLAIM` | tabel/view | 1 | View klaim di sistem sumber — dicek saat membuka proteksi agar tidak dobel |
| `@OPJAVA` | `NEW_GENERAL.M_USER` | tabel/view | 1 | Master user sistem OPJAVA |
| `@OPJAVA` | `NEW_GENERAL.M_USER_JOB` | tabel/view | 1 | Beban kerja (job count) per user — dipakai menyeimbangkan penugasan |
| `@PROD_ASM` | `POOLDATA.AGENT` | tabel/view | 1 | Master agen di database produksi ASM |
| `@PROD_TKA` | `ANEKA.MST_SHARE_PU` | tabel/view | 1 | Master share Penanggung Utama untuk lini TKA — dipakai perhitungan spreading |
| `@SIMASNET` | `GENERAL.MST_BUKA_PROTEKSI` | function | 1 | Procedure pembukaan proteksi di sistem sumber — dipanggil saat Open Protection dibuat |
| `@SIMASNET` | `V_KLAIM` | tabel/view | 1 | View klaim di sistem sumber — dicek saat membuka proteksi agar tidak dobel |
| `@SMI` | `GENERAL.MST_BUKA_PROTEKSI` | function | 1 | Procedure pembukaan proteksi di sistem sumber — dipanggil saat Open Protection dibuat |
| `@SMI` | `V_KLAIM` | tabel/view | 1 | View klaim di sistem sumber — dicek saat membuka proteksi agar tidak dobel |

## 2.3 Proses bisnis yang terdampak

Dikelompokkan menurut proses, bukan menurut rule — inilah yang menentukan modul mana yang
tertahan bila API penggantinya belum ada.

| Proses bisnis | Objek remote yang dipakai | Modul terdampak |
|---|---|---|
| Pencarian & tampilan Polis | `GET_NAMA_AGEN` · `GET_NAMA_BISNIS` · `GET_NAMA_CABANG` · `GET_NAMA_CLIENT` · `GET_NAMA_MO` · `MST_SALES` · `LST_DET_CABANG` | B-1 |
| Open Protection | `MST_BUKA_PROTEKSI` · `V_KLAIM` · `MST_BUKA_PROTEKSI` · `V_KLAIM` · `MST_BUKA_PROTEKSI` · `V_KLAIM` | B-13 |
| Registrasi · Komite · Master user | `V_HRD_MST` · `LST_USER_ASURANSI` | B-2 · B-7 · F-4 |
| Spreading TKA | `AGENT` · `MST_SHARE_PU` | B-4 · B-9 |
| Laporan TAT & KPI | `GET_WORKING_HOURS` | S-2 · S-7 |
| Salvage & Recovery | `MST_DET_SALES` | B-12 |
| Investigator & Laporan Mitra | `LST_MITRA` | B-11 · S-2 |
| Laporan Compliance & TAT | `MST_DET_SALES` | S-2 |
| Perhitungan Premi | `TEMP_PREMI_WOM` | B-5 |
| Perhitungan hari kerja (TAT) | `HRD_LBR` | S-7 |
| Pencarian klaim by rekening | `T_ALL_PAYMENT` | B-10 |
| Perhitungan Premi (Asuransi Kredit) | `T_CADANGAN_KLAIM_KREDIT_04203` | B-5 |
| Laporan Outstanding per Cabang | `TREATY_LOSS` | S-2 |
| Master User Teknis | `M_USER` | F-4 |
| Master User Teknis (beban kerja) | `M_USER_JOB` | F-4 · B-6 |

## 2.4 Daftar lengkap 27 rule pemakai

| Rule Connect-SQL | DB Link | Objek remote | Dipanggil aktivitas |
|---|---|---|---|
| `AmbilDataKlaimDenganNoRekening` | `@ASMD` | `T_ALL_PAYMENT` | `PNCSearchHistoryKlaim_Act` |
| `BrowseNonMBUUsers` | `@ASMD` | `V_HRD_MST` | `PNCCallRDBName_act` |
| `ExportDataKomitesKlaimNONMBU` | `@ASMD` | `GET_WORKING_HOURS` | `PNCReportDataKomites_act` |
| `GetIDCabang1` | `@ASMD` | `V_HRD_MST` · `LST_USER_ASURANSI` | `GetLostAdjuster_act`, `KomitePost_Survey`, `SetTempLostAdjuster` |
| `ExportDetailMitraReport` | `@ASMD` | `LST_MITRA` | `PNCMitraReport_Act` |
| `GetDataOutstandingperCabangExport` | `@ASMD` | `TREATY_LOSS` | `ExportDataOSCabang` |
| `GetDataCabangToEmail` | `@ASMD` | `V_HRD_MST` · `LST_USER_ASURANSI` | `SendDataDariCabangKeKantorPusat_ACT` |
| `GetDataMitraLoginInvest` | `@ASMD` | `LST_MITRA` | `InjectDataInvetigatorMitra` |
| `SetTotalJobMstUserTeknis` | `@OPJAVA` `@ASMD` | `V_HRD_MST` · `M_USER` · `M_USER_JOB` | `SetTotalJob_act` |
| `CheckProtectTable` | `@ASMD` | `MST_BUKA_PROTEKSI` | `InsertOpenProtectionCase` |
| `InsertOpenPortectionCase` | `@SIMASNET` | `MST_BUKA_PROTEKSI` · `V_KLAIM` | `InsertOpenProtectionCase` |
| `InsertOpenPortectionCaseASM` | `@ASMD` | `MST_BUKA_PROTEKSI` · `V_KLAIM` | `InsertOpenProtectionCase` |
| `InsertOpenPortectionCaseSMI` | `@SMI` | `MST_BUKA_PROTEKSI` · `V_KLAIM` | `InsertOpenProtectionCase` |
| `CheckHoliday_SQL` | `@ASMD` | `HRD_LBR` | `GCNMTimeDifferenceWorkCalender_Act` |
| `BrowseDataPolicyRNWAllFilter_SQL` | `@ASMD` | `GET_NAMA_AGEN` · `GET_NAMA_BISNIS` · `GET_NAMA_CABANG` · `GET_NAMA_CLIENT` · `GET_NAMA_MO` · `MST_DET_SALES` · `MST_SALES` | `NextPageGrid_Act` |
| `BrowseDataPolicyRNWRTTAllFilter_SQL` | `@ASMD` | `GET_NAMA_AGEN` · `GET_NAMA_BISNIS` · `GET_NAMA_CABANG` · `GET_NAMA_CLIENT` · `GET_NAMA_MO` · `LST_DET_CABANG` · `MST_DET_SALES` · `MST_SALES` | `NextPageGrid_Act` |
| `GetMandalaFinancePremi_SQL` | `@ASMD` | `T_CADANGAN_KLAIM_KREDIT_04203` | `GetTotalPremi` |
| `GetwomPremi` | `@ASMD` | `TEMP_PREMI_WOM` | `GetTotalPremi` |
| `GetShareTKA` | `@PROD_ASM` `@PROD_TKA` | `MST_SHARE_PU` · `AGENT` | `SetNilaiResikoSendiri`, `SpreadingTKA` |
| `BroswseKlaimByRegisterDate` | `@ASMD` | `MST_DET_SALES` | `PNCComplianceReport_Act`, `PNCTATReport1_Act` |
| `GetIDCabang` | `@ASMD` | `V_HRD_MST` · `LST_USER_ASURANSI` | `AchiveDocument_klaimAdmin`, `CheckViewPolis_act`, `CreateInputKlaim_PNC` (+10) |
| `GetRecoveryClaimData` | `@ASMD` | `MST_DET_SALES` | `Insert_mst_recoveryKlaimASM` |
| `BrowseDataKPIAdmin` | `@ASMD` | `GET_WORKING_HOURS` | `PNCReportKPIAdmin_Act` |
| `BrowseDataKPIAdmin_PA` | `@ASMD` | `GET_WORKING_HOURS` | `PNCReportKPIAdmin_Act` |
| `GetDataKPIAdmin` | `@ASMD` | `GET_WORKING_HOURS` | `PNCReportKPIAdmin_Act` |
| `GetDataKPIAdminPA` | `@ASMD` | `GET_WORKING_HOURS` | `PNCReportKPIAdmin_Act` |
| `GetDataKPIAdminPA_khususPA` | `@ASMD` | `GET_WORKING_HOURS` | `PNCReportKPIAdmin_Act_khususPA` |

## 2.5 Enam API pengganti yang dibutuhkan

Digabung menurut sistem pemilik, bukan menurut objek — supaya jumlah pihak yang harus
dikoordinasikan sesedikit mungkin.

| # | API yang dibutuhkan | Menggantikan | Prioritas |
|---|---|---|---|
| 1 | **API Jam Kerja & Hari Libur** | `DATAMINING.GET_WORKING_HOURS` (6 rule) · `GENERAL.HRD_LBR` | Tinggi — dipakai seluruh laporan TAT dan KPI |
| 2 | **API Pegawai & Cabang** | `HRDASM.V_HRD_MST` (5 rule) · `LST_USER_ASURANSI` (3) · `LST_DET_CABANG` | Tinggi — `GetIDCabang` dipanggil 13 aktivitas, termasuk registrasi |
| 3 | **API Open Protection** | `GENERAL.MST_BUKA_PROTEKSI` + `V_KLAIM` di **3 sistem** (@ASMD, @SIMASNET, @SMI) | Tinggi — modul B-13 tidak jalan tanpa ini |
| 4 | **API Master Sales & Agen** | `MST_DET_SALES` · `MST_SALES` · `POOLDATA.AGENT` · fungsi `GET_NAMA_*` (5) | Sedang — sebagian besar hanya kode → nama, bisa disalin berkala |
| 5 | **API Premi & Pembayaran** | `GL.T_ALL_PAYMENT` · `COLLECTION.TEMP_PREMI_WOM` · `MBU.T_CADANGAN_KLAIM_KREDIT_04203` | Sedang |
| 6 | **API Mitra & User Lintas Sistem** | `GENERAL.LST_MITRA` · `NEW_GENERAL.M_USER` · `M_USER_JOB` · `ANEKA.MST_SHARE_PU` · `TREATY_LOSS` | Rendah–Sedang |

## 2.6 Mana yang bisa dijembatani salinan berkala

Tidak semua harus menunggu API. Pembedaannya: apakah data harus mutakhir saat itu juga.

| Boleh disalin berkala | Wajib real-time |
|---|---|
| `MST_SALES` · `MST_DET_SALES` · `POOLDATA.AGENT` | `GENERAL.MST_BUKA_PROTEKSI` — procedure yang **menulis** ke sistem sumber |
| `LST_DET_CABANG` · `LST_USER_ASURANSI` | `V_KLAIM` — cek duplikat proteksi harus melihat kondisi terkini |
| `GENERAL.HRD_LBR` (kalender libur) | `GL.T_ALL_PAYMENT` — status pembayaran berubah setiap saat |
| `ANEKA.MST_SHARE_PU` · `TREATY_LOSS` | `NEW_GENERAL.M_USER_JOB` — beban kerja untuk penugasan |
| fungsi `GET_NAMA_*` → salin sebagai tabel referensi | `HRDASM.V_HRD_MST` — bila dipakai saat registrasi |

> **Catatan penting tentang `GENERAL.MST_BUKA_PROTEKSI`:** ini satu-satunya objek remote yang
> **bukan pembacaan**, melainkan **procedure yang menulis** ke tiga sistem berbeda (@ASMD,
> @SIMASNET, @SMI). Penggantinya wajib API sungguhan dan wajib **idempoten** — salinan berkala
> tidak bisa dipakai, dan kegagalan di tengah tidak boleh menghasilkan proteksi ganda.

## 2.7 Risiko tersembunyi

- **Tiga sistem, satu proses.** Open Protection memanggil procedure yang sama di @ASMD,
  @SIMASNET, dan @SMI lewat tiga rule terpisah. Bila ketiganya diganti API, kontraknya harus
  seragam — kalau tidak, satu proses bisnis akan punya tiga perilaku berbeda.
- **`GetIDCabang` adalah titik paling kritis.** Dipanggil **13 aktivitas** termasuk jalur
  registrasi klaim. Bila API pegawai belum siap, modul B-2 ikut tertahan.
- **Perhitungan TAT bergantung penuh pada sistem lain.** `GET_WORKING_HOURS` dan `HRD_LBR`
  ada di @ASMD. Tanpa keduanya, seluruh laporan TAT dan KPI tidak dapat dihitung — dan itu
  laporan yang dipakai harian.
- **Fungsi `GET_NAMA_*` tampak sepele tapi ada di query pencarian polis** yang dipakai
  di layar. Mengganti dengan join ke tabel salinan mengubah rencana eksekusi query — perlu
  diukur ulang pada data sebesar produksi.
