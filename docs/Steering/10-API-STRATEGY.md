# API Strategy — Claim PNC

Kontrak antara SPA React dan backend Go, serta antara Claim PNC dan sistem luar.

---

## 1. Bentuk API

**REST over HTTPS, JSON.** Bukan GraphQL, bukan gRPC.

Alasan: klien hanya satu (SPA kita sendiri), tim belum terbiasa dengan konsep baru (D-09), dan
REST paling mudah ditelusuri dengan alat biasa saat menelusuri masalah di production.

**Endpoint dirancang untuk kebutuhan layar, bukan untuk mencerminkan tabel.** Satu layar
sebaiknya dilayani satu permintaan. Ini penting karena Pega merender seluruh layar sekaligus;
memecah satu layar menjadi delapan panggilan API akan membuat aplikasi baru terasa lebih lambat
dari yang lama meski backend-nya lebih cepat.

---

## 2. Aturan penamaan

| Aturan | Contoh |
|---|---|
| Awalan versi | `/api/v1/...` |
| Sumber daya berbentuk jamak, `kebab-case` | `/api/v1/klaim`, `/api/v1/open-protection` |
| Istilah domain mengikuti `CONTEXT.md` | `/api/v1/klaim/{nomor}/settlement`, bukan `/adjustments` |
| Sub-sumber daya bersarang maksimal 2 tingkat | `/api/v1/klaim/{nomor}/objek` ✅ · `/klaim/{n}/objek/{o}/coverage/{c}/settlement` ❌ |
| Aksi yang bukan CRUD sebagai sub-sumber daya | `POST /api/v1/klaim/{nomor}/akseptasi` |
| Nama field JSON `camelCase` | `nomorKlaim`, `tanggalKejadian` |

**Aksi bisnis dimodelkan sebagai peristiwa, bukan pembaruan field.**
`POST /klaim/{nomor}/akseptasi` — bukan `PATCH /klaim/{nomor}` dengan `{"status": "accepted"}`.
Alasannya: akseptasi punya invarian (I-5: komite harus selesai), memicu peristiwa domain
(penerbitan DLA dan LOD), dan wajib tercatat di jejak audit. Pembaruan field generik akan
melewatkan ketiganya.

---

## 3. Bentuk respons

Seluruh respons memakai amplop yang sama, sehingga frontend punya **satu** cara menangani hasil
dan kesalahan — bukan 74 cara berbeda.

**Berhasil:** objek `data`, ditambah `meta` bila berupa daftar berhalaman.
**Gagal:** objek `error` berisi `code` (dapat dibaca mesin), `message` (dapat dibaca manusia,
bahasa Indonesia), dan `details` berupa daftar kesalahan per field.

**Kesalahan validasi dikembalikan seluruhnya sekaligus**, tidak satu per satu. Ini meniru
perilaku Pega yang menampilkan semua pesan bersamaan — pada form registrasi dengan puluhan field,
mengembalikan satu kesalahan per permintaan akan sangat menyiksa pengguna.

---

## 4. Paginasi, penyaringan, pengurutan

| Aspek | Aturan |
|---|---|
| Paginasi | **Keyset** untuk inbox dan pencarian (D-10: puluhan juta baris). Parameter `cursor` dan `limit` |
| Batas | `limit` maksimum 100. Permintaan lebih besar ditolak, bukan dipenuhi |
| Penyaringan | Parameter query eksplisit per field, **bukan** bahasa filter generik |
| Pengurutan | `sort` dengan daftar kolom yang diizinkan — **tidak pernah** nama kolom mentah dari klien |
| Total baris | Tidak dikembalikan secara baku pada data besar. `COUNT(*)` atas puluhan juta baris mahal; frontend memakai pola "muat lebih banyak" |

> Larangan filter generik dan kolom sort mentah bukan soal kerapian — itu yang menutup celah
> SQL injection yang ada pada pola `{ASIS:...}` warisan (utang teknis 4.5).

