# Module Breakdown — Claim PNC

Pembagian modul beserta ukuran pekerjaannya. Ukuran diambil dari pemetaan 778 activity custom
(di luar 124 activity bawaan Pega), 652 rule SQL, 74 harness, dan 56 report definition.

Kolom **Bergantung pada** menentukan urutan pengerjaan: modul tidak bisa dimulai sebelum
dependensinya selesai.

> **Diperbarui v2.0 (2026-09-14).** Jumlah modul menjadi **33**, bukan 32 — `FR-S8` menjadi modul
> **`S-8` Perkakas Uji Kesetaraan** dan masuk **gelombang 1** (`D-42`), karena ia satu-satunya
> modul yang memblokir gerbang 1 setiap modul lain. Tiga ukuran dikoreksi terhadap bukti:
> **master data ≥29 kelompok** (bukan 14), **median kolom grid 6** (bukan 18–27), dan **daftar job
> terjadwal sudah diketahui** (`R-02` tertutup). Ringkasan di
> [`21-RIWAYAT-REVISI.md`](21-RIWAYAT-REVISI.md).

---

## 1. Modul fondasi — wajib selesai lebih dulu

Tidak ada modul bisnis yang bisa jalan tanpa lima modul ini.

| # | Modul | Isi | Ukuran | Bergantung pada |
|---|---|---|---|---|
| F-1 | **Kerangka Aplikasi** | Struktur folder, konfigurasi, logging, error handling, health check, graceful shutdown, penyajian SPA | Kecil | — |
| F-2 | **Akses Data** | Koneksi, connection pool, transaksi, seam Repository, SQL portabel, migrasi skema | Sedang | F-1 |
| F-3 | **Identitas & Akses** | Login via HCC/HCQ, session/token, **22 peran** dan izin menu, middleware otorisasi di **setiap endpoint** (D-58, D-59) | **Besar — tanpa baseline Pega** | F-1, F-2 |
| F-4 | **Master Data** | **≥29 kelompok master** (lihat Domain Model §5) + layar pengelolanya. Menghapus seluruh hardcode: **66 email · 24 user ID · 8 ambang komite · 3 hostname** (D-15) | **Besar** | F-1, F-2, F-3 |
| F-5 | **Waktu & Zona Waktu** | Seam Clock, penyimpanan UTC, konversi WIB tunggal, aturan hari kalender | **Sedang** — seam-nya kecil, tetapi migrasinya menyentuh **118 titik +7 jam di 36 activity** dan **101 titik +12 jam** | F-1 |
| F-6 | **Portal & Multi-Sumber Data** | Pilihan portal, koneksi per entitas, kewenangan per portal (`D-75`) — **4 portal**, satu database per entitas | Sedang | F-1, F-2, F-3 |

> **F-5 kecil tapi tidak boleh dilewati.** Ia menghapus 7-jam-manual yang tersebar di sistem
> lama. Bila dikerjakan belakangan, seluruh aturan tanggal harus ditulis ulang.

---

## 2. Modul bisnis inti

