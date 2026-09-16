# Database Strategy — Claim PNC

Mengacu pada **PostgreSQL 17+ sebagai desain kanonikal** (D-01, D-24), berjalan di **Oracle 19c
untuk sementara** dengan **satu set SQL portabel** (D-20), **tanpa pemanggilan stored procedure**
(D-02).

> **Diperbarui v2.0 (2026-09-14).** Yang berubah di bab ini: format nomor klaim menjadi
> **`PNCN.YY.xxxx`** (`D-71`) · **soft delete menyeluruh** ditambahkan sebagai §8.1 (`D-66`) ·
> retensi jejak audit **mengikuti retensi data klaim** (`D-62`) · prosedur perubahan skema dan
> aturan penulis tunggal per tabel ditetapkan sebagai §9.1 dan §9.2 (`D-63`, `P-1`) · premis
> `OFFSET 500000` **dikoreksi** karena tidak berdasar · dan **stored procedure boleh ditinggalkan**
> setelah logikanya naik ke Go (`D-68`).

---

## 1. Aturan dasar

| # | Aturan | Sumber |
|---|---|---|
| DB-1 | Desain skema, tipe data, indexing, dan gaya SQL mengacu PostgreSQL | D-01 |
| DB-2 | Tidak ada pemanggilan stored procedure dari aplikasi | D-02 |
| DB-3 | Satu set SQL yang berjalan di Oracle 19c dan PostgreSQL 17+ | D-20 |
| DB-4 | Target PostgreSQL **17 atau lebih baru** — persyaratan mengikat | D-24 |
| DB-5 | Satu database bersama dengan Pega selama masa paralel | D-21 |
| DB-6 | Satu tabel hanya boleh ditulis satu sistem | D-21 |
| DB-7 | Migrasi skema selalu backward-compatible | D-27 |
| DB-8 | Seluruh waktu disimpan UTC | F-5 |
| DB-9 | Jejak audit append-only, tidak dapat diubah aplikasi | D-28 |

---

## 2. Kenapa PostgreSQL 17 wajib

Ini bukan preferensi versi, melainkan syarat agar D-20 bisa dijalankan sama sekali.

Sistem lama memakai **222 pemanggilan SQL/JSON** di 29 rule terhadap `JSON_KLAIM.DATA_JSONBLOB`
dan `JSON_POLIS.DATA_JSONBLOB`:

| Fungsi | Volume | Oracle 12c+ | PostgreSQL ≤16 | PostgreSQL 17+ |
|---|---|---|---|---|
| `JSON_VALUE` | 195× | ✅ | ❌ | ✅ |
| `JSON_QUERY` | (termasuk di atas) | ✅ | ❌ | ✅ |
| `JSON_TABLE` | 27× | ✅ | ❌ | ✅ |

PostgreSQL 17 mengimplementasikan fungsi SQL/JSON standar dengan sintaks yang sama seperti
Oracle. Dengan itu, seluruh query JSON portabel apa adanya.

Pada PostgreSQL 16 ke bawah, ke-29 rule harus ditulis ulang memakai operator `jsonb` khas
PostgreSQL (`->`, `->>`, `jsonb_path_query`), sehingga akan ada **dua versi query** — dan
keputusan "satu set SQL portabel" gugur.

---

## 3. Tiga pengecualian portabilitas

Diakui secara sadar, dikelola, dan tidak boleh bertambah tanpa keputusan tertulis.

### 3.1 Generator nomor klaim

Format nomor klaim sistem baru adalah **`PNCN.YY.xxxx`** — tiga segmen dipisahkan titik
(`D-71`, menyupersede bentuk `PNCN-xxxx` pada `D-22`). Sintaks Oracle yang ditetapkan:

```sql
'PNCN' || '.' || TO_CHAR(SYSDATE,'RR') || '.' || TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)
```

| Segmen | Isi | Sumber |
|---|---|---|
| `PNCN` | penanda tetap asal sistem baru | literal |
| `YY` | dua digit tahun | `TO_CHAR(SYSDATE,'RR')` |
| `xxxx` | nomor urut | `TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)` |

**Sequence-nya belum ada** — `POOLDATA.CLAIM_NO_NONPEGA_SEQ` nol kemunculan di seluruh export; ia
akan dibuat, bukan dipakai ulang.

