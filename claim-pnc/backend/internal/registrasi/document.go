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

// AttachmentStore menyisipkan baris lampiran klaim.
type AttachmentStore interface {
	AddAttachment(ctx context.Context, a NewAttachment) error
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