| # | Modul | Cakupan | Activity lama | Bergantung pada |
|---|---|---|---|---|
| B-1 | **Polis & Snapshot** | Ambil data polis, bekukan sebagai snapshot saat registrasi (D-04) | 18 | F-1…F-5 |
| B-2 | **Registrasi Klaim** | Validasi tanggal, duplikasi, kelengkapan; pembentukan objek dan coverage. Modul terdalam (137 step) | 22 | B-1 |
| B-3 | **Objek & Coverage** | Objek pertanggungan per lini bisnis, coverage, penyebab kerugian, rincian item | 44 | B-2 |
| B-4 | **Spreading Reasuransi** | Aturan total 100%, Fac Out, Ex-Gratia OR→ORS, share ASM | (bagian dari 69) | B-3 |
| B-5 | **Estimasi & Settlement** | Estimasi → Usulan → Akseptasi → Dibayar; salvage; risiko sendiri | 29 | B-3, B-4 |
| B-6 | **Penugasan & Inbox** | Worklist, Workbasket, routing, penguncian (D-26) | 30 | F-3 |
| B-7 | **Komite** | Penjenjangan 1–4 level, matriks nilai × jenis bisnis, jejak persetujuan | 34 | B-5, B-6, F-4 |
| B-8 | **Survey & Adjuster** | Penugasan surveyor, hasil survei, foto lapangan (mobile-friendly per D-12) | 30 | B-6 |
| B-9 | **PLA / Pre-DLA / DLA** | Pemberitahuan bertahap ke koasuransi/reasuransi | 69 (gabungan) | B-4, B-5 |
| B-10 | **Akseptasi & Pembayaran** | Nomor akseptasi, transfer kasir, status pembayaran, LOD | 31 | B-5, B-7 |
| B-11 | **RCL / PUCL / Compliance** | Penolakan, proses ulang, pemeriksaan kepatuhan, investigator, analyst doctor | 6 + jalur alur | B-6 |
| B-12 | **Salvage & Recovery** | Barang sisa, lelang, pemulihan, virtual account | 26 | B-5 |
| B-13 | **Open Protection** | Alur buka proteksi dan penautannya ke klaim | 12 | B-2 |
| B-14 | **Receive Document** | Penerimaan dokumen fisik, ekspedisi, resi | 8 | F-3 |

---

## 3. Modul pendukung

| # | Modul | Cakupan | Ukuran | Bergantung pada |
|---|---|---|---|---|
| S-1 | **Dokumen & Lampiran** | Unggah/ambil lewat API storage internal (D-16), kategori, kelengkapan dokumen | 54 activity | F-1, F-2 |
| S-2 | **Laporan & Export** | 56 laporan + engine PDF/Excel/CSV buatan sendiri (D-11) | 78 activity, 56 RD | seluruh modul bisnis |
| S-3 | **Notifikasi** | Seam Notifier, 5 correspondence, penerima dari master data | 14 activity | F-4 |
| S-4 | **Integrasi Eksternal** | **21 Connect REST keluar** + 6 API pengganti DB Link (D-25) + **4 layanan REST masuk** (`Service REST/`) | 7 activity + **21 REST** + 4 service | F-1 |
| S-5 | **Jejak Audit** | Pencatatan append-only seluruh perubahan bernilai bisnis (D-28) | — (baru, **tanpa baseline Pega**) | F-2 |
| S-6 | **Penjadwalan (Scheduler)** | **5 job terjadwal + 1 agent**, seluruhnya terverifikasi (D-57) | 5 job · 1 agent · 8 activity target | F-1 |
| S-7 | **Dashboard & Monitoring Bisnis** | Dashboard klaim, TAT, KPI, progres | (bagian dari 78) | S-2 |
| **S-8** | **Perkakas Uji Kesetaraan** | Menjalankan kasus yang sama di Pega staging dan Go staging, membandingkan hasil, **dan mengklasifikasikan setiap selisih** terhadap 13 butir `P-5` (D-42, D-54) | Besar — **baru 100%** | F-1, F-2 |

> **`S-8` masuk gelombang 1, bukan gelombang akhir.** Ia satu-satunya modul yang **memblokir
> cutover setiap modul lain**: gerbang 1 setiap modul adalah uji kesetaraan otomatis, sehingga
> menaruh `S-8` di belakang berarti tidak ada satu modul pun dapat lulus sampai gelombang itu tiba
> — membatalkan gagasan Strangler Fig (`D-05`). Lihat `ADR-0027`.
>
> **Dua modul tidak dapat memakai `S-8`**: `F-3` dan `S-5` tidak punya baseline Pega, sehingga
> gerbang 1 keduanya diganti uji fungsional terhadap kontrak (`D-56`, `ADR-0028`).

---

## 4. Frontend