Sejak `D-71`, sakelar dialek di sini membungkus **dua** perbedaan sekaligus, bukan satu: sequence
(Oracle `SEQ.NEXTVAL` versus PostgreSQL `nextval('seq')`) **dan** pemformatan tahun (`TO_CHAR(SYSDATE,'RR')`
versus `to_char(current_date,'YY')`). **Diisolasi di satu berkas**:
`internal/adapter/sqlstore/nomor_klaim.go`. Ini satu-satunya tempat dengan percabangan dialek di
seluruh aplikasi.

**Dua hal yang mengikuti dari sintaks ini dan belum diputuskan** (`ADR-0009`):

1. **Apakah sequence direset tiap awal tahun?** Bila tidak, nomor urut menembus pergantian tahun
   (`PNCN.26.8125` → `PNCN.27.8126`) dan segmen tahun menjadi penanda, bukan penghitung per tahun.
2. **`TO_CHAR(...NEXTVAL)` tanpa format mask tidak memberi nol di depan**, sehingga lebar segmen
   terakhir berubah-ubah dan **pengurutan sebagai teks tidak sesuai urutan penerbitan** (`.10`
   mendahului `.9`). Setiap layar dan laporan yang mengurutkan berdasarkan nomor klaim harus
   menyadarinya.

Catatan ketiga: **tahun diambil dari `SYSDATE`**, yaitu tanggal server basis data — bukan tanggal
kejadian dan bukan tanggal registrasi. Klaim yang terbit di sekitar pergantian tahun mengambil
tahun dari jam server, bertaut dengan `R-12`.

### 3.2 Pemformatan tanggal dan angka
411 pemakaian `TO_CHAR` **dihapus dari SQL**, pemformatan pindah ke Go.

Ini bukan sekadar demi portabilitas. Pola sekarang mengembalikan tanggal sebagai string
`'dd/mm/yyyy'` dari database, sehingga:
- pengurutan tanggal menjadi pengurutan **teks** — `01/12/2024` dianggap lebih kecil dari
  `02/01/2020`;
- penyaringan rentang tanggal tidak bisa memakai index;
- perbandingan tanggal salah tanpa ada yang menyadari.

Memindahkan pemformatan ke Go memperbaiki ketiganya sekaligus.

### 3.3 Paginasi
68 pemakaian `ROWNUM` diganti `OFFSET … FETCH NEXT … ROWS ONLY`, yang didukung Oracle 12c+ dan
PostgreSQL. Pola ini **sudah dipakai di 35 rule** pada sistem lama, jadi bukan hal baru bagi tim.

---

## 4. Padanan sintaks yang mengikat

| Jangan pakai | Pakai | Volume di sistem lama |
|---|---|---|
| `NVL(a, b)` | `COALESCE(a, b)` | 100× |
| `SYSDATE` | `CURRENT_TIMESTAMP` | 69× |
| `DECODE(...)` | `CASE WHEN … END` | 16× |
| `ROWNUM` | `OFFSET … FETCH NEXT … ROWS ONLY` | 68× |
| `INSTR(a, b)` | `POSITION(b IN a)` | 11× |
| `LISTAGG(...)` | `STRING_AGG(...)` | 4× |
| `a = b(+)` | `LEFT JOIN` | 2× |
| `FROM DUAL` | hilangkan klausa `FROM` | 12× |
| `ADD_MONTHS(d, n)` | `d + INTERVAL` | 18× |
| `MONTHS_BETWEEN(a, b)` | hitung di Go | 2× |
| `TRUNC(date)` | `CAST(x AS DATE)` | 150× |
| `TO_CHAR(...)` untuk tampilan | format di Go | 411× |
| `SELECT *` | sebutkan nama kolom | — |
| Perangkaian string SQL (`{ASIS:...}`) | parameter binding | — |

---

## 5. Tipe data

Dipilih agar sama-sama sah di Oracle dan PostgreSQL, dengan PostgreSQL sebagai acuan.

| Kegunaan | PostgreSQL | Oracle | Catatan |
|---|---|---|---|
| Teks pendek | `VARCHAR(n)` | `VARCHAR2(n)` | Panjang eksplisit |
| Teks panjang | `TEXT` | `CLOB` | Kronologi, catatan |
| Bilangan bulat | `BIGINT` | `NUMBER(19)` | — |
| **Nilai uang** | `NUMERIC(18,2)` | `NUMBER(18,2)` | **Tidak pernah** `float`/`double` |
| Persentase share | `NUMERIC(9,6)` | `NUMBER(9,6)` | Presisi cukup untuk aturan 100% |
| Waktu | `TIMESTAMPTZ` | `TIMESTAMP WITH TIME ZONE` | Disimpan UTC (DB-8) |
| Tanggal murni | `DATE` | `DATE` | DOL, tanggal lapor |
| Boolean | `BOOLEAN` | `NUMBER(1)` | Dipetakan di adapter |
| Dokumen JSON | `JSONB` | `CLOB` + `IS JSON` | Snapshot polis |

