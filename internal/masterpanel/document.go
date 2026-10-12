package masterpanel

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Satu panel memegang SATU dokumen, bukan sebuah daftar.
//
// Itu bukan penyederhanaan yang saya pilih, melainkan bentuk datanya:
// `RDB List/GetIDDokumenPanel-SQL.xml` berbunyi
//
//	SELECT DOKUMENID AS "CoverID" FROM POOLDATA.PANEL_HE WHERE ID_PANEL = ?
//
// — satu kolom pada baris panelnya sendiri. Mengunggah berkas kedua MENGGANTI yang pertama;
// tidak ada tempat untuk menyimpan keduanya. Perilaku itu ditiru apa adanya (`P-5`), dan
// layar menyatakannya terang-terangan supaya tidak mengejutkan.
//
// # Kemana isi berkasnya pergi
//
// BUKAN ke basis data kita. Rantai Pega yang sudah ditelusuri sampai ujung:
//
//	Flow Action UploadDocument      berkas ditaruh di halaman sementara (SaveFilePenunjang)
//	 -> CNMUpdatePanelHE_act        saat panel disimpan
//	 -> PNCSaveAttachmentToDB       menyiapkan baris metadata
//	 -> SET_ATTACHMENT_64BIT.prc    INSERT ke POOLDATA.DATA_ATTACHFILE, TANPA kolom isi
//	 -> InsertDokumenPNC            isi berkas dikirim ke layanan penyimpanan internal
//	 -> Connect REST UploadDokumenPNC   POST /api/v1/upload
//	 -> InsertDataPNCStorage        URL dan masa berlakunya ke GENERAL.T_STORAGE_IMAGE
//
// Jadi `DATA_ATTACHFILE` hanya menyimpan METADATA beserta `IMAGEID`, dan `IMAGEID` itulah
// satu-satunya tali ke berkas yang sebenarnya. Itu persis seam `DocumentStore` pada
// `04-FUTURE-ARCHITECTURE.md` §3.3, dan modul `dokumenpenunjang` sudah memegangnya —
// modul ini menyambung ke sana lewat DocumentUploader di bawah, tidak membangunnya ulang.
//
// # Satu nuansa yang mudah salah dibaca
//
// Tabel `DATA_ATTACHFILE` PUNYA kolom `ATTACHFILE` yang berisi berkas, dan kueri modul lain
// membacanya. Yang tidak mengisinya adalah `SET_ATTACHMENT_64BIT`: baris lama memuat isi
// berkas di dalam tabel, baris baru tidak. Kita mengikuti yang baru — kolom itu dibiarkan
// kosong, dan berkasnya diambil lewat `IMAGEID`.

// MaxDocumentNoteLength membatasi catatan unggahan.
//
// `ATTACHNOTE` lebarnya tidak diketahui — DDL belum ada (`R-08`) — sehingga batas ini
// dugaan yang dibuat aman, bukan salinan dari skema. Ia ditaruh di domain supaya satu angka
// berlaku untuk seluruh adapter, dan supaya menggantinya saat DDL tiba hanya satu baris.
const MaxDocumentNoteLength = 250

// MaxDocumentNameLength membatasi nama berkas yang diterima.
const MaxDocumentNameLength = 200

// ErrDocumentMissing: panel ini belum punya dokumen.
var ErrDocumentMissing = errors.New("masterpanel: panel belum punya dokumen")

// ErrDocumentFileEmpty: permintaan unggah datang tanpa isi berkas.
var ErrDocumentFileEmpty = errors.New("masterpanel: berkas kosong")

// UploadFailure menggolongkan kegagalan unggah menurut APA YANG BOLEH DILAKUKAN PENGGUNA.
//
// Penggolongannya mengikuti seam serupa di modul registrasi, dan alasannya sama: pesan
// "gagal" tanpa penggolongan membuat pengguna menebak apakah ia boleh mengulang. Pada
// unggahan, menebak salah berakibat nyata — mengulang unggahan yang separuh berhasil
// menghasilkan berkas ganda.
type UploadFailure int

