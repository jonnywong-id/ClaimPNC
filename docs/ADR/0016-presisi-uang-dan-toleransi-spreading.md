# 0016 — Simpan nilai uang presisi penuh; validasi total spreading dengan toleransi empat desimal

Status: Accepted
Tanggal keputusan: 2026-09-11    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-51`, `D-49` | `Activity/InputRegister_act-Act.xml:13183`
Terkait: CONTEXT.md#Spreading, ADR-0014, ADR-0017, modul `B-4`, `B-5`

## Konteks

Spreading reasuransi membagi risiko sebuah klaim menjadi beberapa bagian yang totalnya harus
**100%**. Sistem lama memvalidasinya begini:

```
@contains(local.totalspreading, 100.0) || local.totalspreading == 100 || @contains(local.totalspreading, 99.99)
```
`Activity/InputRegister_act-Act.xml:13183`

Ini **pencocokan substring, bukan perbandingan numerik** (T-3). Akibatnya total `199.99` dan
`1100.0` ikut lolos — keduanya memuat potongan teks yang dicari — sementara maksud aturannya
adalah "mendekati seratus". Tidak ada pembulatan sama sekali dalam perhitungannya.

Persoalan kedua: sampai berapa desimal nilai uang disimpan, dan kapan dibulatkan. Pembulatan per
langkah perhitungan menumpuk selisih; pembulatan saat menyimpan menghilangkan informasi secara
permanen.

## Opsi yang dipertimbangkan

1. Bulatkan saat menyimpan, sehingga angka tersimpan sama dengan angka tampil.
2. Bulatkan di setiap langkah perhitungan.
3. Pertahankan perilaku lama apa adanya.
4. **Simpan presisi penuh, bulatkan hanya saat ditampilkan.**

## Keputusan

**Nilai uang disimpan presisi penuh.** Pembulatan dilakukan **hanya saat ditampilkan** — tidak
saat menyimpan, dan tidak per langkah perhitungan. Perbandingan terhadap ambang (termasuk ambang
komite, ADR-0014) memakai **nilai presisi penuh**.

**Total spreading reasuransi** divalidasi dengan pembulatan 4 desimal dan toleransi:

```sql
ROUND(SUM(share), 4) BETWEEN 99.9999 AND 100.0001
```

Ini menggantikan pencocokan substring di `InputRegister_act-Act.xml:13183` — butir pertama pada
daftar perbaikan `D-49` (ADR-0017).

Kriteria penerimaan `B-4` yang langsung dapat dipakai: `100.0000` diterima · `99.9999` diterima
(batas bawah) · `100.0001` diterima (batas atas) · `199.99` **ditolak** · `1100.0` **ditolak**.

## Rationale

Toleransi dibutuhkan karena share reasuransi memang kerap tidak berjumlah tepat 100% setelah
pembagian — itu kenyataan bisnis, bukan cacat. Yang salah pada sistem lama bukan adanya toleransi,
melainkan **cara toleransi itu diperiksa**.

Presisi penuh saat menyimpan menjaga agar pembulatan tidak menumpuk sepanjang rantai
`Estimasi → Usulan → Akseptasi → Dibayar`. Membulatkan saat menyimpan berarti setiap tahap
mewarisi selisih tahap sebelumnya.

## Konsekuensi

### Positif

- `199.99` dan `1100.0` tidak lagi lolos validasi. Kelas cacat yang sudah berjalan tertutup.
- Nilai yang tersimpan dapat direkonsiliasi dengan sumbernya tanpa selisih pembulatan.
- Ambang komite dibandingkan terhadap angka yang sama persis setiap kali — jenjang persetujuan
  menjadi dapat diulang hasilnya.

### Negatif / utang teknis

- **Angka tersimpan tidak selalu sama dengan angka yang dilihat pengguna.** Setiap laporan,
  export, dan layar harus konsisten menerapkan aturan pembulatan tampilan — dan satu tempat yang
  lupa akan menampilkan angka berbeda untuk data yang sama.
- **Uji kesetaraan akan menemukan selisih pada data yang sebelumnya lolos**: klaim historis dengan
  total spreading `199.99` valid menurut sistem lama dan tidak valid menurut sistem baru. Selisih
  itu harus dinyatakan sebagai perbaikan yang direncanakan.
- Tipe data desimal presisi penuh menuntut kedisiplinan di seluruh lapisan — database, Go, JSON
  API, dan JavaScript. **Angka pecahan di JavaScript tidak presisi**, sehingga nilai uang tidak
  boleh melewati batas itu sebagai angka.

### Risiko yang diterima secara sadar

- Toleransi `99,9999`–`100,0001` adalah batas yang dipilih; klaim dengan total di luar batas itu
  akan ditolak meski secara bisnis mungkin masih wajar.
- Pengguna yang terbiasa melihat angka bulat akan menemukan selisih tampilan pada beberapa
  laporan, dan itu akan dilaporkan sebagai "bug" sebelum dipahami sebagai perbaikan.

## Pertanyaan terbuka

- Berapa desimal yang dipakai pada tampilan untuk nilai Rupiah dan untuk persentase share?
  Pemilik: Work Owner. Menghalangi penyelesaian tiket `U-2`.
- Berapa presisi yang disepakati untuk kolom uang di skema baru? Pemilik: Lead Engineer + DBA.
