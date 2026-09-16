# Testing Strategy — Claim PNC

Pengujian di project ini punya satu tugas yang lebih penting dari biasanya: **membuktikan bahwa
sistem baru berperilaku sama dengan Pega** (Migration Strategy P-5). Tanpa itu, setiap perbedaan
hasil akan diperdebatkan tanpa cara menyelesaikannya.

> **Diperbarui v2.0 (2026-09-14).** Bab ini menyerap tujuh keputusan baru: perkakas uji kesetaraan
> menjadi modul `S-8` di gelombang 1 (`D-42`), lingkungan dan pelaksananya ditetapkan (`D-53`),
> kewenangan menyetujui selisih dipagari (`D-54`), daftar perbaikan eksplisit `P-5` bertambah dari
> 4 menjadi **13 butir** (`D-49`), dua modul tanpa baseline Pega diukur dengan kontrak (`D-56`),
> `BRD §21.4` dicabut untuk empat modul (`D-55`), dan peran penguji gerbang 2 ditetapkan per
> kelompok modul (`D-60`).

---

## 1. Kenapa pengujian tidak bisa ditawar di sini

Tiga alasan yang khas project ini:

1. **Tim sedang belajar tiga hal baru sekaligus** (D-09). Test adalah jaring pengaman yang
   menangkap kesalahan sebelum sampai ke pengguna.
2. **Perilaku wajib setara dengan Pega.** Satu-satunya cara membuktikannya adalah menuliskan
   aturan bisnis sebagai test yang bisa dijalankan berulang.
3. **Database akan berpindah dari Oracle ke PostgreSQL** (D-01). Test suite yang lengkap adalah
   yang membuktikan perpindahan itu tidak mengubah apa pun. Tanpanya, cutover database menjadi
   lompatan tanpa jaring.

---

## 2. Bentuk piramida

```
        ╱╲          Uji Ujung-ke-Ujung  — sedikit, alur terpenting saja
       ╱──╲
      ╱────╲        Uji Integrasi       — sedang, terhadap database nyata
     ╱──────╲
    ╱────────╲      Uji Aturan Bisnis   — BANYAK, cepat, tanpa infrastruktur
   ╱──────────╲
```

Lapisan terbesar sengaja bukan "unit test" melainkan **uji aturan bisnis**. Perbedaannya bukan
istilah: unit test cenderung menguji fungsi kecil apa pun, sedangkan uji aturan bisnis menguji
**aturan yang bisa disebutkan dalam kalimat bisnis**. Yang kedua jauh lebih berguna saat
membuktikan kesetaraan dengan Pega.

---

## 3. Uji aturan bisnis

**Sasaran:** setiap aturan di `02-BUSINESS-UNDERSTANDING.md` §3 dan setiap invarian di
`05-DOMAIN-MODEL.md` §2 punya test tersendiri.

Berjalan **tanpa database, tanpa jaringan, tanpa berkas** — memakai fake di balik seam
(Future Architecture §3). Ini yang membuatnya cepat dan bisa dijalankan setiap kali menyimpan
berkas.

### 3.1 Cakupan wajib

| Kelompok | Kasus minimum |
|---|---|
| Aturan tanggal | 8 aturan × (lolos, ditolak, tepat di batas) |
| Aturan duplikasi | umum dan varian PA dengan penyebab kerugian `12002` |
| Spreading | tepat 100% · `99,9999` (lolos) · `99,99` (**ditolak** — selisih terencana `D-49` butir 1) · `199.99` (**ditolak**) · kurang · lebih · Fac Out tanpa Fac Offer · Group Panel `003` tanpa Object Name · Ex-Gratia mengubah `OR`→`ORS` |
| Kelengkapan | penyebab kerugian (dan pengecualian Travel) · Nomor SLIK untuk SPK · hubungan tertanggung "lain-lain" |
| Ambang nilai | Large Losses > 1 miliar · ambang komite per lini bisnis |
| Penjenjangan komite | setiap kombinasi matriks nilai × jenis bisnis |
| Status | seluruh transisi yang sah, dan penolakan transisi yang tidak sah |
| Otorisasi | setiap peran terhadap setiap aksi |
| Waktu | perhitungan hari kalender di sekitar tengah malam WIB |

**Kasus "tepat di batas" tidak boleh dilewatkan.** Aturan seperti "Tanggal Lapor ≤ DOL + 7 hari"
paling sering salah tepat pada hari ketujuh — dan sistem lama menangani ini dengan penambahan
7 jam manual yang membuat batasnya bergeser. Inilah tempat perbedaan hasil paling mungkin muncul.

