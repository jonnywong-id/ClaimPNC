# 0025 — Pindahkan seluruh nilai hardcode ke master data dan konfigurasi, dan keluarkan rahasia dari kode

Status: Proposed
Tanggal keputusan: —    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Tim Infra/Security (`D-40`) → Work Owner
Jejak bukti: `D-15`, `D-40` (OPEN), `D-67`, T-8, T-9 | `docs/verifikasi-bukti-adr.md` §7
Terkait: ADR-0014, ADR-0015, ADR-0022, modul `F-4`, `F-5`

> **Arah keputusan sudah ada (`D-15`), tetapi bagian rahasianya belum.** `D-40` masih berstatus
> **OPEN** menunggu Tim Infra/Security. Sampai itu terjawab, berkas ini **tidak boleh** dijadikan
> dasar implementasi `F-4`/`F-5` secara utuh.

## Konteks

`D-15` menetapkan seluruh nilai hardcode menjadi konfigurasi atau master data. Verifikasi Fase 1
menunjukkan populasinya **jauh lebih besar** daripada yang tercatat di `docs/BRD.md:453`:

| Jenis | Klaim BRD | Terverifikasi di seluruh export |
|---|---|---|
| Alamat email | 10 | **66 unik** |
| User ID | 4 | **24 unik** + belasan tertanam di dalam teks SQL |
| Ambang komite | 3 | **8 unik** + 7 ambang uang non-komite |
| Hostname penentu perilaku | 3 | **3 unik, 48 perbandingan** — angkanya benar, **daftarnya salah** |

Angka BRD benar untuk lingkupnya (dua rule saja), tetapi memakainya untuk `FR-F4` akan membuat
estimasi master data meleset sekitar **enam kali lipat**.

**Tiga hal membuat ini bukan sekadar memindahkan konstanta:**

1. **Hostname menentukan uang.** `Activity/GetKomiteApproval-Act.xml:335` mengubah ambang komite
   dari **50.000.000 menjadi 3.500** berdasarkan nama server — cara entitas Timor-Leste dibedakan.
   Menghapus hardcode di sini berarti mengubah cara entitas dikenali, bukan memindahkan angka.
2. **Blok `// TESTING` menimpa email produksi.** Empat step di
   `Activity/InputRegister_act-Act.xml:16693`, `:16830`, `:16998`, `:17141` menimpa email pimpinan
   dan Underwriting; precondition-nya **identik** dengan step produksi dan posisinya sesudah, dan
   salah satu penimpanya **alamat Gmail pribadi**. Teori "blok ini nonaktif" sudah diuji dan
   gugur: `//` adalah label blok (dipakai 952× di 279 activity), dan format export ini **tidak
   memiliki elemen aktif/nonaktif sama sekali**.
3. **Kredensial plaintext ada di dalam export** (T-9): **3 password SMTP di 31 lokasi** + **1
   pasang kredensial OAuth**. Lokasinya sudah diserahkan lengkap ke Tim Infra/Security lewat
   dokumen terpisah **di luar repo**; tidak ada satu nilai pun tertulis di dokumen yang di-commit
   (`D-69`).

Temuan **T-8** menutup jalan yang tampaknya paling mudah: sistem lama **tidak punya satu pun
Dynamic System Setting**. Konfigurasi dinamisnya berupa tabel Oracle yang **dikunci per IP
aplikasi** — pola yang bertabrakan langsung dengan dua instans (`D-27`).

`D-67` menambahkan satu aturan yang sudah diputuskan: **tidak ada akun pribadi sebagai penerima
notifikasi**; seluruhnya berasal dari master Penerima Notifikasi berupa mailbox fungsional.

## Opsi yang dipertimbangkan

**Untuk nilai bisnis** (ambang, email penerima, entitas):
1. Master data di database, dikelola lewat layar `U-6`.
2. Berkas konfigurasi aplikasi.
3. Campuran — nilai yang diubah pengguna bisnis di master, nilai teknis di konfigurasi.

**Untuk rahasia** (password SMTP, kredensial OAuth) — inilah yang menunggu `D-40`:
1. Secret manager korporat, bila ada.
2. Variabel lingkungan yang dikelola tim infra.
3. Berkas konfigurasi terenkripsi di luar repo.

## Konsekuensi bila dibiarkan tidak diputuskan

- **Tiket `F-4` dan `F-5` tidak dapat ditulis lengkap.** `F-4` adalah prasyarat modul bisnis yang
  memakai ambang dan penerima notifikasi.
- **Repo ini menyimpan kredensial plaintext sampai ada keputusan** tentang ke mana ia dipindahkan
  — yang berarti cara repo disimpan dan dibagikan ikut menjadi persoalan keamanan hari ini, bukan
  nanti.
- Pola konfigurasi dinamis belum ditentukan, sehingga ADR-0022 (penjadwal dua instans) ikut
  tertahan.

## Pertanyaan terbuka

1. **Ke mana rahasia dipindahkan, siapa pemiliknya, dan bagaimana rotasinya?** Pemilik: Tim
   Infra/Security. Ini `D-40`, dan ia menghalangi `F-4`, `F-5`, serta seluruh deployment.
2. **Apakah kredensial yang sudah telanjur plaintext di export harus dirotasi** sebelum sistem
   baru jalan? Pemilik: Tim Infra/Security.
3. **Bagaimana entitas (Indonesia, Timor-Leste, Insurtech) dikenali di sistem baru**, kalau bukan
   dari hostname? Pemilik: Work Owner. Ini menentukan ambang komite mana yang berlaku — ADR-0014.
4. Apakah blok `// TESTING` memang tidak seharusnya ada di produksi, dan sejak kapan ia aktif?
   Pemilik: Work Owner. Jawaban "sudah lama aktif" berarti email produksi selama ini tertimpa, dan
   itu temuan operasional tersendiri.
5. Mana nilai yang boleh diubah pengguna bisnis lewat layar, dan mana yang hanya boleh diubah tim
   teknis? Pemilik: Work Owner. Menentukan pembagian antara `F-4` dan `F-5`.
