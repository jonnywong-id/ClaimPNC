# Future Architecture — Claim PNC (Go + React)

Arsitektur sasaran, batas modul, dan seam. Dirancang memakai kosakata dari skill
`mattpocock-skills:codebase-design`: **module** (punya interface dan implementation),
**interface** (segala yang harus diketahui pemanggil), **depth** (banyak perilaku di balik
interface kecil), **seam** (tempat perilaku bisa diganti tanpa mengedit di tempat itu), dan
**adapter** (yang mengisi seam).

Prinsip yang dipegang: **satu adapter berarti seam hipotetis; dua adapter berarti seam nyata.**
Kita hanya membuat seam bila ada yang benar-benar bervariasi di sana.

---

## 1. Gambaran umum

```
┌───────────────────────────────────────────────────────────────────────┐
│  Browser — React 18 + TypeScript + Vite (SPA, berkas statis)          │
│  TanStack Table / AG Grid · TanStack Query · React Router             │
└───────────────────────────────┬───────────────────────────────────────┘
                                │ HTTPS · JSON · Bearer token
┌───────────────────────────────▼───────────────────────────────────────┐
│                     Load Balancer (2+ instance, D-27)                  │
└───────────────────────────────┬───────────────────────────────────────┘
                                │
┌───────────────────────────────▼───────────────────────────────────────┐
│  Aplikasi Go (satu binary — menyajikan API dan berkas statis SPA)     │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  Transport      HTTP handler · routing · autentikasi         │    │
│  │                 validasi request · serialisasi response       │    │
│  └──────────────────────────┬───────────────────────────────────┘    │
│  ┌──────────────────────────▼───────────────────────────────────┐    │
│  │  Domain         Modul bisnis — aturan, invarian, alur        │    │
│  │                 Tidak tahu apa pun soal HTTP maupun SQL       │    │
│  └──────────────────────────┬───────────────────────────────────┘    │
│  ┌──────────────────────────▼───────────────────────────────────┐    │
│  │  Seam           Interface yang dideklarasikan Domain          │    │
│  │                 Repository · Clock · Storage · Notifier ·     │    │
│  │                 ExternalSystem · Identity                     │    │
│  └──────────────────────────┬───────────────────────────────────┘    │
│  ┌──────────────────────────▼───────────────────────────────────┐    │
│  │  Adapter        Implementasi nyata di balik tiap seam         │    │
│  │                 SQL · HTTP client · SMTP · jam sistem         │    │
│  └──────────────────────────┬───────────────────────────────────┘    │
└─────────────────────────────┼─────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
┌───────▼────────┐  ┌─────────▼─────────┐  ┌────────▼──────────┐
│ Oracle 19c     │  │ Storage Dokumen   │  │ HCC / HCQ         │
│ → PostgreSQL   │  │ (app13 / app8)    │  │ Autentikasi       │
│    17+ (D-24)  │  └───────────────────┘  └───────────────────┘
│ SQL portabel   │  ┌───────────────────┐  ┌───────────────────┐
│ tanpa procedure│  │ BRI Surf · SLIK   │  │ 6 API pengganti   │
└────────────────┘  │ Kasir · SMTP      │  │ DB Link (D-25)    │
                    └───────────────────┘  └───────────────────┘
```

**Satu binary Go** yang menyajikan API sekaligus berkas statis SPA. Tidak ada runtime Node.js di
production (D-23). Ini menjaga operasional tetap sederhana di VM on-premise (D-08).

---

## 2. Empat lapisan dan aturan ketergantungannya

Aturannya satu kalimat, dan ini **wajib ditegakkan di code review**:

> **Ketergantungan hanya boleh mengarah ke dalam. Domain tidak boleh mengimpor apa pun dari
> Transport maupun Adapter.**