### 3.2 Gaya penulisan
Nama test menyebutkan **aturannya**, bukan nama fungsinya. Nama yang baik terbaca sebagai
kalimat bisnis, sehingga daftar test menjadi dokumentasi aturan yang selalu mutakhir.

---

## 4. Uji integrasi

**Sasaran:** membuktikan lapisan adapter benar-benar bekerja — SQL sah, pemetaan tipe benar,
transaksi berperilaku sesuai harapan.

| Aspek | Aturan |
|---|---|
| Database | **Database nyata**, bukan tiruan. SQL yang tidak dijalankan terhadap database nyata tidak terbukti sah |
| Dijalankan terhadap | **Oracle dan PostgreSQL keduanya** |
| Data awal | Disiapkan dan dibersihkan per test |
| Cakupan | Setiap query di berkas `.sql` minimal dijalankan sekali |

**Menjalankan uji integrasi terhadap kedua database adalah inti dari D-20.** Inilah satu-satunya
mekanisme yang benar-benar membuktikan SQL portabel — bukan disiplin penulisan, bukan review,
melainkan test yang gagal ketika seseorang menulis `NVL`. Bila hanya diuji terhadap Oracle,
ketidakportabelan baru ditemukan saat cutover, dan pada saat itu memperbaikinya sangat mahal.

Uji integrasi juga mencakup adapter sistem eksternal terhadap server tiruan, untuk memastikan
penanganan batas waktu, retry, dan kegagalan berperilaku benar.

---

## 5. Uji ujung-ke-ujung

Sedikit saja, hanya untuk alur yang bila rusak berarti sistem tidak bisa dipakai:

1. Login sampai membuka inbox
2. Registrasi klaim lengkap sampai tersimpan
3. Estimasi sampai akseptasi melewati komite
4. Unggah dokumen
5. Menjalankan laporan dan mengunduh hasilnya

Dijalankan terhadap staging. Sengaja dibatasi jumlahnya karena uji ujung-ke-ujung lambat dan
rapuh; menambah banyak akan membuat tim mengabaikan hasilnya ketika sering gagal karena alasan
yang tidak berhubungan.

---

## 6. Uji kesetaraan dengan Pega

Khas project migrasi, dan inilah yang menjawab P-5.

| Cara | Kapan | Isi |
|---|---|---|
| **Perbandingan hasil baca** | Tahap 2 migrasi | Query yang sama dijalankan di Pega dan Go terhadap data staging; hasilnya dibandingkan baris per baris. Perbedaan wajib dijelaskan sebagai bug atau sebagai perbaikan yang disengaja |
| **Perbandingan hasil validasi** | Tahap 3 | Data registrasi yang sama dimasukkan ke kedua sistem; daftar pesan kesalahannya dibandingkan |
| **Perbandingan perhitungan** | Tahap 3–5 | Perhitungan spreading, konversi kurs, dan nilai settlement dibandingkan hasilnya |
| **Rekonsiliasi harian** | Selama masa paralel | Jumlah klaim, total nilai akseptasi, dan jumlah penugasan dibandingkan antar sistem |

**Setiap perbedaan wajib punya kesimpulan tertulis:** bug di sistem baru, atau perbaikan yang
disengaja terhadap perilaku lama. Tidak boleh ada perbedaan yang dibiarkan tanpa penjelasan —
karena satu perbedaan yang tidak dijelaskan akan menjadi alasan meragukan seluruh hasil
perbandingan.

### 6.1 Perkakas pelaksananya: modul `S-8`

Uji kesetaraan tidak dijalankan manual. Ia dikerjakan **modul `S-8` Perkakas Uji Kesetaraan**,
yang masuk **gelombang 1** karena memblokir gerbang 1 setiap modul lain (`D-42`).

| Hal | Ketetapan |
|---|---|
| **Lingkungan** | **Pega staging vs Go staging**, atas **salinan data produksi** — bukan data buatan, dan **tidak pernah menembak produksi** (`D-53`) |
| **Pelaksana** | tim pengembang |
| **Keluaran wajib** | setiap selisih **diklasifikasikan**, bukan sekadar dilaporkan: terpetakan ke butir `P-5` yang mana, atau tidak terpetakan (`D-54`) |

**Alasan memakai data produksi, bukan data buatan:** cacat yang ditemukan pada verifikasi Fase 1 —
toleransi spreading berupa pencocokan substring, kurs yang mengembalikan `1`, `IDSALVAGE = NULL` —
**muncul dari data nyata yang tidak akan terpikir dibuat**.

