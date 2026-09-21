package pelaporanklaim

import (
	"errors"
	"strings"
)

// Empat kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
//   - ErrNotFound           → nomor laporan yang diminta tidak ada.
//   - ErrAlreadyTransferred → laporan sudah dikirim ke ASM pusat, sehingga tidak dapat
//     ditransfer ulang. Ini KONFLIK, bukan isian cacat: yang
//     dikirim pengguna sah, tetapi keadaan penyimpanan sudah
//     berubah — biasanya karena orang lain mendahuluinya.
//   - ErrAlreadyRegistered  → laporan sudah menjadi klaim. Perubahan isi dan penautan
//     ulang ditolak; yang berlaku sejak titik itu adalah data
//     klaimnya, bukan laporannya.
//   - ErrNumberTaken        → nomor laporan yang hendak disisipkan sudah dipakai.
//     Seharusnya mustahil karena nomornya dibuat urutan; ia ada
//     supaya kemustahilan itu TERLIHAT bila terjadi, bukan
//     menimpa baris yang sudah ada diam-diam.
var (
	ErrNotFound           = errors.New("pelaporanklaim: laporan klaim tidak ditemukan")
	ErrAlreadyTransferred = errors.New("pelaporanklaim: laporan sudah ditransfer ke ASM pusat")
	ErrAlreadyRegistered  = errors.New("pelaporanklaim: laporan sudah diregistrasi menjadi klaim")
	ErrNumberTaken        = errors.New("pelaporanklaim: nomor laporan sudah dipakai")
)

// Field yang dapat membawa pelanggaran validasi.
//
// NAMA konstantanya berbahasa Inggris (`D-80`); NILAINYA berbahasa Indonesia karena ia
// nama field JSON — kontrak yang dibaca klien, dan termasuk pengecualian `D-80`.
//
// Nilainya dipakai apa adanya oleh lapisan transport sebagai penunjuk isian di layar,
// sehingga antarmuka dapat menandai kolom yang salah — bukan sekadar menampilkan satu
// pesan di atas form. Pada form berisi 20 kolom, pembedaan itu menentukan.
//
// Nilainya SAMA PERSIS dengan nama field JSON pada dto. Bila keduanya berbeda, pesannya
// tetap sampai ke layar tetapi tidak menempel pada kolom mana pun.
const (
	FieldReporterName         = "nama_pelapor"
	FieldSenderEmail          = "email_pengirim"
	FieldSenderPhone          = "telepon_pengirim"
	FieldCourierName          = "nama_kurir"
	FieldEmailSubject         = "subjek_email"
	FieldPolicyNumber         = "nomor_polis"
	FieldInsuredName          = "nama_tertanggung"
	FieldInsuredEmail         = "email_tertanggung"
	FieldReferenceNumber      = "nomor_referensi"
	FieldClaimType            = "tipe_klaim"
	FieldDriverLicense        = "sim_pengendara"
	FieldLossLocation         = "lokasi_kejadian"
	FieldChronology           = "kronologi"
	FieldDamageDetails        = "rincian_kerusakan"
	FieldEstimatedValue       = "nilai_estimasi"
	FieldNotTransferredReason = "alasan_belum_transfer"
	FieldNotRegisteredNote    = "catatan_belum_registrasi"
	FieldDocumentCount        = "jumlah_dokumen"
	FieldClaimNumber          = "nomor_klaim"
)

// Violation adalah satu aturan yang dilanggar, beserta isian yang melanggarnya.
type Violation struct {
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus.
//
// Ia sengaja bukan daftar string: transport perlu tahu field mana yang salah untuk
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
	return "pelaporanklaim: validasi gagal — " + strings.Join(messages, "; ")
}

// NewValidationError membentuk galat validasi, atau nil bila tidak ada pelanggaran.
//
// Mengembalikan nil bertipe error yang benar-benar nil — bukan pointer nil yang
// terbungkus interface — supaya `if err != nil` di pemanggil berperilaku seperti yang
// terbaca.
func NewValidationError(violations []Violation) error {
	if len(violations) == 0 {
		return nil
	}
	return &ValidationError{Violations: violations}
}
