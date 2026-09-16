# 0005 — Tulis satu set SQL portabel: jalan di Oracle sekarang, PostgreSQL 17+ kemudian

Status: Accepted
Tanggal keputusan: 2026-09-07    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-01`, `D-20` (menyupersede `D-06`), `D-24`, `D-02` | 652 rule SQL pada export
Terkait: ADR-0004, ADR-0007, ADR-0009, modul `F-2`

## Konteks

`D-01` menetapkan dua hal yang tampak bertentangan: target akhir adalah **PostgreSQL untuk
seluruh data**, sementara **runtime sementara tetap Oracle 19c** dan tanggal cutover belum
ditentukan. Aplikasi harus berjalan di Oracle hari ini tanpa mengorbankan desain PostgreSQL-first.

`D-06` sempat mengangkat persoalan ini tetapi belum memilih mekanismenya; pertanyaan diajukan
ulang setelah seluruh **652 rule SQL** dianalisis, dan diputuskan sebagai `D-20`.

Volume yang terukur dari export menentukan bentuk keputusannya: **411 pemakaian `TO_CHAR`** dan
**68 pemakaian `ROWNUM`**.

## Opsi yang dipertimbangkan

1. **Repository interface dengan dua implementasi SQL** berdampingan (Oracle dan PostgreSQL).
2. **Satu set SQL portabel**, dialek khusus hanya di sedikit tempat yang dikelola.
3. **Query builder / ORM** yang menangani dialek otomatis.
4. **Bangun untuk PostgreSQL saja**, Oracle diakses lewat lapisan kompatibilitas sementara.

## Keputusan

**Satu set SQL portabel** yang berjalan di Oracle 19c dan PostgreSQL 17+, dengan **tiga
pengecualian yang dikelola secara sadar**:

1. **Pemformatan tanggal dan angka dikeluarkan dari SQL ke Go**, menghapus 411 pemakaian
   `TO_CHAR`.
2. **Generator nomor klaim** menjadi satu-satunya tempat dengan sakelar dialek eksplisit
   (ADR-0009).
3. **Paginasi diseragamkan ke `OFFSET … FETCH NEXT … ROWS ONLY`**, menggantikan 68 pemakaian
   `ROWNUM`. Pola ini sudah dipakai di 35 rule pada codebase yang ada.

**PostgreSQL 17 atau lebih baru adalah persyaratan teknis mengikat** (`D-24`), bukan preferensi.
Tanpa itu keputusan ini tidak dapat dijalankan.

Desain skema, tipe data, indexing, dan gaya SQL mengacu ke **PostgreSQL sebagai kanonikal**.

## Rationale

Dua implementasi SQL berdampingan berarti setiap kueri ditulis dan diuji dua kali, oleh tim yang
sedang belajar Go. Biaya itu berlangsung sepanjang masa paralel yang belum bertanggal akhir.

ORM menyembunyikan SQL justru pada aplikasi yang inti kerumitannya **ada di SQL** — 652 rule,
sebagian dengan 97 predikat join. Menyembunyikannya memindahkan kerumitan, bukan menguranginya.

Mengeluarkan `TO_CHAR` ke Go bukan sekadar demi portabilitas: SQL yang ada **mengembalikan
tanggal sebagai string `'dd/mm/yyyy'`**, sehingga pengurutan dan penyaringan tanggal salah secara
diam-diam. Satu keputusan portabilitas sekaligus menutup cacat yang sudah berjalan.

## Konsekuensi

### Positif

- Satu kueri, satu tempat diuji, dua mesin database.
- Perpindahan ke PostgreSQL kelak menjadi peristiwa infrastruktur, bukan penulisan ulang aplikasi.
- Tanggal berpindah sebagai tipe tanggal, bukan string — memperbaiki pengurutan dan penyaringan
  yang selama ini salah.

### Negatif / utang teknis

- **Fitur khas Oracle tidak boleh dipakai** meski tersedia dan kadang lebih cepat: `result_cache`,
  `JSON_OBJECT_T`, `DBMS_AQ`, hierarki `CONNECT BY`.
- Portabilitas hanya dapat dibuktikan bila **kedua mesin benar-benar diuji**. Tanpa lingkungan
  PostgreSQL sejak awal, klaim "portabel" tidak terverifikasi sampai cutover — dan saat itu sudah
  terlambat.
- Perubahan `ROWNUM` → `OFFSET … FETCH NEXT` mengubah rencana eksekusi pada beberapa kueri; ini
  **perubahan kinerja yang harus diukur**, bukan penggantian sintaks.
- `D-24` mengunci versi minimum PostgreSQL. Bila infrastruktur hanya menyediakan versi lebih
  rendah, seluruh ADR ini gugur.

### Risiko yang diterima secara sadar

- Menulis SQL portabel di atas Oracle berarti sebagian optimasi yang wajar untuk Oracle
  ditinggalkan hari ini demi keuntungan yang baru datang saat cutover.
- Tanggal cutover belum ditentukan (`D-01`), sehingga masa "membayar biaya portabilitas tanpa
  menikmati hasilnya" tidak berbatas.

## Pertanyaan terbuka

- Kapan lingkungan PostgreSQL 17+ tersedia untuk pengujian portabilitas? Pemilik: Tim Infra.
  Selama belum ada, tidak ada tiket yang boleh menyatakan SQL-nya "terbukti portabel".
- Apakah 97 predikat join pada kueri terberat tetap berkinerja wajar di PostgreSQL? Pemilik:
  DBA + Lead Engineer. Menghalangi kepastian NFR kinerja.