**Aturan nilai uang tidak bisa ditawar.** Sistem menghitung pembagian share reasuransi dengan
aturan total harus 100%, **toleransi 4 desimal, `99,9999`–`100,0001`** (`D-51`). Pembulatan floating point akan membuat validasi ini
gagal secara acak dan tidak dapat direproduksi.

---

## 6. Strategi indexing

Profil beban: **data besar, konkurensi rendah** (D-10). Optimasi diarahkan ke volume.

### 6.1 Index wajib

| Tabel | Index | Untuk |
|---|---|---|
| Klaim | `nomor_klaim` (unik) | Pencarian utama |
| Klaim | `nomor_polis, prod_ke` | Pencarian per polis, cek duplikat |
| Klaim | `tanggal_kejadian` | Filter dan laporan periode |
| Klaim | `status_klaim, group_panel` | Filter inbox |
| Klaim | `kode_cabang` | Batas visibilitas per cabang |
| Klaim | `(status_proses, dibuat_pada DESC)` | Urutan inbox |
| Penugasan | `(ditugaskan_ke, status)` | Worklist per orang |
| Penugasan | `(workbasket, status)` | Antrean bersama |
| ObjekPertanggungan | `klaim_id` | Muat anak |
| Coverage | `objek_id` | Muat anak |
| SettlementLine | `coverage_id` | Muat anak |
| SettlementLine | `nomor_akseptasi` | Pencarian akseptasi |
| JejakAudit | `(entitas, entitas_id, terjadi_pada DESC)` | Penelusuran riwayat |

### 6.2 Index parsial (PostgreSQL)
Inbox hanya menampilkan klaim yang belum selesai, sedangkan mayoritas baris adalah klaim lama
yang sudah selesai. Index parsial atas klaim yang masih berjalan membuat ukurannya tetap kecil
walau tabel berisi puluhan juta baris.

Oracle tidak punya index parsial — padanannya adalah function-based index. Karena ini urusan
DDL dan bukan query, perbedaannya **tidak melanggar D-20**.

### 6.3 Paginasi
- Inbox dan pencarian: **keyset pagination** (`WHERE (kolom_urut, id) < (nilai, id)`), bukan
  `OFFSET` besar. Pada puluhan juta baris, `OFFSET` bernilai besar memaksa database membaca dan
  membuang seluruh baris sebelum halaman yang diminta.
- `OFFSET … FETCH` hanya untuk halaman-halaman awal atau data yang sudah tersaring sempit.

> **Koreksi premis (2026-09-14).** Angka `OFFSET 500000` yang dipakai dokumen versi sebelumnya
> **tidak berdasar**: `OFFSET` **nol kemunculan** di seluruh export. Masalah nyatanya berbeda dan
> lebih berat — **3.189 grid terikat page list klipboard** Pega, dan `pyMaxRecords=500` pada
> **54 dari 56** laporan. Artinya sistem lama tidak memaginasi hasil besar sama sekali; ia
> **memotongnya di 500 baris**.
>
> Konsekuensinya bagi uji kesetaraan: **paginasi keyset adalah perubahan perilaku, bukan
> pemeliharaan.** Grid yang hari ini memuat seluruh page list akan berperilaku berbeda saat
> dipaginasi di server, dan laporan yang dulu terpotong kini utuh. Keduanya harus diuji per layar,
> bukan diasumsikan setara.

### 6.4 Partisi
Disiapkan tapi **belum diterapkan** di awal.

Tabel Klaim dan JejakAudit adalah kandidat partisi per tahun berdasarkan tanggal registrasi.
Diterapkan hanya bila pengukuran nyata menunjukkan kebutuhannya — bukan di awal. Menerapkan
partisi tanpa data pengukuran adalah menambah kerumitan tanpa bukti manfaat.

---

## 7. Connection pool

Untuk 200–300 pengguna aktif harian dengan dua instance aplikasi (D-27):

