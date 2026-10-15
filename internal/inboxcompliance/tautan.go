package inboxcompliance

import (
	"context"
	"net/url"
	"time"
)

// DefaultLinkDurationSeconds adalah umur tautan berkas, dalam detik.
//
// 3600 diambil dari `Activity/GetLinkViewDoc_Act-act.xml` langkah 16:
//
//	DocAPI.Durasi = @if(TempDurasi.CARI1="", "3600", TempDurasi.CARI1)
//
// Pega membacanya dari master lebih dulu dan jatuh ke 3600 bila kosong. Di sini 3600
// menjadi nilai bawaan konfigurasi — bukan konstanta yang tertanam di jalur panggilan
// (`D-15`).
const DefaultLinkDurationSeconds = 3600

// OfficeViewerBaseURL adalah penampil dokumen yang dipakai Pega.
//
// Dari langkah 22 activity yang sama:
//
//	Local.ViewLink = "https://view.officeapps.live.com/op/view.aspx?src=" + @encodeURL(Local.Link)
//
// Jadi yang dibuka pengguna BUKAN tautan berkasnya langsung, melainkan tautan itu
// dibungkus penampil Office Online.
//
// # Kenapa ia ditulis di sini, bukan disembunyikan sebagai rincian
//
// Karena akibatnya besar dan mudah terlewat: **berkas klaim dikirim ke layanan Microsoft
// untuk dirender**. Tautan bertanda tangan itu dapat dibuka siapa pun yang memegangnya
// selama masa berlakunya. Itu perilaku sistem lama yang `D-13` tetapkan ditiru, tetapi ia
// wajib terbaca di kode — bukan ditemukan belakangan oleh audit.
const OfficeViewerBaseURL = "https://view.officeapps.live.com/op/view.aspx?src="

// DocumentLink adalah tautan berkas yang baru diterbitkan.
type DocumentLink struct {
	// URL adalah tautan bertanda tangan dari layanan penyimpanan — `URLImage` pada
	// respons.
	URL string

	// ExpiresAt adalah `exp` pada respons. Pointer karena layanan dapat tidak
	// mengirimkannya, dan "tidak tahu kapan kedaluwarsa" berbeda dari "sudah
	// kedaluwarsa".
	ExpiresAt *time.Time
}

// ViewerURL membungkus tautan berkas dengan penampil Office Online, persis seperti
// langkah 22 Pega.
func (l DocumentLink) ViewerURL() string {
	if l.URL == "" {
		return ""
	}
	return OfficeViewerBaseURL + url.QueryEscape(l.URL)
}

// LinkRequest adalah permintaan penerbitan tautan baru.
//
// Isinya adalah halaman `DocAPI` yang Pega serialisasikan dengan `@GetPageJSONString()`
// (langkah 8 dan 17). Nama fieldnya mengikuti nama properti di sana apa adanya — ia
// kontrak dengan layanan penyimpanan, bukan nama internal kita (`D-80`).
type LinkRequest struct {
	// UserInput adalah `OperatorID.pyUserIdentifier` — login petugas yang membuka.
	UserInput string

	// StorageID adalah `DocAPI.ImageID`.
	StorageID string

	// App adalah `DocAPI.App`. Pega mengambilnya dari master lewat kunci `"KLAIMPNC"`.
	App string

	// Folder adalah `DocAPI.Folder`.
	Folder string

	// FileName adalah `DocAPI.NamaFile`.
	FileName string

	// DurationSeconds adalah `DocAPI.Durasi`.
	//
	// Dikirim sebagai ANGKA, bukan teks. Itu bukan pilihan kita: Pega menyusun JSON-nya
	// sebagai teks lalu menukar `"3600"` menjadi `3600` lewat
	// `@replaceAll(DocAPI_JSON.JSON, Local.S_Durasi, DocAPI.Durasi)` dengan
	// `S_Durasi = "\"" + Durasi + "\""`. Satu-satunya gunanya adalah membuang tanda
	// kutipnya — jadi layanan di seberang memang menuntut angka.
	DurationSeconds int
}

