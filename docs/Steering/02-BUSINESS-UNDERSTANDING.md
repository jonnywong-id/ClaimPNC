# Business Understanding — Claim PNC

Seluruh isi dokumen ini diturunkan langsung dari source aplikasi Pega (902 activity, 652 rule
SQL, 70 when rule, 4 flow), dikonfirmasi oleh pemilik bisnis pada 2026-09-07. Istilah mengikuti
`CONTEXT.md`.

---

## 1. Aplikasi ini menangani apa

**Claim PNC** adalah sistem penanganan klaim asuransi umum untuk lini bisnis **Non-Motor
(Non-MBU)** di Asuransi Sinar Mas. Perjalanannya dari laporan kerugian oleh tertanggung sampai
pembayaran ganti rugi dan pemberitahuan ke koasuransi/reasuransi.

Lini bisnis yang ditangani, dikenali lewat **Group Panel**:

| Group Panel | Lini | Karakter penanganan |
|---|---|---|
| `002` | Personal Accident (PA) | Perlu penilaian medis; ada peran Analyst Doctor dan RCL Dokter |
| `003` · `009` | Aneka | Paling beragam; Fac Out sering dipakai |
| `004` | Marine Cargo | Terkait pengangkutan, rute, dan kemasan |
| `005` | Travel | Batas waktu lapor paling longgar (90 hari) |
| `006` | Fire / Property | Terkait lokasi risiko dan okupasi bangunan |

Lini khusus lain yang muncul di kode: **TKA** (Tenaga Kerja Asing), **SPK** (Sinarmas Penjaminan
Kredit / Asuransi Kredit — wajib Nomor SLIK), **Bonding**, **HE** (Heavy Equipment), dan
**Contractors PM**.

---

## 2. Proses bisnis inti

### 2.1 Alur utama — Register Flow

Diturunkan dari `Register_Flow` (23 shape, 30 konektor, 13 assignment, 7 decision).

```
                          [Mulai]
                             │
                      ┌──────▼──────┐
                      │  View Polis │  ambil & kunci snapshot polis
                      └──────┬──────┘
                             │
                    ┌────────▼────────┐
                    │ Input Register  │  ← gerbang validasi terberat
                    └────────┬────────┘
                             │
                        ◇ Kembali? ──ya──► kembali ke View Polis
                             │tidak
                        ◇ Apakah PA?
                   ya ───────┴─────── tidak
                    │                   │
             ┌──────▼──────┐       ◇ Apakah Travel?
             │ Estimation  │      ya ───┴─── tidak
             │    (PA)     │       │           │
             └──────┬──────┘  ┌────▼────┐ ┌────▼─────┐
                    │         │ Input   │ │ Input    │
                ◇ Kembali?    │Estimasi │ │Estimasi  │
                    │         │(Travel) │ │(Non-MBU) │
             ┌──────▼──────┐  └────┬────┘ └────┬─────┘
             │ Investigator│       │           │
             │ (workbasket)│   ◇ Kembali?  ◇ Kembali?
             └──────┬──────┘       │           │
                    │        ┌─────▼─────┐┌────▼──────────┐
                    │        │Send To PIC││Choose Surveyor│
                    │        │  Teknik   ││               │
                    │        └─────┬─────┘└────┬──────────┘
                    │              │           │
             ┌──────▼──────────────▼───┐       └──► [Selesai]
             │     Send To Analis      │
             └────────────┬────────────┘
                          │
        ┌─────────────────▼──────────────────┐
        │  Percabangan: Compliance / PUCL /  │
        │  Analyst Doctor / RCL Dokter       │
        └──┬──────┬──────────┬───────────┬───┘
           │      │          │           │
    Compliance  RCL/PUCL  Analyst    RCL Dokter
    (basket)    (basket)  Doctor
           │      │          │           │
           └──────┴──────────┴───────────┴──► [Selesai]
```

**Delapan ticket** memungkinkan lompatan langsung ke tahap tertentu tanpa melewati urutan di
atas: `setToRegister_ticket`, `SendToEstAdmin`, `SendToEstTravel`, `SendToEstimatorPA`,
`SendToPICTravel`, `SendtoAnalysator`, `SendToInvestigator`, `CompliancePNC`, `AcceptanceKomite`,
`RCLDokter`, `SendtoPUCL`.

> **Konsekuensi penting untuk desain:** alur nyata **bukan rangkaian linear**. Setiap tahap bisa
> dimasuki dari luar. Model status baru harus mengizinkan transisi lateral ini, bukan memaksakan
> urutan kaku yang akan langsung ditolak user.

### 2.2 Tiga proses pendukung

**Open Protection** (`CreateProtection_Flow`)
`Input Protection → Akseptasi Protection → ◇ Diterima? → Selesai / Ditolak`
Permintaan pembukaan proteksi sebelum atau di luar alur klaim normal. Satu Open Protection
dapat ditautkan ke klaim dan ditandai terpakai (`IsUsedPNC = "1"`) saat registrasi.