| Parameter | Nilai awal | Alasan |
|---|---|---|
| Koneksi maksimum per instance | **20** | 2 instance × 20 = 40 koneksi. Cukup untuk beban ini; koneksi berlebih justru membebani database |
| Koneksi idle | 5 | Menghindari biaya pembukaan koneksi berulang |
| Umur maksimum koneksi | 30 menit | Mencegah koneksi basi di balik firewall/load balancer |
| Idle maksimum | 5 menit | — |
| Batas waktu query | 30 detik (transaksi), 5 menit (laporan) | Laporan besar tidak boleh menahan pool transaksi |

**Pool terpisah untuk laporan.** Query laporan berjalan lama dan bervolume besar. Bila memakai
pool yang sama, satu laporan berat bisa menghabiskan seluruh koneksi dan membuat pengguna lain
tidak bisa bertransaksi. Pool laporan diberi batas koneksi lebih kecil dan batas waktu lebih
panjang.

---

## 8. Jejak audit

Wajib per D-28. Dirancang sejak awal, bukan ditambal.

**Isi setiap baris:** entitas, id entitas, jenis aksi, nilai sebelum, nilai sesudah, pelaku,
waktu (UTC), id permintaan.

**Yang wajib diaudit:** nilai estimasi klaim, nilai settlement pada setiap tahap, akseptasi dan
nomornya, keputusan komite, penolakan (RCL), proses ulang (PUCL), perubahan status klaim, dan
pembayaran.

**Sifat append-only ditegakkan di database**, bukan hanya di kode: akun aplikasi hanya diberi
hak `INSERT` dan `SELECT` pada tabel audit — tanpa `UPDATE` maupun `DELETE`. Aturan yang hanya
ada di kode bisa dilanggar oleh kode berikutnya; aturan yang ada di hak akses database tidak.

**Retensi:** **mengikuti retensi data klaim yang berlaku sekarang** (`D-62`) — satu kebijakan
untuk keduanya, bukan kebijakan terpisah. Ini menutup pertanyaan terbuka `D-28`: kebijakannya
sudah ada, yang dibutuhkan hanyalah **angkanya**. Sampai angka itu masuk, `S-5` dibangun dengan
**retensi sebagai parameter konfigurasi**, sehingga modulnya tidak terhalang.

**Kenapa bab ini naik derajat.** `D-59` menetapkan satuan izin adalah menu dan **tidak ada
pemisahan tugas** — satu orang dapat membuat, menyetujui, dan membayarkan satu klaim bila perannya
memiliki ketiga menu itu. Tidak ada kontrol teknis yang mencegahnya, sehingga **jejak audit
menjadi satu-satunya kontrol pengimbang yang tersisa** (`ADR-0023`, `ADR-0026`).

**Dua anti-pola dari sistem lama yang tidak dibawa**, keduanya terbukti di source: `UPDATE` pada
`pooldata.claim_service_log` dan `DELETE` pada `POOLDATA.JSON_KLAIM_LOG`. Log yang dapat diubah
dan dihapus bukan log.

---

## 8.1 Penghapusan data — soft delete menyeluruh

`D-66` menetapkan **tidak ada `DELETE` fisik pada data bernilai bisnis**. Penghapusan dinyatakan
lewat penanda — kolom flag beserta waktu dan pelakunya — bukan lewat pembuangan baris. Dengan
begitu, jejak audit append-only dan kebijakan penghapusan berdiri di atas prinsip yang sama.

**Konsekuensi yang mengikat desain:**

| Hal | Ketetapan |
|---|---|
| Setiap kueri pembaca | **wajib menyaring baris bertanda terhapus**. Satu kueri yang lupa akan menampilkan data yang seharusnya hilang — kelas cacat baru yang tidak ada di sistem lama |
| Keunikan kunci alami | baris yang "terhapus" **masih menempati nilai kuncinya**; constraint unik harus memperhitungkan penanda |
| Indexing dan partisi | dirancang menyadari adanya **baris mati** di atas data historis puluhan juta baris |
| Uji kesetaraan | **tidak boleh** membandingkan `COUNT(*)` tabel — lihat Testing Strategy §6.4 |

**Satu pola lama yang gugur karenanya.** `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503`
memakai **hapus-lalu-sisip-ulang** pada **12 tabel** sebagai mekanisme idempotensi konversi klaim.
Pola itu bergantung pada penghapusan fisik dan **tidak dapat dipertahankan**. Penggantinya —
*upsert* berbasis kunci alami, versioning, atau melarang konversi ulang — **belum diputuskan**
(`ADR-0013`, berstatus `Proposed`).

