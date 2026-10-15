package registrasi

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Unggah Dokumen pada tab checklist klaim.
//
// # Sumber Pega
//
// Section unggahnya (`UploadDocument`) tidak ada di export; yang ada rantai penyimpanannya:
//
//	InsertDokumenPNC          unggah ke layanan penyimpanan internal → IMAGEID (T_STORAGE_IMAGE)
//	PNCSaveAttachmentToDB     SaveAttachmentToDB_Sql → SET_ATTACHMENT_64BIT
//	                          → satu baris POOLDATA.DATA_ATTACHFILE
//
// Baris DATA_ATTACHFILE itulah yang dihitung "Total Sudah Diunggah": `CountUpload` membaca
// baris ber-IMAGEID, `SetCountAttach_act` mencocokkan SUB_CATEGORY dengan DOC_TYPE_DT_ID
// (lihat DocumentChecklist). Isi kolomnya, terverifikasi pada baris Pega 2026-09-28:
//
//	CATEGORY       DOCUMENT_TYPE_ID jenis induk      (mis. 10066)
//	SUB_CATEGORY   DOC_TYPE_DT_ID jenis dokumen      (mis. 14893)
//	ATTACHMIMETYPE ekstensi berkas                   (mis. pdf)
//	IDPEGA         ASM-FW-GCNMFW-WORK <nomor klaim>
//
// Procedure-nya tidak dipanggil (`D-02`); penyisipannya ditulis langsung.

// ErrDocumentTypeUnknown: jenis dokumen yang dipilih bukan milik checklist lini bisnis
// klaim ini.
var ErrDocumentTypeUnknown = errors.New("registrasi: jenis dokumen tidak terdaftar untuk lini bisnis klaim ini")

// ErrDocumentFileEmpty: tidak ada isi berkas yang dikirim.
var ErrDocumentFileEmpty = errors.New("registrasi: berkas kosong")

// UploadFailure menggolongkan kegagalan unggah menurut apa yang boleh dilakukan pengguna.
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

// DocumentUploadError adalah kegagalan unggah beserta pesan untuk pengguna.
type DocumentUploadError struct {
	Kind    UploadFailure
	Message string
	Err     error
}

func (e *DocumentUploadError) Error() string {
	if e.Err != nil {
		return "registrasi: unggah dokumen gagal: " + e.Err.Error()
	}
	return "registrasi: unggah dokumen gagal: " + e.Message
}

func (e *DocumentUploadError) Unwrap() error { return e.Err }

// AttachmentNoteMaxLength dan AttachmentNameMaxLength adalah lebar ATTACHNOTE dan
// ATTACHNAME (VARCHAR2(255)).
const (
	AttachmentNoteMaxLength = 255
	AttachmentNameMaxLength = 255
)

// DocumentFile adalah satu berkas yang diunggah ke layanan penyimpanan.
type DocumentFile struct {
	Portal      string
	ClaimNumber string
	FileName    string
	Content     []byte
	By          string
}

// DocumentUploader adalah seam ke modul dokumen penunjang — layanan penyimpanan internal
// beserta metadatanya (`D-16`). Ia mengembalikan IMAGEID yang diterbitkan layanan.
type DocumentUploader interface {
	Upload(ctx context.Context, f DocumentFile) (imageID string, err error)
}

// DocumentLink adalah alamat baca satu berkas di layanan penyimpanan, beserta masa
// berlakunya — kolom URLPUBLIC dan EXPDATE pada GENERAL.T_STORAGE_IMAGE.
type DocumentLink struct {
	URL string

	// ExpiresAt nol berarti metadata tidak mencatat masa berlaku.
	ExpiresAt time.Time
}

// DocumentLinker membaca alamat berkas dari metadata penyimpanan menurut IMAGEID-nya.
//
// Padanan `Activity/GetLinkViewDoc_Act-act.xml`: alamat tersimpan dipakai selama berlaku;
// bila kosong atau kedaluwarsa, diperpanjang lewat Connect REST `NewLinkDokumenPNC` atas nama
// pengguna yang membuka berkas (by), lalu disimpan kembali.
type DocumentLinker interface {
	Link(ctx context.Context, portal, imageID, by string) (DocumentLink, error)
}

// DocumentRemover menghapus berkas dari layanan penyimpanan — Connect REST `DeleteDokumenPNC`
// pada `Activity/DeleteAttachDoc-act.xml`. Penghapusannya PERMANEN (keputusan-implementasi §171).
type DocumentRemover interface {
	Remove(ctx context.Context, portal, imageID, by string) error
}

