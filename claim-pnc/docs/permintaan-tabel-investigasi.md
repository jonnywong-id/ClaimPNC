# Permintaan tabel baru — hasil investigasi klaim

| | |
|---|---|
| **Status** | ✅ **SELESAI — tabel dibuat Work Owner 2026-10-06.** Dokumen ini disimpan sebagai rekaman permintaannya |
| **Tanggal** | 2026-10-05 |
| **Diminta oleh** | Tim migrasi Claim PNC |
| **Ditujukan ke** | **Work Owner** (persetujuan) lalu **DBA** (pelaksanaan) |
| **Menempuh** | `D-63` — permintaan tertulis, persetujuan Work Owner, pelaksanaan DBA, lalu diuji dengan menjalankan Pega dan Go bersamaan |
| **Modul terdampak** | Inbox Investigator (`MENU_ID 48`) — formulir kerja Investigator |
| **Sifat perubahan** | **Tabel baru**, bukan penambahan kolom pada tabel warisan |

---

## 1. Kenapa ini diminta

Menekan **Nomor Case** di Inbox Investigator menjalankan Flow Action
`InputInvestigator` — bukan membuka tampilan, melainkan **formulir kerja yang menyimpan**:

| Bagian Flow Action | Isi | Bukti |
|---|---|---|
| Pra-proses | `PresetInvestigation` — isi Tanggal Investigasi dengan waktu kini | `Flow Action/InputInvestigator-FA.xml:128` |
| Formulir | `InputClaimInvestigasiDetail` — 30 isian | `:49` |
| Simpan | `SetStatusInvestigator_Act` | `:144` |

Tanpa tempat menyimpan ketiga puluh isian itu, formulirnya **tidak dapat dibangun** — yang
dapat dibangun hanyalah layar yang tampil lengkap lalu gagal saat ditekan Simpan.

## 2. Kenapa tidak memakai tabel yang sudah ada

Ketiganya sudah diperiksa dan tidak satu pun dapat dipakai:

| Tabel | Kenapa tidak |
|---|---|
| `POOLDATA.T_SURVEYORLIST` | tidak punya satu pun kolom investigasi; isinya hasil survei milik kasus `Work-SurveyClaim` |
| `POOLDATA.INVESTIGATIONREPORT` | **nol rujukan** dari rule Pega mana pun di seluruh export — ia milik sistem lain |
| `POOLDATA.JSON_KLAIM.DATA_JSONBLOB` | **ditulis Pega** lewat `PEGA_CONVERT_JSONKLAIM_PNC`; menulis ke sana melanggar `P-1` selama masa paralel |

Hari ini nilainya memang hidup di `JSON_KLAIM` — dan dari sanalah fitur **Export Data
Investigation** membacanya. Membaca boleh; menulis tidak.

## 3. Yang diminta

Satu tabel baru milik aplikasi ini: **`POOLDATA.TC_PNC_INVESTIGASI`**.

> **Koreksi 2026-10-06.** Dokumen ini semula mengusulkan nama `CPNC_INVESTIGASI`, dengan
> alasan "mengikuti awalan `CPNC_`". Alasannya menyebut keluarga yang salah: di basis data
> ini **dua** keluarga hidup berdampingan — `CPNC_*` untuk modul fondasi (`CPNC_PENGGUNA`,
> `CPNC_SESI_AKTIF`, `CPNC_TUGAS`), dan **`TC_PNC_*` untuk modul klaim** (`TC_PNC_KOMITE`,
> `TC_PNC_PUCL`, `TC_PNC_OBJECTITEM`, dan lima lainnya). Tabel ini masuk keluarga kedua, dan
> itulah nama yang dipakai DDL, seluruh kueri, dan mode periksa sejak awal.
>
> Seandainya nama di paragraf ini dipakai apa adanya, tabelnya akan terbuat dengan nama yang
> **tidak dicari aplikasi**, dan layarnya tetap menolak menyimpan tanpa satu pun petunjuk
> mengapa.

### 3.1 Kunci