| Lapisan | Boleh tahu | Dilarang tahu |
|---|---|---|
| **Transport** | Domain | SQL, tabel, detail sistem eksternal |
| **Domain** | Interface seam yang ia deklarasikan sendiri | HTTP, SQL, JSON wire format, nama tabel |
| **Seam** | Tipe domain | apa pun tentang implementasi |
| **Adapter** | Interface seam, teknologi | aturan bisnis |

**Kenapa ini penting khusus untuk project ini:** aturan bisnis Claim PNC saat ini tersebar di
tiga tempat — activity Pega, SQL, dan stored procedure — sehingga satu perubahan aturan harus
dicari di tiga tempat dan sering hanya ditemukan di dua. Aturan ketergantungan ini memaksa
**locality**: satu aturan bisnis hidup di satu tempat.

---

## 3. Seam yang kita buat, dan alasannya

Setiap seam di bawah punya **minimal dua adapter nyata** — kalau tidak, ia tidak dibuat.

### 3.1 Repository — seam ke database

**Interface:** dideklarasikan oleh Domain, satu per aggregate.
**Adapter:** (1) implementasi SQL portabel, (2) fake di memori untuk pengujian.

Meskipun D-20 menetapkan satu set SQL portabel untuk Oracle dan PostgreSQL, seam ini tetap
nyata karena adapter keduanya adalah **fake untuk pengujian**. Itu yang membuat aturan bisnis
bisa diuji tanpa database sama sekali.

Interface dijaga tetap **dangkal secara permukaan, dalam secara isi**: ia bicara dalam bahasa
domain (`FindByNomorKlaim`, `SimpanSettlement`), bukan bahasa SQL (`Query`, `Exec`, `Scan`).

### 3.2 Clock — seam ke waktu

**Interface:** `Now()` dan konversi zona waktu.
**Adapter:** (1) jam sistem, (2) jam tetap untuk pengujian.

Ini **menjawab langsung utang teknis 4.4**. Saat ini `+7 jam` ditambahkan manual di puluhan
tempat lewat `addCalendar(...,0,0,0,0,7,0,0)` dan activity `Set7Hours`. Satu tempat yang lupa
memanggilnya menggeser tanggal 7 jam tanpa terdeteksi — dan pada aturan "Tanggal Lapor ≤ DOL +
7 hari", pergeseran itu mengubah hasil validasi.

Aturan baru yang mengikat:
- Semua waktu disimpan sebagai **UTC** di database.
- Konversi ke **WIB (Asia/Jakarta)** hanya terjadi di **satu tempat**: modul Clock.
- Aturan bisnis berbasis *hari kalender* (batas 7 hari, 30 hari, 90 hari) dihitung terhadap
  **tanggal WIB**, bukan timestamp UTC.
- **Tidak ada satu pun penambahan 7 jam manual** di kode baru.

Seam ini juga membuat seluruh aturan tanggal di bagian 3.1 Business Understanding bisa diuji
secara deterministik.

### 3.3 DocumentStore — seam ke penyimpanan dokumen

**Interface:** unggah, ambil, hapus, daftar — dalam istilah domain (dokumen klaim, kategori,
pemilik), bukan istilah HTTP.
**Adapter:** (1) API storage internal Sinarmas (D-16), (2) fake untuk pengujian.

Menyatukan **tiga mekanisme yang sekarang hidup berdampingan** (BLOB Oracle, tabel Pega, API
eksternal) menjadi satu jalur. Database aplikasi hanya menyimpan metadata dan referensi.

### 3.4 ExternalSystem — seam ke sistem lain

**Interface:** satu interface sempit per sistem, bicara dalam istilah domain.
**Adapter:** (1) HTTP client nyata, (2) fake untuk pengujian.

Menggantikan **64 pemakaian** DB Link (`D-25`) dan **21 Connect-REST** (`D-73`). Setiap sistem eksternal punya
seam sendiri — **bukan satu interface raksasa untuk semuanya**, karena kegagalan dan aturan
retry-nya berbeda-beda.

### 3.5 Identity — seam ke autentikasi

