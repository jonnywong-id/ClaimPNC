# Domain Model — Claim PNC

Model domain sasaran. Istilah mengikuti `CONTEXT.md` tanpa perkecualian. Struktur diturunkan
dari analisis 1.381 referensi property `ClaimData.*` dan 245 tabel di sistem lama, lalu dinamai
ulang sesuai D-19.

Dokumen ini menjelaskan **konsep dan hubungannya**, bukan DDL. Skema fisik ada di
`09-DATABASE-STRATEGY.md`.

> **Diperbarui v2.0 (2026-09-14).** Yang berubah: **Status Klaim menjadi 33 kode `1134`–`1166`**
> (`R-06` tertutup) · **aggregate Komite ditulis ulang sebagai penjenjangan kumulatif** (`D-47`,
> `D-52`, `D-70`) · tiga invarian baru `I-11`…`I-13` (kurs, presisi uang, soft delete) · `I-1`
> memakai toleransi numerik yang tegas · master data dinyatakan **≥29 kelompok**, bukan 14 · dan
> nomor klaim menjadi **`PNCN.YY.xxxx`** (`D-71`).

---

## 1. Aggregate utama: Klaim

**Klaim** adalah aggregate root. Semua di bawahnya hanya boleh diubah lewat Klaim — inilah yang
menjaga invarian seperti "total spreading harus 100%" tetap benar.

```
Klaim  ◄── aggregate root
│
├─ Identitas
│    NomorKlaim            PNCN.YY.xxxx (baru) atau PNC-xxxx (warisan) — D-22, D-71
│    NomorRegisterDokumen  kaitan ke Receive Document
│
├─ SnapshotPolis  ◄── nilai beku saat registrasi (D-04)
│    NomorPolis · ProdKe · Tertanggung · PeriodePertanggungan
│    GroupPanel · BusinessType · SourceOfBusiness · Cabang
│    TotalTSI · TotalPremi · StatusPembayaranPremi
│    DaftarKoasuransi[]     ← pembagian antar perusahaan asuransi
│    DaftarFacOffer[]       ← penawaran fakultatif
│    DaftarAlamatKirim[]
│
├─ DataKerugian
│    TanggalKejadian (DOL) · TanggalLapor · TanggalTerimaDokumen
│    LokasiKejadian · Kronologi · NilaiEstimasi · MataUang
│    HubunganTertanggung · Pelapor · KontakPelapor
│    ExGratia · FlagASO · NomorSLIK
│
├─ StatusProses          alur kerja        ──┐
├─ StatusKlaim           33 kode 1134–1166   ├─ empat konsep berbeda (D-18)
├─ FlagKlaim             penanda biner       │
├─ StatusPosisiProgres   tahapan progres   ──┘
│
├─ ObjekPertanggungan[]  ◄── entity
│  │   KodeObjek · NamaObjek · LokasiObjek · LokasiSurvei
│  │   Okupasi · JenisSurveyor · KodeCabang
│  │   AtributKendaraan   (rangka, mesin, merek, tipe, model)
│  │   AtributOrang       (tanggal lahir, NIK, status peserta)
│  │   AtributKredit      (kolektibilitas, sebab macet, hari tunggakan, suku bunga)
│  │
│  ├─ Coverage[]  ◄── entity
│  │  │   KodeCoverage · NamaCoverage · TSI · SublimitTSI
│  │  │   PenyebabKerugian · Diagnosa
│  │  │
│  │  ├─ SettlementLine[]  ◄── entity  (dulu: AdjustmentList)
│  │  │  │   NilaiEstimasi → NilaiUsulan → NilaiAkseptasi → NilaiDibayar
│  │  │  │   NomorAkseptasi · TanggalAkseptasi · MataUang
│  │  │  │   NilaiSalvage · NilaiRisikoSendiri · ShareASM
│  │  │  │   StatusAkseptasi · StatusTransferKasir · StatusPembayaran
│  │  │  │   PenerimaPembayaran · JenisPembayaran
│  │  │  │   StatusCetakLOD
│  │  │  ├─ JejakKomite[]           persetujuan berjenjang
│  │  │  ├─ PemberitahuanDLA[]      DLA yang diterbitkan
│  │  │  └─ Dokumen[]
│  │  │
│  │  ├─ RincianItem[]      sparepart, biaya, rincian per item
│  │  └─ Spreading[]        pembagian risiko — TOTAL WAJIB 100%
│  │        JenisTreaty · NamaTreaty · PersentaseShare
│  │        PremiTerbagi · Penanggung
│  │
│  └─ RincianItem[]
│
├─ HasilSurvei[]           surveyor, tanggal, lokasi, temuan, foto
├─ Komite[]                daftar dan keputusan komite
├─ Dokumen[]               metadata + referensi ke storage eksternal (D-16)
├─ PenerimaGantiRugi[]     nama, alamat, jenis pembayaran
├─ OpenProtectionTerkait[] Open Protection yang dipakai klaim ini
├─ Salvage[]               barang sisa dan hasil lelang
├─ Recovery[]              pemulihan dari pihak ketiga
├─ RiwayatKomunikasi[]     korespondensi dengan tertanggung
└─ JejakAudit[]            append-only, tidak dapat diubah (D-28)
```