| Kolom | Tipe | Keterangan |
|---|---|---|
| `KLAIM_ID` | `VARCHAR2(100)` | `pzInsKey` klaim — sama dengan `T_CLAIM_PNC.CLAIMID` |
| `URUTAN_SURVEI` | `NUMBER(5)` | indeks `SurveyResults(n)`; hari ini selalu 1 |
| `URUTAN` | `NUMBER(5)` | indeks `SurveyList(n)` |

Kunci utama: ketiganya. Satu klaim dapat punya lebih dari satu baris investigasi, persis
seperti page list di Pega.

### 3.2 Isian formulir — 30 kolom

Nama kolom mengikuti nama property Pega supaya penelusurannya tetap mungkin. Tipe diambil
dari nilai yang benar-benar tersimpan di `JSON_KLAIM` hari ini.

| # | Kolom | Tipe | Isian di layar | Nilai terukur |
|---|---|---|---|---|
| 1 | `TANGGAL_INVESTIGASI` | `DATE` | Tanggal Investigasi | diisi otomatis saat formulir dibuka |
| 2 | `IS_INVESTIGATED` | `VARCHAR2(1)` | Dapat Diinvestigasi | `1` / `0` |
| 3 | `SELECT_RS` | `VARCHAR2(1)` | RS/Klinik yang di Survei | `1` / `0` |
| 4 | `RS_KLINIK_DISURVEI` | `VARCHAR2(200)` | Nama Rumah Sakit | — |
| 5 | `RS_KLINIK_DISURVEI_LAIN` | `VARCHAR2(200)` | Nama Tempat Lainnya | — |
| 6 | `ALAMAT_RS_KLINIK` | `VARCHAR2(500)` | Alamat RS/Klinik | — |
| 7 | `NO_REKAP_MEDIS` | `VARCHAR2(50)` | Nomor Rekam Medik | **data medis** |
| 8 | `NAMA_PASIEN` | `VARCHAR2(200)` | Nama Peserta | **data nasabah** |
| 9 | `TANGGAL_LAHIR` | `DATE` | Tanggal Lahir | **data nasabah** |
| 10 | `FLAG_DOB` | `VARCHAR2(1)` | Verifikasi Tanggal Lahir | — |
| 11 | `KETERANGAN_DOB` | `VARCHAR2(500)` | Keterangan (verifikasi lahir) | — |
| 12 | `PASIEN_TERDAFTAR` | `VARCHAR2(1)` | Peserta Terdaftar Di RS | `1` / `0` |
| 13 | `PT_REG` | `VARCHAR2(200)` | Keterangan Tambahan | — |
| 14 | `TANGGAL_PERAWATAN` | `DATE` | Tanggal Masuk | **data medis** |
| 15 | `TANGGAL_SELESAI_PERAWATAN` | `DATE` | Tanggal Keluar | **data medis** |
| 16 | `TOTAL_TAGIHAN` | `NUMBER(18,2)` | Total Pengajuan | nilai uang |
| 17 | `TAGIHAN_LUNAS` | `VARCHAR2(1)` | status pelunasan tagihan | — |
| 18 | `BAYAR_PASIEN` | `VARCHAR2(5)` | Yang Melakukan Pembayaran — Pasien | `true` / `false` |
| 19 | `BAYAR_PERUSAHAAN` | `VARCHAR2(5)` | idem — Perusahaan | `true` / `false` |
| 20 | `BAYAR_ASURANSI_LAIN` | `VARCHAR2(5)` | idem — Asuransi Lain | `true` / `false` |
| 21 | `BAYAR_TIDAK_ADA` | `VARCHAR2(5)` | idem — Tidak ada pembayaran | `true` / `false` |
| 22 | `ASURANSI_LAIN` | `VARCHAR2(200)` | Nama Asuransi/Perusahaan Lain | — |
| 23 | `KONFIRMASI_KWITANSI` | `VARCHAR2(1)` | Konfirmasi Model Kwitansi | `1` / `0` |
| 24 | `NAMA_PIC_RS` | `VARCHAR2(200)` | Nama PIC RS | — |
| 25 | `NAMA_PENELEPON` | `VARCHAR2(200)` | Nama Penelepon | — |
| 26 | `NAMA_KARYAWAN` | `VARCHAR2(200)` | Nama PIC Yang Dapat Dihubungi | — |
| 27 | `KODE_AREA_TELP` | `VARCHAR2(10)` | Kode | — |
| 28 | `NO_TELP_DIHUBUNGI` | `VARCHAR2(50)` | Nomor Telepon Yang Dapat Dihubungi | — |
| 29 | `EKSTENSI` | `VARCHAR2(10)` | No. Ext. | — |
| 30 | `REMAKS` | `VARCHAR2(2000)` | Hasil Investigasi | — |

