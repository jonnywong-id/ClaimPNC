# Lampiran — Status 40 Usulan Revisi (`D-39`)

| | |
|---|---|
| **Tanggal sapuan** | 2026-09-15 |
| **Sumber daftar** | `docs/verifikasi-bukti-adr.md` §11 — 40 kontradiksi `K-1`…`K-40` |
| **Dasar** | `D-39` — seluruh angka mengikuti hasil verifikasi bukti, selisihnya jadi usulan revisi |
| **Status lampiran** | **hasil pemeriksaan, bukan perubahan** — belum ada yang diterapkan dari sapuan ini |

## Kenapa lampiran ini ada

`D-39` memutuskan setiap selisih antara dokumen dan bukti dicatat sebagai usulan revisi. Daftarnya
ada di `verifikasi-bukti-adr.md` §11 — tetapi **tanpa penanda status sama sekali**. Akibatnya tidak
ada cara mengetahui mana yang sudah diterapkan dan mana yang masih menggantung, selain memeriksa
satu per satu ke dokumennya. Itulah yang dikerjakan sapuan ini.

## Ringkasan

| Status | Jumlah | Artinya |
|---|---:|---|
| **Sudah diterapkan** | **10** | Dokumen sudah memuat angka/fakta terverifikasi |
| **Diterapkan pada 2026-09-15** | **10** | Disetujui Work Owner; `K-13` sengaja ditahan |
| **Ditahan** | **1** | `K-13` — mengubah aturan penolakan klaim ganda, menunggu konfirmasi tim bisnis |
| **BERTENTANGAN** | **1** | `K-6` — dua rekaman verifikasi tidak sepakat |
| **Tidak perlu tindakan** | **2** | Bukti justru mendukung klaim dokumen |
| **Perlu keputusan, bukan koreksi teks** | **11** | Menyangkut keputusan bisnis/arsitektur, bukan salah ketik angka |
| **Belum tuntas diverifikasi** | **5** | Perlu pembacaan rule lebih dalam; **tidak saya nyatakan bersih** |

## 1. Masih menggantung — 11 butir

Inilah yang dapat langsung diperbaiki bila disetujui. Semuanya sudah diverifikasi ke berkasnya.

| # | Klaim dokumen sekarang | Terverifikasi | Lokasi |
|---|---|---|---|
| `K-1` | 45 activity hilang | **40** (5 positif palsu, bawaan Pega) | `BRD.md:446`, `BRD.md:1289` |
| `K-3` | workbasket `komitepnc1..4` | **`komitepnc`** tanpa angka, dan **worklist**, bukan workbasket | `19-GAP-EXPORT-DETAIL.md:177` |
| `K-8` | 10 email · 4 user ID · 3 ambang | **66 · 24 · 8** | `BRD.md:267` |
| `K-10` | `InputRegister_act` step 103 dan 107 | **sub-step 2 dan 6 di dalam step 53** | `BRD.md:268` |
| `K-13` | kunci duplikasi = Polis + Objek + Lokasi | **juga DOL**; lokasi hanya untuk non-PA/non-Travel | `BRD.md:646` |
| `K-18` | ~20 inbox berbasis peran | **26** harness bernama *Inbox* | `06-MODULE-BREAKDOWN.md:85`, `BRD.md:526`, `BRD.md:1136` |
| `K-26` | 27 modul | **34** (32 enumerasi + `FR-S8` pada `D-42` + `FR-F6` pada `D-75`) | `BRD.md:90`, `BRD.md:1355` |
| `K-30` | 245 tabel di 5+ skema | **177** objek di **9 skema** | `BRD.md:82` |
| `K-31` | `OFFSET 500000` membaca setengah juta baris | **`OFFSET` nol kemunculan** di seluruh export | `requirement-summary.md:168` |
| `K-33` | 47 menu portal | **47 harness target** ✔ tetapi **51 item menu**; 11 dari 47 harness tidak ada di export | `requirement-summary.md:121`, `02-BUSINESS-UNDERSTANDING.md:179` |
| `K-37` | `F-5` berukuran **Kecil** | "Kecil" hanya benar untuk seam Clock — migrasinya **118 titik +7 jam di 36 activity** ditambah 101 titik +12 jam | `06-MODULE-BREAKDOWN.md:28` |