| # | Modul | Cakupan | Ukuran |
|---|---|---|---|
| U-1 | **Kerangka SPA** | Routing, layout, autentikasi, penanganan error, state global | Sedang |
| U-2 | **Pustaka Komponen** | Tabel baku (**median 6 kolom**, paginasi server-side), form baku, unggah berkas, pemilih tanggal | **Sedang** |
| U-3 | **Layar Inbox** | **26** harness bernama *Inbox*, tetapi **Inbox = daftar pekerjaan pengguna** (`D-79`) — **7 di antaranya layar data acuan** dan diusulkan ke `U-6` | Besar |
| U-4 | **Layar Transaksi** | Registrasi, estimasi, survei, komite, akseptasi | Besar |
| U-5 | **Layar Laporan** | 56 laporan + unduhan | Besar |
| U-6 | **Layar Master Data** | Pengelolaan **≥29 kelompok master** | **Besar** |

**Total permukaan UI:** 74 harness, 269 section, 268 di antaranya bergrid.

> **U-2 adalah investasi terpenting di frontend.** 268 section memakai pola grid yang sama.
> Membuat satu komponen tabel baku yang benar akan dipakai ratusan kali (**leverage**), dan
> perbaikan bug di satu tempat memperbaiki seluruh layar (**locality**). Melewatkan U-2 berarti
> 268 implementasi tabel yang berbeda-beda — persis kegagalan yang harus dicegah pada tim di D-09.

> **Koreksi ukuran terhadap bukti (2026-09-14).**
>
> - **`U-2` turun dari Besar menjadi Sedang.** Spesifikasi "tabel baku 18–27 kolom" salah sasaran:
>   **median kolom sebenarnya 6**, dan tiga fitur grid yang biasanya paling mahal — tambah baris
>   inline, hapus baris inline, dan resize kolom — **tidak dipakai sama sekali** di seluruh export.
> - **`U-6` naik dari Sedang menjadi Besar**, dan `F-4` ikut naik: master data berjumlah
>   **≥29 kelompok**, bukan 14.
> - **Masalah paginasi bukan `OFFSET` besar.** `OFFSET` **nol kemunculan** di seluruh export;
>   yang nyata adalah **3.189 grid terikat page list klipboard** dan `pyMaxRecords=500` pada
>   **54 dari 56** laporan. Paginasi keyset karena itu **perubahan perilaku**, bukan pemeliharaan,
>   dan harus diuji per layar.

---

## 5. Urutan pengerjaan berdasarkan ketergantungan

```
Gelombang 1 — Fondasi          F-1 → F-2 → F-5 → F-3 → F-4
                               S-8 (perkakas uji kesetaraan)
                                       ↓
Gelombang 2 — Kerangka UI      U-1 → U-2          S-5 (jejak audit)
                                       ↓
Gelombang 3 — Jalur klaim      B-1 → B-2 → B-3 → B-4 → B-5
                               B-6 (penugasan) · B-14 (receive) · S-1 (dokumen)
                                       ↓
Gelombang 4 — Persetujuan      B-7 (komite) → B-10 (akseptasi) → B-11 (RCL/PUCL)
                               B-8 (survey) · B-13 (open protection)
                                       ↓
Gelombang 5 — Nilai & luar     B-9 (PLA/DLA) · B-12 (salvage) · S-3 · S-4
                                       ↓
Gelombang 6 — Laporan          S-2 · S-7 · U-5
                                       ↓
Gelombang 7 — Sisa             S-6 (scheduler) · U-6
```

**Empat rantai kritis** yang tidak bisa diparalelkan:
1. `F-2 → B-1 → B-2 → B-3 → B-4 → B-5 → B-7 → B-10` — jalur nilai klaim
2. `F-3 → F-4 → B-7` — komite butuh master ambang persetujuan
3. `U-2 → U-3/U-4/U-5` — semua layar butuh komponen tabel baku
4. **`S-8` → gerbang 1 seluruh modul** — tidak ada modul yang dapat dinyatakan lulus sebelum
   perkakas uji kesetaraan berjalan (`D-42`)

**`S-6` tidak lagi menunggu `R-02`.** Daftar job sudah lengkap (`D-57`); yang tersisa adalah
keputusan rancangan di `ADR-0022` — siapa yang menjalankan job saat aplikasi hidup di dua instans,
dan apakah `AutoAcceptKomite` yang menyetujui komite otomatis tiap hari jam 06:00 dipertahankan.

---

## 6. Modul yang terhalang informasi

