package archivedokumenklaim

import (
	"errors"
	"strings"
)

// Kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
//   - ErrCallerUnknown  → identitas pemanggil tidak terbaca. Kolom USERINPUT dan saringan
//     jabatan keduanya bertumpu padanya.
//   - ErrNotFound       → baris arsip yang diminta tidak ada.
//   - ErrAlreadySent    → barisnya sudah pernah dikirim ke layanan Arsip.
//   - ErrServiceFailed  → layanan Arsip menolak atau tidak dapat dihubungi.
//   - ErrServiceAddress → alamat layanan Arsip belum terdaftar di katalog layanan.
var (
	ErrCallerUnknown  = errors.New("archivedokumenklaim: identitas pemanggil tidak terbaca")
	ErrNotFound       = errors.New("archivedokumenklaim: berkas arsip tidak ditemukan")
	ErrAlreadySent    = errors.New("archivedokumenklaim: berkas arsip sudah dikirim ke layanan Arsip")
	ErrServiceFailed  = errors.New("archivedokumenklaim: layanan Arsip tidak dapat dihubungi")
	ErrServiceAddress = errors.New("archivedokumenklaim: alamat layanan Arsip belum terdaftar")
)

// Field yang dapat membawa pelanggaran validasi.
//
// NAMA konstantanya berbahasa Inggris (`D-80`); NILAINYA berbahasa Indonesia karena ia
// nama field JSON — kontrak yang dibaca klien, dan termasuk pengecualian `D-80`.
//
// Nilainya SAMA PERSIS dengan nama field JSON pada dto. Bila keduanya berbeda, pesannya
// tetap sampai ke layar tetapi tidak menempel pada isian mana pun.
const (
	FieldSearchMode = "tipe_pencarian"
	FieldKeyword    = "kata_kunci"
	FieldFrom       = "tanggal_dari"
	FieldTo         = "tanggal_sampai"

	FieldClaimSearchType = "tipe_input"
	FieldClaimKeyword    = "keyword_klaim"

	FieldClaimNumber  = "nomor_klaim"
	FieldSheetCount   = "jumlah_lembar"
	FieldDocumentType = "tipe_dokumen"
	FieldDocumentKind = "jenis_dokumen"
	FieldBoxName      = "nama_box"
	FieldFillingCode  = "kode_filling"
	FieldReceivedDate = "tanggal_terima_dokumen"
)

// Violation adalah satu aturan yang dilanggar, beserta isian yang melanggarnya.
type Violation struct {
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus.
//
// Ia sengaja bukan daftar string: transport perlu tahu isian mana yang salah untuk
// menandainya di layar, dan informasi itu hilang bila pesannya dirangkai menjadi satu
// kalimat.
type ValidationError struct {
	Violations []Violation
}

func (e *ValidationError) Error() string {
	messages := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		messages = append(messages, v.Field+": "+v.Message)
	}
	return "archivedokumenklaim: validasi gagal — " + strings.Join(messages, "; ")
}

// NewValidationError membentuk galat validasi, atau nil bila tidak ada pelanggaran.
//
// Mengembalikan nil bertipe error yang benar-benar nil — bukan pointer nil yang terbungkus
// interface — supaya `if err != nil` di pemanggil berperilaku seperti yang terbaca.
func NewValidationError(violations []Violation) error {
	if len(violations) == 0 {
		return nil
	}
	return &ValidationError{Violations: violations}
}
