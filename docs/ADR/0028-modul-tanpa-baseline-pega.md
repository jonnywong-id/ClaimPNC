# 0028 — Ukur modul tanpa baseline Pega dengan kontrak fungsional, dan cabut `BRD §21.4` untuk empat modul

Status: Accepted
Tanggal keputusan: 2026-09-12    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-56`, `D-55`, `D-42`, T-2, T-14 | `BRD §21.2` kriteria #3 | `BRD §21.4`
Terkait: ADR-0024, ADR-0026, ADR-0027, modul `F-3`, `S-5`, `B-5`, `B-7`, `B-9`, `B-10`, `B-12`

> **Menyupersede `BRD §21.4`** untuk `B-7`, `B-9`, `B-10`, dan `B-12`. Revisi BRD-nya diajukan
> sebagai pertanyaan terbuka di bawah — BRD **tidak** diedit oleh ADR ini.

## Konteks

Gerbang 1 (ADR-0027) membandingkan perilaku sistem baru dengan sistem lama. Dua modul **tidak
punya sistem lama untuk dibandingkan**:

- **`F-3` Identitas & Akses** — HCC/HCQ **nol jejak di export** (T-2). Tidak ada mekanisme
  autentikasi yang dapat dijadikan pembanding.
- **`S-5` Jejak Audit** — sistem lama **tidak punya jejak audit atas nilai** (T-14). Tidak ada
  keluaran yang dapat dibandingkan.

Untuk keduanya, `BRD §21.2` kriteria #3 (uji kesetaraan) **tidak dapat diberlakukan**.

Persoalan kedua datang dari sisi lain: `BRD §21.4` menahan sejumlah modul sampai penghalangnya
hilang. Setelah export bertambah pada 2026-09-09 dan `Database/` diterima, sebagian penghalang itu
**sudah tidak ada lagi** — sementara pasalnya masih mengikat.

## Opsi yang dipertimbangkan

1. **Ganti gerbang 1 dengan uji fungsional terhadap kontrak** yang disepakati.
2. Hapus gerbang 1 untuk kedua modul, cukup UAT.
3. Tunda kedua modul sampai ada pembanding.

## Keputusan

**Untuk `F-3` dan `S-5`, gerbang 1 diganti uji fungsional terhadap kontrak yang disepakati:**

| Modul | Kontrak yang menjadi acuan | Pemilik kontrak |
|---|---|---|
| **F-3** | kontrak API **HCC/HCQ** — field request/response login, kode galat, timeout, endpoint refresh/validasi | pemilik API HCC/HCQ |
| **S-5** | **daftar peristiwa wajib audit** beserta field yang harus tercatat | Compliance |

**`BRD §21.4` dicabut untuk `B-7`, `B-9`, `B-10`, dan `B-12`**, dan **dipertahankan hanya untuk
`B-5`**. Sisa penghalang per modul:

| Modul | Status | Sisa penghalang |
|---|---|---|
| **B-7** Komite | lepas dari §21.4 | PA dan Travel di atas Rp 200.000.000 tidak punya baris master |
| **B-9** PLA/DLA | lepas dari §21.4 | status `VALID`/`INVALID` `GET_GROUPBUSINESS_XOL` di produksi; penulisan ulang procedure ber-9-commit (ADR-0007) |
| **B-10** Akseptasi | lepas dari §21.4 | 3 activity hilang: `InsertDataAkseptasiToLeader`, `InsertLogKasir_act`, `TransferCashierDataASM_act` |
| **B-12** Salvage | lepas dari §21.4 | `SET_ATTACHFILETEMPSALVAGE`; hitungan DBA atas baris ber-`IDSALVAGE` NULL |
| **B-5** Settlement | **tetap terikat §21.4** | isi `POOLDATA.GCNM_FEE_SCALE` (17 pita) · isi `m_currencystandard` · `BrowseT_Claim_Adjustment_SQL` |

## Rationale

Menghapus gerbang 1 (opsi 2) justru pada dua modul **paling sensitif** — `F-3` adalah otorisasi,
`S-5` adalah jejak audit — menyisakan hanya UAT sebagai pemeriksaan. Dan UAT tidak memeriksa hal
yang tidak terlihat di layar: tidak ada penguji bisnis yang dapat melihat apakah sebuah perubahan
nilai benar-benar tercatat.

Menunda kedua modul (opsi 3) mustahil: `F-3` adalah prasyarat seluruh modul yang memerlukan
identitas pengguna.

Mempertahankan `§21.4` untuk modul yang penghalangnya sudah hilang berarti menahan pekerjaan atas
alasan yang tidak lagi berlaku — persis jenis penundaan yang `D-35` tolak (angka mengikuti bukti,
bukan dokumen).

## Konsekuensi

### Positif

- `F-3` dan `S-5` punya ukuran kelulusan yang nyata, bukan sekadar dikecualikan.
- Empat modul bisnis dapat mulai dikerjakan tanpa menunggu pencabutan pasal secara formal.
- Penghalang yang tersisa per modul menjadi eksplisit dan dapat ditelusuri satu per satu.

### Negatif / utang teknis

- **Kedua kontrak belum ada.** Sampai keduanya diterima, `F-3` tidak dapat lulus gerbang apa pun
  dan tiketnya tetap `needs-info` — begitu pula `S-5`. Keputusan ini **menciptakan ketergantungan
  baru** kepada dua pihak luar.
- Uji terhadap kontrak hanya sekuat kontraknya. Kontrak yang ditulis buru-buru akan meloloskan
  modul yang sebenarnya belum benar.
- Mencabut `§21.4` untuk empat modul berarti **BRD dan kenyataan kini berbeda** sampai revisinya
  disetujui. Dokumen yang tidak sinkron adalah sumber kebingungan bagi pembaca baru.
- `B-5` tetap tertahan, dan tiga penghalangnya seluruhnya bergantung pada data yang harus diminta
  ke DBA.

### Risiko yang diterima secara sadar

- Dua modul paling sensitif diuji dengan cara yang berbeda dari 31 modul lainnya — perbedaan yang
  harus diingat setiap kali hasil pengujian dibaca.
- Pencabutan `§21.4` dilakukan lewat ADR, sementara BRD-nya menunggu revisi. Bila revisi itu tidak
  pernah disetujui, terdapat dua dokumen yang saling bertentangan.

## Pertanyaan terbuka

- **Usulan revisi `BRD §21.4`** agar sesuai keputusan ini — menunggu persetujuan Work Owner.
  Diajukan di Fase 6, tidak dikerjakan diam-diam.
- Kapan kontrak API HCC/HCQ tersedia? Pemilik: Tim HCC/HCQ (ADR-0024).
- Kapan daftar peristiwa wajib audit tersedia? Pemilik: Compliance (ADR-0026).
- Baris master komite untuk PA dan Travel di atas Rp 200.000.000 — memang tidak ada, atau belum
  diisi? Pemilik: Work Owner (ADR-0014).