---

## 2. Invarian aggregate

Aturan yang **wajib benar setiap saat**. Inilah alasan Klaim menjadi aggregate root — kalau
Coverage bisa diubah langsung tanpa lewat Klaim, invarian ini tidak bisa dijamin.

| # | Invarian | Kapan diperiksa |
|---|---|---|
| I-1 | Total Spreading setiap Coverage = 100%, diuji **`ROUND(SUM(share),4) BETWEEN 99.9999 AND 100.0001`** (`D-51`) | Saat registrasi dan setiap perubahan spreading |
| I-2 | `TanggalKejadian ≤ TanggalLapor ≤ TanggalTerimaDokumen ≤ hari ini` | Saat registrasi dan setiap perubahan tanggal |
| I-3 | `TanggalKejadian` berada dalam periode polis (+30 hari Bonding, +90 hari Travel/PA) | Saat registrasi |
| I-4 | Nilai Settlement tidak melebihi TSI Coverage | Setiap perubahan nilai |
| I-5 | Akseptasi hanya sah bila seluruh jenjang komite yang diwajibkan sudah selesai | Saat akseptasi |
| I-6 | Objek tanpa Coverage tidak boleh tersimpan | Saat registrasi |
| I-7 | Setiap perubahan nilai bisnis menghasilkan satu baris JejakAudit | Setiap perubahan |
| I-8 | Tidak ada klaim ganda: Polis + Objek + Lokasi (PA: + Penyebab Kerugian `12002`) | Saat registrasi |
| I-9 | Nomor SLIK terisi bila lini bisnis SPK / Asuransi Kredit | Saat registrasi |
| I-10 | Penyebab Kerugian terisi, kecuali lini Travel | Saat registrasi |
| **I-11** | Klaim valuta asing hanya sah bila **kurs pada tanggal kejadian** tersedia; bila tidak, klaim **ditolak** — tidak ada nilai bawaan (`D-48`) | Saat registrasi dan setiap perubahan nilai |
| **I-12** | Nilai uang disimpan **presisi penuh**; pembulatan hanya saat ditampilkan, dan perbandingan terhadap ambang memakai nilai presisi penuh (`D-51`) | Setiap perhitungan |
| **I-13** | **Tidak ada penghapusan fisik** data bernilai bisnis; penghapusan dinyatakan lewat penanda (`D-66`) | Setiap penghapusan |

> **I-1 dan I-11 menutup cacat yang berjalan hari ini.** Toleransi spreading sistem lama adalah
> **pencocokan substring**, bukan perbandingan numerik — total `199.99` dan `1100.0` ikut lolos.
> Dan fungsi kurs mengembalikan `1` ketika kurs tidak ditemukan, sehingga klaim bernilai besar
> menyusut menjadi kecil lalu **lolos tanpa komite**. Keduanya masuk daftar 13 perbaikan eksplisit
> `P-5` (`D-49`, `ADR-0017`).

---

## 3. Empat konsep status

Dikonfirmasi memang berbeda (D-18). Di sistem lama keempatnya bernama mirip dan tersimpan
berdampingan sehingga sering tertukar. Di sistem baru namanya dibuat **tidak mungkin tertukar**.

| Konsep | Nama lama | Nilai | Menjawab pertanyaan |
|---|---|---|---|
| **Status Proses** | `StatusWork` | `Berjalan`, `Selesai`, `Ditolak` | Klaim ini masih dikerjakan atau sudah tuntas? |
| **Status Klaim** | `StatusClaim` | **33 kode, `1134`–`1166`** | Klaim ini berada di keadaan bisnis apa? |
| **Flag Klaim** | `ClaimStatus` | `0` / `1` | *(makna persisnya masih perlu dikonfirmasi)* |
| **Status Posisi Progres** | `StatusPosisi` | `On Progress`, `Done` | Tahapan progres mana yang sudah dilalui? |

