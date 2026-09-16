# Understanding Log — Aplikasi Claim PNC

Ringkasan hasil analisis pemahaman aplikasi **Claim PNC** (Pega PRPC 8.3) sebagai dasar migrasi
ke **Golang + React**. Disusun mengikuti 19 aspek pemahaman yang diminta pada Phase 1.

| | |
|---|---|
| **Tanggal** | 2026-09-08 · **disinkronkan 2026-09-14** |
| **Single Source of Truth** | [`Steering/STEERING.md`](Steering/STEERING.md) **v2.0** — Riwayat Revisi + 16 bab + Lampiran A–F |
| **Sumber kerja** | [`Steering/`](Steering/) — 24 berkas · [`ADR/`](ADR/) — 29 ADR |
| **Basis analisis** | **2.634 berkas XML** + 55 `.prc` + 8 `.fnc` — 902 activity, 652 rule SQL, 269 section, 74 harness, 70 when rule, 4 flow |
| **Konfirmasi bisnis** | Pemilik bisnis, 2026-09-07 dan 2026-09-08 |
| **Status pemahaman** | Lengkap untuk 13 dari 19 aspek · 6 aspek bergantung informasi pihak luar |

---

## Catatan metode — apa yang saya lakukan dan tidak lakukan

Pemahaman mendalam atas aplikasi ini **sudah dilakukan dan terdokumentasi** pada sesi
2026-09-07/08, bersumber langsung dari pembacaan export rule XML. Berkas ini **tidak menurunkan
ulang** pemahaman itu — ia merangkumnya per aspek, menunjuk ke bab sumbernya, dan menandai
dengan tegas di mana pemahaman itu **berhenti**.

Yang saya kerjakan pada sesi ini:

1. Membaca seluruh berkas Steering dan `Steering/STEERING.md`.
2. Memeriksa konsistensi antar berkas (hasil: §21 di bawah).
3. Mewawancarai pemilik project untuk **enam hal yang benar-benar tidak ada** di Steering dan
   hanya bisa dijawab olehnya — semuanya diperlukan BRD (lihat [`interview-history.md`](interview-history.md), Sesi 3).
4. Menyusun BRD dan dokumen turunannya.

**Tidak ada asumsi baru yang saya tambahkan.** Setiap pernyataan di bawah punya penanda asal:

`[KODE]` = terbukti dari source XML · `[BISNIS]` = dikonfirmasi pemilik bisnis · `[TERBUKA]` = belum diketahui, tidak boleh ditebak

> **Disinkronkan dengan Steering v2.0 (2026-09-14).** Log ini adalah **hasil analisis bertanggal
> 2026-09-08**. Angka dan rujukan yang kini berbeda sudah dikoreksi di tempatnya; kesimpulan
> analisisnya **tidak dihitung ulang**. Apa yang berubah sejak itu ada di
> [`Steering/21-RIWAYAT-REVISI.md`](Steering/21-RIWAYAT-REVISI.md).

---

## 1. Business Process — **DIPAHAMI** `[KODE]` `[BISNIS]`

Klaim asuransi umum lini **Non-Motor (Non-MBU = PNC)** di Asuransi Sinar Mas, dari laporan
kerugian tertanggung sampai pembayaran ganti rugi dan pemberitahuan ke koasuransi/reasuransi.

Enam lini bisnis dikenali lewat **Group Panel**: `002` PA · `003`/`009` Aneka · `004` Marine
Cargo · `005` Travel · `006` Fire/Property. Lini khusus: TKA, SPK (Asuransi Kredit), Bonding,
HE, Contractors PM.

Empat proses: **Register** (alur utama), **Open Protection**, **Receive Document**, **Komite**.

→ `Steering/STEERING.md` §1.1–1.2 · [`Steering/02-BUSINESS-UNDERSTANDING.md`](Steering/02-BUSINESS-UNDERSTANDING.md) §1–2

## 2. Business Objective — **DIPAHAMI** `[BISNIS]`

Pendorong utama dikonfirmasi pada 2026-09-08 (I-01): **kemandirian teknologi dan
maintainability** — lepas dari ketergantungan platform berlisensi dan vendor, agar tim dapat
mengembangkan sendiri tanpa keterbatasan model rule Pega.

Yang **bukan** pendorongnya, dan ini penting karena mengubah dasar argumen jadwal:

