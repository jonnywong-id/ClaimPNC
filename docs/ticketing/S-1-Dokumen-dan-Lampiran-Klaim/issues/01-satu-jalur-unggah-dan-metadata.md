---
title: "TKT-S1-001 — Satu jalur unggah dan metadata dokumen"
labels: [modul::S-1, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-S1-001 — Satu jalur unggah dan metadata dokumen

Status: needs-info
Kesiapan: **terhalang keputusan**
Modul: **S-1 Dokumen & Lampiran** · Gelombang: 3 · Bergantung pada: TKT-U2-003, TKT-F2-003
Requirement: FR-S1    Keputusan: D-16    ADR: 0010, 0012    Risiko: —
Rule Pega yang digantikan: **tiga mekanisme penyimpanan** yang hidup bersamaan · `Activity/InsertDokumenPNC` · `Activity/UploadDocumentToGoogleStorage-Act.xml` (**nol `<pyMethod>`**, mendelegasikan lewat `Call InsertDokumenPNC` pada `:1849`) · `SET_ATTACHMENT_64BIT` · `DATAPEGA.PC_LINK_ATTACHMENT`
Peran penguji gerbang 2: **PncAdmin** dan **PNCSurveyor**

## Hasil yang diharapkan (dan nilai bisnisnya)

Seluruh dokumen klaim masuk lewat **satu jalur**, dan basis data hanya menyimpan **metadata**.

Nilai bisnisnya: hari ini ada **tiga mekanisme** penyimpanan yang hidup bersamaan, sehingga
menjawab "di mana dokumen klaim ini" menuntut memeriksa ketiganya. Satu jalur berarti satu tempat
mencari dan satu tempat memperbaiki.

## Ruang lingkup

- Satu jalur unggah lewat **API storage internal Sinarmas** yang sudah berjalan (`ADR-0010`).
- Metadata di basis data: ID gambar, URL, masa berlaku, kategori, kaitan ke klaim atau objek.
- Kategori dan kelengkapan dokumen per lini bisnis, dari master.
- Penanganan **dokumen yatim** — berkas tersimpan tanpa metadata, atau sebaliknya.

## Non-goal

- **Tidak** membangun komponen unggah — itu `TKT-U2-003`.
- **Tidak** memigrasikan dokumen lama — keputusannya belum diambil.
- **Tidak** menyimpan berkas di basis data.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **`COMMIT` di 4 lapis → pola outbox, atau menerima dokumen yatim?** | **Work Owner + Lead Engineer** | `ADR-0010` mencatat **tidak ada atomisitas** antara berkas dan metadata. Pola penanganannya belum dipilih, dan itu menentukan bentuk tabel |
| **`SET_ATTACHMENT_64BIT` menghapus pada `tCOMMAND` apa pun selain `'INSERT'`** — sengaja? | **Work Owner** | Perilaku menghapus sebagai efek samping perintah yang bukan hapus adalah pola berbahaya; bila tidak sengaja, ia celah yang sudah berjalan |
| **Dokumen lama pada tiga mekanisme — dimigrasikan atau dibaca di tempatnya selamanya?** | **Work Owner** (`ADR-0010`) | Menyatukan jalur tulis **tidak menyatukan data yang sudah ada** |
| `SET_ATTACHFILETEMPSALVAGE` dan `POOLDATA.base64decode` | **DBA** | Aturan lampiran salvage dan penyandian berkas |

## Acceptance criteria

- [ ] Seluruh unggah melewati **satu jalur**; **nol jalur alternatif** — diuji pemindaian.
- [ ] Basis data menyimpan **hanya metadata**; **nol kolom berisi isi berkas** — diuji pemeriksaan
      skema.
- [ ] Berkas terkirim tetapi metadata gagal tersimpan **terdeteksi dan dilaporkan**, bukan diam —
      diuji dengan kegagalan sengaja.
- [ ] Metadata tersimpan tetapi berkas gagal terkirim **tidak meninggalkan metadata yang menunjuk
      berkas tidak ada** — diuji.
- [ ] Penghapusan dokumen adalah **soft delete** pada metadata (`ADR-0012`); perilaku terhadap
      berkas di storage mengikuti keputusan Work Owner — diuji sesuai jawabannya.
- [ ] Kategori dokumen dan kelengkapannya dibaca dari master, bukan konstanta.
- [ ] Gerbang 1: metadata yang tersimpan **sama dengan Pega** pada 20 dokumen contoh.
- [ ] Gerbang 2: UAT **PncAdmin** dan **PNCSurveyor**.

## Dependency / Blocked by

`TKT-U2-003` · `TKT-F2-003` · `TKT-F4-001` (master kategori dokumen). **Terhalang empat keputusan
dan DBA.**

## Constraint keamanan, data, operasional

- Dokumen klaim memuat **data nasabah**, termasuk **data medis** pada lini PA dan Travel —
  aksesnya tunduk `FR-R2` dan berlaku **juga di staging** (`ADR-0029`).
- **Penyimpanan dokumen menjadi ketergantungan runtime pada sistem lain** (`ADR-0010`): bila API
  storage mati, unggah berhenti meski seluruh aplikasi sehat.
- Soft delete **tidak berlaku pada berkas di storage** — menghapus metadata tidak menghapus
  berkasnya, dan itu keputusan tersendiri.

## Migrasi skema / rollout / rollback

Menambah tabel metadata dokumen. Backward-compatible; tidak menyentuh `PC_LINK_ATTACHMENT` yang
masih dibaca Pega.

**Rollback:** metadata tetap ada; berkas di storage **tidak ikut dibersihkan**.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/dokumen/... -run TestSatuJalurUnggah
go test ./internal/app/dokumen/... -run TestBerkasTerkirimMetadataGagal
go run ./cmd/tools/cek-skema-tanpa-blob    # HARUS 0 kolom berisi isi berkas
go run ./cmd/s8 banding --modul S-1 --kasus 20
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Tiga mekanisme penyimpanan disatukan | `D-16` · `ADR-0010` |
| `UploadDocumentToGoogleStorage` bukan mekanisme keempat | `Activity/UploadDocumentToGoogleStorage-Act.xml:1849` |
| Tidak ada atomisitas berkas dan metadata | `ADR-0010` Negatif/utang teknis |
| `SET_ATTACHMENT_64BIT` menghapus pada `tCOMMAND` selain `'INSERT'` | `docs/verifikasi-bukti-adr.md` §15 baris `S-1` |
| `PC_LINK_ATTACHMENT` dibaca 3 rule Pega | `D-21` · `ADR-0004` |

## Comments