> **`K-31` bukan sekadar angka salah.** `OFFSET 500000` dipakai sebagai **dasar** `NFR-13`. Karena
> `OFFSET` tidak pernah muncul sekali pun di export, requirement itu berdiri di atas premis yang
> tidak ada — dan perlu **ditulis ulang**, bukan diperbaiki angkanya.

## 2. Sudah diterapkan — 11 butir

`K-2` · `K-9` · `K-11` · `K-12` · `K-19` · `K-24` · `K-25` · `K-27` · `K-32` · `K-36`

Contoh yang diperiksa langsung: `K-12` (toleransi spreading kini `ROUND(SUM(share),4) BETWEEN
99.9999 AND 100.0001` per `D-51`, menggantikan pencocokan substring) · `K-9` (`03-CURRENT-ARCHITECTURE.md:182`
sudah memuat "Koreksi daftar hostname") · `K-32` dan `K-36` (koreksi ukuran `U-2`, `F-4`, `U-6`
sudah tercatat di Module Breakdown).

## 3. Tidak perlu tindakan — 2 butir

| # | Sebabnya |
|---|---|
| `K-39` | **Cocok persis** — 68 pemakaian `ROWNUM` di 44 berkas; angka dokumen benar |
| `K-40` | Klaim didukung angka — `pyPrivilegeName` non-kosong **1 dari 902** activity |

## 4. Perlu keputusan, bukan koreksi teks — 11 butir

Kesebelas ini **tidak bisa diselesaikan dengan menyunting angka**. Masing-masing menyangkut
keputusan yang pemiliknya bukan saya.

| # | Pokok persoalan | Pemilik |
|---|---|---|
| `K-5` | Lingkup audit export: **7 tipe rule belum diaudit**; gap ±242 versus 43 yang tercatat | **Work Owner + Tim Pega** |
| `K-7` | `D-18` mempertahankan empat konsep status, tetapi `ClaimStatus` milik **GISFW** dan `StatusPosisi` bernilai tunggal | **Work Owner** — perlu keputusan baru yang menyebut `D-18` |
| `K-20` | Mekanisme penyimpanan dokumen ternyata **empat**, bukan tiga — GCS yang keempat, rule-nya hilang | **Tim Pega + Work Owner** |
| `K-21` | HCC/HCQ **nol jejak di export** → ini **integrasi baru**, bukan migrasi; gerbang 1 tidak berlaku bagi `F-3` | **Work Owner** |
| `K-22` | `D-15` konfigurasi tiga lapis: **nol DSS**; pola lama terkunci per IP dan bertabrakan dengan `D-27` | **Lead Engineer + Work Owner** |
| `K-23` | `D-26` mempertahankan model penugasan, padahal penguncian **nol kustomisasi** → sistem baru bebas memilih | **Work Owner** |
| `K-28` | **Jalur validasi API/JSON lebih longgar** daripada jalur layar — aturan 7/30/90 tidak berlaku di sana | **Work Owner** |
| `K-29` | Perangkaian SQL dari nilai pengguna — melanggar `BRD §21.2` #11, **skalanya jauh lebih besar** (538 `{ASIS:}`) | **Work Owner + Keamanan Informasi** |
| `K-34` | Paginasi sistem lama praktis **client-side** (3.189 grid page list) → `NFR-12`/`NFR-13` adalah **perubahan perilaku**, bukan penyalinan | **Work Owner** |
| `K-35` | Matriks jenjang komite **sudah berupa master data**, bukan hardcode yang perlu dipindah | **Work Owner** |
| `K-38` | `D-20` mengganti `ROWNUM` → `OFFSET`; angkanya benar tetapi **mayoritas `FETCH NEXT 1 ROW ONLY`** — bukan paginasi | **Lead Engineer** |

