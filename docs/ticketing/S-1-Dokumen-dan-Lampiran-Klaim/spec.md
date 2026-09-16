# S-1 · Dokumen & Lampiran Klaim

| | |
|---|---|
| **Nama di sistem lama** | **Dokumen Klaim** — harness `PNCArchiveDokumen`, `ListDocumentObject`, `ListDocumentTravel`, `ListDocumentTypeInbox`, `DetTypeDocumenBisnis`, `BrowseMasterDocumentTravel_Harness`; activity `InsertDokumenPNC`, `UploadDocumentToGoogleStorage`, `SET_ATTACHMENT_64BIT` |
| **Kode modul** | `S-1` |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | 54 activity |
| **Bergantung pada** | `F-1`, `F-2`, `U-2` |
| **Kesiapan** | **SEBAGIAN** |

## Apa yang dikerjakan modul ini

Menyimpan dan mengambil **dokumen klaim** — berkas yang diunggah petugas, foto survei, lampiran
salvage, dan dokumen pendukung lain — lewat **satu jalur API storage internal** (`ADR-0010`).

Basis data hanya menyimpan **metadata dan referensi**: ID gambar, URL, masa berlaku, dan kategori.

## Tiga mekanisme lama yang disatukan

Sistem lama menyimpan dokumen lewat **tiga mekanisme berbeda** yang tumbuh berurutan dan masih
hidup bersamaan. Akibatnya tidak ada satu tempat pun yang menjawab "di mana dokumen klaim ini
berada" tanpa memeriksa ketiganya.

> **Catatan yang mencegah salah hitung.** `Activity/UploadDocumentToGoogleStorage-Act.xml`
> **bukan mekanisme keempat** — rule itu **nol `<pyMethod>`** dan mendelegasikan seluruh kerjanya
> lewat `Call InsertDokumenPNC` (`:1849`). Namanya menyesatkan, perilakunya tidak.

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | `SET_ATTACHFILETEMPSALVAGE` · `POOLDATA.base64decode` | **DBA** |
| **Keputusan** | **Token MD5 tanpa secret dan tanpa TTL — diterima Keamanan Informasi?** | **Keamanan Informasi** |
| **Keputusan** | TTL token dan URL berapa? · siapa membersihkan `GCP_IMAGE`? | **Work Owner** |
| **Keputusan** | **`COMMIT` di 4 lapis → pola outbox, atau menerima dokumen yatim?** | **Work Owner + Lead Engineer** |
| **Keputusan** | `SET_ATTACHMENT_64BIT` **menghapus pada `tCOMMAND` apa pun selain `'INSERT'`** — sengaja? | **Work Owner** |
| **Keputusan** | Dokumen lama pada tiga mekanisme — dimigrasikan, atau dibaca di tempatnya selamanya? | **Work Owner** (`ADR-0010`) |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-S1-001](issues/01-satu-jalur-unggah-dan-metadata.md) | Satu jalur unggah dan metadata dokumen | `needs-info` |
| [TKT-S1-002](issues/02-akses-dokumen-dan-token.md) | Akses dokumen, token, dan masa berlakunya | `needs-info` |