### 3.3 Kolom jejak

| Kolom | Tipe | Keterangan |
|---|---|---|
| `DIBUAT_OLEH` · `DIBUAT_PADA` | `VARCHAR2(100)` · `TIMESTAMP` | pelaku dan waktu |
| `DIUBAH_OLEH` · `DIUBAH_PADA` | `VARCHAR2(100)` · `TIMESTAMP` | idem |
| `DIHAPUS_PADA` | `TIMESTAMP` | **soft delete** — `D-66` melarang penghapusan fisik data bernilai bisnis |

### 3.4 Dua kolom yang TIDAK masuk tabel ini

Keduanya milik klaim, bukan milik baris investigasi, dan kolomnya sudah ada di
`T_CLAIM_PNC`:

| Property Pega | Isian di layar |
|---|---|
| `.ClaimData.AnalystRemaksInvestigator` | Pertanyaan Dari Analyst / Alasan dari Analyst |
| `.ClaimData.TanggalSelesaiRawatInap` | Tanggal Selesai Perawatan (tingkat klaim) |

Perlu dikonfirmasi DBA apakah keduanya sudah punya kolom setelah `T_CLAIM_PNC` dipangkas
dari 97 menjadi 82 kolom pada 2026-09-26.

## 4. Yang TIDAK diminta

**Tidak ada perubahan pada tabel warisan Pega.** Perpindahan status yang dilakukan
`SetStatusInvestigator_Act` — `StatusClaim` menjadi `1151`, `InvestTfDate`,
`AnalystTransferDate` — seluruhnya menulis ke `POOLDATA.T_CLAIM_PNC`, yang **sudah ditulis
aplikasi ini** lewat `klaim_perbarui`. Tidak ada kepemilikan baru yang perlu dinegosiasikan
untuk bagian itu.

## 5. Data sensitif

Tabel ini memuat **data medis** — nomor rekam medis, tanggal rawat inap, nama pasien.
`FR-R2` membatasi aksesnya pada peran tertentu, dan `D-64` menetapkan salinan produksi di
staging **tidak disamarkan**. Hak akses tabel ini karena itu perlu ditetapkan bersamaan
dengan pembuatannya, bukan sesudahnya.

## 6. Keadaan per 2026-10-06 — SELESAI

Tabel dibuat Work Owner pada 2026-10-06, dan bentuknya **cocok persis** dengan DDL di
[`ddl/tc_pnc_investigasi.sql`](ddl/tc_pnc_investigasi.sql) — diperiksa kolom per kolom
terhadap `ALL_TAB_COLUMNS`:

| Yang diperiksa | Hasil |
|---|---|
| Jumlah kolom | **38** — 3 kunci + 30 isian + 5 jejak |
| Tipe dan panjang | seluruhnya sesuai |
| Kunci utama | `PK_TC_PNC_INVESTIGASI` atas `KLAIM_ID`, `URUTAN_SURVEI`, `URUTAN` |
| Indeks | `IX_TC_PNC_INVESTIGASI_KLAIM` atas `KLAIM_ID`, `DIHAPUS_PADA` |
| Kolom yang dirujuk kueri tulis | seluruhnya ada — `investigasi_ambil`, `investigasi_perbarui`, `investigasi_sisip`, `investigasi_check_table` |

`claimpnc -periksa` kini menyatakannya hijau:

```
[ok]    antrean Inbox Investigator dapat dibaca
[ok]    seluruh pekerjaan punya tanggal survei
[ok]    formulir Investigator dapat menyimpan hasil investigasi
```

### Yang tersisa, dan bukan lagi soal tabel

