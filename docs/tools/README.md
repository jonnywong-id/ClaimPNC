# docs/tools — Pembangun Dokumen Gabungan

Skrip di folder ini membangun tiga dokumen gabungan beserta versi `.docx`-nya dari berkas sumber
di `docs/`. Seluruhnya hanya membaca `docs/` dan menulis ke `docs/` — **tidak pernah menyentuh
rule XML Pega**.

| Berkas | Fungsi |
|---|---|
| `md2html.js` | **Pustaka bersama** — pengubah Markdown → HTML bergaya. Dipakai ketiga generator; tidak dijalankan langsung |
| `build-steering.js` | Menyusun **`docs/Steering/STEERING.md`** dari 16 bab sumber + 7 lampiran, lalu menulis HTML-nya |
| `build-adr.js` | Menyusun **`docs/ADR/ADR.md`** dari `docs/ADR/README.md` + `docs/ADR/00NN-*.md`, lalu menulis HTML-nya |
| `md2doc.js` | Mengubah **satu** berkas Markdown menjadi HTML bergaya — dipakai untuk `docs/BRD/BRD.md`, yang tidak punya generator sendiri |
| `build-inventaris-harness.js` | Membangun ulang **tabel** di `docs/Steering/22-INVENTARIS-HARNESS.md` langsung dari direktori `Harness/`, sehingga daftar 74 harness tidak dapat menyimpang dari export. Prosa di atas tabel tidak disentuh |
| `html2docx.ps1` | Mengubah HTML menjadi `.docx` lewat otomasi Microsoft Word |

---

## Dokumen mana dibangun dari mana

| Dokumen gabungan | Sumbernya | Generator |
|---|---|---|
| `docs/Steering/STEERING.md` | 16 bab + 7 lampiran di `docs/Steering/` | `build-steering.js` |
| `docs/ADR/ADR.md` | 29 ADR + README di `docs/ADR/` | `build-adr.js` |
| `docs/BRD/BRD.md` | **dirinya sendiri** — disunting langsung | — (hanya `md2doc.js` untuk `.docx`-nya) |

> **`STEERING.md` dan `ADR.md` adalah hasil bangunan.** Menyuntingnya langsung sia-sia — perubahan
> hilang pada pembangunan berikutnya. Yang disunting adalah berkas sumbernya. **`BRD.md`
> sebaliknya**: ia berkas sumber, disunting langsung.

---

## Cara menjalankan

Seluruh perintah dijalankan dari root proyek. Ganti `<ROOT>` dengan jalur proyek.

### Steering

```bash
node docs/tools/build-steering.js "<ROOT>" "%TEMP%\STEERING.html"
```

```powershell
powershell -File docs\tools\html2docx.ps1 -HtmlPath "$env:TEMP\STEERING.html" -DocxPath "docs\Steering\Steering Document - Migrasi CLAIM PNC ke Golang.docx" -Title 'STEERING DOCUMENT - Migrasi Aplikasi CLAIM PNC'
```

### ADR

```bash
node docs/tools/build-adr.js "<ROOT>" "%TEMP%\ADR.html"
```

```powershell
powershell -File docs\tools\html2docx.ps1 -HtmlPath "$env:TEMP\ADR.html" -DocxPath "docs\ADR\ADR.docx" -Title 'ADR - Migrasi Aplikasi Claim PNC ke Golang'
```

### BRD

```bash
node docs/tools/md2doc.js "docs/BRD/BRD.md" "%TEMP%\BRD.html" "Business Requirement Document"
```

```powershell
powershell -File docs\tools\html2docx.ps1 -HtmlPath "$env:TEMP\BRD.html" -DocxPath "docs\BRD\BRD - Migrasi Aplikasi Claim PNC ke Golang.docx" -Title 'Business Requirement Document - Migrasi Aplikasi Claim PNC'
```

