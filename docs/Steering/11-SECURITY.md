# Security, Authentication & Authorization

Mengacu pada D-07: **autentikasi didelegasikan ke API internal HCC/HCQ**, **otorisasi dimiliki
sepenuhnya oleh aplikasi Claim PNC**. Ditambah kewajiban jejak audit dari D-28.

> **Diperbarui v2.0 (2026-09-14).** Tiga perubahan besar di bab ini: **satuan izin adalah menu,
> bukan aksi** (`D-59` — mengoreksi §3.1 versi sebelumnya) · **peran ditetapkan 22, satu-untuk-satu
> dengan access group** (`D-58`) · dan **HCC/HCQ ternyata nol jejak di export**, sehingga
> autentikasi menjadi integrasi greenfield tanpa baseline (`T-2`, `ADR-0024`). Ditambah bab baru
> **§6 Data nasabah di lingkungan non-produksi** (`D-64`, `D-69`), dan daftar §5 yang angkanya
> dikoreksi terhadap bukti.

---

## 1. Pemisahan yang mendasar

| | Authentication | Authorization |
|---|---|---|
| Menjawab | Siapa pengguna ini? | Boleh melakukan apa? |
| Pemilik | **HCC/HCQ** (sistem lain) | **Claim PNC** (kita) |
| Sumber data | API HCC/HCQ | Tabel milik aplikasi |
| Bila sistem lain mati | Pengguna baru tidak bisa login | Tidak terpengaruh |

Pemisahan ini adalah keputusan yang baik dan patut dipertahankan. Sistem lama menggabungkan
keduanya di Pega lewat 22 access group, sehingga menambah satu peran berarti mengubah konfigurasi
platform. Dengan pemisahan ini, peran dan izin menjadi **data biasa yang bisa diubah tanpa
deploy** (D-15).

---

## 2. Authentication

### 2.1 Alur

1. Pengguna memasukkan username dan password di SPA.
2. Backend meneruskannya ke **API HCC/HCQ**.
3. HCC/HCQ mengembalikan **profil lengkap**: NIK, nama, cabang, jabatan, email (D-07).
4. Backend mencari atau membuat catatan pengguna lokal berdasarkan NIK.
5. Backend menerbitkan **token session miliknya sendiri**.
6. Token dipakai untuk seluruh permintaan berikutnya.

**Backend menerbitkan token sendiri, bukan meneruskan token HCC/HCQ.** Alasannya: masa berlaku,
pencabutan, dan isi token menjadi kendali kita; aplikasi tidak bergantung pada HCC/HCQ untuk
setiap permintaan; dan bila HCC/HCQ sedang bermasalah, pengguna yang sudah login tetap bisa
bekerja.

> **Temuan yang mengubah derajat alur ini (2026-09-14).** `HCC` dan `HCQ` muncul **2× di seluruh
> export**, dan **keduanya teks pesan galat** yang menyuruh pengguna menghubungi helpdesk. Tidak
> ada Connect REST ke HCC/HCQ, tidak ada pemetaan field respons, tidak ada penanganan kegagalan
> autentikasi, dan tidak ada mekanisme sesi yang dapat dijadikan pembanding.
>
> Artinya alur enam langkah di atas **bukan pemindahan perilaku lama, melainkan integrasi
> greenfield sepenuhnya** — dan `F-3` tidak punya baseline untuk diuji kesetaraannya (`D-56`).
> Empat hal karenanya belum dapat ditetapkan dan menunggu Tim HCC/HCQ (`ADR-0024`, `Proposed`):
> bentuk kontraknya, perilaku saat HCC/HCQ tidak dapat dihubungi, **cara mencocokkan identitas
> HCC/HCQ dengan `OPERATOR_ID` yang dipakai di seluruh data klaim**, dan masa berlaku sesi.
>
> Butir ketiga yang paling mudah terlewat: tanpa pemetaan itu, pengguna yang berhasil login tetap
> **tidak dikenali oleh data klaimnya sendiri**.

### 2.2 Aturan token

| Aturan | Nilai |
|---|---|
| Bentuk | JWT bertanda tangan, atau token opaque + penyimpanan session |
| Masa berlaku access token | 30–60 menit |
| Refresh token | Ada, dengan rotasi |
| Penyimpanan di browser | **Cookie `HttpOnly` + `Secure` + `SameSite=Strict`** |
| Isi token | NIK, id session, waktu kedaluwarsa. **Tanpa izin** |
| Pencabutan | Daftar session aktif di server; logout mencabut |

**Kenapa izin tidak dimasukkan ke dalam token:** izin bisa berubah kapan saja lewat layar master
data. Bila izin tertanam di token, pencabutan hak akses baru berlaku setelah token kedaluwarsa —
bisa satu jam kemudian. Izin dibaca dari database (dengan cache in-process berumur pendek) agar
perubahan berlaku hampir seketika.

