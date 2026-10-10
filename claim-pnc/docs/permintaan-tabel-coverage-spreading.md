# Permintaan tabel coverage dan spreading polis — `POOLDATA.T_COVERAGELIST_*`

| | |
|---|---|
| **Tanggal** | 2026-10-06 |
| **Diminta oleh** | Tim migrasi Claim PNC |
| **Ditujukan ke** | **DBA** dan **tim pemilik sistem polis (GISFW)** |
| **Menempuh** | `D-63` — permintaan tertulis tim pengembang, persetujuan Work Owner, pelaksanaan DBA |
| **Modul terdampak** | `B-2` Registrasi Klaim — tahap **Input Register** |
| **Sifat** | Tabel **dibuat** dan **diisi**. Tidak ada kolom sistem lama yang diubah atau dihapus. |
| **Mendesak** | **Ya** — registrasi klaim sedang mati untuk PA, Travel, Marine Cargo, dan Aneka |

---

## 1. Ringkas

Work Owner menetapkan pada 2026-10-06 bahwa coverage dan spreading polis tidak lagi dibaca
dari kolom BLOB berisi dokumen JSON, melainkan dari tabel relasional. Aplikasi sudah
mengikuti ketetapan itu.

**Tiga dari empat tabel penggantinya tidak ada di basis data portal `ASM`**, sehingga
registrasi klaim gagal dengan `ORA-00942` untuk setiap lini kecuali Fire.

| Lini | Tabel coverage | Ada? | Terisi? |
|---|---|:---:|---|
| Fire | `POOLDATA.T_COVERAGELIST_FIRE` | ✅ | ✅ 51.165 baris · 17.734 polis |
| PA, Travel | `POOLDATA.T_COVERAGELIST_PERSON` | ❌ | — |
| Marine Cargo | `POOLDATA.T_COVERAGELIST_CARGO` | ❌ | — |
| Aneka | `POOLDATA.T_COVERAGELIST_ANEKA` | ❌ | — |
| seluruh lini | `POOLDATA.T_SPREADINGLIST` | ✅ | ⚠️ **1.904 baris · hanya 448 polis** |

Dua hal yang diminta, dan **keduanya diperlukan**:

1. **§2 — tiga tabel dibuat.** Tanpa ini registrasi tetap mati untuk tiga lini.
2. **§3 — keempat tabel diisi, termasuk `T_SPREADINGLIST` yang sudah ada.** Tanpa ini
   registrasi berhenti gagal, tetapi klaim lahir tanpa coverage atau tanpa spreading dan
   ditolak gerbang validasi dengan pesan yang menyesatkan.

---

## 2. Tiga tabel yang diminta dibuat

Bentuknya diambil dari DDL yang Work Owner lampirkan di `Sample Form/` — yaitu bentuk yang
berlaku di basis data tempat ketiganya sudah ada. **Yang di bawah hanya kolom yang
benar-benar dibaca aplikasi**; bila tabel aslinya punya kolom lain, salin apa adanya dari
basis data asalnya alih-alih memakai potongan ini.

```sql
CREATE TABLE POOLDATA.T_COVERAGELIST_PERSON
(
  APPLICATIONID          VARCHAR2(100 BYTE),
  IDPEGA                 VARCHAR2(150 BYTE),
  NOPOLIS                VARCHAR2(100 BYTE),
  PRODKE                 VARCHAR2(100 BYTE),
  INDEXOBJECT            NUMBER,
  INDEXCOVERAGE          NUMBER,
  COVERAGEID             VARCHAR2(100 BYTE),
  COVERAGENOTE           VARCHAR2(1000 BYTE),
  TSI                    NUMBER,
  FLAGDELETE             VARCHAR2(10 BYTE)
);

CREATE TABLE POOLDATA.T_COVERAGELIST_CARGO
(
  APPLICATIONID          VARCHAR2(100 BYTE),
  IDPEGA                 VARCHAR2(100 BYTE),
  NOPOLIS                VARCHAR2(100 BYTE),
  PRODKE                 NUMBER,
  INDEXOBJECT            NUMBER,
  INDEXCOVERAGE          VARCHAR2(10 BYTE),
  COVERAGE               VARCHAR2(100 BYTE),
  COVERAGENOTE           VARCHAR2(1000 BYTE),
  TSI                    NUMBER,
  SUMTSI                 NUMBER,
  FLAGDELETE             VARCHAR2(10 BYTE)
);
```

**`T_COVERAGELIST_ANEKA` belum punya DDL.** Berkas `Sample Form/T_COVERAGELIST_ANEKA.sql`
ternyata berisi `T_OCCUPATIONLIST_ANEKA` — daftar okupasi, bukan coverage. Aplikasi membaca
kolom yang sama dengan `T_COVERAGELIST_CARGO`:

```
NOPOLIS, PRODKE, INDEXOBJECT, INDEXCOVERAGE, COVERAGE, COVERAGENOTE, TSI, SUMTSI, FLAGDELETE
```

**Mohon DDL `T_COVERAGELIST_ANEKA` yang sebenarnya dikirimkan**, bukan dibuat dari potongan
di atas. Aneka adalah lini terbesar di basis data ini — **105.606 polis** di
`T_ANEKALIST` — sehingga tabel yang salah bentuk di sini berdampak paling luas.

---

## 3. Pengisian — bagian yang lebih menentukan daripada pembuatan tabelnya