## 5. Belum tuntas diverifikasi — 5 butir

Saya **tidak menyatakan kelimanya bersih**. Memastikannya menuntut pembacaan rule lebih dalam
daripada yang dilakukan sapuan ini.

`K-4` (router `ReceiveDocument.UserAdmin` — properti operator, bukan workbasket) ·
`K-14` (kunci duplikasi PA di SQL) · `K-15` (Group Panel `003` hanya baris (1)(1)) ·
`K-16` (ambang PA/Travel dipicu **jabatan operator**, bukan lini bisnis) ·
`K-17` (nilai klaim ≤ TSI, **PA dikecualikan** — belum ada di BRD)

> `K-16` dan `K-17` menyangkut **aturan yang menentukan uang**. Keduanya layak diprioritaskan di
> atas koreksi angka mana pun di bagian 1.

## Yang tidak saya sentuh, dan alasannya

| Tempat | Alasan |
|---|---|
| Entri lama `00-DECISION-LOG.md` (`D-15`, `D-18`, `D-20`, `D-26`) | Aturan proyek melarang menyunting entri lama. Perubahan pikiran ditulis sebagai keputusan baru yang menyebut ID yang disupersede |
| `verifikasi-bukti-adr.md` §11 | Ia **rekaman temuan**. Statusnya dilacak di lampiran ini, bukan dengan mengubah temuannya |
| Tabel ringkasan v1.0 → v2.0 di `BRD.md:34` | Isinya memang pasangan **angka lama → angka baru**. Menggantinya akan menghapus jejak koreksinya sendiri |

---

## 6. Pembaruan 2026-09-15 — sepuluh diterapkan, satu ditahan, satu bertentangan

Work Owner menyetujui penerapan. Hasilnya:

### Sepuluh diterapkan

`K-1` · `K-3` · `K-8` · `K-10` · `K-18` · `K-26` · `K-30` · `K-31` · `K-33` · `K-37`

Berkas yang disunting: `BRD.md` (10 baris) · `06-MODULE-BREAKDOWN.md` (2) ·
`19-GAP-EXPORT-DETAIL.md` (1) · `02-BUSINESS-UNDERSTANDING.md` (1) · `requirement-summary.md` (3).

### `K-13` ditahan atas saran saya

Ia **bukan koreksi angka** melainkan **aturan penolakan klaim ganda**: kuncinya ternyata menyertakan
**DOL**, dan lokasi hanya berlaku untuk non-PA/non-Travel. Mengubahnya mengubah klaim mana yang
ditolak sebagai duplikat — layak dikonfirmasi ke tim bisnis lebih dulu.

### `K-6` BERTENTANGAN — tidak diterapkan

Dua rekaman verifikasi **tidak sepakat** tentang jumlah pemakaian DB Link:

| Sumber | Angka |
|---|---|
| `verifikasi-bukti-adr.md` §11 `K-6` | **71 pemakaian · 22 objek `@ASMD`** |
| `16-RISK-ANALYSIS.md:115` | **64 pemakaian · 6 DB Link · 28 objek remote · 27 rule** — ditulis sebagai *"angka terverifikasi"* |

Keduanya mengklaim hasil verifikasi, dan keduanya tidak dapat benar bersamaan. Saya sempat
menerapkan angka `K-6` ke tiga berkas, lalu **membatalkannya** begitu pertentangan ini terlihat —
karena memilih salah satu berarti menegaskan angka yang belum tentu benar.

**Perlu dihitung ulang langsung dari export**, bukan dipilih dari salah satu dokumen.
Pemilik: **Lead Engineer** (perhitungan) lalu **Work Owner** (penetapan).

### Tiga verdict "sudah diterapkan" saya ternyata salah