| Bukan pendorong | Konsekuensi |
|---|---|
| Penghematan biaya lisensi Pega | Tidak ada angka penghematan yang bisa dipakai sebagai justifikasi ke manajemen |
| Lisensi atau dukungan vendor berakhir | **Tidak ada tenggat eksternal.** Target akhir September 2026 (D-30) adalah target internal manajemen, bukan tanggal yang dipaksakan pihak luar |
| Arahan standardisasi teknologi grup | Bukan keputusan top-down |

→ `Steering/STEERING.md` §1.6–1.7 · [`interview-history.md`](interview-history.md) I-01

## 3. Functional Requirement — **DIPAHAMI** `[KODE]`

27 modul terpetakan dari 778 activity custom: 5 fondasi (F-1…F-5), 14 bisnis inti (B-1…B-14),
7 pendukung (S-1…S-7), 6 frontend (U-1…U-6). Setiap modul punya ukuran terukur dari jumlah
activity lama dan urutan ketergantungannya.

Daftar requirement bernomor: [`requirement-summary.md`](requirement-summary.md) §2.

→ [`Steering/06-MODULE-BREAKDOWN.md`](Steering/06-MODULE-BREAKDOWN.md)

## 4. Non Functional Requirement — **DIPAHAMI** (dua angka terbuka) `[BISNIS]`

Profil beban: **data besar, konkurensi rendah** — 200–300 pengguna aktif harian, ribuan klaim
baru per bulan, data historis puluhan juta baris, ketersediaan 24/7, 2 VM on-premise.

Target persentil 95: layar sederhana < 1 s · inbox & pencarian < 3 s · simpan registrasi < 3 s ·
spreading < 1 s · laporan interaktif < 10 s · export besar asinkron · sistem eksternal batas 30 s.

`[TERBUKA]` Lama retensi audit (D-28, ke Compliance) dan angka RPO/RTO (D-29, ke tim infra).

→ `Steering/STEERING.md` §22–25 · [`Steering/15-NFR-PERFORMANCE-SCALABILITY.md`](Steering/15-NFR-PERFORMANCE-SCALABILITY.md)

## 5. Workflow — **DIPAHAMI, dengan satu temuan penting** `[KODE]`

`Register_Flow`: 23 shape, 30 konektor, 13 assignment, 7 decision. Percabangan utama PA /
Travel / Non-MBU, lalu Compliance / PUCL / Analyst Doctor / RCL Dokter.

> **Temuan yang mengikat desain:** alur nyata **bukan rangkaian linear.** Sebelas **ticket**
> memungkinkan lompatan langsung ke tahap mana pun tanpa melewati urutan. Model status di sistem
> baru **wajib mengizinkan transisi lateral** — memaksakan urutan kaku akan langsung ditolak user.

Dua konsep penugasan dipertahankan (D-26): **Worklist** (per orang) dan **Workbasket** (antrean
bersama). Pembagiannya terbukti dipakai nyata di `Register_Flow`, bukan warisan kosong.

Komite berjenjang sampai **4 level** (`komitepnc` → `komitepnc4`); tiap putaran menaikkan
`KomiteCount` dan mereset `AcceptStatus`.

→ [`Steering/02-BUSINESS-UNDERSTANDING.md`](Steering/02-BUSINESS-UNDERSTANDING.md) §2

## 6. Business Rules — **DIPAHAMI, tidak lengkap** `[KODE]`

Diekstraksi dari `InputRegister_act` (137 step) dan 70 when rule. Aturan yang paling sering
menolak input user berhasil diambil seluruhnya — lihat [`requirement-summary.md`](requirement-summary.md) §4.

Empat kelompok: aturan tanggal · duplikasi · reasuransi · nilai & notifikasi · kelengkapan.

Ambang yang ditemukan di kode: komite **Rp 50 juta** (umum), **Rp 30 juta** (PA/Travel),
**Rp 3.500** (server Timor-Leste); Notice of Large Losses **> Rp 1 miliar**. Diperlakukan sebagai
*nilai awal yang harus divalidasi ulang tim bisnis* (D-14), bukan sebagai kebenaran.

`[TERBUKA]` Tiga sumber aturan bisnis belum terbaca: 64 procedure/function database (R-01),
matriks komite di `POOLDATA.EMAILKOMITE` (D-14), dan `ValidasiSisaTSI` yang namanya menunjukkan
aturan sisa TSI yang **tidak muncul di analisis manapun** (R-07).

## 7. Validation Rules — **DIPAHAMI** `[KODE]`