Argumen path HTML boleh dikosongkan pada ketiga skrip Node — bila tidak diisi, HTML ditulis ke
folder temp sistem, sehingga berkas antara itu **tidak ikut ter-commit**.

---

## Prasyarat

| Kebutuhan | Keterangan |
|---|---|
| **Node.js** | tanpa dependensi eksternal; hanya modul bawaan `fs`, `path`, `os` |
| **Microsoft Word** | hanya untuk langkah `.docx`. Diakses lewat COM (`New-Object -ComObject Word.Application`) |

Tanpa Word, langkah Node tetap berjalan dan berkas `.md` tetap terbangun — yang tidak terbentuk
hanya `.docx`-nya.

---

## Kapan dijalankan

| Berubah | Jalankan |
|---|---|
| Salah satu bab di `docs/Steering/` | `build-steering.js` + `html2docx.ps1` |
| Salah satu ADR atau `docs/ADR/README.md` | `build-adr.js` + `html2docx.ps1` |
| `docs/BRD/BRD.md` | `md2doc.js` + `html2docx.ps1` |
| `00-DECISION-LOG.md` atau `CONTEXT.md` | **`build-steering.js`** — keduanya menjadi Lampiran B dan A |

Butir terakhir mudah terlewat: setiap keputusan baru yang ditambahkan ke Decision Log **tidak
muncul di `STEERING.md`** sampai generatornya dijalankan.

---

## Yang perlu disunting di dalam skrip

Sebagian isi dokumen gabungan tidak berasal dari berkas sumber dan ditulis langsung di generator.

### `build-steering.js`

| Bagian dokumen | Lokasi |
|---|---|
| Nomor versi dan tanggal | konstanta `VERSI` dan `TANGGAL` |
| **Urutan dan judul 16 bab** | konstanta `BAB` |
| **Urutan dan judul 6 lampiran** | konstanta `LAMPIRAN` |
| Tabel identitas di sampul | blok `P('\| **Status** \| …')` dan seterusnya |

Menambah bab baru = menambah satu baris di `BAB`. Nomor babnya mengikuti **urutan array**, bukan
nama berkasnya.

### `build-adr.js`

| Bagian dokumen | Lokasi |
|---|---|
| Pengelompokan A–F beserta paragraf pengantarnya | konstanta `GROUPS` |
| §1.3 lima keputusan yang belum diambil | blok `P('### 1.3 …')` |
| §1.4 keputusan yang menyupersede dokumen lain | blok `P('### 1.4 …')` |
| §1.5 tiga keputusan paling berisiko | blok `P('### 1.5 …')` |
| §4 hubungan dengan dokumen lain | blok `P('## 4. …')` |
| Lampiran A dan B | blok `P('## Lampiran A …')` dan `P('## Lampiran B …')` |

Bagian lain — index, papan status, isi tiap ADR, dan **Lampiran C (rekapitulasi pertanyaan
terbuka)** — dihasilkan otomatis.

---

## Catatan perilaku yang sudah diketahui

- **Word kadang gagal menutup dirinya sendiri** setelah menyimpan (`RPC failed`, `0x800706BE`).
  **Berkas `.docx`-nya sudah tertulis utuh** saat itu terjadi; skrip melaporkannya, tetap
  memverifikasi hasilnya, dan membersihkan proses Word yang menggantung. Periksa baris `OK: …`.
- **Pengubah Markdown hanya menangani subset** yang dipakai dokumen proyek ini: heading, tabel
  pipa, blok kode berpagar, kutipan, daftar berurut dan tak berurut, `**tebal**`, `*miring*`,
  `~~coret~~`, `` `kode` ``, dan tautan. Bila kelak dipakai sintaks di luar itu — HTML mentah,
  gambar, footnote — pengubahnya perlu ditambah.
- **Tautan internal (`#anchor`) menjadi teks biasa** di `.docx`, karena jangkar antar-bagian tidak
  terbawa lewat impor HTML.