Ketiganya salah karena sebab yang sama: **saya memeriksa BRD saja, bukan seluruh dokumen.**

| # | Yang terlewat |
|---|---|
| `K-12` | Toleransi spreading `99,99%` masih tertulis di **enam berkas** — `02-BUSINESS-UNDERSTANDING`, `04-FUTURE-ARCHITECTURE`, `09-DATABASE-STRATEGY`, `14-TESTING-STRATEGY`, `CONTEXT.md`, `requirement-summary`. Sudah diperbaiki ke **4 desimal `99,9999`–`100,0001`** (`D-51`) |
| `K-6` | Lihat di atas — bukan hanya terlewat, tetapi **bertentangan** |
| `K-32` | Angka `18–27 kolom` masih ada di badan `01-FRONTEND-ANALYSIS.md`. **Dibiarkan dengan sengaja**: dokumen itu punya catatan koreksi di kepalanya, dan badannya adalah analisis historis yang menghasilkan keputusan — diperlakukan sama seperti entri Decision Log lama |

Ditemukan pula **dua berkas yang terlewat dari koreksi Connect REST kemarin** (`D-73`):
`03-CURRENT-ARCHITECTURE.md:26` dan `04-FUTURE-ARCHITECTURE.md:136` masih menulis
`12 Connect-REST`. Keduanya sudah diperbaiki menjadi **21**.

---

## 7. Hasil verifikasi lanjutan 2026-09-15 — `K-6`, `K-16`, `K-17`

Diminta Work Owner. Dihitung dan dibaca langsung dari export, bukan dari dokumen turunan.

### `K-6` — kedua angka yang bertentangan **sama-sama tidak cocok**

Hitungan langsung atas seluruh export (`*.xml`, `*.prc`, `*.fnc`), pola `@ASMD|@SIMASNET|@SMI|@OPJAVA|@PROD_ASM|@PROD_TKA`:

| Satuan hitung | Hasil |
|---|---:|
| Kemunculan mentah | **203** |
| Berkas/rule yang memakainya | **37** |
| Objek remote unik (`SKEMA.OBJEK@LINK`) | **49** |
| — di antaranya `@ASMD` | **34** |

| Angka dokumen | Cocok? |
|---|---|
| `16-RISK-ANALYSIS.md:115` — 64 pemakaian · 28 objek remote | **tidak** |
| `verifikasi-bukti-adr` `K-6` — 71 pemakaian · 22 objek | **tidak** |

**Kenapa ketiganya berbeda: satuan hitungnya berbeda, dan dua berkas mendominasi.**

| Berkas | Kemunculan |
|---|---:|
| `RDB List/GetTotalKlaimCreditValue_NGPW-SQL.xml` | **68** (seluruhnya `@SIMASNET`, 9 objek unik) |
| `Database/CONVERTJSONPRODUCTION.prc` | **51** |
| 35 berkas lainnya | 84 |

Menghitung **hanya `RDB List` tanpa berkas pencilan pertama** menghasilkan **66** — sangat dekat
dengan angka 64 yang tercatat. Dugaan terkuat: angka lama dihitung **sebelum export bertambah**
dari 2.167 menjadi 2.634 berkas, dan tidak pernah dihitung ulang.

**Dua akibat yang lebih penting daripada angkanya sendiri:**

1. **Inventaris `D-25` salah besar untuk `@SIMASNET`.** Tabel `D-25` mencatatnya **3×**; hitungan
   sebenarnya **71**, dan **68 di antaranya ada di satu rule** — `GetTotalKlaimCreditValue_NGPW`,
   sebuah kueri lintas database ke **9 objek remote**. `@SIMASNET` dicatat sebagai link kecil,
   padahal ia yang terberat kedua.
2. **`DATAMINING.GET_KURS_STANDARD@SIMASNET` adalah objek remote.** Kurs standar — yang menentukan
   konversi nilai klaim — diambil lintas database. Ini menyambung langsung ke `R-19` dan
   `TKT-F4-004` (isi `m_currencystandard` belum ada dari DBA).