| Hal | Pemilik |
|---|---|
| **Tiga kolom `T_CLAIM_PNC` sudah dipastikan TIDAK ADA** — lihat §7 | Work Owner → DBA |
| Tabel ini **belum dibuat di tiga portal entitas lain** (`D-75`) | Work Owner + DBA |
| Hak akses data medis (`FR-R2`) | Compliance |
| Tab "Unggah Dokumen" — **BUKAN** tertahan modul Dokumen; panelnya sudah ada di Registrasi. Yang menahan adalah cacat `URUTAN` NULL pada pemuat klaim, yang mengenai **1.665 dari 1.686 klaim** | Lead engineer |

---

## 7. Permintaan susulan — tiga kolom pada `T_CLAIM_PNC`

| | |
|---|---|
| **Diajukan** | 2026-10-06 |
| **Menempuh** | `D-63` — permintaan tertulis → Work Owner → DBA |
| **Sifat** | **penambahan kolom pada tabel warisan**, bukan tabel baru |

Ketiganya diperiksa langsung ke `ALL_TAB_COLUMNS` pada 2026-10-06 dan **tidak ada di mana
pun** — bukan di `T_CLAIM_PNC`, bukan di tabel lain:

| Properti Pega | Isian di layar | Usulan kolom | Tipe |
|---|---|---|---|
| `.ClaimData.AnalystRemaksInvestigator` | **Pertanyaan Dari Analyst** | `ANALYST_REMAKS_INVESTIGATOR` | `VARCHAR2(4000)` |
| `.ClaimData.TanggalSelesaiRawatInap` | Tanggal Keluar (tingkat klaim) | `TANGGAL_SELESAI_RAWATINAP` | `DATE` |
| `.ClaimData.SelectRawatInap` | penentu tampil/tidaknya isian rawat inap | `SELECT_RAWATINAP` | `VARCHAR2(5)` |

Panjang `VARCHAR2(4000)` mengikuti kolom catatan lain pada tabel yang sama — `ALASAN`,
`CLOSECLAIMNOTE`, `REMARKRECOMENDATION`, `KRONOLOGI` — seluruhnya `VARCHAR2(4000)`.
`VARCHAR2(5)` mengikuti pola `'true'`/`'false'` yang dipakai penanda lain di formulir ini.

### Kenapa tidak ditumpangkan ke `TC_PNC_INVESTIGASI`

Ketiganya milik **klaim**, bukan milik satu baris investigasi. Satu klaim secara mekanisme
dapat punya lebih dari satu baris investigasi; menaruh ketiganya di sana akan menggandakan
nilai yang seharusnya tunggal, lalu menimbulkan pertanyaan baris mana yang benar.

### Yang tertahan selama ketiganya belum ada

| Hal | Keadaan |
|---|---|
| **"Pertanyaan Dari Analyst"** | terlihat di layar lama, **belum dibangun** — tidak ada tempat memuatnya |
| Isian rawat inap | diperlakukan **selalu tampil**, karena penentunya tidak ada. Menyembunyikan isian yang mungkin seharusnya terisi lebih merugikan daripada menampilkan yang mungkin tidak perlu |

### Satu jebakan yang sengaja TIDAK diambil

`TanggalSelesaiRawatInap` punya satu kemunculan di seluruh skema:
`LOG_PC_ASM_FW_GCNMFW_WORK.TANGGALSELESAIRAWATINAP_1` — kolom berakhiran `_1` pada tabel
**LOG** milik engine Pega.

Pola yang sama sudah menjebak modul ini sekali: `SURVEYDATE_1` ada di katalog tetapi
**kosong pada seluruh 2.646 klaim**. Kolom itu karena itu **tidak dipakai**, dan permintaan
di atas tetap berlaku.

### Catatan untuk DBA

`T_CLAIM_PNC` kini **106 kolom**, bukan 82 seperti yang tercatat setelah pemangkasan
2026-09-26 — ia bertambah sejak itu. Ketiga nama di atas tetap tidak ada di antaranya, jadi
ketiadaannya **bukan akibat pemangkasan tersebut**.

Perubahan ini menyentuh tabel yang dibaca **116 rule Pega** yang sedang melayani produksi,
sehingga `D-63` menuntutnya diuji dengan menjalankan Pega dan Go bersamaan. Penambahan kolom
*nullable* bersifat backward-compatible (`P-4`), sehingga Pega tidak terpengaruh.
