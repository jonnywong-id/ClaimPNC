package masterrecovery

import (
	"errors"
	"strings"
)

// Kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
//   - ErrBatchTaken          → nomor batch yang hendak disisipkan sudah dipakai.
//     Seharusnya mustahil karena nomornya diterbitkan di
//     dalam transaksi yang sama; ia ada supaya kemustahilan
//     itu TERLIHAT bila terjadi. Ini memperbaiki cacat nyata
//     sistem lama, yang pada keadaan ini justru DIAM —
//     `INSERTMASTERRECOVERYKLAIM.prc` melewati insert-nya
//     tanpa satu pun pesan, sehingga petugas melihat
//     "berhasil" atas data yang tidak pernah tersimpan.
//   - ErrPrincipalNotFound   → principal dengan Client ID dan nama itu belum ada di
//     master Virtual Account.
//   - ErrPolicyNotFound      → nomor polis tidak dikenal, sehingga identitas lini
//     bisnis, cabang, agen, dan marketing tidak dapat diisi.
//   - ErrIssuerUnreachable   → layanan penerbit VA sedang tidak dapat dihubungi.
//     Mencoba lagi masuk akal.
//   - ErrIssuerUnconfigured  → alamat layanan penerbit VA belum terdaftar di
//     POOLDATA.GCNM_CONNECT_REST. DIBEDAKAN dari yang di
//     atas dengan sengaja: yang ini tidak akan pulih sendiri,
//     dan menyuruh pengguna mencoba lagi hanya membuang
//     waktunya.
//   - ErrIssuerRejected      → layanan menjawab, tetapi menolak menerbitkan.
var (
	ErrBatchTaken         = errors.New("masterrecovery: nomor batch recovery sudah dipakai")
	ErrPrincipalNotFound  = errors.New("masterrecovery: principal tidak ditemukan di master virtual account")
	ErrPolicyNotFound     = errors.New("masterrecovery: nomor polis tidak ditemukan")
	ErrIssuerUnreachable  = errors.New("masterrecovery: layanan penerbit virtual account tidak dapat dihubungi")
	ErrIssuerUnconfigured = errors.New("masterrecovery: alamat layanan penerbit virtual account belum terdaftar")
	ErrIssuerRejected     = errors.New("masterrecovery: layanan penerbit virtual account menolak permintaan")
	ErrDocumentNotSaved   = errors.New("masterrecovery: bukti bayar gagal disimpan")
)

// Field yang dapat membawa pelanggaran validasi.
//
// Nilainya dipakai apa adanya oleh lapisan transport sebagai penunjuk isian di layar,
// sehingga antarmuka dapat menandai kolom yang salah — bukan sekadar menampilkan satu
// pesan di atas form. Namanya mengikuti nama field JSON pada dto, karena itulah yang
// dikenali layar.
const (
	FieldPrincipalName   = "nama_principal"
	FieldClientID        = "client_id"
	FieldVirtualAccount  = "nomor_virtual_account"
	FieldYear            = "tahun"
	FieldClaimAmount     = "nilai_klaim"
	FieldPreviousPayment = "pembayaran_sebelumnya"
	FieldPayment         = "pembayaran"
	FieldRemark          = "keterangan"
	FieldCasePosition    = "posisi_kasus"
	FieldPolicyNo        = "nomor_polis"
	FieldClaimLine       = "baris_klaim"
	FieldDocument        = "bukti_bayar"
	FieldEmail           = "email_inputor_va"
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
// kalimat. Inilah yang menggantikan satu pesan tunggal "Wajib ISI semua field" di sistem
// lama, yang tidak memberi tahu isian mana yang kosong.
type ValidationError struct {
	Violation []Violation
}

func (e *ValidationError) Error() string {
	message := make([]string, 0, len(e.Violation))
	for _, v := range e.Violation {
		message = append(message, v.Field+": "+v.Message)
	}
	return "masterrecovery: validasi gagal — " + strings.Join(message, "; ")
}

// NewValidationError membentuk galat validasi, atau nil bila tidak ada pelanggaran.
//
// Mengembalikan nil bertipe error yang benar-benar nil — bukan pointer nil yang terbungkus
// interface — supaya `if err != nil` di pemanggil berperilaku seperti yang terbaca.
func NewValidationError(violation []Violation) error {
	if len(violation) == 0 {
		return nil
	}
	return &ValidationError{Violation: violation}
}
