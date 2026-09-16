# CONTEXT — Ubiquitous Language, Domain Klaim PNC

Glossary bahasa domain untuk sistem Claim PNC. Dokumen ini **hanya** kamus istilah — tanpa
detail implementasi, tanpa nama tabel, tanpa keputusan teknis. Keputusan teknis ada di
`00-DECISION-LOG.md` dan dokumen Steering lainnya.

Sumber istilah: 902 activity, 652 SQL rule, 70 when rule, 4 flow Pega — dikonfirmasi langsung
oleh pemilik bisnis pada 2026-09-07.

Setiap istilah punya penanda asal:
`[BISNIS]` = dikonfirmasi pemilik bisnis · `[KODE]` = disimpulkan dari source · `[TERBUKA]` = belum pasti

---

## Lini Bisnis & Segmentasi

**MBU — Motor Business Unit** `[BISNIS]`
Lini bisnis kendaraan bermotor. Bukan cakupan utama aplikasi ini.

**Non-MBU** `[BISNIS]`
Seluruh lini bisnis di luar kendaraan bermotor. **Non-MBU inilah yang disebut PNC.**

**PNC** `[BISNIS]`
Sinonim dari Non-MBU. Nama aplikasi "Claim PNC" berarti *penanganan klaim untuk lini bisnis
non-motor*. Jangan mengartikan PNC sebagai singkatan teknis lain.

**Group Panel** `[KODE]`
Kode segmentasi lini bisnis yang menyetir hampir seluruh percabangan aturan — validasi,
routing, estimasi, komite, dan spreading:

| Kode | Lini |
|---|---|
| `002` | Personal Accident (PA) |
| `003` | Aneka |
| `004` | Marine Cargo |
| `005` | Travel |
| `006` | Fire / Property |
| `009` | Aneka (varian lain) |

**Business Type** `[KODE]`
Klasifikasi lebih rinci di dalam Group Panel: `MBUCar`, `MBUMotorCycle`, `Fire`, `MarineCargo`,
`HE`, `ContractorsPM`, `Bonding`, `BondingKBG`, `PAYDI_PA`, `Travel`.

**TKA — Tenaga Kerja Asing** `[BISNIS]`
Lini bisnis asuransi untuk pekerja warga negara asing.

**SPK — Sinarmas Penjaminan Kredit** `[BISNIS]`
Lini bisnis penjaminan kredit. Di kode muncul sebagai "Asuransi Kredit".
Punya kewajiban khusus: **Nomor SLIK wajib diisi** saat registrasi.

---

## Inti Klaim

**Klaim (Claim)** `[BISNIS]`
Pengajuan ganti rugi atas satu polis akibat satu peristiwa kerugian. Satu klaim mencakup satu
atau lebih Objek Pertanggungan.

**Polis (Policy)** `[BISNIS]`
Kontrak asuransi. Dimiliki oleh domain lain (GISFW, tim berbeda). Domain Klaim hanya menyimpan
**Snapshot Polis**.

**Snapshot Polis** `[BISNIS]` (D-04)
Salinan data polis pada saat klaim diregistrasi. Setelah snapshot diambil, klaim tidak lagi
terpengaruh perubahan polis. Ini yang membuat domain Klaim berdiri sendiri.

**Objek Pertanggungan (Insured Item)** `[KODE]`
Barang atau orang yang dipertanggungkan dan terdampak kerugian — kendaraan, orang, properti,
kargo, atau lokasi. Di sistem lama bernama `ObjectList`.
→ Istilah `Object` sengaja **dihindari** di sistem baru karena bertabrakan dengan makna
pemrograman.

**Coverage (Jaminan)** `[KODE]`
Jenis jaminan yang melekat pada satu Objek Pertanggungan, beserta nilai TSI dan Penyebab
Kerugian yang berlaku.

**TSI — Total Sum Insured** `[KODE]`
Nilai pertanggungan. Nilai klaim tidak boleh melebihi TSI — divalidasi saat registrasi.