Urutan tanggal wajib: **DOL ≤ Tanggal Lapor ≤ Tanggal Terima Dokumen ≤ hari ini**.
Total spreading **wajib 100%** (toleransi 99,99%) — submit ditolak bila tidak terpenuhi.
Penyebab Kerugian wajib kecuali Travel. Nomor SLIK wajib untuk SPK. Nilai klaim tidak boleh
melebihi TSI.

Perilaku pengumpulan kesalahan **dipertahankan**: seluruh kesalahan validasi dikumpulkan dan
dikembalikan bersamaan (HTTP `422`), tidak berhenti pada yang pertama. Ini kesetaraan perilaku
(P-5), bukan preferensi — form registrasi punya puluhan field.

→ `Steering/STEERING.md` §12 · [`requirement-summary.md`](requirement-summary.md) §5

## 8. Input — **DIPAHAMI** `[KODE]`

Snapshot polis (dibekukan saat registrasi, D-04) · data kejadian (DOL, tanggal lapor, tanggal
terima, penyebab kerugian, lokasi risiko) · objek pertanggungan dan coverage beserta TSI ·
spreading reasuransi/koasuransi · estimasi nilai · dokumen dan foto lapangan · hasil survei ·
keputusan komite · data rekening pembayaran.

Permukaan input: **74 harness, 269 section** (268 di antaranya bergrid).

## 9. Output — **DIPAHAMI** `[KODE]`

Nomor klaim `PNCN.YY.xxxx` · nomor akseptasi · **PLA → Pre-DLA → DLA** ke koasuransi/reasuransi ·
**LOD** ke tertanggung · 56 laporan · export PDF/Excel/CSV (dibuat sendiri di Go, D-11) ·
5 correspondence email · pelaporan **SLIK OJK** · data ke Kasir · umpan balik ke BRI Surf.

## 10. Database — **DIPAHAMI struktur, TERBUKA detail** `[KODE]`

**245 tabel** di 5+ schema: `POOLDATA` (bisnis inti), `DATAPEGA` (engine Pega), `GENERAL`
(storage & token), `GL`/`COLLECTION`/`MBU`/`ANEKA`/`HRDASM` (lewat DB Link).

Tabel terpenting: `T_CLAIM_PNC` (124×), `GCNM_PROGRESS_CLAIM` (119×),
`PC_ASM_FW_GCNMFW_WORK` (116× — milik engine Pega, **harus digantikan**),
`T_CLAIM_ADJUSTMENT` (92×).

> **Dua model penyimpanan hidup berdampingan.** Data klaim yang sama disimpan dua kali:
> tabel relasional `POOLDATA` **dan** dokumen JSON (`JSON_KLAIM`, `JSON_POLIS`) — 222 pemakaian
> `JSON_VALUE`/`JSON_TABLE` di 29 rule. Sinkronisasi oleh stored procedure, dan **tidak ada
> mekanisme yang memeriksa apakah keduanya konsisten** (R-10).

`[TERBUKA]` DDL — tipe kolom, panjang, index, constraint, foreign key — **tidak diketahui**
(R-08). Dampak nyata: kode lama memotong `ObjectName` pada 3.800 karakter "karena melebihi batas
database", dan batas sebenarnya tidak ada yang tahu.

→ `Steering/STEERING.md` §2.3, §18 · [`Steering/09-DATABASE-STRATEGY.md`](Steering/09-DATABASE-STRATEGY.md)

## 11. API — **DIPAHAMI** `[KODE]`

12 Connect-REST keluar. Sistem baru: REST, sumber daya jamak-kebab, respons berbentuk seragam,
**keyset pagination**, kontrak OpenAPI sebagai sumber tipe frontend, idempotensi pada operasi
yang mengubah uang.

→ `Steering/STEERING.md` §11 · [`Steering/10-API-STRATEGY.md`](Steering/10-API-STRATEGY.md)

## 12. Integration — **DIPAHAMI, enam belum ada** `[KODE]` `[BISNIS]`

| Sistem | Arah | Keperluan |
|---|---|---|
| BRI Surf | Keluar | OAuth token + umpan balik klaim |
| Arsip Dokumen (`app8`) | Keluar | Injeksi arsip dokumen |
| Storage Dokumen (`app13`) | Dua arah | Unggah/ambil dokumen (D-16) |
| Konversi Gambar | Keluar | Konversi AVIF |
| History Payment | Masuk | Riwayat pembayaran produksi |
| **HCC / HCQ** | Masuk | **Autentikasi user** (D-07) |
| **6 DB Link** | Masuk | HRD, jam kerja, GL payment, master sales, polis, buka proteksi |
| Kasir · SLIK OJK · SMTP | Keluar | Pembayaran, regulator, notifikasi |