**Interface:** verifikasi kredensial, kembalikan profil pengguna.
**Adapter:** (1) API HCC/HCQ (D-07), (2) fake untuk pengujian dan pengembangan lokal.

Otorisasi **tidak** berada di seam ini — otorisasi dimiliki aplikasi dan hidup di Domain.

> **Seam ini naik derajat (2026-09-14).** `HCC` dan `HCQ` muncul **2× di seluruh export**, keduanya
> teks pesan galat — **tidak ada Connect REST, tidak ada pemetaan field, tidak ada penanganan
> kegagalan**. Adapter pertama karena itu dibangun terhadap **kontrak yang belum ada**
> (`ADR-0024`, `Proposed`), dan `F-3` tidak punya baseline untuk diuji kesetaraannya.
>
> Justru karena itu seam ini **wajib ada sejak awal**: ia satu-satunya cara `F-3` dapat dikerjakan
> sebelum kontrak HCC/HCQ tiba, dengan adapter fake sebagai penopang sementara. Konsekuensinya
> disadari — jalur yang dipakai saat pengembangan bukan jalur yang dipakai di produksi, sehingga
> kelas cacat integrasi baru muncul terlambat.

### 3.6 Notifier — seam ke pemberitahuan

**Interface:** kirim pemberitahuan sesuai peristiwa domain (`NotifikasiKerugianBesar`,
`NotifikasiPremiTertunggak`), bukan `SendEmail(to, subject, body)`.
**Adapter:** (1) SMTP, (2) fake yang merekam.

Perbedaan ini penting: dengan interface berbasis peristiwa domain, **siapa penerimanya adalah
urusan konfigurasi** (D-15), bukan urusan pemanggil. Inilah yang menghapus hardcode email UW
dan pimpinan sekaligus menutup kemungkinan blok "TESTING" terulang.

---

## 4. Modul domain yang dalam

Tiap modul di bawah menyembunyikan banyak perilaku di balik interface kecil. Ukurannya diambil
dari sistem lama agar terlihat **leverage**-nya.

### 4.1 Registrasi Klaim — modul terdalam

| | |
|---|---|
| **Interface** | Terima data registrasi, kembalikan klaim yang sah atau daftar kesalahan validasi |
| **Menyembunyikan** | 8 aturan tanggal · 2 aturan duplikasi · aturan spreading 100% · kelengkapan penyebab kerugian dan Nomor SLIK · penautan Open Protection · konversi kurs · ambang Large Losses · pembentukan snapshot polis |
| **Asal di sistem lama** | `InputRegister_act` — **137 step** |
| **Leverage** | Satu pemanggilan menggantikan 137 step yang tersebar |
| **Locality** | Seluruh aturan registrasi berubah di satu tempat |

**Uji deletion:** kalau modul ini dihapus, kompleksitasnya muncul kembali di setiap pemanggil —
API registrasi, impor batch, dan koreksi data. Modul ini jelas membayar dirinya sendiri.

### 4.2 Spreading Reasuransi

| | |
|---|---|
| **Interface** | Terima daftar spreading sebuah coverage, kembalikan hasil perhitungan atau pelanggaran aturan |
| **Menyembunyikan** | Aturan total 100% (**toleransi 4 desimal, `99,9999`–`100,0001`** (`D-51`)) · penanganan Fac Out dan kelengkapan Fac Offer · aturan Group Panel `003` · konversi Ex-Gratia `OR`→`ORS` |
| **Kenapa dalam** | Aturan ini sekarang tersebar di `InputRegister_act`, `CallSpreadingView`, dan 45 activity bernama `*Spreading*`/`*Reas*` |

### 4.3 Penjenjangan Komite