**Sisa TSI** `[BISNIS]`
Bagian TSI yang belum terpakai untuk satu Objek Pertanggungan pada satu Coverage: TSI dikurangi
akumulasi nilai akseptasi yang masih **Outstanding**, lalu **ditambah** nilai **Salvage** —
salvage memulihkan kapasitas pertanggungan. Nilai usulan penyelesaian tidak boleh melebihi sisa
TSI. Berlaku pada lini Personal Accident.

**Cause of Loss (Penyebab Kerugian)** `[KODE]`
Sebab terjadinya kerugian. Wajib diisi kecuali untuk lini Travel.

**Date of Loss (DOL) — Tanggal Kejadian** `[KODE]`
Tanggal peristiwa kerugian terjadi. Titik acuan hampir seluruh aturan tanggal.

**Report Date — Tanggal Lapor** `[KODE]`
Tanggal tertanggung melaporkan kerugian.

**Date Received — Tanggal Terima Dokumen** `[KODE]`
Tanggal dokumen klaim diterima.

> Urutan wajib: **DOL ≤ Tanggal Lapor ≤ Tanggal Terima Dokumen ≤ hari ini**

---

## Nilai & Penyelesaian

**Estimasi Klaim (Claim Estimate)** `[KODE]`
Perkiraan awal nilai kerugian saat registrasi. Dipakai untuk PLA dan untuk memicu notifikasi
kerugian besar.

**Settlement Line** `[KODE]`
Baris penyelesaian nilai untuk satu Coverage. Di sistem lama bernama `AdjustmentList`.
Menyimpan perjalanan nilai uang klaim:

`Estimasi → Usulan (Propose) → Akseptasi (Accepted) → Dibayar (Paid)`

> Istilah lama `Adjustment` menyesatkan karena dalam praktik asuransi "adjusting" berarti proses
> penilaian kerugian, sedangkan di sini isinya adalah **nilai penyelesaian**. Di sistem baru
> dipakai istilah **Settlement**.

**Akseptasi** `[BISNIS]`
Persetujuan atas **nilai** klaim yang akan dibayarkan. Menghasilkan **Nomor Akseptasi**.
Berbeda dari persetujuan bahwa klaim itu dijamin (liability).

**OS — Outstanding** `[BISNIS]`
Klaim yang sudah diakui nilainya tetapi belum selesai dibayar. "OS Akseptasi" = akseptasi
yang masih menggantung.

**Ex-Gratia** `[KODE]`
Pembayaran kebijakan di luar kewajiban polis. Bila klaim ditandai ex-gratia, tipe treaty
`OR` otomatis berubah menjadi `ORS`.

**Salvage** `[KODE]`
Nilai sisa barang rusak yang bisa dijual kembali (termasuk lewat balai lelang). Mengurangi
nilai bersih klaim.

**Recovery** `[KODE]`
Pemulihan dana dari pihak ketiga, termasuk lewat Virtual Account.

**Notice of Large Losses** `[KODE]`
Pemberitahuan wajib ke Underwriting dan jajaran pimpinan bila estimasi klaim (setelah konversi
kurs) melebihi **Rp 1.000.000.000**.

**Kurs Standar** `[BISNIS]` (D-48)
Nilai tukar yang dipakai untuk mengubah nilai klaim mata uang asing menjadi Rupiah. Kurs yang
berlaku adalah kurs **pada tanggal kejadian** — bukan kurs saat klaim diproses atau dibayar,
sehingga nilai Rupiah sebuah klaim tidak berubah karena keterlambatan proses. Bila kurs untuk
tanggal itu tidak tersedia, klaim **ditolak dengan pesan yang menyebutkan mata uang dan
tanggalnya** — tidak ada kurs pengganti dan tidak ada nilai bawaan.

---

## Reasuransi & Koasuransi

