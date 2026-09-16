# 0009 — Terbitkan nomor klaim baru berformat `PNCN.YY.xxxx` dari sequence database

Status: Accepted
Tanggal keputusan: 2026-09-07 (`D-22`), format direvisi 2026-09-14 (`D-71`)    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-22`, `D-71` (menyupersede bagian format pada `D-22`), `D-20` | `POOLDATA.CLAIM_NO_NONPEGA_SEQ` | `docs/verifikasi-bukti-adr.md` §2
Terkait: CONTEXT.md#Klaim, ADR-0004, ADR-0005, modul `B-2`

## Konteks

Identitas klaim di sistem lama adalah `pzInsKey` berformat
`ASM-FW-GCNMFW-WORK PNC-xxxx` — **kunci teknis Pega yang bocor menjadi identitas bisnis**. Format
itu memuat nama aplikasi, nama ruleset, dan nama kelas kerja Pega: seluruhnya hal yang akan
lenyap bersama Pega.

Nomor klaim bukan sekadar kunci internal. Ia muncul di surat ke tertanggung, di PLA/DLA ke
koasuransi dan reasuransi, serta di pelaporan. Ia **tidak dapat diubah surut**.

Selama masa paralel (ADR-0003), kedua sistem menerbitkan klaim baru ke database yang sama
(ADR-0004), sehingga nomor dari kedua sistem harus dapat hidup berdampingan dan dibedakan.

**Sequence-nya belum ada.** `POOLDATA.CLAIM_NO_NONPEGA_SEQ` **nol kemunculan di seluruh export** —
ia akan dibuat, bukan dipakai ulang.

## Opsi yang dipertimbangkan

1. **Format baru berprefix `PNCN`** dari sequence khusus, prefix Pega ditinggalkan.
2. **Pertahankan format lama** agar tidak ada yang berubah bagi pengguna dan pihak luar.
3. **Penomoran ulang seluruh klaim lama** ke format baru saat migrasi data.

Setelah opsi 1 dipilih, bentuk segmennya masih terbuka: dengan atau tanpa unsur tahun. `D-71`
menutupnya.

## Keputusan

Klaim yang dibuat sistem baru memakai format **`PNCN.YY.xxxx`** — tiga segmen dipisahkan
**titik**:

| Segmen | Isi | Sumber |
|---|---|---|
| `PNCN` | penanda tetap asal sistem baru | literal |
| `YY` | dua digit tahun | `TO_CHAR(SYSDATE,'RR')` |
| `xxxx` | nomor urut | `TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)` |

Sintaks Oracle yang ditetapkan, apa adanya:

```sql
'PNCN' || '.' || TO_CHAR(SYSDATE,'RR') || '.' || TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)
```

Prefix `ASM-FW-GCNMFW-WORK` **tidak dipakai lagi**.

Nomor klaim lama **dibiarkan apa adanya** — tidak dinomori ulang. Kedua format hidup berdampingan
secara permanen.

Generator nomor klaim adalah **satu-satunya tempat dengan sakelar dialek eksplisit** dalam
kebijakan SQL portabel (ADR-0005). Sejak `D-71`, sakelar itu membungkus **dua** perbedaan dialek
sekaligus — sequence dan pemformatan tahun — bukan satu.

> **`D-71` menyupersede `D-22` hanya pada bentuk nomornya** (`PNCN-xxxx` → `PNCN.YY.xxxx`). Sisa
> isi `D-22` — prefix Pega ditinggalkan, nomor lama tidak dinomori ulang — tetap berlaku.

## Rationale

Membawa `ASM-FW-GCNMFW-WORK` ke sistem yang tidak lagi memakai Pega berarti mengabadikan nama
platform yang sudah tiada di dalam data bisnis selamanya.

Penomoran ulang klaim lama (opsi 3) mustahil dipertanggungjawabkan: nomor itu sudah tercetak di
surat, sudah dikirim ke reasuransi, dan sudah dipakai pihak luar untuk merujuk klaim yang sama.

Prefix `PNCN` membuat asal sebuah klaim dapat dibaca langsung dari nomornya — berguna justru
selama masa paralel, ketika dua sistem menerbitkan klaim bersamaan.

Segmen tahun membuat **usia sebuah klaim terbaca tanpa membuka datanya** — berguna pada berkas
fisik, surat, dan percakapan dengan pihak luar, dan itu praktik lazim pada penomoran dokumen
asuransi.

## Konsekuensi

### Positif

- Identitas bisnis lepas dari kunci teknis platform.
- Asal klaim (Pega atau Go) **dan tahun terbitnya** terbaca dari nomornya tanpa melihat data lain.
- Sequence database menjamin keunikan tanpa koordinasi antar instans aplikasi (`D-27`).

### Negatif / utang teknis

- **Dua format nomor klaim hidup permanen.** Setiap pencarian, laporan, dan integrasi harus
  menerima keduanya. Ini bukan keadaan sementara — klaim lama tidak akan pernah berubah format.
- **Segmen terakhir tidak berlebar tetap.** `TO_CHAR(...NEXTVAL)` tanpa format mask tidak memberi
  angka nol di depan, sehingga nomor tumbuh `PNCN.26.9` → `PNCN.26.10` → `PNCN.26.1000`.
  Akibatnya **pengurutan sebagai teks tidak sesuai urutan penerbitan** — `.10` mendahului `.9`.
  Setiap layar dan laporan yang mengurutkan berdasarkan nomor klaim harus menyadarinya.
- **Tahun diambil dari `SYSDATE`**, yaitu tanggal server basis data — bukan tanggal kejadian dan
  bukan tanggal registrasi. Klaim yang diterbitkan di sekitar pergantian tahun mengambil tahun
  dari jam server, bertaut dengan `R-12` (pergeseran zona waktu) dan dengan penyesuaian 7 jam pada
  ADR-0017 butir 3.
- Pengurutan berdasarkan nomor klaim tidak bermakna kronologis lintas kedua format.
- Pihak luar — koasuransi, reasuransi, broker — akan menerima dua bentuk nomor dari perusahaan
  yang sama, dan sebagian sistem mereka mungkin memvalidasi formatnya.
- Titik sebagai pemisah lebih rawan daripada tanda hubung pada perkakas yang memperlakukan titik
  sebagai pemisah ekstensi berkas atau pemisah desimal — khususnya saat nomor klaim dipakai
  sebagai nama berkas atau ditempel ke lembar kerja.

### Risiko yang diterima secara sadar

- Segmen `xxxx` bertambah lebar seiring waktu; tidak ada batas panjang yang ditetapkan, sehingga
  kolom penyimpan dan bidang tampilan harus disiapkan longgar sejak awal.
- Sequence adalah satu-satunya pengecualian dialek yang disengaja; ia harus diuji di dua mesin
  database, bukan satu.
- Sintaks work owner dipakai **apa adanya**. Catatan netral: pada `TO_CHAR`, format `'RR'`
  menghasilkan dua digit tahun yang identik dengan `'YY'` — perbedaan perilaku `RR` hanya berlaku
  saat menafsirkan masukan (`TO_DATE`), bukan saat mengeluarkan teks.

## Pertanyaan terbuka

- **Apakah sequence direset setiap awal tahun?** Bila tidak, nomor urut terus bertambah melewati
  pergantian tahun (`PNCN.26.8125` → `PNCN.27.8126`) dan segmen tahun menjadi penanda, bukan
  penghitung per tahun. Bila ya, nomor mulai dari 1 tiap tahun. Keduanya sah; pilihannya mengubah
  cara nomor dibaca orang. Pemilik: Work Owner. Menghalangi penyelesaian tiket `B-2`.
- **Apakah lebar segmen terakhir perlu dibuat tetap** (mis. `TO_CHAR(seq.NEXTVAL,'FM0000')`) agar
  pengurutan teks benar? Pemilik: Work Owner. Sampai dijawab, sintaks yang ditetapkan dipakai apa
  adanya.
- Apakah pihak luar (reasuransi, broker, BPPDAN) perlu diberi tahu perubahan format sebelum klaim
  pertama terbit dari sistem baru? Pemilik: Work Owner.
