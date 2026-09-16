# Riwayat Revisi — v1.0 → v2.0

Bab ini menjawab satu pertanyaan bagi siapa pun yang sudah pernah membaca atau menyetujui versi
sebelumnya: **apa yang berubah, dan atas dasar apa.**

| | |
|---|---|
| **Versi sebelumnya** | v1.0 — 2026-09-07, bersumber dari Decision Log `D-01`…`D-30` |
| **Versi ini** | **v2.0 — 2026-09-14**, bersumber dari `D-01`…`D-72` |
| **Dasar perubahan** | 41 keputusan baru (`D-31`…`D-72`) · 14 temuan verifikasi (`T-1`…`T-14`) · 40 kontradiksi (`K-1`…`K-40`) · 29 ADR |
| **Kewenangan menyunting** | `D-72` — larangan menyunting `STEERING.md` dan berkas `.docx` dicabut |

---

## 1. Tiga hal yang paling berubah

### 1.1 Export bertambah dua kali, dan sebagian penghalang terbesar hilang

Snapshot v1.0 memakai **2.167 rule**. Pada 2026-09-09 export bertambah dan **tiga folder yang
sebelumnya tidak ada** muncul — `Database/`, `Job Scheduler/`, `Agents/`, ditambah
`Service REST/`. Snapshot yang berlaku sekarang memuat **2.634 berkas XML**, ditambah **55 `.prc`
dan 8 `.fnc`**.

Akibatnya, empat penghalang lama tertutup — tetapi **tujuh penghalang baru muncul**, dan tiga di
antaranya memblokir gerbang kelulusan, bukan sekadar memperlambat pengerjaan. Rinciannya di §4.

### 1.2 Aturan komite ternyata dibaca terbalik

Versi v1.0 menyebut jumlah jenjang komite ditentukan **matriks nilai × jenis bisnis** yang memilih
satu baris. Kenyataannya **kumulatif**: setiap jenjang yang ambang bawahnya sudah terlampaui
**ikut menyetujui**.

Ini nyaris "diperbaiki" menjadi salah. Usulan mengganti penyaring menjadi rentang tertutup akan
mengembalikan tepat satu baris — **satu jenjang persetujuan berapa pun nilai klaim** — dan
menghapus penjenjangan yang menjadi inti `D-14`. Koreksinya diambil sebelum jawaban diterapkan
(`D-47`).

### 1.3 Otorisasi: satuan izin adalah menu, bukan aksi

Versi v1.0 menetapkan izin **berbutir aksi** (`klaim.akseptasi`, `komite.setujui`, …). `D-59`
memutuskan sebaliknya: **satuan izin adalah menu**, dan **tidak ada pemisahan tugas formal**.

Yang tetap berubah dari sistem lama adalah **tempat penegakannya** — dari penyembunyian menu di
antarmuka menjadi pemeriksaan di server pada setiap endpoint. Konsekuensinya diterima secara
sadar, dan menaikkan `S-5` Jejak Audit dari modul pendukung menjadi **satu-satunya kontrol
pengimbang yang tersisa**.

---

## 2. Angka yang dikoreksi

Seluruhnya terverifikasi langsung dari export. Angka lama tidak salah untuk lingkupnya; ia
**terlalu sempit** atau bersumber dari dokumen, bukan dari bukti.