`[TERBUKA]` **64 pemakaian DB Link ke 6 database** harus diganti API (D-25) — dan API-nya
kemungkinan besar belum ada, dibangun **tim lain dengan jadwal sendiri** (R-03). Dua titik paling
kritis: `GetIDCabang` (dipanggil 13 activity, termasuk jalur registrasi) dan
`GENERAL.MST_BUKA_PROTEKSI` — **satu-satunya objek remote yang menulis**, ke tiga sistem berbeda,
sehingga penggantinya wajib API idempoten dan tidak bisa dijembatani salinan berkala.

→ [`Steering/20-DETAIL-KOMITE-DBLINK.md`](Steering/20-DETAIL-KOMITE-DBLINK.md)

## 13. Dependency — **DIPAHAMI** `[KODE]` `[BISNIS]`

**Internal:** tiga rantai kritis yang tidak bisa diparalelkan —
`F-2 → B-1 → B-2 → B-3 → B-4 → B-5 → B-7 → B-10` (jalur nilai klaim) ·
`F-3 → F-4 → B-7` (komite butuh master ambang) ·
`U-2 → U-3/U-4/U-5` (semua layar butuh komponen tabel baku).

**Eksternal — dan ini yang menentukan jadwal:** DBA (source procedure, DDL, isi master) ·
Tim Pega (Rule-Agent, 3 router, 45 activity, pembekuan perubahan) · Tim pemilik 6 sistem
(API pengganti DB Link) · Tim GISFW (struktur polis) · Compliance (retensi) · Infra (RPO/RTO).

> **Sembilan dari sepuluh tindakan hari pertama bergantung pada pihak di luar tim pengembang.**
> Waktu tunggunya **tidak dapat dipercepat dengan menambah developer.**

## 14. Error Handling — **DIPAHAMI** `[KODE]`

Tiga jenis: validasi bisnis (`422`, dikumpulkan seluruhnya) · pelanggaran aturan/konflik (`409`,
satu kesalahan jelas) · kegagalan teknis (`500`, dicatat lengkap, pesan umum ke pengguna).

Kesalahan domain adalah **tipe, bukan string**. Detail internal tidak pernah bocor ke klien.
Pesan pengguna berbahasa Indonesia dan menjelaskan cara memperbaiki.

→ `Steering/STEERING.md` §12

## 15. Security — **DIPAHAMI** `[KODE]`

Autentikasi ke **HCC/HCQ** (mengembalikan profil lengkap: NIK, nama, cabang, jabatan, email);
otorisasi **dimiliki aplikasi** lewat tabel peran/izin baru yang belum ada. Parameter binding
tanpa perkecualian. Masking data sensitif dipertahankan. Akses data medis dibatasi Analyst Doctor
dan RCL Dokter. Jejak audit append-only ditegakkan lewat **hak akses database** — aplikasi hanya
diberi `INSERT` dan `SELECT`, sehingga tidak ada jalur di dalam aplikasi yang bisa menghapus
jejaknya sendiri.

→ `Steering/STEERING.md` §15–17 · [`Steering/11-SECURITY.md`](Steering/11-SECURITY.md)

## 16. Edge Case — **SEBAGIAN DIPAHAMI** `[KODE]` `[TERBUKA]`

Yang **sudah** diketahui: DOL sampai 30 hari setelah polis berakhir (Bonding) · tanggal cetak
sampai 90 hari setelah polis berakhir (Travel & PA) · Tanggal Lapor ≤ DOL+7 kecuali PA ·
duplikasi PA diperketat dengan penyebab kerugian `12002` · Ex-Gratia mengubah treaty `OR`→`ORS` ·
spreading `FlagDelete='1'` dibuang sebelum perhitungan · objek tanpa coverage dibuang otomatis ·
polis Deklarasi tidak dapat diklaim kecuali Aneka · `ObjectName` dipotong 3.800 karakter ·
hubungan tertanggung "lain-lain" (`7`) wajib keterangan.

`[TERBUKA]` Kasus tepi di dalam **64 procedure yang belum terbaca** tidak diketahui sama sekali.
Ini kelas ketidaktahuan yang tidak bisa diperkecil dengan analisis lebih lanjut — hanya dengan
mendapatkan source-nya.

## 17. Limitation — **DIPAHAMI** `[KODE]`