// DocumentLinker menerbitkan tautan baru untuk satu berkas.
//
// # Kenapa seam, dan kenapa tautannya selalu BARU
//
// Tautan berkas berumur terbatas (`Durasi`), sehingga ia tidak dapat disimpan di kolom
// lalu dipakai ulang. Setiap kali dokumen dibuka, tautannya diterbitkan lagi — itulah
// sebabnya `NewLinkDokumenPNC` ada sebagai layanan tersendiri, bukan sekadar kolom URL
// di tabel.
//
// Adapternya dua: klien HTTP ke `POST /api/v1/geturl`, dan fake untuk pengujian.
type DocumentLinker interface {
	NewLink(ctx context.Context, request LinkRequest) (DocumentLink, error)
}

// UploadRequest adalah permintaan unggah satu berkas.
//
// Isinya halaman `DocAPI` pada `Activity/InsertDokumenPNC-Act.xml` langkah 11 dan 16.
type UploadRequest struct {
	// UserInput adalah login petugas yang mengunggah.
	UserInput string

	// ClaimNumber adalah `DocAPI.NoClaim` = `pyWorkPage.pyID`.
	//
	// Pega menggantinya dengan `"-"` bila kosong (langkah 16), dan itu ditiru di
	// adapter — layanan di seberang tampaknya menolak nilai kosong.
	ClaimNumber string

	// FileName adalah `DocAPI.NamaFile`.
	//
	// Pega MEMBERSIHKANNYA lebih dulu: `@pxReplaceAllViaRegex(param.Filename,
	// "[^a-zA-Z0-9]", "")` pada langkah 2 — seluruh karakter selain huruf dan angka
	// dibuang, **termasuk titik dan ekstensinya**. Pembersihan itu dikerjakan adapter,
	// bukan dibebankan ke pemanggil.
	FileName string

	// MimeTypeHint adalah ekstensi berkas apa adanya (`pdf`, `JPG`, …).
	//
	// Adapter memetakannya ke tipe MIME memakai daftar yang sama dengan langkah 16.
	MimeTypeHint string

	// Content adalah isi berkas. Adapter yang meng-Base64-kannya.
	Content []byte
}

// UploadResult adalah hasil unggah.
type UploadResult struct {
	// StorageID adalah `ImageID` yang diterbitkan layanan — kunci berkas, yang kemudian
	// disimpan pada kolom `IMAGEID`.
	StorageID string
}

// DeleteRequest adalah permintaan hapus satu berkas.
//
// Dari `Activity/DeleteAttachDoc-act.xml` langkah 11:
//
//	DocAPI.NamaFile  = @replaceAll(TempDelete.pxResults(1).appfolder, "gs://klaimasmpnc/", "")
//	DocAPI.UserInput = ""
//
// Dua hal yang terbaca dari situ. Pertama, yang dikirim adalah **jalur objek**, bukan
// `ImageID` — dan awalan bucket-nya dipangkas lebih dulu. Kedua, `UserInput` SENGAJA
// dikosongkan; penghapusan tidak membawa pelaku ke layanan penyimpanan.
//
// Yang kedua itu catatan keamanan, bukan sekadar penyalinan: jejak siapa yang menghapus
// hanya ada di sisi kita (`D-59` menjadikan jejak audit satu-satunya kontrol pengimbang).
type DeleteRequest struct {
	// ObjectPath adalah jalur objek, boleh beserta awalan `gs://<bucket>/` — adapter
	// yang memangkasnya.
	ObjectPath string
}

// DocumentStore menyimpan dan menghapus berkas di layanan penyimpanan.
//
// Terpisah dari DocumentLinker, dan itu disengaja: menerbitkan tautan adalah pembacaan
// yang dilakukan setiap kali dokumen dibuka, sedangkan menyimpan dan menghapus mengubah
// isi penyimpanan. Memisahkannya membuat pemanggil yang hanya perlu membaca tidak
// memegang kemampuan menghapus.
type DocumentStore interface {
	Upload(ctx context.Context, request UploadRequest) (UploadResult, error)
	Delete(ctx context.Context, request DeleteRequest) error
}