**Receive Document** (`ReceiveDocument_Flow`)
`Input Receive Document → Selesai`
Pencatatan penerimaan dokumen fisik, termasuk pengiriman antar cabang (ekspedisi, nomor resi,
estimasi tiba). Menghasilkan `RCV_ID` yang ditautkan ke klaim.

**Komite** (`Komite_Flow`)
`Komite Router → Lihat Detail Transfer → ◇ Masih ada level berikutnya? → ulang / Selesai`
Persetujuan berjenjang. Setiap putaran menaikkan `KomiteCount` dan mereset `AcceptStatus`, lalu
meneruskan ke penyetuju berikutnya.

> **Koreksi v2.0 — jumlah jenjang tidak dipilih dari matriks, melainkan diakumulasi.** Jumlah
> jenjang **sama dengan jumlah baris master yang ambang bawahnya sudah terlampaui nilai klaim**
> (`D-47`): setiap jenjang yang terlampaui **ikut menyetujui**, bukan memilih satu jenjang tunggal.
> Klaim kecil melewati sedikit jenjang, klaim besar melewati banyak.
>
> Batas **4 level** adalah akibat isi master hari ini, **bukan aturan** — secara mekanisme jumlah
> jenjang mengikuti jumlah baris. Untuk lini **Non-MBU** saja, akumulasi didahului pemilihan
> **pita nilai** (sampai Rp 100.000.000 → pita `1`; di atasnya → pita `2`); lini lain tidak punya
> langkah pendahuluan itu (`D-52`, `D-70`). Rinciannya di `20-DETAIL-KOMITE-DBLINK.md` §1.8.

---

## 3. Aturan bisnis yang berhasil diekstraksi

Diambil terutama dari `InputRegister_act` (137 step) dan 70 when rule. Ini bukan daftar lengkap,
tapi ini aturan yang **paling sering menolak input user** sehingga wajib benar di sistem baru.

### 3.1 Aturan tanggal

| Aturan | Berlaku untuk |
|---|---|
| Tanggal Kejadian (DOL) harus di dalam periode polis | Semua |
| DOL boleh sampai 30 hari setelah polis berakhir | Bonding |
| Tanggal cetak boleh sampai 90 hari setelah polis berakhir | Travel & PA |
| Tanggal Lapor ≤ DOL + 7 hari | Semua **kecuali** PA |
| Tanggal Terima Dokumen ≤ DOL + 90 hari | Travel |
| DOL ≤ Tanggal Lapor | Semua (sama hari diperbolehkan) |
| Tanggal Lapor ≤ Tanggal Terima Dokumen | Semua |
| Tidak boleh melebihi tanggal hari ini | DOL, Tanggal Lapor, Tanggal Terima Dokumen |

### 3.2 Aturan duplikasi

- Klaim ditolak bila sudah ada klaim lain dengan **Nomor Polis + Objek + Lokasi** yang sama.
- Untuk **PA**, pengecekan diperketat: **Nomor Polis + Objek + Penyebab Kerugian `12002` + Lokasi**.
- Pesan yang dikembalikan menyertakan nomor klaim yang sudah ada.

### 3.3 Aturan reasuransi

- **Total spreading wajib 100%**, **toleransi 4 desimal, `99,9999`–`100,0001`** (`D-51`). Bila tidak, submit ditolak. Menggantikan pencocokan substring lama yang meloloskan `199.99` dan `1100.0`.
- Bila ada spreading **Fac Out** (`TreatyType = 10015`), data Fac Offer wajib ada.
- Untuk **Group Panel `003`**, Fac Offer wajib menyertakan Object Name.
- Bila klaim ditandai **Ex-Gratia**, treaty `OR` (`10001`) otomatis berubah menjadi `ORS` (`10007`).
- Spreading yang bertanda `FlagDelete = "1"` dibuang sebelum perhitungan.

### 3.4 Aturan nilai dan notifikasi

- Nilai estimasi dikonversi ke IDR memakai **kurs standar** sebelum dibandingkan.
- Estimasi **> Rp 1.000.000.000** memicu **Notice of Large Losses** ke Underwriting sesuai Group
  Panel dan ke jajaran pimpinan.