| Hal | v1.0 | **v2.0** | Sumber |
|---|---|---|---|
| Snapshot export | 2.167 rule | **2.634 berkas XML** + 55 `.prc` + 8 `.fnc` | `D-45` |
| Jumlah modul | 32 | **33** — `S-8` Perkakas Uji Kesetaraan | `D-42` |
| Jumlah risiko | 12 | **19** — menyerap `R-13`…`R-15`, menambah `R-16`…`R-19` | `D-38` |
| Format nomor klaim | `PNCN-xxxx` | **`PNCN.YY.xxxx`** | `D-71` |
| Kelompok master data | 14 | **≥29** | `T-13` |
| Kolom grid inbox | 18–27 | **median 6** | `T-11` |
| Alamat email hardcode | 10 | **66 unik** | `D-15` terverifikasi |
| Operator ID hardcode | 4 | **24 unik** | idem |
| Ambang komite hardcode | 3 | **8 unik** + 7 ambang uang non-komite | idem |
| Hostname penentu perilaku | 3 | **3** — angkanya benar, **daftarnya salah** | idem |
| Domain kode Status Klaim | `1142`–`1151` (10 kode) | **`1134`–`1166` (33 kode)** | `R-06` tertutup |
| Gap export | puluhan rule | **±242 rule**, termasuk **137 When rule** | `R-16` |
| Ticket rule | "11 ticket" | **17 dirujuk · 8 ada · 9 tanpa rule** | koreksi `FR-W2` |
| Objek database diminta | 86 procedure | **64 objek** diminta, **62 diterima** | `D-45` |
| Perbaikan eksplisit `P-5` | 4 butir | **13 butir** | `D-49` |
| Batas hasil laporan | — | `pyMaxRecords=500` pada **54 dari 56** laporan | `T-12` |
| `OFFSET 500000` | dipakai sebagai premis | **tidak berdasar** — `OFFSET` nol kemunculan | `T-12` |

---

## 3. Perubahan per bab

| Bab sumber | Yang berubah |
|---|---|
| `01-FRONTEND-ANALYSIS.md` | Banner koreksi: "18–27 kolom" → **median 6**; pilihan React tidak berubah, tetapi **ukuran `U-2` turun** dan pilihan pustaka tabel menjadi terbuka. Rujukan `D-20` di kepala dokumen dikoreksi menjadi **`D-23`** — keputusan frontend, bukan dialek SQL |
| `02-BUSINESS-UNDERSTANDING.md` | Alur Komite ditulis ulang sebagai **kumulatif**; ambang komite dikoreksi menjadi **8 unik** |
| `03-CURRENT-ARCHITECTURE.md` | Snapshot export diperbarui; `R-02` dan `R-06` ditandai **tertutup**; format nomor klaim |
| `04-FUTURE-ARCHITECTURE.md` | Seam **Identity** naik derajat — kontrak HCC/HCQ belum ada; pernyataan "scheduler cukup di dalam aplikasi" diturunkan menjadi **belum diputuskan** |
| `05-DOMAIN-MODEL.md` | **Status Klaim 33 kode** · aggregate **Komite ditulis ulang kumulatif** · invarian baru **`I-11`…`I-13`** (kurs, presisi uang, soft delete) · `I-1` memakai toleransi numerik tegas · master **≥29 kelompok** |
| `06-MODULE-BREAKDOWN.md` | **`S-8` ditambahkan, total 33 modul** · `S-8` masuk gelombang 1 · ukuran `F-3`, `F-4`, `U-2`, `U-6` dikoreksi · **rantai kritis menjadi empat** · tabel modul terhalang diperbarui menyeluruh |
| `07-MIGRATION-STRATEGY.md` | `P-5` **13 butir** · Tahap 0 diperbarui dengan status nyata tiap permintaan · verifikasi kesetaraan diarahkan ke `S-8` |
| `08-TECHNICAL-STRATEGY.md` | **Kepemilikan transaksi pindah ke Go**; `B-4`/`B-9` dapat dibuat atomik; kontrak galat `ErrMsg` tidak dibawa |
| `09-DATABASE-STRATEGY.md` | Generator nomor klaim **`PNCN.YY.xxxx`** · **§8.1 soft delete menyeluruh** · retensi audit mengikuti retensi klaim · **§9.1 prosedur perubahan skema** dan **§9.2 penulis tunggal** · premis `OFFSET` dikoreksi |
| `10-API-STRATEGY.md` | **§8.5 permukaan REST masuk** — 4 layanan, dua di antaranya menerima persetujuan komite dari sistem lain · `GET_WORKING_HOURS` **ditulis ulang di Go**, bukan dipanggil lewat API · status kontrak HCC/HCQ |
| `11-SECURITY.md` | **Satuan izin menjadi menu** (`D-59`) · **22 peran** (`D-58`) · HCC/HCQ **nol jejak** · daftar §5 diperluas dan angkanya dikoreksi · **§6 baru: data nasabah di lingkungan non-produksi** |
| `12-CROSSCUTTING.md` | Master data diperluas dengan ukuran terverifikasi · **§3.4 larangan hostname** dirinci · **§3.5 nol Dynamic System Setting** · **§3.6 rahasia masih terbuka** |
| `13-DEPLOYMENT.md` | **Staging memakai data produksi apa adanya, tanpa penyamaran** (`D-64`) — mengoreksi kewajiban penyamaran pada v1.0 · **§9 baru: job terjadwal pada dua instans** |
| `14-TESTING-STRATEGY.md` | **§6.1–§6.4** perkakas `S-8`, kewenangan menyetujui selisih, selisih yang sudah dapat diperkirakan, dan **dua hal yang tidak boleh dibandingkan** · **§10 dua gerbang penerimaan**, modul tanpa baseline, pencabutan `BRD §21.4` |
| `15-NFR-PERFORMANCE-SCALABILITY.md` | Premis `OFFSET` dikoreksi · **kapasitas laporan tidak punya dasar historis** karena batas 500 baris |
| `16-RISK-ANALYSIS.md` | **19 risiko** · `R-02` dan `R-06` **tertutup** · `R-01` sebagian tertutup · `R-04` **turun** menjadi verifikasi · `R-13`…`R-19` ditambahkan · **ringkasan tindakan hari pertama diperbarui** menjadi 15 baris berstatus |
| `17-FUTURE-ENHANCEMENT.md` | Rujukan kode status dan format nomor klaim disesuaikan |
| `18-SKILLS-USAGE-LOG.md` | **Sesi 3 ditambahkan** — `grilling`, `domain-modeling`, beserta **empat kesalahan sendiri dan tiga koreksi atas laporan sub-agen** |
| `19-GAP-EXPORT-DETAIL.md` | Banner: dokumen ini **jauh dari lengkap** — tujuh tipe rule tidak diauditnya sama sekali; penggantinya adalah **export ulang berbasis Product rule** |
| `20-DETAIL-KOMITE-DBLINK.md` | **§1.8 baru** — jawaban final setelah master diterima: mekanisme kumulatif, `TYPE_KOMITE` sebagai pita **hanya di Non-MBU**, dan tabel dampak bila diberlakukan seragam |
| `CONTEXT.md` | 61 → **70 istilah**; tidak ada `[TERBUKA]` tersisa |
| `00-DECISION-LOG.md` | `D-31`…`D-72` ditambahkan sebagai **Sesi 3**. **Entri `D-01`…`D-30` tidak disunting sama sekali** |