**`R-06` tertutup.** Isi master diterima (`v_sts_claim.csv`) dan **memperbaiki premisnya sendiri**:
domain `StatusClaim` bukan 10 kode `1142`–`1151`, melainkan **33 kode `1134`–`1166`**. Sebelas
kode pertama (`1134`–`1144`) membawa penomoran lama `01`–`11`; sisanya hanya bernomor baru.

Contoh: `1142` Rejected Claim · `1143` Close Claim for this object · `1144` Cancelled Claim ·
`1147` Register · `1149` Claim Committee · `1163` Paid · `1164` Reopen Claim.

> **Rentang `1142`–`1151` yang dipakai dokumen-dokumen awal ternyata sebagian dari domainnya,
> bukan keseluruhannya.** Tiga arti kode yang sempat disimpulkan dari pemakaiannya di rule
> **seluruhnya salah** — `1143` bukan status awal melainkan Close Claim, `1150` bukan penanda
> terdaftar melainkan LOD Report, `1151` bukan Investigator melainkan Analyst. Arti kode status
> tidak boleh disimpulkan dari pemakaian.

**Dua catatan yang membatasi dua dari empat konsep:**

- **`ClaimStatus` milik ruleset GISFW**, bukan Claim PNC — salah satu dari "empat konsep" berada
  di bounded context tim lain, dan hubungannya dengan snapshot polis (`D-04`) belum dirumuskan.
- **`StatusPosisi` hanya pernah muncul dengan satu nilai** (`'On Progress'`) pada 21 titik panggil
  di export. Kueri `SELECT DISTINCT STATUSPOSISI` pada produksi masih ada di daftar permintaan DBA
  untuk menyelesaikannya secara empiris. Pemilik bisnis menegaskan keempat konsep memang berbeda,
  sehingga keempatnya tetap dipertahankan (`D-18`, `ADR-0018`).

---

## 4. Aggregate lain

### Komite
Aggregate tersendiri karena punya siklus hidup sendiri dan **berjenjang secara kumulatif**.

```
Komite
├─ NomorKomite · KlaimTerkait · JenisKomite
├─ JenjangDibutuhkan[]  ← seluruh jenjang yang ambang bawahnya sudah terlampaui (D-47)
├─ JenjangTerpenuhi     ← pencacah persetujuan yang sudah masuk
├─ AnggotaKomite[]      nama, peran, urutan (DEGREE)
├─ Keputusan[]          setuju / tolak / kembalikan · catatan · waktu · oleh siapa
└─ StatusPersetujuan
```

**Invarian:** klaim tidak boleh diakseptasi sebelum **seluruh jenjang yang dibutuhkan** menyetujui.

**Cara jumlah jenjang dihitung** (`D-47`, `D-52`, `D-70` — `ADR-0014`):

1. **Untuk lini Non-MBU saja**, pita nilai dipilih lebih dulu: sampai **Rp 100.000.000** → pita
   `1`; di atasnya → pita `2`. Akumulasi kemudian berjalan **di dalam pita itu saja**.
2. **Akumulasi menurut ambang bawah.** Setiap jenjang yang **ambang bawahnya sudah terlampaui**
   nilai klaim ikut menyetujui — bukan memilih satu jenjang tunggal. Jumlah penyetuju **bertambah**
   seiring besarnya klaim.

> **Ini bukan "matriks nilai × jenis bisnis" yang memilih satu baris.** Di sistem lama, jumlah
> jenjang = **jumlah baris yang dikembalikan kueri**
> (`Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459`), dan kueri itu menyaring **hanya
> dengan batas bawah**. `LIMIT_BOTTOM` muncul di 11 SQL rule; **`LIMIT_TOP` muncul di 0**.
> Menerapkan rentang tertutup akan mengembalikan tepat satu baris — **satu jenjang persetujuan
> berapa pun nilai klaim** — dan menghapus penjenjangan yang menjadi inti `D-14`.

**`LIMIT_TOP` tetap dipakai, tetapi sebagai validasi integritas master**: sistem baru menolak
master yang rentangnya tumpang tindih atau berlubang antar `DEGREE` dalam satu `TYPE_BUSINESS` +
`TYPE_KOMITE`. Karena tangga PA tidak tersusun menurut pita, validasi itu **hanya dapat dijalankan
per lini**, bukan lintas lini.