- Persetujuan komite dipicu ambang **Rp 50.000.000** (umum) dan **Rp 30.000.000** (PA/Travel).
  Verifikasi menemukan **8 ambang komite unik**, bukan dua — termasuk `3.500` yang dipicu
  **hostname entitas Timor-Leste** dan `7.000`, `20.000.000`, `100.000.000` pada jalur lain. Satu
  ambang yang sama (`50.000.000`) bahkan dibandingkan dengan **tiga operator berbeda** di tiga
  rule; itu masuk daftar perbaikan eksplisit `P-5` (`D-49` #2).
- Bila premi belum lunas (`AgingAmount > 1`), notifikasi premi tertunggak dikirim.

### 3.5 Aturan kelengkapan

- **Penyebab Kerugian wajib diisi**, kecuali lini Travel.
- **Nomor SLIK wajib diisi** untuk SPK / Asuransi Kredit.
- Objek tanpa Coverage dibuang otomatis dari daftar.
- Bila hubungan tertanggung dipilih "lain-lain" (`7`), keterangannya wajib diisi.
- Polis Deklarasi tidak dapat diklaim, kecuali lini Aneka.
- `ObjectName` dipotong pada 3.800 karakter karena batas kolom database.

---

## 4. Peran pengguna

22 access group ditemukan di when rule, mengendalikan **47 harness target unik** lewat **51 item menu aksi**; **11 dari 47 harness tidak ada di export** (`K-33`).

| Peran | Tanggung jawab utama |
|---|---|
| **PncAdmin** | Registrasi klaim, input data, unggah dokumen |
| **PncManagerAdmin** | Persetujuan tingkat admin, pemantauan |
| **PncPICTeknik** | Penanggung jawab teknis sesuai lini bisnis |
| **PNCKomiteTeknik** · **PNCKomite** | Persetujuan nilai klaim berjenjang |
| **CaseManager** | Pengawasan lintas kasus |
| **PncRCLPUCL** | Penanganan penolakan dan proses ulang klaim |
| **PncAnalystDoctor** | Penilaian medis klaim PA |
| **PncComplience** | Pemeriksaan kepatuhan |
| **PncInvestigator** | Penyelidikan klaim mencurigakan |
| **PNCSurveyor** | Survei lapangan, unggah foto dan hasil survei |
| **PncPLADLA** | Pengelolaan pemberitahuan ke koasuransi/reasuransi |
| **PncReceive** · **PncManagerReceive** | Penerimaan dokumen fisik |
| **PncCollection** · **PncOPCGeneral** | Open Protection |
| **PNCServiceCenter** | Layanan pelanggan |
| **TreatyIn** | Klaim treaty masuk |
| **ViewClaimPNC** | Akses baca saja |
| **PNCReportClaimInternal** · **PNCReportClaimEksternal** | Akses laporan; eksternal terbatas |
| **Administrators** | Akses penuh |

---

## 5. Aliran nilai uang klaim

Ini tulang punggung bisnisnya, dan urutan inilah yang menentukan model data settlement.

```
Estimasi Klaim                    saat registrasi
      │                           dipakai untuk PLA & ambang Large Losses
      ▼
Usulan Nilai (Propose)            hasil survei / penilaian adjuster
      │                           dipakai untuk Pre-DLA
      ▼
Persetujuan Komite                berjenjang 1–4 level bila melewati ambang
      │
      ▼
Akseptasi (Accepted)              terbit Nomor Akseptasi
      │                           dipakai untuk DLA & LOD ke tertanggung
      ▼
Transfer ke Kasir                 TransferCashierStatus
      │
      ▼
Pembayaran                        ClaimPaidStatus
```

Di setiap tahap, nilai dikurangi **Salvage** dan **Recovery** bila ada, lalu dibagi ke para
penanggung sesuai **Spreading** dan **Koasuransi**.

---

## 6. Integrasi dengan dunia luar

| Sistem | Arah | Keperluan |
|---|---|---|
| **BRI Surf** (`partner.api.bri.co.id`) | Keluar | OAuth token + kirim umpan balik klaim |
| **Arsip Dokumen** (`app8/asm-archive`) | Keluar | Injeksi data arsip dokumen klaim |
| **Storage Dokumen** (`app13/api`) | Dua arah | Unggah dan ambil dokumen klaim |
| **Konversi Gambar** (`aiimage`) | Keluar | Konversi format AVIF |
| **History Payment** (WebLogic internal) | Masuk | Riwayat pembayaran produksi |
| **HCC / HCQ** | Masuk | Autentikasi user (D-07) |
| **@ASMD** dan 5 database lain | Masuk | HRD, GL payment, master sales, polis, jam kerja (D-25) |
| **Kasir** | Keluar | Data rekening dan permintaan pembayaran |
| **SLIK OJK** | Keluar | Pelaporan regulator untuk lini SPK |
| **Email SMTP** | Keluar | 5 correspondence: notifikasi register, large losses, VA, laporan klaim, error produksi |

---

## 7. Karakter beban kerja

| Aspek | Angka | Sumber |
|---|---|---|
| User aktif harian | 200–300 | Pemilik project |
| Klaim baru | Ribuan per bulan | D-10 |
| Data historis | Puluhan juta baris | D-10 |
| Ketersediaan | 24/7 | D-27 |
| Layar | 74 harness, 269 section | Source |
| Laporan | 56 report definition | Source |

**Profil bebannya: data besar, konkurensi rendah.** 200–300 user bersamaan bukan beban berat
untuk Go. Yang berat adalah **query terhadap puluhan juta baris**, terutama pada inbox
berkolom banyak dan laporan lintas periode. Optimasi harus diarahkan ke sana, bukan ke jumlah
request per detik.