// CanDeleteAttachment menyatakan tombol Delete tampil untuk lampiran ini: hanya pengunggahnya
// sendiri, dan hanya bila berkasnya tersimpan di layanan penyimpanan (ada IMAGEID).
//
// `Section/GCNMViewAttachment2-sect.xml` mensyaratkan `.exp <= 60.0 && .UserInput ==
// OperatorID.pyUserIdentifier`. Bagian `.UserInput` dibawa. Bagian `.exp <= 60.0` TIDAK: artinya
// diisi `InputParamUpload_act` kelas `ASM-FW-GCNMFW-Int-V_LST_DET_TYPE_DOC` yang belum ada di export,
// dan Work Owner menetapkan (2026-10-04) berkas yang sudah diunggah tetap dapat dihapus —
// menggantikan asumsi "60 menit sejak unggah" yang sempat dipakai (keputusan-implementasi §171).
// `now` dipertahankan supaya syarat waktu dapat dikembalikan tanpa mengubah pemanggil.
func CanDeleteAttachment(a Attachment, identity string, _ time.Time) bool {
	identity = strings.TrimSpace(identity)
	if identity == "" || strings.TrimSpace(a.ImageID) == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(a.UploadedBy), identity)
}

// ErrAttachmentDeleteNotAllowed: pemanggil bukan pengunggahnya.
var ErrAttachmentDeleteNotAllowed = errors.New("registrasi: lampiran ini tidak dapat dihapus oleh pengguna ini")

// ErrDocumentDeleteFailed: layanan penyimpanan tidak menghapus berkasnya. Tidak ada yang berubah.
var ErrDocumentDeleteFailed = errors.New("registrasi: berkas tidak dapat dihapus dari penyimpanan")

// ErrDocumentDeleteHalfDone: berkas SUDAH terhapus dari penyimpanan, tetapi catatan lampirannya
// gagal dihapus — baris lampiran menunjuk berkas yang sudah tidak ada.
var ErrDocumentDeleteHalfDone = errors.New("registrasi: berkas terhapus dari penyimpanan, catatan lampiran gagal dihapus")

// ErrDocumentLinkRenewFailed: layanan penyimpanan tidak dapat memperpanjang alamat berkas.
var ErrDocumentLinkRenewFailed = errors.New("registrasi: alamat berkas tidak dapat diperpanjang")

// ErrAttachmentNotFound: lampiran yang diminta bukan milik klaim ini.
var ErrAttachmentNotFound = errors.New("registrasi: lampiran tidak ditemukan pada klaim ini")

// ErrDocumentLinkEmpty: metadata penyimpanan belum mencatat alamat berkasnya.
var ErrDocumentLinkEmpty = errors.New("registrasi: alamat berkas belum tercatat")

// ErrDocumentLinkUnavailable: metadata penyimpanan tidak dapat dibaca.
var ErrDocumentLinkUnavailable = errors.New("registrasi: metadata penyimpanan dokumen tidak dapat dibaca")

// DocumentLinkExpiredError: alamat berkas sudah lewat masa berlakunya.
type DocumentLinkExpiredError struct {
	ExpiresAt time.Time
}

func (e *DocumentLinkExpiredError) Error() string {
	return "registrasi: alamat berkas kedaluwarsa sejak " + e.ExpiresAt.Format(time.RFC3339)
}

// NewAttachment adalah satu baris DATA_ATTACHFILE yang akan disisipkan.
type NewAttachment struct {
	ClaimKey    string // IDPEGA
	Name        string // ATTACHNAME
	Note        string // ATTACHNOTE
	Extension   string // ATTACHMIMETYPE
	ImageID     string
	Category    string // DOCUMENT_TYPE_ID
	SubCategory string // DOC_TYPE_DT_ID
	By          string
	At          time.Time
}

// AttachmentStore menyisipkan dan menghapus baris lampiran klaim.
type AttachmentStore interface {
	AddAttachment(ctx context.Context, a NewAttachment) error

	// DeleteAttachment menghapus baris DATA_ATTACHFILE dan JSON_FORM_KLAIM menurut IMAGEID —
	// `DeleteDataAttachFile_SQL` dan `DeleteDataJSON_FORM_KLAIM_SQL`. Penghapusan FISIK,
	// pengecualian `D-66` (keputusan-implementasi §171).
	DeleteAttachment(ctx context.Context, imageID string) error
}

// AttachmentExtension mengambil ekstensi berkas dalam huruf kecil, seperti isi
// ATTACHMIMETYPE pada baris Pega (`pdf`).
func AttachmentExtension(name string) string {
	dot := strings.LastIndex(name, ".")
	if dot < 0 || dot == len(name)-1 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(name[dot+1:]))
}

// Truncate memotong teks pada batas rune kolom tujuannya.
func Truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