const (
	// UploadInvalid: isi permintaannya salah — perbaiki lalu ulangi.
	UploadInvalid UploadFailure = iota + 1
	// UploadTooLarge: berkas melampaui batas ukuran.
	UploadTooLarge
	// UploadUnavailable: layanan hulu gagal dan belum ada yang tersimpan — aman diulang.
	UploadUnavailable
	// UploadHalfDone: berkas sudah terkirim tetapi catatannya gagal — JANGAN diulang.
	UploadHalfDone
	// UploadMisconfigured: salah konfigurasi di sisi aplikasi.
	UploadMisconfigured
)

// DocumentUploadError adalah galat unggah beserta golongan dan pesan siap tampil.
type DocumentUploadError struct {
	Kind    UploadFailure
	Message string
	Err     error
}

func (e *DocumentUploadError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Unwrap membuka galat aslinya supaya errors.Is tetap bekerja menembus terjemahan.
func (e *DocumentUploadError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// DocumentFile adalah satu berkas yang hendak diunggah.
type DocumentFile struct {
	// Portal menentukan basis data entitas tujuannya (`D-75`).
	Portal string

	// FileName adalah nama asli yang dipilih pengguna, LENGKAP dengan ekstensinya —
	// ekstensi itulah yang menentukan tipe medianya di hilir.
	FileName string

	// Content adalah isi berkas apa adanya. Penyandian base64 urusan adapter penyimpanan,
	// bukan urusan modul ini.
	Content []byte

	// By adalah login pengunggah.
	By string
}

// DocumentUploader adalah seam ke layanan penyimpanan internal (`D-16`).
//
// Ia mengembalikan IMAGEID saja, karena hanya itu yang disimpan modul ini. URL dan masa
// berlakunya dimiliki modul dokumen penunjang dan dibaca dari sana saat dibutuhkan —
// menyalinnya ke sini akan membuat dua tempat menyimpan masa berlaku yang sama, dan yang
// satu pasti basi lebih dulu.
type DocumentUploader interface {
	Upload(ctx context.Context, f DocumentFile) (imageID string, err error)
}

// PanelDocument adalah satu baris metadata dokumen panel.
//
// Padanan kolom `POOLDATA.DATA_ATTACHFILE` yang benar-benar diisi
// `Database/SET_ATTACHMENT_64BIT.prc`. Kolom `ATTACHFILE` sengaja TIDAK ada di sini:
// procedure itu tidak mengisinya, dan menyediakan tempatnya di struct akan mengundang
// seseorang mengisinya — yang berarti isi berkas tersimpan dua kali, di tabel dan di
// layanan penyimpanan.
type PanelDocument struct {
	// DataID adalah kunci barisnya, dan inilah yang ditulis ke `PANEL_HE.DOKUMENID`.
	//
	// Bentuknya `tahun || lpad(nomor,10,'0')` — `SET_ATTACHMENT_64BIT.prc`.
	DataID string

	// ImageID adalah kunci berkasnya di layanan penyimpanan. Tanpa ini, barisnya menunjuk
	// ke tempat yang tidak ada.
	ImageID string

	// Name, MimeType, dan Note mengisi ATTACHNAME, ATTACHMIMETYPE, ATTACHNOTE.
	Name     string
	MimeType string
	Note     string

	// UploadedBy mengisi INPUTOPERATOR; UploadedAt mengisi INPUTDATE.
	UploadedBy string
	UploadedAt *time.Time

	// PanelID adalah panel yang dokumen ini menempel padanya, mengisi `IDPEGA`.
	//
	// Pega memakai kolom itu untuk kunci kasus klaim; di sini yang masuk adalah ID panel.
	// Pilihan itu diambil sadar: tanpa jejak balik ke panelnya, satu baris yatim di
	// `DATA_ATTACHFILE` tidak dapat ditelusuri milik siapa.
	PanelID string
}

// DocumentUpload adalah permintaan unggah dokumen sebuah panel, sudah lengkap.
type DocumentUpload struct {
	PanelID string
	File    DocumentFile
	Note    string
}

// CleanDocument merapikan permintaan unggah sebelum diperiksa.
//
// Ia TIDAK menyentuh nama berkas selain memangkas spasi: pembersihan nama adalah aturan
// modul dokumen penunjang (`BersihkanNamaBerkas`, tiruan `pxReplaceAllViaRegex` Pega), dan
// menirunya di sini akan membuat dua tempat memutuskan hal yang sama dengan hasil yang
// dapat berbeda.
func (u DocumentUpload) CleanDocument() DocumentUpload {
	u.PanelID = strings.TrimSpace(u.PanelID)
	u.Note = strings.TrimSpace(u.Note)
	u.File.FileName = strings.TrimSpace(u.File.FileName)
	u.File.By = strings.TrimSpace(u.File.By)
	u.File.Portal = strings.TrimSpace(u.File.Portal)
	return u
}

// CheckDocument memeriksa permintaan unggah, dan mengumpulkan SELURUH pelanggarannya.
//
// Mengumpulkan semuanya, bukan berhenti pada yang pertama, mengikuti aturan penanganan
// galat `11-CROSSCUTTING.md` §1.2 — yang sendirinya meniru Pega, yang menampilkan seluruh
// pesan sekaligus.
func (u DocumentUpload) CheckDocument() error {
	var v []Violation
	add := func(field, message string) { v = append(v, Violation{Field: field, Message: message}) }

	if u.PanelID == "" {
		add("id", "Panel tidak dikenali.")
	}
	switch {
	case u.File.FileName == "":
		add("berkas", "Pilih berkas yang akan diunggah.")
	case len(u.File.FileName) > MaxDocumentNameLength:
		add("berkas", "Nama berkas terlalu panjang.")
	}
	if len(u.File.Content) == 0 {
		add("berkas", "Berkas kosong. Pilih berkas yang berisi lalu unggah ulang.")
	}
	if len(u.Note) > MaxDocumentNoteLength {
		add("catatan", "Catatan terlalu panjang.")
	}
	if len(v) == 0 {
		return nil
	}
	return &ValidationError{Violation: v}
}

// DocumentRepo adalah seam ke penyimpan metadata dokumen panel.
//
// Ia DIPISAHKAN dari Repo, meski keduanya berujung di basis data yang sama, karena
// keduanya menyentuh tabel yang berbeda dengan pemilik yang berbeda: `PANEL_HE` milik modul
// ini, `DATA_ATTACHFILE` dipakai bersama banyak modul. Menyatukannya akan membuat seolah
// modul ini memiliki keduanya.
type DocumentRepo interface {
	// SaveDocument mencatat metadata dokumen dan menautkannya ke panelnya.
	//
	// Keduanya satu operasi, bukan dua: baris `DATA_ATTACHFILE` tanpa `PANEL_HE.DOKUMENID`
	// yang menunjuk padanya adalah baris yatim yang tidak terlihat dari layar mana pun, dan
	// tidak ada yang akan membersihkannya. `D-68` memindahkan kepemilikan transaksi ke Go
	// justru supaya hal seperti ini dapat dibungkus sekali jalan.
	//
	// DataID diterbitkan di dalam operasi ini, dan dikembalikan.
	SaveDocument(ctx context.Context, doc PanelDocument) (dataID string, err error)

	// LinkDocument menautkan dokumen yang SUDAH tercatat ke sebuah panel.
	//
	// Dipisahkan dari SaveDocument karena unggah CSV master menautkan SATU dokumen — berkas
	// CSV-nya sendiri — ke BANYAK panel sekaligus. Mencatatnya ulang per panel akan
	// menerbitkan satu baris DATA_ATTACHFILE per panel untuk berkas yang sama persis.
	//
	// Pega menempuhnya begitu: `PNCSaveAttachmentToDB` dipanggil sekali di luar perulangan
	// baris, lalu `TempPanel.CoverID := tempDocumentPanel.CaseID` disetel pada setiap baris.
	LinkDocument(ctx context.Context, panelID, dataID string) error

	// DocumentOf mengembalikan dokumen sebuah panel; ErrDocumentMissing bila belum ada.
	DocumentOf(ctx context.Context, panelID string) (PanelDocument, error)
}