**Kenapa cookie, bukan `localStorage`:** token di `localStorage` dapat dibaca JavaScript
sehingga satu kerentanan XSS langsung berarti pencurian session. Cookie `HttpOnly` tidak dapat
dibaca JavaScript.

### 2.3 Kegagalan HCC/HCQ
Bila API tidak dapat dihubungi, pesan yang ditampilkan harus **membedakan** antara "sistem
autentikasi sedang bermasalah" dan "username atau password salah". Pesan yang sama untuk
keduanya membuat pengguna mencoba berulang kali dan membanjiri sistem yang sedang bermasalah.

Pengguna yang **sudah** login tidak terpengaruh, karena token diterbitkan aplikasi sendiri.

---

## 3. Authorization

### 3.1 Model

Tiga tingkat, dari kasar ke halus:

```
Pengguna  ──►  Peran  ──►  Izin  ──►  akses Menu dan Aksi
                             │
                             └──►  Batas Data (cabang · lini bisnis)
```

**Peran berjumlah 22, satu-untuk-satu dengan access group Pega** (`D-58`), hanya dinamai ulang
agar terbaca manusia. Tidak ada penggabungan dan tidak ada pemecahan. Seluruhnya terverifikasi ada
di export sebagai literal `GCNMFW:<nama>`: PncAdmin, PncManagerAdmin, PncPICTeknik,
PNCKomiteTeknik, PNCKomite, CaseManager, PncRCLPUCL, PncAnalystDoctor, PncComplience,
PncInvestigator, PNCSurveyor, PncPLADLA, PncReceive, PncManagerReceive, PncCollection,
PncOPCGeneral, PNCServiceCenter, TreatyIn, ViewClaimPNC, PNCReportClaimInternal,
PNCReportClaimEksternal, Administrators.

> **Koreksi v2.0 — satuan izin adalah menu, bukan aksi.** Versi sebelumnya menetapkan izin
> berbutir aksi (`klaim.akseptasi`, `komite.setujui`, …). **`D-59` memutuskan sebaliknya:** satuan
> izin adalah **menu**, dan pengguna yang memiliki akses ke sebuah menu berwenang atas **seluruh
> tindakan yang dijangkau menu itu** — termasuk membatalkan klaim, mengubah nilai setelah
> persetujuan komite, dan menyetujui komite. **Tidak ada pemisahan tugas formal.**

**Yang tetap berubah dari sistem lama adalah tempat penegakannya.** Sistem lama hanya
menyembunyikan menu — `pyPrivilegeName` terisi pada **1 dari 902 activity**, dan yang satu itu
privilege bawaan Pega untuk ekspor ruleset, bukan aturan bisnis. Di sistem baru, **setiap endpoint
memeriksa di server**: *apakah peran pemanggil memiliki menu yang memberi akses ke endpoint ini*.
Menyembunyikan menu hanyalah kenyamanan tampilan. Dengan bacaan itu, `BRD §21.2` kriteria #8 dan
`FR-R1` tetap terpenuhi.

**Konsekuensi yang diterima secara sadar** (`D-59`):

1. **Orang yang sama dapat membuat, menyetujui, dan membayarkan satu klaim** bila perannya
   memiliki ketiga menu itu. Tidak ada kontrol teknis yang mencegahnya.
2. **Jejak audit menjadi satu-satunya kontrol pengimbang.** Ini menaikkan `S-5` dari modul
   pendukung menjadi kontrol utama — dan `S-5` adalah kemampuan **baru 100% tanpa baseline**.
3. **`AutoAcceptKomite` melewati kontrol apa pun.** Job terjadwal harian jam 06:00 menyetujui
   komite tanpa pengguna sama sekali, sehingga bahkan kontrol berbasis menu tidak berlaku padanya.
4. Bila kemudian ada temuan audit atau pentest yang menuntut pemisahan tugas, perubahannya
   menyentuh model izin `F-3` — bukan penyesuaian kecil.

**Yang menjadi wajib karenanya:** `BRD §21.2` kriteria #9 — jejak audit untuk setiap perubahan
bernilai bisnis — berlaku **tanpa pengecualian** pada seluruh tiket modul bisnis.

**Tiga hal yang harus ditangani `F-3` saat membangun tabel peran:**

