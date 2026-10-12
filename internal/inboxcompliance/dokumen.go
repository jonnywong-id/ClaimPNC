package inboxcompliance

import "time"

// Document adalah satu baris pada grid dokumen form Compliance Checker.
//
// # Dari mana bentuk ini berasal
//
// Grid S8/S9 `Section/CompliancePNC-Section.xml`, sel header 55–58 dan sel isi 60–63:
//
//	"Nama File"   .pyFileName       READONLY
//	"Type File"   .pyFileMimeType   READONLY
//	"Category"    .pyCategory       READONLY
//	""            .pyTemplateInputBox  fmt=pxIcon   <- tombol lihat berkas
//
// Judul kolom keempat memang KOSONG di Pega — ia kolom ikon, bukan kolom data.
//
// Gridnya bersyarat `pyFileName` tidak kosong pada baris pertama, yakni ia
// tersembunyi ketika klaimnya belum punya berkas. Itu sebabnya grid ini tidak terlihat
// pada tangkapan layar `PNC-2114`.
//
// # Sumbernya di basis data
//
// `POOLDATA.DATA_ATTACHFILE`, dikunci `IDPEGA` = `PZINSKEY` klaimnya. Kuncinya terbaca
// dari `RDB List/CountUpload-SQL.xml`:
//
//	select CATEGORY, SUB_CATEGORY from data_attachfile
//	 where IDPEGA = {pyWorkPage.pzInsKey} and IMAGEID IS NOT NULL
//
// Nama kolom lainnya dari `RDB List/GetAttachmentFromDB_Sql-SQL.xml` dan
// `RDB List/GetDocumentData-SQL.xml`.
//
// # Kenapa `IMAGEID IS NOT NULL` ikut disalin
//
// Karena `CountUpload` memakainya, dan maknanya nyata: `IMAGEID` adalah kunci berkas di
// penyimpanan luar, diisi setelah `UploadDokumenPNC` berhasil. Baris tanpa `IMAGEID`
// adalah unggahan yang **belum selesai** — berkasnya tidak dapat dibuka, sehingga
// menampilkannya hanya memberi pengguna baris yang tombolnya tidak pernah bekerja.
type Document struct {
	// ID adalah `DATAID`, kunci barisnya.
	ID string

	// Name adalah `ATTACHNAME` — kolom "Nama File".
	Name string

	// MimeType adalah `ATTACHMIMETYPE` — kolom "Type File".
	MimeType string

	// Category adalah `CATEGORY` — kolom "Category".
	Category string

	// SubCategory adalah `SUB_CATEGORY`.
	//
	// TIDAK digambar sebagai kolom — grid Pega hanya punya tiga kolom data. Ia dibawa
	// karena `CountUpload` membacanya berdampingan dengan `CATEGORY`, dan karena
	// kelengkapan dokumen per lini bisnis bersandar padanya.
	SubCategory string

	// StorageID adalah `IMAGEID` — kunci berkas di penyimpanan luar.
	//
	// Inilah yang dikirim ke `NewLinkDokumenPNC` untuk menerbitkan tautan baru setiap
	// kali dokumen dibuka. Ia TIDAK pernah digambar di layar; yang digambar tombolnya.
	StorageID string

	// UploadedAt adalah `INPUTDATE`. Pointer karena kolomnya dapat kosong.
	UploadedAt *time.Time

	// ClaimReference adalah `IDPEGA` — `PZINSKEY` klaim pemiliknya.
	//
	// TIDAK dibaca kueri daftar (di sana ia penyaringnya), tetapi WAJIB diisi saat
	// menulis: tanpa itu lampiran tidak melekat pada klaim mana pun.
	ClaimReference string

	// UploadedBy adalah `INPUTOPERATOR` — login petugas yang mengunggah.
	//
	// Satu-satunya jejak pelaku pada baris lampiran. Layanan penyimpanan tidak
	// menyimpannya, dan pada penghapusan Pega bahkan mengosongkannya.
	UploadedBy string

	// Note adalah `ATTACHNOTE`. Boleh kosong.
	Note string
}