| | |
|---|---|
| **Interface** | Terima klaim, kembalikan langkah persetujuan berikutnya (atau: sudah selesai) |
| **Menyembunyikan** | Matriks nilai klaim × jenis bisnis (D-14) · penambahan `KomiteCount` · reset `AcceptStatus` · penentuan workbasket tujuan · jejak persetujuan |
| **Catatan** | Matriks adalah **master data**, bukan kode (D-15). Modul membaca matriks, tidak memuatnya. |

### 4.4 Settlement Klaim

| | |
|---|---|
| **Interface** | Ajukan nilai, setujui nilai, catat pembayaran |
| **Menyembunyikan** | Perjalanan Estimasi → Usulan → Akseptasi → Dibayar · penerbitan Nomor Akseptasi · pengurangan Salvage dan Recovery · pembagian ke koasuransi · jejak audit (D-28) |
| **Invarian yang dijaga** | Nilai tidak boleh melebihi TSI · akseptasi hanya sah setelah komite selesai · setiap perubahan nilai tercatat |

### 4.5 Pemberitahuan Reasuransi (PLA / Pre-DLA / DLA)

| | |
|---|---|
| **Interface** | Terbitkan pemberitahuan sesuai tahap klaim |
| **Menyembunyikan** | Pemilihan tahap yang tepat · perhitungan nilai per penanggung · pembuatan dokumen · pengiriman ke koasuransi/reasuransi |
| **Asal di sistem lama** | 69 activity bernama `*DLA*`/`*PLA*`/`*Treaty*`/`*XOL*`/`*Coins*` |

### 4.6 Penugasan (Worklist & Workbasket)

| | |
|---|---|
| **Interface** | Tugaskan, ambil dari antrean, lepaskan, pindahkan |
| **Menyembunyikan** | Aturan routing per tahap · pembedaan Worklist dan Workbasket (D-26) · penguncian agar dua orang tidak mengerjakan tugas yang sama |
| **Menggantikan** | `PC_ASSIGN_WORKLIST`, `PC_ASSIGN_WORKBASKET`, `PR_SYS_LOCKS`, dan 5 router |

### 4.7 Otorisasi

| | |
|---|---|
| **Interface** | Boleh atau tidak pengguna ini melakukan aksi ini pada klaim ini |
| **Menyembunyikan** | Pemetaan pengguna → peran → menu → izin (D-07) · aturan visibilitas berdasarkan cabang dan lini bisnis |
| **Menggantikan** | 22 access group dan puluhan when rule berbasis `AccessGroup.pyAccessGroup` |

### 4.8 Pembuatan Dokumen (PDF / Excel / CSV)

| | |
|---|---|
| **Interface** | Hasilkan dokumen dari data laporan |
| **Menyembunyikan** | Tata letak · penomoran halaman · streaming untuk data besar · pembatasan memori |
| **Sesuai** | D-11 — dibuat sendiri di Go, bukan engine Pega maupun BI eksternal |

---

## 5. Bounded Context

Empat konteks, dengan **Domain Klaim sebagai inti**. Batasnya ditentukan oleh **siapa pemilik
datanya**, bukan oleh kemiripan teknis.

```
┌──────────────────────────────────────────────────────────────┐
│                    DOMAIN KLAIM (inti)                        │
│                       kita miliki                             │
│                                                               │
│  Klaim · Objek Pertanggungan · Coverage · Settlement          │
│  Spreading · Komite · Survey · Salvage · Recovery             │
│  Open Protection · Receive Document · Penugasan               │
└───┬──────────────┬─────────────────┬─────────────────┬────────┘
    │ snapshot     │ referensi       │ referensi       │ kirim
    │ (D-04)       │                 │                 │
┌───▼──────────┐ ┌─▼─────────────┐ ┌─▼──────────────┐ ┌▼──────────────┐
│ POLIS        │ │ IDENTITAS     │ │ MASTER DATA    │ │ KOASURANSI &  │
│ (GISFW)      │ │ & AKSES       │ │                │ │ REASURANSI    │
│              │ │               │ │                │ │               │
│ tim lain     │ │ HCC/HCQ +     │ │ kita miliki    │ │ pihak luar    │
│              │ │ kita miliki   │ │                │ │               │
│ hanya baca   │ │               │ │ Cabang · Bisnis│ │ PLA · Pre-DLA │
│ lewat        │ │ Auth: HCC/HCQ │ │ Penyebab Rugi  │ │ DLA · LOD     │
│ snapshot     │ │ Authz: kita   │ │ Surveyor · Bank│ │               │
│              │ │               │ │ Ambang komite  │ │               │
└──────────────┘ └───────────────┘ └────────────────┘ └───────────────┘
```