| Hal | Isi |
|---|---|
| **Kapitalisasi tidak konsisten** | tiga nama muncul dalam dua bentuk — `ViewClaimPNC`/`VIEWCLAIMPNC` dan `PncReceive`/`PNCRECEIVE`. Sistem baru wajib menormalkannya menjadi satu identitas per peran |
| **Peta peran → menu hidup di rule, bukan data** | pemetaan ke **51 item menu** ada di **34 When rule** + `Navigation/pyCaseWorkerNavigation-Navigation.xml`. **Lima di antaranya hilang dari export**: `IsGCNMReport`, `IsKomite`, `IsNotViewClaim`, `IsPNCBonding`, `IsSurvey` |
| **Penugasan operator ke peran tidak ada di database** | `POOLDATA.T_ACCESS_GROUP_PNC` hanya memetakan `OPERATOR_ID` → `OLD_OPERATOR_ID`. Tanpa artefak ini, `F-3` dapat membangun tabelnya tetapi **tidak dapat mengisinya** |

### 3.2 Batas data

Selain "boleh melakukan aksi apa", ada "boleh melihat data siapa". Dari analisis when rule
sistem lama, batasnya berdasarkan:
- **Cabang** — pengguna cabang hanya melihat klaim cabangnya
- **Lini bisnis / Group Panel** — PIC Teknik PA tidak melihat klaim Marine
- **Organisasi** — pengguna `Eksternal` dibatasi tegas (`OperatorID.pyOrgUnit != "Eksternal"`)

**Batas data ditegakkan di lapisan query**, bukan disaring setelah data terambil. Menyaring
setelah pengambilan berarti data yang tidak boleh dilihat sempat berada di memori aplikasi dan
ikut terhitung pada paginasi — bocor lewat jumlah baris.

### 3.3 Tabel baru yang dibutuhkan

Belum ada (D-07), dirancang dalam project ini:

| Tabel | Isi |
|---|---|
| Pengguna | NIK, nama, email, cabang, jabatan, status aktif |
| Peran | kode, nama, deskripsi — **22 baris** (`D-58`) |
| PenggunaPeran | pemetaan pengguna → peran |
| Izin | **kode menu** — 51 item (`D-59`) |
| PeranIzin | pemetaan peran → menu |
| BatasDataPengguna | cabang dan lini bisnis yang boleh diakses |
| SessionAktif | untuk pencabutan |

> **Tabel dapat dibangun, tetapi belum dapat diisi.** Dua isian menunggu pihak lain: **daftar
> operator per peran** (tidak ada di database) dan **kontrak API HCC/HCQ** yang menentukan bentuk
> identitas penggunanya (`ADR-0024`, berstatus `Proposed`).

---

## 4. Pengamanan lain

### 4.1 Masukan pengguna

| Ancaman | Penangkal |
|---|---|
| SQL injection | **Parameter binding tanpa perkecualian.** Nama kolom sort dan filter hanya dari daftar yang diizinkan. Ini yang menutup celah `{ASIS:...}` warisan |
| XSS | React meng-escape secara baku. `dangerouslySetInnerHTML` **dilarang** kecuali disetujui tertulis |
| CSRF | Cookie `SameSite=Strict` + token CSRF pada permintaan yang mengubah data |
| Unggahan berkas berbahaya | Validasi jenis berkas dan ukuran; nama berkas dihasilkan sistem, tidak pernah dari pengguna |
| Path traversal | Tidak ada akses berkas berdasarkan jalur dari pengguna |

### 4.2 Data sensitif

Sistem menyimpan NIK, nomor telepon, alamat, data medis (klaim PA), dan nomor rekening.

- Sistem lama sudah punya konsep **masking** (`FlagMaskingKTP`, `TelpMasking`,
  `NoKTPMasking`, `SetMaskingCIF_DT`) — ini **dipertahankan**.
- Data sensitif **tidak pernah** ditulis ke log (lihat §12 Cross-Cutting).
- Akses ke data medis dibatasi peran Analyst Doctor dan RCL Dokter.
- Sambungan ke database dan sistem eksternal wajib terenkripsi.

### 4.3 Kredensial
Tidak pernah di dalam kode maupun di repository. Diambil dari variabel lingkungan atau
pengelola rahasia. Berkas konfigurasi contoh hanya memuat nilai kosong.

### 4.4 Jejak audit sebagai kendali keamanan
Jejak audit (D-28) bukan hanya kebutuhan bisnis — ia juga kendali keamanan. Sifat append-only
ditegakkan lewat **hak akses database** (aplikasi hanya diberi `INSERT` dan `SELECT`), sehingga
tidak ada jalur di dalam aplikasi yang bisa menghapus jejaknya sendiri.

---

## 5. Yang harus dihilangkan dari sistem lama