---

## 5. Kode status HTTP

| Kode | Dipakai untuk |
|---|---|
| `200` | Berhasil |
| `201` | Sumber daya dibuat |
| `400` | Permintaan tidak valid secara bentuk |
| `401` | Belum login atau token kedaluwarsa |
| `403` | Sudah login tapi tidak berwenang |
| `404` | Tidak ditemukan |
| `409` | Melanggar aturan bisnis atau konflik konkurensi |
| `422` | Validasi bisnis gagal (mis. total spreading bukan 100%) |
| `429` | Terlalu banyak permintaan |
| `500` | Kesalahan tak terduga — **detail internal tidak pernah dibocorkan ke klien** |

Pembedaan `422` dan `400` disengaja: `400` berarti klien salah membentuk permintaan (bug
frontend), `422` berarti permintaannya benar tapi melanggar aturan bisnis (kesalahan pengguna).
Frontend menanganinya berbeda.

---

## 6. Kontrak dan pembuatan tipe

- Kontrak API ditulis sebagai **OpenAPI 3**, disimpan di repository, dan menjadi **sumber
  kebenaran**.
- **Tipe TypeScript dihasilkan otomatis** dari kontrak ini. Ditulis tangan dilarang.
- Perubahan kontrak yang merusak kompatibilitas memerlukan versi baru.

Alasan aturan ini penting untuk tim di D-09: tanpa tipe hasil generate, backend dan frontend
akan berbeda persepsi tentang bentuk data, dan kesalahannya baru muncul saat runtime di layar
pengguna.

---

## 7. Idempotensi

Aksi yang menimbulkan akibat di luar sistem — akseptasi, transfer ke kasir, penerbitan DLA,
pengiriman notifikasi — wajib **idempoten**.

Klien mengirim kunci idempotensi; permintaan ulang dengan kunci yang sama mengembalikan hasil
yang sama tanpa mengulang akibatnya. Ini mencegah pembayaran ganda ketika pengguna menekan
tombol dua kali atau jaringan terputus setelah permintaan terkirim.

---

## 8. Integrasi keluar

### 8.1 Prinsip
Setiap sistem eksternal punya **seam sendiri** (Future Architecture §3.4) — bukan satu interface
raksasa. Kegagalan dan aturan retry setiap sistem berbeda, dan menyatukannya akan memaksa
perlakuan yang sama untuk hal yang tidak sama.

### 8.2 Aturan pemanggilan keluar

| Aturan | Alasan |
|---|---|
| Batas waktu **wajib** di setiap pemanggilan | Tanpa batas waktu, satu sistem yang menggantung akan menghabiskan seluruh koneksi kita |
| Retry hanya untuk operasi idempoten, dengan jeda bertambah | Retry pada operasi non-idempoten menyebabkan pengiriman ganda |
| **Tidak pernah di dalam transaksi database** | Kegagalan jaringan tidak boleh menahan kunci baris |
| Circuit breaker untuk sistem yang sering gagal | Berhenti mencoba ketika jelas sedang bermasalah |
| Setiap pemanggilan dicatat: tujuan, lama, hasil | Tanpa ini, menelusuri masalah integrasi mustahil |
| Kredensial dari konfigurasi, tidak pernah dari kode | Keamanan |

### 8.3 Sistem eksternal

| Sistem | Arah | Sifat | Bila gagal |
|---|---|---|---|
| **HCC/HCQ** | Keluar | Sinkron, menghalangi login · **kontraknya belum ada — nol jejak di export** | Pengguna tidak bisa masuk. **Apakah ada jalur cadangan belum diputuskan** (`ADR-0024`) |
| **Storage Dokumen** (app13/app8) | Dua arah | Sinkron saat unggah | Unggah gagal, klaim tetap tersimpan; dokumen bisa diulang |
| **BRI Surf** | Keluar | Asinkron | Antrekan dan coba lagi |
| **Kasir** | Keluar | Asinkron, idempoten wajib | Antrekan; **tidak boleh kirim ganda** |
| **SLIK OJK** | Keluar | Batch | Catat kegagalan, laporkan |
| **6 API pengganti DB Link** | Masuk | Sinkron | Lihat §8.4 |
| **SMTP** | Keluar | Asinkron | Antrekan dan coba lagi |

