# 0014 — Hitung jenjang komite secara kumulatif menurut ambang bawah, dengan pita nilai hanya di Non-MBU

Status: Accepted
Tanggal keputusan: 2026-09-11 (`D-47`, `D-52`), 2026-09-14 (`D-70`)    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-47`, `D-52`, `D-70`, `D-14` | `Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459`, `:16501` | `When/IsKomiteLoop-When.xml` | `RDB List/EmailKomiteBerjenjang_sql-SQL.xml:52` | `Database/emailkomite.csv`
Terkait: CONTEXT.md#Jenjang-Kumulatif, CONTEXT.md#Pita-Nilai-Komite, ADR-0017, ADR-0027, modul `B-7`

## Konteks

Komite menyetujui nilai klaim secara berjenjang. Cara jumlah jenjang ditentukan **tidak
terdokumentasi di mana pun** dan hanya dapat dibaca dari kode.

Mekanismenya, terverifikasi:

| Bukti | Isi |
|---|---|
| `Activity/SetListComiteeClaimPerObjAdj-Act.xml:16456-16459` | `childPageKomite.KomiteLoop := TempRDBSearchEmailKomite.pxResultCount` |
| `When/IsKomiteLoop-When.xml` | `.AcceptStatus = "1"` **dan** `.KomiteCount <= .KomiteLoop` |

**Jumlah jenjang persetujuan sama dengan jumlah baris yang dikembalikan kueri.** Dan kueri itu
menyaring **hanya dengan ambang bawah**:

```sql
... AND LIMIT_BOTTOM <= {tempAdj.ConvertAdjustmentValue}
```

`LIMIT_BOTTOM` muncul di **11 SQL rule**; `LIMIT_TOP` muncul di **0**. Batas atas tercatat di
master tetapi tidak pernah dipakai menyaring — itulah sebabnya hasilnya banyak baris, bukan satu.

Dari 17 kueri yang membaca `EMAILKOMITE`, hanya **4** yang memfilter `TYPE_KOMITE`. Jalur utama
**PA** (`EmailKomiteBerjenjangPA_sql-SQL.xml:5`) dan **Travel**
(`EmailKomiteBerjenjangTravel_sql-SQL.xml:103`) tidak memfilternya — dan isi master menjelaskan
mengapa:

| Lini | Ambang bawah per jenjang | `TYPE_KOMITE` |
|---|---|---|
| **NONMBU** | `0` · `50.000.001` | `1` · `1` |
| **NONMBU** | `100.000.001` · `500.000.001` · `1.000.000.001` | `2` · `2` · `2` |
| **PA** | `0` · `10.000.001` · `50.000.001` · `100.000.001` | **`2` · `1` · `1` · `2`** |
| **TRAVEL** | `0` · `50.000.001` · `100.000.001` | `1` · `1` · `1` |

Pada Non-MBU, `TYPE_KOMITE` memang memisahkan pita nilai. Pada PA ia berselang-seling menaiki
tangga — di sana ia membedakan **jalur PA reguler dari PA TKI**
(`EmailKomiteBerjenjangPATKI_sql-SQL.xml:82` mematok `type_komite='2'`), bukan pita nilai.

## Opsi yang dipertimbangkan

1. **Rentang tertutup** `LIMIT_BOTTOM <= nilai <= LIMIT_TOP` — satu jenjang per klaim.
2. Hanya batas bawah, dengan aturan "DEGREE tertinggi yang memenuhi menang".
3. **Kumulatif** — seluruh jenjang yang memenuhi harus menyetujui, `LIMIT_TOP` menjadi validasi
   integritas master.
4. Untuk filter pita: berlaku seragam di semua lini, atau per lini sesuai kuerinya hari ini.

## Keputusan

**Model kumulatif dipertahankan.** Jumlah jenjang persetujuan = jumlah baris `EMAILKOMITE` yang
memenuhi `LIMIT_BOTTOM <= nilai klaim` (ditambah filter `STS_ADJ`, `STS_AKTIF`, `TYPE_BUSINESS`),
diurutkan `DEGREE`.

**`LIMIT_TOP` dipakai sebagai validasi integritas master data**, bukan untuk memilih baris:
sistem baru menolak master yang rentangnya tumpang tindih atau berlubang antar `DEGREE` dalam satu
`TYPE_BUSINESS` + `TYPE_KOMITE`.

**Filter pita nilai (`TYPE_KOMITE`) berlaku khusus lini Non-MBU** (`D-70`). Di sana pita dipilih
lebih dulu — ≤ Rp 100.000.000 → `1`, di atasnya → `2` — lalu akumulasi berjalan di dalam pita itu
saja. **Di lini lain tidak ada langkah pendahuluan**; akumulasi berjalan atas seluruh jenjang lini
tersebut, persis seperti kuerinya hari ini.

`D-70` **membatasi cakupan `D-52`**, tidak membatalkannya: isi `D-52` tetap benar untuk Non-MBU.

## Rationale

Menerapkan rentang tertutup akan mengembalikan tepat satu baris → `KomiteLoop = 1` → **hanya satu
jenjang persetujuan berapa pun nilai klaim**, menghapus penjenjangan yang menjadi inti `D-14` dan
`BRD §11.4`.

Menyeragamkan filter pita ke semua lini terbukti lebih buruk lagi. Dihitung dari master:

| Kasus | Perilaku sistem lama | Bila pita disaring seragam |
|---|---|---|
| PA Rp 5.000.000 | 1 penyetuju | **0 penyetuju** — klaim mandek |
| PA Rp 75.000.000 | 3 penyetuju | 2 penyetuju |
| PA Rp 150.000.000 | 4 penyetuju | 2 penyetuju |
| Travel Rp 150.000.000 | 3 penyetuju | **0 penyetuju** — klaim mandek |
| Non-MBU seluruh nilai | — | tidak berubah |

Dua kasus menghasilkan klaim tanpa penyetuju sama sekali. Itu perubahan perilaku, bukan
perbaikan, dan melanggar `P-5`.

## Konsekuensi

### Positif

- Perilaku sistem lama dipertahankan persis di setiap lini, sehingga gerbang 1 `B-7` dapat
  dijalankan tanpa pengecualian.
- Penjenjangan menjadi terdokumentasi untuk pertama kalinya.
- `LIMIT_TOP` yang selama ini mati mendapat kegunaan nyata: menjaga master tetap konsisten saat
  diisi.

### Negatif / utang teknis

- **Satu kolom memikul dua arti berbeda.** `TYPE_KOMITE` berarti pita nilai di Non-MBU dan varian
  jalur di PA. Pemisahannya menjadi dua kolom dipertimbangkan dan **tidak diambil**, sehingga arti
  gandanya wajib didokumentasikan di model data baru.
- **Validasi `LIMIT_TOP` hanya dapat dijalankan per lini**, tidak lintas lini — karena tangga PA
  memang tidak tersusun menurut pita.
- Aturan penjenjangan berbeda antarlini membuat `B-7` tidak dapat ditulis sebagai satu model
  tunggal; ia adalah beberapa model yang berbagi mekanisme.
- Jumlah penyetuju bergantung penuh pada isi master. Satu baris master yang salah mengubah
  kewenangan persetujuan tanpa ada perubahan kode.

### Risiko yang diterima secara sadar

- Kerangka pertanyaan atas mekanisme ini **pernah salah dua kali** selama analisis (Q12 dan Q32).
  Ia mudah disalahpahami, dan setiap perubahan di masa depan berisiko mengulang kesalahan yang
  sama.
- Dua kueri jalur Simasnet memakai `dbms_random.value` dalam pemilihan komite; jalur utama
  memakai `ORDER BY DEGREE` dan deterministik. Ketidakseragaman ini dibawa apa adanya.

## Pertanyaan terbuka

- **Belum terverifikasi:** tiga kueri yang memfilter `TYPE_KOMITE` secara dinamis —
  `EmailKomiteBerjenjang_sql` (`Activity/SetEmailKomite-Act.xml`), `EmailKomiteAdjuster_sql`
  (`Activity/SetEmailKomiteAdjuster-Act.xml`), `EmailKomiteSalvage_sql`
  (`Activity/SetEmailKomiteSalvage-Act.xml`) — menerima nilainya dari pemanggil lewat
  `tempAdj.pyMemo` dan `tempAdj.AcceptedNo`. **Lini apa saja yang benar-benar melewati ketiga
  kueri itu belum ditelusuri sampai ke sumber nilainya.** Pemilik: Lead Engineer. Ini bagian
  Definition of Ready tiket `B-7` dan `B-12`.
- Apakah `TYPE_KOMITE` kelak dipecah menjadi dua kolom saat master diisi ulang di `F-4`?
  Pemilik: Work Owner.
