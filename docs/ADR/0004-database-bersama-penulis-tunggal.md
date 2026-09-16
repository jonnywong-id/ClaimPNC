# 0004 — Pakai satu database bersama selama masa paralel, dengan penulis tunggal per tabel

Status: Accepted
Tanggal keputusan: 2026-09-07    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-21`, `D-63`, `D-05`, `P-1`, `P-4` | `docs/verifikasi-bukti-adr.md` §14
Terkait: ADR-0003, ADR-0005, ADR-0007, ADR-0012, ADR-0027, seluruh modul

## Konteks

Selama masa paralel (ADR-0003), Pega dan aplikasi Go melayani klaim yang sama pada hari yang
sama. Keduanya harus melihat data yang sama — status klaim, penugasan, nilai estimasi — tanpa
jeda sinkronisasi.

Data itu hari ini hidup di **tabel milik engine Pega**, dan sebagian dibaca sangat luas:

| Tabel Pega | Dibaca oleh | Isi |
|---|---|---|
| `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` | **116 rule** | header klaim |
| `DATAPEGA.PC_ASSIGN_WORKLIST` | 18 rule | penugasan per orang |
| `DATAPEGA.PC_ASSIGN_WORKBASKET` | 6 rule | antrean bersama |
| `DATAPEGA.PR_OPERATORS` | 6 rule | master pengguna |
| `DATAPEGA.PC_LINK_ATTACHMENT` | 3 rule | kaitan lampiran |

Angka 116 itu yang menentukan sifat keputusan ini: satu `ALTER` yang keliru pada tabel header
klaim menghentikan sistem yang sedang melayani produksi.

## Opsi yang dipertimbangkan

1. **Satu database bersama**, dengan tabel baru milik Claim PNC menggantikan tabel engine Pega
   secara bertahap.
2. **Database terpisah + sinkronisasi dua arah** (CDC, replikasi, atau antrean pesan).
3. **Database terpisah + integrasi lewat API** antara Pega dan Go.

## Keputusan

Kedua sistem memakai **satu database yang sama** selama masa paralel. Data yang hari ini hidup di
tabel engine Pega dipindahkan ke **tabel baru milik aplikasi Claim PNC** — tabel yang belum ada
dan dirancang di proyek ini.

Tiga aturan mengikat berlaku sepanjang masa paralel:

1. **Penulis tunggal per tabel (`P-1`).** Untuk setiap tabel, tepat satu sistem berwenang
   menulis. Sistem yang lain hanya membaca. Kewenangan berpindah saat modul pemiliknya lulus
   gerbang 2, bukan sebelum itu.
2. **Perubahan skema wajib backward-compatible (`P-4`).** Kolom dihapus dalam dua tahap, tidak
   pernah sekali jalan. Setiap tiket yang menyentuh skema memuat bagian rollback yang tidak
   kosong.
3. **Perubahan skema dijalankan DBA** atas permintaan tertulis tim pengembang dengan persetujuan
   Work Owner, dan **wajib diuji dengan menjalankan Pega dan Go bersamaan** terhadap skema hasil
   perubahan (`D-63`).

## Rationale

Sinkronisasi dua arah antar database adalah sumber selisih data yang paling sulit ditelusuri, dan
sistem ini menangani uang klaim **tanpa jejak audit atas perubahan nilai di sistem lama** (T-14).
Selisih yang muncul tidak akan punya sumber pembanding.

Integrasi lewat API antara Pega dan Go menuntut perubahan besar di sisi Pega — tepatnya di
aplikasi yang sedang ditinggalkan, dikerjakan oleh tim yang sedang berpindah teknologi.

Database bersama memindahkan seluruh persoalan konsistensi ke satu tempat yang sudah memiliki
mekanisme untuk itu: transaksi database. Harganya adalah kopling, dan kopling itu dijinakkan
dengan aturan penulis tunggal.

## Konsekuensi

### Positif

- Tidak ada jeda sinkronisasi dan tidak ada konflik penggabungan data.
- Uji kesetaraan gerbang 1 (ADR-0027) dapat membandingkan hasil di atas **data yang benar-benar
  sama**, bukan dua salinan yang mungkin berbeda.
- Peralihan kewenangan per tabel menjadi penanda kemajuan migrasi yang konkret dan dapat diaudit.

### Negatif / utang teknis

- **Kopling terkuat yang mungkin ada** antara sistem lama dan baru. Selama masa paralel, kedua
  aplikasi tersandera skema yang sama.
- **Iterasi melambat secara permanen selama masa paralel**: tiga pihak untuk setiap perubahan
  skema (`D-63`).
- Aplikasi Go tidak dapat merancang skemanya secara ideal; ia harus hidup berdampingan dengan
  tabel yang bentuknya ditentukan engine Pega.
- Aturan penulis tunggal **tidak ditegakkan mesin mana pun**. Ia hanya disiplin manusia, dan
  pelanggarannya baru terlihat sebagai data rusak.

### Risiko yang diterima secara sadar

- Satu `ALTER` keliru pada `PC_ASM_FW_GCNMFW_WORK` menghentikan produksi yang dilayani 116 rule
  Pega. Mitigasinya adalah prosedur `D-63`, bukan mekanisme teknis.
- Tabel baru milik Claim PNC **belum ada sama sekali** dan harus dirancang dari nol, sementara
  bentuk tabel Pega hanya dapat dibaca dari cara rule memakainya — bukan dari dokumentasi.

## Pertanyaan terbuka

- Bagaimana pelanggaran aturan penulis tunggal dideteksi — pemeriksaan berkala, trigger audit,
  atau tidak sama sekali? Pemilik: Lead Engineer + DBA. Tanpa jawaban, `P-1` hanya imbauan.
- Apakah aplikasi Go mendapat skema (schema) Oracle sendiri, atau menumpang skema yang ada?
  Pemilik: DBA + Work Owner. Menghalangi penulisan tiket `F-2`.