### 8.4 Pengganti DB Link (D-25)

| API baru | Menggantikan | Data |
|---|---|---|
| API HRD | `HRDASM.V_HRD_MST@ASMD` | Data pegawai |
| API Jam Kerja | `DATAMINING.GET_WORKING_HOURS@ASMD` (17×) | Perhitungan TAT hari kerja |
| API Pembayaran GL | `GL.T_ALL_PAYMENT@ASMD` | Riwayat pembayaran |
| API Master Sales | `MST_DET_SALES@ASMD`, `MST_SALES@ASMD` | Agen, MO, cabang |
| API Buka Proteksi | `GENERAL.MST_BUKA_PROTEKSI@ASMD/@SIMASNET/@SMI` | Status proteksi |
| API Pengguna Lintas Sistem | `NEW_GENERAL.M_USER@OPJAVA` | Pengguna sistem lain |

**Risiko:** API-API ini kemungkinan belum ada dan harus dibangun tim lain (**R-03**).

**Mitigasi selama menunggu:** untuk data yang jarang berubah — master sales, cabang, agen —
gunakan **salinan yang disegarkan berkala** sebagai jembatan, dengan seam yang sama sehingga
penggantian ke API nyata nanti tidak menyentuh kode domain. Untuk data yang harus mutakhir,
modul yang bergantung padanya tertahan sampai API tersedia.

> **Satu pengecualian yang tidak boleh diganti panggilan jaringan.**
> `DATAMINING.GET_WORKING_HOURS@ASMD` dipakai **18 kali di 7 berkas** di dalam kalkulasi laporan
> massal. Menjadikannya panggilan jaringan per baris akan menghancurkan kinerja laporan. `D-50`
> karena itu menetapkan **logikanya ditulis ulang di Go**, bukan dipanggil lewat API —
> perhitungan jam kerja dan kalender libur adalah **aturan bisnis**, bukan pengambilan data. Hal
> yang sama berlaku untuk `HRD_LBR`.
>
> Konsekuensinya: **kalender libur menjadi master data milik aplikasi ini** (`F-4`), dan dari mana
> daftar hari libur diperoleh setiap tahun **belum ditetapkan**.

### 8.5 Permukaan masuk yang belum pernah dihitung

Seluruh analisis integrasi sebelumnya hanya melihat Connect REST **keluar**. Folder
`Service REST/` yang diterima pada 2026-09-09 memuat **empat layanan REST masuk**:

| Layanan | Isi |
|---|---|
| `KomiteAcceptAdjustment` | **menerima persetujuan komite dari sistem lain** |
| `KomiteAcceptAdjustmentPA` | **menerima persetujuan komite dari sistem lain** (jalur PA) |
| `RecivedDataandAttachmentLelangASMSimasbid` | menerima data dan lampiran hasil lelang |
| `RequestCreateClaimCredit2` | permintaan pembuatan klaim kredit |

**Dua di antaranya menerima persetujuan komite dari luar**, sehingga menyentuh langsung kewenangan
menyetujui uang (`B-7`) dan model otorisasi `D-59`.

> **Lingkup `S-4` lebih besar daripada yang disetujui.** Permukaan masuk ini belum pernah masuk
> hitungan `FR-S4` maupun `D-25`, dan otentikasi setiap layanan masuk harus ditetapkan — siapa
> boleh memanggilnya, dan dengan kredensial apa. Diajukan sebagai pertanyaan terbuka pada
> `ADR-0008`.