**Peringatan model data:** kolom `TYPE_KOMITE` **memikul dua arti berbeda** — pita nilai di
Non-MBU, dan **varian jalur** (PA reguler versus PA TKI) di lini PA. Pemisahannya menjadi dua kolom
dipertimbangkan dan **tidak diambil**, sehingga arti gandanya wajib didokumentasikan di master
baru.

**Catatan penugasan:** penjenjangan komite menugaskan ke **operator bernama**, bukan ke workbasket
— seluruh export hanya memuat **4 workbasket**. Model penugasan komite karena itu tidak mengikuti
pola Worklist/Workbasket pada aggregate Penugasan di bawah.

### Open Protection
```
OpenProtection
├─ NomorProteksi · NomorPolis · JenisProteksi
├─ TanggalInput · Pemohon
├─ StatusAkseptasi
└─ SudahDipakaiKlaim   ← ditandai saat ditautkan ke klaim
```

### Receive Document
```
ReceiveDocument
├─ NomorRegisterDokumen (RCV_ID)
├─ CabangPengirim · Ekspedisi · NomorResi
├─ TanggalKirim · EstimasiTiba · TanggalTerima
├─ JumlahLembar · JenisDokumen
└─ KlaimTerkait
```

### Penugasan
```
Penugasan
├─ KlaimTerkait · Tahap · JenisPenugasan (Worklist | Workbasket)   ← D-26
├─ DitugaskanKe        diisi bila Worklist
├─ Workbasket          diisi bila Workbasket
├─ DiambilOleh · WaktuDiambil    penguncian antar-pengguna
└─ StatusPenugasan
```

---

## 5. Master data

Semua yang berikut adalah **data yang dapat diubah tanpa deploy** (D-15). Di sistem lama
sebagian di-hardcode di dalam activity.

> **Koreksi jumlah (2026-09-14).** Verifikasi terhadap export menemukan **≥29 kelompok master**,
> bukan 14. Daftar di bawah adalah **14 kelompok yang sudah terpetakan**; sisanya diinventarisasi
> saat tiket `F-4` ditulis. Akibatnya `F-4` naik menjadi **Besar** dan `U-6` dari Sedang menjadi
> **Besar**.

| Master | Isi | Menggantikan hardcode |
|---|---|---|
| Lini Bisnis | Group Panel, Business Type | — |
| Penyebab Kerugian | kode, deskripsi, kaitan lini bisnis | — |
| Cabang & Wilayah | kode, nama, kanwil | — |
| Surveyor & Adjuster | internal/eksternal, spesialisasi | — |
| Jenis Treaty | OR, ORS, Fac Out, XOL, BPPDAN | — |
| **Mata Uang & Kurs** | kode, **kurs per tanggal** — konversi memakai kurs **tanggal kejadian**; bila tidak ada, klaim **ditolak** tanpa nilai bawaan (`D-48`) | ✅ `GETCURRENCYSTANDARD` yang mengembalikan `1` saat kurs tidak ditemukan |
| Bank & Rekening | untuk pembayaran | — |
| Jenis Dokumen | per lini bisnis, wajib/opsional | — |
| **Ambang Komite** | **tangga ambang bawah per lini** (`TYPE_BUSINESS` + `TYPE_KOMITE` + `LIMIT_BOTTOM` + `DEGREE`), diakumulasi (`D-47`, `D-70`) | ✅ **8 ambang komite unik**, antara lain `50000000`, `30000000`, `20000000`, `7000`, `3500` |
| **Penerima Notifikasi** | per jenis peristiwa dan Group Panel — **mailbox fungsional saja, tidak ada akun pribadi** (`D-67`) | ✅ **66 alamat email**, termasuk ≥6 akun Gmail pribadi di jalur produksi |
| **Ambang Large Losses** | nilai pemicu Notice of Large Losses | ✅ `1000000000` |
| **Batas Aturan Tanggal** | 7 hari, 30 hari, 90 hari per lini bisnis | ✅ tertanam di validasi |
| Peran & Izin Menu | **22 peran** → 51 item menu (`D-58`, `D-59`) | ✅ 22 access group + **24 Operator ID hardcode** |
| Master Status Klaim | **33 kode `1134`–`1166`** dan artinya | — |
| **Hari Libur & Jam Kerja** | kalender libur dan definisi jam kerja untuk perhitungan TAT (`D-50`) | ✅ `DATAMINING.GET_WORKING_HOURS@ASMD` dan `HRD_LBR` lewat DB Link |
| **Retensi Data** | lama simpan data klaim **dan** jejak audit — satu kebijakan (`D-62`) | ✅ tidak ada; belum pernah ditetapkan |