### 6.2 Kewenangan menyetujui selisih

| Jenis selisih | Perlakuan |
|---|---|
| Cocok dengan salah satu dari **13 butir `P-5`** | **lolos otomatis**, cukup dicatat |
| **Di luar 13 butir itu** | wajib **persetujuan Work Owner secara tertulis** sebelum modul dinyatakan lulus |

Ketiga belas butir sudah diputuskan eksplisit di `D-49`; meminta persetujuan ulang per modul
menambah beban tanpa menambah kendali. Yang menuntut perhatian adalah selisih **yang tidak
terduga** — itulah yang dipagari.

### 6.3 Selisih yang sudah dapat diperkirakan sekarang

Tiga perubahan yang diputuskan akan **pasti** memunculkan selisih, dan harus dinyatakan di muka
sebagai perbaikan terencana — bukan ditemukan sebagai kejutan:

| Selisih | Sebab | Rujukan |
|---|---|---|
| Seluruh data historis **valuta asing** berbeda | basis kurs berubah menjadi **kurs tanggal kejadian** | `D-48`, `ADR-0015` |
| Klaim dengan total spreading di luar toleransi kini **ditolak** | pencocokan substring diganti `ROUND(SUM(share),4) BETWEEN 99.9999 AND 100.0001` | `D-51`, `ADR-0016` |
| Laporan yang dulu terpotong **500 baris** kini utuh | batas `pyMaxRecords=500` dihapus | `ADR-0011` |

### 6.4 Dua hal yang **tidak boleh** dibandingkan begitu saja

1. **Jumlah baris tabel.** `D-66` menetapkan **soft delete menyeluruh** — yang terhapus tetap
   tidak muncul bagi pengguna, tetapi barisnya tetap ada. Perbandingan berbasis `COUNT(*)` pada
   tabel akan **selalu berbeda dan bukan indikasi cacat**. Yang dibandingkan adalah **hasil kueri
   sesuai aturan bisnis**.
2. **Perilaku saat gagal pada `B-4` dan `B-9`.** Sistem lama menempuh sembilan `COMMIT` dan dapat
   meninggalkan data setengah jalan; sistem baru membungkusnya satu transaksi (`D-68`,
   `ADR-0007`). Kegagalan menghasilkan keadaan akhir yang berbeda **secara sengaja**.

---

## 7. Pengujian frontend

| Jenis | Cakupan |
|---|---|
| Komponen | Pustaka komponen baku (U-2) — terutama `DataTable` dan komponen form. Dipakai ratusan kali, jadi kesalahan di sini berlipat ganda |
| Integrasi | Alur per fitur dengan API tiruan |
| Tipe | `tsc --noEmit` di CI — mode ketat, `any` dilarang |

Komponen khusus fitur diuji lebih ringan; investasi pengujian diarahkan ke komponen bersama
karena di situlah **leverage**-nya.

---

## 8. Uji beban

Dijalankan sebelum go-live, terhadap staging dengan data sebesar produksi (D-10: puluhan juta
baris).

| Skenario | Sasaran |
|---|---|
| Inbox dengan data penuh | Waktu tampil di bawah target NFR |
| Pencarian klaim rentang lebar | Tidak menghabiskan connection pool |
| Laporan besar bersamaan | Tidak mengganggu transaksi pengguna lain |
| 300 pengguna bersamaan | Waktu respons tetap dalam target |
| Export besar | Memori tidak meledak — harus streaming, bukan dimuat seluruhnya |

**Uji beban wajib memakai data sebesar produksi.** Query yang cepat terhadap seribu baris bisa
sangat lambat terhadap sepuluh juta baris, dan perbedaannya tidak akan terlihat pada data uji
yang kecil. Ini risiko terbesar pada profil beban D-10.

---

## 9. Yang wajib lulus sebelum merge

| Pemeriksaan | Blokir merge |
|---|---|
| Seluruh uji aturan bisnis lulus | Ya |
| Seluruh uji integrasi lulus (Oracle **dan** PostgreSQL) | Ya |
| Lint backend dan frontend bersih | Ya |
| Aturan ketergantungan antar lapisan tidak dilanggar (`depguard`) | Ya |
| Tidak ada pola SQL terlarang | Ya |
| Tidak ada kesalahan tipe TypeScript | Ya |
| Uji ujung-ke-ujung lulus | Sebelum rilis, tidak setiap merge |

---

## 10. Dua gerbang penerimaan per modul

Setiap modul melewati **dua gerbang** sebelum dinyatakan pindah:

| Gerbang | Isi | Berlaku untuk |
|---|---|---|
| **Gerbang 1** | uji kesetaraan otomatis oleh `S-8` | seluruh modul, **kecuali** `F-3` dan `S-5` |
| **Gerbang 2** | UAT pengguna bisnis | **modul bisnis saja** |

**Peran penguji gerbang 2** (`D-60`):

| Kelompok modul | Gerbang 2 | Penguji |
|---|---|---|
| **Modul bisnis** — `B-1`…`B-14`, `S-1`…`S-4`, `S-6`, `S-7`, `U-1`, `U-3`…`U-6` | **berlaku** | peran bisnis pemakai inbox/layar modul itu |
| **Modul fondasi** — `F-1`…`F-5`, `S-5`, `S-8`, `U-2` | **tidak berlaku** | gerbang 1 + **persetujuan Work Owner** |

Memaksakan gerbang 2 pada modul yang tidak punya peran pemakai akan membuat tiket fondasi macet
di gerbang yang tidak dapat dilewati siapa pun — tidak ada pengguna bisnis yang membuka layar
"Akses Data".

### 10.1 Dua modul yang tidak punya baseline Pega

`F-3` dan `S-5` **tidak dapat diuji kesetaraannya** karena tidak ada yang bisa dibandingkan:
HCC/HCQ tidak meninggalkan jejak apa pun di export, dan sistem lama tidak mencatat perubahan nilai
sama sekali. Untuk keduanya, gerbang 1 diganti **uji fungsional terhadap kontrak** (`D-56`):

| Modul | Kontrak yang menjadi acuan | Pemilik kontrak | Status kontrak |
|---|---|---|---|
| **F-3** Identitas & Akses | kontrak API HCC/HCQ — field request/response login, kode galat, timeout, endpoint refresh/validasi | pemilik API HCC/HCQ | **belum ada** |
| **S-5** Jejak Audit | daftar peristiwa wajib audit beserta field yang harus tercatat | Compliance | **belum ada** |

Menghapus gerbang 1 begitu saja **ditolak**: keduanya justru modul paling sensitif — `F-3` adalah
otorisasi, `S-5` adalah jejak audit — dan UAT tidak memeriksa hal yang tidak terlihat di layar.

**Konsekuensi yang harus diterima:** sampai kedua kontrak diterima, `F-3` dan `S-5` **tidak dapat
lulus gerbang apa pun**.

### 10.2 `BRD §21.4` dicabut untuk empat modul

Setelah `Database/` diterima, penghalang yang mendasari `BRD §21.4` sudah tidak berlaku untuk
sebagian modul (`D-55`):

| Modul | Status | Sisa penghalang |
|---|---|---|
| **B-7** Komite | **lepas** | PA dan Travel di atas Rp 200.000.000 tidak punya baris master |
| **B-9** PLA/DLA | **lepas** | status `VALID`/`INVALID` `GET_GROUPBUSINESS_XOL`; penulisan ulang procedure ber-9-`COMMIT` |
| **B-10** Akseptasi | **lepas** | 3 activity hilang: `InsertDataAkseptasiToLeader`, `InsertLogKasir_act`, `TransferCashierDataASM_act` |
| **B-12** Salvage | **lepas** | `SET_ATTACHFILETEMPSALVAGE`; hitungan DBA atas baris ber-`IDSALVAGE` NULL |
| **B-5** Settlement | **tetap terikat** | isi `POOLDATA.GCNM_FEE_SCALE` (17 pita) · isi `m_currencystandard` · `BrowseT_Claim_Adjustment_SQL` |

Usulan revisi `BRD §21.4` menunggu persetujuan Work Owner; pencabutannya dinyatakan lewat
`ADR-0028`, bukan dengan menyunting BRD diam-diam.

---

## 11. Catatan atas tekanan jadwal

Target D-30 sangat ketat, dan pada project seperti ini pengujian adalah hal pertama yang
tergoda untuk dikorbankan.

Bila pengujian dipangkas, yang **paling akhir** boleh dikorbankan adalah:
1. **Uji aturan bisnis** — inilah yang membuktikan kesetaraan dengan Pega; tanpanya tidak ada
   dasar untuk menyatakan migrasi berhasil.
2. **Uji integrasi terhadap kedua database** — tanpanya, janji SQL portabel (D-20) tidak
   terbukti dan cutover ke PostgreSQL menjadi taruhan.

Uji ujung-ke-ujung dan sebagian uji komponen frontend jauh lebih bisa ditunda tanpa kehilangan
dasar pembuktian.