| Masalah di sistem lama | Ukuran terverifikasi | Perbaikan |
|---|---|---|
| Perangkaian SQL `{ASIS:...}` dari nilai pengguna | 538 kemunculan | Parameter binding wajib |
| Potongan klausa SQL disimpan sebagai nilai property | — | Dilarang sepenuhnya |
| **User ID di-hardcode sebagai penentu perilaku** (`MORASOTARDODOTARIGAN`, `ELLENSUPRIYATI`, `IRMANOPITAPURBA_1`, `RATNAGUSNITASARI`, …) | **24 unik**, ditambah belasan tertanam di dalam teks SQL | Peran dan izin dari master data (`D-15`) |
| **Alamat email di-hardcode** sebagai penerima notifikasi | **66 unik**, termasuk **≥6 akun Gmail pribadi di jalur produksi** dan **5 alamat dipakai sebagai Operator ID** | Seluruh penerima dari master **Penerima Notifikasi**, berupa **mailbox fungsional**; **tidak ada akun pribadi** (`D-67`) |
| **Ambang uang di-hardcode** | **8 ambang komite unik** + 7 ambang uang non-komite | Master Ambang Komite (`F-4`) |
| **Hostname server menentukan perilaku bisnis** — salah satunya mengubah ambang komite dari Rp 50.000.000 menjadi 3.500 | **3 hostname, 48 perbandingan** terhadap `pxRequestor.pxReqServer` | Konfigurasi per lingkungan, bukan deteksi hostname. **Cara entitas dikenali di sistem baru belum diputuskan** (`ADR-0025`) |
| **Blok `// TESTING` menimpa email produksi** | 4 step di `Activity/InputRegister_act-Act.xml` (`:16693`, `:16830`, `:16998`, `:17141`), precondition **identik** dengan step produksi | Dilarang; ditolak di code review |
| **Kredensial plaintext di dalam rule** | **3 password SMTP di 31 lokasi** + 1 pasang kredensial OAuth; `UseSSL=false` di seluruh kemunculannya | Dipindahkan keluar dari kode — **tujuan penyimpanannya masih `OPEN`** (`D-40`, `R-17`) |
| **Dua integrasi menunjuk host sandbox** di ruleset produksi | 2 Connect REST | Endpoint per lingkungan sebagai konfigurasi (`R-18`) |
| Otorisasi bergantung pada penyembunyian menu | `pyPrivilegeName` terisi di **1 dari 902** activity | Pemeriksaan izin **di setiap endpoint backend** (`D-59`) |
| **Nol Dynamic System Setting** — konfigurasi dinamis berupa tabel Oracle yang dikunci **per IP aplikasi** | — | Konfigurasi aplikasi yang tidak terikat IP; bertabrakan dengan tuntutan dua instans (`D-27`) |

> **Nilai sensitif tidak direproduksi di dokumen ini.** Hostname produksi, alamat email,
> kredensial, dan data nasabah dirujuk dengan `berkas:baris` + nama elemen saja (`D-69`). Lokasi
> lengkap kredensial sudah diserahkan ke Tim Infra/Security lewat dokumen terpisah **di luar
> repository**.

---

## 6. Data nasabah di lingkungan non-produksi

`D-64` menetapkan data produksi disalin ke **staging apa adanya, tanpa penyamaran**, dengan **hak
akses staging diperketat** sebagai kontrol penggantinya. Alasannya ada di Testing Strategy §6.1:
cacat yang ditemukan pada verifikasi Fase 1 muncul dari data nyata yang tidak akan terpikir
dibuat.

**Konsekuensi yang diterima secara sadar:**

1. **Staging memuat data nasabah nyata** — nomor polis, nama tertanggung, NPWP, nomor rekening,
   dan **data medis** pada lini PA dan Travel. Perlindungannya sepenuhnya bergantung pada hak
   akses, bukan pada sifat datanya.
2. **Klasifikasi staging naik setara produksi** untuk keperluan keamanan.
3. Pembatasan akses data medis yang `FR-R2` terapkan pada peran Analyst Doctor dan RCL Dokter
   **berlaku juga di staging**.

**Masih terbuka:** siapa yang menyetujui akses staging, berapa lama salinan disimpan, dan
prosedur pemusnahannya — ditujukan ke pihak yang sama dengan `D-40`.

**Aturan penulisan dokumen** (`D-69`), berlaku untuk seluruh artefak yang di-commit:

| Jenis nilai | Perlakuan |
|---|---|
| Nama Operator ID | **boleh ditulis lengkap** — diperlukan agar tiket dapat menunjuk hardcode mana yang dihapus |
| Alamat email | **selalu disamarkan** |
| Data nasabah — nomor polis, nama tertanggung, nomor klaim asli, NPWP, nomor rekening | **tidak pernah ditulis**; dirujuk dengan `berkas:baris` atau nama kolom |
| Kredensial, kunci API, hostname/IP produksi | **tidak pernah ditulis** |