Modul berikut **tidak dapat diselesaikan** sampai informasi yang hilang dilengkapi.

Diperbarui 2026-09-14 setelah export bertambah dan `Database/` diterima (`D-45`).

| Modul | Terhalang oleh | Risiko | Status |
|---|---|---|---|
| B-7 Komite | Isi tabel `POOLDATA.EMAILKOMITE` belum diambil | D-14 | ✅ **lepas** — 21 kolom, 30 baris diterima; model kumulatif ditetapkan (`ADR-0014`) |
| S-6 Scheduler | Rule Agent/Queue Processor tidak ada di export | R-02 | ✅ **lepas** — 5 job + 1 agent (`D-57`); sisa keputusan rancangan di `ADR-0022` |
| Seluruh modul berstatus | Arti kode status `1142`–`1151` belum ada | R-06 | ✅ **lepas** — 33 kode `1134`–`1166` diterima |
| B-6 Penugasan | `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter` tidak ada di export | R-04 | **Turun** — algoritma beban terbaca dari `BrowsePICRandomTeam-SQL.xml:39-40`; tinggal verifikasi |
| B-9, B-10, B-12 | Source procedure & function database belum ada | R-01 | **Lepas dari `BRD §21.4`** (`D-55`); sisa penghalang spesifik per modul |
| **B-5 Settlement** | isi `POOLDATA.GCNM_FEE_SCALE` (17 pita) · isi `m_currencystandard` · `BrowseT_Claim_Adjustment_SQL` | R-01, R-19 | **Masih terikat `BRD §21.4`** |
| Seluruh modul | **12 dependensi** yang dipanggil 62 procedure yang sudah diterima — terberat `UPDATE_LOG_KONVERSI` (162×) | R-01 | **Terbuka** |
| S-4 Integrasi | API pengganti 6 DB Link kemungkinan belum ada | R-03 | Terbuka · **lingkup bertambah**: 4 layanan REST masuk belum pernah dihitung, dan **9 Connect REST keluar baru ditemukan** — totalnya 21, bukan 12 (`D-73`) |
| S-3 Notifikasi | `SendEmailNotification` (dipanggil 15×) tidak ada di export | R-07 | Terbuka |
| **F-3 Identitas** | **kontrak API HCC/HCQ — nol jejak di export** | R-14, R-16 | **Terbuka** — `ADR-0024`; menghalangi login seluruh aplikasi |
| **F-3 Identitas** | penugasan operator ke peran **tidak ada di database**; 5 When rule menu hilang | R-16 | **Terbuka** — tabel izin dapat dibangun tetapi **tidak dapat diisi** |
| **F-4 Master Data** | tujuan penyimpanan rahasia belum ditetapkan | R-17 | **Terbuka** — `D-40` |
| **S-5 Jejak Audit** | **daftar peristiwa wajib audit** dari Compliance | R-14 | **Terbuka** — `ADR-0026` |
| **S-8 Perkakas Uji** | ketersediaan **Pega staging yang dapat ditembak dari luar** | R-14 | **Terbuka** — penghalang gerbang 1 seluruh modul |
| B-7, B-11, B-13, B-14 | **8 Ticket rule custom hilang**; 14 dari 17 nama tanpa pemicu di export | R-16 | **Terbuka** — `ADR-0021` |
| **B-2 Registrasi** | pengganti pola hapus-lalu-sisip-ulang belum diputuskan | — | **Terbuka** — `ADR-0013` |

> Ini bukan alasan menunda mulai — modul fondasi dan sebagian besar jalur klaim tidak terhalang.
> Tapi daftar ini menunjukkan **apa yang harus diminta sekarang juga**, karena waktu tunggunya
> ada di luar kendali tim pengembang.
>
> **Yang berubah sejak v1.0:** empat penghalang tertutup (`D-14`, `R-02`, `R-06`, sebagian `R-01`),
> satu turun derajat (`R-04`), dan **tujuh penghalang baru** muncul dari verifikasi bukti —
> seluruhnya menyangkut artefak atau keputusan yang sebelumnya tidak diketahui hilang.