---

## 6. Peristiwa domain

Peristiwa yang memicu akibat lain. Dipakai sebagai interface seam Notifier (bagian 3.6 Future
Architecture) — pemanggil menyatakan **apa yang terjadi**, bukan **siapa yang harus diberi tahu**.

| Peristiwa | Akibat |
|---|---|
| `KlaimDiregistrasi` | Notifikasi register · penautan Open Protection · pencatatan progres |
| `EstimasiMelebihiAmbangBesar` | Notice of Large Losses ke UW sesuai Group Panel dan pimpinan |
| `PremiBelumLunas` | Notifikasi premi tertunggak |
| `KlaimDiestimasi` | Penerbitan **PLA** ke koasuransi/reasuransi |
| `NilaiDiusulkan` | Penerbitan **Pre-DLA** |
| `KomiteMenyetujui` | Lanjut ke jenjang berikutnya, atau buka akseptasi |
| `KomiteMenolak` | Klaim masuk jalur **RCL** |
| `KlaimDiakseptasi` | Penerbitan **DLA** dan **LOD** ke tertanggung |
| `KlaimDitransferKeKasir` | Permintaan pembayaran |
| `KlaimDibayar` | Penutupan klaim · pencatatan progres |
| `KlaimDitolak` (RCL) | Surat penolakan · penutupan |
| `KlaimDiprosesUlang` (PUCL) | Pembukaan kembali klaim yang sudah ditutup |

---

## 7. Pemetaan ke sistem lama

Rujukan saat implementasi, agar penelusuran ke source Pega tetap mungkin.

| Konsep baru | Kelas Pega | Tabel Oracle |
|---|---|---|
| Klaim | `ASM-FW-GCNMFW-Work-PNC` | `POOLDATA.T_CLAIM_PNC` + `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` |
| DataKerugian | `ASM-FW-GCNMFW-Data-ClaimData` | kolom di `T_CLAIM_PNC` |
| SnapshotPolis | `ASM-FW-GISFW-Data-Policy` | `POOLDATA.JSON_POLIS.DATA_JSONBLOB` |
| ObjekPertanggungan | `ASM-FW-GCNMFW-Data-Object` | `POOLDATA.T_CLAIM_OBJECTLIST` |
| Coverage | `ASM-FW-GCNMFW-Data-ObjectCoverage` | `POOLDATA.T_CLAIM_OBJECTCOVERAGE` |
| SettlementLine | `ASM-FW-GCNMFW-Data-Adjustment` | `POOLDATA.T_CLAIM_ADJUSTMENT` |
| Spreading | `ASM-FW-GISFW-Data-SpreadingRisk` | tertanam di JSON dan tabel reas |
| Komite | `ASM-FW-GCNMFW-Work-Komite` | `POOLDATA.T_CLAIM_KOMITE_LIST` |
| OpenProtection | `ASM-FW-GCNMFW-Work-OpenProtection` | `POOLDATA.MST_PROTEKSI_DATA_PNC` |
| ReceiveDocument | `ASM-FW-GCNMFW-Work-ReceiveDocument` | tabel receive |
| HasilSurvei | `ASM-FW-GCNMFW-Data-Survey` | `POOLDATA.T_SURVEYORLIST`, `T_DOC_SURVEY` |
| Dokumen | `Data-WorkAttach-File` | `POOLDATA.DATA_ATTACHFILE`, `GENERAL.T_STORAGE_IMAGE` |
| Penugasan | `Assign-Worklist` / `Assign-Workbasket` | `DATAPEGA.PC_ASSIGN_*` |
| StatusPosisiProgres | — | `POOLDATA.GCNM_PROGRESS_CLAIM` + `GCNM_MST_PROGRESS_KLAIM` |
| Salvage | — | `POOLDATA.PNC_SALVAGE`, `DETAIL_PNC_SALVAGE` |
| PLA / DLA | `ASM-FW-GCNMFW-Data-PLA` / `-DLA` | `POOLDATA.T_PLALIST`, `T_DLALIST`, `T_PREDLALIST` |