Membuat tabel kosong **tidak menyelesaikan apa pun**. Klaim akan tetap lahir, tetapi tanpa
coverage, dan gerbang validasi Input Register menolaknya dengan
*"Objek … tidak memiliki coverage."* — pesan yang menyalahkan data polis padahal yang
kosong adalah tabelnya.

### 3.1 `T_SPREADINGLIST` jauh dari lengkap

Ini yang paling perlu diperhatikan, dan ia menyangkut tabel yang **sudah ada**:

| Tabel | Baris | Polis berbeda |
|---|---:|---:|
| `T_COVERAGELIST_FIRE` | 51.165 | **17.734** |
| `T_PROPERTYLIST` (objek Fire) | 20.228 | 17.729 |
| `T_ANEKALIST` (objek Aneka) | 115.027 | 105.606 |
| **`T_SPREADINGLIST`** | **1.904** | **448** |

Coverage Fire lengkap — 17.734 polis berbanding 17.729 polis objeknya. **Spreading tidak**:
448 polis dari sekurang-kurangnya 123.000 polis berobjek.

Akibatnya, **meski ketiga tabel di §2 dibuat hari ini**, hampir setiap klaim akan lahir
dengan coverage tetapi tanpa spreading, lalu ditolak gerbang validasi dengan
*"Total spreading harus 100%."* Dari sudut pandang petugas, registrasi tetap tidak dapat
diselesaikan — hanya pesannya yang berubah.

### 3.2 Kunci penggabungan perlu dipastikan

Aplikasi menggabungkan baris dengan `NOPOLIS` + `PRODKE`, lalu `INDEXOBJECT` +
`INDEXCOVERAGE` — kunci yang sama dengan yang dipakai membaca tabel objek polis. Isi
`T_SPREADINGLIST` hari ini menunjukkan kunci itu **mungkin bukan kunci yang benar**:

| Kolom | Baris bernilai NULL (dari 1.904) |
|---|---:|
| `IDPEGA` | **0** |
| `NOPOLIS` | 294 |
| `PRODKE` | 416 |
| `INDEXOBJECT` | 0 |
| `INDEXCOVERAGE` | 0 |

`IDPEGA` selalu terisi; `NOPOLIS` dan `PRODKE` tidak. `Database/INSERTT_PERSONLIST.prc`
juga memperlakukan `IDPEGA` sebagai kuncinya. **Mohon dipastikan**: apakah
`T_SPREADINGLIST` dan `T_COVERAGELIST_*` memang dapat dibaca dengan `NOPOLIS` + `PRODKE`,
atau `IDPEGA` yang seharusnya dipakai. Catatan yang sama berlaku untuk
`T_COVERAGELIST_FIRE`, yang `NOPOLIS`-nya NULL pada 3.525 dari 51.165 baris.

Bila jawabannya `IDPEGA`, kueri aplikasi perlu diubah dan permintaan ini bertambah satu
butir: klaim harus membawa `IDPEGA` polisnya, dan snapshot polis saat ini tidak
menyimpannya.

---

## 4. Cara memastikan permintaan ini sudah selesai

Aplikasi punya pemeriksa bawaan yang **tidak menulis apa pun**:

```
claimpnc -periksa
```

Keluarannya menyebut kelima tabel satu per satu. Permintaan ini selesai ketika kelimanya
berbunyi `[ok]`:

```
[ok] POOLDATA.T_COVERAGELIST_PERSON — kolom yang ditulis modul tersedia
[ok] POOLDATA.T_COVERAGELIST_FIRE   — kolom yang ditulis modul tersedia
[ok] POOLDATA.T_COVERAGELIST_CARGO  — kolom yang ditulis modul tersedia
[ok] POOLDATA.T_COVERAGELIST_ANEKA  — kolom yang ditulis modul tersedia
[ok] POOLDATA.T_SPREADINGLIST       — kolom yang ditulis modul tersedia
```

Pemeriksa itu hanya menguji **bentuk**, bukan **isi**. Kelengkapan isinya diperiksa
terpisah; angka yang diharapkan adalah jumlah polis pada tabel coverage dan spreading
sepadan dengan jumlah polis pada tabel objeknya, seperti Fire hari ini:

```sql
SELECT 'objek   ' AS bagian, COUNT(DISTINCT NOPOLIS) FROM POOLDATA.T_PROPERTYLIST
UNION ALL
SELECT 'coverage', COUNT(DISTINCT NOPOLIS) FROM POOLDATA.T_COVERAGELIST_FIRE
UNION ALL
SELECT 'spreading', COUNT(DISTINCT NOPOLIS) FROM POOLDATA.T_SPREADINGLIST;
```

---

## 5. Keadaan sementara permintaan ini belum selesai

**Registrasi klaim mati untuk PA, Travel, Marine Cargo, dan Aneka.** Lini Fire berjalan,
tetapi klaimnya lahir tanpa spreading pada polis yang tidak termasuk 448 polis itu.

Pilihan "mundur sementara ke kolom BLOB bila tabelnya tidak ada" diajukan dan
**ditolak Work Owner pada 2026-10-06**; yang dipilih adalah melengkapi tabelnya lebih dulu.
Keputusan itu dihormati, dan dampaknya dicatat di sini supaya terlihat: selama permintaan
ini belum selesai, petugas tidak dapat meregistrasi klaim pada empat dari lima lini.
