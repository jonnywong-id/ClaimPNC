# 0024 — Delegasikan autentikasi ke HCC/HCQ dan terbitkan sesi milik aplikasi sendiri

Status: Proposed
Tanggal keputusan: —    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Tim HCC/HCQ (kontrak) → Work Owner (keputusan akhir)
Jejak bukti: `D-07`, T-2 | `docs/verifikasi-bukti-adr.md` §1.1 | `D-56`
Terkait: ADR-0023, ADR-0028, modul `F-3`

> **Belum dapat dinyatakan `Accepted`.** `D-07` sudah memutuskan arahnya, tetapi **kontrak yang
> menjadi tumpuan keputusan itu tidak dapat diverifikasi sama sekali** dari bahan yang ada.
> Berkas ini memuat arah keputusan beserta apa yang menghalanginya — jangan dijadikan dasar
> implementasi sampai pertanyaan di bawah terjawab.

## Konteks

`D-07` menetapkan pembagian berikut:

- **Authentication** didelegasikan ke **API internal HCC/HCQ** (username + password). Responsnya
  memuat profil lengkap: NIK, nama, cabang, jabatan, email.
- **Authorization** dimiliki dan dikelola sepenuhnya oleh Claim PNC lewat tabel yang memetakan
  pengguna → menu/izin (ADR-0023).
- Aplikasi menerbitkan **session/token miliknya sendiri** setelah HCC/HCQ memvalidasi kredensial.

Pemisahan itu sehat: identitas dimiliki sistem identitas, kewenangan dimiliki aplikasi yang
memahami bisnisnya.

**Yang menghalangi:** `HCC` dan `HCQ` muncul **2×** di seluruh export, dan **keduanya teks pesan
galat** yang menyuruh pengguna menghubungi helpdesk (T-2). Tidak ada satu pun:

- Connect REST ke HCC/HCQ,
- pemetaan field respons,
- penanganan kegagalan autentikasi,
- mekanisme sesi yang dapat dijadikan pembanding.

Artinya integrasi ini adalah **greenfield sepenuhnya**. `F-3` tidak punya baseline Pega untuk
diuji kesetaraannya (ADR-0028), sehingga **seluruh kelulusannya bertumpu pada kontrak yang belum
ada**.

## Opsi yang dipertimbangkan

**Opsi 1 — Jalankan `D-07` apa adanya** begitu kontrak HCC/HCQ diperoleh.
Sesuai keputusan yang sudah diambil; seluruh jadwal `F-3` bergantung pada pihak luar.

**Opsi 2 — Rancang lapisan autentikasi dengan antarmuka yang dapat diganti**, dan implementasi
sementara berbasis tabel pengguna lokal untuk pengembangan dan pengujian.
`F-3` dapat berjalan tanpa menunggu; tetapi jalur yang dipakai di pengembangan bukan jalur yang
dipakai di produksi, sehingga kelas cacat integrasi baru muncul terlambat.

**Opsi 3 — Autentikasi lokal sepenuhnya**, tidak memakai HCC/HCQ.
Menghapus ketergantungan pada pihak luar; tetapi memindahkan penyimpanan dan pengelolaan kata sandi
ke aplikasi ini — tanggung jawab keamanan yang jauh lebih besar dan bertentangan dengan `D-07`.

## Konsekuensi bila dibiarkan tidak diputuskan

- **Tiket `F-3` tidak dapat ditulis lengkap**, dan `F-3` adalah prasyarat bagi `F-4` dan seluruh
  modul bisnis yang memerlukan identitas pengguna.
- Tanpa autentikasi, **tidak ada satu pun modul yang dapat dirilis ke pengguna** — ini jalur kritis
  paling awal di seluruh rencana migrasi.
- Kriteria kelulusan `F-3` tidak dapat dirumuskan: ADR-0028 menggantikan gerbang 1 dengan uji
  fungsional terhadap kontrak, dan kontrak itulah yang tidak ada.

## Pertanyaan terbuka

1. **Apakah API HCC/HCQ benar-benar ada hari ini, atau harus dibangun?** Pemilik: Tim HCC/HCQ.
   Ini pertanyaan pertama yang harus dijawab sebelum `F-3` dijadwalkan.
2. **Bagaimana bentuk kontraknya** — endpoint, format permintaan dan respons, kode galat, batas
   percobaan? Pemilik: Tim HCC/HCQ.
3. **Apa yang terjadi bila HCC/HCQ tidak dapat dihubungi?** Seluruh aplikasi tidak dapat diakses,
   atau ada jalur cadangan? Pemilik: Work Owner. Ini menyentuh tuntutan 24/7 `D-27`.
4. **Bagaimana pengguna dicocokkan** antara identitas HCC/HCQ dan `OPERATOR_ID` yang dipakai di
   seluruh data klaim? Pemilik: Work Owner + Tim HCC/HCQ. Tanpa pemetaan ini, pengguna yang
   berhasil login tetap tidak dikenali oleh data klaimnya sendiri.
5. Berapa lama sesi berlaku, dan bagaimana ia diperbarui? Pemilik: Work Owner + Security.