### Kontrak antar konteks

| Dari → Ke | Bentuk | Aturan |
|---|---|---|
| Polis → Klaim | **Snapshot** saat registrasi (D-04) | Klaim **tidak pernah** membaca polis secara langsung setelah registrasi. Perubahan polis tidak mengubah klaim yang sudah berjalan. |
| Identitas → Klaim | Autentikasi via HCC/HCQ, profil dikembalikan | Otorisasi **tidak** didelegasikan — dimiliki Domain Klaim (D-07) |
| Master Data → Klaim | Referensi berdasarkan kode | Master data dimiliki aplikasi dan dapat diubah tanpa deploy (D-15) |
| Klaim → Koasuransi/Reasuransi | Dokumen pemberitahuan | Satu arah keluar; tidak ada ketergantungan runtime |

**Kenapa snapshot, bukan pemanggilan langsung:** GISFW dikembangkan tim lain dengan jadwal
sendiri (D-03). Bila Domain Klaim memanggilnya saat runtime, ketersediaan klaim menjadi
bergantung pada ketersediaan sistem tim lain — dan target 24/7 (D-27) menjadi mustahil dipenuhi
secara sepihak. Snapshot memutus ketergantungan itu, sekaligus benar secara bisnis: **klaim
harus dinilai berdasarkan kondisi polis pada saat kejadian, bukan kondisi hari ini.**

---

## 6. Yang sengaja tidak dibangun

Menyatakan yang tidak dibangun sama pentingnya dengan yang dibangun.

| Tidak dibangun | Alasan |
|---|---|
| **Microservices** | 200–300 user harian, satu tim, satu database. Microservices menambah kerumitan jaringan, deployment, dan penelusuran tanpa menyelesaikan satu pun masalah nyata kita. Satu binary jauh lebih cocok untuk VM on-premise. |
| **Message broker (Kafka/RabbitMQ)** | Tidak ada kebutuhan pemrosesan asinkron bervolume tinggi. Bila kelak dibutuhkan, seam Notifier sudah siap menerimanya. **Catatan:** pernyataan "job terjadwal cukup ditangani scheduler di dalam aplikasi" **belum menjadi keputusan** — dengan dua instans di belakang load balancer, mekanismenya masih dipilih di `ADR-0022` (`Proposed`). |
| **Cache terdistribusi (Redis)** | Sistem lama berjalan **tanpa caching sama sekali** (7 data page semuanya `refresh=never`). Masalah performa ada di query terhadap puluhan juta baris — itu diselesaikan index dan paginasi, bukan cache. Cache in-process untuk master data sudah memadai. |
| **Event sourcing / CQRS** | Kebutuhan jejak audit (D-28) dijawab tabel riwayat append-only yang jauh lebih sederhana dan bisa dipahami tim eks-Pega. |
| **GraphQL** | Klien hanya satu (SPA kita sendiri). REST dengan endpoint yang dirancang untuk kebutuhan layar lebih sederhana dan lebih mudah di-cache. |
| **ORM penuh** | D-20 menetapkan SQL ditulis langsung. Query warisan Pega terlalu kompleks untuk ORM, dan tim akan kesulitan men-debug SQL hasil generate. |
| **Kubernetes** | D-08 menetapkan VM on-premise. Rolling deployment 24/7 (D-27) bisa dicapai dengan dua instance di belakang load balancer. |
