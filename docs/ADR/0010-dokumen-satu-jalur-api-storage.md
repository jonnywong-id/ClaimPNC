# 0010 — Simpan dokumen lewat satu jalur API storage internal; database hanya menyimpan metadata

Status: Accepted
Tanggal keputusan: 2026-09-07    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-16` | `Activity/UploadDocumentToGoogleStorage-Act.xml:1849` | `DATAPEGA.PC_LINK_ATTACHMENT` | `docs/verifikasi-bukti-adr.md` §3, §14.2
Terkait: ADR-0004, ADR-0026, modul `S-1`

## Konteks

Sistem lama menyimpan dokumen klaim lewat **tiga mekanisme berbeda** yang tumbuh berurutan, dan
ketiganya masih hidup bersamaan. Akibatnya tidak ada satu tempat pun yang menjawab "di mana
dokumen klaim ini berada" tanpa memeriksa ketiganya.

Satu koreksi penting atas analisis awal: `Activity/UploadDocumentToGoogleStorage-Act.xml` **bukan
mekanisme keempat**. Rule itu tidak memiliki satu pun `<pyMethod>` dan mendelegasikan seluruh
kerjanya lewat `Call InsertDokumenPNC` (`:1849`) — namanya menyesatkan, perilakunya tidak.
Hitungan tiga mekanisme pada `D-16` benar.

## Opsi yang dipertimbangkan

1. **Satu jalur lewat API storage internal Sinarmas** yang sudah berjalan; database hanya
   menyimpan metadata.
2. **Simpan berkas di database** sebagai BLOB — satu tempat, satu transaksi.
3. **Simpan di filesystem bersama** milik aplikasi.
4. Pertahankan ketiga mekanisme yang ada.

## Keputusan

Dokumen disimpan lewat **API storage internal Sinarmas yang sudah berjalan**. Database aplikasi
hanya menyimpan **metadata dan referensi**: ID gambar, URL, masa berlaku, dan kategori.

**Ketiga mekanisme yang ada disatukan menjadi satu jalur.** Tidak ada modul yang boleh menulis
dokumen dengan cara lain.

## Rationale

Menyimpan berkas di database (opsi 2) memindahkan beban penyimpanan ke tempat yang paling mahal
untuk di-backup dan direplikasi, pada sistem yang datanya sudah puluhan juta baris (`D-10`).

Filesystem bersama (opsi 3) menuntut storage bersama antar dua instans aplikasi (`D-27`) —
komponen infrastruktur tambahan yang bertentangan dengan `D-08`.

API storage internal sudah ada, sudah dipakai, dan sudah punya pemilik. Memakainya menghapus
seluruh kelas persoalan penyimpanan berkas dari lingkup proyek ini.

## Konsekuensi

### Positif

- Satu jalur, satu tempat mencari, satu tempat memperbaiki.
- Database tetap ramping dan cepat di-backup.
- Aplikasi tetap stateless — prasyarat `D-27`.

### Negatif / utang teknis

- **Penyimpanan dokumen menjadi ketergantungan runtime pada sistem lain.** Bila API storage mati,
  unggah dokumen berhenti meski seluruh aplikasi sehat.
- **Tidak ada atomisitas antara dokumen dan metadata.** Berkas tersimpan di sistem lain sementara
  barisnya di database, sehingga selalu mungkin terjadi berkas yatim (tersimpan tanpa metadata)
  atau metadata yatim (baris tanpa berkas). Mekanisme pembersihannya **harus dirancang** dan
  belum ada.
- **Dokumen lama tersebar di tiga mekanisme.** Menyatukan jalur tulis tidak menyatukan data yang
  sudah ada; pembacaan harus tetap menjangkau ketiganya sampai ada migrasi data dokumen —
  pekerjaan yang belum dijadwalkan.
- Soft delete (ADR-0012) tidak berlaku pada berkas di sistem lain. Menghapus metadata tidak
  menghapus berkasnya, dan itu keputusan tersendiri yang belum diambil.

### Risiko yang diterima secara sadar

- Kontrak API storage internal dimiliki tim lain dan dapat berubah di luar kendali proyek ini.
- Masa berlaku URL yang disimpan sebagai metadata berarti referensi dokumen dapat kedaluwarsa;
  perilaku saat itu terjadi belum ditentukan.

## Pertanyaan terbuka

- Bagaimana dokumen lama pada tiga mekanisme lama diperlakukan — dimigrasikan, atau dibaca di
  tempatnya selamanya? Pemilik: Work Owner. Menghalangi penyelesaian tiket `S-1`.
- Saat metadata dokumen dihapus secara soft delete, apakah berkas di storage ikut dihapus?
  Pemilik: Work Owner + Compliance.
- Berapa masa berlaku URL dokumen, dan apa yang terjadi setelah lewat? Pemilik: Tim pemilik API
  storage.