Pega PRPC 8.3 menyatukan UI dan logika dalam satu model rule · tanpa test otomatis · tanpa
CI/CD · tanpa caching sama sekali (7 data page seluruhnya `refresh=never`) · logika bisnis
tersebar di tiga tempat sekaligus (activity, SQL, stored procedure) sehingga satu aturan harus
dicari di tiga tempat · otorisasi bergantung pada penyembunyian menu, bukan pemeriksaan di
backend.

## 18. Technical Debt — **DIPAHAMI** `[KODE]`

Delapan temuan, masing-masing dengan bukti dari source dan jawaban desainnya:

| # | Utang teknis | Bukti | Dijawab |
|---|---|---|---|
| 1 | Kunci teknis Pega bocor ke data bisnis | `CLAIMID = 'ASM-FW-GCNMFW-WORK ' \|\| no_klaim` | D-22 |
| 2 | Alias kolom menyesatkan | `BUSINESSNAME AS "NOPOLIS"`, `NOPOLIS AS "NoKTP"`, `picteknik AS "UserAdmin"` | D-19 |
| 3 | Nilai bisnis di-hardcode | 10 alamat email, 4 user ID, 3 ambang, 3 hostname penentu perilaku | D-15 |
| 4 | **Blok `// TESTING` di jalur produksi** | `InputRegister_act` step 106–109 **menimpa email UW asli dengan email personal tester** | D-15 |
| 5 | Zona waktu ditangani manual | `+7 jam` di puluhan tempat, activity khusus `Set7Hours` | F-5 |
| 6 | **SQL dirangkai dari string** | `{ASIS:InputData.CARI4}`; klausa SQL disimpan sebagai nilai property | §9, §15 |
| 7 | Duplikasi masif per lini bisnis | `Browse*`/`Insert*` digandakan `_AsuransiKredit`, `_Travel`, `_Kredit_PA` — satu perubahan aturan harus diterapkan di empat tempat, dan sering hanya sebagian | Modul bersama |
| 8 | Salah ketik dipertahankan di nama rule | `Broswse*` (11 rule), `Complience`, `Proccedure`, `SALAVAGEDOCUMENT` | D-19 |

> Temuan #4 dan #6 adalah yang paling serius. #4 berarti notifikasi Underwriting untuk kerugian
> besar **kemungkinan tidak pernah sampai ke penerima yang benar** di produksi. #6 adalah celah
> **SQL injection** sekaligus penghalang portabilitas.

→ `Steering/STEERING.md` §2.4

## 19. Migration Risk — **DIPAHAMI** `[KODE]` `[BISNIS]`

**19 risiko** terdokumentasi (naik dari 12; `D-38`). Beberapa berdampak tinggi dan **sudah terjadi, bukan kemungkinan**:

| ID | Risiko | Dampak |
|---|---|---|
| **R-01** | Source 64 procedure & function tidak tersedia | **Memblokir 5 modul jantung nilai uang klaim**: B-5, B-7, B-9, B-10, B-12 |
| **R-02** | Job terjadwal tidak diketahui | **Kegagalan yang tidak terlihat** — ada proses otomatis berjalan hari ini yang tidak akan berjalan di sistem baru, tanpa ada yang menyadarinya sampai sesuatu tidak terkirim |
| **R-03** | 6 API pengganti DB Link belum ada | Bergantung tim lain dengan jadwal sendiri |
| **R-05** | Jadwal tidak sepadan dengan ukuran pekerjaan | Diterima manajemen (D-30); perkiraan terukur **9–15 bulan, tim 5–6 orang** |

→ [`Steering/16-RISK-ANALYSIS.md`](Steering/16-RISK-ANALYSIS.md) · [`migration-readiness.md`](migration-readiness.md)

---

## 20. Ukuran sistem yang dimigrasi

| Artefak | Jumlah |
|---|---|
| Rule XML | **2.634 berkas** + 55 `.prc` + 8 `.fnc` |
| Activity | 902 — **15.063 step**, 778 custom |
| Connect-SQL | 652 — 534 KB SQL |
| Section (komponen UI) | 269 — **268 bergrid** |
| Harness (layar) | 74 |
| Report Definition | 56 |
| Data Transform | 80 |
| When (business rule) | 70 |
| Flow Action · Ticket · Flow | 29 · 8 · 4 |
| Connect-REST | 12 |
| Tabel database | 245 di 5+ schema |
| Stored procedure & function | 70 (56 proc + 8 func lokal + 6 func remote) |
| DB Link | 6 — 64 pemakaian di 27 rule |
| Access group · menu portal | 22 · 47 |

---