- **Heading `#` di dalam berkas sumber diturunkan satu tingkat** saat digabungkan, agar `#` di
  dokumen gabungan hanya dipakai untuk judul bab dan lampiran.
- **Nilai header ADR yang memuat tanda `|`** diubah menjadi `·` saat disusun menjadi tabel dua
  kolom, agar tidak merusak tabelnya.
- **Berkas `.docx` versi lama disimpan sebagai arsip** dengan awalan `ARSIP v1.0 (tanggal) - …`
  dan tidak dihapus (`D-72`).

---

## Lebar tabel pada berkas `.docx`

**Gejala.** Tabel berkolom banyak terpotong tulisannya di `.docx` Steering dan ADR.

**Penyebab — terukur, bukan dugaan.** Word mengimpor tabel HTML dengan lebar **sel** yang tidak
terikat. Dibaca langsung dari `word/document.xml` di dalam `.docx` lama: **5.244 dari 5.244 sel**
Steering ber-`w:type="auto"`, artinya isi selnya yang menentukan lebar kolom. Isi yang panjang
mendorong kolom melewati area cetak, dan sisanya terpotong.

**Yang TIDAK menyelesaikannya** — semuanya sudah dicoba dan diukur, tidak satu pun mengubah lebar
sedikit pun:

| Dicoba lewat Word COM | Hasil |
|---|---|
| `AutoFitBehavior(wdAutoFitContent)` dan `(wdAutoFitWindow)` | lebar tidak berubah |
| `Columns.Item(n).Width` | gagal **diam-diam** — tidak melempar galat, nilainya tetap |
| `Cell.PreferredWidth` per sel | lebar tidak berubah |
| Mengubah view ke Print (`View.Type = 3`) lalu `Repaginate()` | lebar tidak berubah |

Sebagai ukuran betapa jauhnya: satu tabel **2 kolom** berisi kata "Accepted" memakai kolom selebar
**946pt** pada halaman yang area teksnya **415pt**.

**Yang menyelesaikannya.** Lebar ditetapkan di **HTML**, sebelum Word sempat menghitung sendiri —
atribut `width` dalam persen pada `<table>` dan tiap sel (`lebarKolom()` di `md2html.js`),
dibagi sebanding panjang isi kolom dengan batas bawah 7% dan batas atas 55%.

Ditambah dua hal di `html2docx.ps1`: baris judul diulang di tiap halaman, dan huruf dikecilkan pada
tabel berkolom ≥ 5.

**Hasil, dibaca dari dalam `.docx`:**

| Dokumen | Tabel | Sel | Bukan persen | Tabel ≠ 100% | Baris yang jumlah %-nya menyimpang |
|---|---:|---:|---:|---:|---:|
| ADR | 64 | 1.137 | 0 | 0 | 0 |
| Steering | 213 | 5.238 | 0 | 0 | 0 |
| BRD | 80 | 1.624 | 0 | 0 | 0 |

**Cacat kedua yang ikut ketahuan.** Pemecah sel memakai `split("|")` biasa, sehingga ikut memecah
pada pipa ter-escape `\|`. Baris yang memuat operator SQL `||` **pecah menjadi kolom tambahan dan
isinya terpotong** — bukan sekadar salah lebar, melainkan salah isi. Terdeteksi pada
`00-DECISION-LOG.md:460` dan `BRD.md:263`. Diperbaiki dengan `belahSel()` yang menghormati escape.

**Cara memeriksa ulang.** Buka `.docx` sebagai zip, baca `word/document.xml`, dan pastikan setiap
`<w:tblW>` dan `<w:tcW>` ber-`w:type="pct"`. Jangan memakai `Cell.Width` lewat Word COM untuk
memeriksa: dokumen hasil impor HTML dibuka dalam Web Layout, dan nilai yang dikembalikannya
**bukan** lebar cetak — itu sempat menyesatkan pemeriksaan ini.