**Spreading** `[KODE]`
Pembagian risiko satu Coverage ke para penanggung. **Total share wajib 100%** (toleransi
Pembagian risiko satu Coverage ke para penanggung. **Total share wajib 100%**, **toleransi 4 desimal, `99,9999`–`100,0001`** (`D-51`) —
divalidasi saat registrasi dan menolak submit bila tidak terpenuhi. Angka `99,99` yang dulu
lolos kini **ditolak** — selisih yang direncanakan (`D-49` butir 1).
**Koasuransi (Coins)** `[KODE]`
Pembagian risiko antar sesama perusahaan asuransi. Ada peran Leader dan Member.

**Reasuransi (Reinsurance)** `[KODE]`
Pengalihan risiko ke perusahaan reasuransi.

**Treaty** `[KODE]`
Perjanjian reasuransi otomatis. Tipe yang muncul: `OR`, `ORS`, Fac Out (`10015`), Quota Share,
XOL, BPPDAN.

**BPPDAN — Badan Pengelola Pusat Data Asuransi Nasional** `[BISNIS]`
Salah satu jenis treaty asuransi.

**Fac Out (Facultative Outward)** `[KODE]`
Pengalihan risiko per kasus, bukan otomatis. Bila ada spreading Fac Out, data Fac Offer wajib
lengkap — termasuk Object Name untuk Group Panel `003`.

**XOL — Excess of Loss** `[KODE]`
Treaty non-proporsional yang menanggung kerugian di atas batas tertentu.

**PLA — Preliminary Loss Advice** `[BISNIS]`
Pemberitahuan **nilai estimasi** klaim kepada koasuransi/reasuransi.

**Pre-DLA** `[BISNIS]`
Pemberitahuan nilai klaim yang **akan** diakseptasi kepada koasuransi/reasuransi, dikirim
**sebelum** akseptasi dilakukan.

**DLA — Definite Loss Advice** `[BISNIS]`
Pemberitahuan **nilai akseptasi** klaim kepada koasuransi/reasuransi.

> Urutan pemberitahuan: **PLA** (estimasi) → **Pre-DLA** (akan diaksep) → **DLA** (sudah diaksep)

**KBRU — Kali Besar Raya Utama** `[BISNIS]`
Salah satu broker asuransi yang berintegrasi dengan sistem.

---

## Proses & Peran

**Register (Registrasi Klaim)** `[KODE]`
Tahap awal: pencatatan klaim, validasi tanggal, cek duplikat, penetapan objek dan coverage,
serta spreading. Gerbang validasi terberat di seluruh sistem.

**Open Protection (Buka Proteksi)** `[KODE]`
Permintaan pembukaan proteksi sebelum/di luar alur klaim normal. Satu Open Protection dapat
ditautkan ke klaim dan ditandai terpakai.

**Receive Document** `[KODE]`
Pencatatan penerimaan dokumen fisik klaim, termasuk pengiriman antar cabang (ekspedisi, no.
resi, estimasi tiba).

**Surveyor / Loss Adjuster** `[KODE]`
Pihak yang meninjau kerugian di lapangan. Bisa internal (ASM) atau eksternal. Menghasilkan
hasil survei dan foto dokumentasi.

**Komite (Committee)** `[BISNIS]`
Forum persetujuan berjenjang atas nilai klaim. Jumlah jenjang ditentukan **kombinasi nilai klaim
dan jenis bisnis** (D-14), dan bersifat **kumulatif**: makin besar nilai klaim, makin banyak
jenjang yang harus menyetujui (D-47). Komite dapat menyetujui, menolak, atau mengembalikan.

**Pita Nilai Komite** `[BISNIS]` (D-52, D-70)
Pengelompokan klaim **Non-MBU** menurut besar nilainya, yang menentukan **rangkaian jenjang
mana** yang menangani klaim itu. Dua pita: sampai **Rp 100.000.000**, dan di atas
**Rp 100.000.000**. Pita dipilih **lebih dulu**, sebelum jenjang mana pun dihitung, dan setiap
pita punya rangkaian jenjangnya sendiri. Di sistem lama pita ini dipilih berdasarkan nama orang;
di sistem baru diturunkan dari nilai klaim.

> Istilah ini **hanya berlaku di lini Non-MBU** (D-70). Lini lain tidak mengenal pita nilai —
> jenjangnya satu rangkaian utuh, dibedakan hanya oleh ambang bawah.

**Jenjang Kumulatif** `[BISNIS]` (D-47, D-70)
Cara menghitung berapa banyak persetujuan yang dibutuhkan sebuah klaim. Setiap jenjang punya
**ambang bawah** — nilai klaim minimum yang membuatnya ikut campur. Semua jenjang yang ambang
bawahnya **sudah terlampaui** nilai klaim harus menyetujui, bukan hanya jenjang tertinggi.
Akibatnya jumlah penyetuju **bertambah** seiring besarnya klaim, bukan berpindah orang: klaim
yang melampaui satu ambang butuh satu persetujuan, klaim yang melampaui tiga ambang butuh tiga.

Di lini **Non-MBU** akumulasi itu didahului satu langkah: **pita dipilih lebih dulu**, lalu
akumulasi berjalan **di dalam pita itu saja** dan tidak menyeberang ke pita lain. Di lini lain
tidak ada langkah pendahuluan — akumulasi langsung berjalan atas seluruh jenjang lini tersebut.

**Batas atas** setiap jenjang tetap dicatat di master, tetapi hanya untuk memastikan pita dan
jenjang tidak bertumpang tindih dan tidak berlubang saat master diisi — batas atas tidak pernah
menentukan siapa yang menyetujui sebuah klaim.

**RCL — Rejected Klaim** `[BISNIS]`
Klaim yang ditolak.

**RCL Dokter** `[KODE]`
Penolakan yang memerlukan pertimbangan medis, dipakai pada lini Personal Accident.

**PUCL — Proses Ulang Klaim** `[BISNIS]`
Klaim yang diproses ulang setelah sebelumnya ditolak atau ditutup.

**Compliance Check** `[KODE]`
Pemeriksaan kepatuhan sebelum klaim diselesaikan.

**Investigator** `[KODE]`
Peran yang menyelidiki klaim yang mencurigakan.

**Analyst Doctor** `[KODE]`
Peran tenaga medis yang menilai klaim kesehatan/kecelakaan diri.

**PIC Teknik / User Teknis** `[KODE]`
Penanggung jawab teknis klaim dari sisi keahlian lini bisnis.

**User Admin** `[KODE]`
Petugas administrasi yang melakukan registrasi dan input data klaim.

**LOD — Letter of Discharge** `[BISNIS]`
Surat pemberitahuan nilai ganti rugi yang disetujui asuransi kepada **tertanggung**.
Berbeda dari PLA/DLA yang ditujukan ke koasuransi/reasuransi.

**TAT — Turn Around Time** `[KODE]`
Waktu penyelesaian klaim, dipantau lewat laporan TAT.

**SLIK OJK** `[KODE]`
Pelaporan ke Sistem Layanan Informasi Keuangan OJK. Wajib untuk lini SPK / Asuransi Kredit —
Nomor SLIK tidak boleh kosong.

**Tugas (Assignment)** `[BISNIS]` (D-26)
Satu pekerjaan yang menunggu dikerjakan orang pada satu tahap klaim. Sebuah klaim melewati
banyak tugas berturut-turut; satu tugas selalu berada di **Worklist** atau di **Workbasket**,
tidak pernah di keduanya.

**Worklist** `[BISNIS]` (D-26)
Daftar tugas milik **satu orang tertentu**. Tugas di dalamnya sudah punya pemilik dan hanya
dikerjakan orang itu. Dipakai untuk tahap yang butuh kesinambungan penanganan — antara lain
Input Register, Estimasi, pemilihan surveyor, dan penilaian Analyst Doctor.

**Workbasket** `[BISNIS]` (D-26)
Antrean tugas **bersama** yang belum bertuan: siapa pun yang berwenang atas antrean itu boleh
mengambilnya. Dipakai untuk tahap yang dikerjakan sebuah tim, antara lain RCL/PUCL,
Investigator, dan Compliance.

**Inbox** `[BISNIS]` (D-79)
**Layar yang menampilkan daftar pekerjaan milik seorang pengguna** — yakni **Tugas** dari
**Worklist** atau **Workbasket** yang menunggu dikerjakan.

Yang membuatnya Inbox bukan bentuknya, melainkan isinya. Empat ciri yang membedakannya dari layar
daftar biasa: barisnya adalah **pekerjaan**, bukan data acuan · baris **hilang** setelah selesai
dikerjakan · "hanya milik saya" adalah **aturan kewenangan**, bukan sekadar penyaring · dan
barisnya punya **tenggat**.

**Layar yang menampilkan data acuan (master) bukan Inbox**, sekalipun dapat dicari dan sekalipun
namanya di sistem lama mengandung kata "Inbox". Dari 26 harness bernama *Inbox* di export,
sebagian bersidik jari layar master — lihat Lampiran Inventaris 74 Harness.

---

## Status — empat konsep berbeda

Dikonfirmasi pemilik bisnis (D-18): keempatnya **memang berbeda** dan semuanya dipertahankan.
Di sistem baru masing-masing diberi nama yang tidak lagi bisa tertukar.

**Status Proses** — sistem lama: `StatusWork` `[BISNIS]`
Posisi klaim dalam alur kerja: sedang berjalan, selesai, atau ditolak.

**Status Klaim** — sistem lama: `StatusClaim`, kode `1134`–`1166` `[BISNIS]`
Status bisnis klaim. **33 kode**, masing-masing berlabel di master status. Sebelas kode pertama
(`1134`–`1144`) membawa penomoran lama `01`–`11`; sisanya hanya bernomor baru.
Contoh: `1142` Rejected Claim · `1143` Close Claim for this object · `1144` Cancelled Claim ·
`1147` Register · `1149` Claim Committee · `1163` Paid · `1164` Reopen Claim.

> Rentang `1142`–`1151` yang dipakai dokumen-dokumen awal ternyata **sebagian** dari domainnya,
> bukan keseluruhannya.

**Flag Klaim** — sistem lama: `ClaimStatus` (`0`/`1`) `[BISNIS]`
Penanda biner. Makna persisnya masih perlu dikonfirmasi.

**Status Posisi Progres** — sistem lama: `StatusPosisi` (`On Progress` / `Done`) `[BISNIS]`
Posisi klaim pada rangkaian tahapan progres yang dicatat terpisah dari alur kerja utama.

---

## Istilah yang mudah tertukar

**Lompatan Lateral (Lateral Transition)** `[KODE]` — sistem lama: **Ticket rule**
Perpindahan klaim langsung ke sebuah tahap yang **bukan** tahap berikutnya dalam urutan normal —
misalnya klaim dikembalikan dari Komite ke petugas estimasi begitu seluruh anggota komite
menyetujui, atau klaim berpindah ke PUCL saat ditolak pada Group Panel tertentu. Di sistem lama
mekanisme ini bernama **Ticket rule**: sebuah nama tujuan dipasang pada tahap tertentu, lalu
dipicu dari tempat lain sehingga klaim "melompat" ke sana.

> **Jangan tertukar dengan tiket pekerjaan.** Di seluruh dokumen proyek ini, kata **"tiket"**
> tanpa keterangan lain berarti **tiket pekerjaan** — satuan pekerjaan yang dikerjakan di
> `docs/ticketing/`. Mekanisme Pega di atas **selalu** disebut lengkap sebagai **"Ticket rule"**,
> tidak pernah sekadar "ticket" atau "tiket". Folder `Ticket/` pada export berisi Ticket rule,
> bukan tiket pekerjaan.

**Tiket Pekerjaan** `[BISNIS]` (D-31)
Satuan pekerjaan migrasi yang dapat dikerjakan dan dinyatakan selesai secara mandiri, disimpan
di `docs/ticketing/`. Tidak ada hubungannya dengan klaim maupun dengan Ticket rule.

**Master Data** `[BISNIS]` (D-73)
Data acuan yang dipakai seluruh sistem dan berubah jauh lebih jarang daripada data klaim — antara
lain rekening, supplier, bengkel, sparepart, panel, sebab kerugian, pasal penolakan, ambang komite,
dan kurs standar. Berjumlah **sekurang-kurangnya 29 kelompok**; jumlah pastinya
**BELUM DIPUTUSKAN — pertanyaan terbuka** (pemilik: Work Owner). Sebagian di antaranya menentukan
hasil hitungan uang, sehingga perubahannya berakibat pada nilai klaim yang dihitung sesudahnya.

**Correspondence** `[KODE]` (D-73)
Nama mekanisme Pega untuk **isi surat dan email** yang dikirim sistem. Tidak ada satu pun direktori
`Correspondence/` di export, sehingga seluruh isinya tidak diketahui (`R-16`). Jumlahnya pun belum
pasti — dua dokumen internal menyebut angka berbeda (3 dan 5).

**Notifier** `[KODE]` (D-73)
Nama seam di sistem baru yang melepaskan **peristiwa domain** — bukan perintah "kirim email ke
alamat ini". Pemanggil menyatakan *apa yang terjadi*; penerimanya ditentukan dari Master Data.
Menggantikan `SendEmailNotification` yang dipanggil 15 activity dan **tidak ada di export**.

**Job Terjadwal** `[BISNIS]` (D-57, D-73)
Pekerjaan yang berjalan sendiri pada jam tertentu **tanpa ada orang yang menekan tombol**. Ada
lima di sistem lama, ditambah satu **Agent** yang berjalan tiap 30 menit. Karena tidak ada pengguna
yang bertindak, perubahan data olehnya hanya dapat ditelusuri lewat jejak audit.

**Uji Kesetaraan** `[BISNIS]` (D-42, D-54)
Menjalankan **kasus yang sama** di Pega dan di sistem baru, lalu membandingkan hasilnya. Setiap
selisih wajib **diklasifikasikan** terhadap 13 butir perbaikan `P-5`: yang cocok lolos otomatis,
yang di luar itu menunggu persetujuan Work Owner tertulis. Dikerjakan modul `S-8`.

**Gerbang 1** `[BISNIS]` (BRD §21.1, D-42)
Gerbang kelulusan pertama sebuah modul: hasilnya **setara dengan Pega** menurut Uji Kesetaraan.
Tidak ada modul yang dapat dinyatakan lulus sebelum perkakas `S-8` berjalan. Dua modul **tidak
punya baseline Pega** untuk diuji — `F-3` (identitas) dan `S-5` (jejak audit) — sehingga bagi
keduanya gerbang ini diganti ukuran lain.

**Gerbang 2** `[BISNIS]` (BRD §21.1)
Gerbang kelulusan kedua: **UAT oleh peran bisnis** yang benar-benar memakai modul itu, bukan oleh
pengembang. Setiap tiket menyebut peran pengujinya di kepala tiket.

**Portal** `[BISNIS]` (D-75)
Pilihan **entitas** di dalam satu aplikasi. Setiap portal menampilkan data dari **database
entitasnya sendiri** — satu database per entitas, bukan satu database bersama dengan penanda
entitas. Minimal empat: Asuransi Sinar Mas, Asuransi Simas Insurtech, Sinarmas Asuransi Syariah,
dan Timor-Leste. Jumlah pastinya **BELUM DIPUTUSKAN — pertanyaan terbuka** (pemilik: Work Owner),
karena itu daftarnya diperlakukan sebagai **data**, bukan konstanta di kode.

Di sistem lama tidak ada istilah maupun pilihan ini: perilaku entitas ditentukan dengan
**membandingkan nama server** (48 perbandingan terhadap `pxRequestor.pxReqServer`). Lihat
[[uji-kesetaraan]] — portal Syariah belum punya baseline utuh karena rule pembedanya hilang.

---

## Istilah yang sengaja tidak dipakai lagi

| Istilah lama | Alasan ditinggalkan | Pengganti |
|---|---|---|
| `Object` / `ObjectList` | Bertabrakan dengan makna pemrograman | Objek Pertanggungan / Insured Item |
| `Adjustment` / `AdjustmentList` | Menyesatkan; isinya nilai penyelesaian, bukan proses adjusting | Settlement Line |
| `CaseID` | Dipakai untuk 5 hal berbeda di SQL yang berbeda | Nama eksplisit per konteks |
| `pzInsKey` / `CLAIMID` berformat `ASM-FW-GCNMFW-WORK PNC-xxxx` | Kunci teknis Pega bocor ke data bisnis | Nomor Klaim sebagai identitas bisnis |
| Alias kolom SQL warisan (`LOCATION AS "RISKLOCATION"`, `NOPOLIS AS "NoKTP"`, `picteknik AS "UserAdmin"`) | Nama tidak mencerminkan isi; dipaksa agar cocok dengan clipboard Pega | Penamaan ulang menyeluruh |