## 21. Hasil pemeriksaan konsistensi (Phase 3)

Diperiksa pada 2026-09-08 terhadap seluruh berkas Steering dan `Steering/STEERING.md`.

| Diperiksa | Hasil |
|---|---|
| Angka procedure/function antar berkas | **Konsisten.** `Steering/STEERING.md` §2.3 menulis "56 procedure + 14 function" = 70 total; README dan R-01 menulis "64 objek lokal" (56 + 8) + 6 remote. Keduanya angka yang sama dibaca dari sudut berbeda — dan koreksi dari hitungan awal "86" sudah dicatat terbuka di D-02 |
| Istilah domain terhadap `CONTEXT.md` | **Konsisten.** Tidak ditemukan pemakaian `Adjustment` atau `Object` sebagai istilah kanonikal di dokumen manapun |
| Referensi silang D-xx dan R-xx | **Konsisten.** D-01…D-30 dan R-01…R-12 semuanya terdefinisi dan terpakai |
| D-06 berstatus SUPERSEDED oleh D-20 | **Benar dan ditandai eksplisit** |
| Keputusan yang berbeda dari rekomendasi analis | **Tercatat terbuka** di D-20, D-23, dan D-30 — beserta rekomendasi yang tidak dipilih dan alasannya |
| Pemetaan bab `Steering/STEERING.md` ke berkas `Steering/` | **Lengkap dan kini otomatis.** Riwayat Revisi + 16 bab + 6 lampiran, dibangun `docs/tools/build-steering.js` |

**Satu kontradiksi baru ditemukan pada sesi ini**, bukan di dalam Steering melainkan antara
Steering dan hasil wawancara:

> **I-01 menyatakan tidak ada tenggat eksternal** (bukan lisensi berakhir, bukan biaya, bukan
> arahan grup), sementara **D-30 menetapkan tenggat keras akhir September 2026** dan R-05
> mengukur pekerjaannya 9–15 bulan.
>
> Artinya tenggat itu **tidak dipaksa pihak luar** — sehingga penjadwalan ulang adalah keputusan
> yang sepenuhnya berada di dalam kendali manajemen Sinarmas, tanpa penalti kontraktual.
> Ini **bukan** alasan mengabaikan D-30, yang sudah ditegaskan ulang pemilik project dan tetap
> menjadi dasar Migration Strategy. Tetapi ia menghilangkan satu-satunya alasan yang akan
> membenarkan menerima risiko R-05 apa adanya, dan karena itu wajib disampaikan ke manajemen.

Dicatat sebagai risiko baru **R-13** di [`migration-readiness.md`](migration-readiness.md) §6 dan
[`BRD.md`](BRD/BRD.md) §20.3. Dua risiko baru lain lahir dari sesi yang sama: **R-14** (ukuran
keberhasilan tidak dapat dibuktikan untuk 5 modul) dan **R-15** (waktu user bisnis untuk UAT
belum dialokasikan).

---

## 22. Kesimpulan pemahaman

**Pemahaman bisnis: lengkap.** Proses, peran, aturan, aliran nilai uang, dan integrasi sudah
terpetakan dari source dan dikonfirmasi pemilik bisnis. Cukup untuk menyusun BRD.

**Pemahaman teknis: lengkap untuk merancang, tidak lengkap untuk mengimplementasikan.**
Arsitektur, domain model, batas modul, dan strategi migrasi sudah bisa diputuskan. Tetapi lima
modul yang merupakan **jantung nilai uang klaim** (B-5, B-7, B-9, B-10, B-12) tidak dapat
diselesaikan sampai source 64 procedure diterima.

**Enam aspek masih terbuka** — dan **tidak satu pun dapat ditutup oleh analisis lebih lanjut atas
export XML ini.** Semuanya memerlukan artefak dari pihak luar:

| Terbuka | Dari | Risiko |
|---|---|---|
| Isi 64 procedure & function | DBA | R-01 |
| Daftar job terjadwal | Tim Pega | R-02 |
| Kontrak 6 API pengganti DB Link | Tim pemilik sistem | R-03 |
| Aturan 3 router penugasan | Tim Pega | R-04 |
| ~~Arti kode status~~ — ✅ **diterima**, 33 kode `1134`–`1166` | ~~DBA~~ | ~~R-06~~ tertutup |
| DDL + statistik ukuran tabel | DBA | R-08 |

Status permintaan: **sudah dikirim, menunggu jawaban** (I-05).

Verdict kesiapan migrasi: [`migration-readiness.md`](migration-readiness.md).