Catatan pendukung: dua tabel pada blok itu (`T_DLALIST`, `T_PLALIST`) **delete-nya sudah
dikomentari** di sumber (`:497`, `:498`) sementara insert-nya tetap aktif (`:1296`, `:1325`) —
sehingga konversi ulang pada kedua tabel itu **sudah berpotensi menduplikasi baris hari ini**.

---

## 9. Migrasi skema

- Dikelola `golang-migrate` dengan berkas SQL polos yang bisa di-review.
- **Selalu backward-compatible** (DB-7) karena rolling deployment 24/7.
- Penamaan: `NNNN_deskripsi_singkat.up.sql` dan `.down.sql`.
- Setiap migrasi wajib punya `down` yang benar-benar berfungsi.
- Migrasi dijalankan **terpisah dari start aplikasi**, sebagai langkah deployment tersendiri.
  Menjalankannya saat start akan menyebabkan dua instance mencoba bermigrasi bersamaan.

**Urutan untuk perubahan yang tidak kompatibel:**
1. Rilis N: tambah struktur baru, tulis ke lama dan baru, baca dari lama.
2. Rilis N+1: baca dari baru.
3. Rilis N+2: berhenti menulis ke lama.
4. Rilis N+3: hapus struktur lama.

### 9.1 Siapa yang menjalankan perubahan skema

`D-63` menetapkan prosedurnya, dan ia **menempuh tiga pihak**:

| Langkah | Pelaku |
|---|---|
| Permintaan **tertulis** | tim pengembang |
| Persetujuan | **Work Owner** |
| Pelaksanaan | **DBA** |
| Verifikasi | **wajib diuji dengan menjalankan Pega dan Go bersamaan** terhadap skema hasil perubahan |

**Alasannya bukan birokrasi.** `P-4` mewajibkan verifikasi backward-compatible dengan menjalankan
versi lama dan baru bersamaan — itu tidak dapat dilakukan DBA sendirian maupun tim pengembang
sendirian. Dan karena `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` dibaca **116 rule Pega**, satu `ALTER` yang
keliru menghentikan sistem yang sedang melayani produksi.

**Konsekuensi yang diterima:** iterasi melambat selama seluruh masa paralel. Setiap tiket yang
menyentuh skema wajib memuat bagian rollback yang **tidak kosong**.

### 9.2 Penulis tunggal per tabel selama masa paralel

Selama Pega dan Go berjalan bersamaan di atas satu database (`D-21`, `ADR-0004`), berlaku `P-1`:
**untuk setiap tabel, tepat satu sistem berwenang menulis**; yang lain hanya membaca. Kewenangan
berpindah saat modul pemiliknya lulus gerbang 2 — bukan sebelum itu.

> **Aturan ini tidak ditegakkan mesin mana pun.** Ia disiplin manusia, dan pelanggarannya baru
> terlihat sebagai data rusak. Cara mendeteksinya — pemeriksaan berkala, trigger audit, atau tidak
> sama sekali — **belum diputuskan**.

---

## 10. Rencana perpindahan ke PostgreSQL

Dilakukan **setelah** aplikasi Go stabil di Oracle. Terpisah dari migrasi aplikasi.

1. Siapkan PostgreSQL 17+ dengan skema setara (DDL berbeda, query sama).
2. Pindahkan data. Perhatikan: tipe boolean (`NUMBER(1)` → `BOOLEAN`), `CLOB` → `TEXT`/`JSONB`,
   dan zona waktu.
3. Buat sequence `claim_no_nonpega_seq` dengan nilai awal melanjutkan Oracle.
4. Ganti konfigurasi driver dan sakelar dialek generator nomor klaim (§3.1) — **dua perbedaan**:
   `nextval('pooldata.claim_no_nonpega_seq')` dan `to_char(current_date,'YY')`.
5. Jalankan seluruh test suite terhadap PostgreSQL.
6. Jalankan paralel sementara untuk membandingkan hasil.
7. Cutover.

**Yang tidak perlu disentuh:** logika bisnis, query, handler, dan frontend. Inilah imbalan dari
D-20 — dan alasan mengapa disiplin SQL portabel di §4 harus dijaga sejak baris pertama, bukan
diperbaiki menjelang cutover.