**Status `K-6`: tetap BERTENTANGAN, kini dengan angka ketiga.** Penetapannya milik **Work Owner**,
setelah menyepakati **satuan hitung** — "pemakaian" bisa berarti kemunculan, rule, atau objek unik,
dan ketiganya menghasilkan angka yang jauh berbeda.

### `K-16` — terkonfirmasi sebagian, dan yang ditemukan lebih berat

`Activity/GetKomiteApproval-Act.xml` membandingkan:

| Properti | Dibandingkan dengan | Jumlah |
|---|---|---:|
| `pyWorkPage.Policy.Quotation.BusinessType` | `"PA"` · `"Travel"` · `"Bonding"` | 3 |
| `TempRDBSearchEmailKomite.pxResults(1).BUSINESS_CODE` | — | 3 |
| `tempAdj.ConvertAdjustmentValue` | `30000000` dan ambang berpindah menurut hostname | 2 |
| `OperatorID.pyUserIdentifier` | **nama satu orang, tertanam di rule** | 1 |
| `pyWorkPage.ClaimData.UserTeknis` | idem | 1 |

**Klaim `K-16` — "dipicu jabatan operator, bukan lini bisnis" — tidak sepenuhnya tepat: keduanya
dipakai.** Lini bisnis memang dibandingkan (3×).

**Yang jauh lebih berat, dan tidak tercatat di mana pun:** persetujuan komite bercabang pada
**identitas satu orang tertentu** yang namanya **tertanam di dalam rule**, bukan pada peran maupun
jabatan. Bila orang itu berpindah tugas atau keluar, perilaku komite berubah — dan tidak ada
dokumen yang menjelaskan mengapa. Penanganannya termasuk dalam penghapusan hardcode (`66 email ·
24 user ID · 8 ambang`), tetapi **akibatnya pada aturan komite belum pernah dibahas**.

Ambang `Rp 50.000.000` juga terbukti berpindah menjadi **3.500** lewat perbandingan hostname —
bukti langsung untuk `D-75`/`ADR-0030`. *(Nilai hostname-nya sengaja tidak disalin ke dokumen ini.)*

### `K-17` — **tidak terkonfirmasi**

`Activity/ValidasiSisaTSI-Act.xml` dibaca seluruhnya pada bagian pembandingnya. Aturannya:

```
sisa TSI = local.sumTSI − local.NilaiAkseptasiKlaim (+ local.NilaiSalvage)
tolak bila nilai adjustment melebihi sisa itu
```

**Tidak ada satu pun perbandingan terhadap `"PA"` di dalam rule itu.** Pemanggilnya
(`SetNilaiResikoSendiri-Act.xml`) hanya dijaga `.ConfirmationAnswer`, juga tanpa cabang PA.

Jadi klaim *"PA dikecualikan"* **tidak terbukti dari isi rule**. Bila pengecualian itu nyata, ia
terjadi karena **klaim PA tidak pernah sampai ke jalur ini** — dan itu pertanyaan **alur**, bukan
isi rule. Memastikannya menuntut penelusuran `Flow` dan pemanggil-pemanggilnya.

**Status `K-17`: turun dari "belum tuntas diverifikasi" menjadi TIDAK TERBUKTI pada tingkat rule;
sisa pemeriksaannya ada di tingkat alur.**

### Yang masih belum diverifikasi — 3 butir

`K-4` (router `ReceiveDocument.UserAdmin` — properti operator, bukan workbasket) ·
`K-14` (kunci duplikasi PA di SQL: `policyno` + `objectid` + `coverageid='10009'`, tanpa lokasi dan
tanpa cause of loss) · `K-15` (Group Panel `003` — hanya baris `(1)(1)` yang diperiksa).

Ketiganya menuntut pembacaan rule yang lebih dalam daripada pemeriksaan pembanding. **Tidak satu
pun saya nyatakan bersih.**