---

## 4. Penghalang: empat tertutup, tujuh baru

**Tertutup atau turun derajat:**

| Penghalang | Status |
|---|---|
| `R-02` job terjadwal tidak diketahui | **tertutup** — 5 job + 1 agent (`D-57`) |
| `R-06` arti kode status | **tertutup** — 33 kode diterima |
| `D-14` isi master komite | **tertutup** — 21 kolom, 30 baris |
| `R-01` source procedure | **sebagian** — 62 dari 64 diterima |
| `R-04` router penugasan | **turun** menjadi verifikasi — algoritma beban terbaca dari kueri lain |

**Baru, dan tiga di antaranya memblokir gerbang kelulusan:**

| Penghalang | Menghalangi | Pemilik |
|---|---|---|
| **Kontrak API HCC/HCQ** — nol jejak di export | `F-3`, dan login seluruh aplikasi | Tim HCC/HCQ |
| **Daftar peristiwa wajib audit** | `S-5` | Compliance |
| **Pega staging yang dapat ditembak dari luar** | `S-8`, karenanya **gerbang 1 seluruh modul** | Tim Pega + Infra |
| **12 dependensi** yang dipanggil 62 procedure (`UPDATE_LOG_KONVERSI` 162×) | `B-1` dan modul nilai uang | DBA |
| **±242 rule hilang**, 137 di antaranya When rule | percabangan bisnis hampir semua modul | Tim Pega |
| **Tujuan penyimpanan rahasia** (`D-40` masih `OPEN`) | `F-4`, `F-5`, seluruh deployment | Tim Infra / Security |
| **8 Ticket rule custom hilang**; 14 dari 17 nama tanpa pemicu | `B-7`, `B-11`, `B-13`, `B-14` | Tim Pega |

---

## 5. Yang **tidak** berubah

Dicatat agar tidak perlu dibaca ulang:

| Hal | Keterangan |
|---|---|
| Pilihan teknologi | Go modular monolith · React + TypeScript + Vite · PostgreSQL 17+ sebagai target · Oracle 19c sementara |
| Strategi migrasi | Strangler Fig, modul dialihkan bertahap, database bersama selama masa paralel |
| Batas bounded context | Data polis milik GISFW; Claim PNC menyimpan snapshot |
| Empat konsep status | **tetap empat** — pemilik bisnis menegaskan keempatnya memang berbeda; usulan menggabungkan **ditarik** |
| Worklist dan Workbasket | tetap dua model penugasan |
| Jadwal | **tidak berubah** (`D-61`) — selisih antara jadwal dan kesiapan menjadi tanggung jawab manajemen |
| Entri `D-01`…`D-30` | **tidak disunting sama sekali**; perubahan pikiran ditulis sebagai keputusan baru yang menyebut entri yang disupersede |

---

## 6. Cara dokumen ini dibangun

Sejak v2.0, **`STEERING.md` tidak disunting langsung**. Ia dibangun dari berkas sumber di
`docs/Steering/` oleh `docs/tools/build-steering.js`, sehingga tidak ada lagi dua versi pernyataan
yang sama yang bisa berbeda (`D-72`, Q35 Opsi 1).

**Suntingan manual pada `STEERING.md` akan hilang pada pembangunan berikutnya.** Yang disunting
adalah bab sumbernya.

Satu akibat langsung yang menutup keusangan terbesar: **Lampiran B kini memuat `D-01`…`D-73`**,
bukan berhenti di `D-30` seperti sebelumnya.

Berkas `.docx` lama disimpan sebagai arsip bertanda versi dan **tidak dihapus** (`D-72`, Q36
Opsi 3).

---

## 7. Koreksi angka setelah v2.0 — disetujui 2026-09-14

Ketiga angka berikut ditemukan keliru saat seluruh modul ditiketkan (`D-73`), dan **disetujui Work
Owner untuk diterapkan** ke Steering dan BRD.

| Angka | Tertulis sebelumnya | Terverifikasi | Cara memastikannya |
|---|---|---|---|
| Pemanggil `SendEmailNotification` | **17** | **15** | `19-GAP-EXPORT-DETAIL.md:196` menyebut kelima belas pemanggilnya satu per satu |
| Lokasi 3 password SMTP | **44** | **31** | `D-40` · `16-RISK-ANALYSIS.md:484` |
| Connect REST keluar | **12** | **21** | direktori `Connect REST/` dihitung langsung — 12 yang terhitung sejak awal ditambah **9 yang baru ditemukan** |

**Berkas yang disunting:** `06-MODULE-BREAKDOWN.md` (baris modul `S-4` dan baris risikonya) ·
`16-RISK-ANALYSIS.md` (tabel ukuran) · `BRD.md` (ringkasan ukuran, permukaan yang harus dibangun,
`FR-S4`, dan dasar perkiraan `§20.1`) · `verifikasi-bukti-adr.md` (dua baris `S-3`, satu baris
`S-4`) · `migration-readiness.md` · `requirement-summary.md`.

**Dua tempat sengaja TIDAK disunting**, karena keduanya adalah rekaman, bukan pernyataan fakta
yang berlaku:

| Tempat | Alasan |
|---|---|
| `00-DECISION-LOG.md:729` | Merekam **keberatan yang benar-benar diajukan** pada saat itu, beserta angka yang dipakai saat itu. Mengubahnya berarti memalsukan catatan rapat |
| `00-DECISION-LOG.md` tabel koreksi `D-73` | Tabel itu **justru berisi pasangan angka lama → angka baru**. Menggantinya akan menghapus jejak koreksinya sendiri |

**Yang berubah bukan hanya angka.** Koreksi ketiga **menambah lingkup `S-4`**: dari 12 integrasi
keluar menjadi 21, ditambah 4 layanan masuk yang belum pernah dihitung. Sembilan yang baru
ditemukan itu **belum dianalisis setara dengan 12 yang lama**, dan seluruhnya
ber-`pyUseAuthentication=false`. Penambahan ini tercermin di `BRD §20.1` sebagai dasar perkiraan,
bukan disembunyikan sebagai detail teknis.
